# Level advancement, multiclassing and subclass choice

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `character-aggregate.md`, `derived-stats.md`, `ability-scores.md`, `spellcasting.md`, `content-classes.md`, `content-source-port.md`, `content-schema-and-decoder.md`, `errors-and-results.md`, `observability-ports.md` |

## 1. Purpose and scope

The parts of "Level Advancement" and "Multiclassing" that no other spec owns, tied to the ones other specs own by
reference: a read-only **preview of gaining a level** (allowed or not, what it gives, pending choices, what a new
class grants), built on `character-aggregate.md`; the **ability score increase** of Ability Score Improvement, the
Epic Boon feats and Grappler; **choosing a subclass** (`shared-types-and-ownership.md` FR-020 places `SubclassChosen`
and `SubclassRejected` here); **multiclassing** (pp. 24-26: prerequisites cross-checked, grants of a first level in a
new class FR-108 to FR-112, Extra Attack FR-054 to FR-058, every other rule delegated, 4.5). Starting at Higher
Levels, its equipment table and Trinkets moved to `higher-level-start-equipment-and-trinkets.md` (4.10; size, FR-062
of the decisions).

This spec is the topic-specific owner (`shared-types-and-ownership.md` FR-048) of the advancement plan preview and
defines only the types marked "defined here" in section 3. `spellcasting.md` (rank 13) is a lower-rank peer; the edge
runs from this spec to it only.

**Ownership of the neighbours.** The Character Advancement table as data: `character-aggregate.md` FR-004, FR-006 to
FR-008 and `derived-stats.md` FR-059. `LevelUpCharacter`, Hit Point gain, the Hit Dice pool, `AwardExperience`,
`ChangeAbilityScore` and their rejections: `character-aggregate.md` FR-020 to FR-050, FR-061 to FR-088 (previewed and
composed here). Feat choices, Repeatable, prerequisites and `FeatTaken`: `feats-behaviour.md` (a level opens a grant,
FR-016; `TakeFeat` accepts it; the increase is applied here). Derived numbers: `derived-stats.md`; Spell Slots and
Pact Magic: `spellcasting.md`; class features as content: `content-classes.md`; class behaviour: the class specs.
Building a character at a higher starting level, which composes this spec's `SubclassSelection` and
`AbilityScoreIncreaser`: `higher-level-start-equipment-and-trinkets.md`.

**Out of scope:** **Bonus Feats at Level 20** (p. 24), a GM option owned by `character-aggregate.md` (FR-113, FR-114); a
level-up at total level 20 is rejected regardless of XP (`character-aggregate.md` FR-030); **Tiers of Play** (no
rule); ability score generation and `CreateCharacter`. Not in SRD 5.2.1, so not added: a rule for losing a level, a
cap on classes held, choosing a subclass again, a limit on Epic Boon feats, a partial increase past the cap (only Q-7
and Q-18 decide).

**Supersedes (`shared-types-and-ownership.md` FR-048).** Owned here instead of the class specs: the generic multiclass
grant text (FR-108 to FR-112; class-specific choices stay); subclass gate, second choice, rejection reasons and event
names (FR-041 to FR-047, FR-049, FR-017, FR-051; an unknown id is `SubclassRejected(UnknownSubclass)`, FR-046, Q-19);
Extra Attack across classes (FR-054 to FR-058). The prerequisite is `character-aggregate.md` FR-031, FR-032,
cross-checked by FR-052; `MaximumLevel` is `LevelUpMaximumLevel` (FR-024); `NotNextLevel` has no analogue (Q-17); the
level event is the aggregate's `LevelGained`; the Hit Point value, die and Constitution rise are the aggregate's
(`character-aggregate.md` FR-018, FR-033 to FR-039, FR-045, FR-046). The other specs re-point their own withdrawal
rows; this spec does not edit them.

## 2. Source

Printed page numbers equal PDF page numbers (checked: printed page 23 "Level Advancement" is PDF page 23).

- SRD 5.2.1, "Ability Scores", p. 5; Ability Modifiers table, p. 6; "Create Your
  Character" (level, XP, "Starting at a Higher Level"), Class Overview table, p. 19
- "Level Advancement" (Character Advancement table, "Gaining a Level", Fixed Hit
  Points by Class), p. 23; "Tiers of Play" pp. 23-24; "Bonus Feats at Level 20", p. 24
- "Multiclassing", "Prerequisites", pp. 24-25 (headings Experience Points ...
  Spellcasting, p. 25); Multiclass Spellcaster table, p. 26; "As a Multiclass
  Character": Barbarian p. 28, Fighter p. 47, Wizard p. 77
- "Parts of a Feat", "Ability Score Improvement", "Grappler", p. 87; "Epic Boon Feats", p. 88; "Level 3: Cleric Subclass", p. 37;
  "Level 3: Rogue Subclass", p. 62
- Package design: `doc/rules/architecture.md`, `doc/rules/observability.md`,
  `doc/rules/testing.md`, `shared-types-and-ownership.md`, `errors-and-results.md`.

## 3. Vocabulary

Used by reference: the types of `character-aggregate.md`, `derived-stats.md`, `content-classes.md` and
`content-schema-and-decoder.md` (typed ids, FR-114).

| SRD term | Type (defined here) | Notes |
|---|---|---|
| (preview of gaining a level) | `LevelAdvancementPreview`, closed: `LevelAdvancementGain`, `LevelAdvancementBlock`; pure `LevelAdvancementPreviewer` | Read-only; never changes a `Character`. A gain carries the chosen class's `HitPointDie` (FR-108) and the row's `HasSubclassFeature` (FR-015). |
| "gain a level in a new class" (p. 24) | `AdvancementClassStatus`, closed: `AdvancementClassStatusExisting(CurrentClassLevel)`, `AdvancementClassStatusNew(grants)` | `grants` is the owned `MulticlassGrants`. |
| (a choice a level leaves open) | `PendingAdvancementChoice`, closed: `PendingSubclassChoice(ClassID, ClassLevel)`, `PendingFeatGrant(ClassID, ClassLevel, FeatureID, category)` | For a repeated grant `FeatureID` is the older entry's id, so `ClassLevel` tells grants apart (FR-016). Only choices every class shares; a host builds the `feats-behaviour.md` `FeatGrant` from it. |
| Ability Score Improvement, Epic Boon, Grappler (increase) | `AbilityScoreIncrease` (`increases`: defensively copied `Ability` to positive int; `MaximumScore` 20 to 30; factories `AbilityScoreImprovementOneByTwo`, `...TwoByOne`, `EpicBoon`); `AbilityScoreIncreaser`; `AbilityScoreIncreaseRejection`: `IncreaseExceedsMaximum(abilities, MaximumScore)` | Distinct from `AbilityScoreRejection` (`ability-scores.md`) and `AbilityScoreChangeRejection` (aggregate). |
| Subclass held on the character | `ChosenSubclass` (a `FeatureState`), `SubclassStateKey.ForClass(ClassID)` (a `FeatureStateKey<ChosenSubclass>`); `SubclassSelection` pure service: `choose`, `IsPending` | Stored through `Character.WithFeatureState` (Q-10). Not named `SubclassChoice`, a Bard value in `class-bard.md` (FR-104). |
| (subclass chosen / refused) | event `SubclassChosen`, the one member of the closed `LevelAdvancementEvent` (implements `CharacterEvent`, carries the `CharacterID`: FR-049); `SubclassRejected` holding enum `SubclassRejectionReason`: `NotInClass`, `BelowSubclassLevel`, `AlreadyChosen`, `UnknownSubclass` | `shared-types-and-ownership.md` FR-018 to FR-020. |
| Multiclass Proficiencies | `MulticlassTraining` domain service | FR-112. |
| Extra Attack | `ExtraAttackCombination` pure function over per-class attack counts | p. 25; counts come from the class specs. |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. Withdrawn ids are in section 4.10.

### 4.1 Character Advancement table: not owned here

The table (p. 23) belongs to `character-aggregate.md` FR-004, FR-006 to FR-008, FR-024, FR-028, FR-029 and
`derived-stats.md` FR-059; this spec adds only `ExperienceStillNeeded` (FR-013).

### 4.2 Previewing a level (the level-up flow)

`PreviewLevelAdvancement` is the read-only counterpart of `LevelUpCharacter` for a `Character` and a `ClassID`. The
five steps of "Gaining a Level" (p. 23) are specified in `character-aggregate.md` FR-024 to FR-046 (steps 1, 2, 4, 5)
and `content-classes.md` FR-048 (step 3); this spec previews steps 1, 3 and 4 (FR-009 to FR-017) and does not preview
step 2 (the gain depends on a roll; the die is `HitPointDie`, FR-108). **Availability (Q-13):** the use case builds
the `AdvancementClass` from the looked-up `ClassDefinition`, calls `Character.LevelUp` with the class's
`FixedHitPointGain` (faces / 2 + 1, `character-aggregate.md` FR-018) and the `RequirementsOfCurrentClasses` map, and
reads (never keeps) the returned character, so availability, rejection, levels and Proficiency Bonus cannot disagree
with the aggregate. `LevelAdvancementGain` holds `ClassStatus`, `NewClassLevel`, `NewTotalLevel`,
`ProficiencyBonusAfter`, `ProficiencyBonusIncreases`, `HitPointDie`, `FeaturesGained`, `HasSubclassFeature`,
`PendingChoices` and `SpellSlotsAfter` (Q-13); `LevelAdvancementBlock` holds `ClassStatus`, the `LevelUpRejection` and
`ExperienceStillNeeded`. The subclass chosen so far is read from `FeatureStates` (FR-048).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-009 | For a character that is eligible, choosing a class it already has, the result MUST be `PreviewLevelAdvancementAvailable` with `ClassStatus` `AdvancementClassStatusExisting(CurrentClassLevel)`, `NewClassLevel` one above the current level in the class and `NewTotalLevel` one above the total level. | SRD 5.2.1, "Gaining a Level" step 1, p. 23 | happy |
| FR-010 | For a character that is eligible and meets the prerequisites, choosing a class it does not have, the result MUST be `PreviewLevelAdvancementAvailable` with `ClassStatus` `AdvancementClassStatusNew`, `NewClassLevel` 1 and `NewTotalLevel` one above the total level. | SRD 5.2.1, "Multiclassing", "Prerequisites", p. 24 | happy |
| FR-011 | When a level cannot be gained the result MUST be `PreviewLevelAdvancementUnavailable` carrying the one `LevelUpRejection` that `Character.LevelUp` returns (the aggregate's fixed order: `LevelUpMaximumLevel`, then `LevelUpInsufficientExperience`, then `LevelUpPrerequisitesNotMet` with `ShortAbilities`); prerequisites are evaluated only when the chosen class is new (table PR). | SRD 5.2.1, "Level Advancement", p. 23; "Multiclassing", "Prerequisites", p. 24; `character-aggregate.md` FR-027 to FR-032, FR-043 | unhappy (state error) |
| FR-012 | For every case of table PR and every combined rejection case of `character-aggregate.md` section 4.5, the result MUST be available if and only if `Character.LevelUp` returns `CharacterChanged` for the same character, class and prerequisites, and an unavailable result's rejection MUST equal the one `LevelUp` returns. This holds by construction (Q-13); the test pins it. | `character-aggregate.md` FR-024 to FR-032, FR-043; Q-13 | unhappy (rule interaction) |
| FR-013 | `ExperienceStillNeeded` MUST be the minimum XP for the total level plus one (`ExperiencePoints.MinimumForLevel`), less the character's XP, and 0 when that is not positive or when the total level is 20 (table XN). | SRD 5.2.1, Character Advancement table, p. 23 | unhappy (boundary) |
| FR-014 | `ProficiencyBonusAfter` MUST be the Proficiency Bonus of the character `LevelUp` returns, and `ProficiencyBonusIncreases` MUST be true exactly when it is higher than at the current total level (table PB; the bonus follows total level, so a level in a second class raises it at the same totals as in the first). | SRD 5.2.1, "Gaining a Level" step 4, p. 23; "Multiclassing", "Proficiency Bonus", p. 25 | unhappy (boundary) |
| FR-015 | `FeaturesGained` MUST be `ClassDefinition.RowFeatures(NewClassLevel).entries` (`content-classes.md` FR-048): the Features table row at the new class level with each cell resolved to its feature entry, in printed order, empty when the row names none. A repeated feature (Ability Score Improvement at Wizard 8, Fighter 6, Rogue 10) is one entry at its first level that later rows reference, so the entry's own `level` is lower than `NewClassLevel`; `EntriesGainedAt` MUST NOT be used for this field. `HasSubclassFeature` MUST be the row's `HasSubclassFeature`, so a row that only says "Subclass feature" is reported. For a new class this is the level 1 row (FR-108). | SRD 5.2.1, "Gaining a Level" step 3, p. 23; "Multiclassing", "Class Features", p. 25; Wizard Features table, p. 77; Fighter, p. 47; Rogue, p. 62 | happy |
| FR-016 | `PendingChoices` MUST contain one `PendingFeatGrant` (class, class level, `FeatureID` of the resolved entry, `FeatCategory`) for each entry of `RowFeatures(NewClassLevel).entries` that carries a `GrantsFeat` fact (`content-classes.md` FR-055) of category `General` (Ability Score Improvement) or `Epic Boon`, and no other feat grant; its categories MUST equal those of `ClassDefinition.FeatGrantsAt(NewClassLevel)` (`content-classes.md` FR-049) that are `General` or `Epic Boon`. For a repeated grant the `FeatureID` is the older entry's id and the `ClassLevel` tells the grants apart. The match is by typed fact and `FeatureID`, never by display name. A host builds the `TakeFeat` grant as `feats-behaviour.md` table G pins it (source `ClassFeatureFeat(ClassID, FeatureID)`, scope `AnyFeat`); this spec holds no `FeatGrant`. Class-specific grants (Fighting Style, Expertise) are reported by the class specs. Whether a grant is used is `CharacterFeats.TakenFrom(source)` (`feats-behaviour.md` FR-062) for `TakeFeat`, and the `ConsumedFeatGrant` of `higher-level-start-equipment-and-trinkets.md` FR-118 for a start step; no other ledger is kept here. | SRD 5.2.1, Features tables: Wizard (Ability Score Improvement at 4, 8, 12, 16; Epic Boon at 19), p. 77; Fighter (4, 6, 8, 12, 14, 16), p. 47; Rogue (4, 8, 10, 12, 16), p. 62; "Parts of a Feat", p. 87 | happy |
| FR-017 | A `PendingSubclassChoice` MUST be reported for the chosen class when the new class level is at or above `ClassDefinition.SubclassLevel` (the level of its one `GrantsSubclass` feature, `content-classes.md` FR-056) and the character holds no `ChosenSubclass` under that class's key; it MUST NOT be reported once one is held. A pending subclass choice MUST NOT make the preview unavailable (FR-051). The check reads the `GrantsSubclass` fact, not a feature name; a later row that only says "Subclass feature" sets `HasSubclassFeature` (FR-015) and nothing more. | SRD 5.2.1, "Level 3: Cleric Subclass", p. 37; "Level 3: Rogue Subclass", p. 62; "Create Your Character", p. 19 | unhappy (rule interaction) |
| FR-018 | `SpellSlotsAfter` MUST be the `SpellSlotPools` (the Spellcasting `SpellSlots` and the optional `PactMagicSlots`, owned by `spellcasting.md`, Q-13) that `SpellcastingRules.SlotsFor` returns for the `ClassLevels` after the level, at full maximums (a preview spends nothing): multiclass slots follow the Multiclass Spellcaster table when two or more classes have the Spellcasting feature, else the class's own table (table SS; rules are `spellcasting.md` FR-030 to FR-034). | SRD 5.2.1, "Multiclassing", "Spellcasting", pp. 25-26; Multiclass Spellcaster table, p. 26 | happy |
| FR-019 | The `ClassLevels` of an available result MUST differ from the current ones by exactly one level in the chosen class, so `AdvanceSpellcasting` accepts them and never reports `LevelsSkipped`, `LevelLowered` or `ClassRemoved`. | `spellcasting.md` FR-112; SRD 5.2.1, "Gaining a Level", p. 23 | unhappy (state error) |
| FR-020 | A chosen class that the `content.Source` does not have MUST give `PreviewLevelAdvancementContentMissing` listing its `MissingContent` and no preview. | `doc/rules/architecture.md` §4, §6; `doc/rules/testing.md` §4 | unhappy (missing content) |
| FR-021 | When the chosen class is new, the use case MUST also look up every class the character already has; if any lookup is not found the result MUST be `PreviewLevelAdvancementContentMissing` listing every missing `MissingContent` (chosen and current), defensively copied, without duplicates, ordered as in `character-aggregate.md` FR-077, with no prerequisite evaluated. When the chosen class is already held, no other class MUST be looked up. | `character-aggregate.md` FR-061, FR-077 | unhappy (missing content) |
| FR-022 | A `*srderr.PortError` from the content port MUST propagate out of the preview unchanged, with no result built. | `doc/rules/architecture.md` §4; `character-aggregate.md` FR-062 | unhappy (collaborator failure) |
| FR-023 | A preview MUST NOT alter its command or the `Character`, MUST return an equal result for the same inputs on every call, and MUST expose only defensively copied collections. | `doc/rules/architecture.md` §3, §4; `doc/rules/testing.md` §4 | unhappy (determinism, immutability) |
| FR-024 | For a character at total level 20 the result MUST be `PreviewLevelAdvancementUnavailable` with `LevelUpMaximumLevel` and `ExperienceStillNeeded` 0, whatever the XP and whether the chosen class is held or new. | SRD 5.2.1, Character Advancement table (ends at level 20), p. 23 | unhappy (boundary) |

Table PR (FR-011, FR-012, FR-052). Follows the p. 24 example (a Barbarian multiclassing into Druid needs Strength and
Wisdom of 13 or higher; Primary Abilities Strength and Wisdom, p. 19). Fighter is "Strength or Dexterity" (`AnyOf`),
Monk "Dexterity and Wisdom" (`AllOf`), Wizard Intelligence. `ShortAbilities` is in `Ability` declaration order and,
for an unmet `AnyOf`, holds all its abilities (`character-aggregate.md` FR-031). Sources: pp. 19, 23-25.

| Label | Class levels | XP | Choose | Scores | Expected |
|---|---|---|---|---|---|
| existing class, enough XP | Barbarian 1 | 300 | Barbarian | any | Available, `Existing(1)` |
| new class, SRD example | Barbarian 1 | 300 | Druid | STR 13, WIS 13 | Available, `New` |
| new class, Strength short (12 fails) | Barbarian 1 | 300 | Druid | STR 12, WIS 13 | Unavailable(`LevelUpPrerequisitesNotMet [Strength]`) |
| new class, Wisdom short | Barbarian 1 | 300 | Druid | STR 13, WIS 12 | Unavailable(`LevelUpPrerequisitesNotMet [Wisdom]`) |
| new class, both short | Barbarian 1 | 300 | Druid | STR 12, WIS 12 | Unavailable(`LevelUpPrerequisitesNotMet [Strength, Wisdom]`) |
| new class, comfortably above (14 and more pass) | Barbarian 1 | 300 | Druid | STR 15, WIS 18 | Available |
| existing class skips prerequisites | Barbarian 1 | 300 | Barbarian | STR 8 | Available |
| Fighter new, Strength 13 only | Wizard 1 | 300 | Fighter | INT 13, STR 13, DEX 8 | Available (`AnyOf` met by Strength) |
| Fighter new, Dexterity 13 only | Wizard 1 | 300 | Fighter | INT 13, STR 8, DEX 13 | Available (`AnyOf` met by Dexterity) |
| Fighter new, neither at 13 | Wizard 1 | 300 | Fighter | INT 13, STR 12, DEX 12 | Unavailable(`LevelUpPrerequisitesNotMet [Strength, Dexterity]`) |
| current class short | Wizard 1 | 300 | Fighter | INT 12, STR 13, DEX 8 | Unavailable(`LevelUpPrerequisitesNotMet [Intelligence]`) |
| two-ability class, both at 13 | Barbarian 1 | 300 | Monk | STR 13, DEX 13, WIS 13 | Available |
| two-ability class, Dexterity short | Barbarian 1 | 300 | Monk | STR 13, DEX 12, WIS 13 | Unavailable(`LevelUpPrerequisitesNotMet [Dexterity]`) |
| two-ability class, both short | Barbarian 1 | 300 | Monk | STR 13, DEX 12, WIS 12 | Unavailable(`LevelUpPrerequisitesNotMet [Dexterity, Wisdom]`) |
| not enough XP, existing class | Barbarian 1 | 299 | Barbarian | any | Unavailable(`LevelUpInsufficientExperience`), still needed 1 |
| XP beats prerequisites | Barbarian 1 | 299 | Druid | STR 12, WIS 12 | Unavailable(`LevelUpInsufficientExperience`) |
| maximum level beats everything | Barbarian 20 | 400,000 | Druid | STR 12, WIS 12 | Unavailable(`LevelUpMaximumLevel`) |

Table XN (FR-013; p. 23), (total level, XP) gives still needed: (1, 0) 300; (1, 299) 1; (1, 300) 0; (1, 6,500) 0; (19,
354,999) 1; (19, 355,000) 0; (20, 400,000) 0.

Table PB (FR-014; pp. 23, 25), total level before to after: bonus before to after, increases. Single class: 1 to 2: +2
to +2, no; 4 to 5: +2 to +3, yes; 8 to 9: +3 to +4, yes; 12 to 13: +4 to +5, yes; 16 to 17: +5 to +6, yes; 19 to 20:
+6 to +6, no. Multiclass: Fighter 4 to Fighter 4 / Rogue 1 (total 4 to 5): +2 to +3, yes; Fighter 3 / Rogue 2 to
Fighter 3 / Rogue 3 (total 5 to 6): +3 to +3, no.

Table PC (FR-015 to FR-017). Features tables: Wizard (p. 77) Ability Score Improvement at 4, 8, 12, 16, Epic Boon 19,
"Subclass feature" 6, 10, 14; Fighter (p. 47) Ability Score Improvement 4, 6, 8, 12, 14, 16, Epic Boon 19, "Fighter
Subclass" 3, "Subclass feature" 7, 10, 15, 18; Rogue (p. 62) Ability Score Improvement 4, 8, 10, 12, 16, "Rogue
Subclass" 3. A repeated grant carries the same `FeatureID` and differs only in `ClassLevel`; `feat` is a
`PendingFeatGrant`.

| Label | Class | New class level | Subclass held | Expected pending choices |
|---|---|---|---|---|
| Wizard level 4 | Wizard | 4 | `evoker` | `feat` (Ability Score Improvement entry, `General`, ClassLevel 4) |
| Wizard level 6, subclass row | Wizard | 6 | `evoker` | none; subclass row |
| Wizard level 8, repeated grant | Wizard | 8 | `evoker` | `feat` (the level 4 entry's id, `General`, ClassLevel 8) |
| Wizard level 12, repeated grant | Wizard | 12 | `evoker` | `feat` (same id, ClassLevel 12) |
| Wizard level 16, repeated grant | Wizard | 16 | `evoker` | `feat` (same id, ClassLevel 16) |
| Wizard level 19 | Wizard | 19 | `evoker` | `feat` (Epic Boon entry, `Epic Boon`) |
| Fighter level 6, repeated grant | Fighter | 6 | `champion` | `feat` (the level 4 entry's id, ClassLevel 6) |
| Fighter level 14, repeated grant | Fighter | 14 | `champion` | `feat` (same id, ClassLevel 14) |
| Rogue level 10, repeated grant | Rogue | 10 | `thief` | `feat` (the Rogue level 4 entry's id, ClassLevel 10) |
| Rogue level 6, Expertise | Rogue | 6 | `thief` | none (Expertise is class-specific) |
| Fighter level 7, subclass row | Fighter | 7 | `champion` | none; subclass row (not a pending choice) |
| Wizard level 3, no subclass | Wizard | 3 | none | `PendingSubclassChoice` |
| Wizard level 3, subclass held | Wizard | 3 | `evoker` | none |
| Fighter level 4, subclass still pending | Fighter | 4 | none | `PendingSubclassChoice`, `feat` |
| Fighter level 2, below subclass level | Fighter | 2 | none | none |
| Fighter level 1 of a new class | Fighter | 1 | none | none (the Fighting Style entry is `GrantsFeat` of category `Fighting Style`, not reported here) |
| level with no named choice | Wizard | 7 | `evoker` | none; no subclass row |

Table SS (FR-018, FR-019). Source: pp. 25-26 and `spellcasting.md` table H.

| Label | Class levels before | Choose | Expected `SpellSlotsAfter` (Spellcasting pool; Pact Magic) |
|---|---|---|---|
| SRD example | Ranger 4, Sorcerer 2 | Sorcerer | total spellcaster level 5: four level 1, three level 2, two level 3; no Pact Magic |
| first spellcaster level | Fighter 3 | Wizard | Wizard's own table, level 1: two level 1 slots |
| second spellcasting class | Wizard 3 | Cleric | total level 4: four level 1, three level 2 |
| non-spellcasting class adds nothing | Wizard 5 | Fighter | table C at 5: four level 1, three level 2, two level 3 |
| Warlock is a separate pool | Warlock 5, Wizard 3 | Wizard | Wizard is the only Spellcasting class, so its own table at level 4: four level 1, three level 2; Pact Magic as Warlock 5 |

### 4.3 Ability Score Improvement and Epic Boon (the increase)

The feats are content (`content-species-backgrounds-feats.md`); the player's choice and whether the feat may be taken
are `feats-behaviour.md` (it converts its choices: `feats-behaviour.md` FR-020, FR-023, FR-024, and reports a refusal
as `FeatIncreaseRejected`, FR-079). The Hit Point rule for a Constitution increase is the aggregate's
(`character-aggregate.md` FR-045). There is **no standalone use case** that applies an increase: it reaches a
character only through `TakeFeat` for a grant a class level opened (FR-016) or a start step (`higher-level-start-equipment-and-trinkets.md` FR-113). The level
clauses (Level 4+, Level 19+) are clauses of the feats (`feats-behaviour.md` FR-009, FR-012, FR-015); the increaser
does not judge level or classes (FR-037).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-025 | `NewAbilityScoreImprovementOneByTwo(ability)` MUST hold an increase of 2 to that ability and `MaximumScore` 20. | SRD 5.2.1, "Ability Score Improvement" ("Increase one ability score of your choice by 2 ... can't increase an ability score above 20"), p. 87 | happy |
| FR-026 | `NewAbilityScoreImprovementTwoByOne(first, second)` MUST hold an increase of 1 to each of two distinct abilities and `MaximumScore` 20, and MUST return an error matching `srderr.ErrInvalidInput` when the abilities are equal. | SRD 5.2.1, "Ability Score Improvement" ("or increase two ability scores of your choice by 1"), p. 87 | happy / unhappy (invalid input) |
| FR-027 | `AbilityScoreIncrease.EpicBoon(ability)` MUST hold an increase of 1 to that ability and `MaximumScore` 30. | SRD 5.2.1, "Epic Boon Feats" (Boon of Combat Prowess: "Increase one ability score of your choice by 1, to a maximum of 30"), p. 88 | happy |
| FR-028 | The general constructor `NewAbilityScoreIncrease(increases, maximumScore)` (the extension point for other feats, for example Grappler's +1 to Strength or Dexterity with a maximum of 20) MUST return `(AbilityScoreIncrease, error)`: an error matching `srderr.ErrInvalidInput` for an empty `increases`, any amount below 1, or a `maximumScore` below 20 or above 30, carrying the offending parameter name and invalid value. `MustAbilityScoreIncrease` panics with that same error. | SRD 5.2.1, "Grappler", p. 87; "Ability Scores", p. 5 (20 and 30) | unhappy (invalid input) |
| FR-029 | `NewAbilityScoreIncrease` MUST NOT panic for any input and, as an aggregate parser, MUST report every problem found (`errors.Join` of typed problems, each matching `srderr.ErrInvalidInput`, in the fixed order empty, non-positive amount (by ability declaration order), maximum below 20, maximum above 30) if and only if the input is invalid; the `MustX` form panics with the same error (`errors-and-results.md` FR-023, FR-025, FR-026). | `errors-and-results.md` FR-023, FR-025, FR-026 | unhappy (invalid input) |
| FR-030 | `AbilityScoreIncrease` MUST compare by value (`Equal`) and expose `Increases()` as a defensive copy (mutating the map passed in later does not change it; mutating the returned map leaves the value unchanged). | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |
| FR-031 | Applying an increase (`AbilityScoreIncreaser`) MUST give each listed ability a score equal to its old score plus its amount, leave the others unchanged, and emit one `AbilityScoreChanged` per listed ability in `Ability` declaration order (Strength, Dexterity, Constitution, Intelligence, Wisdom, Charisma). | SRD 5.2.1, "Ability Score Improvement", p. 87; "Gaining a Level" step 5, p. 23 | happy |
| FR-032 | Each resulting score MUST NOT exceed `MaximumScore`: a result equal to the maximum MUST be accepted and one above it rejected, with no clamping and no partial increase. A score already above the maximum (22 under a maximum of 20) is rejected for any positive amount (table AI). | SRD 5.2.1, "Ability Score Improvement" ("can't increase an ability score above 20"), p. 87; "Epic Boon Feats" ("to a maximum of 30"), p. 88; Q-7 | unhappy (boundary) |
| FR-033 | When any listed ability would exceed the maximum the increaser MUST return `IncreaseExceedsMaximum` with every such ability in declaration order and the maximum, no ability MUST change, no event MUST be emitted, and the `Character` MUST be unchanged. | SRD 5.2.1, "Ability Score Improvement", p. 87 | unhappy (rule interaction) |
| FR-034 | When the Constitution modifier rises, `Character.HitPointMaximum` MUST rise by the aggregate's rule (one per level attained for each point the modifier rises; the SRD example is Constitution 17 to 18 at level 8, +8) and one `HitPointMaximumChanged` MUST be emitted (table AI). The modifier is read from the Ability Modifiers table, not from the parity of the score. | SRD 5.2.1, "Gaining a Level" step 5, p. 23; Ability Modifiers table, p. 6; `character-aggregate.md` FR-045, FR-046 | happy |
| FR-035 | A Constitution increase that leaves the modifier unchanged, and any increase to another ability, MUST NOT change the Hit Point maximum and MUST NOT emit `HitPointMaximumChanged`. | SRD 5.2.1, "Gaining a Level" step 5, p. 23; `character-aggregate.md` FR-046 | unhappy (rule interaction) |
| FR-036 | Event order MUST be every `AbilityScoreChanged` (declaration order) followed by at most one `HitPointMaximumChanged`, whose old value is the maximum before the whole increase and whose new value is the maximum after it, even when an ability listed after Constitution also changes (table AI, event rows). `TakeFeat` puts `FeatTaken` before these events (`feats-behaviour.md` FR-026). | `character-aggregate.md` FR-058 (extended to several abilities); `doc/rules/architecture.md` §3 | unhappy (determinism) |
| FR-037 | `AbilityScoreIncreaser` MUST NOT reject because of the character's level or classes (a level 1 character may be given an Ability Score Improvement increase by a direct call); whether the feat may be taken is `feats-behaviour.md`'s decision, and whether a start step may carry an increase is `higher-level-start-equipment-and-trinkets.md` FR-113. | SRD 5.2.1, "Ability Score Improvement" (Prerequisite: Level 4+), "Parts of a Feat", p. 87 | unhappy (rule interaction) |
| FR-038 | Applying the same increase twice MUST be two independent applications, each checked against the maximum on its own, because the Ability Score Improvement feat is Repeatable (table AI, repeat rows). | SRD 5.2.1, "Ability Score Improvement" ("Repeatable. You can take this feat more than once"), p. 87 | unhappy (state error) |
| FR-040 | `AbilityScoreIncreaser` MUST NOT alter its inputs and MUST return an equal result for the same inputs on every call. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (immutability, determinism) |

Table AI (FR-031 to FR-038). Unless a row says otherwise the character is a single-class Fighter whose levels all use
the fixed value (so the per-level minimum of 1 never binds), as in `character-aggregate.md` section 4.6. Modifiers are
from the Ability Modifiers table, p. 6.

| Label | Increase | Scores before | Total level | Expected | Path |
|---|---|---|---|---|---|
| one by two | Ability Score Improvement, +2 Strength | STR 15 | 4 | STR 17 (+2 to +3); events: `AbilityScoreChanged` | happy |
| one by two, up to the maximum | +2 Strength | STR 18 | 4 | STR 20 | unhappy (boundary) |
| one by two, one above the maximum | +2 Strength | STR 19 | 4 | rejected, `[Strength]`, maximum 20; STR stays 19 | unhappy (boundary) |
| one by two, at 20 | +2 Strength | STR 20 | 4 | rejected, `[Strength]` | unhappy (boundary) |
| two by one | +1 Strength, +1 Dexterity | STR 15, DEX 14 | 4 | STR 16, DEX 15; events in order Strength, Dexterity | happy |
| two by one, both reach the maximum | +1 Strength, +1 Dexterity | STR 19, DEX 19 | 8 | STR 20, DEX 20 | unhappy (boundary) |
| two by one, one capped | +1 Strength, +1 Dexterity | STR 20, DEX 14 | 8 | rejected, `[Strength]`; DEX stays 14 (atomic) | unhappy (rule interaction) |
| two by one, both capped | +1 Strength, +1 Dexterity | STR 20, DEX 20 | 8 | rejected, `[Strength, Dexterity]` | unhappy (rule interaction) |
| score already above 20 | +1 Strength under maximum 20 | STR 22 | 12 | rejected, `[Strength]` | unhappy (boundary) |
| single +1 with maximum 20 | general constructor, +1 Dexterity, maximum 20 | DEX 13 | 4 | DEX 14 (+1 to +2) | happy |
| Epic Boon | `EpicBoon(Strength)` | STR 20 | 19 | STR 21 (+5 to +5); no Hit Point change | happy |
| Epic Boon, modifier rises | `EpicBoon(Strength)` | STR 21 | 19 | STR 22 (+5 to +6) | happy |
| Epic Boon, up to 30 | `EpicBoon(Strength)` | STR 29 | 19 | STR 30 | unhappy (boundary) |
| Epic Boon, one above 30 | `EpicBoon(Strength)` | STR 30 | 19 | rejected, `[Strength]`, maximum 30 | unhappy (boundary) |
| Constitution, SRD example | general constructor, +1 Constitution, maximum 20 | CON 17 | 8 | CON 18 (+3 to +4); Hit Point maximum +8; events `AbilityScoreChanged`, `HitPointMaximumChanged` | happy |
| Constitution, one by two, modifier rises by one | +2 Constitution | CON 16 | 8 | CON 18 (+3 to +4); +8 | happy |
| Constitution, modifier unchanged | general constructor +1 Constitution, maximum 20 | CON 16 | 8 | CON 17 (+3 to +3); no `HitPointMaximumChanged` | unhappy (rule interaction) |
| Constitution, odd score to even score | general constructor +1 Constitution, maximum 20 | CON 15 | 5 | CON 16 (+2 to +3); +5 | happy |
| Constitution at level 19 | `EpicBoon(Constitution)` | CON 21 | 19 | CON 22 (+5 to +6); +19 | unhappy (boundary) |
| other ability, no Hit Point change | +2 Wisdom | WIS 15 | 8 | WIS 17; Hit Point maximum unchanged; no `HitPointMaximumChanged` | unhappy (rule interaction) |
| event order with a later ability | +1 Constitution, +1 Wisdom | CON 17, WIS 11 | 8 | events: `AbilityScoreChanged(Constitution)`, `AbilityScoreChanged(Wisdom)`, `HitPointMaximumChanged` (+8) | unhappy (determinism) |
| event order with an earlier ability | +1 Strength, +1 Constitution | STR 15, CON 17 | 8 | events: `AbilityScoreChanged(Strength)`, `AbilityScoreChanged(Constitution)`, `HitPointMaximumChanged` | unhappy (determinism) |
| level 1 character, direct call | +2 Wisdom | WIS 10 | 1 | WIS 12; accepted (FR-037) | unhappy (rule interaction) |
| repeat, second application | +2 Strength, twice | STR 15 | 8 | 17, then 19 | unhappy (state error) |
| repeat, third application | +2 Strength, three times | STR 15 | 12 | 17, 19, then rejected (21 over 20) | unhappy (state error) |

### 4.4 Choosing a subclass

The chosen subclass is held in `FeatureStates` under `SubclassStateKey.ForClass(ClassID)`, a
`FeatureStateKey<ChosenSubclass>` named `subclass_` plus the `ClassID` (`character-aggregate.md` FR-093;
`content-schema-and-decoder.md` FR-143), written by `ChooseSubclass` with `Character.WithFeatureState` (Q-10). The subclass
level is `ClassDefinition.SubclassLevel` (`content-classes.md` FR-056); a subclass is validated with
`ClassDefinition.subclass` (`content-classes.md` FR-074). An id the class does not list is a rule rejection, not
content-missing (Q-19).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-041 | `ChooseSubclass` MUST accept a subclass when the character has levels in the class, its level in that class is at least the class's subclass level, the character holds no `ChosenSubclass` under that class's key, and `ClassDefinition.subclass` finds the subclass id; it MUST return `ChooseSubclassSucceeded` with the new `Character` (the `ChosenSubclass` written under the key, every other state unchanged) and one `SubclassChosen` event (character id, class, subclass, class level). | SRD 5.2.1, "Level 3: Cleric Subclass", p. 37; "Level 3: Rogue Subclass", p. 62; "Create Your Character", p. 19 | happy |
| FR-042 | A class the character has no levels in MUST be rejected with `SubclassRejected(NotInClass)`. | SRD 5.2.1, "Multiclassing" (levels in a class carry its features), p. 25 | unhappy (state error) |
| FR-044 | A choice made when the level in the class is below the class's subclass level MUST be rejected with `SubclassRejected(BelowSubclassLevel)`, and one at exactly that level accepted. A choice above it MUST be accepted, so a pending choice can be made at any later level (FR-107). The level compared is the level in the class, not the total level (a Cleric 2 / Fighter 4 cannot choose a Cleric subclass). | SRD 5.2.1, "Level 3: Cleric Subclass", p. 37; "Multiclassing", p. 25 | unhappy (boundary) |
| FR-045 | A second choice for a class that already holds a `ChosenSubclass` MUST be rejected with `SubclassRejected(AlreadyChosen)`, even for a different valid subclass. A character with two classes MAY hold one subclass in each. "One subclass per class, not replaceable" is a package decision: the SRD says only "You gain a Rogue subclass of your choice" (Q-18). | SRD 5.2.1, "Level 3: Rogue Subclass", p. 62; package design (Q-18) | unhappy (state error) |
| FR-046 | A subclass id that `ClassDefinition.subclass` does not find (including another class's subclass id) MUST be rejected with `SubclassRejected(UnknownSubclass)`; it is not content-missing (Q-19). | `content-classes.md` FR-074; package design (Q-19) | unhappy (invalid input) |
| FR-047 | When several reasons hold, the one reported MUST be chosen in the fixed order `NotInClass`, `BelowSubclassLevel`, `AlreadyChosen`, `UnknownSubclass`. | `character-aggregate.md` D-4 (determinism) | unhappy (rule interaction) |
| FR-048 | The subclass MUST be stored only under `SubclassStateKey.ForClass(ClassID)`, so a class holds at most one; `ChosenSubclass` MUST compare by value; `ChooseSubclass` MUST return a new `Character` and leave the original unchanged; reading a key whose stored type differs MUST return an error matching `srderr.ErrInvalidInput` (`character-aggregate.md` FR-093). | `character-aggregate.md` FR-092, FR-093; `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |
| FR-049 | `SubclassChosen` MUST carry the `CharacterID`, the `ClassID`, the `SubclassID` and the level in the class, MUST be immutable, compare by value, be past tense, and be a variant of the closed `LevelAdvancementEvent`, which implements the open interface `CharacterEvent` (so a host switches exhaustively over this spec's events); `SubclassRejected` MUST carry its reason and no display text. | `shared-types-and-ownership.md` FR-019, FR-020; `character-aggregate.md` FR-085 | happy |
| FR-050 | A class the `content.Source` does not have MUST give `ChooseSubclassContentMissing` listing its `MissingContent`, a `*srderr.PortError` from the port MUST propagate unchanged, and neither MUST build a result or change the `Character`. | `doc/rules/architecture.md` §4, §6 | unhappy (missing content, collaborator failure) |
| FR-051 | No rule of this spec MUST reject a level-up, or an ability score increase, because a subclass has not been chosen; the choice stays pending (FR-017). This answers `class-fighter.md` section 9 ("Can a Fighter reach level 4 without having chosen a subclass?"): yes, and the aggregate does not block. | SRD 5.2.1, "Gaining a Level", p. 23 (no step blocks on a subclass) | unhappy (state error) |
| FR-107 | `SubclassSelection.IsPending(Character, ClassDefinition)` MUST be true exactly when the character has levels in the class, its level in that class is at least `ClassDefinition.SubclassLevel`, and it holds no `ChosenSubclass` under the class's key. It is pure and uses the same rule as FR-017 and FR-041, so a host can list what is owed after a start or a house-rule change (table SP). | SRD 5.2.1, "Create Your Character" ("Starting at a Higher Level"), p. 19 | happy |

Table SP (FR-107; p. 19), `IsPending` by class levels, class, held subclass: Fighter 3, `fighter`, none: true; Fighter
2, `fighter`, none: false; Fighter 5, `fighter`, `champion`: false; Fighter 5, `wizard` (not held), none: false;
Fighter 3 / Wizard 1, `wizard`, none: false; Fighter 3 / Wizard 3, `wizard`, none: true.

Table SC (FR-041, FR-042, FR-044 to FR-047). Subclass level 3 for the sample classes (Cleric p. 37, Rogue p. 62);
subclass ids `life_domain` and `thief` (`content-classes.md` bundled list). "Held" is the `ChosenSubclass` already on
the character.

| Label | Class levels | Class chosen | Subclass | Held | Expected |
|---|---|---|---|---|---|
| at the subclass level | Cleric 3 | `cleric` | `life_domain` | none | `ChooseSubclassSucceeded`, `SubclassChosen` |
| above the subclass level | Cleric 5 | `cleric` | `life_domain` | none | accepted |
| one below the subclass level | Cleric 2 | `cleric` | `life_domain` | none | `SubclassRejected(BelowSubclassLevel)` |
| multiclass, class level too low | Cleric 2, Fighter 4 | `cleric` | `life_domain` | none | `BelowSubclassLevel` |
| multiclass, one in each | Cleric 3, Rogue 3 | `rogue` | `thief` | cleric: `life_domain` | accepted; both held |
| class not held | Fighter 4 | `cleric` | `life_domain` | none | `NotInClass` |
| already chosen, same | Cleric 3 | `cleric` | `life_domain` | cleric: `life_domain` | `AlreadyChosen` |
| already chosen, different | Cleric 3 | `cleric` | another listed subclass | cleric: `life_domain` | `AlreadyChosen` |
| unknown subclass | Cleric 3 | `cleric` | `no_such_subclass` | none | `UnknownSubclass` |
| another class's subclass | Cleric 3 | `cleric` | `thief` | none | `UnknownSubclass` |
| order: not in class beats all | Fighter 4 | `cleric` | `no_such_subclass` | cleric: `life_domain` | `NotInClass` |
| order: below level beats already chosen | Cleric 2 | `cleric` | `no_such_subclass` | cleric: `life_domain` | `BelowSubclassLevel` |
| order: already chosen beats unknown | Cleric 3 | `cleric` | `no_such_subclass` | cleric: `life_domain` | `AlreadyChosen` |

### 4.5 Multiclassing: cross-checks, grants of a new class, Extra Attack and delegated rules

Delegated rules (Experience Points, Hit Points and Hit Dice, Proficiency Bonus, Armor Class, Spellcasting) are FR-059
to FR-061, FR-063, FR-064; Proficiencies and Class Features are FR-062, FR-108 to FR-112 (consumed by
`derived-stats.md` FR-019, FR-040); Extra Attack is FR-054 to FR-058.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-052 | A preview for a new class MUST apply the prerequisite of the new class and of every current class: each requirement is an `AbilityRequirement` evaluated as `AllOf` or `AnyOf` per `character-aggregate.md` FR-031, a score of 12 fails and 13 passes (and 14 passes), and `ShortAbilities` is as that requirement defines it (table PR: single-ability classes, Fighter's `AnyOf` with Strength 13 only, Dexterity 13 only and neither, and Monk's `AllOf`). | SRD 5.2.1, "Multiclassing", "Prerequisites", pp. 24-25; Class Overview table, p. 19 | unhappy (boundary) |
| FR-053 | A character MAY gain its first level in a second class at total level 1 (the second level overall) and MAY return to an earlier class afterwards: Fighter 1, then Rogue 1, then Fighter 2 are each available when the XP and prerequisites allow (table MS). | SRD 5.2.1, "Multiclassing" ("the option of gaining a level in a new class whenever you advance in level"), p. 24 | happy |
| FR-054 | The number of attacks a multiclass character makes with the Attack action from Extra Attack MUST be the largest of the per-class counts: the features do not stack (table EA). | SRD 5.2.1, "Multiclassing", "Extra Attack", p. 25 | unhappy (rule interaction) |
| FR-055 | A count above 2 MUST NOT arise from adding two classes' counts: two classes that each give 2 give 2, and a class that gives 3 or 4 keeps its count when another class gives 2. The SRD sentence is: "You can't make more than two attacks with this feature unless you have a feature that says you can (such as the Fighter's Two Extra Attacks feature)." | SRD 5.2.1, "Multiclassing", "Extra Attack", p. 25 | unhappy (boundary) |
| FR-056 | For a single class, or for classes none of which gives Extra Attack (each reporting 1), the result MUST be that count unchanged. | SRD 5.2.1, "Multiclassing", "Extra Attack", p. 25 | happy |
| FR-057 | `ExtraAttackCombination` MUST return an error matching `srderr.ErrInvalidInput` for an empty collection or any count below 1, carrying the parameter `name` and `InvalidValue`. | `doc/rules/architecture.md` §6; `errors-and-results.md` FR-027 | unhappy (invalid input) |
| FR-058 | `ExtraAttackCombination` MUST NOT depend on the order of the counts, MUST NOT alter its input, and MUST return the same value for the same counts on every call. A Warlock's Thirsting Blade is not an input: it adds no attack to a character that has Extra Attack (`class-warlock.md` FR-155). | SRD 5.2.1, "Multiclassing", "Extra Attack" (Thirsting Blade "doesn't give you additional attacks if you also have Extra Attack"), p. 25 | unhappy (determinism) |
| FR-059 | Delegated: `Experience Points` for a multiclass character uses total level (`character-aggregate.md` FR-027, FR-028). | SRD 5.2.1, "Multiclassing", "Experience Points", p. 25 | n/a (delegated) |
| FR-060 | Delegated: Hit Dice are pooled by die size and only the level 1 Hit Points of the initial class are the maximum face (`character-aggregate.md` FR-038, FR-039). | SRD 5.2.1, "Multiclassing", "Hit Points and Hit Point Dice", p. 25 | n/a (delegated) |
| FR-061 | Delegated: the Proficiency Bonus follows total level (`character-aggregate.md` FR-010; `derived-stats.md` FR-009); FR-014 checks it through the preview. | SRD 5.2.1, "Multiclassing", "Proficiency Bonus", p. 25 | n/a (delegated) |
| FR-062 | Owned here: the Proficiencies and Class Features a first level in a new class grants are FR-108 to FR-112. The withdrawn multiclass rows of the class specs are not a source. | SRD 5.2.1, "Multiclassing", "Proficiencies" and "Class Features", p. 25 | n/a (see FR-108 to FR-112) |
| FR-063 | Delegated: only one Armor Class calculation applies at a time (`derived-stats.md` FR-037, FR-054). | SRD 5.2.1, "Multiclassing", "Armor Class", p. 25 | n/a (delegated) |
| FR-064 | Delegated: multiclass Spell Slots, Pact Magic, Spells Prepared and cantrip scaling by total level are `spellcasting.md` FR-030 to FR-039; FR-018 and FR-019 check that the level-up flow feeds them. | SRD 5.2.1, "Multiclassing", "Spellcasting", pp. 25-26 | n/a (delegated) |
| FR-108 | When the chosen class is new (the character's first level in a class other than its initial class), `AdvancementClassStatusNew.grants` MUST be the class's `MulticlassGrants` (weapon, armor and tool training and any `SkillChoice`, `content-classes.md` FR-026), `LevelAdvancementGain.HitPointDie` MUST be the class's `HitPointDie` (the die pooled by `character-aggregate.md` FR-037, FR-038), and the class's level 1 row MUST be in `FeaturesGained` (`RowFeatures(1)`, FR-015). `HitPointDie` is set for a held class too: it is the die of the class the level is gained in. Nothing else of the class's starting proficiencies is granted. | SRD 5.2.1, "Multiclassing", "Proficiencies" ("you gain only some of the new class's starting proficiencies"), "Class Features", p. 25; "As a Multiclass Character", pp. 28, 47, 77 | happy |
| FR-109 | A class the character already holds, and the initial class, MUST yield `AdvancementClassStatusExisting` and no grants: Fighter 1, Rogue 1, Fighter 2 grants nothing at the third step, and Rogue's grants are given once, at the second step. | SRD 5.2.1, "Multiclassing", "Proficiencies" ("When you gain your first level in a class other than your initial class"), p. 25 | unhappy (rule interaction) |
| FR-110 | The grants of a new class MUST NOT include saving throw proficiencies, starting equipment, or skill proficiencies beyond the class's `SkillChoice` grant (a class with none grants no skill); this matches `derived-stats.md` FR-019 (later classes add no saving throw proficiencies). | SRD 5.2.1, "As a Multiclass Character" (Barbarian, Fighter, Wizard lists), pp. 28, 47, 77 | unhappy (rule interaction) |
| FR-111 | The grants MUST be the same for every character: they depend on the class only, not on the character's scores, the other classes held, or the order in which classes were taken. | SRD 5.2.1, "Multiclassing", p. 25; package design (determinism) | unhappy (determinism) |
| FR-112 | `MulticlassTraining.GrantsFor(ClassLevels, definitions)` MUST return, for each class of the `ClassLevels` after the first (the initial class), that class's `MulticlassGrants` in `ClassLevels` order, and an empty list for a single-class character; it MUST return an error matching `srderr.ErrInvalidInput` when a class has no definition in the map. This is the value a host passes to `derived-stats.md` as `MulticlassArmorTraining` (FR-040) and uses for weapon, tool and skill training. | SRD 5.2.1, "Multiclassing", "Proficiencies", p. 25; `derived-stats.md` FR-019, FR-040 | happy |

Table MS (FR-053). XP is the minimum for the new total level; scores meet every prerequisite (Fighter: Strength or
Dexterity 13; Rogue: Dexterity 13). Step 1: Fighter 1 (XP 300) chooses Rogue: Available, `New`, class level 1, total
2. Step 2: Fighter 1 / Rogue 1 (XP 900) chooses Fighter: Available, `Existing(1)`, class level 2, total 3. Step 3:
Fighter 2 / Rogue 1 (XP 2,700) chooses Rogue: Available, `Existing(1)`, class level 2, total 4.

Table MG (FR-108 to FR-112). Grants are the `MulticlassGrants` of `content-classes.md` table MC; rows below are the
SRD wording. `HitPointDie` is the die of the class the level is gained in (held or new); level 1 features are always
listed for a new class.

| Label | Class levels before | Choose | Expected grants | `HitPointDie` | Source |
|---|---|---|---|---|---|
| Barbarian as a new class | Fighter 1 | Barbarian | Martial weapons, Shields; level 1 features Rage, Unarmored Defense, Weapon Mastery | d12 | p. 28 |
| Fighter as a new class | Barbarian 1 | Fighter | Martial weapons, Light and Medium armor, Shields; level 1 features | d10 | p. 47 |
| Wizard as a new class | Fighter 1 | Wizard | no proficiencies; level 1 features Spellcasting, Ritual Adept, Arcane Recovery | d6 | p. 77 |
| existing class | Fighter 1, Rogue 1 | Fighter | none (`Existing`) | d10 | pp. 25, 47 |
| third step returns to an earlier class | Fighter 2, Rogue 1 | Rogue | none (`Existing`) | Rogue's die | p. 25 |
| no saves, no equipment | Wizard 1 | Barbarian | grants hold no saving throw proficiency and no starting equipment | d12 | pp. 25, 28 |
| `GrantsFor`, two classes | Fighter 2, Wizard 1 | n/a | Wizard's grants only | n/a | p. 25 |
| `GrantsFor`, single class | Fighter 3 | n/a | empty list | n/a | p. 25 |
| `GrantsFor`, three classes | Fighter 2, Wizard 1, Barbarian 1 | n/a | Wizard's, then Barbarian's, in `ClassLevels` order | n/a | p. 25 |
| `GrantsFor`, missing definition | Fighter 1, Wizard 1; no Wizard entry | n/a | an invalid-input error (`srderr.ErrInvalidInput`) | n/a | n/a |

Table EA (FR-054 to FR-058). Per-class counts are the class specs' own (`class-fighter.md` FR-026: 1 below level 5, 2
at 5 to 10, 3 at 11 to 19, 4 at 20; `class-paladin.md` FR-085 and `class-monk.md` FR-075: 1 below level 5, 2 from
level 5). Source: p. 25.

| Label | Per-class counts | Combined |
|---|---|---|
| single class, below level 5 | Fighter 1 | 1 |
| single class, level 5 | Fighter 2 | 2 |
| no class gives Extra Attack | Fighter 1 (level 4), Paladin 1 (level 4) | 1 |
| two classes give 2, no stacking | Paladin 2 (level 5), Monk 2 (level 5) | 2 |
| three classes give 2 | Paladin 2, Ranger 2, Monk 2 | 2 |
| Fighter 5 and Monk 5 | 2, 2 | 2 |
| Fighter 11 and Paladin 5 | 3, 2 | 3 |
| Fighter 20 and Ranger 5 | 4, 2 | 4 |
| order does not matter | 2, 3 and 3, 2 | 3 and 3 |
| one class has it, other does not | Fighter 2, Paladin 1 | 2 |
| empty | none | an invalid-input error (`srderr.ErrInvalidInput`) |
| count zero | 0, 2 | an invalid-input error (`srderr.ErrInvalidInput`) |
| negative count | -1 | an invalid-input error (`srderr.ErrInvalidInput`) |

### 4.9 Use cases, telemetry and cross-cutting behaviour

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-098 | `PreviewLevelAdvancement` and `ChooseSubclass` MUST return equal results with the default the zero-value `telemetry.Noop` (`shared-types-and-ownership.md` FR-036) and with a recording one. | `doc/rules/observability.md` §9 | happy |
| FR-099 | Each use case MUST emit the signals of section 7 once per call, using defined constants for names and attribute keys, and MUST NOT log. `srd5e.outcome` MUST be only `success` or `failure`; for every returned result the variant MUST be in `srd5e.character.result` (a returned port failure carries none, FR-100). Metrics are the shared `srd5e.character.operations` counter and `srd5e.character.duration` histogram, the histogram with the single attribute `srd5e.character.operation` (`character-aggregate.md` FR-103, D-21). | `doc/rules/observability.md` §5-§7, §9; `errors-and-results.md` FR-043 | happy |
| FR-100 | A returned `...ContentMissing` MUST be recorded exactly once: `srd5e.outcome` `failure`, `srd5e.character.result` `content_missing`, span error status, one increment of the operations counter, no `error.type` and no log. A returned `*srderr.PortError` (a content port failure) MUST be recorded exactly once with `Span.RecordError`, `srd5e.outcome` `failure`, the result attribute absent, `error.type` equal to `srderr.ErrorType(err)`, and no log. | `doc/rules/observability.md` §6, §7; `errors-and-results.md` FR-034, FR-043; `character-aggregate.md` FR-071, FR-088 | unhappy (telemetry on failure) |
| FR-101 | A rule outcome (an unavailable preview, a refused subclass) MUST be recorded as a normal outcome: `srd5e.outcome` `failure`, result `unavailable` or `rejected`, `srd5e.character.rejection` on the counter and span, span status not error, no `WARNING` or `SEVERE` log. An available preview and a chosen subclass are `srd5e.outcome` `success`. | `doc/rules/observability.md` §6, §7; `character-aggregate.md` FR-072 | unhappy (telemetry on failure) |
| FR-102 | The `error.type` attribute MUST be set only for a returned `*srderr.PortError` and MUST equal `srderr.ErrorType(err)` (`errors-and-results.md` FR-034; never `%T` or `err.Error()`). `content_missing` and `unexpected` are not `error.type` values: a returned failure carries none, and a panic from a telemetry backend propagates unrecorded by a counter. | `errors-and-results.md` FR-005, FR-034, FR-043; `character-aggregate.md` FR-088 | unhappy (telemetry on failure) |
| FR-103 | Every domain type of this spec MUST contain no logging, metric or span call and MUST NOT import `telemetry` or any adapter package: `AbilityScoreIncrease`, `AbilityScoreIncreaseRejection`, `AbilityScoreIncreaser`, `ChosenSubclass`, `SubclassStateKey`, `SubclassSelection`, `SubclassRejected`, `LevelAdvancementEvent`, `SubclassChosen`, `LevelAdvancementPreviewer`, `LevelAdvancementPreview` and its variants, `AdvancementClassStatus` and its variants, `PendingAdvancementChoice` and its variants, `MulticlassTraining` and `ExtraAttackCombination`. | `doc/rules/observability.md` §3, §10; `doc/rules/architecture.md` §9 | unhappy (architecture) |
| FR-104 | Every name exported by this spec MUST be unique across the package, no generic variant name (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`) MUST be exported, each closed result MUST be named `<UseCase>Result` with variants prefixed by the use case, and no type owned by another spec MUST be redefined (in particular none of `LevelAdvancementStep`, `CharacterEvent`, `AbilityScoreImprovementChoices`, `SingleAbilityIncrease`, `FeatGrant`, `ClassDefinition`, `SubclassDefinition`, `MissingContent`). The subclass domain service is `SubclassSelection`, because `SubclassChoice` is a Bard value in `class-bard.md`. | `doc/rules/architecture.md` §4, §9; `shared-types-and-ownership.md` FR-017, FR-021 | unhappy (name collision) |
| FR-105 | Every collection exposed by the results, commands and value objects of this spec MUST be a defensive copy (mutating the returned slice leaves the value unchanged). | `doc/rules/architecture.md` §4 | unhappy (immutability) |
| FR-117 | Neither `PreviewLevelAdvancement` nor `ChooseSubclass` MUST call another use case or be called by another spec's use case: each composes pure domain services (`Character.LevelUp`, `SubclassSelection`) and its own ports, and records only its own signals, taking attribute values from the domain result. The use cases of `higher-level-start-equipment-and-trinkets.md` compose `SubclassSelection` and `AbilityScoreIncreaser` as domain services. | `shared-types-and-ownership.md` FR-044 | unhappy (nested telemetry) |

### 4.10 Withdrawn ids

| Id | Requirement |
|---|---|
| FR-001 | Withdrawn: moved to character-aggregate.md (FR-004) and derived-stats.md (FR-059) |
| FR-002 | Withdrawn: moved to character-aggregate.md (FR-006) |
| FR-003 | Withdrawn: moved to character-aggregate.md (FR-007) |
| FR-004 | Withdrawn: moved to character-aggregate.md (FR-004, FR-006); table-driven tests live there |
| FR-005 | Withdrawn: moved to character-aggregate.md (FR-008) |
| FR-006 | Withdrawn: moved to derived-stats.md (FR-059); character-aggregate.md (FR-010) applies it to total level |
| FR-007 | Withdrawn: moved to character-aggregate.md (FR-027, FR-028) |
| FR-008 | Withdrawn: dropped; "XP is never spent" had no SRD sentence; the aggregate's `LevelUp` leaves XP unchanged (character-aggregate.md FR-025) |
| FR-039 | Withdrawn: moved to feats-behaviour.md (FR-079); the use case `ApplyAbilityScoreIncrease` is dropped, `AbilityScoreIncreaser` returns its own closed value |
| FR-043 | Withdrawn: moved to content-classes.md (FR-056); a class without a subclass feature cannot decode, so `NoSubclassFeature` is unreachable |
| FR-065 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-065) |
| FR-066 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-066) |
| FR-067 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-067) |
| FR-068 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-068) |
| FR-069 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-069) |
| FR-070 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-070) |
| FR-071 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-071) |
| FR-072 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-072) |
| FR-073 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-073) |
| FR-074 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-074) |
| FR-075 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-075) |
| FR-076 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-076) |
| FR-077 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-077) |
| FR-078 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-078) |
| FR-079 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-079) |
| FR-080 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-080) |
| FR-081 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-081) |
| FR-082 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-082) |
| FR-083 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-083) |
| FR-084 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-084) |
| FR-085 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-085) |
| FR-086 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-086) |
| FR-087 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-087) |
| FR-088 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-088) |
| FR-089 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-089) |
| FR-090 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-090) |
| FR-091 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-091) |
| FR-092 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-092) |
| FR-093 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-093) |
| FR-094 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-094) |
| FR-095 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-095) |
| FR-096 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-096) |
| FR-097 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-097) |
| FR-106 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-106) |
| FR-113 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-113) |
| FR-114 | Withdrawn: dropped before review; the rule is FR-113, what an increase leaves behind is FR-118 |
| FR-115 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-115) |
| FR-116 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-116) |
| FR-118 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-118) |
| FR-119 | Withdrawn: moved to higher-level-start-equipment-and-trinkets.md (FR-119) |

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

Domain (`character`) is pure, immutable, synchronous and silent; its types are those of section 3 (the
list of FR-103). `AbilityScoreIncreaser` calls `Character.ChangeAbilityScore` once per ability with the increase's
`MaximumScore` as the cap, checks the cap for all abilities first so the whole increase is accepted or refused, and
orders events as FR-036. `LevelAdvancementPreviewer` builds a preview from the `Character`, the looked-up
`ClassDefinition`s, `SpellcastingRules` and the result of `Character.LevelUp` (Q-13).

| Invariant | Enforced by | Failure |
|---|---|---|
| Increase non-empty, amounts at least 1, maximum 20 to 30; two-by-one names two distinct abilities | `AbilityScoreIncrease` (FR-026, FR-028) | an invalid-input error (`srderr.ErrInvalidInput`); aggregate error result (FR-029) |
| Each increased score at most the maximum, all or nothing | `AbilityScoreIncreaser` (FR-032, FR-033) | `IncreaseExceedsMaximum` |
| At most one subclass per class, fixed rule order; key name and stored type | `SubclassStateKey`; `SubclassSelection` (FR-045, FR-047); `FeatureStates` (`character-aggregate.md` FR-093) | `SubclassRejected`; an invalid-input error (`srderr.ErrInvalidInput`) |
| Extra Attack counts non-empty, each at least 1; a definition for every later class | `ExtraAttackCombination` (FR-057); `MulticlassTraining` (FR-112) | an invalid-input error (`srderr.ErrInvalidInput`) |
| Preview availability equals `LevelUp` | `LevelAdvancementPreviewer` (FR-012) | n/a |

Rule outcomes are results; invalid arguments are errors matching `srderr.ErrInvalidInput` (architecture §6). The domain does no I/O and returns
no `*srderr.PortError`. References to other contexts are by id and published language only. `character` depends on
`content` per the context map; this spec uses no `d20` or `combat` type. The one same-context direction added is
this spec on `spellcasting.md` (this spec's Q-14). `feats-behaviour.md` (rank 15) needs `AbilityScoreIncrease`,
`AbilityScoreIncreaser` and `IncreaseExceedsMaximum` from this spec (rank 14) and lists it in its "Depends on"
(Q-22). FR-016 holds a plain `PendingFeatGrant`, so this spec needs nothing from `feats-behaviour.md`.

## 6. Interface

Use cases are structs built by `NewX(opts ...Option)` with one method `Run(ctx, cmd)`, an immutable command and their own closed `<UseCase>Result` (interface with an unexported marker method), an optional
`telemetry.Telemetry` (`WithTelemetry`, no-op default) and ports through the constructor; each use case declares the small port interfaces it consumes, and returns `(Result, error)` with the error non-nil only for a port failure or cancelled `ctx`.

| Use case | Command | Result (closed) | Signature |
|---|---|---|---|
| `PreviewLevelAdvancement` | `PreviewLevelAdvancementCommand`: `Character`, `ClassID` | `PreviewLevelAdvancementAvailable(LevelAdvancementGain)`, `PreviewLevelAdvancementUnavailable(LevelAdvancementBlock)`, `PreviewLevelAdvancementContentMissing(missing)` | async (content port) |
| `ChooseSubclass` | `ChooseSubclassCommand`: `Character`, `ClassID`, `SubclassID` | `ChooseSubclassSucceeded(character, event)`, `ChooseSubclassRejected(SubclassRejected)`, `ChooseSubclassContentMissing(missing)` | async (content port) |

`missing` is a defensively copied, non-empty `MissingContent` list, ordered and deduplicated as `character-aggregate.md`
FR-077. A rule rejection is a result; only a port failure is a returned `*srderr.PortError`.

Ports (none new): `content.Source` (takes `ctx`;
a `*srderr.PortError` for `content.Port` for infrastructure failure, which propagates: FR-022, FR-050); `telemetry.Telemetry` (default
the zero-value `telemetry.Noop`, `shared-types-and-ownership.md` FR-036). **Exports:** package `character`
exports every type of FR-103, the two use cases, commands and results; no internal type appears in
an exported signature.

## 7. Observability

Domain code is **silent by design**; the use cases are instrumented (`doc/rules/observability.md` §2). Metrics are the
character context's own (`character-aggregate.md` FR-103, D-21): counter `srd5e.character.operations` and histogram
`srd5e.character.duration`, the latter with the single attribute `srd5e.character.operation`. This spec adds the
values `preview` and `choose_subclass` to that attribute and records nothing else on the histogram (attributes of a
timed operation are fixed before the body runs).

Evaluation: (1) wrong: content missing, a failing port, rule outcomes, bugs. (2) Operator signal: content missing is
span error status with result `content_missing` and no `error.type`; a returned port failure is span error status with
`error.type`; rule outcomes are `srd5e.outcome` `failure` with result `unavailable`/`rejected` and
`srd5e.character.rejection`, span status not error. (3) Diagnosis: operation, outcome, result, rejection or error
type; ids, XP and scores are in the returned values. (4) Counted or timed: calls and outcomes, duration by operation,
lookup time. (5) Reused, not redefined: `srd5e.outcome`, `error.type` (`errors-and-results.md` FR-043),
`srd5e.character.operations`, `srd5e.character.duration`, `srd5e.character.operation`, `srd5e.character.result` and
`srd5e.character.rejection` (`character-aggregate.md`). (6) Monitor: any `error.type` increment, `preview`
`insufficient_experience` (UI offering levels early), `content_missing`, duration regressions. (7) Noise: no log; no
per-level metric.

Values added to `srd5e.character.result`: `preview`: `available` (success), `unavailable`, `content_missing`
(failure); `choose_subclass`: `chosen` (success), `rejected`, `content_missing` (failure). A returned `*srderr.PortError`
has none (FR-100). Values added to `srd5e.character.rejection`: `preview`: `maximum_level`, `insufficient_experience`,
`multiclass_prerequisites_not_met`; `choose_subclass`: `not_in_class`, `below_subclass_level`, `already_chosen`,
`unknown_subclass`. New values go to the catalogue (`observability.md` §5).

| Signal | Type, unit | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.advancement.preview` | span | `srd5e.outcome`, `srd5e.character.result`, `srd5e.character.rejection` (unavailable only) | One level preview including lookups | Error status rate above 0 for sustained periods |
| `srd5e.character.advancement.choose_subclass` | span | `srd5e.outcome`, `srd5e.character.result`, `srd5e.character.rejection` (rejected only) | One subclass choice including its lookup | Error status rate above 0 for sustained periods |
| `srd5e.character.advancement.load_content` | span (child of `preview`, `choose_subclass` and the start use case of `higher-level-start-equipment-and-trinkets.md`) | `srd5e.outcome` | Time in `content.Source` lookups | Alert if lookup time dominates the parent span |
| `srd5e.character.operations` (shared) | counter, `{operation}` | `srd5e.character.operation` (`preview`, `choose_subclass`), `srd5e.outcome`, result (returned results only), rejection (rule outcomes only), `error.type` (returned port failures only) | Throughput and outcome mix | Alert on any `error.type` increment; watch the share of `unavailable` and `rejected` |
| `srd5e.character.duration` (shared) | histogram, `s` | `srd5e.character.operation` only | Latency per use case | p99 regression per operation |

Attribute values are low-cardinality enumerations (no ids, class names, XP or scores). Entries feed
`doc/observability/signals.md`.

## 8. Test plan

Doubles (from `telemetrytest` and `contenttest`): `telemetrytest.Recorder`, in-memory `content.Source` (`contenttest.FakeSource`, also failing with
an error); content fixtures for Cleric, Rogue, Fighter, Wizard, Barbarian, Druid, Monk. No new port or
contract suite. Delegated FR-059 to FR-061, FR-063, FR-064 are tested by the owning specs.

| Id | Level | Notes |
|---|---|---|
| FR-009 to FR-012, FR-024 | unit, application | Table PR through the use case with an in-memory source; agreement with `Character.LevelUp` row by row |
| FR-013, FR-014 | unit | Tables XN and PB (including the two multiclass rows) |
| FR-015 to FR-017 | unit, application | Table PC through `RowFeatures`: repeated grants report the older `FeatureID` with the new `ClassLevel`; subclass row sets `HasSubclassFeature` only; pending subclass at 3 and 4 cleared by a held `ChosenSubclass`; matched by fact and `FeatureID`; Fighting Style not reported; pending subclass does not make the preview unavailable |
| FR-018, FR-019 | unit, application | Table SS; the preview's `ClassLevels` accepted by `AdvanceSpellcasting` |
| FR-020 to FR-022 | application | Empty source; chosen class present and a current class missing, then a held class (no other lookup); failing source |
| FR-023 | unit, application | `LevelAdvancementPreviewer` is pure: equal result for equal inputs, `Character` and command unaltered, collections defensively copied, at both levels |
| FR-025 to FR-030, FR-037, FR-040 | unit | Constructors, invalid inputs, `MustX` agreement with `NewX`, value equality and copying; level 1 character accepted by a direct call; inputs unaltered |
| FR-031 to FR-036, FR-038 | unit | Table AI, one test per row, including event-order and repeat rows |
| FR-041, FR-042, FR-044 to FR-051, FR-107 | unit, application | Tables SC and SP, one test per row, order rows pin FR-047; key name and stored type, one subclass per class, original `Character` unchanged, event and rejection fields; missing class, failing source; level-up and increase unaffected by a pending subclass |
| FR-052, FR-053 | unit, application | Table PR rows for 12/13/14, Fighter `AnyOf`, Monk `AllOf`; table MS through the preview |
| FR-054 to FR-058 | unit | Table EA, one test per row; permuted inputs; input list not altered |
| FR-062, FR-108 to FR-112 | unit, application | Table MG: grants of a new class, none for a held or initial class, no saves or equipment, same for every character; `GrantsFor` rows |
| FR-098 to FR-102 | application | Recording doubles: span names, attribute keys, `success`/`failure` only, result and rejection attributes, no log and no `error.type` on a returned failure, each failure recorded once, `error.type` equal to `srderr.ErrorType`; duration histogram carries `srd5e.character.operation` only; no-op and recording `telemetry.Telemetry` give equal results |
| FR-103, FR-104, FR-117 | architecture | No `telemetry` or adapter imports in domain files; export scan for duplicate and generic names and for `LevelAdvancementStep`; no use case imports another; signal constants appear in `doc/observability/signals.md` |
| FR-105 | unit | Mutation of every exposed collection is a copy (mutating it leaves the value unchanged) |

- **Scenario** (public API only): create a level 1 Wizard with `CreateCharacter`,
  award 6,500 XP, preview and level up to 4, choose Evoker at 3, take an Ability
  Score Improvement via the reported `PendingFeatGrant` and `TakeFeat`, check Hit
  Points after a Constitution increase, preview Fighter as a new class (INT 13,
  STR 13: grants, `SpellSlotsAfter`, Proficiency Bonus), level up, combine Extra
  Attack counts. Cites FR-009, FR-010, FR-014, FR-016, FR-018, FR-031, FR-034,
  FR-041, FR-054, FR-108 and delegated FR-060 to FR-064.
- **Mutation testing:** `character` changed lines, command in
  `doc/rules/testing.md` §8 (`gomu`, at least 80; run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors); the `*_usecase.go` files SHOULD be. Emphasis:
  the cap comparison of FR-032, subclass level of FR-044, order of FR-047,
  maximum of FR-054, existing-versus-new split of FR-108 and FR-109.

## 9. Open questions and decisions

No open questions remain and none is escalated.

Settled (binding, cited by the FR text): Q-7 no partial increase and no clamping, refused whole (FR-032, FR-033;
package decision); Q-9 this spec owns the new-class grants (FR-108 to FR-112, `shared-types-and-ownership.md` FR-048);
Q-10 the subclass lives in `Character.FeatureStates` (`shared-types-and-ownership.md` FR-022); Q-13 the
preview calls `LevelUp` with the fixed-value gain, so FR-012 holds by construction (a pure `CheckLevelUp` may replace
it later), and `SpellSlotsAfter` is `SpellSlotPools` (Spellcasting `SpellSlots` plus optional `PactMagicSlots`), the
value `SlotsFor` returns; package design: one value keeps the two pools of p. 26 apart (`spellcasting.md` FR-034), and
`spellcasting.md` states it (amendment); Q-14 `ExtraAttackCombination` is owned here and the peer edges inside `character` follow
`shared-types-and-ownership.md` FR-016 and `architecture-guardrails.md` FR-091 (no cycle); Q-16 Multiclassing is not a
rule option (`rule-options.md`); Q-17 one level per transition (p. 23; `spellcasting.md` FR-112); Q-18 one subclass
per class, not replaceable (FR-045), a pending subclass blocks nothing (FR-051), revisit only on SRD errata; Q-19 an
unlisted subclass id is `SubclassRejected(UnknownSubclass)` (FR-046); Q-20 a subclass above level 3 is chosen in the
start plan, reported as required, or chosen later (FR-044, FR-107); Q-22 `feats-behaviour.md` is rank 15 (`shared-types-and-ownership.md` 4.11) and
lists this spec, so the types FR-025 to FR-038 stay here; Q-1 to Q-6, Q-8, Q-15 resolved by
`shared-types-and-ownership.md` FR-016, FR-019, FR-020, FR-041, FR-044, FR-048. Q-11, Q-12 and Q-21 moved with the
start, equipment and Trinket requirements.

## 10. Changelog impact

`CHANGELOG.md`, `Unreleased` / **Added**: "`character`: `PreviewLevelAdvancement` with pending choices, the grants of
a first level in a new class and multiclass Spell Slots; `AbilityScoreIncrease` and `AbilityScoreIncreaser`;
`ChooseSubclass` with `SubclassChosen`, `SubclassRejected`, `ChosenSubclass` and `SubclassSelection.IsPending`;
`MulticlassTraining`; `ExtraAttackCombination`; telemetry `srd5e.character.advancement.*` spans and the values
`preview` and `choose_subclass` of `srd5e.character.operation`." Not **BREAKING** (first release of these types). The
follow-up edit of Q-9 changes other specs and may be **BREAKING** when applied to released code; it is not part of
this change.
