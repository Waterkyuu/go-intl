package intl

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/waterkyuu/go-intl/internal/cldrtext"
	"golang.org/x/text/feature/plural"
)

// List joins items using the selected locale's long CLDR conjunction patterns.
// Items are copied verbatim, including empty strings. The patterns use static
// conjunctions and do not adapt to the pronunciation of the following item.
func (l Locale) List(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	table := cldrtext.For(l.Tag())
	if len(items) == 2 {
		separator, suffix := table.ListParts(0)
		return items[0] + separator + items[1] + suffix
	}
	start, afterStart := table.ListParts(1)
	middle, afterMiddle := table.ListParts(2)
	end, afterEnd := table.ListParts(3)
	n := len(start) + len(afterStart) + (len(items)-3)*(len(middle)+len(afterMiddle)) + len(end) + len(afterEnd)
	for _, item := range items {
		n += len(item)
	}
	var text strings.Builder
	text.Grow(n)
	text.WriteString(items[0])
	text.WriteString(start)
	text.WriteString(items[1])
	text.WriteString(afterStart)
	for _, item := range items[2 : len(items)-1] {
		text.WriteString(middle)
		text.WriteString(item)
		text.WriteString(afterMiddle)
	}
	text.WriteString(end)
	text.WriteString(items[len(items)-1])
	text.WriteString(afterEnd)
	return text.String()
}

// relativeSeconds orders units from seconds through years as in the CLDR rows.
// A month is one twelfth of a 365-day year.
var relativeSeconds = [...]float64{1, 60, 3600, 86400, 7 * 86400, 365 * 86400 / 12, 365 * 86400}

// RelativeTime describes target relative to reference using the largest elapsed
// unit from seconds to years. It rounds to the nearest integer with ties toward
// the future. Days are 24 hours and months are one twelfth of a 365-day year.
func (l Locale) RelativeTime(target, reference time.Time) string {
	seconds := float64(target.Unix()) - float64(reference.Unix())
	seconds += float64(target.Nanosecond()-reference.Nanosecond()) / 1e9
	unit := len(relativeSeconds) - 1
	for unit > 0 && math.Abs(seconds) < relativeSeconds[unit] {
		unit--
	}
	value := math.Floor(seconds/relativeSeconds[unit] + 0.5)
	tag := l.Tag()
	table := cldrtext.For(tag)
	if unit == 0 && value == 0 && table.Now() != "" {
		return table.Now()
	}
	count := math.Abs(value)
	integer := pluralOperand(strconv.FormatFloat(count, 'f', 0, 64))
	form := plural.Cardinal.MatchPlural(tag, integer, 0, 0, 0, 0)
	pattern := table.Relative(unit, seconds < 0, form)
	// Some forms, such as Arabic duals, include the count in the word itself.
	if !strings.Contains(pattern, "{0}") {
		return pattern
	}
	return strings.Replace(pattern, "{0}", l.Number(count), 1)
}
