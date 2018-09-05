package contacts

import (
	"fmt"
	"strings"
	"time"
)

type Age struct {
	Years   int
	Months  int
	Days    int
	Hours   int
	Minutes int
	Seconds int
}

func NewAge(from, to time.Time) Age {
	y, M, d, h, m, s := diff(from, to)
	return Age{Years: y, Months: M, Days: d, Hours: h, Minutes: m, Seconds: s}
}

func (a Age) Year() string   { return stringify(a.Years, "Year", "Years") }
func (a Age) Month() string  { return stringify(a.Months, "Month", "Months") }
func (a Age) Day() string    { return stringify(a.Days, "Day", "Days") }
func (a Age) Hour() string   { return stringify(a.Hours, "Hour", "Hours") }
func (a Age) Minute() string { return stringify(a.Minutes, "Minute", "Minutes") }
func (a Age) Second() string { return stringify(a.Seconds, "Second", "Seconds") }

func (a Age) String() string { return join([]string{a.Year(), a.Month(), a.Day()}) }
func (a Age) Full() string {
	return join([]string{a.Year(), a.Month(), a.Day(), a.Hour(), a.Minute(), a.Second()})
}
func (a Age) Short() string {
	for _, fn := range []func() string{
		a.Year,
		a.Month,
		a.Day,
		a.Hour,
		a.Minute,
		a.Second,
	} {
		if short := fn(); short != "" {
			return short
		}
	}
	return ""
}

// diff computes elapsed time beyond hours.
// See https://stackoverflow.com/a/36531443/102371 for credit.
func diff(a, b time.Time) (year, month, day, hour, min, sec int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	year = int(y2 - y1)
	month = int(M2 - M1)
	day = int(d2 - d1)
	hour = int(h2 - h1)
	min = int(m2 - m1)
	sec = int(s2 - s1)

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}

	return
}

func join(items []string) string {
	var squeezed []string
	for _, item := range items {
		if item != "" {
			squeezed = append(squeezed, item)
		}
	}
	return strings.Join(squeezed, " ")
}

func stringify(amt int, singular, plural string) string {
	switch amt {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("1 %s", singular)
	default:
		return fmt.Sprintf("%d %s", amt, plural)
	}
}
