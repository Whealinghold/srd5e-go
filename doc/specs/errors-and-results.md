# Errors and results

| | |
|---|---|
| Status | partially implemented |
| Context | `package` |
| Roadmap phase | 0 |
| Depends on | none |

## 1. Purpose and scope

Defines how every part of srd5e reports that something did not go to plan:

- the `srderr` package: `PortError`, `WrapPort`, `Guard`, `GuardValue`,
  `ErrorType` and the shared sentinels. `PortError` is produced only when a port
  or its backend fails, including a port that breaks its contract (FR-036);
- the conventions every closed result type follows (expected outcomes, absence,
  rejection, untrusted input that fails validation);
- the split between returned results, returned errors and panics;
- the guards use cases use to wrap port failures without hiding programmer errors;
- the `error.type` value registry (FR-034, FR-005; decisions FR-041), the naming
  rules for results and variants (FR-041, FR-042; decisions FR-017, FR-018) and
  how a returned failure is classified in telemetry (FR-043).

This spec is the contract every other spec's result types, constructors and
ports follow. It defines no game rule. It builds on nothing but the binding
decisions. Other files are cited, not depended on: `dice-and-rolls.md` and
`content-source-port.md` depend on this one and own the port names;
`observability-ports.md` is cited for the `error.type` key constant;
`release-0-1-0.md` and `content-schema-and-decoder.md` for the example-code scan
(FR-045) and the decoder test of FR-024.

**Out of scope:**

- Any concrete result type of a context (content lookup, `d20` outcomes,
  decoder problems, every use-case result); contexts define them following 4.3.
- Any shared result variant or failure-detail type. `MissingContent{Kind, ID}` is
  owned by `content` (decisions FR-018); FR-042 states the convention.
- Any port name constant and its operations: `dice.Port` is owned by
  [dice-and-rolls.md](dice-and-rolls.md), the content port's by
  [content-source-port.md](content-source-port.md).
- A shared problem type (each context owns its own; OQ-2) and a generic
  `Result[T, E]` (OQ-1).
- SRD content. Nothing here comes from SRD 5.2.1.

## 2. Source

n/a (package design). Derived from, and extending:

- `doc/rules/architecture.md` §4 (port failure contract), §5 (decoders return
  every problem), §6 (failure model), §7 (public API changes).
- `doc/rules/go.md` §3 (error idioms), §4 (context rules).
- `doc/rules/observability.md` §5 (error counter by error type; shared
  attribute values), §7 ("Errors: who records what").
- `doc/rules/testing.md` §4 ("Expected outcomes vs errors"), §5, §7.
- `doc/specs/shared-types-and-ownership.md` FR-017, FR-018, FR-038, FR-039,
  FR-041, FR-051, FR-055 (binding; wins on any conflict).
- `doc/roadmap.md`, Phase 0 (the `srderr` package).

## 3. Vocabulary

Every row is a package term, not an SRD term.

| Term | Go form | Notes |
|---|---|---|
| port failure | `*srderr.PortError{Port, Op, Err}` | Wraps the underlying error; `Unwrap()` returns `Err`. Matched with `errors.As`. |
| port name | `string` constant `Port` in the port's package | Lowercase snake case, unique across the module (`dice`, `content`). |
| sentinel | `var Err... = errors.New(...)` | Matched with `errors.Is` (section 5). |
| closed result | Interface with an unexported marker method, one struct type per variant (or a typed enum) | Returns an expected outcome, absence or rejection. A use case's is `<Op>Result`; variants carry the operation prefix (FR-041). |
| failure detail | Value object held as a field | Detail shared by several results (`MissingContent`), never a shared variant (FR-042). |
| problem | Typed value, defined per context | One entry in a decoder error built with `errors.Join` (architecture §5). |
| rejection reason | Typed value (enum or error type) | The single typed reason a scalar `NewX` / `ParseX` returns (FR-035). |
| `WrapPort`, `Guard`, `GuardValue` | functions in `srderr` | Used by use cases around port calls. |
| `PanicError` | `*srderr.PanicError{Value, Stack}` | The error a recovered port panic becomes (FR-009). |
| `ContractViolation`, `ContractError` | func and `*srderr.ContractError{Detail}` | Builds the `PortError` for a port that returned contract-breaking data (FR-014, FR-036). |
| `Quote` | `srderr.Quote(s string) string` | Go-quotes untrusted text for error details, cut at 32 bytes on a rune boundary. |

## 4. Requirements

Source for every row is `n/a (package design)` plus the cited rule section. Path
labels follow `doc/rules/testing.md` §4, plus "architecture check" for rows
enforced by `internal/archtest` (checks A1 to A13 in architecture.md §9). Convention
rows (FR-018 onward) apply to every type in the package; each context spec cites
and tests them for its own types.

### 4.1 The `PortError` type

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `srderr.PortError` MUST be a concrete struct implementing `error` and `Unwrap() error`, used as `*PortError`, so a consumer reads it with `errors.As(err, &pe)`. Its shape is in section 5. | architecture.md §6 | happy |
| FR-002 | `PortError` has exported fields `Port`, `Op` (strings) and `Err` (error), set by `WrapPort`. `Unwrap` MUST return `Err`, the same instance and unchanged, so `errors.Is` / `errors.As` reach the original cause. | architecture.md §4, §6 | happy |
| FR-003 | A `PortError` built without `Err` (nil) MUST still work: `Error()` does not panic and `Unwrap()` returns nil. `WrapPort` called with a nil `err` returns nil, never a `PortError`. | architecture.md §4 | unhappy (empty and minimal) |
| FR-004 | Every port failure surfaced by a use case or adapter helper, including `dice` and `content`, MUST be a `*PortError` carrying that port's `Port`. This spec defines no per-port error type; a port refines a failure with a sentinel in the chain (FR-055) or its own typed cause. | architecture.md §4 | happy |
| FR-005 | `srderr.ErrorType(err error) string` MUST return `<Port>_error` for the outermost `*PortError` found by `errors.As` (`dice_roller_error`, `content_error`), and "" when none is found. The value is low-cardinality, stable, one per port (not per reason or instance) and a public telemetry contract (decisions FR-041; observability.md §5). | observability.md §5, §7 | happy |
| FR-006 | Port names MUST be unique across the module and match `^[a-z][a-z0-9_]*$`. Check A8 scans every `const Port` and fails on a duplicate or a non-matching value. Table membership in section 7 is checked in review only (architecture.md A8 checks pattern and uniqueness). | package design | architecture check (pattern, uniqueness); review (table) |
| FR-007 | A `PortError`, a sentinel, and every result or rejection type MUST carry data, not display text: no exported field or method named `Message`, `Description`, `Text` or `DisplayText`. `Error()` is for logs and debugging only; no code or test may branch on it. Check A8 enforces the names. | architecture.md §6 | architecture check |
| FR-049 | The set of port names is closed by documentation (the section 7 table) and check A8, not by the type system: `Port` is a string, so a consumer adapter may wrap with its own port name. A `switch` on `ErrorType` or `Port` needs a default branch. | architecture.md §4, §6 | happy |
| FR-050 | `PortError` fields are exported for reading. Consumers MUST construct one only through `WrapPort`, `ContractViolation` or `Guard`; direct construction is allowed in tests only. Go cannot prevent it; review and check A6 (use only in use-case files) confine production use. | architecture.md §6 | architecture check |

### 4.2 Guard helpers for use cases

Signatures are frozen (OQ-7):

```go
func WrapPort(port, op string, err error) error
func Guard(port, op string, fn func() error) error
func GuardValue[T any](port, op string, fn func() (T, error)) (T, error)
func ContractViolation(port, op, detail string) error
func ErrorType(err error) string
```

`Guard` and `GuardValue` take no `ctx`; a `fn` that needs one captures it. They are
the only code allowed to call `recover` (check A5). Both are synchronous: there is
no sync/async split in Go, so context cancellation travels as an ordinary error
(FR-054).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-008 | When `fn` returns nil, `Guard` returns nil. `GuardValue` returns the value and a nil error unchanged. Neither wraps anything on success. | architecture.md §4 | happy |
| FR-009 | When `fn` panics, `Guard` / `GuardValue` MUST recover and return `*PortError{Port, Op, Err: *PanicError}`. `PanicError` holds the panic `Value` and the `Stack` (`runtime/debug.Stack()` captured at recovery); `Unwrap() []error` returns `ErrPortPanic` and, when `Value` is an `error`, that error. A port is third-party code, so a panic in it, including a runtime error, is a port failure, not a programmer error of this module. | architecture.md §6 | unhappy (failure of collaborator) |
| FR-010 | When `fn` returns an error that is not a `*PortError` for the same port, `Guard` / `WrapPort` return `*PortError{Port, Op, Err: err}`. When the outermost `*PortError` in the chain already has the same `Port`, the error is returned unchanged (no double wrapping). A `*PortError` for a different port is wrapped (the outer port wraps the inner). | architecture.md §6, observability.md §7 | unhappy (rule interaction) |
| FR-011 | `Guard` MUST NOT recover a panic raised by srd5e's own code outside `fn`, and MUST NOT run anything but `fn` and its deferred recovery. A `MustX` panic that originates in module code called from `fn` is indistinguishable from a port panic; therefore a use case calls domain `MustX` helpers outside `Guard`. | architecture.md §6 | unhappy (state error) |
| FR-012 | `runtime.Goexit` from `fn` MUST NOT be turned into an error or swallowed: `Guard` does not set a result when `recover()` returns nil. | architecture.md §6 | unhappy (failure of collaborator) |
| FR-013 | `Guard` calls `fn` exactly once and never retries. | architecture.md §4 | happy |
| FR-014 | `ContractViolation(port, op, detail)` MUST return `*PortError{Port, Op, Err}` where `Err` wraps `ErrContractViolation` and carries `detail` as data (a string field of the cause, not the sentinel). | architecture.md §4 | happy |
| FR-015 | `WrapPort` and the guards MUST preserve `errors.Is` for the original error, for `context.Canceled` and for `context.DeadlineExceeded`. | architecture.md §6 | happy |
| FR-016 | The panic value of a recovered panic is never re-panicked, and its `Stack` is preserved in `PanicError`, so the report points at the port failure (Go form of preserving the original stack trace). | architecture.md §4 | unhappy (failure of collaborator) |
| FR-017 | `Guard` / `GuardValue` / `WrapPort` are safe to call from multiple goroutines (they hold no state). | go.md §2 | happy |
| FR-037, FR-038, FR-039, FR-040, FR-052 | Withdrawn: Go has no sync/async port split and no `Error`/`Exception` distinction (see Withdrawn ids). | n/a | n/a |

### 4.3 Result conventions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | A result with different data per variant MUST be an interface with an unexported marker method and one struct type per variant (the "closed variant set"). The `//sumtype:decl` marker is planned, not yet applied (go.md §3). A result with one shape is a struct; a single named state is a typed enum with `String()`. An expected outcome, absence or rejection MUST be returned as a value and MUST NOT be an `error` and MUST NOT be a panic. Naming is FR-041; shared failure detail is FR-042. | architecture.md §4, §6; decisions FR-017, FR-018 | happy |
| FR-019 | Result variants MUST be immutable (unexported fields or read-only exported fields, value receivers) and compare by value: equal data compares equal with `==` where the type is comparable, otherwise through an `Equal` method; any differing field gives inequality. | architecture.md §3 | happy |
| FR-020 | A closed result and any rejection or problem type MUST carry typed data (ids, enums, numbers, value objects), never engine-generated display text: no exported field or method named `Message`, `Description`, `Text` or `DisplayText`. Consumers render their own text. Exempt, because they hold verbatim SRD text or describe telemetry: content value objects (`Spell.Description`, `FeatureEntry.Text` and the like) and telemetry descriptors (the instrument `Description` of observability-ports.md FR-025). The exemption is by declaration site: files under `content/` and the descriptor types in `telemetry/` are not scanned. | architecture.md §6 | architecture check (A8) |
| FR-021 | A collection exposed by a result or other value object that crosses a package boundary MUST be a copy: mutating the returned slice does not change the value, and mutating the slice passed to the constructor afterwards does not change the result (copy on construction and on read, or expose an `iter.Seq`). | architecture.md §4 ("Handoffs") | unhappy (immutability) |
| FR-022 | Results SHOULD carry the information needed to explain them (inputs, modifiers applied, roll made, DC) so consumers render their own breakdown. | architecture.md §6 | happy |
| FR-023 | A decoder or aggregate parser for untrusted input MUST return `(T, error)` and report every problem found, not only the first, through `errors.Join` of typed problems. Each problem has a path and a typed kind, ordered deterministically (input order of the offending items). Scalar constructors follow FR-035. The problem type is owned by the context (OQ-2). | architecture.md §5 | unhappy (invalid input) |
| FR-024 | A decoder MUST NOT return a non-nil error that holds no problem, and MUST return a nil error only for a fully valid input: `errors.Join()` of an empty list is nil, so the code builds the list first and returns nil only if the list is empty. A single-problem failure is valid. Tested in content-schema-and-decoder.md against its real type. | architecture.md §3 | unhappy (empty and minimal) |
| FR-025 | `NewX` and `ParseX` MUST NOT panic for any input of their parameter type, including extreme values, empty strings, nil slices and empty collections; they return an error instead. | architecture.md §6 | unhappy (invalid input) |
| FR-026 | For the same input, `MustX` panics if and only if `NewX` / `ParseX` returns an error, and the panic value is that error (so `errors.As` works on the recovered value). | architecture.md §6 | unhappy (rule conflict) |
| FR-027 | A constructor rejecting an argument MUST return an error that is `errors.Is` a documented sentinel or `errors.As` a typed error carrying the argument name and invalid value as fields (for example `*RangeError{Name, Value, Min, Max}`). Every such error also satisfies `errors.Is(err, srderr.ErrInvalidInput)`. Tests assert these, never the message text. | architecture.md §3, §6; testing.md §4 | unhappy (invalid input) |
| FR-028 | A panic MUST be used only for a broken invariant or API misuse that is a bug (in `MustX` and unreachable branches). An action that is illegal in the current state, but that a caller can reasonably attempt, MUST be a result variant. | architecture.md §6 | unhappy (state error) |
| FR-035 | A scalar constructor (`dice.ParseExpression`, `ability.NewScore`) MUST return one typed rejection reason (a sentinel or typed error, optionally with a small enum `Reason`), never display text. It need not list every problem (FR-023 is for decoders). | architecture.md §5, §6 | unhappy (invalid input) |
| FR-041 | Every use case returns its own closed result named for the operation (`<Op>Result`). Its variants are prefixed with the operation (`DamageApplied`, `DamageRejected`, `CreateCharacterContentMissing`); a prefix that only repeats the package name is dropped (`d20.RolledCheck`, not `d20.RolledCheck`). The generic names `Applied`, `Rejected`, `Succeeded`, `ContentMissing`, `Found`, `NotFound` MUST NOT be exported from a context package. | decisions FR-017; architecture.md §4, §9 | architecture check (A7) |
| FR-042 | Failure detail shared by several results MUST be a value object held as a field of each variant, never a shared variant, because a closed variant set cannot span packages (its marker method is unexported). `content` owns `MissingContent{Kind, ID}`; this spec defines no shared detail type. | decisions FR-018 | happy |
| FR-043 | A failure returned as a result variant (decode failure, content not found; not a rule refusal, which is a normal outcome per observability.md §7), not as an error, MUST NOT be given an `error.type` value: that attribute is the closed set of FR-034. A use case counts a returned failure with `srd5e.outcome` = `failure` (the only values are `success` and `failure`) and puts the variant in a low-cardinality attribute defined by its own spec. A spec MUST NOT add other `srd5e.outcome` values, such as `rejected`. Decided (OQ-11). | decisions FR-041, FR-051; observability.md §5 | unhappy (rule interaction) |
| FR-044 | Every interface with only unexported marker methods (a closed variant set) MUST have a doc comment naming every type in its package that implements the marker. Enforced by check A7. | architecture.md §4, §9 | architecture check |
| FR-045 | The repo's own switches over a closed result MUST list every variant and use no catch-all that hides a new variant. A `default:` is allowed only to panic with an error value as an unreachable branch (go.md §3). The `exhaustive` linter checks enum switches today; `gochecksumtype` for closed interfaces is planned (go.md §3), not enabled. Example functions show the switch without a default. | architecture.md §4 | happy |

### 4.4 Placement and use

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-029 | `PortError`, `WrapPort`, `Guard`, `GuardValue` and `ContractViolation` MUST be referenced only from use-case files, defined by role (the file holding a use case entry point that takes ports and emits telemetry, for example `d20/resolver.go`; the `_usecase.go` suffix is not required, go.md) (and `srderr` itself, `*test` packages, and the package that declares the `const Port` of the port being wrapped, which guards its own port: `dice.Roll` is the dice port's guard, not a use case). Domain files are pure and synchronous and report expected outcomes as results. Check A6. | architecture.md §1, §2, §6 | architecture check |
| FR-030 | `recover()` MUST appear only in `srderr/guard.go` and in `*test` packages and `_test.go` files. Check A5. | architecture.md §6 | architecture check |
| FR-046 | `panic` MUST appear only inside functions named `Must*` or in an unreachable `default:` branch, and its argument MUST be an `error` value, never a string literal. Check A5. | architecture.md §6 | architecture check |
| FR-047 | Recording an error on a span does not require `recover`: a use case ends its span with `defer span.End()` and records a returned error with `span.RecordError(err)`. No `defer func() { recover() ... }` exists outside `Guard`. | architecture.md §6; observability.md §7 | architecture check |
| FR-048 | A use case MUST NOT start a goroutine to wait for a port; cancellation is the port's responsibility through its `ctx` (check A4: no `go` statements). | architecture.md §4 | architecture check |
| FR-031 | A use case MUST wrap each error returned by a port exactly once with `WrapPort` (or `Guard`) and return it: it MUST NOT convert the failure to a result and MUST NOT swallow it. `errors.Is` / `errors.As` still reach the original cause. | architecture.md §4, observability.md §7 | unhappy (failure of collaborator) |
| FR-032 | Every port whose failure contract is an `error` return (`dice.Roller`, the content source) MUST ship a shared contract suite in its `*test` package (`dicetest.RunRollerContract(t, factory)`) proving that the adapter returns an error and does not panic on invalid input, and a failing double (`dicetest.CallbackRoller (returning an error)`) that use-case tests use to prove FR-031. The telemetry ports are exempt (OQ-4): no error returns. | architecture.md §4; testing.md §7 | unhappy (failure of collaborator) |
| FR-033 | The `srderr` API of section 4.2, `PortError`, `PanicError` and the sentinels MUST be exported with doc comments, so consumers branch on them and write use-case style wrappers with the same guards. A consumer adapter returns errors; it does not build `PortError`. Each port name constant is exported by its owning package (dice-and-rolls.md, content-source-port.md). | architecture.md §7 | happy |
| FR-034 | A use case that records an error MUST set the `error.type` attribute to `srderr.ErrorType(err)`, never `%T` or `err.Error()`, so the attribute stays within the closed set of section 7. | observability.md §5, §7; decisions FR-041 | unhappy (failure of collaborator) |
| FR-036 | A port that returns data breaking its documented contract (for example a `Roller` returning a result for a different expression) has failed, not the caller. Detection MUST return `srderr.ContractViolation(port, op, detail)` (a `*PortError` wrapping `ErrContractViolation`), never a panic. A panic is reserved for a caller breaking a documented precondition. | package design (extends architecture.md §4, §6) | unhappy (failure of collaborator) |
| FR-051 | Telemetry port methods return no error. A panic in a telemetry backend is the host's bug: it propagates unrecovered, no counter is incremented, no `error.type` is set, and `defer span.End()` still runs. A use case adds no recovery for it. | architecture.md §6; observability.md §7 | unhappy (failure of collaborator) |
| FR-053 | The test plan of every use-case spec (one whose section 7 defines a `srd5e.<context>.<operation>` span) MUST contain a row for each of FR-031, FR-034 and FR-051, or state that the use case calls no port. Enforcement is a spec-set scan in `internal/archtest`, beside the citation check of decisions FR-052. | testing.md §2 | unhappy (rule interaction) |
| FR-054 | A function taking `ctx` that observes cancellation returns `ctx.Err()` (possibly wrapped by `WrapPort`); a use case MUST NOT convert `context.Canceled` or `context.DeadlineExceeded` into a result variant, and MUST NOT record it as a failure outcome (it is neither success nor failure of the rule). | go.md §4 | unhappy (failure of collaborator) |
| FR-055 | `srderr` owns exactly these sentinels: `ErrInvalidInput` (untrusted input failed validation; every validation error `Is` it), `ErrPortPanic` (a port panicked; in every `PanicError` chain) and `ErrContractViolation` (a port returned contract-breaking data). Adding a sentinel is a spec change. A package may add its own sentinels for finer branching (for example `ability.InvalidError`, which carries a `Reason`). | package design | happy |
| FR-056 | Error strings MUST be lowercase with no trailing punctuation; a `PortError` renders as `<port> <op>: <cause>` (for example `dice roll: invalid expression`). The format is for humans and is not a contract. | go.md §3 | happy |
| FR-057 | Every fallible exported function's doc comment MUST state what it returns on each failure class (validation error, port failure, cancellation) and whether it can panic. | go.md §2 | happy |

### Withdrawn ids

| Id | Status |
|---|---|
| FR-037 | Withdrawn: Go has no separate async guard; `Guard` is the only form. |
| FR-038 | Withdrawn: there is no wrapper callback that can fail. |
| FR-039 | Withdrawn: covered by FR-009 / FR-016 (`PanicError.Stack`). |
| FR-040 | Withdrawn: a Go guard recovers every panic (FR-009); non-error returns do not exist. |
| FR-052 | Withdrawn: no future-returning body exists to misuse. |

## 5. Model

| Situation | Go form | Examples |
|---|---|---|
| Expected outcome, absence, rejection | Closed result value: struct, typed enum, or sealed interface (FR-018, FR-041) | A failed D20 Test; content id not found |
| Untrusted input fails validation | `(T, error)` from `NewX` / `ParseX` / a decoder; typed error or sentinel (FR-027) | Host-supplied ability score; decoded content with problems |
| Infrastructure failure inside a port | `*srderr.PortError` wrapping the port's error | A `Roller` returns an error; the content backend fails |
| A port returns contract-violating output | `srderr.ContractViolation` (FR-036) | A `Roller` returns a result for a different expression |
| A port panics | `*PortError` wrapping `*PanicError` (FR-009) | A buggy host roller dereferences nil |
| Programmer error | `panic(error)` in `MustX` / unreachable branch, never recovered | `MustScore(99)` |

Split rule: if a well-behaved caller can trigger it with valid dependencies, it is a
result. If it is invalid untrusted input, it is a validation error. If a
collaborator outside the module failed or broke its port contract, it is a
`PortError`. If the caller broke a documented precondition, it is a panic.

### Types

`srderr` (stdlib only: `errors`, `fmt`, `runtime/debug`):

```go
// PortError reports that port Port failed while performing Op.
type PortError struct {
    Port string
    Op   string
    Err  error
}

func (e *PortError) Error() string { /* "<Port> <Op>: <Err>"; "<head>: failed" when Err is nil (FR-003) */ }
func (e *PortError) Unwrap() error { return e.Err }

// PanicError is the error a recovered port panic becomes.
type PanicError struct {
    Value any
    Stack []byte
}

var (
    ErrInvalidInput      = errors.New("invalid input")
    ErrPortPanic         = errors.New("port panicked")
    ErrContractViolation = errors.New("port contract violation")
)
```

Port names live with their port: `const Port = "dice_roller"` in package `dice`.

### Examples

A use case wraps once and returns; the caller branches with the standard library:

```go
func (u Resolver) Resolve(ctx context.Context, cmd Command) (Result, error) {
    ctx, span := u.tracer.Start(ctx, d20.SpanResolve)
    defer span.End()

    roll, err := srderr.GuardValue(dice.Port, "roll", func() (dice.Result, error) {
        return u.roller.Roll(cmd.Expression())
    })
    if err != nil {
        span.RecordError(err)
        return nil, err // already a *srderr.PortError
    }
    return decide(cmd, roll), nil // expected outcomes are values
}

// Consumer:
res, err := resolver.Resolve(ctx, cmd)
var pe *srderr.PortError
switch {
case errors.As(err, &pe):
    log.Printf("port %s failed", pe.Port) // branch on pe.Port, not on strings
case errors.Is(err, context.Canceled):
    // stop quietly
case err == nil:
    switch r := res.(type) {
    case Hit:
        _ = r
    case Miss:
    }
}
```

Constructors, parsers and `MustX`:

```go
s, err := ability.NewScore(31)
var ie *ability.InvalidError
if errors.As(err, &ie) { _ = ie.Reason } // typed
_ = errors.Is(err, srderr.ErrInvalidInput) // true

_ = ability.MustScore(15) // panics with the error NewScore would return
```

A decoder reports every problem:

```go
func Decode(data []byte) (Set, error) {
    var problems []error
    // ... append *Problem{Path, Kind} in input order ...
    if len(problems) > 0 {
        return Set{}, errors.Join(problems...)
    }
    return set, nil
}
```

## 6. Interface

- **One failure model, no sync/async split.** `Guard` is for any port call; the
  port's own `ctx` handles cancellation.
- **Port failure contract** (every port): expected absence or rejection is a
  closed result in the return type; infrastructure failure and contract-violating
  output are the method's `error`, wrapped as a `*PortError` by the caller.
- **Exports.** `srderr` is a normal public package (FR-033). `dicetest` and
  `telemetrytest` own doubles and contract suites. No facade or re-export.
- **Consumer view.** `errors.As(err, &pe)` reads `pe.Port`, `pe.Op`; `errors.Is`
  reaches sentinels and the original cause; `srderr.ErrorType(err)` gives the
  telemetry value.

## 7. Observability

Silent by design: error types and guards with no use case, no port call and no
I/O.

| Signal | Owned here |
|---|---|
| Metrics, spans, log events | none (nothing for doc/observability/signals.md) |

Nothing is logged or counted here; FR-010 and FR-031 prevent double recording. The
error counter, `error.type` and `srd5e.outcome` are shared signals owned by the use
case specs and `observability-ports.md`. Defects (a leaked unwrapped port error,
double wrapping, a swallowed panic, a duplicate port name) are caught by the
FR-006 and FR-008 to FR-016 tests and the contract suites (FR-032). At run time,
use cases record a returned `PortError` once (FR-031) with `error.type` =
`ErrorType(err)` (FR-034); a consumer should monitor any non-zero rate of a use
case error counter with an `error.type` from the table below, meaning a
host-supplied port is failing, panicking or breaking its contract.

Allowed `error.type` values (owned here; the key constant is owned by
observability-ports.md). A use case never uses `%T` or `err.Error()` (FR-034).

| `error.type` value | Port name constant | Owning spec |
|---|---|---|
| `dice_roller_error` | `dice.Port` | dice-and-rolls.md |
| `content_error` | `content.Port` | content-source-port.md |

A new port adds a row here (FR-006; membership by review). `content_error` is a forward registration:
the content port ships in phase 2. A returned failure (decode failure, not found)
has no `error.type` at all (FR-043).

## 8. Test plan

Tests live in `srderr/` (external `package srderr_test`) and `internal/archtest/`. Citing the FR id in each test is planned (testing.md §5 `Requirement:` convention), not yet done; the Go tests do not carry it.
Unit tests of `PortError` and the guards use test-local errors, so they import
no port package (testing.md §3). The registry of real port names is used only by
check A8 and the FR-004 / FR-005 tests, which are therefore `architecture` level.
Convention rows for result types are proved by the owning specs' tests on their
real types. Tests are table-driven with `t.Run`, use `t.Parallel`, and never assert
message text.

**Architecture checks** (owned here; each is tested in memory with a violating overlay on a clean fixture module (`testing/fstest`); comments and strings never trigger them, as the
checks work on the AST). FR-006, FR-007, FR-020, FR-029, FR-030, FR-041, FR-044,
FR-046, FR-047 and FR-048 are enforced by checks A4 to A8 (architecture.md §9).

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002, FR-003 | unit | `errors.As` finds `*PortError`; `Unwrap` returns the identical `Err`; nil `err` to `WrapPort` returns nil; nil `Err` field does not panic in `Error()`. |
| FR-004, FR-005 | architecture, unit | Per registered port: `ErrorType` equals the section 7 value (`<port>_error`); "" for a plain error; outermost `PortError` wins. |
| FR-006, FR-007, FR-049, FR-050 | architecture | Check A8: duplicate or malformed `Port` flagged; `Message`/`Description`/`Text`/`DisplayText` flagged, `Path` not; check A6 flags `PortError` in a non-use-case (value-object) file. |
| FR-008, FR-013 | unit | `fn` nil error and a returned value pass unchanged; `fn` called once (counter). |
| FR-009, FR-016 | unit | Table of panic values (`error`, `string`, `int`, nil-dereference runtime error): result is `*PortError` with `*PanicError`; `errors.Is(err, ErrPortPanic)`; the panic `error` reachable by `errors.As`; `Stack` non-empty. |
| FR-010, FR-015 | unit | Same-port `PortError` returned identical; different-port wrapped once; `errors.Is` reaches `io.EOF`, `context.Canceled`, `context.DeadlineExceeded`. |
| FR-011, FR-012 | unit | Domain `MustX` outside `fn` panics through; `runtime.Goexit` in a goroutine is not swallowed. |
| FR-014, FR-055 | unit | `ContractViolation` wraps `ErrContractViolation` with detail data; sentinels distinct. |
| FR-017 | unit (race) | Concurrent guard calls under `-race`. |
| FR-018, FR-044, FR-041, FR-042 | architecture, unit | Check A7: an interface with a marker whose doc omits a variant is flagged; generic exported variant name flagged; each variant carries its detail as a field in the owning spec. |
| FR-020 | architecture | Check A8 over every closed result and rejection or problem type; fixtures: a variant with a `Message` field is flagged; a content value object with `Text` under `content/` and a telemetry descriptor with `Description` are not. |
| FR-029, FR-030, FR-046, FR-047, FR-048 | architecture | Checks A4 to A6: `recover()` outside `guard.go`, `panic("literal")`, `panic` outside `Must*`/`default:`, and a `go` statement are flagged; `panic(err)` in `MustX` and in a `default:` branch is accepted. |
| FR-031 | use case | Per use case, with `dicetest.CallbackRoller (returning an error)`: returns `*PortError`, `errors.Is` reaches the double's error, returned once, not a result. |
| FR-032 | contract | Per port's suite: invalid input yields an error and no panic; negative self-test: an adapter that panics on invalid input makes the suite fail. Phase 0 covers `dice.Roller`. |
| FR-033 | scenario | `scenarios_test` imports only public packages: `errors.As` a `PortError`. |
| FR-034 | use case, architecture | Per use case: recorded `error.type` equals `ErrorType(err)`. Architecture: `%T` and `.Error()` not passed to the attribute helper in use-case files. |
| FR-036 | unit | In the port's owning spec: a wrong-data double yields `ContractViolation`; never a panic. |
| FR-043 | use case | Closed value sets: observability-ports.md. Per use case: a returned failure records `srd5e.outcome = failure` and no `error.type`. |
| FR-045 | example | `Example*` switches show no `default`; (the `gochecksumtype` lint is planned). |
| FR-051 | use case | Per use case: a panicking telemetry double propagates the panic; the span `End` double was called; no counter incremented. |
| FR-053 | architecture | Spec-set scan: each use-case spec's test plan cites FR-031, FR-034 and FR-051 or states it calls no port; a fixture spec missing a row is flagged. |
| FR-054 | unit | A cancelled `ctx` reaches the caller as `context.Canceled` through `errors.Is`, never as a result. |
| FR-019, FR-021, FR-023 to FR-028, FR-035, FR-056 | unit (owning spec) | Per-type cases below. |
| FR-022, FR-057 | review | SHOULD / documentation rows, checked in code review. |

Owed by owning-spec tests (they cite `Requirement:
doc/specs/errors-and-results.md FR-nnn`), with the per-type case each owes:
FR-018 an expected-outcome case returns a value without an error; FR-019 equality
and per-field inequality; FR-021 mutating a returned slice and the source slice
leaves the value unchanged; FR-023 several problems at once in deterministic
order; FR-024 an error implies at least one problem; FR-025 and FR-026 extremes and
empty input never panic, one input table through both `NewX` and `MustX`; FR-027
`errors.Is` / `errors.As` on the typed error, never message text; FR-028 an
illegal-state action returns a variant; FR-035 a typed reason, no string display
field; FR-056 no assertion on message text.

Deliberately not covered: FR-022 and FR-057 - what a result carries and whether a
doc comment states its failures cannot be asserted mechanically beyond check A9's
presence test; judged in review.

Deliberately not covered: FR-023, FR-024, FR-028, FR-031, FR-041, FR-042, FR-043,
FR-051 - no decoder, state-dependent action or use case exists in phase 0; each
owning spec owes the row (FR-053 requires it).

- **Mutation testing:** `gomu` on `srderr`, threshold 80, run only inside Docker
  with `--cpus` and `--workers` capped at a quarter of the host's processors (command in
  testing.md). Never run it as a bare host process.

**Implementation status.**

- Done (code in `srderr/`, tests in `srderr/*_test.go`): every live id except those below. FR-022 and FR-057 are review-only.
- Not done: FR-018 (the `//sumtype:decl` marker is not applied), FR-045 (`gochecksumtype` is not in `.golangci.yml`; only the `exhaustive` linter and Example coverage exist), and FR-023, FR-024, FR-028, FR-031, FR-041, FR-042, FR-043, FR-051 (owed by the specs that add a decoder, a state-dependent action or a use case), FR-053 (the spec-set scan has no code in `internal/archtest`) and the citation scan of shared-types FR-052 (also not in `internal/archtest`).

Not yet implemented: FR-018 (marker), FR-023, FR-024, FR-028, FR-031, FR-041, FR-042, FR-043, FR-045 (lint), FR-051, FR-053 (scan), shared-types FR-052 (citation scan).

## 9. Open questions and decisions

No open questions. Settled ones, one line each:

| Id | Decision |
|---|---|
| OQ-1 | No generic `Result[T, E]`: each use case has its own closed result (FR-041). Revisit only if three or more contexts copy one shape after release. |
| OQ-2 | No shared problem type; each context owns its own (decisions FR-018; FR-042). |
| OQ-3 | Guards live in `srderr` (OQ-10). |
| OQ-4 | The telemetry ports return no errors in phase 0; a panicking backend propagates (FR-051). Reopen only if a real failure shows up. |
| OQ-5 | `PortError` is a concrete struct, not an interface with per-port subtypes: Go's `errors.As` on one type plus a `Port` field gives the same branching with one type to learn; the closed set of ports is documented (FR-049). Adding a port is non-breaking, removing or renaming a port name is **BREAKING**. |
| OQ-6 | Per-port detail rides in the wrapped cause (typed error or sentinel), owned by the port spec; the wrapper carries only `Port`, `Op`, `Err`. |
| OQ-7 | Guard signatures are frozen (4.2). `GuardValue` is generic because it removes real duplication in an implementation helper, never in a port (go.md §6). |
| OQ-8 | `PanicError.Value` is `any`: Go can panic with anything and ports are third-party code. |
| OQ-9 | Checks A1 to A13 are owned by architecture.md §9 and `internal/archtest`; this spec owns the semantic rules they enforce. |
| OQ-10 | `srderr` is a kernel package with no module imports, so every package may use it and `d20`-style contexts can return validation errors that wrap `ErrInvalidInput`. Use of `PortError` itself is confined to use-case files (FR-029). |
| OQ-11 | A returned failure carries no `error.type` (FR-043; decisions FR-041, FR-051). |
| OQ-12 | Guards recover all panics, including runtime errors, because a port is third-party code and its panic is its failure. Panics from this module's own `MustX` are bugs and propagate (FR-011). |

## 10. Changelog impact

`CHANGELOG.md`, `Unreleased`, **Added**:

- "`srderr` package: `PortError`, `WrapPort`, `Guard`, `GuardValue`,
  `ContractViolation`, `ErrorType`, sentinel errors, and the documented rules for
  results versus errors versus panics."

Not **BREAKING**: the module is unreleased. After release, removing or renaming a
port name or an `error.type` value is **BREAKING** (architecture §7, observability
§5); adding a sentinel or port is not.

---

## Read receipt

RULE-TAG: `tundra-beacon-dbec5f`

This document is a spec, not a rule file; the line above is kept for traceability
of the rewrite and is not collected by the rule-tag check (A13 scans
`doc/rules/*.md` only).
