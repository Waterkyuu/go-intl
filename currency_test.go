package intl

import (
	"math"
	"testing"
)

// TestCurrencyCLDR checks currency affixes, bidirectional marks, grouping and local digits.
func TestCurrencyCLDR(t *testing.T) {
	cases := []struct {
		// tag selects a packaged CLDR currency convention.
		tag string
		// positive is the formatted positive USD amount.
		positive string
		// negative is the formatted negative USD amount.
		negative string
	}{
		{"en", "$1,234,567.20", "-$1,234,567.20"},
		{"de", "1.234.567,20\u00a0$", "-1.234.567,20\u00a0$"},
		{"fr", "1\u202f234\u202f567,20\u00a0$US", "-1\u202f234\u202f567,20\u00a0$US"},
		{"hi", "$12,34,567.20", "-$12,34,567.20"},
		{"hi-u-nu-deva", "$१२,३४,५६७.२०", "-$१२,३४,५६७.२०"},
		{"fa", "\u200e$۱٬۲۳۴٬۵۶۷٫۲۰", "\u200e−\u200e$۱٬۲۳۴٬۵۶۷٫۲۰"},
		{"ar", "\u200f1,234,567.20\u00a0US$", "\u200f\u200e-1,234,567.20\u00a0US$"},
		{"ar-EG", "\u200f١٬٢٣٤٬٥٦٧٫٢٠\u00a0US$", "\u061c-\u200f١٬٢٣٤٬٥٦٧٫٢٠\u00a0US$"},
	}
	for _, tc := range cases {
		loc := linguisticLocale(t, tc.tag)
		for amount, want := range map[float64]string{1234567.2: tc.positive, -1234567.2: tc.negative} {
			got, err := loc.Currency(amount, "USD")
			if err != nil || got != want {
				t.Errorf("%s Currency(%v) = %q, %v; want %q", tc.tag, amount, got, err, want)
			}
		}
	}
}

// TestCurrencyPrecision checks alphabetic symbols, non-finite values and decimal carries.
func TestCurrencyPrecision(t *testing.T) {
	loc := linguisticLocale(t, "en")
	cases := []struct {
		// amount supplies a magnitude in major currency units.
		amount float64
		// code supplies an ISO currency with a known standard scale.
		code string
		// want includes the expected symbol and any required alphabetic-symbol spacing.
		want string
	}{
		{-0.001, "USD", "-$0.00"},
		{math.Copysign(0, -1), "USD", "-$0.00"},
		{math.Inf(-1), "USD", "-$∞"},
		{math.NaN(), "USD", "$NaN"},
		{0.125, "USD", "$0.13"},
		{1.005, "USD", "$1.01"},
		{999.995, "USD", "$1,000.00"},
		{-9.5, "JPY", "-¥10"},
		{99.9995, "KWD", "KWD\u00a0100.000"},
	}
	for _, tc := range cases {
		got, err := loc.Currency(tc.amount, tc.code)
		if err != nil || got != tc.want {
			t.Errorf("Currency(%v, %q) = %q, %v; want %q", tc.amount, tc.code, got, err, tc.want)
		}
	}
}
