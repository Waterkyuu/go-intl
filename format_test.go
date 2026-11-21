package intl_test

import (
	"testing"
	"time"

	"github.com/waterkyuu/go-intl"
	"golang.org/x/text/language"
)

// TestLocaleFormatting checks separators, currency precision, date styles and timezone preservation.
func TestLocaleFormatting(t *testing.T) {
	c := testCatalog(t)
	fr := c.Locale("fr")
	en := c.Locale("en")
	if got := fr.Number(12.5); got != "12,5" {
		t.Errorf("French decimal = %q", got)
	}
	if got := en.Percent(0.25); got != "25%" {
		t.Errorf("English percent = %q", got)
	}
	if got, err := en.Currency(23.45, "USD"); got != "$23.45" || err != nil {
		t.Errorf("USD = %q, %v", got, err)
	}
	if _, err := fr.Currency(10, "xyz"); err == nil {
		t.Error("expected invalid ISO currency error")
	}
	when := time.Date(2026, time.January, 5, 17, 42, 0, 0, time.FixedZone("CET", 3600))
	if got := fr.Date(when, intl.Long); got != "5 janvier 2026" {
		t.Errorf("French date = %q", got)
	}
	if got := fr.Time(when, intl.Short); got != "17:42" {
		t.Errorf("French time = %q", got)
	}
	if got := en.LanguageName(language.French); got != "French" {
		t.Errorf("English language name = %q", got)
	}
}

// TestScriptDirection checks default, alternate and explicitly selected writing systems.
func TestScriptDirection(t *testing.T) {
	for tag, want := range map[string]bool{
		"ar": true, "he": true, "fa": true, "az-Arab": true, "az-Latn": false, "ff-Adlm": true, "zh": false,
	} {
		catalog, err := intl.New(tag, intl.Define(tag, nil))
		if err != nil {
			t.Fatal(err)
		}
		if got := catalog.Locale(tag).RightToLeft(); got != want {
			t.Errorf("RightToLeft(%s) = %v, want %v", tag, got, want)
		}
	}
}
