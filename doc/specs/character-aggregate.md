# Character aggregate

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `ability-scores.md`, `derived-stats.md`, `dice-and-rolls.md`, `observability-ports.md`, `content-source-port.md`, `content-classes.md`, `content-schema-and-decoder.md`, `errors-and-results.md`, `rule-options.md` |

## 1. Purpose and scope

Defines the `Character` aggregate of the `character` context: root, typed id, invariants, domain events and the state-in /
new-state-and-events-out transitions for creating a character, gaining Experience Points, gaining a level and changing an
ability score. The use cases wrapping the level, XP and ability score transitions are in section 6. `CreateCharacter` is
`character-creation.md` (decisions FR-048); this spec keeps `character.New` (and `MustNew`), the Go form of the earlier create and parse pair.

The aggregate holds only what the rules tie to identity and advancement: ability scores, Experience Points, the ordered
`ClassLevels` (first entry is the initial class), the stored `HitPointMaximum` and `HitDicePool`, `HeroicInspiration` and
the `FeatureStates` slot that class specs fill. There is **no `LevelHistory`** and no per-level gain is stored (decisions
FR-008); each transition maintains the stored Hit Point maximum (D-11). It depends on `derived-stats.md` (`character` order,
decisions FR-016), never the reverse. It owns `CharacterID`, `ExperiencePoints` (with the Character Advancement XP column),
`ExperienceAmount`, `HeroicInspiration`, `FeatureStates`, the root and its transitions, the class-feature signals (FR-118)
and `CharacterRuleOptions` (4.10; decisions FR-057; pattern in `rule-options.md`).

**Out of scope:**

- Derived stats (Armor Class, save and skill totals, Passive Perception, attack bonuses, spell save DC): `derived-stats.md`;
  `HitPointMaximumBonus` is an input there and `Character.HitPointMaximum` excludes it. Spellcasting and the Multiclass
  Spellcaster table (p. 26): `spellcasting.md`.
- Class features, subclass choice (`level-advancement-and-multiclassing.md`, decisions FR-020) and feature choices on
  gaining a level (p. 23): one spec per class; only the `FeatureStates` slot is here (FR-092). Rests and spending Hit Dice:
  `rests-and-hit-dice.md`; feats: `feats-behaviour.md`; current and temporary Hit Points and Death Saving Throws: `combat`.
- Heroic Inspiration mechanics: `heroic-inspiration.md` (decisions FR-048); only the state slot (decisions FR-010) and two
  primitive transitions (FR-089 to FR-091) are here. `d20` owns `Roll` and `RerollDecision` (`d20-tests.md` FR-059).
- `CreateCharacter`, ability score generation (p. 21) and creation choices: `character-creation.md`. Background, species
  details, languages, alignment, equipment, feats, proficiencies, name and appearance (species and background are held by id
  only); `ClassDefinition` and `AbilityRequirement` (`content-classes.md`, decisions FR-025).
- Optional "Bonus Feats at Level 20" behaviour (p. 24; only its option is in 4.10); Extra Attack and Armor Class
  interactions of multiclassing (p. 25); any decrease of a Constitution score or modifier (FR-049); persistence and id
  generation (the host supplies ids; D-1).

## 2. Source

PDF page numbers equal printed page numbers (printed page 19 is PDF page 19).

- SRD 5.2.1, "Ability Scores", p. 5; "Ability Modifiers", pp. 5-6; "Round Down", p. 5
- SRD 5.2.1, "Heroic Inspiration", p. 8; Rules Glossary, "Heroic Inspiration", p. 183; "Proficiency" and its Proficiency
  Bonus table, p. 8
- SRD 5.2.1, "Create Your Character" and "Step 1: Choose Class" (Class Overview table, Primary Ability), p. 19; "Step 3:
  Ability Scores", "Generate Your Scores", p. 21
- SRD 5.2.1, "Step 5: Character Creation Details" ("Hit Points", "Hit Point Dice"), Level 1 Hit Points by Class table, p. 22
- SRD 5.2.1, "Level Advancement", Character Advancement table, "Gaining a Level", Fixed Hit Points by Class table, p. 23
- SRD 5.2.1, "Starting at Higher Levels", "Bonus Feats at Level 20", "Multiclassing" ("Prerequisites"), p. 24;
  "Multiclassing" ("Experience Points", "Hit Points and Hit Point Dice", "Proficiency Bonus"), p. 25
- Class pages, "Core ... Traits" tables (Hit Point Die per class), pp. 28-80
- Package design: `doc/rules/architecture.md` (§3, §4, §6), `doc/rules/observability.md` (§2, §5, §6, §7),
  `doc/rules/testing.md` (§4); `shared-types-and-ownership.md` (FR-001, FR-008, FR-010, FR-013 to FR-025, FR-036 to FR-043).

## 3. Vocabulary

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

| SRD term | Type | Notes |
|---|---|---|
| Character | Aggregate root `Character` | Identity `CharacterID` (not an SRD term; typed id per architecture §3). |
| Experience Points | `ExperiencePoints` (here: FR-004, FR-006 to FR-008) | Non-negative; owns the Character Advancement XP column. |
| (XP award) | `ExperienceAmount` | Positive whole number (FR-022, FR-081). |
| Ability Scores / modifier | `ability.Scores`, `ability.Score`, `ability.Modifier` (`ability-scores.md`) | Constitution modifier drives Hit Points. |
| Class, Species, Background | `ClassID`, `SpeciesID`, `BackgroundID` (`content-schema-and-decoder.md` FR-114; decisions FR-005, FR-046) | Held by id only. |
| Hit Point Die | `HitPointDie` (d6, d8, d10, d12, `faces`; `content-schema-and-decoder.md` FR-132, decisions FR-053; D-20) | From `ClassDefinition.HitPointDie`; an out-of-set die is unrepresentable (FR-018). |
| Hit Points / Hit Dice | `HitPointMaximum`, `HitDicePool` (`derived-stats.md`) | Stored (FR-096); pooled per die size (p. 25). Current Hit Points belong to `combat`. |
| (Hit Point gain) | `HitPointGain`: `RolledHitPointGain(value, die)`, `FixedHitPointGain(value)` (`derived-stats.md` FR-062) | Input to `LevelUp`; `amount(ability.Modifier)` is the level's contribution. |
| Proficiency Bonus | `ProficiencyBonus` (`derived-stats.md`) | `Character.ProficiencyBonus` (FR-010). |
| Heroic Inspiration | closed `HeroicInspiration` (`Inspired`, `NotInspired`) | Slot here; mechanics in `heroic-inspiration.md`; pp. 8, 183. |
| (class feature state) | `FeatureStates`, `FeatureState`, `FeatureStateKey<T>` | Slot and read/write API only (decisions FR-022). |
| (creation choices) | `ClassCreationChoices`, `NoClassCreationChoices` | Carried by the `CreateCharacter` command (decisions FR-024). |
| Primary Ability | `AbilityRequirement` (`AllOf`, `AnyOf`; `content-classes.md`, decisions FR-025) | p. 19: "and" classes are all-of, the Fighter "or" is any-of (FR-031). |
| (domain event) | `CharacterCreated`, `ExperiencePointsGained`, `LevelGained`, `AbilityScoreChanged`, `HitPointMaximumChanged`, `HeroicInspirationGained`, `HeroicInspirationExpended` | Past tense, immutable; members of closed `CharacterRecordEvent`, a sub-family of open `Event` (FR-085). |
| Standard Array, Random Generation, Point Cost | `AbilityScoreGenerationMethod` (enum: `StandardArray`, `RandomGeneration`, `PointCost`), owned here (decisions FR-057) | p. 21; only which methods the table permits (FR-105). |
| Starting at Higher Levels | `CharacterRuleOptions.StartingLevel` | p. 24 (FR-107). |
| Bonus Feats at Level 20 | closed `BonusFeatsAtLevel20`: `NoBonusFeats`, `BonusFeatsFromXp` | p. 24; default `NoBonusFeats` (the SRD says a GM "can" use it). Variant names are package terms. |

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids.

### 4.1 Typed id

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | Two `CharacterID`s built from equal values MUST be equal with `==`; different values MUST be unequal. | n/a (architecture §3) | happy |
| FR-002 | The `CharacterID` constructor MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) for an empty or whitespace-only value. | n/a (architecture §3, §6) | unhappy (invalid input) |
| FR-003 | `NewCharacterID(string) (CharacterID, error)` MUST return the id, or an error naming the single typed reason (FR-101). It MUST NOT panic; `MustCharacterID` panics with that error. | n/a (architecture §6) | unhappy (invalid input) |
| FR-073 | `NewCharacterID` MUST NOT panic for any string; `MustCharacterID` MUST panic if and only if `NewCharacterID` returns an error for the same string, and the panic value is that error. | n/a (errors-and-results.md FR-025, FR-026) | unhappy (invalid input) |

### 4.2 Level, Experience Points and Proficiency Bonus

`ExperiencePoints` and the XP column are owned here. `CharacterLevel` and `ProficiencyBonus` (with its per-level band) are
owned by `derived-stats.md` (decisions FR-008).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-004 | `ExperiencePoints.MinimumForLevel(CharacterLevel)` MUST return the minimum XP of the Character Advancement table for every level 1 to 20 (table below). `Character` MUST obtain every threshold from this type and MUST NOT hold a second copy. | SRD 5.2.1, "Level Advancement", Character Advancement table, p. 23 | happy |
| FR-006 | `ExperiencePoints.LevelFor` MUST return the highest level whose minimum the total equals or exceeds. | SRD 5.2.1, "Level Advancement", p. 23 ("equals or exceeds") | unhappy (boundary) |
| FR-007 | `ExperiencePoints.LevelFor` MUST return level 20 for a total above 355,000. | SRD 5.2.1, "Level Advancement", p. 23 | unhappy (boundary) |
| FR-008 | `NewExperiencePoints(int) (ExperiencePoints, error)` MUST reject a negative value with a validation error and accept 0; it MUST NOT panic; `MustExperiencePoints` panics if and only if it returns an error. | SRD 5.2.1, "Create Your Character", p. 19 ("0 XP") | unhappy (invalid input) |
| FR-010 | `Character.ProficiencyBonus` MUST return `ProficiencyBonus.ForLevel` (`derived-stats.md`) of the total level, never of a single class level. Cases: level 3 Fighter / level 2 Rogue gives +3 (the SRD example); level 4 Cleric / level 4 Fighter gives +3; level 1 Wizard gives +2; level 5 single class gives +3. | SRD 5.2.1, "Multiclassing" ("Proficiency Bonus"), p. 25 | unhappy (rule interaction) |

Character Advancement XP minimums (FR-004, FR-006, FR-007), level: minimum XP: 1: 0; 2: 300; 3: 900; 4: 2,700; 5: 6,500; 6:
14,000; 7: 23,000; 8: 34,000; 9: 48,000; 10: 64,000; 11: 85,000; 12: 100,000; 13: 120,000; 14: 140,000; 15: 165,000; 16:
195,000; 17: 225,000; 18: 265,000; 19: 305,000; 20: 355,000.

`LevelFor` boundary cases: 0 and 299 give level 1; 300 gives level 2; 354,999 gives level 19; 355,000 and 400,000 give level
20.

### 4.3 Creating a character

Creation produces a level 1 character in one class. A character who starts higher is created with the XP for that level and
then gains levels with `LevelUp` (FR-014). `character.New` returns `NewCharacter` (the `Character` and a
list holding the one `CharacterCreated` event).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-011 | `character.New` MUST return a `NewCharacter` whose character is at total level 1 holding the supplied id, `ClassID`, `SpeciesID`, `BackgroundID` and ability scores, with `ClassLevels` of one entry (the initial class, level 1; FR-095), the supplied starting Experience Points (default 0), `HeroicInspiration` `NotInspired` unless supplied (FR-089), and the `FeatureStates` supplied (default empty; FR-092, FR-094). | SRD 5.2.1, "Create Your Character", p. 19 ("A level 1 character has 0 XP") | happy |
| FR-012 | The initial `Character.HitPointMaximum` MUST equal the maximum face of the initial class's Hit Point Die plus the Constitution modifier, stored as the `HitPointMaximum` (FR-096; cases below). | SRD 5.2.1, "Step 5: Character Creation Details" ("Hit Points"), p. 22 | happy |
| FR-013 | The initial `HitDicePool` MUST contain exactly one Hit Die, of the initial class's die size. | SRD 5.2.1, "Step 5" ("Hit Point Dice"), p. 22 | happy |
| FR-014 | A character created with the minimum XP for level N (FR-004) MUST be eligible to gain levels 2, ..., N through repeated `LevelUp` without further XP. | SRD 5.2.1, "Starting at Higher Levels", p. 24 | happy |
| FR-015 | `character.New` MUST emit exactly one `CharacterCreated` event carrying the character id, `ClassID`, `SpeciesID`, `BackgroundID`, ability scores, starting XP and initial Hit Point maximum. | n/a (architecture §3, domain events) | happy |
| FR-016 | Each starting ability score MUST be from 1 to 20. The lower bound is enforced by `ability.Score` (`ability-scores.md` FR-005 to FR-007, FR-009); a score above 20 MUST make `character.New` return an error that wraps `srderr.ErrInvalidInput` and holds a `*StartingScoreAboveMaximum` problem listing the abilities (FR-082, FR-100). | SRD 5.2.1, "Ability Scores", p. 5 | unhappy (boundary) |
| FR-018 | The Hit Point Die is the `content` enum `HitPointDie` (`content-schema-and-decoder.md` FR-132; decisions FR-053), supplied by `ClassDefinition.HitPointDie`; this spec defines no die type. A die outside the four the SRD gives values for (pp. 22-23) cannot be passed to `character.New` or `LevelUp`, so no runtime check exists. The level 1 base is `die.faces` and the fixed value is `die.faces` / 2 + 1 (d6 4, d8 5, d10 6, d12 7), derived, never stored. | SRD 5.2.1, Level 1 Hit Points by Class table, p. 22; Fixed Hit Points by Class table, p. 23 | happy |

Level 1 Hit Points cases (FR-012; classes are samples from p. 22, modifiers from p. 6), as class, die, Constitution score
(modifier): expected maximum. Barbarian d12, 14 (+2): 14. Fighter d10, 14 (+2): 12. Bard d8, 10 (+0): 8. Wizard d6, 8 (-1):
5. Wizard d6, 1 (-5): 1. Barbarian d12, 20 (+5): 17.

Boundary creation cases (FR-012, FR-018): d6 at Constitution 1 (-5) is allowed, maximum 1; every `HitPointDie` at
Constitution 10 (+0) gives maximum 6, 8, 10, 12. A level 1 contribution below 1 cannot arise (lowest face count 6, lowest
modifier -5); the `Character` constructor check of FR-084 is the only guard (D-3).

### 4.4 Awarding Experience Points

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-020 | `AwardExperience(amount)` MUST return a `CharacterChanged` whose character has XP equal to the old total plus `amount`, and one `ExperiencePointsGained` event (character id, amount, new total). | SRD 5.2.1, "Level Advancement", p. 23 | happy |
| FR-021 | Awarding XP MUST NOT change total level, `ClassLevels`, Hit Point maximum, Hit Dice pool, Heroic Inspiration or feature states. A level is gained only by `LevelUp`. | SRD 5.2.1, "Gaining a Level", p. 23 | unhappy (rule interaction) |
| FR-022 | `ExperienceAmount` MUST be a positive whole number; its constructor MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) for zero or a negative value. `AwardExperience` takes an `ExperienceAmount`, so an award cannot be rejected as a rule outcome. | n/a (architecture §6) | unhappy (invalid input) |
| FR-023 | Awards that take the total past 355,000 MUST succeed; total level is unaffected (the maximum is enforced by `LevelUp`, FR-030). | SRD 5.2.1, "Level Advancement", p. 23; "Bonus Feats at Level 20", p. 24 | unhappy (boundary) |
| FR-024 | `Character.IsEligibleForNextLevel` MUST be true exactly when total level is below 20 and XP is at least the minimum for `totalLevel + 1` (cases below). | SRD 5.2.1, "Level Advancement", p. 23; "Experience Points" under Multiclassing, p. 25 | unhappy (boundary) |
| FR-081 | `NewExperienceAmount(int)` MUST NOT panic; `MustExperienceAmount` MUST panic if and only if `NewExperienceAmount` returns an error for the same value. | n/a (errors-and-results.md FR-025, FR-026) | unhappy (invalid input) |

Eligibility cases (FR-024), as total level, XP: eligible. Level 1: 299 no; 300 yes; 6,500 yes. Level 19: 354,999 no; 355,000
yes. Level 20: 400,000 no.

### 4.5 Gaining a level

`LevelUp` takes the character, an `AdvancementClass`, a `HitPointGain` and, when the class is new,
`RequirementsOfCurrentClasses` (a map from `ClassID` to `AbilityRequirement` for each current class). It returns
`CharacterTransition<LevelUpRejection>`, closed: `CharacterChanged` (new character, events) or `CharacterRejected` (a
`LevelUpRejection`). `LevelUpRejection` is closed with `LevelUpMaximumLevel`, `LevelUpInsufficientExperience` and
`LevelUpPrerequisitesNotMet(shortAbilities)` (a slice in `Ability` declaration order, without duplicates, copied on read).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-025 | A level-up in a class the character already has MUST return a character whose level in that class and total level are each one higher, and one `LevelGained` event. The new `ClassLevels` is built through the `ClassLevels` constructor (`derived-stats.md` FR-058) from the old entries with that class raised by one and every position kept. | SRD 5.2.1, "Gaining a Level", p. 23 | happy |
| FR-026 | A level-up in a class the character does not have MUST add that class at level 1 and raise total level by one, when the prerequisites of FR-031 hold, building the new `ClassLevels` through its constructor (`derived-stats.md` FR-058) with the class appended last. | SRD 5.2.1, "Multiclassing", p. 24 | happy |
| FR-027 | A level-up MUST be rejected with `LevelUpInsufficientExperience` when `IsEligibleForNextLevel` is false and total level is below 20. | SRD 5.2.1, "Experience Points" under Multiclassing, p. 25 | unhappy (state error) |
| FR-028 | Eligibility MUST use total character level, not the level in the chosen class: a level 6 Cleric / level 1 Fighter needs the XP for level 8 to gain level 8, whichever class is chosen. | SRD 5.2.1, "Experience Points" under Multiclassing, p. 25 | unhappy (rule interaction) |
| FR-029 | A level-up with XP exactly at the minimum for `total + 1` MUST succeed; one below MUST be rejected. | SRD 5.2.1, "Level Advancement", p. 23 | unhappy (boundary) |
| FR-030 | A level-up at total level 20 MUST be rejected with `LevelUpMaximumLevel`, regardless of XP. | SRD 5.2.1, "Level Advancement", p. 23 | unhappy (boundary) |
| FR-031 | Gaining a level in a class the character does not have MUST require that the `AbilityRequirement` of the new class and of every current class are all met, where a score of at least 13 meets an ability. `AllOf` is met when every listed ability is at least 13; `AnyOf` when at least one is. Otherwise the level-up MUST be rejected with `LevelUpPrerequisitesNotMet`, whose `ShortAbilities` holds every ability of every unmet requirement that is below 13 (for an unmet `AnyOf`, all its abilities), without duplicates (cases below). | SRD 5.2.1, "Multiclassing" ("Prerequisites"), p. 24; Class Overview table, p. 19 | unhappy (boundary) |
| FR-032 | The prerequisite check of FR-031 MUST NOT apply when the character already has the chosen class. | SRD 5.2.1, "Multiclassing" ("Prerequisites"), p. 24 | unhappy (rule interaction) |
| FR-033 | The Hit Point gain of a level after the first with a `RolledHitPointGain` MUST be `HitPointGain.amount(constitutionModifier)` (`derived-stats.md` FR-062: value plus Constitution modifier, minimum of 1); this spec keeps only the case tables. | SRD 5.2.1, "Gaining a Level" step 2, p. 23 ("minimum of 1") | happy |
| FR-034 | The gain with a `FixedHitPointGain` MUST be `HitPointGain.amount(constitutionModifier)` as in FR-033, the fixed value being the die's fixed value (FR-018); the minimum of 1 is owned by `derived-stats.md` FR-062 (D-2). | SRD 5.2.1, "Gaining a Level" step 2 and Fixed Hit Points by Class table, p. 23 | happy |
| FR-035 | `LevelUp` MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) when a `RolledHitPointGain`'s `die` differs from the chosen class's `HitPointDie`, or when a `FixedHitPointGain`'s value differs from `die.faces` / 2 + 1 for that class (so `FixedHitPointGain(100)` cannot succeed). The rolled value's range is checked by `RolledHitPointGain` (`derived-stats.md` FR-062); a roll from the `dice.Roller` port outside it is FR-079. | n/a (constructor invariant); SRD 5.2.1, "Gaining a Level" step 2, p. 23 | unhappy (invalid input) |
| FR-036 | After a level-up, `Character.HitPointMaximum` MUST equal the maximum before plus the Hit Points gained and MUST NOT be recomputed (FR-096). | SRD 5.2.1, "Gaining a Level" step 2, p. 23 | happy |
| FR-037 | A level-up MUST add one Hit Die of the class's size by building the new `HitDicePool` through its constructor (`derived-stats.md` FR-070) from the old per-die counts with the chosen die's count raised by one (absent counts as 0). | SRD 5.2.1, "Gaining a Level" step 2, p. 23 | happy |
| FR-038 | Pooling is owned by `derived-stats.md` FR-061. `LevelUp` adds its die as FR-037 says, so a level 5 Fighter / level 5 Paladin reached by repeated `LevelUp`s holds ten d10s, and a level 5 Cleric / level 5 Paladin holds five d8s and five d10s. | SRD 5.2.1, "Hit Points and Hit Point Dice" under Multiclassing, p. 25 | unhappy (rule interaction) |
| FR-039 | A level-up in a class the character does not have MUST add the gain of FR-033 or FR-034 and MUST NOT add the class's level 1 maximum-face value. | SRD 5.2.1, "Hit Points and Hit Point Dice" under Multiclassing, p. 25 ("only when your total character level is 1") | unhappy (rule interaction) |
| FR-040 | The `LevelGained` event MUST carry the character id, `ClassID`, new class level, new total level, Hit Points gained, new Hit Point maximum and the Proficiency Bonus at the new total level. | SRD 5.2.1, "Gaining a Level" steps 2 and 4, p. 23 | happy |
| FR-041 | When the level-up crosses a Proficiency Bonus boundary (total level 4 to 5, 8 to 9, 12 to 13, 16 to 17), the event's Proficiency Bonus MUST be the new, higher value; elsewhere unchanged. | SRD 5.2.1, "Gaining a Level" step 4, p. 23 | unhappy (boundary) |
| FR-042 | A rejected level-up MUST return the rejection and no events, and the input character MUST be unchanged. | n/a (architecture §3, §6) | unhappy (state error) |
| FR-043 | When several rejection conditions hold, the reason MUST be chosen in this order: `LevelUpMaximumLevel`, `LevelUpInsufficientExperience`, `LevelUpPrerequisitesNotMet` (cases below; D-4). | n/a (determinism) | unhappy (rule interaction) |

Level-up Hit Point gain cases (FR-033, FR-034): expected `HitPointGain.amount` values; fixed values follow p. 23 through the
die (FR-018), class rows are samples.

| Label | Mode | Die (faces) | Roll / fixed | Con score | Con mod | Expected gain |
|---|---|---|---|---|---|---|
| roll average | rolled | 8 | 5 | 14 | +2 | 7 |
| roll maximum | rolled | 8 | 8 | 10 | +0 | 8 |
| roll minimum | rolled | 8 | 1 | 10 | +0 | 1 |
| roll one, Con -1 | rolled | 8 | 1 | 8 | -1 | 1 (clamped from 0) |
| roll two, Con -5 | rolled | 6 | 2 | 1 | -5 | 1 (clamped from -3) |
| roll exactly clamp edge | rolled | 6 | 3 | 6 | -2 | 1 |
| roll one above clamp edge | rolled | 6 | 4 | 6 | -2 | 2 |
| fixed Barbarian | fixed | 12 | 7 | 14 | +2 | 9 |
| fixed Fighter | fixed | 10 | 6 | 14 | +2 | 8 |
| fixed Bard | fixed | 8 | 5 | 10 | +0 | 5 |
| fixed Wizard | fixed | 6 | 4 | 10 | +0 | 4 |
| fixed Wizard, Con -5 | fixed | 6 | 4 | 1 | -5 | 1 (clamped from -1) |

Gain-validation cases (FR-035):

| Label | Class die | Gain | Expected |
|---|---|---|---|
| roll of another die | d8 | `RolledHitPointGain(5, d10)` | validation error |
| roll of the class die | d8 | `RolledHitPointGain(5, d8)` | accepted |
| fixed value 100 | d10 | `FixedHitPointGain(100)` | validation error |
| fixed value of the d8 | d10 | `FixedHitPointGain(5)` | validation error |
| fixed value of the d10 | d10 | `FixedHitPointGain(6)` | accepted |
| fixed value one above | d6 | `FixedHitPointGain(5)` | validation error |

Multiclass prerequisite cases (FR-031, FR-032). The first rows follow the p. 24 example (Barbarian into Druid needs Strength
and Wisdom 13). A class listing "or" is `AnyOf`; the rest are `AllOf`.

| Label | Existing class primary | New class primary | Scores | Expected |
|---|---|---|---|---|
| example from the SRD | Strength | Wisdom | STR 13, WIS 13 | allowed |
| existing class ability short | Strength | Wisdom | STR 12, WIS 13 | rejected (Strength) |
| new class ability short | Strength | Wisdom | STR 13, WIS 12 | rejected (Wisdom) |
| both short | Strength | Wisdom | STR 12, WIS 12 | rejected (Strength, Wisdom) |
| comfortably above | Strength | Wisdom | STR 15, WIS 18 | allowed |
| same class, scores low | Strength | Strength (existing class chosen) | STR 8 | allowed (FR-032) |
| class with two primary abilities | Dexterity and Wisdom | Charisma | DEX 13, WIS 13, CHA 13 | allowed |
| class with two primary abilities, one short | Dexterity and Wisdom | Charisma | DEX 13, WIS 12, CHA 13 | rejected (Wisdom) |
| Fighter as new class, Dexterity only | Wisdom | Fighter: Strength or Dexterity | WIS 13, STR 8, DEX 13 | allowed |
| Fighter as new class, Strength only | Wisdom | Fighter: Strength or Dexterity | WIS 13, STR 13, DEX 8 | allowed |
| Fighter as new class, both short | Wisdom | Fighter: Strength or Dexterity | WIS 13, STR 12, DEX 12 | rejected (Strength, Dexterity) |
| Fighter as existing class, satisfied by Dexterity | Fighter: Strength or Dexterity | Intelligence | STR 8, DEX 13, INT 13 | allowed |
| Fighter as existing class, new ability short | Fighter: Strength or Dexterity | Intelligence | STR 8, DEX 13, INT 12 | rejected (Intelligence) |
| two classes unmet, deduplicated | Strength | Strength and Wisdom | STR 10, WIS 10 | rejected (Strength, Wisdom) |

Combined rejection cases (FR-043; level 6 needs 14,000 XP):

| Label | Total level | XP | New class | Expected |
|---|---|---|---|---|
| maximum beats everything | 20 | 400,000 | unmet | `LevelUpMaximumLevel` |
| maximum, at the threshold | 20 | 355,000 | none | `LevelUpMaximumLevel` |
| experience beats prerequisites | 5 | 6,500 | unmet | `LevelUpInsufficientExperience` |
| experience, one short | 5 | 13,999 | none | `LevelUpInsufficientExperience` |
| prerequisites unmet | 5 | 14,000 | unmet | `LevelUpPrerequisitesNotMet` |
| prerequisites met | 5 | 14,000 | met | `CharacterChanged` |
| existing class, check skipped | 5 | 14,000 | existing | `CharacterChanged` |

Proficiency Bonus boundary cases (FR-041), total level before to after: bonus before to after. 4 to 5: +2 to +3; 8 to 9: +3
to +4; 12 to 13: +4 to +5; 16 to 17: +5 to +6; 1 to 2: +2 to +2; 19 to 20: +6 to +6.

### 4.6 Changing an ability score

An ability score change (for example from a feat) arrives as a new `ability.Score` for one ability; what grants it is out of
scope. `ChangeAbilityScore` returns `CharacterTransition<AbilityScoreChangeRejection>`, closed with
`AbilityScoreChangeExceedsMaximum` (carrying the cap applied) and `AbilityScoreChangeConstitutionDecrease`. It is unrelated
to `AbilityScoreRejection` (construction bounds), owned by `ability-scores.md` (decisions FR-001).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-044 | `ChangeAbilityScore` MUST return a `CharacterChanged` with the new score for that ability only and one `AbilityScoreChanged` event (character id, ability, old score, new score). | SRD 5.2.1, "Gaining a Level" step 5, p. 23 | happy |
| FR-045 | When the new Constitution modifier is higher than the old, the stored `HitPointMaximum` MUST increase by (new modifier minus old modifier) times the total level, and a `HitPointMaximumChanged` event (character id, old, new) MUST follow the `AbilityScoreChanged` event (cases below). No per-level gain is consulted (FR-096). | SRD 5.2.1, "Gaining a Level" step 5, p. 23 ("When your Constitution modifier increases by 1, your Hit Point maximum increases by 1 for each level you have attained") | happy |
| FR-046 | When the Constitution score changes but its modifier does not, or any other ability changes, the Hit Point maximum MUST NOT change and no `HitPointMaximumChanged` is emitted. Whether the modifier changes is read from the Ability Modifiers table, not from the parity of the score. | SRD 5.2.1, "Ability Modifiers", p. 6; "Gaining a Level" step 5, p. 23 | unhappy (rule interaction) |
| FR-047 | A new score above the cap MUST be rejected with `AbilityScoreChangeExceedsMaximum`. The cap defaults to 20 and a command MAY carry a higher cap (FR-074) standing for a feature that says otherwise. | SRD 5.2.1, "Ability Scores", p. 5 ("20 ... the highest an adventurer's score can go unless a feature says otherwise"; 30 is the highest a score can go) | unhappy (boundary) |
| FR-048 | A score below 1 or above 30 cannot be represented as an `ability.Score` (`ability-scores.md` FR-009, FR-010). `ChangeAbilityScoreCommand` takes an `ability.Score`; hosts with a raw integer use `ability.NewScore`. | SRD 5.2.1, "Ability Scores", p. 5 (scores run 1 to 30) | unhappy (invalid input) |
| FR-049 | A change that lowers the Constitution modifier MUST be rejected with `AbilityScoreChangeConstitutionDecrease`: the SRD gives no Hit Point rule for it (D-5). | SRD 5.2.1, "Gaining a Level" step 5, p. 23 (only increases are described) | unhappy (state error) |
| FR-050 | Setting an ability to its current score MUST return a `CharacterChanged` holding an equal character and an empty event list: a successful no-op, not a rejection (D-14). | n/a (architecture §3) | unhappy (boundary) |
| FR-074 | A cap carried by a command MUST be from 20 to 30 inclusive; a cap below 20 or above 30 MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`). | SRD 5.2.1, "Ability Scores", p. 5 | unhappy (boundary) |
| FR-075 | When several conditions hold for one change, the outcome MUST be chosen in this order: no-op success (FR-050), `AbilityScoreChangeExceedsMaximum`, `AbilityScoreChangeConstitutionDecrease` (cases below; D-4). | n/a (determinism) | unhappy (rule interaction) |
| FR-084 | `Character.HitPointMaximum` MUST be at least the total level for every valid character. Every transition preserves this (level 1 gives at least 1, FR-012; each `LevelUp` adds at least 1; a Constitution rise only adds), and the `Character` constructor MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) for a stored maximum below the total level. | SRD 5.2.1, "Gaining a Level" step 2, p. 23 ("minimum of 1") | unhappy (boundary) |

Constitution change cases (FR-045, FR-046, FR-049, FR-050): single-class Fighter, fixed values; the maximum changes by
(modifier change) times total level; modifiers from p. 6.

| Label | Score, old to new | Level | Maximum change | Events |
|---|---|---|---|---|
| SRD example | Con 17 to 18 | 8 | +8 (76 to 84) | `AbilityScoreChanged`, `HitPointMaximumChanged` |
| level 1 | Con 17 to 18 | 1 | +1 | both |
| modifier +2 to +3 | Con 15 to 17 | 5 | +5 | both |
| two-step rise | Con 14 to 18 | 3 | +6 | both |
| level 20 | Con 17 to 18 | 20 | +20 | both |
| modifier unchanged | Con 16 to 17 | 8 | 0 | `AbilityScoreChanged` only |
| lower, same modifier | Con 17 to 16 | 8 | 0 | `AbilityScoreChanged` only |
| other ability | Str 10 to 12 | 8 | 0 | `AbilityScoreChanged` only |
| lowering the modifier | Con 18 to 17 | 8 | rejected (FR-049) | none |
| same score | Con 17 to 17 | 8 | 0 (FR-050) | none |

Cap and order cases (FR-047, FR-050, FR-074, FR-075):

| Label | Score, old to new | Cap | Expected |
|---|---|---|---|
| default cap edge | Str 19 to 20 | 20 | changed |
| over default cap | Str 19 to 21 | 20 | `AbilityScoreChangeExceedsMaximum` |
| raised cap | Str 20 to 22 | 24 | changed |
| at raised cap | Str 20 to 24 | 24 | changed |
| over raised cap | Str 20 to 25 | 24 | `AbilityScoreChangeExceedsMaximum` |
| cap at 30 | Str 29 to 30 | 30 | changed |
| cap out of range | any change | 19 or 31 | validation error (FR-074) |
| same score above cap | Str 22 to 22 | 20 | changed, no events (no-op wins) |
| lowering, still over | Str 24 to 22 | 20 | `AbilityScoreChangeExceedsMaximum` |
| over cap and a decrease | Con 24 to 21 | 20 | `AbilityScoreChangeExceedsMaximum` (cap first) |
| Con decrease within cap | Con 18 to 17 | 20 | `AbilityScoreChangeConstitutionDecrease` |

### 4.7 Invariants, immutability and determinism

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-051 | Total level MUST be the sum of the `ClassLevels` and be from 1 to 20 (`CharacterLevel`). A `Character` built (for example by a decoder) with empty `ClassLevels` or a sum above 20 MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`); the constructor MUST NOT accept a value that bypasses the `ClassLevels` type's own checks. | SRD 5.2.1, "Multiclassing" ("Experience Points"), p. 25; Character Advancement table, p. 23 | unhappy (invalid input) |
| FR-052 | A `Character` MUST have at least one class (the first `ClassLevels` entry; FR-095), and its level in each class it has is at least 1. | SRD 5.2.1, "Create Your Character" and "Step 1: Choose Class", p. 19 | unhappy (invalid input) |
| FR-053 | `Character.HitDicePool` MUST contain exactly as many dice as the total level (1 to 20); a `Character` built with a pool of another size MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`). | SRD 5.2.1, "Gaining a Level" step 2, p. 23; "Hit Point Dice", p. 22 | unhappy (boundary) |
| FR-054 | A `Character` is immutable: every transition returns a new instance and leaves the input equal to its prior value. | n/a (architecture §3) | unhappy (immutability) |
| FR-055 | `Character`, its value objects and every event MUST compare by value. `Character` equality includes the `ClassLevels` in order (a different initial class is unequal), the `FeatureStates` and `HeroicInspiration`. | n/a (architecture §3) | happy |
| FR-056 | The collections exposed by `Character` (`ClassLevels`, `HitDicePool`, `FeatureStates`, ability scores) and the event lists in `CharacterChanged` and `NewCharacter` MUST be copies; a change to a returned slice or map has no effect on the character. | n/a (architecture §4) | unhappy (immutability) |
| FR-057 | The same character, command and Hit Point gain MUST give an equal `CharacterTransition` on every call (no clock, randomness or ordering dependence). | n/a (architecture §3, purity) | unhappy (determinism) |
| FR-058 | Event order within one transition MUST be fixed: `AbilityScoreChanged` before `HitPointMaximumChanged`; `CharacterCreated`, `LevelGained`, `ExperiencePointsGained`, `HeroicInspirationGained` and `HeroicInspirationExpended` stand alone. | n/a (determinism) | unhappy (determinism) |
| FR-059 | A rejected `LevelUp` or `ChangeAbilityScore` MUST NOT panic or return an `error`; invalid arguments (FR-022, FR-035, FR-074, FR-100, FR-102) return a validation error and are never results. `AwardExperience` has no rejection. | n/a (architecture §6) | unhappy (state error) |
| FR-082 | `character.New(params) (NewCharacter, error)` is an aggregate parser (errors-and-results.md FR-023) and reports every problem at once: on failure the `error` is the `errors.Join` of the typed `CharacterCreationProblem` values of FR-100 (non-empty, in declaration order, each variant at most once) and wraps `srderr.ErrInvalidInput`. It MUST NOT panic for any valid-typed input; `MustNew` panics with that error. It returns an error if and only if some creation argument is invalid (for the reason of FR-016 here). | n/a (errors-and-results.md FR-023 to FR-026) | unhappy (invalid input) |
| FR-085 | `character.Event` MUST be an interface with an unexported marker method and a `CharacterID() CharacterID` method, there is no kernel `DomainEvent` (decisions FR-001, FR-019); every spec of this context lives in package `character`, so class and feature specs implement it without an exported hook. The seven events of this spec MUST be members of the closed sub-family `CharacterRecordEvent` (an interface embedding `Event`, with its own unexported marker and every member listed in its doc comment) so consumers switch exhaustively over them; class and feature events implement `Event` directly, outside that switch. Every `Event` MUST carry the `CharacterID` it concerns. | n/a (architecture §4) | happy |

### 4.8 Use cases

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-061 | If any looked-up entry is not found, `LevelUpCharacter` MUST return `LevelUpCharacterContentMissing` and MUST NOT return an `error` or panic. It MUST perform every lookup before returning, so the result names every missing entry (FR-077). `CreateCharacterContentMissing` is owned by `character-creation.md` FR-006. | n/a (architecture §4, §6; decisions FR-017, FR-018) | unhappy (missing content) |
| FR-062 | A `srderr.PortError` from the content port MUST propagate out of every use case that looks up content, unwrapped, unswallowed and unlogged; it is recorded once per FR-071 and rethrown unchanged. | n/a (architecture §4; observability §7) | unhappy (collaborator failure) |
| FR-063 | `LevelUpCharacter` with the rolled choice MUST roll exactly one Hit Die of the class's size through the `dice.Roller` port by calling `dice.Roll` (`dice-and-rolls.md` FR-050 to FR-053) for the one-die `dice.Expression` of the class die, and use the value as in FR-033. A roller returning a result for a different expression makes `dice.Roll` return a `*srderr.PortError` wrapping `srderr.ErrContractViolation`, which `LevelUpCharacter` returns as its `error` and records once as in FR-078. | SRD 5.2.1, "Gaining a Level" step 2, p. 23 | happy |
| FR-064 | `LevelUpCharacter` with the fixed choice MUST NOT call the dice roller. | SRD 5.2.1, "Gaining a Level" step 2, p. 23 | happy |
| FR-065 | With a fixed roller returning 1 and returning `die.faces`, `LevelUpCharacter` MUST give the clamped minimum and the maximum gain respectively. | SRD 5.2.1, "Gaining a Level" step 2, p. 23 | unhappy (determinism) |
| FR-066 | `LevelUpCharacter` MUST look up the chosen class always, and MUST load the `AbilityRequirement` of each current class through the content-source port only when a new class is being added. | SRD 5.2.1, "Multiclassing" ("Prerequisites"), p. 24 | happy |
| FR-067 | `LevelUpCharacter` MUST return `LevelUpCharacterRejected` carrying the `LevelUpRejection` of FR-027, FR-030 or FR-031, and `LevelUpCharacterContentMissing` for any class (chosen or current) the source lacks. All lookups (FR-066) complete before any rule is evaluated, so content-missing wins over every rejection (a level 20 character whose chosen class is missing gets `LevelUpCharacterContentMissing`, not `LevelUpMaximumLevel`). | SRD 5.2.1, "Multiclassing", p. 24 | unhappy (state error) |
| FR-068 | `AwardExperience` MUST return only `AwardExperienceSucceeded`. `ChangeAbilityScore` MUST return `ChangeAbilityScoreSucceeded` (including a no-op, FR-050) or `ChangeAbilityScoreRejected` with the `AbilityScoreChangeRejection`. Both need no ports other than observability. | n/a (architecture §4) | happy |
| FR-069 | Every use case MUST behave identically with `telemetry.Noop` and with a recording bundle (`telemetrytest`). | n/a (observability §9) | unhappy (determinism) |
| FR-070 | Use cases MUST emit the signals in section 7, once per call, using the defined constants and attribute keys. No use case logs. | n/a (observability §5, §6, §9) | happy |
| FR-071 | A `LevelUpCharacterContentMissing` result MUST be recorded exactly once (`CreateCharacterContentMissing` likewise, `character-creation.md` FR-098): `srd5e.outcome` `failure`, `srd5e.character.result` `content_missing`, span error status, one increment of the operations counter, no `error.type` (errors-and-results.md FR-043) and no log. A propagated `srderr.PortError` MUST be recorded exactly once with `srd5e.outcome` `failure`, `error.type` equal to its `ErrorType` (FR-088), no `srd5e.character.result`, span error status, one counter increment and no log. There is no separate errors counter: the operations counter with `srd5e.outcome=failure` and `error.type` already gives errors by type and operation. | n/a (observability §6, §7; errors-and-results.md FR-034, FR-043) | unhappy (telemetry on failure) |
| FR-072 | A rule rejection (`LevelUpRejection` or `AbilityScoreChangeRejection`; `CreateCharacterRejected` likewise, `character-creation.md` FR-099) MUST be recorded as a normal rule outcome: `srd5e.outcome` `failure`, `srd5e.character.result` `rejected`, the `srd5e.character.rejection` attribute on counter and span, span status not error, no `WARNING` or `SEVERE` log. A successful no-op (FR-050) is `srd5e.outcome` `success` with result `changed`. | n/a (observability §5, §6, §7; errors-and-results.md FR-043) | unhappy (telemetry on failure) |
| FR-076 | When a current class is not found during a new-class check (FR-066), `LevelUpCharacter` MUST return `LevelUpCharacterContentMissing` listing it as a `MissingContent` and MUST NOT evaluate the prerequisites. | n/a (architecture §4; testing.md §4) | unhappy (missing content) |
| FR-077 | `LevelUpCharacterContentMissing` MUST hold a non-empty `[]MissingContent` (copied on read) (kind and id; decisions FR-018; `content-schema-and-decoder.md` FR-147), ordered by content kind declaration order then id ordinal comparison (`content-source-port.md` FR-011), without duplicates, with no display text. The constructor MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) for an empty list. | shared-types-and-ownership.md FR-017, FR-018; errors-and-results.md FR-020, FR-021, FR-023, FR-024 | unhappy (missing content) |
| FR-078 | A roller error during `LevelUpCharacter` MUST be returned as the use case's `error`, a `*srderr.PortError` wrapped once (`errors.Is` still reaches the roller's cause), and be recorded once per FR-071 with `error.type` `dice_roller_error` (`srderr.ErrorType`; FR-088; dice-and-rolls.md FR-086). | n/a (architecture §4; errors-and-results.md FR-034) | unhappy (collaborator failure) |
| FR-079 | When the roller returns a value below 1 or above `die.faces` for a one-die roll, `LevelUpCharacter` MUST return a `*srderr.PortError` wrapping `srderr.ErrContractViolation` (a port contract violation, never a validation error or a panic), recorded once as in FR-078, and MUST NOT build a character. | errors-and-results.md FR-036; testing.md §4 | unhappy (collaborator failure) |
| FR-087 | No use case MAY alter its command or the `Character` in it; after any call the command's character and fields equal their prior values. | n/a (architecture §3, §4) | unhappy (immutability) |
| FR-088 | The `error.type` attribute MUST be set only for a returned `*srderr.PortError`, then equal to `srderr.ErrorType(err)`: `content_error` or `dice_roller_error` (decisions FR-041; dice-and-rolls.md FR-086; content-source-port.md FR-038). A returned port failure is recorded with `srd5e.outcome=failure`, `error.type` and no `srd5e.character.result`; a returned failure carries no `error.type`. The `srd5e.character.result` values here are `created`, `changed`, `rejected`, `content_missing`; other `character` specs add theirs in their own section 7 (`character-creation.md` FR-099 adds `rolled` and `duplicate`); the full set is `doc/observability/signals.md`. `srd5e.character.rejection` MUST use the snake_case value mapped in section 7. `unexpected` is not a value: a telemetry-backend panic propagates unrecorded by a counter (observability §7; errors-and-results.md FR-051). | shared-types-and-ownership.md FR-041; errors-and-results.md FR-005, FR-034, FR-043; observability §7 | unhappy (telemetry on failure) |

### 4.9 Heroic Inspiration, feature states and stored Hit Points

Applies `shared-types-and-ownership.md` FR-008, FR-010, FR-017, FR-022 to FR-024, FR-044, FR-048.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-089 | `HeroicInspiration` MUST be a typed enum with exactly the values `Inspired` and `NotInspired` (zero value invalid), and `Character.HeroicInspiration` MUST expose it, so a character never holds more than one instance. `character.New` MUST default to `NotInspired` and MAY take `Inspired`. | SRD 5.2.1, "Heroic Inspiration", p. 8 ("Only One at a Time"); Rules Glossary, p. 183; decisions FR-010 | happy |
| FR-090 | `GrantHeroicInspiration()` is the primitive transition on the slot. On `NotInspired` it MUST return a `CharacterChanged` with `Inspired` and one `HeroicInspirationGained`. On `Inspired` it MUST return a `CharacterChanged` holding an equal character and no events. The fate of the excess instance and its host-boundary operation are `heroic-inspiration.md` (FR-003 to FR-006; decisions FR-048). | SRD 5.2.1, Rules Glossary, "Heroic Inspiration", p. 183 ("If you gain Heroic Inspiration but already have it, it's lost unless you give it to a player character who lacks it"); p. 8 | unhappy (boundary) |
| FR-091 | `ExpendHeroicInspiration()` is the primitive transition on the slot. On `Inspired` it MUST return a `CharacterChanged` with `NotInspired` and one `HeroicInspirationExpended`. On `NotInspired` it MUST return `CharacterRejected` with `HeroicInspirationNotAvailable` (closed `HeroicInspirationRejection`), no events and an unchanged character. The reroll and use cases are `heroic-inspiration.md` (FR-012 to FR-014, FR-034); applying a face to a D20 Test is `d20-tests.md` FR-059. | SRD 5.2.1, "Heroic Inspiration", p. 8 ("you can expend it to reroll any die immediately after rolling it"); decisions FR-002, FR-010 | unhappy (state error) |
| FR-092 | `Character.FeatureStates` MUST return an immutable `FeatureStates`, empty by default. `FeatureStates.Read` with a `FeatureStateKey` MUST return the stored `FeatureState` and `true`, or the zero value and `false` when none (typed by the key, no `any` in the public signature); `Character.WithFeatureState(key, state)` MUST return a new `Character` holding that state under the key and every other entry unchanged, with no events. The aggregate MUST NOT interpret any `FeatureState`. | decisions FR-022 | happy |
| FR-093 | A `FeatureStateKey<T>` MUST have a `name` matching `[a-z][a-z0-9_]*` (the constructor MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) otherwise) and two keys with the same name MUST be equal. Reading or writing a name with a key whose `T` differs from the stored state's type MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`). `FeatureStates` and `FeatureState` implementations MUST compare by value. | decisions FR-007, FR-022 | unhappy (invalid input) |
| FR-094 | `ClassCreationChoices` MUST be an interface with two pure methods: one taking a `FeatureStates` and returning the states seeded for the class, and one returning the `content.Key`s the choices name (a copy, empty for `NoClassCreationChoices`), which `CreateCharacter` looks up with the other content (`character-creation.md` FR-103). `NoClassCreationChoices` returns the states unchanged, names no keys and is the command default. `CreateCharacter` (`character-creation.md` FR-102) MUST pass the seeded states to `character.New`. The aggregate stores no class-specific field and depends on no class spec. | decisions FR-016, FR-022, FR-024 | happy |
| FR-095 | `Character.ClassLevels` MUST be the stored `ClassLevels`, ordered, initial class first. `Character.InitialClass` MUST return the first entry's `ClassID`. A level-up in a class not yet held MUST append it last; in a held class MUST keep every position; both built as in FR-025 and FR-026. | SRD 5.2.1, "Multiclassing" ("Proficiencies"), p. 25; decisions FR-008 | happy |
| FR-096 | `Character` MUST store its `HitPointMaximum` and `HitDicePool`. `create` sets them (FR-012, FR-013); `LevelUp` adds the gain (FR-036) and one die (FR-037); a Constitution modifier rise adds (change times total level) (FR-045); no other transition changes them and nothing recomputes them from history or ability scores. | SRD 5.2.1, "Gaining a Level" steps 2 and 5, p. 23; decisions FR-008 | happy |
| FR-097 | `AwardExperience`, `LevelUp` and `ChangeAbilityScore` MUST leave `HeroicInspiration` and the `FeatureStates` of the input character unchanged in the result. | n/a (architecture §3; decisions FR-022, FR-023) | unhappy (rule interaction) |
| FR-098 | Each use case MUST return its own closed `<UseCase>Result` with operation-prefixed variants (section 6). No exported name in this context MAY be generic (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`), duplicate another context's export, or equal `AbilityScoreRejection`; the ability-change rejection is `AbilityScoreChangeRejection`. | decisions FR-001, FR-017, FR-021 | unhappy (name collision) |
| FR-099 | No use case of this spec MAY call another use case (decisions FR-044). `LevelUpCharacter`, `AwardExperience` and `ChangeAbilityScore` call the `Character` transitions and, for `LevelUpCharacter`, the `content.Source` and `dice.Roller` ports; each records only its own signals. The use cases of `character-creation.md` and `heroic-inspiration.md` likewise call transitions, not these. | decisions FR-044; architecture.md §4 | unhappy (nested telemetry) |
| FR-100 | The problems of `character.New` MUST be values of the closed `CharacterCreationProblem` family (an interface with an unexported marker method, every variant listed in its doc comment, each implementing `error`). This spec defines `StartingScoreAboveMaximum{Abilities}`: a non-empty slice of abilities whose starting score is above 20, in `Ability` declaration order, without duplicates. The joined error MUST hold each variant at most once, in the family's declaration order; constructing a `StartingScoreAboveMaximum` with an empty list is a validation error (and `MustStartingScoreAboveMaximum` panics). Further variants may be added as new requirements. | errors-and-results.md FR-023, FR-024; SRD 5.2.1, "Ability Scores", p. 5 | unhappy (invalid input) |
| FR-101 | The scalar constructors MUST return an error with a single typed reason and no display text (errors-and-results.md FR-035), each `errors.Is` `srderr.ErrInvalidInput`: `NewCharacterID` returns a `*CharacterIDError{Reason}` with enum `CharacterIDReason` (`Empty`, covering whitespace-only); `NewExperiencePoints` returns a `*ExperiencePointsError{Reason}` with `ExperiencePointsReason` (`Negative`); `NewExperienceAmount` returns a `*ExperienceAmountError{Reason}` with `ExperienceAmountReason` (`Zero`, `Negative`). None MAY panic. Each is exported once (FR-098). | errors-and-results.md FR-025, FR-026, FR-035; decisions FR-017, FR-021 | unhappy (invalid input) |
| FR-102 | When the chosen class is not held, `LevelUp` MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) if `RequirementsOfCurrentClasses` lacks an entry for any current class. This check runs before any rejection is evaluated, so it wins over every `LevelUpRejection` (FR-043). When the chosen class is held the map is not read and entries for classes not held are ignored. `LevelUpCharacter` always supplies every current class for a new class (FR-066). | n/a (architecture §6) | unhappy (invalid input) |
| FR-103 | The histogram `srd5e.character.duration` MUST carry the single attribute `srd5e.character.operation`: attributes passed to the `Meter` timing operation are fixed before the timed body runs (observability-ports.md FR-029, FR-031), so `srd5e.outcome` and `srd5e.character.result` are on the counter and span. A spec that adds a `character` use case MAY add values of `srd5e.character.operation` and MUST NOT add attributes to this histogram (D-21). | observability.md §5, §7; observability-ports.md FR-029, FR-031 | happy |
| FR-118 | This spec owns the class-feature signals; a class spec lists them as existing and defines none (decisions FR-042, FR-063). Counter `srd5e.character.feature_used` (`{use}`): `srd5e.class`, `srd5e.feature`, `srd5e.outcome` (`success` or `failure`), `srd5e.character.rejection` (only on a rejection or content-missing failure); `srd5e.outcome` and `srd5e.character.rejection` are extra counter attributes beyond decisions FR-042 (the values follow decisions FR-041). Histogram `srd5e.character.feature_used.duration` (`s`): `srd5e.class` and `srd5e.feature` only, recorded through the injected clock and `telemetry.Meter` (`telemetry.Timer`), never by reading the clock directly. Counter `srd5e.character.feature_used.errors` (`{error}`): `srd5e.class`, `srd5e.feature`, `error.type`; it counts only a returned `*srderr.PortError`. A returned port failure increments `feature_used` once with `srd5e.outcome=failure`, no `error.type` and no rejection, and `.errors` once; a returned failure carries no `error.type` and does not touch `.errors` (decisions FR-041; errors-and-results.md FR-043). A class spec adds values of `srd5e.feature` and `srd5e.character.rejection`, never an attribute or a per-feature metric name. | observability.md §5, §7; decisions FR-041, FR-042, FR-063 | happy |
| FR-119 | Emission of the signals of FR-103 and FR-118 belongs to the use case (decisions FR-023, FR-063): the outermost use case records each once; pure feature functions and domain transitions emit nothing. A class feature use case MUST use the FR-118 signals and MUST NOT write to `srd5e.character.duration`. | n/a (architecture §4; observability.md §3); decisions FR-023, FR-063 | unhappy (duplicate signal) |
| FR-120 | Every character event defined by any spec, `StrokeOfLuckUsed` included, MUST implement `Event` (FR-085) and carry `CharacterID`; none joins the closed `CharacterRecordEvent`. A spec whose event has no character id (an exploration event, for example) implements the `Event` interface of its own context. | n/a (architecture §4); decisions FR-019, FR-063 | unhappy (event base) |

### 4.10 `CharacterRuleOptions`

Moved here from `rule-options.md` (decisions FR-057); follows its pattern and shared rules (`rule-options.md` FR-023,
FR-029, FR-033, FR-034, FR-038). Every option is one the SRD offers the table: "Generate Your Scores" (p. 21), "Starting at
Higher Levels" and the "Bonus Feats at Level 20" sidebar (p. 24). Consumers: `character-creation.md` FR-054 and FR-071;
`BonusFeatsAtLevel20` has no consuming FR yet (`feats-behaviour.md`).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-104 | `CharacterRuleOptions.standard` MUST permit all three ability score generation methods (`StandardArray`, `RandomGeneration`, `PointCost`), have a starting level of 1 and use `NoBonusFeats`. | SRD 5.2.1, "Generate Your Scores", p. 21; "Starting at Higher Levels", "Bonus Feats at Level 20", p. 24 | happy |
| FR-105 | `CharacterRuleOptions` MUST accept any non-empty subset of the three methods `StandardArray`, `RandomGeneration`, `PointCost` (7 subsets) and report exactly that set, and MUST reject an empty set, because a character then cannot be created. | SRD 5.2.1, "Generate Your Scores", p. 21 | happy / unhappy (invalid input, empty) |
| FR-106 | `StartingLevel` MUST accept every whole number from 1 to 20 and MUST reject any value below 1 (0, -1, and -9007199254740991, exact on both VM and web) and any above 20 (21, and 9007199254740991). | SRD 5.2.1, "Starting at Higher Levels", p. 24; "Character Advancement", p. 23 | happy / unhappy (boundary, invalid input) |
| FR-107 | The range check of `StartingLevel` MUST be delegated to `CharacterLevel` (`derived-stats.md` FR-057; decisions FR-008) and not repeated as literals. `NewCharacterRuleOptions` MUST validate through the `CharacterLevel` constructor, converting its error into a `CharacterRuleOptionsProblem` carrying this parameter (`StartingLevel`) and the offending value, and mapping `BelowMinimum` to `CharacterRuleOptionsProblemKind` `BelowMinimum` and `AboveMaximum` to `AboveMaximum`. `StartingLevel` stays an `int`. | decisions FR-008; SRD 5.2.1, "Character Advancement", p. 23 | happy / unhappy (boundary) |
| FR-108 | `NewCharacterRuleOptions` MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`), carrying the offending parameter and value (a `CharacterRuleOptionsProblem`), for each violation of FR-105 and FR-106. `MustCharacterRuleOptions` checks methods, then level, and panics with the first violation only, so input violating both reports `AbilityScoreMethods`; `NewCharacterRuleOptions` itself reports every problem (FR-109). | architecture.md §6; errors-and-results.md FR-027 | unhappy (invalid input) |
| FR-109 | `NewCharacterRuleOptions` MUST NOT panic for any input of its parameter types (including an empty set and extreme `int` values). It MUST return the options and a nil error for valid input, and otherwise the `errors.Join` of every problem, in the order methods then starting level. Each problem is a `CharacterRuleOptionsProblem` with a `CharacterRuleOptionsParameter` (enum: `AbilityScoreMethods`, `StartingLevel`) and a `CharacterRuleOptionsProblemKind` (enum: `empty`, `BelowMinimum`, `AboveMaximum`): an empty set is (`AbilityScoreMethods`, `empty`); a level below 1 is (`StartingLevel`, `BelowMinimum`); above 20 is (`StartingLevel`, `AboveMaximum`). | architecture.md §5, §6; errors-and-results.md FR-023, FR-024, FR-025 | happy / unhappy (invalid input, several problems) |
| FR-110 | For the same input, `MustCharacterRuleOptions` MUST panic if and only if `NewCharacterRuleOptions` returns an error, and the panic value is that error. | errors-and-results.md FR-026 | unhappy (rule conflict) |
| FR-111 | The permitted-methods set MUST be a copy on construction and on read, unchanged when the set passed in is later mutated (`rule-options.md` FR-033). | architecture.md §4; errors-and-results.md FR-021; testing.md §4 | unhappy (immutability) |
| FR-112 | `CharacterRuleOptions` MUST have an `Equal` method comparing by value (permitted methods regardless of insertion order, `StartingLevel`, `BonusFeatsAtLevel20`; the type holds a set, so `==` is not offered). `WithPermittedMethods`, `WithStartingLevel` and `WithBonusFeatsAtLevel20` MUST each return a new instance (and an error where the constructor would) changing only the supplied field, leave the original unchanged, and apply the constructor's validation. | architecture.md §3 | happy / unhappy (rule interaction, invalid input) |
| FR-113 | `NoBonusFeats.FeatsEarned(xp)` MUST return 0 for every `ExperiencePoints` total, including 355,000 and far above it. | SRD 5.2.1, "Bonus Feats at Level 20", p. 24 | happy / unhappy (boundary) |
| FR-114 | `BonusFeatsFromXp.FeatsEarned(xp)` MUST return 0 for XP at or below the level 20 minimum, otherwise the number of complete 30,000 XP steps above it, rounded down (`(xp - threshold) ~/ 30000`). The threshold (355,000) MUST come from `ExperiencePoints.MinimumForLevel` (FR-004) for `CharacterLevel` 20, never a literal; only the step is held, as the constant `xpPerFeat = 30000` (fixed by the SRD, not a parameter). A negative XP cannot occur (FR-008). | SRD 5.2.1, "Bonus Feats at Level 20", p. 24; "Round Down", p. 5 | happy / unhappy (boundary) |
| FR-115 | A joined error from `NewCharacterRuleOptions` MUST NOT be returned with an empty problem list (an error implies at least one problem; errors-and-results.md FR-024); the problem values follow `rule-options.md` FR-033 (immutable, by value, no display text). | errors-and-results.md FR-019, FR-020, FR-021, FR-024 | unhappy (empty and minimal, immutability) |
| FR-116 | `CharacterRuleOptions` MUST have a validating constructor and `StandardCharacterRuleOptions()` MUST return the standard value without package-level state (no mutable global); the option types satisfy `rule-options.md` FR-023. | architecture.md §3, §6, §10 | happy / unhappy (immutability, state) |
| FR-117 | The problem type returned by `NewCharacterRuleOptions` MUST be the typed `CharacterRuleOptionsProblem` (no generic name; `rule-options.md` FR-033), joined with `errors.Join`; the "path" of errors-and-results.md FR-023 is the `CharacterRuleOptionsParameter`. | decisions FR-017; errors-and-results.md FR-018, FR-023, FR-024, FR-041 | unhappy (name collision) |

`BonusFeatsFromXp.FeatsEarned` cases (FR-114; threshold 355,000; SRD 5.2.1, "Bonus Feats at Level 20", p. 24, "Round Down",
p. 5), one table-driven test per row; `NoBonusFeats` (FR-113): XP 0, 355,000, 385,000 and 1,000,000 each give 0.

| Label | XP | Feats | Source | Path |
|---|---|---|---|---|
| minimal | 0 | 0 | p. 24 | happy |
| exactly the level 20 threshold | 355,000 | 0 | p. 24 | boundary |
| one above the threshold | 355,001 | 0 | p. 5 (round down) | boundary |
| one short of the first feat | 384,999 | 0 | p. 5 | boundary |
| first feat | 385,000 | 1 | p. 24 | boundary |
| one short of the second | 414,999 | 1 | p. 5 | boundary |
| second feat | 415,000 | 2 | p. 24 | happy |
| a large total rounds down | 1,000,000 | 21 | p. 5 | unhappy (large) |

### 4.11 Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-005 | Withdrawn: moved to derived-stats.md (FR-057) |
| FR-009 | Withdrawn: moved to derived-stats.md (FR-059) |
| FR-017 | Withdrawn: moved to ability-scores.md (FR-018, FR-019) |
| FR-019 | Withdrawn: moved to character-aggregate.md (FR-084) |
| FR-060 | Withdrawn: moved to character-creation.md (FR-001) |
| FR-080 | Withdrawn: moved to character-aggregate.md (FR-096) |
| FR-083 | Withdrawn: moved to character-aggregate.md (FR-096) |
| FR-086 | Withdrawn: moved to content-classes.md (FR-011) |

## 5. Model

**Aggregate root: `Character`.** Outside code holds only the root; all parts are immutable value objects reachable only
through it. It holds `CharacterID`, `ExperiencePoints`, `ClassLevels`, `ability.Scores`, `HitPointMaximum`, `HitDicePool`,
`HeroicInspiration`, `FeatureStates`, `SpeciesID` and `BackgroundID`; derived accessors are `InitialClass`, `TotalLevel`,
`ProficiencyBonus` and `IsEligibleForNextLevel`.

`AdvancementClass` is owned here: a value object `AdvancementClass(ClassID, HitPointDie, AbilityRequirement)`, built by
`LevelUpCharacter` from the looked-up `ClassDefinition` (`content-classes.md`); `LevelUp` is its only consumer (FR-025,
FR-026, FR-031, FR-035). `AbilityScoreGenerationMethod` is an enum (`StandardArray`, `RandomGeneration`, `PointCost`) owned
here (decisions FR-057), not by `character-creation.md`, which uses its members. `CharacterRuleOptions` is an option value,
not aggregate state (4.10).

Owned elsewhere and reused as is: `CharacterLevel`, `ClassLevels`, `ProficiencyBonus`, `HitPointMaximum`, `HitDicePool`,
`HitPointGain` (`derived-stats.md`); `Ability`, `ability.Score`, `ability.Modifier`, `ability.Scores`, `AbilityScoreRejection`
(`ability-scores.md`); `ClassID`, `SpeciesID`, `BackgroundID`, `HitPointDie` (`content-schema-and-decoder.md` FR-114,
FR-132); `MissingContent` (`content-schema-and-decoder.md` FR-147; decisions FR-018); `ClassDefinition`, `AbilityRequirement`
(`content-classes.md`); (no kernel event type exists). `LevelUp` builds `ClassLevels` and `HitDicePool` through their
constructors (FR-025, FR-026, FR-037).

**Transitions** are pure, deterministic domain operations on the root:

| Operation | Result |
|---|---|
| `character.New(...)` | `(NewCharacter, error)`; the error joins every problem on invalid arguments (FR-016, FR-082) |
| `AwardExperience(ExperienceAmount)` | `CharacterChanged` |
| `LevelUp(AdvancementClass, HitPointGain, {requirementsOfCurrentClasses})` | `CharacterTransition<LevelUpRejection>` |
| `ChangeAbilityScore(Ability, ability.Score, {maximumScore = 20})` | `CharacterTransition<AbilityScoreChangeRejection>` |
| `GrantHeroicInspiration()` / `ExpendHeroicInspiration()` | `CharacterChanged` (FR-090) / `CharacterTransition<HeroicInspirationRejection>` (FR-091) |
| `WithFeatureState(FeatureStateKey<T>, T)` | `Character` (FR-092) |

`CharacterTransition<R>` is closed: `CharacterChanged<R>` (new `Character`, an event slice copied on read) or
`CharacterRejected<R>` (a closed `R`, no text).

**Invariants and where enforced:**

| Invariant | Enforced by | Failure |
|---|---|---|
| Id non-empty; XP non-negative; award positive | `CharacterID`, `ExperiencePoints`, `ExperienceAmount` constructors | validation error (FR-101) |
| Starting scores 1-20 | `character.New` | joined validation error (FR-100) |
| `ClassLevels` non-empty, total level 1-20; pool size equals total level; stored maximum at least total level | `NewClassLevels`, `character.New` (FR-051 to FR-053, FR-084) | validation error |
| Level-up legality; score cap and Con decrease; no Heroic Inspiration to expend | `LevelUp`; `ChangeAbilityScore`; `ExpendHeroicInspiration` | `CharacterRejected<...Rejection>` |
| Cap from 20 to 30; feature key name and type | `ChangeAbilityScore`; `NewFeatureStateKey`, `FeatureStates` (FR-093) | validation error |

Expected rule outcomes are `CharacterRejected` results; invalid untrusted arguments are validation errors from the `NewX`
constructors, and a broken invariant (programmer error) is a panic in `MustX` only
(architecture §6). The domain does no I/O, so it has no `srderr.PortError`s; a dice roller breaking its contract is caught by
the use case (FR-079). Other contexts are referenced by typed id; no `combat` or `d20` type appears (decisions FR-013);
`d20` has its own `RerollDecision` (`d20-tests.md` FR-059), composed in `heroic-inspiration.md` (FR-034). `toolbox`
(decisions FR-045) is downstream.

## 6. Interface

The use cases are types with a single `Run(ctx, cmd)` method (`ctx` first because they emit spans and `LevelUpCharacter` calls the content port; go.md §4) and an immutable command. They take `telemetry.Telemetry` through a `WithTelemetry` option, default the zero-value `telemetry.Noop` (decisions FR-036), and return their own closed `<UseCase>Result` plus an `error` that is non-nil only for a port failure or a cancelled `ctx` (decisions FR-017).

| Use case | Command | Result (closed) | Calls |
|---|---|---|---|
| `LevelUpCharacter` | `LevelUpCharacterCommand`: `Character`, `ClassID`, `HitPointChoice` (closed: `RollHitDie`, `TakeFixedValue`) | `LevelUpCharacterResult`: `LevelUpCharacterSucceeded` (character, events), `LevelUpCharacterRejected` (`LevelUpRejection`), `LevelUpCharacterContentMissing` (`[]MissingContent`) | content-source port and `dice.Roller`; `ctx` first |
| `AwardExperience` | `AwardExperienceCommand`: `Character`, `ExperienceAmount` | `AwardExperienceResult`: `AwardExperienceSucceeded` (character, events) | pure; `ctx` first (emits a span) |
| `ChangeAbilityScore` | `ChangeAbilityScoreCommand`: `Character`, `Ability`, new `ability.Score`, optional cap (FR-074) | `ChangeAbilityScoreResult`: `ChangeAbilityScoreSucceeded` (character, events; a no-op included), `ChangeAbilityScoreRejected` (carries the `AbilityScoreChangeRejection`) | pure; `ctx` first (emits a span) |

Every domain variant maps to exactly one result (`CharacterChanged` to the `...Succeeded` variant, `CharacterRejected` to
the `...Rejected` variant); `LevelUpCharacterContentMissing` has no domain counterpart. `CreateCharacter` is
`character-creation.md` and the Heroic Inspiration use cases are `heroic-inspiration.md` (D-15); feature-state changes are
domain transitions called by class specs. No use case calls another (FR-099).

`RollHitDie` rolls (FR-063) and builds a `RolledHitPointGain(value, die)`; `TakeFixedValue` builds
`FixedHitPointGain(die.faces / 2 + 1)` (FR-064). The domain `LevelUp` takes only the `HitPointGain`. The use case looks up
the class through `content.Source.Lookup(ctx, key)` (`content-source-port.md`); the found `ClassDefinition` supplies the die (FR-018)
and the `AbilityRequirement` (D-7), packed into the `AdvancementClass`. Ports: `content.Source` (takes `ctx`; a not-found
outcome becomes a `MissingContent`, a `*srderr.PortError` is returned, FR-062), `dice.Roller` (synchronous, no `ctx`;
`RollHitDie` only; FR-063, FR-078, FR-079) and `telemetry.Telemetry`. Consumer-side interfaces: `LevelUpCharacter` declares its own
small `ContentSource` interface for what it calls and is the owning spec of `character.Roller` (file `ports.go`: the one-method
`Roll(dice.Expression) (dice.Result, error)` that any `dice.Roller` satisfies; declared once in package `character`, shared-types-and-ownership.md FR-021; the other specs of the package say "uses `character.Roller`"). Tests use `contenttest.FakeSource` and `dicetest` doubles
(decisions FR-030, FR-031, FR-038).

**Exports.** Package `character` exports every type owned here (section 5), the events with `Event`
and `CharacterRecordEvent`, the rejection families, the three use cases with their commands, result types and variants, and
`HitPointChoice` with its variants. Because `character-creation.md` FR-054 and FR-071 read them, it also exports
`CharacterRuleOptions`, `AbilityScoreGenerationMethod`, `BonusFeatsAtLevel20` and variants and the
`CharacterRuleOptionsProblem` type (the recorded exception of `rule-options.md` FR-029). The package doc comment lists each
command, result and event and states that an option is a value in the command, never a global setting (`rule-options.md`
FR-034). Raw domain constructors are exported (hosts decode stored characters with them). Every exported name is unique
(FR-098; package-scan test, decisions FR-017, FR-021); there is no facade or re-export package.

## 7. Observability

Domain code is **silent by design**: it returns events and results that carry the what and why. The use cases are
instrumented (`doc/rules/observability.md` §2) and log nothing: span and counter already record the outcome (observability
§7). A returned port failure is span error status plus `error.type` on the counter; a returned `...ContentMissing` is
`srd5e.outcome=failure`, `srd5e.character.result=content_missing`, span error status, no `error.type`; a rejection is
`failure`, `rejected` and `srd5e.character.rejection`, span status not error; a telemetry-backend panic propagates (FR-071, FR-072, FR-088).
Shared keys (`srd5e.outcome`, `error.type`) come from the `telemetry` constants, values per decisions FR-041. `created` and `changed`
(a no-op too) are `success`; `rejected` and `content_missing` are `failure`. Rejection values: `maximum_level`,
`insufficient_experience`, `multiclass_prerequisites_not_met` (the three `LevelUpRejection`s, in order),
`exceeds_maximum_score`, `constitution_modifier_decrease`. Heroic Inspiration signals are `heroic-inspiration.md` FR-036.
Names do not overlap `srd5e.character.derived_stats.*` or `srd5e.d20_test.*`.

| Signal | Type, unit | Attributes | Purpose | Emitted by | Suggested monitoring |
|---|---|---|---|---|---|
| `srd5e.character.create` | span | defined by `character-creation.md` section 7 | One character creation | `CreateCharacter` | Error status rate above 0 for sustained periods |
| `srd5e.character.level_up` | span | `srd5e.outcome`, `srd5e.character.result` (`changed`, `rejected`, `content_missing`), `srd5e.character.hit_point_choice` (`rolled`, `fixed`), `srd5e.character.rejection` (rejections only) | One level-up including content and dice calls | `LevelUpCharacter` | As above |
| `srd5e.character.award_experience` | span | `srd5e.outcome`, `srd5e.character.result` (`changed`) | One XP award | `AwardExperience` | None needed beyond the counter |
| `srd5e.character.change_ability_score` | span | `srd5e.outcome`, `srd5e.character.result` (`changed`, `rejected`), `srd5e.character.rejection` (rejections only) | One ability score change | `ChangeAbilityScore` | None needed beyond the counter |
| `srd5e.character.load_content` | span (child of every `character` use case that looks up content) | `srd5e.outcome` | Time in `content.Source` lookups | `CreateCharacter`, `LevelUpCharacter`, `TakeFeat`, `ChangeMagicInitiateSpell`, `ReplaceHighElfCantrip` and any other `character` use case that looks up content | Compare with the parent span; alert if lookup time dominates |
| `srd5e.character.operations` | counter, `{operation}` | `srd5e.character.operation` (`create`, `level_up`, `award_experience`, `change_ability_score`; siblings add `calculate_spellcasting`, `create_spellcasting`, `expend_spell_slot`, `regain_spell_slot`, `advance_spellcasting`, `take_feat`, `change_magic_initiate_spell`, `expend_species_trait_use`, `renew_species_trait_uses`, `use_relentless_endurance`, `roll_halfling_luck`, `roll_stones_endurance`, `replace_high_elf_cantrip`, `resolve_wish`, `roll_wish_stress_damage`), `srd5e.outcome`, `srd5e.character.result` (absent when an exception left; siblings add `taken`, `resolved`, `not_applicable`, each `success`), `srd5e.character.feat_category` and `srd5e.character.feat_source` (`take_feat` only, `feats-behaviour.md` section 7), `srd5e.character.hit_point_choice` (`level_up` only), `srd5e.character.rejection` (rejections only; siblings add values, e.g. the ten of `species-trait-behaviour.md` section 7), `srd5e.character.wish_effect` and `srd5e.character.wish_lockout` (`resolve_wish` only, `wish-spell.md` section 7; values there), `error.type` (returned `srderr.PortError` only) | Throughput, outcome mix, errors by `error.type` | the three use cases here and `CreateCharacter` | Alert on any `error.type` increment; watch the share of `rejected` for `level_up` and a rise in `content_missing` |
| `srd5e.character.duration` | histogram, `s` | `srd5e.character.operation` only (FR-103, D-21); its values are those of the counter plus the snake_case name of every sibling use case that has its own counter (the eight in `spellcasting.md`, the four in `casting-and-concentration.md`, `resolve_wish` and `roll_wish_stress_damage`) | Latency per use case via the `telemetry.Meter` histogram | as above | p99 regression against baseline per operation; content lookups dominate |
| `srd5e.character.feature_used` | counter, `{use}` | `srd5e.class`, `srd5e.feature`, `srd5e.outcome`, `srd5e.character.rejection` (FR-118) | Throughput and outcome mix per class feature | every class feature use case | Warn when the `failure` share for one feature stays high for 15 minutes |
| `srd5e.character.feature_used.duration` | histogram, `s` | `srd5e.class`, `srd5e.feature` only (FR-118) | Latency per class feature | every class feature use case | p99 regression against baseline per feature |
| `srd5e.character.feature_used.errors` | counter, `{error}` | `srd5e.class`, `srd5e.feature`, `error.type` (FR-118) | Port failures leaving a class feature use case | every class feature use case | Alert on any increase: a failing `dice.Roller` or `content.Source` |

Attribute values are low-cardinality enumerations; no ids, names or XP totals (observability §5). Entries feed
`doc/observability/signals.md`.

## 8. Test plan

Doubles from the `*test` packages (decisions FR-038; `_test.go` files only): `dicetest.FixedRoller` (including values 0 and
size + 1) and `dicetest.CallbackRoller (returning an error)`, the `telemetrytest` recorder, `contenttest.FakeSource`. No new port, so no new
contract suite (decisions FR-039). Every live FR id appears below.

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002, FR-003, FR-073, FR-100, FR-101 | unit | `CharacterID` and the scalar constructors and their `MustX` forms agree (empty and whitespace id; XP -1; amount 0 and -1); `StartingScoreAboveMaximum` with an empty list returns a validation error; several abilities above 20 listed once each in declaration order |
| FR-004, FR-006, FR-007, FR-008, FR-010 | unit | all 20 levels through `MinimumForLevel`; `LevelFor` boundaries (0, 299, 300, 354,999, 355,000, 400,000); `ExperiencePoints` -1, 0, 1; `ProficiencyBonus` rows of FR-010 |
| FR-011, FR-012, FR-013, FR-014, FR-015, FR-016, FR-018, FR-082 | unit | `character.New` table-driven for FR-012; each `HitPointDie` gives base `faces` and fixed `faces` / 2 + 1 (no rejection test, an out-of-set die is not representable); create at XP for levels 2, 10 and 20, then `LevelUp` to N; score 20 and 21 per ability, `New` and `MustNew` agree |
| FR-020, FR-021, FR-022, FR-023, FR-024, FR-081 | unit | one award; class levels unchanged; award past 355,000; `ExperienceAmount` 0, -1, 1 with `MustExperienceAmount` agreeing; eligibility table |
| FR-025, FR-026, FR-027, FR-028, FR-029, FR-030, FR-031, FR-032, FR-043 | unit | existing class, new class, below threshold; level 6 Cleric / level 1 Fighter either class; exactly at and one below the threshold; total level 20; prerequisite and combined rejection tables (Fighter and deduplication rows included) |
| FR-033, FR-034, FR-035, FR-036, FR-037, FR-038, FR-039 | unit | gain table; gain-validation table including `FixedHitPointGain(100)`; maximum and pool after a level-up in an existing, a same-die and a different-die class; Fighter 5 / Paladin 5 holds ten d10s (pooling itself: `derived-stats.md` FR-061) |
| FR-040, FR-041, FR-042 | unit | event fields; Proficiency Bonus boundary cases; rejection returns no events and the input is equal afterwards |
| FR-044, FR-045, FR-046, FR-047, FR-048, FR-049, FR-050, FR-074, FR-075 | unit | Constitution and cap tables; one test that the command accepts `ability.NewScore` output (bounds tested in `ability-scores.md` FR-009, FR-010) |
| FR-051, FR-052, FR-053, FR-054, FR-055, FR-056, FR-057, FR-058, FR-059 | unit | `ClassLevels` of 0, 1, 20 and 21 total levels; pool size per total level and a pool of another size; a change to a returned slice has no effect on every exposed collection; value equality (class order matters; feature states and Heroic Inspiration included); determinism; event order; rejected transitions return values, invalid arguments return validation errors |
| FR-084, FR-096 | unit | stored maximum after create (d6 at Constitution 1 gives 1), each level-up and a Constitution rise; constructor rejects a maximum below the total level; level 20 with every roll 1 at Con 1 gives 20 |
| FR-085, FR-120 | architecture, unit | every event carries the id and is an `Event`; the seven switch exhaustively as `CharacterRecordEvent`; a type outside the package cannot implement the closed families (unexported marker methods) |
| FR-089, FR-090, FR-091, FR-092, FR-093, FR-094, FR-095, FR-097, FR-102 | unit | grant from NotInspired and from Inspired (no events); expend from both (rejection, input unchanged); test-only `FeatureState` and key: read absent, write, overwrite, invalid name, mismatched type; seeding at create (`NoClassCreationChoices` names no keys and returns the seeded states unchanged); class order; other transitions leave states unchanged; new class with a map missing a current class returns a validation error (also at level 20), existing class with an empty map succeeds |
| FR-061, FR-062, FR-066, FR-067, FR-076, FR-077 | use case | `contenttest.FakeSource` (no entries, missing chosen or current class, one failing source returning a `*srderr.PortError`); current classes looked up only for a new class; ordered, deduplicated key list; each `LevelUpRejection` reaches `LevelUpCharacterRejected`; a level 20 character with a missing chosen class gives `LevelUpCharacterContentMissing` |
| FR-063, FR-064, FR-065, FR-078, FR-079 | use case | fixed roller: call recorded for `RollHitDie`, none for `TakeFixedValue`; extremes 1 and die size; another expression, a failing roller, and 0 or size + 1 give a `*srderr.PortError` (the last two wrapping `srderr.ErrContractViolation`), returned, recorded once, no character built |
| FR-068, FR-069, FR-070, FR-071, FR-072, FR-087, FR-088, FR-103 | use case | result variants with no ports; command and `Character` equal before and after; recording `telemetrytest` bundle and `Meter`: span names, `srd5e.outcome`, `srd5e.character.result`, counter, no log; `error.type` only for a returned port failure; rejection strings; duration histogram carries `srd5e.character.operation` only |
| FR-118, FR-119 | use case | test-only feature use case: counter attributes for success, rejection, content-missing; a returned `*srderr.PortError` gives one `feature_used` increment without `error.type` and one `.errors` increment with it; feature duration histogram carries `srd5e.class` and `srd5e.feature` only; `srd5e.character.duration` untouched |
| FR-098, FR-099 | architecture | package API scan finds no generic or duplicate name and no collision with `ability.Reason` or `ability.InvalidError`; use cases mention no other use case |
| FR-104, FR-105, FR-106, FR-107, FR-108, FR-109, FR-110 | unit | `standard` equals an explicit instance; the 7 subsets of `StandardArray`, `RandomGeneration`, `PointCost` round-trip; empty set returns an error (also from `WithPermittedMethods`); levels 1, 2, 19, 20 accepted, 0, -1, -9007199254740991, 21, 9007199254740991 rejected, agreeing with `CharacterLevel`; the error carries parameter and value (`errors.As` the problem); empty set with level 0 gives both problems in order from `NewCharacterRuleOptions` (`MustCharacterRuleOptions` panics with the first); `NewCharacterRuleOptions` never panics; `MustCharacterRuleOptions` panics iff it errors |
| FR-111, FR-112, FR-113, FR-114 | unit | exposed set a copy and independent of the original; `Equal` independent of set order; `WithPermittedMethods` and `WithStartingLevel` each change one field and validate; one test per row of the 4.10 tables, and 355,000 equals `MinimumForLevel` for level 20 |
| FR-115, FR-116, FR-117 | unit, architecture | a joined error always holds at least one problem; equality, copied lists; no display text on problem types; `StandardCharacterRuleOptions()` equals an explicit instance and shares no mutable state; the `rule-options.md` FR-023 and `errors-and-results.md` FR-018, FR-041 scans cover the files |

- **Delegated:** FR-005, FR-009, FR-017, FR-060, FR-086 are tested in the specs of 4.11; FR-019, FR-080, FR-083 are replaced.
- **Scenario** (`scenarios` package, public API only): create, award 300 XP, level up in the same class (fixed), gain a level in a new class
  after raising a primary ability, change Constitution; level-up with too little XP (unchanged character); grant and expend
  Heroic Inspiration, expend again; level up in a class the source lacks. Cites FR-011, FR-020, FR-025, FR-026, FR-027,
  FR-045, FR-061, FR-090, FR-091.
- **Architecture:** no file of package `character` imports `combat` and the aggregate imports no `class-*` code
  (decisions FR-013, FR-016); signal constants appear in `doc/observability/signals.md`.
- **Mutation testing:** `gomu` on `./character`, threshold 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (`doc/rules/testing.md` §8; never as a bare host process).
  Emphasis: FR-024, FR-029, FR-031 (any-of branch), FR-033, FR-035, FR-084, FR-075, FR-045.

## 9. Open questions and decisions

No open questions. Settled ones (one line each; other specs cite these ids):

| Id | Settled | Basis |
|---|---|---|
| D-1 | The host supplies the `CharacterID`; an `IdGenerator` port may come later. | architecture §3 |
| D-2 | The minimum of 1 applies to the fixed value as well as the roll (FR-034). | `derived-stats.md` FR-062 |
| D-3 | A level 1 contribution below 1 cannot arise (closed `HitPointDie`); the only guard is FR-084. | FR-018, FR-084 |
| D-4 | Rejection order is fixed (FR-043, FR-075): the most fundamental problem first. | FR-043, FR-075 |
| D-5 | A Constitution modifier decrease is rejected (FR-049) until the SRD defines it. | FR-049 |
| D-6 | Class dice come from the class pages (`ClassDefinition.HitPointDie`: d12 Barbarian; d10 Fighter, Paladin, Ranger; d8 Bard, Cleric, Druid, Monk, Rogue, Warlock; d6 Sorcerer, Wizard), matching pp. 22-23; level 1 base and fixed value are derived (FR-018). | decisions FR-053 |
| D-7 | `AbilityRequirement` (`AllOf` / `AnyOf`) is owned by `content-classes.md` and consumed in FR-031. | decisions FR-025 |
| D-9 | Class state goes through `FeatureStates` (FR-092, FR-093) and `ClassCreationChoices` (FR-094); spell lists are `spellcasting.md`. | decisions FR-022, FR-024 |
| D-11 | `Character` stores the Hit Point maximum (FR-096); no minimum-of-1 interaction on a Constitution rise. | decisions FR-008 |
| D-14 | Setting an ability to its current score is a successful no-op, not a rejection (FR-050). | FR-050 |
| D-15 | Heroic Inspiration use cases and signals are `heroic-inspiration.md`; feature-state changes stay domain transitions, with a use case deferred until a consumer needs it. | decisions FR-048 |
| D-19 | A returned content-missing result carries no `error.type`. | decisions FR-041; errors-and-results.md FR-043 |
| D-20 | `HitPointDie` is defined by `content-schema-and-decoder.md` FR-132; `LevelUp` builds `ClassLevels` and `HitDicePool` through public constructors. | decisions FR-053 |
| D-21 | The duration histogram carries `srd5e.character.operation` only (FR-103); this spec owns all character and class-feature signals (FR-118, FR-119); siblings add values, never attributes. | decisions FR-063 |

## 10. Changelog impact

`CHANGELOG.md`, under `Unreleased` / **Added**: "`character`: `Character` aggregate with typed `CharacterID`,
`ExperiencePoints`, stored `ClassLevels`, `HitPointMaximum` and `HitDicePool`, `HeroicInspiration` and the `FeatureStates`
slot; creation (with `ClassCreationChoices`), Experience Points, level-up (multiclassing prerequisites, Hit Point gain),
ability score changes, the Heroic Inspiration state transitions, the seven domain events, the use cases `LevelUpCharacter`,
`AwardExperience` and `ChangeAbilityScore` returning `<UseCase>Result` with operation-prefixed variants, and telemetry
`srd5e.character.*`." Not **BREAKING** (first release of this context). Also **Added**, when `CharacterRuleOptions` is
exported (`rule-options.md` FR-029): "`character`: `CharacterRuleOptions` (ability score generation methods, starting level,
bonus feats at level 20)". Adding or removing a variant of the closed `BonusFeatsAtLevel20` later is breaking
(architecture.md §7).
