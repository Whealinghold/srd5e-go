# Class: Monk

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `ability-scores.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `d20-tests.md`, `derived-stats.md`, `character-aggregate.md`, `rests-and-hit-dice.md`, `level-advancement-and-multiclassing.md`, `character-creation.md`, `content-classes.md`, `content-equipment.md`, `content-source-port.md`, `content-schema-and-decoder.md`, `errors-and-results.md`, `observability-ports.md` |

The `combat` specs and `feats-behaviour.md` are named in prose only; this spec has no dependency on
them (`shared-types-and-ownership.md` FR-013, FR-016, FR-056).

## 1. Purpose and scope

Specifies the Monk as SRD 5.2.1 presents it. Class and subclass **data** (Core Monk Traits, Monk
Features table, Warrior of the Open Hand) is content-classes.md's (`shared-types-and-ownership.md`
FR-025); this spec owns **behaviour only**: the numbers and eligibility rules (pure functions), the
tracked resources (Focus Points, Uncanny Metabolism, Wholeness of Body uses, Quivering Palm
vibrations), and the few use cases that need dice or content (decisions FR-023). State in, new state
and events out (`doc/rules/architecture.md` §4): each function takes the `Character` (Monk state
lives in its `FeatureStates`) and returns the new `Character` plus events. The Monk has no
Spellcasting feature (SRD 5.2.1, p. 49).

**Out of scope** (owned elsewhere; inputs or typed outcomes handed to `combat`, named only): Unarmed
Strike damage, Grapple and Shove, Opportunity Attacks, Dash, Disengage and Dodge, Initiative, rest
duration, falling damage, Total Cover, Resistance, Temporary Hit Points, halving, Hit Point caps,
save resolution (d20-tests.md) and the conditions; level-up, multiclass prerequisites and the choice
of a subclass (level-advancement-and-multiclassing.md); feats (feats-behaviour.md); weapon, armor,
tool and pack definitions and resolving starting-equipment ids (content equipment specs,
`CreateCharacter`); elapsed time (FR-116); Armor Class sources and arithmetic (derived-stats.md
FR-036, FR-037). Extra Attack stacking and multiclass Armor Class interactions are not modelled.
Warrior of the Open Hand is the SRD's only subclass (p. 52); no core Monk content missing from the
PDF was identified.

## 2. Source

SRD 5.2.1, "Monk", "Core Monk Traits", "Monk Features", "Warrior of the Open Hand", pp. 49-52;
section and page per requirement.

## 3. Vocabulary

Feature names are the SRD's; only terms that need a type or a decision are listed.

| SRD term | Type | Notes |
|---|---|---|
| Monk level | `MonkLevel` | 1-20; derived from the `ClassLevels` entry for `ClassId('monk')` (derived-stats.md) |
| Monk weapons | predicate over content-classes.md `WeaponTraining` and content-equipment.md facts | Matches the Monk's `weaponTraining` (Simple; Martial requiring Light, content-classes.md FR-021) and is Melee (the Melee restriction is Martial Arts' own, p. 50). No `WeaponSelector` exists |
| Unarmored Defense | `AbilityBased` basis (derived-stats.md) | 10 + Dexterity + Wisdom, `shieldAllowed: false`, `FeatureId` of content entry `monk_unarmored_defense` |
| save DC (Monk's Focus) | `DifficultyClass` (d20-tests.md) | 8 + Wisdom modifier + Proficiency Bonus (p. 50: "The save DC"); the bonus is the character's `ProficiencyBonus` (total level), not a Monk-table value |
| target's saving throws outcome | `D20TestOutcome` | `success` or `failure`, a command input (FR-141); no Monk-local type |
| Superior Defense | `SuperiorDefenseEffect` | Resists every `DamageType` except `force` (FR-102) |
| damage types, conditions, sizes, abilities | `DamageType`, `ConditionKind`, `CreatureSize` (content-schema-and-decoder.md FR-109), `Ability`, `AbilityModifier` (ability-scores.md) | Closed vocabularies owned elsewhere (decisions FR-005, FR-006, FR-053); read, never defined here, so there is no Monk-local size or ability type |

**Go conventions in this spec** (binding vocabulary: errors-and-results.md,
shared-types-and-ownership.md). Function and method names written lowerCamelCase
below (`rowAt`, `activateRage`) are exported PascalCase in Go (`RowAt`,
`ActivateRage`); enum-like values written lowerCamelCase (`notABarbarian`) are typed
constants with a `String()` method. A "closed" result or choice is an interface with
an unexported marker method, one struct per variant, handled by an exhaustive type
switch; the variant list is in its doc comment. Untrusted input is validated by
`NewX` / `ParseX`, which return `(T, error)` with a typed, `errors.As`-able error and
report every problem at once with `errors.Join`; a port failure is a returned
`*srderr.PortError` (`errors.Is` / `errors.As`); a panic carrying an error value is
reserved for programmer errors (violated precondition) and appears only in `MustX`
helpers and unreachable branches. A function that may do I/O or emit spans takes
`context.Context` first; pure rule functions do not.

## 4. Requirements

Stable ids; never renumber. "Result" means a closed result (`doc/rules/architecture.md` §6); a rule
rejection is never an error, and a constructor given an invalid argument panics with an error value.
Per `shared-types-and-ownership.md` FR-017, FR-018, `MonkRejected(reason)` is shorthand for the
operation's own `<Operation>Rejected` variant carrying the `MonkRejection`, and `ContentMissing` for
its `<Operation>ContentMissing` variant carrying `MissingContent(kind, id)`; neither is an exported
name. Only `ValidateMonkCreationChoices` carries another reason type (`MonkChoiceProblem`, FR-139).

### 4.1 Creation choices

Core Monk Traits data is content-classes.md's; the choices are checked by
`ValidateMonkCreationChoices` (FR-139, FR-143).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-004 | A Monk character MUST be creatable with exactly 2 Skill Proficiencies chosen from Acrobatics, Athletics, History, Insight, Religion and Stealth (the list is `SkillChoice` data in content-classes.md; this is the choice rule). | SRD 5.2.1, "Core Monk Traits", p. 49 | happy |
| FR-005 | For a Level 1 Monk (FR-139), a skill choice with fewer or more than 2 skills, a skill outside the six, or the same skill twice MUST yield `ValidateMonkCreationChoicesRejected` carrying `MonkChoiceProblem` (`skillCount`, `skillNotOnList` or `skillDuplicate`), and MUST NOT yield a partial choice. | SRD 5.2.1, "Core Monk Traits", p. 49 | unhappy (invalid input) |
| FR-007 | A Monk character MUST be creatable with exactly one Tool Proficiency, chosen as one type of Artisan's Tools or Musical Instrument. | SRD 5.2.1, "Core Monk Traits", p. 49 | happy |
| FR-008 | For a Level 1 Monk (FR-139), a tool choice outside the categories of the `ClassDefinition`'s `ToolTraining`, or naming no tool, MUST yield `ValidateMonkCreationChoicesRejected` carrying `toolNotAllowed` or `toolMissing`. | SRD 5.2.1, "Core Monk Traits", p. 49 | unhappy (invalid input) |
| FR-012 | For a Level 1 Monk (FR-139), choosing neither starting-equipment option, both, or one other than A or B MUST yield `ValidateMonkCreationChoicesRejected` carrying `equipmentOption`; the valid options are those of the `ClassDefinition`. | SRD 5.2.1, "Core Monk Traits", p. 49 | unhappy (invalid input) |
| FR-014 | A Monk taken as a Level 1 Character (`MonkEntry.level1`, FR-139) MUST gain all the Core Monk Traits and the Monk's level 1 features. | SRD 5.2.1, "As a Level 1 Character", p. 49 | happy |
| FR-015 | A Monk taken as a Multiclass Character (`MonkEntry.multiclass`, FR-139) MUST gain only the Hit Point Die trait plus the level 1 features; it MUST NOT gain the Primary Ability, saving throws, skill, weapon, tool or armor proficiencies, or Starting Equipment. | SRD 5.2.1, "As a Multiclass Character", p. 49 | unhappy (rule interaction) |

### 4.2 Monk Features table

The 20-row table (SRD 5.2.1, "Monk Features", p. 50) is content-classes.md data (`martial_arts`,
`focus_points`, `unarmored_movement` columns, FR-040); the behaviour reads one row as a
`MonkLevelProfile` (FR-152). Tests build fixtures from the SRD rows under `testdata/srd/`
(decisions FR-033), one per level.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-016 | `MonkLevel` MUST reject 0, negative values and 21 or more with panic, and `NewMonkLevel` MUST return a result for the same inputs. | SRD 5.2.1, "Monk Features", p. 50 | unhappy (boundary) |
| FR-018 | The Proficiency Bonus used by every Monk calculation (FR-037) MUST be the character's `ProficiencyBonus` from derived-stats.md, based on total character level, and MUST NOT be read from the Monk row. | SRD 5.2.1, "Monk Features", p. 50 | happy |
| FR-023 | A feature use requested for a `Character` whose `ClassLevels` has no entry for `ClassId('monk')` MUST yield `MonkRejected(notAMonk)` and MUST NOT change state. A `Character` with a Monk level but no stored `MonkState` is a Monk: its state reads as the initial `MonkState` (FR-150), so `notAMonk` never fires for a legitimate multiclass Monk. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |

### 4.3 Martial Arts and Unarmored Defense

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-024 | A weapon MUST count as a Monk weapon if it is a Simple Melee weapon, or a Martial Melee weapon that has the Light property. | SRD 5.2.1, "Level 1: Martial Arts", p. 50 | happy |
| FR-025 | A Simple or Martial Ranged weapon, and a Martial Melee weapon without the Light property, MUST NOT count as a Monk weapon. | SRD 5.2.1, "Level 1: Martial Arts", p. 50 | unhappy (rule interaction) |
| FR-026 | The Martial Arts benefits (FR-027 to FR-030) MUST apply only while the Monk is unarmed or wielding only Monk weapons, is not wearing armor, and is not wielding a Shield (table 4.3a). | SRD 5.2.1, "Level 1: Martial Arts", p. 50 | unhappy (rule interaction) |
| FR-027 | While FR-026 holds, the Monk MUST be offered a Bonus Unarmed Strike as a Bonus Action; otherwise the offer MUST be absent. | SRD 5.2.1, "Level 1: Martial Arts", p. 50 | happy |
| FR-028 | While FR-026 holds, an Unarmed Strike or Monk weapon attack MUST offer the Martial Arts die as a damage die in place of the normal damage die, as an option the player may take or leave; the engine MUST NOT force it. | SRD 5.2.1, "Level 1: Martial Arts", p. 50 | happy |
| FR-029 | While FR-026 holds, the attack roll and damage roll of an Unarmed Strike or Monk weapon MAY use the Dexterity modifier instead of the Strength modifier. | SRD 5.2.1, "Level 1: Martial Arts", p. 50 | happy |
| FR-030 | While FR-026 holds, the save DC of the Unarmed Strike's Grapple or Shove option MAY use the Dexterity modifier instead of the Strength modifier. | SRD 5.2.1, "Level 1: Martial Arts", p. 50 | happy |
| FR-031 | From Monk level 1, `MonkRules.unarmoredDefenseBasis(FeatureId)` MUST return derived-stats.md's `AbilityBased` basis: base 10 plus Dexterity (intrinsic) plus the named ability set `{Wisdom}`, with the supplied `FeatureId` (content entry `monk_unarmored_defense`, not hard-coded). The host supplies it to `CalculateDerivedStats` in `alternativeBases` (derived-stats.md FR-036). This spec does not compute the Armor Class. | SRD 5.2.1, "Level 1: Unarmored Defense", p. 50; derived-stats.md FR-036 | happy |
| FR-032 | The declaration MUST have `shieldAllowed: false`, so derived-stats.md offers no Monk base while a Shield is wielded, and none while armor is worn (one test per fact: the named ability set is exactly `{Wisdom}`, `shieldAllowed` is false, the supplied `FeatureId` is carried, and the declaration is the same at every Monk level 1 to 20; the arithmetic, e.g. Dexterity +3, Wisdom +3 giving 16, belongs to derived-stats.md). | SRD 5.2.1, "Level 1: Unarmored Defense", p. 50; derived-stats.md FR-036, FR-037 | unhappy (rule interaction) |

**Table 4.3a. Martial Arts conditions (FR-026), one test per row.** SRD 5.2.1, "Level 1: Martial Arts", p. 50.

| Case | Held items | Armor worn | Shield | Benefits apply | Path |
|---|---|---|---|---|---|
| unarmed | none | no | no | yes | happy |
| one Monk weapon | one Monk weapon | no | no | yes | happy |
| two Monk weapons | two Monk weapons | no | no | yes | happy |
| Monk weapon and other weapon | one Monk weapon, one non-Monk weapon | no | no | no | unhappy (rule interaction) |
| non-Monk weapon only | one non-Monk weapon | no | no | no | unhappy (rule interaction) |
| armor | none | yes | no | no | unhappy (rule interaction) |
| Shield | none | no | yes | no | unhappy (rule interaction) |
| Monk weapon and Shield | one Monk weapon | no | yes | no | unhappy (rule interaction) |
| armor and Monk weapon | one Monk weapon | yes | no | no | unhappy (rule interaction) |

### 4.4 Monk's Focus (Focus Points)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-033 | A Monk of level 1 MUST have no Focus Points (maximum 0); the maximum from level 2 is the Focus Points column of the `MonkLevelProfile` (the former FR-020 is withdrawn). | SRD 5.2.1, "Level 2: Monk's Focus", p. 50 | happy |
| FR-034 | Expending Focus Points MUST reduce the available points by the cost and add the cost to the expended points; spending exactly the remaining points MUST succeed and leave 0 (table 4.4a). | SRD 5.2.1, "Level 2: Monk's Focus", p. 50 | unhappy (boundary) |
| FR-035 | Expending more Focus Points than are available MUST yield `MonkRejected(notEnoughFocusPoints)` and MUST NOT change state, make a roll or emit a spend event. | SRD 5.2.1, "Level 2: Monk's Focus", p. 50 | unhappy (state error) |
| FR-036 | Expended Focus Points MUST stay unavailable until the Monk finishes a Short Rest or a Long Rest, at the end of which all expended points are regained. | SRD 5.2.1, "Level 2: Monk's Focus", p. 50 | happy |
| FR-037 | The save DC (Monk's Focus) MUST be 8 plus the Wisdom modifier plus the character's Proficiency Bonus, based on total character level (FR-018; table 4.4b). | SRD 5.2.1, "Level 2: Monk's Focus", p. 50 | happy |
| FR-038 | Flurry of Blows MUST cost 1 Focus Point and grant two Unarmed Strikes as a Bonus Action. | SRD 5.2.1, "Level 2: Monk's Focus", p. 50 | happy |
| FR-039 | Patient Defense MUST offer the Disengage action as a Bonus Action at no Focus Point cost. | SRD 5.2.1, "Level 2: Monk's Focus", pp. 50-51 | happy |
| FR-040 | Patient Defense MUST, for 1 Focus Point, offer both the Disengage and the Dodge actions as a Bonus Action. | SRD 5.2.1, "Level 2: Monk's Focus", p. 51 | happy |
| FR-041 | Step of the Wind MUST offer the Dash action as a Bonus Action at no Focus Point cost. | SRD 5.2.1, "Level 2: Monk's Focus", p. 51 | happy |
| FR-042 | Step of the Wind MUST, for 1 Focus Point, offer both the Disengage and the Dash actions as a Bonus Action and double the jump distance for the turn. | SRD 5.2.1, "Level 2: Monk's Focus", p. 51 | happy |
| FR-043 | A Focus feature, or any Focus Point spend, requested below Monk level 2 MUST yield `MonkRejected(levelTooLow)` and MUST NOT change state. | SRD 5.2.1, "Monk Features", p. 50 | unhappy (boundary) |
| FR-136 | `FocusPoints` MUST reject a spend with a cost below 1 with panic; a zero or negative cost is a caller bug, not a rule rejection (table 4.4a). | `doc/rules/testing.md` §4 (invalid input); SRD 5.2.1, "Level 2: Monk's Focus", p. 50 (costs are whole points); engine decision, section 9 | unhappy (invalid input) |

**Table 4.4a. Spending Focus Points (FR-034, FR-035, FR-136), one test per row.** SRD 5.2.1, "Level 2: Monk's Focus", p. 50.

| Case | Available | Cost | Result | Path |
|---|---|---|---|---|
| spend one of several | 2 | 1 | success, 1 left | happy |
| spend exactly all | 1 | 1 | success, 0 left | unhappy (boundary) |
| one short | 0 | 1 | `notEnoughFocusPoints` | unhappy (boundary) |
| cost above available | 2 | 3 | `notEnoughFocusPoints`, state unchanged | unhappy (state error) |
| zero cost | 2 | 0 | panic | unhappy (invalid input) |

**Table 4.4b. Save DC (Monk's Focus) (FR-037), one test per row.** SRD 5.2.1, "Level 2: Monk's Focus", p. 50.

| Case | Wisdom modifier | Proficiency Bonus | DC | Path |
|---|---|---|---|---|
| typical | +3 | +2 | 13 | happy |
| zero modifier | 0 | +2 | 10 | unhappy (boundary) |
| negative modifier | -1 | +2 | 9 | unhappy (boundary) |
| mid level | +5 | +4 | 17 | happy |
| top level | +5 | +6 | 19 | unhappy (boundary) |
| multiclass, total level above Monk level (Monk 5, Fighter 5, total level 10) | +3 | +4 (total level), not the +3 of Monk row 5 | 15 | unhappy (rule interaction) |

### 4.5 Unarmored Movement and Uncanny Metabolism

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-044 | From Monk level 2, while not wearing armor and not wielding a Shield, the Monk's speed MUST increase by the Unarmored Movement bonus of the `MonkLevelProfile` (the former FR-021 is withdrawn), as a `SpeedBonus` in feet for the host to add to the speed the other specs compute. | SRD 5.2.1, "Level 2: Unarmored Movement", p. 51 | happy |
| FR-045 | While wearing armor or wielding a Shield, or at Monk level 1, the Unarmored Movement bonus MUST be 0 feet. | SRD 5.2.1, "Level 2: Unarmored Movement", p. 51 | unhappy (rule interaction) |
| FR-046 | Uncanny Metabolism, used when the Monk rolls Initiative, MUST regain all expended Focus Points, roll the Martial Arts die, and regain Hit Points equal to the Monk level plus the number rolled (table 4.5a); the result carries the amount and the host applies it (the cap is hit-points-and-damage.md). | SRD 5.2.1, "Level 2: Uncanny Metabolism", p. 51 | happy |
| FR-047 | The roll MUST use the Martial Arts die of the `MonkLevelProfile` for the Monk's current level, rolled by `UseUncannyMetabolism` through the `dice.Roller` port. | SRD 5.2.1, "Level 2: Uncanny Metabolism", p. 51 | happy |
| FR-048 | A second use of Uncanny Metabolism before a Long Rest MUST yield `MonkRejected(alreadyUsed)` and MUST NOT roll or change state. | SRD 5.2.1, "Level 2: Uncanny Metabolism", p. 51 | unhappy (state error) |
| FR-049 | Finishing a Long Rest MUST make Uncanny Metabolism usable again; finishing a Short Rest MUST NOT. | SRD 5.2.1, "Level 2: Uncanny Metabolism", p. 51 | unhappy (state error) |
| FR-050 | Uncanny Metabolism MUST be accepted only when the command reports that the Monk rolled Initiative; any other trigger MUST yield `MonkRejected(notEligible)`. | SRD 5.2.1, "Level 2: Uncanny Metabolism", p. 51 | unhappy (state error) |

**Table 4.5a. Uncanny Metabolism Hit Points regained (FR-046, FR-047), one test per row.** SRD 5.2.1, "Level 2: Uncanny Metabolism", p. 51. The Hit Point maximum cap is hit-points-and-damage.md's.

| Case | Monk level | Martial Arts die | Roll | Hit Points regained | Path |
|---|---|---|---|---|---|
| lowest level, lowest roll | 2 | d6 | 1 | 3 | unhappy (boundary) |
| lowest level, highest roll | 2 | d6 | 6 | 8 | unhappy (boundary) |
| d8 tier | 5 | d8 | 8 | 13 | happy |
| d10 tier | 11 | d10 | 10 | 21 | happy |
| top level, highest roll | 20 | d12 | 12 | 32 | unhappy (boundary) |

### 4.6 Deflect Attacks and Deflect Energy

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-051 | From Monk level 3, when an attack roll hits the Monk and its damage includes Bludgeoning, Piercing or Slashing damage, the Monk MUST be offered a Reaction to reduce the attack's total damage; this holds when the damage also includes other types. | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 | happy |
| FR-052 | Below Monk level 13, an attack whose damage includes none of Bludgeoning, Piercing or Slashing MUST yield `MonkRejected(notEligible)` and MUST NOT roll. | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 | unhappy (rule interaction) |
| FR-053 | The reduction MUST equal 1d10 plus the Dexterity modifier plus the Monk level, with the d10 rolled through the `dice.Roller` port (table 4.6a). | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 | happy |
| FR-054 | The attack's total damage after reduction MUST NOT go below 0, including when the reduction exceeds the damage and when it equals it exactly. | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 | unhappy (boundary) |
| FR-055 | A reduction that evaluates below 0 (a low Dexterity modifier making the sum negative) MUST be treated as 0 and MUST NOT raise the damage. | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 (the feature reduces damage); engine decision, section 9 | unhappy (boundary) |
| FR-056 | When the reduction leaves 0 damage, the Monk MUST be offered to expend 1 Focus Point to redirect some of the attack's force. | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 | happy |
| FR-057 | A redirect request when the damage after reduction is above 0 MUST yield `MonkRejected(notEligible)` and MUST NOT roll or spend Focus Points. | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 | unhappy (state error) |
| FR-058 | A redirect request with no Focus Point available MUST yield `MonkRejected(notEnoughFocusPoints)` and MUST NOT roll; the earlier reduction MUST stand. | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 | unhappy (state error) |
| FR-059 | The redirect target MUST be a creature within 5 feet of the Monk that the Monk can see if the attack was melee, or a creature within 60 feet that the Monk can see and that is not behind Total Cover if the attack was ranged; a target outside it MUST yield `MonkRejected(notEligible)`. | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 | unhappy (boundary) |
| FR-060 | The redirected creature MUST make a Dexterity saving throw against the save DC (Monk's Focus) and take damage equal to two rolls of the Martial Arts die plus the Dexterity modifier on a failed save, both dice rolled through the `dice.Roller` port; on a success it takes no damage. The save outcome is a command input (FR-141); the result MUST carry the DC used, the damage on a failed save (table 4.6b) and the damage applied. The DC is the Monk's Focus save DC: p. 50, "Some features that use Focus Points require your target to make a saving throw. The save DC equals 8 plus your Wisdom modifier and Proficiency Bonus". | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51; "Level 2: Monk's Focus", p. 50 | happy |
| FR-061 | The redirected damage MUST be the same type as the damage the attack dealt. | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 | happy |
| FR-062 | From Monk level 13, Deflect Attacks MUST also apply to attacks whose damage includes none of Bludgeoning, Piercing or Slashing, for any damage type (table 4.6a). | SRD 5.2.1, "Level 13: Deflect Energy", p. 52 | happy |
| FR-063 | Below Monk level 3, Deflect Attacks MUST yield `MonkRejected(levelTooLow)`. | SRD 5.2.1, "Monk Features", p. 50 | unhappy (boundary) |
| FR-137 | The redirected damage (FR-060) MUST be clamped at 0 when two dice plus a negative Dexterity modifier sum below 0 (table 4.6b). | SRD 5.2.1, "Level 3: Deflect Attacks", p. 51 (silent on a negative total); engine decision, section 9 | unhappy (boundary) |

**Table 4.6a. Deflect Attacks reduction (FR-051 to FR-055, FR-062), one test per row.** SRD 5.2.1, "Level 3: Deflect Attacks", p. 51; "Level 13: Deflect Energy", p. 52. Halving and rounding are not involved.

| Case | Monk level | Dexterity modifier | d10 roll | Incoming total | Damage types | Reduction | Damage after | Path |
|---|---|---|---|---|---|---|---|---|
| low roll | 3 | +3 | 1 | 20 | Bludgeoning | 7 | 13 | happy |
| high roll | 3 | +3 | 10 | 20 | Piercing | 16 | 4 | happy |
| reduction equals damage | 3 | +3 | 10 | 16 | Slashing | 16 | 0 | unhappy (boundary) |
| reduction one below damage | 3 | +3 | 10 | 17 | Slashing | 16 | 1 | unhappy (boundary) |
| reduction exceeds damage | 3 | +3 | 10 | 10 | Slashing | 16 | 0 | unhappy (boundary) |
| higher level | 10 | +4 | 5 | 30 | Bludgeoning | 19 | 11 | happy |
| negative sum | 3 | -5 | 1 | 10 | Bludgeoning | 0 | 10 | unhappy (boundary) |
| mixed types | 3 | +3 | 1 | 20 | Fire and Slashing | 7 | 13 | unhappy (rule interaction) |
| Fire only, level 3 | 3 | +3 | 5 | 10 | Fire | not eligible | 10 | unhappy (rule interaction) |
| Fire only, level 12 | 12 | +3 | 5 | 10 | Fire | not eligible | 10 | unhappy (boundary) |
| Fire only, level 13 | 13 | +3 | 5 | 30 | Fire | 21 | 9 | unhappy (boundary) |
| level 2 | 2 | +3 | 5 | 10 | Slashing | not available | 10 | unhappy (boundary) |

**Table 4.6b. Redirected damage on a failed save (FR-060, FR-061, FR-137), one test per row.** SRD 5.2.1, "Level 3: Deflect Attacks", p. 51.

| Case | Monk level | Martial Arts die | Rolls | Dexterity modifier | Damage on failed save | Path |
|---|---|---|---|---|---|---|
| lowest rolls | 3 | d6 | 1, 1 | +3 | 5 | unhappy (boundary) |
| highest rolls | 3 | d6 | 6, 6 | +3 | 15 | unhappy (boundary) |
| d8 tier | 5 | d8 | 8, 8 | +2 | 18 | happy |
| negative modifier | 3 | d6 | 1, 1 | -1 | 1 | unhappy (boundary) |
| negative total | 3 | d6 | 1, 1 | -5 | 0 (never negative, FR-137) | unhappy (boundary) |

### 4.7 Monk Subclass

Choosing a subclass (`SubclassChosen`, `SubclassRejected`) is
level-advancement-and-multiclassing.md's (decisions FR-020); the chosen subclass id is an explicit
input of the subclass operations (FR-148), never stored in `MonkState`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-068 | The character MUST have each feature of the chosen subclass whose level is at or below the Monk level, and none above it: Open Hand Technique at 3, Wholeness of Body at 6, Fleet Step at 11, Quivering Palm at 17; a use below its level MUST yield `MonkRejected(levelTooLow)`. A missing or different subclass is `notEligible` (FR-148). | SRD 5.2.1, "Level 3: Monk Subclass", p. 51; "Monk Subclass: Warrior of the Open Hand", p. 52 | unhappy (boundary) |

### 4.8 Level 4 to 9 class features

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-072 | From Monk level 4, Slow Fall MUST offer a Reaction when the Monk falls, reducing the fall damage by 5 times the Monk level (table 4.8a). | SRD 5.2.1, "Level 4: Slow Fall", p. 51 | happy |
| FR-073 | Fall damage after Slow Fall MUST NOT go below 0, including when the reduction equals the damage exactly. | SRD 5.2.1, "Level 4: Slow Fall", p. 51 | unhappy (boundary) |
| FR-074 | Slow Fall below Monk level 4 MUST yield `MonkRejected(levelTooLow)`. | SRD 5.2.1, "Monk Features", p. 50 | unhappy (boundary) |
| FR-075 | From Monk level 5, the Attack action MUST allow two attacks instead of one; below level 5 it MUST allow the one attack. | SRD 5.2.1, "Level 5: Extra Attack", p. 51 | unhappy (boundary) |
| FR-076 | From Monk level 5, once per turn when the Monk hits a creature with a Monk weapon or an Unarmed Strike, the Monk MUST be offered to expend 1 Focus Point to attempt a stunning strike, and the target MUST make a Constitution saving throw against the save DC (Monk's Focus), whose outcome is a command input (FR-141). | SRD 5.2.1, "Level 5: Stunning Strike", p. 51 | happy |
| FR-077 | On a failed Stunning Strike save, the result MUST carry a `StunApplied(condition: ConditionKind.stunned, until: MonkEffectEnd.startOfMonkNextTurn)` effect (section 5). | SRD 5.2.1, "Level 5: Stunning Strike", p. 51 | happy |
| FR-078 | On a successful save, the result MUST carry `StunResisted(speedHalved: true, advantageOnNextAttack: true, until: MonkEffectEnd.startOfMonkNextTurn)`: the target's Speed halved until the start of the Monk's next turn, and Advantage for the next attack roll made against it before then. | SRD 5.2.1, "Level 5: Stunning Strike", p. 51 | happy |
| FR-079 | A second Stunning Strike in the same turn MUST yield `MonkRejected(alreadyUsed)` and MUST NOT spend Focus Points. | SRD 5.2.1, "Level 5: Stunning Strike", p. 51 | unhappy (state error) |
| FR-080 | Stunning Strike with no Focus Point available MUST yield `MonkRejected(notEnoughFocusPoints)`. | SRD 5.2.1, "Level 5: Stunning Strike", p. 51 | unhappy (state error) |
| FR-081 | Stunning Strike on a hit made with a non-Monk weapon, or on a miss, MUST yield `MonkRejected(notEligible)` and MUST NOT spend Focus Points. | SRD 5.2.1, "Level 5: Stunning Strike", p. 51 | unhappy (rule interaction) |
| FR-082 | From Monk level 6, damage dealt with the Monk's Unarmed Strike MUST offer a choice of Force damage or the strike's normal damage type; below level 6 only the normal type MUST be offered. | SRD 5.2.1, "Level 6: Empowered Strikes", p. 51 | unhappy (boundary) |
| FR-083 | From Monk level 7, when subjected to an effect that allows a Dexterity saving throw for half damage, a Monk who is not Incapacitated MUST take no damage on a success and half damage on a failure (table 4.8b). | SRD 5.2.1, "Level 7: Evasion", p. 51 | happy |
| FR-084 | A Monk with the Incapacitated condition MUST NOT benefit from Evasion; the effect's own result MUST stand unchanged. | SRD 5.2.1, "Level 7: Evasion", p. 51 | unhappy (rule interaction) |
| FR-085 | Evasion MUST apply only to effects that allow a Dexterity saving throw to take half damage; a Constitution or other save, or a Dexterity save with no half-damage clause, MUST be unchanged. | SRD 5.2.1, "Level 7: Evasion", p. 51 | unhappy (rule interaction) |
| FR-086 | From Monk level 9, while not wearing armor and not wielding a Shield, the Monk MUST be granted movement along vertical surfaces and across liquids on the Monk's turn without falling during the movement; with armor or a Shield, or below level 9, it MUST NOT be granted. | SRD 5.2.1, "Level 9: Acrobatic Movement", p. 51 | unhappy (rule interaction) |

**Table 4.8a. Slow Fall (FR-072, FR-073), one test per row.** SRD 5.2.1, "Level 4: Slow Fall", p. 51. Fall damage is supplied by the caller.

| Case | Monk level | Fall damage | Reduction | Damage after | Path |
|---|---|---|---|---|---|
| lowest level | 4 | 30 | 20 | 10 | unhappy (boundary) |
| reduction equals damage | 4 | 20 | 20 | 0 | unhappy (boundary) |
| reduction exceeds damage | 4 | 10 | 20 | 0 | unhappy (boundary) |
| zero fall damage | 4 | 0 | 20 | 0 | unhappy (boundary) |
| top level | 20 | 120 | 100 | 20 | unhappy (boundary) |
| below level 4 | 3 | 30 | not available | 30 | unhappy (boundary) |

**Table 4.8b. Evasion (FR-083 to FR-085), one test per row.** SRD 5.2.1, "Level 7: Evasion", p. 51. Rounding of half damage is hit-points-and-damage.md's, so rows use even damage. "Effect's own result" is `ApplyEvasionUnchanged` (a valid outcome, not a rejection).

| Case | Monk level | Save | Half damage on success clause | Incapacitated | Outcome | Path |
|---|---|---|---|---|---|---|
| success | 7 | Dexterity, success | yes | no | 0 damage | happy |
| failure | 7 | Dexterity, failure | yes | no | 10 of 20 | happy |
| success or failure, Incapacitated | 7 | Dexterity, success or failure | yes | yes | effect's own result | unhappy (rule interaction) |
| other save | 7 | Constitution, success | yes | no | effect's own result | unhappy (rule interaction) |
| no half clause | 7 | Dexterity, success | no | no | effect's own result | unhappy (rule interaction) |
| below level 7 | 6 | Dexterity, success | yes | no | `ApplyEvasionRejected(levelTooLow)`; the effect's own result stands | unhappy (boundary) |

### 4.9 Level 10 to 20 class features

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-087 | From Monk level 10, Flurry of Blows MUST allow three Unarmed Strikes instead of two for the same 1 Focus Point. | SRD 5.2.1, "Level 10: Heightened Focus", pp. 51-52 | happy |
| FR-088 | From Monk level 10, Patient Defense with 1 Focus Point MUST also grant Temporary Hit Points equal to two rolls of the Martial Arts die, rolled through the `dice.Roller` port. | SRD 5.2.1, "Level 10: Heightened Focus", pp. 51-52 | happy |
| FR-089 | From Monk level 10, Step of the Wind with 1 Focus Point MUST allow choosing a willing creature within 5 feet whose `CreatureSize` is `size <= CreatureSize.large` by declaration order (Large or smaller) and moving it with the Monk until the end of the turn, and that creature's movement MUST NOT provoke Opportunity Attacks; a creature whose `CreatureSize` is above `large` or not willing MUST yield `MonkRejected(notEligible)`. | SRD 5.2.1, "Level 10: Heightened Focus", pp. 51-52 | happy |
| FR-090 | Below Monk level 10, Flurry of Blows MUST give two strikes and Patient Defense and Step of the Wind MUST give none of the FR-088 and FR-089 benefits. | SRD 5.2.1, "Level 10: Heightened Focus", pp. 51-52 | unhappy (boundary) |
| FR-091 | From Monk level 10, at the end of each of the Monk's turns the Monk MUST be offered to remove from self one of the Charmed, Frightened or Poisoned conditions. | SRD 5.2.1, "Level 10: Self-Restoration", p. 52 | happy |
| FR-092 | Self-Restoration MUST NOT remove any other condition, and MUST NOT remove more than one condition per turn end. | SRD 5.2.1, "Level 10: Self-Restoration", p. 52 | unhappy (rule interaction) |
| FR-093 | From Monk level 10, forgoing food and drink MUST NOT give the Monk levels of Exhaustion. | SRD 5.2.1, "Level 10: Self-Restoration", p. 52 | happy |
| FR-094 | From Monk level 14, the Monk MUST have proficiency in all saving throws, so each saving throw modifier includes the Proficiency Bonus; below level 14 the Monk MUST gain no save proficiency from this feature (a Level 1 Monk has Strength and Dexterity only; a multiclass Monk gains none from this class, FR-015). | SRD 5.2.1, "Level 14: Disciplined Survivor", p. 52 | unhappy (boundary) |
| FR-095 | From Monk level 14, when the Monk makes a saving throw and fails, the Monk MUST be offered to expend 1 Focus Point to reroll it; the operation MUST spend the point and return `UseDisciplinedSurvivorAuthorised` (the new `Character` and the `DisciplinedSurvivorUsed` event; no other field). The host then rolls the new d20 and calls `Roll.applyReroll(RerollDie(i), newValue: n)` (d20-tests.md FR-012, FR-059; decisions FR-002, FR-010), which takes no authorisation input and uses the new value even when lower, so "must use the new roll" is guaranteed there. | SRD 5.2.1, "Level 14: Disciplined Survivor", p. 52 | happy |
| FR-096 | A reroll request for a saving throw that did not fail, or with no Focus Point available, MUST yield `MonkRejected(notEligible)` or `MonkRejected(notEnoughFocusPoints)` respectively, with no roll and no spend. | SRD 5.2.1, "Level 14: Disciplined Survivor", p. 52 | unhappy (state error) |
| FR-097 | From Monk level 15, when the Monk rolls Initiative and does not use Uncanny Metabolism, Perfect Focus MUST regain expended Focus Points until the Monk has 4 if the Monk has 3 or fewer, and MUST change nothing if the Monk has 4 or more (table 4.9a). | SRD 5.2.1, "Level 15: Perfect Focus", p. 52 | happy |
| FR-098 | Perfect Focus MUST NOT apply when Uncanny Metabolism is used for that Initiative roll. | SRD 5.2.1, "Level 15: Perfect Focus", p. 52 | unhappy (rule interaction) |
| FR-099 | Perfect Focus below Monk level 15 MUST yield `MonkRejected(levelTooLow)`. | SRD 5.2.1, "Monk Features", p. 50 | unhappy (boundary) |
| FR-100 | From Monk level 18, at the start of the Monk's turn the Monk MUST be offered to expend 3 Focus Points to gain, for 1 minute or until the Incapacitated condition, Resistance to all damage except Force damage. | SRD 5.2.1, "Level 18: Superior Defense", p. 52 | happy |
| FR-101 | Superior Defense with fewer than 3 Focus Points available, outside the start of the Monk's turn, or below level 18 MUST yield `MonkRejected` of `notEnoughFocusPoints`, `notEligible` or `levelTooLow` respectively, with no spend. | SRD 5.2.1, "Level 18: Superior Defense", p. 52 | unhappy (boundary) |
| FR-102 | The Superior Defense effect MUST end when the Monk has the Incapacitated condition, and MUST be a `SuperiorDefenseEffect` carrying `duration` of `time.Duration(seconds: 60)` (the SRD's 1 minute) for the host to expire, `endsWhenIncapacitated: true`, and `resistsAllExcept` of exactly `{DamageType.force}`; the host applies the Resistance from this value (hit-points-and-damage.md). A test asserts Force is not in the resisted set. | SRD 5.2.1, "Level 18: Superior Defense", p. 52 | unhappy (state error) |
| FR-103 | At Monk level 20, Body and Mind MUST increase the Dexterity and Wisdom scores by 4 each, to a maximum of 25 (table 4.9b). | SRD 5.2.1, "Level 20: Body and Mind", p. 52 | happy |
| FR-104 | Body and Mind MUST NOT lower a Dexterity or Wisdom score that is already above 25; it leaves that score unchanged. | SRD 5.2.1, "Level 20: Body and Mind", p. 52 (a maximum of 25 for the increase); engine decision, section 9 | unhappy (boundary) |

**Table 4.9a. Perfect Focus (FR-097, FR-098), one test per row.** SRD 5.2.1, "Level 15: Perfect Focus", p. 52.

| Case | Focus Points available | Uncanny Metabolism used | Focus Points after | Path |
|---|---|---|---|---|
| none left | 0 | no | 4 | unhappy (boundary) |
| three left | 3 | no | 4 | unhappy (boundary) |
| exactly four | 4 | no | 4 | unhappy (boundary) |
| five left | 5 | no | 5 | unhappy (boundary) |
| Uncanny Metabolism used | 0 | yes | unchanged by Perfect Focus | unhappy (rule interaction) |

**Table 4.9b. Body and Mind (FR-103, FR-104), one test per row.** SRD 5.2.1, "Level 20: Body and Mind", p. 52.

| Case | Dexterity before | Wisdom before | Dexterity after | Wisdom after | Path |
|---|---|---|---|---|---|
| well below cap | 16 | 14 | 20 | 18 | happy |
| exactly reaches cap | 21 | 21 | 25 | 25 | unhappy (boundary) |
| already at cap | 25 | 25 | 25 | 25 | unhappy (boundary) |
| one score capped | 22 | 18 | 25 | 22 | unhappy (boundary) |
| above cap | 27 | 20 | 27 | 24 | unhappy (boundary) |

### 4.10 Warrior of the Open Hand

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-105 | From Monk level 3 in this subclass, whenever the Monk hits a creature with an attack granted by Flurry of Blows, the Monk MUST be offered to impose one Open Hand Technique effect on that target; a hit by an Unarmed Strike not granted by Flurry of Blows MUST NOT offer it. | SRD 5.2.1, "Level 3: Open Hand Technique", p. 52 | unhappy (rule interaction) |
| FR-106 | The Addle effect MUST prevent the target from making Opportunity Attacks until the start of its next turn. | SRD 5.2.1, "Level 3: Open Hand Technique", p. 52 | happy |
| FR-107 | The Push effect MUST require the target to make a Strength saving throw against the save DC (Monk's Focus) (interpretation, section 9) and, on a failure, be pushed up to 15 feet away from the Monk; the DC used MUST be recorded in the result. | SRD 5.2.1, "Level 3: Open Hand Technique", p. 52 | happy |
| FR-108 | The Topple effect MUST require the target to make a Dexterity saving throw against the save DC (Monk's Focus) (interpretation, section 9) and, on a failure, have the Prone condition; the DC used MUST be recorded in the result. | SRD 5.2.1, "Level 3: Open Hand Technique", p. 52 | happy |
| FR-109 | At most one Open Hand Technique effect MUST be imposed per hit; a request naming two effects, or none that exists, MUST yield `MonkRejected(notEligible)`. | SRD 5.2.1, "Level 3: Open Hand Technique", p. 52 ("one of the following") | unhappy (invalid input) |
| FR-110 | From Monk level 6 in this subclass, Wholeness of Body MUST be usable as a Bonus Action: roll the Martial Arts die and regain Hit Points equal to the number rolled plus the Wisdom modifier, with a minimum of 1 Hit Point regained (table 4.10a). | SRD 5.2.1, "Level 6: Wholeness of Body", p. 52 | happy |
| FR-111 | Wholeness of Body MUST have a number of uses equal to the Wisdom modifier, with a minimum of once, and MUST regain all expended uses when the Monk finishes a Long Rest, but not a Short Rest (table 4.10b). | SRD 5.2.1, "Level 6: Wholeness of Body", p. 52 | happy |
| FR-112 | A Wholeness of Body request with no uses left MUST yield `MonkRejected(noUsesRemaining)` and MUST NOT roll or change state. | SRD 5.2.1, "Level 6: Wholeness of Body", p. 52 | unhappy (state error) |
| FR-113 | From Monk level 11 in this subclass, `ApplyFleetStep` given the `BonusActionTaken` `other` (a Bonus Action other than Step of the Wind) MUST return `ApplyFleetStepAvailable(granted: MonkFocusFeature.stepOfTheWind)`, recording that Step of the Wind is also available immediately after that Bonus Action; `BonusActionTaken.stepOfTheWind`, a missing or different subclass (FR-148), or a use below level 11 MUST yield `MonkRejected(notEligible)` or `MonkRejected(levelTooLow)` in FR-135 order. The follow-up use adds no Bonus Action cost in this engine (section 9). | SRD 5.2.1, "Level 11: Fleet Step", p. 52 | happy |
| FR-114 | From Monk level 17 in this subclass, when the Monk hits a creature with an Unarmed Strike, the Monk MUST be offered to expend 4 Focus Points to start the vibrations in the target. | SRD 5.2.1, "Level 17: Quivering Palm", p. 52 | happy |
| FR-115 | Starting the Quivering Palm with fewer than 4 Focus Points, on a hit with a Monk weapon that is not an Unarmed Strike, or below Monk level 17, MUST yield `MonkRejected` of `notEnoughFocusPoints`, `notEligible` or `levelTooLow` respectively, with no spend. | SRD 5.2.1, "Level 17: Quivering Palm", p. 52 | unhappy (boundary) |
| FR-116 | The vibrations MUST be recorded with a duration of days equal to the Monk level at the time they start, and the target by an opaque target reference; the engine MUST NOT read the clock. | SRD 5.2.1, "Level 17: Quivering Palm", p. 52 | happy |
| FR-117 | While vibrations are active the Monk MUST be offered to end them by taking an action; the vibrations are harmless unless ended. | SRD 5.2.1, "Level 17: Quivering Palm", p. 52 | happy |
| FR-118 | When the Monk takes the Attack action on a turn, the Monk MUST be offered to forgo one of the attacks to end the vibrations. | SRD 5.2.1, "Level 17: Quivering Palm", p. 52 | happy |
| FR-119 | Ending the vibrations while the Monk and the target are not on the same plane of existence MUST yield `MonkRejected(notEligible)`, and the vibrations MUST stay active. | SRD 5.2.1, "Level 17: Quivering Palm", p. 52 | unhappy (state error) |
| FR-120 | When the vibrations are ended with effect, the target MUST make a Constitution saving throw against the save DC (Monk's Focus) (the Monk's Focus DC, p. 50: "Some features that use Focus Points require your target to make a saving throw"), whose outcome is a command input (FR-141), and the result MUST carry the DC used and the Force damage applied: 10d12 on a failed save or half as much on a successful one, with the 10d12 rolled through the `dice.Roller` port (table 4.10c). | SRD 5.2.1, "Level 17: Quivering Palm", p. 52; "Level 2: Monk's Focus", p. 50 | happy |
| FR-121 | Starting vibrations while another creature is already under the effect MUST yield `MonkRejected(alreadyUsed)` and MUST NOT spend Focus Points (section 9). | SRD 5.2.1, "Level 17: Quivering Palm", p. 52 ("only one creature"); the refusal is an engine decision, section 9 | unhappy (state error) |
| FR-122 | The Monk MUST be able to end the vibrations harmlessly with no action, with no save, no damage and no roll. | SRD 5.2.1, "Level 17: Quivering Palm", p. 52 | happy |
| FR-123 | Ending vibrations when none are active MUST yield `MonkRejected(notEligible)`. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |

**Table 4.10a. Wholeness of Body Hit Points (FR-110), one test per row.** SRD 5.2.1, "Level 6: Wholeness of Body", p. 52. The Hit Point maximum cap is hit-points-and-damage.md's.

| Case | Martial Arts die roll | Wisdom modifier | Hit Points regained | Path |
|---|---|---|---|---|
| typical | 3 | +3 | 6 | happy |
| high | 6 | +5 | 11 | happy |
| zero modifier | 1 | 0 | 1 | unhappy (boundary) |
| sum below minimum | 1 | -1 | 1 (minimum) | unhappy (boundary) |
| sum exactly at minimum | 2 | -1 | 1 | unhappy (boundary) |

**Table 4.10b. Wholeness of Body uses (FR-111), one test per row.** SRD 5.2.1, "Level 6: Wholeness of Body", p. 52.

| Case | Wisdom modifier | Uses | Path |
|---|---|---|---|
| negative | -2 | 1 | unhappy (boundary) |
| zero | 0 | 1 | unhappy (boundary) |
| one | +1 | 1 | unhappy (boundary) |
| three | +3 | 3 | happy |
| five | +5 | 5 | happy |

**Table 4.10c. Quivering Palm end with effect (FR-120), one test per row.** SRD 5.2.1, "Level 17: Quivering Palm", p. 52.

| Case | Constitution save | 10d12 rolls | Damage | Path |
|---|---|---|---|---|
| failed save, lowest | failure | all 1 | 10 Force | unhappy (boundary) |
| failed save, highest | failure | all 12 | 120 Force | unhappy (boundary) |
| successful save, even total | success | all 2 (total 20) | 10 Force | happy |
| successful save, lowest | success | all 1 (total 10) | 5 Force | unhappy (boundary) |

### 4.11 Rests, content, collaborators and cross-cutting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-124 | Given a `RestKind` (rests-and-hit-dice.md; its `RestApplied` event is defined there), a Short Rest MUST regain all expended Focus Points only; a Long Rest MUST regain all expended Focus Points, restore Uncanny Metabolism, and restore all Wholeness of Body uses. A rest MUST NOT end active Quivering Palm vibrations (`RestMonk` does not touch them; an inference from the duration in days, engine decision, section 9). | SRD 5.2.1, "Level 17: Quivering Palm", p. 52; "Level 2: Monk's Focus", p. 50; "Level 2: Uncanny Metabolism", p. 51; "Level 6: Wholeness of Body", p. 52 | happy |
| FR-125 | When the Monk class definition or the progression row for the level cannot be found in the content source, `GetMonkLevelProfile` MUST return its `GetMonkLevelProfileContentMissing` variant holding a `MissingContent(kind, id)`, and nothing else changes. Every other operation receives the `MonkLevelProfile` as an argument and has no content-missing variant (FR-140, FR-145). | `doc/rules/architecture.md` §4; decisions FR-017, FR-018 | unhappy (missing content) |
| FR-126 | A typed `*srderr.PortError` returned by the `ContentSource` (in `GetMonkLevelProfile`) or by the `dice.Roller` (in the use cases that roll) MUST propagate unchanged, with no partial state change and no spend of Focus Points. | `doc/rules/architecture.md` §4, §6 | unhappy (failure of collaborator) |
| FR-127 | With the same inputs, state and fixed dice, an operation MUST return an equal result every time; all randomness MUST come from the `dice.Roller` port (only in the use cases that roll, which pass rolled values to the pure function); all-1 and all-maximum rolls MUST return the minimum and maximum values in the relevant table. | `doc/rules/testing.md` §4 (determinism and randomness) | unhappy (determinism) |
| FR-128 | A use case MUST NOT alter its input state, and every collection in a returned state, result or event MUST be an unmodifiable view. | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |
| FR-129 | Every use case MUST return an equal result with the no-op `telemetry.Telemetry` and with a recording one. (Pure functions are silent and take no `telemetry.Telemetry`.) | `doc/rules/observability.md` §9 | happy |
| FR-130 | A failure that leaves a use case (the `GetMonkLevelProfileContentMissing` result, a returned `*srderr.PortError`) MUST be recorded once: span error status and one increment of `srd5e.character.feature_used` with `srd5e.outcome=failure`, and no log. A returned `*srderr.PortError` is also counted once on the shared `srd5e.character.feature_used.errors` with `error.type` (FR-147; decisions FR-063), never as `error.type` on the `feature_used` counter; a returned content-missing result is not an error signal (errors-and-results.md FR-043) and instead sets `srd5e.character.rejection=content_missing` on the counter. A `...Rejected` outcome MUST NOT set error status. | `doc/rules/observability.md` §6, §7; decisions FR-041, FR-042 | unhappy (telemetry on failure) |
| FR-132 | Every Monk operation that changes state MUST take the `Character` and return the new `Character` plus events; Monk state is reached only through the root, as a `MonkState` in its `FeatureStates` (FR-144). The Monk level MUST be derived from the `ClassLevels` entry for `ClassId('monk')` (derived-stats.md); `MonkState` MUST NOT store its own copy of the level or of any maximum derived from it (FR-133). | character-aggregate.md (`FeatureStates`); decisions FR-008, FR-022; `doc/rules/architecture.md` §3 | unhappy (state error) |
| FR-133 | `MonkState` MUST store only the Focus Points `expended` count. The maximum MUST be derived on read from the `MonkLevelProfile` for the current Monk level (`FocusPoints(maximum: profile.focusPointsMaximum, expended: state.expended)`, FR-033), so gaining a Monk level needs no Monk operation: expended points are kept and available points rise by the difference (1 of 2 spent at level 2 becomes 1 of 3 spent at level 3, 2 available). | SRD 5.2.1, "Monk Features", p. 50; "Level 2: Monk's Focus", p. 50 | unhappy (rule interaction) |
| FR-134 | The Wholeness of Body uses remaining MUST be the maximum of FR-111 minus the uses spent, but never below 0. When the Wisdom modifier falls so the maximum drops below the uses spent, remaining MUST be 0 (a use then yields `MonkRejected(noUsesRemaining)`); the spent count MUST NOT go negative and a later rise in the maximum MUST NOT restore uses before a Long Rest. | SRD 5.2.1, "Level 6: Wholeness of Body", p. 52; engine decision, section 9 | unhappy (boundary) |
| FR-135 | When a command could yield more than one `MonkRejected` reason, the use case MUST check in this fixed order and return the first that applies: (1) `notAMonk`, (2) `levelTooLow`, (3) `notEligible` (which includes a missing or different subclass, FR-148), (4) `alreadyUsed` or `noUsesRemaining`, (5) `notEnoughFocusPoints`. Content that cannot be loaded never competes: the profile is loaded earlier by `GetMonkLevelProfile` (FR-125), and `subclassAlreadyChosen` left with the subclass choice (FR-067 withdrawn). Table 4.11a gives the boundary rows. | `doc/rules/testing.md` §4 (state errors); engine decision, section 9 | unhappy (rule interaction) |
| FR-139 | `MonkCreationChoices` (FR-143) MUST hold a closed `MonkEntry` (`level1` or `multiclass`) and be checked by the pure `ValidateMonkCreationChoices(choices, ClassDefinition)`. FR-005, FR-008 and FR-012 apply only to `level1`; for `multiclass` it MUST accept no skill, tool or equipment choice: any supplied choice yields `ValidateMonkCreationChoicesRejected` carrying `MonkChoiceProblem.notEligible` (the one operation whose rejected variant carries a `MonkChoiceProblem`), and the `ValidateMonkCreationChoicesAccepted` variant MUST list the traits gained (FR-014, FR-015). | SRD 5.2.1, "As a Level 1 Character", "As a Multiclass Character", p. 49 | unhappy (rule interaction) |
| FR-140 | The pure `MonkRules` and `MonkFeatures` functions MUST take a `MonkLevelProfile` (or the level-derived number from it) as an argument and MUST NOT hold SRD table data in code; only `GetMonkLevelProfile` loads the profile through the `ContentSource`, building it from the content-classes.md `ProgressionRow` (FR-152), and the host passes it to every other operation. The one other content input is the `ClassDefinition` passed to `ValidateMonkCreationChoices` (FR-004, FR-139); no skill list or tool category is hard-coded. | `doc/rules/architecture.md` §4, §5; content and code separation (NOTICE.md) | happy |
| FR-141 | Where a feature involves a saving throw by a target (Stunning Strike, Deflect redirect, Open Hand Technique, Quivering Palm), the input MUST carry the `D20TestOutcome` (`success` or `failure`; no Monk-local type) as a fact and the operation MUST return the applied effect for it; rolling the save is d20-tests.md's. For the Monk's own failed save (Disciplined Survivor) the operation only spends the Focus Point; the host rerolls through `Roll.applyReroll` (FR-095). | `doc/rules/architecture.md` §4; decisions FR-002, FR-004; SRD 5.2.1, "Level 2: Monk's Focus", p. 50 | happy |
| FR-142 | `ResolveOpenHandTechnique` MUST take the chosen optional `SubclassId` (FR-148), the hit-by-Flurry-of-Blows fact, the chosen effect and the target's `D20TestOutcome` where the effect has a save, and return the effect and the `DifficultyClass` used (FR-105 to FR-109); `OfferSelfRestoration` MUST take the `ConditionKind`s on the Monk and return the removable ones (FR-091, FR-092). Both are pure (FR-145). | SRD 5.2.1, "Level 3: Open Hand Technique", "Level 10: Self-Restoration", p. 52 | happy |
| FR-143 | The Monk's creation choices MUST be a `MonkCreationChoices` implementing both pure methods of `ClassCreationChoices` (character-aggregate.md FR-094), carried by `CreateCharacter` (character-creation.md). It holds the `MonkEntry` and the skill, tool and equipment choices as plain fields. (1) Seeding (`FeatureStates` to `FeatureStates`): for both `MonkEntry` variants it MUST write an initial `MonkState` under the `monk` key (FR-150). (2) The key-listing method MUST return an unmodifiable empty list of `ContentKey`s, as class-fighter.md FR-144 does: the tool is a `ChosenTool` of the command (character-creation.md FR-110, FR-111, looked up under FR-006) and the starting-equipment ids are resolved from the plain fields (character-creation.md FR-015 to FR-020), so nothing is left unresolved; this closes character-creation.md FR-103 and Q-26 on the Monk side. Validation is the separate pure `ValidateMonkCreationChoices` (FR-139), called by the host before `CreateCharacter`; the aggregate stores no Monk-specific field, and `classChoicesMismatch` is withdrawn (character-creation.md FR-021). | decisions FR-024; character-aggregate.md FR-094; character-creation.md FR-103 | happy |
| FR-144 | The Monk state MUST be a `MonkState` implementing `FeatureState`, stored in the `Character`'s `FeatureStates` under a typed `FeatureStateKey[MonkState]` named `monk` (matching `[a-z][a-z0-9_]*`) that this spec defines. A test MUST show the key round-trips a `MonkState` and that reading it with a key of another type `T` panics with an error value. | decisions FR-022; character-aggregate.md FR-093 | happy |
| FR-145 | Each Monk feature MUST be a pure domain function from (`Character`, inputs, `MonkLevelProfile`) to (new `Character`, events) or a `...Rejected` variant. Only a feature that needs dice (Uncanny Metabolism, Patient Defense at level 10 and above, Deflect Attacks and its redirect, Wholeness of Body, Quivering Palm ending with effect) or content (`GetMonkLevelProfile`) has a use case, which supplies the rolled values or the profile through ports. | decisions FR-023 | happy |
| FR-146 | Each operation MUST return its own closed `<Operation>Result` whose variants are prefixed with the operation. The names `MonkRejected` and `ContentMissing` MUST NOT be exported; the reason is a `MonkRejection` field and the missing content a `MissingContent(kind, id)` field. Event and result names MUST be unique across the package API. | decisions FR-017, FR-018, FR-021 | unhappy (name collision) |
| FR-147 | Use cases MUST record feature use on the shared counter `srd5e.character.feature_used` with `srd5e.class` (`monk`), `srd5e.feature`, `srd5e.outcome` (`success` or `failure`), plus `srd5e.character.rejection` for every non-success returned variant (a `MonkRejection` value, or `content_missing`; FR-130). A returned `*srderr.PortError` is counted on the shared `.errors` signal (character-aggregate.md, decisions FR-063) with `srd5e.class`, `srd5e.feature` and `error.type` (the snake_case subtype name from errors-and-results.md), once, by the outermost use case; the counter carries no `error.type`. No Monk-specific counter name exists. Each use case also has one span and times itself on the shared `.duration` histogram (section 7). | decisions FR-041, FR-042, FR-043, FR-063 | happy |
| FR-148 | The subclass operations (`ResolveOpenHandTechnique`, `UseWholenessOfBody`, `ApplyFleetStep`, `StartQuiveringPalm`) MUST take the chosen optional `SubclassId` as an explicit input, because `Character` has no subclass accessor (character-aggregate.md). A null subclass, or one other than `SubclassId('warrior_of_the_open_hand')`, MUST yield `MonkRejected(notEligible)` with no roll and no state change. The check sits at position (3) of FR-135, so a level 5 Monk with no subclass asking for Wholeness of Body gets `levelTooLow` (table 4.11a). If the aggregate later exposes the subclass, the input is replaced by it. | SRD 5.2.1, "Monk Subclass: Warrior of the Open Hand", p. 52; decisions FR-020, FR-022 | unhappy (state error) |
| FR-149 | Every operation that takes a `MonkLevelProfile` MUST check, after `notAMonk` (FR-135), that `profile.level` equals the character's Monk level from `ClassLevels`. A mismatch is a caller wiring bug and MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error), not a rejection, with no state change; a profile for level 5 with a level 8 character is the test case. | `doc/rules/testing.md` §4 (invalid input); engine decision, section 9 | unhappy (invalid input) |
| FR-150 | A `Character` with a Monk level in `ClassLevels` but no `MonkState` MUST be read as holding the initial `MonkState` (no Focus Points expended, Uncanny Metabolism unused, no Wholeness of Body uses spent, no vibrations); the first state change writes it. `MonkCreationChoices` seeds the same value at creation (FR-143). The pure `MonkFeatures.seedMonkState(Character)` MUST write the initial state for a Character with a Monk level and no stored state (`SeedMonkStateSeeded`), return `SeedMonkStateRejected(notAMonk)` with no Monk level and `SeedMonkStateRejected(alreadyUsed)` when a `MonkState` is already stored. A host or the level-up flow MAY call it on the first Monk level gained by multiclassing, but correctness does not depend on it, so a multiclass Monk is never wrongly `notAMonk` (FR-023). | character-aggregate.md FR-092, FR-094; decisions FR-022 | unhappy (state error) |
| FR-151 | Slow Fall, Evasion, Body and Mind, Unarmored Movement and the Martial Arts offers MUST each be a pure operation with its own `<Operation>Result` (section 6): `ApplySlowFall` (FR-072 to FR-074), `ApplyEvasion` (FR-083 to FR-085), `ApplyBodyAndMind` (FR-103, FR-104), `GetUnarmoredMovement` (FR-044, FR-045), `GetMartialArtsOffers` (FR-026 to FR-030). Each returns `...Rejected(notAMonk)` first (FR-135). The first three then return `...Rejected(levelTooLow)` below Monk level 4, 7 and 20 respectively; `GetUnarmoredMovement` and `GetMartialArtsOffers` have no level floor of their own, and Unarmored Movement at level 1 returns a 0 ft `SpeedBonus` (FR-045). | SRD 5.2.1, "Monk Features", p. 50; decisions FR-017 | unhappy (boundary) |
| FR-152 | `MonkLevelProfile` MUST be built by one pure factory from a content-classes.md `ProgressionRow` for the Monk: the `martial_arts` column's `ColumnDice` becomes the `MartialArtsDie`; the `focus_points` `ColumnCount` the Focus Points maximum; the `unarmored_movement` `ColumnFeetBonus` the `SpeedBonus`; a `ColumnNone` (level 1 Focus Points and Unarmored Movement) becomes 0. A column of the wrong `ColumnValue` kind is a programmer error (panic, the decoder having checked it, content-classes.md FR-040). | SRD 5.2.1, "Monk Features", p. 50; content-classes.md FR-040 | happy |

**Table 4.11a. Rejection precedence (FR-135), one test per row.** Engine decision, section 9.

| Case | Conditions that apply together | Result | Path |
|---|---|---|---|
| no Monk levels, Slow Fall | no Monk part and level too low | `notAMonk` | unhappy (rule interaction) |
| Slow Fall at level 3, Reaction already used | level 3 and the Reaction spent (host fact) | `levelTooLow` | unhappy (boundary) |
| Deflect at level 2, Fire only | level 2 and Fire only | `levelTooLow` (not `notEligible`) | unhappy (rule interaction) |
| Deflect at level 3, Fire only, no Focus Points | Fire only at level 3 | `notEligible` | unhappy (rule interaction) |
| Redirect, damage after above 0, no Focus Points | not eligible and no Focus Points | `notEligible` (not `notEnoughFocusPoints`) | unhappy (rule interaction) |
| Stunning Strike, second in turn, no Focus Points | already used and no Focus Points | `alreadyUsed` | unhappy (rule interaction) |
| Quivering Palm start, level 17, Monk weapon, 3 Focus Points | not an Unarmed Strike and too few points | `notEligible` | unhappy (rule interaction) |
| Quivering Palm start, level 16, 0 Focus Points | level too low and too few points | `levelTooLow` | unhappy (boundary) |
| Wholeness of Body, level 5, no uses left or no subclass chosen | level too low and no uses (or no subclass) | `levelTooLow` | unhappy (boundary) |
| Wholeness of Body, level 6, no subclass chosen, no uses left | subclass missing and no uses | `notEligible` (FR-148; not `noUsesRemaining`) | unhappy (rule interaction) |
| Quivering Palm start, level 17, subclass other than Warrior of the Open Hand, 0 Focus Points | wrong subclass and too few points | `notEligible` | unhappy (rule interaction) |
| Any operation, no Monk level, profile level mismatch | no Monk part and a wrong profile | `notAMonk` (the profile check FR-149 comes after) | unhappy (rule interaction) |
| Uncanny Metabolism, already used, not an Initiative trigger | already used and wrong trigger | `notEligible` | unhappy (rule interaction) |

**Withdrawn ids** (moved to the named owner; ids stay reserved)

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (Primary Ability; decisions FR-025) |
| FR-002 | Withdrawn: moved to content-classes.md (Hit Point Die; decisions FR-025) |
| FR-003 | Withdrawn: moved to content-classes.md (saving throws; decisions FR-025) |
| FR-006 | Withdrawn: moved to content-classes.md (weapon proficiencies; decisions FR-025) |
| FR-009 | Withdrawn: moved to content-classes.md (armor training; decisions FR-025) |
| FR-010 | Withdrawn: moved to content-classes.md (starting equipment option A; decisions FR-025) |
| FR-011 | Withdrawn: moved to content-classes.md (starting equipment option B; decisions FR-025) |
| FR-013 | Withdrawn: moved to character-creation.md (`CreateCharacter` content-missing for every missing id; decisions FR-017, FR-018, FR-024) |
| FR-017 | Withdrawn: moved to content-classes.md (Class Features column; decisions FR-025) |
| FR-019 | Withdrawn: moved to content-classes.md (Martial Arts column; decisions FR-025) |
| FR-020 | Withdrawn: moved to content-classes.md (Focus Points column; decisions FR-025) |
| FR-021 | Withdrawn: moved to content-classes.md (Unarmored Movement column; decisions FR-025) |
| FR-022 | Withdrawn: moved to content-classes.md (feature availability by level; decisions FR-025) |
| FR-064 | Withdrawn: moved to level-advancement-and-multiclassing.md (subclass choice; decisions FR-020) |
| FR-065 | Withdrawn: moved to level-advancement-and-multiclassing.md (subclass level; decisions FR-020) |
| FR-066 | Withdrawn: moved to level-advancement-and-multiclassing.md (unknown subclass id; decisions FR-017, FR-018, FR-020) |
| FR-067 | Withdrawn: moved to level-advancement-and-multiclassing.md (`SubclassRejected`; decisions FR-020) |
| FR-069 | Withdrawn: moved to content-classes.md (feat-slot levels) and feats-behaviour.md (applying a feat) |
| FR-070 | Withdrawn: moved to feats-behaviour.md (feat qualification) |
| FR-071 | Withdrawn: moved to content-classes.md (Epic Boon slot level) and feats-behaviour.md |
| FR-131 | Withdrawn: moved to content-classes.md FR-068, FR-051, FR-052 (Monk feature completeness) and the decoder |
| FR-138 | Withdrawn: moved to feats-behaviour.md (`ChooseMonkFeat` and `MonkFeatChosen` no longer exist) |

## 5. Model

Domain (`character` package, pure domain files), pure, synchronous and silent. Types owned elsewhere (decisions
FR-001 to FR-012, FR-025): `D20TestOutcome`, `DifficultyClass`, `Roll` (d20-tests.md); `Ability`,
`AbilityModifier` (ability-scores.md); `dice.Roller` (dice-and-rolls.md); `ProficiencyBonus`,
`ClassLevels` (derived-stats.md); `Character`, `FeatureStates`, `FeatureState`, `FeatureStateKey`,
`CharacterEvent` (character-aggregate.md); `RestKind` (rests-and-hit-dice.md);
`ClassCreationChoices`, `CreateCharacter` (character-creation.md); `ClassDefinition`,
`SubclassDefinition`, `ProgressionRow` (content-classes.md); `ContentSource`
(content-source-port.md); `telemetry.Telemetry` (observability-ports.md); `CreatureSize`
(content-schema-and-decoder.md FR-109); `ClassId`, `SubclassId`, `FeatureId`, `DamageType`,
`ConditionKind`, `MissingContent` (`content`).

- **Value objects** (members not given in a requirement): `MonkLevel` (FR-016);
  `MartialArtsDie`; `FocusPoints` (`maximum`, `expended`; read-time view, FR-133;
  invariant `0 <= expended <= maximum`, FR-136); `MonkLevelProfile` (`level`,
  `martialArtsDie`, `focusPointsMaximum`, `unarmoredMovement`; FR-152);
  `SpeedBonus` (feet); `DamageReduction`; `MonkFocusFeature` (`flurryOfBlows`,
  `patientDefense`, `stepOfTheWind`); `MonkEffectEnd` (`startOfMonkNextTurn`);
  `StunningStrikeEffect` (closed: `StunApplied`, `StunResisted`); `BonusActionTaken`
  (`stepOfTheWind`, `other`); `SuperiorDefenseEffect`; `QuiveringPalmTarget`
  (opaque, caller-supplied: `character` cannot hold a `CombatantId`, decisions
  FR-012, FR-013; section 9); `QuiveringPalmVibrations` (target reference,
  `durationDays`); `MonkEntry` (closed: `level1`, `multiclass`);
  `MonkCreationChoices` (FR-143); `MonkChoiceProblem`; `MonkRejection` (FR-135).
- **Entity part:** `MonkState`, a `FeatureState` under `FeatureStateKey[MonkState]`
  `monk` (FR-144), changed only through the root. It holds `focusPointsExpended`,
  `uncannyMetabolismUsed`, `wholenessOfBodyUsesSpent` and optional vibrations;
  NOT the Monk level (FR-132), any maximum, the chosen subclass or feats.
- **Domain services (pure):** `MonkRules` for calculations; `MonkFeatures` for
  state changes (including `RestMonk`), holding no SRD table data (FR-140, FR-145).
- **Events (past tense, defined only here, decisions FR-019):** each implements
  the open `CharacterEvent` (character-aggregate.md FR-085), carries a
  `characterId`, and sits outside the closed `CharacterRecordEvent` switch:
  `FocusPointsExpended`, `FocusPointsRegained`, `UncannyMetabolismUsed`,
  `AttackDeflected`, `DeflectedForceRedirected`, `WholenessOfBodyUsed`,
  `StunningStrikeAttempted`, `QuiveringPalmStarted`, `QuiveringPalmEnded`,
  `DisciplinedSurvivorUsed`, `SuperiorDefenseActivated`.

Content ids are `snake_case` (decisions FR-007): class `monk`, subclass `warrior_of_the_open_hand`.
Bundled data reaches the code only through the one `ContentSetDecoder` (decisions FR-030, FR-031).
The engine produces no display strings.

## 6. Interface

Features needing neither dice nor content are **pure functions** on `MonkFeatures` / `MonkRules`,
with no ports and no telemetry. Features needing dice or content are **use cases**: one `call`
method, an immutable command, an optional `telemetry.Telemetry` (no-op default) and the ports below
(FR-145). Every operation on a Monk takes the `Character`; every success result carries the new
`Character` and events (FR-132). The `MonkLevelProfile` is passed in (FR-140) and checked by FR-149
(panic, so no variant); targets' saves are command facts (FR-141). In the Variants column
`...` is the operation's name (FR-146); every operation that can refuse has `...Rejected` holding a
`MonkRejection` (`MonkChoiceProblem` for `ValidateMonkCreationChoices`); only `GetMonkLevelProfile`
has `...ContentMissing`.

| Operation | Form | Inputs | `<Operation>Result`: variants | Ports |
|---|---|---|---|---|
| `GetMonkLevelProfile` | use case, context-aware | `MonkLevel` (FR-125, FR-152) | `...Found` (the `MonkLevelProfile`), `...ContentMissing` | `ContentSource` |
| `ValidateMonkCreationChoices` | pure | `MonkCreationChoices`, the Monk `ClassDefinition` (FR-004 to FR-012, FR-139) | `...Accepted` (traits gained), `...Rejected` (`MonkChoiceProblem`) | none |
| `MonkCreationChoices` seeding and key-listing methods | pure (the two `ClassCreationChoices` methods, FR-143) | `FeatureStates` (seeding); none (key-listing) | seeding: `FeatureStates` with the initial `MonkState` (FR-150); key-listing: an unmodifiable empty `ContentKey` list; no result types | none |
| `SeedMonkState` | pure | `Character` (FR-150) | `...Seeded` (`Character`), `...Rejected` (`notAMonk`, `alreadyUsed`) | none |
| `GetMartialArtsOffers` | pure | `Character`, `MonkLevelProfile`, held-items facts, armor worn, Shield wielded (FR-026 to FR-030) | `...Offered` (empty when FR-026 fails), `...Rejected` (`notAMonk`) | none |
| `GetUnarmoredDefenseBasis` | pure | the content entry's `FeatureId` | the `AbilityBased` value (FR-031, FR-032); no result type | none |
| `GetUnarmoredMovement` | pure | `Character`, `MonkLevelProfile`, armor worn, Shield wielded (FR-044, FR-045) | `...Found` (`SpeedBonus`), `...Rejected` (`notAMonk`) | none |
| `UseFocusFeature` | use case, sync | `Character`, `MonkLevelProfile`, `MonkFocusFeature`, whether Points are spent, optional carried-creature facts (willing, `CreatureSize`, distance; FR-033 to FR-043, FR-087 to FR-090) | `...Used` (offers, Temporary Hit Points rolled, carried-creature decision), `...Rejected` | `dice.Roller` (Patient Defense, level 10+) |
| `UseUncannyMetabolism` | use case, sync | `Character`, `MonkLevelProfile`, Initiative trigger (FR-046 to FR-050) | `...Resolved` (roll, Hit Points regained), `...Rejected` | `dice.Roller` |
| `ApplyPerfectFocus` | pure | `Character`, `MonkLevelProfile`, Initiative trigger, whether Uncanny Metabolism was used (FR-097 to FR-099) | `...Applied`, `...Rejected` | none |
| `ResolveDeflectAttacks` | use case, sync | `Character`, `MonkLevelProfile`, incoming damage with `DamageType`s, Dexterity modifier (FR-051 to FR-055, FR-062, FR-063) | `...Resolved` (reduction, damage after), `...Rejected` | `dice.Roller` |
| `RedirectDeflectedAttack` | use case, sync | `Character`, `MonkLevelProfile`, the `ResolveDeflectAttacksResolved`, melee or ranged, target range facts, the target's `D20TestOutcome` (FR-056 to FR-061, FR-137) | `...Resolved` (`DifficultyClass` used, damage on a failed save, damage applied, `DamageType`), `...Rejected` | `dice.Roller` |
| `ApplySlowFall` | pure | `Character`, fall damage (FR-072 to FR-074) | `...Applied` (reduction, damage after), `...Rejected` (`notAMonk`, `levelTooLow`) | none |
| `AttemptStunningStrike` | pure | `Character`, `MonkLevelProfile`, hit facts, turn flag, the target's `D20TestOutcome` (FR-076 to FR-081) | `...Resolved` (`DifficultyClass` used, `StunApplied` or `StunResisted`), `...Rejected` | none |
| `ApplyEvasion` | pure | `Character`, the save's `Ability`, `D20TestOutcome`, half-damage clause, Incapacitated, the effect's damage (FR-083 to FR-085) | `...Applied` (damage taken), `...Unchanged` (the effect's own result stands), `...Rejected` (`notAMonk`, `levelTooLow`) | none |
| `UseDisciplinedSurvivor` | pure | `Character`, `MonkLevelProfile`, failed-save fact (FR-094 to FR-096) | `...Authorised` (`Character`, `DisciplinedSurvivorUsed`; the host then calls `Roll.applyReroll`, FR-095), `...Rejected` | none |
| `ActivateSuperiorDefense` | pure | `Character`, `MonkLevelProfile`, turn-start fact (FR-100 to FR-102) | `...Resolved` (`SuperiorDefenseEffect`), `...Rejected` | none |
| `OfferSelfRestoration` | pure | `Character`, `ConditionKind`s on the Monk, turn-end fact (FR-091 to FR-093, FR-142) | `...Offered` (removable conditions), `...Rejected` | none |
| `ApplyBodyAndMind` | pure | `Character` (FR-103, FR-104) | `...Applied` (new scores; the host applies them), `...Rejected` (`notAMonk`, `levelTooLow`) | none |
| `ResolveOpenHandTechnique` | pure | `Character`, optional `SubclassId`, hit-by-Flurry fact, chosen effect, `D20TestOutcome` where the effect has a save (FR-105 to FR-109, FR-142, FR-148) | `...Resolved` (effect, `DifficultyClass` used), `...Rejected` | none |
| `UseWholenessOfBody` | use case, sync | `Character`, optional `SubclassId`, `MonkLevelProfile` (FR-110 to FR-112, FR-134, FR-148) | `...Resolved` (Hit Points regained), `...Rejected` | `dice.Roller` |
| `ApplyFleetStep` | pure | `Character`, optional `SubclassId`, `BonusActionTaken` (FR-113, FR-148) | `...Available` (`granted: MonkFocusFeature.stepOfTheWind`), `...Rejected` | none |
| `StartQuiveringPalm` | pure | `Character`, optional `SubclassId`, `MonkLevelProfile`, `QuiveringPalmTarget`, hit-with-Unarmed-Strike fact (FR-114 to FR-116, FR-121) | `...Started` (`Character`, `QuiveringPalmStarted`), `...Rejected` | none |
| `EndQuiveringPalm` | use case, sync | `Character`, plane fact, harmless or with effect, `D20TestOutcome` for an effect (FR-117 to FR-120, FR-122, FR-123) | `...Resolved` (`DifficultyClass` used, damage applied), `...Rejected` | `dice.Roller` (with effect) |
| `RestMonk` | pure | `Character`, `RestKind` (FR-124) | `...Rested` (`Character`, events), `...Rejected` (`notAMonk` only) | none |

Use cases are synchronous because `dice.Roller` is; only `GetMonkLevelProfile` is context-aware
(`ContentSource` may do I/O). Its `ContentLookup` (`ContentFound` | `ContentNotFound`) maps to
`GetMonkLevelProfileContentMissing` (FR-125); an infrastructure failure is a typed `*srderr.PortError`
that propagates (FR-126).

**Exports.** the `character` package exports the types above and the module's public packages
re-exports them (FR-146). Test doubles are the `*test` packages's (decisions FR-038); this spec adds
no port, adapter or double.

## 7. telemetry.Telemetry

Class specs define no signals of their own (decisions FR-042, FR-063). The seven use cases use the
shared `srd5e.character.feature_used` counter and its `.duration` and `.errors` companions
(character-aggregate.md); FR-130 and FR-147 give the recording rules. No log is emitted; a rule
rejection is a normal outcome with no error status. The domain and every pure operation are **silent
by design**; the host or wrapping character use case counts their calls. Evaluation
(`doc/rules/observability.md` §2). What can go wrong: missing Monk content, a port failure
(`ContentSource`, `dice.Roller`), a rule rejection or an invalid argument. Missing content and port
failures show as span error status and `.errors` increments with `error.type`; a returned
content-missing result is `failure` with `rejection=content_missing` (errors-and-results.md FR-043);
roll values stay in results, so cardinality is bounded. Names are constants in the `character`
context and feed `doc/observability/signals.md`.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| Shared `srd5e.character.feature_used`, `.duration`, `.errors` (this spec adds values only) | counter, histogram | `srd5e.class`=`monk`; `srd5e.feature`=`get_profile`, `flurry_of_blows`, `patient_defense`, `step_of_the_wind`, `uncanny_metabolism`, `deflect_attacks`, `deflect_redirect`, `wholeness_of_body`, `quivering_palm_end`; `srd5e.outcome`; `srd5e.character.rejection`=snake_case `MonkRejection` or `content_missing` | Feature use and failure by feature | Any `.errors` increment, any `content_missing`, a rising share of `not_enough_focus_points` |
| Spans `srd5e.character.<name>`: `get_monk_level_profile`, `use_focus_feature`, `use_uncanny_metabolism`, `resolve_deflect_attacks`, `redirect_deflected_attack`, `use_wholeness_of_body`, `end_quivering_palm` | span | `srd5e.class`, `srd5e.feature`, `srd5e.outcome`, `srd5e.character.rejection`; `error.type` on a returned port failure | Latency and failures per use case | Error status; `GetMonkLevelProfile` is timed with `telemetry.Timer`, the others with `telemetry.Timer` |

## 8. Test plan

Doubles from the `*test` packages (decisions FR-038): a fixed `dice.Roller`, a recording
`telemetry.Telemetry`, `FakeContentSource`. Fixtures per section 4.2. No new port, so no contract suite.
Pure functions use plain values.

| Id | Level | Notes |
|---|---|---|
| FR-004 to FR-016, FR-018, FR-023, FR-132, FR-139, FR-140, FR-143, FR-144, FR-149 to FR-152 | unit | `ValidateMonkCreationChoices` table-driven over valid and invalid choices with a `ClassDefinition` fixture; `level1` versus `multiclass` (any supplied choice rejected with `notEligible`; the multiclass case uses a `Character` that already has another class); `MonkCreationChoices` seeding writes the initial `MonkState` for both entries and leaves other keys alone; the key-listing method returns an unmodifiable empty list (FR-143); `MonkLevel` boundaries (0, 1, 20, 21; `MonkLevel` still rejects 0 while `notAMonk` arises from the `Character` input) and the `New` constructor (returns an error); Proficiency Bonus from the character; no Monk levels gives `notAMonk` but a stored-state-less multiclass Monk reads as initial; `SeedMonkState`; level 5 profile with a level 8 character panics; key `monk` round-trips and a key of another type panics (character-aggregate.md FR-093); FR-151 operations; the profile factory maps `ColumnDice`, `ColumnCount`, `ColumnFeetBonus`, `ColumnNone` and rejects a wrong kind |
| FR-024 to FR-050, FR-133, FR-136 | unit and application | Table 4.3a; the Unarmored Defense declaration (named set `{Wisdom}`, `shieldAllowed` false, supplied `FeatureId` carried, identical at levels 1 to 20); arithmetic is derived-stats.md's; Tables 4.4a and 4.4b (with the multiclass row); a higher-level profile keeps expended points; offers, spend and rejections on `MonkFeatures`; `UseFocusFeature` with a fixed roller; Speed bonus by level with and without armor and Shield; table 4.5a; once-per-Long-Rest and trigger checks; `UseUncannyMetabolism` rolls the profile's die |
| FR-051 to FR-063, FR-068, FR-105 to FR-109, FR-113, FR-137, FR-142, FR-148 | unit and application | Tables 4.6a (levels 12 and 13 included) and 4.6b with the `D20TestOutcome` as input; rejections through `RedirectDeflectedAttack`; Subclass feature availability by level; `ResolveOpenHandTechnique` offers and `DifficultyClass` used; `ApplyFleetStep` for `other` and `stepOfTheWind`; subclass operations return `notEligible` for a null or different optional `SubclassId` in FR-135 order |
| FR-072 to FR-096, FR-141 | unit and application | Tables 4.8a, 4.8b; `AttemptStunningStrike` carries `StunApplied` or `StunResisted`; level boundary offers (FR-075, FR-082, FR-086); Heightened Focus before and after level 10; `OfferSelfRestoration`; save proficiency set (including a multiclass Monk); `UseDisciplinedSurvivor` returns `...Authorised` with no reroll flag and composed with `Roll.applyReroll(RerollDie(i), newValue: n)` a lower new value is kept (d20-tests.md FR-059) |
| FR-097 to FR-104, FR-110 to FR-123, FR-134 | unit and application | Tables 4.9a, 4.9b; `ActivateSuperiorDefense` duration `time.Duration(seconds: 60)` and `resistsAllExcept` equal to `{DamageType.force}` (every other `DamageType` resisted); Tables 4.10a to 4.10c; `UseWholenessOfBody`; uses after a Wisdom drop; `StartQuiveringPalm`, `EndQuiveringPalm` with the `D20TestOutcome` as input |
| FR-124 to FR-128 | unit and application | `RestMonk` for each `RestKind`; `GetMonkLevelProfile` with missing-content and port-failure doubles; a failing `dice.Roller` through every rolling use case; Equal results on repeat calls; all-1 and all-maximum rolls give table minimum and maximum; input `Character` unchanged; returned collections reject mutation |
| FR-129, FR-130, FR-135 (table 4.11a), FR-145 to FR-147 | application, architecture | Recording `telemetry.Telemetry` against section 7: span names, `.duration` once per call, equal result with no-op and recording; content-missing is `failure` with `rejection=content_missing` and error status, not on `.errors`; a returned `*srderr.PortError` counted once on `.errors` with `error.type`, none on the counter; no log; Rejection precedence table-driven through the operations named in each row; package API scan: no exported `MonkRejected` or `ContentMissing`, no duplicate result or event names, no `context.Context`-taking or a `context.Context`-taking call in pure functions |

Withdrawn ids are covered by their owners. A scenario test (`scenarios/`, public API only)
walks a Monk from level 1 to 5, spends Focus Points, deflects an attack and rests. Mutation testing
(`doc/rules/testing.md` §8): changed the `character` package (pure domain files) lines MUST score at least 80 (every
row of tables 4.3a to 4.10c is a mutation target); use-case lines SHOULD.

## 9. Open questions and decisions

| Question | Recommendation | Status |
|---|---|---|
| Which DC applies to the Deflect redirect (p. 51) and Quivering Palm (p. 52) saves, and to the Open Hand Technique saves (Push, Topple)? | The save DC (Monk's Focus) (FR-037). For the Deflect redirect (1 Focus Point) and Quivering Palm (4) the SRD states it (p. 50: "Some features that use Focus Points require your target to make a saving throw. The save DC equals 8 plus your Wisdom modifier and Proficiency Bonus"). Open Hand Technique is triggered through Flurry of Blows and spends no Focus Point itself, so extending that DC to it is an interpretation (the only Monk save DC the SRD gives). The `DifficultyClass` used is recorded in each result (FR-060, FR-107, FR-108, FR-120) so a consumer can override. | decided (SRD for FR-060/FR-120; interpretation for FR-107/FR-108) |
| Fleet Step: does the follow-up Step of the Wind use cost a second Bonus Action? | No new cost is modelled: the SRD says you "can also use Step of the Wind immediately after that Bonus Action" (p. 52), an additional use, not a second Bonus Action. The result records availability only (FR-113); Step of the Wind keeps its own Focus Point cost and `combat` owns the action economy. | decided (interpretation) |
| Where does the opaque Quivering Palm target reference live (`character` cannot import `CombatantId`, decisions FR-012, FR-013)? | `QuiveringPalmTarget` stays here: an opaque value type with no behaviour, owned by this spec for its own handoff (shared FR-048). class-ranger.md (`AttackerKey`) and class-cleric.md keep their own; shared FR-005 makes `content` the owner of closed vocabulary only, so no shared key is forced. Unifying into one `character`-owned key is a later change to character-aggregate.md, not needed now. | decided |
| Who supplies the chosen subclass to the subclass operations? | An explicit optional `SubclassId` (FR-148); replace it if character-aggregate.md or level-advancement-and-multiclassing.md adds an accessor. | decided |

Settled (engine decisions, recorded in their requirements): FR-055, FR-104, FR-121, FR-122, FR-124,
FR-134, FR-135, FR-137, FR-149, FR-150. Not owned here: subclass change, `RestKind`, class data,
equipment, state, choices, results and signals (decisions FR-017, FR-018, FR-020, FR-022 to FR-025,
FR-042, FR-063). Tool and equipment ids are looked up through the key-listing method (FR-143).

## 10. Changelog impact

Under `Unreleased`, **Added**: the Monk's feature rules in the `character` context as pure functions
plus the use cases that roll dice or load the profile. Class and subclass data is
content-classes.md's; the shared signals carry `srd5e.class=monk` values and no Monk-specific signal
is added. Not **BREAKING**: purely additive.
