# Phase 1 plan: the `d20` package

| | |
|---|---|
| Status | done (implemented in Go) |
| Roadmap phase | 1 ([../roadmap.md](../roadmap.md)) |
| Specs implemented | [../specs/d20-tests.md](../specs/d20-tests.md), [../specs/resolve-d20-test-use-case.md](../specs/resolve-d20-test-use-case.md) (both `implemented`) |
| Depends on | Phase 0: kernel packages, ports, doubles, guardrails |

## 1. Goal

Ship the whole `d20` bounded context as a vertical slice: the pure domain
model of D20 Tests (ability checks, saving throws, attack rolls, DCs and ACs,
Advantage and Disadvantage application, natural 20 and natural 1, success and
failure, rerolls and forced failure), then the package's first use case
(`Resolver.Resolve`) with its commands and result types, its telemetry
contract and the first scenario tests. Phase 1 has no upstream context; the
package imports only kernel packages (`dice`'s value types, `modifier`,
`ability`, `srderr`, `telemetry`) and declares its own consumer-side `Roller`
interface.

The package is the **reference template for every later context** (constructor
injection with options, results as values, error handling, telemetry shape).

Both specs carry settled decisions (D-01 to D-22 in d20-tests.md, plus the
ownership table in resolve-d20-test-use-case.md). This plan sequences and
traces them and changes none. Where this plan and a spec disagree, the spec wins.

## 2. What already exists (Phase 0, reused as-is)

| Asset | Where | Used for |
|---|---|---|
| `modifier.Modifier`, `modifier.RollMode` (`Resolve`, `DiceCount`, `Select`), `Advantage`, `Disadvantage` | `modifier/` | `RollModifiers`, mode resolution, kept-die selection |
| `ability.Ability` | `ability/` | the ability label on commands and the `srd5e.ability` attribute |
| `dice.D20`, `dice.Expression` | `dice/` | the `1d20` or `2d20` request |
| `dice.Roller`, `srderr.PortError`, `srderr.Guard` | `dice/`, `srderr/` | the one dice request per rolling call and failure wrapping |
| `telemetry.Telemetry` and no-ops | `telemetry/` | telemetry, defaulting to no-op |
| `dicetest` and `telemetrytest` doubles | `dice/dicetest`, `telemetry/telemetrytest` | tests; no new double was needed |
| Architecture tests | `internal/archtest/` | enforce the dependency allow-list |

## 3. Scope fence

**In scope:** every live requirement of the two specs: d20-tests.md FR-009 to
FR-036, FR-041, FR-050 to FR-052, FR-055, FR-058 to FR-073; resolve-d20-test-use-case.md
FR-001, FR-002, FR-005 to FR-018, FR-020 to FR-032, FR-035 to FR-041, FR-042 to
FR-047, FR-049, FR-051 to FR-053, FR-055 to FR-062 (ids withdrawn or moved are
listed in each spec's changelog table and have no row here).

**Out of scope** (the specs' own lists):

- Death Saving Throws, damage, extra Critical Hit dice, contests, group and
  passive checks, automatic success or failure on natural 20 or 1 for checks
  and saves (the SRD states it for attack rolls only, p. 7).
- Heroic Inspiration as a use case; only the reroll operations on `Roll`.
- Deriving ability modifiers, Proficiency Bonus, AC, which skill uses which
  ability, sources of Advantage or Disadvantage: callers supply finished values.
- Any `content`, `character`, `combat` or `toolbox` work; any bundled data;
  any change to the kernel packages.

## 4. Delivery sequence

Three steps, each independently green. Each step is a reviewable PR.

| Step | Ships | Requirements | Status |
|---|---|---|---|
| 1 | Domain values: `Roll` (`NewRoll`, reroll and replace-a-die), `RollModifiers`, `DifficultyClass`, `ArmorClass`, `Kind`, `Outcome`, the package `doc.go` | d20-tests FR-009 to FR-015, FR-017, FR-018, FR-021, FR-022, FR-055, FR-058, FR-059, FR-061, FR-067, FR-068 (D-02: no range check) | done |
| 2 | Domain results and resolution: `CheckResult`, `AttackRollResult` (critical range), forced-failure result, `AdjustedBy`, the pure rules, `ForcedFailureReason` | d20-tests FR-016, FR-019, FR-020, FR-023 to FR-036, FR-041, FR-050 to FR-052, FR-060, FR-062 to FR-073 (D-22: critical range is an input) | done |
| 3 | Use case: `Command` variants, `CheckAttempt` and `SaveAttempt` variants, `Resolver` (`NewResolver`, `WithTelemetry`, `WithClock`), `Result` variants, signal constants, telemetry, scenarios, signal catalogue rows | resolve FR-001, FR-002, FR-005 to FR-007, FR-010, FR-018, FR-020 to FR-032, FR-035 to FR-041, FR-045 to FR-047, FR-049, FR-051 to FR-053, FR-055 to FR-062 (FR-008, FR-009, FR-011 to FR-017, FR-043 delegate to the Step 2 rules) | done |

Process contract for every step (from `AGENTS.md` and the rule files):

- Every test cites its requirement: a `Requirement:` doc comment naming the spec
  path and FR id (`Requirement: per case` on table rows), or an SRD rule
  citation ([../rules/testing.md](../rules/testing.md) §5).
- `just check` is green before review: gofmt, `go vet`, golangci-lint,
  `go test -race -shuffle=on -cover ./...`, `go mod tidy`, architecture tests.
- Statement coverage of the package is at least 90 percent; uncovered lines
  have a stated reason ([../rules/testing.md](../rules/testing.md)).
- Mutation testing with `gomu`, only inside Docker with `--cpus` and
  `--workers` capped at a quarter of the host's processors
  ([../rules/testing.md](../rules/testing.md) §8). Survivors are fixed with
  better tests unless provably equivalent.
- `CHANGELOG.md` entry under `Unreleased`.
- The PR description carries `Testing` and `Observability` entries
  ([../../.github/pull_request_template.md](../../.github/pull_request_template.md)).

## 5. Step 1: domain values (done)

Files in `d20/`: `roll.go`, `roll_modifiers.go`, `kind.go` (kinds,
outcomes, `DifficultyClass` constants, `ArmorClass`),
`error.go` (the typed `InvalidError`, which matches `srderr.ErrInvalidInput`),
`doc.go`.

Tests: `roll_test.go` (table of reroll and replace cases),
`roll_modifiers_test.go` (shape and immutability), `kind_test.go` (enum
strings, validity, DC constants), `error_test.go`.

## 6. Step 2: domain results and rules (done)

Files: `check_result.go`, `attack_roll_result.go`, `rules.go` (the pure
resolution of checks and attack rolls, natural 20 and natural 1, critical
range), `forced_failure_result.go`.

Tests: `rules_test.go` (resolution tables run for both check kinds and for
attack rolls, including Advantage and Disadvantage kept dice and critical
ranges) and the re-resolution cases of `AdjustedBy`.

## 7. Step 3: the `Resolver` use case (done)

Files: `command.go`, `attempt.go`, `result.go`, `resolver.go` (one
`modifier.Resolve`, one `dice` request sized by the mode, no request for forced
failures), `signals.go` (name and attribute-key constants).

Also: `doc/observability/signals.md` rows for the signals, and
`scenarios/d20_test.go` (public API only).

Tests: `resolver_test.go` (end-to-end case table with `dicetest.FixedRoller`;
roller failures through `dicetest.CallbackRoller`; determinism),
`telemetry_test.go` (recording doubles: span, counters, histograms, failure
recorded once, signal names pinned), `decoupling_test.go` (the package depends
on its own `Roller` interface, not on `dice` adapters), `example_test.go`.

## 8. Verification commands

```sh
just check
go test -race -shuffle=on -coverprofile=coverage.txt ./d20/... && go tool cover -func=coverage.txt | tail -n 1
```

Mutation testing (never on the host; the justfile recipe runs gomu inside Docker
with `--cpus` and `--workers` capped at a quarter of the processors and the
threshold flags, see [../rules/testing.md](../rules/testing.md) §8):

```sh
just mutate ./d20
```

## 9. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Telemetry contract drift (this is the template for later use cases) | Tests pin the literal signal names and attribute keys; review against [../rules/observability.md](../rules/observability.md) §5 to §7 |
| Catalogue and constants disagree | Constants and `signals.md` rows land in the same change |
| Kernel API mismatch discovered mid-implementation | Stop and raise it; do not fork the rule into `d20` |

## 10. Out-of-scope observations (recorded, not acted on)

- resolve-d20-test-use-case.md §9 flags sibling follow-ups: class-barbarian.md
  must stop calling the resolver use case (compose the pure rules instead), and
  social-interaction.md must drop the roll mode for forced failures. Both are
  later-phase spec edits.

## 11. Done

Phase 1 is done: both specs read `implemented`, the signals are in the
catalogue, scenario tests exist and the changelog carries the entries.
