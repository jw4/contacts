package contacts

import (
	"fmt"
	"time"
)

type Anniversary struct {
	Name  string
	Event time.Time
}

func (r Anniversary) Age(asOf time.Time) string {
	event := r.Event
	if event.IsZero() || asOf.IsZero() {
		return ""
	}
	if event.Year() == 0 {
		event = time.Date(asOf.Year(), event.Month(), event.Day(), event.Hour(), event.Minute(), event.Second(), event.Nanosecond(), event.Location())
		if event.Before(asOf) {
			event = time.Date(event.Year()+1, event.Month(), event.Day(), event.Hour(), event.Minute(), event.Second(), event.Nanosecond(), event.Location())
		}
	}
	age := NewAge(event, asOf)
	return age.String()
}

func (r Anniversary) String() string {
	if r.Event.IsZero() {
		return fmt.Sprintf("%-15s %-35s", "", r.Name)
	}
	now := time.Now()
	if r.hasYear() {
		return fmt.Sprintf("%-15s %-35s  Age: %s", r.Event.Format("Jan _2 2006"), r.Name, r.Age(now))
	}
	return fmt.Sprintf("%-15s %-35s Away: %s", r.Event.Format("Jan _2"), r.Name, r.Age(now))
}

func (r Anniversary) hasYear() bool { return r.Event.Year() > 0 }

type ByMonthDay []Anniversary

func (b ByMonthDay) Len() int           { return len(b) }
func (b ByMonthDay) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByMonthDay) Less(i, j int) bool { return compareMonthDay(b[i], b[j]) }

type ByEvent []Anniversary

func (b ByEvent) Len() int           { return len(b) }
func (b ByEvent) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByEvent) Less(i, j int) bool { return compareEvent(b[i], b[j]) }

type ByName []Anniversary

func (b ByName) Len() int           { return len(b) }
func (b ByName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool { return compareName(b[i], b[j]) }

func compareMonthDay(lhs, rhs Anniversary) bool {
	if lhs.Event.Month() == rhs.Event.Month() {
		if lhs.Event.Day() == rhs.Event.Day() {
			if lhs.Event.Year() == rhs.Event.Year() {
				return compareName(lhs, rhs)
			}
			return lhs.Event.Year() < rhs.Event.Year()
		}
		return lhs.Event.Day() < rhs.Event.Day()
	}
	return lhs.Event.Month() < rhs.Event.Month()
}

func compareEvent(lhs, rhs Anniversary) bool {
	if lhs.Event.Equal(rhs.Event) {
		return compareName(lhs, rhs)
	}
	return lhs.Event.Before(rhs.Event)
}

func compareName(lhs, rhs Anniversary) bool { return lhs.Name < rhs.Name }
