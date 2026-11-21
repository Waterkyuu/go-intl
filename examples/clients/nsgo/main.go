// Command nsgo displays a localized counter in a native window.
package main

import (
	"flag"
	"log"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"
	"github.com/waterkyuu/go-intl"
	"github.com/waterkyuu/go-intl/examples/clients/internal/counter"
)

// counterView binds the translator to the window and recomputes the count label when its signal changes.
func counterView(t intl.Translator) func(*core.Scope) core.Node {
	return func(cx *core.Scope) core.Node {
		cx.SetTitle(t("title"))
		count := core.NewSignal(cx, 0)
		return core.View{
			Style: core.Style{Padding: core.All(24), Gap: 12},
			Children: []core.Node{
				core.Text{TestID: "count", ContentFunc: func() string {
					return t("items", intl.Values{"n": count.Get()})
				}},
				ui.Button{Text: t("add"), OnPress: func() { count.Set(count.Get() + 1) }},
			},
		}
	}
}

// main selects the display language before starting the native event loop.
func main() {
	locale := flag.String("locale", "en", "Display language")
	flag.Parse()
	t, err := counter.Messages(*locale)
	if err != nil {
		log.Fatal(err)
	}
	nsgo.Run(counterView(t), nsgo.WindowSize(360, 240))
}
