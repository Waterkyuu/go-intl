// Command gen regenerates the cldrtext tables in the current directory.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// pool assigns stable indices to distinct Go literals in first-seen order.
type pool struct {
	// literals contains the data to emit in index order.
	literals []string
	// index maps each literal to its assigned position.
	index map[string]int
}

// add returns the index of a literal, allocating a new entry only on first use.
func (p *pool) add(literal string) int {
	if p.index == nil {
		p.index = make(map[string]int)
	}
	if idx, ok := p.index[literal]; ok {
		return idx
	}
	idx := len(p.literals)
	p.index[literal] = idx
	p.literals = append(p.literals, literal)
	return idx
}

// quote keeps Unicode text, including CLDR spacing and direction marks, in source literals.
func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// writeArray emits literals in the supplied order.
func writeArray(out *bytes.Buffer, name, elementType string, items []string) {
	fmt.Fprintf(out, "\nvar %s = [...]%s{\n", name, elementType)
	for _, s := range items {
		fmt.Fprintf(out, "%s,\n", s)
	}
	out.WriteString("}\n")
}

// listDoc holds the conjunction patterns for one locale.
type listDoc struct {
	// ListPatterns groups the two-item and longer-list patterns by style.
	ListPatterns map[string]map[string]string
}

// dateDoc holds the relative-time fields for one locale.
type dateDoc struct {
	// Dates groups names and numeric patterns by time unit.
	Dates struct {
		// Fields includes relative units alongside unrelated CLDR date fields.
		Fields map[string]struct {
			// Now names the unmodified unit, such as the current instant.
			Now string `json:"relative-type-0"`
			// Future maps plural categories to future relative-time patterns.
			Future map[string]string `json:"relativeTime-type-future"`
			// Past maps plural categories to past relative-time patterns.
			Past map[string]string `json:"relativeTime-type-past"`
		}
	}
}

// phraseSlots follows the row layout read by cldrtext.Table.
func phraseSlots(list listDoc, date dateDoc) ([]string, error) {
	slots := make([]string, 0, 93)
	for _, part := range []string{"2", "start", "middle", "end"} {
		pattern := list.ListPatterns["listPattern-type-standard"][part]
		body, ok := strings.CutPrefix(pattern, "{0}")
		if !ok || strings.Count(body, "{1}") != 1 {
			return nil, fmt.Errorf("unsupported list pattern %q", pattern)
		}
		between, after, _ := strings.Cut(body, "{1}")
		slots = append(slots, between, after)
	}
	fields := date.Dates.Fields
	slots = append(slots, fields["second"].Now)
	for _, unit := range []string{"second", "minute", "hour", "day", "week", "month", "year"} {
		field := fields[unit]
		for _, patterns := range []map[string]string{field.Future, field.Past} {
			fallback, ok := patterns["relativeTimePattern-count-other"]
			if !ok {
				return nil, fmt.Errorf("missing %s relative-time pattern for other", unit)
			}
			// Form order matches golang.org/x/text/feature/plural.Form.
			for _, form := range []string{"other", "zero", "one", "two", "few", "many"} {
				s, ok := patterns["relativeTimePattern-count-"+form]
				if !ok {
					s = fallback
				}
				slots = append(slots, s)
			}
		}
	}
	return slots, nil
}

// writePhrases builds a shared word pool and deduplicates the rows that index it.
func writePhrases(out *bytes.Buffer, license []byte) error {
	lists, err := loadLocales[listDoc]("cldr-misc-full", "listPatterns.json", license)
	if err != nil {
		return err
	}
	dates, err := loadLocales[dateDoc]("cldr-dates-full", "dateFields.json", license)
	if err != nil {
		return err
	}
	if len(lists) != len(dates) {
		return errors.New("locale coverage differs between list and date data")
	}
	var words, rows pool
	tags := slices.Sorted(maps.Keys(lists))
	indexes := make([]string, len(tags))
	for i, tag := range tags {
		date, ok := dates[tag]
		if !ok {
			return fmt.Errorf("missing date data for %s", tag)
		}
		slots, err := phraseSlots(lists[tag], date)
		if err != nil {
			return fmt.Errorf("%s: %w", tag, err)
		}
		for j, word := range slots {
			slots[j] = strconv.Itoa(words.add(quote(word)))
		}
		idx := rows.add("{" + strings.Join(slots, ",") + "}")
		indexes[i] = strconv.Itoa(idx)
		tags[i] = quote(tag)
	}
	if max(len(words.literals), len(rows.literals)) > 1<<16 {
		return errors.New("phrase data exceeds uint16 index width")
	}
	writeArray(out, "locales", "string", tags)
	writeArray(out, "localeRows", "uint16", indexes)
	writeArray(out, "words", "string", words.literals)
	writeArray(out, "rows", "[93]uint16", rows.literals)
	return nil
}

// generate replaces data.go only after every archive and the generated syntax pass validation.
func generate() error {
	license, err := os.ReadFile("LICENSE")
	if err != nil {
		return err
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "// Code generated from Unicode CLDR %s by go generate. DO NOT EDIT.\n", version)
	out.WriteString("// Licensed under the Unicode License v3. See LICENSE.\npackage cldrtext\n")
	if err := writePhrases(&out, license); err != nil {
		return err
	}
	if err := writeCurrencies(&out, license); err != nil {
		return err
	}
	source, err := format.Source(out.Bytes())
	if err != nil {
		return fmt.Errorf("format generated Go code: %w", err)
	}
	dir, err := os.MkdirTemp(".", ".cldr-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	file := filepath.Join(dir, "data.go")
	if err := os.WriteFile(file, source, 0o644); err != nil {
		return err
	}
	return os.Rename(file, "data.go")
}

// main reports generation failures to go generate.
func main() {
	if err := generate(); err != nil {
		log.Fatal(err)
	}
}
