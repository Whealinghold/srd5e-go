# Toolbox hazard effects (the `toolbox` context and the shared save-gated effect model)

| | |
|---|---|
| Status | draft |
| Context | `toolbox` |
| Roadmap phase | 5 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `ability-scores.md`, `observability-ports.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md` |
| Referenced, not imported | [resolve-d20-test-use-case.md](resolve-d20-test-use-case.md) (the host rolls the save with it; no `toolbox` use case calls it, shared-types-and-ownership.md FR-044); [rule-options.md](rule-options.md) (the option pattern); [architecture-guardrails.md](architecture-guardrails.md) (rank 90, owns the import, cycle, leaf-context and use-case-call checks: FR-018, FR-019, FR-020, FR-021, FR-097, FR-098) |
| Supersedes | The generic save-then-effect mechanics duplicated in the sibling `toolbox` specs (section 5.2); the siblings keep their catalogues and topic rules. Already withdrawn in siblings: `toolbox-environmental-effects.md` rows 006, 007, 008, 031 (exemption order; replaced by FR-024 to FR-027), `toolbox-curses-and-contagions.md` rows 057, 059 (onset check; replaced by FR-048, FR-063), `toolbox-poison.md` rows 053, 054 (duration expiry; replaced by FR-054, FR-064). No spec outside `toolbox` is superseded. |

Binding decisions: shared-types-and-ownership.md FR-013, FR-017, FR-018, FR-019, FR-038, FR-041, FR-044, FR-045, FR-046, FR-048.

## 1. Purpose and scope

1. **Bootstrap the `toolbox` context.** SRD 5.2.1 "Gameplay Toolbox" (p. 192) holds rules that fit no existing context (travel, curses and contagions, environmental effects, fear and mental stress, poison, traps, encounter budgets). Many are "make a saving throw, then take damage and/or gain a condition or Exhaustion". Applying them uses `combat` types and `character` cannot import `combat`, so one context, `toolbox`, sits downstream of `content`, `d20`, `character` and `combat`, with nothing depending on it (shared-types-and-ownership.md FR-045). This spec states its shape and map row (FR-001, FR-002, FR-005, FR-006); the generic checks are architecture-guardrails.md's.
2. **Specify the shared save-gated effect model** the other `toolbox` specs build on: save ability and DC; optional Disadvantage; automatic success for Resistance or Immunity; failed-save damage, conditions and Exhaustion; optional half damage on a success; a duration; repeat saves that end the effect after a number of successes, or whose success reduces Exhaustion until none is left; delayed onset; an effect that applies with no save; damage that cannot be healed.
3. **Own `ToolboxRuleOptions`** (section 4.9), moved from `rule-options.md` by shared-types-and-ownership.md FR-057.

The engine is state in, new state and events out (`doc/rules/architecture.md` §4). It rolls the dice a rule asks for, decides who must save and what the outcome is, and returns structured data. No use case here calls another (shared-types-and-ownership.md FR-044): the two that decide a save (`ResolveSaveGatedEffect`, `ResolveRepeatSave`) share the pure `SaveGatedPlanner` and `SaveGatedEvaluator`, and the other two use the evaluator's ending rules and `DiceRequired`. The host rolls the saving throw through `d20` and applies the returned damage, conditions and Exhaustion through the combat use cases. The engine reads no clock.

**One owner.** Sibling specs build `SaveGatedEffect` values and cite types here by name and FR id; they never define a second effect model. A missing expressiveness is added by amending this spec (section 9 lists what was settled as a sibling's own concern). The planner is the only place that decides (a) automatic success from Resistance or Immunity to a listed damage type and (b) the Disadvantage from a `SaveGatedCircumstance` (FR-024 to FR-032). Facts that are not damage affinities (the Huge exemption of the Falling Net, p. 199; environmental adaptation, p. 195) arrive as `SaveGatedSubject.exemption` (FR-076); this spec never decides why.

**Out of scope:**

- The named catalogues (content kinds, `content-schema-and-decoder.md` FR-129): 14 sample poisons (pp. 197-198), 8 example traps (pp. 199-201), environmental effects (p. 195), three contagions (p. 194), Fear Effects and Mental Stress (p. 196); owned by the `toolbox-*.md` siblings. Section 5.1 shows each is expressible and lists gaps.
- Rolling the save, applying damage, conditions and Exhaustion removal, and Death at Exhaustion level 6 (`conditions.md`, `hit-points-and-damage.md`).
- Triggers ("each time the creature takes damage other than Psychic damage"), more than one repeat trigger per effect (Demonic Possession), post-recovery immunity, suppression, early-end triggers on one condition ("wakes up if it takes damage", pp. 197-198), riders that are not conditions (Speed halved, cannot knowingly lie), Harvesting Poison (an ability check), DC scaling by hour (Extreme Heat): the owning sibling models each (section 9, Q-11, Q-16).
- Time keeping: durations and repeat triggers are data; the host reports elapsed time.
- Object rules (Rolling Stone, p. 201); anything not in SRD 5.2.1.

## 2. Source

PDF page numbers equal printed page numbers.

- `SRD 5.2.1, "Condition", p. 179` (a condition does not stack with itself)
- `SRD 5.2.1, "Saving Throws", p. 7` (an effect forces a save; "you can choose to fail the save without rolling"; the effect sets the DC)
- `SRD 5.2.1, "Advantage/Disadvantage", "They Don't Stack", p. 8`
- `SRD 5.2.1, "Damage Rolls", p. 16`; `"Saving Throws and Damage", "Damage against Multiple Targets" and "Half Damage", p. 16`
- `SRD 5.2.1, "Resistance and Vulnerability", "No Stacking" and "Immunity", p. 17`
- `SRD 5.2.1, "Exhaustion [Condition]", p. 181` (levels 1 to 6, cumulative)
- `SRD 5.2.1, "Rest and Recuperation", "Cackle Fever", p. 194` (repeat saves after each Long Rest, ends after three successes; repeat at the end of each turn)
- `SRD 5.2.1, "Extended Travel", p. 192` (DC 10 plus 1 per hour past 8); `"Demonic Possession", p. 193`; `"Sewer Plague" (Fatigue, "Fighting the Contagion": a success decreases Exhaustion by 1, ending the contagion at level 0), "Sight Rot", p. 194`; `"Falling Net", p. 199`; `"Poisoned Darts", p. 200`
- `SRD 5.2.1, "Traps", "Duration", p. 199` (instantaneous: resolved instantly)
- `SRD 5.2.1, "Fear and Mental Stress", "Fear Effects", "Mental Stress Effects", "Sample Mental Stress Effects", p. 196` (a DC and a Psychic die expression per row, no printed average: 1d6, 3d6, 9d6; "discuss them with your players at the start of the campaign"; the "Consider these examples" riders; "half as much damage ... at your discretion"): the source of `ToolboxRuleOptions`
- `SRD 5.2.1, "Environmental Effects", p. 195` (Deep Water, Extreme Cold, Extreme Heat, Frigid Water)
- `SRD 5.2.1, "Poison", "Sample Poisons", pp. 197-198` ("Any effect that ends the Poisoned condition neutralizes this poison" spans both pages)
- `SRD 5.2.1, "Traps", "Poisoned Needle", pp. 200-201` (a sentence spanning both pages); `"Spiked Pit", p. 201` ("57 (13d8)")
- `SRD 5.2.1, "Damage Notation", p. 257` ("You decide whether to use the number or the die expression in parentheses; don't use both.")

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Saving throw, DC, Ability, Disadvantage | `D20TestKind.savingThrow`, `DifficultyClass` (`d20`, d20-tests.md FR-021); `ability.Ability`; `Disadvantage` (`modifier.RollCircumstance`, modifiers-and-advantage.md FR-042) | Referenced. The host rolls the save. |
| Resistance, Immunity, Half Damage | `DamageAffinities` (hit-points-and-damage.md FR-091); `HalfDamage` (`DamageAdjustment`, applied by `ApplyDamage`, FR-014, FR-099) | Read, not applied. |
| Condition, Damage Type | `ConditionKind` (content-schema-and-decoder.md FR-110), `DamageType` (FR-107), `Damage` (`combat`) | Handed back as a kind, never a combat `Condition` (FR-046, Q-5). |
| Exhaustion level | `SaveGatedExhaustionGain.levels`; `SaveGatedExhaustionReduction.levels`; `ExhaustionLevel` (`conditions.md`, read in `SaveGatedSubject.exhaustion`) | Host applies a gain as that many `ConditionSet.apply(Exhaustion)` calls (conditions.md FR-123) and a reduction with its own use case. |
| damage "can't be healed" (Pale Tincture, p. 198) | `SaveGatedDamageHealing` | |
| duration, repeat the save, Long Rest | `SaveGatedDuration`, `SaveGatedRepeatSaves`, `SaveGatedRepeatAfterLongRest` | Long Rest is a trigger only (rests-and-hit-dice.md). |
| (package terms) | `SaveGatedEffect`, `ActiveSaveGatedEffect`, `SaveGatedResolvedOnset`, `SaveGatedPrintedAverage` | The shared model; one effect on one creature; a deferred effect's onset after dice; a printed number (`SaveGatedPrintedAverageOf`) or none (`SaveGatedNoPrintedAverage`, Sample Mental Stress Effects, p. 196). |
| (package terms) | `SaveGatedPlanner`, `SaveGatedEvaluator`, `DiceRequired` | The three stateless services of section 5; named so siblings cite them (toolbox-traps.md FR-087). |
| Fear Effects riders; half damage on a successful Mental Stress save (p. 196) | `ToolboxRuleOptions.fear` (`FearOptions`, `FearRider`), `.mentalStressOnSuccess` (`MentalStressOnSuccess`, default `MentalStressNoDamageOnSuccess`) | Field types defined here (FR-102 to FR-104); `toolbox-fear-and-mental-stress.md` imports them. The bundle is not an SRD term. |

Names that exist elsewhere and are **not** used here: `SavingThrowEffect` (conditions.md), `ConditionGrant` (class-ranger.md), `DamageRoll` (removed from hit-points-and-damage.md), `EffectDamage` (conditions.md), `CunningStrikeConditionOnFailedSave` (class-rogue.md). Every type here carries the prefix `SaveGated`, a use-case name (`ResolveSaveGatedEffect...`, `ResolveRepeat...`, `EndSaveGatedEffect...`, `RollSaveGatedDamage...`) or `Toolbox`; `ActiveSaveGatedEffect` keeps the name siblings cite. The one exported function is `DiceRequired`; the event family is `SaveGatedEvent`; no generic name such as `PhaseActive` is exported.

## 4. Requirements

Ids are stable. Source "package design" means no SRD text applies. In case tables one test runs per listed outcome, and its path follows the requirement it cites.

### 4.0 Reference effects used by the case tables

Each is the structure of one SRD entry; tests cite the page. "Synthetic" fixtures test structure only. RE-E and RE-F are structural fixtures; the catalogue values belong to `toolbox-environmental-effects.md`.

| Ref | Shape | Source |
|---|---|---|
| RE-A | Constitution, DC 10; failure: 1d12 Poison (printed 6) and Poisoned; success: half damage; 24 hours; no repeat | "Assassin's Blood", p. 197 |
| RE-B | Constitution, DC 13; failure: 3d6 Poison (10); success: none; until ended; repeat at the start of each of its turns, ends after 3 successes, a failure reapplies 1d6 Poison (3) | "Burnt Othur Fumes", p. 197 |
| RE-C | Constitution, DC 16; failure: 1d6 Poison (3) and Poisoned, damage cannot be healed while the effect lasts; success: none; until ended; repeat every 24 hours, ends after 7 successes, a failure reapplies 1d6 Poison (3) only | "Pale Tincture", p. 198 |
| RE-D | Constitution, DC 13; failure: Poisoned, also Unconscious when the save fails by 5 or more; success: none; 1 hour | "Spider's Sting", p. 198 |
| RE-E | (structural) Constitution, DC 10; failure: 1 Exhaustion level; automatic success for Resistance or Immunity to Cold; instantaneous | "Extreme Cold", p. 195 |
| RE-F | (structural) Constitution, DC 5; failure: 1 Exhaustion level; Disadvantage when wearing Medium or Heavy armor; automatic success for Resistance or Immunity to Fire; instantaneous | "Extreme Heat", p. 195 |
| RE-G | Onset at midnight; Constitution, DC 17; failure: 9d6 Poison (31); success: half damage; instantaneous | "Midnight Tears", pp. 197-198 |
| RE-H | Constitution, DC 13; failure: Poisoned and Paralyzed; 1 minute; repeat at the end of each of its turns, ends after 1 success, a failure has no further effect | "Crawler Mucus", p. 197 |
| RE-I | Constitution, DC 13; failure: 1d10 Psychic (5) and Incapacitated; 1 minute; repeat at turn end, ends after 1 success | "Cackle Fever" (Uncontrollable Laughter), p. 194 |
| RE-J | Constitution, DC 15; failure: Poisoned; 4d6 hours; no repeat | "Torpor", p. 198 |
| RE-K | Synthetic: Constitution, DC 13; placeholder failure Poisoned (the SRD has no initial infection save in "Fighting the Contagion"); repeat after each Long Rest, ends after 3 successes | timing from "Cackle Fever", p. 194 |
| RE-N | `SaveGatedNoSave`; onset 1d4 days; failure: 1 Exhaustion level lasting until the effect ends; until ended; no repeat | "Cackle Fever", "Fever", p. 194 |
| RE-P | Constitution, DC 11; placeholder failure 1 Exhaustion level, until removed (as RE-K); repeat after 24 hours, success reduces Exhaustion by 1 and the effect ends at level 0 (`SaveGatedEndsWhenExhaustionReachesZero`), a failure reapplies 1 Exhaustion level, until removed; until ended; built directly in `SaveGatedPhaseActive(0)` (FR-109) | "Sewer Plague", "Fighting the Contagion", p. 194 |
| RE-M | (structural) Wisdom, DC 10; failure: 1d6 Psychic with `SaveGatedNoPrintedAverage`; success: none; instantaneous. "RE-M-half" adds `SaveGatedHalfDamageOnSuccess` (p. 196) | "Sample Mental Stress Effects", p. 196 |
| RE-S1 | Synthetic: two Damage instances (1d6 Bludgeoning, 2d8 Piercing) on a failure | structure only |
| RE-S2 | Synthetic: RE-B with Poison in the automatic-success list | structure only |
| RE-S3 | Synthetic: failure 1 Exhaustion level lasting until the effect ends; until ended; repeat at turn end, 1 success | structure only |
| RE-S4 | Synthetic: Dexterity, DC 10; failure: Restrained; until ended; no repeat; host passes `SaveGatedExempt` for a Huge target | shape of "Falling Net", p. 199 |
| RE-S5, RE-S6, RE-S7 | Synthetic: RE-C with `healing` `healable` at effect level and a different reapplied failure (S5); RE-B whose reapplied 1d6 has `SaveGatedNoPrintedAverage` (S6); RE-B with duration `SaveGatedDurationFor(SaveGatedRolledTime(1d4, hours))` (S7) | structure only |
| RE-S8 | Synthetic: initial failure of two `SaveGatedDamage` (1d6 Bludgeoning, 2d8 Piercing); reapplied failure of one (1d4 Poison); repeat at turn end, 2 successes; until ended | structure only |

### 4.1 Context and boundaries

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `toolbox` MUST be a bounded context in the single public package `github.com/Whealinghold/srd5e-go/toolbox` (`toolbox/`), with a `doc.go`, one file per concept and use-case files named `*_usecase.go`; no `domain/`, `application/` or `adapters/` sub-folders. It adds no port implementation and declares one consumer-side port, `Roller` (`Roll(dice.Expression) (dice.Result, error)`, satisfied by any `dice.Roller`), plus `ContentSource` where a sibling needs content. | package design; `doc/rules/architecture.md` §1, §2 | happy |
| FR-002 | The context map in `doc/rules/architecture.md` §2 MUST list `toolbox` with upstream `content`, `d20`, `character` and `combat`, and no other row may list `toolbox` as upstream. The files of this spec import nothing from `character`; the edge is for sibling specs (Q-14). | shared-types-and-ownership.md FR-045; `doc/rules/architecture.md` §2 | happy |
| FR-005 | `AGENTS.md` "Current contexts" and its directory-structure section MUST name `toolbox/` with the purpose text of the map row, in the same change. | `doc/rules/architecture.md` §2 | happy |
| FR-006 | The package `toolbox` MUST export its published language, each name once, each with a doc comment starting with its name, and MUST NOT export a generic variant name or any name listed as not used in section 3. There is no facade or re-export package. The published language includes `SaveGatedPlanner`, `SaveGatedEvaluator`, `DiceRequired` and the signal-name constants (`toolbox/signals.go`, the context's one place for signal names, `doc/rules/observability.md` §5; architecture-guardrails.md FR-077). A sibling adds its own constants to that file in its own change and lists its operation and `result` values in its own section 7 (section 7), never a second signals file. | `doc/rules/architecture.md` §4, §7, §9; `doc/rules/observability.md` §5 | unhappy (name collision) |

### 4.2 Definition validity

Every violation is reported by the constructor as an error wrapping `srderr.ErrInvalidInput` (`NewX(...) (X, error)`; `MustX` panics with that error; `doc/rules/architecture.md` §6; errors-and-results.md FR-025, FR-027). In the case tables "`ErrInvalidInput` error" means exactly that. Tests use `errors.Is` and `errors.As` on the typed problem and its value, never message text. Constructor notation: a name written like a call, such as `SaveGatedFixedTime(0, hours)`, stands for the Go constructor `NewSaveGatedFixedTime(0, hours)` returning `(SaveGatedFixedTime, error)`; closed-interface variants without validation are plain struct literals. Fields and accessors written in lowerCamel in the tables of this spec and its siblings (`endsAfterSuccesses`, `damageTaken`, `unspentXp`) name the exported field or accessor of the Go type (`EndsAfterSuccesses`, `DamageTaken`, `UnspentXP`); the lowerCamel form is notation only.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-008 | `SaveGatedEffect` MUST hold: a `SaveGatedRequirement` (`SaveGatedSaveRequired(ability, difficultyClass)` or `SaveGatedNoSave`, FR-105); Disadvantage circumstances (`[]SaveGatedCircumstance` (treated as a set)); automatic-success damage types (`[]DamageType` (treated as a set)); `SaveGatedOnset`; `SaveGatedFailure`; `SaveGatedSuccess`; `SaveGatedRepeat`; `SaveGatedDuration`; and the effect-level `SaveGatedDamageHealing` (FR-077). Immutable, equal (equal values) when every field is equal, lists by content and order. | SRD 5.2.1, "Saving Throws", p. 7; section 4.0 | happy |
| FR-009 | A `SaveGatedFailure` with no `SaveGatedDamage`, no `SaveGatedCondition` and no Exhaustion gain MUST be rejected. | package design | unhappy (invalid input) |
| FR-010 | `SaveGatedHalfDamageOnSuccess` MUST be rejected when the failure holds no `SaveGatedDamage`. | SRD 5.2.1, "Half Damage", p. 16 | unhappy (invalid input) |
| FR-011 | The `SaveGatedCondition` list MUST NOT hold two entries of the same `ConditionKind`. | SRD 5.2.1, "Condition", p. 179; conditions.md FR-012 | unhappy (invalid input) |
| FR-012 | A `SaveGatedCondition` MUST NOT name the Exhaustion kind; Exhaustion is only `SaveGatedExhaustionGain`. | SRD 5.2.1, "Exhaustion", p. 181; conditions.md FR-123 | unhappy (invalid input) |
| FR-013 | `SaveGatedExhaustionGain.levels` MUST be an integer from 1 to 6 inclusive. | SRD 5.2.1, "Exhaustion", p. 181; conditions.md FR-008 to FR-010 | unhappy (boundary) |
| FR-014 | `SaveGatedFailedByAtLeast(margin)` MUST require `margin` of at least 1. | SRD 5.2.1, "Spider's Sting", p. 198 | unhappy (boundary) |
| FR-015 | `SaveGatedEndsAfterSuccesses.endsAfterSuccesses` MUST be at least 1. | SRD 5.2.1, "Crawler Mucus", "Burnt Othur Fumes", p. 197 | unhappy (boundary) |
| FR-016 | A `SaveGatedRepeatSaves` MUST NOT be combined with `SaveGatedDurationInstantaneous`. | package design; nearest text SRD 5.2.1, "Burnt Othur Fumes", p. 197 | unhappy (rule interaction) |
| FR-017 | A `SaveGatedFixedTime` amount and a `SaveGatedRepeatAfterInterval` amount MUST be at least 1, and a `SaveGatedRolledTime` MUST hold a valid `dice.Expression` (dice-and-rolls.md FR-007, FR-008). | package design; SRD 5.2.1, "Torpor" (4d6 hours), p. 198 | unhappy (boundary) |
| FR-018 | `SaveGatedDamage.printedAverage` MUST be a closed interface `SaveGatedPrintedAverage`: `SaveGatedNoPrintedAverage` (the SRD prints no number, Sample Mental Stress Effects) or `SaveGatedPrintedAverageOf(amount)` with `amount` at least 1, stored as printed. It MUST NOT be derived from or checked against the dice ("57 (13d8)" is kept as printed; 13d8 averages 58) and a number the SRD does not print MUST NOT be invented or defaulted. Behaviour in printed-average mode: FR-089. | SRD 5.2.1, "Spiked Pit", p. 201; "Damage Notation", p. 257; "Sample Mental Stress Effects", p. 196 | unhappy (boundary) |
| FR-019 | Every collection reachable from an effect, state, result or event MUST be defensively copied, and mutating a collection passed to a constructor afterwards MUST NOT change the object. | `doc/rules/architecture.md` §4 | unhappy (immutability) |
| FR-020 | A `SaveGatedExhaustionGain` with lifetime `untilEffectEnds` MUST be rejected when the duration is `SaveGatedDurationInstantaneous`. | SRD 5.2.1, "Cackle Fever" ("lasts until the contagion ends"), p. 194 | unhappy (rule interaction) |
| FR-021 | An `ActiveSaveGatedEffect` in `SaveGatedPhaseAwaitingOnset` MUST be rejected unless its effect's onset is not `SaveGatedOnsetImmediate`. | package design; SRD 5.2.1, "Midnight Tears", p. 197 | unhappy (state error) |
| FR-022 | An `ActiveSaveGatedEffect` in `SaveGatedPhaseActive(successes)` MUST have `successes` from 0 up to `endsAfterSuccesses - 1`, and `successes` above 0 requires `SaveGatedEndsAfterSuccesses` (under `SaveGatedEndsWhenExhaustionReachesZero` it stays 0). | package design; SRD 5.2.1, "Pale Tincture" (seven successful saves), p. 198 | unhappy (boundary) |
| FR-023 | An `ActiveSaveGatedEffect` in `SaveGatedPhaseActive` or `SaveGatedPhaseEnded` MUST NOT hold a `SaveGatedRolledTime` in its resolved `duration`. | package design | unhappy (invalid input) |
| FR-077 | The healing mode is one effect-level field, `SaveGatedEffect.healing`. `SaveGatedFailure`, including the one inside `SaveGatedRepeatFailureReapplies`, holds no healing field, so initial and reapplied failures cannot disagree. | SRD 5.2.1, "Pale Tincture" ("can't be healed ... while the creature remains Poisoned"), p. 198 | unhappy (rule interaction) |
| FR-088 | `SaveGatedEffect`, `SaveGatedFailure`, `SaveGatedConsequences`, `SaveGatedRollPlan`, `ActiveSaveGatedEffect`, every phase, event and result variant MUST be immutable value objects: equal when all fields are equal. | `doc/rules/testing.md` §4; d20-tests.md FR-050 | unhappy (immutability) |
| FR-090 | An `ActiveSaveGatedEffect` in `SaveGatedPhaseAwaitingOnset` MUST hold a resolved onset matching its effect's onset: `SaveGatedOnsetIn` for `SaveGatedOnsetAfter`, `SaveGatedOnsetAtMidnightResolved` for `SaveGatedOnsetAtMidnight`; any other pairing MUST be rejected. | package design; SRD 5.2.1, "Midnight Tears", p. 197; "Cackle Fever" (1d4 days), p. 194 | unhappy (state error) |
| FR-091 | A resolved onset MUST never be rolled: `SaveGatedOnsetIn` holds a `SaveGatedFixedTime` (not a `SaveGatedTime`). `ResolveSaveGatedEffect` MUST roll the onset first and build the phase from the rolled amount (FR-048). | package design; SRD 5.2.1, "Cackle Fever", p. 194 | unhappy (invalid input) |
| FR-094 | `SaveGatedHalfDamageOnSuccess` MUST be rejected when the automatic-success set is non-empty, and when the repeat is `SaveGatedRepeatSaves`, because an automatic success returns no consequences (FR-024) and a successful repeat save returns none (FR-055), so the half damage would be silently lost. A sibling needing the combination amends this spec (Q-11, Q-18). | SRD 5.2.1, "Half Damage", p. 16; package design | unhappy (rule interaction) |
| FR-097 | A `Rejected` variant MUST accept only the `SaveGatedRejection` reasons its use case can return per the table in section 6, and MUST reject any other (an error wrapping `srderr.ErrInvalidInput`). A sibling's own `Rejected` variant states its subset in its spec. | package design; shared-types-and-ownership.md FR-018 | unhappy (invalid input) |
| FR-105 | An effect with `SaveGatedNoSave` MUST have no Disadvantage circumstances, an empty automatic-success set, `SaveGatedNoEffectOnSuccess`, `SaveGatedNoRepeat` and no `SaveGatedFailedByAtLeast` condition (no save, so no margin); any other combination is an error wrapping `srderr.ErrInvalidInput`. Its repeat, if any, is a second effect (RE-K, RE-P). | SRD 5.2.1, "Cackle Fever", "Fever" ("gains 1 Exhaustion level, which lasts until the contagion ends"), "Sewer Plague", "Fatigue", p. 194 | unhappy (rule interaction) |
| FR-106 | `SaveGatedRepeatSaves` MUST hold `onSuccess` (`SaveGatedRepeatSuccessCounts` or `SaveGatedRepeatSuccessReducesExhaustion(levels)`, `levels` 1 to 6) and `endRule` (`SaveGatedEndsAfterSuccesses(endsAfterSuccesses)` or `SaveGatedEndsWhenExhaustionReachesZero`); `Counts` pairs only with `EndsAfterSuccesses`, `ReducesExhaustion` only with `EndsWhenExhaustionReachesZero`. A `SaveGatedFailure` MUST NOT hold a `SaveGatedExhaustionReduction`. Violations are an error wrapping `srderr.ErrInvalidInput`. | SRD 5.2.1, "Sewer Plague", "Fighting the Contagion", p. 194; "Exhaustion", p. 181 | unhappy (rule interaction) |
| FR-110 | `SaveGatedDamage.instances` (default `SaveGatedOneInstance`) MAY be `SaveGatedRolledInstances(count)`: the damage is dealt once per instance, the instance count being the roll of `count`, and each instance is its own roll of `dice` (the darts' "per dart"). `count.minimum` MUST be at least 1 (an error wrapping `srderr.ErrInvalidInput`); `printedAverage` is per instance. Legal in any failure, including a reapplied one. | SRD 5.2.1, "Poisoned Darts" ("struck by 1d3 darts, taking 3 (1d6) Poison damage per dart"), p. 200 | unhappy (boundary) |

Case table 4.2a (FR-009 to FR-023, FR-077, FR-088, FR-090, FR-091, FR-094, FR-097, FR-105, FR-106, FR-110):

| Label | Construct | Expected | FR |
|---|---|---|---|
| empty failure | `SaveGatedFailure` with no damage, conditions or Exhaustion | `ErrInvalidInput` error | FR-009 |
| half damage without damage | RE-D failure with `SaveGatedHalfDamageOnSuccess` | `ErrInvalidInput` error | FR-010 |
| duplicate condition | `[poisoned, poisoned]` | `ErrInvalidInput` error | FR-011 |
| Exhaustion as condition | `SaveGatedCondition` of the Exhaustion kind | `ErrInvalidInput` error | FR-012 |
| Exhaustion levels 0 / 1 / 6 / 7 | `SaveGatedExhaustionGain(n, ...)` | `ErrInvalidInput` error / ok / ok / `ErrInvalidInput` error | FR-013 |
| margin 0 / 1 / 5 | `SaveGatedFailedByAtLeast(n)` | `ErrInvalidInput` error / ok / ok | FR-014 |
| successes 0 / 1 | `endsAfterSuccesses: n` | `ErrInvalidInput` error / ok | FR-015 |
| repeat and instantaneous | RE-B with `SaveGatedDurationInstantaneous` | `ErrInvalidInput` error | FR-016 |
| amount 0 | `SaveGatedFixedTime(0, hours)`; `SaveGatedRepeatAfterInterval(0, hours)`; `SaveGatedOnsetIn(SaveGatedFixedTime(0, days))` | `ErrInvalidInput` error each | FR-017 |
| average 0 / 1 | `SaveGatedPrintedAverageOf(n)` on 1d6 Poison | `ErrInvalidInput` error / ok | FR-018 |
| printed 57 for 13d8; no number | `SaveGatedPrintedAverageOf(57)`; RE-M `SaveGatedNoPrintedAverage()` | accepted unchanged; accepted | FR-018 |
| Exhaustion until effect ends, instantaneous | RE-E with lifetime `untilEffectEnds` | `ErrInvalidInput` error | FR-020 |
| awaiting onset, immediate / deferred | `SaveGatedPhaseAwaitingOnset` for RE-A / RE-G | `ErrInvalidInput` error / ok | FR-021 |
| successes 3 / 2 / -1 for RE-B; 1 for RE-A | `SaveGatedPhaseActive(n)` | `ErrInvalidInput` error / ok / `ErrInvalidInput` error / `ErrInvalidInput` error | FR-022 |
| rolled time in active or ended duration | `SaveGatedDurationFor(SaveGatedRolledTime(4d6, hours))` in `SaveGatedPhaseActive`; in `SaveGatedPhaseEnded(durationElapsed)` | `ErrInvalidInput` error each | FR-023 |
| healing is effect-level | RE-S5 | `SaveGatedFailure` has no healing field; the reapplied failure uses the effect's | FR-077 |
| equality | two `ActiveSaveGatedEffect` from equal fields; two results differing in one field | equal; not equal | FR-088 |
| half damage with automatic-success set; with repeat saves (turn end, 1 success, 1 hour) | RE-A modified | `ErrInvalidInput` error each | FR-094 |
| half damage, empty set, no repeat | RE-A | accepted | FR-094 |
| onset mismatch | `SaveGatedPhaseAwaitingOnset(SaveGatedOnsetAtMidnightResolved)` for an effect with `SaveGatedOnsetAfter` | `ErrInvalidInput` error | FR-090 |
| onset match | midnight for RE-G; `SaveGatedOnsetIn(SaveGatedFixedTime(3, days))` for `SaveGatedOnsetAfter(SaveGatedRolledTime(1d4, days))` | accepted | FR-090, FR-091 |
| rolled instances count minimum 0 / 1 | `SaveGatedRolledInstances(1d3 - 1)`; `(1d3)` | `ErrInvalidInput` error / ok | FR-110 |
| rejection reason not allowed | `ResolveSaveGatedEffectRejected(noRepeatSaves)`; `ResolveRepeatSaveRejected(effectNotAwaitingOnset)`; `EndSaveGatedEffectRejected(saveDoesNotMatchEffect)` | `ErrInvalidInput` error | FR-097 |
| rejection reasons allowed | each pairing of the section 6 table | accepted | FR-097 |
| `SaveGatedNoSave` with Disadvantage / automatic-success type / half damage / repeat saves / `SaveGatedFailedByAtLeast`; plain RE-N | RE-N modified | `ErrInvalidInput` error each; accepted | FR-105 |
| mismatched pairing; levels 0 / 1 / 6 / 7; Reduction in a failure | `Counts` with `EndsWhenExhaustionReachesZero`; `ReducesExhaustion(n)` with `EndsAfterSuccesses`; `SaveGatedFailure` with `SaveGatedExhaustionReduction` | `ErrInvalidInput` error; `ErrInvalidInput` error / ok / ok / `ErrInvalidInput` error; `ErrInvalidInput` error | FR-106 |

### 4.3 Planning: automatic success and roll mode

`SaveGatedPlanner` is the single decider of automatic success and of the armor Disadvantage. Siblings put damage types in the effect's automatic-success set, `SaveGatedCircumstance` values on the `SaveGatedSubject`, and any non-affinity exemption in `SaveGatedSubject.exemption` (FR-076). Sibling-only deciding rules (adaptation in toolbox-environmental-effects.md; Huge or larger target, toolbox-traps.md FR-068) stay with the sibling.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-024 | A subject with Resistance or Immunity to a damage type in the automatic-success set MUST automatically succeed: the result is `ResolveSaveGatedEffectAutoSucceeded` (carrying `ability` and `saveResult` `autoSucceeded`, FR-092), no save is rolled and no consequence returned. No half damage is lost (FR-094). | SRD 5.2.1, "Extreme Cold" ("Resistance or Immunity to Cold damage automatically succeed"), "Extreme Heat", p. 195 | happy |
| FR-025 | A Resistance whose scope is all damage MUST count as Resistance to every listed damage type. | SRD 5.2.1, "Resistance and Vulnerability", "No Stacking", p. 17; hit-points-and-damage.md FR-022, FR-091. Interpretation: p. 195 names only Cold or Fire damage; p. 17 shows Resistance to all damage exists. | unhappy (rule interaction) |
| FR-026 | Resistance or Immunity to an unlisted damage type, Vulnerability alone, or no affinities MUST NOT cause automatic success. | SRD 5.2.1, "Extreme Cold", p. 195 | unhappy (boundary) |
| FR-027 | An effect whose automatic-success set is empty MUST never automatically succeed, whatever the affinities. | SRD 5.2.1, "Assassin's Blood", p. 197 | unhappy (rule interaction) |
| FR-028 | When the subject has at least one circumstance in the effect's Disadvantage set, the roll circumstances MUST be exactly one `Disadvantage`, even when two match. | SRD 5.2.1, "Extreme Heat", p. 195; "They Don't Stack", p. 8 | happy |
| FR-029 | When no circumstance matches, or the Disadvantage set is empty, the roll circumstances MUST be empty. | SRD 5.2.1, "Extreme Heat", p. 195 | unhappy (empty and minimal) |
| FR-030 | When a save is needed and none was supplied (`SaveGatedSaveNotRolled`), `ResolveSaveGatedEffect` MUST return `ResolveSaveGatedEffectSaveRequired` with a `SaveGatedRollPlan` (the `Ability`, `DifficultyClass` and roll circumstances) from which the host builds a `SavingThrowCommand`. A normal result, not an error. | SRD 5.2.1, "Saving Throws", p. 7 | happy |
| FR-031 | `AutoSucceeded` and `SaveRequired` MUST NOT request any dice. | package design | unhappy (empty and minimal) |
| FR-032 | When the subject qualifies for automatic success (FR-024 or FR-076), a rolled or forced save supplied anyway MUST be ignored and not validated (a mismatching one is not rejected): the result is `AutoSucceeded`. | SRD 5.2.1, "Extreme Cold", p. 195 | unhappy (rule interaction) |
| FR-076 | `SaveGatedSubject.exemption` (closed interface: `SaveGatedNoExemption`, `SaveGatedExempt`) is a fact decided by the host or a sibling before the handoff. With `SaveGatedExempt` the result MUST be `AutoSucceeded` as in FR-024, whatever the affinities and circumstances, and the planner MUST NOT ask why. With `SaveGatedNoExemption` the planner follows FR-024 to FR-029. | SRD 5.2.1, "Falling Net" ("succeeds automatically if it's Huge or larger"), p. 199; "Deep Water", "Frigid Water", p. 195 | unhappy (rule interaction) |
| FR-098 | `SaveGatedExempt` on an effect with `SaveGatedHalfDamageOnSuccess` MUST still return `AutoSucceeded` with no consequences and no dice: the exemption means the effect does not apply (the definition cannot forbid this, FR-076). Interpretation (Q-18). | SRD 5.2.1, "Falling Net", p. 199; "Half Damage", p. 16 | unhappy (rule interaction) |

Case table 4.3a (planner, FR-024 to FR-032, FR-076, FR-098, FR-030, FR-031):

| Label | Effect | Subject | Expected | FR |
|---|---|---|---|---|
| Resistance / Immunity to Cold | RE-E | Resistance / Immunity to Cold | `AutoSucceeded` | FR-024 |
| Resistance to all damage | RE-E | Resistance, all damage | `AutoSucceeded` | FR-025 |
| unlisted type, Vulnerability only, none | RE-E | Resistance to Fire; Vulnerability to Cold; none | `SaveRequired` Constitution, DC 10, no circumstances | FR-026 |
| empty set | RE-A | Immunity to Poison | `SaveRequired` Constitution, DC 10 | FR-027 |
| armor Disadvantage | RE-F | wearing Medium or Heavy armor | `SaveRequired` DC 5, `[Disadvantage]` | FR-028 |
| armor absent; circumstance not listed | RE-F not wearing; RE-A wearing | | `SaveRequired`, `[]` | FR-029 |
| automatic success beats Disadvantage; two circumstances | RE-F with armor and Resistance to Fire; a synthetic effect listing two, both present | | `AutoSucceeded`; `SaveRequired`, exactly `[Disadvantage]` | FR-024, FR-028 |
| exempt | RE-S4 | `SaveGatedExempt`, no affinities | `AutoSucceeded`; no dice | FR-076 |
| not exempt | RE-S4 | `SaveGatedNoExemption` | `SaveRequired` Dexterity, DC 10 | FR-076 |
| exempt beats Disadvantage | RE-F | `SaveGatedExempt` and armor | `AutoSucceeded` | FR-076 |
| exempt on a half-damage effect | RE-A | `SaveGatedExempt` | `AutoSucceeded`; no consequences; no dice | FR-098 |
| supplied save ignored | RE-S4 exempt with a `SaveGatedSaveRolled` against DC 12; RE-E Immunity to Cold with a failed `SaveGatedSaveRolled` | | `AutoSucceeded` (not rejected) | FR-032 |
| save not supplied | RE-A | none | `SaveRequired`, no dice requested | FR-030, FR-031 |

### 4.4 Initial resolution

The host passes the finished save as `SaveGatedSaveRolled(CheckResult)` or `SaveGatedSaveForcedFailure(ForcedFailureResult)` (including a chosen failure, SRD p. 7). The save total and the DC give the margin of failure.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-033 | A rolled success with `SaveGatedNoEffectOnSuccess` MUST return `ResolveSaveGatedEffectSaved` with empty consequences and no dice. | SRD 5.2.1, "Burnt Othur Fumes", p. 197 | happy |
| FR-034 | A rolled success with `SaveGatedHalfDamageOnSuccess` MUST return `Saved` with the failure damage rolled once per `SaveGatedDamage`, adjustments `[HalfDamage]`, and no conditions or Exhaustion. | SRD 5.2.1, "Assassin's Blood" ("takes half as much damage only"), p. 197; "Half Damage", p. 16 | happy |
| FR-035 | Half damage MUST be reported as the `HalfDamage` adjustment, never as a halved number: rounding is owned by `ApplyDamage`. | SRD 5.2.1, "Half Damage" (round down), p. 16; hit-points-and-damage.md FR-014, FR-099 | unhappy (rule interaction) |
| FR-036 | A rolled or forced failure MUST return `ResolveSaveGatedEffectFailed` with `SaveGatedConsequences` (damage, conditions, Exhaustion gain, `SaveGatedDamageHealing`), the `ActiveSaveGatedEffect` and the events. | SRD 5.2.1, "Assassin's Blood", p. 197; "Poisoned Needle", pp. 200-201 | happy |
| FR-037 | On a rolled failure a `SaveGatedCondition` MUST be returned when it requires `SaveGatedAnyFailure`, or `SaveGatedFailedByAtLeast(m)` and the DC minus the save total is at least `m`. Conditions keep the effect's order. | SRD 5.2.1, "Spider's Sting", p. 198 | happy; unhappy (boundary) |
| FR-038 | A forced failure (automatic or chosen) MUST return the base conditions only: a `SaveGatedFailedByAtLeast` condition MUST NOT be returned, as no total exists. Interpretation (Q-6). | SRD 5.2.1, "Saving Throws", p. 7 | unhappy (rule interaction) |
| FR-039 | A rolled save whose `CheckResult` is not a saving throw or whose target DC differs from the effect's, and a forced failure whose kind is not `savingThrow`, MUST return `ResolveSaveGatedEffectRejected` (or `ResolveRepeatSaveRejected`) with `saveDoesNotMatchEffect`; nothing is rolled. A host value that fails validation is a returned rejection, not a panic or an error (`doc/rules/architecture.md` §6). **Limit:** results carry no `Ability` (d20-tests.md FR-041), so a save made with another ability is not detected; using `SaveGatedRollPlan.ability` is the host's duty (Q-19). A forced failure carries no DC, so only its kind is checked. | SRD 5.2.1, "Saving Throws", p. 7; d20-tests.md FR-041 | unhappy (invalid input) |
| FR-040 | With `SaveGatedDamageAmount.rolled`, the use case MUST request one roll of each `SaveGatedDamage` of the applied failure (FR-042), in list order, and return one `Damage` per instance (its `DamageType`, the rolled total). | SRD 5.2.1, "Damage Rolls", p. 16 | happy |
| FR-041 | With `SaveGatedDamagePrintedAverage`, the use case MUST return each instance's printed number as its damage and request no damage dice. `SaveGatedNoPrintedAverage` is FR-089. Interpretation (Q-9; p. 257 is a stat-block rule, applying it to Toolbox entries is the interpretation). | SRD 5.2.1, "Damage Notation", p. 257 | unhappy (rule interaction) |
| FR-042 | The damage instances a call applies are those of the failure it resolves: the effect's `SaveGatedFailure` for `ResolveSaveGatedEffect`, the failure inside `SaveGatedRepeatFailureReapplies` for `ResolveRepeatSave`. With `SaveGatedDamageShared(damage)` the use case MUST use the list unchanged and request no dice; a list whose length (for rolled instances, FR-111) or `DamageType`s differ from those instances MUST return `Rejected` with `sharedDamageMismatch`; nothing is rolled. Applies only when the call would return damage. | SRD 5.2.1, "Damage against Multiple Targets", p. 16 | unhappy (invalid input) |
| FR-043 | `RollSaveGatedDamage` MUST roll the instances of the failure selected by `SaveGatedDamageSource` (`initialFailure` or `reappliedFailure`, FR-096) once (as FR-040) and return the `Damage` list and `totalDamage` for the host to pass as `SaveGatedDamageShared` to every target. It rolls dice only: a table using printed numbers passes `SaveGatedDamagePrintedAverage` to each target (FR-041). | SRD 5.2.1, "Damage against Multiple Targets", p. 16 | happy |
| FR-044 | A failed save with Exhaustion MUST return `SaveGatedExhaustionGain(levels, lifetime)`; the engine applies none. The host applies one `ConditionSet.apply(Exhaustion)` per level. | SRD 5.2.1, "Deep Water", "Extreme Cold", p. 195; conditions.md FR-123 | happy |
| FR-045 | Every `SaveGatedConsequences` for an effect (initial failure, half-damage success, reapplied failure) MUST carry that effect's `healing` (FR-077): `unhealableWhileEffectLasts` for RE-C, `healable` for every other effect. | SRD 5.2.1, "Pale Tincture", p. 198 | happy |
| FR-046 | Result conditions MUST be `ConditionKind` values only. The engine MUST NOT construct a combat `Condition`, apply damage or a condition, or change a `ConditionSet` or `HitPointStatus`. | `doc/rules/architecture.md` §2, §4; Q-5 | unhappy (forbidden edge) |
| FR-047 | An initial failure MUST emit exactly one `SaveGatedEffectStarted(target)`, first in the event list (a reapplied failure emits none, FR-095); `AutoSucceeded`, `SaveRequired` and `Saved` emit none. | `doc/rules/architecture.md` §3 | happy |
| FR-082 | `Deferred` MUST emit no event; `SaveGatedEffectStarted` is emitted when the pending effect later fails (FR-050). | package design; SRD 5.2.1, "Midnight Tears", p. 197 | unhappy (empty and minimal) |
| FR-083 | An initial failure with `SaveGatedDurationInstantaneous` MUST emit `[SaveGatedEffectStarted, SaveGatedEffectEnded(instantaneous)]`; that `Ended` lists no conditions and Exhaustion to remove 0 (consequences are not undone). | package design; SRD 5.2.1, "Extreme Cold", p. 195 | happy |
| FR-087 | A pure function `DiceRequired(effect, stage, saveOutcome, damageAmount, rolledCounts)` MUST return the ordered `dice.Expression`s to roll; use cases roll exactly that list and decide nothing else about dice. `stage` is a `SaveGatedDiceStage` (`initialExposure`, `onsetReached`, `repeatSave`, `sharedInitialFailure`, `sharedReappliedFailure`); `saveOutcome` is a `SaveGatedSaveResult` (always `failed` for the shared stages). It is called only after validation (FR-039, FR-042, FR-051, FR-061, FR-089). **Initial and onset:** a `SaveGatedNoEffectOnSuccess` success, `AutoSucceeded`, `SaveRequired`, printed-average and shared damage request no damage dice; a half-damage success requests the failure's damage dice; a failure requests the damage dice then a rolled duration; a `Deferred` initial exposure requests only a rolled onset. **Repeat:** a success requests nothing; a failure under `SaveGatedRepeatFailureReapplies` requests the reapplied damage dice only, never a duration or onset; under `SaveGatedRepeatFailureNoFurtherEffect` nothing. **Shared:** the damage dice of the initial or reapplied failure only. Order as FR-070; rolled instance counts: FR-111. | package design (`doc/rules/architecture.md` §4); dice-and-rolls.md FR-050 | happy; unhappy (empty and minimal) |
| FR-089 | With `SaveGatedDamagePrintedAverage`, a damage instance the call would return and whose `printedAverage` is `SaveGatedNoPrintedAverage` MUST return `Rejected` with `noPrintedAverage`; nothing is rolled, no number is invented, and no state, damage or event is produced. A call that returns no damage is not rejected, and rolled and shared damage never read the printed number. The check follows save validation (FR-039). | SRD 5.2.1, "Sample Mental Stress Effects", p. 196; "Damage Notation", p. 257 | unhappy (invalid input) |
| FR-095 | A failed repeat save with `SaveGatedRepeatFailureReapplies` MUST keep the resolved duration and `successes` unchanged, roll no duration, return the same `SaveGatedPhaseActive(successes)`, emit no `SaveGatedEffectStarted`, and add the reapplied conditions and Exhaustion to `grantedConditions` and `exhaustionToRemove` (FR-067). Of FR-036 to FR-047 only the consequence rules apply (damage FR-040 to FR-042 on the reapplied instances, conditions FR-037/FR-038, Exhaustion FR-044, healing FR-045, FR-046), not the state-creating ones (FR-036's new state, FR-047, FR-052 to FR-054, FR-083). | SRD 5.2.1, "Burnt Othur Fumes" ("On each successive failed save ... 3 (1d6) Poison damage"), p. 197; "Pale Tincture", p. 198 | unhappy (rule interaction) |
| FR-096 | `RollSaveGatedDamageCommand` with `reappliedFailure` MUST be rejected (an error wrapping `srderr.ErrInvalidInput`) when the effect's repeat is not `SaveGatedRepeatSaves` with `SaveGatedRepeatFailureReapplies`. A selected failure with no `SaveGatedDamage` (RE-E) is not an error: `RollSaveGatedDamageRolled` with empty `damage`, `totalDamage` 0, no dice. | package design; SRD 5.2.1, "Burnt Othur Fumes", p. 197 | unhappy (empty and minimal) |
| FR-107 | For a `SaveGatedNoSave` effect `ResolveSaveGatedEffect` MUST not consult the planner or the subject and MUST ignore a supplied save (as FR-032); it never returns `SaveRequired`, `AutoSucceeded` or `Saved`. Immediate onset returns `Failed` (no `ability`, `saveResult` `notRequired`) with the failure's consequences, state and events as FR-036 to FR-047, FR-052 to FR-054, FR-083; a deferred onset returns `Deferred` (FR-048, FR-049) and `SaveGatedOnsetReached` then returns `Failed`. `DiceRequired` treats it as a failure. | SRD 5.2.1, "Cackle Fever" (effects begin 1d4 days after infection), "Sewer Plague", p. 194 | unhappy (rule interaction) |
| FR-111 | A call that returns damage for a `SaveGatedRolledInstances` damage MUST roll `count` once per such damage (list order, before every instance), then each instance's `dice`, and return one `Damage` per instance in order. In printed-average mode the count is still rolled (it is a number of instances, not a damage roll; interpretation: the SRD prints no average for it) and each instance returns its printed number. With `SaveGatedDamageShared` nothing is rolled and the list matches when, for each damage in order, it holds between `count.minimum` and `count.maximum` entries of that type (FR-042). `RollSaveGatedDamage` rolls the count and the instances once for all targets, so every target takes the same number of instances (p. 16 "roll the damage once for all the targets"; applying it to the count is an interpretation). A call that returns no damage rolls no count. `DiceRequired` takes `rolledCounts` (`[]int`, one per rolled-count damage, default empty): without it, it returns only the count expressions in list order; with it, the rest of the list (instance dice, each damage repeated by its count, then the duration; the counts are not repeated) (a count outside `count.minimum` to `count.maximum` is an error wrapping `srderr.ErrInvalidInput`). | SRD 5.2.1, "Poisoned Darts", p. 200; "Damage against Multiple Targets", p. 16 | happy; unhappy (boundary) |

Case table 4.4a. Level (`doc/rules/testing.md` §3): *unit* calls the planner, evaluator or `DiceRequired` with already-rolled values and imports no port; *application* runs a use case with `dicetest.FixedRoller` and asserts requests, a returned `Rejected` or events. The save is `SaveGatedSaveRolled` unless stated.

| Label | Level | Effect and input | Expected | FR |
|---|---|---|---|---|
| RE-A failure | application | total 9 vs DC 10; 1d12 rolls 7 | `Failed` (`ability` Constitution, `saveResult` `failed`); damage `[7 Poison]`; `totalDamage` 7; conditions `[poisoned]`; 24 hours; `healable`; events `[Started]` | FR-036, FR-040, FR-045, FR-047, FR-092 |
| RE-A success | unit | total 10 vs DC 10; 1d12 value 7 | `Saved`; damage `[7 Poison]`; `[HalfDamage]`; no conditions | FR-034, FR-035 |
| RE-A printed average | application | failure, `printedAverage` | damage `[6 Poison]`; no dice | FR-041 |
| RE-A chosen failure | unit | `ForcedFailureResult` (kind `savingThrow`), 1d12 value 7 | `Failed`, damage and conditions equal to the rolled failure | FR-036, FR-038 |
| RE-A shared damage | application | `SaveGatedDamageShared([4 Poison])` | damage `[4 Poison]`; no dice | FR-042 |
| RE-A shared wrong type / wrong length | application | `Shared([4 Fire])`; `Shared([])` | `Rejected(sharedDamageMismatch)`; no dice | FR-042 |
| RE-A wrong DC; ability check; forced failure of kind `attackRoll`; of kind `savingThrow` | application | DC 11; `CheckResult` kind `abilityCheck`; `ForcedFailureResult` kind `attackRoll`; kind `savingThrow` | `Rejected(saveDoesNotMatchEffect)` with no dice for the first three; `Failed` (not rejected) for the last | FR-039 |
| RE-A save of another ability | unit | `CheckResult` kind `savingThrow`, DC 10, made with Strength | evaluated as a save (not detected; the host's duty) | FR-039 |
| RE-B success | application | total 13 vs DC 13 | `Saved`; empty; no dice | FR-033 |
| RE-B failure | unit | total 12 vs DC 13; 3d6 value 11 | `Failed`; `[11 Poison]`; no conditions; active, 0 successes | FR-036, FR-040 |
| RE-C failure | unit | total 3; 1d6 value 2 | `Failed`; `[2 Poison]`; `[poisoned]`; `unhealableWhileEffectLasts` | FR-045 |
| RE-E failure | application | total 9 vs DC 10 | `Failed`; no damage; `totalDamage` 0; Exhaustion gain 1; no dice; events `[Started, Ended(instantaneous)]`, the `Ended` with no conditions and Exhaustion 0 | FR-044, FR-083 |
| RE-E success | unit | total 10 vs DC 10 | `Saved`; empty | FR-033 |
| RE-A success dice | unit | total 10 vs DC 10 | `DiceRequired` is `[1d12]` | FR-087 |
| RE-B success dice | unit | total 13 vs DC 13 | `DiceRequired` is `[]` | FR-087 |
| RE-J failure dice | unit | total 14 vs DC 15 | `DiceRequired` is `[4d6]` (duration) | FR-087, FR-070 |
| RE-J failure | unit | 4d6 value 14 | `Failed`; `[poisoned]`; 14 hours | FR-052 |
| RE-S1 failure | application | 1d6 rolls 3, 2d8 rolls 9 | damage `[3 Bludgeoning, 9 Piercing]` in order; two dice requests in order | FR-040, FR-070 |
| RE-M failure, rolled | unit | Wisdom DC 10; total 9; 1d6 value 4 | `Failed`; `[4 Psychic]`; `totalDamage` 4; `healable` | FR-018, FR-040 |
| RE-M printed average | application | failure; success with RE-M-half; plain success (total 10 vs DC 10) | `Rejected(noPrintedAverage)` with no dice or state for the first two; `Saved`, empty, not rejected for the plain success | FR-089 |
| darts (rolled instances) | application | Synthetic: DC 13 Dexterity, `SaveGatedDamage(1d6, Poison, 3, SaveGatedRolledInstances(1d3))`; failure, count rolls 2, then 1d6 rolls 4 and 5; `DiceRequired` without and with `rolledCounts [2]`; `RollSaveGatedDamage` once for two targets; `Shared` of 1, 3 and 4 Poison, of 3 Fire; printed-average mode | `Failed`, `[4 Poison, 5 Poison]`, `totalDamage` 9, requests `[1d3, 1d6, 1d6]` in that order; `[1d3]` then `[1d6, 1d6]` (the shared stages likewise); one roll set, both targets `[4, 5]`; accepted for 1 to 3, `Rejected(sharedDamageMismatch)` for 4 entries and for the type; the count still rolled, each instance returns 3 | FR-110, FR-111, FR-087, FR-042 |
| RE-M-half success, rolled | unit | total 10; 1d6 value 4 | `Saved`; `[4 Psychic]`; `[HalfDamage]` | FR-034 |
| RE-M failure, shared | application | `Shared([4 Psychic])` | `[4 Psychic]`; no dice; not rejected | FR-089, FR-042 |
| printed number present | application | RE-A failure, `SaveGatedPrintedAverageOf(6)` | `[6 Poison]`; not rejected | FR-041, FR-089 |
| RE-N no save | application | `SaveGatedSaveNotRolled`, 1d4 rolls 2; onset reached with a `SaveGatedSaveRolled` (ignored) | `Deferred` (`SaveGatedOnsetIn(SaveGatedFixedTime(2, days))`), no `SaveRequired`; `Failed` (no `ability`, `notRequired`), Exhaustion gain 1 `untilEffectEnds`, `Active(0)` until ended; no `saves` increment | FR-107, FR-048 |

Case table 4.4b, margin of failure (RE-D, DC 13; unit tests of the evaluator; FR-033, FR-037, FR-038):

| Label | Save | Expected conditions |
|---|---|---|
| success at the DC | total 13 | none (`Saved`) |
| fails by 1 / by 4 | total 12 / 9 | `[poisoned]` |
| fails by 5 / by 10 | total 8 / 3 | `[poisoned, unconscious]` |
| negative total | total -2 | `[poisoned, unconscious]` |
| automatic failure; chosen failure | `ForcedFailureResult` | `[poisoned]` |

Case table 4.4c, `RollSaveGatedDamage` and the shared stages of `DiceRequired` (FR-043, FR-087, FR-096):

| Label | Level | Input | Expected | FR |
|---|---|---|---|---|
| initial failure | application | RE-S1, `initialFailure`; 1d6 rolls 3, 2d8 rolls 9 | `Rolled([3 Bludgeoning, 9 Piercing])`; `totalDamage` 12; two dice in order | FR-043, FR-040 |
| reapplied failure | application | RE-S8, `reappliedFailure`; 1d4 rolls 2 | `Rolled([2 Poison])`; `totalDamage` 2; one die | FR-043 |
| reapplied failure absent | unit | `Command(RE-A, reappliedFailure)`; `Command(RE-H, reappliedFailure)` | `ErrInvalidInput` error | FR-096 |
| failure with no damage | application | RE-E, `initialFailure` | `Rolled([])`; `totalDamage` 0; no dice | FR-096 |
| one roll, many targets | application | RE-A rolled once (7); two targets with `Shared([7 Poison])` | one roller request in total; each target's damage `[7 Poison]` | FR-043, FR-042 |
| shared dice | unit | `DiceRequired(RE-S8, sharedReappliedFailure / sharedInitialFailure, failed, rolled)` | `[1d4]` / `[1d6, 2d8]` | FR-087 |

### 4.5 Onset and duration

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | `ResolveSaveGatedEffect` at `SaveGatedInitialExposure` with onset `SaveGatedOnsetAfter(time)` MUST return `Deferred` with an `ActiveSaveGatedEffect` in `SaveGatedPhaseAwaitingOnset(onset: SaveGatedOnsetIn(fixed))`, where `fixed` is the time resolved to a `SaveGatedFixedTime` of the same unit (rolling its dice when rolled, FR-091). The host schedules `SaveGatedOnsetReached` from it. The state's `duration` is the effect's own, still unrolled (rolled at start, FR-052). No save, damage or condition is returned. | SRD 5.2.1, "Midnight Tears" ("suffers no effect until the stroke of midnight"), p. 197; "Cackle Fever" (1d4 days), p. 194 | happy |
| FR-049 | An effect with onset `SaveGatedOnsetAtMidnight` MUST return `Deferred` with a state in `SaveGatedPhaseAwaitingOnset(onset: SaveGatedOnsetAtMidnightResolved)` and no dice. | SRD 5.2.1, "Midnight Tears", p. 197 | happy |
| FR-050 | `SaveGatedOnsetReached(pending)` with a `SaveGatedPhaseAwaitingOnset` state MUST proceed as an initial resolution of the pending effect (FR-024 to FR-047). | SRD 5.2.1, "Midnight Tears" ("DC 17 Constitution saving throw, taking 31 (9d6) Poison damage on a failed save or half as much damage on a successful one"), p. 198 | happy |
| FR-051 | `SaveGatedOnsetReached(pending)` with a `SaveGatedPhaseActive` or `SaveGatedPhaseEnded` state MUST return `Rejected` with `effectNotAwaitingOnset`; nothing is rolled. | package design | unhappy (state error) |
| FR-052 | A `SaveGatedRolledTime` duration MUST be rolled when the effect starts and returned as a `SaveGatedFixedTime` of the rolled amount and unit. | SRD 5.2.1, "Torpor" (4d6 hours), p. 198 | happy |
| FR-053 | An initial failure with `SaveGatedDurationInstantaneous` MUST return a state in `SaveGatedPhaseEnded(instantaneous)`. | SRD 5.2.1, "Traps", "Duration", p. 199; "Extreme Cold", p. 195 | happy |
| FR-054 | An initial failure (not a reapplied one, FR-095) with any other duration MUST return `SaveGatedPhaseActive(0)` carrying the resolved duration. The engine MUST NOT expire it: the host calls `EndSaveGatedEffect`. | SRD 5.2.1, "Assassin's Blood" (24 hours), p. 197; "Poisoned Needle" (1 hour), pp. 200-201 | happy |

Case table 4.5a (FR-048 to FR-054):

| Label | Effect and input | Expected | FR |
|---|---|---|---|
| midnight onset | RE-G at initial exposure | `Deferred`; `AwaitingOnset(SaveGatedOnsetAtMidnightResolved)`; no dice; no save | FR-049 |
| rolled onset | synthetic 1d4 days, roll 3 / extreme rolls 1 and 4 | `AwaitingOnset(SaveGatedOnsetIn(SaveGatedFixedTime(3, days)))`; one die; state `duration` the effect's own / onset 1 day and 4 days | FR-048, FR-091 |
| fixed onset | synthetic `SaveGatedOnsetAfter(SaveGatedFixedTime(2, days))` | `Deferred`; `SaveGatedOnsetIn(SaveGatedFixedTime(2, days))`; no dice | FR-048 |
| onset reached, failure | RE-G pending; total 16 vs DC 17; 9d6 rolls 30 | `Failed`; `[30 Poison]`; `Ended(instantaneous)` | FR-050, FR-053 |
| onset reached, success | RE-G pending; total 17 | `Saved`; damage rolled; `[HalfDamage]` | FR-050 |
| onset reached on active / ended state | RE-B in `Active(0)`; RE-G in `Ended(neutralized)` | `Rejected(effectNotAwaitingOnset)`; no dice | FR-051 |
| rolled duration | RE-J failure; 4d6 rolls 14 | 14 hours, no rolled time | FR-052 |
| instantaneous; fixed; until ended | RE-E; RE-D; RE-C failure | `Ended(instantaneous)`; `Active(0)` 1 hour; `Active(0)` until ended | FR-053, FR-054 |

### 4.6 Repeat saves

The host calls `ResolveRepeatSave` when the trigger occurs (reported on the state, FR-063); the engine cannot check the time.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-055 | A successful repeat save under `SaveGatedRepeatSuccessCounts` on an active effect MUST return `ResolveRepeatSaveContinues` with `successes` plus 1, empty consequences and one `SaveGatedEffectRepeatSaveMade(result: succeeded)`. | SRD 5.2.1, "Burnt Othur Fumes", p. 197 | happy |
| FR-056 | When `successes` reaches `endsAfterSuccesses`, the result MUST be `ResolveRepeatSaveEnds` with `SaveGatedPhaseEnded(successesReached)` and one `SaveGatedEffectEnded`. | SRD 5.2.1, "Pale Tincture" (seven successful saves), p. 198; "Crawler Mucus", p. 197 | unhappy (boundary) |
| FR-057 | Successes MUST be counted cumulatively: a failure between two successes MUST NOT reset the count. Interpretation (Q-8). | SRD 5.2.1, "Cackle Fever" ("succeeds on three of these saves"), p. 194 | unhappy (rule interaction) |
| FR-058 | A failed repeat save with `SaveGatedRepeatFailureReapplies(outcome)` MUST return `Continues` with `SaveGatedConsequences` built from that outcome (rules as FR-095 lists them), `successes` and duration unchanged, and `SaveGatedEffectRepeatSaveMade(result: failed)`; dice are those of `DiceRequired` at stage `repeatSave` (FR-087). | SRD 5.2.1, "Burnt Othur Fumes", p. 197 | happy |
| FR-059 | A failed repeat save with `SaveGatedRepeatFailureNoFurtherEffect` MUST return `Continues` with empty consequences and `successes` unchanged. | SRD 5.2.1, "Crawler Mucus", p. 197 | unhappy (empty and minimal) |
| FR-060 | An automatic success on a repeat save (FR-024) MUST count as a success, with `result: autoSucceeded`. | package design; SRD 5.2.1, "Extreme Cold", p. 195 | unhappy (rule interaction) |
| FR-061 | `ResolveRepeatSave` MUST return `Rejected` with `noRepeatSaves` for an effect without repeat saves, and `effectNotActive` for a `SaveGatedPhaseAwaitingOnset` or `SaveGatedPhaseEnded` state; nothing is rolled. | package design | unhappy (state error) |
| FR-062 | A repeat save with no save supplied MUST return `ResolveRepeatSaveSaveRequired` with the `SaveGatedRollPlan` (FR-030); a supplied save MUST be validated as in FR-039; the damage amount as in FR-042 and FR-089 against the reapplied failure's instances. A `SaveGatedSaveForcedFailure` of kind `savingThrow` counts as a failed repeat save: FR-058 or FR-059 applies (the effect continues; no further effect under `SaveGatedRepeatFailureNoFurtherEffect`), only its kind is checked (no DC exists, FR-039) and no `SaveGatedFailedByAtLeast` condition is returned (FR-038). | SRD 5.2.1, "Saving Throws", p. 7 | unhappy (invalid input) |
| FR-063 | The state MUST expose the effect's `SaveGatedRepeatTrigger` (`SaveGatedRepeatAtTurnStart`, `SaveGatedRepeatAtTurnEnd`, `SaveGatedRepeatAfterInterval(amount, unit)`, `SaveGatedRepeatAfterLongRest`) so the host can schedule the call. The engine MUST NOT verify the timing. | SRD 5.2.1, "Burnt Othur Fumes", "Crawler Mucus", p. 197; "Pale Tincture", p. 198; "Cackle Fever", p. 194 | happy |
| FR-084 | When a successful repeat save reaches `endsAfterSuccesses`, the events MUST be `[SaveGatedEffectRepeatSaveMade(result, successes: final count), SaveGatedEffectEnded(successesReached)]`. A `Continues` result emits only the `RepeatSaveMade`. | package design; SRD 5.2.1, "Pale Tincture", p. 198 | happy |
| FR-085 | `ResolveRepeatSave` MUST check the state before the definition: an awaiting or ended state returns `effectNotActive` even without repeat saves; `noRepeatSaves` only for an active state. A save supplied with a rejected state is not validated. | package design | unhappy (state error) |
| FR-108 | A successful or automatically successful (FR-060) repeat save under `SaveGatedRepeatSuccessReducesExhaustion(levels)` MUST read `SaveGatedSubject.Exhaustion` (an `ExhaustionLevel` plus a `known` flag, absent means none; ignored under any other effect). Absent: `Continues` with empty consequences, nothing ends. Otherwise the consequences hold `SaveGatedExhaustionReduction(r)`, `r` the smaller of `levels` and the current level (never below 0); when `levels` is at least the current level the result is `Ends` with `SaveGatedPhaseEnded(successesReached)` and events as FR-084, else `Continues`. `successes` stays 0, no dice are requested, a reduction is handed back and never applied (FR-046), and `SaveGatedEffectEnded` never lists it (FR-067). A failed save follows FR-058. | SRD 5.2.1, "Sewer Plague", "Fighting the Contagion" ("decreases by 1 ... reduced to 0, the contagion ends"), p. 194; "Exhaustion", p. 181 | unhappy (boundary) |
| FR-109 | A sibling MAY build an `ActiveSaveGatedEffect` directly in `SaveGatedPhaseActive(0)` for a repeat-only effect (RE-K, RE-P) whose initial exposure its own rules decided; the evaluator treats it like any other active state (FR-055 to FR-063). | SRD 5.2.1, "Cackle Fever", "Fighting the Contagion", p. 194 | unhappy (state error) |

Case table 4.6a, repeat sequences (S success, F failure; dice fixed; FR-055 to FR-063, FR-084, FR-085, FR-087, FR-089, FR-095, FR-108, FR-109; rows asserting a roller request or a returned `Rejected` are application tests, the rest unit):

| Label | Effect and sequence | Expected | FR |
|---|---|---|---|
| single success ends | RE-H, S | `Ends`; `Ended(successesReached)`; events `[RepeatSaveMade(succeeded, 1), Ended(successesReached)]` | FR-056, FR-084 |
| failure, nothing reapplied | RE-H, F | `Continues`; empty consequences; successes 0 | FR-059 |
| two successes; third ends | RE-B, S S; S S S | `Continues`, successes 2; `Ends` on the third; first call's events `[RepeatSaveMade(succeeded, 1)]` | FR-055, FR-056, FR-084 |
| cumulative | RE-B, S F S F S | `Ends` on the fifth call | FR-057 |
| failure reapplies | RE-B, F with 1d6 rolling 4 | `Continues`; `[4 Poison]`; successes 0; no conditions; events `[RepeatSaveMade(failed, 0)]` only | FR-058, FR-095 |
| six successes; seventh ends | RE-C, S x6; S x7 | `Continues`, successes 6; `Ends` on the seventh | FR-055, FR-056 |
| reapplied failure stays unhealable | RE-C, F | `[Poison]`; `unhealableWhileEffectLasts`; no conditions | FR-045, FR-058 |
| triggers exposed | RE-C; RE-B; RE-H; RE-K | `SaveGatedRepeatAfterInterval(24, hours)`; `AtTurnStart`; `AtTurnEnd`; `AfterLongRest` | FR-063 |
| automatic success counts | RE-S2 with Immunity to Poison | `Continues`; successes +1; event result `autoSucceeded`; no dice | FR-060 |
| no repeat saves | RE-A active | `Rejected(noRepeatSaves)` | FR-061 |
| ended or awaiting state | RE-B in `Ended(successesReached)`; RE-G in `AwaitingOnset`; RE-A in `Ended(durationElapsed)` | `Rejected(effectNotActive)` (state first) | FR-061, FR-085 |
| no save supplied; wrong save supplied | RE-B active; `CheckResult` against DC 12, `ForcedFailureResult` kind `abilityCheck` | `SaveRequired`, no dice; `Rejected(saveDoesNotMatchEffect)`, no dice | FR-062, FR-039 |
| forced failure of kind savingThrow (DC not checked, no margin condition) | RE-B, 1d6 rolls 4; a synthetic reapplied failure with a `SaveGatedFailedByAtLeast` condition | `Continues`; `[4 Poison]`, successes 0; the base conditions only | FR-062, FR-058, FR-038 |
| keeps the duration | RE-S7 active (3 hours rolled earlier), F, 1d6 rolls 4 | `Continues`; `[4 Poison]`; duration 3 hours; successes 0; exactly one die (the 1d6, not the 1d4) | FR-095, FR-087 |
| keeps the successes | RE-B, S, F | `Continues`; successes 1 | FR-095, FR-057 |
| repeat dice | RE-B failed / succeeded; RE-H failed; RE-S7 failed | `[1d6]`; `[]`; `[]`; `[1d6]` (never a duration) | FR-087 |
| shared damage | RE-S8 active, F, `Shared([4 Poison])`; `Shared([3 Bludgeoning, 9 Piercing])` | `Continues`, `[4 Poison]`, no dice; `Rejected(sharedDamageMismatch)` | FR-042 |
| printed average | RE-B, F, `SaveGatedDamagePrintedAverage` | `Continues`; `[3 Poison]`; no dice | FR-041 |
| Exhaustion reduction | RE-P, S at level 3 / 1 / none; a `ReducesExhaustion(3)` variant at level 2 | `Continues`, `SaveGatedExhaustionReduction(1)`, successes 0 / `Ends`, reduction 1, events `[RepeatSaveMade, Ended(successesReached)]` / `Continues`, empty / `Ends`, reduction 2; no dice | FR-108, FR-084 |
| Exhaustion reduction, failure | RE-P, F at level 2 | `Continues`, gain 1 `untilRemoved`, no reduction | FR-058, FR-108 |
| printed average absent | RE-S6, F; RE-S6, S | `Rejected(noPrintedAverage)`, no dice, state unchanged; `Continues`, not rejected | FR-089 |

### 4.7 Ending an effect

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-064 | `EndSaveGatedEffect` with `durationElapsed` or `neutralized` on a `SaveGatedPhaseActive` or `SaveGatedPhaseAwaitingOnset` state MUST return `EndSaveGatedEffectEnded` with `SaveGatedPhaseEnded(cause)` and one `SaveGatedEffectEnded`. | SRD 5.2.1, "Midnight Tears", pp. 197-198; "Cackle Fever", p. 194 | happy |
| FR-065 | `EndSaveGatedEffect` on an already `SaveGatedPhaseEnded` state MUST return `Rejected` with `effectNotActive`. | package design | unhappy (state error) |
| FR-066 | `EndSaveGatedEffectCommand` MUST take a `SaveGatedEndRequest` (`durationElapsed`, `neutralized`), not a `SaveGatedEffectEndCause`: `successesReached` and `instantaneous` are engine-decided and cannot be constructed as input, so no runtime check exists. | package design | unhappy (invalid input) |
| FR-067 | `SaveGatedEffectEnded` MUST list the `ConditionKind`s granted over the effect's life (unique, first-granted order) and the total Exhaustion levels with lifetime `untilEffectEnds`, so the host can remove them. The state MUST accumulate both from every consequence set it produced. | SRD 5.2.1, "Crawler Mucus", p. 197; "Cackle Fever", p. 194 | happy |

Case table 4.7a:

| Label | State and input | Expected | FR |
|---|---|---|---|
| neutralized while active | RE-C active (granted `[poisoned]`); `neutralized` | `Ended`; event conditions `[poisoned]`; Exhaustion to remove 0 | FR-064, FR-067 |
| duration elapsed | RE-A active after failure | `Ended`; event conditions `[poisoned]` | FR-064, FR-067 |
| neutralized before onset; two conditions | RE-G `AwaitingOnset`; RE-H active | `Ended`, no conditions, Exhaustion 0; event conditions `[poisoned, paralyzed]` | FR-064, FR-067 |
| Exhaustion until effect ends | RE-S3 after one failure and one reapplied failure | Exhaustion to remove 2 | FR-067 |
| Exhaustion not removed | RE-E-shaped lifetime `untilRemoved` | Exhaustion to remove 0 | FR-067 |
| ended twice; request type | RE-C in `Ended(neutralized)`; `SaveGatedEndRequest.values` | `Rejected(effectNotActive)`; exactly `durationElapsed`, `neutralized`, each mapping to the same-named cause | FR-065, FR-066 |

### 4.8 Purity, ports and telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-068 | Domain functions MUST be pure and deterministic: the same effect, subject, save and rolled values give an equal result, and no input is altered. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (determinism); unhappy (immutability) |
| FR-069 | A dice port failure MUST be returned from every rolling use case wrapped exactly once as `*srderr.PortError` (`srderr.WrapPort(dice.Port, "roll", err)`, errors-and-results.md FR-031), so `errors.Is` and `errors.As` still reach the roller's own error; no partial result is returned (the result is nil) and the input state is unchanged. A cancelled `ctx` reaches the caller as `context.Canceled` (errors-and-results.md FR-054). | `doc/rules/architecture.md` §4; dice-and-rolls.md FR-052 | unhappy (failure) |
| FR-070 | Dice MUST be requested through `dice.Roll`, one request per expression of `DiceRequired` (FR-087), in this order: each rolled instance count (FR-111), then each `SaveGatedDamage` instance of the applied failure in list order, then the duration (when rolled); at `Deferred`, only the onset (when rolled); at a repeat save, only the reapplied damage; for `RollSaveGatedDamage`, only the selected failure's damage. | dice-and-rolls.md FR-050 to FR-053 | happy |
| FR-071 | Every use case MUST return an equal result with the default no-op telemetry and with a recording one. | `doc/rules/observability.md` §9 | happy |
| FR-072 | Each use case call MUST emit exactly one span and increment `srd5e.toolbox.operations` once, with the attributes of section 7, including a call ending in a dice port error (FR-078); names are the `toolbox` signal constants. Attribute values come from the domain result (`ability`, `saveResult`, `totalDamage`, FR-092); no use case records another's signals (shared-types-and-ownership.md FR-044). Attributes per variant: FR-093. | `doc/rules/observability.md` §5, §6, §7 | happy |
| FR-073 | A `...Rejected` result MUST set `srd5e.outcome` to `failure` and the span status to error, counted once, with no log and no `error.type` (errors-and-results.md FR-043). `SaveRequired`, `AutoSucceeded`, `Saved`, `Failed`, `Deferred`, `Continues` and `Ends` are `success`: a failed save is a normal outcome. | `doc/rules/observability.md` §6, §7 | unhappy (failure) |
| FR-074 | A dice port failure MUST be recorded once at the use-case boundary: `span.RecordError(err)` once on the returned `*srderr.PortError`, one increment of `srd5e.toolbox.errors` with `srd5e.toolbox.operation` and `error.type` = `srderr.ErrorType(err)` (`dice_roller_error`), and no log. | `doc/rules/observability.md` §7; errors-and-results.md FR-034 | unhappy (failure) |
| FR-075 | Telemetry attributes MUST be low-cardinality: the `CombatantID`, the effect's contents and any amount MUST NOT be an attribute. | `doc/rules/observability.md` §5 | unhappy (invalid input) |
| FR-078 | A call ending in a dice port failure MUST still increment `srd5e.toolbox.operations` once (`srd5e.outcome` `failure`, `srd5e.toolbox.result` `error`) and record `srd5e.toolbox.operation_duration`; the wrapped error is then returned (FR-069). A rejection never increments `errors`. Use cases return no `ErrInvalidInput` error: invalid host input is a rejection (FR-039, FR-042, FR-062) and a malformed definition fails in its constructor. | `doc/rules/observability.md` §6, §7 | unhappy (failure) |
| FR-079 | `srd5e.toolbox.saves` MUST be incremented once per decided save with `srd5e.toolbox.save_ability` and `srd5e.toolbox.save_result` (`auto_succeeded`, `succeeded`, `failed`), initial and repeat, including a forced failure (`failed`); NOT for `save_required`, `notRequired` (FR-107), a rejection, `Deferred` or a dice port error. | `doc/rules/observability.md` §5 | happy; unhappy (failure) |
| FR-080 | `srd5e.toolbox.damage_rolled` MUST record, once per call whose result holds a non-empty `damage` list (printed, shared and `roll_save_gated_damage` included), the result's `totalDamage` (FR-092), in hit points, **counted once per call**: a `SaveGatedDamageShared` list is one amount however many targets a call resolves (a sibling resolving several targets in one call, such as `resolve_trap_effect`, records it once, not once per target); nothing for a call with no damage, a `Rejected` result or an error. | `doc/rules/observability.md` §5 | happy; unhappy (empty and minimal) |
| FR-081 | `srd5e.toolbox.operation_duration` MUST be recorded once per call, in seconds, through `telemetry.StartTimer` and the injected clock, with `srd5e.toolbox.operation`, for every outcome including a rejection and a dice port error. | `doc/rules/observability.md` §5 | happy |
| FR-086 | `ToolboxEvent` MUST be an interface with an unexported marker method, defined once (`toolbox/toolbox_event.go`) and exported. Each `toolbox` spec's event family MUST be a closed interface that embeds `ToolboxEvent` and lists its variants in its doc comment. This spec's family is `SaveGatedEvent` (`SaveGatedEffectStarted`, `SaveGatedEffectRepeatSaveMade`, `SaveGatedEffectEnded`), in `toolbox/save_gated_event.go`. A sibling declares its own family (for example `PoisonEvent`); none adds an event to `SaveGatedEvent`. Each event has one defining spec and a past-tense name. There is no kernel `DomainEvent` (shared-types-and-ownership.md FR-001). | shared-types-and-ownership.md FR-018, FR-019, FR-021 | happy |
| FR-092 | Every result variant that decided a save MUST carry its facts so the use case re-derives nothing: `AutoSucceeded`, `Saved`, `Failed`, `ResolveRepeatSaveContinues` and `Ends` carry `ability` and `saveResult: SaveGatedSaveResult`; `SaveRequired` variants carry the `SaveGatedRollPlan`. Constructors MUST reject (an error wrapping `srderr.ErrInvalidInput`) a value the variant cannot hold: `AutoSucceeded` holds `autoSucceeded`, `Saved` `succeeded`, `Failed` `failed` (or `notRequired` with no `ability`, and only then, FR-107), `Ends` `succeeded` or `autoSucceeded`, `Continues` any of the three. A forced failure is `failed`. `Ends` carries `consequences` (empty unless FR-108). `SaveGatedConsequences` MUST carry `totalDamage`, the sum of its `Damage` amounts before adjustment (0 when none), computed in the domain; `RollSaveGatedDamageRolled` carries the same. | `doc/rules/observability.md` §3, §5; shared-types-and-ownership.md FR-044 | happy; unhappy (invalid input) |
| FR-093 | Each use case MUST attach exactly the attributes and counters of table 4.8a: `operations` carries `srd5e.toolbox.operation`, `srd5e.outcome`, `srd5e.toolbox.result`; the two resolve spans also carry `srd5e.toolbox.save_ability` only when the result carries an ability (every decided-save variant and every `SaveRequired`); `Rejected`, `Deferred`, `EndSaveGatedEffectEnded`, `RollSaveGatedDamageRolled` and a dice port error carry none. The `end_save_gated_effect` and `roll_save_gated_damage` spans never carry it. | `doc/rules/observability.md` §3, §5, §6, §7 | happy; unhappy (failure) |

Case table 4.8a, telemetry per result (one application test per row, recording doubles; the `srd5e.toolbox.` prefix omitted; FR-072, FR-073, FR-078 to FR-080, FR-092, FR-093):

| Result | `operations` (`outcome`, `result`) | Span `save_ability` | `saves` | `damage_rolled` |
|---|---|---|---|---|
| `ResolveSaveGatedEffectAutoSucceeded` (RE-E, Resistance to Cold) | `success`, `auto_succeeded` | `constitution` | +1, `constitution`, `auto_succeeded` | none |
| `...SaveRequired` (either use case) | `success`, `save_required` | the plan's `ability` | none | none |
| `...Saved` (RE-A half damage; RE-B no damage) | `success`, `saved` | `constitution` | +1, `succeeded` | the `totalDamage`; none |
| `...Failed` (RE-A; forced failure; RE-E without damage; RE-N) | `success`, `failed` | `constitution`; absent for RE-N | +1, `failed`; none for RE-N | the `totalDamage`; none for RE-E, RE-N |
| `...Deferred` | `success`, `deferred` | absent | none | none |
| `ResolveRepeatSaveContinues` (RE-B S; F; RE-S2 Immunity; RE-P S) | `success`, `continues` | `constitution` | +1, `succeeded`; `failed`; `auto_succeeded`; `succeeded` | none; the `totalDamage` on F; none; none |
| `ResolveRepeatSaveEnds` | `success`, `ends` | `constitution` | +1, `succeeded` | none |
| `ResolveSaveGatedEffectRejected` | `failure`, `rejected_effect_not_awaiting_onset`, `rejected_save_does_not_match_effect`, `rejected_shared_damage_mismatch`, `rejected_no_printed_average`; span error | absent | none | none |
| `ResolveRepeatSaveRejected` | `failure`, `rejected_no_repeat_saves`, `rejected_effect_not_active`, `rejected_save_does_not_match_effect`, `rejected_shared_damage_mismatch`, `rejected_no_printed_average`; span error | absent | none | none |
| `EndSaveGatedEffectEnded` / `Rejected` | `success`, `ended` / `failure`, `rejected_effect_not_active` | not an attribute | none | none |
| `RollSaveGatedDamageRolled` (RE-S1; RE-E) | `success`, `rolled` | not an attribute | none | the `totalDamage`; none |
| dice port error from any rolling use case | `failure`, `error`; span error; `errors` +1 (`dice_roller_error`) | absent | none | none |
| `AutoSucceeded` built with `saveResult` `failed` | `ErrInvalidInput` error | n/a | n/a | n/a |

### 4.9 `ToolboxRuleOptions`

Moved here from `rule-options.md` (shared-types-and-ownership.md FR-057); it follows that spec's pattern (FR-023, FR-029, FR-033, FR-034, FR-038). The SRD has no "optional rules" section, but p. 196 offers the Fear Effects riders and half damage on a successful Mental Stress save as Game Master choices, so those are the options. The Ingested-poison partial dose (p. 197) is a per-exposure input of `toolbox-poison.md` and Prolonged Effects (p. 196) a per-event choice in `toolbox-fear-and-mental-stress.md`; neither is a field here. Consumers: `toolbox-fear-and-mental-stress.md` FR-100 and FR-029. No requirement of this spec reads an option.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-099 | `ToolboxRuleOptions` MUST hold exactly two fields, of types defined in this spec (FR-102 to FR-104): `fear` (`FearOptions`, default `FearOptions.baseline`) and `mentalStressOnSuccess` (`MentalStressOnSuccess`, default `MentalStressNoDamageOnSuccess`). It MUST NOT hold an on/off flag or a second rider or half-damage type. `ToolboxRuleOptions.standard` MUST equal the instance built from both defaults (the SRD baseline: Frightened as the baseline effect; no damage on a success unless the GM chooses half). | SRD 5.2.1, "Fear Effects", "Mental Stress Effects", p. 196; shared-types-and-ownership.md FR-045 | happy |
| FR-100 | `ToolboxRuleOptions` MUST be a plain comparable value (`==` works), equal by value on both fields (riders in a different order are equal because `FearOptions` is a set), with methods `WithFear(FearOptions)` and `WithMentalStressOnSuccess(MentalStressOnSuccess)` returning a changed copy. Its zero value is the SRD baseline (no riders, no damage on a success), so the type has no invalid state and no `NewToolboxRuleOptions`. Shape: `type ToolboxRuleOptions struct { Fear FearOptions; MentalStressOnSuccess MentalStressOnSuccess }`. | architecture.md §3; shared-types-and-ownership.md FR-045 | happy; unhappy (rule interaction) |
| FR-101 | The riders and the half damage MUST reach `toolbox-fear-and-mental-stress.md` use cases only as the `ToolboxRuleOptions` field of the command, never as a static, top-level or environment setting. Rider behaviour (`fearRiderEffects`, `checkFrightenedTurn`) is owned by that spec; the option types are owned here. | shared-types-and-ownership.md FR-045, FR-048; architecture.md §3, §10 | unhappy (rule interaction) |
| FR-102 | `FearRider` MUST be an enum of exactly the three p. 196 riders: `dashEachTurn`, `attackersHaveAdvantage`, `oneActionKindPerTurn`. Behaviour is owned by `toolbox-fear-and-mental-stress.md`. | SRD 5.2.1, "Fear Effects", p. 196 ("Consider these examples") | happy |
| FR-103 | `FearOptions` MUST be a comparable value object holding the set of riders supplied (a bit set, so mutating the input slice or a returned slice MUST NOT change it), equal by value regardless of order; every subset of the three riders (all 8) MUST be constructible with `NewFearOptions(riders ...FearRider)`, which returns an error wrapping `srderr.ErrInvalidInput` for a value that is not one of the three riders. The zero `FearOptions` is the baseline: it has no riders and adds no effect beyond Frightened, and `FearOptions{}` MUST equal `NewFearOptions()`. | SRD 5.2.1, "Fear Effects", p. 196; `doc/rules/architecture.md` §4 | happy; unhappy (boundary, immutability) |
| FR-104 | `MentalStressOnSuccess` MUST be a typed enum with a `String` method and exactly two values, `MentalStressNoDamageOnSuccess` (the zero value, default) and `MentalStressHalfDamageOnSuccess`; no variant carries data, so no interface is needed. | SRD 5.2.1, "Mental Stress Effects", p. 196; `doc/rules/architecture.md` §3 | happy |

### 4.10 Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-003, FR-004, FR-007 | Withdrawn: moved to architecture-guardrails.md (FR-003: FR-018, FR-019, FR-020, FR-098; FR-004: FR-021; FR-007: FR-097, import legality FR-018, FR-019) |

## 5. Model

Everything in the domain files of package `toolbox` (every file not named `*_usecase.go`) is pure, synchronous and silent.

- **Referenced, not defined:** `ability.Ability`, `modifier.RollCircumstance` (`Disadvantage`), `dice.Expression` (the kernel packages); `DifficultyClass`, `CheckResult`, `ForcedFailureResult`, `Kind`, `Outcome` (`d20`); `ConditionKind`, `DamageType`, `DurationUnit` (`content`); `Damage`, `DamageAdjustment`, `DamageAffinities`, `CombatantID`, `ExhaustionLevel` (`combat`).
- **The effect:** `SaveGatedEffect` (FR-008) with `SaveGatedRequirement` (closed interface: `SaveGatedSaveRequired(ability, difficultyClass)`, `SaveGatedNoSave`); `SaveGatedCircumstance` (enum; `wearingMediumOrHeavyArmor`, Disadvantage only: Advantage is a roll-mode source of the host, section 9); `SaveGatedFailure(damage: []SaveGatedDamage, conditions: []SaveGatedCondition, exhaustion: SaveGatedExhaustion)`; `SaveGatedDamage(dice, type, printedAverage: SaveGatedPrintedAverage, instances: SaveGatedInstances)`; `SaveGatedInstances` (closed interface: `SaveGatedOneInstance` default, `SaveGatedRolledInstances(count: dice.Expression)`; FR-110); `SaveGatedPrintedAverage` (closed interface: `SaveGatedNoPrintedAverage`, `SaveGatedPrintedAverageOf(amount)`); `SaveGatedCondition(kind, margin: SaveGatedFailureMargin)`; `SaveGatedFailureMargin` (closed interface: `SaveGatedAnyFailure`, `SaveGatedFailedByAtLeast(margin)`); `SaveGatedExhaustion` (closed interface: `SaveGatedNoExhaustion`, `SaveGatedExhaustionGain(levels, lifetime)`, `SaveGatedExhaustionReduction(levels)`, the last only in a `SaveGatedConsequences`, FR-106); `SaveGatedExhaustionLifetime` (`untilRemoved`, `untilEffectEnds`); `SaveGatedDamageHealing` (`healable`, `unhealableWhileEffectLasts`; an enum so no flag stands in for a state); `SaveGatedSuccess` (closed interface: `SaveGatedNoEffectOnSuccess`, `SaveGatedHalfDamageOnSuccess`); `SaveGatedRepeat` (closed interface: `SaveGatedNoRepeat`, `SaveGatedRepeatSaves(trigger, onSuccess, endRule, onFailure)`); `SaveGatedRepeatSuccess` (closed interface: `SaveGatedRepeatSuccessCounts`, `SaveGatedRepeatSuccessReducesExhaustion(levels)`); `SaveGatedEndRule` (closed interface: `SaveGatedEndsAfterSuccesses(endsAfterSuccesses)`, `SaveGatedEndsWhenExhaustionReachesZero`; FR-106); `SaveGatedRepeatTrigger` (FR-063); `SaveGatedRepeatFailure` (closed interface: `SaveGatedRepeatFailureNoFurtherEffect`, `SaveGatedRepeatFailureReapplies(SaveGatedFailure)`); `SaveGatedOnset` (closed interface: `SaveGatedOnsetImmediate`, `SaveGatedOnsetAfter(SaveGatedTime)`, `SaveGatedOnsetAtMidnight`); `SaveGatedDuration` (closed interface: `SaveGatedDurationInstantaneous`, `SaveGatedDurationFor(SaveGatedTime)`, `SaveGatedDurationUntilEnded`); `SaveGatedTime` (closed interface: `SaveGatedFixedTime(amount, DurationUnit)`, `SaveGatedRolledTime(dice.Expression, DurationUnit)`).
- **A run:** `SaveGatedSubject(affinities, circumstances: []SaveGatedCircumstance (treated as a set), exemption: SaveGatedExemption, exhaustion: ExhaustionLevel (optional))` with `SaveGatedExemption` closed interface (`SaveGatedNoExemption`, `SaveGatedExempt`); `SaveGatedRollPlan(ability, difficultyClass, circumstances)`; `SaveGatedSave` (closed interface: `SaveGatedSaveRolled(CheckResult)`, `SaveGatedSaveForcedFailure(ForcedFailureResult)`, `SaveGatedSaveNotRolled`); `SaveGatedDamageAmount` (closed interface: `SaveGatedDamageRolled`, `SaveGatedDamagePrintedAverage`, `SaveGatedDamageShared([]Damage)`); `SaveGatedStage` (closed interface: `SaveGatedInitialExposure(effect, target)`, `SaveGatedOnsetReached(pending)`); `SaveGatedDiceStage` (FR-087); `SaveGatedDamageSource` (`initialFailure`, `reappliedFailure`; FR-043, FR-096); `SaveGatedConsequences(damage, adjustments: []DamageAdjustment, conditions: []ConditionKind, exhaustion, healing)` with derived `totalDamage` (FR-092); `SaveGatedSaveResult` (`autoSucceeded`, `succeeded`, `failed`, `notRequired`); `SaveGatedEffectEndCause` (`successesReached`, `instantaneous`, `durationElapsed`, `neutralized`); `SaveGatedEndRequest` (`durationElapsed`, `neutralized`; FR-066); `SaveGatedRejection` (`effectNotAwaitingOnset`, `effectNotActive`, `noRepeatSaves`, `saveDoesNotMatchEffect`, `sharedDamageMismatch`, `noPrintedAverage`; each `Rejected` variant accepts its own subset, FR-097).
- **State:** `ActiveSaveGatedEffect(effect, target: CombatantID, phase, duration, grantedConditions: []ConditionKind, exhaustionToRemove: int)`; `phase` is a closed interface (`SaveGatedPhaseAwaitingOnset(onset: SaveGatedResolvedOnset)`, `SaveGatedPhaseActive(successes)`, `SaveGatedPhaseEnded(cause)`); `SaveGatedResolvedOnset` is a closed interface (`SaveGatedOnsetIn(SaveGatedFixedTime)`, `SaveGatedOnsetAtMidnightResolved`). Immutable; the host stores it (no aggregate).
- **Events:** `SaveGatedEffectStarted(target)`, `SaveGatedEffectRepeatSaveMade(target, result, successes)`, `SaveGatedEffectEnded(target, cause, conditions, exhaustionLevelsToRemove)`, in the family `SaveGatedEvent` implementing `ToolboxEvent` (FR-086).
- **Domain services** (stateless zero-size struct types usable as `SaveGatedPlanner{}`, with `DiceRequired` a package function; exported so siblings compose them, shared-types-and-ownership.md FR-044): `SaveGatedPlanner` (FR-024 to FR-029, FR-076, FR-098), `DiceRequired` (FR-087), `SaveGatedEvaluator` (FR-033 to FR-038, FR-055 to FR-060, FR-064 to FR-067, FR-089, FR-095, FR-107, FR-108, FR-111). These three are the names a sibling cites and composes (toolbox-traps.md calls `SaveGatedPlanner`, `DiceRequired` and `SaveGatedEvaluator.EvaluateInitial` from its pure `TrapResolver`, never a use case). The evaluator exposes three pure operations, each taking already-rolled dice, which a sibling composes instead of calling a use case: `EvaluateInitial` (initial and onset resolution, behind `ResolveSaveGatedEffect`), `EvaluateRepeat` (behind `ResolveRepeatSave`) and `EvaluateEnd` (behind `EndSaveGatedEffect`). A use case calls planner, validation, `DiceRequired`, roll, evaluator in that order.
- **Failures:** a definition that cannot be built (FR-009 to FR-023, FR-090, FR-091, FR-094, FR-097, FR-105, FR-106) is an error wrapping `srderr.ErrInvalidInput` from its constructor. A run-time host value failing validation (wrong DC, misfit shared list, wrong state) is a returned `Rejected`. The only port failure is dice port error; use cases throw nothing else.

### 5.1 The model against the SRD evidence

| Entry | Page | Expressed as | Gap (owner) |
|---|---|---|---|
| Assassin's Blood | 197 | RE-A | The Ingested-poison partial dose is a per-use choice; `toolbox-poison`. |
| Burnt Othur Fumes; Pale Tincture; Spider's Sting | 197-198 | RE-B; RE-C; RE-D | None. |
| Crawler Mucus | 197 | RE-H; both conditions end with the effect (FR-067) | None. |
| Midnight Tears | 197-198 | RE-G; neutralized through `EndSaveGatedEffect` (FR-064) | Whether the creature is Poisoned before midnight is not stated; `toolbox-poison`. |
| Essence of Ether, Oil of Taggit | 197-198 | Conditions yes | "Wakes up if it takes damage": the host reports it; the early end of one condition is the sibling's (Q-11); `toolbox-poison`. |
| Torpor, Truth Serum | 198 | Poisoned with a duration (RE-J) | Speed halved; cannot knowingly lie: non-condition riders are the sibling's (Q-11); `toolbox-poison`. |
| Poisoned Needle | 200-201 | 1d10 Poison and Poisoned, half on a success, 1 hour | The trap's duration, triggers, Avoid, Detect, Disarm; `toolbox-traps`. |
| Cackle Fever, Uncontrollable Laughter | 194 | RE-I | Trigger ("each time it takes damage other than Psychic damage"); `toolbox-curses-and-contagions`. |
| Cackle Fever Fever; Sewer Plague Fatigue | 194 | RE-N (`SaveGatedNoSave`, FR-105, FR-107); Exhaustion lifetime `untilEffectEnds` for the Fever | None. |
| Cackle Fever, Fighting the Contagion | 194 | RE-K (placeholder failure; FR-009 forbids an empty one), built directly (FR-109) | Immunity for 1 year (`DurationUnit` has no years): `toolbox-curses-and-contagions`. |
| Deep Water, Extreme Cold, Frigid Water | 195 | RE-E shape | Exemptions "lacks a Swim Speed", "naturally adapted": arrive as `SaveGatedExempt` (FR-076); `toolbox-environmental-effects`. |
| Extreme Heat | 195 | RE-F | DC 5 rising 1 per hour: the sibling builds the effect for that hour. |
| Sewer Plague | 194 | Fatigue is RE-N; "Fighting the Contagion" is RE-P (daily; a failure gains 1 Exhaustion, a success reduces it, ending at level 0: FR-106, FR-108) | The infection save is the sibling's. |
| Sight Rot | 194 | DC 15 Constitution, Blinded, until ended | Suppression for 24 hours, three doses ending it, 24 hour immunity on a successful save: the sibling's (Q-16). |
| Demonic Possession | 193 | DC 15 Charisma; a Long Rest save ends the effect (`SaveGatedRepeatSaves(SaveGatedRepeatAfterLongRest, ...)`, one success); Dispel Evil and Good is `EndSaveGatedEffect(neutralized)` | The turn-end save that only returns control, and the non-condition control rider: the sibling's (Q-16). |
| Fear Effects | 196 | Wisdom, DC from the Sample Fear DCs; Frightened; repeat at turn end, 1 success, until ended | The Frightened source is attached by the host (Q-5; toolbox-fear-and-mental-stress.md FR-010). |
| Sample Mental Stress Effects | 196 | RE-M: Wisdom (or Intelligence or Charisma), DC 10, 15 or 20; 1d6, 3d6 or 9d6 Psychic, `SaveGatedNoPrintedAverage`; none, or half at the GM's discretion | Prolonged effects are not save-gated. In printed-average mode this effect is rejected (FR-089), by design. |
| Rest and Recuperation | 194 | (not an effect) | Advantage on saves for 24 hours: the host's roll-mode source, `SaveGatedCircumstance` is Disadvantage only (Q-11). |
| Slippery Ice | 195 | DC 10 Dexterity, Prone, instantaneous | The trigger and Difficult Terrain; `toolbox-environmental-effects`. |
| Extended Travel | 192 | Constitution, 1 Exhaustion, instantaneous, DC for the hour (10 plus 1 per hour past 8) | The sibling computes the DC; `toolbox-travel-pace`. |
| Poisoned Darts | 200 | DC 13 Dexterity, 3 (1d6) Poison per dart, `SaveGatedRolledInstances(1d3)` (FR-110, FR-111) | None; `toolbox-traps` builds the effect. |
| Falling Net | 199 | DC 10 Dexterity, Restrained, until ended (RE-S4) | The Huge exemption arrives as `SaveGatedExempt` (FR-076), decided by `toolbox-traps`. |

### 5.2 Reconciliation with the sibling specs

The types here win; a sibling counterpart is replaced or kept only as a thin builder of a `SaveGatedEffect` or a wrapper of the state. This spec edits no sibling; the edits that follow from section 9 are returned as amendments to each sibling.

| Sibling type (file) | Counterpart here | State |
|---|---|---|
| `HazardSaveRequirement`, `HazardSaveConsequence`, `ExemptionReason`, `NoSaveReason`, armor Disadvantage (toolbox-environmental-effects.md) | `SaveGatedRollPlan`, `SaveGatedConsequences`, `SaveGatedSubject.exemption`, `SaveGatedCircumstance` | Replaced (its FR-009, FR-089). It keeps `EnvironmentalNoSaveReason` for an unmet precondition (no save called for). Nothing owed. |
| Huge or larger exemption, `TrapSaveForDamage`, `TrapSaveForCondition` (toolbox-traps.md FR-068) | `SaveGatedEffect`, `SaveGatedExempt`, the three services | Traps keep catalogue and size test; `ResolveTrapEffect` composes the services (its FR-087). Its A-2 is met by FR-110, FR-111 (the per-dart `expand` is then removed there); A-3, A-4, A-5, A-6 are applied (section 5 services, section 7, FR-080). |
| `PoisonSaveRequest`, `PoisonTestOutcome`, `ResolvePoisonSave`, `RepeatingSave`, `RepeatSaveTrigger` (toolbox-poison.md) | `SaveGatedRollPlan`, `ResolveSaveGatedEffect`, `SaveGatedRepeatSaves`, `SaveGatedRepeatTrigger` | Replaced. `PoisonAffliction` wraps `ActiveSaveGatedEffect` (composition). Its Q-22 (2), (3) become its own signal list. |
| `ContagionEffect` (toolbox-curses-and-contagions.md) | `SaveGatedConsequences`, `SaveGatedEffectEnded` | Gone; it builds `SaveGatedEffect`s. Its HA-1, HA-2, HA-4 are accepted (FR-105 to FR-109, section 5); HA-3 is its own signal list. |
| Fear and Mental Stress save types (toolbox-fear-and-mental-stress.md) | `SaveGatedEffect`, Q-5 | Built via `saveGatedEffect` (its FR-008, FR-040). Its Q-16 (a) resolved by FR-018, FR-089; (b) by FR-086; (c) becomes its own signal list. |

Not decided here: applying Immunity to a condition, an already-held condition, and Death at Exhaustion level 6 belong to `conditions.md` and the host (FR-046).

## 6. Interface

- **Use cases** (`*_usecase.go`), synchronous, each constructed with `NewX(roller Roller, opts ...Option) (X, error)` (`WithTelemetry`, `WithClock`; default `telemetry.Noop`; an error wrapping `ErrInvalidInput` for a nil roller), each with one method `Run(ctx context.Context, cmd) (Result, error)`. The result is a closed interface with the variants listed; the `error` carries only a port failure (`*srderr.PortError`) or a cancelled `ctx`:
  - `ResolveSaveGatedEffect(ResolveSaveGatedEffectCommand(stage, subject, save, damageAmount))` returning `ResolveSaveGatedEffectResult` = `...AutoSucceeded(ability, saveResult)` | `...SaveRequired(plan)` | `...Saved(ability, saveResult, consequences, events)` | `...Failed(ability?, saveResult, consequences, active, events)` (no `ability` only for `SaveGatedNoSave`, FR-107) | `...Deferred(active, events)` | `...Rejected(reason)`.
  - `ResolveRepeatSave(ResolveRepeatSaveCommand(state, subject, save, damageAmount))` returning `ResolveRepeatSaveResult` = `...SaveRequired(plan)` | `...Continues(ability, saveResult, consequences, active, events)` | `...Ends(ability, saveResult, consequences, active, events)` | `...Rejected(reason)`.
  - `EndSaveGatedEffect(EndSaveGatedEffectCommand(state, request))` returning `EndSaveGatedEffectResult` = `...Ended(active, events)` | `...Rejected(reason)`.
  - `RollSaveGatedDamage(RollSaveGatedDamageCommand(effect, source))` returning `RollSaveGatedDamageResult` = `RollSaveGatedDamageRolled(damage, totalDamage)`; no `Rejected` variant (its only invalid input is an unbuildable command, FR-096).

  Variants are prefixed with the operation (shared-types-and-ownership.md FR-017); `reason` is a `SaveGatedRejection` field (FR-018 there).
- **Allowed rejection reasons** (FR-097):

  | `SaveGatedRejection` | `ResolveSaveGatedEffectRejected` | `ResolveRepeatSaveRejected` | `EndSaveGatedEffectRejected` |
  |---|---|---|---|
  | `effectNotAwaitingOnset` | yes (FR-051) | no | no |
  | `effectNotActive` | no | yes (FR-061, FR-085) | yes (FR-065) |
  | `noRepeatSaves` | no | yes (FR-061) | no |
  | `saveDoesNotMatchEffect` | yes (FR-039) | yes (FR-062) | no |
  | `sharedDamageMismatch` | yes (FR-042) | yes (FR-042) | no |
  | `noPrintedAverage` | yes (FR-089) | yes (FR-089) | no |

- **Domain services:** `SaveGatedPlanner`, `SaveGatedEvaluator`, `DiceRequired` (section 5), exported so a sibling use case composes them instead of calling these use cases.
- **Host protocol,** in `toolbox/doc.go` in terms of this context's value objects only (no use case or method of another context is named, so architecture-guardrails.md FR-097 has nothing to flag): call `ResolveSaveGatedEffect` with `SaveGatedSaveNotRolled`; on `SaveRequired`, make the saving throw the plan describes (its ability, DC and circumstances, combined with the creature's own roll-mode sources) and call again with `SaveGatedSaveRolled` or `SaveGatedSaveForcedFailure`. The engine checks kind and DC but not the ability (FR-039, Q-19). For an area effect call `RollSaveGatedDamage` once and pass `SaveGatedDamageShared` to each target. Apply the returned damage and adjustments, each condition kind, each Exhaustion level gained or reduced, and persist `active`; a repeat of an Exhaustion-reducing effect needs the creature's `SaveGatedSubject.exhaustion` (FR-108). Schedule `SaveGatedOnsetReached` from a `Deferred` state. Call `ResolveRepeatSave` on each trigger and `EndSaveGatedEffect` when the duration ends or the effect is neutralized, and remove what `SaveGatedEffectEnded` lists. *For the reader of this spec only:* the host rolls with `d20.Resolver` and applies with the `combat` damage and condition functions.
- **Option value:** `ToolboxRuleOptions{Fear FearOptions; MentalStressOnSuccess MentalStressOnSuccess}` (FR-099 to FR-104), the zero value as the SRD baseline (`StandardToolboxRuleOptions()` returns it), `WithFear`, `WithMentalStressOnSuccess`, value equality; `FearRider`, `FearOptions`, `MentalStressOnSuccess` (FR-102 to FR-104) are defined here and exported; `toolbox-fear-and-mental-stress.md` imports them (its FR-024 to FR-026 and FR-051 are withdrawn there by amendment).
- **Ports:** `Roller` only; a dice port failure is returned wrapped once (FR-069). No new port implementation, double or contract suite; tests use `dicetest.FixedRoller` and the `telemetrytest` recorder (shared-types-and-ownership.md FR-038).
- **Exports:** package `toolbox` exports the four use cases, commands, results, value objects, state, events, domain services and the signal-name constants (FR-006) and `ToolboxRuleOptions`. `toolbox/doc.go` lists inputs, outputs and the host protocol, and states an option is a command value, never global (`rule-options.md` FR-034).

## 7. Observability

Evaluation answers (`doc/rules/observability.md` §2): (1) what can go wrong: dice port error; a call in the wrong state, a save for a different DC or a misfit shared list (rejections, FR-039, FR-042); a wrong rule result (a missed automatic success, a misapplied margin rider). (2) The operator sees rejections and port failures by operation (`srd5e.outcome`, span status) and a drifting save mix in `srd5e.toolbox.saves`. (3) Diagnosis: operation, result, save ability and save result as attributes; amounts and kinds are in the returned consequences and events. (4) Counted and timed: operations, saves, damage rolled, errors, duration. (5) No earlier `toolbox` signals exist; `srd5e.outcome` follows shared-types-and-ownership.md FR-041; `operations`, `operation_duration` and `errors` are owned here for the whole context. (6) Monitor: any `errors` over 5 minutes; a `rejected_*` share above 5% of an operation; a shift of the `failed` share for one ability; p99 duration above twice its 7-day baseline. (7) Noise: none; no log in the normal path, a failed save is a normal outcome, failures are recorded once and not logged.

`srd5e.toolbox.operation` values: `resolve_save_gated_effect`, `resolve_repeat_save`, `end_save_gated_effect`, `roll_save_gated_damage`. Allowed `srd5e.toolbox.result` values (the `rejected_<reason>` values are exactly the pairings of the section 6 table, in snake case; no value exists for a reason a use case cannot return):

| Use case | `result` values | `srd5e.outcome` |
|---|---|---|
| `resolve_save_gated_effect` | `auto_succeeded`, `save_required`, `saved`, `failed`, `deferred` | `success` |
| `resolve_repeat_save` | `save_required`, `continues`, `ends` | `success` |
| `end_save_gated_effect` | `ended` | `success` |
| `roll_save_gated_damage` | `rolled` | `success` |
| each use case | `rejected_<reason>` (not for `roll_save_gated_damage`) | `failure` |
| every use case | `error` (a dice port error left the call, FR-078) | `failure` |
**Sibling operations** (amendments from the environmental, fear, poison and traps specs, accepted). `srd5e.toolbox.operation` also takes the values below, each with the `result` values shown (`srd5e.outcome` `success` or `failure` as the column says; `error` is a returned dice port error, FR-078, and exists only where listed; every other operation rolls nothing). Each sibling's own section 7 holds its prose.

| Operation (owner) | `result`, `success` | `result`, `failure` |
|---|---|---|
| `assess_environmental_exposure` (environmental) | `save_gated`, `no_save` | none |
| `assess_high_altitude`; `assess_thin_ice`; `describe_environmental_area_effects` | `assessed`; `ice_holds`, `ice_broken`; `described` | none |
| `roll_thin_ice_tolerance` | `rolled` | `error` |
| `resolve_fear_effect` (fear) | `save_required`, `resisted`, `frightened` | `rejected_save_does_not_match_effect` |
| `resolve_fear_repeat_save` | `save_required`, `continues`, `ended` | `rejected_effect_not_active`, `rejected_save_does_not_match_effect` |
| `resolve_mental_stress` | `save_required`, `avoided`, `halved`, `full` | `rejected_save_does_not_match_effect`, `rejected_shared_damage_mismatch`, `rejected_no_printed_average`, `error` |
| `start_prolonged_effect` | `started` | `error` |
| `elapse_prolonged_effect` | `continues`, `ended` | none |
| `change_prolonged_effect` | `suppressed`, `resumed`, `removed`, `cure_not_effective` | `rejected_already_suppressed`, `rejected_not_suppressed` |
| `expose_to_poison` (poison) | `exposed`, `not_delivered` | `rejected_delivery_does_not_match_poison`, `rejected_shared_damage_mismatch` |
| `progress_poison` | `updated`, `ended`, `rider_ended`, `no_change` | `rejected_affliction_ended`, `rejected_progress_not_applicable`, `rejected_other_affliction_of_another_creature` |
| `harvest_poison` | `succeeded`, `failed`, `failed_exposed` | `rejected`, `error` |
| `resolve_trap_effect` (traps) | `resolved` | `rejected_trap_not_armed`, `rejected_too_many_targets`, `rejected_duplicate_target`, `rejected_missing_save_outcome`, `error` |
Shared keys added by those amendments: `srd5e.toolbox.prolonged_term` (`short_term`, `long_term`, `indefinite`) on the three prolonged-effect spans only; the six-value `save_ability` also applies to the fear save spans (`resolve_fear_effect`, `resolve_fear_repeat_save`, `resolve_mental_stress`, when the result decided or planned a save) and to the mental stress histogram. `srd5e.toolbox.errors` takes `operation` `harvest_poison`, `roll_thin_ice_tolerance`, `resolve_mental_stress`, `start_prolonged_effect` and `resolve_trap_effect` with `error.type` `dice_roller_error`.

**Shared contract for the whole `toolbox` context** (settled in section 9: each sibling lists its own operation and `result` values, this spec owns the shared keys and instruments):

- `srd5e.toolbox.errors` is one counter with `srd5e.toolbox.operation` and `error.type`: `dice_roller_error` (this spec) and `content_error` (toolbox-combat-encounter-difficulty.md). No sibling uses `srd5e.toolbox.use_case`.
- Constants live in the `toolbox` signal constants (FR-006). A sibling adds its constants to the `toolbox` signal constants and lists its operation values and `result` values in its own section 7 in the change that introduces them (this section's tables hold only this spec's use cases), and defines no second signals class. `result` and `save_ability` are shared keys (the union of per-use-case values, each value belonging to one operation); `save_ability` is `strength`, `dexterity`, `constitution`, `intelligence`, `wisdom` or `charisma`, a sibling using a subset. `srd5e.toolbox.save_result` is `auto_succeeded`, `succeeded` or `failed`.
- A sibling signal with its own schema carries a sibling prefix (`srd5e.toolbox.poison.*`, `srd5e.toolbox.fear.*`), never a generic name from this table.
- the `toolbox` signal constants members, by owner (spec-owned prose and thresholds stay in the owning spec): environmental, `srd5e.toolbox.environment.no_save`, `.heat_exposure_hour`, `.thin_ice_tolerance` and attributes `.environment.effect`, `.environment.reason`; fear, `srd5e.toolbox.fear.frightened` with `.fear.riders` and `srd5e.toolbox.mental_stress.damage`; poison, `srd5e.toolbox.poison.exposures` (counter) with `.poison.type` and `.poison.delivery_result`; traps, `srd5e.toolbox.trap.resolved` (counter), `.trap.targets` (histogram) and attributes `.trap.severity`, `.trap.level_band`, `.trap.level_fit`, `.trap.effect`, `.trap.rejection`. Each is one constant in the one class.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.toolbox.operations` | counter `{operation}` | `operation`; `srd5e.outcome`; `result` | Throughput and result of every use case (FR-072) | Rate by operation and result; alert if `rejected_*` exceeds 5% |
| `srd5e.toolbox.saves` | counter `{save}` | `save_ability`; `save_result` | Decided saves, initial and repeat, not `save_required` (FR-079) | By ability; investigate a 10-point drift of the `failed` share over 1,000 saves |
| `srd5e.toolbox.damage_rolled` | histogram `{hit_point}` | none | `totalDamage` per call before adjustment (FR-080, FR-092) | p99 above twice baseline; collapse to 0 means a content error |
| `srd5e.toolbox.operation_duration` | histogram `s` | `operation` | Latency of each use case, rejections and port failures included (FR-081) | p99 above twice its 7-day baseline |
| `srd5e.toolbox.errors` | counter `{error}` | `operation`; `error.type` | Port failures at the use-case boundary (returned errors only) | Any non-zero count over 5 minutes |
| `srd5e.toolbox.<operation>` (one span per use case) | span | `srd5e.outcome`; `result`; `save_ability` only on the two resolve spans and only when the result carries an ability (FR-093) | One call | Trace sampling |

(The `srd5e.toolbox.` prefix is omitted from attribute keys above.) Entries feed `doc/observability/signals.md`; `save_ability` and `save_result` are new values for new keys, so the shared sets in observability.md §5 are unchanged.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002, FR-006, FR-086 | architecture | (FR-005, `AGENTS.md` wording: doc review in the same change.) `internal/archtest`, toolbox fixtures only: the map row lists exactly the four upstream edges and no other row lists `toolbox`; one package, no sub-folders by layer; no duplicate or generic exported name; the signal-name constants, `SaveGatedPlanner`, `SaveGatedEvaluator`, `DiceRequired` exported once; `ToolboxEvent` has an unexported marker method, `SaveGatedEvent` is a closed interface whose doc comment lists every variant, an exhaustive type switch in an `Example` compiles. Generic checks belong to architecture-guardrails.md. |
| FR-008 to FR-023, FR-077, FR-088, FR-090, FR-091, FR-094, FR-097, FR-105, FR-106, FR-110 | unit | `toolbox/*_test.go`, table 4.2a; equality (`==` or `Equal`) and immutability tests (FR-008, FR-019, FR-088); assert `errors.Is` `ErrInvalidInput`, `errors.As` of the typed problem and its value; FR-097 asserts every pairing of the section 6 table. |
| FR-024 to FR-029, FR-076, FR-098; FR-033 to FR-038, FR-052; FR-087; FR-044 to FR-047, FR-082, FR-083, FR-092 | unit | Planner, evaluator and `DiceRequired` with already-rolled values (tables 4.3a planner rows, 4.4a *unit* rows, 4.4b, rows named "dice"); value fields and events; FR-046 asserts result types hold only `ConditionKind`; FR-092 constructor checks and derived `totalDamage`. |
| FR-030 to FR-032, FR-076, FR-098; FR-033 to FR-036, FR-039 to FR-047, FR-082, FR-083, FR-089, FR-096, FR-107, FR-111 | application | `ResolveSaveGatedEffect` and `RollSaveGatedDamage` with `dicetest.FixedRoller`: exact variant, `Requests()` empty for `AutoSucceeded` and `SaveRequired`, a returned `Rejected`, events, the shared roll fed to several targets (tables 4.3a "no dice" rows, 4.4a *application* rows, 4.4c). |
| FR-048 to FR-054; FR-055 to FR-063, FR-084, FR-085, FR-095, FR-108, FR-109; FR-064 to FR-067 | unit + application | Tables 4.5a, 4.6a (sequences driven through the use case with scripted results; successes and kept duration at unit level, dice requests and returned `Rejected` at application level) and 4.7a. |
| FR-068 to FR-071 | unit + application | Same inputs give equal results and inputs are unchanged; a failing roller double (`dicetest` callback roller returning an error) yields one wrapped `*srderr.PortError` with state unchanged for each rolling use case; `dicetest.FixedRoller.Requests` order (FR-070); equal results with no-op and recording `telemetrytest` bundle. |
| FR-072 to FR-075, FR-078 to FR-081, FR-093 | application | Recording doubles, one test per row of table 4.8a: one span, one `operations` increment, attributes from section 7 and the result's own fields; `save_ability` present exactly where the table says; rejection is `failure` with error status, counted once, not logged; a port failure increments `operations` once (`failure`, `error`), `errors` once and `operation_duration` once, not logged; `saves` never for `save_required`, rejections, `Deferred` or a port failure; `damage_rolled` per FR-080; no attribute holds an id or amount. |
| FR-099, FR-100, FR-102 to FR-104 | unit | `FearRider` has exactly 3 values, all 8 subsets constructible and equal regardless of order, the stored set is not aliased to the input slice, the zero `FearOptions` equals `NewFearOptions()`, `MentalStressOnSuccess` has two values with `NoDamage` as zero. The zero `ToolboxRuleOptions` equals `StandardToolboxRuleOptions()`; two fields only (no on/off flag); default and non-default per field; `==` per field (riders in a different order equal); `WithFear` changes one field; no `NewToolboxRuleOptions` exists (exported-name scan). |
| FR-101 | unit, scenario | Two commands in one process with different `ToolboxRuleOptions` give independent results (through `toolbox-fear-and-mental-stress.md` FR-029 and FR-100 cases); the option file has no package-level `var` (`rule-options.md` FR-023 scan). |
| FR-001 to FR-104, end to end | scenario | One test in `scenarios/` through the public API of the module's packages only: an RE-C effect fails, the host applies damage through `ApplyDamage` and Poisoned through `ApplyConditionToSet`, six successes, a seventh ends it; `unhealableWhileEffectLasts` present. Added in Phase 5 when the combat use cases ship. |

- **Doubles and contract suites:** none new.
- **Mutation testing** (`doc/rules/testing.md` §8): `gomu` on package `toolbox`, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (command in testing.md), threshold 80. Mutation-sensitive lines: the planner, the margin comparison (`>=` against `>`), the successes threshold, the cumulative count, the reduction against the current level (FR-108), the stage branches of `DiceRequired` (FR-087), the printed-average rejection (FR-089) and the rejection-reason pairing (FR-097). The `*_usecase.go` files SHOULD be mutation tested too.
- Every live FR id appears above. FR-005 is not mechanically covered. FR-003, FR-004, FR-007 are withdrawn (section 4.10) and have no test here.

## 9. Open questions and decisions

Settled (one line each): Q-1 context shape, Q-2 catalogue placement, Q-3 option riders as explicit fields, Q-4 no overlap with `CunningStrikeConditionOnFailedSave` or `SavingThrowEffect`, Q-13 no overlap with `environment-hazards-and-objects.md`, Q-14 the `character` upstream edge: all by shared-types-and-ownership.md FR-045, FR-048, FR-057. Q-12: the printed "57 (13d8)" is carried as printed (FR-018). Q-5: results carry `ConditionKind` only; the host builds the source-bearing `Condition` (toolbox-fear-and-mental-stress.md FR-010). Q-6: a forced failure returns base conditions only (FR-038). Q-8: repeat successes are cumulative, as the SRD never says "consecutive" (FR-057). Q-9: the printed number is an explicit per-call input, an interpretation of p. 257, a stat-block rule applied to Toolbox entries (FR-041, FR-089). Q-18: an exemption returns `AutoSucceeded` with no consequences (FR-098); the definition forbids half damage with an automatic-success set or repeat saves (FR-094). Q-19: the engine cannot check the save's ability because results carry none (d20-tests.md FR-041); the host uses `SaveGatedRollPlan.ability` (FR-039).

Settled, later pass (none escalated; the SRD text, the sibling specs and, for Q-16 (b), the SRD's "Poisoned Darts" wording answered each):

- **Q-2b, Q-10, Q-17:** `content` decodes a data-only entry record and `toolbox` converts it (poison `PoisonEntry` FR-082, FR-083; traps `TrapEntry` FR-092, FR-093; shared-types-and-ownership.md FR-045, FR-047; `content-schema-and-decoder.md` FR-129), so no type moves. Effects with no save are `SaveGatedNoSave` (FR-105, FR-107; Fever and Fatigue, p. 194). Q-17 is closed by `environment-hazards-and-objects.md` (HB-1), and this spec reports the lifetime (FR-067).
- **Q-7 (closed):** this spec carries only the `healing` flag (FR-045, unchanged). The healing ceiling is owned by `toolbox-poison.md` `PoisonEffects.healingCeiling` (its FR-047), computed from tracked unhealable damage (an interpretation of "can't be healed ... while the creature remains Poisoned", p. 198); no `hit-points-and-damage.md` amendment is requested.
- **Q-16 (a), (d):** an Exhaustion-reducing repeat success ending at level 0 (p. 194) is in the model (FR-106, FR-108), and a repeat-only effect is built directly (FR-109). **(b), reopened:** a rolled number of damage instances (1d3 darts, then 1d6 per dart, p. 200) is in the model as `SaveGatedRolledInstances` (FR-110, FR-111), so `toolbox-traps.md` no longer needs its per-dart expansion (its A-2). **(c), (e):** a second repeat trigger that does not end the effect (Demonic Possession, p. 193), non-condition riders and time-bound suppression (Sight Rot, p. 194) stay with the owning sibling.
- **Q-11:** Advantage (p. 194) is a roll-mode source the host combines with the plan (section 6), so `SaveGatedCircumstance` stays Disadvantage only; the early end of one condition ("wakes up", pp. 197-198), non-condition riders and DC scaling (Extreme Heat, p. 195) are the sibling's, built per moment (FR-063, section 5.1); adding them would add speculative API.
- **Q-15, Q-20:** each sibling owns its operation, `result` and attribute values in its own section 7 and adds its constants to the `toolbox` signal constants; this spec owns the shared keys and instruments and lists every sibling's values (section 7, amendments accepted). HA-1, HA-2, HA-4 are accepted (FR-105 to FR-109). The three option types are defined here (FR-102 to FR-104).
- **Q-21 (amendments, 2026-10):** applied from the environmental, fear, poison and traps specs: the sibling operation and `result` tables and the shared keys (section 7), the forced-failure repeat row (FR-062), rolled instances (FR-110, FR-111), the once-per-call shared amount (FR-080) and the named services (section 5). Rejected: none; the traps request is met by FR-110 and FR-111 rather than the earlier "multiplier" wording.

## 10. Changelog impact

Under `Unreleased`, **Added**: the `toolbox` context with `SaveGatedEffect` and its parts, `ActiveSaveGatedEffect`, the use cases `ResolveSaveGatedEffect`, `ResolveRepeatSave`, `EndSaveGatedEffect` and `RollSaveGatedDamage` with their results, the services `SaveGatedPlanner`, `SaveGatedEvaluator` and `DiceRequired`, `ToolboxEvent`, `SaveGatedEvent`, the `toolbox` signal constants (exported from package `toolbox`), the signals in section 7, and `ToolboxRuleOptions` (consumed by `toolbox-fear-and-mental-stress.md` FR-100). **Changed** (documentation and guardrails, not API): the context map in `doc/rules/architecture.md` §2, the architecture tests, `AGENTS.md`. Adding a field or a variant to a closed interface later is breaking (architecture.md §7). Not **BREAKING**: a new context with no change to an existing exported member or signal.
