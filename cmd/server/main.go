package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path"

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

	tmpl, err := template.New("").Funcs(tfns).ParseGlob(path.Join(tmplFolder, "*.html"))
	if err != nil {
		log.Fatal(err)
	}

	cs := &server{config: config, tmpl: tmpl}

	s := &http.Server{
		Addr:    ":8818",
		Handler: cs,
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
	case "/detail/":
		s.showDetail(w, r)
	case "/edit/":
	case "/":
		s.showList(w, r)
	default:
	}
}

func (s *server) showDetail(w http.ResponseWriter, r *http.Request) {
	dn := r.URL.Query().Get("dn")
	contact, err := contacts.GetContact(s.config, dn)
	if err != nil {
		log.Fatal(err)
	}

	err = s.tmpl.ExecuteTemplate(w, "detail.html", contact)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *server) showList(w http.ResponseWriter, r *http.Request) {
	records, err := contacts.GetContacts(s.config)
	if err != nil {
		log.Fatal(err)
	}
	ordered := map[string][]contacts.Contact{}
	for _, contact := range records {
		ordered[contact.BirthMonth()] = append(ordered[contact.BirthMonth()], contact)
	}
	err = s.tmpl.ExecuteTemplate(w, "list.html", ordered)
	if err != nil {
		log.Fatal(err)
	}
}

var tfns = map[string]interface{}{"months": months}

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
