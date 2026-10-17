# Rests and Hit Dice

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `ability-scores.md`, `observability-ports.md`, `content-schema-and-decoder.md`, `derived-stats.md`, `character-aggregate.md` |

## 1. Purpose and scope

Specifies the Short Rest and Long Rest and spending Hit Point Dice, as the SRD
states them: durations, when a rest may start, what stops it and what an
interrupted one still confers, what a finished rest restores (Hit Points, Hit
Point Dice, a reduced Hit Point maximum, reduced ability scores, one Exhaustion
level, Temporary Hit Points, features), and the Hit Point Dice rule (spend one
die, roll it, add the Constitution modifier, at least 1 Hit Point regained).
That rule includes the one spend needing no rest: the Warlock's Lifedrinker
("expend one of your Hit Point Dice to roll it", SRD 5.2.1, "Lifedrinker",
p. 73), which updates the same `SpentHitDice` (FR-069). It **owns** `RestKind`
and the `RestApplied` event. Section 5.4 lists the deferrals it satisfies.

The package has no clock and no current Hit Points in `character`. The host
reports what happened (hours rested, what interrupted) and the facts the rules
need (current Hit Points, Exhaustion level); the spec returns what the rest
confers as effects the host applies in the context that owns each.

**Out of scope:**

- Applying Hit Points, Temporary Hit Points and a Hit Point maximum:
  `hit-points-and-damage.md`. The `Exhaustion` condition and its levels:
  `conditions.md` (this spec decides only that a Long Rest lowers it by 1, Q-4, and takes the level as a plain `int`).
- Exhaustion from dehydration or malnutrition, which "can't be removed" by a
  rest until the creature drinks or eats the full amount (SRD 5.2.1,
  "Dehydration", p. 181; "Malnutrition", p. 185): `environment-hazards-and-objects.md`
  (`RemoveExhaustionLevels`, Q-9).
- Each class feature's own recharge state: the `class-*.md` specs (Table R, 4.8,
  records which rest recharges what; a scenario test pins it). Spell slot
  recovery and prepared-spell changes: `spellcasting.md`.
- Monster Hit Dice ("A creature can spend Hit Dice during a Short Rest", SRD
  5.2.1, "Hit Point Dice", p. 183): the d4 to d20 dice of `content-monsters.md`;
  monsters are not `character` (Q-8).
- The sleeping Unconscious condition in a Long Rest (SRD 5.2.1, "Long Rest",
  p. 185) and the knocked-out creature's Short Rest ending Unconscious (SRD
  5.2.1, "Knocking Out a Creature", p. 17): Q-7.
- Attunement in a Short Rest (SRD 5.2.1, p. 102): `magic-item-rules.md`. Mounts
  needing a rest (p. 12): `exploration.md`. Feat renewal: `feats-behaviour.md`.
  They consume `RestKind`, the resolution and `RestApplied` (5.4).
- The sleep and light-activity requirements of a Long Rest ("sleep for at least
  6 hours and perform no more than 2 hours of light activity", SRD 5.2.1, "Long
  Rest", p. 185) and the Short Rest activity limit ("nothing more strenuous
  than reading, talking, eating, or standing watch", p. 187): host
  preconditions (FR-063).
- Effects that block or stand in for rest benefits: Sewer Plague (SRD 5.2.1,
  "Magical Contagions", p. 194), a pure filter in
  `toolbox-curses-and-contagions.md`; Demonic Possession (p. 193) reacts to `RestApplied(LongRest)`; the Raise
  Dead and Resurrection penalty and strain (pp. 157, 159) are defined in
  `revival-and-restoration-spells.md` FR-016 to FR-022 (`ReviveOrdeal.AfterRest`,
  `CasterStrain.AfterRest` take `RestApplied.BenefitsOf()`); Prayer of Healing
  ("gain the benefits of a Short Rest", p. 154) uses FR-064.
- Reviving a dead creature (SRD 5.2.1, "Dead", p. 180); counting Long Rests for
  a feature (Greater Divine Intervention, p. 38: the class spec counts
  `RestApplied`); time (the host, `hit-points-and-damage.md` Q-12); any rule
  from another edition or outside SRD 5.2.1.

## 2. Source

- SRD 5.2.1, "Healing" and "Knocking Out a Creature", p. 17; "Temporary Hit
  Points", "Duration", p. 18; "Rest" sidebar, p. 16; "Character Creation",
  Step 5, "Hit Point Dice", p. 22; "Gaining a Level", step 2, p. 23;
  "Multiclassing", "Hit Points and Hit Point Dice", p. 25.
- SRD 5.2.1, "Rules Glossary": "Exhaustion", p. 181; "Hit Points", "Hit Point
  Dice", p. 183; "Long Rest", p. 185; "Per Day", p. 186; "Short Rest", p. 187.
- Rest triggers in class text (Tables R, W): Barbarian p. 28 to 30; Bard p. 32;
  Cleric p. 37, 38; Druid p. 42, 43, 46; Fighter p. 48; Monk p. 50 to 52;
  Paladin p. 53 to 55, 57; Ranger p. 58, 59, 61; Rogue p. 62, 63; Sorcerer
  p. 65, 66, 70; Warlock p. 71 to 74, 76; Wizard p. 77 to 79, 82.
- Species and feats: "Dwarf", "Dragonborn", "Elf" (Elven Lineage), p. 84; "Elf"
  (Trance), "Gnome", p. 85; "Goliath", pp. 85 to 86; "Human", "Orc",
  "Tiefling" (Fiendish Legacy), p. 86; "Magic Initiate", p. 87; "Boon of
  Fate", p. 88. Warlock "Lifedrinker", p. 73.
- Out of scope, section 1: "Prayer of Healing", p. 154; "Raise Dead",
  "Resurrection", pp. 157, 159; "Demonic Possession", p. 193; "Magical
  Contagions", p. 194.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Short Rest, Long Rest | `RestKind` (`ShortRest`, `LongRest`) | Owned here (`shared-types-and-ownership.md` FR-009). Reported by the host. |
| "Rolling Initiative", "Casting a spell other than a cantrip", "Taking any damage", "1 hour of walking or other physical exertion" | `RestInterruption` (`RollingInitiative`, `CastingSpellOtherThanCantrip`, `TakingDamage`, `PhysicalExertion`) | The Interrupting the Rest lists, pp. 185, 187. The Short Rest list has three entries; `PhysicalExertion` also stops a Short Rest by its definition (p. 187, FR-019, Q-10). |
| Interrupting the Rest | `RestEnd` (closed variant set: `RestRanItsCourse`, `RestStopped`) | How the host reports a rest ended. |
| (a rest report) | `RestReport` | Kind, Long Rest variant, and a `RestEnd`; `NewRestReport` for host input (FR-066). |
| Trance ("finish a Long Rest in 4 hours") | `LongRestVariant` (`StandardLongRest`, `TranceLongRest`) | The host says whether the creature has the trait. |
| "wait at least 16 hours before starting another one" | `LongRestSpacing` | Time comes from the host. |
| Hit Point Dice | `HitPointDie` (`content-schema-and-decoder.md` FR-132, published by `content`, `shared-types-and-ownership.md` FR-053: `d6`, `d8`, `d10`, `d12`, fact `Faces()`; no die type is defined here); pool `HitDicePool` (`derived-stats.md` FR-061, FR-070); spent ones `SpentHitDice` (owned here). The use case rolls `dice.NewExpression(1, d, 0)` (with `d` the `dice.Die` of `die.Faces()` sides) | |
| Special Feature (recharged by a rest) | `FeaturesRecharged(RestKind)`; `RechargeTrigger` (`content-schema-and-decoder.md` FR-112); `RestKind.Recharges(content.RechargeTrigger)` | The mapping is owned here (`shared-types-and-ownership.md` FR-009; FR-032, FR-065). |
| Exhaustion level | plain `int` 0 to 6 | `conditions.md` owns `ExhaustionLevel`; `character` cannot import it (Q-4). |

## 4. Requirements

Ids are stable. Required hours: Short Rest 1, Long Rest 8, Long Rest with Trance
4. All hours are whole hours.

### 4.1 Vocabulary and durations

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `RestKind` MUST have exactly two values, `ShortRest` and `LongRest`, and no other rest value. | SRD 5.2.1, "Short Rest", p. 187; "Long Rest", p. 185 | happy |
| FR-002 | The minimum duration of a Short Rest MUST be 1 hour. | SRD 5.2.1, "Short Rest" ("a 1-hour period of downtime"), p. 187 | happy |
| FR-003 | The minimum duration of a Long Rest of the `StandardLongRest` variant MUST be 8 hours. | SRD 5.2.1, "Long Rest" ("at least 8 hours"), p. 185 | happy |
| FR-004 | The minimum duration of a Long Rest of the `TranceLongRest` variant MUST be 4 hours. The variant MUST NOT change a Short Rest. | SRD 5.2.1, "Elf", Trance, p. 85 | happy |
| FR-005 | `RestInterruption` MUST have exactly four values: `RollingInitiative`, `CastingSpellOtherThanCantrip`, `TakingDamage`, `PhysicalExertion`. Casting a cantrip MUST NOT be representable as an interruption. | SRD 5.2.1, "Long Rest", "Interrupting the Rest", p. 185; "Short Rest", p. 187 | unhappy (boundary: cantrip) |
| FR-006 | The interruptions that stop a Short Rest MUST be `RollingInitiative`, `CastingSpellOtherThanCantrip` and `TakingDamage` (its list) and `PhysicalExertion` (inference from its definition, "nothing more strenuous than reading, talking, eating, or standing watch", Q-10); those that stop a Long Rest MUST be all four (table 4.8a). | SRD 5.2.1, "Short Rest", "Interrupting the Rest" and definition, p. 187; "Long Rest", p. 185 | happy |

### 4.2 Starting a rest

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-007 | A Short Rest MUST be startable with at least 1 current Hit Point, and MUST NOT be startable at 0 (`RestMayNotStart` with `RestStartNoHitPoints`). A creature at exactly 1 Hit Point, as a knocked-out creature is, MAY start one. | SRD 5.2.1, "Short Rest" ("you must have at least 1 Hit Point"), p. 187; "Knocking Out a Creature", p. 17 | unhappy (boundary: 0 and 1) |
| FR-008 | A Long Rest MUST be startable with at least 1 current Hit Point, and MUST NOT be startable at 0 (`RestStartNoHitPoints`). | SRD 5.2.1, "Long Rest" ("you must have at least 1 Hit Point"), p. 185 | unhappy (boundary: 0 and 1) |
| FR-009 | A Long Rest MUST NOT be startable fewer than 16 hours after the previous Long Rest finished (only a `RestCompleted(LongRest)` counts as finished, FR-072; `RestStartTooSoonAfterLongRest(hoursStillToWait)`, where `HoursStillToWait` is 16 minus the hours since the previous Long Rest, at least 1); at exactly 16 hours it MUST be startable; with no previous Long Rest it MUST NOT be blocked by spacing. | SRD 5.2.1, "Long Rest" ("you must wait at least 16 hours before starting another one"), p. 185 | unhappy (boundary: 15, 16, 17 hours) |
| FR-010 | A Short Rest MUST NOT be blocked by `LongRestSpacing`: one may be taken straight after a Long Rest finished. | SRD 5.2.1, "Rest" paragraph ("hour-long Short Rests in the midst of a day and an 8-hour Long Rest to end it"), p. 16; "Short Rest", p. 187 (no spacing rule) | unhappy (rule interaction) |
| FR-011 | A creature with Exhaustion level 6 MUST NOT be able to start either rest (`RestStartExhaustionLevelSix`). Inference: a creature at level 6 has died, so it cannot rest (Q-10). | SRD 5.2.1, "Exhaustion", "Exhaustion Levels" ("You die if your Exhaustion level is 6"), p. 181 | unhappy (state error) |
| FR-012 | When several blockers apply, `RestMayNotStart` MUST list every one, once each, in the fixed order `RestStartNoHitPoints`, `RestStartExhaustionLevelSix`, `RestStartTooSoonAfterLongRest`, and never only the first. | architecture.md section 5 (list every problem, not only the first); testing.md section 4 (unhappy-path coverage) | unhappy (rule interaction) |
| FR-013 | A negative current Hit Points value, a negative hours value or an Exhaustion level outside 0 to 6 MUST be rejected by the constructors (`NewTakeRestCommand`, `NewCheckRestStart`-style domain functions) with an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (a host input error, not a rule outcome); the error lists every problem (FR-066). Nothing panics. | architecture.md section 6 | unhappy (invalid input) |

### 4.3 Ending a rest: completion and interruption
`RestRanItsCourse(hoursRested, interruptionsResumed)` says the rest was not
stopped and lasted that long after resuming `InterruptionsResumed` times.
`RestStopped(interruption, hoursRestedBefore, interruptionsResumed)` says it was
stopped by that interruption after `HoursRestedBefore` hours, resumed that many
times before (FR-071 needs the count). Hours are cumulative totals (FR-067); the
1-hour threshold of FR-020 and FR-021 is on the total, not per segment.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `RestRanItsCourse` with `HoursRested` at or above the required hours MUST resolve to `RestCompleted(kind)`, which grants the benefits of that `RestKind`. | SRD 5.2.1, "Short Rest", "Benefits of the Rest" ("When you finish the rest"), p. 187; "Long Rest", p. 185 | happy |
| FR-015 | `RestRanItsCourse` with `HoursRested` below the required hours MUST resolve to `RestIncomplete(hoursStillNeeded)` and grant nothing. One hour short MUST give `HoursStillNeeded` 1. | SRD 5.2.1, "Short Rest", p. 187; "Long Rest", p. 185 | unhappy (boundary) |
| FR-016 | A Long Rest resumed after `n` interruptions MUST require the base hours plus `n` ("1 additional hour per interruption") of total rested time (FR-067). With the `TranceLongRest` variant the base is 4 (inference, Q-12). | SRD 5.2.1, "Long Rest", "Interrupting the Rest", p. 185 | unhappy (boundary) |
| FR-017 | A Short Rest report with `InterruptionsResumed` above 0 (on `RestRanItsCourse` or on `RestStopped`) MUST resolve to `RestReportRejected(ShortRestCannotBeResumed)`, the one variant of the closed `RestReportRejection`: a rule result of `ResolveRest`, not a creation failure, because the report is well-typed (FR-066). The SRD gives resumption for a Long Rest only (Q-11). | SRD 5.2.1, "Long Rest" ("You can resume a Long Rest immediately after an interruption"), p. 185; "Short Rest" ("An interrupted Short Rest confers no benefits"), p. 187 | unhappy (state error) |
| FR-018 | A Short Rest stopped by `RollingInitiative`, `CastingSpellOtherThanCantrip` or `TakingDamage` MUST resolve to `RestStoppedWithoutBenefits`, however many hours had been rested. | SRD 5.2.1, "Short Rest", "Interrupting the Rest" ("An interrupted Short Rest confers no benefits"), p. 187 | unhappy (state error) |
| FR-019 | A Short Rest reported as stopped by `PhysicalExertion` MUST resolve to `RestStoppedWithoutBenefits`, however many hours had been rested, exactly as FR-018. Inference: the Short Rest definition allows nothing more strenuous than reading, talking, eating or standing watch, so strenuous activity ends it even though the three-entry list omits it; the host can therefore report a Short Rest that was spent exerting (Q-10). | SRD 5.2.1, "Short Rest" (definition and "Interrupting the Rest"), p. 187 | unhappy (state error) |
| FR-020 | A Long Rest stopped by any of the four interruptions after at least 1 hour rested and fewer hours than the required hours of FR-014 and FR-016 (FR-071) MUST resolve to `RestStoppedWithShortRestBenefits`, which grants the benefits of `ShortRest`. | SRD 5.2.1, "Long Rest" ("If you rested at least 1 hour before the interruption, you gain the benefits of a Short Rest"), p. 185 | happy |
| FR-021 | A Long Rest stopped after 0 hours rested MUST resolve to `RestStoppedWithoutBenefits`; after exactly 1 hour it MUST resolve as FR-020. | SRD 5.2.1, "Long Rest", p. 185 | unhappy (boundary: 0 and 1 hour) |
| FR-022 | Every one of the four interruptions MUST stop a Long Rest (table 4.8a). | SRD 5.2.1, "Long Rest", "Interrupting the Rest", p. 185 | happy |
| FR-023 | A negative `HoursRested`, `HoursRestedBefore` or `InterruptionsResumed` (on either `RestEnd` variant) MUST be rejected by `NewRestReport` with an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` that lists every such problem (FR-066). A rule outcome is never returned this way. | architecture.md section 6 | unhappy (invalid input) |

### 4.4 Benefits of a rest
`NewRestBenefits(benefitsOf, exhaustionLevel, spentHitDice)` returns the effects in
the fixed order of FR-034. Effects that would change nothing for a known reason
are left out (FR-026, FR-029); the others are always present because the host
knows the current Hit Points, maximum, ability scores and Temporary Hit Points.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-024 | The benefits of a Short Rest MUST be exactly two effects: `SpendHitPointDiceAvailable` and `FeaturesRecharged(ShortRest)`. | SRD 5.2.1, "Short Rest", "Benefits of the Rest", p. 187 | happy |
| FR-025 | The benefits of a Long Rest MUST include `AllLostHitPointsRegained`. | SRD 5.2.1, "Long Rest", "Regain All HP", p. 185 | happy |
| FR-026 | The benefits of a Long Rest MUST include `HitPointDiceRegained(count)` with `count` equal to the total number of spent Hit Point Dice when that total is above 0, and MUST omit it when none are spent. The spent dice then become none. | SRD 5.2.1, "Long Rest", "Regain All HP" ("all lost Hit Points and all spent Hit Point Dice"), p. 185 | happy / unhappy (boundary: 0 spent) |
| FR-027 | The benefits of a Long Rest MUST include `HitPointMaximumReturnedToNormal`, an effect the host applies to its Hit Point maximum (Q-15). | SRD 5.2.1, "Long Rest" ("If your Hit Point maximum was reduced, it returns to normal"), p. 185 | happy |
| FR-028 | The benefits of a Long Rest MUST include `AbilityScoresReturnedToNormal`, an effect the host applies by lifting its ability score reductions (Q-15). | SRD 5.2.1, "Long Rest", "Ability Scores Restored", p. 185 | happy |
| FR-029 | The benefits of a Long Rest at Exhaustion level `n` of 1 or more MUST include `ExhaustionLevelReduced{From: n, To: n - 1}`, and MUST omit it at level 0. Level 1 gives `To` 0, which is the end of the condition. The effect states the Long Rest rule alone; the host applies it through `environment-hazards-and-objects.md` `RemoveExhaustionLevels` (levels 1), which withholds levels locked by Dehydration or Malnutrition (SRD 5.2.1, pp. 181, 185; Q-9), so a blocked removal leaves the level unchanged. | SRD 5.2.1, "Long Rest", "Exhaustion Reduced", p. 185; "Exhaustion", "Removing Exhaustion Levels", p. 181 | unhappy (boundary: 0, 1, 2, 5) |
| FR-030 | The benefits of a Long Rest MUST include `TemporaryHitPointsEndedByRest` (a name distinct from any event: the effect is data the host applies, not something that happened, FR-059). The benefits of a Short Rest MUST NOT. | SRD 5.2.1, "Temporary Hit Points", "Duration" ("until they're depleted or you finish a Long Rest"), p. 18 | happy / unhappy (rule interaction) |
| FR-031 | The benefits of a Long Rest MUST include `FeaturesRecharged(LongRest)`. | SRD 5.2.1, "Long Rest", "Special Feature", p. 185 | happy |
| FR-032 | `FeaturesRecharged(kind)` MUST mean `kind.Recharges(trigger)` of `RestKind.Recharges(content.RechargeTrigger)` (FR-065, table 4.8f), where `RechargeTrigger` is the `content`-owned enum of `content-schema-and-decoder.md` FR-112: `ShortRest` recharges a feature whose trigger is `ShortRest` or `ShortOrLongRest`; `LongRest` one whose trigger is `LongRest` or `ShortOrLongRest`. A Long Rest MUST NOT count as also finishing a Short Rest for features that say Short Rest only (Q-8). A "Per Day" feature is data with trigger `LongRest` (SRD "Per Day": "you must finish a Long Rest to use it again"). How much is regained (one use, all uses) is the feature's own rule. | SRD 5.2.1, "Short Rest", "Special Feature", p. 187; "Long Rest", "Special Feature", p. 185; "Per Day", p. 186 | happy |
| FR-033 | The benefits of a Short Rest MUST NOT include any Hit Point, Hit Point Dice, Hit Point maximum, ability score, Exhaustion or Temporary Hit Points effect. | SRD 5.2.1, "Short Rest", p. 187 | unhappy (rule interaction) |
| FR-034 | The effects of a Long Rest MUST be ordered `AllLostHitPointsRegained`, `HitPointDiceRegained`, `HitPointMaximumReturnedToNormal`, `AbilityScoresReturnedToNormal`, `ExhaustionLevelReduced`, `TemporaryHitPointsEndedByRest`, `FeaturesRecharged`, skipping omitted ones, so equal inputs give equal lists. The order is package design: it follows the SRD list (Regain All HP, which covers Hit Points, Hit Point Dice and the Hit Point maximum; Ability Scores Restored; Exhaustion Reduced; Special Feature) with the Hit Point Dice and Hit Point maximum split out and the Temporary Hit Points ending (p. 18) placed before the features. | package design; SRD 5.2.1, "Long Rest", "Benefits of the Rest", p. 185; "Temporary Hit Points", "Duration", p. 18 (inputs) | unhappy (determinism) |
| FR-035 | `RestStoppedWithShortRestBenefits` MUST give exactly the effects of FR-024. `RestStoppedWithoutBenefits`, `RestIncomplete` and a rejected report MUST give no effects and leave the spent Hit Point Dice unchanged. | SRD 5.2.1, "Long Rest", p. 185; "Short Rest", p. 187 | unhappy (state error) |
| FR-036 | `RestApplied{CharacterID, BenefitsOf, Source, Effects}` MUST be a past-tense event naming the `RestKind` whose benefits were applied (`ShortRest` for an interrupted Long Rest, FR-020), the `RestSource` (`RestFinished`, or `LongRestInterrupted` for FR-020, FR-061) and the effect list. `RestInterrupted{CharacterID, Kind, Interruption, BenefitsGranted}` MUST be raised for every rest that resolves to `RestStoppedWithShortRestBenefits` or `RestStoppedWithoutBenefits`, and for no other result; a stop that FR-071 resolves to `RestCompleted` raises `RestApplied` only (LI-11 to LI-13, LI-15). It carries `BenefitsGranted` of the closed type `RestBenefitsGranted` (`RestBenefitsOfShortRest`, `RestBenefitsNone`). Both are variants of the closed `RestEvent` family (FR-068). | architecture.md section 3 (domain events); `shared-types-and-ownership.md` FR-019, FR-020; `character-aggregate.md` FR-085 | happy |

### 4.5 Hit Point Dice
`SpentHitDice` holds how many dice of each size are spent, keyed by
`HitPointDie`; the pool is the `HitDicePool` of `derived-stats.md` (FR-061,
FR-070: a count per `HitPointDie`, entries by ascending faces). An absent pool
key means 0 (Q-5); no `CountOf` accessor is assumed. "d8" means `HitPointDie.d8`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-037 | `SpentHitDice` MUST hold a count of at least 1 for each `HitPointDie` listed and omit dice with none spent, so equal spent sets compare equal; a count below 1 MUST be rejected by `NewSpentHitDice` with an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (FR-073); the zero `SpentHitDice` is the empty set. A die other than d6, d8, d10 or d12 cannot be expressed, because `HitPointDie` is a closed enum. Entries are ordered by ascending `HitPointDie.Faces()`, as in the pool. | SRD 5.2.1, "Hit Point Dice", p. 22 ("note how many Hit Dice you've spent"); class "Hit Point Die" traits, pp. 28 to 77 | unhappy (invalid input) |
| FR-038 | The dice left of a size MUST be the pool's count of that size minus the spent count, and MUST NOT be negative. A `SpentHitDice` whose count for any size exceeds the pool's count (host-stored state that disagrees with the pool) MUST make `Spend` and `SpendOutsideRest` return `HitDieSpendRejected(spentExceedsPool)` and change nothing; no operation returns an error or panics for it (FR-073). `TakeRest` has no pool and does not check this: a Long Rest clears every spent die (FR-026). | SRD 5.2.1, "Hit Point Dice", p. 22; "Multiclassing", p. 25 | unhappy (invalid input) |
| FR-039 | `SpentHitDice.Spend(pool, die, benefitsOf)` (`BenefitsOf` the `RestKind` whose benefits are claimed) with at least one die of that size left and `BenefitsOf` `ShortRest` MUST return `HitDieSpendAllowed` with the spent count of that size raised by 1, the other sizes unchanged, and the original value unchanged. | SRD 5.2.1, "Short Rest", "Spend Hit Point Dice" ("spend one or more of your Hit Point Dice"), p. 187 | happy |
| FR-040 | `Spend` of a size with none left MUST return `HitDieSpendRejected(noneRemaining)`. `Spend` of a size the pool has none of MUST return `HitDieSpendRejected(notInPool)`. Neither changes the state. When several reasons apply, the reported one follows the fixed order `NotAShortRestBenefit` (FR-045), `SpentExceedsPool` (FR-038), `NotInPool`, `NoneRemaining`. | SRD 5.2.1, "Hit Point Dice", p. 183 ("A creature can spend Hit Dice during a Short Rest"); "Multiclassing", p. 25 | unhappy (boundary: last die, no die) |
| FR-041 | Dice of different sizes MUST be tracked separately: spending a d8 MUST NOT change the d10 count, and a level 5 Cleric and level 5 Paladin MUST have five d8 and five d10 dice to spend. | SRD 5.2.1, "Multiclassing", "Hit Points and Hit Point Dice", p. 25 | unhappy (rule interaction) |
| FR-042 | A level 1 character, whose pool is one die, MUST be able to spend one die and then none (second spend `NoneRemaining`). Gaining a level adds one die to the pool and leaves the spent dice unchanged, so one more die is left to spend. | SRD 5.2.1, "Step 5", "Hit Point Dice" ("At level 1, your character has 1 Hit Die"), p. 22; "Gaining a Level", step 2 ("you gain an additional Hit Die"), p. 23; `character-aggregate.md` FR-037 | unhappy (boundary) |
| FR-043 | The Hit Points regained by spending one die MUST be the die roll plus the Constitution modifier, and at least 1 (table 4.8c). | SRD 5.2.1, "Short Rest", "Spend Hit Point Dice" ("You regain Hit Points equal to the total (minimum of 1 Hit Point)"), p. 187 | happy / unhappy (boundary) |
| FR-044 | A die roll below 1 or above `HitPointDie.Faces()` passed to `HitPointsRegained(die, roll, constitutionModifier) (int, error)` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` naming `roll` (a typed `*RollRangeError{Name, Value, Min, Max}`). A `SpendHitPointDie` run cannot reach it, because a `dice.Result` cannot hold a face its die does not have (`dice-and-rolls.md` FR-032); the check guards a host that supplies a roll directly (FR-054 reads the face from `dice.Result.DiceTotal()`). | dice-and-rolls.md FR-032 (a roll is within its die); architecture.md section 6 | unhappy (invalid input) |
| FR-045 | Spending Hit Point Dice in a rest MUST be allowed only when the benefits being claimed are those of `ShortRest` (a finished Short Rest, or a Long Rest that earned them under FR-020). `SpentHitDice.Spend(pool, die, benefitsOf)` MUST do this check: `BenefitsOf` `LongRest` MUST be rejected as `HitDieSpendRejected(notAShortRestBenefit)` (Q-3), and the `SpendHitPointDie` use case MUST only pass its command's `RestKind` to it and hold no rule of its own (FR-052, FR-053). The check does not apply to `SpendOutsideRest` (FR-069). | SRD 5.2.1, "Short Rest", "Benefits of the Rest", p. 187 (listed there only) | unhappy (state error) |
| FR-046 | Each spend MUST be one die, so the host decides whether to spend another after seeing each roll. There MUST be no operation that spends several dice at once. | SRD 5.2.1, "Short Rest" ("You can decide to spend an additional Hit Point Die after each roll"), p. 187 | happy |
| FR-047 | The result of a spend MUST carry the amount of Hit Points regained uncapped by the Hit Point maximum. Capping belongs to the host that applies Hit Points (non-normative hand-off to `hit-points-and-damage.md`: Hit Points regained in excess of the maximum are lost). | SRD 5.2.1, "Healing", p. 17; "Hit Points", p. 183 | unhappy (boundary) |
| FR-048 | `SpentHitDice` MUST be an immutable comparable-by-`Equal` value: every accessor returns a copy of its collection (mutating the returned slice or map changes nothing), the map given to `NewSpentHitDice` is copied, and an operation MUST NOT change its input. | architecture.md section 4 (handoffs), section 3 | unhappy (immutability) |

### 4.6 Use cases
Both use cases are synchronous; `TakeRest` calls only `telemetry.Telemetry`, `SpendHitPointDie` also the `dice.Roller`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-049 | `TakeRest` MUST return `TakeRestApplied` (the new `SpentHitDice` and the `RestApplied` event, whose `effects` are the effects) for `RestCompleted`, and `TakeRestAppliedAfterInterruption` (the same, plus the `RestInterrupted` event) for `RestStoppedWithShortRestBenefits`. Both are variants of the closed `TakeRestResult`; no field holds a nil or a flag for the other case. | SRD 5.2.1, "Long Rest", p. 185; "Short Rest", p. 187 | happy |
| FR-050 | `TakeRest` MUST return `TakeRestNotStarted(blockers)` when FR-007 to FR-011 block the rest, evaluated on the values at the start of the rest (FR-070), with the list of FR-012, before it looks at the end of the rest. | SRD 5.2.1, "Short Rest", p. 187; "Long Rest", p. 185 (conditions to start) | unhappy (state error) |
| FR-051 | `TakeRest` MUST return `TakeRestIncomplete(hoursStillNeeded)` for FR-015, `TakeRestStopped` (with `RestInterrupted`, no effects) for `RestStoppedWithoutBenefits`, and `TakeRestRejected(problem)` for FR-017 (`problem` is a `RestReportRejection`). The `SpentHitDice` MUST be returned unchanged in all three. | SRD 5.2.1, "Long Rest", p. 185; "Short Rest", p. 187 | unhappy (state error) |
| FR-052 | `TakeRest` MUST contain no rule logic: every outcome is the one the domain functions of 4.2 to 4.4 return for the same inputs. | architecture.md section 4 (a use case orchestrates) | happy |
| FR-053 | `SpendHitPointDie` MUST check the spend by calling `SpentHitDice.Spend(pool, die, benefitsOf)` (FR-039, FR-040, FR-045) before it rolls, and on a rejection MUST NOT call the `dice.Roller`. | architecture.md section 4; testing.md section 4 | unhappy (state error) |
| FR-054 | On an allowed spend `SpendHitPointDie` MUST build `dice.NewExpression(1, d, 0)` (with `d` the `dice.Die` of `die.Faces()` sides) from the chosen `HitPointDie` (modifier 0), roll it exactly once through `dice.Roll`, read the face from the single `dice.DieRoll` (`dice.Result.DiceTotal()`), and return `SpendHitPointDieApplied` with the roll, the Constitution modifier, the Hit Points regained (FR-043), the new `SpentHitDice` and the `HitPointDieSpent` event. | SRD 5.2.1, "Short Rest", "Spend Hit Point Dice" ("roll the die and add your Constitution modifier"), p. 187; dice-and-rolls.md FR-006, FR-050 | happy |
| FR-055 | A port failure from the roller (the `*srderr.PortError` that `dice.Roll` returns) MUST be returned unchanged with the zero result, and the state MUST NOT change; the use case MUST NOT swallow it or turn it into a result variant (`errors-and-results.md` FR-031). A roller that returns a result for a different expression (a different count, die or modifier) makes `dice.Roll` return a `*srderr.PortError` matching `srderr.ErrContractViolation`, which MUST be returned unchanged and leave the state unchanged. A roll outside the die cannot reach the use case (FR-044). | architecture.md section 4 (port failure contract); dice-and-rolls.md FR-050, FR-051, FR-052 | unhappy (failure of a collaborator) |
| FR-056 | With fixed rolls `SpendHitPointDie` MUST return the Hit Points of table 4.8c, including the extremes (roll 1 with a negative modifier, roll equal to the die's faces with a positive one). | SRD 5.2.1, "Short Rest", p. 187 | unhappy (determinism) |
| FR-057 | Each use case MUST return an equal result with the no-op `telemetry.Telemetry` and with a recording one, and MUST emit the signals of section 7, including for a returned `*srderr.PortError`: `srd5e.outcome` `failure`, `error.type` `dice_roller_error` and no `srd5e.character.result`. | observability.md section 9 | happy |
| FR-058 | Neither use case MUST read or hold current Hit Points, a clock or a `Character`; the effects they return are data the host applies in the contexts that own them (non-normative hand-off, 5.4). | `shared-types-and-ownership.md` FR-013 | happy |

### 4.7 Public surface and hand-offs

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-059 | Every type this spec exports (section 6) MUST be defined once, here, and exported from package `character`. No other type named `RestKind` or `RestApplied`, no `RestKind` member spelled `short` or `long`, and no exported name equal to one of this spec's names, MAY exist in the package (`shared-types-and-ownership.md` FR-021). The rest effect `TemporaryHitPointsEndedByRest` is data the host applies and MUST NOT share a name with any event, and no `RestEffect` variant, blocker, event, result or command name may equal an export of another spec (grep over `doc/specs/` and the package scan). Every type named in 5.1, including `RestReportRejection`, `HitDieSpendRejection`, `RestBenefitsGranted`, `RestReportProblem`, `TakeRestCommandProblem`, `SpentHitDiceProblem` and `SpendHitPointDieCommandProblem` with their members, is in that export list and the scan. Variants carry an operation prefix (`RestStart...`, `RestBenefits...`, `TakeRest...`, `SpendHitPointDie...`, `HitDieSpend...`) and no generic name (`Applied`, `Rejected`) is exported (`shared-types-and-ownership.md` FR-017). | `shared-types-and-ownership.md` FR-009, FR-017, FR-019, FR-020, FR-021; architecture.md section 9 (duplicate-name scan (`internal/archtest`)) | unhappy (duplicate) |
| FR-060 | Every class, species or feat function that reacts to finishing a rest MUST take a `RestKind` (or the `RestApplied` event) from this spec, or a context-local trigger enum mapped from it one to one (`feats-behaviour.md` `FeatRenewal`: `ShortRestFinished` from `ShortRest`, `LongRestFinished` from `LongRest`), and MUST accept both rest values, so a Short Rest and a Long Rest are each handled (Table R). | `shared-types-and-ownership.md` FR-009, FR-023 | happy |
| FR-061 | Table W (4.8) lists the "whenever you finish a ... Rest, you can ..." windows. `RestApplied` with `BenefitsOf` `LongRest` MUST be the only trigger a window that names a Long Rest accepts, and `RestApplied` with `BenefitsOf` `ShortRest` and `source` `RestFinished` the only trigger for one that names a Short Rest. A window naming Short or Long Rest MUST accept `source` `RestFinished` of either kind. Every "when/whenever you finish a Short Rest" window of Table W (W-3, W-11 to W-14 included) is a window, not a recharge: it MUST accept `source` `RestFinished` only, so an interrupted Long Rest never enables Natural Recovery, Tireless, Sorcerous Restoration or Arcane Recovery (their once-until-Long-Rest limits are recharges and stay in Table R). An interrupted Long Rest gives `RestApplied(ShortRest)` with `source` `LongRestInterrupted`: it recharges features (FR-020, FR-032) but opens no "finish a Short Rest" window. Inference: the SRD says the creature "gains the benefits of a Short Rest" (Hit Point Dice and Special Feature), not that it finishes one (Q-16). | SRD 5.2.1, the pages of Table W; "Long Rest", p. 185 | unhappy (rule interaction) |
| FR-062 | The `CheckRestStart`, `RestResolution`, `RestBenefits`, `SpentHitDice` and `HitDieRecovery` domain code MUST be pure, synchronous, deterministic and silent (no logging, metrics or spans). | architecture.md section 3 (purity); observability.md section 3 | unhappy (architecture) |
| FR-063 | The sleep and light-activity requirements of a Long Rest (at least 6 hours asleep and no more than 2 hours of light activity) and the Short Rest activity limit are host preconditions: the package sees neither. A host MUST NOT report a period that fails them as a Long Rest. Strenuous activity during a Short Rest is reported as `PhysicalExertion` (FR-019) and a Long Rest broken by 1 hour of walking or other exertion as `PhysicalExertion` (FR-022). `RestReport` and `TakeRestCommand` document this in their doc comments, and the `TakeRestCommand` comment also documents FR-070 (start values) and FR-072 (what counts as a finished Long Rest). | SRD 5.2.1, "Long Rest", p. 185; "Short Rest", p. 187 | unhappy (host precondition) |
| FR-064 | `NewRestBenefits(ShortRest, exhaustionLevel, spentHitDice)` MUST be callable without a `RestReport` and MUST return exactly the effects of FR-024 and the unchanged `SpentHitDice`, so a host can grant "the benefits of a Short Rest" to a creature that did not rest (the Prayer of Healing spell, SRD 5.2.1, p. 154). Calling it raises no `RestApplied` event, because nothing finished; `RestApplied` is raised by `TakeRest` only. | SRD 5.2.1, "Prayer of Healing", p. 154; "Short Rest", p. 187 | happy |
| FR-065 | `RestKind.Recharges(content.RechargeTrigger)` MUST be the one mapping from a rest to the `content`-owned `RechargeTrigger` (`content-schema-and-decoder.md` FR-112: `ShortRest`, `LongRest`, `ShortOrLongRest`) and MUST return true exactly as table 4.8f shows. This spec defines no wording enum of its own and no string getter of SRD wording MAY exist; "Per Day" is data with trigger `LongRest` (FR-032). The class and species specs apply the mapping through this predicate (`shared-types-and-ownership.md` FR-009). | SRD 5.2.1, "Short Rest", "Special Feature", p. 187; "Long Rest", "Special Feature", p. 185; "Per Day", p. 186 | unhappy (boundary: the Long Rest does not recharge a `ShortRest` trigger) |
| FR-066 | Input from a host (`int` values) MUST be validated by the constructors `NewRestReport` and `NewTakeRestCommand`, which return `(T, error)`. The error is `errors.Join` of every problem, once each, in input order, each a value of the enum `RestReportProblem` (`NegativeHoursRested`, `NegativeHoursRestedBefore`, `NegativeInterruptionsResumed`) or `TakeRestCommandProblem` (`NegativeCurrentHitPoints`, `ExhaustionLevelOutOfRange`); the enums implement `error`, so `errors.Is(err, NegativeHoursRested)` works, and every problem also satisfies `errors.Is(err, srderr.ErrInvalidInput)`. `NewSpentHitDice` and `NewSpendHitPointDieCommand` are in FR-073. The constructors reject only malformed values; rule results come from the domain: a resumed Short Rest is `RestReportRejected{ShortRestCannotBeResumed}` from `ResolveRest` (FR-017), and a Short Rest stopped by exertion is not rejected at all (FR-019). `MustRestReport` and the other `MustX` helpers panic with the same error and are for literals in tests. | architecture.md section 5 (list every problem), section 6; errors-and-results.md (operation-prefixed variants) | unhappy (invalid input) |
| FR-067 | `HoursRested` and `HoursRestedBefore` MUST be total rested time across all segments of one rest, excluding the interruptions themselves, as whole hours the host has floored. A Long Rest stopped after 3 hours, resumed and stopped again after 2 more hours is reported as `HoursRestedBefore` 5. FR-014, FR-015, FR-016, FR-020 and FR-021 MUST all compare this total. | SRD 5.2.1, "Long Rest" ("rested at least 1 hour before the interruption"; "1 additional hour per interruption"), p. 185; package design for the cumulative reading (Q-17) | unhappy (boundary: stop, resume, stop again) |
| FR-068 | `RestApplied`, `RestInterrupted` and `HitPointDieSpent` MUST be the variants of the closed family `RestEvent`, which implements `CharacterEvent` (`character-aggregate.md` FR-085, an interface with an unexported marker method and a `CharacterID()` method; there is no kernel base event, `shared-types-and-ownership.md` FR-019) and carries the `CharacterID` on the interface (`CharacterID() CharacterID`). A host switches exhaustively over `RestEvent` for the rest family and over `CharacterEvent` for the context (`shared-types-and-ownership.md` FR-019). Events are past tense. `TakeRestCommand` and `SpendHitPointDieCommand` carry the `CharacterID`. | `character-aggregate.md` FR-085; `shared-types-and-ownership.md` FR-019; architecture.md section 4 (handoffs) | happy |
| FR-069 | `SpentHitDice.SpendOutsideRest(pool, die)` MUST spend one die exactly as `Spend` (FR-039, FR-040, FR-038) but without a `RestKind` and without the `NotAShortRestBenefit` check, returning `HitDieSpendAllowed` or `HitDieSpendRejected` with `NoneRemaining`, `NotInPool` or `SpentExceedsPool`. It exists for the one SRD spend that is not a rest benefit, the Warlock's Lifedrinker ("you can expend one of your Hit Point Dice to roll it", p. 73), so that the die is recorded in the same `SpentHitDice` and cannot be spent again at the next Short Rest (table 4.8d SP-13 to SP-15). This spec defines no Lifedrinker use case or roll; those are `class-warlock.md` FR-078, whose `UseLifedrinker` takes and returns the `SpentHitDice` and calls `SpendOutsideRest` (Q-22). | SRD 5.2.1, "Lifedrinker", p. 73; "Hit Point Dice", p. 22 | unhappy (rule interaction) |
| FR-070 | The values on `TakeRestCommand` that decide whether a rest may start (`CurrentHitPoints`, `LongRestSpacing`) MUST be the values at the start of the rest, not at the time it is reported; `ExhaustionLevel` MUST be the level at the start of the rest and is also the level FR-029 lowers, which is the same by contract (nothing in the rest changes the level). A Long Rest stopped by `TakingDamage` that leaves the creature at 0 Hit Points when reported is therefore reported with the Hit Points it started with, and is not `TakeRestNotStarted`. The `RestReport` and `TakeRestCommand` doc comments state this (FR-063). | SRD 5.2.1, "Long Rest" and "Short Rest" ("To start a rest, you must have at least 1 Hit Point"), pp. 185, 187; package design for the contract | unhappy (rule interaction: stopped by damage, reported at 0 Hit Points; table 4.8a LI-10) |
| FR-071 | A Long Rest reported as `RestStopped(interruption, hoursRestedBefore, interruptionsResumed)` with `HoursRestedBefore` at or above the required hours (the base 8, or 4 for `TranceLongRest`, plus `InterruptionsResumed`, as FR-016) MUST resolve to `RestCompleted(LongRest)`: the required time had already passed, so the rest is finished, and the late interruption does not take the Long Rest benefits away. Below the required hours FR-020 and FR-021 apply. A Short Rest is unchanged (FR-018, FR-019): no benefits however many hours. This is an inference (Q-21). | SRD 5.2.1, "Long Rest" ("at least 8 hours"; "If you rested at least 1 hour before the interruption"), p. 185 | unhappy (boundary: 7, 8, 9 hours; resumed n plus 8) |
| FR-072 | Only a `RestCompleted(LongRest)` (FR-014, FR-071) MUST count as a finished Long Rest for `LongRestSpacing`: a Long Rest that was interrupted and earned Short Rest benefits (`RestStoppedWithShortRestBenefits`), stopped without benefits or incomplete does not start the 16-hour clock, and neither does a Short Rest (table 4.8g). The host updates `LongRestSpacing` only after a `TakeRestApplied` whose `RestApplied` has `BenefitsOf` `LongRest` and `source` `RestFinished`. The `LongRestSpacing` and `TakeRestCommand` doc comments state this. | SRD 5.2.1, "Long Rest" ("after you finish a Long Rest"), p. 185 | unhappy (rule interaction) |
| FR-073 | Host-stored spent counts MUST be validated by constructors returning `(T, error)`. `NewSpentHitDice(map[content.HitPointDie]int)` returns `errors.Join` of every problem once each from `SpentHitDiceProblem` (`CountBelowOne`). `NewSpendHitPointDieCommand` takes the host's spent counts as `map[content.HitPointDie]int` and returns `errors.Join` of problems from `SpendHitPointDieCommandProblem` (`SpentCountBelowOne`); a spent count above the pool's count is not a construction problem but the returned rejection `SpentExceedsPool` (FR-038), never an error or panic, and the use case maps it to `SpendHitPointDieRejected{SpentExceedsPool}` with its own `srd5e.character.rejection` value `spent_exceeds_pool`. All problems satisfy `errors.Is(err, srderr.ErrInvalidInput)`. | architecture.md section 5, section 6 (constructors returning errors for untrusted values) | unhappy (invalid input; table 4.8d SP-9, SP-11, SP-12) |

### 4.8 Case tables

**Table 4.8a. Interruptions by rest (FR-005, FR-006, FR-018, FR-019, FR-022, FR-067, FR-070, FR-071).** One test per row. `InterruptionsResumed` is 0 unless a row says otherwise; hours are totals; a Long Rest is `StandardLongRest` unless a row says `TranceLongRest`.

| Case | Rest | Interruption | Hours rested before | Expected | Source | Path |
|---|---|---|---|---|---|---|
| SI-1 to SI-3 | Short | `RollingInitiative`, `CastingSpellOtherThanCantrip`, `TakingDamage` (one test each) | 0 | `RestStoppedWithoutBenefits` | p. 187 | unhappy (state error) |
| SI-4 | Short | `PhysicalExertion` | 0 | `RestStoppedWithoutBenefits` (inference, Q-10) | p. 187 | unhappy (state error) |
| SI-5 | Short | `PhysicalExertion` | 1 | `RestStoppedWithoutBenefits` | p. 187 | unhappy (boundary) |
| LI-1 to LI-4 | Long | `RollingInitiative`, `CastingSpellOtherThanCantrip`, `TakingDamage`, `PhysicalExertion` (one test each) | 3 | `RestStoppedWithShortRestBenefits` | p. 185 | happy |
| LI-5, LI-6 | Long | `TakingDamage` | 1; 0 | `RestStoppedWithShortRestBenefits`; `RestStoppedWithoutBenefits` | p. 185 | unhappy (boundary) |
| LI-7 | Long | `RollingInitiative` | 7 | `RestStoppedWithShortRestBenefits` (not a Long Rest) | p. 185 | unhappy (boundary: one hour short) |
| LI-8 | Long | `TakingDamage`, after a stop at 3 hours, a resume and 2 more hours (`InterruptionsResumed` 1) | 5 (total, FR-067) | `RestStoppedWithShortRestBenefits` | p. 185 | unhappy (boundary: stop, resume, stop again) |
| LI-9 | Long | `RollingInitiative`, after a stop at 0 hours and a resume (`InterruptionsResumed` 1) | 0 | `RestStoppedWithoutBenefits` | p. 185 | unhappy (boundary: no hours in any segment) |
| LI-10 | Long | `TakingDamage`, command reports current Hit Points 12 as at the start of the rest although the creature is at 0 when the rest is reported (FR-070) | 3 | `TakeRestAppliedAfterInterruption` (Short Rest benefits); not `TakeRestNotStarted` | p. 185 | unhappy (rule interaction: stopped by damage, reported at 0 Hit Points) |
| LI-11, LI-12 | Long | `RollingInitiative` | 8; 9 | `RestCompleted(LongRest)` (inference, Q-21) | p. 185 | unhappy (boundary: required hours reached) |
| LI-13 | Long, `TranceLongRest` | `TakingDamage` | 4 | `RestCompleted(LongRest)` (inference, Q-21) | p. 185, p. 85 | unhappy (boundary) |
| LI-14 | Long | `TakingDamage`, `InterruptionsResumed` 1 | 8 | `RestStoppedWithShortRestBenefits` (required hours are 9) | p. 185 | unhappy (boundary: resumed n plus 8, one short) |
| LI-15 | Long | `TakingDamage`, `InterruptionsResumed` 1 | 9 | `RestCompleted(LongRest)` (inference, Q-21) | p. 185 | unhappy (boundary: resumed n plus 8) |

**Table 4.8b. Required hours and completion (FR-002 to FR-004, FR-014 to FR-017).** One test per row.

| Case | Rest | Variant | Resumed interruptions | Hours rested | Expected | Source | Path |
|---|---|---|---|---|---|---|---|
| RH-1 | Short | n/a | 0 | 1 | `RestCompleted(ShortRest)` | p. 187 | happy |
| RH-2 | Short | n/a | 0 | 0 | `RestIncomplete(1)` | p. 187 | unhappy (boundary) |
| RH-3 | Long | `StandardLongRest` | 0 | 8 | `RestCompleted(LongRest)` | p. 185 | happy |
| RH-4 | Long | `StandardLongRest` | 0 | 7 | `RestIncomplete(1)` | p. 185 | unhappy (boundary) |
| RH-5 | Long | `StandardLongRest` | 1 | 9 | `RestCompleted(LongRest)` | p. 185 | happy |
| RH-6 | Long | `StandardLongRest` | 1 | 8 | `RestIncomplete(1)` | p. 185 | unhappy (boundary) |
| RH-7 | Long | `StandardLongRest` | 3 | 11 | `RestCompleted(LongRest)` | p. 185 | unhappy (rule interaction) |
| RH-8 | Long | `TranceLongRest` | 0 | 4 | `RestCompleted(LongRest)` | p. 85 | happy |
| RH-9 | Long | `TranceLongRest` | 0 | 3 | `RestIncomplete(1)` | p. 85 | unhappy (boundary) |
| RH-10 | Long | `TranceLongRest` | 1 | 5 | `RestCompleted(LongRest)` | p. 185, p. 85 (derived, Q-12) | unhappy (rule interaction) |
| RH-11 | Short | `TranceLongRest` | 0 | 1 | `RestCompleted(ShortRest)` | p. 85 | unhappy (rule interaction) |
| RH-12 | Short | n/a | 1 | 1 | rejected `ShortRestCannotBeResumed` | p. 187 | unhappy (state error) |
| RH-13 | Long | `StandardLongRest` | 2 | 10 (total of segments 4, 3 and 3, FR-067) | `RestCompleted(LongRest)` | p. 185 | unhappy (boundary: stop, resume, stop again, resume) |
| RH-14 | Long | `StandardLongRest` | 2 | 9 (total) | `RestIncomplete(1)` | p. 185 | unhappy (boundary) |

**Table 4.8c. Hit Points regained by one Hit Point Die (FR-043, FR-044, FR-056).** One test per row.

| Case | Die | Roll | Constitution modifier | Expected Hit Points regained | Source | Path |
|---|---|---|---|---|---|---|
| HD-1 | d8 | 4 | +2 | 6 | p. 187 | happy |
| HD-2 | d8 | 4 | 0 | 4 | p. 187 | happy |
| HD-3 | d8 | 4 | -3 | 1 (total 1, no raise needed) | p. 187 | unhappy (boundary) |
| HD-4 | d8 | 1 | -3 | 1 (total -2, raised to 1) | p. 187 | unhappy (boundary) |
| HD-5 | d8 | 5 | -3 | 2 | p. 187 | unhappy (boundary) |
| HD-6 | d10 | 1 | -5 | 1 (total -4, raised to 1) | p. 187 | unhappy (boundary) |
| HD-7 | d12 | 12 | +5 | 17 | p. 187 | unhappy (boundary: maximum roll) |
| HD-8 | d6 | 1 | 0 | 1 | p. 187 | unhappy (boundary: minimum roll) |
| HD-9, HD-10 | d8 | 0; 9 | +2 | error `Is` `srderr.ErrInvalidInput` (roll) | dice-and-rolls.md | unhappy (invalid input) |

**Table 4.8d. Pool and spending (FR-037 to FR-042, FR-045, FR-069, FR-073).** One test per row. "Spend" is `Spend(pool, die, ShortRest)` unless a row says otherwise.

| Case | Pool | Spent before | Action | Expected | Source | Path |
|---|---|---|---|---|---|---|
| SP-1 | d10 x 1 (level 1 Fighter) | none | spend d10 | allowed, d10 spent 1 | p. 22 | happy |
| SP-2 | d10 x 1 | d10 x 1 | spend d10 | rejected `NoneRemaining` | p. 22 | unhappy (boundary) |
| SP-3 | d8 x 5, d10 x 5 (level 5 Cleric and level 5 Paladin) | none | spend d10 | allowed, d10 spent 1, d8 spent 0 | p. 25 | happy |
| SP-4 | d8 x 5, d10 x 5 | d10 x 5 | spend d10 | rejected `NoneRemaining`; spend d8 allowed | p. 25 | unhappy (boundary, rule interaction) |
| SP-5 | d8 x 5, d10 x 5 | none | spend d12 | rejected `NotInPool` | p. 25 | unhappy (invalid input) |
| SP-6 | d10 x 10 (level 5 Fighter and level 5 Paladin) | none | spend d10 ten times | ten allowed, the eleventh rejected | p. 25 | unhappy (boundary) |
| SP-7 | d10 x 1 | d10 x 1 | level up: pool d10 x 2 | one die left to spend | p. 23 | unhappy (boundary) |
| SP-8 | d10 x 1 | none | `Spend(pool, d10, LongRest)` (claiming Long Rest benefits) | rejected `NotAShortRestBenefit`; no roll in the use case | p. 187 | unhappy (state error) |
| SP-9 | d8 x 2 | d8 x 3 | any spend | rejected `SpentExceedsPool` (host-stored state disagrees with the pool); no error, no panic | p. 22 | unhappy (invalid input) |
| SP-10 | d8 x 2 | d8 x 0 given as an entry | `NewSpentHitDice` | error `Is` `srderr.ErrInvalidInput` (count below 1) | p. 22 | unhappy (invalid input) |
| SP-11 | n/a | d8 x 0 given as an entry | `NewSpentHitDice` | error joining `CountBelowOne`, `Is` `srderr.ErrInvalidInput` | p. 22 | unhappy (invalid input) |
| SP-12 | d8 x 2 | d8 x 3 as host ints | `NewSpendHitPointDieCommand` | a valid command (`SpentExceedsPool` is a returned rejection, not a construction problem, FR-073); `SpendHitPointDie` on that command returns `SpendHitPointDieRejected{SpentExceedsPool}` | p. 22 | unhappy (invalid input) |
| SP-13 | d8 x 2 | none | `SpendOutsideRest(pool, d8)` (Lifedrinker) | allowed, d8 spent 1; no `NotAShortRestBenefit` check | p. 73 | happy |
| SP-14 | d8 x 2 | d8 x 1 (from Lifedrinker) | `Spend(pool, d8, ShortRest)` then `Spend(pool, d8, ShortRest)` | first allowed (d8 spent 2), second `NoneRemaining`: the Lifedrinker die is not spendable again at the Short Rest | p. 73, p. 187 | unhappy (rule interaction) |
| SP-15 | d8 x 2 | d8 x 2 | `SpendOutsideRest(pool, d8)` | rejected `NoneRemaining` | p. 73 | unhappy (boundary) |

**Table 4.8e. Benefits of a Long Rest by state (FR-025 to FR-034).** One test per row. Effect order is FR-034.

| Case | Exhaustion level | Spent dice | Expected effects | Source | Path |
|---|---|---|---|---|---|
| LB-1 | 0 | none | all lost Hit Points regained, Hit Point maximum returned, ability scores returned, Temporary Hit Points ended, features recharged (long rest) | p. 185, p. 18 | happy |
| LB-2 | 0 | d8 x 3 | as LB-1 plus `HitPointDiceRegained(3)` after the first effect | p. 185 | happy |
| LB-3 | 1 | none | as LB-1 plus `ExhaustionLevelReduced(1 to 0)` | p. 185, p. 181 | unhappy (boundary) |
| LB-4 | 2 | d8 x 1, d10 x 2 | as LB-1 plus `HitPointDiceRegained(3)` and `ExhaustionLevelReduced(2 to 1)` | p. 185 | unhappy (rule interaction) |
| LB-5 | 5 | none | `ExhaustionLevelReduced(5 to 4)` | p. 185 | unhappy (boundary) |
| LB-6 | 6 | none | not startable (`RestStartExhaustionLevelSix`), no effects | p. 181 | unhappy (state error) |
| SB-1 | 3 | d8 x 2 | Short Rest: `SpendHitPointDiceAvailable`, features recharged (short rest) only; Exhaustion and spent dice unchanged | p. 187 | unhappy (rule interaction) |
| SB-2 | 0 | none | interrupted Long Rest with 1 hour rested: same as SB-1 | p. 185 | unhappy (boundary) |

**Table 4.8f. `RestKind.Recharges(trigger)` over `content.RechargeTrigger` (FR-032, FR-065).** One test per cell, 6 in all.

| Case | Trigger | `ShortRest` recharges | `LongRest` recharges | Source | Path |
|---|---|---|---|---|---|
| RW-1 | `ShortRest` | true | false | p. 187 | unhappy (rule interaction: a Long Rest is not also a Short Rest) |
| RW-2 | `LongRest` | false | true | p. 185 | happy |
| RW-3 | `ShortOrLongRest` | true | true | pp. 185, 187 | happy |

**Table 4.8g. What starts the 16-hour clock (FR-009, FR-072).** One test per row.

| Case | Rest result | Counts as a finished Long Rest for `LongRestSpacing` | Source | Path |
|---|---|---|---|---|
| SC-1 | `RestCompleted(LongRest)` | yes | p. 185 | happy |
| SC-2 | `RestStoppedWithShortRestBenefits` (interrupted Long Rest) | no | p. 185 | unhappy (rule interaction) |
| SC-3 | `RestStoppedWithoutBenefits` of a Long Rest | no | p. 185 | unhappy (state error) |
| SC-4 | `RestIncomplete` of a Long Rest | no | p. 185 | unhappy (state error) |
| SC-5 | `RestCompleted(ShortRest)` | no | p. 187 | unhappy (rule interaction) |
| SC-6 | `RestCompleted(LongRest)` reached through FR-071 (stopped after the required hours) | yes | p. 185 | unhappy (boundary) |

**Table R. What recharges, ends or is capped on which rest (FR-032, FR-060).**
One scenario test per row, through the owner's public API; "amount" is the
owner's rule, this spec supplies only the `RestKind`. The table lists the
class, species and feat features of SRD 5.2.1 pp. 28 to 88 whose text ties a
use, recharge, created resource or summoned creature to a rest, plus the
subclass and Invocation features the class specs model (R-17, R-27, R-34, R-39,
R-43). It is a cross-check: FR-060 is the requirement, and a feature missing
from the table is still bound by it. The "finish a Short Rest" windows are in
Table W; Table R keeps only the recharge halves.

| Row | Feature (SRD name) | `ShortRest` | `LongRest` | Source | Owner spec |
|---|---|---|---|---|---|
| R-1 | Rage uses | regain 1 expended use | regain all | "Level 1: Rage", p. 28 | class-barbarian.md FR-025, FR-026 |
| R-2 | Relentless Rage DC | resets to 10 | resets to 10 | "Level 11: Relentless Rage", p. 30 | class-barbarian.md |
| R-3 | Persistent Rage (regain Rage on Initiative) | no change | available again | "Level 15: Persistent Rage", p. 30 | class-barbarian.md FR-071 |
| R-4 | Bardic Inspiration, Bard level 1 to 4 | no change | regain all | "Bardic Inspiration", p. 32 | class-bard.md FR-038, FR-040 |
| R-5 | Bardic Inspiration, Bard level 5+ | regain all | regain all | "Level 5: Font of Inspiration", p. 32 | class-bard.md FR-039 |
| R-6 | Spell slots (Spellcasting) | no change | regain all expended | "Spellcasting" of Bard p. 32, Cleric p. 37, Druid p. 42, Paladin p. 54, Ranger p. 57, Sorcerer p. 65, Wizard p. 78 | spellcasting.md FR-025, FR-026 |
| R-7 | Pact Magic slots | regain all | regain all | "Level 1: Pact Magic", p. 71 | spellcasting.md FR-025, FR-026; class-warlock.md FR-028 |
| R-8 | Channel Divinity uses (Cleric, Paladin) | regain 1 | regain all | "Level 2: Channel Divinity", p. 37; "Level 3: Channel Divinity", p. 54 | class-cleric.md FR-042, FR-043; class-paladin.md FR-067 |
| R-9 | Divine Intervention | no change | available again | "Level 10: Divine Intervention", p. 38 | class-cleric.md |
| R-10 | Wild Shape uses | regain 1 | regain all | "Level 2: Wild Shape", p. 42 | class-druid.md FR-068, FR-069 |
| R-11 | Natural Recovery free cast of a Circle spell (once until a Long Rest) | no change (the slot recovery is window W-11) | available again | "Level 6: Natural Recovery", p. 46 | class-druid.md FR-125, FR-156 |
| R-12 | Second Wind uses | regain 1 | regain all | "Level 1: Second Wind", p. 48 | class-fighter.md FR-063 |
| R-13 | Action Surge uses | regain all | regain all | "Level 2: Action Surge", p. 48 | class-fighter.md FR-074 |
| R-14 | Indomitable uses | no change | regain all | "Level 9: Indomitable", p. 48 | class-fighter.md FR-083 |
| R-15 | Focus Points | regain all expended | regain all expended | "Level 2: Monk's Focus", p. 50 | class-monk.md FR-036, FR-124 |
| R-16 | Uncanny Metabolism | no change | available again | "Level 2: Uncanny Metabolism", p. 51 | class-monk.md FR-124 |
| R-17 | Wholeness of Body uses | no change | regain all | "Level 6: Wholeness of Body", p. 52 | class-monk.md FR-111 |
| R-18 | Lay On Hands pool | no change | replenished | "Level 1: Lay On Hands", p. 53 | class-paladin.md |
| R-19 | Paladin's Smite free cast | no change | available again | "Level 2: Paladin's Smite", p. 54 | class-paladin.md FR-064 |
| R-20 | Favored Enemy free casts | no change | regain all | "Level 1: Favored Enemy", p. 58 | class-ranger.md FR-036 |
| R-21, R-22 | Tireless uses; Nature's Veil uses | no change | regain all | "Level 10: Tireless", p. 59; "Level 14: Nature's Veil", p. 59 | class-ranger.md FR-075, FR-080 |
| R-23 | Tireless, Decrease Exhaustion | no recharge (the Short Rest reduction is window W-12) | lowers by 1 (the Long Rest rule, FR-029 here) | "Level 10: Tireless", p. 59 | class-ranger.md FR-076 (Short Rest); FR-029 here (Long Rest) |
| R-24 | Stroke of Luck | available again | available again | "Level 20: Stroke of Luck", p. 63 | class-rogue.md FR-088 |
| R-25 | Sorcery Points | no change (Sorcerous Restoration is window W-13; its once-until-a-Long-Rest limit is reset by the Long Rest) | regain all | "Level 2: Font of Magic", "Level 5: Sorcerous Restoration", p. 66 | class-sorcerer.md FR-083, FR-107 |
| R-26 | Innate Sorcery uses | no change | regain all | "Level 1: Innate Sorcery", p. 66 | class-sorcerer.md |
| R-27 | Magical Cunning, Contact Patron, Mystic Arcanum, Gift of the Depths, Dark One's Own Luck, Hurl Through Hell | no change | available again | pp. 72, 73, 76 | class-warlock.md FR-139, FR-140 |
| R-28 | Arcane Recovery (once until a Long Rest) | no change (the recovery is window W-14) | available again | "Level 1: Arcane Recovery", p. 78 | class-wizard.md FR-050 |
| R-29 | Signature Spells free casts | available again | available again | "Level 20: Signature Spells", p. 79 | class-wizard.md FR-074 |
| R-30 | Overchannel use count | no change | resets | "Level 14: Overchannel", p. 82 | class-wizard.md FR-090 |
| R-31 | Species uses equal to Proficiency Bonus: Breath Weapon, Stonecunning, Giant Ancestry; Draconic Flight; Large Form; Relentless Endurance | no change | regain all | "Dragonborn", "Dwarf", p. 84; "Goliath", p. 85, 86; "Orc", p. 86 | species-trait-behaviour.md FR-030 to FR-033 (use state, limits and renewal; content-thin-slice.md FR-013, FR-018, FR-020 hold the data) |
| R-32 | Orc Adrenaline Rush uses | regain all | regain all | "Orc", p. 86 | species-trait-behaviour.md FR-030 to FR-033 |
| R-33 | Epic Boon "Boon of Fate", Improve Fate | available again | available again | "Boon of Fate", p. 88 | feats-behaviour.md FR-050 (`FeatRenewal`: `ShortRestFinished`, `LongRestFinished`) |
| R-34 | Intimidating Presence (Path of the Berserker) | no change | available again (or restored by expending a use of Rage) | "Level 14: Intimidating Presence", p. 30 | class-barbarian.md FR-082 |
| R-35 | Wild Companion familiar | no change | the familiar disappears (`WildCompanionDismissed`) | "Level 2: Wild Companion", p. 43 | class-druid.md FR-090; summoned-and-companion-creatures.md FR-053 (`ReportLongRestFinished` departs it, cause `LongRestFinished`) |
| R-36, R-37 | Wild Resurgence (Wild Shape use for a level 1 slot); Nature Magician (Wild Shape uses for a slot) | no change | available again | "Level 5: Wild Resurgence", p. 43; "Level 20: Archdruid", p. 43 | class-druid.md FR-102, FR-113 |
| R-38, R-39 | Faithful Steed free cast of Find Steed; Holy Nimbus (Oath of Devotion) | no change | available again | "Level 5: Faithful Steed", p. 55; "Level 20: Holy Nimbus", p. 57 | class-paladin.md FR-062, FR-091 |
| R-40 | Spell slots created with Creating Spell Slots | no change | the created slots vanish | "Level 2: Font of Magic", Creating Spell Slots, p. 66 | class-sorcerer.md FR-054 |
| R-41, R-42 | Dragon Wings (Draconic Sorcery); Dragon Companion free cast of Summon Dragon | no change | available again | "Level 14: Dragon Wings", p. 70; "Level 18: Dragon Companion", p. 70 | class-sorcerer.md FR-093, FR-095 |
| R-43 | Gift of the Protectors (Eldritch Invocation) | no change | available again (no creature benefits until then) | "Gift of the Protectors", p. 73 | class-warlock.md FR-103 |
| R-44 | Elven Lineage spells (levels 3 and 5) free casts | no change | regain the free cast | "Elf", Elven Lineage, p. 84 | species-trait-behaviour.md FR-021, FR-033 (content-thin-slice.md FR-024 holds the data) |
| R-45 | Forest Gnome Speak with Animals uses (Proficiency Bonus) | no change | regain all | "Gnome", Gnomish Lineage, p. 85 | species-trait-behaviour.md FR-030 to FR-033 (content-species-backgrounds-feats.md FR-020 holds the data) |
| R-46 | Fiendish Legacy spells (levels 3 and 5) free casts | no change | regain the free cast | "Tiefling", Fiendish Legacy, p. 86 | species-trait-behaviour.md FR-021, FR-033 (content-species-backgrounds-feats.md FR-019 holds the data) |
| R-47 | Magic Initiate level 1 spell free cast | no change | regain the free cast | "Magic Initiate", p. 87 | feats-behaviour.md FR-032 |

**Table W. Windows opened by finishing a rest (FR-061).** These are not
recharges: "whenever you finish a ... Rest, you can ...". An interrupted Long Rest
(`source` `LongRestInterrupted`) opens none of them.

| Row | Window | Rest | Source | Owner spec |
|---|---|---|---|---|
| W-1 | Change prepared spells (Cleric, Druid, Wizard: any; Paladin, Ranger: replace one) | Long | pp. 37, 42, 54, 58, 78 | spellcasting.md FR-048, FR-049 |
| W-2 | Replace one Wizard cantrip; Spell Mastery swap | Long | "Spellcasting", p. 77; "Level 18: Spell Mastery", p. 79 | spellcasting.md FR-053; class-wizard.md FR-069 |
| W-3 | Memorize Spell | Short | "Level 5: Memorize Spell", p. 79 | spellcasting.md FR-091; class-wizard.md FR-061 |
| W-4 | Weapon Mastery swap (Barbarian, Fighter, Paladin, Ranger, Rogue) | Long | pp. 29, 48, 54, 59, 62 | weapon-properties-and-mastery.md FR-024; class-fighter.md FR-046, FR-047 |
| W-5 | Fiendish Resilience damage type | Short or Long | "Level 10: Fiendish Resilience", p. 76 | class-warlock.md FR-131, FR-133 |
| W-6 | Book of Shadows | Short or Long | "Pact of the Tome", p. 74 | class-warlock.md FR-097 |
| W-7 | Hunter's Prey, Defensive Tactics option | Short or Long | p. 61 | class-ranger.md |
| W-8 | Known form replacement; land type choice | Long | "Wild Shape", p. 42; "Circle of the Land Spells", p. 46 | class-druid.md FR-153, FR-156 |
| W-9 | High Elf cantrip replacement | Long | "Elf", p. 85 | species-trait-behaviour.md FR-025, FR-033 |
| W-10 | Resourceful: gain Heroic Inspiration | Long | "Human", p. 86 | species-trait-behaviour.md FR-045, FR-033; heroic-inspiration.md (`shared-types-and-ownership.md` FR-048); character-aggregate.md FR-089 (the `HeroicInspiration` state) |
| W-11 | Natural Recovery: recover spell slots (once until a Long Rest) | Short | "Level 6: Natural Recovery", p. 46 | class-druid.md FR-127, FR-153 |
| W-12 | Tireless: Decrease Exhaustion (reduce Exhaustion by 1) | Short | "Level 10: Tireless", p. 59 | class-ranger.md FR-076 |
| W-13 | Sorcerous Restoration: regain Sorcery Points (once until a Long Rest) | Short | "Level 5: Sorcerous Restoration", p. 66 | class-sorcerer.md FR-083, FR-107 |
| W-14 | Arcane Recovery: recover spell slots (once until a Long Rest) | Short | "Level 1: Arcane Recovery", p. 78 | class-wizard.md FR-050 |

### 4.9 Withdrawn ids

None. (`PerDay` and `RestRechargeWording` were removed: "Per Day" is data with trigger `LongRest`, FR-032.)

## 5. Model

### 5.1 Types

Go forms. Every type is in package `character`. Rows write variant data as `Name{Field, ...}`; in Go each variant is a struct with unexported fields, a `NewX` constructor where it validates, and read-only accessors named for the fields (`RestApplied.BenefitsOf()`), so values are immutable and compared with `==` or `Equal`. Every closed variant set is an interface with an unexported marker method whose doc comment names every variant; switches over it list every variant. Collections are copied on construction and on read. Enum zero values are invalid unless stated.

| Part | Kind | Holds |
|---|---|---|
| `RestKind` | enum | `ShortRest`, `LongRest` (typed `int` with `String()`; the zero value is invalid, `Valid()`). Methods `MinimumHours(LongRestVariant)` (FR-002 to FR-004) and `Recharges(content.RechargeTrigger)` (FR-032, FR-065). |
| `LongRestVariant`, `RestInterruption`, `RestSource` | enums | `StandardLongRest`, `TranceLongRest`; the four values of FR-005 with `Stops(RestKind)` (FR-006); `RestFinished`, `LongRestInterrupted` (FR-036, FR-061). |
| `LongRestSpacing` | closed variant set | `NoLongRestFinishedYet`, `LongRestFinishedHoursAgo` (built by `NewLongRestFinishedHoursAgo(hours) (LongRestFinishedHoursAgo, error)`, hours >= 0); start-of-rest value (FR-070, FR-072). Interface with an unexported marker method; the doc comment lists both variants. A nil `LongRestSpacing` is treated as `NoLongRestFinishedYet`. |
| `CheckRestStart` | domain function | `CheckRestStart(kind, currentHitPoints, exhaustionLevel, spacing) (RestStartDecision, error)`: the error satisfies `errors.Is(err, srderr.ErrInvalidInput)` for an input outside FR-013; otherwise a closed `RestStartDecision` (`RestMayStart` or `RestMayNotStart{Blockers}`), each blocker a closed `RestStartBlocker`: `RestStartNoHitPoints`, `RestStartExhaustionLevelSix`, `RestStartTooSoonAfterLongRest{HoursStillToWait}` (FR-007 to FR-012). |
| `RestEnd` | closed variant set | `RestRanItsCourse{HoursRested, InterruptionsResumed}`, `RestStopped{Interruption, HoursRestedBefore, InterruptionsResumed}`; interface with an unexported marker method, doc comment lists both; hours are totals (FR-067). |
| `RestReport` | value object | kind, `LongRestVariant`, `RestEnd`. Built by `NewRestReport(kind, variant, end) (RestReport, error)`; the error joins every `RestReportProblem` (FR-066). The zero `RestReport` is invalid and `ResolveRest` returns an `srderr.ErrInvalidInput` error for it. |
| `RestResolution` | closed variant set | `RestCompleted{Kind}`, `RestIncomplete{HoursStillNeeded}`, `RestStoppedWithShortRestBenefits{Interruption}`, `RestStoppedWithoutBenefits{Interruption}`, `RestReportRejected{Problem RestReportRejection}`. `ResolveRest(report) (RestResolution, error)` is the domain function; the error is only for an invalid (zero) report. |
| `RestReportRejection`, `RestBenefitsGranted` | closed variant sets | `ShortRestCannotBeResumed` (FR-017); `RestBenefitsOfShortRest`, `RestBenefitsNone` (FR-036). |
| `RestReportProblem`, `TakeRestCommandProblem`, `SpentHitDiceProblem`, `SpendHitPointDieCommandProblem` | enums (each implements `error` and matches `srderr.ErrInvalidInput`) | `NegativeHoursRested`, `NegativeHoursRestedBefore`, `NegativeInterruptionsResumed` (FR-066); `NegativeCurrentHitPoints`, `ExhaustionLevelOutOfRange` (FR-066); `CountBelowOne` (FR-073); `SpentCountBelowOne` (FR-073). Returned joined by `errors.Join`. |
| `RestBenefits` | value object | `NewRestBenefits(benefitsOf, exhaustionLevel, spentHitDice) (RestBenefits, error)` returns the ordered effect list (`Effects()`, a copy) and the new `SpentHitDice` (`SpentHitDice()`) (FR-024 to FR-035, FR-064); the error satisfies `errors.Is(err, srderr.ErrInvalidInput)` for an Exhaustion level outside 0 to 6. It takes no pool: a Long Rest clears every spent die. |
| `RestEffect` | closed variant set | `SpendHitPointDiceAvailable`, `AllLostHitPointsRegained`, `HitPointDiceRegained{Count}`, `HitPointMaximumReturnedToNormal`, `AbilityScoresReturnedToNormal`, `ExhaustionLevelReduced{From, To}`, `TemporaryHitPointsEndedByRest`, `FeaturesRecharged{Kind}`. Interface with an unexported marker method; the doc comment lists all eight. |
| `SpentHitDice` | value object, implements `FeatureState` (`character-aggregate.md` FR-092) | counts by `content.HitPointDie` (FR-037, FR-038); the zero value is the empty set; `NewSpentHitDice(map[content.HitPointDie]int) (SpentHitDice, error)` (FR-073); `Spend(HitDicePool, HitPointDie, RestKind benefitsOf)` and `SpendOutsideRest(HitDicePool, HitPointDie)` (FR-069) return a closed `HitDieSpendResult`: `HitDieSpendAllowed{Spent}` or `HitDieSpendRejected{Reason}`. |
| `HitDieSpendRejection` | enum | The `reason` of `HitDieSpendRejected` and `SpendHitPointDieRejected`: `NoneRemaining`, `NotInPool`, `NotAShortRestBenefit` (`Spend` only), `SpentExceedsPool`. |
| `SpentHitDiceKey` | `FeatureStateKey` for `SpentHitDice` | Name `spent_hit_dice` (`character-aggregate.md` FR-093), defined here as `shared-types-and-ownership.md` FR-022 asks; the functions here take `SpentHitDice` as an argument and never read the store. |
| `HitPointsRegained` | domain function | `HitPointsRegained(die, roll, constitutionModifier) (int, error)` (FR-043, FR-044); the error is a `*RollRangeError` matching `srderr.ErrInvalidInput`. |
| `RestEvent` | closed event family | interface embedding `character.Event` (`character-aggregate.md` FR-085) with an unexported marker; `CharacterID()` on the interface (FR-068). Variants: `RestApplied{CharacterID, BenefitsOf, Source, Effects}`, `RestInterrupted{CharacterID, Kind, Interruption, BenefitsGranted}`, `HitPointDieSpent{CharacterID, Die, Roll, ConstitutionModifier, HitPointsRegained}`. Past tense, immutable. |

### 5.2 Invariants and results

Constructors and domain functions return an error satisfying
`errors.Is(err, srderr.ErrInvalidInput)` for invalid input: a `SpentHitDice` count below 1, a negative hours,
interruption count or current Hit Points, an Exhaustion level outside 0 to 6, a die roll outside its
die (`HitPointsRegained`, FR-044). `MustX` helpers panic with that error and are for test literals only. Collections exposed by values are copies (FR-048). Host input is validated by the
`NewX` constructors, which return every problem (FR-066, FR-073). A spent
count above the pool's is a returned `SpentExceedsPool` from `Spend`,
`SpendOutsideRest` and the `SpendHitPointDie` use case, not from
`NewRestBenefits`, which has no pool (FR-038). Start-of-rest values on
`TakeRestCommand` are a documented contract, not checkable (FR-070). A resumed
Short Rest is a `ResolveRest` result (FR-017), not a construction
failure.

### 5.3 Results and errors

Expected outcomes (not started, incomplete, stopped, rejected report,
rejected spend) are results; invalid host input is an error from the
`NewX` constructors; the only infrastructure failure is a failure of the
`dice.Roller` port, returned by `SpendHitPointDie` as a `*srderr.PortError`
(`srderr.ErrorType` gives `dice_roller_error`), matched with `errors.Is` / `errors.As`.

### 5.4 Deferrals satisfied and mechanics superseded

Other specs left these to a rest spec (or wrongly to `spellcasting.md`); each
owner reconciles its own text (Q-1, Q-14; `shared-types-and-ownership.md`
FR-048). Rows naming `combat` specs are non-normative hand-offs: nothing here
depends on them (`shared-types-and-ownership.md` FR-013).

| Spec | Left to a rest spec | Satisfied by here |
|---|---|---|
| shared-types-and-ownership.md FR-009, FR-016, FR-020 | `RestKind`, rest effects and `RestApplied` owned here; `character` maps `RestKind` to triggers | FR-001, FR-032, FR-036, FR-059, FR-065 |
| spellcasting.md (FR-025, FR-026, FR-048, FR-091) | A rest "arrives as a `RestKind`" | FR-001 to FR-006, FR-014 to FR-022, FR-060; R-6, R-7, W-1 to W-3. Slot recovery is not superseded. The telemetry key `srd5e.character.rest_kind` is defined here (section 7); `spellcasting.md` reuses it |
| hit-points-and-damage.md FR-034; conditions.md FR-036, FR-043, D-15; encounter-aggregate.md | A Long Rest ends Temporary Hit Points and lowers Exhaustion by 1 | FR-030 and FR-029 (rules and triggers); the operations stay there. FR-007, FR-009, FR-058 |
| character-aggregate.md; derived-stats.md FR-028 | Spending Hit Dice and the spent state | FR-037 to FR-048; Q-6 |
| class-fighter.md (`TakeRest`, `LongRestFinished`) | A `TakeRest` applying recovery to all class states | FR-049 to FR-052, FR-036. Classes are downstream (`shared-types-and-ownership.md` FR-016): the host calls each class function with `RestApplied.BenefitsOf()` (Q-1); `LongRestFinished` is `RestApplied(LongRest)` (FR-061) |
| class-barbarian, -bard, -cleric, -druid, -monk, -paladin, -ranger, -rogue, -sorcerer, -wizard (rest-trigger text, `RestKind.short`, a result variant `RestApplied`) | Own rest triggers, wording and `RestKind` ownership, including an interrupted Long Rest | FR-001, FR-020, FR-059 to FR-061 (Q-2); `InitiativeRolled` stays Bard-owned; R-4 to R-6; they define no rest event (`shared-types-and-ownership.md` FR-020) |
| class-warlock.md (FR-028, FR-078, FR-139, FR-140) | Rest and Hit Die rules; Lifedrinker's spent die | FR-001 to FR-048; FR-069 (taken up by FR-078, Q-22) |
| class-ranger.md FR-076 | Long Rest lowering of Tireless | FR-029, R-23, W-12 |
| content-thin-slice.md, content-species-backgrounds-feats.md, species-trait-behaviour.md | Species recharge: data in content, use state and windows in species-trait-behaviour.md | FR-004, FR-032, FR-065; R-31, R-32, R-44 to R-46; W-9, W-10 |
| revival-and-restoration-spells.md (FR-016 to FR-022) | Raise Dead penalty and Resurrection strain end at a finished Long Rest | FR-036, FR-061: `AfterRest` takes `RestApplied.BenefitsOf()`; an interrupted Long Rest gives Short Rest benefits and reduces nothing |
| content-monsters.md FR-035 | A monster part recharges after "a Short or Long Rest" | Not a dependency; monsters are out of scope (Q-8) |
| magic-item-rules.md (`ShortRestFocus`; FR-041, FR-114; Q3) | A parallel Short Rest completion vocabulary | Keeps `ShortRestFocus` and `RestPurpose` (host facts about one item); its FR-114 builds the focus from `RestResolution`: `IsCompleted` is true for `RestCompleted(ShortRest)` and false for `RestStoppedWithoutBenefits` of a Short Rest (FR-018, FR-019) or `RestIncomplete`; a Long Rest, even interrupted with Short Rest benefits (FR-020), is `NotAShortRest` there, agreeing with FR-061 (Q-19) |
| feats-behaviour.md (`FeatRenewal`, `RenewFeatUses`; FR-032, FR-050 to FR-052; Q9) | Renews on `ShortRestFinished` and `LongRestFinished`; who calls it | FR-060: `ShortRestFinished` from `RestApplied.BenefitsOf() == ShortRest` (any `RestSource`, renewal being a recharge, FR-020) and `LongRestFinished` from `LongRest`; the host calls `RenewFeatUses` after each `TakeRestApplied` (Q-1); R-33, R-47 |
| exploration.md (`MountedPaceBoost.restFinished(RestKind)`; FR-022 to FR-024) | Restores a mount boost after "a Short or Long Rest" | FR-001, FR-060: either `RestApplied.BenefitsOf()` restores it; an unfinished or refused rest is not reported |
| toolbox-curses-and-contagions.md (`ContagionSewerPlagueRestRules`; FR-082, FR-083) | A hook for a condition that changes rest benefits | None needed: its filter takes the `RestBenefits` effects and a Hit Die result; the names in FR-025, FR-026, FR-029 are stable |
| weapon-properties-and-mastery.md FR-024; magic-item-rules.md FR-041 | The Long Rest moment for a Weapon Mastery change; whether an interrupted rest counts for attunement | W-4 (FR-061); FR-018 to FR-020, FR-061 (the attunement consequence stays there) |

## 6. Interface

Both use cases are structs built by a constructor that takes the context's functional options (`WithTelemetry`, `WithClock`; telemetry defaults to no-ops, `character-aggregate.md`), with one method that takes
`ctx` first, because it emits a span, and an immutable command value. Both are synchronous (no goroutines) and neither calls another use case (`shared-types-and-ownership.md` FR-044).

| Use case | Command | Result (closed `<UseCase>Result`, `shared-types-and-ownership.md` FR-017) | Ports |
|---|---|---|---|
| `TakeRest` (`RestTaker.TakeRest(ctx, cmd) TakeRestResult`; `NewRestTaker(opts ...Option)`) | `TakeRestCommand`: `CharacterID`, `RestReport`, `CurrentHitPoints` (int), `ExhaustionLevel` (int 0 to 6), `LongRestSpacing`, `SpentHitDice`; the three start values are as at the start of the rest (FR-070); built by `NewTakeRestCommand(...) (TakeRestCommand, error)` (FR-066) | `TakeRestResult`: `TakeRestApplied{SpentHitDice, Applied RestApplied}`, `TakeRestAppliedAfterInterruption{SpentHitDice, Applied RestApplied, Interrupted RestInterrupted}` (effects are `Applied.Effects()`, FR-049), `TakeRestNotStarted{Blockers}`, `TakeRestIncomplete{HoursStillNeeded}`, `TakeRestStopped{Interrupted RestInterrupted}`, `TakeRestRejected{Problem RestReportRejection}`; interface `TakeRestResult` with an unexported marker, doc comment lists all six | `telemetry.Telemetry` only |
| `SpendHitPointDie` (`HitDieSpender.SpendHitPointDie(ctx, cmd) (SpendHitPointDieResult, error)`; `NewHitDieSpender(roller, opts ...Option) (HitDieSpender, error)`; the error is for a nil roller) | `SpendHitPointDieCommand`: `CharacterID`, `HitDicePool`, `SpentHitDice`, `HitPointDie`, `AbilityModifier` (Constitution), `RestKind` (the benefits being claimed); `NewSpendHitPointDieCommand` takes the host's spent counts as `map[content.HitPointDie]int` and returns `(SpendHitPointDieCommand, error)` (FR-073) | `SpendHitPointDieResult`: `SpendHitPointDieApplied{Roll, ConstitutionModifier, HitPointsRegained, SpentHitDice, HitPointDieSpent}`, `SpendHitPointDieRejected{Reason HitDieSpendRejection}`; interface `SpendHitPointDieResult`, doc comment lists both | the consumer-side one-method `Roller` (`Roll(dice.Expression) (dice.Result, error)`, satisfied by `dice.Roller`), `telemetry.Telemetry` |

`RestInterrupted` is carried by `TakeRestAppliedAfterInterruption` and `TakeRestStopped`.

- **Port failure contract:** the roller returns a result or an error; the use case calls it through `dice.Roll`, so a returned error, a panic and a contract-breaking result all come back as a `*srderr.PortError` (`Port` is `dice.Port`), which `SpendHitPointDie` returns unchanged with the zero result (FR-055). No other port fails.
- **Exports.** Package `character` exports every type of 5.1, the two use cases with their commands and results (including `TakeRestApplied`, `TakeRestAppliedAfterInterruption`) and the domain functions (`CheckRestStart`, `ResolveRest`, `NewRestBenefits`, `HitPointsRegained`). Named variants: `HitDieSpendAllowed`, `HitDieSpendRejected`. `HitDicePool`, `CharacterID` and `AbilityModifier` are exported by their owners, `HitPointDie` by `content`. FR-059 applies to every name; the package doc comment lists each command, result and event. No facade or re-export exists.

## 7. `telemetry.Telemetry`

Per `doc/rules/observability.md` section 2: the host may never report a rest,
may report too early or inconsistently, the `dice.Roller` may fail, or stored
spent dice may exceed the pool. Rule results are values of
`srd5e.character.result` on the counter and span; a roller failure is a returned
`*srderr.PortError` with `error.type`; corrupt stored state is a returned
rejection (`spent_exceeds_pool`, FR-073). Counted: rests by kind and result,
spends by die size and result, Hit Points per spend, returned port failures, duration.
`srd5e.character.operations` and `.duration` (`character-aggregate.md`) are
not reused. This spec defines `srd5e.character.rest_kind` (values `short`,
`long`, the snake_case of `RestKind`); `spellcasting.md` reuses it. To
diagnose: the rest kind, result, first blocker or rejection reason and, for
spends, the die size are span and counter attributes and are in the returned
result. Alert on a rise in `not_started`, `rests` flat at zero while spells
are cast, and any `error.type` on `hit_point_dice_spent`. No logs: spans and
counters carry every result; one event per rest; each failure is counted once.

**Attribute values (errors-and-results.md FR-043, shared-types-and-ownership.md
FR-041).** `srd5e.outcome` is `success` or `failure`. `success` means the
operation did what was asked; every other result is a rule outcome that
conferred nothing, recorded as `failure` with the variant in
`srd5e.character.result` (as `character-aggregate.md` FR-072 does).

| Operation | `srd5e.character.result` | `srd5e.outcome` |
|---|---|---|
| `take_rest` | `applied`, `applied_after_interruption` | `success` |
| `take_rest` | `stopped`, `incomplete`, `not_started`, `rejected` | `failure` |
| `spend_hit_point_die` | `spent` | `success` |
| `spend_hit_point_die` | `rejected` | `failure` |
| `spend_hit_point_die` | (returned `dice_roller_error`; result absent) | `failure`, with `error.type` `dice_roller_error` |

A returned rule result leaves span status not error and `error.type` absent (observability.md section 6); only a returned `*srderr.PortError` sets span error status (an `srderr.ErrInvalidInput` from a constructor happens before any use case runs and is not counted). `TakeRest` calls no port, so it returns no error.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.take_rest` | span | `srd5e.outcome`, `srd5e.character.result` (`applied`, `applied_after_interruption`, `stopped`, `incomplete`, `not_started`, `rejected`), `srd5e.character.rest_kind` (`short`, `long`; the kind reported; defined here), `srd5e.character.rejection` (`not_started` and `rejected` only: the first blocker or the problem, snake_case) | Trace one `TakeRest` | Error status above 0 means a bug (`TakeRest` has no failing port) |
| `srd5e.character.spend_hit_point_die` | span | `srd5e.outcome`, `srd5e.character.result` (`spent`, `rejected`), `srd5e.character.hit_die_faces` (`6`, `8`, `10`, `12`), `srd5e.character.rejection` (rejections only: `none_remaining`, `not_in_pool`, `not_a_short_rest_benefit`, `spent_exceeds_pool`), `error.type` (returned `dice_roller_error` only) | Trace one spend, including the single roll | Error status above 0 |
| `srd5e.character.rests` | counter, `{rest}` | `srd5e.character.rest_kind`, `srd5e.outcome`, `srd5e.character.result`, `srd5e.character.rest_benefits` (`short`, `long`, `none`), `srd5e.character.rejection` (`not_started` and `rejected` only) | Rest traffic and the share of interrupted or refused rests (the only `TakeRest` counter) | `not_started` above a baseline share; a zero rate beside spell casts |
| `srd5e.character.hit_point_dice_spent` | counter, `{spend}` | `srd5e.character.hit_die_faces`, `srd5e.outcome`, `srd5e.character.result` (absent when a port failure left the use case), `srd5e.character.rejection` (rejections only), `error.type` (port failures only) | One increment per `SpendHitPointDie` call (the only spend counter, so a failure is counted once) | Any `error.type`; compare `spent` with `rests` of kind `short` |
| `srd5e.character.hit_die_hit_points` | histogram, `{hit_point}` | `srd5e.character.hit_die_faces` | Hit Points regained per spent die, including the minimum of 1 | Dashboard; the mean tracks half the faces plus one plus the Constitution modifier |
| `srd5e.character.rest_operation_duration` | histogram, `s` | `srd5e.character.rest_operation` (`take_rest`, `spend_hit_point_die`) only: attributes are fixed before the timed body runs (`observability-ports.md` FR-029, FR-031) | Latency of each use case, via the `Meter` timing operation | p99 regression per operation |

Entries here feed `doc/observability/signals.md`; names and keys are constants defined once in the context (observability.md section 5). The domain (`CheckRestStart`, `RestResolution`, `RestBenefits`, `SpentHitDice`, `HitDieRecovery`) is **silent by design**; the use cases translate its values into signals.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-006, FR-065 | unit | `rest_kind_test.go` (package `character_test`): values, minimum hours per variant, interruption set per rest, table 4.8a rows SI-1 to SI-5; table 4.8f (all six cells) |
| FR-009, FR-072 | unit | `long_rest_spacing_test.go`: table 4.8g rows SC-1 to SC-6 |
| FR-007 to FR-013 | unit | `rest_start_check_test.go`: Hit Points 0 and 1; spacing 15, 16, 17 hours, none, and a Short Rest straight after a Long Rest; Exhaustion level 6; every blocker together and their order; `HoursStillToWait` at 1 and 15 hours since; negative and out-of-range inputs return errors matching `srderr.ErrInvalidInput` |
| FR-014 to FR-023, FR-066, FR-067, FR-071 | unit | `rest_resolution_test.go`: tables 4.8a (all rows) and 4.8b (all rows); `NewRestReport` and `NewTakeRestCommand` return every problem once each (`errors.Is` per problem value and against `srderr.ErrInvalidInput`), table-driven row by row; a resumed Short Rest is a `ResolveRest` result, not a construction failure; `FuzzNewRestReport` never panics |
| FR-024 to FR-036, FR-064, FR-068 | unit | `rest_benefits_test.go`: table 4.8e rows LB-1 to LB-6, SB-1, SB-2; effect order; no effects for stopped, incomplete and rejected rests; `NewRestBenefits(ShortRest, ...)` without a report equals FR-024 and raises no event; event contents (`RestInterrupted` only for the two stopped results: LI-11 to LI-13 and LI-15 raise `RestApplied` only); every `RestEvent` is a `character.Event` and a type switch over `RestEvent` lists every variant; events of two characters are distinguishable |
| FR-037 to FR-048, FR-069, FR-073 | unit | `spent_hit_dice_test.go`, `hit_points_regained_test.go`: tables 4.8c and 4.8d (SP-1 to SP-15, both `NewX` constructors); a corrupt state returns `SpentExceedsPool` and never errors or panics; returned maps and slices are copies; inputs not altered; `SpentHitDice` stored and read back under `SpentHitDiceKey` through `FeatureStates` |
| FR-049 to FR-052, FR-058, FR-070 | application | `take_rest_usecase_test.go`: every result variant (including `TakeRestAppliedAfterInterruption`); table 4.8a LI-10; no port other than telemetry; state returned unchanged on non-applied outcomes |
| FR-053 to FR-056 | application | `spend_hit_point_die_usecase_test.go`: `dicetest.NewFixedRoller` records no call (`Requests()` empty) on a rejection and exactly one call (one die, no modifier) on an allowed spend; extremes; a `dicetest.CallbackRoller` returning an error gives a `*srderr.PortError` returned unchanged (`errors.Is` with the cause) and the state unchanged; a roller answering a different expression (count, die, modifier, one case each) gives a `*srderr.PortError` matching `srderr.ErrContractViolation`, returned unchanged; a panicking roller gives one matching `srderr.ErrPortPanic` |
| FR-057 | application | recording (`telemetrytest`) and no-op `telemetry.Telemetry` give equal results; neither use case calls another (`shared-types-and-ownership.md` FR-044); the signals of section 7 are asserted (spans, counter attributes, no log; `srd5e.outcome` only `success` or `failure`; `error.type` only `dice_roller_error`; a port failure gives `failure`, `error.type` set, no `srd5e.character.result`, counted once in `hit_point_dice_spent`); a cancelled `ctx` is not recorded as a failure outcome (`errors-and-results.md` FR-054); `errors-and-results.md` FR-031, FR-034, FR-051 rows apply to `SpendHitPointDie`, and `TakeRest` calls no port |
| FR-059, FR-062 | architecture | the duplicate-name scan (`internal/archtest`) finds each exported name of sections 5.1 and 6 exactly once (`RestKind`, `RestApplied`, `TemporaryHitPointsEndedByRest` included), no `RestKind` member `short` or `long`; no logging, metric or span call, `context`, goroutine, clock or randomness in the domain files |
| FR-060, FR-061 | scenario | `scenarios/rests_test.go` (package `scenarios_test`), public API only: Table R rows through each owner's function for both `RestKind` values (one table-driven test per row); Table W rows accept only their named rest and `source` (an interrupted Long Rest opens no Short Rest window, W-3 and W-11 to W-14 included, but recharges Short Rest features); `FeatRenewal` and `MountedPaceBoost` map from `RestApplied.BenefitsOf()`. Cites the owner FR in the row |
| FR-063 | doc review | the doc comments of `RestReport` and `TakeRestCommand` state the sleep, light-activity and Short Rest activity preconditions (no clock in the package, so not testable in code) |

- **Doubles:** `dicetest.NewFixedRoller` and `dicetest.CallbackRoller` (`dice/dicetest`) and the recording telemetry of `telemetry/telemetrytest` (`shared-types-and-ownership.md` FR-038); no new port or contract suite.
- **Mutation testing:** the rest and Hit Dice domain files of package `character` MUST score at least 80 (testing.md section 8; `gomu` runs only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`, and is not run by this spec); likely survivors are the boundaries 0 and 1 Hit Point, 15, 16, 17 hours, 0 and 1 hour rested, required hours minus one, and the minimum of 1 Hit Point regained. The `*_usecase.go` files SHOULD be mutated.

## 9. Open questions and decisions

**Settled** (the decision, not a question; none is escalated):

- Q-1: no use case calls another and class logic is pure (`shared-types-and-ownership.md` FR-044, FR-023): the host calls class functions, spell slot restoration and `RenewFeatUses` with `RestApplied.BenefitsOf()`. Q-2: values `ShortRest`, `LongRest`; `RestApplied` is this event; class specs define no rest event (`shared-types-and-ownership.md` FR-009, FR-017, FR-020). Q-18: `RestEvent` implements `CharacterEvent` (`shared-types-and-ownership.md` FR-019; `character-aggregate.md` FR-085; FR-068).
- Q-3: no Hit Die spend claiming Long Rest benefits (FR-045); a completed Long Rest has regained every die. Q-11: a Short Rest cannot be resumed (FR-017). Q-10: Exhaustion 6 cannot start a rest (FR-011) and `PhysicalExertion` stops a Short Rest, accepted not rejected (FR-006, FR-019). All readings of the text.
- Q-5: the pool is `HitDicePool` keyed by `HitPointDie` (`shared-types-and-ownership.md` FR-008; `derived-stats.md` FR-061); absent key means 0; a published `CountOf` would replace it in FR-038 to FR-042. Q-20: `HitPointDie` is owned by `content-schema-and-decoder.md` FR-132 (`shared-types-and-ownership.md` FR-053); the roll is `dice.NewExpression(1, d, 0)` (with `d` the `dice.Die` of `die.Faces()` sides) (FR-054). Q-6: `SpentHitDice` lives in `FeatureStates` under `SpentHitDiceKey` (`shared-types-and-ownership.md` FR-022; `character-aggregate.md` FR-092, FR-093).
- Q-8: a Long Rest is not also a Short Rest (FR-032); mapping `RestKind.Recharges(content.RechargeTrigger)` (FR-065, `shared-types-and-ownership.md` FR-009); monster rests are out of scope (`content-monsters.md` FR-035). Q-13: the host owns time (`hit-points-and-damage.md` Q-12). Q-14: each owner reconciles its own deferral (5.4; `shared-types-and-ownership.md` FR-048). Q-19: `magic-item-rules.md` Q3 and `feats-behaviour.md` Q9 are answered in 5.4; those specs still edit their own text.
- Q-4: `character` may not import `combat` (`shared-types-and-ownership.md` FR-013), so the Exhaustion level is a plain `int` 0 to 6 here and in `class-ranger.md`; the host converts to `ExhaustionLevel` at the `combat` boundary. Permanent design, not pending: an upstream level type would be a mechanical edit. Q-7: Knocking Out a Creature (SRD 5.2.1, p. 17) and sleeping Unconscious in a Long Rest (p. 185) are conditions and Hit Point rules owned by `conditions.md` and `hit-points-and-damage.md`; this spec supplies a Short Rest startable at 1 Hit Point (FR-007) and `RestApplied(ShortRest)` as the trigger. Q-9: dehydration and malnutrition Exhaustion "can't be removed" until the daily amount is taken (pp. 181, 185); `environment-hazards-and-objects.md` keeps that lock in `HazardRecord` and `RemoveExhaustionLevels` (FR-060 to FR-065), so the host routes `ExhaustionLevelReduced` through it (FR-029). Q-15: `HitPointMaximumReturnedToNormal` and `AbilityScoresReturnedToNormal` are effects the host applies (interpretation: the SRD says only that they "return to normal", p. 185, and a rest needs no stored original); owner operations are requested as amendments, none is a dependency. Q-22: `class-warlock.md` FR-078 already takes `SpentHitDice` and calls `SpendOutsideRest` (FR-069).
- Derived: Q-12 Trance plus resumption is 4 plus one per interruption (FR-016, RH-10). Q-16 an interrupted Long Rest recharges features and allows Hit Die spends but opens no "finish a Short Rest" window (`source` `LongRestInterrupted`, FR-061; a host with the opposite reading treats it as `RestFinished`). Q-17 hours are cumulative totals floored by the host (FR-067). Q-21 a Long Rest stopped after the required hours is `RestCompleted(LongRest)` (FR-071), so 9 hours never confers less than 8; a Short Rest stays no-benefits (FR-018).

## 10. Changelog impact

`CHANGELOG.md`, under `Unreleased`, **Added**: "Short Rest and Long Rest
(`RestKind`, interruption, what each restores, `TakeRest`), Hit Point Dice
spending (`SpentHitDice`, `SpendHitPointDie`) and the `RestApplied` event, in
the `character` context; telemetry `srd5e.character.take_rest`,
`srd5e.character.spend_hit_point_die`, the `rests`, `hit_point_dice_spent`,
`hit_die_hit_points` and `rest_operation_duration` metrics."
Not **BREAKING**: new API in an unreleased package. Specs that renamed their own
rest types (Q-2) change documentation only.
