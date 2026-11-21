package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/waterkyuu/go-intl/examples/clients/internal/counter"
)

// TestTranslatedView drives the real model through its singular and plural states.
func TestTranslatedView(t *testing.T) {
	translations, err := counter.Messages("es")
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = model{t: translations}
	for _, want := range []string{"0 artículos", "1 artículo", "2 artículos"} {
		if got := m.View().Content; !strings.HasPrefix(got, want+"\n") {
			t.Fatalf("view = %q, want first line %q", got, want)
		}
		m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	}
}
