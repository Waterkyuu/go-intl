// Package cldrtext exposes read-only CLDR list and relative-time patterns.
package cldrtext

import (
	"slices"
	"strings"

	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
)

//go:generate python3 generate.py

// Table indexes one deduplicated CLDR locale row.
type Table uint16

// For selects the language's table, following CLDR parent tags to the neutral root.
func For(tag language.Tag) Table {
	for {
		name := tag.String()
		if i, ok := slices.BinarySearch(locales[:], name); ok {
			return Table(localeRows[i])
		}
		tag = tag.Parent()
	}
}

// ListParts returns the separator and suffix for pair, start, middle or end positions (0–3).
func (t Table) ListParts(position int) (string, string) {
	row := &rows[t]
	return words[row[2*position]], words[row[2*position+1]]
}

// Now returns the locale's name for the current instant, when available.
func (t Table) Now() string {
	return words[rows[t][8]]
}

// Relative returns the long numeric pattern for a unit in ascending duration order.
func (t Table) Relative(unit int, past bool, form plural.Form) string {
	direction := 0
	if past {
		direction = 1
	}
	return words[rows[t][9+(2*unit+direction)*6+int(form)]]
}

// CurrencyFormat contains a locale's standard monetary affixes and decimal notation.
type CurrencyFormat struct {
	// Digits indexes a decimal digit by its numeric value.
	Digits [10]rune
	// Patterns contains standard positive, negative and their alphabetic-symbol variants.
	Patterns [4]string
	// Group separates integer digit groups.
	Group string
	// Decimal separates major and fractional units.
	Decimal string
	// Infinity names an unbounded magnitude.
	Infinity string
	// NaN names an indeterminate magnitude.
	NaN string
	// Primary is the size of the rightmost integer group.
	Primary int
	// Secondary is the size of each preceding group.
	Secondary int
	// Minimum is the minimum leading digit count needed to show a group separator.
	Minimum int
}

// Currency selects CLDR currency rules and respects a published numbering-system variant.
func Currency(tag language.Tag) *CurrencyFormat {
	nu := tag.TypeForKey("nu")
	for {
		name, _, _ := strings.Cut(tag.String(), "-u-")
		if nu != "" {
			if i, ok := slices.BinarySearch(currencyLocales[:], name+"-u-nu-"+nu); ok {
				return &currencies[currencyRows[i]]
			}
		}
		if i, ok := slices.BinarySearch(currencyLocales[:], name); ok {
			return &currencies[currencyRows[i]]
		}
		tag = tag.Parent()
	}
}
