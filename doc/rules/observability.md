# srd5e Observability Rules

Normative. MUST / MUST NOT / SHOULD are binding on all code and review.
Read together with [architecture.md](architecture.md); observability follows
its layering and dependency rules.

## 1. Principles

1. **The goal is good observability.** Someone running a system built on srd5e
   MUST be able to tell what it is doing, whether it is healthy, and why it
   gave a particular result, without changing the code. Observability is a
   feature of every change, not an afterthought and not optional polish.
2. **The library emits; the consumer decides where it goes.** srd5e defines
   observability ports in `telemetry`. Consumers inject OpenTelemetry, zap,
   Sentry or anything else through their own adapter. srd5e never chooses a
   backend.
3. **Instrument for diagnosis.** Emit the signals someone would need to debug
   a wrong result, find a slow path, or understand how a rule resolved. The
   evaluation in §2 decides which. Do not wait for an incident to add them.
4. **Signal over noise.** Emit what helps, at the right level, once. Volume is
   a cost for every consumer.
5. **Zero cost when unused.** With no telemetry injected, everything is a
   no-op and behaviour is identical.
6. **Monitoring is part of the design.** A signal nobody can act on is
   incomplete. Each signal states what it is for and what a consumer should
   watch or alert on (see §2).

---

## 2. Evaluating observability is mandatory

Agents and contributors MUST actively evaluate observability and monitoring
for **every change**: a new use case, changed behaviour, a new port or adapter,
a new failure mode, a performance-sensitive path, or a bug fix. A change is
not done until this evaluation has been performed, acted on and reported.
Skipping it, or answering it with a bare "not applicable", is a defect.

### The evaluation

Before finishing, answer each of these for the change. Think about it, do not
just tick it off.

1. **What can go wrong here?** Failure modes: bad content, a port failing,
   invalid input, an invariant violated, unexpected rule interaction.
2. **How would an operator know it went wrong?** What signal (metric, span,
   log) surfaces each failure mode? Is it distinguishable from the normal
   outcome?
3. **What would someone need to diagnose it?** The identifiers, inputs summary,
   intermediate values and outcome, as low-cardinality attributes or in the
   returned result.
4. **What should be counted or timed?** Throughput, outcomes, latency, sizes,
   errors (§5).
5. **Is the existing instrumentation still correct?** Does the change make an
   existing metric, span or log misleading, stale or incomplete? Update it.
6. **What should a consumer monitor or alert on?** Which signal, what
   condition, and why it matters (e.g. any content decode failure; a latency
   regression; an outcome rate that suddenly shifts).
7. **Is anything now noisy?** Does the change add volume without value, or log
   at the wrong level?

### The outcome

- **Add or change the instrumentation** the evaluation calls for, following the
  rest of this document.
- **Update the signal catalogue** (below) for every metric, span or
  log event added, changed or removed.
- **Report it.** The PR description (see `.github/pull_request_template.md`)
  and the agent's final summary MUST contain an `Observability` entry saying
  what was added, changed or removed, and what a consumer should monitor.
- **If you add nothing, justify it specifically.** State why no failure mode,
  diagnostic need or measurement applies to this change (for example: "pure
  rename, no behavioural change; existing `srd5e.d20_test.resolved` unaffected").
  "N/A" or "no changes" alone is not acceptable.
- **Bug fixes:** also add the signal that would have exposed the bug sooner,
  or state why none could have.
- **Reviewers MUST reject** a change that lacks the evaluation or its report.

### The signal catalogue

`doc/observability/signals.md` is the single list of every signal srd5e emits.
It MUST stay in sync with the code. For each signal it records:

| Field | Meaning |
|---|---|
| Name | The metric, span or log-event name, exactly the string value of the Go constant |
| Go constant | The exported constant that holds it, e.g. `d20.SpanResolve` |
| Type | Counter, histogram, gauge, span or log event (with level) |
| Unit | Unit of measure, where applicable |
| Attributes | Allowed attribute keys and their possible values |
| Emitted by | The use case or adapter that produces it (`package.Type.Method`) |
| Purpose | What question it answers or what failure it exposes |
| Suggested monitoring | What consumers should dashboard or alert on, and the threshold or condition to start from |

srd5e is a library and cannot run alerts. The catalogue is how it tells
consumers what to monitor, so the "Suggested monitoring" column MUST be filled
in for every signal where an operator could act on it.

---

## 3. Where observability lives

| Layer | Observability rule |
|---|---|
| Domain types and pure functions (value objects, rules, `dice`, `ability`, `modifier`) | **Silent.** No logging, metrics or spans, and no `telemetry` import. They expose what happened through return values. |
| Use cases (the entry points of a context package, e.g. `d20.Resolver.Resolve`) | **Instrumented.** Emit metrics, spans and logs through the injected ports. This is the main place telemetry is produced. |
| Adapters | **Instrumented only for what the use case cannot see**: retries, fallbacks, partial decodes, I/O timing. Emitted through the same ports, injected the same way. Failures the adapter returns are recorded by the use case, not by the adapter (§7, "Errors: who records what"). |

A context package holds domain and use case in one package (architecture.md
§1), so the split is by file: telemetry calls appear only in use-case files
(`d20/resolver.go`, `signals.go`) (and in adapter packages). Domain files in the same package MUST NOT
import `telemetry` or take a telemetry value.

- Result types already carry the *what* and *why* (modifiers applied, roll
  made, DC, outcome). Use cases translate those into telemetry. They MUST NOT
  re-derive them.
- A use case MUST NOT reach into domain internals to emit telemetry. If the
  domain holds information worth observing, it returns it.

---

## 4. Ports

Defined once in the kernel package `telemetry` (every context uses them; see
[architecture.md](architecture.md) §1). `telemetry` imports no other srd5e
package and only the standard library (`context`, `time`).

| Port | Responsibility |
|---|---|
| `Logger` | Emit a log event: `context.Context`, level (a typed enum), constant message, structured attributes supplied lazily; an error is passed as the `telemetry.Err` attribute. Exposes `Enabled(ctx, level) bool`. |
| `Tracer` | `Start(ctx, name, ...)` returns a derived `context.Context` and a `Span`. `Span` has `SetAttributes`, `RecordError`, `SetStatus` and `End`. |
| `Meter` | Create counters, histograms and gauges; each instrument records a value with attributes |

Each interface is small (1 to 3 methods where possible), carries no vendor
types, and is shaped so an OpenTelemetry adapter is a thin translation. The
exact method sets are defined by the Go declarations; this document owns the
rules, not the signatures.

A use case accepts the ports it needs, either as individual options or as a
bundle struct (`telemetry.Telemetry`) holding a `Logger`, `Tracer` and
`Meter`. The constructor replaces any nil field with the matching Noop, so the
use case never branches on "is telemetry enabled".

**Timing.** Domain and use-case code may not read the clock (A4 in
architecture.md §9: no `time.Now`, `time.Since`). Durations are therefore
measured by the adapters: a span's duration by the `Tracer` implementation, and
duration histograms through a timing helper in `telemetry` that takes an
injected clock (`func() time.Time`) supplied through the use case's options.
The default is the function value `time.Now` (referenced, never called directly, so A4 holds; reference: `d20.NewResolver`), overridden with
`WithClock`. Tests inject a deterministic clock.

**Context propagation.** Trace context travels in the `context.Context` that
`Tracer.Start` returns. Every use case that emits telemetry takes `ctx` as its
first parameter, passes the context returned by `Start` to every port call it
makes (content sources, nested spans, `Logger` calls) and never stores it in a
struct. srd5e does not parse, inject or propagate trace headers: linkage
between parent and child spans is entirely the consumer's `Tracer` adapter
working on that `ctx`.

Rules:

- **No-op by default.** `telemetry.Noop` implementations (`NoopLogger`,
  `NoopTracer`, `NoopMeter`, and a `Telemetry` bundle) hold no state and allocate nothing on the
  hot path. The zero `Telemetry` has nil fields and is valid input to a
  constructor, which calls `WithDefaults`; use `telemetry.Noop{}` for a
  ready-to-use value. They live in `telemetry`, not in an adapter package, so use cases
  can default to them.
- **Dependency injection only.** No package-level logger, tracer or meter, no
  global registries, no `slog.Default()` / `otel.Tracer(...)` lookups, no
  `init()` registration. A zero-value Noop is stateless and is not a global.
- **No vendor types.** Ports use standard-library types only: strings,
  numbers, booleans, `time.Duration`, attributes as plain key/value pairs.
  No OpenTelemetry, zap, Sentry or other SDK type may appear in a port or
  anywhere in this module.
- **Shaped for OpenTelemetry.** Port semantics MUST map cleanly onto the OTel
  model (spans with attributes and status, counters, histograms, severity
  levels) so a consumer's adapter is a thin translation.
- **Small.** Add a port method only when an instrumentation point needs it.
  Ports are interfaces defined next to their vocabulary in `telemetry`; a use
  case that needs only a counter MAY declare its own one-method consumer-side
  interface instead of taking the whole `Meter`.
- **No error returns.** Telemetry methods return nothing. A backend that
  fails must handle it internally; a backend that panics is the host's bug
  and the panic propagates (architecture.md §6). Telemetry never changes a
  use case's result.
- **Synchronous.** Ports are synchronous and non-blocking in contract. A
  backend that buffers or exports does so on its own goroutines; srd5e starts
  none.
- **Test doubles in `telemetry/telemetrytest`**: a recording `Logger`,
  `Tracer` and `Meter` (and a fake clock). Production packages never contain
  doubles.
- **Contract suite.** `telemetrytest` exports `RunLoggerContract`,
  `RunTracerContract` and `RunMeterContract` (each taking `*testing.T` and a
  factory). Every adapter and double MUST pass them: Noop, the recording
  doubles, `slogtelemetry.Logger`, and any adapter a consumer writes.
  The suite checks that methods accept a nil attribute list, tolerate a
  cancelled `ctx`, that `End` is idempotent, that a returned `ctx` still
  carries the caller's values, and that nothing panics for any level.
- **One adapter ships**: `telemetry/slogtelemetry` maps `telemetry.Logger` onto
  a caller-supplied `*slog.Logger` (stdlib only). It never calls
  `slog.Default()` and never creates a handler. Bridges for OpenTelemetry and
  other SDKs are **not** shipped (they would add a dependency for every
  consumer); show them as `Example*` functions in docs and the README.

---

## 5. Metrics

> **Rule: anything worth counting, timing or measuring in aggregate is a
> metric through the `Meter` port,** never only a log line (§8). Consumers
> route metrics to OpenTelemetry through their adapter. srd5e itself never
> imports an OTel package.

### What to record

- **Throughput:** a counter per use case invocation, e.g. a resolved check, an
  applied condition, a computed derived stat.
- **Outcomes:** counters broken down by result, e.g. outcome, rule
  outcome, validation failure, decode failure.
- **Latency:** a duration histogram for each use case, recorded through the
  timing helper (§4).
- **Sizes:** histograms of things that grow, e.g. number of modifiers applied,
  size of an encounter, number of content entries loaded.
- **Errors:** a counter of returned port failures (a non-nil `error` from a
  port) by error type. A failure **result** (content not found, decode
  problems reported as a value) is counted by outcome (`srd5e.outcome` =
  `failure`) plus a variant attribute defined by the use case's spec, and
  carries no `error.type`.
- **Content:** counters or gauges for content loaded, missing and rejected.

### Naming

- Names are lowercase, dot-separated and prefixed: `srd5e.<context>.<noun>` and
  an appropriate suffix, e.g. `srd5e.d20_test.resolved`,
  `srd5e.d20_test.duration`, `srd5e.content.decode_errors`. The wire name uses
  snake_case segments even where they differ from the Go package name (`d20`
  emits `srd5e.d20_test.*`).
- Units follow UCUM as OpenTelemetry does, and are stated in the instrument
  definition, not in the name. Durations are always seconds (`s`, as a
  `float64`); counts of things use an annotation such as `{modifier}`.
- Names (metric, span, log event, attribute key) are exported or package-level
  `const` strings, defined in one `signals.go` per context, or in `telemetry`
  for keys shared across contexts. No string literals at call sites.
  Constants follow Go naming: `SpanResolve`, `MetricResolved`, `AttrOutcome`.
- Renaming or removing a signal or attribute key is a breaking change for
  consumers' dashboards and alerts. Mark it **BREAKING** in `CHANGELOG.md`
  ([architecture.md](architecture.md) §7).

### Attributes

- Attributes MUST be **low-cardinality**: rule names, outcome,
  ability, context name, error type. Never ids, free text, whole
  objects, or unbounded numbers.
- Attributes MUST NOT contain personal or consumer-identifying data.
- Attributes are typed key/value pairs built with the `telemetry` constructors
  (`telemetry.String`, `Int`, `Bool`, `Float`) from constant keys. A value that
  belongs to a closed set is produced from a typed enum's `String()`, never
  from a free string.
- Use consistent attribute keys across metrics, spans and logs
  (`srd5e.outcome`, `srd5e.d20_test.kind`).
- Attribute **values** are shared too, so metrics, spans and logs agree:
  `srd5e.outcome` is `success` or `failure`; `srd5e.d20_test.kind` is
  `ability_check`, `saving_throw` or `attack_roll`;
  `srd5e.d20_test.roll_mode` is `normal`, `advantage` or `disadvantage`;
  `srd5e.d20_test.critical_hit` is `true` or `false` (attack rolls only);
  `srd5e.d20_test.roller_failure` is `unavailable` or `invalid_result`;
  `srd5e.d20_test.forced_failure_reason` is `chosen_to_fail` or `automatic`;
  `error.type` is `<port>_error` from `srderr.ErrorType` for a `*srderr.PortError` (one value per port, for example `dice_roller_error`; [errors-and-results.md](../specs/errors-and-results.md)). Nothing is emitted for other error types; a future non-port error type must first add its own constant to this list and the catalogue. It is set only for an error that left the use case, never for a returned
  failure result. A new value for an existing key is added to this list and to
  the catalogue.

---

## 6. Traces

- Wrap each top-level use case in a span through the `Tracer` port. The span
  name is the use case, e.g. `srd5e.d20_test.resolve`. Start it first, `defer
  span.End()` immediately, and pass the returned `ctx` on (§4).
- Add child spans only for steps that are slow or that cross a port (content
  lookup, decoding). Do not span trivial functions.
- Attach the same low-cardinality attributes as the matching metrics.
- On a port failure, call `span.RecordError(err)` and `span.SetStatus` with
  error status, and return the error. Do not swallow it. A failure **result**
  (failed decode, content not found) also sets error status; a normal rule
  outcome (failed check) does not.
- A panic that `srderr.Guard` recovers from a port is converted into a
  `*srderr.PortError` and then follows the port-failure path above. Any other
  panic is a bug: the deferred `span.End()` still runs, and the panic
  propagates.
- srd5e does not manage trace context beyond passing `ctx` (§4). Parent/child
  linkage is the consumer adapter's job.

---

## 7. Logging

### Levels

Use the lowest level that serves the reader. `telemetry.Level` maps to
`log/slog` levels and to OpenTelemetry severity.

| Level | Use for | Examples |
|---|---|---|
| `LevelTrace` | Step-by-step detail useful only when chasing a specific bug. Off by default; MUST be cheap when off. | Each modifier considered and why it was dropped by stacking rules |
| `LevelDebug` | Normal operation details a developer would want while debugging | Use case started with inputs summary; content source selected; result summary |
| `LevelInfo` | Significant lifecycle events, rare. At most a handful per process, never per call. | Content set loaded (name, count); engine configured |
| `LevelWarn` | Something is wrong but handled; the result is still valid but possibly not what was intended | Unknown optional field ignored while decoding; deprecated content format; falling back to a default |
| `LevelError` | An operation failed and the failure is **not** surfaced to the caller as a returned error or failure result (so a log is the only trace of it) | A best-effort step that failed and was skipped |

There is no fatal level; this package never terminates the process.

- **A library MUST NOT log at `LevelInfo` per use case call.** Per-call detail
  is `LevelDebug` or below. Consumers enable it when needed.
- A rule outcome the player simply failed (a failed check, an illegal but
  expected action) is **not** a warning or an error. It is a normal result.
  Log at `LevelDebug` at most.
- `LevelError` is rare in a library. If the failure reaches the caller as a
  result or error, the caller decides whether to log it, so srd5e does not.

### Errors: who records what

Each failure is recorded once, by the right owner, in the right signal:

| Failure | Recorded by | How |
|---|---|---|
| Returned out of the use case as an `error` (port failure, already wrapped with `srderr.WrapPort`) | The use case at whose boundary it leaves srd5e | `span.RecordError` + error status; increment the error counter with an `error.type` attribute. **No log.** |
| Returned out of the use case as a failure result (decode failure, content not found) | The use case that returns it | Span error status; count it with `srd5e.outcome` = `failure` plus a variant attribute defined by the use case's spec. **No `error.type`, no log.** A result the rules merely refuse is a normal outcome (last row), not this one. |
| Returned by an adapter to a use case | The use case (as above), not the adapter | Adapter emits nothing for it |
| Handled inside srd5e and not surfaced (fallback, skipped best-effort step, ignored unknown field) | The code that handled it | `LevelWarn` log (or `LevelError` if the operation could not complete) + counter |
| Programmer error (panic from a `MustX` helper or unreachable branch) | Nobody in srd5e | The panic propagates; it is a bug for the caller to see. The deferred `span.End()` still runs. |
| Normal rule outcome (failed check, illegal action) | The use case | Outcome attribute on the counter and span. Not an error. |

Use cases never call one another (architecture.md §4), so none records another's
signals and an error is counted once, by the use case it left. Attribute values
come from the result value.

### Log intelligently

- **Log at boundaries, not in the middle.** Spans already record the start,
  end and duration of a use case, so do not log those. At most one
  `LevelDebug` summary per use case call. Log at a port call only for what a
  span cannot hold. Do not narrate internal steps.
- **One event per occurrence.** Do not log the same fact at several layers,
  and never log an error that is also returned (see the table above).
- **Do not log what the result already says.** If the result type carries the
  breakdown, the log line refers to the outcome, not a copy of it.
- **Structured, not prose.** Short constant message, details in attributes:
  `log.Log(ctx, telemetry.LevelDebug, MsgResolved, func() []telemetry.Attr {...})`,
  not `fmt.Sprintf("the check by %s against DC %d rolled %d", ...)`.
- **No hot-loop logging.** Never log inside per-modifier or per-combatant loops
  above `LevelTrace`. Aggregate instead: log counts, or record a metric.
- **Cheap when disabled.** Do not build attribute slices eagerly. Attributes are
  supplied as a closure the `Logger` calls only when the level is enabled;
  there is no eager form. Messages are constants, so never need building.
- **Actionable messages.** An error or warning says what failed, with enough
  attributes to find it (context, rule or content identifier, port), and what
  the consumer can do.
- **No secrets or personal data** in messages or attributes.
- **No printing.** No `fmt.Print*`, `println` or standard-library `log` in any
  non-test file (architecture.md §9, A3 and A4). Anything that would be printed
  is a `LevelDebug` or `LevelTrace` log.

---

## 8. Logs, metrics and traces together

- Pick the right signal. Counting or timing something is a **metric**. Seeing
  the path of one operation is a **trace**. Explaining one unusual event is a
  **log**.
- If a log line would be useful in aggregate (you would grep and count it), it
  should also be a metric. Do not make consumers parse logs to count things.
- The same fact uses the same attribute keys across all three signals so a
  consumer can correlate them.

---

## 9. Testing

- Use cases MUST be tested for their emitted telemetry using the
  `telemetrytest` recording doubles: the expected metrics, span names,
  attribute keys and log levels.
- A test MUST assert that a use case returns an identical result with the
  zero-value Noop and with a recording bundle. Table-driven, run with
  `t.Parallel()` where the doubles are not shared.
- Failure tests assert the error is recorded exactly once (span status, one
  counter increment) and is not also logged.
- Pure domain code has no telemetry tests, because it has no telemetry.
- Metric names and attribute keys are part of the public contract. Tests that
  pin them (comparing each constant to its literal catalogue value) guard
  against accidental renames.
- Time-dependent signals use an injected fake clock; no test sleeps.
- Adapters (`slogtelemetry`, Noop, doubles) run the exported contract suites
  from §4 in their own tests, plus their specific behaviour.

---

## 10. Enforcement

The guardrails below are enforced by `internal/archtest` (architecture.md §9) and
the contract suites, which fail `go test`, only where an item names its check
as implemented (A1 to A13 today). Items marked "Not yet enforced" are
specified and rest on review until their check lands. The checks that apply to
observability are:

- **Imports (A1, A3).** `telemetry` imports no srd5e package; no file in the
  module imports an observability or error-tracking SDK (`go.opentelemetry.io`,
  `github.com/getsentry`, `go.uber.org/zap`, `github.com/rs/zerolog`, ...), the
  standard `log` package, and `go.mod` has no `require`. A3 enforces the `log` ban in kernel
  and context packages and any third-party import anywhere (which covers the SDK ban). Confining
  `log/slog` to `telemetry/slogtelemetry` is not yet enforced; review covers it, and test files such as
  `scenarios/example_test.go` may use `log/slog` for examples.
- **Domain files silent (O1). Not yet enforced.** A non-test file that is not a use-case file or `signals.go` (and
  not under `telemetry/`) importing `telemetry`.
- **No globals (A4).** A package-level `var` other than `_` assertions and
  `Err*` sentinels, which covers static loggers, tracers and meters; `init`
  functions.
- **No printing (A4).** `fmt.Print*`, `println`.
- **No literal names (O2). Not yet enforced.** A string literal passed as the name argument of a
  `Tracer.Start`, `Meter` instrument constructor or `Logger` call, or as an
  attribute key, instead of a defined constant.
- **Catalogue matches code (O3). Not yet enforced.** Every signal constant in a `signals.go` file
  appears in `doc/observability/signals.md` with the same value, and every
  catalogue row's Go constant exists. Entries with an empty "Purpose" or
  "Suggested monitoring" cell fail.
- **Closed value sets (O4). Not yet enforced.** A shared attribute value (the sets in §5
  "Attributes") used in code or listed in the catalogue that is outside its
  defined set.
- **No-op shape (O5). Not yet enforced.** A Noop type that has fields, is not usable as a zero
  value, or lives outside `telemetry`.
- **Test-support isolation (A2).** Only `_test.go` files import `*test` packages (enforced). That doubles live only in `*test` packages is not yet enforced; review covers it.
- **Contract suites.** `telemetrytest` suites run against Noop, the recording
  doubles and `slogtelemetry` in their own packages' tests. A new adapter
  without a contract test fails review and the coverage target.

O1 to O5 will be lexical or AST-level like the A-checks, each with violating and
clean in-memory fixtures (`testing/fstest` overlays in `internal/archtest`). They are specified here
and added to the check table in architecture.md §9 when implemented; until a
check exists, review enforces its rule.

Process checks (evaluation performed, `Observability` entry reported) cannot be
fully automated; they are enforced in review (§2).

---

## 11. Anti-patterns (prohibited)

- **Instrumenting the domain.** It breaks purity and couples rules to a backend.
- **Vendor lock-in:** SDK types in ports, or SDK imports anywhere in the module.
- **Package-level loggers or global tracers**, including reading
  `slog.Default()` or an SDK's global provider.
- **Storing `context.Context` in a struct** to emit telemetry later, or
  dropping the `ctx` returned by `Tracer.Start`.
- **Reading the clock** in domain or use-case code to measure duration.
- **Log spam:** per-call `LevelInfo`, logging in tight loops, logging the same
  event at several layers.
- **Log-and-return:** logging an error and then returning it.
- **Double counting:** an error counted by both an adapter and the use case,
  or a use case that calls another use case and so records its signals twice.
- **Wrong severity:** a failed check recorded as an error, or a handled
  fallback logged at `LevelDebug` instead of `LevelWarn`.
- **High-cardinality attributes:** ids, free text or unbounded values.
- **Prose logs** with the data interpolated into the message.
- **Counting by log parsing:** a countable event that exists only as a log line.
- **Unnamed constants:** metric and attribute names scattered as literals.
- **Telemetry that changes behaviour:** a use case whose result, error or
  control flow depends on whether telemetry is enabled.
- **Skipping the evaluation:** finishing a change without assessing
  observability, or reporting "N/A" without a specific reason.
- **Signals nobody can use:** emitting telemetry with no stated purpose or
  monitoring guidance, or leaving the catalogue out of date.
- **Instrument-after-incident:** waiting for a production problem before adding
  the signal that would have exposed it.

---

## Checklist before submitting

- [ ] I evaluated observability and monitoring for this change (§2), thinking
      through all seven questions.
- [ ] The commit/PR and my final summary include an `Observability` entry
      (signals added, changed, removed, and what to monitor) or a specific
      reason none were needed.
- [ ] `doc/observability/signals.md` is updated, with Go constant, purpose and
      suggested monitoring filled in.
- [ ] For a bug fix: I added the signal that would have exposed it, or said why
      none could.
- [ ] Telemetry is produced in use-case files (and adapters only for what
      the use case cannot see), never in domain files.
- [ ] Each failure is recorded once, by its owner, per "Errors: who records
      what"; nothing is logged and also returned.
- [ ] Everything goes through injected ports with Noop defaults; no vendor
      imports, no globals, no stored `ctx`.
- [ ] The `ctx` from `Tracer.Start` is passed to every downstream port call.
- [ ] Anything countable or timeable is a metric, not only a log line.
- [ ] Durations are recorded in seconds through the injected clock.
- [ ] Signal and attribute names are defined constants, correctly prefixed.
- [ ] Attributes are low-cardinality and free of personal data.
- [ ] Log levels are the lowest that serve the reader; nothing noisy at
      `LevelInfo`.
- [ ] No duplicate logging; no log-and-return; no printing.
- [ ] Telemetry is covered by tests using the `telemetrytest` doubles; new
      adapters pass the contract suites.
- [ ] `go test -race -shuffle=on ./...` passes, including `internal/archtest`.

---

## Read receipt

RULE-TAG: `observability-23b3c5a3488a`

This tag proves you read this document in this session. Quote it verbatim as
required by `AGENTS.md`. Do not copy it anywhere else. Whoever edits this
document MUST replace the tag with a fresh random value.
