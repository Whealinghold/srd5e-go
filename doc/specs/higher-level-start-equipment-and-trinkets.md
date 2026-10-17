# Starting at higher levels, higher-level equipment and Trinkets

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `derived-stats.md`, `character-aggregate.md`, `level-advancement-and-multiclassing.md`, `ability-scores.md`, `content-classes.md`, `content-species-backgrounds-feats.md`, `content-source-port.md`, `content-schema-and-decoder.md`, `content-equipment.md`, `content-magic-items.md`, `dice-and-rolls.md`, `errors-and-results.md`, `observability-ports.md` |

## 1. Purpose and scope

Split out of `level-advancement-and-multiclassing.md` (size, FR-062 of
`shared-types-and-ownership.md`); each requirement keeps its id and is a
withdrawn row there. The parts of "Starting at Higher Levels" (p. 24) and
"Trinkets" (pp. 26-27) that no other spec owns: **building a character at a
starting level above 1** (a plan of level-up steps folded over a level 1
character), **Starting Equipment at Higher Levels** (the p. 24 table of extra gold
and magic items by rarity) and the **Trinket roll**.

This spec is the topic-specific owner (`shared-types-and-ownership.md` FR-048) of
starting level and Experience Points, the plan, the higher-level equipment table
and the Trinket roll, and defines only the types marked "defined here" in
section 3.

**Ownership of the neighbours.** The level-up rules, Hit Point gain and the Hit
Dice pool: `character-aggregate.md` FR-024 to FR-050. Subclass choice
(`SubclassSelection`, `SubclassRejected`, `IsPending`), the ability score increase
(`AbilityScoreIncrease`, `AbilityScoreIncreaser`) and pending feat grants:
`level-advancement-and-multiclassing.md`. Feat choices and `TakeFeat`:
`feats-behaviour.md`. Origin choices, ability score generation, `CreateCharacter`
and the command inputs `HigherLevelEquipmentChoice`, `TrinketChoice`,
`TrinketRoll`: `character-creation.md`, which composes this spec's domain services
(`HigherLevelStart`, `HigherLevelEquipment`, `TrinketTable`, `TotalGold`) because a
use case never calls another use case (`shared-types-and-ownership.md` FR-044).

**Out of scope:** normal starting equipment, buying equipment, the magic item
catalogue; the text of the 100 Trinkets (host-supplied, Q-11); the GM option
Bonus Feats at Level 20 (`rule-options.md`); ability score generation.

**Supersedes (`shared-types-and-ownership.md` FR-048).** Owned here instead of the
older text of `character-creation.md`: FR-071 to FR-078 as FR-065 to FR-080,
FR-106, FR-113; FR-079 to FR-086 (not `HigherLevelEquipmentChoice`) as FR-081 to
FR-089, FR-116; FR-087 to FR-091 (not `TrinketChoice`, `TrinketRoll`) as FR-090 to
FR-097.

## 2. Source

Printed page numbers equal PDF page numbers.

- SRD 5.2.1, "Create Your Character" ("Starting at a Higher Level"), p. 19;
  "Level Advancement" and the Character Advancement table, p. 23
- "Starting at Higher Levels", "Creating Your Character", "Starting Equipment" and
  the Starting Equipment at Higher Levels table, p. 24; "Multiclassing",
  "Prerequisites", pp. 24-25
- "Trinkets", pp. 26-27; "Ability Score Improvement", p. 87; "Level 3: Rogue
  Subclass", p. 62; "Magic Item Rarities and Values", p. 206 (via
  `content-magic-items.md` FR-014)
- Package design: `doc/rules/architecture.md`, `doc/rules/observability.md`,
  `doc/rules/testing.md`, `shared-types-and-ownership.md`, `errors-and-results.md`.

## 3. Vocabulary

Used by reference: the types of `derived-stats.md` (`CharacterLevel`, `ClassLevels`, `HitPointGain`), `character-aggregate.md`,
`level-advancement-and-multiclassing.md`, `ability-scores.md` (`AbilityScores`), `content-classes.md`,
`content-species-backgrounds-feats.md` (the looked-up species and background), `content-schema-and-decoder.md` and
`content-equipment.md` (`Cost`, `Coin.gp`).

| SRD term | Type (defined here) | Notes |
|---|---|---|
| Starting at Higher Levels | `HigherLevelStart` domain service; use case `StartAtHigherLevel`; `HigherLevelStep` (`AdvancementClass`, `HitPointGain`, optional `SubclassID`, optional `AbilityScoreIncrease`); `StartingLevelStep` (command step: `ClassID`, `HitPointChoice`, optional `SubclassID`, optional `AbilityScoreIncrease`); `ConsumedFeatGrant(ClassID, ClassLevel, FeatureID)`; closed `HigherLevelStartRejection` | p. 24. |
| Starting Equipment at Higher Levels | closed `HigherLevelEquipment`: `NoHigherLevelEquipment`, `HigherLevelEquipmentAllowance`; `RolledGold` | p. 24, a guide for the GM. Magic items are keyed by `ItemRarity` (`content-magic-items.md` FR-014); only Common, Uncommon, Rare and Very Rare appear (FR-084). |
| Trinkets | `TrinketTable` (100 entries), `Trinket` (description text only); `TrinketRoll` is `character-creation.md`'s (a plain int 1 to 100 here) | p. 26. A roll of 100 is the table's `00`. Host-supplied pure values; no `ContentKind`, decoder or `content.Source` produces them (FR-093, Q-11). |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids (kept from `level-advancement-and-multiclassing.md`). Never renumber;
new requirements get new ids. Ids absent here stayed with `level-advancement-and-multiclassing.md` or were never used by this spec; withdrawn ones are in 4.5.

### 4.1 Starting at Higher Levels

"Creating a higher-level character uses the same character-creation steps ...
and the rules for advancing beyond level 1" (p. 24). `HigherLevelStart` (domain)
folds `LevelUp` over a plan of steps after a level 1 character built by
`NewCharacter` (starting XP the minimum for the level, FR-065).
`StartAtHigherLevel` wraps it; `CreateCharacter` composes the same domain
service (`character-creation.md` FR-072 to FR-078). Neither calls the other or
any use case (`shared-types-and-ownership.md` FR-044; FR-120). This spec owns the
plan, fold, starting XP, order of validation and rolls, and rejections (FR-115);
`character-creation.md` owns origin choices, ability score generation and
`CreateCharacter`. A step's feat grant is consumed, not taken (FR-118). A step
may carry the subclass chosen at the class level it reaches, and a character is not
complete at level 3 or higher without one (FR-071, FR-073; Q-20); it can also be
chosen later (`level-advancement-and-multiclassing.md` FR-044, FR-107).

`StartAtHigherLevelCommand` holds `CharacterID`, `ClassID`, `SpeciesID`,
`BackgroundID`, `AbilityScores`, the starting `CharacterLevel` and the ordered
`StartingLevelStep`s, one per level from 2 to the starting level, each turned into
a `HigherLevelStep` (FR-119). `HigherLevelStartRejection`:
`HigherLevelStartPlanLengthMismatch(ExpectedSteps, ActualSteps)`,
`HigherLevelStartCreationInvalid(problems)`,
`HigherLevelStartLevelUpRejected(AtLevel, LevelUpRejection)`,
`HigherLevelStartSubclassRejected(AtLevel, SubclassRejected)`,
`HigherLevelStartIncreaseRejected(AtLevel, AbilityScoreIncreaseRejection)`,
`HigherLevelStartIncreaseNotGranted(AtLevel)`,
`HigherLevelStartSubclassRequired(classes)`; `AtLevel` is the total level the step
would have reached.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-065 | A character started at level N MUST have Experience Points equal to the minimum XP for level N (0 for level 1; table SL), total level N, and class levels that sum to N. | SRD 5.2.1, "Starting at Higher Levels", "Creating Your Character" ("You begin with the minimum amount of XP required to reach your starting level. For example, if the GM starts you at level 10, you have 64,000 XP"), p. 24 | happy |
| FR-066 | The number of steps MUST equal the starting level minus 1; otherwise the result MUST be `HigherLevelStartPlanLengthMismatch` carrying the expected and the actual number, with no character built and no dice rolled (also when the plan holds `RollHitDie` steps, table SH). | SRD 5.2.1, "Starting at Higher Levels", p. 24 | unhappy (invalid input) |
| FR-067 | A starting level of 1 with no steps MUST give exactly the `Character` and the one `CharacterCreated` event that `NewCharacter` gives for the same arguments and 0 Experience Points. | SRD 5.2.1, "Create Your Character" ("A level 1 character has 0 XP"), p. 19 | unhappy (empty and minimal) |
| FR-068 | Steps MUST be applied in order, each with the rules of `LevelUp` (prerequisites for a new class, Hit Point gain, Proficiency Bonus, Hit Dice), so the Hit Point maximum, Hit Dice pool and class levels equal what the same sequence of `LevelUp`s gives (table SH). | SRD 5.2.1, "Starting at Higher Levels" ("the rules for advancing beyond level 1"), p. 24; `character-aggregate.md` FR-025 to FR-039 | happy |
| FR-069 | The first step that `LevelUp` rejects MUST stop the build with `HigherLevelStartLevelUpRejected(AtLevel, rejection)` and no later step MUST be applied; the only rejection a valid plan can meet is `LevelUpPrerequisitesNotMet` (table SH). | SRD 5.2.1, "Multiclassing", "Prerequisites", p. 24 | unhappy (state error) |
| FR-070 | `LevelUpInsufficientExperience` and `LevelUpMaximumLevel` MUST be unreachable for a plan of the right length: for every starting level 2 to 20, applying all steps with the starting Experience Points never meets either. | SRD 5.2.1, Character Advancement table, p. 23; "Starting at Higher Levels", p. 24 | unhappy (boundary) |
| FR-071 | A step's subclass choice MUST be applied right after its level is gained, with `SubclassSelection.choose` and the class level reached (`level-advancement-and-multiclassing.md` FR-041, FR-042, FR-044 to FR-047), and a refusal MUST stop the build with `HigherLevelStartSubclassRejected(AtLevel, reason)`. | SRD 5.2.1, "Level 3: Rogue Subclass", p. 62; "Create Your Character" ("Starting at a Higher Level"), p. 19 | unhappy (state error) |
| FR-072 | A step's ability score increase MUST be applied after its level and subclass choice, with `AbilityScoreIncreaser` (`level-advancement-and-multiclassing.md` FR-031 to FR-038) and only when FR-113 allows it, and a refusal by the increaser MUST stop the build with `HigherLevelStartIncreaseRejected(AtLevel, rejection)`. A later step sees the scores the earlier increases produced (table SH). An applied increase is reported as a consumed grant (FR-118). | SRD 5.2.1, "Gaining a Level" step 5, p. 23; "Ability Score Improvement", p. 87 | unhappy (rule interaction) |
| FR-073 | After the last step, every class whose level has reached its `SubclassLevel` and that has no `ChosenSubclass` MUST give `HigherLevelStartSubclassRequired` listing those `ClassID`s in the order they first appear in the `ClassLevels`. | SRD 5.2.1, "Create Your Character" ("Starting at a Higher Level": "If you start at level 3 or higher, write your chosen subclass on your character sheet"), p. 19 | unhappy (state error) |
| FR-074 | Every rule rejection (FR-066, FR-077, FR-078, FR-069, FR-071 to FR-073, FR-113) and the content-port failure half of FR-079 MUST be found before any die is rolled: the use case first validates the plan by a dry run of `HigherLevelStart` with `FixedHitPointGain` for every step (no rejection depends on the gain), then rolls exactly one die of the class's Hit Point Die size through the `dice.Roller` for each `RollHitDie` step, in step order, and uses the result as `character-aggregate.md` FR-033 does; for `TakeFixedValue` it MUST NOT call the roller. The number of rolls equals the number of rolled steps for an accepted plan and is 0 for a plan rejected by a rule or by missing content (table SH). A roller that breaks its contract (FR-079) can only be seen after that roll: the build then stops with the returned error after the rolls made so far, and no character is built. | SRD 5.2.1, "Gaining a Level" step 2, p. 23; `character-aggregate.md` FR-063, FR-064; package design (`character-creation.md` FR-004: a doomed plan rolls nothing) | happy / unhappy (determinism) |
| FR-075 | With a roller that returns 1 for every die and one that returns each die's highest face, the Hit Point maximum MUST be the clamped minimum and the maximum respectively (table SH). | SRD 5.2.1, "Gaining a Level" step 2 ("minimum of 1"), p. 23 | unhappy (determinism) |
| FR-076 | Events MUST be ordered: `CharacterCreated`, then for each step `LevelGained`, then `SubclassChosen` if any, then the step's `AbilityScoreChanged` events and at most one `HitPointMaximumChanged`. | `character-aggregate.md` FR-058; `doc/rules/architecture.md` §3 | unhappy (determinism) |
| FR-077 | The use case MUST look up the initial class, the species, the background and every distinct class named in the steps through the content port, and when any lookup is not found MUST return `StartAtHigherLevelContentMissing` listing every `MissingContent` (ordered as in `character-aggregate.md` FR-077, without duplicates), with no dice rolled and no character built. | `character-aggregate.md` FR-061, FR-077 | unhappy (missing content) |
| FR-078 | Starting scores that `NewCharacter` refuses (a score above 20) MUST give `HigherLevelStartCreationInvalid` carrying the aggregate's problems, with no dice rolled. | SRD 5.2.1, "Ability Scores", p. 5; `character-aggregate.md` FR-016, FR-082 | unhappy (invalid input) |
| FR-079 | A `*srderr.PortError` from the content port (`content.Port`), and one for `dice.Port` from the roller, MUST be returned unchanged as the error of the use case, with a zero result. A roller that returns a value below 1 or above the die size MUST make the use case return a `*srderr.PortError` for `dice.Port` that matches `srderr.ErrContractViolation` as soon as that roll is made (so earlier rolls were made, FR-074), and no character MUST be built. | `character-aggregate.md` FR-062, FR-078, FR-079; `errors-and-results.md` FR-036 | unhappy (collaborator failure) |
| FR-080 | `StartAtHigherLevel` MUST NOT alter its command or its steps, MUST copy the steps into a defensively copied list, and with the same command, content and a fixed roller MUST return an equal result on every call. | `doc/rules/architecture.md` §3, §4; `doc/rules/testing.md` §4 | unhappy (immutability, determinism) |
| FR-106 | When several failures hold for one `StartAtHigherLevel` call, the one reported MUST be chosen in this fixed order: plan length mismatch (before any lookup), content missing, creation invalid, the first failing step in order (a level-up rejection, then the subclass refusal, then the increase not granted, then the increase refused, within the step), subclass required (table SH, order rows). | `character-aggregate.md` D-4 (determinism) | unhappy (rule interaction) |
| FR-113 | A step's ability score increase MUST correspond to a feat grant that the step's class level opens: `ClassDefinition.FeatGrantsAt(ClassLevelReached)` (`content-classes.md` FR-049) holds `General` and the increase's `MaximumScore` is at most 20, or holds `Epic Boon` and it is at most 30. The class level is the level reached in the step's class, not the total level, and a repeated grant counts at every row that names it (Fighter 6, 14; Wizard 8, 12, 16; Rogue 10). Otherwise the build MUST stop with `HigherLevelStartIncreaseNotGranted(AtLevel)`. A step carries at most one increase. An increase at a level with no such grant is refused (a level 2 Fighter step with +2 Strength, or a level 7 step). | SRD 5.2.1, "Ability Score Improvement" (Prerequisite: Level 4+), p. 87; class Features tables, Fighter p. 47, Wizard p. 77, Rogue p. 62; package design | unhappy (rule interaction) |
| FR-115 | Supersession: FR-065 to FR-080 and FR-081 to FR-097 replace the earlier text of `character-creation.md` FR-071 to FR-078, FR-079 to FR-086 and FR-087 to FR-091 for these topics. `character-creation.md` points here instead of redefining them (its sections 4.8 and 4.9), `LevelAdvancementStep` and the earlier `StandardEquipmentOnly` / `StartingEquipmentAtHigherLevels` / numeric `Trinket` are not defined anywhere, and each exported name has one owner (FR-123). | `shared-types-and-ownership.md` FR-016, FR-017, FR-021; `doc/rules/architecture.md` §2 | unhappy (name collision) |
| FR-118 | A start step whose increase FR-113 allowed and the increaser applied MUST emit no `FeatTaken` and change no `CharacterFeats` (none exists here); `HigherLevelStart` and `StartAtHigherLevelStarted` MUST instead report one `ConsumedFeatGrant(ClassID, ClassLevel, FeatureID)` per applied increase, in step order, where `ClassLevel` is the class level the step reached and `FeatureID` the resolved entry's id as in `level-advancement-and-multiclassing.md` FR-016, and none for a step without an increase or for a rejected plan. A host MUST keep the reported grants beside the character and, when it lists the grants still open for the levels already reached, MUST treat a grant as used when it is reported here or counted by `CharacterFeats.TakenFrom(source)` (`feats-behaviour.md` FR-062), so the same Ability Score Improvement is never offered twice. | package design; `feats-behaviour.md` FR-062; SRD 5.2.1, "Ability Score Improvement" (Repeatable), p. 87 | unhappy (state error) |
| FR-119 | The use case MUST turn each `StartingLevelStep` into a `HigherLevelStep` by taking the `AdvancementClass` from the looked-up `ClassDefinition` of its `ClassID` and the `HitPointGain` from its `HitPointChoice` (`FixedHitPointGain` for `TakeFixedValue`, a `RolledHitPointGain` of the rolled value for `RollHitDie`, FR-074), and copying the optional subclass and increase unchanged; it MUST add no other rule. | `character-aggregate.md` FR-033, FR-034; package design | happy |

Table SL (FR-065; pp. 23-24), starting level gives XP: 1: 0; 2: 300; 3: 900 (the
SRD recommends level 3 for seasoned players); 5: 6,500; 10: 64,000 (the SRD
example); 17: 225,000; 20: 355,000.

Table SH (FR-066 to FR-079, FR-106, FR-113, FR-118, FR-119). Single-class Fighter
unless a row says otherwise; Hit Point Die d10 (fixed value 6), Constitution 14
(+2), so the level 1 maximum is 12 and a fixed level gives 8
(`character-aggregate.md` sections 4.3, 4.5). Subclass level 3, Champion as the
Fighter subclass (`class-fighter.md`); Fighter Ability Score Improvement at class
levels 4, 6, 8, 12, 14, 16 and Epic Boon at 19. Every plan with a starting level
of 3 or more includes Champion at level 3 unless its row says otherwise. "Rolls"
is the number of dice the roller is asked for.

| Label | Starting level, plan | Expected | Rolls | Path |
|---|---|---|---|---|
| level 3, fixed | 3: Fighter fixed, Fighter fixed, Champion at level 3 | maximum 28 (12 + 8 + 8); XP 900 | 0 | happy |
| level 3, rolled, all minimum | 3: Fighter rolled x2, Champion | maximum 18 (12 + 3 + 3) with a roller returning 1 | 2 | unhappy (determinism) |
| level 3, rolled, all maximum | 3: Fighter rolled x2, Champion | maximum 36 (12 + 12 + 12) with a roller returning 10 | 2 | unhappy (determinism) |
| level 3, mixed | 3: Fighter rolled, Fighter fixed, Champion | one roll, then no roll | 1 | happy |
| multiclass second class | 2: Druid fixed (Barbarian first class; STR 13, WIS 13) | class levels Barbarian 1, Druid 1; Hit Dice d12 x 1 and d8 x 1 | 0 | happy |
| multiclass prerequisite unmet | 2: Druid fixed (Barbarian first class; STR 12, WIS 13) | `HigherLevelStartLevelUpRejected(AtLevel 2, LevelUpPrerequisitesNotMet [Strength])` | 0 | unhappy (state error) |
| ability increase at level 4 | 4: Fighter fixed x3, Champion at 3, +2 Strength on the level 4 step (STR 15) | STR 17; events end with `AbilityScoreChanged(Strength)` | 0 | happy |
| Constitution increase, retroactive Hit Points | 4: Fighter fixed x3, Champion at 3, +2 Constitution on the level 4 step (CON 14, modifier +2 to +3) | maximum 40 (12 + 8 + 8 + 8 = 36, then +1 for each of 4 levels); events end with `AbilityScoreChanged(Constitution)`, `HitPointMaximumChanged` | 0 | happy |
| later step sees earlier increase | 8: +2 Strength at 4 and +2 Strength at 8 (STR 15) | STR 19; `ConsumedGrants` (Fighter, 4, e) and (Fighter, 8, e) where `e` is the same level 4 Ability Score Improvement entry id | 0 | unhappy (rule interaction) |
| increase refused at level 8 | 8: +2 Strength at 4 and at 8 (STR 17 start) | step 4 gives 19; step 8 gives `HigherLevelStartIncreaseRejected(AtLevel 8, IncreaseExceedsMaximum [Strength], 20)` | 0 | unhappy (boundary) |
| increase at a level with no grant | 2: +2 Strength on the level 2 step | `HigherLevelStartIncreaseNotGranted(AtLevel 2)` | 0 | unhappy (rule interaction) |
| increase between grants | 5: +2 Strength on the level 5 step (Fighter grants at 4 and 6) | `HigherLevelStartIncreaseNotGranted(AtLevel 5)` | 0 | unhappy (rule interaction) |
| Epic Boon size at an ASI level | 4: `EpicBoon(Strength)` on the level 4 step | `HigherLevelStartIncreaseNotGranted(AtLevel 4)` (maximum 30 over the `General` grant) | 0 | unhappy (boundary) |
| subclass below its level | 2: Champion chosen at step 2 | `HigherLevelStartSubclassRejected(AtLevel 2, BelowSubclassLevel)` | 0 | unhappy (boundary) |
| subclass missing at 3 | 3: no subclass choice | `HigherLevelStartSubclassRequired([Fighter])` | 0 | unhappy (state error) |
| subclass not required at 2 | 2: Fighter fixed | `StartAtHigherLevelStarted` | 0 | unhappy (boundary) |
| plan too short | 3: one step | `HigherLevelStartPlanLengthMismatch(expected 2, actual 1)` | 0 | unhappy (invalid input) |
| plan too long | 2: two steps | `HigherLevelStartPlanLengthMismatch(expected 1, actual 2)` | 0 | unhappy (invalid input) |
| plan too short, with rolled steps | 3: one `RollHitDie` step | `HigherLevelStartPlanLengthMismatch(expected 2, actual 1)` | 0 | unhappy (determinism) |
| level 1, no steps | 1: no steps | the same as `NewCharacter` | 0 | unhappy (empty and minimal) |
| highest level | 20: nineteen Fighter fixed steps with a subclass at 3 | total level 20, XP 355,000 | 0 | unhappy (boundary) |
| starting score above 20 | 3: STR 21, rolled steps | `HigherLevelStartCreationInvalid([StartingScoreAboveMaximum])` | 0 | unhappy (invalid input) |
| class missing | 2: Druid step, Druid not in content | `StartAtHigherLevelContentMissing([Druid])`, nothing rolled | 0 | unhappy (missing content) |
| class missing, with rolled steps | 3: Fighter rolled, Druid rolled, Druid not in content | `StartAtHigherLevelContentMissing([Druid])` | 0 | unhappy (determinism) |
| mid-plan rejection after rolled steps | 4: Fighter rolled, Druid rolled with prerequisites unmet, Fighter rolled | `HigherLevelStartLevelUpRejected(AtLevel 3, ...)` | 0 | unhappy (determinism) |
| subclass required, with rolled steps | 3: Fighter rolled x2, no subclass | `HigherLevelStartSubclassRequired([Fighter])` | 0 | unhappy (determinism) |
| accepted plan, rolled steps | 4: Fighter rolled x3, Champion at 3 | rolls in step order | 3 | happy |
| order: plan length before content | 3: one step, Druid not in content | `HigherLevelStartPlanLengthMismatch`, no lookup made | 0 | unhappy (rule interaction) |
| order: content before creation | 3: STR 21, species not in content | `StartAtHigherLevelContentMissing([species])` | 0 | unhappy (rule interaction) |
| order: step before subclass required | 3: Druid step at 2 with prerequisites unmet, no subclass | `HigherLevelStartLevelUpRejected(AtLevel 2, ...)` | 0 | unhappy (rule interaction) |
| order: not granted before refused | 8: +2 Strength at level 5, STR 20 | `HigherLevelStartIncreaseNotGranted(AtLevel 5)` | 0 | unhappy (rule interaction) |
| repeated grant at level 6 | 6: Fighter fixed x5, Champion at 3, +2 Strength on the level 6 step (STR 15) | STR 17; `ConsumedGrants` one entry (Fighter, 6, the level 4 entry's id) | 0 | happy |
| repeated grant at level 8 accepted | 8: Champion at 3, +2 Strength on the level 8 step only (STR 15) | STR 17; `ConsumedGrants` one entry (Fighter, 8, the level 4 entry's id) | 0 | happy |
| subclass-feature row has no grant | 7: +2 Strength on the level 7 step | `HigherLevelStartIncreaseNotGranted(AtLevel 7)` | 0 | unhappy (rule interaction) |
| no increase, nothing consumed | 4: Fighter fixed x3, Champion at 3 | `ConsumedGrants` empty; no `FeatTaken` event | 0 | unhappy (empty and minimal) |
| rejected plan reports no consumed grant | 8: +2 Strength at 4 and at 8 (STR 17 start) | `StartAtHigherLevelRejected(HigherLevelStartIncreaseRejected(...))`, which carries no `ConsumedGrants` | 0 | unhappy (state error) |
| `StartingLevelStep` conversion | 3: two `StartingLevelStep`s (Fighter, `TakeFixedValue`) and Champion at 3, through the use case | the same character and events as the equivalent `HigherLevelStep`s through `HigherLevelStart` | 0 | happy |

### 4.2 Starting Equipment at Higher Levels

The table (p. 24) is "a guide for the GM": by starting level, extra gold (a
fixed amount plus a rolled amount) and magic items by rarity, in addition to the
"normal starting equipment". This spec models the numbers only; whether
`CreateCharacter` asks for it is `character-creation.md`'s
`HigherLevelEquipmentChoice`. `HigherLevelEquipment.ForStartingLevel(CharacterLevel)`
returns `NoHigherLevelEquipment` for level 1 and a `HigherLevelEquipmentAllowance`
otherwise: `FixedGold` (`Cost` in GP), `RolledGold` (empty or one `RolledGold`: a
`dice.Expression` and a multiplier `Cost` in GP) and `MagicItems` (defensively copied
`ItemRarity` to count, non-zero counts only; `content-magic-items.md` FR-014).
`RollHigherLevelStartingMoney` rolls the rolled part through the `dice.Roller`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-081 | `ForStartingLevel(1)` MUST return `NoHigherLevelEquipment`, and every level 2 to 20 MUST return an allowance (table SE, tier boundaries 2, 4, 5, 10, 11, 16, 17, 20). | SRD 5.2.1, Starting Equipment at Higher Levels table (rows start at 2-4), p. 24 | unhappy (boundary) |
| FR-082 | The fixed gold MUST be 0 GP for starting levels 2 to 4, 500 GP for 5 to 10, 5,000 GP for 11 to 16 and 20,000 GP for 17 to 20. | SRD 5.2.1, Starting Equipment at Higher Levels table, p. 24 | happy |
| FR-083 | The rolled gold MUST be absent for starting levels 2 to 4, and otherwise one roll of `1d10` multiplied by 25 GP for levels 5 to 10 and by 250 GP for levels 11 to 16 and 17 to 20. | SRD 5.2.1, Starting Equipment at Higher Levels table, p. 24 | happy |
| FR-084 | The magic item counts, keyed by `ItemRarity` (`content-magic-items.md` FR-014), MUST be as in table SE: 1 Common for levels 2 to 4; 1 Common and 1 Uncommon for 5 to 10; 2 Common, 3 Uncommon and 1 Rare for 11 to 16; 2 Common, 4 Uncommon, 3 Rare and 1 Very Rare for 17 to 20. A rarity not listed (Legendary, Artifact) MUST be absent from the map. | SRD 5.2.1, Starting Equipment at Higher Levels table, p. 24 | happy |
| FR-085 | `RollHigherLevelStartingMoney` MUST, for a level with rolled gold, roll exactly one `1d10` through the `dice.Roller` and return the total that `TotalGold` (FR-116) gives for the fixed gold, the roll and the multiplier in GP (table SM, minimum and maximum of every tier). | SRD 5.2.1, Starting Equipment at Higher Levels table ("500 GP plus 1d10 x 25 GP"), p. 24 | happy |
| FR-086 | For starting levels 2 to 4 `RollHigherLevelStartingMoney` MUST return 0 GP and MUST NOT call the roller; for level 1 it MUST return `RollHigherLevelStartingMoneyNotApplicable`. | SRD 5.2.1, Starting Equipment at Higher Levels table, p. 24 | unhappy (empty and minimal) |
| FR-087 | A roller that returns a value below 1 or above 10 for the `1d10` MUST make the use case returns a `*srderr.PortError` for `dice.Port` with reason `InvalidResult`; a `*srderr.PortError` for `dice.Port` from the roller MUST propagate unchanged; no money is returned in either case. | `errors-and-results.md` FR-036; `doc/rules/testing.md` §4 | unhappy (collaborator failure) |
| FR-088 | `RollHigherLevelStartingMoneyRolled` MUST carry the allowance, the roll and the total `Cost` (GP), so a consumer can show the breakdown, and no display text. | `doc/rules/architecture.md` §6 | happy |
| FR-089 | The allowance, the use case result and `RolledGold` MUST be immutable and compare by value, and the maps and lists they expose MUST be defensively copied (mutating the returned slice leaves the value unchanged). | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |
| FR-116 | `HigherLevelEquipmentAllowance.TotalGold(roll *int) (int, error)` MUST be a pure domain function (no `ctx`): for an allowance with rolled gold it MUST return the fixed gold plus `*roll` times the multiplier in GP, and an error matching `srderr.ErrInvalidInput` (a typed error carrying the argument name and invalid value) when `roll` is nil or outside 1 to the die's faces; for an allowance with no rolled gold it MUST return the fixed gold when `roll` is nil and an error matching `srderr.ErrInvalidInput` when a roll is given. `RollHigherLevelStartingMoney` MUST obtain its total through this function (FR-085), so a host or `CreateCharacter` can roll the `1d10` through its own `dice.Roller` and call the function without calling this use case (FR-120). Rows of table SM, plus the invalid-input rows. | SRD 5.2.1, Starting Equipment at Higher Levels table, p. 24; `shared-types-and-ownership.md` FR-044; `errors-and-results.md` FR-027 | happy / unhappy (invalid input) |

Table SE (FR-081 to FR-084; p. 24; normal starting equipment is in every row and
not modelled), starting level: fixed gold; rolled gold; magic items. 1:
`NoHigherLevelEquipment`. 2 and 4 (tier edges): 0 GP; none; Common 1. 5 and 10:
500 GP; 1d10 x 25 GP; Common 1, Uncommon 1. 11 and 16: 5,000 GP; 1d10 x 250 GP;
Common 2, Uncommon 3, Rare 1. 17 and 20: 20,000 GP; 1d10 x 250 GP; Common 2,
Uncommon 4, Rare 3, Very Rare 1.

Table SM (FR-085, FR-086, FR-116; p. 24), (starting level, roll of 1d10) gives
total: (3, none rolled) 0 GP; (5, 1) 525 GP; (10, 10) 750 GP; (11, 1) 5,250 GP;
(16, 10) 7,500 GP; (17, 1) 20,250 GP; (20, 10) 22,500 GP; (18, 5) 21,250 GP; (1,
none) `RollHigherLevelStartingMoneyNotApplicable`; (5, 0) and (5, 11)
`srderr.ErrContractViolation` (a `*srderr.PortError`).

### 4.3 Trinkets

"When you make your character, you can roll once on the Trinkets table to gain a
Tiny trinket, a simple item lightly touched by mystery. The GM might also use
this table" (p. 26). The table is a 1d100 table of 100 text entries; `00` is a
roll of 100. A `Trinket` is its description text and nothing else (the SRD gives
it no weight, cost or effect).

**Where the table comes from (Q-11).** The entries are game content under
`NOTICE.md`, and `content` has no `ContentKind` for them
(`content-schema-and-decoder.md` FR-142), so no decoder can produce them.
`TrinketTable` and `Trinket` are pure values the host supplies: nothing in the
package decodes, bundles or embeds the entries, and no `content.Source` is
involved. `RollTrinket` takes a `TrinketTable` in its command;
`character-creation.md` records only the number (`TrinketRoll`,
`character-creation.md` FR-091). A host builds the table from its own copy of
SRD 5.2.1 and carries the attribution for that copy; the tests use an
independent transcription under `testdata/srd/` (FR-093).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-090 | `NewTrinketTable(entries)` MUST return `(TrinketTable, error)`: the table MUST hold exactly 100 entries, one for each d100 result 01 to 99 and 00 (100), in that order, and any other number of entries returns an error matching `srderr.ErrInvalidInput` (a typed error carrying the count); `MustTrinketTable` panics with that error and is for literals and tests only. | SRD 5.2.1, "Trinkets" (1d100 table, 01 to 00), pp. 26-27; `errors-and-results.md` FR-025, FR-026 | unhappy (boundary, invalid input) |
| FR-091 | `TrinketTable.EntryFor(roll)` MUST return the entry numbered `roll` for a roll from 1 to 99 and the entry printed as `00` for a roll of 100; a roll of 0 or 101 MUST return an error matching `srderr.ErrInvalidInput` (table TR). | SRD 5.2.1, "Trinkets", pp. 26-27 | unhappy (boundary) |
| FR-092 | A `Trinket` description MUST be non-blank and kept verbatim, including entries with a quoted label (93), a choice left to the reader (47, "your choice"); the domain MUST NOT rewrite it. | SRD 5.2.1, "Trinkets", p. 27 | unhappy (invalid input) |
| FR-093 | `TrinketTable` MUST be a host-supplied pure value: no `ContentKind`, decoder or `content.Source` produces it, and no file under `character/` embeds the 100 entries. The tests MUST build a table from an independent transcription of SRD 5.2.1, "Trinkets", pp. 26-27, kept under `testdata/srd/` (not from production data), and check that `EntryFor` returns the transcribed text for every roll 1 to 100 and that the table has exactly 100 entries. | SRD 5.2.1, "Trinkets", pp. 26-27; `shared-types-and-ownership.md` FR-033 | happy |
| FR-094 | `RollTrinket` MUST roll exactly one `1d100` through the `dice.Roller` and return `RollTrinketRolled` holding the roll (1 to 100) and the matching `Trinket`. It MUST NOT require a `Character` and MUST keep no record of earlier rolls ("you can roll once" is the player's own limit, not state this package keeps). | SRD 5.2.1, "Trinkets" ("you can roll once"; "The GM might also use this table"), p. 26 | happy |
| FR-095 | With a fixed roller returning 1, 34, 35, 72, 73, 99 and 100, `RollTrinket` MUST return the entries in table TR for those rolls, so the page break between 34 and 35 and the `00` entry are covered. | SRD 5.2.1, "Trinkets", pp. 26-27 | unhappy (boundary) |
| FR-096 | A roller that returns 0 or a value above 100 for the `1d100` MUST make `RollTrinket` returns a `*srderr.PortError` for `dice.Port` with reason `InvalidResult`, and a `*srderr.PortError` for `dice.Port` from the roller MUST propagate unchanged; no result is built in either case. | `errors-and-results.md` FR-036; `doc/rules/testing.md` §4 | unhappy (collaborator failure) |
| FR-097 | `RollTrinket` MUST NOT alter its command or table and MUST return an equal result for the same table and roll on every call. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (immutability, determinism) |

Table TR (FR-091, FR-095). Source: pp. 26-27, text verbatim; the full 100 rows
are the fixture of FR-093.

| Roll | Entry (d100) | Entry text |
|---|---|---|
| 1 | 01 | A mummified goblin hand |
| 34 | 34 | A rectangular metal device with two tiny metal cups on one end that throws sparks when wet |
| 35 | 35 | A white, sequined glove sized for a human |
| 47 | 47 | A metal can that has no opening but sounds as if it is filled with liquid, sand, spiders, or broken glass (your choice) |
| 72 | 72 | A silver teardrop earring containing a real teardrop |
| 73 | 73 | An eggshell painted with scenes of misery in disturbing detail |
| 93 | 93 | An empty wine bottle bearing a pretty label that says, "The Wizard of Wines Winery, Red Dragon Crush, 331422-W" |
| 99 | 99 | A wooden box with a ceramic bottom that holds a living worm with a head on each end of its body |
| 100 | 00 | A metal urn containing the ashes of a hero |
| 0 | n/a | an invalid-input error (`srderr.ErrInvalidInput`) |
| 101 | n/a | an invalid-input error (`srderr.ErrInvalidInput`) |

### 4.4 Use cases, telemetry and cross-cutting behaviour

Signals are the shared character-context ones owned by `character-aggregate.md`
FR-103 and D-21; this spec adds only operation and result values (section 7).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-120 | No use case of this spec MUST call another use case, and none MUST be called by another spec's use case: `StartAtHigherLevel`, `RollHigherLevelStartingMoney` and `RollTrinket` compose pure domain services (`Character.LevelUp`, `SubclassSelection`, `AbilityScoreIncreaser`, `HigherLevelStart`, `HigherLevelEquipmentAllowance.TotalGold`, `TrinketTable.EntryFor`) and use their own ports. A spec that needs a Trinket or the starting money inside its own use case (`CreateCharacter`) rolls the die through its own `dice.Roller` and calls `TrinketTable.EntryFor` or `TotalGold`. Each use case records only its own signals, taking attribute values from the domain result. | `shared-types-and-ownership.md` FR-044 | unhappy (nested telemetry) |
| FR-121 | The three use cases MUST return equal results with the zero-value `telemetry.Noop` (`shared-types-and-ownership.md` FR-036) and with a recording one, and MUST emit the signals of section 7 once per call, using defined constants, and MUST NOT log. `srd5e.outcome` MUST be only `success` or `failure`; every returned result's variant MUST be in `srd5e.character.result` (a returned port failure carries none). | `doc/rules/observability.md` §5-§7, §9; `errors-and-results.md` FR-043; `character-aggregate.md` FR-103 | happy |
| FR-122 | A returned `StartAtHigherLevelContentMissing` MUST be recorded exactly once: outcome `failure`, result `content_missing`, span error status, one increment of `srd5e.character.operations`, no `error.type` and no log. A returned `*srderr.PortError` (a content port failure or one for `dice.Port`) MUST be recorded exactly once with `Span.RecordError`, outcome `failure`, no result attribute, `error.type` equal to `srderr.ErrorType(err)` (`errors-and-results.md` FR-034, never `%T` or `err.Error()`), and no log. A rule outcome (a rejected start) MUST be recorded as a normal outcome: `failure`, result `rejected`, `srd5e.character.rejection`, span status not error, no warning or error log; a started character, a rolled money, a money roll that is `not_applicable` (starting level 1) and a rolled Trinket are `success`. A cancelled `ctx` is returned as `ctx.Err()` and is neither success nor failure (`errors-and-results.md` FR-054). | `doc/rules/observability.md` §6, §7; `errors-and-results.md` FR-005, FR-034, FR-043; `character-aggregate.md` FR-071, FR-072, FR-088 | unhappy (telemetry on failure) |
| FR-123 | Every domain type of this spec MUST contain no logging, metric or span call and MUST NOT import `telemetry` or any adapter package (only the `*_usecase.go` files may): `HigherLevelStart`, `HigherLevelStep`, `StartingLevelStep`, `ConsumedFeatGrant`, `HigherLevelStartRejection` and its variants, `HigherLevelEquipment` and its variants, `RolledGold`, `Trinket` and `TrinketTable`. Every exported name MUST be unique across the package, no generic variant name (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`) MUST be exported, each closed result MUST be named `<UseCase>Result` with variants prefixed by the use case, no type owned by another spec MUST be redefined (in particular none of `LevelAdvancementStep`, `HigherLevelEquipmentChoice`, `TrinketChoice`, `TrinketRoll`, `ItemRarity`, `FeatGrant`, `ClassDefinition`, `MissingContent`), and every collection exposed by results, commands and value objects MUST be a defensive copy (mutating the returned slice leaves the value unchanged). | `doc/rules/observability.md` §3, §10; `doc/rules/architecture.md` §4, §9; `shared-types-and-ownership.md` FR-017, FR-021 | unhappy (architecture, name collision, immutability) |

### 4.5 Withdrawn ids

None. Ids FR-065 to FR-097, FR-106, FR-113, FR-115, FR-116, FR-118, FR-119 arrive
live from `level-advancement-and-multiclassing.md`. The gaps FR-098 to FR-105, FR-107 to FR-112, FR-114 and FR-117 are
reserved group spacing, never issued (not lost requirements).

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

Domain (`character`) is pure, immutable, synchronous and silent;
its types are those of FR-123. `HigherLevelStart.Advance` folds `LevelUp`,
subclass choice and increase over the steps and reports consumed grants.

| Invariant | Enforced by | Failure |
|---|---|---|
| Plan length equals starting level minus 1; step sequence legal | `HigherLevelStart` (FR-066, via `LevelUp`) | `HigherLevelStartPlanLengthMismatch` and the other rejections |
| A step's increase matches a grant of its class level; every applied one is reported | `HigherLevelStart` (FR-113, FR-118) | `HigherLevelStartIncreaseNotGranted` |
| Rolled gold only for starting level 5 or more; roll within the die | `HigherLevelEquipmentAllowance.TotalGold` (FR-116) | an invalid-input error (`srderr.ErrInvalidInput`) |
| Trinket table has exactly 100 entries in order; roll 1 to 100 | `NewTrinketTable` (FR-090, FR-091) | an error matching `srderr.ErrInvalidInput` |

Rule outcomes are results; invalid arguments are errors matching `srderr.ErrInvalidInput` (architecture
§6). The domain does no I/O and returns no `*srderr.PortError`; a roller that breaks
its contract is detected by the use case (FR-079, FR-087, FR-096). References to
other contexts are by id and published language only. The edge to
`level-advancement-and-multiclassing.md` is the one same-context direction added
(`shared-types-and-ownership.md` FR-016).

## 6. Interface

Use cases are structs built by a `NewX` constructor with options, one method taking `ctx`
first and an immutable command, and their own
closed `<UseCase>Result` (an interface with an unexported marker method), an optional `telemetry.Telemetry` (`WithTelemetry` option, no-op default) and ports
through the constructor; each use case declares the small port interfaces it consumes.

| Use case | Command | Result (closed) | Signature |
|---|---|---|---|
| `StartAtHigherLevel` | `StartAtHigherLevelCommand` (4.1) | `StartAtHigherLevelStarted{Character, Events, ConsumedGrants}`, `StartAtHigherLevelRejected{Reason HigherLevelStartRejection}`, `StartAtHigherLevelContentMissing{Missing}` | `Start(ctx, cmd) (StartAtHigherLevelResult, error)` (content and dice ports) |
| `RollHigherLevelStartingMoney` | `RollHigherLevelStartingMoneyCommand`: starting `CharacterLevel` | `RollHigherLevelStartingMoneyRolled{Allowance, Roll, Total}`, `RollHigherLevelStartingMoneyNotApplicable` | `Roll(ctx, cmd) (RollHigherLevelStartingMoneyResult, error)` (dice port) |
| `RollTrinket` | `RollTrinketCommand`: `TrinketTable` | `RollTrinketRolled{Roll, Trinket}` | `Roll(ctx, cmd) (RollTrinketResult, error)` (dice port) |

`missing` is a defensively copied, non-empty `MissingContent` list, ordered and
deduplicated as `character-aggregate.md` FR-077. `StartAtHigherLevelStarted` holds
the character (its `FeatureStates` hold the chosen subclasses), the ordered events
(FR-076) and the defensively copied `ConsumedGrants` (FR-118). A rule rejection is a
result; only a port failure or a broken port contract is a returned `*srderr.PortError`. The earlier
names `HigherLevelStarted`, `HigherLevelMoneyRolled`,
`HigherLevelMoneyNotApplicable` and `TrinketRolled` are replaced by the variants
above (Q-21).

Ports (none new): `content.Source` (takes `ctx`; closed `ContentLookup` for absence
mapped to `...ContentMissing`; a `*srderr.PortError` for `content.Port` propagates; the Trinkets
table is never looked up, Q-11); `dice.Roller` (sync; one die per rolled step after
validation FR-074, `1d10` starting money, `1d100` Trinket; a value outside the die
becomes a `*srderr.PortError` for `dice.Port` reason `InvalidResult`, `errors-and-results.md`
FR-036); `telemetry.Telemetry` (default the zero-value `telemetry.Noop`). **Exports:**
package `character` exports every type of
FR-123, the three use cases, commands and results; no internal type appears in an
exported signature.

## 7. Observability

Domain code is **silent by design**; the use cases are instrumented
(`doc/rules/observability.md` §2). Metrics are the character context's own
(`character-aggregate.md` FR-103, D-21): counter `srd5e.character.operations` and
histogram `srd5e.character.duration`, the latter with the single attribute
`srd5e.character.operation`; this spec adds the values `start_at_higher_level`,
`starting_money` and `trinket` to that attribute.

Evaluation: (1) wrong: content missing, a failing or panicking port or roller, a value
outside the die, rule outcomes, bugs. (2) Operator signal: content missing is
span error status with result `content_missing` and no `error.type`; a returned
failure is span error status with `error.type`; a rejected start is `failure`
with result `rejected` and `srd5e.character.rejection`, span status not error.
(3) Diagnosis: operation, outcome, result, rejection or error type, step count.
(4) Counted or timed: calls and outcomes, duration by operation, steps per start,
lookup time. (5) Reused, not redefined: `srd5e.outcome`, `error.type`,
`srd5e.character.result`, `srd5e.character.rejection`, the `load_content` span of
`level-advancement-and-multiclassing.md`. (6) Monitor: any `error.type` increment,
`start_at_higher_level` `subclass_required` or `plan_length_mismatch`,
`content_missing`, duration regressions. (7) Noise: no log; no per-step metric.

Values added to `srd5e.character.result`: `start_at_higher_level`: `started`
(success), `rejected`, `content_missing` (failure); `starting_money`: `rolled`,
`not_applicable` (success); `trinket`: `rolled` (success). Values added to
`srd5e.character.rejection` for `start_at_higher_level`: `plan_length_mismatch`,
`creation_invalid`, `level_up_rejected`, `subclass_rejected`, `increase_rejected`,
`increase_not_granted`, `subclass_required`. New values go to the catalogue
(`observability.md` §5).

| Signal | Type, unit | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.advancement.start_at_higher_level` | span (child `srd5e.character.advancement.load_content`) | `srd5e.outcome`, `srd5e.character.result`, `srd5e.character.rejection` (rejected only) | One higher-level start including lookups and dice | Error status rate above 0 for sustained periods |
| `srd5e.character.advancement.starting_money` | span | `srd5e.outcome`, `srd5e.character.result` | One starting money roll | Error status rate above 0 (dice failures) |
| `srd5e.character.advancement.trinket` | span | `srd5e.outcome`, `srd5e.character.result` | One Trinket roll | Error status rate above 0 (dice failures) |
| `srd5e.character.operations` (shared) | counter, `{operation}` | `srd5e.character.operation` (the values above), `srd5e.outcome`, result (returned results only), rejection (rule outcomes only), `error.type` (returned port failures only) | Throughput and outcome mix | Alert on any `error.type` increment |
| `srd5e.character.duration` (shared) | histogram, `s` | `srd5e.character.operation` only | Latency per use case | p99 regression per operation |
| `srd5e.character.start_steps` | histogram, `{step}` | none | Size of higher-level start plans | Watch for a shift to many steps |

Attribute values are low-cardinality enumerations (no ids, class names, XP,
scores or trinket text). Entries feed `doc/observability/signals.md`.

## 8. Test plan

Doubles (from `dicetest` and `telemetrytest`): `dicetest.FixedRoller` (also returning 0, above the
die, and failing with an error), `telemetrytest.Recorder`, in-memory
`content.Source` (`contenttest.FakeSource`, also failing with an error); content fixtures for
Fighter, Barbarian, Druid, a species and a background. The independent Trinkets
transcription under `testdata/srd/` is the fixture of FR-093.

| Id | Level | Notes |
|---|---|---|
| FR-065 to FR-067 | unit, application | Table SL; level 1 equals `NewCharacter` |
| FR-066, FR-069 to FR-073, FR-106, FR-113, FR-118 | unit | Table SH through `HigherLevelStart.Advance`; property over starting levels 2 to 20 for FR-070; not-granted, repeated-grant and consumed-grant rows |
| FR-068, FR-074 to FR-080, FR-119 | application | Table SH through the use case with fixed rollers (all 1, all maximum, mixed) recording the exact roll count (0 for rejected plans); event order; source missing the initial class, a step class and the species at once; score 21; failing source and roller; roller returning 0 and size + 1; command and steps unaltered |
| FR-081 to FR-084, FR-089, FR-116 | unit | Table SE per row; `TotalGold` on every row of SM, nil roll for a rolled tier, rolls 0 and 11, a roll for a tier with no rolled gold; immutability and value equality |
| FR-085 to FR-088 | application | Table SM with a fixed roller recording whether it was called; out-of-range rolls; failing roller |
| FR-090 to FR-097 | unit, application | 99, 100, 101 entries; blank text; table TR rows 0 and 101; verbatim text; table from the fixture (every roll 1 to 100); fixed roller, roller returning 0 and 101, failing roller; repeat calls; scan that no `character/` file embeds the entries; fuzz `NewTrinketTable` with arbitrary entry counts |
| FR-115, FR-120, FR-123 | architecture | `internal/archtest` allow-list: no `telemetry` or adapter imports in domain files; duplicate and generic exported-name scan, and for `LevelAdvancementStep`, `StandardEquipmentOnly`, `StartingEquipmentAtHigherLevels`; no use case imports another; `CreateCharacter` reaches `EntryFor` and `TotalGold` only; collections defensively copied |
| FR-121, FR-122 | application | Recording doubles: span names, attribute keys, `success`/`failure` only, result and rejection attributes, no log and no `error.type` on a returned failure, each failure recorded once, `error.type` equal to `srderr.ErrorType`, duration histogram with the operation attribute only; no-op and recording `telemetry.Telemetry` give equal results |

- **Scenario** (`package scenarios_test`, public API only):
  `StartAtHigherLevel` (consumed grant reported) with its money through
  `TotalGold`; a Trinket from the fixture table. Cites FR-065, FR-068, FR-085,
  FR-094, FR-118.
- **Mutation testing:** `character` changed lines, command in
  `doc/rules/testing.md` §8 (`gomu`, at least 80; run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors); the `*_usecase.go` files SHOULD be. Emphasis:
  grant check of FR-113 and FR-118, tier boundaries of FR-081 to FR-084, roll
  mapping of FR-091 (100 versus 00).

## 9. Open questions and decisions

Open: none.

Settled: Q-11 `Trinket` and `TrinketTable` are host-supplied pure values (`shared-types-and-ownership.md` FR-005, FR-013,
FR-047), a later content kind would produce this type; Q-12 magic items use `ItemRarity` of `content-magic-items.md`
FR-014; Q-20 a subclass above level 3 is chosen in the plan (FR-071), reported by FR-073, or chosen later; Q-21
`character-creation.md` now uses these names, a `ClassID` in `StartingLevelStep`, and rolls through its own
`dice.Roller` calling `TotalGold` and `TrinketTable.EntryFor` (its FR-073 to FR-082, FR-087 to FR-091, FR-113); Q-23
this spec is rank 15 and `character-creation.md` rank 16 (`shared-types-and-ownership.md` 4.11, FR-056).

## 10. Changelog impact

`CHANGELOG.md`, `Unreleased` / **Added**: "`character`: `StartAtHigherLevel`;
`HigherLevelEquipment` and `RollHigherLevelStartingMoney`; `RollTrinket` and the
host-supplied `TrinketTable`." Not **BREAKING** (first release of these types).
