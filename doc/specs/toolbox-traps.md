# Traps

| | |
|---|---|
| Status | draft |
| Context | `toolbox` |
| Roadmap phase | 5 |
| Depends on | `toolbox-hazard-effects.md`, `content-schema-and-decoder.md`, `content-source-port.md`, `d20-tests.md`, `modifiers-and-advantage.md`, `ability-scores.md`, `derived-stats.md`, `dice-and-rolls.md`, `hit-points-and-damage.md`, `conditions.md`, `environment-hazards-and-objects.md`, `initiative-and-actions.md`, `hiding-searching-and-senses.md`, `combat-foundations.md`, `observability-ports.md`, `errors-and-results.md` |

## 1. Purpose and scope

The Gameplay Toolbox entry **Traps** (SRD 5.2.1, pp. 199-201): the **Parts of a Trap** (severity and levels, trigger, duration, reset), the caution about characters below a trap's level range, and the eight example traps (Collapsing Roof, Falling Net, Fire-Casting Statue, Hidden Pit, Poisoned Darts, Poisoned Needle, Rolling Stone, Spiked Pit) with their DCs, damage, Detect and Disarm steps, Escape, Destroy, Slow and Set options and **At Higher Levels** values for levels 5-10, 11-16 and 17-20.

A trap is an immutable `TrapDefinition` whose values are banded by level (`TrapLevelBand`); a definition and a `CharacterLevel` give a `TrapProfile` (severity and effect). The effect of roof, statue, net, needle and stone is a `SaveGatedEffect` of `toolbox-hazard-effects.md`, which owns the shared save-then-effect model (`SaveGatedEffect` and its parts, `ActiveSaveGatedEffect`, `SaveGatedConsequences`, `RollSaveGatedDamage`, `ResolveSaveGatedEffect`, `ToolboxEvent`, the `toolbox` signal constants); this spec never defines a second one (Q-3, Q-17). `toolbox-poison.md` (rank 41, a peer) is cited in prose only (Q-18).

**Delivery** (`shared-types-and-ownership.md` FR-045, FR-047): the eight examples are named catalogue entries, so they are **content**: kind `trap`, root array `traps`, `TrapID` owned by `content-schema-and-decoder.md` FR-114, JSON in `content/data/traps.json`, embedded like every kind with `go:embed` and decoded by the one content decoder. This spec owns the data-only `TrapEntry` (a `content` type naming no `toolbox` type); `toolbox` converts an entry to a `TrapDefinition` (FR-091 to FR-093). The level bands and scaling pattern stay constants of the `toolbox` domain. `TrapState` and `TrapLifecycle` track armed, reset-pending, active and inert traps.

One use case, `ResolveTrapEffect`, activates a trap and rolls its damage. It does **not** roll saving throws and does **not** apply damage or conditions. A use case never calls another use case (`shared-types-and-ownership.md` FR-044; `environment-hazards-and-objects.md` D-1), so the host resolves each target's save through `d20` and passes the finished `SaveGatedSave` in; the use case returns the shared `SaveGatedConsequences` and `ActiveSaveGatedEffect`, and the host passes them to `ApplyDamage` and `ApplyConditionToSet`. The pits do not re-implement the fall: the host calls `ResolveFall` and `LandAfterFall` with the pit's `FallDistance` (FR-072). State in, new state and events out (`doc/rules/architecture.md` §4). The engine reads no clock and no map: trip wires crossed, creatures in a Cone and elapsed time arrive as values.

**Notation.** Methods, struct fields and enum values named in this spec are exported Go identifiers in PascalCase. Remaining lowerCamelCase values (`pressurePlateEntered`, `trapNotArmed`, `slowedToStandstill` and similar) are the reference spelling of enum values and MUST be exported PascalCase constants in Go (`TrapNotArmed`, `SlowedToStandstill`).

**Domain methods versus use cases.** `TrapLifecycle`, `TrapCheckStep.Assess`, `TrapSlowOption.Assess`, `TrapSetOption.Assess` and the pure domain service `TrapResolver` (target validation, size exemption, darts expansion, per-target mapping, contacts; FR-095) roll nothing, call no port and emit no telemetry (`doc/rules/observability.md` §3). Their sequencing errors are result values (FR-021 to FR-025, FR-048, FR-055, FR-057). Activation, the one transition an operator needs to see, runs inside the observed use case (FR-059).

**Ownership** (`shared-types-and-ownership.md` FR-048). This spec owns the mechanics of the Traps entry: bands, lifecycle, check steps, options, `ResolveTrapEffect`. It defers the fall (`environment-hazards-and-objects.md` FR-015 to FR-017, FR-023), the Search and Study actions (`hiding-searching-and-senses.md`) and the shared model. No requirement of another spec is withdrawn: `toolbox-hazard-effects.md` holds trap mechanics only as non-normative notes (section 5.1 rows "Poisoned Needle", "Poisoned Darts", "Falling Net"; section 5.2 row "Huge or larger automatic success, `TrapSaveForDamage`, `TrapSaveForCondition`") that become pointers here in its own reconcile. `TrapEvent` is this spec's closed interface family under `ToolboxEvent` (FR-094; `toolbox-hazard-effects.md` FR-086).

**Out of scope:**

- The introduction's advice ("Traps should be used sparingly", "Traps that are undetectable and inescapable are rarely fun", p. 199): guidance to the Game Master, not a rule.
- **Designing new traps** from the scaling pattern: the SRD gives no formula ("You can scale the trap for higher levels by increasing the damage and the save DC"), only the eight tables. A house trap is a `TrapDefinition` built by the host.
- Search and Study resolution (`hiding-searching-and-senses.md`), saving throws and checks (`d20-tests.md`, `resolve-d20-test-use-case.md`), applying damage (`hit-points-and-damage.md`) and conditions (`conditions.md`), Initiative and turn order (`initiative-and-actions.md`, `encounter-aggregate.md`), movement and Difficult Terrain, and falling (Falling [Hazard], p. 182; `environment-hazards-and-objects.md` owns `ResolveFall` and `LandAfterFall`).
- The spells the text names (Detect Magic, Arcane Lock, Knock, Spider Climb): spell content (`content-spells.md`, `spell-effects.md`); trap data holds `SpellID`s only.
- The Net's statistics (Adventuring Gear, p. 98: AC 10, 5 HP, Immunity to Bludgeoning, Poison and Psychic damage), the Iron Spike and Thieves' Tools: `content-equipment.md` owns them; this spec holds `ItemID`s, checked by whole-set reference validation (FR-092, Q-11).
- Game Master judgement the text leaves open: cracks added to a pit wall, "or a similar object", "similar magic", redirecting the stone, curving passages (p. 201).
- Anything not in SRD 5.2.1. Nothing core to Traps (pp. 199-201) was left out; the SRD gives no trap cost, Challenge Rating or XP, so none is modelled.

## 2. Source

PDF page numbers equal printed page numbers (page "199" is PDF page 199; text layer compared with page images).

- `SRD 5.2.1, "Traps", introduction, p. 199`; `"Parts of a Trap"` (Severity and Levels, Trigger, Duration) and the caution paragraph after it, p. 199
- `SRD 5.2.1, "Example Traps", "Collapsing Roof", p. 199` (whole entry incl. At Higher Levels); `"Falling Net", p. 199`; `"Fire-Casting Statue", pp. 199-200`; `"Hidden Pit", p. 200`; `"Poisoned Darts", p. 200`; `"Rolling Stone", p. 201`; `"Spiked Pit", p. 201`
- `SRD 5.2.1, "Poisoned Needle", pp. 200-201` (the failed-save sentence breaks across the page: "has the Poisoned" ends p. 200, "condition for 1 hour. On a successful save, the creature takes half as much damage only." opens p. 201)
- `SRD 5.2.1, "Falling [Hazard]", p. 182` (1d6 Bludgeoning per 10 feet fallen, at most 20d6; used only for the consistency test of FR-038); `"Adventuring Gear", "Net", p. 98` (named by the Falling Net; only for Q-11)
- `SRD 5.2.1, "Damage against Multiple Targets", p. 16` and `"Half Damage", p. 16` (via `hit-points-and-damage.md` FR-014, FR-100; `toolbox-hazard-effects.md` FR-034, FR-043)
- Package rules: `doc/rules/architecture.md` (§2, §3, §4, §6, §9), `doc/rules/observability.md` (§2-§7), `doc/rules/testing.md` (§3-§5).

## 3. Vocabulary

Package names have no SRD source and add no rules.

| SRD term | Type | Notes |
|---|---|---|
| Trap | `TrapDefinition`, `TrapID` | `TrapID` (snake_case, `shared-types-and-ownership.md` FR-007, e.g. `collapsing_roof`) is owned by `content-schema-and-decoder.md` FR-114, FR-129. |
| Nuisance, Deadly | `TrapSeverity` (`nuisance`, `deadly`) | p. 199. |
| "Levels 1-4", "5-10", "11-16", "17-20" | `TrapLevelBand`, `CharacterLevel` (`derived-stats.md` FR-057) | Levels are character levels (Q-7). A host converts an `int` with `NewCharacterLevel` (FR-004). |
| Trap as content data | `TrapEntry`, `TrapEntryBand`, `TrapEntryProblem`, `NewTrapDefinition`, `TrapDefinitionResult` (`TrapDefinitionBuilt`, `TrapDefinitionInvalid`) | FR-091 to FR-093. |
| A trap's values at one level | `TrapProfile`, `TrapBandProfile`, `TrapSaving` (`TrapSaveRequirement(ability, difficultyClass)`, `TrapNoSave`), `TrapLevelFit` (`withinRange`, `belowTrapLevels`, `aboveTrapLevels`), `profileFor` results (`TrapProfileResolved`, `TrapProfileBelowTrapLevels`, `TrapProfileAboveTrapLevels`) | `TrapProfile.saving` is the save the host rolls (FR-085); fit FR-060. |
| Trigger | `TrapTrigger`, `TrapTriggerKind`, `TrapTriggerMoment` | FR-013, FR-014. |
| Duration | `TrapDuration` (closed interface) | FR-015. |
| "resets", inert after activation (p. 199) | `TrapState` (`TrapArmed`, `TrapResetPending`, `TrapActive`, `TrapActiveRolling`, `TrapInert`), `TrapLifecycle`, `TrapTransitionResult` (`TrapTransitioned`, `TrapTransitionRejected`), `TrapTransitionRejection`, `TrapEndReason` | Section 4.4. |
| Detect and Disarm | `TrapCheckStep`, `TrapCheckPurpose`, `TrapCheckAction` (`search`, `study`, `action`), `TrapFeature`, `TrapDiscovery`, `TrapStepAssessment` (`TrapStepSucceeded`, `TrapStepFailed`, `TrapStepPrerequisiteMissing`), `TrapStepConsequence` (`none`, `triggersTrap`) | The Search and Study actions are defined in `hiding-searching-and-senses.md`; `action` is an unspecified action (p. 201, p. 200) with no ability restriction (FR-040). |
| Ways to stop a trap without a check | `TrapNeutralisation(method, prerequisite, itemID, spellID)`, `TrapNeutralisationMethod` | Cut the wire, wedge an Iron Spike, plug holes, deface the glyph, Knock, Arcane Lock. |
| Escape | `TrapEscapeOption`, `TrapEscapeRequirement` (`TrapEscapeClimbSpeed`, `TrapEscapeClimbingGear`, `TrapEscapeMagic(spellID)`), `TrapEscapeCheck(ability, skill, difficultyClass)` | Pits: Climb Speed, gear, Spider Climb. Net: Strength (Athletics) by action. |
| Destroy the Net / Stone, Slow the Stone, Set the Trap | `TrapDestroyOption`, `TrapSlowOption`, `RollingMotion`, `TrapSetOption` | The stone is an `ObjectStatus` (FR-052). |
| Effects | `TrapEffect` (`TrapSavedEffect`, `TrapDartsEffect`, `TrapPitEffect`), `TrapTargeting` (`EachInArea`, `SingleTarget`, `PerContact`), `TrapSizeExemption`, `TrapAreaOfEffect`, `TrapAftermath`, `TrapAftermathEffect` | Section 4.5; FR-039. |
| Huge or larger | `CreatureSize` (`content`), `TrapSizeExemption` | The set {Huge, Gargantuan}; handed to the shared planner as `SaveGatedExempt` (FR-068). |
| Initiative, "+8 bonus"; contacts | `Modifier` (kernel), `TrapInitiativeData`, `TrapInitiativeOutcome` (`TrapInitiativeRequired`, `TrapNoInitiative`), `TrapMovementRule`, `TrapContactEvent` | Q-13; FR-058, FR-088. |
| Activation | `ResolveTrapEffect`, `ResolveTrapEffectCommand`, `ResolveTrapEffectResult` (`TrapEffectResolved`, `TrapEffectRejected`), `TrapEffectRejection`, `TrapRejectionCulprit` (`TrapCulpritCombatant`, `TrapNoCulprit`), `TrapTarget`, `TrapTargetResolution` (`TrapTargetAutoSucceeded`, `TrapTargetSaved`, `TrapTargetFailed`, `TrapTargetFell`) | Section 4.8. |
| Rules of one activation | `TrapResolver`, `TrapResolutionPlan` (`TrapPlanRejected`, `TrapPlanReady`), `TrapDiceRequest` (`TrapRollNext`, `TrapRollsComplete`), `TrapResolution` | FR-095. |
| Events | `TrapEvent`, `TrapActivated`, `TrapReset`, `TrapEnded` | FR-094. |
| Saving throw, DC; damage; conditions | `DifficultyClass`, `Ability`, `SaveGatedSave`, `CheckResult`, `D20TestOutcome`; `Damage`, `DamageAdjustment` (`HalfDamage`), `DamageType`, `SaveGated*`; `ConditionKind` | Used by reference from `d20`, the kernel, `content`, `hit-points-and-damage.md` and `toolbox-hazard-effects.md`. `SaveGatedDurationUntilEnded` and `SaveGatedDurationFor(SaveGatedFixedTime(1, DurationUnit.hours))` carry "until it escapes" and "for 1 hour"; the host ends them with `EndSaveGatedEffect`. `DurationUnit`: `content-schema-and-decoder.md` (shared-types-and-ownership.md FR-065). |

## 4. Requirements

Stable ids. Never renumber; new ids go at the end. "Interpretation" marks a reading of a silent SRD, settled in section 9. Pages are printed pages. A table cell listing several cases is one test per case.

### 4.1 Severity, levels and bands

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `TrapSeverity` MUST have exactly two values, `nuisance` and `deadly`. | SRD 5.2.1, "Parts of a Trap", Severity and Levels, p. 199 | happy |
| FR-002 | `TrapLevelBand` MUST have exactly four values in ascending order: `levels1to4` (1-4), `levels5to10` (5-10), `levels11to16` (11-16), `levels17to20` (17-20), each exposing its lowest and highest `CharacterLevel`. | SRD 5.2.1, "Example Traps" headings and At Higher Levels tables, pp. 199-201 | happy |
| FR-003 | `TrapLevelBand.ForLevel(CharacterLevel)` MUST return the band of table 4.1a for every level, including both ends of each band. | SRD 5.2.1, "Collapsing Roof", At Higher Levels table, p. 199 | unhappy (boundary) |
| FR-004 | `TrapLevelBand` MUST take a `CharacterLevel` only and MUST NOT offer an `int` entry point or a second result vocabulary: a host converts an `int` with `NewCharacterLevel` (`derived-stats.md` FR-057 owns the 1 to 20 validation, including 0, 21, -1 and the extreme integers). | SRD 5.2.1, "Collapsing Roof", p. 199 (tables stop at 17-20); `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-005 | Every level 1 to 20 MUST fall in exactly one band: the four ranges are contiguous, do not overlap and cover 1 to 20. | SRD 5.2.1, "Example Traps", pp. 199-201 | unhappy (boundary) |

**4.1a `ForLevel`** (FR-003; p. 199; every row unhappy (boundary)). Invalid integers are rows of `NewCharacterLevel` (`derived-stats.md` FR-057); FR-004 is tested by the exported-name scan (FR-083).

| Levels | Band |
|---|---|
| 1, 4 | `levels1to4` |
| 5, 10 | `levels5to10` |
| 11, 16 | `levels11to16` |
| 17, 20 | `levels17to20` |

### 4.2 Definition and profile lookup

A `TrapDefinition` holds, per covered band, a `TrapBandProfile` (severity, effect), plus the trigger, duration, check steps, neutralisations and options.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-006 | `TrapDefinition` MUST be constructible only with at least one band profile and MUST return an error wrapping `srderr.ErrInvalidInput` for none. | `doc/rules/architecture.md` §3; SRD 5.2.1, "Parts of a Trap", p. 199 | unhappy (empty and minimal) |
| FR-007 | The covered bands MUST be contiguous; the constructor MUST return an error wrapping `srderr.ErrInvalidInput` for a gap (for example 1-4 and 11-16 without 5-10). | `doc/rules/architecture.md` §3; SRD 5.2.1, "Rolling Stone", p. 201 | unhappy (invalid input) |
| FR-008 | `profileFor(CharacterLevel)` MUST return `TrapProfileResolved(profile)` when the level's band is covered; the profile MUST carry the `TrapID`, the band, that band's severity and effect. | SRD 5.2.1, "Parts of a Trap", p. 199; At Higher Levels, pp. 199-201 | happy |
| FR-009 | `profileFor` for a level below the lowest covered band MUST return `TrapProfileBelowTrapLevels(lowestBand, severityAtLowestBand)` and no profile: the caution (advice to the Game Master, not a prohibition) that a trap that is a nuisance at one level range could be deadly to characters of a lower one. `ResolveTrapEffect` does not treat it as an error (FR-060). | SRD 5.2.1, "Parts of a Trap", caution paragraph, p. 199; "Rolling Stone", p. 201 | unhappy (boundary) |
| FR-010 | `profileFor` for a level above the highest covered band MUST return `TrapProfileAboveTrapLevels(highestBand)` and no profile. No example reaches it (all cover level 20); it applies to host-built traps. Interpretation (Q-12). | SRD 5.2.1, "Parts of a Trap", p. 199 (silent above the range); `doc/rules/testing.md` §4 | unhappy (boundary) |
| FR-011 | Severity MUST be stored per band. The Rolling Stone MUST be `deadly` at 11-16 and `nuisance` at 17-20; every other example keeps the severity of its "Levels 1-4" heading at every band it covers (FR-029 to FR-036). Interpretation (Q-6). | SRD 5.2.1, "Rolling Stone", p. 201; "Parts of a Trap", p. 199 | happy |
| FR-012 | `TrapDefinition`, `TrapBandProfile` and `TrapProfile` MUST be immutable values: equal when all fields are equal, unequal when any differs; exposed collections MUST be defensively copied views; mutating a collection passed to a constructor MUST NOT change the value. | `doc/rules/architecture.md` §3, §4; `doc/rules/testing.md` §4 | unhappy (immutability) |

**4.2a `profileFor` for the Rolling Stone** (covers 11-16 and 17-20; FR-008, FR-009, FR-011; pp. 199, 201).

| Level | Result | Severity | Req. | Path |
|---|---|---|---|---|
| 1, 4, 5, 10 | `TrapProfileBelowTrapLevels(levels11to16, deadly)` | n/a | FR-009 | unhappy (boundary) |
| 11 | `TrapProfileResolved` | `deadly` | FR-008, FR-011 | happy |
| 16 | `TrapProfileResolved` | `deadly` | FR-011 | unhappy (boundary) |
| 17, 20 | `TrapProfileResolved` | `nuisance` | FR-011 | unhappy (boundary) |

### 4.3 Trigger and duration

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-013 | `TrapTriggerKind` MUST have exactly six values: `tripWireCrossed`, `pressurePlateEntered`, `pitLidEntered`, `lockOpenedImproperly` (opens a lock improperly or fails to disarm the trap), `doorknobTurned` (named in "Parts of a Trap", used by no example) and `other` for a house-built trigger. The SRD says only "Examples of triggers include", so `other` carries the extension; the catalogue never uses it. | SRD 5.2.1, "Parts of a Trap", Trigger, p. 199; the examples, pp. 199-201 | happy |
| FR-014 | `TrapTriggerMoment` MUST have exactly two values, `FirstTimeOnATurn` and `StartsTurnThere`; a trigger carries a set of them (empty means no qualifier). The Fire-Casting Statue MUST carry both, the Poisoned Darts `FirstTimeOnATurn` only, no other example any. The Rolling Stone's contact qualifiers (p. 201) belong to its effect, not its pressure-plate trigger (FR-088). | SRD 5.2.1, "Fire-Casting Statue", pp. 199-200; "Poisoned Darts", p. 200; "Rolling Stone", p. 201 | happy |
| FR-015 | `TrapDuration` MUST offer exactly: `TrapInstantaneous`; `TrapInstantaneousWithReset(limit)` (unlimited or a limited number of activations); `TrapTimed(count, unit)` (`unit` the shared `DurationUnit`, restricted to `rounds`, `minutes` and `hours`; SRD p. 199); `TrapUntilDestroyedOrDispelled`; `TrapUntilEffectEnds`. The examples use only the first, second and fifth. | SRD 5.2.1, "Parts of a Trap", Duration, p. 199; "Rolling Stone", p. 201 | happy |
| FR-016 | `TrapTimed` MUST return an error wrapping `srderr.ErrInvalidInput` for a count below 1, and a limited reset for a limit below 1; 1 MUST be accepted. | `doc/rules/architecture.md` §3, §6; `doc/rules/testing.md` §4 | unhappy (invalid input) |

### 4.4 Lifecycle

`TrapLifecycle` is a stateless domain service over `TrapState`: `TrapArmed(activations)`, `TrapResetPending(activations)`, `TrapActive(activations)`, `TrapActiveRolling(activations, motion)` (`motion` a `RollingMotion`, never absent) and `TrapInert(activations)`. "An active state" means `TrapActive` or `TrapActiveRolling`. A new trap is `TrapArmed(activations: 0)`. Operations `Activate(definition, state)`, `StartOfNextTurn(state)` and `End(state, reason)` return the closed interface `TrapTransitionResult`: `TrapTransitioned(state, events)` or `TrapTransitionRejected(reason, state)`, `reason` a `TrapTransitionRejection` enum (`trapNotArmed`, `noResetPending`, `trapNotActive`, `trapNotRolling`, `trapNotInert`); the names `TrapNotArmed` etc. below are those values. `TrapSlowOption.Assess` and `TrapSetOption.Assess` return the same type. These are domain methods, not use cases (section 1).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-017 | `activate` on `TrapArmed` for `TrapInstantaneous` MUST return `TrapInert` and one `TrapActivated` event with the new count (1). | SRD 5.2.1, "Parts of a Trap", Duration, p. 199 | happy |
| FR-018 | `activate` on `TrapArmed` for `TrapInstantaneousWithReset` with an unlimited limit MUST return `TrapResetPending` with the count increased by 1 and one `TrapActivated` event, for every count (table 4.4a). | SRD 5.2.1, "Fire-Casting Statue", p. 200 | happy |
| FR-019 | `activate` on `TrapArmed` for a limited reset MUST return `TrapResetPending` while the new count is below the limit and `TrapInert` when it reaches it, with one `TrapActivated` event (table 4.4a; the Poisoned Darts' limit is 3). | SRD 5.2.1, "Poisoned Darts", p. 200 | unhappy (boundary) |
| FR-020 | `activate` on `TrapArmed` for `TrapTimed`, `TrapUntilDestroyedOrDispelled` or `TrapUntilEffectEnds` MUST return an active state with the count increased by 1 and one `TrapActivated` event: `TrapActiveRolling` with a `RollingMotion` at 60 feet when the definition has a slow option (the Rolling Stone, FR-054), else `TrapActive`. | SRD 5.2.1, "Parts of a Trap", Duration, p. 199; "Rolling Stone", p. 201 | happy |
| FR-021 | `activate` on any state other than `TrapArmed` MUST return the rejection `TrapNotArmed` carrying the unchanged state, with no events. | SRD 5.2.1, "Parts of a Trap", p. 199; `doc/rules/testing.md` §4 | unhappy (state error) |
| FR-022 | `startOfNextTurn` on `TrapResetPending` MUST return `TrapArmed` with the count unchanged and one `TrapReset` event. | SRD 5.2.1, "Fire-Casting Statue", "Poisoned Darts", p. 200 | happy |
| FR-023 | `startOfNextTurn` on `TrapArmed`, `TrapActive`, `TrapActiveRolling` or `TrapInert` MUST return the rejection `TrapNoResetPending` with the unchanged state and no events. An inert trap MUST NOT become armed. | SRD 5.2.1, "Parts of a Trap", p. 199; `doc/rules/testing.md` §4 | unhappy (state error) |
| FR-024 | `end(reason)` on an active state MUST return `TrapInert` and one `TrapEnded(reason)` event, `TrapEndReason` one of `stoppedByBarrier`, `slowedToStandstill`, `destroyed`, `dispelled`, `durationElapsed`. `destroyed` and `dispelled` are host-reported (Q-14). | SRD 5.2.1, "Rolling Stone", p. 201; "Parts of a Trap", Duration, p. 199 | happy |
| FR-025 | `end` on `TrapArmed`, `TrapResetPending` or `TrapInert` MUST return the rejection `trapNotActive` with the unchanged state and no events. | `doc/rules/testing.md` §4 | unhappy (state error) |
| FR-026 | No `TrapLifecycle` operation MUST alter the state passed in; equal inputs MUST give equal results. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (immutability) |

**4.4a Activation counts** (FR-017 to FR-021; one `activate` call per row; 1 event unless a rejection).

| Label | Duration | Before | Result | Req. | Source | Path |
|---|---|---|---|---|---|---|
| single use | instantaneous | armed, 0 | inert, count 1 | FR-017 | p. 199 | happy |
| unlimited reset, first | reset, unlimited | armed, 0 | reset pending, count 1 | FR-018 | p. 200 | happy |
| unlimited reset, 50th | reset, unlimited | armed, 49 | reset pending, count 50 | FR-018 | p. 200 | unhappy (boundary) |
| darts, first | reset, limit 3 | armed, 0 | reset pending, count 1 | FR-019 | p. 200 | unhappy (boundary) |
| darts, second | reset, limit 3 | armed, 1 | reset pending, count 2 | FR-019 | p. 200 | unhappy (boundary) |
| darts, third | reset, limit 3 | armed, 2 | inert, count 3 | FR-019 | p. 200 | unhappy (boundary) |
| limit of 1 | reset, limit 1 | armed, 0 | inert, count 1 | FR-019 | package design (FR-016) | unhappy (boundary) |
| rolling stone | until effect ends | armed, 0 | active, count 1 | FR-020 | p. 201 | happy |
| armed again | any | reset pending, active or inert | `TrapNotArmed`, 0 events | FR-021 | p. 199 | unhappy (state error) |

### 4.5 The example traps: identity and effect values

The bundled data MUST hold the eight examples in the SRD's alphabetical order as content entries of kind `trap` (FR-027, FR-091). `TrapDefinition` constructors validate and copy their collections (FR-006, FR-007, FR-012), so definitions are built at run time from decoded entries (FR-093); there is no code table. The values below are the requirement for the bundled data and the built definitions; one `TrapBandProfile` per band cell. The save DC is part of the effect where the SRD scales it.

**Effect model.** `TrapEffect` is a closed interface with three variants that wrap the shared model:

- `TrapSavedEffect(effect: SaveGatedEffect, targeting: TrapTargeting, sizeExemption: TrapSizeExemption?, area: TrapAreaOfEffect?)`: roof, net, statue, needle, stone. `TrapSizeExemption(minimumSize).Exempts(size)` is the trap layer's decision ("succeeds automatically if it's Huge or larger", net only); the resolver hands the shared planner `SaveGatedExempt` (`toolbox-hazard-effects.md` FR-076; FR-068, FR-095). `TrapAreaOfEffect(coneLengthFeet)` is the statue's Cone.
- `TrapDartsEffect{PerDart SaveGatedEffect; DartCount dice.Expression}`: `PerDart` holds one dart's damage, `DartCount` is `1d3`. `TrapDartsEffect.Expand(darts int)` (pure, FR-095) builds a `SaveGatedEffect` with that many copies of the per-dart `SaveGatedDamage` (Q-17, A-2: until `toolbox-hazard-effects.md` Q-16 (b) adds a per-hit multiplier). `profile.Effect` is a `TrapDartsEffect`; the expanded effect is what the shared evaluator receives (FR-087).
- `TrapPitEffect(depth: FallDistance, printedFall: SaveGatedDamage, spikes: SaveGatedDamage?)`: the two pits, no save. `printedFall` is the printed Bludgeoning figure kept for the consistency test (FR-038); the engine does not roll it (FR-072).

Every `SaveGatedEffect` uses: no Disadvantage circumstances, no automatic-success damage types, `SaveGatedOnsetImmediate`, no Exhaustion, `healable` damage, `SaveGatedNoRepeat`; each `SaveGatedDamage` stores its printed average as printed (FR-037).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-027 | The bundled `traps` data, decoded through the bundled content source and the one content decoder, MUST hold exactly eight entries with ids in this order and no duplicate: `collapsing_roof`, `falling_net`, `fire_casting_statue`, `hidden_pit`, `poisoned_darts`, `poisoned_needle`, `rolling_stone`, `spiked_pit`; each MUST convert with `NewTrapDefinition` (FR-093). | SRD 5.2.1, "Example Traps" ("presented in alphabetical order"), p. 199; `shared-types-and-ownership.md` FR-045 | happy |
| FR-028 | Each example's severity (base band), trigger kind, trigger moments and duration MUST equal table 4.5a. | SRD 5.2.1, the eight entries, pp. 199-201 | happy |
| FR-029 | `collapsing_roof` MUST be a `TrapSavedEffect`, targeting `EachInArea`, with a Dexterity save (DC per table 4.5b), failure damage one Bludgeoning `SaveGatedDamage`, `SaveGatedHalfDamageOnSuccess` and `SaveGatedDurationInstantaneous`, at each of the four bands. | SRD 5.2.1, "Collapsing Roof", p. 199 | happy |
| FR-030 | `falling_net` MUST be a `TrapSavedEffect`, targeting `SingleTarget`, with a `TrapSizeExemption` of Huge, a Dexterity save (DC per table 4.5b), failure the `SaveGatedCondition` Restrained (`AnyFailure`), `SaveGatedNoEffectOnSuccess` and `SaveGatedDurationUntilEnded` ("until it escapes"), at each band. It MUST also carry a `TrapEscapeCheck` (Strength (Athletics), by action, freeing the target or another creature within reach) whose DC equals the save DC at that band. | SRD 5.2.1, "Falling Net", p. 199 | happy |
| FR-031 | `fire_casting_statue` MUST be a `TrapSavedEffect`, targeting `EachInArea`, with a Dexterity save at DC 15 at every band (the DC MUST NOT change with the band), failure damage one Fire `SaveGatedDamage`, `SaveGatedHalfDamageOnSuccess`, `SaveGatedDurationInstantaneous` (each activation is instantaneous; the reset is the lifecycle's, FR-018) and a `TrapAreaOfEffect` with the dice and Cone length of table 4.5b. | SRD 5.2.1, "Fire-Casting Statue", pp. 199-200 | happy |
| FR-032 | `hidden_pit` MUST be a `TrapPitEffect` with no saving throw, the depth as a `FallDistance`, the printed Bludgeoning `SaveGatedDamage` of table 4.5b and no spikes. It holds only the depth: `ResolveFall`, called by the host, rolls the fall (FR-072). | SRD 5.2.1, "Hidden Pit", p. 200 | happy |
| FR-033 | `poisoned_darts` MUST be a `TrapDartsEffect` whose `PerDart` is a Dexterity save at DC 13 at every band (DC MUST NOT change with the band), failure one Poison `SaveGatedDamage` (per-dart dice of table 4.5b), `SaveGatedNoEffectOnSuccess` and `SaveGatedDurationInstantaneous` (also when expanded), with `DartCount` `1d3` and targeting `EachInArea` (each creature in the darts' path). | SRD 5.2.1, "Poisoned Darts", p. 200 | happy |
| FR-034 | `poisoned_needle` MUST be a `TrapSavedEffect`, targeting `SingleTarget`, with a Constitution save (DC per table 4.5b), failure one Poison `SaveGatedDamage` and the `SaveGatedCondition` Poisoned (`AnyFailure`), `SaveGatedHalfDamageOnSuccess` and `SaveGatedDurationFor(SaveGatedFixedTime(1, hours))`, at each band (shape RE-A of `toolbox-hazard-effects.md`). | SRD 5.2.1, "Poisoned Needle", pp. 200-201 | happy |
| FR-035 | `rolling_stone` MUST be defined only for 11-16 and 17-20, with the same effect at both: a `TrapSavedEffect`, targeting `PerContact`, Dexterity save at DC 15, failure one 10d10 Bludgeoning `SaveGatedDamage` (printed average 55) and the `SaveGatedCondition` Prone, `SaveGatedNoEffectOnSuccess` (no half damage) and `SaveGatedDurationInstantaneous`. | SRD 5.2.1, "Rolling Stone", p. 201 | happy |
| FR-036 | `spiked_pit` MUST be a `TrapPitEffect` with no saving throw, the depth as a `FallDistance`, the printed Bludgeoning `SaveGatedDamage` and the Piercing `SaveGatedDamage` of the spikes (table 4.5b) at each band. The host calls `ResolveFall`; `ResolveTrapEffect` rolls only the spikes (FR-072). | SRD 5.2.1, "Spiked Pit", p. 201 | happy |
| FR-037 | Every catalogue `SaveGatedDamage` MUST carry `SaveGatedPrintedAverageOf(n)` with n exactly as printed in table 4.5b, never `SaveGatedNoPrintedAverage` (`toolbox-hazard-effects.md` FR-018): the constructor MUST NOT derive or check it against the dice. A catalogue data test (not a constructor check) MUST walk every entry, assert no damage is `SaveGatedNoPrintedAverage` and assert the printed average equals `dice.Expression.average` rounded down for all entries except the Spiked Pit's Piercing damage at 17-20, which the SRD prints as "57 (13d8)" although 13d8 averages 58; the test pins `13d8` and 57 as the single known exception and the catalogue does not correct it. | SRD 5.2.1, "Spiked Pit", p. 201; "Damage Notation", p. 257; "Round Down", p. 187; `toolbox-hazard-effects.md` FR-018, Q-12 | unhappy (rule interaction) |
| FR-038 | As a catalogue consistency test (not a runtime rule), for `hidden_pit` and `spiked_pit` the printed Bludgeoning dice MUST be (depth in feet divided by 10) d6 at every band, agreeing with the fall rule `ResolveFall` applies (`environment-hazards-and-objects.md` FR-015, FR-016). This spec never rolls them. | SRD 5.2.1, "Hidden Pit", "Spiked Pit", pp. 200-201; "Falling [Hazard]", p. 182 | unhappy (rule interaction) |
| FR-039 | `collapsing_roof` MUST record that rubble turns the area into Difficult Terrain, and `hidden_pit` and `spiked_pit` that the lid remains open, as a `TrapAftermath` (`[]TrapAftermathEffect` (treated as a set): `rubbleDifficultTerrain`, `lidRemainsOpen`; empty for the others) on the definition that the host reads. The engine MUST NOT apply either. | SRD 5.2.1, "Collapsing Roof", p. 199; "Hidden Pit", "Spiked Pit", pp. 200-201 | happy |
| FR-085 | `TrapProfile.saving` MUST be a `TrapSaving`: a `TrapSaveRequirement(ability, difficultyClass)` taken from the profile's `SaveGatedEffect` (or `TrapDartsEffect.PerDart`) equal to the ability and DC of table 4.5b for every trap and band, or `TrapNoSave` for the pits (an explicit variant, not an absent value, for the missing save). The host reads it to roll each save before calling `ResolveTrapEffect`. | SRD 5.2.1, the DCs of the eight entries, pp. 199-201 | happy |

**4.5a Identity** (FR-028; heading levels).

| Trap | Severity | Trigger kind | Moments | Duration | Source |
|---|---|---|---|---|---|
| Collapsing Roof | Deadly (1-4) | `tripWireCrossed` | none | instantaneous | p. 199 |
| Falling Net | Nuisance (1-4) | `tripWireCrossed` | none | instantaneous | p. 199 |
| Fire-Casting Statue | Deadly (1-4) | `pressurePlateEntered` | `FirstTimeOnATurn`, `StartsTurnThere` | instantaneous with reset, unlimited, resets at the start of the next turn | pp. 199-200 |
| Hidden Pit | Nuisance (1-4) | `pitLidEntered` | none | instantaneous | p. 200 |
| Poisoned Darts | Deadly (1-4) | `pressurePlateEntered` | `FirstTimeOnATurn` | instantaneous with reset, limit 3, resets at the start of the next turn if activated fewer than three times | p. 200 |
| Poisoned Needle | Nuisance (1-4) | `lockOpenedImproperly` | none | instantaneous | p. 200 |
| Rolling Stone | Deadly (11-16), Nuisance (17-20) | `pressurePlateEntered` | none | until the effect ends (the stone stops rolling) | p. 201 |
| Spiked Pit | Deadly (1-4) | `pitLidEntered` | none | instantaneous | p. 201 |

**4.5b Band values** (FR-029 to FR-036, FR-085; one test per trap and band). Cell = save DC (or `no save`), printed damage (dice), extra. Pits' depth in feet and Bludgeoning are the fall (FR-072); "P" is Piercing. Sources: roof p. 199; net p. 199; statue p. 200; hidden pit p. 200; darts p. 200; needle pp. 200-201 (1-4 on p. 200, 5-10 onward p. 201); stone, spiked pit p. 201.

| Trap (ability, type, on success) | 1-4 | 5-10 | 11-16 | 17-20 |
|---|---|---|---|---|
| Collapsing Roof (Dexterity, Bludgeoning, half) | DC 13, 11 (2d10) | DC 15, 22 (4d10) | DC 17, 55 (10d10) | DC 19, 99 (18d10) |
| Falling Net (Dexterity, no damage, none; Restrained; escape DC = save DC) | DC 10 | DC 12 | DC 14 | DC 16 |
| Fire-Casting Statue (Dexterity, Fire, half) | DC 15, 11 (2d10), Cone 15 | DC 15, 22 (4d10), Cone 30 | DC 15, 55 (10d10), Cone 60 | DC 15, 99 (18d10), Cone 120 |
| Hidden Pit (no save) | depth 10, 3 (1d6) | depth 30, 10 (3d6) | depth 60, 21 (6d6) | depth 120, 42 (12d6) |
| Poisoned Darts (Dexterity, Poison per dart, none; 1d3 darts) | DC 13, 3 (1d6) | DC 13, 7 (2d6) | DC 13, 14 (4d6) | DC 13, 24 (7d6) |
| Poisoned Needle (Constitution, Poison, half; Poisoned 1 hour) | DC 11, 5 (1d10) | DC 13, 11 (2d10) | DC 15, 22 (4d10) | DC 17, 55 (10d10) |
| Rolling Stone (Dexterity, Bludgeoning, none; Prone) | not defined (FR-009) | not defined (FR-009) | deadly, DC 15, 55 (10d10) | nuisance, DC 15, 55 (10d10) |
| Spiked Pit (no save) | depth 10, 3 (1d6), P 9 (2d8) | depth 30, 10 (3d6), P 13 (3d8) | depth 60, 21 (6d6), P 36 (8d8) | depth 120, 42 (12d6), P 57 (13d8; computed average 58, FR-037) |

### 4.6 Detect and Disarm: check steps

A `TrapCheckStep` is one check: purpose, action (`TrapCheckAction`: `search`, `study`, `action`), ability, skill, DC, the `[]TrapFeature` (treated as a set) it reveals or does (one check can reveal several, for example the trip wire and the unstable ceiling), an optional distance in feet, an optional prerequisite `TrapFeature` and the `TrapStepConsequence` of a failure. `TrapDiscovery` is the immutable set of features found. The step does not roll; the host resolves the check (`resolve-d20-test-use-case.md`) and calls `assess(outcome, discovery)`, returning `TrapStepSucceeded(discovery)`, `TrapStepFailed(discovery, consequence)` or `TrapStepPrerequisiteMissing(feature, discovery)`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | A `TrapCheckStep` MUST use Wisdom when its action is `search` and Intelligence when `study`, and the constructor MUST return an error wrapping `srderr.ErrInvalidInput` for any other ability with those two actions. The third value, `action` (an unspecified action, as the Poisoned Needle's disarm), accepts any ability. | SRD 5.2.1, "Search [Action]", p. 187; "Study [Action]", p. 189 (via `hiding-searching-and-senses.md` FR-074, FR-083); `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-041 | Each example MUST carry the detection and understanding steps of table 4.6a with the action, skill, DC, revealed feature and distance given there. | SRD 5.2.1, the "Detect and Disarm" paragraphs, pp. 199-201 | happy |
| FR-042 | The Fire-Casting Statue MUST also carry a spell-reveal step (Detect Magic, no check, reveals an aura of Evocation magic around the statue), and its glyph step MUST require the glyph to be found before the Arcana step (FR-048). | SRD 5.2.1, "Fire-Casting Statue", p. 200 | happy |
| FR-043 | The Poisoned Needle MUST carry a disarm step: Dexterity (Sleight of Hand) at DC 15 with `TrapCheckAction.action` (FR-040), available only once the needle is detected, whose failure triggers the trap. | SRD 5.2.1, "Poisoned Needle", p. 201 | happy |
| FR-044 | Each example MUST carry the neutralisations of table 4.6b with their prerequisite features; a neutralisation involves no check. | SRD 5.2.1, the "Detect and Disarm" and "Avoid" paragraphs, pp. 199-201 | happy |
| FR-045 | `assess(success, discovery)` MUST return `TrapStepSucceeded` with a new `TrapDiscovery` including every feature of the step's set (both trip wire and unstable ceiling for the Collapsing Roof, both trip wire and suspended Net for the Falling Net), leaving the input unchanged; assessing the same success twice MUST give an equal discovery. | SRD 5.2.1, "Collapsing Roof", p. 199 | happy |
| FR-046 | `assess(failure, discovery)` MUST return `TrapStepFailed` with the discovery unchanged and `TrapStepConsequence.none`, except the Poisoned Needle's disarm step, whose consequence MUST be `triggersTrap` (the host then activates the trap through `ResolveTrapEffect`). | SRD 5.2.1, "Poisoned Needle", p. 201 | unhappy (rule interaction) |
| FR-047 | A failed check the SRD does not limit MUST be repeatable: `assess` MUST NOT record an attempt or lock a step after a failure. Interpretation (Q-12). | SRD 5.2.1, "Traps", pp. 199-201 (no attempt limit) | unhappy (state error) |
| FR-048 | `assess` for a step whose prerequisite feature is not in the discovery MUST return `TrapStepPrerequisiteMissing(feature, discovery)` with the discovery unchanged, whatever the outcome. The prerequisites are exactly those in tables 4.6a and 4.6b. | SRD 5.2.1, "Collapsing Roof", "Falling Net", "Poisoned Needle", "Hidden Pit", "Spiked Pit" ("Once detected ..."), pp. 199-201; "Fire-Casting Statue" ("Once the glyph is found"), p. 200 | unhappy (state error) |
| FR-049 | `TrapNeutralisation.IsAvailable(discovery)` MUST be true exactly when its prerequisite feature, if any, is in the discovery, and always true with none. | SRD 5.2.1, as FR-048 | unhappy (rule interaction) |

**4.6a Check steps** (FR-041 to FR-043, FR-045, FR-048). "Reveals" is a feature set; "Prereq." is required first; `action` is `TrapCheckAction.action`.

| Trap | Purpose | Action | Check | DC | Reveals | Distance | Prereq. | Source |
|---|---|---|---|---|---|---|---|---|
| Collapsing Roof | detect | Search | Wisdom (Perception) | 11 | trip wire and unstable section of ceiling | none | none | p. 199 |
| Falling Net | detect | Search | Wisdom (Perception) | 11 | trip wire and suspended Net | none | none | p. 199 |
| Fire-Casting Statue | detect glyph | Search | Wisdom (Perception) | 10 | glyph | within 5 feet of the statue | none | p. 200 |
| Fire-Casting Statue | understand glyph | Study | Intelligence (Arcana) | 15 | glyph meaning ("fire") | none | glyph | p. 200 |
| Fire-Casting Statue | detect pressure plate | Search | Wisdom (Perception) | 15 | pressure plate | none | none | p. 200 |
| Hidden Pit | detect | Study | Intelligence (Investigation) | 15 | pit (its lid) | none | none | p. 200 |
| Poisoned Darts | detect holes | Search | Wisdom (Perception) | 15 | dart holes | none | none | p. 200 |
| Poisoned Darts | detect pressure plate | Search | Wisdom (Perception) | 15 | pressure plate | none | none | p. 200 |
| Poisoned Needle | detect | Search | Wisdom (Perception) | 11 | needle | none | none | p. 201 |
| Poisoned Needle | disarm | action | Dexterity (Sleight of Hand) | 15 | needle disarmed | none | needle | p. 201 |
| Rolling Stone | deduce function | Study | Intelligence (Investigation) | 15 | pressure plate's function | none | none | p. 201 |
| Spiked Pit | detect | Study | Intelligence (Investigation) | 15 | pit (its lid) | none | none | p. 201 |

**4.6b Neutralisations** (FR-044, FR-049; prerequisite feature in brackets, "none" if absent).

| Trap | Neutralisations | Source |
|---|---|---|
| Collapsing Roof | cut or avoid the trip wire [trip wire] | p. 199 |
| Falling Net | cut or avoid the trip wire [trip wire] | p. 199 |
| Fire-Casting Statue | deface the glyph (by action, sharp tool; disarms the trap) [none]; wedge an Iron Spike or similar object under the pressure plate [none] | p. 200 |
| Hidden Pit | wedge an Iron Spike or similar object between lid and floor [pit]; hold the cover shut with Arcane Lock or similar magic [none] | p. 200 |
| Poisoned Darts | plug all the holes (wax, cloth, detritus) [none]; wedge an Iron Spike or similar object under the pressure plate [none] | p. 200 |
| Poisoned Needle | open the lock with Knock or similar magic, the trap does not trigger [none] | p. 201 |
| Rolling Stone | wedge an Iron Spike or similar object under the pressure plate [none] | p. 201 |
| Spiked Pit | wedge an Iron Spike or similar object between lid and floor [pit]; hold the cover shut with Arcane Lock or similar magic [none] | p. 201 |

### 4.7 Options: Escape, Destroy, Slow, Set the Trap, Initiative

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-050 | `hidden_pit` and `spiked_pit` MUST carry a `TrapEscapeOption` whose `[]TrapEscapeRequirement` (treated as a set) is exactly `TrapEscapeClimbSpeed`, `TrapEscapeClimbingGear` and `TrapEscapeMagic(spellID)` with the id of Spider Climb. The engine records the set; whether a creature meets it is the host's. | SRD 5.2.1, "Hidden Pit", "Spiked Pit" ("Escape"), pp. 200-201 | happy |
| FR-051 | `falling_net` MUST carry a destroy option stating that reducing the Net to 0 Hit Points frees any creature trapped in it, referencing the Net by item id; it MUST NOT hold the Net's statistics (Q-11). | SRD 5.2.1, "Falling Net" ("Destroy the Net"), p. 199; "Net", p. 98 | happy |
| FR-052 | `rolling_stone` MUST carry a destroy option whose object is an `ObjectStatus` of size Large, AC 17, Hit Points 100 (current and maximum, Temporary Hit Points 0), Damage Threshold 10, the affinities of `ObjectAffinities.standard()` (Immunity to Poison and Psychic damage only) and `NoAbilityScores` (an object has none, p. 178; its Initiative bonus is therefore a plain `Modifier`, FR-058, Q-5). | SRD 5.2.1, "Rolling Stone" ("Destroy the Stone"), p. 201; `environment-hazards-and-objects.md` FR-072, FR-073, FR-080 | happy |
| FR-053 | `RollingMotion` MUST start at 60 feet per turn; `slowed()` MUST reduce it by 15 and floor it at 0 (60, 45, 30, 15, 0 over four successes, table 4.7a); a motion of 0 MUST report `isStopped`. | SRD 5.2.1, "Rolling Stone", p. 201 | unhappy (boundary) |
| FR-054 | The slow option MUST use a Strength (Athletics) check at DC 20, taken as an action. `TrapSlowOption.Assess(outcome, state)` MUST return `TrapTransitioned` with a `TrapActiveRolling` whose `motion` is reduced by 15 on success and unchanged on failure; a success that brings the motion to 0 MUST end the trap with `TrapEnded(slowedToStandstill)` and return `TrapInert`. | SRD 5.2.1, "Rolling Stone" ("Slow the Stone"), p. 201 | happy |
| FR-055 | `TrapSlowOption.Assess` on a state that is not `TrapActiveRolling` MUST return `TrapTransitionRejected(trapNotRolling, state)` with the unchanged state, whatever the outcome. | `doc/rules/testing.md` §4; SRD 5.2.1, "Rolling Stone", p. 201 | unhappy (state error) |
| FR-056 | `falling_net` MUST carry a set option: requires Thieves' Tools (item id) and all the trap's components including a Net; Dexterity (Sleight of Hand) at DC 13; each attempt takes 10 minutes (an integer). | SRD 5.2.1, "Falling Net" ("Set the Trap"), p. 199 | happy |
| FR-057 | `TrapSetOption.Assess(outcome, state)` on `TrapInert` MUST return `TrapArmed` (count 0) with a `TrapReset` event on success and the unchanged state on failure (repeatable); on any other state it MUST return `TrapTransitionRejected(trapNotInert, state)`. Interpretation (Q-12). | SRD 5.2.1, "Falling Net", p. 199; `doc/rules/testing.md` §4 | unhappy (state error) |
| FR-058 | `rolling_stone` MUST carry a `TrapInitiativeData` (`initiativeModifier: Modifier`, `movementPerTurnFeet: int`, `rules: []TrapMovementRule (treated as a set)`): the stone and nearby creatures roll Initiative; the stone's whole modifier is `Modifier(8)`; it moves 60 feet in one direction on its turn; `rules` is exactly `stoneMovesThroughCreatureSpaces` and `creaturesMoveThroughStoneSpaceAsDifficultTerrain`. The use case MUST return it as data (`TrapInitiativeRequired(data)` on the `TrapEffectResolved` of the activation, `TrapNoInitiative` otherwise) and MUST NOT roll Initiative. | SRD 5.2.1, "Rolling Stone", p. 201 | happy |
| FR-088 | `rolling_stone` MUST carry its contact rule as data, `TrapDefinition.contacts: []TrapContactEvent (treated as a set)`, exactly `stoneEntersCreatureSpaceFirstTimeOnATurn` and `creatureEntersStoneSpaceWhileRolling`; no other example carries contacts. The effect (FR-070) is resolved once per contact the host reports; the engine tracks no turn, position or space, so whether a contact occurred, and whether it is the stone's first entry into that space that turn, is the host's. | SRD 5.2.1, "Rolling Stone", p. 201 ("Whenever the stone enters a creature's space for the first time on a turn or a creature enters the stone's space while the stone is rolling ...") | happy |

**4.7a Slowing the stone** (FR-053 to FR-055; p. 201; one `slowed()` or `assess` call per row).

| Label | Motion before | Outcome | After | Trap state after | Req. | Path |
|---|---|---|---|---|---|---|
| start | 60 | n/a | 60, not stopped | active | FR-053 | happy |
| first success | 60 | success | 45 | active | FR-054 | happy |
| failure | 45 | failure | 45 | active | FR-054 | unhappy (boundary) |
| third success | 30 | success | 15 | active | FR-053 | unhappy (boundary) |
| last success | 15 | success | 0, stopped | inert, `TrapEnded(slowedToStandstill)` | FR-054 | unhappy (boundary) |
| already stopped, or inert (not rolling) | 0, or n/a | success | rejected `TrapNotRolling` | unchanged | FR-055 | unhappy (state error) |

### 4.8 Resolving a trap (`ResolveTrapEffect`)

Command `ResolveTrapEffectCommand(definition, level, state, targets)`: a `TrapDefinition`, a `CharacterLevel`, the `TrapState` and a `[]TrapTarget`. `TrapTarget(combatant: CombatantID, size: CreatureSize, save: SaveGatedSave)` holds the finished save as `toolbox-hazard-effects.md` takes it (`SaveGatedSaveRolled(CheckResult)`, `SaveGatedSaveForcedFailure(ForcedFailureResult)`, `SaveGatedSaveNotRolled`). For traps with no save the save is ignored; for an armed Rolling Stone the whole target list is ignored (FR-062). The host reads each save requirement from `profile.saving` (FR-085) first.

Result, closed interface `ResolveTrapEffectResult`:

- `TrapEffectResolved(state, profile, levelFit, resolutions, rolls, initiative, aftermath, events)`: new `TrapState`, the `TrapProfile` used, its `TrapLevelFit`, one `TrapTargetResolution` per target in order, the `dice.Result`s behind the damage, the `TrapInitiativeOutcome` (FR-058), the `TrapAftermath` (FR-039) and the events (`[]ToolboxEvent`, FR-096).
- `TrapEffectRejected(reason, culprit)`: `reason` a `TrapEffectRejection` (`trapNotArmed`, `tooManyTargets`, `duplicateTarget`, `missingSaveOutcome`); `culprit` a `TrapRejectionCulprit`: `TrapCulpritCombatant(id)` for `duplicateTarget` and `missingSaveOutcome`, `TrapNoCulprit` otherwise.
- `TrapTargetResolution` (FR-097, `requirement` the `TrapSaveRequirement` that applied): `TrapTargetAutoSucceeded(target, requirement)` (size exemption); `TrapTargetSaved(target, requirement, consequences, events)` (`consequences` the shared `SaveGatedConsequences`, empty or the `[HalfDamage]`-adjusted damage); `TrapTargetFailed(target, requirement, consequences, active, events)` (failed or forced to fail; `active` the shared evaluator's `ActiveSaveGatedEffect`, in `SaveGatedPhaseActive(0)` or, for an instantaneous duration, `SaveGatedPhaseEnded(instantaneous)`); `TrapTargetFell(target, fallDistance, consequences)` (a pit: the `FallDistance` for the host's `ResolveFall` and `LandAfterFall`, and the Spiked Pit's Piercing `Damage` in `consequences`, empty for the Hidden Pit).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-059 | For a `TrapArmed` trap, `ResolveTrapEffect` MUST validate the command in the order of FR-090, activate it through `TrapLifecycle.activate`, resolve the effect per target and return `TrapEffectResolved` with the new state, the per-target resolutions and the events of FR-096. These rules are `TrapResolver`'s (FR-095). | SRD 5.2.1, "Parts of a Trap", Trigger and Duration, p. 199 | happy |
| FR-060 | The level caution is advice to the Game Master, so `ResolveTrapEffect` MUST NOT reject it. For any `CharacterLevel` it MUST use `TrapDefinition.nearestProfileFor(level)`: the profile of the level's band when covered (`withinRange`), of the lowest covered band when below (`belowTrapLevels`), of the highest covered band when above (`aboveTrapLevels`, interpretation, Q-12), and report the fit on `TrapEffectResolved.levelFit`. Requests, rejections and transitions are otherwise identical: a Rolling Stone on a level-8 party resolves with the 11-16 profile and `belowTrapLevels` (Q-19). | SRD 5.2.1, "Parts of a Trap", caution paragraph, p. 199 | unhappy (boundary) |
| FR-061 | For any trap other than a Rolling Stone in `TrapActiveRolling`, a state that is not `TrapArmed` MUST return `TrapEffectRejected(trapNotArmed)`, request no roll and leave the state unchanged. | SRD 5.2.1, "Parts of a Trap", p. 199; FR-021 | unhappy (state error) |
| FR-062 | Activating an armed Rolling Stone MUST return `TrapActiveRolling` with `RollingMotion` 60 (FR-020), the `TrapInitiativeData` of FR-058, no per-target resolutions and no roll. The command's `targets` are **ignored**: saves are meaningless before the stone moves, so FR-066, FR-067 and FR-086 do not apply and a `SaveGatedSaveNotRolled`, duplicate or any number of targets is accepted (table 4.8a). The stone's effect on creatures resolves only from `TrapActiveRolling`, once per contact (FR-070, FR-088), and then FR-066, FR-067 and FR-086 apply as for any `PerContact` effect. | SRD 5.2.1, "Rolling Stone", p. 201 ("When a creature moves onto a hidden pressure plate, a 5-foot-radius orb ... begins to roll") | happy |
| FR-063 | For a `TrapSavedEffect` with half damage on success (Collapsing Roof, Fire-Casting Statue, Poisoned Needle), a failed (or forced-failure) target MUST resolve to `TrapTargetFailed` and the full rolled `Damage` with no adjustment, and a target that saved to `TrapTargetSaved` and the same `Damage` with `[HalfDamage]` (table 4.8a). Rounding belongs to `ApplyDamage`. | SRD 5.2.1, "Collapsing Roof", "Fire-Casting Statue", pp. 199-200; "Poisoned Needle", pp. 200-201; "Half Damage", p. 16; `toolbox-hazard-effects.md` FR-034, FR-035, FR-036 | happy |
| FR-064 | For the area effects (Collapsing Roof, Fire-Casting Statue) the damage MUST be rolled once and shared by every target (`SaveGatedDamageShared`); exactly one roll of the band's dice for any number of targets above 0. | SRD 5.2.1, "Damage against Multiple Targets", p. 16; `hit-points-and-damage.md` FR-100; `toolbox-hazard-effects.md` FR-042, FR-043 | unhappy (rule interaction) |
| FR-065 | With no targets, activation MUST still happen (new state and events returned), no roll MUST be requested and the per-target list MUST be empty. | `doc/rules/testing.md` §4; SRD 5.2.1, "Collapsing Roof", p. 199 ("Each creature beneath") | unhappy (empty and minimal) |
| FR-066 | A target whose save is required (every effect except the pits, for a target not size-exempt) and is `SaveGatedSaveNotRolled` MUST yield `TrapEffectRejected(missingSaveOutcome)` carrying the `CombatantID`, no roll, state unchanged. A `SaveGatedSaveNotRolled` for an effect with no save, for a size-exempt target (FR-068) or for an armed Rolling Stone's targets (FR-062) MUST be accepted. This differs on purpose from `toolbox-hazard-effects.md` FR-030 (a `SaveRequired` plan): the host reads `saveRequirement` first (FR-085), so a missing save here is a sequencing error. | `doc/rules/testing.md` §4; SRD 5.2.1, "Saving Throws", p. 7 | unhappy (invalid input) |
| FR-067 | Outside an armed Rolling Stone (FR-062), two targets with the same `CombatantID` MUST yield `TrapEffectRejected(duplicateTarget)` carrying that id, no roll, state unchanged. | `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-068 | For the Falling Net, a Huge or Gargantuan target (the `TrapSizeExemption`) MUST resolve as `TrapTargetAutoSucceeded` (any supplied save, including `SaveGatedSaveNotRolled`, ignored; no consequences, no events). Any other size uses its save: failure is `TrapTargetFailed` with `consequences.conditions` `[restrained]`, `active` in `SaveGatedPhaseActive(0)` with `SaveGatedDurationUntilEnded` and the one event `SaveGatedEffectStarted`; success is `TrapTargetSaved` with empty consequences and no events. No roll (no damage). `TrapResolver` decides the exemption with `TrapSizeExemption.Exempts(size)` and hands the shared planner `SaveGatedExempt` (`toolbox-hazard-effects.md` FR-076), so the automatic success itself is the shared planner's. | SRD 5.2.1, "Falling Net", p. 199 ("The target succeeds automatically if it's Huge or larger") | unhappy (boundary) |
| FR-069 | For the Poisoned Needle a failure MUST also return Poisoned (`consequences.conditions` `[poisoned]`) and `active` in `SaveGatedPhaseActive(0)` with `SaveGatedDurationFor(SaveGatedFixedTime(1, hours))` and the one event `SaveGatedEffectStarted` (not instantaneous, so no `Ended`); a success returns no condition, no `active` and no events. Damage MUST be 1 roll of the band's dice. | SRD 5.2.1, "Poisoned Needle", pp. 200-201 | happy |
| FR-070 | For an active Rolling Stone, called once per contact (FR-088): a failed target MUST resolve to the full 10d10 Bludgeoning `Damage` and `consequences.conditions` `[prone]` (`active` in `SaveGatedPhaseEnded(instantaneous)`, events `[SaveGatedEffectStarted, SaveGatedEffectEnded(instantaneous)]`, FR-096); a target that saved resolves to no damage, no condition (no half damage) and no events. One roll of 10d10 per call when at least one target fails, shared (`SaveGatedDamageShared`). The state MUST stay `TrapActiveRolling` and unchanged, with no lifecycle event and no `TrapActivated` (no activation occurs). | SRD 5.2.1, "Rolling Stone", p. 201 | happy |
| FR-071 | For the Poisoned Darts a success MUST resolve to no damage. When at least one target fails, the use case MUST roll once for all failing targets, as one damaging effect (p. 16): one `1d3` request for the number of darts, then one request of the band's per-dart dice per dart, in that order, and every failing target takes the same dart `Damage` list (one Poison `Damage` per dart, as `SaveGatedDamageShared`). Each failing target is `TrapTargetFailed` with `active` in `SaveGatedPhaseEnded(instantaneous)` and events `[SaveGatedEffectStarted, SaveGatedEffectEnded(instantaneous)]` (FR-033, FR-096). No failing target means no roll. Interpretation (Q-8). | SRD 5.2.1, "Poisoned Darts", p. 200; "Damage against Multiple Targets", p. 16 | happy |
| FR-072 | For the pits (no save) the use case MUST NOT roll or re-implement the fall. Each target resolves to `TrapTargetFell` with the pit's depth as `fallDistance` (no events); the host passes it to `ResolveFall` (one d6 per 10 feet, `HalfDamage` for a liquid) and then `LandAfterFall` with the final damage (`environment-hazards-and-objects.md`). For the Spiked Pit only, the use case MUST also roll the band's Piercing dice once for the target and return it as `consequences.damage` (empty for the Hidden Pit). Size and save MUST NOT matter. | SRD 5.2.1, "Hidden Pit", p. 200; "Spiked Pit", p. 201; "Falling [Hazard]", p. 182; `environment-hazards-and-objects.md` FR-015 to FR-017, FR-023 | happy |
| FR-073 | The `Roller` calls MUST be exactly those of table 4.8b, in that order, each through `dice.Roll`. | `dice-and-rolls.md` FR-050; `toolbox-hazard-effects.md` FR-070 | unhappy (rule interaction) |
| FR-074 | `ResolveTrapEffect` MUST NOT apply Resistance, Vulnerability, Immunity, Temporary Hit Points, Hit Point loss or any condition: it returns `Damage` values, adjustments and `ConditionKind`s in `SaveGatedConsequences` (and the `ActiveSaveGatedEffect`) for the host's `ApplyDamage` and `ApplyConditionToSet`. | `environment-hazards-and-objects.md` section 1 (D-1); `hit-points-and-damage.md` section 1; `toolbox-hazard-effects.md` FR-046 | unhappy (rule interaction) |
| FR-075 | A dice port failure MUST be returned wrapped once as `*srderr.PortError` (a nil result), never swallowed or recovered; the command and state passed in remain unchanged. | `dice-and-rolls.md` FR-052; `doc/rules/architecture.md` §4, §6 | unhappy (failure of collaborator) |
| FR-076 | With a `dicetest.FixedRoller` the result MUST be fully determined: the minimum and maximum rolls of each dice expression resolve to `Damage` amounts equal to those rolls, with no clamping or rounding here. | `doc/rules/testing.md` §4 | unhappy (boundary) |
| FR-077 | The result's collections (resolutions, `Damage` lists, conditions, events) MUST be defensively copied views, and the command's target list MUST NOT be altered. | `doc/rules/architecture.md` §4; `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-078 | Each `TrapTargetResolution` variant MUST carry the fields above, and `TrapEffectResolved` MUST carry the ability and `DifficultyClass` (via `profile.saving` and each `requirement`), band, severity and `dice.Result`s used, so a host can render its own breakdown. | `doc/rules/architecture.md` §6 | happy |
| FR-086 | The single-target effects (Falling Net, Poisoned Needle, Hidden Pit, Spiked Pit; `TrapTargeting.SingleTarget`) MUST accept zero or one target; two or more MUST yield `TrapEffectRejected(tooManyTargets)`, no roll, state unchanged. The SRD affects "the first creature that crosses the trip wire", "the creature" (needle), "a creature that falls into the pit"; choosing which is the host's. `EachInArea` (roof, statue, darts) and `PerContact` (stone) accept any number; an armed stone's targets are ignored, not counted (FR-062). | SRD 5.2.1, "Falling Net", p. 199; "Poisoned Needle", p. 200; "Hidden Pit", p. 200; "Spiked Pit", p. 201 | unhappy (invalid input) |
| FR-087 | `ResolveTrapEffect` MUST NOT define a second effect model or call another use case (`shared-types-and-ownership.md` FR-044). Through `TrapResolver` (FR-095) it obtains the profile's `SaveGatedEffect` (for the darts, `TrapDartsEffect.Expand`) and the automatic-success planning, margin and consequence evaluation and shared-damage handling from the pure planner, `DiceRequired` and evaluator that `toolbox-hazard-effects.md` publishes (FR-024 to FR-029, FR-033 to FR-047, FR-076, FR-083, FR-087), passing the rolled damage as `SaveGatedDamageShared`, and records only its own signals (FR-079). For roof, statue, net, needle and stone the per-target consequences, `active` state and events MUST equal what a host gets from `RollSaveGatedDamage` once and `ResolveSaveGatedEffect` per target with the same effect, rolls and saves; for the darts, from `ResolveSaveGatedEffect` over `TrapDartsEffect.Expand(n)` for the rolled count n. Delegating to the use cases cannot work: activation and the shared roll must be one observed operation. The published services are `SaveGatedPlanner`, `DiceRequired` and `SaveGatedEvaluator` with its operations `EvaluateInitial`, `EvaluateRepeat`, `EvaluateEnd` (Q-17). | `toolbox-hazard-effects.md` section 1, FR-040 to FR-043; `doc/rules/architecture.md` §2, §4 | unhappy (rule interaction) |
| FR-090 | For `ResolveTrapEffect`, `TrapResolver` (FR-095) MUST resolve the profile first (FR-060, never fails), then reject in this order, stopping at the first: `trapNotArmed` (FR-061), `tooManyTargets` (FR-086), `duplicateTarget` (FR-067), `missingSaveOutcome` (FR-066, first target in list order). For an armed Rolling Stone only `trapNotArmed` applies (FR-062). A rejection requests no roll and leaves the state unchanged. Severity, band, effect and level fit therefore exist for every outcome (section 7). | `doc/rules/testing.md` §4; `doc/rules/observability.md` §5 | unhappy (state error) |

**4.8a Save result mapping** (one row is one command; sources: roof p. 199, statue p. 200, needle pp. 200-201, net p. 199, pits pp. 200-201, stone p. 201, darts p. 200, p. 16). `failure` / `success` means a `SaveGatedSaveRolled` that misses / meets the DC (a forced failure behaves as `failure`); an absent save is `SaveGatedSaveNotRolled`. Dice are the band's dice (table 4.5b). Each cell with several cases is one test per case.

| Label | Trap, size, save | Outcome | Req. | Path |
|---|---|---|---|---|
| roof, statue | Collapsing Roof or Fire-Casting Statue, Medium | `failure`: full roll (Bludgeoning or Fire), no adjustment; `success`: same roll, `HalfDamage`; no condition | FR-063 | happy |
| needle | Poisoned Needle, Medium | `failure`: full roll Poison, Poisoned 1 hour; `success`: same roll Poison, `HalfDamage`, no condition | FR-069 | happy; success unhappy (boundary) |
| net, Large | Falling Net, Large (one size below Huge) | `failure`: no damage, Restrained until ended; `success`: nothing | FR-068 | unhappy (boundary) |
| net, Huge or Gargantuan | Falling Net, Huge with an absent save; Gargantuan with `failure` (ignored) | `autoSucceeded`, nothing | FR-068 | unhappy (boundary) |
| single-target, two targets | Falling Net, Poisoned Needle, Hidden Pit, Spiked Pit, two Medium targets, any save | rejected `tooManyTargets` | FR-086 | unhappy (invalid input) |
| net, one target | Falling Net, one Medium, `success` | nothing (accepted) | FR-086 | unhappy (boundary) |
| stone armed | Rolling Stone (armed): target with an absent save; two targets same id with `failure`; three targets with `failure` | activated, no resolutions, no roll | FR-062 | unhappy (boundary) |
| stone not armed | Rolling Stone (inert), any | rejected `trapNotArmed` | FR-061 | unhappy (state error) |
| stone active, invalid | Rolling Stone (active): same id twice; absent save | rejected `duplicateTarget`; rejected `missingSaveOutcome` | FR-062, FR-066, FR-067 | unhappy (invalid input) |
| stone active | Rolling Stone (active), Medium | `failure`: full 10d10 Bludgeoning, Prone; `success`: nothing; two `success`: nothing and no roll | FR-070 | happy; others unhappy (boundary) |
| darts | Poisoned Darts, Medium | one `failure`: one Poison `Damage` per dart; two `failure`: the same dart list for both, one `1d3` roll; `success` or two `success`: nothing (no roll) | FR-071 | happy; others unhappy (rule interaction, boundary) |
| pits | Hidden Pit, Spiked Pit, any size, save ignored | Hidden Pit: `fallDistance` only; Spiked Pit: Piercing plus `fallDistance` | FR-072 | happy |
| save required, no outcome | Collapsing Roof, Medium, absent save | rejected `missingSaveOutcome` | FR-066 | unhappy (invalid input) |
| duplicate target | Collapsing Roof, Medium, `failure` twice same id | rejected `duplicateTarget` | FR-067 | unhappy (invalid input) |

**4.8b Dice requests** (FR-073; band dice from table 4.5b).

| Trap and state | Requests, in order |
|---|---|
| Collapsing Roof or Fire-Casting Statue, armed, at least one target | 1 request of the band's Bludgeoning or Fire dice |
| Poisoned Needle, armed, one target | 1 request of the band's Poison dice (rolled whatever the outcome, a successful save takes half) |
| Poisoned Darts, armed, at least one failing target | once for all failing targets: `1d3`, then one per-dart request per dart (interpretation, Q-8) |
| Spiked Pit, armed, one target | 1 request of the Piercing dice (the host's `ResolveFall` rolls the Bludgeoning) |
| Rolling Stone, active, at least one failing target | 1 request of `10d10`, shared |
| Poisoned Darts with no failing target; Rolling Stone active with no failing target | none |
| Hidden Pit; Rolling Stone, armed; Falling Net; any trap rejected or with no target | none |

**4.8c Level fit** (FR-060; `nearestProfileFor`; the Rolling Stone covers 11-16 and 17-20).

| Trap, level | Profile band | `TrapLevelFit` | Result | Source | Path |
|---|---|---|---|---|---|
| Rolling Stone, 1, 8, 10 | `levels11to16` | `belowTrapLevels` | resolves (severity deadly) | pp. 199, 201 | unhappy (boundary) |
| Rolling Stone, 11 | `levels11to16` | `withinRange` | resolves | p. 201 | happy |
| Collapsing Roof, 20 | `levels17to20` | `withinRange` | resolves | p. 199 | unhappy (boundary) |
| host-built trap 1-4 only, 5 | `levels1to4` | `aboveTrapLevels` | resolves (interpretation, Q-12) | p. 199 (silent) | unhappy (boundary) |

### 4.9 Structure and telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-079 | Each `ResolveTrapEffect` call MUST record one span `srd5e.toolbox.resolve_trap_effect` (pattern of `toolbox-hazard-effects.md` section 7), increment the shared `srd5e.toolbox.operations` once with `srd5e.toolbox.operation` `resolve_trap_effect`, and increment `srd5e.toolbox.trap.resolved` once with the attributes of section 7, whether resolved or rejected (not for a call ending in a dice port error, FR-081). All names are constants in the `toolbox` signal constants. | `doc/rules/observability.md` §5, §6; `toolbox-hazard-effects.md` FR-072 | happy |
| FR-080 | A rejection MUST set the span error status, `srd5e.outcome` `failure` and `srd5e.toolbox.trap.rejection` naming the reason, and MUST NOT be logged. A resolved activation MUST set `srd5e.outcome` `success`, including when targets fail their saves or the fit is not `withinRange`. | `doc/rules/observability.md` §6, §7 | unhappy (failure) |
| FR-081 | A dice port failure MUST be recorded exactly once by the use case: `span.RecordError` (error status) and one increment of the shared `srd5e.toolbox.errors` with operation `resolve_trap_effect` and `error.type` `dice_roller_error`; the call MUST still increment `srd5e.toolbox.operations` once with `srd5e.outcome` `failure` and `srd5e.toolbox.result` `error`, and record `srd5e.toolbox.operation_duration` (`toolbox-hazard-effects.md` FR-078, FR-081); it MUST NOT be logged or counted under a trap-specific error signal. With no result, it MUST NOT increment `srd5e.toolbox.trap.resolved` or `srd5e.toolbox.saves` (`toolbox-hazard-effects.md` FR-079), record `srd5e.toolbox.trap.targets` or `srd5e.toolbox.damage_rolled`, or set `srd5e.toolbox.trap.rejection`. | `doc/rules/observability.md` §7; `toolbox-hazard-effects.md` FR-074, FR-078 | unhappy (failure of collaborator) |
| FR-082 | With the no-op `telemetry.Telemetry` and with a recording one the use case MUST return equal results for equal inputs. | `doc/rules/observability.md` §9 | unhappy (rule interaction) |
| FR-083 | Package `toolbox` MUST export `ResolveTrapEffect` and the published set, exactly the section 3 names starting with `Trap` or `RollingMotion` plus `ResolveTrapEffect` with its command and result types (not the externals, and not `TrapID`, `TrapEntry` and `TrapEntryBand`, which `content` exports, FR-092), each name once with a doc comment starting with its name; there is no re-export package. No exported name MUST be a generic variant name (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`, `Found`, `NotFound`) or duplicate another exported name of the module; the shared `SaveGated*` names belong to the hazard-effects spec. | `doc/rules/architecture.md` §4, §9; `shared-types-and-ownership.md` FR-017 | unhappy (name collision) |
| FR-084 | Package `toolbox` MUST import only the kernel packages and the published API of `content`, `d20`, `character` and `combat` (enforced by `internal/archtest`), and its domain files MUST be pure (no `context.Context`, goroutine, `math/rand`, `time.Now`, display strings), with no file naming a consuming project. | `doc/rules/architecture.md` §2, §3, §9 | unhappy (forbidden edge) |
| FR-089 | Each call MUST record the target count in the histogram `srd5e.toolbox.trap.targets` (unit `{target}`, attribute `srd5e.toolbox.trap.effect`) once, including for a rejection (the command's count) and zero targets (0), and its latency in the shared `srd5e.toolbox.operation_duration` (unit `s`, operation `resolve_trap_effect`) once. Counts apply only to a `TrapEffectResolved` result: a rejection, a dice port error and the targets of an armed Rolling Stone (FR-062) add no `saves`. It MUST increment the shared `srd5e.toolbox.saves` once per target that has a save (`srd5e.toolbox.save_ability`, `srd5e.toolbox.save_result`); a size-exempt target counts as `auto_succeeded`, pits are not counted. No trap-specific duration or error signal exists. It MUST record the shared `srd5e.toolbox.damage_rolled` (hit points) once for a call that returns damage (`toolbox-hazard-effects.md` FR-080): the sum of the amounts of the `Damage` values rolled, before adjustment, each rolled `Damage` counted once however many targets share it (roof, statue, darts and stone record one roll's total; the needle and the Spiked Pit's spikes one amount); nothing for a call with no damage (a rejection, an armed stone, the Falling Net, the Hidden Pit, all targets succeeding against a no-half-damage effect, no targets). | `doc/rules/observability.md` §5, §9, §10; `toolbox-hazard-effects.md` section 7, FR-080 | happy |

### 4.10 Delivery as content, entry conversion and events

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-091 | The eight examples MUST be content, not a code table: kind `trap` (a `content.Kind` member requested of `content-source-port.md`), root array `traps` (revision row R3-traps of `content-schema-and-decoder.md`, requested, adding no `schemaVersion`), typed id `TrapID` (`content-schema-and-decoder.md` FR-114, FR-129). The JSON source of truth is `content/data/traps.json`, embedded directly with `//go:embed` (no generator step); data reaches a host only through the bundled content source and the one decoder. This data depends only on `content-schema-and-decoder.md` and `content-source-port.md`, never on another data spec. | `shared-types-and-ownership.md` FR-030, FR-031, FR-033, FR-045, FR-047; `content-schema-and-decoder.md` FR-103, FR-129 | happy |
| FR-092 | `TrapEntry` MUST be a data-only `content` value (immutable, equal by value) naming no `toolbox` type: `TrapID`, name, one `TrapEntryBand` per covered band (band, severity, effect), trigger, duration, check steps, neutralisations, escape, destroy, slow and set options, aftermath, contacts and Initiative data, written with plain strings for this spec's closed names, integers, `dice.Expression`, `Ability`, `Skill`, `DamageType`, `ConditionKind`, `CreatureSize`, `ItemID` and `SpellID`. Every damage carries its dice and `SaveGatedPrintedAverageOf` average as printed (FR-037). It MUST implement `ContentEntry` (key `ContentKey(trap, id)`, `content-source-port.md`) and `ContentReferences` (`content-schema-and-decoder.md` FR-125), listing exactly the sites of table 4.10a, each registered by row R3-traps as the tuple `(owner kind, field, target kind, constraint?)` of `content-schema-and-decoder.md` FR-126, so an unknown id is an `unresolvedReference` of the whole-set validation, not a run-time failure here. | `shared-types-and-ownership.md` FR-035, FR-047; `content-schema-and-decoder.md` FR-125, FR-126; `content-source-port.md` | happy |
| FR-093 | `NewTrapDefinition(entry content.TrapEntry) (TrapDefinition, error)` MUST be pure. On success it returns the built definition; on failure it returns the zero `TrapDefinition` and an `errors.Join` of one or more `TrapEntryProblem` values (`Path`, `Kind`, where `Kind` is one of `unknownName`, `noBands`, `bandGap`, `outOfRange`, `inconsistent`; each `errors.Is` `srderr.ErrInvalidInput`), non-empty and in entry order. It MUST report every problem and never panic for entry data (untrusted, `doc/rules/architecture.md` §6): it pre-validates every invariant of every constructor it calls and reports a violation as `outOfRange` (number outside its range) or `inconsistent` (a forbidden combination); the list is table 4.10b, and a test walks the table against the constructors so a later invariant is not missed. The panicking `MustX` constructors stay for host-built traps. The eight bundled entries MUST build definitions equal to tables 4.5a, 4.5b, 4.6a and 4.6b. | `doc/rules/architecture.md` §5, §6; `shared-types-and-ownership.md` FR-045 | unhappy (invalid input) |
| FR-094 | `TrapEvent` MUST be a closed interface embedding `ToolboxEvent`, with `TrapActivated{Activations}`, `TrapReset{Activations}` and `TrapEnded{Reason}` its only variants (listed in its doc comment), declared in `toolbox/trap_event.go`, so a host's type switch over `TrapEvent` is exhaustive without sharing a file with the hazard-effects events. Each MUST be a past-tense immutable value defined only here. `TrapTransitioned.Events` is a `[]TrapEvent`; `TrapEffectResolved.Events` a `[]ToolboxEvent` (FR-096). | `shared-types-and-ownership.md` FR-019, FR-021; `doc/rules/architecture.md` §3 | happy |

**4.10a Reference sites and pinned ids** (FR-092; one test per row; no constraint). Ids follow `content-equipment.md` FR-003 (printed name lower-cased, apostrophes removed, other non-alphanumeric runs one underscore) and the spell ids of `content-spells.md` (`detect_magic` is the thin-slice id, `content-thin-slice.md` FR-003).

| Owner kind | Field | Target kind | Pinned ids (printed name) | Source |
|---|---|---|---|---|
| `trap` | `neutralisations[].itemID` | `equipment` | `spikes_iron` ("Spikes, Iron", the table row, p. 100; the trap text says "Iron Spike") | pp. 199-201 |
| `trap` | `setOption.itemIDs` | `equipment` | `thieves_tools`, `net` ("Net", p. 98) | p. 199 |
| `trap` | `destroyOption.itemID` | `equipment` | `net` | p. 199 |
| `trap` | `neutralisations[].spellID` | `spell` | `detect_magic`, `arcane_lock`, `knock` | pp. 200-201 |
| `trap` | `escape.requirements[].spellID` | `spell` | `spider_climb` | pp. 200-201 |
| `trap` | `checkSteps[].spellID` (spell-reveal step, FR-042) | `spell` | `detect_magic` | p. 200 |

**4.10b Entry invariants that `NewTrapDefinition` pre-validates** (FR-093). One entry that breaks only that invariant yields an error with exactly one `TrapEntryProblem` of the kind shown and no panic; a cell listing several invariants is one test per item. `DifficultyClass` has no range check (`d20-tests.md` D-02), so any integer save DC is accepted.

| Label | Called constructor and invariant | Kind | Source |
|---|---|---|---|
| no bands | `TrapDefinition`: at least one band | `noBands` | FR-006 |
| band gap | `TrapDefinition`: contiguous bands | `bandGap` | FR-007 |
| count or limit 0 | `TrapTimed` count at least 1; limited reset limit at least 1 | `outOfRange` | FR-016 |
| days unit | `TrapTimed` unit `days` | `inconsistent` | FR-015 |
| search with Strength | `TrapCheckStep`: Search uses Wisdom, Study Intelligence | `inconsistent` | FR-040 |
| unknown closed name | an unknown trigger, duration, effect, feature or other closed name | `unknownName` | FR-093 |
| shared numeric ranges | `SaveGatedDamage` printed average at least 1 (hazard FR-018); `SaveGatedFixedTime` amount at least 1 and `SaveGatedRolledTime` valid dice (hazard FR-017); Exhaustion gain 1 to 6 (hazard FR-013); `SaveGatedFailedByAtLeast` margin at least 1 (hazard FR-014); a `dice.Expression` that `dice-and-rolls.md` FR-007, FR-008 reject | `outOfRange` | hazard FRs as listed |
| shared combinations | `SaveGatedFailure` has some damage, condition or Exhaustion (hazard FR-009); `SaveGatedHalfDamageOnSuccess` needs damage (hazard FR-010); a `ConditionKind` once (hazard FR-011); no Exhaustion `SaveGatedCondition` (hazard FR-012) | `inconsistent` | hazard FRs as listed |
| object and fall ranges | `FallDistance` 0 feet or more (`environment-hazards-and-objects.md` FR-014); `DamageThreshold` 1 or more (environment FR-080); `ObjectStatus` Temporary Hit Points 0 (environment FR-072) and the `HitPoints` invariants it calls (`hit-points-and-damage.md`) | `outOfRange` | environment FRs as listed |
| two problems | two of the rows above in one entry: both reported, in entry order | both | FR-093 |

### 4.11 Domain resolver, result events and named states

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-095 | The rules of an activation MUST live in the pure, silent domain service `TrapResolver` (a stateless type in `toolbox/trap_resolver.go`), not in the use case. `TrapResolver.Plan(definition, level, state, targets)` MUST return the closed interface `TrapPlanRejected{Reason, Culprit, Profile, LevelFit}` (FR-060, FR-090 order, FR-062; profile and fit exist for a rejection too) or `TrapPlanReady{...}` holding the activation (`TrapLifecycle.Activate`), profile, fit and per-target plans. `TrapPlanReady.NextRequest(rolled)` MUST return `TrapRollNext{Expression}` or `TrapRollsComplete`, in the order of table 4.8b, taken from the shared `DiceRequired` (`toolbox-hazard-effects.md` FR-087) except for the darts (`1d3`, then one per-dart expression per dart, FR-071). `TrapPlanReady.Complete(rolled)` MUST return the `TrapResolution` (state, profile, fit, per-target resolutions, rolls, initiative outcome, aftermath, events) using the shared planner and evaluator (FR-087). The service also owns the size-exemption decision (`TrapSizeExemption.Exempts`: true for Huge and Gargantuan), the darts expansion (`TrapDartsEffect.Expand(darts int)`), the per-target mapping of table 4.8a and the contacts of FR-088. `ResolveTrapEffect` MUST only call `Plan`, roll each `TrapRollNext` through `dice.Roll` in order, call `Complete`, map to `ResolveTrapEffectResult` and record signals, with no branch on the trap's identity, size, save or damage. | `doc/rules/architecture.md` §3, §4, §10; `toolbox-hazard-effects.md` FR-087 | happy |
| FR-096 | `TrapEffectResolved.events` MUST be the lifecycle events of the activation (the one `TrapActivated`, FR-017 to FR-020), then, for each failing target in target order (a call on `TrapActiveRolling` has no lifecycle event: FR-070), exactly the events the shared evaluator returns: `[SaveGatedEffectStarted]` for a non-instantaneous duration (Falling Net, Poisoned Needle) and `[SaveGatedEffectStarted, SaveGatedEffectEnded(instantaneous)]` for `SaveGatedDurationInstantaneous` (Collapsing Roof, Fire-Casting Statue, Poisoned Darts, Rolling Stone; `toolbox-hazard-effects.md` FR-053, FR-083). A target that saved, auto-succeeded or fell contributes none. The same events are on each `TrapTargetFailed.events`; every roof, statue, darts and stone failure has `active` in `SaveGatedPhaseEnded(instantaneous)`. | `toolbox-hazard-effects.md` FR-053, FR-083; SRD 5.2.1, "Parts of a Trap", Duration, p. 199 | happy |
| FR-097 | No published trap type MUST use a nil pointer, nil interface or magic zero for a named state: `TrapState` has the five variants of section 4.4; a profile's saving is `TrapSaving` (FR-085); a target's outcome is `TrapTargetResolution`; the Initiative data of a result is `TrapInitiativeOutcome`; a rejection's culprit is `TrapRejectionCulprit`. The one exception is the optional parts of a definition (size exemption, area, spikes, Initiative data, slow, set and destroy options): the absence of an optional feature of content data, justified in section 5 and exposed as a value plus an `ok` bool, never a nil pointer. | `doc/rules/architecture.md` §3 | unhappy (invalid input) |

**4.11a Events of `TrapEffectResolved`** (FR-096; lifecycle events left out, and a call on an active Rolling Stone has none; one row is one test).

| Label | Trap, targets (save) | Events after the lifecycle events | Req. | Path |
|---|---|---|---|---|
| failure, instantaneous | Collapsing Roof or Fire-Casting Statue, one (failure); Rolling Stone (active), one (failure; state stays `TrapActiveRolling`) | `[Started, Ended(instantaneous)]` (p. 199, p. 200, p. 201) | FR-096 | happy |
| roof, mixed | Collapsing Roof, A (failure), B (success), C (failure) | A's two events then C's two; none for B (p. 199) | FR-096 | unhappy (boundary) |
| roof, all succeed | Collapsing Roof, two (success) | none (p. 199) | FR-096 | unhappy (empty and minimal) |
| darts, two failures | Poisoned Darts, two (failure) | per target `[Started, Ended(instantaneous)]`, in target order (p. 200) | FR-096 | unhappy (rule interaction) |
| needle, net failure | Poisoned Needle, one (failure); Falling Net, one Large (failure) | `[Started]` only (pp. 199-201) | FR-096 | happy |
| net, Huge | Falling Net, one Huge (any) | none (p. 199) | FR-096 | unhappy (boundary) |
| pits | Hidden Pit, Spiked Pit, one | none (pp. 200-201) | FR-096 | unhappy (empty and minimal) |
| equals the evaluator | roof, statue, net, needle, stone; darts with rolled count 2 | events and `active` of each failing target equal `ResolveSaveGatedEffect` with the same effect, roll and save (darts: over `TrapDartsEffect.Expand(2)`) | FR-087, FR-096 | unhappy (rule interaction) |

## 5. Model

**Referenced, not defined:** `SaveGatedEffect`, `SaveGatedDamage(dice, type, SaveGatedPrintedAverage)` (`SaveGatedPrintedAverageOf(n)`), `SaveGatedCondition`, `SaveGatedDuration`, `SaveGatedFixedTime`, `SaveGatedHalfDamageOnSuccess`, `SaveGatedNoEffectOnSuccess`, `SaveGatedConsequences`, `ActiveSaveGatedEffect`, `SaveGatedSave` and variants, `SaveGatedExempt`, `SaveGatedEffectStarted`, `SaveGatedEffectEnded`, `SaveGatedDurationInstantaneous`, `ToolboxEvent`, the `toolbox` signal constants, `RollSaveGatedDamage`, `ResolveSaveGatedEffect` (`toolbox-hazard-effects.md`); `DurationUnit` (`content-schema-and-decoder.md`), `Skill`, `ConditionKind`, `DamageType`, `CreatureSize`, `ItemID`, `SpellID`, `TrapID` (`content`); `FallDistance`, `ObjectStatus`, `ObjectAffinities`, `NoAbilityScores`, `DamageThreshold` (environment spec); `Damage`, `DamageAdjustment`, `HitPoints` (`hit-points-and-damage.md`); `DifficultyClass`, `Outcome`, `CheckResult` (`d20`); `ability.Ability`, `modifier.Modifier`, `dice.Expression`, `dice.Result` (kernel); `Roller` (`dice`); `CharacterLevel`; `CombatantID`. This spec defines no damage-entry, condition-duration, save-then-effect or duration-unit type. Trap-specific effect types are prefixed `Trap` and wrap, never replace, the `SaveGated*` types.

**Value objects** (immutable, by value; the types are in section 3): `TrapDefinition` (`TrapID`, `Map<TrapLevelBand, TrapBandProfile>`, `TrapTrigger`, `TrapDuration`, `[]TrapCheckStep`, `[]TrapNeutralisation`, escape, destroy, slow and set options, `TrapAftermath`, `[]TrapContactEvent` (treated as a set), `TrapInitiativeData?`, with `profileFor` and `nearestProfileFor`); `TrapProfile` (definition id, band, severity, effect, steps, `TrapSaving`); `TrapTrigger` (`TrapTriggerKind`, `[]TrapTriggerMoment` (treated as a set)); `TrapDiscovery` (`[]TrapFeature` (treated as a set)). Results are closed interface with operation-prefixed variants (`shared-types-and-ownership.md` FR-017, FR-018). No entities or aggregates: a trap's state is a value held by the host.

**Domain services** (stateless, pure, silent): `TrapLifecycle`, `TrapLevelBand.ForLevel`, `ProfileFor`/`NearestProfileFor`, `TrapCheckStep.Assess`, `TrapNeutralisation.IsAvailable`, `TrapSlowOption.Assess`, `TrapSetOption.Assess`, and `TrapResolver` (`Plan`, `TrapPlanReady.NextRequest`/`complete`, `TrapSizeExemption.Exempts`, `TrapDartsEffect.Expand`; FR-095). `ResolveTrapEffect` only rolls what `TrapResolver` asks for, delegates and records signals.

**Optional fields.** Named states are never nil or a magic zero (FR-097). The remaining optional fields (`sizeExemption`, `area`, `spikes`, Initiative data, slow, set and destroy options) are the absence of an optional feature, mirror a missing key of the decoded `TrapEntry`, are read through an accessor returning `(value, ok)` and have no second meaning; no field of `ResolveTrapEffectResult` is optional.

**Events:** closed interface `TrapEvent` (FR-094) under `ToolboxEvent` (`shared-types-and-ownership.md` FR-019). Signal names are constants in the `toolbox` signal constants, owned by `toolbox-hazard-effects.md`; the trap constants listed in section 7 are added here (Q-17).

**Invariants and where enforced:**

| Invariant | Enforced by |
|---|---|
| At least one band, no gap | `TrapDefinition` constructor (FR-006, FR-007) |
| Timed count and reset limit at least 1 | `TrapTimed`, limited reset (FR-016) |
| Search uses Wisdom, Study uses Intelligence | `TrapCheckStep` (FR-040) |
| An inert trap never becomes armed by itself | `TrapLifecycle` (FR-021 to FR-023) |
| A motion never goes below 0 | `RollingMotion.slowed` (FR-053) |
| Levels are 1 to 20 | `CharacterLevel` only (FR-004) |
| A printed average is stored as printed | `SaveGatedDamage` (hazard FR-018), checked by bundled-data tests (FR-037) |
| Entries name only existing items and spells | whole-set validation, `content-schema-and-decoder.md` FR-125, sites of table 4.10a (FR-092) |
| Entry data never panics or errors out of `NewTrapDefinition` unreported | pre-validation (FR-093, table 4.10b) |
| Activation rules are not in the use case | `TrapResolver` (FR-095) |

**Failures:** programmer errors (invalid constructor argument, band gap) are an error wrapping `srderr.ErrInvalidInput` from `NewX` (`MustX` panics). Expected outcomes are results: `TrapProfile*`, `TrapStep*`, `TrapTransition*`, `TrapPlanRejected`, `TrapEffectRejected`; entry data that fails validation is the joined `TrapEntryProblem` error of FR-093. A dice port failure is the returned `*srderr.PortError` (FR-075). A host's integer level goes through `NewCharacterLevel` (`doc/rules/architecture.md` §6).

## 6. Interface

```go
// ResolveTrapEffect is the use case of section 4.8.
type ResolveTrapEffect struct{ /* unexported */ }

func NewResolveTrapEffect(roller Roller, opts ...Option) (ResolveTrapEffect, error)

func (u ResolveTrapEffect) Run(ctx context.Context, cmd ResolveTrapEffectCommand) (ResolveTrapEffectResult, error)
```

Signatures are illustrative; the requirements fix behaviour.

- **Synchronous:** `Roller` and `telemetry.Telemetry` are synchronous ports (`doc/rules/architecture.md` §4); `ctx` is first because the use case emits spans. `ContentSource` is not a port of this use case: the host loads a `trap` entry, converts it with `NewTrapDefinition` (FR-093) and passes the definition in. `NewResolveTrapEffect` returns an error wrapping `ErrInvalidInput` for a nil roller; options are `WithTelemetry` and `WithClock` (default `telemetry.Noop`).
- **Command and result:** section 4.8. Ports used: `Roller` (a dice port failure is returned wrapped once as `*srderr.PortError`, FR-075) and `telemetry.Telemetry`; no new port, double or contract suite. `ResolveTrapEffectResult` is a closed interface; a port failure or cancelled `ctx` is the `error`.
- **Host flow.** (0) Build the definition from the `trap` entry once. (1) `definition.NearestProfileFor(level)`; warn the table if not `WithinRange`. (2) Resolve each target's save named in `profile.Saving` through `d20.Resolver` (none for `TrapNoSave`; none for an armed Rolling Stone, FR-062). (3) `ResolveTrapEffect.Run`. (4) Pass each `Damage`, adjustment and `ConditionKind` to the `combat` damage and condition-apply functions; keep each `ActiveSaveGatedEffect` in `SaveGatedPhaseActive` and end it with `EndSaveGatedEffect`; for a pit use the `combat` fall functions with `FallDistance` and apply the Spiked Pit's Piercing `Damage`. (5) Store the returned `TrapState`. Rolling the stone's Initiative is the host's call to `initiative-and-actions.md` using FR-058. Other transitions are direct calls of the domain methods of section 5.
- **Exports:** package `toolbox` exports per FR-083; `toolbox/doc.go` lists inputs and outputs. No re-export package.
- **Layering:** the use case is in `toolbox/resolve_trap_effect_usecase.go`; the rest is in domain files of the same package, importing only the kernel packages and the upstream contexts' published API.

## 7. Observability

Answers to the seven questions (`doc/rules/observability.md` §2). The domain is **silent by design** (§3): pure, it exposes what happened through return values and events. The direct transitions (`startOfNextTurn`, `end`, `assess` of steps, slow and set options) have no port, randomness or I/O, their wrong-state outcomes are result values handled in the same call (FR-021 to FR-025, FR-048, FR-055, FR-057) and the state is returned unchanged. Activation is made inside the observed use case. If operators need to see the others they become use cases in a later change.

1. **What can go wrong?** The host passes a non-armed trap, too many targets, a duplicate target or a missing required save (rejections); the `Roller` fails; a house-built definition has bad numbers (`ErrInvalidInput` error at construction). A level outside the range is not an error (FR-060) but is worth seeing.
2. **How would an operator know?** A rejection is a failure result: error span status, `srd5e.outcome=failure`, `srd5e.toolbox.result=rejected_<reason>`. A roller failure increments the shared `srd5e.toolbox.errors`.
3. **What to diagnose it?** Severity, band, level fit, effect kind and rejection reason as attributes, present for every outcome (the profile resolves first, FR-090); the full breakdown is in the result, not telemetry.
4. **Counted or timed?** Activations by outcome, target count, latency.
5. **Existing instrumentation still correct?** Yes; `toolbox` adds nothing to another context. `srd5e.toolbox.operations`, `.operation_duration`, `.errors`, `.saves` and `.damage_rolled` are owned by `toolbox-hazard-effects.md` for the whole context (FR-079, FR-081, FR-089 cite them); this spec adds the operation value `resolve_trap_effect` and the result values below to that shared contract (Q-17) and defines only the signals in the table, which are added to `doc/observability/signals.md`.
6. **What to monitor?** Any rejection rate above zero (host sequencing bug); any `errors` with operation `resolve_trap_effect`; a high share of `belowTrapLevels` (informational).
7. **Noisy?** No logs: span and counters carry the outcome, and a failed save is a normal rule outcome (§7).

Added to the shared set: `srd5e.toolbox.operation` value `resolve_trap_effect`; `srd5e.toolbox.result` values `resolved` (`srd5e.outcome` `success`) and `rejected_trap_not_armed`, `rejected_too_many_targets`, `rejected_duplicate_target`, `rejected_missing_save_outcome` (`failure`); a roller failure uses the shared `error` result (`toolbox-hazard-effects.md` FR-078).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.toolbox.resolve_trap_effect` | span | `srd5e.outcome`; `srd5e.toolbox.result`; `srd5e.toolbox.trap.severity` (`nuisance`, `deadly`); `srd5e.toolbox.trap.level_band` (`levels_1_4`, `levels_5_10`, `levels_11_16`, `levels_17_20`); `srd5e.toolbox.trap.level_fit` (`within_range`, `below_trap_levels`, `above_trap_levels`); `srd5e.toolbox.trap.effect` (`saved`, `darts`, `pit`); `srd5e.toolbox.trap.rejection` (`trap_not_armed`, `too_many_targets`, `duplicate_target`, `missing_save_outcome`; only on rejection, never for a roller failure) | One resolution, its rule parameters and why it was refused; error status on a rejection | Sample rejected traces; investigate any `failure` |
| `srd5e.toolbox.trap.resolved` | counter, `{activation}` | `srd5e.outcome`; `srd5e.toolbox.trap.severity`; `.level_band`; `.level_fit`; `.effect`; `.rejection` (only on rejection) | The severity, band and effect mix the shared `operations` cannot carry; once per call (FR-079) | Alert on a non-zero `failure` rate with `rejection=trap_not_armed` or `missing_save_outcome`; shifts in the mix |
| `srd5e.toolbox.trap.targets` | histogram, `{target}` | `srd5e.toolbox.trap.effect` | Targets per resolution, once per call including rejections and zero targets (FR-089) | Unusually large target lists |

A call ended by a dice port error records only the shared `operations` (outcome `failure`, result `error`), `errors` and `operation_duration` (FR-081). No attribute carries a trap id, combatant id or free text: trap entries are content (FR-091) and host content is unbounded.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-005 | unit (`toolbox/*_test.go`), table-driven | Enum values; table 4.1a (the eight boundary levels); one test walks levels 1 to 20 and counts bands. Invalid integers are `NewCharacterLevel` rows in `derived-stats.md` |
| FR-004, FR-083, FR-084 | architecture (`internal/archtest`) | exported-name scan (no `int` entry point on `TrapLevelBand`; the FR-083 export list; generic and duplicate names); context map has `toolbox` with upstream `content`, `d20`, `character`, `combat`; domain purity and no consumer names; `TrapEvent` closed interface, implements `ToolboxEvent`, exhaustive `switch` compiles |
| FR-006 to FR-012 | unit, table-driven | Constructor rejections; equality, hash codes, defensively copied views, defensive copy; table 4.2a; a host-built trap stopping at 1-4 for FR-010 |
| FR-013 to FR-016 | unit | Enum sets (six trigger kinds, two moments); stone contacts are not trigger moments; count 0 and 1, limit 0 and 1 |
| FR-017 to FR-026 | unit, table-driven | Table 4.4a; state-error reasons; `TrapActiveRolling.motion` from `activate`; immutability and determinism |
| FR-027, FR-028, FR-029 to FR-036, FR-039, FR-085 | unit, table-driven | Bundled `traps` data through `BundledContentSource`: order, ids, table 4.5a; each entry converts with `NewTrapDefinition` (no `TrapCatalogue`); tables 4.5b per trap and band on definitions from the bundled entries: `SaveGatedEffect` fields (incl. `SaveGatedDurationInstantaneous` for roof, statue, darts, stone), `TrapTargeting`, pit `FallDistance`, aftermath, `TrapSaving` (`TrapNoSave` for pits) |
| FR-037, FR-038 | unit | Printed average equals `dice.Expression.average` except the pinned 13d8/57; `SaveGatedDamage(13d8, piercing, SaveGatedPrintedAverageOf(57))` accepted unchanged; no catalogue damage is `SaveGatedNoPrintedAverage`; pit depth / 10 equals the printed d6 count |
| FR-040 to FR-049 | unit, table-driven | Tables 4.6a and 4.6b per row; constructor rejections (Wisdom with Study, Intelligence with Search; `action` accepts Dexterity); success adds every feature; failure and `TrapStepConsequence`; repeated failure; prerequisite missing; `isAvailable` with and without the prerequisite |
| FR-050 to FR-058, FR-088 | unit, table-driven | Option data, `TrapEscapeRequirement` set, the stone's `ObjectStatus` (with `ObjectAffinities.standard()`, `NoAbilityScores`); table 4.7a; set option; `TrapInitiativeData` with `Modifier(8)`, 60 feet, both movement rules; stone contacts as data, empty elsewhere |
| FR-059 to FR-072, FR-074, FR-086, FR-090, FR-095 to FR-097 | unit (`TrapResolver`, scripted `dice.Result`s, no roller) | Tables 4.8a, 4.8c, 4.11a, one test per case; each trap's activation; empty, duplicate and too many targets; rejection order of FR-090; armed stone ignoring targets; `Exempts` at Large, Huge, Gargantuan; `Expand(1..3)`; `TrapDiceRequest` sequence of table 4.8b; no application of damage or conditions; pits return `TrapTargetFell`; closed-interface variants, no nil named state |
| FR-059, FR-061, FR-062, FR-064 to FR-066, FR-070 to FR-073, FR-075 to FR-078, FR-087, FR-096 | use case (`toolbox/*_usecase_test.go`) | Real domain, `dicetest.FixedRoller`/recording roller: one smoke test per trap (result equals the `TrapResolver` outcome for the same rolls); exact requests and order of table 4.8b (darts once for two failing targets, none when none fails); a failing roller double returns one wrapped `*srderr.PortError` (`errors.Is` reaches the double's error), inputs unchanged, a panicking telemetry double propagates; extreme fixed rolls; result accessors return copies; breakdown fields; per-target consequences, `active` and events equal `ResolveSaveGatedEffect` (darts over `Expand(n)`); scan: the `*_usecase.go` file imports no `Trap*` effect variant and no other use case |
| FR-079 to FR-082, FR-089 | application | Recording `telemetry.Telemetry`: span name, attributes (all present for a rejection; `rejection` only on a rejection), `operations` and `trap.resolved` once, `trap.targets` once (also rejections and zero targets), `operation_duration` once, `saves` per saving target (none for a rejection with saving targets or an armed stone's targets), `damage_rolled` once per call returning damage (shared damage counted once; none for a rejection, armed stone, net, Hidden Pit, no failing target against a no-half-damage effect); roller error: `errors` once, not logged, `operations` once with `failure`/`error`, `operation_duration` once, none of `trap.resolved`, `trap.targets`, `saves`, `damage_rolled`; no-op equals recording; names pinned against the `toolbox` signal constants and `doc/observability/signals.md` |
| FR-091, FR-092 | unit + architecture | Decoder test with the bundled array (shape, ids, `TrapEntry` implements `ContentEntry` with key `ContentKey(trap, id)` and `ContentReferences`; table 4.10a one test per row; a missing `ItemID` or `SpellID` is `unresolvedReference`); NOTICE and drift checks (`shared-types-and-ownership.md` FR-032, FR-033); `TrapEntry` imports no `toolbox` file |
| FR-093, FR-094 | unit, table-driven, plus a `Fuzz` test over the trap JSON decoder | One row per problem kind and per invariant of table 4.10b (one problem, no panic); a test walks the table against the constructors; any integer DC accepted; two problems in order; built definitions of the eight entries equal tables 4.5a, 4.5b, 4.6a, 4.6b. Events: equality, past-tense names, exhaustive type switch; `TrapEffectResolved.events` is a `[]ToolboxEvent` |
| (scenario) | scenario (`scenarios/`) | Public API only: a level-3 target triggers the Spiked Pit; the host resolves no save, calls `ResolveFall` with `fallDistance`, applies the Piercing `Damage` and the fall through `ApplyDamage`, calls `LandAfterFall` (Prone), ends with the expected hit points. A level-6 party against the Collapsing Roof with one failed and one successful save, both applied through `ApplyDamage` (FR-063, FR-064 end to end) |

- **Doubles and contracts:** `dicetest.FixedRoller` and the recording `telemetrytest` bundle from `dicetest`, `telemetrytest`, `contenttest`; no new double or contract suite.
- **Mutation testing** (`doc/rules/testing.md` §8): `gomu` on package `toolbox` MUST score at least 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (command in testing.md); the `*_usecase.go` file SHOULD. Boundary mutants are the main risk: band edges 4/5, 10/11, 16/17, the reset limit, 0 and 15 feet in `RollingMotion`, the Huge threshold, one versus two targets, the nearest-band choice below and above.
- **Deliberately not covered:** the host's own flow (steps 1, 2, 4, 5 of section 6) beyond the scenarios; Initiative rolling and the stone's movement (other specs); GM-judgement items (section 1). No FR id is left without a test.

## 9. Open questions and decisions

All questions are settled; none is open or escalated. Kept for citation: Q-1 and Q-2 by `shared-types-and-ownership.md` FR-045 and FR-047 (traps are a content kind, `toolbox` converts entries, FR-091 to FR-093; `toolbox` is in the context map); Q-3 `toolbox-hazard-effects.md` owns the save-then-effect model and this spec defines only trap-specific types (the earlier `TrapSaveForDamage`, `TrapSaveForCondition`, `TrapSaveForDarts`, `TrapFall`, `TrapRollingStoneEffect`, `TrapDamageEntry`, `TrapConditionDuration` are gone); Q-4 the environment spec uses `DifficultyClass` and `D20TestOutcome` (`shared-types-and-ownership.md` FR-002, FR-004); Q-7 one `CharacterLevel` per call, the host picks it; Q-10 pits hold a `FallDistance` and the host calls `ResolveFall` and `LandAfterFall` (FR-072); Q-15 unused duration forms, `doorknobTurned` and `other` are kept for host-built traps; Q-16 the host resolves saves because a use case never calls another (`shared-types-and-ownership.md` FR-044); Q-18 `toolbox-poison.md` is not reused beyond the shared model and is not a dependency; Q-19 a level outside the range resolves with the nearest band and reports `TrapLevelFit` (FR-060).

Settled, one line each (all decided; none escalated):

- **Q-5** The Rolling Stone's object statistics (p. 201) are an `ObjectStatus` with `NoAbilityScores` (objects have none, p. 178, FR-052); its "+8 bonus" on Initiative is a plain `Modifier(8)` (FR-058), a bonus rather than a Dexterity score, so the two do not conflict. The engine only returns data; a no-score object rolling Initiative is Q-13.
- **Q-6** Severity is stored per band; bands the SRD does not restate keep the base severity (FR-011). Interpretation: only the Rolling Stone states two severities (p. 201), and the scaling tables change damage and DC only, so no other entry is re-labelled.
- **Q-8** The `1d3` dart count and the per-dart dice are rolled once for all failing targets, each taking the same Poison `Damage` list (FR-071). Basis: "Damage against Multiple Targets", p. 16 ("roll the damage once for all the targets"); reading the dart count as part of the damage roll is an interpretation; order pinned by table 4.8b.
- **Q-9** "Resets at the start of the next turn": the engine exposes `startOfNextTurn` and leaves the turn to the host (FR-022); `TrapResetPending` blocks a second activation until reset (FR-021). Interpretation: the statue's own wording "first time on a turn" (p. 200) limits per-creature triggers, and the reset line is the only re-arming rule, so a second creature in the same turn finds the trap spent.
- **Q-11** The Net's statistics (p. 98) belong to `content-equipment.md`; entries store `ItemID`s only (`net`, `spikes_iron`, `thieves_tools`, table 4.10a) and an id equipment data lacks fails whole-set validation (FR-092). Trap text says "Iron Spike", the equipment table "Spikes, Iron": the id is `spikes_iron`. Structured Net statistics, if ever added there, amend FR-051 only. Prose only, no dependency: the Net's AC, Hit Points and Immunities are `NetProfile` (`deployed-and-restraining-gear.md` FR-044), and the host begins the Falling Net hold with `GearHolds.begin` using this trap's escape DC (that spec's FR-033).
- **Q-12** Where the SRD is silent (level above the bands, repeating a failed Detect or Disarm, re-setting after a failed Set the Trap) there is no extra restriction: FR-010, FR-047, FR-057, FR-060 (nearest band, `aboveTrapLevels`). Basis: "Parts of a Trap", p. 199 gives only advice on level mismatch and no attempt limit; limits are Game Master judgement (section 1).
- **Q-13** The Rolling Stone is NOT a combatant: `CombatantKind` has exactly two variants (`combat-foundations.md` FR-010) and an object is neither (p. 178). The trap returns `TrapInitiativeData` (FR-058) and the host rolls d20 + 8 and places the stone in its own ordering; the SRD text "the stone and all creatures nearby roll Initiative" (p. 201) is satisfied by the data. No change to `encounter-aggregate.md`.
- **Q-14** `end` accepts `destroyed` and `dispelled` (FR-024) and the host decides when they apply; "until the trap is destroyed or dispelled" (p. 199) is a duration, and what destroys or dispels a trap is the host's call (destroy options, FR-051, FR-052).
- **Q-17** Closed with `toolbox-hazard-effects.md` (its Q-15, Q-16): A-2 by `TrapDartsEffect.Expand` (FR-071; the hazard model has no per-hit multiplier); A-3 and A-5 are this spec's own section 7 list (trap signal names added to the `toolbox` signal constants, operation `resolve_trap_effect` and its result values); A-4 is satisfied by the published `SaveGatedPlanner`, `DiceRequired` and `SaveGatedEvaluator` (`EvaluateInitial`, `EvaluateRepeat`, `EvaluateEnd`; FR-087); A-6 is stated in FR-089 (a `SaveGatedDamageShared` amount is counted once per call). A-1 is settled by its FR-076 (FR-068); trap events as `part` files are withdrawn (`shared-types-and-ownership.md` FR-019, FR-062). No trap-specific stand-in type remains.
- **Q-20** Amendment to `content-schema-and-decoder.md`: row R3-traps reads "the eight sample traps as `traps` entries (`TrapEntry`, toolbox-traps.md FR-092)" (not `TrapCatalogue.srd`), and its reference-sites table lists the six rows of table 4.10a (`trap`, field, target kind `equipment` or `spell`, no constraint) with the pinned ids.
- **Q-21** `DurationUnit` is owned by `content-schema-and-decoder.md` (`shared-types-and-ownership.md` FR-065); this spec defines no duration-unit type.


## 10. Changelog impact

`CHANGELOG.md`, under `Unreleased`, **Added**: "Traps (`toolbox`): the eight example traps as content (kind `trap`, `traps` array) with level-banded scaling, check steps, lifecycle and `ResolveTrapEffect`; new signals `srd5e.toolbox.trap.*`." Not **BREAKING**: `toolbox` is new and no existing exported member, rule result or signal changes. The change also carries revision row R3-traps (`content-schema-and-decoder.md`), the `trap` kind of `content-source-port.md`, `data/srd_5_2_1/traps.json` with its embedded library and the new rows of `doc/observability/signals.md`; the context map and `AGENTS.md` already list `toolbox` (`shared-types-and-ownership.md` FR-045).
