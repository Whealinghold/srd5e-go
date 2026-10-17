# Signal catalogue

Every metric, span and log event srd5e emits. Must match the code (see
[observability rules](../rules/observability.md), §2 and §10). Consumers use
this to build dashboards and alerts; srd5e cannot run alerts itself.

Add a row for each signal in the same change as its Go constant. A signal
whose code has not landed keeps its planned row in its owning spec
(`doc/specs/`), not here.

## Conventions

- **Name** is the exact string value of the exported constant. It is the wire
  name and stays `srd5e.<context>.<noun>` with snake_case segments, even where
  the Go package name has no underscore (the `d20` package emits
  `srd5e.d20_test.*`, named after the SRD term "D20 Test"). The Go constant
  carries the identifier: `Name` is a column below.
- **Constants** live in a `signals.go` file of the emitting package (for
  example `d20.SpanResolve`), or in `telemetry` for keys shared across
  contexts (`telemetry.AttrOutcome`, `telemetry.AttrErrorType`).
- **Emitted by** names the Go entry point (`package.Type.Method`).
- Attribute values listed here are the closed sets and are binding. They are
  checked in review and by tests that pin them; nothing yet rejects an
  out-of-set value mechanically. O3/O4 (`doc/rules/observability.md` section
  10) will mechanise it.
- Durations are seconds (`s`, float64); counts use UCUM annotations.

## d20

Go names: constants in package `d20` (`signals.go`); entry point
`d20.Resolver.Resolve(ctx, ...)`.

| Name | Go constant | Type | Unit | Attributes | Emitted by | Purpose | Suggested monitoring |
|---|---|---|---|---|---|---|---|
| `srd5e.d20_test.resolve` | `d20.SpanResolve` | Span | | `srd5e.d20_test.kind` (`ability_check`, `saving_throw`, `attack_roll`), `srd5e.ability` (`strength`, `dexterity`, `constitution`, `intelligence`, `wisdom`, `charisma`), `srd5e.d20_test.roll_mode` (`normal`, `advantage`, `disadvantage`; absent for a forced failure), `srd5e.outcome` (`success`, `failure`), `srd5e.d20_test.critical_hit` (`true`, `false`; attack rolls only), `srd5e.d20_test.forced_failure_reason` (`chosen_to_fail`, `automatic`; forced failures only), `srd5e.d20_test.roller_failure` (`unavailable`, `invalid_result`; error spans only); error status on failure | `d20.Resolver.Resolve` | Trace one D20 Test; the only signal that carries the recorded error, including a recovered roller panic (`srderr.Guard`) | Inspect error-status spans when `errors` alerts, or when a span has error status without an `errors` increment. Do not alert on span error rate as well as on `errors`; they overlap for port failures |
| `srd5e.d20_test.resolved` | `d20.MetricResolved` | Counter | `{test}` | same as the span | `d20.Resolver.Resolve` | Throughput and outcome mix | Over at least 500 attack rolls in `normal`, `critical_hit=true` outside roughly 2 to 8 percent suggests a biased roller (expected 5 percent; widened ranges are applied by callers). Covers direct host calls only: no other use case calls `Resolve`, so combat attack rolls are counted under those use cases' own signals. A `forced_failure_reason`=`automatic` share near 100 percent for one ability, sustained, suggests misapplied caller condition wiring rather than gameplay. Alert on a drop to zero throughput against the consumer's baseline |
| `srd5e.d20_test.duration` | `d20.MetricDuration` | Histogram | `s` | `srd5e.d20_test.kind` | `d20.Resolver.Resolve` | Latency, including failed calls; dominated by the injected roller | p99 above 5 times the consumer's baseline for a sustained window; expect microseconds with an in-memory roller |
| `srd5e.d20_test.modifiers` | `d20.MetricModifiers` | Histogram | `{modifier}` | `srd5e.d20_test.kind` | `d20.Resolver.Resolve` | Size of the modifier set callers pass (minimum 1) | p99 above 20 sustained suggests a caller accumulating modifiers; dashboard otherwise |
| `srd5e.d20_test.natural` | `d20.MetricNatural` | Histogram | `{face}` | `srd5e.d20_test.kind` | `d20.Resolver.Resolve` | The d20 face each rolled call used (1 to 20), so an unfair or stuck injected roller is visible on every face, not only on the natural 20 | Over at least 500 rolled calls in `normal`, any face's share outside roughly 2 to 8 percent (expected 5) points at the roller; a share pinned at one face means a stuck roller. Compare with the `critical_hit` monitor on `resolved`, which sees only the 20 face |
| `srd5e.d20_test.errors` | `d20.MetricErrors` | Counter | `{error}` | `srd5e.d20_test.kind`, `error.type` (`dice_roller_error`), `srd5e.d20_test.roller_failure` (`unavailable`, `invalid_result`) | `d20.Resolver.Resolve` | Port failures leaving the use case, counted once, with why the roller failed | Alert on any non-zero rate sustained over 5 minutes. Split the alert by `srd5e.d20_test.roller_failure`: `unavailable` is infrastructure (the dice source is down; page ops), `invalid_result` is the host's adapter answering a different expression (a contract breach; file a bug). The span of the same call holds the error |

`error.type` is `srderr.ErrorType` of the error that left the use case:
`<port>_error` for `*srderr.PortError`, so a roller failure is
`dice_roller_error`.
