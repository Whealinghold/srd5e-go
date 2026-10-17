# Mounted Combat and Fire Resistance underwater

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md`, `d20-tests.md`, `resolve-d20-test-use-case.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `observability-ports.md`, `errors-and-results.md`, `content-schema-and-decoder.md` |
| Deviations from the decisions | None. |

Prose notes on dependencies (not part of the cell). Binding: `shared-types-and-ownership.md` (applied: FR-001, FR-002, FR-005, FR-012 to FR-015, FR-017 to FR-019, FR-021, FR-023, FR-026, FR-036 to FR-041, FR-044, FR-046, FR-048, FR-051, FR-052). This spec (rank 24) uses no type of `initiative-and-actions.md` (rank 25); the tracker is NOT used (D-4). `content` vocabulary used: `CreatureSize`, `DamageType`, `ConditionKind` (`content-schema-and-decoder.md`, `shared-types-and-ownership.md` FR-005, FR-046). Referenced, not depended on: `attacks.md` (Impeded Weapons and its own `underwater` fact, D-5), `content-equipment.md` (Mounts and Other Animals and Saddle rows), `environment-hazards-and-objects.md` (Q-17). `encounter-aggregate.md` and `combat-action-resolution.md` are not dependencies (`combat-foundations.md` FR-026); they appear only as targets of follow-ups in section 6.

## 1. Purpose and scope

The Mounted Combat rules (SRD 5.2.1, pp. 15 to 16) and the one Underwater Combat rule no other spec owns, Fire Resistance (p. 16):

- **Mounting and Dismounting.** Whether a creature can serve as a mount, and the movement a rider spends to mount or dismount (half the rider's Speed, rounded down).
- **Controlling a Mount.** A controlled mount's Initiative matches its rider's, it moves on the rider's turn and has only Dash, Disengage and Dodge; an independent mount keeps its place in the Initiative order.
- **Falling Off.** A DC 10 Dexterity saving throw when an effect is about to move the mount against its will, or the rider or mount is knocked Prone; a failure ends the ride and lands the rider Prone.
- **Fire Resistance underwater.** A pure rule turning "this creature is underwater" into Resistance to Fire in its `DamageAffinities`.

State in, new state and events out (`doc/rules/architecture.md` §4). The engine models no map: the host reports every positional fact (distance, unoccupied space, underwater). The mounting use cases take no `TurnTracker`; they return the movement cost as data and the caller charges it (D-4).

**Ownership.** Decisions: `shared-types-and-ownership.md` (wins over this spec).

- Owned here: `RiderStatus` (`NotMounted`, `MountedOn`), `MountControl`, `MountFacts` and its enums, `MountingTurnFacts`, `RiderTurnState`, `MountInitiativeChange`, `MountTurn`, `FallingOffTrigger`, `LandingSpace`, `UnderwaterStatus`, `UnderwaterResistanceGrant`, the pure functions (the unexported mounting rule functions, `FallingOffDifficultyClass`, `UnderwaterFireResistanceAffinities`), the use cases `MountCreature`, `DismountCreature`, `ResolveFallingOff`, the events `CreatureMounted`, `CreatureDismounted`, `FellOffMount` (closed `MountEvent`), `FallingOffSave`, and the `Mounting<Reason>Rejection` types with closed `MountRejection`, `DismountRejection`, `FallingOffRejection`.
- Referenced, never redefined: `CombatantID` (`combat-foundations.md`); `CreatureSize`, `DamageType`, `ConditionKind` (`content`); `DamageAffinities`, `DamageScope` (`hit-points-and-damage.md`); the `d20` rules functions, `Roll`, `CheckResult`, `ForcedFailureResult`, `ForcedFailureReason`, `DifficultyClass`, `RollModifiers` (`d20-tests.md`); `SaveAttempt` with `RollAttempt`, `ChooseToFail`, `AutomaticFailure` (`resolve-d20-test-use-case.md`); `RollMode` (`modifiers-and-advantage.md`).
- Impeded Weapons is `attacks.md` FR-039 to FR-042, FR-103, FR-113. Resistance maths (halving, no stacking, order) is `hit-points-and-damage.md` FR-016 to FR-025; this spec only supplies the affinity.
- **Supersedes** (`shared-types-and-ownership.md` FR-048): the more specific spec for Mounted Combat and underwater Fire Resistance. No older spec holds a requirement for either (they only name Mounted Combat as excluded), so nothing is withdrawn; follow-ups F-1 to F-7 ask for edits.
- **No use case calls a use case** (`shared-types-and-ownership.md` FR-044). `ResolveFallingOff` composes the pure `d20` rules functions (`ResolveCheck`, `FailWithoutRoll`) (`d20-tests.md` FR-069) and gets the d20s from `dice.Roller` (FR-071); it never calls `d20.Resolver`.

**Out of scope:** Impeded Weapons; Resistance maths; mount stat blocks, Speeds and Hit Points (the Mounts and Other Animals table, SRD 5.2.1 p. 100, gives only Carrying Capacity and Cost: `content-equipment.md`, `content-monsters.md`; this spec's Q-4; the host reports sizes and the rider's Speed, D-4); whether a rider's gear fits a mount's Carrying Capacity (no SRD rule, p. 100); the Military Saddle's Advantage on "any ability check you make to remain mounted" (p. 100; no rule calls for the check, Q-14); features naming a mount (Lance "unless mounted" p. 91; Phantom Steed p. 152; Find Steed p. 130 is `summoned-and-companion-creatures.md`, whose FR-018 supplies the steed's `MountFacts`); a mount at 0 Hit Points while carrying a rider and fall damage (no SRD rule on the pages read); positions, grids and choice of landing space (host); other underwater rules (Swimming, `initiative-and-actions.md` FR-077; breathing; visibility) and the object damage flow (`UnderwaterFireResistanceAffinities` works over any `DamageAffinities`, Q-17); which actions a controlled mount may take (`ControlledMountRules.permitsAction`, moved to `initiative-and-actions.md`, F-8); applying Prone (`conditions.md`); charging the `TurnTracker` and applying the Initiative change (F-1, F-2); display text.

## 2. Source

The PDF page index equals the printed page number (checked: PDF page 15 shows printed page 15 and "Mounted Combat").

- SRD 5.2.1, "Combat", "Mounted Combat", "Mounting and Dismounting", p. 15; "Controlling a Mount", "Falling Off", p. 16.
- SRD 5.2.1, "Combat", "Underwater Combat", "Impeded Weapons" (boundary only), "Fire Resistance", p. 16.
- SRD 5.2.1, "Damage and Healing", "Resistance and Vulnerability", "No Stacking", "Order of Application", "Immunity", p. 17.
- SRD 5.2.1, "Combat", "Creature Size", Creature Size and Space table, p. 14.
- SRD 5.2.1, "Equipment", "Mounts and Vehicles", "Saddles", p. 100.
- SRD 5.2.1, "Rules Glossary": "Resistance", "Round Down", "Saving Throw", p. 187; "Speed", "Size", p. 188; "Prone [Condition]", p. 186; "Unoccupied Space", p. 191.
- SRD 5.2.1, "Playing the Game", "D20 Tests", step 6 "Compare the Total to a Target Number", p. 6 (equals or exceeds). "Rolling 20 or 1", p. 7, defines natural 20 and 1 for attack rolls only.
- SRD 5.2.1, "Monsters A-Z", actions stating "Being underwater doesn't grant Resistance to this Fire damage", pp. 281 and 308.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| mount; rider | `CombatantID` each | "A willing creature that is at least one size larger than a rider and that has an appropriate anatomy can serve as a mount" (p. 15). Ids are opaque here (FR-066). |
| Willing | `MountWillingness` | `Willing`, `Unwilling`; host-reported. |
| appropriate anatomy | `MountAnatomy` | `Appropriate`, `NotAppropriate`; no SRD test, host judges (Q-16). |
| at least one size larger | `CreatureSize` order | Declared ascending order, Tiny to Gargantuan (`content-schema-and-decoder.md` FR-109; p. 14). |
| Exotic Saddle; aquatic or flying mount | `ExoticSaddle` (`Fitted`, `NotFitted`), `MountHabitat` (`Other`, `AquaticOrFlying`) | "An Exotic Saddle is required for riding an aquatic or a flying mount" (p. 100); host-reported (Q-16). |
| mounted, not mounted | `RiderStatus` (closed) | `NotMounted`, `MountedOn(mount, control)`. Held by the host or encounter (Q-2). |
| controlled / independent mount | `MountControl` | `Controlled`, `Independent` (p. 16); also the `RequestedControl` when mounting. |
| trained to accept a rider | `MountTraining` | `TrainedToAcceptRider`, `NotTrained`; host-reported. |
| Initiative of a controlled mount changes to match yours | `MountInitiativeChange` (closed) | `MountInitiativeMatchesRider(rider)`, `MountInitiativeUnchanged`. Data only (F-2, Q-2). |
| moves on your turn; retains its place in the Initiative order | `MountTurn` | `OnRidersTurn`, `OnOwnTurn`. |
| Falling Off | `FallingOffTrigger` | `MountMovedAgainstItsWill`, `RiderKnockedProne`, `MountKnockedProne` (p. 16). |
| DC 10 Dexterity saving throw | `FallingOffDifficultyClass` | `d20.DifficultyClass(10)`. |
| the save and its result | `FallingOffSave` (closed) | `FallingOffSaveRolled(CheckResult)`, `FallingOffSaveForced(ForcedFailureResult)` (FR-073). |
| unoccupied space | `LandingSpace` | `UnoccupiedSpaceWithin5Feet`, `NoUnoccupiedSpaceReported`; "no creatures are in it and it isn't completely filled by objects" (p. 191); host judges. |
| Prone | `ConditionKind` Prone | The rider "lands with the Prone condition" (p. 16); reported as the kind to apply. |
| half your Speed; whose turn; movement left | `MountingTurnFacts` (`RiderTurn`: `RiderTurnState` `Active`/`NotActive`; `WalkingSpeedFeet`; `WalkingMovementLeftFeet`) | Derived by the caller from its tracker (D-4; `initiative-and-actions.md` FR-115, FR-117). |
| underwater | `UnderwaterStatus` | `Underwater`, `NotUnderwater`; "Anything underwater has Resistance to Fire damage" (p. 16); host-reported (Q-11). |
| "Being underwater doesn't grant Resistance..." | `UnderwaterResistanceGrant` | `Granted`, `WithheldByEffect` (pp. 281, 308). |
| Resistance (to Fire) | `DamageOfType` Fire in `DamageAffinities.resistances` | Owned by `hit-points-and-damage.md`. |

Names that are not SRD terms (`RiderStatus`, `MountedOn`, `MountFacts`, the `Mount*`, `ExoticSaddle`, `MountingTurnFacts`, `RiderTurnState`, `MountEvent`, `FallingOffTrigger`, `LandingSpace`, `UnderwaterStatus`, `UnderwaterResistanceGrant`, `FallingOffSave`, the rejection classes and event names) add no rules. Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. Test references use `doc/specs/mounted-and-underwater-combat.md FR-nnn`.

### 4.1 Who can serve as a mount

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `MountCreature` MUST refuse with `MountingMountUnwillingRejection` when the host reports `MountWillingness.unwilling`, and MUST NOT refuse for that reason when `willing`. | SRD 5.2.1, "Mounted Combat", p. 15 ("A willing creature") | unhappy (state error) |
| FR-002 | `MountCreature` MUST refuse with `MountingMountNotLargerRejection` unless the mount's `CreatureSize` is at least one size larger than the rider's (`CreatureSize` order greater, the ascending order fixed by `content-schema-and-decoder.md` FR-109; no helper is defined here, Q-12). A mount several sizes larger is accepted. Both sizes are fields of `MountCreatureCommand` (`RiderSize`, `MountSize`), supplied by the caller. See cases 4.1a. | SRD 5.2.1, "Mounted Combat", p. 15; "Creature Size", p. 14 | unhappy (boundary) |
| FR-003 | `MountCreature` MUST refuse with `MountingMountAnatomyRejection` when the host reports `MountAnatomy.NotAppropriate`. | SRD 5.2.1, "Mounted Combat", p. 15 ("an appropriate anatomy") | unhappy (state error) |
| FR-004 | `MountCreature` MUST refuse with `MountingExoticSaddleRequiredRejection` when the host reports `MountHabitat.AquaticOrFlying` and `ExoticSaddle.NotFitted`, and MUST NOT when the saddle is `fitted`, or when the habitat is `other` whatever the saddle. | SRD 5.2.1, "Saddles", p. 100 | unhappy (rule interaction) |
| FR-005 | `NewMountCreatureCommand` with the same `CombatantID` for rider and mount MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`). | `doc/rules/testing.md` §4 (invalid input); `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-006 | `MountCreature` for a rider whose supplied `RiderStatus` is `MountedOn` MUST be refused with `MountingRiderAlreadyMountedRejection`. A package decision: the SRD gives no rule for a second mount (Q-13). | package design; SRD 5.2.1, "Mounting and Dismounting", p. 15 (read as one mount at a time) | unhappy (state error) |
| FR-008 | `MountCreature` MUST refuse with `MountingOutOfReachRejection` (carrying the distance) when the host-reported distance between rider and mount is more than 5 feet, and accept 0 to 5 feet. A negative distance MUST make `NewMountCreatureCommand` return an `*InvalidError`. See cases 4.1b. | SRD 5.2.1, "Mounting and Dismounting", p. 15 ("a creature that is within 5 feet of you") | unhappy (boundary) |

#### Cases 4.1a: size (one test per row; all other conditions satisfied)

| Label | Rider | Mount | Expected | Source | Path |
|---|---|---|---|---|---|
| one size larger | Medium | Large | accepted | FR-002 | happy |
| several sizes larger | Small | Huge | accepted | FR-002 | unhappy (boundary) |
| same size | Medium | Medium | refused `MountingMountNotLargerRejection` | FR-002 | unhappy (boundary) |
| one size smaller | Large | Medium | refused `MountingMountNotLargerRejection` | FR-002 | unhappy (boundary) |
| bottom of the scale | Tiny | Small | accepted | FR-002 | unhappy (boundary) |
| top of the scale, no larger exists | Gargantuan | Gargantuan | refused `MountingMountNotLargerRejection` | FR-002 | unhappy (boundary) |
| one below the top | Huge | Gargantuan | accepted | FR-002 | unhappy (boundary) |
| Tiny rider, Tiny mount | Tiny | Tiny | refused `MountingMountNotLargerRejection` | FR-002 | unhappy (empty and minimal) |

#### Cases 4.1b: distance, saddle, willingness and anatomy (one test per row)

| Label | Distance (ft) | Willing | Anatomy | Habitat, saddle | Expected | Source | Path |
|---|---|---|---|---|---|---|---|
| adjacent | 0 | Willing | Appropriate | Other, NotFitted | accepted | FR-008 | unhappy (boundary) |
| exactly 5 feet | 5 | Willing | Appropriate | Other, NotFitted | accepted | FR-008 | unhappy (boundary) |
| 6 feet | 6 | Willing | Appropriate | Other, NotFitted | refused `MountingOutOfReachRejection` (6) | FR-008 | unhappy (boundary) |
| negative distance | -1 | any | any | any | `*InvalidError` from the constructor | FR-008 | unhappy (invalid input) |
| Unwilling | 5 | Unwilling | Appropriate | Other, NotFitted | refused `MountingMountUnwillingRejection` | FR-001 | unhappy (state error) |
| anatomy | 5 | Willing | NotAppropriate | Other, NotFitted | refused `MountingMountAnatomyRejection` | FR-003 | unhappy (state error) |
| flying mount, no Exotic Saddle | 5 | Willing | Appropriate | AquaticOrFlying, NotFitted | refused `MountingExoticSaddleRequiredRejection` | FR-004 | unhappy (rule interaction) |
| flying mount, Exotic Saddle | 5 | Willing | Appropriate | AquaticOrFlying, fitted | accepted | FR-004 | happy |
| ground mount, Exotic Saddle fitted | 5 | Willing | Appropriate | Other, fitted | accepted | FR-004 | unhappy (boundary) |

### 4.2 Mounting and dismounting: when and at what cost

The caller reads three facts from its tracker and passes them as `MountingTurnFacts`: whether the rider is the active combatant, the rider's walking Speed (`initiative-and-actions.md` FR-117) and the walking movement left (walking Speed plus the Dash extra minus feet moved, FR-115). The use case returns the cost as data; the caller charges it (F-1).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-009 | `MountCreature` and `DismountCreature` MUST be refused with `MountingRiderNotActiveRejection` when `MountingTurnFacts.riderTurn` is `RiderTurnState.NotActive`: mounting and dismounting happen "during your move". Readied moves are excluded (Q-22). | SRD 5.2.1, "Mounting and Dismounting", p. 15 | unhappy (state error) |
| FR-010 | The movement cost of mounting and of dismounting MUST equal half `MountingTurnFacts.walkingSpeedFeet`, rounded down. The Speed is the walking Speed only, never the pool with the Dash extra. See cases 4.2a. | SRD 5.2.1, "Mounting and Dismounting", p. 15; "Round Down", p. 187; "Speed", p. 188 | happy |
| FR-011 | An accepted mount or dismount MUST report `MovementCostFeet` (FR-010) in its result and event, and nothing else: no `TurnTracker` and no field for an action, Bonus Action, Reaction or free interaction, so none is used. The caller charges the cost against the walking Speed plus the Dash extra, the pool a stand-up draws on (`initiative-and-actions.md` FR-091), through F-1; the engine cannot enforce this. | SRD 5.2.1, "Mounting and Dismounting", p. 15 ("During your move"; "costs an amount of movement"); "Your Turn", p. 13 | happy |
| FR-012 | When `WalkingMovementLeftFeet` is 0 or less, the use case MUST refuse with `MountingNoMovementRemainingRejection`. Above 0 and less than the cost, it MUST refuse with `MountingInsufficientMovementRejection(shortfallFeet)` (cost minus movement left). It MUST accept when movement left equals the cost. These reasons carry no segment index. A refusal MUST leave every input unchanged. Speed 0 is Q-7. See cases 4.2a. | SRD 5.2.1, "Mounting and Dismounting", p. 15; "Movement and Position", p. 14; mirrors `initiative-and-actions.md` FR-086, FR-089 | unhappy (boundary) |
| FR-013 | The cost MUST NOT include the Dash extra: after Dash, Speed 30 with 60 feet left still costs 15. | SRD 5.2.1, "Mounting and Dismounting", p. 15 ("half your Speed"); `initiative-and-actions.md` table 4.7g | unhappy (rule interaction) |
| FR-014 | An accepted `MountCreature` MUST return `RiderStatus` `MountedOn(mount, control)` with the control of FR-019 to FR-022 and one `CreatureMounted` event carrying rider, mount, control, movement cost, `MountInitiativeChange` and `MountTurn`. | SRD 5.2.1, "Mounting and Dismounting", p. 15; "Controlling a Mount", p. 16 | happy |
| FR-015 | `DismountCreature` MUST be refused with `MountingRiderNotMountedRejection` when the supplied `RiderStatus` is `NotMounted`. | SRD 5.2.1, "Mounting and Dismounting", p. 15 ("or dismount") | unhappy (state error) |
| FR-016 | An accepted `DismountCreature` MUST report the cost of FR-010 and FR-011, return `NotMounted`, and emit one `CreatureDismounted` event carrying rider, the mount it was on, the control it had and the movement cost. | SRD 5.2.1, "Mounting and Dismounting", p. 15 | happy |
| FR-017 | When several refusal reasons apply, the result MUST be the first in this order. `MountCreature`: rider not active, rider already mounted, mount unwilling, mount not larger, anatomy, Exotic Saddle required, out of reach, control not possible (FR-019), then the movement refusals (FR-012). `DismountCreature`: rider not active, rider not mounted, then the movement refusals. A refusal MUST leave every input unchanged. See cases 4.2b. | package design (the SRD gives no order); `doc/rules/architecture.md` §3; pattern of `initiative-and-actions.md` FR-118 | unhappy (rule interaction) |
| FR-018 | Mounting and dismounting MUST NOT depend on the rider's `ConditionActionAvailability`: `MountCreatureCommand`, `DismountCreatureCommand` and `MountingTurnFacts` MUST declare no field of type `ConditionActionAvailability`, `ConditionSet` or `ConditionKind`, because both are movement and Incapacitated only removes actions, Bonus Actions and Reactions. Satisfied by absence and checked by an architecture source scan (as FR-050), not a behaviour row. | SRD 5.2.1, "Mounting and Dismounting", p. 15; "Incapacitated [Condition]", "Inactive", p. 184 | unhappy (rule interaction) |
| FR-074 | `NewDismountCreatureCommand` MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`) when the rider equals the mount of a supplied `MountedOn`, so `CreatureDismounted.Combatants()` is duplicate-free (FR-068). | `combat-foundations.md` FR-018; `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-066 | The mounting use cases MUST NOT take, return or read a `TurnTracker`, and MUST NOT refuse or return an error for a `CombatantID` they do not recognise: ids are opaque. Checking that rider and mount are in the tracker (`initiative-and-actions.md` FR-119 `unknownCombatant`) is the caller's step. | package design (D-4); `combat-foundations.md` FR-026 | unhappy (invalid input) |

#### Cases 4.2a: cost and affordability (one test per row; walking movement left already includes any Dash extra)

| Label | Walking Speed | Walking movement left | Operation | Expected | Source | Path |
|---|---|---|---|---|---|---|
| the SRD's example | 30 | 30 | mount | accepted, cost 15 | FR-010 | happy |
| dismount, same cost | 30 | 30 | dismount | accepted, cost 15 | FR-016 | happy |
| odd Speed rounds down | 25 | 25 | mount | accepted, cost 12 | FR-010 | unhappy (boundary) |
| Speed 1 costs nothing | 1 | 1 | mount | accepted, cost 0 | FR-010 | unhappy (boundary) |
| Speed 5 | 5 | 5 | mount | accepted, cost 2 | FR-010 | unhappy (boundary) |
| exactly enough left | 30 | 15 | mount | accepted, cost 15 | FR-012 | unhappy (boundary) |
| one short | 30 | 14 | mount | refused `MountingInsufficientMovementRejection`, shortfall 1 | FR-012 | unhappy (boundary) |
| nothing left | 30 | 0 | mount | refused `MountingNoMovementRemainingRejection` | FR-012 | unhappy (boundary) |
| less than nothing left (Speed lowered after moving) | 30 | -5 | mount | refused `MountingNoMovementRemainingRejection` | FR-012 | unhappy (boundary) |
| after Dash, half of Speed only | 30 | 60 | mount | accepted, cost 15 | FR-013 | unhappy (rule interaction) |
| after moving 20 | 30 | 10 | mount | refused, shortfall 5 | FR-012 | unhappy (rule interaction) |
| Speed reduced mid-turn to 20 | 20 (as reported after the update) | 20 | mount | accepted, cost 10 | FR-010 | unhappy (rule interaction) |
| Speed 0 (Q-7) | 0 | 0 | mount | refused `MountingNoMovementRemainingRejection` | FR-012 | unhappy (boundary) |
| dismount, one short | 30 | 14 | dismount | refused `MountingInsufficientMovementRejection`, shortfall 1 | FR-012 | unhappy (boundary) |
| negative Speed | -1 | 0 | mount | `*InvalidError` from `NewMountingTurnFacts` | FR-052 | unhappy (invalid input) |
| dismount, rider is its own mount | 30 | 30 | dismount | `*InvalidError` from `NewDismountCreatureCommand` | FR-074 | unhappy (invalid input) |
| rider not active | 30 | 30 | mount | refused `MountingRiderNotActiveRejection` | FR-009 | unhappy (state error) |
| rider not active, dismount | 30 | 30 | dismount | refused `MountingRiderNotActiveRejection` | FR-009 | unhappy (state error) |

#### Cases 4.2b: refusal order (one test per row)

| Label | Conditions that hold | Expected (first in the order) | Source | Path |
|---|---|---|---|---|
| not active beats the mount's facts | not active, unwilling | `MountingRiderNotActiveRejection` | FR-009, FR-017 | unhappy (rule interaction) |
| not active beats already mounted | not active, already mounted | `MountingRiderNotActiveRejection` | FR-009, FR-017 | unhappy (rule interaction) |
| already mounted beats unwilling | already mounted, unwilling | `MountingRiderAlreadyMountedRejection` | FR-006, FR-017 | unhappy (rule interaction) |
| unwilling beats size | unwilling, same size | `MountingMountUnwillingRejection` | FR-001, FR-017 | unhappy (rule interaction) |
| size beats anatomy | same size, NotAppropriate | `MountingMountNotLargerRejection` | FR-002, FR-017 | unhappy (rule interaction) |
| anatomy beats saddle | NotAppropriate, flying, NotFitted | `MountingMountAnatomyRejection` | FR-003, FR-017 | unhappy (rule interaction) |
| saddle beats reach | flying, NotFitted, 6 feet | `MountingExoticSaddleRequiredRejection` | FR-004, FR-017 | unhappy (rule interaction) |
| reach beats control | 6 feet, requested controlled, NotTrained | `MountingOutOfReachRejection` | FR-008, FR-017 | unhappy (rule interaction) |
| control beats movement | requested controlled, NotTrained, no movement left | `MountingMountCannotBeControlledRejection` | FR-019, FR-017 | unhappy (rule interaction) |
| dismount: not active beats not mounted | not active, not mounted | `MountingRiderNotActiveRejection` | FR-009, FR-017 | unhappy (rule interaction) |
| dismount: not active beats movement | not active, mounted, no movement left | `MountingRiderNotActiveRejection` | FR-009, FR-017 | unhappy (rule interaction) |
| dismount: not mounted beats movement | active, not mounted, no movement left | `MountingRiderNotMountedRejection` | FR-015, FR-017 | unhappy (rule interaction) |
| dismount: movement last | active, mounted, one foot short | `MountingInsufficientMovementRejection` | FR-012, FR-017 | unhappy (rule interaction) |

### 4.3 Controlling a mount

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-019 | A rider that requests `MountControl.controlled` MUST be refused with `MountingMountCannotBeControlledRejection` unless the host reports `MountTraining.TrainedToAcceptRider`. See cases 4.3a. | SRD 5.2.1, "Controlling a Mount", p. 16 ("You can control a mount only if it has been trained to accept a rider") | unhappy (state error) |
| FR-020 | A rider that requests `MountControl.independent` MUST be accepted whether or not the mount is trained. | SRD 5.2.1, "Controlling a Mount", p. 16; "Mounted Combat", p. 15 | unhappy (boundary) |
| FR-021 | Mounting a controlled mount MUST report `MountInitiativeMatchesRider(rider)` and `MountTurn.onRidersTurn`. The use case returns data and reorders nothing (F-2, Q-2). | SRD 5.2.1, "Controlling a Mount", p. 16 | happy |
| FR-022 | Mounting an independent mount MUST report `MountInitiativeUnchanged` and `MountTurn.onOwnTurn`. | SRD 5.2.1, "Controlling a Mount", p. 16 | happy |
| FR-023 | Mounting a controlled mount MUST NOT be refused because of the mount's own turn (its turn passed, it used its action, movement or Reaction): it "can move and act even on the turn that you mount it", and `OnRidersTurn` is reported whatever the mount did. Satisfied by absence: `MountCreatureCommand` and `MountFacts` declare no field about the mount's turn or resources; checked by an architecture scan of declared field types (no `TurnResources`, `TurnTracker`, `ConditionActionAvailability`). | SRD 5.2.1, "Controlling a Mount", p. 16 | unhappy (rule interaction) |
| FR-027 | `ResolveFallingOff` that stays mounted (FR-030) MUST return the `RiderStatus` it was given, equal by value, so the control chosen at mounting is unchanged. Only `MountCreature` creates a `MountedOn`. | SRD 5.2.1, "Controlling a Mount", p. 16 | unhappy (state error) |

#### Cases 4.3a: control (one test per row; all other conditions satisfied)

| Label | Requested | Training | Expected | Source | Path |
|---|---|---|---|---|---|
| trained, controlled | Controlled | TrainedToAcceptRider | accepted, `MountInitiativeMatchesRider`, `OnRidersTurn` | FR-019, FR-021 | happy |
| untrained, controlled | Controlled | NotTrained | refused `MountingMountCannotBeControlledRejection` | FR-019 | unhappy (state error) |
| untrained, independent | Independent | NotTrained | accepted, `MountInitiativeUnchanged`, `OnOwnTurn` | FR-020, FR-022 | unhappy (boundary) |
| trained, independent | Independent | TrainedToAcceptRider | accepted, `MountInitiativeUnchanged`, `OnOwnTurn` | FR-020, FR-022 | unhappy (rule interaction) |

### 4.4 Falling off

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-028 | `ResolveFallingOff` MUST make one Dexterity saving throw against `FallingOffDifficultyClass` (`DifficultyClass` 10), resolving the rider's `SaveAttempt` through the pure `d20` rules functions (`ResolveCheck`, `FailWithoutRoll`) as FR-071 and FR-072 say (never `d20.Resolver`, `shared-types-and-ownership.md` FR-044), taking the attempt's `RollModifiers` and `RollCircumstance`s unchanged, so Advantage, Disadvantage and modifiers (including Dodge's Advantage on Dexterity saves, `initiative-and-actions.md` FR-055) arrive from the caller. A total that equals or exceeds the DC succeeds. A natural 1 or 20 has no special effect on a saving throw (`d20-tests.md` FR-031; `resolve-d20-test-use-case.md` FR-015); the SRD defines them for attack rolls only. See cases 4.4a. | SRD 5.2.1, "Falling Off", p. 16; "D20 Tests", step 6, p. 6; "Rolling 20 or 1", p. 7; `d20-tests.md` FR-031 | happy |
| FR-029 | The save MUST be required for each of the three triggers reported while the rider is mounted: `MountMovedAgainstItsWill`, `RiderKnockedProne`, `MountKnockedProne`. The trigger MUST NOT change the DC or the ability, only what the result and event report. | SRD 5.2.1, "Falling Off", p. 16 | happy |
| FR-030 | On a successful save the result MUST be `ResolveFallingOffStayedMounted`, the `RiderStatus` unchanged and equal by value, and no event. | SRD 5.2.1, "Falling Off", p. 16 ("you must succeed on a DC 10 Dexterity saving throw or fall off") | happy |
| FR-031 | On a failed save the result MUST be `ResolveFallingOffFellOff` with `RiderStatus.NotMounted` and one `FellOffMount` event carrying the rider (`ResolveFallingOffCommand.rider`), the mount (from `MountedOn`), the control, the trigger and the `LandingSpace`. | SRD 5.2.1, "Falling Off", p. 16 | unhappy (state error) |
| FR-032 | `FellOffMount` MUST carry the `ConditionKind` Prone as the condition the rider lands with, so the caller applies it through `conditions.md`. This spec MUST NOT apply any condition. A rider already Prone (`RiderKnockedProne`) that fails still reports Prone; `conditions.md` answers `ConditionAlreadyHeld` (its FR-018, order in FR-125). | SRD 5.2.1, "Falling Off", p. 16; "Prone [Condition]", p. 186 | unhappy (rule interaction) |
| FR-033 | The `LandingSpace` is host-supplied (no map) and MUST be carried unchanged in `FellOffMount`. With `NoUnoccupiedSpaceReported`, the save MUST still be rolled and a failure MUST still end the ride (Q-9). | SRD 5.2.1, "Falling Off", p. 16; "Unoccupied Space", p. 191 | unhappy (state error) |
| FR-034 | A `SaveAttempt` of `ChooseToFail` or `AutomaticFailure` (for example Paralyzed) MUST resolve through `d20.FailWithoutRoll` (`ChosenToFail` or `Automatic`, `d20-tests.md` FR-035, FR-036) as a failed save with no d20 rolled and no dice request, and the rider MUST fall off as in FR-031. | SRD 5.2.1, "Saving Throw", p. 187; "Paralyzed [Condition]", p. 186 | unhappy (rule interaction) |
| FR-035 | `ResolveFallingOff` for a rider whose `RiderStatus` is `NotMounted` MUST be refused with `MountingRiderNotMountedRejection`, no dice rolled and no event ("while you're on it"). | SRD 5.2.1, "Falling Off", p. 16 | unhappy (state error) |
| FR-036 | `ResolveFallingOff` MUST NOT depend on whose turn it is or any turn resource: `ResolveFallingOffCommand` and the result variants declare no field of type `TurnTracker`, `TurnResources`, `MountingTurnFacts` or `ConditionActionAvailability`. Checked by an architecture scan, not a behaviour row. | SRD 5.2.1, "Falling Off", p. 16 (no resource is named) | unhappy (rule interaction) |
| FR-037 | When the `dice.Roller` fails, `ResolveFallingOff` MUST propagate the `*srderr.PortError` unchanged (`unavailable`, or `invalidResult` from `dice.Roll`), return no result and leave the `RiderStatus` as the caller's. Recorded once, by this use case (FR-062). | `doc/rules/architecture.md` §4; `dice-and-rolls.md` FR-051, FR-052 | unhappy (collaborator failure) |
| FR-067 | `ResolveFallingOffCommand` MUST carry the rider's `CombatantID` and `NewResolveFallingOffCommand` MUST return an `*InvalidError` when the rider equals the mount of a supplied `MountedOn`. Whether the id is in the tracker is the caller's check (FR-066). | `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-071 | `ResolveFallingOff` MUST NOT take, build, hold or call `d20.Resolver` or any other use case. It composes `modifier.Resolve` (`modifiers-and-advantage.md` FR-010 to FR-017), `Roll` and the `d20` rules functions (`d20-tests.md` FR-069), supplying dice through its own `dice.Roller`. Its constructor takes a `dice.Roller` and an optional `telemetry.Telemetry` only. | `shared-types-and-ownership.md` FR-044; `d20-tests.md` FR-069; `resolve-d20-test-use-case.md` FR-057; `doc/rules/architecture.md` §4 | unhappy (rule interaction) |
| FR-072 | For a `RollAttempt`, `ResolveFallingOff` MUST: (1) call `modifier.Resolve` once on the attempt's `Circumstances`; (2) make exactly one `dice.Roll` request for `dice.MustExpression(mode.DiceCount(), dice.D20(), 0)` with modifier 0 (`1d20` in `Normal`, `2d20` in `WithAdvantage` or `WithDisadvantage`); (3) build the `Roll` with `d20.NewRoll(mode, faces)`; (4) call `d20.ResolveCheck(d20.SavingThrow, roll, attempt.Modifiers, FallingOffDifficultyClass)`. It MUST NOT re-select the kept die or compare the total itself. A request is made only for a `RollAttempt` and only after the FR-035 check passed. Mirrors `resolve-d20-test-use-case.md` FR-006, FR-007; a shared helper with `attacks.md` is Q-21. | SRD 5.2.1, "D20 Tests", p. 6; "Roll Two D20s", p. 8; `resolve-d20-test-use-case.md` FR-006, FR-007; `d20-tests.md` FR-063, FR-069 | happy |
| FR-073 | `FallingOffSave` MUST be closed with exactly two variants, `FallingOffSaveRolled(CheckResult)` and `FallingOffSaveForced(ForcedFailureResult)` (FR-034), and expose the held result's `d20.Outcome` as `outcome`. The rider stays mounted exactly when `outcome` is `success`. Equal by value to the result it holds. | `shared-types-and-ownership.md` FR-004, FR-017, FR-018; `d20-tests.md` FR-041 | happy |
| FR-038 | `FallingOffDifficultyClass` MUST be a `DifficultyClass` equal to 10, and equal by value to any `d20.DifficultyClass(10)`. | SRD 5.2.1, "Falling Off", p. 16 | happy |

#### Cases 4.4a: the save (one test per row; fixed dice; the Dexterity modifier and others are in the supplied `RollModifiers`)

| Label | Trigger | Save attempt | d20 (kept) | Modifier | Total | Expected | Source | Path |
|---|---|---|---|---|---|---|---|---|
| total equals the DC | mount moved | roll | 8 | +2 | 10 | stayed mounted | FR-028, FR-030 | unhappy (boundary) |
| one below the DC | mount moved | roll | 7 | +2 | 9 | fell off | FR-028, FR-031 | unhappy (boundary) |
| rider knocked Prone, success | rider Prone | roll | 15 | 0 | 15 | stayed mounted | FR-029, FR-030 | happy |
| mount knocked Prone, failure | mount Prone | roll | 3 | 0 | 3 | fell off, trigger `MountKnockedProne` | FR-029, FR-031 | unhappy (state error) |
| natural 20 is only a 20 | mount moved | roll | 20 | -11 | 9 | fell off | FR-028 | unhappy (boundary) |
| natural 1 is only a 1 | mount moved | roll | 1 | +9 | 10 | stayed mounted | FR-028 | unhappy (boundary) |
| Advantage takes the higher | mount moved | roll, Advantage | 4 and 14, 14 kept | 0 | 14 | stayed mounted | FR-028 | unhappy (rule interaction) |
| Disadvantage takes the lower | mount moved | roll, Disadvantage | 14 and 4, 4 kept | 0 | 4 | fell off | FR-028 | unhappy (rule interaction) |
| chosen failure | mount moved | `ChooseToFail` | none | n/a | n/a | fell off, no die rolled | FR-034 | unhappy (rule interaction) |
| automatic failure | mount moved | `AutomaticFailure` | none | n/a | n/a | fell off, no die rolled | FR-034 | unhappy (rule interaction) |
| negative modifier, minimum die | mount moved | roll | 1 | -5 | -4 | fell off | FR-028 | unhappy (boundary) |
| not mounted | any | any | none | n/a | n/a | refused `MountingRiderNotMountedRejection`, no die rolled | FR-035 | unhappy (state error) |
| landing space missing | mount moved | roll | 2 | 0 | 2 | fell off, `NoUnoccupiedSpaceReported` carried | FR-033 | unhappy (state error) |
| rider id in the event | mount moved | roll | 2 | 0 | 2 | fell off, `FellOffMount.rider` equals the command's rider, `combatants` rider then mount | FR-031, FR-068 | unhappy (state error) |
| rider is its own mount | mount moved | n/a | n/a | n/a | n/a | `*InvalidError` from the constructor | FR-067 | unhappy (invalid input) |
| independent mount | mount moved | roll | 2 | 0 | 2 | fell off, control `independent` in the event | FR-031 | unhappy (rule interaction) |

### 4.5 Fire Resistance underwater

`UnderwaterFireResistanceAffinities(base, status, grant)` is a pure function from a creature's `DamageAffinities` and two host-reported facts to the `DamageAffinities` for damage taken now. It adds one resistance and changes nothing else; halving, no stacking and order stay in `hit-points-and-damage.md`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-039 | With `UnderwaterStatus.underwater` and `UnderwaterResistanceGrant.granted` the affinities MUST include Resistance to the Fire `DamageType` (a `DamageOfType` scope), so one instance of Fire damage is halved, rounded down, by the damage pipeline. See cases 4.5a. | SRD 5.2.1, "Fire Resistance", p. 16; "Resistance and Vulnerability", p. 17 | happy |
| FR-040 | With `UnderwaterStatus.NotUnderwater` the affinities MUST equal the base by value, whatever the grant. | SRD 5.2.1, "Fire Resistance", p. 16 ("Anything underwater") | unhappy (boundary) |
| FR-041 | Underwater MUST add Resistance to Fire only: for each of the other twelve `DamageType`s the resolution MUST equal the base. Table 4.5b lists the twelve other types in one row; the test generates one test per type (`doc/rules/testing.md` §5). | SRD 5.2.1, "Fire Resistance", p. 16 | unhappy (rule interaction) |
| FR-042 | Underwater Resistance MUST NOT stack with Resistance the creature already has to Fire or to all damage: the result equals the base then, and Fire damage is halved once. | SRD 5.2.1, "No Stacking", p. 17; "Resistance", p. 187; `hit-points-and-damage.md` FR-021, FR-022 | unhappy (rule interaction) |
| FR-043 | Underwater Resistance MUST combine with the pipeline as any Resistance does: with Vulnerability to Fire both apply (halved, rounded down, then doubled); Immunity to Fire still gives 0; vulnerabilities and immunities are carried through unchanged. | SRD 5.2.1, "Order of Application", "Immunity", p. 17; `hit-points-and-damage.md` FR-019, FR-020, FR-024 | unhappy (rule interaction) |
| FR-044 | With `UnderwaterResistanceGrant.WithheldByEffect` the underwater Resistance MUST NOT be added (result equals base); a Resistance to Fire held for another reason MUST be kept. The grant is a per-damage input at the call site; who reads the stat-block sentence is Q-20. | SRD 5.2.1, "Monsters A-Z", pp. 281, 308 | unhappy (rule interaction) |
| FR-045 | The Resistance MUST be derived on each call and NOT stored: `NotUnderwater` for the same base returns the base, and two identical calls return equal results. | SRD 5.2.1, "Fire Resistance", p. 16 | unhappy (state error) |
| FR-046 | The function MUST NOT alter the base `DamageAffinities`, and the result's `Resistances` MUST be a fresh copy (mutating it changes nothing held, `hit-points-and-damage.md` FR-091). | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-047 | The status is per creature: one `Damage` applied to two creatures, one underwater and one not, MUST give each its own result and MUST NOT alter the `Damage`. | SRD 5.2.1, "Damage against Multiple Targets", p. 16; `hit-points-and-damage.md` FR-100 | unhappy (rule interaction) |
| FR-048 | Underwater, 1 Fire damage MUST become 0, 0 MUST stay 0, and an odd amount MUST round down. | SRD 5.2.1, "Resistance and Vulnerability", p. 17; "Round Down", p. 187 | unhappy (boundary) |
| FR-049 | An adapted worked example MUST hold: a creature with Vulnerability to Fire, underwater (Resistance from the water, not from Resistance to all damage), inside an aura reducing all damage by 5, taking 28 Fire damage, takes 22 (28 - 5 = 23, halved 11, doubled 22). The SRD's own example is pinned by `hit-points-and-damage.md` FR-019. | SRD 5.2.1, "Order of Application", p. 17 (adapted); `hit-points-and-damage.md` FR-019 | unhappy (rule interaction) |
| FR-070 | Underwater Resistance MUST NOT stack with Resistance to all damage in the SRD's worked-example setting: Resistance to all damage and Vulnerability to Fire, underwater or not, aura of -5, 28 Fire damage, takes 22 either way. | SRD 5.2.1, "Order of Application", "No Stacking", p. 17 | unhappy (rule interaction) |

#### Cases 4.5a: Fire damage underwater (one test per row; affinities from `UnderwaterFireResistanceAffinities` go to the internal `DamageResolver` of `hit-points-and-damage.md` (FR-016 to FR-025), not `ApplyDamage`, until F-3 lands; "base" is the creature's own affinities)

| Label | Base affinities | Status, grant | Damage | Flat adjustment | Expected damage taken | Source | Path |
|---|---|---|---|---|---|---|---|
| underwater halves Fire | none | underwater, granted | 15 Fire | 0 | 7 | FR-039 | happy |
| not underwater | none | NotUnderwater, granted | 15 Fire | 0 | 15 | FR-040 | unhappy (boundary) |
| not underwater, grant withheld | none | NotUnderwater, WithheldByEffect | 15 Fire | 0 | 15 | FR-040 | unhappy (boundary) |
| Cold unaffected | none | underwater, granted | 15 Cold | 0 | 15 | FR-041 | unhappy (rule interaction) |
| Acid unaffected | none | underwater, granted | 15 Acid | 0 | 15 | FR-041 | unhappy (rule interaction) |
| Lightning unaffected | none | underwater, granted | 15 Lightning | 0 | 15 | FR-041 | unhappy (rule interaction) |
| already resists Fire | Resistance to Fire | underwater, granted | 15 Fire | 0 | 7 (once) | FR-042 | unhappy (rule interaction) |
| already resists all damage | Resistance to all damage | underwater, granted | 15 Fire | 0 | 7 (once) | FR-042 | unhappy (rule interaction) |
| Vulnerability to Fire | Vulnerability to Fire | underwater, granted | 15 Fire | 0 | 14 (15, halved 7, doubled 14) | FR-043 | unhappy (rule interaction) |
| Immunity to Fire | Immunity to Fire | underwater, granted | 15 Fire | 0 | 0 | FR-043 | unhappy (rule interaction) |
| grant withheld by the effect | none | underwater, WithheldByEffect | 15 Fire | 0 | 15 | FR-044 | unhappy (rule interaction) |
| withheld, but resists Fire anyway | Resistance to Fire | underwater, WithheldByEffect | 15 Fire | 0 | 7 | FR-044 | unhappy (rule interaction) |
| one Fire damage | none | underwater, granted | 1 Fire | 0 | 0 | FR-048 | unhappy (boundary) |
| zero Fire damage | none | underwater, granted | 0 Fire | 0 | 0 | FR-048 | unhappy (empty and minimal) |
| odd amount rounds down | none | underwater, granted | 9 Fire | 0 | 4 | FR-048 | unhappy (boundary) |
| adapted worked example (Resistance from the water) | Vulnerability to Fire | underwater, granted | 28 Fire | -5 | 22 | FR-049 | unhappy (rule interaction) |
| SRD worked example, not underwater | Resistance to all damage, Vulnerability to Fire | NotUnderwater, granted | 28 Fire | -5 | 22 | FR-070, `hit-points-and-damage.md` FR-019 | unhappy (rule interaction) |
| SRD worked example, underwater: no stacking | Resistance to all damage, Vulnerability to Fire | underwater, granted | 28 Fire | -5 | 22 | FR-070 | unhappy (rule interaction) |
| leaving the water | none | underwater, then NotUnderwater | 15 Fire | 0 | 7, then 15 | FR-045 | unhappy (state error) |
| two creatures, one underwater | none and none | underwater and NotUnderwater | 15 Fire to each | 0 | 7 and 15 | FR-047 | unhappy (rule interaction) |
| base unchanged | Vulnerability to Cold | underwater, granted | n/a | n/a | base still holds only Vulnerability to Cold | FR-046 | unhappy (immutability) |

#### Cases 4.5b: every Damage Type (one test per row; base none, underwater, granted, 15 damage of the type, flat adjustment 0)

| Damage Type | Expected damage taken | Source | Path |
|---|---|---|---|
| Acid, Bludgeoning, Cold, Force, Lightning, Necrotic, Piercing, Poison, Psychic, Radiant, Slashing, Thunder (twelve rows, one test each) | 15 (same as base) | FR-041 | unhappy (rule interaction) |
| Fire | 7 (Resistance added) | FR-039, FR-041 | happy |

### 4.6 Packaging, purity and errors

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-050 | The rule functions (the unexported mounting rules, `FallingOffDifficultyClass`, `UnderwaterFireResistanceAffinities`) and value objects MUST be pure, synchronous and silent: no goroutine, `os`, randomness, clock or telemetry call, and no import of an adapter package or of `telemetry`. | `doc/rules/architecture.md` §3, §9; `doc/rules/observability.md` §3 | happy |
| FR-051 | No operation MUST alter the command or the `RiderStatus` passed in, and every collection reachable from a command, result or event MUST be a copy (mutating it changes nothing held). | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-052 | Only a malformed command or value MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`): equal rider and mount ids (FR-005), a negative distance (FR-008), a rider that is its own mount in `ResolveFallingOffCommand` (FR-067) or `DismountCreatureCommand` (FR-074), and a negative `WalkingSpeedFeet` (from `NewMountingTurnFacts`). Every other non-success MUST be an operation-prefixed result variant carrying its reason and MUST NOT be returned as an error. | `doc/rules/architecture.md` §6; `initiative-and-actions.md` FR-095 | unhappy (invalid input) |
| FR-053 | The rejection reasons MUST be three closed hierarchies with one type per reason carrying its data. `MountRejection`: `MountingRiderNotActiveRejection`, `MountingRiderAlreadyMountedRejection`, `MountingMountUnwillingRejection`, `MountingMountNotLargerRejection`, `MountingMountAnatomyRejection`, `MountingExoticSaddleRequiredRejection`, `MountingOutOfReachRejection(distanceFeet)`, `MountingMountCannotBeControlledRejection`, `MountingNoMovementRemainingRejection`, `MountingInsufficientMovementRejection(shortfallFeet)`. `DismountRejection`: `MountingRiderNotActiveRejection`, `MountingRiderNotMountedRejection`, `MountingNoMovementRemainingRejection`, `MountingInsufficientMovementRejection`. `FallingOffRejection`: `MountingRiderNotMountedRejection` only. A reason type used by several use cases implements each closed interface (same package). A reason is held as a field of each `...Rejected` variant (`shared-types-and-ownership.md` FR-018). Naming follows `combat-foundations.md` FR-023 and `initiative-and-actions.md` FR-126; the result class is `refused` (`combat-foundations.md` FR-031). The telemetry value is the reason in snake_case (FR-060). | `shared-types-and-ownership.md` FR-017, FR-018; `combat-foundations.md` FR-023, FR-031; `initiative-and-actions.md` FR-126; `doc/rules/architecture.md` §4, §6 | happy |
| FR-054 | Each use case MUST return its own closed result: `MountCreatureResult` (`MountCreatureApplied{RiderStatus, MovementCostFeet, Events}`, `MountCreatureRejected(MountRejection)`), `DismountCreatureResult` (`DismountCreatureApplied{RiderStatus, MovementCostFeet, Events}`, `DismountCreatureRejected(DismountRejection)`), `ResolveFallingOffResult` (`ResolveFallingOffStayedMounted{RiderStatus, Save}`, `ResolveFallingOffFellOff{RiderStatus, Save, Events}`, `ResolveFallingOffRejected(FallingOffRejection)`). `Save` is the `FallingOffSave` of FR-073. `Events` is a fresh `[]MountEvent` in order (`combat-foundations.md` FR-021). The names `Applied`, `Refused`, `Rejected` and `TurnStep` MUST NOT be exported. | `shared-types-and-ownership.md` FR-017, FR-018, FR-021; `combat-foundations.md` FR-021, FR-022 | unhappy (name collision) |
| FR-055 | `CreatureMounted`, `CreatureDismounted` and `FellOffMount` MUST be past-tense immutable value objects, the only members of the closed `MountEvent`, and `MountEvent` MUST implement `CombatEvent` (`combat-foundations.md` FR-015, D-1). No event implements `CombatEvent` directly (`combat-foundations.md` FR-016). Each name is defined here once. | `shared-types-and-ownership.md` FR-019, FR-021; `combat-foundations.md` FR-015, FR-016, FR-020 | happy |
| FR-068 | Each event MUST have a constant `Kind()` matching `[a-z][a-z0-9_]*`, unique in the package: `creature_mounted`, `creature_dismounted`, `fell_off_mount`. Each event's `Combatants()` MUST be a fresh, duplicate-free slice: the rider, then the mount. | `combat-foundations.md` FR-017, FR-018 | unhappy (name collision) |
| FR-069 | Each event MUST be an immutable value (unexported fields, value receivers) equal by value including `Kind()` and `Combatants()`; no timestamp, no display text. | `combat-foundations.md` FR-019 | unhappy (immutability) |
| FR-056 | The tests MUST obtain `dicetest.FixedRoller`, `dicetest.CallbackRoller` and the recording `telemetry.Telemetry` doubles from the `dicetest` and `telemetrytest` packages and MUST NOT define a double of their own. This spec adds no port, adapter, contract suite or no-op. | `shared-types-and-ownership.md` FR-036 to FR-039 | happy |
| FR-057 | Nothing in the code, tests, fixtures or docs may name a consuming project or contain material from outside SRD 5.2.1; fixtures use synthetic combatants. The consumer-name guardrail (`internal/archtest`) checks the Go files; the fixture half is a review rule. | `AGENTS.md`, "Hard constraints" | happy |

### 4.7 Telemetry

Signals go through `CombatTelemetry` (`combat-foundations.md` FR-035); constants are added to `CombatSignals` (`combat-foundations.md` FR-027). The shared signals (span `srd5e.combat.<operation>`, counters `operations`, `events` with `srd5e.combat.event` = event `kind`, `errors`, histogram `operation_duration`; `srd5e.outcome`, `result_class`) are owned by `combat-foundations.md` FR-029 to FR-031 and not repeated. No log is emitted (`combat-foundations.md` FR-034). Domain services are silent.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-058 | Each use case MUST run through `CombatTelemetry` with operation `mount_creature`, `dismount_creature` or `resolve_falling_off`, recording the shared span, `operations`, `operation_duration` and `events` (one increment per event, none for zero events) as `combat-foundations.md` FR-029 requires. This spec MUST NOT add a counter, operation attribute or duration histogram of its own. | `combat-foundations.md` FR-028, FR-029, FR-035; `doc/rules/observability.md` §4 to §6 | happy |
| FR-059 | The classifier MUST map `MountCreatureApplied`, `DismountCreatureApplied`, `ResolveFallingOffStayedMounted` and `ResolveFallingOffFellOff` to `CombatResultClass.applied` (`success`, no error status; a failed save is normal, D-1), and every `...Rejected` variant to `refused` (`failure`, no error status). No result is `invalid`: ids are opaque (FR-066) and other malformed inputs fail in the command constructors before a use case runs (FR-052). | `combat-foundations.md` FR-031 | happy |
| FR-060 | `srd5e.combat.result` MUST be a closed set per use case. `mount_creature`: `applied`, or the reason in snake_case (FR-053 without `Mounting` and `Rejection`: `rider_not_active`, `rider_already_mounted`, `mount_unwilling`, `mount_not_larger`, `mount_anatomy`, `exotic_saddle_required`, `out_of_reach`, `mount_cannot_be_controlled`, `no_movement_remaining`, `insufficient_movement`). `dismount_creature`: `applied`, `rider_not_active`, `rider_not_mounted`, `no_movement_remaining`, `insufficient_movement`. `resolve_falling_off`: `stayed_mounted`, `fell_off`, `rider_not_mounted`. The `srd5e.combat.refusal` attribute is not used. | `combat-foundations.md` FR-030, section 5 | happy |
| FR-061 | Besides the shared attributes, the operations counter and span MUST carry `srd5e.combat.mount_control` (`controlled`, `independent`: requested control for `mount_creature`, the ride's control otherwise, absent when not mounted) and, for `resolve_falling_off` only, `srd5e.combat.falling_off_trigger` (`mount_moved_against_its_will`, `rider_knocked_prone`, `mount_knocked_prone`). Passed through the optional attributes argument of the `CombatTelemetry` run helper (`combat-foundations.md` FR-041); not a second counter. | `combat-foundations.md` FR-029, FR-033, FR-041 | happy |
| FR-062 | A `*srderr.PortError` (FR-037) MUST be recorded once, by `ResolveFallingOff`, as `combat-foundations.md` FR-032 and FR-035 require: span error status with the error and one increment of `srd5e.combat.errors` with `srd5e.combat.operation` and `error.type` (`dice_roller_error`, `shared-types-and-ownership.md` FR-041; constant in `errors-and-results.md`); NOT logged. There is no nested use case and no silenced `telemetry.Telemetry` (`shared-types-and-ownership.md` FR-044): the `d20` rules functions is pure, and `srd5e.d20_test.*` signals do not fire. A returned rejection carries no `error.type` (`shared-types-and-ownership.md` FR-051). | `combat-foundations.md` FR-032, FR-035; `shared-types-and-ownership.md` FR-041, FR-044, FR-051; `doc/rules/observability.md` §7 | unhappy (collaborator failure, telemetry on failure) |
| FR-063 | Every use case MUST return equal results with the no-op `telemetry.Telemetry` and with a recording one. | `doc/rules/observability.md` §9; `combat-foundations.md` FR-036 | happy |
| FR-064 | The one new signal, histogram `srd5e.combat.mounting_movement_spent` (unit `[ft_i]`, attribute `srd5e.combat.operation` `mount_creature` or `dismount_creature`), MUST record `MovementCostFeet` of each accepted mount or dismount (0 recorded as 0) and nothing on a refusal. It and the two attribute keys of FR-061 MUST be `CombatSignals` constants pinned by tests; no attribute MUST carry a combatant id. | `combat-foundations.md` FR-027, FR-033; `doc/rules/observability.md` §5, §9, §10 | happy |
| FR-065 | `UnderwaterFireResistanceAffinities` MUST emit nothing: its Resistance is visible in `resisted` of `srd5e.combat.damage_effect` on `srd5e.combat.damage_resolved` (`hit-points-and-damage.md` section 7). | `doc/rules/observability.md` §3 | happy |

### Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-007 | Withdrawn: moved to mounted-and-underwater-combat.md (FR-066) |
| FR-024 | Withdrawn: moved to initiative-and-actions.md (new FR requested by F-8) |
| FR-025 | Withdrawn: moved to initiative-and-actions.md (new FR requested by F-8) |
| FR-026 | Withdrawn: moved to initiative-and-actions.md (new FR requested by F-8) |

## 5. Model

- **Referenced, not defined here:** `CombatantID`, `CombatEvent`, `CombatTelemetry`, `CombatSignals`; `CreatureSize`, `DamageType`, `ConditionKind`; `DamageAffinities`, `DamageScope`; the `d20` rules functions, `Roll`, `d20.Kind`, `d20.Outcome`, `CheckResult`, `ForcedFailureResult`, `ForcedFailureReason`, `DifficultyClass`, `RollModifiers`; `SaveAttempt` (`RollAttempt`, `ChooseToFail`, `AutomaticFailure`); `RollMode`; `dice.Roller`, `*srderr.PortError`. Not used: `d20.Resolver`, `SavingThrowCommand`, `ResolveD20TestResult` (FR-071), `TurnTracker`, `CombatantProfile`, `TurnRejection` (D-4).
- **Value objects and enums** (immutable, by value): `RiderStatus` (`NotMounted`; `MountedOn(mount, control)`), `MountControl`, `MountWillingness`, `MountAnatomy`, `MountTraining`, `MountHabitat`, `ExoticSaddle`, `MountFacts` (those five fields, all enums), `RiderTurnState`, `MountingTurnFacts` (`RiderTurn`, `WalkingSpeedFeet` (negative returns an `*InvalidError` from `NewMountingTurnFacts`), `WalkingMovementLeftFeet` (0 or less is legal)), `MountInitiativeChange`, `MountTurn`, `FallingOffTrigger`, `LandingSpace`, `UnderwaterStatus`, `UnderwaterResistanceGrant`, `MountRejection`, `DismountRejection`, `FallingOffRejection` (closed, FR-053), `FallingOffSave` (FR-073).
- **Commands** (validated in constructors `NewMountCreatureCommand`, `NewDismountCreatureCommand`, `NewResolveFallingOffCommand`, each returning `(T, error)`): `MountCreatureCommand` (rider, mount, `RiderSize`, `MountSize`, `RiderStatus`, `MountFacts`, `MountingTurnFacts`, distance in feet, requested `MountControl`; rejects equal ids and a negative distance), `DismountCreatureCommand` (rider, `RiderStatus`, `MountingTurnFacts`; FR-074), `ResolveFallingOffCommand` (rider, `RiderStatus`, `FallingOffTrigger`, `SaveAttempt`, `LandingSpace`; FR-067).
- **Events:** `CreatureMounted`, `CreatureDismounted`, `FellOffMount` (closed `MountEvent` of `CombatEvent`, FR-055, FR-068).
- **Pure functions** (no service types): the unexported mounting rules (FR-001 to FR-004, FR-006, FR-008, FR-010, FR-012), `FallingOffDifficultyClass` (a constant, FR-038: the rules call no upstream rules, `doc/rules/architecture.md` §1), `UnderwaterFireResistanceAffinities` (FR-039 to FR-049, FR-070).
- **Aggregate:** none new. `RiderStatus` is held by the host or the `Encounter` (Q-2); the ride has no identity.
- **Invariants:** rider differs from mount (FR-005, FR-067, FR-074); distance 0 or more (FR-008); Speed 0 or more (FR-052). Everything else is a rule outcome and a result.
- **Failures:** results for every refusal and for a failed save (FR-031); no new `*srderr.PortError`; `*srderr.PortError` propagates (FR-037); `*InvalidError` per FR-052 only.
- **Where the save is resolved.** The use case (application) calls `d20.ResolveCheck` and `d20.FailWithoutRoll` (FR-072, FR-034) as `attacks.md` FR-002 and FR-094 do; `FallingOffDifficultyClass` holds only the DC. The only non-deterministic input, the d20, comes from the `dice.Roller` port.

## 6. Interface

Use cases are structs built by a constructor (`NewMountCreature`, `NewDismountCreature`, `NewResolveFallingOff`; functional options such as `WithTelemetry`, defaulting to the zero `telemetry.Telemetry`, `shared-types-and-ownership.md` FR-036) with one method `Run(ctx context.Context, cmd <UseCase>Command) (<UseCase>Result, error)` that takes an immutable command and returns a closed `<UseCase>Result` interface (`doc/rules/architecture.md` §4). The error is non-nil only for a port failure (a `*srderr.PortError`) or an invalid value that bypassed the command constructor; every rule outcome is a result variant.

| Use case | Constructor ports | Command | Result |
|---|---|---|---|
| `MountCreature` | none | `MountCreatureCommand` | `MountCreatureResult` |
| `DismountCreature` | none | `DismountCreatureCommand` | `DismountCreatureResult` |
| `ResolveFallingOff` | `dice.Roller` (required) | `ResolveFallingOffCommand` | `ResolveFallingOffResult` |

- **Sync.** Each `Run` is synchronous and takes `context.Context` first (it emits spans; the context is passed to telemetry only and never stored). `ResolveFallingOff` calls the synchronous `dice.Roller` through `dice.Roll`, then the pure `d20` rules functions (`ResolveCheck`, `FailWithoutRoll`) (FR-072).
- **Port failure contract.** `dice.Roller` returns an error, panics or answers a different expression; `dice.Roll` turns each into a `*srderr.PortError` (a returned error is wrapped, a panic recovered, a wrong answer a contract violation); all are returned from `Run` unchanged (FR-037). Non-success is a result variant (FR-052). No port, double or contract suite is added (FR-056).
- **The tracker boundary (D-4).** `initiative-and-actions.md` FR-123 forbids a `TurnTracker` outside its library, and it offers no profile read, movement-left query or movement-charging write. The caller passes `MountingTurnFacts` and charges `MovementCostFeet` (F-1); `MountInitiativeChange` and `MountTurn` are applied by F-2.
- **Wiring (requested of `combat-action-resolution.md`, F-4; `shared-types-and-ownership.md` FR-026).** The caller checks rider and mount are in the tracker; applies Prone from `FellOffMount` via `ConditionSet.Apply`; gives the damage path `UnderwaterFireResistanceAffinities` (F-3); builds the rider's `SaveAttempt` from `ConditionEffects.SavingThrow` (`AutomaticFailure` stays `AutomaticFailure`; `Disadvantage` becomes a `RollAttempt` with a Disadvantage circumstance). The three action options of a controlled mount are `initiative-and-actions.md`'s once F-8 lands.
- **Follow-up edits to other specs (not made here).**

| Id | File and FR | Edit |
|---|---|---|
| F-1 | `initiative-and-actions.md`, new FRs after FR-091 | Add `SpendMovement` (tracker, combatant, feet) charging a fixed cost to the active combatant's walking movement, refused for a non-active combatant (`notActiveCombatant`, FR-088) and for a cost above movement left (`insufficientMovement`, FR-086), plus read queries for `CombatantProfile` and walking movement left (FR-115). Generalises the stand-up charge of FR-091. |
| F-2 | `initiative-and-actions.md` and `encounter-aggregate.md` FR-015 | Narrow FR-015 ("no transition MUST change the order") to allow the controlled-mount Initiative change; give a controlled mount's turn a place in the rider's (Q-2). The order part is delivered by `combatant-roster-changes.md` (`RosterAfterCombatant(rider)` via `repositionCombatant`, FR-028, D-8); the rider/mount status stays the host's. |
| F-3 | `hit-points-and-damage.md`, new FR | Add optional `SituationalAffinities` to `ApplyDamageCommand`, merged under FR-021 and FR-022. Until then FR-039 is tested through `DamageResolver`. |
| F-4 | `combat-action-resolution.md` | List the wiring duties above as its own FRs. |
| F-5 | `combat-foundations.md` FR-041 | None needed (the `attributes` extension exists). Row kept so the id stays stable. |
| F-6 | `shared-types-and-ownership.md` FR-015 | None needed beyond table 4.11, which ranks this spec 24, so FR-015 inserts it before `initiative-and-actions` (Q-3). |
| F-8 | `initiative-and-actions.md`, new FRs | Add `PermitsMountAction(control, choice)` over its own `ActionChoice`, using `MountControl` from this spec (rank 24 < 25): allowed for a `Controlled` mount exactly for Dash, Disengage and Dodge (SRD 5.2.1, "Controlling a Mount", p. 16, "only three action options"), forbidden for the other nine `StandardAction`s, for feature and improvised actions (Q-6); allowed for everything when `Independent`. Withdrawn FR-024 to FR-026 here carry its former cases (one test per action). |
| F-7 | `rules-glossary-coverage.md` GL-A04 | Update the Where cell to `FR-039 to FR-049, FR-070`, and add a row for Mounted Combat pp. 15 to 16 (owner this spec, FR-001 to FR-038). |
| F-9 | `environment-hazards-and-objects.md`, `content-monsters.md` | Object damage flow passes `UnderwaterStatus` to `UnderwaterFireResistanceAffinities` (Q-17); the monster damage component keeps the "Being underwater doesn't grant Resistance" sentence as a flag the host maps to `UnderwaterResistanceGrant` (Q-20). |

- **Exports.** Package `combat` exports, once each and with a doc comment (no re-export): the three use cases, commands and results with all variants; `RiderStatus` (`NotMounted`, `MountedOn`), `MountControl`, `MountFacts` and its enums, `RiderTurnState`, `MountingTurnFacts`, `MountInitiativeChange` (two variants), `MountTurn`, `FallingOffTrigger`, `LandingSpace`, `MountRejection`, `DismountRejection`, `FallingOffRejection` with all types, `FallingOffSave` (two variants), `UnderwaterFireResistanceAffinities`, `UnderwaterStatus`, `UnderwaterResistanceGrant`, `MountEvent` and the three events. The mounting rule functions stay unexported. The package comment lists each use case's inputs and outputs.

## 7. Observability

Answers to the seven questions (`doc/rules/observability.md` §2). Shared signals are not repeated (`combat-foundations.md` FR-037); this spec adds operation values, the attributes of FR-061 and one histogram.

1. **What can go wrong?** The `dice.Roller` fails (FR-037); forbidden rides (normal refusals); a host out of step with the tracker; a host reporting the wrong underwater fact (not detectable here, visible in the `damage_effect` mix); a malformed command (`*InvalidError`).
2. **How would an operator know?** Dice failure: span error status plus `srd5e.combat.errors` with the `*srderr.PortError` `error.type`. Refusals: `operations` with `result_class` `refused`, `srd5e.outcome` `failure`, reason as `srd5e.combat.result`, never error status. A failed save is `fell_off`, class `applied`.
3. **What is needed to diagnose?** Results and events carry the breakdown (d20s, modifiers, total, cost, control, trigger, landing space, refusal data). Attributes are low-cardinality; combatant ids are never attributes (FR-064).
4. **What should be counted or timed?** Calls by operation and result, events, movement spent (`mounting_movement_spent`), duration, errors.
5. **Is existing instrumentation still correct?** Yes. Movement spent mounting is not added to `srd5e.combat.turn_movement_spent` because its `srd5e.combat.movement_mode` values are SRD modes (Q-15). Fire Resistance is already visible via `damage_resolved` (FR-065).
6. **What should a consumer monitor?** Sustained `errors` for `resolve_falling_off`; a rising `refused` share with `rider_not_active` or `insufficient_movement` (host out of step); the `fell_off` share against what a DC 10 save gives; p99 of `operation_duration`.
7. **Is anything now noisy?** No. No log is emitted; counters are per call.

Allowed values: `srd5e.combat.operation` (added) `mount_creature`, `dismount_creature`, `resolve_falling_off`; `srd5e.combat.result` FR-060; `srd5e.combat.event` (added) `creature_mounted`, `creature_dismounted`, `fell_off_mount`.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.mounting_movement_spent` | histogram, unit `[ft_i]` | `srd5e.combat.operation` (`mount_creature`, `dismount_creature`) | Movement spent per accepted mount or dismount | Shift in the distribution |
| `srd5e.combat.mount_control` | attribute on the shared span and operations counter | `controlled`, `independent` | Mix of controlled and independent rides | Mix after a host change |
| `srd5e.combat.falling_off_trigger` | attribute on the shared span and operations counter (`resolve_falling_off` only) | `mount_moved_against_its_will`, `rider_knocked_prone`, `mount_knocked_prone` | How often riders save, and why | `fell_off` share per trigger |

These rows feed `doc/observability/signals.md` when the code lands.

## 8. Test plan

Levels: unit (`combat/*_test.go`, package `combat_test`, pure functions), use case (the same package: real rules, real `d20` rules functions, doubles from `dicetest` and `telemetrytest`, FR-056; command constructors are tested here), scenario (`scenarios/`, public API only), architecture (`internal/archtest`). Tables 4.1a to 4.2b, 4.4a to 4.5b are table-driven, one test per row, each row with its own requirement and path (`doc/rules/testing.md` §5); no assertion inside a loop or conditional.

Checklist (`doc/rules/testing.md` §4). Boundaries: 4.1a, 4.1b, 4.2a, 4.4a, 4.5a. Invalid input: FR-005, FR-008, FR-052, FR-067, FR-074. Empty and minimal: Tiny and zero rows, Speed 1, zero Fire damage. Rule interactions: 4.2b, Dash, Vulnerability and Immunity, No Stacking. Collaborator failure: `*srderr.PortError` both reasons (FR-037, FR-062). Missing content: not applicable (every mount fact is a host-supplied enum). State errors: FR-006, FR-015, FR-035. Determinism: fixed dice, extremes in 4.4a. Immutability: FR-046, FR-051. Telemetry on failure: FR-062.

| Ids | Level | Notes |
|---|---|---|
| FR-001, FR-003, FR-004 | unit, application | Table 4.1b rows `unwilling`, `anatomy` and the three saddle rows |
| FR-002 | unit | Table 4.1a; all six sizes |
| FR-005, FR-067, FR-074 | application | Command constructors: equal ids, rider equal to the mount (tables 4.4a, 4.2a rows) |
| FR-006, FR-015 | unit, application | Rider already `MountedOn`; dismount while `NotMounted`; refusal order rows of 4.2b |
| FR-008 | unit, application | Table 4.1b rows 0, 5, 6 and the negative distance |
| FR-009 to FR-013 | unit, application | Unit: the mounting rule functions' cost (rounded down), movement refusals and boundary rows. Application: Tables 4.2a rows (`rider not active`, cost, `Speed reduced mid-turn`, boundary, `after Dash`); result holds `MovementCostFeet` and no tracker or resource field; inputs equal before and after a refusal |
| FR-014, FR-016 | application | Accepted mount: `MountedOn`, `CreatureMounted` fields; accepted dismount: `NotMounted`, `CreatureDismounted` fields, cost |
| FR-017 | application | Table 4.2b, every row |
| FR-018, FR-023, FR-036 | architecture | Source scans of commands, `MountingTurnFacts`, `MountFacts`, `ResolveFallingOffCommand` and results for the forbidden field types (FR-018, FR-023, FR-036 lists) |
| FR-019 to FR-022 | unit, application | Table 4.3a; `MountInitiativeMatchesRider`/`OnRidersTurn` and `MountInitiativeUnchanged`/`OnOwnTurn` |
| FR-027, FR-030 | unit, application | `MountedOn` equal after a surviving save, no events; control carried into `FellOffMount` |
| FR-028, FR-029 | application | Table 4.4a with the real the `d20` rules functions and `dicetest.FixedRoller`; the use case (not the DC constant) calls the resolver with `FallingOffDifficultyClass`; supplied `RollModifiers` and circumstances reach the resolver unchanged; same DC for each trigger |
| FR-031 to FR-033 | application | Fell off: `NotMounted`, one `FellOffMount` with every field including rider id, Prone kind, no condition applied; rows `rider id in the event`, `landing space missing`, `RiderKnockedProne` |
| FR-034, FR-035 | application | Rows `chosen failure`, `automatic failure`, `not mounted`: `dicetest.FixedRoller.Requests()` empty, `FallingOffSaveForced` with `ChosenToFail`/`Automatic`, no event for a refusal |
| FR-037 | application | `dicetest.CallbackRoller` returning an error, panicking, and returning a result for another expression; each a `*srderr.PortError` (matching the cause, `ErrPortPanic` or `ErrContractViolation`), no result |
| FR-038, FR-073 | unit | `difficultyClass` equals `d20.DifficultyClass(10)`; `FallingOffSave` has exactly two variants, `outcome` is the held `d20.Outcome`, equal by value |
| FR-039 to FR-049, FR-070 | unit | Tables 4.5a and 4.5b (Fire, and one generated test per other `DamageType`) through `DamageResolver` (F-3 pending; after F-3 also `ApplyDamage`); base unchanged and a copy (mutating the returned slice changes nothing) on `resistances` mutation; equal results for equal calls |
| FR-050, FR-056, FR-057, FR-065, FR-071 | architecture | Domain-purity, observability-in-domain, consumer-name and signal guardrails cover them; doubles imported only from the `*test` packages; import scan: `ResolveFallingOff` imports no use case, constructor takes a `dice.Roller` and optional `telemetry.Telemetry` only; fixtures half of FR-057 is a review rule |
| FR-051, FR-052 | unit, application | Inputs equal before and after; copied slices; refusals asserted as results; `*InvalidError` only for FR-005, FR-008, FR-067, FR-074 and a negative Speed |
| FR-053 to FR-055, FR-068, FR-069 | unit, architecture | Each rejection type carries its data and each closed interface has exactly its variants; one-arm type switch over `ResolveFallingOffRejected`; name scan fails on a generic or duplicate exported name; exhaustive type switch; `Events` is a fresh copy; events are `MountEvent` and `CombatEvent`; constant unique `Kind()`; `Combatants()` rider then mount; immutable values, value equality |
| FR-058 to FR-061, FR-063 | unit, application | Classifier table (unit). Application: recording tracer and meter: span name, one `operations`, one duration sample, one `events` increment per event, none for a refusal; classifier table (applied vs refused, no error status); one row per FR-060 result value (closed set); `mount_control` and `falling_off_trigger` on span and counter; equal results with no-op and recording observability |
| FR-062 | application | `dicetest.CallbackRoller` throwing: one span error, one `errors` increment with `error.type` `dice_roller_error`, no log, no `srd5e.d20_test.*` signal |
| FR-064 | architecture, application | Constants checked by the signal guardrail; histogram samples including 0 and nothing on refusal; no id attribute |
| FR-066 | architecture, application | No `TurnTracker` in commands or results; an arbitrary `CombatantID` is accepted |
| FR-072 | application | `dicetest.FixedRoller.Requests()` is `[1d20]` for normal, `[2d20]` for advantage and disadvantage, exactly one request, modifier 0; none for a forced failure or a not-mounted rider; kept die follows `Roll` |

- **Scenario test (one):** through the published packages only (package `scenarios_test`). A rider with Speed 30 mounts a trained, willing Large horse during their move (cost 15, controlled, Initiative to match), a forced-movement effect triggers the DC 10 Dexterity save and a fixed roller fails it (rider Prone, no longer mounted), then `UnderwaterFireResistanceAffinities` underwater adds Resistance to Fire and on land returns the base. `ApplyDamage` joins when F-3 lands. Proves FR-010, FR-019, FR-021, FR-028, FR-031 and FR-039 together.
- **Contract suites:** none; no port is added.
- **Mutation testing:** every changed line of the pure rule functions in `combat` is mutated (`doc/rules/testing.md` §8, score at least 80; `gomu` runs only in Docker with `--cpus` and `--workers` capped at a quarter of `nproc`), in particular the size comparison, the rounded-down cost, the distance and DC boundaries, and the resistance merge. The use-case files SHOULD be.
- **Deliberately not covered:** the fixtures half of FR-057 (review only); applying the Initiative change and charging the cost to the tracker (F-1, F-2).

## 9. Open questions and decisions

No question is open or escalated. F-1 to F-8 are requests to other specs (their own files own the FRs), not open questions; each has a stated interim behaviour. Each is settled below, with the SRD page or design basis.

- Q-1: nothing needed of `TurnTracker` (D-4). Q-4: Carrying Capacity and Cost live in `content-equipment.md`, Speeds in `content-monsters.md`. Q-11: follow-up F-3 and D-3. Q-14: saddle Advantage is data only in `content-equipment.md`. Q-16: anatomy, training and habitat are host-reported enums, never inferred.
- Q-2: settled. `RiderStatus` is held by the host (or a later spec), not by this spec. A controlled mount's Initiative "changes to match yours when you mount it" (p. 16), so `encounter-aggregate.md` FR-015 is narrowed to allow exactly that change (F-2, amendment). Interpretation: the mount acts immediately after its rider among equal counts; the SRD has no tie rule beyond the GM deciding (p. 13), and "moves on your turn as you direct it" is the closest reading.
- Q-3: settled. This spec ranks 24 (table 4.11), before `initiative-and-actions` (25); `shared-types-and-ownership.md` FR-015's textual order gains the one-line insertion (F-6, amendment).
- Q-5: settled, not modelled. The SRD is silent on reverting Initiative (p. 16); the events carry the control the ride had, so a host that wants a revert has the data. Interpretation: the SRD states only the change at mounting.
- Q-6: settled. "Only three action options" (p. 16) restricts the mount's action choices: feature and improvised actions forbidden, Bonus Actions and Reactions not restricted (the sentence names action options only). Owned by `initiative-and-actions.md` (F-8).
- Q-7: settled. A rider with movement left of 0 or less is refused `MountingNoMovementRemainingRejection` (FR-012); Speed 1 costs 0 and is accepted. Basis: half of Speed, round down (p. 15) is 0, and the sibling rule `initiative-and-actions.md` FR-089.
- Q-8: settled, not modelled. Prone allows only crawling or righting oneself (p. 186); the engine has no posture, so the host refuses or the rider stands first (`conditions.md` owns Prone). Mounting is a move, not an action, so no engine-side refusal exists to add.
- Q-9: settled. With no unoccupied space within 5 feet the save is still rolled and a failure still ends the ride, reporting `noUnoccupiedSpaceReported` (FR-033); the SRD gives no alternative outcome (p. 16), so the host decides placement.
- Q-10: settled. "The same save" (p. 16) names the same DC 10 Dexterity save, not a shared result. One call is one save for the one trigger the host reports; a host whose effect both moves the mount and knocks the rider Prone makes one call per trigger it chooses to apply (interpretation: each listed event is a separate trigger; the SRD says nothing of merging them). FR-029 stands.
- Q-12: settled. FR-002 compares `CreatureSize` order (ascending order fixed by `content-schema-and-decoder.md` FR-109); no published comparison is requested.
- Q-13: settled. One `RiderStatus` per rider; a rider on a second mount is refused (FR-006, package decision, read from "a mount", p. 15); riders per mount are not counted because the SRD gives no capacity rule (Carrying Capacity, p. 100, is not modelled).
- Q-15: settled. Mounting movement is not added to `srd5e.combat.turn_movement_spent` (its `movement_mode` values are SRD modes); `mounting_movement_spent` stays (FR-064). If F-1 later makes the charge a tracker operation, that operation records the sample and this histogram is withdrawn in the same change.
- Q-17: settled. "Anything underwater" (p. 16) includes objects, so `affinities` works over any `DamageAffinities`, including an object's. `environment-hazards-and-objects.md` is asked to pass the status in its object damage flow (amendment).
- Q-20: settled. The "does not grant Resistance" sentence (pp. 281, 308) is a per-damage-entry fact. The engine takes it as the `UnderwaterResistanceGrant` input at the call (FR-044). `content-monsters.md` is asked to keep the sentence as a flag on the damage component so a host can pass it through (amendment).
- Q-21: settled. No shared one-save helper is requested: FR-072 states the steps, which is enough for this spec (the `attacks.md` repetition is that spec's call).
- Q-22: settled by interpretation. Mounting is "during your move" (p. 15). A readied move (Ready, p. 186) is a Reaction response judged by `initiative-and-actions.md` FR-057, FR-074 over `MoveSegment`s, which carry no mount step, so FR-009 keeps `RiderTurnState` to `active`/`notActive` and refuses off-turn mounting. A host wanting to mount on a readied move reports the rider `active` for that moment and charges the cost itself.

- D-1: a failed Falling Off save is a successful result (`fell_off`, class `applied`, `combat-foundations.md` FR-031).
- D-2: the use cases report what they do not own (Prone, Initiative reorder, landing space); `shared-types-and-ownership.md` principle 5.
- D-3: Fire Resistance is derived per call and never stored on `HitPointStatus` (FR-045).
- D-4: the mounting use cases take no `TurnTracker`; the caller charges the cost (F-1), because `initiative-and-actions.md` FR-123 forbids constructing one and this spec must not assume an API it lacks.
- D-5: `attacks.md` keeps its own `underwater` fact (it sits left of this spec, `combat-foundations.md` FR-026); a shared fact would move to `hit-points-and-damage.md` or `combat-foundations.md`.

## 10. Changelog impact

Under `Unreleased`, **Added**: the `combat` context's Mounted Combat and underwater Fire Resistance types and use cases (`MountCreature`, `DismountCreature`, `ResolveFallingOff`, their closed results, `RiderStatus`, `MountControl`, `MountFacts`, `MountingTurnFacts`, `UnderwaterFireResistanceAffinities`, the refusals and `MountEvent` events in section 5), exported from package `combat`, and the `srd5e.combat.mounting_movement_spent` signal and the `mount_control` and `falling_off_trigger` attributes in `doc/observability/signals.md`. Not **BREAKING**: all new API. If F-1 to F-3 change exported signatures of already-released specs, those entries are **BREAKING** in their own changelog lines.
