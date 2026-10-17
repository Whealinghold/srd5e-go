# Modifiers and Advantage/Disadvantage

| | |
|---|---|
| Status | implemented |
| Context | `modifier` (kernel package) |
| Roadmap phase | 0 |
| Depends on | none |

## 1. Purpose and scope

Defines the two small value-level concepts of a D20 Test that every context
shares:

1. `Modifier`: a signed whole number added to the number rolled on the d20,
   and the rule that modifiers are added together.
2. Advantage and Disadvantage: how any number of Advantage and Disadvantage
   circumstances on one roll resolve to a roll mode (Advantage, Disadvantage
   or normal), how many d20s that means rolling, and which d20 is used.

All of it is pure, synchronous, deterministic domain logic. Rolling is done by
the `dice.Roller` port; this spec only consumes the rolled values (package `modifier` does not import `dice`).

Ownership (shared-types-and-ownership.md FR-001, FR-002): package `modifier` owns
`Modifier` (plain signed integer, no kind tag) and `RollMode` (`Normal`,
`WithAdvantage`, `WithDisadvantage`, with `Resolve`, `DiceCount`, `Select`).
`RollCircumstance` (`Advantage`, `Disadvantage`) is `Resolve`'s input
type and stays here (FR-042). `Roll`, `RollModifiers` and the outcome types
belong to package `d20` (`d20-tests.md`), which calls `RollMode.Select`.

Relationships (none is a code dependency): package `modifier` imports only
`srderr` (for `srderr.ErrInvalidInput`) and the standard library; it imports no other rules package. Consumers include `d20-tests.md`, `conditions.md`,
`attacks.md`, `initiative-and-actions.md` and `derived-stats.md`.
`ability-scores.md` and `dice-and-rolls.md` are sibling kernel specs, not
consumers (`ability-scores.md` FR-027).

**Out of scope:**

- Comparing a total to a DC or AC, success and failure, natural 20 and 1
  (SRD 5.2.1, "Rolling 20 or 1", p. 7); `Roll`, rerolls, `RollModifiers`
  (shared-types-and-ownership.md FR-002, FR-003): `d20-tests.md`. `Select` is
  pure, so `d20` can call it again with a replaced value.
- Deriving an ability modifier from a score (SRD 5.2.1, "Ability Modifiers",
  pp. 5-6; `ability-scores.md`), the Proficiency Bonus and its "doesn't stack"
  rule (p. 8; `derived-stats.md`) and the "Round Down" rule (p. 5). Here a
  `Modifier` is just a number.
- Which circumstances grant Advantage or Disadvantage (Blinded, SRD 5.2.1,
  p. 177; Dodge, p. 181; features, spells, GM rulings). The callers of
  `Resolve` are `resolve-d20-test-use-case.md` FR-006, `conditions.md`
  FR-105, `attacks.md` FR-011, `grappling.md` FR-037 and
  `spell-effects.md` FR-024, each deciding for its own command.
- The d20 reduction from Exhaustion ("D20 Tests Affected", p. 181) and Heroic
  Inspiration (p. 8; `heroic-inspiration.md`, shared-types-and-ownership.md
  FR-010, FR-048).
- Any modifier stacking or ordering rule beyond plain addition (the SRD states
  none) and any cap on modifier size (FR-035 withdrawn).

## 2. Source

- SRD 5.2.1, "D20 Tests", steps "Roll 1d20" and "Add Modifiers", p. 6. The
  page says "Add these modifiers to the number rolled on the d20" and speaks of
  a "bonus or penalty". It does not say that modifiers are whole numbers, that
  a total may leave 1 to 20, or that order is irrelevant; requirements resting
  on those points are marked "derived from".
- SRD 5.2.1, "Ability Modifiers" table, p. 6 (the +0 row); "Rolling 20 or 1",
  p. 7.
- SRD 5.2.1, "Advantage/Disadvantage", pp. 7-8, incl. "Roll Two D20s" and
  "They Don't Stack" (p. 8).
- SRD 5.2.1, "Rules Glossary", "Advantage", p. 176; "Disadvantage", p. 181;
  "D20 Test", p. 180.
- Package design: `doc/rules/architecture.md` (§3, §6), `doc/rules/testing.md` (§4).

Page numbering: PDF page index equals printed page number (no offset). In the
case tables of section 4, each FR id's own Source cell applies to its rows.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| D20 Test | concept (no type here) | Ability check, saving throw or attack roll (pp. 6, 180). |
| modifier; bonus / penalty | `Modifier` | Bonus is a `Modifier` above zero, penalty below zero; not separate types ("Circumstantial Bonuses and Penalties", p. 6). |
| the number rolled on the d20 | `int` 1 to 20 (`Select`'s result) | The used d20 value, before modifiers. |
| total | `int` | d20 number plus modifiers (p. 6). |
| Advantage / Disadvantage | `modifier.Advantage` / `modifier.Disadvantage` (`RollCircumstance`) | Positive / negative circumstances of a d20 roll (p. 7). |
| neither | `modifier.Normal` | SRD: "the roll has neither of them" (p. 8). Name fixed by shared-types-and-ownership.md FR-001, FR-041; its string `normal` is the value of `srd5e.d20_test.roll_mode`. It is the zero `RollMode`. |
| (a roll with) Advantage / Disadvantage | `modifier.WithAdvantage` / `modifier.WithDisadvantage` (strings `advantage`, `disadvantage`) | Resolved state of one roll. |
| roll two d20s | `RollMode.DiceCount()` = 2 | "Roll Two D20s", p. 8. |

`RollMode` is a package name for an SRD concept without a noun and contains no
term from another edition. A rejected input is a `*modifier.InvalidError` (section 5).

## 4. Requirements

Ids are stable; withdrawn ids are in the table at the end of this section.

Modifier.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | A `Modifier` MUST hold one whole number (any `int`, no bound) and expose it as `Value()` (constructed by `modifier.New(v)`, which accepts any `int` and cannot fail); positive (bonus), zero and negative (penalty) values are all valid. | derived from SRD 5.2.1, "D20 Tests", "Add Modifiers", p. 6 ("bonus or penalty") | happy |
| FR-002 | Two `Modifier`s with the same value MUST be equal under `==` (the type is comparable, so it can be a map key); different values MUST NOT be equal. | package design (architecture.md §3, value objects) | happy |
| FR-003 | The zero value `Modifier{}` MUST have value 0 (a useful zero value, +0). | SRD 5.2.1, "Ability Modifiers" table, +0 row, p. 6 | unhappy (empty and minimal) |
| FR-004 | `modifier.Sum(ms ...Modifier)` over any number of modifiers and `Modifier.Add` MUST give the arithmetic sum of the values. | SRD 5.2.1, "D20 Tests", "Add Modifiers", p. 6 | happy |
| FR-005 | The sum of no modifiers (`Sum()`) MUST be the zero `Modifier`. | derived from SRD 5.2.1, "D20 Tests", p. 6 (no modifiers: the total is the d20 number) | unhappy (empty and minimal) |
| FR-006 | Bonuses and penalties MUST combine by signed addition: a mixed collection sums to the net value, which MAY be negative. | SRD 5.2.1, "D20 Tests", "Circumstantial Bonuses and Penalties", p. 6 | unhappy (rule interaction) |
| FR-007 | The sum MUST NOT depend on the order of the modifiers. | derived from SRD 5.2.1, "D20 Tests", "Add Modifiers", p. 6 | unhappy (rule interaction) |
| FR-008 | Applying a modifier to a d20 number MUST give number plus value, with no clamping: the total MAY be below 1 or above 20. `ApplyTo` accepts any `int` and does not validate 1 to 20 (that check belongs to `RollMode.Select`, FR-027), so `ApplyTo(0)` and `ApplyTo(25)` return `Value()` and `25 + Value()` and return no error. | derived from SRD 5.2.1, "D20 Tests", "Compare the Total to a Target Number", p. 6 (no clamping is stated); package design | unhappy (boundary) |
| FR-009 | Summing MUST NOT alter the slice passed in (`Sum(ms...)`) or its elements. | package design (testing.md §4, immutability) | unhappy (immutability) |
| FR-036 | Summing and applying MUST be exact plain integer arithmetic with no bound or wrap added by the kernel: the sum of two `Modifier`s of 1000000 is 2000000, and `ApplyTo` returns number plus value. Exactness is that of Go `int` on the platform (at least 32 bits, 64 on the supported targets); the kernel adds no overflow guard because no SRD value approaches the limit. | package design (silent wrong totals are the worst failure of a rules engine) | unhappy (boundary) |
| FR-037 | Adding the zero `Modifier` to any modifier or sum MUST leave the value unchanged. | derived from SRD 5.2.1, "Ability Modifiers" table, +0 row, p. 6 | unhappy (empty and minimal) |

Resolving Advantage and Disadvantage.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | With no `RollCircumstance` the roll mode MUST be `Normal`. | SRD 5.2.1, "D20 Tests", "Roll 1d20", p. 6 | unhappy (empty and minimal) |
| FR-011 | With exactly one `Advantage` and no `Disadvantage` the roll mode MUST be `WithAdvantage`. | SRD 5.2.1, "Rules Glossary", "Advantage", p. 176 | happy |
| FR-012 | With exactly one `Disadvantage` and no `Advantage` the roll mode MUST be `WithDisadvantage`. | SRD 5.2.1, "Rules Glossary", "Disadvantage", p. 181 | happy |
| FR-013 | Two or more `Advantage` circumstances and no `Disadvantage` MUST resolve to `WithAdvantage`. | SRD 5.2.1, "Advantage/Disadvantage", "They Don't Stack", p. 8 | unhappy (rule interaction) |
| FR-014 | Two or more `Disadvantage` circumstances and no `Advantage` MUST resolve to `WithDisadvantage`. | SRD 5.2.1, "Advantage/Disadvantage", "They Don't Stack", p. 8 | unhappy (rule interaction) |
| FR-015 | At least one `Advantage` and at least one `Disadvantage` MUST resolve to `Normal`, whatever the counts (1 and 1, 3 and 1, 1 and 3, 4 and 4). They cancel as a whole, not one for one. | SRD 5.2.1, "They Don't Stack", p. 8 ("even if multiple circumstances impose Disadvantage and only one grants Advantage or vice versa"); "Rules Glossary", "Advantage", p. 176 | unhappy (rule interaction) |
| FR-016 | The resolved mode MUST NOT depend on the order of the circumstances. | derived from SRD 5.2.1, "They Don't Stack", p. 8 (only presence is described) | unhappy (rule interaction) |
| FR-017 | Resolving MUST NOT alter the slice passed in. | package design (testing.md §4, immutability) | unhappy (immutability) |
| FR-039 | Adding further `Advantage` circumstances to a set that already resolves to `WithAdvantage` MUST NOT change the mode (1, 2 and 5 all give the same mode). | SRD 5.2.1, "They Don't Stack", p. 8 ("you still roll only two d20s") | unhappy (rule interaction) |
| FR-040 | Adding further `Disadvantage` circumstances to a set that already resolves to `WithDisadvantage` MUST NOT change the mode. | SRD 5.2.1, "They Don't Stack", p. 8 | unhappy (rule interaction) |
| FR-034 | `Advantage` values MUST be equal to one another under `==`; `Disadvantage` likewise; an `Advantage` MUST NOT equal a `Disadvantage` (compared as `RollCircumstance` interface values). A type switch over a `RollCircumstance` with an `Advantage` case and a `Disadvantage` case covers every variant value (the interface's doc comment lists them); no implementation can exist outside the package because the marker method is unexported. The value-receiver marker methods also make `*Advantage` and `*Disadvantage` satisfy the interface; those pointers are not variants (see FR-042). | package design (architecture.md §3, value objects; encodes "They Don't Stack", p. 8) | happy |
| FR-042 | `RollCircumstance` (a closed variant set: an interface with the unexported marker method `rollCircumstance()`, variants `Advantage` and `Disadvantage`) MUST be defined in the kernel as the input type of `modifier.Resolve(cs ...RollCircumstance)`, which MUST accept any number of circumstances (a caller with a slice passes `cs...`; the slice is read once and not retained), and the variants MUST carry no data (no cause, no source). `Resolve` MUST ignore a nil circumstance and a pointer to a variant (`*Advantage`, `*Disadvantage`) exactly as if it had not been passed, without error or panic; only the `Advantage` and `Disadvantage` values count. The kernel imports nothing from `conditions.md`. | package design; shared-types-and-ownership.md FR-001, FR-013, FR-026 | happy |

Dice count.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | `Normal` MUST require 1 d20 (`DiceCount() == 1`). | SRD 5.2.1, "D20 Tests", "Roll 1d20", p. 6; "They Don't Stack", p. 8 ("you roll one d20") | happy |
| FR-019 | `WithAdvantage` MUST require 2 d20s. | SRD 5.2.1, "Advantage/Disadvantage", "Roll Two D20s", p. 8 | happy |
| FR-020 | `WithDisadvantage` MUST require 2 d20s. | SRD 5.2.1, "Advantage/Disadvantage", "Roll Two D20s", p. 8 | happy |

Selecting the used d20. `RollMode.Select(rolled []int) (int, error)` returns the used
number as an `int`; it builds no `Roll`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-021 | With `Normal`, the used number MUST be the one d20 rolled. | SRD 5.2.1, "D20 Tests", "Roll 1d20", p. 6 | happy |
| FR-022 | With `WithAdvantage`, the used number MUST be the higher of the two d20s. | SRD 5.2.1, "Roll Two D20s", p. 8 ("Use the higher of the two rolls") | happy |
| FR-023 | With `WithDisadvantage`, the used number MUST be the lower of the two d20s. | SRD 5.2.1, "Roll Two D20s", p. 8 ("use the lower roll") | happy |
| FR-024 | When the two d20s are equal, the used number MUST be that value under either mode. | derived from SRD 5.2.1, "Roll Two D20s", p. 8 | unhappy (boundary) |
| FR-025 | At the extremes: `WithAdvantage` with 1 and 20 MUST use 20; `WithDisadvantage` with 1 and 20 MUST use 1. | SRD 5.2.1, "Roll Two D20s", p. 8 | unhappy (boundary) |
| FR-026 | `Select` with the wrong number of values for the mode (0, 1 or 3 or more for `WithAdvantage`/`WithDisadvantage`; 0, 2 or more for `Normal`) MUST return 0 and an error that is a `*modifier.InvalidError` with `Reason` `ReasonRollCountMismatch` (matching `srderr.ErrInvalidInput` with `errors.Is`; the offending values are not retained). A list failing both this check and the range check (FR-027) reports every problem found through `errors.Join`, so `errors.Is` matches either `*InvalidError` reason; no order between the problems is claimed. | SRD 5.2.1, "Roll Two D20s", p. 8 ("roll a second d20"); "D20 Tests", p. 6 | unhappy (invalid input) |
| FR-027 | `Select` with any value outside 1 to 20 (0, 21, negatives) MUST return 0 and a `*modifier.InvalidError` with `Reason` `ReasonFaceOutOfRange` (matching `srderr.ErrInvalidInput`; errors-and-results.md FR-025); 1 and 20 MUST be accepted. | SRD 5.2.1, "Rolling 20 or 1", p. 7; derived from "D20 Tests", p. 6 | unhappy (boundary, invalid input) |
| FR-030 | `Select` MUST NOT alter the input slice. | package design (testing.md §4, immutability) | unhappy (immutability) |
| FR-031 | `Select` MUST be deterministic and pure: the same mode and values always give the same used number. | package design (architecture.md §3, purity) | unhappy (determinism) |
| FR-038 | Swapping the two values given to `Select` MUST NOT change the used number under `WithAdvantage` or `WithDisadvantage`. | SRD 5.2.1, "Roll Two D20s", p. 8 | unhappy (determinism) |

Package enforcement.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-043 | The kernel MUST provide no conversion between `ability.Modifier` and `modifier.Modifier`; neither package imports the other. Callers convert with `modifier.New(abilityModifier.Value())` (`ability-scores.md` FR-027); `RollModifiers.ability` in `d20-tests.md` is a plain `Modifier` (`d20-tests.md` section 4.3, D-16). | package design; shared-types-and-ownership.md FR-001, FR-003; ability-scores.md FR-027 | unhappy (rule interaction) |
| FR-044 | `ParseRollMode(s)` MUST return the mode whose `String()` equals `s` ignoring case (`normal`, `advantage`, `disadvantage`) and otherwise `Normal` with an error that is a `*InvalidError` with `Reason` `ReasonUnknownRollMode`; it MUST NOT panic for any string, and `Detail` MUST NOT echo an unbounded input. An undefined `RollMode` value passed to `Select` is rejected with the same reason and `String()` returns `unknown` for it. | package design (errors-and-results.md FR-025, FR-035); the SRD names no text form | unhappy (invalid input) |
| FR-045 | `*InvalidError` MUST match `srderr.ErrInvalidInput` with `errors.Is`, match another `*InvalidError` with the same `Reason` and no other `Reason`, and be readable with `errors.As`; `Reason` MUST have a `String()` (`unknown` outside the defined values). The zero `Reason` is not a rejection. | package design (errors-and-results.md FR-020, FR-023) | happy |

### Case tables

FR-004 to FR-008, FR-036, FR-037, modifier arithmetic.

| Label | Inputs | Expected | Requirement | Path |
|---|---|---|---|---|
| one bonus | [+3] | +3 | FR-004 | happy |
| two bonuses | [+2, +3] | +5 | FR-004 | happy |
| one penalty | [-2] | -2 | FR-001, FR-004 | happy |
| bonus and penalty, net positive | [+3, -2] | +1 | FR-006 | unhappy (rule interaction) |
| bonus and penalty, net zero | [+2, -2] | 0 | FR-006 | unhappy (rule interaction) |
| bonus and penalty, net negative | [+1, -4] | -3 | FR-006 | unhappy (rule interaction) |
| empty | [] | 0 | FR-005 | unhappy (empty and minimal) |
| zero among others | [0, +2] | +2 | FR-037 | unhappy (empty and minimal) |
| zero value | `Modifier{}` | value 0 | FR-003 | unhappy (empty and minimal) |
| reordered | [-4, +1] vs [+1, -4] | both -3 | FR-007 | unhappy (rule interaction) |
| d20 1, penalty | 1 with -5 | total -4 | FR-008 | unhappy (boundary) |
| d20 20, bonus | 20 with +10 | total 30 | FR-008 | unhappy (boundary) |
| d20 10, no modifiers | 10 with [] | total 10 | FR-005, FR-008 | unhappy (empty and minimal) |
| large sum | [+1000000, +1000000] | +2000000 | FR-036 | unhappy (boundary) |
| ApplyTo below 1 | 0 with +2 | total 2, no error | FR-008 | unhappy (boundary) |
| ApplyTo above 20 | 25 with -3 | total 22, no error | FR-008 | unhappy (boundary) |
| extreme total | 20 with +1000000 | total 1000020 | FR-036 | unhappy (boundary) |
| extreme negative total | 1 with -1000000 | total -999999 | FR-036 | unhappy (boundary) |

FR-010 to FR-016, FR-039, FR-040, FR-042, resolution (A = count of `Advantage`,
D = count of `Disadvantage`).

| Label | A | D | Expected mode | Requirement | Path |
|---|---|---|---|---|---|
| none | 0 | 0 | normal | FR-010 | unhappy (empty and minimal) |
| one Advantage | 1 | 0 | advantage | FR-011 | happy |
| one Disadvantage | 0 | 1 | disadvantage | FR-012 | happy |
| two Advantage | 2 | 0 | advantage | FR-013, FR-039 | unhappy (rule interaction) |
| many Advantage | 5 | 0 | advantage | FR-013, FR-039 | unhappy (rule interaction) |
| two Disadvantage | 0 | 2 | disadvantage | FR-014, FR-040 | unhappy (rule interaction) |
| many Disadvantage | 0 | 5 | disadvantage | FR-014, FR-040 | unhappy (rule interaction) |
| one each | 1 | 1 | normal | FR-015 | unhappy (rule interaction) |
| many Advantage, one Disadvantage | 3 | 1 | normal | FR-015 | unhappy (rule interaction) |
| one Advantage, many Disadvantage | 1 | 3 | normal | FR-015 | unhappy (rule interaction) |
| many each | 4 | 4 | normal | FR-015 | unhappy (rule interaction) |
| order: Disadvantage first | D, A | | normal | FR-016 | unhappy (rule interaction) |
| order: mixed | A, D, A, D, A | | normal | FR-016 | unhappy (rule interaction) |
| slice reused after the call | `cs := []RollCircumstance{Advantage{}, Disadvantage{}}`; `Resolve(cs...)` called twice | | normal both times, `cs` unchanged | FR-042, FR-017 | unhappy (rule interaction) |
| variadic call, Advantage only | `Resolve(Advantage{}, Advantage{})` | | advantage | FR-042, FR-017 | unhappy (rule interaction) |
| duplicates of both kinds in one call | `Resolve(Advantage{}, Disadvantage{}, Advantage{}, Disadvantage{})` | | normal | FR-042 | unhappy (rule interaction) |
| empty call and nil slice | `Resolve()` and `Resolve(nil...)` | | normal | FR-042, FR-010 | unhappy (empty and minimal) |

FR-034, circumstance equality.

| Label | Inputs | Expected | Requirement | Path |
|---|---|---|---|---|
| Advantage with Advantage | `Advantage{}` vs `Advantage{}` | equal under `==` | FR-034 | happy |
| Disadvantage with Disadvantage | `Disadvantage{}` vs `Disadvantage{}` | equal under `==` | FR-034 | happy |
| Advantage with Disadvantage | `Advantage{}` vs `Disadvantage{}` (as `RollCircumstance`) | not equal | FR-034 | unhappy (rule interaction) |
| pointer and nil circumstances | `Resolve(&Advantage{}, Disadvantage{})`, `Resolve(nil, Advantage{})`, `Resolve((*Disadvantage)(nil), Advantage{})` | disadvantage, advantage, advantage (ignored: no error, no panic) | FR-042 | unhappy (empty and minimal) |
| exhaustive type switch | `switch c.(type)` with two value cases | handles both variant values (pointers are not variants); a test in the package adds a compile-time `var _ RollCircumstance = Advantage{}` assertion for each | FR-034 | happy |

FR-018 to FR-027, FR-038, selection (values in the order given). Dice count
(FR-018 to FR-020): `Normal` 1, `WithAdvantage` 2, `WithDisadvantage` 2. The mode column holds the `RollMode.String()` labels.

| Label | Mode | Values | Expected used | Requirement | Path |
|---|---|---|---|---|---|
| single d20 | normal | [14] | 14 | FR-021 | happy |
| single d20, lowest | normal | [1] | 1 | FR-021, FR-027 | unhappy (boundary) |
| single d20, highest | normal | [20] | 20 | FR-021, FR-027 | unhappy (boundary) |
| Advantage, SRD example | advantage | [18, 3] | 18 | FR-022 | happy |
| Disadvantage, SRD example | disadvantage | [18, 3] | 3 | FR-023 | happy |
| Advantage, swapped | advantage | [3, 18] | 18 | FR-022, FR-038 | unhappy (determinism) |
| Disadvantage, swapped | disadvantage | [3, 18] | 3 | FR-023, FR-038 | unhappy (determinism) |
| Advantage, equal | advantage | [7, 7] | 7 | FR-024 | unhappy (boundary) |
| Disadvantage, equal | disadvantage | [7, 7] | 7 | FR-024 | unhappy (boundary) |
| Advantage, extremes | advantage | [1, 20] | 20 | FR-025 | unhappy (boundary) |
| Disadvantage, extremes | disadvantage | [20, 1] | 1 | FR-025 | unhappy (boundary) |
| Advantage, both 1 | advantage | [1, 1] | 1 | FR-024 | unhappy (boundary) |
| Disadvantage, both 20 | disadvantage | [20, 20] | 20 | FR-024 | unhappy (boundary) |
| normal, no values | normal | [] | `*InvalidError` (`ReasonRollCountMismatch`) | FR-026 | unhappy (invalid input) |
| normal, two values | normal | [4, 9] | `*InvalidError` (`ReasonRollCountMismatch`) | FR-026 | unhappy (invalid input) |
| Advantage, no values | advantage | [] | `*InvalidError` (`ReasonRollCountMismatch`) | FR-026 | unhappy (invalid input) |
| Advantage, one value | advantage | [12] | `*InvalidError` (`ReasonRollCountMismatch`) | FR-026 | unhappy (invalid input) |
| Advantage, three values | advantage | [1, 2, 3] | `*InvalidError` (`ReasonRollCountMismatch`) | FR-026 | unhappy (invalid input) |
| Disadvantage, one value | disadvantage | [12] | `*InvalidError` (`ReasonRollCountMismatch`) | FR-026 | unhappy (invalid input) |
| Disadvantage, three values | disadvantage | [5, 6, 7] | `*InvalidError` (`ReasonRollCountMismatch`) | FR-026 | unhappy (invalid input) |
| wrong count and out of range | advantage | [0] | both `ReasonRollCountMismatch` and `ReasonFaceOutOfRange` (`errors.Join`) | FR-026, FR-027 | unhappy (invalid input) |
| value zero | normal | [0] | `*InvalidError` (`ReasonFaceOutOfRange`) | FR-027 | unhappy (boundary) |
| value 21 | normal | [21] | `*InvalidError` (`ReasonFaceOutOfRange`) | FR-027 | unhappy (boundary) |
| value negative | normal | [-3] | `*InvalidError` (`ReasonFaceOutOfRange`) | FR-027 | unhappy (invalid input) |
| one die out of range under Advantage | advantage | [20, 21] | `*InvalidError` (`ReasonFaceOutOfRange`) | FR-027 | unhappy (boundary) |
| one die out of range under Disadvantage | disadvantage | [0, 10] | `*InvalidError` (`ReasonFaceOutOfRange`) | FR-027 | unhappy (boundary) |

### Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-028 | Withdrawn: moved to d20-tests.md (FR-011, FR-055) |
| FR-029 | Withdrawn: moved to d20-tests.md (FR-068) |
| FR-032 | Withdrawn: moved to d20-tests.md (FR-050) |
| FR-033 | Withdrawn: moved to d20-tests.md (FR-009, FR-010, FR-011); kernel side is FR-026, FR-027, FR-022, FR-023 |
| FR-035 | Withdrawn: settled, no spec. `Modifier` has no bound (shared-types-and-ownership.md FR-001); `modifier.New` has no failure case, and `Modifier` is comparable, so it can sit in a package-level table (testing.md §6) |
| FR-041 | Withdrawn: settled, no spec. No purity scan of the kernel packages (architecture-guardrails.md FR-015, FR-027); purity shown by FR-030, FR-031 |

## 5. Model

All types are immutable, comparable value objects (unexported fields, value
receivers, no setters). No entities, aggregates, events or services beyond the
stateless functions below. Package `modifier` (`modifier/modifier.go`,
`roll_circumstance.go`, `roll_mode.go`, `error.go`, `doc.go`).

- **`Modifier`**: a struct holding one `int`. `modifier.New(v int) Modifier`
  accepts any `int` and cannot fail; the zero value is `+0` and is useful
  (FR-003). `Add(Modifier) Modifier` and `Sum(...Modifier) Modifier` go through
  the same arithmetic; `ApplyTo(d20 int) int` returns `int` (FR-008, FR-036);
  `Value() int` reads it; `String()` renders `+3`, `+0`, `-2`. Equality is `==`
  (FR-002). No `ability.Modifier` conversion (FR-043).
- **`RollCircumstance`**: an interface with the unexported marker method
  `rollCircumstance()`; the closed variant set is `Advantage` and `Disadvantage`
  (FR-042), both empty comparable structs with a `String()` method. Only
  counts matter, which encodes "they don't stack".
- **`RollMode`**: an `int` enum with `Normal` (the zero value), `WithAdvantage`
  and `WithDisadvantage`, and a `String()` (`normal`, `advantage`,
  `disadvantage`; `unknown` for an undefined value).
  - `Resolve(cs ...RollCircumstance) RollMode`: the only place the cancel rule
    lives (FR-010 to FR-017, FR-039, FR-040). It never fails.
  - `(RollMode) DiceCount() int`: 1 or 2 (FR-018 to FR-020).
  - `(RollMode) Select(rolled []int) (int, error)`: the used d20 number
    (FR-021 to FR-027, FR-030, FR-031, FR-038); the single owner of the
    higher/lower rule. `dice.Result.SumOfHighest(1)`/`SumOfLowest(1)` in
    `dice-and-rolls.md` are generic dice arithmetic and are not used to pick a
    D20 Test die; the `[18, 3]` rows guard that both agree.
  - `ParseRollMode(s string) (RollMode, error)` (FR-044): accepts the three
    `String()` names, case-insensitively; anything else is an
    `*InvalidError` with `ReasonUnknownRollMode`.
- **`InvalidError` and `Reason`**: `*InvalidError{Reason, Detail}` is the typed
  rejection of untrusted input. `Reason` is an `int` enum
  (`ReasonRollCountMismatch`, `ReasonFaceOutOfRange`, `ReasonUnknownRollMode`)
  with a `String()`. The error matches `srderr.ErrInvalidInput` with
  `errors.Is`, and matches another `*InvalidError` with the same `Reason`
  (`Detail` is ignored). `Detail` is for logs; nothing branches on it
  (errors-and-results.md FR-020).
- No `Roll`, `RollModifiers` or outcome type here
  (shared-types-and-ownership.md FR-002); `Select` returns a plain `int`.

Invariants and enforcement:

| Invariant | Enforced by |
|---|---|
| `len(rolled) == mode.DiceCount()` | `Select` (FR-026): `*InvalidError`, `ReasonRollCountMismatch` |
| every rolled value in 1..20 | `Select` (FR-027): `*InvalidError`, `ReasonFaceOutOfRange`; every problem is reported through `errors.Join`, no check order claimed |
| the mode is one of the three defined values | `Select` and `ParseRollMode` (FR-044): `ReasonUnknownRollMode` |
| returned number is what the mode selects | `Select` (FR-022, FR-023), the only code that picks a die |
| a `Modifier` holds one `int`, equal by value | the type itself (FR-001, FR-002); `Add` and `Sum` cannot violate it |
| input not mutated | `Select` and `Resolve` only read it (FR-017, FR-030) |

Failures. Every combination of circumstances resolves to a mode, so `Resolve`
has no failure case and there are no expected-outcome results. A wrong dice
count, an out-of-range die or an undefined mode is untrusted input (a roller
can return anything): `Select` and `ParseRollMode` return `(T, error)` and
never panic for any input of their parameter type (errors-and-results.md
FR-025). There is no `MustX` helper and no panic path in this package. A bad
`dice.Roller` result is caught upstream by `dice-and-rolls.md` FR-051; `Select`
is a second line of defence.

## 6. Interface

- No commands, results, use cases or ports. The `d20` resolver
  (`d20-tests.md`; shared-types-and-ownership.md FR-044) calls `RollMode.Select`
  (inside `d20.NewRoll`); `resolve-d20-test-use-case.md` uses `DiceCount`
  to size the request it makes on `dice.Roller`. The callers of `Resolve` are
  listed in section 1. `modifier` consumes no interface and declares none.
- Everything is synchronous, pure and takes no `context.Context` (decision 2b).
- Public API of package `github.com/Whealinghold/srd5e-go/modifier`, one
  package with no facade and no re-export from the root `srd5e` package:
  `Modifier`, `New`, `Sum`, `RollCircumstance`, `Advantage`, `Disadvantage`,
  `RollMode`, `Normal`, `WithAdvantage`, `WithDisadvantage`, `Resolve`,
  `ParseRollMode`, `InvalidError`, `Reason` and its three constants, each with a
  godoc comment starting with its name. `Roll` is exported from package
  `d20` (names are unique: shared-types-and-ownership.md FR-021).
- The package MUST NOT import `dice`, `telemetry` or any context; its only
  module import is `srderr` (`internal/archtest`).

## 7. Observability

Silent by design (`doc/rules/observability.md` §2): pure in-memory integer
operations, no I/O. The only failures are rejected untrusted inputs
(a `*modifier.InvalidError` from `Select` or `ParseRollMode`, which the calling use case wraps or surfaces and whose span records it, observability.md §7) and wrong rule results, caught by
unit tests. The `d20` resolver owns the signals, including
`srd5e.d20_test.roll_mode` (shared-types-and-ownership.md FR-040, FR-041,
FR-043; `resolve-d20-test-use-case.md` section 7). The domain emits nothing
(observability.md §3).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none | | | | |

Existing instrumentation: none, nothing is affected. Consumers watching `d20` outcome rates can use `srd5e.d20_test.roll_mode` to spot an unexpected shift in Advantage frequency. No signal is added, so nothing becomes noisy, and nothing is added to `doc/observability/signals.md`.

## 8. Test plan

All tests are unit tests in `modifier/modifier_test.go` (external package `modifier_test`; no ports, doubles or I/O; `t.Parallel()` throughout). Fuzz tests: `FuzzSelect` and `FuzzParseRollMode`. Testable documentation: `ExampleResolve` and `ExampleRollMode_Select`. The package has 100% statement coverage.
Rule maths is table-driven from the case tables in section 4, one test per row
(`Requirement: per case`). No contract suite, no application tests; scenario
coverage belongs to `d20-tests.md`.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-003 | unit | values +3, 0, -2 readable; equal and unequal pairs under `==` (also as a map key); the zero `Modifier` is 0 |
| FR-004 | unit | rows "one bonus", "two bonuses", "one penalty", through `modifier.Sum` and again through `Add` |
| FR-005 to FR-007 | unit | rows "empty", "d20 10, no modifiers"; "bonus and penalty" nets; "reordered" |
| FR-008 | unit | rows "d20 1, penalty", "d20 20, bonus": no clamping; "ApplyTo below 1", "ApplyTo above 20": any `int` accepted, no error |
| FR-009 | unit | input slice and elements unchanged after `Sum` |
| FR-036, FR-037 | unit | rows "large sum" (via `Sum` and `Add`), "extreme total", "extreme negative total"; "zero among others"; add-zero identity via `Add` and `Sum` |
| FR-010 to FR-016 | unit | resolution table: none, one, 2 and 5 of each kind, 1/1, 3/1, 1/3, 4/4; two orderings give the same mode |
| FR-017 | unit | input slice unchanged; the rows "slice reused after the call" and "duplicates of both kinds in one call" |
| FR-039, FR-040 | unit | 1, 2 and 5 of the same kind give the same mode |
| FR-034 | unit | equality table (`==` on `RollCircumstance` interface values) and a two-case type switch |
| FR-042 | unit | exactly two variants, no fields (FR-034 switch row); the variadic, nil-slice and empty-call rows |
| FR-018 to FR-020 | unit | `DiceCount` is 1, 2, 2 |
| FR-021 to FR-025 | unit | selection table: single d20, SRD example, equal-dice and extreme rows under both modes |
| FR-026, FR-027 | unit | wrong-count rows (incl. "wrong count and out of range": asserts that `errors.Is(err, &InvalidError{Reason: ...})` finds each of the two reasons on the joined error, no check order); 0, 1, 20, 21, negative; one-die-out-of-range rows; assertions use `errors.Is(err, srderr.ErrInvalidInput)` and `errors.As` to a `*InvalidError` and compare `Reason`, never message text |
| FR-030, FR-031, FR-038 | unit | slice unchanged after `Select`; repeat call same number; swapped-order rows |
| FR-043 | architecture | the import check of `ability-scores.md` FR-027 is the enforcement; no signature scan is claimed |
| FR-044, FR-045 | unit, fuzz | `TestRollModeBasics`-style table for `ParseRollMode` (three names, mixed case, empty, unknown, over-long input) and `String()` of an undefined value; `TestInvalidErrorAndReason` (`errors.Is` against the sentinel and against a same-reason `*InvalidError`, not against another reason; `Reason.String()`); `FuzzParseRollMode` and `FuzzSelect` assert no panic and that every error matches `srderr.ErrInvalidInput` |

- **Doubles and contract suites:** none required.
- **Mutation testing:** the files of package `modifier`, with gomu run only inside
  Docker with `--cpus` and `--workers` capped at 1/4 of `nproc` (testing.md §8;
  never run locally; see the documented command there). The `>`/`<` in the
  higher/lower selection and the `1`/`20` bounds in FR-027 are mutation-prone;
  the boundary rows exist to kill them.
- **Architecture tests:** the `internal/archtest` allow-list row for `modifier` (it may import only `srderr`; architecture.md §9) covers this package; the purity scans
  do not reach the kernel (FR-041).
- **Not covered:** none. Every live FR id is in the table.

## 9. Open questions and decisions

| Question | Recommendation | Status |
|---|---|---|
| Who owns `RollMode`, `Roll`, the `Normal` name, source-less `Modifier`, rerolls? | shared-types-and-ownership.md FR-001, FR-002, FR-003, FR-010, FR-048. Bard Countercharm, SRD p. 33: "the new roll has Advantage", so a new D20 Test with one more Advantage circumstance (`class-bard.md` FR-084). | decided |
| Bound `Modifier`? | No (FR-035 withdrawn, FR-036): a bound on the public constructor is not an invariant if `Add` and `Sum` can exceed it, and it would make `New` fallible for no SRD reason (testing.md §6). The SRD gives no bound. | decided |
| Kernel purity scans? | None (FR-041 withdrawn); architecture.md §2, §9. | decided |
| Should `Advantage`/`Disadvantage` carry a cause? | No: counting is all the rules need (p. 8). | decided |
| `AbilityModifier` conversion in the kernel? | None: `ability-scores.md` FR-027; FR-043 here. | decided |
| Bad dice from `dice.Roller`? | `dice-and-rolls.md` FR-051, plus FR-026 and FR-027 here. | decided |
| Who picks higher/lower of two d20s? | `Select`, not `SumOfHighest(1)`/`SumOfLowest(1)` (pp. 6, 8, 21). | decided |
| Does a roll type hold the total? | No: totals belong to `d20-tests.md`. | decided |
| Which rolls take Advantage? | Only d20 rolls: D20 Tests (pp. 6-8, 176, 181) and, as an SRD fact, the Death Saving Throw (p. 17; Defy Death, p. 49). Whether the package applies it to the Death Saving Throw is open question hit-points-and-damage.md Q-4 (v1: no, FR-061); Defy Death is `class-fighter.md` FR-095. No other die is supported. | decided here; Q-4 open elsewhere |

## 10. Changelog impact

Under `Unreleased`, **Added**: package `modifier` (`github.com/Whealinghold/srd5e-go/modifier`) with `Modifier`, `RollCircumstance` (`Advantage`,
`Disadvantage`), `RollMode`, `Resolve`, `ParseRollMode` and `InvalidError`, with resolution of Advantage and Disadvantage
and selection of the used d20. Not
**BREAKING** (new API). Later changes to this surface are breaking-change
candidates because every context shares the kernel (architecture.md §2; the Go compatibility promise is in doc/rules/go.md).
