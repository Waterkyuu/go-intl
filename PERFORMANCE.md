# Translation benchmarks

Measurements use Go 1.27.1 on an Apple M3 Pro (darwin/arm64, 11 logical CPUs). The head-to-head rendering comparison uses 1,024 distinct static keys in one English namespace plus a greeting and a cardinal plural. The spread trace visits each key once per cycle. Static probes use separate string storage from the dictionary key. Both sides reuse argument maps in the head-to-head comparison. Measurements report medians of three runs. Parallel ns/op measures aggregate throughput across workers.

## Rendering

| Operation | go-intl (ns/op) | go-i18n (ns/op) | Allocations/op (go-intl / go-i18n) |
| --- | ---: | ---: | ---: |
| Static, one popular key | 12.9 | 116.8 | 0 / 0 |
| Static, spread keys | 15.8 | 125.8 | 0 / 0 |
| Named greeting | 66.2 | 288.2 | 3 / 7 |
| English plural with count 7 | 81.2 | 336.5 | 2 / 8 |
| Static, 11 parallel workers on one key | 2.11 | 16.6 | 0 / 0 |

The go-i18n baseline reuses a warmed localizer and supplies template arguments explicitly. Its templates use Go syntax. Both engines produced `Ready`, `Hello, Ada!` and `7 items` for these cases. The English count takes a short integer path. A grouped count of 12,345 took 317 ns in English, 324 ns in French and 350 ns in Arabic with go-intl. The French and Arabic benchmarks validate the exact localized digits and separators.

### Calling patterns

| Greeting rendered as `Hello, Ada!` | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| Bound root translator, prepared values | 66 | 32 | 3 |
| Bound root translator, fresh `Values{"name": "Ada"}` each call | 180 | 368 | 5 |
| Bound `screen` translator, prepared values | 67 | 32 | 3 |
| Bind `screen` on an existing locale each call, prepared values | 76 | 32 | 3 |
| Bind locale and `screen` each call, prepared values | 339 | 352 | 8 |

### Where to spend optimization time

1. Bind the locale at the view or request boundary. Reusing the scoped translator removed 80% of the fresh-bind greeting time and five allocations. Rebinding only the namespace added 9 ns on this single-level namespace. In the upstream hot-key profile, locale matching accounted for 64% of CPU samples, bounding the gain from eliminating that work alone at 2.8× on that workload.
2. For number-heavy screens, profile number formatting before changing key lookup. Each dictionary now owns a printer and its date formatter. Binding the printer at construction reduced grouped French plural calls from 413 ns and 560 B to 324 ns and 384 B. In the final CPU profile, number formatting accounted for 0.94 of 1.28 CPU seconds under translation, or 21% of all 4.38 process CPU seconds including runtime work. Completely eliminating those samples alone bounds the process CPU reduction at 21% with other costs held fixed.
3. In a view that already owns stable arguments, reusing them saved 336 B and two allocations per greeting. For changing arguments, keep mutable maps local to their rendering goroutine. Application-level gains depend on the mix of static, interpolated and formatted messages.

## Index prototype

A separate checkout of go-i18n contains the prototype. The base revision is `93061b1` and the local experiment commit is `4c403a8` on `waterkyuu/perf/translation-lookup`. Each lookup resolves a short string to an integer row index. The custom slot table uses FNV-1a, linear probing, 32-bit indices and 50% occupancy. Every measured lookup allocated zero bytes. [Upstream source](https://github.com/nicksnyder/go-i18n/tree/93061b1f8a26732db0079c195b3927291f3e72c0)

| Keys | Go map, hot / spread (ns) | Sorted slice, hot / spread (ns) | Slot table, hot / spread (ns) |
| ---: | ---: | ---: | ---: |
| 16 | 7.2 / 7.2 | 18.4 / 19.4 | 10.0 / 12.2 |
| 1,024 | 7.3 / 8.0 | 34.1 / 38.3 | 9.9 / 19.8 |
| 16,384 | 7.4 / 12.3 | 40.9 / 126.0 | 10.0 / 29.8 |

The map gave the fastest successful lookup at all three sizes. The sorted layout incurs increasingly expensive comparisons for spread reads. The slot table adds custom hashing and collision handling. These results apply to 17-byte ASCII identifiers on this CPU. Misses, longer keys, memory-constrained devices and different hash implementations can change the tradeoff.

## Construction and concurrency

| Startup operation | Median | Allocated per build |
| --- | ---: | ---: |
| Build from 16 Go keys + 2 patterns | 14.8 µs | 15.6 KiB |
| Build from 1,024 Go keys + 2 patterns | 142 µs | 220 KiB |
| Build from 16,384 Go keys + 2 patterns | 2.11 ms | 3.14 MiB |
| Load JSON and build 1,024 keys + 2 patterns | 362 µs | 585 KiB |

Startup bytes measure total allocations during the operation. The Go build rows start with a prepared source. The JSON row includes reading in-memory file contents, decoding, validation and building. Subtracting the matching Go build gives about 220 µs, or 61% of startup, associated with loading the source. The upper bound from eliminating that entire stage on this fixture is 2.55×. Sizing the program slice and initial index to the source length reduced 16,384-key build allocations from 6.36 MiB to 3.14 MiB with the same program representation.

| Static workload | 1 worker | 4 workers | 11 workers |
| --- | ---: | ---: | ---: |
| One popular key (ns/op) | 13.2 | 3.33 | 2.06 |
| Spread keys (ns/op) | 16.2 | 4.33 | 2.61 |

Each worker reads the same bound catalog. Additional readers add no shared writes. The 11-worker hot-key case delivered about 6.4× single-worker throughput. Tail latency requires request-level timing under the application workload. Translation data scales with defined messages per locale. Misses traverse the registered parent/default chain. Argument maps must remain read-only during each call, and rich-text callbacks own synchronization for their state.

## Reproduction

From the repository root, run all 13 README cases against both libraries with Go 1.26 or later:

```sh
go test -C benchmarks -run '^$' -bench '^BenchmarkTranslation$' -benchmem -cpu=1 -benchtime=250ms -count=6
```

Each round produces 26 results. The [benchmark source](./benchmarks/translation_benchmark_test.go) checks the translated text before timing. The comparison module uses the local go-intl checkout, pins go-i18n to revision `93061b1`, and builds both with the same dependencies. The release measurements used separate builds and rotated their order between rounds.

In the go-intl checkout, run

```sh
go test . -run '^$' -bench 'Benchmark(Translate|LocalizedPlural|CatalogBuild|JSONStartup)' -benchmem -benchtime=350ms -count=3
go test . -run '^$' -bench '^BenchmarkTranslate$/^parallel-' -benchmem -cpu=1,4,11 -benchtime=350ms -count=3
go test . -run '^$' -bench '^BenchmarkLocalizedPlural$/^fr$' -benchtime=3s -o /tmp/go-intl.test -cpuprofile=/tmp/go-intl.cpu
go tool pprof -top -cum /tmp/go-intl.cpu
```

In the sibling `go-intl-bench-upstream` checkout at the experiment commit, run

```sh
go test ./benchprobe -run '^$' -bench '^BenchmarkProbeLayout' -benchmem -benchtime=250ms -count=3
go test ./benchprobe -run '^$' -bench '^BenchmarkReference' -benchmem -benchtime=350ms -count=3
go test ./benchprobe -run '^$' -bench '^BenchmarkReference$/^hot$' -benchtime=2s -o /tmp/i18n-ref.test -cpuprofile=/tmp/i18n-ref.cpu
go tool pprof -top -cum /tmp/i18n-ref.cpu
```
