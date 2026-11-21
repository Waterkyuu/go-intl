// Package counter holds the same native Go translations for the terminal and desktop.
package counter

import intl "github.com/waterkyuu/go-intl"

// Messages selects the display language and scopes all client lookups to Counter.
func Messages(locale string) (intl.Translator, error) {
	catalog, err := intl.New(
		"en",
		intl.Define("en", map[string]string{
			"Counter.title": "Counter",
			"Counter.items": "{n, plural, one {# item} other {# items}}",
			"Counter.add":   "Add one",
			"Counter.help":  "Press space to add one or q to quit.",
		}),
		intl.Define("es", map[string]string{
			"Counter.title": "Contador",
			"Counter.items": "{n, plural, one {# artículo} other {# artículos}}",
			"Counter.add":   "Añadir uno",
			"Counter.help":  "Pulsa espacio para añadir uno o q para salir.",
		}),
	)
	if err != nil {
		return nil, err
	}
	return catalog.Translations(locale, "Counter"), nil
}
