package intl

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"strings"
	"testing"
	"testing/fstest"
)

// TestDefine checks that application code can reuse its map after defining a source.
func TestDefine(t *testing.T) {
	messages := map[string]string{"menu.save": "Save"}
	source := Define("en", messages)
	messages["menu.save"] = "Changed"
	if source.locale != "en" || source.messages["menu.save"] != "Save" {
		t.Fatalf("Define returned %+v", source)
	}
}

// TestLoadFSFormats exercises equivalent nested, dotted and multiline messages in every format.
func TestLoadFSFormats(t *testing.T) {
	want := map[string]string{
		"menu.save":    "Save, now",
		"menu.confirm": "Hi {name}!\nContinue?",
		"empty":        "",
	}
	files := map[string]string{
		"json": `{"menu":{"save":"Save, now","confirm":"Hi {name}!\nContinue?"},"empty":""}`,
		"toml": "empty = \"\"\n[menu]\nsave = \"Save, now\"\nconfirm = \"Hi {name}!\\nContinue?\"\n",
		"yaml": "menu:\n  save: Save, now\n  confirm: |-\n    Hi {name}!\n    Continue?\nempty: ''\n",
		"yml":  "menu.save: Save, now\nmenu.confirm: \"Hi {name}!\\nContinue?\"\nempty: ''\n",
		"csv":  "\ufeffkey,message\r\nmenu.save,\"Save, now\"\r\nmenu.confirm,\"Hi {name}!\r\nContinue?\"\r\nempty,\r\n",
	}
	for ext, contents := range files {
		t.Run(ext, func(t *testing.T) {
			name := "locales/zh-Hant-TW." + ext
			filesystem := fstest.MapFS{name: &fstest.MapFile{Data: []byte(contents)}}
			sources, err := LoadFS(filesystem, "locales/*", name)
			if err != nil {
				t.Fatal(err)
			}
			if len(sources) != 1 {
				t.Fatalf("got %d sources, want one for overlapping patterns", len(sources))
			}
			source := sources[0]
			if source.origin != name || source.locale != "zh-Hant-TW" || !maps.Equal(source.messages, want) {
				t.Errorf("got %+v, want %q, zh-Hant-TW, %v", source, name, want)
			}
		})
	}
}

// TestLoadFSInvalidDocuments verifies per-file diagnostics and all-or-nothing results.
func TestLoadFSInvalidDocuments(t *testing.T) {
	cases := []struct {
		// name describes the invalid input.
		name string
		// path supplies a locale filename for format detection.
		path string
		// content contains the exact input document.
		content string
	}{
		{"duplicate JSON key", "en.json", `{"a":{},"a":{}}`},
		{"JSON path collision", "en.json", `{"a":{"b":"hi"},"a.b":"bye"}`},
		{"invalid JSON root", "en.json", `["hello"]`},
		{"invalid JSON leaf", "en.json", `{"a":null}`},
		{"unterminated JSON", "en.json", `{"a":{"b":"hi"}`},
		{"trailing JSON value", "en.json", `{} {}`},
		{"invalid JSON key", "en.json", `{"a..b":"hello"}`},
		{"duplicate TOML key", "en.toml", "a = \"hi\"\na = \"bye\""},
		{"TOML path collision", "en.toml", "\"a.b\"=\"one\"\n[a]\nb=\"two\""},
		{"TOML numeric leaf", "en.toml", "a = 3"},
		{"duplicate YAML key", "en.yaml", "a: hi\na: bye"},
		{"YAML boolean leaf", "en.yaml", "a: true"},
		{"YAML second document", "en.yaml", "a: hi\n---\nb: bye"},
		{"YAML numeric key", "en.yaml", "a:\n  2: hi"},
		{"invalid CSV header", "en.csv", "id,text\na,hi"},
		{"invalid CSV record", "en.csv", "key,message\na,hi,bye"},
		{"duplicate CSV record", "en.csv", "key,message\na,hi\na,bye"},
		{"empty CSV key", "en.csv", "key,message\n,translation"},
		{"unsupported format", "en.txt", "a = hi"},
		{"invalid locale", "invalid%tag.json", "{}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filesystem := fstest.MapFS{
				"fr.json": &fstest.MapFile{Data: []byte("{}")},
				tc.path:   &fstest.MapFile{Data: []byte(tc.content)},
			}
			got, err := LoadFS(filesystem, "fr.json", tc.path)
			if err == nil || !strings.Contains(err.Error(), tc.path) || got != nil {
				t.Fatalf("got %v, %v; want file-specific error with no partial result", got, err)
			}
		})
	}
}

// TestLoadFSYAMLAnchors checks localized copy reused through aliases and merge keys.
func TestLoadFSYAMLAnchors(t *testing.T) {
	data := "base: &base\n  save: &label Save\nmenu:\n  <<: *base\n  cancel: Cancel\naction: *label\ndialog: *base\n"
	files := fstest.MapFS{"en.yml": &fstest.MapFile{Data: []byte(data)}}
	got, err := LoadFS(files, "en.yml")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"base.save":   "Save",
		"menu.save":   "Save",
		"menu.cancel": "Cancel",
		"action":      "Save",
		"dialog.save": "Save",
	}
	if !maps.Equal(got[0].messages, want) {
		t.Fatalf("got %v, want %v", got[0].messages, want)
	}
}

// TestLoadFSGlobs verifies file order, missing files and invalid patterns.
func TestLoadFSGlobs(t *testing.T) {
	files := fstest.MapFS{
		"locales/en.json": &fstest.MapFile{Data: []byte("{}")},
		"locales/de.json": &fstest.MapFile{Data: []byte("{}")},
	}
	got, err := LoadFS(files, "locales/*.json")
	if err != nil || len(got) != 2 {
		t.Fatalf("LoadFS returned %v, %v", got, err)
	}
	if got[0].locale != "de" || got[1].locale != "en" {
		t.Fatalf("unexpected locale order: %v", got)
	}
	if _, err := LoadFS(files, "missing/*.toml"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected missing match, got %v", err)
	}
	if _, err := LoadFS(files, "["); err == nil {
		t.Error("expected invalid pattern error")
	}
	if _, err := LoadFS(files); err == nil {
		t.Error("expected missing pattern error")
	}
	if _, err := LoadFS(nil, "*"); err == nil {
		t.Error("expected missing filesystem error")
	}
}

// BenchmarkLoadFSYAML measures startup work as a single translation table grows.
func BenchmarkLoadFSYAML(b *testing.B) {
	for _, count := range []int{1000, 4000} {
		var data strings.Builder
		for i := range count {
			fmt.Fprintf(&data, "menu.item%d: 'Choice %d'\n", i, i)
		}
		files := fstest.MapFS{"en.yaml": &fstest.MapFile{Data: []byte(data.String())}}
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := LoadFS(files, "en.yaml"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
