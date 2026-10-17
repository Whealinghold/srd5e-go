# Movement: jumping, flying, burrowing, movement checks and the load Speed limit

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `conditions.md`, `equipment-rules.md`, `d20-tests.md`, `ability-scores.md`, `content-schema-and-decoder.md`, `initiative-and-actions.md` |

Data reference only: `content-monsters.md` (the `Hover` flag of `SpeedEntry`,
FR-016). Handoff targets, not dependencies: `environment-hazards-and-objects.md`
(`ResolveFall`, `FallDistance`) and `combat-action-resolution.md` (the review
event of FR-085).

## 1. Purpose and scope

The movement rules of the Rules Glossary that [initiative-and-actions.md](initiative-and-actions.md)
leaves out: how far a creature can jump (Long Jump and High Jump), when a flying
creature falls and how hovering prevents it, where a creature with a Burrow
Speed can burrow, the ability checks the SRD attaches to jumping, climbing and
swimming, and how the Speed limit of dragging, lifting or pushing a heavy load
meets `MovementSpeeds`.

The engine models no map. The host says how far a creature wants to jump, what it
lands in and what it flies with; this spec judges legality and returns structured
data. Everything is pure domain code: package-level functions and immutable value
objects, no use case, no port, no I/O, no dice. Checks are described, not rolled:
the host resolves them through `d20.Resolver` (d20-tests.md,
resolve-d20-test-use-case.md).

**Ownership.** This spec defines the types in section 5 and no others. Used from
elsewhere: `MovementSpeeds`, `MovementMode`, `Terrain` (initiative-and-actions.md);
`ConditionSet`, `ConditionEffects` (conditions.md); `CarryingCapacity`,
`LoadAssessment`, `LoadSpeedLimited` (equipment-rules.md, which owns the Carrying
Capacity table and reports the limit as `LimitFeet`, SRD 5.2.1, p. 178);
`Ability`, `AbilityScore` (ability-scores.md); `DifficultyClass` (d20-tests.md);
`Skill`, `ConditionKind` (content-schema-and-decoder.md). `ArmorRules.speedPenaltyFeet`, `Armor` (equipment-rules.md FR-007 to FR-010, content-equipment.md) give the penalty in feet.

**Supersedes** (shared-types-and-ownership.md FR-048, topic-specific spec). The
Speed arithmetic of a load limit is owned here (FR-001, FR-002, FR-058);
equipment-rules.md FR-019 keeps only `LimitFeet` and points here (documentation
only, not a dependency edge: a `character` spec cannot depend on a `combat` spec,
FR-013). Whether a flier falls (FR-037 to FR-047) had no earlier owner. No
requirement of initiative-and-actions.md, conditions.md or
environment-hazards-and-objects.md is superseded.

**Out of scope:**

- The Carrying Capacity table and the assessment of a load against it:
  equipment-rules.md FR-011 to FR-020. This spec only turns `LimitFeet` into a
  limited `MovementSpeeds`.
- Totalling item weights, and the Carrying Capacity of mounts (p. 100):
  `character` and `content-equipment.md`. Encumbrance variants are not in SRD
  5.2.1 and are not modelled.
- The extra-foot costs of Climbing, Crawling and Swimming, the speeds that remove
  them, the refusal of a Burrow or Fly move without such a speed, and Difficult
  Terrain: initiative-and-actions.md FR-077, FR-082, FR-115.
- Falling (distance, 1d6 Bludgeoning per 10 feet, landing Prone, the Reaction
  check into water, "Falling", p. 182): environment-hazards-and-objects.md
  (`ResolveFall`, `FallDistance`, `LandAfterFall`; Q4). Applying Prone after a
  failed landing: conditions.md.
- Spells and features that grant jumping, flying, hovering or burrowing, and
  features that change a jump (Q8); mounted and underwater flight and movement.
- Whether the GM invokes an "At the GM's option" check (rule-options.md, section
  1). Display text: results are structured data only.

## 2. Source

The PDF page index equals the printed page number (checked: PDF page 178 prints
178, page 189 prints 189).

- SRD 5.2.1, "Burrow Speed", p. 178; "Carrying Capacity", p. 178.
- SRD 5.2.1, "Climbing", pp. 178-179 (the GM-option check is at the top of p.
  179); "Climb Speed", p. 179.
- SRD 5.2.1, "Difficult Terrain", p. 181; "Falling [Hazard]", p. 182
  (cross-reference only).
- SRD 5.2.1, "Flying", "Fly Speed", p. 182; "Hover", p. 183; "High Jump", p. 183.
- SRD 5.2.1, "Jumping", p. 184; "Long Jump", pp. 184-185.
- SRD 5.2.1, "Incapacitated [Condition]", p. 184; "Paralyzed [Condition]",
  "Petrified [Condition]", "Prone [Condition]", p. 186; "Stunned [Condition]",
  p. 189; "Unconscious [Condition]", p. 191 (implication table: conditions.md
  FR-026).
- SRD 5.2.1, "Exhaustion", p. 181; "Grappled [Condition]", p. 182; "Restrained
  [Condition]", p. 187 (conditions that reduce a Fly Speed, FR-040).
- SRD 5.2.1, species "Size" traits, pp. 84-86 (height ranges, Q9).
- SRD 5.2.1, "Round Down", p. 187; "Speed" ("Special Speeds", "Changes to Your
  Speeds"), p. 188; "Swimming", "Swim Speed", p. 189.
- Package design: `doc/rules/architecture.md` sections 2, 3 and 6;
  `doc/rules/observability.md` sections 2 and 3; `doc/rules/testing.md` section 4.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Jumping, Long Jump, High Jump | `JumpKind` (`LongJump`, `HighJump`) | "either a Long Jump (horizontal) or a High Jump (vertical)" (p. 184). |
| (10 feet before the jump; a standing jump) | `JumpApproach` (`Running`, `Standing`) | "Standing" is the SRD's word; `Running` names the 10-foot condition. |
| (a jump to judge) | `JumpAttempt` (value object) | Kind, Strength score, conditions, movement made before, distance asked, movement left. |
| (the verdict) | `JumpAssessment` (closed: `JumpPermitted`, `JumpRefused`); `JumpRefusal` (`ProneRestrictedMovement`, `ExceedsMaximumDistance`, `InsufficientMovement`) | Operation-prefixed variants (shared-types-and-ownership.md FR-017). |
| Long Jump landing in Difficult Terrain | `JumpLanding` (closed: `JumpLandsClear`, `JumpLandingCheckRequired`) | p. 185. |
| Long Jump, the GM's low obstacle option | `JumpObstacle` (closed: `JumpObstacleCheckRequired`, `JumpObstacleNotCovered`) | p. 185. |
| (an ability check the movement rules name) | `MovementCheckRequirement`, `MovementCheckKind`, `MovementCheckFailureEffect` | `Skill` (its `ability` gives the Ability), DC and failure meaning. Describes a check; rolls nothing. |
| Flying, Fly Speed | `FlightFacts`, `FlightAssessment` (closed: `FlightStaysAloft`, `FlightFalls`), `FlightFallCause` (`Incapacitated`, `Prone`, `FlySpeedZero`) | p. 182. |
| Hover | `HoverAbility` (`none`, `Hover`) | p. 183. |
| Burrow Speed | `BurrowMedium` (`Sand`, `Earth`, `Mud`, `Ice`, `SolidRock`), `SolidRockAccess` (`none`, `AllowedByTrait`), `BurrowPermission` (closed: `BurrowAllowed`, `BurrowRefused`), `BurrowRefusal` (`SolidRock`) | p. 178. |
| Speed limit of 5 feet while dragging, lifting or pushing | `LimitSpeedsByLoad`, `LoadLimitScope` (`AllSpeedsByEqualAmount`, `WalkOnly`) | Cap from `LoadSpeedLimited.limitFeet`; "Changes to Your Speeds" (p. 188) carries it to special speeds. |
| Difficult Terrain | `Terrain` (initiative-and-actions.md) | Not redefined. |

Terms use SRD 5.2.1 wording (`doc/rules/architecture.md` section 3). Type names
were checked against every spec in `doc/specs/` for collisions.

## 4. Requirements

Ids are stable. Each cases-table row is one table-driven test.

### 4.1 Load and Speed

The Carrying Capacity maths and the assessment that yields the 5-foot limit are
equipment-rules.md's (FR-011 to FR-019, exposing only `LimitFeet`). Applying the
limit to a Speed and carrying it to special speeds is this spec's (FR-058). The armor Speed penalty is
applied the same way (FR-060).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `LimitSpeedsByLoad(speeds, limit, {scope})` MUST obtain the capped walking Speed by FR-058 from `limit.limitFeet` and MUST NOT hold the value 5 itself. With the default scope (FR-002) the result is `speeds.adjustedBy(-(speeds.walk - capped))`; with `LoadLimitScope.walkOnly` (FR-057) it is `speeds` with only the walking Speed replaced by `capped`. See cases 4.1a. | SRD 5.2.1, "Carrying Capacity" ("your Speed can be no more than 5 feet"), p. 178; equipment-rules.md FR-017, FR-019 | happy |
| FR-002 | The default scope, `LoadLimitScope.allSpeedsByEqualAmount`, MUST treat the cap as a decrease of the Speed and apply that decrease (walking Speed minus capped walking Speed) to every special speed the creature has, through `MovementSpeeds.adjustedBy`, so no speed falls below 0 (initiative-and-actions.md FR-076). A special speed that is absent stays absent; a walking Speed already at or below the cap gives a decrease of 0 and leaves every speed unchanged. See Q2. | SRD 5.2.1, "Speed", "Changes to Your Speeds" ("if an effect increases or decreases your Speed for a time, any special speed you have increases or decreases by an equal amount"), p. 188; "Carrying Capacity", p. 178 | unhappy (rule interaction) |
| FR-005 | `LimitSpeedsByLoad` MUST return a new `MovementSpeeds` and MUST NOT alter its input; with a walking Speed already at or below the cap the result MUST equal the input. | `doc/rules/architecture.md` section 3 (immutability); `doc/rules/testing.md` section 4 | unhappy (immutability) |
| FR-006 | `LimitSpeedsByLoad` MUST be the only code in this folder that touches a load limit, and it MUST read the cap only from `LoadSpeedLimited.limitFeet`: the folder MUST declare no Carrying Capacity type, table or variant of `LoadAssessment`, and its source MUST NOT contain the literal 5 as a Speed cap. | shared-types-and-ownership.md FR-017, FR-021, FR-048; `doc/rules/architecture.md` section 9; equipment-rules.md FR-019 | unhappy (duplicate) |
| FR-057 | `LoadLimitScope.walkOnly` MUST change only the walking Speed and leave Burrow, Climb, Fly and Swim speeds exactly as given. It exists for a host that rules the cap is not a decrease (Q2). | SRD 5.2.1, "Carrying Capacity", p. 178 (the 5-foot cap names "your Speed"); Q2 | unhappy (rule interaction) |
| FR-058 | The capped walking Speed MUST be the lower of `speeds.walk` and `limit.limitFeet`: a Speed above the cap becomes the cap, a Speed at or below it is unchanged (the cap never raises a Speed), and a Speed of 0 stays 0. Boundary rows with the cap 5: 30 gives 5, 6 gives 5, 5 gives 5, 4 gives 4, 0 gives 0. See cases 4.1a. | SRD 5.2.1, "Carrying Capacity" ("no more than 5 feet"), p. 178; equipment-rules.md FR-017, FR-019 | unhappy (boundary) |
| FR-059 | `LimitSpeedsByLoad` MUST accept only a `LoadSpeedLimited`; the other `LoadAssessment` variants (within carry, beyond the Drag/Lift/Push maximum) are not inputs, so a host cannot limit a Speed for a load that does not limit it. Whether a load is beyond the maximum is equipment-rules.md's assessment, not repeated here. | SRD 5.2.1, "Carrying Capacity", p. 178; equipment-rules.md FR-016 to FR-018 | unhappy (rule interaction) |
| FR-060 | `ApplyArmorSpeedPenalty(speeds, armor, strength)` MUST return `speeds.adjustedBy(-ArmorRules.speedPenaltyFeet(armor, strength))` (equipment-rules.md FR-007 to FR-010): a penalty of 10 is a decrease of the Speed that reaches every special speed by an equal amount, no speed falling below 0 (FR-002; initiative-and-actions.md FR-076); a penalty of 0 returns a value equal to the input. The penalty value is read only from `SpeedPenaltyFeet`; this spec holds no literal 10 and no Strength requirement. When a heavy load also limits the Speed, the host applies the armor first and `LimitSpeedsByLoad` to the result (interpretation: the 5-foot cap names the Speed the creature has, p. 178, so the armor decrease is already in it). See cases 4.1b. | SRD 5.2.1, "Armor" ("Strength"), p. 92; "Speed", "Changes to Your Speeds", p. 188; equipment-rules.md Q20 | unhappy (rule interaction) |

Cases 4.1a (one test per row; the cap is 5):

| Label | Input speeds | Scope | Expected speeds | Source | Path |
|---|---|---|---|---|---|
| walking above cap | walk 30 | default | walk 5 | FR-001 | happy |
| special speeds follow the decrease | walk 30, fly 40, swim 30, burrow 15, climb 20 | default | walk 5, fly 15, swim 5, burrow 0, climb 0 (decrease 25, floored at 0) | FR-002 | unhappy (rule interaction) |
| no special speeds | walk 30 | default | walk 5, no special speeds | FR-002 | unhappy (empty case) |
| walk 0 with fly | walk 0, fly 40 | default | walk 0, fly 40 (decrease 0) | FR-002, FR-058 | unhappy (rule interaction) |
| already at the cap | walk 5, fly 40 | default | unchanged, equal to the input | FR-002, FR-005, FR-058 | unhappy (boundary) |
| one over the cap | walk 6, fly 40 | default | walk 5, fly 39 (decrease 1) | FR-058, FR-002 | unhappy (boundary) |
| one under the cap | walk 4, fly 40 | default | unchanged | FR-058 | unhappy (boundary) |
| below the cap | walk 3, fly 40 | default | unchanged | FR-002, FR-005, FR-058 | unhappy (boundary) |
| walk 0 alone | walk 0 | default | walk 0 | FR-058 | unhappy (boundary) |
| walk only, special speeds kept | walk 30, fly 40, swim 30, burrow 15, climb 20 | walkOnly | walk 5, fly 40, swim 30, burrow 15, climb 20 | FR-057 | unhappy (rule interaction) |
| input unchanged | walk 30, fly 40 | default | the input value is still walk 30, fly 40 after the call | FR-005 | unhappy (immutability) |

Cases 4.1b (one test per row; Chain Mail, Strength requirement 13):

| Label | Input speeds | Strength | Expected speeds | Source | Path |
|---|---|---|---|---|---|
| below the requirement | walk 30, fly 40 | 12 | walk 20, fly 30 | FR-060 | unhappy (boundary) |
| at the requirement | walk 30, fly 40 | 13 | unchanged, equal to the input | FR-060 | unhappy (boundary) |
| floor at 0 | walk 5 | 12 | walk 0 | FR-060 | unhappy (boundary) |
| armor then heavy load | walk 30, fly 40 | 12 | after `LimitSpeedsByLoad` (cap 5): walk 5, fly 15 (decreases 10, then 15) | FR-060, FR-002 | unhappy (rule interaction) |

### 4.2 Jump distance

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-007 | `JumpKind` MUST have exactly two values, in this order: `LongJump`, `HighJump`. | SRD 5.2.1, "Jumping", p. 184 | happy |
| FR-008 | For a Long Jump, `JumpApproachFor` MUST give `Running` when the creature moved at least 10 feet immediately before the jump and `Standing` otherwise, whatever the mode of that movement. See cases 4.2b. | SRD 5.2.1, "Long Jump" ("if you move at least 10 feet immediately before the jump"), pp. 184-185 | happy |
| FR-009 | For a High Jump, `JumpApproachFor` MUST give `Running` only when the creature moved at least 10 feet **on foot** (mode `Walk`) immediately before the jump, and `Standing` for any shorter run or any other mode. See cases 4.2b and Q3. | SRD 5.2.1, "High Jump" ("if you move at least 10 feet on foot immediately before the jump"), p. 183 | unhappy (boundary) |
| FR-010 | A negative run-up distance MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`); 0 MUST be accepted and give `Standing`. | `doc/rules/architecture.md` section 6; `doc/rules/testing.md` section 4 (invalid input) | unhappy (invalid input) |
| FR-011 | The maximum `Running` Long Jump distance MUST equal the creature's Strength score in feet. See cases 4.2a. | SRD 5.2.1, "Long Jump" ("up to your Strength score"), p. 184 | happy |
| FR-012 | The maximum `Standing` Long Jump distance MUST be half the running maximum, rounded down. See cases 4.2a. | SRD 5.2.1, "Long Jump" ("only half that distance"), p. 185; "Round Down", p. 187 | unhappy (boundary) |
| FR-013 | The maximum `Running` High Jump height MUST be 3 plus the Strength modifier in feet, and never less than 0. See cases 4.2a. | SRD 5.2.1, "High Jump" ("3 plus your Strength modifier (minimum of 0 feet)"), p. 183 | happy |
| FR-014 | The maximum `Standing` High Jump height MUST be half the running maximum (taken after the minimum of 0), rounded down. See cases 4.2a. | SRD 5.2.1, "High Jump" ("only half that distance"), p. 183; "Round Down", p. 187 | unhappy (boundary) |
| FR-015 | `AssessJump` MUST return `JumpPermitted` when the creature is not Prone (FR-056) and the requested distance is at most the maximum for the kind and approach and at most the movement left. It MUST carry the kind, the approach, the maximum, the distance and the movement cost. See cases 4.2c. | SRD 5.2.1, "Long Jump", pp. 184-185; "High Jump", p. 183 | happy |
| FR-016 | A request above the maximum MUST return `JumpRefused` with reason `ExceedsMaximumDistance`, carrying the maximum and the requested distance. A request equal to the maximum MUST be permitted. See cases 4.2c and Q5. | SRD 5.2.1, "Long Jump" ("up to"), p. 184; "High Jump", p. 183 | unhappy (boundary) |
| FR-017 | A request within the maximum but above the movement left MUST return `JumpRefused` with reason `InsufficientMovement`, carrying the movement left. A request equal to the movement left MUST be permitted. | SRD 5.2.1, "Long Jump" ("each foot you jump costs a foot of movement"), p. 185; "High Jump", p. 183 | unhappy (boundary) |
| FR-018 | Refusal precedence, first match wins: `ProneRestrictedMovement` (FR-056), then `ExceedsMaximumDistance`, then `InsufficientMovement`. In particular, a request above the maximum **and** above the movement left gives `ExceedsMaximumDistance`. | `doc/rules/testing.md` section 4 (rule interactions); package design | unhappy (rule interaction) |
| FR-019 | A request of 0 feet MUST be permitted with a movement cost of 0, including when the maximum is 0 (a High Jump with Strength 1), when the movement left is 0 or less, and when the creature is Prone (a jump of 0 feet is no movement). | SRD 5.2.1, "High Jump" ("minimum of 0 feet"), p. 183 | unhappy (boundary) |
| FR-020 | A negative requested distance or a negative run-up (FR-010) MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`). A negative movement left is a legal game state and MUST NOT be an error: it MUST be treated as 0, so any request above 0 gives `InsufficientMovement` (carrying 0) and a request of 0 is permitted. The movement left is the remainder of the **walking** Speed, derived by the host as initiative-and-actions.md FR-115 does (speed plus Dash extra minus feet moved, which can be 0 or less after a Speed drop, FR-116), after any switch of speeds subtracts the distance already moved (p. 188). | SRD 5.2.1, "Speed", "Special Speeds", p. 188; initiative-and-actions.md FR-115, FR-116; `doc/rules/architecture.md` section 6 (a legal host value must give a result, not an `Error`) | unhappy (state error) |
| FR-021 | The movement cost of a permitted jump MUST equal the distance jumped, one foot of movement per foot, for both kinds and both approaches; this spec adds no Difficult Terrain extra and no multiplier. How the host charges it to the movement ledger is Q6. | SRD 5.2.1, "Long Jump" ("each foot you jump costs a foot of movement"), p. 185; "High Jump", p. 183 | happy |
| FR-022 | `JumpRules` MUST take no dice roller and roll nothing; equal inputs MUST give equal results on every call. | SRD 5.2.1, "Long Jump", "High Jump" (no roll is part of the distance); `doc/rules/testing.md` section 4 (determinism) | unhappy (determinism) |
| FR-056 | A request above 0 feet by a creature with the Prone condition effective (including by implication from Unconscious, conditions.md FR-026) MUST return `JumpRefused` with reason `ProneRestrictedMovement`: a Prone creature's only movement options are to crawl or to spend half its Speed to right itself, so it cannot jump. Standing up is `ConditionEffects.standUp` (conditions.md FR-086), not here. | SRD 5.2.1, "Prone [Condition]" ("Your only movement options are to crawl or to spend an amount of movement equal to half your Speed (round down) to right yourself and thereby end the condition"), p. 186 | unhappy (rule interaction) |

Cases 4.2a: maxima (feet; one test per row; running / standing):

| Label | Strength (modifier) | Long Jump | High Jump | Source | Path |
|---|---|---|---|---|---|
| lowest score | 1 (-5) | 1 / 0 | 0 / 0 (3 - 5 is below 0) | FR-011 to FR-014 | unhappy (boundary) |
| High Jump minimum 0 at -3 | 4 (-3) | 4 / 2 | 0 / 0 | FR-013 | unhappy (boundary) |
| High Jump 1 at -2 | 6 (-2) | 6 / 3 | 1 / 0 | FR-013, FR-014 | unhappy (boundary) |
| negative modifier | 8 (-1) | 8 / 4 | 2 / 1 | FR-013, FR-014 | unhappy (boundary) |
| modifier 0 | 10 (+0) | 10 / 5 | 3 / 1 | FR-011 to FR-014 | happy |
| odd score, standing rounds down | 11 (+0) | 11 / 5 | 3 / 1 | FR-012 | unhappy (boundary) |
| modifier +1 | 12 (+1) | 12 / 6 | 4 / 2 | FR-013 | happy |
| odd score 15 | 15 (+2) | 15 / 7 | 5 / 2 | FR-012, FR-014 | unhappy (boundary) |
| score 20 | 20 (+5) | 20 / 10 | 8 / 4 | FR-011, FR-013 | happy |
| highest score | 30 (+10) | 30 / 15 | 13 / 6 | FR-011, FR-013 | unhappy (boundary) |

Cases 4.2b: approach (one test per row):

| Label | Kind | Run-up feet | Run-up mode | Expected | Source | Path |
|---|---|---|---|---|---|---|
| Long Jump, exactly 10 | LongJump | 10 | Walk | Running | FR-008 | unhappy (boundary) |
| Long Jump, one short | LongJump | 9 | Walk | Standing | FR-008 | unhappy (boundary) |
| Long Jump, one more | LongJump | 11 | Walk | Running | FR-008 | happy |
| Long Jump, no run-up | LongJump | 0 | Walk | Standing | FR-008, FR-010 | unhappy (empty case) |
| Long Jump, any mode counts | LongJump | 10 | Fly | Running | FR-008 | unhappy (rule interaction) |
| High Jump, exactly 10 on foot | HighJump | 10 | Walk | Running | FR-009 | unhappy (boundary) |
| High Jump, one short | HighJump | 9 | Walk | Standing | FR-009 | unhappy (boundary) |
| High Jump, flew the 10 feet | HighJump | 10 | Fly | Standing | FR-009 | unhappy (rule interaction) |
| High Jump, crawled the 10 feet | HighJump | 10 | Crawl | Standing | FR-009, Q3 | unhappy (rule interaction) |
| High Jump, climbed or swam | HighJump | 15 | Climb | Standing | FR-009 | unhappy (rule interaction) |
| negative run-up | HighJump | -1 | Walk | `*InvalidError` | FR-010 | unhappy (invalid input) |

Cases 4.2c: assessment (Strength 10 unless stated; movement left 30 unless stated; no conditions unless stated):

| Label | Kind and approach | Requested | Movement left | Expected | Source | Path |
|---|---|---|---|---|---|---|
| at the maximum | Long Jump, running (max 10) | 10 | 30 | `JumpPermitted`, cost 10 | FR-015, FR-016 | unhappy (boundary) |
| one over the maximum | Long Jump, running | 11 | 30 | `JumpRefused(ExceedsMaximumDistance)`, max 10 | FR-016 | unhappy (boundary) |
| shorter than the maximum | Long Jump, running | 4 | 30 | `JumpPermitted`, cost 4 | FR-015, FR-021 | happy |
| standing max | Long Jump, standing (max 5) | 5 | 30 | `JumpPermitted`, cost 5 | FR-012, FR-015 | unhappy (boundary) |
| standing one over | Long Jump, standing | 6 | 30 | `JumpRefused(ExceedsMaximumDistance)`, max 5 | FR-016 | unhappy (boundary) |
| High Jump at the maximum | High Jump, running (max 3) | 3 | 30 | `JumpPermitted`, cost 3 | FR-015 | unhappy (boundary) |
| High Jump one over | High Jump, running | 4 | 30 | `JumpRefused(ExceedsMaximumDistance)`, max 3 | FR-016 | unhappy (boundary) |
| movement exactly enough | Long Jump, running | 10 | 10 | `JumpPermitted` | FR-017 | unhappy (boundary) |
| movement one short | Long Jump, running | 10 | 9 | `JumpRefused(InsufficientMovement)`, left 9 | FR-017 | unhappy (boundary) |
| both limits exceeded | Long Jump, running | 11 | 9 | `JumpRefused(ExceedsMaximumDistance)` | FR-018 | unhappy (rule interaction) |
| zero distance, no movement | Long Jump, running | 0 | 0 | `JumpPermitted`, cost 0 | FR-019 | unhappy (boundary) |
| zero maximum | High Jump, running, Strength 1 (max 0) | 0 | 30 | `JumpPermitted`, cost 0 | FR-019 | unhappy (boundary) |
| zero maximum, any height | High Jump, running, Strength 1 | 1 | 30 | `JumpRefused(ExceedsMaximumDistance)`, max 0 | FR-016 | unhappy (boundary) |
| negative request | Long Jump, running | -1 | 30 | `*InvalidError` | FR-020 | unhappy (invalid input) |
| negative movement left, any request | Long Jump, running | 5 | -5 | `JumpRefused(InsufficientMovement)`, left 0 | FR-020 | unhappy (state error) |
| negative movement left, no jump | Long Jump, running | 0 | -5 | `JumpPermitted`, cost 0 | FR-019, FR-020 | unhappy (state error) |
| Prone | Long Jump, running, Prone | 5 | 30 | `JumpRefused(ProneRestrictedMovement)` | FR-056 | unhappy (rule interaction) |
| Unconscious implies Prone | High Jump, running, Unconscious | 1 | 30 | `JumpRefused(ProneRestrictedMovement)` | FR-056 | unhappy (rule interaction) |
| Prone beats the other reasons | Long Jump, running, Prone | 11 | 9 | `JumpRefused(ProneRestrictedMovement)` | FR-018, FR-056 | unhappy (rule interaction) |
| Prone, zero distance | Long Jump, running, Prone | 0 | 30 | `JumpPermitted`, cost 0 | FR-019, FR-056 | unhappy (boundary) |
| same inputs twice | Long Jump, running | 10 | 30 | equal results on two calls | FR-022 | unhappy (determinism) |

### 4.3 Landing, obstacles and reach

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-023 | `JumpLandingFor(LongJump, TerrainDifficult)` MUST return `JumpLandingCheckRequired` carrying the requirement `RequirementFor(LongJumpLanding)` (Dexterity (Acrobatics), DC 10; failure means the creature has the Prone condition). | SRD 5.2.1, "Long Jump" ("If you land in Difficult Terrain, you must succeed on a DC 10 Dexterity (Acrobatics) check or have the Prone condition"), p. 185 | happy |
| FR-024 | `JumpLandingFor(LongJump, TerrainOpen)` MUST return `JumpLandsClear`. | SRD 5.2.1, "Long Jump", p. 185 | unhappy (rule interaction) |
| FR-025 | `JumpLandingFor(HighJump, ...)` MUST return `JumpLandsClear` for both terrains: the SRD states the landing check under Long Jump only. | SRD 5.2.1, "High Jump", p. 183; "Long Jump", p. 185 | unhappy (rule interaction) |
| FR-026 | `ObstacleForLongJump` MUST return `JumpObstacleCheckRequired` carrying `RequirementFor(LongJumpObstacle)` (Strength (Athletics), DC 10; failure means the creature hits the obstacle) when the jump distance is at least 1 foot and the obstacle height, in whole feet, is at most the quarter of the distance **rounded down** (`distanceFeet / 4`), a height of exactly that quarter included. See cases 4.3a and Q9. | SRD 5.2.1, "Long Jump" ("At your GM's option, you must succeed on a DC 10 Strength (Athletics) check to clear a low obstacle (no taller than a quarter of the jump's distance) ... Otherwise, you hit the obstacle"), p. 185; "Round Down" ("Whenever you divide or multiply a number in the game, round down"), p. 187 | happy |
| FR-027 | A taller obstacle MUST return `JumpObstacleNotCovered`; the SRD states no rule for it and none is invented (Q7). A jump distance of 0 (no jump, FR-019) MUST also return `JumpObstacleNotCovered`, whatever the obstacle height: a creature that does not leave the ground clears nothing. | SRD 5.2.1, "Long Jump" ("assumes that the height of the jump doesn't matter"), p. 185 | unhappy (boundary) |
| FR-028 | `ObstacleForLongJump` MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`) for a negative distance and for a negative obstacle height. Both are `int` feet, so there is no NaN or infinite input. A distance of 0 is not an error (FR-027). | `doc/rules/architecture.md` section 6; `doc/rules/testing.md` section 4 (invalid input) | unhappy (invalid input) |
| FR-029 | `HighJumpReachFeet(jumpHeightFeet, creatureHeightFeet)` MUST equal the jump height plus one and a half times the creature's height, in whole feet, with the multiplication rounded down: `jumpHeightFeet + creatureHeightFeet * 3 / 2`. See cases 4.3b and Q9. | SRD 5.2.1, "High Jump" ("you can reach a distance equal to the height of the jump plus 1 1/2 times your height"), p. 183; "Round Down", p. 187 | happy |
| FR-030 | `HighJumpReachFeet` MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`) for a negative jump height and for a creature height below 1 foot. Both are `int` feet; the host chooses a height inside the species range the SRD gives (Q9). | `doc/rules/architecture.md` section 6; `doc/rules/testing.md` section 4 (invalid input) | unhappy (invalid input) |

Cases 4.3a: low obstacle (one test per row; whole feet; quarter = distance / 4):

| Label | Jump distance (ft) | Obstacle height (ft) | Expected | Source | Path |
|---|---|---|---|---|---|
| exactly a quarter | 20 | 5 | `JumpObstacleCheckRequired` | FR-026 | unhappy (boundary) |
| one over a quarter | 20 | 6 | `JumpObstacleNotCovered` | FR-027 | unhappy (boundary) |
| well under | 20 | 4 | `JumpObstacleCheckRequired` | FR-026 | happy |
| flat (height 0) | 20 | 0 | `JumpObstacleCheckRequired` | FR-026 | unhappy (boundary) |
| quarter rounds down, at it | 7 | 1 | `JumpObstacleCheckRequired` (7 / 4 = 1) | FR-026 | unhappy (boundary) |
| quarter rounds down, over it | 7 | 2 | `JumpObstacleNotCovered` | FR-027 | unhappy (boundary) |
| short jump, quarter is 0 | 3 | 1 | `JumpObstacleNotCovered` (3 / 4 = 0) | FR-026, FR-027 | unhappy (boundary) |
| one-foot jump, flat | 1 | 0 | `JumpObstacleCheckRequired` | FR-026 | unhappy (boundary) |
| one-foot jump, one foot | 1 | 1 | `JumpObstacleNotCovered` | FR-027 | unhappy (boundary) |
| no jump | 0 | 0 | `JumpObstacleNotCovered` | FR-027 | unhappy (empty case) |
| no jump, any height | 0 | 1 | `JumpObstacleNotCovered` | FR-027 | unhappy (empty case) |
| negative distance | -1 | 1 | `*InvalidError` | FR-028 | unhappy (invalid input) |
| negative height | 20 | -1 | `*InvalidError` | FR-028 | unhappy (invalid input) |

Cases 4.3b: reach (one test per row; whole feet):

| Label | Jump height (ft) | Creature height (ft) | Expected reach (ft) | Source | Path |
|---|---|---|---|---|---|
| ordinary | 3 | 6 | 12 | FR-029 | happy |
| no jump | 0 | 6 | 9 | FR-029 | unhappy (boundary) |
| half foot rounds down | 0 | 5 | 7 (5 * 3 / 2 = 7) | FR-029 | unhappy (boundary) |
| tall jump | 13 | 6 | 22 | FR-029 | happy |
| smallest creature height | 0 | 1 | 1 (1 * 3 / 2 = 1) | FR-029, FR-030 | unhappy (boundary) |
| creature height 0 | 3 | 0 | `*InvalidError` | FR-030 | unhappy (invalid input) |
| negative creature height | 3 | -1 | `*InvalidError` | FR-030 | unhappy (invalid input) |
| negative jump height | -1 | 6 | `*InvalidError` | FR-030 | unhappy (invalid input) |

### 4.4 Movement checks

These describe checks. They do not roll them (FR-034).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | `MovementCheckRequirement` MUST offer exactly four requirements with the skill and DC of cases 4.4a; the ability is `skill.ability` (shared-types-and-ownership.md FR-006) and is not stored separately, so the two cannot disagree (Athletics gives Strength, Acrobatics gives Dexterity). | SRD 5.2.1, "Long Jump", p. 185; "Climbing", p. 179; "Swimming", p. 189 | happy |
| FR-032 | The failure effect MUST be `BecomesProne` for the Long Jump landing, `HitsObstacle` for the Long Jump obstacle, and `NotSpecified` for climbing and swimming, whose SRD text states no consequence of failure. | SRD 5.2.1, "Long Jump", p. 185; "Climbing", p. 179; "Swimming", p. 189 | unhappy (rule interaction) |
| FR-033 | The four requirements MUST be offered as values the host uses when its GM invokes the option. This spec MUST NOT decide whether a surface is slippery, has few handholds or whether water is rough, and MUST NOT carry any switch for "the GM's option". Checked by an architecture scan of the public signatures and fields of this folder (no `bool` or enum parameter named for the GM option), since it is a negative property that no runtime test can exercise. | SRD 5.2.1, "Climbing" ("At the GM's option"), p. 179; "Swimming" ("At the GM's option"), p. 189; rule-options.md section 1 | unhappy (rule interaction) |
| FR-034 | A `MovementCheckRequirement` MUST carry no roll, total or outcome (checked by an architecture scan of its fields). The `ability` and the target (`DifficultyClass`, d20) are the inputs of an `AbilityCheckCommand`; the host resolves it through `d20.Resolver` and the proficiency, modifiers and circumstances are the host's to supply. | SRD 5.2.1, "D20 Tests", p. 6; resolve-d20-test-use-case.md FR-001; d20-tests.md FR-021; shared-types-and-ownership.md FR-002 | unhappy (rule interaction) |
| FR-035 | Because `MovementCheckRequirement` has no exported constructor and `RequirementFor` is the only source, the four requirements MUST be equal to themselves on every access (`==`) and MUST be pairwise distinct (unequal). | `doc/rules/architecture.md` section 3 (value object) | unhappy (immutability) |

Cases 4.4a (one test per row; the Ability column is `skill.ability`, shown for reading):

| Label | `MovementCheckKind` | Ability | Skill | DC | Failure effect | Source | Path |
|---|---|---|---|---|---|---|---|
| Long Jump landing | `LongJumpLanding` | Dexterity | Acrobatics | 10 | `BecomesProne` | p. 185 | happy |
| Long Jump obstacle | `ObstacleForLongJump` | Strength | Athletics | 10 | `HitsObstacle` | p. 185 | happy |
| Climbing a hard surface (slippery or few handholds) | `ClimbingHardSurface` | Strength | Athletics | 15 | `NotSpecified` | p. 179 | happy |
| Swimming in rough water | `SwimmingRoughWater` | Strength | Athletics | 15 | `NotSpecified` | p. 189 | happy |

### 4.5 Flying and Hover

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-036 | `FlightFacts` MUST be built from the creature's `ConditionSet`, its **base** Fly Speed in feet (0 or more; before the effect of conditions, but after any non-condition change such as a spell) and its `HoverAbility`; a negative base Fly Speed MUST return an `*InvalidError` (matching `srderr.ErrInvalidInput`). `AssessFlight` MUST compute the effective Fly Speed as `ConditionEffects.speed(conditions, baseFlySpeedFeet).value` (conditions.md FR-107, applied to the Fly Speed independently of the walking Speed, D-14), so the host does not pre-apply conditions. A creature with no Fly Speed is not flying and is not described by `FlightFacts`. | SRD 5.2.1, "Fly Speed" ("While you have a Fly Speed"), p. 182 | unhappy (invalid input) |
| FR-037 | `AssessFlight` MUST return `FlightStaysAloft` with no overridden causes when the creature is neither Incapacitated nor Prone and its effective Fly Speed (FR-036) is above 0. | SRD 5.2.1, "Flying", p. 182; "Fly Speed", p. 182 | happy |
| FR-038 | A creature with the Incapacitated condition effective, including by implication from Paralyzed, Petrified, Stunned or Unconscious, MUST fall with cause `Incapacitated` unless it can hover. | SRD 5.2.1, "Flying" ("you fall if you have the Incapacitated or Prone condition"), p. 182; conditions.md FR-026 | unhappy (rule interaction) |
| FR-039 | A creature with the Prone condition effective, including by implication from Unconscious, MUST fall with cause `Prone` unless it can hover. | SRD 5.2.1, "Flying", p. 182; conditions.md FR-026 | unhappy (rule interaction) |
| FR-040 | A creature whose **effective** Fly Speed (FR-036) is 0 MUST fall with cause `FlySpeedZero` unless it can hover; an effective Fly Speed of 1 MUST NOT cause a fall. A condition that sets Speed to 0 (Grappled, Paralyzed, Petrified, Restrained, Unconscious) or an Exhaustion reduction that takes the Fly Speed to 0 therefore gives `FlySpeedZero` without the host reducing anything. A change that is not a condition (a spell) is the host's to fold into the base Fly Speed; the host learns that a condition changed a Speed from the review event of combat-action-resolution.md FR-085 (prose only, not a dependency) and keeps the tracker's speeds current with `UpdateMovementSpeeds` (initiative-and-actions.md FR-116, FR-117). | SRD 5.2.1, "Flying" ("or your Fly Speed is reduced to 0"), p. 182; "Speed", "Changes to Your Speeds" ("if your Speed is reduced to 0 and you have a Climb Speed, your Climb Speed is also reduced to 0"), p. 188; "Grappled", p. 182; "Restrained", p. 187; "Exhaustion", p. 181; conditions.md FR-107 | unhappy (boundary) |
| FR-041 | When several causes apply, `FlightFalls` MUST list each cause once, in the fixed order `Incapacitated`, `Prone`, `FlySpeedZero`. | `doc/rules/testing.md` section 4 (rule interactions); SRD 5.2.1, "Flying", p. 182 | unhappy (rule interaction) |
| FR-042 | A creature that can hover MUST return `FlightStaysAloft` for every cause of FR-038 to FR-040, and the result MUST list the causes hovering overrode, in the order of FR-041. | SRD 5.2.1, "Flying" ("You can stay aloft in those circumstances if you can hover"), p. 182; "Hover", p. 183 | unhappy (rule interaction) |
| FR-043 | A creature that can hover with no cause present MUST return `FlightStaysAloft` listing no causes. | SRD 5.2.1, "Hover", p. 183 | unhappy (empty case) |
| FR-044 | `HoverAbility` MUST be an input. This spec MUST NOT derive hovering from a speed, a condition or a creature type; the host reads it from the stat block (`SpeedEntry.hover`, content-monsters.md FR-016) or from the effect that grants it. Checked by an architecture scan of the public signatures of `FlightRules` and `FlightFacts` (hover is only ever read from the `HoverAbility` field). | SRD 5.2.1, "Hover" ("as noted in their stat blocks, and some spells and other effects grant the ability"), p. 183 | unhappy (rule interaction) |
| FR-045 | The assessment MUST depend on the facts only: neither movement left this turn, nor the turn, nor how long the creature has been aloft is an input, so a creature with no movement left stays aloft and the same facts give the same answer on any turn. Checked by an architecture scan of the `FlightFacts` fields and the `assess` signature (no movement, turn or duration parameter), plus a unit test that equal facts give equal results on repeated calls. | SRD 5.2.1, "Fly Speed" ("you can stay aloft until you land, fall, or die"), p. 182 | unhappy (state error) |
| FR-046 | `FlightFalls` MUST carry only its causes. It MUST NOT carry a distance, damage or a landing condition; those belong to `ResolveFall`, `FallDistance` and `LandAfterFall` in environment-hazards-and-objects.md and to `conditions.md`. | SRD 5.2.1, "Falling [Hazard]", p. 182 (cross-reference) | unhappy (rule interaction) |
| FR-047 | `AssessFlight` MUST NOT alter its inputs, and equal facts MUST give equal results with equality on every call. | `doc/rules/testing.md` section 4 (immutability, determinism) | unhappy (immutability) |

Cases 4.5a (one test per row; `ConditionSet` shown as its explicit conditions; the
effective set follows conditions.md FR-124; the Fly Speed column is the **base**
Fly Speed, FR-036):

| Label | Conditions | Fly Speed | Hover | Expected | Source | Path |
|---|---|---|---|---|---|---|
| nothing wrong | none | 60 | none | `FlightStaysAloft`, no causes | FR-037 | happy |
| Incapacitated | Incapacitated | 60 | none | `FlightFalls([Incapacitated])` | FR-038 | unhappy (rule interaction) |
| Paralyzed implies Incapacitated, and has Speed 0 | Paralyzed | 60 | none | `FlightFalls([Incapacitated, FlySpeedZero])` | FR-038, FR-040 | unhappy (rule interaction) |
| Stunned implies Incapacitated | Stunned | 60 | none | `FlightFalls([Incapacitated])` | FR-038 | unhappy (rule interaction) |
| Prone | Prone | 60 | none | `FlightFalls([Prone])` | FR-039 | unhappy (rule interaction) |
| Unconscious implies both, and has Speed 0 | Unconscious | 60 | none | `FlightFalls([Incapacitated, Prone, FlySpeedZero])` | FR-038, FR-039, FR-040, FR-041 | unhappy (rule interaction) |
| Fly Speed 0 | none | 0 | none | `FlightFalls([FlySpeedZero])` | FR-040 | unhappy (boundary) |
| Fly Speed 1 | none | 1 | none | `FlightStaysAloft`, no causes | FR-040 | unhappy (boundary) |
| Prone and Fly Speed 0 | Prone | 0 | none | `FlightFalls([Prone, FlySpeedZero])` | FR-041 | unhappy (rule interaction) |
| all three | Incapacitated, Prone | 0 | none | `FlightFalls([Incapacitated, Prone, FlySpeedZero])` | FR-041 | unhappy (rule interaction) |
| Grappled has Speed 0 | Grappled | 60 | none | `FlightFalls([FlySpeedZero])` | FR-040 | unhappy (rule interaction) |
| Restrained has Speed 0 | Restrained | 60 | none | `FlightFalls([FlySpeedZero])` | FR-040 | unhappy (rule interaction) |
| Stunned alone keeps its Speed | Stunned | 60 | none | `FlightFalls([Incapacitated])` (no `FlySpeedZero`) | FR-038, FR-040 | unhappy (rule interaction) |
| Exhaustion takes Fly Speed to 0 | Exhaustion 4 | 15 | none | `FlightFalls([FlySpeedZero])` (15 - 20 floored at 0) | FR-040 | unhappy (boundary) |
| Exhaustion leaves some Fly Speed | Exhaustion 1 | 10 | none | `FlightStaysAloft`, no causes (effective 5) | FR-040 | unhappy (boundary) |
| Grappled with Exhaustion | Grappled, Exhaustion 2 | 30 | none | `FlightFalls([FlySpeedZero])` (Speed 0 wins) | FR-040 | unhappy (rule interaction) |
| Grappled, hover | Grappled | 60 | hover | `FlightStaysAloft`, overrode [FlySpeedZero] | FR-040, FR-042 | unhappy (rule interaction) |
| hover, Incapacitated | Incapacitated | 60 | hover | `FlightStaysAloft`, overrode [Incapacitated] | FR-042 | unhappy (rule interaction) |
| hover, Prone and Fly Speed 0 | Prone | 0 | hover | `FlightStaysAloft`, overrode [Prone, FlySpeedZero] | FR-042 | unhappy (rule interaction) |
| hover, nothing wrong | none | 60 | hover | `FlightStaysAloft`, no causes | FR-043 | unhappy (empty case) |
| negative base Fly Speed | none | -1 | none | `*InvalidError` | FR-036 | unhappy (invalid input) |

### 4.6 Burrowing

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | `BurrowMedium` MUST have exactly five values, in this order: `Sand`, `Earth`, `Mud`, `Ice`, `SolidRock`. | SRD 5.2.1, "Burrow Speed", p. 178 | happy |
| FR-049 | `BurrowPermissionFor` MUST return `BurrowAllowed` for sand, earth, mud and ice, whatever the `SolidRockAccess`. | SRD 5.2.1, "Burrow Speed" ("move through sand, earth, mud, or ice"), p. 178 | happy |
| FR-050 | `BurrowPermissionFor(SolidRock, SolidRockNone)` MUST return `BurrowRefused` with reason `SolidRock`. | SRD 5.2.1, "Burrow Speed" ("The creature can't burrow through solid rock unless the creature has a trait that allows it to do so"), p. 178 | unhappy (rule interaction) |
| FR-051 | `BurrowPermissionFor(SolidRock, SolidRockAllowedByTrait)` MUST return `BurrowAllowed`. The host decides that a trait allows it; the engine does not read trait text. | SRD 5.2.1, "Burrow Speed", p. 178 | unhappy (rule interaction) |
| FR-052 | `BurrowRules` MUST NOT read the creature's speeds. A burrow move by a creature with no Burrow Speed is refused by `initiative-and-actions.md` FR-082 and not here. | SRD 5.2.1, "Burrow Speed" ("A creature that has a Burrow Speed"), p. 178; initiative-and-actions.md FR-082 | unhappy (rule interaction) |

Cases 4.6a (one test per row):

| Label | Medium | Solid rock access | Expected | Source | Path |
|---|---|---|---|---|---|
| sand | sand | none | `BurrowAllowed` | FR-049 | happy |
| earth | earth | none | `BurrowAllowed` | FR-049 | happy |
| mud | mud | none | `BurrowAllowed` | FR-049 | happy |
| ice | ice | none | `BurrowAllowed` | FR-049 | happy |
| earth, trait irrelevant | earth | allowedByTrait | `BurrowAllowed` | FR-049 | unhappy (rule interaction) |
| solid rock | solidRock | none | `BurrowRefused(solidRock)` | FR-050 | unhappy (rule interaction) |
| solid rock with a trait | solidRock | allowedByTrait | `BurrowAllowed` | FR-051 | unhappy (rule interaction) |

### 4.7 Package rules

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-053 | All types in section 5 MUST be immutable, pure, synchronous and deterministic, with no goroutine, `context.Context`, port, `math/rand`, `time.Now` or `os`. In package `combat` (`movement_*.go`) they MUST import only the kernel packages and the published API of `content`, `character` (for `LoadSpeedLimited`) and `d20` (the last for `DifficultyClass`, FR-034), and the combat specs to its left in the shared-types-and-ownership.md FR-015 order (conditions.md, initiative-and-actions.md), all allowed by the context map in `doc/rules/architecture.md` section 2. It MUST NOT import any adapter package, any `toolbox` code, `encounter-aggregate`, or `combat-action-resolution` (a forbidden or downstream edge, shared-types-and-ownership.md FR-013, FR-045). It defines no use case, so shared-types-and-ownership.md FR-044 (no use case calls a use case) and the `<UseCase>Result` naming of shared-types-and-ownership.md FR-017 are met trivially; its closed outcomes keep operation-prefixed variant names. | `doc/rules/architecture.md` sections 1, 2, 3, 9; shared-types-and-ownership.md FR-013, FR-015 | unhappy (forbidden edge) |
| FR-054 | No result type or failure effect of this spec MUST carry a display string; every outcome is an enum, a closed variant or a number. | `doc/rules/architecture.md` sections 3 and 6 | unhappy (invalid input) |
| FR-055 | Every type in section 5 MUST be exported once, with a doc comment comment, from package `combat` (no re-export), and none MUST share a name with any other exported member. | `doc/rules/architecture.md` sections 4, 7 and 9; shared-types-and-ownership.md FR-017, FR-021 | unhappy (name collision) |
| FR-061 | Species handoffs (`species-trait-behaviour.md`) are host facts; this spec reads no species type. (1) The walking Speed of `MovementSpeeds` is `SpeciesPassives.speedFeet(baseSpeed, largeFormActive)` (35 for the Wood Elf lineage, plus 10 while Large Form lasts), before FR-060 and FR-001. (2) While Draconic Flight lasts (`DraconicFlightEffect.flySpeedEqualsSpeed`), the host puts a Fly Speed equal to that walking Speed into the same `MovementSpeeds` before FR-060 and FR-001 are applied once to the whole, so both speeds fall by the same amount (FR-002); `FlightFacts` takes it as the base Fly Speed with `HoverAbility` none, so Incapacitated or Prone makes it fall (FR-038, FR-039), and the host ends the effect on retracting or Incapacitated. (3) The Advantage of Large Form on Strength checks is a `SpeciesAdvantage` the host adds to the `circumstances` of the Strength requirements (FR-034), not to a Dexterity (Acrobatics) one. (4) Halfling Nimbleness (moving through a space one size larger) is not a movement-lifting rule: passing and ending in occupied spaces is `initiative-and-actions.md` FR-080 and FR-081, whose owner must accept the pass-through permission. Interpretation of (2): "equal to your Speed" is the Speed the creature has when the wings appear. | SRD 5.2.1, "Draconic Flight", "Halfling Nimbleness", "Large Form", pp. 84-86; "Flying", p. 182 | unhappy (rule interaction) |

### Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-003 | Withdrawn: moved to movement-lifting-and-flight.md (FR-058) |
| FR-004 | Withdrawn: moved to movement-lifting-and-flight.md (FR-058) |

## 5. Model

All types are immutable values (unexported fields where an invariant exists,
value receivers, comparable where they hold no slice), in package `combat`
(files `movement_*.go`, one per concept). In the combat order
(shared-types-and-ownership.md FR-015) this spec is a leaf after
`initiative-and-actions`; nothing imports it (Q10). There are no service types:
the rules are package-level functions.

- **`LimitSpeedsByLoad`** (function): `func LimitSpeedsByLoad(speeds MovementSpeeds, limit LoadSpeedLimited, scope LoadLimitScope) MovementSpeeds` (FR-001, FR-002, FR-005, FR-006, FR-057 to FR-059). **`LoadLimitScope`**: `AllSpeedsByEqualAmount` (the zero value, the default), `WalkOnly`.
- **`ApplyArmorSpeedPenalty`** (function): `func ApplyArmorSpeedPenalty(speeds MovementSpeeds, armor Armor, strength ability.Score) MovementSpeeds` (FR-060).
- **`JumpKind`**, **`JumpApproach`**: enums (FR-007 to FR-009).
- **`JumpAttempt`**: `Kind`, `Strength` (`ability.Score`), `Conditions` (`ConditionSet`), `RunUpFeet`, `RunUpMode` (`MovementMode`), `RequestedFeet`, `MovementLeftFeet` (remainder of the walking Speed, FR-020), built by `NewJumpAttempt(...) (JumpAttempt, error)`. The constructor returns an `*InvalidError` (matching `srderr.ErrInvalidInput`) for a negative run-up or requested distance (FR-010, FR-020); a negative movement left is stored as 0. It derives the approach. The zero `JumpAttempt` is invalid (`IsZero`).
- **`JumpAssessment`** (closed interface): `JumpPermitted{Kind, Approach, MaximumFeet, DistanceFeet, MovementCostFeet}`, `JumpRefused{Kind, Approach, Reason JumpRefusal, MaximumFeet, RequestedFeet, MovementLeftFeet}`. **`JumpRefusal`**: `ProneRestrictedMovement`, `ExceedsMaximumDistance`, `InsufficientMovement`, in precedence order (FR-015 to FR-018, FR-056).
- **`JumpLanding`** (closed): `JumpLandsClear`, `JumpLandingCheckRequired{Requirement}`. **`JumpObstacle`** (closed): `JumpObstacleCheckRequired{Requirement}`, `JumpObstacleNotCovered`.
- **Jump functions:** `func JumpApproachFor(kind JumpKind, runUpFeet int, runUpMode MovementMode) (JumpApproach, error)`, `func JumpMaximumFeet(kind JumpKind, strength ability.Score, approach JumpApproach) int`, `func AssessJump(attempt JumpAttempt) JumpAssessment`, `func JumpLandingFor(kind JumpKind, terrain Terrain) JumpLanding`, `func ObstacleForLongJump(distanceFeet, obstacleHeightFeet int) (JumpObstacle, error)`, `func HighJumpReachFeet(jumpHeightFeet, creatureHeightFeet int) (int, error)`. All distances and heights are whole feet.
- **`MovementCheckRequirement`**: `Kind()` (`MovementCheckKind`: `LongJumpLanding`, `LongJumpObstacle`, `ClimbingHardSurface`, `SwimmingRoughWater`), `Skill()`, `Ability()` (returns `Skill().Ability()`), `Target()` (`d20.DifficultyClass`), `OnFailure()` (`MovementCheckFailureEffect`: `BecomesProne`, `HitsObstacle`, `NotSpecified`). Unexported fields; the four values are obtained only from `RequirementFor(kind MovementCheckKind) MovementCheckRequirement` (the zero requirement is invalid, `IsZero`) (FR-031).
- **`FlightFacts`**: `Conditions`, `BaseFlySpeedFeet`, `Hover`, built by `NewFlightFacts(...) (FlightFacts, error)`; the effective Fly Speed uses `ConditionEffects.Speed` (FR-036, FR-040). **`FlightAssessment`** (closed): `FlightStaysAloft{OverriddenCauses}`, `FlightFalls{Causes}`, `Causes` non-empty and ordered (FR-041). **`FlightFallCause`**; **`AssessFlight(FlightFacts) FlightAssessment`**.
- **`BurrowMedium`**, **`SolidRockAccess`**, **`BurrowPermission`** (closed: `BurrowAllowed`, `BurrowRefused{Reason BurrowRefusal}`), **`BurrowRefusal`**, **`BurrowPermissionFor(medium, access) BurrowPermission`**.

Invariants: non-negative inputs are enforced in the `NewJumpAttempt` and
`NewFlightFacts` constructors and in the functions taking raw numbers;
`FlightFalls.Causes` is non-empty and ordered where it is built; the four
requirements are the only `MovementCheckRequirement` values.

Failures: every rule outcome is a closed result; only invalid arguments are
`*InvalidError`s (matching `srderr.ErrInvalidInput`; `doc/rules/architecture.md`
section 6). Every numeric input is computed by the host from its own state, so a
negative value is the caller's error: the `NewX` constructors and the checked
functions return it as an error, and no `MustX` is offered for host-computed
numbers (as equipment-rules.md FR-020). No infrastructure failure, so no
`*srderr.PortError`.

Rounding: every division and multiplication is "Round Down" (p. 187) on whole
feet: halving a jump (integer `/ 2`), the quarter of a jump (`distance / 4`,
FR-026), one and a half times a height (`height * 3 / 2`, FR-029). Integer
division of non-negative `int`s rounds down. No floating point (Q9).

**Relationship to other described checks.** `HideCheckSpec`
(hiding-searching-and-senses.md), `FallLandingCheck` (environment-hazards-and-objects.md)
and the "save due" of toolbox-environmental-effects.md also describe a check.
`MovementCheckRequirement` is an intentional parallel: a closed set of four
values, with a failure effect (`OnFailure`) the others lack, one skill where
`FallLandingCheck` offers two, and no stored `Ability` (derived from `Skill`,
shared-types-and-ownership.md FR-006). It is not conditions.md's `CheckRequirement`
(sight, hearing, none). All migrate together if `d20` publishes a shared
type (Q17).

## 6. Interface

- **No commands, use cases or ports.** The functions are synchronous, take plain
  values and no `context.Context` (no I/O, no telemetry). No package use case calls
  them today; the host does, resolves any returned `MovementCheckRequirement`
  through `d20.Resolver`, applies conditions through conditions.md and moves the
  creature through initiative-and-actions.md.
- **Handoffs out:** `JumpPermitted.MovementCostFeet` for the movement ledger (Q6);
  `JumpLandingCheckRequired` and `JumpObstacleCheckRequired` for an ability check;
  `FlightFalls.Causes` for the fall hazard (distance to `ResolveFall` and
  `FallDistance`; `LandAfterFall` applies Prone); a limited `MovementSpeeds` for
  `UpdateMovementSpeeds` (initiative-and-actions.md FR-116). The hazards spec and
  initiative-and-actions.md each point here for "whether a flier falls" (Q4).
- **Exports:** package `combat` exports the section 5 types once each, with a doc
  comment (FR-055). No test double: there is no port.

## 7. Observability

Silent by design: pure domain functions emit no logs, metrics or spans
(`doc/rules/observability.md` section 3), and no package use case calls them
(shared-types-and-ownership.md FR-044), so none can emit for them. The host
records its own outcomes under its own signals.

1. **What can go wrong?** An invalid number (negative distance, run-up, base Fly
   Speed, or creature height below 1 foot): `*InvalidError`. A negative movement
   left is not one (FR-020). A wrong rule result (off-by-one at the 10-foot
   run-up, rounding, a wrong DC).
2. **How would an operator know?** An `*InvalidError` is recorded on the span of
   the running use case (`doc/rules/observability.md` section 7); wrong results
   are caught by the section 8 tests.
3. **What to diagnose it?** The results carry maximum, approach, distance, cost,
   refusal reason, fall causes, overridden causes, and the requirement's ability,
   skill and DC.
4. **Count or time?** Nothing here; outcome counts belong to the calling host or a
   future use case (combat-action-resolution.md would add an `outcome` value to its
   own signal).
5. **Existing instrumentation still correct?** None exists; `srd5e.d20_test.*`
   is unaffected (a movement check is an ordinary ability check).
6. **What should a consumer monitor?** A wrapper may count refusals by `JumpRefusal`
   and falls by cause; a rise in `ProneRestrictedMovement` or `FlySpeedZero`
   points at stale `MovementSpeeds` or conditions on its side.
7. **Anything noisy?** No, nothing is emitted.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none | | | | |

`doc/observability/signals.md` is unchanged.

## 8. Test plan

Levels: unit (`combat/movement_*_test.go`, package `combat_test`), architecture (`internal/archtest`),
and scenarios through the published packages (`scenarios/`, package `scenarios_test`).

| Id | Level | Notes |
|---|---|---|
| FR-060 | unit + architecture | Cases 4.1b, one test per row; source scan: no literal 10, penalty only from `SpeedPenaltyFeet` |
| FR-001, FR-002, FR-005, FR-057, FR-058 | unit | Cases 4.1a, one test per row; the cap comes from `LimitFeet`; input unchanged (mutants: `min` against `max`, `<` against `<=`) |
| FR-006, FR-059 | architecture | Source scan: `LimitSpeedsByLoad` reads `LimitFeet`, no literal 5 as a cap, no Carrying Capacity or `LoadAssessment` declaration, takes `LoadSpeedLimited` only; name scan for duplicate exported names |
| FR-007 to FR-014 | unit | Exact `JumpKind` list; cases 4.2b and 4.2a, one test per row and column |
| FR-015 to FR-021, FR-056 | unit | Cases 4.2c, one test per row (negative movement left, Prone rows); cost equals distance for both kinds and approaches |
| FR-022 | unit + architecture | Repeat call equal; no `dice.Roller` import in the folder |
| FR-023 to FR-030 | unit | The four kind and terrain combinations; cases 4.3a and 4.3b |
| FR-031, FR-032, FR-035 | unit | Cases 4.4a; `Ability()` equals `Skill().Ability()`; named requirements equal to themselves and pairwise unequal |
| FR-033, FR-034 | unit + architecture | Four named instances, no public constructor; unit: requirement exposes skill, ability (equal to skill.ability) and target (DifficultyClass); signature and field scan: no GM-option parameter, no roll, total or outcome type. S-2 feeds the requirement to `d20.Resolver` |
| FR-036 to FR-043 | unit | Cases 4.5a, one test per row; effective Fly Speed equals `ConditionEffects.speed(...).value` |
| FR-044 to FR-046 | architecture | Hover read only from `HoverAbility`; no movement, turn or duration input; `FlightFalls` has causes only |
| FR-045, FR-047 | unit | Equal facts give equal results; inputs unchanged |
| FR-048 to FR-051 | unit | Cases 4.6a; value list and order |
| FR-052 | architecture | `BurrowPermissionFor` has no speed parameter |
| FR-061 | unit | S-3: Speed 40 and Fly Speed 40, then armor penalty 10 gives walk 30, fly 30; Incapacitated makes the flier fall |
| FR-053 to FR-055 | architecture | Import and purity checks (including the `d20` import), no `string` fields in results, exports present, documented, unique |

Scenarios:

- **S-1:** a Fly Speed 60 creature is Paralyzed by `ConditionSet.Apply`;
  `AssessFlight` returns `FlightFalls([Incapacitated, FlySpeedZero])`; with
  hover it stays aloft overriding both (FR-038, FR-040, FR-042).
- **S-2:** a Strength 10 creature runs 10 feet and asks for a 10-foot Long Jump into
  Difficult Terrain; `JumpPermitted` costs 10 and `JumpLandingFor` returns the DC 10
  Dexterity (Acrobatics) requirement, fed to `d20.Resolver` with a
  `dicetest.FixedRoller` (FR-015, FR-023, FR-034).

- **S-3:** a creature with Speed 40 (Speed 30 plus Large Form) and a Fly Speed of 40 from Draconic Flight, in Chain Mail with Strength 12, gets walk 30, fly 30; Incapacitated makes `AssessFlight` return `FlightFalls([Incapacitated])` (FR-061, FR-060, FR-038).

No port, no contract suite. Mutation testing (`doc/rules/testing.md` section 8):
every changed `combat/movement_*.go` file (`gomu` run only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`),
threshold 80. Mutants that matter: run-up `>= 10` against `> 10`, `<=` against `<`
in the maximum, movement-left and quarter comparisons, halving and its rounding,
the minimum of 0, `3 +`, `* 3 / 2`, `/ 4`, the cause order, the prone
precedence, the clamp of negative movement left. Cases 4.1a to 4.5a supply the
boundary rows. Every FR id appears above.

## 9. Open questions and decisions

Open: none. Each former open item is a decided default plus a requested amendment owned by the named spec (rows Q6, Q8, Q12, Q15, Q17 below); none changes this spec.

Settled:

- Q1: equipment-rules.md owns the table and `LimitFeet`; this spec owns the Speed arithmetic (shared-types-and-ownership.md FR-048).
- Q2: the cap is a decrease applied to every special speed, floored at 0 (FR-002); `WalkOnly` for a host that rules otherwise (FR-057).
- Q3: only mode `Walk` counts as "on foot" (FR-009).
- Q4: `FlightFalls` is the handoff (FR-046); this spec owns "whether a flier falls" (shared-types-and-ownership.md FR-048).
- Q5: both jump kinds give maxima (FR-016, FR-021).
- Q7: a taller obstacle is `JumpObstacleNotCovered` (FR-027).
- Q9: Round Down everywhere, heights are `int` feet within the species range (FR-012, FR-014, FR-026, FR-029).
- Q10: rank 26; depends on `conditions` and `initiative-and-actions`; FR-015 not violated.
- Q11: check frequency is the host's ruling (FR-033).
- Q13: `BurrowMedium` is closed (FR-048); the host maps other media.
- Q14: `DifficultyClass` is `d20`'s, `Skill` is `content`'s (shared-types-and-ownership.md FR-002, FR-005, FR-046).
- Q16: the host recomputes on a load change and sends `UpdateMovementSpeeds` (initiative-and-actions.md FR-116).
- Q6: the host charges a jump as a walk segment of `distance` feet over `Terrain.open` (1 foot per foot, FR-021); no jump mode. Requested amendment, owned by initiative-and-actions.md: an optional fixed-cost segment with no Difficult Terrain extra.
- Q8: this spec stays Strength-only. Requested amendment, owned here in a new FR when class-monk.md FR-042 and class-rogue.md FR-094 are implemented: `jumpAbilityScore` and an integer `distanceMultiplier` (default 1) on `JumpAttempt`.
- Q12: `AssessFlight` is a pure function of facts (FR-045). Requested amendment, owned by encounter-aggregate.md: `Encounter` holds the aloft fact and the landing or death that ends flight (SRD 5.2.1, p. 182).
- Q15: `Varies` and `NoWeight` item weights are totalled by equipment-rules.md (or `character`), never here. Requested amendment owned by equipment-rules.md.
- Q17: this spec keeps its own check shape (section 5). Requested amendment, owned by d20-tests.md: a shared described-check type from `d20`.

## 10. Changelog impact

`CHANGELOG.md`, under Unreleased:

- **Added:** movement rules for the combat context: jump distances and the Long Jump
  landing and obstacle checks, the High Jump reach, the Speed limit while dragging,
  lifting or pushing a heavy load, flight and falling with hover, burrowing through
  solid rock, and the climbing and swimming check requirements the SRD names.

Not **BREAKING**: all additions; no signal is added or renamed.
