# srd5e Architecture Rules

Normative. MUST / MUST NOT / SHOULD are binding on all code and review.

Target architecture: **Domain-Driven Design** inside a **Hexagonal
(Ports & Adapters)** structure, realised as flat, public Go packages. The
module is a library with no UI, storage or framework. The hexagon *is* the
module; every consumer is an adapter on the outside. Go-specific conventions
(naming, doc comments, generics, compatibility) are in [go.md](go.md); the
failure model in detail is in
[errors-and-results.md](../specs/errors-and-results.md).

Goals, in priority order:

1. Idiomatic Go.
2. Testable: every unit is testable with plain values and deterministic doubles.
3. Decoupled modules that can be replaced independently.
4. A rules model that reads like the fifth-edition rules in SRD 5.2.1 itself.

---

## 1. Hexagonal structure

```
                 driving side                         driven side
        (consumers call into us)             (we call out to the host)

  host app ──► [ use cases ]  ┌──────────────┐  [ ports ] ──► adapter
  CLI      ──►  (exported     │    DOMAIN    │  dice.Roller   dice.SeededRoller
  server   ──►   funcs/types) │ (pure rules) │  content src   host's own impls
                              └──────────────┘  telemetry.*   telemetry.Noop
```

- **Domain**: pure rules. Entities, value objects, aggregates, domain events,
  domain services. Pure CPU, synchronous, deterministic.
- **Use cases** (the *driving side*): what consumers can ask for. Exported types
  or functions in a context package. They orchestrate; they hold no rules logic.
- **Ports** (the *driven side*): small interfaces for everything the domain needs
  from outside (§4). Defined where they are consumed.
- **Adapters**: implementations of ports and decoders from external formats into
  domain types. The host supplies its own; the module MAY ship generic ones.

### Package layout

Packages are flat, public and short-named. There are no barrels, no facade
package and no re-exports: a consumer imports the package that owns the type.
`doc.go` at the module root declares `package srd5e` for documentation only (no
logic, no aliases).

| Package | Holds | Kind |
|---|---|---|
| `srd5e` (root `doc.go`) | Module documentation only | docs |
| `dice` | `Die`, `Expression` (parses `2d8+2`), `Roll`, `Result`, the `Roller` port, the `SeededRoller` adapter | kernel |
| `dice/dicetest` | `FixedRoller`, `CallbackRoller`, min/max rollers, contract suites `RunRollerContract(t, factory)` and `RunFailureContract` | test support |
| `ability` | `Ability` enum, `Score`, derived modifier | kernel |
| `modifier` | `Modifier`, advantage/disadvantage `RollMode`, `RollCircumstance` | kernel |
| `srderr` | `PortError`, `WrapPort`, `Guard`, `GuardValue`, `ContractViolation`, `ContractError`, `PanicError`, `Quote`, sentinel errors, `ErrorType` (§6) | kernel |
| `telemetry` | `Logger`, `Tracer`, `Meter` ports, zero-value-usable `Noop` implementations, signal-name constants | kernel |
| `telemetry/telemetrytest` | Recording doubles and the contract suite for the telemetry ports | test support |
| `telemetry/slogtelemetry` | Optional adapter from `log/slog` to `telemetry.Logger` (stdlib only) | adapter |
| `dice/randroller` | Unseeded, concurrency-safe `dice.Roller` over `math/rand/v2` (stdlib only) | adapter |
| `content/data` | Embedded SRD JSON and its `go:embed` file (created with the first content) | adapter |
| `content/contenttest` | Test doubles for the content ports (`FakeSource`, `FakeDecoder`) and their contract suite | test support |
| `d20` | First bounded context: domain and use case in one package, unexported helpers, one file per concept | context |
| `internal/archtest` | Architecture tests (§9) | tooling |
| `scenarios` | Cross-package tests through the public API only (`package scenarios_test`) | tests |

Future contexts (`content`, `character`, `combat`, `toolbox`) each become one
top-level package and are not created before their specs are implemented.
Bundled content data lives in `content/data/` as JSON files embedded directly
with `go:embed` (no generator step).

A package named `x` that owns test support puts it in `x/xtest` (like
`net/http/httptest`). Test doubles are never mixed into production packages.

### Kernel packages

`dice`, `ability`, `modifier`, `srderr` and `telemetry` are the **kernel**: small,
shared by every context, free of context knowledge. The kernel MUST stay small;
if only some contexts need a concept it belongs to the most upstream of them and
is published from there. Changing a kernel package affects every context, so
treat changes as breaking-change candidates.

### Dependency rule

Dependencies point inward only. The allow-list below is the single source of
truth; `internal/archtest` (§9) encodes it as a table and fails when a package
imports something not listed.

| Package | MAY import (module packages) | MUST NOT import |
|---|---|---|
| `srd5e` (root) | nothing | everything |
| `srderr` | nothing | everything else in the module |
| `telemetry` | nothing | everything else in the module |
| `modifier` | `srderr` | everything else |
| `dice` | `srderr` | everything else |
| `ability` | `srderr` | everything else |
| `dice/dicetest` | `dice`, `srderr` | contexts, other `*test` packages |
| `telemetry/telemetrytest` | `telemetry` | contexts, other `*test` packages |
| `telemetry/slogtelemetry` | `telemetry` | everything else |
| `dice/randroller` | `dice`, `srderr` | everything else |
| `content/data` | `content`, kernel packages | every context; imported only by tests and consumers |
| `content/contenttest` | `content`, `srderr` | other contexts, other `*test` packages, kernel packages other than `srderr` |
| `d20` | `dice`, `ability`, `modifier`, `srderr`, `telemetry` | `*test` packages (production files), `internal/` |
| `scenarios` (tests only) | every public package, including `*test` | `internal/` |
| `internal/archtest` | nothing in the module | everything (it scans files, it does not import them) |

Rules that hold for every row:

- Standard library only. `go.mod` MUST have no `require` directive. Third-party
  runtime dependencies are forbidden; adapters to third-party systems
  (OpenTelemetry, zap, Sentry) live in the consumer's repo or in documented
  examples, never here.
- Only `_test.go` files import `*test` packages, and only `*test` packages and
  `scenarios` may import more than one parent package (the parent package of
  a `*test` package is the package it supports, e.g. `dice` for `dicetest`).
- Nothing imports `internal/`; only `internal/archtest` lives there and it is
  executed by `go test`, never imported.
- A kernel package MUST NOT import a context package.
- A context package imports kernel packages and the **published API** of its
  upstream contexts (§2), nothing else.
- Dependencies form a directed acyclic graph (the compiler enforces no cycles;
  the test enforces the direction).

---

## 2. Bounded contexts (modules)

Each top-level package other than the kernel, the tests-only `scenarios` directory and `internal/` is a
**bounded context**: one cohesive part of the rules with its own model and
vocabulary. A context is **one Go package**:

```
<context>/
├── doc.go             # package documentation: purpose, inputs, outputs, handoff types
├── ports.go           # (optional) consumer-side interfaces, once a context has several; may sit beside their use case
├── <concept>.go       # one file per domain concept (value object, aggregate, service)
├── <usecase>.go       # one file per use case (d20: resolver.go); the only files that call ports
├── *_test.go          # external tests (package <context>_test); internal only for edge logic
└── testdata/          # fixtures and golden files
```

Rules:

- A context MUST be understandable and testable on its own.
- The package's exported identifiers are its published API. Everything else is
  unexported; there is no `internal/` subtree for contexts.
- The **published language** of a context is the subset of its exported API made
  of value objects, ids and domain events. Downstream domain code MAY use it;
  downstream code MUST NOT call another context's use cases.
- Prefer referencing another context's aggregates **by id**, not by holding the
  object.
- If two contexts need the same concept, the upstream one owns and publishes it.
  Move it to the kernel only if every context needs it. Do not duplicate it and
  do not reach into unexported detail (the compiler already forbids that).
- `content` owns every closed SRD vocabulary type and typed id (`DamageType`,
  `ConditionKind`, `Skill`, `CreatureSize`, `WeaponProperty`, `ClassID`,
  `SpellID`, and so on). Vocabulary types carry static SRD facts only; the
  behaviour that interprets one is owned by the context where the rule is
  applied. This keeps `content` upstream of `character` and `combat`.
- Domain events: each context declares its own `Event` interface (a sealed
  marker, §4). There is no kernel `DomainEvent` type.

### Context map

Context dependencies MUST form a directed acyclic graph. The allowed directions
are listed here and nowhere else; anything not listed is forbidden. The kernel
packages are always allowed and are not repeated.

| Context package | Purpose | May import (upstream contexts) |
|---|---|---|
| `content` | Species, backgrounds, classes, feats, spells, equipment, monsters; content-source port and decoders | none |
| `d20` | D20 Tests (ability checks, saving throws, attack rolls), DCs, Advantage and Disadvantage, success and failure resolution, from modifiers and a roll | none |
| `character` | Abilities, Proficiency Bonus, level, derived stats, spellcasting, equipment rule maths (Armor Class, Carrying Capacity, crafting) | `content`, `d20` |
| `combat` | Initiative, actions, attacks, conditions, hit points, damage | `character`, `d20`, `content` |
| `toolbox` | Gameplay Toolbox rules (travel pace, environmental effects, poison, traps, curses and contagions, fear and mental stress, XP-budget encounter planning) | `content`, `d20`, `character`, `combat` |

Nothing imports `toolbox`. Only `d20` exists today; the others are reserved
names. `internal/archtest` encodes this table and the §1 table together.

The contexts are **provisional**. Adding, merging or re-pointing a context is an
architecture change: update this table, `AGENTS.md` and the architecture test
table (§9) in the same change. A new edge MUST NOT create a cycle; if two
contexts seem to need each other, a concept is in the wrong place.

---

## 3. Domain model (DDD)

### Ubiquitous language

- Names MUST use the terminology of **SRD 5.2.1** exactly: `D20Test`, `Score`
  in `ability`, `ProficiencyBonus`, `Advantage`, `Disadvantage`, `ArmorClass`,
  `HitPoints`, `Species` (not `Race`). Do not invent synonyms.
- Terms from other editions of the rules and removed concepts MUST NOT appear in
  the domain. Decoders translate them at the boundary (§5).
- When the rules text and the code disagree on a term, the code changes.
- Domain code MUST NOT use generic technical names (`Manager`, `Helper`,
  `Processor`, `Data`) for domain concepts.
- Go naming applies on top (no stutter, initialisms): see [go.md](go.md) §1.

### Building blocks

- **Value object**: immutable, comparable by value, no identity, no side
  effects. Most rules concepts are value objects: `modifier.Modifier`,
  `ability.Score`, a difficulty class. Prefer these. Unexported fields, a
  validating constructor, value receivers, no setters; methods return new
  values (§3 "Immutability").
- **Entity**: identity persists across changes (a character, an encounter).
  Identity is an explicit typed id (`type CharacterID string`), not a raw
  `string`. Entities MUST NOT carry timestamps or persistence concerns.
- **Aggregate**: a cluster of entities and value objects with one **aggregate
  root**. Outside code holds and calls the root only and never mutates inside
  parts. The root enforces the invariants.
- **Domain event**: an immutable fact about something that happened in the
  rules (`DamageTaken`, `ConditionApplied`), named in the past tense.
- **Domain service**: rules logic that belongs to no single entity or value
  object (for example resolving a check against a DC). Stateless; takes
  everything it needs as arguments.

### Immutability and state transitions

- Domain types MUST be immutable: unexported fields, exported accessor methods
  named without `Get` (`Value()`, not `GetValue()`), value receivers, no
  setters. Small values are never handled through pointers.
- A state change returns a new value, plus the domain events it produced. Never
  mutate in place.
- Constructors validate (`NewX`, `ParseX` return `(T, error)`; `MustX` panics,
  §6) so an invalid instance cannot be built from untrusted input. Invariants
  are enforced by the type, not by callers.
- The **zero value** is either useful or explicitly invalid. Each type's doc
  comment says which; an explicitly invalid zero value has `IsZero()` or
  `Valid()`. A zero value MUST NOT silently mean something wrong.
- Comparable types stay comparable (no slices, maps or funcs in value objects
  that are compared with `==`). Where a collection is part of a value, expose a
  copy (`slices.Clone`) or an `iter.Seq`, and copy on construction.
- Closed sets of named states are typed enums (§4 results) with a `String()`
  method, not `bool` or `int` literals standing in for a state.
- Functional options (`WithX`) only where a constructor would otherwise take more
  than about three optional parameters.

### Purity and determinism

Domain code MUST be pure, synchronous and deterministic:

- No global mutable state, no `init()` side effects, no package-level
  randomness or time. Randomness and time enter only through injected ports
  (`dice.Roller`, a clock) and arrive in the domain as plain values.
- No goroutines, channels, `sync` or `context` in pure domain functions. Library
  types are immutable or documented not safe for concurrent use (§4, Concurrency).
- No `os`, `net`, `reflect`, `unsafe`, `math/rand`, `time.Now`/`time.Since`/
  `time.Sleep`, `log`, `fmt.Print*`, or any third-party package.
- No display strings. Return structured data (§6).

### Optional rules and extension points

The SRD presents some rules as optional or leaves them to the Game Master, and
consumers will want house rules. Support them without consumer-specific code:

- Optional rules are an immutable `RuleOptions` value (or one per context) that
  names each option the **SRD itself presents**. It is part of the state passed
  in a command, never a global setting.
- Domain code that varies by rule option uses an exhaustive `switch` over a typed
  enum or a domain-level strategy value. No scattered boolean flags.
- House rules and new content arrive as **content** through the content source
  (feats, items, effects expressed as data), not as code paths keyed to a
  consumer.
- Where a consumer needs to change behaviour content cannot express, expose a
  narrow, documented interface in the domain with the standard rule as its
  default. Add one only when a real need appears.

---

## 4. Use cases, ports and handoffs

### Use cases (driving side)

- A use case is a single-purpose exported type or function exposing one thing a
  consumer can ask for (`Resolve`, `CalculateArmorClass`). Prefer a function when
  there is no state to hold; a struct with a method when it carries injected
  ports.
- Input is an explicit immutable **command** value carrying all data needed.
  Output is an explicit **result** (below). No `map[string]any`, no long runs of
  loose positional primitives across a boundary.
- A use case orchestrates: calls ports, calls the domain, returns a result. It
  MUST NOT contain rules logic; that lives in domain files.
- A use case MUST NOT call another use case. It composes the pure domain
  services that upstream contexts publish, supplying dice and content through
  its own ports. Each use case records only its own signals
  ([observability.md](observability.md) §7).
- A use case receives its collaborators through its constructor (`NewX(ports...)`)
  or parameters, typed as small interfaces. It MUST NOT construct adapters or
  reach for globals. Telemetry is injected as individual ports or a bundle,
  defaulting to `telemetry.Noop` (the zero value works).
- `context.Context` is the first parameter of anything that may do I/O, block or
  emit spans (content sources, use-case entry points that call telemetry). Pure
  CPU domain functions and `dice.Roller` (synchronous, in-memory) take no `ctx`.
  `ctx` is never stored in a struct.
- State in, new state and events out: use cases take current state in the command
  and return the new state and events in the result. They hold no hidden mutable
  state, so the host decides how and where state is stored.

### Results: the Go forms

Expected rule outcomes are **return values**, never errors. Pick the form by what
the outcome carries:

| Outcome carries | Go form | Example |
|---|---|---|
| A single named state | Typed enum (`iota` or string-backed) with `String()` | `type Outcome int` with `Success`, `Failure` |
| One shape of data | A result struct with exported read-only fields or accessor methods | `CheckResult{Outcome, Roll, DC}` |
| Different data per variant (a closed set) | Interface with an unexported marker method plus one struct type per variant, consumed by an exhaustive type switch | `ApplyResult` = `DamageApplied` \| `DamageRejected` |

A closed variant set:

```go
// ApplyResult is the closed set of outcomes of Apply. Variants:
// DamageApplied, DamageRejected.
type ApplyResult interface{ isApplyResult() }

type DamageApplied struct{ State State; Events []Event }
type DamageRejected struct{ Reason RejectReason }

func (DamageApplied) isApplyResult()  {}
func (DamageRejected) isApplyResult() {}
```

- The interface's doc comment MUST list every variant (architecture test A7, §9).
  Variants are value types (not pointers) so they are comparable where their
  fields are.
- Consumers switch with `switch r := res.(type)`; the repo's own switches have
  no `default` for covered variants (a `gochecksumtype` lint is planned, not yet enabled). Where a `default`
  is needed to satisfy the compiler it panics with an error value (§6 "unreachable").
- Every use case has its own result named for the operation (`ApplyResult`).
  Variant names are prefixed with the operation unless the package-qualified name
  already says it. Reference exception: `d20.Result` with `RolledCheck`,
  `RolledAttack`, `ForcedFailure`. Within a package,
  bare generic names (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`,
  `Found`, `NotFound`) are not exported from a context package; the package
  qualifier is not enough when a context holds several use cases.
- Failure detail shared by several results is a value object held as a **field**
  of each variant, not a shared variant. (The marker method is unexported, so a
  variant set cannot span packages; fields can.)
- Results SHOULD carry what is needed to explain them (modifiers applied, roll
  made, DC), as data, never display text.

### Ports (driven side)

- A port is a small interface (1 to 3 methods), **named for what the domain
  needs**, not the technology (`dice.Roller`, not `RandWrapper`).
- **Accept interfaces, return structs.** A port is defined by its consumer. A
  context declares its own one-method interface for what it needs (for example
  `d20` declares a `Roller` with `Roll(dice.Expression) (dice.Result, error)`)
  and is not coupled to implementations; any type with that method satisfies it
  structurally.
- A port shared by several packages lives beside the type that defines its
  vocabulary (`dice.Roller`, like `io.Reader` in `io`). A port used by exactly one
  context lives in that context's `ports.go`. When a second context needs it,
  promote it next to its vocabulary type; do not import it across contexts.
- Ports use domain types only. No DTOs, maps, SQL, HTTP clients or `any`.
- Constructors return concrete types. Adapters carry a compile-time assertion:
  `var _ Roller = (*SeededRoller)(nil)`.
- **Failure contract.** Each port documents its failures:
  - Expected absence or rejection (content id not found) is part of the return
    type, as a result (§4 results). It is not an error.
  - Infrastructure or contract failure (I/O error, backend down, port returned
    data breaking its contract) is the port method's `error` return. The
    **caller** wraps it with `srderr.WrapPort(port, op, err)` and callers branch
    with `errors.Is` / `errors.As` (§6).
- **Shipped test support.** Every port ships an exported contract suite in its
  `*test` package (`dicetest.RunRollerContract`, and `RunFailureContract` for dice failures; a port may ship several) that every adapter and double
  passes, plus deterministic doubles (fixed, min/max, and a callback roller that returns an error for failure paths).
- **No-op defaults** (`telemetry.Noop`) are zero-value-usable types in the port
  package itself. They are not adapters: the use case needs the default, and may
  not import an adapter package.
- Expected ports: dice rolling, content lookup, observability (logging, tracing,
  metrics; see [observability.md](observability.md)). Add others only when a real
  need appears.

### Concurrency

- Library types are immutable or documented **not safe for concurrent use**.
  Immutable value types are safe to share.
- `SeededRoller` wraps a caller-supplied `*rand.Rand` and is NOT safe for
  concurrent use; its doc says so. Provide a mutex-guarded wrapper only if a spec
  requires one.
- Domain and use-case code spawns no goroutines. Concurrency is the host's choice.
  A context-aware method honours cancellation by checking `ctx.Err()` at its port
  boundary.

### Handoffs

Every boundary, between files of different packages, between contexts, and
between the module and its consumer, is crossed by one of: a command, a result,
a domain event, a value object, or an id. Nothing else.

- Handoff types are immutable and documented.
- Each context's `doc.go` MUST document its inputs and outputs and list its
  handoff types.
- A handoff type MUST NOT expose an internal collection for mutation; return
  copies or iterators.

---

## 5. Adapters and anti-corruption

- Adapters implement ports and translate between external formats and domain
  types. They are the only place that knows about JSON shapes, file layouts,
  legacy terminology or host specifics.
- External data enters the domain only through a **decoder** that parses and
  validates it into domain values ("parse, don't validate"). A decoder returns
  `(T, error)`; on invalid data the error is built with `errors.Join` of typed
  problems (path, kind) and reports **every** problem found, not the first. The
  domain MUST NOT see `map[string]any`.
- Generic adapters MAY be shipped (`dice.SeededRoller`, an in-memory content
  source, JSON decoders, the `slog` adapter). They MUST NOT assume a particular
  consumer.
- I/O adapters (file system, network) MUST live in a separate optional package
  so the core stays platform-neutral and the import allow-list stays small.
- Adapters bridging third-party systems MUST NOT be shipped in this module
  (they would force a dependency on every consumer). Document them as examples.
- Bundled content is JSON embedded with `go:embed` under `content/data/` and goes
  through the same decoder as host-supplied JSON. No generator step. `NOTICE.md`
  covers `content/data/` only; the MIT `LICENSE` covers code.
- A consumer's adapters live in the consumer's repo, never here.

---

## 6. Errors, results and panics

The failure model has four classes. Details, the `srderr` API and examples are in
[errors-and-results.md](../specs/errors-and-results.md).

| Situation | Go form | Caller expected to handle? |
|---|---|---|
| Expected rule outcome (failed check, illegal action in this state, not found) | **Return value**: result struct, typed enum, or sealed interface (§4) | Yes, by reading it |
| Untrusted input that fails validation (decoded content, host values) | `NewX` / `ParseX` return `(T, error)`; error is a typed value (`errors.As`) or wraps a sentinel (`errors.Is`); decoders `errors.Join` all problems | Yes |
| Infrastructure or port failure (roller, content source, telemetry backend) | Port method returns `error`; the caller wraps it with `srderr.WrapPort`; branch with `errors.Is` / `errors.As` | Yes |
| Programmer error (violated precondition, broken invariant) | `panic` carrying an `error` value, only inside `MustX` helpers and unreachable branches | No; it is a bug |

- Exported APIs that take untrusted input NEVER panic. No panic crosses a package
  boundary for an expected condition.
- Every fallible function's doc comment states what it returns on each failure
  class.
- Errors carry data, not display text. Error strings are lowercase with no
  trailing punctuation and are never branched on; tests assert `errors.Is` /
  `errors.As` and typed fields, never message text.
- `recover` appears only in `srderr.Guard` / `GuardValue` (and in test support).
  Everything else lets panics propagate: they are bugs.
- A use case never converts a port failure into a result and never wraps the same
  failure twice. Context cancellation (`context.Canceled`,
  `context.DeadlineExceeded`) stays visible through `errors.Is`.
- Telemetry port methods have no `error` return; a panicking backend is the
  host's bug and propagates.

---

## 7. Public API, packages and versioning

- Every package is public API, except `internal/`. Anything exported is a
  compatibility promise; export deliberately, keep helpers unexported.
- Every exported identifier has a doc comment starting with its name; every
  package has a `doc.go` with a package comment.
- Every port ships with its test support in the matching `*test` package.
  Contract suites are exported there (they import `testing`, which is why they
  are not in the production package).
- Examples are `Example*` functions in `_test.go` files so `go test` verifies
  them. An `examples/` directory is added only when it earns its keep.

### Versioning

- The module follows [Semantic Versioning](https://semver.org/) via git tags. It
  stays on the `v0` / `v1` module path; no `/v2` until a breaking v2 exists.
  Before `v1.0.0` a breaking change bumps the minor version (`v0.1.0` to
  `v0.2.0`), anything else the patch.
- Breaking changes include: removing or renaming an exported identifier,
  changing a signature, adding a variant to a closed variant set consumers switch
  over, changing a rule's result for the same input (unless fixing a bug against
  the rules), raising the minimum Go version, and renaming or removing a
  telemetry signal or attribute ([observability.md](observability.md) §5).
- Every user-visible change gets an entry in `CHANGELOG.md` under `Unreleased`,
  grouped Added / Changed / Deprecated / Removed / Fixed (Keep a Changelog).
  Breaking changes are marked **BREAKING**.
- Deprecate with a `// Deprecated:` paragraph for at least one release before
  removing, where practical. The Go compatibility promise is in [go.md](go.md) §9.

---

## 8. Testing

Per-layer approach, in short (full rules: [testing.md](testing.md)):

- Domain: plain table-driven unit tests with no ports, doubles or I/O.
- Use cases: the real domain with doubles from the `*test` packages.
- Adapters and doubles: the shared contract suite for their port.
- Scenarios: through the public API only, in `scenarios` (`package scenarios_test`).
- Randomness: deterministic rollers from `dicetest` only. `go test -race
  -shuffle=on -cover` MUST pass.

---

## 9. Enforcement

Rules are checked mechanically wherever possible. `internal/archtest` is a test
package (stdlib only: `go/parser`, `go/ast`, `go/token`, `os`, `path/filepath`)
that scans the module's `.go` files and MUST fail the build when it finds any of
the following. Each check is tested in memory (`testing/fstest`): a violating overlay on a clean
fixture module, so the checker itself is tested.

| Check | Enforces |
|---|---|
| A1 import allow-list | Every import of a module package in a non-test file matches the allow-list rows of §1 and the context map of §2, encoded as a table in `archtest`. An import not listed, a kernel package importing a context, or a context importing a downstream context fails. |
| A2 test-support isolation | A non-`_test.go` file importing any `*test` package; a `*test` package importing a context or another `*test` package; any import of `internal/`. In package `content`, the source value-type files (`lookup_result.go`, `query.go`, `catalogue.go`) import none of `context`, `embed`, `testing`, `content/contenttest` or `content/data` and never name `PortError`, and `content/contenttest/fake_source.go` imports only `content` from this module (rule `source-files`). |
| A3 stdlib deny-list | In kernel and context packages (non-test files): imports of `os`, `os/exec`, `net`, `net/*`, `io/fs`, `io/ioutil`, `reflect`, `unsafe`, `syscall`, `log`, `math/rand` (`math/rand/v2` only in `dice` for the `*rand.Rand` type), and `sync` / `sync/atomic` (no exception today; a mutex-guarded roller would add a named one in `stdlibExceptions`); any third-party import anywhere. Test-support and adapter packages are not scanned for standard-library imports; review covers them. |
| A4 no hidden effects | In non-test files: an `init` function; a package-level `var` other than `_` assertions and `Err*` sentinels; a call to `time.Now`, `time.Since`, `time.Sleep`, `time.After`, `time.Tick`; a `go` statement; a call to `fmt.Print*` or `println`; a call to `os.Exit`. |
| A5 panic discipline | `panic(` outside a function named `Must*` or a `default:` case clause; a `panic` whose argument is a string literal (must be an `error` value); `recover(` outside `srderr/guard.go` and `*test` packages. |
| A6 port use confined | A non-test file that calls `srderr.WrapPort`, `Guard`, `GuardValue` or `ContractViolation`, or names `srderr.PortError` (matched on the local import name, so aliases count), must be a use-case file, live in `srderr` or a `*test` package, or belong to the package that declares `const Port` (the owner of a port wraps its own failures, as `dice.Roll` does). A use-case file is defined by role: a file of a context package that imports `telemetry` and declares an exported type with an exported method. Selector expressions only, so comments are not matched. In package `content`, which owns the `Source` port and so is exempt above, only `load_usecase.go` refers to `PortError`, `WrapPort`, `Guard`, `GuardValue` or `ContractViolation` (rule `content-port-use`). |
| A7 sealed sets documented | For each interface with at least one unexported method (a closed variant set), the doc comment names, as whole words, every type of the package that has all of the interface's methods (value or pointer receiver; types that gain the methods by embedding are not seen). Generic variant names (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`, `Found`, `NotFound`) declared as a type or const in a context package fail. |
| A8 error shape | No exported field or method named `Message`, `Description`, `Text` or `DisplayText` on an exported error or result type: a type with an `Error()` method, a name ending in `Error`, `Result`, `Rejection`, `Problem` or `Reason`, or an implementer of a sealed interface. Files under `content/`, the `telemetry` package and `*test` packages are not scanned. Every top-level `const Port` is a string literal matching `^[a-z][a-z0-9_]*$` and is unique in the module. That a new port is listed in the errors-and-results table is checked in review. |
| A9 docs and naming | Every exported identifier has a doc comment starting with its name; every package has a package comment in `doc.go` (`content/data`, which may hold only one Go file, carries it in that file); no stutter (an exported identifier does not begin with its package name, see [go.md](go.md) §1). |
| A10 module hygiene | `go.mod` has no `require`; no file contains the name of a consuming project (the org name appears only in the module path/import paths and `AGENTS.md`); every package directory is listed in the §1 table or §2 context map. Each closed vocabulary type, `CreatureSpace` and typed id of the content schema (schema FR-128, FR-130) is declared at most once in the module's non-test files, and in package `content` (rule `vocabulary-declaration`; the real-module test also requires each to exist). Only the files named `decode*.go` of package `content` import `encoding/json` (rule `json-confined`). Once `content/decode.go` exists, `.github/workflows/ci.yml`, `AGENTS.md` and `doc/specs/release-0-1-0.md` each name the 32-bit decoder run `GOARCH=386 go test ./content/...` (rule `thirty-two-bit`, schema FR-099). |
| A11 cycles and unknown packages | A cycle in the package graph, or a package directory in neither table. |
| A12 embedded data | Only `content/data` imports `embed` (non-test files). `content/data` is a Policy row of its own (it may import `content` and the kernel; no context may import it). It holds only `.json` files, `.gitkeep`, `README.md`, test files and exactly one other Go file, which carries `//go:embed`; its `//go:embed` variable is the one package-level var A4 allows besides sentinels. Once it holds bundled content, the backticked paths ending in `/` in the "Scope" section of `NOTICE.md` equal `content/data/` exactly. Vacuous while `content/data` is empty or absent. |
| A13 rule tags | Each `doc/rules/*.md` has exactly one line starting `RULE-TAG:` whose value is `` `<file stem>-<12 lowercase hex digits>` ``; the value appears nowhere else in the repository (other files, `.github/`, `internal/`, other lines of its own file, other rule files); each rule file is in the `AGENTS.md` "Read the rules first" list and each path listed there exists. |

Implemented: A1 to A13, plus the package-var tightening of A4 (an `Err`-prefixed var typed `error` or untyped, with no literal value) and a documentation-only check for the module root and the tests-only `scenarios` directory (no declarations in non-test files). The scan of A10 covers `.github/`. Not mechanised, and left to review: the rule that only `*test` packages and `scenarios` import more than one parent package, that the §1 and §2 tables equal `DefaultPolicy` and `AGENTS.md` (`DefaultPolicy` is the encoding; change both in one change), that a new port is in the errors-and-results table, and the `math/rand` call ban. `internal/archtest` tests its rules in memory (`testing/fstest`) rather than from `testdata/` files: each rule has a violating overlay on a clean fixture module.

Checks A1 to A13 are lexical or AST-level, not type-checked, so each documents its
known blind spots (for example A4 cannot see a `time.Now` assigned to a variable
first) and relies on review for those.

### Bootstrap

`internal/archtest` is the first thing added after the kernel packages: the
first change that adds production code MUST add every check that can apply to the
code present, and every port it introduces MUST arrive with its doubles and
contract suite. After that these rules apply in full.

---

## 10. Anti-patterns (prohibited)

- **Anemic domain**: all logic in use cases with the domain reduced to data
  holders.
- **God object**: one `Game` or `Engine` type that knows everything.
- **Leaky ports**: a port that exposes a JSON map, a file path or an HTTP client.
- **Reaching across contexts**: importing a context not upstream of you in the
  context map.
- **Shared mutable state**: package-level variables, singletons, service
  locators, `init()` registration. A zero-value `telemetry.Noop` is stateless and
  exempt.
- **Hidden non-determinism**: random numbers or time read inside the domain.
- **Stringly-typed domain**: raw strings or ints where a value object or id type
  exists.
- **Errors or panics for outcomes**: returning an `error` or panicking for a
  failed check, missing content or illegal action instead of a result.
- **Display text in results or errors**: pre-formatted messages from the engine.
- **Global rule toggles**: optional rules as package flags or environment
  settings instead of options in the state.
- **Fat interfaces**: a port with more than about three methods, or an interface
  defined next to its implementation only so it can be mocked.
- **Consumer-shaped API**: any type or option that exists only because one
  consuming project needed it.

---

## Checklist before submitting

- [ ] Code sits in the right package; the allow-list and context map hold.
- [ ] No import of a downstream context, an unlisted package, or a `*test`
      package from production code.
- [ ] Domain types are immutable, validated on construction, with a documented
      zero value; domain code is pure and synchronous.
- [ ] Randomness, time and content arrive through ports.
- [ ] Expected outcomes are results; untrusted-input failures are `(T, error)`;
      port failures are wrapped with `srderr.WrapPort`; panics only in `MustX` and
      unreachable branches.
- [ ] Every fallible function's doc comment states its failure behaviour.
- [ ] `context.Context` is the first parameter where I/O or spans occur, and is
      never stored.
- [ ] Interfaces are small, defined where consumed; constructors return concrete
      types; adapters have compile-time assertions.
- [ ] Names match SRD 5.2.1 terminology and Go naming (no stutter).
- [ ] Every new port has doubles and a contract suite in its `*test` package.
- [ ] Each closed variant set documents all variants; no generic variant names.
- [ ] Nothing refers to a specific consuming project.
- [ ] Exported identifiers are documented; `CHANGELOG.md` is updated; breaking
      changes are marked.
- [ ] `go vet`, `golangci-lint run`, `go test -race -shuffle=on -cover ./...`
      and the architecture tests pass.

---

## Read receipt

RULE-TAG: `architecture-d95a5a39ea8f`

This tag proves you read this document in this session. Quote it verbatim as
required by `AGENTS.md`. Do not copy it anywhere else. Whoever edits this
document MUST replace the tag with a fresh random value.
