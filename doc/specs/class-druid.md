# Druid class

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `character-aggregate.md`, `derived-stats.md`, `spellcasting.md`, `rests-and-hit-dice.md`, `level-advancement-and-multiclassing.md`, `feats-behaviour.md`, `content-classes.md`, `content-monsters.md`, `content-spells.md`, `content-schema-and-decoder.md`, `d20-tests.md`, `ability-scores.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md` |

`character` depends only on `content`, `d20` and the shared kernel; this spec names no `combat` type.
Roadmap: [../roadmap.md](../roadmap.md).

## 1. Purpose and scope

Specifies the **`character` context behaviour** of the Druid class of SRD 5.2.1 (pp. 41-46): what a Druid gains at
each level, its choices (Primal Order, Elemental Fury, land type, Beast forms), the per-feature state (Wild Shape uses,
once-per-Long-Rest limits) and the pure state-in / new-state-and-events-out functions that spend, restore and apply
them. The Druid's **content data** (Core Druid Traits, Features table, spell list, Circle of the Land tables) and its
slots, cantrips known, prepared spell counts and maximum spell level are owned by `content-classes.md`
(shared-types-and-ownership.md FR-025) and derived by `spellcasting.md` (FR-010, FR-014, FR-015, FR-017); the
progression, spell-list and land-table tests of `content-classes.md` are the oracle.

**Out of scope:** class and subclass data shapes (`content-classes.md`); choosing a subclass, multiclass prerequisites, Hit
Points and Hit Dice (`level-advancement-and-multiclassing.md`); `RestKind`, `RestApplied` (`rests-and-hit-dice.md`); generic
spellcasting, `ChangePreparedSpells`, rituals, Concentration, Armor Training (`spellcasting.md`); Beast stat blocks and
`ChallengeRating` (`content-monsters.md`; a form arrives as facts, FR-171); Armor Class, speed and attacks in a form,
Temporary Hit Points, damage, resistance, conditions, cover and rolling dice (the consumer); feat slots and Ability
Score Improvement / Epic Boon contents (`feats-behaviour.md`); "other sources" for Beast forms, narrative retention
while shape-shifted, Druidic messages beyond the detection rule, other subclasses, a Beast form reduced to 0 Hit Points
(no rule in the pages read), the Find Familiar text and Starting Equipment item definitions.

## 2. Source

PDF page numbers equal printed page numbers.
- SRD 5.2.1, "Proficiency Bonus" and "The Bonus Doesn't Stack", p. 8; "Multiclassing" (Proficiency Bonus, Spell Slots), p. 25
- SRD 5.2.1, "Druid": Core Druid Traits, "Becoming a Druid ...", Druid Features table, "Level 1: Spellcasting", pp. 41-42; "Level 1: Druidic", "Level 1: Primal Order", p. 42
- SRD 5.2.1, "Level 2: Wild Shape", Beast Shapes table, p. 42; "Rules While Shape-Shifted" and the features from "Level 2: Wild Companion" to "Level 20: Archdruid", p. 43
- SRD 5.2.1, "Druid Spell List", pp. 44-45; "Druid Subclass: Circle of the Land", p. 46
- Package design: `shared-types-and-ownership.md`, `doc/rules/architecture.md` (§3, §4, §6, §9), `doc/rules/observability.md`, `doc/rules/testing.md` (§4, §8).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Druid, Core Druid Traits, Features table, spell list; Druid level | `ClassDefinition`, `ProgressionRow`s; Druid entry of `ClassLevels` | `content-classes.md`; Druid level is not total level, carried in `DruidCharacterFacts` (FR-167). |
| Primal Order, Elemental Fury | `PrimalOrder` (`PrimalOrderMagician`, `PrimalOrderWarden`), `ElementalFury` (`ElementalFuryPotentSpellcasting`, `ElementalFuryPrimalStrike`) | Prose uses the shorthand. |
| Wild Shape, Known Forms, Beast | `WildShapeState`, `KnownForms`, `BeastFormFacts` | Fly Speed: `hasFlySpeed` (content-monsters.md FR-016). Temporary Hit Points: an `int` on `WildShapeAssumed`. |
| Circle of the Land, land type | `SubclassId` `circle_of_the_land`, `LandType` (arid, polar, temperate, tropical) | Ids are snake_case. |
| Spell slots, Concentration | `SpellcastingState`, none here | `spellcasting.md`; slots changed only by host-applied `DruidSlotChange` (FR-160); Concentration untouched (FR-083). |
| Spell save DC | `SpellSaveDc` (spellcasting.md) | Supplied by the caller from the character's `SpellcastingStats`. |
| Rejection reasons; rest-timed choice | `DruidRejection` (enum); `RestWindowPermission` (`landTypeChange`, `knownFormReplacement`, `naturalRecoveryRecovery`) | PascalCase names here are shorthand for camelCase members; FR-153. |
| Primal Strike attack | `PrimalStrikeAttackSource`, `AttackRollResult` (d20-tests.md) | Host-supplied (FR-108); not a combat type. |
| Magic action, Bonus Action | `DruidActionCost` (`magicAction`, `bonusAction`, `none`) | Data only. |
| Druidic detection | `DruidicDetection` (`presence`, `content`); `DifficultyClass` (d20-tests.md) | FR-045 to FR-047. |
| Druid state, character id, item key | `DruidFeatureState` in `FeatureStates`; `CharacterId`; `DruidItemKey` (defined type on `string`) | FR-162, FR-166; opaque host key (FR-084). |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

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

Stable ids. "The Druid" is a character with at least one Druid level; "Druid level" is the Druid class level. Case tables
follow; rejections are results (section 5), programmer errors are panics, D-n are decisions (section 9).

### 4.1 Core Druid Traits and levels 1 to 20

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-004 | A starting Druid MUST choose exactly 2 skill proficiencies from Animal Handling, Arcana, Insight, Medicine, Nature, Perception, Religion and Survival, carried in `DruidCreationChoices` (FR-163); a valid choice is accepted. The list is read from the Druid `ClassDefinition`. | SRD 5.2.1, "Core Druid Traits" (Skill Proficiencies), p. 41 | happy |
| FR-005 | A skill choice with 0, 1 or 3 skills MUST be rejected with `WrongNumberOfSkills`. | SRD 5.2.1, "Core Druid Traits", p. 41 ("Choose 2") | unhappy (boundary) |
| FR-006 | A skill choice naming the same skill twice MUST be rejected with `DuplicateSkill`. | SRD 5.2.1, "Core Druid Traits", p. 41 | unhappy (invalid input) |
| FR-007 | A skill choice containing a skill outside the list in FR-004 (for example Acrobatics) MUST be rejected with `SkillNotOnList`. | SRD 5.2.1, "Core Druid Traits", p. 41 | unhappy (invalid input) |
| FR-011 | Choosing a Starting Equipment package MUST require exactly one of A or B; any other package id MUST be rejected with `UnknownEquipmentPackage`. | SRD 5.2.1, "Core Druid Traits" ("Choose A or B"), p. 41 | unhappy (invalid input) |
| FR-012 | A Druid taken as the starting class MUST gain every Core Druid Trait of its `ClassDefinition` and the level 1 features (Spellcasting, Druidic, Primal Order). | SRD 5.2.1, "Becoming a Druid ... As a Level 1 Character", p. 41 | happy |
| FR-013 | A Druid taken as a multiclass MUST gain only the Hit Point Die and training with Light armor and Shields (the multiclass trait list of its `ClassDefinition`), plus the level 1 features; it MUST NOT gain the saving throws, skill, weapon or tool proficiencies or Starting Equipment, and its pending choices MUST NOT include skills or an equipment package. | SRD 5.2.1, "Becoming a Druid ... As a Multiclass Character", p. 41 | unhappy (rule interaction) |
| FR-023 | Gaining a Druid level MUST accept only the next level (none to 1, or n to n+1); any other target MUST be rejected with `LevelNotConsecutive`. | SRD 5.2.1, Druid Features table, p. 41; `character-aggregate.md` | unhappy (boundary) |
| FR-024 | Gaining a level above 20 MUST be rejected with `MaximumLevelReached`. | SRD 5.2.1, Druid Features table, p. 41 | unhappy (boundary) |
| FR-025 | The result of gaining level n MUST list the features granted at exactly level n and the pending choices that level creates, and MUST NOT repeat features of earlier levels. | SRD 5.2.1, "Druid Class Features", p. 41 | happy |
| FR-026 | Gaining level 1 as the starting class MUST create the Druid-owned pending choices: 2 skills, an equipment package and a Primal Order; as a multiclass the skill and equipment choices MUST be absent. The cantrip and prepared-spell choices of level 1 are derived by `spellcasting.md` from `ClassLevels` (FR-038, FR-039). | SRD 5.2.1, "Core Druid Traits", "Level 1: Spellcasting", "Level 1: Primal Order", pp. 41-42 | happy |
| FR-027 | Gaining level 2 MUST grant Wild Shape with its uses at maximum (2), a pending choice of 4 known forms, and Wild Companion. | SRD 5.2.1, "Level 2: Wild Shape", "Level 2: Wild Companion", pp. 42-43 | happy |
| FR-030 | Gaining a level that raises the number of known forms (levels 4 and 8) MUST add the difference as pending known-form choices (table B). | SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), p. 42 | happy |
| FR-031 | Gaining a level MUST NOT remove or alter any earlier Druid choice (Primal Order, known forms, Elemental Fury, land type); the subclass choice is kept by the character. | SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), "Level 3: Druid Subclass" ("For the rest of your career"), pp. 42-43 | unhappy (immutability) |

### 4.2 Spellcasting inputs

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-035 | The spellcasting ability MUST be Wisdom and a Druidic Focus MUST be accepted as a Spellcasting Focus; these MUST agree with `spellcasting.md`. | SRD 5.2.1, "Level 1: Spellcasting" (Spellcasting Ability, Spellcasting Focus), p. 42 | happy |
| FR-037 | Choosing and validating Druid cantrips and prepared spells MUST be delegated to `ChangePreparedSpells` (`spellcasting.md` FR-040 to FR-052). A spell not on the Druid list (for example Fireball) is rejected with its `NotOnClassList`; this spec defines no Druid-named rejection. Spells arriving through Circle Spells and the Magician's cantrip are handled as in FR-120 and FR-051. | SRD 5.2.1, "Level 1: Spellcasting" ("from the Druid spell list"), p. 42 | unhappy (invalid input) |
| FR-038 | Cross-spec test, no supply: `spellcasting.md`, fed the Druid `ClassLevels`, MUST give the maximum preparable spell level and slot counts of the Druid progression (a level 3 Druid has slots to level 2; spellcasting.md FR-015, FR-017); a level 3 spell chosen by a level 3 Druid is then rejected with `SpellLevelTooHigh` (`spellcasting.md` FR-041). | SRD 5.2.1, "Level 1: Spellcasting" (Prepared Spells of Level 1+), p. 42 | unhappy (boundary) |
| FR-039 | Cross-spec test, no supply: the prepared-spell limit and cantrips known of the Druid progression are derived by `spellcasting.md` (FR-010, FR-014); `ChangePreparedSpells` accepts any mix of levels within it (a level 5 Druid with 9 prepared spells MAY hold 6 of level 1, 2 of level 2 and 1 of level 3) and rejects a longer list with `TooManyPrepared` (`spellcasting.md` FR-042). | SRD 5.2.1, "Level 1: Spellcasting" (Prepared Spells of Level 1+), p. 42 | unhappy (boundary) |
| FR-040 | The Druid MUST supply Speak with Animals as an always-prepared spell from level 1 (`spellcasting.md` FR-045); it does not count against the prepared limit and removing or replacing it is rejected with `AlwaysPreparedNotReplaceable` (`spellcasting.md` FR-046). | SRD 5.2.1, "Level 1: Druidic", "Level 1: Spellcasting" (always prepared), p. 42 | unhappy (state error) |
| FR-041 | Spells a feature always prepares (Speak with Animals, Circle Spells) MUST be supplied to `spellcasting.md` as always-prepared spells, which it counts as Druid spells (`spellcasting.md` FR-045). | SRD 5.2.1, "Level 1: Spellcasting" (Prepared Spells of Level 1+), p. 42 | happy |
| FR-042 | Cross-spec test: with the Druid `ClassLevels` entry at level 3 inside a Druid 3 / Fighter 5 character, `spellcasting.md` MUST give 2 cantrips, 6 prepared spells and a maximum spell level of 2 ("as if single-classed", spellcasting.md FR-017). | SRD 5.2.1, Druid Features table, p. 41 | unhappy (rule interaction) |

### 4.3 Druidic

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-044 | A Druid MUST know the language Druidic (`Language.druidic`) from level 1. | SRD 5.2.1, "Level 1: Druidic", p. 42 | happy |
| FR-045 | `DruidRules.druidicDetection(languages)` (a `Set<Language>`, the observer's known languages) MUST return a `DruidicDetection` with `presence` (closed `DruidicPresence`: `DruidicPresenceAutomaticallySpotted` or `DruidicPresenceRequiresCheck`) and `content` (closed `DruidicContent`: `DruidicContentReadable` or `DruidicContentNeedsMagic`). A creature that knows Druidic gets `DruidicPresenceAutomaticallySpotted` and `DruidicContentReadable`, with no check. | SRD 5.2.1, "Level 1: Druidic", p. 42 | happy |
| FR-046 | For a creature that does not know Druidic, `presence` MUST be `DruidicPresenceRequiresCheck(dc: DifficultyClass(15), ability: Ability.intelligence, skill: Skill.investigation)`; the consumer rolls. A successful check spots the presence only. | SRD 5.2.1, "Level 1: Druidic", p. 42 | unhappy (rule interaction) |
| FR-047 | For a creature that does not know Druidic, `content` MUST be `DruidicContentNeedsMagic` whatever the result of the presence check ("can't decipher it without magic"); the two facts are independent fields. | SRD 5.2.1, "Level 1: Druidic", p. 42 | unhappy (rule interaction) |

### 4.4 Primal Order

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | The Primal Order MUST be a pending choice at level 1 between Magician and Warden, as the starting class or as a multiclass. | SRD 5.2.1, "Level 1: Primal Order", p. 42; "Becoming a Druid", p. 41 | happy |
| FR-049 | Choosing a Primal Order when one is already chosen MUST be rejected with `PrimalOrderAlreadyChosen`. | SRD 5.2.1, "Level 1: Primal Order", p. 42 | unhappy (state error) |
| FR-050 | Magician MUST add one cantrip known on top of the table, always from the Druid spell list: 3 at levels 1-3, 4 at 4-9 and 5 at 10-20 (table D). | SRD 5.2.1, "Level 1: Primal Order" (Magician), p. 42 | happy |
| FR-051 | The Magician MUST supply its extra-cantrip allowance (`ClassSpellcasting.extraCantrips`, 1) to `spellcasting.md` (FR-096); an extra cantrip not on the Druid list or already known is rejected by `ChangePreparedSpells` with `NotOnClassList` or `DuplicateSpell` (`spellcasting.md` FR-052). | SRD 5.2.1, "Level 1: Primal Order" (Magician), p. 42 | unhappy (invalid input) |
| FR-052 | Magician MUST give a bonus to Intelligence (Arcana) and Intelligence (Nature) checks equal to the Wisdom modifier, with a minimum bonus of +1 (table D), computed by `DruidRules.magicianCheckBonus` (table M). | SRD 5.2.1, "Level 1: Primal Order" (Magician), p. 42 | happy |
| FR-053 | The Magician bonus MUST NOT apply to any other check: Intelligence (History), Wisdom (Nature) and Wisdom (Perception) checks get 0. | SRD 5.2.1, "Level 1: Primal Order" (Magician), p. 42 | unhappy (rule interaction) |
| FR-054 | Warden MUST grant proficiency with Martial weapons and training with Medium armor, and MUST NOT grant an extra cantrip or the Magician check bonus. | SRD 5.2.1, "Level 1: Primal Order" (Warden), p. 42 | happy |
| FR-055 | Magician MUST NOT grant Martial weapon proficiency or Medium armor training. | SRD 5.2.1, "Level 1: Primal Order", p. 42 | unhappy (rule interaction) |
| FR-056 | A multiclass Druid MUST receive the effects of its chosen Primal Order, because Primal Order is a level 1 feature gained as a multiclass (D-6). | SRD 5.2.1, "Becoming a Druid ... As a Multiclass Character", p. 41 | happy |

### 4.5 Wild Shape

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-057 | Wild Shape MUST be unavailable below Druid level 2: any Wild Shape operation MUST be rejected with `FeatureNotGranted` and leave the state unchanged. | SRD 5.2.1, "Level 2: Wild Shape", p. 42 | unhappy (state error) |
| FR-058 | Maximum uses MUST be 2 at levels 2-5, 3 at levels 6-16 and 4 at levels 17-20 (the Druid progression), and 0 below level 2. | SRD 5.2.1, "Level 2: Wild Shape" (Number of Uses), p. 42; Druid Features table, p. 41 | happy |
| FR-059 | When the maximum rises by n through a level, the remaining uses MUST rise by n; gaining Wild Shape at level 2 MUST set remaining equal to the maximum (Q-1). | SRD 5.2.1, "Level 2: Wild Shape" ("You gain additional uses"), p. 42 | happy |
| FR-060 | Assuming a Beast form MUST spend exactly one use, set the active form, and emit `WildShapeAssumed` with the form, the duration and the Temporary Hit Points amount. | SRD 5.2.1, "Level 2: Wild Shape", p. 42 | happy |
| FR-061 | Assuming a form with 0 uses remaining MUST be rejected with `NoUsesRemaining` and leave the state unchanged. | SRD 5.2.1, "Level 2: Wild Shape" (Number of Uses), p. 42 | unhappy (state error) |
| FR-062 | Assuming a form not among the known forms MUST be rejected with `FormNotKnown`. | SRD 5.2.1, "Level 2: Wild Shape" ("a Beast form that you have learned"), p. 42 | unhappy (invalid input) |
| FR-063 | Assuming a form while already in a Beast form (a switch) MUST spend another use, end the current form with reason `usedWildShapeAgain`, and activate the new form, emitting `WildShapeEnded` then `WildShapeAssumed` in that order (D-3; table O). With 0 uses remaining it MUST be rejected with `NoUsesRemaining` and the current form MUST stay active. The new form, duration, item dispositions and Temporary Hit Points follow FR-060, FR-065, FR-070, FR-084 and FR-085. | SRD 5.2.1, "Level 2: Wild Shape" ("or until you use Wild Shape again"), p. 42 | unhappy (state error) |
| FR-064 | The `WildShapeAssumed` event and the leave-early result MUST each carry `actionCost: DruidActionCost.bonusAction`. | SRD 5.2.1, "Level 2: Wild Shape", p. 42 | happy |
| FR-065 | The duration MUST be a number of hours equal to half the Druid level, as an exact `time.Duration` (table C). | SRD 5.2.1, "Level 2: Wild Shape", p. 42 | happy |
| FR-066 | Ending Wild Shape MUST clear the active form and emit `WildShapeEnded` with the reason `durationElapsed`, `incapacitated`, `died`, `usedWildShapeAgain` (produced only by FR-063) or `leftEarly`. | SRD 5.2.1, "Level 2: Wild Shape" ("for a number of hours equal to half your Druid level or until you use Wild Shape again, have the Incapacitated condition, or die. You can also leave the form early"), p. 42 | happy |
| FR-067 | Ending when not in a Beast form MUST be rejected with `NotInBeastForm`. Leaving early or any other end MUST NOT refund a use. | SRD 5.2.1, "Level 2: Wild Shape", p. 42 | unhappy (state error) |
| FR-068 | Finishing a Short Rest MUST regain exactly one expended use, never above the maximum, and none when none are expended. | SRD 5.2.1, "Level 2: Wild Shape" (Number of Uses), p. 42 | unhappy (boundary) |
| FR-069 | Finishing a Long Rest MUST regain all expended uses. | SRD 5.2.1, "Level 2: Wild Shape" (Number of Uses), p. 42 | happy |
| FR-070 | The Temporary Hit Points on assuming a form MUST equal the Druid level (table C), which for a multiclass character is the Druid level and not the total level. | SRD 5.2.1, "Rules While Shape-Shifted" (Temporary Hit Points), p. 43 | unhappy (rule interaction) |
| FR-071 | The Temporary Hit Points MUST be reported as an amount for the consumer to apply; this context MUST NOT hold them. In a switch, `WildShapeAssumed` carries a fresh amount equal to the Druid level and `WildShapeEnded(usedWildShapeAgain)` carries none and removes none: replace-or-keep is the consumer's no-stacking rule (D-1). | SRD 5.2.1, "Rules While Shape-Shifted" (Temporary Hit Points), p. 43 | happy |
| FR-072 | Known forms, maximum Challenge Rating and Fly Speed permission MUST follow table B by Druid level (levels 2-3, 4-7 and 8-20), including the boundary levels 3, 4, 7 and 8. | SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), Beast Shapes table, p. 42 | unhappy (boundary) |
| FR-073 | A form MUST be eligible only if it is a Beast, its Challenge Rating is at most the maximum for the Druid level, and it has no Fly Speed unless the Druid level is at least 8. The comparison uses the total order of `ChallengeRating` (content-monsters.md FR-097); `BeastShapeTier.maxChallengeRating` is a `ChallengeRating`. | SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), p. 42; content-monsters.md FR-097 | happy |
| FR-074 | An ineligible form MUST be reported as `FormEligibilityIneligible` carrying `NotABeast`, `ChallengeRatingTooHigh` or `FlySpeedNotAllowed`; when more than one applies the first in that order MUST be reported. An eligible form is `FormEligibilityEligible`. | SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), p. 42 | unhappy (rule interaction) |
| FR-075 | Choosing known forms MUST reject duplicates with `DuplicateForm` and a list longer than the allowed number with `TooManyKnownForms`; fewer than the number is accepted and the rest stay pending. | SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), p. 42 | unhappy (boundary) |
| FR-076 | Known forms MAY be replaced one at a time only while the `knownFormReplacement` permission is open (opened by finishing a Long Rest, FR-153, FR-154), at most one replacement per Long Rest; with the permission closed MUST be rejected with `ChangeNotAllowedNow`, and a second replacement in the same window with `TooManyReplacements`. | SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), p. 42 | unhappy (state error) |
| FR-077 | A replacement MUST name a currently known form and an eligible form not already known; otherwise `FormNotKnown`, an ineligibility reason (FR-074), or `DuplicateForm`, in the order of D-5. | SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), p. 42 | unhappy (invalid input) |
| FR-078 | Known forms already chosen MUST stay valid when the Druid level rises. | SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), p. 42 | unhappy (rule interaction) |
| FR-079 | `DruidRules.shapeShiftedStats` MUST return a `RetainedStats` / `ReplacedStats` pair. `RetainedStats` lists creature type, Hit Points, Hit Dice, Intelligence, Wisdom and Charisma scores, class features, languages, feats, and the Druid's skill proficiencies, saving throws proficiencies and Proficiency Bonus (in addition to the Beast's proficiencies). `ReplacedStats` is every other game statistic, read from the Beast stat block. | SRD 5.2.1, "Rules While Shape-Shifted" (Game Statistics), p. 43 | happy |
| FR-080 | For a skill or saving throws the shape-shifted modifier MUST be the higher of the Druid's own (`AbilityModifier` plus Proficiency Bonus if `ShapeShiftedProficiency.proficient`) and the Beast stat block's `Modifier` when it lists one (table E). | SRD 5.2.1, "Rules While Shape-Shifted" (Game Statistics), p. 43 | unhappy (boundary) |
| FR-081 | A proficiency held by both the Druid and the Beast MUST add the Proficiency Bonus once, not twice. | SRD 5.2.1, "Rules While Shape-Shifted" (Game Statistics), p. 43; "The Bonus Doesn't Stack", p. 8 | unhappy (rule interaction) |
| FR-082 | `DruidRules.beastFormCastCheck` (table M) MUST return `BeastFormCastCheckBlocked` with reason `CannotCastInBeastForm` for any spell cast while in a form below Druid level 18 (No Spellcasting), and `BeastFormCastCheckAllowed` when the Druid is not in a form. The host consults it as in FR-155. | SRD 5.2.1, "Rules While Shape-Shifted" (No Spellcasting), p. 43 | unhappy (state error) |
| FR-083 | Assuming or being in a form MUST NOT break Concentration or interfere with a spell already cast. Concentration is owned by `spellcasting.md` (shared-types-and-ownership.md FR-011): no Druid operation reads, writes or reports it, and `WildShapeAssumed` has no Concentration field. | SRD 5.2.1, "Rules While Shape-Shifted" (No Spellcasting), p. 43 | unhappy (rule interaction) |
| FR-084 | Each worn or carried item, identified by the host's opaque `DruidItemKey`, MUST be given exactly one `EquipmentDisposition` (`EquipmentDispositionFallsInSpace`, `EquipmentDispositionMergesIntoForm`, `EquipmentDispositionWorn`) on assuming a form; the command carries the full key list and an unassigned key MUST be rejected with `ItemUnassigned`. | SRD 5.2.1, "Rules While Shape-Shifted" (Objects), p. 43 | unhappy (invalid input) |
| FR-085 | `EquipmentDispositionWorn` MUST be accepted only when the host states the form can wear the item (a per-item boolean in the command); otherwise `ItemCannotBeWorn`. `WildShapeAssumed` MUST list the worn keys (items that keep functioning) and the merged keys (items that do not function in the form) as two separate lists. | SRD 5.2.1, "Rules While Shape-Shifted" (Objects), p. 43 | unhappy (rule interaction) |

### 4.6 Wild Companion

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-086 | Wild Companion MUST be unavailable below level 2 (`FeatureNotGranted`). | SRD 5.2.1, "Level 2: Wild Companion", p. 43 | unhappy (state error) |
| FR-087 | Using it MUST spend exactly one of: one spell slot of a chosen level (1 or higher), or one use of Wild Shape; the `WildCompanionCast` event MUST carry `actionCost: DruidActionCost.magicAction`; the spell has no Material component when cast this way. It consults the same `beastFormCastCheck` as every other cast (FR-091, FR-155). | SRD 5.2.1, "Level 2: Wild Companion", p. 43 | happy |
| FR-088 | The slot payment MUST be rejected with `DruidRejection.slotRejected` carrying `SlotOperationRejection.noSlotAvailable` (spellcasting.md FR-109; the Druid defines no slot rejection of its own) when the chosen level has no remaining slot in the given `SpellcastingState`, and the Wild Shape payment with `NoUsesRemaining` when none remain; the state MUST be unchanged on rejection. On success the result carries a `DruidSlotChangeSpend` (FR-160) and the Wild Shape state is untouched. `NoSlotAvailable` and `NoExpendedSlot` are shorthand for `slotRejected` with that value. | SRD 5.2.1, "Level 2: Wild Companion", p. 43 | unhappy (state error) |
| FR-089 | Using it MUST NOT require Find Familiar to be prepared (an inference: the SRD says only "cast the Find Familiar spell"; Q-13). The familiar itself is `FamiliarOrigin.wildCompanion` in `summoned-and-companion-creatures.md` (prose pointer). | SRD 5.2.1, "Level 2: Wild Companion", p. 43 | unhappy (rule interaction) |
| FR-090 | The familiar summoned this way MUST be marked Fey, and finishing a Long Rest MUST emit `WildCompanionDismissed` and clear it; the creature's own departure is `ReportLongRestFinished` in `summoned-and-companion-creatures.md` (FR-053, FR-076), owned there. | SRD 5.2.1, "Level 2: Wild Companion", p. 43 | happy |
| FR-091 | Using it while in a Beast form below level 18 MUST be rejected with `CannotCastInBeastForm` (Q-7); from level 18 it is allowed, because the spell has no Material component when cast this way (FR-155 gives `MaterialComponentRestricted` no case here). | SRD 5.2.1, "Level 2: Wild Companion"; "Rules While Shape-Shifted" (No Spellcasting), p. 43 | unhappy (rule interaction) |

### 4.7 Wild Resurgence

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-096 | Wild Resurgence MUST be unavailable below level 5 (`FeatureNotGranted`). | SRD 5.2.1, "Level 5: Wild Resurgence", p. 43 | unhappy (boundary) |
| FR-097 | Exchanging a spell slot for a Wild Shape use MUST, when no uses remain, spend one slot of a chosen level, add one use and need no action. | SRD 5.2.1, "Level 5: Wild Resurgence", p. 43 | happy |
| FR-098 | That exchange MUST be rejected with `WildShapeUsesRemain` when at least one use remains, and with `NoSlotAvailable` (`slotRejected`, FR-088) when the chosen slot level has none; on success the result carries a `DruidSlotChangeSpend` for that level and the use is added in the returned state. | SRD 5.2.1, "Level 5: Wild Resurgence", p. 43 | unhappy (state error) |
| FR-099 | That exchange MUST be allowed once on each turn: a second one on the same turn MUST be rejected with `AlreadyUsedThisTurn` until a turn starts. | SRD 5.2.1, "Level 5: Wild Resurgence" ("Once on each of your turns"), p. 43 | unhappy (state error) |
| FR-100 | Exchanging a Wild Shape use for a level 1 spell slot MUST spend one use, return a `DruidSlotChangeRegain` of one level 1 slot (FR-160), need no action, and be rejected with `NoUsesRemaining` when no use remains (Q-4). | SRD 5.2.1, "Level 5: Wild Resurgence", p. 43 | happy |
| FR-101 | That exchange MUST be rejected with `NoExpendedSlot` (`slotRejected` carrying `SlotOperationRejection.noExpendedSlot`, FR-088) when no level 1 slot is expended in the given `SpellcastingState` (Q-4). | SRD 5.2.1, "Level 5: Wild Resurgence", p. 43 | unhappy (state error) |
| FR-102 | That exchange MUST be allowed once until a Long Rest is finished; a second MUST be rejected with `AlreadyUsedSinceLongRest`. | SRD 5.2.1, "Level 5: Wild Resurgence" ("can't do so again until you finish a Long Rest"), p. 43 | unhappy (state error) |
| FR-103 | The once-per-turn limit MUST apply only to the slot-for-use exchange and the once-per-Long-Rest limit only to the use-for-slot exchange; using one MUST NOT block the other. | SRD 5.2.1, "Level 5: Wild Resurgence", p. 43 | unhappy (rule interaction) |

### 4.8 Elemental Fury, Beast Spells, Epic Boon, Archdruid

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-104 | Elemental Fury MUST be a pending choice at level 7 between Potent Spellcasting and Primal Strike; before level 7 it MUST be unavailable, and a second choice MUST be rejected with `ElementalFuryAlreadyChosen` (Q-3). | SRD 5.2.1, "Level 7: Elemental Fury", p. 43 | unhappy (state error) |
| FR-105 | `DruidRules.potentSpellcastingBonus` (table M) MUST take the content `Spell` (a cantrip is `Spell.level.isCantrip`, content-spells.md FR-011) and the Wisdom `AbilityModifier`, and return `PotentBonusApplied(Modifier)` with the Wisdom modifier for a Druid cantrip, including cantrips that count as Druid spells for the character (Magician's and Circle Spells cantrips; Q-14), and `PotentBonusNotApplicable` (`NotADruidCantrip`) for a level 1+ spell or a cantrip of another class (table F; Q-5). | SRD 5.2.1, "Level 7: Elemental Fury" (Potent Spellcasting), p. 43; "Level 1: Spellcasting", p. 42 | unhappy (rule interaction) |
| FR-106 | Potent Spellcasting MUST NOT apply when Primal Strike was chosen, and Primal Strike MUST NOT apply when Potent Spellcasting was chosen (reason `FeatureNotChosen` in `PotentBonusNotApplicable`, `RangeChangeNotApplicable` and `PrimalStrikeOutcomeIneligible`). | SRD 5.2.1, "Level 7: Elemental Fury" ("one of the following options"), p. 43 | unhappy (rule interaction) |
| FR-107 | Primal Strike MUST, on a hit with a weapon attack (`PrimalStrikeAttackSource.weapon`) or a Beast form's attack while in Wild Shape (`beastForm`), return an extra 1d8 damage (2d8 from level 15) of a type chosen from Cold, Fire, Lightning or Thunder (table C). A weapon attack made while in Wild Shape is eligible (D-7). | SRD 5.2.1, "Level 7: Elemental Fury" (Primal Strike), "Level 15: Improved Elemental Fury", p. 43 | happy |
| FR-108 | The host MUST supply a `PrimalStrikeAttackSource` and the attack's `AttackRollResult` (d20-tests.md; its `hit`). Primal Strike MUST be rejected with `AttackNotEligible` for a miss, for `spell` and `other`, and for `beastForm` when the Druid is not in Wild Shape. | SRD 5.2.1, "Level 7: Elemental Fury" (Primal Strike), p. 43 | unhappy (invalid input) |
| FR-109 | Primal Strike MUST be allowed once on each of the Druid's turns; a second use the same turn MUST be rejected with `AlreadyUsedThisTurn` until a turn starts. | SRD 5.2.1, "Level 7: Elemental Fury" (Primal Strike), p. 43 | unhappy (state error) |
| FR-110 | `DruidRules.improvedPotentRange` (table M) takes the cantrip's `SpellRange` (content-spells.md FR-036) and MUST, from level 15 and only when Potent Spellcasting was chosen (otherwise `RangeChangeNotApplicable`, `FeatureNotChosen`), return `RangeChangeIncreased` adding 300 feet to a `distance` range of 10 feet or greater, and `RangeChangeUnchanged` for Self, Touch, shorter ranges and every other `SpellRange` variant; at level 14 it MUST add nothing (table C). | SRD 5.2.1, "Level 15: Improved Elemental Fury" (Potent Spellcasting), p. 43 | unhappy (boundary) |
| FR-111 | Beast Spells MUST, from level 18, allow casting in Beast form for any spell except one whose `MaterialComponent` has a non-empty `costsGp` or `consumesMaterial` true, which MUST be blocked with `MaterialComponentRestricted` (FR-155); at level 17 it MUST allow none (`CannotCastInBeastForm`; supersedes FR-082 from 18). | SRD 5.2.1, "Level 18: Beast Spells", p. 43 | unhappy (boundary) |
| FR-112 | Evergreen Wild Shape MUST, at level 20 when Initiative is rolled and no Wild Shape uses remain, regain one expended use; with a use remaining or below level 20 it changes nothing but the rest window. It is performed by `ApplyDruidTrigger(initiativeRolled)` (FR-154), which emits `WildShapeUsesChanged` with cause `evergreenWildShape`. It is passive, so it is not a feature use for FR-165. | SRD 5.2.1, "Level 20: Archdruid" (Evergreen Wild Shape), p. 43 | unhappy (boundary) |
| FR-113 | Nature Magician MUST, at level 20, convert a chosen number n of unexpended Wild Shape uses (1 to the uses remaining) into one spell slot of level 2n (a `DruidSlotChangeRegain`, FR-160), with no action, once until a Long Rest is finished (table I; Q-4). | SRD 5.2.1, "Level 20: Archdruid" (Nature Magician), p. 43 | happy |
| FR-114 | Nature Magician MUST be rejected for n of 0 (`InvalidConversion`), n above the uses remaining (`NoUsesRemaining`), a second use before a Long Rest (`AlreadyUsedSinceLongRest`), or when no slot of the resulting level is expended (`NoExpendedSlot`). Several at once: the order of D-5 (`FeatureNotGranted`, `AlreadyUsedSinceLongRest`, `InvalidConversion`, `NoUsesRemaining`, `NoExpendedSlot`; table I). | SRD 5.2.1, "Level 20: Archdruid" (Nature Magician), p. 43 | unhappy (boundary) |
| FR-115 | Longevity MUST be exposed as data: for every ten years that pass, the body ages one year. | SRD 5.2.1, "Level 20: Archdruid" (Longevity), p. 43 | happy |
| FR-116 | Beast Spells, Epic Boon and Archdruid features MUST be unavailable one level before they are granted (levels 17, 18 and 19; `FeatureNotGranted` for a use, no effect for the passive Beast Spells, Evergreen Wild Shape and Nature Magician). | SRD 5.2.1, Druid Features table, p. 41 | unhappy (boundary) |

### 4.9 Circle of the Land

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-117 | The Circle of the Land MUST grant its features at Druid levels 3 (Circle Spells, Land's Aid), 6 (Natural Recovery), 10 (Nature's Ward) and 14 (Nature's Sanctuary). | SRD 5.2.1, "Druid Subclass: Circle of the Land", p. 46 | happy |
| FR-118 | The land type MUST be one of arid, polar, temperate or tropical, chosen when a Long Rest is finished, that is while the `landTypeChange` permission is open (FR-153, FR-154); a choice consumes the permission. Choosing with the permission closed (other than the first choice, Q-6) MUST be rejected with `ChangeNotAllowedNow`. | SRD 5.2.1, "Level 3: Circle of the Land Spells", p. 46 | unhappy (state error) |
| FR-119 | The Circle Spells prepared MUST be the Circle Spells of the land type (`content-classes.md` data) at the Druid levels 3, 5, 7 and 9 that are at or below the Druid level (3 spells at levels 3-4, 4 at 5-6, 5 at 7-8, 6 from 9; level 4 gives only the level 3 row). | SRD 5.2.1, "Level 3: Circle of the Land Spells", p. 46 | unhappy (boundary) |
| FR-120 | Circle Spells MUST be always prepared: they MUST NOT count against the prepared-spell limit, MUST NOT be removable (`AlwaysPreparedNotReplaceable`) and MUST count as Druid spells. | SRD 5.2.1, "Level 3: Circle of the Land Spells", p. 46; "Level 1: Spellcasting", p. 42 | unhappy (state error) |
| FR-121 | Changing the land type MUST remove the previous land's Circle Spells and add the new land's, and emit `LandTypeChosen`. | SRD 5.2.1, "Level 3: Circle of the Land Spells", p. 46 | unhappy (rule interaction) |
| FR-122 | Land's Aid MUST be unavailable below level 3 or without the subclass (`FeatureNotGranted`). | SRD 5.2.1, "Level 3: Land's Aid", p. 46 | unhappy (boundary) |
| FR-123 | Land's Aid MUST spend one Wild Shape use and return: `DruidActionCost.magicAction`, a point within 60 feet, a 10-foot-radius Sphere, a Constitution saving throw against the caller-supplied `SpellSaveDc`, Necrotic damage with half on a success, and healing of one creature chosen in the area, with damage and healing dice from table C (2d6, 3d6 from level 10, 4d6 from level 14). | SRD 5.2.1, "Level 3: Land's Aid", p. 46 | happy |
| FR-124 | Land's Aid with no uses remaining MUST be rejected with `NoUsesRemaining` and leave the state unchanged. | SRD 5.2.1, "Level 3: Land's Aid", p. 46 | unhappy (state error) |
| FR-125 | Natural Recovery casting MUST allow casting one prepared Circle Spell of level 1 or higher without expending a slot, once until a Long Rest is finished. It consults `beastFormCastCheck` with the content `Spell` (FR-155): in Beast form it is rejected with `CannotCastInBeastForm` below level 18, and from level 18 with `MaterialComponentRestricted` when the spell has a costed or consumed Material component. | SRD 5.2.1, "Level 6: Natural Recovery", p. 46 | happy |
| FR-126 | That casting MUST be rejected with `NotACircleSpell` for a spell prepared from another source or a cantrip, and with `AlreadyUsedSinceLongRest` after use. | SRD 5.2.1, "Level 6: Natural Recovery", p. 46 | unhappy (invalid input) |
| FR-127 | Natural Recovery slot recovery MUST be possible only while the `naturalRecoveryRecovery` permission is open (opened by finishing a Short Rest, FR-153, FR-154), once until a Long Rest is finished; with the permission closed `RecoveryNotAllowedNow`, a second time `AlreadyUsedSinceLongRest`. | SRD 5.2.1, "Level 6: Natural Recovery", p. 46 | unhappy (state error) |
| FR-128 | The recovered slots MUST have a combined level of at most half the Druid level rounded up, and none MUST be level 6 or higher (table K); a violation MUST be rejected with `RecoveryBudgetExceeded` or `SlotLevelTooHigh`. | SRD 5.2.1, "Level 6: Natural Recovery", p. 46 | unhappy (boundary) |
| FR-129 | Only expended slots MUST be recoverable: recovering a slot not expended in the given `SpellcastingState` MUST be rejected with `NoExpendedSlot` (`slotRejected`, FR-088); a valid recovery returns one `DruidSlotChangeRegain` per recovered slot (FR-160). | SRD 5.2.1, "Level 6: Natural Recovery" ("expended spell slots"), p. 46 | unhappy (invalid input) |
| FR-130 | The casting use and the recovery use MUST have independent once-until-Long-Rest limits. | SRD 5.2.1, "Level 6: Natural Recovery", p. 46 | unhappy (rule interaction) |
| FR-131 | From level 10 with the subclass the Druid MUST be immune to the Poisoned condition (data, `ConditionKind.poisoned`) and have Resistance to the `DamageType` for the current land: Fire for arid, Cold for polar, Lightning for temperate, Poison for tropical (`content-classes.md` data). | SRD 5.2.1, "Level 10: Nature's Ward", p. 46 | happy |
| FR-132 | Nature's Ward resistance MUST change with the land type; with no land type chosen the resistance MUST be `LandTypeNotChosen` and the Poisoned immunity unaffected. | SRD 5.2.1, "Level 10: Nature's Ward", p. 46 | unhappy (missing content) |
| FR-133 | Nature's Sanctuary MUST spend one Wild Shape use and create a 15-foot Cube on the ground within 120 feet lasting 1 minute, with `DruidActionCost.magicAction`; it MUST be rejected with `NoUsesRemaining` when none remain and `FeatureNotGranted` below level 14 or without the subclass (FR-157). | SRD 5.2.1, "Level 14: Nature's Sanctuary", p. 46 | happy |
| FR-134 | While the Cube is active, the Druid and its allies in it MUST be reported as having Half Cover (a fact for the host; this spec defines no cover type) and the allies as having the current Nature's Ward Resistance (a `DamageType`). | SRD 5.2.1, "Level 14: Nature's Sanctuary", p. 46 | happy |
| FR-135 | The Cube MUST end when the minute elapses, or the Druid is Incapacitated or dies, with `NaturesSanctuaryEnded` and the reason. | SRD 5.2.1, "Level 14: Nature's Sanctuary", p. 46 | unhappy (state error) |
| FR-136 | Moving the Cube MUST cost a Bonus Action (`DruidActionCost.bonusAction`), move it at most 60 feet, and place it on ground within 120 feet of the Druid; 61 feet or 121 feet MUST be rejected with `MoveTooFar` and a move with no active Cube with `NoActiveSanctuary`. | SRD 5.2.1, "Level 14: Nature's Sanctuary", p. 46 | unhappy (boundary) |
| FR-137 | Creating a Cube while one is active MUST be rejected with `SanctuaryAlreadyActive` (Q-8). | SRD 5.2.1, "Level 14: Nature's Sanctuary", p. 46 | unhappy (state error) |
| FR-138 | Everything that depends on the land type (Circle Spells, Natural Recovery cast eligibility, Nature's Ward and Nature's Sanctuary resistance) MUST be derived from the land type currently in `DruidFeatureState`, never cached. Test: land `arid` chosen after Long Rest A, `polar` after Long Rest B; after B the Circle Spells are the polar set, Nature's Ward is Cold and a Sanctuary created after B gives Cold, with no arid spell or Fire result. | SRD 5.2.1, "Level 3: Circle of the Land Spells", "Level 10: Nature's Ward", p. 46 | unhappy (rule interaction) |

### 4.10 Cross-cutting, state and types

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-139 | Every class-feature scaling and limit in this spec (uses, duration, Temporary Hit Points, dice, budgets, cantrips, prepared spells) MUST use the Druid level, and slot counts come from the shared pool (FR-140; spellcasting.md FR-017; SRD 5.2.1, "Multiclassing" (Spell Slots), p. 25), so a Druid 3 / Fighter 5 has 2 Wild Shape uses, a 1.5 hour duration and 3 Temporary Hit Points. The exception is the Proficiency Bonus (FR-152). | SRD 5.2.1, Druid Features table, p. 41; "Rules While Shape-Shifted", p. 43; "Multiclassing", "Proficiency Bonus", p. 25 | unhappy (rule interaction) |
| FR-140 | Every operation that spends or restores spell slots MUST read the character's shared `SpellcastingState` (the pool named in the command), not a separate Druid pool, and its `DruidSlotChange` instructions MUST name only the levels it affects (FR-160); the Druid owns no pool. | SRD 5.2.1, "Becoming a Druid ... As a Multiclass Character", p. 41; `spellcasting.md` | unhappy (rule interaction) |
| FR-141 | The `DruidFeatureState` constructor, given the `DruidCharacterFacts` and `DruidFeatureTables` it is validated against (FR-167), MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error) when the remaining Wild Shape uses are negative or above the maximum for the level, when a feature state exists below the level that grants it (`WildShapeState` or `WildCompanionState` below level 2, a Wild Resurgence flag below 5, an `ElementalFury` or Primal Strike flag below 7, a Nature Magician flag below 20, a Natural Recovery flag, `LandType` or open `landTypeChange` / `naturalRecoveryRecovery` permission without the subclass or below level 3 / 6, a Cube below 14 or without the subclass), or when an active Beast form or Cube exists without the feature that creates it. | SRD 5.2.1, "Level 2: Wild Shape", p. 42 | unhappy (invalid input) |
| FR-142 | A non-panicking `NewDruidFeatureState` MUST return `(DruidFeatureState, error)` whose error joins the typed problems, listing every violated invariant (enum `DruidInvariantViolation`) for the same inputs, for hosts restoring stored state. | `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-147 | Every operation MUST be deterministic and use no dice: equal inputs give equal results, and dice are returned as expressions for the consumer to roll. | `doc/rules/architecture.md` §3 (Purity); `doc/rules/testing.md` §4 | unhappy (determinism) |
| FR-148 | No operation MUST modify its input state, and every returned collection MUST be a defensive copy. | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |
| FR-152 | No operation MUST derive the Proficiency Bonus from the Druid level. Every operation that needs it (`shapeShiftedModifier`, Land's Aid and Natural Recovery casts that need a save DC) takes a `ProficiencyBonus` from the caller, from total character level: a Druid 3 / Fighter 5 passes +3 although the single-class value at Druid level 3 is +2. | SRD 5.2.1, "Multiclassing", "Proficiency Bonus", p. 25; "Proficiency Bonus", p. 8 | unhappy (rule interaction) |
| FR-153 | `ApplyDruidTrigger(restFinished(kind))` MUST replace the open permissions in `DruidFeatureState.restWindow` (a set of `RestWindowPermission`). Finishing a Long Rest opens `knownFormReplacement` when Wild Shape is granted (level 2) and `landTypeChange` when the Druid has the subclass; finishing a Short Rest opens `naturalRecoveryRecovery` when Natural Recovery is granted (level 6 with the subclass). A permission whose feature is not granted MUST NOT be opened. | SRD 5.2.1, "Level 2: Wild Shape" (Known Forms), p. 42; "Level 3: Circle of the Land Spells", "Level 6: Natural Recovery", p. 46 | happy |
| FR-154 | All open permissions MUST be closed by `turnStarted`, by `initiativeRolled` and by the next `restFinished` (which then opens its own, FR-153). `ApplyDruidTrigger(initiativeRolled)` MUST additionally, for a Druid of level 20 with 0 Wild Shape uses remaining, regain exactly one use and emit `WildShapeUsesChanged` (from 0, to 1, cause `evergreenWildShape`, FR-112); otherwise it changes only the window (table N). It touches no other flag. A land type choice (FR-118) closes `landTypeChange`; Natural Recovery recovery (FR-127) closes `naturalRecoveryRecovery`; a known-form replacement leaves `knownFormReplacement` open and is limited by FR-076. Every rest-timed operation attempted with its permission closed MUST be rejected with its timing reason (`ChangeNotAllowedNow`, `RecoveryNotAllowedNow`) and leave the state unchanged. | `doc/rules/architecture.md` §3 (purity); D-4 | unhappy (state error) |
| FR-155 | `DruidRules.beastFormCastCheck(state, Spell)` (table M) MUST return `BeastFormCastCheckAllowed`, or `BeastFormCastCheckBlocked` with `CannotCastInBeastForm` (in a form, Druid level 17 or below) or `MaterialComponentRestricted` (in a form, level 18 or above, and `Spell.components.material` is present with a non-empty `MaterialComponent.costsGp` or `consumesMaterial` true). The host MUST call it before `CastSpell` and pass `CastingContext.canCastSpells: false` when blocked, so `CastSpell` rejects with `CannotCastSpells` (`casting-and-concentration.md` FR-113) and the host reports the Druid reason. Wild Companion (FR-091) and Natural Recovery casting (FR-125) call the same check inside `UseDruidFeature`. | SRD 5.2.1, "Rules While Shape-Shifted" (No Spellcasting), "Level 18: Beast Spells", p. 43; `casting-and-concentration.md` FR-113 | unhappy (state error) |
| FR-156 | `ApplyDruidTrigger(restFinished(long))` MUST clear every once-until-Long-Rest flag (Wild Resurgence use for slot, Natural Recovery casting, Natural Recovery recovery, Nature Magician) and the known-form replacement count, regain all expended Wild Shape uses (FR-069), and emit `WildCompanionDismissed` when a familiar is present (FR-090). | SRD 5.2.1, "Level 5: Wild Resurgence", "Level 20: Archdruid" (Nature Magician), p. 43; "Level 6: Natural Recovery", p. 46; "Level 2: Wild Shape" (Known Forms), p. 42 | happy |
| FR-157 | Each Circle of the Land feature MUST be unavailable below its level or without the subclass: Land's Aid (level 3), Natural Recovery (6), Nature's Ward (10), Nature's Sanctuary (14), and Circle Spells (3). A use operation MUST be rejected with `FeatureNotGranted`; the passive Nature's Ward and Circle Spells give nothing, with no error (table L). | SRD 5.2.1, "Druid Subclass: Circle of the Land", p. 46; "Level 3: Druid Subclass", p. 43 | unhappy (boundary) |
| FR-158 | `ApplyDruidTrigger(restFinished(short))` MUST regain exactly one expended Wild Shape use (FR-068) and MUST NOT clear any once-until-Long-Rest flag or the known-form replacement count, including after Natural Recovery recovery was used: a second recovery after a later Short Rest is rejected with `AlreadyUsedSinceLongRest`. | SRD 5.2.1, "Level 6: Natural Recovery" ("can't do so again until you finish a Long Rest"), p. 46 | unhappy (state error) |
| FR-159 | `ApplyDruidTrigger(turnStarted)` MUST clear exactly the two once-per-turn flags (Wild Resurgence slot for use, FR-099; Primal Strike, FR-109) and close the rest window (FR-154), and MUST change nothing else. | SRD 5.2.1, "Level 5: Wild Resurgence", "Level 7: Elemental Fury", p. 43 | unhappy (rule interaction) |
| FR-160 | `ApplyDruidTrigger` MUST NOT take or return spell slots; Long Rest slot restoration stays in `RestoreSpellSlots` (`spellcasting.md` FR-025 to FR-027). The Druid domain never calls a use case. `UseDruidFeature` takes a read-only `SpellcastingState` to check that a slot is available or expended, and returns the change as data: `DruidSlotChange` (closed: `DruidSlotChangeSpend(level)`, `DruidSlotChangeRegain(level)`), each carrying the pool named in the command, one instruction per slot (Wild Companion and Wild Resurgence spend; Wild Resurgence, Natural Recovery recovery and Nature Magician regain). The host applies each with `ExpendSpellSlot` or `RegainSpellSlot` (`spellcasting.md` FR-109, FR-110), which re-check and own `SlotOperationRejection`; this spec reports it only as `DruidRejection.slotRejected` with the value in a field. Availability is read through the public remaining-count and maximum accessors of `SpellcastingState` per level and pool (Q-19). | SRD 5.2.1, "Level 1: Spellcasting" (Spell Slots), p. 42; `spellcasting.md` | unhappy (rule interaction) |
| FR-161 | When more than one rejection applies to one operation, the reason MUST be the first in the total order of D-5, which places every `DruidRejection` member in exactly one group. Combined rows: tables I, K, P and B (replacement). | `character-aggregate.md` D-4; `doc/rules/architecture.md` §3 | unhappy (rule interaction) |
| FR-162 | The Druid's state MUST be one `DruidFeatureState` implementing `FeatureState`, stored in the character's `FeatureStates` under one typed `FeatureStateKey[DruidFeatureState]`. This spec defines the class and the key; the aggregate holds no Druid field. | shared-types-and-ownership.md FR-022 | happy |
| FR-163 | Druid creation choices MUST travel in `CreateCharacter` as a `DruidCreationChoices` (implementing `ClassCreationChoices`, character-aggregate.md FR-094) carrying the 2 skills (FR-004), the equipment package (FR-011) and the Primal Order (FR-048). A multiclass Druid carries only the Primal Order (FR-013). The skills and package are read by `character-creation.md` and are not feature state. | shared-types-and-ownership.md FR-024 | happy |
| FR-164 | Every Druid operation (`GainDruidLevel`, `ApplyDruidChoice`, `UseDruidFeature`, `ApplyDruidTrigger`) MUST be a pure domain function from (character facts, `DruidFeatureState`, inputs) to (new state, events), with no port, no `telemetry.Telemetry` and no use case: it needs neither dice (FR-147) nor content lookups (tables arrive as `DruidFeatureTables`, FR-168). Each returns its own closed `<Operation>Result` with operation-prefixed variants; the rejection reason is the `DruidRejection` enum held as a field, never a variant. | shared-types-and-ownership.md FR-017, FR-018, FR-023 | happy |
| FR-165 | This spec records no signal (shared-types-and-ownership.md FR-023, FR-063). It provides a pure mapping `DruidRules.featureAttribute(DruidFeatureUse) -> DruidFeatureAttribute (optional)` from each recordable use to the `srd5e.feature` value for the shared counter `srd5e.character.feature_used`: `wild_shape` (assume or end), `wild_companion`, `wild_resurgence` (either exchange), `lands_aid`, `natural_recovery` (cast or recovery), `natures_sanctuary` (create, move or end), `nature_magician`, `primal_strike`; null for passive effects. `srd5e.class` = `druid` is a constant of the mapping. The host, or the aggregate-level use case once character-aggregate.md D-15 is lifted, emits the counter with `srd5e.outcome` `success` or `failure` (shared values, FR-041); this spec does not itself emit it (Q-16). For a failure, `srd5e.character.rejection` (character-aggregate.md FR-118) is the snake_case of the `DruidRejection` member. | shared-types-and-ownership.md FR-041, FR-042, FR-063; character-aggregate.md D-15 | happy |
| FR-166 | Every Druid event MUST be a past-tense immutable value object implementing `CharacterEvent` (character-aggregate.md FR-085) and carry the `CharacterId` taken from `DruidCharacterFacts` (FR-167). Druid events need not be members of the closed `CharacterRecordEvent` family. This spec defines only the Druid events of section 5, none of `SubclassChosen`, `SubclassRejected`, `RestApplied`, `FightingStyleChoice` or `WeaponMasteryChoice`. Content ids are lowercase snake_case (`circle_of_the_land`). | shared-types-and-ownership.md FR-007, FR-019, FR-020; character-aggregate.md FR-085 | happy |
| FR-167 | Every Druid operation that depends on the Druid level, the subclass or the Proficiency Bonus MUST take them as `DruidCharacterFacts` (`CharacterId`, Druid level from `ClassLevels`, optional `SubclassId`, `ProficiencyBonus` from total level where needed, FR-152); `DruidFeatureState` stores none of them. | shared-types-and-ownership.md FR-008, FR-020 | unhappy (rule interaction) |
| FR-168 | `DruidFeatureTables` (Wild Shape uses by level, Beast Shapes tiers, Circle Spells by land and level, Nature's Ward resistance by land; no cantrip, prepared-spell or slot column) MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error) unless it has exactly 20 Wild Shape rows in level order, three Beast Shapes tiers and four lands each with levels 3, 5, 7 and 9; `NewDruidFeatureTables` MUST return `(DruidFeatureTables, error)` whose error joins the typed problems, listing every problem as a `DruidInvariantViolation`. | `doc/rules/architecture.md` §6; shared-types-and-ownership.md FR-025 | unhappy (invalid input) |
| FR-169 | `NewDruidCreationChoices(skills, equipmentPackage, primalOrder, entryKind)` MUST return a `DruidCreationChoicesCreation` (closed: `DruidCreationChoicesCreationCreated(choices)`, `DruidCreationChoicesCreationInvalid(reasons)`), each reason a `DruidRejection` member of FR-005, FR-006, FR-007 or FR-011, in the order of D-5. Skills and package are validated for a `startingClass` entry and must be absent for a `multiclass` entry (FR-013). `ClassCreationChoices` has no rejection channel (character-aggregate.md FR-094), so a host calls the `New` constructor (returns an error) first and passes only a created value to `CreateCharacter`. | character-aggregate.md FR-094; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-170 | The `ClassCreationChoices` method of `DruidCreationChoices` (a pure `FeatureStates -> FeatureStates`) MUST return the given states with a level 1 `DruidFeatureState` under the Druid key (FR-162) whose `PrimalOrder` is the chosen one and every other field at its default (no Wild Shape state, no flags, empty rest window), for a starting Druid and for a multiclass Druid at its first Druid level (FR-056). It seeds nothing from the skills or package. It MUST also implement the content-keys method (`contentKeys()`, character-aggregate.md FR-094) and return an empty unmodifiable list, because skills, packages and Primal Order name no content entries (`CreateCharacter` then looks nothing up, character-creation.md FR-103). | character-aggregate.md FR-094; shared-types-and-ownership.md FR-022, FR-024 | happy |
| FR-171 | `BeastFormFacts.fromMonster(Monster)` MUST map the content `Monster` to the facts the rules read: form id from `MonsterId`, `CreatureType` from `Monster.type`, `ChallengeRating` from `Monster.challengeRating`, and `hasFlySpeed` true iff any `SpeedEntry` has the fly mode (content-monsters.md FR-016, FR-089). Test rows: a walking Beast (false), a flying Beast (true), a swimming and climbing Beast (false), a non-Beast. | content-monsters.md FR-089, FR-030 | happy |
| FR-172 | `DruidFeatureTables.fromClassData(ClassDefinition, SubclassDefinition (optional))` MUST build the tables from the `content-classes.md` data: Wild Shape uses from the Druid `wild_shape` count column (table CC; "none" at level 1 is 0), Beast Shapes tiers from the `FeatureTable` of table TB (cell "1/4" parsed with `ParseChallengeRating`, "Yes" and "No" as true and false), Circle Spells from the `AlwaysPreparedSpells` facts of the four land options (table FA) and Nature's Ward resistance from its `FeatureTable` (`DamageType` spellings). Any cell it cannot parse MUST be a problem in the returned error, never a guess. | content-classes.md FR-033, FR-042, FR-066, FR-067, FR-060 | unhappy (invalid input) |
| FR-173 | `DruidRejection` MUST be an enum whose members are the camelCase of every reason named in this spec, so none collides in the package API (architecture.md §9); reasons shared in name with other specs (`NoUsesRemaining`, `DuplicateSkill`) are the enum's own members, not exported types. Every closed Druid value type prefixes its variants with its own type name, and a reason is held in a field, never reused as a variant name. A package API-scan test lists the exported Druid names (section 8). | shared-types-and-ownership.md FR-017, FR-021; `doc/rules/architecture.md` §9 | unhappy (name collision) |

### 4.11 Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (Druid `ClassDefinition`, Hit Point Die) |
| FR-002 | Withdrawn: moved to content-classes.md (primary ability as an `AbilityRequirement`) |
| FR-003 | Withdrawn: moved to content-classes.md (saving throws proficiencies) |
| FR-008 | Withdrawn: moved to content-classes.md (weapon, tool and armor training) |
| FR-009 | Withdrawn: moved to content-classes.md (starting equipment option A) |
| FR-010 | Withdrawn: moved to content-classes.md (starting equipment option B) |
| FR-014 | Withdrawn: moved to content-classes.md (`ProgressionRow` and `FeatureEntry`, features by level) |
| FR-015 | Withdrawn: moved to content-classes.md (`ProgressionRow` accessors and level bounds) |
| FR-016 | Withdrawn: moved to content-classes.md (`ProgressionRow` for levels with no feature) |
| FR-017 | Withdrawn: moved to content-classes.md (accumulation of features up to a level) |
| FR-018 | Withdrawn: moved to content-classes.md (Proficiency Bonus column; the character's own bonus is `derived-stats.md` `ProficiencyBonus`, caller-supplied here, FR-139, FR-152) |
| FR-019 | Withdrawn: moved to content-classes.md (`ClassDefinition.featGrantsAt`, content-classes.md FR-049) and feats-behaviour.md; this spec defines no feat-slot type |
| FR-020 | Withdrawn: moved to feats-behaviour.md and level-advancement-and-multiclassing.md (`SlotAlreadyFilled`, `SlotNotGranted`, `FeatSlotFilled` dropped; Q-18) |
| FR-021 | Withdrawn: moved to level-advancement-and-multiclassing.md (subclass choice and feature slots, `SubclassChosen`) |
| FR-022 | Withdrawn: moved to content-classes.md (level 3 Druid Subclass feature) |
| FR-028 | Withdrawn: moved to spellcasting.md (FR-014; cantrips known column) |
| FR-029 | Withdrawn: moved to spellcasting.md (`ChangePreparedSpells`, cantrip replacement) |
| FR-032 | Withdrawn: moved to content-classes.md (`ProgressionRow` spell slot columns) |
| FR-033 | Withdrawn: moved to content-classes.md (cantrips known column) |
| FR-034 | Withdrawn: moved to content-classes.md (prepared spells column) |
| FR-036 | Withdrawn: moved to content-classes.md (Druid spell list data and its `Spell.classes` consistency test) |
| FR-043 | Withdrawn: dropped; the recommended starting picks (SRD 5.2.1, p. 42) are descriptive text, not rules |
| FR-092 | Withdrawn: moved to level-advancement-and-multiclassing.md (subclass choice below the subclass level) |
| FR-093 | Withdrawn: moved to level-advancement-and-multiclassing.md (subclass must belong to the class) |
| FR-094 | Withdrawn: moved to level-advancement-and-multiclassing.md (second subclass choice) |
| FR-095 | Withdrawn: moved to level-advancement-and-multiclassing.md (subclass features on a late choice) |
| FR-143 | Withdrawn: moved to content-classes.md (validation of the decoded Druid class entry); behaviour inputs are FR-168 |
| FR-144 | Withdrawn: moved to content-classes.md and content-schema-and-decoder.md (missing class or subclass entry) |
| FR-145 | Withdrawn: moved to content-classes.md (class loading does not require the spells it names) |
| FR-146 | Withdrawn: moved to content-classes.md (no load use case here) |
| FR-149 | Withdrawn: no use cases here, no signals (shared-types-and-ownership.md FR-023; mapping is FR-165) |
| FR-150 | Withdrawn: no use cases here, no logs (shared-types-and-ownership.md FR-023) |
| FR-151 | Withdrawn: no use cases here; programmer errors propagate unchanged (FR-141) |

### Data tables (case tables)

The Druid progression, spell list and Circle of the Land tables are `content-classes.md` data; its progression, spell-list and
land-table tests are the oracle. The Proficiency Bonus is the single-class value: a Druid 3 / Fighter 5 has total level 8
and +3 (FR-152), not the +2 of Druid level 3. The cells FR-038, FR-039, FR-042 and FR-058 quote are the only ones restated here.

### Table B: Beast Shapes (SRD 5.2.1, p. 42), for FR-072 to FR-077

| Label | Druid level | Known forms | Max CR | Fly Speed allowed | Path |
|---|---|---|---|---|---|
| tier 1 low, high | 2, 3 | 4 | 1/4 | no | happy, unhappy (boundary) |
| tier 2 low, high | 4, 7 | 6 | 1/2 | no | unhappy (boundary) |
| tier 3 low, high | 8, 20 | 8 | 1 | yes | unhappy (boundary), happy |

Eligibility cases (hypothetical Beast facts), FR-073 and FR-074:

| Label | Druid level | Creature type | CR | Fly Speed | Expected | Path |
|---|---|---|---|---|---|---|
| Beast at max CR | 2 | Beast | 1/4 | no | eligible | unhappy (boundary) |
| one CR step over | 2 | Beast | 1/2 | no | `ChallengeRatingTooHigh` | unhappy (boundary) |
| flying below 8; at 8 | 7; 8 | Beast | 1/2; 1 | yes | `FlySpeedNotAllowed`; eligible | unhappy (boundary) |
| not a Beast | 8 | other type | 1/4 | no | `NotABeast` | unhappy (invalid input) |
| not a Beast and too high | 2 | other type | 1 | no | `NotABeast` | unhappy (rule interaction) |
| too high and flying | 4 | Beast | 2 | yes | `ChallengeRatingTooHigh` | unhappy (rule interaction) |

Replacement cases with several reasons (FR-076, FR-077, D-5), Druid level 8, Wolf known; a CR 2 Beast is over the maximum:

| Label | Window open | Replaces | With | Expected | Path |
|---|---|---|---|---|---|
| closed and unknown | no | Bear (not known) | CR 1 Beast | `ChangeNotAllowedNow` (timing first) | unhappy (rule interaction) |
| unknown and ineligible | yes | Bear (not known) | CR 2 Beast | `FormNotKnown` (known-ness before eligibility) | unhappy (rule interaction) |
| ineligible and duplicate | yes | Wolf | already known CR 2 Beast | `ChallengeRatingTooHigh` (eligibility before `DuplicateForm`) | unhappy (rule interaction) |
| duplicate | yes | Wolf | already known CR 1 Beast | `DuplicateForm` | unhappy (invalid input) |
| second replacement and ineligible | yes, one used | Wolf | CR 2 Beast | `ChallengeRatingTooHigh` (arguments before `TooManyReplacements`) | unhappy (rule interaction) |
| second replacement | yes, one used | Wolf | eligible CR 1 Beast | `TooManyReplacements` | unhappy (state error) |

### Table C: Values by Druid level (FR-065, FR-070, FR-107, FR-110, FR-123)

Land's Aid is unavailable below level 3; the Improved Potent Spellcasting rows use the cantrip's `SpellRange`.

| Label | Druid level | Input | Expected | Path |
|---|---|---|---|---|
| first level, odd level | 2, 3 | Wild Shape | 1 hour, 2 Temporary Hit Points; 1 hour 30 minutes, 3 | happy, unhappy (boundary) |
| middle | 10 | Wild Shape | 5 hours, 10 | happy |
| one below top, top | 19, 20 | Wild Shape | 9 hours 30 minutes, 19; 10 hours, 20 | unhappy (boundary), happy |
| multiclass | 3 (Druid 3 / Fighter 5) | Wild Shape | 1 hour 30 minutes, 3 | unhappy (rule interaction) |
| before Elemental Fury | 6 | Primal Strike; Land's Aid | unavailable; 2d6 | unhappy (boundary) |
| Elemental Fury, one below 10 | 7, 9 | Primal Strike; Land's Aid | 1d8; 2d6 | happy, unhappy (boundary) |
| level 10, one below 14 | 10, 13 | Primal Strike; Land's Aid | 1d8; 3d6 | unhappy (boundary) |
| level 14 | 14 | Primal Strike; Land's Aid | 1d8; 4d6 | unhappy (boundary) |
| level 15, top | 15, 20 | Primal Strike; Land's Aid | 2d8; 4d6 | unhappy (boundary), happy |
| before level 15 | 14 | range 60 feet | 60 feet | unhappy (boundary) |
| Self, Touch, below 10 feet | 15 | range Self; Touch; 5 feet | unchanged | unhappy (boundary) |
| exactly 10 feet | 15 | range 10 feet | 310 feet | unhappy (boundary) |
| 60 feet | 15 | range 60 feet | 360 feet | happy |
| 120 feet at 20 | 20 | range 120 feet | 420 feet | happy |
| Primal Strike chosen | 15 | range 60 feet | 60 feet (`FeatureNotChosen`) | unhappy (rule interaction) |

### Table D: Magician (FR-050, FR-052)

Check bonus by Wisdom score (modifier): 1 (-5), 8 (-1), 10 (0) and 12 (+1) give +1 (the minimum, unhappy boundary); 14 (+2) gives +2; 20 (+5) gives +5; 30 (+10) gives +10. Extra cantrips (FR-050): Druid level 1 gives 2 + 1 = 3, level 3 gives 3, level 4 gives 3 + 1 = 4, level 9 gives 4, level 10 gives 4 + 1 = 5, level 20 gives 5; a Warden has 2, 3 and 4 at those levels.

### Table E: Shape-shifted skill and saving throws modifiers (FR-080, FR-081)

Inputs: the ability modifier read from the form's scores (Intelligence, Wisdom and Charisma from the Druid, others from
the Beast; Q-11), whether the Druid is proficient, the Proficiency Bonus and the Beast stat block's modifier.

| Label | Ability modifier | Proficient | Proficiency Bonus | Stat block modifier | Result | Path |
|---|---|---|---|---|---|---|
| not proficient; proficient, none listed | +1; +2 | no; yes | +2 | none | +1; +4 | happy |
| stat block higher | +2 | yes | +3 | +6 | +6 | unhappy (boundary) |
| own higher | +2 | yes | +3 | +4 | +5 | unhappy (boundary) |
| equal | +2 | yes | +3 | +5 | +5 | unhappy (boundary) |
| not proficient, stat block listed | 0 | no | +3 | +3 | +3 | unhappy (rule interaction) |
| proficient in both | +1 | yes | +2 | +3 | +3 (bonus counted once) | unhappy (rule interaction) |
| Druid 3 / Fighter 5 | +2 | yes | +3 (total level 8, FR-152) | none | +5 (a Druid-level +2 would give +4) | unhappy (rule interaction) |

### Table F: Potent Spellcasting damage (FR-105, FR-106)

Wisdom modifier +3 unless stated: Druid cantrip +3 (happy); modifier 0 gives 0, -1 gives -1 (Q-5); a Magician extra cantrip and a Circle Spells cantrip give the modifier (+2 shown); a level 1+ Druid spell and a cantrip of another class give 0; with Primal Strike chosen a Druid cantrip gives 0 (`FeatureNotChosen`).

### Table I: Nature Magician (FR-113, FR-114)

| Label | Uses remaining | Uses converted | Slot level created | Expended slot of that level | Result | Path |
|---|---|---|---|---|---|---|
| one use | 4 | 1 | 2 | yes | level 2 slot restored, 3 uses left | happy |
| two uses; all uses | 4 | 2; 4 | 4; 8 | yes | level 4 slot restored (the SRD's example); level 8 slot restored, 0 uses left | happy, unhappy (boundary) |
| none | 4 | 0 | none | n/a | `InvalidConversion` | unhappy (boundary) |
| more than remain | 3 | 4 | 8 | yes | `NoUsesRemaining` | unhappy (boundary) |
| no expended slot | 4 | 2 | 4 | no | `NoExpendedSlot` | unhappy (state error) |
| used already | 4 | 1 | 2 | yes | `AlreadyUsedSinceLongRest` | unhappy (state error) |
| zero and used already | 4 | 0 | none | n/a | `AlreadyUsedSinceLongRest` (D-5 order) | unhappy (rule interaction) |
| too many and no expended slot | 3 | 4 | 8 | no | `NoUsesRemaining` (D-5 order) | unhappy (rule interaction) |
| zero and no expended slot | 4 | 0 | none | no | `InvalidConversion` (D-5 order) | unhappy (rule interaction) |

### Table K: Natural Recovery slot recovery (FR-128)

The budget is half the Druid level rounded up.

| Label | Druid level | Budget | Slots chosen (levels) | Expended | Result | Path |
|---|---|---|---|---|---|---|
| SRD examples | 6 | 3 | 3; or 2, 1; or 1, 1, 1 | yes | each accepted | happy |
| one over the budget | 6 | 3 | 2, 2 | yes | `RecoveryBudgetExceeded` | unhappy (boundary) |
| odd level rounds up; one over | 7 | 4 | 3, 1; 3, 2 | yes | accepted; `RecoveryBudgetExceeded` | unhappy (boundary) |
| level 5 slot; level 6 slot | 10; 20 | 5; 10 | 5; 6 | yes | accepted; `SlotLevelTooHigh` | unhappy (boundary) |
| not expended | 6 | 3 | 1, 1 | one only | `NoExpendedSlot` | unhappy (invalid input) |
| no slots chosen | 6 | 3 | none | n/a | accepted, nothing recovered | unhappy (empty and minimal) |
| over budget and level 6 | 20 | 10 | 6, 5, 5 | yes | `SlotLevelTooHigh` (D-5 order) | unhappy (rule interaction) |
| over budget and not expended | 6 | 3 | 2, 2 | one only | `RecoveryBudgetExceeded` (D-5 order) | unhappy (rule interaction) |
| permission closed and not expended | 6 | 3 | 1 | no | `RecoveryNotAllowedNow` (D-5 order) | unhappy (rule interaction) |

### Table L: Circle of the Land availability (FR-157)

For each of Land's Aid (level 3), Natural Recovery (6, cast and recovery), Nature's Ward (10) and Nature's Sanctuary (14),
three cases: one level below with the subclass (boundary) and at the level without it (state error) give `FeatureNotGranted`
for a use (Ward: no immunity, no resistance, no error); at the level with the subclass the feature is available (happy).

### Table N: Triggers, flags and windows (FR-153 to FR-159)

Flags: WR-slot (Wild Resurgence use for slot), NR-cast, NR-recover, NM (Nature Magician), replacements (known-form
replacement count), WR-use (slot for use), PS (Primal Strike). "keeps" means unchanged. Triggers are the closed
`DruidTrigger` variants, written here as `restFinished(kind)`, `turnStarted`, `initiativeRolled`.

| Label | Trigger | WR-slot | NR-cast | NR-recover | NM | replacements | WR-use | PS | Window | Path |
|---|---|---|---|---|---|---|---|---|---|---|
| Long Rest clears | `restFinished(long)` | cleared | cleared | cleared | cleared | 0 | keeps | keeps | opens `knownFormReplacement`, `landTypeChange` | happy |
| Short Rest clears none | `restFinished(short)` | keeps | keeps | keeps (still used) | keeps | keeps | keeps | keeps | opens `naturalRecoveryRecovery`; a recovery after an earlier one is rejected `AlreadyUsedSinceLongRest` | unhappy (rule interaction, state error) |
| turn starts | `turnStarted` | keeps | keeps | keeps | keeps | keeps | cleared | cleared | closed | unhappy (rule interaction) |
| initiative rolled, level 19 | `initiativeRolled`, level 19, 0 uses | keeps | keeps | keeps | keeps | keeps | keeps | keeps | closed; uses stay 0, no event | unhappy (boundary) |
| initiative rolled, level 20, none left | `initiativeRolled`, level 20, 0 uses | keeps | keeps | keeps | keeps | keeps | keeps | keeps | closed; uses 0 to 1, `WildShapeUsesChanged(evergreenWildShape)` (FR-112) | happy |
| initiative rolled, level 20, uses left | `initiativeRolled`, level 20, 1 or more uses | keeps | keeps | keeps | keeps | keeps | keeps | keeps | closed; uses unchanged, no event | unhappy (rule interaction) |
| operation outside its window | land type choice with no Long Rest; recovery with no Short Rest; form replacement after `turnStarted` | n/a | n/a | n/a | n/a | n/a | n/a | n/a | `ChangeNotAllowedNow`; `RecoveryNotAllowedNow`; `ChangeNotAllowedNow` | unhappy (state error) |
| second replacement | form replacement twice in one Long Rest window | n/a | n/a | n/a | n/a | 1 | n/a | n/a | second: `TooManyReplacements` | unhappy (state error) |
| feature not granted | Long Rest at level 1 (no Wild Shape) | n/a | n/a | n/a | n/a | n/a | n/a | n/a | `knownFormReplacement` not opened | unhappy (boundary) |

### Table O: Switching Beast forms (FR-063, FR-071)

| Label | Uses remaining | Active form | Result | Events in order | Path |
|---|---|---|---|---|---|
| switch | 2 | Wolf | 1 use left, active form Riding Horse | `WildShapeEnded(usedWildShapeAgain)`, `WildShapeAssumed` (Temporary Hit Points = Druid level) | happy |
| switch with none left | 0 | Wolf | `NoUsesRemaining`, Wolf stays active, state unchanged | none | unhappy (state error) |
| switch to unknown form | 2 | Wolf | `FormNotKnown`, Wolf stays active | none | unhappy (invalid input) |
| switch to the same form | 2 | Wolf | allowed: one use spent, ends and re-assumes Wolf | `WildShapeEnded(usedWildShapeAgain)`, `WildShapeAssumed` | unhappy (boundary) |
| assume from no form | 2 | none | 1 use left | `WildShapeAssumed` only | happy |

### Table P: Operations with several reasons (FR-087, FR-088, FR-091, FR-104 to FR-109, FR-133 to FR-137, D-5)

| Label | Operation | Setup | Expected | Path |
|---|---|---|---|---|
| not granted | Wild Companion | level 1, not in a form, slot available | `FeatureNotGranted` | unhappy (boundary) |
| form below 18 and no slot | Wild Companion | level 10, in a form, slot payment, none left | `CannotCastInBeastForm` (casting before resources) | unhappy (rule interaction) |
| form below 18 and no uses | Wild Companion | level 10, in a form, Wild Shape payment, 0 uses | `CannotCastInBeastForm` | unhappy (rule interaction) |
| form at 18, no slot | Wild Companion | level 18, in a form, slot payment, none left | `NoSlotAvailable` (the spell has no Material component) | unhappy (rule interaction) |
| no form, no uses | Wild Companion | level 10, not in a form, Wild Shape payment, 0 uses | `NoUsesRemaining` | unhappy (state error) |
| form at 18, paid by use | Wild Companion | level 18, in a form, 1 use | accepted | unhappy (boundary) |
| not granted | Primal Strike | level 6, `weapon`, hit | `FeatureNotGranted` | unhappy (boundary) |
| other option chosen, miss | Primal Strike | level 7, Potent Spellcasting, `weapon`, miss | `FeatureNotChosen` (availability before argument) | unhappy (rule interaction) |
| second use and a miss; second use, spell source | Primal Strike | level 7, used this turn, `weapon` miss; `spell` hit | `AlreadyUsedThisTurn` (timing before `AttackNotEligible`) | unhappy (rule interaction) |
| first use, miss | Primal Strike | level 7, `weapon`, miss | `AttackNotEligible` | unhappy (state error) |
| first use, Beast attack, not in form | Primal Strike | level 7, `beastForm`, hit | `AttackNotEligible` | unhappy (invalid input) |
| first use, weapon attack in form | Primal Strike | level 7, `weapon`, hit, in Wild Shape | 1d8 (D-7) | happy |
| not granted and Cube active | Sanctuary create | level 14, no subclass, Cube active, 1 use | `FeatureNotGranted` | unhappy (rule interaction) |
| Cube active and no uses | Sanctuary create | level 14, subclass, Cube active, 0 uses | `SanctuaryAlreadyActive` (state before resources) | unhappy (rule interaction) |
| no uses, no Cube | Sanctuary create | level 14, subclass, no Cube, 0 uses | `NoUsesRemaining` | unhappy (state error) |
| move without a Cube, too far | Sanctuary move | level 14, subclass, no Cube, 61 feet | `NoActiveSanctuary` (state before argument) | unhappy (rule interaction) |
| move 61 feet; 60 feet | Sanctuary move | Cube active | `MoveTooFar`; accepted | unhappy (boundary) |

## 5. Model

`DruidFeatureTables` is an immutable plain value (20 Wild Shape uses rows, 3 `BeastShapeTier`s, `CircleOfTheLandTables`), so
every rule is testable with a hand-built instance (FR-168, FR-172). `Concentration`, slot operations,
`SlotOperationRejection` and feat slots are not defined here.

**New value objects (domain):** `PrimalOrder`, `ElementalFury`, `LandType`, `DruidActionCost`, `BeastFormFacts`, `KnownForms`
(unmodifiable ordered set of form ids), `DruidItemKey`, `EquipmentDisposition` (FR-084), `DruidCharacterFacts` (`CharacterId`,
Druid level, optional `SubclassId`, `ProficiencyBonus`), `DruidCreationChoices` and its `...Creation` result, `DruidSlotChange`,
`RestWindowPermission`, `DruidInvariantViolation`, `DruidFeatureAttribute` and the closed outputs of table M.
`DruidRejection` is an **enum** of typed reasons with no text (FR-173).

`DruidFeatureState` (FR-162) holds optional `PrimalOrder`, `ElementalFury` and `LandType`, `WildShapeState` (uses remaining,
`KnownForms`, active form, replacements used since the last Long Rest), the once-per-turn flags (Wild Resurgence
exchange, Primal Strike), the once-until-Long-Rest flags (Wild Resurgence use for slot, Natural Recovery casting and
recovery, Nature Magician), `restWindow` (empty by default), `WildCompanionState` and the active Cube. It holds no feat
slots and neither the Druid level nor the subclass (FR-167). `DruidRules` is a stateless domain service.

**Events** (immutable, past tense, each a `CharacterEvent` with the `CharacterId`, FR-166): `DruidLevelGained`,
`PrimalOrderChosen`, `ElementalFuryChosen`, `LandTypeChosen`, `KnownFormsChosen`, `KnownFormReplaced`, `WildShapeAssumed`
(form, duration, Temporary Hit Points amount, worn and merged `DruidItemKey` lists, `actionCost`), `WildShapeEnded`,
`WildCompanionCast`, `WildCompanionDismissed`, `WildShapeUsesChanged` (from, to, cause), `SpellSlotsExchanged`,
`LandsAidUsed`, `NaturalRecoveryUsed`, `NaturesSanctuaryCreated`, `NaturesSanctuaryMoved`, `NaturesSanctuaryEnded`,
`PrimalStrikeUsed`.

**Commands (all variants prefixed with the type name, FR-173).** `DruidChoice` (Primal Order, Elemental Fury, land type,
known forms, form replacement: `DruidChoiceLandType`); `DruidFeatureUse` (assume and end Wild Shape, Wild Companion, both
Wild Resurgence exchanges, Land's Aid, Natural Recovery cast and recovery, Sanctuary create, move and end, Nature
Magician, Primal Strike: `DruidUseLandsAid`); `DruidTrigger` (`DruidTriggerRestFinished(kind)`, `DruidTriggerTurnStarted`,
`DruidTriggerInitiativeRolled`). Results are `<Operation>Applied` / `<Operation>Rejected` (`UseDruidFeatureRejected` adds an
optional `SlotOperationRejection`; `ApplyDruidTriggerApplied` has no rejected variant).

**Invariants:** remaining uses 0 to the level maximum; every active form, Cube or open permission implies its feature is
granted (FR-141); known forms are distinct and within the allowed number; tables are complete (FR-168). All panics
panic (`doc/rules/architecture.md` §6). Expected outcomes from host input are rejection results holding a
`DruidRejection` member, with no display text. The domain does no I/O: a missing class entry is the decoder's failure.

## 6. Interface

No application layer (shared-types-and-ownership.md FR-023): each operation is a pure domain function over immutable
inputs (FR-164). The **host** (or a character use case that character-aggregate.md D-15 defers) stores the returned
`DruidFeatureState` in `FeatureStates`, publishes the events and applies `DruidSlotChange` instructions with
`ExpendSpellSlot` and `RegainSpellSlot`. Rejections are results, never errors.

| Operation | Input | Result (closed) | Sync / takes ctx |
|---|---|---|---|
| `GainDruidLevel` | optional `DruidFeatureState`, `DruidFeatureTables`, `DruidCharacterFacts` (target level), entry kind (`startingClass`, `multiclass`) | `GainDruidLevelApplied` (state, features granted, Druid-owned pending choices, events), `GainDruidLevelRejected` | sync, pure |
| `ApplyDruidChoice` | state, tables, facts, `DruidChoice` (forms as `BeastFormFacts`) | `ApplyDruidChoiceApplied` (state, events), `ApplyDruidChoiceRejected` | sync, pure |
| `UseDruidFeature` | state, tables, facts, the character's `SpellcastingState` (read only), `DruidFeatureUse` with the facts it needs (the content `Spell` for casts) | `UseDruidFeatureUsed` (state, `DruidSlotChange` list, effect data such as dice and DC, events), `UseDruidFeatureRejected` | sync, pure |
| `ApplyDruidTrigger` | state, tables, facts, `DruidTrigger` | `ApplyDruidTriggerApplied` (state, events) | sync, pure |

Creation choices are built with `NewDruidCreationChoices` (FR-169); only a created value goes to
`CreateCharacter` (FR-170). `restFinished(kind)` opens a window in `restWindow` that `ApplyDruidChoice` and `UseDruidFeature`
consume as FR-154 states (D-4). Cantrips and prepared spells are not a `DruidChoice`: the host calls `ChangePreparedSpells`
(FR-038 to FR-041, FR-051).

**Table M: pure `DruidRules` methods** (synchronous, total; none reads state it is not given).

| Method | Inputs | Output | FRs |
|---|---|---|---|
| `MagicianCheckBonus` | optional `PrimalOrder`, Wisdom `AbilityModifier`, `Ability`, `Skill` | `Modifier` (zero, or at least +1 for Intelligence (Arcana) and (Nature) with Magician) | FR-052, FR-053 |
| `ShapeShiftedStats` | none | `RetainedStats`, `ReplacedStats` | FR-079 |
| `ShapeShiftedModifier` | `AbilityModifier`, `ShapeShiftedProficiency` (`proficient`, `notProficient`), `ProficiencyBonus` (total level, FR-152), Beast optional `Modifier` | `ShapeShiftedModifier` (`Modifier`, source `druid` or `statBlock`) | FR-080, FR-081 |
| `BeastFormCastCheck` | `DruidFeatureState`, content `Spell` | `BeastFormCastCheckAllowed` or `BeastFormCastCheckBlocked(reason)`: `CannotCastInBeastForm`, `MaterialComponentRestricted` | FR-082, FR-111, FR-155 |
| `PotentSpellcastingBonus` | state, `Spell`, Wisdom `AbilityModifier`, `SpellId`s that count as Druid spells through features | `PotentBonusApplied(Modifier)` or `PotentBonusNotApplicable(reason)`: `FeatureNotChosen`, `NotADruidCantrip` | FR-105, FR-106 |
| `ImprovedPotentRange` | state, Druid level, the cantrip's `SpellRange` | `RangeChangeIncreased(newFeet)`, `RangeChangeUnchanged`, `RangeChangeNotApplicable(reason)`: `FeatureNotChosen` | FR-110 |
| `PrimalStrikeDice` | state, `PrimalStrikeAttackSource` (`weapon`, `beastForm`, `spell`, `other`), `AttackRollResult` | `PrimalStrikeOutcomeDamage(dice, choosable types)` or `PrimalStrikeOutcomeIneligible(reason)`: `AttackNotEligible`, `FeatureNotChosen` | FR-106 to FR-108 |
| `LandsAidDice` | Druid level | dice expression | FR-123 |
| `NaturalRecoveryBudget` | Druid level | `int` (half, rounded up) | FR-128 |
| `CircleSpells` | `LandType`, Druid level | unmodifiable set of `SpellId` | FR-119 |
| `FormEligibility` | `BeastFormFacts`, tables, Druid level | `FormEligibilityEligible` or `FormEligibilityIneligible(reason)`: `NotABeast`, `ChallengeRatingTooHigh`, `FlySpeedNotAllowed` | FR-073, FR-074 |
| `DruidicDetection` | `Set<Language>` known | `DruidicDetection` | FR-045 to FR-047 |
| `FeatureAttribute` | `DruidFeatureUse` | optional `DruidFeatureAttribute` | FR-165 |

The host asks `DruidRules` before `CastSpell` (FR-155) and applies `potentSpellcastingBonus` and `improvedPotentRange`
when building a cantrip's damage and range; `UseDruidFeature` calls the same methods for feature uses that are casts.
**Ports used:** none (FR-147, FR-164).

**Exports.** the `character` package exports the types above and the four operations with their commands;
the module's public packages re-exports them. No generic variant name (`Applied`, `Rejected`, `ContentMissing`, `Eligible`,
`Readable`; shared-types-and-ownership.md FR-017) and none of `ConcentrationEffect`, `AttackKind`, `ActionCost`,
`PotentSpellcasting`, `FeatChoiceSlot`, `SpellFacts` is exported (FR-173).

## 7. telemetry.Telemetry

Silent by design: pure domain code with no `telemetry.Telemetry`, span, metric or log (`doc/rules/observability.md` §3;
shared-types-and-ownership.md FR-023). Per FR-063 it defines no signal of its own: it uses the shared
`srd5e.character.feature_used` counter and the shared `.duration` and `.errors` signals of `character-aggregate.md`,
with `srd5e.class` = `druid`, `srd5e.feature` as mapped by `DruidRules.featureAttribute` and the rejection value of
FR-165. The earlier `srd5e.character.druid.*` names were never shipped. What can go wrong is a rejection (a typed
result) or a bug (panic); the host that applies the result owns recording, and until a host emits it
(Q-16) no Druid use is counted. Once emitted, monitor the `failure` share per `srd5e.feature`: a sustained rise points
to a host not tracking resources or turns. Nothing here logs, and no `doc/observability/signals.md` entry is added.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-004 to FR-007, FR-011, FR-169, FR-170 | unit | the `New` constructor (returns an error) table: counts 0/1/2/3, duplicate, not on list, unknown package, several at once in D-5 order, multiclass with skills; seeding stores a level 1 state with the chosen Primal Order only; the content-keys method returns an empty unmodifiable list |
| FR-012, FR-013, FR-023 to FR-027, FR-030, FR-031, FR-035, FR-037 to FR-042, FR-051 | unit | starting vs multiclass grants from a hand-built `ClassDefinition`; levels 0 to 1, 1 to 3 (rejected), 20 to 21; pending choices per level; Druid inputs to `ChangePreparedSpells`; fed the Druid `ClassLevels`, the values `spellcasting.md` derives and its `PreparationRejection`s; Druid 3 / Fighter 5 |
| FR-044 to FR-078 | unit | with Druidic, spotted and readable; without, `DruidicPresenceRequiresCheck(15, Intelligence, Investigation)` and `DruidicContentNeedsMagic`; table D; Magician and Warden grants; multiclass; tables B, C, O; rest and end-of-form transitions; eligibility and replacement rows; timing through the rest window (table N) |
| FR-079 to FR-085, FR-152, FR-155 | unit | retained and replaced stats; table E; `beastFormCastCheck` at 17 and 18 with a costed, a consumed and a plain Material component; no Concentration input or output; dispositions, `ItemUnassigned`, `ItemCannotBeWorn` |
| FR-086 to FR-091, FR-096 to FR-103 | unit | payment by slot (`DruidSlotChangeSpend`, `slotRejected`) and by use; Long Rest dismissal; turn and Long Rest resets; table P; `SpellcastingState` input unchanged |
| FR-104 to FR-116 | unit | tables C, F, I, P; weapon attack in Wild Shape (D-7); Beast Spells at 17 and 18; Evergreen Wild Shape via `initiativeRolled` at 19 and 20; `Modifier`, `Spell`, `SpellRange` typing |
| FR-117 to FR-138, FR-157 | unit | every land at levels 2 (none), 3, 4, 5, 6, 7, 8, 9 and 20 against the `content-classes.md` land data; tables K, L, P; Land's Aid dice and costs; independent limits; Sanctuary move at 60, 61, 120, 121 feet and end reasons; the FR-138 land A then B test |
| FR-153, FR-154, FR-156, FR-158 to FR-161 | unit | table N row per trigger and flag; choice and recovery outside a window; `ApplyDruidTrigger` has no slot parameter or result; `UseDruidFeature` never mutates its `SpellcastingState`; `character-aggregate.md` D-4; `doc/rules/architecture.md` §3; unhappy (rule interaction); D-5 total order over tables I, K, P, the replacement rows and the assume order; every `DruidRejection` member has exactly one group |
| FR-139 to FR-142, FR-147, FR-148, FR-168, FR-171 to FR-173 | unit | constructors and the `New` constructor (returns an error) listing all problems; `fromMonster` (walking, flying, swimming, non-Beast); `fromClassData` with the class data as oracle, "1/4" and "Yes" cells, an unparseable cell as a problem; `DruidRejection` is an enum; package API scan finds none of `ConcentrationEffect`, `AttackKind`, `ActionCost`, `PotentSpellcasting`, `FeatChoiceSlot`, `SpellFacts`; Druid 3 / Fighter 5 scaling; shared slot pool; determinism; inputs and returned lists unchanged |
| FR-162 to FR-167 | unit + architecture | every operation callable with plain values and no collaborators (FR-164); level and subclass come only from `DruidCharacterFacts` (FR-167); state round-trips under its typed key; `DruidCreationChoices` carried by `CreateCharacter`; every event is a `CharacterEvent` with the `CharacterId` from the facts; `featureAttribute` maps each `DruidFeatureUse` to one `srd5e.feature` value or null; no signal is emitted |

- **Levels:** unit (`character_test` package, `druid_test.go` and siblings), scenario (`scenarios/druid_test.go`, public API only: levels 1 to 6,
  choose a land, assume a form, Short Rest recovery, Long Rest), architecture (no `combat` import); no doubles.
- **Mutation testing:** changed Druid files under the `character` package (pure domain files) MUST be mutation tested with `gomu` (Docker, CPU-capped, `--base-branch`), score at least 80 (`doc/rules/testing.md` §8). Likely survivors: boundaries at levels 3/4, 7/8,
  14/15, 17/18; 60/61 and 120/121 feet; budget; slot level 5/6.

## 9. Open questions and decisions

No question is open or escalated.

Settled: Q-4 (interpretation): Wild Resurgence "give yourself" and Nature Magician "produce" a slot restore an expended slot of that level and never exceed the table maximum, because the SRD gives no vanishing-slot clause (unlike Font of Magic, p. 66), so an extra slot would be unbounded; community consensus agrees, no official ruling (FR-100, FR-101, FR-113, FR-114; a use at full slots returns `NoExpendedSlot`). Q-16 by design: this spec maps the use to `srd5e.feature` (FR-165) and the host emits the counter; no Druid use case until a consumer needs one (character-aggregate.md D-15). Q-19: availability is read from the `SpellSlots` remaining and maximum counts of `SpellcastingState` (spellcasting.md FR-019), changes are returned as `DruidSlotChange` data (FR-160); see the amendment to spellcasting.md. Q-2 by FR-063, Q-9 by FR-171, Q-12 by FR-108, Q-15 by FR-172, Q-10 by FR-025 of shared-types-and-ownership.md,
Q-17 by content-monsters.md FR-097 (`ChallengeRating` is `Comparable`, FR-073), Q-18 by design (feat slots are not this
spec's, FR-019 and FR-020 withdrawn). Assumptions written into one FR are decided there: Q-1 FR-059, Q-3 FR-049 and
FR-104, Q-5 FR-105, Q-6 FR-118, Q-7 FR-091, Q-8 FR-137, Q-11 FR-080, Q-13 FR-089, Q-14 FR-105.

- **D-1** Wild Shape Temporary Hit Points are an event amount, not state; `character` is upstream of `combat` (shared-types-and-ownership.md FR-013). **D-2** Druid feature state is a `FeatureState` in `FeatureStates` (FR-162).
- **D-3** Using Wild Shape again in a Beast form is legal ("or until you use Wild Shape again", p. 42), including the same form (FR-063).
- **D-4** A rest is reported once and opens a window in state; commands carry no rest context (FR-153, FR-154).
- **D-5** Fixed rejection order over every `DruidRejection` member, eight groups: (1) availability: `FeatureNotGranted`, `FeatureNotChosen`, `LandTypeNotChosen`; (2) state: `NotInBeastForm`, `NoActiveSanctuary`, `SanctuaryAlreadyActive`, `PrimalOrderAlreadyChosen`, `ElementalFuryAlreadyChosen`; (3) casting: `CannotCastInBeastForm`, `MaterialComponentRestricted`; (4) timing: `ChangeNotAllowedNow`, `RecoveryNotAllowedNow`, `AlreadyUsedThisTurn`, `AlreadyUsedSinceLongRest`; (5) arguments: `InvalidConversion`, `AttackNotEligible`, `NotACircleSpell`, `FormNotKnown`, `NotABeast`, `ChallengeRatingTooHigh`, `FlySpeedNotAllowed`, `DuplicateForm`, `TooManyKnownForms`, `ItemUnassigned`, `ItemCannotBeWorn`, `MoveTooFar`; (6) resources: `NoUsesRemaining`, `WildShapeUsesRemain`, `NoSlotAvailable` (`slotRejected`); (7) limits: `SlotLevelTooHigh`, `RecoveryBudgetExceeded`, `TooManyReplacements`; (8) `NoExpendedSlot` (`slotRejected`). Creation-choice reasons (`WrongNumberOfSkills`, `DuplicateSkill`, `SkillNotOnList`, `UnknownEquipmentPackage`, in that order) and level reasons (`LevelNotConsecutive`, `MaximumLevelReached`) form their own groups. Like `character-aggregate.md` D-4.
- **D-6** Primal Order applies to a multiclass Druid: a level 1 feature gained as a multiclass (p. 41; FR-056). **D-7** A weapon attack made in Wild Shape is eligible for Primal Strike ("a weapon or a Beast form's attack", p. 43; FR-107).

## 10. Changelog impact

Under `Unreleased`, **Added**: the Druid class behaviour: `DruidFeatureState` (a `FeatureState` under a typed key),
`DruidFeatureTables`, `DruidCharacterFacts`, `DruidCreationChoices` and `DruidRules`; the pure operations
`GainDruidLevel`, `ApplyDruidChoice`, `UseDruidFeature` and `ApplyDruidTrigger` with their `<Operation>Result` types;
and the `DruidRules.featureAttribute` mapping for `srd5e.character.feature_used`. Class data ships with
`content-classes.md`. Not **BREAKING**: everything is new.
