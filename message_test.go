package intl

import (
	"math"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/language"
)

// TestMessageRendering exercises ICU features across locales and numeric representations.
func TestMessageRendering(t *testing.T) {
	for _, tc := range []struct {
		// source is the ICU template supplied by the translator.
		source string
		// tag controls plural rules and locale-aware output.
		tag language.Tag
		// values bind the template's arguments.
		values Values
		// want is the expected visible text.
		want string
	}{
		{"", language.English, nil, ""},
		{"Don't say ''hello'' to '{me}'", language.English, nil, "Don't say 'hello' to {me}"},
		{"Hi {name}!", language.English, Values{"name": "Sam"}, "Hi Sam!"},
		{"{enabled, select, true {on} false {off} other {?}}",
			language.English, Values{"enabled": false}, "off"},
		{"{n, plural, other {# files} one {# file} =1 {exactly #}}",
			language.English, Values{"n": 1}, "exactly 1"},
		{"{n, plural, offset:1 =0 {nobody} one {you and one guest} other {you and # guests}}",
			language.English, Values{"n": 3}, "you and 2 guests"},
		{"{n, plural, offset:08 one {one} other {#}}",
			language.English, Values{"n": 9}, "one"},
		{"{n, plural, offset:1 other {#}}",
			language.English, Values{"n": int64(math.MinInt64)}, "-9,223,372,036,854,775,809"},
		{"{n, plural, offset:1 one {one} other {#}}",
			language.German, Values{"n": 2.5}, "1,5"},
		{"{n, plural, one {one} few {few} many {many} other {other}}",
			language.Russian, Values{"n": int64(100000021)}, "one"},
		{"{n, plural, one {one} few {few} many {many} other {other}}",
			language.Russian, Values{"n": 1.5}, "other"},
		{"{n, plural, one {one} few {few} many {many} other {other}}",
			language.Russian, Values{"n": -24}, "few"},
		{"{n, plural, one {one} two {two} few {few} other {other}}",
			language.Arabic, Values{"n": 2}, "two"},
		{"{n, plural, one {one} other {other}}", language.Chinese, Values{"n": 1}, "other"},
		{"{n, plural, =18446744073709551615 {exact} other {other}}",
			language.English, Values{"n": uint64(math.MaxUint64)}, "exact"},
		{"{n, plural, =18446744073709551614 {wrong} other {other}}",
			language.English, Values{"n": uint64(math.MaxUint64)}, "other"},
		{"{n, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}",
			language.English, Values{"n": 22}, "22nd"},
		{"{n, plural, other {# {m, plural, other {#}} #}}",
			language.English, Values{"n": 2, "m": 3}, "2 3 2"},
		{"{n, plural, other {'#' <b>#</b> {s, select, other {#}}}}",
			language.English, Values{"n": 4, "s": "x", "b": func(s string) string { return "[" + s + "]" }}, "# [4] #"},
		{"{x, select, a {{absent}} other {ok}}", language.English, Values{"x": "b"}, "ok"},
		{"{n, number} / {n, number, integer}", language.German, Values{"n": 1234.5}, "1.234,5 / 1.234"},
		{"{n, number, percent}", language.English, Values{"n": 0.15}, "15%"},
		{"2 < 3 and <br/>", language.English, nil, "2 < 3 and <br/>"},
	} {
		t.Run(tc.source+"/"+tc.tag.String(), func(t *testing.T) {
			m, err := compileMessage(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			if m.raw != tc.source {
				t.Fatalf("raw = %q, want %q", m.raw, tc.source)
			}
			d := newDictionary(tc.tag)
			got, err := m.render(&d, tc.values)
			if err != nil || got != tc.want {
				t.Fatalf("render = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

// TestDateTimeArguments checks that the message format and width reach the calendar formatter.
func TestDateTimeArguments(t *testing.T) {
	d := newDictionary(language.French)
	when := time.Date(2026, time.May, 4, 18, 15, 0, 0, time.UTC)
	for _, kind := range []string{"date", "time"} {
		for _, style := range []string{"", "short", "medium", "long", "full"} {
			pattern := "{v, " + kind
			if style != "" {
				pattern += ", " + style
			}
			m, err := compileMessage(pattern + "}")
			if err != nil {
				t.Fatal(err)
			}
			want := formatDateTime(d.dates, when, kind, style)
			got, err := m.render(&d, Values{"v": when})
			if err != nil || got != want {
				t.Errorf("%s/%s = %q, %v; want %q", kind, style, got, err, want)
			}
		}
	}
}

// TestMessageValidation ensures malformed or excessively deep input fails at load time.
func TestMessageValidation(t *testing.T) {
	for _, s := range []string{
		"{", "}", "{}", "{x", "{x garbage}", "{x, unknown}", "{x, number, unrecognized}",
		"{x, plural, one {x}}", "{x, select, other {1} other {2}}", "{x, plural, few {x}",
		"{x, plural, =1.0 {x} =01 {y} other {z}}", "{x, plural, =. {x} other {z}}",
		"{x, plural, plenty {x} other {y}}", "{x, plural, offset:-1 other {x}}",
		"<a>", "<a></b>", "<a x='y'>value</a>", "{x, date, enormous}",
		strings.Repeat("<b>", 65) + strings.Repeat("</b>", 65),
	} {
		if _, err := compileMessage(s); err == nil {
			t.Errorf("accepted malformed input %q", s)
		}
	}
}

// TestMessageArgumentErrors checks that rendering failures discard the partial string.
func TestMessageArgumentErrors(t *testing.T) {
	d := newDictionary(language.English)
	for source, value := range map[string]any{
		"prefix {missing}": nil, "prefix {x, plural, other {#}}": "2", "{x, number}": math.Inf(1),
		"{x, date}": 1, "<x>a</x>": (func(string) string)(nil),
	} {
		m, err := compileMessage(source)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := m.render(&d, Values{"x": value}); err == nil || got != "" {
			t.Errorf("%q returned %q, %v", source, got, err)
		}
	}
}

// FuzzCompileMessage checks that arbitrary message data is safe to compile and render.
func FuzzCompileMessage(f *testing.F) {
	d := newDictionary(language.English)
	for _, seed := range []string{"{n, plural, one {a} other {#}}", "{", "'{'", "<b>{n}</b>", "{\xff}"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if m, err := compileMessage(source); err == nil {
			_, _ = m.render(&d, Values{"n": 2})
		}
	})
}
