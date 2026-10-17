# Combatant roster changes

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `conditions.md`, `grappling.md`, `attack-and-help-actions.md`, `initiative-and-actions.md`, `encounter-aggregate.md`, `combat-action-resolution.md`, `errors-and-results.md`, `observability-ports.md` |

Type phase 4 ([shared-types-and-ownership.md](shared-types-and-ownership.md) FR-014). Rank 30 (amendment A-1, section 9): the first `combat` spec after `combat-action-resolution`, which it consumes only for the `ActionLedger` value (its FR-112). Binding decisions applied as written: shared-types-and-ownership.md FR-017 to FR-019, FR-021, FR-036, FR-044, FR-048, FR-062.

> Go notation used below. A name in lowerCamelCase (a field, command argument or enum member) stands for the exported PascalCase Go identifier (`attackerID` is the field `AttackerID`). A trailing `?` marks an optional value (the zero value or a `(T, bool)` pair). `[]T` is a slice; `set of T` is a collection without duplicates. A `(T, error)` pair replaces an accepted/rejected result pair: the nil-error case is the accepted case and the error is the typed rejection, satisfying `errors.Is(err, srderr.ErrInvalidInput)`; a parser reports every problem at once through `errors.Join`. Rule outcomes (refusals, not-possible stops) stay result values. A use case is a struct with `Run(ctx, cmd)`; its `error` is non-nil only for a port failure.

## 1. Purpose and scope

Adds, removes and repositions a combatant of an `Encounter` after combat has started. Four specs deferred this to "a later spec" (encounter-aggregate.md section 1 "Out of scope"; initiative-and-actions.md section 1 and its settled list; combat-action-resolution.md FR-123, Q-24; conditions.md D-12). It owns:

- `RosterPosition`, the explicit place a combatant takes in the `TurnOrder` (the SRD gives a place only for summoned creatures: "immediately after yours", FR-002);
- three `Encounter` transitions (`addCombatant`, `removeCombatant`, `repositionCombatant`) and the matching `TurnTracker` operations, one use case each;
- the events `CombatantJoined`, `CombatantRemoved`, `CombatantRepositioned` (closed `RosterEvent`);
- the clean-up a departure needs: grapples that involve the leaver end through `GrappleRules.EndOnSet`; the leaver's pending Help benefits are dropped from the `ActionLedger` (resolves combat-action-resolution.md Q-24).
- Left to the host (deliberately): effects in `ActiveEffects`/`OngoingEffects` (active-effects.md) sourced from or targeting the leaver, and other combatants' `ReadiedAction`s whose trigger concerns it, are untouched by a departure (FR-048).

It prerequisites summoned-and-companion creatures (a later spec calls `addCombatant`) and revival of a creature already removed (FR-027).

**Out of scope:**
- Rolling Initiative for a newcomer, resolving its ties, or deciding that reinforcements arrive. The host runs `RollInitiative` and passes a count; the engine never rolls here (FR-015). "Reinforcements left to the host" is a package decision.
- Summon spells, Find Familiar and the like: creating the creature, its stat block and its duration. This spec only places the creature in the order once the host has a `Combatant`.
- Positions, Cover, distance (p. 13 to 15) and surprise rounds (not in SRD 5.2.1).
- A second turn for one combatant in a round ("Thief's Reflexes", p. 64): the order holds one entry per `CombatantID` (initiative-and-actions.md FR-018), so a second turn at "Initiative minus 10" cannot be a roster entry. Flagged, not modelled.
- Ending the encounter when a side has left (FR-020: the host calls `end`), waking, healing or reviving (returning-to-life.md), and Legendary Action timing (monster-limited-use.md).
- Removing the active combatant: refused (FR-019); the host advances the turn first.
- Material outside SRD 5.2.1: none found missing. The tie rule for a late arrival is not in the SRD (p. 13 gives "the GM decides" for ties); it is a host choice (FR-003).

## 2. Source

- SRD 5.2.1, "Combat", "The Order of Combat", "Combat Step by Step", "Initiative" ("Initiative Order", "Ties"), p. 13: "The Initiative order remains the same from round to round"; "the GM decides the order among tied monsters, and the players decide the order among tied characters".
- SRD 5.2.1, "Combat", "Ending Combat", p. 14: combat ends when one side is "defeated, which can mean the creatures are killed or knocked out or have surrendered or fled".
- SRD 5.2.1, "Mounted Combat", "Controlling a Mount", p. 16: "The Initiative of a controlled mount changes to match yours when you mount it ... A controlled mount can move and act even on the turn that you mount it."
- Placement of summoned creatures: "Find Familiar" (rolls its own Initiative) and "Find Steed" (shares the Initiative count; "takes its turn immediately after yours" only while the summoner has the Incapacitated condition), p. 130; "Giant Insect", p. 135, and "Summon Dragon", p. 166 ("shares your Initiative count, but it takes its turn immediately after yours"); "Animate Objects", p. 109; "True Polymorph", p. 170 ("takes its turns immediately after yours"); "Staff of the Python", p. 247 ("shares your Initiative count, taking its turn immediately after yours"). All "disappear" at 0 Hit Points or when the spell ends.
- SRD 5.2.1, "Rules Glossary": "Condition", p. 179; "Grappled [Condition]", "Grappling" ("Ending a Grapple"), p. 182; "Dead", p. 180; "Help [Action]", pp. 182-183.
- Web research (informing the judgement only; nothing is copied): there is no official rule for a creature that enters an ongoing combat; common table practice is that it rolls Initiative, is slotted by that count, and acts when its place next comes up. FR-008 follows from the order mechanics, not from that practice.
- Package design: `doc/rules/architecture.md` §2 to §6; `doc/rules/observability.md` §2, §5 to §7; shared-types-and-ownership.md; combat-foundations.md; encounter-aggregate.md; initiative-and-actions.md.

Page numbers are printed numbers; the PDF index matches them (checked on pages 13, 14, 16, 64, 109, 130, 135, 166, 170, 247).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Initiative order, Initiative count | `TurnOrder`, `InitiativeCount` (initiative-and-actions.md) | p. 13. Reused. Each entry already carries its count (`InitiativeStanding`), so no change to `TurnOrder`. |
| "immediately after yours" | `RosterAfterCombatant(anchor)` | p. 109, 130, 135, 166, 170, 247. The newcomer shares the anchor's count. |
| a rolled count for a latecomer | `RosterAtInitiative(count, TiePlacement)` | p. 13; "rolls its own Initiative", p. 130. `TiePlacement` (`beforeTied`, `afterTied`) is the host's tie decision (p. 13 "Ties"). |
| (placement) | closed `RosterPosition` | Not an SRD term. |
| Combatant | `Combatant`, `CombatantProfile` (encounter-aggregate.md, initiative-and-actions.md) | p. 13. Reused. |
| side defeated, fled, surrendered | `RemovalCause` (`fled`, `surrendered`, `disappeared`, `departed`) | p. 14; "disappears", pp. 130, 135, 166. `departed` is any other reason the host reports (teleported away). |
| (why a combatant arrived) | `JoinReason` (`summoned`, `arrived`) | `summoned`: placed by a spell or item; `arrived`: reinforcement or a creature that was absent. |
| (events) | closed `RosterEvent` implementing `CombatEvent`: `CombatantJoined`, `CombatantRemoved`, `CombatantRepositioned` | Facts, past tense (combat-foundations.md FR-020). |
| (refusals) | closed `RosterRejection`: `RosterEncounterEndedRejection`, `RosterUnknownCombatantRejection`, `RosterDuplicateCombatantRejection`, `RosterCombatantIsActiveRejection`, `RosterWouldEmptySideRejection` | Held in the `...Rejected` variants. |
| (tracker step) | `RosterStep` (closed: `RosterStepApplied(tracker, standing)`, an error (RosterRejection)) | Tracker-level result, silent and pure (FR-039). |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids; never renumber. Cases are tables 4.1a, 4.2a, 4.3a and 4.4a (one test per row). Base order for the tables: A 15, B 15, C 10, D 5, with B active in round 2.

### 4.1 Positions and the order

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `RosterPosition` MUST be closed with exactly `RosterAfterCombatant(anchor)` and `RosterAtInitiative(count, tiePlacement)`; both compare by value. `count` accepts any integer (negative and zero included). | SRD 5.2.1, "Initiative", p. 13 (a count can be negative, initiative-and-actions.md FR-011) | unhappy (boundary) |
| FR-002 | `RosterAfterCombatant(anchor)` MUST place the entry directly after the anchor's entry and give it the anchor's count; every other entry keeps its relative order. With several entries placed after one anchor, the latest placed is nearest the anchor (A, then Y, X, B). | SRD 5.2.1, "Giant Insect", p. 135; "Summon Dragon", p. 166 ("shares your Initiative count, but it takes its turn immediately after yours"); "Find Steed", p. 130 (shares the count; the "immediately after yours" wording is conditional on the Incapacitated condition) | happy |
| FR-003 | `RosterAtInitiative(count, tie)` MUST place the entry after every entry with a higher count and before every entry with a lower count and give it `count`. Placement is computed on the order with the subject removed (so a reposition never counts itself in a tie group). Among entries with an equal count, `beforeTied` places it before the first of them and `afterTied` after the last; without an equal count the two give the same order. The engine does not decide the tie; the host does (interpretation: p. 13 leaves ties to the GM and players, and a late arrival is one more tie). | SRD 5.2.1, "Initiative", "Initiative Order", "Ties", p. 13; "Find Familiar", p. 130 (rolls its own Initiative) | happy |
| FR-004 | After every roster operation the counts along the order MUST be non-increasing, each id MUST occur once, and the order MUST depend only on the previous order and the operation (same inputs, same order; no clock or randomness). | SRD 5.2.1, "Initiative Order", p. 13; architecture.md §3 | unhappy (determinism) |
| FR-005 | A roster operation MUST NOT change the round, the active combatant, the active combatant's turn resources, any other entry's spent Reaction, Dodge, readied action or `MovementSpeeds`, or any `Combatant` other than the subject. | SRD 5.2.1, "Your Turn", p. 13; encounter-aggregate.md FR-047 | unhappy (rule interaction) |
| FR-006 | A combatant placed before the active combatant in the order MUST NOT take a turn until the round wraps; one placed after it MUST take its turn this round when `advance` reaches it. No `TrackerTurnStarted` is produced by a roster operation. | SRD 5.2.1, "Combat Step by Step", step 3, p. 13 ("Each participant ... takes a turn in Initiative order") | unhappy (rule interaction) |
| FR-007 | A joiner MUST start with its Reaction available, not Dodging, no readied action and no turn resources; its `MovementSpeeds` are those of the supplied `CombatantProfile`. Its resources are set when its own turn starts (initiative-and-actions.md FR-034). | SRD 5.2.1, "Reactions", p. 10; "Your Turn", p. 13 | happy |

#### Cases 4.1a: placement (one test per row; result is the order and the newcomer X's count)

| Label | Operation | Order after | X count | Source | Path |
|---|---|---|---|---|---|
| after a combatant already past | `RosterAfterCombatant(A)` | A, X, B, C, D | 15 | FR-002 | happy |
| after the active combatant | `RosterAfterCombatant(B)` | A, B, X, C, D | 15 | FR-002, FR-006 | happy |
| after the last | `RosterAfterCombatant(D)` | A, B, C, D, X | 5 | FR-002 | unhappy (boundary) |
| two after one anchor | X then Y, both `RosterAfterCombatant(A)` | A, Y, X, B, C, D | 15, 15 | FR-002 | unhappy (rule interaction) |
| highest count | `RosterAtInitiative(20, afterTied)` | X, A, B, C, D | 20 | FR-003 | unhappy (boundary) |
| tied with two, before; after | `RosterAtInitiative(15, beforeTied)`; `(15, afterTied)` | X, A, B, C, D; A, B, X, C, D | 15 | FR-003 | unhappy (rule interaction) |
| tied with one, before; after | `RosterAtInitiative(10, beforeTied)`; `(10, afterTied)` | A, B, X, C, D; A, B, C, X, D | 10 | FR-003 | unhappy (rule interaction) |
| between counts, both placements | `RosterAtInitiative(7, beforeTied)`; `(7, afterTied)` | A, B, C, X, D (both) | 7 | FR-003 | happy |
| lowest tie, then below all | `RosterAtInitiative(5, afterTied)`; `(-3, beforeTied)` | A, B, C, D, X; A, B, C, D, X | 5; -3 | FR-003, FR-001 | unhappy (boundary) |
| zero count | `RosterAtInitiative(0, afterTied)` | A, B, C, D, X | 0 | FR-001 | unhappy (boundary) |

### 4.2 Adding a combatant

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-008 | `Encounter.AddCombatant(combatant, profile, position, reason)` on an in-progress encounter MUST return `RosterApplied` whose encounter has the joiner in the tracker's order at the position of FR-002 or FR-003, the combatants list in that same order (encounter-aggregate.md FR-052), the status unchanged, and the single event `CombatantJoined`. | SRD 5.2.1, "Initiative Order", p. 13 | happy |
| FR-009 | An add MUST be refused, in this order, with `RosterEncounterEndedRejection` (ended), `RosterDuplicateCombatantRejection(id)` (the id is already a combatant of the encounter) and `RosterUnknownCombatantRejection(id)` (the anchor of `RosterAfterCombatant` is not a combatant; an anchor equal to the new id is such a case, the newcomer is not yet a combatant; a dead anchor is accepted, FR-027). A refusal changes nothing and produces no event. Duplicate precedes unknown because a duplicate id is invalid input of the subject itself (encounter-aggregate.md FR-012), checked before the anchor is read; FR-078 there orders ids named by a command that already has a valid subject. | SRD 5.2.1, "Ending Combat", p. 14; encounter-aggregate.md FR-012, FR-045, FR-078 | unhappy (state) |
| FR-010 | `AddCombatantToEncounterCommand` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when `profile.id` differs from the combatant's id. A command is built from an encounter and values; it does not validate the anchor (FR-009 does). | architecture.md §6; initiative-and-actions.md FR-025 | unhappy (invalid input) |
| FR-011 | Either `Side` MAY be joined, any number of times; the encounter's size is not limited. A joiner of any `LifeState` or `ConditionSet` MUST be accepted unchanged and unchecked (as at start, encounter-aggregate.md FR-083). | SRD 5.2.1, "The Order of Combat", p. 13; design decision (the engine holds two sides, encounter-aggregate.md FR-011) | unhappy (boundary) |
| FR-012 | `CombatantJoined` MUST carry the `EncounterID`, the round, the combatant, its `Side`, the `JoinReason`, its count and `precededBy` (FR-034), `kind` `combatant_joined`, `combatants` `[id]`. | combat-foundations.md FR-017 to FR-019 | happy |
| FR-013 | A summoned creature MUST be placed with `RosterAfterCombatant(summoner)`, giving it the summoner's count. Find Steed's steed shares the count and, as a controlled mount, is placed after its rider by the mount rule (p. 16, D-8); p. 130 states "immediately after yours" only when the summoner is Incapacitated, which does not alter the placement. The host passes `JoinReason.Summoned`. | SRD 5.2.1, "Giant Insect", p. 135; "Summon Dragon", p. 166; "True Polymorph", p. 170; "Staff of the Python", p. 247 (unconditional); "Animate Objects", p. 109; "Find Steed", p. 130; "Controlling a Mount", p. 16 | happy |
| FR-014 | A creature that "rolls its own Initiative" MUST be placed with `RosterAtInitiative(count, tie)` from a count the host rolled with `RollInitiative` and `InitiativeCount.FromRoll` (initiative-and-actions.md FR-139). The engine rolls nothing and calls no port. | SRD 5.2.1, "Find Familiar", p. 130; "Initiative", p. 13 | happy |
| FR-015 | Reinforcements MUST be decided by the host; no operation here fires on its own and none inspects hit points, sides or rounds to add a creature. | package design (the SRD gives no reinforcement rule) | unhappy (rule interaction) |

#### Cases 4.2a: add (one test per row)

| Label | State | Operation | Expected | Source | Path |
|---|---|---|---|---|---|
| summoner is active | order A, B, C, D, B active | X after B | applied; B still active, round 2; next advance starts X | FR-006, FR-008 | happy |
| latecomer placed before active | same | X at 20 | applied; X first; round and active unchanged; X acts next round | FR-006 | unhappy (rule interaction) |
| duplicate id | X is already a combatant | X after A | `RosterDuplicateCombatantRejection(X)`; encounter equal | FR-009 | unhappy (state) |
| unknown anchor | no Z | X after Z | `RosterUnknownCombatantRejection(Z)` | FR-009 | unhappy (invalid input) |
| ended and duplicate | ended encounter, X present | X after A | `RosterEncounterEndedRejection` | FR-009 | unhappy (rule interaction) |
| duplicate beats unknown anchor | X present | X after Z | duplicate | FR-009 | unhappy (rule interaction) |
| profile id differs | combatant X, profile Y | construct command | `srderr.ErrInvalidInput` | FR-010 | unhappy (invalid input) |
| dead and Dying joiners | joiner `Dead`; joiner `Dying` | at 10 | applied; state unchanged; Dying joiner gets `DyingCombatantTurnStarted` only when `advance` starts its turn | FR-011 | unhappy (boundary) |
| other side joins | X on `sideB` where B is `sideB` | X after B | applied; both sides unchanged | FR-011 | happy |
| anchor is the new id | X not yet a combatant | X after X | `RosterUnknownCombatantRejection(X)` | FR-009 | unhappy (invalid input) |
| dead anchor | A is `Dead` | X after A | applied; X placed after A, count 15 | FR-009, FR-027 | unhappy (state) |
| stale tracker after add | add X succeeded | `applyTurnChange` with the tracker held from before the add | `TurnOrderChanged` rejection; encounter equal | FR-038 | unhappy (state) |

### 4.3 Removing a combatant

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-016 | `Encounter.RemoveCombatant(id, cause)` MUST return `RosterApplied` whose encounter lacks the combatant in the combatants list, the order, the profiles, the spent-Reaction set, the Dodging set and the readied actions; the round, active combatant and every other entry are unchanged (FR-005). The events are those of FR-021 then `CombatantRemoved`. | SRD 5.2.1, "Ending Combat", p. 14 ("fled"); "Find Steed", p. 130 ("disappears") | happy |
| FR-017 | A removed id MUST be free again: a later add of the same id is not a duplicate and starts fresh (FR-007); the engine keeps no record of the removed combatant. | package design (state in, state out, architecture.md §4) | unhappy (state) |
| FR-018 | A combatant of any `LifeState` MAY be removed (dead, Dying, Stable, Unconscious included). Removal MUST NOT produce `Died` or any hit point or condition event for the leaver, and its recorded death leaves with it. | SRD 5.2.1, "Dead", p. 180; encounter-aggregate.md FR-068 | unhappy (rule interaction) |
| FR-019 | Removing the active combatant MUST be refused with `RosterCombatantIsActiveRejection(id)`. The host first ends its turn with `AdvanceCombatTurn` (combat-action-resolution.md FR-126), which applies the turn-end duties this spec would otherwise repeat; then it removes. | SRD 5.2.1, "Your Turn", p. 13; design decision (D-2) | unhappy (state) |
| FR-020 | Removal MUST be refused with `RosterWouldEmptySideRejection(side)` when it leaves no combatant on that side, and the encounter MUST NOT end by itself. A side that has entirely fled is "defeated": the host calls `end` with `EndReasonSideDefeated(side)`. | SRD 5.2.1, "Ending Combat", p. 14; encounter-aggregate.md FR-011, FR-044 | unhappy (boundary) |
| FR-021 | Removal MUST end every grapple that involves the leaver, before `CombatantRemoved`, with `GrappleRules.EndOnSet(set, grappled, GrappleEndCause.OutOfRange)` (grappling.md FR-089, FR-090): first the leaver's own Grappled condition (when it holds one), then, in turn order, each remaining combatant whose Grappled names the leaver as grappler. Each ending wraps `ConditionRemoved` then `GrappleEnded` as `ConditionEventOccurred` and `GrappleEventOccurred` (encounter-aggregate.md FR-084, FR-086), so a host holding a `Grapples` record updates it from the event. Interpretation: a creature that has left the battle is beyond any grapple's reach, the SRD ending the grapple "if the distance ... exceeds the range" (p. 182); `GrappleEndCause` gains no member. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182; grappling.md FR-085 | unhappy (rule interaction) |
| FR-022 | Any other condition on a remaining combatant that names the leaver as its source (Charmed, Frightened) MUST be left unchanged: the SRD ends those by their own terms, not by the source leaving, so the host removes them if the effect ends. | SRD 5.2.1, "Condition", p. 179 ("Various rules define how to end a condition") | unhappy (rule interaction) |
| FR-023 | `RemovalCause` MUST be the closed set `fled`, `surrendered`, `disappeared`, `departed`; it is reported in `CombatantRemoved` and does not change any rule (`surrendered` and `fled` are not tracked states, encounter-aggregate.md FR-044). | SRD 5.2.1, "Ending Combat", p. 14 | happy |
| FR-024 | Removal MUST be refused, in this order, with `RosterEncounterEndedRejection`, `RosterUnknownCombatantRejection(id)`, `RosterCombatantIsActiveRejection(id)`, `RosterWouldEmptySideRejection(side)`. | encounter-aggregate.md FR-078; design decision | unhappy (rule interaction) |
| FR-025 | The use case `RemoveCombatantFromEncounter` MUST, after a successful removal, call `HelpBenefits.DropFor(id)` on the command's `ActionLedger` and return the new ledger and the resulting `HelpBenefitDropped` events in `ledgerEvents`, apart from the encounter events (they are not `RosterEvent`s or `EncounterEvent`s). A refusal returns the ledger it was given, equal. The Attack action's progress is untouched: the leaver is not the active combatant (FR-019), so its progress already ended with its turn (combat-action-resolution.md FR-115). | SRD 5.2.1, "Help [Action]", pp. 182-183; attack-and-help-actions.md FR-069, FR-072; combat-action-resolution.md FR-123 | happy |
| FR-026 | `CombatantRemoved` MUST carry the `EncounterID`, the round, the combatant, its `Side`, the `RemovalCause`, `kind` `combatant_removed`, `combatants` `[id]`. | combat-foundations.md FR-017 to FR-019 | happy |
| FR-027 | A creature revived while still a combatant (dead combatants stay in the order, encounter-aggregate.md FR-020) needs no roster operation: its `HitPointStatus` changes by returning-to-life.md. A creature that was removed and is then revived is added with `RosterAtInitiative` and a count the host kept, or `RosterAfterCombatant` of its reviver. | SRD 5.2.1, "Dead", p. 180; returning-to-life.md FR-003 | unhappy (rule interaction) |

#### Cases 4.3a: removal (base order A, B, C, D; B active; A, B on `sideA`, C, D on `sideB`)

| Label | Operation and state | Expected | Source | Path |
|---|---|---|---|---|
| middle, not active | remove C (`fled`) | order A, B, D; B still active, round 2; `CombatantRemoved` | FR-016 | happy |
| first, already passed | remove A (`disappeared`) | order B, C, D | FR-016 | unhappy (boundary) |
| last | remove D | order A, B, C | FR-016, FR-020 | unhappy (boundary) |
| the active combatant | remove B | `RosterCombatantIsActiveRejection(B)`; encounter equal | FR-019 | unhappy (state) |
| sole remaining member is active | remove A (applied), then remove B (active, last of `sideA`) | B refused `RosterCombatantIsActiveRejection(B)` (active check precedes would-empty) | FR-020, FR-024 | unhappy (rule interaction) |
| last of its side, not active | remove A (applied); the host advances so C is active; remove B | `RosterWouldEmptySideRejection(sideA)` | FR-020, FR-024 | unhappy (boundary) |
| two combatants, one per side | order A (`sideA`), C (`sideB`), A active; remove C | `RosterWouldEmptySideRejection(sideB)` | FR-020 | unhappy (boundary) |
| only combatant of a side, side empties | remove C then D | C applied; D refused `RosterWouldEmptySideRejection(sideB)`; status `EncounterInProgress` | FR-020 | unhappy (boundary) |
| unknown id | remove Z | `RosterUnknownCombatantRejection(Z)` | FR-024 | unhappy (invalid input) |
| ended | any removal | `RosterEncounterEndedRejection` | FR-024 | unhappy (state) |
| dead leaver | C is `Dead(monsterDeath)` | applied; no `Died`; no hit point event | FR-018 | unhappy (rule interaction) |
| leaver holds a grapple | D grappled by C; remove C | `ConditionRemoved(Grappled)` D, `GrappleEnded(D, C, outOfRange)`, then `CombatantRemoved`; D's other conditions unchanged | FR-021 | unhappy (rule interaction) |
| leaver is grappled | C grappled by D; remove C | `ConditionRemoved`, `GrappleEnded(C, D, outOfRange)`, `CombatantRemoved` | FR-021 | unhappy (rule interaction) |
| two grapples held, turn order | D and A both grappled by C; remove C | endings for A then D (turn order), then `CombatantRemoved` | FR-021 | unhappy (rule interaction) |
| Charmed by the leaver | D Charmed by C; remove C | D's Charmed kept; no condition event | FR-022 | unhappy (rule interaction) |
| spent Reaction, Dodging, readied action | C holds all three; remove C; add C again at 10 | after the add C has Reaction available, not Dodging, no readied action | FR-016, FR-017, FR-007 | unhappy (state) |
| Help benefits | ledger has C helping D and A helping C; remove C | ledger without both; two `HelpBenefitDropped` in `ledgerEvents`; refused removal leaves the ledger equal | FR-025 | happy |
| cause carried | each of four causes | event carries the cause; same order and state | FR-023 | happy |

### 4.4 Repositioning a combatant

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-028 | `Encounter.RepositionCombatant(id, position)` MUST move the combatant's entry to the place of FR-002 or FR-003 and set its count accordingly; the combatants list follows the new order, the subject's `Combatant` and every other entry are unchanged (FR-005, FR-030). The event is `CombatantRepositioned`. | SRD 5.2.1, "Controlling a Mount", p. 16 ("changes to match yours") | happy |
| FR-029 | Repositioning MUST be refused, in this order, with `RosterEncounterEndedRejection`, `RosterUnknownCombatantRejection` (the subject first, then the anchor), `RosterCombatantIsActiveRejection(id)` (the subject is active; an active anchor is allowed). `RepositionCombatantInEncounterCommand` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for `RosterAfterCombatant(anchor)` where the anchor is the subject. | SRD 5.2.1, "Your Turn", p. 13; architecture.md §6 | unhappy (rule interaction) |
| FR-030 | The subject's spent Reaction, Dodge, readied action and `MovementSpeeds` MUST be kept; only its place and count change. | SRD 5.2.1, "Reaction", p. 186 | unhappy (state) |
| FR-031 | A combatant moved from before the active combatant to after it MAY take a second turn this round, and one moved from after to before it MAY miss its turn: the engine follows the order only. This is the SRD's own outcome for a controlled mount ("can move and act even on the turn that you mount it"). When the effect ends the host repositions with `RosterAtInitiative` and the count it kept. | SRD 5.2.1, "Controlling a Mount", p. 16; "Initiative Order", p. 13 | unhappy (rule interaction) |
| FR-032 | `CombatantRepositioned` MUST carry the `EncounterID`, the round, the combatant, its new count and `precededBy` (FR-034), its `previousCount` and `previousPrecededBy` (the same two values before the operation), `kind` `combatant_repositioned`, `combatants` `[id]`. It is produced even when the order does not change; then new and previous values are equal. With the previous values an observer tells a no-op from a move and a host can apply the event to an order it holds. | combat-foundations.md FR-017 to FR-019; package design | unhappy (boundary) |

#### Cases 4.4a: reposition (base order A, B, C, D; B active)

| Label | Operation | Expected | Source | Path |
|---|---|---|---|---|
| mount after rider (active) | D after B | A, B, D, C; D count 15; D acts after B this round | FR-028, FR-031 | happy |
| back to own count | D at 5 (`afterTied`) | A, B, C, D; count 5 | FR-028 | unhappy (rule interaction) |
| no change | C at 10 (`beforeTied`) | order and count equal; `CombatantRepositioned` produced | FR-032 | unhappy (boundary) |
| moves across the active combatant | A at 3 (`afterTied`) | B, C, D, A; A may miss round 2 | FR-031 | unhappy (rule interaction) |
| subject is active | B after A | `RosterCombatantIsActiveRejection(B)` | FR-029 | unhappy (state) |
| unknown subject; unknown anchor; both | Z after A; C after Z; Z after Y | `RosterUnknownCombatantRejection(Z)`; `(Z)`; `(Z)` | FR-029 | unhappy (invalid input) |
| anchored to itself | C after C | `srderr.ErrInvalidInput` at command construction | FR-029 | unhappy (invalid input) |
| keeps state | C with spent Reaction and Dodging | both kept after the move | FR-030 | unhappy (state) |
| ended | any | `RosterEncounterEndedRejection` | FR-029 | unhappy (state) |
| subject in a tie group, before | A at (15, `beforeTied`) | A, B, C, D; A count 15; previous values equal | FR-003, FR-032 | unhappy (boundary) |
| subject in a tie group, after | A at (15, `afterTied`) | B, A, C, D (A is placed against B only) | FR-003, FR-028 | unhappy (rule interaction) |
| other subject joins a tie group | C at (15, `beforeTied`); C at (15, `afterTied`) | C, A, B, D; A, B, C, D | FR-003, FR-028 | unhappy (rule interaction) |
| dead anchor | A is `Dead`; D after A | A, D, B, C; D count 15 | FR-028 | unhappy (state) |

### 4.5 Events and domain operations

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-033 | `RosterEvent` MUST be a closed interface implementing `CombatEvent` with exactly the three events; none implements `CombatEvent` directly and none is an envelope. Each is an immutable value (unexported fields, value receivers), equal by value including `Kind()` and `Combatants()`, with a constant `Kind()` unique across the package. | combat-foundations.md FR-015 to FR-017, FR-019; shared-types-and-ownership.md FR-019 | happy |
| FR-034 | `precededBy` MUST be the `CombatantID` of the entry directly before the subject in the order after the operation, or the zero `CombatantID` when the subject is first. `count` is the subject's count after the operation. | package design | unhappy (boundary) |
| FR-035 | The events of a transition MUST be an read-only (accessors return copies) `[]CombatEvent` in the order they happened (combat-foundations.md FR-021): grapple endings (FR-021) then `CombatantRemoved`; `CombatantJoined`; `CombatantRepositioned`. Adding to the list MUST NOT change the stored list (accessors return copies). | combat-foundations.md FR-021; architecture.md §4 | unhappy (immutability) |
| FR-036 | The three `Encounter` methods MUST return a `RosterTransition` (closed, domain level, not exported): `RosterApplied(encounter, events)` or an error (encounter, rejection), where a rejected transition carries the unchanged encounter and no events. Each use case maps it to its own result. | encounter-aggregate.md FR-048, FR-089 | unhappy (state) |
| FR-037 | Every operation MUST return a new `Encounter` and leave the receiver unchanged; two encounters with the same id, status, tracker and combatants stay equal (encounter-aggregate.md FR-053) and the combatants list stays read-only (accessors return copies) and in the tracker's order. | architecture.md §3; encounter-aggregate.md FR-049, FR-052 | unhappy (immutability) |
| FR-038 | encounter-aggregate.md FR-015 is amended (A-3) to: after start only the three roster operations of this spec may change the order or the combatants list; no other transition does. `applyTurnChange` keeps rejecting a tracker whose `TurnOrder` differs (`TurnOrderChanged`, encounter-aggregate.md FR-097), so a host holding a tracker from before a roster change is refused until it takes the tracker from the new `Encounter`; this is intended and no event or result hides it. | SRD 5.2.1, "Initiative Order", p. 13; encounter-aggregate.md FR-015, FR-097 | unhappy (rule interaction) |
| FR-039 | `TurnTracker.InsertCombatant(profile, position)`, `removeCombatant(id)` and `moveCombatant(id, position)` MUST be domain operations in the library of `TurnTracker` (initiative-and-actions.md FR-123, so they can build a tracker), returning `RosterStep`: `RosterStepApplied(tracker, standing)` (`standing` is the subject's id and new count) or an error (RosterRejection). They are synchronous, pure and silent, emit no `TurnEvent`, and leave the receiver unchanged. The `Encounter` methods call them and add the encounter-level checks (FR-009, FR-019, FR-020, FR-024, FR-029). | initiative-and-actions.md FR-140; shared-types-and-ownership.md FR-044 | happy |
| FR-040 | The tracker rejections are `RosterUnknownCombatantRejection`, `RosterDuplicateCombatantRejection`; the encounter adds the others. The tracker itself MUST NOT refuse the active combatant: that is the encounter's decision (FR-019, FR-029). | package design | unhappy (rule interaction) |

### 4.6 Use cases and telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-041 | Three synchronous use cases MUST exist, each taking its `<UseCase>Command` and returning its own closed `<UseCase>Result` with `...Applied` and `...Rejected(encounter, rejection)` variants (`RemoveCombatantFromEncounter` also carries the `ActionLedger`): `AddCombatantToEncounter` (`encounter`, `combatant`, `profile`, `position`, `JoinReason`), `RemoveCombatantFromEncounter` (`encounter`, `ledger`, `combatantID`, `RemovalCause`), `RepositionCombatantInEncounter` (`encounter`, `combatantID`, `position`). No use case calls another or `AdvanceCombatTurn`. | shared-types-and-ownership.md FR-017, FR-044; combat-foundations.md FR-022, FR-023 | happy |
| FR-042 | Each use case MUST run through `Instrumenter` (combat-foundations.md FR-035) with the operations `add_combatant_to_encounter`, `remove_combatant_from_encounter`, `reposition_combatant_in_encounter`, recording the shared span, `operations`, `operation_duration` and `events` signals only. Classification and `srd5e.combat.result` names are closed sets: `applied` (class `applied`); `encounter_already_ended`, `combatant_is_active`, `would_empty_side` (class `refused`); `unknown_combatant`, `duplicate_combatant` (class `invalid`: host input the engine cannot use; a duplicate id is invalid input as at start, encounter-aggregate.md FR-012, FR-058 extends the class to ids the engine cannot find). A name appears only where its rejection can occur (table 4.6a). | combat-foundations.md FR-028 to FR-031; encounter-aggregate.md FR-058, FR-059 | unhappy (rule interaction) |
| FR-043 | Each call, refused ones included, MUST add the extra attribute `srd5e.combat.roster_reason` through `Instrumenter`'s `attributes` (combat-foundations.md FR-041). The use case builds the `attributes` closure from the command (the captured `JoinReason` or `RemovalCause`), not from the result, so a `...Rejected` result need not carry it. The value is the `JoinReason` for an add, the `RemovalCause` for a removal, `repositioned` for a reposition. The closed set is `summoned`, `arrived`, `fled`, `surrendered`, `disappeared`, `departed`, `repositioned`. No attribute carries an id, count or free text. The key is a constant in `CombatSignals` (combat-foundations.md FR-027). | combat-foundations.md FR-033, FR-041; observability.md §5 | unhappy (invalid input) |
| FR-044 | Events MUST be counted once each by `srd5e.combat.events` under their `kind` (wrapped grapple and condition events under the wrapped kind, combat-foundations.md FR-038); a rejection counts none. `HelpBenefitDropped` events in `ledgerEvents` are not counted (they are not in the event list, FR-025). | combat-foundations.md FR-029, FR-038 | unhappy (boundary) |
| FR-045 | The three use cases, their commands, results and variants, `RosterPosition` with its variants, `TiePlacement`, `JoinReason`, `RemovalCause`, `RosterEvent` with its events, `RosterRejection` with its variants and `RosterStep` MUST be exported once each from `combat/doc.go` with a godoc comment. `RosterTransition` is not exported. Rejection classes end in `Rejection`; no generic name (`Applied`, `Rejected`, `Joined`) is exported. | shared-types-and-ownership.md FR-017, FR-021; combat-foundations.md FR-022 to FR-024 | unhappy (name collision) |
| FR-046 | The domain types MUST NOT import `context`, `time`, `math/rand` or `telemetry` (they are domain files; only the use-case files in `combat` import `context` and `telemetry`), and MUST NOT read a clock or random source. No port is added; no `*srderr.PortError` is returned. A panic from domain code propagates unchanged. | architecture.md §3, §9; combat-foundations.md FR-035 | happy |
| FR-047 | Fixtures MUST use synthetic combatants; no code, test or doc here names a consuming project or holds material from outside SRD 5.2.1. | AGENTS.md, "Hard constraints" | happy |
| FR-048 | A departure MUST NOT touch effects in `ActiveEffects`/`OngoingEffects` (active-effects.md) that name the leaver as source or target, nor another combatant's `ReadiedAction`; the host ends them by the owning spec's rules (active-effects.md for effects; initiative-and-actions.md FR-038 for readied actions, which expire at their owner's next turn). A readied trigger is a host id with no description (initiative-and-actions.md FR-137), so one that concerns a departed combatant is simply never reported by the host and expires unused. Only the leaver's own readied action leaves with it (FR-016). | SRD 5.2.1, "Ready [Action]", p. 186; package design (the engine holds no link from a trigger to a combatant) | unhappy (rule interaction) |
| FR-049 | A removal MUST NOT touch `Containments` (monster-engulf-and-swallow.md), which the encounter does not hold; `CombatantRemoved` is the fact the host feeds to `ReleaseOccupants`. For a leaver that is an occupant the host passes `OccupantGone(occupant)` (monster-engulf-and-swallow.md FR-060), so no containment record outlives it. For a leaver that is a container whose death caused the removal, the host passes `ContainerDied(container)` (its FR-057); for a living container that left (`fled`, `disappeared`, `departed`), `ContainerDied` is wrong because its death rules (escape from a corpse) do not apply, so the host resolves each occupant itself, normally `OccupantGone` for any occupant also leaving. No use case here calls `ReleaseOccupants` (FR-041; monster-engulf-and-swallow.md FR-074 forbids that edge). | SRD 5.2.1, "Shambling Mound" p. 323 ("Until the grapple ends"); "Purple Worm" p. 317; package design (encounter-aggregate.md section 9 note on `ReleaseOccupants`) | unhappy (state error) |

#### Table 4.6a: results (FR-042)

| Use case | Refused | Invalid |
|---|---|---|
| `AddCombatantToEncounter` | `encounter_already_ended` | `duplicate_combatant`, `unknown_combatant` |
| `RemoveCombatantFromEncounter` | `encounter_already_ended`, `combatant_is_active`, `would_empty_side` | `unknown_combatant` |
| `RepositionCombatantInEncounter` | `encounter_already_ended`, `combatant_is_active` | `unknown_combatant` |

#### Withdrawn ids

| Id | Status |
|---|---|
| none | No id is withdrawn. |

## 5. Model

- **Value objects:** closed `RosterPosition` (`RosterAfterCombatant(anchor)`, `RosterAtInitiative(count, tiePlacement)`), `TiePlacement`, `JoinReason`, `RemovalCause`, closed `RosterRejection` (interface with an unexported marker method) with five variant structs (FR-009, FR-019, FR-020), `RosterStep` (FR-039). Reused, not defined: `CombatantID`, `Combatant`, `CombatantProfile`, `Side`, `InitiativeCount`, `InitiativeStanding`, `TurnOrder`, `TurnTracker`, `EncounterID`, `ActionLedger`, `HelpBenefits`, `HelpBenefitDropped`, `GrappleRules`, `GrappleEndCause`.
- **Events:** closed `RosterEvent` (FR-033): `CombatantJoined(encounterID, round, combatant, side, reason, count, precededBy)`, `CombatantRemoved(encounterID, round, combatant, side, cause)`, `CombatantRepositioned(encounterID, round, combatant, count, precededBy, previousCount, previousPrecededBy)`.
- **Aggregates:** none new. `Encounter` gains three methods and `TurnTracker` three operations; the behaviour is specified here, the code lives in their files of package `combat` (unexported fields stay reachable because Go privacy is per package).- **Invariants:** counts non-increasing, ids unique (FR-004): the tracker operations; both sides non-empty after a removal (FR-020) and a non-active subject (FR-019, FR-029): the encounter; `profile.id` equals the combatant's id (FR-010): the command.
- **Failures:** every refusal is a result (FR-009, FR-024, FR-029); `srderr.ErrInvalidInput` only for a malformed command (FR-010, FR-029); no `*srderr.PortError`.
- **Purity:** synchronous, deterministic, no port.

## 6. Interface

| Use case | Constructor (`NewX(opts ...Option)`) options | Command (struct fields) |
|---|---|---|
| `AddCombatantToEncounter` | `telemetry.Telemetry` (`WithTelemetry`; default `telemetry.Noop`) | `encounter`, `combatant`, `profile`, `position`, `reason` |
| `RemoveCombatantFromEncounter` | same | `encounter`, `ledger`, `combatantID`, `cause` |
| `RepositionCombatantInEncounter` | same | `encounter`, `combatantID`, `position` |

- Results: `AddCombatantToEncounterApplied(encounter, events)`, `RemoveCombatantFromEncounterApplied(encounter, ledger, events, ledgerEvents)`, `RepositionCombatantInEncounterApplied(encounter, events)`; each `...Rejected` holds the unchanged encounter (and ledger) and the `RosterRejection`.
- **Why no other port:** the engine rolls nothing and reads no content; the host supplies the count and the creature. Expected failures are results; there is no port failure contract and no test double or contract suite to add.
- **Exports:** FR-045. Telemetry constants (three operations, spans, the `roster_reason` attribute and values) are added to `CombatSignals`.
- The host stores the `Encounter` and `ActionLedger` between calls. A host that wants shared-space Prone, review prompts or ledger expiry at a turn start calls `AdvanceCombatTurn` before removing the active combatant (FR-019).

## 7. Observability

1. **What can go wrong?** A host that adds an id twice, names a missing anchor or combatant, acts on an ended encounter, or tries to remove the active combatant or a side's last combatant: results, not errors. A malformed command returns an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. 2. **How would an operator know?** `srd5e.combat.operations` by operation, `result` and `result_class`: `invalid` (`duplicate_combatant`, `unknown_combatant`) sets span error status; `refused` does not. 3. **What is needed to diagnose?** Operation, result name, class, event kinds and `srd5e.combat.roster_reason`; ids stay out of attributes. 4. **What is counted or timed?** Calls, events by kind (`combatant_joined`, `combatant_removed`, `combatant_repositioned`) and duration, all shared; the reason is an extra attribute on the shared signals. 5. **Is existing instrumentation still correct?** Yes; use cases do not nest and the domain is silent, so nothing is counted twice. 6. **What should a consumer monitor?** A rising `invalid` share (desynchronised host), `would_empty_side` refusals (host should end the encounter), the mix of `roster_reason`. 7. **Is anything now noisy?** No; no logs (combat-foundations.md FR-034), counters per call.

The domain is silent by design; telemetry is produced only in the use cases (observability.md §3).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.add_combatant_to_encounter` | span | shared set, `srd5e.combat.roster_reason` | One add. | Duration with the shared histogram. |
| `srd5e.combat.remove_combatant_from_encounter` | span | shared set, `srd5e.combat.roster_reason` | One removal. | `would_empty_side` and `combatant_is_active` refusals. |
| `srd5e.combat.reposition_combatant_in_encounter` | span | shared set, `srd5e.combat.roster_reason` | One reposition. | Duration. |
| `srd5e.combat.roster_reason` | attribute (extra on `operations` and the spans above) | `summoned`, `arrived`, `fled`, `surrendered`, `disappeared`, `departed`, `repositioned` | Why the roster changed. | Share of `fled` and `surrendered` against `summoned`. |

These rows feed `doc/observability/signals.md`; the Attributes cell of the shared `srd5e.combat.operations` row lists the extra key (amendment A-5).

## 8. Test plan

Doubles: the recording `telemetry.Telemetry` double from the `<pkg>test` packages. Domain tests build trackers through `InitiativeOrdering.Establish` (initiative-and-actions.md FR-123) and encounters with `Encounter.Start`; the ledger comes from `ActionLedger`'s constructor. No dice, ports or contract suites.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-004 | unit | Table 4.1a, one test per row; equality and hash for both variants; counts non-increasing and ids unique after every sequence of operations (property-style loop over the table's results); same inputs, same order twice. |
| FR-005 to FR-007 | unit | Round, active combatant, resources, other entries' Reaction, Dodge, readied action and speeds equal before and after every operation; a joiner before the active combatant is skipped until the wrap (advance through two rounds), one after it acts this round; fresh joiner state. |
| FR-008 to FR-011 | unit | Table 4.2a; combatants list in tracker order; status unchanged; `srderr.ErrInvalidInput` for a mismatched profile; every `LifeState` and a `sideB` joiner. |
| FR-012 to FR-015 | unit, application | Event fields and `kind`; summoned placement after an Incapacitated summoner; `fromRoll` count placed; no `dice.Roller` argument exists (source scan: no port imported). |
| FR-016 to FR-018, FR-023, FR-024 | unit | Table 4.3a rows for removal, free id, dead leaver, four causes, refusal precedence (one test per ordered pair). |
| FR-019, FR-020 | unit | Active refused; last of a side refused; encounter still in progress; `end` with `EndReasonSideDefeated` then succeeds. |
| FR-021, FR-022 | unit | Grapple rows: event order and fields exactly, the same events as `GrappleRules.EndOnSet` for the same inputs; Charmed kept. |
| FR-025 | application | Ledger rows; refusal leaves ledger and encounter equal; `ledgerEvents` equals `dropFor` events; encounter events exclude them. |
| FR-026, FR-027 | unit | Event fields; remove, revive, add scenario with a kept count. |
| FR-028 to FR-032 | unit | Table 4.4a, one test per row; state kept; event produced for no change; `srderr.ErrInvalidInput` for self anchor. |
| FR-033 to FR-038 | unit, architecture | `precededBy` first and middle; mutating a returned events slice leaves the stored events unchanged; value equality; closed family, constant unique kinds, immutable values, no direct `CombatEvent` implementer; `RosterTransition` not exported; receiver unchanged; encounter-aggregate.md FR-053 equality holds across a roster change; no other method reorders; `applyTurnChange` with a pre-add tracker gives `TurnOrderChanged`. |
| FR-039, FR-040 | unit | Tracker operations alone: insert, remove, move over table 4.1a; no `TurnEvent`; duplicate and unknown refused; active not refused at tracker level. |
| FR-041 to FR-044 | application | Table 4.6a per result: operation, result name, class, no `errors` increment, span error status only for `invalid`; `roster_reason` on every call, a refused one included (built from the command); one `events` increment per event under the wrapped kind; results equal with no-op and recording `telemetry.Telemetry`; a returning double propagates the error; no id equals an attribute value. |
| FR-048, FR-049 | unit | Effect registries and another combatant's readied action equal before and after a removal; after removing an occupant and a container, the encounter equals one built from the same fields (no containment state held), no `Containments` or `ReleaseOccupants` argument or import exists in the use cases (source scan). |
| FR-045 to FR-047 | architecture | Package scan: names unique, suffixes, no generic names, `RosterTransition` unexported; domain import rules; no consumer name in fixtures (review rule). |
| errors-and-results.md FR-031, FR-034, FR-051 | application | n/a — the use case calls no port (no `dice.Roller` argument exists, FR-014), so there is no port failure to propagate or record. |

- **Scenario test** in `scenarios/` (package `scenarios_test`) through the public packages only: four combatants start; a summoner's creature is added after it and acts after it in round 1; a latecomer joins before the active combatant and first acts in round 2; a combatant that holds a grapple leaves (grapple ends with `GrappleEnded`); its Help benefit is dropped from the ledger; the last combatant of a side cannot leave and the host ends the encounter.
- **Mutation testing:** the roster code in `combat` (placement, refusals, `precededBy`), `gomu` score 80 or more, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors (`doc/rules/testing.md` §8); the use-case files SHOULD be included. Boundary rows (first, last, equal counts, count 5 and -3) kill comparison mutants.
- Every FR id appears above; none is deliberately uncovered.

## 9. Open questions and decisions

No open questions remain; none is escalated. Settled, one line each:

- **D-1 Tie placement of a latecomer.** The SRD leaves ties to the GM and players (p. 13); `TiePlacement` carries the host's decision, with `RosterAfterCombatant` for an exact place. Web research found no official rule for late arrivals, only table practice; the engine follows the order and does not invent a convention (FR-003, FR-006).
- **D-2 Active combatant cannot leave or move.** Ending a turn has duties (shared-space Prone, Help expiry, review prompts) that live in `AdvanceCombatTurn`; refusing keeps one path for them (FR-019, FR-029). A summon that vanishes on its own turn is removed after the turn ends.
- **D-3 Emptying a side.** Refused; ending is the host's call (FR-020; encounter-aggregate.md FR-044). This keeps a non-empty order and both sides present (encounter-aggregate.md FR-011).
- **D-4 Departing grapples.** Ended as `outOfRange` (FR-021) rather than a new `GrappleEndCause` (grappling.md FR-085 stays exact); interpretation from p. 182 and grappling.md Q-10.
- **D-5 Conditions sourced by a leaver.** Left in place (FR-022); the SRD ends them by their own rules (p. 179).
- **D-6 Revival.** Needs no roster change while the body stays in the order (FR-027).
- **D-7 One entry per combatant.** A second turn per round (Thief's Reflexes, p. 64) is not a roster feature; a class-feature spec would extend the tracker.
- **D-8 Mounts.** A controlled mount's Initiative change is `repositionCombatant` with `RosterAfterCombatant(rider)` (resolves mounted-and-underwater-combat.md F-2 for the order; the rider/mount status stays the host's, its Q-2).

### Amendments to other documents

Made together with this spec by the owners of those files; this spec is not valid until they hold.

- **A-1** shared-types-and-ownership.md section 4.11: add `combatant-roster-changes` at rank 30 (after `combat-action-resolution`, 29; it imports only `ActionLedger`, its FR-112); add its row to `documentation_index.md`, its section 7 rows to `observability/signals.md` and an `Unreleased` entry (section 10) to `CHANGELOG.md`.
- **A-2** Withdrawn before use: `TurnOrder` already holds ordered `InitiativeStanding(id, count)` (initiative-and-actions.md), so it needs no change.
- **A-3** encounter-aggregate.md FR-015 becomes "only the three roster operations of combatant-roster-changes.md change the order"; its "Out of scope" note and the "combatant list never changes" row of its invariants table point here.
- **A-4** initiative-and-actions.md: FR-123 and FR-140 list `insertCombatant`, `removeCombatant`, `moveCombatant` and `RosterStep` among the `TurnTracker` library members and drop the "later spec" workaround from the settled list; combat-action-resolution.md FR-123 and Q-24 say `RemoveCombatantFromEncounter` drops the leaver's Help benefits; conditions.md D-12 points here.
- **A-5** `observability/signals.md`: the Attributes cell of the shared `srd5e.combat.operations` row lists `srd5e.combat.roster_reason`; `CombatSignals` gets the three operation names, spans and attribute constants (combat-foundations.md FR-027).

## 10. Changelog impact

Under `Unreleased`, **Added**: the `combat` context's combatant roster changes (`AddCombatantToEncounter`, `RemoveCombatantFromEncounter`, `RepositionCombatantInEncounter`, `RosterPosition`, `RosterEvent`). Not **BREAKING**: before 0.1.0 and additive; `TurnOrder` is unchanged and existing behaviour does not change.
