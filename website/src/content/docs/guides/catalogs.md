---
title: Translation files
description: Load Go, JSON, TOML, YAML, and CSV into a shared catalog.
sidebar:
  order: 2
---

## Choose a source

Put native definitions in ordinary Go files with `intl.Define`. For data files, use a BCP 47 language tag as the filename stem, such as `fr-CA.toml`. Each of these sources defines the same message.

| Source | Definition                                                     |
| ------ | -------------------------------------------------------------- |
| Go     | `intl.Define("en", map[string]string{"Inbox.title": "Inbox"})` |
| JSON   | `{"Inbox": {"title": "Inbox"}}`                                |
| TOML   | `Inbox.title = "Inbox"`                                        |
| YAML   | `Inbox: {title: Inbox}`                                        |
| CSV    | Header `key,message` followed by the row `Inbox.title,Inbox`   |

## Embed translations

Place translation files in a `locales` directory beside the Go file. Compile the catalog once at application startup.

```go
package translations

import (
    "embed"

    "github.com/waterkyuu/go-intl"
)

//go:embed locales/*
var files embed.FS

func Load() (*intl.Catalog, error) {
    sources, err := intl.LoadFS(files, "locales/*")
    if err != nil {
        return nil, err
    }
    return intl.New("en", sources...)
}
```

## Validate your definitions

Multiple files for a locale can contribute disjoint keys. Empty messages are valid. Overlapping keys and invalid message patterns cause catalog construction to fail. CSV supports quoted commas and multiline messages.
