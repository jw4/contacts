package contacts

import (
	"fmt"
	"strings"
	"time"

	ldap "gopkg.in/ldap.v2"
)

const (
	CNameAttr    = "cn"
	NameAttr     = "displayName"
	FirstAttr    = "givenName"
	LastAttr     = "sn"
	BirthdayAttr = "birthDate"
	EmailAttr    = "mail"
	PhoneAttr    = "telephoneNumber"
	LabelsAttr   = "label"
)

var (
	attributes = []string{
		CNameAttr,
		NameAttr,
		BirthdayAttr,
		FirstAttr,
		LastAttr,
		EmailAttr,
		PhoneAttr,
		LabelsAttr,
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

type Contact struct {
	ID       string    `ldap:"dn"`
	Name     string    `ldap:"displayName"`
	First    string    `ldap:"givenName"`
	Last     string    `ldap:"sn"`
	Birthday time.Time `ldap:"birthDate"`
	Email    []string  `ldap:"mail"`
	Phone    []string  `ldap:"telephoneNumber"`
	Labels   []string  `ldap:"label"`
}

func FindByLabel(baseDN string, labels []string) *ldap.SearchRequest {
	var b strings.Builder
	for _, label := range labels {
		fmt.Fprintf(&b, "(label=%s)", label)
	}
	return ldap.NewSearchRequest(
		fmt.Sprintf("ou=contacts,%s", baseDN),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf("(&(objectClass=contact)%s)", b.String()),
		attributes, nil)
}

func Update(original, updated Contact) *ldap.ModifyRequest {
	req := ldap.NewModifyRequest(updated.ID)

	update := func(key, oldval, newval string) {
		if oldval != newval {
			if newval == "" {
				req.Delete(key, []string{oldval})
			} else if oldval == "" {
				req.Add(key, []string{newval})
			} else {
				req.Replace(key, []string{newval})
			}
		}
	}

	update(NameAttr, original.Name, updated.Name)
	update(BirthdayAttr, original.FullBirthDate(), updated.FullBirthDate())
	update(FirstAttr, original.First, updated.First)
	update(LastAttr, original.Last, updated.Last)

	req.Replace(EmailAttr, updated.Email)
	req.Replace(PhoneAttr, updated.Phone)
	req.Replace(LabelsAttr, updated.Labels)

	return req
}

func Add(baseDN string, contact Contact) *ldap.AddRequest {
	contact.ID = fmt.Sprintf("cn=%s,ou=contacts,%s", contact.DisplayName(), baseDN)
	req := ldap.NewAddRequest(contact.ID)
	for k, v := range contact.attributeValues() {
		req.Attribute(k, v)
	}
	return req
}

func FromEntry(entry *ldap.Entry) Contact {
	c := Contact{
		ID:       entry.DN,
		Name:     entry.GetAttributeValue(NameAttr),
		Birthday: parseDate(entry.GetAttributeValue(BirthdayAttr)),
		First:    entry.GetAttributeValue(FirstAttr),
		Last:     entry.GetAttributeValue(LastAttr),
		Email:    entry.GetAttributeValues(EmailAttr),
		Phone:    entry.GetAttributeValues(PhoneAttr),
		Labels:   entry.GetAttributeValues(LabelsAttr),
	}
	if c.Name == "" {
		c.Name = entry.GetAttributeValue(CNameAttr)
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

func (c Contact) BirthDate() string      { return LDAPDate(c.Birthday).Date() }
func (c Contact) BirthDayOfWeek() string { return LDAPDate(c.Birthday).DayOfWeek() }
func (c Contact) BirthDayOfMonth() int   { return LDAPDate(c.Birthday).DayOfMonth() }
func (c Contact) BirthMonth() string     { return LDAPDate(c.Birthday).Month() }
func (c Contact) BirthYear() int         { return LDAPDate(c.Birthday).Year() }
func (c Contact) FullBirthDate() string  { return LDAPDate(c.Birthday).FullDate() }
func (c Contact) DisplayName() string {
	if c.Name != "" {
		return c.Name
	}
	return strings.Join(
		[]string{
			strings.TrimSpace(c.First),
			strings.TrimSpace(c.Last),
		}, " ")
}

func (c *Contact) attributeValues() map[string][]string {
	return attributeValues(c)
}

func (c *Contact) changes(other *Contact) map[string]map[string][]string {
	return changes(attributeValues(c), attributeValues(other))
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

func parseDate(given string) time.Time {
	for _, dateFormat := range dateFormats {
		if date, err := time.ParseInLocation(dateFormat, given, time.Local); err == nil {
			return date
		}
	}
	return time.Time{}
}
