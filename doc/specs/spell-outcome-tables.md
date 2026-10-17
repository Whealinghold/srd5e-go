# Spell outcome tables

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `observability-ports.md`, `d20-tests.md`, `resolve-d20-test-use-case.md`, `content-schema-and-decoder.md`, `spell-effect-data.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md`, `spell-effects.md` |

Dependency notes (prose, not dependencies): rank 28 in table 4.11 of `shared-types-and-ownership.md` (after `spell-effects`, beside `encounter-aggregate`; an amendment adds the row, section 9). `magical-effects.md` Teleportation (p. 190, placement of an arrival) and `spellcasting.md` (the cast, the spell save DC) are used by the host around this spec, not imported.

## 1. Purpose and scope

Six spells print a table whose result is chosen by a die and changes what happens next: **Teleport** (a d100 on a familiarity row, then Mishap damage and rerolls, or an Off Target distance and direction; p. 168), **Confusion** (1d10 behaviour each turn; p. 117), **Prismatic Spray** (1d8 ray per target; p. 154), **Prismatic Wall** (seven layers crossed in order and destroyed in order; p. 155), **Reincarnate** (1d10 species, a 1 rolls again; p. 158) and **Blink** (1d6 at the end of each turn; pp. 113-114). `spell-effect-data.md` keeps such spells as text ("mode choice" and branching outcomes stay text, its table E) and `spell-effects.md` returns `SpellEffectsTextOnly` or `partial` for them, so nothing in the package rolls these tables.

The spec is small pure tables plus **one use case**, `ResolveSpellOutcomeTable`, that rolls through the `dice.Roller` and returns what the table says. Everything that depends on a result is a plain value (a `Damage`, a `ConditionKind`, a direction, a distance, a `SpeciesID`) for the host to apply with the existing use cases.

**Out of scope:**

- Casting, slots, Concentration (`casting-and-concentration.md`), choosing targets, range, line of sight, Wall shape and placement, the Wall's light and its initial Constitution save (p. 155), and the Ethereal Plane (what may affect what): the host.
- Knowing the **familiarity** of a destination ("carefully studied", "seen more than once", a "false destination" are Game Master facts, p. 168), the six-month limit on a Linked object, that the destination is on the same plane, and the object limits of Teleport (Large or smaller, not held by an unwilling creature): the host supplies the facts.
- **Placing** an arrival (nearest unoccupied space, sight): `magical-effects.md` Teleportation (p. 190). Where the Similar Area's "closest similar place" is: the host.
- Recurring saves and durations (Confusion's save at the end of each turn, Blink's 1 minute, Concentration): an active-effects model (`spell-effects.md` Q11). This spec resolves one turn's roll and, for Prismatic Spray, hands the host pure follow-up rules (FR-071 to FR-073).
- Changing the creature when it is reincarnated (loses the old species' traits, gains the new one's, p. 158): character state; flagged in section 9 as a future `character` spec.
- Teleportation Circle (p. 168, no table), the Teleportation Glossary entry (p. 190), Sphere and other spells with tables not listed in section 1: none found with the same shape, flagged below if met later.
- The Prismatic Layers' other printed Additional Effects (red blocks nonmagical ranged attacks, orange magical ranged attacks, indigo spells cast through it) and the wall's AC 10 (p. 155): host facts, not modelled (table W covers only how a layer is destroyed).
- Anything outside SRD 5.2.1. Nothing core was found missing from the PDF.

## 2. Source

SRD 5.2.1, printed page equals PDF page (checked on pp. 113-114, 117, 154-155, 158, 168).

- "Teleport", "Teleportation Outcome" table and its "Familiarity", "Mishap", "Similar Area", "Off Target", "On Target" paragraphs, p. 168.
- "Confusion" and its "1d10 Behavior for the Turn" table, p. 117.
- "Prismatic Spray" and "Prismatic Rays", pp. 154-155; "Prismatic Wall" and "Prismatic Layers", p. 155.
- "Reincarnate" and its species table, p. 158.
- "Blink", pp. 113-114.
- "Damage Rolls", "Damage against Multiple Targets", "Half Damage", p. 16; "Saving Throws" (spell), p. 106; "Damage Types", p. 180; "Condition", p. 179.
- Package rules: `doc/rules/architecture.md` §2 to §6; `doc/rules/observability.md` §2 to §7; `doc/rules/testing.md` §4, §5, §8. Binding decisions: `shared-types-and-ownership.md` FR-005, FR-007, FR-013, FR-017, FR-019, FR-044, FR-048, FR-062.
- Research (informs readings only, nothing copied): community consensus rolls Mishap damage per creature and applies a repeated Prismatic ray twice; recorded as interpretations in section 9.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Teleportation Outcome table, Familiarity | `TeleportFamiliarity` (`PermanentCircle`, `LinkedObject`, `VeryFamiliar`, `SeenCasually`, `ViewedOnceOrDescribed`, `FalseDestination`) | Wording of p. 168. A host fact (FR-018). |
| Mishap, Similar Area, Off Target, On Target | `TeleportOutcomeKind` | One value per table column. |
| (the arrival the table gives) | closed variant set `TeleportArrival`: `ArrivesOnTarget`, `ArrivesInSimilarArea`, `ArrivesOffTarget(miles, direction)` | Package term. |
| (who teleports) | closed variant set `TeleportSubject`: `TeleportingCreatures([]CombatantID)`, `TeleportingObject` | "you and up to eight willing creatures ... or a single object", p. 168. |
| Mishap damage | `TeleportMishap([]Damage)` | One `Damage` per teleporting creature, in order; exactly one for an object. |
| east, southeast ... northeast | `CompassDirection` (eight values in the order of the d8) | Confusion's north, east, south, west are four of them. |
| Behavior for the Turn | `ConfusionRow`, closed variant set `ConfusionBehavior` | FR-030 to FR-036. |
| Prismatic Rays, Prismatic Layers | `PrismaticColor` (`red` to `violet`, `order` 1 to 7) | The ray number equals the layer order (pp. 154-155). |
| Failed Save / Successful Save | `PrismaticStrike`, `PrismaticDamageStrike`, `PrismaticConditionStrike` | FR-064. |
| successes and failures "needn't be consecutive" | `IndigoSaveTrack`, `PrismaticFollowUp` | FR-066, FR-071. |
| the seven layers, destroyed in order | `PrismaticWallState`, `PrismaticLayerDestroyer`, `PrismaticLayerDestruction` | FR-075 to FR-080. |
| species table | `ReincarnationSpeciesFor` over `SpeciesID` | `SpeciesID` and `CreatureType` are owned by `content-schema-and-decoder.md` (FR-105 and FR-114). |
| Ethereal Plane | `BlinkLocation` (`BlinkOnOtherPlane`, `BlinkOnEtherealPlane`) | FR-040 to FR-044. |
| a table with banded rows | `OutcomeTable[T]`, `outcomeBand[T]` | Internal (not exported). FR-001. |

Terms use SRD wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Ids are allocated in blocks; an id absent from this section and not in the Withdrawn ids was never used by this spec. Case-table labels are not ids. Withdrawn ids: none.

### 4.1 Tables and rolling conventions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `OutcomeTable[T]` (built by `newOutcomeTable(die, bands)`) MUST hold `outcomeBand[T]{low, high, value}` values that start at 1, are contiguous (each `low` is the previous `high` plus 1), do not overlap and end at `die.Sides()`; any gap, overlap, wrong start or wrong end MUST be reported by the unexported `OutcomeTable[T].validate() error` (an error naming `bands`), which a test calls for every table; `newOutcomeTable(die, bands)` is a plain constructor with no `panic` and no runtime validation, because every table is fixed program data built by a function (FR-003), so a defect is a programmer error caught by the first test run (it never sees untrusted input). | Package design (a printed table covers every face exactly once); pp. 117, 158, 168 | unhappy (invalid input) |
| FR-002 | `OutcomeTable[T].lookup(face)` MUST return the band's value at both ends of every band and MUST return an error that is `errors.Is` `ErrFaceOutOfRange` (which wraps `srderr.ErrInvalidInput`) for 0, a negative face and `sides + 1`. | Package design | unhappy (boundary) |
| FR-003 | Every printed table MUST be built by an unexported function that returns a fresh `OutcomeTable[T]` on each call (for example `teleportTable()`), or be expressed as a `switch` over the face; there MUST be no package-level variable, no `init()`, no lazy build and no `sync` (`doc/rules/go.md` §5 and §10), so no state is shared or mutable by callers, covering exactly the faces of its table (tables T, C, R, P, B below). The percentile table uses `dice.D100()`; the printed "00" is face 100 (`01–00` is 1 to 100). | SRD 5.2.1, "Teleport", p. 168; `doc/rules/architecture.md` §2 to §6 (pure domain data), by analogy with `shared-types-and-ownership.md` FR-045 | happy |
| FR-004 | Where a table says to roll again (Reincarnate 1; Prismatic Spray 8; a Teleport Mishap), the reroll MUST be a new request of the same expression through `dice.Roll` (which rejects a result for another expression as `srderr.ErrContractViolation`). A "reroll" is each further request after the first of the same expression (Reincarnate: each 1 rolled; Prismatic Spray: each 8 inside the two-ray rerolls; Teleport: each Mishap's new d100). The 101st reroll request (the 102nd face) MUST fail with `srderr.ContractViolation(dice.Port, "roll", ...)` (an error `errors.Is` `srderr.ErrContractViolation`, never a panic and never a partial result); exactly `maxRerolls` (100) rerolls are allowed (a stuck roller is a defect of the collaborator, so the use case reports it as a port failure; with a fair roller the chance is below 2 to the power of -100 on the worst row). | SRD 5.2.1, pp. 154, 158, 168; package design (Q-3) | unhappy (boundary) |
| FR-005 | The pure tables and rules MUST roll nothing, log nothing and emit nothing; only the use case requests dice. A row whose outcome is certain MUST request no die (FR-011). | `doc/rules/observability.md` §3; package design | happy |
| FR-006 | For a fixed `dice.Roller` sequence the request order MUST be the one stated in each section (FR-015, FR-033, FR-062, FR-078) and a rejection (FR-090) MUST request no die. | `doc/rules/testing.md` §4 (determinism, state errors) | unhappy (determinism) |

### 4.2 Teleport

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | `TeleportKindFor(familiarity, face) (TeleportOutcomeKind, error)` MUST give the column of table T for the row and the 1d100 face. | SRD 5.2.1, "Teleportation Outcome", p. 168 | happy |
| FR-011 | `PermanentCircle` and `LinkedObject` MUST give `TeleportOnTarget` for every face (`01–00`) and `TeleportNeedsRoll(familiarity)` MUST be false for them: the use case MUST request no d100 for these two rows. Rolling would only move the host's roller sequence (as `spell-effects.md` FR-020 treats an empty area; Q-2). | SRD 5.2.1, "Teleportation Outcome", p. 168 | unhappy (rule interaction) |
| FR-012 | `FalseDestination` MUST give only Mishap (1 to 50) and Similar Area (51 to 100); Off Target and On Target MUST be unreachable for it (the printed "—"). | SRD 5.2.1, "Teleportation Outcome", p. 168 | unhappy (boundary) |
| FR-013 | A Mishap MUST deal `3d10` Force damage to **each** teleporting creature, rolled separately for each (an object: one roll). Each is a `Damage(total, DamageType.force)` in the order of the subject. | SRD 5.2.1, "Mishap" ("Each teleporting creature (or the target object) takes 3d10 Force damage"), p. 168; interpretation (Q-1) | happy |
| FR-014 | After a Mishap the use case MUST roll the d100 again on the **same** familiarity row ("rerolls on the table to see where you wind up"), and a further Mishap MUST deal its damage again; every mishap MUST be listed in order in `TeleportOutcome.Mishaps`. The arrival is that of the first non-Mishap roll. A `FalseDestination` therefore ends as Similar Area, since Mishap is its only other column. | SRD 5.2.1, "Mishap" ("multiple Mishaps can occur, dealing damage each time"), p. 168 | unhappy (rule interaction) |
| FR-015 | Request order for Teleport: the d100 (FR-011 permitting); on Mishap, the `3d10` of each creature in subject order, then the d100 again; on Off Target, `2d12` and then `1d8`. | SRD 5.2.1, p. 168; `doc/rules/testing.md` §4 | unhappy (determinism) |
| FR-016 | Off Target MUST give `ArrivesOffTarget(miles, direction)` with `miles` the total of `2d12` (2 to 24) and `direction` from the `1d8` by `CompassDirection` order: 1 east, 2 southeast, 3 south, 4 southwest, 5 west, 6 northwest, 7 north, 8 northeast. | SRD 5.2.1, "Off Target", p. 168 | happy |
| FR-017 | Similar Area MUST give `ArrivesInSimilarArea` and On Target `ArrivesOnTarget`; neither rolls a die. Which similar area is the host's. | SRD 5.2.1, "Similar Area", "On Target", p. 168 | happy |
| FR-018 | The command MUST carry the `TeleportFamiliarity` the host judged. `NewTeleportingCreatures([]CombatantID) (TeleportingCreatures, error)` MUST accept 1 to 9 distinct `CombatantID`s ("you and up to eight") and otherwise return an error that is `errors.Is` `srderr.ErrInvalidInput`; a repeated id is rejected the same way. | SRD 5.2.1, "Teleport", "Familiarity", p. 168 | unhappy (invalid input) |
| FR-019 | `TeleportOutcome` MUST carry the familiarity, the subject, the `Mishaps` (possibly empty) and the final arrival, as immutable values (slices copied on construction and on read) with value equality. It MUST NOT apply damage (the host passes each `Damage` to `ApplyDamage`) or move anyone (the host places the arrival with `magical-effects.md` Teleportation). | SRD 5.2.1, "Teleportation", p. 190; `doc/rules/architecture.md` §4 | unhappy (immutability) |

### Table T: Teleportation Outcome (FR-010 to FR-012, FR-014; one test per row and boundary)

Faces are 1d100; cells are the inclusive band. "On" is On Target.

| Label | Familiarity | Mishap | Similar Area | Off Target | On | Boundary faces tested |
|---|---|---|---|---|---|---|
| T1 | `PermanentCircle` | none | none | none | 1-100 | 1, 100 (no roll, FR-011) |
| T2 | `LinkedObject` | none | none | none | 1-100 | 1, 100 (no roll) |
| T3 | `VeryFamiliar` | 1-5 | 6-13 | 14-24 | 25-100 | 1, 5, 6, 13, 14, 24, 25, 100 |
| T4 | `SeenCasually` | 1-33 | 34-43 | 44-53 | 54-100 | 1, 33, 34, 43, 44, 53, 54, 100 |
| T5 | `ViewedOnceOrDescribed` | 1-43 | 44-53 | 54-73 | 74-100 | 1, 43, 44, 53, 54, 73, 74, 100 |
| T6 | `FalseDestination` | 1-50 | 51-100 | none | none | 1, 50, 51, 100 |

### 4.3 Confusion

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-030 | `ConfusionRowFor(face) (ConfusionRow, error)` MUST follow table C: 1 `ConfusionRowMovesRandomly`; 2 to 6 `ConfusionRowDoesNothing`; 7 and 8 `ConfusionRowAttacksRandomCreature`; 9 and 10 `ConfusionRowChoosesBehavior`. | SRD 5.2.1, "Confusion", p. 117 | happy |
| FR-031 | Row 1 MUST give `ConfusionMovesRandomly(direction)` with the direction from `1d4`: 1 north, 2 east, 3 south, 4 west; the target takes no action and uses all its movement. Only these four `CompassDirection` values are valid; another MUST be rejected by `NewConfusionMovesRandomly` with an error that is `errors.Is` `srderr.ErrInvalidInput`. | SRD 5.2.1, "Confusion" (row 1), p. 117 | happy |
| FR-032 | Rows 2 to 6 MUST give `ConfusionDoesNothing` (no movement, no actions); rows 9 and 10 MUST give `ConfusionTargetChoosesBehavior` (the host or player decides; nothing is rolled). | SRD 5.2.1, "Confusion" (rows 2-6, 9-10), p. 117 | happy |
| FR-033 | Rows 7 and 8: the command lists `CreaturesWithinReach` (the host's fact, in a fixed order, without the confused target). In both variants the target does not move, carried as data (`TargetMoves` false). With none the result MUST be `ConfusionNoCreatureInReach` (no action, no movement) and no die is requested; with one it MUST be `ConfusionAttacksRandomCreature(thatID)` with no die; with `n` of 2 or more it MUST request `1d<n>` (a `dice.NewDie(n)`) and the face picks the creature at that position. Request order: `1d10`, then the `1d4` or the `1d<n>`. | SRD 5.2.1, "Confusion" (rows 7-8, "The target doesn't move, and it takes the Attack action", "against a random creature within reach", "If none are within reach, the target takes no action"), p. 117; `dice-and-rolls.md` FR-002 | unhappy (boundary) |
| FR-034 | A list of more than `dice.MaxSides` (1000) creatures, or one with a repeated id, MUST be rejected with an error that is `errors.Is` `srderr.ErrInvalidInput`. | `dice-and-rolls.md` FR-004; package design | unhappy (invalid input) |
| FR-035 | Row 7 and 8's attack MUST be reported as a single melee attack (the Attack action, one attack): the behaviour value says `AttackCount` 1, and no Extra Attack or other attack source adds to it. | SRD 5.2.1, "Confusion" ("the Attack action to make one melee attack"), p. 117 | unhappy (rule interaction) |
| FR-036 | Every `ConfusionTurn` MUST report `CannotTakeBonusActionsOrReactions` true, a fact of the spell, not of the row, and MUST NOT roll the end-of-turn save, which is the host's (section 1). | SRD 5.2.1, "Confusion" ("can't take Bonus Actions or Reactions"), p. 117 | happy |

### Table C: Behavior for the Turn (FR-030 to FR-033)

| Label | 1d10 | Row | Extra dice | Result |
|---|---|---|---|---|
| C1 | 1 | `ConfusionRowMovesRandomly` | `1d4` (1 N, 2 E, 3 S, 4 W) | `ConfusionMovesRandomly(direction)` |
| C2 | 2 | `ConfusionRowDoesNothing` | none | `ConfusionDoesNothing` (lower edge of 2-6) |
| C3 | 6 | `ConfusionRowDoesNothing` | none | `ConfusionDoesNothing` (upper edge) |
| C4 | 7 | `ConfusionRowAttacksRandomCreature` | `1d<n>`, n at least 2 | attacks the creature at the face; `TargetMoves` false |
| C5 | 8 | `ConfusionRowAttacksRandomCreature` | none; one candidate | attacks it; `TargetMoves` false |
| C6 | 8 | `ConfusionRowAttacksRandomCreature` | none; no candidate | `ConfusionNoCreatureInReach`; no action, no movement |
| C7 | 9 | `ConfusionRowChoosesBehavior` | none | `ConfusionTargetChoosesBehavior` (lower edge of 9-10) |
| C8 | 10 | `ConfusionRowChoosesBehavior` | none | `ConfusionTargetChoosesBehavior` |

### 4.4 Blink

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | At the end of the caster's turn the use case MUST roll `1d6`. A 1 to 3 MUST give `BlinkStaysOnPlane`; a 4 to 6 with the caster `BlinkOnOtherPlane` MUST give `BlinkVanishesToEtherealPlane` (the new location is `BlinkOnEtherealPlane`). | SRD 5.2.1, "Blink" ("On a roll of 4–6, you vanish"), p. 113 | happy |
| FR-041 | A 4 to 6 with the caster already `BlinkOnEtherealPlane` MUST give `BlinkEndsAlreadyEthereal`: the spell ends instantly, and because the caster is on the Ethereal Plane when it ends the result carries `BlinkReturnFor(BlinkOnEtherealPlane)` (a `BlinkReturnsToOtherPlane`, FR-043) as data. | SRD 5.2.1, "Blink" ("the spell ends instantly if you are already on that plane"), p. 113; ("when the spell ends if you are on the Ethereal Plane"), p. 114 | unhappy (rule interaction) |
| FR-042 | A 1 to 3 MUST leave the location unchanged whichever it is, and `BlinkStaysOnPlane` MUST carry the unchanged location. | SRD 5.2.1, "Blink", p. 113 | unhappy (boundary) |
| FR-043 | `BlinkReturnFor(location)` (a pure function used at the start of the caster's next turn and when the spell ends) MUST give `BlinkReturnsToOtherPlane` for `BlinkOnEtherealPlane` and `BlinkNothingToReturn` for `BlinkOnOtherPlane`. The result states the placement rule as data (an unoccupied space of the caster's choice that they can see within 10 feet of the space they left, else the nearest unoccupied space) and chooses no space. | SRD 5.2.1, "Blink" ("You return to the other plane at the start of your next turn and when the spell ends"), p. 114 | happy |
| FR-044 | `BlinkVanishesOn(face) (bool, error)` MUST return true for 4, 5 and 6 and false for 1, 2 and 3; a face outside 1 to 6 MUST return an error that is `errors.Is` `ErrFaceOutOfRange`. | SRD 5.2.1, "Blink", p. 113 | unhappy (boundary) |

### 4.5 Reincarnate

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-050 | `ReincarnationSpeciesFor(face) (species SpeciesID, rollAgain bool, err error)` MUST follow table R: 1 is a roll-again marker; 2 `dragonborn`, 3 `dwarf`, 4 `elf`, 5 `gnome`, 6 `goliath`, 7 `halfling`, 8 `human`, 9 `orc`, 10 `tiefling`, each a `SpeciesID` (lowercase `snake_case`, `shared-types-and-ownership.md` FR-007). | SRD 5.2.1, "Reincarnate", p. 158 | happy |
| FR-051 | The use case MUST roll `1d10` and, on a 1, roll again until the face is not 1 (FR-004). The result MUST be `ReincarnationSpeciesRolled(species, rerolls)` with `rerolls` the number of 1s rolled first. | SRD 5.2.1, "Reincarnate" (1, "Roll again"), p. 158 | unhappy (rule interaction) |
| FR-052 | When the command carries a `ChosenSpecies`, the result MUST be `ReincarnationSpeciesChosen(species)` and no die requested: "or the GM chooses another playable species". The package does not check that the species is playable (a content lookup is the host's). | SRD 5.2.1, "Reincarnate", p. 158 | happy |
| FR-053 | The command MUST carry the dead creature's `CreatureType` and `DeadFor` (a `time.Duration`). A creature type other than Humanoid MUST be refused with `NotHumanoid`; a `DeadFor` greater than 10 days MUST be refused with `DeadTooLong`; exactly 10 days is accepted. Both are reported together in that order. A negative `DeadFor` MUST be rejected with an error that is `errors.Is` `srderr.ErrInvalidInput`. The check applies when a species is chosen too. | SRD 5.2.1, "Reincarnate" ("a dead Humanoid or a piece of one", "dead no longer than 10 days"), p. 158 | unhappy (boundary) |
| FR-054 | The result MUST say that the new species' traits replace the old ones as data (`ReplacesSpeciesTraits` true) and apply nothing: character change is the host's (section 1). The SpeciesIDs of table R MUST all exist in the bundled content set (a scenario test; `shared-types-and-ownership.md` FR-054). | SRD 5.2.1, "Reincarnate", p. 158 | unhappy (missing content) |

### Table R: Reincarnate species (FR-050, FR-051; one test per row)

| Label | 1d10 | Species | | Label | 1d10 | Species |
|---|---|---|---|---|---|---|
| R1 | 1 | roll again | | R6 | 6 | `goliath` |
| R2 | 2 | `dragonborn` | | R7 | 7 | `halfling` |
| R3 | 3 | `dwarf` | | R8 | 8 | `human` |
| R4 | 4 | `elf` | | R9 | 9 | `orc` |
| R5 | 5 | `gnome` | | R10 | 10 | `tiefling` |

### 4.6 Prismatic Spray

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-060 | `PrismaticRayFor(face) (PrismaticColor, error)` MUST follow table P for 1 to 7 (`red` to `violet`) and mark 8 as `special`. | SRD 5.2.1, "Prismatic Rays", pp. 154-155 | happy |
| FR-061 | For each target the use case MUST roll `1d8`. A 1 to 7 gives that one ray. An 8 gives **two** rays: it rolls `1d8` twice more, each of the two rerolled while it shows 8 (FR-004), so no target has more than two rays. The two rays MAY be the same colour (Q-4). | SRD 5.2.1, "Prismatic Rays" (8, "Roll twice, rerolling any 8"), p. 155 | unhappy (rule interaction) |
| FR-062 | Request order: every target's ray rolls in command order (each target's second ray after its first); then, only when at least one ray strikes with a damage colour (`red` to `blue`), one `12d6`; then each target's Dexterity save in command order. | SRD 5.2.1, "Damage against Multiple Targets", p. 16; `spell-effects.md` FR-028, FR-041 (damage before saves) | unhappy (determinism) |
| FR-063 | The damage MUST be rolled **once** for the cast and shared by every damage ray of every target, whatever its colour (the colours differ only in damage type): one effect that makes two or more targets save at once. A target struck by two damage rays takes that amount twice, once per ray's type (interpretation, Q-4). | SRD 5.2.1, "Damage against Multiple Targets" ("roll the damage once for all the targets"), p. 16; "Prismatic Spray", p. 154 | happy |
| FR-064 | Each target MUST make one Dexterity save against the spell save DC, resolved as `spell-effects.md` FR-024 resolves a `SavingThrowResolution` (the pure `d20.ResolveCheck`, the target's `SaveAttempt`, `ChooseToFail` and `AutomaticFailure` rolling no d20; never a use case, `shared-types-and-ownership.md` FR-044). The one save applies to every ray that strikes the target. | SRD 5.2.1, "Prismatic Spray" ("Each creature in the Cone makes a Dexterity saving throw"), p. 154 | happy |
| FR-065 | A damage ray (`red` Fire, `orange` Acid, `yellow` Lightning, `green` Poison, `blue` Cold) MUST give `PrismaticDamageStrike(Damage(total, type), Halved)` with `Halved` true on a successful save and false on a failed one. The amount MUST be the full roll: halving is `hit-points-and-damage.md` FR-099's `HalfDamage`, passed by the host. | SRD 5.2.1, "Prismatic Rays" (1-5), p. 154-155; "Half Damage", p. 16 | happy |
| FR-066 | `indigo` on a failed save MUST give `PrismaticConditionStrike{Restrained, FollowUp: IndigoFollowUp{IndigoSaveTrack{}}}`; `violet` on a failed save MUST give `PrismaticConditionStrike{Blinded, FollowUp: VioletFollowUpPending{}}`, both values of the closed `PrismaticFollowUp`. On a successful save neither gives anything. | SRD 5.2.1, "Prismatic Rays" (6-7), p. 155 | happy |
| FR-067 | An empty target list MUST be valid and return an empty result with no die requested (`spell-effects.md` FR-020). A `PrismaticSaveTarget` id duplicated in the list MUST be rejected with an error that is `errors.Is` `srderr.ErrInvalidInput`. | `spell-effects.md` FR-015, FR-020 | unhappy (empty and minimal) |
| FR-068 | The command MUST carry, for each target, its Dexterity `SaveAttempt` (a target is an id and attempt pair, so none can be missing) and the spell save DC; no refusal reason of FR-090 applies to Prismatic Spray. | `spell-effects.md` FR-059 (pattern) | unhappy (missing content) |

### Table P: Prismatic Rays and Layers (FR-060, FR-065, FR-066, FR-075)

For the Spray the number is the 1d8 face; for the Wall it is the layer order. Damage is `12d6`; "half" is the Halved flag on a successful save.

| Label | Number | Colour | Failed save | Successful save | Wall layer destroyed by (table W) |
|---|---|---|---|---|---|
| P1 | 1 | `red` | 12d6 Fire | half | at least 25 Cold damage |
| P2 | 2 | `orange` | 12d6 Acid | half | a strong wind |
| P3 | 3 | `yellow` | 12d6 Lightning | half | at least 60 Force damage |
| P4 | 4 | `green` | 12d6 Poison | half | Passwall or an equal or higher level portal spell |
| P5 | 5 | `blue` | 12d6 Cold | half | at least 25 Fire damage |
| P6 | 6 | `indigo` | Restrained, Constitution saves | nothing | Bright Light of Daylight |
| P7 | 7 | `violet` | Blinded, Wisdom save | nothing | Dispel Magic |
| P8 | 8 | special | two more rays (each rerolls an 8) | n/a (Spray only) | n/a |

### 4.7 Indigo, Violet and the Wall

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-071 | `IndigoSaveTrack{successes, failures}` MUST start at 0 and 0 (the zero value) and `Record(d20.Outcome)` MUST return `IndigoContinues(track)` with the count raised, `IndigoFreed` on the third success (Restrained ends) or `IndigoPetrified` on the third failure (Petrified until freed by an effect like Greater Restoration). Successes and failures need not be consecutive. A track with 3 or more of either (built by `NewIndigoSaveTrack(successes, failures) (IndigoSaveTrack, error)`) MUST be rejected with an error that is `errors.Is` `srderr.ErrInvalidInput`. | SRD 5.2.1, "Prismatic Rays" (6, Indigo), p. 155 | unhappy (boundary) |
| FR-072 | `ResolveVioletFollowUp(d20.Outcome)` (the Wisdom save at the start of the caster's next turn) MUST give `VioletBlindnessEnds` on success and `VioletBanished` on failure; in both the Blinded condition ends, and `VioletBanished` also reports a teleport to **another plane of existence** whose choice is the host's. | SRD 5.2.1, "Prismatic Rays" (7, Violet), p. 155 | happy |
| FR-073 | The Indigo Constitution saves (end of each of the target's turns) and the Violet Wisdom save are rolled by the host (the `d20` resolver); FR-071 and FR-072 take the outcomes. The Prismatic Spray and Wall use cases MUST NOT roll them (recurring saves are out of scope, section 1). | SRD 5.2.1, "Prismatic Rays", p. 155; `spell-effects.md` Q11 | unhappy (rule interaction) |
| FR-075 | `PrismaticWallState` MUST hold `destroyedCount` (0 to 7: layers are destroyed only from red onward, FR-076, so the present layers are always orders `destroyedCount + 1` to 7), start with `PrismaticWallState{}` (the zero value is the intact wall) (0), and be immutable. The host resumes a stored wall with `NewPrismaticWallStateAfter(count) (PrismaticWallState, error)`, which MUST return an error that is `errors.Is` `ErrWallDestroyedCount` (wrapping `srderr.ErrInvalidInput`) outside 0 to 7. `Destroy` returns a new state. | SRD 5.2.1, "Prismatic Wall" ("The wall consists of seven layers", "If a layer is destroyed, it is gone"), p. 155 | happy |
| FR-076 | `PrismaticWallState.Destroy(layer, destroyer)` MUST destroy a layer only when it is the **outermost layer still present** ("destroyed one layer at a time, in order from red to violet"); otherwise it MUST return `PrismaticLayerNotDestroyed(notOutermost)` or, for a layer already gone, `AlreadyDestroyed`. | SRD 5.2.1, "Prismatic Wall", p. 155 | unhappy (rule interaction) |
| FR-077 | The destroyer MUST match table W: red needs `DamageToLayer(Cold, amount)` of at least 25; yellow `DamageToLayer(Force, amount)` of at least 60; blue `DamageToLayer(Fire, amount)` of at least 25; orange `StrongWind`; green `PortalOpened(spellLevel)` with `spellLevel` of at least 5 (Passwall is level 5); indigo `DaylightBrightLight`; violet `DispelMagic`. A wrong kind MUST give `WrongDestroyer`; a `DamageToLayer` of the right type below the threshold MUST give `BelowThreshold`. The amount is one damage instance (the SRD gives a threshold, the wall no Hit Points; Q-5). | SRD 5.2.1, "Prismatic Layers" (Additional Effects), p. 155 | unhappy (boundary) |
| FR-078 | `ResolveSpellOutcomeTable` for a Wall crossing MUST resolve one creature through the layers still present in order from `red` to `violet`: for each layer, the creature's Dexterity save (FR-064's method, the same `SaveAttempt` and DC for every layer); then for a damage layer one `12d6` rolled for that layer whatever the save (half on a success, FR-065); `indigo` and `violet` give their strike only on a failed save (FR-066). Request order: layer by layer, the save then the damage. A layer's damage is rolled per layer and per crossing, not shared. | SRD 5.2.1, "Prismatic Wall" ("Each layer forces the creature to make a Dexterity saving throw"), p. 155; interpretation: the crossing order is the Order column 1-7 of the Prismatic Layers table, p. 155 (the SRD fixes red-to-violet order only for destruction, FR-076) | happy |
| FR-079 | A crossing of a wall with no layer left MUST be valid, give an empty result and request no die. Whether the creature keeps moving after a strike (Restrained at the indigo layer, say) is not stated and is not modelled: the result lists every layer (Q-6). | SRD 5.2.1, "Prismatic Wall", p. 155; package design | unhappy (empty and minimal) |
| FR-080 | Antimagic Field has no effect on the wall, and Dispel Magic affects only the violet layer: the destroyer set of FR-077 MUST have no value for either on another layer, and no Antimagic Field value MUST exist. A host that sees Dispel Magic on a lower layer gets `WrongDestroyer`. | SRD 5.2.1, "Prismatic Wall", p. 155 | unhappy (rule interaction) |

### Table W: destroying a layer (FR-076, FR-077; one test per row)

Present layers are all seven unless the row says otherwise; "outermost" means the lowest remaining order.

| Label | Layer | Destroyer | Layers present | Result |
|---|---|---|---|---|
| W1 | red | `DamageToLayer(Cold, 25)` | all | destroyed (boundary 25) |
| W2 | red | `DamageToLayer(Cold, 24)` | all | `BelowThreshold` |
| W3 | red | `DamageToLayer(Fire, 99)` | all | `WrongDestroyer` |
| W4 | orange | `StrongWind` | red gone | destroyed |
| W5 | orange | `StrongWind` | all | `NotOutermost` |
| W6 | yellow | `DamageToLayer(Force, 60)` / `59` | red, orange gone | destroyed / `BelowThreshold` |
| W7 | green | `PortalOpened(5)` / `PortalOpened(4)` | red to yellow gone | destroyed / `BelowThreshold` (modelled as a level below 5) |
| W8 | blue | `DamageToLayer(Fire, 25)` | red to green gone | destroyed |
| W9 | indigo | `DaylightBrightLight` | red to blue gone | destroyed |
| W10 | violet | `DispelMagic` | red to indigo gone | destroyed; wall empty (FR-079) |
| W11 | indigo | `DispelMagic` | red to blue gone | `WrongDestroyer` (FR-080) |
| W12 | red | `StrongWind` after red already destroyed | red gone | `AlreadyDestroyed` |

### 4.8 The use case, commands and results

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-090 | `ResolveSpellOutcomeTable.Resolve(ctx, cmd) (ResolveSpellOutcomeTableResult, error)` MUST take a `ResolveSpellOutcomeTableCommand` (an interface with an unexported marker method) with exactly the variants `RollTeleportOutcomeCommand`, `RollConfusionBehaviorCommand`, `RollPrismaticSprayCommand`, `ResolvePrismaticWallCrossingCommand`, `RollReincarnationSpeciesCommand` and `RollBlinkTurnEndCommand`, and return a `ResolveSpellOutcomeTableResult` (an interface with an unexported marker method, switched exhaustively) with exactly `SpellTableTeleportRolled`, `SpellTableConfusionRolled`, `SpellTablePrismaticSprayRolled`, `SpellTablePrismaticWallCrossed`, `SpellTableReincarnationRolled`, `SpellTableBlinkRolled` and `SpellTableRefused{Refusals []SpellTableRefusal}`. A refusal is `NotHumanoid` or `DeadTooLong` (FR-053), the only rule refusals; any other invalid input (a malformed command: FR-018, FR-034, FR-053, FR-067) returns a nil result and an error that is `errors.Is` `srderr.ErrInvalidInput`; a roller failure returns a `*srderr.PortError` (FR-091). The returned error is never a result variant. | `shared-types-and-ownership.md` FR-017; `doc/rules/architecture.md` §4, §6 | happy |
| FR-091 | A roller failure MUST be returned wrapped exactly once as a `*srderr.PortError` for `dice.Port` (`errors.Is` the original cause, `srderr.ErrorType` `dice_roller_error`), with a nil result and no partial result, never converted to a refusal; a reroll-limit failure (FR-004) is the same kind of error (`srderr.ErrContractViolation`). A panic in the roller is recovered by `dice.Roll` as `srderr.ErrPortPanic`; the use case itself adds no `recover`. | `dice-and-rolls.md`, `errors-and-results.md` FR-031; `doc/rules/architecture.md` §6 | unhappy (collaborator failure) |
| FR-092 | Commands and results MUST NOT be mutated by the call, every slice they expose MUST be a copy, and equal commands with equal roller sequences MUST give equal results (comparable with `==` or the type's `Equal` method). | `doc/rules/testing.md` §4 (immutability, determinism) | unhappy (immutability) |
| FR-093 | The use case MUST NOT call, hold or import any use case; its constructor `NewResolveSpellOutcomeTable(roller dice.Roller, opts ...Option) (ResolveSpellOutcomeTable, error)` takes the roller (a nil roller is rejected with an error `errors.Is` `srderr.ErrInvalidInput`) and the option `WithTelemetry(telemetry.Telemetry)` (default: silent no-ops; no global). It declares its own one-method consumer-side roller interface, as `d20.Roller` does, so it is not coupled to a `dice` implementation. It composes the pure `d20.ResolveCheck`, `ResolveAttack` and `FailWithoutRoll` and the pure tables only. | `shared-types-and-ownership.md` FR-044; `spell-effects.md` FR-096 | unhappy (rule interaction) |
| FR-094 | It produces no `combat.Event`: the host applies results with `ApplyDamage` and `ApplyConditionToSet` and receives their events. | `combat-foundations.md` FR-019 | happy |

### 4.9 Observability

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-100 | The use case MUST record through the package's `combatTelemetry` helper (built from the injected `telemetry.Telemetry`) with the operation `resolve_spell_outcome_table` (span `srd5e.combat.resolve_spell_outcome_table`, started with `Tracer.Start(ctx, ...)`) and the four shared `combat` signals as `combat-foundations.md` FR-029 to FR-032 define them; it defines no span, throughput counter, duration histogram or error counter of its own. Results: every `...Rolled` and `...Crossed` variant is `applied` (result `spell_table_rolled`); `SpellTableRefused` is `refused` (result `spell_table_refused`, no span error status). | `combat-foundations.md` FR-029 to FR-032, FR-035, FR-040 | happy |
| FR-101 | It MUST add the attribute `srd5e.combat.spell_table` (`teleport`, `confusion`, `prismatic_spray`, `prismatic_wall`, `reincarnate`, `blink`) through the `attributes` closure (`combat-foundations.md` FR-041), so it is on the span and the `operations` increment and nowhere else. The key and its six values MUST be listed in the Attributes cell of the shared `srd5e.combat.operations` row. | `combat-foundations.md` FR-037, FR-041 | happy |
| FR-102 | It MUST increment `srd5e.combat.spell_table_rolls` (`{roll}`) once per roll **on a printed table** (a table that is a printed list of faces; Blink's 1-3 and 4-6 split is prose, so its die band is counted as `stays` or `vanishes` and the location-dependent end is the result, not the table outcome), rerolls included, with `srd5e.combat.spell_table` and `srd5e.combat.table_outcome` (the closed set of table 7.1). A roll that is not a table lookup (the damage dice, the d20 saves, the `1d4` and `1d8` direction dice, a `dice.NewDie(n)` pick) MUST NOT be counted. A refused call and a call that returns an error record no increment. | `doc/rules/observability.md` §5 | happy |
| FR-103 | A port failure MUST be recorded once by the combat telemetry helper (`error.type` from `srderr.ErrorType(err)`) and not logged; the use case MUST NOT log any outcome. No attribute MAY carry a combatant id, `SpeciesID`, amount or distance. All names are exported constants in `combat/signals.go`. | `combat-foundations.md` FR-033, FR-034; `doc/rules/observability.md` §7 | unhappy (collaborator failure) |

### 4.10 Layering and exports

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-110 | The tables, rules and value objects MUST live in package `combat`, in files named `spell_outcome_*.go` (pure, silent, no `dice.Roller`); the use case, commands, results and refusal enum in `resolve_spell_outcome_table_usecase.go`, synchronous (the only port is the synchronous `dice.Roller`) with `ctx` first because it emits a span. No `spell_outcome_*.go` file MAY reference `dice.Roller`, `srderr` or `telemetry` (check A6). | `doc/rules/architecture.md` §1, §9; `dice-and-rolls.md` FR-045 | happy |
| FR-111 | It MUST import `content` and `d20` only through their published packages (`DamageType`, `ConditionKind`, `CreatureType`, `SpeciesID`, `d20.ResolveCheck`, `d20.Outcome`, `SaveAttempt`) and nothing from `character` and the same-context specs to its left only. It MUST NOT import `toolbox` or `magical-effects.md` files. | `combat-foundations.md` FR-026; `shared-types-and-ownership.md` FR-013 | unhappy (rule interaction) |
| FR-112 | Every exported name MUST be unique across the module's packages' public APIs, none a generic variant name, none referring to a consuming project. Everything of sections 3 and 6 except `OutcomeTable` and `OutcomeBand` is exported from package `combat`. | `shared-types-and-ownership.md` FR-017, FR-021; `doc/rules/architecture.md` §4 | unhappy (name collision) |
| FR-113 | No new port, test double or contract suite: tests use `dicetest.FixedRoller` and the recording `telemetrytest` doubles. | `doc/rules/architecture.md` §7 | happy |

## 5. Model

Notation: `Name(a, b)` lists the fields of a struct type `Name`; a "closed" type is an interface with an unexported marker method, a doc comment naming every variant, and one struct per variant, switched exhaustively (`errors-and-results.md` FR-018). Constructors are `NewX` returning `(T, error)` with errors satisfying `errors.Is(err, srderr.ErrInvalidInput)`; `MustX` panics with that error. Slices and maps are copied on construction and on read.

- **Tables and pure rules (package functions over unexported table-building functions, each over the internal `OutcomeTable[T]`):** `TeleportKindFor`, `TeleportNeedsRoll`, `ConfusionRowFor`, `ReincarnationSpeciesFor`, `PrismaticRayFor`, `BlinkVanishesOn`, `BlinkReturnFor`.
- **Value objects:** `TeleportSubject` (closed variant set), `TeleportOutcome`, `TeleportMishap`, `TeleportArrival` (closed variant set), `CompassDirection`, `ConfusionBehavior` (closed variant set) and `ConfusionTurn`, `BlinkLocation`, `BlinkTurnEndOutcome` (closed variant set), `BlinkReturn` (closed variant set), `PrismaticColor`, `PrismaticStrike` (closed: damage and condition), `PrismaticTargetResult`, `IndigoSaveTrack`, `PrismaticFollowUp` (closed: `IndigoFollowUp`, `VioletFollowUpPending`), `VioletFollowUp`, `PrismaticSaveTarget`, `PrismaticWallState`, `PrismaticLayerDestroyer` (closed variant set), `PrismaticLayerDestruction` (closed variant set), `PrismaticWallCrossing`, `ReincarnationOutcome` (closed variant set).
- **Invariants and where enforced:** band coverage (`OutcomeTable[T].validate`, test-only); 1 to 9 distinct teleporters (`TeleportingCreatures`); cardinal-only Confusion direction; unique candidates up to 1000; `IndigoSaveTrack` counts below 3; wall `destroyedCount` 0 to 7 (`NewPrismaticWallStateAfter`); non-negative `DeadFor`. Every constructor returns an error that is `errors.Is` `srderr.ErrInvalidInput` (untrusted input), except the unexported `OutcomeTable[T].validate`, which is called only by tests (programmer error, fixed table data only); the two Reincarnate refusals are results (`doc/rules/architecture.md` §6).
- **State:** none held by the package. `PrismaticWallState` and `IndigoSaveTrack` are values the host stores and passes back; each operation returns a new value.
- **Failures:** no error type of its own beyond `ErrFaceOutOfRange` and `ErrWallDestroyedCount`; roller failures are returned as `*srderr.PortError` (FR-091).

## 6. Interface

```go
// package combat, spell_outcome_*.go (indicative; pure, silent)
func TeleportKindFor(f TeleportFamiliarity, d100 int) (TeleportOutcomeKind, error) // FR-010, FR-012
func TeleportNeedsRoll(f TeleportFamiliarity) bool                                 // FR-011
func ConfusionRowFor(d10 int) (ConfusionRow, error)                                // FR-030
func ReincarnationSpeciesFor(d10 int) (species SpeciesID, rollAgain bool, err error)
func PrismaticRayFor(d8 int) (color PrismaticColor, special bool, err error)       // special = the 8
func BlinkVanishesOn(d6 int) (bool, error)                                         // FR-044
func BlinkReturnFor(location BlinkLocation) BlinkReturn                            // FR-043

var ErrFaceOutOfRange error // wraps srderr.ErrInvalidInput; FR-002, FR-044

type IndigoSaveTrack struct{ /* successes, failures; zero value is 0 and 0 */ } // FR-071
func NewIndigoSaveTrack(successes, failures int) (IndigoSaveTrack, error)
func (t IndigoSaveTrack) Record(o d20.Outcome) IndigoStep

func ResolveVioletFollowUp(o d20.Outcome) VioletResult // FR-072

type PrismaticWallState struct{ /* destroyedCount; zero value is the intact wall */ }
func NewPrismaticWallStateAfter(count int) (PrismaticWallState, error) // FR-075, error outside 0..7
func (w PrismaticWallState) Destroy(layer PrismaticColor, d PrismaticLayerDestroyer) (PrismaticWallState, PrismaticLayerDestruction) // FR-076

// resolve_spell_outcome_table_usecase.go
type ResolveSpellOutcomeTable struct{ /* roller, telemetry */ }
func NewResolveSpellOutcomeTable(roller dice.Roller, opts ...Option) (ResolveSpellOutcomeTable, error)
func (u ResolveSpellOutcomeTable) Resolve(ctx context.Context, cmd ResolveSpellOutcomeTableCommand) (ResolveSpellOutcomeTableResult, error) // FR-090
```


- **Commands:** `RollTeleportOutcomeCommand(subject, familiarity)`; `RollConfusionBehaviorCommand(CreaturesWithinReach)`; `RollPrismaticSprayCommand(SaveDC, targets: []PrismaticSaveTarget{ID, SaveAttempt})`; `ResolvePrismaticWallCrossingCommand(SaveDC, wall, creature: PrismaticSaveTarget)`; `RollReincarnationSpeciesCommand(creatureType, DeadFor, ChosenSpecies?)`; `RollBlinkTurnEndCommand(location)`.
- **Sync:** `dice.Roller` is synchronous (`dice-and-rolls.md` FR-045), so the use case returns directly; it takes `ctx` first only because it starts a span (`doc/rules/go.md` §4) and never stores it.
- **Driving it:** `spell-effects.md` returns `SpellEffectsTextOnly` for these spells; the host then calls this use case at the moment the spell calls for the roll (Confusion at the start of each affected turn, Blink at the end of each of the caster's turns). The host applies each `Damage` with `ApplyDamage` (passing `HalfDamage` where `Halved`), each condition with `ApplyConditionToSet`, and places a teleport arrival with `magical-effects.md` Teleportation.
- **Exports:** FR-112; `dice/dicetest` and `telemetry/telemetrytest` gain nothing.

## 7. Observability

Answers to the seven questions of `doc/rules/observability.md` §2:

1. **Failure modes:** a roller that fails (`srd5e.combat.errors`, `error.type` `dice_roller_error`); a stuck or biased roller (a table outcome that never or always occurs, FR-004's reroll-limit contract violation); a host passing wrong facts (not detectable).
2. **Signals that show it:** span error status, the shared `errors` count, and the share of each table outcome in `spell_table_rolls`.
3. **Attributes:** closed sets only, no id, species, amount or distance (FR-103).
4. **What to count:** each table lookup once (FR-102), so a biased roller is visible per table.
5. **Existing instrumentation:** the shared `combat` signals are used unchanged (FR-100).
6. **Logs:** none (`combat-foundations.md` FR-034).
7. **Cardinality:** `spell_table` 6 values and `table_outcome` at most 12 per table (at most 28 pairs: 4 + 4 + 8 + 10 + 2; `prismatic_wall` records none).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.resolve_spell_outcome_table` | span | `srd5e.combat.operation`, `srd5e.outcome`, `srd5e.combat.result` (`spell_table_rolled`, `spell_table_refused`), `srd5e.combat.result_class` (`applied`, `refused`), `srd5e.combat.spell_table` | The path of one table resolution | Sample traces; alert on error-status spans (a dice port failure) |
| `srd5e.combat.spell_table_rolls` | counter `{roll}` | `srd5e.combat.spell_table`, `srd5e.combat.table_outcome` | Rule outcome of every printed-table roll, so a stuck or biased roller shows | Compare each table's share with its printed band widths (for example Confusion row `does_nothing` is 5 in 10; Very familiar Mishap is 5 in 100); a share near 0% or 100% over at least 200 rolls points at the roller |

### 7.1 Closed values of `srd5e.combat.table_outcome`

`teleport`: `mishap`, `similar_area`, `off_target`, `on_target`. `confusion`: `moves_randomly`, `does_nothing`, `attacks_random_creature`, `chooses_behavior`. `prismatic_spray`: `red`, `orange`, `yellow`, `green`, `blue`, `indigo`, `violet`, `special`. `reincarnate`: `roll_again`, `dragonborn`, `dwarf`, `elf`, `gnome`, `goliath`, `halfling`, `human`, `orc`, `tiefling`. `blink`: `stays`, `vanishes` (the die band only; an already-ethereal 4 to 6 records `vanishes`, and `BlinkEndsAlreadyEthereal` is the result, not a table outcome). `prismatic_wall` records no `spell_table_rolls` (it makes no table roll). A Teleport row with no roll (FR-011) records nothing.

Constants (`combat/signals.go`, `combat-foundations.md` FR-039, FR-040): `ValueOperationResolveSpellOutcomeTable`, `SpanResolveSpellOutcomeTable`, `MetricSpellTableRolls`, `AttrSpellTable`, `AttrTableOutcome`, the two result values and every value above, each with a doc comment naming its attribute. These rows feed `doc/observability/signals.md`.

## 8. Test plan

Tests are in package `combat_test` (`combat/spell_outcome_*_test.go` for unit tests, `combat/resolve_spell_outcome_table_usecase_test.go` for the use case); the unit tests of unexported edge logic (`OutcomeTable[T]`) are an internal test file. Each case-table row is one table-driven test. Unit tests use plain values, no ports.

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002 | unit | `outcome_table_test.go`: gap, overlap, wrong start, wrong end rows; lookup at both ends of a band; 0, -1, `sides + 1` return `ErrFaceOutOfRange` |
| FR-003, FR-010 to FR-012 | unit | `teleport_table_test.go`: every boundary face of table T (T1 to T6), including 100 and the two no-roll rows; the False destination missing columns |
| FR-004, FR-005, FR-006 | application | `resolve_spell_outcome_table_test.go` with `dicetest.FixedRoller`: on Reincarnate 101 consecutive 1s (the 101st reroll request) return a `srderr.ErrContractViolation` error, 100 ones then a non-1 give `rerolls` 100 and do not; no pure table touches a roller; a refusal requests nothing |
| FR-013 to FR-017 | application | Teleport: one creature, three creatures (three `3d10`), an object (one), two Mishaps in a row (order of requests asserted via `requests`), Off Target with 2 and 24 miles and every direction face, `FalseDestination` ending as Similar Area, certain rows request nothing |
| FR-018, FR-019 | unit | `teleport_subject_test.go`: 0, 1, 9, 10 creatures, repeated id; outcome immutability |
| FR-030 to FR-036 | unit, application | `confusion_table_test.go`: C1 to C8; application: candidate counts 0, 1, 2, 1000, 1001, duplicate id; request order `1d10`, then `1d4` or `1d<n>`; `AttackCount` 1; `TargetMoves` false for C4 to C6 and no movement on no creature; the fact flag |
| FR-040 to FR-044 | unit, application | `blink_test.go`: faces 1 to 6 on both locations (an ethereal 4 to 6 reports the return as data, FR-041; recorded outcome `vanishes`), `BlinkReturnFor` both locations, face 0 and 7 return `ErrFaceOutOfRange`; application rolls `1d6` once |
| FR-050 to FR-054 | unit, application | R1 to R10; `1,1,5` gives `elf` with 2 rerolls; GM choice requests nothing; 10 days accepted, 10 days and 1 millisecond refused, `NotHumanoid` with `DeadTooLong` in order, negative duration returns an error; scenario: every table `SpeciesID` exists in the bundled content set |
| FR-060 to FR-068 | application | Prismatic Spray with `dicetest.FixedRoller`: each face 1 to 8, the 8 with an 8 inside the rerolls, one damage roll shared by two targets, a target with two damage rays, success halves, indigo and violet give a strike only on failure, empty target list, `ChooseToFail` rolls no d20, request order of FR-062 |
| FR-071 to FR-073 | unit | `indigo_save_track_test.go`: 3 successes, 3 failures, 2 and 2 then each outcome, `NewIndigoSaveTrack` with 3 returns an error; `prismatic_wall_state_test.go`: `NewPrismaticWallStateAfter` -1 and 8 return `ErrWallDestroyedCount`, 0 and 7 accepted; `violet_follow_up_test.go` both outcomes; architecture: the use case file does not name the Constitution or Wisdom save |
| FR-075 to FR-080 | unit | `prismatic_wall_state_test.go`: W1 to W12, immutability of the old state; application: crossing with all seven layers (request order save, damage per layer), with three layers left, with none, half damage on success |
| FR-090 to FR-094 | application, architecture | Exhaustive `switch` over command and result; refusal reasons; the roller failure is returned as a `*srderr.PortError` (also on the second of several rolls); equal results with no-op and recording `telemetrytest` doubles; no use case import (`architecture-guardrails.md` FR-097); no `combat.Event` |
| FR-100 to FR-103 | application | Recording `telemetry.Telemetry`: span name and attributes, `spell_table` on span and `operations`, one `spell_table_rolls` increment per table roll (rerolls counted, damage dice not), none for a refusal or a returned error, no log, no id in any attribute; catalogue and the `combat` signal constants match |
| FR-110 to FR-113 | architecture | Directory purity, import allowlist, export scan (godoc comment on every exported name, duplicate and generic names), consumer-name scan |
| errors-and-results.md FR-031, FR-034, FR-051 | application | The port failure of FR-091 is returned wrapped once as a `*srderr.PortError` (also on the second of several rolls) with a nil result and no partial result; the recorded failure's `error.type` is `srderr.ErrorType(err)` (FR-103); a panic that is not a port panic is not recovered. |

- **Doubles and suites:** none new (FR-113).
- **Mutation testing:** the `spell_outcome_*.go` files at a score of at least 80 with `gomu`, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors (`doc/rules/testing.md` §8), watching `<=` versus `<` at every band edge (table T), the 25 and 60 thresholds, the 10-day limit, the 4 to 6 Blink split, the three-of-a-kind counts and the outermost-layer rule. `ResolveSpellOutcomeTable` SHOULD be mutation tested.
- **Not covered:** nothing.

## 9. Open questions and decisions

No question is open and none is escalated. Web research was used to check readings; the SRD text and the package rules decide, and "interpretation" marks a reading the SRD does not state. Settled, one line each:

- **Q-1** (interpretation) Mishap damage is rolled per teleporting creature: the SRD says "Each teleporting creature ... takes 3d10", and the single-roll rule of p. 16 is stated only for effects that force saving throws, which a Mishap does not (FR-013). Community consensus agrees.
- **Q-2** (interpretation) a row whose whole range is On Target needs no die, so none is requested (FR-011); the Game Master's "roll" there cannot change the result.
- **Q-3** Reroll loops are unbounded in the SRD; the 100-reroll guard is package design for a stuck roller and cannot occur with a fair one (FR-004).
- **Q-4** (interpretation) a Prismatic Spray 8 gives two independent rays that may match; each is applied once, and one `12d6` is shared per cast because every target saves at once (FR-061, FR-063).
- **Q-5** (interpretation) a Wall layer's damage threshold is one damage instance, since the wall tracks no Hit Points (FR-077).
- **Q-6** (interpretation) a Wall crossing resolves every present layer; whether Restrained stops the creature is the host's (FR-079).
- **Q-7** the Teleport object limits, the Large size and the six-month Linked object limit are host facts (section 1); the familiarity judgement is the Game Master's.
- **Q-8** Confusion and Blink recurring saves and durations are an active-effects matter (`spell-effects.md` Q11); the table roll is all this spec owns.
- **Q-9** one use case with a closed command set, not six, to keep one span, one operation value and one telemetry contract; the cost is a wider closed variant set (FR-090).
- **Q-10** Reincarnate's character change is out of scope; a future `character` spec may consume `ReincarnationOutcome` (section 1).

**Amendments returned (this spec edits no other file):** (1) `shared-types-and-ownership.md` table 4.11: add `spell-outcome-tables` to rank 28. (2) `spell-effect-data.md` table E (E25 and the Blink row) and `spell-effects.md` section 1: point the text-only spells Teleport, Confusion, Prismatic Spray, Prismatic Wall, Reincarnate and Blink to this spec. (3) `combat-foundations.md` FR-037: add `srd5e.combat.spell_table` to the Attributes cell of `srd5e.combat.operations`. (4) `doc/documentation_index.md`, `release-0-1-0.md`, `architecture-guardrails.md` spec lists and `rules-glossary-coverage.md` (Teleportation, p. 190): add the new spec. (5) `magical-effects.md`: a prose pointer from Teleportation to the arrival table.

Open: none. Escalated: none.

## 10. Changelog impact

`Added` (combat): the printed spell tables for Teleport, Confusion, Prismatic Spray, Prismatic Wall, Reincarnate and Blink, the `ResolveSpellOutcomeTable` use case, and the signals `srd5e.combat.resolve_spell_outcome_table` and `srd5e.combat.spell_table_rolls`. Not breaking (new API only).
