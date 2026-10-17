# D20 Tests

| | |
|---|---|
| Status | implemented |
| Context | `d20` |
| Roadmap phase | 1 |
| Depends on | `errors-and-results.md`, `modifiers-and-advantage.md` |

## 1. Purpose and scope

Models the three kinds of D20 Test (ability checks, saving throws, attack
rolls) as defined by SRD 5.2.1: a d20 is rolled (two with Advantage or
Disadvantage), modifiers are added, the total is compared with a target number
(a Difficulty Class or an Armor Class), and the test succeeds or fails. It
covers the natural 20 and natural 1 rules for attack rolls exactly as the SRD
states them, saving throws a creature chooses to fail, automatic failure, and
rerolling or replacing a d20. The results carry the full breakdown so a
consumer can render its own explanation.

**This spec is domain-only and silent.** It owns the pure model of the
`d20` context: `Roll` (with its reroll and replace-a-die operations),
`RollModifiers`, `DifficultyClass`, `Kind`, `Outcome`, `AttackRollResult`,
`ArmorClass`, `CheckResult`, `ForcedFailureResult`, `ForcedFailureReason`,
`RerollDecision` and the pure rules functions `ResolveCheck`, `ResolveAttack`
and `FailWithoutRoll` (shared-types-and-ownership.md FR-002 to FR-004, FR-044).
The resolution rules are here. The three functions are the published domain
service other use cases compose instead of calling `Resolver.Resolve` (FR-069).
The use case (`Resolver`), the command and attempt types, `Result` and **every
signal** are owned by [resolve-d20-test-use-case.md](resolve-d20-test-use-case.md)
(D-17; shared-types-and-ownership.md FR-017, FR-040, FR-043). They live in the
same package `d20`, in separate files.

The kernel owns `modifier.Modifier`, `modifier.RollMode` (`Normal`,
`WithAdvantage`, `WithDisadvantage`), `modifier.Resolve`, `RollMode.Select` and
`modifier.RollCircumstance` (shared-types-and-ownership.md FR-001;
modifiers-and-advantage.md FR-042). This spec uses them and does not redefine
them. `dice.Roller`, `dice.Expression` and `dice.Result` belong to
`dice-and-rolls.md`; the domain functions never see them because dice values
arrive as plain integers. This context has no upstream context
(`doc/rules/architecture.md` §2) and imports only the kernel packages.

**Out of scope:**

- Deriving an ability modifier from a score (pp. 5-6), the Proficiency Bonus by
  level and Expertise (p. 8): `d20` receives finished `Modifier`s. The
  Proficiency Bonus slot holds the already-multiplied value.
- Which skill uses which ability (p. 9), proficiencies, which ability an attack
  uses (p. 7) and Armor Class calculation (p. 7; Cover, p. 15):
  `derived-stats.md`, `attacks.md`. `d20` owns only `ArmorClass` as a
  target number.
- Damage, extra Critical Hit dice and half damage on a save (p. 16). This spec
  reports a Critical Hit; `combat` acts on it. Automatic Critical Hits from
  conditions are `conditions.md` and `attacks.md` (shared-types-and-ownership.md
  FR-026); this spec reports only the natural 20.
- Death Saving Throws (pp. 17-18): not tied to an ability score, own natural 1
  and 20 effects; `combat` specifies them.
- Sources of Advantage, Disadvantage and automatic failure, and the Exhaustion
  penalty (p. 181): callers supply circumstances, failure kinds or plain
  `Modifier`s in `RollModifiers.Others()` (D-10).
- Heroic Inspiration: slot `character-aggregate.md` (FR-089 to FR-091);
  mechanics and new face `heroic-inspiration.md` (shared-types-and-ownership.md
  FR-048; its FR-034 consumes `ApplyReroll`). This spec takes the reroll
  decision and the new face as input (FR-059).
- The use case, commands, attempt types, `dice.Roller` calls and telemetry:
  resolve-d20-test-use-case.md.
- Whether the GM calls for a D20 Test and the DC set; contested, group and
  passive checks (not in the sections of §2).
- **Not in SRD 5.2.1 as a D20 Test rule, so left out:** automatic success on a
  natural 20 or failure on a natural 1 for ability checks and saving throws; the
  SRD states it for attack rolls only (p. 7).

## 2. Source

PDF page numbers equal printed page numbers. The three steps of a D20 Test are
numbered 4, 5, 6 in the PDF text (list continuation), so this spec cites them by
bold name: "Roll 1d20", "Add Modifiers", "Compare the Total to a Target Number".

- SRD 5.2.1, "The Six Abilities", "Ability Modifiers", pp. 5-6
- SRD 5.2.1, "D20 Tests" (the three steps), p. 6
- SRD 5.2.1, "Ability Checks", "Difficulty Class", Typical Difficulty Classes table, p. 6
- SRD 5.2.1, "Saving Throws", p. 7
- SRD 5.2.1, "Attack Rolls", "Armor Class", "Rolling 20 or 1", p. 7
- SRD 5.2.1, "Advantage/Disadvantage", p. 7; "Roll Two D20s", "They Don't
  Stack", "Interactions with Rerolls", p. 8
- SRD 5.2.1, "Heroic Inspiration" (sidebar, p. 8): "reroll any die immediately
  after rolling it, and you must use the new roll"
- SRD 5.2.1, "Proficiency", "The Bonus Doesn't Stack", p. 8
- SRD 5.2.1, Rules Glossary: "Ability Check", "Advantage", "Armor Class",
  "Attack Roll", "Critical Hit", "D20 Test", "Difficulty Class", "Disadvantage",
  pp. 176-181; "Blinded", p. 177; "Object" ("No Ability Scores"), p. 178;
  "Deafened", p. 181; "Exhaustion", p. 181
- SRD 5.2.1, "Critical Hits", p. 16 (cross-reference only)
- Package design: `doc/rules/architecture.md` §2 to §6,
  `doc/rules/observability.md` §3, `doc/rules/testing.md` §4


## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| D20 Test | concept; `d20.Kind` | "ability checks, attack rolls, and saving throws" (p. 180) |
| Ability Check / Saving Throw / Attack Roll | `Kind` values `AbilityCheck` / `SavingThrow` / `AttackRoll` | Checks and saves against a DC (pp. 6-7); attack rolls against an AC (p. 7) |
| Difficulty Class (DC) | `DifficultyClass` | Target number of a check or save (p. 6) |
| Armor Class (AC) | `ArmorClass` | Target number of an attack roll (p. 7); owned here (shared-types-and-ownership.md FR-002) |
| Advantage / Disadvantage / neither | kernel `modifier.RollMode` | Not redefined here |
| d20 / "the roll" | `Roll` | The d20s rolled, the kept die, the natural value |
| natural 20 / natural 1 | `Roll.Natural()` equal to 20 / 1 | Kept d20 before modifiers (p. 7) |
| modifier | kernel `modifier.Modifier` | Plain signed integer |
| the modifiers added | `RollModifiers` | One ability modifier, at most one Proficiency Bonus, any number of others (shared-types-and-ownership.md FR-003) |
| Critical Hit | `AttackRollResult.IsCriticalHit()` | Natural 20, or a d20 at or above the critical range, on an attack roll; it hits (p. 7, p. 49, p. 179) |
| hit / miss | `AttackRollResult.Hit()` / `Outcome` `Failure` | `Hit()` is `Outcome() == Success` (shared-types-and-ownership.md FR-004) |
| succeeds / fails | `Outcome` values `Success`, `Failure` | Every kind (p. 6) |
| Heroic Inspiration | not modelled here | See §1; the reroll decision is `RerollDecision` |

Package names that are not SRD terms: `Kind`, `ForcedFailureReason`,
`RerollDecision`, `CheckResult`, `ForcedFailureResult`, `ResolveCheck`,
`ResolveAttack`, `FailWithoutRoll`. There is no parallel outcome enum
(`AttackOutcome`, `AttackConclusion`, `AttackRollOutcome`); all kinds use
`Outcome` (shared-types-and-ownership.md FR-004). `attacks.md` may add richer
results under its own names. Terms use SRD 5.2.1 wording exactly
(`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. Withdrawn ids are
listed at the end of this section and have no test here.

Failure vocabulary (errors-and-results.md): invalid input is returned as an
`*d20.InvalidError` carrying a typed `Reason` (`ReasonIndexOutOfRange`,
`ReasonFaceOutOfRange`, `ReasonUnknownKind`, `ReasonKindMismatch`,
`ReasonCriticalRange`, `ReasonForcedFailureNotAllowed`, `ReasonUnknownReason`,
`ReasonUnknownDecision`, `ReasonZeroRoll`; the use case adds more, owned by
resolve-d20-test-use-case.md). It matches `srderr.ErrInvalidInput` with
`errors.Is`, and matches another `*InvalidError` with the same `Reason`
(`errors.Is(err, &d20.InvalidError{Reason: r})`). The kernel's
`*modifier.InvalidError` (`ReasonRollCountMismatch`, `ReasonFaceOutOfRange`,
`ReasonUnknownRollMode`) is returned as it is by `NewRoll`. Tests assert the
reason, never the message. None of these functions panics on any input;
`MustRoll` is the only panicking form and panics with the error
`NewRoll` would return.

### 4.1 Roll mode (owned by the kernel)

`Roll` only calls `RollMode.Select` (FR-011); the mode rules are
modifiers-and-advantage.md.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-009 | `NewRoll(mode, dice)` and `ReplaceDie` MUST reject any die value outside 1 to 20: construction returns the kernel's `*modifier.InvalidError` (`modifier.ReasonFaceOutOfRange`) unchanged, and `ReplaceDie` returns `*InvalidError` with `ReasonFaceOutOfRange` (FR-012, FR-014). `RollMode.Select` stays the sole authority for the natural value and the kept index (FR-011, FR-058). | SRD 5.2.1, "Roll 1d20", p. 6 | unhappy (invalid input) |
| FR-010 | A `Roll` MUST hold exactly `mode.DiceCount()` dice: one for `Normal`, two for `WithAdvantage` and `WithDisadvantage`. Any other count returns `*modifier.InvalidError` with `modifier.ReasonRollCountMismatch` (modifiers-and-advantage.md FR-026). | SRD 5.2.1, "Roll Two D20s", "They Don't Stack", p. 8 | unhappy (invalid input) |
| FR-011 | `Roll.Natural()` MUST be the number `RollMode.Select` returns: the higher die under `WithAdvantage`, the lower under `WithDisadvantage`, the only die under `Normal`. Both dice stay visible in `Roll.Dice()`. Equal dice give that value (modifiers-and-advantage.md FR-021 to FR-025). | SRD 5.2.1, "Roll Two D20s", p. 8 | happy; unhappy (boundary) at equal dice |

### 4.2 The roll: kept die, rerolls and replaced dice

SRD 5.2.1 p. 8 ("Interactions with Rerolls"): when something lets you reroll or
replace the d20 under Advantage or Disadvantage, you can do so for "only one die,
not both", and you choose which. Heroic Inspiration's reroll must use the new
roll (p. 8). Rerolls are operations on the roll, so they live with `Roll`
(shared-types-and-ownership.md FR-002). A decision names one index (FR-059).
`Roll` keeps no hidden "already replaced" state (FR-067), so the SRD limit of
one replaced die per roll is the callers' responsibility, not enforced by this
type. The zero `Roll` holds no dice and is not a roll (`IsZero`); the rules
functions reject it with `ReasonZeroRoll`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-012 | `Roll.ReplaceDie(index, value)` MUST return a new `Roll` with the die at `index` set to `value`, the same mode, and the natural value re-selected with `RollMode.Select`. Example: dice `[18, 3]` under advantage, replace index 0 with 2, gives dice `[2, 3]` and natural 3. | SRD 5.2.1, "Interactions with Rerolls", p. 8 | happy |
| FR-013 | A single `ReplaceDie` call MUST change exactly one die: every other die keeps its value, in its position. | SRD 5.2.1, "Interactions with Rerolls", p. 8 ("only one die, not both") | unhappy (rule interaction) |
| FR-014 | `ReplaceDie` MUST validate `index` and `value` itself: an index outside `0` to `len(Dice()) - 1` (negative, equal to the length, or any index under `Normal` other than 0, or any index on the zero roll) MUST return `*InvalidError` with `ReasonIndexOutOfRange`, and a value outside 1 to 20 `ReasonFaceOutOfRange`. The index is checked first. | `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-015 | `ReplaceDie` and `ApplyReroll` MUST NOT alter the original `Roll`; the original's dice, natural value and kept index are unchanged (value receivers; `Dice()` returns a copy). | `doc/rules/architecture.md` §3 | unhappy (immutability) |
| FR-055 | A `Roll` built by `NewRoll(mode, dice)` from a mode and the face values in the order rolled MUST select its natural value with the kernel `RollMode.Select`. `[18, 3]` under advantage gives natural 18, kept index 0; under disadvantage, natural 3, kept index 1. | SRD 5.2.1, "Roll Two D20s", p. 8; modifiers-and-advantage.md FR-021 to FR-023 | happy |
| FR-058 | `Roll.KeptIndex()` MUST be the index of the kept die: the first index whose value equals the natural value. For equal dice that is index 0. After `ReplaceDie` it is recomputed. | SRD 5.2.1, "Roll Two D20s", p. 8; package design (D-15) | unhappy (boundary: equal dice) |
| FR-059 | `RerollDecision` MUST be a closed interface (unexported marker method; variants `KeepRoll{}` and `RerollDie{Index, Face}`, both listed in its doc comment) so a type switch is exhaustive. `Roll.ApplyReroll(decision)` MUST return the roll unchanged for `KeepRoll`, and for `RerollDie` MUST return `ReplaceDie(Index, Face)`. The new face is used even when lower than the old one. A nil or foreign decision returns `*InvalidError` with `ReasonUnknownDecision`. Because the face is part of `RerollDie` and `KeepRoll` carries none, the earlier design's rejections "a new value supplied with `KeepRoll`" and "no new value with `RerollDie`" are unrepresentable by type. | SRD 5.2.1, "Interactions with Rerolls", "Heroic Inspiration", p. 8 | happy; unhappy (rule interaction: lower new face kept; invalid input) |
| FR-067 | Package design: `Roll` MUST keep no "already replaced" state, so a second `ReplaceDie` on a replaced roll is allowed and returns a valid roll. **The SRD limit "only one die, not both" (p. 8) is enforced by the caller**, not by this type. | package design (D-20); limit: SRD 5.2.1, "Interactions with Rerolls", p. 8 | unhappy (state errors) |
| FR-068 | `Roll` MUST hold a copy of the dice slice it is given (it stores them in a fixed-size array): mutating the source slice afterwards MUST NOT change `Dice()`, `Natural()` or `KeptIndex()`, and mutating the slice `Dice()` returns MUST NOT change the roll. Each result of `ReplaceDie` and `ApplyReroll` has its own dice. | `doc/rules/architecture.md` §3, §4; `doc/rules/testing.md` §4 | unhappy (immutability) |

Reroll cases (table-driven; build the roll, apply the operation, assert dice,
natural and kept index). Sources: SRD 5.2.1 p. 8 for reroll, replace and dice
count rows, p. 6 for the 1 to 20 range (value and die 0/21/negative rows),
`testing.md` §4 for invalid-input rows.

| Case label | Mode, dice | Operation | Expected dice | Expected natural, kept index | Path |
|---|---|---|---|---|---|
| replace the higher die, advantage | advantage, [18, 3] | `ReplaceDie(0, 2)` | [2, 3] | 3, 1 | happy |
| replace the lower die, advantage | advantage, [18, 3] | `ReplaceDie(1, 19)` | [18, 19] | 19, 1 | happy |
| replace the unused die, no change to natural | advantage, [18, 3] | `ReplaceDie(1, 5)` | [18, 5] | 18, 0 | unhappy (rule interaction) |
| replace under disadvantage | disadvantage, [18, 3] | `ReplaceDie(1, 20)` | [18, 20] | 18, 0 | unhappy (rule interaction) |
| replace the single die | normal, [7] | `ReplaceDie(0, 20)` | [20] | 20, 0 | happy |
| replacement to equal dice | advantage, [18, 3] | `ReplaceDie(0, 3)` | [3, 3] | 3, 0 | unhappy (boundary) |
| replacement to the minimum | normal, [20] | `ReplaceDie(0, 1)` | [1] | 1, 0 | unhappy (boundary) |
| replacement to the maximum | normal, [1] | `ReplaceDie(0, 20)` | [20] | 20, 0 | unhappy (boundary) |
| replacement twice (caller enforces the SRD limit) | advantage, [18, 3] | `ReplaceDie(0, 2)` then `ReplaceDie(1, 9)` | [2, 9] | 9, 1 | unhappy (state errors) |
| source list mutated after construction | advantage, list `[18, 3]` | construct, then set source `[0] = 1` | `dice` [18, 3] | 18, 0 | unhappy (immutability) |
| `ApplyReroll` leaves the original | advantage, [18, 3] | `ApplyReroll(RerollDie{Index: 0, Face: 2})` | original [18, 3]; result [2, 3] | original 18, 0; result 3, 1 | unhappy (immutability) |
| index equal to length | advantage, [18, 3] | `ReplaceDie(2, 5)` | `ReasonIndexOutOfRange` | n/a | unhappy (invalid input) |
| negative index | advantage, [18, 3] | `ReplaceDie(-1, 5)` | `ReasonIndexOutOfRange` | n/a | unhappy (invalid input) |
| index 1 under normal | normal, [7] | `ReplaceDie(1, 5)` | `ReasonIndexOutOfRange` | n/a | unhappy (invalid input) |
| value 0 | normal, [7] | `ReplaceDie(0, 0)` | `ReasonFaceOutOfRange` | n/a | unhappy (boundary) |
| value 21 | normal, [7] | `ReplaceDie(0, 21)` | `ReasonFaceOutOfRange` | n/a | unhappy (boundary) |
| reroll lower than before is kept | normal, [15] | `ApplyReroll(RerollDie{Index: 0, Face: 2})` | [2] | 2, 0 | unhappy (rule interaction) |
| keep the roll | advantage, [18, 3] | `ApplyReroll(KeepRoll{})` | [18, 3] | 18, 0 | happy |
| keep with a face supplied | advantage, [18, 3] | `ApplyReroll(KeepRoll{})` | unrepresentable: `KeepRoll` carries no face, so there is nothing to reject | n/a | unhappy (invalid input) |
| reroll to face 0 | advantage, [18, 3] | `ApplyReroll(RerollDie{Index: 0, Face: 0})` | `ReasonFaceOutOfRange` (a face is part of the decision, so "no new value" is unrepresentable) | n/a | unhappy (invalid input) |
| nil decision | advantage, [18, 3] | `ApplyReroll(nil)` | `ReasonUnknownDecision` | n/a | unhappy (invalid input) |
| wrong die count for the mode | advantage, [18] | construct | `modifier.ReasonRollCountMismatch` | n/a | unhappy (invalid input) |
| three dice | advantage, [1, 2, 3] | construct | `modifier.ReasonRollCountMismatch` | n/a | unhappy (invalid input) |
| equal dice | advantage, [9, 9] | construct | [9, 9] | 9, 0 | unhappy (boundary) |
| construct with a die of 0 | advantage, [0, 5] | construct | `modifier.ReasonFaceOutOfRange` | n/a | unhappy (boundary) |
| construct with a die of 21 | advantage, [5, 21] | construct | `modifier.ReasonFaceOutOfRange` | n/a | unhappy (boundary) |
| construct with a negative die | normal, [-3] | construct | `modifier.ReasonFaceOutOfRange` | n/a | unhappy (invalid input) |
| normal with two dice | normal, [7, 8] | construct | `modifier.ReasonRollCountMismatch` | n/a | unhappy (invalid input) |
| no dice under advantage | advantage, [] | construct | `modifier.ReasonRollCountMismatch` | n/a | unhappy (empty and minimal) |
| normal with one die | normal, [7] | construct | [7] | 7, 0 | happy |

### 4.3 Modifiers and the total

`RollModifiers` (shared-types-and-ownership.md FR-003) is built with
`NewRollModifiers(ability)` and the copy-returning builders
`WithProficiencyBonus(b)` and `WithOthers(ms...)`; it enforces by shape every
"at most one of X" rule, so no runtime check exists. The slots hold kernel
`modifier.Modifier`s (plain integers), not `ability`'s derived modifier or
`character`'s `ProficiencyBonus` (D-16); a caller converts with
`modifier.New(abilityModifier.Value())`. In the tables a modifier set is written
`A+3 P+2 O[+1, -2]`: `A` the ability modifier (always present; `A+0` is "none"),
`P` the Proficiency Bonus (omitted when absent), `O` the other modifiers
(omitted when empty).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-016 | The total of a rolled D20 Test MUST equal `Roll.Natural()` plus the kernel sum of the ability modifier, the Proficiency Bonus (when present) and every other modifier (`RollModifiers.Total()`), with no clamping: a total MAY be below 1 or above 20. | SRD 5.2.1, "Add Modifiers", p. 6; modifiers-and-advantage.md FR-004, FR-008 | happy |
| FR-017 | `NewRollModifiers` MUST require exactly one ability modifier (its only parameter) and `WithOthers` MUST accept any number of other modifiers, including none. With only the ability modifier, the total is the natural value plus that modifier. The zero `RollModifiers` is an ability modifier of +0 and nothing else. | SRD 5.2.1, "Add Modifiers", p. 6; modifiers-and-advantage.md FR-005 | unhappy (empty and minimal) |
| FR-018 | `RollModifiers` MUST hold at most one Proficiency Bonus (a single slot with a presence flag, `ProficiencyBonus() (Modifier, bool)`), so it cannot be added twice: a second `WithProficiencyBonus` replaces the first. It MUST NOT apply any other stacking rule. | SRD 5.2.1, "The Bonus Doesn't Stack", p. 8 | unhappy (rule interaction) |
| FR-019 | Modifiers MAY be positive, zero or negative, in any mix, and a total MAY be zero or negative. | SRD 5.2.1, "Add Modifiers" (Circumstantial Bonuses and Penalties), p. 6 | happy; unhappy (empty and minimal) |
| FR-020 | The total MUST NOT depend on the order in which the other modifiers are listed. | SRD 5.2.1, "Add Modifiers", p. 6; modifiers-and-advantage.md FR-007 | unhappy (determinism) |
| FR-061 | `RollModifiers` MUST hold a copy of `others`: mutating the caller's slice afterwards MUST NOT change the value, and mutating the slice `Others()` returns MUST NOT change it. `Others()` is empty by default. | `doc/rules/architecture.md` §3, §4 | unhappy (immutability; empty and minimal) |

Total cases (source SRD 5.2.1 p. 6):

| Case label | Natural | Modifiers | Expected total | Path |
|---|---|---|---|---|
| ability only, zero | 10 | `A+0` | 10 | unhappy (empty and minimal) |
| zero other modifier | 10 | `A+0 O[0]` | 10 | unhappy (empty and minimal) |
| single bonus | 10 | `A+3` | 13 | happy |
| ability and Proficiency Bonus | 10 | `A+3 P+2` | 15 | happy |
| bonuses and penalties | 10 | `A+3 P+2 O[+1, -2]` | 14 | happy |
| negative modifier | 5 | `A-5` | 0 | unhappy (boundary) |
| total below zero | 1 | `A-5 O[-2]` | -6 | unhappy (boundary) |
| total above 20 | 20 | `A+10` | 30 | unhappy (boundary) |
| other modifiers reordered, first order | 10 | `A+3 O[-2, +2]` | 13 | unhappy (determinism) |
| other modifiers reordered, second order | 10 | `A+3 O[+2, -2]` | 13 | unhappy (determinism) |
| very large bonus | 1 | `A+100` | 101 | unhappy (boundary) |
| very large penalty | 20 | `A-100` | -80 | unhappy (boundary) |

### 4.4 Target numbers

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-021 | `DifficultyClass` MUST be a distinct integer type, with named constants for the Typical Difficulty Classes: `DCVeryEasy` 5, `DCEasy` 10, `DCMedium` 15, `DCHard` 20, `DCVeryHard` 25, `DCNearlyImpossible` 30. | SRD 5.2.1, Typical Difficulty Classes table, p. 6 | happy |
| FR-022 | `DifficultyClass` and `ArmorClass` MUST compare equal by value within their own type, and MUST NOT compare equal to each other for the same number: they are distinct types, so comparing them is a compile error and the target parameters cannot be swapped. | `doc/rules/architecture.md` §3; the SRD names the two target numbers, pp. 6-7 | unhappy (boundary) |

The SRD gives no minimum or maximum for a DC or an AC, so neither type rejects
any integer (D-02). Cases: `DCVeryEasy == DifficultyClass(5)`, `DCEasy == 10`,
`DCMedium == 15`, `DCHard == 20`, `DCVeryHard == 25`, `DCNearlyImpossible == 30`,
`DifficultyClass(10) == DifficultyClass(10)`, `ArmorClass(10) == ArmorClass(10)`,
`DifficultyClass(10) != DifficultyClass(11)`; a DC and an AC of 10 are not
comparable (compile-time).

### 4.5 Success and failure resolution

The entry points are the package-level pure functions (FR-069):
`ResolveCheck(kind, roll, modifiers, target) (CheckResult, error)`,
`ResolveAttack(roll, modifiers, target, criticalRange) (AttackRollResult, error)`
and `FailWithoutRoll(kind, reason) (ForcedFailureResult, error)`. There are no
static delegates on the result types (D-18, FR-070). The functions accept any
`Roll`, including a rerolled or replaced one.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-023 | An ability check whose total equals or exceeds the DC MUST resolve to `Success`. | SRD 5.2.1, "Compare the Total to a Target Number", p. 6 | happy; unhappy (boundary) at equal |
| FR-024 | An ability check whose total is below the DC MUST resolve to `Failure`. | SRD 5.2.1, "Compare the Total to a Target Number", p. 6 | happy; unhappy (boundary) at one below |
| FR-025 | A saving throw whose total equals or exceeds the DC MUST resolve to `Success`. | SRD 5.2.1, "Compare the Total to a Target Number", p. 6; "Saving Throws", p. 7 | happy; unhappy (boundary) at equal |
| FR-026 | A saving throw whose total is below the DC MUST resolve to `Failure`. | SRD 5.2.1, "Compare the Total to a Target Number", p. 6; "Saving Throws", p. 7 | happy; unhappy (boundary) at one below |
| FR-027 | An attack roll whose natural value is neither 1 nor 20 and whose total equals or exceeds the AC (and is below the critical range, FR-071) MUST resolve to `Success` (`Hit()` true) with `IsCriticalHit()` false. | SRD 5.2.1, "Attack Rolls", p. 7 | happy; unhappy (boundary) at equal |
| FR-028 | An attack roll whose natural value is neither 1 nor 20 and whose total is below the AC MUST resolve to `Failure` (a miss) with `IsCriticalHit()` false. | SRD 5.2.1, "Attack Rolls", p. 7 | happy; unhappy (boundary) at one below |
| FR-029 | An attack roll with a natural 20 MUST resolve to `Success` with `IsCriticalHit()` true, regardless of modifiers or the AC and of any critical range (FR-071). | SRD 5.2.1, "Rolling 20 or 1", p. 7; "Critical Hit", p. 179 | unhappy (boundary) |
| FR-030 | An attack roll with a natural 1 MUST resolve to `Failure` with `IsCriticalHit()` false, regardless of modifiers or the AC. | SRD 5.2.1, "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| FR-031 | The natural 20 and natural 1 rules MUST apply to attack rolls only. A check or save MUST resolve by comparing its total with the DC alone, whatever the natural value, and a `CheckResult` has no critical-hit method. | SRD 5.2.1, "Rolling 20 or 1", p. 7 ("for an attack roll") | unhappy (rule interaction) |
| FR-032 | Under `WithAdvantage` or `WithDisadvantage`, the natural 20 and natural 1 rules MUST be judged on `Roll.Natural()` (the kept die), not the discarded one. | SRD 5.2.1, "Roll Two D20s", p. 8; "Rolling 20 or 1", p. 7 | unhappy (rule interaction) |
| FR-033 | With the default critical range (20), a natural 19 MUST NOT be a Critical Hit: it resolves by its total. | SRD 5.2.1, "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| FR-034 | `AttackRollResult` MUST carry its `Roll`, `RollModifiers`, total, `ArmorClass` target, `Outcome`, `IsCriticalHit()` and critical range, and expose `Hit()` as `Outcome() == Success`. There is no basis enum and no `criticalHit` outcome. | SRD 5.2.1, "Attack Rolls", p. 7; `doc/rules/architecture.md` §6 | happy |
| FR-035 | A saving throw the creature chooses to fail MUST resolve to `Failure` with no `Roll`, and the result MUST say so (`ChosenToFail`). `FailWithoutRoll(kind, reason)` produces it. | SRD 5.2.1, "Saving Throws", p. 7 ("you can choose to fail the save without rolling") | happy |
| FR-036 | A check or save declared an automatic failure by the caller MUST resolve to `Failure` with no `Roll`, reason `Automatic`. Sources the caller may rely on: a check that requires sight while Blinded (p. 177) or hearing while Deafened (p. 181), and, for saving throws only, an object with no ability scores ("fails all saving throws", p. 178; it "can't make ability checks", so objects are not a source for checks). | SRD 5.2.1, "Blinded", p. 177; "Deafened", p. 181; "Object", p. 178 | happy |
| FR-063 | `ResolveCheck(kind, roll, modifiers, target)` MUST resolve any `Roll` (fresh, rerolled or replaced) exactly as FR-016 and FR-023 to FR-026 say, so downstream domain code gets an outcome from a rerolled roll without the use case. It rejects `AttackRoll` (FR-062). | SRD 5.2.1, "Interactions with Rerolls", "Heroic Inspiration", p. 8; D-18 | happy; unhappy (rule interaction) |
| FR-064 | `ResolveAttack(roll, modifiers, target, criticalRange)` MUST resolve any `Roll`, including a rerolled or replaced one, by FR-027 to FR-033, judging the natural 20 and natural 1 on the roll's current natural value. | SRD 5.2.1, "Rolling 20 or 1", p. 7; "Interactions with Rerolls", p. 8; D-18 | happy; unhappy (rule interaction) |
| FR-065 | `CheckResult.AdjustedBy(delta int) (CheckResult, error)` and `AttackRollResult.AdjustedBy(delta int) (AttackRollResult, error)` MUST return a new result for the same roll and target with `modifier.New(delta)` appended to the other modifiers, the total re-derived (FR-016) and the outcome re-resolved by the rules of `ResolveCheck` / `ResolveAttack` (with the result's own critical range). The original is unchanged. A delta of 0 is allowed. `ForcedFailureResult` has no `AdjustedBy`. Applied to a zero result they return the rules function's error (`ReasonZeroRoll`). Effects that add or subtract after the roll (Bardic Inspiration, Cutting Words) call it. | SRD 5.2.1, "Add Modifiers", p. 6; D-18 | happy; unhappy (boundary; immutability) |
| FR-066 | After `AdjustedBy`, an attack roll MUST still be judged on its natural value: a natural 20 stays a hit and a Critical Hit and a natural 1 stays a miss at any total; a natural 20 or 1 on a check or save stays irrelevant (FR-031). | SRD 5.2.1, "Rolling 20 or 1", p. 7 | unhappy (boundary; rule interaction) |

Check and save resolution (identical rows for each kind; one table-driven test
per row per kind). Source SRD 5.2.1 p. 6; rows with "natural 20" or "natural 1"
in the label also p. 7 (no natural rule for checks and saves).

| Case label | Natural | Modifiers | Total | DC | Expected | Path |
|---|---|---|---|---|---|---|
| total equals the DC | 10 | `A+3 P+2` | 15 | 15 | success | unhappy (boundary) |
| total one below the DC | 10 | `A+4` | 14 | 15 | failure | unhappy (boundary) |
| total one above the DC | 10 | `A+6` | 16 | 15 | success | unhappy (boundary) |
| well above | 15 | `A+5 P+3` | 23 | 10 | success | happy |
| well below | 3 | `A+0` | 3 | 20 | failure | happy |
| ability only meets the DC | 15 | `A+0` | 15 | 15 | success | unhappy (empty and minimal) |
| natural 20 below the DC fails | 20 | `A-6` | 14 | 15 | failure | unhappy (rule interaction) |
| natural 20 short of a high DC | 20 | `A+0` | 20 | 30 | failure | unhappy (rule interaction) |
| natural 1 above the DC succeeds | 1 | `A+10 O[+4]` | 15 | 15 | success | unhappy (rule interaction) |
| natural 20 reaches DC 30 | 20 | `A+10` | 30 | 30 | success | unhappy (boundary) |
| negative total | 1 | `A-5 O[-2]` | -6 | 5 | failure | unhappy (boundary) |
| DC 5, very easy, low roll | 2 | `A+3` | 5 | 5 | success | unhappy (boundary) |
| very large bonus | 1 | `A+100` | 101 | 30 | success | unhappy (boundary) |
| very large penalty | 20 | `A-100` | -80 | 5 | failure | unhappy (boundary) |


Attack roll resolution (SRD 5.2.1 p. 7; rows labelled Advantage or Disadvantage
also p. 8 for the kept die; the dice column shows both dice when the mode is not `Normal`):

| Case label | Mode, dice | Natural | Modifiers | Total | AC | Expected outcome | `IsCriticalHit()` | Path |
|---|---|---|---|---|---|---|---|---|
| total equals the AC | normal, 10 | 10 | `A+3 P+2` | 15 | 15 | success | false | unhappy (boundary) |
| total one below the AC | normal, 10 | 10 | `A+4` | 14 | 15 | failure | false | unhappy (boundary) |
| total one above the AC | normal, 10 | 10 | `A+6` | 16 | 15 | success | false | unhappy (boundary) |
| natural 20 beats any AC | normal, 20 | 20 | `A+0` | 20 | 15 | success | true | unhappy (boundary) |
| natural 20, total far below AC | normal, 20 | 20 | `A-5` | 15 | 30 | success | true | unhappy (boundary) |
| natural 20, total equals AC | normal, 20 | 20 | `A+5` | 25 | 25 | success | true | unhappy (boundary) |
| natural 20, total above AC | normal, 20 | 20 | `A+10` | 30 | 15 | success | true | unhappy (boundary) |
| natural 1 misses a low AC | normal, 1 | 1 | `A+12` | 13 | 10 | failure | false | unhappy (boundary) |
| natural 1, total far above AC | normal, 1 | 1 | `A+20` | 21 | 5 | failure | false | unhappy (boundary) |
| natural 1, total also below AC | normal, 1 | 1 | `A+2` | 3 | 15 | failure | false | unhappy (boundary) |
| natural 19 one below AC | normal, 19 | 19 | `A+0` | 19 | 20 | failure | false | unhappy (boundary) |
| natural 19 above AC is not critical | normal, 19 | 19 | `A+10` | 29 | 20 | success | false | unhappy (boundary) |
| range 19: natural 19 crits below AC | normal, 19; `criticalRange` 19 | 19 | `A-5` | 14 | 30 | success | true | unhappy (boundary) |
| range 19: natural 18 does not | normal, 18; `criticalRange` 19 | 18 | `A+0` | 18 | 30 | failure | false | unhappy (boundary) |
| range 18: natural 18 crits below AC | normal, 18; `criticalRange` 18 | 18 | `A+0` | 18 | 30 | success | true | unhappy (boundary) |
| range 18: natural 17 does not | normal, 17; `criticalRange` 18 | 17 | `A+0` | 17 | 30 | failure | false | unhappy (boundary) |
| range 18: natural 20 and natural 1 | normal, 20 / 1; `criticalRange` 18 | 20 / 1 | `A+0` | 20 / 1 | 30 | success / failure | true / false | unhappy (boundary) |
| range 17 or 21 | `criticalRange` 17, 21 | any | any | any | any | `ReasonCriticalRange` | none | unhappy (invalid input) |
| natural 2 above AC | normal, 2 | 2 | `A+10 O[+4]` | 16 | 15 | success | false | happy |
| Advantage dice 20 and 1 | advantage, 20 and 1 | 20 | `A+0` | 20 | 15 | success | true | unhappy (rule interaction) |
| Disadvantage dice 20 and 1 | disadvantage, 20 and 1 | 1 | `A+20` | 21 | 5 | failure | false | unhappy (rule interaction) |
| Advantage dice 1 and 1 | advantage, 1 and 1 | 1 | `A+20` | 21 | 5 | failure | false | unhappy (boundary) |
| Disadvantage dice 20 and 20 | disadvantage, 20 and 20 | 20 | `A-3` | 17 | 30 | success | true | unhappy (boundary) |
| Advantage dice 12 and 5 | advantage, 12 and 5 | 12 | `A+3` | 15 | 15 | success | false | happy |
| Advantage, natural 1 discarded | advantage, 1 and 12 | 12 | `A+3` | 15 | 15 | success | false | unhappy (rule interaction) |
| very large bonus | normal, 10 | 10 | `A+100` | 110 | 30 | success | false | unhappy (boundary) |
| natural 20, very large penalty | normal, 20 | 20 | `A-100` | -80 | 10 | success | true | unhappy (boundary) |
| natural 1, very large bonus | normal, 1 | 1 | `A+100` | 101 | 10 | failure | false | unhappy (boundary) |

Re-resolution after a reroll or adjustment (FR-063 to FR-066). Each row builds
the original result, then applies the operation. Source SRD 5.2.1 p. 6 (check
rows), p. 7 (attack rows; "natural 20 check adjusted below the DC" p. 6 and
p. 7), p. 8 (rows using `ReplaceDie`).

| Case label | Original | Operation | Expected | Path |
|---|---|---|---|---|
| attack rerolled from a miss to a hit | attack, advantage, [4, 3], `A+3`, AC 15 (total 7, failure) | `ResolveAttack` with the `ReplaceDie(0, 14)` roll | natural 14, total 17, success, not critical | happy |
| attack replaced to a natural 20 | attack, normal, [5], `A+0`, AC 30 | `ResolveCheck` with `ReplaceDie(0, 20)` | success, `IsCriticalHit()` true | unhappy (rule interaction) |
| attack replaced to a natural 1 | attack, normal, [18], `A+10`, AC 10 | `ResolveCheck` with `ReplaceDie(0, 1)` | failure, `IsCriticalHit()` false | unhappy (rule interaction) |
| check rerolled | check, normal, [3], `A+2`, DC 10 | `ResolveCheck` with `ReplaceDie(0, 19)` | total 21, success | happy |
| adjusted check up to the DC | check, normal, [8], `A+1`, DC 15 (total 9) | `AdjustedBy(+6)` | total 15, success; `others` gains `+6` | unhappy (boundary) |
| adjusted check down one below | check, normal, [10], `A+5`, DC 15 (total 15) | `AdjustedBy(-1)` | total 14, failure | unhappy (boundary) |
| adjusted by zero | check, normal, [10], `A+5`, DC 15 | `AdjustedBy(0)` | equal to a result with `others` `[0]`; total 15, success | unhappy (empty and minimal) |
| natural 20 attack adjusted far down | attack, normal, [20], `A+0`, AC 15 | `AdjustedBy(-100)` | total -80, success, `IsCriticalHit()` true | unhappy (boundary) |
| natural 1 attack adjusted far up | attack, normal, [1], `A+0`, AC 5 | `AdjustedBy(+100)` | total 101, failure, `IsCriticalHit()` false | unhappy (boundary) |
| natural 19 attack adjusted onto the AC | attack, normal, [19], `A+0`, AC 25 | `AdjustedBy(+6)` | total 25, success, not critical | unhappy (boundary) |
| natural 20 check adjusted below the DC | check, normal, [20], `A+0`, DC 15 | `AdjustedBy(-6)` | total 14, failure (no natural 20 rule) | unhappy (rule interaction) |
| adjustment leaves the original | any row above | `AdjustedBy(n)` | original result unchanged | unhappy (immutability) |
| rerolled roll for a save | save, advantage, [18, 3], `A+0`, DC 15 | `ResolveCheck` with `ReplaceDie(0, 2)` | natural 3, total 3, failure | unhappy (rule interaction) |
| `ResolveCheck` with AttackRoll | kind `AttackRoll`, normal, [10], `A+0`, DC 15 | `ResolveCheck` | `ReasonKindMismatch` | unhappy (invalid input) |

Forced failure (FR-035, FR-036, FR-052):

| Case label | Kind | Reason | Expected | Source | Path |
|---|---|---|---|---|---|
| chosen to fail a save | SavingThrow | ChosenToFail | failure, reason `ChosenToFail`, no roll | p. 7 | happy |
| automatic failure of a save | SavingThrow | Automatic | failure, reason `Automatic`, no roll | p. 178 | happy |
| automatic failure of a check | AbilityCheck | Automatic | failure, reason `Automatic`, no roll | p. 177, p. 181 | happy |
| chosen to fail a check | AbilityCheck | ChosenToFail | `ReasonForcedFailureNotAllowed` | p. 6 (no such rule) | unhappy (invalid input) |
| forced failure of an attack roll | AttackRoll | Automatic | `ReasonForcedFailureNotAllowed` | p. 7 (no such rule) | unhappy (invalid input) |
| forced failure of an attack roll, chosen | AttackRoll | ChosenToFail | `ReasonForcedFailureNotAllowed` | p. 7 (no such rule) | unhappy (invalid input) |

### 4.6 Result shape, value semantics and guards

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-041 | A rolled `CheckResult` MUST carry its kind, `Roll`, `RollModifiers`, total, `DifficultyClass` target and `Outcome`; an `AttackRollResult` carries what FR-034 lists; a `ForcedFailureResult` MUST carry its kind and `ForcedFailureReason` and no roll. Slices reachable from a result (`Dice()`, `Others()`) MUST be copies. A result carries no `Ability` and no separate roll-mode field: the mode is `Roll.Mode()` (D-19). | `doc/rules/architecture.md` §6, §4 | happy; unhappy (immutability) |
| FR-050 | `DifficultyClass`, `ArmorClass`, `Roll` and `ForcedFailureResult` MUST be comparable and equal (`==`) when every field is equal; `RollModifiers`, `CheckResult` and `AttackRollResult` hold slices and MUST provide an `Equal` method that is true when every field is equal (including the other modifiers' contents and order); the `RerollDecision` variants are comparable structs. Any single differing field MUST give inequality. | `doc/rules/architecture.md` §3 | happy; unhappy (any differing field) |
| FR-051 | For an attack roll with a natural 20, `IsCriticalHit()` MUST be true, and with a natural 1 it MUST be false and the outcome `Failure`, even when the total alone would give the other outcome (nat 20 with total below AC; nat 1 with total at or above AC). | SRD 5.2.1, "Rolling 20 or 1", p. 7; `doc/rules/architecture.md` §6 | unhappy (rule interaction) |
| FR-052 | `FailWithoutRoll(kind, reason)` MUST reject with `ReasonForcedFailureNotAllowed` an attack roll (any reason) and an ability check with reason `ChosenToFail`; an invalid kind or reason is `ReasonUnknownKind` / `ReasonUnknownReason`. | SRD 5.2.1, "Saving Throws", p. 7; `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-060 | `Outcome` MUST be exactly `Success` and `Failure` for every kind (a typed enum with `String()` returning `success` / `failure`), and `IsCriticalHit()` MUST exist only on `AttackRollResult`. No `criticalHit` outcome and no `AttackOutcome`, `AttackConclusion` or `AttackRollOutcome` type is exported. | shared-types-and-ownership.md FR-004; SRD 5.2.1, "Rolling 20 or 1", p. 7 | happy |
| FR-062 | `ResolveCheck` MUST reject a kind of `AttackRoll` with `ReasonKindMismatch` (an invalid kind with `ReasonUnknownKind`, the zero roll with `ReasonZeroRoll`). A DC cannot be paired with an attack roll, nor an AC with a check or save, because the target parameter types differ. | SRD 5.2.1, "Ability Checks", p. 6; "Attack Rolls", p. 7; `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-069 | The pure rules MUST be the three package-level functions `ResolveCheck` (FR-063, FR-062), `ResolveAttack` (FR-064) and `FailWithoutRoll` (FR-035, FR-036, FR-052), stateless, in package `d20`, with no port, no telemetry and no dice. Other use cases compose them and record their own signals from the results (shared-types-and-ownership.md FR-044). `Resolver` (resolve-d20-test-use-case.md) calls the same functions and nothing else resolves a D20 Test. | shared-types-and-ownership.md FR-002, FR-044; `doc/rules/architecture.md` §4 | happy; unhappy (invalid input: error rows) |
| FR-070 | There MUST be no second entry point: the result types expose no static constructors and `AdjustedBy` (FR-065) stays a method of the results, delegating to the same two functions, so equal inputs give equal results and the same error. | package design (D-18, D-21) | happy; unhappy (rule interaction) |
| FR-071 | `ResolveAttack` MUST take a `criticalRange int` (`DefaultCriticalRange` = 20). A roll whose natural value is at or above it MUST resolve to `Success` with `IsCriticalHit()` true regardless of modifiers or AC (FR-029); below it, FR-027, FR-028 and FR-030 apply. A natural 20 is always a Critical Hit and a natural 1 never is, whatever the range. | SRD 5.2.1, "Critical Hit", p. 179; Fighter "Level 3: Improved Critical", "Level 15: Superior Critical", p. 49 | happy; unhappy (boundary at 19 and 18) |
| FR-072 | `criticalRange` MUST be 18, 19 or 20; any other value MUST return `*InvalidError` with `ReasonCriticalRange`. Interpretation: the lowest range the SRD grants is 18-20 (p. 49), so a wider one is a caller bug. Checks and saves take no range (FR-031). | SRD 5.2.1, p. 49; `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-073 | `AttackRollResult` MUST also carry its critical range (`CriticalRange()`, default 20), included in `Equal` (FR-050); `AdjustedBy` MUST re-resolve with it, so a natural at or above the range stays a Critical Hit at any total (FR-066). | SRD 5.2.1, p. 179; package design | happy; unhappy (rule interaction) |

### 4.7 Withdrawn ids

Kept so references stay valid; no test here.

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to modifiers-and-advantage.md (FR-010, FR-018) |
| FR-002 | Withdrawn: moved to modifiers-and-advantage.md (FR-011, FR-019) |
| FR-003 | Withdrawn: moved to modifiers-and-advantage.md (FR-012, FR-020) |
| FR-004 | Withdrawn: moved to modifiers-and-advantage.md (FR-013) |
| FR-005 | Withdrawn: moved to modifiers-and-advantage.md (FR-014) |
| FR-006 | Withdrawn: moved to modifiers-and-advantage.md (FR-015, FR-016) |
| FR-007 | Withdrawn: moved to modifiers-and-advantage.md (FR-010) |
| FR-008 | Withdrawn: moved to modifiers-and-advantage.md (FR-010 to FR-017) |
| FR-037 | Withdrawn: moved to resolve-d20-test-use-case.md (dice requested) |
| FR-038 | Withdrawn: moved to resolve-d20-test-use-case.md (port failure) |
| FR-039 | Withdrawn: moved to resolve-d20-test-use-case.md (invalid roller result) |
| FR-040 | Withdrawn: moved to resolve-d20-test-use-case.md (determinism) |
| FR-042 | Withdrawn: moved to resolve-d20-test-use-case.md (command not altered) |
| FR-043 | Withdrawn: moved to resolve-d20-test-use-case.md (counter) |
| FR-044 | Withdrawn: moved to resolve-d20-test-use-case.md (span) |
| FR-045 | Withdrawn: moved to resolve-d20-test-use-case.md (duration histogram) |
| FR-046 | Withdrawn: moved to resolve-d20-test-use-case.md (modifiers histogram) |
| FR-047 | Withdrawn: moved to resolve-d20-test-use-case.md (error recording) |
| FR-048 | Withdrawn: moved to resolve-d20-test-use-case.md (no-op and recording identity) |
| FR-049 | Withdrawn: moved to resolve-d20-test-use-case.md (constants) |
| FR-053 | Withdrawn: moved to resolve-d20-test-use-case.md (forced failure signal) |
| FR-054 | Withdrawn: moved to resolve-d20-test-use-case.md (attribute values) |
| FR-056 | Withdrawn: moved to resolve-d20-test-use-case.md (roller `Error` propagates) |
| FR-057 | Withdrawn: moved to resolve-d20-test-use-case.md (command defaults and copies; live domain equivalents FR-061, FR-068) |


## 5. Model

Layout (`doc/rules/architecture.md` §4): the domain files of package `d20`
hold everything in this spec (`roll.go`, `roll_modifiers.go`, `kind.go`,
`rules.go`, `check_result.go`, `attack_roll_result.go`,
`forced_failure_result.go`, `error.go`); they are pure, synchronous, immutable
(unexported fields, value receivers) and import only the kernel packages. The use
case files (`resolver.go`, `command.go`, `attempt.go`, `result.go`,
`signals.go`) are owned by resolve-d20-test-use-case.md.

- `Roll`: mode, dice (copied; a fixed-size array internally), natural value,
  kept index (`Mode()`, `Dice()`, `Natural()`, `KeptIndex()`, FR-058);
  `NewRoll(mode, dice)`, `MustRoll(mode, dice...)`, `ReplaceDie` (FR-012)
  and `ApplyReroll` (FR-059) return new rolls. The constructor copies the slice
  (FR-068) and calls `RollMode.Select` for the kept die (FR-009 to FR-011).
  Comparable; the zero value is invalid (`IsZero`). No replaced state (FR-067).
- `RerollDecision`: closed interface, `KeepRoll{}` | `RerollDie{Index, Face}`.
- `RollModifiers`: `Ability()` (required), `ProficiencyBonus() (Modifier, bool)`,
  `Others()` (a copy), `Count()`; `Total()` is the kernel sum (FR-016 to FR-020,
  FR-061). Built with `NewRollModifiers(ability).WithProficiencyBonus(b).WithOthers(ms...)`.
- `DifficultyClass` (with constants) and `ArmorClass`: integer types with value
  equality, no range check (FR-021, FR-022, D-02).
- `Kind`: `AbilityCheck` | `SavingThrow` | `AttackRoll`. `Outcome`: `Success` |
  `Failure`. `ForcedFailureReason`: `ChosenToFail` | `Automatic`. Each is a typed
  enum with `String()`; the zero value is not valid (`Kind.Valid()`).
- Results: `CheckResult` (`Kind`, `Roll`, `Modifiers`, `Total`, `Target`
  `DifficultyClass`, `Outcome`); `AttackRollResult` (`Roll`, `Modifiers`, `Total`,
  `Target` `ArmorClass`, `Outcome`, `IsCriticalHit`, `CriticalRange`, `Hit`);
  `ForcedFailureResult` (`Kind`, `Reason`; always `Failure`, no roll).
  `AdjustedBy` on the first two (FR-065, FR-066).
- Rules functions (stateless, FR-069): `ResolveCheck(Kind, Roll, RollModifiers,
  DifficultyClass)` (no natural 20/1 branch: FR-016, FR-023 to FR-026, FR-031,
  FR-062); `ResolveAttack(Roll, RollModifiers, ArmorClass, criticalRange int)`
  (the only natural 20/1 branch: FR-027 to FR-034, FR-051, FR-071 to FR-073);
  `FailWithoutRoll(Kind, ForcedFailureReason)` (FR-035, FR-036, FR-052).

No domain events: a D20 Test is a calculation; `combat` raises events from the
result.

**Failures.** Success, failure, hit, miss and Critical Hit are results, never
errors. No `*srderr.PortError` is returned by the domain. Every rejection
follows errors-and-results.md FR-027: an `*InvalidError` (or the kernel's
`*modifier.InvalidError`) carrying a typed `Reason` that matches
`srderr.ErrInvalidInput`; tests assert the reason and `errors.Is`/`errors.As`,
never the message. `NewRoll` returns the kernel error from `RollMode.Select`
unchanged. The error
covers an invalid `Roll`, a bad `ReplaceDie` or `ApplyReroll`, a
`FailWithoutRoll` combination without an SRD rule, and `ResolveCheck` with an
attack roll (FR-009, FR-010, FR-014, FR-052, FR-059, FR-062): invalid input from
callers, returned and never panicked.

## 6. Interface

- No use case, command, attempt type or port here (D-17); no new test double or
  contract suite.
- Domain API: the types in section 5.
- Package `d20` exports `Roll`, `NewRoll`, `MustRoll`,
  `RerollDecision` (`KeepRoll`, `RerollDie`), `RollModifiers`,
  `NewRollModifiers`, `DifficultyClass` and its `DC...` constants, `ArmorClass`,
  `Kind`, `Outcome`, `ForcedFailureReason`, `CheckResult`, `AttackRollResult`,
  `ForcedFailureResult`, `DefaultCriticalRange`, `ResolveCheck`, `ResolveAttack`,
  `FailWithoutRoll`, `Reason`, `InvalidError`, alongside the use case exports of
  resolve-d20-test-use-case.md. It does not re-export `Modifier`, `RollMode` or
  other kernel types. Its package doc comment (`doc.go`) lists inputs and
  outputs.
- There is no facade or re-export package. Downstream code resolves a rerolled
  or adjusted roll with the three functions and `AdjustedBy` without the use
  case (D-18, FR-069, FR-070), and no generic variant names are exported
  (`doc/rules/architecture.md` §4).

## 7. Observability

Silent by design: the domain is pure (`doc/rules/observability.md` §3) and all
`Resolver` signals belong to resolve-d20-test-use-case.md
(shared-types-and-ownership.md FR-040, FR-043). No signal is defined here and
none feeds `doc/observability/signals.md` from this spec. No log is emitted.

1. **What can go wrong?** A caller builds an invalid roll, reroll or forced
   failure (FR-009, FR-010, FR-014, FR-052, FR-059, FR-062): an `*InvalidError`
   at the call site. Wrong rule maths would show as a drifting outcome mix at the
   use case.
2. **How would an operator know?** Through the use case's counter and span,
   whose critical-hit and outcome attributes come from `AttackRollResult` and
   `Outcome` (shared-types-and-ownership.md FR-041).
3. **What to diagnose with?** The results carry dice, kept die, modifiers,
   total, target and outcome (FR-034, FR-041).
4. **What to count or time?** Nothing is counted in the domain; the use case
   counts and times.
5. **Is existing instrumentation still correct?** Nothing here can go stale.
6. **What to monitor?** The use case's critical-hit share; this spec adds the
   `IsCriticalHit` method it uses.
7. **What is noisy?** Nothing; no log is emitted.

## 8. Test plan

Levels: unit (external `package d20_test` files `roll_test.go`,
`roll_modifiers_test.go`, `rules_test.go`, `kind_test.go`, `error_test.go`),
architecture (`internal/archtest`). Use case and scenario tests are in
resolve-d20-test-use-case.md (`resolver_test.go`, `telemetry_test.go`,
`decoupling_test.go`, `example_test.go`, `scenarios/d20_test.go`). Row names
refer to the case tables in section 4. Tests are table-driven with `t.Run`;
`FuzzResolveAttack` exercises the attack rules over arbitrary dice, modifiers and
ranges.

| Ids | Level | Notes |
|---|---|---|
| FR-009, FR-010 | unit | Reroll rows "value 0", "value 21", "construct with a die of 0/21/negative", "wrong die count", "three dice", "no dice under advantage", "normal with two dice" (the kernel reasons); control "normal with one die" |
| FR-011, FR-055, FR-058 | unit | Reroll row "equal dice"; Advantage and Disadvantage attack rows; `[18, 3]` under both modes; kept index after each replacement |
| FR-012, FR-013 | unit | Reroll replace rows; "replace the unused die"; other dice unchanged in position |
| FR-014 | unit | Reroll index and value rows asserting `ReasonIndexOutOfRange` / `ReasonFaceOutOfRange` |
| FR-015, FR-068 | unit | Original equal before and after `ReplaceDie` and `ApplyReroll`; "source list mutated after construction"; mutating the `Dice()` slice leaves the roll unchanged |
| FR-059 | unit | Reroll `ApplyReroll` rows; nil decision; exhaustive type switch over the two variants |
| FR-067 | unit | Row "replacement twice": no error, valid roll |
| FR-016 to FR-020 | unit | Total cases through `ResolveCheck` and `ResolveAttack`; zero-other rows; Proficiency Bonus row; penalty rows; reordered rows plus a permutation of every row's `others`; the zero `RollModifiers` totals +0 |
| FR-061 | unit | Mutate source slice; mutate the `Others()` slice; empty default |
| FR-021, FR-022 | unit | `DifficultyClass` constants; `DifficultyClass`/`ArmorClass` equality cases (cross-type comparison is a compile error, not tested) |
| FR-023 to FR-026, FR-031 | unit | Check resolution table, run for both kinds; natural 20 and natural 1 rows |
| FR-027 to FR-030, FR-032 to FR-034, FR-051 | unit | Attack table by row group (ordinary, natural 20, natural 1, natural 19, Advantage/Disadvantage via `RollMode.Select`, the "far below/above" rows); `Hit()`; outcome and `IsCriticalHit()` columns |
| FR-071 to FR-073 | unit, fuzz | Range 19 and 18 rows of the attack table (crit, non-crit, natural 20/1, invalid 17 and 21 as `ReasonCriticalRange`); `AdjustedBy` keeps the range; `Equal` differs by range; `FuzzResolveAttack` |
| FR-035, FR-036, FR-052 | unit | Forced failure table, success and error rows |
| FR-041, FR-050 | unit | Result fields per variant; returned slices are copies; equal and unequal pairs, one inequality case per field for every domain handoff type |
| FR-060 | unit + architecture | Exhaustive `switch` over `Outcome` with two cases; exported-name scan finds no `AttackOutcome`, `AttackConclusion` or `AttackRollOutcome` |
| FR-062 | unit | `ResolveCheck(AttackRoll, ...)` returns `ReasonKindMismatch`; unknown kind; zero roll |
| FR-063 to FR-066 | unit | Re-resolution table by row group: check/save rerolls, attack rerolls, `AdjustedBy` rows, natural 19/20/1 adjusted rows, "adjustment leaves the original", "`ResolveCheck` with attackRoll" |
| FR-069, FR-070 | unit + architecture | Check, attack and forced-failure tables through the three functions; `AdjustedBy` equals a direct call of the function with the extra modifier; `rules.go` imports no port or telemetry package and declares no goroutine (`internal/archtest`) |

- **Doubles and contract suites:** none; domain tests use plain values
  (`doc/rules/testing.md` §3, `observability.md` §9).
- **Table-driven tests:** one subtest per row of the section 4 tables, each
  citing the requirement in a comment.
- **Mutation testing:** scope the pure domain files of package `d20` (score
  at least 80), run with `gomu` only inside Docker with `--cpus` and `--workers`
  capped at a quarter of the host's processors (command in testing.md §8).
  Boundary rows (equal, one below, one above, natural 19, 20, 1; first and last
  die index; values 0, 1, 20, 21) kill `>=` to `>` and `==` to `!=` mutants.
- **Not covered at runtime:** compile-time guarantees (one required ability
  modifier, one Proficiency Bonus slot, a DC only for a check or save and an AC
  only for an attack roll); the compiler enforces them.

## 9. Open questions and decisions

No open questions. Settled decisions cited elsewhere:

- D-02: no range check on `DifficultyClass`/`ArmorClass`; the SRD never bounds
  them (p. 6) and rejecting values would invent a rule.
- D-10: the Exhaustion reduction (p. 181) is a plain negative `Modifier` in
  `others`, wrapped by the caller; conditions.md FR-041 publishes the number.
- D-15: with equal dice the kept die is the first index (FR-058).
- D-16: `RollModifiers` slots are plain `Modifier`s because `d20` cannot
  import `character` and the bounded derived ability modifier cannot build the
  boundary cases; callers convert (ability-scores.md FR-027).
- D-17: command and attempt types are use-case types owned by
  resolve-d20-test-use-case.md.
- D-18: re-resolution is published as the package functions and `AdjustedBy` on
  the result value objects, not static members (FR-063 to FR-066).
- D-19: results carry no `Ability` and no separate roll mode (FR-041).
- D-20: no hidden replaced state in `Roll`; the caller enforces the SRD
  limit (FR-067). Revisit if a consumer needs it enforced (a replaced-index
  field would make a second replacement return an error).
- D-21: the pure rules functions are the published service; no static
  delegates (FR-069, FR-070; shared-types-and-ownership.md FR-044, FR-048).
- D-22: critical range is an input (FR-071 to FR-073) so features that widen it
  (class-fighter.md FR-090) are reachable; no other spec owns it.
- D-01: `RollMode`, `RollCircumstance`, `Resolve`, `Select` live in the kernel;
  `Roll` is here (shared-types-and-ownership.md FR-001, FR-002).
- D-03: `d20` owns and publishes `ArmorClass`; the attack outcome is
  `Outcome` plus `AttackRollResult.IsCriticalHit()`, with no parallel enum
  (shared-types-and-ownership.md FR-002, FR-004).
- D-04: one use case, `Resolver` (resolve-d20-test-use-case.md).
- D-05: structured `RollModifiers` (shared-types-and-ownership.md FR-003).
- D-06: the `Ability` label is not a domain question; it belongs to the use case
  (resolve-d20-test-use-case.md FR-055).
- D-07: no per-call log (shared-types-and-ownership.md FR-043).
- D-08: one authority for the use case and its signals, resolve-d20-test-use-case.md
  (shared-types-and-ownership.md FR-040).
- D-09, D-13, D-14: circumstance list, failed-call duration and `2d20` versus
  two `1d20` requests are use-case matters (resolve-d20-test-use-case.md).
- D-11: Death Saving Throws are `combat`'s; FR-031 stays attack-roll-only.
- D-12: reroll operations are here (FR-012 to FR-015, FR-059, FR-067); the
  Inspiration slot and mechanics are elsewhere (shared-types-and-ownership.md
  FR-002, FR-010, FR-048).

## 10. Changelog impact

`CHANGELOG.md`, `Unreleased`, **Added**: "D20 Tests (package `d20`, domain
model): `Roll` with reroll and replace-a-die operations, `RollModifiers`,
`DifficultyClass`, `ArmorClass`, `Outcome`, `AttackRollResult` with
`IsCriticalHit`, forced failures, the pure `ResolveCheck`, `ResolveAttack` and
`FailWithoutRoll` functions, and the natural 20 and natural 1 attack rules,
using the kernel's `modifier.Modifier` and `modifier.RollMode`." Not
**BREAKING**: first release of these members. The `Resolver` use case and its
`srd5e.d20_test.*` signals are recorded by resolve-d20-test-use-case.md; from
the first release containing them, renaming any signal or attribute key is
**BREAKING** (`doc/rules/observability.md` §5).
