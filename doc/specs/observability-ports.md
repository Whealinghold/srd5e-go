# Observability ports and `telemetrytest`

| | |
|---|---|
| Status | partially implemented |
| Context | `telemetry` (kernel package), `telemetry/telemetrytest`, `telemetry/slogtelemetry` |
| Roadmap phase | 0 |
| Depends on | `errors-and-results.md` |

## 1. Purpose and scope

Defines the three observability ports (`telemetry.Logger`, `telemetry.Tracer`,
`telemetry.Meter`), the `telemetry.Telemetry` bundle, their zero-value no-op
defaults (`NoopLogger`, `NoopTracer`, `NoopMeter`, `Noop`; in the port package,
not an adapters package: `shared-types-and-ownership.md` FR-036), the
`telemetry.Timer` that stands in for clock access, the recording test doubles and
contract suites (package `telemetry/telemetrytest`: `shared-types-and-ownership.md`
FR-038, FR-039) and the one optional adapter, `telemetry/slogtelemetry`. Every use
case takes these ports (`doc/rules/observability.md` §4), so this is the contract
the module is instrumented against. Type phase 0.

**Dependencies (prose; the header lists tokens only, FR-056 of
`shared-types-and-ownership.md`).** `shared-types-and-ownership.md` is binding
(FR-014, FR-036 to FR-039, FR-041, FR-051). `errors-and-results.md` supplies the
rules that a returned failure result carries no `error.type` (FR-043, cited by
FR-059), that `srderr.Guard` is the only `recover` in the module (FR-030, FR-047;
relied on by FR-012) and `srderr.ErrorType`, the source of the `error.type`
value. `dice-and-rolls.md` FR-064 has the consumer inject the seeded
`*rand.Rand` (why FR-053 has no carve-out) and specifies the `dicetest` doubles.
`architecture-guardrails.md` is a prose reference only
(`shared-types-and-ownership.md` FR-061): its checks are the ones `internal/archtest`
runs (FR-050, FR-051, FR-053, FR-060 point to them). This spec supplies the shared
attribute keys (FR-052) to it. `telemetry` is a kernel package: it imports the
standard library only and no other package of the module (FR-051).

**Rule-file amendments (applied).** `shared-types-and-ownership.md` FR-051
accepted the rule text this spec relies on; the rule files carry it with fresh
tags (FR-059): the ports return no errors and take `context.Context` first
(`observability.md` §4, `architecture.md` §4); closure-only log attributes and no
`error.type` on a returned failure (`observability.md` §7); zero-value no-op
instances are exempt from the package-level logger, tracer or meter ban
(`observability.md` §4, §10, §11; `architecture.md` §10).

**Out of scope:**

- Any game rule. This spec has no SRD content.
- Which signals a use case emits (each context's specs and
  `doc/observability/signals.md`). The ports emit nothing (section 7).
- Bridges to OpenTelemetry, zap, Sentry or any SDK: not shipped
  (`observability.md` §4); examples go in the README and `Example*` functions.
  The standard-library `log/slog` adapter (`slogtelemetry`) is the only adapter.
- Trace-context propagation across goroutines or process boundaries: the
  consumer adapter's job, carried by the `context.Context` that `Tracer.Start`
  returns (`observability.md` §6).
- The `dicetest` doubles and doubles for other ports (such as the fake content
  source: `shared-types-and-ownership.md` FR-038); only the shape of
  `telemetrytest` is fixed here.
- The value sets of shared attribute keys: see `shared-types-and-ownership.md`
  FR-041 and `observability.md` §5. This spec defines only the key constants
  (FR-052).
- A helper asserting "same result with no-op and recording telemetry":
  deferred (section 9).

## 2. Source

`n/a (package design)`. Derived from `shared-types-and-ownership.md` (FR-013,
FR-014, FR-036 to FR-039, FR-041, FR-051; ids of other specs are written with
the file name) and:

- `doc/rules/observability.md` §1, §4 (ports, timing), §5 (naming, attributes,
  units), §6 (traces), §7 (levels, lazy attributes, errors), §9, §10, §11.
- `doc/rules/architecture.md` §1, §3 (no clock), §4 (ports, context), §6, §7
  (entry points, every port ships a double), §9, §10.
- `doc/rules/go.md` §3 (error idioms), §4 (context rules), §7 (interfaces).
- `doc/rules/testing.md` §3, §4 (state errors, boundaries, invalid input, empty
  and minimal, immutability), §6, §7 (contract suites), §10.

## 3. Vocabulary

No SRD terms apply. Package terms:

| Term | Type | Notes |
|---|---|---|
| `Logger`, `Tracer`, `Meter` | driven ports (interfaces in `telemetry`) | Emit a log event; start a named span; create instruments. |
| `Level` | `int` enum | `LevelTrace`, `LevelDebug`, `LevelInfo`, `LevelWarn`, `LevelError`; `String()` and `Valid()`. |
| `Span` | handle (interface) | Returned by `Tracer.Start`; carries attributes, status, recorded errors. |
| `Status` | `int` enum | `StatusUnset` (zero), `StatusOK`, `StatusError`; `String()`. |
| `Counter`, `Histogram`, `Gauge` | handles (interfaces) | Created by `Meter` from a `Descriptor`. |
| `Descriptor` | comparable struct | `Name`, `Unit` (UCUM), `Description`. |
| `Attr` | comparable struct, unexported fields | One key/value; the value is a `string`, `int64`, `float64` or `bool` by construction (`String`, `Int`, `Int64`, `Float`, `Bool`, `Err`). The zero `Attr` is invalid (`Valid`). |
| `Telemetry` | value type | `{Logger, Tracer, Meter}`; a nil field means no-op; `WithDefaults()`. |
| `Timer` | value type | `StartTimer(clock, histogram)`; `Stop(ctx, attrs...)` records elapsed seconds. Stands in for `Meter` timing: the port reads no clock. |
| Shared keys | constants | `AttrOutcome`, `AttrErrorType`, `OutcomeSuccess`, `OutcomeFailure` (FR-052). |
| No-op | implementation | `NoopLogger`, `NoopTracer`, `NoopMeter`, `NoopSpan`, `Noop`: empty structs, silent, stateless, read no clock, zero value ready to use. Production types, not adapters (`shared-types-and-ownership.md` FR-036). |
| No-op handles | implementation | The counter, histogram and gauge returned by `NoopMeter` (FR-019, FR-020, FR-064): one unexported type, `noopInstrument`, not exported. |
| Recording double | test double | `telemetrytest.Logger`, `telemetrytest.Tracer`, `telemetrytest.Meter`, `telemetrytest.Recorder`, `telemetrytest.Clock`; zero values ready to use (`shared-types-and-ownership.md` FR-038). |
| Record types | values | `telemetrytest.LogRecord`, `SpanRecord`, `Measurement`, `InstrumentInfo`, `Kind` (`KindCounter`, `KindHistogram`, `KindGauge`). |
| Violation | error | Misuse a double collects instead of panicking (FR-010, FR-023, FR-024, FR-027, FR-038, FR-039); `telemetrytest.Violator`, `RequireNoViolations`. |

## 4. Requirements

One behaviour each. Source is the rule-file section named in the row. Test names
cite `doc/specs/observability-ports.md` and the FR id. Go replaces the callback-style
`trace`/`time` wrappers with `Tracer.Start` and `Timer` used with `defer`, so no
body callback exists and the async variants have no counterpart; those ids stay
reserved and are listed in 4.7.

### 4.1 Logger

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `telemetry.Level` MUST have exactly the values `LevelTrace`, `LevelDebug`, `LevelInfo`, `LevelWarn`, `LevelError`, in ascending severity (the zero value is `LevelTrace`). It MUST NOT have a `fatal` value. `String()` MUST return the lowercase name and `"invalid"` outside the set; `Valid()` MUST report membership. | observability.md §7, "Levels" | happy |
| FR-002 | `Logger` MUST expose `Enabled(ctx context.Context, level Level) bool` and `Log(ctx context.Context, level Level, msg string, attrs func() []Attr)`, and nothing else. Attributes are a closure so they are built only when needed. The closure MAY be nil. An error is reported with the `Err(err)` attribute (key `AttrErrorMessage`, `exception.message`), which is the invalid zero `Attr` for a nil error; `Log` has no error or stack parameter. | observability.md §4, §7 "Cheap when disabled" | happy |
| FR-003 | `NoopLogger.Enabled` MUST return `false` for every `Level`, including a level outside the defined set. | observability.md §4, "No-op by default" | happy |
| FR-004 | `NoopLogger.Log` MUST NOT invoke the `attrs` closure and MUST have no observable effect, for every level (valid or not), a nil closure and a cancelled context. | observability.md §1.5, §7 | happy |
| FR-005 | Reserved. Go has no extension methods, so there are no level shortcuts; callers pass a `Level` constant to `Log`. A use case MUST guard expensive work with `Enabled` only when building `attrs` is not enough. | observability.md §7 | n/a (see 4.7) |
| FR-006 | Every `Logger` implementation MUST NOT invoke the `attrs` closure when `Enabled(ctx, level)` is `false` for that call, and MUST invoke it at most once otherwise. `Enabled` MUST be monotonic: if a level is enabled, every higher level is. | observability.md §7 "Cheap when disabled" | unhappy (collaborator: disabled level) |
| FR-007 | `telemetrytest.Logger` MUST store one `LogRecord` (`Level`, `Message`, `Attrs`) per enabled call, in call order, and return them as copies from `Records()`; `RecordsAt(level)` MUST return those at exactly that level. | observability.md §9 | happy |
| FR-008 | `telemetrytest.Logger` MUST record events at its minimum level and above; the minimum is `LevelTrace` for the zero value, set by `NewLogger(min)` or `SetMinLevel`. A call below the minimum MUST NOT be recorded; a call exactly at the minimum MUST be recorded; `Enabled` MUST agree with both. | observability.md §7 | unhappy (boundary) |
| FR-009 | `telemetrytest.Logger` MUST invoke the `attrs` closure exactly once per recorded call and store a copy of the result. When no closure is given, the record's `Attrs` MUST be empty. | observability.md §7 | unhappy (empty and minimal) |
| FR-010 | An `Attr` cannot hold a value other than `string`, `int64`, `float64` or `bool`: the constructors are the only way to build one, so an unsupported value is unrepresentable. Each recording double MUST drop an invalid `Attr` (empty key or no value, such as the zero `Attr`) and collect an `ErrInvalidAttr` violation naming the key; it MUST NOT panic. | observability.md §4 "No vendor types", §5 "Attributes" | unhappy (invalid input) |

### 4.2 Tracer and Span

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-011 | `Tracer.Start(ctx context.Context, name string, attrs ...Attr) (context.Context, Span)` MUST return a non-nil `Span` and a context that still carries the caller's values, and MUST tolerate a cancelled context. The span is the unit of work between `Start` and `End`; the caller defers `End` immediately. | observability.md §4, §6 | happy |
| FR-012 | A use case whose operation returns an error MUST record it once on its span: `span.RecordError(err)` and `span.SetStatus(telemetry.StatusError)`, with `End` deferred so the span also ends on a panic. The `Tracer` never swallows, wraps or recovers anything: a panic from the module's own code propagates, and a port panic has already been turned into a `*srderr.PortError` by `srderr.Guard` (`errors-and-results.md` FR-030, FR-047), which is the only `recover` in the module. A failure result (not an error) leaves the status unset unless the spec says otherwise (FR-017). | observability.md §6, §7 "Errors: who records what"; errors-and-results.md FR-047, FR-030 | unhappy (failure) |
| FR-013 | Reserved: no async wrapper. Go has no `Future`; a use case that does I/O takes `context.Context` and returns synchronously, so `Start` and a deferred `End` already span it correctly. | observability.md §6; architecture.md §4 | n/a (see 4.7) |
| FR-014 | Reserved: see FR-013 and FR-012. | observability.md §6 | n/a (see 4.7) |
| FR-015 | `Span.SetAttributes(attrs ...Attr)` MUST add the attributes or overwrite an earlier value for a key. Attributes passed to `Start` MUST be present from the start. | observability.md §6 | happy |
| FR-016 | A new span's status MUST be `StatusUnset`. `Span.SetStatus` MUST change it, and the last call MUST win. | observability.md §6 | happy |
| FR-017 | `Span.RecordError(err)` MUST append the error to the span in call order and MUST NOT change the span's status or end it. A nil `err` MUST be ignored. A failure result (not an error) is recorded by the use case calling `SetStatus(StatusError)`. | observability.md §6, §7 | happy |
| FR-018 | If a span's status is `StatusOK` or `StatusUnset` and the operation then fails, the final status MUST be `StatusError` (the use case's `SetStatus(StatusError)` is the last call, FR-016). | observability.md §6 | unhappy (rule interaction) |
| FR-019 | `NoopTracer.Start` MUST return the context unchanged and a `NoopSpan`, read no clock and validate nothing. | observability.md §4 | happy |
| FR-020 | A `NoopSpan` MUST accept `SetAttributes`, `SetStatus`, `RecordError` and `End` at any time, including after `End`, without panicking and without changing any recorded state. | observability.md §1.5 | unhappy (state) |
| FR-021 | `telemetrytest.Tracer` MUST store one `SpanRecord` per span, in span start order (a parent precedes its children): `ID` (1-based start order), `ParentID`, `Name`, `Attrs` (start attributes plus later ones, first-set order), `Status`, `Errors` (in call order) and `Ended`. `Spans()` MUST return snapshots; `SpansNamed(name)` those with that name. No duration is recorded (it would need a clock). | observability.md §9 | happy |
| FR-022 | A span started from the context returned by another span of the same tracer MUST have that span's `ID` as `ParentID`; a top-level span MUST have `ParentID` 0. Linkage follows the context, so nested and concurrent use are both correct. | observability.md §6, §7 ("inner spans still record status") | unhappy (rule interaction) |
| FR-023 | A write (`SetAttributes`, `SetStatus`, `RecordError`) to a span after `End` MUST be ignored and collected as an `ErrSpanEnded` violation; `End` itself MUST be idempotent. | testing.md §4 "State errors" | unhappy (state) |
| FR-024 | `telemetrytest.Tracer` MUST reject a span name for which `telemetry.ValidName` is false (`srd5e` prefix, at least three dot-separated segments, each segment starting with a lowercase letter and holding lowercase letters, digits or underscores) by collecting an `ErrInvalidName` violation and returning the context and a `NoopSpan` without a record. | observability.md §5 "Naming"; architecture-guardrails.md FR-038 | unhappy (invalid input) |

### 4.3 Meter and instruments

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-025 | `Meter` MUST offer `Counter(Descriptor)`, `Histogram(Descriptor)` and `Gauge(Descriptor)`, each returning an instrument whose `Descriptor()` returns exactly the descriptor given. Unit is UCUM and may be empty; description may be empty. | observability.md §4, §5 "Naming" | happy |
| FR-026 | `Counter.Add(ctx, n int64, attrs ...Attr)` on an instrument created by `telemetrytest.Meter` MUST record the value and attributes of each call as a `Measurement`. | observability.md §4, §5 | happy |
| FR-027 | `telemetrytest.Meter`'s `Counter.Add` MUST accept `0` and MUST NOT record a negative value: it collects an `ErrNegativeCount` violation instead. | testing.md §4 "Boundaries" | unhappy (boundary) |
| FR-028 | `Histogram.Record(ctx, value float64, attrs ...Attr)` and `Gauge.Record(ctx, value float64, attrs ...Attr)` MUST each be recorded by `telemetrytest.Meter` as one `Measurement` with the given value and attributes. | observability.md §5 "What to record" | happy |
| FR-029 | `telemetry.StartTimer(clock func() time.Time, hist Histogram)` MUST read the clock once and return a running `Timer`; `Timer.Stop(ctx, attrs ...Attr)` MUST read the clock once and record the elapsed time in seconds, as a `float64`, into `hist` with `attrs`. Use `defer timer.Stop(ctx, attrs...)`. The `Meter` port reads no clock. | observability.md §4 "Timing" | happy |
| FR-030 | Because `Stop` is deferred, a failing or panicking operation MUST still record its elapsed time. `Stop` MAY be called more than once and records each time. | observability.md §4 "Timing"; testing.md §4 | unhappy (failure) |
| FR-031 | Reserved: no `timeAsync`; see FR-013. | observability.md §4 | n/a (see 4.7) |
| FR-032 | Reserved: see FR-030 and FR-031. | observability.md §4 | n/a (see 4.7) |
| FR-033 | A `Timer` made with a nil clock or a nil histogram, and the zero `Timer`, MUST record nothing and read no clock. A use case with no injected clock therefore skips the duration rather than reading the wall clock. `NoopMeter` instruments record nothing. | observability.md §4, architecture.md §3 | happy |
| FR-034 | `NoopMeter.Counter`, `Histogram` and `Gauge` MUST accept any descriptor (including an empty name and empty strings) without panicking, validate nothing, and return an instrument whose `Descriptor()` returns it. | observability.md §1.5 | unhappy (empty and minimal) |
| FR-035 | `telemetrytest.Meter` MUST store one `Measurement` per recorded value (`Name`, `Kind`, `Unit`, `Value`, `Attrs`), in call order, across all instruments; `Measurements()` returns copies, `MeasurementsNamed(name)` filters, `Instruments()` lists `InstrumentInfo` in creation order. | observability.md §9 | happy |
| FR-036 | Timing in tests MUST be deterministic: `telemetrytest.NewClock(start, step)` returns a clock whose `Now` returns the current time and then advances it by `step`, so a timer started and stopped once records exactly `step` in seconds; `Advance(d)` moves it by hand. | testing.md §6 "Deterministic" | unhappy (determinism) |
| FR-037 | A clock that goes backwards MUST NOT produce a negative duration: `Timer.Stop` MUST record an elapsed time below zero as zero. | testing.md §4 "Boundaries" | unhappy (boundary) |
| FR-038 | When kind and descriptor both equal an existing instrument's, `telemetrytest.Meter` MUST return an equivalent instrument. The same name with a different kind, or the same name and kind with a different unit or description, MUST collect an `ErrInstrumentConflict` violation, keep the first definition, and return an instrument that records nothing. | testing.md §4 "State errors" | unhappy (state) |
| FR-039 | `telemetrytest.Meter` MUST reject an instrument name for which `telemetry.ValidName` is false by collecting an `ErrInvalidName` violation and returning an instrument that records nothing. | observability.md §5 "Naming" | unhappy (invalid input) |
| FR-040 | Reserved: a `Timer` is not tied to a `Meter`, so there is no "histogram from another meter" failure; a histogram from any meter records into that meter. | testing.md §4 "Invalid input" | n/a (see 4.7) |

### 4.4 Telemetry bundle and recorder

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-041 | `telemetry.Telemetry` MUST be a struct of exported fields `Logger`, `Tracer` and `Meter`; a nil field means "no-op". The zero value is valid. | observability.md §4 "No-op by default" | happy |
| FR-042 | `Telemetry.WithDefaults()` MUST return a copy with every nil port replaced by its no-op. A use case calls it once in its constructor (default `Telemetry{}` through its `WithTelemetry` option) and MUST NOT nil-check a port afterwards. `telemetry.Noop{}` implements all three ports and is usable directly. | observability.md §4 | happy |
| FR-043 | Supplying only one port MUST leave the other two as their no-ops after `WithDefaults` (not nil). Each default port, exercised through the bundle (`Enabled` is `false`, `Start` returns a `NoopSpan`, instruments record nothing), MUST behave as a no-op. | testing.md §4 "Empty and minimal" | unhappy (empty and minimal) |
| FR-044 | `Telemetry` MUST be an immutable-by-convention value type: `WithDefaults` returns a new value and nothing mutates the receiver; it is comparable when the dynamic types of its fields are. No setters. | architecture.md §3 | architecture (convention; `go vet` and review) |
| FR-045 | `telemetrytest.Recorder` MUST create one `Logger`, `Tracer` and `Meter` (`NewRecorder`), expose them as fields, and `Telemetry()` MUST return a bundle wrapping exactly those three objects. | observability.md §9 | happy |
| FR-046 | `Recorder.Clear()` MUST remove the log, span and measurement records and the collected violations. It MUST keep created instruments (`Meter.Instruments`) and configuration (the logger's minimum level); earlier instrument handles stay valid and keep recording. | testing.md §6 "Independent" | unhappy (state) |
| FR-047 | Slices returned by the recording doubles (`Records`, `Spans`, `Measurements`, a record's `Attrs` and `Errors`) MUST be copies: mutating one MUST NOT change the double. The doubles MUST be safe for concurrent use, and their zero values ready to use. | testing.md §4 "Immutability" | unhappy (immutability) |

### 4.5 Package shape and guardrails

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | Package `telemetry` MUST export, each with a doc comment starting with its name: `Logger`, `Level` and its five constants, `Tracer`, `Span`, `Status` and its three constants, `Meter`, `Counter`, `Histogram`, `Gauge`, `Descriptor`, `Attr` with `String`, `Int`, `Int64`, `Float`, `Bool`, `Err`, `Telemetry`, `Timer`, `StartTimer`, `ValidName`, the no-ops (`NoopLogger`, `NoopTracer`, `NoopSpan`, `NoopMeter`, `Noop`) and the shared keys (FR-052). The doc comment of each port (`Logger`, `Tracer`, `Meter`) MUST state the contract in FR-056. | architecture.md §7 | happy |
| FR-049 | Package `telemetrytest` MUST be the only package that exports a telemetry double or contract suite (`Recorder`, `Logger`, `Tracer`, `Meter`, `Clock`, `RunLoggerContract`, `RunTracerContract`, `RunMeterContract`, the record types and violations). Package `telemetry` MUST NOT export any double. | architecture.md §7; shared-types-and-ownership.md FR-038 | happy |
| FR-050 | Pointer, no check here: the `internal/archtest` rules `test-support-isolation` (no non-test file imports a `*test` package) and `third-party-import` / `go-mod-require` (the module has no `require`) cover this: `telemetrytest` imports `testing` only for its contract suites and `RequireNoViolations`, and no production package imports it. | testing.md §6 "No test-only code in production packages"; architecture-guardrails.md | guardrail |
| FR-051 | Package `telemetry` MUST import the standard library only and no package of the module; `telemetrytest` MUST import `telemetry` only (plus the standard library); `slogtelemetry` MUST import `telemetry` and the standard library only. Enforced by the `internal/archtest` allow-list (`import-not-allowed`, `kernel-imports-context`) and its standard-library deny-list (`stdlib-denied`: `os`, `net`, `reflect`, `log`, `math/rand` and similar). No port, no-op or double may hold a package-level logger, tracer or meter variable (`package-level-var`); empty-struct no-ops used as values are exempt (FR-059). | architecture.md §1, §9; observability.md §4, §10 | architecture |
| FR-052 | The shared attribute keys MUST be defined once, in `telemetry/signals.go`, as exported constants: `AttrOutcome = "srd5e.outcome"`, `AttrErrorType = "error.type"`, with the values of `AttrOutcome` `OutcomeSuccess = "success"` and `OutcomeFailure = "failure"`. Other specs refer to `telemetry.AttrOutcome` and `telemetry.AttrErrorType`. This file is the only definition site and the string values are part of the public contract. `AttrErrorType` carries `srderr.ErrorType(err)` for a returned error only, never for a failure result. The value sets other than these are owned by `doc/rules/observability.md` §5 and `shared-types-and-ownership.md` FR-041. | observability.md §5 "Naming", "Attributes", §7 table; shared-types-and-ownership.md FR-041 | happy |
| FR-053 | No non-test file in `telemetry`, `telemetrytest` or `slogtelemetry` may call `time.Now`, `time.Since`, `rand.New` or any `math/rand` function, read the environment, or spawn a goroutine; this extends `architecture.md` §9. There is no carve-out: the seeded `*rand.Rand` is injected by the consumer (`dice-and-rolls.md` FR-064), and time is read only through a clock the caller injects (`Timer`, `telemetrytest.Clock`). Enforced by `internal/archtest` (`hidden-effect`, `stdlib-denied`). Printing to standard output is banned module-wide by the same rule. | architecture.md §3, §9; observability.md §4 "Timing"; dice-and-rolls.md FR-064 | unhappy (determinism) |

### 4.6 Added requirements

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-054 | Reserved: no `traceAsync`; see FR-013. | observability.md §6 | n/a (see 4.7) |
| FR-055 | Reserved: no `timeAsync`; see FR-031. | observability.md §4 "Timing" | n/a (see 4.7) |
| FR-056 | The doc comments of `Logger`, `Tracer` and `Meter` (and `telemetry/doc.go`) MUST state that the ports return no errors, that a backend handles its own failures, and that a backend that panics is the host's bug: "srd5e does not recover" (`srderr.Guard` wraps module-supplied ports that return errors, not telemetry). The recording doubles in `telemetrytest` collect violations on misuse by design and never panic. A test MUST read the doc comments of the three port declarations and fail if the sentence is missing. | section 9 (decided); `architecture.md` §6 | unhappy (invalid input) |
| FR-057 | The doc comment of `Tracer.Start` MUST state that the caller defers `Span.End` immediately and passes the returned context to child work, and the doc comment of `Timer` that `Stop` is meant for `defer`. Checked with FR-056's doc-comment test. | observability.md §6 | unhappy (invalid input) |
| FR-058 | Duration instruments MUST be created with unit `"s"` and `Timer.Stop` records seconds. The use case that creates the histogram owns this; `telemetrytest.Meter` records the unit it was given and the signal contract tests of each use case pin `"s"`. | observability.md §5 "Naming" (durations are always seconds) | unhappy (invalid input) |
| FR-059 | The rule files MUST state the departures this spec relies on, as accepted by `shared-types-and-ownership.md` FR-051 and now applied: `observability.md` §4 (the ports are synchronous, return no errors and take `context.Context` first; a `Timer` with an injected clock replaces `Meter` timing); `observability.md` §7 (log attributes are closure-only, the example is `logger.Log(ctx, telemetry.LevelDebug, "check resolved", func() []telemetry.Attr { return []telemetry.Attr{telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeSuccess)} })`, and a returned failure carries no `error.type`, per `errors-and-results.md` FR-043); `observability.md` §4, §10, §11 and the `architecture.md` §10 "Shared mutable state: package-level variables" anti-pattern (a zero-value no-op struct is not a "package-level logger, tracer or meter"). Any later edit of those passages MUST keep these statements and give the file a fresh `RULE-TAG`. | architecture.md §4, §10; observability.md §4, §7, §10, §11; shared-types-and-ownership.md FR-051 | architecture (the no-op exemption test) |
| FR-060 | Pointer, nothing added here: the `internal/archtest` rules `test-support-isolation` (no production file imports a `*test` package) and the package allow-list own the checks. Doubles MAY depend on ports and kernel packages. | architecture.md §1, §9; shared-types-and-ownership.md FR-038 | guardrail |
| FR-062 | `telemetry.Noop` MUST be an empty struct embedding `NoopLogger`, `NoopTracer` and `NoopMeter`, so that `Noop{}` is a `Logger`, a `Tracer` and a `Meter` (compile-time assertions), and `Telemetry{}.WithDefaults()` MUST equal a bundle of the three no-ops. A consumer cannot break the zero-value guarantee because the no-ops hold no state. A test MUST assert the assertions and the equality. | shared-types-and-ownership.md FR-036; observability.md §4 | happy |
| FR-063 | The contract suites are three exported functions, one per port, in package `telemetrytest`: `RunLoggerContract(t, factory)`, `RunTracerContract(t, factory)` and `RunMeterContract(t, factory)`, each taking `func(t *testing.T) <port>`. They MUST be run against every adapter and double, in this module the no-ops, the `telemetrytest` doubles and `slogtelemetry.Logger`, and are meant to be reused by consumers' own adapters (`shared-types-and-ownership.md` FR-039). | testing.md §3, §7; shared-types-and-ownership.md FR-039 | architecture |
| FR-064 | Every instrument returned by a `NoopMeter` (counter, histogram, gauge) MUST accept `Add` or `Record` with any value (including a negative one) and any attributes, and a cancelled context, without panicking and without any observable effect. | observability.md §1.5 | unhappy (empty and minimal) |
| FR-065 | `Span`, `Counter`, `Histogram` and `Gauge` are handles, not driven ports. They need no double or contract suite of their own; their behaviour is pinned through the port contract suites of FR-063. | architecture-guardrails.md | architecture |
| FR-066 | `slogtelemetry.New(*slog.Logger)` MUST return a `Logger` that implements `telemetry.Logger` on the caller-supplied logger. It MUST NOT call `slog.Default` or create a handler. Levels map trace to `slog.LevelDebug-4`, debug to `slog.LevelDebug`, info to `slog.LevelInfo`, warn to `slog.LevelWarn`, error to `slog.LevelError`; a level outside the set is clamped. A nil `*slog.Logger` and the zero `Logger` MUST enable no level and log nothing. The `attrs` closure is called at most once and only when the level is enabled; an invalid `Attr` is skipped. | observability.md §4; go.md §7 | happy; unhappy (empty and minimal) |

### 4.7 Withdrawn and reserved ids

| Id | Status |
|---|---|
| FR-005 | Reserved: level shortcut extension; Go has none (callers use `Log`). |
| FR-013, FR-014 | Reserved: `traceAsync`; `Start` plus deferred `End` spans synchronous and context-taking code alike. |
| FR-031, FR-032 | Reserved: `timeAsync`; `Timer` with `defer`. |
| FR-040 | Reserved: a `Timer` is not bound to a `Meter`. |
| FR-054, FR-055 | Reserved: synchronous throw from an async body; no async body exists. |
| FR-061 | Withdrawn: moved to shared-types-and-ownership.md (FR-038). |

Ids are never reused. FR-066 is new.

## 5. Model

No domain layer: `telemetry` holds ports, handles and plain value types.

- **Ports** (interfaces): `Logger`, `Tracer`, `Meter`. All three are synchronous
  and return no errors; a use case that does I/O passes its `context.Context` and
  the same ports serve it (FR-013).
- **Handles, not ports:** `Span`, `Counter`, `Histogram`, `Gauge` (FR-065).
- **Value types:** `Level`, `Status`, `Descriptor`, `Attr`, `Telemetry`, `Timer`;
  in `telemetrytest` `Kind`, `LogRecord`, `SpanRecord`, `Measurement`,
  `InstrumentInfo`, all returned as copies.
- **`Telemetry`:** a struct of three interface fields; `Noop` is the all-no-op
  value (FR-062) and `Telemetry{}` is valid.
- **Invariants:** a recording span is writable only between start and end
  (FR-023); an instrument name maps to exactly one kind and descriptor (FR-038);
  an attribute value is one of four plain types by construction (FR-010). The
  first is enforced by the recording double, the only place that sees the
  violation; the third by the type.
- **Failures:** misuse is a collected violation (a plain `error` matching a
  sentinel such as `telemetrytest.ErrSpanEnded` with `errors.Is`), reported by
  `Violations()` and failed by `RequireNoViolations`. The no-ops never validate
  and never panic (FR-034, FR-064). No `srderr.PortError`: nothing here does
  I/O. srd5e does not recover from a consumer's telemetry backend (FR-056).
- **Layout** (`shared-types-and-ownership.md` FR-036 to FR-038): ports, handles,
  no-ops, `Timer` and the bundle in `telemetry/` (one file per concept:
  `logger.go`, `tracer.go`, `meter.go`, `timer.go`, `telemetry.go`, `attr.go`,
  `level.go`, `signals.go`); doubles and contract suites in
  `telemetry/telemetrytest/`, imported only by tests; the optional adapter in
  `telemetry/slogtelemetry/`. Doubles depend on ports, never the reverse. A
  future OpenTelemetry adapter would be its own package and, because it needs a
  third-party module, its own Go module.

## 6. Interface

Standard-library types only. Indicative; the source of truth is the Go code.

```go
package telemetry

type Level int // LevelTrace, LevelDebug, LevelInfo, LevelWarn, LevelError
type Status int // StatusUnset, StatusOK, StatusError

type Attr struct{ /* key string; val any */ }
func String(key, value string) Attr
func Int(key string, value int) Attr
func Int64(key string, value int64) Attr
func Float(key string, value float64) Attr
func Bool(key string, value bool) Attr
func Err(err error) Attr
func (a Attr) Key() string
func (a Attr) Value() any
func (a Attr) Valid() bool

type Logger interface {
	Enabled(ctx context.Context, level Level) bool
	Log(ctx context.Context, level Level, msg string, attrs func() []Attr)
}

type Span interface {
	SetAttributes(attrs ...Attr)
	SetStatus(status Status)
	RecordError(err error)
	End()
}
type Tracer interface {
	Start(ctx context.Context, name string, attrs ...Attr) (context.Context, Span)
}

type Descriptor struct{ Name, Unit, Description string }
type Counter interface {
	Descriptor() Descriptor
	Add(ctx context.Context, n int64, attrs ...Attr)
}
type Histogram interface {
	Descriptor() Descriptor
	Record(ctx context.Context, value float64, attrs ...Attr)
}
type Gauge interface {
	Descriptor() Descriptor
	Record(ctx context.Context, value float64, attrs ...Attr)
}
type Meter interface {
	Counter(d Descriptor) Counter
	Histogram(d Descriptor) Histogram
	Gauge(d Descriptor) Gauge
}

func StartTimer(clock func() time.Time, hist Histogram) Timer
func (t Timer) Stop(ctx context.Context, attrs ...Attr)

type Telemetry struct {
	Logger Logger
	Tracer Tracer
	Meter  Meter
}
func (t Telemetry) WithDefaults() Telemetry

type Noop struct{ NoopLogger; NoopTracer; NoopMeter }

// signals.go
const (
	AttrOutcome   = "srd5e.outcome"
	AttrErrorType = "error.type"
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
)
func ValidName(name string) bool
```

Notes:

- Every method that can emit takes `context.Context` first, so a backend can link
  spans and cancel; none stores it. Use cases create their instruments once, in
  the constructor. Attributes are per call, not bound to instruments.
- Duration instruments are created with `Unit: "s"`; `Timer.Stop` records seconds
  as a `float64`; the clock is injected (`time.Now` from the host, or
  `telemetrytest.Clock`).
- The ports return no errors and srd5e does not recover from them (FR-056); the
  recording doubles collect violations on misuse by design.

### Shape of `telemetrytest`

`telemetry/telemetrytest` is the home of the telemetry test doubles and contract
suites, in the manner of `net/http/httptest`. It imports `testing` and `telemetry`
only. Other specs add `dicetest` (`dice-and-rolls.md`) and fakes for later ports,
each in its own `<pkg>test` package.

```go
package telemetrytest

type Recorder struct{ Logger *Logger; Tracer *Tracer; Meter *Meter }
func NewRecorder() *Recorder
func (r *Recorder) Telemetry() telemetry.Telemetry
func (r *Recorder) Clear()
func (r *Recorder) Violations() []error

type LogRecord struct {
	Level   telemetry.Level
	Message string
	Attrs   []telemetry.Attr
}
type SpanRecord struct {
	ID, ParentID int
	Name         string
	Attrs        []telemetry.Attr
	Status       telemetry.Status
	Errors       []error
	Ended        bool
}
func (r SpanRecord) AttrValue(key string) (any, bool)
type Measurement struct {
	Name  string
	Kind  Kind // KindCounter, KindHistogram, KindGauge
	Unit  string
	Value float64
	Attrs []telemetry.Attr
}

func NewClock(start time.Time, step time.Duration) *Clock
func RequireNoViolations(t testing.TB, vs ...Violator)
func RunLoggerContract(t *testing.T, factory func(t *testing.T) telemetry.Logger)
func RunTracerContract(t *testing.T, factory func(t *testing.T) telemetry.Tracer)
func RunMeterContract(t *testing.T, factory func(t *testing.T) telemetry.Meter)
```

| Type | Constructor | Read | Control |
|---|---|---|---|
| `telemetrytest.Logger` | zero value, or `NewLogger(min telemetry.Level)` | `Records()`, `RecordsAt(level)` | `SetMinLevel`, `Clear` |
| `telemetrytest.Tracer` | zero value | `Spans()` (start order), `SpansNamed(name)` | `Clear` |
| `telemetrytest.Meter` | zero value | `Measurements()`, `MeasurementsNamed(name)`, `Instruments()` | `Clear` |
| `telemetrytest.Recorder` | `NewRecorder()` | `Logger`, `Tracer`, `Meter`, `Telemetry()` | `Clear()` |
| `telemetrytest.Clock` | `NewClock(start, step)` | `Now` | `Advance` |

No context package imports telemetry doubles; `telemetry` is a kernel package.

## 7. Observability

Silent by design: these are the ports through which every signal is emitted. A
port that observed itself would recurse, and a no-op that emitted anything
would break "zero cost when unused" (`observability.md` §1.5).
`doc/observability/signals.md` gets no rows from this spec. The seven questions:

1. **What can go wrong?** A consumer's implementation panics, is slow, or
   violates the contract (calls the `attrs` closure while disabled, mutates a
   span after it ended). A srd5e use case may pass a bad name or attribute,
   which only the recording doubles detect.
2. **How would an operator know?** They would not: the ports cannot report on
   themselves. Defence is in tests (recording doubles collect violations, the
   contract suites) and architecture tests (FR-051, FR-053), not at runtime.
3. **What would someone need to diagnose it?** A panic from a consumer's
   implementation arrives with its own stack trace, since srd5e does not recover
   from it (FR-056). Nothing more.
4. **What should be counted or timed?** Nothing here.
5. **Is existing instrumentation still correct?** None exists yet.
6. **What should a consumer monitor?** Nothing from the ports; adapters should
   be cheap and non-panicking.
7. **Is anything noisy?** No. No-ops emit nothing; doubles are test-only.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| (none) | | | | |

`srd5e.outcome` and `error.type` (FR-052) are not signals; they are vocabulary
for other specs' signals.

## 8. Test plan

Tests live beside the code: `telemetry/telemetry_test.go` (and `example_test.go`)
for the ports, no-ops, `Timer` and bundle; `telemetry/telemetrytest/*_test.go` for
the doubles; `telemetry/slogtelemetry/*_test.go` for the adapter; plus
`internal/archtest` for guardrails. "Contract" rows run against the no-op, the
recording double and the slog adapter through one shared suite per port (FR-063).
Each suite takes a factory per implementation and is exported for consumer reuse
(`shared-types-and-ownership.md` FR-039). Tests are table-driven with `t.Run`
and `t.Parallel` where safe, in external test packages; `Example*` functions
document `Recorder`, `StartTimer`, `WithDefaults` and `ValidName`.

| Ids | Level | Notes |
|---|---|---|
| FR-001 | unit | values, order, `String()` and `Valid()` of `Level` (`TestLevel`) |
| FR-002 | unit | a fake `Logger` receives the closure and the `Err` attribute intact; `Err(nil)` is invalid |
| FR-003, FR-004 | unit | table over all levels including invalid, with nil and non-nil closure, cancelled context; closure probe (`TestNoopBehaviour`) |
| FR-006 | contract | probe closure; no-op, the recording logger with a minimum above the call, the slog adapter; monotonic `Enabled` |
| FR-007, FR-009 | unit | order and all fields; closure counter; no closure gives empty `Attrs`; later mutation of the result changes nothing |
| FR-008 | unit | table: below, at, above the minimum, across every level (`TestLoggerMinLevel`) |
| FR-010 | unit | invalid `Attr` (zero, empty key) dropped and reported with `ErrInvalidAttr` for the three doubles |
| FR-011 | contract | `Start` returns a non-nil span, keeps context values, tolerates a cancelled context |
| FR-012, FR-017, FR-018 | unit | recorded error and `StatusError` through a use case with a failing roller (see `resolve-d20-test-use-case.md`); `RecordError` appends in order and leaves status and `End` alone; `StatusOK` then `StatusError` ends as error; nil error ignored |
| FR-015, FR-016 | unit | start attributes, later attribute, overwrite; default `StatusUnset`, last write wins |
| FR-019, FR-020 | contract | no-op `Start` returns the same context; every `NoopSpan` method after `End` |
| FR-021, FR-022 | unit | all `SpanRecord` fields including `ParentID`; start order with a nested pair; two levels deep; concurrent starts (`-race`); top-level has `ParentID` 0 |
| FR-023 | unit | each mutator after `End` is ignored and reported with `ErrSpanEnded`; `End` twice |
| FR-024, FR-039 | unit | shared name table through `ValidName` (and a fuzz test, `FuzzValidName`): valid and invalid (empty, uppercase, two segments, trailing dot, space, no `srd5e` prefix, `srd5e.1x.y`, `srd5e._x.y`) |
| FR-025 | contract | each create method; `Descriptor()` returns the descriptor |
| FR-026 to FR-028 | unit | recorded `Add`; boundary `0`, `1`, `-1`; histogram and gauge, int-like and fractional values |
| FR-029, FR-030, FR-033, FR-037 | unit | `Timer` with `telemetrytest.Clock`: one record in seconds; deferred `Stop` after a panicking function still records; nil clock or nil histogram records nothing and reads no clock; backwards clock records 0 (`TestTimer`, `ExampleStartTimer`) |
| FR-034, FR-064 | unit | `NoopMeter` create methods: empty descriptor; instruments accept `0`, negative, fractional values, attributes and a cancelled context |
| FR-035 | unit | order across a counter and a histogram; `Kind` is the matching constant; copies |
| FR-036 | unit | table: step 0, 1500 ms (1.5), 1 microsecond (0.000001); `Advance` |
| FR-038 | unit | same descriptor twice; same name other kind; same name and kind, other unit or description; first definition kept, conflicting handle records nothing |
| FR-041 to FR-043 | unit | defaults; table: only logger, only tracer, only meter, each default exercised as a no-op (`TestTelemetryWithDefaults`) |
| FR-044 | review | no setters; `WithDefaults` leaves the receiver unchanged (unit assertion) |
| FR-045, FR-046 | unit | `Telemetry()` returns the recorder's own three objects; `Clear` then keep using an earlier instrument; `Instruments()` and the minimum level unchanged |
| FR-047 | unit | mutating a returned slice changes nothing; concurrent use under `-race` |
| FR-048, FR-049 | architecture | `internal/archtest` `missing-doc` and `missing-package-doc`; export surface pinned by `scenarios` compiling against the public API |
| FR-050, FR-060 | n/a | covered by `internal/archtest` `test-support-isolation` (no check here) |
| FR-051, FR-053 | architecture | `import-not-allowed`, `stdlib-denied`, `hidden-effect`, `package-level-var` applied to `telemetry`, `telemetrytest`, `slogtelemetry` |
| FR-052 | unit | pins both keys and both outcome values (`TestSharedKeys`); `srderr.ErrorType` is the only source of `error.type` (use-case tests) |
| FR-056, FR-057 | unit | doc comments of the three ports, `Tracer.Start` and `Timer` contain the required sentences, read with `go/parser` |
| FR-058 | unit | each use case's signal test pins `Unit: "s"` for its duration histogram |
| FR-059 | architecture | a zero-value no-op used as a field is not flagged by `package-level-var`; a package-level `telemetry.Logger` variable is. Fresh-tag-on-edit is not automated (below) |
| FR-062 | unit | compile-time assertions that `Noop{}` is a `Logger`, `Tracer` and `Meter`; `Telemetry{}.WithDefaults()` holds the three no-ops |
| FR-063 | contract | `TestNoopContracts`, `TestContracts` (doubles) and the slog adapter's `TestContract` run the three suites |
| FR-065 | review | no handle has its own suite |
| FR-066 | unit | level mapping table including an invalid level; nil `*slog.Logger`; `attrs` called once when enabled and never when disabled; invalid `Attr` skipped (`TestLog`, `TestDisabledSkipsAttrs`, `TestNilLoggerIsSilent`, `ExampleNew`) |

- **Pyramid:** nearly all unit and contract tests; no scenario tests here. Per
  `architecture.md` §9, ports, doubles and suites ship together with the first
  code that uses them.
- **Mutation testing:** `telemetry/` is a kernel package and SHOULD be mutated
  (validation branches in FR-024, FR-027, FR-037, FR-038, FR-039), target score at
  least 80. `gomu` is run only inside Docker with `--cpus` and `--workers` capped
  at 1/4 of `nproc`, as documented in `doc/rules/testing.md`; it is never run on
  the host.
- **Not covered deliberately:** (a) whether a later edit of the FR-059 passages
  gives the rule file a fresh `RULE-TAG` is checked in review; (b) behaviour of a
  consumer's own implementations (section 1).



Deliberately not covered: FR-050 - a pointer row that adds no requirement here; the checks are `internal/archtest` rules.

Deliberately not covered: FR-060 - a pointer row that adds no requirement here; the checks belong to `internal/archtest`.

**Implementation status.**

- Done (code in `telemetry/`, `telemetry/slogtelemetry/`, `telemetry/telemetrytest/`, with tests): every live id. FR-044 is review-only and FR-050 and FR-060 are pointer rows.
- FR-056 and FR-057: the sentences are in `telemetry/doc.go` and the `Logger`, `Tracer`, `Meter`, `Tracer.Start` and `Timer` comments, pinned by `TestPortDocComments` (`go/parser`).

Not yet implemented: none.

## 9. Open questions and decisions

No open questions. Settled:

- Attributes are closure-only on `Log`; there are no async wrappers because Go
  has no `Future`: a use case that does I/O takes `context.Context` and the same
  `Start`/`End` and `Timer` serve it (FR-013). Adding a method to a port later
  would break implementers, so the ports stay at their present size.
- No recover around consumer telemetry in use cases: it would add a branch to
  every use case, hide bugs and make srd5e swallow panics (`architecture.md` §6);
  revisit if a real consumer needs it (FR-056). Dice and content ports, which can
  fail, are wrapped by `srderr.Guard`; telemetry ports return no errors.
- No-ops do not validate (zero cost); recording doubles do, by collecting
  violations rather than panicking. The "identical behaviour" test of
  `observability.md` §9 covers valid use only.
- Doubles live in `telemetrytest` (`shared-types-and-ownership.md` FR-038);
  no-ops are plain structs in the port package, not adapters (FR-036); contract
  suites are exported from `telemetrytest` (FR-039, FR-063).
- Zero-value no-op structs are exempt from the package-level-variable ban
  (FR-059).
- A "run with no-op and recording" helper in `telemetrytest` is deferred until
  two contexts repeat the pattern.
- `Meter` has no timing method; `Timer` with an injected clock replaces it, so the
  port needs no clock and `telemetry` never reads the wall clock (FR-029,
  FR-053).
- `telemetrytest.Tracer` records no span duration (it would need a clock).
- Shared attribute keys are only `srd5e.outcome` and `error.type`; context keys
  live in their contexts; value sets are in `shared-types-and-ownership.md`
  FR-041.
- `telemetrytest.Tracer` links parents through the returned `context.Context`,
  so concurrent spans are linked correctly (FR-022); the limitation of a stack-based design to
  nested and sequential use does not carry over.

## 10. Changelog impact

Under `Unreleased` / **Added**:

- Package `telemetry`: the `Logger`, `Tracer` and `Meter` ports with `Level`,
  `Span`, `Status`, `Counter`, `Histogram`, `Gauge`, `Descriptor` and `Attr`; the
  `Telemetry` bundle with zero-value no-op defaults and `WithDefaults`; `Timer`;
  shared attribute keys (`AttrOutcome`, `AttrErrorType`).
- `NoopLogger`, `NoopTracer`, `NoopSpan`, `NoopMeter`, `Noop` (zero-value, in the
  port package).
- Package `telemetry/telemetrytest` with `Recorder`, the recording doubles,
  `Clock` and the three contract suites; package `telemetry/slogtelemetry` with
  the `log/slog` adapter.

Rule-file amendments (not user-visible) are applied to `doc/rules/observability.md`
§4, §7, §10, §11 and `doc/rules/architecture.md` §4, §10 (FR-059).

Not **BREAKING**: first release of these APIs. After release, renaming a shared
attribute key, adding a method to a port interface, or changing a no-op's
behaviour is **BREAKING** (`doc/rules/architecture.md` §7).
