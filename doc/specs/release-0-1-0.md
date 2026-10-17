# Release 0.1.0

| | |
|---|---|
| Status | draft |
| Context | `package` |
| Roadmap phase | 5 |
| Depends on | `architecture-guardrails.md`, `content-licensing-and-notice.md`, `content-source-port.md`, `rules-glossary-coverage.md`, `content-schema-and-decoder.md`, `errors-and-results.md`, `observability-ports.md`, `resolve-d20-test-use-case.md`, `dice-and-rolls.md`, `spell-effect-data.md`, `monster-limited-use.md`, `class-wizard-evoker.md`, `content-classes-data.md`, `higher-level-start-equipment-and-trinkets.md`, `readying-a-spell.md`, `returning-to-life.md`, `telepathy.md`, `grid-positions-and-movement.md` |

## 1. Purpose and scope

Defines what must be true of the repository before the first public release,
version `v0.1.0`, is tagged:

- the public API surface of the published packages (every package that is not
  under `internal/`) and how it is reviewed and pinned;
- how consumers reach the bundled SRD content: only through the embedded JSON
  under `content/data/` and the bundled-content source that package exports
  (shared-types-and-ownership.md FR-030, FR-031; the adapter is defined by
  content-source-port.md FR-059; ownership is shared-types-and-ownership.md
  FR-047), never through a file path;
- the runnable examples (`Example*` functions and the cross-package scenarios),
  including the observability bridge examples (documented, not shipped);
- `README.md`, `CHANGELOG.md`, `go.mod` and the module zip a consumer downloads;
- the release checklist, as release gates a maintainer can run and record.

This spec adds no runtime behaviour and no game rule. Its requirements are
checked by architecture tests (`internal/archtest`) that read repository files,
by example and scenario tests run by `go test`, and by release checks run in CI
or by hand (section 8).

**Out of scope:**

- Pushing a tag. Publishing a Go module is pushing a version tag; the module
  proxy then caches it permanently, so it is irreversible and is a maintainer's
  decision (FR-063). The spec defines the gates, not the act.
- Any game rule or SRD content. What the release contains is whatever the
  other specs have implemented by then; this spec only requires the README to
  say so honestly (FR-045).
- Shipped bridges to OpenTelemetry, zap or any other SDK. They are documented
  only (`doc/rules/architecture.md` §5, `doc/rules/observability.md` §4). A
  separate companion module would be a different project.
- A documentation site and an API reference beyond godoc (pkg.go.dev renders it).
- A `/v2` module path. None exists at `v0.1.0` (`doc/rules/go.md` §9).
- SRD material that looks like core fifth-edition content but is not in SRD
  5.2.1. Nothing in this spec depends on any; the README coverage table
  (FR-045) lists what is not included.

## 2. Source

`n/a (package design)`. Derived from:

- `doc/rules/architecture.md` §5 (adapters, bridges not shipped), §7 (public
  API, documented exports, versioning, changelog), §9, §10; `doc/rules/go.md` §8
  to §10.
- `doc/rules/observability.md` §4 to §7 and §10; `doc/rules/testing.md` §3, §6,
  §7, §8, §10.
- `doc/specs/shared-types-and-ownership.md` (it wins over this spec): FR-013,
  FR-014, FR-017, FR-021, FR-030 to FR-045, FR-047, FR-048, FR-050 to FR-052,
  FR-056. Header and graph: rank 99 in table 4.11, may depend on any spec; no
  spec lists this one. It is outside the context map (it adds no code). Phase 5
  follows the type-phase rule (FR-014).
- This spec's own citations of other specs ("<file>.md FR-nnn") are checked
  under shared-types-and-ownership.md FR-052: each names an existing spec and a
  live id. The release manifest (FR-078) marks which specs are `in 0.1.0`.
  The specs added after the first draft are in the inventory of section 6
  (`class-wizard-evoker`, `content-classes-data`,
  `higher-level-start-equipment-and-trinkets`, `monster-limited-use`,
  `readying-a-spell`, `returning-to-life`, `spell-effect-data`, `spell-outcome-tables`, `telepathy`;
  shared-types-and-ownership.md FR-059, FR-060) and FR-098 pins their rows.
- `dice-and-rolls.md` (`dice.Roller`, `dice.SeededRoller`, `dicetest.FixedRoller`; FR-010, FR-011, FR-019, section 6) and `resolve-d20-test-use-case.md` (`d20.Resolver`; FR-019, FR-028): the types the quickstart and the smoke test use.
- `AGENTS.md` ("Hard constraints", "Licensing and source material", "Reporting
  a change", "Directory structure"); `doc/roadmap.md` Phase 5;
  `doc/research/sources.md` ("Structure of the document", for the coverage table).
- Go ecosystem conventions for modules, `go.mod`, `CHANGELOG.md` and `Example*`
  functions: tooling conventions, not SRD content; re-check pkg.go.dev's
  documentation when the release is prepared (Q4, settled).

## 3. Vocabulary

No SRD terms apply. Package terms used in this spec:

| Term | Type | Notes |
|---|---|---|
| Published package | package | Any package of the module outside `internal/`. Its exported identifiers are the compatibility surface (`doc/rules/go.md` §9). There is no facade and no re-export package: the root package `srd5e` is documentation only (FR-001). |
| Public API snapshot | file | `internal/archtest/testdata/public_api.golden`: a committed, line-oriented list of every exported symbol of every published package (section 6), regenerated only by an explicit `-update` flag. |
| Port | interface | A driven-port interface exported by a production package (for example `dice.Roller`, `telemetry.Logger`, `telemetry.Tracer`, `telemetry.Meter`, a content source, and a consumer-side port such as `d20.Roller`). |
| Double | type | A test implementation of a port, declared in a `*test` package (`dice/dicetest`, `telemetry/telemetrytest`) and imported by tests only (shared-types-and-ownership.md FR-038). A fake content source is a double, not an adapter. |
| Adapter | type | A generic implementation of a port or an external-data decoder, declared in a package of kind adapter (`telemetry/slogtelemetry`, `content/data`) or, for a stdlib-only one, beside its port (`dice.SeededRoller`). The zero-value no-ops are not adapters (FR-036). |
| Contract suite | test code | An exported `Run*Contract` function in a `*test` package that every adapter and double of a port must pass (shared-types-and-ownership.md FR-039). |
| Embedded content | files | The JSON under `content/data/`, embedded with `go:embed`, no generator step (shared-types-and-ownership.md FR-030). |
| Bridge | adapter | An implementation of `telemetry.Logger`, `telemetry.Tracer` or `telemetry.Meter` that forwards to a third-party module. Documented, never shipped. `telemetry/slogtelemetry` is the one shipped adapter because it needs the standard library only. |
| Moved-id row | table row | A requirement row of the form `\| FR-nnn \| Withdrawn: moved to <spec>.md (FR-mmm) \|` kept in the compact table at the end of section 4 (shared-types-and-ownership.md FR-062). Its id is never reused and never cited as a live id. |
| Release gate | check | A condition that must hold before tagging; automated where possible (FR-065 to FR-075, FR-097). |
| Module zip | artifact | What a consumer's `go get` downloads: every file of the module at the tag, apart from `vendor/`, nested modules and VCS metadata. Go has no ignore file for it (FR-061). |
| Release candidate commit (RC) | commit | The head commit of the branch `release/0.1.0`. After its PR merges, the merge commit on `main` is re-gated in RC mode; that merge commit is the one that is tagged `v0.1.0` (FR-076). |
| RC mode | test setting | Environment variable `SRD5E_RELEASE_CANDIDATE=1`, read once by the architecture test support code (FR-076). |
| Release manifest | file | `doc/release/0.1.0.md` (planned; the change that creates it also adds its `documentation_index.md` row): one row per spec in `doc/specs/` (except `_template.md` and this spec) with `Released in` (`in 0.1.0` or `deferred`) and `SRD sections` (the section names and page ranges of `doc/research/sources.md` the spec covers, or `none` for a spec with no SRD source) (FR-078). |

### Check phases

Each requirement belongs to one phase. Checks of phase **always** run on every
commit and, for a file that does not exist yet, pass vacuously unless the id is
listed under RC-only. Checks of phase **RC-only** assert the final state of the
release and run only in RC mode. **Post-release** says what replaces an
RC-only check once `v0.1.0` is tagged (RC mode is then never set).

| Phase | Requirements |
|---|---|
| always | FR-001 to FR-013, FR-015 to FR-017, FR-019 to FR-028, FR-030 to FR-041, FR-044, FR-048, FR-050, FR-059 to FR-061, FR-063, FR-066, FR-072, FR-076, FR-078, FR-080 to FR-086, FR-088 to FR-092, FR-096, FR-098 |
| RC-only | FR-014, FR-018, FR-029, FR-042, FR-043, FR-045 to FR-047, FR-049, FR-051 to FR-058, FR-062, FR-065, FR-067 to FR-071, FR-073, FR-075, FR-079, FR-087, FR-093 to FR-095, FR-097 |
| post-release | FR-014 and FR-056 expire; FR-051 to FR-055 are replaced by FR-077, which starts running once `v0.1.0` is tagged; FR-057 becomes an always check (the tag `v0.1.0` MUST keep pointing at the released commit); FR-074 is post-release only and runs once, on the follow-up PR |

FR-029 to FR-039 and FR-090 need `doc/telemetry-bridges.md` (the bridge
documentation): its existence is RC-only (FR-029); once it exists, the checks on
its content run always (FR-090 is listed under always for that reason).

## 4. Requirements

Stable ids. Never renumber. Sources name a rule file and section, or
"package design" for ecosystem conventions. Test names cite this file and the
FR id.

### 4.1 Public API surface

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | The module root MUST contain exactly one non-test Go file, `doc.go` (package `srd5e`: documentation only, no logic, no alias, no re-export). Any other non-test `.go` file directly in the root is a Violation. | architecture.md §7 | happy |
| FR-002 | No code block in `README.md` or the bridge documentation, and no file under `scenarios/`, MAY import a path under `.../internal/`. (The compiler already forbids it outside the module root tree; the scan covers prose.) | architecture.md §7 ("Consumers MUST NOT import from it") | unhappy (invalid input) |
| FR-003 | A published package MUST NOT re-export: no exported type alias (`type X = other.Y`), no exported `var` or `const` initialised from another package's identifier, and no dot import. One name has one home. | architecture.md §7 ("Export deliberately"); go.md §10; section 9, Q2 (decided) | unhappy (invalid input) |
| FR-004 | The public API snapshot MUST list every exported symbol of every published package, one line per symbol, sorted by import path and then by Name, in the line grammar of section 6 (FR-084), and MUST equal the symbols the packages actually export. | architecture.md §7 ("Anything exported is a compatibility promise") | happy |
| FR-005 | When a package exports a symbol missing from the snapshot, or the snapshot lists a symbol the package no longer exports, or a symbol moved between packages, the suite MUST fail and name every such symbol. | architecture.md §7 | unhappy (rule interaction) |
| FR-006 | Every exported identifier MUST have a doc comment starting with its name: `golangci-lint` (`revive` `exported`, `godot`) MUST report no issue and the `internal/archtest` rules `missing-doc` and `missing-package-doc` MUST report none. | architecture.md §7 | happy |
| FR-007 | Each published package MUST have a package comment (in `doc.go`) stating its purpose and listing, with godoc links (`[Name]`), the commands, results, events, ports and value objects it exports, grouped by context. A check MUST extract the bracketed links and require that (a) each names a symbol of that package in the snapshot (or of a package it documents), and (b) every symbol of kind `port`, `usecase`, `double` or `adapter` of that package appears. Other kinds are listed at the author's discretion. | architecture.md §4 ("Handoffs"), §7 | happy |
| FR-008 | A symbol MUST be exported from exactly one package (no re-export, FR-003); a name used by two published packages is permitted only when its package-qualified form is unambiguous and it is not a banned generic name (FR-091). | architecture.md §7 | unhappy (rule interaction) |
| FR-009 | A production package (kernel or context) MUST NOT export an adapter for a third-party facility or a test double, and MUST NOT import a `*test` package (`internal/archtest` `test-support-isolation`). The zero-value no-op implementations (`telemetry.NoopLogger`, `NoopTracer`, `NoopMeter`, `Noop`) are declared in the port package, not an adapter package, and are exported from it (shared-types-and-ownership.md FR-036). | architecture.md §7; testing.md §6 ("No test-only code in production packages") | unhappy (invalid input) |
| FR-010 | An adapter package (kind adapter in the `internal/archtest` policy: `telemetry/slogtelemetry`, `content/data`) MUST export only types that implement a port named in a snapshot `port` line (shown by a compile-time assertion `var _ Port = (*T)(nil)` that the check reads, with no cross-package type resolution), or that are listed in the `decoderTypes` constant of the architecture test code (external-data decoders), or in the `adapterSupportTypes` constant of the same code (adapter-package failure or helper types that implement no port; the first entry is the invalid-bundle error that content-source-port.md FR-061 requires the bundled-content package to export). | architecture.md §5, §7; content-source-port.md FR-061 | unhappy (invalid input) |
| FR-011 | Every port exported by a production package MUST have a double and a contract suite in the `*test` package beside the type that defines its vocabulary (`dice.Roller`: `dicetest.FixedRoller`, `dicetest.CallbackRoller`, `dicetest.RunRollerContract`; the telemetry ports: `telemetrytest`). The snapshot MUST record the mapping on the port's line (`double:`, one or more names: a port may have several doubles). A consumer-side port that restates a shared one (`d20.Roller`) names the shared double. A port without a double MUST fail the suite. The port set is derived from the source, not read from the snapshot (FR-084), so a port forgotten in the snapshot is still checked. Driven ports that are not obvious infrastructure, such as the content decoder, are covered by the same rule. | architecture.md §7 ("Every port MUST ship with a test double"); observability.md §4 | unhappy (missing content) |
| FR-012 | The transitive import closure of the published packages MUST use only the standard library, within the `internal/archtest` deny-list (no `os`, `os/exec`, `net`, `net/*`, `reflect`, `unsafe`, `syscall`, `log`, `io/ioutil`, `math/rand`, and `sync` and `sync/atomic`; `math/rand/v2` is allowed only in `dice`), and packages of this module. That check scans non-test files of kernel and context packages only; test-support (`*test`) and adapter packages are not scanned for standard-library imports and are covered by review (architecture.md A3). `go.mod` MUST have no `require` directive. The `*test` packages and test files MAY also import `testing`. | architecture.md §5, §9 bullet 1; AGENTS.md "Hard constraints" | unhappy (invalid input) |
| FR-013 | For every closed interface exported by a published package (an interface with an unexported marker method, documenting its variants), the snapshot MUST record its variants (`variants:`), so adding or removing a variant changes the snapshot (FR-081 requires each variant to be exported). Error types such as `*srderr.PortError` are concrete, not closed, and carry no `variants:` pin; the set of `error.type` values is covered by errors-and-results.md FR-006 and architecture-guardrails.md FR-085. | architecture.md §7 ("adding a variant to a closed variant set consumers switch over" is breaking) | unhappy (rule interaction) |
| FR-014 | In RC mode no exported identifier MAY carry a `// Deprecated:` paragraph: nothing has shipped yet that needs a deprecation cycle. The check expires at release (section 3, Check phases); afterwards deprecation follows go.md §9. | architecture.md §7; go.md §9 | unhappy (empty and minimal) |
| FR-015 | `scenarios/` MUST contain a public-API smoke test (package `scenarios_test`) that imports only published packages and calls every use case exported by a production package (kind `usecase`, FR-084) at least once, with test doubles only. Naming every use case is checked by FR-085. | testing.md §3 (scenario level) | happy |
| FR-016 | Every exported use case MUST be constructible and callable without supplying telemetry, and MUST give the same result with the zero-value `telemetry.Telemetry` as with a recording bundle. The smoke test (FR-015) MUST check both for each use case. | observability.md §4 ("No-op by default"), §9 | unhappy (determinism) |

### 4.2 Examples

Examples are `Example*` functions in `_test.go` files, so `go test` compiles and runs them and compares their
`// Output:` comment; there is no separate subprocess runner. Cross-package examples live in
`scenarios/example_test.go`, the others in the `example_test.go` of the package they document.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-017 | Every example file MUST be an external test file (`package x_test` or `scenarios_test`), declare only `Example*` functions and their helpers, and import only published packages and the standard library. | architecture.md §7; section 9, Q4 (decided) | happy |
| FR-018 | The examples MUST include at least: a quickstart (`Example` in the root `example_test.go`, package `srd5e_test`), a content-loading example, a character example, a combat example, a console telemetry bridge (`ExampleConsole...` in `telemetry/telemetrytest`'s or the scenarios' external tests), and a telemetry-inspection example (`ExampleRecorder` in `telemetry/telemetrytest`). An example MAY be omitted only when the release manifest (FR-078) marks every spec of its context `deferred` (content-loading: `content`; character: `character`; combat: `combat`). No example is required for `toolbox` (shared-types-and-ownership.md FR-045); the README coverage rows for the SRD sections those specs cover are then `not included` or `partial`. | roadmap.md Phase 5 ("Examples") | happy |
| FR-019 | The region of the root `example_test.go` between `// #region quickstart` and `// #endregion` (FR-044) MUST have at most 40 lines, and the quickstart MUST create the dice roller from a seeded `*rand.Rand` the example owns, run one D20 Test through the published `d20.Resolver`, and print the `d20.Outcome` (`success` or `failure`; shared-types-and-ownership.md FR-004) and the values the result carries (roll, modifiers, DC), not a pre-formatted sentence from the engine. | architecture.md §6 ("carry data, not display text") | happy |
| FR-020 | The content-loading example MUST decode a JSON document containing at least two deliberate problems and print every problem the decoder reports (path and kind, from the joined error), then look up an entry that exists and one that does not, handling both outcomes of the lookup. When the release bundles content, the example MUST also load it through the bundled-content source from `content/data` (the only consumer route to bundled content, which goes through the same decoder; shared-types-and-ownership.md FR-031; owner content-source-port.md) and look up one entry from it. | architecture.md §5 ("every problem found"), §6 | unhappy (invalid input) |
| FR-021 | Examples MUST handle each closed result interface with a type switch that has every variant and no `default` arm, and at least one example MUST branch on a `*srderr.PortError` with `errors.As` for a failing port, forced with a fake content source (the scriptable double from its `*test` package) scripted to fail. | architecture.md §3, §6 | unhappy (failure) |
| FR-022 | The character and combat examples MUST each show the domain events or state returned by the use case (state in, new state and events out) and MUST NOT mutate any input. | architecture.md §3 ("Immutability and state transitions"), §4 | happy |
| FR-023 | Every example MUST be deterministic: fixed seeds, no `math/rand` global, `time.Now`, `time.Since`, environment, network or filesystem access. The only way an example may observe time is through a clock injected as a `func() time.Time` argument (FR-041), which the example fills with `telemetrytest.Clock`. Two runs MUST print identical output (`go test -count=2`). | testing.md §6 ("Deterministic") | unhappy (determinism) |
| FR-024 | Each example's expected output MUST be its `// Output:` comment, so `go test` fails when the output differs. No subprocess and no golden file read is needed; the amendment of testing.md §6 and §10 that the earlier design required (Q15) is therefore not required. Longer expected texts live in `testdata/` golden files read by the example's test with `os.ReadFile` only in the repository-reading architecture checks and nowhere else. | testing.md §3, §6, §10 | happy |
| FR-025 | An example whose output differs MUST fail the test naming the example (`go test` reports `got:`/`want:`); an example without an `// Output:` comment is a Violation (it would be compiled but not run). | testing.md §4 | unhappy (failure) |
| FR-026 | Example files MUST be formatted and linted by the same `gofmt`, `go vet` and `golangci-lint run` as production code, with no `//nolint` comment, and MUST write output with `fmt.Println` inside `Example*` functions (which `go test` captures) or `fmt.Fprintln` to an injected writer; no example uses the `log` package. | architecture.md §9; AGENTS.md "Commands" | unhappy (invalid input) |
| FR-027 | Game content appearing in an example (names, numbers, rules text) MUST come from SRD 5.2.1 and carry a citation comment of the form `SRD 5.2.1, "<section>", p. <n>`; this citation check is review-only (no tool can tell game content from other literals). Examples MUST NOT name any consuming project or the hosting organisation (the module path in imports excepted). | AGENTS.md "Hard constraints", "Licensing and source material" | unhappy (invalid input) |
| FR-028 | The telemetry-inspection example MUST run a published use case with a `telemetrytest.Recorder` and print the spans, measurements and log records it captured by name, attribute keys and level. The use case is `d20.Resolver`, whose signals are owned by `resolve-d20-test-use-case.md` and which emits spans and metrics and no log (shared-types-and-ownership.md FR-040, FR-043), so the printed log section is empty for it and the example MUST say so rather than invent a record. | observability.md §9 | happy |

### 4.3 Observability bridge examples (documented, not shipped)

The bridge documentation is `doc/telemetry-bridges.md` (Markdown, relative links only).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-029 | `doc/telemetry-bridges.md` MUST exist and document three bridges: `telemetry.Logger` to a structured logger (for example `go.uber.org/zap`), `telemetry.Tracer` to OpenTelemetry, `telemetry.Meter` to OpenTelemetry, targeting the OpenTelemetry Go API and SDK (`go.opentelemetry.io/otel`, `.../otel/trace`, `.../otel/metric`). A bridge whose snippet the maintainer cannot compile against the then-current version MUST be labelled a sketch and carry `Verified against: none (sketch)`. It MUST state that the bridges are not shipped and why (the module has no third-party dependency, and a bridge would force one). | observability.md §4; architecture.md §5 | happy |
| FR-030 | Each bridge MUST be a complete type implementing every method of its port as defined by the port's current signature, and MUST carry a line `Verified against: <module> <version>` naming the third-party version the snippet was compiled against in a scratch module. The line MUST match `^Verified against: [a-z0-9./_-]+ v\d+\.\d+\.\d+(\S*)$` or `^Verified against: none \(sketch\)$` (Q12). A check MUST also verify that each snippet type declares every method name of its port, read from the port's source (FR-086). | observability.md §4 ("a consumer's adapter is a thin translation"); section 9, Q3, Q12 | happy |
| FR-031 | The documentation MUST include a table mapping each `telemetry.Level` (`LevelTrace`, `LevelDebug`, `LevelInfo`, `LevelWarn`, `LevelError`) to the target logger's level, to the OpenTelemetry severity and to the `slog` level that `slogtelemetry` uses. There is no fatal level; the target's fatal level MUST be marked as unused by srd5e. | observability.md §7 ("Levels") | happy |
| FR-032 | The `Logger` bridge MUST implement `Enabled` by asking the backend, and MUST call the `attrs` closure only after `Enabled(ctx, level)` returned true, and at most once. | observability.md §7 ("Cheap when disabled") | unhappy (collaborator: disabled level) |
| FR-033 | The `Tracer` bridge MUST end each span exactly once (`End` is idempotent). It MUST implement `RecordError` without ending the span or changing its status, and the use case sets `StatusError` itself (observability-ports.md FR-012, FR-017). The bridge MUST return a context that carries the new span and still carries the caller's values. | observability.md §6, §7 ("Errors: who records what") | unhappy (failure) |
| FR-034 | The `Tracer` bridge MUST map `telemetry.StatusUnset`, `StatusOK` and `StatusError` to the backend's statuses, and MUST implement `RecordError` without ending the span or changing its status. | observability.md §4, §6 | happy |
| FR-035 | The `Meter` bridge MUST map `Counter`, `Histogram` and `Gauge` to the backend's instruments, passing unit and description through from the `Descriptor`. Duration histograms carry unit `s` and record the seconds a `telemetry.Timer` produced with the consumer's own monotonic clock (observability-ports.md FR-029, FR-033), also when the timed operation fails. | observability.md §4 ("Timing"), §5 ("Units follow UCUM"; "Durations are always seconds") | unhappy (failure) |
| FR-036 | The bridges MUST pass attribute values to the backend with their native types (`string`, `bool`, `int64`, `float64`) and MUST NOT stringify them or add attributes of their own. | observability.md §5 ("Attributes"), §8 | unhappy (invalid input) |
| FR-037 | The documentation MUST state that production implementations MUST NOT panic from port methods and that srd5e does not recover from them, and that the recording doubles in `telemetrytest` collect violations on misuse by design and are for tests only. Source: observability-ports.md §9 (decisions) and its FR-010, FR-023, FR-056. It MUST also state that srd5e does not manage trace context: parent and child linkage is the bridge's job, carried by the `context.Context` that `Tracer.Start` returns. Source: observability.md §6. | observability-ports.md §9; observability.md §6 | unhappy (collaborator failure) |
| FR-038 | The documentation MUST show composition: build one `telemetry.Telemetry{Logger:, Tracer:, Meter:}` and pass it to each use case through its `WithTelemetry` option; and show a partial bundle (only a `Meter`) leaving the other ports as no-ops. It MUST NOT show a global, package-level or service-locator registration. | observability.md §4 ("Constructor injection only"); architecture.md §10 | happy |
| FR-039 | The documentation MUST point to the signal catalogue (`doc/observability/signals.md`) by a relative link, as the list of what to dashboard and alert on, and to the recording doubles for testing a bridge. Both files are in the module zip (FR-061), so the links work in the repository view and no file needs the repository URL (section 9, Q13). The documentation's "What to monitor" section is specified by FR-082. | observability.md §2 ("The signal catalogue"), §9 | happy |
| FR-040 | Non-test Go files and `go.mod` MUST NOT reference OpenTelemetry, zap, Sentry or any other SDK, and `go.mod` MUST have no `require`; test files and example files MUST NOT import one either. Only the Markdown documentation (and other prose) names them. The SDK denylist is architecture-guardrails.md FR-032, an ordinary named constant in `internal/archtest`; this requirement extends it to test and example files (guardrails FR-033 exempts them). The trademark list is stored as content-licensing-and-notice.md FR-031 specifies (FR-080). | observability.md §10; architecture.md §5; architecture-guardrails.md FR-032, FR-033 | unhappy (invalid input) |
| FR-041 | A console implementation of all three ports MUST exist as a type in an external test file (it is an example, not production code), use only standard-library packages, write to an injectable `io.Writer` (the example passes a `strings.Builder`), and be wired into a use case, so the shape of a bridge is compiled and run even though the third-party ones are not. Its `Tracer` and `Meter` take their clock as an injected `func() time.Time` constructor argument and read no other clock; the example passes `telemetrytest.Clock.Now`, and the expected output shows the fake durations, each span's status and any recorded error, and attribute values with their native types (FR-088). The type MUST pass `telemetrytest.RunLoggerContract`, `RunTracerContract` and `RunMeterContract` (the three suites of observability-ports.md FR-063; the generic checks only, as FR-088 lists them). | observability.md §4 | happy |

### 4.4 README

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-042 | `README.md` MUST NOT contain the phrases "scaffold only", "No functionality yet", "planned design", "does not exist yet", "Names are provisional" or "Status: early", nor any statement that describes a released feature as future. | architecture.md §7 | unhappy (state) |
| FR-043 | `README.md` MUST contain these H2 headings, exactly and in this order, after a one-paragraph summary directly under the H1: `## Installation` (`go get github.com/Whealinghold/srd5e-go@v0.1.0`, without a `replace` directive), `## Quickstart`, `## Packages` (a table of the published packages and what each is for), `## Observability`, `## Coverage`, `## Versioning`, `## Development`, `## Licence and content`. No other H2 heading may appear before `## Development`. | architecture.md §7 | happy |
| FR-044 | The README quickstart snippet MUST be byte-identical to the region of the root `example_test.go` between the markers `// #region quickstart` and `// #endregion`, and a test MUST fail when they differ. | package design (README code must not rot) | unhappy (state) |
| FR-045 | The coverage table MUST have one row per top-level section in `doc/research/sources.md` ("Structure of the document") with a status of `included`, `partial` or `not included` and, for `included` or `partial`, the package that provides it. The specs covering a row are those whose `SRD sections` cell in the release manifest (FR-078) names the section. A row MUST be `included` only if every covering spec is `in 0.1.0` and has Status `implemented`; it MUST be `not included` if every covering spec is `deferred` or none covers it; otherwise `partial`. Each `deferred` spec MUST be named in the table (in a Notes column). | AGENTS.md "Licensing and source material"; doc/research/sources.md | unhappy (missing content) |
| FR-046 | The README MUST state the versioning policy: before 1.0.0 a breaking change bumps the minor version, anything else the patch version, and metric, span and attribute names are part of the public contract. | architecture.md §7, observability.md §5 | happy |
| FR-047 | The README Observability section MUST describe the real API, name the example functions and the file (`doc/telemetry-bridges.md`) and state that bridges to third-party modules are not shipped. | observability.md §4 | happy |
| FR-048 | Every link in every `.md` file that is part of the release (`README.md`, `CHANGELOG.md`, `NOTICE.md`, `doc/telemetry-bridges.md`, `doc/observability/signals.md`) MUST be either an absolute URL or a relative path that resolves to a file in the repository. An absolute URL MUST NOT point at the repository: no file may name the hosting organisation except through the module path and `AGENTS.md` (AGENTS.md "Hard constraints"; FR-072), and pkg.go.dev links the repository from the module path itself. The change that lands this requirement MUST also rewrite any link that violates it. | package design; AGENTS.md "Hard constraints" | unhappy (invalid input) |
| FR-049 | The README licence section MUST state that the code is MIT-licensed, that the game content comes from SRD 5.2.1 under CC-BY-4.0, and MUST link `NOTICE.md`. While content is bundled the section also contains the attribution statement itself (content-licensing-and-notice.md FR-028), because pkg.go.dev renders the README. It MAY say the module is "compatible with fifth edition" or "5E compatible" and MUST NOT use any trademarked name or logo as a brand or imply endorsement. | AGENTS.md "Trademarks and attribution"; content-licensing-and-notice.md FR-028 | unhappy (invalid input) |
| FR-050 | The README MUST state the minimum Go version, and the stated version MUST equal the `go` directive in `go.mod` (architecture check; the statement is mandatory in RC mode and, when present, always checked). The claim that the published packages have no platform restriction is separate (FR-087). | package design | happy |

### 4.5 Changelog

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-051 | In RC mode, `CHANGELOG.md` MUST contain, from the top: an empty `## Unreleased` heading, then a heading beginning `## 0.1.0` with the release date. | architecture.md §7 | happy |
| FR-052 | In the release candidate commit, every entry under `Unreleased` on `main` MUST be moved into the 0.1.0 section; none MAY be dropped. The reviewer compares the 0.1.0 section with `Unreleased` on `main` in the PR diff (review); the architecture check asserts only that `Unreleased` is empty in RC mode. | architecture.md §7 ("Every user-visible change gets an entry") | unhappy (state) |
| FR-053 | The 0.1.0 section MUST group entries under `Added` only, one bullet per published package group (kernel packages, each context including `toolbox` when `in 0.1.0`, the `*test` packages, the adapter packages), and one each for the examples, the bundled content (when any) and the signal catalogue, and MUST NOT carry the **BREAKING** marker, because there is no earlier release to break. | architecture.md §7 | unhappy (empty and minimal) |
| FR-054 | The 0.1.0 section MUST list the published packages, name the signals in `doc/observability/signals.md` as a stable contract from this version, and say what game content is bundled and where the attribution is (`NOTICE.md`). | observability.md §5; AGENTS.md "Licensing" | happy |
| FR-055 | Every bullet in the 0.1.0 section MUST describe something present in the public API snapshot, a signal in the catalogue, or a file in the repository. The check: every backticked token that looks like an exported Go identifier (matches `^[A-Z][A-Za-z0-9]*$`, optionally package-qualified as `pkg.Name`) MUST be a symbol in the snapshot. Backticked tokens that are file paths (contain `/` or end in `.md`/`.go`), import paths, a package name from the context map (`dice`, `ability`, `modifier`, `srderr`, `telemetry`, `content`, `d20`, `character`, `combat`, `toolbox`), or a signal name in the catalogue are exempt from the identifier check; each exempt class has its own fixture. A bullet naming an identifier absent from the snapshot is a Violation. | architecture.md §7 | unhappy (invalid input) |

### 4.6 `go.mod` and the module zip

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-056 | In RC mode `go.mod` MUST declare the module path `github.com/Whealinghold/srd5e-go` with no `/vN` suffix (the `v0`/`v1` import path, go.md §9) and a `go` directive equal to the documented minimum (FR-050), and MUST contain no `toolchain`, `replace`, `exclude` or `retract` directive. The version itself is the git tag (FR-057, FR-074), not a file. After release FR-077 replaces this check. | architecture.md §7; section 9, Q9 (decided) | happy |
| FR-057 | In RC mode the tag `v0.1.0` MUST NOT exist (`git tag --list v0.1.0` is empty): the tag is the publication, and the module proxy caches a published version forever. Creating it is part of the maintainer's act after the gates (FR-063). Post-release this becomes an always check: the tag exists and still points at the released commit. | package design | happy |
| FR-058 | The first sentence of the root `doc.go` package comment (the summary pkg.go.dev shows) MUST be 60 to 180 characters, MUST state what the module is without any trademarked product name used as a brand, and MUST mention neither a UI framework nor any consumer. The trademark list is the one in `internal/archtest` (FR-080). | AGENTS.md "Trademarks and attribution"; package design | unhappy (boundary) |
| FR-059 | The hosting organisation MUST appear only in the module path (`go.mod`, import paths, the README install line) and `AGENTS.md`; the `internal/archtest` rule `consumer-name` enforces it. `LICENSE` MUST be the standard, unmodified MIT text so pkg.go.dev's licence detector recognises it, and the module MUST NOT declare a second licence outside `NOTICE.md`'s covered paths. There is no `repository`, `issue_tracker` or `topics` field in Go modules. | AGENTS.md "Hard constraints"; package design; section 9, Q13 | unhappy (boundary) |
| FR-060 | The dependency rule is owned by architecture-guardrails.md FR-083 and is not restated here. This release adds only: `go.mod` has no `require`, no `replace` and no `go.work` file is committed (`go.work` is in `.gitignore`), and `go.sum` is absent or empty (`go mod tidy -diff` exits 0). | AGENTS.md "Hard constraints"; architecture-guardrails.md FR-083 | unhappy (invalid input) |
| FR-061 | A Go module zip contains every file of the module apart from `vendor/`, nested modules and VCS metadata, and there is no ignore-file equivalent, so what is committed is what ships. Therefore: the repository MUST contain no nested `go.mod`, no `vendor/`, no credentials, no generated or scratch output (the committed `.gitignore` MUST list `*.test`, `*.out`, `/bin/`, `/vendor/`, `go.work`, `go.work.sum`, `coverage/`, `coverage.txt`, `coverage.html`, `*.coverprofile` and `.gomu/`), and `doc/research/` notes that hold SRD text MUST be Markdown only and carry the attribution (shared-types-and-ownership.md FR-033; they ship in the zip and `NOTICE.md` covers them). `AGENTS.md`, `CLAUDE.md`, `.github/` and `doc/` ship as ordinary repository files; the denylist scan (FR-072) covers them. Bundled content ships only as `content/data/*.json` (section 9, Q14; content-licensing-and-notice.md FR-019). | package design | unhappy (invalid input) |
| FR-062 | The repository root MUST contain `LICENSE`, `NOTICE.md`, `README.md`, `CHANGELOG.md`, `go.mod`, `.golangci.yml` and `doc.go`, plus `doc/observability/signals.md`; the packages (which include any bundled game content under `content/data/`) and the examples are the rest. A missing `NOTICE.md` is a Violation. Its exact file set is FR-067. | AGENTS.md "The notice"; content-licensing-and-notice.md | unhappy (missing content) |
| FR-063 | An agent MUST NOT create or push a version tag, create a GitHub release, request the module from the public proxy (for example `go list -m github.com/Whealinghold/srd5e-go@v0.1.0`, which makes the version permanent in the proxy cache), or hold a push credential for tags. An agent MAY edit `CHANGELOG.md` and `go.mod`'s `go` line only in the release candidate commit and only when the maintainer asks explicitly. The maintainer performs the tag after the gates in 4.7 pass. The review row covers the process; CI workflows MUST NOT contain a tag or release step (FR-083). | package design (a published version cannot be deleted) | unhappy (state) |

### 4.7 Release checklist (release gates)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-065 | The full CI check in `AGENTS.md` ("Commands") MUST pass on the release candidate commit: `gofmt` (no diff), `go vet ./...`, `golangci-lint run ./...`, `go test -race -shuffle=on -coverprofile=coverage.txt ./...`, `go mod tidy -diff` and the architecture tests (`make check`). | testing.md §10 | happy |
| FR-066 | CI MUST cross-compile every published package for `GOOS=js GOARCH=wasm` and `GOOS=wasip1 GOARCH=wasm` and for `GOOS=linux GOARCH=amd64`, `GOOS=darwin GOARCH=arm64` and `GOOS=windows GOARCH=amd64` (`go build ./...`), and MUST run `go vet` and the test suite on the lowest Go version allowed by the `go` directive as well as on the latest stable release. The web test run that content-schema-and-decoder.md FR-099 asks this spec to list is met by the `GOOS=js GOARCH=wasm go vet ./content/...` and `go build ./content/...` steps: the decoder and the bundled source use no platform facility, and the tests run natively. A failure of any step fails the release gate. | AGENTS.md "Hard constraints" (pure Go); package design | unhappy (boundary) |
| FR-067 | `go mod tidy -diff`, `go mod verify` and `go build ./...` MUST exit 0, and a scratch consumer module outside the repository (a `replace` to the checkout) MUST compile a program importing every published package (`go list ./...` minus `internal/...`). The file set of the module zip is `git ls-files` (FR-061) and MUST contain the FR-062 root files and MUST contain no nested `go.mod`, `vendor/` or `.env` path. The step runs in RC mode only. | package design | happy |
| FR-068 | Mutation testing, as defined in `doc/rules/testing.md` §8, MUST be run over the kernel packages (`dice`, `ability`, `modifier`, `srderr`, `telemetry`) and each context package (every context of architecture.md §2, `toolbox` included), with the exact `gomu` target paths listed in the release record (FR-075), threshold 80. `gomu` runs only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc` (minimum 1), never directly on the host. The pass condition is score >= 80 and every surviving mutant either fixed or recorded with a reason it is equivalent; a survivor that is not equivalent blocks the release even when the score is above 80. | testing.md §8 | unhappy (failure) |
| FR-069 | Statement coverage of the kernel packages, the contexts and their use cases MUST be 100% or each uncovered statement MUST be listed with a reason in the release record (the standing floor is 90% per package, testing.md §6). | testing.md §7 | unhappy (boundary) |
| FR-070 | `doc/observability/signals.md` MUST contain none of the placeholder phrases `No signals yet`, `TODO`, `TBD`, and none of its rows MAY say the Go constant is a "planned target" (it states which constants exist). The catalogue is not otherwise re-checked here: that signal names in code equal the catalogue is architecture-guardrails.md FR-042 and FR-043. Empty or placeholder Purpose and Suggested monitoring cells are already forbidden by architecture-guardrails.md FR-045 and FR-046, and an empty Unit is allowed for spans and log events (observability.md §2: Unit "where applicable"). Attribute values and names follow FR-093. | observability.md §2, §10; architecture-guardrails.md FR-042, FR-043, FR-045, FR-046 | unhappy (state) |
| FR-071 | No spec marked `in 0.1.0` in the release manifest (FR-078) MAY have Status `draft` on the release candidate commit; `deferred` specs may stay `draft` and MUST be named in the README coverage table (FR-045). Every FR id in the `in 0.1.0` specs MUST have a test (the coverage rule is architecture-guardrails.md FR-055), except moved-id rows (architecture-guardrails.md FR-087), or be listed in the release record with the reason it has none. | testing.md §5 ("every requirement SHOULD be covered"); architecture-guardrails.md FR-055, FR-087 | unhappy (state) |
| FR-072 | The architecture suite MUST pass with the denylist scan covering every text file of the repository (Go files, `doc/`, `README.md`, `CHANGELOG.md`, `NOTICE.md`, `go.mod`) with the exemptions of architecture-guardrails.md FR-026: the module path, `AGENTS.md` and `go.sum`. Because FR-048 forbids repository URLs, a compliant README, CHANGELOG, bridge document and shipped catalogue pass this scan. | architecture.md §9; architecture-guardrails.md FR-026 | unhappy (invalid input) |
| FR-073 | `pkgsite` (`golang.org/x/pkgsite/cmd/pkgsite`, run outside the module so it adds no dependency) or the equivalent MUST be run on the checkout, and `go doc -all` MUST render for every published package. Every documentation, licence or rendering issue it reports MUST be listed with its cause and a decision (fix or accept) in the release record. | package design | unhappy (failure) |
| FR-074 | After the maintainer tags, the tag MUST be `v0.1.0` and point at the re-gated merge commit, and `AGENTS.md`, `README.md` and the roadmap MUST describe the module as released. | package design; section 9, Q8 | happy |
| FR-075 | The release record MUST be the PR description of the release candidate commit, with the `Testing` and `Observability` entries of `AGENTS.md` ("Reporting a change"), and MUST contain: Go versions, the `gomu` target paths, mutation score and survivors per target, coverage exceptions, the pkgsite result, the file list, and the README coverage table. The release PR is `release/0.1.0` into `main`; the merge commit is re-gated in RC mode (FR-076). | AGENTS.md "Reporting a change" | happy |

### 4.8 Added requirements

Added after review and reconciliation; ids continue the sequence.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-076 | RC mode MUST be the environment variable `SRD5E_RELEASE_CANDIDATE=1`, read in one place in the architecture test support code. CI MUST set it in a job that runs on pull requests whose head branch is `release/0.1.0`, and on the merge commit on `main` through a manually dispatched workflow; ordinary runs MUST NOT set it. The commit tagged is that re-gated merge commit. RC-only requirements (section 3, Check phases) MUST be skipped when the variable is unset, and MUST run when it is set. | package design | unhappy (state) |
| FR-077 | After release, the latest tag `vX.Y.Z` MUST equal the version in the first non-`Unreleased` heading of `CHANGELOG.md`. | architecture.md §7 | happy |
| FR-078 | `doc/release/0.1.0.md` MUST contain a table with one row per file in `doc/specs/` (excluding `_template.md` and this spec) giving `Released in` (`in 0.1.0` or `deferred`) and `SRD sections` (names and pages as in `doc/research/sources.md`, or `none` for a spec with no SRD source, such as shared-types-and-ownership.md, errors-and-results.md and architecture-guardrails.md). A spec's context (used by FR-018) is read from the Context row of its header (architecture-guardrails.md FR-094), not from the manifest. A spec without a row, or with a `Released in` value other than those two, or an empty `SRD sections` cell, MUST fail the suite. | package design; section 9, Q1 | unhappy (missing content) |
| FR-079 | The root `doc.go` summary sentence MUST be written without any UI-framework clause, for example "Package srd5e is a pure-Go implementation of fifth-edition rules from the SRD 5.2.1." (length and trademark rules of FR-058 apply). | AGENTS.md "Trademarks"; package design | unhappy (state) |
| FR-080 | The trademark denylist and the consumer and hosting-organisation denylist used by this spec's checks MUST live in `internal/archtest` (`names.go`, spelled in parts so the file does not match its own rule) and MUST NOT appear in plain text anywhere else in the repository; their storage and matching are content-licensing-and-notice.md FR-031 and architecture-guardrails.md FR-026 and FR-068, not restated here. The SDK denylist (architecture-guardrails.md FR-032) is an ordinary named constant in the same package, is matched only as FR-040 says, and its names MAY appear in prose. | AGENTS.md "Hard constraints"; architecture-guardrails.md FR-032 | unhappy (invalid input) |
| FR-081 | Every variant of a closed interface exported by a published package MUST itself be exported, or be listed with a reason in the `exportExceptions` constant of the architecture test code. | architecture.md §4, §6, §7; FR-013 of this spec | unhappy (rule interaction) |
| FR-082 | The "What to monitor" section of the bridge documentation MUST name the two generic cases (any non-zero use case error rate; a latency regression against the consumer's own baseline) and link the catalogue for thresholds, without repeating them. | observability.md §2 | happy |
| FR-083 | No workflow file under `.github/workflows/` MAY contain `git tag`, `git push` with tags, `gh release create` or a module fetch of this module's own version from the public proxy. | package design | unhappy (invalid input) |
| FR-084 | Snapshot kinds are derived by parsing the source (`go/parser` and `go/ast`, standard library, no type checker). A `port` is a driven port exactly as architecture-guardrails.md FR-082 defines it, including the handle exemption (observability-ports.md FR-065: `Span`, `Counter`, `Histogram`, `Gauge`); this spec defines no second port list. A handle is recorded with its Go kind and needs no double (FR-011). A `usecase` is a struct type with a constructor `NewX` and an exported method taking `context.Context` first (or documented as the use case's entry point), in a context package; a `double` is a symbol exported by a `*test` package whose declaration carries a compile-time assertion against a port; an `adapter` is a symbol declared in an adapter package or asserted against a port beside its port; any other symbol has its Go kind (`struct`, `interface`, `closed`, `type`, `func`, `const`, `var`). `func`, `const` and `var` (a top-level function, constant or variable, such as `d20.DefaultCriticalRange`) are listed by the same parse. The kind recorded in the snapshot MUST equal the derived kind. The line grammar is in section 6. | architecture.md §7; architecture-guardrails.md FR-082 | unhappy (invalid input) |
| FR-085 | A check MUST fail when the `scenarios/` public-API smoke test source does not name a use case that has a `usecase` line in the snapshot. | testing.md §3 | unhappy (missing content) |
| FR-086 | A check MUST read each port's method names from its source and fail when a bridge snippet type in `doc/telemetry-bridges.md` omits one. | observability.md §4 | unhappy (rule interaction) |
| FR-087 | The README MAY state that the published packages have no platform restriction only once the FR-066 job is green on the release candidate; the release record (FR-075) holds that evidence. | package design | happy |
| FR-088 | The console implementation of FR-041 MUST be run against `telemetrytest.RunLoggerContract`, `RunTracerContract` and `RunMeterContract` (observability-ports.md FR-063), limited to the checks that apply to any implementation: observability-ports.md FR-006, FR-011, FR-025, FR-029, FR-030 and the generic parts of FR-012 (span ended, error recorded without ending) and FR-017. Assertions that read a recording double's state read the console implementation's injected writer instead (FR-041). FR-019 and FR-033 of that spec are `NoopTracer` and `Timer` rules (record nothing, read no clock) and are NOT run against the console implementation. These cover this spec's FR-032, FR-033 (end once, record without ending) and FR-035 for the console implementation. FR-034 and FR-036, and the status and recorded-error part of FR-033, are not contract-level; for the console implementation they are asserted on the example output (FR-041); the Markdown snippets stay review. | testing.md §3; observability-ports.md | unhappy (collaborator failure) |
| FR-089 | No production package MAY export a contract suite, a check function standing in for one, or anything that imports `testing`; contract suites exist only in `*test` packages (`Run*Contract` in `dicetest` and `telemetrytest`). | shared-types-and-ownership.md FR-039; architecture.md §7 | unhappy (rule interaction) |
| FR-090 | The "Testing a bridge" section of `doc/telemetry-bridges.md` MUST state that the port contract suites live in the `*test` packages (`telemetrytest.RunLoggerContract` and its siblings) because they import `testing`, and MUST say how a consumer reuses them: call the suite for the port from their own `_test.go` file with a factory for their adapter. It MUST NOT contain a repository URL (FR-048). | shared-types-and-ownership.md FR-039; testing.md §10 | happy |
| FR-091 | No symbol in the snapshot MAY carry a banned generic name (the list is owned by architecture-guardrails.md FR-077 and is not repeated here), and no Name MAY appear on two snapshot lines of the same package. The uniqueness check applies to the Name field of each line, the symbol being declared; names inside `variants:` and `double:` clauses are references and are excluded. The check reuses the name scan of architecture-guardrails.md rather than redefining it. | shared-types-and-ownership.md FR-017, FR-021; architecture-guardrails.md FR-077; architecture.md §9 | unhappy (name collision) |
| FR-092 | Every `usecase` line in the snapshot MUST have a matching result type: a `closed` interface (or a documented result struct) named in the use case's entry-point signature, and the entry point MUST return that type, with an `error` as its last result where it can fail (for example `d20.Resolver.Resolve` returns `(Result, error)`). | shared-types-and-ownership.md FR-017; architecture.md §4 | unhappy (missing content) |
| FR-093 | In `doc/observability/signals.md`: the values of the shared attributes (`srd5e.outcome`, `srd5e.d20_test.kind`, `srd5e.d20_test.roll_mode`, `srd5e.d20_test.critical_hit`) and of `error.type` are checked in code and catalogue by architecture-guardrails.md FR-085 and are not restated here. Unique to this release: no returned failure result carries `error.type` in the catalogue (errors-and-results.md FR-043); class features use the single counter `srd5e.character.feature_used` with attributes `srd5e.class` and `srd5e.feature` and no per-feature metric name exists; and no log row exists for the d20 test. | shared-types-and-ownership.md FR-041 to FR-043; architecture-guardrails.md FR-085; observability.md §5 | unhappy (state) |
| FR-094 | The content format MUST have one `schemaVersion` integer. At release, `content-schema-and-decoder.md`'s revision table MUST show every revision landed as that single version, no content spec MAY claim another version number, the JSON under `content/data/` MUST carry it, and the decoder MUST reject any other value. | shared-types-and-ownership.md FR-034 | unhappy (version mismatch) |
| FR-095 | Every embedded JSON file under `content/data/` MUST decode with zero problems through the bundled-content source and the real content decoder (no second loader); with `go:embed` of the files themselves there is no generator, so a drift check is replaced by this decode test and by `go test ./content/...` on the release candidate commit. | shared-types-and-ownership.md FR-030 to FR-032 | unhappy (drift) |
| FR-096 | Every directory directly under the module root that holds a Go package other than the kernel packages (`dice`, `ability`, `modifier`, `srderr`, `telemetry`) MUST be a context in the table of `doc/rules/architecture.md` §2 (`content`, `d20`, `character`, `combat`, `toolbox`) or an allowed support directory (`internal`, `scenarios`); the snapshot kinds and the re-export rules (FR-003, FR-084) apply to `toolbox` like any other context. Each `toolbox` use case in the snapshot is called by the FR-015 smoke test. The smoke test MUST record telemetry per use case and assert that each call emits only that use case's own catalogue signals, because a use case never calls another use case (shared-types-and-ownership.md FR-044). | shared-types-and-ownership.md FR-044, FR-045; architecture.md §2 | unhappy (rule interaction) |
| FR-097 | In RC mode, `doc/specs/rules-glossary-coverage.md` MUST have no unit in state `gap` whose interim or planned owner spec has Status `accepted` or `implemented` (an exported rule); a release MUST NOT be tagged while one exists. The release record (FR-075) states the `gap` count published in that spec's section 4.6. | rules-glossary-coverage.md FR-009, Q-17; roadmap.md Phase 5 | unhappy (state) |
| FR-098 | The release manifest (FR-078) MUST have a row for each spec added after the first draft, with the context its header gives (the inventory of section 6): `class-wizard-evoker.md` (`character`), `content-classes-data.md` (`content`), `grid-positions-and-movement.md` (`combat`; SRD pp. 13 to 15), `higher-level-start-equipment-and-trinkets.md` (`character`), `monster-limited-use.md` (`combat`), `readying-a-spell.md` (`combat`), `returning-to-life.md` (`combat`), `spell-effect-data.md` (`content`), `spell-outcome-tables.md` (`combat`) and `telepathy.md` (`combat`), and the ten further rows of the inventory (`wish-spell.md`, `magic-item-effects.md`, `spell-scroll-use.md`, `item-use-effects.md`, `monster-engulf-and-swallow.md`, `monster-stat-block-actions.md`, `monster-traits.md`, `monster-spellcasting.md`, `persistent-spell-zones.md`, `summoned-and-companion-creatures.md`; contexts as the inventory gives). When one is `in 0.1.0`, its exported types, ports and use cases are in the snapshot under the rules of FR-003 to FR-013 and FR-084, its signals are in the catalogue (FR-070, FR-093), and the FR-015 smoke test calls its use cases (a spec whose section 6 states no use case adds none); when `deferred`, it is named in the README coverage table (FR-045). | shared-types-and-ownership.md FR-025, FR-059, FR-060 | unhappy (missing content) |

### 4.9 Withdrawn ids

| Id | Status |
|---|---|
| FR-064 | Withdrawn: moved to content-licensing-and-notice.md (FR-002, FR-011, FR-012); the covered-path list is architecture-guardrails.md FR-079 (shared-types-and-ownership.md FR-050) |

## 5. Model

No domain layer: this spec describes a release, not rules.

- **Public API snapshot:** a text file kept in step with the code by test, so
  a public API change is a visible, reviewed diff (format in section 6).
- **Release gates (4.7):** conditions on a commit, not code; each has an owner
  (CI job, architecture test, or maintainer) and leaves evidence in the
  release record.
- **Checks as pure functions:** architecture-level rows of section 8 follow
  architecture-guardrails.md (a `fs.FS` of sources in, violations out, with a
  violating and a clean `fstest.MapFS` fixture, as `internal/archtest` does);
  scenario rows run through the public API.
- **Invariants:** the published packages are the whole public surface and none
  re-exports another (FR-001, FR-003); a symbol has one home (FR-008) and a
  compatibility promise (FR-004, FR-005); no file claims something the module
  lacks (FR-048, FR-055, FR-062).
- **Failures:** all checks report violations as test failures; no result type
  or error is added to the module.

## 6. Interface

No new Go API. This spec fixes shapes of repository artefacts.

### Published packages

| Package kind | Packages (by import path suffix) | Exports | Must not export |
|---|---|---|---|
| Documentation | `.` (`srd5e`) | nothing (FR-001) | any identifier |
| Kernel | `dice`, `ability`, `modifier`, `srderr`, `telemetry` | value objects, ids, ports, the zero-value no-ops, the error vocabulary (`srderr.PortError` is concrete, not closed; errors-and-results.md FR-001) | adapters for third parties, doubles |
| Context | `d20`, and later `content`, `character`, `combat`, `toolbox` | use cases, commands, results (closed interfaces), events, value objects and ids of the context | adapters, doubles, re-exports of kernel types |
| Adapter | `telemetry/slogtelemetry`, `content/data` (bundled-content source, JSON decoders) | generic adapters, platform-neutral | ports, doubles, no-ops, anything needing `os` |
| Test support | `dice/dicetest`, `telemetry/telemetrytest`, and the `*test` package of each later port | a double for every port, recording records, `Run*Contract` suites | production adapters; imported by tests only |

### Inventory of specs added after the first draft

Each contributes to its context package only what its own section 6 exports; the
snapshot (FR-004) is the record, this table is the checklist behind FR-098.

| Spec | Context | Rank | Contribution to the published packages |
|---|---|---|---|
| `spell-effect-data.md` | `content` | 7 | Spell effect data types and queries; no use case or port. |
| `content-classes-data.md` | `content` | 7 | Data only: twelve class JSON files under `content/data/`, loaded through the bundled-content source (FR-095); no type, port or use case. |
| `class-wizard-evoker.md` | `character` | 18 | Evoker use cases (`AddEvocationSavantSpells`, `ResolvePotentCantrip`, and so on) with their result types. |
| `higher-level-start-equipment-and-trinkets.md` | `character` | 15 | `StartAtHigherLevel`, `RollHigherLevelStartingMoney`, `RollTrinket` and their results. |
| `monster-limited-use.md` | `combat` | 26 | Limited-use state, outcomes and events, and the use case `ApplyMonsterTurnStart` with its result. |
| `readying-a-spell.md` | `combat` | 26 | Pure functions and value objects; the use cases that call them are an amendment to `combat-action-resolution.md`, not this spec's. |
| `grid-positions-and-movement.md` | `combat` | 26 | Grid value objects and pure `GridRules` functions; no use case or port; callers as for `readying-a-spell.md`. |
| `telepathy.md` | `combat` | 23 | Pure functions and `TelepathicContactState`; callers as for `readying-a-spell.md`. |
| `returning-to-life.md` | `combat` | 27 | A pure function and `ReviveCreatureResult`; callers as for `readying-a-spell.md`. |
| `spell-outcome-tables.md` | `combat` | 28 | The pure tables and the use case `ResolveSpellOutcomeTable` with its closed command and result types. |
| `wish-spell.md` | `character` | 16 | What its own section 6 exports; use cases with their results. |
| `magic-item-effects.md` | `character` | 14 | What its own section 6 exports; use cases with their results. |
| `spell-scroll-use.md` | `character` | 15 | What its own section 6 exports; use cases with their results. |
| `item-use-effects.md` | `combat` | 25 | What its own section 6 exports; use cases with their results. |
| `monster-engulf-and-swallow.md` | `combat` | 24 | What its own section 6 exports; use cases with their results. |
| `monster-stat-block-actions.md` | `combat` | 27 | What its own section 6 exports; use cases with their results. |
| `monster-traits.md` | `combat` | 27 | What its own section 6 exports; use cases with their results. |
| `monster-spellcasting.md` | `combat` | 27 | What its own section 6 exports; use cases with their results. |
| `persistent-spell-zones.md` | `combat` | 28 | What its own section 6 exports; use cases with their results. |
| `summoned-and-companion-creatures.md` | `combat` | 31 | What its own section 6 exports; use cases with their results. |

### Public API snapshot (`internal/archtest/testdata/public_api.golden`)

One line per symbol, sorted by import path suffix and then by Name (byte order). Regenerated by
`go test ./internal/archtest -update`; the flag is defined in the test and never implicit
(`doc/rules/go.md` §8). Grammar (FR-084):

```
line    = pkg SP kind SP Name [SP "variants:" SP Name {"," SP Name}]
          [SP "double:" SP Name {"," SP Name}]
pkg     = import path suffix, "." for the root
kind    = "port" | "usecase" | "double" | "adapter"
        | "struct" | "interface" | "closed" | "type"
        | "func" | "const" | "var"
```

`variants:` appears only on `closed` lines and lists the variants (FR-013);
`double:` appears only on `port` lines (FR-011), with at least one name. There
are no free-standing mapping lines. A symbol with a role kind (`port`,
`usecase`, `double`, `adapter`) uses it in place of its Go kind. Methods are not
listed; a change to an exported method is covered by the type's compile-time
assertions and the package tests.

```
content/data adapter BundledSource
d20 closed Result variants: ForcedFailure, RolledAttack, RolledCheck
d20 struct ForcedFailure
d20 port Roller double: dicetest.FixedRoller, dicetest.CallbackRoller
d20 struct RolledAttack
d20 struct RolledCheck
d20 usecase Resolver
dice port Roller double: dicetest.FixedRoller, dicetest.CallbackRoller
dice/dicetest double FixedRoller
```

(Names above illustrate the format and are not a list of what exists.)
Regenerating the file is a deliberate act (a documented command in the test's
failure message), and the diff is reviewed with the CHANGELOG entry.

### Example and documentation layout

```
example_test.go                   # package srd5e_test: Example (quickstart) with region markers for the README
<pkg>/example_test.go             # Example* functions of that package, // Output: verified
scenarios/example_test.go         # cross-package examples (content loading, character, combat)
telemetry/telemetrytest/example_test.go   # ExampleRecorder: telemetry inspection
doc/telemetry-bridges.md          # OpenTelemetry and structured-logger bridges, documented only
doc/release/0.1.0.md              # the release manifest (FR-078)
```

`// #region quickstart` and `// #endregion` in the root `example_test.go` delimit
the text the README embeds (FR-044).

### Module zip layout

Everything committed ships (FR-061): the root files of FR-062, the packages
(including `content/data/*.json`), `scenarios/`, `internal/` (not importable), and
`doc/` and the repository files. Nothing is excluded by an ignore file because Go
has none; the module therefore keeps non-distributable material out of the
repository instead.

### Bridge documentation outline

1. Why bridges are not shipped. 2. Composition of `telemetry.Telemetry`. 3. `Logger`
to a structured logger. 4. `Tracer` to OpenTelemetry. 5. `Meter` to
OpenTelemetry. 6. Testing a bridge with the recording doubles and the
`Run*Contract` suites (FR-090). 7. What to monitor (FR-082).

## 7. Observability

Silent by design. This spec adds no use case, port call or adapter, so it adds
no signal and `doc/observability/signals.md` gets no row
(`doc/rules/observability.md` §3). The seven questions:

1. **What can go wrong?** Undocumented or placeholder signals (catalogue
   drift); an example or bridge document showing a pattern that breaks the
   observability rules (package-level logger, panicking bridge, eager attributes); a
   bridge that stringifies attributes or ignores the closure; the catalogue is missing from the repository (FR-061, FR-062 and FR-067 guard this).
2. **How would an operator know?** At release, by the gates: FR-070 and FR-093
   with architecture-guardrails.md FR-042, FR-043, FR-085; FR-032 to FR-039 and
   FR-088 (bridge rules); FR-016 (same result with and without telemetry). After release, by the owning specs' error counters and duration histograms.
3. **What would someone need to diagnose it?** The release record (FR-075), the
   catalogue, the bridge documentation and the telemetry-inspection example.
4. **What should be counted or timed?** Nothing new. Mutation score, coverage
   and the pkgsite result are one-off evidence, not runtime signals.
5. **Is existing instrumentation still correct?** FR-070 is the check: the
   release makes every signal name a compatibility promise
   (`doc/rules/architecture.md` §7), so a stale entry is fixed before tagging.
6. **What should a consumer monitor?** FR-082: any non-zero use case error rate
   and a latency regression, with the catalogue linked for the rest.
7. **Is anything now noisy?** No. The console implementation prints to its
   injected writer by design, as a demonstration.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none | | | | |

## 8. Test plan

Levels: unit, use case, contract, scenario, architecture, plus **release
check** (a CI step or maintainer action, evidenced in the release record) and
**review** (judged by a human). Architecture rows have a violating and a clean
fixture each (an `fstest.MapFS`, as `internal/archtest` does) unless the note
says otherwise.

| Id | Level | Notes |
|---|---|---|
| FR-001 | architecture | Lists root `*.go`; fixtures with an extra file and a missing `doc.go`. |
| FR-002 | architecture | Scans the README and bridge-doc code blocks and `scenarios/` for `internal/` imports. |
| FR-003 to FR-005 | architecture | Alias, dot-import and initialised-from-other-package fixtures; snapshot comparison: added, removed and moved symbol, every one named. |
| FR-006 | release check | `golangci-lint run` in CI; also asserted by reading `.golangci.yml` for `revive` and `godot`, and by the `missing-doc` rules. |
| FR-007 | architecture | Bracketed links from each package comment; fixtures: phantom name, missing port, clean doc. |
| FR-008, FR-009 | architecture | A name exported twice; an adapter or a double exported from a production package; a zero-value no-op in `telemetry` passes. |
| FR-010 | architecture | Fixtures: a non-adapter symbol in an adapter package; a type with a port assertion passes; a no-op or a double fails; the invalid-bundle error (in `adapterSupportTypes`) passes, absent from the constant fails. |
| FR-011 | architecture | Port list derived from source (FR-084); fixtures: a port lacking a double, a port missing from the snapshot, a handle (`Span`) with no double (passes). |
| FR-012 | architecture | Import-closure walk shared with architecture-guardrails.md and `internal/archtest` (`stdlib-denied`, `third-party-import`, `go-mod-require`); fixtures importing `os` and a third-party module. |
| FR-013, FR-014 | architecture | A closed interface gaining a variant changes the snapshot; a variant left unexported (FR-081). `// Deprecated:` scan; FR-014 RC-only. |
| FR-015, FR-016 | scenario | One test per use case, public API only (imports asserted by FR-002); table-driven zero `Telemetry` versus recording bundle, equal results. |
| FR-017 | architecture | Scans example files for package name, declarations and imports. |
| FR-018 | architecture | Example function existence; an omitted example must match a context whose specs (context from each header, FR-078) are all `deferred`, and the README rows (FR-045). |
| FR-019 | architecture | Line count of the region and the seed literal; output built from result data is by review. |
| FR-020 | scenario | `// Output:` shows both decoder problems, both lookup outcomes and, when bundled, the bundled-source lookup. |
| FR-021 | architecture | AST scan: no `default:` in a type switch over a closed result in example files (conservative); `errors.As` with `*srderr.PortError` present; a fake content source scripted to fail is used. |
| FR-022 | scenario | `// Output:` shows returned state and events; immutability is covered by the owning specs' unit tests. |
| FR-023 | scenario | `go test -count=2` gives identical output; architecture scan bans the calls except a clock via the injected `func() time.Time`. |
| FR-024, FR-025 | scenario + architecture | Every `Example*` has an `// Output:` comment (fixture: one without is reported by name); no subprocess runner exists. |
| FR-026 | release check | Full CI check; an architecture check scans example files for `//nolint` and `log.`. |
| FR-027 | architecture | Consumer-name denylist scan; the citation-comment check is review-only. |
| FR-028 | scenario | `// Output:` lists the recorded spans and measurements of `d20.Resolver` and states no log record was emitted (shared-types-and-ownership.md FR-043). |
| FR-029, FR-037 to FR-039, FR-082 | architecture | File exists with the required headings and "not shipped" statement; keyword scans for the FR-037 statements, `telemetry.Telemetry{` composition without a package-level registration, the "What to monitor" section; the catalogue link resolves in the repository. |
| FR-030 | review | Compile-in-scratch-module is a maintainer action (Q3); the architecture check asserts a `Verified against:` line per bridge by the FR-030 regex and (FR-086) every port method name in each snippet type. |
| FR-031 | architecture | The table has a row for every `telemetry.Level` constant read from the source. |
| FR-032, FR-035 | contract + review | Logger contract suite for `Enabled` and the lazy closure; Meter suite (observability-ports.md FR-025, FR-029, FR-033) for instruments, units and the `Timer` with an injected clock. Snippets reviewed. |
| FR-033 | contract + scenario + review | Tracer suite for end-once and context propagation; error status and recorded error in the console example output (FR-041). Snippets reviewed. |
| FR-034, FR-036 | scenario + review | Status mapping, `RecordError` not ending the span and native attribute types are asserted on the console example output (FR-041), not as contract rows. Snippets reviewed. |
| FR-040 | architecture | Import and `go.mod` scan with the denylist from `internal/archtest` (FR-080), extended to test and example files. |
| FR-041 | scenario + contract | Example run with its fake clock, output shows fake durations; the type passes the three contract suites (FR-088). |
| FR-042 to FR-047 | architecture | README: phrase scan; the eight exact H2 headings in order; region equals README block; coverage table rows compared with `doc/research/sources.md`, expected status derived from manifest and spec `Status` lines; versioning keywords; the example function names and bridge file appear and exist. |
| FR-048 | architecture | Link extraction over every release `.md`; fixtures: a link to a missing file, the catalogue link (allowed), an absolute non-repository link, a link to an existing file. |
| FR-049, FR-050 | architecture | Keyword scan and `NOTICE.md` link check (trademark wording by review); stated Go version equals the `go` directive, fixture with a mismatch. |
| FR-051, FR-053 to FR-055 | architecture | Heading order; group headings, bullet subjects and the marker; keyword scan (packages, "signal", `NOTICE.md`); identifier tokens against the snapshot with a fixture per exempt class (path, import path, package name, signal name) and an identifier absent from the snapshot. RC-only for FR-051. |
| FR-052 | review | Nothing-dropped needs a diff of the release PR against `Unreleased` on `main`; the architecture check asserts `Unreleased` is empty in RC mode. |
| FR-056, FR-057 | architecture | Reads `go.mod` and `git tag --list`; RC mode only (FR-076). |
| FR-058, FR-079 | architecture | Both RC-only (the summary sentence is rewritten before the release). Boundary fixtures at 59, 60, 180, 181 characters; trademark and UI-framework scan using `internal/archtest`; fixture with an old sentence. |
| FR-059 | architecture | Fixtures: the organisation in a README link (violation), in the module path (clean), in `AGENTS.md` (clean); a modified `LICENSE` text. |
| FR-060 | architecture | `require`, `replace` in `go.mod` (violations); a committed `go.work`; a lone `go` directive passes. Allowlist fixtures are guardrails FR-083's. |
| FR-061 | architecture | `.gitignore` entries against the required set; a nested `go.mod` and a `vendor/` path are Violations; `doc/research/` file that is not Markdown (content-licensing-and-notice.md FR-019). |
| FR-062, FR-067 | release check | The scratch-consumer, `go mod verify` and `go build ./...` steps in `.github/workflows/ci.yml` (RC mode) fail on any non-zero exit and `git ls-files` is compared with the required list; the architecture check covers file existence. |
| FR-063 | review + architecture | Process rule is review of the release PR; the workflow scan is FR-083. CI never holds a tag-push credential. |
| FR-065, FR-066 | release check | CI; matrix, cross-compile steps and `go vet` for the js/wasm content packages, which the architecture check asserts the workflow contains. |
| FR-068, FR-069 | release check | Docker command from testing.md §8 over the FR-068 paths, `--cpus` and `--workers` at a quarter of `nproc`; `coverage.txt` filtered to the FR-069 packages; results in the release record. |
| FR-070 | architecture | Reuses guardrails FR-045, FR-046; adds the placeholder-phrase scan. Fixture: a span row with an empty Unit passes. |
| FR-071 | architecture | Manifest and `Status` of each `in 0.1.0` spec; coverage by a scan of requirement-id comments in tests; moved-id rows skipped (fixture); exceptions in the release record. |
| FR-072 | architecture | The guardrails scan; this requirement pins its scope at release. |
| FR-073 | release check | Manual `pkgsite` run; evidence in the release record. |
| FR-074, FR-075 | review | Post-tag, checked on the follow-up PR; the PR template carries the headings, completeness is judged by the reviewer. |
| FR-076, FR-077 | architecture | Flag unset skips RC-only checks, set runs them; a workflow scan asserts the RC job sets it on `release/0.1.0` and ordinary jobs do not; latest tag equal to and different from the top changelog heading (FR-077 is post-release: it does not run before `v0.1.0` is tagged). |
| FR-078 | architecture | Lists `doc/specs/`, compares with the manifest; fixtures: missing row, bad value, a package-design spec with `none` (clean), an empty `SRD sections` cell. |
| FR-080, FR-083 | architecture | Plain-text trademark and consumer names outside `internal/archtest` (SDK names not scanned here); fixtures `git tag` and `go test` steps. |
| FR-081, FR-084 to FR-086 | architecture | Unexported variant of an exported closed interface, and a listed exception; fixtures per kind, a hand-edited kind that disagrees with the parser, an interface in a production package missing from the snapshot, `Span` (a handle, no double demanded) beside `Tracer` (a port), a `func`, `const` and `var` line; a use case the smoke test does not name; a snippet type missing one port method. |
| FR-087 | release check | CI matrix result (FR-066) in the release record; the README claim is by review against it. |
| FR-088 | contract | The console implementation runs the generic Logger, Tracer and Meter checks while writing to a `strings.Builder`; a fixture lists the no-op-only rows (observability-ports.md FR-019, FR-033) as excluded. |
| FR-089, FR-090 | architecture | A production package exporting a `Run*Contract` symbol or importing `testing`; keyword scan of "Testing a bridge" for `telemetrytest.Run` and `*test` packages, and for repository URLs (FR-048). |
| FR-091, FR-092 | architecture | A snapshot with a banned name, one with a Name on two lines; clean: a `closed` type with `variants:` and a port with `double:` names, each with its own line; a `usecase` line without its result type; an entry point returning another type. |
| FR-093 | architecture | Parses the catalogue; fixtures: a per-feature metric name, an `error.type` on a returned-failure row, a d20 test log row. Value sets are guardrails FR-085's. |
| FR-094 | architecture + unit | Revision table and `schemaVersion` of the `content/data/` JSON (the decoder rejection test is content-schema-and-decoder.md's). RC-only. |
| FR-095 | architecture + scenario | Every embedded file loads through the bundled-content source with zero problems. RC-only. |
| FR-096 | architecture + scenario | Root directories against the context map, fixture with an unknown package directory; smoke test with a recording `Telemetry` asserts per-use-case signals; fixture: two use cases that share a pure domain service each emit only their own catalogue signals, with no nested or double-counted span or counter. |
| FR-097 | architecture | RC-only. Unit table of rules-glossary-coverage.md and owner `Status`; fixtures: `gap` with `implemented` owner (Violation), `gap` with `draft` owner (clean), no `gap` (clean). The record's `gap` count equals section 4.6 (review). |
| FR-098 | architecture | Manifest lacking any inventory row (Violation, one per missing spec); all present (clean); a row whose context differs from the spec header; an `in 0.1.0` spec whose use case has no snapshot line. |

- **Doubles and contract suites required:** the recording doubles and the fixed
  `dice.Roller`, content source and decoder doubles from the `*test` packages.
  No new port; the console implementation implements Logger, Tracer and Meter
  and passes their existing contract suites (FR-088).
- **Mutation-testing scope:** the check functions are test support code with
  their own fixtures; the production packages gain no code from this spec.
  Release-wide mutation testing is FR-068 (Docker, CPU cap as above; never run
  on the host).
- **Deliberately not mechanically covered** (the form architecture-guardrails.md
  reads for its coverage rule):

  Deliberately not covered: FR-052 - nothing-dropped needs a PR diff review
  Deliberately not covered: FR-074 - a post-tag action, checked on the follow-up PR
  Deliberately not covered: FR-075 - completeness of the release record is judged by the reviewer

## 9. Open questions and decisions

No open questions.

Settled (decided; the requirement named is the decision):

- Q1 release content is whatever is `in 0.1.0` in the manifest, not full SRD coverage: FR-078, FR-071, FR-045.
- Q2 no re-exports, no facade: FR-003. Q3 third-party snippets are not compiled in this repository, compiled once in a scratch module: FR-030.
- Q4 the package example is the root `Example`; pkgsite confirms how it renders: FR-018, FR-073.
- Q5 contract suites are exported from the `*test` packages, not from production packages (shared-types-and-ownership.md FR-039; content-source-port.md Q2): FR-089, FR-090.
- Q6 Go has no ignore file for the module zip, so tests ship with the module and nothing needs excluding: FR-061, FR-067.
- Q7 a maintainer tags; agents prepare the release candidate only: FR-063. Q8 tag `vX.Y.Z` on the merge commit: FR-074.
- Q9 the version is the git tag, set only after the release candidate is re-gated: FR-056, FR-076. Q10 the minimum Go version is the `go` directive (1.24), lower bound tested: FR-066.
- Q11 pkgsite need not report a clean result; every loss is explained: FR-073.
- Q12 bridges target OpenTelemetry Go (tracing and metrics) and a structured logger; versions go in each `Verified against:` line at release, sketch label only if the scratch compile fails: FR-029, FR-030.
- Q13 the catalogue is in the repository and linked relatively, no repository URL: FR-039, FR-048, FR-059.
- Q14 only `content/data/*.json` carries content; `go:embed` of the files themselves is the only route (shared-types-and-ownership.md FR-030, FR-031; content-licensing-and-notice.md FR-019): FR-061.
- Q15 a subprocess example runner would have needed an amendment of testing.md §6 and §10; in Go `Example*` functions are run by `go test`, so no amendment is needed (FR-024, FR-025). The repository-reading architecture checks use `fs.FS`/`os.ReadFile` in `internal/archtest` only.
- The console implementation may write output: the rule against printing scopes production code (go.md §10); examples write through their injected writer or `fmt.Println` inside `Example*` (FR-026, FR-041).
- A compiled console implementation exists so the port shape is compiled and run in CI while the third-party snippets stay thin Markdown (FR-041).

## 10. Changelog impact

No `CHANGELOG.md` entry of its own: the work this spec describes is the
release. The release candidate commit moves everything under `Unreleased` into:

```
## 0.1.0 - <release date>

### Added
- Kernel packages `dice`, `ability`, `modifier`, `srderr`, `telemetry`: one bullet each.
- Test support `dice/dicetest`, `telemetry/telemetrytest` and the optional adapter
  `telemetry/slogtelemetry`: one bullet each.
- One bullet per implemented context (`d20`, `content`, `character`,
  `combat`, `toolbox`); a `deferred` context (FR-078) has none.
- Bundled SRD 5.2.1 content (only when any is bundled), attribution in `NOTICE.md`.
- Examples (`Example*` functions), including bridge documentation (not shipped as code).
- Telemetry signals in `doc/observability/signals.md`, stable from this version.
```

There is no "First public release." line (FR-055). Not **BREAKING**: there is no
earlier release. From 0.1.0 onward, a change to a snapshot entry, a closed
interface's variant set, a rule result for the same input, or a signal or
attribute name is breaking and bumps the minor version (`doc/rules/architecture.md`
§7, `doc/rules/go.md` §9).
