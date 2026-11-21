package intl_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/waterkyuu/go-intl"
	"golang.org/x/text/language"
)

// testCatalog supplies languages with regional overrides, empty strings and fallback messages.
func testCatalog(t *testing.T) *intl.Catalog {
	t.Helper()
	c, err := intl.New("en",
		intl.Define("fr-CA", map[string]string{"Menu.save": "Sauvegarder"}),
		intl.Define("fr", map[string]string{"Menu.save": "Enregistrer", "Menu.empty": ""}),
		intl.Define("en", map[string]string{
			"Menu.save": "Save", "Menu.empty": "nonempty", "Menu.file.open": "Open",
			"Menu.items": "{n, plural, one {# item} other {# items}}",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestLookup covers negotiation, scoping and per-key fallback with the message author's rules.
func TestLookup(t *testing.T) {
	c := testCatalog(t)
	view := c.Match("es;q=0.3, fr-CA;q=1, en;q=0.8").Scope("Menu")
	if got := view.Tag().String(); got != "fr-CA" {
		t.Fatalf("matched %s", got)
	}
	if view.Text("save") != "Sauvegarder" || view.Scope("file").Text("open") != "Open" {
		t.Fatal("expected regional override and nested English fallback")
	}
	if !view.Has("empty") || view.Text("empty") != "" || view.Has("missing") {
		t.Fatal("empty translations must override fallback")
	}
	raw, found := view.Raw("items")
	if !found || !strings.Contains(raw, "plural") {
		t.Fatal("expected inherited ICU pattern")
	}
	if text := view.Text("items", intl.Values{"n": 0}); text != "0 items" {
		t.Fatalf("English fallback pluralized as %q", text)
	}
	if _, err := view.Format("missing"); !errors.Is(err, intl.ErrMissingMessage) {
		t.Fatal("missing key error lost its cause")
	}
	if text, err := view.Format("items"); err == nil || text != "" {
		t.Fatalf("missing argument returned %q and %v", text, err)
	}
	if text := view.Text("invalid"); text != "Menu.invalid" {
		t.Fatalf("missing key fallback = %q", text)
	}
	if _, err := view.Format("save", intl.Values{}, intl.Values{}); err == nil {
		t.Fatal("accepted extra value maps")
	}
	if c.Translations("fr-BE", "Menu")("save") != "Enregistrer" {
		t.Fatal("expected French for Belgian locale")
	}
	if text := c.Translations("en", "Menu")("file.open"); text != "Open" {
		t.Fatalf("relative dotted lookup = %q", text)
	}
	for _, header := range []string{"", "de", "bad@@header", "fr;q=0, en;q=0.8"} {
		if tag := c.Match(header).Tag(); tag != language.English {
			t.Errorf("Match(%q) = %s", header, tag)
		}
	}
}

// TestRegionalDefault checks fallback after the matcher orders general tags before regional ones.
func TestRegionalDefault(t *testing.T) {
	c, err := intl.New("en-GB",
		intl.Define("en-GB", map[string]string{"spelling": "colour"}),
		intl.Define("fr", map[string]string{"title": "Accueil"}),
		intl.Define("en", map[string]string{"common": "Help"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, preferred := range []string{"", "??", "ko"} {
		if locale := c.Locale(preferred); locale.Tag().String() != "en-GB" || locale.Text("common") != "Help" {
			t.Errorf("unexpected default or parent for %q", preferred)
		}
	}
	if got := c.Translations("fr")("common"); got != "Help" {
		t.Errorf("expected the regional default's parent, got %q", got)
	}
}

// TestSourceComposition builds a single catalog from embedded-style data and Go definitions.
func TestSourceComposition(t *testing.T) {
	files := fstest.MapFS{
		"json/en.json": {Data: []byte(`{"Page":{"title":"Inbox"}}`)},
		"toml/de.toml": {Data: []byte("[Page]\ntitle = 'Postfach'")},
		"yaml/es.yml":  {Data: []byte("Page:\n  title: Bandeja")},
		"csv/fr.csv":   {Data: []byte("key,message\nPage.title,Réception")},
	}
	sources, err := intl.LoadFS(files, "*/*")
	if err != nil {
		t.Fatal(err)
	}
	c, err := intl.New("en", append(sources, intl.Define("it", map[string]string{"Page.title": "Posta"}))...)
	if err != nil {
		t.Fatal(err)
	}
	for tag, want := range map[string]string{
		"en": "Inbox", "de": "Postfach", "es": "Bandeja", "fr": "Réception", "it": "Posta",
	} {
		if got := c.Translations(tag, "Page")("title"); got != want {
			t.Errorf("%s = %q, want %q", tag, got, want)
		}
	}
}

// TestConstructionErrors verifies that a bad source never publishes a partial catalog.
func TestConstructionErrors(t *testing.T) {
	for _, sources := range [][]intl.Source{
		nil,
		{intl.Define("fr", nil)},
		{intl.Define("@", nil)},
		{intl.Define("en", map[string]string{".broken": ""})},
		{intl.Define("en", map[string]string{"a": "{"})},
		{intl.Define("en", map[string]string{"a": ""}), intl.Define("EN", map[string]string{"a": ""})},
	} {
		if c, err := intl.New("en", sources...); c != nil || err == nil {
			t.Fatalf("invalid catalog returned %v, %v", c, err)
		}
	}
	if _, err := intl.New("@"); err == nil {
		t.Fatal("accepted invalid default language")
	}
}

// TestConcurrentTranslations shares a catalog and a scoped translator across client and request goroutines.
func TestConcurrentTranslations(t *testing.T) {
	c := testCatalog(t)
	ui := c.Translations("en", "Menu")
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 100 {
				if ui("items", intl.Values{"n": 2345}) != "2,345 items" ||
					c.Match("fr-CA").Text("Menu.save") != "Sauvegarder" {
					t.Error("inconsistent concurrent translation")
					return
				}
			}
		})
	}
	wg.Wait()
}

// ExampleCatalog_Match serves two requests from one catalog with isolated language preferences.
func ExampleCatalog_Match() {
	catalog, err := intl.New("en",
		intl.Define("en", map[string]string{"Home.title": "Inbox"}),
		intl.Define("fr", map[string]string{"Home.title": "Boîte de réception"}),
	)
	if err != nil {
		panic(err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		view := catalog.Match(r.Header.Get("Accept-Language"))
		w.Header().Set("Content-Language", view.Tag().String())
		w.Header().Add("Vary", "Accept-Language")
		_, _ = fmt.Fprint(w, view.Translations("Home")("title"))
	})
	for _, preference := range []string{"fr-CA, en;q=0.8", "en"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Language", preference)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		fmt.Println(response.Body.String())
	}
	// Output:
	// Boîte de réception
	// Inbox
}
