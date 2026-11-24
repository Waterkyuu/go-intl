package intl

import (
	"strings"
	"testing"
)

// BenchmarkSingleInterpolation measures public translations with a bound translator and a prepared argument map.
func BenchmarkSingleInterpolation(b *testing.B) {
	for _, pattern := range []string{"{id}", "{id} online", "Hello, {id}", "Hello, {id}!", "Item {id}."} {
		catalog, err := New("en", Define("en", map[string]string{"message": pattern}))
		if err != nil {
			b.Fatal(err)
		}
		translate := catalog.Translations("en")
		for _, value := range []any{"Ada", 42} {
			name, want := "string", "Ada"
			if value == 42 {
				name, want = "int", "42"
			}
			args := Values{"id": value}
			b.Run(pattern+"/"+name, func(b *testing.B) {
				expected := strings.ReplaceAll(pattern, "{id}", want)
				if got := translate("message", args); got != expected {
					b.Fatalf("got %q, want %q", got, expected)
				}
				b.ReportAllocs()
				for b.Loop() {
					translate("message", args)
				}
			})
		}
	}
}
