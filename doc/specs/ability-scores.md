# Ability scores and modifiers

| | |
|---|---|
| Status | implemented |
| Context | `ability` (kernel package) |
| Roadmap phase | 0 |
| Depends on | `errors-and-results.md` |

Ownership: per `shared-types-and-ownership.md` FR-001 this spec is the single
definition of `ability.Ability`, `ability.Score`, `ability.Modifier`,
`ability.Scores` and the score rejection error (`ability.InvalidError` with
`ability.Reason`); FR-001 there lists all of them (Q-2, Q-5). Other specs
reference these types by file and id and define none of them. Type phase 0
(`shared-types-and-ownership.md` FR-014). `errors-and-results.md` is the
convention dependency (result, error and `MustX` rules).

## 1. Purpose and scope

The six abilities, the ability score a creature has in each, the ability
modifier derived from a score, and a complete set of six scores. `d20`
uses the modifier of an ability, `character` stores the scores, `combat` reads
them. This spec defines the types, the valid score range, the score-to-modifier
derivation and the complete set.

**Out of scope:**

- How scores are determined or raised (Character Creation, feats, features):
  `character` and `content`.
- The adventurer cap of 20 "unless a feature says otherwise" (p. 5): a character
  rule. The kernel enforces only 1 to 30. Character-level caps and rejections use
  types named in `character-aggregate.md`, not `ability.InvalidError`.
- A score of 0: the SRD says only "If an effect reduces a score to 0, that effect
  explains what happens" (p. 5), so 0 is not a valid `Score`. The zero value of
  `Score` is the explicitly invalid score.
- Objects: "An object lacks ability scores unless a rule assigns scores to the
  object" (p. 178). The kernel has no "no scores" value; a consumer modelling an
  object holds no `Scores`.
- Using a modifier in a D20 Test, Proficiency Bonus, Armor Class, Carrying
  Capacity and other derived rules (`d20`, `character`).
- The generic `modifier.Modifier` (owned by `modifiers-and-advantage.md`) and
  `RollModifiers`, whose ability slot is a plain `modifier.Modifier`
  (`d20-tests.md` section 4.3, D-16). Callers (`character`, `combat`) convert
  with `modifier.New(abilityModifier.Value())`. See
  `shared-types-and-ownership.md` FR-003.
- `Skill` and its `Ability` link (`content`, `shared-types-and-ownership.md`
  FR-005).
- The "Score Measures" text, the "Meaning" column (p. 5) and the abbreviations
  Str., Dex., Con., Int., Wis., Cha. (p. 176): display text; the domain holds no
  display strings (`doc/rules/architecture.md` §3).

## 2. Source

- SRD 5.2.1, "The Six Abilities", p. 5 (Ability Descriptions table).
- SRD 5.2.1, "Ability Scores", p. 5 (range 1 to 20, up to 30 for some monsters;
  Ability Scores table).
- SRD 5.2.1, "Ability Modifiers", p. 5 ("a modifier that you apply whenever you
  make a D20 Test with that ability"), and the Ability Modifiers table, p. 6.
- SRD 5.2.1, "Round Down" sidebar, p. 5 (consistent with the table; does not
  decide the negative case).
- SRD 5.2.1, "Rules Glossary", "Ability Score and Modifier", p. 176.
- SRD 5.2.1, "Rules Glossary", "Breaking Objects", "No Ability Scores", p. 178
  (out-of-scope note only).

PDF page index equals the printed page number (checked on pages 5 and 176).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| ability | `ability.Ability` (enum, `int`) | Strength, Dexterity, Constitution, Intelligence, Wisdom, Charisma. "All creatures ... have six abilities" (p. 5). Zero value is invalid. |
| ability score | `ability.Score` (value object) | Integer 1 to 30 (p. 5). Zero value is invalid (`IsZero`). |
| ability modifier | `ability.Modifier` (value object) | Derived from a score by the Ability Modifiers table (p. 6). Integer -5 to +10. Not `modifier.Modifier`. |
| (the six scores of one creature) | `ability.Scores` (value object) | "A creature has six ability scores" (p. 176): exactly one `Score` per `Ability`. |

## 4. Requirements

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `Ability` MUST have exactly six values, in this order: `Strength`, `Dexterity`, `Constitution`, `Intelligence`, `Wisdom`, `Charisma`. `All()` returns them in that order as a fresh slice. | SRD 5.2.1, "The Six Abilities", p. 5 | happy |
| FR-002 | `NewScore` MUST accept every integer from 1 to 30 inclusive and `Score.Value()` MUST return it. | SRD 5.2.1, "Ability Scores", p. 5 | happy |
| FR-003 | `NewScore` MUST accept 1, the lowest score ("the lowest a score can normally go"). | SRD 5.2.1, "Ability Scores", p. 5 | unhappy (boundary) |
| FR-004 | `NewScore` MUST accept 30, the highest score ("the highest a score can go"). | SRD 5.2.1, "Ability Scores", p. 5 | unhappy (boundary) |
| FR-005 | `NewScore(0)` MUST return a non-nil error and the zero `Score`. | SRD 5.2.1, "Ability Scores", p. 5 | unhappy (boundary) |
| FR-006 | `NewScore(31)` MUST return a non-nil error and the zero `Score`. | SRD 5.2.1, "Ability Scores", p. 5 | unhappy (boundary) |
| FR-007 | `NewScore` of a negative integer MUST return a non-nil error and the zero `Score`. | SRD 5.2.1, "Ability Scores", p. 5 | unhappy (invalid input) |
| FR-008 | `NewScore` MUST return a valid `Score` and a nil error for any integer from 1 to 30, without panicking. | SRD 5.2.1, "Ability Scores", p. 5 | happy |
| FR-009 | `NewScore` MUST return an `*InvalidError` whose `Reason` is `ReasonBelowMinimum` for any integer below 1, without panicking. | SRD 5.2.1, "Ability Scores", p. 5 | unhappy (invalid input) |
| FR-010 | `NewScore` MUST return an `*InvalidError` whose `Reason` is `ReasonAboveMaximum` for any integer above 30, without panicking. | SRD 5.2.1, "Ability Scores", p. 5 | unhappy (invalid input) |
| FR-011 | Two `Score`s with the same value MUST be equal with `==`; different values MUST NOT be equal. `Score` is comparable (unexported field, value receivers). | `doc/rules/architecture.md` §3 (value object) | happy |
| FR-012 | `Score.Modifier()` MUST return the `Modifier` given by the Ability Modifiers table for every score from 1 to 30 (table in this section). | SRD 5.2.1, "Ability Modifiers", p. 5, table p. 6 | happy |
| FR-013 | For odd scores below 10 the modifier MUST be the table's value (floor division), not the truncating-division value: score 1 gives -5 (not -4), score 3 gives -4 (not -3), score 9 gives -1 (not 0). | SRD 5.2.1, Ability Modifiers table, p. 6 (the "Round Down" sidebar, p. 5, is consistent with it but does not decide the negative case) | unhappy (rule interaction) |
| FR-014 | Score 10 MUST give a modifier of +0. This and FR-015 are the labelled boundary rows of the FR-012 table, kept so a mutation of the band edges around zero fails a named test. | SRD 5.2.1, Ability Modifiers table, p. 6 | unhappy (boundary) |
| FR-015 | Score 1 MUST give the lowest modifier, -5. | SRD 5.2.1, Ability Modifiers table, p. 6 | unhappy (boundary) |
| FR-016 | Two `Modifier`s with the same value MUST be equal with `==`; different values MUST NOT be equal. | `doc/rules/architecture.md` §3 (value object) | happy |
| FR-017 | `Modifier.Value()` MUST return the signed integer, so a consumer can apply it to a D20 Test. | SRD 5.2.1, "Ability Modifiers", p. 5 ("a modifier that you apply whenever you make a D20 Test with that ability"); "Ability Score and Modifier", p. 176 ("Add the modifier when you make a D20 Test with the corresponding ability") | happy |
| FR-018 | `NewScores(map[Ability]Score)` MUST accept a map holding a valid `Score` for each of the six abilities and return the `Scores` and a nil error. | SRD 5.2.1, "Ability Score and Modifier", p. 176 | happy |
| FR-019 | `NewScores` MUST return a non-nil error when any ability has no score (or a zero `Score`), including an empty or nil map. It reports every missing ability at once through `errors.Join` of `*InvalidError` with `ReasonMissingScore`, and an invalid map key as `ReasonUnknownAbility`. | SRD 5.2.1, "Ability Score and Modifier", p. 176 | unhappy (invalid input) |
| FR-020 | `Scores.Score(Ability)` MUST return the `Score` held for that ability, for all six abilities (the zero `Score` for an invalid `Ability`). | SRD 5.2.1, "Ability Score and Modifier", p. 176 | happy |
| FR-021 | `Scores.With(ability, score)` MUST return a new `Scores` that has the new score for that ability and the other five unchanged, and MUST NOT alter the receiver. It returns an `*InvalidError` (`ReasonUnknownAbility` or `ReasonMissingScore`) and the zero `Scores` for an invalid ability or a zero score. | `doc/rules/architecture.md` §3 (immutability) | unhappy (immutability) |
| FR-022 | A `Scores` MUST NOT change when the map passed to `NewScores` is mutated afterwards (the six scores are copied into a fixed array). | `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-023 | Two `Scores` with the same score for every ability MUST be equal with `==`; differing in any one ability MUST NOT be equal. | `doc/rules/architecture.md` §3 (value object) | happy |
| FR-024 | `Scores.Modifier(Ability)` MUST return the `Modifier` of the score held for that ability (equal to `Score(a).Modifier()`), for all six abilities. | SRD 5.2.1, "Ability Score and Modifier", p. 176 ("each of which has a corresponding modifier") | happy |
| FR-025 | `ability.MinScore` MUST be 1 and `ability.MaxScore` MUST be 30, exported constants other contexts reuse instead of repeating the bounds. | SRD 5.2.1, "Ability Scores", p. 5 | happy |
| FR-026 | `InvalidError` and `Reason` (`ReasonUnknownAbility`, `ReasonBelowMinimum`, `ReasonAboveMaximum`, `ReasonMissingScore`, with `String()`) MUST be defined only here and exported once. No other spec defines a type of that name or re-exports its values. Character-level rejections (for example a score above 20) use differently named types owned by `character-aggregate.md`. | `shared-types-and-ownership.md` FR-001, FR-021 | unhappy (name collision) |
| FR-027 | `ability.Modifier` MUST expose `Value()`; package `ability` MUST NOT import package `modifier`, and `modifier` MUST NOT import `ability`. Callers convert with `modifier.New(abilityModifier.Value())` (`character` and `combat` own that call). `RollModifiers.ability` is a plain `modifier.Modifier`, so `d20` tests need no ability scores. | `shared-types-and-ownership.md` FR-001, FR-003; `d20-tests.md` section 4.3 and D-16 | happy |
| FR-028 | The error returned by `NewScore` for a rejected integer MUST be an `*InvalidError` (`errors.As`-able) carrying a typed `Reason`, and MUST satisfy `errors.Is(err, srderr.ErrInvalidInput)`. Tests assert `Reason` and the sentinel, never the message text (`InvalidError.Detail` is for logs). | `errors-and-results.md` FR-027 | unhappy (invalid input) |
| FR-029 | `NewScore` MUST NOT panic for any `int`, including the minimum and maximum integers. | `errors-and-results.md` FR-025 | unhappy (invalid input) |
| FR-030 | `MustScore(n)` MUST panic if and only if `NewScore(n)` returns an error, and the panic value MUST be that error (`errors.As` works on the recovered value). | `errors-and-results.md` FR-026 | unhappy (rule conflict) |
| FR-031 | Rejection of a score is a typed error, not a closed result: there is no `ValidScore` / `InvalidScore` variant pair (a scalar constructor returns one typed rejection reason, `errors-and-results.md` FR-035). | `errors-and-results.md` FR-018, FR-035 | happy |
| FR-032 | `ParseAbility(s)` MUST accept the lowercase SRD name of each ability case-insensitively and return the `Ability`; any other string MUST return an `*InvalidError` with `ReasonUnknownAbility` and the zero `Ability`, never a panic. `Ability.String()` returns the lowercase name ("unknown" for an invalid value) and round-trips through `ParseAbility`. | SRD 5.2.1, "The Six Abilities", p. 5 | unhappy (invalid input) |
| FR-033 | `Ability.Valid()` MUST be true for exactly the six values of FR-001 and false for the zero value and any other integer. | SRD 5.2.1, "The Six Abilities", p. 5 | unhappy (boundary) |

### Ability Modifiers table cases (FR-012 to FR-015)

One row per band of the Ability Modifiers table (SRD 5.2.1, p. 6). Each band
becomes two table-driven cases (lower and upper score), or one for a
single-score band, so all 30 scores are checked.

| Label | Scores | Modifier | Source | Path |
|---|---|---|---|---|
| lowest score | 1 | -5 | p. 6 | unhappy (boundary) |
| band 2-3 | 2, 3 | -4 | p. 6 | happy |
| band 4-5 | 4, 5 | -3 | p. 6 | happy |
| band 6-7 | 6, 7 | -2 | p. 6 | happy |
| band 8-9 | 8, 9 | -1 | p. 6 | unhappy (boundary) |
| human average | 10, 11 | +0 | p. 6 | unhappy (boundary) |
| band 12-13 | 12, 13 | +1 | p. 6 | happy |
| band 14-15 | 14, 15 | +2 | p. 6 | happy |
| band 16-17 | 16, 17 | +3 | p. 6 | happy |
| band 18-19 | 18, 19 | +4 | p. 6 | happy |
| band 20-21 | 20, 21 | +5 | p. 6 | happy |
| band 22-23 | 22, 23 | +6 | p. 6 | happy |
| band 24-25 | 24, 25 | +7 | p. 6 | happy |
| band 26-27 | 26, 27 | +8 | p. 6 | happy |
| band 28-29 | 28, 29 | +9 | p. 6 | happy |
| highest score | 30 | +10 | p. 6 | unhappy (boundary) |

The table is the sole authority. `floor((score - 10) / 2)` is not in SRD 5.2.1;
it is an implementation shortcut verified against all 30 rows. The code uses an
arithmetic right shift (`(score - 10) >> 1`), which floors; Go's `/` truncates
toward zero and would be wrong for odd scores below 10. If the shortcut and the
table disagree, the table wins.

### Withdrawn ids

None.

## 5. Model

- `Ability`: `int` enum of the six abilities, starting at 1 so the zero value is
  invalid (FR-001, FR-033). `String`, `Valid`, `All`, `ParseAbility` (FR-032).
- `Score`: immutable, comparable value object wrapping an `int` in 1 to 30 in an
  unexported field, with constants `MinScore` (1) and `MaxScore` (30) (FR-025).
  `NewScore(int) (Score, error)` validates in every build (FR-002 to FR-010,
  FR-028, FR-029). `MustScore` panics with the error `NewScore` returns, for
  programmer error and constants only (FR-030). `Value`, `IsZero`, `Modifier`
  (FR-012 to FR-015), `String`. The zero `Score` is invalid and its `Modifier`
  is the zero `Modifier`.
- `Modifier`: immutable value object wrapping an `int` in -5 to +10, method
  `Value()`, `String()` (signed, `+2`). It has no exported constructor; both
  `Score` and `Modifier` live in package `ability` so `Score.Modifier` is the
  only way to build one, and no outside code can build an out-of-range
  modifier.
- `Scores`: immutable, comparable value object (a fixed array of six `Score`),
  exactly one `Score` per `Ability`. `NewScores(map[Ability]Score)` enforces
  completeness (FR-019) and copies the input (FR-022). Members: `Score`
  (FR-020), `Modifier` (FR-024), `With` (FR-021), `IsZero`, `String`. It
  exposes no collection, so `errors-and-results.md` FR-021 does not apply;
  consumers iterate `All()`.
- Placement: `Scores` is in the kernel because `character` and `combat` both
  need the complete set and neither may depend on the other
  (`doc/rules/architecture.md` §2). `content` monsters hold their own
  `AbilityBlock` (`content-monsters.md` FR-018). `Ability` and `Modifier` are in
  the kernel because `d20` has no upstream context.
- Invariants: score in 1 to 30; modifier in -5 to +10; six scores present.
- Failures: every failure is untrusted input failing validation, so
  `NewScore`, `NewScores`, `Scores.With` and `ParseAbility` return
  `(T, error)` where the error is a `*InvalidError` (or an `errors.Join` of
  them for `NewScores`) that is `errors.Is` `srderr.ErrInvalidInput` and
  carries a typed `Reason` (FR-028, `errors-and-results.md` FR-035), data only,
  no display text (`errors-and-results.md` FR-020). `InvalidError.Is` matches a
  target `*InvalidError` with the same `Reason`. A panic occurs only in
  `MustScore`. The package defines no use case, so the `<Op>Result` rule
  (`shared-types-and-ownership.md` FR-017) does not apply. No domain events.
- All types are immutable, pure and synchronous, with value receivers and no
  `ctx`. Package `ability` imports only `srderr`.

## 6. Interface

- No use cases, commands, ports or I/O; other contexts use these types directly
  in domain code (`doc/rules/architecture.md` §1).
- Package `ability` depends only on the kernel package `srderr`
  (`shared-types-and-ownership.md` FR-013).
- Exports from `github.com/Whealinghold/srd5e-go/ability`: `Ability` (consts
  `Strength` .. `Charisma`, `All`, `ParseAbility`, `Valid`, `String`), `Score`
  (`MinScore`, `MaxScore`, `NewScore`, `MustScore`, `Value`, `IsZero`,
  `Modifier`), `Modifier` (`Value`), `Scores` (`NewScores`, `Score`, `Modifier`,
  `With`, `IsZero`), `InvalidError`, `Reason` and its constants. Every exported
  identifier has a doc comment starting with its name; the package doc is in
  `ability/doc.go`.
- Layout: package `ability` with `ability.go` (`Ability`), `score.go` (`Score`
  and `Modifier`), `scores.go` (`Scores`), `error.go` (`InvalidError`,
  `Reason`), `doc.go`.

## 7. Observability

**Silent by design.** Pure kernel value types stay free of observability
(`doc/rules/observability.md` §3). The seven questions:

1. **What can go wrong?** An out-of-range score or incomplete set: an
   `*InvalidError` returned for untrusted input (or a `MustScore` panic for a
   programmer error).
2. **How would an operator know?** The error propagates to the calling use case,
   whose span records it. A rejection is recorded by the use case at whose
   boundary it leaves srd5e, if any.
3. **What to diagnose it?** The `Reason` of the `*InvalidError`, reachable with
   `errors.As`.
4. **Counted or timed?** Nothing; constant-time arithmetic.
5. **Existing instrumentation still correct?** Unaffected.
6. **What should a consumer monitor?** Nothing owned here. The expected caller of
   `NewScore` is the content decoder for monster scores (`content-monsters.md`
   FR-018; `content-schema-and-decoder.md` FR-094 requires the decoder to use
   domain limit constants, here `ability.MinScore` and `ability.MaxScore`, not
   literals). It maps the `*InvalidError` to its own problem type
   (`srd5e.content.decode_errors` with `srd5e.content.problem_kind`) and does
   not use `error.type`, which a validation error lacks
   (`errors-and-results.md` FR-043).
7. **Noisy?** No signals are added.

No signal is owned by this spec; `doc/observability/signals.md` is unchanged.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none | | | | |

## 8. Test plan

No ports, so no doubles or contract suites. Plain unit tests in `ability/`
(external package `ability_test`; `ability_test.go` today, split per file as it
grows). Score-to-modifier cases are table-driven from section 4. Tests use
`t.Parallel` and assert `Reason` and `errors.Is`, never message text. Fuzz
tests `FuzzParseAbility` and `FuzzNewScore` cover the two parsers; `Example`
functions (`ExampleNewScore`, `ExampleScores_With`) are the testable docs.

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-033 | unit | The six values, their order, `All()` returns a fresh slice, `Valid` false for 0 and 7. |
| FR-002 to FR-004 | unit | Loop 1 to 30, `Value()` round-trips; 1 and 30 accepted. |
| FR-005 to FR-007 | unit | 0, 31, -1 and `math.MinInt` return an error and the zero `Score`; `errors.Is(err, srderr.ErrInvalidInput)`. |
| FR-008 | unit | Valid `Score` and nil error for 1, 15 and 30. |
| FR-009, FR-010 | unit | `ReasonBelowMinimum` for 0, -5, `math.MinInt`; `ReasonAboveMaximum` for 31, 1000, `math.MaxInt`; no panic. |
| FR-011 | unit | Equal and unequal pairs with `==`. |
| FR-016 | unit | Equal and unequal `Modifier` pairs with `==`. |
| FR-023 | unit | Equal sets and sets differing in one ability. |
| FR-012 | unit | Table-driven, all 30 scores, expected modifiers written as literals from the section 4 table, not recomputed. `Requirement:` and `Path:` comments are on this test; the rest of the suite carries them once `testing.md` section 5 ("planned") is rolled out. |
| FR-013 | unit | 1 gives -5, 3 gives -4, 9 gives -1. Mutation target: truncation would give -4, -3, 0. |
| FR-014 | unit | 10 gives +0 (neighbours 9 gives -1, 11 gives +0, 12 gives +1). |
| FR-015 | unit | 1 gives -5; modifiers across all 30 scores stay in -5 to +10 (30 gives +10). Out-of-range modifiers cannot be built from outside the package, so this enumeration is the test. |
| FR-017 | unit | `Value()` is the signed integer, e.g. -5, 0, +10; `String()` shows the sign. |
| FR-018, FR-020 | unit | A full map builds; for all six abilities `Score` returns the input. |
| FR-019 | unit | Empty map, nil map and each single missing ability return an error; several missing abilities are all reported (`errors.Join`). |
| FR-021 | unit | Table-driven over six abilities: changed score for that ability, other five unchanged, original unchanged; invalid ability and zero score return an error. |
| FR-022 | unit | Mutate the source map after `NewScores`; `Scores` unchanged. |
| FR-024 | unit | For all six abilities `Modifier(a)` equals `Score(a).Modifier()`, using scores that give different modifiers per ability. |
| FR-025 | unit | `MinScore` is 1, `MaxScore` is 30. |
| FR-028, FR-029 | unit | One table of 0, 1, 30, 31, `math.MinInt`, `math.MaxInt`: `NewScore` never panics; the error `errors.As`es `*InvalidError` with the expected `Reason` and `Is` `ErrInvalidInput`. |
| FR-030 | unit | `MustScore` panics iff `NewScore` errors; the recovered value is that error (`errors.As`). |
| FR-026 | architecture | `Reason` and `InvalidError` are declared once in the module; no generic exported `BelowMinimum` / `AboveMaximum` in a context package (check A7). |
| FR-027 | architecture | Import check A1: `ability` and `modifier` do not import each other. A unit test (scenario) shows `modifier.New(ability.MustScore(20).Modifier().Value())` equals `modifier.New(5)`; the `RollModifiers` slot is tested in `d20-tests.md` section 4.3. |
| FR-031 | review | No closed result type exists in `ability`; checked by check A7 finding no marker interface in the package. |
| FR-032 | unit, fuzz | Round-trip `ParseAbility(a.String())`; case-insensitive; unknown and over-long strings rejected; `FuzzParseAbility` never panics. |

- Scenario: none beyond the FR-027 conversion. Architecture: the generic
  allow-list test (check A1) that `ability` imports only `srderr` covers these
  files, plus FR-026 and FR-027.
- Mutation testing (`doc/rules/testing.md` §8): `gomu` on `./ability`, run only
  inside Docker with `--cpus` and `--workers` capped at a quarter of the host's
  processors (command in testing.md). Never run it as a bare host process.
  Likely targets: the range checks (`< MinScore`, `> MaxScore`), the shift and
  the band boundaries; the boundary rows exist to kill them.
- Consumers get an `ability.Modifier` in their own tests with
  `ability.MustScore(n).Modifier()` (20 gives +5, 8 gives -1, 10 gives +0).
- Unexported fields (no way to build an out-of-range `Modifier`) are a
  compile-time property, not tested directly. Every FR id is covered.

## 9. Open questions and decisions

Settled: `ability.Modifier` to `modifier.Modifier` conversion is by callers
(`shared-types-and-ownership.md` FR-003, FR-027 here); the rejection error has a
single definition (`shared-types-and-ownership.md` FR-001, FR-026 here);
`Scores` is a kernel type (decided here, section 5); no 20 cap and no score of 0
(section 1); the floor formula is acceptable, with the table as source of truth
(section 4).

Settled, one line each:

- **Q-1 (decided amendment, roadmap; done).** `doc/roadmap.md` Phase 0 must list `Ability`, `Score`, `Modifier`, `Scores`, and Phase 3 must drop "Ability modifiers" (`shared-types-and-ownership.md` FR-001 and FR-014 already make them kernel types, phase 0). Score generation stays out of scope.
- **Q-2 (decided amendment, shared types; done).** `shared-types-and-ownership.md` FR-001 adds `Scores` to its kernel list, with this spec as owner.
- **Q-3 (decided: no).** SRD abbreviations (p. 176) are not exposed on `Ability`: display text. Revisit if `content` needs them for decoding.
- **Q-4 (decided: no).** `Score` does not offer ordering helpers now (callers compare `Value()`); add when a use case needs it.
- **Q-5 (decided: implemented name wins; done).** The rejection error is `ability.InvalidError` with a `Reason` enum, which also covers unknown abilities and missing scores; `shared-types-and-ownership.md` FR-001 uses the same name. `InvalidError` carries `Reason` and a log-only `Detail`; it does not carry the rejected integer as a field (`errors-and-results.md` FR-027 allows a sentinel or a typed reason).

## 10. Changelog impact

Added: package `ability` with `Ability`, `Score` (range 1 to 30, `MinScore`,
`MaxScore`, validating `NewScore` and `MustScore`), `Modifier` (derivation from
the Ability Modifiers table), `Scores` (`NewScores`, `Score`, `Modifier`,
`With`), `ParseAbility`, and the typed `InvalidError` / `Reason`. Not
**BREAKING**: all new exports in a first release.
