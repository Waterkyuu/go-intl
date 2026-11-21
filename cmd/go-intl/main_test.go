package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// putFixture writes a test source file and returns its absolute path.
func putFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestExtractCommand checks alias resolution, static extraction, locale filtering and stable JSON order.
func TestExtractCommand(t *testing.T) {
	dir := t.TempDir()
	source := putFixture(t, dir, "messages.go", `package catalog
import loc "github.com/waterkyuu/go-intl"
func init() { panic("This must never execute") }
var English = loc.Define("en", map[string]string{"z": "Last", "a": "Hi {name}!"})
var Spanish = loc.Define("es", map[string]string{"a": "Hola"})
func unrelated() {
	loc := struct{ Define func(string, any) }{}
	loc.Define("en", nil)
}
`)
	putFixture(t, dir, "other_test.go", "this file is not part of the catalog")
	putFixture(t, dir, "vendor/unused/unused.go", "skip vendored Go code")
	var out bytes.Buffer
	if err := run([]string{"extract", "-locale", "en", dir, source}, &out); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"a\": \"Hi {name}!\",\n  \"z\": \"Last\"\n}\n"
	if got := out.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	out.Reset()
	if err := run([]string{"extract", "-locale", "de", dir}, &out); err == nil || out.Len() != 0 {
		t.Fatalf("expected missing locale error and empty output, got %v, %q", err, out.String())
	}
}

// TestExtractInvalid prevents silent partial catalogs for unsupported or duplicate literals.
func TestExtractInvalid(t *testing.T) {
	for _, message := range []string{
		"someVariable",
		"map[string]string{dynamicKey: \"hello\"}",
		"map[string]string{\"same\": \"one\", \"same\": \"two\"}",
		"map[string]string{\"bad\": \"{unterminated\"}",
	} {
		dir := t.TempDir()
		source := "package p\nimport \"github.com/waterkyuu/go-intl\"\nvar _ = intl.Define(\"en\", " + message + ")"
		file := putFixture(t, dir, "messages.go", source)
		var out bytes.Buffer
		if err := run([]string{"extract", file}, &out); err == nil || out.Len() != 0 {
			t.Errorf("extract %q returned %v and %q", message, err, out.String())
		}
	}
}

// TestCatalogCommands checks format-independent validation, completeness and additive merging.
func TestCatalogCommands(t *testing.T) {
	dir := t.TempDir()
	base := putFixture(t, dir, "en.toml", "[menu]\nopen = \"Open\"\nsave = \"Save {name}\"\n")
	target := putFixture(t, dir, "fr.csv", "key,message\nmenu.open,Ouvrir\nmenu.quit,Quitter\n")
	extra := putFixture(t, dir, "extra/fr.yml", "menu:\n  save: \"Enregistrer {name}\"")
	var out bytes.Buffer
	err := run([]string{"check", "-base", base, target}, &out)
	if err == nil || !strings.Contains(err.Error(), "menu.save") {
		t.Fatalf("missing key diagnostic: %v", err)
	}
	if err := run([]string{"check", "-base", base, target, extra, base, extra}, &out); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"merge", "-base", base, target}, &out); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"menu.open\": \"Ouvrir\",\n  \"menu.quit\": \"Quitter\",\n  \"menu.save\": \"Save {name}\"\n}\n"
	if got := out.String(); got != want {
		t.Errorf("merge got %q, want %q", got, want)
	}
	target = putFixture(t, dir, "fr.csv", "key,message\nmenu.open,{bad\n")
	for _, cmd := range []string{"merge", "check"} {
		out.Reset()
		if err := run([]string{cmd, "-base", base, target}, &out); err == nil || out.Len() != 0 {
			t.Errorf("%s should reject malformed messages before emitting output", cmd)
		}
	}
}
