// Package benchmarks compares the public translation APIs used in the README.
package benchmarks

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	intl "github.com/waterkyuu/go-intl"
	"golang.org/x/text/language"
)

// interpolation describes an English message rendered with prepared arguments.
type interpolation struct {
	// name identifies the case in go test output.
	name string
	// pattern uses go-intl syntax for ordinary placeholders.
	pattern string
	// values are shared read-only by both implementations.
	values map[string]any
	// want is the expected translation checked before timing.
	want string
}

// fields builds the slash-delimited messages from the release comparison.
func fields(count int, repeat bool) interpolation {
	tc := interpolation{name: "fields" + strconv.Itoa(count), values: make(map[string]any)}
	if repeat {
		tc.name = "repeat" + strconv.Itoa(count)
	}
	placeholders := make([]string, count)
	words := make([]string, count)
	for i := range count {
		index := i
		if repeat {
			index = 0
		}
		key := fmt.Sprintf("k%02d", index)
		words[i] = fmt.Sprintf("v%02d", index)
		tc.values[key] = words[i]
		placeholders[i] = "{" + key + "}"
	}
	tc.pattern = strings.Join(placeholders, "/")
	tc.want = strings.Join(words, "/")
	return tc
}

// BenchmarkTranslation compares ten ordinary interpolations, two static reads, and a plural.
func BenchmarkTranslation(b *testing.B) {
	first := map[string]any{"first": "Ada"}
	names := map[string]any{"first": "Ada", "last": "Lovelace"}
	for _, tc := range []interpolation{
		{name: "one", pattern: "{first}", values: first, want: "Ada"},
		{name: "hello", pattern: "Hello, {first}!", values: first, want: "Hello, Ada!"},
		{name: "integer", pattern: "Item {id}.", values: map[string]any{"id": 42}, want: "Item 42."},
		{name: "separated", pattern: "{first} {last}", values: names, want: "Ada Lovelace"},
		{name: "adjacent", pattern: "{first}{last}", values: names, want: "AdaLovelace"},
		fields(4, false), fields(8, false), fields(16, false), fields(32, false), fields(16, true),
	} {
		b.Run(tc.name, tc.benchmark)
	}
	benchmarkLookupsAndPlural(b)
}

// benchmark verifies and warms each single-message dictionary before its timed loop.
func (tc interpolation) benchmark(b *testing.B) {
	b.Run("go-intl", func(b *testing.B) {
		catalog, err := intl.New("en", intl.Define("en", map[string]string{"message": tc.pattern}))
		if err != nil {
			b.Fatal(err)
		}
		t := catalog.Translations("en")
		args := intl.Values(tc.values)
		check(b, t("message", args), tc.want)
		for b.Loop() {
			t("message", args)
		}
	})
	b.Run("go-i18n", func(b *testing.B) {
		pattern := strings.NewReplacer("{", "{{.", "}", "}}").Replace(tc.pattern)
		bundle := i18n.NewBundle(language.English)
		bundle.MustAddMessages(language.English, &i18n.Message{ID: "message", Other: pattern})
		loc := i18n.NewLocalizer(bundle, "en")
		cfg := &i18n.LocalizeConfig{MessageID: "message", TemplateData: tc.values}
		check(b, loc.MustLocalize(cfg), tc.want)
		for b.Loop() {
			loc.MustLocalize(cfg)
		}
	})
}

// benchmarkLookupsAndPlural uses a shared key sequence and equivalent 1,026-message dictionaries.
func benchmarkLookupsAndPlural(b *testing.B) {
	messages := map[string]string{
		"screen.welcome": "Hello, {name}!",
		"screen.items":   "{count, plural, one {# item} other {# items}}",
	}
	reference := []*i18n.Message{
		{ID: "screen.welcome", Other: "Hello, {{.name}}!"},
		{ID: "screen.items", One: "{{.PluralCount}} item", Other: "{{.PluralCount}} items"},
	}
	var keys, probes [1024]string
	for i := range keys {
		key := fmt.Sprintf("screen.label%05d", i)
		keys[i] = key
		messages[key] = "Ready"
		reference = append(reference, &i18n.Message{ID: key, Other: "Ready"})
	}
	for i := range probes {
		// The odd stride visits every key once. Clone avoids pointer-equal lookups.
		probes[i] = strings.Clone(keys[(i*541+13)&(len(keys)-1)])
	}

	catalog, err := intl.New("en", intl.Define("en", messages))
	if err != nil {
		b.Fatal(err)
	}
	t := catalog.Translations("en")
	bundle := i18n.NewBundle(language.English)
	bundle.MustAddMessages(language.English, reference...)
	loc := i18n.NewLocalizer(bundle, "en")
	b.Run("hot/go-intl", func(b *testing.B) {
		key := probes[0]
		check(b, t(key), "Ready")
		for b.Loop() {
			t(key)
		}
	})
	b.Run("hot/go-i18n", func(b *testing.B) {
		cfg := &i18n.LocalizeConfig{MessageID: probes[0]}
		check(b, loc.MustLocalize(cfg), "Ready")
		for b.Loop() {
			loc.MustLocalize(cfg)
		}
	})
	b.Run("spread/go-intl", func(b *testing.B) {
		for _, key := range probes {
			check(b, t(key), "Ready")
		}
		var i int
		for b.Loop() {
			t(probes[i&(len(probes)-1)])
			i++
		}
	})
	b.Run("spread/go-i18n", func(b *testing.B) {
		for _, key := range probes {
			check(b, loc.MustLocalize(&i18n.LocalizeConfig{MessageID: key}), "Ready")
		}
		var i int
		for b.Loop() {
			loc.MustLocalize(&i18n.LocalizeConfig{MessageID: probes[i&(len(probes)-1)]})
			i++
		}
	})
	b.Run("plural/go-intl", func(b *testing.B) {
		args := intl.Values{"count": 7}
		check(b, t("screen.items", args), "7 items")
		for b.Loop() {
			t("screen.items", args)
		}
	})
	b.Run("plural/go-i18n", func(b *testing.B) {
		cfg := &i18n.LocalizeConfig{
			MessageID:    "screen.items",
			PluralCount:  7,
			TemplateData: map[string]any{"PluralCount": 7},
		}
		check(b, loc.MustLocalize(cfg), "7 items")
		for b.Loop() {
			loc.MustLocalize(cfg)
		}
	})
}

// check rejects mismatched translations before measuring them.
func check(b *testing.B, got, want string) {
	b.Helper()
	if got != want {
		b.Fatalf("got %q, want %q", got, want)
	}
}
