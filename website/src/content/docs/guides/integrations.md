---
title: Client and server integrations
description: Share translations across Bubble Tea, MyGo, NSGo, and HTTP.
sidebar:
  order: 4
---

## Terminal and desktop

Keep a bound translator in each model or window. When a user changes languages, bind a replacement from the same immutable catalog.

```go
t := catalog.Translations("es")
title := t("Inbox.title")
```

The [client examples](https://github.com/Waterkyuu/go-intl/tree/v0.1.0/examples/clients) share English and Spanish definitions. From a repository checkout, run the example for your framework. The example module requires Go 1.27.3 or later.

```sh
cd examples/clients
go run ./bubbletea -locale es
go run ./mygo -locale es
go run ./nsgo -locale es
```

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
