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

var (
	parseStrings = []string{
		"Monday, January 02, 2006",
		"Monday, January _2, 2006",
		"Monday, January 2, 2006",
		"January 02, 2006",
		"January _2, 2006",
		"January 2, 2006",
		"January 02",
		"January _2",
		"January 2",
		"Jan 02, 2006",
		"Jan _2, 2006",
		"Jan 2, 2006",
		"Jan 02",
		"Jan _2",
		"Jan 2",
		"01/02/2006",
		"1/2/2006",
		"01/02/06",
		"1/2/06",
		"01/02",
		"1/2",
	}
)

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
		birthdate := time.Time{}
		name := e.GetAttributeValue("displayName")
		bday := e.GetAttributeValue("birthDate")
		if name == "" {
			name = e.GetAttributeValue("cn")
		}
		if bday != "" {
			for _, parseString := range parseStrings {
				if birthdate, err = time.ParseInLocation(parseString, bday, time.Local); err == nil {
					break
				}
			}
			if err != nil {
				fmt.Printf("error parsing %q: %v\n", bday, err)
				birthdate = time.Time{}
			}
		}
		birthdays = append(birthdays, Anniversary{Name: name, Event: birthdate})
	}

	sort.Sort(ByMonthDay(birthdays))

	return birthdays, nil
}
