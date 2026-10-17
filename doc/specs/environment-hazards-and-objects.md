# Environment hazards and objects

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `observability-ports.md`, `d20-tests.md`, `ability-scores.md`, `content-schema-and-decoder.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md` |
| Consumed by | Hosts (turn-start, turn-end and day-end calls, results passed to `ApplyDamage` and the condition operations); in `toolbox`: `toolbox-traps.md`, `toolbox-environmental-effects.md`, `toolbox-curses-and-contagions.md` (`ContagionLevels`, FR-115). Hand-off by fact only, no import: `movement-lifting-and-flight.md`, `mounted-and-underwater-combat.md`, `initiative-and-actions.md`. `rests-and-hit-dice.md` and `exploration.md` are upstream `character` specs and cannot depend on this one (FR-116). |

Roadmap: `../roadmap.md`; types ship in phase 4 (`shared-types-and-ownership.md` FR-014). Rank 23 (table 4.11 there).

## 1. Purpose and scope

The glossary **Hazards** (Hazard, p. 182: "an environmental danger") and the **Objects** rules, both excluded by `hit-points-and-damage.md` (hazard damage; Q-11) and `conditions.md` (hazards that cause Exhaustion).

- **Burning** (p. 178): 1d4 Fire damage at the start of each of the creature's turns; the ways the fire goes out.
- **Falling** (p. 182): 1d6 Bludgeoning per 10 feet, at most 20d6; landing Prone; the Reaction check that halves the damage into a liquid.
- **Dehydration** (p. 181) and **Malnutrition** (p. 185): daily Water and Food needs by size, the Exhaustion each can cause, and the rule that this Exhaustion cannot be removed until the full daily amount is taken.
- **Suffocation** (p. 189): breath holding, Exhaustion each turn while out of breath or choking, removal when it can breathe again.
- **Objects** (pp. 12, 177-178, 180, 185, 188): Object Armor Class and Hit Points, Immunity to Poison and Psychic, GM rulings on damage types, Damage Threshold, destroyed at 0 Hit Points, Huge and Gargantuan objects split into sections, composite things (p. 12), automatic breaking (p. 12 action, p. 177 GM option), no ability scores.

State in, new state and events out (`doc/rules/architecture.md` §4). The host decides **when** things happen; the engine reads no clock, and elapsed time and distances arrive as values in commands.

**How damage and Exhaustion are handed on.** Damage hazards (Burning, Falling) **roll and return a `Damage`** (and, for a fall into liquid, a `HalfDamage` adjustment); the host passes it to `ApplyDamage` (a creature) or `ApplyDamageToObject` (an object). Exhaustion hazards call the domain operations `ConditionSet.Apply` and `ConditionSet.ReduceExhaustion`, which `ApplyConditionToSet` and `ReduceExhaustionOnSet` wrap (`conditions.md` FR-116, FR-117), and return the set and events unchanged. A use case never calls another use case (`shared-types-and-ownership.md` FR-044; D-1).

**Uses** (header lists tokens only, `shared-types-and-ownership.md` FR-056): `shared-types-and-ownership.md` FR-001, FR-002, FR-005, FR-046, FR-012 to FR-015, FR-017, FR-019, FR-021, FR-041, FR-044, FR-045, FR-048; `combat-foundations.md` (`CombatantID`, `Event`, `CombatTelemetry`, `CombatSignals`, `CombatResultClass`, FR-015 to FR-038); `hit-points-and-damage.md` (`Damage`, `HalfDamage`, `DamageAffinities`, `HitPoints`, `DamageResolution`; FR-016 to FR-025, FR-099); `conditions.md`; `ability-scores.md` (`Ability`, `AbilityScore`, `AbilityModifier`); `d20-tests.md` (published language only); `content-schema-and-decoder.md` (`DamageType`, `CreatureSize`, `Skill`, FR-106 to FR-109). Magic Item Resilience (`content-magic-items.md`, `magic-item-rules.md`, FR-123) is referenced only, never imported.

**Supersedes** (`shared-types-and-ownership.md` FR-048; no older requirement is withdrawn; owners may delete these deferrals):

- `hit-points-and-damage.md` section 1 (objects, Object Hit Points, Object Armor Class, Damage Threshold, hazard damage) and Q-11: object threshold is FR-079 to FR-085, creature half FR-112, hazard damage FR-005 to FR-026.
- `rests-and-hit-dice.md` Q-9: the lock is `HazardRecord` with `RemovableLevels` and `RemoveExhaustionLevels` (FR-060 to FR-065, FR-115).
- `toolbox-curses-and-contagions.md` HB-1 and Q-18: accepted as FR-115. `rules-glossary-coverage.md` Q-15 (objects half of Occupied Space): Q-23, not modelled.

**Out of scope:**

- **Long Rest** (p. 185) and its removal of 1 Exhaustion level: `rests-and-hit-dice.md` reports the level and the host passes it to `RemoveExhaustionLevels` instead of `ReduceExhaustionOnSet` (FR-116). **Death from Exhaustion 6**: `conditions.md` `IsDeadFromExhaustion` and `hit-points-and-damage.md` `RecordExhaustionDeath` (FR-095 there); a hazard at level 6 reports `exhaustionAtMaximum` (`conditions.md` FR-039) and 5 to 6 reports `exhaustionGained` (FR-111).
- **Wiring a creature's Damage Threshold into `ApplyDamage`** (p. 180): `DamageThresholdGate` (FR-112) is pure and serves a creature too, but `hit-points-and-damage.md` is left of this spec and cannot import it; the host evaluates the gate and skips `ApplyDamage` for `notMet` (Q-20).
- **Utilize action** (p. 191), Time-Limited Object Interactions (p. 12), spending the action, Reaction or movement, and **Flying, Hover, Climbing, Swimming, Jumping, Difficult Terrain** (pp. 178-182, 184-185, 189): `initiative-and-actions.md`.
- **Resolving the D20 Tests** (landing check, Constitution save): `d20-tests.md` and `resolve-d20-test-use-case.md`; hazard use cases take the finished `d20.Outcome` (FR-118). **Deciding that a creature falls** (flying fall, `FellOffMount`, thin-ice `fallenThrough`): the host builds `FallDistance` and `FallSurface`.
- **Healing or repairing objects** (Hit Points, p. 183; Ship Repair row, `equipment-rules.md`, p. 101), **Occupied and Unoccupied Space** by objects (pp. 185, 191; Q-23), spells that affect objects, magic items (content): not modelled. **Gameplay Toolbox** (pp. 192-203): the `toolbox` context (`shared-types-and-ownership.md` FR-045). Anything not in SRD 5.2.1.

## 2. Source

PDF page numbers equal printed page numbers.

- `SRD 5.2.1, "Hazard", p. 182`; "Burning [Hazard]", p. 178; "Falling [Hazard]", p. 182; "Dehydration [Hazard]", "Water Needs per Day", p. 181; "Malnutrition [Hazard]", "Food Needs per Day", p. 185; "Suffocation [Hazard]", p. 189; "Exhaustion [Condition]", p. 181; "Prone [Condition]", p. 186; "Reaction", p. 186
- `SRD 5.2.1, "Object", p. 185`; "What Is an Object?" and "Breaking Objects", p. 12 ("a building or a vehicle, which are composed of many objects"; "As an action, you can automatically break or otherwise destroy a fragile, nonmagical object"); "Breaking Objects", p. 177 (harm; automatic breaking); "Object Armor Class", "Object Hit Points", "Damage Types and Objects", "Damage Threshold", "No Ability Scores", p. 178; "Damage Threshold", p. 180; "Size", p. 188; "Creature Size", p. 14; "Magic Item Resilience", p. 206 (FR-123)
- `SRD 5.2.1, "Hit Points", p. 183`; "Immunity", p. 183; "Temporary Hit Points", p. 18; "Saving Throw", p. 187; "Round Down", p. 187; "Magical Contagions", "Cackle Fever", p. 194 ("lasts until the contagion ends on the creature"; FR-115). Out of scope: "Utilize [Action]", p. 191, "Long Rest", p. 185, "Occupied Space", p. 185, "Unoccupied Space", p. 191.
- Package rules: `doc/rules/architecture.md` (§1-§4, §6), `doc/rules/observability.md` (§2-§7), `doc/rules/testing.md` (§3-§5), `shared-types-and-ownership.md`.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Hazard | `HazardKind` (enum) | The five (pp. 12, 182); closed source of the `srd5e.combat.hazard.kind` attribute (FR-103) via `name`. |
| Burning | `BurningState` (`notBurning`, `burning`), `BurningRejection` (`notBurning`, `extinguishBlockedByImmunity`) | A hazard, not a condition (`conditions.md` section 1). Lower-case enum values so no exported type is `NotBurning` or `Burning` (FR-113). |
| "DC 15 Strength (Athletics) or Dexterity (Acrobatics) check" | `FallLandingCheck`, `LiquidLanding` (`NoLandingReaction`, `LandingCheckMade(d20.Outcome)`) | Resolved in `d20-tests.md`. |
| Water / Food Needs per Day | `WaterNeedsPerDay`, `WaterAmount` / `FoodNeedsPerDay`, `FoodAmount` | Eighths of a gallon / pound (Q-18). Rules tables as unexported package-level values in package `combat` (files `hazard_*.go`; Q-1). |
| holds breath, out of breath, choking | `BreathState` (`BreathNormal`, `BreathHolding(elapsed, capacity)`, `BreathSuffocating(cause)`), `SuffocationCause` (`ranOutOfBreath`, `choking`), `BreathCapacity` (`Duration`, FR-046), `BreathRejection` (`notBreathing`, `notHoldingBreath`, `alreadySuffocating`) | `Breath` prefix avoids bare `Breathing`/`Suffocating` (`combat-foundations.md` FR-024). |
| Object | `ObjectStatus` | "A nonliving, distinct thing" (p. 185). |
| Object Armor Class | `ObjectSubstance`, `ObjectArmorClassTable` | Seven substances (p. 178); constants here (Q-1). |
| Object Hit Points | `ObjectDurability` (Fragile, Resilient), `ObjectHitPointsEntry`, `ObjectHitPointsTable` | Tiny to Large rows (p. 178); unexported tables in package `combat` (files `object_*.go`). |
| Damage Threshold | `DamageThreshold`, `DamageThresholdGate`, `ThresholdOutcome` (`notApplicable`, `met`, `notMet`) | Creature or object (p. 180); gate is not object-specific (FR-112). |
| composite things | `CompositeObject`, `CollapseRuling` (`DoesNotCollapse`, `CollapsesWhole`) | GM decides collapse (p. 178); building or vehicle is a `CompositeObject` (FR-114). |

**No SRD source** (descriptive package names, no rules added): `HazardRecord`, `HazardRecordCreation`, `HazardRecordRejection`, `RemoveExhaustionLevels`, `ExhaustionRemovalRejection`, `ObjectDamageRejection`, `BurningState`, `BurningRejection`, `BreathRejection`, `ExtinguishMethod`, `FallSurface` (`FallOntoGround`, `FallIntoLiquid(landing)`), `LiquidLanding`, `FallLandingCheck`, `WaterAmount`, `FoodAmount`, `FoodSavingThrow` (`FoodSaveNotRolled`, `FoodSaveRolled(d20.Outcome)`), `BreathState`, `BreathChange` (`BeginHoldingBreath(constitutionModifier)`, `HoldBreathFor(elapsed)`, `BeginChoking`), `BreathCapacity`, `SuffocationCause`, `HazardExhaustionOutcome` (D-2), `ProneApplicationOutcome`, `ExhaustionReductionOutcome` (FR-120), `ObjectStatus`, `ObjectSubstance`, `ObjectDurability`, `ObjectHitPointsEntry`, `ObjectHitPointsLookup` (`ObjectHitPointsFound`, `ObjectHitPointsSplitRequired`), `AutomaticBreakBasis`, `ObjectAffinities`, `ObjectAbilityScores` (`NoAbilityScores`, `AssignedAbilityScores`), `CollapseRuling`, `CompositeStatus`, `ObjectHarm` (`ObjectDamageInstance(damages, adjustments)`, `ObjectBrokenAutomatically(basis)`), `ThresholdOutcome`, `DamageThresholdGate`, `EnvironmentEvent`. Used by reference: `DamageType`, `CreatureSize`, `Skill` (`content-schema-and-decoder.md`); `Damage`, `DamageAffinities`, `HitPoints` (`hit-points-and-damage.md`); `ConditionSet`, `ConditionPermission`, `SavingThrowEffect` (`conditions.md`); `Ability`, `AbilityScore`, `AbilityModifier` (`ability-scores.md`); `DifficultyClass`, `ArmorClass`, `d20.Outcome` (`d20-tests.md`); `dice.Expression`, `RollResult` (kernel).

## 4. Requirements

Stable ids; never renumber. No requirement is withdrawn and none is provisional: every question in section 9 is settled. Case tables list the fixtures: the pages are in each heading, and every row is unhappy path (boundary, rule interaction, state or invalid input) unless marked *happy*.

### 4.1 The hazard set and the hazard record

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `HazardKind` MUST have exactly five values: burning, dehydration, falling, malnutrition and suffocation, in that (alphabetical) order. | SRD 5.2.1, "Hazard", p. 182 | happy |
| FR-002 | `HazardRecord(dehydrationLevels, malnutritionLevels, suffocationLevels, daysWithoutFood, contagionLevels)` MUST be constructible with each value 0 or more (`ContagionLevels` defaults to 0, FR-115) and the four level counts summing to at most 6, and MUST expose the five values. `HazardRecord.Empty` MUST be all zeros. A level count is the number of Exhaustion levels currently attributed to that source. | SRD 5.2.1, "Exhaustion", p. 181 (levels 1 to 6); package design | happy |
| FR-003 | `NewHazardRecord` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a negative level count, a negative `daysWithoutFood`, or four level counts summing to more than 6 (FR-122 gives the typed error). A use case clamps instead (FR-110); a host restoring a persisted ledger uses `NewHazardRecord` too. | `doc/rules/architecture.md` §3, §6; SRD 5.2.1, "Exhaustion", p. 181 | unhappy (invalid input) |
| FR-004 | Two `HazardRecord`s with the same five values MUST be equal under `==` (the type is comparable); changing any one value MUST make them unequal. | `doc/rules/architecture.md` §3 (value object) | happy |

### 4.2 Burning

`RollBurningDamage(state)` rolls through the `dice.Roller` and returns a `Damage`; `ExtinguishBurning` ends the fire.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-005 | For `BurningState.Burning`, `RollBurningDamage` MUST request exactly one `1d4` roll and return a `Damage` whose amount is the total and type is Fire, with the `RollResult`. Fixed rolls 1, 2, 3, 4 give 1, 2, 3, 4 Fire damage. | SRD 5.2.1, "Burning [Hazard]", p. 178 | happy |
| FR-006 | `RollBurningDamage` on `BurningState.NotBurning` MUST return a rejection (`BurningRejection.NotBurning`), request no roll, and return no `Damage`. | `doc/rules/testing.md` §4 (state errors); SRD 5.2.1, "Burning [Hazard]", p. 178 | unhappy (state error) |
| FR-007 | A successful `RollBurningDamage` MUST return `BurningState.Burning` unchanged: the fire does not go out by itself, and calling it again (the next turn) rolls again. | SRD 5.2.1, "Burning [Hazard]", p. 178 ("at the start of each of its turns") | unhappy (state error) |
| FR-008 | `RollBurningDamage` MUST NOT apply Resistance, Vulnerability, Immunity, Temporary Hit Points or the Hit Points; the `Damage` is full Fire damage and `ApplyDamage` applies Immunity later (`hit-points-and-damage.md` FR-016 to FR-024). | `hit-points-and-damage.md` section 1; D-1 | unhappy (rule interaction) |
| FR-009 | `ExtinguishBurning` with `RollOnTheGround` on a burning creature MUST apply Prone through `ConditionSet.Apply`, return `BurningState.NotBurning`, carry the condition set and events from that call unchanged, hold the call's result in `proneApplication` as `ProneApplied(application)`, and emit `BurningEnded(rollOnTheGround)` (FR-106). | SRD 5.2.1, "Burning [Hazard]", p. 178; "Prone [Condition]", p. 186 | happy |
| FR-010 | `RollOnTheGround` on a creature that already has Prone (explicit, or implied by Unconscious) MUST still end the fire, return `ProneApplied(ConditionAlreadyHeld)`, produce no `ConditionApplied` event, and emit `BurningEnded(rollOnTheGround)`. | SRD 5.2.1, "Burning [Hazard]", p. 178; `conditions.md` FR-027, FR-125 | unhappy (rule interaction) |
| FR-011 | `RollOnTheGround` on a creature with Immunity to Prone (the application would be `ConditionImmune`) MUST leave the state `burning`, the condition set unchanged, and return `BurningRejection.ExtinguishBlockedByImmunity`.. | SRD 5.2.1, "Burning [Hazard]", p. 178 ("by giving yourself the Prone condition"); "Immunity", p. 183 | unhappy (rule interaction) |
| FR-012 | `doused`, `submerged` and `suffocated` MUST each end the fire, leave the condition set unchanged with no condition events, return `ProneNotApplied` (no `ConditionSet.Apply` call), and record the method in `BurningEnded` (FR-106). | SRD 5.2.1, "Burning [Hazard]", p. 178 ("doused, submerged, or suffocated") | happy |
| FR-013 | `ExtinguishBurning` on `BurningState.NotBurning` MUST return `BurningRejection.NotBurning`, whatever the method, and MUST NOT apply Prone. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |

**4.2a Extinguishing** (FR-009 to FR-013, FR-106; p. 178, 183, 191). One call per row; events in order.

| Label | Method | Conditions before | Result | After | Req. |
|---|---|---|---|---|---|
| roll on the ground (*happy*), state burning | rollOnTheGround | none | notBurning, `ConditionAdded`; `ConditionApplied`, `BurningEnded(rollOnTheGround)` | Prone | FR-009 |
| already Prone; Unconscious (implies Prone) | rollOnTheGround | Prone; Unconscious | notBurning, `ConditionAlreadyHeld`; `BurningEnded` only | unchanged | FR-010 |
| Immune to Prone | rollOnTheGround | immune to Prone | burning, `extinguishBlockedByImmunity` | unchanged | FR-011 |
| doused, submerged, suffocated (*happy*, one row each) | that method | none | notBurning, `BurningEnded(method)` | none | FR-012 |
| not burning (state notBurning) | rollOnTheGround or doused | none | `BurningRejection.NotBurning`, no Prone | none | FR-013 |

### 4.3 Falling

One die per full 10 feet fallen ("Round Down", p. 187), at most 20.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `FallDistance` MUST accept 0 feet or more and returns an `srderr.ErrInvalidInput` error for a negative distance. | `doc/rules/testing.md` §4; SRD 5.2.1, "Falling [Hazard]", p. 182 | unhappy (invalid input) |
| FR-015 | The number of d6 MUST be `min(20, floor(feet / 10))` (table 4.3a).. | SRD 5.2.1, "Falling [Hazard]", p. 182; "Round Down", p. 187 | unhappy (boundary) |
| FR-016 | For a fall of 10 feet or more, `ResolveFall` MUST request exactly one roll of that many d6 and return a Bludgeoning `Damage` of the total, with the `RollResult` and the dice count. | SRD 5.2.1, "Falling [Hazard]", p. 182 | happy |
| FR-017 | For a fall of less than 10 feet, `ResolveFall` MUST return `FallWithoutDamage`, request no roll, return no `Damage`.. | SRD 5.2.1, "Falling [Hazard]", p. 182; "Round Down", p. 187 | unhappy (boundary) |
| FR-018 | A fall onto `FallOntoGround` MUST return no damage adjustments, whatever else is supplied. | SRD 5.2.1, "Falling [Hazard]", p. 182 (halving is for liquid only) | happy |
| FR-019 | A fall into a liquid with `LandingCheckMade(success)` MUST return the `HalfDamage` adjustment with the `Damage`, so `ApplyDamage` halves it (rounded down, before Resistance and Vulnerability); the `Damage` stays the full rolled amount. | SRD 5.2.1, "Falling [Hazard]", p. 182; "Round Down", p. 187; `hit-points-and-damage.md` FR-014, FR-099 | happy |
| FR-020 | A fall into a liquid with `LandingCheckMade(failure)` or `NoLandingReaction` MUST return no adjustments. | SRD 5.2.1, "Falling [Hazard]", p. 182 | unhappy (rule interaction) |
| FR-021 | `FallLandingCheck` MUST publish DC 15 and the two skill choices Strength (Athletics) and Dexterity (Acrobatics); it does not resolve the check. | SRD 5.2.1, "Falling [Hazard]", p. 182 | happy |
| FR-022 | A fall of less than 10 feet into a liquid MUST return `FallWithoutDamage` whatever the `LiquidLanding`; a landing check supplied for a harmless fall MUST be ignored. | SRD 5.2.1, "Falling [Hazard]", p. 182 | unhappy (rule interaction) |
| FR-023 | `LandAfterFall` with a damage amount of 1 or more MUST apply Prone through `ConditionSet.Apply` and return `FallLandedProne` with the application and condition events. | SRD 5.2.1, "Falling [Hazard]", p. 182 ("has the Prone condition unless it avoids taking any damage") | happy |
| FR-024 | `LandAfterFall` with a damage amount of 0 MUST return `FallLandedStanding` with the set unchanged. The amount is the final amount after Resistance, Vulnerability and Immunity, so a fall reduced to 0 also avoids Prone.. | SRD 5.2.1, "Falling [Hazard]", p. 182; "Immunity", p. 183 | unhappy (boundary) |
| FR-025 | `LandAfterFall` on a creature already Prone or immune to it MUST carry the `ConditionAlreadyHeld` or `ConditionImmune` application unchanged in `FallLandedProne` and produce no new event. | `conditions.md` FR-018, FR-021, FR-125 | unhappy (rule interaction) |
| FR-026 | `LandAfterFall` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a negative damage amount. | `hit-points-and-damage.md` FR-094; `doc/rules/architecture.md` §6 | unhappy (invalid input) |

**4.3a Fall dice** (FR-015 to FR-017; pp. 182, 187). Dice = number of d6.

| Distance (ft) | 0 | 9 | 10 | 19 | 20 | 199 | 200 | 201 | 1000 |
|---|---|---|---|---|---|---|---|---|---|
| Dice | none, `FallWithoutDamage` | none, `FallWithoutDamage` | 1 | 1 | 2 | 19 | 20 | 20 | 20 |

**4.3b Landing in liquid** (FR-018 to FR-022; pp. 182, 187), fixed roll 3d6 = 11.

| Label | Distance | Surface | Landing | Adjustments | After `ApplyDamage` (no affinities) | Req. |
|---|---|---|---|---|---|---|
| ground (*happy*) | 30 | ground | n/a | none | 11 | FR-018 |
| liquid, check succeeds (*happy*) | 30 | liquid | success | `HalfDamage` | 5 (11 halved, rounded down) | FR-019 |
| liquid, check fails; no Reaction | 30 | liquid | failure; no reaction | none | 11 | FR-020 |
| liquid, harmless fall | 9 | liquid | success | `FallWithoutDamage` | no damage | FR-022 |
| liquid, 20d6 maximum | 500 | liquid | success | `HalfDamage` | half of the 20d6 total | FR-019 |

**4.3c Landing after a fall** (FR-023 to FR-026; p. 182). One `LandAfterFall` call per column; "Damage" is the final amount.

| Damage | 11 (*happy*) | 1 | 0 | 0 (Immunity reduced it) | 11, already Prone | 11, immune to Prone | -1 |
|---|---|---|---|---|---|---|---|
| Result | `FallLandedProne`, Prone added | `FallLandedProne` | `FallLandedStanding`, set unchanged | `FallLandedStanding` | `FallLandedProne(ConditionAlreadyHeld)`, no event | `FallLandedProne(ConditionImmune)`, no event | an `srderr.ErrInvalidInput` error |
| Req. | FR-023 | FR-023 | FR-024 | FR-024 | FR-025 | FR-025 | FR-026 |

### 4.4 Daily needs

Amounts are whole eighths of a gallon (water) or pound (food), so half of the Tiny requirement (1/8) is exact (Q-18).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-027 | `WaterNeedsPerDay.ForSize` MUST return the amount in table 4.4a for each of the six sizes. | SRD 5.2.1, "Dehydration [Hazard]", Water Needs per Day, p. 181 | happy |
| FR-028 | `FoodNeedsPerDay.ForSize` MUST return the amount in table 4.4a for each of the six sizes. | SRD 5.2.1, "Malnutrition [Hazard]", Food Needs per Day, p. 185 | happy |
| FR-029 | `WaterAmount` and `FoodAmount` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a negative amount, be distinct types (a `WaterAmount` of 8 is not equal to a `FoodAmount` of 8, and neither is accepted where the other is expected), and be comparable with `==`. | `doc/rules/architecture.md` §3 (stringly-typed anti-pattern); `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-030 | `isLessThanHalfOf(required)` MUST be true exactly when `2 * amount < required`, integer arithmetic only (table 4.4b). | SRD 5.2.1, "Dehydration [Hazard]", p. 181 ("less than half"); "Malnutrition [Hazard]", p. 185 | unhappy (boundary) |

**4.4a Needs per day** (FR-027, FR-028; pp. 181, 185; same for water in gallons and food in pounds; *happy*).

| Size | Tiny | Small | Medium | Large | Huge | Gargantuan |
|---|---|---|---|---|---|---|
| Printed amount | 1/4 | 1 | 1 | 4 | 16 | 64 |
| Eighths | 2 | 8 | 8 | 32 | 128 | 512 |

**4.4b Half of a requirement** (FR-030; p. 181). "Less than half" is `2 * amount < required`.

| Required (eighths) | 8 | 8 | 8 | 8 | 2 | 2 | 512 | 512 |
|---|---|---|---|---|---|---|---|---|
| Amount | 0 | 3 | 4 | 5 | 0 | 1 | 255 | 256 |
| Less than half | yes | yes | no | no | yes | no | yes | no |

### 4.5 Dehydration

`ResolveDehydrationDayEnd` is called at the end of each day with the day's total water drunk. The result carries a `HazardExhaustionOutcome` (`noExhaustion`, `exhaustionGained`, `exhaustionAtMaximum`, `immune`; D-2): a projection of the Exhaustion `ApplyConditionToSetResult` (`ConditionAdded` is `exhaustionGained`, `ExhaustionAtMaximum` is `exhaustionAtMaximum`, `ConditionImmune` is `immune`; `ConditionAlreadyHeld` cannot occur, `conditions.md` FR-016, FR-039, FR-125) plus `noExhaustion` for a day on which none is applied.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | When the water drunk is less than half the Water Needs per Day (FR-030), `ResolveDehydrationDayEnd` MUST apply Exhaustion once through `ConditionSet.Apply` and, when added, increase `dehydrationLevels` by 1. | SRD 5.2.1, "Dehydration [Hazard]", p. 181 | happy |
| FR-032 | When the water drunk is half the requirement or more, it MUST apply no Exhaustion and leave the set and record unchanged (exactly half is not less than half). | SRD 5.2.1, "Dehydration [Hazard]", p. 181 | unhappy (boundary) |
| FR-033 | When the water drunk is the full requirement or more, it MUST set `dehydrationLevels` to 0 (levels no longer locked, FR-060) and emit `DehydrationRecovered` with the levels unlocked when that is above 0. It MUST NOT remove any Exhaustion level itself. | SRD 5.2.1, "Dehydration [Hazard]", p. 181 ("can't be removed until the creature drinks the full amount") | unhappy (rule interaction) |
| FR-034 | One call MUST gain at most one Exhaustion level, however little was drunk. | SRD 5.2.1, "Dehydration [Hazard]", p. 181 ("gains 1 Exhaustion level at the day's end") | unhappy (boundary) |
| FR-035 | When the application is not `ConditionAdded` (`ExhaustionAtMaximum` or `ConditionImmune`), the record MUST be unchanged and the result MUST say which (`exhaustionAtMaximum` or `immune`). | `conditions.md` FR-039, FR-125; SRD 5.2.1, "Exhaustion", p. 181 | unhappy (rule interaction) |

**4.5a Dehydration at day's end** (FR-031 to FR-035, FR-110, FR-111; pp. 181, 183). One call per row; "record" is `dehydrationLevels`.

| Label | Size | Drunk (eighths) | Exhaustion before | Record before | Outcome | Exhaustion after | Record after | Req. |
|---|---|---|---|---|---|---|---|---|
| nothing drunk (*happy*) | Medium | 0 | none | 0 | `exhaustionGained` | 1 | 1 | FR-031 |
| one below half | Medium | 3 | none | 0 | `exhaustionGained` | 1 | 1 | FR-031 |
| exactly half | Medium | 4 | none | 0 | `noExhaustion` | none | 0 | FR-032 |
| just under full | Medium | 7 | 2 (other source) | 0 | `noExhaustion` | 2 | 0 | FR-032 |
| full amount unlocks | Medium | 8 | 2 | 2 | `noExhaustion`, `DehydrationRecovered(2)` | 2 (not removed) | 0 | FR-033 |
| full amount, nothing locked | Medium | 8 | none | 0 | `noExhaustion`, no event | none | 0 | FR-033 |
| second day without water | Medium | 0 | 1 | 1 | `exhaustionGained` | 2 | 2 | FR-034 |
| Tiny exactly half; Tiny nothing | Tiny | 1; 0 | none | 0 | `noExhaustion`; `exhaustionGained` | none; 1 | 0; 1 | FR-032, FR-031 |
| Gargantuan one below half | Gargantuan | 255 | none | 0 | `exhaustionGained` | 1 | 1 | FR-031 |
| level 5 to 6 | Medium | 0 | 5 | 2 | `exhaustionGained` (host then calls `IsDeadFromExhaustion`) | 6 | 3 | FR-031, FR-111 |
| record above the set | Medium | 0 | 2 | 4 | `exhaustionGained`, record clamped | 3 | 3 | FR-110 |
| at level 6 | Medium | 0 | 6 | 3 | `exhaustionAtMaximum` | 6 | 3 | FR-035 |
| immune to Exhaustion | Medium | 0 | none | 0 | `immune` | none | 0 | FR-035 |

### 4.6 Malnutrition

`ResolveMalnutritionDayEnd` is called at the end of each day with the day's total food eaten and a `FoodSavingThrow`. It reads the SRD literally (Q-9, settled): the Constitution saving throw is for a creature that **eats but** consumes less than half; a creature that eats **nothing** has no save and gains Exhaustion automatically at the end of its fifth day without food.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-036 | When the food eaten is half the requirement or more, `ResolveMalnutritionDayEnd` MUST apply no Exhaustion, require no saving throw, and set `daysWithoutFood` to 0. | SRD 5.2.1, "Malnutrition [Hazard]", p. 185 | happy |
| FR-037 | When the food eaten is the full requirement or more, it MUST also set `malnutritionLevels` to 0 and emit `MalnutritionRecovered` with the levels unlocked when above 0. It MUST NOT remove any Exhaustion level itself. | SRD 5.2.1, "Malnutrition [Hazard]", p. 185 ("can't be removed until the creature eats the full amount") | unhappy (rule interaction) |
| FR-038 | When some food but less than half was eaten and the saving throw is `FoodSaveNotRolled`, it MUST return `MalnutritionSaveRequired` carrying DC 10 and Constitution, and change no state. | SRD 5.2.1, "Malnutrition [Hazard]", p. 185 ("a DC 10 Constitution saving throw") | happy |
| FR-039 | With `FoodSaveRolled(failure)` after eating less than half, it MUST apply Exhaustion once, increase `malnutritionLevels` by 1 when added, and set `daysWithoutFood` to 0. | SRD 5.2.1, "Malnutrition [Hazard]", p. 185 | happy |
| FR-040 | With `FoodSaveRolled(success)` after eating less than half, it MUST apply no Exhaustion and set `daysWithoutFood` to 0. | SRD 5.2.1, "Malnutrition [Hazard]", p. 185 | unhappy (rule interaction) |
| FR-041 | A `FoodSaveRolled` supplied when no save is required (half or more eaten, or nothing eaten) MUST be ignored: the result equals that for `FoodSaveNotRolled`. | `doc/rules/testing.md` §4; SRD 5.2.1, "Malnutrition [Hazard]", p. 185 | unhappy (invalid input) |
| FR-042 | When nothing was eaten, it MUST increase `daysWithoutFood` by 1; while the new count is 1 to 4 it MUST apply no Exhaustion and require no save. | SRD 5.2.1, "Malnutrition [Hazard]", p. 185 ("eats nothing for 5 days") | unhappy (boundary) |
| FR-043 | When nothing was eaten and the new `daysWithoutFood` is 5 or more, it MUST apply Exhaustion once without a save (1 level at the end of the fifth day, then 1 more each following day without food).. | SRD 5.2.1, "Malnutrition [Hazard]", p. 185 | unhappy (boundary) |
| FR-044 | Eating any food (even less than half) MUST end the run of days without food: `daysWithoutFood` becomes 0 and the next run starts from 1.. | SRD 5.2.1, "Malnutrition [Hazard]", p. 185 | unhappy (state error) |
| FR-045 | When the application is not `ConditionAdded` (`ExhaustionAtMaximum` or `ConditionImmune`), the level counts MUST be unchanged, `daysWithoutFood` MUST still follow FR-042 to FR-044, and the result MUST say which outcome occurred. | `conditions.md` FR-039, FR-125 | unhappy (rule interaction) |

**4.6a Malnutrition at day's end** (FR-036 to FR-045; pp. 181, 183, 185). Medium creature (requirement 8 eighths); "days" is `daysWithoutFood`.

| Label | Eaten | Days before | Save | Result | Exhaustion after | Days after | Req. |
|---|---|---|---|---|---|---|---|
| ate enough (*happy*) | 4 | 3 | none | `noExhaustion` | none | 0 | FR-036 |
| ate the full amount (*happy*) | 8 | 0 | none | `noExhaustion` | none | 0 | FR-036, FR-037 |
| full amount unlocks | 8 | 0 | none | `noExhaustion`, `MalnutritionRecovered(1)` (record 1 before) | unchanged | 0 | FR-037 |
| less than half, no save yet (*happy*) | 3 | 0 | not rolled | `MalnutritionSaveRequired(DC 10, Constitution)`, state unchanged | unchanged | 0 | FR-038 |
| less than half, save fails (*happy*) | 3 | 0 | failure | `exhaustionGained`, record +1 | 1 | 0 | FR-039 |
| less than half, save succeeds | 3 | 0 | success | `noExhaustion` | none | 0 | FR-040 |
| one below half, save fails | 3 | 2 | failure | `exhaustionGained` | 1 | 0 | FR-039, FR-044 |
| exactly half, save ignored | 4 | 0 | failure | `noExhaustion` | none | 0 | FR-041 |
| nothing eaten, save ignored | 0 | 0 | failure | `noExhaustion` | none | 1 | FR-041, FR-042 |
| nothing eaten, day 1; day 4 | 0 | 0; 3 | none | `noExhaustion` | none | 1; 4 | FR-042 |
| nothing eaten, day 5 | 0 | 4 | none | `exhaustionGained` | 1 | 5 | FR-043 |
| nothing eaten, day 6; day 7 | 0 | 5; 6 | none | `exhaustionGained` | 2 (from 1); 3 (from 2) | 6; 7 | FR-043 |
| a little food resets the run | 1 | 4 | success | `noExhaustion` | none | 0 | FR-044 |
| restart after eating | 0 | 0 (after a meal) | none | `noExhaustion` | none | 1 | FR-044 |
| day 5 at level 6 | 0 | 4 | none | `exhaustionAtMaximum` | 6 | 5 | FR-043, FR-045 |
| day 5, immune to Exhaustion | 0 | 4 | none | `immune` | none | 5 | FR-045 |

### 4.7 Suffocation

`BreathCapacity` and `elapsed` are `Duration`s; tables use whole seconds, so no unit conversion is left to the host. Every breath use case takes the `CombatantID` `target` (FR-105). The host reports breathing changes through `AdvanceBreath`, calls `ResolveSuffocationTurnEnd` at the end of each of the creature's turns and `RestoreBreathing` when it can breathe again.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-046 | `BreathCapacity.ForConstitution(modifier)` MUST return a `Duration`: the longer of 30 seconds and `(1 + modifier)` minutes (table 4.7a). | SRD 5.2.1, "Suffocation [Hazard]", p. 189 | unhappy (boundary) |
| FR-047 | `AdvanceBreath` with `BeginHoldingBreath(constitutionModifier)` on `BreathNormal` MUST return `BreathHolding(Duration.Zero, capacity)` and the event `BreathHeldStarted(target)`. | SRD 5.2.1, "Suffocation [Hazard]", p. 189 | happy |
| FR-048 | `HoldBreathFor(elapsed)` on `BreathHolding` MUST add `elapsed` to the time held and, while the total is below capacity, keep `BreathHolding`. A zero `elapsed` MUST be a valid no-op. | SRD 5.2.1, "Suffocation [Hazard]", p. 189 | happy |
| FR-049 | When the time held reaches the capacity (equal or above), `HoldBreathFor` MUST return `BreathSuffocating(ranOutOfBreath)` and the event `RanOutOfBreath(target)` (table 4.7b).. | SRD 5.2.1, "Suffocation [Hazard]", p. 189 ("before suffocation begins") | unhappy (boundary) |
| FR-050 | `BeginChoking` on `BreathNormal` or `BreathHolding` MUST return `BreathSuffocating(choking)` without waiting for any capacity, and the event `BeganChoking(target)`. | SRD 5.2.1, "Suffocation [Hazard]", p. 189 ("runs out of breath or is choking") | happy |
| FR-051 | `AdvanceBreath` MUST return `BreathAdvanceRejected` holding a `BreathRejection`, state unchanged, no event: `notBreathing` for `BeginHoldingBreath` when not `BreathNormal`, `notHoldingBreath` for `HoldBreathFor` when not `BreathHolding`, `alreadySuffocating` for `BeginChoking` when already `BreathSuffocating`. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-052 | `HoldBreathFor` with a negative `elapsed` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. | `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-053 | `ResolveSuffocationTurnEnd` on a `BreathSuffocating` creature MUST apply Exhaustion once through `ConditionSet.Apply` and, when added, increase `suffocationLevels` by 1. | SRD 5.2.1, "Suffocation [Hazard]", p. 189 ("gains 1 Exhaustion level at the end of each of its turns") | happy |
| FR-054 | `ResolveSuffocationTurnEnd` on a `BreathNormal` or `BreathHolding` creature MUST apply no Exhaustion and return `noExhaustion`, everything unchanged. | SRD 5.2.1, "Suffocation [Hazard]", p. 189 | unhappy (state error) |
| FR-055 | When the application is not `ConditionAdded`, `suffocationLevels` MUST be unchanged and the result MUST say which outcome occurred. | `conditions.md` FR-039, FR-125 | unhappy (rule interaction) |
| FR-056 | `RestoreBreathing` on a `BreathSuffocating` creature MUST return `BreathNormal`, reduce Exhaustion by `suffocationLevels` through `ConditionSet.ReduceExhaustion` (only when `suffocationLevels` is 1 or more, so it never receives 0), set `suffocationLevels` to 0 ("removes all levels of Exhaustion it gained from suffocating"), hold the result in `reduction` as `ExhaustionReductionMade(result)`, and emit `BreathRestored(target, levelsRemoved)` (FR-107). With `suffocationLevels` 0, no call is made and `reduction` is `ExhaustionReductionNotAttempted`. | SRD 5.2.1, "Suffocation [Hazard]", p. 189 | happy |
| FR-057 | `RestoreBreathing` MUST remove only the levels in `suffocationLevels`; other Exhaustion, including `dehydrationLevels` and `malnutritionLevels`, MUST remain (table 4.7c). | SRD 5.2.1, "Suffocation [Hazard]", p. 189 ("gained from suffocating") | unhappy (rule interaction) |
| FR-058 | `RestoreBreathing` on a `BreathNormal` or `BreathHolding` creature MUST return `BreathNormal` and leave Exhaustion unchanged when `suffocationLevels` is 0, with no event and `ExhaustionReductionNotAttempted`; it MUST NOT call `reduceExhaustion` with 0 levels (`conditions.md` FR-045 rejects it). | `conditions.md` FR-045; SRD 5.2.1, "Suffocation [Hazard]", p. 189 | unhappy (empty and minimal) |
| FR-059 | When `suffocationLevels` exceeds the Exhaustion the set holds, `RestoreBreathing` MUST end the Exhaustion without a negative level, tolerate `ExhaustionReductionNotPresent` (held as `ExhaustionReductionMade(ExhaustionReductionNotPresent)`), and still set `suffocationLevels` to 0. | `conditions.md` FR-044, FR-046; `doc/rules/testing.md` §4 | unhappy (state error) |

**4.7a Breath capacity** (FR-046; p. 189).

| Constitution modifier | -5 | -2 | -1 | 0 | 1 | 3 | 10 |
|---|---|---|---|---|---|---|---|
| `1 + modifier` minutes | -4 | -1 | 0 | 1 | 2 | 4 | 11 |
| Capacity (s) | 30 | 30 | 30 | 60 | 120 | 240 | 660 |

**4.7b Holding breath over time** (FR-047 to FR-052; p. 189). Constitution modifier 0 (capacity 60 s); one `HoldBreathFor` per row.

| Label | Held before (s) | Elapsed (s) | Result | Req. |
|---|---|---|---|---|
| partway (*happy*) | 0 | 30 | `BreathHolding(30)` | FR-048 |
| zero elapsed | 30 | 0 | `BreathHolding(30)` | FR-048 |
| one second below capacity | 30 | 29 | `BreathHolding(59)` | FR-048 |
| exactly capacity | 30 | 30 | `BreathSuffocating(ranOutOfBreath)` | FR-049 |
| beyond capacity | 30 | 600 | `BreathSuffocating(ranOutOfBreath)` | FR-049 |
| negative elapsed | 30 | -1 | an `srderr.ErrInvalidInput` error | FR-052 |
| not holding | BreathNormal | 10 | `notHoldingBreath` | FR-051 |
| already suffocating | BreathSuffocating | 10 | `notHoldingBreath` | FR-051 |

**4.7c Restoring breath** (FR-056 to FR-059; pp. 181, 185, 189). Levels are Exhaustion before and after; Susp. = `suffocationLevels`, Dehyd. = `dehydrationLevels`, Malnut. = `malnutritionLevels`.

| Label | State | Level | Susp. | Dehyd. | Malnut. | Level after | Susp. after | Req. |
|---|---|---|---|---|---|---|---|---|
| all suffocation levels (*happy*) | BreathSuffocating | 2 | 2 | 0 | 0 | none | 0 | FR-056 |
| some from suffocation | BreathSuffocating | 3 | 2 | 0 | 0 | 1 | 0 | FR-057 |
| dehydration level kept | BreathSuffocating | 3 | 1 | 2 | 0 | 2 | 0 | FR-057 |
| malnutrition level kept | BreathSuffocating | 2 | 1 | 0 | 1 | 1 | 0 | FR-057 |
| holding or breathing, nothing gained | BreathHolding (level 1, Dehyd. 1); BreathNormal (none) | 1; none | 0 | 1; 0 | 0 | 1; none | 0 | FR-058 |
| record exceeds the level | BreathSuffocating | 1 | 3 | 0 | 0 | none | 0 | FR-059 |
| record but no Exhaustion | BreathSuffocating | none | 2 | 0 | 0 | none (`ExhaustionReductionNotPresent` tolerated) | 0 | FR-059 |
| from level 6 | BreathSuffocating | 6 | 6 | 0 | 0 | none | 0 | FR-056 |

### 4.8 Hazard Exhaustion that cannot be removed

Levels attributed to Dehydration and Malnutrition are **locked** until the full daily amount is taken (FR-033, FR-037); Suffocation levels are not. `RemoveExhaustionLevels` is the single removal path that respects the lock; a host (or the rest spec) calls it instead of `ReduceExhaustionOnSet` (Q-11).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-060 | `HazardRecord.RemovableLevels(currentLevel)` MUST return `max(0, currentLevel - dehydrationLevels - malnutritionLevels - contagionLevels)`, with level 0 for a creature without Exhaustion (table 4.8a). | SRD 5.2.1, "Dehydration [Hazard]", p. 181; "Malnutrition [Hazard]", p. 185; "Cackle Fever", p. 194 (FR-115) | unhappy (boundary) |
| FR-061 | `RemoveExhaustionLevels` MUST remove `min(levels, removable)` levels through `ConditionSet.ReduceExhaustion` and return the number removed and blocked (`levels - removed`). | SRD 5.2.1, "Exhaustion", "Removing Exhaustion Levels", p. 181; "Dehydration [Hazard]", p. 181 | happy |
| FR-062 | When nothing is removable, it MUST return `RemoveExhaustionLevelsRejected` holding an `ExhaustionRemovalRejection` (blocked count and a `cause`: `locked` when all are locked, `noExhaustion` when the set holds none), leave the set and record unchanged, and produce no condition events. | SRD 5.2.1, "Dehydration [Hazard]", p. 181; "Malnutrition [Hazard]", p. 185; `combat-foundations.md` FR-023 | unhappy (rule interaction) |
| FR-063 | After a removal, `suffocationLevels` MUST be reduced to at most the new level minus the locked levels (dehydration, malnutrition, contagion), so the record never claims more levels than the set holds. | package design (the record mirrors the set); SRD 5.2.1, "Exhaustion", p. 181 | unhappy (rule interaction) |
| FR-064 | `RemoveExhaustionLevels` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for `levels` below 1, before any other check. | `conditions.md` FR-045; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-065 | After the full amount is taken (FR-033, FR-037 set the count to 0), the same levels MUST become removable: 1 level blocked, then removable after a full day's water. | SRD 5.2.1, "Dehydration [Hazard]", p. 181 ("until the creature drinks the full amount") | unhappy (rule interaction) |

**4.8a Removing Exhaustion and the contagion lock** (FR-060 to FR-065, FR-115, FR-110; pp. 181, 185, 189, 194). Dehyd., Malnut., Susp., Cont. are the record counts; "asked" is `levels`.

| Label | Level | Dehyd. | Malnut. | Susp. | Cont. | Asked | Removable | Removed | Blocked | Level after | Req. |
|---|---|---|---|---|---|---|---|---|---|---|---|
| nothing locked (*happy*) | 3 | 0 | 0 | 0 | 0 | 1 | 3 | 1 | 0 | 2 | FR-061 |
| partly locked | 3 | 2 | 0 | 0 | 0 | 1 | 1 | 1 | 0 | 2 | FR-061 |
| all locked | 3 | 3 | 0 | 0 | 0 | 1 | 0 | 0 | 1 | 3 | FR-062 |
| locked by both | 3 | 2 | 1 | 0 | 0 | 1 | 0 | 0 | 1 | 3 | FR-062 |
| ask for more than removable | 4 | 2 | 0 | 0 | 0 | 3 | 2 | 2 | 1 | 2 | FR-061 |
| ask for exactly removable | 4 | 2 | 0 | 0 | 0 | 2 | 2 | 2 | 0 | 2 | FR-061 |
| suffocation not locked | 2 | 0 | 0 | 2 | 0 | 1 | 2 | 1 | 0 | 1 | FR-061, FR-063 |
| record clamped | 2 | 0 | 0 | 2 | 0 | 2 | 2 | 2 | 0 | none (Susp. 0) | FR-063 |
| no Exhaustion | none | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 1 (`noExhaustion`) | none | FR-062 |
| zero asked | 3 | 0 | 0 | 0 | 0 | 0 | n/a | an `srderr.ErrInvalidInput` error | n/a | n/a | FR-064 |
| from level 6 | 6 | 1 | 1 | 2 | 0 | 1 | 4 | 1 | 0 | 5 | FR-061 |
| unlocked after drinking | 1 | 0 (was 1) | 0 | 0 | 0 | 1 | 1 | 1 | 0 | none | FR-065 |
| contagion level locked | 1 | 0 | 0 | 0 | 1 | 1 | 0 | 0 | 1 (`locked`) | 1 | FR-115, FR-062 |
| one level free | 3 | 0 | 0 | 0 | 1 | 1 | 2 | 1 | 0 | 2 | FR-115, FR-061 |
| contagion and dehydration | 3 | 1 | 0 | 0 | 1 | 2 | 1 | 1 | 1 | 2 | FR-115, FR-061 |
| contagion ended, then removed | 1 | 0 | 0 | 0 | 0 (set from 1) | 1 | 1 | 1 | 0 | none | FR-115, FR-065 |
| sum above 6; negative count | n/a | 2; 0 | 2; 0 | 1; 0 | 2; -1 | n/a | n/a | an `srderr.ErrInvalidInput` error | n/a | n/a | FR-003, FR-115 |
| drift clamps contagion last | 2 | 1 | 0 | 0 | 2 | n/a | n/a | n/a | n/a | dehydration 0, contagion 2 | FR-110 |

### 4.9 Objects

An object is a nonliving, distinct thing (p. 185). `ObjectStatus` holds size, Armor Class, Hit Points, optional Damage Threshold, damage affinities and ability scores. The SRD "suggests" Armor Class and Hit Points (pp. 177-178); the host may set any values, and the tables are lookups.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-066 | `ObjectArmorClassTable.Of(substance)` MUST return the `ArmorClass` in table 4.9a for each of the seven substances. | SRD 5.2.1, "Breaking Objects", Object Armor Class, pp. 177-178 | happy |
| FR-067 | `ObjectHitPointsTable.Lookup(size, durability)` MUST return the fixed Hit Points and die expression in table 4.9b for Tiny, Small, Medium and Large, Fragile and Resilient. | SRD 5.2.1, "Breaking Objects", Object Hit Points, pp. 177-178 | happy |
| FR-068 | For every table row the fixed Hit Points MUST equal the die expression's average rounded down (`dice.Expression.Average`, `dice-and-rolls.md` FR-012). | SRD 5.2.1, "Round Down", p. 187; "Object Hit Points", p. 178 | unhappy (rule interaction) |
| FR-069 | `lookup` for Huge or Gargantuan MUST return `ObjectHitPointsSplitRequired(size)`, not a row: such an object is divided into Large or smaller sections tracked separately. | SRD 5.2.1, "Breaking Objects", pp. 177-178 ("divide it into Large or smaller sections") | unhappy (boundary) |
| FR-070 | `RollObjectHitPoints` MUST request exactly one roll of the entry's die expression and return `HitPoints` with current and maximum both equal to the total and Temporary Hit Points 0. | SRD 5.2.1, "Breaking Objects", Object Hit Points, pp. 177-178 | happy |
| FR-071 | `RollObjectHitPoints` for Huge or Gargantuan MUST return `ObjectHitPointsRollSplitRequired` and request no roll. | SRD 5.2.1, "Breaking Objects", pp. 177-178 | unhappy (boundary) |
| FR-072 | `ObjectStatus` MUST be constructible with `HitPoints` whose Temporary Hit Points are 0, and returns an `srderr.ErrInvalidInput` error for any above 0. Package design: the SRD gives objects Hit Points (p. 183) but grants them no Temporary Hit Points (p. 18 describes them for a creature).. | package design; SRD 5.2.1, "Hit Points", p. 183; "Temporary Hit Points", p. 18 | unhappy (invalid input) |
| FR-073 | `ObjectAffinities.Standard()` MUST return `DamageAffinities` with Immunity to Poison and Psychic damage and no Resistance or Vulnerability. | SRD 5.2.1, "Breaking Objects", "Damage Types and Objects", pp. 177-178 | happy |
| FR-074 | `ObjectAffinities.Standard(rulings)` MUST add the supplied Resistances, Vulnerabilities and Immunities to the standard Immunities (for example Vulnerability to Fire for paper or cloth). The GM's ruling arrives as data. | SRD 5.2.1, "Breaking Objects", "Damage Types and Objects", pp. 177-178 | unhappy (rule interaction) |
| FR-075 | `ApplyDamageToObject` MUST reduce current Hit Points by the damage that gets through the pipeline; an object reaching 0 MUST be destroyed: `isDestroyed` true, one `ObjectDestroyed` event on the transition (events: FR-109). | SRD 5.2.1, "Breaking Objects", pp. 177-178 ("destroyed when it has 0 Hit Points") | happy |
| FR-076 | Damage greater than the remaining Hit Points MUST leave 0 (not negative) and destroy the object once. | SRD 5.2.1, "Hit Points", p. 183 ("can't have less than 0") | unhappy (boundary) |
| FR-077 | `ApplyDamageToObject` on a destroyed object MUST return `ObjectDamageRejection.AlreadyDestroyed`, status unchanged, no events. | `doc/rules/testing.md` §4; SRD 5.2.1, "Breaking Objects", pp. 177-178 | unhappy (state error) |
| FR-078 | `ObjectDamageInstance` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for an empty list of `Damage`. | `hit-points-and-damage.md` FR-085; `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-079 | For an object with no Damage Threshold, `ApplyDamageToObject` MUST send each `Damage` through the pipeline separately (adjustments, Resistance, Vulnerability, Immunity overriding all), sum the results, and report the `DamageResolution` breakdown, exactly as `hit-points-and-damage.md` FR-085, FR-016 to FR-025 and FR-099 specify for a creature. `ObjectDamageResolver` MUST do so by calling the internal `DamageResolver` of `hit-points-and-damage.md` (FR-109), an import of a `combat` domain file allowed by FR-116, and MUST NOT call `HitPointStatus.ApplyDamage` or the `ApplyDamage` use case. It MUST NOT redefine the pipeline. | SRD 5.2.1, "Damage Types and Objects", p. 178; `hit-points-and-damage.md` FR-016 to FR-025, FR-085, FR-099 | happy |
| FR-080 | `DamageThreshold(amount)` MUST accept 1 or more and returns an `srderr.ErrInvalidInput` error for 0 or a negative amount. | package design; `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-081 | When the total of an instance of damage is **below** the threshold, `ApplyDamageToObject` MUST leave the status unchanged, produce the single event `ObjectDamageDeflected` and report `ThresholdOutcome.NotMet` (Immunity to all damage of that instance). | SRD 5.2.1, "Damage Threshold", p. 180 | happy |
| FR-082 | When the total is **equal to or above** the threshold, the object MUST take the **whole** instance (not only the excess) through the pipeline and report `ThresholdOutcome.Met` (table 4.9c). | SRD 5.2.1, "Damage Threshold", p. 180 ("it takes that entire instance of damage") | unhappy (boundary) |
| FR-083 | The threshold test MUST come before the pipeline: it compares the incoming amounts, before any adjustment, Resistance or Vulnerability. Resistance cannot make a met instance fail, and Vulnerability cannot make a missed one succeed.. | SRD 5.2.1, "Damage Threshold", p. 180; `hit-points-and-damage.md` Q-11 | unhappy (rule interaction) |
| FR-084 | The test MUST compare the **sum of the amounts of every `Damage` in one `ObjectDamageInstance`**, not each alone.. | SRD 5.2.1, "Damage Threshold", p. 180 ("from a single attack or effect") | unhappy (rule interaction) |
| FR-085 | A met threshold MUST NOT override Immunity: a `Damage` of an Immune type in a met instance still resolves to 0.. | SRD 5.2.1, "Damage Threshold", p. 180; "Immunity", p. 183 | unhappy (rule interaction) |
| FR-086 | `ObjectBrokenAutomatically(basis)` MUST reduce an intact object to 0 Hit Points, destroy it with one `ObjectDestroyed` event, and ignore the Damage Threshold and affinities (no damage is dealt). `basis` is an `AutomaticBreakBasis`: `fragileNonmagical` (p. 12: "As an action, you can automatically break or otherwise destroy a fragile, nonmagical object, such as a glass container or a piece of paper") or `gmAllowed` (p. 177: "If an object is exceedingly fragile, the GM may allow a creature to break it automatically with the Attack or Utilize action"). Fragility, magic and whether the action was spent are the host's decisions. A destroyed object returns `ObjectDamageRejected(alreadyDestroyed)`.. | SRD 5.2.1, "Breaking Objects", p. 12 and p. 177 | unhappy (rule interaction) |
| FR-087 | `ObjectStatus.AbilityCheck(ability)` MUST return `ConditionPermission.Forbidden` for `NoAbilityScores` or no score for that ability, and `ConditionPermission.Allowed` when the ability has a score (`AssignedAbilityScores` holds the scores by `Ability`). | SRD 5.2.1, "Breaking Objects", "No Ability Scores", pp. 177-178 | happy |
| FR-088 | `ObjectStatus.SavingThrow(ability)` MUST return `SavingThrowEffect.AutomaticFailure` for `NoAbilityScores` or no score for that ability (all six in the no-score case), and `unaffected` when it has a score. The caller builds the failure with `d20.FailWithoutRoll` and `d20.Automatic` (`d20-tests.md` FR-036). | SRD 5.2.1, "Breaking Objects", pp. 177-178 ("fails all saving throws"); "Saving Throw", p. 187 | unhappy (rule interaction) |
| FR-089 | `CompositeObject` MUST be constructible from 2 or more `ObjectStatus` sections, each with a `CollapseRuling`, returns an `srderr.ErrInvalidInput` error for fewer than 2, and expose its sections as a copy. | SRD 5.2.1, "Object", p. 185 ("more than one object"); "Breaking Objects", pp. 177-178 | unhappy (invalid input) |
| FR-090 | `CompositeObject.ReplaceSection(index, status)` MUST return a new composite with that section replaced and the events of FR-091 to FR-093, and returns an `srderr.ErrInvalidInput` error for an index outside the sections. | `doc/rules/architecture.md` §3 | unhappy (invalid input) |
| FR-091 | When a section whose ruling is `CollapsesWhole` becomes destroyed, `CompositeStatus` MUST become `collapsed` and one `ObjectCollapsed` event be produced. A destroyed `DoesNotCollapse` section MUST NOT collapse the composite. The ruling is the GM's, supplied by the host. | SRD 5.2.1, "Breaking Objects", pp. 177-178 ("The GM determines whether destroying part of an object causes the whole thing to collapse") | happy |
| FR-092 | When every section is destroyed and none had `CollapsesWhole`, the status MUST be `allSectionsDestroyed`; `collapsed` takes precedence when both hold. | SRD 5.2.1, "Breaking Objects", pp. 177-178 | unhappy (rule interaction) |
| FR-093 | `ObjectCollapsed` MUST be produced once only: a later `ReplaceSection` on an already collapsed composite MUST NOT produce it again. | `doc/rules/observability.md` §11; `doc/rules/testing.md` §4 | unhappy (state error) |

**4.9a Object Armor Class** (FR-066; p. 178; *happy*).

| Substance | `clothPaperRope` | `crystalGlassIce` | `wood` | `stone` | `ironSteel` | `mithral` | `adamantine` |
|---|---|---|---|---|---|---|---|
| Armor Class | 11 | 13 | 15 | 17 | 19 | 21 | 23 |

**4.9b Object Hit Points** (FR-067 to FR-071; p. 178). Rows Tiny to Large are *happy*; the rest are unhappy (boundary).

| Size | Fragile (HP, die) | Resilient (HP, die) | Req. |
|---|---|---|---|
| Tiny | 2, 1d4 | 5, 2d4 | FR-067, FR-068 |
| Small | 3, 1d6 | 10, 3d6 | FR-067, FR-068 |
| Medium | 4, 1d8 | 18, 4d8 | FR-067, FR-068 |
| Large | 5, 1d10 | 27, 5d10 | FR-067, FR-068 |
| Huge, Gargantuan (both durabilities) | `ObjectHitPointsSplitRequired`, no die | same | FR-069 |
| roll, Medium Resilient minimum | n/a | 4 (4d8 all 1s) | FR-070 |
| roll, Medium Resilient maximum | n/a | 32 (4d8 all 8s) | FR-070 |
| roll, Huge | `ObjectHitPointsRollSplitRequired`, no roll | | FR-071 |

**4.9c Damage Threshold** (FR-073 to FR-085; pp. 178, 180, 183). Threshold 10, Hit Points 30, standard affinities, unless the row says otherwise. "Taken" is the Hit Points lost.

| Label | Instance | Affinity | Threshold test | Taken | Events | Req. |
|---|---|---|---|---|---|---|
| one below | 9 Slashing | none | notMet | 0 | `ObjectDamageDeflected` | FR-081 |
| equal | 10 Slashing | none | met | 10 | `ObjectDamaged` | FR-082 |
| one above takes all | 11 Slashing | none | met | 11 (not 1) | `ObjectDamaged` | FR-082 |
| zero amount | 0 Slashing | none | notMet | 0 | `ObjectDamageDeflected` | FR-081 |
| Resistance after the test | 12 Slashing | Resistance Slashing | met | 6 | `ObjectDamaged` | FR-083 |
| Vulnerability does not lift a miss | 9 Slashing | Vulnerability Slashing | notMet | 0 (not 18) | `ObjectDamageDeflected` | FR-083 |
| Resistance does not drop a hit | 10 Slashing | Resistance Slashing | met | 5 | `ObjectDamaged` | FR-083 |
| components add up | 6 Slashing and 5 Fire | none | met (11) | 11 | `ObjectDamaged` | FR-084 |
| components each below | 4 Slashing and 5 Fire | none | notMet (9) | 0 | `ObjectDamageDeflected` | FR-084 |
| Immune type in a met instance | 8 Poison and 4 Slashing | standard | met (12) | 4 | `ObjectDamaged` | FR-085 |
| no threshold (*happy*) | 1 Slashing | no threshold | notApplicable | 1 | `ObjectDamaged` | FR-079 |
| Immunity only | 5 Poison | standard, no threshold | notApplicable | 0 | none | FR-073, FR-079 |
| Vulnerability ruling | 4 Fire | Vulnerability Fire, no threshold | notApplicable | 8 | `ObjectDamaged` | FR-074 |
| destroys; exactly destroys | 40 Slashing; 30 Slashing | no threshold | notApplicable | 30 (Hit Points 0) | `ObjectDamaged`, `ObjectDestroyed` | FR-075, FR-076 |
| already destroyed | 5 Slashing | none | n/a | `alreadyDestroyed` | none | FR-077 |

**4.9d Ability checks and saving throws** (FR-087, FR-088; pp. 178, 187).

| Ability scores | Ability | Check | Saving throw | Req. |
|---|---|---|---|---|
| `NoAbilityScores` (*happy*; one case per ability: Strength, Dexterity, Constitution, Intelligence, Wisdom, Charisma) | each | `ConditionPermission.Forbidden` | `SavingThrowEffect.AutomaticFailure` | FR-087, FR-088 |
| `AssignedAbilityScores` with Strength 20 only | Strength; Dexterity | `allowed`; `forbidden` | `unaffected`; `automaticFailure` | FR-087, FR-088 |

**4.9e Composite objects** (FR-089 to FR-093; pp. 178, 185). Two Large sections, A and B.

| Label | A ruling | B ruling | Change | Status after | Events | Req. |
|---|---|---|---|---|---|---|
| one section only; bad index | n/a | n/a | construct with 1; replace index 2 | an `srderr.ErrInvalidInput` error | none | FR-089, FR-090 |
| damaged, not destroyed (*happy*) | CollapsesWhole | DoesNotCollapse | A to 1 Hit Point | standing | the section's own | FR-091 |
| collapsing section destroyed (*happy*) | CollapsesWhole | DoesNotCollapse | A destroyed | collapsed | `ObjectCollapsed` | FR-091 |
| other section destroyed | CollapsesWhole | DoesNotCollapse | B destroyed | standing | none | FR-091 |
| all destroyed, no collapse | DoesNotCollapse | DoesNotCollapse | A then B destroyed | allSectionsDestroyed | none | FR-092 |
| collapse beats all destroyed | CollapsesWhole | DoesNotCollapse | B then A destroyed | collapsed | `ObjectCollapsed` once | FR-092, FR-093 |
| already collapsed | CollapsesWhole | DoesNotCollapse | B destroyed after A | collapsed | none | FR-093 |

### 4.10 Cross-cutting, events, record drift, rejections and thresholds

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-094 | `RollBurningDamage`, `ResolveFall` and `RollObjectHitPoints` MUST obtain every roll through the injected roller. When `Roll` returns an error, the use case MUST return it wrapped once with `srderr.WrapPort(dice.Port, "roll", err)` (a `*srderr.PortError`; `errors.Is` reaches the cause) and return no result and no new state. | `doc/rules/architecture.md` §4; `doc/rules/testing.md` §4; `dice-and-rolls.md` FR-049, FR-052 | unhappy (collaborator failure) |
| FR-095 | When the `dice.Roller` returns a result for the wrong expression (count, die or modifier), those use cases MUST return `srderr.ContractViolation(dice.Port, "roll", detail)` (a `*srderr.PortError` wrapping `srderr.ErrContractViolation`), never a panic. A fixed roller MUST show `RollBurningDamage` requested exactly `1d4` once, `ResolveFall` exactly `Nd6` once (N from FR-015) and `RollObjectHitPoints` exactly the entry's expression once. | `doc/rules/testing.md` §4; `dice-and-rolls.md` FR-051 | unhappy (collaborator failure) |
| FR-096 | Every operation and use case MUST return new values and not mutate inputs; every slice in a command, result or event MUST be a copy (accessors copy on read), and mutating a slice passed to a constructor afterwards MUST NOT change the instance. | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |
| FR-097 | Given the same state, command and fixed rolls, every operation MUST return an equal result. No operation may read the clock. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 (determinism) | unhappy (determinism) |
| FR-098 | Hazard use cases MUST change a `ConditionSet` only through `ConditionSet.Apply` and `ConditionSet.ReduceExhaustion`, return the events and outcome unchanged, and give the same outcome as `ApplyConditionToSet` and `ReduceExhaustionOnSet` (`conditions.md` FR-116, FR-117). Applying Exhaustion passes level 1 (`conditions.md` FR-123). | `conditions.md` FR-116, FR-117, FR-123; D-1 | happy |
| FR-099 | Each use case MUST run through `CombatTelemetry` (`combat-foundations.md` FR-035): one span `srd5e.combat.<operation>` (`<operation>` is the `snake_case` use-case name; this spec adds its twelve constant pairs and span rows, section 7), recording once per call that returns a result the shared `srd5e.combat.operations`, `srd5e.combat.operation_duration` and `srd5e.combat.events` signals (error path: FR-101) with the shared attributes `srd5e.combat.operation`, `srd5e.outcome`, `srd5e.combat.result` and `srd5e.combat.result_class` (`combat-foundations.md` FR-029, FR-030). This spec MUST NOT define its own operations counter, duration histogram, errors counter, second span family or operation/result attribute. | `combat-foundations.md` FR-027 to FR-030, FR-035, FR-037; `doc/rules/observability.md` §5, §6 | happy |
| FR-100 | Every result MUST be classified as one `CombatResultClass` by the section 7 table. `applied` records `srd5e.outcome` = `success` and no error status. `refused` (legal input the rules disallow in the current state) records `failure`, **no** error status, and MUST NOT be logged. The `refused` values are at least both `BurningRejection` values, every `BreathRejection`, `RemoveExhaustionLevelsRejected`, `ObjectDamageRejection.AlreadyDestroyed`, `ObjectHitPointsRollSplitRequired`. `MalnutritionSaveRequired`, normal rule outcomes (a destroyed object, a deflected hit, `noExhaustion`, `exhaustionAtMaximum`, `immune`) and `FallWithoutDamage` are `applied`. No result is `invalid`: a precondition violation is a thrown an `srderr.ErrInvalidInput` error, propagated with the span recording it (`combat-foundations.md` FR-035). | `combat-foundations.md` FR-031, FR-034, FR-035 (consistent with `hit-points-and-damage.md` FR-080); `doc/rules/observability.md` §6, §7 | unhappy (telemetry on failure) |
| FR-101 | A returned `*srderr.PortError` MUST be recorded once, by the use case that returned it (`shared-types-and-ownership.md` FR-044): the span records it through the `Tracer`, and `CombatTelemetry` increments the shared `srd5e.combat.errors` counter with `srd5e.combat.operation` and `error.type` (`dice_roller_error`, from `srderr.ErrorType`, `shared-types-and-ownership.md` FR-041) and not `srd5e.combat.operations`. It MUST be returned (the `PortError` from `WrapPort`, `errors.Is` reaching the cause) and not logged. A `refused` result MUST NOT increment `srd5e.combat.errors` or carry an `error.type`. | `combat-foundations.md` FR-032, FR-035; `doc/rules/observability.md` §7 | unhappy (telemetry on failure) |
| FR-102 | Every use case MUST return identical results with the default zero-value `telemetry.Noop` and with a recording double. | `doc/rules/observability.md` §9 | unhappy (determinism) |
| FR-103 | Attributes MUST come from closed sets (operation names, result names, result classes, event kinds, `HazardKind` names for `srd5e.combat.hazard.kind`) and MUST NOT contain amounts, distances, ids or free text. | `doc/rules/observability.md` §5; `combat-foundations.md` FR-033 | unhappy (telemetry) |
| FR-104 | Package `combat` MUST export every use case and handoff type in section 6, every variant of each closed type, each with a godoc comment starting with its name; there is no re-export package package. The architecture test derives its expected list from every type named in sections 3, 5 and 6, except the four internal services, and checks that each is exported. | `doc/rules/architecture.md` §4, §7 | happy |
| FR-105 | Every event in section 5 MUST satisfy `Event` through the one closed sub-family `EnvironmentEvent` that this spec owns (`shared-types-and-ownership.md` FR-019), be an immutable value (unexported fields, value receivers), be named in the past tense, and be equal by value including `kind` and `combatants`. `kind` MUST be a constant per type, match `[a-z][a-z0-9_]*`, be unique across the package and never come from reflection or `%T` (values in the section 5 table). `Combatants()` MUST return a copy: each of the seven creature events holds exactly one id, the `target` of the producing call; each of the four object events returns an empty slice. The use cases that produce a creature event (`ExtinguishBurning`, `AdvanceBreath`, `ResolveDehydrationDayEnd`, `ResolveMalnutritionDayEnd`, `RestoreBreathing`) therefore take a `CombatantID` `target`; `RollBurningDamage`, `ResolveFall` and `RollObjectHitPoints` produce no event. | `combat-foundations.md` FR-015 to FR-021; `shared-types-and-ownership.md` FR-019 | happy |
| FR-106 | `BurningEnded(target, method)` MUST be emitted exactly once by every `ExtinguishBurning` call that ends the fire, whatever the method (`RollOnTheGround` with Prone `ConditionAdded` or `ConditionAlreadyHeld`, `doused`, `submerged`, `suffocated`), after any condition events of the call. It MUST NOT be emitted on a rejection. | SRD 5.2.1, "Burning [Hazard]", p. 178; `combat-foundations.md` FR-021 | happy |
| FR-107 | `BreathRestored(target, levelsRemoved)` MUST be emitted exactly once by every `RestoreBreathing` call that changes a `BreathSuffocating` creature to `BreathNormal` or removes levels (FR-108), after any condition events. `levelsRemoved` is the levels actually removed from the set: `suffocationLevels` limited to the level the set holds (FR-059); it MAY be 0 (choking, restored before a turn ends). It MUST NOT be emitted in the no-op case of FR-058. | SRD 5.2.1, "Suffocation [Hazard]", p. 189 | happy |
| FR-108 | `RestoreBreathing` on a `BreathNormal` or `BreathHolding` creature whose `suffocationLevels` is 1 or more (a leftover record) MUST behave as FR-056: remove those levels through `reduceExhaustion` (limited as in FR-059), set `suffocationLevels` to 0, return `BreathNormal` and emit `BreathRestored`. | SRD 5.2.1, "Suffocation [Hazard]", p. 189; `doc/rules/testing.md` §4 | unhappy (state error) |
| FR-109 | `ApplyDamageToObject` MUST produce these events and no others, in order: threshold not met, `ObjectDamageDeflected` alone (status unchanged); otherwise, when the final total through the pipeline is above 0, `ObjectDamaged(amount)` (Hit Points actually lost, limited to the current Hit Points, FR-076), then `ObjectDestroyed` when Hit Points reach 0; when the final total is 0 (all Immune, or a zero amount with no threshold) no event and result value `no_damage`. `ObjectDamaged` MUST NOT carry 0. A rejection produces no event. `ObjectBrokenAutomatically` produces `ObjectDestroyed` only. | SRD 5.2.1, "Breaking Objects", pp. 177-178; "Damage Threshold", p. 180; `doc/rules/observability.md` §11 | unhappy (rule interaction) |
| FR-110 | A day-end, turn-end, restore or removal use case MUST NOT panic because a host-supplied `HazardRecord` disagrees with the `ConditionSet`. After its own increments it MUST clamp the returned record so the sum of the four counts does not exceed the Exhaustion level the returned set holds, reducing `suffocationLevels` first, then `malnutritionLevels`, `dehydrationLevels`, `ContagionLevels`. The record received is read through the same clamp. The constructor still throws (FR-003). | package design (FR-063); `doc/rules/architecture.md` §6; `doc/rules/testing.md` §4 | unhappy (state error) |
| FR-111 | A hazard taking a creature from Exhaustion 5 to 6 MUST return `exhaustionGained` and NOT decide the death. The host calls `IsDeadFromExhaustion` (`conditions.md`) and `RecordExhaustionDeath` (`hit-points-and-damage.md`) afterwards. | SRD 5.2.1, "Exhaustion", p. 181 (death at level 6); `hit-points-and-damage.md` FR-095 | unhappy (boundary) |
| FR-112 | `DamageThresholdGate.Evaluate(threshold, instance)` MUST take an optional `DamageThreshold` and the `Damage` list of one instance, know nothing of `ObjectStatus`, and return `notApplicable` for no threshold, `met` when the sum is equal to or above it and `notMet` below (FR-081 to FR-084). A creature with a threshold ("A creature or an object", p. 180) uses the same gate. `ApplyDamage` does not call it (Q-20): the host evaluates the gate, skips `ApplyDamage` for `notMet`, and passes the whole instance for `met`. | SRD 5.2.1, "Damage Threshold", p. 180; `hit-points-and-damage.md` Q-11 (closed here) | unhappy (rule interaction) |
| FR-113 | A rejection reason MUST be an enum or value object held as a field of the `...Rejected` variant: `BurningDamageRejected(BurningRejection)`, `BurningExtinguishRejected(BurningRejection)`, `BreathAdvanceRejected(BreathRejection)`, `ObjectDamageRejected(ObjectDamageRejection)`, `RemoveExhaustionLevelsRejected(ExhaustionRemovalRejection)`. No exported name MUST be a bare `NotBurning`, `Burning`, `Breathing`, `HoldingBreath`, `Suffocating`, `Doused` or similar. | `combat-foundations.md` FR-022 to FR-024 | unhappy (name collision) |
| FR-114 | A building or a vehicle ("composed of many objects", p. 12) MUST NOT be modelled as one `ObjectStatus`: the host models it as a `CompositeObject` of sections (FR-089 to FR-093); the engine offers no building or vehicle type. | SRD 5.2.1, "What Is an Object?", p. 12 | unhappy (boundary) |

**4.10a Suffocation turn end, record drift and breath restoration** (FR-053 to FR-055, FR-107, FR-108, FR-110, FR-111; pp. 181, 189).

| Label | Call | Set before | Record before | Result | Set / record after | Req. |
|---|---|---|---|---|---|---|
| drift, restore | `RestoreBreathing` | Exhaustion 1 | suffocation 3 | `BreathNormal` | none, suffocation 0, `BreathRestored(1)` | FR-107, FR-110 |
| leftover record, breathing | `RestoreBreathing` on `BreathNormal` | Exhaustion 2 | suffocation 2 | `BreathNormal` | none, suffocation 0, `BreathRestored(2)` | FR-108 |
| leftover record, holding | `RestoreBreathing` on `BreathHolding` | Exhaustion 3 | suffocation 1, dehydration 2 | `BreathNormal` | level 2, suffocation 0, `BreathRestored(1)` | FR-108 |
| choking, restored at once | `RestoreBreathing` on `BreathSuffocating` | none | suffocation 0 | `BreathNormal` | none, `BreathRestored(0)` | FR-107 |
| turn end, suffocating (*happy*) | `ResolveSuffocationTurnEnd` on `BreathSuffocating` | none | suffocation 0 | `exhaustionGained` | level 1, suffocation 1 | FR-053 |
| turn end, breathing or holding | `ResolveSuffocationTurnEnd` | Exhaustion 1 | suffocation 0 | `noExhaustion` | unchanged | FR-054 |
| turn end, level 5 to 6 | `ResolveSuffocationTurnEnd` on `BreathSuffocating` | Exhaustion 5 | suffocation 5 | `exhaustionGained` (host then calls `IsDeadFromExhaustion`) | level 6, suffocation 6 | FR-053, FR-111 |
| turn end, level 6; immune | `ResolveSuffocationTurnEnd` on `BreathSuffocating` | Exhaustion 6; none (immune) | suffocation 6; 0 | `exhaustionAtMaximum`; `immune` | unchanged | FR-055 |

### 4.11 Ownership, the contagion lock, naming, signals and result parts

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-115 | `HazardRecord.ContagionLevels` MUST hold the Exhaustion levels that an effect holds until it ends. It MUST default to 0, count in the 6-level sum of FR-002 and FR-003, be subtracted in `RemovableLevels` (FR-060), be clamped last (FR-110), and change only through `HazardRecord.WithContagionLevels(count)`, which returns a new record and returns an `srderr.ErrInvalidInput` error for a negative count or a sum above 6. No use case here raises it: the host sets it from the toolbox rule that applies the level, and lowers it, then calls `RemoveExhaustionLevels`, when the effect ends.. | SRD 5.2.1, "Cackle Fever", p. 194 ("gains 1 Exhaustion level, which lasts until the contagion ends on the creature"); "Long Rest", p. 185; `toolbox-curses-and-contagions.md` HB-1 | unhappy (rule interaction) |
| FR-116 | This spec imports only specs left of `attacks` and has no dependents. Its domain files (`hazard_*.go`, `object_*.go`) MUST import only the kernel packages, the published API of `content` and `d20`, and the `combat` domain files of `combat-foundations`, `hit-points-and-damage` and `conditions`; they MUST NOT import `telemetry` (`doc/rules/architecture.md` §1, §9). Its `*_usecase.go` files MAY also import `telemetry`, use `CombatTelemetry` and `CombatSignals`, and use the exported types of those three specs. Neither kind of file MAY import `attacks`, `initiative-and-actions`, `encounter-aggregate`, `combat-action-resolution` or any `character` file. No `combat` spec to its left and no `character`, `d20` or `content` spec may depend on it. A `character` host (Long Rest, travel, food) reaches `RemoveExhaustionLevels` and the day-end use cases through the host, not an import. | `shared-types-and-ownership.md` FR-013, FR-015, FR-045; `combat-foundations.md` FR-024, FR-026; `doc/rules/architecture.md` §2 | unhappy (forbidden edge) |
| FR-117 | Every use case MUST take `<UseCase>Command` and return its own closed `<UseCase>Result` with operation-prefixed variants; section 6 names are the only ones exported, no generic name (`Applied`, `Rejected`, `Succeeded`, `Found`, `NotFound`), reasons held as fields (FR-113). Every exported name is unique across package `combat`, so a name another spec uses (`Damage`, `ApplyDamage`, `DamageApplied`, `ConditionAdded`) is never redefined. | `shared-types-and-ownership.md` FR-017, FR-018, FR-021; `combat-foundations.md` FR-022 to FR-025 | unhappy (name collision) |
| FR-118 | No use case of this spec MAY call, construct or depend on another use case (`ApplyDamage`, `ApplyConditionToSet`, `ReduceExhaustionOnSet`, `d20.Resolver.Resolve`, `RollObjectHitPoints` from `ApplyDamageToObject`, and so on). Each takes finished values (`Damage`, `d20.Outcome`, `ConditionSet`) from its command, composes only pure domain operations, and records only its own signals. | `shared-types-and-ownership.md` FR-044; `doc/rules/architecture.md` §4; `doc/rules/observability.md` §7 | unhappy (nested telemetry) |
| FR-119 | `srd5e.combat.hazard.damage_rolled` MUST be recorded once by each `RollBurningDamage` call returning `BurningDamageRolled` (`srd5e.combat.hazard.kind` = `burning`) and each `ResolveFall` call returning `FallDamageRolled` (`falling`), with the amount of the returned `Damage` as rolled, before any `HalfDamage` or Resistance; not for `BurningDamageRejected` or `FallWithoutDamage`. `srd5e.combat.hazard.exhaustion_gained` MUST be recorded with value 1 once by each `ResolveDehydrationDayEnd`, `ResolveMalnutritionDayEnd` or `ResolveSuffocationTurnEnd` call whose outcome is `exhaustionGained`, with kind `dehydration`, `malnutrition` or `suffocation`; not for `noExhaustion`, `exhaustionAtMaximum`, `immune` or `MalnutritionSaveRequired`. The metric names and the `srd5e.combat.hazard.kind` key MUST be `CombatSignals` constants (`combat-foundations.md` FR-039), the value the `HazardKind` `name`, and nothing else is an attribute (FR-103). | `doc/rules/observability.md` §5, §6; `combat-foundations.md` FR-037, FR-039 | happy |
| FR-120 | A result field for an optional part MUST be a named state, never a nil value or a boolean: `BurningExtinguished.ProneApplication` is a `ProneApplicationOutcome` (`ProneApplied(application)` or `ProneNotApplied`) and `BreathingRestored.Reduction` an `ExhaustionReductionOutcome` (`ExhaustionReductionMade(result)` or `ExhaustionReductionNotAttempted`). `RollOnTheGround` is `ProneApplied` (FR-009, FR-010); `doused`, `submerged`, `suffocated` are `ProneNotApplied` (FR-012); `RestoreBreathing` with `suffocationLevels` 1 or more is `ExhaustionReductionMade` (FR-056, FR-059, FR-108) and with 0 `ExhaustionReductionNotAttempted` (FR-058). `ExhaustionLevelsRemoved.Reduction` is the `ReduceExhaustionOnSetResult` itself (FR-061). | `doc/rules/architecture.md` §3; `conditions.md` FR-044 | unhappy (rule interaction) |
| FR-121 | `FallDistance`, `WaterAmount` and `FoodAmount` are programmer-supplied preconditions, not untrusted input: a negative value returns an `srderr.ErrInvalidInput` error (FR-014, FR-029). | `hit-points-and-damage.md` FR-094; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-122 | `NewHazardRecord(...) (HazardRecord, error)` MUST return the record and a nil error for values it accepts, otherwise a `*HazardRecordError` carrying a `HazardRecordRejection` (`negativeCount`: any level count or `daysWithoutFood` below 0; `levelSumAboveSix`) that satisfies `errors.Is(err, srderr.ErrInvalidInput)`. It MUST NOT panic for any input; `MustHazardRecord` panics with that error iff `NewHazardRecord` returns one. | `doc/rules/architecture.md` §3, §6 (untrusted input gets a non-panicking `NewX`) | unhappy (invalid input) |
| FR-123 | **Magic Item Resilience is referenced, not restated.** Resilience of a magic item that is an object (p. 206) is owned by `content-magic-items.md` FR-087 (`MagicItemResilience`, `MagicItemResilienceFacts.DefaultFor`) and `magic-item-rules.md` FR-073 (`MagicItemResilienceRules.Effective`). This spec MUST NOT define a second table or enum for it, and `ObjectAffinities.Standard` and `ApplyDamageToObject` MUST NOT read it. A host with `resistantToAllDamage` or `impervious` adjusts the `ObjectDamageInstance` it passes in (`DamageAdjustment`); `nonmagicalDurability` means the tables here apply as for a nonmagical object. | SRD 5.2.1, "Magic Item Resilience", p. 206; `shared-types-and-ownership.md` FR-058 | unhappy (rule interaction) |
| FR-124 | **Underwater Fire Resistance is the host's hand-off, not a dependency.** "Anything underwater has Resistance to Fire damage" (p. 16) includes objects. `ApplyDamageToObject` takes no `UnderwaterStatus` and MUST NOT import `mounted-and-underwater-combat.md`, whose spec (rank 24) is downstream of this one (rank 23; `shared-types-and-ownership.md` section 4.11). A host whose object is underwater passes an `ObjectStatus` whose `DamageAffinities` already hold Resistance to Fire (the output of `UnderwaterFireResistance.Affinities` over the object's affinities, or `ObjectAffinities.Standard(rulings)` with that Resistance, FR-074); the pipeline then halves Fire damage as for any Resistance (FR-079), and the host restores the dry affinities when the object leaves the water. | SRD 5.2.1, "Fire Resistance", p. 16; "Resistance and Vulnerability", p. 17; `shared-types-and-ownership.md` section 4.11, FR-044 | unhappy (rule interaction) |

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

Everything in the `hazard_*.go` and `object_*.go` domain files of package `combat` is pure, synchronous and silent. Types are defined in section 3; this section adds only what is not there.

- **Value objects:** `HazardRecord` (FR-002 to FR-004, FR-060, FR-115, clamp FR-110), `FallDistance`, `WaterAmount`, `FoodAmount`, `BreathCapacity`, `ObjectHitPointsEntry`, `DamageThreshold`, `ObjectAffinities` (factory over `DamageAffinities`), `ObjectStatus`, `ExhaustionRemovalRejection` (blocked count and `cause`: `locked`, `noExhaustion`), `CompositeObject` (an `ObjectStatus` slice, copied on read, with a `CollapseRuling` each; `ReplaceSection` returns the new composite and events; models a building or vehicle, FR-114).
- **Checked-construction and optional-part types:** `HazardRecordCreation`, `HazardRecordRejection` (`negativeCount`, `levelSumAboveSix`, FR-122); `ProneApplicationOutcome`, `ExhaustionReductionOutcome` (FR-120).
- **Enums:** `AutomaticBreakBasis` (`fragileNonmagical`, `gmAllowed`), `ObjectDamageRejection` (`alreadyDestroyed`), `CompositeStatus` (`standing`, `collapsed`, `allSectionsDestroyed`), `HazardExhaustionOutcome` (`noExhaustion`, `exhaustionGained`, `exhaustionAtMaximum`, `immune`; D-2).
- **Entities and aggregates:** none. The host holds one status value per creature or object; `CombatantID` is owned by `combat-foundations.md` FR-001 (Q-3).
- **Domain events** (past tense, FR-105; one closed family `EnvironmentEvent` satisfying `Event`, Q-16; condition events come from `conditions.md` and are carried). Each is `kind` / `combatants` / emitted by: `BurningEnded(target, method)` `burning_ended` [target] FR-012, FR-106; `DehydrationRecovered(target, levels)` `dehydration_recovered` [target] FR-033; `MalnutritionRecovered(target, levels)` `malnutrition_recovered` [target] FR-037; `BreathHeldStarted(target)` `breath_held_started` [target] FR-047; `RanOutOfBreath(target)` `ran_out_of_breath` [target] FR-049; `BeganChoking(target)` `began_choking` [target] FR-050; `BreathRestored(target, levelsRemoved)` `breath_restored` [target] FR-056, FR-107, FR-108; `ObjectDamaged(amount)` `object_damaged` [] FR-075, FR-109; `ObjectDamageDeflected` `object_damage_deflected` [] FR-081, FR-109; `ObjectDestroyed` `object_destroyed` [] FR-075, FR-086, FR-109; `ObjectCollapsed` `object_collapsed` [] FR-091, FR-093.
- **Domain services** (stateless): `DamageThresholdGate` (FR-081 to FR-085, FR-112), `FallResolver`, `HazardDayEnd`, `BreathStateMachine`, `ObjectDamageResolver` (uses the `hit-points-and-damage.md` pipeline, FR-079).
- **Failures:** expected outcomes are results with reasons held as `...Rejection` values (FR-113), plus the split-required results and `MalnutritionSaveRequired`. A violated precondition is a returned `srderr.ErrInvalidInput` error (FR-003, FR-014, FR-026, FR-029, FR-052, FR-064, FR-072, FR-078, FR-080, FR-089, FR-090, FR-121); a restored ledger uses `NewHazardRecord` (FR-122). Host-state drift in a `HazardRecord` is a clamp (FR-110). Port failure is `*srderr.PortError` (FR-094, FR-095). Nothing here reads content.

## 6. Interface

Each use case is a type built by `NewX(opts ...Option)` (the three with a port take it first: `NewRollBurningDamage(roller Roller, opts ...Option)`) with one method `Run(ctx context.Context, cmd <UseCase>Command) (<UseCase>Result, error)`; `ctx` is first because the call emits a span, and the `error` return carries only port failures (`doc/rules/architecture.md` §4, go.md section 4). Each takes an optional telemetry bundle (`WithTelemetry`, default the zero-value `telemetry.Noop`), runs through `CombatTelemetry` (FR-099), takes an explicit `<UseCase>Command` and returns its own closed `<UseCase>Result`; variants are operation-prefixed and a rejection reason is a field (FR-113).

| Use case | Command fields | Result variants | Port |
|---|---|---|---|
| `RollBurningDamage` | `state` | `BurningDamageRolled(damage, roll, state)`, `BurningDamageRejected(BurningRejection)` | `dice.Roller` |
| `ExtinguishBurning` | `state, method, conditions, target` | `BurningExtinguished(state, conditions, proneApplication, events)`, `BurningExtinguishRejected(BurningRejection)` | none |
| `ResolveFall` | `distance, surface` | `FallDamageRolled(damage, adjustments, roll, diceCount)`, `FallWithoutDamage` | `dice.Roller` |
| `LandAfterFall` | `conditions, target, damageTaken` | `FallLandedProne(conditions, application, events)`, `FallLandedStanding(conditions)` | none |
| `ResolveDehydrationDayEnd` | `size, water, conditions, target, record` | `DehydrationDayEnded(conditions, record, outcome, events)` | none |
| `ResolveMalnutritionDayEnd` | `size, food, savingThrow, conditions, target, record` | `MalnutritionDayEnded(conditions, record, outcome, events)`, `MalnutritionSaveRequired(dc, ability)` | none |
| `AdvanceBreath` | `state, change, target` | `BreathAdvanced(state, events)`, `BreathAdvanceRejected(BreathRejection)` | none |
| `ResolveSuffocationTurnEnd` | `state, conditions, target, record` | `SuffocationTurnEnded(conditions, record, outcome, events)` | none |
| `RestoreBreathing` | `state, conditions, target, record` | `BreathingRestored(state, conditions, record, reduction, levelsRemoved, events)` | none |
| `RemoveExhaustionLevels` | `conditions, target, record, levels` | `ExhaustionLevelsRemoved(conditions, record, reduction, removed, blocked, events)`, `RemoveExhaustionLevelsRejected(ExhaustionRemovalRejection)` | none |
| `RollObjectHitPoints` | `size, durability` | `ObjectHitPointsRolled(entry, roll, hitPoints)`, `ObjectHitPointsRollSplitRequired(size)` | `dice.Roller` |
| `ApplyDamageToObject` | `status, harm` | `ObjectDamageApplied(status, events, resolutions, threshold)`, `ObjectDamageRejected(ObjectDamageRejection)` | none |

- **Field types.** `application` is the `ApplyConditionToSetResult` of the call (`ConditionAdded`, `ConditionAlreadyHeld`, `ConditionImmune` or `ExhaustionAtMaximum`); `events` are the carried condition events plus this spec's, in order; `target` is a `CombatantID`; `outcome` is a `HazardExhaustionOutcome`; `proneApplication` and `reduction` of `BreathingRestored` are the named-state types of FR-120 (`ExhaustionReductionMade(result)` holds `ExhaustionReduced`, `ExhaustionEnded` or `ExhaustionReductionNotPresent`); `reduction` of `ExhaustionLevelsRemoved` is the `ReduceExhaustionOnSetResult` itself (a call was made, FR-061). No field is null or a boolean standing in for a named state (FR-120).
- **Ports:** only the consumer-side `Roller` interface declared in `combat`'s `ports.go` (`Roll(dice.Expression) (dice.Result, error)`, satisfied structurally by `dice.Roller` implementations); no new port, test double or contract suite. Failure is a `*srderr.PortError`; a wrong result is detected by the use case itself and returned as `srderr.ContractViolation` (FR-094, FR-095).
- **Handoffs:** `Damage`, `HalfDamage` and the `DamageDelivery` `OtherSource` go to `ApplyDamage`; `ConditionSet` and condition events come from and return to `conditions.md`; `d20.Outcome` for the landing check and save comes from `d20.Resolver.Resolve`, which the host runs with the published `FallLandingCheck` or `MalnutritionSaveRequired` data (FR-118). A fall decided elsewhere reaches `ResolveFall` as a `FallDistance` and `FallSurface` the host builds.
- **Export list of package `combat`** (FR-104 derives from this): the twelve use cases with their `<UseCase>Command` and closed `<UseCase>Result` types and every variant above (FR-117), every type named in sections 3, 5 and 6 (except the four internal services) including every variant of each closed type, `HazardKind`, `HazardRecord`, tables (`WaterNeedsPerDay`, `FoodNeedsPerDay`, `ObjectArmorClassTable`, `ObjectHitPointsTable`), `DamageThresholdGate`, `ObjectAffinities`, `CompositeObject`, and `EnvironmentEvent` with its eleven events. `FallResolver`, `HazardDayEnd`, `BreathStateMachine` and `ObjectDamageResolver` stay internal. The published language for downstream contexts is `HazardRecord`, `ObjectStatus`, `DamageThreshold`, `HazardKind` and the events. The package `doc.go` lists inputs and outputs of this slice.

## 7. Observability

The domain is **silent by design** (`doc/rules/observability.md` §3). Use cases are instrumented only through `CombatTelemetry` (FR-099); no log is emitted. The counters `srd5e.combat.operations`, `.events`, `.errors`, histogram `srd5e.combat.operation_duration` and attributes `srd5e.combat.operation`, `srd5e.outcome`, `srd5e.combat.result`, `srd5e.combat.result_class`, `error.type` are shared and owned by `combat-foundations.md` (FR-029 to FR-032). Names are `CombatSignals` constants. For each of its twelve operations this spec adds the pair `<operation>OperationValue` / `<operation>Span` (for example `rollBurningDamageOperationValue`, `rollBurningDamageSpan`) and one catalogue row for the span `srd5e.combat.<operation>` (`combat-foundations.md` FR-037, FR-040).

Signals owned by this spec (recording rules: FR-119):

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.hazard.damage_rolled` | histogram, `{hit_point}` | `srd5e.combat.hazard.kind` (`burning`, `falling`) | Hazard damage handed to the host before the pipeline | p99 above 2x its 7-day baseline; a fall p99 stuck at the 20d6 range suggests unbounded distances |
| `srd5e.combat.hazard.exhaustion_gained` | counter, `{level}` | `srd5e.combat.hazard.kind` (`dehydration`, `malnutrition`, `suffocation`) | Exhaustion levels applied by hazards | Rate by kind; any `exhaustion_at_maximum` result on the shared counter means a hazard keeps applying to a creature that should be dead |

Operations and results (`srd5e.combat.result_class`, `combat-foundations.md` FR-030, FR-031). `refused` records `failure` with no error status; `applied` records `success`; no result is `invalid` (FR-100).

| Operation | applied results | refused results |
|---|---|---|
| `roll_burning_damage` | `damage_rolled` | `not_burning` |
| `extinguish_burning` | `extinguished` | `not_burning`, `blocked_by_immunity` |
| `resolve_fall` | `damage_rolled`, `no_damage` | none |
| `land_after_fall` | `landed_prone`, `landed_standing` | none |
| `resolve_dehydration_day_end`, `resolve_malnutrition_day_end`, `resolve_suffocation_turn_end` | `no_exhaustion`, `exhaustion_gained`, `exhaustion_at_maximum`, `immune`; malnutrition also `save_required` | none |
| `advance_breath` | `advanced` | `not_breathing`, `not_holding_breath`, `already_suffocating` |
| `restore_breathing` | `restored` | none |
| `remove_exhaustion_levels` | `removed` | `blocked` |
| `roll_object_hit_points` | `rolled` | `split_required` |
| `apply_damage_to_object` | `damaged`, `deflected`, `no_damage`, `destroyed` | `already_destroyed` |

Evaluation (`doc/rules/observability.md` §2). **Can go wrong:** the `dice.Roller` fails or returns a wrong result; an operation in the wrong state; a hazard keeps applying Exhaustion at level 6; an `srderr.ErrInvalidInput` error; a wrong rule result (missed half-damage, threshold at the wrong step), so outcomes must be visible. **Operator sees:** refusals on the shared `srd5e.combat.operations` counter by `result_class` = `refused`; port failures on `srd5e.combat.errors`; an `srderr.ErrInvalidInput` error only as a thrown error on the span (`combat-foundations.md` FR-035). **Counted or timed:** operations, results, events and duration on the shared signals; hazard damage rolled and Exhaustion gained by kind (the two signals above). **Diagnosis:** operation, result, hazard kind; rolls and amounts are in the returned result, not attributes. **Existing instrumentation:** still correct; `conditions.md` signals are untouched because hazards call the domain, not the condition use cases (D-1); consistent with `hit-points-and-damage.md` FR-080. **Monitor:** non-zero `srd5e.combat.errors` over 5 minutes; `exhaustion_at_maximum`; a `refused` share above 5% of an operation (a high `already_destroyed` share means the host keeps damaging destroyed objects; a high `deflected` share is normal for thresholds); `save_required` with no follow-up call. **Noise:** none; no per-call `INFO` and no log (`combat-foundations.md` FR-034).

Entries here feed `doc/observability/signals.md`.

## 8. Test plan

Doubles: `dicetest.FixedRoller`, `dicetest.CallbackRoller` and `dicetest.CallbackRoller (returning an error)` (`dice-and-rolls.md`, package `dice/dicetest`) for the three use cases with a port; the recording doubles of `telemetry/telemetrytest`; real `ConditionSet` and `HazardRecord` values. No new port, double or contract suite.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-004 | unit | `HazardKind` order; construction, rejection of negatives and of sums above 6 (including `ContagionLevels`), equality under `==` |
| FR-005 to FR-013, FR-106 | unit, application | Table 4.2a one row per test, real `ConditionSet`s (Prone, Unconscious, immune); fixed rolls 1 to 4; not-burning rejection requests no roll; state stays `burning` after a roll and the `Damage` is unadjusted Fire (FR-007, FR-008); `BurningEnded` once per method after condition events, none on rejection |
| FR-014 to FR-030 | unit, application | Tables 4.3a, 4.3b, 4.3c, 4.4a, 4.4b (integer arithmetic only); fixed roller shows the die count; boundary rows 0, 9, 10, 19, 200, 201 |
| FR-031 to FR-035, FR-110, FR-111 | unit, application | Tables 4.5a and 4.10a rows; a drifted record is clamped (suffocation first) and never panics; level 6 reports `exhaustionGained` and a scenario calls `IsDeadFromExhaustion` |
| FR-036 to FR-045 | unit, application | Table 4.6a; consecutive-day rows run as a sequence of calls |
| FR-046 to FR-052 | unit | Tables 4.7a, 4.7b; capacity as `Duration`; state-machine rejections through `BreathRejection` |
| FR-053 to FR-059, FR-107, FR-108 | unit, application | Tables 4.7c and 4.10a; turn-end outcomes at level 0, 5, 6 and immune; other-source Exhaustion and inconsistent records; `BreathRestored` with `levelsRemoved` 0 and above, none in the FR-058 no-op |
| FR-060 to FR-065, FR-115 | unit, application, scenario | Table 4.8a; the FR-065 scenario runs `ResolveDehydrationDayEnd`, `RemoveExhaustionLevels` (blocked), a full day's water, then removal; `WithContagionLevels` rejection of a negative count and of a sum above 6; clamp order |
| FR-066 to FR-078 | unit, application | Tables 4.9a, 4.9b; FR-068 is a data-integrity test over every row; rolled rows use the fixed roller; construction invariants, standard affinities, destroyed at 0, floor at 0, destroyed rejection, empty damage list |
| FR-079 to FR-086, FR-109, FR-124 | unit, application | FR-124 row: an object whose `DamageAffinities` hold Fire Resistance takes half of Fire damage and full damage of other types; Table 4.9c with the real `hit-points-and-damage.md` resolver, not a double; events column; `ObjectDestroyed`-only break; no event for an all-Immune instance |
| FR-087 to FR-093, FR-114 | unit | Tables 4.9d (six abilities for the no-score case) and 4.9e; a two-section `CompositeObject` stands in for a vehicle; no building or vehicle factory (exported-name scan) |
| FR-094, FR-095 | application | A failing roller's error comes back wrapped once as a `*srderr.PortError` (`errors.Is` reaches the cause); wrong count, die and modifier each give `srderr.ErrContractViolation`; fixed roller shows the single requested expression |
| FR-096 to FR-098 | unit, application | Inputs not mutated; slices copied; equal results for equal fixed rolls; no clock read (architecture scan); parity with `ApplyConditionToSet` and `ReduceExhaustionOnSet` for the same input |
| FR-099 to FR-103, FR-119 | application | Recording telemetry double: span per each of the twelve operations; one `operations`, `operation_duration` and per-event `events` record per result; `result` and `result_class` per section 7; `refused` is `failure` with no error status and no `errors` increment; `*srderr.PortError` counted once on `errors` (`dice_roller_error`), `operations` and `events` unchanged, not logged; identical results with no-op; closed attribute values; `damage_rolled` once with the unadjusted amount (not for `BurningDamageRejected` or `FallWithoutDamage`); `exhaustion_gained` 1 only for `exhaustionGained`; names and key are `CombatSignals` constants |
| FR-104, FR-113, FR-117 | architecture | Export list derived from every type named in sections 3, 5 and 6 including every closed variant (except the four internal services); `...Rejected` variants hold a `...Rejection`; no generic or duplicated exported name in package `combat` (`shared-types-and-ownership.md` FR-017, FR-021; `combat-foundations.md` FR-022 to FR-024) |
| FR-105 | unit, architecture | One closed `EnvironmentEvent` of `Event` (architecture test lists implementers); `kind` constants unique and matching the pattern; `Combatants()` copied, one id for the seven creature events and empty for the four object events; equality including `kind` and `combatants`; |
| FR-112 | unit | `DamageThresholdGate.Evaluate` with no object type: no threshold, below, equal, above, summed instance; same gate used for a creature's threshold |
| FR-116, FR-123, FR-124 | architecture | Import scans (check A1) for the `hazard_*.go` and `object_*.go` domain files (no `telemetry`, no use case) and this spec's `*_usecase.go` files; no spec to the left, and no `character`, `d20` or `content` file, imports them; no `MagicItemResilience`-named type in package `combat`, no import of a `magic_item` or `character` package in the `object_*.go` files; `ApplyDamageToObject` and its command import no `mounted-and-underwater-combat` file and take no `UnderwaterStatus` (FR-124) |
| FR-118 | architecture, application | No use case file imports or constructs another; each runs with only the section 6 ports; one call produces one set of shared signals |
| FR-120 to FR-122 | unit, application | Each `proneApplication` and `reduction` case with real `ConditionSet`s; no nil-able optional field (exported-name and type scan); negative `FallDistance`, `WaterAmount`, `FoodAmount` return an error; `NewHazardRecord` returns the record or each `HazardRecordRejection`, never panics (a `FuzzNewHazardRecord` test) |
| (scenario) | scenario | Public API only: a creature falls 30 feet into water, the host resolves the landing check, passes `Damage` and `HalfDamage` to `ApplyDamage`, lands Prone through `LandAfterFall`; a burning creature rolls on the ground; an object with a Damage Threshold survives a 9 and is destroyed by a 40 |

- **Mutation testing:** the `hazard_*.go` and `object_*.go` files with `gomu`, run only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`, score at least 80 (`doc/rules/testing.md` §8), watching `<` against `<=` at the half, capacity, 10-foot and threshold boundaries, the 5-day and 20-dice limits, and the min/max in FR-060. Every FR id appears above; none is deliberately uncovered.

## 9. Open questions and decisions

Settled: D-1 hazards never call another use case (`shared-types-and-ownership.md` FR-044; FR-098, FR-118). D-2 `HazardExhaustionOutcome` projects the condition result and adds `noExhaustion` (FR-035, FR-045, FR-055). Q-1 to Q-3 rules tables are constants owned here, `CreatureSize` is imported, `CombatantID` is `combat-foundations.md` FR-001 (`shared-types-and-ownership.md` FR-005, FR-012, FR-046, FR-048). Q-4 `DifficultyClass`, `d20.Outcome` (`shared-types-and-ownership.md` FR-002, FR-004). Q-11 the lock is judged on the day's total at day's end (FR-033, FR-037), kept as a count (FR-060), and every removal goes through `RemoveExhaustionLevels` routed by the host (FR-116). Q-13 a vehicle is a `CompositeObject` (FR-114). Q-15 both the lookup (FR-067) and `RollObjectHitPoints` (FR-070). Q-16 one closed `EnvironmentEvent` family (`shared-types-and-ownership.md` FR-019). Q-17 `MalnutritionSaveRequired` is `applied` (FR-100). Q-18 integer eighths (FR-030). Q-19 the ledger sits beside the `ConditionSet` because `Exhaustion(level)` is one cumulative level (`conditions.md` FR-123). Q-20 `ApplyDamage` stays without a threshold (FR-112; `shared-types-and-ownership.md` FR-015, FR-048). Q-23 Occupied Space by objects is a host fact (`initiative-and-actions.md` FR-081).

Settled by SRD text and research (all closed; none escalated). Where the SRD is silent the reading is an interpretation, named as such. Q-5 and Q-7: "1d6 ... for every 10 feet it fell" (p. 182) counts whole 10-foot steps and "Round Down" (p. 187) forbids rounding up, so 9 feet is no damage and no Prone, 19 feet is 1d6 (FR-015, FR-017; interpretation, the usual reading of the rule). Q-6: the final amount after the pipeline and before Temporary Hit Points decides Prone, because Temporary Hit Points absorb damage that is still taken (p. 18) while Immunity and Resistance stop it being dealt (FR-024; interpretation). Q-8: Prone cannot be given to a creature Immune to it, so rolling fails and the host picks another method (p. 178; "Immunity", p. 183; FR-011). Q-9: p. 185 is explicit: the save is for a creature that "eats but" consumes under half; "eats nothing for 5 days" gains a level automatically at the end of the fifth day and one more each following day; the day-5 level replaces the save; "5 days" is consecutive, any food ends the run (FR-042 to FR-044; the run reading is an interpretation). Q-10: breath is held "before suffocation begins" (p. 189), so suffocation begins when the held time reaches the capacity (FR-049; interpretation). Q-12: the test compares the incoming amounts of the whole instance ("a single attack or effect", p. 180) before Resistance or Vulnerability, which apply to damage the object "takes" and so to the entire instance once the test is met (FR-083, FR-084; interpretation: the SRD gives no order and published commentary is split, the chosen order keeps the gate independent of affinities, FR-112). Q-14: a burning object has no turn, so the host chooses when to call `RollBurningDamage` and passes the `Damage` to `ApplyDamageToObject` (p. 178; package design, the engine reads no clock). Q-21: the threshold lifts only the "Immunity to all damage" it grants (p. 180); a type Immunity (p. 183, FR-073) is a separate rule and still applies (FR-085). Q-22: automatic breaking deals no damage ("automatically break", pp. 12, 177), so no threshold is consulted (FR-086). Q-24: a specific effect ("lasts until the contagion ends", Cackle Fever, p. 194) outranks the general Long Rest removal (p. 185), so the level is locked (FR-115; matches `toolbox-curses-and-contagions.md` HB-1). Q-25: the SRD gives objects Hit Points (p. 183) and no Temporary Hit Points, so an object holds none (FR-072; package design). Q-26: underwater objects (p. 16 "Anything underwater") get Fire Resistance by the host passing affinities that include it, because the rank-24 owner of the rule cannot be imported at rank 23 (FR-124; package design).

## 10. Changelog impact

`CHANGELOG.md`, under Unreleased, **Added**: hazard rules (Burning, Falling, Dehydration, Malnutrition, Suffocation) and object rules (Object Armor Class, Object Hit Points, Damage Threshold, composite objects) in `combat`, with the twelve use cases (each with its `<UseCase>Command` and closed `<UseCase>Result`), their value objects, the `EnvironmentEvent` family (eleven events), the `HazardRecord` ledger (including `ContagionLevels`) and the Exhaustion lock, and the `srd5e.combat.hazard.damage_rolled` and `srd5e.combat.hazard.exhaustion_gained` signals. Not **BREAKING**: new API only.
