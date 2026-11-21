package intl

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"path"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
	"golang.org/x/text/language"
)

// Source holds messages for a single locale. Catalog construction combines sources.
type Source struct {
	// locale is the source language tag, including optional region or script subtags.
	locale string
	// messages maps dotted keys to untranslated message patterns for compilation.
	messages map[string]string
	// origin identifies a filesystem source in diagnostics and is empty for Go definitions.
	origin string
}

// Define snapshots dotted message keys from native Go code for use with New.
func Define(locale string, messages map[string]string) Source {
	return Source{locale: locale, messages: maps.Clone(messages)}
}

// LoadFS reads catalog files from an fs.FS, including embed.FS and os.DirFS.
// Each path.Match pattern must match. Overlapping patterns load each file once.
// Use a BCP 47 tag as the basename, for example "locales/pt-BR.toml".
// JSON, TOML and YAML accept nested objects with string leaves or dotted keys.
// CSV requires the header "key,message" followed by one dotted key and message per row.
func LoadFS(fsys fs.FS, patterns ...string) ([]Source, error) {
	if fsys == nil {
		return nil, errors.New("load locales: nil filesystem")
	}
	if len(patterns) == 0 {
		return nil, errors.New("load locales: a filename or glob is required")
	}
	loaded := make(map[string]struct{})
	sources := make([]Source, 0, len(patterns))
	for _, pattern := range patterns {
		names, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("match locale files %q: %w", pattern, err)
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("match locale files %q: %w", pattern, fs.ErrNotExist)
		}
		for _, name := range names {
			if _, ok := loaded[name]; ok {
				continue
			}
			src, err := loadSource(fsys, name)
			if err != nil {
				return nil, fmt.Errorf("load locale file %q: %w", name, err)
			}
			loaded[name] = struct{}{}
			sources = append(sources, src)
		}
	}
	return sources, nil
}

// loadSource derives the language from the filename and owns the decoded strings.
func loadSource(fsys fs.FS, name string) (Source, error) {
	ext := path.Ext(name)
	locale := strings.TrimSuffix(path.Base(name), ext)
	if _, err := language.Parse(locale); err != nil {
		return Source{}, fmt.Errorf("parse locale %q: %w", locale, err)
	}
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return Source{}, err
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	messages := make(map[string]string)
	switch strings.ToLower(ext) {
	case ".json":
		err = readJSONMessages(data, messages)
	case ".toml":
		var object map[string]any
		err = toml.Unmarshal(data, &object)
		if err == nil {
			err = flattenMessages(messages, "", object)
		}
	case ".yaml", ".yml":
		err = readYAMLMessages(data, messages)
	case ".csv":
		err = readCSVMessages(data, messages)
	default:
		err = fmt.Errorf("unsupported catalog extension %q", ext)
	}
	if err != nil {
		return Source{}, err
	}
	return Source{locale: locale, messages: messages, origin: name}, nil
}

// validMessageKey requires each component of a dotted key to contain visible text.
func validMessageKey(key string) bool {
	for segment := range strings.SplitSeq(key, ".") {
		if strings.TrimSpace(segment) == "" {
			return false
		}
	}
	return true
}

// childKey combines a namespace and a relative key after validating the suffix.
func childKey(prefix, key string) (string, error) {
	if !validMessageKey(key) {
		return "", fmt.Errorf("invalid message key %q under %q", key, prefix)
	}
	if prefix == "" {
		return key, nil
	}
	return prefix + "." + key, nil
}

// putMessage rejects collisions between flattened paths, including identical translations.
func putMessage(messages map[string]string, key, value string) error {
	if _, ok := messages[key]; ok {
		return fmt.Errorf("duplicate message key %q", key)
	}
	messages[key] = value
	return nil
}

// flattenMessages walks decoded TOML or YAML tables and requires string leaves.
func flattenMessages(messages map[string]string, prefix string, object map[string]any) error {
	for key, child := range object {
		full, err := childKey(prefix, key)
		if err != nil {
			return err
		}
		if err := flattenValue(messages, full, child); err != nil {
			return err
		}
	}
	return nil
}

// flattenValue inserts a decoded message or descends into its namespace.
func flattenValue(messages map[string]string, key string, value any) error {
	switch child := value.(type) {
	case string:
		return putMessage(messages, key, child)
	case map[string]any:
		return flattenMessages(messages, key, child)
	default:
		return fmt.Errorf("message %q must be a string or mapping", key)
	}
}

// readJSONMessages tokenizes an object so duplicate fields remain observable.
func readJSONMessages(data []byte, messages map[string]string) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	token, err := dec.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return errors.New("expected a JSON object")
	}
	if err = readJSONObject(dec, "", messages); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("unexpected data after JSON object")
	}
	return nil
}

// readJSONObject consumes all key/value pairs and the closing brace.
func readJSONObject(dec *json.Decoder, prefix string, messages map[string]string) error {
	seen := make(map[string]struct{})
	for dec.More() {
		field, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := field.(string)
		if !ok {
			return errors.New("expected a JSON object key")
		}
		full, err := childKey(prefix, key)
		if err != nil {
			return err
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate JSON key %q", full)
		}
		seen[key] = struct{}{}
		value, err := dec.Token()
		if err != nil {
			return err
		}
		switch value := value.(type) {
		case string:
			err = putMessage(messages, full, value)
		case json.Delim:
			if value != '{' {
				return fmt.Errorf("message %q must be a string or object", full)
			}
			err = readJSONObject(dec, full, messages)
		default:
			return fmt.Errorf("message %q must be a string or object", full)
		}
		if err != nil {
			return err
		}
	}
	_, err := dec.Token()
	return err
}

// readYAMLMessages accepts one mapping document, including block scalar strings.
func readYAMLMessages(data []byte, messages map[string]string) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return err
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return errors.New("expected a YAML mapping with string keys")
	}
	var next yaml.Node
	if err := dec.Decode(&next); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return errors.New("unexpected second YAML document")
	}
	return flattenYAML(messages, "", doc.Content[0])
}

// flattenYAML walks parsed nodes directly so duplicate detection stays linear on wide tables.
func flattenYAML(messages map[string]string, prefix string, node *yaml.Node) error {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].ShortTag() == "!!merge" {
			// Delegate merge precedence and recursive-alias checks to the YAML decoder.
			var merged map[string]any
			if err := node.Decode(&merged); err != nil {
				return err
			}
			return flattenMessages(messages, prefix, merged)
		}
	}
	seen := make(map[string]struct{}, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		field, value := node.Content[i], node.Content[i+1]
		if field.Kind != yaml.ScalarNode || field.ShortTag() != "!!str" {
			return fmt.Errorf("YAML key at line %d must be a string", field.Line)
		}
		full, err := childKey(prefix, field.Value)
		if err != nil {
			return err
		}
		if _, exists := seen[field.Value]; exists {
			return fmt.Errorf("duplicate YAML key %q at line %d", full, field.Line)
		}
		seen[field.Value] = struct{}{}
		switch value.Kind {
		case yaml.MappingNode:
			err = flattenYAML(messages, full, value)
		case yaml.AliasNode:
			var decoded any
			if err = value.Decode(&decoded); err == nil {
				err = flattenValue(messages, full, decoded)
			}
		default:
			if value.Kind != yaml.ScalarNode || value.ShortTag() != "!!str" {
				return fmt.Errorf("message %q must be a string or mapping", full)
			}
			err = putMessage(messages, full, value.Value)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// readCSVMessages accepts RFC 4180 quoting and preserves whitespace in translations.
func readCSVMessages(data []byte, messages map[string]string) error {
	dec := csv.NewReader(bytes.NewReader(data))
	dec.FieldsPerRecord = 2
	dec.ReuseRecord = true
	header, err := dec.Read()
	if err != nil {
		return fmt.Errorf("read CSV header: %w", err)
	}
	if header[0] != "key" || header[1] != "message" {
		return errors.New("expected CSV header key,message")
	}
	for {
		row, err := dec.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if !validMessageKey(row[0]) {
			return fmt.Errorf("invalid message key %q", row[0])
		}
		if err = putMessage(messages, row[0], row[1]); err != nil {
			return err
		}
	}
}

// Messages returns an editable copy of the source's original patterns.
func (s Source) Messages() map[string]string { return maps.Clone(s.messages) }

// Locale returns the BCP 47 locale identifier supplied with this source.
func (s Source) Locale() string { return s.locale }
