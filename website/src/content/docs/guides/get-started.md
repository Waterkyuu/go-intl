---
title: Get started
description: Install go-intl, render your first Go translation, and configure fallback languages and optional namespaces.
sidebar:
  order: 1
---

## Install go-intl

In a Go module using Go 1.25 or later, add the library to your dependencies.

```sh
go get github.com/waterkyuu/go-intl
```

## Render your first message

Save this as `main.go` and run `go run .`. The module path ends in `go-intl` and its package name is `intl`.

```go
package main

import (
    "fmt"
    "log"

    "github.com/waterkyuu/go-intl"
)

func main() {
    catalog, err := intl.New("en",
        intl.Define("en", map[string]string{
            "Inbox.hello": "Hello, {name}!",
            "Inbox.title": "Inbox",
        }),
        intl.Define("fr", map[string]string{
            "Inbox.hello": "Bonjour, {name} !",
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    t := catalog.Translations("fr")
    fmt.Println(t("Inbox.hello", intl.Values{"name": "Ada"}))
    fmt.Println(t("Inbox.title"))
}
```

The program prints `Bonjour, Ada !` followed by `Inbox`. Bind the translator once per window, model, or request and pass it to the code that renders text.

## Set the fallback language

The first argument to `intl.New` sets the fallback language. Here the French catalog has no `Inbox.title`, so the English translation supplies it. To use a different default, pass its language tag and include a source for that language.

## Bind a namespace

Use full dotted keys by default. When a view uses one namespace repeatedly, bind it once.

```go
inbox := catalog.Translations("fr", "Inbox")
fmt.Println(inbox("title"))
```
