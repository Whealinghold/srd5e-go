# Character creation

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `ability-scores.md`, `observability-ports.md`, `rule-options.md`, `content-schema-and-decoder.md`, `content-source-port.md`, `content-classes.md`, `content-species-backgrounds-feats.md`, `content-equipment.md`, `derived-stats.md`, `character-aggregate.md`, `feats-behaviour.md`, `level-advancement-and-multiclassing.md`, `higher-level-start-equipment-and-trinkets.md` |

## 1. Purpose and scope

Defines how a character is created, following the five steps of "Create Your Character" (SRD 5.2.1, p. 19): class,
origin (background, species, languages), ability scores, alignment, and the details creation decides (starting
equipment, the optional trinket). Starting above level 1 ("Starting at Higher Levels", p. 24) only composes the pure
domain services of `higher-level-start-equipment-and-trinkets.md`. The `character` context owns here:

- the `CreateCharacter` use case with its command and closed result (single definition, Q-4). It validates every
  choice, looks up content, rolls the dice creation asks for, calls `character.New` (above level 1 also the pure
  domain service `HigherLevelStart`), applies the two Origin feats with the pure domain service `FeatTakingRules`, and
  returns the character, a `CreationRecord` of the choices the aggregate does not store, and the events. **It never
  calls another use case** (`shared-types-and-ownership.md` FR-044);
- ability score generation, the background's increases, the language rules (count, distinctness, d12 mapping; the
  `Language` vocabulary is `content-schema-and-decoder.md` FR-111, Q-1), and the starting equipment grant with the
  order of rolls that compose the higher-level allowance and the Trinkets roll;
- two small use cases that roll for the player: `RollAbilityScores`, `RollLanguages`.

Under `shared-types-and-ownership.md` FR-048 this spec takes over what other specs defer to "a character creation
spec": ability score generation and the consumers of `CharacterRuleOptions.AbilityScoreMethods` (FR-054) and
`StartingLevel` (FR-071, `rule-options.md` FR-029); the origin choices, feats, proficiencies and trinket of a new
character; the generic class skill, tool and equipment-letter rules (FR-015 to FR-020, FR-111) and class-specific
creation choices (FR-021, FR-102, FR-103; Q-3); and the former `CreateCharacter` rows of `character-aggregate.md`
(Q-4). Starting Experience Points and the Starting Equipment at Higher Levels table stay with
`higher-level-start-equipment-and-trinkets.md` (FR-065, FR-081 to FR-089); this spec composes them (FR-072, FR-079 to
FR-086).

**Out of scope:**

- Derived numbers of "Step 5" (pp. 22-23) and the Hit Point maximum rule (`derived-stats.md`,
  `character-aggregate.md`); saving throw, weapon and armor proficiencies come from the initial class and are not
  copied into the record.
- Class features and class-specific level 1 choices (p. 22): the `class-*.md` specs through `ClassCreationChoices`
  (FR-104); spellcasting (pp. 23, 25-26): `spellcasting.md`.
- Gaining a level, Multiclassing (pp. 23-25): `character-aggregate.md`, `level-advancement-and-multiclassing.md`; the
  plan fold above level 1: `higher-level-start-equipment-and-trinkets.md`. A feat's own rules, "Bonus Feats at Level
  20" (p. 24): `feats-behaviour.md`, `rule-options.md`.
- Spending starting coins (p. 20), Armor Class from worn armor, an inventory model (Q-14); Trinket and other content
  text (`content`); persistence and `CharacterID` generation (`character-aggregate.md` D-1).
- Narrative material with no rule (pp. 19-20, Tiers of Play); the domain holds no display strings (architecture §3).
  SRD 5.2.1 has no ideals, bonds, flaws or personality traits for a character, so none are modelled.

## 2. Source

PDF page numbers equal printed page numbers (pages 19 and 83 checked).

- SRD 5.2.1: "Create Your Character" steps and tables, pp. 19-23 (Class Overview, "Ability Scores and Backgrounds", p.
  19; Standard and Rare Languages, p. 20; "Step 3" tables incl. Standard Array by Class, "Step 4: Alignment", p. 21;
  "Unaligned Creatures", p. 22; Level 1 Hit Points by Class, pp. 22-23; Character Advancement table, p. 23); "Starting
  at Higher Levels", p. 24; "Trinkets", pp. 26-27; "Character Backgrounds", p. 83; "Parts of a Species" and "Species
  Descriptions", pp. 83-86; "Origin Feats", p. 87; "Ability Scores", p. 5; "Ability Modifiers", p. 6; "Alignment", p.
  176; Core Traits (Bard, p. 31; Fighter, p. 47; Monk, p. 49).
- Package design: `doc/rules/architecture.md` (§2 to §6), `doc/rules/observability.md` (§2, §3, §5 to §7),
  `doc/rules/testing.md` (§4, §5, §8) and `shared-types-and-ownership.md`.

## 3. Vocabulary

Terms use SRD 5.2.1 wording (`doc/rules/architecture.md` §3); "package term" has no SRD source. Types owned elsewhere
are cited, not defined.

| SRD term | Type | Notes |
|---|---|---|
| Create Your Character | `CreateCharacter`, `CreateCharacterCommand`, closed `CreateCharacterResult` (`CreateCharacterSucceeded`, `CreateCharacterRejected`, `CreateCharacterContentMissing`); `CreationRecord` (package term: choices the aggregate does not store) | p. 19. Single definition (Q-4). |
| Class, Background, Species | `ClassID`, `BackgroundID`, `SpeciesID`; content `ClassDefinition` (`content-classes.md`), `Background`, `Species`; `ClassCreationChoices`, `NoClassCreationChoices` (`character-aggregate.md`, `shared-types-and-ownership.md` FR-024) | pp. 19-20, 83-86. "Species", never "Race". |
| Ability Scores (of a background) | `BackgroundAbilityIncrease` (closed: `PlusTwoPlusOne`, `PlusOneToAll`; package terms; `NewPlusTwoPlusOne`, `ApplyTo(ability.Scores, background abilities)` returning a closed result) | "Increase one by 2 and another one by 1, or increase all three by 1" (pp. 21, 83). |
| Feat (of a background), Origin feat | `FeatID`; `RequiredSpellList` (`ClassID`, `feats-behaviour.md` FR-076) | pp. 83, 87. |
| Skill and Tool Proficiency | `Skill` (`shared-types-and-ownership.md` FR-046); `ToolSelection` (closed: `NoToolSelection`, `ChosenTool(ItemID)`; package term). The background's own proficiency is the content `ToolProficiency` (`content-schema-and-decoder.md` FR-119). | pp. 9, 83. |
| Equipment (Choose A or B) | `EquipmentOptionLetter` (`a`, `b`, `c`; package term), `StartingEquipmentGrant` | pp. 20, 28, 83. Letters `a`, `b`, `c` match the uppercase keys `A`, `B`, `C` of `EquipmentOptions` (`content-species-backgrounds-feats.md` FR-029) and `StartingEquipment` (`content-classes.md` FR-024). |
| Size | `CreatureSize`, `SizeChoice` (`NoSizeChoice`, `ChosenSize`) | "chosen when you select this species" (pp. 84, 86). |
| (species choice, grant) | `SpeciesTraitSelection` (`TraitName` of a content `SpeciesChoice` and the `name` of one option; `content-species-backgrounds-feats.md` FR-016), `SpeciesGrantSelections` (optional `Skill`, optional Origin `FeatID`, `map[String]Ability` from lineage `TraitName` to spellcasting ability; package terms) | Draconic Ancestry, Elven Lineage, Gnomish Lineage, Giant Ancestry, Fiendish Legacy; Skillful, Keen Senses, Versatile (pp. 84-86); FR-106. |
| Languages | `Language`, `Language.IsStandard` (`content-schema-and-decoder.md` FR-111); `Languages`, `StandardLanguageRoll` | p. 20. |
| Generate Your Scores | `AbilityScoreGenerationMethod` (`rule-options.md`), `StandardArray`, `PointCost`, `RolledAbilityScores` with `RolledAbilityScore`, `AbilityScoreAssignment`; `ability.Scores` (`ability-scores.md`) | pp. 21, 6. `NewAbilityScoreAssignment(method, map[ability.Ability]int) (AbilityScoreAssignment, error)` reports every problem found as the `errors.Join` of typed `AbilityScoreAssignmentProblem` values (enum: `AbilityMissing`, `NotStandardArray`, `ScoreOutsideRolledRange`, `ScoreOutsidePointCostRange`, `PointCostExceedsBudget`), each matching `srderr.ErrInvalidInput`; `MustAbilityScoreAssignment` panics with that error (FR-056 to FR-063). |
| Alignment, Unaligned | `Alignment` (`content-schema-and-decoder.md` FR-137) | pp. 21-22. |
| Starting at Higher Levels, Trinkets | `CharacterRuleOptions.StartingLevel`; `StartingLevelStep` (`ClassID`, `HitPointChoice`, optional `SubclassID`, optional `AbilityScoreIncrease`), `HigherLevelStep`, `HigherLevelStart`, `HigherLevelEquipment` with `TotalGold`, `ConsumedFeatGrant`, `TrinketTable`, `Trinket` (`higher-level-start-equipment-and-trinkets.md`); here `HigherLevelEquipmentChoice` (`DeclineHigherLevelEquipment`, `GrantHigherLevelEquipment`), `TrinketChoice` (`NoTrinket`, `RollForTrinket`), `TrinketRoll` (1 to 100); package terms | pp. 24, 26-27. |
| (internal) | `CharacterCreationRules`, pure domain service (see section 5); not exported | FR-004, FR-005, FR-074. |
| Gold Pieces; (a problem in the command) | `Cost`, `Coin` (`content-equipment.md`; no raw integer gold, p. 20); `CreationProblem`, `CreationStep`, `CreationProblemKind` (typed data, no display text) | |

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. "Phase" refers to the flow of FR-003. Class data is read
from `ClassDefinition`, background and species data from `content-species-backgrounds-feats.md`.

### 4.1 The use case and its flow

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `CreateCharacter.Run(ctx, cmd)` MUST return the closed `CreateCharacterResult`: `CreateCharacterSucceeded` (character, `CreationRecord`, events), `CreateCharacterRejected` (problems) or `CreateCharacterContentMissing` (missing content), and a nil `error`. A rule outcome MUST NOT be an `error` or a panic; the `error` is reserved for a port failure or a cancelled `ctx`. | SRD 5.2.1, "Create Your Character", p. 19; architecture §4, §6 | happy |
| FR-002 | On success the character MUST be at the starting level (1 by default, FR-071), built with `character.New` and, above level 1, by delegating the fold to `HigherLevelStart` (FR-074). The events MUST be those of `higher-level-start-equipment-and-trinkets.md` FR-076 (`CharacterCreated` first; then per step `LevelGained`, `SubclassChosen` if any, `AbilityScoreChanged`, `HitPointMaximumChanged`), followed by the events of the feats of FR-105 and FR-108 (`FeatTakingChanged`). Each is a `CharacterEvent` (`shared-types-and-ownership.md` FR-019) defined by its owner; this spec defines none. | SRD 5.2.1, "Create Your Character", p. 19; "Starting at Higher Levels", p. 24 | happy |
| FR-003 | The flow MUST run in a fixed order: phase 1 validates the command with no port call (FR-004); phase 2 looks up all content (FR-006); phase 3 validates the choices that need content; phase 4 rolls the dice (FR-008); phase 5 builds the final character and record with the pure domain services. Phase 3 also builds a throwaway character with no dice (`character.New` at level 1; above level 1 the dry run of `HigherLevelStart` with fixed Hit Point gains) and applies FR-105 then FR-108 to it (FR-108 sees the character after the background feat), so every rejection of those is found before a die is rolled. Phase 5 repeats the calls with the rolled gains, in the same order. A phase that finds problems ends the flow. | n/a (package design; architecture §4, §5 "report every problem") | unhappy (state error) |
| FR-004 | Problems found in phase 1 or phase 3 MUST give `CreateCharacterRejected` carrying every problem found in that phase, and MUST NOT roll any die. A phase 1 rejection MUST NOT call the `content.Source`. | n/a (architecture §5, testing.md §4) | unhappy (invalid input) |
| FR-005 | The problems of one rejection MUST be a slice copied on read, without duplicates, ordered by `CreationStep` in SRD step order (`CharacterClass`, `background`, `species`, `languages`, `ability.Scores`, `alignment`, `StartingLevel`, `StartingEquipment`) and within a step by the order of section 5. The same command always gives the same list. | n/a (determinism, testing.md §4) | unhappy (rule interaction) |
| FR-006 | Phase 2 MUST look up, through the `content.Source`, the class, the background, the species, the background's origin feat, the species-granted Origin feat (FR-108), every tool named by a `ChosenTool` (FR-034, FR-111), every content id exposed by the `ClassCreationChoices` (FR-103), the spells and tools `FeatTakingRules` needs for FR-105 and FR-108 (the entries `feats-behaviour.md` FR-066 looks up; `CreateCharacter` does not call `TakeFeat`) and every class named in the plan (FR-073). When any entry is absent it MUST return `CreateCharacterContentMissing` holding an copied, non-empty list of `MissingContent` ordered by `ContentKind` declaration order then `ContentID` ordinal comparison, without duplicates, and MUST NOT roll any die or run phase 3. | n/a (architecture §4, §6; `shared-types-and-ownership.md` FR-018; `character-aggregate.md` FR-061) | unhappy (missing content) |
| FR-007 | A `*srderr.PortError` returned by the `content.Source` MUST be returned unchanged (wrapped at most once), with no result built. | n/a (architecture §4; observability §7) | unhappy (collaborator failure) |
| FR-008 | Dice MUST be rolled only in phase 4, by `CreateCharacter` through its own `dice.Roller` port, in this order: (1) the allowance die of FR-082 (`1d10`), (2) the trinket die of FR-088 (`1d100`), (3) one Hit Point die per plan step that chose `RollHitDie`, in level order (FR-075). It MUST NOT call `RollHigherLevelStartingMoney`, `RollTrinket`, `StartAtHigherLevel` or `TakeFeat` (`shared-types-and-ownership.md` FR-044). A a dice port failure (`*srderr.PortError`) MUST propagate unchanged, and a roller result for a different expression than requested MUST surface as a dice port failure (`*srderr.PortError`) with reason `InvalidResult` (through `RollChecked`, `dice-and-rolls.md` FR-050, FR-051; that is all `RollChecked` detects, the range of a face is the roller's obligation, FR-047 there). | n/a (package design; `dice-and-rolls.md` FR-050 to FR-052) | unhappy (collaborator failure) |
| FR-009 | A command that asks for no roll (`DeclineHigherLevelEquipment` or a starting level of 1 or 2 to 4, `NoTrinket`, no `RollHitDie` step) MUST NOT call the `dice.Roller` at all. | n/a (testing.md §4 determinism) | unhappy (empty and minimal) |
| FR-010 | The same command, content and dice values MUST give an equal result, identical with the no-op `telemetry.Telemetry` and with a recording one. | n/a (architecture §3, observability §9) | unhappy (determinism) |
| FR-011 | `CreateCharacter` MUST NOT alter its command. Every collection in the command, record, result and events MUST be a copy; a change to a returned slice has no effect. | n/a (architecture §3, §4 handoffs) | unhappy (immutability) |
| FR-012 | `CreateCharacterRejected` and `CreateCharacterContentMissing` MUST carry no `Character` and no events. `CreateCharacterRejected`'s constructor MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) for an empty problem list. | n/a (errors-and-results.md FR-023, FR-024) | unhappy (empty and minimal) |

### 4.2 Step 1: class

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-013 | The class MUST be looked up as `ContentKind.CharacterClass` and its `HitPointDie` handed to `character.New`, so the level 1 Hit Point maximum is the die's maximum face plus the Constitution modifier. | SRD 5.2.1, "Step 1: Choose Class", p. 19; Level 1 Hit Points by Class table, p. 22 | happy |
| FR-014 | With default options a new character MUST be at level 1 with 0 Experience Points and Proficiency Bonus +2. | SRD 5.2.1, "Write Your XP", p. 19; "Note Proficiencies", p. 20 | happy |
| FR-015 | When the class's `SkillChoice` is `SkillChoiceListed` (`content-classes.md` FR-019), the command's class skills MUST be exactly `picks` distinct skills, all from the listed skills, and are granted as proficiencies. | SRD 5.2.1, "Note Proficiencies", p. 20; Fighter, p. 47 | happy |
| FR-016 | A class skills list with fewer or more than `picks` entries (including none) MUST be rejected with `ClassSkillCountWrong`. | SRD 5.2.1, Core Fighter Traits "Skill Proficiencies" ("Choose 2"), p. 47 | unhappy (boundary) |
| FR-017 | A class skill not among the listed skills of a `SkillChoiceListed` MUST be rejected with `ClassSkillNotOffered`, naming the skill. | SRD 5.2.1, Core Fighter Traits, p. 47 | unhappy (invalid input) |
| FR-018 | A skill repeated in the class skills list MUST be rejected with `ClassSkillRepeated`, naming the skill. | n/a (a proficiency is held once) | unhappy (invalid input) |
| FR-019 | When `SkillChoice` is `SkillChoiceAny` (Bard, "Choose any 3 skills"), the class skills MUST be exactly `picks` distinct skills of the 18; FR-016 and FR-018 apply and `ClassSkillNotOffered` cannot occur. | SRD 5.2.1, "Core Bard Traits", p. 31 | happy |
| FR-020 | The class Starting Equipment option MUST be a key of the class's `StartingEquipment` (`A`, `B`, and `C` only where the class has it; `content-classes.md` FR-024); the `EquipmentOptionLetter` `a`, `b`, `c` is matched to these uppercase content keys. An unknown letter MUST be rejected with `ClassEquipmentOptionUnknown`. A valid letter grants exactly the items and gold of that option (FR-070). | SRD 5.2.1, "Choose Starting Equipment", p. 20; Core Fighter Traits ("Choose A, B, or C"), p. 47 | unhappy (invalid input) |
| FR-021 | `ClassCreationChoices` MUST default to `NoClassCreationChoices`, accepted for every class. The choices MUST be stored unchanged in the record and not interpreted here beyond FR-102 and FR-103; the class spec that defines them validates them. The problem `ClassChoicesMismatch` is withdrawn: the interface of `character-aggregate.md` FR-094 carries no class id, so a mismatch cannot be evaluated here. | `shared-types-and-ownership.md` FR-024; `character-aggregate.md` FR-094; SRD 5.2.1, "Record Class Features", p. 22 | unhappy (rule interaction) |

Class skills cases (FR-015 to FR-019): Fighter list per `content-classes.md` table CT1, p. 47; Bard p. 31.

| Label | Class | Chosen skills | Expected | Path |
|---|---|---|---|---|
| two from the list; any three | Fighter (choose 2); Bard (any, choose 3) | Athletics, Perception; Arcana, Deception, Stealth | accepted | happy |
| wrong count | Fighter; Bard | (empty); Athletics; Athletics, Perception, Survival; Arcana, Deception | `ClassSkillCountWrong` | unhappy (empty and minimal, boundary) |
| not on the list | Fighter | Athletics, Arcana | `ClassSkillNotOffered` (Arcana) | unhappy (invalid input) |
| repeated | Fighter | Athletics, Athletics | `ClassSkillRepeated` (Athletics) | unhappy (invalid input) |

### 4.3 Step 2: background

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-022 | The background MUST be looked up as `ContentKind.background` and its id handed to `character.New`. | SRD 5.2.1, "Choose a Background", p. 19 | happy |
| FR-023 | `BackgroundAbilityIncrease` MUST be a closed variant set with exactly two variants: `PlusTwoPlusOne(plusTwo, plusOne)` and `PlusOneToAll`. | SRD 5.2.1, "Parts of a Background: Ability Scores", p. 83; "Adjust Ability Scores", p. 21 | happy |
| FR-024 | `NewPlusTwoPlusOne(plusTwo, plusOne ability.Ability)` MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) when `PlusTwo` and `PlusOne` are the same ability, without panicking; `MustPlusTwoPlusOne` panics with that error. | SRD 5.2.1, "Adjust Ability Scores" ("a different one by 1"), p. 21 | unhappy (invalid input) |
| FR-025 | An increase naming an ability the background does not list MUST be rejected with `BackgroundIncreaseAbilityNotListed`, naming the ability. `PlusOneToAll` is always valid. | SRD 5.2.1, "Adjust Ability Scores" ("Your background lists three abilities"), p. 21; p. 83 | unhappy (invalid input) |
| FR-026 | `PlusTwoPlusOne` MUST add 2 to `PlusTwo` and 1 to `PlusOne` and change no other score. `PlusOneToAll` MUST add 1 to each of the three listed abilities and change no other score. | SRD 5.2.1, "Adjust Ability Scores", p. 21; p. 83 | happy |
| FR-027 | No increase MAY raise a score above 20. `BackgroundAbilityIncrease.ApplyTo` MUST return a failure listing every ability that would exceed 20 and leave the scores unchanged; exactly 20 MUST be accepted. In the use case the problem is `BackgroundIncreaseExceedsMaximum`. This kind is defensive: the highest assignable score is 18 (Random Generation), 18 + 2 = 20, so no command reaches it; it is tested at unit level only (hand-built `ability.Scores`) and excluded from the "every kind is produced by a row" check of FR-094. | SRD 5.2.1, "Adjust Ability Scores" ("None of these increases can raise a score above 20"), p. 21; p. 83 | unhappy (boundary) |
| FR-028 | The increases MUST be applied to the assigned scores before `character.New`, so the Constitution modifier used for the level 1 Hit Point maximum is the increased one. | SRD 5.2.1, "Adjust Ability Scores", p. 21; "Hit Points", p. 22 | unhappy (rule interaction) |
| FR-029 | The background's increases MUST be the only change to ability scores creation makes: neither class, species nor a feat changes a score here. | SRD 5.2.1, "Adjust Ability Scores", p. 21; "Parts of a Species", pp. 83-84 | unhappy (rule interaction) |
| FR-030 | The background's Origin feat MUST be looked up as `ContentKind.feat` and recorded by `FeatID` with, when the background's printed qualifier (`feat.choice`, "Cleric" or "Wizard", `content-species-backgrounds-feats.md` FR-026) names a spell list, the `ClassID` of that list as `RequiredSpellList` (`feats-behaviour.md` FR-076). It is granted by FR-105. The choices the feat requires (Magic Initiate lists and ability, Skilled proficiencies) come in the command's `FeatChoices` and are validated by `FeatTakingRules`, not here. | SRD 5.2.1, "Parts of a Background: Feat", p. 83; "Record Your Feat", p. 20; "Origin Feats", p. 87 | happy |
| FR-031 | The background's two Skill Proficiencies MUST be granted and recorded. | SRD 5.2.1, "Parts of a Background", p. 83; "Note Proficiencies", p. 20 | happy |
| FR-032 | When the background's Tool Proficiency is a specific tool (Calligrapher's Supplies, Thieves' Tools), it MUST be granted and recorded, and a `ChosenTool` MUST be rejected with `BackgroundToolNotAllowed`. | SRD 5.2.1, "Parts of a Background: Tool Proficiency", "Acolyte", "Criminal", p. 83 | unhappy (invalid input) |
| FR-033 | When the Tool Proficiency is a choice from a category (Soldier: "Choose one kind of Gaming Set"), the command MUST carry a `ChosenTool`; `NoToolSelection` MUST be rejected with `BackgroundToolMissing`. | SRD 5.2.1, "Soldier", p. 83 | unhappy (invalid input) |
| FR-034 | A `ChosenTool` MUST be looked up in `ContentKind.equipment` and belong to the category the background names (an Artisan's Tools entry, or a Gaming Set variant). Otherwise `BackgroundToolNotInCategory`. Categories resolve by `ToolGroup` (Artisan's Tools) or `ToolFamily` membership (Gaming Set; Q-18). | SRD 5.2.1, "Parts of a Background: Tool Proficiency", p. 83; "Tools", pp. 93-94 | unhappy (invalid input) |
| FR-035 | The background Starting Equipment option MUST be `A` or `B` (`EquipmentOptionLetter` `a`, `b`, matched to the uppercase content keys). `A` grants the printed items (placeholders resolved by FR-109) and Gold Pieces; `B` grants 50 GP and no items. Any other letter MUST be rejected with `BackgroundEquipmentOptionUnknown`. | SRD 5.2.1, "Parts of a Background: Equipment" ("a package of equipment and 50 GP"), p. 83 | unhappy (invalid input) |
| FR-036 | The recorded skill proficiencies MUST be the union of class skills, background skills and the skill of a species grant (FR-106), held once each. Skills named by more than one source MUST be recorded as `OverlappingSkills` and MUST NOT be a rejection; the SRD gives no replacement rule (Q-6). | SRD 5.2.1, "Note Proficiencies", p. 20 | unhappy (rule interaction) |

Background increase cases (FR-023 to FR-027): Fighter Standard Array scores (Strength 15, Dexterity 14, Constitution
13, Intelligence 8, Wisdom 10, Charisma 12; p. 21); Soldier lists Strength, Dexterity, Constitution (p. 83). Cap rows
use hand-built scores (unit only, FR-027).

| Label | Increase (Soldier) | Starting scores | Expected | Source | Path |
|---|---|---|---|---|---|
| plus two Strength, plus one Constitution; plus two Constitution, plus one Dexterity; plus one to all | `PlusTwoPlusOne(Strength, Constitution)`; `PlusTwoPlusOne(Constitution, Dexterity)`; `PlusOneToAll` | the Fighter row | 17, 14, 14, 8, 10, 12; 15, 15, 15, 8, 10, 12; 16, 15, 14, 8, 10, 12 | p. 21, p. 83 | happy |
| not listed (plus two; plus one) | `PlusTwoPlusOne(Intelligence, Strength)`; `PlusTwoPlusOne(Strength, Charisma)` | the Fighter row | `BackgroundIncreaseAbilityNotListed` (Intelligence; Charisma) | p. 83 | unhappy (invalid input) |
| same ability twice | `NewPlusTwoPlusOne(Strength, Strength)` | any | validation error; `MustPlusTwoPlusOne` panics | p. 21 | unhappy (invalid input) |
| lands on 20 (plus two; plus one) | `PlusTwoPlusOne(Strength, Dexterity)`; `PlusTwoPlusOne(Constitution, Strength)` | Strength 18; Strength 19 | Strength 20 | p. 21 | unhappy (boundary) |
| over 20 (plus two; plus one; plus one to all with Dexterity 20) | `PlusTwoPlusOne(Strength, Dexterity)`; `PlusTwoPlusOne(Dexterity, Strength)`; `PlusOneToAll` | Strength 19; Strength 20; Dexterity 20 | failure (Strength; Strength; Dexterity), scores unchanged | p. 21 | unhappy (boundary) |

Level 1 Hit Point cases with the increase applied (FR-013, FR-028; formula `character-aggregate.md` FR-012; modifiers
p. 6; Sage lists Constitution, Intelligence, Wisdom, p. 83). Source for all rows: p. 21, p. 22.

| Label | Class (die) | Assigned Constitution | Background and increase | Final Constitution | Level 1 Hit Point maximum | Path |
|---|---|---|---|---|---|---|
| plus two, or plus one to all, lifts the modifier | Fighter (d10) | 13 (+1) | Soldier, `PlusTwoPlusOne(Strength, Constitution)`; Soldier, `PlusOneToAll` | 14 (+2) | 12 | unhappy (rule interaction) |
| plus two crosses a band | Barbarian (d12) | 15 (+2) | Soldier, `PlusTwoPlusOne(Constitution, Strength)` | 17 (+3) | 15 | unhappy (rule interaction) |
| increase lifts a negative modifier | Wizard (d6) | 8 (-1) | Sage, `PlusTwoPlusOne(Constitution, Intelligence)` | 10 (+0) | 6 | unhappy (rule interaction) |
| modifier unchanged: elsewhere; plus one | Wizard (d6) | 8 (-1) | Sage, `PlusTwoPlusOne(Intelligence, Wisdom)`; `PlusTwoPlusOne(Intelligence, Constitution)` | 8 (-1); 9 (-1) | 5 | happy, unhappy (boundary) |

Equipment option cases (FR-020, FR-035, FR-109, FR-110): Fighter options per `content-classes.md` table CT3 (A: 4 GP;
B: 11 GP; C: 155 GP, no items); Bard A and B only. The letters `a`, `b`, `c` are matched to the content keys `A`, `B`,
`C`.

| Label | Class option | Background option | Expected | Source | Path |
|---|---|---|---|---|---|
| Fighter A, Soldier A; C, Soldier B; B, Acolyte A | Fighter `a`; `c`; `b` | Soldier `a`; Soldier `b`; Acolyte `a` | Fighter items then Soldier items, 18 GP; no items, 205 GP; Fighter items then Acolyte items, 19 GP | pp. 47, 83 | happy |
| letter the option lacks (class; background) | Bard `c`; Fighter `a` | Soldier `a`; Soldier `c` | `ClassEquipmentOptionUnknown`; `BackgroundEquipmentOptionUnknown` | pp. 20, 83 | unhappy (invalid input, boundary) |
| class letter outside A to C | Fighter `d` | Soldier `a` | not representable as `EquipmentOptionLetter`; the host maps unknown text to a rejection before building the command | p. 47 | unhappy (invalid input) |
| same-as-above: resolved; no tool | Fighter `a` | Soldier `a`, `ChosenTool` Dice; `NoToolSelection` | the grant lists Dice where the printed item is "Gaming Set (same as above)"; `BackgroundToolMissing` and `EquipmentPlaceholderUnresolved` | p. 83 | unhappy (rule interaction, invalid input) |
| Bard instrument left as printed | Bard `a`, three `ChosenTool`s (Lute, Flute, Drum) | Acolyte `a` | "Musical Instrument of your choice" stays as printed; no placeholder problem | p. 31 | unhappy (rule interaction) |
| Bard, no tool chosen | Bard `a`, no `ChosenTool` | Acolyte `a` | `ClassToolCountWrong` only (phase 3) | p. 31 | unhappy (invalid input) |
| Monk instrument: resolved; no tool | Monk `a`, one `ChosenTool` (Lute); none | Acolyte `a` | Monk items list Lute where the printed item is "Artisan's Tools or Musical Instrument chosen for the tool proficiency above"; `ClassToolCountWrong` and `EquipmentPlaceholderUnresolved`, both phase 3, in the order of section 5 | p. 49 | unhappy (rule interaction, invalid input) |

### 4.4 Step 2: species

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-037 | The species MUST be looked up as `ContentKind.species` and its id handed to `character.New`. | SRD 5.2.1, "Choose a Species", p. 20 | happy |
| FR-038 | A species with one size gives a character of that size; `NoSizeChoice` MUST be accepted, and `ChosenSize` of that same size MUST be accepted. The size is recorded. | SRD 5.2.1, "Choose a Species", p. 20; "Size", pp. 83-84 | happy |
| FR-039 | A species with two sizes (Human, Tiefling: Medium or Small) MUST be given a `ChosenSize`; `NoSizeChoice` MUST be rejected with `SizeRequired`; a size the species does not offer with `SizeNotOffered`. | SRD 5.2.1, "Human", "Tiefling", p. 86; "Size", pp. 83-84 | unhappy (invalid input) |
| FR-040 | For each structured species `choices` element (Draconic Ancestry, Elven Lineage, Gnomish Lineage, Giant Ancestry, Fiendish Legacy) the command MUST carry exactly one `SpeciesTraitSelection`. A missing one MUST be rejected with `SpeciesChoiceMissing`, an option not among the options with `SpeciesChoiceOptionUnknown`, a selection for a trait with no choice with `SpeciesChoiceNotOffered`, two selections for one trait with `SpeciesChoiceRepeated`. | SRD 5.2.1, "Special Traits", p. 84; "Dragonborn", p. 84; "Elf", pp. 84-85; "Gnome", "Goliath", p. 85; "Tiefling", p. 86 | unhappy (invalid input) |
| FR-041 | Selections MUST be recorded and their effects MUST NOT be applied: no Breath Weapon damage type, no lineage spell, no Speed change, no resistance. The record holds the choice only. Applying them is the host's step after `CreateCharacter`: it builds `SpeciesTraits` through `SpeciesTraitChoices.TryFromSelections(SpeciesID, selections, SpeciesGrantSelections)` (`species-trait-behaviour.md` FR-007), parallel to `TakeFeat` for Versatile; `CreateCharacter` calls neither (FR-113). | SRD 5.2.1, "Species Descriptions", pp. 84-86 | unhappy (rule interaction) |
| FR-042 | Creature type, Speed and the special traits of the species MUST NOT be copied into the record; they stay readable from the species entry. | SRD 5.2.1, "Parts of a Species", pp. 83-84 | unhappy (rule interaction) |

Species cases (FR-038 to FR-042; sizes and choices per `content-species-backgrounds-feats.md` FR-012, FR-016). Rows
for a species with a grant carry no or valid `SpeciesGrantSelections` (FR-106).

| Label | Species | Size input | Selections | Expected | Source | Path |
|---|---|---|---|---|---|---|
| single size: none chosen; same size | Dwarf (Medium); Gnome (Small) | `NoSizeChoice`; `ChosenSize(Small)` | none; Gnomish Lineage: Forest Gnome | Medium; Small | pp. 84-85 | happy |
| two sizes, either chosen | Human; Tiefling | `ChosenSize(Small)`; `ChosenSize(Medium)` | none; Fiendish Legacy: Infernal | Small; Medium | p. 86 | happy |
| size rejected: other size; none for two sizes; not offered | Dwarf; Human; Tiefling | `ChosenSize(Small)`; `NoSizeChoice`; `ChosenSize(Large)` | none; none; Fiendish Legacy: Infernal | `SizeNotOffered`; `SizeRequired`; `SizeNotOffered` | pp. 84, 86 | unhappy (invalid input) |
| ancestry chosen | Dragonborn | `NoSizeChoice` | Draconic Ancestry: Gold | recorded | p. 84 | happy |
| ancestry missing; not an option | Dragonborn | `NoSizeChoice` | none; Draconic Ancestry: Sonic | `SpeciesChoiceMissing`; `SpeciesChoiceOptionUnknown` | p. 84 | unhappy (invalid input) |
| trait with no choice; same trait twice | Elf | `NoSizeChoice` | Elven Lineage: Drow, Fiendish Legacy: Abyssal; Elven Lineage: Drow, Elven Lineage: Wood Elf | `SpeciesChoiceNotOffered` (Fiendish Legacy); `SpeciesChoiceRepeated` | pp. 84-86 | unhappy (invalid input) |
| species with no structured choice | Orc | `NoSizeChoice` | none | accepted | p. 86 | unhappy (empty and minimal) |
| record shape | Dragonborn; Elf | `NoSizeChoice` | Draconic Ancestry: Gold; Elven Lineage: High Elf | the record holds the selection only: no damage type, Breath Weapon, resistance, lineage spell, Speed change, Darkvision or trait text; the Elf's Speed is still read from the species entry (FR-041, FR-042) | pp. 84-85 | unhappy (rule interaction) |

### 4.5 Step 2: languages

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-043 | A new character MUST know Common and two more languages, three in all, recorded as `Languages` with Common always present. | SRD 5.2.1, "Choose Languages", p. 20 | happy |
| FR-044 | The `Language` vocabulary and the Standard and Rare membership (10 Standard including Common, 9 Rare, Primordial one entry, 19 in all, no overlap) is static SRD data owned by `content-schema-and-decoder.md` FR-111 (`shared-types-and-ownership.md` FR-046). This spec defines no table, list or rarity type: every rule below reads `Language` and `IsStandard` from `content`. | SRD 5.2.1, "Standard Languages", "Rare Languages", Primordial note, p. 20 | happy |
| FR-045 | `StandardLanguageRoll.LanguageFor(roll)` MUST map a d12 roll as in the case table below (Common is not reachable) and MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) for a roll below 1 or above 12. | SRD 5.2.1, Standard Languages table (1d12), p. 20 | unhappy (boundary) |
| FR-046 | The command MUST carry exactly two chosen languages; any other count MUST be rejected with `LanguageCountWrong`. | SRD 5.2.1, "Choose Languages", p. 20 | unhappy (boundary) |
| FR-047 | A chosen language MUST have `Language.IsStandard` true. A Rare language, which only a feature can grant, MUST be rejected with `LanguageNotStandard`. | SRD 5.2.1, "Choose Languages", p. 20 | unhappy (invalid input) |
| FR-048 | The two chosen languages MUST differ from each other and from Common; a repeat MUST be rejected with `LanguageRepeated` (a character knows at least three languages). | SRD 5.2.1, "Choose Languages", p. 20 | unhappy (invalid input) |
| FR-049 | `Languages` MUST be an immutable value object that always contains Common, holds each language once and exposes its languages as a copy. `Languages.WithAdded(language)` (the route for a language a class or feature grants) MUST return a new value including the language (a Rare language is allowed) and an equal value when it is already known. | SRD 5.2.1, "Choose Languages" ("Your class and other features might also give you languages"), p. 20 | unhappy (rule interaction) |
| FR-050 | `RollLanguages` MUST roll two separate `1d12` requests through the `dice.Roller`, in order, map each with FR-045 and, when the languages differ, return `RollLanguagesSucceeded` with the two languages and two rolls. | SRD 5.2.1, "Choose Languages", Standard Languages table, p. 20 | happy |
| FR-051 | When the two rolled languages are the same (rolls 3 and 4 are both Dwarvish), `RollLanguages` MUST return `RollLanguagesDuplicate` carrying the two rolls and the language, and MUST NOT roll again. The host rolls again. | SRD 5.2.1, "Choose Languages", p. 20 (no duplicate rule; Q-8) | unhappy (rule interaction) |
| FR-052 | `RollLanguages` MUST NOT call the `content.Source`, and the languages it returns MUST be accepted as the command's chosen languages (FR-046 to FR-048) without change. | n/a (package design) | happy |
| FR-053 | The content `Language` vocabulary (all 19 values, `content-schema-and-decoder.md` FR-111) MUST be the one source of the permitted language set: a feature granting a language from "the language tables" (Rogue, Ranger) takes all `Language` values. Choosing a known language again through a feature is not decided here (Q-1). | SRD 5.2.1, "Choose Languages", p. 20 | happy |

Standard Languages d12 mapping (FR-045), one test per roll, p. 20: 1 Common Sign Language; 2 Draconic; 3, 4 Dwarvish;
5, 6 Elvish; 7 Giant; 8 Gnomish; 9 Goblin; 10, 11 Halfling; 12 Orc; 0 and 13 validation error.

Language choice and roll cases (FR-046 to FR-051). Source p. 20.

| Label | Input | Expected | Path |
|---|---|---|---|
| two standard languages | Dwarvish, Elvish | `Languages` of Common, Dwarvish, Elvish | happy |
| wrong count: none, one, three | (empty); Dwarvish; Dwarvish, Elvish, Giant | `LanguageCountWrong` | unhappy (empty and minimal, boundary) |
| not Standard: a Rare language; Thieves' Cant | Abyssal, Elvish; Thieves' Cant, Elvish | `LanguageNotStandard` (Abyssal; Thieves' Cant) | unhappy (invalid input) |
| repeated: Common again; same twice | Common, Elvish; Orc, Orc | `LanguageRepeated` | unhappy (invalid input) |
| rolls 3, 5; rolls 1, 12 | two `1d12` rolls | `RollLanguagesSucceeded`: Dwarvish, Elvish; Common Sign Language, Orc | happy (boundary) |
| duplicate: rolls 3, 4; 10, 11 | two `1d12` rolls | `RollLanguagesDuplicate` (Dwarvish; Halfling) | unhappy (rule interaction, boundary) |
| `WithAdded`: a Rare language; a known one | `Languages` (Common, Dwarvish, Elvish) plus Thieves' Cant; plus Dwarvish | four languages; equal to the original | unhappy (rule interaction, boundary) |

### 4.6 Step 3: ability scores

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-054 | The command MUST carry the `CharacterRuleOptions` in force (default `standard`, permitting all three methods). An assignment whose method is not in `AbilityScoreMethods` MUST be rejected with `AbilityScoreMethodNotPermitted`. This is the requirement that reads `AbilityScoreMethods` (`rule-options.md` FR-029). | SRD 5.2.1, "Generate Your Scores", p. 21; `character-aggregate.md` FR-104, FR-105 | unhappy (invalid input) |
| FR-055 | `StandardArray()` MUST return exactly 15, 14, 13, 12, 10 and 8, as a fresh slice (no mutable package-level array). | SRD 5.2.1, "Standard Array", p. 21 | happy |
| FR-056 | A Standard Array assignment MUST give each of the six abilities one of the six scores, each used once. A set of six scores that is not a permutation of `StandardArray()` MUST be rejected with `NotStandardArray`, including a set with the same total (72). | SRD 5.2.1, "Standard Array", "Assign Ability Scores", p. 21 | unhappy (invalid input) |
| FR-057 | `RollAbilityScores` MUST issue exactly six `4d6` requests through the `dice.Roller`, in order, and record for each the four dice and the total of the highest three (`SumOfHighest(3)`). | SRD 5.2.1, "Random Generation", p. 21 | happy |
| FR-058 | Every total from FR-057 MUST be from 3 to 18; the six totals MUST be returned in roll order, with their dice, in a slice copied on read. The use case does not assign them. | SRD 5.2.1, "Random Generation", "Assign Ability Scores", p. 21 | unhappy (boundary) |
| FR-059 | A Random Generation assignment MUST give each ability a score from 3 to 18; otherwise `ScoreOutsideRolledRange`. The package cannot check the numbers were the ones rolled (Q-9). | SRD 5.2.1, "Random Generation", p. 21 | unhappy (boundary) |
| FR-060 | `PointCost.of(score)` MUST return 0, 1, 2, 3, 4, 5, 7 and 9 for the scores 8 to 15 and MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) below 8 or above 15. `PointCost.budget` MUST be 27. | SRD 5.2.1, Ability Score Point Costs table, p. 21 | happy |
| FR-061 | A Point Cost assignment MUST give each ability a score from 8 to 15, else `ScoreOutsidePointCostRange`; and the total cost MUST be at most 27, else `PointCostExceedsBudget`. The total is checked only when every score is in range. A total below 27 MUST be accepted (Q-10). | SRD 5.2.1, "Point Cost" ("You have 27 points to spend"), p. 21 | unhappy (boundary) |
| FR-062 | `NewAbilityScoreAssignment(method, scores)` MUST take the host's raw integers by ability, report `AbilityMissing` for each ability without an entry, and report an integer that is not an `ability.Score` (below 1 or above 30) as the problem of its method: `NotStandardArray` (`StandardArray`), `ScoreOutsideRolledRange` (`RandomGeneration`), `ScoreOutsidePointCostRange` (`PointCost`). It MUST list every problem found. | SRD 5.2.1, "Assign Ability Scores", p. 21; "Ability Scores", p. 5 | unhappy (invalid input) |
| FR-063 | `NewAbilityScoreAssignment` MUST NOT panic for any map of integers (including nil and extreme values); `MustAbilityScoreAssignment` MUST panic if and only if it returns an error for the same input, and the panic value is that error. | n/a (errors-and-results.md FR-025, FR-026) | unhappy (invalid input) |
| FR-064 | The scores handed to `character.New` MUST be the assigned scores after the background increase (FR-026); every score of the created character MUST be at most 20. | SRD 5.2.1, "Ability Scores", p. 5; "Adjust Ability Scores", p. 21 | unhappy (boundary) |
| FR-065 | The record MUST keep the method, the assigned scores before the increase and the increase, so a consumer can explain the final scores. | n/a (architecture §6) | happy |
| FR-066 | The Standard Array MUST be a valid Point Cost assignment as well (15, 14, 13, 12, 10, 8 cost 9, 7, 5, 4, 2, 0 = 27); a consistency check of FR-055 and FR-060, no new rule. | SRD 5.2.1, "Standard Array", "Point Cost", p. 21 | unhappy (rule interaction) |

Standard Array cases (FR-055, FR-056, FR-066), ordered Strength, Dexterity, Constitution, Intelligence, Wisdom,
Charisma; accepted rows are the Standard Array by Class rows. Source p. 21.

| Label | Method | Scores | Expected | Path |
|---|---|---|---|---|
| the Fighter, Wizard and Barbarian rows | standardArray | 15, 14, 13, 8, 10, 12; 8, 12, 13, 15, 14, 10; 15, 13, 14, 10, 12, 8 | accepted | happy |
| not a permutation: a score twice; same total 72; off by one; all equal | standardArray | 15, 15, 13, 12, 10, 8; 14, 14, 13, 12, 10, 9; 15, 14, 13, 12, 10, 7; 10 six times | `NotStandardArray` | unhappy (invalid input, boundary) |
| missing ability | standardArray | five scores | `AbilityMissing` | unhappy (empty and minimal) |
| Standard Array as Point Cost | pointCost | 15, 14, 13, 12, 10, 8 | accepted, cost 27 | unhappy (boundary) |

Random Generation cases (FR-057 to FR-059): dice in roll order, total is the highest three (`dice-and-rolls.md`
FR-041). Source p. 21.

| Label | Four dice or assignment | Total or expected | Path |
|---|---|---|---|
| ordinary; order does not matter | 6, 5, 4, 1; 1, 4, 6, 5 | 15; 15 | happy |
| all ones; all sixes | 1, 1, 1, 1; 6, 6, 6, 6 | 3; 18 | unhappy (boundary) |
| tie at the cut; lowest die dropped | 3, 3, 3, 2; 2, 6, 6, 6 | 9; 18 | unhappy (boundary) |
| assignment of 3 or 18; of 2 or 19 | assignment score 3; 18; 2; 19 | accepted; accepted; `ScoreOutsideRolledRange` (2, 19) | unhappy (boundary) |

Point Cost cases (FR-060, FR-061); costs 8 to 15: 0, 1, 2, 3, 4, 5, 7, 9 (`PointCost.of(14)` is 7). Source p. 21.

| Label | Scores | Total cost | Expected | Path |
|---|---|---|---|---|
| accepted: all eights; three fifteens; balanced | 8, 8, 8, 8, 8, 8; 15, 15, 15, 8, 8, 8; 14, 14, 14, 10, 10, 10 | 0; 27; 27 | accepted | happy (boundary) |
| over budget | 15, 15, 15, 8, 8, 9; 13, 13, 13, 13, 13, 13; 15, 15, 15, 15, 15, 15 | 28; 30; 54 | `PointCostExceedsBudget` | unhappy (boundary) |
| outside the range | 7, 8, 8, 8, 8, 8; 16, 8, 8, 8, 8, 8 | n/a | `ScoreOutsidePointCostRange` | unhappy (boundary) |
| both problems | 16, 15, 15, 15, 8, 8 | n/a | `ScoreOutsidePointCostRange` only (cost not computed for an out-of-range set) | unhappy (rule interaction) |

Raw integer cases (FR-062): Strength gets an integer outside 1 to 30 (not an `ability.Score`); the other five scores
are valid. Source p. 21.

| Label | Method | Raw integer | Expected | Path |
|---|---|---|---|---|
| 0 or 31 | standardArray; randomGeneration; pointCost | 0; 31 | `NotStandardArray`; `ScoreOutsideRolledRange`; `ScoreOutsidePointCostRange` | unhappy (boundary) |
| 30 is an `ability.Score` | randomGeneration | 30 | `ScoreOutsideRolledRange` (an `ability.Score`, out of the method range; p. 5) | unhappy (boundary) |

### 4.7 Step 4: alignment

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-067 | The command's `Alignment` MUST be accepted and recorded when it is one of the nine: Lawful Good, Neutral Good, Chaotic Good, Lawful Neutral, Neutral, Chaotic Neutral, Lawful Evil, Neutral Evil, Chaotic Evil. | SRD 5.2.1, "The Nine Alignments", pp. 21-22; Rules Glossary, "Alignment", p. 176 | happy |
| FR-068 | `Alignment.unaligned` MUST be rejected with `AlignmentNotChoosable`: the sidebar describes creatures lacking rational thought, and the step offers only the nine. | SRD 5.2.1, "Step 4: Alignment", p. 21; "Unaligned Creatures", p. 22 | unhappy (invalid input) |
| FR-069 | An Evil alignment MUST be accepted without a rejection, a warning or a different result: the SRD only says the game assumes player characters are not evil and to check with the GM. | SRD 5.2.1, "Step 4: Alignment", p. 21 | unhappy (rule interaction) |

Alignment cases (FR-067 to FR-069), one test per alignment named. Source pp. 21-22.

| Label | Alignment | Expected | Path |
|---|---|---|---|
| the six non-Evil | lawfulGood, neutralGood, chaoticGood, lawfulNeutral, neutral, chaoticNeutral | accepted | happy |
| the three Evil | lawfulEvil, neutralEvil, chaoticEvil | accepted, no warning | unhappy (rule interaction) |
| Unaligned | unaligned | `AlignmentNotChoosable` | unhappy (invalid input) |

### 4.8 Step 5: starting equipment, starting level and the higher-level start

Starting above level 1 is owned by `higher-level-start-equipment-and-trinkets.md` (`HigherLevelStart`: FR-065 to
FR-080; `HigherLevelEquipment`: FR-081 to FR-084; `TrinketTable`: FR-090 to FR-092). The rows below state only how
`CreateCharacter` composes those pure services (inputs, order of rolls, one content-missing result, one rejection
list); it never calls that spec's use cases (`shared-types-and-ownership.md` FR-044), so the dice are rolled here
(FR-008).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-070 | `StartingEquipmentGrant` MUST hold the items of the chosen class option, then the items of the chosen background option, in that order, as `{name, quantity}` as the content stores them (after the placeholder resolution of FR-109 and FR-110), plus gold as a `Cost` in Gold Pieces (`content-equipment.md`): class option gold plus background option gold plus the allowance total of FR-082 when one applies. | SRD 5.2.1, "Choose Starting Equipment", p. 20 | happy |
| FR-071 | The starting level MUST be `CharacterRuleOptions.StartingLevel` (default 1). This is the requirement that reads `StartingLevel` (`rule-options.md` FR-029; `character-aggregate.md` FR-106). | SRD 5.2.1, "Starting at Higher Levels", p. 24; "Write Your Level", p. 19 | happy |
| FR-072 | Starting Experience Points MUST be those of `higher-level-start-equipment-and-trinkets.md` FR-065 (the minimum for the starting level). `CreateCharacter` MUST NOT compute them and the command has no Experience Points field. At level 1 the character has 0 (FR-014). | SRD 5.2.1, "Creating Your Character", p. 24; Character Advancement table, p. 23 | happy |
| FR-073 | The command MUST carry the higher-level plan as an ordered list of `StartingLevelStep` (a domain value defined in `higher-level-start-equipment-and-trinkets.md` section 3: a `ClassID`, a `HitPointChoice` (`RollHitDie` or `TakeFixedValue`, `character-aggregate.md`), an optional `SubclassID` and an optional `AbilityScoreIncrease`, so the phase 1 length check needs no application type); default empty. A list whose length is not the starting level minus 1 MUST be rejected in phase 1 with `LevelAdvancementCountWrong` carrying expected and actual number (the rule of `higher-level-start-equipment-and-trinkets.md` FR-066, whose `HigherLevelStartPlanLengthMismatch` this check makes unreachable), and no die is rolled. | SRD 5.2.1, "Starting at Higher Levels", p. 24; `higher-level-start-equipment-and-trinkets.md` FR-066 | unhappy (boundary) |
| FR-074 | For a starting level above 1 the character MUST be built by the pure domain service `HigherLevelStart` (`higher-level-start-equipment-and-trinkets.md` FR-065 to FR-076), not its use case `StartAtHigherLevel`; `CreateCharacter` MUST NOT fold `LevelUp` itself. The pure mapping in `CharacterCreationRules` (the conversion of `higher-level-start-equipment-and-trinkets.md` FR-119, which adds no other rule) turns each `StartingLevelStep` and the looked-up `ClassDefinition` of its `ClassID` into one `HigherLevelStep`, whose `AdvancementClass` is built from the `ClassDefinition` (id, `HitPointDie`, ability requirement; `character-aggregate.md` FR-018) and whose `HitPointGain` is `FixedHitPointGain` of the die's fixed value (`TakeFixedValue`, and every step of the dry run) or `RolledHitPointGain(roll, die)` (`RollHitDie`, FR-075); subclass and `AbilityScoreIncrease` pass through. The `HigherLevelStartRejection` variants `HigherLevelStartLevelUpRejected` carrying `LevelUpPrerequisitesNotMet`, `HigherLevelStartSubclassRejected`, `HigherLevelStartIncreaseRejected`, `HigherLevelStartIncreaseNotGranted` and `HigherLevelStartSubclassRequired` MUST become the problem `LevelAdvancementRejected` carrying that rejection. A `HigherLevelStartLevelUpRejected` carrying `LevelUpInsufficientExperience` or `LevelUpMaximumLevel` is governed by FR-076. The rejection MUST be found in phase 3 by a dry run with fixed gains (no rejection depends on the gain), so no die is rolled for a command that will be rejected. | SRD 5.2.1, "Gaining a Level", p. 23; "Multiclassing", "Prerequisites", p. 24; "Create Your Character", p. 19 | unhappy (state error) |
| FR-075 | A step with `RollHitDie` MUST be rolled by `CreateCharacter` through the `dice.Roller` (via `dice.Roll`), one die of the step class's Hit Point Die in step order (same rule as `higher-level-start-equipment-and-trinkets.md` FR-074), and handed to `HigherLevelStart` as `RolledHitPointGain`; `TakeFixedValue` steps do not roll. | SRD 5.2.1, "Gaining a Level" step 2, p. 23 | happy |
| FR-076 | `HigherLevelStartCreationInvalid`, a `HigherLevelStartLevelUpRejected` carrying `LevelUpInsufficientExperience` or `LevelUpMaximumLevel`, and a length mismatch are unreachable here (scores at most 20 by FR-064; Experience Points the minimum for the level, `higher-level-start-equipment-and-trinkets.md` FR-070; length checked by FR-073). If one is returned `CreateCharacter` MUST panic with an error value (an unreachable branch, never a result and never a returned `error`); this takes precedence over FR-074's mapping. The branches have no injection seam (`HigherLevelStart` is a pure domain service) and are listed as uncovered in section 8. | SRD 5.2.1, Character Advancement table, p. 23; `higher-level-start-equipment-and-trinkets.md` FR-070, FR-078 | unhappy (state error) |
| FR-077 | A start at level 3 or above whose plan leaves a class without its subclass MUST be rejected (`LevelAdvancementRejected` carrying `HigherLevelStartSubclassRequired`, `higher-level-start-equipment-and-trinkets.md` FR-073); `CreateCharacter` makes no subclass choice itself (the plan's steps carry it, `higher-level-start-equipment-and-trinkets.md` FR-071). Feat choices and class feature choices other than those of FR-102 and FR-105 MUST NOT be made by `CreateCharacter`. | SRD 5.2.1, "Write Your Level", p. 19; "Starting at Higher Levels", p. 24 | unhappy (rule interaction) |
| FR-078 | The Hit Point maximum, Hit Dice and class levels of a created character MUST equal what `HigherLevelStart` gives (the level 1 rule applies to the first class only; `higher-level-start-equipment-and-trinkets.md` FR-068, table SH). | SRD 5.2.1, "Hit Points", p. 22; "Gaining a Level", p. 23 | unhappy (rule interaction) |
| FR-079 | The command MUST carry a `HigherLevelEquipmentChoice`, closed: `DeclineHigherLevelEquipment` (default) and `GrantHigherLevelEquipment` (distinct from the level-advancement spec's allowance variant `NoHigherLevelEquipment`). It is a command input, not a `CharacterRuleOptions` field, because the SRD makes it a GM decision and `rule-options.md` lists it out of scope (Q-16). | SRD 5.2.1, "Starting Equipment" (under "Starting at Higher Levels"), p. 24 | happy |
| FR-080 | With `DeclineHigherLevelEquipment` the grant MUST be exactly the class and background options at every starting level, no allowance recorded and no die rolled for it. | SRD 5.2.1, "Starting Equipment", p. 24 | happy |
| FR-081 | With `GrantHigherLevelEquipment` the allowance MUST be `HigherLevelEquipment.ForStartingLevel(startingLevel)` (`higher-level-start-equipment-and-trinkets.md` FR-081 to FR-084): `NoHigherLevelEquipment` at level 1, otherwise a `HigherLevelEquipmentAllowance`. The record holds that value unchanged (magic item counts included). | SRD 5.2.1, Starting Equipment at Higher Levels table, p. 24 | happy |
| FR-082 | With `GrantHigherLevelEquipment` and an allowance whose `RolledGold` is non-empty (levels 5 to 20), `CreateCharacter` MUST roll the one `RolledGold` expression (`1d10`) through the `dice.Roller` (via `dice.Roll`) as the first die of FR-008, then obtain the money by `HigherLevelEquipmentAllowance.TotalGold(roll: roll)` (`higher-level-start-equipment-and-trinkets.md` FR-116, FR-120), added to the grant's gold. It does no arithmetic and no roll-range validation of its own: `TotalGold` returns a validation error for a roll outside 1 to 10 (`shared-types-and-ownership.md` FR-048). Levels 2 to 4 have no `RolledGold`, roll nothing and call `TotalGold()` without a roll. The record keeps the allowance, the roll and the total. Tier and boundary cases are that spec's (Q-23). | SRD 5.2.1, Starting Equipment at Higher Levels table ("plus 1d10 x 25 GP"), p. 24 | unhappy (boundary) |
| FR-083 | A starting level of 1 with `GrantHigherLevelEquipment` MUST give no allowance and roll nothing (the table has no row for level 1). | SRD 5.2.1, Starting Equipment at Higher Levels table (rows start at level 2), p. 24 | unhappy (empty and minimal) |
| FR-084 | The magic items of an allowance are chosen by the GM and MUST NOT be chosen, rolled or looked up by `CreateCharacter`; only the counts by rarity are recorded (FR-081). | SRD 5.2.1, Starting Equipment at Higher Levels table ("Magic Items"), p. 24 | unhappy (rule interaction) |
| FR-086 | Coins from the grant MUST stay coins: Gold Pieces are one `Cost` and nothing is purchased, so no item is added for them. | SRD 5.2.1, "Choose Starting Equipment", p. 20 | unhappy (rule interaction) |
| FR-109 | An item whose printed name is "(same as above)" in the background's option (Soldier: "Gaming Set (same as above)") MUST be resolved to the `ChosenTool` of the background's Tool Proficiency (FR-033): the grant holds that tool's name. When such an item exists and no `ChosenTool` exists, `EquipmentPlaceholderUnresolved` MUST be reported. | SRD 5.2.1, "Soldier" Equipment, p. 83 | unhappy (rule interaction) |
| FR-110 | Only a class option item the SRD ties to an earlier choice MUST be resolved: the Monk's option A item "Artisan's Tools or Musical Instrument chosen for the tool proficiency above" (p. 49) MUST become the one `ChosenTool` of the class tool choice (FR-111, `picks` 1); with no `ChosenTool`, `EquipmentPlaceholderUnresolved` MUST be reported under step `CharacterClass`. The Bard's "Musical Instrument of your choice" (p. 31) is not linked by the SRD to the three instrument proficiencies, so it MUST stay as printed, is flagged in the summary (Q-21) and raises no problem. Other "of your choice" items are not modelled (Q-3). | SRD 5.2.1, "Core Monk Traits", Starting Equipment, p. 49; "Core Bard Traits", p. 31 | unhappy (rule interaction) |
| FR-111 | A class `ToolProficiencies` element of kind `choice` (`content-classes.md` FR-023; Bard "Choose 3 Musical Instruments", p. 31) MUST be answered by exactly `picks` distinct `ChosenTool`s, each a tool of one of the element's categories (looked up in `ContentKind.equipment`). A wrong count is `ClassToolCountWrong`, a tool outside the categories `ClassToolNotInCategory`; a `specific` element needs no selection and is granted. | SRD 5.2.1, "Core Bard Traits", p. 31; "Core Monk Traits", p. 49 | unhappy (invalid input) |
| FR-112 | Proficiencies and choices a class adds when taken as a new class in a plan step (p. 25 "Proficiencies", `ClassDefinition.MulticlassGrants`, e.g. Bard) are NOT chosen by `CreateCharacter`; they belong to `level-advancement-and-multiclassing.md` and `derived-stats.md`. A multiclass start records only the first class's choices (Q-20). | SRD 5.2.1, "Multiclassing", "Proficiencies", p. 25 | unhappy (rule interaction) |

Composition cases (FR-072 to FR-083); the owner's tables (SL, SH, SE, SM) give the numbers, only composition is
asserted. A Fighter (d10, Constitution modifier +2).

| Label | Starting level | Plan | Equipment choice | Dice rolled, in order | Expected | Source | Path |
|---|---|---|---|---|---|---|---|
| default level 1 | 1 | none | decline | none | 0 Experience Points, Hit Point maximum 12, no allowance | p. 19, p. 22 | happy |
| level 3, fixed steps, subclass in step 2 | 3 | Fighter fixed, Fighter fixed with a subclass | decline | none | 900 Experience Points; Hit Point maximum 28; events `CharacterCreated`, two `LevelGained`, `SubclassChosen` | pp. 19, 23 | happy |
| rejected plan: subclass missing; prerequisite short | 3; 2 | Fighter fixed, Fighter fixed; Cleric fixed (Wisdom 12) | decline | none | `LevelAdvancementRejected` (`HigherLevelStartSubclassRequired`; `HigherLevelStartLevelUpRejected`) | pp. 19, 24 | unhappy (state error) |
| plan length wrong: too short; too long at level 1 | 3; 1 | one step | decline | none | `LevelAdvancementCountWrong` (expected 2, actual 1; expected 0, actual 1) | p. 24 | unhappy (boundary) |
| level 5, everything rolled | 5 | four steps: two `RollHitDie`, two `TakeFixedValue`, subclass in the step that reaches level 3 | grant, trinket roll | allowance `1d10`, trinket `1d100`, then two Hit Point dice | allowance total through `TotalGold` (SM), trinket roll recorded, gold added to the grant | pp. 24, 26 | happy |
| grant at level 2; at level 1 | 2; 1 | one step; none | grant | none | allowance of 0 GP and 1 Common magic item, grant gold unchanged; no allowance | p. 24 | unhappy (boundary, empty and minimal) |
| roller returns another expression | 2 | `RollHitDie`; `dicetest.CallbackRoller` returns a result for a different expression (count, die or modifier; one case each) | decline | one `1d10` | `a port failure (`*srderr.PortError` for the dice port)(invalidResult)` | p. 23 | unhappy (collaborator failure) |

### 4.9 Trinkets

The table is `higher-level-start-equipment-and-trinkets.md` FR-090 to FR-092; `CreateCharacter` rolls the die itself
(not `RollTrinket`) and keeps the number.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-087 | The command MUST carry a `TrinketChoice`, closed: `NoTrinket` (default) and `RollForTrinket` (no fields). With `NoTrinket` no die is rolled and the record has no trinket. | SRD 5.2.1, "Trinkets", p. 26 | happy |
| FR-088 | With `RollForTrinket` the use case MUST roll one `1d100` through the `dice.Roller` (via `dice.Roll`; the 1 to 100 range of the face is the roller's obligation and a `DieRoll` cannot hold another face, so no range check here) and record a `TrinketRoll` holding the number from 1 to 100. | SRD 5.2.1, "Trinkets", Trinkets table, pp. 26-27 | happy |
| FR-089 | A roll of 100 MUST be recorded as 100 (the last row is printed "00", `higher-level-start-equipment-and-trinkets.md` FR-091). | SRD 5.2.1, Trinkets table, p. 27 | unhappy (boundary) |
| FR-090 | A character MUST have at most one trinket from creation, by the shape of `TrinketChoice`. | SRD 5.2.1, "Trinkets" ("you can roll once"), p. 26 | unhappy (rule interaction) |
| FR-091 | The record MUST hold only the `TrinketRoll`; the `Trinket` text is `TrinketTable.EntryFor(roll)` and MUST NOT be copied into the record. | SRD 5.2.1, "Trinkets", p. 26; `higher-level-start-equipment-and-trinkets.md` FR-090 to FR-092 | unhappy (rule interaction) |

### 4.10 The record, the events and the shape of the result

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-092 | `CreationRecord` MUST hold: the ability score method, the assigned scores before the increase and the `BackgroundAbilityIncrease`; the skill proficiencies and `OverlappingSkills`; the tool proficiency and the class `ChosenTool`s; the Origin feat (`FeatID`, `RequiredSpellList`); the `CreatureSize`; the `SpeciesTraitSelection`s and `SpeciesGrantSelections`; the `Languages`; the `Alignment`; the `ClassCreationChoices`; the `StartingEquipmentGrant` with the `HigherLevelEquipment` allowance; the `TrinketRoll` when there is one; the `CharacterFeats` of FR-105 and FR-108; the `ConsumedFeatGrant`s of FR-115; and the starting level. It MUST hold nothing `Character` stores. | SRD 5.2.1, "Create Your Character", p. 19 | happy |
| FR-093 | `CreationRecord`, `StartingEquipmentGrant`, `SpeciesGrantSelections`, `TrinketRoll`, `Languages`, `AbilityScoreAssignment`, `BackgroundAbilityIncrease`, `RolledAbilityScores` and `CreationProblem` MUST be immutable and compare by value. | n/a (architecture §3) | happy |
| FR-094 | `CreationProblem` MUST carry typed data only (step, kind, and the offending ability, skill, language, letter, option or step number where there is one) and no display text. `CreationProblemKind` MUST be exactly the kinds of section 5 (`BackgroundIncreaseExceedsMaximum` is defensive and unreachable, FR-027). | n/a (architecture §6; errors-and-results.md FR-020) | unhappy (invalid input) |
| FR-095 | The events of a successful creation MUST be those of FR-002, each carrying the `CharacterID`. Creation MUST NOT emit an event for the record: it is not aggregate state (Q-2). | n/a (architecture §3; `character-aggregate.md` FR-015, FR-085) | happy |
| FR-096 | `CreateCharacter` MUST give a character equal to the one `character.New` (and, above level 1, `HigherLevelStart`) builds from the same arguments; the aggregate's rules (scores at most 20, Hit Point maximum at least 1, legal Hit Point Die) are not repeated here, and a violation reaching the aggregate is its validation error (unreachable from a validated command, so FR-076's panic applies). | n/a (architecture §3) | unhappy (rule interaction) |

### 4.11 telemetry.Telemetry behaviour

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-097 | Each use case call MUST emit its span, its `operations` increment and its `duration` record once, with the defined constants and attribute keys, and MUST NOT log. | n/a (observability §5, §6, §7) | happy |
| FR-098 | A `CreateCharacterContentMissing` result MUST be recorded exactly once: `srd5e.outcome` `failure`, `srd5e.character.result` `content_missing`, span error status, one increment of `srd5e.character.operations`, no `error.type` (a returned failure carries none), no log. A returned `*srderr.PortError` (from the content source or the dice roller) MUST be recorded exactly once with span error status, one counter increment, no `srd5e.character.result`, no log and `error.type` equal to `srderr.ErrorType(err)` (`content_error`, `dice_roller_error`). A returned validation error or a cancelled `ctx` MUST NOT be counted as a port failure and MUST carry no `error.type`. | `errors-and-results.md` FR-034, FR-043; `character-aggregate.md` FR-071, FR-088; observability §7 | unhappy (telemetry on failure) |
| FR-099 | A `CreateCharacterRejected` and a `RollLanguagesDuplicate` result MUST be recorded as normal rule outcomes: `srd5e.outcome` `failure`, `srd5e.character.result` `rejected` (create) or `duplicate` (roll languages), span status not error, no `WARNING` or `SEVERE` log, no `error.type`. A rejection MUST carry `srd5e.character.rejection` with the snake_case name of the first problem's kind. A success MUST be `srd5e.outcome` `success` with result `created` (create) or `rolled` (`RollAbilityScores`, `RollLanguagesSucceeded`). | `character-aggregate.md` FR-072; observability §6, §7; `errors-and-results.md` FR-043 | unhappy (telemetry on failure) |
| FR-100 | `RollAbilityScores` MUST record each of its six totals in the histogram `srd5e.character.ability_score_roll`, so a consumer can see a skewed roller. | n/a (observability §5) | happy |
| FR-101 | The `CreateCharacter` span and counter MUST carry `srd5e.character.ability_score_method` (`standard_array`, `random_generation`, `point_cost`) on every outcome of `create` (the command always carries an assignment; rejection, content-missing and returned port failure included). It is absent from `roll_ability_scores` and `roll_languages`. | n/a (observability §5; low cardinality) | happy |

### 4.12 Composed class, feat and species grants

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-102 | `CreateCharacter` MUST call the seeding method of the command's `ClassCreationChoices` (`character-aggregate.md` FR-094) with an empty `FeatureStates` and pass the result to `character.New` (above level 1, to the level 1 character `HigherLevelStart` extends). `NoClassCreationChoices` returns the states unchanged. The choices MUST be stored unchanged in the record (FR-021). | `shared-types-and-ownership.md` FR-022, FR-024; `character-aggregate.md` FR-094, FR-096 | happy |
| FR-103 | In phase 2 `CreateCharacter` MUST look up every content id the command's `ClassCreationChoices` exposes (the method returning the `content.Key`s the choices name, a copy, empty for `NoClassCreationChoices`; `character-aggregate.md` FR-094) and report each absent one in `CreateCharacterContentMissing` beside the others (FR-006). Which choices types return a non-empty list is theirs. `MonkCreationChoices` returns an empty list (`class-monk.md` FR-143): its tool and equipment choices are the plain command fields of FR-111 and FR-110 (Q-26). | `character-aggregate.md` FR-094; `shared-types-and-ownership.md` FR-017, FR-018 | unhappy (missing content) |
| FR-104 | The Core Class Traits (Hit Point Die, saving throws, weapon, armor and tool training, skill choice) are `ClassDefinition` data and MUST NOT be copied into the record; `derived-stats.md` and `equipment-rules.md` read them from the class id. Level 1 features are listed by `ClassDefinition.progression` and `features`; the per-character state a feature needs at level 1 (Fighter Second Wind uses, Paladin Lay On Hands pool) is whatever the class's `ClassCreationChoices` seed into the `FeatureStates` (FR-102). `CreateCharacter` grants no other trait and stores no feature. A class spec wanting level 1 state MUST seed it through its choices, not a use case of its own. The level 1 trait rows of the Fighter, Ranger, Warlock and Paladin specs were withdrawn to this requirement. | SRD 5.2.1, "As a Level 1 Character" (Fighter, p. 47); "Record Class Features", p. 22 | happy |
| FR-105 | The background's Origin feat MUST be applied to the created character with the pure domain service `FeatTakingRules.Take` of `feats-behaviour.md` (the rules of `TakeFeat`, which is not called), with a `FeatGrant` of source `BackgroundFeat` and scope `SpecifiedFeat(featID, requiredSpellList)` built from the background (FR-030) and the command's `FeatChoices` for it, after the character is built and before the Versatile feat of FR-108. The result carries the `CharacterFeats`, `ProficiencyGrants` and `FeatGrantedSpells` of the `FeatTakingChanged` it returns. A `FeatTakingRejected` MUST become the problem `BackgroundFeatRejected` carrying its `FeatRejection`, found in phase 3 on the throwaway character of FR-003 (no die rolled) and applied again in phase 5 to the final character. | SRD 5.2.1, "Parts of a Background: Feat", p. 83; "Record Your Feat", p. 20; `feats-behaviour.md` Q17 | unhappy (rule interaction) |
| FR-106 | A species that asks the player to choose when selected MUST be answered by the command's `SpeciesGrantSelections`: a skill (Human Skillful: any; Elf Keen Senses: Insight, Perception or Survival), an Origin feat (Human Versatile), and for each of Elven Lineage, Gnomish Lineage and Fiendish Legacy the spellcasting ability (Intelligence, Wisdom or Charisma). No species data says which species offers which grant (`content-species-backgrounds-feats.md` Q17 declines per-entry data), so `CreateCharacter` validates only what the command carries and records it; the package neither requires nor forbids a selection (Q-7). A feat that is not an Origin feat (`FeatCategory.origin`, `content-schema-and-decoder.md` FR-117, read from the entry of FR-006) MUST be rejected with `SpeciesFeatNotOrigin`; an ability other than Intelligence, Wisdom or Charisma with `SpeciesAbilityNotOffered`; an ability entry keyed by a `TraitName` that is not a `choices` trait of the chosen species with `SpeciesGrantNotOffered`. The skill is any `Skill`; the Keen Senses restriction is left to the host (Q-7). The host passes the same `SpeciesGrantSelections` to `TryFromSelections`, which ignores its Versatile feat (taken through `TakeFeat`). | SRD 5.2.1, "Human", p. 86; "Elf", pp. 84-85; "Gnome", p. 85; "Tiefling", p. 86 | unhappy (invalid input) |
| FR-107 | The skill of a species grant MUST be granted as a proficiency, recorded in the skill union and counted in `OverlappingSkills` (FR-036). The spellcasting ability MUST be recorded and not applied (FR-041). | SRD 5.2.1, "Note Proficiencies", p. 20; "Species Descriptions", pp. 84-86 | unhappy (rule interaction) |
| FR-108 | An Origin feat chosen for Versatile MUST be looked up as `ContentKind.feat` and applied with `FeatTakingRules.Take`, a `FeatGrant` of source `SpeciesTraitFeat` and scope `FeatFromCategory(Origin)`, after the background feat of FR-105, with the command's `FeatChoices` for it. A `FeatTakingRejected` is the problem `SpeciesFeatRejected` carrying its `FeatRejection`. It is found in phase 3 by applying FR-105 then this requirement to the throwaway character (so it sees the character after the background feat, e.g. a repeated non-repeatable Origin feat) and applied again in the same order in phase 5. | SRD 5.2.1, "Human" (Versatile), p. 86; "Origin Feats", p. 87; `feats-behaviour.md` Q17 | unhappy (rule interaction) |
| FR-113 | `CreateCharacter` MUST NOT call any other use case. It composes only pure domain services and values published in package `character` and its upstreams (`character.New`, `HigherLevelStart`, `HigherLevelEquipment`, `FeatTakingRules`, `TrinketTable`, the `ClassCreationChoices` method) and supplies content and dice through its own consumer-side `ContentSource` and `Roller` interfaces (satisfied by `content.Source` and `dice.Roller`). An architecture check (section 8, FR-097 of `architecture-guardrails.md`) fails when the use case file mentions another use case (`StartAtHigherLevel`, `RollHigherLevelStartingMoney`, `RollTrinket`, `TakeFeat`). | `shared-types-and-ownership.md` FR-044; `doc/rules/architecture.md` §4 | unhappy (rule interaction) |
| FR-114 | `CreateCharacter` MUST record only its own signals (section 7): no `srd5e.character.take_feat`, `srd5e.character.advancement.*` or other use case span or metric; attribute values come from its own result. | `shared-types-and-ownership.md` FR-044; `doc/rules/observability.md` §5 to §7 | unhappy (telemetry on failure) |
| FR-115 | `CreationRecord` MUST hold, in step order, the `ConsumedFeatGrant`s that `HigherLevelStart` reports for the plan's applied increases (none at level 1 or for a plan with no increase). `CreateCharacter` emits no `FeatTaken` for them and changes no `CharacterFeats` because of them; the host keeps them beside the character. | `higher-level-start-equipment-and-trinkets.md` FR-118; package design | happy |

### Withdrawn ids

| Id | Requirement |
|---|---|
| FR-085 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-081) |

## 5. Model

No aggregate of its own (`Character` is `character-aggregate.md`'s). This spec adds the stateless pure domain service
`CharacterCreationRules` (phase 1 and 3 validation FR-004, problem ordering FR-005, the `StartingLevelStep` to
`HigherLevelStep` mapping FR-074 with no money arithmetic FR-082, assembly of character and record from looked-up and
rolled values) and the value objects, enums and closed types of section 3 (immutable, closed where a choice is closed;
`StartingLevelStep` is a domain value, so phase 1 can check plan length; no language list lives in `character`).
Domain code is synchronous, silent and does no I/O. `CreationStep` values, in SRD step order (FR-005):
`CharacterClass`, `background`, `species`, `languages`, `ability.Scores`, `alignment`, `StartingLevel`,
`StartingEquipment`. `CreationProblemKind` values per step, in the order FR-005 uses within a step (telemetry value is
the snake_case name):

| Step | Kinds |
|---|---|
| `CharacterClass` | `ClassSkillCountWrong` (FR-016, FR-019), `ClassSkillNotOffered` (FR-017), `ClassSkillRepeated` (FR-018), `ClassToolCountWrong`, `ClassToolNotInCategory` (FR-111), `ClassEquipmentOptionUnknown` (FR-020), `EquipmentPlaceholderUnresolved` (FR-110, the Monk item). `ClassChoicesMismatch` is withdrawn (FR-021). |
| `background` | `BackgroundIncreaseAbilityNotListed` (FR-025), `BackgroundIncreaseExceedsMaximum` (FR-027), `BackgroundToolNotAllowed` (FR-032), `BackgroundToolMissing` (FR-033), `BackgroundToolNotInCategory` (FR-034), `BackgroundEquipmentOptionUnknown` (FR-035), `BackgroundFeatRejected` (FR-105), `EquipmentPlaceholderUnresolved` (FR-109, the background item) |
| `species` | `SizeRequired`, `SizeNotOffered` (FR-039), `SpeciesChoiceMissing`, `SpeciesChoiceOptionUnknown`, `SpeciesChoiceNotOffered`, `SpeciesChoiceRepeated` (FR-040), `SpeciesGrantNotOffered`, `SpeciesFeatNotOrigin`, `SpeciesAbilityNotOffered` (FR-106), `SpeciesFeatRejected` (FR-108). `SpeciesGrantMissing` and `SpeciesSkillNotOffered` do not exist (Q-7). |
| `languages` | `LanguageCountWrong` (FR-046), `LanguageNotStandard` (FR-047), `LanguageRepeated` (FR-048) |
| `ability.Scores` | `AbilityScoreMethodNotPermitted` (FR-054) |
| `alignment` | `AlignmentNotChoosable` (FR-068) |
| `StartingLevel` | `LevelAdvancementCountWrong` (FR-073), `LevelAdvancementRejected` (FR-074, FR-077) |
| `StartingEquipment` | none: every equipment problem is reported under the step that owns the option |

Phase 1 problems (no content needed): `AbilityScoreMethodNotPermitted`, `LanguageCountWrong`, `LanguageNotStandard`,
`LanguageRepeated`, `AlignmentNotChoosable`, `LevelAdvancementCountWrong`, `ClassSkillRepeated`,
`SpeciesAbilityNotOffered`. All others need a content entry and are phase 3 (`ClassSkillCountWrong` reads `picks` from
class data; the tool kinds and `EquipmentPlaceholderUnresolved` read the class's `ToolProficiencies`,
`content-classes.md` FR-023; `SpeciesFeatNotOrigin` reads the feat entry; `LevelAdvancementRejected` comes from the
dry run of FR-074; the two feat rejections from `FeatTakingRules`).

**Invariants and failures.** `AbilityScoreAssignment`, `PlusTwoPlusOne`, `Languages`, `StandardLanguageRoll` and
`TrinketRoll` enforce their ranges with validation errors from their `NewX` constructors (and panics only in `MustX`);
`CreateCharacterRejected` holds at least one problem (FR-012). Rule outcomes are results (architecture §6); port
failures are `*srderr.PortError`s returned from `Run` (FR-007, FR-008); the domain returns none, and a branch FR-076 calls
unreachable is a panic with an error value. Other contexts are referenced by typed id only.

## 6. Interface

All use cases are types with a single `Run(ctx context.Context, cmd <UseCase>Command)` method and an immutable command; `ctx` is first because they emit spans and `CreateCharacter` reads the content source (go.md §4). `Run` returns the `<UseCase>Result` and an `error` that is non-nil only for a port failure (a `*srderr.PortError`) or a cancelled `ctx`. They take `telemetry.Telemetry` through a `WithTelemetry` option (zero-value `telemetry.Noop` default).

| Use case | Command | Result (closed) | Calls |
|---|---|---|---|
| `CreateCharacter` | `CreateCharacterCommand`: `CharacterID`, `ClassID`, `BackgroundID`, `SpeciesID`, `AbilityScoreAssignment`, `BackgroundAbilityIncrease`, class skills (`[]Skill`), class `ChosenTool`s, class and background `EquipmentOptionLetter`, `ToolSelection`, `SizeChoice`, `SpeciesTraitSelection`s, `SpeciesGrantSelections`, `FeatChoices` for the background feat and the Versatile feat, chosen `Language`s (a list, so a repeat can be reported), `Alignment`, `ClassCreationChoices` (default `NoClassCreationChoices`), `TrinketChoice` (default `NoTrinket`), `CharacterRuleOptions` (default `standard`), the plan as `StartingLevelStep`s (default empty), `HigherLevelEquipmentChoice` (default `DeclineHigherLevelEquipment`) | `CreateCharacterResult`: `CreateCharacterSucceeded`, `CreateCharacterRejected`, `CreateCharacterContentMissing` | `ctx` first; reads the content source and the dice roller |
| `RollAbilityScores` | `RollAbilityScoresCommand` (no fields; uniform call shape) | `RollAbilityScoresResult`: `RollAbilityScoresSucceeded` (`RolledAbilityScores`) | `ctx` first (span); dice port only |
| `RollLanguages` | `RollLanguagesCommand` (no fields) | `RollLanguagesResult`: `RollLanguagesSucceeded`, `RollLanguagesDuplicate` | `ctx` first (span); dice port only |

Ports: `content.Source` (takes `ctx`; expected absence is a not-found outcome, becoming `CreateCharacterContentMissing`;
infrastructure failure is a returned `*srderr.PortError`, FR-007); `dice.Roller` (synchronous, no `ctx`, through `dice.Roll`, in the order
of FR-008, FR-057, FR-050; its failure is returned as a `*srderr.PortError`); `telemetry.Telemetry`. `RollAbilityScores` and `RollLanguages`
are separate use cases because the host needs the rolled values before it can build the command; both are stateless.

**Exports.** Package `character` exports the three use cases with their commands
and results, every value object, enum and closed type of sections 3 and 5 defined here (with variants and problem
types; not `CharacterCreationRules`, which stays internal), and the set `rule-options.md` section 6 lists for the
`character` package (`CharacterRuleOptions`, `AbilityScoreGenerationMethod`, `BonusFeatsAtLevel20` and variants).
`BonusFeatsAtLevel20` has no consumer FR yet, an accepted exception to `rule-options.md` FR-029 (Q-17). Types owned
elsewhere are not re-exported; no exported name is generic (architecture §4). The package doc comment lists
each command, result and the inputs of `CreateCharacterCommand`.

## 7. Observability

Domain code is **silent by design**: it returns problems and values that carry the what and why. The use cases share
`srd5e.character.operations`, `srd5e.character.duration` and the child span `srd5e.character.load_content` with the
aggregate's use cases, owned by `character-aggregate.md` section 7 (`shared-types-and-ownership.md` FR-063 pattern).
`srd5e.outcome` stays `success` or `failure` (`shared-types-and-ownership.md` FR-041; `errors-and-results.md` FR-043):
`created` and `rolled` are `success`; `rejected`, `content_missing`, `duplicate` are `failure`. `error.type` appears
only when a returned `srderr.PortError` left the use case (FR-098).

| Question (`doc/rules/observability.md` §2) | Answer |
|---|---|
| 1. What can go wrong? | A choice that breaks a rule (typed problems); content missing; the content port or dice roller throwing or returning a result for a different expression; a rolled language duplicate; invalid arguments (bugs). |
| 2. How would an operator know? | `srd5e.outcome` `failure` with result `rejected` and `srd5e.character.rejection`; `content_missing` with span error status; a returned port failure as span error and `error.type`. All distinguishable from `created`. |
| 3. What is needed to diagnose it? | Operation, outcome, result, ability score method, first problem kind or error type. The full problem list, choices and rolls are in the result and record, not in attributes. |
| 4. What is counted or timed? | Calls and outcomes per operation, duration per operation (`character-aggregate.md` FR-103), time in lookups, the distribution of rolled ability score totals. |
| 5. Is existing instrumentation still correct? | Yes; shared signals keep their definitions and gain additive values. `CreateCharacter` records only these signals (FR-114). |
| 6. What should a consumer monitor? | Any `error.type` increment; the `rejected` share of `create` by rejection kind; the mean of `ability_score_roll` (about 12.2 for a fair roller); p99 of `srd5e.character.duration` for operation `create` (outcome is on the counter and span, not the histogram; `character-aggregate.md` FR-103). |
| 7. Is anything noisy? | No log at all (observability §7). The histogram adds six values per `RollAbilityScores` call. |

Signals (they feed `doc/observability/signals.md`); the first two extend shared signals and add no new name:

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.operations` (extended) | counter | operation adds `roll_ability_scores`, `roll_languages`; `srd5e.character.result` adds `rejected`, `rolled`, `duplicate` (`character-aggregate.md` FR-088); `srd5e.character.ability_score_method` on `create`; `srd5e.character.rejection` on rejections | Calls by operation and outcome | Share of `rejected`; any `error.type` |
| `srd5e.character.duration` (shared) | histogram, s | `srd5e.character.operation` values only, no new attribute | Time per operation | p99 for `create` |
| `srd5e.character.create` | span | `srd5e.outcome`, `srd5e.character.result` (`created`, `rejected`, `content_missing`), `srd5e.character.ability_score_method` (`standard_array`, `random_generation`, `point_cost`; every `create` outcome), `srd5e.character.rejection` (rejections only) | Path of one creation including lookups and rolls | Error status above 0 for sustained periods; share of `rejected` |
| `srd5e.character.roll_ability_scores` | span | `srd5e.outcome` (`success`), `srd5e.character.result` (`rolled`) | Path of one ability score roll | Latency regression only |
| `srd5e.character.roll_languages` | span | `srd5e.outcome`, `srd5e.character.result` (`rolled`, `duplicate`) | Path of one language roll | Share of `duplicate` (expected 1 in 8) |
| `srd5e.character.ability_score_roll` | histogram, `{score}` | none | Distribution of the six totals (3 to 18) | Mean far from about 12.2, or totals outside 3 to 18 (broken roller) |

Rejection attribute values are the snake_case names of the `CreationProblemKind` values in section 5 (for example
`class_skill_count_wrong`, `size_required`).

## 8. Test plan

Doubles from the `*test` packages (`shared-types-and-ownership.md` FR-038; `_test.go` files only): `dicetest.FixedRoller` (including values that run
out; an out-of-range value is a contract violation, `dice-and-rolls.md`
FR-057), `dicetest.CallbackRoller (returning an error)`, a `dicetest.CallbackRoller` returning a result for a different expression, a `contenttest.FakeSource` (class,
background, species, feat and tool entries; one with entries missing; one that returns a `*srderr.PortError`) and the
`telemetrytest` recorder. No new port, so no contract suite. Domain unit tests use no doubles; tables are
table-driven.

| Ids | Level | Notes |
|---|---|---|
| FR-001, FR-002, FR-095, FR-096 | use case | One success per shape: level 1; level 3 with two fixed steps and a subclass in step 2. Events in the order of `higher-level-start-equipment-and-trinkets.md` FR-076; character equals the aggregate's own result. |
| FR-003 to FR-012 | use case, unit | Source that records requests: a phase 1 rejection makes no lookup and no roll; problem ordering across two steps and determinism; missing content gives an ordered, deduplicated key list and no roll; a source `*srderr.PortError` is returned (`errors.Is` reaches the cause); roll order allowance, trinket, Hit Point dice, no roll requested means no roller call, roller failure, different-expression result (`srderr.ErrContractViolation`); equal results with no-op and recording `telemetrytest` bundle; command unchanged; a change to a returned slice has no effect on every collection; a rejection never carries an empty problem list. |
| FR-013, FR-014, FR-022, FR-037 | use case | Level 1 Hit Point maximum from the class die; 0 Experience Points and +2; background and species looked up and ids reach `character.New`. |
| FR-015 to FR-019 | unit | Table "Class skills cases" with a list-kind and an any-kind class. |
| FR-020, FR-035, FR-070, FR-086, FR-109 to FR-112 | unit, use case | Table "Equipment option cases": letters the class has and lacks (`a`, `b`, `c` against keys `A`, `B`, `C`), gold totals, item order, placeholder rows, Bard with three `ChosenTool`s, wrong count and tool outside category as phase 3 problems, a multiclass plan step adds no tool or skill choice. |
| FR-021, FR-102, FR-103 | unit, use case | `NoClassCreationChoices` for every class, choices stored unchanged (none for the withdrawn `ClassChoicesMismatch`); a double seeds a `FeatureStates` entry reaching the character (also at level 3); a double exposing two content keys, one absent, gives `CreateCharacterContentMissing` beside the others. |
| FR-023 to FR-029 | unit, use case | Table "Background increase cases"; `MustPlusTwoPlusOne` agrees with the constructor; only the three listed abilities change. Cap rows and `BackgroundIncreaseExceedsMaximum` use hand-built scores (FR-027). Table "Level 1 Hit Point cases" runs through `CreateCharacter` with the real aggregate (FR-028 ordering). |
| FR-030 to FR-034, FR-036 | unit, use case | Feat looked up as `ContentKind.feat`, recorded with and without a qualifier (Acolyte "Cleric", Criminal none); skills granted; specific tool granted and `ChosenTool` rejected; choice tool required, looked up, outside category rejected; skill union with and without an overlap. |
| FR-038 to FR-042 | unit | Table "Species cases", including the record-shape row. |
| FR-043, FR-045 to FR-053 | unit, use case | `Languages` holds Common, no repeats, `WithAdded`; the d12 mapping line and table "Language choice and roll cases"; fixed roller: two requests, no content call, no further roll after a duplicate; the content `Language` values are the permitted set. |
| FR-044 | data (in `content`) | `content-schema-and-decoder.md` FR-111 (Q-1); no test here. |
| FR-054 to FR-066 | unit, use case | A method outside the permitted set rejected for each of the 7 subsets; tables "Standard Array cases", "Random Generation cases", "Point Cost cases", "Raw integer cases"; six `4d6` requests in order, fixed roller extremes 1 and 6; several problems at once; `MustAbilityScoreAssignment` agrees with the constructor for generated maps (fuzz test `FuzzNewAbilityScoreAssignment` never panics); final scores at most 20; record keeps method, scores and increase. |
| FR-067 to FR-069 | unit | Table "Alignment cases". |
| FR-071 to FR-084 | unit, use case | Table "Composition cases"; `StartingLevel` read for 1, 2 and 20; Experience Points equal table SL; `DeclineHigherLevelEquipment` at levels 1, 5 and 20 gives no allowance and no roll; grant rows add `TotalGold` money to the grant gold, no roll for levels 1 to 4. The `StartingLevelStep` mapping is unit-tested in `CharacterCreationRules`. Hit Point numbers are asserted through one row (28), otherwise in the owner's table SH; tier and boundary cases are the owner's tests (`higher-level-start-equipment-and-trinkets.md` FR-081 to FR-089, FR-116). |
| FR-087 to FR-091 | use case | `NoTrinket` (no roll); `RollForTrinket` with 1, 50 and 100; the record holds the `TrinketRoll` only. |
| FR-092 to FR-094, FR-115 | unit | Record fields; value equality and copied collections; the problem kind set equals section 5 (one test per kind some row produces through `CreateCharacter`; `BackgroundIncreaseExceedsMaximum` excluded, FR-027). |
| FR-097 to FR-101 | use case | Recording `telemetry.Telemetry`: span names, attributes, counter, histograms, no log; `CreateCharacterContentMissing` (no `error.type`), `content_error` and `dice_roller_error` once each, no port-failure count for a validation error; rejection strings for every `CreationProblemKind`; six histogram records per `RollAbilityScores`; method attribute for each method. |
| FR-104 to FR-108 | use case, unit | Record has no field for saving throws, training or features, and a seeded state is on the character; Soldier (Savage Attacker) and Acolyte (Magic Initiate, Cleric) applied with `FeatTakingRules`, Versatile second; a rejected feat gives `BackgroundFeatRejected` or `SpeciesFeatRejected` and no roll; species grant selections absent (accepted), valid ones recorded, `SpeciesFeatNotOrigin`, an ability outside Intelligence, Wisdom and Charisma, `SpeciesGrantNotOffered`; the skill joins the union and `OverlappingSkills` (Q-7). |
| FR-113, FR-114 | architecture, use case | Import scan of the use case file; a recording `telemetrytest` bundle shows only the `srd5e.character.create` signals and its `load_content` child, none of `take_feat` or `advancement`, for a level 3 start with both feats. |

- **Scenario** (public API only): roll six ability scores with a fixed roller, assign them as a Fighter, create a
  Fighter with the Soldier background and a Human (chosen size, Skillful and Versatile selections), rolling two
  languages and a trinket; check Hit Point maximum, Proficiency Bonus, record, feats and events; the same character at
  level 3 (Hit Point maximum 28, XP 900); a class id the source lacks gives `CreateCharacterContentMissing`. Cites
  FR-002, FR-013, FR-028, FR-050, FR-057, FR-072, FR-074, FR-088, FR-105.
- **Architecture:** FR-113 import check; exported names defined once; signal constants in
  `doc/observability/signals.md`; the domain imports only `content`'s published language. **Mutation testing:**
  `gomu` on `./character` at score 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (`doc/rules/testing.md` §8; never as a bare host process), emphasis on the
  boundaries of FR-016, FR-027 (20, 21), FR-059 (3, 18, 2, 19), FR-060 and FR-061 (27, 28, 8, 15), FR-073, FR-111 and
  the d12 mapping of FR-045.
- **Not covered at its own level:** FR-044 (in `content`); FR-085 is withdrawn. **Uncovered error branches (testing.md
  §7):** the three unreachable panicking branches of FR-076. No command reaches them (FR-064, minimum Experience Points, phase
  1 length check) and `HigherLevelStart` has no injection seam, so a test would have to break the aggregate. Each is a
  one-line `panic(err)` of an error value in `CharacterCreationRules`.

## 9. Open questions and decisions

All questions are settled; none is escalated.

Settled (not relitigated): Q-1 `Language` belongs to `content-schema-and-decoder.md` FR-111. Q-4 this spec owns
`CreateCharacter`, the aggregate keeps `character.New`. Q-5 events are `CharacterEvent`
(`shared-types-and-ownership.md` FR-019); this spec adds none. Q-6 an overlapping skill is held once and reported
(FR-036). Q-7 no species data says who offers which grant, so FR-106 validates only what the command carries. Q-8 a
rolled duplicate language is returned and the host rolls again (FR-051). Q-9 any six numbers from 3 to 18 are accepted
(FR-059). Q-10 a Point Cost total of at most 27 is accepted (FR-061). Q-12
`higher-level-start-equipment-and-trinkets.md` owns `Trinket` and `TrinketTable`; this spec records the `TrinketRoll`
only (FR-091). Q-15 the background entries on p. 83 are authoritative (FR-025); the p. 19-20 table is advice, not
modelled. Q-16 the higher-level equipment table is a GM decision, a command input (FR-079). Q-19 no signal for an Evil
alignment (FR-069). Q-20 any class in the plan may advance at creation (FR-074); partial proficiencies of a new class
are not captured (FR-112). Q-22 resolution compares tool names (Q-21). Q-23 the allowance money is
`higher-level-start-equipment-and-trinkets.md` FR-116. Q-25 the plan step is its `StartingLevelStep` (FR-073). Q-27
the size exception is closed. Q-24 `FeatSource` is owned by `feats-behaviour.md`; this spec supplies `BackgroundFeat`
(FR-105) and `SpeciesTraitFeat` (FR-108) only. Q-2 the host stores the `CreationRecord` beside the `Character`: the
aggregate stores no skills, languages, feats, alignment or equipment (`character-aggregate.md` D-9), and no creation
event carries it (FR-095). Q-3 the generic class skill, tool and equipment-letter checks are owned here (FR-015 to
FR-020, FR-111); class choices types keep only class-specific state and map their rejections to `ClassSkillCountWrong`
and `ClassEquipmentOptionUnknown` (amendments to the class specs). Q-11 `EquipmentOptionLetter` is a `character`-context
enum (`a`, `b`, `c` to keys `A`, `B`, `C`, FR-020); it stays here because only creation reads the keys as a choice,
which keeps `content` free of a player-input type (design). Q-13 the Standard Array by Class table (p. 21) is advice,
not a rule (the text says the array may be assigned freely): not modelled, so no class name enters `character` code;
the rows are test data, and `content-classes.md` may carry them as data checked to be permutations of
`StandardArray()` (all 12 rows checked against the PDF). Q-14 starting coins stay coins (FR-070, FR-086); purchases,
carrying capacity and Armor Class from worn items belong to a later inventory spec and are out of scope (p. 20 gives no
spending rule beyond "buy equipment"). Q-17 `BonusFeatsAtLevel20` has no consumer FR here; the recorded exception to
`rule-options.md` FR-029 is accepted as permanent for this spec (exporting the option type is harmless; its consumer
is `feats-behaviour.md`, amendment). Q-18 a category named by a background or class resolves through
`content-equipment.md`: "Artisan's Tools" by `ToolGroup` (FR-113), "Gaming Set" and "Musical Instrument" by `ToolFamily`
and `VariantOf` (FR-030), so FR-034 and FR-111 need no text comparison. Q-21 only placeholders tied to a prior choice are
resolved (FR-109, FR-110); the Bard's "Musical Instrument of your choice" (p. 31) stays as printed and is flagged in the
summary (interpretation: p. 31 does not link it to the three instrument proficiencies). Q-26 closed by `class-monk.md`
FR-143: the Monk returns an empty key list, its tool is a command `ChosenTool` (FR-110, FR-111).

## 10. Changelog impact

`CHANGELOG.md`, under `Unreleased` / **Added**: "`character`: character creation (`CreateCharacter`,
`CreateCharacterCommand`, `CreationRecord`, the three results); ability score generation (Standard Array,
`RollAbilityScores`, Point Cost); background increases; `RollLanguages`; starting equipment, the higher-level start
and the trinket roll; exports of `CharacterRuleOptions` and `AbilityScoreGenerationMethod`; telemetry
`srd5e.character.roll_ability_scores`, `.roll_languages`, `.ability_score_roll` and new attribute values on
`srd5e.character.create` and `.operations`." Not **BREAKING**: first release of this context (Q-4).
