# Persistent spell zones

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `content-schema-and-decoder.md`, `content-spells.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md`, `initiative-and-actions.md`, `magical-effects.md`, `spell-effect-data.md`, `spell-effects.md` |

Rank 28 in table 4.11 of `shared-types-and-ownership.md` (amendment RA-3): right of
`spell-effects` (27), the last spec it imports; same rank as `encounter-aggregate` and
`active-effects`, with which it shares no edge (equal rank cannot depend, FR-056). "Depends
on" lists only the specs whose types sections 5 and 6 import: `content-spells.md` (`Spell`,
`SpellID`, `SpellDuration`, `Spell.RequiresConcentration`; rank 6) and
`content-schema-and-decoder.md` (`DurationUnit`; rank 4) are the `content` imports of
FR-049. `environment-hazards-and-objects.md` (Objects, Difficult Terrain) is prose only
(Q-11); `casting-and-concentration.md` (Concentration) is a host fact, not an import
(FR-010). The content half of every zone (the facts a spell carries) is a change to
`spell-effect-data.md` (RA-1); this spec owns the runtime state and the rules.

## 1. Purpose and scope

`spell-effects.md` resolves a spell once, at the cast, and excludes "anything that lasts or
recurs" (its section 1, Q11). Twelve SRD spell texts say a creature acts "only once per
turn" (counted in the PDF, pp. 113-173), and thirteen more leave an area behind that acts
when a creature enters it, ends or starts its turn in it, or when the area moves: Spirit
Guardians, Cloudkill, Moonbeam, Flaming Sphere, Spike Growth, Wall of Fire and their kin
(table Z, 25 spells). Nothing owns that.
This spec is the **persistent zone**: a pure domain state `PersistentZone` (spell, caster,
`AreaShape`, rules, designated exemptions, a per-creature once-per-turn ledger, mobility,
duration) with state-in/state-out operations that answer, for one reported event, "which
recurring part of the spell is now due against which creature?". It rolls nothing and applies
nothing: a due effect is handed to `ResolveSpellEffects` (RA-2) and its plan to the existing
damage and condition use cases (`shared-types-and-ownership.md` FR-044).

**The map boundary** is `magical-effects.md`'s: the package models no map. The host reports
whether a creature is inside the zone (computed with `AreaMembership.assess`), which named
region of it, how far it travelled inside, and where the zone went.

**Ownership vs `active-effects.md`** (`shared-types-and-ownership.md` FR-048). Zones own
area-triggered recurrence (enters, ends or starts a turn in an area, the area moving), the
per-turn ledger and mobility of an area. `active-effects.md` owns timed effects on a
creature or place: repeat saves, `EndsOnDamage`, Concentration-bound conditions and stored
damage. A spell is one or the other. Flaming Sphere, Moonbeam and every table Z spell get a
zone fact set and no `EffectTemplate`; RA-6 withdraws the Flaming Sphere pilot and the
`CreatureInArea` damage audience's pilot use from `active-effects.md` (its FR-027, Q11).
`ZoneDuration` stays here (equal rank, no shared import); both specs end on a host-reported
Concentration loss, and since a spell is never both, no duration or Concentration code is
duplicated for one spell.

**Out of scope:**

- Casting, slots and Concentration state (`casting-and-concentration.md`): the host calls
  `EndZone` when Concentration breaks (FR-010). The spell's initial effect on appearance (a
  save and damage when the Sphere appears) is `ResolveSpellEffects`; this spec only lets the
  host record that application in the ledger (FR-030).
- Rolling saves and damage, Resistance, Concentration saves, riders written in a spell
  (Moonbeam ends a shape-shift, Sleet Storm breaks Concentration, Black Tentacles' escape
  check): `spell-effects.md`, `magical-effects.md` (`ShapeShifting`), `casting-and-concentration.md`.
  They stay text and make the spell `partial` there.
- Summoned creatures and spirits as creatures (stat blocks, Conjure Minor Elementals' attack
  bonus, p. 118). The area triggers of Conjure Animals, Conjure Celestial, Conjure Woodland
  Beings and Guardian of Faith ARE zones (table Z). Conjure Elemental (p. 118) is left out:
  its force-a-save trigger is gated on whether the spirit already has a Restrained creature,
  a spell-state fact `ZoneFacts` cannot express; a host may withhold events meanwhile.
- Tsunami (p. 171): "only once per round" needs a round-keyed ledger and a 50-foot forced
  drift with shrinking height; deferred (Q-12). Hallow (p. 138) lasts as static
  state-while-inside effects, with no trigger. Equipment areas (Oil flask, p. 98; Ball
  Bearings and Caltrops, p. 96) and traps belong to equipment and `toolbox`, though a host
  may reuse the ledger idea.
- Geometry: positions, distances, lines of effect, who "you can see" is. The host supplies
  facts (`ZoneCreatureFacts`).
- Walls as objects (AC, Hit Points, destruction): `environment-hazards-and-objects.md`.
  Detecting camouflaged Spike Growth (a Search action against the spell save DC, p. 164):
  `hiding-searching-and-senses.md`.
- Spells outside SRD 5.2.1. Coverage: every PDF spell with a once-per-turn clause or a
  recurring enter, end-turn or start-turn clause is in table Z except those listed above;
  nothing needed was found missing from the PDF.

## 2. Source

PDF page numbers equal printed page numbers (checked on pp. 116, 132, 150, 164, 165).
SRD 5.2.1: spell texts Black Tentacles and Blade Barrier, p. 113; Cloudkill, p. 116;
Conjure Animals, pp. 117-118; Conjure Celestial, p. 118; Conjure Woodland Beings,
pp. 118-119; Control Water (Whirlpool), pp. 119-120; Flaming Sphere, p. 132; Forbiddance,
p. 133; Grease and Guardian of Faith, p. 137; Gust of Wind, p. 138; Incendiary Cloud,
pp. 142-143; Insect Plague, p. 143; Moonbeam, pp. 150-151; Prismatic Wall, p. 155; Sleet
Storm, p. 163; Spike Growth, p. 164; Spirit Guardians, pp. 164-165; Stinking Cloud, p. 165;
Symbol, p. 167; Wall of Fire, p. 172; Wall of Thorns, p. 173; Web, p. 174; Zone of Truth,
p. 175 (excluded: Conjure Elemental, p. 118; Hallow, p. 138; Tsunami, p. 171). Glossary:
"Area of Effect", p. 177; "Concentration", p. 179; "Difficult Terrain", "Emanation",
p. 181; "Speed" ("Changes to Your Speeds"), p. 188; "Initiative", "Combat Step by Step",
"Round", p. 13; "Duration", "Combining Spell Effects", p. 106. Rule files: `doc/rules/architecture.md`
§2 to §6; `doc/rules/testing.md` §4, §5, §8; `doc/rules/observability.md` §2, §3.

Web research informed four judgements only (Q-1, Q-3, Q-4, Q-5) and nothing was copied: the
community reading of "once per turn" is one application per creature per initiative turn,
whoever's turn it is; whether "Combining Spell Effects" reaches repeated area damage is
contested (some read it for buffs only), so Q-5 leaves the call to the host.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| (a spell's lingering area) | `PersistentZone`, `PersistentZoneID` | Package terms. State of one cast. |
| "makes this save only once per turn" | `ZoneFacts.OncePerTurn`, `ZoneLedger`, `TurnMarker` | `TurnMarker(round, active)` names one turn of the `TurnTracker` (`TrackerTurnStarted`, `initiative-and-actions.md` FR-152). |
| "for the first time on a turn" | `ZoneTrigger.CreatureFirstEntryOnTurn` | Separate from once per turn (FR-028). |
| enters, ends its turn, starts its turn, area moves into its space, "for every 5 feet it travels" | `ZoneTrigger`: `CreatureEnters`, `CreatureEndsTurnInside`, `CreatureStartsTurnInside`, `ZoneMovesIntoSpace`, `TravelsWithin` | Owned by `spell-effect-data.md` (RA-1) with `ZoneFacts`, `ZoneRuleFact`, `ZoneTargetScope`, `ZoneMobility`, `ZoneRegionID`, `ZoneMovementFacts`, `DesignationPolicy`. |
| "designate creatures to be unaffected" | `ZoneExemptions` | Fixed at the cast (FR-007). |
| "you can force", "you can bathe" | `ZoneRuleFact.CasterChoice` | Optional application (FR-023). |
| "moves 10 feet away", "move the Cylinder up to 60 feet", "moves with the creature" | `ZoneMobility`: `StationaryZone`, `MovesWithOrigin`, `MovedByCaster`, `DriftsAtTurnStart` | FR-033 to FR-039. |
| Difficult Terrain, "Speed is halved" | `ZoneMovementEffects` | FR-045 to FR-049. |
| (what is due) | `ZoneEffectDue` | Part index, creature, trigger, units. |
| (events reported by the host) | `ZoneTriggerEvent` (closed): `CreatureEntered`, `CreatureEndedTurn`, `CreatureStartedTurn`, `ZoneMovedIntoSpace`, `CreatureTravelled` | Table E. |
| Area of Effect, Emanation | `AreaShape` (`spell-effect-data.md` FR-013), `OriginInclusionDecision` (`magical-effects.md` FR-016) | Reused. |

Package names with no SRD source: `PersistentZone`, `PersistentZoneID`, `ZoneLedger`,
`TurnMarker`, `ZoneTriggerEvent` and variants, `ZoneEffectDue`, `ZoneCreatureFacts`,
`ZoneDuration`, `ZoneEndReason`, `ZoneEvent`. Terms use SRD wording (`doc/rules/architecture.md` §3).

## 4. Requirements

Ids are stable. "Facts" are the zone's `ZoneFacts`; "part" is a `SpellEffectPart` of the
spell's model. Every operation is a pure function from state and input to a closed
operation-prefixed result (decisions FR-017); constructor violations are `*InvalidError`.

### 4.1 State and creation

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `PersistentZoneID` MUST be a non-blank string type (`type PersistentZoneID string`); `NewPersistentZoneID` MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`) for a blank value, and the zero value is invalid. | `doc/rules/architecture.md` §3 | unhappy (invalid input) |
| FR-002 | `PersistentZone` MUST hold its id, `SpellID`, caster `CombatantID`, `AreaShape`, `ZoneFacts`, `ZoneExemptions`, `ZoneLedger`, `ZoneDuration`, the cast `TurnMarker`, damage dealt and status (`Active`, `Ended(ZoneEndReason)`), be immutable and equal by value, and expose unmodifiable collections. | `doc/rules/architecture.md` §3 | happy |
| FR-003 | `ZoneFacts` MUST have at least one rule; every rule's `PartIndex` MUST name a part of the spell's `ModelledEffects`, else `CreateZoneRejected` with `PartIndexOutOfRange`; a `TravelsWithin` rule MUST carry `UnitFeet` of at least 1 and no other trigger MAY carry one (`*InvalidError`). | SRD 5.2.1, "Spike Growth" ("for every 5 feet"), p. 164 | unhappy (boundary) |
| FR-004 | `PersistentZones.create` with valid input MUST return `CreateZoneCreated(zone)` with status `Active`, an empty ledger, damage dealt 0, and emit one `ZoneCreated` event. | SRD 5.2.1, "Spirit Guardians", p. 164 | happy |
| FR-005 | A spell whose `SpellEffectModel` is `TextOnlyEffects` or has no zone facts MUST return `CreateZoneRejected` with `NoZoneFacts` and no event. | `spell-effect-data.md` (RA-1); SRD 5.2.1, "Effects", p. 106 | unhappy (missing content) |
| FR-006 | `CreateZone` MUST accept any caster `CombatantID` and return `CreateZoneCreated` for it: the zone trusts the host's caster id, with no cross-check against the cast (no `character` import). | `doc/rules/architecture.md` §2 | unhappy (rule interaction) |
| FR-007 | Designated exemptions MUST be accepted only when the facts' `DesignationPolicy` is `AtCastingOnly` (else `DesignationNotAllowed`), MUST be fixed for the zone's life (no operation changes them), and a repeated id MUST give `DuplicateExemption`. All problems MUST be listed in fixed order (class `invalid`). | SRD 5.2.1, "Spirit Guardians" ("When you cast this spell, you can designate creatures"), p. 164 | unhappy (invalid input) |
| FR-008 | When the facts' mobility is `MovesWithOrigin` (an Emanation on the caster), the caster MUST NOT be a target of the zone unless the command carries `CreatorIncludesOrigin`; for every other mobility the caster is an ordinary creature (a Cloudkill Sphere affects its caster). An Emanation's origin may be an object (Guardian of Faith's guardian: `EmanationArea(10)`, stationary); an object origin is no creature, so nothing is excluded. | SRD 5.2.1, "Emanation" (from "a creature or an object"; origin not included unless its creator decides otherwise), p. 181; "Spirit Guardians" ("Any other creature"), p. 164 | unhappy (rule interaction) |
| FR-009 | `ZoneDuration` MUST be `RoundsDuration(rounds)` (at least 1, else `NewRoundsDuration` returns an `*InvalidError`), `HostTimedDuration` or `UntilDispelled`. `ZoneDurationFromSpellDuration(SpellDuration)` MUST return a closed `ZoneDurationConverted(duration)` or `ZoneDurationRefused(DurationNotZone)`: `Timed` in rounds gives `RoundsDuration(n)`, in minutes or hours 10 and 600 rounds per unit, in any longer unit `HostTimedDuration`; `UntilDispelled` and `UntilDispelledOrTriggered` give `UntilDispelled`; `Instantaneous` and `Special` give the refusal, which `CreateZone` returns as `CreateZoneRejected(DurationNotZone)`. `CreateZone` MAY carry a `DurationOverride`: Symbol is made at activation by the host with `RoundsDuration(100)`, its stated 10 minutes after the trigger (case table 4.7b). | SRD 5.2.1, "Combat Step by Step" (a round is about 6 seconds), p. 13; "Duration", p. 106; "Symbol" ("Until dispelled or triggered", glyph glows 10 minutes), p. 167 | happy |
| FR-010 | The zone MUST record `EndsWithConcentration` from the spell (`Spell.RequiresConcentration`). The engine MUST NOT detect Concentration: the host calls `end(ConcentrationEnded)` when the creator loses it. | SRD 5.2.1, "Concentration" ("If the effect's creator loses Concentration, the effect ends"), p. 179 | happy |

### 4.2 Membership and who is affected

The host reports each creature as `ZoneCreatureFacts(creature, regions, IsEnemyOfCaster,
SeenByCaster)`. `regions` is the set of `ZoneRegionID`s the creature is in (the default region
is `body`); an empty set means outside the zone. A Wall of Fire's "within 10 feet of that side"
is a second region the host computes.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-011 | A creature MUST be inside a rule exactly when its `regions` contain the rule's region; an empty set is `NotInside` for every rule. | SRD 5.2.1, "Area of Effect", p. 177; "Wall of Fire", p. 172 | happy |
| FR-012 | A creature in `ZoneExemptions` MUST never have an effect due, whatever trigger, region or ledger state (skip `DesignatedExempt`). | SRD 5.2.1, "Spirit Guardians", p. 164 | happy |
| FR-013 | A rule with scope `EnemiesOfCaster` MUST skip a creature not reported as an enemy (`NotAnEnemy`); `CreaturesCasterCanSee` MUST skip one not reported as seen (`NotSeenByCaster`); `AnyCreature` MUST skip neither. | SRD 5.2.1, "Guardian of Faith" ("Any enemy"), p. 137; "Conjure Woodland Beings" ("a creature you can see"), p. 118 | unhappy (rule interaction) |
| FR-014 | The skip reasons MUST apply in this order, and only the first is reported per rule: `ZoneHasEnded`, `DesignatedExempt`, `OriginCreature` (FR-008), `NotInside`, `NotAnEnemy`, `NotSeenByCaster`, `AlreadyAffectedThisTurn`, `NotFirstEntryThisTurn`, `LessThanOneUnit`. | `doc/rules/testing.md` §4 (determinism) | unhappy (rule interaction) |

### 4.3 Triggers

Table E maps each reported event to the rule triggers it can satisfy.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-015 | Each `ZoneTriggerEvent` MUST be matched against every rule of the zone whose trigger it satisfies (table E); a rule whose trigger the event does not satisfy MUST be ignored without a skip reason. | SRD 5.2.1, "Cloudkill", p. 116; "Wall of Thorns", p. 173 | happy |
| FR-016 | `CreatureEntered` is the creature's own arrival into the zone. A creature that begins its turn already inside MUST NOT be matched by `CreatureEnters` or `CreatureFirstEntryOnTurn`; a zone that moves onto a creature is `ZoneMovedIntoSpace`, never `CreatureEntered`. | SRD 5.2.1, "Spirit Guardians" ("whenever the Emanation enters a creature's space and whenever a creature enters the Emanation"), p. 164 | unhappy (rule interaction) |
| FR-017 | One event MUST produce at most one `ZoneEffectDue` per `PartIndex`: two rules of one zone that match the same event and name the same part (Wall of Fire's body and damaging side) merge into one, in declaration order of the first. | SRD 5.2.1, "Wall of Fire" ("takes the same damage"), p. 172 | unhappy (rule interaction) |
| FR-018 | Due effects MUST be listed in rule declaration order and carry the zone id, `SpellID`, the zone's cast `TurnMarker`, caster, creature, `PartIndex`, trigger, `units` and the `CasterChoice` flag. | `doc/rules/testing.md` §4 (determinism) | happy |
| FR-019 | `CreatureTravelled(FeetInside)` MUST give `units = FeetInside / UnitFeet` for a `TravelsWithin` rule; `units` below 1 MUST skip with `LessThanOneUnit`; no remainder is carried to a later segment (Q-4). A negative distance MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`). | SRD 5.2.1, "Spike Growth", p. 164; "Breaking Up Your Move", p. 14 | unhappy (boundary) |
| FR-020 | The cause of the movement (own or forced) MUST NOT change a `TravelsWithin` result: the event has no cause field (Q-3). | SRD 5.2.1, "Spike Growth" ("When a creature moves into or within the area"), p. 164 | unhappy (rule interaction) |
| FR-021 | Every trigger other than `TravelsWithin` MUST give `units` 1. | SRD 5.2.1, "Cloudkill", p. 116 | happy |
| FR-022 | An operation on an `Ended` zone MUST leave it unchanged with no events and return, by family: `AssessZoneTrigger` and `ApplyOptionalZoneEffect` `AssessZoneTriggerEnded`; `MoveZone` `MoveZoneRefused(ZoneHasEnded)`; `RecordZoneCastApplication` `RecordCastApplicationZoneEnded`; `RecordZoneDamage` `RecordDamageZoneEnded` (all class `refused`); `EndZone` `EndZoneAlreadyEnded` (FR-040). `ZoneStartOfTurn` and `ZoneRoundStarted` return the zone unchanged with nothing due; `ZoneMovementEffectsFor` skips it (FR-044). | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-023 | A rule with `CasterChoice` MUST be returned in `OptionalDue`, not `due`, and the ledger MUST NOT be updated for it; `ApplyOptionalZoneEffect(zone, due)` records it (refused with `AlreadyAffectedThisTurn` when the ledger already holds the creature this turn). Declining is doing nothing (Q-2). | SRD 5.2.1, "Conjure Woodland Beings" ("you can force"), p. 118; "Conjure Celestial" ("you can bathe"), p. 118 | unhappy (rule interaction) |
| FR-024 | `AssessZoneTrigger` MUST record each mandatory due effect in the ledger in the zone it returns, and MUST emit one `ZoneTriggered` event per due effect (mandatory or optional). | `doc/rules/architecture.md` §4 (state in, new state and events out) | happy |

### 4.4 The once-per-turn and first-entry ledger

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-025 | When `OncePerTurn` is true, a creature MUST have at most one application per zone per `TurnMarker`, across all rules and triggers; a second is skipped `AlreadyAffectedThisTurn`. | SRD 5.2.1, "Cloudkill" ("A creature makes this save only once per turn"), p. 116; "Symbol" ("targeted only once per turn"), p. 167 | happy |
| FR-026 | A `TurnMarker` is equal when round and active combatant are equal. A different combatant's turn in the same round, or the same combatant's turn in a later round, MUST allow a new application (Q-1). | SRD 5.2.1, "Combat Step by Step", p. 13 (a turn is one combatant's) | unhappy (boundary) |
| FR-027 | When `OncePerTurn` is false, the ledger MUST NOT limit applications: Flaming Sphere, Spike Growth and Wall of Fire apply on every matching event. | SRD 5.2.1, "Flaming Sphere", p. 132; "Spike Growth", p. 164 | happy |
| FR-028 | `CreatureFirstEntryOnTurn` MUST match a creature's first `CreatureEntered` per `TurnMarker` only; leaving and re-entering in the same turn MUST skip `NotFirstEntryThisTurn`. This ledger is independent of the once-per-turn one, except that an `OncePerTurn` zone also applies FR-025. | SRD 5.2.1, "Web" ("The first time a creature enters the webs on a turn"), p. 174; "Wall of Fire", p. 172 | unhappy (rule interaction) |
| FR-029 | A ledger is per zone: the engine MUST NOT dedupe across zones, so a creature inside two zones MUST have a due from each, each carrying `SpellID`, zone id and cast marker (FR-018). For zones of one spell the host MUST apply "Combining Spell Effects" to the rolled results: the most potent applies, the later cast marker on a tie. Zones of different spells add. Potency needs the host's rolls, so the engine only supplies the keys (Q-5, interpretation). | SRD 5.2.1, "Combining Spell Effects", p. 106; "Spirit Guardians" ("makes this save only once per turn"), p. 165 | unhappy (rule interaction) |
| FR-030 | `RecordZoneCastApplication(zone, creature, marker)` MUST put the creature in the once-per-turn ledger for that marker (the appearance of the zone counts as the turn's application, Q-6); it MUST return `RecordCastApplicationNotOncePerTurn` unchanged for a zone without `OncePerTurn`. | SRD 5.2.1, "Incendiary Cloud" ("When the cloud appears ... only once per turn"), p. 142 | unhappy (rule interaction) |
| FR-031 | The ledger MUST hold at most one marker per creature (the latest); recording replaces an older one, so its size never exceeds the creatures seen. | `doc/rules/architecture.md` §3 | unhappy (boundary) |
| FR-032 | `NewTurnMarker` with a round below 1 MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`); one whose round is below a round already in the ledger for the creature MUST return `...StaleTurn` (class `invalid`), the zone unchanged. | `doc/rules/testing.md` §4 (invalid input) | unhappy (invalid input) |

#### Table L: ledger scenarios (FR-025 to FR-031; one test per row)

Zone: Spirit Guardians (`OncePerTurn`, rules: enters, EndsTurnInside, ZoneMovesIntoSpace). Creature X, caster C.

| Label | Sequence | Expected | FR | Path |
|---|---|---|---|---|
| L1 | X ends its turn inside (round 2, X) | due once | FR-025 | happy |
| L2 | X enters (round 2, X), then ends turn inside (round 2, X) | enter due; end `AlreadyAffectedThisTurn` | FR-025 | unhappy (rule interaction) |
| L3 | zone moves into X on (round 2, C); X ends turn on (round 2, X) | both due | FR-026 | unhappy (boundary) |
| L4 | zone moves into X, then enters again, both on (round 2, C) | first due; second skipped | FR-025 | unhappy (rule interaction) |
| L5 | X ends turn on (round 2, X) and (round 3, X) | both due | FR-026 | unhappy (boundary) |
| L6 | X exempt (designated), ends turn inside | skip `DesignatedExempt` | FR-012 | unhappy (rule interaction) |
| L7 | two zones of one spell, X in both, ends turn | two dues, same `SpellID`, different zone ids and cast markers; no engine dedupe (host applies p. 106) | FR-029 | unhappy (rule interaction) |
| L13 | two zones of different spells, X in both, ends turn | two dues; effects add | FR-029 | unhappy (rule interaction) |
| L8 | `RecordZoneCastApplication(X, (round 1, C))`, then `ZoneMovedIntoSpace` on (round 1, C) | skipped | FR-030 | unhappy (rule interaction) |
| L9 | same on a zone without `OncePerTurn` | `RecordCastApplicationNotOncePerTurn` | FR-030 | unhappy (boundary) |
| L10 | Web: X enters, leaves, enters on (round 1, X) | first due; second `NotFirstEntryThisTurn` | FR-028 | unhappy (rule interaction) |
| L11 | Web: X starts turn inside (round 2, X) after entering on (round 1, X) | start rule due | FR-016, FR-028 | happy |
| L12 | marker round 0 | `*InvalidError` | FR-032 | unhappy (invalid input) |

### 4.5 Mobility

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-033 | `MoveZone` on a `StationaryZone` MUST return `MoveZoneRefused(NotMovable)`, unchanged, no event. | SRD 5.2.1, "Spike Growth" (no movement clause), p. 164 | unhappy (state error) |
| FR-034 | `MoveZone` on a `MovesWithOrigin` zone MUST return `MoveZoneRefused(MovesWithOrigin)`: it follows its origin (the host moves the caster and reports `ZoneMovedIntoSpace` for creatures it now overlaps). | SRD 5.2.1, "Emanation" ("moves with the creature"), p. 181; "Spirit Guardians", p. 164 | unhappy (rule interaction) |
| FR-035 | `ZoneActionKind` is `BonusAction`, `MagicAction` or `WithCasterMovement` (the host sends it while the caster moves on its turn; no action is spent). A `MovedByCaster` zone MUST accept a `CasterMove(DistanceFeet, how)` only when `how` equals the facts' action kind, `0 < DistanceFeet <= MaxFeet`, the marker's active combatant is the caster, and (when the facts say `LaterTurnsOnly`) the marker differs from the cast marker. Refusal order for `MoveZoneRefused`: `ZoneHasEnded`, `NotMovable` (a `CasterMove` on any other mobility, or a `DriftMove` on a `MovedByCaster` zone), `MovesWithOrigin` (FR-034), `StoppedForTurn` (FR-038), `WrongActionKind`, `ExceedsMaxDistance`, `NotCastersTurn`, `NotOnLaterTurn`. A distance of 0 or below MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`). Spending the action is `initiative-and-actions.md`'s. | SRD 5.2.1, "Moonbeam" ("on later turns ... Magic action ... up to 60 feet"), p. 150; "Flaming Sphere" ("As a Bonus Action ... up to 30 feet"), p. 132; "Conjure Celestial" and "Conjure Animals" ("when you move on your turn, you can also move"), pp. 117-118 | unhappy (boundary) |
| FR-036 | A `DriftsAtTurnStart` zone MUST report `ZoneDriftDue(feet)` from `ZoneStartOfTurn` exactly when the marker's active combatant is the caster, and nothing on any other turn; `DriftsAtTurnStart(feet, DirectionRule)` has `DirectionRule` `AwayFromCaster` (Cloudkill) or `AwayFromCasterChosen` (Incendiary Cloud: caster picks the direction), and the host owns the geometry. The move is a `DriftMove(DistanceFeet)` accepted for `0 < DistanceFeet <= feet`; less than `feet` exists only so the host can report a blocked drift. | SRD 5.2.1, "Cloudkill" ("moves 10 feet away from you at the start of each of your turns"), p. 116; "Incendiary Cloud" ("away from you in a direction you choose"), pp. 142-143 | happy |
| FR-037 | An accepted move MUST return `MoveZoneMoved` with the same zone, and emit one `ZoneMoved` event; it MUST NOT touch the ledger or exemptions. | `doc/rules/architecture.md` §4 | happy |
| FR-038 | A `ZoneMovedIntoSpace` event on a zone whose facts say `StopsMoverOnContact` MUST mark the zone stopped for the current marker (`AssessZoneTriggerDue.ZoneStoppedForTurn` true); a later `MoveZone` on the same marker MUST be refused `StoppedForTurn`. A new marker clears it. | SRD 5.2.1, "Flaming Sphere" ("the sphere stops moving for the turn"), p. 132 | unhappy (rule interaction) |
| FR-039 | `ZoneMovedIntoSpace` MUST apply to the creature whatever it does next: it never satisfies `CreatureEnters` (FR-016), and a creature moved into does not need to be reported again when it later ends its turn there (that is `CreatureEndedTurn`). | SRD 5.2.1, "Moonbeam" ("when the spell's area moves into its space and when it enters"), p. 151 | happy |

#### Table M: moves (FR-033 to FR-038; one test per row)

| Label | Zone, event | Expected | FR | Path |
|---|---|---|---|---|
| M1 | Moonbeam, `CasterMove(60, MagicAction)`, caster's later turn | moved | FR-035 | happy |
| M2 | Moonbeam, same on the cast marker | `NotOnLaterTurn` | FR-035 | unhappy (rule interaction) |
| M3 | Moonbeam, `CasterMove(61, MagicAction)` | `ExceedsMaxDistance` | FR-035 | unhappy (boundary) |
| M4 | Moonbeam, `CasterMove(30, BonusAction)` | `WrongActionKind` | FR-035 | unhappy (invalid input) |
| M5 | Flaming Sphere, `CasterMove(30, BonusAction)` on the cast marker | moved | FR-035 | happy |
| M6 | Flaming Sphere, move on another combatant's marker | `NotCastersTurn` | FR-035 | unhappy (state error) |
| M7 | Flaming Sphere, contact then `CasterMove(5, ...)` same marker | `StoppedForTurn` | FR-038 | unhappy (rule interaction) |
| M8 | Cloudkill, `ZoneStartOfTurn` on caster's turn | `ZoneDriftDue(10)` | FR-036 | happy |
| M9 | Cloudkill, `ZoneStartOfTurn` on another's turn | none | FR-036 | unhappy (boundary) |
| M10 | Cloudkill, `DriftMove(11)` | refused `ExceedsMaxDistance` | FR-036 | unhappy (boundary) |
| M11 | Spirit Guardians, any move | `MovesWithOrigin` | FR-034 | unhappy (rule interaction) |
| M12 | Spike Growth, any move | `NotMovable` | FR-033 | unhappy (state error) |
| M13 | Conjure Celestial, `CasterMove(30, WithCasterMovement)` on caster's marker | moved | FR-035 | happy |
| M14 | Conjure Celestial, same on another combatant's marker | `NotCastersTurn` | FR-035 | unhappy (state error) |
| M15 | Conjure Animals, `CasterMove(31, WithCasterMovement)` | `ExceedsMaxDistance` | FR-035 | unhappy (boundary) |
| M16 | Moonbeam, `DriftMove(10)`; Cloudkill, `CasterMove(10, BonusAction)` | `NotMovable` | FR-035 | unhappy (rule interaction) |
| M17 | Ended zone, any move | `ZoneHasEnded` | FR-022 | unhappy (state error) |
| M18 | Flaming Sphere stopped this marker, `CasterMove(30, BonusAction)` by a wrong kind | `StoppedForTurn` (before `WrongActionKind`) | FR-035 | unhappy (rule interaction) |

### 4.6 Lifecycle

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | `EndZone(zone, reason)` on an active zone MUST return `EndZoneEnded` with status `Ended(reason)`, one `ZoneEnded` event, the ledger and facts kept; `ZoneEndReason` is `ConcentrationEnded`, `DurationElapsed`, `dispelled`, `dispersed` (strong wind, a collapsed web), `BudgetSpent`, `CasterEnded`. On an ended zone it MUST return `EndZoneAlreadyEnded` (class `refused`), unchanged. | SRD 5.2.1, "Cloudkill" ("until strong wind ... disperses it"), p. 116; "Concentration", p. 179; "Web", p. 174 | happy |
| FR-041 | `ZoneRoundStarted(zone, round)` on a `RoundsDuration` zone MUST end it with `DurationElapsed` when `round >= CastRound + rounds` (Q-8) and otherwise return it unchanged; `HostTimedDuration` and `UntilDispelled` zones are never ended by it. A round below the cast round MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`). | SRD 5.2.1, "Duration", p. 106; "Combat Step by Step", p. 13 | unhappy (boundary) |
| FR-042 | `RecordZoneDamage(zone, amount)` on a zone with a `DamageBudget` MUST add the amount the host reports and, once the total reaches the budget, end the zone with `BudgetSpent` in the same result; on a zone without a budget it MUST return `RecordDamageNoBudget` (class `refused`); a negative amount MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`). The engine sums the host's figure and does not decide what counts as dealt (Q-10). | SRD 5.2.1, "Guardian of Faith" ("vanishes when it has dealt a total of 60 damage"), p. 137 | unhappy (boundary) |
| FR-043 | `ZoneStartOfTurn(zone, marker)` MUST end nothing and change nothing except to report the drift (FR-036) and clear a stopped mark from an earlier marker (FR-038). | SRD 5.2.1, "Cloudkill", p. 116 | happy |

#### Case table 4.7b: duration conversion (FR-009)

| Case | Spell duration | Expected | Path |
|---|---|---|---|
| Spirit Guardians, p. 164 | Concentration, up to 10 minutes | `RoundsDuration(100)`, `EndsWithConcentration` | happy |
| Flaming Sphere, p. 132 | Concentration, up to 1 minute | `RoundsDuration(10)` | happy |
| Web, p. 174 | Concentration, up to 1 hour | `RoundsDuration(600)` | unhappy (boundary) |
| Guardian of Faith, p. 137 | 8 hours | `RoundsDuration(4800)` | unhappy (boundary) |
| a duration in days | 7 days | `HostTimedDuration` | unhappy (boundary) |
| a duration in rounds | Concentration, up to 6 rounds (Tsunami, p. 171, as a fixture) | `RoundsDuration(6)` | unhappy (boundary) |
| a spell lasting "Until dispelled" (Continual Flame, p. 119) | until dispelled | `UntilDispelled` | unhappy (rule interaction) |
| Symbol, p. 167 | Until dispelled or triggered | `UntilDispelled`; host passes `DurationOverride` `RoundsDuration(100)` at activation | unhappy (rule interaction) |
| an Instantaneous spell | Instantaneous | `CreateZoneRejected(DurationNotZone)` | unhappy (rule interaction) |
| a Special-duration spell | Special | `CreateZoneRejected(DurationNotZone)` | unhappy (missing content) |

### 4.7 Movement effects on creatures inside

`ZoneMovementFacts(DifficultTerrain, SpeedHalved, ExtraFeetPerFoot)` is part of the facts. The
engine returns the merged facts; the host passes them to `initiative-and-actions.md`
(`Terrain`, `ExtraCostPerFoot` FR-077, `UpdateMovementSpeeds` FR-116) and changes no Speed.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-044 | `ZoneMovementEffectsFor(zones, creature)` MUST consider only active zones in which the creature is in the `body` region and is not exempt. | SRD 5.2.1, "Spirit Guardians" ("Any other creature's Speed is halved in the Emanation"), p. 164 | happy |
| FR-045 | `DifficultTerrain` MUST be true when any considered zone says so, and is never cumulative. | SRD 5.2.1, "Difficult Terrain" ("isn't cumulative"), p. 181; "Spike Growth", p. 164; "Web", p. 174 | unhappy (rule interaction) |
| FR-046 | `SpeedHalved` MUST be true when any considered zone says so; two zones MUST NOT halve twice (Q-7). Special speeds follow by the host. | Interpretation (no SRD text on stacked halving; the zone texts say "Speed is halved", a state, and Q-7 reads it as one by analogy with Difficult Terrain, p. 181); SRD 5.2.1, "Speed" ("Changes to Your Speeds": special speeds change with Speed), p. 188 | unhappy (rule interaction) |
| FR-047 | `ExtraFeetPerFoot` MUST be the largest among considered zones (0 when none), by analogy with FR-045. The extra cost per foot moved MUST be the larger of 1 (when `DifficultTerrain`) and `ExtraFeetPerFoot`, never their sum: Wall of Thorns' "4 feet of movement for every 1 foot" is 3 extra, so Thorns plus Spike Growth costs 4 per foot, not 5. | SRD 5.2.1, "Wall of Thorns", p. 173; "Difficult Terrain" (1 extra foot; not cumulative), p. 181 | unhappy (boundary) |
| FR-048 | A creature outside every zone, or inside only ended or exempting zones, MUST get `ZoneMovementEffects.none`. | `doc/rules/testing.md` §4 (empty) | unhappy (empty and minimal) |

### 4.8 Layering, events and reuse

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-049 | Every type MUST live in package `combat`, be pure and synchronous, and import only the kernel packages and the published API of `hit-points-and-damage.md`, `conditions.md`, `magical-effects.md`, `spell-effects.md`, `initiative-and-actions.md` and `content` (`SpellID`, `AreaShape`, `SpellDuration`, the zone facts). It MUST NOT import any adapter package, `telemetry`, `character` services, `toolbox`, `hiding-searching-and-senses.md` or `encounter-aggregate.md`. | `doc/rules/architecture.md` §1, §2, §9; `shared-types-and-ownership.md` FR-013, FR-044 | unhappy (forbidden edge) |
| FR-050 | The `ZoneEvent` family MUST be a closed interface (unexported marker method; the variants are listed in its doc comment) whose variants also satisfy `CombatEvent`, and be past tense: `ZoneCreated` (`zone_created`), `ZoneTriggered` (`zone_triggered`), `ZoneMoved` (`zone_moved`), `ZoneEnded` (`zone_ended`); `Combatants()` is the caster for created, moved and ended, the affected creature for triggered; every `Kind()` is unique in the package. | `combat-foundations.md` FR-015 to FR-019; `shared-types-and-ownership.md` FR-019 | unhappy (name collision) |
| FR-051 | The package code MUST contain no spell id literal and no per-spell branch: every behaviour comes from the zone facts, so a custom spell with the same facts behaves the same. | `spell-effect-data.md` section 5.1 (data, not strategy) | unhappy (rule interaction) |
| FR-052 | An operation MUST NOT alter its inputs, MUST give equal results for equal inputs, and return fresh slices (mutating one changes nothing held). | `doc/rules/testing.md` §4 (immutability, determinism) | unhappy (immutability) |
| FR-053 | Host input whose validity depends on state (a stale marker FR-055, a move or record on an ended zone FR-022, a repeated exemption) MUST be an operation-prefixed result classified by `combat-foundations.md` FR-031 (malformed input `invalid`, state-dependent `refused`); an `*InvalidError` is kept for constructor invariants (FR-001, FR-003, FR-009, FR-019, FR-032, FR-035, FR-041, FR-042). | `doc/rules/architecture.md` §6; `magical-effects.md` FR-144 | unhappy (invalid input) |
| FR-054 | A `ZoneEffectDue` MUST be resolvable by `ResolveSpellEffects` restricted to its `PartIndex` with `units` instances (RA-2); this spec MUST NOT roll, save or apply. A condition a part applies "until the spell ends" ends when the host calls `EndZone` and removes it through `conditions.md`; no operation here removes a condition. | `shared-types-and-ownership.md` FR-044; SRD 5.2.1, "Black Tentacles" ("until the spell ends"), p. 113 | unhappy (rule interaction) |
| FR-055 | Any operation taking a `TurnMarker` (`AssessZoneTrigger`, `ApplyOptionalZoneEffect`, `MoveZone`, `ZoneStartOfTurn`, `RecordZoneCastApplication`) with a round below the zone's cast marker round MUST return the family's `...StaleTurn` result (class `invalid`), the zone unchanged, no events. `ZoneRoundStarted` with such a round returns an `*InvalidError` (FR-041). | `doc/rules/testing.md` §4 (invalid input); SRD 5.2.1, "Combat Step by Step" (rounds only advance), p. 13 | unhappy (invalid input) |

#### Table E: events and the triggers they satisfy (FR-015, FR-016, FR-019)

| Event | Triggers matched | Notes |
|---|---|---|
| `CreatureEntered(creature)` | `CreatureEnters`, `CreatureFirstEntryOnTurn` | The creature's own arrival. |
| `CreatureEndedTurn(creature)` | `CreatureEndsTurnInside` | Host reports at the end of the creature's turn. |
| `CreatureStartedTurn(creature)` | `CreatureStartsTurnInside` | Host reports at the start of the creature's turn. |
| `ZoneMovedIntoSpace(creature)` | `ZoneMovesIntoSpace` | After a `ZoneMoved`, a drift or the origin's move. |
| `CreatureTravelled(creature, FeetInside)` | `TravelsWithin` | One segment per report. |

#### Table Z: SRD zone spells and their facts (RA-1 data; one fixture row each)

`once` = `OncePerTurn`. Region `side` is a band the host computes. Part index is the recurring
part of the spell's model. Damage and save lines are `spell-effect-data.md`'s, not repeated.

| Spell, page | Shape, mobility | Rules (trigger, scope) | Ledger | Other facts | Path |
|---|---|---|---|---|---|
| Spirit Guardians, pp. 164-165 | Emanation 15, `MovesWithOrigin` | enters, EndsTurnInside, ZoneMovesIntoSpace; any | once | designation at cast; Speed halved; caster excluded | happy |
| Cloudkill, p. 116 | Sphere 20, `DriftsAtTurnStart(10, AwayFromCaster)` | enters, EndsTurnInside, ZoneMovesIntoSpace; any | once | applies on appearance (FR-030); strong wind ends | happy |
| Incendiary Cloud, pp. 142-143 | Sphere 20, `DriftsAtTurnStart(10, AwayFromCasterChosen)` | same as Cloudkill | once | as Cloudkill | happy |
| Insect Plague, p. 143 | Sphere 20, stationary | FirstEntryOnTurn, EndsTurnInside; any | once | Difficult Terrain; on appearance | unhappy (rule interaction) |
| Moonbeam, pp. 150-151 | Cylinder 5x40, `MovedByCaster(60, MagicAction, LaterTurnsOnly)` | enters, EndsTurnInside, ZoneMovesIntoSpace; any | once | on appearance | happy |
| Flaming Sphere, p. 132 | Sphere (5 ft diameter), `MovedByCaster(30, BonusAction)` | EndsTurnInside in region `within5`; ZoneMovesIntoSpace; any | none | stops on contact | happy |
| Spike Growth, p. 164 | Sphere 20, stationary | TravelsWithin (unit 5); any | none | Difficult Terrain | happy |
| Wall of Fire, p. 172 | Line wall, stationary | FirstEntryOnTurn in `body`; EndsTurnInside in `body`; EndsTurnInside in `side` (three rules, one part; merged by FR-017); any | none | one side damages; on appearance | unhappy (rule interaction) |
| Wall of Thorns, p. 173 | wall, stationary | FirstEntryOnTurn, EndsTurnInside; any | once | 4 feet per foot (extra 3) | unhappy (boundary) |
| Black Tentacles, p. 113 | 20-foot square, stationary | enters, EndsTurnInside; any | once | Difficult Terrain; on appearance | happy |
| Blade Barrier, p. 113 | wall, stationary | enters, EndsTurnInside; any | once | on appearance | happy |
| Web, p. 174 | Cube 20, stationary | FirstEntryOnTurn, StartsTurnInside; any | none | Difficult Terrain | unhappy (rule interaction) |
| Sleet Storm, p. 163 | Cylinder 20x40, stationary | FirstEntryOnTurn, StartsTurnInside; any | none | Difficult Terrain | unhappy (rule interaction) |
| Stinking Cloud, p. 165 | Sphere, stationary | StartsTurnInside; any | none | none | happy |
| Conjure Woodland Beings, pp. 118-119 | Emanation 10, `MovesWithOrigin` | ZoneMovesIntoSpace, enters, EndsTurnInside; seen; `CasterChoice` | once | none | unhappy (rule interaction) |
| Conjure Celestial, p. 118 | Cylinder 10x40, `MovedByCaster(30, WithCasterMovement)` | same; seen; `CasterChoice` | once | none | unhappy (rule interaction) |
| Conjure Animals, pp. 117-118 | Emanation 10 on the pack (object origin), `MovedByCaster(30, WithCasterMovement)` | ZoneMovesIntoSpace, enters, EndsTurnInside; seen; `CasterChoice` | once | Advantage on Strength saves near the pack is not a zone fact | unhappy (rule interaction) |
| Guardian of Faith, p. 137 | `EmanationArea(10)` on the guardian (object origin), stationary | FirstEntryOnTurn, StartsTurnInside; enemies | none | `DamageBudget` 60 | unhappy (boundary) |
| Grease, p. 137 | 10-foot square, stationary | enters, EndsTurnInside; any | none | Difficult Terrain; Prone save on appearance | happy |
| Gust of Wind, p. 138 | Line 60x10 from the caster, stationary (a later-turn redirect is host geometry; no zone state) | EndsTurnInside; any | none | push and the 2-feet-per-foot cost toward the caster are directional: host | happy |
| Control Water (Whirlpool), pp. 119-120 | 50x50 square, stationary | FirstEntryOnTurn, EndsTurnInside; any | none | none | unhappy (rule interaction) |
| Forbiddance, p. 133 | stationary | FirstEntryOnTurn, EndsTurnInside; any (the host reports only creatures of the chosen types) | none | none | unhappy (rule interaction) |
| Zone of Truth, p. 175 | Sphere 15, stationary | FirstEntryOnTurn, StartsTurnInside; any | none | none | happy |
| Prismatic Wall, p. 155 | wall, stationary | enters region `within20`, StartsTurnInside region `within20`; any (the host reports only creatures that can see the wall; designated creatures exempt) | none | designation at cast | unhappy (rule interaction) |
| Symbol, p. 167 | Sphere, stationary | FirstEntryOnTurn, EndsTurnInside; any | once | on activation | unhappy (rule interaction) |

Conjure Celestial and Conjure Animals move with the caster's move ("when you move on your
turn, you can also move the Cylinder up to 30 feet"): `MovedByCaster(30, WithCasterMovement)`,
the third action kind (FR-035), never `MovesWithOrigin`. Symbol (p. 167) is Sphere, stationary,
FirstEntryOnTurn and EndsTurnInside, `once`, created by the host at activation (FR-009).
Hallow, Conjure Elemental and Tsunami are out of scope (section 1).

## 5. Model

Pure, synchronous and silent: state in, new state and events out. Owners of reused types are
named in sections 1 to 3 (`CombatantID`, `CombatEvent`, `SpellID`, `AreaShape`).

- **Value objects:** `PersistentZone`, `PersistentZoneID` (a string type with a validating constructor),
  `ZoneExemptions`, `ZoneLedger` (creature to latest `TurnMarker`, plus the first-entry map),
  `TurnMarker`, `ZoneDuration` (closed: `RoundsDuration`, `HostTimedDuration`,
  `UntilDispelled`), `ZoneEndReason`, `ZoneCreatureFacts`, `ZoneTriggerEvent` (closed),
  `ZoneEffectDue`, `ZoneSkipReason`, `ZoneMove` (closed: `CasterMove`, `DriftMove`),
  `ZoneDriftDue`, `ZoneMovementEffects`, `ZoneDurationConversion` (closed:
  `ZoneDurationConverted`, `ZoneDurationRefused`). Content-side (RA-1,
  `spell-effect-data.md`): `ZoneFacts`, `ZoneRuleFact`, `ZoneTrigger`, `ZoneTargetScope`,
  `ZoneMobility`, `ZoneRegionID`, `ZoneMovementFacts`, `DesignationPolicy`, `ZoneActionKind`
  (`BonusAction`, `MagicAction`, `WithCasterMovement`), `DriftDirectionRule`
  (`AwayFromCaster`, `AwayFromCasterChosen`).
- **Events:** closed `ZoneEvent` interface whose variants satisfy `CombatEvent` (FR-050).
- **Pure functions** (no service type, no state): `CreateZone`, `AssessZoneTrigger`, `ApplyOptionalZoneEffect`,
  `RecordZoneCastApplication`, `MoveZone`, `ZoneStartOfTurn`, `ZoneRoundStarted`, `RecordZoneDamage`,
  `EndZone`, `ZoneMovementEffectsFor`.
- **Invariants** (`NewX` constructors returning `(T, error)`): FR-001, FR-003, FR-009, FR-019, FR-032, FR-035, FR-041,
  FR-042. Operations that can receive malformed input (a negative amount, a distance of 0 or below, a round below the cast round for `ZoneRoundStarted`) return `(result, error)` with an `*InvalidError`; nothing panics. **Failures:** state-dependent and malformed-input outcomes are operation-prefixed
  closed results (`CreateZoneResult`, `AssessZoneTriggerResult`, `MoveZoneResult`,
  `EndZoneResult`, `RecordCastApplicationResult`, `RecordDamageResult`; each family has
  a `...StaleTurn` variant, FR-055, and an ended-zone refusal, FR-022); no
  `*srderr.PortError`: no port.
- **Host protocol** (documented, not enforced): the host owns the spell's `ZoneFacts` through
  the `Spell`, holds the `PersistentZone`s beside its other per-creature values (as
  `magical-effects.md` Q-11), reports events from its own geometry, feeds `TrackerTurnStarted`
  and `TrackerRoundStarted` into `ZoneStartOfTurn` and `ZoneRoundStarted`, and calls `EndZone` when
  Concentration breaks or the spell is dispelled.

## 6. Interface

Every operation is a synchronous pure function (no use case, port, `context.Context` or dice). Exported
from package `combat` only (no re-export); the name scan must find no
duplicate name: `TurnMarker`, `ZoneEvent` and the result names are unique, and the
content-side names are exported by package `content`, not here. Handoff types:
`PersistentZone`, `ZoneEffectDue`, `ZoneMovementEffects`, `ZoneEvent`. No test doubles, no
contract suite.

## 7. Observability

Silent by design: the domain is pure (`doc/rules/observability.md` §3) and this spec owns no
signal. Telemetry is the use-case layer's through the shared `CombatTelemetry`
(`combat-foundations.md` FR-029 to FR-035); amendment RA-4 to `combat-action-resolution.md`
adds one use case per operation family and these `CombatOperation` names:
`create_persistent_zone`, `assess_zone_trigger`, `move_persistent_zone`,
`end_persistent_zone`, and the closed `srd5e.combat.result` values below (to
`doc/observability/signals.md`). Result values carry no ids, amounts or distances.
Operation mapping: `assess_zone_trigger` wraps `AssessZoneTrigger`, `ApplyOptionalZoneEffect` and
`RecordZoneCastApplication`; `move_persistent_zone` wraps `MoveZone` and `ZoneStartOfTurn`;
`end_persistent_zone` wraps `EndZone`, `ZoneRoundStarted` and `RecordZoneDamage` (the latter two
may end a zone, `completed` whether or not they did). `ZoneMovementEffectsFor` is a pure query
read inside the host's movement use case and has no operation of its own.

| Operation | Variant | `srd5e.combat.result` | Class |
|---|---|---|---|
| `create_persistent_zone` | `CreateZoneCreated` | `completed` | applied |
| | `CreateZoneRejected` (`DesignationNotAllowed`, `DuplicateExemption`, `PartIndexOutOfRange`, `DurationNotZone`) | `zone_create_invalid` | invalid |
| | `CreateZoneRejected` (`NoZoneFacts`) | `zone_no_facts` | refused |
| `assess_zone_trigger` | `AssessZoneTriggerDue` (effects, none due, optional); `RecordCastApplicationRecorded` | `completed` | applied |
| | `AssessZoneTriggerEnded`, `RecordCastApplicationZoneEnded`, `RecordCastApplicationNotOncePerTurn` | `zone_refused` | refused |
| | `AssessZoneTriggerStaleTurn`, `RecordCastApplicationStaleTurn` | `zone_stale_turn` | invalid |
| `move_persistent_zone` | `MoveZoneMoved`; `ZoneStartOfTurn` report (drift or none) | `completed` | applied |
| | `MoveZoneRefused` (any reason) | `zone_move_refused` | refused |
| | `MoveZoneStaleTurn`, `StartOfTurnStaleTurn` | `zone_stale_turn` | invalid |
| `end_persistent_zone` | `EndZoneEnded`; `ZoneRoundStarted` and `RecordZoneDamage` accepted | `completed` | applied |
| | `EndZoneAlreadyEnded` | `zone_already_ended` | refused |
| | `RecordDamageNoBudget`, `RecordDamageZoneEnded` | `zone_refused` | refused |

The seven questions (`doc/rules/observability.md` §2), one line each:

1. **What can go wrong?** The host reports wrong geometry or events twice (a double application is prevented by the ledger but a missed event is silent), forgets to call `EndZone`, passes a stale turn marker, or builds a zone from facts that disagree with the spell text (caught offline by `spell-effect-data.md` bundle checks).
2. **How would an operator know?** `srd5e.combat.operations` by operation and result once RA-4 lands; `invalid` sets the span error status, `refused` does not.
3. **Diagnosis:** results and events carry the deciding facts: skip reason, trigger, units, end reason.
4. **Counted or timed:** each call by operation and result, through `CombatTelemetry`.
5. **Existing instrumentation still correct?** Yes; no signal changes, and no use case is called from another (`shared-types-and-ownership.md` FR-044).
6. **Monitor:** a sustained `zone_stale_turn` or `zone_create_invalid` share means the host wiring is wrong.
7. **Noise?** None: no log is added.

## 8. Test plan

All unit tests in `combat/*_test.go` (package `combat_test`) (no ports, doubles or observability tests,
`doc/rules/observability.md` §9), except architecture tests in `internal/archtest` and one
scenario. Fixtures are built from table Z with hand-made `ZoneFacts` (the bundle is
`spell-effect-data.md`'s).

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-006 | unit | Blank id; any caster id creates (FR-006); value equality and immutability; `PartIndexOutOfRange`, `NoZoneFacts`, unit-feet invariant; created zone state and event |
| FR-007, FR-008 | unit | Designation allowed, not allowed, duplicates, all problems listed in order; caster excluded for a `MovesWithOrigin` zone, included with `CreatorIncludesOrigin`, ordinary for a Sphere |
| FR-009, FR-010 | unit | Table 4.7b one test per row (rounds, days, Instantaneous, Special, Symbol override, UntilDispelledOrTriggered); `EndsWithConcentration` flag; no Concentration detection |
| FR-011 to FR-014 | unit | Regions, exempt, enemy and seen scopes; the fixed skip order with a creature failing several tests at once |
| FR-015 to FR-021 | unit | Table E; starts-inside is not an entry; merge of two rules on one part (Wall of Fire); declaration order; Spike Growth segment of 0, 4, 5, 9, 10 feet, negative feet; no cause field (a compile-level absence, checked by the exhaustive `switch` over the event) |
| FR-022, FR-023, FR-024 | unit | Ended zone per operation family (FR-022, table M M17); stale marker per family (FR-055); round below the cast round; optional due not recorded, `ApplyOptionalZoneEffect` records and refuses a repeat; mandatory due recorded; one `ZoneTriggered` per due effect |
| FR-025 to FR-032 | unit | Table L one test per row (L7 same-spell dues carry keys, L13 different spells); ledger size bound; stale marker; round 0 |
| FR-033 to FR-039 | unit | Table M one test per row; `StoppedForTurn` cleared by the next marker (FR-043); zone moved into space is never an entry |
| FR-040 to FR-043 | unit | Each `ZoneEndReason`; ended twice; `ZoneRoundStarted` at round 9, 10 and 11 of a 10-round zone cast in round 1, host-timed and until-dispelled never end; budget at 59, 60, 61; no-budget refusal; negative amount |
| FR-044 to FR-048 | unit | Thorns plus Spike Growth costs 4 per foot, not 5; two zones both halving (one halving), Difficult Terrain not cumulative, `ExtraFeetPerFoot` maximum, exempt and ended zones, empty list |
| FR-049, FR-051 | architecture | Import-direction check of the zone files; scan that no spell id literal or per-spell branch exists in them |
| FR-050 | architecture, unit | Each `kind` constant unique in the package; `combatants` order |
| FR-052, FR-053 | unit | Inputs unchanged; equal results for equal inputs; each result classified as section 7 states |
| FR-054 | scenario | `scenarios/`, public API only: Spirit Guardians cast; C ends turn, X enters on C's turn then ends its own turn; `AssessZoneTrigger` returns two due effects; each goes to `ResolveSpellEffects` (RA-2) and `ApplyDamage`; final Hit Points asserted (deferred until RA-2 lands) |
| errors-and-results.md FR-031, FR-034, FR-051 | unit | n/a — the spec's operations call no port: there is no `dice.Roller` or `ContentSource` (the host reports events, moves and damage figures), so there is no port failure to propagate or record. |

- **Levels:** unit, architecture, one scenario. No contract suite (no port).
- **Mutation testing** (`doc/rules/testing.md` §8): changed lines of the zone files in `combat` (`gomu` run only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`)
  at score >= 80. Most valuable: marker equality (FR-026), the `/` in FR-019, the skip order
  (FR-014), `>=` in FR-041 and FR-042.
- **Not covered by unit tests:** FR-020's cause-free event (a type-level fact); FR-054's
  scenario waits for RA-2.

## 9. Open questions and decisions

Open: none (Q-12 below is settled as out of scope, with a default). Escalated: none. Package decisions where the SRD is silent (changing any is
**BREAKING**):

- Q-1: interpretation. "Turn" in "only once per turn" is one combatant's initiative turn, so a
  creature can be affected once on each combatant's turn (FR-026). The rival reading (once per
  creature's own turn) would make "whenever the Emanation enters a creature's space" (p. 164)
  almost never apply on the caster's turn; community consensus agrees with the former.
- Q-2: an optional ("you can force") application is not in the ledger until the host applies
  it (FR-023): a save never made cannot use up the turn.
- Q-3: forced movement counts for Spike Growth. The text says "moves into or within the area" and
  "travels" with no willingness qualifier (p. 164); rulings are mixed, so the engine takes the
  host's reported distance whatever the cause (FR-020).
- Q-4: interpretation. "For every 5 feet it travels" is whole units per reported segment,
  rounded down, no carry-over (FR-019), because the engine models no map and the SRD lets a
  move be broken up (p. 14).
- Q-5: interpretation. The ledger is per zone, not per caster or per spell id (FR-029): "this
  save" belongs to one cast. "Combining Spell Effects" (p. 106) says repeated castings of one
  spell do not combine (the most potent applies, the latest on a tie), which on its face
  reaches two overlapping Spirit Guardians; community reading is split (some limit it to
  buffs). The engine cannot compare potency without rolls, so it emits every due with
  `SpellID`, zone id and cast marker and the host applies p. 106; a host preferring the
  stacking reading just ignores the keys.
- Q-6: the zone's appearance counts as the turn's application (FR-030), since the text puts
  "when the cloud appears" and "only once per turn" in one effect (p. 142).
- Q-7: interpretation. Speed halved is a state, not a multiplier, so it does not stack (FR-046);
  "Changes to Your Speeds" (p. 188) halves special speeds with it.
- Q-8: interpretation. A `RoundsDuration` zone ends at the start of round `CastRound + rounds`
  (FR-041); the SRD gives no sub-round fencepost, and this errs short by part of a round.
- Q-9: the zone facts are content (`spell-effect-data.md`, RA-1), not constants in `combat`.
  `content` ranks below this spec so it cannot import combat types, and `NOTICE.md` covers
  only the data paths (`shared-types-and-ownership.md` FR-033).
- Q-10: Guardian of Faith's "dealt a total of 60 damage" is the host's figure (FR-042); the SRD
  does not say whether Resistance or half damage on a save changes what counts as dealt.
- Q-11: `environment-hazards-and-objects.md` is not a dependency: walls as objects and the
  Wall of Ice frigid air stay there and in the host's hands.

Amendments asked of other specs (none made here). **RA-1**
`spell-effect-data.md`: new FR-048 up: the zone facts types of section 5, an optional
`ZoneFacts` on `ModelledEffects`, decoder kinds, bundle checks and a pilot table from table Z;
its out-of-scope line on recurring effects points here. **RA-2** `spell-effects.md`: new
FR-098 up: `ResolveSpellEffectsCommand` may name `PartIndexes` and an instance count so a due
effect re-applies one part, and a recurring-only part is skipped at the cast. **RA-3**
`shared-types-and-ownership.md` 4.11: add `persistent-spell-zones` at rank 28.
**RA-4** `combat-action-resolution.md`: the use cases and telemetry of section 7.
**RA-5** `documentation_index.md`: one row for this spec; `roadmap.md` Phase 4 list.
**RA-6** `active-effects.md`: withdraw the Flaming Sphere pilot (FR-027 source cell, its
test row for FR-035 to FR-039, Q11 and the section 1 out-of-scope mention) and point to this
spec. Q-12 (settled: out of scope, default host-side, no FR; revisit only with a Tsunami spec): Tsunami's "only once per round" needs a ledger keyed
by round (a `LedgerWindow` of `PerTurn` / `PerRound`) and a multi-step drift; add with
Tsunami later, since a default exists (host-side) and no behaviour changes today.

## 10. Changelog impact

Expected `CHANGELOG.md` entry under `Unreleased`:

- **Added:** persistent spell zones in `combat`: `PersistentZone`, the once-per-turn ledger,
  trigger assessment, mobile zones, designated exemptions and movement effects
  (`doc/specs/persistent-spell-zones.md`), with `doc/observability/signals.md` rows once RA-4
  lands.
- **BREAKING:** no. Everything is new. After the first release, changing a decision in
  section 9 (Q-1 to Q-8, Q-12) is **BREAKING**.
