package contacts

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	ldap "gopkg.in/ldap.v2"
)

type Contact struct {
	ID         string
	Name       string    `ldap:"displayName"`
	First      string    `ldap:"givenName"`
	Last       string    `ldap:"sn"`
	Suffix     string    `ldap:"generationQualifier"`
	Birthday   time.Time `ldap:"birthDate"`
	Email      []string  `ldap:"mail"`
	Phone      []string  `ldap:"telephoneNumber"`
	Labels     []string  `ldap:"label"`
	CommonName string    `ldap:"cn"`
	Street     []string  `ldap:"street"`
	City       string    `ldap:"l"`
	State      string    `ldap:"st"`
	Zip        string    `ldap:"postalCode"`
	Country    string    `ldap:"countryCode"`
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

func List(config Config, labels []string) ([]*Contact, error) {
	request := buildSearchRequest(config.BaseDN, labels)
	var contacts []*Contact
	err := getEntries(config, request, func(e *ldap.Entry) {
		contacts = append(contacts, fromEntry(e))
	})
	if err != nil {
		return nil, err
	}
	return contacts, nil
}

func Single(config Config, dn string) (*Contact, error) {
	request := buildSearchRequest(config.BaseDN, nil)
	request.BaseDN = dn
	request.Scope = ldap.ScopeBaseObject

	var contacts []*Contact
	err := getEntries(config, request, func(e *ldap.Entry) {
		contacts = append(contacts, fromEntry(e))
	})
	if err != nil {
		return nil, err
	}

	switch len(contacts) {
	case 0:
		return nil, errors.New("err not found")
	case 1:
		return contacts[0], nil
	default:
		sort.Sort(ByName(contacts))
		return contacts[0], nil
	}
}

func Delete(config Config, dn string) error {
	if err := del(config, buildDeleteRequest(dn)); err != nil {
		log.Printf("error deleting %q: %v", dn, err)
		return errors.New("error deleting")
	}
	return nil
}

func Save(config Config, original, updated *Contact) error {
	if updated == nil {
		return nil
	}
	if original == nil {
		original = &Contact{}
	}
	// Update
	if updated.ID != "" && original.ID == updated.ID {
		if err := save(config, buildModifyRequest(original, updated)); err != nil {
			log.Printf("error saving changes: %v", err)
			return errors.New("error saving changes")
		}
		return nil
	}
	// Create
	if err := create(config, buildAddRequest(config.BaseDN, updated)); err != nil {
		log.Printf("error creating contact: %v", err)
		return errors.New("error creating contact")
	}
	return nil
}

func buildSearchRequest(baseDN string, labels []string) *ldap.SearchRequest {
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

func buildModifyRequest(original, updated *Contact) *ldap.ModifyRequest {
	if updated == nil {
		return nil
	}
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
	return req
}

func buildAddRequest(baseDN string, contact *Contact) *ldap.AddRequest {
	if contact == nil {
		return nil
	}
	contact.ID = fmt.Sprintf("cn=%s,ou=contacts,%s", contact.DisplayName(), baseDN)
	req := ldap.NewAddRequest(contact.ID)
	req.Attribute("objectClass", []string{
		"contact",
		"inetOrgPerson",
		"organizationalPerson",
		"person",
		"top",
	})
	for k, v := range contact.attributeValues() {
		req.Attribute(k, v)
	}
	return req
}

func buildDeleteRequest(dn string) *ldap.DelRequest { return ldap.NewDelRequest(dn, nil) }

func fromEntry(entry *ldap.Entry) *Contact {
	if entry == nil {
		return nil
	}

	c := &Contact{ID: entry.DN}
	setAttributes(c, entry)
	return c
}
