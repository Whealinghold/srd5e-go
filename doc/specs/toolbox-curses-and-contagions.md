# Curses and Magical Contagions

| | |
|---|---|
| Status | draft |
| Context | `toolbox` |
| Roadmap phase | 5 |
| Depends on | `toolbox-hazard-effects.md`, `magical-effects.md`, `content-schema-and-decoder.md`, `content-source-port.md`, `d20-tests.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `conditions.md`, `hit-points-and-damage.md`, `combat-foundations.md`, `rests-and-hit-dice.md`, `ability-scores.md`, `errors-and-results.md`, `observability-ports.md` |

## 1. Purpose and scope

The mechanical rules of "Curses and Magical Contagions" (SRD 5.2.1, pp. 193-194) as pure domain code with a thin application layer: state in, new
state and events out (`doc/rules/architecture.md` §4). The host rolls the saves through `d20.Resolver` (no use case here calls it,
shared-types-and-ownership.md FR-044) and reports them. The engine reads no clock: spans of time (24 hours, 1 year, "days", "1 minute") arrive as
reported events or leave as values the host tracks.

**No second save-then-effect model.** `toolbox-hazard-effects.md` owns `SaveGatedEffect` and its planner, `DiceRequired` and evaluator; 4.0 maps each
contagion effect onto it and 4.0a records the accepted hazard amendments (HA-1, HA-2, HA-4; HA-3 is section 7). Results carry the hazard `SaveGatedConsequences` and events (no
`ContagionEffect`). Effects imposed on a creature (Exhaustion level, Blinded, Incapacitated, Psychic damage) are handed back as requests: damage to
`ApplyDamage`, conditions and Exhaustion to `ApplyConditionToSet`, `RemoveConditionFromSet`, `ReduceExhaustionOnSet` (`conditions.md`). A use case
here calls only hazard **domain** functions (planner, `DiceRequired`, evaluator operations `EvaluateInitial`, `EvaluateRepeat`, `EvaluateEnd`), never a use case. The three contagions are also
named catalogue entries (shared-types-and-ownership.md FR-045): `ContagionDefinition` holds identity and printed data only (FR-117); every rule is
`toolbox` domain behaviour.

Owned here: the curse potency benchmark, Demonic Possession, Rest and Recuperation, contagion exposure, Sewer Plague's rest filters, Sight Rot's
ointment suppression, and the infection as an `OngoingEffect` of category `magicalContagion` (FR-109), so Dead (p. 180) lapses and revives it.
Possession prevention and its other endings (p. 186) reuse `magical-effects.md` (FR-037, FR-039). The hazard model has one repeat trigger and no
non-condition rider (its Q-16 c, e) and no Advantage circumstance (its Q-11), so possession and recuperation are modelled here.

**Supersedes (shared-types-and-ownership.md FR-048):** this spec is the topic owner of Demonic Possession, Rest and Recuperation, the three contagions
and the potency benchmark; the possession requirements in `magical-effects.md` 4.6 (moved here: FR-007, FR-009, FR-025, FR-027, FR-108), the Sewer
Plague notes in `rests-and-hit-dice.md` (its `RestEffect` members are only filtered here, FR-082, FR-083) and the gap rows in
`toolbox-hazard-effects.md` 5.1 yield to it. `magical-effects.md` FR-145 stays.

**Out of scope:** Cursed Creatures, Narrative Curses and other Environmental Curses (p. 193; prose and GM advice, `content-monsters.md`); Cursed Magic
Items (p. 193; `magic-item-rules.md` `CurseStatus`, FR-057 to FR-059, FR-071); Creating a Background (pp. 192-193;
`content-species-backgrounds-feats.md` FR-072, FR-073, shared-types-and-ownership.md FR-045; this spec's Q-20); what the spells do (Bestow Curse, Remove Curse,
Greater Restoration, Dispel Evil and Good, Protection from Evil and Good, Heal, Lesser Restoration, Contagion: `content-spells.md`, `spellcasting.md`,
`RemoveCurse` in `magical-effects.md`; this spec receives only the fact that such magic was applied); curses with neither a specified time nor "until
dispelled" (no benchmark, FR-004); other contagions and altered DCs (p. 194 "Feel free to alter" is GM advice, not a `ToolboxRuleOptions` field,
shared-types-and-ownership.md FR-045; a host-authored contagion has no rules here, Q-3; only Rest and Recuperation, FR-031 to FR-036, is generic);
time tracking, the Long Rest, Exhaustion, Blinded and Incapacitated (`rests-and-hit-dice.md`, `conditions.md`; death at level 6 is the host's,
toolbox-hazard-effects.md FR-046); poison (a separate toolbox spec). Nothing core to these two pages is missing from the PDF.

## 2. Source

PDF page numbers equal printed page numbers (the page printed "193" is PDF page 193).

- `SRD 5.2.1, "Curses and Magical Contagions", p. 193`; `"Curses", "Bestow Curse", p. 193`; `"Curses", "Cursed Creatures",
  "Narrative Curses", "Environmental Curses", "Cursed Magic Items", p. 193`; `"Curses [glossary]", p. 179`; `"Possession
  [glossary]", p. 186`; `"Dead", p. 180` (a revived creature returns with contagions and curses still ongoing)
- `SRD 5.2.1, "Magical Contagions", "Rest and Recuperation", p. 194`; `"Example Contagions", "Cackle Fever" / "Sewer
  Plague" / "Sight Rot", p. 194`
- `SRD 5.2.1, "Attack Rolls", "Rolling 20 or 1", p. 7`; `"Advantage/Disadvantage", pp. 7-8`; `"Disadvantage [glossary]",
  p. 181`; `"Saving Throws", p. 7`; `"Exhaustion [Condition]", p. 181`; `"Emanation [Area of Effect]", p. 181`; `"Immunity",
  p. 183`; `"Long Rest", p. 185`; `"Short Rest", "Spend Hit Point Dice", p. 187`; `"Round Down", p. 187`
- Package rules: `doc/rules/architecture.md` (§2 to §4, §6), `doc/rules/observability.md` (§2 to §7),
  `doc/rules/testing.md` (§3 to §5), `shared-types-and-ownership.md`.

## 3. Vocabulary

Every exported name is final and carries a `Contagion`, `Possession`, `Curse` or `DemonicPossession` prefix or a use-case prefix, except the three
`DemonicPossessionState` variants (names `magical-effects.md` 4.6 already cites). Hazard names are used by reference. Package names with no SRD source
add no rules.

| SRD term | Type | Notes |
|---|---|---|
| curse, "specified time", "until dispelled" / "equates to a level 3 spell" | `CurseDuration` (closed interface: `CurseForSpecifiedTime(Duration)`, `CurseUntilDispelled`); `CursePotency.benchmark` returning closed interface `CursePotencyBenchmark` (`CursePotencyBenchmarkFound(spellLevel)`, `CursePotencyBenchmarkUnavailable`) | p. 193. `spellLevel` is a plain `int` (3 or 9, `ErrInvalidInput` error outside 1 to 9): this spec imports no `SpellLevel`, the host maps it to one. |
| Demonic Possession / "a DC 15 Charisma saving throw" | `DemonicPossessionState` (closed interface: `NotPossessed`, `PossessedCreatureInControl`, `DemonicEntityInControl`); `DemonicPossession.saveDc` (`DifficultyClass`), `saveAbility` (`Ability`) | p. 193. `NotPossessed` is exported only here. Constants. |
| "rolls a 1 on a D20 Test" / "end of each later turn" | `PossessionD20TestRolled(roll: Roll, kind: D20TestKind)`; `PossessionTurn` (closed interface: `PossessionEndOfSeizureTurn`, `PossessionEndOfLaterTurn`) | `Roll.Natural()` is the d20 used (`d20-tests.md` FR-011). Q-5. |
| prevention and ending | `PossessionPrevention`; `PossessionEndingMagic` (closed interface: `PossessionEndingMagicDispel(PossessionEnding)`, `PossessionEndingMagicCurseRemoval(CurseRemovalMeans)`); `PossessionEndCause` (enum: `longRestSave`, `dispelEvilAndGood`, `curseRemoved`) | `PossessionEnding`, `CurseRemovalMeans` reused (`magical-effects.md` FR-039, FR-028, FR-029). `longRestSave` is `PossessionEnding.possessingEffect`. |
| why a step changed nothing or was refused | `PossessionUnchangedReason` (enum: `notPossessed`, `noControlLossRoll`, `entityAlreadyInControl`, `noSaveDue`, `seizureTurnNotLater`, `saveFailed`, `notALongRest`), `PossessionRejectionReason` (enum: `alreadyPossessed`, `saveDoesNotMatchRequirement`) | `reason` fields (shared-types-and-ownership.md FR-018). |
| Long Rest / magical contagion | `RestKind`; `ContagionExample` (enum: `cackleFever`, `sewerPlague`, `sightRot`) | `rests-and-hit-dice.md`; only `longRest` counts (FR-023). SRD heading "Example Contagions". |
| Rest and Recuperation | `ContagionRecuperationReport` (closed interface: `ContagionRecuperationRanItsCourse`, `ContagionRecuperationStopped(interruption, daysBefore)`), `ContagionAdvantageWindow` (closed interface: `ContagionAdvantageWindowHeld` (24 hours), `ContagionAdvantageWindowNotHeld`) | Interruptions are `RestInterruption` (`rests-and-hit-dice.md` FR-005). |
| "Advantage on saving throws to fight off" | `ContagionSavePurpose` (enum: `exposure`, `recuperation`, `fightingTheContagion`, `uncontrollableLaughter`, `laughterRepeat`), `ContagionSaveCircumstances` | FR-036. An input to the T2 lookup; no plan or result carries it. |
| the save a host must roll / infected creature / "1d4 days after infection" | `SaveGatedRollPlan`; `ContagionInfection` (closed interface: `ContagionCackleFeverInfection`, `ContagionSewerPlagueInfection`, `ContagionSightRotInfection`); hazard `SaveGatedOnsetAfter(SaveGatedRolledTime(1d4, days))` | Hazard; carried by every `...SaveRequired` (FR-112). Host holds one per creature and contagion; holds hazard `ActiveSaveGatedEffect` states (FR-111) and the `OngoingEffectID` (FR-109). Incubating is `SaveGatedPhaseAwaitingOnset`; `ContagionInfection.isIncubating` reads it. |
| how infected | `ContagionExposureRoute` (closed interface: `ContagionWoundedByCarrier(carrier)`, `ContagionContactWithFilth`, `ContagionDrankTaintedWater`, `ContagionStartedTurnInEmanation(carrier, distanceFeet)`, `ContagionSkinContact(carrier)`), `ContagionExposedCreature(creatureType, species)`, `ContagionHeldState(protections, immunities, current)` | `carrier` is a `CombatantID`; `species` is an optional `SpeciesID` (the zero value means none, `IsZero`); `current` is an optional `ContagionInfection` (a value plus an `ok` bool). |
| exposure planning | `ContagionExposureCheck` (closed interface: `ContagionExposureSaveRequired(plan)`, `ContagionExposureRefusal(reason)`), `ContagionRefusalReason` (enum, FR-049), `ContagionExposureDecision` (closed interface: `...Refused(reason)`, `...ResistedProtected(protection)`, `...Resisted`, `...Infected`) | FR-115. |
| 24-hour protection, 1-year immunity | `ContagionProtection(contagion, carrier)` (FR-046), `ContagionImmunityGrant(contagion, duration)`, `ContagionImmunityDuration` (enum: `oneYear`) | Host-held. `DurationUnit` has no years. |
| Uncontrollable Laughter | hazard CE-LAUGHTER (= hazard RE-I) as `ContagionCackleFeverInfection.laughter`; `ContagionLaughterCheck` (closed interface: `ContagionLaughterSaveRequired(plan)`, `ContagionLaughterNotDue(reason)`), `ContagionLaughterRefusal` (enum: `notActive`, `noExhaustion`, `psychicDamage`) | `ContagionLaughterRules.check` (FR-070). |
| ointment, "suppresses" / Herbalism Kit | `ContagionOintmentDose`, `ContagionSightRotSuppression(suppressed, completedPeriods)`, `ContagionSuppressionState` (`notSuppressed`, `suppressed`); `ContagionHerbalismProficiency` (`proficient`, `notProficient`), `ContagionHerbalismKit` (`held`, `notHeld`) | FR-086, FR-114. |
| "Heal or Lesser Restoration" | `ContagionCuringMagic` (closed interface: `ContagionHealCast`, `ContagionLesserRestorationCast`, `ContagionGmRuledCuringMagic`) | Q-13. |
| steps the host reports | `ContagionStep` (closed interface: `ContagionOnsetReachedStep`, `ContagionLongRestFinishedStep`, `ContagionDawnReachedStep`, `ContagionDamageTakenStep`, `ContagionLaughterTurnEndedStep`, `ContagionLaughterMinuteElapsedStep`, `ContagionOintmentAppliedStep`, `ContagionSuppressionElapsedStep`, `ContagionCuringMagicStep`) | Fields in section 6. |
| step outcomes | `ContagionUnchangedReason` (enum: `notIncubating`, `saveFailed`, `notSuppressed`, `notLaughing`, `noExhaustionToReduce`, `notALongRest`), `ContagionRejectionReason` (enum: `alreadySuppressed`, `notProficient`, `noKit`, `saveDoesNotMatchRequirement`), `ContagionEndCause` (enum: `threeLongRestSaves`, `exhaustionReducedToZero`, `ointmentTotal`, `curingMagic`) | `reason` fields. |
| the three contagions as catalogue entries | `ContagionDefinition(id, name, infectionDc, fightingDc, source)` (a `content` value object), `ContagionID` | FR-117, FR-122. `ContagionID`: `content-schema-and-decoder.md` (FR-114, FR-129); the entry shape is owned here and registered by row R3-contagions. `ContagionExample.tryFromID` (FR-118). |
| this spec's events | `CurseAndContagionEvent` (closed interface; implements the `ToolboxEvent` interface, `toolbox-hazard-effects.md` FR-086) | The nine events of FR-095. |

**Names not used here** (exported names never collide, architecture.md §9): `DamageTaken` (`combat-action-resolution.md` FR-090), `LongRestFinished`
(`class-bard.md`), `DaysPassed`, `DawnReached`, `HealCast`, `LesserRestorationCast`, `PossessionState`, `Possessor`, `PossessionBegan`,
`PossessionEnded` (moved out of `magical-effects.md`), `ContagionEffect`, `ExampleContagion`. `Ability`, `Damage`, `DamageType`, `ExhaustionLevel`,
`ConditionKind`, `ContagionID`, `CreatureType`, `SpeciesID`, `RestKind`, `RestInterruption`, `RestEffect`, `Roll`, `D20TestKind`,
`CheckResult`, `DifficultyClass`, `RollCircumstance`, `CombatantID`, `OngoingEffect` and the `SaveGated...` types come from their owners
(shared-types-and-ownership.md FR-021, FR-046).

## 4. Requirements

Stable ids. Distances are whole feet; pages are printed
SRD 5.2.1 pages. A result variant is written without its use-case prefix (FR-099).

### 4.0 The three contagions in the hazard model

Each is a hazard `SaveGatedEffect` (its notation "RE-x") built from p. 194; these are the only effects the contagions start. Constants: T2.

| Ref | Hazard shape | Source |
|---|---|---|
| CE-FEVER | Cackle Fever, Fever. `SaveGatedNoSave` (hazard FR-105, FR-107); onset `SaveGatedOnsetAfter(SaveGatedRolledTime(1d4, days))`; failure `SaveGatedExhaustionGain(1, untilEffectEnds)`; duration `SaveGatedDurationUntilEnded`; no repeat | "Cackle Fever", "Fever", p. 194 |
| CE-LAUGHTER | Cackle Fever, Uncontrollable Laughter. The hazard's RE-I: Constitution DC 13; failure 1d10 Psychic (printed 5) and Incapacitated; 1 minute; repeat at the end of each of its turns, ends after 1 success | "Uncontrollable Laughter", p. 194 |
| CE-FIGHT-CACKLE | Cackle Fever, Fighting the Contagion. The hazard's RE-K: Constitution DC 13; repeat `SaveGatedRepeatAfterLongRest`, ends after 3 successes; failure `SaveGatedRepeatFailureNoFurtherEffect`. Built directly in `SaveGatedPhaseActive(0)` with a placeholder initial failure (hazard FR-109, RE-K); that failure is never resolved | "Fighting the Contagion", p. 194 |
| CE-FATIGUE | Sewer Plague, Fatigue. `SaveGatedNoSave` (hazard FR-105, FR-107); onset 1d4 days; failure `SaveGatedExhaustionGain(1, untilRemoved)`; until ended; no repeat | "Sewer Plague", "Fatigue", p. 194 |
| CE-DAWN | Sewer Plague, Fighting the Contagion. Constitution DC 11; placeholder initial failure 1 Exhaustion level, until removed (as CE-FIGHT-CACKLE; never resolved); `SaveGatedRepeatSaves(trigger: SaveGatedRepeatAfterInterval(24, hours), onSuccess: SaveGatedRepeatSuccessReducesExhaustion(1), endRule: SaveGatedEndsWhenExhaustionReachesZero, onFailure: SaveGatedRepeatFailureReapplies(SaveGatedExhaustionGain(1, untilRemoved)))` (interval is the nearest trigger to "daily at dawn"; the host reports dawn, toolbox-hazard-effects.md FR-063; no success counter). The reduction is handed back as `SaveGatedExhaustionReduction(1)` in `SaveGatedConsequences` only. Built directly in `SaveGatedPhaseActive(0)` (hazard FR-109, RE-P); the creature's level is read from `SaveGatedSubject.exhaustion` | "Fighting the Contagion", p. 194 |
| CE-BLINDED | Sight Rot. `SaveGatedNoSave` (hazard FR-105, FR-107), immediate onset; failure condition Blinded (`SaveGatedAnyFailure`); until ended; no repeat. Its infection save is the exposure save (FR-050) | "Sight Rot", p. 194 |

### 4.0a Amendments to other specs

This spec edits no other file. `toolbox-hazard-effects.md` accepted HA-1, HA-2 and HA-4 (its FR-105 to FR-109, section 5); no row is provisional. HA-3 is replaced by section 7 (hazard Q-15: each sibling owns its signal values).

| Ref | Spec | Outcome |
|---|---|---|
| HA-1 | `toolbox-hazard-effects.md` | **Accepted.** `SaveGatedNoSave` (FR-105, FR-107): `Failed` with no `ability` and `saveResult` `notRequired`, never `SaveRequired`, `AutoSucceeded` or `Saved`; it carries no repeat, Disadvantage, automatic-success set, half damage or margin condition (so a repeat is a second effect). A repeat-only effect is built directly in `SaveGatedPhaseActive(0)` (FR-109). |
| HA-2 | `toolbox-hazard-effects.md` | **Accepted** as `SaveGatedRepeatSaves(trigger, onSuccess, endRule, onFailure)` with `SaveGatedRepeatSuccessCounts` or `SaveGatedRepeatSuccessReducesExhaustion(levels)` and `SaveGatedEndsAfterSuccesses(endsAfterSuccesses)` or `SaveGatedEndsWhenExhaustionReachesZero` (FR-106); `SaveGatedSubject.exhaustion` (`ExhaustionLevel?`); `SaveGatedExhaustionReduction` only in `SaveGatedConsequences`, including on `ResolveRepeatSaveEnds` (FR-108). |
| HA-3 | n/a | **Replaced** by section 7 (this spec lists its own operation, result, step, kind and end_cause values). |
| HA-4 | `toolbox-hazard-effects.md` | **Accepted.** Evaluator operations `EvaluateInitial`, `EvaluateRepeat`, `EvaluateEnd`, each taking already-rolled dice (section 5). |
| HB-1 | `environment-hazards-and-objects.md` (Q-18) | `HazardRecord` carries `contagionLevels`, counted in the 6-level sum and subtracted in `removableLevels`, so a Long Rest cannot remove Cackle Fever's Fever level. **Accepted and complete** (its FR-002, FR-060, FR-115). |

### 4.1 Curse potency benchmark

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `CurseDuration.specified(duration)` MUST reject a zero or negative `Duration` with `ErrInvalidInput` error. | SRD 5.2.1, "Curses and Magical Contagions", p. 193 ("lasts for a specified time") | unhappy (invalid input) |
| FR-002 | `CursePotency.benchmark` of a curse lasting exactly 1 minute MUST be `CursePotencyBenchmarkFound` with `spellLevel` 3. | SRD 5.2.1, "Bestow Curse", p. 193 | happy |
| FR-003 | `CursePotency.benchmark` of `CurseUntilDispelled` MUST be `CursePotencyBenchmarkFound` with `spellLevel` 9. | SRD 5.2.1, "Bestow Curse", p. 193 | happy |
| FR-004 | `CursePotency.benchmark` of any other specified duration (T10) MUST be `CursePotencyBenchmarkUnavailable`: the SRD gives two benchmarks and the engine MUST NOT interpolate. | SRD 5.2.1, "Bestow Curse", p. 193 | unhappy (boundary) |
| FR-005 | `CursePotency.benchmark` MUST be a pure function of the `CurseDuration`: equal durations give equal results, and it alters nothing. | `doc/rules/architecture.md` §3 (purity) | unhappy (determinism) |

### 4.2 Demonic Possession

Operations take the state and return the new state and events. "Unchanged" means state unchanged, no event, with the stated reason. Every save is a
hazard `SaveGatedSave` validated by FR-112; "fails" includes `SaveGatedSaveForcedFailure`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-006 | `DemonicPossession.saveDc` MUST be 15 and `saveAbility` Charisma, for the first save, the end-of-turn save and the Long Rest save. | SRD 5.2.1, "Environmental Curses", p. 193 | happy |
| FR-007 | A `NotPossessed` creature, not prevented (FR-108), that fails the save MUST become `PossessedCreatureInControl` and the operation MUST emit one `DemonicPossessionBegan`. | SRD 5.2.1, "Environmental Curses", p. 193 | happy |
| FR-008 | A `NotPossessed` creature, not prevented, that succeeds MUST stay `NotPossessed` with no event. | SRD 5.2.1, "Environmental Curses", p. 193 | happy |
| FR-009 | Targeting a creature already possessed (either control state) MUST be rejected (`alreadyPossessed`), state unchanged, no event, whatever save is supplied. | SRD 5.2.1, "Environmental Curses", p. 193 (silent on a second possession) | unhappy (state error) |
| FR-010 | A natural 1 on a D20 Test by a `PossessedCreatureInControl` MUST give `DemonicEntityInControl` and emit one `DemonicEntityTookControl`. | SRD 5.2.1, "Environmental Curses", p. 193 | happy |
| FR-011 | A natural 2 to 20 by a `PossessedCreatureInControl` MUST be Unchanged (`noControlLossRoll`). | SRD 5.2.1, "Environmental Curses", p. 193 | unhappy (boundary) |
| FR-012 | A natural 1 by a creature in `DemonicEntityInControl` MUST be Unchanged (`entityAlreadyInControl`). | SRD 5.2.1, "Environmental Curses", p. 193 | unhappy (state error) |
| FR-013 | A natural 1 by a `NotPossessed` creature MUST be Unchanged (`notPossessed`). | SRD 5.2.1, "Environmental Curses", p. 193 | unhappy (state error) |
| FR-014 | The number tested MUST be `Roll.Natural()`, the d20 used. With Advantage, a 1 and a 17 (17 used) MUST NOT seize control; with Disadvantage, a 1 and a 17 (1 used) MUST. | SRD 5.2.1, "Environmental Curses", p. 193 ("rolls a 1 on a D20 Test"); "Disadvantage [glossary]", p. 181 and "Advantage/Disadvantage", pp. 7-8 (which die counts); "Attack Rolls", "Rolling 20 or 1", p. 7 (written for attack rolls). Reading the natural 1 as the d20 used for every D20 Test follows `d20-tests.md` FR-011; p. 193 alone extends it to checks and saves. | unhappy (rule interaction) |
| FR-015 | A natural 1 MUST seize control on every kind of D20 Test (ability check, saving throw, attack roll); the `D20TestKind` MUST NOT change the outcome. | SRD 5.2.1, "Environmental Curses", p. 193 ("a D20 Test") | happy |
| FR-016 | At the end of a later turn (`PossessionEndOfLaterTurn`) of a creature in `DemonicEntityInControl`, a successful save MUST give `PossessedCreatureInControl` (it regains control and stays possessed) and emit one `PossessedCreatureRegainedControl`. | SRD 5.2.1, "Environmental Curses", p. 193 | happy |
| FR-017 | The same save failed MUST be Unchanged (`saveFailed`). | SRD 5.2.1, "Environmental Curses", p. 193 | unhappy (boundary) |
| FR-018 | At a turn end of a `PossessedCreatureInControl` no save is due: the result MUST be Unchanged (`noSaveDue`), whatever save is supplied (the save is how the creature "regain[s] control"). | SRD 5.2.1, "Environmental Curses", p. 193 | unhappy (state error) |
| FR-019 | At `PossessionEndOfSeizureTurn` no save is due: the result MUST be Unchanged (`seizureTurnNotLater`), whatever save is supplied. | SRD 5.2.1, "Environmental Curses" ("later turns"), p. 193 | unhappy (boundary) |
| FR-020 | A turn end of a `NotPossessed` creature MUST be Unchanged (`notPossessed`). | SRD 5.2.1, "Environmental Curses", p. 193 | unhappy (state error) |
| FR-021 | A finished Long Rest (`RestKind.longRest`) with a successful save MUST give `NotPossessed` from either possessed state and emit one `DemonicPossessionEnded` with cause `longRestSave`. | SRD 5.2.1, "Environmental Curses", p. 193 | happy |
| FR-022 | A finished Long Rest with a failed save MUST be Unchanged (`saveFailed`), including `DemonicEntityInControl`. | SRD 5.2.1, "Environmental Curses", p. 193 | unhappy (boundary) |
| FR-023 | A rest reported as `RestKind.shortRest` (including a Long Rest stopped by an interruption that granted Short Rest benefits) MUST be Unchanged with reason `notALongRest`. It is a rule outcome, not a rejection: a host reports every rest it finishes. | SRD 5.2.1, "Environmental Curses" ("After finishing a Long Rest"), p. 193; "Long Rest", p. 185; `doc/rules/observability.md` §7 | unhappy (boundary) |
| FR-024 | A finished Long Rest for a `NotPossessed` creature MUST be Unchanged (`notPossessed`). | SRD 5.2.1, "Environmental Curses", p. 193 | unhappy (state error) |
| FR-025 | `PossessionEndingMagicDispel(PossessionEnding.dispelEvilAndGood)` MUST give `NotPossessed` from either possessed state and emit one `DemonicPossessionEnded` with cause `dispelEvilAndGood`; no save. | SRD 5.2.1, "Environmental Curses", p. 193; "Possession", p. 186 | happy |
| FR-026 | `PossessionEndingMagicCurseRemoval(means)` for any `CurseRemovalMeans` (`RemoveCurseSpell`, `GreaterRestorationSpell`, `MagicThatEndsCurses`) MUST give `NotPossessed` from either possessed state and emit one `DemonicPossessionEnded` with cause `curseRemoved`. | SRD 5.2.1, "Environmental Curses", p. 193; "Curses", p. 179 | happy |
| FR-027 | Ending magic applied to a `NotPossessed` creature MUST be Unchanged (`notPossessed`). | SRD 5.2.1, "Environmental Curses", p. 193 | unhappy (state error) |
| FR-028 | After control is regained (FR-016) a later natural 1 MUST seize control again: regaining control MUST NOT end the possession. Only FR-021, FR-025 and FR-026 end it. | SRD 5.2.1, "Environmental Curses", p. 193 | unhappy (rule interaction) |
| FR-029 | No possession operation MUST alter the state passed in; the events list MUST be defensively copied. | `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-030 | Possession operations MUST roll no dice and read no clock. `ResolveDemonicPossession` is built without a `Roller`; an architecture check MUST find no `Roller` import in its file and no `time.Now` in the possession domain files. | `doc/rules/architecture.md` §3 (purity); SRD 5.2.1, "Environmental Curses", p. 193 (all four saves are D20 Tests resolved by the host) | unhappy (determinism) |
| FR-108 | A `PossessionTargeted` command with a non-empty `[]PossessionPrevention` (treated as a set) (the host's fact that Protection from Evil and Good protects the target) for a `NotPossessed` creature MUST return `ResolveDemonicPossessionPrevented` carrying the prevention, state unchanged, no event, no save needed. An already possessed creature is rejected first (FR-009). | SRD 5.2.1, "Possession", p. 186 ("Possession can be prevented by the Protection from Evil and Good spell"); `magical-effects.md` FR-037 | happy |
| FR-113 | `PossessionEndingMagicDispel` MUST return an error wrapping `srderr.ErrInvalidInput` for `PossessionEnding.possessingEffect`: that value names an end the possessing effect defines (FR-021), not a spell the host applies. | `magical-effects.md` FR-039; `doc/rules/architecture.md` §3 | unhappy (invalid input) |

### 4.3 Rest and Recuperation (applies to any magical contagion, p. 194)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | A `ContagionRecuperationRanItsCourse` report (3 days with no activity that would interrupt a Long Rest) MUST make one Constitution DC 15 save due (`SaveRequired` with the plan until the save is supplied, FR-112). | SRD 5.2.1, "Rest and Recuperation", p. 194 | happy |
| FR-032 | A successful recuperation save MUST grant a `ContagionAdvantageWindowHeld` of 24 hours: Advantage on saves to fight off the contagion for the next 24 hours. | SRD 5.2.1, "Rest and Recuperation", p. 194 | happy |
| FR-033 | A failed recuperation save MUST give no grant and no other effect (the SRD states none). | SRD 5.2.1, "Rest and Recuperation", p. 194 | unhappy (boundary) |
| FR-034 | A `ContagionRecuperationStopped` report MUST make no save due and return `Interrupted` carrying the `RestInterruption`, for each of the four interruptions (`rollingInitiative`, `castingSpellOtherThanCantrip`, `takingDamage`, `physicalExertion`), whatever save is supplied. Restarting is Q-15. | SRD 5.2.1, "Rest and Recuperation", p. 194; "Long Rest", "Interrupting the Rest", p. 185 | unhappy (state error) |
| FR-035 | `ContagionRecuperationStopped.daysBefore` MUST accept 0, 1 and 2 and returns an error wrapping `srderr.ErrInvalidInput` for a negative value or 3 or more (a full period is `...RanItsCourse`). | SRD 5.2.1, "Rest and Recuperation" ("3 days"), p. 194 | unhappy (boundary) |
| FR-036 | `ContagionSaveCircumstances.of(purpose, window)` MUST return one Advantage `RollCircumstance` for purpose `fightingTheContagion` while the window is held, and none for any other purpose or when not held. It returns contributions only (combining with Disadvantage is `modifiers-and-advantage.md`). A `SaveRequired` plan for a Long Rest or dawn step (FR-064, FR-078) MUST carry the hazard plan's circumstances plus these. | SRD 5.2.1, "Rest and Recuperation", p. 194 | unhappy (rule interaction) |

### 4.4 Contagion model

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-037 | `ContagionExample` MUST have exactly three values: `cackleFever`, `sewerPlague`, `sightRot`. | SRD 5.2.1, "Example Contagions", p. 194 | happy |
| FR-038 | Every contagion save MUST use the ability and DC of T2 (4.12) for its contagion and purpose (all Constitution); the effects of 4.0 MUST be built with the same constants. | SRD 5.2.1, "Cackle Fever", "Sewer Plague", "Sight Rot", p. 194 | happy |
| FR-039 | Constructing an infection MUST return an error wrapping `srderr.ErrInvalidInput` for: a Cackle Fever or Sewer Plague infection whose Fever or Fatigue effect is `SaveGatedPhaseEnded`; a `ContagionCackleFeverInfection` with a `laughter` effect while its Fever is `SaveGatedPhaseAwaitingOnset`; component effects with different targets; a `ContagionSightRotSuppression` with `completedPeriods` outside 0 to 2 (the third ends the contagion). Every infection holds its `OngoingEffectID` (FR-109) as a required field. The fight effect's success count is bounded by toolbox-hazard-effects.md FR-022 and not re-checked (CE-DAWN has none). | SRD 5.2.1, "Cackle Fever", "Sight Rot", p. 194; `doc/rules/architecture.md` §3; toolbox-hazard-effects.md FR-022 | unhappy (invalid input) |
| FR-040 | Two infections of the same variant with equal fields MUST be equal; infections differing in any field MUST NOT be equal. | `doc/rules/architecture.md` §3 (value object); toolbox-hazard-effects.md FR-088 | happy |
| FR-109 | An infection MUST be accompanied by one `OngoingEffect` of category `magicalContagion`. The exposure command carries the host's `OngoingEffectID` and `EffectSource` (magical, `magical-effects.md` FR-008); on infection the engine builds the effect from them, `ResolveContagionExposureInfected` carries it in `effect`, and the `ContagionInfection` stores the id (FR-039). Every `ResolveContagionProgressEnded` carries that `effectID` for the host to remove. On death the host's `elapsedEffects` decides whether the effect lapses (`returning-to-life.md` FR-010); a revived creature returns with the infection only "if the durations ... are still ongoing". This spec stores no clock; a lapsed infection is simply not passed back in. | SRD 5.2.1, "Dead", p. 180; `magical-effects.md` FR-006, FR-008; `returning-to-life.md` FR-010 | happy |
| FR-111 | Each infection MUST hold exactly the hazard `ActiveSaveGatedEffect` states of 4.0: Cackle Fever the Fever (CE-FEVER) and fight effect (CE-FIGHT-CACKLE), plus the laughter effect (CE-LAUGHTER) while active; Sewer Plague the Fatigue (CE-FATIGUE) and dawn (CE-DAWN) effects; Sight Rot the Blinded effect (CE-BLINDED) and its suppression. Their constants MUST equal 4.0 and T2. | SRD 5.2.1, "Cackle Fever", "Sewer Plague", "Sight Rot", p. 194; toolbox-hazard-effects.md FR-008 | happy |

### 4.5 Exposure

`ContagionExposureRules.check` returns the save required or the refusal (no save needed), so the host rolls only when needed.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-041 | A route the contagion does not have MUST be refused (`routeNotApplicable`). Cackle Fever has only `ContagionStartedTurnInEmanation`; Sewer Plague only `ContagionWoundedByCarrier` and `ContagionContactWithFilth`; Sight Rot only `ContagionDrankTaintedWater` and `ContagionSkinContact` (T3). | SRD 5.2.1, "Cackle Fever", "Sewer Plague", "Sight Rot", p. 194 | unhappy (invalid input) |
| FR-042 | Creature types per route: Cackle Fever Humanoid only; Sewer Plague Humanoid only; Sight Rot by `ContagionDrankTaintedWater` Beast or Humanoid, by `ContagionSkinContact` Humanoid only. Any other `CreatureType` MUST be refused (`creatureTypeNotAffected`). | SRD 5.2.1, "Cackle Fever" ("affects Humanoids only"), "Sewer Plague", "Sight Rot", p. 194 | unhappy (boundary) |
| FR-043 | A gnome (species `gnome`) MUST be refused for Cackle Fever (`speciesImmune`) and MUST NOT be for Sewer Plague or Sight Rot. A creature with no species MUST NOT be refused on species grounds. | SRD 5.2.1, "Cackle Fever" ("gnomes are strangely immune"), p. 194; "Immunity", p. 183 | unhappy (rule interaction) |
| FR-044 | A creature whose held immunities include the contagion (the 1-year immunity of FR-066) MUST be refused (`immuneForOneYear`). The engine tracks no year; the host drops the immunity. | SRD 5.2.1, "Cackle Fever" ("immune to Cackle Fever for 1 year"), p. 194 | unhappy (state error) |
| FR-045 | A creature already infected with the same contagion MUST be refused (`alreadyInfected`), infection unchanged. | SRD 5.2.1, "Magical Contagions", p. 194 (silent on reinfection) | unhappy (state error) |
| FR-046 | A route with a `carrier` for which the host holds a `ContagionProtection(contagion, carrier)` MUST be refused (`protectedFromSource`). | SRD 5.2.1, "Cackle Fever", "Sight Rot" ("can't catch the contagion from that particular infected creature for the next 24 hours"), p. 194 | unhappy (state error) |
| FR-047 | `ContagionStartedTurnInEmanation` MUST be in range at 0 to 10 feet inclusive and refused (`outOfRange`) at 11 feet or more. | SRD 5.2.1, "Cackle Fever" ("within a 10-foot Emanation"), p. 194; "Emanation", p. 181 | unhappy (boundary) |
| FR-048 | `ContagionStartedTurnInEmanation` with a negative `distanceFeet` MUST return an error wrapping `srderr.ErrInvalidInput` from its constructor. | `doc/rules/testing.md` §4; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-049 | When several refusals apply the first in this order MUST win: `routeNotApplicable`, `creatureTypeNotAffected`, `speciesImmune`, `immuneForOneYear`, `alreadyInfected`, `protectedFromSource`, `outOfRange` (T3); these seven are the `ContagionRefusalReason` values. A package decision (the SRD does not rank them). | design decision (SRD silent); nearest text: SRD 5.2.1, "Immunity", p. 183 | unhappy (rule interaction) |
| FR-050 | With no refusal and a failed save the result MUST be `Infected` carrying the new infection, the `OngoingEffect` of FR-109, the consequences and the events in the order of FR-097: `ContagionContracted` first, then the hazard events of starting the infection's effects (FR-052, FR-053). | SRD 5.2.1, "Sewer Plague", "Sight Rot", p. 194 | happy |
| FR-051 | With no refusal and a successful save the result MUST be `ResistedProtected` for `ContagionStartedTurnInEmanation` (Cackle Fever) and `ContagionSkinContact` (Sight Rot): it carries a 24-hour `ContagionProtection(contagion, carrier)` and emits one `ContagionProtectionGranted`. For every other route it MUST be `Resisted`, with no protection field and no event. | SRD 5.2.1, "Cackle Fever", "Sight Rot", p. 194 | happy |
| FR-052 | Infection by Cackle Fever or Sewer Plague MUST start CE-FEVER or CE-FATIGUE through the hazard evaluator at `SaveGatedInitialExposure`, rolling exactly one `1d4` of days (toolbox-hazard-effects.md FR-048, FR-087) and returning `Deferred` with the effect in `SaveGatedPhaseAwaitingOnset` (faces 1 to 4, T4). The infection also holds its fight or dawn effect in `SaveGatedPhaseActive(0)`. | SRD 5.2.1, "Cackle Fever", "Sewer Plague" ("1d4 days after infection"), p. 194; toolbox-hazard-effects.md FR-048 | happy |
| FR-053 | Infection by Sight Rot MUST roll no dice and start CE-BLINDED at once: hazard result `Failed` with Blinded in its consequences, the effect in `SaveGatedPhaseActive(0)`. Cackle Fever or Sewer Plague MUST return no condition and no Exhaustion until the onset is reached (FR-058, FR-062, FR-063). | SRD 5.2.1, "Sight Rot" ("have the Blinded condition until the contagion ends"), p. 194 | happy |
| FR-054 | A refused or resisted exposure MUST make no `Roller` request. | `doc/rules/testing.md` §4 (determinism); SRD 5.2.1, "Cackle Fever", p. 194 | unhappy (empty and minimal) |
| FR-055 | A dice port failure from the onset roll of FR-052 MUST be returned wrapped once as `*srderr.PortError` with a nil result; an invalid roll result surfaces as the same kind of error through `dice.Roll` (errors-and-results.md FR-036). The held state is untouched. | `doc/specs/dice-and-rolls.md` (failure contract); `doc/rules/architecture.md` §4; toolbox-hazard-effects.md FR-069 | unhappy (collaborator failure) |
| FR-056 | `ContagionExposureRules.check` MUST return the save required (a `SaveGatedRollPlan` with the T2 ability and DC) when no refusal applies, else the first refusal of FR-049, without rolling dice. The purpose `exposure` selects the T2 constants and is not a plan field. | SRD 5.2.1, "Cackle Fever", "Sewer Plague", "Sight Rot", p. 194 | happy |

### 4.6 Progress: common steps

A step not belonging to the infection's contagion returns `NotApplicable` (T5). Counting days is the host's job (Q-7); the resolved onset time is the
hazard `Deferred` state and `ContagionOnsetReachedStep` replaces a day counter.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-058 | `ContagionOnsetReachedStep` on an incubating infection MUST proceed as the hazard onset (toolbox-hazard-effects.md FR-050): `Failed` with the effect in `SaveGatedPhaseActive(0)`, the Exhaustion gain of FR-062 or FR-063 in its consequences, and the hazard `SaveGatedEffectStarted`. The reported onset suffices; the engine checks no day count. | SRD 5.2.1, "Cackle Fever", "Sewer Plague", p. 194; toolbox-hazard-effects.md FR-050 | unhappy (boundary) |
| FR-060 | `ContagionOnsetReachedStep` on an already active infection MUST return `Unchanged` (`notIncubating`), no consequences, no event. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-061 | A step MUST apply only to the contagions in T5; any other pairing MUST return `NotApplicable`, infection unchanged, no consequences, no event. | SRD 5.2.1, "Cackle Fever", "Sewer Plague", "Sight Rot", p. 194 | unhappy (invalid input) |
| FR-062 | When Cackle Fever's effects begin the consequences MUST carry `SaveGatedExhaustionGain(1, untilEffectEnds)` (the Fever). | SRD 5.2.1, "Cackle Fever", "Fever", p. 194 | happy |
| FR-063 | When Sewer Plague's effects begin the consequences MUST carry `SaveGatedExhaustionGain(1, untilRemoved)` (the Fatigue). | SRD 5.2.1, "Sewer Plague", "Fatigue", p. 194 | happy |

### 4.7 Cackle Fever

Fighting the Contagion is the evaluator's repeat-save (`EvaluateRepeat`) on CE-FIGHT-CACKLE (toolbox-hazard-effects.md FR-055 to FR-063); this spec adds only the
contagion end.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-064 | `ContagionLongRestFinishedStep` (`RestKind.longRest`) with a successful DC 13 Constitution save MUST apply the hazard repeat success to CE-FIGHT-CACKLE: `successes` rises by 1 (T7), no consequences. Without a supplied save the result is `SaveRequired` (FR-112, FR-036). | SRD 5.2.1, "Cackle Fever", "Fighting the Contagion", p. 194; toolbox-hazard-effects.md FR-055 | happy |
| FR-065 | A finished Long Rest with a failed save MUST leave the infection unchanged (`saveFailed`; toolbox-hazard-effects.md FR-059). | SRD 5.2.1, "Cackle Fever", "Fighting the Contagion", p. 194 | unhappy (boundary) |
| FR-066 | The third successful Long Rest save MUST end the contagion: `Ended` with cause `threeLongRestSaves`, the FR-109 `effectID`, one `ContagionEnded`, a `ContagionImmunityGrant(cackleFever, oneYear)`, the hazard end of the fight effect (toolbox-hazard-effects.md FR-056, FR-084) and, through the evaluator's end operation with `neutralized` (`EvaluateEnd`; toolbox-hazard-effects.md FR-064), the end of the Fever effect, whose `SaveGatedEffectEnded` lists the Exhaustion level to remove (toolbox-hazard-effects.md FR-067: 1 if the Fever was active, 0 if awaiting onset). | SRD 5.2.1, "Cackle Fever", "Fever", "Fighting the Contagion", p. 194; toolbox-hazard-effects.md FR-064, FR-067 | happy |
| FR-067 | The successes MUST NOT need to be consecutive: a failure MUST NOT reset the count (toolbox-hazard-effects.md FR-057, Q-8; T7). | SRD 5.2.1, "Cackle Fever" ("succeeds on three of these saves"), p. 194 | unhappy (rule interaction) |
| FR-068 | A rest reported as `RestKind.shortRest` MUST return `Unchanged` (`notALongRest`), infection unchanged, as a rule outcome (FR-023). | SRD 5.2.1, "Cackle Fever" ("At the end of each Long Rest"), p. 194; "Long Rest", p. 185 | unhappy (boundary) |
| FR-069 | The Long Rest save MUST apply to an incubating infection too ("an infected creature"). A contagion ended while incubating MUST end the Fever effect from `SaveGatedPhaseAwaitingOnset` (toolbox-hazard-effects.md FR-064) with no Exhaustion level to remove. | SRD 5.2.1, "Cackle Fever", "Fighting the Contagion", p. 194 | unhappy (rule interaction) |
| FR-070 | `ContagionLaughterRules.check(infection, damageType, exhaustion)` MUST return `ContagionLaughterSaveRequired` (a `SaveGatedRollPlan` for CE-LAUGHTER from the hazard planner) only when the Fever is active, the creature has Exhaustion (`exhaustion` present) and the damage type is not Psychic; otherwise `ContagionLaughterNotDue` with `notActive`, `noExhaustion` or `psychicDamage` in that order (T8). This is the damage trigger the hazard model does not express (hazard 5.1, "Trigger"). | SRD 5.2.1, "Cackle Fever", "Uncontrollable Laughter", p. 194 | unhappy (rule interaction) |
| FR-071 | A failed Uncontrollable Laughter save (`ContagionDamageTakenStep`) MUST start CE-LAUGHTER through the hazard evaluator: roll `1d10` through `dice.Roll` as `DiceRequired` names it (toolbox-hazard-effects.md FR-040, FR-087), return consequences with that many points of Psychic damage and Incapacitated, hold the effect in `SaveGatedPhaseActive(0)`, and emit the hazard `SaveGatedEffectStarted`. The host may pass `SaveGatedDamagePrintedAverage` for 5 (toolbox-hazard-effects.md FR-041, Q-10). | SRD 5.2.1, "Cackle Fever", "Uncontrollable Laughter" ("5 (1d10) Psychic damage and has the Incapacitated condition"), p. 194; toolbox-hazard-effects.md FR-040, FR-041 | happy |
| FR-072 | A successful Uncontrollable Laughter save MUST leave the infection unchanged: no consequences, no event, no roll (the hazard `Saved` with empty consequences). | SRD 5.2.1, "Cackle Fever", "Uncontrollable Laughter", p. 194; toolbox-hazard-effects.md FR-033 | unhappy (boundary) |
| FR-073 | At the end of each turn of a laughing creature (`ContagionLaughterTurnEndedStep`) a repeat save that succeeds MUST end CE-LAUGHTER (toolbox-hazard-effects.md FR-056: ends after 1 success), remove the `laughter` effect from the infection and list Incapacitated to lift in the hazard `SaveGatedEffectEnded`. | SRD 5.2.1, "Cackle Fever", "Uncontrollable Laughter" ("ending the effect on itself on a success"), p. 194; toolbox-hazard-effects.md FR-056, FR-067 | happy |
| FR-074 | A laughter repeat save that fails MUST leave the infection unchanged (toolbox-hazard-effects.md FR-059). | SRD 5.2.1, "Cackle Fever", "Uncontrollable Laughter", p. 194 | unhappy (boundary) |
| FR-075 | `ContagionLaughterMinuteElapsedStep` MUST end CE-LAUGHTER through the evaluator's end operation (`EvaluateEnd`) with `durationElapsed` (toolbox-hazard-effects.md FR-064), with no save and no roll, and the same infection change and condition list as FR-073. The host decides the minute has passed (Q-9). | SRD 5.2.1, "Cackle Fever", "Uncontrollable Laughter" ("After 1 minute, it succeeds automatically"), p. 194; toolbox-hazard-effects.md FR-064 | unhappy (boundary) |
| FR-076 | A laughter step for an infection with no `laughter` effect MUST return `Unchanged` (`notLaughing`). | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-077 | A creature already laughing MUST be subject to the trigger again (FR-070): a further failed save rolls the damage, returns it with Incapacitated again and replaces the stored `laughter` effect with the new `SaveGatedPhaseActive(0)` state. The hazard evaluator emits its own `SaveGatedEffectStarted`, which this spec does not suppress; the host MUST hold one `laughter` effect per infection. | SRD 5.2.1, "Cackle Fever" ("each time it takes damage other than Psychic damage"), p. 194 | unhappy (rule interaction) |
| FR-110 | Ending Cackle Fever (FR-066) while a `laughter` effect is active MUST also end it, through the end operation with `neutralized`, so the same result lists Incapacitated to lift; the infection is not passed back in afterwards. | SRD 5.2.1, "Cackle Fever", "Uncontrollable Laughter", p. 194; toolbox-hazard-effects.md FR-064, FR-067 | unhappy (rule interaction) |

### 4.8 Sewer Plague

Fighting the Contagion is the evaluator's repeat-save operation (`EvaluateRepeat`) on CE-DAWN, with the Exhaustion reduction of toolbox-hazard-effects.md FR-108.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-078 | At dawn (`ContagionDawnReachedStep`) a failed DC 11 Constitution save MUST return consequences with `SaveGatedExhaustionGain(1, untilRemoved)` (toolbox-hazard-effects.md FR-058) and leave the infection active. A failure at level 5 requests a gain to level 6; death at 6 is the host's (toolbox-hazard-effects.md FR-046, `conditions.md`; T12). Without a supplied save the result is `SaveRequired` (FR-112, FR-036). | SRD 5.2.1, "Sewer Plague", "Fighting the Contagion", p. 194; "Exhaustion", p. 181 | happy |
| FR-079 | At dawn a successful save by a creature at Exhaustion level 2 or above MUST reduce the level by 1 and the contagion continue; the `Advanced` consequences carry `SaveGatedExhaustionReduction(1)` (toolbox-hazard-effects.md FR-108), which the host passes to `ReduceExhaustionOnSet`. | SRD 5.2.1, "Sewer Plague", "Fighting the Contagion", p. 194 | happy |
| FR-080 | At dawn a successful save at Exhaustion level 1 MUST reduce the level to 0 and end the contagion: `Ended` with cause `exhaustionReducedToZero`, the FR-109 `effectID`, one `ContagionEnded`, consequences carrying `SaveGatedExhaustionReduction(1)`, and the hazard end of the dawn effect and (end operation, `neutralized`) of the Fatigue effect. | SRD 5.2.1, "Sewer Plague" ("If the creature's Exhaustion level is reduced to 0, the contagion ends"), p. 194; "Exhaustion", p. 181 | unhappy (boundary) |
| FR-081 | At dawn a successful save by a creature with no Exhaustion (`exhaustion` absent) MUST return `Unchanged` (`noExhaustionToReduce`), no consequences; the contagion does not end. | SRD 5.2.1, "Sewer Plague", p. 194 | unhappy (boundary) |
| FR-082 | `ContagionSewerPlagueRestRules.hitDieHealing(infection, exhaustion, normal)` MUST return `normal` halved and rounded down when the infection is active and the creature has any Exhaustion, else `normal` (T6; other variants FR-123). The unit is the Hit Points regained for one spent Hit Point Die. `normal` is the amount already floored at 1 by the Short Rest, so a die regaining 1 gives 0 (Q-14). | SRD 5.2.1, "Sewer Plague", "Weakness", p. 194; "Round Down", p. 187; "Short Rest", "Spend Hit Point Dice", p. 187 | happy |
| FR-083 | `ContagionSewerPlagueRestRules.filterLongRestEffects(infection, exhaustion, effects)` MUST remove `AllLostHitPointsRegained` and `ExhaustionLevelReduced` when the infection is active and the creature has any Exhaustion, and keep every other effect in order. | SRD 5.2.1, "Sewer Plague", "Restlessness", p. 194; "Long Rest", p. 185 | happy |
| FR-084 | Both `ContagionSewerPlagueRestRules` queries MUST leave the rest unchanged for an incubating infection and for no Exhaustion (T6). | SRD 5.2.1, "Sewer Plague" ("suffers the following effects 1d4 days after infection"; "While the creature has any Exhaustion levels"), p. 194 | unhappy (boundary) |
| FR-085 | `filterLongRestEffects` MUST NOT alter the list passed in and MUST return an defensively copied list. | `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-124 | `ContagionDawnReachedStep` on an incubating Sewer Plague infection MUST apply the dawn effect (FR-078 to FR-081), because "an infected creature" makes the dawn save (Q-8): a failure requests `SaveGatedExhaustionGain(1, untilRemoved)` and leaves the Fatigue in `SaveGatedPhaseAwaitingOnset`, so a later onset adds its own level; a success with no Exhaustion returns `Unchanged` (`noExhaustionToReduce`); a success reducing the level to 0 ends the contagion, ending the Fatigue from awaiting onset with no level to remove. | SRD 5.2.1, "Sewer Plague" ("Daily at dawn, an infected creature makes a DC 11 Constitution saving throw"), p. 194 | unhappy (rule interaction) |

### 4.9 Sight Rot

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-086 | `CraftSightRotOintment` with `ContagionHerbalismProficiency.proficient` and `ContagionHerbalismKit.held` MUST return `Crafted` with one `ContagionOintmentDose` and a time of 1 hour. | SRD 5.2.1, "Sight Rot" ("proficient with an Herbalism Kit ... can use it to create one dose of nonmagical ointment, which takes 1 hour"), p. 194 | happy |
| FR-087 | `CraftSightRotOintment` with `notProficient` MUST be rejected (`notProficient`), no dose, whether or not the kit is held. | SRD 5.2.1, "Sight Rot", p. 194 | unhappy (state error) |
| FR-114 | `CraftSightRotOintment` with proficiency and `ContagionHerbalismKit.notHeld` MUST be rejected (`noKit`), no dose ("can be used to create"). | SRD 5.2.1, "Sight Rot", p. 194 | unhappy (state error) |
| FR-088 | `ContagionOintmentAppliedStep(dose)` on a not-suppressed infection MUST set the suppression to `suppressed` and emit one `ContagionSuppressed` of 24 hours listing Blinded as the condition to lift (CE-BLINDED stays active in the hazard model). | SRD 5.2.1, "Sight Rot" ("suppresses the contagion on that creature for 24 hours"), p. 194 | happy |
| FR-089 | `ContagionOintmentAppliedStep` on a suppressed infection MUST be rejected (`alreadySuppressed`), infection unchanged. | SRD 5.2.1, "Sight Rot", p. 194 (silent) | unhappy (state error) |
| FR-090 | `ContagionSuppressionElapsedStep` on a suppressed infection with `completedPeriods` 0 or 1 MUST increase it by 1, set `notSuppressed` and emit one `ContagionSuppressionEnded` listing Blinded to impose again. | SRD 5.2.1, "Sight Rot" ("suppressed ... for a total of 72 hours"), p. 194 | happy |
| FR-091 | `ContagionSuppressionElapsedStep` on a suppressed infection with `completedPeriods` 2 (the third 24-hour period, 72 hours total) MUST end the contagion: `Ended` with cause `ointmentTotal`, the FR-109 `effectID`, one `ContagionEnded`, and the hazard end of CE-BLINDED (end operation, `neutralized`). Blinded is not re-imposed: `ContagionEnded.conditionsStillImposed` is empty and authoritative over the condition list of the hazard `SaveGatedEffectEnded` (the ointment already lifted it; the host does not lift it twice). | SRD 5.2.1, "Sight Rot" ("the contagion ends on the creature"), p. 194 | unhappy (boundary) |
| FR-092 | `ContagionSuppressionElapsedStep` on a not-suppressed infection MUST return `Unchanged` (`notSuppressed`). | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-093 | `completedPeriods` MUST persist between applications: the periods need not be consecutive, and between two applications the infection is not suppressed, with Blinded imposed (T9). | SRD 5.2.1, "Sight Rot" ("a total of 72 hours"), p. 194 | unhappy (rule interaction) |
| FR-094 | `ContagionCuringMagicStep` with `ContagionHealCast`, `ContagionLesserRestorationCast` or `ContagionGmRuledCuringMagic` MUST end the contagion at once: `Ended` with cause `curingMagic`, the FR-109 `effectID`, one `ContagionEnded`, and the hazard end of CE-BLINDED. `ContagionEnded.conditionsStillImposed` MUST list Blinded unless the infection was suppressed (already lifted), authoritative over the hazard condition list as in FR-091. | SRD 5.2.1, "Sight Rot" ("Magic such as a Heal or Lesser Restoration spell ends the contagion immediately"), p. 194 | happy |

### 4.10 Events, results and immutability

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-095 | The events defined here (`DemonicPossessionBegan`, `DemonicEntityTookControl`, `PossessedCreatureRegainedControl`, `DemonicPossessionEnded`, `ContagionContracted`, `ContagionProtectionGranted`, `ContagionSuppressed`, `ContagionSuppressionEnded`, `ContagionEnded`) MUST be the variants of one closed interface `CurseAndContagionEvent` (unexported marker method, variants listed in its doc comment) in `toolbox/curse_and_contagion_event.go`, embedding `ToolboxEvent` (`toolbox-hazard-effects.md` FR-086), so a host type-switches exhaustively. None joins the hazard `SaveGatedEvent`. Each MUST be an immutable past-tense value object, equal by value, carrying the target `CombatantID`. Facts a hazard event already carries are not repeated (effect start `SaveGatedEffectStarted`, repeat save `SaveGatedEffectRepeatSaveMade`, effect end `SaveGatedEffectEnded`). Each event has this spec as sole defining spec. | `doc/rules/architecture.md` §3; shared-types-and-ownership.md FR-019, FR-021; `toolbox-hazard-effects.md` FR-086; `combat-action-resolution.md` FR-090 | happy |
| FR-096 | `DemonicPossessionEnded` MUST carry a `PossessionEndCause`; `ContagionEnded` the `ContagionExample`, a `ContagionEndCause` and `conditionsStillImposed` (`[]ConditionKind`); `ContagionSuppressed` and `ContagionSuppressionEnded` the `ConditionKind` list to lift or impose and, for the first, the 24 hours. | `doc/rules/architecture.md` §3 | happy |
| FR-097 | Events and consequence lists of every result MUST be defensively copied `[]ToolboxEvent` (toolbox-hazard-effects.md FR-086) ordered: (1) `ContagionContracted` first, when present; (2) the hazard events in the order of the hazard operations (for `Infected`, effects started Fever or Fatigue or Blinded; for `Ended`, the `SaveGatedEffectEnded` of the effect that reached its end, then each other effect in the infection's field order, 4.0); (3) this spec's other events in the order listed in the requirement; (4) `ContagionEnded` last, when present. In one consequence set damage precedes conditions (hazard FR-036). | `doc/rules/architecture.md` §4; `doc/rules/testing.md` §4 (determinism) | unhappy (determinism) |
| FR-098 | No contagion operation MUST alter the infection, state or lists passed in. | `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |

### 4.11 Save binding, flow, use cases and telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-112 | Every command depending on a save (`PossessionTargeted`, `PossessionTurnEnded`, `PossessionLongRestFinished`, exposure, recuperation, Long Rest, dawn and laughter steps) MUST carry a hazard `SaveGatedSave`. `SaveGatedSaveNotRolled` MUST return that use case's `SaveRequired` with the T2 `SaveGatedRollPlan` and request no dice. A `SaveGatedSaveRolled` whose `CheckResult` is not a saving throw, or whose target `DifficultyClass` differs from the T2 constant, MUST return the operation's `Rejected` (`ResolveContagionExposureRejected` for exposure) with `saveDoesNotMatchRequirement`, nothing rolled, state unchanged (toolbox-hazard-effects.md FR-039). `SaveGatedSaveForcedFailure` counts as a failure. `CheckResult` carries no ability (d20-tests.md FR-041), so the plan hands the ability to the host (Q-25). | SRD 5.2.1, "Saving Throws", p. 7; toolbox-hazard-effects.md FR-030, FR-039; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-115 | An operation whose dice depend on its decision MUST be split into pure steps: (1) the domain returns a plan or decision (`ContagionExposureRules.check` and `decide`, `ContagionLaughterRules.check`, the hazard planner); (2) the use case asks `DiceRequired` which `dice.Expression`s to roll, rolls exactly that list through `dice.Roll` in the order of toolbox-hazard-effects.md FR-070, and rolls nothing when empty; (3) it passes the `dice.Result`s to a pure transition (`ContagionInfection.begin`, the evaluator). The use case decides nothing else about dice, which makes FR-054, FR-072 and FR-116 checkable. | `doc/rules/architecture.md` §4; toolbox-hazard-effects.md FR-070, FR-087 | happy; unhappy (empty and minimal) |
| FR-116 | `ResolveContagionProgress` MUST return a dice port failure from the laughter damage roll (FR-071) wrapped once as `*srderr.PortError` (an invalid result is the same kind of error through `dice.Roll`), with a nil result: no consequences or event, the infection untouched. | `doc/specs/dice-and-rolls.md`; `doc/rules/architecture.md` §4; toolbox-hazard-effects.md FR-069 | unhappy (collaborator failure) |
| FR-099 | Each use case (`ResolveDemonicPossession`, `ResolveContagionExposure`, `ResolveContagionRecuperation`, `ResolveContagionProgress`, `CraftSightRotOintment`) MUST return the same variant, state and lists as the domain operation it wraps, as its own closed interface `<UseCase>Result` with operation-prefixed variants (`ResolveContagionExposureInfected`, `ResolveDemonicPossessionBegan`, `CraftSightRotOintmentCrafted`). No generic variant name (`Infected`, `Resisted`, `Ended`, `Rejected`, `Unchanged`, `Failed`, `Crafted`) is exported; shared failure detail is a value object in a `reason` field. Every `events` field is a `[]ToolboxEvent` (FR-097). | `doc/rules/architecture.md` §4; shared-types-and-ownership.md FR-017, FR-018 | happy |
| FR-100 | Each use case MUST give identical results with the no-op `telemetry.Telemetry` and a recording one. | `doc/rules/observability.md` §9 | happy |
| FR-101 | Each call MUST emit one span, increment `srd5e.toolbox.operations` once and record `srd5e.toolbox.operation_duration` (section 7). Each decided save (not `save_required`, a rejection or a port failure) MUST increment `srd5e.toolbox.saves` once with its ability and result (toolbox-hazard-effects.md FR-079); a `SaveGatedNoSave` failure (Fever, Fatigue, Blinded start) has no ability and MUST NOT. `ResolveContagionProgress` MUST record `srd5e.toolbox.damage_rolled` once per call returning laughter damage (rolled or printed average), with the sum of the returned `Damage` amounts, and nothing otherwise (toolbox-hazard-effects.md FR-080). | `doc/rules/observability.md` §5, §6; toolbox-hazard-effects.md FR-072, FR-079, FR-080, FR-081 | happy |
| FR-102 | `srd5e.outcome` MUST follow the section 7 table: a **rule outcome** is `success`, a **host-misuse result** `failure`. Rule outcomes: every `Refused` exposure, `Resisted`, `ResistedProtected`, `Prevented`, `Interrupted`, a failed save, `SaveRequired`, and `Unchanged` with `notALongRest` (FR-023); no error counter, no log. Host-misuse: every `Rejected`, every `NotApplicable`, and every `Unchanged` naming a step that cannot apply (`notPossessed`, `noSaveDue`, `seizureTurnNotLater`, `notIncubating`, `notSuppressed`, `notLaughing`). A host-misuse result MUST set span status error, counted once in `operations`, with no `errors` increment, no `error.type` (errors-and-results.md FR-043) and no log (toolbox-hazard-effects.md FR-073). | `doc/rules/observability.md` §6, §7; shared-types-and-ownership.md FR-041; toolbox-hazard-effects.md FR-073 | unhappy (state error) |
| FR-103 | A dice port failure in `ResolveContagionExposure` or `ResolveContagionProgress` MUST be recorded once at the use-case boundary: one `span.RecordError`, one `srd5e.toolbox.errors` increment with `srd5e.toolbox.operation` and `error.type` `dice_roller_error`, no log, one `operations` increment with `srd5e.outcome` `failure` and `srd5e.toolbox.result` `error`, duration recorded; the wrapped error is then returned. | `doc/rules/observability.md` §7; toolbox-hazard-effects.md FR-074, FR-078 | unhappy (collaborator failure) |
| FR-104 | An `ErrInvalidInput` error from a domain constructor inside a use case is a bug and MUST be returned unchanged (not wrapped as a `PortError`); the span records it, no error counter or log records it again; the duration IS recorded, `operations` is NOT. A host value failing validation is a returned `Rejected` (FR-112). | `doc/rules/observability.md` §7; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-105 | Telemetry attributes MUST come from the closed sets of section 7 and MUST NOT contain a `CombatantID`, a species or free text. | `doc/rules/observability.md` §5 | unhappy (invalid input) |
| FR-106 | A result of `Ended` MUST increment `srd5e.toolbox.contagion.ended` once with the contagion and end cause, in addition to `operations`. | `doc/rules/observability.md` §5 | happy |
| FR-107 | No use case here MUST log at any level: results carry the detail, span and counters the aggregate. | `doc/rules/observability.md` §7 ("Log intelligently") | unhappy (state error) |
| FR-120 | No use case here MUST call another use case (of this spec, the hazard model or any context). It composes the pure domain functions of its own context (hazard planner, `DiceRequired`, evaluator, this spec's domain operations), supplies dice through its `Roller`, and records only its own signals with attribute values from the domain result. An architecture check finds no use case import and no `d20.Resolver` import in the five use cases' files. | shared-types-and-ownership.md FR-044; `doc/rules/architecture.md` §4; `doc/rules/observability.md` §7 | unhappy (forbidden edge) |

### 4.12 Case tables

One test per row (`doc/rules/testing.md` §5, §6); source SRD 5.2.1 p. 194 (p. 193 for T1, T10) unless stated; deviations: T1 Advantage and Disadvantage rows pp. 193, 181, 7-8; T6 p. 187; T11 p. 7; T12 p. 181. A row's requirement is the FR in its table title or Expected cell, and its Path is that requirement's Path in section 4; a row that lists several inputs separated by ';' or '/' is one test per input and takes the path of the requirement it exercises (a refusal, boundary or rejection is unhappy, a decided infection or grant is happy).

| T1. Natural 1 (FR-010 to FR-015) | State | Roll (dice, used) | Expected |
|---|---|---|---|
| 1, each D20 Test kind (ability check, saving throw, attack roll) | `PossessedCreatureInControl` | 1, used 1 | `DemonicEntityInControl` (FR-010, FR-015) |
| 2 (ability check); 20 (attack roll) | `PossessedCreatureInControl` | 2; 20 | unchanged `noControlLossRoll` (FR-011) |
| Advantage, 1 and 17 | `PossessedCreatureInControl` | used 17 | unchanged `noControlLossRoll` (FR-014) |
| Disadvantage, 1 and 17 | `PossessedCreatureInControl` | used 1 | `DemonicEntityInControl` (FR-014) |
| 1, entity already in control | `DemonicEntityInControl` | 1, used 1 | unchanged `entityAlreadyInControl` (FR-012) |
| 1, not possessed | `NotPossessed` | 1, used 1 | unchanged `notPossessed` (FR-013) |

| T2. Save constants (FR-006, FR-031, FR-038) | Purpose | Ability | DC |
|---|---|---|---|
| Cackle Fever | `exposure` (spread) | Constitution | 10 |
| Cackle Fever | `fightingTheContagion` (Long Rest); `uncontrollableLaughter`; `laughterRepeat` | Constitution | 13 |
| Sewer Plague | `exposure`; `fightingTheContagion` (dawn) | Constitution | 11 |
| Sight Rot | `exposure` (tainted water, skin contact) | Constitution | 15 |
| any magical contagion | `recuperation` | Constitution | 15 |
| Demonic Possession (p. 193; not a contagion) | n/a | Charisma | 15 |

| T3. Exposure (FR-041 to FR-052, FR-056) | Contagion and route | Creature; held state | Expected |
|---|---|---|---|
| range, 0 / 10 / 11 ft | Cackle Fever, emanation | Humanoid, human; none | save DC 10; save DC 10; refused `outOfRange` (FR-047) |
| negative distance | Cackle Fever, emanation -1 | Humanoid, human; none | `ErrInvalidInput` error (FR-048) |
| gnome | Cackle Fever, emanation 5; Sewer Plague, wounded | Humanoid, gnome; none | refused `speciesImmune`; save DC 11 (FR-043) |
| Beast | Sight Rot, water | Beast, no species; none | save DC 15 (FR-042) |
| Beast, other | Cackle emanation 5; Sight Rot skin; Sewer Plague filth | Beast, no species; none | refused `creatureTypeNotAffected` (FR-042) |
| wrong route | Cackle Fever, tainted water | Humanoid, human; none | refused `routeNotApplicable` (FR-041) |
| immune one year | Cackle Fever, emanation 5 | Humanoid, human; immune to Cackle Fever | refused `immuneForOneYear` (FR-044) |
| already infected | Sewer Plague, filth | Humanoid, human; infected with Sewer Plague | refused `alreadyInfected` (FR-045) |
| protection | Sight Rot, skin | Humanoid, human; protected from carrier A, exposed by A; by B | refused `protectedFromSource`; save DC 15 (FR-046) |
| refusal order | route before type: Cackle Fever, tainted water, Beast; species before immunity: Cackle emanation 5, gnome, immune; immunity before infected: Cackle emanation 5, human, immune and infected; infected before protection: Cackle emanation 5, human, infected and protected; protection before range: Cackle emanation 30, human, protected | as named | `routeNotApplicable`; `speciesImmune`; `immuneForOneYear`; `alreadyInfected`; `protectedFromSource` (FR-049) |
| save fails | Sewer Plague, wounded | Humanoid, human; none | infected, incubating (FR-050, FR-052) |
| save succeeds, protected routes | Cackle emanation 5; Sight Rot skin | Humanoid, human; none | `ResistedProtected` with 24-hour protection (FR-051) |
| save succeeds, other routes | Sewer Plague filth; Sight Rot water | Humanoid, human; none | `Resisted`, no protection field (FR-051) |

| T4. Onset (FR-052, FR-053, FR-055) | Contagion | Face | Expected |
|---|---|---|---|
| Cackle Fever; Sewer Plague | Cackle Fever; Sewer Plague | 1, 4; 2, 3 | Fever `SaveGatedPhaseAwaitingOnset`, onset 1 day, 4 days; Fatigue awaiting onset, 2 days, 3 days |
| Sight Rot | Sight Rot | no roll | CE-BLINDED `SaveGatedPhaseActive(0)`, Blinded in the consequences |
| failing roller | Cackle Fever | error | wrapped `*srderr.PortError` returned, nil result |

**T5. Step applicability (FR-060, FR-061).** One test per step and contagion. Applicable: `ContagionOnsetReachedStep` to
Cackle Fever and Sewer Plague; `ContagionLongRestFinishedStep`, `ContagionDamageTakenStep`, `ContagionLaughterTurnEndedStep`, `ContagionLaughterMinuteElapsedStep` to Cackle Fever only; `ContagionDawnReachedStep` to Sewer Plague only; `ContagionOintmentAppliedStep`,
`ContagionSuppressionElapsedStep`, `ContagionCuringMagicStep` to Sight Rot only. Every other pairing is `NotApplicable`.

| T6. Sewer Plague rests (FR-082 to FR-084, FR-123) | Exhaustion level | Normal healing | Hit Die healing | Long Rest effects kept |
|---|---|---|---|---|
| active | 1; 3; 5 | 7; 8; 5 | 3; 4; 2 | all but `AllLostHitPointsRegained`, `ExhaustionLevelReduced` |
| active, 1 normal (Q-14) | 2 | 1 | 0 (half of 1, rounded down) | same |
| active, no Exhaustion; incubating | none; 1 | 7 | 7 | all |
| Cackle Fever active; Sight Rot | 1; 3 | 7; 8 | 7; 8 | all |

| T7. Cackle Fever Long Rest saves (FR-064 to FR-069) | Sequence of saves | Fight successes then | Ended |
|---|---|---|---|
| one, two successes | success; success, success | 1; 2 | no |
| three successes | success x3 | ends on the third | yes |
| failure keeps the count | success, failure | 1 | no |
| not consecutive | success, failure, success, failure, success | ends on the third success | yes |
| all failures | failure x4 | 0 | no |
| three successes, incubating | success x3, Fever awaiting onset | ends; Exhaustion to remove 0 | yes |
| three successes after onset | onset, success x3 | ends; Exhaustion to remove 1 | yes |
| short rest | `RestKind.shortRest`, success | unchanged, `notALongRest` | no |

| T8. Laughter trigger (FR-070, FR-123) | Phase; Exhaustion | Damage type | Expected |
|---|---|---|---|
| Slashing; Fire | active; 1; 5 | Slashing; Fire | save DC 13 |
| Psychic | active; 1 | Psychic | refused `psychicDamage` |
| no Exhaustion | active; none | Slashing | refused `noExhaustion` |
| incubating; order (not active before no Exhaustion; no Exhaustion before Psychic) | incubating, 1, Slashing; incubating, none, Psychic; active, none, Psychic | as named | `notActive`; `notActive`; `noExhaustion` |
| other infections | Sewer Plague active; Sight Rot; 1 | Slashing | refused `notActive` |

| T9. Sight Rot suppression (FR-090, FR-091, FR-093) | Sequence | Expected |
|---|---|---|
| one, two applications | (apply, elapse) x1; (apply, elapse) x2 | count 1; 2, Blinded to impose again |
| three applications | (apply, elapse) x3 | ends, cause `ointmentTotal`, no Blinded |
| gap between applications | apply, elapse, (days pass), apply, elapse | count 2 |
| elapse without application | elapse | `Unchanged` `notSuppressed` |

| T10. Curse potency (FR-002 to FR-004) | Duration | Expected |
|---|---|---|
| exactly 1 minute | 60 seconds | level 3 |
| until dispelled | `CurseUntilDispelled` | level 9 |
| any other | 59 s; 61 s; 10 minutes; 24 hours | unavailable |
| zero, negative | 0 seconds; -1 second | `ErrInvalidInput` error |

| T11. Save binding (FR-112) | Operation and supplied save | Expected |
|---|---|---|
| not rolled | Cackle Fever spread; `PossessionTargeted` (`SaveGatedSaveNotRolled`) | `SaveRequired`, Constitution DC 10; Charisma DC 15 (p. 193); no dice |
| wrong kind (p. 7) | Long Rest step; Sewer Plague exposure; `CheckResult` of kind `abilityCheck` | `Rejected(saveDoesNotMatchRequirement)` (exposure: `ResolveContagionExposureRejected`), nothing rolled |
| wrong DC | possession vs 12 (needs 15); Long Rest vs 15 (13); dawn vs 13 (11); recuperation vs 13 (15); Cackle exposure vs 11 (10) | `Rejected(saveDoesNotMatchRequirement)`, nothing rolled, state unchanged |
| forced failure | `PossessionTargeted`, `ForcedFailureResult` | counts as a failure: possessed (FR-007) |
| matching save | exposure, saving throw `CheckResult` against DC 10 | decided by its total |

| T12. Dawn save (FR-078 to FR-081, FR-124) | Exhaustion level; save | Expected |
|---|---|---|
| fail at level 1 | 1; failure | Exhaustion gain 1; continues (FR-078) |
| fail at level 5 | 5; failure | gain 1, to level 6; death is the host's (p. 181) (FR-078) |
| succeed at level 3; 2 | 3; 2; success | `SaveGatedExhaustionReduction(1)`; continues (FR-079) |
| succeed at level 1 | 1; success | reduction 1; `Ended` `exhaustionReducedToZero`, `effectID` carried (FR-080) |
| succeed with none | none; success | `Unchanged` `noExhaustionToReduce` (FR-081) |
| incubating, none | none, Fatigue awaiting onset; failure; success | gain 1, Fatigue stays awaiting onset; `Unchanged` `noExhaustionToReduce` (FR-124) |
| incubating, level 1, succeed | 1, awaiting onset; success | `Ended` `exhaustionReducedToZero`; Fatigue ended from awaiting onset, no level to remove (FR-124) |

### 4.13 Catalogue entries and composition

Content kind `contagion` (row R3-contagions, `content-schema-and-decoder.md` FR-129; shared-types-and-ownership.md FR-045).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-117 | `ContagionDefinition(id, name, infectionDc, fightingDc, source)` MUST be a `content` value object (package `content`, built with `NewContagionDefinition(...) (ContagionDefinition, error)`), comparable by value: `id` a `ContagionID`, `name` non-blank, `infectionDc` an integer of at least 1, `fightingDc` absent or at least 1, `source` a `SourceCitation` (`content-schema-and-decoder.md` FR-115). An argument outside those bounds MUST return an error wrapping `srderr.ErrInvalidInput` (`MustContagionDefinition` panics). The entry holds no route, creature type, effect or timing. | SRD 5.2.1, "Example Contagions", p. 194; `content-schema-and-decoder.md` FR-129; `doc/rules/architecture.md` §3 | happy; unhappy (invalid input) |
| FR-118 | `ContagionExampleFromID(ContagionID) (ContagionExample, bool)` MUST return `ContagionCackleFever` for `cackle_fever`, `ContagionSewerPlague` for `sewer_plague`, `ContagionSightRot` for `sight_rot`, and `ok` false for any other id (a host-authored contagion has no rules here, Q-3). Ids follow shared-types-and-ownership.md FR-007. | SRD 5.2.1, "Example Contagions", p. 194; shared-types-and-ownership.md FR-007 | unhappy (boundary) |
| FR-119 | The three bundled entries MUST equal T2: Cackle Fever `infectionDc` 10 and `fightingDc` 13, Sewer Plague 11 and 11, Sight Rot 15 and absent; every infection constant of FR-038 comes from code constants, never a loaded entry. A test decodes the embedded `content/data/contagions.json` through the content decoder (shared-types-and-ownership.md FR-031) and compares it with T2. | SRD 5.2.1, "Cackle Fever", "Sewer Plague", "Sight Rot", p. 194; shared-types-and-ownership.md FR-031, FR-047 | unhappy (rule interaction) |
| FR-121 | The registration MUST request `ContentKind.contagion`, `ContagionID` and root array `contagions` as row R3-contagions of the schema revision table and MUST NOT claim a `schemaVersion` (shared-types-and-ownership.md FR-034); it applies with the toolbox data change (phase 5). This spec adds the bundled embedded library to `BundledContentSource` (shared-types-and-ownership.md FR-047). | `content-schema-and-decoder.md` FR-129; shared-types-and-ownership.md FR-034, FR-047 | happy |
| FR-122 | The `contagions` array decoded by the content decoder MUST hold objects with exactly these lower-camel-case keys: `id` (required; `[a-z][a-z0-9_]*`, shared-types-and-ownership.md FR-007; becomes the `ContagionID`), `name` (required; non-blank), `infectionDc` (required; integer at least 1), `fightingDc` (optional; integer at least 1; absent for Sight Rot) and `source` (optional `{document, section, page}`, `content-schema-and-decoder.md` FR-034). The decoder MUST report these as typed problems in an `errors.Join`, never panic (`content-schema-and-decoder.md` FR-078): a missing `id`, `name` or `infectionDc` as `missing_field`; a wrong JSON type `wrong_type`; `infectionDc` or `fightingDc` below 1 `out_of_range`; a blank name, invalid id or an explicit JSON `null` for `fightingDc` `invalid_value`; any other key `unknown_field`; a repeated id `duplicate` (kinds of `content-schema-and-decoder.md` FR-100). Every accepted entry MUST build a `ContagionDefinition` without an `ErrInvalidInput` error (FR-117). | `content-schema-and-decoder.md` FR-034, FR-078, FR-100, FR-129; shared-types-and-ownership.md FR-007, FR-045; SRD 5.2.1, "Example Contagions", p. 194 | unhappy (invalid input) |
| FR-123 | Queries taking any `ContagionInfection` MUST define every pairing. `ContagionLaughterRules.check` for Sewer Plague or Sight Rot MUST return `ContagionLaughterNotDue(notActive)`. `ContagionSewerPlagueRestRules.hitDieHealing` for Cackle Fever or Sight Rot MUST return `normal`, and `filterLongRestEffects` for either MUST return the list unchanged (an defensively copied copy, FR-085): Weakness and Restlessness are Sewer Plague only. | SRD 5.2.1, "Uncontrollable Laughter" (Cackle Fever only), "Weakness", "Restlessness" (Sewer Plague only), p. 194 | unhappy (rule interaction) |

| T13. Catalogue entries (FR-117 to FR-119, FR-122) | Input | Expected | Req. |
|---|---|---|---|
| the three entries | `cackle_fever` 10, 13; `sewer_plague` 11, 11; `sight_rot` 15, no fighting DC | built; `tryFromID` gives `cackleFever`, `sewerPlague`, `sightRot` | FR-117, FR-118 |
| unknown id | id `green_slime` | `ContagionExampleFromID` returns `ok` false | FR-118 |
| bad bounds | blank name `" "`; `infectionDc` 0; `fightingDc` 0 | `ErrInvalidInput` error | FR-117 |
| bundled drift | embedded entries against T2 | equal | FR-119 |
| JSON, Sight Rot | `{"id":"sight_rot","name":"Sight Rot","infectionDc":15}` | decodes; no fighting DC | FR-122 |
| JSON, field problems | missing `infectionDc` (`{"id":"x","name":"X"}`); `"infectionDc":"10"`; `"fightingDc":0` | `missing_field`; `wrong_type`; `out_of_range` | FR-122 |
| JSON, value problems | `"fightingDc":null`; `"id":"Sight-Rot"` | `invalid_value` | FR-122 |
| JSON, keys | `"route":"water"`; two entries `sight_rot` | `unknown_field`; `duplicate` | FR-122 |

### 4.14 Withdrawn ids

| Id | Status |
|---|---|
| FR-057 | Withdrawn: moved to toolbox-hazard-effects.md (FR-048) |
| FR-059 | Withdrawn: moved to toolbox-hazard-effects.md (FR-063) |

## 5. Model

- **Referenced, not defined:** `SaveGated...` types and hazard domain functions (`toolbox-hazard-effects.md`); `OngoingEffect`,
  `PossessionPrevention`, `PossessionEnding`, `CurseRemovalMeans` (`magical-effects.md`).
- **Value objects (`toolbox` domain):** the section 3 types, `CurseAndContagionEvent` and its nine events;
  `ContagionDefinition` (FR-117) is a `content` value object whose shape this spec owns.
- **Domain services (pure):** `DemonicPossession`, `ContagionExposureRules` (`check`, `decide`), `ContagionLaughterRules`,
  `ContagionSewerPlagueRestRules`, `CursePotency`, `ContagionSaveCircumstances.of`. Infection variants carry the step
  transitions (FR-058 to FR-094) and `begin` (FR-115), calling the hazard planner, `DiceRequired` and evaluator with
  already-rolled dice. These queries have no use case and no telemetry. No aggregates: the host holds one
  `DemonicPossessionState` per creature and one `ContagionInfection` per creature and contagion (creatures are `CombatantID`s).
- **Invariants** (an error wrapping `srderr.ErrInvalidInput` from `NewX`, `MustX` panics; `doc/rules/architecture.md` §6): FR-001, FR-035, FR-039, FR-048, FR-113, FR-117.
  A host value failing validation is a returned `Rejected` (FR-112); the only error a use case returns is a dice port failure (wrapped once as `*srderr.PortError`) or a cancelled `ctx`. A persisted infection goes through the constructors; no separate non-failing constructor until needed (Q-26).
- **Hand-off:** `SaveGatedConsequences` and `SaveGatedEffectEnded` are requests the host applies, feeding the new Exhaustion back.

## 6. Interface

Synchronous: the only port is the synchronous `Roller`; a dice port failure is returned wrapped once (`dice.Roll`), with no absence
result, and no new double or contract suite is needed. Each use case is built with `NewX(roller, opts ...Option)` where it rolls (options `WithTelemetry`, `WithClock`; default `telemetry.Noop`), has one method `Run(ctx context.Context, cmd) (<UseCase>Result, error)`, returns its own closed interface `<UseCase>Result` (`...X` below is exported as
`<UseCase>X`, FR-099), and holds `[]ToolboxEvent` in every `Events` field.

- `ResolveDemonicPossession(ResolveDemonicPossessionCommand)`, no port. The closed interface command always carries the target
  `CombatantID` and `DemonicPossessionState`: `PossessionTargeted(preventions: []PossessionPrevention (treated as a set), save)`,
  `PossessionD20TestRolled(roll: Roll, kind: D20TestKind)`, `PossessionTurnEnded(turn: PossessionTurn, save)`,
  `PossessionLongRestFinished(rest: RestKind, save)`, `PossessionMagicApplied(magic: PossessionEndingMagic)` (`save` is a
  `SaveGatedSave`). Result: `...SaveRequired(plan)`, `...Began(state, events)`, `...Resisted(state)`,
  `...Prevented(prevention)`, `...ControlSeized(state, events)`, `...ControlRegained(state, events)`,
  `...Ended(state, events)`, `...Unchanged(state, reason: PossessionUnchangedReason)`, `...Rejected(state, reason: PossessionRejectionReason)`.
- `ResolveContagionExposure(ResolveContagionExposureCommand)`, takes a `Roller`. Command: contagion,
  `ContagionExposedCreature`, `ContagionExposureRoute`, `ContagionHeldState`, target `CombatantID`, the host's
  `OngoingEffectID` and `EffectSource` (FR-109), a `SaveGatedSave`. Result: `...SaveRequired(plan)`,
  `...Infected(infection, effect: OngoingEffect, consequences, events)`, `...ResistedProtected(protection, events)` (routes
  of FR-051), `...Resisted(events)`, `...Refused(reason: ContagionRefusalReason)`, `...Rejected(reason: ContagionRejectionReason)` (`saveDoesNotMatchRequirement` only).
- `ResolveContagionRecuperation(ResolveContagionRecuperationCommand)`, no port. Command: `ContagionRecuperationReport`,
  a `SaveGatedSave`. Result: `...SaveRequired(plan)`, `...AdvantageGranted(window)`, `...Failed`,
  `...Interrupted(interruption)`, `...Rejected(reason: ContagionRejectionReason)`.
- `ResolveContagionProgress(ResolveContagionProgressCommand)`, takes a `Roller` (laughter damage only, FR-115).
  Command: infection, target, hazard `SaveGatedSubject` (carries the Exhaustion level), `SaveGatedDamageAmount` (default
  rolled) and a closed interface `ContagionStep`: `ContagionOnsetReachedStep`; `ContagionLongRestFinishedStep(rest: RestKind, save,
  window: ContagionAdvantageWindow)`; `ContagionDawnReachedStep(save, window)`; `ContagionDamageTakenStep(damageType:
  DamageType, save)`; `ContagionLaughterTurnEndedStep(save)`; `ContagionLaughterMinuteElapsedStep`;
  `ContagionOintmentAppliedStep(dose)`; `ContagionSuppressionElapsedStep`; `ContagionCuringMagicStep(magic)`. Result:
  `...SaveRequired(plan)`, `...Advanced(infection, consequences, events)`, `...Ended(effectID: OngoingEffectID,
  consequences, events, immunities: []ContagionImmunityGrant)`, `...Unchanged(infection, reason: ContagionUnchangedReason)`,
  `...NotApplicable(infection)`, `...Rejected(infection, reason: ContagionRejectionReason)`. The dawn Exhaustion reduction travels in
  `consequences` as `SaveGatedExhaustionReduction` (it appears only in `SaveGatedConsequences`, toolbox-hazard-effects.md FR-106).
- `CraftSightRotOintment(CraftSightRotOintmentCommand)`, no port. Command: `ContagionHerbalismProficiency`,
  `ContagionHerbalismKit`. Result: `...Crafted(dose, hours)`, `...Rejected(reason: ContagionRejectionReason)`.
- **Host protocol** (`toolbox/doc.go`): call with `SaveGatedSaveNotRolled`; on `SaveRequired` roll the plan's
  ability and DC through `d20.Resolver` and call again with the finished save. Apply `consequences`, add the `effect` of
  `Infected`, remove the `OngoingEffect` named by the `effectID` of `Ended` (FR-109), store the new infection, and report
  the next step when its trigger occurs. Time is reported, never read (Q-7).
- **Exports:** package `toolbox` is owned by `toolbox-hazard-effects.md` (FR-001, FR-006); this spec adds the
  section 3 types, five use cases with commands and results, and the `CurseAndContagionEvent` family; `ContagionDefinition`
  is exported from `content`. Each name is exported once with a doc comment starting with its name; there is no re-export package.
  `toolbox` is downstream of `content`, `d20`, `character` and `combat` and nothing depends on it
  (shared-types-and-ownership.md FR-045; `doc/rules/architecture.md` §2); `dice` is imported by the `*_usecase.go` files only.

## 7. Observability

The pure queries and `ContagionDefinition` are **silent by design** (`doc/rules/observability.md` §3). The five use cases use the **shared `toolbox`
signals** of `toolbox-hazard-effects.md` section 7 (`srd5e.toolbox.operations`, `.saves`, `.damage_rolled`, `.operation_duration`, `.errors` and the
per-use-case spans; FR-101, FR-103; HA-3, toolbox-hazard-effects.md Q-15); this spec lists its own operation, `result`, step, kind and end-cause values below, adds their constants to the `toolbox` signal constants and owns only the signal marked new.

Evaluation questions, in order: (1) a `Roller` fault; host misuse (wrong-state step, save for another DC or kind); an error wrapping `srderr.ErrInvalidInput` bug; wrong
rule results (a missed gnome immunity). (2) Counted by operation, contagion and result with `srd5e.outcome` and span status (FR-102); port failures
reach `srd5e.toolbox.errors`. (3) Operation, step, contagion, result, end cause. (4) Operations, outcomes, saves, endings by cause, errors, duration.
(5) Shared signals reused; no parallel `contagion.*` operations, duration or errors. (6) Any non-zero `errors`; a `failure` share above 5% of an
operation; `refused_already_infected` above 10% of exposures; `unchanged_no_save_due` above 20% of possession turn ends; p99 duration above 2x its
7-day baseline. (7) Nothing noisy: no log (FR-107).

**Operation values** added to `srd5e.toolbox.operation`: `resolve_demonic_possession`, `resolve_contagion_exposure`, `resolve_contagion_recuperation`,
`resolve_contagion_progress`, `craft_sight_rot_ointment`.

**New attribute keys** (closed sets, constants in the `toolbox` signal constants): `srd5e.toolbox.curse.step` is one of `targeted`, `d20_test_rolled`, `turn_ended`,
`long_rest_finished`, `magic_applied` (possession) or `onset_reached`, `long_rest_finished`, `dawn_reached`, `damage_taken`, `laughter_turn_ended`,
`laughter_minute_elapsed`, `ointment_applied`, `suppression_elapsed`, `curing_magic` (progress); `srd5e.toolbox.contagion.kind` is one of
`cackle_fever`, `sewer_plague`, `sight_rot`, `none`; `srd5e.toolbox.contagion.end_cause` is one of `three_long_rest_saves`,
`exhaustion_reduced_to_zero`, `ointment_total`, `curing_magic`. Spans also carry `srd5e.d20_test.kind` on `d20_test_rolled` only (an existing d20 key reused with its closed set, resolve-d20-test-use-case.md FR-038; not redefined here).

Allowed `srd5e.toolbox.result` values and the outcome each maps to (FR-102); `error` (a dice port error left the call, FR-103) is `failure` for
every use case:

| Use case | `success` results | `failure` results |
|---|---|---|
| `resolve_demonic_possession` | `save_required`, `began`, `resisted`, `prevented`, `control_seized`, `control_regained`, `ended`, `unchanged_no_control_loss_roll`, `unchanged_entity_already_in_control`, `unchanged_save_failed`, `unchanged_not_a_long_rest` | `unchanged_not_possessed`, `unchanged_no_save_due`, `unchanged_seizure_turn_not_later`, `rejected_already_possessed`, `rejected_save_does_not_match_requirement` |
| `resolve_contagion_exposure` | `save_required`, `infected`, `resisted`, `resisted_protected`, `refused_route_not_applicable`, `refused_creature_type_not_affected`, `refused_species_immune`, `refused_immune_for_one_year`, `refused_already_infected`, `refused_protected_from_source`, `refused_out_of_range` | `rejected_save_does_not_match_requirement` |
| `resolve_contagion_recuperation` | `save_required`, `advantage_granted`, `failed`, `interrupted` | `rejected_save_does_not_match_requirement` |
| `resolve_contagion_progress` | `save_required`, `advanced`, `ended`, `unchanged_save_failed`, `unchanged_no_exhaustion_to_reduce`, `unchanged_not_a_long_rest` | `unchanged_not_incubating`, `unchanged_not_suppressed`, `unchanged_not_laughing`, `not_applicable`, `rejected_already_suppressed`, `rejected_save_does_not_match_requirement` |
| `craft_sight_rot_ointment` | `crafted` | `rejected_not_proficient`, `rejected_no_kit` |

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.toolbox.contagion.ended` (new) | counter `{infection}` | `srd5e.toolbox.contagion.kind`; `srd5e.toolbox.contagion.end_cause` | How contagions end (FR-106); feeds `doc/observability/signals.md`, constant in the `toolbox` signal constants | Dashboard; an end cause that never occurs for a contagion in use suggests the host skips a step |

## 8. Test plan

Doubles from `dicetest`, `telemetrytest`, `contenttest` (shared-types-and-ownership.md FR-038): `dicetest.FixedRoller` (`1d4`, `1d10` faces), a failing roller double (`dicetest.CallbackRoller`), the `telemetrytest` recorder; no new contract suite. Mutation testing (`doc/rules/testing.md` §8): `gomu` on package `toolbox`, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (command in testing.md), score at least 80. Table-driven per case table (T1 to T13), one row per `t.Run`.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-005 | unit | T10; zero and negative `Duration` return an `ErrInvalidInput` error; equal inputs, equal results. |
| FR-006 to FR-030, FR-108, FR-113 | unit, application, architecture, scenario | T1 (three kinds; Advantage and Disadvantage with a two-dice `Roll`); fail begins and emits, success stays; already possessed rejected in both states; regain on success, unchanged on failure; no save due in both control states and on the seizing turn with a success supplied; Long Rest save from both states; `shortRest` is `Unchanged` `notALongRest` with outcome `success`; each ending magic and `CurseRemovalMeans`; not possessed; prevented from `NotPossessed` (no save requested), already possessed rejected first; `possessingEffect` returns an `ErrInvalidInput` error; the sequence of FR-028 on the domain (optionally a public-API scenario); inputs unchanged, events defensively copied; no `Roller` import in `resolve_demonic_possession_usecase.go` (the use case needs none), no `time.Now` in possession domain files. |
| FR-031 to FR-036 | unit, application | 5 purposes by held or not; full period, 24-hour grant, failure no grant; four interruptions; `daysBefore` 0 and 2 valid, -1 and 3 return an `ErrInvalidInput` error; circumstances by purpose and window, a Long Rest plan carries the extra Advantage. |
| FR-037 to FR-040, FR-111 | unit | Enum values; T2; 4.0 effects equal T2 constants; each invalid construction; equality and hash. |
| FR-041 to FR-056 | unit, application | T3 incl. order rows; negative distance returns an `ErrInvalidInput` error; `check` returns plan or first refusal with no roll; Infected (with the `OngoingEffect`), `ResistedProtected`, `Resisted`; onset per `1d4` face (T4); Sight Rot no roll; zero roller requests when refused or resisted; failing roller returns one wrapped `*srderr.PortError`, invalid result via `dice.Roll`. |
| FR-058, FR-060 to FR-069 | unit, application | Onset on incubating and active infections; T5 one test per cell; Exhaustion lifetimes; T7; short rest `notALongRest`; incubating end without Fever release; `Ended` carries `effectID`. |
| FR-070 to FR-077, FR-110 | unit, application | T8; `1d10` faces 1 and 10 (damage, Incapacitated, start event); printed average requests no dice; success makes no roll; repeat success and failure; after the minute; not laughing; laughing again replaces the state; ending Cackle Fever while laughing lists Incapacitated. |
| FR-078 to FR-085, FR-123, FR-124 | unit, application | T12; T6 and T8 other-variant rows; list not mutated and defensively copied. |
| FR-086 to FR-094, FR-114 | unit, application | Crafted with 1 hour; rejected without proficiency (kit held or not) and without the kit; apply, double apply, elapse 1 and 2, third elapse, unsuppressed elapse, T9, magic cure from suppressed and not (`conditionsStillImposed`). |
| FR-095 to FR-098 | unit, architecture | Event equality, past-tense names, fields; every event a `part` of the closed interface `CurseAndContagionEvent` implementing `ToolboxEvent`, exhaustive `switch` compiles, no event duplicates a hazard event; order: `ContagionContracted` first; Fever, Fatigue or Blinded `SaveGatedEffectStarted` in order (Infected); the triggering effect's `SaveGatedEffectEnded` first and `ContagionEnded` last (Ended); lists defensively copied; inputs unchanged. |
| FR-099 to FR-107 | application | Result parity with the domain; same results with no-op and recording `telemetrytest` bundle; one span, counter and duration per call; `saves` once per decided save, not for a `SaveGatedNoSave` failure; `damage_rolled` once (rolled or printed average); outcome and span status per section 7 (refusal and `notALongRest` `success`, `NotApplicable` `failure` with error status, no log); failing roller in exposure and progress: span error once, error counter once (`dice_roller_error`), `operations` once (result `error`), duration, no log; an `ErrInvalidInput` error is returned unchanged, span records it, counters not; closed attribute sets; `Ended` increments `contagion.ended`; no logger call on any path. |
| FR-109, FR-112, FR-115, FR-116 | unit, application | Infection with `OngoingEffect` of category `magicalContagion`; non-magical source returns an `ErrInvalidInput` error (`magical-effects.md` FR-008); `Infected` carries the built effect, lapsed effect leaves no state; `Ended` returns the id; T11 for every save-taking use case; nothing rolled when rejected; `DiceRequired` empty for refused, resisted and saved results and names only the onset `1d4` or laughter `1d10`, roller request log equal to that list; failing roller and invalid result return one wrapped `*srderr.PortError`, infection unchanged. |
| FR-117 to FR-122, (architecture) | unit, adapter, architecture, doc review | FR-038 constants not read from an entry; each use case records only its own span and counters; T13: bounds, `ContagionExampleFromID`, drift against T2 via the content decoder, each JSON problem kind without panicking, a `Fuzz` test over the contagion JSON decoder, a `ContagionDefinition` from every accepted entry; row R3-contagions exists and claims no `schemaVersion`; no use case file imports a use case or `d20.Resolver`; exported-name scan finds no duplicate, generic or "not used" name; signal names exist in `doc/observability/signals.md`. |

Not covered: none (FR-057, FR-059 are in 4.14).

## 9. Open questions and decisions

Settled, not reopened: Q-1, Q-12, Q-20 (shared-types-and-ownership.md FR-045, FR-012); Q-2 (hazard model reused); Q-3 (shared-types-and-ownership.md
FR-045, `content-schema-and-decoder.md` FR-129); Q-7 (time is reported); Q-10 (toolbox-hazard-effects.md FR-041); Q-18
(`environment-hazards-and-objects.md` FR-060, FR-115); Q-19 (FR-082, FR-083); Q-25 (FR-112).

Settled, one line each (decided from the SRD text; web research found no official clarification, so these are interpretations of pp. 193-194 and
are marked so):
- **Q-4** (interpretation): a "1" is the d20 used (`Roll.Natural()`, `d20-tests.md` FR-011) for every D20 Test; Disadvantage uses the lower die (p. 181), Advantage the higher (pp. 7-8). p. 193 says "a D20 Test", not only attack rolls. FR-014.
- **Q-5** (interpretation): the save exists only while the entity has control and not on the seizing turn ("later turns", p. 193). FR-018, FR-019.
- **Q-6**: regaining control does not end the possession (p. 193 lists only the Long Rest save, Dispel Evil and Good, curse removal); a second targeting is rejected. FR-009, FR-028.
- **Q-8** (interpretation): "an infected creature" (p. 194) includes one still incubating; reinfection is refused. FR-045, FR-069, FR-124.
- **Q-9** (interpretation): the minute runs from the failed save; a further failure rolls the damage again and replaces the stored state, restarting the minute. FR-077.
- **Q-11** (interpretation): only `fightingTheContagion` saves (Cackle Fever Long Rest, Sewer Plague dawn, the "Fighting the Contagion" entries) get the 24-hour Advantage; exposure and laughter saves do not. FR-036.
- **Q-13**: the contagion ends when the third 24-hour period elapses (FR-091); a second dose on a suppressed creature is rejected (FR-089); "Magic such as" is `ContagionGmRuledCuringMagic` (FR-094).
- **Q-14** (interpretation): per spent die, half the normal amount (already floored at 1, p. 187), rounded down (p. 187), so it can be 0. FR-082, T6.
- **Q-15**: after an interruption the host starts a new period; each completed period gives one save. FR-034.
- **Q-16**: a dawn success with no Exhaustion does not end the contagion; only a reduction to 0 does (p. 194). FR-081.
- **Q-17**: Restlessness withholds only `AllLostHitPointsRegained` and `ExhaustionLevelReduced`; regained Hit Point Dice stay. FR-083.
- **Q-21**: Protection from Evil and Good is a host-passed fact preventing possession without a save (p. 186); already possessed is rejected first. FR-108, FR-009.
- **Q-22**: skin contact infects, and an infected creature suffers the contagion's Blinded (p. 194 "also become infected"). FR-053.
- **Q-23**: the crafter needs the Herbalism Kit as well as proficiency ("can use it", p. 194). FR-114.
- **Q-24**: laughter ends with the contagion. FR-110.
- **Q-26**: persisted states go through the validating constructors (`NewX`, with `MustX` for trusted data); no separate non-failing variant until a consumer needs one (package design, `doc/rules/architecture.md` §6).

## 10. Changelog impact

`CHANGELOG.md`, `Unreleased`, **Added**: the `toolbox` curses and contagions slice (potency benchmark, Demonic Possession, Rest and Recuperation,
Cackle Fever, Sewer Plague, Sight Rot) on the shared save-gated effect model, five use cases with telemetry (shared `srd5e.toolbox.*` signals plus
`srd5e.toolbox.contagion.ended`) and the `contagion` catalogue entries (`ContagionDefinition`, `ContagionID`, bundled library, row R3-contagions). Not
breaking: the API is new. The 4.0a amendments are already accepted in the hazard spec.
