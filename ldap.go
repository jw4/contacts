package contacts

import (
	"fmt"
	"log"
	"reflect"
	"time"

	ldap "github.com/go-ldap/ldap/v3"
)

func connect(config Config) (*ldap.Conn, error) {
	conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%s", config.Host, config.Port))
	if err != nil {
		return nil, err
	}

	err = conn.Bind(config.Username, config.Password)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func del(config Config, request *ldap.DelRequest) error {
	conn, err := connect(config)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Del(request)
}

func save(config Config, request *ldap.ModifyRequest) error {
	log.Printf("save: %+v", request)
	conn, err := connect(config)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Modify(request)
}

func create(config Config, request *ldap.AddRequest) error {
	log.Printf("create: %+v", request)
	conn, err := connect(config)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Add(request)
}

func getEntries(config Config, request *ldap.SearchRequest, handle func(*ldap.Entry)) error {
	conn, err := connect(config)
	if err != nil {
		return err
	}
	defer conn.Close()

	s, err := conn.Search(request)
	if err != nil {
		return err
	}

	for _, e := range s.Entries {
		handle(e)
	}
	return nil
}

func setAttributes(c interface{}, entry *ldap.Entry) {
	if c == nil {
		return
	}
	handleLDAPAttributes(c, func(n string, v reflect.Value) {
		attrs := entry.GetAttributeValues(n)
		if !v.CanSet() {
			return
		}
		if !v.CanInterface() {
			return
		}
		switch v.Interface().(type) {
		case string:
			sval := ""
			if len(attrs) > 0 {
				sval = attrs[0]
			}
			v.SetString(sval)
		case []string:
			v.Set(reflect.ValueOf(attrs))
		case time.Time:
			sval := ""
			if len(attrs) > 0 {
				sval = attrs[0]
			}
			v.Set(reflect.ValueOf(parseDate(sval)))
		}
	})
}

func attributeNames(c interface{}) []string {
	var names []string
	if c == nil {
		return names
	}

	handleLDAPAttributes(c, func(n string, v reflect.Value) {
		names = append(names, n)
	})

	return names
}

func attributeValues(c interface{}) map[string][]string {
	vals := map[string][]string{}
	if c == nil {
		return vals
	}

	handleLDAPAttributes(c, func(n string, fval reflect.Value) {
		if !fval.CanInterface() {
			return
		}
		switch v := fval.Interface().(type) {
		case string:
			if v != "" {
				vals[n] = []string{v}
			}
		case []string:
			if len(v) > 0 {
				vals[n] = v
			}
		case time.Time:
			if !v.IsZero() {
				vals[n] = []string{LDAPDate(v).FullDate()}
			}
		}
	})

	return vals
}

func handleLDAPAttributes(c interface{}, fn func(key string, val reflect.Value)) {
	if c == nil {
		return
	}

	cval := reflect.ValueOf(c).Elem()
	ctype := cval.Type()

	for i := 0; i < ctype.NumField(); i++ {
		field := ctype.Field(i)
		if n, ok := field.Tag.Lookup("ldap"); ok {
			fval := cval.Field(i)
			fn(n, fval)
		}
	}
}

func changes(cur, updates map[string][]string) map[string]map[string][]string {
	diff := map[string]map[string][]string{
		"add":     {},
		"delete":  {},
		"modify":  {},
		"replace": {},
	}
	if reflect.DeepEqual(cur, updates) {
		return nil
	}
	for k, v := range updates {
		if k == "cn" {
			continue
		}
		if pre, ok := cur[k]; ok {
			if reflect.DeepEqual(v, pre) {
				continue
			}
			if len(pre) == len(v) {
				if len(pre) > 1 {
					diff["replace"][k] = v
					continue
				}
				diff["modify"][k] = v
				continue
			} else {
				if len(v) == 0 {
					diff["delete"][k] = pre
					continue
				}
				if len(pre) == 0 {
					diff["add"][k] = v
					continue
				}
				diff["replace"][k] = v
				continue
			}
		} else {
			diff["add"][k] = v
		}
	}
	for k, pre := range cur {
		if k == "cn" {
			continue
		}
		if _, ok := updates[k]; !ok {
			diff["delete"][k] = pre
		}
	}
	return diff
}

type LDAPDate time.Time

func (l LDAPDate) FullDate() string {
	if time.Time(l).IsZero() {
		return ""
	}
	if time.Time(l).Year() > 0 {
		return time.Time(l).Format("Monday, January 2, 2006")
	}
	return time.Time(l).Format("January 2")
}

func (l LDAPDate) Date() string {
	if time.Time(l).IsZero() {
		return ""
	}
	if time.Time(l).Year() > 0 {
		return time.Time(l).Format("Jan _2, 2006")
	}
	return time.Time(l).Format("Jan _2")
}

func (l LDAPDate) DayOfWeek() string {
	if time.Time(l).IsZero() {
		return ""
	}
	return time.Time(l).Format("Monday")
}

func (l LDAPDate) DayOfMonth() int {
	if time.Time(l).IsZero() {
		return -1
	}
	return time.Time(l).Day()
}

func (l LDAPDate) Month() string {
	if time.Time(l).IsZero() {
		return ""
	}
	return time.Time(l).Format("January")
}

func (l LDAPDate) Year() int {
	if time.Time(l).IsZero() {
		return -1
	}
	return time.Time(l).Year()
}

var (
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
