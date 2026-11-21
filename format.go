package intl

import (
	"time"

	"github.com/bep/golocales"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"golang.org/x/text/number"
)

// DateStyle selects a locale's CLDR date or time pattern.
type DateStyle string

const (
	// Short selects a compact numeric pattern.
	Short DateStyle = "short"
	// Medium selects the default pattern.
	Medium DateStyle = "medium"
	// Long selects a more descriptive pattern.
	Long DateStyle = "long"
	// Full selects the most detailed pattern.
	Full DateStyle = "full"
)

// dateLocale resolves the formatting tables, including CLDR parent fallback.
func dateLocale(tag language.Tag) golocales.Translator {
	for tag != language.Und {
		if tr := golocales.New(tag.String()); tr != nil {
			return tr
		}
		tag = tag.Parent()
	}
	return golocales.New("en")
}

// Date formats a date in the supplied time's location. The default style is Medium.
func (l Locale) Date(value time.Time, style DateStyle) string {
	return formatDateTime(l.catalog.locales[l.chain[0]].dates, value, "date", string(style))
}

// Time formats a clock time in the supplied time's location.
func (l Locale) Time(value time.Time, style DateStyle) string {
	return formatDateTime(l.catalog.locales[l.chain[0]].dates, value, "time", string(style))
}

// formatDateTime shares CLDR formatting between message placeholders and public methods.
func formatDateTime(tr golocales.Translator, value time.Time, kind, style string) string {
	if kind == "time" {
		switch DateStyle(style) {
		case Short:
			return tr.FormatTimeShort(value)
		case Long:
			return tr.FormatTimeLong(value)
		case Full:
			return tr.FormatTimeFull(value)
		default:
			return tr.FormatTimeMedium(value)
		}
	}
	switch DateStyle(style) {
	case Short:
		return tr.FormatDateShort(value)
	case Long:
		return tr.FormatDateLong(value)
	case Full:
		return tr.FormatDateFull(value)
	default:
		return tr.FormatDateMedium(value)
	}
}

// Number formats a decimal using local digits, grouping and decimal separators.
func (l Locale) Number(value float64) string {
	return l.catalog.locales[l.chain[0]].printer.Sprint(number.Decimal(value))
}

// Percent displays a ratio as a percentage, so 0.25 denotes 25 percent.
func (l Locale) Percent(ratio float64) string {
	return l.catalog.locales[l.chain[0]].printer.Sprint(number.Percent(ratio))
}

// LanguageName gives the name of a language in the selected UI language.
func (l Locale) LanguageName(tag language.Tag) string {
	return display.Tags(l.Tag()).Name(tag)
}
