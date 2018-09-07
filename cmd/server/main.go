package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"jw4.us/contacts"
)

func main() {
	config := contacts.LDAPConfig{
		Host:     os.Getenv("LDAP_HOST"),
		Port:     os.Getenv("LDAP_PORT"),
		Username: os.Getenv("LDAP_USER"),
		Password: os.Getenv("LDAP_PASS"),
		BaseDN:   os.Getenv("LDAP_BASE"),
	}

	tmplFolder := os.Getenv("TEMPLATE_FOLDER")
	publicFolder := os.Getenv("PUBLIC_FOLDER")

	tmpl, err := template.New("").Funcs(tfns).ParseGlob(path.Join(tmplFolder, "*.html"))
	if err != nil {
		log.Fatal(err)
	}

	cs := &server{config: config, tmpl: tmpl}

	mux := http.NewServeMux()
	mux.Handle("/contacts/", cs)
	mux.Handle("/", http.FileServer(http.Dir(publicFolder)))

	s := &http.Server{
		Addr:    ":8818",
		Handler: mux,
	}

	log.Fatal(s.ListenAndServe())
}

type server struct {
	config contacts.LDAPConfig
	tmpl   *template.Template
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("PATH %q", r.URL.Path)
	if err := r.ParseForm(); err != nil {
		log.Printf("error parsing form: %v", err)
		http.Error(w, "Bad Input", http.StatusBadRequest)
		return
	}
	switch r.URL.Path {
	case "/contacts/", "/contacts/list/":
		s.showList(w, r)
	case "/contacts/birthdays/":
		s.showBirthdays(w, r)
	case "/contacts/detail/":
		s.showDetail(w, r)
	case "/contacts/edit/":
		switch r.Method {
		case "POST":
			s.handleEdit(w, r)
		default:
			s.showEdit(w, r)
		}
	default:
	}
}

func (s *server) showEdit(w http.ResponseWriter, r *http.Request) {
	dn := r.Form.Get("dn")
	contact, err := contacts.GetContact(s.config, dn)
	if err != nil {
		log.Printf("finding %q: %v", dn, err)
		http.NotFound(w, r)
		return
	}

	err = s.tmpl.ExecuteTemplate(w, "edit.html", viewData{Title: makeTitle("Edit", contact.Name), Labels: nil, Contacts: []contacts.Contact{contact}, Request: r})
	if err != nil {
		log.Fatalf("executing template: %v", err)
	}
}

func (s *server) handleEdit(w http.ResponseWriter, r *http.Request) {
	switch r.Form.Get("submit") {
	case "Save":
		log.Printf("POST: %+v", r.Form)
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
		if err := contacts.SaveContact(s.config, contacts.Contact{
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
	http.Redirect(w, r, detailLink(r.Form), http.StatusSeeOther)
}

func (s *server) showDetail(w http.ResponseWriter, r *http.Request) {
	dn := r.Form.Get("dn")
	contact, err := contacts.GetContact(s.config, dn)
	if err != nil {
		log.Fatal(err)
	}

	err = s.tmpl.ExecuteTemplate(w, "detail.html", viewData{Title: makeTitle("Detail", contact.Name), Labels: nil, Contacts: []contacts.Contact{contact}, Request: r})
	if err != nil {
		log.Fatal(err)
	}
}

func (s *server) showList(w http.ResponseWriter, r *http.Request) {
	labels, _ := r.Form["label"]
	records, err := contacts.GetContacts(s.config, labels)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(contacts.ByName(records))
	err = s.tmpl.ExecuteTemplate(w, "list.html", viewData{Title: makeTitle("Contacts", labels...), Labels: labels, Contacts: records, Request: r})
	if err != nil {
		log.Fatal(err)
	}
}

func (s *server) showBirthdays(w http.ResponseWriter, r *http.Request) {
	labels, _ := r.Form["label"]
	records, err := contacts.GetContacts(s.config, labels)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(contacts.ByBirthday(records))
	ordered := map[string][]contacts.Contact{}
	for _, contact := range records {
		ordered[contact.BirthMonth()] = append(ordered[contact.BirthMonth()], contact)
	}
	err = s.tmpl.ExecuteTemplate(w, "birthdays.html", viewData{Title: makeTitle("Birthdays", labels...), Labels: labels, Contacts: records, ByMonth: ordered, Request: r})
	if err != nil {
		log.Fatal(err)
	}
}

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

type viewData struct {
	Title    string
	Labels   []string
	Contacts []contacts.Contact
	ByMonth  map[string][]contacts.Contact
	Request  *http.Request
}

var (
	tfns = map[string]interface{}{
		"months":        months,
		"makeValues":    makeValues,
		"editLink":      editLink,
		"detailLink":    detailLink,
		"contactsLink":  contactsLink,
		"birthdaysLink": birthdaysLink,
		"mailtoLink":    mailtoLink,
		"mailtoLinks":   mailtoLinks,
	}
	monthNames = []string{
		"January",
		"February",
		"March",
		"April",
		"May",
		"June",
		"July",
		"August",
		"September",
		"October",
		"November",
		"December",
	}
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
)

func months() []string { return monthNames }

func makeValues(key, val string) url.Values { v := url.Values{}; v.Set(key, val); return v }

func editLink(v url.Values) string     { return makelink("/contacts/edit/", filter(v, detailFilter)) }
func detailLink(v url.Values) string   { return makelink("/contacts/detail/", filter(v, detailFilter)) }
func contactsLink(v url.Values) string { return makelink("/contacts/list/", filter(v, listFilter)) }
func birthdaysLink(v url.Values) string {
	return makelink("/contacts/birthdays/", filter(v, listFilter))
}
func mailtoLinks(list []contacts.Contact) template.HTML { return mailtoLink(list...) }
func mailtoLink(list ...contacts.Contact) template.HTML {
	filtered := contactsWithEmail(list...)
	switch len(filtered) {
	case 0:
		return template.HTML("")
	case 1:
		contact := filtered[0]
		return template.HTML(fmt.Sprintf("<a href='mailto:%s'>%s</a>", safeEmailAddress(contact.Name, contact.Email[0]), contact.Email[0]))
	default:
		var addr []string
		for _, contact := range filtered {
			addr = append(addr, safeEmailAddress(contact.Name, contact.Email[0]))
		}
		return template.HTML(fmt.Sprintf("<a href='mailto:%s'>Email All</a>", strings.Join(addr, ",")))
	}
}
func contactsWithEmail(list ...contacts.Contact) []contacts.Contact {
	var filtered []contacts.Contact
	for _, contact := range list {
		if len(contact.Email) > 0 {
			filtered = append(filtered, contact)
		}
	}
	return filtered
}
func safeEmailAddress(name, email string) string {
	return url.PathEscape(fmt.Sprintf("%q <%s>", name, email))
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
func makelink(base string, v url.Values) string {
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
	for k, val := range q {
		for _, vv := range val {
			v.Add(k, vv)
		}
	}
	u.RawQuery = v.Encode()
	return u.String()
}
