# Returning to life

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `conditions.md`, `hit-points-and-damage.md`, `magical-effects.md` |
| Consumed by | `encounter-aggregate.md` (revival transition), `combat-action-resolution.md` (amendment: the `ReviveCombatant` use case) |

Split from `magical-effects.md` (its FR-097 to FR-108, FR-110, FR-111 and FR-146,
withdrawn there in favour of FR-001 to FR-015 here). Rank 27
(`shared-types-and-ownership.md` 4.11): right of `magical-effects` (26), whose
`OngoingEffects` and `OngoingEffectID` it imports, with `HitPointStatus` and its
companions (`hit-points-and-damage.md`, 21), `ConditionSet` and its operations
(`conditions.md`, 22) and `CombatantID` / `combat.Event` (`combat-foundations.md`).
It has no port and returns no `srderr.PortError`, so `errors-and-results.md` and
`observability-ports.md` are not dependencies. Supersedes no numbered requirement
of another spec (decisions FR-048).

## 1. Purpose and scope

The Rules Glossary "Dead" entry (SRD 5.2.1, p. 180), the part about returning to
life, deferred here by `hit-points-and-damage.md` (Q-7) and `encounter-aggregate.md`.
A Dead creature has no Hit Points and cannot regain them until revived by magic.
Revival is a value-level transformation: it never mutates the Dead `HitPointStatus`;
it builds a new one through the restore factory (`hit-points-and-damage.md` FR-087),
so Dead stays terminal there. The host holds `ConditionSet` and `OngoingEffects`
(`magical-effects.md` FR-006 to FR-010) and reports what elapsed.

**Out of scope:** the spells that revive (Raise Dead and the like): their Hit Point
amounts, eligibility, ordeal and strain are owned by `revival-and-restoration-spells.md`,
which composes `combat.Revive`; Attunement after revival (`magic-item-rules.md` FR-054, the
host calls `EndMagicItemAttunement`); any clock or duration tracking (Q-3).

## 2. Source

- SRD 5.2.1, "Dead", p. 180; "Exhaustion [Condition]", p. 181; "Hit Points", p. 183
  ("You can't have more Hit Points than your Hit Point maximum").
- Rule files: `doc/rules/architecture.md` §3, §6; `doc/rules/testing.md` §4, §5;
  `doc/rules/observability.md` §2, §3.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Dead, returning to life | `ReviveCreatureCommand`, `ReviveCreatureResult`, `RevivalConsent` (`SpiritAccepts`, `SpiritRefuses`) | Operation prefix `ReviveCreature`. |
| Hit Points, Dead | `HitPointStatus`, `HitPoints`, `LifeState`, `DeathCause`, `ZeroHitPointsRule`, `DamageAffinities` | `hit-points-and-damage.md`. |
| conditions, curses, contagions | `ConditionSet`, `OngoingEffects`, `OngoingEffectID` | `conditions.md`; `magical-effects.md`. |

## 4. Requirements

Ids are stable. Case-table rows are one test each, citing the FR in the row.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `ReviveCreature` on a creature whose `LifeState` is `Alive`, `Dying` or `Stable` MUST return `ReviveCreatureNotDead` with the status unchanged and no events. | SRD 5.2.1, "Dead", p. 180 | unhappy (state error) |
| FR-002 | `ReviveCreature` with `RevivalConsent` value `SpiritRefuses` MUST return `ReviveCreatureRefused` with the status unchanged and no events ("the spirit ... can refuse"). | SRD 5.2.1, "Dead", p. 180 | unhappy (rule interaction) |
| FR-003 | On success `ReviveCreature` MUST return `ReviveCreatureApplied`, the creature's current Hit Points MUST be the number the revival effect determines, and its `LifeState` MUST be `Alive` (case table 4.1). | SRD 5.2.1, "Dead" ("the revival effect determines the creature's current Hit Points"), p. 180 | happy |
| FR-004 | A revival Hit Point value below 1 MUST return `ReviveCreatureHitPointsBelowOne`: a living creature at 0 Hit Points would be Dying, which the SRD does not describe for a revival. | Package design (Q-1 settled); `hit-points-and-damage.md` FR-090 (Alive at 0 is rejected); SRD 5.2.1, "Dead", p. 180 | unhappy (boundary) |
| FR-005 | A revival Hit Point value above the Hit Point maximum MUST be capped at the maximum. | SRD 5.2.1, "Hit Points" ("You can't have more Hit Points than your Hit Point maximum"), p. 183 | unhappy (boundary) |
| FR-006 | A Dead creature whose `HitPoints.Maximum` is below 1 MUST return `ReviveCreatureMaximumBelowOne`: nothing in the revival text restores a Hit Point maximum. | SRD 5.2.1, "Hit Points", p. 183; "Dead", p. 180 | unhappy (boundary) |
| FR-007 | Revival MUST be possible after every `DeathCause` (`MonsterDeath`, `HitPointMaximumZero`, `MassiveDamage`, `DeathSavingThrowFailures`, `Exhaustion`), except that `HitPointMaximumZero` is rejected by FR-006 when the maximum is still below 1 (one test per cause; a level 6 Exhaustion death returns at level 5, FR-011). | SRD 5.2.1, "Dead", p. 180 | unhappy (rule interaction) |
| FR-008 | The returned `HitPointStatus` MUST keep the creature's `ZeroHitPointsRule` and `DamageAffinities` (the restore factory, `hit-points-and-damage.md` FR-087). It MUST have 0 Temporary Hit Points: package design, since "Dead" says nothing about them. | `hit-points-and-damage.md` FR-087; package design | happy |
| FR-009 | An explicit condition (other than Exhaustion, which FR-011 and FR-015 decide) the creature held at death MUST be kept when its duration is still ongoing and dropped when the host reports it elapsed (`ElapsedConditions`); a kind not reported elapsed counts as ongoing (case table 4.2; Q-2). Each dropped condition emits `conditions.md`'s `ConditionRemoved` (FR-031), through `ConditionSet.Remove`. | SRD 5.2.1, "Dead" ("with any conditions ... that were affecting it at death if the durations of those effects are still ongoing"), p. 180 | happy |
| FR-010 | An `OngoingEffect` of category `MagicalContagion` or `Curse` MUST be kept when its id is not in `ElapsedEffects` and dropped when it is; effects of the other categories follow the same rule. | SRD 5.2.1, "Dead" ("magical contagions, or curses"), p. 180 | happy |
| FR-011 | A creature that died with Exhaustion level `n` MUST return with level `n - 1`, and with no Exhaustion when `n` is 1 (case table 4.3); this is `conditions.md` `ReduceExhaustion(1)` and its events. | SRD 5.2.1, "Dead" ("it returns with 1 fewer level"), p. 180; "Exhaustion", p. 181 | happy |
| FR-012 | A creature that died with no Exhaustion MUST return with none, and `ReviveCreature` MUST NOT treat the missing level as an error (`ExhaustionReductionNotPresent` is a normal outcome). | SRD 5.2.1, "Dead" ("If the creature died with any Exhaustion levels"), p. 180; `conditions.md` FR-046 | unhappy (empty and minimal) |
| FR-013 | A revival MUST emit one `CreatureReturnedToLife` event carrying the creature and its new current Hit Points, followed by `conditions.md`'s own events for the conditions dropped under FR-009 (`ConditionRemoved`) and for the Exhaustion reduction under FR-011 (`ConditionLevelReduced` or `ConditionRemoved`), in that order. | `doc/rules/architecture.md` §3 (domain event); `conditions.md` FR-031, FR-036 | happy |
| FR-014 | `ReviveCreature` MUST NOT alter its inputs, MUST return copied slices (mutating a returned slice changes nothing in the result) and MUST give equal results for equal inputs. | `doc/rules/testing.md` §4 (immutability, determinism) | unhappy (immutability) |
| FR-015 | An id or kind named in `ElapsedConditions` or `ElapsedEffects` that the creature does not hold as an explicit condition or listed effect MUST be ignored, with no event and no rejection: only what the host reports elapsed is dropped (Q-2), and a report about something absent changes nothing (case table 4.2, last rows). This includes a kind effective only by implication (`conditions.md` FR-033: the implying explicit condition is what lapses) and Exhaustion, which `ElapsedConditions` cannot drop: FR-011 alone decides it. | SRD 5.2.1, "Dead", p. 180; Q-2 | unhappy (invalid input) |
| FR-016 | Layering, events and telemetry follow `combat-foundations.md` FR-015 to FR-019, FR-026 and FR-029 to FR-035: types live in package `combat`, are pure and import only kernel packages and the published language of `conditions.md`, `hit-points-and-damage.md` and `magical-effects.md`; `CreatureReturnedToLife` (kind `creature_returned_to_life`, `combatants` the creature) is the one event of the closed family `RevivalEvent` (an interface embedding `combat.Event` with an unexported marker method; its doc comment lists the event); the use case is `ReviveCombatant(ctx, ...)` (span `revive_combatant`). | `doc/rules/architecture.md` §1, §2, §9; `shared-types-and-ownership.md` FR-019, FR-044 | unhappy (forbidden edge) |

#### Case table 4.1: revival Hit Points (FR-003 to FR-006)

Baseline: Dead player character, `HitPoints.Maximum` 20, spirit accepts.

| Case | Revival value | Expected | FR | Source | Path |
|---|---|---|---|---|---|
| one Hit Point | 1 | Alive at 1 | FR-003 | p. 180 | happy |
| a typical value | 7 | Alive at 7 | FR-003 | p. 180 | happy |
| equal to the maximum | 20 | Alive at 20 | FR-003 | p. 180 | unhappy (boundary) |
| one above the maximum | 21 | Alive at 20 | FR-005 | p. 183 | unhappy (boundary) |
| zero | 0 | `ReviveCreatureHitPointsBelowOne` | FR-004 | package | unhappy (boundary) |
| negative | -3 | `ReviveCreatureHitPointsBelowOne` | FR-004 | package | unhappy (invalid input) |
| maximum below 1 | status maximum 0, value 5 | `ReviveCreatureMaximumBelowOne` | FR-006 | p. 183 | unhappy (boundary) |

#### Case table 4.2: what returns with the creature (FR-009, FR-010, FR-015)

Source for every row: SRD 5.2.1, "Dead", p. 180.
Baseline: `ConditionSet` {Poisoned, Blinded}, `OngoingEffects` [contagion `c1`, curse `c2`, spell `s1`].

| Case | `ElapsedConditions` | `ElapsedEffects` | Expected conditions | Expected effects | Path |
|---|---|---|---|---|---|
| all ongoing | none | none | Poisoned, Blinded | `c1`, `c2`, `s1` | happy |
| a condition lapsed | {Poisoned} | none | Blinded | `c1`, `c2`, `s1` | unhappy (rule interaction) |
| a curse lapsed | none | {`c2`} | Poisoned, Blinded | `c1`, `s1` | unhappy (rule interaction) |
| a contagion lapsed | none | {`c1`} | Poisoned, Blinded | `c2`, `s1` | unhappy (rule interaction) |
| everything lapsed | {Poisoned, Blinded} | {`c1`, `c2`, `s1`} | empty | empty | unhappy (boundary) |
| nothing held | none | none, empty inputs | empty | empty | unhappy (empty and minimal) |
| elapsed names an absent item (FR-015) | {Charmed} | none | Poisoned, Blinded | `c1`, `c2`, `s1` | unhappy (invalid input) |
| elapsed names an implied-only kind (FR-015) | set {Paralyzed}, {Incapacitated} | none | Paralyzed | `c1`, `c2`, `s1` | unhappy (rule interaction) |
| elapsed names Exhaustion (FR-015, FR-011) | set {Exhaustion 3}, {Exhaustion} | none | Exhaustion 2 | `c1`, `c2`, `s1` | unhappy (rule interaction) |

#### Case table 4.3: Exhaustion on return (FR-011, FR-012)

| Case | Level at death | Level after | Source | Path |
|---|---|---|---|---|
| none | none | none | p. 180 | unhappy (empty and minimal) |
| level 1 | 1 | none (the condition ends) | p. 180, p. 181 | unhappy (boundary) |
| level 2 | 2 | 1 | p. 180 | happy |
| level 5 | 5 | 4 | p. 180 | happy |
| level 6 (died of Exhaustion) | 6 | 5 | p. 180, p. 181 | unhappy (boundary) |

## 5. Model

- **Referenced, not defined here:** `CombatantID`, `combat.Event`; `ConditionSet`,
  `ReduceExhaustion`, `ConditionSet.Remove`, `ConditionRemoved`, `ConditionLevelReduced`;
  `HitPointStatus`, `HitPoints`, `LifeState`, `DeathCause`, `ZeroHitPointsRule`,
  `DamageAffinities`; `OngoingEffects`, `OngoingEffectID` (`magical-effects.md`).
- **Value objects:** the types of section 3. **Event:** `CreatureReturnedToLife`
  in the closed family `RevivalEvent`. **Domain service** (stateless):
  `combat.Revive`.
- **Failures:** `ReviveCreatureResult`, an interface with an unexported marker
  method and one struct type per variant (the five named in section 7). Every
  refusal is a returned variant, never an `error`; there is no `NewX` error of its
  own and no `srderr.PortError` (no port). The interface doc comment lists all
  five variants; switches over it are exhaustive.

## 6. Interface

- No port. `combat.Revive(cmd ReviveCreatureCommand) ReviveCreatureResult` is a
  synchronous pure function over host-held values: no `ctx`, no I/O, no
  goroutines. Safe for concurrent use (it holds no state).
- Package: exported from package `combat` (`github.com/Whealinghold/srd5e-go/combat`);
  the handoff type is `ReviveCreatureResult`. No facade, no test doubles.

## 7. Observability

Silent by design (`doc/rules/observability.md` §3); no signal of its own. The
use case `ReviveCombatant` (`revive_combatant`) of the `combat-action-resolution.md` amendment records through `CombatTelemetry`
(`combat-foundations.md` FR-029 to FR-035).

| Variant | `srd5e.combat.result` | Class |
|---|---|---|
| `ReviveCreatureApplied` | `completed` | applied |
| `ReviveCreatureNotDead` | `revive_not_dead` | refused |
| `ReviveCreatureRefused` | `revive_refused` | refused |
| `ReviveCreatureHitPointsBelowOne` | `revive_hit_points_below_one` | refused |
| `ReviveCreatureMaximumBelowOne` | `revive_maximum_below_one` | refused |

Seven questions: (1) the host calls revive on a living creature, or passes a bad
Hit Point value or maximum (`refused`). (2) By `srd5e.combat.operations` once that use case
lands. (3) The result carries the Hit Points after capping and, in events, what
was dropped. (4) Each call by result. (5) Existing instrumentation is unchanged.
(6) Sustained `revive_not_dead` is a host state bug; a rise in
`revive_maximum_below_one` is an upstream Hit Point maximum bug. (7) Nothing is
noisy: no log.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-008 | unit | Table 4.1, one test per `DeathCause` and the not-dead and refused rows; the returned status keeps rule and affinities, 0 Temporary Hit Points |
| FR-009, FR-010, FR-015 | unit | Table 4.2, one test per row, including the absent-item row |
| FR-011, FR-012 | unit | Table 4.3; level 6 gives 5; none gives none |
| FR-013, FR-014 | unit | Event order including the wrapped condition events; inputs unchanged |
| FR-016 | architecture | Import direction; `kind` unique across the package (scan) |

No doubles or contract suites. Mutation testing (`doc/rules/testing.md` §8): all
changed lines in `combat`; boundary rows exist for FR-004 and FR-005.

## 9. Open questions and decisions

| Id | Question | Recommendation | Status |
|---|---|---|---|
| Q-1 | Revival effect says 0 Hit Points? | Settled, see below. | settled |
| Q-4 | "Unless otherwise stated" overrides? | Settled, see below. | settled |

Settled, one line each: Q-1 a revival value below 1 is rejected (FR-004): a living creature at 0 is not constructible (`hit-points-and-damage.md` FR-090) and p. 180 says only that "the revival effect determines" the Hit Points, and the SRD revival spells give 1 Hit Point or all of them (Revivify, Resurrection-type effects; interpretation: a positive value is the reasonable reading). Q-2 only what the host reports elapsed is dropped; the package keeps no clock or durations (FR-009, FR-015). Q-3 same. Q-4 no extra input: "unless otherwise stated" (p. 180) is met by the host listing a condition or effect it wants cleared in `ElapsedConditions` / `ElapsedEffects`, by calling `Remove` or `ReduceExhaustion` afterwards for further changes, and by adding anything the revival imposes with the ordinary `conditions.md` operations after `ReviveCreatureApplied` (design: no per-spell override type).

## 10. Changelog impact

- **Added:** returning to life (the "Dead" entry) in `combat`
  (`doc/specs/returning-to-life.md`).
- **BREAKING:** no. Everything is new.
