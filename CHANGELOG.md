# Changelog

All notable changes to this module are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the module
follows [Semantic Versioning](https://semver.org/) (see
[doc/rules/go.md](doc/rules/go.md) for the compatibility promise).

## Unreleased

This is the Go port of the earlier Dart implementation of srd5e. It keeps the
rules and the specs; the API follows Go conventions instead of Dart ones.

### Changed

- Compatibility promise (`doc/rules/go.md` section 9): exported-field structs
  must be built with keyed literals and new fields can break comparability;
  tightening a `*test` contract suite is a minor-version change listed under
  Changed; new struct fields are excepted from the non-breaking additions rule.
- Docs: package comments use godoc links; concurrency requirements for
  `dice.Roller`, the telemetry ports, `d20.Resolver`, `WithTelemetry` and
  `WithClock` are stated precisely; README gains a pkg.go.dev badge, a Go 1.24
  line and a pasteable first snippet.
- Internal: `scenarios` is a tests-only directory (`doc.go` removed), no longer
  an importable package. CI pins golangci-lint v2.15.0 and the lint config
  enables more linters.
- **BREAKING:** the `d20test` package is renamed `d20` (import path
  `github.com/Whealinghold/srd5e-go/d20`). The `test` suffix is reserved for
  test-support packages (`dicetest`, `telemetrytest`). Telemetry wire names
  (`srd5e.d20_test.*`) are unchanged. To satisfy the no-stutter rule, `D20Roll`,
  `NewD20Roll` and `MustD20Roll` are renamed `Roll`, `NewRoll` and `MustRoll`.

### Added

- `dice/randroller`: an unseeded `dice.Roller` over `math/rand/v2` that is safe
  for concurrent use and whose zero value works, for servers sharing one roller
  between goroutines.
- New runnable examples: `d20.NewResolver`, `AbilityCheckCommand`,
  `AttackRollCommand`, `dicetest.NewFixedRoller` and a root `Example`;
  `Example_telemetry` now records and prints real signals.
- `srderr`: the shared error vocabulary. `ErrInvalidInput`, `ErrPortPanic` and
  `ErrContractViolation` sentinels, `PortError` with `WrapPort`,
  `ContractViolation` (carrying a `ContractError`), `ErrorType`, `Quote` for
  escaping untrusted text in error details, and `Guard` and `GuardValue`, which turn a
  port panic into a `PanicError` inside a `PortError`.
- `dice`: `Die` (`NewDie`, `D3` to `D100`), `Expression` (`NewExpression`,
  `ParseExpression` for text such as `1d20` and `2d8 + 2`, `Minimum`,
  `Maximum`, `Average`, `Double`, `Notation`), `DieRoll`, `Result`
  (`DiceTotal`, `Total`, `SumOfHighest`, `SumOfLowest`), the `Roller` port,
  `Roll` (which wraps a roller error, panic or wrong answer in a
  `*srderr.PortError`), `SeededRoller` over a caller-supplied `*rand.Rand`,
  and the typed `InvalidError` with a `Reason`. `MustX` helpers for constants.
- `dice/dicetest`: `FixedRoller` (`NewFixedRoller`, `NewMinimumRoller`,
  `NewMaximumRoller`), `CallbackRoller`, and the exported contract suites
  `RunRollerContract` and `RunFailureContract`.
- `modifier`: `Modifier` (`New`, `Add`, `Sum`, `ApplyTo`), the
  `RollCircumstance` variants `Advantage` and `Disadvantage`, and `RollMode`
  with `Resolve` (Advantage and Disadvantage do not stack and cancel as a
  whole), `DiceCount`, `Select` and `ParseRollMode`.
- `ability`: `Ability` (the six abilities, `ParseAbility`, `All`), `Score`
  (1 to 30, `NewScore`, derived `Modifier` from the SRD's Ability Modifiers
  table), `Modifier` (independent of `modifier.Modifier`) and `Scores` (one
  score per ability, `NewScores` reporting every problem at once, `With`).
- `telemetry`: the `Logger`, `Tracer` and `Meter` ports bundled as
  `Telemetry` (zero value is a silent no-op, `WithDefaults`), `Attr`
  constructors, `Level`, `Status`, `Descriptor`, `ValidName` and `StartTimer`.
- `telemetry/telemetrytest`: recording doubles (`Recorder`, `Logger`, `Meter`,
  `Tracer`, `Clock`) that report misuse as violations, and the contract suites
  `RunLoggerContract`, `RunTracerContract` and `RunMeterContract`.
- `telemetry/slogtelemetry`: an adapter from `log/slog` to `telemetry.Logger`,
  standard library only.
- `d20` (D20 Tests): ability checks, saving throws and attack rolls.
  `Resolver` (`NewResolver` with `WithTelemetry` and `WithClock`) takes a
  `Command` (`AbilityCheckCommand`, `SavingThrowCommand`, `AttackRollCommand`)
  and returns a `Result`: `RolledCheck`, `RolledAttack` or `ForcedFailure`. A
  failed test is a value, not an error. Domain types: `Roll`
  with reroll and replace-a-die operations, `RollModifiers`, `DifficultyClass`,
  `ArmorClass`, `Outcome`, `CheckResult`, `AttackRollResult` (natural 20 and
  natural 1 on attack rolls, critical hit, critical range) and forced
  failures. The consumer-side `Roller` interface keeps the context free of the
  `dice` implementations.
- Telemetry signals `srd5e.d20_test.resolve` (span), `.resolved`, `.duration`,
  `.modifiers`, `.natural` and `.errors`, with their attribute keys, as
  exported constants in `d20` and catalogued in
  [doc/observability/signals.md](doc/observability/signals.md).
- `internal/archtest`: architecture tests that enforce the dependency
  allow-list, naming rules and documentation rules as part of `go test ./...`.
- `scenarios`: cross-package tests through the public API only.
- Fuzz tests for every parser, `Example*` functions as testable docs, a
  committed `.golangci.yml`, a `Makefile` (`make check`) and a CI workflow.
