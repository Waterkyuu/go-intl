---
title: Performance
description: Compare rendering costs and reproduce the measurements.
sidebar:
  order: 6
---

## Rendering benchmarks

These results use Go 1.27.1 on an Apple M3 Pro (darwin/arm64). The English catalog contains 1,024 static keys, a greeting, and a plural. Each result is the median of three runs, with a bound translator or warmed localizer and reused argument maps.

| Operation               | go-intl (ns/op) | go-i18n (ns/op) | Allocations/op (go-intl / go-i18n) |
| ----------------------- | --------------: | --------------: | ---------------------------------: |
| Repeated static key     |            12.9 |           116.8 |                              0 / 0 |
| Rotating static keys    |            15.8 |           125.8 |                              0 / 0 |
| Named greeting          |            66.2 |           288.2 |                              3 / 7 |
| English plural, count 7 |            81.2 |           336.5 |                              2 / 8 |

## Bind once per view

Reuse the locale and translator across renders. A fresh argument map on each greeting raised go-intl's measured cost from 66 to 180 ns/op. Keep mutable values local to their goroutine. The immutable catalog can be shared by concurrent readers.

## Reproduce the results

Run the benchmarks from the go-intl repository root.

```sh
go test . -run '^$' -bench 'Benchmark(Translate|LocalizedPlural|CatalogBuild|JSONStartup)' -benchmem -benchtime=350ms -count=3
```

The [full benchmark report](https://github.com/Waterkyuu/go-intl/blob/v0.1.0/PERFORMANCE.md) includes startup costs, parallel throughput, numeric formatting, the upstream comparison, and profiling commands.
