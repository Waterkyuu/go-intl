# Go development guidelines

## Package design

- Read the surrounding package, its callers, and its tests before changing behavior. Preserve compatibility unless the task requires an API change.
- Place code in the package that owns the behavior. Add an abstraction when a current caller needs it.
- Pass dependencies explicitly through constructors. Define small interfaces in the consuming package when callers need substitution. Prefer concrete types elsewhere.
- Prefer the standard library and existing dependencies. Keep mutable state with the component that owns it.

## Go conventions

- Follow the Uber Go Style Guide and the conventions of the surrounding package. Use snake_case for multiword filenames, such as tool_registry.go.
- Use `any` for an unconstrained type. Use struct embedding when the embedded type's behavior is part of the containing type's contract.
- Group related constants. Use typed constants with `iota` for sequential values when the numeric values can follow declaration order.
- Give each function one responsibility. Use early returns to keep the main path easy to follow.
- Add context when returning an error across a package or operation boundary. Use `%w` when callers should be able to inspect the cause with `errors.Is` or `errors.As`.

## Declaration comments

- Write comments in English immediately above every named function, method, struct type, struct field (including embedded fields), and constant (including each entry in a group). This applies to exported and unexported declarations.
- Start exported declaration comments with the declaration name. Describe the contract, ownership, units, or invariants that a caller or maintainer needs to know.
- Comment an anonymous function when its lifecycle or side effects need explanation. For a code review fix, document the triggering case and why the fix handles it.

## Testing and checks

- Keep `*_test.go` files beside the implementation. Add or update tests for changes in observable behavior.
- Use table-driven tests when cases share setup and assertions. Cover relevant failures, cancellation, and boundary cases.
- Keep tests deterministic. Inject external services and clocks when a test needs to control them.
- Format changed Go files with `gofmt`. In a Go module, run focused package tests while iterating, then run `go vet ./...` and `go test ./...` before committing Go changes.
