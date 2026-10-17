# <Spec title>

| | |
|---|---|
| Status | draft / accepted / partially implemented / implemented |
| Context | `<context>`, a kernel package (`dice`, `ability`, `modifier`, `srderr`, `telemetry`), or `package` for a cross-cutting spec that belongs to no single context |
| Roadmap phase | <n> (bare phase number; compound forms such as "Types 2; bulk data 5" are allowed where phases differ) |
| Depends on | <other specs, by file name> |

**Status values.** `draft`: the spec is under review. `accepted`: the spec is agreed and has no or little code. `partially implemented`: code and tests exist for some FR ids, and the spec ends (end of section 8) with a `Not yet implemented: FR-...` line naming the rest, plus an `Implementation status` list of done and not-done ids. `implemented`: every live FR id has code and a test, or is marked review-only, reserved or withdrawn in section 8. Set the value only after auditing the code, in the same change as the code.

## 1. Purpose and scope

What this spec covers, in a few sentences.

**Out of scope:** things a reader might assume are included. Record here any
material that looks like core fifth-edition content but could not be found in
SRD 5.2.1 (`AGENTS.md`, "Licensing and source material").

## 2. Source

- SRD 5.2.1 sections and pages used, e.g. `SRD 5.2.1, "D20 Tests", p. 6`.
- For package-design specs with no SRD source, write `n/a (package design)` and
  list the rule files the spec derives from.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. One behaviour each,
stated as a testable MUST / SHOULD sentence.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | | SRD 5.2.1, "<section>", p. <n> | happy |
| FR-002 | | | unhappy (boundary) |

Cover the unhappy-path checklist in `doc/rules/testing.md` §4 wherever it
applies: boundaries, invalid input, empty and minimal cases, rule interactions,
collaborator failure, missing content, state errors, determinism, immutability.

Rule maths with many cases gets a table, one row per case (label, inputs,
expected, source, path). Each row becomes one table-driven test.

## 5. Model

- Value objects, entities, aggregates (and the root), events, domain services.
- Invariants, and which constructor enforces each.
- Which failures are results, which are `(T, error)` from `NewX`/`ParseX`, and which
  are `MustX` panics (`doc/rules/architecture.md` §6).

## 6. Interface

- Commands, results, use cases and ports; `context.Context` first parameter where I/O occurs.
- Port failure contract: result value for expected absence, `srderr.PortError`
  for infrastructure failure.
- What the package exports (no barrels, no facade).

## 7. Observability

Answers to the seven evaluation questions in `doc/rules/observability.md` §2.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|

Domain-only specs state "silent by design" and say why no signal applies.
Entries here feed `doc/observability/signals.md`.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|

- Levels: unit, application, contract, scenario, architecture.
- Doubles and contract suites required.
- Mutation-testing scope (`doc/rules/testing.md` §8).
- Every FR-id appears in this table; list any deliberately not covered, with
  the reason.

## 9. Open questions and decisions

| Question | Recommendation | Status |
|---|---|---|

Decisions record their reasoning so they are not relitigated.

## 10. Changelog impact

The expected `CHANGELOG.md` entry, and whether it is **BREAKING**.
