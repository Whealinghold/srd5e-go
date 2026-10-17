# srd5e Go Rules

Normative. MUST / MUST NOT / SHOULD are binding on all code and review. These are
the Go conventions of this repository; the structure they serve is in
[architecture.md](architecture.md), the failure model in
[errors-and-results.md](../specs/errors-and-results.md), testing in
[testing.md](testing.md).

Baseline: Go as declared in `go.mod` (`go 1.24` unless a newer feature is truly
needed), standard library only, `gofmt` + `goimports` ordering, `go vet` and
`golangci-lint` (committed `.golangci.yml`) clean.

---

## 1. Naming

- Packages: short, lowercase, no underscores, no plurals for the sake of it
  (`d20`, not `d20_test`). Test-support packages are `<pkg>test`; that suffix is
  reserved for them, so a context package name MUST NOT end in "test".
- **No stutter.** An exported identifier MUST NOT begin with its package name
  plus more words: `dice.Expression`, not `dice.DiceExpression`. An identifier
  equal to the package name is fine (`modifier.Modifier`, like `rand.Rand`).
- Initialisms keep one case: `ID`, `URL`, `JSON`, `DC`, `AC`, `HP` (`CharacterID`,
  `ParseURL`), never `Id` or `Json`.
- No `Get` prefix on accessors: `s.Value()`, not `s.GetValue()`. Boolean
  accessors read as predicates (`IsZero`, `Valid`).
- Errors: sentinels `ErrX`; error types `XError`. Constructors `NewX`, parsers
  `ParseX`, panicking helpers `MustX`.
- Interfaces: one-method interfaces are named for the method plus `er` where
  natural (`Roller`). No `I` prefix, no `Impl` suffix.
- Files: `snake_case.go`, one concept per file (`seeded_roller.go`,
  `resolver.go`). A use case lives in a file named for it (`resolver.go`); the `_usecase.go` suffix is not required.
- SRD terminology wins over Go taste for domain nouns (architecture.md §3).

## 2. Documentation

- Every exported identifier has a doc comment that starts with its name and is a
  full sentence ending in a period (`godot`).
- Every package has a `doc.go` with a package comment (`// Package dice ...`).
  The root `doc.go` is `package srd5e` and holds documentation only.
- A context's `doc.go` lists inputs, outputs and handoff types.
- Every fallible exported function states what it returns on each failure class:

  ```go
  // ParseExpression parses a dice expression such as "2d8+2".
  //
  // It returns an error wrapping srderr.ErrInvalidInput (as *InvalidError)
  // when s is not a valid expression. It never panics.
  func ParseExpression(s string) (Expression, error)
  ```

- Document the zero value of every exported type ("The zero Score is invalid").
- Document concurrency: "Safe for concurrent use" or "Not safe for concurrent
  use". Default for immutable value types is safe.
- Runnable documentation is `Example*` functions in `_test.go` files.

## 3. Error handling idioms

The failure classes are fixed by architecture.md §6; this section is the Go
idiom.

- Expected outcomes are values, not errors (results, enums, sealed interfaces).
- Validation: `NewX(...) (X, error)` and `ParseX(string) (X, error)`. The error
  is a typed value or wraps a sentinel:

  ```go
  // Each package uses one InvalidError with a typed Reason enum and a Detail
  // (dice.InvalidError, ability.InvalidError, modifier.InvalidError,
  // d20.InvalidError). Its Is method matches another *InvalidError by
  // Reason, so errors.Is(err, &ability.InvalidError{Reason: r}) branches on it.
  type InvalidError struct {
      Reason Reason // typed enum with String()
      Detail string // human-readable specifics
  }

  func (e *InvalidError) Error() string { return "ability: invalid score: " + e.Reason.String() + ": " + e.Detail }
  func (e *InvalidError) Unwrap() error { return srderr.ErrInvalidInput }
  ```

  Every validation error also satisfies `errors.Is(err, srderr.ErrInvalidInput)`
  (via `Unwrap() []error` or by wrapping both).
- Decoders report all problems: `return zero, errors.Join(problems...)` with typed
  problems; a non-nil error means at least one problem.
- Wrap with context using `%w` and `errors.Is` / `errors.As`; never compare error
  strings, never use `==` on non-sentinel errors, never type-assert on `error`.
- Error strings: lowercase, no trailing punctuation, no "failed to" prefix chains.
- Port failures are wrapped once, by the caller (the use case), with
  `srderr.WrapPort(port, op, err)`; nothing wraps an already wrapped failure of
  the same port.
- `MustX` is the only place a validating constructor panics, and the panic value
  is the `error` from `NewX`:

  ```go
  func MustScore(v int) Score {
      s, err := NewScore(v)
      if err != nil {
          panic(err)
      }
      return s
  }
  ```

- `panic` is also allowed in an unreachable `default:` branch, with an error value
  (`panic(fmt.Errorf("unreachable: %T", r))`). A string literal panic is
  forbidden. Exported APIs taking untrusted input never panic.
- `recover` is used only by `srderr.Guard` / `GuardValue` and test support.
- Never ignore an error (`errcheck`); `_ =` needs a comment saying why it is safe.

## 4. Context rules

- `context.Context` is the first parameter, named `ctx`, of anything that may do
  I/O, block or emit spans: content sources, use-case entry points that call
  telemetry.
- Pure CPU domain functions and `dice.Roller` (synchronous, in-memory) take no
  `ctx`.
- Never store a `ctx` in a struct. Never pass `nil` (use `context.Background()`
  in `main` and tests only; libraries never create a root context).
- Cancellation errors pass through `errors.Is(err, context.Canceled)`; never turn
  them into results.
- `ctx` is not a bag for parameters: no `context.WithValue` for domain data.

## 5. Zero values, constructors and options

- Make the zero value useful (`telemetry.Noop{}`, an empty `Result`) or explicitly
  invalid and documented, with `IsZero()` / `Valid()` where a caller needs to
  tell.
- Value objects: unexported fields, validating constructor, value receivers, no
  setters, methods return new values. Do not use pointer receivers on small
  value types; do not return pointers to small structs.
- Keep comparable types comparable. If a type holds a slice, either it is not
  compared with `==` and provides `Equal`, or it holds an array/string form.
- Constructors return concrete types, not interfaces.
- Functional options (`type Option func(*config)`, `WithX`) only where a
  constructor would otherwise take more than about three optional parameters.
  Otherwise use plain parameters or a small config struct. Options default to
  safe values (`telemetry.Noop`).
- No package-level mutable state, no `init()` side effects, no package-level
  randomness or time. Time and randomness come from injected ports.
- Package-level `var` is limited to `var _ I = (*T)(nil)` assertions and `Err*`
  sentinels. Use `const` and functions for tables.

## 6. Generics policy

- Use generics only where they remove real duplication in an implementation
  (`srderr.GuardValue[T]`, a typed-set helper). Never because a type could be
  generic.
- **Never in a public port.** Ports (`dice.Roller`, content source, telemetry
  interfaces) are non-generic. Generic parameters on exported domain types
  require a spec that justifies them.
- No type-parameter constraints beyond `any`, `comparable` and stdlib
  `cmp.Ordered` without a recorded reason.

## 7. Interfaces

- Small: 1 to 3 methods. Defined where consumed ("accept interfaces, return
  structs"); the consumer declares the minimal interface it needs.
- A port shared by several packages lives next to the type defining its vocabulary
  (`dice.Roller`, as `io.Reader` lives in `io`).
- Closed variant sets: interface with an unexported marker method, variants as
  value-receiver struct types, doc comment listing all variants, consumed by a
  type switch. (`//sumtype:decl` plus the `gochecksumtype` linter is planned, not
  yet applied.)
- Adapters carry `var _ Port = (*Adapter)(nil)`.
- Do not define an interface only so a test can mock it; use a real in-memory
  double from the `*test` package.
- No `interface{}` / `any` in exported signatures of domain code.
- No reflection (`reflect`) in domain code.

## 8. File and package organisation

- One package per bounded context; domain, use cases and ports in the same
  package, separated by files (`<concept>.go`, a file per use case, `ports.go` when a context has more than one port; `d20` declares its `Roller` in `resolver.go`).
- Unexported helpers by default; export only what the published language needs.
- External test packages (`package x_test`) for public behaviour; internal tests
  (`package x`) only for unexported edge logic.
- Fixtures live in `testdata/` (ignored by the toolchain); golden files are
  updated by an explicit `-update` flag defined in the test, never implicitly.
- Import groups: standard library, then module packages; `goimports` ordering.
  No dot imports, no import aliases except to resolve a clash.

## 9. Compatibility promise and deprecation

- The module follows semantic versioning with git tags and stays on the `v0` /
  `v1` import path. Before `v1.0.0` a breaking change bumps the minor version.
- The exported identifiers of every non-`internal` package are the compatibility
  surface. Breaking: removal or rename, signature change, adding a method to an
  exported interface, adding a variant to a closed variant set, changing a result
  for the same input (unless fixing a bug against SRD 5.2.1), raising the minimum
  Go version, renaming a telemetry signal.
- The minimum Go version is the `go` line in `go.mod`; the module is tested
  against it and the latest Go release (CI runs both). Raising it is announced in `CHANGELOG.md`
  and treated as a minor-version change.
- Deprecation: add a `// Deprecated: use X instead.` paragraph to the doc comment,
  list it under **Deprecated** in `CHANGELOG.md`, keep it for at least one
  release, then remove under **Removed** with **BREAKING**.
- Exported-field structs MUST be constructed with keyed literals (each such
  type's doc comment says so). Adding a field is non-breaking only if a
  comparable type stays comparable; adding a slice, map or func field to a
  comparable type is breaking and needs a minor bump before v1 and a BREAKING
  changelog entry.
- Tightening a contract suite in a *test package (RunRollerContract,
  RunLoggerContract, ...) is a minor-version change listed under Changed in
  CHANGELOG.md, never a patch.
- New exported methods on concrete types and new exported identifiers are
  non-breaking, except new struct fields under the rule above.

## 10. Forbidden

- `init()` side effects, package-level mutable state, singletons and service
  locators.
- Reflection (`reflect`), `unsafe`, `cgo` in domain code.
- Third-party runtime dependencies (`go.mod` has no `require`).
- Panics for expected conditions, or panics across a package boundary for them.
- Goroutines, channels and `sync` in domain and use-case code.
- Reading the clock or randomness directly (`time.Now`, `math/rand` globals).
- Printing or logging directly (`fmt.Print*`, `log`); use `telemetry.Logger`.
- Display text in results and errors; branching on error strings.
- Exporting a type only because a single consumer asked for it.

---

## Checklist before submitting

- [ ] Names follow §1: no stutter, initialisms, no `Get`, snake_case files.
- [ ] Doc comments on all exported identifiers; `doc.go` per package.
- [ ] Failure behaviour documented on each fallible function.
- [ ] `ctx` first where I/O or spans occur; never stored.
- [ ] Zero value documented; constructors validate and return concrete types.
- [ ] No generics in ports; generics justified where used.
- [ ] No forbidden construct from §10.
- [ ] `gofmt`, `go vet`, `golangci-lint run`, `go test -race -shuffle=on -cover ./...`
      pass.

---

## Read receipt

RULE-TAG: `go-a6d9ed0dc9f2`

This tag proves you read this document in this session. Quote it verbatim as
required by `AGENTS.md`. Do not copy it anywhere else. Whoever edits this
document MUST replace the tag with a fresh random value.
