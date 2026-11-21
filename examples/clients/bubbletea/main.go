// Command bubbletea runs the translated counter in a terminal.
package main

import (
	"flag"
	"log"

	tea "charm.land/bubbletea/v2"
	intl "github.com/waterkyuu/go-intl"
	"github.com/waterkyuu/go-intl/examples/clients/internal/counter"
)

// model binds one terminal session to its locale.
type model struct {
	// t renders the Counter namespace in the selected language.
	t intl.Translator
	// count is the current number of items.
	count int
}

// Init starts the model without an initial command.
func (m model) Init() tea.Cmd { return nil }

// Update increments the counter on space and exits on q or Ctrl+C.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "space":
			m.count++
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View formats the current count using the locale's plural forms.
func (m model) View() tea.View {
	count := m.t("items", intl.Values{"n": m.count})
	return tea.NewView(count + "\n" + m.t("help"))
}

// main binds the requested language before entering the Bubble Tea event loop.
func main() {
	locale := flag.String("locale", "en", "Display language")
	flag.Parse()
	t, err := counter.Messages(*locale)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := tea.NewProgram(model{t: t}).Run(); err != nil {
		log.Fatal(err)
	}
}
