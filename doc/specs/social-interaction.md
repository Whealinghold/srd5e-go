# Social interaction

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `d20-tests.md`, `resolve-d20-test-use-case.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `ability-scores.md`, `content-schema-and-decoder.md`, `observability-ports.md`, `errors-and-results.md` |

Notes. `resolve-d20-test-use-case.md` is listed only for the `d20` signal
attribute key constants (its FR-040; here `d20.AttrRollMode`). Leaf in `character` (Q-10); names no `combat` type
(`shared-types-and-ownership.md` FR-013). Phases: `d20` types 1, `content`
vocabulary 2 (`shared-types-and-ownership.md` FR-014).

## 1. Purpose and scope

Social interaction as the SRD states it as rules: a monster's **attitude**
(Friendly, Indifferent, Hostile), the **Influence action**, and how an Influence
attempt resolves. The GM first decides whether the monster is Willing,
Unwilling or Hesitant. Only a Hesitant monster needs an ability check, resolved
by the `d20` domain functions, with the attitude contributing Advantage or
Disadvantage, against a default DC of 15 or the monster's Intelligence score,
whichever is higher. A failed check starts a wait before the same urging may be
tried again.

The slice is a small pure domain plus one use case, `ResolveInfluence`, which
rolls the d20s through a one-method `Roller` port (satisfied by `dice.Roller`) and hands them to the pure
`d20.ResolveCheck` and `d20.FailWithoutRoll` (`shared-types-and-ownership.md` FR-044). It adds no dice
logic or outcome rule and never calls `d20.Resolver.Resolve`.

**Ownership** (`shared-types-and-ownership.md` FR-048). This spec owns
`Attitude` (`rules-glossary-coverage.md` GL-013, GL-069, GL-082, GL-088) and the
resolution of the Influence action's effect. **Supersedes:** nothing; the
`initiative-and-actions.md` and `combat-action-resolution.md` entries only list
and spend the action, and `conditions.md` FR-054 owns the Charmed Advantage.

**Out of scope:**

- **Roleplaying and deciding Willing, Unwilling or Hesitant.** "The GM uses an
  NPC's personality ... to determine how an NPC reacts" (SRD 5.2.1,
  "Roleplaying", pp. 10-11) and "The GM then determines whether the monster
  feels willing, unwilling, or hesitant" (p. 184) are adjudication. The
  determination is an *input* (FR-013 to FR-015); `Alignment`
  (`content-schema-and-decoder.md` FR-137) is not used to infer it.
- **The GM choosing the check** (p. 184). The Influence Checks table only
  *suggests* (FR-008, FR-009).
- **Changing a monster's attitude.** The Actions table (p. 10) and "Ability
  Checks" (p. 11) mention altering attitude, but the Influence entry says only
  that on a success "the monster does as urged" (p. 184). The SRD gives no rule
  for moving between attitudes, so none is modelled (Q-04).
- **Tracking time and "the same way"**: host state (FR-019, FR-020; Q-05).
- **Spending the action**, a `combat` concern; the host composes the two
  (`combat-action-resolution.md` Q-23). The Help action (p. 182) is owned by
  `attack-and-help-actions.md` (FR-043); its Advantage arrives as a circumstance.
- **Item and spell effects on social checks**, caller-supplied circumstances or
  modifiers: Charmed ("Charmed", p. 178), Perfume (Advantage on Charisma
  (Persuasion) checks to influence an Indifferent Humanoid within 5 feet, p. 99),
  Speak with Animals (p. 164).
- **Skill modifiers**, built by `derived-stats.md`; the caller passes `Modifier`s.
- **Influence on a player character.** The SRD defines attitude for "a monster
  toward a player character" (p. 177) and the action for a monster only.
- **Anything not in the SRD.** No reaction table, attitude-shift table or
  social-encounter structure exists in SRD 5.2.1.

## 2. Source

- SRD 5.2.1, "Actions" table (Influence row), p. 10; "Social Interaction",
  "Roleplaying", "Ability Checks", pp. 10-11.
- Rules Glossary: "Attitude", p. 177; "Friendly [Attitude]", p. 182; "Hostile
  [Attitude]", p. 183; "Indifferent [Attitude]", "Influence [Action]" and the
  "Influence Checks" table, p. 184; "Monster" and "Nonplayer Character", p. 185.
- "Skills" table, p. 9; "D20 Tests", pp. 6-8; "Communicating", p. 13;
  "Charmed", p. 178; "Perfume", p. 99; "Speak with Animals", p. 164.
- Package rules: `doc/rules/architecture.md` §2, §3, §4, §6;
  `doc/rules/observability.md` §2 to §7; `doc/rules/testing.md` §4.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Attitude | `Attitude` (typed enum plus the constant `MonsterDefaultAttitude`, in package `character`) | "A monster has a starting attitude toward a player character: Friendly, Hostile, or Indifferent" (p. 177). Static fact only; behaviour is `AttitudeCircumstances` (Q-01). |
| Influence [Action] | use case `InfluenceResolver.ResolveInfluence` (`NewInfluenceResolver`) | Resolves the effect only; the action is a `combat` concern. |
| Willing, Unwilling, Hesitant | variants of closed `InfluenceCommand` | The GM's determination (p. 184); input, never inferred. |
| Influence Checks table row | `InfluenceApproach` (typed enum: `Deceive`, `Intimidate`, `Amuse`, `Persuade`, `Coax`) | Deceiving or Persuading a monster that understands you; Intimidating; Amusing; Gently coaxing a Beast or Monstrosity (p. 184). |
| "a monster that understands you" | `InfluenceComprehension` (typed enum: `Understands`, `DoesNotUnderstand`) | Named state, not a boolean (`architecture.md` §3). |
| the monster | `InfluenceTarget` (immutable value object built by `NewInfluenceTarget`) | Attitude, Intelligence score, creature type, comprehension. A monster is "a creature controlled by the GM" and an NPC "is a monster that has a personal name and a distinct personality" (p. 185), so one type serves both (Q-06). |
| caller's modifiers and circumstances | closed `InfluenceAttempt` (an interface with an unexported marker method; variants `InfluenceRoll`, `InfluenceAutomaticFailure`) | Package term (FR-017, FR-028, FR-041). |

The model uses `monster` terms, as the rules do (Q-06). Friendly gives
Advantage (p. 182), Hostile Disadvantage (p. 183), Indifferent is "the default
attitude of a monster" (p. 184).

## 4. Requirements

Ids are stable and never renumbered. No requirement of this spec is withdrawn.

### 4.1 Attitude

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `Attitude` (owned by this spec, in package `character`) MUST be a typed enum of exactly `Friendly`, `Indifferent` and `Hostile` (3 values, with `String()`), in that order, and expose the static fact as the constant `MonsterDefaultAttitude`. The zero `Attitude` is not one of the three and is invalid (`Valid()` reports false). It MUST carry no behaviour. | SRD 5.2.1, "Attitude", p. 177; "Social Interaction", p. 10; `shared-types-and-ownership.md` FR-006 | happy |
| FR-002 | `NewInfluenceTarget` MUST take the attitude as the optional `WithAttitude` option and default to `MonsterDefaultAttitude`, which MUST equal `Indifferent`. A target built without an attitude MUST yield no attitude circumstance (no Advantage or Disadvantage), the same as an explicit Indifferent one. | SRD 5.2.1, "Indifferent [Attitude]", p. 184 ("the default attitude of a monster") | happy; unhappy (empty and minimal) |
| FR-003 | `AttitudeCircumstances(Friendly)` MUST return exactly one `modifier.Advantage`. | SRD 5.2.1, "Friendly [Attitude]", p. 182 | happy |
| FR-004 | `AttitudeCircumstances(Hostile)` MUST return exactly one `modifier.Disadvantage`. | SRD 5.2.1, "Hostile [Attitude]", p. 183 | happy |
| FR-005 | `AttitudeCircumstances(Indifferent)` MUST return an empty slice: an Indifferent attitude contributes neither Advantage nor Disadvantage. | SRD 5.2.1, "Indifferent [Attitude]", p. 184 (no effect on the check is stated) | unhappy (empty and minimal) |

### 4.2 Approach and check suggestion

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-006 | `InfluenceApproach` MUST enumerate exactly the five rows of the Influence Checks table (`Deceive`, `Intimidate`, `Amuse`, `Persuade`, `Coax`) and map them to the skills Deception, Intimidation, Performance, Persuasion and Animal Handling respectively. | SRD 5.2.1, "Influence Checks" table, p. 184; "Actions", Influence row, p. 10 | happy |
| FR-007 | The ability of an approach's check MUST be read from `content.Skill.Ability()` (published by `content`, `content-schema-and-decoder.md` FR-106): Charisma for `Deceive`, `Intimidate`, `Amuse` and `Persuade`; Wisdom for `Coax`. This spec MUST NOT declare a second skill-to-ability mapping. | SRD 5.2.1, "Skills" table, p. 9; "Influence Checks" table, p. 184 | happy |
| FR-008 | `IsSuggested(approach, target)` MUST be true exactly for the table's rows: `Intimidate` and `Amuse` for any target; `Deceive` and `Persuade` when the target `Understands`; `Coax` when the target's creature type is Beast or Monstrosity. Cases in section 4.7. | SRD 5.2.1, "Influence Checks" table, p. 184 | happy; unhappy (boundary) |
| FR-009 | The use case MUST NOT reject an approach that is not suggested for the target (for example `Coax` on a Humanoid, or `Persuade` on a target that `DoesNotUnderstand`); it MUST resolve the check as for any other approach. | SRD 5.2.1, "Influence [Action]", p. 184 ("The Influence Checks table suggests ... The GM chooses the check") | unhappy (rule interaction) |

### 4.3 Difficulty Class

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | `DefaultDC(intelligence ability.Score)` MUST return `d20.DCMedium` (15) when the Intelligence score is 15 or lower, and `d20.DifficultyClass(score)` when it is higher. Cases in section 4.7, including 1, 14, 15, 16 and 30. | SRD 5.2.1, "Influence [Action]", p. 184 ("a default DC equal to 15 or the monster's Intelligence score, whichever is higher") | happy; unhappy (boundary) |
| FR-011 | When the command carries a GM-set `DifficultyClass`, that `DifficultyClass` MUST be the check's target instead of the default; the default is only a default. | SRD 5.2.1, "Influence [Action]", p. 184 ("has a default DC") | happy |
| FR-012 | The result of a check MUST record how its DC was chosen as `DcBasis` (`DefaultRule` or `GmSet`), together with the `DifficultyClass` used, so a consumer can explain it. | `doc/rules/architecture.md` §6 (results carry what explains them); SRD 5.2.1, p. 184 | happy |

### 4.4 Resolution

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-013 | An `InfluenceWillingCommand` MUST return `ResolveInfluenceFulfilled` without any request to the `Roller` and without calling `d20`. | SRD 5.2.1, "Influence [Action]", "Willing", p. 184 ("no ability check is necessary; the monster fulfills your request") | happy |
| FR-014 | An `InfluenceUnwillingCommand` MUST return `ResolveInfluenceRefused` without any request to the `Roller` and without calling `d20`. | SRD 5.2.1, "Influence [Action]", "Unwilling", p. 184 ("no ability check is necessary; it doesn't comply") | happy |
| FR-015 | An `InfluenceHesitantCommand` whose attempt is an `InfluenceRoll` MUST make exactly one ability check: it MUST make exactly one request to the roller through `dice.Roll` for the `dice.Expression` of `mode.DiceCount()` d20s with modifier 0 (a single `1d20` for `modifier.Normal`, a single `2d20` for `modifier.WithAdvantage` or `modifier.WithDisadvantage`), build the `d20.Roll` with `d20.NewRoll(mode, faces)` from the mode and the faces in order, and obtain the result from `d20.ResolveCheck(d20.AbilityCheck, roll, modifiers, dc)`, with the `DifficultyClass` of FR-010 or FR-011 and the `d20.RollModifiers` of FR-017. It MUST NOT compute any total, outcome or kept die itself. | SRD 5.2.1, "Influence [Action]", "Hesitant", p. 184 ("you must make an ability check"); "D20 Tests", p. 6; `d20-tests.md` FR-010, FR-069; `shared-types-and-ownership.md` FR-044 | happy |
| FR-016 | For a rolled attempt, the roll mode MUST be `modifier.Resolve` called once over the attitude's circumstances (FR-003 to FR-005) followed by the `InfluenceRoll`'s circumstances, unchanged and in order. This spec implements no stacking or cancelling. Cases in section 4.7. | SRD 5.2.1, "Friendly", p. 182; "Hostile", p. 183; "Advantage/Disadvantage", p. 7; "They Don't Stack", p. 8; `modifiers-and-advantage.md` FR-010 to FR-017 | unhappy (rule interaction) |
| FR-017 | The `RollModifiers` MUST be built from the `InfluenceRoll` unchanged: the ability modifier chosen by FR-041 for the approach's ability, the optional Proficiency Bonus and the other `Modifier`s in order. With only the ability modifier (no Proficiency Bonus, no other modifiers) the total MUST equal the natural d20 used plus that ability modifier; there is no bare-modifier-list form (`shared-types-and-ownership.md` FR-003). | SRD 5.2.1, "D20 Tests", "Add Modifiers", p. 6; `d20-tests.md` FR-016, FR-017 | unhappy (empty and minimal) |
| FR-018 | A check whose outcome is `d20.Success` MUST return `ResolveInfluenceCheckSucceeded`, carrying the `d20.CheckResult`; a total equal to the DC succeeds, and the result carries no retry wait. | SRD 5.2.1, "Influence [Action]", p. 184 ("On a successful check, the monster does as urged"); "Compare the Total to a Target Number", p. 6 | happy; unhappy (boundary) at equal |
| FR-019 | A check whose outcome is `failure` MUST return `ResolveInfluenceCheckFailed`, carrying the `d20.CheckResult` and a `RetryAfter` of 24 hours (`24 * time.Hour`) when no GM duration is given; a total one below the DC fails. | SRD 5.2.1, "Influence [Action]", p. 184 ("you must wait 24 hours (or a duration set by the GM) before urging it in the same way again") | happy; unhappy (boundary) at one below |
| FR-020 | When the command carries a GM-set wait, `RetryAfter` on a failed check MUST equal it. A wait of `0` is accepted. | SRD 5.2.1, "Influence [Action]", p. 184 ("or a duration set by the GM") | unhappy (boundary) |
| FR-021 | `NewInfluenceHesitantCommand` called with a negative GM-set wait (`WithGMRetryAfter`) MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` and the zero command. `MustInfluenceHesitantCommand` panics with that error. | Package design (not an SRD rule): `doc/rules/architecture.md` §6; `doc/rules/testing.md` §4 (invalid input) | unhappy (invalid input) |
| FR-022 | A command whose attempt is `InfluenceAutomaticFailure` MUST obtain its result from `d20.FailWithoutRoll(d20.AbilityCheck, d20.Automatic)`, MUST make no request to the `Roller`, MUST return `ResolveInfluenceCheckForcedFailure` carrying the `d20.ForcedFailureResult` and the same `RetryAfter` as FR-019, and the attitude's circumstances MUST NOT be applied. The caller decides when a check cannot succeed (for example one that requires sight or hearing while Blinded or Deafened). | SRD 5.2.1, "Blinded", p. 177; "Deafened", p. 181; `d20-tests.md` FR-036 | unhappy (rule interaction) |
| FR-023 | Natural 20 and natural 1 MUST NOT change the outcome: the check resolves by total against the DC alone (a natural 20 whose total is below the DC fails; a natural 1 whose total reaches the DC succeeds). | SRD 5.2.1, "Rolling 20 or 1", p. 7 ("for an attack roll"); `d20-tests.md` FR-031 | unhappy (boundary) |
| FR-024 | `ResolveInfluenceFulfilled` and `ResolveInfluenceRefused` MUST carry no fields; they are distinguished by type alone. No result variant has an attitude-change field (structural; Q-04). | `doc/rules/architecture.md` §6; SRD 5.2.1, p. 184 (no attitude change is stated) | happy |

### 4.5 Failures, determinism and immutability

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-025 | When the roller fails (it returns an error, panics, or answers a different expression), `dice.Roll` yields a `*srderr.PortError` (`Port` is `dice.Port`; it matches the cause, `srderr.ErrPortPanic` or `srderr.ErrContractViolation` with `errors.Is`) and `ResolveInfluence` MUST return that same error value, MUST NOT request dice again and MUST return the zero result. It never converts the failure into a result variant (`errors-and-results.md` FR-031). | `doc/rules/architecture.md` §4 (port failure contract); `dice-and-rolls.md` FR-051, FR-052 | unhappy (failure of collaborator) |
| FR-026 | When the roller is a `dicetest.FixedRoller` that has run out of values, it returns an error, and FR-025 applies: the use case does not add recovery of its own beyond `dice.Roll`. | `errors-and-results.md` FR-009, FR-031; `dice-and-rolls.md` FR-053 | unhappy (failure of collaborator) |
| FR-027 | For equal commands and equal d20 values, `ResolveInfluence` MUST return equal results, including at the extremes (all 1s, all 20s) for each of the three attitudes. | `doc/rules/architecture.md` §3 (purity); `doc/rules/testing.md` §4 (determinism) | unhappy (determinism) |
| FR-028 | `InfluenceRoll` MUST hold its other `modifier.Modifier`s, its circumstances and its ability-modifier map as copies (mutating the caller's slice or map afterwards MUST NOT change the value; its accessors return fresh copies, so mutating a returned slice or map changes nothing), and `ResolveInfluence` MUST NOT alter its command. | `doc/rules/testing.md` §4 (immutability); `doc/rules/architecture.md` §4 (handoffs) | unhappy (immutability) |
| FR-029 | `InfluenceTarget`, the three commands and every result variant MUST be equal when every field is equal and unequal when any single field differs (including collection contents and order): `==` for the comparable ones (`InfluenceTarget`, the result variants without slices) and an `Equal` method for the types holding slices or maps. | `doc/rules/architecture.md` §3 (value objects) | happy; unhappy (any differing field) |
| FR-030 | `NewInfluenceTarget` MUST take its Intelligence score as an `ability.Score`; this spec adds no range check of its own and relies on the kernel's (`ability-scores.md` FR-002), except that the zero `ability.Score` (`IsZero`) MUST be rejected with an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. | `ability-scores.md` FR-002 | unhappy (boundary) |

### 4.6 Telemetry and structure

This use case is the outermost one for every call: it records its own signals,
including its own error count, from the domain results
(`shared-types-and-ownership.md` FR-044; `doc/rules/observability.md` §7).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | Each call MUST run inside one span `srd5e.character.resolve_influence`, opened through the `telemetry.Tracer` port (`Tracer.Start(ctx, ...)` with `defer span.End()`), with no child spans. For a hesitant command it MUST set `srd5e.influence.approach` and `srd5e.influence.attitude` at start, and `srd5e.influence.response`, `srd5e.outcome` and (FR-043) `srd5e.d20_test.roll_mode` when a d20 was rolled (a forced failure has none). A failed check or a refusal MUST NOT set error status (`telemetry.Status` stays unset). | `doc/rules/observability.md` §6 | happy; unhappy (normal rule outcome is not an error) |
| FR-032 | Each completed call MUST increment `srd5e.character.influence_resolved` once, with `srd5e.influence.response` and `srd5e.outcome` always, and `srd5e.influence.approach`, `srd5e.influence.attitude` for hesitant commands only, and `srd5e.d20_test.roll_mode` only for rolled hesitant attempts (FR-043). | `doc/rules/observability.md` §5 | happy |
| FR-033 | Each call, completing or returning an error, MUST record its duration in seconds in `srd5e.character.influence_duration` exactly once through `telemetry.StartTimer(clock, histogram)` with a deferred `Timer.Stop(ctx)`, with no attribute, and the timer MUST stop before `span.End()` runs. The use case MUST NOT read the wall clock: the clock is injected with the `WithClock` option, and without one no duration is recorded (`telemetry.Timer`). | `doc/rules/observability.md` §4, §5 | happy; unhappy (failure of collaborator) |
| FR-034 | When a `*srderr.PortError` leaves the use case (from the roller), it MUST be recorded exactly once, here, because this use case is the outermost one: span error status with `span.RecordError(err)`, and `srd5e.character.influence_errors` incremented once with `error.type` equal to `srderr.ErrorType(err)` (`telemetry.AttrErrorType`; never `%T` or `err.Error()`). It MUST NOT be logged, `influence_resolved` MUST NOT increment, and `influence_duration` is still recorded (FR-033). A cancelled or expired `ctx` is neither success nor failure of the rule: it MUST NOT be turned into a result or counted as a failure outcome (`errors-and-results.md` FR-054). A panic in a telemetry backend is the host's bug and propagates (`errors-and-results.md` FR-051). A returned failure result carries no `error.type`, and none of the four results is an error. | `doc/rules/observability.md` §7 ("Errors: who records what"); `shared-types-and-ownership.md` FR-044, FR-051; `errors-and-results.md` FR-005, FR-031, FR-034, FR-043, FR-051, FR-054 | unhappy (failure of collaborator, telemetry on failure) |
| FR-035 | Every attribute value MUST come from a closed set through a fixed mapping: `srd5e.influence.response` (`fulfilled`, `refused`, `check_succeeded`, `check_failed`); `srd5e.outcome` (`success` for `ResolveInfluenceFulfilled` and `ResolveInfluenceCheckSucceeded`, `failure` for `ResolveInfluenceRefused`, `ResolveInfluenceCheckFailed` and `ResolveInfluenceCheckForcedFailure`); `srd5e.influence.approach` (`Deceive`, `Intimidate`, `Amuse`, `Persuade`, `Coax`); `srd5e.influence.attitude` (`friendly`, `indifferent`, `hostile`); `srd5e.d20_test.roll_mode` (`normal`, `advantage`, `disadvantage`); `error.type` (FR-034). Both failed-check variants map to response `check_failed`. Attributes MUST NOT carry ids, scores, DCs, totals or free text. | `doc/rules/observability.md` §5 (attributes); `shared-types-and-ownership.md` FR-041 | unhappy (boundary) |
| FR-036 | Results, the dice requested and the error type returned for a failing roller MUST be identical with `telemetry.Noop` and with a recording one (`telemetrytest.Recorder`). | `doc/rules/observability.md` §9 | unhappy (determinism) |
| FR-037 | Signal and attribute-key names MUST be defined once as exported constants in the package's `signals.go` (the `influence` group; shared keys `srd5e.outcome` and `error.type` are `telemetry.AttrOutcome` and `telemetry.AttrErrorType`; the key `srd5e.d20_test.roll_mode` is `d20.AttrRollMode`, which `resolve-d20-test-use-case.md` FR-040 and FR-042 export, not redefined), MUST NOT appear as literals at call sites, and MUST be pinned by tests that assert their literal values. | `doc/rules/observability.md` §5, §9, §10 | unhappy (boundary: contract pinning) |
| FR-038 | The domain files of this spec in package `character` MUST import only kernel packages (`ability`, `modifier`), the published API of `content` (`Skill`, `CreatureType`) and of `d20` (`DifficultyClass`). The `resolve_influence_usecase.go` file MUST additionally import `dice`, `srderr` and `telemetry` and, from `d20`, only `NewRoll`, `Roll`, `RollModifiers`, `Kind`, `Outcome`, `ResolveCheck`, `FailWithoutRoll`, `ForcedFailureReason`, `CheckResult`, `ForcedFailureResult` and `DifficultyClass` (the command and the `ResolveInfluenceCheck*` results carry it). No file MUST import `combat`, an adapter, a `*test` package (outside `_test.go` files) or use `d20.Resolver`. | `doc/rules/architecture.md` §1, §2 (dependency rule, context map); `shared-types-and-ownership.md` FR-013, FR-044 | unhappy (forbidden edge) |
| FR-039 | Package `character` MUST export, each once and each with a doc comment starting with its name, `InfluenceResolver` (with `NewInfluenceResolver` and its `ResolveInfluence` method), `InfluenceCommand` with `InfluenceWillingCommand`, `InfluenceUnwillingCommand` and `InfluenceHesitantCommand`, `InfluenceAttempt` with `InfluenceRoll` and `InfluenceAutomaticFailure`, `ResolveInfluenceResult` with `ResolveInfluenceFulfilled`, `ResolveInfluenceRefused`, `ResolveInfluenceCheckSucceeded`, `ResolveInfluenceCheckFailed` and `ResolveInfluenceCheckForcedFailure`, `Attitude`, `InfluenceApproach`, `InfluenceTarget`, `InfluenceComprehension`, `DcBasis`, the functions `AttitudeCircumstances`, `IsSuggested` and `DefaultDC`, and the `influence` signal constants. There is no re-export package, and `character` MUST NOT re-export `d20`, kernel or `content` types (aliases included). | `doc/rules/architecture.md` §2, §4, §7 | happy |

### 4.7 Cases

Default DC (FR-010). Source for every row: SRD 5.2.1, "Influence [Action]", p. 184.

| Label | Intelligence score | Default `DifficultyClass` | Path |
|---|---|---|---|
| lowest score | 1 | 15 | unhappy (boundary) |
| below the floor | 14 | 15 | unhappy (boundary) |
| at the floor | 15 | 15 | unhappy (boundary) |
| one above the floor | 16 | 16 | unhappy (boundary) |
| typical high | 20 | 20 | happy |
| highest score | 30 | 30 | unhappy (boundary) |

Approach suggested for the target (FR-008). Source: p. 184, "Influence Checks".

| Label | Approach | Comprehension | Creature type | `IsSuggested` | Path |
|---|---|---|---|---|---|
| intimidate, any target | `Intimidate` | `DoesNotUnderstand` | Aberration | true | unhappy (boundary) |
| amuse, any target | `Amuse` | `DoesNotUnderstand` | Beast | true | happy |
| deceive, understands | `Deceive` | `Understands` | Humanoid | true | happy |
| deceive, does not understand | `Deceive` | `DoesNotUnderstand` | Humanoid | false | unhappy (boundary) |
| persuade, understands | `Persuade` | `Understands` | Giant | true | happy |
| persuade, does not understand | `Persuade` | `DoesNotUnderstand` | Giant | false | unhappy (boundary) |
| coax a Beast | `Coax` | `DoesNotUnderstand` | Beast | true | happy |
| coax a Monstrosity | `Coax` | `Understands` | Monstrosity | true | happy |
| coax a Humanoid | `Coax` | `Understands` | Humanoid | false | unhappy (boundary) |
| coax an Ooze | `Coax` | `DoesNotUnderstand` | Ooze | false | unhappy (boundary) |

Roll mode of a hesitant check (FR-016), from the attitude's circumstance and the caller's. The mode is the existing `modifier.Resolve` over both lists.

| Label | Attitude | Caller's circumstances | Expected `RollMode` | Dice requested | Source | Path |
|---|---|---|---|---|---|---|
| Friendly alone | friendly | none | advantage | 2d20 | p. 182 | happy |
| Indifferent alone | indifferent | none | normal | 1d20 | p. 184 | unhappy (empty and minimal) |
| Hostile alone | hostile | none | disadvantage | 2d20 | p. 183 | happy |
| Friendly, caller adds Advantage | friendly | Advantage | advantage (no stacking) | 2d20 | p. 8 | unhappy (rule interaction) |
| Friendly, caller adds Disadvantage | friendly | Disadvantage | normal (cancel) | 1d20 | p. 8 | unhappy (rule interaction) |
| Hostile, caller adds Advantage (the charmer, Charmed, p. 178) | hostile | Advantage | normal (cancel) | 1d20 | pp. 8, 178, 183 | unhappy (rule interaction) |
| Indifferent, caller adds Advantage (Perfume, p. 99) | indifferent | Advantage | advantage | 2d20 | pp. 7-8 | happy |
| Hostile, caller adds two Advantages | hostile | Advantage, Advantage | normal (cancel) | 1d20 | p. 8 | unhappy (rule interaction) |

Resolution (FR-018, FR-019, FR-020, FR-023). Fixed rolls through `dicetest.NewFixedRoller`; Intelligence 10 (default DC 15), no GM-set `DifficultyClass`.

| Label | Attitude | d20 served | Modifier | Used d20 | Total | Result | `RetryAfter` | Source | Path |
|---|---|---|---|---|---|---|---|---|---|
| total equals DC | indifferent | 12 | +3 | 12 | 15 | `ResolveInfluenceCheckSucceeded` | none | pp. 6, 184 | unhappy (boundary) |
| total one below | indifferent | 11 | +3 | 11 | 14 | `ResolveInfluenceCheckFailed` | 24 hours | pp. 6, 184 | unhappy (boundary) |
| natural 20, total below DC | indifferent | 20 | -10 | 20 | 10 | `ResolveInfluenceCheckFailed` | 24 hours | pp. 7, 184 | unhappy (boundary) |
| natural 1, total reaches DC | indifferent | 1 | +17 | 1 | 18 | `ResolveInfluenceCheckSucceeded` | none | pp. 7, 184 | unhappy (boundary) |
| Friendly, higher die kept | friendly | 4, 16 | +3 | 16 | 19 | `ResolveInfluenceCheckSucceeded` | none | pp. 8, 182 | happy |
| Hostile, lower die kept | hostile | 16, 4 | +3 | 4 | 7 | `ResolveInfluenceCheckFailed` | 24 hours | pp. 8, 183 | happy |
| GM-set wait of 1 hour | indifferent | 2 | +3 | 2 | 5 | `ResolveInfluenceCheckFailed` | 1 hour | p. 184 | happy |
| GM-set wait of zero | indifferent | 2 | +3 | 2 | 5 | `ResolveInfluenceCheckFailed` | `0` | p. 184 | unhappy (boundary) |
| automatic failure | any | none served | n/a | none | n/a | `ResolveInfluenceCheckForcedFailure` | 24 hours | pp. 177, 184 | unhappy (rule interaction) |


"Modifier" is the total: the ability modifier is +3 (caller-supplied); where
the total differs, a second `Modifier` carries the difference (-13 and +14 in
the natural-roll rows), because `RollModifiers` always holds an ability modifier
(FR-017).

### 4.8 Added requirements

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | `ResolveInfluenceCheckSucceeded`, `ResolveInfluenceCheckFailed` and `ResolveInfluenceCheckForcedFailure` MUST each carry the `InfluenceApproach`, the `Attitude` the urging was made against, the `DifficultyClass` and the `DcBasis`. The first two carry the `d20.CheckResult` of the rolled check as `Check()`; the third carries the `d20.ForcedFailureResult` as `Check()` (FR-022). The two failure variants additionally carry `RetryAfter`. | `doc/rules/architecture.md` §3 (illegal states unrepresentable) and §6 (results carry what explains them); SRD 5.2.1, p. 184 | happy |
| FR-041 | The ability modifier in the check's `RollModifiers` MUST be the one for the approach's ability (FR-007). `InfluenceRoll` carries its ability modifiers as a map from `ability.Ability` to `modifier.Modifier`; the use case reads the entry for `approach.Skill().Ability()`. `NewInfluenceHesitantCommand` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when its `InfluenceRoll` has no entry for the approach's ability. Entries for other abilities are ignored. Trade-off: Q-13. | SRD 5.2.1, "Influence Checks" table, p. 184; "Skills" table, p. 9 | unhappy (invalid input; rule interaction) |
| FR-042 | `InfluenceApproach` MUST NOT hold an ability field or switch of its own: its `Ability()` method MUST delegate to `content.Skill.Ability()`. | SRD 5.2.1, "Skills" table, p. 9; `shared-types-and-ownership.md` FR-006 | happy |
| FR-043 | For a rolled attempt, `srd5e.d20_test.roll_mode` MUST equal `Roll.Mode()` of the roll inside the `d20.CheckResult`; for `ResolveInfluenceCheckForcedFailure` (no d20 rolled) and for fulfilled and refused calls it MUST be absent, as in `resolve-d20-test-use-case.md` FR-029 and `attacks.md` FR-118. | `shared-types-and-ownership.md` FR-041 | happy; unhappy (boundary: attribute presence) |
| FR-044 | `ResolveInfluence` MUST NOT call, construct, hold or mention another use case (in particular `d20.Resolver`). It calls the pure `d20` domain functions directly; they hold no state, so nothing needs injecting. | `shared-types-and-ownership.md` FR-044; `architecture-guardrails.md` FR-097 | unhappy (forbidden edge) |
| FR-045 | The use case MUST emit no log event at any level, on success or on failure. | `doc/rules/observability.md` §7 ("Do not log what the result already says"); `shared-types-and-ownership.md` FR-043 (same convention) | happy |


## 5. Model

Layout (`doc/rules/architecture.md` §2): one package, `character`, no
sub-folders. Domain files: `attitude.go`, `influence_approach.go`,
`influence_target.go`, `dc_basis.go`, `influence_rules.go`. Use-case files:
`resolve_influence_usecase.go` with the commands, `InfluenceAttempt` and
variants, the results, and `signals.go`. Types and rules are as section 3 and
section 4 state; the shapes not given there:

- `NewInfluenceTarget(intelligence ability.Score, creatureType content.CreatureType,
  comprehension InfluenceComprehension, opts ...TargetOption) (InfluenceTarget, error)`
  with the option `WithAttitude(Attitude)` (default `MonsterDefaultAttitude`,
  FR-002): a plain comparable value, no monster entity (Q-06). The error
  satisfies `errors.Is(err, srderr.ErrInvalidInput)` for a zero score, an
  invalid `Attitude` or comprehension (FR-030). The zero `InfluenceTarget` is
  invalid.
- Package-level pure functions (stateless): `AttitudeCircumstances(Attitude)
  []modifier.RollCircumstance`; `IsSuggested(InfluenceApproach, InfluenceTarget)
  bool`; `DefaultDC(ability.Score) d20.DifficultyClass`.
- Closed `InfluenceCommand` (interface with an unexported marker method; the
  doc comment lists all three variants): `InfluenceWillingCommand{}`,
  `InfluenceUnwillingCommand{}` (no fields) and `InfluenceHesitantCommand`, built
  by `NewInfluenceHesitantCommand(approach, target, attempt InfluenceAttempt,
  opts ...HesitantOption) (InfluenceHesitantCommand, error)` with the options
  `WithGMDC(d20.DifficultyClass)` and `WithGMRetryAfter(time.Duration)`
  (absent by default, so the optional GM values need no pointers). The
  constructor validates FR-021 and FR-041. The GM's determination chooses the
  variant, so a check cannot be requested for a Willing or Unwilling monster.
- Closed `InfluenceAttempt`: `InfluenceRoll` (built by `NewInfluenceRoll(abilityModifiers
  map[ability.Ability]modifier.Modifier, opts ...RollOption)` with the options
  `WithProficiencyBonus(modifier.Modifier)`, `WithOtherModifiers(...modifier.Modifier)`
  and `WithCircumstances(...modifier.RollCircumstance)`; every collection copied) and
  `InfluenceAutomaticFailure{}`.
- Closed `ResolveInfluenceResult` (variant prefix per
  `shared-types-and-ownership.md` FR-017): the five variants of FR-039, fields
  per FR-024 and FR-040. No domain events: nothing persists.
- `InfluenceApproach` holds its `content.Skill` (FR-006) and exposes it as `Skill()`;
  its `Ability()` method delegates to `Skill().Ability()` (FR-042).
- Result variants, with read-only accessors for each field:
  `ResolveInfluenceCheckSucceeded{Approach, Attitude, DC, DCBasis, Check d20.CheckResult}`;
  `ResolveInfluenceCheckFailed{Approach, Attitude, DC, DCBasis, Check d20.CheckResult, RetryAfter time.Duration}`;
  `ResolveInfluenceCheckForcedFailure{Approach, Attitude, DC, DCBasis, Check d20.ForcedFailureResult, RetryAfter time.Duration}`.
- Failures: results are never errors. A roller failure is returned as a
  `*srderr.PortError` (FR-025, FR-034); invalid constructor input is an error
  satisfying `errors.Is(err, srderr.ErrInvalidInput)` (FR-021, FR-041 and the
  kernel range check, FR-030). `MustX` helpers panic with that error and are for
  test literals. No untrusted-input decoder: values come from callers.

## 6. Interface

- **Use case:** `InfluenceResolver`, built by `NewInfluenceResolver(roller Roller,
  opts ...Option) (InfluenceResolver, error)` (the error is for a nil roller) with
  the context's options `WithTelemetry(telemetry.Telemetry)` (default no-ops) and
  `WithClock(func() time.Time)`. One method: `ResolveInfluence(ctx context.Context,
  cmd InfluenceCommand) (ResolveInfluenceResult, error)` (`doc/rules/architecture.md`
  §4; `ctx` first because the call emits a span; the error is only a port
  failure, FR-025). Safe for concurrent use when the roller is.
  `character` may use `d20` (context map). The rules live in the functions
  of section 5 and in `d20`; the flow is FR-013 to FR-023.
- **Ports and doubles:** `character.Roller` (owned by `character-aggregate.md`, `ports.go`; this spec only uses it) is the one-method
  consumer-side interface `Roll(dice.Expression) (dice.Result, error)` that any
  `dice.Roller` satisfies; the use case calls it only through `dice.Roll`. No new
  port, double or contract suite. Tests use `dicetest.NewFixedRoller`,
  `dicetest.CallbackRoller` and `telemetrytest.Recorder`
  (`shared-types-and-ownership.md` FR-038). Port failures: FR-025, FR-026.
  Exports: FR-039.

## 7. Observability

The use case is instrumented, the domain is silent (`doc/rules/observability.md`
§3). The four signals below are owned here; shared keys are cited in FR-037.

Evaluation: a failing dice source is the only failure; a
refusal or failed check is a normal outcome. Operators see span error status and
`influence_errors` with `error.type`, counted once here (outermost use case);
the result carries approach, attitude, DC, basis and full check, so signals hold
only the low-cardinality summary. Nothing is double counted: no `d20.Resolver.Resolve`
call, so no `srd5e.d20_test.*` signal fires (`shared-types-and-ownership.md`
FR-044). Consumers monitor error-status spans, `influence_errors`, latency, and a
Friendly `check_succeeded` share not above the Indifferent one (or a Hostile share far above it). Counted: calls and outcomes, errors; timed: duration, also on failure. A negative GM wait is rejected by the constructor, before any signal. No log is emitted
(FR-045).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.resolve_influence` | span | `srd5e.influence.approach`, `srd5e.influence.attitude` (hesitant only, at start); `srd5e.influence.response`, `srd5e.outcome`, `srd5e.d20_test.roll_mode` (only when a d20 was rolled) | See one Influence resolution; error status when the dice source fails | Alert when the error-status span rate stays above 0 for 5 minutes |
| `srd5e.character.influence_resolved` | counter, unit `{influence}` | `srd5e.influence.response` (`fulfilled`, `refused`, `check_succeeded`, `check_failed`), `srd5e.outcome`; `srd5e.influence.approach`, `srd5e.influence.attitude` for hesitant calls only; `srd5e.d20_test.roll_mode` for rolled attempts only | Throughput and outcome mix | Dashboard the `check_succeeded` share by attitude; investigate a Friendly share not above the Indifferent share over at least 500 hesitant calls |
| `srd5e.character.influence_errors` | counter, unit `{error}` | `error.type` | Returned port failures, counted once | Alert on any sustained increase |
| `srd5e.character.influence_duration` | histogram, unit `s` (double) | none | Latency, dominated by the injected roller; recorded on failing calls too | Alert when p99 over 15 minutes exceeds 5 times the consumer's baseline |

These entries feed `doc/observability/signals.md`.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002 | unit | `Attitude` has exactly `Friendly`, `Indifferent`, `Hostile` in order and the zero value is invalid; `MonsterDefaultAttitude` is `Indifferent`; no behaviour member; a target without an attitude equals one with explicit Indifferent and its `AttitudeCircumstances` is empty (`influence_rules_test.go`, package `character_test`) |
| FR-003 to FR-005 | unit | One `Advantage` for Friendly; one `Disadvantage` for Hostile; empty list for Indifferent |
| FR-006, FR-007 | unit | Table-driven: five approaches, their skills, exact count; `Ability()` equals `Skill().Ability()` |
| FR-008 | unit | Suggestion cases table, every row |
| FR-009 | application | `Coax` on a Humanoid and `Persuade` without comprehension still make one check |
| FR-010 | unit | Default DC table, every row |
| FR-011, FR-012 | application | GM-set DC of 8 and 25 against Intelligence 10 and 20: target equals the GM-set DC; `DcBasis` `DefaultRule` and `GmSet` on a success and a failure |
| FR-013, FR-014 | application | `FixedRoller.Requests()` empty; result `ResolveInfluenceFulfilled` or `ResolveInfluenceRefused` |
| FR-015 | application | Real `d20` functions over `dicetest.NewFixedRoller`: ability modifier per approach (Charisma and Wisdom), target `DifficultyClass`, exactly one request of `1d20` or `2d20` with modifier 0, `Roll` faces in served order |
| FR-016 | application + scenario | Roll mode table through `dicetest.NewFixedRoller`: assert `Roll.Mode()` and `Requests()`; scenario runs a Friendly and a Hostile urging through the public API |
| FR-017 | application | Ability modifier only: total is natural d20 plus modifier (`[1d20]`); ability plus Proficiency Bonus plus other modifiers `[+3, -1]` reach the total in order |
| FR-018 to FR-020, FR-023 | application | Resolution table rows: equal, one below, natural 20 below DC, natural 1 reaching DC, Friendly higher die, Hostile lower die, GM-set waits; default 24 hours |
| FR-021 | application | `-1` second: `NewInfluenceHesitantCommand` returns an error matching `srderr.ErrInvalidInput` |
| FR-022 | application | `InfluenceAutomaticFailure`: no roller request; `ResolveInfluenceCheckForcedFailure` carries a `ForcedFailureResult` with reason `automatic`; `RetryAfter` 24 hours; no attitude circumstance applied |
| FR-024 | application | Both non-check variants have no fields (equal instances, no attitude-change member) |
| FR-025, FR-026 | application | `dicetest.CallbackRoller` returning an error: the `*srderr.PortError` returned wraps it (`errors.Is`), zero result, one request recorded; a panicking roller gives a `*srderr.PortError` matching `srderr.ErrPortPanic`; a roller answering a different expression gives one matching `srderr.ErrContractViolation`; a drained `FixedRoller` gives a port failure; `errors.Is(err, context.Canceled)` is preserved for a cancelled `ctx` |
| FR-027 | application | Equal results across repeated calls with all 1s and all 20s, each attitude |
| FR-028 | application | `InfluenceRoll` copy semantics (mutate the caller's slice and map afterwards, mutate each returned copy), command unchanged after the call |
| FR-029, FR-030 | unit | Equality table (`==` / `Equal`), one differing field per row; scores 1 and 30 through `DefaultDc` (15 and 30); the `ability.Score` signature is compile-time, range rejection is tested in the kernel (`ability-scores.md` FR-002) |
| FR-031 to FR-033 | application | `telemetrytest.Recorder` (`Tracer.Spans()`, `Meter.Measurements()`): span name, one span, attributes, no error status on refusal or failed check; one counter increment with the hesitant and non-hesitant attribute sets; one duration measurement inside the span, on success and on a failing roller, no attribute (a `telemetrytest.Clock` is injected with `WithClock`) |
| FR-034 | application | Port failure: span error status with the error recorded, `influence_errors` once with `error.type` `dice_roller_error`, no log, no `influence_resolved`; a cancelled `ctx` is not recorded as a failure outcome; a panic in a telemetry double propagates (`errors-and-results.md` FR-051); the `errors-and-results.md` FR-031 and FR-034 rows |
| FR-035, FR-037 | unit | Enum-to-string mapping table, every value, both failure variants mapping to `check_failed`; literal values of every constant pinned |
| FR-036 | application | `telemetry.Noop` and a recording `telemetrytest.Recorder` give equal results, including a failing roller |
| FR-038, FR-039, FR-044 | architecture | Import-direction scan (`internal/archtest`: no `combat`, no `d20.Resolver`); export scan (each name once, doc comment, no upstream alias); use-case scan (`architecture-guardrails.md` FR-097) |
| FR-040, FR-043 | application | Each check variant carries approach, attitude, `DifficultyClass`, `DcBasis`, check result type and, for a failure, `RetryAfter`; `roll_mode` on span and counter equals `Roll.mode`, absent for forced failure, fulfilled and refused |
| FR-041, FR-042 | unit + application | Map with Charisma +4 and Wisdom -1: `Coax` uses -1, `Persuade` uses +4; missing ability returns an error matching `srderr.ErrInvalidInput`; extra entries ignored; `InfluenceApproach` declares no ability member |
| FR-045 | application | `telemetrytest.Recorder` logger at the finest level records zero events on every row |

- **Mutation testing** (`doc/rules/testing.md` §8): required for the domain files
  (`influence_rules.go`, `influence_approach.go`, `influence_target.go`, `attitude.go`; `gomu` only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`, not run by this spec);
  boundary rows (Intelligence 14, 15, 16; total equal and one below; wait of
  zero) kill comparison and constant mutants.
- **Scenario:** a Hostile Beast, a `Coax` approach and Advantage from a caller,
  through the public packages only (`scenarios/influence_test.go`, package `scenarios_test`). Every FR id is covered.

## 9. Open questions and decisions

Open: none.

Decided: Q-06 a constructor from a `content-monsters.md` definition to
`InfluenceTarget` is host code, added when that spec ships; the engine
exposes no such API (NPC and monster are one thing, p. 185).

Settled:

- Q-01 `Attitude` is owned here (not in `shared-types-and-ownership.md` FR-005
  or FR-046; `rules-glossary-coverage.md` GL-013, GL-069, GL-082, GL-088).
- Q-02, Q-03, Q-11, Q-12 Type names and ownership: `shared-types-and-ownership.md`
  FR-001, FR-002, FR-004, FR-005, FR-044, FR-046; `d20-tests.md` section 5.
- Q-04 No attitude change is modelled: the SRD states no rule (p. 184).
- Q-05 No memory of "the same way again": host state (`RetryAfter` and approach
  are in the result). Q-07 Outermost use case, records the failure once
  (`shared-types-and-ownership.md` FR-044; FR-034). Q-10 Leaf in `character`. Sibling edits not made here: roadmap, `AGENTS.md` `character/` line, `architecture.md` §2 `character` Purpose cell (add social interaction).
- Q-08, Q-09 Charmed and the action itself are host composition
  (`combat-action-resolution.md` Q-23); no edge to `combat`
  (`shared-types-and-ownership.md` FR-013).
- Q-13 The `map[ability.Ability]modifier.Modifier` with one validated constructor check
  (FR-041) is kept: one `InfluenceRoll` can be built before the approach is
  chosen and a Charisma-for-Wisdom mix-up is impossible.

## 10. Changelog impact

Under `Unreleased` / **Added**: social interaction in `character` (`Attitude`,
`ResolveInfluence`, `InfluenceApproach`, `InfluenceTarget`, `InfluenceRules`,
the Influence commands and results) and the signals
`srd5e.character.resolve_influence`, `srd5e.character.influence_resolved`,
`srd5e.character.influence_errors` and `srd5e.character.influence_duration`.
Not **BREAKING**: all additions.
