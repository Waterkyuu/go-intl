---
title: Catalog CLI
description: Use the go-intl CLI to extract Go messages, check translations in CI, and merge missing catalog keys.
sidebar:
  order: 5
---

## Install the CLI

```sh
go install github.com/waterkyuu/go-intl/cmd/go-intl@v0.1.0
```

## Extract messages

From a repository checkout, extract the example client's English and Spanish definitions. Extraction reads literal language tags and string map literals passed to `intl.Define`.

```sh
go-intl extract -locale en ./examples/clients/internal/counter > en.json
go-intl extract -locale es ./examples/clients/internal/counter > es.json
```

## Check and merge

Use `check` in CI to validate message patterns and catch missing translations. Use `merge` to fill missing keys with source text while keeping existing translations. Write to a separate output file so the shell preserves the input.

```sh
go-intl check -base en.json es.json
go-intl merge -base en.json es.json > es.merged.json
```
