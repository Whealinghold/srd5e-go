# Encounter aggregate

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md`, `grappling.md`, `initiative-and-actions.md`, `d20-tests.md`, `errors-and-results.md`, `observability-ports.md`, `character-aggregate.md`, `content-schema-and-decoder.md` |

Reuse map (prose, not dependencies): `CombatantID`, `CombatantKind`, `CombatantRef`, `Event`, `CombatTelemetry` (combat-foundations.md FR-010, FR-013, FR-015 to FR-021, FR-027 to FR-038); `ConditionSet` (conditions.md); `d20.Outcome` (d20-tests.md); `CharacterID` only (character-aggregate.md); `ConditionKind` FR-110 and `CreatureSize` FR-109 (content-schema-and-decoder.md; shared-types-and-ownership.md FR-005, FR-046). `GrappleRules.Hold`, `GrappleRules.EndOnSet` (grappling.md FR-087, FR-089) and the types `GrappleEnded`, `GrappleEndCause`, `GrappleHold` are grappling.md's (shared-types-and-ownership.md FR-029, FR-048; section 4.11). `IsBloodied` (derived-stats.md FR-071; shared-types-and-ownership.md FR-049) is reached through hit-points-and-damage.md, which owns `BecameBloodied`. The spec that follows this one in the `combat` order calls this one and is neither a dependency nor named in the header (shared-types-and-ownership.md FR-015). Consumers (prose, FR-056): hosts build a `Combatant` for a monster from `MonsterSetup` (`ref`, `hitPointStatus`, `conditions`; monster-combatant-setup.md); `combatant-roster-changes.md` adds the three roster methods (FR-015); the per-turn feat-use records of feats-behaviour.md FR-049 live in the `TurnTracker`'s per-turn resources (section 9).

## 1. Purpose and scope

Defines the `Encounter` aggregate of the `combat` context: the one root that holds a combat from its start to its end. "Encounter" here means a combat encounter ("A typical combat encounter", SRD 5.2.1, p. 13); the SRD glossary term is wider (section 3). It covers:

- the root, its typed id (`EncounterID`) and the `Combatant` entity;
- the invariants: two sides, a turn order held by a `TurnTracker`, an in-progress or ended status, one consistent state per combatant;
- the "state in, new state and events out" contract: every transition returns a new `Encounter` and the ordered events it produced (section 5);
- composing rules defined elsewhere into one state change per command. Turn and round arithmetic go to the `TurnTracker`, hit point and death rules to `HitPointStatus`, condition rules to `ConditionSet`. The encounter adds only what needs two or more at once: Unconscious on dropping to 0 Hit Points, death by Exhaustion, the `DyingCombatantTurnStarted` fact, rejection order;
- applying the end of a grapple: the encounter holds both creatures and wraps `GrappleEnded` (shared-types-and-ownership.md FR-029). The grapple rules are grappling.md's and are not restated (shared-types-and-ownership.md FR-048; FR-086, FR-093, FR-094 keep only the encounter's part);
- a thin application layer: one use case per command, each returning its own `<UseCase>Result`, run through `CombatTelemetry`.

Damage arithmetic and the meaning of each condition belong to hit-points-and-damage.md and conditions.md; Initiative ordering, round arithmetic and turn events to initiative-and-actions.md. The spec that follows this one is the only place that wires attack results into `ApplyDamage` (combat-action-resolution.md FR-036, FR-075); this spec answers its two requests (FR-096, FR-097).

**Out of scope:**

- Rolling Initiative, Surprise, the Initiative count formula and tie-breaks (`RollInitiative`, `EstablishInitiativeOrder`). The encounter receives an established `TurnTracker`.
- Actions, Bonus Actions, Reactions, movement and Opportunity Attacks as commands, and attack results. The only hook is `ApplyTurnChange(tracker, events)` (FR-096). There is no `applyTurnStep` command and no attack input.
- Rolling the d20 of a Death Saving Throw or of a grapple escape check, and the 1d4 hours of the stable recovery delay. Every roll arrives as a command input; no `dice.Roller`.
- Ending Temporary Hit Points by a Long Rest, Short and Long Rests, removing Exhaustion by resting. `ReduceExhaustion` exists; the rest rules that call it do not.
- Ending a knock-out by first aid (the DC 10 Wisdom (Medicine) check) or by the Short Rest: the host calls `RemoveCondition` (FR-092). Regaining Hit Points is the encounter's.
- Reviving a dead creature (`Raise Dead`, `Revivify`; "Dead", p. 180); spell effects.
- Adding, removing or repositioning a combatant after start (including a controlled mount's changed Initiative, p. 16): combatant-roster-changes.md owns the three methods (FR-015). Fleeing or surrendering as tracked state stays out of scope.
- Grapple rules (the `Grapples` record, the Unarmed Strike grapple option, escape DC and check, release, movement while grappling): grappling.md and attacks.md. A host that keeps a `Grapples` record updates it from the wrapped `GrappleEnded` (grappling.md FR-060, FR-029). Positions are out of scope, so the host supplies distance and range (FR-086). Swallow and Engulf (`Containments`) are the host's too, beside `Grapples` (monster-engulf-and-swallow.md; section 9).
- Positions, Creature Size and Space, Cover, the grid (p. 13 to 15; the opt-in value layer is grid-positions-and-movement.md, held by the host, not by `Encounter`); durations measured in rounds (a round "represents about 6 seconds", p. 13, but no rule here depends on time); Mounted Combat (p. 15 to 16) and Underwater Combat (p. 16); persistence and display.

## 2. Source

- SRD 5.2.1, "Combat", "The Order of Combat", "Combat Step by Step", "Initiative", "Your Turn", p. 13; "Ending Combat", "Moving around Other Creatures", p. 14.
- SRD 5.2.1, "Damage and Healing", "Hit Points", "Damage Rolls", "Damage against Multiple Targets", p. 16; "Resistance and Vulnerability", "No Stacking", "Order of Application" (worked example), "Immunity", "Healing", "Dropping to 0 Hit Points", "Instant Death", "Monster Death", "Knocking Out a Creature", "Falling Unconscious", "Massive Damage", "Hit Point Maximum of 0", "Death Saving Throws", p. 17; "Death Saving Throws" (continued), "Rolling a 1 or 20", "Damage at 0 Hit Points", "Stabilizing a Character", "Temporary Hit Points", "Lose Temporary Hit Points First", p. 18.
- SRD 5.2.1, "Rules Glossary": "Condition", p. 179; "Dead", p. 180; "Death Saving Throw", "Exhaustion [Condition]", "Encounter", p. 181; "Grappled [Condition]", "Grappling", p. 182; "Hit Points", "Immunity", p. 183; "Incapacitated [Condition]", "Initiative", "Knocking Out a Creature", p. 184; "Monster", p. 185; "Paralyzed [Condition]", "Player Character", p. 186; "Speed", "Stable", "Stat Block", p. 188; "Unconscious [Condition]", p. 191.
- Package design: `doc/rules/architecture.md` §2, §3, §4, §6; `doc/rules/observability.md` §2, §3, §5 to §7; shared-types-and-ownership.md (binding, cited as FR-nnn there); combat-foundations.md (naming, events, telemetry).

Page numbers are printed numbers; the PDF index matches them (checked against pages 12 to 20 and 176 to 191).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Encounter | `Encounter` (aggregate root) | Glossary: "a scene in an adventure that is part of at least one of the game's three pillars" (p. 181). Here one combat encounter ("a clash between two sides", p. 13), first turn to end. Kept as `Encounter` because the context is `combat`; rename only if another encounter kind appears. |
| (encounter identity) | `type EncounterID string`; `ParseEncounterID(string) (EncounterID, error)` (a `*EncounterIDError` carrying an `EncounterIDRejection`, enum: `Empty`); `MustEncounterID` | Not an SRD term. Mirrors `CombatantID`. |
| Combatant | `Combatant` (entity) | "each participant in a battle" (p. 13). Holds a `CombatantRef`, a side, a `HitPointStatus`, a `ConditionSet`, the markers `knockedOut` and `deathSavingThrowMade` (FR-098, FR-099). |
| (combatant identity) | `CombatantID` (combat-foundations.md; shared-types-and-ownership.md FR-012) | Reused; the same id names the combatant in the `TurnTracker` and is the target and source id in conditions. No `CreatureID`. |
| Player Character / Monster | `CombatantKind`: `PlayerCharacterCombatant(CharacterID)`, `MonsterCombatant(MonsterID)`; `CombatantRef` | p. 186, p. 185. Identity only; no rule reads the kind. |
| Side | `Side` (enum: `sideA`, `sideB`) | p. 13. Two sides is a design decision (FR-011). |
| Round, Turn | `TurnTracker.Round` (starts at 1), `TurnTracker.ActiveCombatant` | p. 13. The 6 seconds is not modelled. |
| Initiative, Initiative order | `InitiativeCount`, `TurnOrder` (initiative-and-actions.md) | p. 13. Held in the `TurnOrder`, not in `Combatant`. |
| Hit Points, Temporary Hit Points, Stable, Dead | `HitPointStatus` (`HitPoints`, `LifeState`, `ZeroHitPointsRule`, `DamageAffinities`) | p. 16 to 18, 180, 183, 188. |
| Damage | `Damage`, `DamageDelivery`, `HitKind`, `DamageResolution` | p. 16, 17. `DamageType` is a `content` type. |
| Condition, Exhaustion, Immunity | `Condition`, `ConditionSet`, `ConditionKind`, `ExhaustionLevel`; `DamageAffinities.Immunities`, `ConditionSet.Immunities` | p. 179, 181, 183. Exhaustion is the exception to "a condition doesn't stack with itself". |
| Grapple end | `GrappleEnded`, `GrappleEndCause`, `GrappleHold`, `GrappleRules.Hold`, `GrappleRules.EndOnSet` | "Ending a Grapple", p. 182. Owned by grappling.md (FR-085 `GrappleEndCause`; FR-081, FR-086 `GrappleEnded`; FR-087, FR-088 `hold`; FR-089 to FR-091 `endOnSet`). |
| Death | `Combatant.Death` (`DeathCause?`) and the wrapped `Died` | "A dead creature has no Hit Points" (p. 180). |
| Death Saving Throw | `DyingCombatantTurnStarted` (fact that a Dying combatant's turn began); `ApplyDeathSavingThrowToCombatant` (command with the natural face and the total) | p. 17, 181. Not a prompt (combat-foundations.md FR-020). |
| Ending Combat | `EndReason` (closed: `EndReasonSideDefeated(Side)`, `EndReasonMutualAgreement`); `EncounterStatus` (closed: `EncounterInProgress`, `EncounterEndedStatus(EndReason)`) | p. 14. |
| `EncounterEvent` | closed sub-family satisfying `Event` (combat-foundations.md FR-015, FR-016) | Facts a sibling spec defines are wrapped, not redefined. |
| `EncounterTransition` | closed, domain level, not exported: `TransitionApplied`, `TransitionRejected` | Each use case maps it to its own `<UseCase>Result` (FR-089). |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids (the highest so far is FR-100). Moved requirements are in "Withdrawn ids" (4.12). "Wrapped" means the sibling spec's event is carried unchanged inside an `EncounterEvent` envelope that adds the `EncounterID` and the round (FR-084). In the case tables an event name without comment is such a wrapped event; encounter-only events are marked "(encounter)".

### 4.1 Identity and construction

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `ParseEncounterID` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for an empty string or a string of only whitespace. | n/a (architecture §3, §6) | unhappy (invalid input) |
| FR-002 | `ParseEncounterID` MUST return the id and a nil error for a string it accepts and a `*EncounterIDError` carrying `EncounterIDRejection.Empty` for one it rejects (`errors.As`-able); it MUST NOT panic for any string. `MustEncounterID` MUST panic with that same error if and only if `ParseEncounterID` returns one. Mirrors `ParseCombatantID` (combat-foundations.md FR-005; errors-and-results.md FR-025, FR-035). | n/a (architecture §6) | unhappy (invalid input) |
| FR-005 | Two `EncounterID`s from the same string MUST be equal under `==`; different strings MUST differ. An `EncounterID` MUST NOT be assignable where a `CombatantID` is expected, nor the reverse: both are distinct named string types, so the compiler enforces it and this spec adds no test (a conversion is always explicit). | n/a (architecture §3) | happy |

### 4.2 Starting an encounter

The turn order arrives as the `InitiativeOrderEstablished(tracker, events)` variant of `EstablishInitiativeOrderResult` (initiative-and-actions.md FR-018 to FR-027, FR-123). Ranking, ties and tie-breaks are decided there; the encounter does not rank again.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-007 | `Encounter.Start(id, established, combatants)` with valid input MUST return `Started(encounter, events)` (the domain variant of `EncounterStart`; the use case maps it to `StartEncounterApplied`) whose encounter is `EncounterInProgress`, holds the `TurnTracker` of `established` (round 1, first combatant active) and the combatants stored in the tracker's turn order: the input list permuted stably by the tracker's order (`CombatantID` is unique, FR-012), not the order the host passed. The combatants themselves are unchanged. | SRD 5.2.1, "Combat Step by Step", p. 13 | happy |
| FR-008 | Starting MUST produce exactly these events in order: `EncounterStarted` (encounter), the events of `established` wrapped (`TrackerRoundStarted` 1, `TrackerTurnStarted` for the first combatant), then `DyingCombatantTurnStarted` for that combatant when FR-018 applies. | SRD 5.2.1, "Combat Step by Step", p. 13 | happy |
| FR-009 | The root delegates ordering: starting MUST accept only a tracker from `InitiativeOrderEstablished`, MUST NOT re-sort or re-validate its order, and the encounter's turn order MUST equal the tracker's (initiative-and-actions.md FR-018). Two tests: tracker order C, A, B is kept; the same start with combatants passed as A, B, C stores C, A, B and equals the one built from C, A, B (FR-007, FR-053). | SRD 5.2.1, "Initiative", "Initiative Order", p. 13 | unhappy (rule interaction) |
| FR-010 | The root delegates ties: a tracker whose order came from a tie-break MUST be accepted as is and keep its relative order (initiative-and-actions.md FR-019 to FR-021). One test with a tied pair. | SRD 5.2.1, "Initiative", "Ties", p. 13 | unhappy (boundary) |
| FR-011 | Starting MUST reject a combatant list that is empty or in which one `Side` has no combatant; one per side is the minimum accepted. The limit to two sides is a design decision, not an SRD rule (the SRD says "a clash between two sides"): it keeps `Side` and the victory claim simple, and a three-sided combat needs a richer `Side` and a new spec. | SRD 5.2.1, "The Order of Combat", p. 13; design decision (this row) | unhappy (empty and minimal) |
| FR-012 | Starting MUST reject two combatants with the same `CombatantID`. | n/a (architecture §3, entity identity) | unhappy (invalid input) |
| FR-013 | `Encounter.Start(id, established, combatants)` MUST return `(Encounter, []Event, error)`. Untrusted host input that fails validation yields a non-nil error that is the `errors.Join` of one typed `EncounterProblem` per problem found, in input order, and satisfies `errors.Is(err, srderr.ErrInvalidInput)`; `ProblemsOf(err)` returns them. Problems are `EncounterProblem`s (section 5): `NoCombatants`, `MissingSide(side)`, `DuplicateCombatantID(id)`, `CombatantNotInTurnOrder(id)`, `TurnOrderEntryWithoutCombatant(id)`. `Start` MUST NOT panic for any input; `MustStart` panics with the same error. | n/a (architecture §5, §6) | unhappy (invalid input) |
| FR-014 | The encounter MUST NOT change if the caller later mutates the list passed to `Start` (it takes its own copy). | n/a (architecture §4, handoffs) | unhappy (immutability) |
| FR-015 | After start no transition of this spec MUST change the order or the size of the combatants list; it stays in the tracker's turn order, the same in every round. Only `AddCombatant`, `RemoveCombatant` and `RepositionCombatant` of combatant-roster-changes.md (its FR-008, FR-016, FR-028, FR-038) may, and they keep FR-007 and FR-052 true. That includes the controlled mount, whose Initiative "changes to match yours" (p. 16): `RepositionCombatant` after the rider, so the mount acts right after it among equal counts (mounted-and-underwater-combat.md F-2). One mechanism serves summoned creatures joining and leaving. | SRD 5.2.1, "Initiative Order", p. 13; "Controlling a Mount", p. 16 | happy |

### 4.3 Turns and rounds

`AdvanceTurn` calls the domain operation `TurnTracker.Advance` (initiative-and-actions.md FR-141, FR-027 to FR-043, FR-135, which returns a `TrackerStep`), never the `AdvanceTrackerTurn` use case: domain files may not call a use case (`*_usecase.go`) and a use case never calls another (shared-types-and-ownership.md FR-044; architecture §1, §4). The encounter adds only the envelope and `DyingCombatantTurnStarted`, so FR-016 and FR-017 test the envelope and the new tracker state, not the arithmetic.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-016 | `AdvanceTurn` in the middle of the order MUST replace the tracker with the result of the tracker's advance operation and wrap its events in the order returned (`TrackerTurnEnded` previous, `TrackerTurnStarted` next, plus any other event the tracker returns). | SRD 5.2.1, "The Order of Combat", p. 13 | happy |
| FR-017 | `AdvanceTurn` from the last combatant MUST do the same, and the wrapped events MUST be `TrackerTurnEnded` (last), `TrackerRoundStarted` (new round), `TrackerTurnStarted` (first), in that order (table 4.3.1). | SRD 5.2.1, "The Order of Combat", p. 13 | unhappy (boundary) |
| FR-018 | After a `TrackerTurnStarted` (at start and at an advance) the encounter MUST produce `DyingCombatantTurnStarted` (encounter) for the active combatant when its `HitPointStatus.MakesDeathSavingThrows` is true, that is, its `LifeState` is `Dying` (hit-points-and-damage.md FR-096). It states a fact, not a prompt; the host decides to roll and calls `ApplyDeathSavingThrow` (combat-foundations.md FR-020). The rule reads `LifeState`, never `CombatantKind`: a monster given `FallsUnconscious` produces it. It MUST be the last event of the start or advance, after every wrapped tracker event including a `ReadiedActionExpired` returned after `TrackerTurnStarted` for the same combatant (initiative-and-actions.md FR-135, step 6); nothing is inserted between wrapped tracker events. | SRD 5.2.1, "Death Saving Throws", p. 17; "Death Saving Throw", p. 181; "Monster Death", p. 17 | happy |
| FR-019 | After a `TrackerTurnStarted` the encounter MUST NOT produce `DyingCombatantTurnStarted` when the active combatant's `LifeState` is `Alive`, `Stable` or `Dead` (dead of Exhaustion is `Dead(exhaustion)`, FR-037). A monster whose `HitPointStatus` carries `DiesInstantly` (the host supplies the rule, hit-points-and-damage.md FR-087) is never `Dying` (table 4.3.2). | SRD 5.2.1, "Death Saving Throws", p. 17; "Stable", p. 188; "Monster Death", p. 17 | unhappy (rule interaction) |
| FR-020 | `AdvanceTurn` MUST NOT skip a combatant that is dead, Unconscious or at 0 Hit Points; every combatant in the order gets a `TrackerTurnStarted` each round. The Initiative order "remains the same from round to round" and a creature at 0 Hit Points still has turns (Death Saving Throws); a host that wants to skip the dead calls `AdvanceTurn` again. | SRD 5.2.1, "Initiative Order", p. 13; design decision (this row) | unhappy (rule interaction) |
| FR-096 | `ApplyTurnChange(tracker, events)` MUST, for a `tracker` whose `TurnOrder` equals the encounter's, return `TransitionApplied` whose encounter holds that tracker and whose events are the given tracker events of one step, each wrapped in `TurnEventOccurred` (FR-084) in the order given, with the `EncounterID` and the round of FR-051. It MUST wrap every kind of tracker event (the 13 of initiative-and-actions.md FR-125: `TrackerRoundStarted`, `TrackerTurnStarted`, `TrackerTurnEnded`, `ActionTaken`, `BonusActionUsed`, `ReactionTaken`, `MovementSpent`, `ObjectInteracted`, `DroppedProne`, `StoodUp`, `FellProneInSharedSpace`, `ReadiedActionExpired`, `MovementSpeedsUpdated`), add no event of its own (in particular no `DyingCombatantTurnStarted`: FR-018 belongs to `Start` and `AdvanceTurn`) and change no combatant, status or combatant order. An empty `events` list is valid. It has no use case here: the use cases of combat-action-resolution.md call it as domain code (its FR-008, FR-014). | n/a (architecture §3, §4); combat-action-resolution.md FR-008, FR-014; initiative-and-actions.md FR-125, FR-128 | happy |
| FR-097 | `ApplyTurnChange` MUST return `TransitionRejected` and change nothing when, in this order: (1) the encounter has ended (`EncounterAlreadyEnded`, FR-045); (2) an event names a `CombatantID` that is not a combatant (`UnknownCombatant`, the smallest id in code unit order, FR-078); (3) the `TurnOrder` of `tracker` is not equal to the encounter's (`TurnOrderChanged`, exported class `TurnOrderChangedRejection`, FR-090; the root keeps no second order, FR-015). No combatant is named by the command, so `CombatantIsDead` does not apply (the caller stops a dead combatant's step, combat-action-resolution.md FR-012). Whether the new tracker succeeds the stored one cannot be detected; a stale tracker with an equal order is accepted (combat-action-resolution.md Q-1). | n/a (architecture §3, §6); combat-action-resolution.md FR-014 | unhappy (state, rule interaction) |

#### 4.3.1 Turn and round arithmetic (FR-016, FR-017)

Order is A, B, C; the encounter starts with A active in round 1. All are above 0 Hit Points and Alive, so no `DyingCombatantTurnStarted` appears except where a row says so.

| Label | Advances so far | Active after | Round after | Events of the last advance | Source | Path |
|---|---|---|---|---|---|---|
| first and second advance | 1, 2 | B, then C | 1 | `TrackerTurnEnded` A, `TrackerTurnStarted` B; then `TrackerTurnEnded` B, `TrackerTurnStarted` C | p. 13 | happy |
| wrap to round 2 | 3 | A | 2 | `TrackerTurnEnded` C, `TrackerRoundStarted` 2, `TrackerTurnStarted` A | p. 13 | unhappy (boundary) |
| one past the wrap | 4 | B | 2 | `TrackerTurnEnded` A, `TrackerTurnStarted` B | p. 13 | unhappy (boundary) |
| two full rounds | 6 | A | 3 | `TrackerTurnEnded` C, `TrackerRoundStarted` 3, `TrackerTurnStarted` A | p. 13 | unhappy (boundary) |
| minimal encounter (A and B), wrap | 2 | A | 2 | `TrackerTurnEnded` B, `TrackerRoundStarted` 2, `TrackerTurnStarted` A | p. 13 | unhappy (empty and minimal) |
| next is a Dying player character (C, 0 Hit Points) | 2 | C | 1 | `TrackerTurnEnded` B, `TrackerTurnStarted` C, `DyingCombatantTurnStarted` C (encounter) | p. 13, p. 17 | unhappy (rule interaction) |
| Dying with an expiring readied action (C, 0 Hit Points) | 2 | C | 1 | `TrackerTurnEnded` B, `TrackerTurnStarted` C, `ReadiedActionExpired` C, `DyingCombatantTurnStarted` C (encounter) last | p. 13, p. 17; initiative-and-actions.md FR-135 | unhappy (rule interaction) |

#### 4.3.2 Dying combatant's turn started (FR-018, FR-019)

| Label | Kind | `ZeroHitPointsRule` | Hit Points | `LifeState` | Exhaustion | `DyingCombatantTurnStarted` | Source | Path |
|---|---|---|---|---|---|---|---|---|
| player character at 0 | player character | `FallsUnconscious` | 0 | `Dying` | none | yes | p. 17 | happy |
| player character at 1 | player character | `FallsUnconscious` | 1 | `Alive` | none | no | p. 17 ("start your turn with 0") | unhappy (boundary) |
| stable player character | player character | `FallsUnconscious` | 0 | `Stable` | none | no | p. 188 | unhappy (rule interaction) |
| dead player character | player character | `FallsUnconscious` | 0 | `Dead` | none | no | p. 180 | unhappy (state) |
| monster with `DiesInstantly` at 0 | monster | `DiesInstantly` | 0 | `Dead` (monster death) | none | no | p. 17 | unhappy (rule interaction) |
| monster the GM treats like a character | monster | `FallsUnconscious` | 0 | `Dying` | none | yes | p. 17, "Monster Death" | unhappy (rule interaction) |
| dead of Exhaustion | player character | `FallsUnconscious` | 0 | `Dead` (`exhaustion`) | 6 | no | p. 181 | unhappy (rule interaction) |

#### 4.3.3 Turn change cases (FR-096, FR-097)

Order is A, B, C; round 1, A active.

| Label | Command | Expected | Source | Path |
|---|---|---|---|---|
| action spent | `ApplyTurnChange(tracker with A's action used, [ActionTaken A])` | tracker replaced; `TurnEventOccurred(ActionTaken)`; combatants equal; no `DyingCombatantTurnStarted` | p. 13 | happy |
| every event kind | the 13 tracker event kinds, one per call | each wrapped under its own `kind`, none dropped | initiative-and-actions.md FR-125 | happy |
| several events in order | `[MovementSpent A, ObjectInteracted A]` | two wrapped events, same order | p. 13 | unhappy (rule interaction) |
| no events | `ApplyTurnChange(tracker, [])` | tracker replaced, no events, combatants equal | n/a | unhappy (empty and minimal) |
| Dying active combatant | A is Dying; `[ActionTaken A]` | `TurnEventOccurred(ActionTaken)` only | p. 17 | unhappy (rule interaction) |
| different order | tracker with order B, A, C | `TransitionRejected` (`TurnOrderChanged`); encounter equal | n/a | unhappy (state) |
| unknown id in an event | `[ActionTaken z]` | `UnknownCombatant` z | n/a | unhappy (invalid input) |
| two unknown ids | events naming `z` then `b` | `UnknownCombatant` carrying `b` | n/a | unhappy (rule interaction) |
| ended or unknown id, and different order | tracker with another order, on an ended encounter; or with an event naming an unknown id | `EncounterAlreadyEnded`; `UnknownCombatant` | p. 14 | unhappy (rule interaction) |

### 4.4 Damage

Damage arithmetic is hit-points-and-damage.md's (`DamageResolver`, FR-016 to FR-025 there). FR-021 to FR-029, FR-070 to FR-072 and FR-091 say what the encounter does with the result. A `Damage` is one instance against one combatant; damage to several targets (p. 16, "Damage against Multiple Targets") is one command per target.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-021 | `ApplyDamage` for a living combatant MUST replace that combatant's `HitPointStatus` with the status the hit point rules return and wrap the events they return. No other combatant, nor the round or active combatant, changes. | SRD 5.2.1, "Hit Points", "Damage Rolls", p. 16 | happy |
| FR-022 | The breakdown of the damage (amount after adjustments, Resistance, Vulnerability and Immunity; amount absorbed by Temporary Hit Points; Hit Points lost) MUST reach the host through the `DamageResolution`s (one per `Damage`, `DamageApplied.Resolutions` in hit-points-and-damage.md) carried by the `TransitionApplied` and by `ApplyDamageToCombatantApplied`; the wrapped `DamageTaken` carries what the hit points spec defines. The encounter MUST NOT recompute or restate either. | SRD 5.2.1, "Order of Application", "Immunity", p. 17; "Lose Temporary Hit Points First", p. 18 | happy |
| FR-023 | When the hit point rules return `FellUnconscious` (Dying at 0 Hit Points), the encounter MUST apply Unconscious through `ConditionSet.Apply` and, when the result is `ConditionAdded`, produce the wrapped `ConditionApplied` (Unconscious) after the hit point events. | SRD 5.2.1, "Falling Unconscious", p. 17 | happy |
| FR-024 | When damage reduces a combatant to 0 Hit Points and its `ZeroHitPointsRule` is `DiesInstantly` (hit-points-and-damage.md FR-087), the rules leave it `Dead(monsterDeath)` and return `DroppedToZeroHitPoints` then `Died` (FR-041, FR-042 there); the encounter MUST wrap them (FR-084) and MUST NOT add Unconscious. A monster given `FallsUnconscious` follows FR-023. | SRD 5.2.1, "Monster Death", p. 17 | happy |
| FR-025 | When damage reduces a `FallsUnconscious` combatant to 0 Hit Points and the remainder equals or exceeds its Hit Point maximum, the rules leave it `Dead(massiveDamage)` and return `Died` (wrapped); the encounter MUST NOT add Unconscious. One less than the maximum MUST NOT kill it. | SRD 5.2.1, "Massive Damage", p. 17 | unhappy (boundary) |
| FR-026 | Damage taken while a `Dying` combatant already has 0 Hit Points MUST wrap the hit point events (the failures are in the status), including `Died` when the status becomes `Dead` (a third failure, or damage that equals or exceeds the maximum). A Critical Hit (`HitKind.CriticalHit`) adds two failures. | SRD 5.2.1, "Damage at 0 Hit Points", p. 18 | unhappy (rule interaction) |
| FR-027 | When the combatant falls to 0 Hit Points while already holding Unconscious from another source (a spell), `ConditionSet.Apply` returns `ConditionAlreadyHeld`: the encounter MUST NOT produce a second `ConditionApplied`, and the combatant MUST hold Unconscious exactly once. | SRD 5.2.1, "Condition", p. 179 | unhappy (rule interaction) |
| FR-028 | Damage that is 0 after the pipeline (a 0 roll, or Immunity) MUST NOT produce `DamageTaken`, add a Death Saving Throw failure or change conditions (hit-points-and-damage.md FR-048). The transition is `TransitionApplied`, status unchanged, no events. Damage above 0 that Temporary Hit Points fully absorb is FR-091. | SRD 5.2.1, "Damage Rolls", p. 16; "Immunity", p. 17 | unhappy (boundary) |
| FR-029 | `ApplyDamage` for a dead combatant MUST return `TransitionRejected` with reason `CombatantIsDead`. The SRD says a dead creature "has no Hit Points" and gives no rule for it; hit-points-and-damage.md FR-066, FR-039 and Q-7 reject too. | SRD 5.2.1, "Dead", p. 180; design decision (this row) | unhappy (state) |
| FR-070 | When the rules return `KnockedOut` (damage delivered as `MeleeAttackKnockOut` that would have reduced the combatant to 0), the combatant MUST be left with the 1 Hit Point the rules set, and the encounter MUST apply Unconscious through `ConditionSet.Apply` and produce the wrapped `ConditionApplied` (Unconscious) after `KnockedOut`. Knocking out MUST NOT produce `Died` or `DyingCombatantTurnStarted`, and sets `knockedOut` (FR-098). | SRD 5.2.1, "Knocking Out a Creature", p. 17; hit-points-and-damage.md FR-050 | unhappy (rule interaction) |
| FR-071 | `ApplyDamageToCombatantCommand` (not `ApplyDamageCommand`, which hit-points-and-damage.md keeps; shared-types-and-ownership.md FR-021) MUST carry the current `Encounter`, the `CombatantID`, a non-empty list of `Damage`, a list of `DamageAdjustment`s (flat adjustments and `HalfDamage`), a `DamageDelivery` and a `HitKind` (the inputs of the hit point spec's `ApplyDamage`), and pass them through unchanged. Knocking Out is `DamageDelivery.MeleeAttackKnockOut`, not a separate command. | SRD 5.2.1, "Damage Rolls", p. 16; "Knocking Out a Creature", p. 17 | happy |
| FR-072 | Damage that takes Hit Points from a `Stable` combatant MUST make it `Dying` with one failure (two for a Critical Hit) as hit-points-and-damage.md FR-071 states (its Q-3 is settled the same way, section 9), wrapping the events returned (including `LostStability`). It stays Unconscious, so the encounter MUST NOT produce `ConditionApplied`; the events include `Died` only when the status becomes `Dead`. | SRD 5.2.1, "Stabilizing a Character", "Damage at 0 Hit Points", p. 18 | unhappy (rule interaction) |
| FR-091 | Damage above 0 after the pipeline and fully absorbed by Temporary Hit Points MUST produce the wrapped `DamageTaken` with 0 Hit Points lost, and the wrapped `TemporaryHitPointsEnded` when they reach 0 (hit-points-and-damage.md FR-084, table 4.9a). It MUST NOT add a failure or change `LifeState` or conditions. The transition is `TransitionApplied`; this holds at 0 Hit Points too. | SRD 5.2.1, "Hit Points", p. 16; "Lose Temporary Hit Points First", p. 18; hit-points-and-damage.md FR-084 | unhappy (boundary) |

#### 4.4.1 Damage event cases (FR-021 to FR-029, FR-070, FR-072, FR-091)

One row, one table-driven test. "Incoming" is what the command carries; "After pipeline" is the amount taken from Temporary Hit Points and Hit Points. Events are complete and in order: the hit point events are those of hit-points-and-damage.md table 4.9a for the transition (bracketed optional events listed when the state makes them apply), each wrapped (FR-084), followed by the wrapped `ConditionApplied` where named. A combatant at or below half its maximum is Bloodied (`IsBloodied`), so `BecameBloodied` appears only where damage takes a combatant from above half to half or below and it is not dead afterwards (hit-points-and-damage.md FR-101). "Normal" is `HitKind.NormalHit` with `DamageDelivery.OtherSource`.

| Label | Combatant before | Incoming | After pipeline | Remaining | Events | Source | Path |
|---|---|---|---|---|---|---|---|
| ordinary damage | player character, 10 of 20 (already Bloodied) | 5 Slashing, normal | 5 | 5 | `DamageTaken` | p. 16 | happy |
| exactly to 0, character | player character, 6 of 12 | 6 Slashing, normal | 6 | 0 (Dying) | `DamageTaken`, `DroppedToZeroHitPoints`, `FellUnconscious`, `ConditionApplied` (Unconscious) | p. 17 | unhappy (boundary) |
| overkill below maximum | player character, 6 of 12 | 17 Slashing, normal | 17 (remainder 11) | 0 (Dying) | as above | p. 17 | unhappy (boundary) |
| massive damage, remainder equals maximum | player character, 6 of 12 | 18 Slashing, normal | 18 (remainder 12) | 0 (Dead) | `DamageTaken`, `DroppedToZeroHitPoints`, `Died` (`massive_damage`); no `ConditionApplied` | p. 17 (worked example) | unhappy (boundary) |
| massive damage, remainder above maximum | player character, 6 of 12 | 19 Slashing, normal | 19 (remainder 13) | 0 (Dead) | as above | p. 17 | unhappy (boundary) |
| monster with `DiesInstantly` to 0 | monster (`DiesInstantly`), 7 of 7 | 7 Slashing, normal | 7 | 0 (Dead) | `DamageTaken`, `DroppedToZeroHitPoints`, `Died` (`monster_death`); no `ConditionApplied`, no `BecameBloodied` | p. 17 | happy |
| monster treated like a character | monster (`FallsUnconscious`), 7 of 7 | 7 Slashing, normal | 7 | 0 (Dying) | `DamageTaken`, `BecameBloodied`, `DroppedToZeroHitPoints`, `FellUnconscious`, `ConditionApplied` (Unconscious); no `Died`; `DyingCombatantTurnStarted` at its next turn | p. 17, "Monster Death" | unhappy (rule interaction) |
| Temporary Hit Points absorb part | player character, 10 of 20 plus 5 temporary | 7 Slashing, normal | 7 (5 absorbed, 2 lost) | 8 | `DamageTaken`, `TemporaryHitPointsEnded` | p. 18 (worked example) | happy |
| Temporary Hit Points absorb all | 10 of 20 plus 5 temporary | 5 Slashing, normal | 5 (5 absorbed, 0 lost) | 10, temporary 0 | `DamageTaken` (0 lost), `TemporaryHitPointsEnded`; `TransitionApplied` (FR-091) | p. 18; hit-points-and-damage.md FR-084 | unhappy (boundary) |
| damage below the Temporary Hit Points | 10 of 20 plus 5 temporary | 3 Slashing, normal | 3 (3 absorbed, 0 lost) | 10, temporary 2 | `DamageTaken` (0 lost); no `TemporaryHitPointsEnded` | p. 18 | unhappy (boundary) |
| absorbed while Dying at 0 | 0 Hit Points, Dying, 0 failures, 5 temporary | 3 Slashing, normal | 3 (3 absorbed, 0 lost) | 0, temporary 2 | `DamageTaken` (0 lost); no failure; no `ConditionApplied` | p. 18 | unhappy (rule interaction) |
| Resistance to all, Vulnerability to Fire, aura | monster, 40 of 40, Resistance to all damage, Vulnerability to Fire | 28 Fire, normal, flat -5 (the aura) | 22 (28, 23, 11, 22) | 18 | `DamageTaken` (22 lost), `BecameBloodied` (40 to 18 of 40); the transition's `DamageResolution` lists the four steps | p. 17 (worked example); hit-points-and-damage.md table 4.3a | unhappy (rule interaction) |
| zero damage; immune damage type | 10 of 20; monster, 12 of 12, Immunity to Poison | 0 Slashing; 9 Poison, normal | 0 | 10; 12 | none | pp. 16, 17 | unhappy (boundary) |
| Unconscious from a spell, then dropped to 0 | 5 of 20, already Unconscious (explicit) | 5 Slashing, normal | 5 | 0 (Dying) | `DamageTaken`, `DroppedToZeroHitPoints`, `FellUnconscious`; no `ConditionApplied`; one Unconscious in the set | p. 17, p. 179 | unhappy (rule interaction) |
| damage at 0, already Unconscious | 0 Hit Points, Dying, Unconscious | 1 Slashing, normal | 1 | 0 | `DamageTaken` (failures 1); no `ConditionApplied` | p. 18, p. 179 | unhappy (rule interaction) |
| third failure by damage | 0 Hit Points, Dying, 2 failures | 1 Slashing, normal | 1 | 0 (Dead) | `DamageTaken` (failures 3), `Died` (`death_saving_throw_failures`) | p. 18 | unhappy (rule interaction) |
| critical hit at 0 | 0 Hit Points, Dying, 0 failures | 1 Slashing, `HitKind.CriticalHit` | 1 | 0 | `DamageTaken` (failures 2) | p. 18 | unhappy (rule interaction) |
| damage to a Stable creature | 0 Hit Points, Stable, Unconscious | 3 Slashing, normal | 3 | 0 (Dying, failures 1) | `DamageTaken`, `LostStability`; no `ConditionApplied`; no `Died` | p. 18 | unhappy (rule interaction) |
| knock-out | 6 of 12 (already Bloodied) | 6 Slashing, `meleeAttackKnockOut` | n/a | 1 (Alive) | `DamageTaken`, `KnockedOut`, `ConditionApplied` (Unconscious); `knockedOut` set; no `Died`, no `DyingCombatantTurnStarted` | p. 17 | unhappy (rule interaction) |
| knock-out requested, does not reach 0 | 10 of 12 | 3 Slashing, `meleeAttackKnockOut` | 3 | 7 | `DamageTaken`; no `KnockedOut` | p. 17; hit-points-and-damage.md FR-051 | unhappy (boundary) |
| damage to the dead | dead combatant | 4 Slashing, normal | n/a | n/a | `TransitionRejected` (`CombatantIsDead`) | p. 180 | unhappy (state) |

### 4.5 Healing

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-030 | `ApplyHealing` for a living combatant MUST add the restored Hit Points up to the maximum, never above, by `HitPointStatus.Restore` (hit-points-and-damage.md FR-109), and wrap the events returned (`HitPointsRestored`, and `Recovered` when a combatant at 0 returns to Alive). The amount regained is read from them. | SRD 5.2.1, "Healing", p. 17 | happy |
| FR-031 | When healing restores at least 1 Hit Point to a combatant at 0 Hit Points that holds Unconscious, the encounter MUST remove Unconscious through `ConditionSet.Remove` and produce the wrapped `ConditionRemoved` (Unconscious) after the hit point events, for a `Dying` and for a `Stable` combatant. It is left Prone (FR-066). | SRD 5.2.1, "Falling Unconscious", p. 17 | happy |
| FR-032 | Healing that restores 0 Hit Points (at maximum, or amount 0) MUST change neither Hit Points nor conditions, MUST NOT restore a combatant at 0, and MUST NOT produce `ConditionRemoved`. The transition is `TransitionApplied`; its events are those the rules return (hit-points-and-damage.md FR-036, FR-037). | SRD 5.2.1, "Healing", "Falling Unconscious", p. 17 | unhappy (boundary) |
| FR-033 | `ApplyHealing` for a dead combatant MUST return `TransitionRejected` with reason `CombatantIsDead`. | SRD 5.2.1, "Dead", p. 180 | unhappy (state) |
| FR-080 | `ApplyHealingToCombatantCommand` (not `ApplyHealingCommand`; shared-types-and-ownership.md FR-021) MUST carry `int amount`; an amount below 0 MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when the command is built, before any rejection (hit-points-and-damage.md FR-037). An amount of 0 is valid (FR-032). | SRD 5.2.1, "Healing", p. 17 | unhappy (invalid input) |
| FR-092 | Healing that restores at least 1 Hit Point to a combatant with `knockedOut` set (Alive at 1 Hit Point, FR-070) MUST remove Unconscious as FR-031 does (`ConditionRemoved` after the hit point events; Prone stays, FR-066) and clear the marker. Healing an Unconscious combatant above 0 Hit Points that is not `knockedOut` (a spell's Unconscious) MUST NOT remove it. First aid (the DC 10 Wisdom (Medicine) check) and the end of the Short Rest are the host's: it calls `RemoveCondition` (Unconscious), which clears the marker. | SRD 5.2.1, "Knocking Out a Creature", pp. 17 and 184 | unhappy (rule interaction) |
| FR-098 | `Combatant.KnockedOut` (default false) is the source marker of a knock-out. It MUST be set when `KnockedOut` is returned and `ConditionSet.Apply` (Unconscious) returned `ConditionAdded` (FR-070); it MUST stay false when Unconscious was already held or `ConditionImmune` (FR-067), so a spell's Unconscious is never ended by healing. It MUST be cleared whenever Unconscious is removed (FR-031, FR-039, FR-075, FR-092) and when the combatant later drops to 0 Hit Points (FR-023): that Unconscious belongs to the Dying rules. It is a marker beside the `ConditionSet` (conditions.md is unchanged; its sources name creatures only, FR-003). | SRD 5.2.1, "Knocking Out a Creature", p. 17; conditions.md FR-003 | unhappy (rule interaction) |

#### 4.5.1 Healing cases (FR-030 to FR-033, FR-080, FR-092)

| Label | Combatant before | Amount | After | Events | Source | Path |
|---|---|---|---|---|---|---|
| partial healing | 14 of 20 | 5 | 19 | `HitPointsRestored` (regained 5) | p. 17 | happy |
| healing capped | 14 of 20 | 8 | 20 | `HitPointsRestored` (regained 6) | p. 17 (worked example) | unhappy (boundary) |
| exactly to maximum | 14 of 20 | 6 | 20 | `HitPointsRestored` (regained 6) | p. 17 | unhappy (boundary) |
| already at maximum | 20 of 20 | 4 | 20 | no `ConditionRemoved`; state unchanged | p. 17 | unhappy (boundary) |
| zero healing | 14 of 20 | 0 | 14 | no `ConditionRemoved`; state unchanged | p. 17 | unhappy (boundary) |
| zero healing at 0, Unconscious | 0 of 20, Dying, Unconscious | 0 | 0, still Dying | no `ConditionRemoved`; `LifeState` unchanged | p. 17; hit-points-and-damage.md FR-037 | unhappy (boundary) |
| wakes at 1 Hit Point | 0 of 20, Dying, Unconscious | 1 | 1 | `HitPointsRestored`, `Recovered`, `ConditionRemoved` (Unconscious); Prone remains (FR-066) | p. 17 | happy |
| Stable but Unconscious, healed | 0 of 20, Stable, Unconscious | 3 | 3 | `HitPointsRestored`, `Recovered`, `ConditionRemoved` (Unconscious) | p. 18, p. 17 | unhappy (rule interaction) |
| knocked out, healed above 0 | 1 of 12, Alive, Unconscious, `knockedOut` | 3 | 4 | `HitPointsRestored`, `ConditionRemoved` (Unconscious); marker cleared (FR-092) | pp. 17, 184 | unhappy (rule interaction) |
| spell's Unconscious, healed above 0 | 5 of 12, Alive, Unconscious, not `knockedOut` | 3 | 8 | `HitPointsRestored`; Unconscious stays | p. 179 | unhappy (rule interaction) |
| negative amount | 14 of 20 | -1 | n/a | an `srderr.ErrInvalidInput` error when the command is built | p. 17 | unhappy (invalid input) |
| healing the dead | dead combatant | 10 | n/a | `TransitionRejected` (`CombatantIsDead`) | p. 180 | unhappy (state) |

### 4.6 Conditions

All condition rules are `ConditionSet`'s (conditions.md). The encounter passes the `CombatantID` as the `target` (shared-types-and-ownership.md FR-012). Outcome mapping: `ConditionAdded`, `ConditionCleared`, `ExhaustionLowered`, `ExhaustionReduced`, `ExhaustionEnded` give a `TransitionApplied` with the wrapped events; `ConditionAlreadyHeld` gives `AlreadyHasCondition`; `ConditionImmune` gives `ImmuneToCondition`; `ConditionRemovalNotPresent` and `ExhaustionReductionNotPresent` give `DoesNotHaveCondition`; `ConditionRemovalImpliedBy` gives `ConditionImpliedBy`; `ExhaustionAtMaximum` has no counterpart (FR-064). Reasons are written without the `Rejection` suffix the exported class carries.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-034 | `ApplyCondition` for a condition the combatant does not have MUST add it and wrap `ConditionApplied` carrying the combatant and the condition. | SRD 5.2.1, "Condition", p. 179 | happy |
| FR-035 | `ApplyCondition` for a condition already held (other than Exhaustion) MUST return `TransitionRejected` with `AlreadyHasCondition` and not change the combatant. | SRD 5.2.1, "Condition", p. 179 | unhappy (rule interaction) |
| FR-036 | `ApplyCondition` for Exhaustion MUST raise the level by 1 (none to 1, or n to n + 1) and wrap `ConditionApplied` carrying the resulting level (conditions.md FR-037, FR-038). | SRD 5.2.1, "Exhaustion [Condition]", p. 181 | unhappy (rule interaction) |
| FR-037 | When Exhaustion reaches level 6 (`isDeadFromExhaustion`, conditions.md FR-040) the encounter MUST apply the hit point rule behind `RecordExhaustionDeath` (hit-points-and-damage.md FR-095), leaving `Dead` with cause `exhaustion`, and wrap the `Died(exhaustion)` it returns after the wrapped `ConditionApplied` (FR-084). The encounter, not the spec that follows it, wires this (conditions.md D-3). The death is recorded on the combatant (FR-068). | SRD 5.2.1, "Exhaustion [Condition]", p. 181 | unhappy (boundary) |
| FR-038 | `ApplyCondition` for a condition the combatant is Immune to MUST return `TransitionRejected` with `ImmuneToCondition` and not change the combatant. | SRD 5.2.1, "Immunity", p. 183; "Stat Block", p. 188 | unhappy (rule interaction) |
| FR-039 | `RemoveCondition` for a condition the combatant has MUST remove it and wrap `ConditionRemoved`. | SRD 5.2.1, "Condition", p. 179 | happy |
| FR-040 | `RemoveCondition` for a condition the combatant does not have MUST return `TransitionRejected` with `DoesNotHaveCondition`. | SRD 5.2.1, "Condition", p. 179 | unhappy (state) |
| FR-041 | `RemoveCondition` for Exhaustion MUST behave as `ReduceExhaustion(1)` (conditions.md FR-036, D-15): at level 2 or above the level drops by 1 and one `ConditionLevelReduced` (new level) is wrapped; at level 1 the condition ends and one `ConditionRemoved` is wrapped. It MUST NOT end a higher level in one step; several levels is `ReduceExhaustion` (FR-063). | SRD 5.2.1, "Exhaustion [Condition]", p. 181 | unhappy (boundary) |
| FR-063 | `ReduceExhaustion(levels)` MUST call `ConditionSet.ReduceExhaustion`: at level n with `levels < n` the level becomes n - levels and `ConditionLevelReduced` (new level) is wrapped; with `levels >= n` the condition ends and one `ConditionRemoved` is wrapped. `levels` below 1 MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when the command is built (conditions.md FR-045). Without Exhaustion it MUST return `TransitionRejected` with `DoesNotHaveCondition`. | SRD 5.2.1, "Exhaustion [Condition]", "Removing Exhaustion Levels", p. 181 | unhappy (boundary) |
| FR-064 | Level 6 is death (FR-037) and leaves the status `Dead`, so the dead rejection of FR-078 applies first: `ApplyCondition` (any condition) and `ReduceExhaustion` for it return `CombatantIsDead`. `ExhaustionAtMaximum` is unreachable through the encounter and has no rejection reason. | SRD 5.2.1, "Exhaustion [Condition]", p. 181; conditions.md FR-039, FR-040 | unhappy (boundary) |
| FR-065 | `RemoveCondition` for a kind held only because another explicit condition implies it (Incapacitated or Prone while Unconscious; Incapacitated while Paralyzed) MUST return `TransitionRejected` with `ConditionImpliedBy` naming the implying kinds, and not change the combatant. | SRD 5.2.1, "Unconscious", p. 191; "Paralyzed", p. 186; conditions.md FR-033 | unhappy (state) |
| FR-066 | Applying Unconscious MUST make Incapacitated and Prone effective; removing Unconscious (by `RemoveCondition` or FR-031) MUST end the implied Incapacitated (when nothing else implies it), leave the combatant Prone as an explicit condition, and not produce `ConditionApplied` for it (conditions.md FR-026, FR-029). | SRD 5.2.1, "Unconscious", "Inert", p. 191 | unhappy (rule interaction) |
| FR-067 | When the encounter applies Unconscious as a consequence of a hit point result (FR-023, FR-070) and `ConditionSet.Apply` returns anything other than `ConditionAdded` (`ConditionAlreadyHeld`, FR-027, or `ConditionImmune`), it MUST produce no `ConditionApplied`, keep the hit point status the rules returned, and not reject the damage command. | SRD 5.2.1, "Immunity", p. 17; "Condition", p. 179 | unhappy (rule interaction) |

#### 4.6.1 Exhaustion and implied-condition cases (FR-036, FR-037, FR-041, FR-063 to FR-066)

| Label | Level before | Command | Level after | Events | Source | Path |
|---|---|---|---|---|---|---|
| first level | none | apply | 1 | `ConditionApplied` (level 1) | p. 181 | happy |
| raise from 1 | 1 | apply | 2 | `ConditionApplied` (level 2) | p. 181 | unhappy (rule interaction) |
| level 5 to 6 | 5 | apply | 6 (dead) | `ConditionApplied` (level 6), `Died` (`exhaustion`) | p. 181 | unhappy (boundary) |
| apply at 6 | 6 (dead) | apply | 6 | `TransitionRejected` (`CombatantIsDead`) | p. 181 | unhappy (boundary) |
| remove lowers one level | 3 | remove | 2 | `ConditionLevelReduced` (level 2); not `ConditionRemoved` | p. 181 | unhappy (boundary) |
| remove at level 1 | 1 | remove | none | `ConditionRemoved` | p. 181 | unhappy (boundary) |
| reduce from 2 by 1 | 2 | reduce 1 | 1 | `ConditionLevelReduced` (level 1) | p. 181 | unhappy (boundary) |
| reduce from 1 by 1 | 1 | reduce 1 | none | `ConditionRemoved` | p. 181 | unhappy (boundary) |
| reduce by more than the level | 2 | reduce 3 | none | `ConditionRemoved` | p. 181 | unhappy (boundary) |
| reduce with none | none | reduce 1 | none | `TransitionRejected` (`DoesNotHaveCondition`) | p. 181 | unhappy (state) |
| remove with none | none | remove | none | `TransitionRejected` (`DoesNotHaveCondition`) | p. 181 | unhappy (state) |
| reduce by 0 | 2 | reduce 0 | n/a | an `srderr.ErrInvalidInput` error when the command is built | p. 181 | unhappy (invalid input) |
| remove Prone while Unconscious | Unconscious held | remove Prone | n/a | `TransitionRejected` (`ConditionImpliedBy`, [Unconscious]) | p. 191 | unhappy (state) |
| remove Unconscious | Unconscious held | remove Unconscious | n/a | `ConditionRemoved` (Unconscious); Prone explicit and effective; no `ConditionApplied` | p. 191 | unhappy (rule interaction) |

### 4.7 Ending combat

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-042 | `end` with `EndReasonSideDefeated(side)` MUST set the status to `EncounterEndedStatus` with that reason and produce `EncounterEnded` (encounter) carrying it. `side` is the side defeated, not the winner. `end` MUST NOT verify the claim (FR-044): `EndReasonSideDefeated(sideA)` succeeds while every `sideA` combatant is alive. | SRD 5.2.1, "Ending Combat", p. 14 | happy |
| FR-043 | `end` with `EndReasonMutualAgreement` MUST set the status to `EncounterEndedStatus` with that reason and produce `EncounterEnded` (encounter) carrying it. | SRD 5.2.1, "Ending Combat", p. 14 | happy |
| FR-044 | The encounter MUST NOT end by itself: when every combatant of a side is dead or at 0 Hit Points the status stays `EncounterInProgress` and no `EncounterEnded` is produced, because "defeated" can also mean surrendered or fled, states the engine does not hold. | SRD 5.2.1, "Ending Combat", p. 14 | unhappy (rule interaction) |
| FR-045 | Once ended, every command (`AdvanceTurn`, `ApplyTurnChange`, `ApplyDamage`, `ApplyHealing`, `ApplyCondition`, `RemoveCondition`, `ReduceExhaustion`, `ApplyDeathSavingThrow`, `Stabilize`, `RecoverStable`, `GainTemporaryHitPoints`, `ReduceHitPointMaximum`, `CheckGrapple`, `ReleaseGrapple`, `EscapeGrapple`, and `end` itself) MUST return `TransitionRejected` with `EncounterAlreadyEnded`; the three roster methods return their `RosterEncounterEndedRejection` (combatant-roster-changes.md FR-009). | SRD 5.2.1, "Ending Combat", p. 14 | unhappy (state) |

### 4.8 Hit point operations other than damage and healing

Each command calls the matching pure operation of `HitPointStatus` named in hit-points-and-damage.md FR-109 (`applyDeathSave`, `Stabilize`, `RecoverStable`, `gainTemporary`, `reduceMaximum`, `recordExhaustionDeath`; `ApplyDamage` and `restore` for 4.4 and 4.5; each returns a `HitPointTransition`) with the `CombatantID` as `subject`, never the use case of the same role (shared-types-and-ownership.md FR-044); none rolls dice. A refusal (`HitPointRejection`) is returned as the `HitPointRule` rejection holding it (section 5), never re-declared.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-073 | `ApplyDeathSavingThrow(combatantID, natural, total)` MUST forward both to `HitPointStatus.ApplyDeathSave(subject, natural, total)` (hit-points-and-damage.md FR-053 to FR-061) and wrap the events returned. `natural` outside 1 to 20 MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when the command is built; `total` is any `int` (Exhaustion and Advantage already applied by the host, hit-points-and-damage.md Q-4): success iff `natural` is not 1 or 20 and `total` >= 10; a natural 1 is two failures and a natural 20 regains 1 Hit Point, whatever the total. A combatant that is not `Dying` MUST return `TransitionRejected` with `HitPointRule` carrying `NotDying`; one that already threw this turn, `DeathSavingThrowAlreadyMade` (FR-099). The third failure yields the wrapped `Died` (`death_saving_throw_failures`); a natural 20 (1 Hit Point) or the third success (`Stable`) MUST NOT. A natural 20 MUST remove Unconscious as in FR-031. | SRD 5.2.1, "Death Saving Throws", p. 17; "Rolling a 1 or 20", p. 18 | happy |
| FR-074 | `Stabilize(combatantID)` MUST make a `Dying` combatant `Stable` with both tallies 0 (hit-points-and-damage.md FR-067) and wrap the events. The Help-action check's success is the host's input. A combatant not `Dying` MUST return `HitPointRule` carrying `NotDying`. It stays Unconscious. | SRD 5.2.1, "Stabilizing a Character", p. 18 | happy |
| FR-075 | `RecoverStable(combatantID)` MUST set a `Stable` combatant to 1 Hit Point and `Alive` (hit-points-and-damage.md FR-073), remove Unconscious as in FR-031, and wrap the events. The host decides when the 1d4 hours have passed. A combatant not `Stable` MUST return `HitPointRule` carrying `NotStable`. | SRD 5.2.1, "Stabilizing a Character", p. 18 | unhappy (state) |
| FR-076 | `GainTemporaryHitPoints(combatantID, amount, choice)` MUST apply hit-points-and-damage.md FR-030 to FR-033 (amount below 1 returns an `srderr.ErrInvalidInput` error when the command is built; no effect on conditions or `LifeState`; a `TemporaryHitPointsChoice` decides keep or accept) and wrap the events. | SRD 5.2.1, "Temporary Hit Points", p. 18 | happy |
| FR-077 | `ReduceHitPointMaximum(combatantID, amount)` MUST apply hit-points-and-damage.md FR-046, FR-047 (a negative amount returns an `srderr.ErrInvalidInput` error when the command is built) and wrap the events. At maximum 0 the status is `Dead(hitPointMaximumZero)` and the events include the wrapped `Died` (`hit_point_maximum_zero`) after `HitPointMaximumReduced`. | SRD 5.2.1, "Hit Point Maximum of 0", p. 17 | unhappy (boundary) |
| FR-099 | `Combatant.DeathSavingThrowMade` (default false) MUST be set by every accepted `ApplyDeathSavingThrow` (any natural) and cleared for the combatant whose turn started, at `Start` and at `AdvanceTurn`, when the `TrackerTurnStarted` is produced and before FR-018; no event. A second `ApplyDeathSavingThrow` while it is set MUST return `TransitionRejected` with `DeathSavingThrowAlreadyMade` (exported `DeathSavingThrowAlreadyMadeRejection(id)`; result `death_saving_throw_already_made`, class `refused`), ranked after the hit point rule's `NotDying` within the command (FR-078 step 4). Reading: the throw is made "whenever you start your turn" (p. 17), so one per turn start; a Dying combatant that drops again the same turn waits for the next. The encounter does not check whose turn it is (combat-action-resolution.md FR-010 5a does). | SRD 5.2.1, "Death Saving Throws", p. 17 (interpretation, SRD silent on a repeat); combat-action-resolution.md Q-4 | unhappy (state) |
| FR-100 | `Encounter.PrecheckDeathSavingThrow(combatantID)` MUST return the `*EncounterRejection` (nil for none) that `ApplyDeathSavingThrow` would give before any value is applied (`EncounterAlreadyEnded`, `UnknownCombatant`, `CombatantIsDead`, `HitPointRule(NotDying)`, `DeathSavingThrowAlreadyMade`, in that order) and null when a throw would be accepted. It changes nothing, emits nothing and has no use case; combat-action-resolution.md calls it before any die is rolled (its FR-068). | combat-action-resolution.md FR-010, FR-068; n/a (architecture §3) | unhappy (state) |

#### 4.8.1 Death Saving Throw, stabilizing and recovery cases (FR-073 to FR-077)

| Label | Combatant before | Command | After | Events | Source | Path |
|---|---|---|---|---|---|---|
| success | 0 Hit Points, Dying, 0 and 0 | save, natural 12, total 12 | Dying, 1 success; flag set | `DeathSavingThrowMade` | p. 17 | happy |
| natural 1 | Dying, 0 and 1 | save, natural 1, total 12 | Dead (3 failures) | `DeathSavingThrowMade`, `Died` (`death_saving_throw_failures`) | p. 18 | unhappy (boundary) |
| natural 20 | Dying, 1 failure | save, natural 20, total 4 | 1 Hit Point, Alive | `DeathSavingThrowMade`, `HitPointsRestored`, `Recovered`, `ConditionRemoved` (Unconscious) | p. 18 | unhappy (boundary) |
| third success | Dying, 2 successes | save, natural 10, total 10 | Stable | `DeathSavingThrowMade`, `BecameStable`; Unconscious kept | p. 17 | unhappy (boundary) |
| Exhaustion lowers the total | Dying, 0 and 0 | save, natural 11, total 7 | Dying, 1 failure | `DeathSavingThrowMade` | p. 17, p. 181 | unhappy (rule interaction) |
| second throw in one turn | Dying, flag set | save, natural 15, total 15 | n/a | `TransitionRejected` (`DeathSavingThrowAlreadyMade`); precheck gives the same | p. 17 | unhappy (state) |
| next turn clears the flag | Dying, flag set | `AdvanceTurn` round, then save | accepted | flag false after `TrackerTurnStarted`, set again | p. 17 | unhappy (rule interaction) |
| natural out of range | Dying | save, natural 21 | n/a | an `srderr.ErrInvalidInput` error when the command is built | p. 17 | unhappy (invalid input) |
| save while Alive | 8 of 20 | save, natural 15, total 15 | n/a | `TransitionRejected` (`HitPointRule`, `NotDying`) | p. 17 | unhappy (state) |
| stabilize a Dying character | Dying | stabilize | Stable | `BecameStable` | p. 18 | happy |
| stabilize a Stable / recover a Dying character | Stable; Dying | stabilize; recover | n/a | `TransitionRejected` (`HitPointRule`: `NotDying`; `NotStable`) | p. 18 | unhappy (state) |
| recover a Stable character | 0 of 20, Stable, Unconscious | recover | 1, Alive | `HitPointsRestored`, `Recovered`, `ConditionRemoved` (Unconscious) | p. 18 | happy |
| temporary Hit Points at 0 | 0 of 20, Dying | gain 5 | Dying, 5 temporary | `TemporaryHitPointsGained`; `LifeState` and Unconscious unchanged | p. 18 | unhappy (rule interaction) |
| maximum reduced to 0 | 5 of 5 | reduce by 5 | Dead | `HitPointMaximumReduced`, `Died` (`hit_point_maximum_zero`) | p. 17 | unhappy (boundary) |

### 4.9 Every transition

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-046 | A command that names a `CombatantID` not in the encounter MUST return `TransitionRejected` with `UnknownCombatant` carrying that id. The id is untrusted host input, so the use case classifies it `invalid` (FR-059), not `refused`; the result is still a `...Rejected` holding the rejection, not an exception. The same holds for an unknown id inside a command's other inputs (FR-085, FR-087). | n/a (architecture §6; observability §6, §7) | unhappy (invalid input) |
| FR-047 | Every command that names a combatant MUST change only that combatant; the other combatants, the order, the round, the active combatant and the status stay equal. | n/a (architecture §3, aggregate root) | unhappy (rule interaction) |
| FR-048 | A `TransitionRejected` MUST carry the unchanged encounter and no events. | n/a (architecture §3, §6) | unhappy (state) |
| FR-049 | Every transition MUST return a new `Encounter` and leave the receiver unchanged. | n/a (architecture §3) | unhappy (immutability) |
| FR-050 | `EncounterEvent` MUST be a closed sub-family whose members satisfy the `Event` interface (combat-foundations.md FR-015, FR-016, D-1: `Event` is an interface with an unexported marker method, so closure is the closed sub-family plus the architecture test there; shared-types-and-ownership.md FR-019). Every member MUST be an immutable value (unexported fields, value receivers), compared by value including `kind` and `combatants` (combat-foundations.md FR-019). The events of a transition MUST be a slice (copied by accessors) in the order they happened. Every member MUST be named in the past tense and state a fact (shared-types-and-ownership.md FR-019; combat-foundations.md FR-020): no name ends in `Due`, `Required` or `Needed`, which is why FR-018's event is `DyingCombatantTurnStarted`. Only the four transport wrappers (`TurnEventOccurred`, `HitPointEventOccurred`, `ConditionEventOccurred`, `GrappleEventOccurred`) end in `Occurred` (combat-foundations.md FR-020, FR-038); the four wrappers MUST satisfy `TransportEnvelope` (combat-foundations.md FR-038); an architecture test checks the suffixes. | n/a (architecture §3, §4); combat-foundations.md FR-015 to FR-021 | unhappy (immutability) |
| FR-051 | Every event MUST carry the `EncounterID` and the round in which it happened (the new round for `TrackerRoundStarted`, `TrackerTurnStarted` and what follows at a wrap; the old round for the preceding `TrackerTurnEnded`); events about one combatant MUST carry its `CombatantID`. Every `EncounterEvent` MUST define `kind` (constant unique `snake_case`, combat-foundations.md FR-017) and `combatants` (unmodifiable, duplicate-free, FR-018 there): `EncounterStarted` is `encounter_started` with all ids in turn order; `EncounterEnded` is `encounter_ended` with the same ids; `DyingCombatantTurnStarted` is `dying_combatant_turn_started` with `[active combatant]`. A wrapper has no constant of its own and delegates both to the wrapped event, so `srd5e.combat.events` counts the wrapped kind; the uniqueness scan of combat-foundations.md FR-017 exempts the four wrappers (FR-038 there). | n/a (architecture §4, handoffs); combat-foundations.md FR-015 to FR-019 | happy |
| FR-052 | The combatant list MUST be returned as a copy in the tracker's turn order (FR-007); mutating the returned slice MUST NOT change the encounter. | n/a (architecture §4, handoffs) | unhappy (immutability) |
| FR-053 | Two encounters with the same id, status, tracker and combatants MUST be equal under `Encounter.Equal` (the type holds a slice, so it is not comparable with `==`); any difference MUST make them unequal. Because combatants are stored in turn order (FR-007), two starts differing only in the input order are equal. Value equality for an entity-like root departs from architecture §3 on purpose: a transition returns a new instance with the same id, so tests and hosts must tell states apart. Identity remains the id. | n/a (architecture §3, value equality) | happy |
| FR-054 | The same encounter and the same command MUST give equal transitions on every call. The domain MUST NOT read a clock or random source. | n/a (architecture §3, purity) | unhappy (determinism) |
| FR-068 | Every wrapped `Died` MUST correspond to a combatant whose `death` (a `DeathCause` plus `ok`, hit-points-and-damage.md) is present afterwards, and a combatant whose `Death()` is present MUST NOT lose it in a later transition (a roster removal takes the combatant away whole, combatant-roster-changes.md FR-018). `death` is derived from the status alone: `cause` when `LifeState` is `Dead(cause)`, otherwise null. A death by Exhaustion is `Dead(exhaustion)` (FR-037), so no second source of death exists. | SRD 5.2.1, "Dead", p. 180; "Exhaustion [Condition]", p. 181 | unhappy (state) |
| FR-069 | After a combatant died of Exhaustion (level 6), `ApplyDamage` and `ApplyHealing` for it MUST return `CombatantIsDead`, because its `LifeState` is `Dead(exhaustion)` (FR-029, FR-033). | SRD 5.2.1, "Dead", p. 180; "Exhaustion [Condition]", p. 181 | unhappy (state) |
| FR-078 | When several rejections could apply, the encounter MUST pick the first in this order: (1) `EncounterAlreadyEnded`; (2) `UnknownCombatant` (the named combatant, an id in the command's other inputs, or the grappler id in the named combatant's Grappled condition, FR-087; with several unknown ids, the smallest `CombatantID.Value` in code unit order, so the choice does not depend on how the command was built); (3) `CombatantIsDead`; (4) the command's own reason, in the order of table 4.9.1 within one command. Invalid inputs (an `srderr.ErrInvalidInput` error) are rejected when the command is built, before any of this. Because (3) comes first, the hit point rule's `CreatureIsDead` is unreachable here. The SRD gives no order; Immune is checked before already-has, relying on conditions.md FR-013, FR-021 (an immune kind is never effective). | n/a (architecture §6; design decision, this row) | unhappy (rule interaction) |
| FR-079 | Every command that names a combatant (`ApplyDamage`, `ApplyHealing`, `ApplyCondition`, `RemoveCondition`, `ReduceExhaustion`, `ApplyDeathSavingThrow`, `Stabilize`, `RecoverStable`, `GainTemporaryHitPoints`, `ReduceHitPointMaximum`, `CheckGrapple`, `ReleaseGrapple`, `EscapeGrapple`) for a dead combatant (including death by Exhaustion, FR-064) MUST return `CombatantIsDead` and not change it, as in table 4.10.1. For a grapple command the dead combatant is the named (grappled) one; a dead grappler does not matter (FR-086). The SRD gives no rule for conditions on a dead creature; the encounter refuses, as for damage and healing. `AdvanceTurn`, `ApplyTurnChange` and `end` name no combatant and are not affected. | SRD 5.2.1, "Dead", p. 180; design decision (this row) | unhappy (state) |
| FR-082 | Two `Combatant`s with the same `CombatantRef` (id and kind), side, `HitPointStatus`, `ConditionSet`, `knockedOut` and `deathSavingThrowMade` MUST be equal under `Combatant.Equal`; a difference in any of them (the kind with the id equal) MUST make them unequal (seven cases over six fields: id, kind, side, `HitPointStatus`, `ConditionSet`, `knockedOut`, `deathSavingThrowMade`). | n/a (architecture §3, value equality) | happy |
| FR-083 | The `Combatant` constructor adds no invariant: it MUST NOT panic for any valid `CombatantRef`, `HitPointStatus`, `ConditionSet` and markers, each part validates itself (hit-points-and-damage.md FR-002; conditions.md FR-012). Consistency between a `LifeState`, the markers and the conditions (a `Dying` combatant without Unconscious; `knockedOut` without it) is the host's input and not checked. | n/a (architecture §3) | unhappy (invalid input) |
| FR-084 | Facts a sibling spec already defines MUST be carried by wrapping, never by a second event of the same fact. The encounter wraps tracker events in `TurnEventOccurred`, hit point events (all of hit-points-and-damage.md's, including `Died`) in `HitPointEventOccurred(combatantID, event)`, conditions events (`ConditionApplied`, `ConditionRemoved`, `ConditionLevelReduced`, conditions.md FR-139) in `ConditionEventOccurred(combatantID, event)` and `GrappleEnded` (grappling.md FR-081) in `GrappleEventOccurred(combatantID, event)`, each also carrying the `EncounterID` and the round. There is no encounter death event: a death is the one `Died` the hit point rules return, reported once (shared-types-and-ownership.md FR-019). The constructors of `HitPointEventOccurred` and `ConditionEventOccurred` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when the wrapped event's `combatants` is not `[combatantID]`. `GrappleEnded` is the one event about two creatures: `GrappleEventOccurred` takes the grappled combatant and MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` unless it is the first of the event's `combatants`. | shared-types-and-ownership.md FR-019; n/a (architecture §3, §4) | happy |
| FR-085 | `AdvanceTurn` MUST pass the command's optional updated `MovementSpeeds` and `sharedSpaceOccupants` (the `CreatureSize`s of other creatures in the ending combatant's space; empty when none) to the tracker's advance operation unchanged (initiative-and-actions.md FR-042, FR-043). It MUST first check that every `CombatantID` in the updated speeds is a combatant, and return `UnknownCombatant` otherwise (FR-046, FR-078), so an untrusted id never reaches the tracker and its own unknown-id rule cannot occur here. It MUST NOT apply Prone for `FellProneInSharedSpace`, only wrap the event: the host applies `ApplyCondition` (Prone), because the initiative spec leaves it to conditions and applying it needs the other creature's size. | SRD 5.2.1, "Moving around Other Creatures", p. 14 | unhappy (rule interaction) |

#### 4.9.1 Rejection precedence cases (FR-078, FR-079)

One row, one test. The first reason wins.

| Label | State and command | Reasons that apply | Expected | Source | Path |
|---|---|---|---|---|---|
| ended and unknown | ended encounter, damage for an unknown id | `EncounterAlreadyEnded`, `UnknownCombatant` | `EncounterAlreadyEnded` | p. 14 | unhappy (rule interaction) |
| ended and dead | ended encounter, damage for a dead combatant | `EncounterAlreadyEnded`, `CombatantIsDead` | `EncounterAlreadyEnded` | p. 14, p. 180 | unhappy (rule interaction) |
| unknown id, another combatant dead | healing for an unknown id while a different combatant is dead | `UnknownCombatant` | `UnknownCombatant` (class `invalid`) | n/a | unhappy (rule interaction) |
| unknown id in the speeds update | `AdvanceTurn` with updated speeds for an unknown id | `UnknownCombatant` | `UnknownCombatant` (class `invalid`); tracker not called | p. 188 | unhappy (invalid input) |
| two unknown ids in the speeds update | updates for unknown `z` and `b`, built in either order | `UnknownCombatant` | `UnknownCombatant` carrying `b`, whatever the order | n/a | unhappy (rule interaction) |
| dead and immune | condition command: dead combatant also immune | `CombatantIsDead`, `ImmuneToCondition` | `CombatantIsDead` | p. 180, p. 17 | unhappy (rule interaction) |
| immune and already has | holds Poisoned (suppressed by Immunity), gets Poisoned | `ImmuneToCondition`, `AlreadyHasCondition` | `ImmuneToCondition` | p. 17, p. 179 | unhappy (rule interaction) |
| dead and not dying | death saving throw for a dead combatant | `CombatantIsDead`, `HitPointRule` (`NotDying`) | `CombatantIsDead` | p. 180 | unhappy (rule interaction) |
| dead of Exhaustion, condition | condition for a combatant at Exhaustion 6 | `CombatantIsDead` (`ExhaustionAtMaximum` is not a rejection here) | `CombatantIsDead` | p. 181 | unhappy (boundary) |
| end or advance on ended | `end` or `AdvanceTurn` on an ended encounter | `EncounterAlreadyEnded` | `EncounterAlreadyEnded` | p. 14 | unhappy (state) |
| remove on dead | `RemoveCondition` for a dead combatant | `CombatantIsDead`, `DoesNotHaveCondition` | `CombatantIsDead` | p. 180 | unhappy (state) |
| grapple on dead | `CheckGrapple`, `ReleaseGrapple` or `EscapeGrapple` for a dead combatant holding Grappled | `CombatantIsDead` | `CombatantIsDead`; grapple not ended | p. 180, p. 182 | unhappy (state) |
| grapple on dead, not Grappled | `ReleaseGrapple` for a dead combatant without Grappled | `CombatantIsDead`, `DoesNotHaveCondition` | `CombatantIsDead` | p. 180 | unhappy (rule interaction) |
| dead grappled, unknown grappler | `CheckGrapple` for a dead combatant whose Grappled names a grappler the encounter does not hold | `UnknownCombatant` (grappler id, step 2), `CombatantIsDead` | `UnknownCombatant` (class `invalid`) | p. 180, p. 182 | unhappy (rule interaction) |

### 4.10 Use cases and telemetry

Every signal is recorded through `CombatTelemetry` (combat-foundations.md FR-027 to FR-037). This spec adds one signal (FR-062); the span, throughput counter, duration histogram and event counter are combat-foundations.md's (FR-029); the death counter is hit-points-and-damage.md's section 7, used here (FR-081).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-055 | Each command has one synchronous use case (one per command, architecture §4; hosts wanting one entry point compose them) that takes a command carrying the current `Encounter` (for start, the id, the established order and the combatants) and returns its own closed `<UseCase>Result` (FR-089), holding no state between calls. | n/a (architecture §4) | happy |
| FR-056 | Every use case MUST return equal results with the default zero-value `telemetry.Noop` (shared-types-and-ownership.md FR-036) and with the recording double from `telemetry/telemetrytest`. | n/a (observability §9); combat-foundations.md FR-036 | happy |
| FR-057 | Every use case MUST run through `CombatTelemetry` (combat-foundations.md FR-035) with its `CombatOperation`, the `snake_case` use case name (table 4.10.1). It then records once per call the span `srd5e.combat.<operation>`, counter `srd5e.combat.operations`, histogram `srd5e.combat.operation_duration` and one `srd5e.combat.events` per event, with the shared attributes `srd5e.combat.operation`, `srd5e.outcome`, `srd5e.combat.result`, `srd5e.combat.result_class`. This spec defines none of these and no span of its own; no attribute carries a `CombatantID`, `EncounterID`, amount or free text (combat-foundations.md FR-033). | combat-foundations.md FR-029, FR-030, FR-033, FR-035; observability §5, §6 | happy |
| FR-058 | A `...Rejected` result whose `EncounterRejection` is anything except `UnknownCombatant` MUST be classified `refused` (`CombatResultClass.Refused`): `srd5e.outcome` `failure`, no span error status, no `srd5e.combat.errors` increment, no `error.type`, no log. `srd5e.combat.result` is the rejection name in `snake_case` (table 4.10.1). An `...Applied` result is `applied` (`success`). | combat-foundations.md FR-030, FR-031, FR-032, FR-034; errors-and-results.md FR-043; observability §6, §7 | unhappy (state) |
| FR-059 | A `StartEncounterInvalid` result (`invalid_encounter`) and a `...Rejected` result with `UnknownCombatant` (`unknown_combatant`) MUST be classified `invalid`: `failure`, span error status set, no `srd5e.combat.errors` increment, no `error.type` (returned failures, not `*srderr.PortError`), nothing logged. | combat-foundations.md FR-031, FR-032, FR-034; observability §6, §7 | unhappy (invalid input) |
| FR-060 | `srd5e.combat.events` MUST count one event per event the use case returns, with `srd5e.combat.event` equal to the event's `kind`; a wrapped event counts under the wrapped kind (FR-051); a rejection counts none. | combat-foundations.md FR-029; observability §5 | happy |
| FR-061 | A panic raised inside a telemetry backend (or any returned non-port error) MUST propagate unchanged out of the use case, the span recording it (the deferred `span.End()` still runs) (combat-foundations.md FR-035). The encounter calls no port, so it returns no `*srderr.PortError` and never increments `srd5e.combat.errors`. The only sources are a panicking recording telemetry double double and a domain call rejecting an input as a programmer error; the propagation rule is tested in combat-foundations.md FR-035's suite. | combat-foundations.md FR-032, FR-035; observability §7 | unhappy (failure) |
| FR-062 | `StartEncounter` MUST record the number of combatants in the histogram `srd5e.combat.encounter_size` when the start succeeds, and MUST NOT record it when invalid. | n/a (observability §5, "Sizes") | unhappy (boundary) |
| FR-081 | Every use case whose result holds a wrapped `Died` MUST increment `srd5e.combat.deaths` once per `Died`, with `srd5e.combat.death_cause` from the event's `DeathCause` (`monster_death`, `hit_point_maximum_zero`, `massive_damage`, `death_saving_throw_failures`, `exhaustion`). It is the one package death counter, defined by hit-points-and-damage.md; this spec defines no second one. The encounter use cases do not call the hit point use cases, so no death is counted twice. | hit-points-and-damage.md section 7; observability §5 | happy |
| FR-089 | Every use case MUST return its own closed `<UseCase>Result` (`StartEncounterResult`, `AdvanceEncounterTurnResult`, and so on, section 6), never a shared type, with variants prefixed by the use case (`<UseCase>Applied`, `<UseCase>Rejected`; `StartEncounterInvalid` for invalid start input). A rejected variant MUST hold the rejection as a field (an `EncounterRejection`, shared-types-and-ownership.md FR-018), not as a variant. `EncounterTransition`, `TransitionApplied`, `TransitionRejected`, `EncounterStart`, `Started` and `StartEncounterInvalid` MUST NOT be exported. | shared-types-and-ownership.md FR-017, FR-018; architecture §4 | unhappy (name collision) |
| FR-090 | Every rejection reason and start problem MUST be an exported type whose name ends in `Rejection` or `Problem` (`CombatantIsDeadRejection`, `HitPointRuleRejection`, `NoCombatantsProblem`), so none collides with a sibling type in package `combat` (hit-points-and-damage.md exports `NotDying`, `NotStable`, `CreatureIsDead`). `EncounterStatus` and `EndReason` variants are prefixed with their type (`EncounterInProgress`, `EncounterEndedStatus`, `EndReasonSideDefeated`, `EndReasonMutualAgreement`); generic names (`InProgress`, `Ended`, `SideDefeated`) are never exported. Prose writes a reason without the suffix. | shared-types-and-ownership.md FR-017, FR-021; combat-foundations.md FR-022, FR-024 | unhappy (name collision) |
| FR-095 | The `CombatOperation`, `srd5e.combat.result` names and `CombatResultClass` of every use case MUST be exactly those of table 4.10.1. `applied` is `applied`; `refused` names are the rejection in `snake_case`; `invalid` names are `unknown_combatant` and `invalid_encounter`. The `HitPointRule` rejection is reported as the carried reason (`not_dying`, `not_stable`). The result names are closed sets; the operation names repeat no operation of hit-points-and-damage.md, conditions.md or initiative-and-actions.md. | combat-foundations.md FR-024, FR-028, FR-030, FR-031 | unhappy (rule interaction) |

#### 4.10.1 Operations, results and classes (FR-057 to FR-059, FR-095)

`encounter_already_ended` and `combatant_is_dead` are `refused` wherever they appear; `unknown_combatant` is `invalid` wherever it appears. Every use case also has the result `applied` (class `applied`).

| Use case | `srd5e.combat.operation` | Refused results | Invalid results |
|---|---|---|---|
| `StartEncounter` | `start_encounter` | none | `invalid_encounter` |
| `AdvanceEncounterTurn` | `advance_encounter_turn` | `encounter_already_ended` | `unknown_combatant` (speeds update) |
| `ApplyDamageToCombatant` | `apply_damage_to_combatant` | `encounter_already_ended`, `combatant_is_dead` | `unknown_combatant` |
| `ApplyHealingToCombatant` | `apply_healing_to_combatant` | `encounter_already_ended`, `combatant_is_dead` | `unknown_combatant` |
| `ApplyConditionToCombatant` | `apply_condition_to_combatant` | `encounter_already_ended`, `combatant_is_dead`, `already_has_condition`, `immune_to_condition` | `unknown_combatant` |
| `RemoveConditionFromCombatant` | `remove_condition_from_combatant` | `encounter_already_ended`, `combatant_is_dead`, `does_not_have_condition`, `condition_implied_by` | `unknown_combatant` |
| `ReduceCombatantExhaustion` | `reduce_combatant_exhaustion` | `encounter_already_ended`, `combatant_is_dead`, `does_not_have_condition` | `unknown_combatant` |
| `ApplyDeathSavingThrowToCombatant` | `apply_death_saving_throw_to_combatant` | `encounter_already_ended`, `combatant_is_dead`, `not_dying`, `death_saving_throw_already_made` | `unknown_combatant` |
| `StabilizeCombatant` | `stabilize_combatant` | `encounter_already_ended`, `combatant_is_dead`, `not_dying` | `unknown_combatant` |
| `RecoverStableCombatant` | `recover_stable_combatant` | `encounter_already_ended`, `combatant_is_dead`, `not_stable` | `unknown_combatant` |
| `GiveCombatantTemporaryHitPoints` | `give_combatant_temporary_hit_points` | `encounter_already_ended`, `combatant_is_dead` | `unknown_combatant` |
| `ReduceCombatantHitPointMaximum` | `reduce_combatant_hit_point_maximum` | `encounter_already_ended`, `combatant_is_dead` | `unknown_combatant` |
| `EndEncounter` | `end_encounter` | `encounter_already_ended` | none |
| `CheckCombatantGrapple` | `check_combatant_grapple` | `encounter_already_ended`, `combatant_is_dead`, `does_not_have_condition` | `unknown_combatant` (grappled or grappler) |
| `ReleaseCombatantGrapple` | `release_combatant_grapple` | `encounter_already_ended`, `combatant_is_dead`, `does_not_have_condition` | `unknown_combatant` |
| `EscapeCombatantGrapple` | `escape_combatant_grapple` | `encounter_already_ended`, `combatant_is_dead`, `does_not_have_condition` | `unknown_combatant` |

### 4.11 Ending a grapple

The grapple rules are grappling.md's, the Unarmed Strike grapple option attacks.md's (shared-types-and-ownership.md FR-029, FR-048); conditions.md keeps only the Grappled entry and its plain `remove` (FR-031, FR-060 to FR-062 there). What only the encounter can do is hold both creatures: read the grappler's `ConditionSet`, change the grappled creature's, and wrap the events. It keeps no positions, so distance and range come from the host.

- **`GrappleRules.Hold(grapplerConditions, distanceFeet, rangeFeet)`** (grappling.md FR-087): decides whether the hold has ended and with which `GrappleEndCause` (`grapplerIncapacitated` or `outOfRange`, as the matching `GrappleHold`); otherwise `GrappleHold.Holds`. When both apply it returns `grapplerIncapacitated` (FR-088 there).
- **`GrappleRules.EndOnSet(set, target, cause)`** (grappling.md FR-089, FR-090): ends Grappled on the grappled creature's set and returns `GrappleEndOnSetApplied` (new set, `ConditionRemoved` then `GrappleEnded`), or `GrappleEndOnSetNotGrappled` with no event. `GrappleEndCause` is FR-085 there; `GrappleEnded` is FR-086, in the closed `GrappleEvent` (FR-081).

The encounter never ends a grapple with `ConditionSet.Remove`: that emits `ConditionRemoved` only (conditions.md FR-031), while `endOnSet` also emits `GrappleEnded` with the cause (grappling.md FR-091).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-086 | `CheckGrapple(grappledID, distanceFeet, grappleRangeFeet)` for a combatant holding `Grappled(grappler)` MUST call `GrappleRules.Hold` with the grappler's `ConditionSet` as held in this encounter and the command's distance and range. When it returns `grapplerIncapacitated` or `outOfRange` the encounter MUST call `GrappleRules.EndOnSet(set, grappledID, cause)` on the grappled combatant's set with the matching cause, store the returned set and wrap `ConditionRemoved` (Grappled) then `GrappleEnded`, in that order. When it returns `holds` the transition MUST be `TransitionApplied` with no events and an equal encounter. `distanceFeet` and `grappleRangeFeet` MUST be non-negative `int`s; a negative value MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when the command is built. Which cause wins when both apply is `hold`'s, not decided here. `GrappleEndOnSetNotGrappled` is unreachable (Grappled was read from the same set); if it occurs it maps to `DoesNotHaveCondition`. `CheckGrapple` reads only the grappler's `ConditionSet`, so a dead grappler's hold is not ended by the check (the host calls `ReleaseGrapple`); the SRD says nothing of a dead grappler (p. 182 names only Incapacitated and distance). | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182 | happy |
| FR-087 | `CheckGrapple` for a combatant not holding Grappled MUST return `TransitionRejected` with `DoesNotHaveCondition`; when the named grappler is not a combatant it MUST return `UnknownCombatant` carrying the grappler's id (class `invalid`, FR-046). The grappler id read from the Grappled condition sits in step (2) of FR-078: looked up after `EncounterAlreadyEnded` and before `CombatantIsDead`, so a dead grappled creature whose grappler is unknown returns `UnknownCombatant` (table 4.9.1). | SRD 5.2.1, "Grappled", p. 182; n/a (architecture §6) | unhappy (state) |
| FR-088 | The encounter MUST NOT end a grapple on its own: no other command (`ApplyCondition`, `ApplyDamage`, `AdvanceTurn` and the rest) checks or removes Grappled; the host calls `CheckGrapple`, `ReleaseGrapple` or `EscapeGrapple`. This keeps each command free of hidden effects and testable with plain values. | SRD 5.2.1, "Ending a Grapple", p. 182; design decision (this row) | unhappy (rule interaction) |
| FR-093 | `ReleaseGrapple(grappledID)` MUST call `GrappleRules.EndOnSet(set, grappledID, GrappleEndCause.Released)` (grappling.md FR-089, FR-085) on the set of a combatant holding Grappled, store the returned set and wrap `ConditionRemoved` (Grappled) then `GrappleEnded` (cause `released`). Release costs no action (grappling.md FR-029, FR-032), so no turn or resource is checked. A combatant not holding Grappled MUST return `DoesNotHaveCondition` (the mapping of `GrappleEndOnSetNotGrappled`). The grappler is not looked up: the event carries the grappler the removed condition held. Release and escape are two commands, not one taking a `GrappleEndCause`, which would let the host assert `grapplerIncapacitated` or `outOfRange` without the rule evaluating them. | SRD 5.2.1, "Ending a Grapple", p. 182 | happy |
| FR-094 | `EscapeGrapple(grappledID, outcome)` MUST take the host's `d20.Outcome` of the escape check (the check, its record and outcomes are grappling.md FR-033, FR-037 to FR-039; the action it costs, DC and options FR-035, FR-036, FR-041). On `success` it MUST call `GrappleRules.EndOnSet(set, grappledID, GrappleEndCause.Escaped)` (grappling.md FR-089), store the set and wrap `ConditionRemoved` (Grappled) then `GrappleEnded` (cause `escaped`). On `failure` it MUST return `TransitionApplied` with no events and an equal encounter. A combatant not holding Grappled MUST return `DoesNotHaveCondition`. | SRD 5.2.1, "Ending a Grapple", p. 182 | happy |

#### 4.11.1 Grapple cases (FR-086 to FR-088, FR-093, FR-094)

| Label | State | Command | Expected | Source | Path |
|---|---|---|---|---|---|
| grappler Incapacitated | B Grappled by A; A holds Incapacitated | check B, distance 5, range 5 | `ConditionRemoved` (Grappled), `GrappleEnded` (`grapplerIncapacitated`); B no longer Grappled | p. 182 | happy |
| grappler Stunned (implied) | B Grappled by A; A holds Stunned | check B, distance 5, range 5 | `ConditionRemoved` (Grappled), `GrappleEnded` (`grapplerIncapacitated`) | p. 182 | unhappy (rule interaction) |
| distance exceeds range | B Grappled by A; A can act | check B, distance 6, range 5 | `ConditionRemoved` (Grappled), `GrappleEnded` (`outOfRange`) | p. 182 | happy |
| distance equals range | B Grappled by A; A can act | check B, distance 5, range 5 | `TransitionApplied`, no events, encounter equal | p. 182 | unhappy (boundary) |
| both causes | B Grappled by A; A Incapacitated | check B, distance 10, range 5 | `ConditionRemoved` (Grappled), `GrappleEnded` (`grapplerIncapacitated`) | p. 182; grappling.md FR-088 | unhappy (rule interaction) |
| negative distance | B Grappled by A | check B, distance -1, range 5 | an `srderr.ErrInvalidInput` error when the command is built | n/a | unhappy (invalid input) |
| not Grappled | B holds no Grappled | check B | `TransitionRejected` (`DoesNotHaveCondition`) | p. 182 | unhappy (state) |
| grappler not in the encounter | B Grappled by an id the encounter does not hold | check B | `TransitionRejected` (`UnknownCombatant`, the grappler id), class `invalid` | n/a | unhappy (invalid input) |
| released | B Grappled by A | release B | `ConditionRemoved` (Grappled), `GrappleEnded` (`released`) | p. 182 | happy |
| release, not Grappled | B holds no Grappled | release B | `TransitionRejected` (`DoesNotHaveCondition`) | p. 182 | unhappy (state) |
| escape succeeds | B Grappled by A | escape B, `success` | `ConditionRemoved` (Grappled), `GrappleEnded` (`escaped`) | p. 182 | happy |
| escape fails | B Grappled by A | escape B, `failure` | `TransitionApplied`, no events, encounter equal | p. 182 | unhappy (rule interaction) |
| escape, not Grappled | B holds no Grappled | escape B, `success` | `TransitionRejected` (`DoesNotHaveCondition`) | p. 182 | unhappy (state) |
| no automatic end | B Grappled by A; `ApplyCondition` (Incapacitated) on A | apply | B still Grappled | p. 182 | unhappy (rule interaction) |

### 4.12 Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-003 | Withdrawn: moved to combat-foundations.md (FR-002 to FR-004) (`CombatantID`; shared-types-and-ownership.md FR-012); no test here |
| FR-004 | Withdrawn: moved to combat-foundations.md (FR-005) (`CombatantID` parsing); no test here |
| FR-006 | Withdrawn: moved to conditions.md (FR-012) ("no condition twice" is the `ConditionSet` invariant); no test here |

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

### Types

- **Aggregate root:** `Encounter` (immutable value: unexported fields, value receivers, every transition returns a new `Encounter`): `ID() EncounterID`, `Status() EncounterStatus` (closed interface with an unexported marker method: `EncounterInProgress`, `EncounterEndedStatus{Reason EndReason}`), `Tracker() TurnTracker` (holds the `TurnOrder`, round and active combatant), `Combatants() []Combatant` (a copy, in the tracker's order, FR-007, FR-052). Round and `ActiveCombatantID` are read from the tracker. Outside code holds and calls only the root (architecture §3). The zero `Encounter` is invalid (`IsZero`).
- **Entity:** `Combatant`: `Ref() CombatantRef` (id and `CombatantKind`, combat-foundations.md FR-013), `Side() Side`, `HitPoints() HitPointStatus`, `Conditions() ConditionSet`, `KnockedOut() bool`, `DeathSavingThrowMade() bool` (both default false, FR-098, FR-099). The kind is identity only; every rule reads `HitPointStatus` (`LifeState`, `ZeroHitPointsRule`). Derived: `Death() (DeathCause, bool)` (FR-068). Compared with `Equal` (FR-082).
- **Value objects and ids:** `EncounterID`, `EncounterIDError`, `EncounterIDRejection` (FR-002); `Side`; `EndReason`; `EncounterProblem` (closed; exported types carry `Problem`, FR-090: `NoCombatants`, `MissingSide(Side)`, `DuplicateCombatantID(id)`, `CombatantNotInTurnOrder(id)`, `TurnOrderEntryWithoutCombatant(id)`). No `EncounterDeathCause`: `DeathCause` already has `exhaustion`. Reused from sibling specs without new rules: `CombatantID`, `CombatantKind`, `CombatantRef`, `TurnTracker`, `TurnOrder`, `InitiativeCount`, `InitiativeOrderEstablished` (the start input), `HitPointStatus`, `HitPointRejection`, `DeathCause`, `ConditionSet`, `GrappleEndCause`, `GrappleHold`, `ConditionKind`.
- **Domain-level results** (not exported, FR-089): `EncounterTransition` (closed: `TransitionApplied(encounter, events, [damageResolutions])`, `TransitionRejected(encounter, rejection)`; `damageResolutions` non-empty only for `ApplyDamage`, FR-022); (no exported start result: `Encounter.Start` returns the encounter, its events and an error, and the `StartEncounter` use case maps a non-nil error to `StartEncounterInvalid(problems)`).
- **Rejection value object** (exported, a field of every `...Rejected` variant): `EncounterRejection` (closed; the exported type adds `Rejection`, FR-090): `EncounterAlreadyEnded`, `UnknownCombatant(id)`, `CombatantIsDead(id)`, `AlreadyHasCondition(id, kind)`, `ImmuneToCondition(id, kind)`, `DoesNotHaveCondition(id, kind)`, `ConditionImpliedBy(id, kind, impliedBy)`, `HitPointRule(id, HitPointRejection reason)`, `TurnOrderChanged` (FR-097; result `turn_order_changed`, class `refused`), `DeathSavingThrowAlreadyMade(id)` (FR-099). `HitPointRule` holds the hit point spec's rejection instead of re-declaring `NotDying` and `NotStable`; its `CreatureIsDead` is never carried (FR-078). `CombatantIsDead` stays because it also covers condition and grapple commands.
- **Use-case results** (exported, one closed `<UseCase>Result` per use case): `<UseCase>Applied(encounter, events)` and `<UseCase>Rejected(encounter, rejection)`; `StartEncounterResult` has `StartEncounterApplied` and `StartEncounterInvalid(problems)`; `ApplyDamageToCombatantApplied` also carries `damageResolutions`.
- **Domain events** (closed `EncounterEvent`, satisfying `Event`; immutable values, `Kind()` and `Combatants()`, FR-050, FR-051; all carry `EncounterID` and the round):

| Event | `kind` | `combatants` | Produced by | Carries (besides the above) |
|---|---|---|---|---|
| `EncounterStarted` | `encounter_started` | all ids in turn order | start | the ids in turn order |
| `EncounterEnded` | `encounter_ended` | all ids in turn order | end | the `EndReason` |
| `TurnEventOccurred` | the wrapped event's | the wrapped event's | start, advance, turn change (FR-096) | one tracker event unchanged, any of the 13 kinds of FR-096 |
| `HitPointEventOccurred` | the wrapped event's | `[combatant]` | damage, healing, death saving throw, stabilize, recover, temporary Hit Points, maximum reduction, Exhaustion death | the combatant and one `HitPointEvent` unchanged (`DamageTaken`, `DroppedToZeroHitPoints`, `FellUnconscious`, `KnockedOut`, `BecameBloodied`, `HitPointsRestored`, `TemporaryHitPointsGained`, `TemporaryHitPointsEnded`, `HitPointMaximumReduced`, `DeathSavingThrowMade`, `BecameStable`, `LostStability`, `Recovered`, `Died`) |
| `ConditionEventOccurred` | the wrapped event's | `[combatant]` | damage, healing and knock-out (Unconscious), condition commands | the combatant and one conditions event unchanged (`ConditionApplied`, `ConditionRemoved`, `ConditionLevelReduced`) |
| `GrappleEventOccurred` | the wrapped event's | `[grappled, grappler]` | grapple commands | the grappled combatant and `GrappleEnded` unchanged |
| `DyingCombatantTurnStarted` | `dying_combatant_turn_started` | `[active combatant]` | start, advance | the active combatant |

### Domain service

The root composes and never re-implements other specs' rules. For each command it calls the stateless rules (the tracker's advance operation, `DamageResolver` and the other hit point resolvers, `ConditionSet`, `GrappleRules.Hold` and `GrappleRules.EndOnSet`) with the combatant's state and turns their results into the new `Combatant`, tracker and events of section 4. It calls no use case of a sibling spec (so no sibling signal is emitted by an encounter command), never attacks.md, and no `ConditionEffects` query: merging roll contributions is the following spec's job (shared-types-and-ownership.md FR-026), and the encounter takes damage as a plain input.

### Invariants and where they are enforced

| Invariant | Enforced by | Failure |
|---|---|---|
| `EncounterID` is non-empty, non-blank | constructor (FR-001) | an `srderr.ErrInvalidInput` error; `NewX` (FR-002) |
| A combatant holds no condition twice | `ConditionSet` constructor (conditions.md FR-012) | an `srderr.ErrInvalidInput` error |
| Both sides have a combatant; ids unique; combatants are exactly the tracker's ids, in its order | `Encounter.Start` (FR-007, FR-011 to FR-013) | an `srderr.ErrInvalidInput` error; `StartEncounterInvalid` from `Start` |
| The order runs highest to lowest Initiative, ties resolved | `EstablishInitiativeOrder`, not re-checked (FR-009, FR-010) | n/a |
| The combatant list changes in size or order only by a roster method | only `AddCombatant`, `RemoveCombatant`, `RepositionCombatant` (FR-015; combatant-roster-changes.md) | none possible |
| The round only increases, by 1, at a wrap | the tracker's advance (FR-017) | none possible |
| Nothing changes after the encounter ended | every transition method (FR-045) | `EncounterAlreadyEnded` |
| A dead combatant is not changed | every combatant command (FR-029, FR-033, FR-079) | `CombatantIsDead` |
| A death is recorded on the combatant | `Combatant.Death` derivation (FR-068) | none possible |
| A condition is not held twice, except Exhaustion levels | `ConditionSet.Apply` (FR-035, FR-036) | `AlreadyHasCondition` |
| A grapple ends with `GrappleEnded`, never by plain removal | `GrappleRules.EndOnSet` (FR-086, FR-093, FR-094) | none possible |
| A wrapper names the combatant its event is about | wrapper constructors (FR-084) | an `srderr.ErrInvalidInput` error |

### Failures: results and errors

- Expected outcomes are results (architecture §6): every `EncounterRejection`, and `StartEncounterInvalid` for untrusted start input. No rejection carries display text. An unknown id anywhere in a command is a rejection, not an error (FR-046, FR-085).
- a returned `srderr.ErrInvalidInput` error (or a `MustX` panic) only for invalid values caught when the command is built: invalid `EncounterID`, negative healing amount, `levels` below 1, `natural` outside 1 to 20, Temporary Hit Points amount below 1, negative maximum reduction, negative grapple distance or range; a wrapper built with a mismatched combatant (FR-084); `Encounter.Start` with invalid input (it returns the joined problems; `MustStart` panics).
- No `*srderr.PortError`: the domain calls no port.

## 6. Interface

### Commands, use cases and results

One use case per command (architecture §4). Every command carries the current `Encounter` except `StartEncounterCommand`. Names are distinct from the sibling specs' use cases (`ApplyDamage`, `RestoreHitPoints`, `ApplyConditionToSet`, `RemoveConditionFromSet`, `ReduceExhaustionOnSet`, `AdvanceTrackerTurn`, `MakeDeathSavingThrow`, `StabilizeCreature`, `RecoverStableCreature`, `GainTemporaryHitPoints`, `ReduceHitPointMaximum`, `RecordExhaustionDeath`), which share package `combat`; the operation names of table 4.10.1 are their `snake_case`.

| Use case | Command (fields besides `Encounter`) | Result |
|---|---|---|
| `StartEncounter` | `StartEncounterCommand(id, established, combatants)`; no `Encounter` | `StartEncounterResult` |
| `AdvanceEncounterTurn` | `...Command(encounter, [updatedSpeeds], [sharedSpaceOccupants])` | `AdvanceEncounterTurnResult` |
| `ApplyDamageToCombatant` | `...Command{Encounter, CombatantID, Damage []Damage, Adjustments, Delivery DamageDelivery, HitKind HitKind}` | `ApplyDamageToCombatantResult` |
| `ApplyHealingToCombatant` | `...Command(encounter, combatantID, amount)` | `ApplyHealingToCombatantResult` |
| `ApplyConditionToCombatant` | `...Command(encounter, combatantID, condition)` | `ApplyConditionToCombatantResult` |
| `RemoveConditionFromCombatant` | `...Command(encounter, combatantID, kind)` | `RemoveConditionFromCombatantResult` |
| `ReduceCombatantExhaustion` | `...Command(encounter, combatantID, levels)` | `ReduceCombatantExhaustionResult` |
| `ApplyDeathSavingThrowToCombatant` | `...Command(encounter, combatantID, natural, total)` | `ApplyDeathSavingThrowToCombatantResult` |
| `StabilizeCombatant` | `...Command(encounter, combatantID)` | `StabilizeCombatantResult` |
| `RecoverStableCombatant` | `...Command(encounter, combatantID)` | `RecoverStableCombatantResult` |
| `GiveCombatantTemporaryHitPoints` | `...Command(encounter, combatantID, amount, choice)` | `GiveCombatantTemporaryHitPointsResult` |
| `ReduceCombatantHitPointMaximum` | `...Command(encounter, combatantID, amount)` | `ReduceCombatantHitPointMaximumResult` |
| `EndEncounter` | `EndEncounterCommand(encounter, reason)` | `EndEncounterResult` |
| `CheckCombatantGrapple` | `...Command(encounter, grappledID, distanceFeet, grappleRangeFeet)` (non-negative `int`s) | `CheckCombatantGrappleResult` |
| `ReleaseCombatantGrapple` | `...Command(encounter, grappledID)` | `ReleaseCombatantGrappleResult` |
| `EscapeCombatantGrapple` | `...Command(encounter, grappledID, outcome d20.Outcome)` | `EscapeCombatantGrappleResult` |

Domain entry points on the root (the use cases call them): package-level `Encounter.Start(id, established, combatants) (Encounter, []Event, error)` (an `errors.Join` of problems for invalid input) and `MustStart`; methods `AdvanceTurn`, `ApplyTurnChange(tracker, events)` (FR-096; no use case here calls it, those of combat-action-resolution.md do), `ApplyDamage`, `ApplyHealing`, `ApplyCondition`, `RemoveCondition`, `ReduceExhaustion`, `ApplyDeathSavingThrow`, `PrecheckDeathSavingThrow` (FR-100; returns the rejection or nil), `Stabilize`, `RecoverStable`, `GainTemporaryHitPoints`, `ReduceHitPointMaximum`, `CheckGrapple`, `ReleaseGrapple`, `EscapeGrapple`, `end`, each returning an `EncounterTransition`; plus `AddCombatant`, `RemoveCombatant`, `RepositionCombatant` from a part file, owned by combatant-roster-changes.md (returning its `RosterTransition`, which has the use cases). Both return types are not exported; each use case maps them to its own result (FR-089).

- All use cases are synchronous and take no `ctx` for rule work, except that the entry point takes `ctx context.Context` first because it emits a span (go.md section 4). Each is a type built by `NewX(opts ...Option)` with one method `Run(ctx, cmd) (XResult, error)`. Their only collaborator is the optional telemetry bundle (`WithTelemetry(telemetry.Telemetry)`, defaulting to the zero-value `telemetry.Noop`; observability §4; shared-types-and-ownership.md FR-036), handed to `CombatTelemetry` (FR-057).
- No driven port is introduced: no port failure contract, test double or contract suite, nothing added to `telemetry/telemetrytest` (the recording double exists, FR-038 there). No `dice.Roller`: every amount and roll arrives in the command.
- The host stores the `Encounter` between calls; two commands built from the same `Encounter` give two independent successors.
- These use cases are bare transitions. `AdvanceEncounterTurn`, `ApplyDeathSavingThrowToCombatant`, `StabilizeCombatant` and `ApplyDamageToCombatant` have wired counterparts in the spec that follows this one (its FR-126). A host that wants shared-space Prone, review prompts, Concentration checks, ledger expiry or the stabilizing check MUST call those. All encounter use cases stay exported (tests, replays, hosts wiring their own side effects); retiring four would break the one-use-case-per-command symmetry (FR-055).

### Exports

- Package `combat` (import path `github.com/Whealinghold/srd5e-go/combat`, no re-export package): the use cases, commands and results above, `Encounter`, `Combatant`, `EncounterID`, `EncounterIDError`, `EncounterIDRejection`, `ParseEncounterID`, `MustEncounterID`, `Side`, `EndReason` with its variants, `EncounterStatus` with its variants, `EncounterProblem` with its `...Problem` variants, `EncounterRejection` with its `...Rejection` variants, and every `EncounterEvent` type. `EncounterTransition` and `EncounterStart` are not exported. `CombatantID`, `CombatantKind`, `CombatantRef`, `TurnTracker` and the sibling event types are exported by the sibling slices. A duplicate or generic exported name fails the exported-name scan (architecture §9, check A7). The package `doc.go` lists inputs and outputs.
- There is no re-export package; every exported member has a godoc comment starting with its name. Telemetry names and attribute keys are constants in `CombatSignals` (combat-foundations.md FR-027); this spec adds `srd5e.combat.encounter_size`.

## 7. Observability

Answers to the seven evaluation questions (`doc/rules/observability.md` §2). The shared signals are catalogued by combat-foundations.md and `srd5e.combat.deaths` by hit-points-and-damage.md section 7; neither is repeated here.

1. **What can go wrong?** The domain returns an `srderr.ErrInvalidInput` error for caller bugs (the span records the returned error, observability §7; srd5e does not count it). Expected failures are returned results: invalid untrusted input (`StartEncounterInvalid`, unknown id), class `invalid`; and commands that do not fit the state (ended, dead, repeated or absent condition, Immunity, a save for a non-Dying combatant), class `refused`. A stale encounter cannot be detected.
2. **How would an operator know?** `srd5e.combat.operations` by `srd5e.combat.operation`, `result` and `result_class`; a rising `refused` share is visible. An `invalid` result sets the span error status; it does not increment `srd5e.combat.errors`, which counts port failures, and this spec has none.
3. **What is needed to diagnose?** The operation, result name (carries the rejection reason), class, event kinds, combatant count and, for deaths, `srd5e.combat.death_cause`; all low-cardinality. Ids are never attributes; the returned rejection and events carry them.
4. **What is counted or timed?** Calls by operation, result and class; events by kind; deaths by cause; duration; encounter size at start (the only new signal). Not counted on the encounter path: hit-points-and-damage.md's own `srd5e.combat.damage_resolved`, `hit_points_lost`, `hit_points_restored`, `death_saves`, `hit_point_refusals`; they are emitted by that spec's use cases only and the encounter use cases call domain code, so a host running combat through the encounter does not get them. It can use `srd5e.combat.events` by kind (`damage_taken`, `hit_points_restored`, `death_saving_throw_made`, `died`) and `operations` by `result` (`not_dying`, `not_stable`); amounts and damage types are not attributes (FR-057).
5. **Is existing instrumentation still correct?** Yes. The encounter calls only domain code, never sibling use cases, so their signals are unaffected and nothing is counted twice; use cases do not nest. This spec defines none of the superseded `encounter_*` signals or attributes (combat-foundations.md section 5). Deaths use `srd5e.combat.deaths` (FR-081).
6. **What should a consumer monitor?** The shared rows of combat-foundations.md: the `refused` share per operation, any `invalid` rise (`unknown_combatant`: the host sends ids the encounter does not hold; `invalid_encounter`: bad data), and `operation_duration` p99 (pure rules take microseconds, so slow means a slow `Observability` adapter). Specific here: upper percentiles of `srd5e.combat.encounter_size`.
7. **Is anything now noisy?** No. No log events are emitted (combat-foundations.md FR-034); events are counted, never logged one by one (observability §7, "Log intelligently").

Telemetry is produced only in the use cases, through `CombatTelemetry`; the domain is silent by design (observability §3), so it has no observability tests.

| Signal | Type | Unit | Attributes | Emitted by | Purpose | Suggested monitoring |
|---|---|---|---|---|---|---|
| `srd5e.combat.encounter_size` | histogram | `{combatant}` | none | `StartEncounter` | Size of encounters at start. | Watch the upper percentiles; large encounters cost the most on every transition. |

Values this spec gives the shared attributes are in table 4.10.1; the `srd5e.combat.event` values are the event `kind`s (FR-051). The entry above feeds `doc/observability/signals.md`.

## 8. Test plan

Doubles: the recording telemetry double from `telemetry/telemetrytest` (shared-types-and-ownership.md FR-038) and the zero-value `telemetry.Noop` (FR-036); use-case tests (package `combat_test`, table-driven, one `t.Run` per table row) import only those. No port doubles or contract suite (section 6). Fixed `Damage`, healing, d20 face and `InitiativeCount` values, no `dice.Roller`. Domain unit tests build a tracker with `InitiativeOrdering.Establish` (initiative-and-actions.md FR-123) and take the `InitiativeOrderEstablished` variant; they never import an `application/` use case (testing.md §3). Only the application tests use the encounter's own use cases.

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002, FR-005 | unit | `ParseEncounterID("")` and `ParseEncounterID("   ")` return an error; `ParseEncounterID` over empty, blank, valid, very long, non-ASCII strings never panics (a `FuzzParseEncounterID` test) and `MustEncounterID` panics iff it returns an error; equality under `==`. Non-assignability between `EncounterID` and `CombatantID` is a compile-time property (not covered by a runtime test here). |
| FR-007 to FR-015 | unit | Two- and four-combatant starts, input order differing from the tracker's (stored in tracker order); event list exact with and without `DyingCombatantTurnStarted`; FR-009 two tests; FR-010 tracker from a tie-break; FR-011 empty, one side, one per side; FR-012 duplicate id on one side and across; FR-013 one test per `EncounterProblem`, several at once, `Start` returns the joined error for every invalid input and never panics (fuzz), `MustStart` panics exactly then; FR-014 mutate the source list; FR-015 every command type and many advances keep the order; the three roster methods are tested in combatant-roster-changes.md. |
| FR-016, FR-017, FR-018, FR-019, FR-020 | unit | Tables 4.3.1 and 4.3.2, one test per row; dead and Unconscious combatants each get a turn. |
| FR-096, FR-097 | unit | Table 4.3.3, one test per row; the 13 kinds each under their own `kind`; combatants, status, order equal and receiver unchanged. |
| FR-021 to FR-029, FR-070, FR-072, FR-091 | unit | Table 4.4.1, one test per row (FR-022: `DamageResolution` asserted field by field, no recomputation; arithmetic is tested in the hit points spec; FR-072 plus a Critical Hit variant). |
| FR-071 | unit | Each command field reaches the hit point rules unchanged (result equals the hit point spec's for the same inputs). |
| FR-030 to FR-033, FR-080, FR-092 | unit | Table 4.5.1, one test per row; FR-092: Unconscious stays, then host `RemoveCondition` removes it. |
| FR-034 to FR-041, FR-063 to FR-067 | unit | Each condition type at least once; immune combatant; absent condition; table 4.6.1, one test per row; FR-067 immune to Unconscious dropped to 0: no `ConditionApplied`, damage still applied. |
| FR-042 to FR-045 | unit | Both `EndReason`s; both sides; `EndReasonSideDefeated(sideA)` accepted while all `sideA` alive; one side dead keeps `EncounterInProgress` and no event; one test per command type on an ended encounter (including `ApplyTurnChange`, `end`, the three grapple commands). |
| FR-073 to FR-077, FR-098 to FR-100 | unit | Table 4.8.1, one test per row; `PrecheckDeathSavingThrow` over each of its five outcomes equals what `ApplyDeathSavingThrow` returns and changes nothing; flag cleared by `Start` and `AdvanceTurn` for the combatant whose turn began only; `knockedOut` set, kept false under a held or immune Unconscious, cleared on every removal and on dropping to 0. |
| FR-046 to FR-049, FR-052 to FR-054, FR-082, FR-083 | unit | Unknown id per command type; after each command all other fields equal; each rejection leaves the encounter equal with no events; receiver unchanged; mutating the slice returned by `Combatants()` leaves the encounter unchanged; equal and unequal pairs one differing field per case (encounter, and seven cases for `Combatant`); two starts differing only in input order are equal; repeat each command twice; combatants of each `LifeState` do not throw, a `Dying` combatant without Unconscious accepted. |
| FR-068, FR-069 | unit | Each of the five `DeathCause` values sets `death`; Exhaustion 6 sets `Dead(exhaustion)`; `death` stays set; damage and healing after level 6 rejected. |
| FR-078, FR-079 | unit | Table 4.9.1, one test per row; one test per command naming a combatant on a dead one, including dead of Exhaustion; `AdvanceTurn`, `ApplyTurnChange` and `end` unaffected. |
| FR-084, FR-085 | unit | Wrapped events equal the sibling spec's for the same inputs; a death yields one wrapped `Died` and no other; mismatched wrapper combatants return an `srderr.ErrInvalidInput` error; `GrappleEnded` under the grappled id; updated speeds and occupants reach the tracker; unknown id in the update is `UnknownCombatant` and the tracker is not called; two unknown ids report the smallest; `FellProneInSharedSpace` wrapped, no Prone added. |
| FR-086 to FR-088, FR-093, FR-094 | unit | Table 4.11.1, one test per row; the grappler's Incapacitated is read from the encounter's own `ConditionSet`; events exact and ordered (`ConditionRemoved` then `GrappleEnded`); FR-088 runs every other command type and asserts Grappled untouched. |
| FR-050, FR-051 | unit and architecture | Mutating a returned event slice leaves the event unchanged; value equality including `kind` and `combatants`; unexported fields with value receivers; every event carries encounter id and round (and combatant id where applicable, round at a wrap); own events' `kind` and `combatants` unmodifiable and duplicate-free; wrappers report the wrapped `kind` and `combatants` (shared table helper of combat-foundations.md section 8). Architecture test: the family is closed, each member implements `Event` (combat-foundations.md FR-016), no name ends in `Due`, `Required` or `Needed`, only the four wrappers end in `Occurred`; the marked `TransportEnvelope` set equals exactly these four wrappers. |
| FR-055, FR-056 | application | One use case per command with real domain and no stored state (two calls match); results equal with the no-op and the recording telemetry double. |
| FR-057 to FR-060 | application | One span `srd5e.combat.<operation>`, one `operations` increment and one duration sample per call, no attribute value equals a supplied id; every refusal of table 4.10.1 is `refused`, `failure`, no error status, no `errors`, no `error.type`, no log, result name = rejection name, `applied` is `success`; invalid start and unknown combatant (also in the speeds update and as a grappler) are `invalid` with error status; one `events` increment per event with its `kind` for a start (three events), a damage to 0 (wrapped kinds) and a rejection (none). |
| FR-061 | application | A recording double whose `Meter` panics with an error value on the first counter increment: the panic propagates unchanged out of `AdvanceEncounterTurn`, the span records it as in combat-foundations.md FR-035's test, `errors` not incremented. |
| FR-062, FR-081 | application | Size recorded on a valid start only; one `srd5e.combat.deaths` increment per wrapped `Died` with the right cause for each of the five causes, none for a rejection. |
| FR-089, FR-090 | architecture and application | Exported-name scan (check A7): no generic variant name or duplicate exported name; every rejection and problem type ends in `Rejection` or `Problem`; status and end-reason variants carry their prefix; none equals a hit-points-and-damage.md name; each use case returns its own `<UseCase>Result` whose rejected variant holds an `EncounterRejection`; domain-level types not exported. |
| FR-095 | application and architecture | Table 4.10.1, one test per use case and listed result: operation, result name and class exact; closed result sets; no operation equals a sibling's. |

- **Scenario test** in `scenarios/` (package `scenarios_test`) through public packages only: four combatants start; two rounds advance; a player character goes to 0 Hit Points (Unconscious, then `DyingCombatantTurnStarted` at its next turn), makes a Death Saving Throw and is healed; a monster dies; a grapple is started by the host's `ApplyCondition` and ended by `ReleaseGrapple`; the encounter ends with `EndReasonSideDefeated`. It supplements the unit tests.
- **Architecture tests:** the existing guardrails (domain files call no use case or `*test` package, no goroutine, clock or random source in the domain, signal constants and catalogue in sync, the context map and in-context order of shared-types-and-ownership.md FR-013 and FR-015: no import of a later `combat` spec or a downstream context), plus the suffix check of FR-050 and the exported-name scans of FR-089 and FR-090.
- **Mutation testing:** the `combat` encounter, combatant, id and event files, `gomu` run only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`, threshold 80; the `*_usecase.go` files SHOULD be included (`doc/rules/testing.md` §8). The boundary rows (remainder 11, 12, 13; wrap rows; Exhaustion levels 5 and 6; d20 faces 1, 10, 20; grapple distance 5 and 6 against range 5) kill comparison mutants.

## 9. Open questions and decisions

No open questions remain; none is escalated. Settled, one line each:

- **`TurnTracker` value equality.** FR-049, FR-053, FR-054, FR-097 and the "encounter equal" rows of table 4.11.1 compare trackers, so `TurnTracker`, `TurnOrder`, `InitiativeCount`, profiles and readied actions compare by value (architecture §3; amendment requested of initiative-and-actions.md).
- **Damage to a Stable creature (hit-points-and-damage.md Q-3).** Both effects, FR-072. SRD 5.2.1, "Stabilizing a Character", p. 18: damage ends Stable and Death Saving Throws resume; "Damage at 0 Hit Points", p. 18: any damage at 0 Hit Points is one failure (two for a Critical Hit). The creature is at 0 Hit Points and takes damage, so both apply (interpretation; the common reading of the 2024 text agrees). hit-points-and-damage.md closes Q-3 and drops "provisional" from its FR-071 (amendment requested).
- **Knock-out marker (combat-action-resolution.md Q-41).** `Combatant.KnockedOut` (FR-098), not a condition source: the SRD ends a knock-out early on regaining Hit Points (p. 17), which FR-092 now does, and conditions.md keeps sources for creatures only. Interpretation: a spell's Unconscious is never marked, so healing cannot end it. First aid and the Short Rest stay the host's. This replaces the earlier deviation.
- **Per-turn feat limits (feats-behaviour.md Q10).** The uses of a `FeatUsageLimit` are `TurnTracker` per-turn resources (initiative-and-actions.md, amendment requested), reset by its advance; the encounter holds no field and applyTurnChange (FR-096) already carries them.
- **Swallow and Engulf (monster-engulf-and-swallow.md amendment) not applied.** The encounter holds no `Grapples` record (section 1; the host keeps it), so `Containments` beside it would be a new aggregate part; that spec's use cases take a `dice.Roller` and own their state in and out, and its FR-074 forbids importing this one. The host feeds `ReleaseOccupants` from facts this spec emits: wrapped `Died` (`ContainerDied`), `GrappleEnded` (`GrappleReview`), a roster removal (`OccupantGone`). No dependency added.
- Earlier settled items are in the rows that carry them (FR-011, FR-020, FR-029, FR-053, FR-078, FR-079, FR-085, FR-088, FR-092, FR-093).

## 10. Changelog impact

Under `Unreleased`, "Added":

- `Encounter` aggregate in the `combat` context: `EncounterID` with `ParseEncounterID`, `MustEncounterID`, `EncounterIDError` and `EncounterIDRejection`, `Combatant`, `Side`, `EndReason` (`EndReasonSideDefeated`, `EndReasonMutualAgreement`), `EncounterStatus` (`EncounterInProgress`, `EncounterEndedStatus`), `EncounterProblem`, `EncounterRejection`, the encounter events (`EncounterStarted`, `EncounterEnded`, `TurnEventOccurred`, `HitPointEventOccurred`, `ConditionEventOccurred`, `GrappleEventOccurred`, `DyingCombatantTurnStarted`), and the use cases of section 6 (`StartEncounter` to `EscapeCombatantGrapple`), each with its own closed `<UseCase>Result` and variants (`<UseCase>Applied`, `<UseCase>Rejected`).
- The root method `Encounter.ApplyTurnChange` (FR-096), the hook combat-action-resolution.md calls; no use case or signal of its own.
- `ApplyDeathSavingThrow(natural, total)`, `PrecheckDeathSavingThrow`, the `DeathSavingThrowAlreadyMade` rejection, `Combatant.KnockedOut` and `deathSavingThrowMade` (FR-092, FR-098 to FR-100). Telemetry signal `srd5e.combat.encounter_size` (section 7). The use cases also record the shared `combat` signals and `srd5e.combat.deaths`.

Not breaking: all new API in an unreleased package. The event set is a closed sub-family; adding an event after release would be breaking for consumers that switch over it (architecture §7).
