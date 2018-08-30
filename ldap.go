package contacts

import (
	"fmt"
	"sort"
	"time"

	ldap "gopkg.in/ldap.v2"
)

type LDAPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	BaseDN   string
}

func GetBirthdays(config LDAPConfig) ([]Anniversary, error) {
	conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%s", config.Host, config.Port))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	err = conn.Bind(config.Username, config.Password)
	if err != nil {
		return nil, err
	}

	request := ldap.NewSearchRequest(
		fmt.Sprintf("ou=contacts,%s", config.BaseDN),
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(&(objectClass=contact))",
		[]string{"cn", "displayName", "birthDate"}, nil)

	s, err := conn.Search(request)
	if err != nil {
		return nil, err
	}

	var birthdays []Anniversary

	for _, e := range s.Entries {
		birthdate := time.Date(0, time.Month(0), 0, 0, 0, 0, 0, time.Local)
		name := e.GetAttributeValue("displayName")
		bday := e.GetAttributeValue("birthDate")
		if name == "" {
			name = e.GetAttributeValue("cn")
		}
		if bday != "" {
			if birthdate, err = time.ParseInLocation("Monday, January 02, 2006", bday, time.Local); err != nil {
				if birthdate, err = time.ParseInLocation("January 02", bday, time.Local); err != nil {
					fmt.Printf("error parsing %q: %v\n", bday, err)
					continue
				}
			}
		}
		birthdays = append(birthdays, Anniversary{Name: name, Event: birthdate})
	}

	sort.Sort(ByMonthDay(birthdays))

	return birthdays, nil
}
