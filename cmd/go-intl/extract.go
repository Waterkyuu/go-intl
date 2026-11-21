package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	intl "github.com/waterkyuu/go-intl"
	"golang.org/x/text/language"
)

// intlImport is the package whose Define calls supply extractable messages.
const intlImport = "github.com/waterkyuu/go-intl"

// extract reads literal Define calls for a single language without running the Go package.
func extract(locale string, paths []string) (map[string]string, error) {
	tag, err := language.Parse(locale)
	if err != nil {
		return nil, fmt.Errorf("parse locale %q: %w", locale, err)
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	messages := make(map[string]string)
	seen := make(map[string]struct{})
	found := false
	for _, input := range paths {
		root, err := filepath.Abs(input)
		if err != nil {
			return nil, err
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if path != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "vendor") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" {
				if path == root {
					return fmt.Errorf("expected a Go file or directory, got %q", path)
				}
				return nil
			}
			if path != root && strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if _, exists := seen[path]; exists {
				return nil
			}
			seen[path] = struct{}{}
			matched, err := extractFile(path, tag, messages)
			found = found || matched
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	if !found {
		return nil, fmt.Errorf("no literal intl.Define calls for locale %q", locale)
	}
	if _, err := intl.New(locale, intl.Define(locale, messages)); err != nil {
		return nil, err
	}
	return messages, nil
}

// extractFile resolves package import aliases and attributes extraction failures to source lines.
func extractFile(path string, locale language.Tag, messages map[string]string) (bool, error) {
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, path, nil, 0)
	if err != nil {
		return false, err
	}
	binding := ""
	for _, spec := range file.Imports {
		pkg, err := strconv.Unquote(spec.Path.Value)
		if err != nil || pkg != intlImport {
			continue
		}
		binding = "intl"
		if spec.Name != nil {
			binding = spec.Name.Name
		}
		break
	}
	if binding == "" || binding == "_" {
		return false, nil
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		if err != nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 || !isDefinition(call.Fun, binding) {
			return true
		}
		name, literal := literalString(call.Args[0])
		if !literal {
			return true
		}
		tag, parseErr := language.Parse(name)
		if parseErr != nil {
			err = fmt.Errorf("%s: invalid source locale %q: %w", positions.Position(call.Pos()), name, parseErr)
			return false
		}
		if tag != locale {
			return false
		}
		found = true
		if parseErr = extractMap(call.Args[1], messages); parseErr != nil {
			err = fmt.Errorf("%s: %w", positions.Position(call.Args[1].Pos()), parseErr)
		}
		return false
	})
	return found, err
}

// isDefinition matches both qualified and dot-imported calls while respecting local shadowing.
func isDefinition(expr ast.Expr, binding string) bool {
	if binding == "." {
		id, ok := expr.(*ast.Ident)
		return ok && id.Name == "Define" && id.Obj == nil
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Define" {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == binding && id.Obj == nil
}

// extractMap requires a concrete map literal so every emitted key and value is known.
func extractMap(expr ast.Expr, messages map[string]string) error {
	obj, ok := expr.(*ast.CompositeLit)
	if !ok {
		return errors.New("expected a map[string]string literal")
	}
	mapType, ok := obj.Type.(*ast.MapType)
	if !ok || !isStringType(mapType.Key) || !isStringType(mapType.Value) {
		return errors.New("expected a map[string]string literal")
	}
	for _, item := range obj.Elts {
		pair, ok := item.(*ast.KeyValueExpr)
		if !ok {
			return errors.New("expected a message key and value")
		}
		key, keyOK := literalString(pair.Key)
		value, valueOK := literalString(pair.Value)
		if !keyOK || !valueOK {
			return errors.New("message keys and values must be string literals")
		}
		if _, duplicate := messages[key]; duplicate {
			return fmt.Errorf("duplicate message key %q", key)
		}
		messages[key] = value
	}
	return nil
}

// isStringType recognizes the built-in string type used in native catalog maps.
func isStringType(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "string" && id.Obj == nil
}

// literalString decodes Go raw strings and double-quoted string literals.
func literalString(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	return value, err == nil
}
