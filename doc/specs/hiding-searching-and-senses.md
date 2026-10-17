# Hiding, Searching, Studying and senses

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `shared-types-and-ownership.md`, `errors-and-results.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `combat-foundations.md`, `conditions.md` |

## 1. Purpose and scope

The rule consequences of light, obscured areas and the four special senses
(Blindsight, Darkvision, Tremorsense, Truesight), and the check mechanics of
the Hide, Search and Study actions, as pure functions over facts the host
supplies: what light and obscurement do to a check made to see
(`LightRules`), what each sense changes at a distance (`SpecialSenseRules`),
whether an observer can see a target (`SightRules.CanSee`), whether a
creature may hide and what ends it (`HideRules`), and which skills the Search
and Study tables suggest (`SearchRules`, `StudyRules`). The engine models no
map: the host decides line of sight, distances, light and cover.
`initiative-and-actions.md` FR-050 only records that the actions were taken.

**Out of scope:**

- **Rolling** (`d20`; no srd5e use case orchestrates these actions, D-1):
  the host spends the action, calls `Assess`, rolls through `d20.Resolver.Resolve`,
  then calls `Resolve`.
- **Features that change what ends hiding**: the Thief's Stealth Attack (p. 64;
  `class-rogue.md` FR-064; D-19).
- **Line of sight, distance, geometry**, weather and terrain obscurement: host
  facts (`attacks.md` section 1).
- **The Invisible and Blinded conditions' effects** (`conditions.md` FR-069 to
  FR-072, FR-126; FR-048 to FR-051) and **attack-roll consequences of unseen
  attackers and targets** (`attacks.md` FR-034 to FR-038, FR-104).
- **Passive Perception** arithmetic (`derived-stats.md` section 4.5); only the
  Lightly Obscured Disadvantage is supplied here (FR-015).
- **Sense ranges as data** (`content-thin-slice.md` FR-010 to FR-013, FR-022;
  `content-monsters.md` FR-027); whether Stonecunning's Tremorsense works is the
  host's report of contact (FR-033).
- **Effects that grant senses or light**, what a Search or Study yields (the
  Game Master decides), and exploration travel rules (p. 12).
- **Anything not in SRD 5.2.1.** Nothing core to these topics was missing.

**Supersedes** (`shared-types-and-ownership.md` FR-048): no other spec
restated these mechanics.
`LightLevel`, `DarknessKind`, `Obscurement` and `SpecialSense` are owned by
`content-schema-and-decoder.md` (FR-133 to FR-136;
`shared-types-and-ownership.md` FR-053); FR-001, FR-002, FR-016 are withdrawn
to it (section 4.15).

## 2. Source

- SRD 5.2.1, "Vision and Light", "Obscured Areas", "Light", "Special Senses",
  "Hiding", p. 11; "Finding Hidden Objects", p. 12; "Skills" table, p. 9;
  "Actions" table, pp. 9 to 10; "Bonus Actions", p. 10; "Unseen Attackers and
  Targets", p. 14.
- SRD 5.2.1, "Rules Glossary": "Blinded [Condition]", "Blindsight", p. 177;
  "Bright Light", p. 178; "Cover", "Creature Type", p. 179; "Darkness",
  "Darkvision", p. 180; "Dim Light", p. 181; "Heavily Obscured", p. 182; "Hide
  [Action]", "Illusions", p. 183; "Invisible [Condition]", "Lightly Obscured",
  p. 184; "Passive Perception", p. 186; "Search [Action]", p. 187; "Study
  [Action]", "Areas of Knowledge", p. 189; "Tremorsense", "Truesight", p. 190.
  The "Truesight" entry names Darkness only, not "Heavily Obscured".
- SRD 5.2.1, "Rogue Subclass: Thief", "Level 9: Supreme Sneak", p. 64 (D-19);
  "Spells", "Darkness", p. 122 (only for FR-025).
- Printed page numbers equal PDF page numbers (checked).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Bright Light, Dim Light, Darkness; normal and magical Darkness; Lightly / Heavily Obscured | `LightLevel` (`content-schema-and-decoder.md` FR-133), `DarknessKind` (FR-134, meaningful only when the light is `darkness`), `Obscurement` (ordered by severity, FR-135) | p. 11, 178, 180, 181, 190. Referenced. |
| (light and obscuring causes of a space) | `SpaceVisibility` | `light`, `DarknessKind`, `OtherObscurement` (fog, foliage, p. 11). |
| (a check made to see, and its effect) | `SightCheckKind` (`PerceptionToSee`, `OtherCheckRequiringSight`, `CheckNotRequiringSight`), `SightCheckEffect` (`unaffected`, `disadvantage`, `AutomaticFailure`) | p. 11, 177, 184. |
| Blindsight, Darkvision, Tremorsense, Truesight | `SpecialSense` (FR-136) | p. 11. Referenced; `content-monsters.md` types `SenseEntry.kind` with it. |
| (a sense with a range), "within a (specific/specified) range" | `SenseRange(sense, RangeFeet)`, `SenseRanges` (defensively copied, one range per sense), `SenseReach` (`WithinRange`, `BeyondRange`) | `RangeFeet` at least 0 (FR-017); `NewSenseRange` (FR-097); a distance up to and including the range is within (D-10). |
| (light as perceived) | `PerceivedLight(level, color)`, `ColorPerception` (`FullColor`, `ShadesOfGray`) | p. 180. |
| line of sight, "can somehow see" | `LineOfSight`, `SightRelation` (`conditions.md`) | Reused (D-5). |
| Invisible, Blinded | `ConditionKind.invisible`, `.blinded` (`content`) | |
| Tremorsense contact | `TremorsenseContact` (`SharedSurfaceOrLiquid`, `NoSharedSurface`, `TargetInTheAir`), `DetectedThing` (`creature`, `MovingObject`, `StationaryObject`), `TremorsenseResult` (`pinpointed`, `NotDetected`) | p. 190. |
| Visual Illusions, true form, Ethereal Plane | `IllusionAppearance` (`transparent`, `AsCreated`), `IllusionSave` (`AutomaticSuccess`, `RollNormally`), `TrueFormPerception` (`discerned`, `NotDiscerned`), `EtherealView` (`seen`, `NotSeen`) | p. 190. |
| (Truesight query facts) | `TruesightScene(DistanceFeet, senses)` | Other facts are explicit arguments (FR-095). |
| out of any enemy's line of sight | `EnemySighting(enemy, LineOfSight, HiderSeesEnemy)` | `enemy` is a `CombatantID`. |
| Three-Quarters / Total Cover | `HideCoverDegree` (`HalfCover`, `ThreeQuartersCover`, `TotalCover`) | p. 179. The host maps `attacks.md` `Cover` to it (a sibling rank, not imported). |
| the Hide requirements | `HideScene`, `HideAssessment` (`HideAvailable(hider, check, DiscernedSightings)`, `HideBlocked(hider, reasons, EnemiesWithLineOfSight, DiscernedSightings)`), `HideBlock` (`NotConcealed`, `InEnemyLineOfSight`) | p. 183. `reasons` is `[]HideBlock`, `EnemiesWithLineOfSight` `[]CombatantID`, `DiscernedSightings` `[]EnemySighting`. |
| (the Hide check) | `HideCheckSpec(ability, skill, dc)` | Dexterity, Stealth, `DifficultyClass(15)` = `DifficultyClass.medium`. |
| hidden, "the DC for a creature to find you", result of a Hide attempt | `HidingStatus` (`HidingStatusHidden(HiddenState)`, `HidingStatusNotHidden`), `HiddenState(FindDC)`, `HideResolution` (`HideSucceeded(state, events)`, `HideCheckFailed(total)`, `HideNotRolled(reason)`) | p. 183; `FindDC` is a `d20` `DifficultyClass`; `reason` is a `ForcedFailureReason` (FR-099). |
| (what ends hiding; events of Hide) | `HidingActivity`, `HidingEffect` (`RemainsHidden`, `StopsBeingHidden`), `HidingFinder` (`enemy`, `NotEnemy`), `HidingChange(status, events)`; `HidingEvent` (closed, implements `CombatEvent`): `CreatureHid(hider, FindDC)`, `HidingEnded(hider, activity)` | p. 183; FR-071, FR-072, FR-092 to FR-094, FR-102. |
| Search [Action] | `SearchTarget` (the four table rows), `SearchSkillAssessment` (closed), `SearchProximity` (`NearTarget`, `NotNearTarget`), `SearchReveal` (`possible`, `impossible`) | p. 187, 12. |
| Study [Action] | `StudySubject` (`StudyTopic(KnowledgeTopic)`, `StudyCreature(CreatureType)`), `KnowledgeTopic` (21 values, D-21), `StudySkillAssessment` (closed) | p. 189. |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3). No type
is a display string.

## 4. Requirements

Stable ids. Distances are whole feet. A cases-table row inherits the Source of
its Requirement column.

### 4.1 Light and obscurement

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-003 | `ObscurementOf(BrightLight)` MUST be `none`. | SRD 5.2.1, "Bright Light", p. 178 | happy |
| FR-004 | `ObscurementOf(DimLight)` MUST be `lightly`. | SRD 5.2.1, "Dim Light", p. 181 | happy |
| FR-005 | `ObscurementOf(darkness)` MUST be `heavily`. | SRD 5.2.1, "Darkness", p. 180 | happy |
| FR-006 | A `SpaceVisibility` with `DarknessKind` `magical` and a light other than `darkness` MUST return an error matching `srderr.ErrInvalidInput`; `DarknessKind` defaults to `nonmagical`. | SRD 5.2.1, "Truesight", p. 190 ("normal and magical Darkness") | unhappy (invalid input) |
| FR-007 | `SpaceVisibility.obscurement` MUST be the more severe of `ObscurementOf(light)` and `OtherObscurement`. | SRD 5.2.1, "Obscured Areas", p. 11; combining is a package decision (D-8) | unhappy (rule interaction) |

#### Cases 4.1a: obscurement of a space

| Label | Light | Other obscurement | Expected | Requirement | Path |
|---|---|---|---|---|---|
| bright, nothing else | Bright Light | none | none | FR-003 | happy |
| dim | Dim Light | none | lightly | FR-004 | happy |
| darkness | Darkness | none | heavily | FR-005 | happy |
| bright (or dim) with heavy fog | Bright Light | heavily | heavily | FR-007 | unhappy (rule interaction) |
| darkness with patchy fog | Darkness | lightly | heavily | FR-007 | unhappy (rule interaction) |
| magical flag on Dim Light | Dim Light (magical) | none | an invalid-input error (`srderr.ErrInvalidInput`) | FR-006 | unhappy (invalid input) |

### 4.2 Checks made to see

The effect belongs to the space of the thing seen (p. 184, p. 182), not the
observer's own; p. 11 words Lightly Obscured from the observer's side (D-18).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-008 | In a Lightly Obscured space, a Wisdom (Perception) check made to see something there (`PerceptionToSee`) MUST have `SightCheckEffect.disadvantage`. | SRD 5.2.1, "Lightly Obscured", p. 184; "Obscured Areas", p. 11; D-18 | happy |
| FR-009 | In a Lightly Obscured space, any other check requiring sight (`OtherCheckRequiringSight`) MUST be `unaffected`. | SRD 5.2.1, "Lightly Obscured", p. 184 | unhappy (boundary) |
| FR-010 | In a Lightly Obscured space, a check not requiring sight (`CheckNotRequiringSight`) MUST be `unaffected`. | SRD 5.2.1, "Obscured Areas", p. 11 ("checks that rely on sight") | unhappy (boundary) |
| FR-011 | In a Heavily Obscured space, a check requiring sight (`PerceptionToSee` or `OtherCheckRequiringSight`) MUST be `AutomaticFailure`: the observer has Blinded while trying to see there (`conditions.md` FR-048). | SRD 5.2.1, "Heavily Obscured", p. 182; "Blinded", p. 177 | happy |
| FR-012 | In a Heavily Obscured space, a check not requiring sight MUST be `unaffected` (`conditions.md` FR-049). | SRD 5.2.1, "Blinded", p. 177 | unhappy (boundary) |
| FR-013 | For an unobscured space (`none`) every `SightCheckKind` MUST be `unaffected`. | SRD 5.2.1, "Bright Light", p. 178 | unhappy (empty and minimal) |
| FR-014 | `AttemptConditions(Obscurement)` MUST return a defensively copied `[]ConditionKind` (FR-100): `{blinded}` for `heavily`, empty for `none` and `lightly`. It is a derived, per-attempt effect: it takes no `ConditionSet` and produces no event. | SRD 5.2.1, "Heavily Obscured", p. 182 ("while trying to see") | unhappy (rule interaction) |
| FR-015 | `disadvantage` MUST convert to the kernel `Disadvantage` `RollCircumstance` (an empty list for `unaffected` and `AutomaticFailure`), for an active check and Passive Perception alike (`derived-stats.md` FR-021). This spec MUST NOT perform Passive Perception arithmetic. | SRD 5.2.1, "Passive Perception", p. 186; "Lightly Obscured", p. 184 | happy |

Tests for 4.2: one per cell of the matrix FR-008 to FR-013 (obscurement `none`, `lightly`, `heavily` by the three `SightCheckKind` values), and FR-015 (`Circumstances(disadvantage)` is `[Disadvantage]`; `Circumstances(unaffected)` and `Circumstances(AutomaticFailure)` are `[]`).

### 4.3 Special senses: ranges and reach

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-017 | A `SenseRange` with `RangeFeet` below 0 MUST return an error matching `srderr.ErrInvalidInput`; 0 is valid. | package validation; `content-monsters.md` FR-028 | unhappy (invalid input) |
| FR-018 | `SenseRanges` MUST hold at most one range per `SpecialSense`; given two of the same sense, the greater is kept. Its view MUST be defensively copied (a copy). | SRD 5.2.1, "Drow", p. 85; D-10 | unhappy (rule interaction) |
| FR-104 | A character's Darkvision comes from `SpeciesPassives.DarkvisionFeet` (`species-trait-behaviour.md` FR-015: 60, or 120 for Dwarf, Orc and Drow; none for Goliath, Halfling and Human). The host passes a result that is present (not the zero value) as `SenseRange(SpecialSense.darkvision, feet)` into `SenseRanges` beside the range of any other source, and FR-018 keeps the greater. This spec reads no species type. | SRD 5.2.1, "Darkvision", p. 180; "Drow", p. 85 | unhappy (rule interaction) |
| FR-019 | `Reach(sense, DistanceFeet, ranges)` MUST be `WithinRange` when the creature has the sense and the distance is at most its range, else `BeyondRange`. | SRD 5.2.1, "Darkvision", p. 180; "Blindsight", p. 177; "Tremorsense", p. 190 ("within a specific range"); "Truesight", p. 190 ("within a specific range") | happy |
| FR-020 | A negative `DistanceFeet` MUST return an error matching `srderr.ErrInvalidInput`. | `doc/rules/testing.md` §4 | unhappy (invalid input) |

#### Cases 4.3a: reach

| Label | Senses | Sense asked | Distance | Expected | Requirement | Path |
|---|---|---|---|---|---|---|
| well inside | Darkvision 60 | Darkvision | 30 | WithinRange | FR-019 | happy |
| at the range | Darkvision 60 | Darkvision | 60 | WithinRange | FR-019 | unhappy (boundary) |
| one beyond | Darkvision 60 | Darkvision | 61 | BeyondRange | FR-019 | unhappy (boundary) |
| zero distance | Darkvision 60 | Darkvision | 0 | WithinRange | FR-019 | unhappy (boundary) |
| range 0, distance 0; 1 | Blindsight 0 | Blindsight | 0; 1 | WithinRange; BeyondRange | FR-019, FR-017 | unhappy (boundary) |
| sense not held; no senses | Darkvision 60; none | Truesight; Darkvision | 5; 0 | BeyondRange | FR-019 | unhappy (empty and minimal) |
| two Darkvision entries (built through the `SenseRanges` constructor: the greater is kept) | Darkvision 60, 120 | Darkvision | 100 | WithinRange | FR-018 | unhappy (rule interaction) |
| negative range | Darkvision -1 | n/a | n/a | an invalid-input error (`srderr.ErrInvalidInput`) | FR-017 | unhappy (invalid input) |
| negative distance | Darkvision 60 | Darkvision | -1 | an invalid-input error (`srderr.ErrInvalidInput`) | FR-020 | unhappy (invalid input) |

### 4.4 Darkvision

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-021 | Within Darkvision range, Dim Light MUST be perceived as Bright Light. | SRD 5.2.1, "Darkvision", p. 180 | happy |
| FR-022 | Within Darkvision range, nonmagical Darkness MUST be perceived as Dim Light. | SRD 5.2.1, "Darkvision", p. 180 | happy |
| FR-023 | Bright Light MUST be perceived as Bright Light whatever the senses. | SRD 5.2.1, "Darkvision", p. 180 | unhappy (boundary) |
| FR-024 | Beyond Darkvision range, or with none, the perceived light MUST equal the actual light. | SRD 5.2.1, "Darkvision", p. 180 | unhappy (boundary) |
| FR-025 | Darkvision MUST NOT change `magical` Darkness: it is perceived as Darkness. | SRD 5.2.1, "Spells", "Darkness", p. 122 ("Darkvision can't see through it"); D-9 |  unhappy (rule interaction) |
| FR-026 | Darkvision MUST change light only. `EffectiveObscurement(space, distance, ranges)` is the one function giving the obscurement an observer meets, combining every sense (FR-101). With no Truesight or Blindsight it MUST be the more severe of `ObscurementOf(PerceivedLight)` and `OtherObscurement`: fog is unaffected, and Darkness seen with Darkvision is Lightly Obscured. | SRD 5.2.1, "Darkvision", p. 180; "Obscured Areas", p. 11 | unhappy (rule interaction) |
| FR-027 | Colors in Darkness within Darkvision range MUST be `ShadesOfGray`; otherwise `FullColor`. | SRD 5.2.1, "Darkvision", p. 180 | happy |

#### Cases 4.4a: perceived light, color and effective obscurement

| Label | Space | Senses | Distance | Perceived light | Color | Effective obscurement | Requirement | Path |
|---|---|---|---|---|---|---|---|---|
| dim, Darkvision | Dim Light | Darkvision 60 | 30 | Bright Light | FullColor | none | FR-021 | happy |
| dark, Darkvision | Darkness | Darkvision 60 | 30 | Dim Light | ShadesOfGray | lightly | FR-022, FR-027 | happy |
| dark, at range; one beyond | Darkness | Darkvision 60 | 60; 61 | Dim Light; Darkness | ShadesOfGray; FullColor | lightly; heavily | FR-022, FR-024 | unhappy (boundary) |
| dark, no Darkvision | Darkness | none | 5 | Darkness | FullColor | heavily | FR-024 | unhappy (empty and minimal) |
| bright, Darkvision | Bright Light | Darkvision 60 | 10 | Bright Light | FullColor | none | FR-023 | unhappy (boundary) |
| magical darkness | Darkness (magical) | Darkvision 60 | 10 | Darkness | FullColor | heavily | FR-025 | unhappy (rule interaction) |
| dark, heavy fog | Darkness, fog heavily | Darkvision 60 | 10 | Dim Light | ShadesOfGray | heavily | FR-026 | unhappy (rule interaction) |
| dim, Blindsight only | Dim Light | Blindsight 30 | 10 | Dim Light | FullColor | none (D-23) | FR-024, FR-101 | unhappy (rule interaction) |

### 4.5 Blindsight

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-028 | An observer with Blindsight within range and a `clear` line MUST see the target even if the observer has Blinded. | SRD 5.2.1, "Blindsight", p. 177 | happy |
| FR-029 | The same observer MUST see a target in Darkness, and in any Heavily Obscured space, within range (`EffectiveObscurement` is `none` within Blindsight range, FR-101, D-23). | SRD 5.2.1, "Blindsight", p. 177; "Heavily Obscured", p. 182 | happy |
| FR-030 | An observer with Blindsight within range MUST see a target with Invisible. | SRD 5.2.1, "Blindsight", p. 177 | happy |
| FR-031 | A `blocked` line (Total Cover is reported as `blocked`, D-14) MUST make the target unseen by Blindsight. | SRD 5.2.1, "Blindsight", p. 177 ("anything that isn't behind Total Cover") | unhappy (state error) |
| FR-032 | Beyond Blindsight range, Blindsight MUST change nothing: `CanSee` applies as if the creature lacked it. | SRD 5.2.1, "Blindsight", p. 177 | unhappy (boundary) |

### 4.6 Tremorsense

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-033 | `Tremorsense(reach, contact, thing)` MUST be `Pinpointed` when the reach is `WithinRange`, the contact is `SharedSurfaceOrLiquid` and the thing is a `creature` or `MovingObject`. | SRD 5.2.1, "Tremorsense", p. 190 | happy |
| FR-034 | It MUST be `NotDetected` when the reach is `BeyondRange`. | SRD 5.2.1, "Tremorsense", p. 190 | unhappy (boundary) |
| FR-035 | It MUST be `NotDetected` when the contact is `NoSharedSurface` (both must touch the same surface or liquid). | SRD 5.2.1, "Tremorsense", p. 190 | unhappy (state error) |
| FR-036 | It MUST be `NotDetected` when the contact is `TargetInTheAir`. | SRD 5.2.1, "Tremorsense", p. 190 | unhappy (state error) |
| FR-037 | It MUST be `NotDetected` for a `StationaryObject`. | SRD 5.2.1, "Tremorsense", p. 190 | unhappy (boundary) |
| FR-038 | Tremorsense MUST NOT count as sight: `CanSee` ignores it, so a creature with only Tremorsense does not see a target in Darkness, even when `pinpointed`. | SRD 5.2.1, "Tremorsense", p. 190 ("doesn't count as a form of sight") | unhappy (rule interaction) |
| FR-039 | Note (no requirement, no test row): a `pinpointed` result tells the observer where the target is; whether that is the location the attacker targeted is the attacker's choice, so this spec yields no `TargetUnseen`; the host decides `TargetAtTargetedLocation` (`attacks.md` FR-104). The attacker still cannot see the target (FR-038), so `attacks.md` FR-034 applies. | SRD 5.2.1, "Tremorsense", p. 190; "Unseen Attackers and Targets", p. 14 | n/a (note) |

#### Cases 4.6a: Tremorsense (range 60)

| Label | Distance | Contact | Thing | Expected | Requirement | Path |
|---|---|---|---|---|---|---|
| creature or moving object, same ground or liquid | 30 | SharedSurfaceOrLiquid | creature; MovingObject | pinpointed | FR-033 | happy |
| at the range | 60 | SharedSurfaceOrLiquid | creature | pinpointed | FR-033 | unhappy (boundary) |
| one beyond | 61 | SharedSurfaceOrLiquid | creature | NotDetected | FR-034 | unhappy (boundary) |
| no shared surface | 30 | NoSharedSurface | creature | NotDetected | FR-035 | unhappy (state error) |
| flying target | 30 | TargetInTheAir | creature; MovingObject | NotDetected | FR-036 | unhappy (state error) |
| stationary object | 30 | SharedSurfaceOrLiquid | StationaryObject | NotDetected | FR-037 | unhappy (boundary) |
| no Tremorsense held | 10 | SharedSurfaceOrLiquid | creature | NotDetected | FR-034 | unhappy (empty and minimal) |

### 4.7 Truesight

All hold only `WithinRange` of the observer's Truesight.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | Darkness (`nonmagical` and `magical`) MUST NOT make the space Heavily Obscured for that observer: the Darkness-derived part of `EffectiveObscurement` is `none` (combined by FR-101). Dim Light keeps its Lightly Obscured effect; that clause is a package decision (D-11). | SRD 5.2.1, "Truesight", p. 190 ("You can see in normal and magical Darkness"); D-11 | happy |
| FR-041 | Truesight MUST NOT remove obscurement from non-light causes: `OtherObscurement` still applies. | SRD 5.2.1, "Truesight", p. 190; "Obscured Areas", p. 11 | unhappy (rule interaction) |
| FR-042 | The observer MUST see a creature or object that has Invisible. | SRD 5.2.1, "Truesight", p. 190 | happy |
| FR-043 | A Visual Illusion MUST be `transparent`; beyond range or without Truesight, `AsCreated`. | SRD 5.2.1, "Truesight", p. 190 | happy |
| FR-044 | The observer MUST `AutomaticSuccess` on saves against Visual Illusions; otherwise `RollNormally`. | SRD 5.2.1, "Truesight", p. 190 | happy |
| FR-045 | The true form of a creature or object the observer sees that was transformed by magic MUST be `discerned`; if not seen, or beyond range, `NotDiscerned`. | SRD 5.2.1, "Truesight", p. 190 | happy |
| FR-046 | The observer MUST see into the Ethereal Plane (`seen`); beyond range, `NotSeen`. | SRD 5.2.1, "Truesight", p. 190 | happy |
| FR-047 | Truesight MUST NOT help an observer with Blinded: only Blindsight is stated to work "even if you have the Blinded condition". | SRD 5.2.1, "Truesight", p. 190; "Blindsight", "Blinded", p. 177; D-11 | unhappy (rule interaction) |
| FR-048 | A Visual Illusion save MUST be reported as `AutomaticSuccess` through `IllusionSave`, not by changing a roll: the caller omits the d20 and `d20` is not asked to resolve it. | SRD 5.2.1, "Truesight", p. 190; "Saving Throw", p. 187 | unhappy (rule interaction) |

#### Cases 4.7a: Truesight 30 ft.

| Label | Distance | Probe | Expected | Requirement | Path |
|---|---|---|---|---|---|
| nonmagical Darkness | 30 | `EffectiveObscurement`, Darkness, no fog | none | FR-040, FR-101 | happy |
| magical Darkness | 30 | `EffectiveObscurement`, magical Darkness, no fog | none | FR-040, FR-101 | happy |
| Dim Light (D-11) | 30 | `EffectiveObscurement`, Dim Light, no fog | lightly | FR-040 | unhappy (boundary) |
| magical Darkness, beyond | 31 | `EffectiveObscurement`, magical Darkness, no fog | heavily | FR-040 | unhappy (boundary) |
| heavy fog | 10 | `EffectiveObscurement`, Bright Light, fog heavily | heavily | FR-041 | unhappy (rule interaction) |
| Darkness with patchy fog | 10 | `EffectiveObscurement`, Darkness, fog lightly; `SightCheckEffect` for `PerceptionToSee` | lightly; disadvantage | FR-041, FR-101 | unhappy (rule interaction) |
| Invisible creature; beyond | 30; 31 | sees Invisible | seen; not seen | FR-042 | happy; boundary |
| Visual Illusion; beyond | 30; 31 | appearance | transparent; AsCreated | FR-043 | happy; boundary |
| save against Visual Illusion; beyond | 30; 31 | save | AutomaticSuccess; RollNormally | FR-044 | happy; boundary |
| no Truesight | 10 | save against Visual Illusion | RollNormally | FR-044 | unhappy (empty and minimal) |
| polymorphed creature seen; not seen | 10 | true form | discerned; NotDiscerned | FR-045 | happy; state error |
| Ethereal Plane; beyond | 10; 50 | view | seen; NotSeen | FR-046 | happy; boundary |
| Blinded observer | 10 | `CanSee`, Bright Light | CannotSee | FR-047 | unhappy (rule interaction) |

### 4.8 Can the observer see the target

`SightRules.CanSee(SightScene)` returns the `conditions.md` `SightRelation`. A
`SightScene` holds `DistanceFeet`, `LineOfSight`, `space` (the space of the
thing seen), `ObserverSenses`, and `ObserverConditions` and
`TargetConditions` (effective `ConditionKind` sets, `conditions.md` FR-013;
defensively copied copies, FR-100).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-049 | `CanSee` MUST evaluate in order, stopping at the first that decides: (1) `blocked` line gives `CannotSee`; (2) Blindsight `WithinRange` gives `CanSee` (FR-028 to FR-030); (3) observer Blinded gives `CannotSee`; (4) Invisible target gives `CannotSee` unless Truesight is `WithinRange` (FR-042); (5) `EffectiveObscurement` of `heavily` gives `CannotSee` (`CanSee` calls `SpecialSenseRules.EffectiveObscurement`, FR-101, and does not re-derive it); (6) otherwise `CanSee`. Lightly Obscured never makes it `CannotSee` (FR-008). | SRD 5.2.1, "Blinded", "Blindsight", p. 177; "Heavily Obscured", p. 182; "Invisible", "Lightly Obscured", p. 184; "Darkvision", p. 180; "Truesight", p. 190; order is a package decision (D-12) | unhappy (rule interaction) |
| FR-050 | `CanSee`, `EffectiveObscurement`, `reach` and the other queries MUST be pure and deterministic: equal inputs give equal outputs in any order, and no input is changed. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (immutability) |

#### Cases 4.8a: `CanSee` (no other senses unless stated; line `clear` unless stated)

| Label | Space | Observer senses and state | Target | Distance | Expected | Requirement | Path |
|---|---|---|---|---|---|---|---|
| plain sight | Bright Light | none | ordinary | 30 | CanSee | FR-049 | happy |
| Dim Light | Dim Light | none | ordinary | 30 | CanSee | FR-049, FR-008 | unhappy (boundary) |
| Darkness | Darkness | none | ordinary | 30 | CannotSee | FR-049, FR-005 | happy |
| Darkness, Darkvision; beyond | Darkness | Darkvision 60 | ordinary | 30; 61 | CanSee; CannotSee | FR-049, FR-022, FR-024 | happy; boundary |
| magical Darkness, Darkvision | Darkness (magical) | Darkvision 60 | ordinary | 30 | CannotSee | FR-049, FR-025 | unhappy (rule interaction) |
| magical Darkness, Truesight; beyond | Darkness (magical) | Truesight 30 | ordinary | 30; 31 | CanSee; CannotSee | FR-049, FR-040 | happy; boundary |
| heavy fog, Darkvision | Bright Light, fog heavily | Darkvision 60 | ordinary | 10 | CannotSee | FR-049, FR-026 | unhappy (rule interaction) |
| heavy fog, Truesight | Bright Light, fog heavily | Truesight 60 | ordinary | 10 | CannotSee | FR-049, FR-041 | unhappy (rule interaction) |
| heavy fog, Blindsight | Bright Light, fog heavily | Blindsight 30 | ordinary | 30 | CanSee; `EffectiveObscurement` none | FR-049, FR-029, FR-101 | unhappy (rule interaction) |
| observer Blinded | Bright Light | Blinded | ordinary | 5 | CannotSee | FR-049 | unhappy (state error) |
| Blinded, Blindsight | Bright Light | Blinded, Blindsight 10 | ordinary | 10 | CanSee | FR-049, FR-028 | unhappy (rule interaction) |
| Blinded, Truesight | Bright Light | Blinded, Truesight 60 | ordinary | 10 | CannotSee | FR-049, FR-047 | unhappy (rule interaction) |
| Invisible target; with Darkvision only | Bright Light | none; Darkvision 60 | Invisible | 5 | CannotSee | FR-049 | unhappy (rule interaction) |
| Invisible, Truesight; Blindsight (separate observers) | Bright Light | Truesight 30; Blindsight 30 | Invisible | 30 | CanSee | FR-049, FR-042, FR-030 | unhappy (rule interaction) |
| Invisible, Blinded, Blindsight | Bright Light | Blinded, Blindsight 30 | Invisible | 30 | CanSee | FR-049, FR-028, FR-030 | unhappy (rule interaction) |
| line blocked, Blindsight | Bright Light | Blindsight 30 | ordinary, `blocked` | 5 | CannotSee | FR-049, FR-031 | unhappy (state error) |
| line blocked, Truesight (separate observer) | Bright Light | Truesight 30 | ordinary, `blocked` | 5 | CannotSee | FR-049 | unhappy (state error) |
| Tremorsense only, Darkness | Darkness | Tremorsense 60 | ordinary | 10 | CannotSee | FR-049, FR-038 | unhappy (rule interaction) |
| Blindsight one beyond, Darkness | Darkness | Blindsight 30 | ordinary | 31 | CannotSee | FR-049, FR-032 | unhappy (boundary) |
| Blinded among other conditions | Bright Light | observer `{blinded, poisoned}` | ordinary | 5 | CannotSee | FR-103, FR-049 | unhappy (rule interaction) |
| Invisible among other conditions | Bright Light | none | `{invisible, prone}` | 5 | CannotSee | FR-103, FR-049 | unhappy (rule interaction) |
| other conditions only; empty sets | Bright Light | `{poisoned, frightened}`; `{}` | `{prone}`; `{}` | 5 | CanSee | FR-103 | unhappy (boundary) |

### 4.9 Hide

`Resolve` records a rolled check; `ResolveForcedFailure` one never rolled
(FR-099). The `HideScene` holds the hider (`CombatantID`), the hider's space
`Obscurement` (raw, D-20), the `HideCoverDegree` values (empty for none), `ObscuredByLargerCreature` (FR-105) and
one `EnemySighting` per enemy.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-051 | The Hide check MUST be Dexterity (Stealth) against DC 15: `HideCheckSpec` carries `Ability.dexterity`, `Skill.stealth`, `DifficultyClass(15)`. | SRD 5.2.1, "Hide [Action]", p. 183; "Actions" table, p. 10 | happy |
| FR-052 | The concealment requirement MUST be met when the hider's space is Heavily Obscured, or the hider is behind Three-Quarters Cover or Total Cover. | SRD 5.2.1, "Hide [Action]", p. 183 | happy |
| FR-053 | The concealment requirement MUST NOT be met by Lightly Obscured, no obscurement, Half Cover alone, or no cover. | SRD 5.2.1, "Hide [Action]", p. 183; "Cover", p. 179 | unhappy (boundary) |
| FR-054 | The line-of-sight requirement MUST be met only when every `EnemySighting` has a `blocked` line. `HideBlocked.EnemiesWithLineOfSight` MUST list, in scene order, the `CombatantID` of every enemy whose line is `clear`, whether or not the hider can see it (a host fact, FR-057, D-13). | SRD 5.2.1, "Hide [Action]", p. 183 | unhappy (state error) |
| FR-055 | With no enemies the line-of-sight requirement MUST be met. | SRD 5.2.1, "Hide [Action]", p. 183; D-13 | unhappy (empty and minimal) |
| FR-056 | When neither requirement is met, `HideBlocked` MUST carry both `HideBlock` reasons, `NotConcealed` then `InEnemyLineOfSight`; deterministic. | SRD 5.2.1, "Hide [Action]", p. 183; `doc/rules/testing.md` §4 | unhappy (rule interaction) |
| FR-057 | `DiscernedSightings` (on `HideAvailable` and `HideBlocked`) MUST hold, in scene order, exactly the `EnemySighting`s whose `HiderSeesEnemy` is `CanSee`, each with its `LineOfSight`; an unseen enemy MUST NOT appear, whatever its line, and nothing else is reported about it except in `EnemiesWithLineOfSight`. | SRD 5.2.1, "Hide [Action]", p. 183 ("if you can see a creature, you can discern whether it can see you"); D-13 | happy |
| FR-058 | When both requirements are met the assessment MUST be `HideAvailable` (hider, `HideCheckSpec`, `DiscernedSightings`). `Resolve` and `ResolveForcedFailure` MUST take `HideAvailable`, so a `HideBlocked` is not accepted (a runtime test, no analyzer test); no check is rolled when `HideBlocked` (D-17). | SRD 5.2.1, "Hide [Action]", p. 183 | unhappy (state error) |
| FR-059 | `Resolve` with `d20.Outcome.success` (total at least 15) MUST return `HideSucceeded(state, events)` with `HiddenState.FindDC` equal to the total. A `HiddenState` with `FindDC` below 15 MUST return an error matching `srderr.ErrInvalidInput`, and so MUST `Resolve` when outcome and total disagree (success below 15, failure at 15 or more). | SRD 5.2.1, "Hide [Action]", p. 183 ("the DC for a creature to find you") | happy |
| FR-060 | `Resolve` with `failure` (total below 15) MUST return `HideCheckFailed` carrying the total, with no `HiddenState` and no event. | SRD 5.2.1, "Hide [Action]", p. 183 | unhappy (boundary) |
| FR-061 | `HideSucceeded.events` MUST hold exactly one `CreatureHid(hider, FindDC)`; `HideCheckFailed` and `HideNotRolled` carry none (FR-092 to FR-094). | `doc/rules/architecture.md` §3; `combat-foundations.md` FR-015 to FR-021 | happy |
| FR-062 | While `HidingStatusHidden`, `ImpliedConditions` MUST return a defensively copied `[]ConditionKind` (FR-100) containing `invisible`; for `HidingStatusNotHidden`, empty. The benefit lapses against a creature that can somehow see the hider (`conditions.md` FR-072). The host unions it into the effective condition sets; it is never stored (D-6). | SRD 5.2.1, "Hide [Action]", p. 183; "Invisible", p. 184; D-6 | happy |
| FR-063 | Note (no requirement, no test row): `Assess` and `Resolve` take no argument about how Hide was taken, so one path serves the Hide action and a Hide granted as a Bonus Action (`initiative-and-actions.md` FR-112, FR-114). | SRD 5.2.1, "Bonus Actions", p. 10; "Hide [Action]", p. 183 | n/a (note) |
| FR-105 | `HideScene` MUST take an optional `ObscuredByLargerCreature` (default false), a host fact. When true the concealment requirement (FR-052) MUST be met even if the hider's space is not Heavily Obscured and no cover applies; the line-of-sight requirement (FR-054) is unchanged. The host sets it from `SpeciesPassives.ObscuresForHide(own, obscurer)` (`species-trait-behaviour.md` FR-017, Naturally Stealthy) only when that creature is the sole obscurer. Interpretation: "obscured" in the trait is the concealment of the Hide requirement, which the trait relaxes, and no other part of Hide. | SRD 5.2.1, "Naturally Stealthy", p. 86; "Hide [Action]", p. 183 | unhappy (rule interaction) |

#### Cases 4.9a: Hide requirements

| Label | Hider's space | Cover | Enemies (lines) | Expected | Requirement | Path |
|---|---|---|---|---|---|---|
| heavily obscured, no line | heavily | none | one, blocked | `HideAvailable` | FR-052, FR-054 | happy |
| Three-Quarters or Total Cover | none | Three-Quarters; Total | one, blocked | `HideAvailable` | FR-052 | happy |
| Half Cover only; Lightly Obscured only; nothing | none, lightly, none | Half, none, none | one, blocked | `HideBlocked(NotConcealed)` | FR-053 | unhappy (boundary) |
| Half and Three-Quarters Cover | none | Half, Three-Quarters | one, blocked | `HideAvailable` | FR-052 | unhappy (rule interaction) |
| Naturally Stealthy: larger creature obscures; same with an enemy line clear | none | none | one, blocked; clear | `HideAvailable`; `HideBlocked(InEnemyLineOfSight)` only | FR-105, FR-054 | unhappy (rule interaction) |
| concealed, an enemy sees | heavily | none | two: blocked, clear; or clear, clear | `HideBlocked(InEnemyLineOfSight)`, lists the clear ones in order | FR-054 | unhappy (state error) |
| no enemies | heavily | none | none | `HideAvailable` | FR-055 | unhappy (empty and minimal) |
| neither requirement | none | none | one, clear | `HideBlocked(NotConcealed, InEnemyLineOfSight)` | FR-056 | unhappy (rule interaction) |
| discern: seen, blocked | heavily | none | one, blocked, hider sees it | `HideAvailable`; `DiscernedSightings` holds it with `blocked` | FR-057 | happy |
| discern: unseen, blocked | heavily | none | one, blocked, hider cannot see it | `HideAvailable`; `DiscernedSightings` empty | FR-057 | unhappy (boundary) |
| discern: clear, hider sees it; cannot see it | heavily | none | one, clear | `HideBlocked`; lists it; `DiscernedSightings` holds it with `clear`; empty | FR-057, FR-054 | unhappy (rule interaction) |
| Darkness, enemy Darkvision: host says blocked; clear | heavily (Darkness) | none | one | `HideAvailable` (raw space value); `HideBlocked(InEnemyLineOfSight)` | FR-052, FR-054, FR-098 | unhappy (rule interaction) |

#### Cases 4.9b: Hide check (requirements met)

| Label | Total | Outcome from `d20` | Expected | `FindDC` | Requirement | Path |
|---|---|---|---|---|---|---|
| exactly DC 15 | 15 | success | `HideSucceeded` | 15 | FR-059 | unhappy (boundary) |
| one below; very low | 14; 2 | failure | `HideCheckFailed(14)`; `(2)` | n/a | FR-060 | unhappy (boundary) |
| one above; high | 16; 23 | success | `HideSucceeded` | 16; 23 | FR-059 | unhappy (boundary) |
| direct state below 15; inconsistent success (14); inconsistent failure (15) | n/a; 14; 15 | n/a; success; failure | an invalid-input error (`srderr.ErrInvalidInput`) for `HiddenState(14)`; for each | n/a | FR-059 | unhappy (invalid input) |
| event, Invisible while hidden | 18 | success | one `CreatureHid(hider, 18)`; `ImpliedConditions` is `{invisible}` | 18 | FR-061, FR-062 | happy |
| no event on failure | 12 | failure | no events | n/a | FR-061 | unhappy (boundary) |
| not hidden | n/a | n/a | `ImpliedConditions` is empty | n/a | FR-062 | unhappy (empty and minimal) |
| forced failure | n/a | automatic failure | `HideNotRolled(reason)`, no events | n/a | FR-099, FR-061 | unhappy (state error) |

### 4.10 Ending hiding and finding a hidden creature

`ApplyActivity(HidingStatus, HidingActivity, hider)` returns a `HidingChange`
for `HidingActivity` values `SoundLouderThanWhisper`,
`SoundNoLouderThanWhisper`, `FoundByEnemy`, `FoundByNonEnemy`, `AttackRoll`,
`SpellWithVerbalComponent`, `SpellWithoutVerbalComponent`; it is defined in
terms of `EffectOf` (FR-102).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-064 | A sound louder than a whisper MUST end hiding (`StopsBeingHidden`). | SRD 5.2.1, "Hide [Action]", p. 183 | happy |
| FR-065 | A sound no louder than a whisper MUST NOT end hiding (`RemainsHidden`). | SRD 5.2.1, "Hide [Action]", p. 183 | unhappy (boundary) |
| FR-066 | Being found by an enemy MUST end hiding. | SRD 5.2.1, "Hide [Action]", p. 183 | happy |
| FR-067 | Being found by a creature that is not an enemy MUST NOT end hiding. | SRD 5.2.1, "Hide [Action]", p. 183 ("an enemy finds you") | unhappy (boundary) |
| FR-068 | An attack roll (weapon, Unarmed Strike or spell attack) MUST end hiding, after the roll: the status the caller holds when it rolls is still `HidingStatusHidden` (`ApplyActivity` returns a new status, FR-073). `attacks.md` never sees a `HidingStatus`: the host derives the "attacker hidden" fact of `attacks.md` FR-037 from the status it read before calling `ApplyActivity`. | SRD 5.2.1, "Hide [Action]", p. 183; "Level 9: Supreme Sneak", p. 64 (D-19); "Attack Roll", p. 177; "Unseen Attackers and Targets", p. 14 | unhappy (rule interaction) |
| FR-069 | Casting a spell with a Verbal component MUST end hiding. | SRD 5.2.1, "Hide [Action]", p. 183 | happy |
| FR-070 | Casting a spell without a Verbal component MUST NOT end hiding. | SRD 5.2.1, "Hide [Action]", p. 183 | unhappy (boundary) |
| FR-071 | An activity whose `EffectOf` is `StopsBeingHidden` MUST return a `HidingChange` with `HidingStatusNotHidden` and exactly one `HidingEnded(hider, activity)`. Any activity on `HidingStatusNotHidden` MUST return `HidingStatusNotHidden` with no event (not an error). | `doc/rules/architecture.md` §3; SRD 5.2.1, "Hide [Action]", p. 183 | unhappy (state error) |
| FR-072 | `AttemptToFind(status, hider, finder, WisdomPerceptionTotal)` for an `enemy` MUST end hiding when the total is at least `FindDC` and leave it unchanged when below; a `NotEnemy` MUST leave it unchanged whatever the total. When hiding ends the one event is `HidingEnded(hider, HidingActivity.FoundByEnemy)`. The total is a caller-supplied Wisdom (Perception) total (D-7). | SRD 5.2.1, "Hide [Action]", p. 183; "Search [Action]", p. 187 | unhappy (boundary) |
| FR-073 | `HiddenState`, `HidingStatus`, `HideAssessment` and the events MUST be value objects (equal when every field is equal); lists in `HideBlocked` and in an assessment MUST be defensively copied (a copy); no function changes a status passed to it. | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |

#### Cases 4.10a: activities and finding (hider Hidden, `FindDC` 16; `EffectOf` agrees with Expected status: `NotHidden` is `StopsBeingHidden`, `Hidden(16)` is `RemainsHidden`)

| Label | Activity or finder | Total | Expected status | Events | Requirement | Path |
|---|---|---|---|---|---|---|
| shout | `SoundLouderThanWhisper` | n/a | NotHidden | one `HidingEnded` | FR-064, FR-071 | happy |
| whisper; silent spell | `SoundNoLouderThanWhisper`; `SpellWithoutVerbalComponent` | n/a | Hidden(16) | none | FR-065, FR-070 | unhappy (boundary) |
| enemy finds | `FoundByEnemy` | n/a | NotHidden | one `HidingEnded` | FR-066 | happy |
| ally finds; non-enemy, total 30 | `FoundByNonEnemy`; NotEnemy | n/a; 30 | Hidden(16) | none | FR-067, FR-072 | unhappy (boundary) |
| attack roll | `AttackRoll`; the status read before applying stays Hidden(16) | n/a | NotHidden | one `HidingEnded` | FR-068 | unhappy (rule interaction) |
| Verbal spell | `SpellWithVerbalComponent` | n/a | NotHidden | one `HidingEnded` | FR-069 | happy |
| already not hidden | `AttackRoll`; enemy with total 30, on NotHidden | n/a | NotHidden | none | FR-071 | unhappy (state error) |
| ended once only | `AttackRoll` twice | n/a | NotHidden | one `HidingEnded` in total | FR-071 | unhappy (rule interaction) |
| enemy, total equals DC; above | enemy | 16; 22 | NotHidden | one `HidingEnded` | FR-072 | unhappy (boundary) |
| enemy, total one below | enemy | 15 | Hidden(16) | none | FR-072 | unhappy (boundary) |
| every activity | all seven | n/a | each has exactly one `EffectOf`; `ApplyActivity` on Hidden agrees | n/a | FR-102 | unhappy (rule interaction) |

### 4.11 Search

Search is a Wisdom check; its table "suggests" skills, so the Game Master may
accept another Wisdom skill.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-074 | The Search check ability MUST be Wisdom. | SRD 5.2.1, "Search [Action]", p. 187 | happy |
| FR-075 | `Skill.insight` MUST be suggested for `SearchTarget.CreaturesStateOfMind`. | SRD 5.2.1, "Search [Action]", table, p. 187 | happy |
| FR-076 | `Skill.medicine` MUST be suggested for `CreaturesAilmentOrCauseOfDeath`. | SRD 5.2.1, "Search [Action]", table, p. 187 | happy |
| FR-077 | `Skill.perception` MUST be suggested for `ConcealedCreatureOrObject`. | SRD 5.2.1, "Search [Action]", table, p. 187 | happy |
| FR-078 | `Skill.survival` MUST be suggested for `TracksOrFood`. | SRD 5.2.1, "Search [Action]", table, p. 187 | happy |
| FR-079 | A Wisdom skill not on the table (Animal Handling) MUST be assessed `OffTableSearchSkill`: accepted at the Game Master's discretion (package decision, D-16). | SRD 5.2.1, "Actions" table, p. 10; "Search [Action]", p. 187 ("suggests"); "Skills" table, p. 9 | unhappy (boundary) |
| FR-080 | A skill whose ability is not Wisdom MUST be assessed `InvalidSearchSkill` carrying the skill and its ability. | SRD 5.2.1, "Search [Action]", p. 187; "Skills" table, p. 9 | unhappy (invalid input) |
| FR-081 | Across the 18 skills, exactly four MUST be `SuggestedSearchSkill`, one `OffTableSearchSkill` and thirteen `InvalidSearchSkill`. | SRD 5.2.1, "Skills" table, p. 9; "Search [Action]", p. 187 | unhappy (boundary) |
| FR-082 | `SearchRules.CanReveal(SearchProximity)` MUST return `impossible` for `NotNearTarget` (a search nowhere near a hidden object reveals nothing, whatever the total) and `possible` for `NearTarget`. Finding a hidden creature (FR-072) is unaffected. | SRD 5.2.1, "Finding Hidden Objects", p. 12 | unhappy (state error) |

Tests for 4.11: one per skill of FR-075 to FR-080 (Arcana, Stealth and Persuasion are the `InvalidSearchSkill` examples), the all-18-skills partition (FR-081), `SearchRules.ability` (FR-074), and both `CanReveal` values (FR-082).

### 4.12 Study

Study is an Intelligence check; every Intelligence skill is on the Areas of
Knowledge table, which "suggests" skills.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-083 | The Study check ability MUST be Intelligence. | SRD 5.2.1, "Study [Action]", p. 189 | happy |
| FR-084 | Arcana MUST cover the topics spells, magic items, eldritch symbols, magical traditions and planes of existence, and the creature types Aberration, Construct, Elemental, Fey and Monstrosity. | SRD 5.2.1, "Study [Action]", Areas of Knowledge, p. 189 | happy |
| FR-085 | History MUST cover the topics historic events, historic people, ancient civilizations and wars, and the creature types Giant and Humanoid. | SRD 5.2.1, "Study [Action]", Areas of Knowledge, p. 189 | happy |
| FR-086 | Investigation MUST cover the topics traps, ciphers, riddles and gadgetry, and no creature type. | SRD 5.2.1, "Study [Action]", Areas of Knowledge, p. 189 | happy |
| FR-087 | Nature MUST cover the topics terrain, flora and weather, and the creature types Beast, Dragon, Ooze and Plant. | SRD 5.2.1, "Study [Action]", Areas of Knowledge, p. 189 | happy |
| FR-088 | Religion MUST cover the topics deities, religious hierarchies, religious rites, holy symbols and cults, and the creature types Celestial, Fiend and Undead. | SRD 5.2.1, "Study [Action]", Areas of Knowledge, p. 189 | happy |
| FR-089 | Every `CreatureType` (the fourteen) MUST be covered by exactly one Study skill: the sets of FR-084 to FR-088 partition them. | SRD 5.2.1, "Creature Type", p. 179; "Study [Action]", Areas of Knowledge, p. 189 | unhappy (boundary) |
| FR-090 | Every `KnowledgeTopic` (twenty-one) MUST be covered by exactly one Study skill, and `SkillFor(subject)` MUST return it. | SRD 5.2.1, "Study [Action]", Areas of Knowledge, p. 189 | unhappy (boundary) |
| FR-091 | A skill whose ability is not Intelligence MUST be `InvalidStudySkill` with no subjects (`SubjectsOf` empty); an Intelligence skill MUST be `SuggestedStudySkill` carrying its subjects. | SRD 5.2.1, "Study [Action]", p. 189; "Skills" table, p. 9 | unhappy (invalid input) |

Tests for 4.12: one per topic and creature type listed in FR-084 to FR-088 (including that Investigation has no `StudyCreature`); the partition rows (all fourteen types and all twenty-one topics map to exactly one skill, FR-089, FR-090); an Intelligence skill is `SuggestedStudySkill`, and Perception and Athletics are `InvalidStudySkill` with no subjects (FR-091); `StudyRules.ability` (FR-083).

### 4.13 Events, Truesight queries, factories and Hide details

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-092 | `HidingEvent` MUST be a `closed class` implementing `CombatEvent` (`combat-foundations.md` FR-015), the only sub-family through which `CreatureHid` and `HidingEnded` implement it. This spec owns all three. | `combat-foundations.md` FR-015, FR-016, FR-020; `shared-types-and-ownership.md` FR-019 | happy |
| FR-093 | `CreatureHid.Kind()` MUST return the constant `"creature_hid"` and `HidingEnded.Kind()` `"hiding_ended"`: `[a-z][a-z0-9_]*`, unique across the package, not derived from `fmt` or reflection. | `combat-foundations.md` FR-017 | unhappy (name collision) |
| FR-094 | `combatants` of both events MUST be the defensively copied list `[hider]`. Both MUST be immutable (unexported fields) and comparable by value including `Kind()` and `Combatants()`; a `CreatureHid` never equals a `HidingEnded`. | `combat-foundations.md` FR-018, FR-019 | unhappy (immutability) |
| FR-095 | Truesight queries MUST take a `TruesightScene(DistanceFeet, senses)` plus explicit facts: `IllusionAppearance(scene, {required bool DeceivesSight})`, `IllusionSave(scene, {required bool DeceivesSight})` (`DeceivesSight` is `IllusionTraits.DeceivesSight`, `magical-effects.md` FR-043, documentation only: nothing is imported), `TrueForm(scene, {required bool TransformedByMagic, required SightRelation ObserverSeesIt})`, `EtherealView(scene)`. `DeceivesSight` and `TransformedByMagic` stay booleans: each is an external predicate with no named state here (`doc/rules/architecture.md` §3). An illusion with `DeceivesSight` false MUST be `AsCreated` and `RollNormally` whatever the range. `TrueForm` MUST be `discerned` only when within range, `TransformedByMagic` and `ObserverSeesIt` is `CanSee`. | SRD 5.2.1, "Truesight", p. 190; "Illusions", p. 183 | unhappy (state error) |
| FR-096 | `NewSightScene` and `NewTruesightScene` MUST validate `DistanceFeet` at least 0 and return an error matching `srderr.ErrInvalidInput` otherwise. | `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-097 | `NewSenseRange(sense, rangeFeet)` and `NewSpaceVisibility(light, darknessKind, otherObscurement)` MUST return `(T, error)` and never panic for any input: a `*SenseRangeError` or `*SpaceVisibilityError` (each carrying a typed `Reason` enum, `SenseRangeReason` and `SpaceVisibilityReason`, and matching `srderr.ErrInvalidInput`) if and only if the arguments are invalid. `MustSenseRange` and `MustSpaceVisibility` panic with that same error. The scene constructors follow FR-096. | `doc/rules/architecture.md` §6; `errors-and-results.md` FR-025, FR-026, FR-035 | unhappy (invalid input) |
| FR-098 | `HideScene` MUST carry the raw `Obscurement` of the hider's space (`SpaceVisibility.obscurement`), not a per-enemy value; what each enemy perceives reaches the rules as that enemy's `LineOfSight`. | SRD 5.2.1, "Hide [Action]", p. 183 ("while you're Heavily Obscured"); D-20 | unhappy (rule interaction) |
| FR-099 | `ResolveForcedFailure(HideAvailable, ForcedFailureReason)` MUST return `HideNotRolled(reason)` with no `HiddenState` and no event, for a check `d20.Resolver.Resolve` resolved as `d20.ForcedFailure` (no total). | `resolve-d20-test-use-case.md` (`d20.ForcedFailure`); SRD 5.2.1, "Hide [Action]", p. 183 | unhappy (state error) |

#### Cases 4.13a: events, Truesight queries and factories

| Label | Input | Expected | Requirement | Path |
|---|---|---|---|---|
| kinds | the two events | both implement `CombatEvent` through `HidingEvent`; `creature_hid`, `hiding_ended`, distinct | FR-092, FR-093 | unhappy (name collision) |
| combatants, equality | `CreatureHid(hider, 16)`, twice | `[hider]` (mutation is a copy); equal (`==`) | FR-094 | unhappy (immutability) |
| different events differ | `CreatureHid(hider, 16)`, `HidingEnded(hider, AttackRoll)` | not equal | FR-094 | unhappy (boundary) |
| non-visual illusion, in range | `DeceivesSight` false, 10 ft, Truesight 30 | `AsCreated`, `RollNormally` | FR-095 | unhappy (boundary) |
| transformed, not seen; not transformed, seen | `TransformedByMagic` true, `CannotSee`; false, `CanSee` | `NotDiscerned` | FR-095 | unhappy (state error) |
| negative distance | `TruesightScene(-1, ...)` | an invalid-input error (`srderr.ErrInvalidInput`) | FR-096 | unhappy (invalid input) |
| `NewSenseRange`, `NewSpaceVisibility` | `SenseRange` Darkvision 60; -1; `SpaceVisibility` Darkness, magical; Dim Light, magical | no error; `*SenseRangeError`; no error; `*SpaceVisibilityError` | FR-097 | happy; invalid input |

### 4.14 Combined senses, defensively copied values and activity effects

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-100 | Every collection a function returns, and every collection a value object holds, MUST be a defensive copy, and a constructor MUST copy what it is given. This covers `AttemptConditions`, `ImpliedConditions`, `Circumstances`, `SuggestedSkills`, `SubjectsOf`, `SenseRanges`, `HidingChange.events`, `HideSucceeded.events`, `HideAvailable.DiscernedSightings`, the `HideBlocked` lists, the enemy list and cover collection of `HideScene`, and the condition sets of `SightScene`. Mutating one is a copy (mutating it leaves the value unchanged); mutating the caller's own collection after construction MUST NOT change the value or an assessment already made. | `doc/rules/architecture.md` §3, §4; `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-101 | `EffectiveObscurement(space, distance, ranges)` MUST account for every special sense, taking the least severe of: (a) the more severe of `ObscurementOf(PerceivedLight)` and `OtherObscurement` (FR-026, Darkvision applied); (b) when Truesight is `WithinRange` and the light is `darkness` (either kind), `OtherObscurement` alone (FR-040, FR-041); (c) when Blindsight is `WithinRange`, `none` (D-23). Tremorsense never changes it (FR-038). `SightCheckEffect(EffectiveObscurement(...), kind)` is the effect on a check made to see for every observer, so a Truesight observer in Darkness within range gets `unaffected`. | SRD 5.2.1, "Truesight", "Tremorsense", p. 190; "Blindsight", p. 177; "Darkvision", p. 180; "Heavily Obscured", p. 182; D-23 | unhappy (rule interaction) |
| FR-102 | `HideRules.EffectOf(HidingActivity)` MUST return `StopsBeingHidden` for `SoundLouderThanWhisper`, `FoundByEnemy`, `AttackRoll` and `SpellWithVerbalComponent`; `RemainsHidden` for `SoundNoLouderThanWhisper`, `FoundByNonEnemy` and `SpellWithoutVerbalComponent`. `ApplyActivity` MUST be defined in terms of it: on Hidden, `StopsBeingHidden` gives NotHidden and one `HidingEnded`, `RemainsHidden` leaves the status with no event; on NotHidden, unchanged with no event (FR-071). | SRD 5.2.1, "Hide [Action]", p. 183 | happy |
| FR-103 | `SightScene` MUST carry the observer's and target's effective `ConditionKind` sets, not booleans. `CanSee` MUST treat the observer as Blinded exactly when `blinded` is in the observer's set and the target as Invisible exactly when `invisible` is in the target's set; every other kind is ignored. | SRD 5.2.1, "Blinded", p. 177; "Invisible", p. 184; `doc/rules/architecture.md` §3 | unhappy (rule interaction) |

#### Cases 4.14a: `EffectiveObscurement` and its effect with several senses

| Label | Space | Senses | Distance | Effective obscurement | `SightCheckEffect` for `PerceptionToSee` | Requirement | Path |
|---|---|---|---|---|---|---|---|
| Darkvision and Truesight, in both | Darkness | Darkvision 60, Truesight 30 | 20 | none | unaffected | FR-101 | unhappy (rule interaction) |
| only Darkvision reaches | Darkness | Darkvision 60, Truesight 30 | 40 | lightly | disadvantage | FR-101, FR-026 | unhappy (rule interaction) |
| magical Darkness, both in range | Darkness (magical) | Darkvision 60, Truesight 30 | 20 | none | unaffected | FR-101, FR-040 | unhappy (rule interaction) |
| magical Darkness, Truesight out | Darkness (magical) | Darkvision 60, Truesight 30 | 40 | heavily | AutomaticFailure | FR-101, FR-025 | unhappy (rule interaction) |
| Darkness, Blindsight one beyond | Darkness | Blindsight 30 | 31 | heavily | AutomaticFailure | FR-101, FR-032 | unhappy (boundary) |
| Tremorsense only | Darkness | Tremorsense 60 | 10 | heavily | AutomaticFailure | FR-101, FR-038 | unhappy (rule interaction) |

### 4.15 Withdrawn ids

| Id | Requirement |
|---|---|
| FR-001 | Withdrawn: moved to content-schema-and-decoder.md (FR-133) |
| FR-002 | Withdrawn: moved to content-schema-and-decoder.md (FR-135) |
| FR-016 | Withdrawn: moved to content-schema-and-decoder.md (FR-136) |

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

Behaviour lives in package `combat` (suggested files `sight.go`,
`special_senses.go`, `hiding.go`, `search.go`, `study.go`). All types
are immutable value objects with unexported fields, synchronous and
deterministic (`doc/rules/architecture.md` §3); nothing here takes a `ctx`
or starts a goroutine.

- **Failures.** Expected outcomes are results or enum values (`CannotSee`,
  `HideBlocked`, `HideCheckFailed`, `InvalidSearchSkill`); untrusted input
  goes through the closed assessments and the `NewX` constructors (FR-097), which return
  `(T, error)` with an error matching `srderr.ErrInvalidInput`; a `MustX`
  helper panics with that error and is for literals and tests only (`doc/rules/architecture.md` §6;
  `errors-and-results.md` FR-025, FR-026).
  No port, so no infrastructure failure.
- **Reused, not redefined:** `Skill`, `CreatureType`, `ConditionKind`, the light
  and sense enums (`content`); `Ability`; `RollCircumstance`; `DifficultyClass`,
  `d20.Outcome`, `ForcedFailureReason` (`d20`); `LineOfSight`,
  `SightRelation` (`conditions.md`); `CombatantID`, `CombatEvent`. Not imported
  (same or higher rank, `shared-types-and-ownership.md` FR-056): `Cover`
  (`attacks.md`), `StandardAction`, `IllusionTraits.DeceivesSight` (a boolean).

## 6. Interface

- **No use case, port, driven dependency or orchestrator**: domain services
  over values (D-1; `combat-action-resolution.md` Q-37 is a reference only).
- **Signatures** (static, pure; every `XRules` is a zero-size struct with value-receiver methods, and `->` reads "returns"): `LightRules.ObscurementOf(LightLevel) Obscurement`,
  `SightCheckEffect(Obscurement, SightCheckKind) SightCheckEffect`,
  `AttemptConditions(Obscurement) []ConditionKind`,
  `Circumstances(SightCheckEffect) []RollCircumstance`.
  `SpecialSenseRules.Reach(SpecialSense, int, SenseRanges) SenseReach`,
  `PerceivedLight(SpaceVisibility, int, SenseRanges) PerceivedLight`,
  `EffectiveObscurement(SpaceVisibility, int, SenseRanges) Obscurement`,
  `Tremorsense(SenseReach, TremorsenseContact, DetectedThing) TremorsenseResult`,
  `IllusionAppearance`, `IllusionSave`, `TrueForm` and `EtherealView`
  (each returns the type of its name; FR-095). `SightRules.CanSee(SightScene) SightRelation`.
  `HideRules.Assess(HideScene) HideAssessment`,
  `Resolve(HideAvailable, total int, outcome d20.Outcome) HideResolution`,
  `ResolveForcedFailure(HideAvailable, ForcedFailureReason) HideNotRolled`,
  `EffectOf(HidingActivity) HidingEffect`,
  `ApplyActivity(HidingStatus, HidingActivity, CombatantID) HidingChange`,
  `AttemptToFind(HidingStatus, CombatantID, HidingFinder, int) HidingChange`,
  `ImpliedConditions(HidingStatus) []ConditionKind`.
  `SearchRules.Ability() Ability`, `SuggestedSkills(SearchTarget) []Skill`,
  `Assess(Skill) SearchSkillAssessment`, `CanReveal(SearchProximity) SearchReveal`.
  `StudyRules.Ability() Ability`, `SkillFor(StudySubject) Skill`,
  `SubjectsOf(Skill) []StudySubject`, `Assess(Skill) StudySkillAssessment`.
  None takes a `ctx`; none can fail except through the constructors of FR-097.
- **Hide flow** (the host's): `HideRules.Assess(scene)`; if `HideAvailable`, roll
  `HideCheckSpec` through `d20.Resolver.Resolve`; a rolled check goes to `Resolve`,
  a forced failure to `ResolveForcedFailure`; merge `ImpliedConditions` into the
  hider's effective conditions; later activity goes to `ApplyActivity`; the
  host publishes the `events`.
- **Seeing flow:** build a `SightScene`, call `CanSee`, pass the `SightRelation`
  to `conditions.md` (`IsValidTarget`) and `attacks.md` (FR-104); the reverse
  direction is a second call with roles swapped.
- **Exports.** Package `combat` exports the
  value objects, enums, closed results, `HidingEvent` and its events, and the
  six services, documented per `doc/rules/architecture.md` §4; results carry
  the operation prefix (§9). No test double or contract suite.

## 7. Observability

Silent by design: domain code only (`doc/rules/observability.md` §3). No srd5e
use case runs these rules (`combat-action-resolution.md` Q-37), so there is no
span; an invalid fact is an `srderr.ErrInvalidInput` error returned to the host's code, and
result values name the deciding rule (`HideBlock`, `SightRelation`,
`SightCheckEffect`, `HidingEnded.activity`). Nothing is counted or timed;
`ActionTaken` for Hide, Search and Study (`initiative-and-actions.md`) is
unchanged. What can go wrong: a host supplies a wrong fact or a negative
distance (rejected by the constructor, FR-096); a consumer monitors nothing from this spec;
nothing is emitted, so nothing is noisy. No `srd5e.outcome` value other than `success` or `failure`
(`shared-types-and-ownership.md` FR-041, `combat-foundations.md` FR-030). A
later Hide counter would use `srd5e.combat.result` (`combat-foundations.md`
FR-027). `doc/observability/signals.md` is not changed.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none | n/a | n/a | n/a | n/a |

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-003 to FR-015 | unit | 4.1a and the 4.2 matrix; FR-014 asserts a defensively copied set and no event. |
| FR-017 to FR-039, FR-104 | unit | 4.3a, 4.4a, 4.6a; Blindsight and FR-038 through 4.8a rows; FR-017 to FR-020 include the defensively copied `SenseRanges` view (add or remove on it is a copy), and the greater-range rule goes through the `SenseRanges` constructor; FR-039 is a note, no row. |
| FR-040 to FR-050 | unit | 4.7a and every row of 4.8a; FR-048 asserts the `IllusionSave` value, not a roll; FR-050 calls twice, mutates nothing. |
| FR-051 to FR-063, FR-105 | unit | 4.9a, 4.9b; FR-058 passes each assessment through a function typed `HideAvailable`; FR-063 is a note, no row. |
| FR-064 to FR-073, FR-102 | unit | 4.10a (`EffectOf` and `ApplyActivity` agree); FR-073 equality (`==` or `Equal`), defensively copied lists. |
| FR-074 to FR-091 | unit | The per-skill tests under 4.11 and 4.12; the 14-type and 21-topic partitions fail if an SRD type is added without a row. |
| FR-092 to FR-099 | unit, architecture | 4.13a, 4.9a, 4.9b; the architecture test of `combat-foundations.md` FR-016, FR-017 lists `HidingEvent` as the sole sub-family and checks the unique `kind` constants. |
| FR-100, FR-101, FR-103 | unit | 4.14a, the condition rows of 4.8a, the first rows of 4.7a and 4.4a. FR-100: mutation (add, remove, clear) of every returned or held collection is a copy (mutating it leaves the value unchanged) (`AttemptConditions`, `ImpliedConditions`, `Circumstances`, `SuggestedSkills`, `SubjectsOf`, `SenseRanges`, `HidingChange.events`, `HideSucceeded.events`, `DiscernedSightings`, `HideBlocked` lists, `HideScene` enemy and cover collections, `SightScene` condition sets); mutating the caller's own collection after construction changes nothing (the assessment or `CanSee` equals the one made before). |

Unit plus the architecture test; no application or contract test. Scenario
tests in `scenarios/` chain Hide, a roll, an attack and the
location-given-away flag (`attacks.md` FR-037; FR-068). No doubles
(`doc/rules/testing.md` §3). Mutation testing (§8): `gomu` on the changed lines in
`combat`, score at least 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors; the boundary rows kill `<` and
`<=` mutants.

Deliberately not covered: FR-039 and FR-063 (path n/a, note) - notes, not requirements.

## 9. Open questions and decisions

Nothing is open. Settled (decision ids cited in the requirements):

- D-6: `Invisible` is never stored in the `ConditionSet`; `ImpliedConditions` (FR-062) is a derived set the host unions into the effective `ConditionKind` sets it gives `SightScene` and `conditions.md` queries, so a spell's Invisible is untouched when hiding ends (Hide grants Invisible "while hidden", p. 183). No change to `conditions.md`. D-9: FR-025 applies "Darkvision can't see through it" (p. 122) to every `magical` Darkness; a creature printed "unimpeded by magical Darkness" is modelled by the host passing `nonmagical` for that observer's query (no sense qualifier; design basis: `SpaceVisibility` is per query).
- D-11: Truesight is stated for Darkness only (p. 190), so Dim Light keeps Lightly Obscured (FR-040); a Blinded observer is not helped (FR-047, Blindsight alone says "even if you have the Blinded condition", p. 177); colors are not modelled. D-12: the `CanSee` order (FR-049) lets each stated override work; Heavily Obscured makes the observer Blinded only "while trying to see" (p. 182), so it is per attempt (FR-014): its attacks on what it cannot see use `TargetUnseen` (`attacks.md` FR-034), and attacks against it get no Advantage from this spec (interpretation: the condition is scoped to the attempt, not to the creature).
- D-15: Hide while already Hidden: `Assess` ignores the current status and a new success replaces the state and `FindDC` (SRD silent; the action is repeatable). An enemy whose Blindsight or Truesight is `WithinRange` of the hider sees an Invisible hider (p. 177, p. 190), so that enemy "finds you": the host passes `FoundByEnemy` (FR-066) (interpretation: sight through Invisible defeats the hiding benefit). D-19: the Thief's Stealth Attack (p. 64) is a class feature (`class-rogue.md` FR-064): it keeps the Invisible condition if the turn ends behind Three-Quarters or Total Cover, so the host skips `ApplyActivity(AttackRoll)` for that attack and keeps `HidingStatusHidden`; no new `HidingActivity` (design basis: no consumer or class logic in this spec).
- D-1: no use case orchestrates these rules; rank 23 (`shared-types-and-ownership.md` FR-044, FR-056, table 4.11). D-2: `combat-foundations.md` FR-001 owns `CombatantID`. D-3: `DifficultyClass` and `d20.Outcome` win (`shared-types-and-ownership.md` FR-002, FR-004); no-total checks use FR-099. D-4: light and sense vocabularies belong to `content-schema-and-decoder.md` (FR-133, FR-136). D-5: `SightRelation`, `LineOfSight` reused from `conditions.md`.
- D-7: `AttemptToFind` takes any Wisdom (Perception) total; the host picks active or passive (`derived-stats.md` FR-020). D-8: the more severe obscurement wins (FR-007). D-10: a distance equal to the range is within; the greater range wins (FR-018, FR-019).
- D-13: no enemies satisfies line of sight; `DiscernedSightings` holds only enemies the hider can see (FR-055, FR-057). D-14: the host reports Total Cover and any other blocker as `blocked`. D-16: Search accepts Animal Handling off table (FR-079). D-17: `HideBlocked` ends the attempt without a roll, the action still spent (FR-058).
- D-18: Lightly Obscured follows the glossary (p. 184, p. 182) and applies to the space the host passes; a host preferring p. 11 passes the observer's space. D-20: `HideScene` carries the raw obscurement (FR-098). D-21: `KnowledgeTopic` splits two SRD phrases so each topic has one skill (FR-090): `HistoricEvents`, `HistoricPeople`, `ReligiousHierarchies`, `ReligiousRites`. D-22: variant names satisfy `shared-types-and-ownership.md` FR-017, FR-021. D-23: the least severe candidate wins (FR-101); Blindsight in range gives `none`: Blindsight sees without relying on physical sight (p. 177) and Lightly Obscured Disadvantage is for checks that rely on sight; the SRD does not say so in words (package decision).

## 10. Changelog impact

`CHANGELOG.md`, `Unreleased`, **Added** (not breaking): `SpaceVisibility`,
`LightRules`, `SpecialSenseRules`, `SightRules.CanSee`, `HideRules` with its
state, status, change and events (`CreatureHid`, `HidingEnded`), `SearchRules`,
`StudyRules`. No change to `conditions.md` (D-6, D-12).
