package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"

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
	switch r.URL.Path {
	case "/contacts/", "/contacts/list/":
		s.showList(w, r)
	case "/contacts/birthdays/":
		s.showBirthdays(w, r)
	case "/contacts/detail/":
		s.showDetail(w, r)
	case "/contacts/edit/":
	default:
	}
}

func (s *server) showDetail(w http.ResponseWriter, r *http.Request) {
	dn := r.URL.Query().Get("dn")
	contact, err := contacts.GetContact(s.config, dn)
	if err != nil {
		log.Fatal(err)
	}

	err = s.tmpl.ExecuteTemplate(w, "detail.html", viewData{Labels: nil, Contacts: []contacts.Contact{contact}})
	if err != nil {
		log.Fatal(err)
	}
}

func (s *server) showList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	labels, _ := q["label"]
	records, err := contacts.GetContacts(s.config, labels)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(contacts.ByName(records))
	err = s.tmpl.ExecuteTemplate(w, "list.html", viewData{Labels: labels, Contacts: records})
	if err != nil {
		log.Fatal(err)
	}
}

func (s *server) showBirthdays(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	labels, _ := q["label"]
	records, err := contacts.GetContacts(s.config, labels)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(contacts.ByBirthday(records))
	ordered := map[string][]contacts.Contact{}
	for _, contact := range records {
		ordered[contact.BirthMonth()] = append(ordered[contact.BirthMonth()], contact)
	}
	err = s.tmpl.ExecuteTemplate(w, "birthdays.html", viewData{Labels: labels, Contacts: records, ByMonth: ordered})
	if err != nil {
		log.Fatal(err)
	}
}

type viewData struct {
	Labels   []string
	Contacts []contacts.Contact
	ByMonth  map[string][]contacts.Contact
}

var tfns = map[string]interface{}{
	"months":        months,
	"contactsLink":  contactsLink,
	"birthdaysLink": birthdaysLink,
}

func months() []string {
	return []string{
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
}

func contactsLink(labels []string) string  { return makeLink("/contacts/list/", labels) }
func birthdaysLink(labels []string) string { return makeLink("/contacts/birthdays/", labels) }
func makeLink(base string, labels []string) string {
	var b strings.Builder
	b.WriteString(base)
	if len(labels) > 0 {
		b.WriteString("?")
		for _, label := range labels {
			fmt.Fprintf(&b, "label=%s&", label)
		}
		fmt.Fprintf(&b, "labels=%d", len(labels))
	}
	return b.String()
}
