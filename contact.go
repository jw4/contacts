package contacts

import (
	"fmt"
	"log"
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

type Contact struct {
	ID         string
	Name       string    `ldap:"displayName"`
	First      string    `ldap:"givenName"`
	Last       string    `ldap:"sn"`
	Birthday   time.Time `ldap:"birthDate"`
	Email      []string  `ldap:"mail"`
	Phone      []string  `ldap:"telephoneNumber"`
	Labels     []string  `ldap:"label"`
	CommonName string    `ldap:"cn"`
}

func FindByLabel(baseDN string, labels []string) *ldap.SearchRequest {
	var b strings.Builder
	for _, label := range labels {
		fmt.Fprintf(&b, "(label=%s)", label)
	}
	c := &Contact{}
	return ldap.NewSearchRequest(
		fmt.Sprintf("ou=contacts,%s", baseDN),
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf("(&(objectClass=contact)%s)", b.String()),
		c.attributeNames(), nil)
}

func Update(original, updated *Contact) *ldap.ModifyRequest {
	changes := original.changes(updated)
	req := ldap.NewModifyRequest(updated.ID)
	for k, v := range changes["delete"] {
		req.Delete(k, v)
	}
	for k, v := range changes["add"] {
		req.Add(k, v)
	}
	for k, v := range changes["modify"] {
		req.Replace(k, v)
	}
	for k, v := range changes["replace"] {
		req.Replace(k, v)
	}
	log.Printf("UPDATE: %+v", req)
	return req
}

func Add(baseDN string, contact *Contact) *ldap.AddRequest {
	if contact == nil {
		return nil
	}
	contact.ID = fmt.Sprintf("cn=%s,ou=contacts,%s", contact.DisplayName(), baseDN)
	req := ldap.NewAddRequest(contact.ID)
	for k, v := range contact.attributeValues() {
		req.Attribute(k, v)
	}
	return req
}

func FromEntry(entry *ldap.Entry) *Contact {
	if entry == nil {
		return nil
	}

	c := &Contact{ID: entry.DN}
	setAttributes(c, entry)
	return c
}

func (c *Contact) Age() string { return c.AgeOn(time.Now()) }
func (c *Contact) AgeOn(date time.Time) string {
	event := c.birthdayOrZero()
	if event.IsZero() || date.IsZero() {
		return ""
	}
	if event.Year() == 0 {
		return ""
	}
	age := NewAge(event, date)
	return age.Short()
}

func (c *Contact) BirthDate() string      { return LDAPDate(c.birthdayOrZero()).Date() }
func (c *Contact) BirthDayOfWeek() string { return LDAPDate(c.birthdayOrZero()).DayOfWeek() }
func (c *Contact) BirthDayOfMonth() int   { return LDAPDate(c.birthdayOrZero()).DayOfMonth() }
func (c *Contact) BirthMonth() string     { return LDAPDate(c.birthdayOrZero()).Month() }
func (c *Contact) BirthYear() int         { return LDAPDate(c.birthdayOrZero()).Year() }
func (c *Contact) FullBirthDate() string  { return LDAPDate(c.birthdayOrZero()).FullDate() }
func (c *Contact) DisplayName() string {
	if c == nil {
		return ""
	}
	if c.Name != "" {
		return c.Name
	}
	if c.CommonName != "" {
		return c.CommonName
	}
	return strings.Join(
		[]string{
			strings.TrimSpace(c.First),
			strings.TrimSpace(c.Last),
		}, " ")
}

func (c *Contact) attributeNames() []string             { return attributeNames(c) }
func (c *Contact) attributeValues() map[string][]string { return attributeValues(c) }

func (c *Contact) birthdayOrZero() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.Birthday
}

func (c *Contact) changes(other *Contact) map[string]map[string][]string {
	return changes(attributeValues(c), attributeValues(other))
}

type ByName []*Contact

func (b ByName) Len() int           { return len(b) }
func (b ByName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool { return compareName(b[i], b[j]) }

type ByBirthday []*Contact

func (b ByBirthday) Len() int           { return len(b) }
func (b ByBirthday) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByBirthday) Less(i, j int) bool { return compareBirthday(b[i], b[j]) }

func compareBirthday(lhs, rhs *Contact) bool {
	lb, rb := lhs.birthdayOrZero(), rhs.birthdayOrZero()
	if lb.Month() == rb.Month() {
		if lb.Day() == rb.Day() {
			if lb.Year() == rb.Year() {
				return compareDisplay(lhs, rhs)
			}
			return lb.Year() < rb.Year()
		}
		return lb.Day() < rb.Day()
	}
	return lb.Month() < rb.Month()
}

func compareDisplay(lhs, rhs *Contact) bool {
	if lhs == rhs {
		return false
	}
	if lhs == nil {
		return true
	}
	if rhs == nil {
		return false
	}
	return lhs.Name < rhs.Name
}

func compareName(lhs, rhs *Contact) bool {
	if lhs == rhs {
		return false
	}
	if lhs == nil {
		return true
	}
	if rhs == nil {
		return false
	}
	if lhs.Last == rhs.Last {
		if lhs.First == rhs.First {
			return compareDisplay(lhs, rhs)
		}
		return lhs.First < rhs.First
	}
	return lhs.Last < rhs.Last
}
