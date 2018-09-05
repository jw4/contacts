package contacts

import (
	"errors"
	"fmt"
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

func GetContacts(config LDAPConfig) ([]Contact, error) {
	var contacts []Contact
	err := getEntries(config, SearchRequest(config.BaseDN), func(e *ldap.Entry) {
		contacts = append(contacts, FromEntry(e))
	})
	if err != nil {
		return nil, err
	}
	return contacts, nil
}

func GetContact(config LDAPConfig, dn string) (Contact, error) {
	request := SearchRequest(config.BaseDN)
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
