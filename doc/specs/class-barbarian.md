# Class: Barbarian

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `content-schema-and-decoder.md`, `content-source-port.md`, `content-classes.md`, `errors-and-results.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `ability-scores.md`, `observability-ports.md`, `d20-tests.md`, `derived-stats.md`, `character-aggregate.md`, `rests-and-hit-dice.md`, `feats-behaviour.md`, `weapon-properties-and-mastery.md`, `level-advancement-and-multiclassing.md`, `character-creation.md` |

## 1. Purpose and scope

Specifies the Barbarian class as SRD 5.2.1 presents it (pp. 28-30): Core Barbarian
Traits, the Features table, every feature from level 1 to 20, and Path of the
Berserker, the one subclass the SRD gives. Two parts:

1. **Content data** (`ClassDefinition`, `SubclassDefinition`, `ProgressionRow`,
   `FeatureEntry`, `AbilityRequirement`, the bulk Barbarian data and its oracle
   tables CT1 to CT3, MC, CC, F, RR, SF) is owned by content-classes.md
   (shared-types-and-ownership.md FR-025, FR-035). This spec holds no second oracle
   and defines no content type; its one check, `NewBarbarianProgression`
   (FR-108), is structural, so a host can ship house-rule class content.
2. **Behaviour in `character`**: value objects, state transitions and use cases
   for features that change a character's own state (Rage, Weapon Mastery
   choices, Reckless Attack, Brutal Strike, Relentless Rage, Persistent Rage,
   Intimidating Presence, Primal Champion). Effects resolved in `combat` appear only
   as structured data in `BarbarianEffects`.

**Cross-context note (host facts).** `character` MUST NOT depend on `combat`
(`doc/rules/architecture.md` §2; shared-types-and-ownership.md FR-013), so Hit Points,
the Reaction and combat-side conditions never appear as `combat` types.
`ConditionKind` is a `content` type and is used directly. Callers supply
`BarbarianHostFacts` (`incapacitated`, `unconscious`, `wearingHeavyArmor`,
`droppedToZeroHitPoints`, `sourceWithinFiveFeet`, `activeConditions`),
`BarbarianTurnState` (the only home of Bonus Action and Reaction availability, FR-125),
`BarbarianAttackFacts` (FR-113) and a sealed `RageEndCause` interface. Results report
"condition to end" as a set of `ConditionKind` and Hit Points to set as a plain `int`.

**Shape (shared-types-and-ownership.md FR-023).** Class logic is pure domain
functions from (state, inputs) to (new state, events), taking a
`BarbarianProgression` (or one `ProgressionRow`) and returning typed rejections
for level gating. Exactly four use cases exist, each because it needs content or
dice: `LoadBarbarianProgression`, `ChooseBarbarianWeaponMasteries`,
`AdvanceBarbarian` (content), `UseRelentlessRage` (dice).

**Out of scope:** spellcasting (none; Rage forbids casting, p. 29), other classes,
multiclassing, Experience Points and gaining a level (character-aggregate.md,
level-advancement-and-multiclassing.md); choosing the subclass (this spec records only
the features each subclass level holds, FR-054); character creation flow and
equipment application, `WeaponMasteryChoice` and the Long Rest swap
(weapon-properties-and-mastery.md); the Unarmored Defense Armor Class formula
(derived-stats.md; FR-039, FR-112 supply the basis); item, armor and weapon definitions,
feats (feats-behaviour.md); rests, Initiative, Speed, Opportunity Attacks, Emanation, Hit
Points and the `combat` side of conditions; converting "10 minutes" or "1 minute" to
rounds (no clock, FR-036, FR-038); decoding class content; other subclasses and any
Barbarian material not on SRD 5.2.1 pp. 28-30.

## 2. Source

PDF page numbers equal printed page numbers.

- SRD 5.2.1, "Barbarian" and "Barbarian Subclass: Path of the Berserker": "Core
  Barbarian Traits", "Barbarian Features" table, p. 28; "Level 1: Rage" pp. 28-29; levels
  1 to 7, p. 29; "Level 9: Brutal Strike" pp. 29-30; levels 11 to 20 and the subclass, p. 30
- SRD 5.2.1, "Proficiency" table, p. 8; "Round Down", p. 5; "Armor Class", "Only One Base
  AC", "Rolling 20 or 1", p. 7; "They Don't Stack", p. 8; "Multiclassing", p. 25
- Rules Glossary: "Incapacitated [Condition]", p. 184; "Unconscious [Condition]", p. 191;
  "Critical Hit", p. 179

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Barbarian; Path of the Berserker | Content ids (`ClassID` `barbarian`, `SubclassID` `path_of_the_berserker`) | Shape and data: content-classes.md |
| Core Barbarian Traits | Content data | Tables CT1 to CT3. Skills: choose 2 of 6 (`Skill`). Equipment: A (Greataxe, 4 Handaxes, Explorer's Pack, 15 GP) or B (75 GP); resolving names to `ItemID`s is character-creation.md's |
| Rage | Domain state (`RageState`) | Active or inactive, uses, duration; effects exposed by `BarbarianEffects` |
| Unarmored Defense | Character-owned value | `BarbarianUnarmoredDefenseBasis()` (FR-112) |
| Weapon Mastery | Value (choices) | `WeaponMasteryChoice`, weapon-properties-and-mastery.md |
| Brutal Strike effect | Typed constants | Forceful Blow, Hamstring Blow, Staggering Blow, Sundering Blow |
| Relentless Rage; Intimidating Presence | Feature state | DC 10 +5 per use; Wisdom save DC 8 + Strength modifier + Proficiency Bonus |
| Danger Sense; Fast Movement; Feral Instinct; Indomitable Might; Rage Damage; Extra Attack | `BarbarianEffects` methods | FR-028, FR-047, FR-056 to FR-058, FR-073 |
| Reckless Attack; Instinctive Pounce; Primal Knowledge | `BarbarianTurnState`; `RageActivated.pounce`; skill and substitution | FR-048 to FR-052, FR-059, FR-097 |
| Persistent Rage; Frenzy; Mindless Rage; Retaliation; Primal Champion | Rage rules; Berserker functions; `abilityChanges` | FR-071 to FR-072, FR-075, FR-077 to FR-080 |

**Go conventions in this spec** (binding vocabulary: errors-and-results.md,
shared-types-and-ownership.md). Function, method and field names written lowerCamelCase
below (`rowAt`, `activateRage`, `rageUsesRemaining`) are exported PascalCase in Go (`RowAt`,
`ActivateRage`, `RageUsesRemaining`); enum-like values written lowerCamelCase (`notABarbarian`) are typed
constants with a `String()` method. A "sealed" result or choice is an interface with
an unexported marker method, one struct per variant, handled by an exhaustive type
switch; the variant list is in its doc comment. Untrusted input is validated by
`NewX` / `ParseX`, which return `(T, error)` with a typed, `errors.As`-able error and
report every problem at once with `errors.Join`; a port failure is a returned
`*srderr.PortError` (`errors.Is` / `errors.As`); a panic carrying an error value is
reserved for programmer errors (violated precondition) and appears only in `MustX`
helpers and unreachable branches. A function that may do I/O or emit spans takes
`context.Context` first; pure rule functions do not.

## 4. Requirements

Stable ids; new ones start at FR-128. A "Source" naming section 9 or a rule file is a
package design decision, not an SRD rule.

### 4.1 Core Barbarian Traits (creation choices)

Skills and equipment (FR-004 to FR-011) are validated by character-creation.md
(FR-015 to FR-020) from the definition's `skillChoice` and `startingEquipment`; this
section fixes the Barbarian values they must produce (Tables CT1 to CT3).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-004 | A Barbarian skill selection of exactly 2 distinct skills from Animal Handling, Athletics, Intimidation, Nature, Perception and Survival MUST be accepted. The allowed set and the pick count are read from the definition's `skillChoice` (content-classes.md FR-019), so house-rule content works; these SRD values are the default, tested through the bundled data. | SRD 5.2.1, "Core Barbarian Traits", p. 28 | happy |
| FR-005 | A skill selection whose count is not 2 (0, 1, 3) or that names a skill twice MUST be rejected with a typed failure result. | SRD 5.2.1, "Core Barbarian Traits", p. 28 ("Choose 2") | unhappy (invalid input) |
| FR-006 | A skill selection naming any skill outside the six (for example Arcana) MUST be rejected with a typed failure naming the skill. | SRD 5.2.1, "Core Barbarian Traits", p. 28 | unhappy (invalid input) |
| FR-009 | Starting Equipment option A MUST select option A of the definition's `startingEquipment` (name, quantity, gold as printed; Table CT3); it resolves no `ItemID`. | SRD 5.2.1, "Core Barbarian Traits", p. 28 | happy |
| FR-010 | Option B MUST select option B (75 GP and nothing else) and nothing from option A. | SRD 5.2.1, "Core Barbarian Traits", p. 28 | happy |
| FR-011 | A choice other than A or B MUST be rejected with a typed failure, not an error. | SRD 5.2.1, "Core Barbarian Traits", p. 28 ("Choose A or B") | unhappy (invalid input) |

### 4.2 Barbarian Features table

Decoding and the oracle are content-classes.md and content-schema-and-decoder.md.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | Looking up a progression row for a level below 1 or above 20 MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error). | SRD 5.2.1, "Barbarian Features", p. 28 | unhappy (boundary) |

#### 4.2.1 Progression fixtures (one table-driven test per row)

Cross-check only (normative: content-classes.md Tables CC and F); fixtures for maxima and
gating, never asserted equal. SRD 5.2.1, "Barbarian Features", p. 28.

| Level | Prof. Bonus | Gains | Rages | Rage Damage | Weapon Mastery |
|---|---|---|---|---|---|
| 1 | +2 | Rage, Unarmored Defense, Weapon Mastery | 2 | +2 | 2 |
| 2 | +2 | Danger Sense, Reckless Attack | 2 | +2 | 2 |
| 3 | +2 | Barbarian Subclass, Primal Knowledge | 3 | +2 | 2 |
| 4 | +2 | Ability Score Improvement | 3 | +2 | 3 |
| 5 | +3 | Extra Attack, Fast Movement | 3 | +2 | 3 |
| 6 | +3 | Subclass feature | 4 | +2 | 3 |
| 7 | +3 | Feral Instinct, Instinctive Pounce | 4 | +2 | 3 |
| 8 | +3 | Ability Score Improvement | 4 | +2 | 3 |
| 9 | +4 | Brutal Strike | 4 | +3 | 3 |
| 10 | +4 | Subclass feature | 4 | +3 | 4 |
| 11 | +4 | Relentless Rage | 4 | +3 | 4 |
| 12 | +4 | Ability Score Improvement | 5 | +3 | 4 |
| 13 | +5 | Improved Brutal Strike | 5 | +3 | 4 |
| 14 | +5 | Subclass feature | 5 | +3 | 4 |
| 15 | +5 | Persistent Rage | 5 | +3 | 4 |
| 16 | +5 | Ability Score Improvement | 5 | +4 | 4 |
| 17 | +6 | Improved Brutal Strike | 6 | +4 | 4 |
| 18 | +6 | Indomitable Might | 6 | +4 | 4 |
| 19 | +6 | Epic Boon | 6 | +4 | 4 |
| 20 | +6 | Primal Champion | 6 | +4 | 4 |

Path: happy for levels 1 and 20, unhappy (boundary) elsewhere. Boundary pairs the
fixtures MUST hit (last level before, first after): Proficiency Bonus 4/5, 8/9, 12/13,
16/17; Rage Damage 8/9, 15/16; Weapon Mastery 3/4, 9/10; Rages 2/3, 5/6, 11/12,
16/17; plus the two panic rows, levels 0 and 21 (FR-014).

### 4.3 Becoming a Barbarian

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-020 | On reaching Barbarian level L (1 to 20) the character MUST gain the Class Features listed for level L. Class features held MUST be derivable from the level alone; subclass features held from the level and the chosen subclass (FR-054). | SRD 5.2.1, "Barbarian Class Features", p. 28; "Level 3: Barbarian Subclass", p. 29 | happy |

### 4.4 Rage (level 1)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-021 | Entering Rage MUST require a Bonus Action (`BarbarianTurnState.bonusActionAvailable`, FR-125) and MUST be rejected with a typed result while the character wears Heavy armor (`wearingHeavyArmor`) or when no Bonus Action is available. | SRD 5.2.1, "Level 1: Rage", p. 28 | unhappy (rule interaction) |
| FR-022 | Entering Rage MUST be rejected with a typed result when no Rage use is unexpended. | SRD 5.2.1, "Level 1: Rage", p. 28 | unhappy (state error) |
| FR-023 | Entering Rage MUST expend exactly one use and start the Rage; the maximum uses come from the Rages column (4.2.1). | SRD 5.2.1, "Level 1: Rage", p. 28 | happy |
| FR-024 | Entering Rage while a Rage is active MUST be rejected with a typed result and MUST NOT expend a use. | design decision, section 9 | unhappy (state error) |
| FR-025 | Finishing a Short Rest (`RestKind`, rests-and-hit-dice.md) MUST regain one expended Rage use, never exceeding the level's row maximum (passed to the transition, FR-116); with none expended it changes nothing. | SRD 5.2.1, "Level 1: Rage", p. 28 | unhappy (boundary) |
| FR-026 | Finishing a Long Rest MUST regain all expended Rage uses. | SRD 5.2.1, "Level 1: Rage", p. 28 | happy |
| FR-027 | While Rage is active the character MUST have Resistance to Bludgeoning, Piercing and Slashing damage (`BarbarianEffects.damageResistances`); empty with no Rage. | SRD 5.2.1, "Rage", Damage Resistance, p. 29 | happy |
| FR-028 | While Rage is active the Rage Damage bonus for the Barbarian level MUST be exposed by `BarbarianEffects.RageDamageBonusFor(BarbarianAttackFacts)` for an attack using Strength (`usesStrength`) with a weapon or Unarmed Strike (`kind`) that deals damage (`dealsDamage`); zero for any other attack or with Rage inactive (cases 4.4.1). | SRD 5.2.1, "Rage", Rage Damage, p. 29; "Barbarian Features", p. 28 | happy |
| FR-029 | While Rage is active the character MUST have Advantage on Strength checks and Strength saving throws (`BarbarianEffects.strengthAdvantage`); false with no Rage. | SRD 5.2.1, "Rage", Strength Advantage, p. 29 | happy |
| FR-030 | While Rage is active the character MUST be reported as unable to maintain Concentration and to cast spells (`cannotMaintainConcentration`, `cannotCastSpells`; the host acts on it, FR-127). | SRD 5.2.1, "Rage", No Concentration or Spells, p. 29 | happy |
| FR-031 | A Rage MUST last until the end of the character's next turn. At the end of its own turn, a Rage that began that turn MUST remain active; one that began on an earlier turn MUST end unless extended during that turn. | SRD 5.2.1, "Rage", time.Duration, p. 29 | happy |
| FR-032 | If Rage is still active on the character's next turn, any one of making an attack roll against an enemy, forcing an enemy to make a saving throw, or taking a Bonus Action (`bonusActionAvailable`) MUST extend it to the end of that turn. Each of the three triggers MUST be accepted; the Bonus Action trigger MUST be rejected when no Bonus Action is available. | SRD 5.2.1, "Rage", time.Duration, p. 29 | happy |
| FR-033 | Extending a Rage on the turn it began, extending one that is not active, or extending with a trigger when already extended that turn MUST return a typed no-effect result and MUST NOT change state. | design decision, section 9 | unhappy (state error) |
| FR-034 | A Rage MUST end early when the character dons Heavy armor. | SRD 5.2.1, "Rage", time.Duration, p. 29 | unhappy (rule interaction) |
| FR-035 | A Rage MUST end early when the character gains the Incapacitated condition (`incapacitated`; before Persistent Rage, FR-072). | SRD 5.2.1, "Rage", time.Duration, p. 29 | unhappy (rule interaction) |
| FR-036 | A Rage MUST end when maintained for 10 minutes of elapsed time, supplied by the host as a value; exactly 10 minutes MUST end it, one second below MUST NOT. | SRD 5.2.1, "Rage", time.Duration, p. 29 ("up to 10 minutes"; reading in section 9) | unhappy (boundary) |
| FR-037 | Every Rage end MUST produce a `RageEnded` event whose `RageEndReason` is one of `notExtended`, `heavyArmorDonned`, `incapacitated`, `unconscious`, `tenMinutesElapsed`; a start MUST produce `RageStarted`. Below level 15 a character gaining Unconscious ends the Rage with reason `incapacitated` (Unconscious implies Incapacitated); `unconscious` is used only from level 15 (FR-072, FR-096). | SRD 5.2.1, "Rage", time.Duration, p. 29; Rules Glossary, "Unconscious [Condition]", p. 191 | happy |
| FR-038 | Entering Rage MUST NOT read any clock; durations are driven only by elapsed-time and turn-end values in commands. | doc/rules/architecture.md §3; SRD 5.2.1, "Rage", time.Duration, p. 29 | unhappy (determinism) |
| FR-095 | Entering Rage MUST be rejected with reason `incapacitated` while the host fact `incapacitated` is true (the Rage would end at once, and Incapacitated prevents the Bonus Action); it MUST NOT expend a use. | SRD 5.2.1, "Level 1: Rage", "time.Duration", p. 29; Rules Glossary, "Incapacitated [Condition]", p. 184; section 9 | unhappy (rule interaction) |
| FR-096 | With Persistent Rage (level 15 or higher) a character gaining Incapacitated without Unconscious MUST NOT end the Rage; gaining Unconscious MUST end it with reason `unconscious`. Below level 15 `RageBecameUnconscious` ends it with reason `incapacitated` (FR-037). | SRD 5.2.1, "Level 15: Persistent Rage", p. 30; Rules Glossary, "Unconscious [Condition]", p. 191 | unhappy (rule interaction) |

#### 4.4.1 Rage Damage application cases (one test per row; verify FR-028)

Source: SRD 5.2.1, "Rage", Rage Damage, p. 29, and "Barbarian Features", p. 28.

| Label | Rage | Attack | Level | Expected | Path |
|---|---|---|---|---|---|
| RD-01 | active | weapon, Strength, deals damage | 1 | +2 | happy |
| RD-02 | active | Unarmed Strike, Strength, deals damage | 5 | +2 | happy |
| RD-03 | active | weapon, Strength | 8 | +2 | unhappy (boundary) |
| RD-04 | active | weapon, Strength | 9 | +3 | unhappy (boundary) |
| RD-05 | active | weapon, Strength | 15 | +3 | unhappy (boundary) |
| RD-06 | active | weapon, Strength | 16 | +4 | unhappy (boundary) |
| RD-07 | active | weapon, Strength | 20 | +4 | happy |
| RD-08 | active | weapon, Dexterity | 9 | none | unhappy (rule interaction) |
| RD-09 | active | Strength, deals no damage | 9 | none | unhappy (rule interaction) |
| RD-10 | inactive | weapon, Strength | 9 | none | unhappy (state error) |

### 4.5 Unarmored Defense and Weapon Mastery (level 1)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-039 | `BarbarianUnarmoredDefenseBasis()` (FR-112) MUST return an `AbilityBasedArmorClassBasis` whose `abilities` are exactly `{constitution}` (the base 10 plus Dexterity is the basis's own rule), whose `ID` is `FeatureID("barbarian_unarmored_defense")`, applicable only while no body armor is worn. Armor Class values and cases are derived-stats.md FR-036, FR-037; this spec unit-tests only the returned value. | SRD 5.2.1, "Level 1: Unarmored Defense", p. 29; "Armor Class", "Only One Base AC", p. 7 | happy |
| FR-040 | The returned basis MUST have `shieldAllowed` true. | SRD 5.2.1, "Level 1: Unarmored Defense", p. 29 ("You can use a Shield and still gain this benefit") | unhappy (rule interaction) |
| FR-041 | The number of weapon kinds whose mastery the character can use MUST equal the Weapon Mastery column (2 at levels 1-3, 3 at 4-9, 4 at 10-20). | SRD 5.2.1, "Level 1: Weapon Mastery", p. 29; "Barbarian Features", p. 28 | happy |
| FR-043 | The Barbarian's eligibility rule for a Weapon Mastery kind is a Simple or Martial Melee weapon. It is supplied to `WeaponMasteryChoice`, which rejects a kind that fails it, a duplicate or a wrong count. A kind the `ContentSource` does not know (`ContentNotFound`) MUST be rejected by `ChooseBarbarianWeaponMasteries` as a typed failure (FR-103). | SRD 5.2.1, "Level 1: Weapon Mastery", p. 29 | unhappy (invalid input) |
| FR-046 | On reaching a level whose Weapon Mastery value exceeds the current number of choices (levels 4 and 10) the character MUST gain pending choice slots for the difference, each filled by a `WeaponMasteryChoice` validated against FR-043. `AdvanceBarbarian` produces them (FR-100). | SRD 5.2.1, "Level 1: Weapon Mastery", p. 29 | happy |

### 4.6 Levels 2 to 5

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-047 | From level 2 Danger Sense MUST expose Advantage on Dexterity saving throws (`BarbarianEffects.dangerSenseAdvantage`), except while `incapacitated` is true; not below level 2. | SRD 5.2.1, "Level 2: Danger Sense", p. 29 | happy |
| FR-048 | From level 2, on the first attack roll of the turn the character MAY declare Reckless Attack (`declareRecklessAttack`), giving Advantage on attack rolls using Strength until the start of its next turn and giving attack rolls against it Advantage during that time. | SRD 5.2.1, "Level 2: Reckless Attack", p. 29 | happy |
| FR-049 | Declaring Reckless Attack on any attack roll other than the first of the turn (`firstAttackRollMade`), or below level 2, MUST be rejected with a typed result. | SRD 5.2.1, "Level 2: Reckless Attack", p. 29 | unhappy (state error) |
| FR-050 | Reckless Attack MUST give no Advantage to an attack roll that does not use Strength. | SRD 5.2.1, "Level 2: Reckless Attack", p. 29 | unhappy (rule interaction) |
| FR-051 | Reaching level 3 MUST grant one additional skill proficiency from the definition's `skillChoice` set (FR-004); a skill already proficient MUST be rejected with a typed failure. | SRD 5.2.1, "Level 3: Primal Knowledge", p. 29 | unhappy (invalid input) |
| FR-052 | From level 3, while Rage is active, an ability check using Acrobatics, Intimidation, Perception, Stealth or Survival MUST be allowed to use Strength (`BarbarianEffects.strengthSkillSubstitutions`); not for other skills or with Rage inactive. | SRD 5.2.1, "Level 3: Primal Knowledge", p. 29 | unhappy (rule interaction) |
| FR-054 | The character MUST have each subclass feature whose level is at or below its Barbarian level (Path of the Berserker: 3, 6, 10, 14). | SRD 5.2.1, "Level 3: Barbarian Subclass", p. 29; "Barbarian Features", p. 28 | happy |
| FR-055 | At levels 4, 8, 12 and 16 the character MUST gain one feat choice (Ability Score Improvement or another feat it qualifies for); the catalogue is owned by the feats content specs. `AdvanceBarbarian` produces the pending grants (FR-100). | SRD 5.2.1, "Level 4: Ability Score Improvement", p. 29 | happy |
| FR-056 | From level 5 the character MUST be reported as making two attacks with the Attack action (`BarbarianEffects.attacksPerAttackAction`). | SRD 5.2.1, "Level 5: Extra Attack", p. 29 | happy |
| FR-057 | From level 5 Fast Movement MUST add 10 feet to Speed while not wearing Heavy armor (`BarbarianEffects.speedBonusFeet`), and nothing while wearing it or below level 5. | SRD 5.2.1, "Level 5: Fast Movement", p. 29 | unhappy (rule interaction) |

### 4.7 Levels 7 to 9

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-058 | From level 7 Feral Instinct MUST expose Advantage on Initiative rolls (`BarbarianEffects.initiativeAdvantage`); not below level 7. | SRD 5.2.1, "Level 7: Feral Instinct", p. 29 | happy |
| FR-059 | From level 7, entering Rage MUST also report an allowance to move up to half Speed as part of the Bonus Action (`RageActivated.pounce`, carrying Speed and half Speed rounded down; cases 4.7.1); not below level 7. | SRD 5.2.1, "Level 7: Instinctive Pounce", p. 29; "Round Down", p. 5 | happy |
| FR-060 | Brutal Strike is chosen before the attack roll ("forgo any Advantage"). From level 9 `declareBrutalStrike` MUST be accepted only when Reckless Attack was used that turn (`recklessDeclared`) and the attack roll is Strength-based, and MUST return `BrutalStrikeDeclared` with `forgoAdvantage: true` (the caller drops every Advantage source on that roll) and the effects allowed at the level (FR-068 to FR-070). Otherwise, below level 9, or for a non-Strength attack it MUST be rejected with a typed result. | SRD 5.2.1, "Level 9: Brutal Strike", p. 29 | unhappy (state error) |
| FR-061 | After the roll `resolveBrutalStrike` MUST be rejected with a typed result when the attack roll's `RollMode` (after the Advantage was forgone) is `disadvantage`. A roll with both Advantage and Disadvantage has neither and arrives as `RollMode.normal` (FR-102); it is allowed. | SRD 5.2.1, "Level 9: Brutal Strike", p. 29 ("mustn't have Disadvantage") | unhappy (rule interaction) |
| FR-062 | If the Brutal Strike attack hits (`BarbarianAttackFacts.hit`), `resolveBrutalStrike` MUST return an extra 1d10 `dice.Expression` of the weapon's or Unarmed Strike's `DamageType` (2d10 from level 17), without rolling it (FR-114). The character MAY also cause one effect: the reported list is zero or one effect (a miss reports no damage and no effects). Choosing no effect is valid. | SRD 5.2.1, "Level 9: Brutal Strike", p. 29; "Level 17: Improved Brutal Strike", p. 30 | happy |
| FR-063 | Forceful Blow MUST be exposed as: the target is pushed 15 feet straight away, after which the character can move up to half its Speed (rounded down) straight toward the target without provoking Opportunity Attacks. | SRD 5.2.1, "Level 9: Brutal Strike", Forceful Blow, p. 29; "Round Down", p. 5 | happy |
| FR-064 | Hamstring Blow MUST be exposed as: the target's Speed is reduced by 15 feet until the start of the character's next turn; a target can be affected by only one Hamstring Blow at a time, the most recent. | SRD 5.2.1, "Level 9: Brutal Strike", Hamstring Blow, p. 30 | happy |

#### 4.7.1 Half Speed rounding cases (one test per row; verify FR-059, FR-063)

Source: SRD 5.2.1, "Round Down", p. 5; "Level 7: Instinctive Pounce", p. 29.

| Label | Speed (ft) | Half Speed (ft) | Path |
|---|---|---|---|
| HS-01 | 30 | 15 | happy |
| HS-02 | 25 | 12 | unhappy (boundary: one-half rounds down) |
| HS-03 | 0 | 0 | unhappy (boundary: minimum) |
| HS-04 | 1 | 0 | unhappy (boundary) |
| HS-05 | 40 | 20 | happy |

### 4.8 Levels 11 to 20

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-065 | From level 11, when the character drops to 0 Hit Points while Rage is active and does not die outright (`droppedToZeroHitPoints`), it MAY use Relentless Rage: a Constitution saving throw against the current DC. On success Hit Points MUST become twice the Barbarian level, reported as a plain `int` for the host to apply; on failure nothing is restored. `UseRelentlessRage` MUST take a `RollMode` and the `RollModifiers` in its command, roll 1 d20 (2 for advantage or disadvantage) through its own `dice.Roller`, build the `Roll` and call `ResolveCheck(SavingThrow, roll, modifiers, dc)` (d20-tests.md FR-069); the modifiers are built by derived-stats.md (Constitution modifier, and the Proficiency Bonus only with Constitution save proficiency, which a multiclass Barbarian lacks, SRD 5.2.1, "As a Multiclass Character", p. 28), MUST add nothing to it, and reads the `D20TestOutcome` (cases 4.8.1). | SRD 5.2.1, "Level 11: Relentless Rage", p. 30 | happy |
| FR-066 | The DC MUST start at 10 and increase by 5 each use after the first, whether or not the save succeeded; a Short or Long Rest MUST reset it to 10. | SRD 5.2.1, "Level 11: Relentless Rage", p. 30 | unhappy (boundary) |
| FR-067 | Relentless Rage MUST be rejected with a typed result when Rage is not active or the level is below 11. | SRD 5.2.1, "Level 11: Relentless Rage", p. 30 | unhappy (state error) |
| FR-068 | From level 13 Staggering Blow and Sundering Blow MUST be available besides FR-063 and FR-064. Staggering Blow: the target has Disadvantage on the next saving throw it makes and can't make Opportunity Attacks until the start of the character's next turn. Sundering Blow: before the start of the character's next turn, the next attack roll by another creature against the target gains +5, and an attack roll can gain only one Sundering Blow bonus. | SRD 5.2.1, "Level 13: Improved Brutal Strike", p. 30 | happy |
| FR-069 | A Brutal Strike naming Staggering Blow or Sundering Blow below level 13 MUST be rejected with a typed result. | SRD 5.2.1, "Level 13: Improved Brutal Strike", p. 30 | unhappy (boundary) |
| FR-070 | The number of effects is a permission, not a requirement: from level 17 it is 0, 1 or 2 different effects; below 17 at most 1 (FR-062). Resolution MUST reject only the same effect named twice, an effect not yet unlocked (FR-069) and more effects than the level allows. Exactly-N is NOT required (cases 4.8.3). | SRD 5.2.1, "Level 17: Improved Brutal Strike", p. 30 ("two different Brutal Strike effects") | unhappy (invalid input) |
| FR-071 | From level 15, when the character rolls Initiative it MAY regain all expended Rage uses, then MUST NOT do so again until a Long Rest; a second attempt before a Long Rest MUST be rejected with a typed result, and a Long Rest makes it available again. | SRD 5.2.1, "Level 15: Persistent Rage", p. 30 | unhappy (state error) |
| FR-072 | From level 15 a Rage MUST last 10 minutes without extension (FR-031, FR-032 no longer end it) and MUST end early only when the character has Unconscious (not merely Incapacitated), dons Heavy armor, or 10 minutes elapse. | SRD 5.2.1, "Level 15: Persistent Rage", p. 30 | unhappy (rule interaction) |
| FR-073 | From level 18, if the total of a Strength check or Strength saving throws is less than the Strength score, the score MUST be used instead (`BarbarianEffects.indomitableMightFloor`); a total equal or above MUST be unchanged; not for other abilities. | SRD 5.2.1, "Level 18: Indomitable Might", p. 30 | unhappy (boundary) |
| FR-074 | At level 19 the character MUST gain one feat choice (an Epic Boon feat or another feat it qualifies for; catalogue owned by the feats content specs). `AdvanceBarbarian` produces the pending grant (FR-100). | SRD 5.2.1, "Level 19: Epic Boon", p. 30 | happy |
| FR-075 | On reaching level 20 Strength and Constitution MUST each increase by 4 to a maximum of 25 (cases 4.8.2); the increase MUST NOT lower a score already above 25. `AdvanceBarbarian` returns the new scores as `abilityChanges` (ability, new score, cap 25) for the host to feed to `ChangeAbilityScore` (FR-101). | SRD 5.2.1, "Level 20: Primal Champion", p. 30 (above-25 reading: section 9) | unhappy (boundary) |

#### 4.8.1 Relentless Rage cases (one test per row)

Source: SRD 5.2.1, "Level 11: Relentless Rage", p. 30. Rows RR-01 to RR-05, RR-12 to RR-15 verify FR-065 and FR-066; RR-06, RR-07 FR-066; RR-08, RR-09 FR-067; RR-10, RR-11 FR-084; RR-16, RR-17 FR-065. Only the total against the DC counts for a saving throw; a natural 20 or 1 has no special effect ("Rolling 20 or 1", p. 7: attack rolls only).

| Label | Level | Rage | Prior uses since rest | Save total vs DC | Expected | Path |
|---|---|---|---|---|---|---|
| RR-01 | 11 | active | 0 | above DC 10 | success; Hit Points 22; next DC 15 | happy |
| RR-02 | 11 | active | 0 | below DC 10 | failure; Hit Points unchanged; next DC 15 | unhappy (boundary) |
| RR-03 | 11 | active | 1 | above DC 15 | success; Hit Points 22; next DC 20 | happy |
| RR-04 | 14 | active | 2 | below DC 20 | failure; next DC 25 | unhappy (boundary) |
| RR-05 | 20 | active | 0 | above DC 10 | success; Hit Points 40 | happy |
| RR-06 | 20 | active | 3 | after a Short Rest | DC 10 again | unhappy (state error) |
| RR-07 | 20 | active | 3 | after a Long Rest | DC 10 again | unhappy (state error) |
| RR-08 | 11 | inactive | 0 | n/a | rejected | unhappy (state error) |
| RR-09 | 10 | active | 0 | n/a | rejected (feature not yet gained) | unhappy (boundary) |
| RR-10 | 11 | active | 0 | natural 20, total 9 (DC 10) | failure | unhappy (rule interaction) |
| RR-11 | 11 | active | 0 | natural 1, total 12 (DC 10) | success | unhappy (rule interaction) |
| RR-12 | 11 | active | 0 | total = DC 10 | success; Hit Points 22 | unhappy (boundary: exactly the DC) |
| RR-13 | 11 | active | 0 | total 9 (DC - 1) | failure | unhappy (boundary: one below) |
| RR-14 | 11 | active | 1 | total = DC 15 | success; next DC 20 | unhappy (boundary: exactly the DC) |
| RR-15 | 11 | active | 1 | total 14 (DC - 1) | failure; next DC 20 | unhappy (boundary: one below) |
| RR-16 | 11 | active | 0 | multiclass Barbarian, `RollModifiers` without Proficiency Bonus (Constitution +2), d20 6, total 8 | failure (the use case adds no bonus) | unhappy (rule interaction) |
| RR-17 | 11 | active | 0 | initial-class Barbarian, `RollModifiers` with Proficiency Bonus +4 (Constitution +2), d20 6, total 12 | success (bonus included because supplied) | happy |

#### 4.8.2 Primal Champion cases (one test per row; verify FR-075)

Source: SRD 5.2.1, "Level 20: Primal Champion", p. 30.

| Label | Strength | Constitution | Expected Strength | Expected Constitution | Path |
|---|---|---|---|---|---|
| PC-01 | 16 | 14 | 20 | 18 | happy |
| PC-02 | 20 | 20 | 24 | 24 | happy (boundary: one below cap) |
| PC-03 | 21 | 22 | 25 | 25 | unhappy (boundary: Strength lands exactly on 25; Constitution capped from 26) |
| PC-04 | 25 | 25 | 25 | 25 | unhappy (boundary: already at cap) |
| PC-05 | 8 | 8 | 12 | 12 | happy (minimum) |
| PC-06 | 27 | 20 | 27 | 24 | unhappy (boundary: above cap unchanged) |

#### 4.8.3 Brutal Strike effect-count cases (one test per row; verify FR-062, FR-069, FR-070)

Source: SRD 5.2.1, "Level 9: Brutal Strike", p. 29; "Level 13: Improved Brutal Strike", "Level 17: Improved Brutal Strike", p. 30. The attack roll hits, has Advantage forgone, is Strength-based and follows a Reckless Attack unless stated.

| Label | Level | Effects named | Expected | Path |
|---|---|---|---|---|
| BS-01 | 9 | none | accepted; extra 1d10; no effect | happy (zero effects is legal) |
| BS-02 | 9 | Forceful Blow | accepted; extra 1d10; one effect | happy |
| BS-03 | 9 | Forceful Blow, Hamstring Blow | rejected: too many effects | unhappy (boundary) |
| BS-04 | 13 | none | accepted | happy |
| BS-05 | 13 | Staggering Blow | accepted | happy |
| BS-06 | 13 | Staggering Blow, Sundering Blow | rejected: too many below level 17 | unhappy (boundary) |
| BS-07 | 9 | Staggering Blow | rejected: not yet unlocked (FR-069) | unhappy (boundary) |
| BS-08 | 17 | none | accepted; extra 2d10 | happy |
| BS-09 | 17 | Sundering Blow | accepted; extra 2d10; one effect | happy |
| BS-10 | 17 | Forceful Blow, Sundering Blow | accepted; two different effects | happy |
| BS-11 | 17 | Forceful Blow, Forceful Blow | rejected: duplicate | unhappy (invalid input) |
| BS-12 | 17 | three different effects | rejected: too many | unhappy (boundary) |
| BS-13 | 17 | any effects, attack roll misses | accepted; no extra damage; no effects | unhappy (state error: a hit is required) |

### 4.9 Path of the Berserker

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-077 | Frenzy (`applyFrenzy`, pure): if the character uses Reckless Attack while Rage is active (`recklessDeclared`), the first target it hits on its turn with a Strength-based attack MUST take extra damage of d6s equal to the Rage Damage bonus, of the weapon's or Unarmed Strike's `DamageType`; returned as a `dice.Expression`, not rolled (FR-114; cases 4.9.1). | SRD 5.2.1, "Level 3: Frenzy", p. 30 | happy |
| FR-078 | Frenzy MUST NOT apply without Rage, without Reckless Attack that turn, for a non-Strength attack, or to a second target hit in the same turn (`frenzyTargetHit`). | SRD 5.2.1, "Level 3: Frenzy", p. 30 | unhappy (rule interaction) |
| FR-079 | While Rage is active, from level 6, the character MUST have Immunity to Charmed and Frightened (`BarbarianEffects.Immunities`, a `map[ConditionKind]struct{}`); if Charmed or Frightened when it enters Rage (`activeConditions`), that condition MUST be reported ended in `RageActivated.endedConditions` for the host to apply. | SRD 5.2.1, "Level 6: Mindless Rage", p. 30 | unhappy (rule interaction) |
| FR-080 | From level 10, when the character takes damage from a creature within 5 feet (`sourceWithinFiveFeet`), `offerRetaliation` MUST report it able to use a Reaction to make one melee attack against that creature with a weapon or Unarmed Strike; NOT offered when the source is farther or the Reaction is spent (`reactionAvailable`). | SRD 5.2.1, "Level 10: Retaliation", p. 30 | unhappy (boundary) |
| FR-081 | From level 14, Intimidating Presence as a Bonus Action MUST yield a Wisdom saving throw DC of 8 plus the Strength modifier plus the Proficiency Bonus (cases 4.9.2), for each creature of the character's choice in a 30-foot Emanation; on a failed save a creature has Frightened for 1 minute, repeating the save at the end of each of its turns and ending the effect on a success. Other creatures' saves are resolved by their own context. | SRD 5.2.1, "Level 14: Intimidating Presence", p. 30 | happy |
| FR-082 | After use it MUST NOT be usable again until a Long Rest unless the character expends a Rage use (no action) to restore it; restoring MUST be rejected with a typed result when it has not been used or no Rage use remains. | SRD 5.2.1, "Level 14: Intimidating Presence", p. 30 | unhappy (state error) |
| FR-121 | `UseIntimidatingPresence` MUST be rejected with `IntimidatingPresenceRejected` and a typed reason: `noBonusAction`, `incapacitated`, `levelTooLow` (below 14), `notBerserker`, `notRestored` (used and not restored, FR-082). A rejection MUST NOT spend anything. | SRD 5.2.1, "Level 14: Intimidating Presence", p. 30; Rules Glossary, "Incapacitated [Condition]", p. 184 | unhappy (rule interaction) |
| FR-122 | Berserker features MUST be gated by level and subclass with reasons `levelTooLow` and `notBerserker`: Frenzy from 3 (`FrenzyNotApplicable`), Mindless Rage from 6 (below it `immunities` has no Charmed or Frightened and `endedConditions` is empty), Retaliation from 10 (`RetaliationNotOffered`). The check compares the `SubclassID` with `path_of_the_berserker`; no subclass is not a Berserker. | SRD 5.2.1, "Level 3: Frenzy", "Level 6: Mindless Rage", "Level 10: Retaliation", p. 30 | unhappy (boundary) |

#### 4.9.1 Frenzy cases (one test per row; verify FR-077, FZ-04 FR-078)

Source: SRD 5.2.1, "Level 3: Frenzy", p. 30; "Barbarian Features", p. 28.

| Label | Level | Rage Damage | Expected extra damage | Path |
|---|---|---|---|---|
| FZ-01 | 3 | +2 | 2d6 | happy |
| FZ-02 | 9 | +3 | 3d6 | unhappy (boundary) |
| FZ-03 | 16 | +4 | 4d6 | unhappy (boundary) |
| FZ-04 | 3 | +2, Rage inactive | none | unhappy (state error) |

#### 4.9.2 Intimidating Presence DC cases (one test per row; verify FR-081, FR-104)

Source: SRD 5.2.1, "Level 14: Intimidating Presence", p. 30. The command carries the `ProficiencyBonus` from derived-stats.md.

| Label | Strength modifier | Proficiency Bonus | DC | Path |
|---|---|---|---|---|
| IP-01 | +3 | +5 | 16 | happy |
| IP-02 | +5 | +6 | 19 | happy |
| IP-03 | -1 | +5 | 12 | unhappy (boundary: negative modifier) |
| IP-04 | 0 | +5 | 13 | unhappy (boundary: zero modifier) |
| IP-05 | +3 | +6 (Barbarian 14 plus 6 levels elsewhere, total level 20) | 17 | unhappy (rule interaction: bonus from total level) |

### 4.10 Cross-cutting behaviour

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-083 | Every transition MUST return a new state and its events without mutating the input; collections exposed by state, progression and effects MUST be a defensive copy. | doc/rules/architecture.md §3 | unhappy (immutability) |
| FR-084 | With a fixed `dice.Roller`, `UseRelentlessRage` MUST return the same result every run. A natural 20 with total below the DC MUST fail and a natural 1 with total at or above the DC MUST succeed (RR-10, RR-11). | SRD 5.2.1, "Level 11: Relentless Rage", p. 30; "Rolling 20 or 1", p. 7; doc/rules/testing.md §4 | unhappy (determinism) |
| FR-085 | Every content-loading use case (`LoadBarbarianProgression`, `ChooseBarbarianWeaponMasteries`, `AdvanceBarbarian`; FR-094) MUST return its own `<UseCase>ContentMissing` variant carrying `MissingContent(kind, id)` (shared-types-and-ownership.md FR-018) naming the id when the `barbarian` class (or chosen subclass) is absent (`ContentNotFound`), and MUST NOT panic. | doc/rules/architecture.md §4, §6; shared-types-and-ownership.md FR-017, FR-018 | unhappy (missing content) |
| FR-086 | When the `ContentSource` throws `*srderr.PortError`, each content-loading use case MUST propagate it unchanged with no partial result. | doc/rules/architecture.md §4 | unhappy (collaborator failure) |
| FR-087 | A pure Barbarian function (and `UseRelentlessRage`) given Barbarian level 0 MUST return its own rejected or not-applicable variant with reason `notABarbarian`, not panic. Exempt: `AdvanceBarbarian`, whose 0 to 1 transition makes a character a Barbarian (FR-118). | design decision, doc/rules/architecture.md §6 | unhappy (state error) |
| FR-088 | Every use case MUST produce identical results with the no-op `telemetry.Telemetry` and a recording one. | doc/rules/observability.md §9 | unhappy (determinism) |
| FR-089 | A successful use case call MUST emit exactly the signals of section 7: its own span named in section 7 (with the shared `srd5e.character.load_content` child where it loads content), one increment of the shared `srd5e.character.feature_used` counter and one record of the shared `.duration` histogram, with the listed attribute keys only and no log. Pure functions emit nothing. | doc/rules/observability.md §5, §9; shared-types-and-ownership.md FR-042, FR-043, FR-063 | happy (telemetry) |
| FR-090 | A returned `*srderr.PortError` from `ContentSource` MUST be recorded once by the use case: span error status, one `feature_used` increment with `srd5e.outcome=failure` and no `error.type`, and one increment of the shared `srd5e.character.feature_used.errors` with `error.type` (`content_error`), no log (character-aggregate.md FR-118). A `*srderr.PortError` (port `dice`) follows FR-093. A returned `<UseCase>ContentMissing` or `<UseCase>ProgressionInvalid` MUST be recorded once as `srd5e.outcome=failure` with `srd5e.character.rejection` `content_missing` or `progression_invalid`, span error status, one counter increment, no `error.type`, no `.errors` increment and no log. | doc/rules/observability.md §7; shared-types-and-ownership.md FR-041, FR-063 | unhappy (telemetry on failure) |
| FR-091 | A rejection returned by a use case (`AdvanceBarbarianRejected`, `BarbarianMasteriesRejected`, `RelentlessRageRejected`) MUST be a normal outcome: `srd5e.outcome=failure` and `srd5e.character.rejection` naming the reason; span status not error, no `error.type`, no log. Rejections of pure functions are results, not signals. | doc/rules/observability.md §7; shared-types-and-ownership.md FR-041 | unhappy (telemetry on rejection) |
| FR-092 | A signals consistency test MUST pin the names, types, units and attribute keys and values of section 7 against `doc/observability/signals.md` and the `telemetry.go` constants, so a rename fails. | doc/rules/observability.md §9, §10 | unhappy (catalogue drift) |
| FR-093 | A `*srderr.PortError` (port `dice`) in `UseRelentlessRage` MUST be recorded by this use case alone (character-aggregate.md FR-118): span error status, one `feature_used` increment with `srd5e.outcome=failure` and no `error.type`, one `.errors` increment with `error.type=dice_roller_error`, then returned unchanged. | doc/rules/observability.md §7; character-aggregate.md FR-118; shared-types-and-ownership.md FR-044 | unhappy (telemetry) |
| FR-094 | The three content-loading use cases load the Barbarian definition and build `NewBarbarianProgression`; when the entry exists but `NewBarbarianProgression` rejects it (FR-108) the use case MUST return its own `<UseCase>ProgressionInvalid` carrying the `BarbarianProgressionError` (the joined problems), and MUST NOT panic. A definition that fails to decode never reaches the use case (shared-types-and-ownership.md FR-020). | doc/rules/architecture.md §5; shared-types-and-ownership.md FR-020 | unhappy (missing or bad content) |
| FR-097 | Reckless Attack MUST end at the start of the character's next turn: `BarbarianTurnState.startNextTurn` clears `recklessDeclared`, and the Advantage on Strength attack rolls and on attack rolls against the character MUST be absent from then on; one step before (end of the declaring turn, other creatures' turns) it MUST still be present. | SRD 5.2.1, "Level 2: Reckless Attack", p. 29 ("until the start of your next turn") | unhappy (boundary) |
| FR-098 | `BarbarianTurnState` MUST be a value object with `recklessDeclared`, `firstAttackRollMade`, `frenzyTargetHit`, `bonusActionAvailable`, `reactionAvailable`, created fresh by `startTurn` and carried in the commands of the pure functions that need it (FR-021, FR-032, FR-048, FR-049, FR-060, FR-061, FR-077, FR-078, FR-080, FR-121). It is per-turn host state, not stored on the character (section 9). | design decision, section 9 | happy |
| FR-099 | `BarbarianEffects` MUST be a named value object (methods `DamageResistances`, `RageDamageBonusFor`, `StrengthAdvantage`, `CannotMaintainConcentration`, `CannotCastSpells`, `Immunities`, `DangerSenseAdvantage`, `RecklessAdvantageOnStrengthAttacks`, `AttacksAgainstHaveAdvantage`, `StrengthSkillSubstitutions`, `AttacksPerAttackAction`, `SpeedBonusFeet`, `InitiativeAdvantage`, `IndomitableMightFloor`) returned by the pure `CalculateBarbarianEffects` from level, subclass, `RageState`, `BarbarianTurnState` and host facts; every requirement that says "exposed" is a method of it. The host passes the values to derived-stats.md and combat. | design decision, section 9 | happy |
| FR-100 | `AdvanceBarbarian` MUST, for a new level L, return the new level, features gained at L (FR-020, FR-054), pending feat grants (4, 8, 12, 16, 19; applied by feats-behaviour.md), pending Weapon Mastery slots (4, 10; FR-046), the pending subclass choice (3; made through level-advancement-and-multiclassing.md), the pending Primal Knowledge skill (3; FR-051) and the Primal Champion `abilityChanges` (20). It MUST reject a level not exactly one above the current, or outside 1-20. | SRD 5.2.1, "Barbarian Features", p. 28 | happy |
| FR-101 | The level 20 increase MUST be returned by `AdvanceBarbarian` as `abilityChanges` (ability, new score, cap 25) for the host to feed to `ChangeAbilityScore` (character-aggregate.md FR-047) with the cap set to 25 for those two abilities only, so 21 to 25 is not rejected against the default cap of 20. `AdvanceBarbarian` does not call `ChangeAbilityScore`. | SRD 5.2.1, "Level 20: Primal Champion", p. 30; character-aggregate.md FR-047 | unhappy (rule interaction) |
| FR-102 | When a Brutal Strike attack roll has both Advantage and Disadvantage it has neither ("They Don't Stack", p. 8), so Brutal Strike MUST be allowed; `resolveBrutalStrike` carries the already-resolved `RollMode` after the Advantage was forgone (cancellation is modifiers-and-advantage.md), so both sources arrive as `normal`. | SRD 5.2.1, "They Don't Stack", p. 8; "Level 9: Brutal Strike", p. 29 | unhappy (rule interaction) |
| FR-103 | `ChooseBarbarianWeaponMasteries` MUST resolve each candidate weapon kind through the `ContentSource` (published `content` language, never `character` types) and pass the resolved definitions, the allowed count (FR-041) and the eligibility rule (FR-043) to `WeaponMasteryChoice`, which validates category, melee, distinctness and count purely. An unknown kind (`ContentNotFound`) is mapped to a typed failure naming it. | doc/rules/architecture.md §4; shared-types-and-ownership.md FR-020, FR-023 | unhappy (invalid input) |
| FR-104 | `UseIntimidatingPresence` MUST take the `ProficiencyBonus` in its command (derived-stats.md, from total character level; SRD 5.2.1, "Multiclassing", p. 25) and MUST NOT derive it from the Barbarian level. | SRD 5.2.1, "Level 14: Intimidating Presence", p. 30; "Multiclassing", p. 25 | unhappy (rule interaction) |
| FR-105 | `character` use cases and domain functions MUST NOT import any `combat` type; they take the section 5 host facts as character-owned values and use `ConditionKind` from `content`. An architecture test enforces it. | doc/rules/architecture.md §2; shared-types-and-ownership.md FR-005, FR-013 | unhappy (architecture) |
| FR-106 | Exactly four use cases exist (`LoadBarbarianProgression`, `ChooseBarbarianWeaponMasteries`, `AdvanceBarbarian`, `UseRelentlessRage`), each single-purpose; no use case takes a union of unrelated commands. Everything else is a pure function. `ChooseBarbarianSkills`, `ChooseBarbarianStartingEquipment` and `ChooseBarbarianSubclass` do not exist: skills and equipment are pure functions in `BarbarianCreationChoices` (FR-109); the subclass choice is level-advancement-and-multiclassing.md (FR-053, withdrawn here). | doc/rules/architecture.md §4; shared-types-and-ownership.md FR-020, FR-023, FR-024 | happy |
| FR-107 | Result and event names MUST be unique across the `character` package API: each use case returns its own sealed `<UseCase>Result` with operation-prefixed variants (no exported `Applied`, `Rejected`, `Succeeded`, `ContentMissing`); pure-function result variants are operation-prefixed too; events are past-tense (`RageStarted`, `RageEnded`, `RelentlessRageUsed`, `RagesRegained`, `IntimidatingPresenceUsed`); no result is named equal to an event. `RestApplied`, `SubclassChosen`, `SubclassRejected` and weapon mastery events belong to their owning specs. Reasons are typed constants, never strings. | doc/rules/architecture.md §5, §6; shared-types-and-ownership.md FR-017, FR-019, FR-020, FR-021 | happy |
| FR-108 | `NewBarbarianProgression(ClassDefinition, SubclassDefinition)` MUST check structure only and return `(BarbarianProgression, error)`, the error joining a typed problem per discrepancy: class id not `barbarian`; columns `rages`, `rage_damage` or `weapon_mastery` missing; row count not 20; a feature id this spec reads (Table F) missing or at the wrong level; the subclass not `path_of_the_berserker` or lacking its four feature ids at their levels. It MUST return the progression for a structurally correct definition and MUST NOT compare any Hit Point Die, Primary Ability, proficiency, Proficiency Bonus, Rages, Rage Damage or Weapon Mastery value with the SRD (content-classes.md bundled-data checks), so a host may ship house-rule content. | doc/rules/architecture.md §3, §5; shared-types-and-ownership.md FR-025 | unhappy (missing or bad content) |
| FR-109 | The creation choices MUST be a typed `BarbarianCreationChoices` (it holds no skills or equipment: those are plain command fields that character-creation.md FR-015 to FR-020 validate against FR-004 to FR-006 and FR-009 to FR-011, rejecting with `classSkillCountWrong`, `classSkillNotOffered`, `classSkillRepeated` and `classEquipmentOptionUnknown`; Q-3 there. It holds `ragesAtLevelOne` read by character-creation.md from the progression row; the initial Weapon Mastery kinds are not in it: `ChooseBarbarianWeaponMasteries` chooses them afterwards, trigger creation) carried in `CreateCharacter` as the `ClassCreationChoices` of character-aggregate.md; `NewBarbarianCreationChoices` validates purely and returns an error joining every problem. The aggregate stores no Barbarian field. | shared-types-and-ownership.md FR-024 | happy |
| FR-110 | The persistent state MUST be `FeatureState` implementations (`RageState`, `BarbarianResources`) stored in `FeatureStates` under `FeatureStateKey[RageState]` `barbarian_rage` and `FeatureStateKey[BarbarianResources]` `barbarian_resources` (character-aggregate.md FR-093). Commands carry the values read from the keys; results return new values for the host to write back. | shared-types-and-ownership.md FR-022; character-aggregate.md FR-093 | happy |
| FR-111 | Every Barbarian domain function MUST be pure, from (state, inputs) to (new state, events), and every event an immutable past-tense value implementing `CharacterEvent` (character-aggregate.md FR-085) directly, outside the aggregate's sealed `CharacterRecordEvent` switch. | shared-types-and-ownership.md FR-019, FR-023; character-aggregate.md FR-085 | happy |
| FR-112 | `BarbarianUnarmoredDefenseBasis()` MUST be a pure function returning `AbilityBasedArmorClassBasis{ID: FeatureID("barbarian_unarmored_defense"), Abilities: {Constitution}, ShieldAllowed: true}`. The definition declares no `ArmorClassBasis` (`content` is upstream and cannot hold a `character` type; content-classes.md `FeatureFact` has no such variant). The host passes the value in `CalculateDerivedStats.alternativeBases` when the character holds the feature (derived-stats.md FR-036). | shared-types-and-ownership.md FR-013; derived-stats.md FR-036 | happy |
| FR-113 | `BarbarianAttackFacts` MUST be a character-owned value with `usesStrength`, `kind` (`BarbarianAttackKind`: `weapon`, `unarmedStrike`), `dealsDamage`, `damageType`, `rollMode` and `hit`, used by FR-028, `declareRecklessAttack`, `declareBrutalStrike`, `resolveBrutalStrike` and `applyFrenzy`. The name `AttackFacts` is not used (conditions.md owns it). | shared-types-and-ownership.md FR-013, FR-021 | happy |
| FR-114 | `resolveBrutalStrike` and `applyFrenzy` MUST NOT roll dice or take a `dice.Roller`: they return a `dice.Expression` (1d10, 2d10 from level 17, or Nd6) plus effects. On a Critical Hit the attack's damage dice are rolled twice (SRD "Critical Hit", p. 179); that doubling is applied by the combat caller that rolls the damage (shared-types-and-ownership.md FR-026). The functions are the same for a critical and a normal hit. | SRD 5.2.1, "Critical Hit", p. 179; shared-types-and-ownership.md FR-004, FR-026 | happy |
| FR-115 | `UseRelentlessRage` MUST be constructed with `dice.Roller` and `telemetry.Telemetry`. It composes the pure `ResolveCheck` (d20-tests.md FR-069) and never builds or calls `ResolveD20Test`. | doc/rules/architecture.md §4; shared-types-and-ownership.md FR-044 | happy |
| FR-116 | `BarbarianResources` MUST enforce in its constructor only `RageUsesRemaining >= 0` (panic). It holds neither level nor maximum, so every transition that depends on the maximum (`RegainOne`, `RegainAll`, `RestoreForInitiative`) takes the level's `ProgressionRow` maximum and clamps; `ExpendRageUse` returns a typed rejection at zero. A value above its level's maximum is never produced. | doc/rules/architecture.md §3; shared-types-and-ownership.md principle 4 | unhappy (invariant) |
| FR-117 | `BarbarianCreationChoices.Seed(FeatureStates)` (character-aggregate.md FR-094) MUST return states holding `RageInactive` under `barbarian_rage` and `BarbarianResources` under `barbarian_resources` with `rageUsesRemaining` equal to `ragesAtLevelOne` and other counters initial. Its second `ClassCreationChoices` method MUST return an empty `[]content.Key` (the mastery kinds are resolved by `ChooseBarbarianWeaponMasteries`, FR-103). | character-aggregate.md FR-094 | happy |
| FR-118 | `AdvanceBarbarian` MUST support the 0 to 1 transition (an existing character multiclassing in) and return freshly seeded `RageInactive` and `BarbarianResources` at the level 1 maximum, as FR-117; exempt from FR-087. Its command carries values, not a `Character`: `CharacterID`, current level (0 to 19), `BarbarianResources` (absent exactly at level 0), and the Strength and Constitution `AbilityScore`s (FR-075). | character-aggregate.md FR-094; shared-types-and-ownership.md FR-023 | unhappy (state error) |
| FR-119 | `LoadBarbarianProgression` MUST load the `barbarian` class and `path_of_the_berserker` subclass definitions through the `ContentSource`, run `NewBarbarianProgression` (FR-108) and return `BarbarianProgressionLoaded`, so a host loads once and passes the progression to every pure function. | shared-types-and-ownership.md FR-023 | happy |
| FR-120 | Every command or pure function that emits an event MUST carry the `CharacterID` and put it on the event (character-aggregate.md FR-085): `ActivateRage`, `AdvanceRage` (via `RageEnded`), `UseRelentlessRage`, `RegainRagesOnInitiative`, `ApplyBarbarianRest`, `UseIntimidatingPresence`. | character-aggregate.md FR-085 | happy |
| FR-123 | Closed variants exported through the package API MUST carry the operation prefix: `RageState` is `RageInactive` or `RageActive`; `RageEndCause` is `RageTurnEnded`, `RageHeavyArmorDonned`, `RageBecameIncapacitated`, `RageBecameUnconscious` or `RageElapsedTime(time.Duration)`. No variant is named `Active`, `Inactive` or `TurnEnded`. | shared-types-and-ownership.md FR-017; package API scan | unhappy (name collision) |
| FR-124 | Each use case MUST record its duration in seconds through the `Meter` timing operation into the shared histogram `srd5e.character.feature_used.duration` (owned by character-aggregate.md FR-118) with attributes `srd5e.class` and `srd5e.feature` only, fixed before the timed body runs. | doc/rules/observability.md §5; character-aggregate.md FR-118, FR-119 | happy (telemetry) |
| FR-125 | `bonusActionAvailable` and `reactionAvailable` MUST be fields of `BarbarianTurnState` only; `BarbarianHostFacts` MUST NOT hold them, so the two cannot disagree. | doc/rules/architecture.md §3 | unhappy (invariant) |
| FR-126 | `RelentlessRageResolved` MUST record `srd5e.outcome` equal to the `D20TestOutcome`. A failed save is `failure` yet a normal outcome: span status not error, no `error.type`, no `srd5e.character.rejection`, no `.errors` increment, no log. | doc/rules/observability.md §7; shared-types-and-ownership.md FR-041 | unhappy (telemetry on a failed save) |
| FR-127 | Every `RageActivated` MUST carry `concentrationEnds: true`, so the host reports the loss with `BreakConcentration` reason `ended` (casting-and-concentration.md FR-113, FR-119); `character` calls no spellcasting type. | SRD 5.2.1, "Rage", No Concentration or Spells, p. 29; casting-and-concentration.md FR-113 | unhappy (rule interaction) |

### 4.11 Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (Table CT1, FR-016: Primary Ability) |
| FR-002 | Withdrawn: moved to content-classes.md (Table CT1: Hit Point Die D12) |
| FR-003 | Withdrawn: moved to content-classes.md (Table CT1: saving throws) |
| FR-007 | Withdrawn: moved to content-classes.md (Table CT2: weapon training) |
| FR-008 | Withdrawn: moved to content-classes.md (Table CT2: armor training) |
| FR-012 | Withdrawn: moved to content-classes.md (FR-029: no `spellcasting` field) |
| FR-013 | Withdrawn: moved to content-classes.md (FR-038, FR-046, Table CC: the 20 rows) |
| FR-015 | Withdrawn: moved to content-classes.md (FR-038, FR-035: Proficiency Bonus column) |
| FR-016 | Withdrawn: moved to content-schema-and-decoder.md (FR-012: every problem in one pass) |
| FR-017 | Withdrawn: moved to character-creation.md and content-classes.md (Tables CT1 to CT3, F) |
| FR-018 | Withdrawn: moved to level-advancement-and-multiclassing.md and content-classes.md (Table MC, FR-026) |
| FR-019 | Withdrawn: moved to level-advancement-and-multiclassing.md and content-classes.md (Table MC, FR-027) |
| FR-042 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-014: `WeaponMasteryChoice` creation) |
| FR-044 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-015, FR-016: count validation) |
| FR-045 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-024, FR-025: Long Rest swap) |
| FR-053 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-041, FR-049: subclass choice; `AdvanceBarbarian` still reports it pending, FR-100) |
| FR-076 | Withdrawn: moved to content-classes.md (Tables SF and F: FR-046, FR-068) |

## 5. Model

Content shapes and data are content-classes.md's (no `schemaVersion`, shared-types-and-ownership.md
FR-034). Shared types are used by name only; no command takes a `Character`.
Domain (`character` package, pure domain files; pure and synchronous):

- **`BarbarianProgression`**: built by `NewBarbarianProgression` (FR-108); `rowAt(level)` throws
  panic outside 1-20 (FR-014); features gained and held at a level
  (FR-020, FR-054).
- **Host-supplied values**: `BarbarianHostFacts` (six facts, no Bonus Action or
  Reaction, FR-125; no `combat` type, FR-105), `RageEndCause` (FR-123),
  `BarbarianAttackFacts` (FR-113), `BarbarianTurnState` (FR-098; transitions
  `startTurn`, `startNextTurn` (FR-097), `recordAttackRoll`, `recordFrenzyTarget`,
  `declareReckless`, `spendBonusAction`, `spendReaction`).
- **`RageState`**: `FeatureState`, sealed interface (unexported marker method): `RageInactive` | `RageActive`
  (`beganThisTurn`, `extendedThisTurn`, `elapsed` supplied by the caller);
  transitions `enter`, `extend`, `endOwnTurn`, `advance`, `endEarly(RageEndCause)`
  return the new state plus `RageStarted` and `RageEnded(reason)` (FR-021 to
  FR-038, FR-072, FR-095, FR-096).
- **`BarbarianResources`**: `FeatureState` with `rageUsesRemaining`,
  `relentlessRageUses` (since the last rest), `persistentRageAvailable`,
  `intimidatingPresenceAvailable`; transitions `ExpendRageUse`, `RegainOne`,
  `RegainAll`, `RestoreForInitiative`, `onRest(RestKind, max)` take the row
  maximum (FR-116). Weapon Mastery choices are not held here.
- **`BarbarianEffects`** (FR-099); **`BarbarianCreationChoices`** (FR-109, FR-117);
  **`BarbarianUnarmoredDefense`** (FR-112).
- **`BrutalStrike`**: chosen effects (`BrutalStrikeEffect`, typed constants: `ForcefulBlow`,
  `HamstringBlow`, `StaggeringBlow`, `SunderingBlow`), 0 to 2; `NewBrutalStrike` enforces
  availability, "all different" and the level maximum (FR-060 to FR-070).
- **Pure functions and services**: the functions of section 6, plus
  `IndomitableMight` (FR-073), `PrimalChampion` (FR-075), `IntimidatingPresenceDc`
  (FR-081) and the Relentless Rage DC and Hit Points (FR-065, FR-066). There is no
  Armor Class service (FR-039).
- **Events**: `RageStarted`, `RageEnded`, `RelentlessRageUsed`, `RagesRegained`,
  `IntimidatingPresenceUsed`, each carrying the `CharacterID` (FR-111, FR-120).

**Invariants and failures** (architecture.md §6). `NewBarbarianCreationChoices`, `NewBrutalStrike` and `NewBarbarianProgression`
return an error joining every problem (`errors.Join` of typed problems, FR-108).
`BarbarianResources` enforces `rageUsesRemaining >= 0` in its constructor and `rowAt`
rejects levels outside 1-20 (both panic, programmer errors). State errors
and rule interactions are typed rejections; `*srderr.PortError` comes only from ports.

## 6. Interface

**Use cases** (`character` package, use-case files): the three content-loading ones are
take a `context.Context` first (the `ContentSource` port is context-aware); `UseRelentlessRage` is synchronous (its
only port is the synchronous `dice.Roller`). Each returns its own
sealed `<UseCase>Result` (shared-types-and-ownership.md FR-017); content-loading
ones add `<UseCase>ContentMissing(MissingContent)` and
`<UseCase>ProgressionInvalid(BarbarianProgressionError)`.

| Use case | Constructor | Command carries | Specific variants | Requirements |
|---|---|---|---|---|
| `LoadBarbarianProgression` | `ContentSource`, `telemetry.Telemetry` | none | `BarbarianProgressionLoaded(progression)` | FR-085, FR-086, FR-094, FR-108, FR-119 |
| `AdvanceBarbarian` | `ContentSource`, `telemetry.Telemetry` | `CharacterID`, level (0 to 19), `BarbarianResources` (absent at 0), Strength and Constitution `AbilityScore`s | `BarbarianAdvanced(level, featuresGained, resources, rageState (seeded at 0 to 1), pendingChoices, abilityChanges, events)` / `AdvanceBarbarianRejected(reason)` | FR-020, FR-046, FR-051, FR-054, FR-055, FR-074, FR-075, FR-100, FR-101, FR-117, FR-118 |
| `ChooseBarbarianWeaponMasteries` | `ContentSource`, `telemetry.Telemetry` | level, candidate weapon kind ids, trigger (creation, level gain, after a Long Rest) | `BarbarianMasteriesChosen(choices, events)` / `BarbarianMasteriesRejected(reasons)` | FR-041, FR-043, FR-046, FR-103 |
| `UseRelentlessRage` | `dice.Roller`, `telemetry.Telemetry` | `CharacterID`, level, `BarbarianResources`, `RageState`, `BarbarianHostFacts` (`droppedToZeroHitPoints`), `RollMode`, `RollModifiers` from derived-stats.md | `RelentlessRageResolved(success or failure, hitPoints (`int`, success only), nextDc, `D20TestOutcome`, d20 roll, resources, events)` / `RelentlessRageRejected(reason)` | FR-065 to FR-067, FR-084, FR-093, FR-115, FR-126 |

`UseRelentlessRage` needs no content and is the only function that rolls a d20
(through its own `dice.Roller`, composing `ResolveCheck`); whether the character "does not die outright" is
`combat`'s, and the use case runs once `droppedToZeroHitPoints` is true.

**Pure domain functions** (the `character` package (pure domain files); no ports, no signals).
Each takes the progression or its row and the level; level 0 gives
`notABarbarian` (FR-087), a feature below its level `levelTooLow`.

| Function | Inputs | Specific variants | Requirements |
|---|---|---|---|
| `ActivateRage` | `CharacterID`, level, row, `BarbarianResources`, `RageState`, `BarbarianTurnState`, `BarbarianHostFacts` (`wearingHeavyArmor`, `incapacitated`, `activeConditions`), subclass id, Speed | `RageActivated(state, resources, turnState, events, pounce, endedConditions, concentrationEnds)` / `RageRejected(reason)` | FR-021 to FR-024, FR-059, FR-079, FR-095, FR-122, FR-127 |
| `AdvanceRage` | `CharacterID`, level, `RageState`, trigger (turn ended, extension trigger, or `RageEndCause`), `BarbarianTurnState` | `RageAdvanced(state, events)` / `RageAdvanceIgnored(reason)` | FR-031 to FR-037, FR-072, FR-096 |
| `CalculateBarbarianEffects` | level, subclass id, `RageState`, `BarbarianTurnState`, `BarbarianHostFacts` | `BarbarianEffectsCalculated(effects)` | FR-027 to FR-030, FR-047, FR-050, FR-052, FR-056 to FR-058, FR-073, FR-079, FR-099 |
| `DeclareRecklessAttack` | level, `BarbarianTurnState`, `BarbarianAttackFacts` | `RecklessDeclared(turnState, effects)` / `RecklessRejected(reason)` | FR-048 to FR-050, FR-097, FR-098 |
| `DeclareBrutalStrike` (pre-roll) | level, `BarbarianTurnState`, `BarbarianAttackFacts` | `BrutalStrikeDeclared(forgoAdvantage, allowedEffects)` / `BrutalStrikeRejected(reason)` | FR-060, FR-068, FR-069 |
| `ResolveBrutalStrike` (post-roll) | level, `BarbarianAttackFacts` (mode after forgoing, `hit`), 0 to 2 `BrutalStrikeEffect`s | `BrutalStrikeResolved(extraDamage, effects)` / `BrutalStrikeRejected(reason)` | FR-061 to FR-064, FR-068 to FR-070, FR-102, FR-114 |
| `ApplyFrenzy` | level, subclass id, `RageState`, `BarbarianTurnState`, `BarbarianAttackFacts` | `FrenzyApplied(extraDamage, turnState)` / `FrenzyNotApplicable(reason)` | FR-077, FR-078, FR-114, FR-122 |
| `OfferRetaliation` | level, subclass id, `sourceWithinFiveFeet`, `reactionAvailable` | `RetaliationOffered` / `RetaliationNotOffered(reason)` | FR-080, FR-122 |
| `RegainRagesOnInitiative` | `CharacterID`, level, row, `BarbarianResources` | `RegainRagesApplied(resources, events)` / `RegainRagesRejected(reason)` | FR-071 |
| `ApplyBarbarianRest` | `CharacterID`, level, row, `BarbarianResources`, `RestKind` | `BarbarianRestApplied(resources, events)` | FR-025, FR-026, FR-066, FR-071, FR-082 |
| `UseIntimidatingPresence` | `CharacterID`, level, subclass id, `BarbarianResources`, `RageState`, Strength modifier, `ProficiencyBonus`, `BarbarianHostFacts`, `BarbarianTurnState`, restore flag | `IntimidatingPresenceApplied(dc, resources, events)` / `IntimidatingPresenceRejected(reason)` | FR-081, FR-082, FR-104, FR-121 |

The host passes `BarbarianEffects` values to derived-stats.md and combat (for example
`BarbarianUnarmoredDefenseBasis()` in `alternativeBases`, FR-112). Ports: `ContentSource` (`ContentNotFound` for
absence, `*srderr.PortError` for failure), `dice.Roller`, `telemetry.Telemetry`. The package API
the `character` package export every type above.

## 7. Observability

Class specs define no signals of their own (shared-types-and-ownership.md FR-042,
FR-063). The use cases use those of character-aggregate.md FR-118, FR-119: counter
`srd5e.character.feature_used` (`srd5e.class`, `srd5e.feature`, `srd5e.outcome`,
`srd5e.character.rejection`), histogram `.feature_used.duration` (`srd5e.class`,
`srd5e.feature` only), counter `.feature_used.errors` (adds `error.type`). Pure
functions are **silent by design**. This spec adds values: `srd5e.class` =
`barbarian`; `srd5e.character.rejection` = the snake_case reason, plus
`content_missing`, `progression_invalid`.

| Span (`srd5e.character.`) | Use case | `srd5e.feature` |
|---|---|---|
| `load_barbarian_progression` | `LoadBarbarianProgression` | `load_progression` |
| `advance_barbarian` | `AdvanceBarbarian` | `advance` |
| `choose_barbarian_weapon_masteries` | `ChooseBarbarianWeaponMasteries` | `choose_weapon_masteries` |
| `use_relentless_rage` | `UseRelentlessRage` | `relentless_rage` |

Span attributes: `srd5e.class`, `srd5e.feature`, `srd5e.outcome`, and
`srd5e.character.rejection` on rejection and content-missing only. The three
content-loading use cases open the shared child span `srd5e.character.load_content`;
`UseRelentlessRage` has none (it composes the pure `ResolveCheck`). Rules:
FR-089 to FR-093, FR-124, FR-126; no log. Constants live in
`character/telemetry.go` (or the shared file).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.feature_used` (shared) | counter | `srd5e.class`, `srd5e.feature`, `srd5e.outcome`, `srd5e.character.rejection` | Feature use by outcome and rejection | rise in `content_missing` or `progression_invalid` |
| `srd5e.character.feature_used.duration` (shared) | histogram | `srd5e.class`, `srd5e.feature` | Use-case latency | p95 drift |
| `srd5e.character.feature_used.errors` (shared) | counter | adds `error.type` | Port and content errors | alert on any increase |

Evaluation (observability.md §2): (1) bad or missing content, a port throwing,
invalid input, a rejected rule; (2) `.errors` for errors, `feature_used` by
`srd5e.outcome` and rejection for the rest; (3) class, feature, rejection reason
and the result carry the diagnosis; (4) one counter and one duration per use
case; (5) nothing existing changes; (6) alert on any `.errors` increase and on a
rise in `content_missing` or `progression_invalid`; (7) no logs, so no noise.

## 8. Test plan

Levels: unit (`character_test` package (`*_test.go`)), application (recording `telemetrytest.Recorder`,
`dicetest.FixedRoller`, `FakeContentSource` from the `*test` packages), contract (shared
`ContentSource` suite; no new port), scenario (public API only), architecture. SRD-text
fixtures live only under `testdata/srd/`. Level-gated features need boundary rows
(one level either side) and a wrong-subclass row (testing.md §4).

| Id | Level | Notes |
|---|---|---|
| FR-004 to FR-006, FR-009 to FR-011, FR-109 | unit | bundled Barbarian data through character-creation.md `CreateCharacter`: skill counts 0, 1, 2, 3, duplicate, Arcana (`classSkillCountWrong`, `classSkillRepeated`, `classSkillNotOffered`), options A, B, C (`classEquipmentOptionUnknown`); `NewBarbarianCreationChoices` holds neither; no `ItemID` resolved |
| FR-014, 4.2.1 | unit | `rowAt` 0 and 21 panics, 1 and 20 succeed; fixtures hit every boundary level listed under 4.2.1, never equality against the definition |
| FR-108 | unit | `NewBarbarianProgression`: correct definition accepted (including house-rule Rages values); wrong class id, missing column, 19 rows, missing or misplaced feature id, wrong subclass all listed in one failure |
| FR-020, FR-054, FR-055, FR-074 | unit | Per-level feature holdings (subclass levels 3, 6, 10, 14); feat grants at 4, 8, 12, 16 and 19 only |
| FR-021 to FR-024, FR-095 | unit | `ActivateRage`: Heavy armor, no use, already raging, Incapacitated, no Bonus Action rejections |
| FR-025, FR-026, FR-116 | unit | Short Rest at 0, 1, max uses (clamped); Long Rest; constructor rejects negative uses; `ExpendRageUse` at zero rejected |
| FR-027 to FR-030, FR-028 table 4.4.1 | unit | `BarbarianEffects` methods present while active, absent otherwise; one test per RD row |
| FR-031 to FR-038, FR-096 | unit | Turn-end sequences, three extension triggers, Bonus Action trigger rejected, invalid extensions; early-end reasons; Unconscious maps to `incapacitated` below 15 and `unconscious` at 15; 10-minute boundary at 9:59 and 10:00; events; no clock (architecture check bans `time.Now(`) |
| FR-039, FR-040, FR-112 | unit, scenario | `BarbarianUnarmoredDefenseBasis()` equals the expected `AbilityBasedArmorClassBasis`; Armor Class values are derived-stats.md's; scenario feeds it through `CalculateDerivedStats.alternativeBases` |
| FR-041, FR-043, FR-046 | unit, application | Weapon Mastery count per level (3/4, 9/10); Simple or Martial Melee rule over weapon fixtures; slots at 4 and 10 via `AdvanceBarbarian` |
| FR-103 | application | Unknown kind returns the typed failure; resolved kinds, count and rule passed to `WeaponMasteryChoice` |
| FR-047 to FR-052, FR-097 | unit | Level gating (1/2); first-attack-only; Strength-only; skill exclusion; substitution list; Reckless Attack present at the end of the declaring turn, absent at the next turn start |
| FR-056 to FR-059 | unit | Gating (4/5, 6/7); Heavy armor cases for Fast Movement; table 4.7.1 |
| FR-060 to FR-064, FR-102 | unit | `declareBrutalStrike` gating (8/9, no Reckless Attack, non-Strength) returns `forgoAdvantage`; `disadvantage` rejected and `normal` allowed; returns a `dice.Expression`, never rolls |
| FR-068 to FR-070 | unit | Table 4.8.3: zero, one, two effects at levels 9, 12/13, 16/17; duplicates, locked effects, too many |
| FR-114 | unit, scenario | `resolveBrutalStrike` and `applyFrenzy` take no `dice.Roller` (signature test) and return the same expression for critical and normal hits; scenario with `dicetest.FixedRoller` shows the combat caller rolling doubled dice |
| FR-065 to FR-067, FR-084, FR-115, FR-126 | unit, application | Table 4.8.1 one test per row including RR-10 to RR-17; levels 10/11 boundary; 1 d20 for `normal`, 2 for `advantage` or `disadvantage` from `dicetest.FixedRoller`; constructor takes `dice.Roller` and `telemetry.Telemetry` only (no `ResolveD20Test`); a failed save has `srd5e.outcome=failure`, no error status, no `error.type`, no rejection attribute, no `.errors` |
| FR-093 | application | Failing `dice.Roller`: span error, error returned unchanged, `feature_used` once with `failure` and no `error.type`, `.errors` once with `dice_roller_error`; nothing in `srd5e.d20_test.*` |
| FR-071, FR-072, FR-073 | unit | Level 14/15 boundary; once until Long Rest; Persistent Rage end reasons; Indomitable Might equal, below, above, non-Strength, 17/18 boundary |
| FR-075, FR-101 | unit, application | Table 4.8.2; `abilityChanges` carry cap 25 for Strength and Constitution only; `AdvanceBarbarian` does not call `ChangeAbilityScore` |
| FR-100, FR-117, FR-118 | application | Pending grants and slots at 3, 4, 8, 10, 12, 16, 19, 20; rejects a level skip and above 20; 0 to 1 returns seeded `RageInactive` and `BarbarianResources`; `seed` yields `ragesAtLevelOne` uses; content keys slices returned as defensive copies; command holds values |
| FR-077, FR-078, FR-122 | unit | Table 4.9.1; non-applicable cases; level 2/3, 5/6, 9/10 boundaries; wrong subclass |
| FR-079, FR-080, FR-127 | unit | Immunity and ended-condition data; 5-foot boundary; Reaction spent |
| FR-081, FR-082, FR-104, FR-121 | unit | Table 4.9.2 including IP-05; restore by expending a Rage use; every `IntimidatingPresenceRejected` reason (levels 13/14) |
| FR-083, FR-098, FR-099, FR-110, FR-111, FR-113, FR-120, FR-125 | unit | `bonusActionAvailable` and `reactionAvailable` exist on `BarbarianTurnState` only; `BarbarianHostFacts` has neither; inputs not mutated, returned slices are defensive copies (mutating one leaves the state unchanged); `BarbarianTurnState` transitions; every `BarbarianEffects` accessor; `RageState` and `BarbarianResources` round-trip through `FeatureStates` under the two keys; events implement `CharacterEvent` and carry the id; `BarbarianAttackFacts` fields |
| FR-085, FR-086, FR-094, FR-119 | application | `ContentNotFound` gives `<UseCase>ContentMissing`; malformed entry gives `<UseCase>ProgressionInvalid`; failing double raises its typed `*srderr.PortError`; `LoadBarbarianProgression` happy path |
| FR-087, FR-088 | unit, application | Level 0 gives `notABarbarian` (`AdvanceBarbarian` 0 to 1 does not); same result with no-op and recording `telemetrytest.Recorder` |
| FR-089 to FR-092, FR-124 | application, unit | Recording doubles: the span names of section 7 and the `load_content` child, one `feature_used` increment and one duration record with exactly the listed keys and no log; `ContentSource` error: error span, `.errors` with `content_error`; content missing and invalid: `failure`, rejection value, no `error.type`; rejection: no error status; catalogue consistency test; pure functions emit nothing |
| FR-105 to FR-107, FR-123 | architecture, unit | `character` imports no `combat` type; exactly four use cases; unique result, event and sealed variant names in the package API (including `RageInactive`, `RageActive`, `RageTurnEnded`, `RageHeavyArmorDonned`, `RageBecameIncapacitated`, `RageBecameUnconscious`, `RageElapsedTime`, `BarbarianAttackFacts`, and no `AttackFacts`) |
| Whole slice | scenario | Through the public packages (`github.com/Whealinghold/srd5e-go/...`): a level 11 Barbarian rages, extends, the host supplies `droppedToZeroHitPoints`, Relentless Rage is used |

- **Mutation testing:** changed the `character` package (pure domain files) lines at score 80 or more
  (`doc/rules/testing.md` §8; `gomu` runs only inside Docker with `--cpus` and `--workers` capped). **Not covered:** none; withdrawn ids are tested in their owners.

## 9. Open questions and decisions

Open: none. Settled (cited by requirements):

| Decision | Reading |
|---|---|
| Exactly 10 minutes of Rage (FR-036) | "Up to 10 minutes" is a limit, so exactly 10 ends it (draft) |
| Primal Champion on a score above 25 (FR-075) | Never lowered; "to a maximum of 25" caps the increase only (draft) |
| Rage while active or Incapacitated; extension no-ops; turn state and effects (FR-024, FR-033, FR-095, FR-098, FR-099) | Design decisions from p. 29; typed results keep a caller from double-counting |
| Concentration when Rage starts (FR-127) | casting-and-concentration.md FR-113, FR-119: the host reports it with `BreakConcentration` reason `ended` |
| Skills and equipment validation (FR-004 to FR-011, FR-109) | Owned by character-creation.md FR-015 to FR-020 (its Q-3); no Barbarian-side duplicate |
| `*srderr.PortError` (port `dice`) recorder; span names (FR-093, FR-089) | The use case alone (character-aggregate.md FR-118); Fighter naming convention |

## 10. Changelog impact

`Added`: the Barbarian class behaviour (Rage, `BarbarianEffects`, Reckless Attack,
Brutal Strike, Frenzy, Retaliation, Intimidating Presence, Relentless Rage, the
Unarmored Defense basis), four use cases and the shared `feature_used` signals. Not
**BREAKING**. Bundled data is content-classes.md's (`NOTICE.md`, shared-types-and-ownership.md FR-033).
