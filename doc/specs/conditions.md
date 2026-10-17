# Conditions

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `errors-and-results.md`, `observability-ports.md`, `content-schema-and-decoder.md` |

Dependency notes (prose, not dependencies): `combat-foundations.md` supplies `CombatantID`, `Event`, `Operation`, `ResultClass`, `Telemetry` and `Signals` (shared-types-and-ownership.md FR-012, FR-015). `modifiers-and-advantage.md` supplies `RollCircumstance` and `RollMode`; `Modifier` is not used, because `D20TestReduction` returns an `int`. `content-schema-and-decoder.md` FR-110 owns `ConditionKind` (shared-types-and-ownership.md FR-005, FR-046). `grappling.md` depends on this spec and is referenced, not depended on (shared-types-and-ownership.md FR-029, FR-048). Nothing here depends on a later `combat` spec or a downstream context (shared-types-and-ownership.md FR-013, FR-015).

## 1. Purpose and scope

The fifteen conditions of the SRD 5.2.1 Rules Glossary as one table-driven spec: a closed `Condition` type (one variant per condition, carrying its source where the SRD names one), each condition's effects as data, how conditions combine (no stacking, except cumulative Exhaustion), which conditions imply others (Incapacitated, Prone), how a creature's `ConditionSet` changes on apply and remove, and the events those changes produce (`ConditionApplied` and siblings, the closed `ConditionEvent` family implementing `Event`, shared-types-and-ownership.md FR-019).

Effects are pure queries over a `ConditionSet`. `ConditionEffects` is the published function of shared-types-and-ownership.md FR-026: it returns `RollContributions`, automatic-failure effects and an `AutomaticCritical` value. `attacks.md` takes those as plain inputs and `combat-action-resolution.md` is the only caller of both (FR-131). No dice are rolled and no D20 Test is resolved here. This spec owns only the Grappled condition **entry** (FR-060 to FR-062); grapple mechanics belong to `grappling.md`.

**Out of scope:**

- **Ending conditions by duration or trigger** ("Various rules define how to end a condition", p. 179). Explicit removal is provided (`RemoveConditionFromSet`, `ReduceExhaustionOnSet`); when to call it belongs to the rule that ends the condition. Spell durations belong to `character` and later specs.
- **Which effects apply a condition.** Spells, monster traits and features are content; they arrive as a request to apply a `Condition`.
- **Grappling rules** ("Grappling", p. 182): the grapple option, escape, movement, ending the grapple and `GrappleEnded` belong to `grappling.md`; the Unarmed Strike belongs to `attacks.md` (shared-types-and-ownership.md FR-029, FR-048).
- **Hit points and death.** Falling Unconscious, Death Saving Throws, Stable and Knocking Out a Creature (pp. 17-18, 184) belong to hit points and damage. "You die if your Exhaustion level is 6" is reported as the query `IsDeadFromExhaustion`; `encounter-aggregate.md` FR-037 reads it and applies `hit-points-and-damage.md` FR-095 (D-3).
- **Hazards causing Exhaustion** (Dehydration p. 181, Malnutrition p. 185, Suffocation p. 189) and the Long Rest that removes a level (p. 185): they change a `ConditionSet` through `Apply`, `Remove` and `ReduceExhaustion`, not through this spec's use cases (shared-types-and-ownership.md FR-044; `environment-hazards-and-objects.md` FR-098, FR-118).
- **Heavily Obscured** (p. 182) and Blindsight (p. 177): lighting and senses.
- **Rules that read a condition**: Concentration (p. 179), Dodge (p. 181), Flying (p. 182), Initiative and Surprise (pp. 184, 189), Influence, Short Rest. This spec exposes the facts (`IsIncapacitated`, Speed 0); the rules belong to their own specs.
- **Narrative effects with no rules consequence:** Petrified "transformed ... into a solid inanimate substance ... weight increases by a factor of ten, and you cease aging", its worn and carried objects (p. 186), and "You're unaware of your surroundings" beyond a flag (p. 191).
- **Innate immunities** (monster immunities): they come from content and enter as an input set.
- **Conditions not in the Glossary list** (Burning, Bloodied, Dead, Stable, Surprised, Concentrating, Cursed, Possessed, Hiding): hazards, states or other rules (p. 179).
- **Anything outside SRD 5.2.1.**

## 2. Source

- SRD 5.2.1, "Rules Glossary", "Condition", p. 179 (the fifteen conditions; "A condition doesn't stack with itself ... The Exhaustion condition is an exception").
- "Rules Glossary" condition entries: Blinded p. 177; Charmed p. 178; Deafened p. 181; Exhaustion p. 181; Frightened p. 182; Grappled p. 182; Incapacitated p. 184; Invisible p. 184; Paralyzed p. 186; Petrified p. 186; Poisoned p. 186; Prone p. 186; Restrained p. 187; Stunned p. 189; Unconscious p. 191.
- "Advantage", p. 176, and "Disadvantage", p. 181 (they cancel); "Round Down", p. 187 (half Speed to stand from Prone).
- "Damage and Healing", "Immunity", p. 17 ("Immunity to a condition means you aren't affected by it"); "Rules Glossary", "Immunity", p. 183 ("it doesn't affect you in any way").
- "Rules Glossary", "Long Rest", p. 185 ("Exhaustion Reduced"); "Damage and Healing", "Falling Unconscious", p. 17 (out-of-scope note only); "Concentration", p. 179 and "Initiative", p. 184 (named by Incapacitated).

PDF page index equals printed page number (PDF page 5 shows printed page 5), so there is no offset.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| condition | `Condition` (closed interface) | "A temporary game state" (p. 179). One variant per condition. |
| name of a condition | `ConditionKind` (enum) | Owned by `content-schema-and-decoder.md` FR-110; not defined here. Its values must be the fifteen Glossary names in Glossary order, because this spec orders by declaration order (FR-130). |
| Blinded, Deafened, Incapacitated, Invisible, Paralyzed, Petrified, Poisoned, Prone, Restrained, Stunned, Unconscious | struct types, no data | Variants of `Condition`. |
| Charmed, Frightened, Grappled | `Charmed{charmer}` with a closed `Charmer` interface, `Frightened(source)`, `Grappled(grappler)` | Frightened and Grappled hold a `CombatantID`; the charmer is a `CombatantID` or an opaque item id (FR-141; pp. 178, 182, 208). The source of fear is a creature (D-2). |
| Exhaustion level | `Exhaustion(level)`, `ExhaustionLevel` | Level 1 to 6 (p. 181); level 0 means no condition. |
| a creature's conditions | `ConditionSet` (value object) | The explicit conditions on one creature; effective conditions are derived. |
| implied condition | `ConditionDefinition.implies` | Paralyzed, Petrified, Stunned, Unconscious imply Incapacitated; Unconscious also Prone. |
| Immunity | a set of `ConditionKind` (unexported; read through `Immunities()`) | Stored immunities plus effect-derived `ImmuneTo` (FR-124). |
| Advantage, Disadvantage | `RollCircumstance` | From `modifiers-and-advantage.md`. |
| Speed | `int` feet | One query per Speed a creature has (FR-107, D-14). |
| Critical Hit | `AutomaticCritical` (`automatic`, `notAutomatic`) | pp. 186, 191. |
| Resistance to all damage | `AllDamageResistance` (flag, FR-082); the data variant is `DamageResistanceAll` | Reported as a flag only (FR-082, shared-types-and-ownership.md FR-006). |
| condition event | `ConditionEvent` (closed interface, extends `Event`) | Members `ConditionApplied`, `ConditionRemoved`, `ConditionLevelReduced` (FR-129, FR-139). |
| Concentration fact | `ConditionConcentration` (`intact`, `broken`) | Reported fact only (FR-065); avoids a clash with `Concentration` of `spellcasting.md` (shared-types-and-ownership.md FR-011). |

## 4. Requirements

Distances are whole feet; "within 5 feet" is 0 to 5 inclusive. A "contribution" is one `RollCircumstance` tagged with the `ConditionKind` that produced it. Withdrawn ids are in the "Withdrawn ids" table at the end of this section.

### 4.1 Catalogue and representation

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-002 | `Condition` MUST be a closed interface (unexported marker method; the doc comment names all fifteen variant types) with exactly one variant type per `ConditionKind`, so a type `switch` is exhaustive (`gochecksumtype`) and each variant reports its `Kind()`. | SRD 5.2.1, "Condition", p. 179 | happy |
| FR-003 | `Charmed` MUST carry a `Charmer` (FR-141), `Frightened` the source of fear's, `Grappled` the grappler's; every other variant, including `Exhaustion` (which carries a level), MUST carry no creature. | SRD 5.2.1, "Charmed", p. 178; "Frightened", p. 182; "Grappled", p. 182 | happy |
| FR-004 | Two `Condition`s MUST be equal under `==` when same kind and same source (or Exhaustion level); a different kind, source or level MUST NOT be equal. | `doc/rules/architecture.md` §3 (value object) | happy |
| FR-007 | A `ConditionDefinition` MUST exist for every `ConditionKind` and list its effects and implied kinds as data (closed `ConditionEffect` interface), so the section 4.7 table is the single source of the rules. | SRD 5.2.1, the condition entries, pp. 177-191 | happy |
| FR-008 | `ExhaustionLevel` MUST accept every integer 1 to 6 inclusive and expose it as `Value()`; `NewExhaustionLevel(n int) (ExhaustionLevel, error)` is the constructor and `MustExhaustionLevel` panics with its error. | SRD 5.2.1, "Exhaustion", p. 181 | happy |
| FR-009 | Constructing an `ExhaustionLevel` from 0 or a negative integer MUST return an error wrapping `srderr.ErrInvalidInput`; level 0 is "does not have the condition". | SRD 5.2.1, "Exhaustion", "Removing Exhaustion Levels", p. 181 | unhappy (boundary) |
| FR-010 | Constructing an `ExhaustionLevel` from 7 or more MUST return an error wrapping `srderr.ErrInvalidInput`; 6 is the highest level. | design decision D-3 (SRD silent on a maximum); nearest text: SRD 5.2.1, "Exhaustion", "Exhaustion Levels", p. 181 ("You die if your Exhaustion level is 6") | unhappy (boundary) |

### 4.2 The condition set

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-011 | An empty `ConditionSet` MUST have no explicit and no effective conditions. | SRD 5.2.1, "Condition", p. 179 | unhappy (empty and minimal) |
| FR-012 | A `ConditionSet` MUST hold at most one explicit `Condition` per kind; `NewConditionSet` given two of the same kind (two Exhaustion, or two `Charmed` with different charmers) MUST return an error wrapping `srderr.ErrInvalidInput`. | SRD 5.2.1, "Condition", p. 179 ("A condition doesn't stack with itself") | unhappy (invalid input) |
| FR-013 | Effective conditions MUST be computed by the pipeline of FR-124, each kind once, ordered as in `ConditionKind`. | SRD 5.2.1, "Paralyzed", p. 186; "Unconscious", p. 191; "Immunity", pp. 17, 183 | happy |
| FR-014 | A `ConditionSet` MUST copy its input and return copies from its accessors (including `Explicit() []Condition`: at most one per kind, ordered by `ConditionKind`, FR-142), so mutating a passed-in or read-out slice never changes the set. | `doc/rules/architecture.md` §4; `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-015 | Two `ConditionSet`s with the same explicit conditions and immunities MUST be equal under `ConditionSet.Equal` (the type holds slices, so it is not comparable with `==`) regardless of supply order; sets differing in either MUST NOT be equal. | `doc/rules/architecture.md` §3 (value object) | happy |

### 4.3 Applying a condition

`Apply(target, condition)` is a `ConditionSet` method taking the creature's `CombatantID` and returning a closed `ApplyConditionToSetResult`: `ConditionAdded(set, events)`, `ConditionAlreadyHeld(set)`, `ConditionImmune(set)` or `ExhaustionAtMaximum(set)`. Precedence is FR-125.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-016 | Applying a condition whose kind is not effective and not blocked (FR-125) MUST return `ConditionAdded` holding it as an explicit condition, with one `ConditionApplied` event. | SRD 5.2.1, "Condition", p. 179 | happy |
| FR-017 | Applying any of the fourteen non-Exhaustion conditions to an empty set MUST give a set whose only explicit condition is that one (Exhaustion: FR-037). | SRD 5.2.1, "Condition", p. 179 | unhappy (empty and minimal) |
| FR-018 | Applying a kind already effective MUST return `ConditionAlreadyHeld`, set unchanged, no events: "a recipient either has a condition or doesn't". | SRD 5.2.1, "Condition", p. 179 | unhappy (rule interaction) |
| FR-019 | Applying `Charmed`, `Frightened` or `Grappled` with a different source (creature, or `Charmer`) while the kind is effective MUST return `ConditionAlreadyHeld` and keep the original source. | design decision D-9 (SRD silent); nearest text: SRD 5.2.1, "Condition", p. 179 | unhappy (rule interaction) |
| FR-020 | Applying a condition MUST NOT alter the set it was applied to. | `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-021 | Applying a kind in the set's stored immunities MUST return `ConditionImmune`, set unchanged, no events. | SRD 5.2.1, "Immunity", pp. 17, 183 | unhappy (rule interaction) |
| FR-022 | An immunity to one kind MUST NOT block a different kind; an empty immunity set MUST block nothing. | SRD 5.2.1, "Immunity", p. 17 | unhappy (boundary) |
| FR-023 | While Petrified is effective, Poisoned is an immunity: applying `Poisoned` MUST return `ConditionImmune`. | SRD 5.2.1, "Petrified", "Poison Immunity", p. 186 | unhappy (rule interaction) |
| FR-024 | A creature with explicit Poisoned when Petrified is applied MUST keep the Poisoned entry but exclude it from the effective conditions while Petrified is effective; once Petrified is removed Poisoned is effective again. | design decision D-5; nearest text: SRD 5.2.1, "Petrified", p. 186; "Immunity", pp. 17, 183 | unhappy (rule interaction) |
| FR-025 | Applying Exhaustion to a creature immune to Exhaustion MUST return `ConditionImmune` and add no level. | SRD 5.2.1, "Immunity", pp. 17, 183 | unhappy (rule interaction) |

### 4.4 Implied conditions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-026 | Paralyzed, Petrified, Stunned and Unconscious MUST each imply Incapacitated, Unconscious also Prone; no other condition implies another (table in 4.11). | SRD 5.2.1, "Paralyzed", "Petrified", p. 186; "Stunned", p. 189; "Unconscious", p. 191 | happy |
| FR-027 | Applying a kind effective only by implication (Incapacitated while Paralyzed; Prone while Unconscious) MUST return `ConditionAlreadyHeld`, set unchanged. | SRD 5.2.1, "Condition", p. 179 | unhappy (rule interaction) |
| FR-028 | When two explicit conditions both imply Incapacitated (Paralyzed and Stunned), it MUST appear once in the effective conditions and its effects count once. | SRD 5.2.1, "Condition", p. 179 | unhappy (rule interaction) |
| FR-029 | Removing Unconscious MUST end the implied Incapacitated (when nothing else implies it) but leave the creature Prone: the result set MUST hold Prone as an explicit condition ("When this condition ends, you remain Prone") and MUST NOT emit a `ConditionApplied` for it (consumers read the returned set, FR-127). | SRD 5.2.1, "Unconscious", "Inert", p. 191 | unhappy (rule interaction) |
| FR-030 | Removing Unconscious while another explicit condition also implies Incapacitated (Paralyzed) MUST leave Incapacitated effective. | SRD 5.2.1, "Paralyzed", p. 186; "Unconscious", p. 191 | unhappy (rule interaction) |

### 4.5 Removing a condition

`Remove(target, kind)` returns a closed `RemoveConditionFromSetResult`: `ConditionCleared(set, events)`, `ExhaustionLowered(set, events)`, `ConditionRemovalNotPresent(set)` or `ConditionRemovalImpliedBy(set, kinds)`. The variant is `ConditionCleared` so `ConditionRemoved` names only the event.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | Removing an explicit condition (other than Exhaustion above level 1, FR-036) MUST return `ConditionCleared` whose set no longer holds it, with one `ConditionRemoved` event. | SRD 5.2.1, "Condition", p. 179 ("various rules define how to end a condition") | happy |
| FR-032 | Removing a kind the creature does not have MUST return `ConditionRemovalNotPresent`, set unchanged, no events. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-033 | Removing a kind effective only by another condition's implication MUST return `ConditionRemovalImpliedBy` naming the implying kinds, set unchanged, no events. | SRD 5.2.1, "Paralyzed", p. 186; "Unconscious", p. 191 | unhappy (state error) |
| FR-034 | Removing a kind that is both explicit and implied MUST return `ConditionCleared` without the explicit entry and with one `ConditionRemoved`, while the kind stays effective. | SRD 5.2.1, "Paralyzed", p. 186 | unhappy (rule interaction) |
| FR-035 | Removing a condition MUST NOT alter the set it was removed from. | `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-036 | `Remove` of Exhaustion MUST give the same set and events as `ReduceExhaustion(1)` (FR-043), with this result's own variants: at level 2 or above `ExhaustionLowered` (level less 1, one `ConditionLevelReduced`); at level 1 `ConditionCleared` (one `ConditionRemoved`). | SRD 5.2.1, "Exhaustion", "Removing Exhaustion Levels", p. 181 ("Finishing a Long Rest removes 1 of your Exhaustion levels"); design decision D-15 | happy |

### 4.6 Exhaustion

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-037 | Applying Exhaustion to a creature without it MUST give level 1, and the `ConditionApplied` event MUST carry level 1, whatever level the passed `Exhaustion` holds (FR-123). | SRD 5.2.1, "Exhaustion", "Exhaustion Levels", p. 181 | happy |
| FR-038 | Applying Exhaustion at level `n` below 6 MUST give `n + 1` and one `ConditionApplied` carrying the new level: "each time you receive it, you gain 1 Exhaustion level" (table in 4.11). | SRD 5.2.1, "Exhaustion", "Exhaustion Levels", p. 181 | happy |
| FR-039 | Applying Exhaustion at level 6 (not immune, FR-125) MUST return `ExhaustionAtMaximum`, set unchanged, no events. | design decision D-3; nearest text: SRD 5.2.1, "Exhaustion", "Exhaustion Levels", p. 181 | unhappy (boundary) |
| FR-040 | `IsDeadFromExhaustion` MUST be true exactly when the level is 6, false at 1 to 5 and with no Exhaustion. | SRD 5.2.1, "Exhaustion", "Exhaustion Levels", p. 181 ("You die if your Exhaustion level is 6") | unhappy (boundary) |
| FR-041 | The D20 Test reduction MUST be 2 times the level (2, 4, 6, 8, 10, 12) and 0 with no Exhaustion. | SRD 5.2.1, "Exhaustion", "D20 Tests Affected", p. 181 | happy |
| FR-042 | The Speed reduction MUST be 5 feet times the level (5 to 30) and 0 with no Exhaustion. | SRD 5.2.1, "Exhaustion", "Speed Reduced", p. 181 | happy |
| FR-043 | `ReduceExhaustion(target, levels)` at level `n` with `levels < n` MUST return `ExhaustionReduced` with level `n - levels` and one `ConditionLevelReduced` carrying the new level. | SRD 5.2.1, "Exhaustion", "Removing Exhaustion Levels", p. 181 ("Finishing a Long Rest removes 1 of your Exhaustion levels"); "Long Rest", "Exhaustion Reduced", p. 185 | happy |
| FR-044 | `ReduceExhaustion` with `levels >= n` MUST return `ExhaustionEnded`: no Exhaustion in the set, one `ConditionRemoved`, no negative level ("When your Exhaustion level reaches 0, the condition ends"). | SRD 5.2.1, "Exhaustion", "Removing Exhaustion Levels", p. 181 | unhappy (boundary) |
| FR-045 | `ReduceExhaustion` with `levels` below 1 MUST return an error wrapping `srderr.ErrInvalidInput` (and the zero result), before any other check (FR-125). Validation lives in the domain method, not in `ReduceExhaustionOnSetCommand`, so the use-case span exists when it fails (FR-121). | design decision (SRD silent); nearest text: SRD 5.2.1, "Exhaustion", p. 181 | unhappy (invalid input) |
| FR-046 | `ReduceExhaustion` on a creature without Exhaustion MUST return `ExhaustionReductionNotPresent`, set unchanged, no events. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-047 | Reducing Exhaustion from level 6 by 1 MUST give level 5 and `IsDeadFromExhaustion` false. | SRD 5.2.1, "Exhaustion", p. 181 | unhappy (boundary) |

### 4.7 Effects of each condition

"Against" is an attack roll made against the creature with the condition; "own" is an attack roll or check it makes. Queries take the `ConditionSet` and the situational facts the SRD names. Each row names its SRD condition entry in the Source column (page of the entry in section 2).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | Blinded: the creature MUST automatically fail an ability check that requires sight. | "Blinded", "Can't See", p. 177 | happy |
| FR-049 | Blinded: it MUST NOT automatically fail an ability check requiring no sight (and Deafened alone MUST NOT fail a sight check). | "Blinded", "Can't See", p. 177 | unhappy (boundary) |
| FR-050 | Blinded: an attack roll against it MUST gain an Advantage contribution from Blinded. | "Blinded", "Attacks Affected", p. 177 | happy |
| FR-051 | Blinded: its own attack rolls MUST gain a Disadvantage contribution. | "Blinded", "Attacks Affected", p. 177 | happy |
| FR-052 | Charmed by a `CharmedByCombatant`: the creature MUST be forbidden to attack that charmer or target it with a damaging ability or magical effect. | "Charmed", "Can't Harm the Charmer", p. 178 | happy |
| FR-053 | Charmed: the restriction MUST NOT apply to another creature, or to a non-damaging effect on the charmer. | "Charmed", "Can't Harm the Charmer", p. 178 | unhappy (boundary) |
| FR-054 | Charmed by a `CharmedByCombatant`: the charmer MUST gain an Advantage contribution on an ability check to interact socially with it; any other creature MUST NOT. | "Charmed", "Social Advantage", p. 178 | happy |
| FR-055 | Deafened: the creature MUST automatically fail an ability check that requires hearing. | "Deafened", "Can't Hear", p. 181 | happy |
| FR-056 | Deafened: it MUST NOT automatically fail an ability check requiring no hearing. | "Deafened", "Can't Hear", p. 181 | unhappy (boundary) |
| FR-057 | Frightened: the creature MUST gain a Disadvantage contribution on its ability checks and attack rolls while the source of fear is within line of sight. | "Frightened", "Ability Checks and Attacks Affected", p. 182 | happy |
| FR-058 | Frightened: it MUST gain no such contribution while the source is not within line of sight. | "Frightened", "Ability Checks and Attacks Affected", p. 182 | unhappy (boundary) |
| FR-059 | Frightened: it MUST NOT willingly move closer to the source of fear and MUST be allowed to move closer to any other creature. | "Frightened", "Can't Approach", p. 182 | happy |
| FR-060 | Grappled: Speed MUST be 0, with the speed result `SpeedIncreaseForbidden` (FR-128). | "Grappled", "Speed 0", p. 182 | happy |
| FR-061 | Grappled: the creature MUST gain a Disadvantage contribution on an attack roll against any target other than its grappler. | "Grappled", "Attacks Affected", p. 182 | happy |
| FR-062 | Grappled: it MUST gain no contribution on an attack roll against its grappler. | "Grappled", "Attacks Affected", p. 182 | unhappy (boundary) |
| FR-064 | Incapacitated: the creature MUST be unable to take any action, Bonus Action or Reaction: `ConditionActionAvailabilityNone` in `ConditionFlags.Actions`. | "Incapacitated", "Inactive", p. 184 | happy |
| FR-065 | Incapacitated: Concentration MUST be reported `ConditionConcentrationBroken` in `ConditionFlags.Concentration`; a reported fact only, the state and rule are `casting-and-concentration.md`'s (FR-076; shared-types-and-ownership.md FR-011). | "Incapacitated", "No Concentration", p. 184 | happy |
| FR-066 | Incapacitated: it MUST be reported `SpeechUnable` in `ConditionFlags.Speech`. | "Incapacitated", "Speechless", p. 184 | happy |
| FR-067 | Incapacitated when rolling Initiative MUST gain a Disadvantage contribution on the roll. | "Incapacitated", "Surprised", p. 184 | happy |
| FR-068 | Invisible when rolling Initiative MUST gain an Advantage contribution on the roll. | "Invisible", "Surprise", p. 184 | happy |
| FR-069 | Invisible: the creature and its worn or carried equipment MUST NOT be a valid target of an effect requiring its target to be seen, unless the effect's creator can somehow see it. | "Invisible", "Concealed", p. 184 | happy |
| FR-070 | Invisible: an attack roll against it MUST gain a Disadvantage contribution from Invisible. | "Invisible", "Attacks Affected", p. 184 | happy |
| FR-071 | Invisible: its own attack rolls MUST gain an Advantage contribution from Invisible. | "Invisible", "Attacks Affected", p. 184 | happy |
| FR-072 | Invisible: against a creature that can somehow see it, neither the Disadvantage on attacks against it nor the Advantage on its own attacks applies (reading chosen in D-13). | "Invisible", "Attacks Affected", p. 184 | unhappy (rule interaction) |
| FR-126 | The Concealed protection of FR-069 MUST NOT apply when the effect's creator can somehow see the creature (`SightRelationCanSee`) and MUST apply when it cannot (`cannotSee`). | "Invisible", "Concealed", p. 184 | unhappy (rule interaction) |
| FR-073 | Paralyzed: Speed MUST be 0, with `SpeedIncreaseForbidden` (FR-128). | "Paralyzed", "Speed 0", p. 186 | happy |
| FR-074 | Paralyzed: it MUST automatically fail Strength and Dexterity saving throws. | "Paralyzed", "Saving Throws Affected", p. 186 | happy |
| FR-075 | Paralyzed: it MUST NOT automatically fail a saving throw with any other ability. | "Paralyzed", "Saving Throws Affected", p. 186 | unhappy (boundary) |
| FR-076 | Paralyzed: an attack roll against it MUST gain an Advantage contribution from Paralyzed. | "Paralyzed", "Attacks Affected", p. 186 | happy |
| FR-077 | Paralyzed: an attack roll that hits it MUST be a Critical Hit when the attacker is within 5 feet. | "Paralyzed", "Automatic Critical Hits", p. 186 | happy |
| FR-078 | Paralyzed: a hit from more than 5 feet away MUST NOT be a Critical Hit by this condition. | "Paralyzed", "Automatic Critical Hits", p. 186 | unhappy (boundary) |
| FR-079 | Petrified: Speed MUST be 0, with `SpeedIncreaseForbidden` (FR-128). | "Petrified", "Speed 0", p. 186 | happy |
| FR-080 | Petrified: an attack roll against it MUST gain an Advantage contribution from Petrified. | "Petrified", "Attacks Affected", p. 186 | happy |
| FR-081 | Petrified: it MUST automatically fail Strength and Dexterity saving throws, and no others. | "Petrified", "Saving Throws Affected", p. 186 | happy |
| FR-082 | Petrified: it MUST have Resistance to all damage, reported `AllDamageResistanceAll` in `ConditionFlags.Resistance` with no damage-type parameter; the consumer passes the flag to `hit-points-and-damage.md`, which interprets it (not imported here). | "Petrified", "Resist Damage", p. 186 | happy |
| FR-083 | Petrified: a hit MUST NOT be a Critical Hit by this condition at any distance (the SRD lists none). | "Petrified", p. 186 | unhappy (rule interaction) |
| FR-084 | Poisoned: its own attack rolls MUST gain a Disadvantage contribution. | "Poisoned", "Ability Checks and Attacks Affected", p. 186 | happy |
| FR-085 | Poisoned: its own ability checks MUST gain a Disadvantage contribution. | "Poisoned", "Ability Checks and Attacks Affected", p. 186 | happy |
| FR-086 | Prone: the only movement options MUST be to crawl or right itself by spending movement equal to half its Speed, rounded down (cases in 4.11). `StandUp` reports the closed `ProneMovement`: `CrawlOrRightItself(costFeet)` when effective Speed is above 0, so a cost is always present with the option. | "Prone", "Restricted Movement", p. 186; "Round Down", p. 187 | happy |
| FR-087 | Prone with Speed 0 MUST be unable to right itself: `StandUp` reports `CrawlOnly`, which carries no cost. | "Prone", "Restricted Movement", p. 186 | unhappy (boundary) |
| FR-088 | Prone: its own attack rolls MUST gain a Disadvantage contribution. | "Prone", "Attacks Affected", p. 186 | happy |
| FR-089 | Prone: an attack roll against it MUST gain an Advantage contribution when the attacker is within 5 feet. | "Prone", "Attacks Affected", p. 186 | happy |
| FR-090 | Prone: an attack roll against it MUST gain a Disadvantage contribution when the attacker is more than 5 feet away. | "Prone", "Attacks Affected", p. 186 | unhappy (boundary) |
| FR-091 | Restrained: Speed MUST be 0, with `SpeedIncreaseForbidden` (FR-128). | "Restrained", "Speed 0", p. 187 | happy |
| FR-092 | Restrained: an attack roll against it MUST gain an Advantage contribution from Restrained. | "Restrained", "Attacks Affected", p. 187 | happy |
| FR-093 | Restrained: its own attack rolls MUST gain a Disadvantage contribution. | "Restrained", "Attacks Affected", p. 187 | happy |
| FR-094 | Restrained: it MUST have Disadvantage on Dexterity saving throws. | "Restrained", "Saving Throws Affected", p. 187 | happy |
| FR-095 | Restrained: it MUST NOT be affected on a saving throw with any ability other than Dexterity. | "Restrained", "Saving Throws Affected", p. 187 | unhappy (boundary) |
| FR-096 | Stunned: it MUST automatically fail Strength and Dexterity saving throws. | "Stunned", "Saving Throws Affected", p. 189 | happy |
| FR-097 | Stunned: an attack roll against it MUST gain an Advantage contribution from Stunned. | "Stunned", "Attacks Affected", p. 189 | happy |
| FR-098 | Stunned MUST NOT set Speed to 0 by itself, and a hit on a Stunned creature MUST NOT be a Critical Hit by this condition (the SRD lists neither). | "Stunned", p. 189 | unhappy (rule interaction) |
| FR-099 | Unconscious: it MUST be reported `HeldItemsDropped` in `ConditionFlags.HeldItems`. | "Unconscious", "Inert", p. 191 | happy |
| FR-100 | Unconscious: Speed MUST be 0, with `SpeedIncreaseForbidden` (FR-128). | "Unconscious", "Speed 0", p. 191 | happy |
| FR-101 | Unconscious: an attack roll against it MUST gain an Advantage contribution from Unconscious. | "Unconscious", "Attacks Affected", p. 191 | happy |
| FR-102 | Unconscious: it MUST automatically fail Strength and Dexterity saving throws. | "Unconscious", "Saving Throws Affected", p. 191 | happy |
| FR-103 | Unconscious: a hit MUST be a Critical Hit when the attacker is within 5 feet and MUST NOT be when more than 5 feet away. | "Unconscious", "Automatic Critical Hits", p. 191 | unhappy (boundary) |
| FR-104 | Unconscious: it MUST be reported `AwarenessUnaware` in `ConditionFlags.Awareness`. | "Unconscious", "Unaware", p. 191 | happy |

### 4.8 Combining effects, pipeline and query contracts

Signatures are in section 5.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-105 | `ConditionEffects.AttackRoll(attacker, target, facts)` MUST merge attacker-side (own-attack) and target-side (attacks-against) contributions into one `RollContributions` whose `Mode()` is `modifier.Resolve` over all: any number of Advantage gives Advantage, any number of Disadvantage gives Disadvantage, at least one of each cancels to `normal` (cases in 4.11). | SRD 5.2.1, "Advantage", p. 176; "Disadvantage", p. 181 | unhappy (rule interaction) |
| FR-106 | `ConditionEffects.SavingThrow` MUST return per ability one `SavingThrowEffect`: `automaticFailure`, `disadvantage` or `unaffected`; `automaticFailure` takes precedence over `disadvantage` (Paralyzed with Restrained on Dexterity). | SRD 5.2.1, "Paralyzed", p. 186; "Restrained", p. 187 | unhappy (rule interaction) |
| FR-107 | `ConditionEffects.Speed` MUST return `EffectiveSpeed` whose `Value` is 0 when any effective condition sets Speed to 0, otherwise base Speed minus the Exhaustion reduction, never below 0; Speed 0 wins over Exhaustion. It takes one Speed and is applied to each named Speed independently (D-14). | SRD 5.2.1, "Exhaustion", "Speed Reduced", p. 181; "Grappled", p. 182; the floor and per-Speed application are design decisions D-8, D-14 | unhappy (rule interaction) |
| FR-108 | Contributions MUST name their `ConditionKind` and be ordered by it, so a breakdown is the same however the set was built. | `doc/rules/architecture.md` §6 | unhappy (determinism) |
| FR-109 | Every query on an empty `ConditionSet` MUST be neutral: no contributions, `unaffected` saving throws, Speed unchanged, `ConditionFlags` at neutral values (`intact`, `SpeechAble`, `available`, `HeldItemsKept`, `AwarenessAware`, `AllDamageResistanceNone`), `AutomaticCriticalNotAutomatic`. | SRD 5.2.1, "Condition", p. 179 | unhappy (empty and minimal) |
| FR-110 | A query given a negative attacker distance or base Speed MUST return an error wrapping `srderr.ErrInvalidInput` (and the zero value); 0 is accepted. `ConditionEffects` has no use case and no span, so this error is never telemetry. | n/a (package design); `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-111 | Queries MUST NOT alter the `ConditionSet`, and every returned collection MUST be a copy. | `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-112 | Queries MUST be deterministic and read no clock, randomness or I/O. | `doc/rules/architecture.md` §3 (purity) | unhappy (determinism) |
| FR-123 | `Apply(target, Exhaustion(level))` MUST ignore the passed level and add exactly one level (empty set: 1; level-2 set: 3). As an argument `Exhaustion(level)` only names the kind; stored state may hold any level 1 to 6 (FR-012). | design decision D-16; SRD 5.2.1, "Exhaustion", p. 181 | unhappy (rule interaction) |
| FR-124 | Effective conditions MUST be computed by this pipeline: (1) start from explicit conditions; (2) remove every explicit condition whose kind is in the stored immunities; (3) add every kind implied by those that remain, so a suppressed condition implies nothing; (4) remove every kind in the stored or effect-derived immunities (`ImmuneTo`, e.g. Petrified gives `ImmuneTo(Poisoned)`), the derived ones read from conditions remaining after step 2; (5) order by `ConditionKind`, each once. A kind removed in step 4 does not remove the immunities it would derive (no iteration). Example: explicit Paralyzed with an immunity to Paralyzed has no effective Paralyzed and no effective Incapacitated. | design decision D-5; SRD 5.2.1, "Immunity", p. 183 | unhappy (rule interaction) |
| FR-125 | `Apply` MUST decide in order: (1) `ConditionImmune` (stored or derived, FR-124), (2) `ConditionAlreadyHeld`, (3) `ExhaustionAtMaximum`, (4) `ConditionAdded`. `Remove` MUST decide: (1) explicit, so removed even when suppressed by immunity (FR-024): `ConditionCleared` or `ExhaustionLowered`; (2) effective by implication: `ConditionRemovalImpliedBy`; (3) `ConditionRemovalNotPresent`. `ReduceExhaustion` MUST decide: (1) `levels` below 1 returns an error, even with no Exhaustion; (2) no explicit Exhaustion: `ExhaustionReductionNotPresent`; (3) reduce, even when suppressed (the level is stored state). Table "Outcome precedence" in 4.11. | design decision D-17; SRD 5.2.1, "Condition", p. 179; "Immunity", p. 183 | unhappy (rule interaction) |
| FR-127 | `ConditionCleared` for Unconscious emits one `ConditionRemoved` and no event for the Prone that remains (FR-029). Consumers MUST read the returned set and MUST NOT infer its contents from events alone; a test MUST show that replaying only the events does not show Prone while the returned set holds it. | design decision D-7; SRD 5.2.1, "Unconscious", "Inert", p. 191 | unhappy (rule interaction) |
| FR-128 | `EffectiveSpeed.Increase` MUST be `SpeedIncreaseForbidden` when an effective condition sets Speed to 0 (Grappled, Paralyzed, Petrified, Restrained, Unconscious), else `SpeedIncreaseAllowed`. | SRD 5.2.1, "Grappled", p. 182 ("Speed 0 ... can't increase"); "Paralyzed", "Petrified", p. 186; "Restrained", p. 187; "Unconscious", p. 191 | happy |

### 4.9 Domain events

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-113 | `ConditionApplied` MUST be an immutable `ConditionEvent` member carrying the target `CombatantID` and the applied `Condition` (for Exhaustion, the level reached), comparable by value; `Kind()` is `condition_applied`, `Combatants()` is `[target]`. | `doc/rules/architecture.md` §3 (domain event) | happy |
| FR-114 | `ConditionRemoved` MUST carry the target and the `ConditionKind` that ended; `ConditionLevelReduced` the target and the new level. Both are immutable, comparable by value, kinds `condition_removed` and `condition_level_reduced`, `Combatants()` `[target]`; past-tense (shared-types-and-ownership.md FR-019). | `doc/rules/architecture.md` §3; SRD 5.2.1, "Exhaustion", p. 181 | happy |
| FR-115 | An operation that changes nothing (`ConditionAlreadyHeld`, `ConditionImmune`, `ExhaustionAtMaximum`, `ConditionRemovalNotPresent`, `ExhaustionReductionNotPresent`, `ConditionRemovalImpliedBy`) MUST return an empty events list, and a change's events list MUST be a copy (mutating it leaves the result unchanged). | `doc/rules/architecture.md` §3 (state transitions) | unhappy (empty and minimal) |
| FR-129 | `ConditionApplied`, `ConditionRemoved` and `ConditionLevelReduced` MUST be the single event set for conditions in `combat`: members of the closed `ConditionEvent`, which extends `Event` (combat-foundations.md FR-015 to FR-019). `GrappleEnded` is `grappling.md`'s. Later specs reference them and MUST NOT redefine them; a spec needing an encounter id or round wraps them (D-12). | `doc/rules/architecture.md` §3; shared-types-and-ownership.md FR-019 | happy |

### 4.10 Use cases and telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-116 | `ApplyConditionToSet` MUST return the same outcome, set and events as the domain `Apply`. | `doc/rules/architecture.md` §4 | happy |
| FR-117 | `RemoveConditionFromSet` MUST match the domain `Remove`, and `ReduceExhaustionOnSet` the domain `ReduceExhaustion`, in outcome, set and events. | `doc/rules/architecture.md` §4 | happy |
| FR-118 | Each use case MUST give identical results with the zero-value `telemetry.Telemetry` (all no-ops) and a recording one from `telemetrytest`. | `doc/rules/observability.md` §9 | happy |
| FR-119 | Each use case MUST hold the injected `telemetry.Telemetry` (default `telemetry.Noop`), build a `Telemetry` and run each call through it (combat-foundations.md FR-035) with operation `apply_condition_to_set`, `remove_condition_from_set` or `reduce_exhaustion_on_set`, recording exactly what combat-foundations.md FR-029 to FR-031 fix (span `srd5e.combat.<operation>`, `srd5e.combat.operations`, `srd5e.combat.operation_duration`, `srd5e.combat.events` per event with `srd5e.combat.event` = event `kind`). This spec defines no other shared signal. Its one extra signal is the histogram `srd5e.combat.condition.active` (FR-122), recorded by the use case through its own `Meter` after `Telemetry` returns (value = effective-condition count of the returned set; also on refused paths; skipped on the error path, FR-121). | combat-foundations.md FR-027 to FR-031, FR-035; `doc/rules/observability.md` §5 | happy |
| FR-120 | Each result variant MUST map to one `srd5e.combat.result` name and one `ResultClass` per the "Result classification" table (a closed set, not derived from the class name, combat-foundations.md FR-030). `ConditionAlreadyHeld`, `ConditionImmune`, `ExhaustionAtMaximum`, `ConditionRemovalNotPresent`, `ExhaustionReductionNotPresent` and `ConditionRemovalImpliedBy` are `refused`: `srd5e.outcome` is `failure`, the span status MUST NOT be error, no error counter MUST be incremented and nothing MUST be logged at `INFO` or above (combat-foundations.md FR-031, FR-034). No variant is `invalid`. | combat-foundations.md FR-030, FR-031, FR-034; `doc/rules/observability.md` §6, §7 | unhappy (state error) |
| FR-121 | An error from `ReduceExhaustion` (FR-045) inside `ReduceExhaustionOnSet` MUST be returned unchanged (a validation error, not a port failure, so it is not wrapped with `srderr.WrapPort`) and the span MUST record it; no `srd5e.combat.errors` increment and no log MUST be recorded (combat-foundations.md FR-035). The duration sample IS recorded; `operations`, `events` and `condition.active` are NOT (no result). Domain queries (FR-110) have no span. | combat-foundations.md FR-035; `doc/rules/observability.md` §7 | unhappy (invalid input) |
| FR-122 | Telemetry attributes MUST come from closed sets: three operation names, eleven result names, three event kinds, and for `srd5e.combat.condition.active` the `srd5e.combat.condition.kind` attribute (the `ConditionKind` snake_case name the call was about; always `exhaustion` for `reduce_exhaustion_on_set`). The histogram records the effective-condition count after the call with `srd5e.combat.operation` and `srd5e.combat.condition.kind`. Attributes MUST NOT contain combatant ids (combat-foundations.md FR-033). | combat-foundations.md FR-029, FR-033; `doc/rules/observability.md` §5 | unhappy (invalid input) |

### 4.11 Case tables

Each row is one table-driven test. A Source page is read as `SRD 5.2.1, "<section>", p. n` with the section named in the Source of the row's Req. ids (testing.md §5); rows verifying a package rule say `n/a (package design)`.

**Result classification** (FR-119, FR-120). No row sets a span error; the returned-error row's error is recorded by the span.

| Variant | Operation | `srd5e.combat.result` | `ResultClass` | `srd5e.outcome` | Req. | Path |
|---|---|---|---|---|---|---|
| `ConditionAdded` | `apply_condition_to_set` | `added` | `applied` | `success` | FR-119 | happy |
| `ConditionAlreadyHeld` | `apply_condition_to_set` | `already_held` | `refused` | `failure` | FR-120 | unhappy (rule interaction) |
| `ConditionImmune` | `apply_condition_to_set` | `immune` | `refused` | `failure` | FR-120 | unhappy (rule interaction) |
| `ExhaustionAtMaximum` | `apply_condition_to_set` | `at_maximum` | `refused` | `failure` | FR-120 | unhappy (boundary) |
| `ConditionCleared` | `remove_condition_from_set` | `cleared` | `applied` | `success` | FR-119 | happy |
| `ExhaustionLowered` | `remove_condition_from_set` | `lowered` | `applied` | `success` | FR-119 | happy |
| `ConditionRemovalNotPresent` | `remove_condition_from_set` | `removal_not_present` | `refused` | `failure` | FR-120 | unhappy (state error) |
| `ConditionRemovalImpliedBy` | `remove_condition_from_set` | `implied_by` | `refused` | `failure` | FR-120 | unhappy (state error) |
| `ExhaustionReduced` | `reduce_exhaustion_on_set` | `reduced` | `applied` | `success` | FR-119 | happy |
| `ExhaustionEnded` | `reduce_exhaustion_on_set` | `ended` | `applied` | `success` | FR-119 | unhappy (boundary) |
| `ExhaustionReductionNotPresent` | `reduce_exhaustion_on_set` | `reduction_not_present` | `refused` | `failure` | FR-120 | unhappy (state error) |
| returned validation error | `reduce_exhaustion_on_set` | none recorded | none | none | FR-121 | unhappy (invalid input) |

**Implied conditions** (FR-026, FR-027, FR-028)

| Label | Explicit | Effective | Req. | Source | Path |
|---|---|---|---|---|---|
| Paralyzed implies Incapacitated | Paralyzed | Paralyzed, Incapacitated | FR-026 | p. 186 | happy |
| Petrified implies Incapacitated | Petrified | Incapacitated, Petrified | FR-026 | p. 186 | happy |
| Stunned implies Incapacitated | Stunned | Incapacitated, Stunned | FR-026 | p. 189 | happy |
| Unconscious implies Incapacitated and Prone | Unconscious | Incapacitated, Prone, Unconscious | FR-026 | p. 191 | happy |
| Blinded implies nothing | Blinded | Blinded | FR-026 | p. 177 | unhappy (boundary) |
| Prone implies nothing | Prone | Prone | FR-026 | p. 186 | unhappy (boundary) |
| Paralyzed and Stunned | Paralyzed, Stunned | Incapacitated, Paralyzed, Stunned | FR-028 | pp. 186, 189 | unhappy (rule interaction) |
| Unconscious and Prone explicit | Unconscious, Prone | Incapacitated, Prone, Unconscious | FR-027, FR-028 | pp. 186, 191 | unhappy (rule interaction) |

**Exhaustion by level** (FR-037 to FR-042); all p. 181

| Label | Level before | Level after applying | D20 Test reduction | Speed reduction (ft) | Dead | Req. | Path |
|---|---|---|---|---|---|---|---|
| none to 1 | none | 1 | 2 | 5 | no | FR-037, FR-041, FR-042 | happy |
| 1 to 2 | 1 | 2 | 4 | 10 | no | FR-038, FR-041, FR-042 | happy |
| 2 to 3 | 2 | 3 | 6 | 15 | no | FR-038, FR-041, FR-042 | happy |
| 3 to 4 | 3 | 4 | 8 | 20 | no | FR-038, FR-041, FR-042 | happy |
| 4 to 5 | 4 | 5 | 10 | 25 | no | FR-038, FR-040, FR-041, FR-042 | unhappy (boundary) |
| 5 to 6 | 5 | 6 | 12 | 30 | yes | FR-038, FR-040, FR-041, FR-042 | unhappy (boundary) |
| 6 stays 6 | 6 | 6 (`ExhaustionAtMaximum`) | 12 | 30 | yes | FR-039 | unhappy (boundary) |
| no Exhaustion | none | none | 0 | 0 | no | FR-040, FR-041, FR-042 | unhappy (empty and minimal) |

**Reducing Exhaustion** (FR-043 to FR-047); all p. 181

| Label | Level before | Levels removed | Result | Event | Req. | Path |
|---|---|---|---|---|---|---|
| Long Rest at 3 | 3 | 1 | level 2 | `ConditionLevelReduced(2)` | FR-043 | happy |
| two levels at 5 | 5 | 2 | level 3 | `ConditionLevelReduced(3)` | FR-043 | happy |
| from 6 | 6 | 1 | level 5, not dead | `ConditionLevelReduced(5)` | FR-047 | unhappy (boundary) |
| level 1 reaches 0 | 1 | 1 | condition ends | `ConditionRemoved(Exhaustion)` | FR-044 | unhappy (boundary) |
| exactly to 0 | 3 | 3 | condition ends | `ConditionRemoved(Exhaustion)` | FR-044 | unhappy (boundary) |
| more than present | 2 | 5 | condition ends | `ConditionRemoved(Exhaustion)` | FR-044 | unhappy (boundary) |
| zero levels | 3 | 0 | validation error | none | FR-045 | unhappy (invalid input) |
| negative levels | 3 | -1 | validation error | none | FR-045 | unhappy (invalid input) |
| no Exhaustion | none | 1 | `ExhaustionReductionNotPresent` | none | FR-046 | unhappy (state error) |

**Attacks against a creature** (FR-050, FR-070, FR-072, FR-076 to FR-078, FR-080, FR-083, FR-089, FR-090, FR-092, FR-097, FR-098, FR-101, FR-103, FR-109). Target's conditions only; "crit" is the automatic Critical Hit on a hit; "sees" is whether the attacker can somehow see the target.

| Label | Target conditions | Distance (ft) | Sees | Contributions | Crit | Req. | Source | Path |
|---|---|---|---|---|---|---|---|---|
| Blinded | Blinded | 30 | no | Advantage | no | FR-050 | p. 177 | happy |
| Invisible | Invisible | 30 | no | Disadvantage | no | FR-070 | p. 184 | happy |
| Invisible, attacker can see | Invisible | 30 | yes | none | no | FR-072 | p. 184 | unhappy (rule interaction) |
| Paralyzed adjacent | Paralyzed | 5 | no | Advantage | yes | FR-076, FR-077 | p. 186 | happy |
| Paralyzed, 6 feet | Paralyzed | 6 | no | Advantage | no | FR-076, FR-078 | p. 186 | unhappy (boundary) |
| Paralyzed, 0 feet | Paralyzed | 0 | no | Advantage | yes | FR-077 | p. 186 | unhappy (boundary) |
| Petrified adjacent | Petrified | 5 | no | Advantage | no | FR-080, FR-083 | p. 186 | unhappy (rule interaction) |
| Prone, 5 feet | Prone | 5 | no | Advantage | no | FR-089 | p. 186 | unhappy (boundary) |
| Prone, 6 feet | Prone | 6 | no | Disadvantage | no | FR-090 | p. 186 | unhappy (boundary) |
| Prone, 0 feet | Prone | 0 | no | Advantage | no | FR-089 | p. 186 | unhappy (boundary) |
| Restrained | Restrained | 30 | no | Advantage | no | FR-092 | p. 187 | happy |
| Stunned adjacent | Stunned | 5 | no | Advantage | no | FR-097, FR-098 | p. 189 | unhappy (rule interaction) |
| Unconscious adjacent | Unconscious | 5 | no | Advantage (Unconscious), Advantage (Prone) | yes | FR-089, FR-101, FR-103 | p. 191 | happy |
| Unconscious, 6 feet | Unconscious | 6 | no | Advantage (Unconscious), Disadvantage (Prone) | no | FR-090, FR-101, FR-103 | p. 191 | unhappy (rule interaction) |
| no conditions | none | 5 | no | none | no | FR-109 | p. 179 | unhappy (empty and minimal) |

**A creature's own attack rolls** (FR-051, FR-057, FR-058, FR-061, FR-062, FR-071, FR-072, FR-084, FR-088, FR-093)

| Label | Attacker conditions | Situation | Contributions | Req. | Source | Path |
|---|---|---|---|---|---|---|
| Blinded | Blinded | any target | Disadvantage | FR-051 | p. 177 | happy |
| Frightened, source in sight | Frightened | source in line of sight | Disadvantage | FR-057 | p. 182 | happy |
| Frightened, source out of sight | Frightened | source not in line of sight | none | FR-058 | p. 182 | unhappy (boundary) |
| Grappled, other target | Grappled | target is not the grappler | Disadvantage | FR-061 | p. 182 | happy |
| Grappled, the grappler | Grappled | target is the grappler | none | FR-062 | p. 182 | unhappy (boundary) |
| Invisible | Invisible | target cannot see attacker | Advantage | FR-071 | p. 184 | happy |
| Invisible, target can see | Invisible | target can somehow see attacker | none | FR-072 | p. 184 | unhappy (rule interaction) |
| Poisoned, Prone, Restrained (one row each) | that condition | any target | Disadvantage | FR-084, FR-088, FR-093 | pp. 186, 186, 187 | happy |

**Ability checks** (FR-048, FR-049, FR-054 to FR-058, FR-085)

| Label | Conditions | Check requires | Result | Req. | Source | Path |
|---|---|---|---|---|---|---|
| Blinded, sight | Blinded | sight | automatic failure | FR-048 | p. 177 | happy |
| Blinded, no sense | Blinded | no sense | none | FR-049 | p. 177 | unhappy (boundary) |
| Blinded, hearing | Blinded | hearing | none | FR-049 | p. 177 | unhappy (boundary) |
| Deafened, hearing | Deafened | hearing | automatic failure | FR-055 | p. 181 | happy |
| Deafened, sight | Deafened | sight | none | FR-056 | p. 181 | unhappy (boundary) |
| Blinded and Deafened | Blinded, Deafened | sight and hearing | automatic failure | FR-048, FR-055 | pp. 177, 181 | unhappy (rule interaction) |
| Poisoned | Poisoned | no sense | Disadvantage | FR-085 | p. 186 | happy |
| Frightened, source in sight | Frightened | source in line of sight | Disadvantage | FR-057 | p. 182 | happy |
| Frightened, source out of sight | Frightened | source not in line of sight | none | FR-058 | p. 182 | unhappy (boundary) |
| Charmed by A: A interacts socially | Charmed by creature A | social, with A | Advantage for A | FR-054 | p. 178 | happy |
| Charmed by A: other creature B interacts socially | Charmed by creature A | social, with B | none | FR-054 | p. 178 | unhappy (boundary) |

**Saving throws** (FR-074, FR-075, FR-081, FR-094 to FR-096, FR-102, FR-106, FR-109). S and D are Strength and Dexterity; "other" is each of the other four abilities.

| Label | Conditions | Ability | Result | Req. | Source | Path |
|---|---|---|---|---|---|---|
| Paralyzed S and D | Paralyzed | S, D | automatic failure | FR-074 | p. 186 | happy |
| Paralyzed other | Paralyzed | other | unaffected | FR-075 | p. 186 | unhappy (boundary) |
| Petrified S and D | Petrified | S, D | automatic failure | FR-081 | p. 186 | happy |
| Petrified other | Petrified | other | unaffected | FR-081 | p. 186 | unhappy (boundary) |
| Stunned S and D | Stunned | S, D | automatic failure | FR-096 | p. 189 | happy |
| Stunned other | Stunned | other | unaffected | FR-096 | p. 189 | unhappy (boundary) |
| Unconscious S and D | Unconscious | S, D | automatic failure | FR-102 | p. 191 | happy |
| Unconscious other | Unconscious | other | unaffected | FR-102 | p. 191 | unhappy (boundary) |
| Restrained, Dexterity | Restrained | D | Disadvantage | FR-094 | p. 187 | happy |
| Restrained, Strength and other | Restrained | S, other | unaffected | FR-095 | p. 187 | unhappy (boundary) |
| Paralyzed and Restrained, Dexterity | Paralyzed, Restrained | D | automatic failure | FR-106 | pp. 186, 187 | unhappy (rule interaction) |
| none | none | any | unaffected | FR-109 | p. 179 | unhappy (empty and minimal) |

**Combining roll contributions** (FR-105, FR-108). A is the count of Advantage contributions, D of Disadvantage.

| Label | Situation | A | D | Mode | Req. | Source | Path |
|---|---|---|---|---|---|---|---|
| attacker Poisoned and Blinded | own attack, two Disadvantages | 0 | 2 | disadvantage | FR-105 | pp. 177, 186 | unhappy (rule interaction) |
| target Restrained and Paralyzed | attack against, two Advantages | 2 | 0 | advantage | FR-105 | pp. 186, 187 | unhappy (rule interaction) |
| attacker Blinded, target Paralyzed | one each | 1 | 1 | normal | FR-105 | pp. 177, 186 | unhappy (rule interaction) |
| attacker Poisoned, target Blinded and Restrained | one Disadvantage, two Advantages | 2 | 1 | normal | FR-105 | pp. 177, 186, 187 | unhappy (rule interaction) |
| attacker Invisible, target Prone at 6 feet | one each | 1 | 1 | normal | FR-105 | pp. 184, 186 | unhappy (rule interaction) |
| built in two orders | same set | any | any | same mode and contribution order | FR-108 | p. 176 | unhappy (determinism) |

**Effective conditions pipeline** (FR-013, FR-124). Sources: "Immunity" p. 183 and the entries named.

| Label | Explicit | Stored immunities | Effective | Req. | Source | Path |
|---|---|---|---|---|---|---|
| suppressed explicit implies nothing | Paralyzed | Paralyzed | none | FR-124 | pp. 183, 186 | unhappy (rule interaction) |
| implied kind blocked | Stunned | Incapacitated | Stunned | FR-124 | pp. 183, 189 | unhappy (rule interaction) |
| Petrified suppresses Poisoned | Petrified, Poisoned | none | Incapacitated, Petrified | FR-024, FR-124 | p. 186 | unhappy (rule interaction) |
| Petrified itself suppressed | Petrified, Poisoned | Petrified | Poisoned | FR-124 | pp. 183, 186 | unhappy (rule interaction) |
| implied Prone blocked | Unconscious | Prone | Incapacitated, Unconscious | FR-124 | pp. 183, 191 | unhappy (rule interaction) |
| duplicate implication, one effect | Paralyzed, Stunned | none | Incapacitated, Paralyzed, Stunned | FR-028, FR-124 | pp. 186, 189 | unhappy (rule interaction) |

**Outcome precedence** (FR-036, FR-045, FR-046, FR-123, FR-125). One call on the state shown.

| Label | Operation | State | Outcome | Req. | Source | Path |
|---|---|---|---|---|---|---|
| Poisoned suppressed by Petrified | `Apply(Poisoned)` | explicit Poisoned, Petrified | `ConditionImmune` (not `ConditionAlreadyHeld`) | FR-125 | p. 186 | unhappy (rule interaction) |
| Exhaustion 6 and immune | `Apply(Exhaustion(1))` | Exhaustion 6, immune | `ConditionImmune` (not `ExhaustionAtMaximum`) | FR-125 | p. 183 | unhappy (rule interaction) |
| Exhaustion 6, not immune | `Apply(Exhaustion(1))` | Exhaustion 6 | `ExhaustionAtMaximum` | FR-039, FR-125 | p. 181 | unhappy (boundary) |
| implied kind | `Apply(Incapacitated)` | Paralyzed | `ConditionAlreadyHeld` | FR-027, FR-125 | p. 186 | unhappy (rule interaction) |
| level-3 argument, empty set | `Apply(Exhaustion(3))` | empty | `ConditionAdded`, level 1 | FR-123 | p. 181 | unhappy (rule interaction) |
| level-3 argument, level-2 set | `Apply(Exhaustion(3))` | Exhaustion 2 | `ConditionAdded`, level 3 | FR-123 | p. 181 | unhappy (rule interaction) |
| remove explicit but suppressed | `Remove(Poisoned)` | explicit Poisoned, Petrified | `ConditionCleared`, one `ConditionRemoved` | FR-125 | n/a (package design) | unhappy (rule interaction) |
| remove implied kind | `Remove(Incapacitated)` | Paralyzed | `ConditionRemovalImpliedBy([Paralyzed])` | FR-033 | p. 186 | unhappy (state error) |
| remove Exhaustion at 3 | `Remove(Exhaustion)` | Exhaustion 3 | `ExhaustionLowered`, level 2, `ConditionLevelReduced(2)` | FR-036 | p. 181 | unhappy (boundary) |
| remove Exhaustion at 1 | `Remove(Exhaustion)` | Exhaustion 1 | `ConditionCleared`, `ConditionRemoved(Exhaustion)` | FR-036 | p. 181 | unhappy (boundary) |
| reduce 0, no Exhaustion | `ReduceExhaustion(0)` | empty | validation error (not `ExhaustionReductionNotPresent`) | FR-045, FR-125 | n/a (package design) | unhappy (invalid input) |
| reduce 1, no Exhaustion | `ReduceExhaustion(1)` | empty | `ExhaustionReductionNotPresent` | FR-046, FR-125 | n/a (package design) | unhappy (state error) |
| reduce, Exhaustion suppressed | `ReduceExhaustion(1)` | Exhaustion 3, immune | `ExhaustionReduced`, level 2 | FR-125 | n/a (package design) | unhappy (rule interaction) |
| Unconscious removed, Prone kept | `Remove(Unconscious)` | Unconscious | `ConditionCleared`; set holds Prone; events only `ConditionRemoved(Unconscious)` | FR-029, FR-127 | p. 191 | unhappy (rule interaction) |

**Speed** (FR-042, FR-060, FR-073, FR-079, FR-091, FR-098, FR-100, FR-107, FR-110). Base Speed in feet.

| Label | Conditions | Base | Effective | Req. | Source | Path |
|---|---|---|---|---|---|---|
| Grappled; Paralyzed; Petrified; Restrained; Unconscious (one row each) | that condition | 30 | 0 | FR-060, FR-073, FR-079, FR-091, FR-100 | pp. 182, 186, 186, 187, 191 | happy |
| Stunned only | Stunned | 30 | 30 | FR-098 | p. 189 | unhappy (rule interaction) |
| Exhaustion 1 | Exhaustion 1 | 30 | 25 | FR-042 | p. 181 | happy |
| Exhaustion 5 | Exhaustion 5 | 30 | 5 | FR-042 | p. 181 | unhappy (boundary) |
| Exhaustion 6 | Exhaustion 6 | 30 | 0 | FR-042 | p. 181 | unhappy (boundary) |
| Exhaustion above base | Exhaustion 4 | 15 | 0 | FR-107 | p. 181 | unhappy (boundary) |
| Speed 0 beats Exhaustion | Grappled, Exhaustion 2 | 30 | 0 | FR-107 | pp. 181, 182 | unhappy (rule interaction) |
| base 0 | none | 0 | 0 | FR-110 | n/a (package design) | unhappy (boundary) |
| negative base | none | -5 | validation error | FR-110 | n/a (package design) | unhappy (invalid input) |

**Righting a Prone creature** (FR-086, FR-087, FR-107). Effective Speed; cost is half, rounded down.

| Label | Effective Speed (ft) | Result | Req. | Source | Path |
|---|---|---|---|---|---|
| Speed 30 | 30 | stand for 15 feet of movement | FR-086 | p. 186 | happy |
| Speed 25 rounds down | 25 | stand for 12 feet | FR-086 | p. 187 | unhappy (boundary) |
| Speed 1 | 1 | stand for 0 feet | FR-086 | p. 187 | unhappy (boundary) |
| Speed 0 | 0 | cannot right itself | FR-087 | p. 186 | unhappy (boundary) |
| Prone and Grappled | 0 (Grappled) | cannot right itself | FR-087, FR-107 | pp. 182, 186 | unhappy (rule interaction) |
| crawl always available | any | crawl is a movement option | FR-086 | p. 186 | happy |

The automatic Critical Hit boundary (FR-077, FR-078, FR-083, FR-103) is covered by the distance rows 0, 5 and 6 of the attacks-against table.

### 4.12 Package-level additions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-130 | Every ordering by `ConditionKind` (FR-013, FR-108, FR-124) MUST use the declaration order of the `content`-owned enum, which MUST be the Glossary order. A test over all values MUST fail if the fifteen names are not in Glossary order. | SRD 5.2.1, "Condition", p. 179; shared-types-and-ownership.md FR-005 | unhappy (determinism) |
| FR-131 | `ConditionEffects` and the `ConditionSet` operations MUST import nothing from `attacks.md`, `initiative-and-actions.md`, `encounter-aggregate.md`, `combat-action-resolution.md` or `hit-points-and-damage.md`, and take no type from a downstream context. Their outputs (`RollContributions`, `SavingThrowEffect`, `AbilityCheckEffect`, `AutomaticCritical`, `ConditionFlags`) are plain values; `combat-action-resolution.md` is the only caller of `ConditionEffects` together with `attacks.md`. | n/a (package design); shared-types-and-ownership.md FR-013, FR-015, FR-026 | unhappy (forbidden edge) |
| FR-139 | `ConditionEvent` MUST be a closed interface that extends `Event` (own unexported marker method), declared in `condition_event.go` of package `combat`, with exactly the members `ConditionApplied`, `ConditionRemoved`, `ConditionLevelReduced` (named in its doc comment). Each member MUST be an immutable value (unexported fields, value receivers), have the constant `Kind()` of FR-113/FR-114 (unique across the package, matching `[a-z][a-z0-9_]*`, not from reflection), and return `Combatants()` as a copy, duplicate-free and ordered. No event here implements `Event` other than through `ConditionEvent`. | combat-foundations.md FR-015 to FR-019; shared-types-and-ownership.md FR-019 | happy |
| FR-140 | `ApplyConditionToSetResult`, `RemoveConditionFromSetResult` and `ReduceExhaustionOnSetResult` with all variants MUST be declared in `condition_result.go` (commands live in the use-case files, `*_usecase.go`), and the use-case files re-use them; domain files MUST NOT reference use-case types (architecture check). | `doc/rules/architecture.md` §1, §9; shared-types-and-ownership.md FR-017 | unhappy (forbidden edge) |
| FR-141 | `Charmed` MUST carry a closed `Charmer` interface: `CharmedByCombatant{CombatantID}` or `CharmedByItem{itemID}`, the id an opaque non-blank string the host supplies (`NewCharmedByItem(itemID string) (CharmedByItem, error)` returns an error wrapping `srderr.ErrInvalidInput` when blank) and this spec never interprets. Both are comparable with `==`. `CanTarget` (FR-052, FR-053) and `SocialCheck` (FR-054) act only for `CharmedByCombatant`: a `CharmedByItem` charmer forbids no combatant and grants no Advantage to any combatant. | SRD 5.2.1, "Charmed", p. 178; "Conflict", p. 208 (a sentient item's control effect gives the Charmed condition); `shared-types-and-ownership.md` FR-013 (`combat` is above `character`, so no `character` id type is named) | unhappy (rule interaction) |
| FR-142 | `ConditionSet` MUST expose `Explicit() []Condition`: a copy of the explicit conditions, at most one per kind, ordered by `ConditionKind` (FR-130). Callers read a source from it, for example the `Grappled` variant's grappler (`grappling.md` FR-089), because `ConditionRemoved(target, kind)` carries no source. | `doc/rules/architecture.md` §4; SRD 5.2.1, "Condition", p. 179 | happy |

### Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-schema-and-decoder.md (FR-110) |
| FR-005 | Withdrawn: moved to content-schema-and-decoder.md (FR-110) |
| FR-006 | Withdrawn: moved to content-schema-and-decoder.md (FR-110) |
| FR-063 | Withdrawn: moved to grappling.md (FR-062) |
| FR-132 | Withdrawn: moved to grappling.md (FR-035, FR-036) |
| FR-133 | Withdrawn: moved to grappling.md (FR-033, FR-038) |
| FR-134 | Withdrawn: moved to grappling.md (FR-039) |
| FR-135 | Withdrawn: moved to grappling.md (FR-034, FR-043) |
| FR-136 | Withdrawn: moved to grappling.md (FR-029) |
| FR-137 | Withdrawn: moved to grappling.md (FR-049, FR-050) |
| FR-138 | Withdrawn: moved to grappling.md (FR-081) |

## 5. Model

Context `combat`, files `condition*.go` of package `combat`, third in the `combat` order (shared-types-and-ownership.md FR-015). All types are immutable values (unexported fields, value receivers), pure and synchronous.

- **`ConditionKind`**: not defined here; imported from package `content`; only its declaration order is relied on (FR-130).
- **`Condition`**: closed interface, one struct type per kind (FR-002); `Charmed` carries a closed `Charmer` (FR-141); `Frightened`, `Grappled` carry a `CombatantID` (FR-003); `Exhaustion` an `ExhaustionLevel`. Equality by value (FR-004).
- **`ExhaustionLevel`**: int 1 to 6 (FR-008 to FR-010); "no Exhaustion" is the absence of the condition.
- **`ConditionDefinition`**: one constant per kind listing `implies` and `effects` (FR-007), variants of the closed `ConditionEffect`: `AutoFailAbilityChecks(requiring: Sense)`; `AttackRollsAgainst`, `OwnAttackRolls`, `OwnAbilityChecks` (each `circumstance, when`, `when` a closed `EffectCondition`: always, attackerWithin5Feet, attackerBeyond5Feet, sourceInLineOfSight, targetIsNotGrappler, unlessSeen); `SavingThrowsAutoFail(abilities)`, `SavingThrowDisadvantage(ability)`; `SpeedZero`, `SpeedReducedBy(feetPerLevel)`, `D20TestReducedBy(perLevel)`; `AutomaticCriticalHit(withinFeet)`; `NoActions`, `ConcentrationBroken`, `CannotSpeak`; `InitiativeCircumstance(circumstance)`; `CharmerRestriction`, `SocialAdvantageForSource`, `CannotApproachSource`, `Concealed`; `DamageResistanceAll`, `ImmuneTo(kind)`, `DropsHeldItems`, `Unaware`. The 4.7 table is the definition; the catalogue holds no rule it does not state.
- **`ConditionSet`**: explicit conditions (one per kind, FR-012; read through `Explicit`, FR-142), an `Immunities` set, derived `Effective` kinds (FR-013, FR-124); no id; equality by value, order-insensitive (FR-015). Operations `Apply`, `Remove`, `ReduceExhaustion` take the creature's `CombatantID` as `target`.
- **Results** (closed variant sets, shared-types-and-ownership.md FR-017, FR-018), declared in `condition_result.go` and re-used by the use-case files (FR-140); the domain operation returns the use case's own result type (D-20). `ApplyConditionToSetResult` = `ConditionAdded` | `ConditionAlreadyHeld` | `ConditionImmune` | `ExhaustionAtMaximum`; `RemoveConditionFromSetResult` = `ConditionCleared` | `ExhaustionLowered` | `ConditionRemovalNotPresent` | `ConditionRemovalImpliedBy`; `ReduceExhaustionOnSetResult` = `ExhaustionReduced` | `ExhaustionEnded` | `ExhaustionReductionNotPresent`. No generic variant name is exported. Each change variant carries the new set and a copied events slice.
- **Events** (FR-129, FR-139): `ConditionApplied(target, condition)`, `ConditionRemoved(target, kind)`, `ConditionLevelReduced(target, level)`; kinds `condition_applied`, `condition_removed`, `condition_level_reduced`; `Combatants()` is `[target]`.
- **Named states** (typed enums with `String()`; the Go constant is the type name followed by the value in PascalCase, for example `SightRelationCanSee`; zero value invalid): `Sense` = sight | hearing; `CheckRequirement` = sight | hearing | none; `SightRelation` = canSee | cannotSee; `LineOfSight` = clear | blocked; `EffectDamage` = damaging | notDamaging; `ConditionPermission` = allowed | forbidden; `SavingThrowEffect` = automaticFailure | disadvantage | unaffected; `AbilityCheckEffect` = automaticFailure | unaffected; `AutomaticCritical` = automatic | notAutomatic (the value `attacks.md` takes, shared-types-and-ownership.md FR-026); `SpeedIncrease` = allowed | forbidden; `ConditionConcentration` = intact | broken; `Speech` = able | unable; `ConditionActionAvailability` = available | none; `HeldItems` = kept | dropped; `Awareness` = aware | unaware; `AllDamageResistance` = none | all. `ConditionActionAvailability` and `ConditionPermission` are also read by `grappling.md`.
- **Query values:** `RollContributions` (a copied slice of `(ConditionKind, modifier.RollCircumstance)` and `Mode()` from `modifier.Resolve`); `EffectiveSpeed{Value, Increase}`; closed `ProneMovement` = `CrawlOnly` | `CrawlOrRightItself(costFeet)`; `ConditionFlags{Actions, Concentration, Speech, HeldItems, Awareness, Resistance}`; `ConditionAttackFacts{AttackerID, TargetID, DistanceFeet, AttackerSeesTarget, TargetSeesAttacker, AttackerFearSource}` with the two see-fields `SightRelation` and `attackerFearSource` `LineOfSight` (read only when the attacker is Frightened).
- **Signatures.** `ConditionEffects` is an empty struct with value-receiver methods (zero value usable, no state), so a downstream package can declare a small consumer-side interface over the queries it uses and a test can substitute it. A query that takes a distance or a Speed returns `(T, error)`: a negative input is an error wrapping `srderr.ErrInvalidInput` (FR-110). Abilities are `ability.Ability`.

  | Member | Signature | Requirements |
  |---|---|---|
  | `ConditionSet.Apply` | `func (s ConditionSet) Apply(target CombatantID, c Condition) ApplyConditionToSetResult` | FR-016 to FR-025, FR-123, FR-125 |
  | `ConditionSet.Remove` | `func (s ConditionSet) Remove(target CombatantID, kind ConditionKind) RemoveConditionFromSetResult` | FR-031 to FR-036, FR-125 |
  | `ConditionSet.ReduceExhaustion` | `func (s ConditionSet) ReduceExhaustion(target CombatantID, levels int) (ReduceExhaustionOnSetResult, error)` | FR-043 to FR-047, FR-125 |
  | `AttackRoll` | `func (ConditionEffects) AttackRoll(attacker, target ConditionSet, facts ConditionAttackFacts) (RollContributions, error)` | FR-050, FR-051, FR-057, FR-058, FR-061, FR-062, FR-070 to FR-072, FR-076, FR-080, FR-084, FR-088 to FR-090, FR-092, FR-093, FR-097, FR-101, FR-105 |
  | `CriticalHit` | `func (ConditionEffects) CriticalHit(target ConditionSet, attackerDistanceFeet int) (AutomaticCritical, error)` | FR-077, FR-078, FR-083, FR-098, FR-103 |
  | `AbilityCheck` | `func (ConditionEffects) AbilityCheck(self ConditionSet, fearSource LineOfSight) RollContributions` | FR-057, FR-058, FR-085 |
  | `AbilityCheckEffect` | `func (ConditionEffects) AbilityCheckEffect(self ConditionSet, requires CheckRequirement) AbilityCheckEffect` | FR-048, FR-049, FR-055, FR-056 |
  | `SocialCheck` | `func (ConditionEffects) SocialCheck(charmed ConditionSet, interactingCreature CombatantID) RollContributions` (Advantage only for the charmer) | FR-054 |
  | `SavingThrow` | `func (ConditionEffects) SavingThrow(self ConditionSet, a ability.Ability) SavingThrowEffect` | FR-074, FR-075, FR-081, FR-094 to FR-096, FR-102, FR-106 |
  | `Initiative` | `func (ConditionEffects) Initiative(self ConditionSet) RollContributions` | FR-067, FR-068 |
  | `Speed` | `func (ConditionEffects) Speed(self ConditionSet, baseSpeedFeet int) (EffectiveSpeed, error)` | FR-042, FR-060, FR-073, FR-079, FR-091, FR-100, FR-107, FR-128 |
  | `StandUp` | `func (ConditionEffects) StandUp(self ConditionSet, baseSpeedFeet int) (ProneMovement, error)` (uses `Speed`) | FR-086, FR-087 |
  | `CanTarget` | `func (ConditionEffects) CanTarget(actor ConditionSet, target CombatantID, damage EffectDamage) ConditionPermission` | FR-052, FR-053 |
  | `IsValidTarget` | `func (ConditionEffects) IsValidTarget(target ConditionSet, effectCreatorSeesTarget SightRelation) ConditionPermission` | FR-069, FR-126 |
  | `CanWillinglyMoveCloserTo` | `func (ConditionEffects) CanWillinglyMoveCloserTo(self ConditionSet, other CombatantID) ConditionPermission` | FR-059 |
  | `D20TestReduction` | `func (ConditionEffects) D20TestReduction(self ConditionSet) int` | FR-041 |
  | `IsIncapacitated` | `func (ConditionEffects) IsIncapacitated(self ConditionSet) bool` | FR-064 |
  | `IsDeadFromExhaustion` | `func (ConditionEffects) IsDeadFromExhaustion(self ConditionSet) bool` | FR-040 |
  | `ActiveFlags` | `func (ConditionEffects) ActiveFlags(self ConditionSet) ConditionFlags` | FR-064 to FR-066, FR-082, FR-099, FR-104, FR-109 |
  | `NewConditionSet` | `func NewConditionSet(explicit []Condition, immunities []ConditionKind) (ConditionSet, error)`; `MustConditionSet` panics with the same error | FR-011 to FR-015 |
  | `NewExhaustionLevel` | `func NewExhaustionLevel(n int) (ExhaustionLevel, error)`; `MustExhaustionLevel` panics with the same error | FR-008 to FR-010 |

- **Invariants:** `ExhaustionLevel` in 1 to 6; one explicit condition per kind; the effective set is derived, never stored, while the explicit set may hold kinds also implied or suppressed (FR-024, FR-029, FR-034); the set is immutable (FR-014).
- **Failures:** invalid input (level out of range, duplicate kinds, negative distance or Speed, `levels` below 1) returns an error wrapping `srderr.ErrInvalidInput` from `NewX` / the query (FR-009, FR-010, FR-012, FR-045, FR-110), and the `MustX` helpers panic with that error; `levels` is checked in the domain method, not the command. Expected outcomes are closed results, never errors. Untrusted condition names are rejected by the content decoder. No `PortError`: nothing here calls a port.

## 6. Interface

- **Use cases** (`*_usecase.go` files of package `combat`), each taking a `context.Context` first (they emit spans) and a command with the `ConditionSet`, the target `CombatantID` and the input: `ApplyConditionToSet(ApplyConditionToSetCommand(set, target, condition))` (FR-116); `RemoveConditionFromSet(RemoveConditionFromSetCommand(set, target, kind))` and `ReduceExhaustionOnSet(ReduceExhaustionOnSetCommand(set, target, levels))` (FR-117). Each is built with `telemetry.Telemetry` through a `WithTelemetry` option (default `telemetry.Noop`, FR-118; shared-types-and-ownership.md FR-036), builds a `Telemetry` and runs through it (FR-119). They are host-facing entry points only; later use cases compose the domain operations, not these. None of these three calls another use case; each calls only the domain `ConditionSet` operation and `Telemetry` (shared-types-and-ownership.md FR-044; architecture-guardrails.md FR-097). The `ToSet`/`FromSet`/`OnSet` names leave `ApplyCondition` and `RemoveCondition` to the encounter-level spec (D-12).
- **Ports:** none. Content-defined conditions or monster immunities arrive in the command as a `ConditionSet` built by the caller.
- **Queries:** `ConditionEffects` is domain code called by later `combat` specs (in practice `combat-action-resolution.md`, FR-131); no use case, no code dependency on `grappling.md`.
- **Exports:** package `combat` exports `Condition` and variants, `ExhaustionLevel`, `ConditionSet`, the result and event types, `ConditionEvent` and its members, `ConditionEffects`, `RollContributions`, the named-state enums and query values of section 5, and the three use cases and commands. It does not export `CombatantID` or `Event` (`combat-foundations.md` exports them), nor `ConditionKind` (package `content`). `Signals`, `Telemetry`, `Operation` and `ResultClass` are combat-foundations.md's helpers, used here as they are defined there. There is no facade or re-export package. Every exported identifier has a doc comment starting with its name.
- **Dependencies:** `modifier` (`RollCircumstance`, `RollMode`, `Resolve`), `ability` (`Ability`), `srderr` (`ErrInvalidInput`), `telemetry` (use-case files only); package `content` (`ConditionKind` only); `combat-foundations.md` (same context, to its left). Nothing from `character`, `hit-points-and-damage.md` or any later `combat` spec (FR-131).

## 7. Observability

The domain is **silent by design** (`doc/rules/observability.md` §3). The three use cases record the shared `combat` signals through `Telemetry` (span, operations counter, duration histogram, events counter; owned by `combat-foundations.md` FR-027 to FR-031, constants in `Signals`) and one signal of their own. Operations, result names and classes: FR-119, FR-120 and the "Result classification" table. Event kinds: `condition_applied`, `condition_removed`, `condition_level_reduced`.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.condition.active` | histogram, `{condition}` | `srd5e.combat.operation`, `srd5e.combat.condition.kind` | Effective conditions on the creature after the call | Dashboard the population p99; Incapacitated and Prone are implied and Petrified suppresses Poisoned, so a p99 above about 6 suggests a host applying conditions without ending them (adjust to the host's baseline). No per-creature alert: ids are not in telemetry |

Evaluation answers (`doc/rules/observability.md` §2):

1. **What can go wrong?** Invalid input (`levels` below 1), a bug; applying a condition the creature has or is immune to, or removing one not there, which are normal `refused` outcomes that may reveal a host out of sync; a hazard that keeps adding Exhaustion at level 6. No ports, so no collaborator failure.
2. **How would an operator know?** The returned validation error is recorded on the span; out-of-sync callers show as `srd5e.combat.operations` with `result_class` = `refused`.
3. **What would someone need to diagnose it?** Operation, result name and class, event kinds as attributes; the result carries the set, events and implying kinds, so no creature ids in telemetry (combat-foundations.md FR-033).
4. **What should be counted or timed?** Shared calls, duration and events, plus `srd5e.combat.condition.active`.
5. **Is existing instrumentation still correct?** The earlier own spans, counter and outcome attribute are replaced by the shared signals (combat-foundations.md section 5). `srd5e.outcome` is `success` or `failure` only (shared-types-and-ownership.md FR-041). Grapple signals are `grappling.md`'s.
6. **What to monitor or alert on?** The `refused` share per operation; a shift towards `removal_not_present`, `implied_by` or `already_held` means host-side state drift; investigate any `at_maximum`. No error-rate alert: rule outcomes are not errors.
7. **Is anything noisy?** No logs: the span and counter carry the outcome (`doc/rules/observability.md` §7). Attributes are closed sets without ids.

This spec adds to `Signals`: `conditionActiveMetric` (`srd5e.combat.condition.active`), `conditionKindAttribute` (`srd5e.combat.condition.kind`), the eleven result `...Value` constants of the "Result classification" table (combat-foundations.md FR-030), and the `...OperationValue` and `...Span` pair for each of the three operations (FR-040).

Four rows feed `doc/observability/signals.md`, as combat-foundations.md FR-040 makes this spec the owner of its span rows: `srd5e.combat.condition.active` and the spans `srd5e.combat.apply_condition_to_set`, `srd5e.combat.remove_condition_from_set` and `srd5e.combat.reduce_exhaustion_on_set`. The shared metric rows stay combat-foundations.md's (FR-037).

## 8. Test plan

Levels: unit (`conditions_test.go` and siblings, `package combat_test`), use case (`condition_usecase_test.go`), scenario (`scenarios/`), architecture (`internal/archtest`). No contract suite (no port). Every row of 4.11 is one test, named with its label and citing its requirement and path.

| Id | Level | Notes |
|---|---|---|
| FR-002 to FR-004, FR-007 to FR-015 | unit | Exhaustive `switch` over `Condition`; sources carried by the three variants only; equality; every kind has a definition matching 4.7 (one assertion per effect variant); levels 1 to 6 accepted and 0, -1, 7, 100 return errors (`MustExhaustionLevel` panics); empty set, duplicate kinds, copy-in and copy-out, order-insensitive `Equal` |
| FR-016 to FR-025 | unit | Apply each of the fifteen kinds to an empty set; already held; second charmer; input set unchanged; immunity per kind and unrelated immunity; Petrified blocks Poisoned, pre-existing Poisoned suppressed then resumed; Exhaustion immunity |
| FR-026 to FR-036 | unit | Implied table; applying implied kinds; Incapacitated once; Unconscious removal leaves Prone and keeps Incapacitated with Paralyzed; removal outcomes; input set unchanged; `Remove(Exhaustion)` lowers by one and ends at level 1 |
| FR-037 to FR-047 | unit | Exhaustion by level and reducing tables |
| FR-048 to FR-104, FR-126 | unit | Effects rows per condition in the attack, ability check, saving throw, Speed and Prone tables; Charmer queries with two distinct ids; `IsValidTarget` with `canSee`/`cannotSee`; distances 0, 5, 6; `ActiveFlags` and `IsIncapacitated` (`ConditionActionAvailabilityNone`, `ConditionConcentrationBroken`, `SpeechUnable`, `HeldItemsDropped`, `AwarenessUnaware`, `AllDamageResistanceAll`); `Initiative` (Incapacitated Disadvantage, Invisible Advantage); `CanWillinglyMoveCloserTo` for the source and for another creature; `CanTarget` with `EffectDamageDamaging` and `notDamaging` for the charmer and for another creature; `CrawlOrRightItself{15}`, `{12}`, `{0}` and `CrawlOnly` by exhaustive type `switch` |
| FR-105 to FR-112, FR-123 to FR-125, FR-128 | unit | Combination, pipeline and outcome-precedence tables; saving throw and Speed precedence; ordering independent of insertion order; empty-set neutrality; negative inputs; immutability; repeated calls equal; `SpeedIncreaseForbidden` for the five Speed-0 conditions, `allowed` for Stunned and none; `Apply(Exhaustion(3))` on empty and level-2 sets |
| FR-127 | unit | Unconscious removed: the set holds Prone, the events only `ConditionRemoved(Unconscious)`; a replay of events alone does not show Prone |
| FR-113 to FR-115, FR-129, FR-139 | unit, architecture | Event equality and fields; empty events for no-change outcomes; copied slices; the events exist once in the module; `ConditionEvent` closed, extends `Event`, exactly three members, exhaustive `switch`; `kind` uniqueness and pattern; `combatants` checks through the shared table helper of combat-foundations.md section 8 |
| FR-116 to FR-118 | application | Real domain; recording (`telemetrytest`) and zero-value `telemetry.Telemetry` give the same results |
| FR-119 to FR-122 | application | Span name per operation; the classification table, one row per variant; `events` once per event kind; `condition.active` with both attributes (`exhaustion` for reduce); on `ReduceExhaustion(0)` the error is returned, the span records it once, the duration sample exists and `operations`, `events`, `errors`, `active` do not; attribute values in the closed sets and none a `CombatantID`; no `srd5e.combat.condition.apply`, `.operations`, `.duration` or `.outcome` name in `Signals` |
| FR-016, FR-029, FR-038, FR-077, FR-105 | scenario | Through the public API only (`package scenarios_test`): grappled, knocked Unconscious, attacked adjacent (Critical Hit) and from range, wakes Prone; a second scenario stacks Exhaustion to level 6 |
| FR-130, FR-131, FR-140 | architecture | `ConditionKind` declaration order equals the Glossary order and the effective ordering follows it; no `condition*.go` file imports `attacks`, `initiative`, `encounter`, `combat-action-resolution` or `hit-points` code or another context's internals; no condition domain file references a use-case type; the three result types are declared in `condition_result.go` |
| FR-141, FR-142 | unit | `CharmedByCombatant` and `CharmedByItem` equality and blank item id; `CanTarget` and `SocialCheck` for an item charmer (no restriction, no Advantage); `Explicit` ordered, one per kind, a copy (mutating it leaves the set unchanged) |
| (architecture) | architecture | Existing tests cover the new files: no domain-file imports of use-case code or ports, no clock or `math/rand`, no `telemetry` calls in domain files, signal constants match `doc/observability/signals.md` |

- **Doubles:** no new port. Uses the recording doubles from `telemetry/telemetrytest` (shared-types-and-ownership.md FR-038) and the zero-value `telemetry.Noop` (FR-036). Domain tests need no double.
- **Mutation testing:** the condition files of package `combat` MUST be mutation tested with `gomu` (`doc/rules/testing.md` §8; Docker only, `--cpus` and `--workers` capped at a quarter of the host's processors), score at least 80; likely survivors: the 5 versus 6 foot distance boundary, the Exhaustion level 5 versus 6 boundary, the Exhaustion arithmetic, implication and precedence. The use cases SHOULD be.
- **Not covered:** none; withdrawn ids are covered by their new owners. "Does not apply" effects are covered by the boundary rows (FR-049, FR-053, FR-056, FR-062, FR-075, FR-083, FR-095, FR-098).

## 9. Open questions and decisions

No question is open.

Settled (one line each):

- D-2: the source of fear is a `CombatantID` (FR-003). The SRD (p. 182) says only "the source of fear" in "line of sight", but its sources are creatures or effects a host attributes to one; a host with a non-creature source assigns it an id or states the line-of-sight fact itself. No closed `FearSource` is added: no SRD-stated case needs the extra public API (design basis: `doc/rules/architecture.md` §3, minimal surface).
- D-3: the condition caps at 6 (`ExhaustionAtMaximum`, FR-039) and exposes `IsDeadFromExhaustion` (FR-040); `encounter-aggregate.md` FR-037 records the death; a package design decision, cited as such by FR-010 and FR-039.
- D-5: Immunity suppresses an existing condition rather than deleting it (FR-024, FR-124); the SRD (pp. 17, 183) does not say whether the entry is kept.
- D-7: no `ConditionApplied` when Unconscious ends and Prone persists (FR-029, FR-127).
- D-8: effective Speed is floored at 0 (FR-107); the SRD (p. 181) states no floor.
- D-9: a second source of the same condition is not kept (FR-019).
- D-6: grapple end and release are grappling.md's; this spec keeps the Grappled entry (FR-060 to FR-062) and plain remove (FR-031) (shared-types-and-ownership.md FR-029, FR-048).
- D-12: encounter-level `ApplyCondition`/`RemoveCondition` and the wrapping events are in `encounter-aggregate.md`, the roster events in `combatant-roster-changes.md`; this spec changes no behaviour; there is one `CombatantID` and one event set (FR-129; shared-types-and-ownership.md FR-019, FR-021).
- D-13: "this benefit" in Invisible removes both the Disadvantage on attacks against it and its own Advantage (FR-072); the SRD sentence is ambiguous and this is the broader reading.
- D-14: `Speed` takes one Speed and is applied to each named Speed independently (FR-107).
- D-15: `Remove(Exhaustion)` lowers by one, matching `ReduceExhaustion(1)` (FR-036); no SRD text ends the whole condition.
- D-16: `Apply(Exhaustion(level))` ignores the level and adds one (FR-123).
- D-17: precedence is immune, already held, at maximum; for `Remove` explicit, then implied, then absent (FR-125); the SRD gives none.
- D-20: the domain operations return the use cases' result types, declared in `condition_result.go` (FR-140); variant names are condition-prefixed to avoid name collisions in package `combat` (shared-types-and-ownership.md FR-017).
- D-21: a sentient item is not a combatant, so the charmer is a closed `Charmer` (FR-141); an item charmer restricts no combatant, since the SRD (p. 208) gives only the item's commands, which the host enforces. Interpretation: the Can't Harm and Social Advantage text names a creature charmer.
- Ownership questions (`CombatantID`, `ConditionKind`, grapple mechanics, roll merging, name parsing) are settled by shared-types-and-ownership.md FR-005, FR-012, FR-026, FR-029, FR-046, FR-048.

## 10. Changelog impact

Expected `CHANGELOG.md` entry under `Unreleased`, "Added":

- Conditions in `combat`: the fifteen SRD 5.2.1 conditions as a closed `Condition` type, `ConditionSet` with stacking, implied conditions and immunity, Exhaustion levels, the `ConditionEffects` queries, the closed `ConditionEvent` family (`ConditionApplied`, `ConditionRemoved`, `ConditionLevelReduced`), the `ApplyConditionToSet`, `RemoveConditionFromSet` and `ReduceExhaustionOnSet` use cases, and the shared `combat` telemetry plus the signal `srd5e.combat.condition.active`.

Not **BREAKING**: additions to a package that has not shipped them. Signal names become public contract once released (renaming one later is **BREAKING**, `doc/rules/architecture.md` §7).
