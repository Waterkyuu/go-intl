package main

import (
	"testing"

	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/waterkyuu/go-intl/examples/clients/internal/counter"
)

// TestTranslatedCounter presses the Spanish button and reads the updated native text.
func TestTranslatedCounter(t *testing.T) {
	translations, err := counter.Messages("es")
	if err != nil {
		t.Fatal(err)
	}
	app := nsgotest.Start(t, counterView(translations))
	for count, want := range []string{"0 artículos", "1 artículo", "2 artículos"} {
		if count > 0 {
			app.Press("role:button Añadir uno")
		}
		if got := app.Find("id:count").Text; got != want {
			t.Fatalf("count = %q, want %q", got, want)
		}
	}
}
