# Monster Swallow and Engulf

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `conditions.md`, `grappling.md`, `d20-tests.md`, `resolve-d20-test-use-case.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `content-schema-and-decoder.md`, `observability-ports.md`, `errors-and-results.md` |

Dependency notes (prose, not dependencies): rank 24 in `shared-types-and-ownership.md` section 4.11 (strictly above `grappling.md`, 23, and `conditions.md`, 22; level with `attacks.md`). It uses `CombatantID`, `CombatEvent` and the `combat` telemetry helper over `telemetry.Telemetry` of `combat-foundations.md`; `ConditionSet`, `ConditionKind`, `ConditionActionAvailability` of `conditions.md`; `Grapples`, `Grapple`, `GrappleRules`, `GrappleEndCause` of `grappling.md`; the pure `D20TestResolver`, `SaveAttempt`, `CheckAttempt` (this spec never calls `ResolveD20Test`, FR-044 of shared-types); `CreatureSize`, `DamageType`, `Ability` from `content` and the kernel. Not dependencies: `hit-points-and-damage.md` (the host applies the damage this spec reports, FR-075), `environment-hazards-and-objects.md` (the host starts and ends choking, FR-037), `encounter-aggregate.md` (rank 28; does not call this spec: the host feeds `ReleaseOccupants` from the encounter's `Died`, `GrappleEnded` and `CombatantRemoved` facts, encounter-aggregate.md section 9, combatant-roster-changes.md FR-049), `content-monsters.md` (stat-block prose stays verbatim; the host transcribes terms, Q-8), `movement-lifting-and-flight.md` and `magical-effects.md` (no rule here needs them).

## 1. Purpose and scope

The state machine for a creature held inside another: the Swallow action (Behir p. 263, Purple Worm p. 317, Remorhaz pp. 319-320, Tarrasque p. 332, Kraken p. 303, Giant Frog p. 351, Giant Toad p. 354) and the Engulf action (Gelatinous Cube p. 287, Shambling Mound p. 323) together with the same-shaped Whelm of the Water Elemental (p. 337). Grappling is owned by `grappling.md`; conditions by `conditions.md`; this spec owns only what is new: the container record, capacity and size caps, the conditions a container imposes and later removes, Total Cover inside, periodic damage due, the damage-threshold Constitution save that makes a container regurgitate, release on the container's death, and the exits.

It covers:

- `ContainmentTerms` (a stat block's numbers, host-transcribed), `Containment`, the immutable `Containments` collection and its invariants;
- the pure domain service `ContainmentRules` (cover, restrictions, release on a `ConditionSet`);
- five use cases `EnterContainer`, `ExitContainer`, `AdvanceContainerTurn`, `ResolveRegurgitation`, `ReleaseOccupants`, with telemetry through the `combat` telemetry helper over `telemetry.Telemetry`.

State in, new state and events out (architecture section 4): the host stores `Containments` beside `Grapples`.

**Out of scope:**

- **Rolling and applying damage.** This spec reports `ContainmentDamageDue` (dice, type, halved or not); the host rolls and applies it through `hit-points-and-damage.md` (FR-075).
- **Positions.** The engine models no map: the host moves occupants with the container, picks the nearest unoccupied space, and states whether a space is free (FR-031, FR-042).
- **Pulling an object out of a cube** ("a creature or an object", p. 287); the pull rule here is for creatures.
- **Damage sharing of the Animated Rug of Smothering** (p. 259) and the **attachment of the Darkmantle** (p. 279): different mechanics (Q-10).
- **Choosing the monster's action and its recharge or timing** (`monster-limited-use.md`, `initiative-and-actions.md`), the attack or save that precedes a Swallow, and the Grapple option that sets up a Swallow (`grappling.md`).
- **Anything not in SRD 5.2.1.** No missing core rule was needed; the places where the SRD is silent are marked interpretation (section 9).

## 2. Source

- SRD 5.2.1, "Swallow" actions: "Behir", p. 263; "Purple Worm", p. 317; "Remorhaz", pp. 319-320; "Tarrasque", p. 332; "Kraken", p. 303; "Giant Frog", p. 351; "Giant Toad", p. 354 (the candidate list said p. 353; the text is on p. 354, Q-9).
- SRD 5.2.1, "Engulf": "Gelatinous Cube", p. 287; "Shambling Mound", p. 323; "Water Elemental" (Whelm), p. 337.
- SRD 5.2.1, "Rules Glossary": "Condition" p. 179; "Blinded" p. 177; "Restrained" p. 187; "Prone" p. 186; "Grappled [Condition]" and "Grappling" p. 182; "Cover" p. 179 (and "Cover", p. 15); "Suffocation [Hazard]" p. 189.
- SRD 5.2.1, "Saving Throws", p. 7; "D20 Tests", p. 6; "Half Damage", p. 16; "Creature Size", p. 14.
- Package design: `doc/rules/architecture.md` sections 2-4, 6; `observability.md` sections 2-7; `testing.md` section 4.

Printed pages equal PDF pages (checked on all pages above).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Swallow, "swallowed" | `ContainmentMode.Swallowed` | Inside the monster; bonus action with a save, or an action with none (frog, toad). |
| Engulf, "engulfed" (cube) | `ContainmentMode.Engulfed` | Inside the cube's space, no grapple. |
| Engulf with the Grappled condition (mound, Whelm) | `ContainmentMode.EngulfedGrappled` | Tied to a `Grapple`: "Until the grapple ends". |
| the monster holding creatures | container | A `CombatantID`. |
| the held creature | occupant | A `CombatantID`. |
| "can have up to three creatures swallowed", "one Large creature or up to four Medium or Small" | `ContainmentCapacity` | Slots and per-size costs (FR-002). |
| "one Large or smaller creature" | `ContainmentTerms.MaxSize` | |
| "Total Cover against attacks and other effects outside" | `ContainmentCover` | FR-040. |
| "takes 30 damage or more on a single turn from a creature inside it" | `Regurgitation`, damage tally | FR-046, FR-047. |
| "regurgitate", "disgorges" | `ContainmentLeaveCause.Regurgitated`, `.disgorged` | |
| "escape from the corpse using 20 feet of movement" | `ContainerDeathRule` (`EscapeFromCorpse`) | FR-011, FR-058. |
| "is suffocating", "can't cast spells with a Verbal component", "can't teleport" | `ContainedRestrictions` | FR-039. |
| "pull a creature out of the cube" | `ExitByPull` | FR-052. |

Package names (no SRD term): `Containments`, `Containment`, `ContainmentTerms`, `EntryTerms`, `DamageTerm`, `PeriodicDamage`, `GrappleHoldTerms`, `ExitTerms`, `ContainmentRules`, `ContainmentEvent`, `ContainmentRejection`, `ContainmentProblem`, `ContainmentDamageDue`, the use cases and their results.

## 4. Requirements

Stable ids. "Container" and "occupant" are `CombatantID`s. Feet are whole feet. Sizes compare in the declaration order of `CreatureSize`. Rows citing a stat block cite its page from section 2.

### 4.1 Terms

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `ContainmentMode` MUST have exactly three members, `swallowed`, `engulfed` and `engulfedGrappled`. | SRD 5.2.1, "Behir" p. 263; "Gelatinous Cube" p. 287; "Shambling Mound" p. 323 | happy |
| FR-002 | `ContainmentCapacity(totalSlots, costs)` MUST have `totalSlots` of 1 or more and each cost between 1 and `totalSlots`; `cost(size)` MUST return the mapped cost or 1. The constructor MUST return an error matching `srderr.ErrInvalidInput` otherwise. | SRD 5.2.1, "Gelatinous Cube" p. 287; "Water Elemental" p. 337; "Purple Worm" p. 317 | unhappy (boundary) |
| FR-003 | A set of occupant sizes MUST fit when the sum of their costs is at most `totalSlots`. Cube: 4 slots, Large costs 4 (one Large or up to four Medium or Small). Whelm: 2 slots, Large costs 2. The Giant Toad: 1 slot, printed ("only one target swallowed at a time", p. 354). The Giant Frog: 1 slot is an interpretation (Q-14), derived from "the target" and the Bite lock (p. 351), which print no count. Every other block: its printed count of slots, cost 1 (table 4.7.2). | SRD 5.2.1, "Gelatinous Cube" p. 287; "Water Elemental" p. 337; "Tarrasque" p. 332; "Giant Toad" p. 354 | happy; unhappy (boundary) |
| FR-004 | `DamageTerm(expression, type)` MUST hold a `dice.Expression` with at least one die (`dice-and-rolls.md`) and a `DamageType`; the printed average is not stored. A fixed number with no dice MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Damage Notation", p. 257 | unhappy (invalid input) |
| FR-005 | `EntryTerms` MUST be a closed variant set (interface with an unexported marker method): `EntryAutomatic` (an action with no save, frog and toad) or `EntryBySave(ability, dc, damage, halfOnSuccess, relocateFeet, targetAnySize)`. `dc` MUST be 1 or more; `damage` is an optional list of `DamageTerm`; `halfOnSuccess` MUST be false when `damage` is absent; `relocateFeet`, when present, MUST be 1 or more. Violations MUST be returned by the constructor as an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Gelatinous Cube" p. 287; "Kraken" p. 303; "Water Elemental" p. 337 | unhappy (invalid input) |
| FR-006 | `PeriodicDamage(damage, timing, firstTickNextTurn)` MUST hold a non-empty list of `DamageTerm`, a `ContainmentTiming` (`startOfContainerTurn`, `endOfContainerTurn`) and a flag `firstTickNextTurn` (default false; true only for the Giant Frog, FR-078). An empty list, or the flag with `startOfContainerTurn`, MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Remorhaz" p. 320 (acid plus fire); "Giant Toad" p. 354; "Giant Frog" p. 351 | unhappy (empty and minimal) |
| FR-007 | `Regurgitation(thresholdDamage, constitutionDC, landingFeet)` MUST have each value 1 or more. It is valid only with mode `swallowed`. | SRD 5.2.1, "Purple Worm" p. 317; "Behir" p. 263 | unhappy (boundary) |
| FR-008 | `ContainerDeathRule` MUST be a closed variant set (interface with an unexported marker method): `ReleaseOnDeath` or `EscapeFromCorpse(feet)` with `feet` of 1 or more. | SRD 5.2.1, "Tarrasque" p. 332; "Giant Frog" p. 351 | unhappy (boundary) |
| FR-009 | `ExitTerms(selfEscapeDC, pullDC, pullerDamage)` MUST hold at least one DC (each 1 or more); `pullerDamage` is allowed only with `pullDC`. `GrappleHoldTerms(escapeDC, partName)` MUST have `escapeDC` of 1 or more and a non-blank trimmed name. | SRD 5.2.1, "Gelatinous Cube" p. 287; "Shambling Mound" p. 323 | unhappy (invalid input) |
| FR-010 | `ContainmentTerms(mode, maxSize, capacity, entry, imposed, totalCover, suffocation, noVerbalSpells, noTeleport, periodic, regurgitation, deathRule, exit, grapple, disgorgeAfterFirstDamage, lockedAction, actionKind)` MUST accept `imposed` only of `ConditionKind.Blinded` and `ConditionKind.Restrained` (the two the SRD imposes); `suffocation` is a `SuffocationRule` (`none`, `always`, `unlessBreathesWater`). | SRD 5.2.1, "Kraken" p. 303 (Restrained only); "Water Elemental" p. 337 | unhappy (invalid input) |
| FR-011 | `ContainmentTerms` MUST require `grapple` exactly when the mode is `engulfedGrappled`, require `regurgitation` only with `swallowed`, require `periodic` with `disgorgeAfterFirstDamage`, and a non-blank trimmed `lockedAction` when present; each violation MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Shambling Mound" p. 323; "Giant Frog" p. 351 | unhappy (rule interaction) |
| FR-012 | All terms values MUST be immutable with value equality; `NewContainmentTerms` (and the other terms' `NewX` constructors) MUST return `(T, error)` and never panic; the error is a `*ContainmentRejected` carrying `ContainmentProblem.InvalidTerms`, matches `srderr.ErrInvalidInput`, and is returned exactly for the inputs FR-002 and FR-004 to FR-011 forbid (the `MustX` forms panic only then). | `errors-and-results.md` FR-025; `combat-foundations.md` FR-005, FR-023 | unhappy (invalid input) |
| FR-013 | The terms of the ten stat blocks MUST be constructible exactly as table 4.7.1 states them (the SRD fixtures). | SRD 5.2.1, pp. 263, 287, 303, 317, 319-320, 323, 332, 337, 351, 354 | happy |

### 4.2 The `Containment` record and `Containments`

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `Containment(container, occupant, occupantSize, terms, phase, addedConditions, partOrdinal, damageFromInside, damageTicks, tickDeferred)` MUST return an error matching `srderr.ErrInvalidInput` when container equals occupant, when a counter is negative, or when `addedConditions` is not a subset of `terms.imposed`. `phase` is `inside` or `trapped(feet)`. | SRD 5.2.1, "Swallow" actions (see FR-013) | unhappy (invalid input) |
| FR-015 | The `Containments` constructor MUST return an error matching `srderr.ErrInvalidInput` when an occupant appears twice, when two records of one container have unequal terms, when the slot costs of one container's occupants exceed its capacity (FR-003), or when an occupant is itself a container (nested containment in either direction, Q-12). | SRD 5.2.1, "Gelatinous Cube" p. 287 ("at a time") | unhappy (invalid input) |
| FR-016 | `Containments` MUST copy its input, expose only copied views, compare equal (with equal comparison) when it holds equal records regardless of insertion order, and iterate in the order occupants entered. `Containments.Empty` MUST hold nothing. | `doc/rules/architecture.md` section 4 | unhappy (immutability) |
| FR-017 | `ContainerOf(occupant)` MUST return `InContainer(record)` or `NotContained` (a closed lookup, never a nil pointer); `OccupantsOf(container)` MUST return the records in entry order, empty when none; `SlotsUsed(container)` MUST return the sum of costs. | SRD 5.2.1, "Gelatinous Cube" p. 287 | happy |
| FR-018 | `enter(container, occupant, size, terms, addedConditions, partOrdinal, tickDeferred)` MUST return `ContainmentTransitionApplied` with the record in phase `inside` and the event `OccupantContained(container, occupant, mode)`. | SRD 5.2.1, "Behir" p. 263; "Gelatinous Cube" p. 287 | happy |
| FR-019 | `enter` MUST reject, in this order, with `ContainmentRejection`: `containerIsContained` (the container is itself an occupant: nested containment is not modelled, Q-12), `occupantIsContainer` (the occupant holds occupants of its own), `occupantAlreadyContained`, `tooLarge` (size above `terms.maxSize`; equal is allowed), `noRoom` (FR-003). A rejection carries no event and leaves the input unchanged. The domain always rejects; whether `EnterContainer` rolls the save first for `tooLarge` and `noRoom` is FR-021 and FR-032. | SRD 5.2.1, "Kraken" p. 303 ("If the target is Large or smaller"); "Giant Toad" p. 354; package design (order) | unhappy (rule interaction) |
| FR-020 | `partOrdinal` MUST be the lowest positive integer not used by that container's other records (so the grapple parts of FR-035 never collide); it MUST be 1 for a container with no occupants. | SRD 5.2.1, "Grappling" p. 182 ("One Grapple per Hand"); package design | unhappy (boundary) |

### 4.3 Entering

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-021 | `EnterContainer` MUST check, before any die is rolled and in this order: `containerIsContained`, `occupantIsContainer`, `occupantAlreadyContained`, then (mode `swallowed`) `notGrappledByContainer` when `Grapples.GrappleOf(occupant)` is not a grapple held by the container, then (mode `engulfedGrappled`) `occupantGrappledElsewhere` when the occupant is held by another grappler, then `tooLarge`, then `noRoom`. `tooLarge` is a pre-dice refusal only unless the entry is `EntryBySave` with `targetAnySize`; `noRoom` is a pre-dice refusal only when the entry has no damage (`EntryAutomatic`, or `EntryBySave` without `damage`, the mound's Engulf). Otherwise the save is rolled and FR-032 applies. A refusal rolls no die. | SRD 5.2.1, "Behir" p. 263 ("Grappled by the behir"); "Kraken" p. 303; "Giant Toad" p. 354; "Shambling Mound" p. 323 | unhappy (state error) |
| FR-022 | For `EntryBySave`, `EnterContainer` MUST make the occupant's saving throw by composing the pure `d20` functions with its own `dice.Roller` as `EscapeGrappleFromRecord` does (`grappling.md` FR-037), kind `savingThrow`, against `DifficultyClass(dc)`, adding no modifier of its own. A `SaveAttempt` of `ChooseToFail` or `AutomaticFailure` MUST draw no die and give a failure. `ability` labels the save and the host puts that ability's modifier in the attempt. | SRD 5.2.1, "Saving Throws" p. 7; "D20 Tests" p. 6; "Purple Worm" p. 317 | happy |
| FR-023 | On a failed save with room and size allowed, `EnterContainer` MUST return `EnterContainerContained` with the new `Containments`, the new `ConditionSet`, the `ContainmentDamageDue` for the entry damage (full), the `ContainmentEffects` of FR-037 and the events. | SRD 5.2.1, "Gelatinous Cube" p. 287 ("Failure: 10 (3d6) Acid damage, and the target is engulfed") | happy |
| FR-024 | On a successful save, `EnterContainer` MUST return `EnterContainerResisted` with the check, the unchanged `Containments`, `ConditionSet` and `Grapples`, and a damage due halved (rounded down by the host, `hit-points-and-damage.md` FR-014) when `halfOnSuccess`, else none. | SRD 5.2.1, "Gelatinous Cube" p. 287 ("Success: Half damage"); "Water Elemental" p. 337; "Half Damage" p. 16 | happy |
| FR-025 | When the save succeeds and `relocateFeet` is present, `EnterContainer` MUST report the occupant moves to an unoccupied space within that distance (`RelocationDue`) if the command states one is free; if none is free the result MUST be the failure outcome ("the target fails the save instead"). | SRD 5.2.1, "Gelatinous Cube" p. 287 | unhappy (rule interaction) |
| FR-026 | `EntryAutomatic` MUST draw no die, apply no save and give `EnterContainerContained` after the checks of FR-021, with no entry damage. | SRD 5.2.1, "Giant Frog" p. 351; "Giant Toad" p. 354 | happy |
| FR-027 | When the cube's occupant enters the container's space (`EntryCircumstance.EnteredContainerSpace`) the save MUST have Disadvantage in addition to the attempt's own circumstances (combined by `RollMode.Resolve`, so one Advantage cancels it); the circumstance MUST be rejected as `ContainmentProblem.CircumstanceNotApplicable` for any mode other than `engulfed`. | SRD 5.2.1, "Gelatinous Cube" p. 287 ("Ooze Cube"); "Advantage/Disadvantage" p. 8 | unhappy (rule interaction) |
| FR-028 | For mode `swallowed`, a contained result MUST end the grapple first: remove the record with `Grapples.Release(container, occupant)` and pass its `GrappleEnding(released)` to `GrappleRules.EndOnSet` on the occupant's `ConditionSet`, so `ConditionRemoved` then `GrappleEnded` precede the imposed conditions. Interpretation: `released` is the nearest of the four causes (the container takes its hold into itself); no new cause is added. | SRD 5.2.1, "Purple Worm" p. 317 ("the Grappled condition ends"); "Grappling" p. 182 | happy |
| FR-029 | Each condition in `terms.imposed` MUST be applied through `ConditionSet.Apply`; only those that return `ConditionAdded` MUST go into `addedConditions` and the result events. `ConditionImmune` and `ConditionAlreadyHeld` outcomes MUST be reported in the result and never recorded or later removed. | SRD 5.2.1, "Condition" p. 179; "Immunity" pp. 17, 183 | unhappy (rule interaction) |
| FR-030 | A result MUST carry the stable fact that imposed conditions are exactly `terms.imposed` and no others: the Kraken imposes Restrained only. | SRD 5.2.1, "Kraken" p. 303 | unhappy (boundary) |
| FR-031 | The result of a contained outcome MUST state `ContainedPlacement` data for the host: the occupant is in the container's space and moves with it (FR-042); no distance is computed here. | SRD 5.2.1, "Shambling Mound" p. 323 ("pulled into the shambling mound's space") | happy |
| FR-032 | For an `EntryBySave` with `damage`, containment that cannot happen MUST NOT cancel the save: when the occupant is above `maxSize` with `targetAnySize`, or when there is no room (`noRoom`), the save MUST still be rolled; a failure MUST return `EnterContainerDamagedOnly` (entry damage due in full, no containment, no condition, grapples unchanged, reason `tooLarge` or `noRoom`; `grappleImmune` is FR-080's, see `DamagedOnlyReason`) and a success `EnterContainerResisted` (FR-024). The damage is printed apart from the containment in each block: "Failure: 22 (4d8 + 4) Bludgeoning damage. If the target is a Large or smaller creature, it has the Grappled condition" (p. 337); the Kraken's "Failure: 23 (3d8 + 10) Piercing damage. If the target is Large or smaller, it is swallowed" (p. 303); a Huge creature entering the cube's space is "subjected to the cube's Engulf" (p. 287). The Whelm's capacity ("one Large creature or up to two Medium or smaller creatures at a time") and the Kraken's "up to four" limit only how many are held, so a full container still deals the damage (interpretation, Q-15). Without `targetAnySize` an oversize target is a pre-dice `tooLarge` refusal. | SRD 5.2.1, "Kraken" p. 303; "Water Elemental" p. 337; "Gelatinous Cube" p. 287 | unhappy (rule interaction) |
| FR-033 | `ContainmentTerms.ActionKind` (`bonusAction`, `action`) MUST be carried unchanged on every contained, resisted and damaged-only result so the host spends the right action: bonus action for the Behir, Purple Worm, Remorhaz and Tarrasque, action for the Kraken, Giant Frog and Giant Toad. | SRD 5.2.1, "Behir" p. 263; "Kraken" p. 303; "Giant Frog" p. 351 | happy |
| FR-034 | `EnterContainer` MUST return `Invalid` with `ContainmentProblem.SameContainerAndOccupant` when container equals occupant; it MUST NOT return an error. | `combat-foundations.md` FR-031 | unhappy (invalid input) |
| FR-035 | For mode `engulfedGrappled`, a contained result MUST also begin a grapple with `NewGrapple(container, occupant, GraspingBodyPart('<partName> <partOrdinal>'), grapple.escapeDC, 0)` through `Grapples.Begin` and apply `Grapple.Condition` to the `ConditionSet`; the new `Grapples` MUST be in the result. The condition is applied first: if it is not `ConditionAdded` (`ConditionImmune`; a Ghost is immune to Grappled, p. 287), the grapple is not begun and nothing is contained: FR-080. | SRD 5.2.1, "Shambling Mound" p. 323 ("has the Grappled condition (escape DC 14)"); "Grappling" p. 182 | happy |
| FR-036 | `EnterContainer` MUST be deterministic: equal commands and fixed dice give equal results, and it MUST NOT alter its command. | `doc/rules/testing.md` section 4 | unhappy (determinism) |
| FR-037 | A contained result MUST carry `ContainmentEffects(suffocation, noVerbalSpells, noTeleport, totalCover)` read from the terms. When `suffocation` is `always`, the host starts choking with `environment-hazards-and-objects.md` `BeginChoking`; when `unlessBreathesWater`, only if the occupant cannot breathe water. This spec never calls that spec. | SRD 5.2.1, "Gelatinous Cube" p. 287; "Water Elemental" p. 337; "Suffocation [Hazard]" p. 189 | happy |

### 4.4 While contained

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-038 | A contained creature's imposed conditions and restrictions MUST persist as long as the record exists, with no check or re-roll. | SRD 5.2.1, "Behir" p. 263 | happy |
| FR-039 | `restrictionsOn(occupant)` MUST return `ContainedRestrictions(cover, noVerbalSpells, noTeleport, suffocation)` from the terms, and `ContainedRestrictions.None` for a creature not contained. The host passes `noVerbalSpells` to `casting-and-concentration.md` as "cannot speak" and `noTeleport` to any teleport. | SRD 5.2.1, "Gelatinous Cube" p. 287 ("can't cast spells with a Verbal component"); "Tarrasque" p. 332 ("can't teleport") | happy; unhappy (empty and minimal) |
| FR-040 | `coverBetween(source, target)` MUST return `ContainmentCover.Total` when the target is contained with `totalCover` and the source is neither its container nor another occupant of that container, otherwise `ContainmentCover.None`. A host feeds `total` to `attacks.md` (no attack possible, FR-019 there) and treats a spell or effect as unable to target it directly. Interpretation (Q-3): the SRD says "outside"; the container is not outside, and creatures inside are not outside each other. | SRD 5.2.1, "Purple Worm" p. 317; "Cover" p. 179 ("can't be targeted directly") | happy; unhappy (rule interaction) |
| FR-041 | The cube's `totalCover` MUST be true; the Shambling Mound's and the Water Elemental's MUST be false (they print none). | SRD 5.2.1, "Gelatinous Cube" p. 287; "Shambling Mound" p. 323 | unhappy (rule interaction) |
| FR-042 | `movesWith(container)` MUST return the container's occupants in entry order, and `carriedFree(container)` the occupants whose cost to the container's movement is nothing: all of them. The host omits them from `Grapples.ExtraMovementCostPerFoot` (`grappling.md` FR-062). | SRD 5.2.1, "Shambling Mound" p. 323 ("costing it no extra movement"); "Gelatinous Cube" p. 287 ("moves with it") | happy |
| FR-043 | `AdvanceContainerTurn` at the `start` boundary MUST return a `ContainmentDamageDue` per occupant, in entry order, for every record whose `periodic.timing` is `startOfContainerTurn`; at the `end` boundary for `endOfContainerTurn`, except a record with `tickDeferred` (FR-078). No `periodic` gives none. Records in phase `trapped` give none (the container takes no turns). | SRD 5.2.1, "Behir" p. 263; "Shambling Mound" p. 323 and "Kraken" p. 303 (printed "its turns", read as the container's, Q-16); "Giant Toad" p. 354; "Giant Frog" p. 351 | happy |
| FR-044 | Each due increments that record's `damageTicks`; a deferred record at the end boundary only has `tickDeferred` cleared (FR-078). At the end boundary, a record whose terms have `disgorgeAfterFirstDamage` and whose occupant is among the command's `survivors` and has `damageTicks` of 1 or more MUST be released as `disgorged` (FR-056); an occupant not among the survivors MUST be removed with cause `occupantRemoved` (FR-060). | SRD 5.2.1, "Giant Frog" p. 351 ("If that damage doesn't kill it, the frog disgorges it") | unhappy (rule interaction) |
| FR-045 | `isActionLocked(container, actionName)` MUST be true exactly when the container has at least one occupant and its terms' `lockedAction` equals `actionName`. | SRD 5.2.1, "Giant Frog" p. 351 ("can't use Bite"); "Giant Toad" p. 354 | unhappy (boundary) |
| FR-046 | `recordDamageFromInside(occupant, amount)` MUST add `amount` to that occupant's `damageFromInside`; zero is allowed, a negative amount MUST return an error matching `srderr.ErrInvalidInput`, an unknown occupant or one whose record is in phase `trapped` (the container is dead, so there is no container to regurgitate, p. 317, 332) gives `ContainmentRejection.NotContained`. Interpretation (Q-1): the SRD says "from a creature inside it", so the threshold is per occupant, never summed over occupants. | SRD 5.2.1, "Purple Worm" p. 317; "Behir" p. 263 | unhappy (rule interaction) |
| FR-047 | `ResolveRegurgitation` is due when some `swallowed` record in phase `inside` (a `trapped` record never counts) has `damageFromInside` is at least its `thresholdDamage` ("30 damage or more": equal triggers). With no such record the result MUST be `ResolveRegurgitationNotDue` and no die is rolled. | SRD 5.2.1, "Purple Worm" p. 317; "Tarrasque" p. 332 | unhappy (boundary) |
| FR-048 | When due, the container makes a Constitution saving throw against `DifficultyClass(constitutionDC)` with the command's `SaveAttempt` (FR-022). On a failure every occupant of that container MUST be released as `regurgitated` (all of them, not only the damaging one), landing within `landingFeet` and gaining Prone through `ConditionSet.Apply` (`ConditionImmune` reported). On a success `ResolveRegurgitationHeld` MUST leave the occupants. | SRD 5.2.1, "Purple Worm" p. 317; "Tarrasque" p. 332 (10 feet); "Behir" p. 263 | happy |
| FR-049 | `ResolveRegurgitation` MUST reset every record's `damageFromInside` to 0 in every outcome (not due, held, regurgitated): a tally never carries into another turn. The host calls it at the end of each turn in which it called `recordDamageFromInside`. Interpretation (Q-2): "a single turn" is any one turn; "at the end of that turn" is that turn's end. | SRD 5.2.1, "Purple Worm" p. 317 ("at the end of that turn") | unhappy (rule interaction) |
| FR-050 | Terms without `regurgitation` MUST never be due; a swallow of the Giant Frog or Giant Toad ignores damage from inside. | interpretation: no regurgitation clause is printed (p. 351, p. 354) | unhappy (boundary) |

### 4.5 Leaving

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-051 | `ExitContainer` with `ExitBySelf(attempt, actions)` MUST require `exit.selfEscapeDC`, make a Strength (Athletics) check against it (composition as FR-022, kind `abilityCheck`), and give `ExitContainerExited` (cause `escaped`, with the `ExitPlacement` of FR-079) on success or `ExitContainerHeld` on failure, both with `actionSpent` true. | SRD 5.2.1, "Gelatinous Cube" p. 287 ("taking an action to make a DC 12 Strength (Athletics) check") | happy |
| FR-052 | `ExitContainer` with `ExitByPull(puller, pullerWithin5Feet, attempt, actions)` MUST require `exit.pullDC` and the puller to be within 5 feet; on success it releases the occupant (cause `pulledOut`, with the `ExitPlacement` of FR-079) and reports `pullerDamage` due on the puller; on failure no damage is due. | SRD 5.2.1, "Gelatinous Cube" p. 287; "Water Elemental" p. 337 | happy |
| FR-053 | `ExitContainer` MUST refuse, with no die: `notContained`; `noWayOut` when the method's DC is absent (a swallow has none); `pullerIsContained` (a creature in this container cannot pull); `pullerOutOfReach`; `noAction` when the acting creature's `ConditionActionAvailability` is `none`. Order: `notContained`, `noWayOut`, `pullerIsContained`, `pullerOutOfReach`, `noAction`. | SRD 5.2.1, "Gelatinous Cube" p. 287; "Incapacitated [Condition]" p. 184 | unhappy (state error) |
| FR-054 | Releasing a mode `engulfedGrappled` occupant by an exit MUST end the grapple through `Grapples.Escape` and `GrappleRules.EndOnSet(..., GrappleEndCause.Escaped)`. A mound occupant leaves by the grapple's own escape (`EscapeGrappleFromRecord`) and `ReleaseOccupants` (FR-059); `ExitContainer` self refuses it with `noWayOut`. | SRD 5.2.1, "Shambling Mound" p. 323; "Water Elemental" p. 337; "Grappling" p. 182 | unhappy (rule interaction) |
| FR-055 | A release MUST remove from the occupant's `ConditionSet` exactly the kinds in the record's `addedConditions` (not a kind another source holds), emit `OccupantReleased(container, occupant, cause)`, set the record aside, and report `restoreBreathing` when the terms had suffocation. | SRD 5.2.1, "Condition" p. 179; "Suffocation [Hazard]" p. 189 ("can breathe again") | unhappy (rule interaction) |
| FR-056 | A release by `regurgitated`, `disgorged` or `escapedCorpse` MUST apply Prone; by `escaped`, `pulledOut`, `containerDied` or `grappleEnded` it MUST not (Q-13). | SRD 5.2.1, "Purple Worm" p. 317; "Giant Frog" p. 351 | unhappy (rule interaction) |
| FR-057 | `ReleaseOccupants` with `ContainerDied(container)` MUST apply the container's `ContainerDeathRule`: `ReleaseOnDeath` releases every occupant (`containerDied`, nothing special; Q-4), and for an `engulfedGrappled` record also ends the grapple first as in FR-028 (`Grapples.Release(container, occupant)`, then `GrappleRules.EndOnSet(..., GrappleEndCause.Released)` on the occupant's set, the `GrappleEnding` in the result), so no Grappled condition or `Grapples` record outlives the dead container ("Until the grapple ends", p. 323, 337); `EscapeFromCorpse(feet)` MUST move each record to phase `trapped(feet)`, remove Restrained if the record added it, and keep every other imposed condition and Total Cover until escape (Q-11). | SRD 5.2.1, "Behir" p. 263; "Purple Worm" p. 317; "Giant Frog" p. 351 | happy; unhappy (rule interaction) |
| FR-058 | `ReleaseOccupants` with `EscapedCorpse(occupant, movementRemainingFeet)` MUST release a `trapped` occupant (cause `escapedCorpse`, Prone) and report `movementSpentFeet` equal to the record's feet when the remaining movement is at least that; otherwise `insufficientMovement`; a record not trapped gives `notTrapped`. | SRD 5.2.1, "Tarrasque" p. 332 ("escape from the corpse using 20 feet of movement, exiting Prone"); "Giant Toad" p. 354 | unhappy (boundary) |
| FR-059 | `ReleaseOccupants` with `GrappleReview(grapples)` MUST release (cause `grappleEnded`) every `engulfedGrappled` record whose container no longer holds the occupant in `grapples`; the Grappled condition is already gone (`grappling.md` FR-089), so only the imposed conditions are removed. | SRD 5.2.1, "Shambling Mound" p. 323 ("Until the grapple ends"); "Water Elemental" p. 337 | happy |
| FR-060 | `ReleaseOccupants` with `OccupantGone(occupant)` (dead, teleported or removed by the host) MUST drop the record with cause `occupantRemoved` and touch no `ConditionSet`. | `combatant-roster-changes.md` (host removes combatants); package design | unhappy (state error) |
| FR-061 | A release of an occupant that is not contained MUST return the refusal `notContained` and change nothing. | `doc/rules/testing.md` section 4 | unhappy (state error) |
| FR-062 | Every release cause MUST be one `ContainmentLeaveCause` of exactly eight: `escaped`, `pulledOut`, `regurgitated`, `disgorged`, `containerDied`, `escapedCorpse`, `grappleEnded`, `occupantRemoved`. | SRD 5.2.1, pp. 263, 287, 317, 323, 332, 351 | happy |
| FR-063 | The command maps of `occupantConditions` MUST contain exactly the occupants affected; a missing or extra id MUST give `Invalid` with `ContainmentProblem.MissingOccupantConditions` or `unknownOccupant`, never an error or a panic. | `combat-foundations.md` FR-031 | unhappy (invalid input) |

### 4.6 Events, use cases and telemetry

Telemetry follows `combat-foundations.md` FR-027 to FR-037 through the `combat` telemetry helper over `telemetry.Telemetry`. The domain is silent.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-064 | `OccupantContained`, `OccupantReleased` and `ContainerTrapped(container, occupant, feet)` MUST be the only events defined here, one closed family `ContainmentEvent` implementing `CombatEvent`, immutable, equal by value, past tense, `kind` constants `occupant_contained`, `occupant_released`, `container_trapped`, `combatants` the copied slice `[occupant, container]`. No other spec defines them. | `combat-foundations.md` FR-015 to FR-021; `shared-types-and-ownership.md` FR-019 | happy |
| FR-065 | The use cases are synchronous; `EnterContainer`, `ExitContainer` and `ResolveRegurgitation` are constructed with a `dice.Roller`; `AdvanceContainerTurn` and `ReleaseOccupants` take no port; all accept a `telemetry.Telemetry` through `WithTelemetry` (the zero value is the Noop default). Each exposes one entry point taking `ctx context.Context` first (it starts a span, decision 2b). None calls another use case; they share the domain services. | `shared-types-and-ownership.md` FR-036, FR-044 | happy |
| FR-066 | Result variants MUST be operation-prefixed (`EnterContainerContained`, `...Resisted`, `...DamagedOnly`, `...Rejected`, `...Invalid`; `ExitContainerExited`, `...Held`, `...Rejected`; `AdvanceContainerTurnAdvanced`, `...Invalid`; `ResolveRegurgitationRegurgitated`, `...Held`, `...NotDue`, `...Invalid`; `ReleaseOccupantsReleased`, `...Rejected`, `...Invalid`). | `shared-types-and-ownership.md` FR-017 | happy |
| FR-067 | Operations MUST be `enter_container`, `exit_container`, `advance_container_turn`, `resolve_regurgitation` and `release_occupants`, each in the span `srd5e.combat.<operation>` with `srd5e.combat.operation`, `srd5e.combat.result` (the closed set of section 7) and `srd5e.combat.result_class`; `CombatSignals` gains an `<operation>OperationValue`/`<operation>Span` pair each. | `combat-foundations.md` FR-027 to FR-030, FR-037 | happy |
| FR-068 | Classes: `Contained`, `Resisted`, `DamagedOnly`, `Exited`, `Held`, `Advanced`, `Regurgitated`, `NotDue`, `Released` are `applied`; `Rejected` is `refused`; `Invalid` is `invalid`. `srd5e.outcome` is `success` or `failure` only. | `combat-foundations.md` FR-029 to FR-031; `shared-types-and-ownership.md` FR-041 | happy |
| FR-069 | Each release MUST increment `srd5e.combat.containment.ended` once with `srd5e.combat.containment.end_reason` (FR-062 in snake_case); each result carrying a `Containments` MUST record its record count in `srd5e.combat.containment.occupants` with `srd5e.combat.operation`. | `doc/rules/observability.md` section 5 | happy |
| FR-070 | A refusal and a held, resisted or not-due result MUST be normal outcomes: span status not error, no error counter, no log above `FINE`. An `Invalid` sets span error status but not `srd5e.combat.errors`. | `combat-foundations.md` FR-031, FR-032, FR-034 | unhappy (state error) |
| FR-071 | When the `dice.Roller` returns an error, the use case MUST return it wrapped by `srderr.WrapPort` (a `*srderr.PortError` for port `dice`) and the `combat` telemetry helper MUST record it once; no `srd5e.d20_test.*` signal fires; nothing is partially applied. | `combat-foundations.md` FR-032, FR-035; `shared-types-and-ownership.md` FR-044 | unhappy (failure of collaborator) |
| FR-072 | Results MUST be equal with the no-op and with a recording `telemetrytest` double; attributes MUST NOT contain ids, DCs, counts of feet or part names. | `combat-foundations.md` FR-033, FR-036; `observability.md` section 5 | unhappy (determinism) |
| FR-073 | A call MUST NOT log at `telemetry.LevelInfo` or above; at most one `telemetry.LevelDebug` event (message `containment operation`) with operation and result, built lazily by the attrs function. | `combat-foundations.md` FR-034, FR-039 | happy |
| FR-074 | The domain files (`Containments`, `ContainmentRules`, terms) MUST contain no logging, metric or span call and import only `combat-foundations.md`, `conditions.md`, `grappling.md`, `content`, `d20` and kernel published language; no file of `attacks.md`, `initiative-and-actions.md`, `encounter-aggregate.md` or `combat-action-resolution.md`. | `doc/rules/architecture.md` section 9; `shared-types-and-ownership.md` FR-013, FR-015 | unhappy (forbidden edge) |
| FR-075 | The use cases MUST NOT roll damage dice or apply damage. Every damage is a `ContainmentDamageDue(container, occupant, terms, halved, cause)`; the host rolls the terms and applies them through `ApplyDamage` (halving by `HalfDamage`). | SRD 5.2.1, "Half Damage" p. 16; `hit-points-and-damage.md` FR-085, FR-099 | happy |
| FR-076 | Package `combat` MUST export each type of section 5 with godoc, no generic variant name, and not `Grappled`, `ConditionSet`, `CreatureSize`, `DamageType` or `dice.Expression`; every exported name MUST be unique within the package. | `doc/rules/architecture.md` sections 4, 7, 9; `combat-foundations.md` FR-022, FR-024 | happy |
| FR-077 | There are two reason enums: `ContainmentRejection` (FR-019, FR-021, FR-046, FR-053, FR-058, FR-061) held by each `...Rejected`, and `ContainmentProblem` (`sameContainerAndOccupant`, `circumstanceNotApplicable`, `invalidTerms`, `missingOccupantConditions`, `unknownOccupant`) held by each `...Invalid`. | `combat-foundations.md` FR-022, FR-023 | happy |
| FR-078 | `EnterContainerCommand.DuringContainerTurn` states whether the entry happens during the container's own turn. The new record gets `tickDeferred` true exactly when `terms.periodic.firstTickNextTurn` and `duringContainerTurn`; otherwise false. `AdvanceContainerTurn` at the `end` boundary MUST skip the due of a `tickDeferred` record and clear the flag, so the Giant Frog's "At the end of the frog's next turn" (p. 351) is the end of the turn after the swallow, never the end of the turn it swallowed on; a frog that swallows on another creature's turn has no deferral, so its first end boundary is the "next turn". The Giant Toad ("at the end of each of the toad's turns", p. 354) has no flag: the end of the swallowing turn already counts (interpretation: "each" includes the current turn, unlike the frog's explicit "next"). | SRD 5.2.1, "Giant Frog" p. 351; "Giant Toad" p. 354 | unhappy (boundary) |
| FR-079 | A release by self-escape, pull, regurgitation, disgorging, corpse escape, container death or grapple end MUST carry an `ExitPlacement`: `NearestUnoccupiedSpace` (escape, pull, disgorged, corpse, death, grapple end: "enters the nearest unoccupied space", p. 287) or `LandingWithin(feet)` (regurgitated, `landingFeet`). No distance or position is computed here; the host places the occupant (FR-031). | SRD 5.2.1, "Gelatinous Cube" p. 287; "Purple Worm" p. 317 | happy |
| FR-080 | When applying `Grapple.Condition` for an `engulfedGrappled` entry does not give `ConditionAdded` (`ConditionImmune`), `EnterContainer` MUST begin no `Grapples` record and contain nothing: it returns `EnterContainerDamagedOnly` (reason `grappleImmune`; entry damage due when the terms have any, none for the mound), the `ConditionSet`, `Grapples` and `Containments` unchanged and the immunity reported. Interpretation: every effect of the Engulf lasts "until the grapple ends" (p. 323, 337), and an immune creature is never Grappled, so no containment can be bounded by a grapple; this also leaves no record to drop with `DropGrappleRecord` (`grappling.md` FR-098). | SRD 5.2.1, "Shambling Mound" p. 323; "Water Elemental" p. 337; "Immunity" p. 183 | unhappy (rule interaction) |

### 4.7 Case tables

Each row becomes one table-driven test.

**4.7.1 SRD fixtures (FR-013)** (all damage is host-rolled; save ability, DC, imposed conditions I: B Blinded, R Restrained; cover C; slots S; max size M)

| Block (page) | Mode | M | S | Entry | I | C | Periodic | Regurgitate | Death rule |
|---|---|---|---|---|---|---|---|---|---|
| Gelatinous Cube (287) | engulfed | Large | 4 (Large 4) | Dex 12, 3d6 Acid, half, relocate 5, any size (Q-15) | R | yes | 3d6 Acid start | none | release (Q-4) |
| Shambling Mound (323) | engulfedGrappled | Medium | 1 | Str 15, escape 14 | B, R | no | 3d6 Lightning start (Q-16) | none | release |
| Water Elemental (337) | engulfedGrappled | Large | 2 (Large 2) | Str 15, 4d8+4 Bludgeoning, half, any size, escape 14 | R | no | 2d8 Bludgeoning start | none | release |
| Behir (263) | swallowed | Large | 1 | Dex 18 | B, R | yes | 6d6 Acid start | 30, DC 14, 10 ft | corpse 15 |
| Purple Worm (317) | swallowed | Large | 3 | Str 19 | B, R | yes | 5d6 Acid start | 30, DC 21, 5 ft | corpse 20 |
| Remorhaz (319-320) | swallowed | Large | 2 | Str 19 | B, R | yes | 3d6 Acid + 3d6 Fire start | 30, DC 15, 5 ft | corpse 15 |
| Tarrasque (332) | swallowed | Large | 6 | Str 27 | B, R, no teleport | yes | 16d6 Acid start | 60, DC 20, 10 ft | corpse 20 |
| Kraken (303) | swallowed | Large | 4 | Dex 25, 3d8+10 Piercing, any size | R | yes | 7d6 Acid start (Q-16) | 50, DC 25, 10 ft | corpse 15 |
| Giant Frog (351) | swallowed | Small | 1 (Q-14) | automatic | B, R | yes | 2d4 Acid end, first tick next turn (FR-078), disgorge after first; locks Bite | none | corpse 5 |
| Giant Toad (354) | swallowed | Medium | 1 | automatic | B, R | yes | 3d6 Acid end; locks Bite | none | corpse 5 |

Cube also: suffocation `always`, no Verbal spells, exit self DC 12 and pull DC 12 with 3d6 Acid on the puller. Water Elemental: suffocation `unlessBreathesWater`, pull DC 14.

**4.7.2 Capacity (FR-002, FR-003, FR-019, FR-021, FR-032, FR-080)**

| Label | Container | Held | New | Expected | Path |
|---|---|---|---|---|---|
| cube, one Large | cube | none | Large | enters | happy |
| cube, Large then Medium | cube | Large | Medium | save rolled; failure `DamagedOnly` (noRoom), success `Resisted` | unhappy (boundary) |
| cube, four Medium | cube | 3 Medium | Medium | enters | unhappy (boundary) |
| cube, fifth Small | cube | 4 Small | Small | save rolled; `DamagedOnly` (noRoom) on failure | unhappy (boundary) |
| cube, Huge | cube | none | Huge | save rolled; `DamagedOnly` (tooLarge) on failure, no containment | unhappy (boundary) |
| worm, fourth | worm | 3 | Large | `noRoom` | unhappy (boundary) |
| frog, Medium | frog | none | Medium | `tooLarge` | unhappy (boundary) |
| Whelm, Large then Small | Whelm | Large | Small | save rolled; failure 4d8+4 due, `DamagedOnly` (noRoom), no Grappled | unhappy (boundary) |
| Kraken, fifth | Kraken | 4 | Large | save rolled; failure 3d8+10 due, `DamagedOnly` (noRoom), still Grappled | unhappy (boundary) |
| mound, second | mound | 1 Medium | Medium | `noRoom` (no entry damage, no die) | unhappy (boundary) |
| mound, Large | mound | none | Large | `tooLarge` (no die) | unhappy (boundary) |
| Ghost vs mound | mound | none | Medium, immune to Grappled | `DamagedOnly` (grappleImmune), nothing recorded | unhappy (rule interaction) |
| nested | container is an occupant | n/a | any | `containerIsContained` | unhappy (rule interaction) |
| nested, other way | occupant holds occupants | n/a | any | `occupantIsContainer` (no die) | unhappy (rule interaction) |

**4.7.3 Regurgitation threshold and periodic ticks (FR-044, FR-046 to FR-049, FR-078)**

| Label | Tally (per occupant) | Result | Path |
|---|---|---|---|
| below | 29 (threshold 30) | `NotDue`, tally reset | unhappy (boundary) |
| equal | 30 | save rolled | unhappy (boundary) |
| two occupants, 20 and 20 | 20 each | `NotDue` (not summed) | unhappy (rule interaction) |
| save fails | 31, Con DC 21 total 20 | all occupants released, Prone | happy |
| save succeeds | 31, total 21 | `Held`, tally reset | unhappy (boundary) |
| no regurgitation terms | 100 | `NotDue` | unhappy (boundary) |
| trapped record | 40 on a `trapped` record | `recordDamageFromInside` gives `notContained`; `NotDue` | unhappy (rule interaction) |
| frog, own turn | swallowed on the frog's turn | its end boundary: no due, flag cleared; next end: 2d4 due, disgorge if it survives | unhappy (boundary) |
| frog, other turn | swallowed on another creature's turn | first end boundary: 2d4 due | unhappy (boundary) |
| toad, own turn | swallowed on the toad's turn | end boundary of that turn: 3d6 due | unhappy (boundary) |

### 4.8 Withdrawn ids

None.

## 5. Model

Context `combat`: one package, files `containment*.go` for the domain and `enter_container.go`, `exit_container.go` and the like for the use cases. All domain types are immutable, pure, synchronous and silent.

- **Terms** (FR-001 to FR-013): `ContainmentMode`, `ContainmentCapacity`, `DamageTerm`, `EntryTerms` (`EntryAutomatic`, `EntryBySave`), `PeriodicDamage`, `ContainmentTiming`, `Regurgitation`, `ContainerDeathRule` (`ReleaseOnDeath`, `EscapeFromCorpse`), `ExitTerms`, `GrappleHoldTerms`, `SuffocationRule`, `ContainmentTerms`.
- **State**: `Containment` (with phase `inside`/`trapped(feet)`), `Containments` (FR-014 to FR-020), `ContainmentOfOccupant` (`InContainer`, `NotContained`), `ContainmentTransition` (`Applied`, `Rejected`).
- **`ContainmentRules`**: stateless service (a zero-size struct or package functions): `CoverBetween`, `RestrictionsOn`, `MovesWith`, `CarriedFree`, `IsActionLocked`, `ReleaseOnSet(conditions, record, cause)` (FR-055, FR-056). Value types: `ExitPlacement` (`NearestUnoccupiedSpace`, `LandingWithin`), `DamagedOnlyReason` (`tooLarge`, `noRoom`, `grappleImmune`), `ContainmentCover`, `ContainedRestrictions`, `ContainmentEffects`, `ContainmentDamageDue`, `ContainmentLeaveCause`, `ContainedPlacement`, `RelocationDue`, `ContainmentActionKind`.
- **Events** (FR-064), **reasons** (FR-077).
- **Invariants and where enforced:** terms constructors (FR-002, FR-004 to FR-011), `Containment` (FR-014), `Containments` (FR-015). Constructors return `(T, error)` with an error matching `srderr.ErrInvalidInput`; every host-supplied value goes through them (FR-012).
- **Failures:** expected rule outcomes (refused entry or exit, resisted save, held, not due) are closed results; unusable host input is `Invalid` with a `ContainmentProblem`; an error matching `srderr.ErrInvalidInput` only from value-object constructors and for the negative `amount` of FR-046; no `*srderr.PortError` arises in the domain.
- **Application**: the five use cases with their commands and closed results (FR-065, FR-066). `EnterContainerCommand(containments, grapples, container, occupant, occupantSize, terms, occupantConditions, circumstance, attempt, relocationSpaceAvailable, duringContainerTurn)`; `ExitContainerCommand(containments, grapples, occupant, occupantConditions, method, actions)`; `AdvanceContainerTurnCommand(containments, container, boundary, occupantConditions, survivors)`; `ResolveRegurgitationCommand(containments, container, occupantConditions, attempt)`; `ReleaseOccupantsCommand(containments, grapples, occupantConditions, trigger)`. A use case holds no rule: it calls the domain and maps the transition.

## 6. Interface

All use cases are synchronous (no goroutines). Only the three that roll hold the shared `dice.Roller`; no new port, double or contract suite.

- **`NewEnterContainer(r dice.Roller, opts ...Option)`**: the host calls it once per creature when a Swallow, Engulf or Whelm resolves (for an area, once per creature). Results carry the check, the new `Containments`, `Grapples` and `ConditionSet`, damage due, effects and events.
- **`NewExitContainer(r dice.Roller, opts ...Option)`**: `ExitBySelf` or `ExitByPull`; results carry `actionSpent`, the `ExitPlacement` and, for a pull, the puller's damage due.
- **`AdvanceContainerTurn`**: called by the host at the start and end of the container's turn.
- **`NewResolveRegurgitation(r dice.Roller, opts ...Option)`**: called at the end of each turn with recorded damage. The host calls `Containments.RecordDamageFromInside` whenever an occupant damages the container.
- **`ReleaseOccupants`**: triggers `ContainerDied`, `EscapedCorpse`, `GrappleReview`, `OccupantGone`.
- **`ContainmentRules`** for hosts that skip the use cases: queries of section 5.
- **Port failure contract:** a `dice.Roller` error is returned wrapped by `srderr.WrapPort` as a `*srderr.PortError` for port `dice`, recorded once by the `combat` telemetry helper over `telemetry.Telemetry` (FR-071).
- **Exports** (package `combat`; no facade): every type of section 5 and its typed rejection errors, the five use cases, commands and results, and the signal-name constants; imported types come from their owners (FR-076).

## 7. Observability

Shared conventions: `combat-foundations.md` FR-027 to FR-037. This section lists only what this spec adds.

1. **What can go wrong?** Expected refusals (no room, too large, not grappled, nothing to pull), a failed dice roller, a host reporting an inconsistent map of conditions (`Invalid`).
2. **How would an operator know?** `srd5e.combat.result` and `result_class` on the shared operations counter; `Invalid` gives span error status; a roller failure raises `srd5e.combat.errors` once.
3. **What to diagnose with?** Operation, result, class, end reason, record count. Ids, DCs and distances stay in results (FR-072).
4. **Counted or timed here?** Containments ended by reason and records in force, plus one span per operation; the rest is shared.
5. **Existing instrumentation still correct?** Yes: the saves compose the `d20` functions and call no use case, so no `srd5e.d20_test.*` signal fires (FR-071).
6. **What should a consumer monitor?** The `refused` share of `enter_container` by result (a high `not_grappled_by_container` share means the host triggers Swallow without a Grapple); the `ended` mix (many `occupant_removed` suggests the host removes occupants without releasing them); any error increment.
7. **Noisy?** No: at most one `LevelDebug` log per call (FR-073).

Closed sets. `srd5e.combat.operation`: the five of FR-067. `srd5e.combat.result` (one class each): `enter_container`: `contained`, `resisted`, `damaged_only` (applied); `container_is_contained`, `occupant_is_container`, `occupant_already_contained`, `not_grappled_by_container`, `occupant_grappled_elsewhere`, `too_large`, `no_room` (refused); `same_container_and_occupant`, `circumstance_not_applicable` (invalid). `exit_container`: `exited`, `held` (applied); `not_contained`, `no_way_out`, `puller_is_contained`, `puller_out_of_reach`, `no_action` (refused). `advance_container_turn`: `advanced` (applied); `missing_occupant_conditions`, `unknown_occupant` (invalid). `resolve_regurgitation`: `regurgitated`, `held`, `not_due` (applied); `missing_occupant_conditions`, `unknown_occupant` (invalid). `release_occupants`: `released` (applied); `not_contained`, `not_trapped`, `insufficient_movement` (refused); `missing_occupant_conditions`, `unknown_occupant` (invalid). `srd5e.combat.containment.end_reason`: the eight causes of FR-062.

Rows to add to `doc/observability/signals.md`:

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.containment.ended` | counter, `{containment}` | `srd5e.combat.containment.end_reason` | How containments end (one per release) | `occupant_removed` above 50 percent over a day: the host removes occupants without releasing them; `regurgitated` far from what thresholds predict suggests wrong damage reports |
| `srd5e.combat.containment.occupants` | histogram, `{containment}` | `srd5e.combat.operation` | Records in a result carrying a `Containments` | p99 above 50 in one encounter: a host that never releases |
| `srd5e.combat.enter_container`, `...exit_container`, `...advance_container_turn`, `...resolve_regurgitation`, `...release_occupants` | span (one row each, owned here) | `srd5e.combat.operation`, `srd5e.combat.result`, `srd5e.combat.result_class`, `srd5e.outcome` | One call (FR-067) | Shared: `combat-foundations.md` section 5 |

Thresholds are starting values to replace with baselines. The constants added to `CombatSignals` are the five operation pairs, the two metrics, the end-reason attribute and the `'containment operation'` `LogEvent`.

## 8. Test plan

Doubles: the fixed `dice.Roller` and the recording `telemetrytest` double. The tests use real `ConditionSet`, `Grapples` and the `d20` functions. No new port, so no contract suite.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-012 | unit | every constructor boundary (slots 0 and 1, cost above slots, DC 0 and 1, empty damage list, fixed number without dice, `halfOnSuccess` without damage, disallowed imposed kind, mode and field mismatch), equality, `MustX` panics exactly when `NewX` returns an error |
| FR-013 | unit | table 4.7.1: the ten fixtures construct, one test per row |
| FR-014 to FR-020 | unit | table 4.7.2 (including `occupantIsContainer` and the constructor failing on nested records); invariants; copied views; order-insensitive equality; lookups; `enter` rejection order; `partOrdinal` lowest-free |
| FR-021 to FR-037 | application | fixed roller: pre-dice order of refusals with zero draws; save passed through with Disadvantage for the cube; failure and success; half damage; relocation with and without a free space; automatic entry; damage-only for oversize and for no room (cube, Whelm, Kraken) and the refusal when there is no entry damage (mound), Grappled-immune occupant gives `DamagedOnly` with nothing recorded (FR-080); swallow ends the grapple (`ConditionRemoved` then `GrappleEnded(released)`) before conditions; mound begins a numbered grapple; immune and already-held conditions not recorded; effects flags; determinism and command unchanged |
| FR-038 to FR-045 | unit | `restrictionsOn`, `coverBetween` (outsider, container, fellow occupant, cube against mound), `movesWith`, `carriedFree`, periodic due at each boundary (the frog's deferred first tick and disgorge, the toad's every turn, FR-078), trapped gives none, `isActionLocked` |
| FR-046 to FR-050 | application | table 4.7.3: tally rules, equal to threshold, per occupant, reset in every outcome, release of all on failure with Prone, immune to Prone reported, a `trapped` record never counts |
| FR-051 to FR-063 | application | self and pull checks, pull damage, refusal order, grapple-bound release, only added conditions removed (a pre-existing Restrained stays), Prone by cause, `ExitPlacement` per cause (FR-079), container death for both rules (a dead mound's occupant loses Grappled and its `Grapples` record, `GrappleEnded(released)` in the result), corpse escape boundary (movement equal, one short), grapple review, occupant removed, unknown occupant, invalid maps |
| FR-064 to FR-073 | application | recording `telemetrytest` double: span names, closed values, class per result, `ended` once per release, `occupants` only with a set, no error counter on refusals, one error record on a roller failure, no `srd5e.d20_test.*`, identical results with the zero-value Noop telemetry, no ids in attributes, no `LevelInfo` log; events closed family, `kind` unique, `combatants` immutable |
| FR-074 | architecture | import scan (`internal/archtest`); no logging in the domain files |
| FR-075 | application | no use case imports a roller for damage; every damage is a `ContainmentDamageDue` |
| FR-076, FR-077 | public API | an external test package (`combat_test`) importing only public packages builds each command and type-switches exhaustively over every result and event; exported-name scan for unique names; exactly two reason enums |
| scenario | scenario | public API only: a Purple Worm Grapples a fighter, `EnterContainer` fails the save, the host applies damage each start of turn, the fighter deals 31 damage in one turn, `ResolveRegurgitation` fails and releases it Prone; a second scenario kills a Behir and the occupant uses `EscapedCorpse`; a third runs a cube Engulf, a failed pull and a successful pull with puller damage |
| errors-and-results.md FR-031, FR-034, FR-051 | application | The roller failure of FR-071 is returned wrapped as a `*srderr.PortError` and recorded once; a recorded failure sets `error.type` from `srderr.ErrorType(err)`; a panicking roller is recovered by `srderr.Guard`. |

- **Mutation testing:** the `containment*.go` files of package `combat`, with gomu run only inside Docker with `--cpus` and `--workers` capped at 1/4 of `nproc` (`doc/rules/testing.md` section 8; never run locally). The boundary rows (threshold 29/30, capacity edges, movement equal or one short) kill comparison mutants; the rejection-order tests kill order mutants; the two-occupant 20+20 row kills a sum-versus-per-creature mutant.
- **Not covered:** none. Every FR id appears above.

## 9. Open questions and decisions

Open: none. Nothing is escalated. Every interpretation is justified from the stat-block wording and package design, as below.

Settled, one line each:

- **Q-1 (damage threshold):** per occupant, not summed: the SRD says "from a creature inside it" (singular, p. 317); the Behir says "from the swallowed creature" (p. 263). Interpretation (FR-046).
- **Q-2 ("a single turn"):** damage dealt by one occupant within one turn, in any creature's turn; the check is "at the end of that turn" (p. 317). The host calls `ResolveRegurgitation` per turn and the tally always resets (FR-049).
- **Q-3 (Total Cover):** "against attacks and other effects outside the" container (p. 317). The container is not outside, and neither are fellow occupants; interpretation (FR-040).
- **Q-4 (container dies, cube, mound, Whelm):** the SRD is silent. Reasonable reading: the occupants are released where the container was, with no Prone and no movement cost (FR-057).
- **Q-5 (grapple end cause on Swallow):** `GrappleEndCause.Released` (FR-028); no amendment to `grappling.md` FR-085.
- **Q-6 ("is suffocating"):** the Suffocation hazard (p. 189) with the cause choking; the host uses `BeginChoking` and `RestoreBreathing` (FR-037, FR-055).
- **Q-7 (Tiny occupant of the cube):** cost 1, like Small (FR-003); the SRD names only Large, Medium and Small (p. 287).
- **Q-8 (monster terms):** the block prose stays verbatim in `content-monsters.md` (its FR-034); the host transcribes `ContainmentTerms` using table 4.7.1. No content change.
- **Q-9 (page):** the Giant Toad's Swallow is printed on p. 354, not p. 353 (checked in the PDF text).
- **Q-10 (Rug, Darkmantle):** the Rug of Smothering halves its own damage and shares it with the target (p. 259) and the Darkmantle attaches (p. 279); neither fits this model without extra rules, so they are out of scope.
- **Q-11 (state after the container dies):** only Restrained ends (p. 317, 332, 303); Blinded and Total Cover stay until escape (FR-057). Interpretation: the stat block removes nothing else.
- **Q-12 (nested containment):** not modelled in either direction; `containerIsContained` and `occupantIsContainer` refuse it and the `Containments` constructor returns an error (FR-015, FR-019).
- **Q-13 ("exiting Prone"):** read as leaving with the Prone condition (the Giant Frog says "exiting with the Prone condition", p. 351; Giant Toad p. 354). Prone is applied on regurgitation, disgorging and corpse escape (FR-056).
- **Q-14 (Giant Frog slots):** the frog prints no count (p. 351); one slot is an interpretation from "the target" and the Bite lock. The toad prints one (p. 354).
- **Q-15 (damage without room, cube size):** entry damage does not depend on capacity (p. 287, 303, 337), so a full container or oversize target still rolls the save and takes the failure damage without being held (FR-032). Interpretation: the cube's `targetAnySize` is true because a Huge creature entering its space is "subjected to the cube's Engulf" (p. 287) yet cannot fit; whether the cube may move through a space is the host's movement decision, not an entry. Grappled-immune occupants: FR-080.
- **Q-16 (whose turn for the Mound and Kraken):** both blocks print "at the start of each of its turns" with the held creature as the grammatical subject (Shambling Mound p. 323, Kraken p. 303), while the Behir, Purple Worm, Remorhaz, Tarrasque, Water Elemental and Gelatinous Cube name the container's turn. Interpretation: modelled as `startOfContainerTurn` anyway, because the held creature takes the damage once per round either way and the engine has a single periodic trigger, `AdvanceContainerTurn`. No `startOfOccupantTurn` member is added (new API for a one-block ambiguity). Not escalated: the difference is one tick per round at a different initiative count.

## 10. Changelog impact

Under `Unreleased`, "Added":

- Monster Swallow and Engulf in the `combat` context: `Containments`, `Containment`, `ContainmentTerms` with its term types, `ContainmentRules`, the closed event family `ContainmentEvent`, and the use cases `EnterContainer`, `ExitContainer`, `AdvanceContainerTurn`, `ResolveRegurgitation` and `ReleaseOccupants`.
- Telemetry signals `srd5e.combat.containment.ended` and `srd5e.combat.containment.occupants`, plus the five operations on the shared combat signals.

Not breaking: all new API in an unreleased package. Closed results and events make a later added variant breaking for consumers that switch over them (architecture section 7).
