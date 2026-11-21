// Package intl compiles translation sources into a read-only, locale-aware catalog.
package intl

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/bep/golocales"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Catalog owns compiled messages and can be shared by goroutines after New returns.
type Catalog struct {
	// locales prioritizes less specific tags when the matcher finds equally close dialects.
	locales []dictionary
	// defaultIndex identifies the application fallback even if the matcher finds no language.
	defaultIndex int
	// matcher negotiates among tags in the same order as locales.
	matcher language.Matcher
}

// dictionary owns the messages explicitly defined for one language.
type dictionary struct {
	// tag applies the message author's language rules when a fallback is rendered.
	tag language.Tag
	// printer holds immutable numeric formatting rules shared by renders in this locale.
	printer *message.Printer
	// dates holds the locale’s resolved CLDR date and currency patterns.
	dates golocales.Translator
	// keys maps full paths to positions in the packed program slice.
	keys map[string]int
	// namespaces indexes direct children by short key for allocation-free scoped calls.
	namespaces map[string]map[string]int
	// programs contains only messages actually defined in this language.
	programs []compiledMessage
	// chain lists this language, its registered parents and the application default.
	chain []int
}

// New validates and compiles all sources before returning a usable catalog.
// Sources for one language may be split across files if their keys are disjoint.
func New(defaultLocale string, sources ...Source) (*Catalog, error) {
	base, err := language.Parse(defaultLocale)
	if err != nil {
		return nil, fmt.Errorf("default locale %q: %w", defaultLocale, err)
	}
	tags := []language.Tag{base}
	indices := map[language.Tag]int{base: 0}
	locales := []dictionary{newDictionary(base)}
	var hasDefault bool
	for _, source := range sources {
		tag, err := language.Parse(source.locale)
		if err != nil {
			return nil, fmt.Errorf("source locale %q: %w", source.locale, err)
		}
		idx, found := indices[tag]
		if !found {
			idx = len(tags)
			tags = append(tags, tag)
			indices[tag] = idx
			locales = append(locales, newDictionary(tag))
		}
		if tag == base {
			hasDefault = true
		}
		if err := locales[idx].add(source); err != nil {
			return nil, fmt.Errorf("compile %s %q: %w", tag, source.origin, err)
		}
	}
	if !hasDefault {
		return nil, fmt.Errorf("missing source for default locale %q", base)
	}
	// CLDR matching uses input order to break ties. Prefer a general translation
	// over an equally close sibling dialect, regardless of the source file order.
	slices.SortStableFunc(locales, func(a, b dictionary) int {
		return cmp.Compare(strings.Count(a.tag.String(), "-"), strings.Count(b.tag.String(), "-"))
	})
	for i := range locales {
		tags[i] = locales[i].tag
		indices[tags[i]] = i
	}
	for i, tag := range tags {
		chain := []int{i}
		for _, ancestor := range []language.Tag{tag.Parent(), base} {
			for ; ancestor != language.Und; ancestor = ancestor.Parent() {
				if idx, exists := indices[ancestor]; exists && !slices.Contains(chain, idx) {
					chain = append(chain, idx)
				}
			}
		}
		if !slices.Contains(chain, indices[base]) {
			chain = append(chain, indices[base])
		}
		locales[i].chain = chain
	}
	return &Catalog{locales: locales, matcher: language.NewMatcher(tags), defaultIndex: indices[base]}, nil
}

// newDictionary initializes the indexes populated while compiling one locale.
func newDictionary(tag language.Tag) dictionary {
	return dictionary{
		tag:        tag,
		printer:    message.NewPrinter(tag),
		dates:      dateLocale(tag),
		namespaces: make(map[string]map[string]int),
	}
}

// add indexes compiled messages by full key and by their immediate namespace.
func (d *dictionary) add(s Source) error {
	if d.keys == nil {
		d.keys = make(map[string]int, len(s.messages))
	}
	d.programs = slices.Grow(d.programs, len(s.messages))
	for key, raw := range s.messages {
		if !validMessageKey(key) {
			return fmt.Errorf("invalid key %q", key)
		}
		if _, exists := d.keys[key]; exists {
			return fmt.Errorf("duplicate key %q", key)
		}
		program, err := compileMessage(raw)
		if err != nil {
			return fmt.Errorf("key %q: %w", key, err)
		}
		idx := len(d.programs)
		d.programs = append(d.programs, program)
		d.keys[key] = idx
		if dot := strings.LastIndexByte(key, '.'); dot >= 0 {
			ns := key[:dot]
			if d.namespaces[ns] == nil {
				d.namespaces[ns] = make(map[string]int)
			}
			d.namespaces[ns][key[dot+1:]] = idx
		}
	}
	return nil
}

// Locale binds the closest supported language to a desktop or terminal view.
// An invalid or unsupported preference selects the default language.
func (c *Catalog) Locale(preferred string) Locale {
	tag, err := language.Parse(preferred)
	if err != nil {
		return c.bind(c.defaultIndex)
	}
	return c.matchTags(tag)
}

// Match binds an HTTP Accept-Language preference list using its quality weights.
// An empty, malformed or unsupported list selects the default language.
func (c *Catalog) Match(acceptLanguage string) Locale {
	tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
	if err != nil {
		return c.bind(c.defaultIndex)
	}
	return c.matchTags(tags...)
}

// matchTags honors the configured default for empty or unrelated preferences.
func (c *Catalog) matchTags(tags ...language.Tag) Locale {
	_, idx, confidence := c.matcher.Match(tags...)
	if confidence == language.No {
		return c.bind(c.defaultIndex)
	}
	return c.bind(idx)
}

// bind captures the previously compiled fallback chain for this language.
func (c *Catalog) bind(idx int) Locale {
	return Locale{catalog: c, chain: c.locales[idx].chain}
}

// Translations returns a callable translator for a locale and optional namespace path.
func (c *Catalog) Translations(locale string, namespace ...string) Translator {
	return c.Locale(locale).Translations(namespace...)
}

// Locale represents an immutable language selection with an optional namespace.
type Locale struct {
	// catalog owns all referenced dictionaries.
	catalog *Catalog
	// chain resolves missing keys without repeating language negotiation.
	chain []int
	// prefix is the full namespace without its trailing dot.
	prefix string
	// direct indexes immediate keys in the first-choice namespace.
	direct map[string]int
}

// Tag returns the supported language selected by the matcher.
func (l Locale) Tag() language.Tag { return l.catalog.locales[l.chain[0]].tag }

// Scope binds a nested namespace. Dotted and consecutive path components are equivalent.
func (l Locale) Scope(namespace ...string) Locale {
	for _, ns := range namespace {
		if ns != "" {
			l.prefix = l.fullKey(ns)
		}
	}
	d := &l.catalog.locales[l.chain[0]]
	l.direct = d.namespaces[l.prefix]
	return l
}

// Translator formats a key with zero or one named value map. A formatting or
// lookup error yields the fully qualified key so a UI can continue to render.
type Translator func(key string, values ...Values) string

// Translations returns a reusable view or request translator.
func (l Locale) Translations(namespace ...string) Translator {
	return l.Scope(namespace...).Text
}

// ErrMissingMessage identifies a key missing throughout the fallback chain.
var ErrMissingMessage = errors.New("missing message")

// Text formats a key, returning its fully qualified path on any error.
func (l Locale) Text(key string, values ...Values) string {
	s, err := l.Format(key, values...)
	if err != nil {
		return l.fullKey(key)
	}
	return s
}

// Format renders a message and preserves lookup and value errors for callers.
func (l Locale) Format(key string, values ...Values) (string, error) {
	var args Values
	switch len(values) {
	case 0:
	case 1:
		args = values[0]
	default:
		return "", fmt.Errorf("format %q: expected one values map", l.fullKey(key))
	}
	program, dict := l.lookup(key)
	if program == nil {
		return "", fmt.Errorf("%s %q: %w", l.Tag(), l.fullKey(key), ErrMissingMessage)
	}
	text, err := program.render(dict, args)
	if err != nil {
		return "", fmt.Errorf("format %s %q: %w", dict.tag, l.fullKey(key), err)
	}
	return text, nil
}

// Has reports whether lookup succeeds, including inherited and empty messages.
func (l Locale) Has(key string) bool {
	program, _ := l.lookup(key)
	return program != nil
}

// Raw returns the original message pattern and whether lookup succeeded.
func (l Locale) Raw(key string) (string, bool) {
	program, _ := l.lookup(key)
	if program == nil {
		return "", false
	}
	return program.raw, true
}

// lookup uses the bound namespace for direct keys and consults parents only on a miss.
func (l Locale) lookup(key string) (*compiledMessage, *dictionary) {
	first := &l.catalog.locales[l.chain[0]]
	keys := first.keys
	if l.prefix != "" {
		keys = l.direct
	}
	if idx, found := keys[key]; found {
		return &first.programs[idx], first
	}
	chain := l.chain[1:]
	if l.prefix != "" {
		key = l.fullKey(key)
		// A relative dotted path can also be found in the first dictionary.
		chain = l.chain
	}
	for _, i := range chain {
		d := &l.catalog.locales[i]
		if idx, found := d.keys[key]; found {
			return &d.programs[idx], d
		}
	}
	return nil, nil
}

// fullKey qualifies a relative key for diagnostics and cross-namespace lookup.
func (l Locale) fullKey(key string) string {
	if l.prefix == "" {
		return key
	}
	return l.prefix + "." + key
}
