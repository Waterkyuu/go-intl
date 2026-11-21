package intl

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

// benchmarkMessages builds a realistic namespace and an evenly distributed key trace.
func benchmarkMessages(size int) (map[string]string, []string) {
	messages := make(map[string]string, size+2)
	keys := make([]string, size)
	for i := range size {
		key := fmt.Sprintf("screen.label%05d", i)
		messages[key] = "Ready"
		keys[i] = key
	}
	queries := make([]string, size)
	for i := range queries {
		// Odd multiplication permutes every identifier in this power-of-two data set.
		queries[i] = strings.Clone(keys[(i*541+13)&(size-1)])
	}
	messages["screen.welcome"] = "Hello, {name}!"
	messages["screen.items"] = "{count, plural, one {# item} other {# items}}"
	return messages, queries
}

// BenchmarkCatalogBuild measures compilation from Go messages, excluding fixture generation.
func BenchmarkCatalogBuild(b *testing.B) {
	for _, size := range []int{16, 1024, 16384} {
		messages, _ := benchmarkMessages(size)
		source := Define("en", messages)
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := New("en", source); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkJSONStartup includes parsing a bundled locale file and compiling its catalog.
func BenchmarkJSONStartup(b *testing.B) {
	messages, _ := benchmarkMessages(1024)
	data, err := json.Marshal(messages)
	if err != nil {
		b.Fatal(err)
	}
	bundle := fstest.MapFS{"locales/en.json": {Data: data}}
	b.ReportAllocs()
	for b.Loop() {
		sources, err := LoadFS(bundle, "locales/*.json")
		if err != nil {
			b.Fatal(err)
		}
		if _, err := New("en", sources...); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTranslate measures successful public-API calls after catalog construction.
func BenchmarkTranslate(b *testing.B) {
	messages, queries := benchmarkMessages(1024)
	catalog, err := New("en", Define("en", messages))
	if err != nil {
		b.Fatal(err)
	}
	t := catalog.Translations("en")
	locale := catalog.Locale("en")
	scoped := locale.Translations("screen")
	name := Values{"name": "Ada"}
	count := Values{"count": 7}
	b.Run("hot", func(b *testing.B) {
		key := queries[0]
		if got := t(key); got != "Ready" {
			b.Fatal(got)
		}
		for b.Loop() {
			t(key)
		}
	})
	b.Run("scoped", func(b *testing.B) {
		key := strings.TrimPrefix(queries[0], "screen.")
		if got := scoped(key); got != "Ready" {
			b.Fatal(got)
		}
		for b.Loop() {
			scoped(key)
		}
	})
	b.Run("spread", func(b *testing.B) {
		var i int
		for b.Loop() {
			t(queries[i&(len(queries)-1)])
			i++
		}
	})
	b.Run("interpolate", func(b *testing.B) {
		if got := t("screen.welcome", name); got != "Hello, Ada!" {
			b.Fatal(got)
		}
		for b.Loop() {
			t("screen.welcome", name)
		}
	})
	b.Run("inline-values", func(b *testing.B) {
		for b.Loop() {
			t("screen.welcome", Values{"name": "Ada"})
		}
	})
	b.Run("plural", func(b *testing.B) {
		if got := t("screen.items", count); got != "7 items" {
			b.Fatal(got)
		}
		for b.Loop() {
			t("screen.items", count)
		}
	})
	b.Run("bound-and-read", func(b *testing.B) {
		for b.Loop() {
			scoped("welcome", name)
		}
	})
	b.Run("bind-namespace-and-read", func(b *testing.B) {
		for b.Loop() {
			locale.Translations("screen")("welcome", name)
		}
	})
	b.Run("bind-and-read", func(b *testing.B) {
		for b.Loop() {
			catalog.Translations("en", "screen")("welcome", name)
		}
	})
	b.Run("parallel-hot", func(b *testing.B) {
		key := queries[0]
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				t(key)
			}
		})
	})
	b.Run("parallel-spread", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			var i int
			for pb.Next() {
				t(queries[i&(len(queries)-1)])
				i++
			}
		})
	})
}

// BenchmarkLocalizedPlural exercises localized digits and grouping across the formatting path.
func BenchmarkLocalizedPlural(b *testing.B) {
	for locale, want := range map[string]string{"en": "12,345", "fr": "12\u00a0345", "ar": "١٢٬٣٤٥"} {
		b.Run(locale, func(b *testing.B) {
			source := Define(locale, map[string]string{"items": "{count, plural, other {#}}"})
			catalog, err := New(locale, source)
			if err != nil {
				b.Fatal(err)
			}
			t := catalog.Translations(locale)
			values := Values{"count": 12345}
			if got := t("items", values); got != want {
				b.Fatalf("locale %s produced %q, want %q", locale, got, want)
			}
			for b.Loop() {
				t("items", values)
			}
		})
	}
}
