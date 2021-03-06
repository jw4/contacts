package contacts

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"
)

var (
	templateFuncs = map[string]interface{}{
		"iterN":       iterN,
		"months":      months,
		"monthdays":   monthdays,
		"years":       years,
		"makeValues":  makeValues,
		"mailtoLink":  mailtoLink,
		"mailtoLinks": mailtoLinks,
	}
	monthNames = []string{
		"January",
		"February",
		"March",
		"April",
		"May",
		"June",
		"July",
		"August",
		"September",
		"October",
		"November",
		"December",
	}
)

func iterN(start, stop int) []int {
	r := []int{}
	if start > stop {
		for i := start; i >= stop; i-- {
			r = append(r, i)
		}
	} else {
		for i := start; i <= stop; i++ {
			r = append(r, i)
		}
	}
	return r
}

func months() []string                          { return monthNames }
func monthdays() []int                          { return iterN(1, 31) }
func years() []int                              { return iterN(time.Now().Year()+1, time.Now().Year()-100) }
func makeValues(key, val string) url.Values     { v := url.Values{}; v.Set(key, val); return v }
func mailtoLinks(list []*Contact) template.HTML { return mailtoLink(list...) }
func mailtoLink(list ...*Contact) template.HTML {
	filtered := contactsWithEmail(list...)
	switch len(filtered) {
	case 0:
		return template.HTML("")
	case 1:
		contact := filtered[0]
		return template.HTML(
			fmt.Sprintf(
				"<a href='mailto:%s'>%s</a>",
				safeEmailAddress(contact.DisplayName(), contact.Email[0]),
				contact.Email[0]))
	default:
		var addr []string
		for _, contact := range filtered {
			addr = append(addr, safeEmailAddress(contact.DisplayName(), contact.Email[0]))
		}
		return template.HTML(
			fmt.Sprintf(
				"<a href='mailto:%s'>Email All</a>",
				strings.Join(addr, ",")))
	}
}

func contactsWithEmail(list ...*Contact) []*Contact {
	var filtered []*Contact
	for _, contact := range list {
		if contact != nil && len(contact.Email) > 0 {
			filtered = append(filtered, contact)
		}
	}
	return filtered
}

func safeEmailAddress(name, email string) string {
	return url.PathEscape(fmt.Sprintf("%q <%s>", name, email))
}
