package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"jw4.us/contacts"
)

func main() {
	config := contacts.Config{
		Host:     os.Getenv("LDAP_HOST"),
		Port:     os.Getenv("LDAP_PORT"),
		Username: os.Getenv("LDAP_USER"),
		Password: os.Getenv("LDAP_PASS"),
		BaseDN:   os.Getenv("LDAP_BASE"),
	}
	records, err := contacts.List(config, nil)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(contacts.ByName(records))
	for _, p := range records {
		fmt.Printf("%30s %-40s %-20s %v\n", p.DisplayName(), p.Email, p.Phone, p.Labels)
	}
}
