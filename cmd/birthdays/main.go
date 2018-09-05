package main

import (
	"fmt"
	"log"
	"os"

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
	records, err := contacts.GetContacts(config)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range records {
		if p.BirthDate() != "" {
			fmt.Printf("%-13s %-30s %s\n", p.BirthDate(), p.Name, p.Age())
		}
	}
}
