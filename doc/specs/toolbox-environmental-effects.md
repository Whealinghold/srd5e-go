# Toolbox: Environmental Effects

| | |
|---|---|
| Status | draft |
| Context | `toolbox` |
| Roadmap phase | 5 |
| Depends on | `toolbox-hazard-effects.md`, `d20-tests.md`, `ability-scores.md`, `dice-and-rolls.md`, `hit-points-and-damage.md`, `combat-foundations.md`, `content-schema-and-decoder.md`, `content-equipment.md`, `observability-ports.md`, `errors-and-results.md` |

Roadmap: [../roadmap.md](../roadmap.md). Context and phase: shared-types-and-ownership.md
FR-014, FR-045 (`toolbox` is downstream of `content`, `d20`, `character` and
`combat`; this slice uses no `character` type; map row: `toolbox-hazard-effects.md`
FR-001, FR-002, FR-005, FR-006). Types come from: `DifficultyClass` (`d20-tests.md`); `Ability`,
`ability.Score`; `Roller`, `dice.Roll`, `dice.Result`, dice port error;
`DamageAffinities` (`hit-points-and-damage.md` FR-091); `ConditionKind`
(`content-schema-and-decoder.md` FR-110); `ConditionSet.apply` is the host's
(`conditions.md` FR-123); `CombatantID`
(`combat-foundations.md` FR-001, `combat` published API); `Skill` and `Obscurement`
(`content-schema-and-decoder.md` FR-106, FR-135; shared-types-and-ownership.md
FR-053); `ArmorCategory` (`content-equipment.md` FR-104). Cited in prose only, not
dependencies: `conditions.md`, `modifiers-and-advantage.md`, `toolbox-travel-pace.md` (rank 41, the consumer of the High Altitude
result, its FR-030 to FR-032; this spec never imports from it),
`environment-hazards-and-objects.md` (Falling), `hiding-searching-and-senses.md`
(`SpaceVisibility.otherObscurement`, FR-007), `exploration.md` (Q-06), `attacks.md`,
`initiative-and-actions.md`.

## 1. Purpose and scope

The nine entries of "Environmental Effects" (SRD 5.2.1, p. 195): Deep Water,
Extreme Cold, Extreme Heat, Frigid Water, Heavy Precipitation, High Altitude,
Slippery Ice, Strong Wind and Thin Ice.

Most entries are "succeed on a saving throw or gain 1 Exhaustion level" (or the
Prone condition): the `SaveGatedEffect` of `toolbox-hazard-effects.md` (FR-008),
which also owns automatic success for Resistance or Immunity (FR-024 to FR-027),
the single Disadvantage for worn armor (FR-028, FR-029), the planner result
`AutoSucceeded` or `SaveRequired` (FR-030 to FR-032) and the subject exemption
(FR-076), and forbids a second effect model. This spec supplies only what is
**specific to each effect**: the preconditions (depth over 100 feet, minutes past
the Constitution score, temperature bands, trigger kind), the Extreme Heat DC
(`4 + hour`), why no save is called for, the Swim Speed and ice-cold-water
exemptions (decided here, passed on as `SaveGatedExempt`), and the standing effects
of an area. It builds a `SaveGatedEffect` per case that needs a save and hands it
over. It owns High Altitude in full (shared-types-and-ownership.md FR-048: whether
an hour is a high-altitude hour, acclimation, the doubling; `toolbox-travel-pace.md`
FR-031 takes both hours values and never doubles) and the one thing that needs
dice, the Thin Ice weight tolerance.

The engine follows "state in, new state and events out" (`doc/rules/architecture.md`
§4). The host decides **when** things happen and which facts hold; they arrive as
typed values and the engine reads no clock. The host passes the handoff to
`ResolveSaveGatedEffect`, rolls the save through `ResolveD20Test`, and applies a
failed save's Exhaustion or Prone through `ConditionSet.apply` (`conditions.md`
FR-123; `toolbox-hazard-effects.md` FR-044, FR-046). No use case here changes a
`ConditionSet` or calls another use case (shared-types-and-ownership.md FR-044).

**Out of scope:**

- Applying a failed save's consequence (Exhaustion, Prone, Immunity, an
  already-held condition, level 6): the host, through `conditions.md` (FR-084).
- Automatic success and Disadvantage logic: `toolbox-hazard-effects.md` FR-024 to FR-032.
- Travel Pace, distance, Extended Travel (p. 192): `toolbox-travel-pace.md`.
- Dehydration and Water Needs (`environment-hazards-and-objects.md`); "without
  access to drinkable water" is a host-supplied fact (FR-019).
- Falling of a flying creature in strong wind (`environment-hazards-and-objects.md`
  FR-014 to FR-026); movement rules (`initiative-and-actions.md`; FR-054 only
  names Difficult Terrain); applying Lightly Obscured or Disadvantage to a roll
  (`hiding-searching-and-senses.md`, `attacks.md`): reported as data.
- Whether a creature is "exposed" (no SRD rule; Q-9); weather generation,
  temperatures by region, creature weights (host-supplied); falling through broken
  ice beyond "frigid water" (FR-063); which creatures are "naturally adapted to
  living in ice-cold water" (no SRD list; host-supplied, Q-10; nothing added to `content`).
- Anything not in SRD 5.2.1. Nothing on p. 195 was left out.

## 2. Source

PDF page numbers equal printed page numbers.

- `SRD 5.2.1, "Environmental Effects"`: "Deep Water", "Extreme Cold", "Extreme
  Heat", "Frigid Water", "Heavy Precipitation", "High Altitude" (cross-reference
  "Travel Pace", p. 192), "Slippery Ice", "Strong Wind", "Thin Ice", all p. 195
- `SRD 5.2.1, "Exhaustion [Condition]", p. 181`; "Prone [Condition]", p. 186;
  "Difficult Terrain", p. 181; "Lightly Obscured", p. 184; "Swim Speed", p. 189;
  "Saving Throw", p. 187; "Resistance", p. 187; "Immunity", p. 183; "Damage
  Types", p. 180; "Ability Scores", p. 5; "Difficulty Class", p. 6; "Thrown"
  (Weapon Properties), p. 90; "Advantage/Disadvantage", "They Don't Stack", p. 8
- Package rules: `doc/rules/architecture.md` (§2 to §6), `doc/rules/observability.md`
  (§2 to §7), `doc/rules/testing.md` (§3 to §5).

## 3. Vocabulary

"Package name" marks a descriptive name with no SRD source (adds no rule).

| SRD term | Type | Notes |
|---|---|---|
| Environmental Effects | `EnvironmentalEffectKind` | Nine values in SRD order. |
| degrees Fahrenheit | `Temperature`; `TemperatureBand` (`extremeCold`, `notExtreme`, `extremeHeat`) | "0 degrees Fahrenheit or lower", "100 degrees Fahrenheit or higher". `notExtreme` is a package name. |
| depth, altitude, host counts | `Feet` (0 or more), `ElevationFeet` (signed), `ExposureHour`, `ImmersedMinutes`, `HoursAtAltitude`, `DaysAtElevation`, `PoundsOfWeight` | Package names; no bare `int` crosses the boundary (FR-078). |
| Swim Speed | `SwimCapability` (`hasSwimSpeed`, `lacksSwimSpeed`) | "a creature that lacks a Swim Speed". |
| "naturally adapted to living in ice-cold water" | `IceColdWaterAdaptation` (`adapted`, `notAdapted`) | Host-supplied (Q-10). |
| "access to drinkable water" | `DrinkableWaterAccess` (`available`, `unavailable`) | Host-supplied. |
| Medium or Heavy armor | `ArmorCategory` | Shield is a fourth value and does not count. |
| (the creature assessed) | `CreatureEnvironmentProfile` | Package name: Constitution `ability.Score`, `SwimCapability`, `DamageAffinities`, `IceColdWaterAdaptation`, worn `ArmorCategory` set. |
| (one exposure) | `EnvironmentalExposure` (closed interface) | Five variants (FR-066). Package name. |
| "succeed on a ... saving throw or gain 1 Exhaustion level" | `SaveGatedEffect`, `SaveGatedSubject`, `SaveGatedRollPlan`, `SaveGatedExhaustionGain`, `SaveGatedCondition`, `SaveGatedExemption` | Defined in `toolbox-hazard-effects.md` (FR-008, FR-076); used, not redefined. |
| (no save is called for) | `EnvironmentalNoSaveReason` (`notDeepWater`, `notExtremeCold`, `notExtremeHeat`, `drinkableWaterAvailable`, `withinConstitutionMinutes`, `alreadyEnteredThisTurn`) | Unmet preconditions only; trait exemptions are `SaveGatedExempt` (FR-089). Distinct from the retired `NoSaveReason` of the hazard spec. Snake_case is the `srd5e.toolbox.environment.reason` value. |
| (why a value was rejected) | `EnvironmentalValueInvalidReason` (`negative` for `Feet`, `DaysAtElevation`, `PoundsOfWeight`; `belowOne` for `ExposureHour`, `ImmersedMinutes`, `HoursAtAltitude`, `ThinIceTolerance`), `ThinIceLoadInvalidReason` (`duplicateOccupant`) | Package names (FR-078). |
| High Altitude "counts as 2 hours" | `AltitudeTravelBasis` (`belowHighAltitude`, `notAcclimated`, `acclimated`), `AltitudeNativity` (`native`, `notNative`) | "acclimated ... by spending 30 days or more at this elevation". Package names. |
| "3d10 x 10 pounds" per 10-foot-square area | `ThinIceTolerance`, `ThinIceLoad`, `ThinIceOccupant(combatantID, weight)` | One tolerance per area. Package names. |
| Heavy rain, heavy snowfall; sandstorm | `PrecipitationKind` (`heavyRain`, `heavySnowfall`), `StrongWindSetting` (`open`, `desertSandstorm`) | The sandstorm is optional ("can"), so the host chooses. |
| Lightly Obscured | `AreaObscurement(Obscurement.lightly)` | The host supplies the value as `SpaceVisibility.otherObscurement` (hiding spec FR-007; `exploration.md` Q-06). |
| Wisdom (Perception); Difficult Terrain | `Skill` (carries its `Ability`, shared-types-and-ownership.md FR-005); `DifficultTerrain` | Movement cost is `initiative-and-actions.md`. |
| (standing effect of an area) | `AreaEffect` (closed interface) | Package name; see 4.9. |

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. A requirement replaced
by a citation keeps its id in "Withdrawn ids" at the end of this section.

Shorthand in case lists (all p. 195):

- "Save due: Con, DC 10, Exhaustion 1" means the use case returns
  `AssessEnvironmentalExposureSaveGated(effectKind, effect, subject)` where
  `effect` has the shown ability and DC, instantaneous duration, no repeat, no
  effect on success, healing `healable`, and a failure of one
  `SaveGatedExhaustionGain(1, untilRemoved)` (Slippery Ice: a `SaveGatedCondition`
  of the Prone `ConditionKind`, margin `SaveGatedAnyFailure`). Non-empty
  automatic-success and Disadvantage sets are shown (FR-009).
- "No save" means `AssessEnvironmentalExposureNoSave(effectKind, reason)`.
- "then Auto": after the host passes the handoff to `ResolveSaveGatedEffect`, it
  returns `AutoSucceeded` (`toolbox-hazard-effects.md` FR-024); an application test
  across both use cases. `AutoSucceeded`, `SaveRequired`, `Failed` abbreviate the
  `ResolveSaveGatedEffect` variants of the hazard spec (FR-024, FR-030).
- Each listed case is one table-driven test: label = inputs, requirement = cited FR,
  source p. 195 unless stated, path = that FR's Path (first case of a list: happy).

No question is open; section 9 records every decision.

### 4.1 Catalogue, temperature and bands

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `EnvironmentalEffectKind` MUST have exactly nine values, in this order: Deep Water, Extreme Cold, Extreme Heat, Frigid Water, Heavy Precipitation, High Altitude, Slippery Ice, Strong Wind, Thin Ice. | SRD 5.2.1, "Environmental Effects", p. 195 | happy |
| FR-002 | `Temperature` MUST accept any whole number of degrees Fahrenheit (negative, or far above 100) and be comparable (`==`) for equal values. `NewTemperature(degrees int) Temperature` MUST always succeed and return no error; no invalid temperature exists (FR-078). | SRD 5.2.1, "Extreme Cold", "Extreme Heat", p. 195 (no bound) | unhappy (boundary) |
| FR-003 | The band of a `Temperature` MUST be `extremeCold` at 0 or lower, `extremeHeat` at 100 or higher, and `notExtreme` from 1 to 99. | SRD 5.2.1, "Extreme Cold", "Extreme Heat", p. 195 | unhappy (boundary) |

**4.1a Temperature bands** (FR-003), one test per degrees value: -40, -1, 0 give
`extremeCold`; 1, 68, 99 give `notExtreme`; 100, 101, 400 give `extremeHeat`.

### 4.2 The creature profile and the save-gated handoff

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-004 | `CreatureEnvironmentProfile` MUST carry the Constitution `ability.Score`, `SwimCapability`, `DamageAffinities`, `IceColdWaterAdaptation` and the set of worn `ArmorCategory` values, and expose each. An empty armor set MUST be valid. | SRD 5.2.1, "Deep Water", "Extreme Heat", "Frigid Water", p. 195 | happy |
| FR-005 | `CreatureEnvironmentProfile` MUST copy its armor set, expose it defensively copied, and be equal to a profile with equal fields whatever the armor order supplied. | `doc/rules/architecture.md` §3, §4; `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-009 | The `SaveGatedEffect` built per case MUST be: Constitution for Deep Water, Extreme Cold, Frigid Water, Extreme Heat; Dexterity for Slippery Ice; automatic-success set `{cold}` for Extreme Cold and Frigid Water, `{fire}` for Extreme Heat, empty otherwise (so a Cold affinity never exempts Fire and the reverse, hazard FR-026); Disadvantage set `{wearingMediumOrHeavyArmor}` for Extreme Heat only; onset immediate; success no effect; no repeat; instantaneous; failure one `SaveGatedExhaustionGain(1, untilRemoved)` (Slippery Ice: `SaveGatedCondition` Prone, margin `SaveGatedAnyFailure`); healing `SaveGatedDamageHealing.healable`. It has exactly the fields of `toolbox-hazard-effects.md` FR-008 and no kind: the `EnvironmentalEffectKind` rides on the result variant beside it. The `SaveGatedSubject` carries the `DamageAffinities` unchanged, the circumstances of FR-022 and the exemption of FR-089. | SRD 5.2.1, "Saving Throw", p. 187; "Environmental Effects", p. 195; `toolbox-hazard-effects.md` FR-008, RE-E, RE-F | happy |
| FR-080 | Two `AssessEnvironmentalExposureSaveGated` results from equal inputs MUST be equal, and the handoff MUST NOT carry a hand-made save, DC or circumstance list beyond FR-009. | `doc/rules/architecture.md` §3, §4 | happy |

### 4.3 Deep Water

`DeepWaterExposure(depth: Feet)` is called at the end of each hour a creature
spends swimming; the rule is the same every hour, so there is no hour count.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | Water is deep only when **more than** 100 feet deep: 100 feet or less MUST give No save with `notDeepWater`; 101 feet or more MUST proceed to FR-011 and FR-012. | SRD 5.2.1, "Deep Water", p. 195 | unhappy (boundary) |
| FR-011 | In deep water a creature with `lacksSwimSpeed` MUST get a save due: Constitution, `DifficultyClass(10)`, no roll circumstances, Exhaustion 1. | SRD 5.2.1, "Deep Water", p. 195 ("a DC 10 Constitution saving throw or gain 1 Exhaustion level") | happy |
| FR-012 | In deep water a creature with `hasSwimSpeed` MUST still receive the handoff of FR-011, with subject exemption `SaveGatedExempt` (FR-089), so `ResolveSaveGatedEffect` returns `AutoSucceeded` with no roll (`toolbox-hazard-effects.md` FR-076). | SRD 5.2.1, "Deep Water", p. 195 ("a creature that lacks a Swim Speed") | unhappy (rule interaction) |
| FR-013 | `NewFeet` MUST accept 0 and more and return an error wrapping `srderr.ErrInvalidInput` for a negative depth, an `*EnvironmentalValueError` with reason `negative`; `MustFeet` panics with that error and neither panics otherwise (FR-078). | `doc/rules/testing.md` §4; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-014 | When the water is not deep the result MUST be `notDeepWater` whether or not the creature has a Swim Speed (depth is judged first). | SRD 5.2.1, "Deep Water", p. 195 | unhappy (rule interaction) |

**4.3a Deep Water cases** (FR-010 to FR-014, FR-089), one call each (depth in feet):

- lacks Swim Speed: 0, 50, 100 give No save `notDeepWater` (FR-010, FR-013); 101
  and 400 give save due: Con, DC 10, Exhaustion 1 (FR-010, FR-011).
- has Swim Speed: 400 gives save due with subject `SaveGatedExempt`, then Auto
  (FR-012, FR-089); 100 gives No save `notDeepWater` (FR-014).
- lacks Swim Speed, depth 400, with Immunity Cold, and separately with Immunity
  Fire: save due with an empty automatic-success set; then `SaveRequired` (FR-009).

### 4.4 Extreme Cold

`ExtremeColdExposure(temperature)` is called at the end of each hour of exposure.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-015 | A temperature above 0 (band `notExtreme` or `extremeHeat`) MUST give No save with `notExtremeCold`. | SRD 5.2.1, "Extreme Cold", p. 195 | unhappy (boundary) |
| FR-016 | At 0 or lower, whatever the affinities, the result MUST be a save due: Constitution, `DifficultyClass(10)`, no roll circumstances, Exhaustion 1. | SRD 5.2.1, "Extreme Cold", p. 195 ("a DC 10 Constitution saving throw at the end of each hour or gain 1 Exhaustion level") | happy |
| FR-017 | At 0 or lower, a creature with Resistance or Immunity to Cold MUST still receive the handoff of FR-016 (set `{cold}`); `ResolveSaveGatedEffect` then returns `AutoSucceeded` with no roll (`toolbox-hazard-effects.md` FR-024). This fixes only that the handoff reaches it. | SRD 5.2.1, "Extreme Cold", p. 195 ("automatically succeed on the save") | unhappy (rule interaction) |

**4.4a Extreme Cold cases** (FR-015 to FR-017), one call each (temperature, affinities):

- 0 and -60, none: save due: Con, DC 10, Exhaustion 1 (FR-016).
- 1 and 120, none: No save `notExtremeCold` (FR-015); 1 with Immunity Cold: the same (exempt but not cold enough).
- -10 with Resistance Cold, and with Immunity Cold: save due (set `{cold}`); then Auto (FR-017).
- -10 with Immunity Fire: save due; planner returns `SaveRequired`, not Auto (hazard FR-026; FR-009).

### 4.5 Extreme Heat

`ExtremeHeatExposure(temperature, hour: ExposureHour, water)` (worn armor comes
from the profile) is called at the end of each hour of exposure; `hour` counts
from 1, kept by the host (Q-4).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | A temperature below 100 (band `notExtreme` or `extremeCold`) MUST give No save with `notExtremeHeat`. | SRD 5.2.1, "Extreme Heat", p. 195 | unhappy (boundary) |
| FR-019 | At 100 or higher, a creature with `DrinkableWaterAccess.available` MUST get No save with `drinkableWaterAvailable` (the save is for a creature "without access to drinkable water"). | SRD 5.2.1, "Extreme Heat", p. 195 | unhappy (rule interaction) |
| FR-020 | The DC MUST be `4 + hour`: 5 for hour 1, 1 higher for each additional hour, with no upper limit. | SRD 5.2.1, "Extreme Heat", p. 195 ("The DC is 5 for the first hour and increases by 1 for each additional hour"); "Difficulty Class", p. 6 | unhappy (boundary) |
| FR-021 | `NewExposureHour` MUST accept 1 or more and return an error wrapping `srderr.ErrInvalidInput` below 1 (`*EnvironmentalValueError`, reason `belowOne`); `MustExposureHour` panics with that error and neither panics otherwise (FR-078). | `doc/rules/testing.md` §4; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-022 | The `SaveGatedSubject` built from a profile MUST carry `wearingMediumOrHeavyArmor` when the worn armor contains Medium or Heavy, and not when it holds only Light, only Shield, Light and Shield, or nothing. The single Disadvantage then comes from the hazard planner (`toolbox-hazard-effects.md` FR-028, FR-029). | SRD 5.2.1, "Extreme Heat", p. 195 ("Creatures wearing Medium or Heavy armor have Disadvantage on the save") | unhappy (rule interaction) |
| FR-023 | The Extreme Heat effect MUST list `fire` in its automatic-success set whatever armor is worn, so `ResolveSaveGatedEffect` returns `AutoSucceeded` for Resistance or Immunity to Fire, the Disadvantage not reported (`toolbox-hazard-effects.md` FR-024, FR-032). | SRD 5.2.1, "Extreme Heat", p. 195 ("automatically succeed on the save") | unhappy (rule interaction) |
| FR-024 | Checks MUST run in this order, stopping at the first that decides: (1) band not `extremeHeat`, (2) drinkable water available, (3) save due. The Fire exemption is decided after the handoff (FR-023). | SRD 5.2.1, "Extreme Heat", p. 195; package design for the order | unhappy (rule interaction) |
| FR-025 | The save due MUST use Constitution with consequence Exhaustion 1. | SRD 5.2.1, "Extreme Heat", p. 195 | happy |

**4.5a DC by hour** (FR-020, FR-025): temperature 100, no water, no armor, no
affinities gives save due: Con, Exhaustion 1, with hour to DC: 1 to 5, 2 to 6, 3 to
7, 10 to 14, 26 to 30, 27 to 31 (beyond the typical range), 100 to 104.

**4.5b Armor** (FR-022): hour 1, temperature 100, no water. Subject carries
`wearingMediumOrHeavyArmor` (planned roll has Disadvantage) for Medium, Heavy, and
Medium with Shield; it carries nothing (no Disadvantage) for nothing worn, Light,
Shield only, and Light with Shield.

**4.5c Precedence** (FR-018 to FR-024), one call each (temp, water, affinities, armor, hour):

- 100, unavailable, none, none, 1: save due, DC 5, no Disadvantage (FR-020, FR-024).
- 99, and -20 (cold is not heat), unavailable, none, none, 1: No save `notExtremeHeat` (FR-018).
- 110, available, none, Heavy, 4: No save `drinkableWaterAvailable` (FR-019); with Immunity Fire, none, 4: the same, water beats exemption (FR-024).
- 110, unavailable, Resistance Fire, none, 4: save due (set `{fire}`); then Auto (FR-023).
- 110, unavailable, Immunity Fire, Heavy, 4: save due; then Auto, no Disadvantage reported (FR-023).
- 110, unavailable, Immunity Cold, none, 4: save due, DC 8; then `SaveRequired`, no Disadvantage (FR-009, FR-024).
- 110, unavailable, none, Medium, 4: save due, DC 8, Disadvantage (FR-020, FR-022).

### 4.6 Frigid Water

`FrigidWaterExposure(minutesImmersed: ImmersedMinutes)` is called at the end of
each minute immersed; the count starts at 1 (Q-4). A creature can be immersed "for a
number of minutes equal to its Constitution score before suffering any ill effects",
so the first save is due at minute `score + 1`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-026 | When `minutesImmersed` is **at most** the Constitution score the result MUST be No save with `withinConstitutionMinutes`, whatever the Resistance, Immunity or adaptation (judged before the handoff). | SRD 5.2.1, "Frigid Water", p. 195 | unhappy (boundary) |
| FR-027 | When `minutesImmersed` is **more than** the Constitution score the result MUST be a save due (set `{cold}`): Constitution, `DifficultyClass(10)`, no roll circumstances, Exhaustion 1. | SRD 5.2.1, "Frigid Water", p. 195 ("Each additional minute ... a DC 10 Constitution saving throw or gain 1 Exhaustion level") | happy |
| FR-028 | `NewImmersedMinutes` MUST accept 1 or more and return an error wrapping `srderr.ErrInvalidInput` below 1 (`*EnvironmentalValueError`, reason `belowOne`); `MustImmersedMinutes` panics with that error and neither panics otherwise (FR-078). | `doc/rules/testing.md` §4; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-029 | After the Constitution-score minutes, a creature with Resistance or Immunity to Cold MUST still receive the handoff of FR-027; `ResolveSaveGatedEffect` then returns `AutoSucceeded` (`toolbox-hazard-effects.md` FR-024). | SRD 5.2.1, "Frigid Water", p. 195 ("Creatures with Resistance or Immunity to Cold damage automatically succeed on the save") | unhappy (rule interaction) |
| FR-030 | After the Constitution-score minutes, a creature with `IceColdWaterAdaptation.adapted` MUST receive the handoff of FR-027 with subject exemption `SaveGatedExempt` (FR-089), whatever its affinities, so `ResolveSaveGatedEffect` returns `AutoSucceeded` with no roll (`toolbox-hazard-effects.md` FR-076). | SRD 5.2.1, "Frigid Water", p. 195 ("as do creatures that are naturally adapted to living in ice-cold water") | unhappy (rule interaction) |
| FR-032 | The grace period MUST use the creature's own Constitution score for every score 1 to 30: the first save is due at minute `score + 1`. The range is enforced by `ability.Score`. | SRD 5.2.1, "Frigid Water", "Ability Scores", pp. 5, 195 | unhappy (boundary) |

**4.6a Frigid Water cases** (FR-026 to FR-032, FR-089), one call each (Con, minute;
not adapted, no affinities unless stated):

- No save `withinConstitutionMinutes`: (10, 1), (10, 10), (1, 1), (30, 30) (FR-026, FR-032); (10, 5) with Immunity Cold; (10, 5) adapted (FR-026).
- Save due: Con, DC 10, Exhaustion 1: (10, 11), (10, 25) (FR-027); (1, 2), (30, 31) (FR-032).
- (10, 11) with Resistance Cold, and with Immunity Cold: save due (set `{cold}`); then Auto (FR-029).
- (10, 11) adapted, and adapted with Resistance Cold: save due (subject `SaveGatedExempt`); then Auto (FR-030, FR-089).
- (10, 11) not adapted: save due (subject `SaveGatedNoExemption`) (FR-089).
- (10, 11) with Immunity Fire: save due; planner returns `SaveRequired` (FR-009, FR-027).

### 4.7 High Altitude

`AssessHighAltitude` with `AssessHighAltitudeCommand(altitude: ElevationFeet,
daysAtElevation: DaysAtElevation, nativity: AltitudeNativity, hours:
HoursAtAltitude)` says whether the hours a creature travels at one elevation are
**high-altitude hours** and how long they count for travel. `altitude` is whole
feet above sea level (any integer). The host counts `daysAtElevation` (Q-6) and
calls once per stretch, summing results. The result is
`AssessHighAltitudeAssessed(basis, countsAsHighAltitude, actualHours,
effectiveHours)`. The factor 2 exists only here (shared-types-and-ownership.md
FR-048). No save is called for.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-033 | Below 10,000 feet the basis MUST be `belowHighAltitude` and `countsAsHighAltitude` false. | SRD 5.2.1, "High Altitude", p. 195 ("altitudes of 10,000 feet or higher") | unhappy (boundary) |
| FR-034 | At 10,000 feet or higher, a creature not acclimated MUST have basis `notAcclimated` and `countsAsHighAltitude` true. | SRD 5.2.1, "High Altitude", p. 195 ("Each hour ... counts as 2 hours for the purpose of determining how long that creature can travel") | happy |
| FR-035 | From 10,000 up to and including 20,000 feet, a creature with 30 days or more at the elevation MUST be acclimated (basis `acclimated`, `countsAsHighAltitude` false). 29 days MUST NOT suffice. | SRD 5.2.1, "High Altitude", p. 195 ("by spending 30 days or more at this elevation") | unhappy (boundary) |
| FR-036 | Above 20,000 feet, a creature with `AltitudeNativity.notNative` MUST NOT become acclimated whatever its days: basis `notAcclimated`, `countsAsHighAltitude` true. | SRD 5.2.1, "High Altitude", p. 195 ("Creatures can't become acclimated to elevations above 20,000 feet unless they are native to such environments") | unhappy (boundary) |
| FR-037 | Above 20,000 feet, a creature with `AltitudeNativity.native` MUST be acclimated only with 30 days or more (the exception lifts the prohibition, not the 30 days; Q-5). | SRD 5.2.1, "High Altitude", p. 195 | unhappy (boundary) |
| FR-090 | `NewHoursAtAltitude` MUST accept 1 or more whole hours and return an error wrapping `srderr.ErrInvalidInput` below 1 (`*EnvironmentalValueError`, reason `belowOne`); `MustHoursAtAltitude` panics with that error and neither panics otherwise (FR-078). No upper bound here; the 24-hour limit is checked in `toolbox-travel-pace.md`. | `doc/rules/testing.md` §4; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-091 | `actualHours` MUST equal the `hours` supplied, and `effectiveHours` MUST be twice `hours` when `countsAsHighAltitude` is true and equal to `hours` otherwise. The factor 2 appears only in this rule. | SRD 5.2.1, "High Altitude", p. 195 | unhappy (boundary) |
| FR-039 | `NewDaysAtElevation` MUST accept 0 and more and return an error wrapping `srderr.ErrInvalidInput` for a negative count (`*EnvironmentalValueError`, reason `negative`); `MustDaysAtElevation` panics with that error and neither panics otherwise (FR-078). | `doc/rules/testing.md` §4; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-040 | A negative altitude (below sea level) MUST be accepted and give basis `belowHighAltitude`; `ElevationFeet` accepts every integer. | SRD 5.2.1, "High Altitude", p. 195 ("above sea level") | unhappy (boundary) |
| FR-041 | The result MUST carry no save requirement and no consequence, only `actualHours` and `effectiveHours`. The Exhaustion that long travel can cause is the Travel Pace rule (p. 192), not applied here. | SRD 5.2.1, "High Altitude", p. 195; "Travel Pace", p. 192 | unhappy (rule interaction) |

**4.7a High Altitude cases** (FR-033 to FR-041), one call each with `hours` 1
(altitude ft, days, nativity gives basis, counts as high altitude):

| Altitude, days, nativity | Basis | Counts | Req. |
|---|---|---|---|
| 0, 0, not native; 9,999, 0, not native; -200, 0, not native | `belowHighAltitude` | no | FR-033, FR-040 |
| 10,000, 0, not native; 12,000, 0, not native; 12,000, 29, not native | `notAcclimated` | yes | FR-034, FR-035 |
| 12,000, 30; 12,000, 400; 20,000, 30 (all not native) | `acclimated` | no | FR-035 |
| 20,001, 30, not native; 25,000, 1,000, not native | `notAcclimated` | yes | FR-036 |
| 25,000, 29, native | `notAcclimated` | yes | FR-037 |
| 25,000, 30, native | `acclimated` | no | FR-037 |

**4.7b Effective hours** (FR-090, FR-091), `actualHours, effectiveHours`:
12,000 ft, 0 days, not native: 1 hour gives 1, 2; 8 hours gives 8, 16. 12,000 ft,
30 days, 8 hours gives 8, 8. 9,999 ft, 8 hours gives 8, 8. 20,001 ft, 400 days, not
native, 3 hours gives 3, 6. 25,000 ft, 30 days, native, 3 hours gives 3, 3.

### 4.8 Slippery Ice

`SlipperyIceExposure(trigger: SlipperyIceTrigger)` is called when a creature moves
onto slippery ice or starts its turn there. `SlipperyIceTrigger` is
`enteredFirstTimeThisTurn`, `enteredAgainThisTurn`, `startedTurnOnIce`; the host
decides which applies (Q-12).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-042 | For `enteredFirstTimeThisTurn` the result MUST be a save due: Dexterity, `DifficultyClass(10)`, no roll circumstances, consequence the Prone condition. | SRD 5.2.1, "Slippery Ice", p. 195 ("moves onto slippery ice for the first time on a turn ... a DC 10 Dexterity saving throw or have the Prone condition"); "Prone", p. 186 | happy |
| FR-043 | For `startedTurnOnIce` the result MUST be the same save due as FR-042. | SRD 5.2.1, "Slippery Ice", p. 195 ("or starts its turn there") | happy |
| FR-044 | For `enteredAgainThisTurn` the result MUST be No save with `alreadyEnteredThisTurn`. | SRD 5.2.1, "Slippery Ice", p. 195 ("for the first time on a turn") | unhappy (rule interaction) |
| FR-045 | Slippery Ice MUST carry no exemption (empty automatic-success and Disadvantage sets). A creature with Resistance or Immunity to Cold or any other type (case: Immunity Cold), a Swim Speed, or any worn armor (case: Heavy, `startedTurnOnIce`) MUST still get the save due of FR-042. | SRD 5.2.1, "Slippery Ice", p. 195 (no exemption stated) | unhappy (rule interaction) |

### 4.9 Standing area effects

`DescribeEnvironmentalAreaEffects` with `DescribeEnvironmentalAreaEffectsCommand(source)`
returns the effects of an **area** as data. `AreaEffectSource` is a closed interface with
exactly `HeavyPrecipitationSource(kind)`, `StrongWindSource(setting)` and
`SlipperyIceSource` (the other six effects have no standing area effect).
`AreaEffect` is a closed interface: `AreaObscurement(obscurement)` (always
`Obscurement.lightly` here), `DisadvantageOnChecks(skill)`,
`DisadvantageOnRangedWeaponAttackRolls`, `ExtinguishesOpenFlames`, `DispersesFog`,
`FlyingCreaturesMustLandAtEndOfTurn`, `DifficultTerrain`. The host maps
`AreaObscurement` to `SpaceVisibility.otherObscurement`; the engine applies it to no roll.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-046 | `AreaEffectSource` MUST have exactly the three variants above, so a kind with no standing area effect cannot be passed. | SRD 5.2.1, "Environmental Effects", p. 195 | happy |
| FR-047 | The described effects MUST be an defensively copied list, each effect once, in the order listed in FR-048, FR-049, FR-051, FR-052 and FR-054. | `doc/rules/architecture.md` §4; `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-048 | For `heavyRain` the effects, in order, MUST be `AreaObscurement(Obscurement.lightly)`, `DisadvantageOnChecks(Skill.perception)`, `ExtinguishesOpenFlames`. | SRD 5.2.1, "Heavy Precipitation", p. 195 ("Everything within an area ... is Lightly Obscured ... Disadvantage on all Wisdom (Perception) checks. Heavy rain also extinguishes open flames.") | happy |
| FR-049 | For `heavySnowfall` the effects, in order, MUST be `AreaObscurement(Obscurement.lightly)`, `DisadvantageOnChecks(Skill.perception)`, and MUST NOT include `ExtinguishesOpenFlames`. | SRD 5.2.1, "Heavy Precipitation", p. 195 ("Heavy rain also extinguishes open flames") | unhappy (rule interaction) |
| FR-050 | The Wisdom (Perception) Disadvantage of Heavy Precipitation MUST apply to **all** Wisdom (Perception) checks and MUST NOT be narrowed to checks to see something in a Lightly Obscured space. Both effects are reported (Q-7). | SRD 5.2.1, "Heavy Precipitation", p. 195 ("Disadvantage on all Wisdom (Perception) checks"); "Lightly Obscured", p. 184 | unhappy (rule interaction) |
| FR-051 | For `open` Strong Wind the effects, in order, MUST be `DisadvantageOnRangedWeaponAttackRolls`, `ExtinguishesOpenFlames`, `DispersesFog`, `FlyingCreaturesMustLandAtEndOfTurn` (a flying creature that does not land falls; the fall is `environment-hazards-and-objects.md`). | SRD 5.2.1, "Strong Wind", p. 195 ("A flying creature in a strong wind must land at the end of its turn or fall") | happy |
| FR-052 | For `desertSandstorm` the effects MUST be the four of FR-051, then `DisadvantageOnChecks(Skill.perception)`. | SRD 5.2.1, "Strong Wind", p. 195 ("A strong wind in a desert can create a sandstorm that imposes Disadvantage on Wisdom (Perception) checks") | unhappy (rule interaction) |
| FR-053 | `DisadvantageOnRangedWeaponAttackRolls` MUST name ranged weapon attack rolls only, never melee weapon attacks or spell attacks. A thrown Melee weapon makes a ranged attack and is included (Q-8). | SRD 5.2.1, "Strong Wind", p. 195 ("Disadvantage on ranged attack rolls with weapons"); "Thrown", p. 90 | unhappy (rule interaction) |
| FR-054 | For `SlipperyIceSource` the effects MUST be exactly `DifficultTerrain`. The save of FR-042 to FR-044 is separate and per creature. | SRD 5.2.1, "Slippery Ice", p. 195 ("Slippery ice is Difficult Terrain"); "Difficult Terrain", p. 181 | happy |
| FR-081 | `DisadvantageOnChecks` MUST hold a `Skill` only and derive its ability from it (Perception gives Wisdom); no constructor accepts a separate `Ability`, so (Strength, Perception) cannot be built. | shared-types-and-ownership.md FR-005 | unhappy (invalid input) |
| FR-082 | `AreaObscurement` MUST be built only with `Obscurement.lightly` by this catalogue; the catalogue MUST NOT define a second obscurement enum or an `AreaEffect.LightlyObscured` variant. | `content-schema-and-decoder.md` FR-135 | unhappy (rule interaction) |

### 4.10 Thin Ice

`RollThinIceTolerance` with `RollThinIceToleranceCommand()` rolls the tolerance of
one 10-foot-square area once; the host keeps the `ThinIceTolerance` and passes it
back with the load (Q-11). `AssessThinIce` with `AssessThinIceCommand(tolerance,
load)` assesses one area. `ThinIceLoad` holds the occupants (`CombatantID` plus
`PoundsOfWeight`) and the other weight on the area.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-055 | `RollThinIceTolerance` MUST request exactly one roll of `3d10` with modifier 0 through `dice.Roll` and return `RollThinIceToleranceRolled(tolerance, roll)` where `tolerance` is `ThinIceTolerance.fromRoll(roll)` (FR-079). The use case MUST contain no multiplication. | SRD 5.2.1, "Thin Ice", p. 195 ("3d10 x 10 pounds"); `dice-and-rolls.md`; `doc/rules/architecture.md` §4 | happy |
| FR-079 | `ThinIceTolerance.fromRoll(dice.Result)` MUST return the roll total times 10 pounds. It is the only place the factor 10 appears. | SRD 5.2.1, "Thin Ice", p. 195; `dice-and-rolls.md` (the consumer multiplies the total) | happy |
| FR-056 | `ThinIceTolerance.fromRoll` MUST give 30 pounds for a total of 3, 40 for 4, 160 for 16, 290 for 29 and 300 for 30. | SRD 5.2.1, "Thin Ice", p. 195 | unhappy (boundary) |
| FR-057 | `NewThinIceTolerance` MUST accept 1 pound or more and return an error wrapping `srderr.ErrInvalidInput` for 0 or negative (`*EnvironmentalValueError`, reason `belowOne`); `MustThinIceTolerance` panics with that error for the same inputs (FR-078). A value the roll cannot produce (for example 35) MUST still be accepted, so a host may set a tolerance. | `doc/rules/testing.md` §4; SRD 5.2.1, "Thin Ice", p. 195 | unhappy (invalid input) |
| FR-058 | The ice MUST hold when the total weight is **at most** the tolerance and break when it **exceeds** it. | SRD 5.2.1, "Thin Ice", p. 195 ("Whenever the total weight on an area of thin ice exceeds its tolerance, the ice in that area breaks") | unhappy (boundary) |
| FR-059 | The total weight MUST be the sum of every occupant's weight and the other weight. | SRD 5.2.1, "Thin Ice", p. 195 ("the total weight on an area") | happy |
| FR-060 | `NewPoundsOfWeight` MUST accept 0 and more (it returns an error wrapping `srderr.ErrInvalidInput` below 0, `*EnvironmentalValueError` with reason `negative`; FR-078). `NewThinIceLoad(occupants, otherWeight)` MUST return an error wrapping `srderr.ErrInvalidInput` for two occupants with the same `CombatantID`, an `*ThinIceLoadError` with reason `duplicateOccupant` (`ThinIceLoadInvalidReason`), and a `ThinIceLoad` otherwise; `MustThinIceLoad` panics with that error. | `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-061 | A load with no occupants and no other weight MUST hold, with total 0. | `doc/rules/testing.md` §4 (empty and minimal) | unhappy (empty and minimal) |
| FR-062 | When the ice holds the result MUST be `AssessThinIceHolds(totalWeight, tolerance)` and no occupant reported as falling. | SRD 5.2.1, "Thin Ice", p. 195 | happy |
| FR-063 | When the ice breaks the result MUST be `AssessThinIceBroken(totalWeight, tolerance, fallenThrough, below)`: `fallenThrough` an defensively copied `[]CombatantID` of **every** occupant in the order supplied, `below` = `EnvironmentalEffectKind.frigidWater`, so the host follows with `FrigidWaterExposure` (FR-026 to FR-032) for each. | SRD 5.2.1, "Thin Ice", p. 195 ("All creatures on broken ice fall through. Below the ice is frigid water") | happy |
| FR-064 | One call MUST assess one area. Calls for two areas MUST be independent: weight MUST NOT be added across areas and a broken area MUST NOT affect an unbroken one. | SRD 5.2.1, "Thin Ice", p. 195 ("per 10-foot-square area") | unhappy (rule interaction) |
| FR-065 | When the `Roller` returns an error, `RollThinIceTolerance` MUST return it wrapped once as `*srderr.PortError` (`srderr.WrapPort(dice.Port, "roll", err)`) with a nil result, so `errors.Is` and `errors.As` still reach the roller's own error. When it returns a result for a different expression (count, die or modifier) the use case MUST return an error wrapping `srderr.ErrContractViolation` (from `dice.Roll`, errors-and-results.md FR-036), never a panic. A cancelled `ctx` is returned as `context.Canceled`, not a result (errors-and-results.md FR-054). | `doc/rules/architecture.md` §4; `doc/rules/testing.md` §4 (collaborator failure); `dice-and-rolls.md` FR-051, FR-052 | unhappy (collaborator failure) |

**4.10a Tolerance** (FR-055, FR-056, FR-079): a fixed roller serving total 3, 4, 16,
29, 30 gives 30, 40, 160, 290, 300 pounds.

**4.10b Does the ice break?** (FR-058 to FR-064), tolerance 160 unless stated;
occupants as id: pounds, then other weight:

- holds: none, 0 (total 0, FR-061); A: 90 (FR-062); A: 159 (FR-058); A: 160 (FR-058); A: 0 (FR-060). At tolerance 300, A: 300 holds (FR-058).
- breaks, below frigid water: A: 161, A falls through (FR-058, FR-063); A: 100, B: 100, total 200, A and B in order (FR-059, FR-063); A: 100 with other 61, total 161, A falls (FR-059); none with other 161, nobody falls (FR-059, FR-063). At tolerance 30, A: 31 breaks (FR-058).
- two areas, totals 200 and 100: area 1 breaks, area 2 holds (FR-064).

### 4.11 Cross-cutting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-066 | `EnvironmentalExposure` MUST be closed interface with exactly `DeepWaterExposure`, `ExtremeColdExposure`, `ExtremeHeatExposure`, `FrigidWaterExposure`, `SlipperyIceExposure`, so a `switch` is exhaustive. Heavy Precipitation and Strong Wind have none (FR-046); High Altitude and Thin Ice are the commands of their own use cases, so no result variant is unreachable. | SRD 5.2.1, "Environmental Effects", p. 195; `doc/rules/architecture.md` §3, §4 | happy |
| FR-067 | No use case here MUST take or return a `ConditionSet`, a condition event or an Exhaustion level count. | `conditions.md` FR-021, FR-025, FR-039, FR-123; shared-types-and-ownership.md FR-013 | happy |
| FR-083 | The consequence of a failed save (Exhaustion 1, or Prone) MUST be returned only as data inside the handoff's `SaveGatedEffect` (FR-009); the host applies it through `ConditionSet.apply` (`conditions.md` FR-123). | SRD 5.2.1, "Deep Water", "Slippery Ice", p. 195; `toolbox-hazard-effects.md` FR-044, FR-046 | happy |
| FR-084 | This spec MUST NOT state or imply that `toolbox-hazard-effects.md` applies Immunity, an already-held condition, Exhaustion at level 6, or any consequence; those are host concerns over `conditions.md`. Verified by doc review. | `toolbox-hazard-effects.md` section 1, FR-046 | unhappy (rule interaction) |
| FR-089 | The `SaveGatedSubject` built from a profile MUST carry `SaveGatedExempt` in exactly two cases: Deep Water (past FR-010) with `hasSwimSpeed`, and Frigid Water (past FR-026) with `IceColdWaterAdaptation.adapted`. Every other combination (including a Swim Speed in Frigid Water or adaptation in Deep Water) MUST carry `SaveGatedNoExemption`. Both cases still return the handoff, never a No save. | SRD 5.2.1, "Deep Water", "Frigid Water", p. 195; `toolbox-hazard-effects.md` FR-076 | unhappy (rule interaction) |
| FR-093 | This spec's change MUST add to the `toolbox` signal constants the constants for the five operation values, the result values of section 7 (including `error` under `roll_thin_ice_tolerance`) and the three `srd5e.toolbox.environment.*` signals, and MUST NOT define a second signals class; `toolbox-hazard-effects.md` section 7 lists none of them. | `toolbox-hazard-effects.md` FR-006, section 7, Q-15; `doc/rules/observability.md` §5 | unhappy (name collision) |
| FR-092 | No use case here MUST call another use case (including `ResolveSaveGatedEffect`, `ResolveD20Test` or its own five) or construct a nested `telemetry.Telemetry`. The effect and subject are built by the pure domain service `EnvironmentalSaveRules`; each use case records only its own signals. | shared-types-and-ownership.md FR-044 | unhappy (nested telemetry) |
| FR-068 | Given the same command and the same fixed roll, every use case MUST return an equal result. No operation reads the clock: hours, minutes, days and turns arrive as command values. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 (determinism) | unhappy (determinism) |
| FR-069 | No use case or domain operation MUST mutate its inputs. Every list in a command, result or value object MUST be defensively copied, and mutating a list after construction MUST NOT change the object. | `doc/rules/architecture.md` §3, §4; `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-070 | `AssessEnvironmentalExposure`, `AssessHighAltitude`, `AssessThinIce` and `DescribeEnvironmentalAreaEffects` MUST make no `Roller` request and read no content. Only `RollThinIceTolerance` uses a port. | `doc/rules/architecture.md` §4 | happy |
| FR-071 | Each use case call MUST be wrapped in exactly one span `srd5e.toolbox.<operation>` with attributes `srd5e.outcome` and `srd5e.toolbox.result`. | `doc/rules/observability.md` §5, §6; `toolbox-hazard-effects.md` section 7 | happy |
| FR-085 | Each call MUST increment the hazard spec's `srd5e.toolbox.operations` once and record its `srd5e.toolbox.operation_duration`; the counter carries `srd5e.toolbox.operation`, `srd5e.outcome` and `srd5e.toolbox.result`, the histogram only `srd5e.toolbox.operation` (`toolbox-hazard-effects.md` section 7, FR-081). This spec defines no operations counter, errors counter or duration histogram. | `doc/rules/observability.md` §5, §6 | happy |
| FR-072 | A handoff, a No save, a high-altitude assessment, ice that holds or breaks, a rolled tolerance and a described list are normal outcomes and MUST be recorded with `srd5e.outcome` = `success`, no error status and no log. | `doc/rules/observability.md` §6, §7 | unhappy (telemetry on failure) |
| FR-073 | A dice port failure from `RollThinIceTolerance` MUST be recorded once on the span (`span.RecordError`, error status) and once in `srd5e.toolbox.errors` with `error.type` = `srderr.ErrorType(err)` (`dice_roller_error`), and not logged. Per `toolbox-hazard-effects.md` FR-078 the call MUST also increment `srd5e.toolbox.operations` once with outcome `failure` and result `error`, and record `operation_duration` once; the wrapped error is then returned. No use case here returns an error wrapping `srderr.ErrInvalidInput`: invalid host input fails in a typed-value constructor before any command exists. | `doc/rules/observability.md` §6, §7; `doc/rules/architecture.md` §6; `toolbox-hazard-effects.md` FR-072, FR-078, FR-081 | unhappy (telemetry on failure) |
| FR-074 | Every use case MUST return identical results with the default the no-op telemetry and a recording one. | `doc/rules/observability.md` §9 | unhappy (determinism) |
| FR-075 | Attributes MUST come from closed sets (operation, result, effect, reason) and contain no DC, hour, minute, weight, altitude, temperature, id or free text. | `doc/rules/observability.md` §5 | unhappy (telemetry) |
| FR-086 | `srd5e.toolbox.environment.no_save` MUST be incremented once per `AssessEnvironmentalExposureNoSave`, with the effect and the `EnvironmentalNoSaveReason`, and for no other result. | `doc/rules/observability.md` §5 | happy |
| FR-087 | `srd5e.toolbox.environment.heat_exposure_hour` MUST be recorded once, with the `ExposureHour`, per Extreme Heat assessment returning a handoff, and not for an Extreme Heat No save or any other effect. | `doc/rules/observability.md` §5 | unhappy (telemetry) |
| FR-088 | `srd5e.toolbox.environment.thin_ice_tolerance` MUST be recorded once, in pounds, per `RollThinIceTolerance` that returns a result, and not when the roller fails. | `doc/rules/observability.md` §5, §7 | unhappy (telemetry on failure) |
| FR-076 | Package `toolbox` MUST export exactly the names in the section 6 export list, each with a doc comment starting with its name (no re-export package); the architecture test derives its expected list from it. | `doc/rules/architecture.md` §4, §7 | happy |
| FR-077 | The domain files of this spec (every file not named `*_usecase.go`) MAY import the kernel packages, the own-context domain (the save-gated value objects of `toolbox-hazard-effects.md`) and the published API of the upstream contexts used (`content`, `d20`, `combat`) permitted by the context map (shared-types-and-ownership.md FR-045; `toolbox-hazard-effects.md` FR-002). They MUST NOT import `dice`, `os`, `net`, `math/rand`, `time`, another context's `internal/` packages or `character`; the `internal/archtest` allow-list encodes this. | `doc/rules/architecture.md` §1, §2, §9 | unhappy (forbidden edge) |
| FR-078 | Every typed host-supplied value MUST have a validating constructor `NewX(v) (X, error)` that returns an error wrapping `srderr.ErrInvalidInput` carrying an `*EnvironmentalValueError{Value, Reason}` (reasons `EnvironmentalValueInvalidReason`: `negative`, `belowOne`), and a `MustX` that panics with that error. `NewThinIceLoad` returns a `*ThinIceLoadError{Reason}` (`ThinIceLoadInvalidReason`). A value accepting every integer (`Temperature`, `ElevationFeet`) has `NewX(v) X` with no error. There are no `<Value>Creation` result families (section 3). | `doc/rules/architecture.md` §4, §6; `toolbox-travel-pace.md` FR-030 | unhappy (invalid input) |

**4.11a Constructor cases** (FR-078). One test per row; `MustX` is
tested on the same inputs and panics with the error `NewX` returns for each invalid input.

| Value | Accepted for | Error for | Req. |
|---|---|---|---|
| `Temperature` | -40, 0, 100, 400 (never invalid) | none | FR-002 |
| `Feet` | 0 | -1 | FR-013 |
| `ElevationFeet` | -200 (every integer) | none | FR-040 |
| `ExposureHour` | 1 | 0, -3 | FR-021 |
| `ImmersedMinutes` | 1 | 0 | FR-028 |
| `HoursAtAltitude` | 1 | 0, -1 (`belowOne`) | FR-090 |
| `DaysAtElevation` | 0 | -1 | FR-039 |
| `PoundsOfWeight` | 0 | -1, -5 | FR-060 |
| `ThinIceTolerance` | 1, 35 (host-set) | 0 | FR-057 |
| `ThinIceLoad` | distinct occupants | two entries for one `CombatantID` (`duplicateOccupant`) | FR-060 |

**Withdrawn ids**

| Id | Fate |
|---|---|
| FR-006 | Withdrawn: moved to toolbox-hazard-effects.md (FR-024) |
| FR-007 | Withdrawn: moved to toolbox-hazard-effects.md (FR-024) |
| FR-008 | Withdrawn: moved to toolbox-hazard-effects.md (FR-026) |
| FR-031 | Withdrawn: moved to toolbox-hazard-effects.md (FR-024) |
| FR-038 | Withdrawn: moved to toolbox-environmental-effects.md (FR-090) |

## 5. Model

Everything in the domain files of package `toolbox` for this spec (not `*_usecase.go`) is pure, synchronous and silent.

- **Value objects** (validating constructor `NewX` plus `MustX`, FR-078), invariants:
  `Temperature` and `ElevationFeet` any integer (FR-002, FR-040); `Feet`,
  `DaysAtElevation`, `PoundsOfWeight` 0 or more (FR-013, FR-039, FR-060);
  `ExposureHour`, `ImmersedMinutes`, `HoursAtAltitude`, `ThinIceTolerance` 1 or
  more (FR-021, FR-028, FR-090, FR-057; `fromRoll` FR-079); `ThinIceOccupant`
  `CombatantID` plus `PoundsOfWeight`; `ThinIceLoad` unique occupant ids (FR-060);
  `CreatureEnvironmentProfile` armor set copied and defensively copied (FR-004, FR-005).
  `SaveGatedEffect`, `SaveGatedSubject` and parts come from `toolbox-hazard-effects.md`.
- **Enums and closed interface types:** those of section 3 plus `SlipperyIceTrigger`,
  `AreaEffectSource`, `AreaEffect`, `EnvironmentalExposure` and the
  typed constructor errors `EnvironmentalValueError` and `ThinIceLoadError`.
- **Domain services** (stateless): `EnvironmentalSaveRules` (FR-009 to FR-032,
  FR-042 to FR-045), `AltitudeAssessment` (FR-033 to FR-041, FR-091), `AreaEffectCatalogue`
  (FR-046 to FR-054), `ThinIceAssessment` (FR-058 to FR-064).
- **Constants** (not `content`, D-2): DC 10; Extreme Heat base 5 and step 1;
  thresholds 0, 100, 10,000, 20,000; 30 days; 100 feet; `3d10 x 10`; the factor 2 of High Altitude, only in `AltitudeAssessment`.
- **Entities, aggregates, events:** none. The host holds each area's tolerance;
  Exhaustion and Prone are events of `conditions.md`.
- **Failures:** expected outcomes are results; a violated precondition of a constructor is an error wrapping `srderr.ErrInvalidInput` (`MustX` panics);
  a port failure is the returned `*srderr.PortError` (FR-065). No content is read (FR-070).

## 6. Interface

All use cases are synchronous (the only port, `Roller`, is). Each is built with `NewX(roller, opts ...Option)` where it rolls, takes an
optional `telemetry.Telemetry` through `WithTelemetry` (no-op default), has one method `Run(ctx context.Context, cmd) (<UseCase>Result, error)`, and returns its own closed interface
`<UseCase>Result` with operation-prefixed variants; every variant is reachable
from every command. `ctx` is first because each use case emits a span; the `error` carries only a port failure or a cancelled `ctx`.

| Use case | Command | Result variants | Port |
|---|---|---|---|
| `AssessEnvironmentalExposure` | `AssessEnvironmentalExposureCommand(exposure: EnvironmentalExposure, profile: CreatureEnvironmentProfile)` | `AssessEnvironmentalExposureSaveGated(effectKind, effect, subject)`; `AssessEnvironmentalExposureNoSave(effectKind, reason)` | none |
| `AssessHighAltitude` | `AssessHighAltitudeCommand(altitude, daysAtElevation, nativity, hours)` | `AssessHighAltitudeAssessed(basis, countsAsHighAltitude, actualHours, effectiveHours)` | none |
| `AssessThinIce` | `AssessThinIceCommand(tolerance, load)` | `AssessThinIceHolds(totalWeight, tolerance)`; `AssessThinIceBroken(totalWeight, tolerance, fallenThrough, below)` | none |
| `DescribeEnvironmentalAreaEffects` | `DescribeEnvironmentalAreaEffectsCommand(source: AreaEffectSource)` | `DescribeEnvironmentalAreaEffectsDescribed(effects)` | none |
| `RollThinIceTolerance` | `RollThinIceToleranceCommand()` | `RollThinIceToleranceRolled(tolerance, roll)` | `Roller` |

- **Ports:** only the existing `Roller` (no new port, double or contract suite);
  infrastructure failure is the returned `*srderr.PortError`, a wrong result is caught by `dice.Roll`.
- **Handoffs:** effect and subject go, with a `SaveGatedInitialExposure` the host
  builds, to `ResolveSaveGatedEffect`; its `SaveGatedRollPlan` to `ResolveD20Test`;
  a failed save's consequence to `conditions.md` FR-123; `actualHours` and
  `effectiveHours` to `toolbox-travel-pace.md` FR-030; `AreaEffect`s to `attacks.md`,
  `hiding-searching-and-senses.md` and `initiative-and-actions.md`; `fallenThrough`
  and `below` drive repeated `FrigidWaterExposure` calls.
- **Export list of package `toolbox`** (FR-076; each name once): the five
  use cases, commands and `<UseCase>Result` types with the variants above;
  `EnvironmentalExposure` with its five variants; `AreaEffectSource` with
  `HeavyPrecipitationSource`, `StrongWindSource`, `SlipperyIceSource`;
  `CreatureEnvironmentProfile`; `ThinIceLoad` (with `ThinIceLoadError`) and `ThinIceOccupant`; `AreaEffect`
  with its seven variants (4.9); the typed values `Temperature`, `Feet`,
  `ElevationFeet`, `ExposureHour`, `ImmersedMinutes`, `HoursAtAltitude`,
  `DaysAtElevation`, `PoundsOfWeight`, `ThinIceTolerance` with their `NewX` and `MustX`
  constructors and `EnvironmentalValueError`; and the enums `EnvironmentalEffectKind`, `TemperatureBand`,
  `SwimCapability`, `IceColdWaterAdaptation`, `DrinkableWaterAccess`,
  `AltitudeNativity`, `AltitudeTravelBasis`, `EnvironmentalNoSaveReason`,
  `SlipperyIceTrigger`, `PrecipitationKind`, `StrongWindSetting`,
  `EnvironmentalValueInvalidReason`, `ThinIceLoadInvalidReason`. Domain services stay unexported.

**Amendments in sibling specs** (made in those files, not here): A-1 `toolbox-hazard-effects.md` FR-076 adds `SaveGatedSubject.exemption` (done). A-3 its section 4.0 names RE-E and RE-F as the shapes of Extreme Cold and Extreme Heat (done). A-4 `toolbox-travel-pace.md` FR-030, FR-031 take `actualHours` and `effectiveHours` and never double (done). A-2 (withdrawn as a sibling request; hazard Q-15 settled that siblings own their operation and result values): this spec lists the five operations and results in section 7 and adds the the `toolbox` signal constants constants itself (FR-093). **A-5 (outstanding)**: `toolbox-hazard-effects.md` lists the three `srd5e.toolbox.environment.*` constants as members of its single the `toolbox` signal constants (until then an in-place addition). Its section 5.2 `NoSaveReason` row amendment is done.

## 7. Observability

The domain is silent by design (`doc/rules/observability.md` §3); only the use
cases are instrumented, with no log. Names are the `toolbox` signal constants, never
literals at call sites; attribute values come from closed sets.

**Shared signals** (owned by `toolbox-hazard-effects.md` section 7): counters
`srd5e.toolbox.operations` and `srd5e.toolbox.errors`, histogram
`srd5e.toolbox.operation_duration`, attributes `srd5e.toolbox.operation` and
`srd5e.toolbox.result`, spans `srd5e.toolbox.<operation>`. This spec owns, and adds to the `toolbox` signal constants itself (FR-093), the operation values and `result` values: `assess_environmental_exposure` gives `save_gated` or `no_save`; `assess_high_altitude` gives `assessed`; `assess_thin_ice` gives `ice_holds` or `ice_broken`; `describe_environmental_area_effects` gives `described`; `roll_thin_ice_tolerance` gives `rolled` or `error` (a returned dice port failure, hazard FR-078).

`srd5e.outcome` is `success` for every value except `error` (`failure`; FR-072,
FR-073). The four pure use cases have no `error` value. Signals this spec owns:

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.toolbox.environment.no_save` | counter, `{assessment}` | `srd5e.toolbox.environment.effect` (`deep_water`, `extreme_cold`, `extreme_heat`, `frigid_water`, `slippery_ice`), `srd5e.toolbox.environment.reason` (snake_case of `EnvironmentalNoSaveReason`) | Why an exposure needed no save, which the shared counter does not carry (FR-086) | Investigate when `drinkable_water_available` or `not_extreme_cold` exceeds 90% of an effect's assessments (the host calls for creatures not exposed) |
| `srd5e.toolbox.environment.heat_exposure_hour` | histogram, `{hour}` | none | Hour of Extreme Heat exposure for which a save is due (DC `4 + hour`); only for a handoff (FR-087) | p99 above 24 (DC 28 or higher): the host never ends an exposure |
| `srd5e.toolbox.environment.thin_ice_tolerance` | histogram, `[lb_av]` | none | Tolerances rolled, in pounds; never when the roller fails (FR-088) | Values outside 30 to 300 mean a roller fault (a host-set tolerance is not recorded) |

Saves that succeed without a roll are counted by the hazard spec's
`srd5e.toolbox.saves`, so there is no automatic-success or saves-required counter
here. The new keys and values are added to `doc/observability/signals.md`.

Evaluation questions (`doc/rules/observability.md` §2): (1) what can go wrong: the
`Roller` fails or returns a wrong result; the host supplies a wrong fact, calls
for an unexposed creature or never ends an exposure; a boundary is off by one
(Constitution minutes, 100 feet, 29 against 30 days, weight against tolerance).
(2, 3) the result attribute and span show the decision, `no_save` shows why, a port
failure sets span error status and the shared errors counter; the DC, circumstances,
basis, weight and tolerance are in the returned result, not in attributes. (4, 5)
counted: shared operations, results, duration, errors, plus the three signals
above; existing instrumentation is unchanged and `conditions.md` signals are
untouched (FR-067). (6) monitor: shared errors over 5 minutes, `no_save` above 90%
for one effect, heat hour p99 above 24, tolerance outside 30 to 300. (7) not noisy:
no per-call `INFO`, normal outcomes are not errors, failures are not logged.

## 8. Test plan

Doubles: `dicetest.FixedRoller` and `dicetest.CallbackRoller` (`dice-and-rolls.md`) for
`RollThinIceTolerance`; the recording `telemetrytest` bundle. No new port, double or
contract suite. Real domain values are used for `DamageAffinities` and
`CreatureEnvironmentProfile`. "then Auto" cases run both use cases in sequence with
a roller that fails on any request.

| Ids | Level | Notes |
|---|---|---|
| FR-001 to FR-005, FR-009, FR-080 | unit | Nine kinds in order; 4.1a one value per test; profile fields, empty armor, defensive copy, defensively copied view, order-independent equality; built `SaveGatedEffect` per case, value equality, no hand-made save |
| FR-010 to FR-017 | unit, application | 4.3a, 4.4a; depths 0, 100, 101; temperatures 0, 1, -60, 120; "then Auto" cases |
| FR-018 to FR-025 | unit, application | 4.5a to 4.5c; DC hours 1, 2, 3, 10, 26, 27, 100; Shield cases assert subject circumstance and planned Disadvantage; precedence cases |
| FR-026 to FR-030, FR-032 | unit, application | 4.6a; Constitution 1, 10, 30; adaptation before the handoff |
| FR-033 to FR-037, FR-039, FR-040, FR-041, FR-090, FR-091 | unit, application | 4.7a, 4.7b; 9,999, 10,000, 20,000, 20,001; 29 and 30 days; native and not; doubling only when the hour counts |
| FR-042 to FR-045 | unit, application | Each trigger; Immunity Cold, Swim Speed and Heavy armor still get the save |
| FR-046 to FR-054, FR-081, FR-082 | unit, application | Each source's list in order, uniqueness, unmodifiability; exhaustive `switch` over `AreaEffectSource`; `DisadvantageOnChecks` has no ability parameter and `DisadvantageOnChecks(Skill.perception)` reports Wisdom (FR-081) |
| FR-055 to FR-064, FR-079 | unit, application | 4.10a, 4.10b; exactly one `3d10` request with modifier 0; `fromRoll` alone; the use case holds no multiplication; tolerance 0, -1, 1, 35; equal and one-over weights; independent areas |
| FR-065 | use case | A failing roller returns one `*srderr.PortError` and `errors.Is` reaches the double's error; wrong count, die, modifier each give an `ErrContractViolation` error; a cancelled `ctx` is `context.Canceled` |
| FR-066, FR-089 | unit, application | Exhaustive `switch` over five variants; exemption in exactly two cases; the other combinations give `SaveGatedNoExemption`; "then Auto" cases of 4.3a, 4.6a |
| FR-067, FR-083, FR-092 | architecture, use case | No signature or result mentions `ConditionSet`; consequence is data in the effect; the `*_usecase.go` files of this spec import or construct no other use case or nested `telemetry.Telemetry` |
| FR-084 | doc review | No sentence says the hazard spec applies a consequence |
| FR-068 to FR-070 | unit, use case | Equal results for equal input and roll; inputs unmutated; returned slices are copies; no clock read (architecture scan); pure use cases run with a failing roller and no `ContentSource` |
| FR-071 to FR-075, FR-085 | use case, architecture | Recording `telemetrytest` bundle: one span, one `operations` increment, one duration record (operation attribute only); `success` for every normal outcome, no error status or log; port error recorded once on the span and once in `srd5e.toolbox.errors`, `operations` once with `failure` and `error`, duration once, `errors.Is` reaches the double's error; identical results with no-op; closed attribute values; a panicking telemetry double propagates (errors-and-results.md FR-051); architecture scan: no use case returns or inspects an `ErrInvalidInput` error |
| FR-086 to FR-088 | use case | `no_save` once per No save and not for a handoff; `heat_exposure_hour` for an Extreme Heat handoff only; `thin_ice_tolerance` once per rolled tolerance, not on a port failure |
| FR-076, FR-077, FR-093 | architecture | Export list derived from section 6 and the doc-comment-on-every-export check; the `toolbox` signal constants include the five operations, the section 7 results and the three `srd5e.toolbox.environment.*` signals, each matching the `doc/observability/signals.md` catalogue, and no second signals file; import-direction scan of the domain files, no `character` import |
| FR-078 | unit | 4.11a: `NewX` against `MustX`; `errors.Is` `ErrInvalidInput` and `errors.As` of the typed error; `NewTemperature` and `NewElevationFeet` never fail; a `Fuzz` test over each integer constructor never panics |
| (scenario) | scenario | Public API only: Constitution 10 in frigid water at minute 11 gets a handoff; the host builds a `SaveGatedInitialExposure`, `ResolveSaveGatedEffect` returns `SaveRequired` with a roll plan, `ResolveD20Test` resolves it, a failure returns `Failed` with `SaveGatedExhaustionGain(1, ...)`, and the **host** applies it through `ConditionSet.apply`. A fixed roll of 16 holds 160 pounds and breaks at 161, and the creatures are assessed for Frigid Water. Slippery Ice gives a Dexterity plan and the Prone `SaveGatedCondition` the same way. |

- **Mutation testing:** `gomu` on package `toolbox` (the environment files), run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (command in `doc/rules/testing.md` §8),
  score at least 80, watching `<` against `<=` at 0
  and 100 degrees, 100 feet, the Constitution score, 10,000 and 20,000 feet, 30
  days, the tolerance, and the `4 + hour` constant.
- Every live FR id appears above; withdrawn ids have no test. Only FR-084 is
  covered by doc review.

## 9. Open questions and decisions

**Settled, one line each** (none open; all rows closed): Q-4 the SRD gives no schedule beyond "end of each hour/minute", so the host calls once per completed hour or minute and keeps the count; `hour` and `minutesImmersed` start at 1, the first Frigid Water save is at minute `score + 1`; whether a count restarts on leaving is a host fact (interpretation: the SRD tracks no history, p. 195; FR-026, FR-027). Q-5 interpretation: "can't become acclimated ... unless native" lifts only the prohibition, so natives still need 30 days (FR-037); the host may pass a native's true days. Q-6 the host counts whole days at the elevation and passes `DaysAtElevation`; no history is tracked, partial hours are rounded by the host (package design). Q-7 report both effects (FR-048, FR-050): p. 195 states them separately and "all" Wisdom (Perception) checks is wider than Lightly Obscured, p. 184; Disadvantage does not stack (p. 8), the kernel's `RollMode.resolve` dedupes; the host supplies `otherObscurement`. Q-8 a thrown Melee weapon is a ranged weapon attack (Thrown, p. 90) and is included; spell attacks are not "with weapons" (FR-053). Q-9 the SRD defines no "exposed" (shelter, clothing), so the host decides and calls only for exposed creatures (FR-015 to FR-025). Q-10 the SRD has no list of ice-cold-water creatures; the host supplies `IceColdWaterAdaptation` and `content` gains no field (FR-089). Q-11 `RollThinIceTolerance` rolls once per 10-foot-square area (p. 195), the host stores and passes it back; the SRD gives no fall depth or damage, so falling through ends in Frigid Water only (FR-055, FR-063, FR-064). Q-12 the host decides whether a flying or swimming creature "moves onto" ice and passes the trigger; no exemption is invented (FR-042 to FR-045). Q-16 no: `toolbox-hazard-effects.md` FR-099 fixes `ToolboxRuleOptions` to two fields, so the sandstorm stays the command field `StrongWindSetting` (no amendment needed; the SRD says "can", p. 195, a per-area GM choice).

**Decided** (one line each): Q-1 context `toolbox`, shared-types-and-ownership.md FR-045 and `toolbox-hazard-effects.md` FR-001, FR-002, FR-005, FR-006. Q-2 `Obscurement` is `content-schema-and-decoder.md` FR-135, `DamageAffinities` is `hit-points-and-damage.md` FR-091, `CombatantID` is `combat-foundations.md` FR-001. Q-3 all-damage Resistance counts, `toolbox-hazard-effects.md` FR-025. Q-13 signal ownership is `toolbox-hazard-effects.md` section 7 (A-2, A-5). Q-14 `DifficultyClass` and `D20TestOutcome`, shared-types-and-ownership.md FR-002, FR-004. Q-15 Swim Speed and adaptation set `SaveGatedExempt`, `toolbox-hazard-effects.md` FR-076 (FR-089). Q-17 `toolbox-travel-pace.md` FR-030, FR-031 take both hours values and never double (A-4). D-1 this spec applies no Exhaustion or Prone (FR-067, FR-083). D-2 the environmental numbers are domain constants, not `content`. D-3 static area effects get a use case for one barrel API and uniform telemetry. D-4 this spec owns the whole High Altitude rule including the doubling, shared-types-and-ownership.md FR-048. D-5 High Altitude and Thin Ice are separate use cases because their results are impossible for the other exposures, so every variant stays reachable.

## 10. Changelog impact

`CHANGELOG.md`, under Unreleased, **Added**: the Environmental Effects rules of the
Gameplay Toolbox in the `toolbox` context (Deep Water, Extreme Cold, Extreme
Heat, Frigid Water, Heavy Precipitation, High Altitude, Slippery Ice, Strong Wind
and Thin Ice), with the use cases `AssessEnvironmentalExposure`,
`AssessHighAltitude`, `AssessThinIce`, `DescribeEnvironmentalAreaEffects` and
`RollThinIceTolerance`, their commands, results and typed values, and the
`srd5e.toolbox.environment.*` signals. Not **BREAKING**: new API only. Adding the
`toolbox` context itself is recorded by `toolbox-hazard-effects.md`.
