// Command go-intl extracts, merges and validates message catalogs.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	intl "github.com/waterkyuu/go-intl"
	"golang.org/x/text/language"
)

// usage describes each command and the output destination.
const usage = `Usage:
  go-intl extract [-locale en] [file.go|directory ...]      Write Go messages as JSON to stdout
  go-intl merge -base locales/en.json locales/fr.yaml       Fill missing target keys and write JSON to stdout
  go-intl check -base locales/en.json [locale-file ...]     Compile messages and report missing keys
`

// main reports command failures on stderr and sets a nonzero process exit code.
func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run dispatches CLI arguments and keeps user files read-only.
func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("expected a command; run go-intl help for usage")
	}
	if slices.Contains([]string{"help", "-h", "--help"}, args[0]) {
		_, err := io.WriteString(out, usage)
		return err
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var base, locale string
	switch args[0] {
	case "extract":
		flags.StringVar(&locale, "locale", "en", "locale to extract")
	case "merge", "check":
		flags.StringVar(&base, "base", "", "reference catalog file")
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], usage)
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err = io.WriteString(out, usage)
		}
		return err
	}
	if args[0] == "extract" {
		messages, err := extract(locale, flags.Args())
		if err != nil {
			return err
		}
		return writeJSON(out, messages)
	}
	if base == "" {
		return fmt.Errorf("%s requires -base", args[0])
	}
	reference, err := readCatalog(base)
	if err != nil {
		return err
	}
	if args[0] == "check" {
		return check(reference, base, flags.Args())
	}
	if flags.NArg() != 1 {
		return errors.New("merge requires exactly one target locale file")
	}
	target, err := readCatalog(flags.Arg(0))
	if err != nil {
		return err
	}
	for _, source := range []intl.Source{reference, target} {
		if _, err := intl.New(source.Locale(), source); err != nil {
			return err
		}
	}
	merged := reference.Messages()
	maps.Copy(merged, target.Messages())
	return writeJSON(out, merged)
}

// readCatalog applies the same format and filename rules as the library.
func readCatalog(filename string) (intl.Source, error) {
	name := filepath.Base(filename)
	if strings.ContainsAny(name, "*?[") {
		return intl.Source{}, fmt.Errorf("expected a single locale filename, got %q", filename)
	}
	sources, err := intl.LoadFS(os.DirFS(filepath.Dir(filename)), name)
	if err != nil {
		return intl.Source{}, fmt.Errorf("read %q: %w", filename, err)
	}
	return sources[0], nil
}

// check validates all source patterns and requires every target locale to cover the base keys.
func check(base intl.Source, basePath string, filenames []string) error {
	path, err := filepath.Abs(basePath)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{path: {}}
	sources := make([]intl.Source, 1, len(filenames)+1)
	sources[0] = base
	defaultTag := language.Make(base.Locale()).String()
	available := map[string]map[string]string{defaultTag: base.Messages()}
	for _, name := range filenames {
		path, err := filepath.Abs(name)
		if err != nil {
			return err
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		source, err := readCatalog(path)
		if err != nil {
			return err
		}
		sources = append(sources, source)
		tag := language.Make(source.Locale()).String()
		if available[tag] == nil {
			available[tag] = source.Messages()
		} else {
			maps.Copy(available[tag], source.Messages())
		}
	}
	if _, err := intl.New(base.Locale(), sources...); err != nil {
		return err
	}
	keys := slices.Sorted(maps.Keys(available[defaultTag]))
	missing := []string{}
	for _, tag := range slices.Sorted(maps.Keys(available)) {
		for _, key := range keys {
			if _, ok := available[tag][key]; !ok {
				missing = append(missing, fmt.Sprintf("%s: missing %q", tag, key))
			}
		}
	}
	if len(missing) > 0 {
		return errors.New(strings.Join(missing, "\n"))
	}
	return nil
}

// writeJSON emits flat keys in the lexical order guaranteed by encoding/json.
func writeJSON(out io.Writer, messages map[string]string) error {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(messages)
}
