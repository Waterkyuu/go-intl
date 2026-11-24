package intl

import (
	"strings"
	"testing"
)

// BenchmarkPlainInterpolation exercises distinct, repeated, and longer sequences of arguments.
func BenchmarkPlainInterpolation(b *testing.B) {
	values := Values{"sender": "Bob", "recipient": "Ada", "name": "Bob", "id": 42, "long": strings.Repeat("x", 4096)}
	for _, tc := range []struct {
		// pattern is the message rendered by the public translator.
		pattern string
		// want is the expected result for the prepared argument map.
		want string
	}{
		{"{sender} invited {recipient}", "Bob invited Ada"},
		{"{name} says hi to {name}", "Bob says hi to Bob"},
		{"{id}/{id}", "42/42"},
		{strings.Repeat("{name} ", 12), strings.Repeat("Bob ", 12)},
		{"{long}/{long}", strings.Repeat("x", 4096) + "/" + strings.Repeat("x", 4096)},
	} {
		b.Run(tc.pattern, func(b *testing.B) {
			catalog, err := New("en", Define("en", map[string]string{"message": tc.pattern}))
			if err != nil {
				b.Fatal(err)
			}
			t := catalog.Translations("en")
			if got := t("message", values); got != tc.want {
				b.Fatalf("got %q, want %q", got, tc.want)
			}
			b.ReportAllocs()
			for b.Loop() {
				t("message", values)
			}
		})
	}
}
