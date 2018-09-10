package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"jw4.us/contacts"
)

const (
	Port          = int64(8818)
	ContactsRoute = "/contacts/"
)

func main() {
	config := contacts.Config{
		Host:     os.Getenv("LDAP_HOST"),
		Port:     os.Getenv("LDAP_PORT"),
		Username: os.Getenv("LDAP_USER"),
		Password: os.Getenv("LDAP_PASS"),
		BaseDN:   os.Getenv("LDAP_BASE"),
	}

	cs, err := contacts.NewWebServer(ContactsRoute, config, os.Getenv("TEMPLATE_FOLDER"))
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle(ContactsRoute, cs)
	mux.Handle("/", http.FileServer(http.Dir(os.Getenv("PUBLIC_FOLDER"))))

	port := Port
	if ports, ok := os.LookupEnv("PORT"); ok {
		port, err = strconv.ParseInt(ports, 10, 16)
		if err != nil {
			log.Fatal(err)
		}
	}

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", int16(port)),
		Handler: mux,
	}

	log.Fatal(s.ListenAndServe())
}
