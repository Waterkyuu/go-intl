---
title: Messages and formatting
description: Write placeholders, conditional messages, and locale-aware plurals.
sidebar:
  order: 3
---

## Interpolate values

Define `Inbox.hello` as `Hello, {name}!` and provide its arguments when rendering.

```go
t := catalog.Translations("en")
greeting := t("Inbox.hello", intl.Values{"name": "Ada"})
// Hello, Ada!
```

## Select a branch

Use `select` for booleans or enum values. Every selection needs an `other` branch. Only the selected branch needs argument values.

```text
{role, select, admin {Manage inbox} other {View inbox}}
```

## Render plurals

Use an exact selector such as `=0` for a special count, and CLDR plural categories for grammatical forms. The `#` marker renders the localized count.

```text
{n, plural, =0 {Inbox empty} one {# unread message} other {# unread messages}}
```

For ordinals use `selectordinal`. For guests other than the recipient, use `offset:1`. Exact selectors compare the original count, while the plural category and `#` use the adjusted count.

## Format numbers and dates

Patterns accept `{price, number}`, `{share, number, percent}`, and `{when, date, long}`. Pass a Go `time.Time` for dates and times. The bound locale also exposes number, currency, list, and relative-time helpers.

## Render rich text

For a pattern such as `Read <strong>{title}</strong>`, supply a function for the tag. Its return value becomes part of the rendered string.

```go
text := t("Inbox.link", intl.Values{
    "title": "Inbox",
    "strong": func(s string) string { return "**" + s + "**" },
})
```

Apply context-appropriate escaping when inserting translations into HTML.
