package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/waterkyuu/go-intl/examples/clients/internal/counter"
)

// TestTranslatedView drives the native widgets without opening a desktop window.
func TestTranslatedView(t *testing.T) {
	translations, err := counter.Messages("es")
	if err != nil {
		t.Fatal(err)
	}
	w := &window{t: translations}
	view := ui.NewTester(w.view, 360, 240)
	for _, want := range []string{"0 artículos", "1 artículo", "2 artículos"} {
		if !view.HasText(want) {
			t.Fatalf("visible text = %q, want %q", view.Texts(), want)
		}
		if err := view.Click("Añadir uno"); err != nil {
			t.Fatal(err)
		}
	}
}
