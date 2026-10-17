# Active effects: state that spans turns

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `observability-ports.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `content-spells.md`, `spell-effect-data.md`, `resolve-d20-test-use-case.md`, `spellcasting.md`, `casting-and-concentration.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md`, `initiative-and-actions.md`, `spell-effects.md` |

Rank 28 in table 4.11 of shared-types-and-ownership.md (see amendment RA-1 in section 9): after `spell-effects` (27) and beside `encounter-aggregate`, before `combat-action-resolution` (29). It does not depend on `encounter-aggregate.md` (equal rank, FR-056); it takes round and combatant as plain input (Q3). It does not depend on `spell-granted-modifiers.md` or `dispel-and-counterspell.md` either (both also rank 28): the host maps applying effects to `SpellModifierSource`s (FR-064, RA-7) and routes a dispel through `RemovedByEffect` (RA-6). `attacks.md` and `magical-effects.md` are prose mentions only (FR-056).

## 1. Purpose and scope

`spell-effects.md` resolves a spell once, at the cast, and hands durations on as data (its FR-040, Q11). `casting-and-concentration.md` owns the caster's Concentration state. `conditions.md` holds conditions with no duration. `magical-effects.md` holds a list of `OngoingEffect`s with no duration and no behaviour. Nothing owns "Hold Person lasts up to 1 minute, the target repeats the save at the end of each of its turns, and the spell ends if the caster loses Concentration". This spec is that owner: a **registry of active effects** and the pure rules that age it, trigger it and end it.

- **Registry.** `ActiveEffects` holds `ActiveEffect`s, each on a creature or on a place, with an origin, a caster, a duration, optional Concentration binding, conditions it holds, triggers and stored damage.
- **Expiry** from `Spell.duration` and `ConditionDuration` (spell-effect-data.md FR-046), in rounds, at turn boundaries, or by elapsed time outside combat.
- **Concentration.** An effect bound to a caster's Concentration ends when the host reports the Concentration ended (the character spec owns the state).
- **Repeated saves** at the end of a turn, **delayed and recurring damage**, **damage that grows** until the effect ends (Delayed Blast Fireball), **staged effects** (Sleep), and **moving** a mark or curse to a new creature (Hunter's Mark, Hex).
- **Combining Spell Effects** (p. 106) for lifetimes: different spells add, the same spell's castings do not combine and the most potent applies. What an applying effect adds to rolls (Bless, Bane, Hex, Hunter's Mark) is `spell-granted-modifiers.md`'s.

Four use cases (`StartActiveEffect`, `AdvanceActiveEffects`, `EndActiveEffects`, `ResolveEffectDamage`) and pure domain services. State in, new state and events out (`doc/rules/architecture.md` §4): the host stores the `ActiveEffects` value (Q2). Nothing here applies damage, conditions or Hit Point changes; results are plans for `ApplyDamage` and for `ConditionSet` use cases.

**Out of scope:**

- Casting, slots and the Concentration state and its break rules (`casting-and-concentration.md`); the Concentration saving throw (the host rolls it and reports a loss); applying damage (`hit-points-and-damage.md`) and conditions (`conditions.md`).
- Resolving what a spell does at the cast (`spell-effects.md`); this spec resolves only what happens later.
- Area membership, line of sight and range: the host reports who is in an area (as in `spell-effects.md`). Moving a movable effect (Flaming Sphere) is a host fact; a `PlaceEffectSubject` has no coordinates.
- Content data for ongoing effects. The `EffectTemplate` types are `combat` domain values built by hosts or by a later data spec (RA-2); `spell-effect-data.md` keeps E14 to E16 as text until then.
- `toolbox`'s `ActiveSaveGatedEffect` (toolbox-hazard-effects.md) stays as it is; `toolbox` is downstream and may not be imported here. Unifying the two is not required.
- Everything an active spell adds to rolls, Armor Class, Speed or damage (Bless, Bane, Hex, Hunter's Mark riders, Death Saving Throws, Critical Hit doubling): `spell-granted-modifiers.md`. Effects that change rolls made against the subject or lift its benefits (Shining Smite, p. 162) stay text. The ending "someone within 5 feet takes an action to shake it out of the spell's effect" (Sleep p. 163, Hypnotic Pattern p. 141) is a host fact reported as `RemovedByEffect` (FR-044). Summons, illusions, Hit Point maximum changes, reading minds, terrain, light, "Identifying an Ongoing Spell" (p. 106: a Study action and a DC 15 Intelligence (Arcana) check, GM-adjudicated here), "Awareness of Being Targeted".
- Spell Scrolls, monster traits, magic item charges (`magic-item-rules.md`), curses (`magical-effects.md` `RemoveCurse`).
- Anything outside SRD 5.2.1. The SRD gives no data format; structure is package design. Nothing core was found missing from the PDF.

**Ownership (shared-types-and-ownership.md FR-048).** This spec owns the lifetime, triggers, ending and moving of effects, and the one live registry of them. `spell-effects.md` owns the cast-time plan; `spell-granted-modifiers.md` owns what an active spell adds to rolls, stats and damage and its own combining key (the host feeds it from FR-064); `spellcasting.md` / `casting-and-concentration.md` own Concentration; `conditions.md` owns what a condition does; `dispel-and-counterspell.md` decides what a dispel ends and reports it (RA-6). `OngoingSpellRegistry` there is not a second lifetime store.

## 2. Source

SRD 5.2.1 (`doc/research/sources.md`). Printed page equals PDF page.

- "Duration" (Concentration, Instantaneous, Time Span, "you can dismiss it ... if you don't have the Incapacitated condition"), p. 106 (the "Time Span" entry begins p. 105 end); "Effects", "Targets", "Invalid Targets", "Saving Throws", "Combining Spell Effects", p. 106; "Using a Higher-Level Spell Slot", pp. 104-105; "Longer Casting Times", p. 105.
- "The Order of Combat" (a round is about 6 seconds), "Combat Step by Step", p. 13; "Damage and Healing", "Damage Rolls", "Half Damage", "Saving Throws and Damage" (one roll for all targets), p. 16; "Damage at 0 Hit Points", p. 18.
- Rules Glossary: "Concentration", "Condition" (a condition doesn't stack; "various rules define how to end a condition"), p. 179; "Critical Hit", p. 179; "Magical Effect", p. 185; "Simultaneous Effects", p. 187; "Area of Effect", p. 177; "Incapacitated", p. 183.
- Spell entries: Acid Arrow p. 107; Bane p. 112; Bless p. 113; Bestow Curse pp. 112-113; Flaming Sphere p. 132; Delayed Blast Fireball p. 122; Hex p. 140; Hold Person, Hunter's Mark and Hypnotic Pattern p. 141; Shining Smite p. 162; Sleep and Slow p. 163.
- Package rules: `doc/rules/architecture.md` §2-§6; `observability.md` §2-§7; `testing.md` §4, §5, §8. Binding: shared-types-and-ownership.md FR-011, FR-013, FR-015, FR-017, FR-019, FR-021, FR-041, FR-044, FR-048, FR-056, FR-062; combat-foundations.md FR-015 to FR-041.
- Research (informs judgement only, nothing copied): the usual table ruling that a one-minute duration is ten rounds and ends at the same Initiative count; see Q1.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| (effect instance) | `ActiveEffect` (immutable value) | Package term. `id` (`ActiveEffectID`), `origin`, `caster`, `subject`, `binding`, `stage`, `potency`, `storedDamage`, clock fields (FR-020). |
| (registry) | `ActiveEffects` | Ordered collection (copied on read), host-held (Q2). |
| (identity) | `ActiveEffectID` (typed value over `String`) | Host-supplied; the package never generates one (FR-001). |
| Spell, "magic item or phenomenon" | `EffectOrigin`: `SpellEffectOrigin(spellID, castAtLevel)`, `OtherEffectOrigin(key)` | p. 185 "Magical Effect". Only spell origins combine (FR-015). |
| target / "area of effect" | `EffectSubject`: `CreatureEffectSubject(CombatantID)`, `PlaceEffectSubject(EffectPlaceID)` | `EffectPlaceID` is an opaque host id (a sphere, a bead). |
| Duration | `EffectDuration`: `EffectLastsRounds(rounds)`, `EffectLastsUntilTurnBoundary(owner, boundary)`, `EffectLastsIndefinitely`, `EffectLastsRemainder` (stage only, FR-061) | Built from `SpellDuration` / `ConditionDuration` (FR-010, FR-011). `TurnBoundary` is spell-effect-data.md's. |
| "a round ... about 6 seconds" | `EffectDuration.roundsPerMinute` = 10 | FR-006 (interpretation). |
| Concentration | `ConcentrationBinding(caster, subject)` | `subject` is spellcasting.md's `ConcentrationSubject`; `Spell.requiresConcentration` (FR-012). |
| Combining Spell Effects, "most potent" | `EffectPotency` (`int`), `EffectCombination`, `EffectCombiningSubject` (`target`, `caster`) | FR-015 to FR-019, FR-064. |
| (shape of a spell's lasting effect) | `EffectTemplate(stages, potency, overall, combiningSubject, survivesSubjectDeath)`; `EffectStage(duration, conditions, triggers, endsOnDamage, gate, onEnd)` | FR-009, FR-061. |
| "repeats the save", "at the end of each of its turns" | `EffectTrigger`: `AtTurnBoundary(whose, boundary)`; `EffectTurnOwner` (`caster`, `subject`, `creatureInArea`) | FR-027 to FR-033. Not `TurnOwner` (spell-effect-data.md) nor `TurnHolder` (magical-effects.md). |
| (what a trigger does) | `EffectAction`: `RepeatSave(ability, dc)`, `DealDamage(amount, type, save, audience)`, `GrowStoredDamage(extra)`, `DealStoredDamage(type, save, audience)`; `EffectDamageAudience` (`subject`, `turnTaker`, `areaMembers`) | FR-029 to FR-039. |
| (clock input) | `EffectClockTick`: `TurnStartReached(combatant)`, `TurnEndReached(combatant)`, `RoundStartReached`, `TimeElapsed(rounds)`, `CombatEnded`, `CombatantLeft(combatant)` | FR-020 to FR-026, FR-062. |
| (why an effect ended) | `EffectEndReason`, `EndingCause` | FR-040 to FR-047. |
| Half Damage | `HalfDamage` (hit-points-and-damage.md) | Reused. |

## 4. Requirements

### 4.1 Model and registry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `ActiveEffectID` MUST be a named `string` type exported once; `NewActiveEffectID(string) (ActiveEffectID, error)` MUST return an error that is `errors.Is` `srderr.ErrInvalidInput` for a blank string (`strings.TrimSpace(value) == ""`) and keep the value unchanged otherwise (no trimming or case folding), and MUST NOT panic for any input; `MustActiveEffectID` panics with that error. The package MUST NOT generate ids (no `Random`, no uuid; as combat-foundations.md FR-009). | `doc/rules/architecture.md` §3, §6 | unhappy (invalid input) |
| FR-002 | `ActiveEffect` MUST be an immutable value equal by value over all fields, carrying `id`, `origin`, `caster` (`CombatantID`), `subject`, optional `binding`, `potency`, the `EffectTemplate` it was started from, the current stage index, the remaining clock (FR-020) and the stored damage (FR-037). Changing it returns a new instance. | `doc/rules/architecture.md` §3 | happy |
| FR-003 | `EffectOrigin` MUST be a closed variant set with exactly `SpellEffectOrigin(spellID, castAtLevel)` (`castAtLevel` 0 to 9, else the constructor returns a validation error) and `OtherEffectOrigin(key)` (non-blank key). | SRD 5.2.1, "Magical Effect", p. 185; "Using a Higher-Level Spell Slot", p. 104 | unhappy (boundary) |
| FR-004 | `EffectSubject` MUST be a closed variant set with exactly `CreatureEffectSubject(CombatantID)` and `PlaceEffectSubject(EffectPlaceID)`. A place subject MUST carry no coordinates, size or shape. | SRD 5.2.1, "Targets" ("a spell's description says whether the spell targets creatures, objects, or something else"), p. 106 | happy |
| FR-005 | `EffectDuration` MUST be a closed variant set with exactly `EffectLastsRounds(rounds)` (`rounds` at least 1, else the constructor returns a validation error), `EffectLastsUntilTurnBoundary(owner, boundary)` (`owner` a `CombatantID`) and `EffectLastsIndefinitely`, and `EffectLastsRemainder` (FR-061; valid only as the duration of a template stage, never as `overall`). | SRD 5.2.1, "Duration", p. 106; "Sleep" ("for the duration"), p. 163 | unhappy (boundary) |
| FR-006 | Converting a time span to rounds MUST use 1 round per round, 10 per minute, 600 per hour and 14,400 per day, from a round being about 6 seconds. This is an arithmetic **interpretation** (the SRD says "about"); an amount of 0 or less MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`). | SRD 5.2.1, "The Order of Combat", p. 13; "Duration" ("Time Span"), p. 106 | unhappy (boundary) |
| FR-007 | `ActiveEffects` MUST keep entries in the order started, expose a copied view, accept an empty list, and `NewActiveEffects` MUST return a validation error when two entries share an `ActiveEffectID`; the returned entries are a copy. Its transitions return a new value. | `doc/rules/testing.md` §4 (empty, immutability); `doc/rules/architecture.md` §3 | unhappy (empty and minimal) |
| FR-008 | `ActiveEffects` MUST offer pure queries `forCreature(CombatantID)`, `forPlace(EffectPlaceID)`, `castBy(CombatantID)` and `boundTo(ConcentrationBinding)`, each preserving registry order, none mutating. | `doc/rules/architecture.md` §3 | happy |
| FR-009 | `EffectTemplate` MUST hold one or more `EffectStage`s, a potency, an optional `overall` clock (FR-061), a `combiningSubject` (`target` default, FR-015) and a `survivesSubjectDeath` flag (default false, FR-043, FR-060). Only the last stage MAY have an `EffectLastsIndefinitely` or `EffectLastsRemainder` duration; a stage with a `gate` MUST NOT be the last; `EffectLastsRemainder` needs an `overall`; a `turnTaker` or `areaMembers` audience is valid only where FR-035/FR-038 allow it. `NewEffectTemplate` returns a validation error otherwise (every problem, `errors.Join`). A stage's `conditions` are `ConditionKind`s (no Exhaustion level), its `onEnd` is a list of actions, its `endsOnDamage` a `bool` (FR-033). | SRD 5.2.1, "Sleep", p. 163 (two stages) | unhappy (invalid input) |

### 4.2 Durations and Concentration binding

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | `EffectDurationFromSpell(SpellDuration) (EffectDuration, error)` MUST map per table D: `Timed` to `EffectLastsRounds` (FR-006), `UntilDispelled`, `UntilDispelledOrTriggered` and `Special` to `EffectLastsIndefinitely`; `Instantaneous` MUST return a validation error (an instantaneous spell has no lasting effect; delayed outcomes are a stage with `EffectLastsUntilTurnBoundary`, FR-024). | SRD 5.2.1, "Duration", p. 106; content-spells.md FR-050 | unhappy (boundary) |
| FR-011 | `EffectDurationFromCondition(ConditionDuration, spellDuration, caster, target) (EffectDuration, error)` MUST map per table D: `ForSpellDuration` to `EffectDurationFromSpell`; `TimedConditionDuration(amount, unit)` to rounds; `UntilNextTurnBoundary(owner, boundary)` to `EffectLastsUntilTurnBoundary` with `owner` the caster or target `CombatantID`. | SRD 5.2.1, "Hold Person", p. 141; "Ray of Sickness", p. 158; spell-effect-data.md FR-046 | happy |
| FR-012 | A spell whose `Spell.requiresConcentration` is true MUST start its effects with a `ConcentrationBinding(caster, subject)` whose subject is that spell; a non-Concentration spell, and any effect of `OtherEffectOrigin`, MUST have none. Starting with a mismatching binding MUST be rejected `bindingMismatch`. A `SpellDurationTimed` limit `concentrationUpTo` is the cap of the Concentration, not a second clock (FR-020). | SRD 5.2.1, "Concentration", p. 179; "Duration", p. 106; content-spells.md FR-057 | unhappy (rule interaction) |

Table D (duration mapping; one test per row).

| Label | Input | Result | Source | Path |
|---|---|---|---|---|
| D1 | Hold Person: `Timed(1, minutes, concentrationUpTo)` | `EffectLastsRounds(10)`, binding | p. 141 | happy |
| D2 | Hex at a level 3 slot: `Timed(8, hours, concentrationUpTo)` | `EffectLastsRounds(4800)`, binding | p. 140 | happy |
| D3 | Sleep: `ForSpellDuration` on a 1-minute Concentration spell | `overall` rounds 10; stage 2 `EffectLastsRemainder` (the rest of the spell clock, FR-061) | p. 163 | happy |
| D4 | Ray of Sickness: `UntilNextTurnBoundary(caster, end)` | until the end of the caster's turn | p. 158 | happy |
| D5 | `Timed(1, rounds, fixed)` | `EffectLastsRounds(1)` | p. 106 | unhappy (boundary) |
| D6 | `UntilDispelled` | indefinite, no binding | p. 106 | happy |
| D7 | `Instantaneous` | validation error | p. 106 | unhappy (invalid input) |
| D8 | Concentration spell, `OtherEffectOrigin` | no binding allowed | p. 185 | unhappy (rule interaction) |

### 4.3 Starting and Combining Spell Effects

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-013 | `StartActiveEffect` MUST add one `ActiveEffect` at the end of the registry, at stage 0, with the remaining clock set from the stage duration and the anchor set to the combatant active at that moment (none outside combat), and emit one `ActiveEffectStarted` (the `overall` clock, FR-061, starts with it). A Bane-like effect is started by the host only for targets that failed the cast-time save (SRD 5.2.1, "Bane", p. 112). The `conditions` list MUST hold only kinds the host actually applied (those it received as `ConditionApplied`, conditions.md FR-016); the package MUST NOT apply any. | SRD 5.2.1, "Duration", p. 106; "Condition", p. 179 | happy |
| FR-014 | `StartActiveEffect` MUST be rejected, registry unchanged and no events, with `duplicateID` (id already in the registry), `bindingMismatch` (FR-012), or `concentrationConflict` (the caster already has effects bound to a different Concentration subject: the host failed to report the earlier loss, FR-041). When several apply the order is `duplicateID`, `bindingMismatch`, `concentrationConflict`. Starting a second effect for the same caster, same subject and another target is allowed (Hold Person at a higher level). | `doc/rules/architecture.md` §6; SRD 5.2.1, "Concentration" ("Another Concentration Effect"), p. 179 | unhappy (state error) |
| FR-015 | Effects of the same spell on the same combining subject MUST NOT combine: among the `SpellEffectOrigin` effects with equal `spellID` and an equal combining subject (the template's `combiningSubject`: the effect's `subject` for `target`, its `caster` for `caster`; this spec defines the key, so two Rangers marking one creature give two applying effects, one per Ranger), exactly one **applies**, the one with the highest `EffectPotency`; on equal potency the one started last (later in registry order). Effects of `OtherEffectOrigin` never suppress or are suppressed. | SRD 5.2.1, "Combining Spell Effects", p. 106 | happy |
| FR-016 | Effects of different spells on the same subject MUST all apply ("add together while their durations overlap"); the registry MUST NOT remove one for another. | SRD 5.2.1, "Combining Spell Effects", p. 106 | happy |
| FR-017 | A suppressed effect MUST stay in the registry and age normally; when the applying effect ends, the suppressed one that is now most potent MUST apply again. Two castings by different casters (two Clerics, Bless) are two effects; two castings by the same caster of a Concentration spell cannot overlap (the first ends with the Concentration, FR-041). | SRD 5.2.1, "Combining Spell Effects" ("the effect continues until the duration of the second Bless ends"), p. 106 | unhappy (rule interaction) |
| FR-018 | `EffectCombination.Applying(effects, subject)` MUST be a pure query returning the applying effects of a subject in registry order, one per spell group; conditions held and triggers of a suppressed effect MUST NOT be read by any query or tick of this spec (FR-027, FR-029). | SRD 5.2.1, "Combining Spell Effects", p. 106 | happy |
| FR-019 | `EffectPotency` MUST be an `int` from the template; potency 0 is the default, so equal-potency castings resolve to the most recent. Which number is "more potent" (the highest bonus) is the template author's, since the SRD gives only the examples. | SRD 5.2.1, "Combining Spell Effects" ("the most potent effect—such as the highest bonus"), p. 106 | unhappy (rule interaction) |

Table C (Combining; one test per row; Bless with potency 0 unless stated).

| Label | Registry | Query on the target | Source | Path |
|---|---|---|---|---|
| C1 | Bless by Cleric A | A's Bless applies | p. 106 | happy |
| C2 | Bless by A, then Bless by B | only B's applies (equal potency, latest); A's stays, ages | p. 106 | unhappy (rule interaction) |
| C3 | as C2, then B's ends (host ended it) | A's applies again | p. 106 | unhappy (rule interaction) |
| C4 | Bless by A and Bane by B on one target | both apply | p. 106 | happy |
| C5 | a same-spell pair on one target, potency 2 (A), potency 1 (B) | A's applies although B is later | p. 106 | unhappy (boundary) |
| C6 | two `OtherEffectOrigin` with the same key | both apply | FR-015 | unhappy (rule interaction) |
| C7 | same spell, same caster, two different targets | both apply (different subjects) | p. 106 | happy |
| C8 | Hunter's Mark by A and by B on one target (`combiningSubject` `caster`) | both apply (one per caster) | p. 106; FR-015 | unhappy (rule interaction) |
| C9 | Hunter's Mark by A moved from X to Y (FR-060), B's on Y | both apply | p. 141 | unhappy (rule interaction) |

### 4.4 The clock

The tick a host passes mirrors the tracker events of initiative-and-actions.md (FR-028, FR-029, FR-135): `TrackerTurnStarted` gives `TurnStartReached(combatant)`, `TrackerTurnEnded` gives `TurnEndReached(combatant)`, `TrackerRoundStarted` gives `RoundStartReached`. `ClockTickFromTurnEvent(TurnEvent) (EffectClockTick, bool)` maps them; any other event returns false. No tick carries a round number: no requirement reads one (the tracker events already carry it). `CombatantLeft` is sent by the host when it removes a combatant (FR-062).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-020 | An `EffectLastsRounds(N)` effect with an anchor (the combatant active when it started) MUST lose one round each time `TurnStartReached` names its anchor in a later turn than the start, and MUST expire when the remaining rounds reach 0, at that start of the anchor's turn: N rounds later at the same Initiative count, so the anchor benefits on exactly N of its turns including the turn it started on. With no anchor the effect MUST lose one round at each `RoundStartReached`. Interpretation (Q1): the SRD gives the duration but no tracking rule; counting at the Initiative count is the usual ruling and uses only tracker facts. | SRD 5.2.1, "Duration" ("the spell ends after 1 minute has passed"), p. 106; "The Order of Combat", p. 13 | happy |
| FR-021 | `TimeElapsed(rounds)` (rounds at least 1, else the constructor returns a validation error) MUST subtract from the remaining rounds of every `EffectLastsRounds` effect, in or out of combat; an effect whose remaining rounds reach 0 or less MUST expire at that tick. Overshoot is not carried: a gate (FR-034) runs at most once per tick and leftover rounds do not carry into the next stage. Turn ticks MUST NOT be sent for time the host also reports as elapsed (the host chooses one clock per span of time). | SRD 5.2.1, "Duration", p. 106 | unhappy (boundary) |
| FR-022 | `CombatEnded` MUST clear every anchor and change no remaining rounds. A later combat then ticks anchorless effects by `RoundStartReached` (FR-020), so the effect keeps its remaining time. An `EffectLastsUntilTurnBoundary` effect is unchanged by it (a later turn of its owner still matches). | SRD 5.2.1, "Duration", p. 106 | unhappy (state error) |
| FR-023 | `EffectLastsIndefinitely` MUST never expire by any tick. | SRD 5.2.1, "Duration" (until dispelled), p. 106; content-spells.md FR-050 | happy |
| FR-024 | An `EffectLastsUntilTurnBoundary(owner, boundary)` effect MUST expire at the first `TurnStartReached` / `TurnEndReached` of `owner` that matches `boundary`, except that an effect started while `owner` is the active combatant with boundary `end` MUST skip that owner's current turn end ("until the end of your next turn"). With boundary `start` the current turn's start has passed. Started outside combat, the first matching tick counts. | SRD 5.2.1, "Ray of Sickness", p. 158; "Acid Arrow", p. 107; "Sleep", p. 163 | unhappy (rule interaction) |
| FR-025 | At one tick the order MUST be: (1) effects whose duration ends at this tick expire and run their stage `onEnd` actions (FR-038); (2) the triggers of the remaining effects fire (FR-027). An expired effect MUST NOT fire a trigger at the tick that expired it. | Package decision (section 9, Q4) | unhappy (rule interaction) |
| FR-026 | `AdvanceActiveEffects` MUST change nothing for a tick that matches no effect: the registry equal to the input, no events, an empty plan, result class `applied`. | `doc/rules/testing.md` §4 (empty and minimal) | unhappy (empty and minimal) |
| FR-061 | An `EffectTemplate` MAY carry an `overall` clock (`EffectLastsRounds` or `EffectLastsIndefinitely`) that starts with the effect and ticks per FR-020 to FR-023 whatever the current stage; when it reaches 0 the effect expires (reason `expired`, `onEnd` of the current stage runs) even mid-stage. A stage whose duration is `EffectLastsRemainder` has no clock of its own: it lasts as long as `overall`, so entering it MUST NOT set a fresh clock. Advancing a stage MUST NEVER extend the effect beyond `overall`. Stages with their own bound keep it. | SRD 5.2.1, "Sleep" (Unconscious "for the duration", the rest of the 1-minute spell), p. 163; "Duration", p. 106 | unhappy (boundary) |
| FR-062 | `CombatantLeft(combatant)` MUST (a) clear the anchor of every anchored effect whose anchor is that combatant (such an effect then ticks on `RoundStartReached`, keeping its remaining time, FR-020), and (b) for an `EffectLastsUntilTurnBoundary` effect whose `owner` is that combatant, whose boundary can no longer occur, replace the duration by `EffectLastsRounds(1)` with no anchor, so it ends at the next `RoundStartReached` (interpretation: the end of the round; the SRD does not say, and an immortal effect is the only unreasonable reading) and clear `skipsCurrentTurnEnd`. A combatant that anchors or owns nothing changes nothing. The package never infers a departure from a missing tick. | SRD 5.2.1, "Duration", p. 106; interpretation (package design: no effect outlives its clock) | unhappy (state error) |

Table T (clock; all effects `EffectLastsRounds(10)` unless stated; A, B are combatants, anchor shown).

| Label | Situation | Result | Source | Path |
|---|---|---|---|---|
| T1 | started on A's turn in round 1, anchor A | A's turn starts in rounds 2 to 10 leave 9 to 1; round 11 start: expires | p. 106 | happy |
| T2 | started by a Reaction on B's turn, anchor B | counts only B's turn starts | p. 13 | unhappy (rule interaction) |
| T3 | `EffectLastsRounds(1)`, anchor A, round 1 | expires at A's next turn start | p. 106 | unhappy (boundary) |
| T4 | `TimeElapsed(3)` | remaining 7 | p. 106 | happy |
| T5 | `TimeElapsed(10)`, `TimeElapsed(11)` | expires at each | p. 106 | unhappy (boundary) |
| T6 | started outside combat (no anchor), tick `TurnStartReached(A)` | unchanged | FR-020 | unhappy (state error) |
| T7 | no anchor in combat, `RoundStartReached` | remaining 9 | FR-020 | happy |
| T8 | `CombatEnded` after T1's round 5 | anchor cleared, remaining kept | FR-022 | unhappy (state error) |
| T9 | until A's turn end, started on A's turn | the current end skipped, expires at the next end | p. 158 | unhappy (rule interaction) |
| T10 | until target B's turn end, started on A's turn | expires at B's end this round | p. 107 | happy |
| T11 | indefinite, 100000 rounds elapsed | stays | p. 106 | unhappy (boundary) |
| T12 | anchor A (round 3, remaining 8), `CombatantLeft(A)`, then `RoundStartReached` | anchor cleared, remaining 8; then 7 | FR-062 | unhappy (state error) |
| T13 | until target B's turn end, `CombatantLeft(B)` | `EffectLastsRounds(1)`, no anchor; expires at the next `RoundStartReached` | FR-062 | unhappy (state error) |
| T14 | Acid Arrow (until B's turn end, `onEnd` `DealDamage` 2d4, audience `subject`) reaches B's turn end | expires, one `subject` plan naming B | p. 107; FR-025, FR-038 | happy |
| T15 | `TimeElapsed(11)` on a stage-1 clock of 10 with a gate | the gate runs once; stage 2 starts with its full remainder (or its `overall` remainder), no carry | FR-021, FR-034 | unhappy (boundary) |
| T16 | Sleep, `overall` 10, stage 1 ends with 8 overall rounds left | stage 2 has 8 left (`EffectLastsRemainder`), not 10 | p. 163; FR-061 | unhappy (boundary) |
| T17 | `CombatantLeft` for a combatant that anchors nothing | registry unchanged, no events | FR-062 | unhappy (empty and minimal) |

### 4.5 Triggers, repeated saves and staged effects

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-027 | A stage's `AtTurnBoundary(whose, boundary)` trigger MUST fire for an applying effect (FR-018) at a matching tick: `whose` `caster` matches the caster's `CombatantID`, `subject` a `CreatureEffectSubject`'s id, and `creatureInArea` the combatant of the tick when the host lists the effect id in `areaEffects` of the tick (area membership is the host's, as in spell-effects.md FR-015). A trigger's `DealDamage` audience is `subject` (the creature subject) or `turnTaker` (the combatant of the tick, for `creatureInArea`). There is no damage trigger: "ends when damaged" is the stage flag `endsOnDamage` alone (FR-033). | SRD 5.2.1, "Flaming Sphere" ("ends its turn within 5 feet"), p. 132; "Hold Person", p. 141 | happy |
| FR-028 | When several effects fire at one tick, `AdvanceActiveEffects` MUST process them in the command's `order` (a permutation of the ids that fire, the host's result of the "Simultaneous Effects" choice by the person whose turn it is) or, with no `order`, in registry order. An `order` that omits, repeats or names an id that does not fire MUST be rejected `invalidOrder`, listing every problem, class `invalid`; it is checked before `missingSaveAttempt` (FR-063). | SRD 5.2.1, "Simultaneous Effects", p. 187 | unhappy (invalid input) |
| FR-029 | A `RepeatSave(ability, dc)` action MUST make the subject's saving throw of `ability` against `dc` through `d20.Resolver` from the host-supplied `SaveAttempt` for that creature and ability (resolve-d20-test-use-case.md; the routine follows spell-effects.md FR-024). On success the effect MUST end (reason `repeatSaveSucceeded`); on failure it continues unchanged. One `ActiveEffectRepeatSaveMade` per save, carrying the outcome. | SRD 5.2.1, "Hold Person" ("repeats the save, ending the spell on itself on a success"), p. 141; "Slow", p. 163 | happy |
| FR-030 | Interpretation: the DC of a `RepeatSave` MUST be fixed in the template when the effect starts (the caster's spell save DC at the cast, spell-effects.md input); a later change to the caster's modifiers MUST NOT change it. The SRD says only that the target "repeats the save" and defines the save DC when the spell is cast, so the cast-time DC is the reasonable reading and needs no caster state. | SRD 5.2.1, "Saving Throws" (spell save DC), p. 106; "Hold Person", p. 141 | unhappy (state error) |
| FR-031 | A `ChooseToFail` or `AutomaticFailure` attempt (resolve-d20-test-use-case.md) MUST count as a failed repeat save without a d20; a creature MAY choose to fail ("you can decide to fail the save without rolling"). A missing attempt for a firing `RepeatSave` MUST be rejected `missingSaveAttempt` listing every missing pair, before any die is rolled. | SRD 5.2.1, "Saving Throw", p. 187 | unhappy (invalid input) |
| FR-032 | A `RepeatSave` fired for a creature MUST be made whatever its conditions, the effect itself being what holds it (Paralyzed, Incapacitated): the package adds no rule. Automatic failure of Strength and Dexterity saves is `conditions.md`'s and arrives inside the `SaveAttempt`. | SRD 5.2.1, "Paralyzed", p. 186; conditions.md | unhappy (rule interaction) |
| FR-033 | A stage with `endsOnDamage` MUST end its effect (reason `subjectTookDamage`) when `EndActiveEffects` receives `subjectTookDamage(subject, total)` with `total` at least 1 (the `DamageTaken.total`, hit-points-and-damage.md FR-102, absorbed damage included) for a creature subject; `total` of 0 or less MUST do nothing. Effects without the flag MUST be unaffected. | SRD 5.2.1, "Sleep" ("ends on a target if it takes damage"), p. 163; "Hypnotic Pattern", p. 141 | unhappy (boundary) |
| FR-034 | A stage `gate` (`RepeatSave`) MUST run when the stage's duration ends: success ends the effect (`repeatSaveSucceeded`), failure advances to the next stage. A gate runs at most once per tick; leftover time does not carry (FR-021). Advancing MUST emit `ActiveEffectStageAdvanced`, set the stage clock from the new stage (FR-020 anchored at the tick's active combatant; none for `EffectLastsRemainder`, which continues the `overall` clock) and never extend the total past `overall` (FR-061), and list the new stage's conditions to add and the old stage's to remove as `conditionsToApply` and `conditionsToRemove` in the plan. A stage without a gate advances on expiry; the last stage's expiry ends the effect (`expired`). | SRD 5.2.1, "Sleep" (Incapacitated until the end of its next turn, then a save, then Unconscious "for the duration"), p. 163 | unhappy (rule interaction) |

### 4.6 Damage over time and delayed damage

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-035 | A `DealDamage(amount, type, save, audience)` action (a `dice.Expression`, a `DamageType`, an optional `EffectDamageSave(ability, dc, onSuccess: none or half)`, and an `EffectDamageAudience`: `subject` = the effect's creature subject, no host list needed; `turnTaker` = the combatant of the tick, valid only in a trigger; `areaMembers` = the creatures the host lists, valid only in `onEnd`) MUST add one `EffectDamagePlan` to the result naming the effect, its audience and the action; a `subject` audience on a place subject adds no plan; nothing is rolled or applied by `AdvanceActiveEffects`. | SRD 5.2.1, "Acid Arrow" (2d4 "at the end of its next turn"), p. 107; "Flaming Sphere", p. 132 | happy |
| FR-036 | `ResolveEffectDamage` MUST roll the plan's damage once for all of its targets, build one `Damage` of the plan's type per target, and mark `HalfDamage` for a target that succeeded on a save whose `onSuccess` is `half`, `none` (no damage) when `onSuccess` is `none` and the save succeeded; targets whose save failed or with no save take full damage. Rounding is `ApplyDamage`'s (hit-points-and-damage.md). | SRD 5.2.1, "Saving Throws and Damage" (rolled once for all targets), "Half Damage", p. 16 | happy |
| FR-037 | `GrowStoredDamage(extra)` MUST add `extra` dice to the effect's `storedDamage` (a `dice.Expression`; the base is set by the template, 12d6 for Delayed Blast Fireball) and emit `ActiveEffectStoredDamageGrew`; growth on a suppressed or ended effect MUST NOT occur. Dice of the same die size add by count; a different die size is a validation error in the template. | SRD 5.2.1, "Delayed Blast Fireball" ("increases by 1d6 whenever your turn ends"), p. 122 | unhappy (boundary) |
| FR-038 | A stage's `onEnd` actions MUST run once when the effect ends for any `EffectEndReason` (expiry, dismissal, Concentration lost, removal), each `DealDamage` / `DealStoredDamage` adding an `EffectDamagePlan` carrying the action's audience, `subject` (Acid Arrow's single target at its turn end, T14; expiry comes before triggers, FR-025) or `areaMembers` (Delayed Blast Fireball; the creatures the host lists in `ResolveEffectDamage`, area membership being the host's). `DealStoredDamage` plans the effect's current `storedDamage`. A subject that died (reason `subjectDied`) MUST NOT run `onEnd`. `turnTaker` is never valid in `onEnd` (FR-009). | SRD 5.2.1, "Delayed Blast Fireball" ("When the spell ends, the bead explodes"), p. 122; "Acid Arrow", p. 107 | unhappy (rule interaction) |
| FR-039 | `ResolveEffectDamage` MUST reject, before any die, an empty target list (`noTargets`, class `refused`: no creature takes damage, as spell-effects.md Q17), a target listed twice (`duplicateTarget`) and a plan target without a needed `SaveAttempt` (`missingSaveAttempt`, FR-031), checked in that order (FR-063). | `doc/rules/architecture.md` §6; spell-effects.md FR-020 | unhappy (invalid input) |

### 4.7 Ending effects

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | `EndingCause` MUST be a closed variant set with exactly `ConcentrationLost(caster, subject)`, `Dismissed(caster, casterIncapacitated, effectIDs)`, `SubjectTookDamage(subject, total)`, `SubjectDied(subject)`, `RemovedByEffect(effectIDs)` and `MoveEffectSubject(effectID, newSubject)` (FR-060: a re-pointing, not an end). `EffectEndReason` MUST be `expired`, `repeatSaveSucceeded`, `concentrationLost`, `dismissed`, `subjectTookDamage`, `subjectDied`, `removedByEffect`. | SRD 5.2.1, "Concentration", "Condition", p. 179; "Duration", p. 106 | happy |
| FR-041 | `ConcentrationLost(caster, subject)` MUST end every effect bound to that caster and subject, in registry order, one `ActiveEffectEnded` each (reason `concentrationLost`), with the stage `onEnd` actions of FR-038. The host sends it when the character spec emits `ConcentrationEnded` (casting-and-concentration.md FR-076, FR-077, FR-099, FR-119, FR-120, FR-141); the cause includes ending it by choice, Incapacitated, death, a failed save and another Concentration effect. No match returns `EndActiveEffectsNothingEnded`. | SRD 5.2.1, "Concentration", p. 179 | happy |
| FR-042 | `Dismissed(caster, casterIncapacitated, effectIDs)` MUST end the named effects. It MUST be rejected `casterIncapacitated` (class `refused`) when the flag is true, before any other check. Each named effect MUST then be checked in this order and every problem listed, nothing ended: `unknownEffect`, `notTheCaster` (the effect's caster differs), `usesConcentration` (it has a binding; it ends with the Concentration, FR-041), `notTimeSpan` (not an `EffectLastsRounds` effect). Classes per FR-058 and the mixed rule of FR-063. Dismissal costs no action. | SRD 5.2.1, "Duration" ("Time Span": "you can dismiss it (no action required) if you don't have the Incapacitated condition"), p. 106 | unhappy (state error) |
| FR-043 | Interpretation: `SubjectDied(subject)` MUST end every effect on that creature subject (reason `subjectDied`) whose template does not set `survivesSubjectDeath` (FR-060), but MUST NOT end an effect the dead creature cast unless it was bound to its Concentration (which the Dead state ends, FR-041): a non-Concentration spell persists. A place subject is unaffected. The SRD says only that Concentration ends when the caster is Dead (p. 179); that a spell ends on a dead subject is the package's reading: a condition or mark on a creature has nothing left to act on, and a revived creature (returning-to-life.md) starts with a clean registry entry, never regaining an ended effect (Q14). | SRD 5.2.1, "Concentration" ("Incapacitated or Dead"), p. 179; "Duration", p. 106; "Hunter's Mark", p. 141 | unhappy (rule interaction) |
| FR-044 | `RemovedByEffect(effectIDs)` MUST end exactly the named effects (reason `removedByEffect`), for Dispel Magic (`dispel-and-counterspell.md`, RA-6), a shift-shape exception (magical-effects.md FR-048), a touched bead, someone's action to shake a creature out of Sleep or Hypnotic Pattern (pp. 163, 141), or any host-decided end. A named id not in the registry MUST be rejected `unknownEffect` listing every unknown id, nothing ended. | SRD 5.2.1, "Shape-Shifting", p. 187; "Delayed Blast Fireball", p. 122 | unhappy (invalid input) |
| FR-045 | Every ended effect MUST add its held condition kinds to `conditionsToRemove(target, kind)` in the result, for the host to pass to `RemoveConditionFromSet`, except that a removal MUST be omitted when another effect remaining in the same registry on the same creature still holds that kind (applying, or suppressed but aging; two Hold Persons, or Hold Person and Sleep stage 2 both giving Incapacitated). Ending the last holder plans the removal. The package cannot know sources outside the registry (a condition "doesn't stack with itself"), so for those the host decides per item. | SRD 5.2.1, "Condition", p. 179; conditions.md FR-031 | unhappy (rule interaction) |
| FR-060 | `MoveEffectSubject(effectID, newSubject)` MUST re-point the named effect to `newSubject`, keeping its remaining rounds, `overall`, binding, potency, stage, anchor and stored damage, and emit `ActiveEffectSubjectMoved`; no `onEnd` runs and no condition plan is made. A template with `survivesSubjectDeath` (Hunter's Mark, Hex) keeps its effect when `SubjectDied` names its subject (FR-043), so the caster can then move it ("If the target drops to 0 Hit Points before this spell ends, you can take a Bonus Action to move the mark", p. 141; Hex "on a later turn", p. 140; the action cost is the host's). Rejections in this order: `unknownEffect`, `notMovable` (template flag false), `newSubjectNotCreature` (a place), `sameSubject`. The Concentration binding is unchanged. | SRD 5.2.1, "Hunter's Mark", p. 141; "Hex", p. 140 | unhappy (state error) |
| FR-046 | `EndActiveEffects` and `AdvanceActiveEffects` MUST remove ended effects from the returned registry and keep all others in order; ending an already-removed id twice in one command MUST be harmless (one end). | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-047 | The suppressed-or-applying state (FR-015) MUST be recomputed after every end; an ended applying effect MUST make the next most potent of its group apply again (C3), and no event is emitted for that change (it is derived). | SRD 5.2.1, "Combining Spell Effects", p. 106 | unhappy (rule interaction) |

### 4.8 Hand-off to spell-granted modifiers

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-064 | `ActiveEffect` MUST expose `spellID` (for a `SpellEffectOrigin`), `caster`, `subject`, `castAtLevel` and the effect's position in registry order, and `EffectCombination.Applying(effects, subject)` (FR-018) MUST return only applying effects, so that a host can build one `SpellModifierSource` per applying effect (`caster`, `target`, a `castOrder` from the registry position, which is distinct per effect and increases with start order). This spec defines no rider, no dice and no roll contribution, and never imports `spell-granted-modifiers.md`; Death Saving Throws, Critical Hit doubling and the Bane/Bless/Hex/Hunter's Mark figures are decided there (its live rider requirements). A suppressed or ended effect MUST NOT appear in `applying`. | SRD 5.2.1, "Combining Spell Effects", p. 106; shared-types-and-ownership.md FR-048 | happy |

### 4.9 Purity, events and use-case conventions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-053 | Everything in sections 4.1 to 4.8 outside the four use cases MUST be domain code in package `combat` (files not named `*_usecase.go`), pure, synchronous, silent (no port, no `telemetry.Telemetry`), and deterministic: equal input gives equal output. | `doc/rules/architecture.md` §3; `doc/rules/observability.md` §3 | happy |
| FR-054 | Events MUST be the closed variant family `ActiveEffectEvent` (an interface with an unexported marker method, embedding `combat.Event`, combat-foundations.md FR-015 to FR-019; its doc comment lists every variant): `ActiveEffectStarted`, `ActiveEffectEnded(reason)`, `ActiveEffectRepeatSaveMade(outcome)`, `ActiveEffectStageAdvanced`, `ActiveEffectStoredDamageGrew`, `ActiveEffectSubjectMoved`. `kind` is the snake_case class name; `combatants` is the subject creature then the caster, de-duplicated, for a place subject the caster only; `ActiveEffectSubjectMoved` lists the new subject, the old, then the caster. | combat-foundations.md FR-015, FR-017, FR-018, FR-020; shared-types-and-ownership.md FR-019 | happy |
| FR-055 | Each use case MUST take a `context.Context` and a `<UseCase>Command` and return a `<UseCase>Result` (a closed variant set: an interface with an unexported marker method, switched exhaustively) and an `error` (port failures only) with operation-prefixed variants (`StartActiveEffectStarted`, `StartActiveEffectRejected(StartActiveEffectRejection)`, `AdvanceActiveEffectsApplied`, `AdvanceActiveEffectsRejected`, `EndActiveEffectsEnded`, `EndActiveEffectsMoved`, `EndActiveEffectsNothingEnded`, `EndActiveEffectsRejected`, `ResolveEffectDamageResolved`, `ResolveEffectDamageRejected`). No generic variant name is exported. Rejections hold reasons in value objects named `...Rejection`. | combat-foundations.md FR-022, FR-023; shared-types-and-ownership.md FR-017 | unhappy (name collision) |
| FR-056 | No use case MUST call another use case; `AdvanceActiveEffects` and `ResolveEffectDamage` compose `d20.Resolver` and pure domain services and use the `dice.Roller` only. A dice port failure MUST be returned as the `error`, a `*srderr.PortError` wrapped once with `srderr.WrapPort`, with no partial result (`errors.Is` reaches the roller's cause). | shared-types-and-ownership.md FR-044 | unhappy (collaborator failure) |
| FR-057 | A result MUST hold its registry, a `[]ActiveEffectEvent` in the order they happened, and the plan lists (`conditionsToApply`, `conditionsToRemove`, `damagePlans`), all returned as copies from accessors (errors-and-results.md FR-021). Inputs MUST NOT be mutated. | `doc/rules/architecture.md` §3, §4; combat-foundations.md FR-021 | unhappy (immutability) |
| FR-058 | Result classes per combat-foundations.md FR-031: `applied` for started, applied, ended, moved and resolved; `refused` for `duplicateID`, `concentrationConflict`, `casterIncapacitated`, `usesConcentration`, `notTheCaster`, `notTimeSpan`, `noTargets`, `notMovable`, `sameSubject`, `NothingEnded`; `invalid` for `bindingMismatch`, `invalidOrder`, `missingSaveAttempt`, `unknownEffect`, `duplicateTarget`, `newSubjectNotCreature`. Mixed rejections: FR-063. | combat-foundations.md FR-031 | unhappy (invalid input) |
| FR-059 | Every type named by a command or result MUST be exported from package `combat` with a doc comment starting with its name; domain service helpers other than those named here stay internal. | `doc/rules/architecture.md` §4, §7 | happy |
| FR-063 | A rejection MAY hold problems of several reasons. Its result class MUST be `invalid` if any listed problem is `invalid`, else `refused` (FR-058). Checks run in a fixed order, the first failing group listing all its problems: `AdvanceActiveEffects`: `invalidOrder`, then `missingSaveAttempt`; `ResolveEffectDamage`: `noTargets`, `duplicateTarget`, `missingSaveAttempt`; `Dismissed`: FR-042; `MoveEffectSubject`: FR-060. | combat-foundations.md FR-031; `doc/rules/architecture.md` §6 | unhappy (invalid input) |

### 4.10 Withdrawn ids

Ids are never reused or renumbered. Rider behaviour (Bless and Bane dice, Hex check Disadvantage, Hunter's Mark and Hex extra damage) is owned by `spell-granted-modifiers.md` (section 1, FR-064); only the moved-id rows below name an id there, as pointers; no live requirement here cites one.

| Id | Requirement |
|---|---|
| FR-048 | Withdrawn: moved to spell-granted-modifiers.md (FR-008); the `EffectRider` family and `RiderBeneficiary` are not modelled here. |
| FR-049 | Withdrawn: moved to spell-granted-modifiers.md (FR-061); the extra damage query is there and the Critical Hit doubling is `attacks.md`'s. |
| FR-050 | Withdrawn: moved to spell-granted-modifiers.md (FR-023); the roll contributions query, Death Saving Throws included, is there. |
| FR-051 | Withdrawn: a place makes no rolls, so the host builds no source and there is no rider on a place. |
| FR-052 | Withdrawn: folded into FR-013 (Bane only for failed targets). |

## 5. Model

**Domain** (package `combat`, file `active_effects.go` and neighbours, one file per concept): `ActiveEffectID` with `NewActiveEffectID` / `MustActiveEffectID`; `ActiveEffect`, `ActiveEffects`; `EffectOrigin`, `EffectSubject`, `EffectPlaceID`, `EffectDuration`, `ConcentrationBinding`, `EffectPotency`; `EffectTemplate`, `EffectStage`, `EffectTrigger`, `EffectTurnOwner`, `EffectAction`, `EffectDamageSave`, `EffectDamageAudience`, `EffectCombiningSubject`; `EffectClockTick`; `EffectCombination` (stateless service); `ActiveEffectEvent` family; `EffectEndReason`, `EndingCause`. All values are immutable (unexported fields, value receivers).

- **Invariants** (constructors return validation errors wrapping `srderr.ErrInvalidInput`; `MustX` panics): FR-003, FR-005, FR-006, FR-007, FR-009, FR-037, FR-061. State-dependent outcomes are results (FR-058).
- **The clock fields** of an `ActiveEffect` are `remainingRounds` (`int`, the stage clock, for `EffectLastsRounds`), `overallRemaining` (optional `int`, FR-061), `anchor` (optional `CombatantID`) and `skipsCurrentTurnEnd` (`bool`, FR-024). No timestamps (no clock in the domain).
- **A transition** is a pure function such as `advance(effects ActiveEffects, tick EffectClockTick, ...)` returning `(ActiveEffects, []ActiveEffectEvent, plans)`; the use cases add dice, nothing else (`doc/rules/architecture.md` §4).

**Use cases** (package `combat`, files `*_usecase.go`): `StartActiveEffect`, `AdvanceActiveEffects`, `EndActiveEffects`, `ResolveEffectDamage` with their commands, results and rejections. The host's inputs are plain values (`SaveAttempt` maps, area lists), never combat aggregates.

**Reused, not defined here:** `CombatantID`, `CombatEvent`, the combat signal-name constants (combat-foundations.md); `Damage`, `HalfDamage`, `DamageTaken` (hit-points-and-damage.md); `Condition`, `ConditionApplied`, `ConditionSet` use cases (conditions.md); `TurnEvent` (initiative-and-actions.md); `ConditionToApply`, `PlannedDamage` (spell-effects.md; not reused for effect damage); `ConditionDuration`, `TurnBoundary`, `SpellAmount` (spell-effect-data.md); `Spell`, `SpellDuration`, `DurationUnit` (content); `ConcentrationSubject` (spellcasting.md); `d20.Resolver`, `DifficultyClass`, `SaveAttempt`, `ChooseToFail`, `AutomaticFailure` (d20 specs); `dice.Expression`, `dice.Roller` (dice-and-rolls.md).

## 6. Interface

Synchronous calls; `context.Context` is the first parameter because the use cases emit spans (go.md §4); `dice.Roller` itself takes none. Illustrative; requirements fix behaviour.

```go
// Roller is the single method AdvanceActiveEffects needs (consumer-side
// interface; any dice.Roller satisfies it).
type Roller interface {
    Roll(dice.Expression) (dice.Result, error)
}

type AdvanceActiveEffects struct { /* unexported: roller, telemetry */ }

func NewAdvanceActiveEffects(roller Roller, opts ...Option) (AdvanceActiveEffects, error)

// Run returns a validation error for a nil roller at construction, a
// *srderr.PortError for a roller failure, ctx.Err() on cancellation; every
// expected rule outcome (including rejections) is the returned result.
func (u AdvanceActiveEffects) Run(ctx context.Context, cmd AdvanceActiveEffectsCommand) (AdvanceActiveEffectsResult, error)
```

- `StartActiveEffectCommand`: the `ActiveEffects`, the new `ActiveEffectID`, `EffectOrigin`, caster, `EffectSubject`, `EffectTemplate`, optional `ConcentrationBinding`, the conditions the host applied, the active combatant (or none), the `overall` clock from the template.
- `AdvanceActiveEffectsCommand`: the `ActiveEffects`, an `EffectClockTick`, `areaEffects`, an optional `order`, `map[CombatantID]map[ability.Ability]d20.SaveAttempt`, the active combatant after the tick (for stage anchors, FR-034). Result `AdvanceActiveEffectsApplied(effects, events, conditionsToApply, conditionsToRemove, damagePlans)`.
- `EndActiveEffectsCommand`: the `ActiveEffects`, an `EndingCause` (including `MoveEffectSubject`). Results `EndActiveEffectsEnded(effects, events, conditionsToRemove, damagePlans)` and `EndActiveEffectsMoved(effects, events)`.
- `ResolveEffectDamageCommand`: one `EffectDamagePlan`, ordered targets (`CombatantID`, optional `SaveAttempt`). Result `ResolveEffectDamageResolved(assignments)`, each `EffectDamageAssignment(target, damage, adjustment, save)` for `ApplyDamage`.
- **Ports:** only `dice.Roller` (through the consumer-side `Roller`) and the optional `telemetry.Telemetry` bundle (option `WithTelemetry`, default `telemetry.Noop`); no new port, double or contract suite. `content.Source` is not called; the host passes the `Spell` facts it already has.
- **Exports:** all types of sections 3 and 5 named by a command or result, from package `combat`; `EffectCombination` as a published service; signal-name constants beside the use cases. No facade or re-export.
- **Position.** Depends only on specs of lower rank (FR-056); nothing earlier changes.

## 7. Observability

`domain/` is silent. The four use cases record through the `telemetry` ports (combat-foundations.md FR-035): span `srd5e.combat.<operation>` for `start_active_effect`, `advance_active_effects`, `end_active_effects`, `resolve_effect_damage`; shared counters and histogram; new `srd5e.combat.result` values `started`, `advanced`, `ended`, `moved`, `nothing_ended`, `effect_damage_resolved`, `rejected_invalid`, `rejected_refused` with their `CombatResultClass`es (FR-058).

1. **What can go wrong?** A `dice.Roller` failure; a host that does not report a Concentration loss, a turn tick or damage, so effects live too long (stale registry); a missing save attempt; a template that disagrees with its spell text; effects piling up.
2. **How would an operator know?** `srd5e.combat.operations` by operation and `result_class`; a rising `refused` share of `concentrationConflict` shows unreported losses; the `active` histogram p99 shows build-up.
3. **Diagnosis:** operation, result name and class, end reason counts, the events' kinds. No ids, spell ids or amounts are attributes (combat-foundations.md FR-033).
4. **Counted or timed:** calls and duration (shared), effects ended by reason, repeat saves by outcome and ability, effects in the registry after a call.
5. **Existing instrumentation still correct?** Yes. No other use case runs (FR-056), so `srd5e.d20_test.*` and `resolve_attack` do not count saves made here; spell attack and save signals of `spell-effects.md` are unaffected.
6. **Monitor or alert:** any error rate; `refused` share above a few percent; `active` p99 above 3 times the 7-day baseline; the `expired` share against `concentration_lost` (a drop in either hints at a host not sending a tick).
7. **Noise?** No logs: rejections are expected outcomes (observability.md §7; combat-foundations.md FR-034).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.start_active_effect`, `srd5e.combat.advance_active_effects`, `srd5e.combat.end_active_effects`, `srd5e.combat.resolve_effect_damage` | span | `srd5e.combat.active_effects.rejection` (rejected only; closed set of FR-058 reasons) | Path of one call | Any error status |
| `srd5e.combat.active_effects.ended` | counter, `{effect}` | `srd5e.combat.active_effects.end_reason` (the seven values of FR-040) | Why effects end | Share of `concentration_lost` and `expired` per day against baseline |
| `srd5e.combat.active_effects.saves` | counter, `{save}` | `srd5e.outcome` (`success`, `failure`; a forced failure is `failure`), `srd5e.ability` (the six, the `d20` constant), `srd5e.combat.active_effects.save_kind` (`repeat`, `stage_gate`, `damage`) | Aggregate outcomes of repeated saves | Success share more than 10 points from baseline over 1000 saves per ability |
| `srd5e.combat.active_effects.active` | histogram, `{effect}` | `srd5e.combat.operation` | Effects in the registry after the call | p99 above 3 times the 7-day baseline |

Attribute keys other than the shared `srd5e.outcome` and `error.type` keys of `telemetry` are constants in package `combat`; renaming any is **BREAKING**. Entries feed `doc/observability/signals.md`.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-009 | unit | Blank, empty, `' a'` ids; `NewActiveEffectID` never panics; equality; origin level 10 and -1; rounds 0; conversion rows; duplicate ids; query order; template with a gated last stage, an early indefinite stage |
| FR-010 to FR-012 | unit | Table D rows D1 to D8; binding mismatch |
| FR-013, FR-014 | application | Start, anchor set, events; each rejection and their fixed order; recording roller sees no die |
| FR-015 to FR-019 | unit | Table C rows C1 to C9 through `EffectCombination`; potency tie to latest; no mutation of the registry |
| FR-020 to FR-026, FR-061, FR-062 | unit and application | Table T rows T1 to T17; boundary values 9, 10, 11 rounds; overall 8 left gives stage 2 8 not 10 (T16); overshoot with a gate (T15); `CombatantLeft` for anchor, owner and bystander; empty tick; `fromTurnEvent` for the three events and for others |
| FR-027, FR-028, FR-063 | unit and application | Each `EffectTurnOwner`; registry-order default; `order` permutation, missing, repeated, extra id; `invalidOrder` together with `missingSaveAttempt` lists only `invalidOrder`; mixed rejection class (invalid wins) |
| FR-029 to FR-034 | application | Real `d20.Resolver` with `dicetest.FixedRoller`: Hold Person save pass and fail; fixed DC after caster change; `ChooseToFail`; missing attempt lists all and rolls nothing; `endsOnDamage` at 0, 1 and absorbed damage; Sleep two-stage with gate success and failure and the condition lists; no `WhenSubjectDamaged` trigger exists |
| FR-035 to FR-039 | application | Acid Arrow end-of-turn plan with a `subject` audience (T14), place subject adds no `subject` plan; `ResolveEffectDamage` with no targets, duplicate and missing save together (order); Flaming Sphere in-area plan; one roll for several targets; half and none on success; Delayed Blast Fireball growth 12d6 to 15d6 over three caster turn ends, detonation on duration end, on Concentration loss, `subjectDied` skipping `onEnd`; rejections |
| FR-040 to FR-047 | application | Concentration lost ends all bound (Hold Person on two targets); recast mismatch; dismiss allowed, incapacitated, concentration and wrong caster; death of subject and of caster (persists unless bound); unknown id lists all; conditions lists; recompute after end (C3) |
| FR-045, FR-060, FR-064 | application | Two holders of one condition: ending one omits the removal, ending the second plans it; `MoveEffectSubject` keeps rounds, binding, potency, stage and stored damage, `SubjectDied` skips a `survivesSubjectDeath` effect, then moved; `unknownEffect`, `notMovable`, place target, same subject; `applying` excludes suppressed effects and gives distinct registry positions |
| FR-053 to FR-059 | architecture, application | Import rules (check A1: `combat` imports no `toolbox` and no downstream package); package API scan for names and the banned `Due`/`Required` suffixes; immutability (mutating a returned slice leaves the result unchanged); no use case mentioned in the use case files; a dice port failure (`*srderr.PortError`) propagates; classes of FR-058 |
| telemetry.Telemetry (section 7) | application | Recording `telemetry.Telemetry`: shared signals once per call, `ended` by reason, `saves` by kind, `active` after; no `srd5e.d20_test.*`; identical result with the no-op; no log |
| errors-and-results.md FR-031, FR-034, FR-051 | application | The roller failure is returned as a `*srderr.PortError` wrapped once (`errors.Is` reaches the cause) with no partial result (FR-031); a recorded failure sets `error.type` from `srderr.ErrorType(err)`; a panicking telemetry double propagates and `span.End` still runs (FR-051). |

- **Levels:** unit (`combat/*_test.go`, external package), use case (`combat/*_usecase_test.go`), architecture, one scenario. No contract suite.
- **Doubles:** `dicetest.FixedRoller` and `dicetest.CallbackRoller (returning an error)`, the `telemetrytest` recorder, the real `d20.Resolver`; no mocks of the domain.
- **Scenario (`scenarios/`, package `scenarios_test`, public API only):** a Cleric casts Bless on two allies (`StartActiveEffect`), a second Cleric casts it on one; the combined query shows one d4; a Wizard casts Hold Person; the target fails, repeats the save at its turn end and succeeds; the registry loses the effect and `conditionsToRemove` goes to `RemoveConditionFromSet`.
- **Mutation testing (`doc/rules/testing.md` §8):** `gomu` on `./combat` at score >= 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (command in testing.md); never as a bare host process. Highest value: the decrement and skip in FR-020/FR-024, tie-break and potency in FR-015, the `total >= 1` guard in FR-033, `onEnd` run-once in FR-038.

## 9. Open questions and decisions

No escalated question remains.

Settled, one line each (package decisions where the SRD is silent; changing any is **BREAKING**, flagged for review):

- Q1: expiry. The SRD gives durations and "a round ... about 6 seconds" (p. 13) and no tracking rule. A one-minute span is ten rounds and ends at the same Initiative count: the anchor's turn start N rounds later, so the anchor gets N turns (FR-006, FR-020). Evidence: the usual table ruling (research); the SRD-only justification is the 6-second round and the tracker facts. Interpretation.
- Q2: the registry is a value the host stores (state in, state out; `doc/rules/architecture.md` §4); `Encounter` is not changed (FR-007).
- Q3: `encounter-aggregate.md` is not a dependency (equal rank 28; FR-056). The commands take the round and active combatant as plain values; the wiring is `combat-action-resolution.md` (rank 29). `CombatantLeft` carries the departure the same way (FR-062).
- Q4: at one tick expiry runs before triggers; an expired effect does not trigger at that tick (FR-025). The SRD does not order them; either order is defensible, this one needs no special case for "until the end of the turn" effects.
- Q5: ending an effect plans condition removal, omitting a kind another effect in the registry still holds; sources outside the registry are the host's call (FR-045, p. 179).
- Q6: Concentration is a binding of caster and subject; the character spec owns the state and the host reports `ConcentrationLost` (FR-012, FR-041). A rejected start with a different bound subject guards a stale registry (FR-014).
- Q7: "end of each of its turns" is `TurnEndReached(subject)` (FR-027).
- Q8: Sleep and similar two-stage effects are stages with a gate, not two chained effects (FR-034); stage 2 "for the duration" is the rest of the spell clock, an `overall` clock with `EffectLastsRemainder`, never a fresh one (FR-061).
- Q9: Combining uses the template's integer potency; the SRD names "highest bonus" only (FR-019).
- Q10: the damage riders and their Critical Hit doubling are `spell-granted-modifiers.md`'s (FR-064); this spec only decides which effects apply.
- Q11: templates are `combat` values built by the host or a later data spec (RA-2); the pilot templates (Hold Person, Acid Arrow, Sleep, Flaming Sphere, Delayed Blast Fireball, Bless, Bane, Hex, Hunter's Mark; the last two with `survivesSubjectDeath`, FR-060) are test fixtures. Moving the model into `content` is rejected now (about fifteen types).
- Q12: out-of-combat time is a host-chosen `TimeElapsed` in rounds; the package keeps no calendar (FR-021).
- Q13: no damage trigger. "Ends when damaged" is `endsOnDamage` only (FR-033); the "someone takes an action to shake it out" ending is a host fact via `RemovedByEffect` (FR-044), pp. 141, 163.
- Q14: interpretation: a spell ends with its dead subject unless the template survives it (FR-043, FR-060); a revived creature does not regain it (returning-to-life.md keeps no registry; the host restarts an effect if the caster recasts). Sources: p. 179 covers only the caster, p. 141 and p. 140 give the move.
- Q15: ownership (package design under shared-types-and-ownership.md FR-048; no web evidence needed). Lifetime and moving are here; roll, damage and Armor Class figures are `spell-granted-modifiers.md`'s, and the combining key is shared (`caster` for Hex and Hunter's Mark, `target` otherwise, FR-015). Hunter's Mark and Hex pilot Q11 now exercise the move. `CombatantLeft` is a host-sent tick, not inferred (FR-062).

Amendments to other specs (returned to the orchestrator, not edited here):

- RA-1: `shared-types-and-ownership.md` section 4.11 rank table: add `active-effects` at rank 28; FR-015's combat order gains `spell-effects` and `active-effects` before `combat-action-resolution`. `documentation_index.md`: add a row after `spell-effects.md`.
- RA-2: `spell-effect-data.md`: E14 to E16 point to this spec; a later data spec maps `Spell.effects` to `EffectTemplate`.
- RA-3: `spell-effects.md` Q11 / FR-040: replace "a future active-effects spec" with a link to this one.
- RA-4: `combat-action-resolution.md` (rank 29): wire `ConcentrationEnded` to `EndActiveEffects`, `DamageTaken` to `EndActiveEffects`, tracker events to `AdvanceActiveEffects`.
- RA-5: `magical-effects.md`: note `OngoingEffects` and `ActiveEffects` are separate lists; a shift keeps the registry (carry-over, FR-047 there) and an exception ends effects through `RemovedByEffect`.
- RA-6: `dispel-and-counterspell.md`: `OngoingSpellRegistry` is withdrawn as a lifetime store or documented as a read model derived from `ActiveEffects`; a dispel ends spells through `EndActiveEffects` with `RemovedByEffect`, and `AdvanceOngoingSpells` is replaced by `AdvanceActiveEffects`.
- RA-7: `spell-granted-modifiers.md`: A6 is met without a dependency: `active-effects` supplies the inputs of FR-064 through the host; the combining key is defined here (FR-015), not there (its FR-003 to FR-006 are withdrawn); Bless/Bane on Death Saving Throws stays its FR-013; "moving a mark" is `MoveEffectSubject` (FR-060); neither lists the other in Depends on (equal rank 28).
- RA-8: `combatant-roster-changes.md` FR-048: the host sends `CombatantLeft` (FR-062) to `AdvanceActiveEffects` when it removes a combatant; `RemovedByEffect` for effects the host wants gone.

## 10. Changelog impact

Under `Unreleased`:

- **Added:** `StartActiveEffect`, `AdvanceActiveEffects`, `EndActiveEffects`, `ResolveEffectDamage`, `ActiveEffects` and the types of section 5, the `srd5e.combat.active_effects.*` signals and the new `srd5e.combat.result` values (added to `doc/observability/signals.md`).
- **Not BREAKING while unreleased.** After the first release, adding a variant to a closed result, changing the expiry rule (Q1) or the combination rule (FR-015), or renaming a signal or attribute is **BREAKING**.
