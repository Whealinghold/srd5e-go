# Resolve a D20 Test (`d20.Resolver`)

| | |
|---|---|
| Status | partially implemented |
| Context | `d20` |
| Roadmap phase | 1 |
| Depends on | `d20-tests.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `ability-scores.md`, `observability-ports.md`, `errors-and-results.md` |

Roadmap: [../roadmap.md](../roadmap.md); type phase 1 (shared-types-and-ownership.md FR-014). Binding
decisions: [shared-types-and-ownership.md](shared-types-and-ownership.md). `ability.Score` and its derived
modifier are not used (callers pass the finished ability modifier in `RollModifiers`; only
`ability.Ability` is used). No dependency on `character`, `combat` or `content`
(shared-types-and-ownership.md FR-013). No use case is a dependency: `Resolver` composes the pure rules
functions (`ResolveCheck`, `ResolveAttack`, `FailWithoutRoll`) and is never called by another use case
(shared-types-and-ownership.md FR-044).

## 1. Purpose and scope

`Resolver` is the first use case of the module. It composes the pure rules functions of `d20-tests.md` and
is a use case **for hosts**: no other use case calls it (shared-types-and-ownership.md FR-044;
`doc/rules/architecture.md` §4). This is the **use-case-layer spec**: how it orchestrates the dice
source, handles port failure, its closed result `Result` and its full telemetry contract. It is the
**single authority for the `d20` signals** (shared-types-and-ownership.md FR-040) and the **reference
template for later use cases** (constructor injection through small consumer-side interfaces and functional
options, error handling, result naming, telemetry shape).

Given a `Command`, `Resolver.Resolve` resolves the roll mode from the attempt's circumstances (FR-006),
builds the roll through the dice source (one request, FR-007), hands the faces to the rules functions, and
returns the `Result` carrying the whole breakdown. It contains no rules logic. Domain functions and use case
live in one Go package, `d20`, with unexported helpers and one file per concept.

**Ownership (shared-types-and-ownership.md FR-001, FR-002, FR-017).**

| Concern | Owner |
|---|---|
| `Roll` (incl. reroll and replace-a-die), `RollModifiers`, `DifficultyClass`, `ArmorClass`, `Kind`, `Outcome`, `CheckResult`, `AttackRollResult`, `ForcedFailureResult`, `ForcedFailureReason`, `RerollDecision` | [d20-tests.md](d20-tests.md) (decisions: `shared-types-and-ownership.md` FR-002) |
| Pure rules functions `ResolveCheck` (FR-063), `ResolveAttack` (FR-064), `FailWithoutRoll` (FR-035, FR-036, FR-052); no state, no ports | [d20-tests.md](d20-tests.md) (decisions FR-002, FR-044) |
| Every rule requirement (used die, total, success/failure, natural 20/1, forced failure, one Proficiency Bonus via the shape of `RollModifiers`) | [d20-tests.md](d20-tests.md); roll-mode combination and used-die selection by [modifiers-and-advantage.md](modifiers-and-advantage.md) (`modifier.Resolve`, `RollMode.Select`) |
| `modifier.RollMode`, `modifier.RollCircumstance`, `modifier.Modifier`, `ability.Ability`, `dice.Expression`, `dice.Die` | kernel packages (decisions FR-001); this spec defines none |
| **Command and attempt types** (`Command`, `AbilityCheckCommand`, `SavingThrowCommand`, `AttackRollCommand`, `CheckAttempt`, `SaveAttempt`, `RollAttempt`, `ChooseToFail`, `AutomaticFailure`), `Resolver`, `NewResolver`, `Option`, `Result` and variants, the single `modifier.Resolve` call, dice requested, port failure handling, no-op versus recording identity, command immutability, the telemetry catalogue and its constants | this spec |
| Heroic Inspiration | Not owned, not a dependency (decisions FR-010, FR-048); reroll operations on `Roll` take the decision as input |

This spec states no rule twice: where a rule matters here it is a "delegated" row in 4.3.
`doc/observability/signals.md` takes the catalogue from section 7 only.

**Out of scope:**

- Deriving the ability modifier, Proficiency Bonus, proficiencies, and choosing the ability
  (SRD 5.2.1, "The Six Abilities", p. 5; "Proficiency", p. 8).
- **Proficiency Bonus stacking** ("The Bonus Doesn't Stack", p. 8): the use case sums the
  `RollModifiers` it is given; a doubled bonus cannot be constructed (decisions FR-003; d20-tests.md).
- Choosing the target number and Armor Class calculation (SRD 5.2.1, "Armor Class", p. 7).
- Deciding whether a D20 Test is warranted. Choosing to fail a saving throw (SRD 5.2.1, "Saving
  Throws", p. 7) and automatic failure are modelled by d20-tests.md (`ForcedFailureReason`); this use
  case only requests no die for them (FR-007).
- **Death Saving Throws** (SRD 5.2.1, pp. 17-18): owned by `combat`.
- Rerolls and replacing a die, including Heroic Inspiration (SRD 5.2.1, "Interactions with
  Rerolls", p. 8): operations on `Roll`; no use case drives them yet.
- Condition effects (Exhaustion, automatic failure, Disadvantage): applied by callers and passed as
  modifiers, circumstances or a forced failure (decisions FR-026).
- Damage and what a Critical Hit does to damage (SRD 5.2.1, "Critical Hit", p. 179).
- **Widened and automatic Critical Hits.** `IsCriticalHit` means "a natural value at or above the critical
  range", and the use case passes the default range, `DefaultCriticalRange` (a natural 20, p. 179). Wider
  ranges (Champion, "Improved Critical" and "Superior Critical", p. 49) and automatic Critical Hits (within
  5 feet of a Paralyzed or Unconscious creature, "Paralyzed [Condition]", p. 186, "Unconscious
  [Condition]", p. 191) are applied by the caller (decisions FR-026), who may call `ResolveAttack` with a
  wider range itself.
- Contests and group checks: not defined as D20 Test variants in the SRD text read.
- Automatic success or failure on a natural 20 or 1 for **ordinary** checks and saves (stated for
  attack rolls, "Rolling 20 or 1", p. 7, and for Death Saving Throws, pp. 17-18). d20-tests.md pins
  the absence.

## 2. Source

PDF page numbers equal printed page numbers in SRD 5.2.1 (PDF page 6 shows printed page 6).

- "D20 Tests", p. 6 (steps cited by name: "Roll 1d20", "Add Modifiers", "Compare the Total to a Target Number"); "Ability Checks" incl. "Difficulty Class", p. 6.
- "Saving Throws", p. 7; "Attack Rolls" incl. "Armor Class" and "Rolling 20 or 1", p. 7.
- "Advantage/Disadvantage", "Roll Two D20s", "They Don't Stack", pp. 7-8; "Proficiency", "The Bonus Doesn't Stack", p. 8.
- "Death Saving Throws", pp. 17-18 (out of scope); "Fighter Subclass: Champion", p. 49 (out of scope).
- "Blinded [Condition]", p. 177; "Paralyzed [Condition]", p. 186; "Unconscious [Condition]", p. 191 (automatic failures and Critical Hits named in the case table; conditions applied by callers).
- "Rules Glossary": "Advantage", p. 176; "Armor Class", p. 177; "Attack Roll", p. 177; "Critical Hit", p. 179; "D20 Test", p. 180; "Difficulty Class", "Disadvantage", p. 181; "Ability Check", p. 176.
- Package design: `doc/rules/architecture.md` (§3, §4, §6), `doc/rules/go.md` (§3, §4, §5, §7), `doc/rules/observability.md` (§2-§9), `doc/rules/testing.md` (§3-§5, §8), shared-types-and-ownership.md. Requirements that are not SRD rules say "package design".

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| D20 Test | closed interface `Result` (`RolledCheck`, `RolledAttack`, `ForcedFailure`) | pp. 6, 180. Declared here (decisions FR-017); the domain values it carries are d20-tests.md's. |
| Ability Check / Saving Throw / Attack Roll | `Kind` (`AbilityCheck`, `SavingThrow`, `AttackRoll`; d20-tests.md); closed interface `Command` (`AbilityCheckCommand`, `SavingThrowCommand`, `AttackRollCommand`, here) | |
| Difficulty Class / Armor Class | `DifficultyClass` / `ArmorClass` | pp. 181, 177. No union target type: each result variant carries the one it can (FR-049). |
| Modifier | `modifier.Modifier` (kernel, plain signed integer), `RollModifiers` (`d20`, structured composite) | p. 6 |
| Advantage / Disadvantage | `modifier.RollCircumstance` (`modifier.Advantage{}`, `modifier.Disadvantage{}`; modifiers-and-advantage.md FR-042), carried by `RollAttempt.Circumstances` | p. 8 |
| (neither) | `modifier.Normal` (kernel enum `modifier.RollMode`: `Normal`, `WithAdvantage`, `WithDisadvantage`) | Also the telemetry value. |
| natural 20 / natural 1 | `Roll.Natural()` | The d20 used, before modifiers (p. 7). |
| Critical Hit | `AttackRollResult.IsCriticalHit()` | Natural value at or above the critical range, 20 by default (p. 179). |
| hit / miss | `AttackRollResult.Hit()` (`Outcome() == Success`) | p. 7; no separate enum (decisions FR-004). |
| succeeds / fails | `Success`, `Failure` (`Outcome`) | One enum for every kind (decisions FR-004); this spec adds no outcome type. |

Package terms introduced here: `Resolver`, `NewResolver`, `Option` (`WithTelemetry`, `WithClock`), `Roller`
(the consumer-side one-method dice interface), the command and attempt types, `Result` with its three
variants, `InvalidError` and `Reason`, and the signal constants (FR-040). None collides with an SRD term.

## 4. Requirements

Ids are stable. A "delegated" requirement is owned by the named requirement elsewhere; its row exists
so earlier ids stay stable and the use-case-level consequence is explicit. Withdrawn ids are kept in
"Withdrawn ids" at the end of this section and never reused. Earlier drafts called the operation
`ResolveD20Test` and its pieces `D20Test*`; the Go names drop the stutter the package name already carries
(`d20.Resolver`, `d20.Result`, `d20.Kind`).

### 4.1 Command

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | The command MUST be a closed `Command` interface (unexported methods, so only this package implements it; section 5) whose variants pair a kind with its target and attempt type: `AbilityCheckCommand{Ability, Target DifficultyClass, Attempt CheckAttempt}`, `SavingThrowCommand{Ability, Target DifficultyClass, Attempt SaveAttempt}`, `AttackRollCommand{Ability, Target ArmorClass, Attempt RollAttempt}`, all values. A DC cannot be paired with an attack roll, nor an AC with a check or save, and an attack roll cannot be forced to fail (d20-tests.md FR-052 rejects it in the domain too); the compiler rejects each mismatch. `CheckAttempt` admits `RollAttempt` and `AutomaticFailure`; `SaveAttempt` admits `RollAttempt`, `ChooseToFail` and `AutomaticFailure` (a creature may choose to fail a saving throw, SRD 5.2.1, "Saving Throws", p. 7; "not a check" is package design, derived from the SRD naming only saves as optional). A nil or foreign command, or a nil or foreign attempt (including a pointer to a variant), is untrusted input: `Resolve` returns an `*InvalidError` with `ReasonUnknownCommand` or `ReasonMissingAttempt` before any signal is emitted (FR-062). | SRD 5.2.1, "D20 Tests", p. 6 | happy; unhappy (invalid input) |
| FR-002 | A minimal command (a `RollAttempt` whose `RollModifiers` holds only the mandatory ability modifier and whose `Circumstances` is empty, so `modifier.Normal`) MUST resolve with one d20 requested and a total equal to the d20 used plus the ability modifier. | SRD 5.2.1, "D20 Tests", p. 6 | unhappy (empty and minimal) |
| FR-005 | A command MUST be a plain value: its exported fields are the whole input (`Ability`, `Target`, the attempt's type, and for a `RollAttempt` its `Modifiers` and `Circumstances`, in order), copying it copies those fields, and two commands built from the same inputs resolve identically (FR-025). Go has no hash code, and `RollAttempt` holds a slice, so a command is not comparable with `==`; tests compare field by field (`slices.Equal`, `RollModifiers.Equal`). `ChooseToFail{}` values are equal, as are `AutomaticFailure{}` values. | `doc/rules/architecture.md` §3 | happy |
| FR-055 | `Ability` on a command is a label for telemetry (`srd5e.ability`) and the result only. The use case does not read it to compute anything: the modifier summed is the one inside `RollModifiers`. The caller MUST keep the two consistent; the use case cannot check it, because `modifier.Modifier` has no ability tag (d20-tests.md D-06, answered here). Only an `Ability` outside the six (`!Ability.Valid()`) is rejected, with `ReasonUnknownAbility`. Mechanism: a test builds a command whose `Ability` and modifier disagree and asserts the total uses the modifier and the attribute uses the label. | package design; d20-tests.md D-06 | unhappy (rule interaction) |
| FR-056 | `Resolve` MUST NOT modify the command: it reads `RollAttempt.Circumstances` and never changes the slice, and `RollModifiers` copies its slices on construction and on access (`WithOthers` copies, `Others` returns a clone; d20-tests.md), so mutating the caller's slice afterwards changes nothing already built and mutating a returned copy changes nothing held. `RollAttempt` declares `Modifiers RollModifiers` and `Circumstances []modifier.RollCircumstance`; `ChooseToFail` and `AutomaticFailure` carry no fields. | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |

### 4.2 Roll mode and dice requested

For a rolling attempt the use case calls the kernel's `modifier.Resolve` on `RollAttempt.Circumstances`
exactly once (modifiers-and-advantage.md; they do not stack, one of each cancels) and uses that one
`RollMode` for the request (FR-007) and the `Roll`. The combination rule is the kernel's.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-006 | Roll mode derived from circumstances, including "They Don't Stack" and cancellation, is owned by modifiers-and-advantage.md (FR-010 to FR-017, `modifier.Resolve`). The use case MUST derive the mode with `modifier.Resolve` and use that one mode both to size the dice request (FR-007) and as the `Mode()` of the `Roll`. Testable: request size and `Roll.Mode()` equal `modifier.Resolve(circumstances)` for the case-table rows. "Exactly once" and "MUST NOT count, filter or combine circumstances itself" are review-only (the call count is not observable through the public API). | SRD 5.2.1, "They Don't Stack", p. 8 | happy (rule owned elsewhere) |
| FR-007 | For a command that rolls, the use case MUST make exactly one request to the `Roller`, through `dice.Roll`, for `dice.MustExpression(mode.DiceCount(), dice.D20(), 0)`: a single `1d20` in `Normal`, a single `2d20` in `Advantage` or `Disadvantage`. For `ChooseToFail` or `AutomaticFailure` it MUST make no request. The faces, in order, are the dice handed to the domain. | SRD 5.2.1, "D20 Tests", step "Roll 1d20", p. 6; "Roll Two D20s", p. 8; dice-and-rolls.md FR-050 | happy |
| FR-008 | Delegated. The d20 used is the higher die under Advantage and the lower under Disadvantage (`RollMode.Select`, modifiers-and-advantage.md FR-021 to FR-027; d20-tests.md FR-011). The use case MUST NOT re-select the die. | SRD 5.2.1, "Roll Two D20s", p. 8 | happy (rule owned elsewhere) |
| FR-009 | Delegated. Equal dice give that value and both dice remain in the result (d20-tests.md FR-011; kept index FR-058, `Roll.KeptIndex`). The SRD example does not state the equal case; it follows from "use the higher / lower" (package-level derivation). | derived from SRD 5.2.1, "Roll Two D20s", p. 8 | unhappy (boundary, rule owned elsewhere) |

### 4.3 Total and outcome (delegated rules and end-to-end checks)

Owned and unit-tested in d20-tests.md; the use case MUST NOT reimplement them. Every kind resolves to
`Success` or `Failure`; an attack roll result adds `IsCriticalHit` and reads `Hit` as
`Outcome() == Success` (decisions FR-004).

| Id | Requirement as seen from the use case (delegated rule owner first) | Source | Path |
|---|---|---|---|
| FR-011 | Delegated to d20-tests.md FR-023 to FR-026: A check or save is `Success` when the total is >= the DC, otherwise `Failure`. | SRD 5.2.1, "D20 Tests", step "Compare the Total to a Target Number", p. 6 | happy |
| FR-012 | Delegated to d20-tests.md FR-027, FR-028: An attack roll whose d20 used is neither 1 nor 20 is a hit (`Success`) when the total is >= the AC, otherwise a miss (`Failure`), and `IsCriticalHit` is false. | SRD 5.2.1, "Attack Rolls", p. 7 | happy |
| FR-013 | Delegated to d20-tests.md FR-029: A natural 20 on the d20 used is a hit with `IsCriticalHit` true, regardless of modifiers or AC. | SRD 5.2.1, "Rolling 20 or 1", p. 7; "Critical Hit", p. 179 | unhappy (boundary) |
| FR-014 | Delegated to d20-tests.md FR-030: A natural 1 on the d20 used is a miss with `IsCriticalHit` false, regardless of modifiers or AC. | SRD 5.2.1, "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| FR-015 | Delegated to d20-tests.md FR-031: For ordinary checks and saves a natural 20 MUST NOT cause automatic success and a natural 1 MUST NOT cause automatic failure; only the DC comparison decides. Death Saving Throws are out of scope. | SRD 5.2.1, "Rolling 20 or 1", p. 7 (under "Attack Rolls") | unhappy (rule interaction) |
| FR-016 | Delegated to d20-tests.md FR-029, FR-031, FR-033: The use case MUST report `IsCriticalHit` true only for a natural 20 on an attack roll; a check or save carries no critical-hit field (`CheckResult` has no such method); an attack roll with a d20 used of 2 to 19 has it false. | SRD 5.2.1, "Critical Hit", p. 179 | unhappy (rule interaction) |
| FR-017 | Delegated to d20-tests.md FR-032: Natural 20 and natural 1 are judged on the d20 used after Advantage or Disadvantage, not on the discarded die. | SRD 5.2.1, "Roll Two D20s", p. 8; "Rolling 20 or 1", p. 7 | unhappy (rule interaction) |

Two requirements stay at the use-case level because they pin that nothing between port and result
alters the numbers.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | The result's total MUST equal the d20 used plus the sum of the modifiers (d20-tests.md FR-016, FR-019), negative modifiers reducing it, and the use case MUST NOT clamp it: a total below zero or above 30 is reported as computed. | derived from SRD 5.2.1, "D20 Tests", step "Add Modifiers", p. 6; the SRD is silent on clamping, so no clamping is a package decision | unhappy (boundary) |
| FR-028 | Extreme rolls MUST resolve through the whole use case: a fixed roller giving 1 and 20 on a Normal roll, and 1 and 1, 20 and 20 on both dice under Advantage and under Disadvantage, MUST each produce the case-table outcome below. | `doc/rules/testing.md` §4 | unhappy (boundary) |

End-to-end case table (one `t.Run` subtest per row in `TestResolveCaseTable`, real rules, `dicetest.FixedRoller`;
"d20" is the d20 used; rolls in request order; every row has the mandatory ability modifier). "Crit" is
`IsCriticalHit` for an attack roll and "n/a" for a check or save. A forced failure names its
`ForcedFailureReason`.

| Label | Kind | Mode | Rolls | Modifiers | Target | Total | Expected outcome | Crit | FR | Source | Path |
|---|---|---|---|---|---|---|---|---|---|---|---|
| check meets DC | check | normal | 10 | ability +5 | DC 15 | 15 | success | n/a | FR-007, FR-010 | SRD 5.2.1, "D20 Tests", p. 6 | happy |
| save fails | save | normal | 8 | ability +1 | DC 10 | 9 | failure | n/a | FR-007, FR-010 | SRD 5.2.1, "Saving Throws", p. 7 | happy |
| attack hits | attack | normal | 12 | ability +3 | AC 15 | 15 | success | false | FR-007, FR-010 | SRD 5.2.1, "Attack Rolls", p. 7 | happy |
| total above 30 | check | normal | 20 | ability +15 | DC 30 | 35 | success | n/a | FR-010, FR-028 | derived from SRD 5.2.1, "D20 Tests", p. 6 | unhappy (boundary) |
| total below zero | check | normal | 1 | ability -5, other -2 | DC 5 | -6 | failure | n/a | FR-010, FR-028 | derived from SRD 5.2.1, "D20 Tests", p. 6 | unhappy (boundary) |
| extreme low, normal | check | normal | 1 | ability +0 | DC 5 | 1 | failure | n/a | FR-028 | SRD 5.2.1, "D20 Tests", p. 6 | unhappy (boundary) |
| extreme high, normal | attack | normal | 20 | ability +0 | AC 30 | 20 | success | true | FR-013, FR-028 | SRD 5.2.1, "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| Advantage uses the higher die | check | advantage | 18, 3 | ability +0 | DC 15 | 18 | success | n/a | FR-007, FR-008 | SRD 5.2.1, "Roll Two D20s", p. 8 | happy |
| Disadvantage uses the lower die | check | disadvantage | 18, 3 | ability +0 | DC 15 | 3 | failure | n/a | FR-007, FR-008 | SRD 5.2.1, "Roll Two D20s", p. 8 | happy |
| Advantage, both dice 1 | attack | advantage | 1, 1 | ability +20 | AC 5 | 21 | failure | false | FR-010, FR-028 | SRD 5.2.1, "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| Disadvantage, both dice 20 | attack | disadvantage | 20, 20 | ability -3 | AC 30 | 17 | success | true | FR-010, FR-028 | SRD 5.2.1, "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| Advantage, both dice 20 | attack | advantage | 20, 20 | ability -3 | AC 30 | 17 | success | true | FR-010, FR-028 | SRD 5.2.1, "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| Disadvantage, both dice 1 | attack | disadvantage | 1, 1 | ability +20 | AC 5 | 21 | failure | false | FR-010, FR-028 | SRD 5.2.1, "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| Disadvantage, natural 1 used | attack | disadvantage | 20, 1 | ability +10 | AC 5 | 11 | failure | false | FR-007, FR-017, FR-028 | SRD 5.2.1, "Roll Two D20s", p. 8 | unhappy (rule interaction) |
| Advantage and Disadvantage cancel | check | normal (circumstances `Advantage`, `Disadvantage`) | 14 | ability +0 | DC 15 | 14 | failure | n/a | FR-006, FR-007 | SRD 5.2.1, "They Don't Stack", p. 8 | unhappy (rule interaction) |
| chosen to fail a save | save | normal (no roll) | none | none | DC 10 | no total | failure (`ForcedFailure`, `ChosenToFail`) | n/a | FR-007 | SRD 5.2.1, "Saving Throws", p. 7 | happy |
| automatic failure on a check | check | normal (no roll) | none | none | DC 10 | no total | failure (`ForcedFailure`, `Automatic`) | n/a | FR-007 | SRD 5.2.1, "Blinded [Condition]", p. 177 | happy |

### 4.4 Result

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | A rolled result MUST carry the breakdown needed to explain it without recomputation, through the section 5 accessors: every variant returns `Ability()` (the command's label), `Kind()` and `Outcome()`, and its domain result holds what d20-tests.md FR-041 and FR-034 list: for a `CheckResult` (`RolledCheck.Check()`) the kind, `Roll` (roll mode `Mode()`, dice in request order `Dice()`, `Natural()`, `KeptIndex()`), `RollModifiers`, `Total()`, `DifficultyClass` `Target()` and `Outcome()`; for an `AttackRollResult` (`RolledAttack.Attack()`) the same with an `ArmorClass` and `IsCriticalHit()`; for a `ForcedFailureResult` (`ForcedFailure.Failure()`) the kind and `Reason()`, plus the `Target()` the variant adds. The roll mode is `Roll.Mode()`; no field duplicates it. The use case MUST populate every field from what it requested and the domain returned. | `doc/rules/architecture.md` §6; SRD 5.2.1, "D20 Tests", p. 6 | happy |
| FR-020 | `Result` MUST be a closed set with exactly the three variants of section 3 (an interface with an unexported marker method, so no other package can add a variant), documented in the interface comment; a consumer's type switch covers all three, and a test pins the list. (`Outcome` being exactly `Success` and `Failure` is d20-tests.md FR-060.) | `doc/rules/architecture.md` §3, §6, §10 | happy |
| FR-021 | A result MUST be a value object: immutable, built only by the package, equal when every field is equal. Each variant has an `Equal` method (`RolledCheck.Equal`, `RolledAttack.Equal`) and is not compared with `==`; a `ForcedFailure` holds comparable fields only. | `doc/rules/architecture.md` §3 | happy |
| FR-022 | The use case MUST NOT modify its command: after `Resolve` returns, the command (`Ability`, `Target`, the attempt's `Modifiers` and `Circumstances`) and the collections it was built from MUST equal what they were before. | `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-043 | Delegated. `RolledCheck` has no kind of its own and no guard: the kind is `CheckResult.Kind()`, and `ResolveCheck` (d20-tests.md FR-062, FR-069) returns an `*InvalidError` with `ReasonKindMismatch` (matching `srderr.ErrInvalidInput`) for kind `AttackRoll`, through a real check; this spec does not require the zero `CheckResult` to reject anything. The other impossible combinations (a check carrying an `AttackRollResult`, an attack roll carrying a `DifficultyClass`) are unrepresentable by the field types. | `doc/rules/architecture.md` §6; d20-tests.md FR-062 | unhappy (invalid input) |
| FR-044 | Result types MUST NOT carry display text. Mechanism: review, plus the fact that every field is an enum, number or value type; no result type has a `string` field or a method returning display text (the `String()` methods of the enums are the telemetry values, FR-038). An architecture rule for this is not yet implemented (see "Not covered"). | `doc/rules/architecture.md` §3, §6, §10 | architecture guardrail (review) |
| FR-049 | The target in a result MUST be typed by variant: `RolledCheck` (through `Check().Target()`) and `ForcedFailure` (`Target()`) carry a `DifficultyClass`, `RolledAttack` (through `Attack().Target()`) an `ArmorClass`. No union or `any` target type exists. | SRD 5.2.1, "D20 Tests", p. 6; "Armor Class", p. 7 | happy |
| FR-053 | The result MUST be named `Result` and its variants `RolledCheck`, `RolledAttack` and `ForcedFailure`: package-qualified (`d20.RolledCheck`) so no operation prefix is needed and nothing stutters. Generic names that would collide across contexts (`Applied`, `Rejected`, `Succeeded`, `Rolled`) MUST NOT be exported from `d20`; the `internal/archtest` `stutter` and `consumer-name` rules cover the stutter half. | shared-types-and-ownership.md FR-017; `doc/rules/architecture.md` §4 | unhappy (name collision) |

### 4.5 Collaborators, determinism and failure

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-023 | `NewResolver(roller Roller, opts ...Option) (Resolver, error)` MUST take its dice source through its constructor, typed as the one-method interface `d20.Roller` declared in this package (`Roll(dice.Expression) (dice.Result, error)`; any `dice.Roller` satisfies it, and so does a hand-written fake), and telemetry through `WithTelemetry(telemetry.Telemetry)`, defaulting to the zero `Telemetry` and thus to no-ops (observability-ports.md FR-042); omitting the option MUST work. A nil `roller` returns an `*InvalidError` with `ReasonMissingRoller` (matching `srderr.ErrInvalidInput`) and the zero `Resolver`, which is not usable. `WithClock(func() time.Time)` sets the clock for the duration histogram (FR-031); a nil clock is ignored. | `doc/rules/architecture.md` §4; `doc/rules/observability.md` §4; `doc/rules/go.md` §5 | happy; unhappy (invalid input) |
| FR-045 | Structural purity: no non-test file of `d20` MAY read the wall clock except through the injected clock (`time.Now` appears only as the default value in `NewResolver`), call a `math/rand` function, spawn a goroutine, hold a package-level mutable variable, or construct a `dice.Roller` adapter or a recording `Telemetry`; referencing the zero-value `Telemetry` default is allowed. Mechanism: `internal/archtest` rules `hidden-effect`, `stdlib-denied` and `package-level-var` (`doc/rules/architecture.md` §9). "Does not branch on whether observability is enabled" is review-only; FR-039 covers it behaviourally. | `doc/rules/architecture.md` §3, §4, §9; `doc/rules/observability.md` §4 | architecture guardrail |
| FR-057 | `Resolver` MUST obtain every result by calling the pure rules functions (`ResolveCheck`, `ResolveAttack`, `FailWithoutRoll`; d20-tests.md FR-069, FR-063, FR-064, FR-035, FR-036, FR-052) and MUST NOT call, construct or hold another use case. No use case in any context may call `Resolver`; one needing a D20 Test result calls the rules functions and records its own signals. Mechanism: `internal/archtest` allow-list (a context imports `d20`'s published API only; `d20` imports no context) and review; `TestResolverRunsOnFakesOfEveryDependency` shows the use case runs on fakes of every dependency. | shared-types-and-ownership.md FR-044; `doc/rules/architecture.md` §4 | architecture guardrail |
| FR-058 | The use case MUST orchestrate as follows: (a) the `Kind` derives from the command variant (`AbilityCheckCommand` gives `AbilityCheck`, `SavingThrowCommand` `SavingThrow`, `AttackRollCommand` `AttackRoll`); (b) `ChooseToFail` maps to `ChosenToFail`, `AutomaticFailure` to `Automatic`; (c) a `RollAttempt` calls `ResolveCheck` for a check or save and `ResolveAttack` for an attack roll, while `ChooseToFail` or `AutomaticFailure` calls `FailWithoutRoll` (d20-tests.md FR-069, FR-035, FR-036); (d) the domain result is wrapped with the command's `Ability` and, for a forced failure, its `Target`. | package design; d20-tests.md FR-035, FR-036, FR-069 | happy |
| FR-024 | `Resolve(ctx context.Context, cmd Command) (Result, error)` MUST be synchronous and return the result directly, because the dice port is synchronous; it takes a `context.Context` first because it emits spans (decisions 2b), passes it to telemetry only and never stores it. | `doc/rules/architecture.md` §4; `doc/rules/go.md` §4 | happy |
| FR-025 | Given equal commands and a `Roller` yielding the same values in the same order, `Resolve` MUST return equal results (`Equal`) on every invocation, in any test order (`-shuffle=on`). A `Resolver` holds no state beyond its injected ports and is safe for concurrent use when the `Roller` is. | `doc/rules/testing.md` §4 | happy |
| FR-026 | When the `Roller` returns an error, `Resolve` MUST return a nil `Result` and a `*srderr.PortError` (`Port` `dice.Port`, `Op` `roll`) that wraps the roller's own error, so `errors.Is(err, cause)` holds (not converted into a result, no partial result). `dice.Roll` does the wrapping (`srderr.WrapPort`; dice-and-rolls.md FR-048, FR-052); the use case does not wrap twice. | `doc/rules/architecture.md` §4, §6; dice-and-rolls.md FR-048, FR-052 | unhappy (collaborator failure) |
| FR-051 | After the `Roller` has failed, `Resolve` MUST NOT request dice again (no retry, no fallback roller): a failing call leaves exactly one request on record. | `doc/rules/architecture.md` §4 | unhappy (collaborator failure) |
| FR-027 | When the `Roller` returns a `dice.Result` for a different expression than requested (a different count or die, or the zero result), `Resolve` MUST return a `*srderr.PortError` matching `srderr.ErrContractViolation` (raised by `dice.Roll`) and MUST NOT return a result built from it. A d20 value of 0 or 21 cannot reach the use case: the kernel's `dice.NewDieRoll` and `dice.NewResult` reject out-of-range faces (dice-and-rolls.md FR-031 to FR-035). | dice-and-rolls.md FR-050, FR-051; `doc/rules/architecture.md` §6 | unhappy (collaborator returns invalid data) |
| FR-037 | A panic by the `Roller` (an earlier draft treated it as a "defective adapter `Error`") is a port failure in Go: `dice.Roll` recovers it through `srderr.GuardValue` into a `*srderr.PortError` wrapping a `*srderr.PanicError` that matches `srderr.ErrPortPanic`. `Resolve` MUST return that error like any other port failure (recorded once, FR-036, `roller_failure` `unavailable`, FR-061) and MUST NOT re-panic. A panic raised by this module's own code is a defect and propagates; the deferred `End` and timer `Stop` still run (FR-012). | `doc/rules/observability.md` §7; `doc/rules/architecture.md` §6; errors-and-results.md FR-030 | unhappy (state error) |
| FR-062 | Untrusted input that fails validation MUST be reported before any roll or signal as a `*InvalidError` (matching `srderr.ErrInvalidInput`) carrying a `Reason`: `ReasonUnknownCommand`, `ReasonMissingAttempt`, `ReasonUnknownAbility`, `ReasonMissingRoller`, plus the rule-level reasons returned by the rules functions (`ReasonKindMismatch`, `ReasonFaceOutOfRange`, ...). The error text is lowercase, begins `d20:` and is never branched on; callers use `errors.Is` or `errors.As`. `Resolve` never panics on untrusted input. No signal is emitted for such a call (no span, no `resolved`, no `errors`). | `doc/rules/go.md` §3; errors-and-results.md | unhappy (invalid input) |

### 4.6 Telemetry

Names are defaults; the constants are in section 7, the single source for `doc/observability/signals.md`.
Attribute values follow decisions FR-041.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-029 | Each `Resolve` call that passes validation MUST run inside one span named `srd5e.d20_test.resolve` obtained through `Tracer.Start`, with `srd5e.d20_test.kind` and `srd5e.ability` set at span start, and `srd5e.d20_test.roll_mode` and `srd5e.outcome` set once the result is known, plus `srd5e.d20_test.critical_hit` for an attack roll only (FR-052). For a `ForcedFailure` no d20 was rolled and the domain result carries no mode, so `roll_mode` MUST be absent (as `critical_hit` is) and `outcome` is `failure`; `roll_mode` therefore always equals `Roll.Mode()` when present. The span ends via `defer`. | `doc/rules/observability.md` §6; decisions FR-041 | happy |
| FR-030 | Each successful `Resolve` MUST increment the counter `srd5e.d20_test.resolved` exactly once with the same attributes as the span, whatever the outcome; a `failure` is a normal outcome. | `doc/rules/observability.md` §5 | happy |
| FR-031 | Each validated `Resolve` MUST record its duration, in seconds, into the histogram `srd5e.d20_test.duration` through a `telemetry.Timer` whose `Stop` is deferred, exactly once, with `srd5e.d20_test.kind`, **including a call that returns an error** (recorded, the same error returned; observability-ports.md FR-029, FR-030). The use case reads time only through the injected clock (`WithClock`; default `time.Now`, supplied by `NewResolver`); the rules functions read none. | `doc/rules/observability.md` §4, §5; observability-ports.md FR-029, FR-030 | happy / unhappy (collaborator failure) |
| FR-032 | Each successful `Resolve` that rolled MUST record the number of modifiers in its `RollModifiers` (`Count()`: ability modifier, Proficiency Bonus when present, each other `Modifier`) into the histogram `srd5e.d20_test.modifiers` (unit `{modifier}`) with `srd5e.d20_test.kind`; the minimum is 1. A `ForcedFailure` MUST record nothing there. | `doc/rules/observability.md` §5 (sizes) | unhappy (empty and minimal) |
| FR-047 | The use case MUST emit no log event at any level, on success or failure, and takes the `Logger` from `Telemetry` without using it. | `doc/rules/observability.md` §1.4, §7; decisions FR-043 | happy |
| FR-035 | A `failure` outcome of any kind (including a missed attack roll) MUST NOT set the span status to error, MUST NOT increment the `errors` counter and MUST NOT be reported as an error in any signal. | `doc/rules/observability.md` §6, §7 | unhappy (rule interaction) |
| FR-036 | When the dice source fails (a returned error, a panic or a contract violation), the failure MUST be recorded exactly once: `span.RecordError(err)` and status `StatusError`, `srd5e.d20_test.errors` incremented once with `srd5e.d20_test.kind` and `error.type` equal to `srderr.ErrorType(err)` (`dice_roller_error` for the dice port), and `srd5e.d20_test.resolved` not incremented. | `doc/rules/observability.md` §7, §9; errors-and-results.md FR-034 | unhappy (collaborator failure) |
| FR-038 | Every attribute value MUST come from a closed set (decisions FR-041): `srd5e.d20_test.kind` in {`ability_check`, `saving_throw`, `attack_roll`} (`Kind.String()`); `srd5e.ability` in {`strength`, `dexterity`, `constitution`, `intelligence`, `wisdom`, `charisma`}, the lower-case name of the command's `ability.Ability` (the test pins all six); `srd5e.d20_test.roll_mode` in {`normal`, `advantage`, `disadvantage`}, absent for a `ForcedFailure`; `srd5e.outcome` in {`success`, `failure`}; `srd5e.d20_test.critical_hit` a `bool`, never a string; `srd5e.d20_test.roller_failure` in {`unavailable`, `invalid_result`}; `srd5e.d20_test.forced_failure_reason` in {`chosen_to_fail`, `automatic`}; `error.type` equal to `srderr.ErrorType(err)` (`dice_roller_error`, errors-and-results.md). Attributes MUST NOT carry ids, totals, rolls, target numbers or free text. | `doc/rules/observability.md` §5 | happy / unhappy (boundary: closed value sets) |
| FR-039 | With no telemetry supplied, and with a recording one, `Resolve` MUST return equal results for the same command and roller values, make the same request, and return an error matching the same sentinel for the same failing roller. | `doc/rules/observability.md` §9 | happy |
| FR-040 | Metric names, span name and attribute keys in this section MUST be defined once as exported constants in `d20/signals.go` (`SpanResolve`, `MetricResolved`, `MetricDuration`, `MetricModifiers`, `MetricNatural`, `MetricErrors`, `AttrKind`, `AttrAbility`, `AttrRollMode`, `AttrCriticalHit`, `AttrRollerFailure`, `AttrForcedFailureReason`, and the `Value*` constants; shared keys `srd5e.outcome` and `error.type` are `telemetry.AttrOutcome` and `telemetry.AttrErrorType`), MUST NOT appear as string literals at call sites, and MUST be pinned by tests asserting their literal values (`TestSignalNamesArePinned`). | `doc/rules/observability.md` §5, §9, §10 | happy |
| FR-041 | The use case MUST NOT open child spans: the dice source is in-memory and the rules are silent, so the one span is the whole trace. | `doc/rules/observability.md` §3, §6 | happy |
| FR-046 | The `Timer` MUST be started after the span and its deferred `Stop` MUST run before the span's deferred `End` (LIFO order), so the span encloses the timed work and a failure reaches the span after the duration is recorded. | `doc/rules/observability.md` §4, §6; observability-ports.md FR-012, FR-030 | happy |
| FR-052 | `srd5e.d20_test.critical_hit` MUST be present, as a `bool`, on the span and on `resolved` for an attack roll that rolled, and absent for a check, a save and any `ForcedFailure`. Its value equals `AttackRollResult.IsCriticalHit()`. The former `srd5e.d20_test.basis` attribute and the outcome values `miss`, `hit` and `critical_hit` no longer exist. | decisions FR-004, FR-041 | unhappy (boundary: attribute presence) |
| FR-059 | A rolled call MUST record the d20 face it used (`Roll.Natural()`, 1 to 20) into the histogram `srd5e.d20_test.natural` (unit `{face}`) with `srd5e.d20_test.kind`, exactly once, through the instrument created once in `NewResolver`. A `ForcedFailure` MUST record nothing there. The face is a histogram measurement, not an attribute, so FR-038's no-rolls rule is untouched. | `doc/rules/observability.md` §5 (distributions); package design (roller-fairness diagnosis) | happy; unhappy (empty and minimal: forced rows) |
| FR-060 | A `ForcedFailure` MUST set `srd5e.d20_test.forced_failure_reason` to `chosen_to_fail` for `ChooseToFail` and `automatic` for `AutomaticFailure`, on the span and on `srd5e.d20_test.resolved`; a rolled call MUST NOT carry the key. | `doc/rules/observability.md` §5; decisions FR-041 | happy; unhappy (boundary: attribute presence) |
| FR-061 | When the dice source fails, the `srd5e.d20_test.errors` counter and the error span MUST carry `srd5e.d20_test.roller_failure`: `invalid_result` when the error matches `srderr.ErrContractViolation`, otherwise `unavailable` (a returned error or a recovered panic). | `doc/rules/observability.md` §5, §7 | unhappy (collaborator failure) |

### 4.7 Public API

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-042 | Package `d20` MUST export, each exactly once and each with a doc comment starting with its name: `Resolver`, `NewResolver`, `Option`, `WithTelemetry`, `WithClock`, `Roller`; `Result`, `RolledCheck`, `RolledAttack`, `ForcedFailure`; `Command`, `AbilityCheckCommand`, `SavingThrowCommand`, `AttackRollCommand`, `CheckAttempt`, `SaveAttempt`, `RollAttempt`, `ChooseToFail`, `AutomaticFailure`; `Outcome` (`Success`, `Failure`), `AttackRollResult`, `CheckResult`, `ForcedFailureResult`, `ForcedFailureReason` (`ChosenToFail`, `Automatic`), `Kind` (`AbilityCheck`, `SavingThrow`, `AttackRoll`); `Roll`, `NewRoll`, `MustRoll`, `RerollDecision`, `KeepRoll`, `RerollDie`, `RollModifiers`, `NewRollModifiers`; `DifficultyClass`, `ArmorClass`; the rules functions `ResolveCheck`, `ResolveAttack`, `FailWithoutRoll` and `DefaultCriticalRange`; `InvalidError`, `Reason` and the `Reason*` constants; and the signal constants of FR-040. This list is not exhaustive: the public API snapshot is authoritative for every exported name, including `NewForcedFailure`, the `DC*` constants (`DCVeryEasy` to `DCNearlyImpossible`), the `Value*` constants and the `KeepRoll` type; further domain types follow d20-tests.md. The kernel types (`modifier.Modifier`, `modifier.RollMode`, `modifier.RollCircumstance`, `modifier.Advantage`, `modifier.Disadvantage`, `ability.Ability`, `dice.Die`, `dice.Expression`) are owned and exported once by the kernel packages, and `d20` MUST NOT re-export or alias them. The package doc comment (`doc.go`) MUST list the use case's inputs and outputs. The names `CheckOutcome`, `AttackOutcome`, `AttackRollOutcome`, `AttackBasis`, `RolledAttackRoll`, `D20TestResult`, `D20TestKind`, `D20TestOutcome`, `Dc`, `RollModifier` and `ModifierSource` MUST NOT be exported. | `doc/rules/architecture.md` §2, §4, §7; modifiers-and-advantage.md §6; decisions FR-002, FR-004, FR-017 | happy |

The telemetry constants are exported on purpose: attribute keys and values are a public contract
(`doc/rules/observability.md` §9). The shape and validation of `DifficultyClass`, `ArmorClass`,
`Roll` and `RollModifiers` are d20-tests.md's; this use case does not re-validate them.

### Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-003 | Withdrawn: moved to d20-tests.md (`RollModifiers` shape and copy; decisions FR-003). Command immutability stays in FR-022. |
| FR-004 | Withdrawn: the command carries no source counts; the combination rule is modifiers-and-advantage.md (FR-010 to FR-017). |
| FR-019 | Withdrawn: moved to d20-tests.md (FR-041; decisions FR-048). |
| FR-033 | Withdrawn: the `FINE` log is dropped (FR-047; decisions FR-043). |
| FR-034 | Withdrawn with FR-033: no log, no lazy attribute closure. |
| FR-054 | Withdrawn: moved to d20-tests.md (FR-034, FR-060; decisions FR-004, FR-048). |

FR-048 and FR-050 were never issued: the gaps are reserved group spacing, not lost requirements. FR-062 is
new in the Go port (typed input errors).

## 5. Model

Domain (pure, synchronous, silent; owned by d20-tests.md and the kernel specs, only used here): value
objects `DifficultyClass`, `ArmorClass`, `RollModifiers`, `Roll`, `CheckResult`, `AttackRollResult`,
`ForcedFailureResult`, `ForcedFailureReason`, `Kind`, `Outcome`, and from the kernel
`modifier.Modifier`, `modifier.RollMode`, `modifier.RollCircumstance`, `ability.Ability`, `dice.Expression`;
and the rules functions `ResolveCheck`, `ResolveAttack`, `FailWithoutRoll` (d20-tests.md FR-069). The use
case composes them and re-derives nothing (FR-057). No entities, aggregates or domain events: a D20 Test is
a one-shot calculation.

Invariants and where they are enforced:

- Kind and target type cannot disagree: closed command variants and typed result variants (FR-001, FR-049).
- One ability modifier, at most one Proficiency Bonus: the shape of `RollModifiers` (decisions FR-003).
- Dice are 1 to 20 and counted per mode: the kernel's `dice.DieRoll`, `dice.Result` and `NewRoll` (dice-and-rolls.md FR-031 to FR-035).
- A check never carries a critical-hit flag (FR-016); `ResolveCheck` rejects kind `AttackRoll` with `ReasonKindMismatch` (d20-tests.md FR-062, FR-063, FR-070; FR-043). No `panic` for it.
- Result collections are copies (d20-tests.md FR-041).

Use-case layer (declared here; signatures are the real ones):

- **Commands (FR-001, FR-055, FR-056).** `Command` interface (unexported methods `kind`, `abilityLabel`,
  `validate`) with the value variants `AbilityCheckCommand{Ability ability.Ability, Target DifficultyClass,
  Attempt CheckAttempt}`, `SavingThrowCommand{Ability, Target DifficultyClass, Attempt SaveAttempt}`,
  `AttackRollCommand{Ability, Target ArmorClass, Attempt RollAttempt}`.
- **Attempts.** `RollAttempt{Modifiers RollModifiers, Circumstances []modifier.RollCircumstance}`
  (circumstances default nil; read, never changed); `ChooseToFail` (`SaveAttempt` only); `AutomaticFailure`
  (`CheckAttempt` and `SaveAttempt`). `CheckAttempt` and `SaveAttempt` are closed interfaces (unexported
  marker methods) implemented by the types above.
- **Results (FR-018, FR-049).** `Result` interface (`Ability()`, `Kind()`, `Outcome()`, unexported `result()`),
  each variant a value object: `RolledCheck` (`Check() CheckResult`), `RolledAttack`
  (`Attack() AttackRollResult`), `ForcedFailure` (`Failure() ForcedFailureResult`, `Target() DifficultyClass`),
  each with `Equal` where it holds a slice.
- `Resolver` orchestrates only: `modifier.Resolve` once (FR-006), one `dice.Roll` request sized by that mode
  (FR-007), builds the `Roll`, calls the rules functions, builds the result and emits telemetry (FR-029
  to FR-032, FR-046). A forced failure makes no request.

| Situation | Mechanism |
|---|---|
| Failed check, failed save, a miss, a forced failure | Result (outcome `Failure`), not an error |
| Roller returns an error | `*srderr.PortError` wrapping it, matching the cause with `errors.Is` (FR-026, FR-036) |
| Roller answers a different expression | `*srderr.PortError` matching `srderr.ErrContractViolation`, from `dice.Roll` (FR-027, FR-036) |
| Roller panics | `*srderr.PortError` matching `srderr.ErrPortPanic`, recovered by `srderr.GuardValue` inside `dice.Roll` (FR-037) |
| Roller returns an out-of-range face | Unrepresentable: `dice.NewDieRoll` and `dice.NewResult` reject it |
| Nil or foreign command or attempt, unknown ability, nil roller, kind mismatch | `*InvalidError`, matching `srderr.ErrInvalidInput` (FR-001, FR-062, FR-043) |

No domain failure is a result variant (commands are built from typed values; an invalid one is an error
before any roll). Port failures are `*srderr.PortError`s returned from `Resolve`; decisions FR-018 and
`MissingContent` do not apply.

## 6. Interface

```go
package d20

// Roller is the dice source Resolver needs: one synchronous method.
type Roller interface {
	Roll(dice.Expression) (dice.Result, error)
}

type Option func(*Resolver)

func WithTelemetry(t telemetry.Telemetry) Option
func WithClock(now func() time.Time) Option

type Resolver struct{ /* unexported */ }

// NewResolver returns *InvalidError (ReasonMissingRoller) when roller is nil.
func NewResolver(roller Roller, opts ...Option) (Resolver, error)

// Resolve returns a *InvalidError for invalid input, a *srderr.PortError for a
// failing, panicking or contract-breaking roller, and otherwise a Result.
func (r Resolver) Resolve(ctx context.Context, cmd Command) (Result, error)
```

- **Sync** (FR-024). **Ports:** `Roller` (declared here, consumer-side) and `telemetry.Telemetry`
  (`Logger` unused, `Tracer`, `Meter`) through `WithTelemetry`.
- **Exact port calls (FR-007).** One `dice.Roll(roller, dice.MustExpression(mode.DiceCount(), dice.D20(), 0))`
  per rolling call, with `mode` from one `modifier.Resolve` (FR-006): a single `1d20` or `2d20`, never
  two `1d20`. A single request is atomic (a failure leaves no half-rolled pair, FR-051) and keeps dice
  order unambiguous. The faces, in order, go to `NewRoll`, which uses `RollMode.Select`.
- **Dice source failure contract as relied on.** The port returns a `dice.Result` per request or an error. A
  roller that cannot roll returns an error, wrapped by `dice.Roll` into a `*srderr.PortError`; one answering a
  different expression is caught by `dice.Roll` as a contract violation; a panicking one is recovered; all
  three are returned unchanged by `Resolve` (FR-026, FR-027, FR-037).
- **Package and layering.** Domain types, rules functions, the use case, the command, attempt and result
  types and the signals live in the single package `github.com/Whealinghold/srd5e-go/d20`, one file per
  concept (`resolver.go`, `command.go`, `attempt.go`, `result.go`, `signals.go`, `rules.go`, ...). It imports
  the kernel packages only (`dice`, `ability`, `modifier`, `srderr`, `telemetry`; decisions FR-013, enforced
  by `internal/archtest`). There is no facade and no re-export; `doc.go` is the package documentation.

## 7. Observability

Evaluation (`doc/rules/observability.md` §2):

1. **What can go wrong?** The dice source fails, panics or answers a different expression; outcome rates
   drift (wrong maths or a biased roller).
2. **How would an operator know?** Port failure: span error status plus `errors`
   (`error.type=dice_roller_error`, `roller_failure` per reason, FR-036, FR-061). A failed check or miss is
   not an error and shows only in the `srd5e.outcome` and `critical_hit` attributes of `resolved`
   (FR-035). A misfiring caller shows as a sustained `automatic` share of `forced_failure_reason`
   (FR-060).
3. **Diagnosis:** kind, ability, roll mode, outcome, the critical-hit flag, the forced-failure
   reason and the roller-failure reason as low-cardinality attributes; the full breakdown is in the
   returned result, not telemetry.
4. **Counted or timed:** throughput and outcomes (`resolved`), latency including failed calls
   (`duration`), sizes (`modifiers`), the d20 face used (`natural`, FR-059), errors (`errors`).
5. **Existing instrumentation:** none; these are the first entries in `doc/observability/signals.md`.
6. **Monitor or alert on:** see the table below; errors split by `roller_failure`, resolved outcome
   mix and `critical_hit` share, the `automatic` forced-failure share, duration, modifiers, the
   natural-face distribution.
7. **Noise:** no logs (FR-047), no child spans, no per-modifier or per-die log events; every signal
   aggregates one record per call.

Constants live in `d20/signals.go` (FR-040); shared keys are `telemetry.AttrOutcome` and
`telemetry.AttrErrorType`. This table is the only catalogue of these signals (decisions FR-040).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.d20_test.resolve` (`SpanResolve`) | span | `srd5e.d20_test.kind` (`ability_check`, `saving_throw`, `attack_roll`), `srd5e.ability` (`strength`, `dexterity`, `constitution`, `intelligence`, `wisdom`, `charisma`), `srd5e.d20_test.roll_mode` (`normal`, `advantage`, `disadvantage`; absent for a forced failure), `srd5e.outcome` (`success`, `failure`), `srd5e.d20_test.critical_hit` (`true`, `false`; attack rolls only), `srd5e.d20_test.forced_failure_reason` (`chosen_to_fail`, `automatic`; forced failures only), `srd5e.d20_test.roller_failure` (`unavailable`, `invalid_result`; error spans only); error status on failure | Trace one D20 Test; the only signal that carries the recorded error, including a recovered roller panic | Inspect error-status spans when `errors` alerts, or when a span has error status without an `errors` increment. Do not alert on span error rate as well as on `errors`; they overlap for port failures |
| `srd5e.d20_test.resolved` (`MetricResolved`) | counter, unit `{test}` | same as the span | Throughput and outcome mix | Over at least 500 attack rolls in `normal`, `critical_hit=true` outside roughly 2 to 8 percent suggests a biased roller (expected 5 percent; widened ranges are applied by callers). Covers direct host calls only: no other use case calls `Resolver` (FR-057), so combat attack rolls are counted under those use cases' own signals. A `forced_failure_reason`=`automatic` share near 100 percent for one ability, sustained, suggests misapplied caller condition wiring rather than gameplay. Alert on a drop to zero throughput against the consumer's baseline |
| `srd5e.d20_test.duration` (`MetricDuration`) | histogram, unit `s` | `srd5e.d20_test.kind` | Latency, including failed calls; dominated by the injected roller | p99 above 5 times the consumer's baseline for a sustained window; expect microseconds with an in-memory roller |
| `srd5e.d20_test.modifiers` (`MetricModifiers`) | histogram, unit `{modifier}` | `srd5e.d20_test.kind` | Size of the `RollModifiers` callers pass (minimum 1) | p99 above 20 sustained suggests a caller accumulating modifiers; dashboard otherwise |
| `srd5e.d20_test.natural` (`MetricNatural`) | histogram, unit `{face}` | `srd5e.d20_test.kind` | The d20 face each rolled call used (1 to 20), so an unfair or stuck injected roller is visible on every face, not only on the natural 20 (FR-059) | Over at least 500 rolled calls in `normal`, any face's share outside roughly 2 to 8 percent (expected 5) points at the roller; a share pinned at one face means a stuck roller. Compare with the `critical_hit` monitor on `resolved`, which sees only the 20 face |
| `srd5e.d20_test.errors` (`MetricErrors`) | counter, unit `{error}` | `srd5e.d20_test.kind`, `error.type` (`dice_roller_error`), `srd5e.d20_test.roller_failure` (`unavailable`, `invalid_result`) | Port failures leaving the use case, counted once, with why the roller failed | Alert on any non-zero rate sustained over 5 minutes. Split the alert by `srd5e.d20_test.roller_failure` (FR-061): `unavailable` is infrastructure (the dice source is down or panicked; page ops), `invalid_result` is the host's adapter answering a different expression (a contract breach; file a bug). The span of the same call holds the error |

Each failure is recorded once, by the use case at whose boundary it leaves srd5e; the dice adapter
and the rules functions emit nothing for it. A caller in another context that wraps this use case
records its own failures under its own signals and must not re-count these.

## 8. Test plan

Levels: unit (`d20/*_test.go`, package `d20_test`), use case (the same package, real rules, doubles
from `dice/dicetest` and `telemetry/telemetrytest`), architecture (`internal/archtest`), scenario
(`scenarios/`). Rules are proven at unit level in d20-tests.md; the use-case level proves orchestration,
port failure and telemetry only. Tests do not yet carry requirement ids (see "Not covered"); the 4.3
case table is a slice of structs in `caseTable()`, one `t.Run` subtest per row. Tests run with `-race
-shuffle=on`; independent subtests call `t.Parallel`.

Doubles: `dicetest.FixedRoller` with its `Requests()` (success rows, dice count and order, FR-007);
`dicetest.CallbackRoller` for every failure and invalid-result case (returns a chosen error, a `dice.Result`
for a different expression, or panics); `telemetrytest.Recorder` with its `Logger` (minimum level
`LevelTrace`, to prove zero records), `Tracer`, `Meter` and `telemetrytest.Clock` (a 250 ms step in the
FR-031 test, so the recorded 0.25 proves the value comes from the injected clock). The `dice.Roller`
contract suite is `dicetest.RunRollerContract`; this spec adds none. `TestResolverRunsOnFakesOfEveryDependency`
and `TestResolverNeedsOnlyARollerFunc` use hand-written fakes of the consumer-side interfaces.

| Ids | Level | Notes |
|---|---|---|
| FR-001, FR-005, FR-056 | unit | Each command variant builds and keeps its fields (type pairings are compiler-enforced, see "Not covered"); nil and foreign commands and attempts are rejected (`TestResolveRejectsInvalidCommands`); mutating the source slice changes nothing already built (`TestRollModifiersAreImmutable`) |
| FR-020, FR-021, FR-049 | unit | A function mapping each `Result` variant to a distinct marker is exercised on every variant (`TestResultTypeSwitchIsExhaustive`); `Equal` on results (`TestResultsAreValues`); one row per variant reads its target (`DifficultyClass`, `ArmorClass`, `DifficultyClass`) |
| FR-008, FR-009, FR-011 to FR-017, FR-043 | unit | Delegated to d20-tests.md (`rules_test.go`: `TestResolveCheck`, `TestResolveCheckRejects` with `ReasonKindMismatch`, `TestResolveAttack`, `FuzzResolveAttack`) plus one use-case smoke row each from the case table; FR-016: no critical-hit accessor on `CheckResult`, `false` on an attack row with a d20 used of 2 to 19 |
| FR-002, FR-007 | use case | Minimal command: one `1d20` request, total = d20 plus the ability modifier. `FixedRoller.Requests()`: `[1d20]` normal, `[2d20]` advantage and disadvantage, none for chosen and automatic failure; one request per rolling call; modifier 0 (`TestResolveCaseTable`) |
| FR-006 | unit + use case | Kernel `modifier.Resolve` tests (modifiers-and-advantage.md FR-010 to FR-017); rows "Advantage uses the higher die", "Disadvantage uses the lower die" and "Advantage and Disadvantage cancel" check the mode reaches the request (`2d20`, `2d20`, single `1d20`) and `Roll.Mode()`. "Exactly once" is review-only |
| FR-010, FR-028 | use case | Rows "total above 30", "total below zero" and the six extreme-roll rows (normal low and high; Advantage and Disadvantage with both 1 and both 20); also unit via d20-tests.md FR-016, FR-019 |
| FR-055 | use case | Command with `Ability` strength and a modifier built for dexterity: total uses the modifier, attribute is `strength` (`TestResultCarriesAbilityLabel`); unknown ability rejected |
| FR-018, FR-022, FR-058 | use case | Every field of one fully populated result per variant (`Ability`, kind, `Roll.Mode()`, dice, natural, modifiers, total, target, outcome, `IsCriticalHit`; forced failure: reason and `Target`); command and source collections compared before and after `Resolve` (`TestResolveLeavesCommandUntouched`); table-driven kind and reason mapping (`ability_check` / `saving_throw` / `attack_roll` with a `RolledAttack`; `ChosenToFail` for `ChooseToFail`; `Automatic` for `AutomaticFailure` on a check and on a save, so a swapped reason fails), wrapped `Ability`, forced-failure `Target`, empty `Requests()` for forced rows |
| FR-023, FR-024, FR-025 | use case | Omitted telemetry works; `NewResolver(nil)` returns `ReasonMissingRoller` (`TestNewResolverRejectsNilRoller`); two calls with the same scripted roller, under `-shuffle=on` (`TestResolveIsDeterministicInAnyOrder`); the zero `Resolver` returns an error (`TestZeroResolverReturnsError`); partial telemetry defaults to no-op (`TestPartialTelemetryDefaultsToNoop`); the default clock is used (`TestDefaultClockIsUsed`) |
| FR-026, FR-027, FR-037, FR-051 | use case | `TestResolveRollerFailures`: an error, a panic, a different count, a different die and the zero result: a `*srderr.PortError` for `dice.Port` matching the cause, `ErrPortPanic` or `ErrContractViolation`; no result; exactly one call recorded; `TestResolveRollerErrorIsNotAResult`, `TestResolveExhaustedScriptIsAPortError` (a spent script is a port error); `TestResolveCancelledContextDoesNotAffectResult` (a cancelled context does not change the result, FR-026) |
| FR-029, FR-030, FR-041, FR-052 | use case | `Recorder`: span name, attributes at start and end, exactly one span per call, one `resolved` increment with the span's attributes for success and failure outcomes (`TestSpanAndResolvedCounter`, `TestSpanStartsWithKindAndAbility`); forced failure: `roll_mode` and `critical_hit` absent, `outcome` `failure` with `forced_failure_reason` (FR-060); attack rows: `critical_hit` present and equal to `IsCriticalHit()`; check, save and forced-failure rows: absent; no `basis` key and no `miss`, `hit` or `critical_hit` outcome value |
| FR-031, FR-032, FR-046 | use case | Clock step 250 ms: one duration record of 0.25, unit `s`, kind attribute (`TestDuration`); repeated with a failing roller: still one record, error returned; the span encloses the measurement; modifier-count histogram 1 (ability only), 2 (with a Proficiency Bonus), several; absent for a forced failure (`TestModifiersAndNaturalHistograms`) |
| FR-035, FR-036, FR-047, FR-061 | use case | A failing check and a miss: span status not error, `errors` unchanged; failing, panicking and wrong-expression rollers: error span with the error, one `errors` increment with `error.type` `dice_roller_error` and `roller_failure`, `resolved` untouched; `Logger` recorder at `LevelTrace`: zero records on every row (`TestNothingIsLogged`) |
| FR-059, FR-060 | use case | The natural histogram records `Roll.Natural()` once per rolled case-table row (unit `{face}`, kind attribute) and nothing for a forced row (`TestNaturalIsTheKeptDie`, `TestForcedFailureRecordsNoRollSignals`); `forced_failure_reason` `chosen_to_fail` / `automatic` on the span and `resolved` for the two forced rows, absent for rolled rows; the new constants pinned by literal |
| FR-038, FR-039 | use case | Every attribute value across all rows is in its closed set; all six abilities map to lower-case names; `critical_hit` is a `bool` (`TestAttributeValuesComeFromClosedSets`); same table with no-op and recording telemetry, comparing results, requests and error sentinels (`TestSilentAndRecordingResolversAgree`) |
| FR-040 | unit | `TestSignalNamesArePinned`, `TestSignalNamesAreValid` (every name satisfies `telemetry.ValidName`) |
| FR-045, FR-053, FR-057 | architecture | `internal/archtest` rules `hidden-effect`, `stdlib-denied`, `package-level-var`, `stutter`, `consumer-name` and the import allow-list; `TestResolverRunsOnFakesOfEveryDependency` (the use case runs on fakes of every dependency, so nothing is hard-wired); "no branching on enabled" review-only |
| FR-062 | unit | `TestInvalidError` (`errors.Is` with `srderr.ErrInvalidInput`, same-reason matching), `TestReasonString`, `TestResolveRejectsInvalidCommands`: no signal emitted for an invalid call |
| FR-042 | scenario | `scenarios/d20_test.go` (package `scenarios_test`) imports the public packages only, references each exported name once and resolves one check, one save, one attack roll and one chosen failure with a fixed roller and recording telemetry; `Example*` functions (`ExampleResolver_Resolve`, `ExampleResolveAttack`, `ExampleFailWithoutRoll`) document the API. This is the first scenario test (roadmap phase 1) and also covers Advantage |

- **Mutation testing:** the use case SHOULD be mutated; the `d20` rules it exercises MUST be (threshold 80,
  `doc/rules/testing.md` §8). Boundary rows live in d20-tests.md and guard against surviving `>=` / `>`
  mutants; the use-case rows only smoke-test wiring. `gomu` is run only inside Docker with `--cpus` and
  `--workers` capped at 1/4 of `nproc`; it is not run on the host.
- **Not covered: requirement traceability.** No test in `d20/` carries an `FR-nnn` id or `Requirement:`/`Path:` comment (doc/rules/testing.md); the test names above are the only mapping. The status stays `partially implemented` until the comments are added.
- **Not covered (compiler-enforced; a test that could not fail at runtime would be a defect,
  `doc/rules/testing.md` §8):** the command and result type pairings (FR-001, FR-049) and the synchronous
  return type (FR-024); `go vet` and `go build` are the check. FR-043 is delegated. FR-044 (no display text in
  result types) and the "exactly once" half of FR-006 are review-only; no architecture rule for FR-044 exists
  yet.

## 9. Open questions and decisions

No open questions. Settled, one line each:

- Ownership split (rules functions in d20-tests.md; use case, commands, result, signals here): decisions FR-002, FR-040, FR-044.
- The command carries `[]modifier.RollCircumstance`, not counts or a resolved mode; `modifier.Resolve` runs once (FR-006).
- The `Ability` label may disagree with the modifier; the caller keeps it consistent (FR-055; d20-tests.md D-06).
- One `2d20` request, not two `1d20` (FR-007).
- Critical Hit is a flag, not an outcome variant (decisions FR-004). Natural 20 and 1 do not apply to ordinary checks and saves (d20-tests.md FR-031).
- No out-of-range roll case: unrepresentable; a wrong answer is a contract violation (FR-027); a roller panic is a recovered port failure, not a propagating defect (FR-037).
- `error.type` values: decisions FR-041 (`srderr.ErrorType`, `dice_roller_error` here). Duration is recorded on failed calls (FR-031). No log (FR-047; decisions FR-043).
- `Modifier` has no kind tag and `RollModifiers` is structured: decisions FR-001, FR-003. No `basis` attribute: decisions FR-041 (FR-052).
- Heroic Inspiration rerolls are not a command option in phase 1: decisions FR-002, FR-010.
- `Resolver` is not reusable by other use cases: decisions FR-044 (FR-057). Combat use cases emit no `srd5e.d20_test.*` signals (attacks.md FR-002 and FR-109 already compose the rules functions).
- The use case declares its own one-method `Roller` rather than importing `dice.Roller`, so it is coupled to no implementation; any `dice.Roller` satisfies it.
- `NewResolver` defaults its clock to `time.Now`; a host that wants no clock access passes `WithClock` with its own. The rules functions read none.
- `roll_mode` is absent for a forced failure (FR-029). social-interaction.md FR-043 and attacks.md FR-118 already omit it.

## 10. Changelog impact

Under `Unreleased` / **Added**:

- `d20.Resolver` use case for ability checks, saving throws and attack rolls (domain types in
  d20-tests.md), returning the closed `d20.Result` (`RolledCheck`, `RolledAttack`, `ForcedFailure`),
  with Advantage and Disadvantage, natural 20 and natural 1 on attack rolls, saves chosen or forced to fail,
  and Critical Hit as a natural 20 (`AttackRollResult.IsCriticalHit`).
- Telemetry signals `srd5e.d20_test.resolve` (span), `srd5e.d20_test.resolved`,
  `srd5e.d20_test.duration`, `srd5e.d20_test.modifiers`, `srd5e.d20_test.natural` and
  `srd5e.d20_test.errors`, with attribute keys `srd5e.d20_test.kind`, `srd5e.ability`,
  `srd5e.d20_test.roll_mode`, `srd5e.outcome`, `srd5e.d20_test.critical_hit`,
  `srd5e.d20_test.roller_failure`, `srd5e.d20_test.forced_failure_reason` and `error.type`, and the
  exported signal constants in `d20`.

Not **BREAKING**: new API with no earlier release. From the first release containing it, renaming any
signal name or attribute key above is **BREAKING** (`doc/rules/observability.md` §5).
