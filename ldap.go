package contacts

import (
	"errors"
	"fmt"
	"log"
	"sort"

	ldap "gopkg.in/ldap.v2"
)

type LDAPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	BaseDN   string
}

func GetContacts(config LDAPConfig, labels []string) ([]Contact, error) {
	request := SearchRequest(config.BaseDN, labels)
	var contacts []Contact
	err := getEntries(config, request, func(e *ldap.Entry) {
		contacts = append(contacts, FromEntry(e))
	})
	if err != nil {
		return nil, err
	}
	return contacts, nil
}

func GetContact(config LDAPConfig, dn string) (Contact, error) {
	request := SearchRequest(config.BaseDN, nil)
	request.BaseDN = dn
	request.Scope = ldap.ScopeBaseObject

	var contacts []Contact
	err := getEntries(config, request, func(e *ldap.Entry) {
		contacts = append(contacts, FromEntry(e))
	})
	if err != nil {
		return Contact{}, err
	}

	switch len(contacts) {
	case 1:
		return contacts[0], nil
	case 0:
		return Contact{}, errors.New("err not found")
	default:
		sort.Sort(ByName(contacts))
		return contacts[0], nil
	}
}

func SaveContact(config LDAPConfig, original, updated Contact) error {
	if updated.ID != "" && original.ID == updated.ID {
		if err := save(config, ModifyRequest(original, updated)); err != nil {
			log.Printf("error saving changes: %v", err)
			return errors.New("error saving changes")
		}
		return nil
	}
	log.Printf("creating new contact not supported yet")
	return nil
}

func save(config LDAPConfig, request *ldap.ModifyRequest) error {
	conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%s", config.Host, config.Port))
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.Bind(config.Username, config.Password)
	if err != nil {
		return err
	}

	return conn.Modify(request)
}

func getEntries(config LDAPConfig, request *ldap.SearchRequest, handle func(*ldap.Entry)) error {
	conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%s", config.Host, config.Port))
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.Bind(config.Username, config.Password)
	if err != nil {
		return err
	}

	s, err := conn.Search(request)
	if err != nil {
		return err
	}

	for _, e := range s.Entries {
		handle(e)
	}
	return nil
}
