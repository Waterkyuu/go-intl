package intl

import (
	"strings"
	"testing"
	"time"
)

// linguisticLocale constructs an empty localized catalog for formatting-only tests.
func linguisticLocale(t *testing.T, tag string) Locale {
	t.Helper()
	catalog, err := New(tag, Define(tag, nil))
	if err != nil {
		t.Fatal(err)
	}
	return catalog.Locale(tag)
}

// TestList checks the CLDR pair, start, middle and end patterns and verbatim item text.
func TestList(t *testing.T) {
	for tag, want := range map[string]string{
		"en":    "A, B, C, and D",
		"en-GB": "A, B, C and D",
		"zh":    "A、B、C和D",
		"ja":    "A、B、C、D",
		"ar":    "A وB وC وD",
		"und":   "A, B, C, D",
	} {
		if got := linguisticLocale(t, tag).List([]string{"A", "B", "C", "D"}); got != want {
			t.Errorf("List(%s) = %q, want %q", tag, got, want)
		}
	}
	en := linguisticLocale(t, "en")
	if got := en.List(nil); got != "" {
		t.Errorf("empty list = %q", got)
	}
	if got := en.List([]string{"{1}"}); got != "{1}" {
		t.Errorf("one item = %q", got)
	}
	if got := en.List([]string{"", "{0}"}); got != " and {0}" {
		t.Errorf("pair = %q", got)
	}
}

// TestRelativeTime checks rounding, unit boundaries, plural morphology and numberless forms.
func TestRelativeTime(t *testing.T) {
	ref := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		// tag is the display locale.
		tag string
		// delta is the elapsed interval relative to ref.
		delta time.Duration
		// want is the expected CLDR result.
		want string
	}{
		{"en", 0, "now"},
		{"en", -500 * time.Millisecond, "now"},
		{"en", 500 * time.Millisecond, "in 1 second"},
		{"en", -126 * time.Minute, "2 hours ago"},
		{"en", -90 * time.Minute, "1 hour ago"},
		{"en", 24 * time.Hour, "in 1 day"},
		{"en", -8 * 24 * time.Hour, "1 week ago"},
		{"en", 35 * 24 * time.Hour, "in 1 month"},
		{"en", -365 * 24 * time.Hour, "1 year ago"},
		{"ru", -2 * time.Hour, "2 часа назад"},
		{"ru", -5 * time.Hour, "5 часов назад"},
		{"zh", 24 * time.Hour, "1天后"},
		{"ar", -2 * time.Hour, "قبل ساعتين"},
		{"ja", 0, "今"},
	}
	for _, tc := range cases {
		got := linguisticLocale(t, tc.tag).RelativeTime(ref.Add(tc.delta), ref)
		if got != tc.want {
			t.Errorf("RelativeTime(%s, %s) = %q, want %q", tc.tag, tc.delta, got, tc.want)
		}
	}
	// Durations saturate near 292 years, but distant dates still use the complete interval.
	en := linguisticLocale(t, "en")
	far := time.Date(4000, time.January, 1, 12, 0, 0, 0, time.UTC)
	if got := en.RelativeTime(far, ref); !strings.Contains(got, "1,975 years") {
		t.Errorf("distant date = %q", got)
	}
}
