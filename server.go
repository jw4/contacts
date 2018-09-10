package contacts

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

func NewWebServer(route string, config Config, templatesFolder string) (http.Handler, error) {
	server := &server{
		baseRoute: route,
		config:    config,
		tmpl:      template.New("").Funcs(templateFuncs),
	}

	if err := server.init(templatesFolder); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(server.editRoute(), server.handleEdit)
	mux.HandleFunc(server.detailRoute(), server.showDetail)
	mux.HandleFunc(server.birthdaysRoute(), server.showBirthdays)
	mux.HandleFunc(server.listRoute(), server.showList)
	mux.Handle("/", http.NotFoundHandler())
	return mux, nil
}

const (
	editRoute      = "edit/"
	listRoute      = "list/"
	createRoute    = "create/"
	detailRoute    = "detail/"
	birthdaysRoute = "birthdays/"

	editTemplate      = "edit.html"
	listTemplate      = "list.html"
	createTemplate    = "create.html"
	detailTemplate    = "detail.html"
	birthdaysTemplate = "birthdays.html"
)

type server struct {
	baseRoute string
	config    Config
	tmpl      *template.Template
}

type viewData struct {
	Title    string
	Labels   []string
	Contacts []Contact
	ByMonth  map[string][]Contact
	Request  *http.Request
}

func (s *server) init(templatesFolder string) error {
	linkFns := map[string]interface{}{
		"contactsLink":  s.listLink,
		"editLink":      s.editLink,
		"createLink":    s.createLink,
		"detailLink":    s.detailLink,
		"birthdaysLink": s.birthdaysLink,
	}
	if _, err := s.tmpl.Funcs(linkFns).ParseGlob(path.Join(templatesFolder, "*.html")); err != nil {
		return err
	}
	return nil
}

func (s *server) handleEdit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("error parsing form: %v", err)
		http.Error(w, "Bad Input", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "POST":
		s.handleEditPost(w, r)
	default:
		s.showEdit(w, r)
	}
}

func (s *server) handleEditPost(w http.ResponseWriter, r *http.Request) {
	switch r.Form.Get("submit") {
	case "Save":
		birthday := time.Time{}
		if month, ok := monthValues[r.Form.Get("birthMonth")]; ok {
			if day, err := strconv.Atoi(r.Form.Get("birthDay")); err == nil {
				year, err := strconv.Atoi(r.Form.Get("birthYear"))
				if err != nil {
					year = 0
				}
				birthday = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
			}
		}
		old, err := GetContact(s.config, r.Form.Get("dn"))
		if err != nil {
			old = Contact{}
		}
		if err = SaveContact(s.config, old, Contact{
			ID:       r.Form.Get("dn"),
			Name:     r.Form.Get("displayName"),
			First:    r.Form.Get("given"),
			Last:     r.Form.Get("sn"),
			Birthday: birthday,
			Email:    dedupe(r.Form["mail"]),
			Phone:    dedupe(r.Form["telephoneNumber"]),
			Labels:   dedupe(r.Form["label"]),
		}); err != nil {
			log.Printf("error saving: %v", err)
			http.Error(w, "unexpected error", http.StatusInternalServerError)
			return
		}
	default:
	}
	http.Redirect(w, r, s.detailLink(r.Form), http.StatusSeeOther)
}

func (s *server) showEdit(w http.ResponseWriter, r *http.Request) {
	dn := r.Form.Get("dn")
	contact, err := GetContact(s.config, dn)
	if err != nil {
		log.Printf("finding %q: %v", dn, err)
		http.NotFound(w, r)
		return
	}

	if err = s.tmpl.ExecuteTemplate(
		w, editTemplate, viewData{
			Title:    makeTitle("Edit", contact.Name),
			Contacts: []Contact{contact},
			Request:  r,
		}); err != nil {
		log.Fatalf("executing template: %v", err)
	}
}

func (s *server) showDetail(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("error parsing form: %v", err)
		http.Error(w, "Bad Input", http.StatusBadRequest)
		return
	}

	dn := r.Form.Get("dn")
	contact, err := GetContact(s.config, dn)
	if err != nil {
		log.Fatal(err)
	}

	if err = s.tmpl.ExecuteTemplate(
		w, detailTemplate,
		viewData{
			Title:    makeTitle("Detail", contact.Name),
			Contacts: []Contact{contact},
			Request:  r,
		}); err != nil {
		log.Fatal(err)
	}
}

func (s *server) showList(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("error parsing form: %v", err)
		http.Error(w, "Bad Input", http.StatusBadRequest)
		return
	}

	labels, _ := r.Form["label"]
	records, err := GetContacts(s.config, labels)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(ByName(records))
	if err = s.tmpl.ExecuteTemplate(
		w, listTemplate,
		viewData{
			Title:    makeTitle("Contacts", labels...),
			Labels:   labels,
			Contacts: records,
			Request:  r,
		}); err != nil {
		log.Fatal(err)
	}
}

func (s *server) showBirthdays(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("error parsing form: %v", err)
		http.Error(w, "Bad Input", http.StatusBadRequest)
		return
	}

	labels, _ := r.Form["label"]
	records, err := GetContacts(s.config, labels)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(ByBirthday(records))
	ordered := map[string][]Contact{}
	for _, contact := range records {
		ordered[contact.BirthMonth()] = append(ordered[contact.BirthMonth()], contact)
	}
	if err = s.tmpl.ExecuteTemplate(
		w, birthdaysTemplate,
		viewData{
			Title:    makeTitle("Birthdays", labels...),
			Labels:   labels,
			Contacts: records,
			ByMonth:  ordered,
			Request:  r,
		}); err != nil {
		log.Fatal(err)
	}
}

func (s *server) rootRoute() string      { return s.baseRoute }
func (s *server) listRoute() string      { return path.Join(s.baseRoute, listRoute) }
func (s *server) editRoute() string      { return path.Join(s.baseRoute, editRoute) }
func (s *server) createRoute() string    { return path.Join(s.baseRoute, createRoute) }
func (s *server) detailRoute() string    { return path.Join(s.baseRoute, detailRoute) }
func (s *server) birthdaysRoute() string { return path.Join(s.baseRoute, birthdaysRoute) }

func (s *server) listLink(v url.Values) string      { return makelink(s.listRoute, listFilter, v) }
func (s *server) editLink(v url.Values) string      { return makelink(s.editRoute, detailFilter, v) }
func (s *server) createLink(v url.Values) string    { return makelink(s.createRoute, noneFilter, v) }
func (s *server) detailLink(v url.Values) string    { return makelink(s.detailRoute, detailFilter, v) }
func (s *server) birthdaysLink(v url.Values) string { return makelink(s.birthdaysRoute, listFilter, v) }

//
// Helpers
//

var (
	monthValues = map[string]time.Month{
		"January":   time.January,
		"February":  time.February,
		"March":     time.March,
		"April":     time.April,
		"May":       time.May,
		"June":      time.June,
		"July":      time.July,
		"August":    time.August,
		"September": time.September,
		"October":   time.October,
		"November":  time.November,
		"December":  time.December,
	}
	detailFilter = []string{"dn"}
	listFilter   = []string{"label"}
	noneFilter   = []string(nil)
)

func makeTitle(main string, parts ...string) string {
	return strings.Join(append([]string{main}, parts...), " :: ")
}

func dedupe(list []string) []string {
	set := map[string]int{}
	i := 0
	for _, item := range list {
		if item != "" {
			if _, ok := set[item]; !ok {
				set[item] = i
				i++
			}
		}
	}
	filtered := make([]string, i)
	for k, v := range set {
		filtered[v] = k
	}
	return filtered
}

func makelink(fn func() string, keys []string, v url.Values) string {
	base := fn()
	u, err := url.Parse(base)
	if err != nil {
		log.Printf("error parsing url %q : %v", base, err)
		return base
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		log.Printf("error parsing query %q : %v", u.RawQuery, err)
		q = url.Values{}
	}

	v = filter(v, keys)
	for k, val := range q {
		for _, vv := range val {
			v.Add(k, vv)
		}
	}
	u.RawQuery = v.Encode()
	return u.String()
}

func filter(v url.Values, keys []string) url.Values {
	if len(keys) == 0 {
		return v
	}
	newV := url.Values{}
	for _, key := range keys {
		for _, val := range v[key] {
			newV.Add(key, val)
		}
	}
	return newV
}
