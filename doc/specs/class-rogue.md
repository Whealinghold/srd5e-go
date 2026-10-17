# Class: Rogue

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `ability-scores.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `content-classes.md`, `derived-stats.md`, `character-aggregate.md`, `rests-and-hit-dice.md`, `feats-behaviour.md`, `weapon-properties-and-mastery.md`, `level-advancement-and-multiclassing.md`, `character-creation.md`, `spell-scroll-use.md`, `tool-proficiency-and-crafting.md` |

Binding decisions: shared-types-and-ownership.md (they win over this spec).
Content data (Core Rogue Traits, Rogue Features table, Thief subclass) is owned
by `content-classes.md` (decisions FR-025). `character` is upstream of
`combat` (decisions FR-013, FR-014, FR-016): `combat` specs consume this spec's
outputs, and this spec names only `content`, `d20` and the kernel packages
published language. `spellcasting.md` is not a dependency (this spec's FR-008).

> Go notation used below. A name in lowerCamelCase (a field, command argument or enum member) stands for the exported PascalCase Go identifier (`attackerID` is the field `AttackerID`). A trailing `?` marks an optional value (the zero value or a `(T, bool)` pair). `[]T` is a slice; `set of T` is a collection without duplicates. A `(T, error)` pair replaces an accepted/rejected result pair: the nil-error case is the accepted case and the error is the typed rejection, satisfying `errors.Is(err, srderr.ErrInvalidInput)`; a parser reports every problem at once through `errors.Join`. Rule outcomes (refusals, not-possible stops) stay result values. A use case is a struct with `Run(ctx, cmd)`; its `error` is non-nil only for a port failure.

## 1. Purpose and scope

Specifies the Rogue's **character behaviour** as the SRD presents it: what a
Rogue gains per level and as a multiclass entry, which choices a level requires
and how they are validated, Sneak Attack eligibility and dice, Cunning Strike
effects and cost, and each other class and Thief feature computable without a
rolled value or game state owned elsewhere. Everything is **domain only**: pure
functions and value objects, no ports and no use case (decisions FR-023). Rolls
(Sneak Attack damage, saves, checks, the Charges die) are made elsewhere and
arrive as values; this spec yields structured descriptions of effects for
`combat` and `d20` to apply.

**Out of scope:**

- Spellcasting: the Rogue has no Spellcasting or Pact Magic feature or slots in
  SRD 5.2.1 (p. 62); see FR-008. Any subclass other than the Thief (the only
  one in the SRD, p. 64).
- Text of other features, items and feats Rogue features name: the Ability
  Score Improvement and Epic Boon feats and Boon of the Night Spirit (feats-behaviour.md),
  Poisoner's Kit, Thieves' Tools, weapon Mastery properties, the
  Dash, Disengage, Hide, Utilize and Magic actions, and the Poisoned, Prone,
  Unconscious, Blinded, Invisible and Incapacitated conditions (`combat`
  condition specs; `ConditionKind` is `content` vocabulary, decisions FR-005).
- Resolving a Cunning Strike save, applying its condition, its 1 minute
  duration and repeat saves (`combat`, d20-tests.md); the Sneak Attack damage
  roll, Critical Hit doubling and Hit Point loss; Reactions, Bonus Actions, turn
  order, movement and grid distance (`combat`; facts arrive as inputs).
- Weapon and tool data, including Finesse or Light (`content` equipment specs;
  SRD 5.2.1, "Equipment", pp. 89-91); Weapon Mastery validation and swap
  (weapon-properties-and-mastery.md). The default attunement limit (the SRD text
  read states only the Thief's four, FR-095).
- Multiclass prerequisites, including Dexterity 13 (SRD 5.2.1, "Multiclassing",
  p. 24): level-advancement-and-multiclassing.md. `Expertise` invariants:
  derived-stats.md (`NewSkillProficiencies` FR-016, doubling FR-015);
  this spec adds only the windows (FR-023 to FR-030, FR-106).
- The Rogue content entry, bundle path and NOTICE (content-classes.md);
  persistence and display text.

## 2. Source

Printed page equals PDF page in the SRD 5.2.1 PDF (checked at pp. 61 and 64).

- SRD 5.2.1, "Rogue": "Core Rogue Traits", "Becoming a Rogue ...", "Level 1:
  Expertise", "Level 1: Sneak Attack", p. 61 (Sneak Attack continues on p. 62)
- "Rogue Features" table, "Level 1: Thieves' Cant", "Level 1: Weapon Mastery",
  "Level 2: Cunning Action", "Level 3: Rogue Subclass", "Level 3: Steady Aim",
  p. 62
- "Level 4: Ability Score Improvement", "Level 5: Cunning Strike", "Level 5:
  Uncanny Dodge", "Level 7: Evasion", "Level 7: Reliable Talent", "Level 11:
  Improved Cunning Strike", "Level 14: Devious Strikes", "Level 15: Slippery
  Mind", "Level 18: Elusive", "Level 19: Epic Boon", "Level 20: Stroke of
  Luck", p. 63
- "Rogue Subclass: Thief" (Fast Hands, Second-Story Work, Supreme Sneak, Use
  Magic Device, Thief's Reflexes), p. 64
- "Playing the Game": "Round Down", p. 5; "D20 Tests", p. 6; "Attack Rolls", p. 7;
  "Advantage/Disadvantage", "They Don't Stack", p. 8. "Combat", "Critical Hits",
  p. 16. "Character Creation", "Choose Languages" and the language tables, p. 20;
  Level Advancement and Proficiency Bonus, p. 23. "Equipment": weapon properties,
  p. 89; Mastery properties, p. 90; Tools, p. 94
- "Rules Glossary": Advantage, p. 176; Expertise, p. 182; Hide, p. 183; Invisible,
  p. 184; Long Rest and Opportunity Attacks, p. 185; Short Rest, p. 187
- Package design: shared-types-and-ownership.md (binding),
  `doc/rules/architecture.md` (§2, §3, §6), `doc/rules/observability.md` (§2,
  §3), `doc/rules/testing.md` (§4, §5, §8).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Rogue; Core Rogue Traits; Rogue Features table | `ClassID` `rogue`, the `ClassDefinition` and `ProgressionRow` / `FeatureEntry` data: `content` / content-classes.md. Behaviour: `RogueProgression`, keyed by `FeatureID` | pp. 61-62. `character` imports no content entry type, so it keeps the level gates and Sneak Attack dice; a content-release test pins them to the decoded entry (FR-009). |
| Rogue feature state key | `rogueStateKey`, a `FeatureStateKey[RogueState]` named `rogue` | character-aggregate.md FR-093. |
| Rogue level (1-20) | `RogueLevel`, `NewRogueLevel(int)` | The Rogue entry of `ClassLevels` (derived-stats.md). Every service takes a `RogueLevel`, never a raw `int` (FR-013). |
| Becoming a Rogue ... | `RogueEntry` closed: `RogueFirstClass`, `RogueMulticlass` | p. 61. |
| Expertise | `RogueExpertiseSelection` (two distinct `Skill`s, a `RogueExpertiseWindow`: `levelOne` or `levelSix`) | The host merges the skills into `SkillProficiencies.Expertise`; doubling is derived-stats.md FR-015 (p. 182). |
| Sneak Attack | `SneakAttack`; `SneakAttackResult` closed: `SneakAttackApplies`, `SneakAttackDoesNotApply` | pp. 61-62. |
| Thieves' Cant | `Language` (`content`) | Rare Languages table, p. 20; caller supplies the permitted set (FR-032). |
| Weapon Mastery | `WeaponMasteryChoice` (weapon-properties-and-mastery.md) | p. 62; this spec supplies the count, 2 (FR-111). |
| Cunning Action | `RogueGrantedAction` (`dash`, `disengage`, `hide`), `RogueGrantedActions` | p. 62. |
| Steady Aim | `SteadyAim`; `SteadyAimResult` closed | p. 62. |
| Cunning Strike; Improved Cunning Strike; Devious Strikes | `CunningStrikeSelection` (`New` constructor), `CunningStrikeEffect` enum, `CunningStrikeOutcome` closed | Poison, Trip, Withdraw (p. 63); Daze, Knock Out, Obscure (p. 63); Stealth Attack (Thief, p. 64). The two later features are level gates in the effect data. |
| Uncanny Dodge, Evasion, Reliable Talent, Elusive | `RogueUncannyDodge`, `RogueEvasion`, `RogueReliableTalent`, `RogueElusive` | p. 63. |
| Slippery Mind | `RogueProgression.SavingThrowProficiencies` | p. 63. |
| Stroke of Luck | `StrokeOfLuck`; `StrokeOfLuckState` (`available`, `spent`) in `RogueState` | p. 63 ("turn the roll into a 20"): every die of the failed `Roll` becomes 20, one `replaceDie` per die (FR-086). |
| Rogue Subclass, Thief | `SubclassID` `thief`; choosing is `ChooseSubclass` / `Subclass` with a nil error / an error (level-advancement-and-multiclassing.md) | pp. 62, 64. This spec owns `RogueProgression.SubclassLevel` (3) and the id. |
| Fast Hands, Second-Story Work, Supreme Sneak, Use Magic Device, Thief's Reflexes | `FeatureID`s `thief_fast_hands`, `thief_second_story_work`, `thief_supreme_sneak`, `thief_use_magic_device`, `thief_thiefs_reflexes` (content-classes.md FR-052) | p. 64. |
| Short Rest, Long Rest; Difficulty Class; Advantage / Disadvantage; Proficiency Bonus | `RestKind` (rests-and-hit-dice.md); `DifficultyClass` (`d20`); `RollMode`, `RollCircumstance` (kernel); `ProficiencyBonus` (derived-stats.md) | Consumed only. |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids; never renumbered. "Rogue level" means levels in the Rogue class,
not total character level. Moved requirements are in "Withdrawn ids" at the end
of this section.

### 4.1 Core Rogue Traits (content data)

The entry (`ClassDefinition`, `ProgressionRow`, `FeatureEntry`) is
content-classes.md's (decisions FR-025); only the two requirements below
belong to `character`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-008 | The Rogue MUST have no Spellcasting or Pact Magic feature and no spell slots at any level 1 to 20. | SRD 5.2.1, "Rogue Features", p. 62 (no spellcasting feature or slot column) | unhappy (empty case) |
| FR-009 | The constants of `RogueProgression` (feature ids per level, Sneak Attack dice per level, subclass level and id) MUST equal the decoded Rogue `ClassDefinition` of content-classes.md. A content-release test in the `content/data` tests decode the bundled Rogue entry through the one decode path, reads it into a test-only record (`RogueClassFacts`, in the test file, not exported) and compares it with the constants, listing every discrepancy in one failure message (wrong class id, a missing or misplaced feature id from FR-010, a wrong Sneak Attack die count, a missing Thief subclass). There is no runtime failure type: a content entry that does not decode is `ContentDecodeFailed` of content-schema-and-decoder.md. | SRD 5.2.1, "Core Rogue Traits", p. 61; "Rogue Features", p. 62; `doc/rules/architecture.md` §5 | unhappy (missing content) |

### 4.2 The Rogue Features table

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | `RogueProgression.FeaturesAt(level)` MUST return the class features gained at that Rogue level, matching the table below, for every level 1 to 20. | SRD 5.2.1, "Rogue Features", p. 62 | happy |
| FR-011 | `RogueProgression.SneakAttackDice(level)` MUST return the Sneak Attack die count (d6s) in the table below for every level 1 to 20. | SRD 5.2.1, "Rogue Features", p. 62; "Sneak Attack", p. 62 ("increases as you gain Rogue levels") | happy |
| FR-012 | The Proficiency Bonus column of the Rogue Features table MUST equal the Character Advancement Proficiency Bonus for the same level (+2 for levels 1-4, +3 for 5-8, +4 for 9-12, +5 for 13-16, +6 for 17-20). | SRD 5.2.1, "Rogue Features", p. 62; "Level Advancement", p. 23 | unhappy (rule interaction) |
| FR-013 | `NewRogueLevel(int)` MUST return `(RogueLevel, error)` (`RogueLevel` with a nil error or `RogueLevelOutOfRange(value)`), the latter for 0, a negative value and 21, and the `RogueLevel` constructor MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for them. `featuresAt`, `sneakAttackDice` and `featuresUpTo` take a `RogueLevel`, so they are never reachable with an invalid level. | SRD 5.2.1, "Rogue Features", p. 62 (levels 1-20); `doc/rules/architecture.md` §3 | unhappy (boundary) |
| FR-014 | `featuresUpTo(level)` MUST return every feature gained at Rogue levels 1 up to and including `level` and none gained above it. At a "Subclass feature" level (9, 13, 17) the subclass's own features of that level are included only when a subclass is supplied. | SRD 5.2.1, "Level 3: Rogue Subclass", p. 62 ("you gain each of your subclass's features that are of your Rogue level or lower") | unhappy (boundary) |

Rogue Features table cases (FR-010, FR-011, FR-012; one test row per level). The
`FeatureID` of each feature is derived by content-classes.md FR-052 (owner id, an
underscore, the printed name lower-cased with apostrophes deleted and other
character runs as one underscore, so `rogue_expertise`, `rogue_thieves_cant`,
`rogue_rogue_subclass`); content-classes.md is authoritative and the FR-009 test
fails if they differ. "Subclass feature" rows (9, 13, 17) are `RowSubclassFeature`
entries in content, not Rogue ids.

| Level | Proficiency Bonus | Class features (ids) | Sneak Attack |
|---|---|---|---|
| 1 | +2 | `rogue_expertise`, `rogue_sneak_attack`, `rogue_thieves_cant`, `rogue_weapon_mastery` | 1d6 |
| 2 | +2 | `rogue_cunning_action` | 1d6 |
| 3 | +2 | `rogue_rogue_subclass`, `rogue_steady_aim` | 2d6 |
| 4 | +2 | `rogue_ability_score_improvement` | 2d6 |
| 5 | +3 | `rogue_cunning_strike`, `rogue_uncanny_dodge` | 3d6 |
| 6 | +3 | `rogue_expertise` | 3d6 |
| 7 | +3 | `rogue_evasion`, `rogue_reliable_talent` | 4d6 |
| 8 | +3 | `rogue_ability_score_improvement` | 4d6 |
| 9 | +4 | Subclass feature | 5d6 |
| 10 | +4 | `rogue_ability_score_improvement` | 5d6 |
| 11 | +4 | `rogue_improved_cunning_strike` | 6d6 |
| 12 | +4 | `rogue_ability_score_improvement` | 6d6 |
| 13 | +5 | Subclass feature | 7d6 |
| 14 | +5 | `rogue_devious_strikes` | 7d6 |
| 15 | +5 | `rogue_slippery_mind` | 8d6 |
| 16 | +5 | `rogue_ability_score_improvement` | 8d6 |
| 17 | +6 | Subclass feature | 9d6 |
| 18 | +6 | `rogue_elusive` | 9d6 |
| 19 | +6 | `rogue_epic_boon` | 10d6 |
| 20 | +6 | `rogue_stroke_of_luck` | 10d6 |

### 4.3 Becoming a Rogue

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-015 | A `RogueFirstClass` entry (Rogue as the character's level 1 class) MUST grant all the Core Rogue Traits (the Rogue `ClassDefinition`, content-classes.md) and the level 1 features: Expertise, Sneak Attack, Thieves' Cant, Weapon Mastery. | SRD 5.2.1, "As a Level 1 Character", p. 61 | happy |
| FR-016 | A `RogueMulticlass` entry MUST grant exactly: the Hit Point Die (d8), proficiency in one skill of the player's choice from the Rogue's skill list, proficiency with Thieves' Tools, training with Light armor, and the level 1 features. | SRD 5.2.1, "As a Multiclass Character", p. 61 | happy |
| FR-017 | A `RogueMulticlass` entry MUST NOT grant saving throw proficiencies, Simple or Martial weapon proficiencies, or Starting Equipment. | SRD 5.2.1, "As a Multiclass Character", p. 61 (the list is exhaustive) | unhappy (rule interaction) |
| FR-018 | `NewRogueSkillChoice` for a first class entry MUST accept exactly 4 distinct skills, all from the Rogue's skill list, and otherwise return an error carrying every problem as a `RogueSkillChoiceProblem` (`wrongCount` for 0, 3 or 5; `duplicate` for a repeat; `notInRogueList`, for example Arcana). The accepted result is `RogueSkillChoice` with a nil error; both are variants of the closed `RogueSkillChoiceResult`. | SRD 5.2.1, "Core Rogue Traits", p. 61 ("Choose 4") | unhappy (invalid input) |
| FR-019 | `NewRogueSkillChoice` for a multiclass entry MUST accept exactly 1 skill from the Rogue's list and otherwise return an error (`wrongCount` for 0 or 2; `notInRogueList`). | SRD 5.2.1, "As a Multiclass Character", p. 61 | unhappy (invalid input) |
| FR-020 | A first class entry MUST record exactly one Starting Equipment option. `NewRogueStartingEquipmentChoice` takes the untrusted selection and returns `(RogueStartingEquipmentChoice, error)`: `RogueStartingEquipment` with a nil error (`RogueStartingEquipmentOption`: `optionA`, `optionB`) or `RogueStartingEquipmentChoiceMissing`. A missing choice is a failure, never defaulted. | SRD 5.2.1, "Core Rogue Traits" (Starting Equipment "Choose A or B"), p. 61 | unhappy (invalid input) |
| FR-021 | `RogueChoices.RequiredFor(entry, rogueLevel)` MUST list the choices a Rogue level gain requires, as in the table below (all 20 levels, `none` where no choice is due), and nothing else. | SRD 5.2.1, "Rogue Features" table (p. 62) and the features cited below, pp. 61-64 | happy |
| FR-022 | `NewRogueChoices` validates only the choices this spec owns (skills, Expertise, Thieves' Cant language, Starting Equipment option). Choices owned elsewhere (subclass: `ChooseSubclass`; a feat: feats-behaviour.md; Weapon Mastery: weapon-properties-and-mastery.md) appear in `requiredFor` but are validated by their owners. It MUST return `(RogueChoices, error)`: `RogueChoices` with a nil error, or an error (problems) listing every invalid or missing choice, never only the first. A problem is a closed `RogueChoiceProblem`: `RogueChoiceSkillProblem`, `RogueChoiceExpertiseProblem`, `RogueChoiceLanguageProblem` (each wrapping the operation's own problem enum) or `RogueChoiceMissing(kind)` with `kind` the enum `RogueChoiceKind`. | `doc/rules/architecture.md` §5, §6 | unhappy (invalid input) |

Required choices per level gain (FR-021; one test row per label, 21 labels
expanded from the rows below: both level 1 entries and levels 2 to 20):

| Entry and Rogue level | Required choices | Source | Path |
|---|---|---|---|
| `RogueFirstClass`, 1 | 4 skills, 2 Expertise skills, 1 language for Thieves' Cant, 2 Weapon Mastery kinds, Starting Equipment option | pp. 61-62 | happy |
| `RogueMulticlass`, 1 | 1 skill, 2 Expertise skills, 1 language, 2 Weapon Mastery kinds | pp. 61-62 | happy |
| either, 3 | Subclass, validated by `ChooseSubclass` (Steady Aim needs no choice) | p. 62 | happy |
| either, 4, 8, 10, 12, 16 | a feat (Ability Score Improvement or another qualifying feat) | p. 63 | happy |
| either, 6 | 2 further Expertise skills | p. 61 | happy |
| either, 19 | an Epic Boon feat or another qualifying feat | p. 63 | happy |
| either, 2 | none (Cunning Action needs no choice) | p. 62 | unhappy (empty case) |
| either, 5, 7, 11, 14, 15, 18, 20 | none (Cunning Strike, Uncanny Dodge; Evasion, Reliable Talent; Improved Cunning Strike; Devious Strikes; Slippery Mind; Elusive; Stroke of Luck) | p. 63 | unhappy (empty case) |
| either, 9, 13, 17 | none (Subclass feature: Supreme Sneak; Use Magic Device; Thief's Reflexes) | p. 64 | unhappy (empty case) |

### 4.4 Expertise

`RogueExpertise.Select` returns `(RogueExpertise, error)`:
`RogueExpertise` with a nil error or an error (problems), each
problem the enum `RogueExpertiseProblem` (`notProficient`,
`alreadyHasExpertise`, `duplicate`, `wrongCount`, `notDueAtThisLevel`,
`already` with a nil error, `levelOneNot` with a nil error). Windows used are a
`set of RogueExpertiseWindow`; in the cases below `{1}` reads `{levelOne}` and
`{1, 6}` reads `{levelOne, levelSix}`. The no-duplicate rule (SRD 5.2.1,
"Expertise", p. 182) is checked here against the Expertise already recorded.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-023 | `RogueExpertise.Select(rogueLevel, proficient, existing, chosen)` at Rogue level 1 MUST accept exactly two distinct skills, each in `proficient` (derived-stats.md FR-016), and return `RogueExpertise` with a nil error with the selection for window `levelOne`. | SRD 5.2.1, "Level 1: Expertise", p. 61 | happy |
| FR-024 | At Rogue level 6 `select` MUST accept exactly two more distinct skills in `proficient` and not in `existing`, returning a selection for window `levelSix`. | SRD 5.2.1, "Level 1: Expertise" ("At Rogue level 6, you gain Expertise in two more of your skill proficiencies"), p. 61 | happy |
| FR-025 | `select` MUST return an error with `notProficient` listing each chosen skill not in `proficient` (same condition as derived-stats.md FR-016; reported here so the result lists every problem, FR-022). | SRD 5.2.1, "Expertise", p. 182 ("in one skill in which you have proficiency") | unhappy (invalid input) |
| FR-026 | `select` MUST reject with `alreadyHasExpertise` a skill in `existing`, and with `duplicate` the same skill chosen twice in one selection. | SRD 5.2.1, "Expertise", p. 182 ("You can't have Expertise in the same skill proficiency more than once") | unhappy (rule interaction) |
| FR-027 | `select` MUST reject with `wrongCount` 0, 1 or 3 skills. | SRD 5.2.1, "Level 1: Expertise", p. 61 ("two") | unhappy (boundary) |
| FR-028 | A second Expertise selection MUST NOT be accepted at Rogue level 5 or lower, and MUST be accepted at level 6. | SRD 5.2.1, "Level 1: Expertise", p. 61; "Rogue Features", p. 62 | unhappy (boundary) |
| FR-029 | Expertise rules MUST key on Rogue level, not total character level: Rogue level 6 at total level 9 gets the second selection; total level 6 with Rogue level 3 does not. | SRD 5.2.1, "Rogue Features" (Rogue levels), p. 62; "Multiclassing", p. 24 | unhappy (rule interaction) |
| FR-030 | Sleight of Hand and Stealth MUST be accepted but not required: any two proficient skills are valid. | SRD 5.2.1, "Level 1: Expertise" ("are recommended"), p. 61 | happy |
| FR-106 | Selections MUST be accepted only at Rogue levels 1 and 6, once each. Any other Rogue level (2 to 5, 7 to 20) MUST be rejected with `notDueAtThisLevel`, a repeat of a used window with `already` with a nil error, and level 6 without a recorded level 1 selection with `levelOneNot` with a nil error. Choosing at level 6 the level 1 skills is rejected by FR-026. The caller passes the windows already used, so the rule has no hidden state. | SRD 5.2.1, "Level 1: Expertise", p. 61; "Rogue Features", p. 62 | unhappy (boundary) |

Expertise window cases (FR-023, FR-024, FR-028, FR-106; Rogue level, windows used gives result):

- Valid (happy): level 1 with none used; level 6 with {1}.
- `already` with a nil error (state error): level 1 with {1}; level 6 with {1, 6}.
- `levelOneNot` with a nil error (state error): level 6 with none used.
- `notDueAtThisLevel` (boundary): level 2 and level 5 with {1}; level 7 and level 20 with {1, 6}.

### 4.5 Thieves' Cant and Weapon Mastery

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | A Rogue MUST know Thieves' Cant and one other language of the player's choice from the language tables. Languages are the closed `Language` vocabulary of `content` (decisions FR-005); the caller supplies the permitted set (Standard and Rare Languages tables, p. 20). | SRD 5.2.1, "Level 1: Thieves' Cant", p. 62; "Choose Languages", p. 20 | happy |
| FR-032 | `NewRogueLanguageChoice(chosen, permittedLanguages)` MUST return `(RogueLanguageChoice, error)`: an error with `isThievesCant` when the choice is Thieves' Cant itself, or `notPermitted` when not in `permittedLanguages`; `RogueLanguageChoice` with a nil error otherwise, including a language the character already knows (the SRD text read does not forbid it; there is no languages spec). | SRD 5.2.1, "Level 1: Thieves' Cant" ("one other language"), p. 62 | unhappy (invalid input) |
| FR-111 | `RogueProgression.WeaponMasteryCount` MUST be 2 (level 1 Weapon Mastery: "two kinds of weapons of your choice with which you have proficiency"). It is passed to `WeaponMasteryChoice` validation (weapon-properties-and-mastery.md), which owns the choice type and rules. | SRD 5.2.1, "Level 1: Weapon Mastery", p. 62 | happy |

### 4.6 Sneak Attack

`SneakAttack.Evaluate` takes: `RogueLevel`, the attack's `AttackRollResult`
(d20; reads its `hit`, decisions FR-004), the weapon's Finesse property
and whether it is Ranged, the roll's net `RollMode` (already cancelled by
modifiers-and-advantage.md), the allies near the target (`AllyNearTarget`:
distance in feet and whether Incapacitated, built by `combat`), whether Sneak
Attack was already used this turn, and the weapon's damage type.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-036 | With Advantage on the attack roll and a weapon with the Finesse property, Sneak Attack MUST apply, with the die count for the Rogue level (FR-011). | SRD 5.2.1, "Level 1: Sneak Attack", pp. 61-62 | happy |
| FR-037 | With Advantage on the attack roll and a Ranged weapon, Sneak Attack MUST apply. | SRD 5.2.1, "Level 1: Sneak Attack", pp. 61-62 | happy |
| FR-038 | A weapon with neither Finesse nor Ranged MUST NOT qualify, even with Advantage or an adjacent ally; reason `weaponNotFinesseOrRanged`. | SRD 5.2.1, "Level 1: Sneak Attack", pp. 61-62 | unhappy (rule interaction) |
| FR-039 | Without Advantage and without a qualifying ally, Sneak Attack MUST NOT apply, reason `noAdvantageOrQualifyingAlly`. | SRD 5.2.1, "Level 1: Sneak Attack", pp. 61-62 | unhappy (rule interaction) |
| FR-040 | Without Advantage, Sneak Attack MUST apply when at least one ally is within 5 feet of the target, that ally is not Incapacitated, and the attacker has no Disadvantage on the attack roll. | SRD 5.2.1, "Level 1: Sneak Attack", pp. 61-62 | happy |
| FR-041 | An ally with the Incapacitated condition MUST NOT qualify; another qualifying ally still qualifies the attack. | SRD 5.2.1, "Level 1: Sneak Attack" (the ally "doesn't have the Incapacitated condition"), p. 62 | unhappy (rule interaction) |
| FR-042 | Disadvantage on the attack roll MUST remove the ally exception. | SRD 5.2.1, "Level 1: Sneak Attack" ("you don't have Disadvantage on the attack roll"), p. 62 | unhappy (rule interaction) |
| FR-043 | An ally exactly 5 feet from the target MUST qualify; an ally 10 feet away MUST NOT. | SRD 5.2.1, "Level 1: Sneak Attack" ("within 5 feet of the target"), p. 62 | unhappy (boundary) |
| FR-045 | Sneak Attack MUST NOT apply when already used this turn, reason `alreadyUsedThisTurn`. | SRD 5.2.1, "Level 1: Sneak Attack" ("Once per turn"), p. 61 | unhappy (state error) |
| FR-046 | Sneak Attack MUST NOT apply when the `AttackRollResult` is not a hit (`hit` false, `d20.Outcome.failure`), reason `attackMissed`. | SRD 5.2.1, "Level 1: Sneak Attack" ("one creature you hit with an attack roll"), p. 61 | unhappy (rule interaction) |
| FR-047 | `SneakAttackApplies` MUST carry the extra damage's type, equal to the weapon's damage type. | SRD 5.2.1, "Level 1: Sneak Attack" ("The extra damage's type is the same as the weapon's type"), p. 62 | happy |
| FR-048 | `SneakAttackApplies` MUST carry `SneakAttackDice` (die count and size d6), never a rolled total, so a Critical Hit can roll those dice twice. `SneakAttackDice.Expression()` converts to the kernel's `dice.Expression` (count d6, modifier 0) so the `combat` consumer calls `doubleDice()` (dice-and-rolls.md FR-014). | SRD 5.2.1, "Combat" ("Critical Hits"), p. 16 (doubling applied by `combat`) | unhappy (rule interaction) |
| FR-049 | `SneakAttackDoesNotApply` MUST list every failing reason, not only the first. | `doc/rules/architecture.md` §6 | unhappy (rule interaction) |

Sneak Attack cases (FR-036 to FR-043, FR-045 to FR-049; Rogue level 1, no ally, not used this turn,
attack hits, unless stated; a cancelled Advantage and Disadvantage arrives as
`normal`, p. 8; format: weapon, roll state, other inputs gives result, page):

- Applies, happy (pp. 61-62): Finesse with Advantage, 1d6; Ranged with Advantage, 1d6. Finesse, Advantage, damage type Piercing: applies with damage type Piercing (p. 62). Finesse, straight, ally at 5 ft active: applies (p. 62).
- Applies, rule interaction: Finesse, straight, one ally Incapacitated at 5 ft and one active at 5 ft (p. 62); Finesse, `normal`, ally at 5 ft active (p. 8).
- Applies, boundary: Finesse, Advantage at Rogue level 20, 10d6; at Rogue level 18, 9d6 (p. 62).
- `weaponNotFinesseOrRanged` (p. 62): neither property, Advantage.
- `noAdvantageOrQualifyingAlly`: Finesse, straight, alone (p. 62); ally at 10 ft (boundary, p. 62); ally at 5 ft Incapacitated (p. 62); Disadvantage with ally at 5 ft active (p. 62); `normal` and alone (p. 8).
- `alreadyUsedThisTurn` (state error, p. 61): Finesse, Advantage, used this turn. `attackMissed` (p. 61): Finesse, Advantage, attack misses.
- Several failures (`architecture.md` §6): neither property, straight, no ally, used this turn, misses gives all four reasons.

### 4.7 Cunning Strike

`NewCunningStrikeSelection` takes: `RogueLevel`, the subclass (or none),
the Sneak Attack result (4.6), the chosen effects in order, whether a
Poisoner's Kit is on the character's person, the Dexterity modifier and
Proficiency Bonus, and the target's size (`CreatureSize`). It returns a closed
result: the validated selection, or a failure listing every problem.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-050 | Below Rogue level 5 no Cunning Strike effect MUST be available; a selection with any effect at level 4 MUST be rejected with `effectNotAvailable`. the `New` constructor returns `(CunningStrikeSelection, error)`: `CunningStrike` with a nil error or an error (problems), each problem the enum `CunningStrikeProblem` (`effectNotAvailable`, `tooManyEffects`, `duplicateEffect`, `insufficientDice`, `sneakAttackNotDealt`, `poisonersKitMissing`). | SRD 5.2.1, "Level 5: Cunning Strike", p. 63 | unhappy (boundary) |
| FR-051 | Poison, Trip and Withdraw MUST be available from Rogue level 5. Daze, Knock Out and Obscure from Rogue level 14. Stealth Attack only to a Thief of Rogue level 9 or higher. | SRD 5.2.1, "Level 5: Cunning Strike", p. 63; "Level 14: Devious Strikes", p. 63; "Level 9: Supreme Sneak", p. 64 | unhappy (boundary) |
| FR-052 | Each effect MUST cost the die count in the effect table below. | SRD 5.2.1, pp. 63-64 (effect "Cost" entries) | happy |
| FR-053 | A selection MUST contain at most one effect below Rogue level 11 and at most two from level 11. | SRD 5.2.1, "Level 5: Cunning Strike" ("add one of the following"), "Level 11: Improved Cunning Strike" ("up to two"), p. 63 | unhappy (boundary) |
| FR-054 | The total cost MUST NOT exceed the number of Sneak Attack dice; otherwise reject with `insufficientDice`. A total equal to the dice (0d6 remaining) is allowed. Interpretation (Q1, settled): the SRD says to "remove the die before rolling" but does not state whether all dice may be forgone; the simplest rule (cost not greater than dice) has one boundary to pin. | SRD 5.2.1, "Level 5: Cunning Strike", p. 63 | unhappy (boundary) |
| FR-055 | A non-empty selection MUST be rejected with `sneakAttackNotDealt` unless the Sneak Attack result is `SneakAttackApplies`. | SRD 5.2.1, "Level 5: Cunning Strike" ("When you deal Sneak Attack damage"), p. 63 | unhappy (state error) |
| FR-056 | A valid selection MUST report the remaining Sneak Attack dice (dice minus total cost) to be rolled. | SRD 5.2.1, "Level 5: Cunning Strike" ("You remove the die before rolling"), p. 63 | happy |
| FR-057 | The DC of a Cunning Strike saving throw MUST equal 8 + the Dexterity modifier + the Proficiency Bonus (cases below). | SRD 5.2.1, "Level 5: Cunning Strike", p. 63 | happy |
| FR-058 | Poison MUST produce a `CunningStrikeConditionOnFailedSave`: save Constitution; condition `poisoned`; duration `oneMinute`; repeat save `CunningStrikeRepeatSave(Constitution, endOfEachOfTargetsTurns, endsOnSuccess: true)`; `endsOnDamage` false. It MUST be rejected with `poisonersKitMissing` when no Poisoner's Kit is on the character's person. | SRD 5.2.1, "Level 5: Cunning Strike" (Poison), p. 63 | unhappy (state error) |
| FR-059 | Trip MUST produce a `CunningStrikeConditionOnFailedSave`: save Dexterity; condition `prone`; duration `noneSpecified`; `CunningStrikeNoRepeatSave`; `endsOnDamage` false. It applies only to a target of size Large or smaller; for a larger target the outcome MUST be `CunningStrikeNoEffect(targetTooLarge)` and the cost is still paid (interpretation, Q3, settled: the selection stays a function of its inputs with no size-dependent re-pricing; the caller may pre-check the size). | SRD 5.2.1, "Level 5: Cunning Strike" (Trip: "if the target is Large or smaller"), p. 63 | unhappy (boundary) |
| FR-060 | Withdraw MUST produce a `CunningStrikeWithdrawMovement`: no saving throw, `maxFeet` equal to half the character's Speed rounded down (`halfSpeed(feet)`), `provokesOpportunityAttacks` false. | SRD 5.2.1, "Level 5: Cunning Strike" (Withdraw), p. 63; "Round Down", p. 5; "Opportunity Attacks", p. 185 | happy |
| FR-061 | Daze MUST produce a `CunningStrikeDazeOnFailedSave`: save Constitution; restriction `oneOfMoveActionBonusAction` applying for `nextTurnOfTarget`; no condition id. | SRD 5.2.1, "Level 14: Devious Strikes" (Daze), p. 63 | happy |
| FR-062 | Knock Out MUST produce a `CunningStrikeConditionOnFailedSave`: save Constitution; condition `unconscious`; duration `oneMinute`; `endsOnDamage` true; repeat save `CunningStrikeRepeatSave(Constitution, endOfEachOfTargetsTurns, endsOnSuccess: true)`. | SRD 5.2.1, "Level 14: Devious Strikes" (Knock Out), p. 63 | happy |
| FR-063 | Obscure MUST produce a `CunningStrikeConditionOnFailedSave`: save Dexterity; condition `blinded`; duration `untilEndOfTargetsNextTurn`; `CunningStrikeNoRepeatSave`; `endsOnDamage` false. | SRD 5.2.1, "Level 14: Devious Strikes" (Obscure), p. 63 | happy |
| FR-064 | Stealth Attack MUST produce a `CunningStrikeStealthAttackKeepsInvisible`: no saving throw; condition `invisible`; required cover at end of turn `threeQuartersOrTotal`. The attack does not end the Invisible condition from the Hide action if the character ends the turn behind Three-Quarters Cover or Total Cover. | SRD 5.2.1, "Level 9: Supreme Sneak", p. 64; "Hide", p. 183 | happy |
| FR-065 | Each selected effect MUST be reported as one `CunningStrikeOutcome` variant carrying its effect, cost (`SneakAttackDice`), save ability or none, the DC (when a save applies), and `timing`. No display text. | `doc/rules/architecture.md` §3, §6; SRD 5.2.1, p. 63 | happy |
| FR-066 | An empty selection MUST be valid: no cost, all Sneak Attack dice remain. | SRD 5.2.1, "Level 5: Cunning Strike" ("you can add"), p. 63 | unhappy (empty case) |
| FR-067 | Failures MUST be collected, not short-circuited: a selection with several problems returns all of them. | `doc/rules/architecture.md` §5 | unhappy (invalid input) |
| FR-068 | Every `CunningStrikeOutcome` of a valid selection MUST carry `timing` equal to the enum value `immediatelyAfterDamage`. | SRD 5.2.1, "Level 5: Cunning Strike" ("the effect occurs immediately after the attack's damage is dealt"), p. 63 | happy |
| FR-107 | A selection naming the same effect twice (Poison twice, Knock Out twice at level 14) MUST be rejected with `duplicateEffect`; each effect may be chosen at most once per Sneak Attack. Interpretation (Q2, settled): the SRD says "up to two Cunning Strike effects" and does not say whether one may repeat; Trip or Poison do not stack on one target, so no value is lost, and a duplicate-free selection keeps the validator a pure set check. | SRD 5.2.1, "Level 11: Improved Cunning Strike", p. 63 | unhappy (boundary) |

`CunningStrikeOutcome` (closed; value objects, no display text). Every variant
carries `effect` (`CunningStrikeEffect`), `cost` (`SneakAttackDice`) and
`timing` (enum `CunningStrikeTiming`, single value `immediatelyAfterDamage`).

| Variant | Fields beyond the common ones |
|---|---|
| `CunningStrikeConditionOnFailedSave` | `saveAbility` (`Ability`), `dc` (`DifficultyClass`), `conditionID` (`ConditionKind`: `poisoned`, `prone`, `unconscious`, `blinded`), `duration` (enum `CunningStrikeDuration`: `oneMinute`, `untilEndOfTargetsNextTurn`, `noneSpecified`), `repeatSave` (closed `CunningStrikeRepeatSaveRule`: `CunningStrikeNoRepeatSave`, or `CunningStrikeRepeatSave(ability, trigger, endsOnSuccess)` with `trigger` the enum `CunningStrikeRepeatTrigger` (`endOfEachOfTargetsTurns`)), `endsOnDamage` (bool) |
| `CunningStrikeDazeOnFailedSave` | `saveAbility`, `dc`, `restriction` (enum `oneOfMoveActionBonusAction`), `appliesTo` (enum `nextTurnOfTarget`) |
| `CunningStrikeWithdrawMovement` | `maxFeet` (int), `provokesOpportunityAttacks` (bool, false) |
| `CunningStrikeStealthAttackKeepsInvisible` | `conditionID` (`ConditionKind.Invisible`), `requiredCover` (enum `threeQuartersOrTotal`) |
| `CunningStrikeNoEffect` | `reason` (enum `targetTooLarge`) |

`endsOnDamage` and `endsOnSuccess` are booleans only as facts about one named
effect; condition, duration and trigger are named enums (`architecture.md` §3).

Effect table (FR-051, FR-052, FR-058 to FR-064):

| Effect | Available from | Cost | Saving throw | Source |
|---|---|---|---|---|
| Poison | Rogue level 5 | 1d6 | Constitution | p. 63 |
| Trip | Rogue level 5 | 1d6 | Dexterity | p. 63 |
| Withdraw | Rogue level 5 | 1d6 | none | p. 63 |
| Daze | Rogue level 14 | 2d6 | Constitution | p. 63 |
| Knock Out | Rogue level 14 | 6d6 | Constitution | p. 63 |
| Obscure | Rogue level 14 | 3d6 | Dexterity | p. 63 |
| Stealth Attack | Thief, Rogue level 9 | 1d6 | none | p. 64 |

Cunning Strike selection cases (FR-050 to FR-056, FR-066, FR-067, FR-107; Sneak
Attack applies and a Poisoner's Kit is carried unless stated; format: Rogue
level, Sneak Attack dice, selection gives result):

- Valid, happy: level 5, 3d6, Poison gives cost 1d6, remaining 2d6; level 5, 3d6, Withdraw gives cost 1d6, remaining 2d6; level 14, 7d6, Obscure gives cost 3d6, remaining 4d6; level 14, 7d6, Knock Out gives cost 6d6, remaining 1d6; level 19, 10d6, Knock Out and Daze gives cost 8d6, remaining 2d6; Thief level 9, 5d6, Stealth Attack gives cost 1d6, remaining 4d6.
- Valid, boundary: level 5, 3d6, empty gives cost 0, remaining 3d6 (empty case); level 11, 6d6, Trip and Withdraw gives cost 2d6, remaining 4d6; level 14, 7d6, Daze gives cost 2d6, remaining 5d6; level 14, 7d6, Knock Out and Trip gives cost 7d6, remaining 0d6 (Q1); level 5, 3d6, Trip on a Huge target gives cost 1d6 and Trip resolves `CunningStrikeNoEffect`.
- `effectNotAvailable`: level 4, 2d6, Trip (boundary); level 13, 7d6, Daze (boundary); Thief level 8, 4d6, Stealth Attack (boundary); level 9 with no subclass, 5d6, Stealth Attack (rule interaction).
- `tooManyEffects` (boundary): level 5, 3d6, Trip and Withdraw; level 10, 5d6, Trip and Withdraw.
- `duplicateEffect`: level 11, 6d6, Poison and Poison (boundary); level 14, 7d6, Knock Out and Knock Out, which also gives `insufficientDice` since 12d6 exceeds 7d6 (rule interaction).
- `insufficientDice` (boundary): level 14, 7d6, Knock Out and Daze (8d6 exceeds 7d6).
- `poisonersKitMissing` (state error): level 5, 3d6, Poison with no kit. `sneakAttackNotDealt` (state error): level 5, `SneakAttackDoesNotApply`, Trip.
- Several problems (invalid input): level 5, 3d6, Poison, Daze, no kit gives `tooManyEffects`, `effectNotAvailable` (Daze) and `poisonersKitMissing`.

Cunning Strike DC cases (FR-057; Dexterity modifier, Proficiency Bonus give DC): -5, +3 give 6; 0, +3 give 11; +3, +3 give 14; +5, +4 give 17 (level 9); +5, +6 give 19 (level 17); -5, +6 give 9; +10, +6 give 24. Modifier 0 and the ordinary rows are happy paths; the lowest and highest modifier rows are boundaries.

Withdraw movement cases (FR-060; Speed gives maximum feet): 30 gives 15 (happy); 25 gives 12, 0 gives 0 (boundary).

### 4.8 Class features

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-069 | From Rogue level 2, Cunning Action MUST grant the Bonus Action options Dash, Disengage and Hide, and no others; at level 1 it MUST grant nothing. | SRD 5.2.1, "Level 2: Cunning Action", p. 62 | unhappy (boundary) |
| FR-070 | `SteadyAim.Use(level, movedThisTurn)` from Rogue level 3 MUST return `SteadyAimGranted`: `advantageOn` = `nextAttackRollThisTurn`, `speedUntilEndOfTurn` = 0, `actionCost` = `bonusAction` (enum). The `combat` caller applies them. | SRD 5.2.1, "Level 3: Steady Aim", p. 62 | happy |
| FR-071 | `SteadyAim.Use` MUST return `SteadyAimNotAvailable` listing `movedThisTurn` when the character has moved this turn. When several reasons fail (level 2 and moved) it lists all, in the fixed order `levelTooLow`, `movedThisTurn` (FR-049, FR-067). | SRD 5.2.1, "Level 3: Steady Aim" ("only if you haven't moved during this turn"), p. 62; `doc/rules/architecture.md` §6 | unhappy (state error) |
| FR-072 | `SteadyAim.Use` MUST return `SteadyAimNotAvailable` with `levelTooLow` at Rogue level 2 or lower (with any other failing reason, FR-071). | SRD 5.2.1, "Rogue Features", p. 62 | unhappy (boundary) |
| FR-073 | `RogueUncannyDodge.Halve(damage)` MUST return the damage halved and rounded down (cases below). | SRD 5.2.1, "Level 5: Uncanny Dodge" ("(round down)"), p. 63 | happy |
| FR-074 | `RogueUncannyDodge.CanUse(level, reactionAvailable, attackerSeen, attack)` (`attack` the `AttackRollResult`) MUST return `RogueUncannyDodgeAvailable` only when Rogue level is 5 or higher, the Reaction is available, the attacker is a creature the character can see, and the attack hit; otherwise `RogueUncannyDodgeNotAvailable` listing every failing reason (`levelTooLow`, `reactionUnavailable`, `attackerNotSeen`, `attackMissed`, in that order; FR-049, FR-067). | SRD 5.2.1, "Level 5: Uncanny Dodge" ("an attacker that you can see hits you with an attack roll", "Reaction"), p. 63 | unhappy (state error) |
| FR-075 | `RogueEvasion.Apply` MUST give 0 damage on a successful Dexterity save and half the damage (rounded down) on a failed one, for an effect whose normal result is half damage on a success (cases below). | SRD 5.2.1, "Level 7: Evasion", p. 63 | happy |
| FR-076 | `RogueEvasion.Apply` MUST NOT change the damage when the character has the Incapacitated condition: the normal result (half on a success, full on a failure) stands. | SRD 5.2.1, "Level 7: Evasion" ("You can't use this feature if you have the Incapacitated condition"), p. 63 | unhappy (state error) |
| FR-077 | `RogueEvasion.Apply` MUST NOT change the damage of an effect that is not a Dexterity saving throw for half damage on a success (another ability, or no damage on a success). | SRD 5.2.1, "Level 7: Evasion" ("an effect that allows you to make a Dexterity saving throw to take only half damage"), p. 63 | unhappy (rule interaction) |
| FR-078 | `RogueReliableTalent.Apply(roll)` MUST return 10 for a d20 roll of 9 or lower and the roll otherwise, for an ability check that uses one of the character's skill or tool proficiencies at Rogue level 7 or higher (cases below). The caller passes `ToolCheckBenefits.UsesProficiency` (tool-proficiency-and-crafting.md FR-016) as that predicate; this spec defines no second one. | SRD 5.2.1, "Level 7: Reliable Talent", p. 63 | happy |
| FR-079 | `RogueReliableTalent.Apply` MUST NOT change the roll for an ability check using no skill or tool proficiency, a saving throw, an attack roll, or at Rogue level 6 or lower. | SRD 5.2.1, "Level 7: Reliable Talent", p. 63 | unhappy (rule interaction) |
| FR-080 | `RogueProgression.SavingThrowProficiencies(level, entry)` MUST return, for `RogueFirstClass`, {Dexterity, Intelligence} for Rogue levels 1 to 14 and {Dexterity, Intelligence, Wisdom, Charisma} from level 15 (Slippery Mind); for `RogueMulticlass` the empty set before level 15 (FR-017) and {Wisdom, Charisma} from level 15, because Slippery Mind is a class feature, not a multiclass trait. It MUST NOT include Wisdom or Charisma at level 14. Slippery Mind grants the proficiency as a class feature, so it applies to a multiclass Rogue 15 (Fighter 5 / Rogue 15) too (Q12, settled): derived-stats.md accepts an `additionalSavingThrows` input (a `set of Ability` of class-feature grants, unioned with the initial class's saves; FR-019 still forbids multiclass entry traits adding saves), and the host passes this result for the Rogue (the amendment is requested of derived-stats.md; until it lands the host unions the result into `SavingThrowProficiencies`). | SRD 5.2.1, "Level 15: Slippery Mind", p. 63 | unhappy (boundary) |
| FR-081 | From Rogue level 18, `RogueElusive.Apply(level, incapacitated, circumstances)`, taking and returning an `[]RollCircumstance` (modifiers-and-advantage.md FR-042; the caller maps its own roll-circumstance sources to `RollCircumstance`), MUST drop every `Advantage` from the circumstances of an attack roll against the character and keep every `Disadvantage`; at level 17 it MUST return them unchanged. | SRD 5.2.1, "Level 18: Elusive", p. 63 | unhappy (boundary) |
| FR-082 | `RogueElusive.Apply` MUST NOT drop any `Advantage` while the character has the Incapacitated condition. | SRD 5.2.1, "Level 18: Elusive" ("unless you have the Incapacitated condition"), p. 63 | unhappy (state error) |
| FR-083 | `RogueElusive.Apply` drops the `Advantage` circumstances before `RollMode.Resolve` cancels, so circumstances holding both end with Disadvantage (interpretation of ordering, Q4, settled: each step is testable alone). | SRD 5.2.1, "Level 18: Elusive" ("No attack roll can have Advantage against you"), p. 63; "They Don't Stack", p. 8 | unhappy (rule interaction) |
| FR-084 | `RogueProgression.FeatLevels` MUST return exactly {4, 8, 10, 12, 16} (Ability Score Improvement levels) and `grantsFeat(level)` MUST be true at those levels and false at every other level 1 to 20 (the level 19 Epic Boon is FR-085). The feat is chosen through the feat content (`FeatID`) and applied by feats-behaviour.md. | SRD 5.2.1, "Level 4: Ability Score Improvement", p. 63; "Rogue Features", p. 62 | unhappy (boundary) |
| FR-085 | `RogueProgression.EpicBoonLevel` MUST be 19, and `RogueChoices.RequiredFor` at level 19 MUST require a feat that is an Epic Boon or another feat for which the character qualifies (decided by the feat content; Boon of the Night Spirit is accepted but not required). | SRD 5.2.1, "Level 19: Epic Boon", p. 63 | happy |
| FR-086 | From Rogue level 20, `StrokeOfLuck.Use(characterID, level, state, test)` MUST return `StrokeOfLuckSpent` for a failed D20 Test that has a roll. `test` is the closed `StrokeOfLuckTest`: `StrokeOfLuckRolledTest(roll, outcome)` (the failed test's `Roll` and `d20.Outcome`) or `StrokeOfLuckUnrolledTest` (a forced or automatic failure with no `Roll`, d20-tests.md FR-035, FR-036). The SRD says "turn the roll into a 20" (p. 63), so the result MUST carry a `Roll` whose **every** die is 20, made by one `replaceDie(i, 20)` call per die index (d20-tests.md FR-012), so `natural` is 20 in all three roll modes; the "only one die" limit of p. 8 governs rerolls, not this feature. The result also carries the new state `spent` and a `StrokeOfLuckUsed(characterID, ...)` event: a past-tense value object implementing `CharacterEvent` (character-aggregate.md FR-085; decisions FR-019, FR-063), outside the aggregate's closed `CharacterRecordEvent` family. The new total (20 plus modifiers) is re-resolved by the caller through d20; Stroke of Luck does not itself declare success. | SRD 5.2.1, "Level 20: Stroke of Luck", p. 63 | happy |
| FR-087 | `StrokeOfLuck.Use` MUST return `StrokeOfLuckNotAvailable` listing `spent` when the feature is already `spent`; with several failing reasons (FR-089) it lists all in the fixed order `levelTooLow`, `spent`, `testNotFailed`, `noRoll`. | SRD 5.2.1, "Level 20: Stroke of Luck" ("you can't use it again until"), p. 63; `doc/rules/architecture.md` §6 | unhappy (state error) |
| FR-088 | `StrokeOfLuck.AfterRest(state, RestKind)` MUST restore a `spent` feature after a Short Rest or a Long Rest. | SRD 5.2.1, "Level 20: Stroke of Luck" ("until you finish a Short or Long Rest"), p. 63; "Short Rest", p. 187; "Long Rest", p. 185 | happy |
| FR-089 | `StrokeOfLuck.Use` MUST return `StrokeOfLuckNotAvailable` with `testNotFailed` when a `StrokeOfLuckRolledTest` has outcome `success`, with `noRoll` for a `StrokeOfLuckUnrolledTest` (no roll to turn into a 20; interpretation, Q13, settled), and `levelTooLow` at Rogue level 19 or lower (each with any other failing reason, FR-087). | SRD 5.2.1, "Level 20: Stroke of Luck" ("If you fail a D20 Test", "turn the roll into a 20"), p. 63; "D20 Tests", p. 6 | unhappy (boundary) |

Uncanny Dodge cases (FR-073; damage gives halved damage): 0 gives 0, 1 gives 0, 2 gives 1, 7 gives 3 (boundary); 8 gives 4, 31 gives 15 (happy).

Evasion cases (FR-075, FR-076, FR-077; effect deals 10 damage, half on a success unless stated; damage taken):

- Dexterity success, not Incapacitated: 0 (happy). Dexterity failure: 5 (happy). Failure with 7 damage: 3; failure with 1 damage: 0 (boundary).
- Incapacitated, Dexterity success: 5; Incapacitated, failure: 10 (state error).
- Constitution save, success: 5; Dexterity effect with no damage on a success: 0, unchanged by Evasion (rule interaction).

Reliable Talent cases (FR-078, FR-079; Rogue level 7 ability check using a skill unless stated; d20 roll gives result):

- Boundary: 1 gives 10; 9 gives 10; 10 (tool) gives 10; 11 gives 11; level 6 with roll 5 gives 5.
- Happy: natural 20 at Rogue level 20 (tool) gives 20.
- Rule interaction (roll 5 stays 5): an ability check using no proficiency, a saving throw, an attack roll.

Elusive cases (FR-081, FR-082, FR-083; resulting state after `RollMode.Resolve`):

- Level 18, Advantage, not Incapacitated: straight (happy). Level 17, Advantage: Advantage (boundary). Level 18, Advantage, Incapacitated: Advantage (state error).
- Level 18, Advantage and Disadvantage: Disadvantage (rule interaction). Level 18, Disadvantage only: Disadvantage (rule interaction).

Stroke of Luck cases (FR-086 to FR-089; Rogue level 20, state `available`, a failed test unless stated):

- Normal, dice [7]: `StrokeOfLuckSpent`; dice [20], natural 20; state `spent`; one `StrokeOfLuckUsed` carrying the `characterID` (happy). Advantage, dice [18, 3]: dice [20, 20], natural 20 (happy). Disadvantage, dice [5, 3]: dice [20, 20], natural 20; a single replaceDie would leave natural 5 or 3 (rule interaction).
- Unrolled (forced failure): `StrokeOfLuckNotAvailable`, `noRoll`, state unchanged (empty case). Rolled with outcome `success`: `testNotFailed` (rule interaction). State `spent`: `spent` (state error). Level 19: `levelTooLow` (boundary).
- Level 19, `spent`, rolled with outcome `success`: `levelTooLow`, `spent`, `testNotFailed` in that order (rule interaction).

### 4.9 Subclass and the Thief

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-090 | `RogueProgression.SubclassLevel` MUST be 3 and `subclassID` MUST be `thief`; `RogueChoices.RequiredFor` MUST require a subclass choice at level 3. The choice is validated and recorded by `ChooseSubclass` / `Subclass` with a nil error / an error (level-advancement-and-multiclassing.md; decisions FR-020); this spec has no subclass choice type. | SRD 5.2.1, "Level 3: Rogue Subclass", p. 62; "Rogue Subclass: Thief", p. 64 | happy |
| FR-092 | The Thief MUST gain Fast Hands and Second-Story Work at Rogue level 3, Supreme Sneak at 9, Use Magic Device at 13 and Thief's Reflexes at 17. | SRD 5.2.1, "Rogue Subclass: Thief", p. 64; "Rogue Features" ("Subclass feature" at 9, 13, 17), p. 62 | happy |
| FR-093 | `ThiefFeatures.FastHands(level)` from Rogue level 3 MUST return the set of `FastHandsOption`: `sleightOfHandCheck` (pick a lock or disarm a trap with Thieves' Tools, or pick a pocket), `utilizeAction`, `magicActionForMagicItem`, each as a Bonus Action; below level 3 the set is empty. | SRD 5.2.1, "Level 3: Fast Hands", p. 64 | happy |
| FR-094 | `ThiefFeatures.SecondStoryWork(level, speedFeet)` from Rogue level 3 MUST return `SecondStoryWorkGranted` with `climbSpeedFeet` equal to `speedFeet` and `jumpAbility` Dexterity; below level 3 `SecondStoryWorkNotGained`. Both are variants of the closed `SecondStoryWorkResult`. | SRD 5.2.1, "Level 3: Second-Story Work", p. 64 | happy |
| FR-095 | `UseMagicDevice.AttunementLimit(level)` MUST return `(AttunementLimit, error)`: `AttunementLimitSet(4)` for a Thief of Rogue level 13 or higher, `AttunementLimitUnchanged` before level 13. | SRD 5.2.1, "Level 13: Use Magic Device" (Attunement), p. 64 | unhappy (boundary) |
| FR-096 | `UseMagicDevice.ChargesExpended(roll)` MUST return that no charges are expended on a d6 roll of 6 and that charges are expended on 1 to 5 (cases below). | SRD 5.2.1, "Level 13: Use Magic Device" (Charges), p. 64 | unhappy (boundary) |
| FR-113 | `UseMagicDevice.ScrollRoute(level, subclass)` MUST return `AnyScrollRoute(Intelligence, reliableUpToLevel 1, checkSkill Arcana)` (spell-scroll-use.md FR-019) for a Thief of Rogue level 13 or higher and no route (`nil`) before level 13 or for any other subclass. The DC 10 plus the spell's level check, the cantrip and level 1 reliable cast and the failed-check disintegration are owned by spell-scroll-use.md (FR-008, FR-010, FR-019, FR-021; cases A9 to A12, A16, C1, C2); this spec adds none. | SRD 5.2.1, "Level 13: Use Magic Device" (Scrolls), p. 64 | happy |
| FR-100 | `ThiefReflexes.InitiativeTurns(level, subclass, initiative)` MUST return `[initiative, initiative - 10]` (two Initiative counts in the first round of combat) for a Thief of Rogue level 17 or higher, and `[initiative]` otherwise. It is a pure result handed to the `combat` initiative code; this spec does not track turns. | SRD 5.2.1, "Level 17: Thief's Reflexes", p. 64 | unhappy (boundary) |

Use Magic Device cases (FR-096, FR-113; one test each):

- Charges d6: 6 gives no charges expended (boundary); 5 and 1 give charges expended (boundary).
- Scroll route: Thief 13 gives `AnyScrollRoute(Intelligence, 1, Arcana)` (happy); Thief 12 gives none (boundary); non-Thief 20 gives none (rule interaction).

### 4.10 Cross-cutting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-101 | Every Rogue service and value object MUST be deterministic, side-effect free, and MUST NOT mutate its inputs; equal inputs give equal results. | `doc/rules/architecture.md` §3 | unhappy (immutability) |
| FR-102 | Constructors and lookups MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for an invalid numeric argument: a negative damage or distance, a d20 roll outside 1-20, a d6 roll outside 1-6, a spell level outside 0-9, a negative Speed or die count. The Rogue level 1-20 invariant is enforced by `RogueLevel` (FR-013). | `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-103 | Collections exposed by results (features, effects, reasons, failures) MUST be read-only (accessors return copies). | `doc/rules/architecture.md` §4 | unhappy (immutability) |
| FR-104 | Every public Rogue type a caller must construct or read (section 6) MUST be exported from the `character` package (every exported name unique across the package, decisions FR-017, FR-021); no file in package `character` MUST import another context's internals. Names another class might reuse (Evasion, Uncanny Dodge, Reliable Talent, Elusive) carry the `Rogue` prefix: class-monk.md has its own Evasion (FR-083 to FR-085 there). | `doc/rules/architecture.md` §2, §7 | happy |
| FR-108 | The Rogue's recorded state is one immutable `RogueState` (`RogueSkillChoice`, the `RogueExpertiseWindow`s used, `WeaponMasteryChoice`, the Thieves' Cant `Language`, the `StrokeOfLuckState`), a `FeatureState` with the typed key `rogueStateKey` (name `rogue`, character-aggregate.md FR-093). The subclass is **not** in `RogueState`: it is recorded by `Subclass` with a nil error and passed to the services that need it. `Character` holds `RogueState` in `FeatureStates` (decisions FR-022; character-aggregate.md FR-092, FR-094); the aggregate stores no Rogue-specific field. **The callers sit above the class specs** (decisions FR-016): a host or composition use case calls `RogueChoices.RequiredFor`, then `NewRogueChoices`, and stores the result with `Character.WithFeatureState(rogueStateKey, newState)`; a failure leaves the character unchanged. The same host calls `StrokeOfLuck.Use`, writes the state back and publishes `StrokeOfLuckUsed`; when rests-and-hit-dice.md reports a rest (`RestApplied`) it maps it to a `RestKind`, calls `StrokeOfLuck.AfterRest` and writes the state back. No spec upstream of this one names or imports a Rogue type. | `doc/rules/architecture.md` §3; decisions FR-016, FR-022, FR-023 | happy |
| FR-109 | Every Rogue feature that changes recorded state is a pure function from (`RogueState`, inputs) to (new `RogueState`, events): `StrokeOfLuck.Use` and `afterRest` return the new state with the `StrokeOfLuckUsed` event where due, and the choice validators return the accepted choices or a rejection. Value-only features (Sneak Attack, Uncanny Dodge, Evasion, Reliable Talent, Elusive, Steady Aim, the Thief services) stay pure functions of their inputs. No Rogue feature needs dice or content through a port, so this spec adds no the use-case files use case. | decisions FR-023 | happy |
| FR-110 | Rogue creation choices travel in `CreateCharacter` as a typed `RogueCreationChoices` implementing `ClassCreationChoices` (decisions FR-024): the first-class entry choices of FR-018, FR-020, FR-023, FR-031 and the Weapon Mastery choice. The aggregate adds no Rogue-specific field. `NewRogueCreationChoices` returns `(RogueCreationChoices, error)` (`RogueCreationChoices` with a nil error or an error (problems), the `RogueChoiceProblem` values of FR-022) listing every problem. | decisions FR-024 | happy |
| FR-112 | Feature use is counted by the shared counter `srd5e.character.feature_used` with `srd5e.class` = `rogue` and `srd5e.feature` = the Rogue `FeatureID` of 4.2 (for example `rogue_stroke_of_luck`). This spec defines no signal and emits none: the counter and the shared `.duration` and `.errors` signals are owned by character-aggregate.md and emitted by the use case that calls the Rogue services (decisions FR-023, FR-042, FR-063). | decisions FR-042, FR-063; `doc/rules/observability.md` §3 | unhappy (duplicate) |

### 4.11 Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (FR-017, Hit Point Die d8) |
| FR-002 | Withdrawn: moved to content-classes.md (FR-016, primary ability Dexterity) |
| FR-003 | Withdrawn: moved to content-classes.md (FR-018, saves Dexterity and Intelligence) |
| FR-004 | Withdrawn: moved to content-classes.md (FR-019, FR-020, "choose 4" skill list) |
| FR-005 | Withdrawn: moved to content-classes.md (FR-021, weapon proficiencies) |
| FR-006 | Withdrawn: moved to content-classes.md (FR-022, FR-023, Thieves' Tools, Light armor) |
| FR-007 | Withdrawn: moved to content-classes.md (FR-024, Starting Equipment options A and B) |
| FR-033 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-014, `WeaponMasteryChoice`) |
| FR-034 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-015 to FR-019, failure cases) |
| FR-035 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-024, swap on a Long Rest) |
| FR-044 | Withdrawn: moved to modifiers-and-advantage.md (FR-015, cancelling; Sneak Attack receives the net `RollMode`) |
| FR-091 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-041 to FR-047, an error) |
| FR-097 | Withdrawn: moved to spell-scroll-use.md (FR-019, reliable cast of a cantrip or level 1 spell); replaced by FR-113 |
| FR-098 | Withdrawn: moved to spell-scroll-use.md (FR-008, the DC 10 plus level check) |
| FR-099 | Withdrawn: moved to spell-scroll-use.md (FR-010, FR-021, success and disintegration) |
| FR-105 | Withdrawn: moved to content-classes.md (FR-003, the Rogue entry and its delivery) |

## 5. Model

Value objects and pure functions (stateless domain services) in
`character/` (`doc/rules/architecture.md` §3); no aggregate.
Recorded state is the immutable `RogueState` under `rogueStateKey` (FR-108).

- **Services:** `RogueProgression` (constants: `featuresAt`, `featuresUpTo`,
  `sneakAttackDice`, `savingThrowProficiencies`, `featLevels`, `grantsFeat`,
  `epicBoonLevel`, `weaponMasteryCount`, `subclassLevel`, `subclassID`),
  `RogueChoices` (`requiredFor`, the `New` constructor), `RogueCreationChoices`,
  `RogueSkillChoice`, `RogueStartingEquipmentChoice`, `RogueExpertise`,
  `RogueLanguageChoice`, `SneakAttack`, `CunningStrikeSelection`,
  `RogueGrantedActions`, `SteadyAim`, `RogueUncannyDodge`, `RogueEvasion`,
  `RogueReliableTalent`, `RogueElusive`, `StrokeOfLuck`, `ThiefFeatures`,
  `UseMagicDevice`, `ThiefReflexes`.
- **Values:** `RogueLevel`, `RogueEntry`, `RogueState`, `RogueExpertiseSelection`,
  `RogueExpertiseWindow`, `StrokeOfLuckState`, `StrokeOfLuckTest` (closed:
  `StrokeOfLuckRolledTest`, `StrokeOfLuckUnrolledTest`), `SneakAttackDice`
  (count of d6, `Expression()`), `AllyNearTarget`, `CunningStrikeEffect`,
  `CunningStrikeOutcome`, `CunningStrikeRepeatSaveRule` (4.7), `RogueGrantedAction`,
  `FastHandsOption`, `SecondStoryWork`. Consumed, not defined: `RestKind`,
  `DifficultyClass`, `Roll`, `d20.Outcome`, `AttackRollResult`,
  `RollMode`, `RollCircumstance`, `dice.Expression`.
- **Results** (decisions FR-017: `<Operation>Result`, variants carry the
  operation prefix, problems are enums or closed values):
  `RogueLevelResult` (`RogueLevel` with a nil error, `RogueLevelOutOfRange`);
  `RogueSkillChoiceResult` (`...Accepted`, `...Rejected`; `RogueSkillChoiceProblem`);
  `RogueStartingEquipmentChoiceResult` (`...Chosen`, `...ChoiceMissing`);
  `RogueExpertiseResult` (`...Accepted`, `...Rejected`; `RogueExpertiseProblem`);
  `RogueLanguageChoiceResult` (`...Accepted`, `...Rejected`;
  `RogueLanguageChoiceProblem`); `RogueChoicesResult` and
  `RogueCreationChoicesResult` (`...Accepted`, `...Rejected`; closed
  `RogueChoiceProblem`); `CunningStrikeSelectionResult` (`CunningStrike` with a nil error,
  an error; `CunningStrikeProblem`); `SneakAttackResult`
  (`SneakAttackReason`); `SteadyAimResult` (`SteadyAimReason`);
  `RogueUncannyDodgeResult` (`RogueUncannyDodgeReason`); `StrokeOfLuckResult`
  (`StrokeOfLuckSpent`, `StrokeOfLuckNotAvailable`; `StrokeOfLuckReason`:
  `levelTooLow`, `spent`, `testNotFailed`, `noRoll`); `SecondStoryWorkResult`;
  `AttunementLimitResult`. Consumed, not defined: `AnyScrollRoute`
  (spell-scroll-use.md), `ToolCheckBenefits`.
- **Event:** `StrokeOfLuckUsed`, past-tense, carrying the `CharacterID`,
  implementing `CharacterEvent` (FR-086).
- **Invariants:** Rogue level 1-20 in `RogueLevel` (FR-013); die count
  non-negative; a Cunning Strike selection is valid only through the `New` constructor
  (FR-054, FR-067).
- **Failures:** expected rule outcomes and untrusted input are closed results
  listing problems; programmer errors are `srderr.ErrInvalidInput` (FR-102).

## 6. Interface

- **No use cases and no ports** (FR-109). The host or composition use case
  above the class specs (decisions FR-016) calls these services and stores
  results with `Character.WithFeatureState` (FR-108); `combat`
  (`combat-action-resolution.md`) wires Sneak Attack and Cunning Strike to the
  attack, and `ThiefReflexes.InitiativeTurns` returns a plain list for the
  `combat` initiative code (FR-100). Slippery Mind feeds the `additionalSavingThrows`
  input of derived-stats.md (this spec's FR-080, Q12). All functions are synchronous.
- **Exports:** `character/doc.go` export
  every type in section 5 and every type or enum named in section 4
  (including each closed result, variant and problem enum), plus `rogueStateKey`; the package duplicate-name scan checks the list.
  Every exported member has a godoc comment. `RogueClassFacts` is test-only.
  `WeaponMasteryChoice`, `RestKind`, `Subclass` with a nil error, an error and
  `ChooseSubclass` are exported by their owning specs.
- **Cross-context:** the domain imports only `content` published language
  (ids, `Skill`, `Language`, `CreatureSize`, `ConditionKind`, `DamageType`),
  `d20` / kernel-package types named in section 5, and upstream
  `character` specs (`ProficiencyBonus`, `ClassLevels`, `SkillProficiencies`,
  `Character`, `CharacterEvent`, `FeatureState`, `FeatureStateKey`). Nothing is
  imported from `combat` (decisions FR-013).

## 7. Observability

**Silent by design:** domain code only, with no use case, adapter or port
(`doc/rules/observability.md` §3); it adds nothing to
`doc/observability/signals.md`. Feature use is counted by the shared
`srd5e.character.feature_used` counter and the shared `.duration` and `.errors`
signals owned by character-aggregate.md and emitted by the calling use case
(FR-112; decisions FR-042, FR-063). Failures return as typed results carrying
every problem (FR-022, FR-049, FR-065, FR-067), so the caller can record them;
programmer errors panic only inside `MustX` helpers. Nothing is counted, timed or logged here. A bundled
Rogue entry that fails to decode shows up as `srd5e.content.decode_errors` and
the `srd5e.outcome=failure` series of `srd5e.content.loads`
(content-schema-and-decoder.md); a constants mismatch fails the FR-009 test.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none (shared signals only, FR-112) | | | | |

## 8. Test plan

Tests cite their requirement (`doc/rules/testing.md` §5); table cases are
table-driven, one row per test. Unit tests live in `character/*_test.go` (package `character_test`)
(`rogue_progression_test.go`, `sneak_attack_test.go`,
`cunning_strike_test.go`, `rogue_features_test.go`, `thief_test.go`),
build inputs from a test builder inside a `_test.go` file, and import no ports, adapters,
decoders or bundled content (`testing.md` §3). No doubles or contract suite
apply (no ports). Withdrawn ids (4.11) have no tests here.

| Id | Level | Notes |
|---|---|---|
| FR-008 | unit | No Spellcasting or Pact Magic feature at any of the 20 levels. |
| FR-009 | unit (content-release) | `content/data` tests decode the bundled Rogue entry by the one decode path (decisions FR-031), reads it into the test-only `RogueClassFacts`, asserts equality with the `RogueProgression` constants: one test per injected defect (wrong id, missing or misplaced feature id, wrong die count, missing Thief), one with several defects (all listed in one message), one clean set. Lands with content-classes.md. |
| FR-010 to FR-014 | unit | Rogue Features table cases (20 rows); `RogueLevel` at 0, -1, 1, 20, 21; `featuresUpTo` at levels 1, 3, 8, 9, 20 with and without a subclass. |
| FR-015 to FR-020 | unit | Entry grants (multiclass gets only the four traits and level 1 features); counts 0, 1, 3, 4, 5; repeats; a skill outside the list; equipment option missing; exact problem values. |
| FR-021, FR-022 | unit | Required choices (21 labels); three independent problems all reported; subclass, feat and Weapon Mastery in `requiredFor` but not validated here. |
| FR-023 to FR-030, FR-106 | unit | Expertise window cases and `RogueExpertiseProblem` values; level 5 / 6; Rogue level 6 at total level 9; Rogue level 3 at total level 6. |
| FR-031, FR-032 | unit | Thieves' Cant itself, an id outside the set, a valid second language. |
| FR-036 to FR-043, FR-045 to FR-049 | unit | Sneak Attack cases (inputs carry a net `RollMode`); result exposes die count and size only; `Expression()` gives count d6, modifier 0 (the Critical Hit consumer test is in the `combat` specs). |
| FR-050 to FR-068, FR-107 | unit | Cunning Strike selection and DC cases; exact `CunningStrikeOutcome` values (every field, including `timing`), one test per effect; Withdraw cases. |
| FR-069 to FR-072 | unit | Levels 1, 2, 3; moved and not moved; level 2 and moved lists both reasons in order. |
| FR-073 to FR-079 | unit | Uncanny Dodge, Evasion and Reliable Talent cases; each failing Uncanny Dodge precondition alone and all four together in order. |
| FR-080 | unit | `savingThrowProficiencies` at levels 14 and 15: first class (Dexterity, Intelligence; then plus Wisdom and Charisma); multiclass (empty; then Wisdom and Charisma only). The cross-spec test (multiclass Rogue 15 gains Wisdom and Charisma) lives with derived-stats.md. |
| FR-081 to FR-083 | unit | Elusive cases; `RollMode.Resolve` over the output asserted for the mixed-sources row. |
| FR-084, FR-085 | unit | `featLevels` exactly {4, 8, 10, 12, 16}, `grantsFeat` for levels 1 to 20, `epicBoonLevel` 19, the level 19 required choice. |
| FR-086 to FR-089 | unit | Stroke of Luck cases plus state transitions (use, reuse, Short Rest, Long Rest); the event carries the `characterID` and is a `CharacterEvent`. |
| FR-090, FR-092 to FR-096, FR-100, FR-113 | unit | `subclassLevel` 3, `subclassID` `thief`; `requiredFor` at level 3; Thief features per level; `fastHands` and `secondStoryWork` at levels 2 and 3; `initiativeTurns` (Initiative 15 gives `[15, 5]`; 5 gives `[5, -5]`), levels 16 and 17, non-Thief gives `[initiative]`; `scrollRoute` at Thief 12 and 13 and non-Thief; Use Magic Device cases; `attunementLimit` at levels 12 (`AttunementLimitUnchanged`) and 13 (`AttunementLimitSet(4)`). |
| FR-101 to FR-103 | unit | Equal inputs give equal results; inputs unchanged; `srderr.ErrInvalidInput` per invalid argument; mutating a returned collection leaves the stored value unchanged (accessors return copies). |
| FR-104 | architecture | Package export check, unique-name scan, dependency rules (architecture-guardrails.md); no import from `combat`. |
| FR-108 to FR-111 | unit | `RogueState` stored and read through `FeatureStates` under `rogueStateKey`; `use` and `afterRest` return (new state, events); state unchanged for rejected inputs; `NewRogueCreationChoices` reports all problems at once; `weaponMasteryCount` is 2. |
| FR-112 | doc review | No Rogue-specific metric name and no emission test (no use case); shared signals are tested where character-aggregate.md and the calling use case own them. |

- **Scenario** (`scenarios/`, public API only): a level 5 Rogue attacks
  with a Shortbow at Advantage, gets Sneak Attack, selects Poison with a kit,
  and the DC and remaining dice match; a 3 Fighter / 2 Rogue multiclass gains
  only the FR-016 traits. They use a test-built Rogue entry, not bundled data.
- **Mutation testing:** every changed line in `character/` for
  the Rogue types MUST be mutation tested (`just mutate ./character`, score at least
  80, `doc/rules/testing.md` §8). The boundary cases (levels 4/5, 5/6, 8/9,
  10/11, 13/14, 16/17, 17/18; rolls 9/10; distances 5/10; DC equality) kill
  `<` to `<=` mutants.
- **Telemetry tests:** none. **Not covered:** applying conditions, saves and
  durations (`combat`, d20-tests.md), Hit Point effects, collaborator failure
  (no ports); the only bad-content cases are FR-009 and FR-032.

## 9. Open questions and decisions

Settled: Q12 (Slippery Mind is a class feature, SRD 5.2.1, p. 63, so it adds saves even to a multiclass Rogue; derived-stats.md takes an `additionalSavingThrows` set, FR-080); Q1 to Q4 and Q13 are interpretations recorded in FR-054, FR-107,
FR-059, FR-083 and FR-089; Q5 is decisions FR-009; Q6 is this spec's FR-032; Q7 is decisions
FR-005 and FR-025; Q9 (one narrow service per feature; a generic mechanism only
when a second class needs one) is architecture.md §3 "extension points"; Q8 is the `AllyNearTarget` input (4.6); Q10 and Q14 are decisions
FR-022 to FR-024 and FR-063 (FR-108 to FR-112); Q11 is FR-081.

## 10. Changelog impact

Under `Unreleased` / **Added**: "Rogue class: progression, Sneak Attack,
Cunning Strike, Expertise choice validation, Rogue class features and the
Thief subclass (`character` context). The Rogue content definition is added
by content-classes.md." Not **BREAKING**: all additions, no existing export
changes.
