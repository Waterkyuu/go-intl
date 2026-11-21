// Command mygo runs the translated counter in a native desktop window.
package main

import (
	"flag"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	intl "github.com/waterkyuu/go-intl"
	"github.com/waterkyuu/go-intl/examples/clients/internal/counter"
)

// window stores the state of one localized native view.
type window struct {
	// t renders the Counter namespace in the selected language.
	t intl.Translator
	// count is owned by the window's event loop.
	count int
}

// view builds native widgets for the translated count and its action.
func (w *window) view(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(12).Children(func() {
		ui.Text(c, w.t("items", intl.Values{"n": w.count}))
		if ui.PrimaryButton(c, w.t("add")).Clicked() {
			w.count++
		}
	})
}

// main binds the requested language before creating the window.
func main() {
	locale := flag.String("locale", "en", "Display language")
	flag.Parse()
	t, err := counter.Messages(*locale)
	if err != nil {
		log.Fatal(err)
	}
	w := &window{t: t}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title: t("title"), Width: 360, Height: 240,
			Content: ui.View(w.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
