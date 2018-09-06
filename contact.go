package contacts

import (
	"fmt"
	"strings"
	"time"

	ldap "gopkg.in/ldap.v2"
)

type Contact struct {
	ID       string
	Name     string
	First    string
	Last     string
	Birthday time.Time
	Email    []string
	Phone    []string
	Labels   []string
}

func SearchRequest(base string, labels []string) *ldap.SearchRequest {
	var b strings.Builder
	for _, label := range labels {
		fmt.Fprintf(&b, "(label=%s)", label)
	}
	return ldap.NewSearchRequest(
		fmt.Sprintf("ou=contacts,%s", base),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf("(&(objectClass=contact)%s)", b.String()),
		attributes, nil)
}

func FromEntry(entry *ldap.Entry) Contact {
	c := Contact{
		ID:       entry.DN,
		Name:     entry.GetAttributeValue("displayName"),
		Birthday: parseDate(entry.GetAttributeValue("birthDate")),
		First:    entry.GetAttributeValue("givenName"),
		Last:     entry.GetAttributeValue("sn"),
		Email:    entry.GetAttributeValues("mail"),
		Phone:    entry.GetAttributeValues("telephoneNumber"),
		Labels:   entry.GetAttributeValues("label"),
	}
	if c.Name == "" {
		c.Name = entry.GetAttributeValue("cn")
	}
	return c
}

func (c Contact) Age() string { return c.AgeOn(time.Now()) }
func (c Contact) AgeOn(date time.Time) string {
	event := c.Birthday
	if event.IsZero() || date.IsZero() {
		return ""
	}
	if event.Year() == 0 {
		return ""
	}
	age := NewAge(event, date)
	return age.Short()
}

func (c Contact) BirthDate() string {
	if c.Birthday.IsZero() {
		return ""
	}
	if c.Birthday.Year() > 0 {
		return c.Birthday.Format("Jan _2, 2006")
	}
	return c.Birthday.Format("Jan _2")
}

func (c Contact) BirthDayOfWeek() string {
	if c.Birthday.IsZero() {
		return ""
	}
	return c.Birthday.Format("Monday")
}

func (c Contact) BirthDayOfMonth() int {
	if c.Birthday.IsZero() {
		return -1
	}
	return c.Birthday.Day()
}

func (c Contact) BirthMonth() string {
	if c.Birthday.IsZero() {
		return ""
	}
	return c.Birthday.Format("January")
}

func (c Contact) BirthYear() int {
	if c.Birthday.IsZero() {
		return -1
	}
	return c.Birthday.Year()
}

type ByName []Contact

func (b ByName) Len() int           { return len(b) }
func (b ByName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool { return compareName(b[i], b[j]) }

type ByBirthday []Contact

func (b ByBirthday) Len() int           { return len(b) }
func (b ByBirthday) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByBirthday) Less(i, j int) bool { return compareBirthday(b[i], b[j]) }

func compareBirthday(lhs, rhs Contact) bool {
	if lhs.Birthday.Month() == rhs.Birthday.Month() {
		if lhs.Birthday.Day() == rhs.Birthday.Day() {
			if lhs.Birthday.Year() == rhs.Birthday.Year() {
				return compareDisplay(lhs, rhs)
			}
			return lhs.Birthday.Year() < rhs.Birthday.Year()
		}
		return lhs.Birthday.Day() < rhs.Birthday.Day()
	}
	return lhs.Birthday.Month() < rhs.Birthday.Month()
}

func compareDisplay(lhs, rhs Contact) bool { return lhs.Name < rhs.Name }

func compareName(lhs, rhs Contact) bool {
	if lhs.Last == rhs.Last {
		if lhs.First == rhs.First {
			return compareDisplay(lhs, rhs)
		}
		return lhs.First < rhs.First
	}
	return lhs.Last < rhs.Last
}

var (
	attributes = []string{
		"cn",
		"displayName",
		"birthDate",
		"givenName",
		"sn",
		"mail",
		"telephoneNumber",
		"label",
	}
	dateFormats = []string{
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

func parseDate(given string) time.Time {
	for _, dateFormat := range dateFormats {
		if date, err := time.ParseInLocation(dateFormat, given, time.Local); err == nil {
			return date
		}
	}
	return time.Time{}
}
