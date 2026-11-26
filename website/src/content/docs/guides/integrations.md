---
title: Client and server integrations
description: Share Go translations across Bubble Tea, MyGo, NSGo, Wails backends, and HTTP handlers using Accept-Language.
sidebar:
  order: 4
---

## Terminal and desktop

Keep a bound translator in each model or window. When a user changes languages, bind a replacement from the same immutable catalog.

```go
t := catalog.Translations("es")
title := t("Inbox.title")
```

The [client examples](https://github.com/Waterkyuu/go-intl/tree/main/examples/clients) share English and Spanish definitions. From a repository checkout, run the example for your framework. The example module requires Go 1.27.3 or later.

```sh
cd examples/clients
go run ./bubbletea -locale es
go run ./mygo -locale es
go run ./nsgo -locale es
```

## Wails local Go backend

Keep an `intl.Translator` on the Go backend and return translated strings to the web frontend. For example, in a Wails v2 app, add a `t intl.Translator` field to the existing `App` struct. Initialize it with `catalog.Translations("fr")` when constructing the instance passed to `options.App.Bind`.

```go
// InboxTitle returns the localized title for the inbox view.
func (a *App) InboxTitle() string {
    return a.t("Inbox.title")
}
```

In a frontend module under `frontend/src/`, call the generated binding. With the catalog from the quick start, the French translator returns `Inbox` through English fallback.

```js
import { InboxTitle } from "../wailsjs/go/main/App.js";

const title = await InboxTitle();
```

Wails generates the binding when you run `wails dev`. [Wails v2 method binding](https://v2.wails.io/docs/v2.15.0/howdoesitwork/#method-binding)

## HTTP language negotiation

Match the request's language preferences, return the selected tag, and tell HTTP caches that the response varies by language.

```go
func Inbox(catalog *intl.Catalog) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        locale := catalog.Match(r.Header.Get("Accept-Language"))
        w.Header().Set("Content-Language", locale.Tag().String())
        w.Header().Add("Vary", "Accept-Language")
        t := locale.Translations()
        _, _ = io.WriteString(w, t("Inbox.title"))
    }
}
```

## Handle a missing message

The callable translator returns the fully qualified key on failure. Use `locale.Format` when you need an error, `locale.Has` to check the fallback chain, or `locale.Raw` to inspect the original pattern.
