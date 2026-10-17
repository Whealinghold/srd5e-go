# Feats: taking a feat and applying its benefits

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | Types 3; bulk data 5 |
| Depends on | `content-species-backgrounds-feats.md`, `content-schema-and-decoder.md`, `content-source-port.md`, `content-spells.md`, `content-classes.md`, `content-equipment.md`, `character-aggregate.md`, `derived-stats.md`, `level-advancement-and-multiclassing.md`, `spellcasting.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `errors-and-results.md`, `observability-ports.md` |

`level-advancement-and-multiclassing.md` (14) supplies `AbilityScoreIncrease`,
`AbilityScoreIncreaser` and `IncreaseExceedsMaximum`; `spellcasting.md` (13) supplies
`SlotLevel` (Q22). `rule-options.md` is referenced by key only. Downstream, named in prose only: the
`combat` specs, rests-and-hit-dice.md, character-creation.md and the `class-*`
specs. Until the bundled feat data exists (Phase 5), a `contenttest.FakeSource`
(package `content/contenttest`) covers it (shared-types-and-ownership.md FR-014).

## 1. Purpose and scope

content-species-backgrounds-feats.md is **data only**. This spec owns **applying** feats in the `character` context: granting
(which feats a grant allows), qualifying (prerequisites, once-only unless Repeatable), the choices made when a feat is taken
(increases, Magic Initiate, Skilled), what a held feat does (state in `character`, plus **declared effects** that `combat` or
the host interprets as a published-language handoff, because `character` cannot depend on `combat`), and
`FightingStyleChoice` (shared-types-and-ownership.md FR-020). The `AbilityScoreIncrease` value and its application are
level-advancement-and-multiclassing.md FR-025 to FR-036. State in, new state and events out (architecture.md §4); feats are
held in a `CharacterFeats` value beside the `Character`, as `SpellcastingState` is (Q4).

**Ownership (shared-types-and-ownership.md FR-048).** This spec owns what a feat does once taken and every check on taking
one. It supersedes the category and repeatability re-checks of class-fighter.md and class-ranger.md, the `notAFightingStyle`
check of class-paladin.md FR-038, the `FightingStyleClassOption` of class-ranger.md FR-051 (Q1) and the local "feat I am
owed" slot types (Q3). It defers to level-advancement-and-multiclassing.md (increase application), grappling.md (this spec
only declares the Grappler effects), spellcasting.md (casting a granted spell, `CastMode.FeatureGranted`) and
heroic-inspiration.md (not a feat). No use case calls another (shared-types-and-ownership.md FR-044; FR-081 here).

**Out of scope:** feat data (content-species-backgrounds-feats.md, content-schema-and-decoder.md); **who is owed a feat and
when** (`featGrantsAt`, content-classes.md FR-049; `PendingFeatGrant`, level-advancement-and-multiclassing.md FR-016; class
specs; character-creation.md), since this spec validates a grant it is handed (Q3, Q17); Blessed Warrior (p. 54) and Druidic
Warrior (p. 59) in class-paladin.md and class-ranger.md (this spec's Q1), and the Fighter's Fighting Style replacement timing (p. 47) in
class-fighter.md; interpreting declared effects at the table (attack rolls, damage, Initiative swaps, once-per-turn limits),
which is `combat` or the host (Q10, Q11), `d20` never sees a `FeatEffect`; casting granted spells (spellcasting.md; this spec's Q8);
the "Bonus Feats at Level 20" XP count (character-aggregate.md FR-114; FR-084 here subtracts feats taken); a prerequisite naming a class or a feature that
waives one (p. 87), which no SRD 5.2.1 feat uses.

## 2. Source

SRD 5.2.1 pp. 83, 86-88 (backgrounds, Human "Versatile", "Parts of a Feat", the 17 feats), pp. 20, 23-24, 26 (feat recording,
level advancement, "Bonus Feats at Level 20", Multiclassing), p. 5 and p. 8 (limits 20 and 30, Proficiency Bonus), the class
features "Ability Score Improvement", "Epic Boon", "Fighting Style" (pp. 47-49, 54, 59, 73), "Always-Prepared Spells" p. 104,
"Wish" p. 175 (replacing a feat, Q20) and "Creating a Background" p. 193. Rule files: architecture.md §2 to §6,
observability.md §2 to §7, testing.md §4, §5, §8. Page numbers are printed pages; the PDF index equals the printed number.

## 3. Vocabulary

SRD terms are used exactly. Reused, not redefined: `Feat` (content-species-backgrounds-feats.md FR-033), `FeatID`, `SpellID`,
`ItemID` (content-schema-and-decoder.md FR-114), `FeatCategory` (FR-117 there), `Prerequisite` clauses `LevelPrerequisite`,
`AbilityScorePrerequisite`, `FeaturePrerequisite` (FR-118 there; clauses combine with AND, the `abilities` of one clause with
OR, content-species-backgrounds-feats.md FR-035), `Skill`, `ConditionKind`, `DamageType`, `WeaponProperty`, `LightLevel`
(FR-133 there: `brightLight`, `dimLight`, `darkness`); `Spell`, `SpellLevel` (content-spells.md; the Cleric, Druid and Wizard
lists are `ClassID`s); `ArmorCategory`, `WeaponType`, `Tool` (content-equipment.md); `Ability`, `AbilityScore`, `Modifier`
(the kernel packages); `CharacterLevel` ("Level 4+" is total level, p. 87), `ProficiencyBonus`, `FlatBonus`, `SkillProficiencies`
(derived-stats.md); `AbilityScoreIncrease`, `AbilityScoreIncreaser`, `IncreaseExceedsMaximum`
(level-advancement-and-multiclassing.md); `SlotLevel` (spellcasting.md); `Feat.Repeatable` ("can be taken only once unless
its description states otherwise", p. 87). Types defined here are in section 5.

## 4. Requirements

Stable ids, grouped by topic. "Table X" is a case table below the requirements, one test per row.

### 4.1 Grants, categories and repeatability

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | A `FeatGrant` MUST accept a feat when its scope allows it: `SpecifiedFeat(id)` allows only that feat id, `FeatFromCategory(c)` a feat whose category is `c`, `AnyFeat` a feat of any of the four categories. | SRD 5.2.1, "Parts of a Feat", Category, p. 87 | happy |
| FR-002 | A `FeatFromCategory(c)` grant MUST reject a feat of another category with `FeatOutsideGrant`, leaving state unchanged. | SRD 5.2.1, "Parts of a Feat", Category ("that category must appear under the feat's name"), p. 87 | unhappy (invalid input) |
| FR-003 | A `SpecifiedFeat(featID, requiredSpellList)` grant MUST reject a different feat with `FeatOutsideGrant`, leaving state unchanged. | SRD 5.2.1, "Parts of a Background", Feat ("a specified Origin feat"), p. 83 | unhappy (invalid input) |
| FR-076 | When a `SpecifiedFeat` grant carries `RequiredSpellListIs(classID)`, the feat taken MUST be Magic Initiate with `MagicInitiateChoices` whose spell list equals `classID`. A different list MUST be rejected with `FeatChoicesInvalid` (`spellList`, kind `listNotAllowed`), a matching list accepted. `NoRequiredSpellList` places no constraint; no nil value stands for it. The caller that builds the grant maps the background's printed qualifier ("Cleric", "Wizard") to the `ClassID` of the same printed list name, once (character-creation.md does so; this spec names no dependency on it). | SRD 5.2.1, "Acolyte" (Magic Initiate (Cleric)) and "Sage" (Magic Initiate (Wizard)), p. 83 | unhappy (invalid input) |
| FR-004 | The scope used for each SRD source of a feat MUST be as in table G. The spec that owns the feature builds the `FeatGrant`; this spec pins the scope so every class spec agrees. | SRD 5.2.1, pp. 24, 47-49, 54, 59, 73, 83, 86-87 | happy |
| FR-005 | A feat the character already holds MUST be rejected with `FeatNotRepeatable` when its `Feat.Repeatable` is false, whatever source it first came from (Savage Attacker from Soldier, then again from Versatile). | SRD 5.2.1, "Parts of a Feat", Repeatable, p. 87 | unhappy (state error) |
| FR-006 | A held feat MUST pass the repeatability check when its `Feat.Repeatable` is true; the feat's own choice rules (FR-030) still apply. | SRD 5.2.1, "Parts of a Feat", Repeatable, p. 87 | happy |
| FR-007 | Repeatability MUST be read from `Feat.Repeatable`, never from a list of feat names in code, so a Fighting Style feat supplied as repeatable by a content source is accepted twice. | SRD 5.2.1, "Parts of a Feat", Repeatable, p. 87; architecture.md §3 (house content as data) | unhappy (rule interaction) |

### 4.2 Prerequisites

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-008 | A feat with no prerequisite clauses MUST qualify for every character. | SRD 5.2.1, "Origin Feats", p. 87 | happy |
| FR-009 | A `level` clause MUST be met exactly when total level is at least its `minimum`. Boundaries: level 3 fails and 4 passes for Level 4+; 18 fails and 19 and 20 pass for Level 19+. | SRD 5.2.1, "Ability Score Improvement", p. 87; "Boon of Combat Prowess", p. 88 | unhappy (boundary) |
| FR-010 | An `abilityScore` clause MUST be met exactly when at least one listed ability has a score of at least its `minimum`. Grappler ("Strength or Dexterity 13+"): 12 and 12 fail; 13 and 12, 12 and 13, 13 and 13 pass. | SRD 5.2.1, "Grappler", p. 87 | unhappy (boundary) |
| FR-011 | A `feature` clause MUST be met exactly when `HeldFeatures` contains a `FeatureID` that table N maps the clause name to. The clause name matches table N by exact, case-sensitive equality; a name not in table N is never met. "Spellcasting Feature" is met by any held feature named exactly "Spellcasting" (Bard, Cleric, Druid, Paladin, Ranger, Sorcerer, Wizard), not by "Pact Magic", which the SRD names as a separate feature (Q5). | SRD 5.2.1, "Archery", p. 87; "Boon of Spell Recall", p. 88; "Fighting Style", pp. 47, 54, 59; "Multiclassing", "Pact Magic", p. 26 | happy |
| FR-012 | Clauses MUST combine with AND, and a failed check MUST report every unmet clause, in the feat's clause order. | SRD 5.2.1, "Grappler", p. 87; "Boon of Spell Recall", p. 88; architecture.md §6 | unhappy (rule interaction) |
| FR-013 | Prerequisites MUST be evaluated against the character before the feat's own benefits, so an increase the feat gives never counts toward its own prerequisite (Strength 12 and Dexterity 12 fail Grappler even when the choice is Strength). | SRD 5.2.1, "Parts of a Feat", Prerequisite, Benefit, p. 87 | unhappy (rule interaction) |
| FR-014 | Prerequisites MUST be checked only when a feat is taken. A later fall in a score or loss of a feature MUST NOT remove the feat or its benefits. | SRD 5.2.1, "Parts of a Feat", Prerequisite ("To take a feat"), p. 87 | unhappy (state error) |
| FR-015 | An unmet prerequisite MUST give `FeatPrerequisiteNotMet` carrying the unmet clauses, no events, and the same `Character` and `CharacterFeats`. | SRD 5.2.1, "Parts of a Feat", Prerequisite, p. 87 | unhappy (invalid input) |
| FR-016 | A Fighting Style feat taken through an `AnyFeat` grant (an Ability Score Improvement slot) by a character with no Fighting Style feature MUST be rejected with `FeatPrerequisiteNotMet`, and accepted by one that has it. | SRD 5.2.1, "Fighting Style Feats", pp. 87-88; "Ability Score Improvement" (Fighter), p. 48 | unhappy (rule interaction) |

### 4.3 Choices and the shape each feat needs

The `AbilityScoreIncrease` value, its application, the maximum check, atomic
rejection, the Constitution Hit Point rule and event order are owned by
level-advancement-and-multiclassing.md (FR-025 to FR-036). FR-020 to FR-026 and
FR-079 add only the abilities each feat may name, the choice shape and the maximum
(20 or 30); `FeatTakingRules` builds the increase and applies it with
`AbilityScoreIncreaser`, so a feat emits one set of signals (those of `TakeFeat`).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-017 | Each of the 17 SRD feats MUST accept exactly the `FeatChoices` shape in table C. A missing required choice gives a `FeatChoicesInvalid` problem of kind `missing`; choices for a feat that takes none give `unexpected`. | SRD 5.2.1, pp. 87-88 | unhappy (invalid input) |
| FR-018 | A feat id with no behaviour here (anything but the 17 of table C) MUST be rejected with `FeatBehaviourUnknown` carrying the id, and no state changes. | SRD 5.2.1, pp. 87-88; architecture.md §3 | unhappy (missing content) |
| FR-019 | `FeatChoicesInvalid` MUST carry a non-empty slice (a copy on read) of `FeatChoiceProblem`s (typed `FeatChoiceName` and `FeatChoiceProblemKind`, no display text), listing every problem found; it is also the `errors.Join`-able `*FeatChoicesError` that the `NewX` constructors return. | architecture.md §5, §6 | unhappy (invalid input) |
| FR-020 | `AbilityScoreImprovementChoices` MUST be one ability by 2 or two distinct abilities by 1 each, converting to `AbilityScoreIncrease` of kind `OneByTwo` or `TwoByOne` (maximum 20). `NewAbilityScoreImprovementChoices` MUST return a `*FeatChoicesError` (non-empty `FeatChoiceProblem` slice, satisfying `errors.Is(err, srderr.ErrInvalidInput)`) for two equal abilities in the two-ability form, and the value with a nil error otherwise; it MUST NOT panic, and `MustAbilityScoreImprovementChoices` panics with that error iff it returns one. | SRD 5.2.1, "Ability Score Improvement", p. 87 | unhappy (invalid input) |
| FR-023 | Grappler's increase MUST be a `SingleAbilityIncrease` of Strength or Dexterity by 1, converted with the general `AbilityScoreIncrease` constructor, maximum 20; any other ability gives a problem (`increasedAbility`, kind `abilityNotAllowed`). | SRD 5.2.1, "Grappler", p. 87 | happy |
| FR-024 | Each Epic Boon feat's increase MUST be a `SingleAbilityIncrease` by 1 from the set in table C, converted with `AbilityScoreIncrease.EpicBoon` (maximum 30); a result above 30 is rejected as in FR-079. | SRD 5.2.1, "Epic Boon Feats", p. 88 | unhappy (boundary) |
| FR-026 | The events of a taken feat MUST be `FeatTaken` first, then those of `AbilityScoreIncreaser` in the order of level-advancement-and-multiclassing.md FR-036 (every `AbilityScoreChanged` in `Ability` declaration order, then at most one `HitPointMaximumChanged`). The order of the choice does not matter. | architecture.md §3; level-advancement-and-multiclassing.md FR-036 | happy |
| FR-079 | A rejected increase MUST make `TakeFeat` return `TakeFeatRejected` (`Character` and `CharacterFeats` unchanged, FR-069) with `FeatIncreaseRejected` carrying the `AbilityScoreIncreaseRejection` (`IncreaseExceedsMaximum`) with the feat's maximum (20 for Ability Score Improvement and Grappler, 30 for an Epic Boon). | SRD 5.2.1, "Ability Score Improvement", p. 87; "Epic Boon Feats", p. 88 | unhappy (boundary) |
| FR-027 | `MagicInitiateChoices` MUST name a spell list that is Cleric, Druid or Wizard, and an ability that is Intelligence, Wisdom or Charisma; any other value gives a problem of kind `listNotAllowed` or `abilityNotAllowed`. | SRD 5.2.1, "Magic Initiate", Two Cantrips, p. 87 | unhappy (invalid input) |
| FR-028 | Magic Initiate MUST take exactly two distinct cantrips (spell level 0) on the chosen list. A wrong count gives `wrongCount`, a repeat `duplicate`, a non-cantrip `notACantrip`, an off-list spell `notOnSpellList`. | SRD 5.2.1, "Magic Initiate", Two Cantrips, p. 87 | unhappy (invalid input) |
| FR-029 | Magic Initiate MUST take exactly one spell of level exactly 1 on the same list as the cantrips. A wrong level gives `notLevelOne`; off-list `notOnSpellList`. | SRD 5.2.1, "Magic Initiate", Level 1 Spell ("from the same list you selected for this feat's cantrips"), p. 87 | unhappy (invalid input) |
| FR-030 | A second or later Magic Initiate MUST choose a list different from that of every Magic Initiate already held; the same list gives `listAlreadyUsed`. Spells of different instances MAY overlap. | SRD 5.2.1, "Magic Initiate", Repeatable ("you must choose a different spell list each time"), p. 87 | unhappy (rule interaction) |
| FR-031 | A held Magic Initiate MUST expose its spells as `FeatGrantedSpells`: the two cantrips, the level 1 spell marked always prepared and not counted against any prepared-spell limit, and the chosen ability as the ability for the feat's spells. | SRD 5.2.1, "Magic Initiate", Level 1 Spell ("You always have that spell prepared"), p. 87; "Always-Prepared Spells" ("doesn't count against the number of spells on that list"), p. 104 | happy |
| FR-032 | Each Magic Initiate instance MUST carry a free-cast `FeatUseState`, `FeatUseIsAvailable` when taken. Expending it gives `FeatUseIsExpended` (and the event `FeatUseExpended`); expending an expended one is rejected with `FeatUseAlreadyExpended`. Only `longRestFinished` renews it; `shortRestFinished` and `initiativeRolled` MUST NOT. | SRD 5.2.1, "Magic Initiate", Level 1 Spell ("once without a spell slot ... when you finish a Long Rest"), p. 87 | unhappy (state error) |
| FR-033 | Casting the level 1 spell with a spell slot MUST NOT touch the free-cast state; this spec exposes the spell and the state and leaves casting to spellcasting.md. | SRD 5.2.1, "Magic Initiate", Level 1 Spell ("You can also cast the spell using any spell slots you have"), p. 87 | unhappy (rule interaction) |
| FR-034 | Spell Change MUST allow replacing one of the feat's spells with a different spell of the same spell level from the chosen list, only in response to a `LevelGained` event (character-aggregate.md FR-040) for this character whose new total level equals the character's current total level. | SRD 5.2.1, "Magic Initiate", Spell Change, p. 87 | happy |
| FR-035 | A replacement MUST name a spell of the same level as the one replaced (cantrip for cantrip, level 1 for the level 1 spell), on the chosen list, different from the one replaced and from the feat's other spells; otherwise `FeatChoicesInvalid` (`levelMismatch`, `notOnSpellList`, `duplicate`). | SRD 5.2.1, "Magic Initiate", Spell Change ("a different spell of the same level from the chosen spell list"), p. 87 | unhappy (invalid input) |
| FR-036 | A replacement without a matching `LevelGained` event (none, another character, or another level) MUST be rejected with `SpellChangeNotAllowedNow`. A second replacement for the same instance at the same total level MUST be rejected with `SpellChangeAlreadyUsed`. | SRD 5.2.1, "Magic Initiate", Spell Change ("Whenever you gain a new level ... one of the spells"), p. 87 | unhappy (state error) |
| FR-037 | A replacement MUST NOT change the instance's list, ability or free-cast `FeatUseState`; it MUST emit `FeatSpellReplaced` (old spell, new spell). | SRD 5.2.1, "Magic Initiate", Spell Change, p. 87; package design (Q16) | unhappy (rule interaction) |
| FR-038 | `SkilledChoices` MUST hold exactly three distinct entries, each a `Skill` or a tool. `NewSkilledChoices` MUST return a `*FeatChoicesError` (non-empty `FeatChoiceProblem` slice, name `skilledPicks`, kind `wrongCount` or `duplicate`, satisfying `errors.Is(err, srderr.ErrInvalidInput)`) for another count or a repeat, the value with a nil error for valid picks, and MUST NOT panic; `MustSkilledChoices` panics with that error iff it returns one. | SRD 5.2.1, "Skilled", p. 87 | unhappy (invalid input) |
| FR-039 | Any combination of skills and tools MUST be accepted (three skills, three tools, two and one, one and two). | SRD 5.2.1, "Skilled" ("any combination of three skills or tools"), p. 87 | happy |
| FR-040 | Taking Skilled MUST expose the three picks as `ProficiencyGrants`, for the host to merge into `SkillProficiencies` and its tool proficiencies. A pick the character already has MUST still be accepted (Q7). | SRD 5.2.1, "Skilled" ("You gain proficiency in"), p. 87 | happy |
| FR-041 | A second Skilled MUST be accepted with its own three picks (Repeatable). | SRD 5.2.1, "Skilled", Repeatable, p. 87 | happy |
| FR-042 | A `FightingStyleChoice` holds one `FeatID`. `ToTakeFeat()` MUST give that `FeatID`, a `FeatFromCategory(Fighting Style)` scope (with the `FeatSource` the caller supplies) and `NoFeatChoices`. | SRD 5.2.1, "Fighting Style" (Fighter), p. 47; "Fighting Style Feats", pp. 87-88 | happy |
| FR-043 | Taking a `FightingStyleChoice` whose feat is not of category Fighting Style MUST be rejected with `FeatOutsideGrant`. No class-option variant exists: Blessed Warrior and Druidic Warrior stay in class-paladin.md and class-ranger.md, which wrap a `FightingStyleChoice` for the feat case. | SRD 5.2.1, "Fighting Style" (Paladin), p. 54; (Ranger), p. 59 | unhappy (invalid input) |

### 4.4 What a held feat does

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-044 | A held feat MUST contribute exactly the declared effects of table E, with exactly the field values listed (section 5 gives the field types of all 19 variants), in the order of the feats held; a character with no feats has no effects. A test asserts each `FeatEffect` by value equality against an instance built from table E. | SRD 5.2.1, "Parts of a Feat", Benefit ("If you have a feat, you gain its benefits"), p. 87 | happy |
| FR-045 | Alert's initiative contribution MUST equal the character's `ProficiencyBonus` for its total level, and zero without Alert. Cases (one test each): levels 1 and 4 give +2, 5 and 8 +3, 9 and 12 +4, 13 and 16 +5, 17 and 20 +6; no Alert 0. | SRD 5.2.1, "Alert", Initiative Proficiency, p. 87; "Level Advancement", p. 23 | unhappy (boundary) |
| FR-046 | Defense's Armor Class contribution MUST be one `FlatBonus` of +1 with source `feature` when the worn armor categories include Light, Medium or Heavy, and none otherwise (a Shield alone, or nothing worn, gives none). Cases (one test each): Light, Medium or Heavy alone +1; Light and Shield +1 once; Shield only none; nothing worn none; Heavy without Defense none. | SRD 5.2.1, "Defense", p. 88 | unhappy (rule interaction) |
| FR-047 | Great Weapon Fighting MUST treat a rolled damage die of 1 or 2 as 3 and leave every other value unchanged. Cases (one test each): 1 and 2 give 3, 3 gives 3, 4 gives 4, 12 gives 12, 0 and -1 return an error. Its `DamageDieMinimum` effect MUST carry the conditions: weapon type Melee, held with two hands, and the Two-Handed or Versatile property. `GreatWeaponFighting.Treat` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a rolled value below 1. The rolled value is a bare `int`: one face of one die as `dice.Roller` returns it (dice-and-rolls.md). | SRD 5.2.1, "Great Weapon Fighting", p. 88 | unhappy (boundary) |
| FR-048 | `OverwhelmingStrike` MUST record the `Ability` (Strength or Dexterity) the Boon of Irresistible Offense increased (field `ability`), and `OverwhelmingStrike.ExtraDamage(AbilityScores)` MUST return that ability's current score as the damage amount; the damage type is that of the attack (`damageType: SameAsAttackDamageType()`, a `FeatDamageTypeRule`, not a `DamageType`). | SRD 5.2.1, "Boon of Irresistible Offense", Overwhelming Strike ("equal to the ability score increased by this feat"), p. 88 | happy |
| FR-049 | Effects limited by turn MUST carry the `FeatUsageLimit` of table E in their `limit` field: `RollDamageTwice` (Savage Attacker) and `UnarmedStrikeDamageAndGrapple` (Punch and Grab) `OncePerTurn`, `MissBecomesHit` (Peerless Aim) `OnceUntilStartOfYourNextTurn`. This spec holds no per-turn state (Q10). | SRD 5.2.1, "Grappler", Punch and Grab; "Savage Attacker"; "Boon of Combat Prowess", Peerless Aim, pp. 87-88 | happy |
| FR-050 | Improve Fate MUST carry a free-use `FeatUseState`, `FeatUseIsAvailable` when taken, renewed by `initiativeRolled`, `shortRestFinished` and `longRestFinished`. Expending an expended one is rejected with `FeatUseAlreadyExpended`. | SRD 5.2.1, "Boon of Fate", Improve Fate, p. 88 | unhappy (state error) |
| FR-051 | `ExpendFeatUse` on a feat not held MUST be rejected with `FeatNotHeld`; on a held feat with no limited-use benefit (Alert) with `FeatHasNoLimitedUse`. | SRD 5.2.1, "Parts of a Feat", Benefit, p. 87; package design | unhappy (state error) |
| FR-052 | `RenewFeatUses` for a trigger MUST renew every held feat use that lists the trigger and is expended, emitting one `FeatUsesRenewed` listing the renewed feats; when nothing is expended or listed it returns the same state and no events (a successful no-op). | SRD 5.2.1, "Magic Initiate"; "Boon of Fate", pp. 87-88; package design | unhappy (empty case) |
| FR-053 | Boon of Spell Recall Free Casting (`FreeCastingRule.Resolve`) MUST decide, from one roll of 1d4 and a `SlotLevel` of 1 to 4, that the slot is not expended (`FreeCastingSpared`) exactly when the roll equals the slot's level, otherwise `FreeCastingSpent` (table F). | SRD 5.2.1, "Boon of Spell Recall", Free Casting, p. 88 | unhappy (boundary) |
| FR-054 | A `SlotLevel` of 5 to 9 MUST NOT roll or call the dice roller, and the result MUST say the benefit does not apply (`slotAboveFour`). A level outside 1 to 9 is not constructible: `NewSlotLevel` returns an `srderr.ErrInvalidInput` error (spellcasting.md). | SRD 5.2.1, "Boon of Spell Recall" ("level 1-4 spell slot"), p. 88; "Spell Level", p. 104 | unhappy (boundary) |
| FR-055 | A character without Boon of Spell Recall MUST be rejected with `FeatNotHeld` for Free Casting (`RollFreeCastingRejected`, as FR-051 does for `ExpendFeatUse`), with no roll. It is a failure, not a `not_applicable` success, so a host that wrongly offers Free Casting is visible in the outcome counter (section 7). | SRD 5.2.1, "Parts of a Feat", Benefit, p. 87 | unhappy (state error) |
| FR-056 | A roll outside 1 to 4 for the 1d4, or a result for a different expression, MUST make the use case return `srderr.ContractViolation(dice.Port, "roll", detail)` (a `*srderr.PortError` wrapping `srderr.ErrContractViolation`), with no result built. | dice-and-rolls.md FR-050, FR-051; character-aggregate.md FR-079 | unhappy (collaborator failure) |
| FR-077 | `RollFreeCasting` MUST check in this order: an invalid slot level (an `srderr.ErrInvalidInput` error, before the use case is called, FR-054), then `FeatNotHeld` (FR-055), then `slotAboveFour` (FR-054), then the roll. Level 0 or 10 returns the invalid-slot-level error whether or not the feat is held; level 5 to 9 without the feat gives `FeatNotHeld`. | architecture.md §3 (determinism); package design | unhappy (rule interaction) |
| FR-078 | An error returned by the roller during `RollFreeCasting` MUST be returned wrapped once with `srderr.WrapPort(dice.Port, "roll", err)`, with no result and no log, and be recorded once (FR-074), as FR-068 does for the content port. | architecture.md §4; errors-and-results.md FR-034; character-aggregate.md FR-078 | unhappy (collaborator failure) |

### 4.5 State, invariants and determinism

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-057 | `CharacterFeats`, `TakenFeat`, `FeatChoices`, every `FeatEffect` and every event MUST be immutable, compare by value (`==` where comparable, otherwise an `Equal` method), and return copies of their collections (mutating a returned slice or map changes nothing). | architecture.md §3, §4 | unhappy (immutability) |
| FR-058 | Every transition MUST return new instances and leave its inputs (`Character`, `CharacterFeats`, command) equal to their prior value. | architecture.md §3 | unhappy (immutability) |
| FR-059 | A rejected transition MUST return the rejection and no events, and MUST NOT panic. Invalid arguments (FR-020, FR-038, FR-047, FR-054, FR-061) returns an `srderr.ErrInvalidInput` error and are never results. | architecture.md §6 | unhappy (state error) |
| FR-060 | When several rejection reasons hold, the reported reason MUST follow the order of table O. | architecture.md §3 (determinism) | unhappy (rule interaction) |
| FR-061 | `NewCharacterFeats` and `NewTakenFeat` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a `FeatChoices` shape that does not match the feat id (table C), and for two Magic Initiate entries with the same spell list. They cannot check repeatability, which needs content. | architecture.md §3; FR-017, FR-030 | unhappy (invalid input) |
| FR-062 | `CharacterFeats.TakenFrom(source)` MUST return the number of held feats taken from a source, matching on source kind and names; class specs use it to work out which grants are used (Q3). | package design | happy |
| FR-063 | `FeatTaken` MUST carry the character id, feat id, `FeatCategory` and `FeatSource`, with no display text. All feat events MUST be members of the closed `FeatEvent` family (FR-080), which implements `character.Event`, and carry the character id. | architecture.md §4; character-aggregate.md FR-085 | happy |
| FR-064 | The same `Character`, `CharacterFeats`, feat, grant and choices MUST give an equal result on every call (no clock, randomness or ordering dependence). | architecture.md §3 | unhappy (determinism) |
| FR-065 | `HeldFeatures` MUST be an immutable set of `FeatureID`s (a blank or malformed id cannot be built: `content.ParseFeatureID` returns an `srderr.ErrInvalidInput` error). | architecture.md §3 | unhappy (invalid input) |

### 4.6 Use cases

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-066 | `TakeFeat` MUST look up the feat through `content.Source`, and every spell and tool named in the choices, before returning, and on a full set found return `TakeFeatSucceeded` with the new `Character`, `CharacterFeats`, any `ProficiencyGrants` and `FeatGrantedSpells`, and the events. | architecture.md §4; shared-types-and-ownership.md FR-017 | happy |
| FR-067 | When any looked-up entry is not found, `TakeFeat` MUST return `TakeFeatContentMissing` holding a non-empty slice (copied on read) of `content.MissingContent` (kind and id), without duplicates, ordered by content kind declaration order then id ordinal comparison, and MUST NOT panic. | shared-types-and-ownership.md FR-018; errors-and-results.md; character-aggregate.md FR-061, FR-077 | unhappy (missing content) |
| FR-068 | An error from the content port MUST be returned wrapped once with `srderr.WrapPort(content.Port, op, err)` out of `TakeFeat` and `ChangeMagicInitiateSpell`, with no log, and be recorded once. | architecture.md §4; observability.md §7 | unhappy (collaborator failure) |
| FR-069 | A domain rejection MUST become `TakeFeatRejected` carrying the `FeatRejection`, with `Character` and `CharacterFeats` unchanged. | architecture.md §4 | unhappy (invalid input) |
| FR-070 | `ChangeMagicInitiateSpell` MUST look up the replacement spell, return `ChangeMagicInitiateSpellSucceeded` (new `CharacterFeats`, `FeatSpellReplaced`), `ChangeMagicInitiateSpellRejected` or `ChangeMagicInitiateSpellContentMissing`, and MUST NOT panic for any of them. | architecture.md §4 | happy |
| FR-071 | `ExpendFeatUse` and `RenewFeatUses` MUST use no port other than the telemetry bundle; `RollFreeCasting` MUST use the `Roller` port and the telemetry bundle only. All five take `ctx context.Context` first because they emit spans; the pure rules they call take none. | architecture.md §4 | happy |
| FR-072 | Every use case MUST behave identically with the zero-value `telemetry.Noop` and a recording double. | observability.md §9 | unhappy (determinism) |
| FR-073 | Each use case MUST emit the signals of section 7 once per call, through defined constants, and MUST NOT log. `srd5e.outcome` MUST be only `success` or `failure`; the variant MUST be in `srd5e.character.result` (`taken`, `changed`, `resolved`, `rejected`, `content_missing`, `not_applicable`), and a rejection's reason in `srd5e.character.rejection`. | observability.md §5, §6, §7, §9; errors-and-results.md FR-043; character-aggregate.md FR-071, FR-072, FR-088 | happy |
| FR-074 | A `TakeFeatContentMissing` result MUST be recorded exactly once with `srd5e.outcome=failure`, `srd5e.character.result=content_missing`, span error status, one counter increment, no `error.type` and no log. A returned `*srderr.PortError` MUST be recorded exactly once with span error status, one counter increment and `error.type` equal to `srderr.ErrorType(err)` (`content_error` or `dice_roller_error`), no `srd5e.character.result`, and no log. A `FeatRejection` MUST be recorded as `srd5e.outcome=failure`, `srd5e.character.result=rejected` and the `srd5e.character.rejection` attribute, span status not error. | observability.md §6, §7; errors-and-results.md FR-034, FR-043; character-aggregate.md FR-071, FR-072 | unhappy (collaborator failure) |
| FR-075 | No use case MAY alter its command or the `Character` and `CharacterFeats` in it; after any call they equal their prior values. | architecture.md §3, §4 | unhappy (immutability) |
| FR-080 | Every use case MUST return its own closed `<UseCase>Result` root (`TakeFeatResult`, `ChangeMagicInitiateSpellResult`, `ExpendFeatUseResult`, `RenewFeatUsesResult`, `RollFreeCastingResult`) whose variants are prefixed with the operation (section 6); no generic variant name is exported. The events `FeatTaken`, `FeatSpellReplaced`, `FeatUseExpended`, `FeatUsesRenewed` and `FeatReplaced` (FR-082) MUST be the members of one closed family `FeatEvent`, which satisfies `character.Event` (character-aggregate.md FR-085), so a host switches exhaustively over them. Each event is a past-tense value object carrying the `CharacterID`, with exactly one defining spec. | shared-types-and-ownership.md FR-017, FR-019, FR-021; character-aggregate.md FR-085 | happy |
| FR-081 | No use case here MAY call another use case (shared-types-and-ownership.md FR-044). `TakeFeat` composes `FeatQualification`, `FeatTakingRules` and `AbilityScoreIncreaser` and records only its own signals; its increase never produces the signals of `ChangeAbilityScore`. `FeatQualification`, `FeatTakingRules`, `FeatEffects`, `FeatUses` and `MagicInitiateSpellChange` are domain services exported from package `character` so another `character` use case MAY compose them. | shared-types-and-ownership.md FR-044; architecture.md §4 | unhappy (nested telemetry) |
| FR-082 | `FeatReplacement.Replace(Character, CharacterFeats, oldFeatID, Feat, FeatGrant, FeatChoices, HeldFeatures, resolvedSpells)` is a domain service for downstream use cases (class-fighter.md Fighting Style swap, wish-spell.md Wish). It MUST check in order: old feat held (else `FeatNotHeld`); old feat gave no ability score increase (else `FeatNotReplaceable`); then remove the old `TakenFeat` with its choices and `FeatUseState`s (FR-083), and run `FeatTakingRules` for the new feat on the result (table O, repeatability and prerequisites seen without the old feat). Success emits `FeatReplaced` (old id, new id), then the events of FR-026. A feat that is a prerequisite of another held feat cannot occur among the 17 (no SRD feat names a feat); a prerequisite of a feature is the caller's check (Q20). | SRD 5.2.1, "Wish", Sudden Learning ("You lose all the benefits of the old feat and gain the benefits of the new one"), p. 175; "Fighting Style" (Fighter), p. 47 | unhappy (state error) |
| FR-083 | `CharacterFeats.Replacing(old FeatID, new TakenFeat)` MUST be the pure transition behind FR-082: the old feat must be held (else `FeatNotHeld`), it is removed first with its choices and use state, then `new` is added unless it equals another held non-Repeatable feat (`FeatNotRepeatable`, FR-005 to FR-007; the old feat no longer counts, so replacing a feat by itself is accepted). It returns a new `CharacterFeats` or the rejection and alters nothing. It checks no grant, prerequisite or choice: a caller that needs them runs `FeatTakingRules.Take` on the result (as FR-082 does). Works for `FeatFromCategory(Fighting Style)` (Fighter, class-fighter.md FR-036, FR-157) and `AnyFeat` (Wish). | SRD 5.2.1, "Fighting Style" (Fighter: replace the feat you chose), p. 47; "Parts of a Feat" (Repeatable), p. 87; "Wish", p. 175 | unhappy (state error) |
| FR-084 | `BonusFeats.Remaining(BonusFeatsAtLevel20 option, ExperiencePoints xp, CharacterFeats feats)` MUST return `max(0, option.featsEarned(xp) - feats.takenFrom(BonusFeat))` (character-aggregate.md FR-113, FR-114; FR-062), so `NoBonusFeats` gives 0. It is the only consumer of `CharacterRuleOptions.BonusFeatsAtLevel20` outside the aggregate: level-advancement-and-multiclassing.md leaves the option out of scope. A positive count lets the host call `TakeFeat` with a `BonusFeat` grant and `AnyFeat` scope; qualification, including level 20, stays in FR-001 to FR-016 (`TakeFeat` verifies no entitlement, Q3). | SRD 5.2.1, "Bonus Feats at Level 20", p. 24 ("each character gains one feat of their choice" per 30,000 XP) | happy / unhappy (boundary) |

### Table G: the scope each SRD source uses (FR-004, FR-076)

| Label | Source | Scope | Printed rule | Cite | Path |
|---|---|---|---|---|---|
| Acolyte background | `BackgroundFeat` | `SpecifiedFeat(magic_initiate, RequiredSpellListIs(cleric))` | "Feat: Magic Initiate (Cleric)" | p. 83 | happy |
| Criminal background | `BackgroundFeat` | `SpecifiedFeat(alert, NoRequiredSpellList)` | "Feat: Alert" | p. 83 | happy |
| Sage background | `BackgroundFeat` | `SpecifiedFeat(magic_initiate, RequiredSpellListIs(wizard))` | "Feat: Magic Initiate (Wizard)" | p. 83 | happy |
| Soldier background | `BackgroundFeat` | `SpecifiedFeat(savage_attacker, NoRequiredSpellList)` | "Feat: Savage Attacker" | p. 83 | happy |
| Custom background | `CustomBackgroundFeat` | `FeatFromCategory(Origin)` | "Choose one feat from the Origin category" | p. 193 | happy |
| Acolyte slot, Magic Initiate on the Cleric list; on the Wizard list | `BackgroundFeat` | Acolyte scope | n/a | p. 83 | happy; unhappy (invalid input): `FeatChoicesInvalid` (`spellList`, `listNotAllowed`) |
| Human Versatile | `SpeciesTraitFeat` | `FeatFromCategory(Origin)` | "an Origin feat of your choice" | p. 86 | happy |
| Ability Score Improvement (every class) | `ClassFeatureFeat` | `AnyFeat` | "the Ability Score Improvement feat or another feat of your choice for which you qualify" | p. 48 | happy |
| Epic Boon (every class, level 19) | `ClassFeatureFeat` | `AnyFeat` | "an Epic Boon feat or another feat of your choice for which you qualify" | p. 48 | happy |
| Fighting Style (Fighter, Paladin, Ranger) | `ClassFeatureFeat` | `FeatFromCategory(Fighting Style)` | "a Fighting Style feat of your choice" | pp. 47, 54, 59 | happy |
| Additional Fighting Style (Champion) | `ClassFeatureFeat` | `FeatFromCategory(Fighting Style)` | "another Fighting Style feat of your choice" | p. 49 | happy |
| Lessons of the First Ones | `EldritchInvocationFeat` | `FeatFromCategory(Origin)` | "one Origin feat of your choice" | p. 73 | happy |
| Bonus Feats at Level 20 | `BonusFeat` | `AnyFeat` | "any feat for which their level 20 character qualifies" | p. 24 | happy |
| Fighting Style slot, a General feat; Versatile slot, an Epic Boon feat; Soldier slot, Alert | `ClassFeatureFeat`; `SpeciesTraitFeat`; `BackgroundFeat` | the scope of the row above each | n/a | pp. 47, 83, 86, 87 | unhappy (invalid input): `FeatOutsideGrant` (one test each) |

### Table N: how a `feature` clause name matches a held feature (FR-011)

A held `FeatureID` is matched by the printed feature name the class spec gives it
(Q19).

| Label | Clause name | Met by holding a feature named | Cite | Path |
|---|---|---|---|---|
| Fighting Style (Fighter level 1, Paladin level 2, Ranger level 2; one test each) | "Fighting Style Feature" | "Fighting Style" | pp. 47, 54, 59, 87 | happy |
| Spellcasting (Bard, Cleric, Druid, Paladin, Ranger, Sorcerer, Wizard; one test each) | "Spellcasting Feature" | "Spellcasting" | pp. 54, 88 | happy |
| Pact Magic | "Spellcasting Feature" | not met by "Pact Magic" (Warlock), a separate feature | pp. 26, 88 | unhappy (rule interaction) |
| no match | either; "Fighting Style Feature"; "spellcasting feature" | `HeldFeatures` empty; "Additional Fighting Style" alone; name not in this table, even with "Spellcasting" held: all unmet (one test each) | pp. 49, 87-88 | unhappy (boundary) |

### Table C: the choices each feat takes (FR-017, FR-023, FR-024, FR-061)

| Label | Feat id | Category | Choices shape | Allowed abilities / rule | Max | Cite | Path |
|---|---|---|---|---|---|---|---|
| no choices, Origin | `alert`, `savage_attacker` | Origin | `NoFeatChoices` | n/a | n/a | p. 87 | happy |
| magic initiate | `magic_initiate` | Origin | `MagicInitiateChoices` | list Cleric, Druid or Wizard; ability Int, Wis or Cha | n/a | p. 87 | happy |
| skilled | `skilled` | Origin | `SkilledChoices` | three distinct skills or tools | n/a | p. 87 | happy |
| ability score improvement | `ability_score_improvement` | General | `AbilityScoreImprovementChoices` | one by 2, or two by 1 | 20 | p. 87 | happy |
| grappler | `grappler` | General | `SingleAbilityIncrease` | Strength or Dexterity, by 1 | 20 | p. 87 | happy |
| Fighting Style feats | `archery` (p. 87); `defense`, `great_weapon_fighting`, `two_weapon_fighting` (p. 88) | Fighting Style | `NoFeatChoices` | n/a | n/a | pp. 87-88 | happy |
| Epic Boon, any ability | `boon_of_combat_prowess`, `boon_of_dimensional_travel`, `boon_of_fate`, `boon_of_the_night_spirit`, `boon_of_truesight` | Epic Boon | `SingleAbilityIncrease` | any one ability, by 1 | 30 | p. 88 | happy |
| boon of irresistible offense | `boon_of_irresistible_offense` | Epic Boon | `SingleAbilityIncrease` | Strength or Dexterity, by 1 | 30 | p. 88 | happy |
| boon of spell recall | `boon_of_spell_recall` | Epic Boon | `SingleAbilityIncrease` | Intelligence, Wisdom or Charisma, by 1 | 30 | p. 88 | happy |
| missing, unexpected or wrong shape | `grappler`, none supplied; `alert` with `SingleAbilityIncrease`; `skilled` with `MagicInitiateChoices` | General, Origin | n/a | n/a | n/a | p. 87 | unhappy (invalid input): `missing`; `unexpected`; `unexpected` (one test each) |
| ability outside set | `boon_of_irresistible_offense` with Wisdom; `grappler` with Constitution; `boon_of_spell_recall` with Strength | Epic Boon, General | `SingleAbilityIncrease` | n/a | n/a | pp. 87-88 | unhappy (invalid input): `abilityNotAllowed` (one test each) |
| feat without behaviour | `house_feat` | any | any | n/a | n/a | p. 87 | unhappy (missing content): `FeatBehaviourUnknown` (FR-018) |

Feat ids are those of content-species-backgrounds-feats.md section 3 (FR-003
there); where a bundled id differs from the snake_case of the printed name, that
spec's id wins and this table changes.

### Table A: which increase a feat builds (FR-020, FR-023, FR-024, FR-026, FR-079)

Arithmetic, atomic rejection, equal abilities and the Constitution rule are
level-advancement-and-multiclassing.md table AI; these rows test the choice, the
allowed abilities, the maximum selected and the mapping of the rejection.

| Label | Feat and choice | Scores before | Expected | Cite | Path |
|---|---|---|---|---|---|
| ASI one by two | ASI, Strength by 2 | STR 18 | `abilityScoreImprovementOneByTwo(Strength)` applied: STR 20 | p. 87 | happy |
| ASI two by one | ASI, Dexterity by 1 and Constitution by 1 | DEX 19, CON 10 | `abilityScoreImprovementTwoByOne` applied: DEX 20, CON 11 | p. 87 | happy |
| ASI over the maximum | ASI, Strength by 2 | STR 19 | `FeatIncreaseRejected(IncreaseExceedsMaximum([Strength], 20))`; no `FeatTaken` | p. 87 | unhappy (boundary) |
| Grappler | Grappler, Dexterity | DEX 19 | general `AbilityScoreIncrease({Dexterity: 1}, 20)`: DEX 20 | p. 87 | happy |
| Grappler at the limit | Grappler, Strength | STR 20 | `FeatIncreaseRejected(IncreaseExceedsMaximum([Strength], 20))` | p. 87 | unhappy (boundary) |
| Boon below 30 | Boon of Fate, Constitution | CON 29 | `AbilityScoreIncrease.EpicBoon(Constitution)`: CON 30 | p. 88 | unhappy (boundary) |
| Boon at 30 | Boon of Fate, Constitution | CON 30 | `FeatIncreaseRejected(IncreaseExceedsMaximum([Constitution], 30))` | p. 88 | unhappy (boundary) |
| Boon above 20 | Boon of Truesight, Wisdom | WIS 20 | WIS 21 (maximum 30, not 20) | p. 88 | happy |
| event order | ASI, Constitution by 1, level 8 | CON 17 | `FeatTaken`, then `AbilityScoreChanged`, `HitPointMaximumChanged` (FR-026) | p. 23 | happy |

### Table E: declared effects of each feat (FR-044, FR-049)

Each effect is a value object with exactly the fields shown as `Name Type = value` (exported read-only accessors in Go); a
test builds the expected instance and asserts equality. Lower camel case values are
enum members (exported typed constants); slices are copies on read (FR-057).

| Feat | Declared effects (variant with field values) | Limit | Cite | Path |
|---|---|---|---|---|
| Alert | `InitiativeProficiency()` (amount is the `ProficiencyBonus`, FR-045); `InitiativeSwap(Timing FeatInitiativeTiming = afterRollingInitiative, Partner FeatSwapPartner = willingAllyInSameCombat, BlockedByCondition ConditionKind = incapacitated)` | none | p. 87 | happy |
| Magic Initiate | `FeatGrantedSpells` (FR-031); free cast (FR-032); spell change (FR-034) | free cast: `UseRenewedBy({longRestFinished})` | p. 87 | happy |
| Savage Attacker | `RollDamageTwice(Trigger FeatTrigger = weaponHit, Limit FeatUsageLimit = OncePerTurn)` | `OncePerTurn` | p. 87 | happy |
| Skilled | `ProficiencyGrants` (FR-040) | none | p. 87 | happy |
| Ability Score Improvement | none beyond the increase (FR-020) | none | p. 87 | happy |
| Grappler | `UnarmedStrikeDamageAndGrapple(Limit FeatUsageLimit = OncePerTurn)`; `AdvantageAgainstGrappledByYou()`; `GrappleMovementDiscount(TargetSize FeatSizeRelation = sameSizeOrSmaller)` | Punch and Grab: `OncePerTurn` | p. 87 | happy |
| Archery | `AttackRollBonus(Bonus Modifier = +2, WeaponType WeaponType = Ranged)` | none | p. 87 | happy |
| Defense | `ArmoredArmorClassBonus(Bonus FlatBonus = +1 source feature, WhileWearing []ArmorCategory = {Light, Medium, Heavy})` (FR-046) | none | p. 88 | happy |
| Great Weapon Fighting | `DamageDieMinimum(Minimum int = 3, AppliesToRolledAtMost int = 2, WeaponType WeaponType = Melee, Hands HandsUsed = twoHands, AnyOfProperties []WeaponProperty = {twoHanded, versatile})` (FR-047) | none | p. 88 | happy |
| Two-Weapon Fighting | `ExtraAttackAbilityModifierToDamage(RequiredProperty WeaponProperty = light)` | none | p. 88 | happy |
| Boon of Combat Prowess | `MissBecomesHit(Limit FeatUsageLimit = OnceUntilStartOfYourNextTurn)` | `OnceUntilStartOfYourNextTurn` | p. 88 | happy |
| Boon of Dimensional Travel | `BlinkSteps(DistanceFeet int = 30, After []FeatTriggerAction = {attackAction, magicAction}, Destination FeatDestination = unoccupiedSpaceYouCanSee)` | none | p. 88 | happy |
| Boon of Fate | `ImproveFate(RangeFeet int = 60, Dice dice.Expression = 2d4, Use UseRenewedBy = {initiativeRolled, shortRestFinished, longRestFinished})` (FR-050) | same `UseRenewedBy` | p. 88 | happy |
| Boon of Irresistible Offense | `IgnoreResistance(DamageTypes []DamageType = {bludgeoning, piercing, slashing})`; `OverwhelmingStrike(Ability Ability = <the increased ability>, Trigger FeatTrigger = natural20OnAttackRoll, DamageType FeatDamageTypeRule = SameAsAttackDamageType())` (FR-048) | none | p. 88 | happy |
| Boon of Spell Recall | `FreeCasting(SlotLevelsFrom SlotLevel = 1, SlotLevelsTo SlotLevel = 4, Die dice.Expression = 1d4)` (FR-053) | none | p. 88 | happy |
| Boon of the Night Spirit | `MergeWithShadows(Requires []LightLevel = {dimLight, darkness}, Grants ConditionKind = invisible, Cost FeatTriggerAction = bonusAction, EndsAfter []FeatTriggerAction = {action, bonusAction, reaction})`; `ShadowyForm(Requires []LightLevel = {dimLight, darkness}, ResistanceToAllExcept []DamageType = {psychic, radiant})` | none | p. 88 | happy |
| Boon of Truesight | `TruesightSense(RangeFeet int = 60)` | none | p. 88 | happy |
| (no feat held) | no effects | n/a | p. 87 | unhappy (empty case) |

### Table F: Free Casting (FR-053 to FR-056, FR-077, FR-078)

| Label | Slot level | Roll of 1d4 | Expected | Cite | Path |
|---|---|---|---|---|---|
| level 1, roll 1; roll 2 | 1 | 1; 2 | `FreeCastingSpared`; `FreeCastingSpent` | p. 88 | unhappy (boundary) |
| level 2, roll 2; level 3, roll 4 | 2; 3 | 2; 4 | `FreeCastingSpared`; `FreeCastingSpent` | p. 88 | happy |
| level 4, roll 4; roll 3 | 4 | 4; 3 | `FreeCastingSpared`; `FreeCastingSpent` | p. 88 | unhappy (boundary) |
| levels 5 and 9 | 5; 9 | none | `slotAboveFour`; no dice call | p. 88 | unhappy (boundary) |
| levels 0 and 10 | 0; 10 | n/a | an `srderr.ErrInvalidInput` error | p. 104 | unhappy (invalid input) |
| not held | 2 | none | `FeatNotHeld` rejection; no dice call | p. 87 | unhappy (state error) |
| roll outside 1 to 4 from port | 2 | 0; 5 | `srderr.ErrContractViolation` (a `*srderr.PortError`) | n/a | unhappy (collaborator failure) |
| roller fails | 2 | the roller returns an error | returned wrapped once as a `*srderr.PortError`, no result, recorded once (FR-078) | n/a | unhappy (collaborator failure) |
| not held, level 0 or 10; level 5 | 0; 10; 5, feat not held | none (no roll) | an `srderr.ErrInvalidInput` error, not `FeatNotHeld`; `FeatNotHeld`, not `slotAboveFour` (FR-077) | pp. 87, 104 | unhappy (rule interaction) |

### Table O: order of rejection reasons (FR-060)

Order: `FeatBehaviourUnknown`, `FeatOutsideGrant`, `FeatNotRepeatable`,
`FeatPrerequisiteNotMet`, `FeatChoicesInvalid`, `FeatIncreaseRejected`. One test per
adjacent pair with both conditions holding: no-behaviour feat outside the grant gives
`FeatBehaviourUnknown`; wrong category and already held, `FeatOutsideGrant`; already
held and level too low, `FeatNotRepeatable`; level too low and a missing choice,
`FeatPrerequisiteNotMet`; a wrong choice shape and a score above the maximum,
`FeatChoicesInvalid`; and one with all else valid and only the maximum exceeded
(`FeatIncreaseRejected`).

### Table M: Magic Initiate choice cases (FR-027 to FR-030, FR-032, FR-034 to FR-037)

| Label | Input | Expected | Cite | Path |
|---|---|---|---|---|
| valid Cleric | list Cleric, ability Wisdom, 2 Cleric cantrips, 1 Cleric level 1 spell | accepted | p. 87 | happy |
| valid Wizard | list Wizard, ability Intelligence, 2 Wizard cantrips, 1 Wizard level 1 spell | accepted | p. 87 | happy |
| list and ability | list Bard; ability Strength | `listNotAllowed`; `abilityNotAllowed` (one test each) | p. 87 | unhappy (invalid input) |
| cantrip count | 1 cantrip; 3 cantrips; 2 equal cantrips | `wrongCount`; `wrongCount`; `duplicate` | p. 87 | unhappy (boundary) |
| wrong level | a level 1 spell in cantrips; a cantrip or level 2 spell as the level 1 spell | `notACantrip`; `notLevelOne` | p. 87 | unhappy (boundary) |
| off list | a spell not on the chosen list | `notOnSpellList` | p. 87 | unhappy (invalid input) |
| second, other list | holds Cleric; new instance Wizard | accepted | p. 87 | happy |
| second, same list | holds Cleric; new instance Cleric | `listAlreadyUsed` | p. 87 | unhappy (rule interaction) |
| overlapping spells | holds Cleric with Guidance; new instance Druid with Guidance | accepted | p. 87 | unhappy (rule interaction) |
| change cantrip | at `LevelGained`, replace a cantrip with another cantrip on the list | accepted; `FeatSpellReplaced` | p. 87 | happy |
| change level 1 spell | at `LevelGained`, replace the level 1 spell with another level 1 spell | accepted; free cast unchanged | p. 87 | happy |
| change, invalid | a cantrip replaced by a level 1 spell; a spell by itself; by the feat's other spell; by an off-list spell | `levelMismatch`; `duplicate`; `duplicate`; `notOnSpellList` (one test each) | p. 87 | unhappy (invalid input) |
| change, no event, other level, other character | no `LevelGained`; event for a level other than the current total; event for another character id | `SpellChangeNotAllowedNow` (one test each) | p. 87 | unhappy (state error) |
| change twice | two replacements at one level | second `SpellChangeAlreadyUsed` | p. 87 | unhappy (state error) |
| change at two levels | one replacement at level 5, one at level 6 | both accepted | p. 87 | unhappy (boundary) |
| free cast, rests | expend, renew on `shortRestFinished`; expend, renew on `longRestFinished` | still expended; available | p. 87 | unhappy (state error) |

### Table U: feat use and granted spells (FR-031 to FR-033, FR-050 to FR-052, FR-070)

| Label | Input | Expected | Cite | Path |
|---|---|---|---|---|
| granted spells | Magic Initiate taken | `FeatGrantedSpells`: two cantrips, level 1 spell always prepared and uncounted, chosen ability | p. 87 | happy |
| slot cast | cast the level 1 spell with a slot | free-cast state unchanged | p. 87 | unhappy (rule interaction) |
| expend twice | Improve Fate expended, expended again | `FeatUseAlreadyExpended` | p. 88 | unhappy (state error) |
| renewal triggers | Improve Fate expended, renew on each of `initiativeRolled`, `shortRestFinished`, `longRestFinished` | available (one test each) | p. 88 | happy |
| renewal of nothing | nothing expended | same state, no events | p. 88 | unhappy (empty case) |
| not held, no limit | expend a feat not held; expend Alert | `FeatNotHeld`; `FeatHasNoLimitedUse` | p. 87 | unhappy (state error) |
| spell change use case | `ChangeMagicInitiateSpell` with the new spell found, missing, domain-rejected | `...Succeeded`; `...ContentMissing`; `...Rejected`; no panic | p. 87 | unhappy (missing content) |

### Table S: Skilled choice cases (FR-038 to FR-041)

| Label | Input | Expected | Cite | Path |
|---|---|---|---|---|
| any combination | Insight, Perception, Stealth; three distinct tools; two skills and one tool; one skill and two tools | accepted (one test each) | p. 87 | happy |
| two, four or no picks | two skills; four skills; none | an `srderr.ErrInvalidInput` error (`NewSkilledChoices`, kind `wrongCount`) | p. 87 | unhappy (boundary) |
| repeated skill or tool | Insight twice, Stealth; one tool twice | an `srderr.ErrInvalidInput` error (`NewSkilledChoices`, kind `duplicate`) | p. 87 | unhappy (invalid input) |
| already proficient | a skill the character already has | accepted; listed in `ProficiencyGrants` | p. 87 | unhappy (rule interaction) |
| missing tool | a tool id the content source does not have | `TakeFeatContentMissing` | p. 87 | unhappy (missing content) |
| second Skilled | holds Skilled; second with three other picks | accepted | p. 87 | happy |

### Table P: prerequisite cases (FR-009 to FR-016)

| Label | Feat | Character | Expected | Cite | Path |
|---|---|---|---|---|---|
| no prerequisite | Alert | level 1 | qualifies | p. 87 | happy |
| ASI level 4 | Ability Score Improvement | level 4 | qualifies | p. 87 | happy |
| ASI level 3 | Ability Score Improvement | level 3 | `FeatPrerequisiteNotMet` (level 4) | p. 87 | unhappy (boundary) |
| Grappler qualifies | Grappler | level 4, STR 13, DEX 8; level 4, STR 8, DEX 13 | qualifies (one test each) | p. 87 | happy |
| Grappler scores low | Grappler | level 4, STR 12, DEX 12 | unmet: ability clause only | p. 87 | unhappy (boundary) |
| Grappler level low | Grappler | level 3, STR 13, DEX 13 | unmet: level clause only | p. 87 | unhappy (boundary) |
| Grappler both unmet | Grappler | level 3, STR 12, DEX 12 | unmet: level clause, then ability clause | p. 87 | unhappy (rule interaction) |
| Grappler self-increase | Grappler, choice Strength | level 4, STR 12, DEX 12 | unmet (FR-013) | p. 87 | unhappy (rule interaction) |
| Epic Boon 19 and 20 | Boon of Fate | level 19; 20 | qualifies | p. 88 | unhappy (boundary) |
| Epic Boon 18 | Boon of Fate | level 18 | unmet (level 19) | p. 88 | unhappy (boundary) |
| Spell Recall ok | Boon of Spell Recall | level 19, Spellcasting held | qualifies | p. 88 | happy |
| Spell Recall unmet | Boon of Spell Recall | level 19, no Spellcasting; level 18, Spellcasting held | unmet: feature clause; level clause (one test each) | p. 88 | unhappy (rule interaction) |
| Archery | Archery | Fighter level 1, Fighting Style held; level 4, none | qualifies; unmet: feature clause | p. 87 | unhappy (rule interaction) |
| later loss | Ability Score Improvement (taken at level 4) | level later reduced in the state passed in | feat stays held, benefits kept | p. 87 | unhappy (state error) |

### Withdrawn ids

| Id | Status |
|---|---|
| FR-021 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-032, FR-033); the mapping to `FeatIncreaseRejected` is FR-079 |
| FR-022 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-033); unchanged-on-rejection is FR-069 and FR-079 |
| FR-025 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-034, FR-035) |

## 5. Model

All in package `character` (domain files `feat_*.go`, use-case files `*_feat_usecase.go`), pure (no ports, `context` or clock in the domain files).

**Held state**

- `CharacterFeats` (value object): ordered `TakenFeat`s, copied on construction and on read; the zero value is the empty set;
  `Holds(FeatID)`, `TakenFrom(FeatSource)` (FR-062), `Effects(...)` (FR-044),
  `GrantedSpells`, `ProficiencyGrants`, `UseStateOf(FeatID)`; not part of the
  `Character` (Q4). `TakenFeat`: `FeatID`, `FeatCategory`, `FeatSource`, `FeatChoices`,
  limited-use state(s); the choices shape matches the feat id (FR-061).
- `FeatSource` (closed interface with an unexported marker method): `BackgroundFeat{BackgroundID}`, `CustomBackgroundFeat`,
  `SpeciesTraitFeat(speciesID, traitID)`, `ClassFeatureFeat(classID, featureID)`,
  `EldritchInvocationFeat(optionID)` (`FeatureOptionID`, content-classes.md FR-061),
  `BonusFeat`. `FeatGrant`: `FeatSource` and `FeatGrantScope` (closed:
  `SpecifiedFeat(featID, requiredSpellList)`, `FeatFromCategory(FeatCategory)`,
  `AnyFeat`); `RequiredSpellList` is closed: `NoRequiredSpellList`,
  `RequiredSpellListIs(ClassID)`.
- `HeldFeatures`: an immutable set of `FeatureID`s (copied on construction and on read) (FR-065), a plain input (Q19).
  `ProficiencyGrants` (skills and tools gained, merged by the host, Q7);
  `FeatGrantedSpells` (cantrips, the always-prepared level 1 spell, the ability, Q8);
  `FeatUseState` (closed: `FeatUseIsAvailable`, `FeatUseIsExpended`); `FeatRenewal`
  (`initiativeRolled`, `shortRestFinished`, `longRestFinished`).

**Choices (closed `FeatChoices`)**

- `NoFeatChoices`; `SingleAbilityIncrease(Ability)` (one ability by 1, checked against
  the feat per table C, then converted: general constructor for Grappler, `epicBoon`
  for Epic Boons); `AbilityScoreImprovementChoices` (`oneByTwo`, `twoByOne`,
  built by `NewAbilityScoreImprovementChoices` (returns a `*FeatChoicesError` on invalid input); converts to the `AbilityScoreIncrease` of
  level-advancement-and-multiclassing.md).
- `MagicInitiateChoices`: list `ClassID`, `Ability`, two cantrip `SpellID`s, one level 1
  `SpellID`, and the total levels at which Spell Change was used (FR-036).
  `SkilledChoices`: three picks, each `SkilledSkill(Skill)` or `SkilledTool(ItemID)`;
  built by `NewSkilledChoices`, which returns a `*FeatChoicesError`; a failed construction holds a non-empty slice (copied on read)
  of `FeatChoiceProblem`.
- `FightingStyleChoice` (shared-types-and-ownership.md FR-020): one `FeatID` of a
  Fighting Style feat, `ToTakeFeat()` (FR-042); no class-option variant (Q2).
- `FeatChoiceProblem` (`name`: `FeatChoiceName`, `kind`: `FeatChoiceProblemKind`).
  Names: `increasedAbility`, `spellList`, `spellcastingAbility`, `cantrips`,
  `levelOneSpell`, `skilledPicks`, `replacementSpell`, `featChoicesShape`. Kinds:
  `missing`, `unexpected`, `abilityNotAllowed`, `listNotAllowed`, `wrongCount`,
  `duplicate`, `notACantrip`, `notLevelOne`, `notOnSpellList`, `levelMismatch`,
  `listAlreadyUsed`.

**Domain services** (stateless)

- `FeatQualification.Check(Feat, FeatQualificationFacts)` returns the unmet clauses
  (FR-008 to FR-016); the facts hold the total `CharacterLevel`, `AbilityScores`,
  `HeldFeatures` and the held feat ids.
- `FeatTakingRules.Take(Character, CharacterFeats, Feat, FeatGrant, FeatChoices,
  HeldFeatures, resolvedSpells map[SpellID]Spell)` returns the closed `FeatTaking` (interface with an unexported marker method):
  `FeatTakingChanged` (`Character`, `CharacterFeats`, `ProficiencyGrants`,
  `FeatGrantedSpells`, events) or `FeatTakingRejected{Rejection}`. It applies the
  checks in the order of table O, then the benefits, applying an increase with
  `AbilityScoreIncreaser`. `HeldFeatures` is an argument so character-creation.md
  FR-105 and FR-108 can supply it. Tool picks are `ItemID`s the use case checks exist.
- `FeatEffects`: `Effects(CharacterFeats, ...)` (table E),
  `InitiativeModifier(CharacterFeats, ProficiencyBonus)` (FR-045),
  `ArmorClassBonuses(CharacterFeats, []ArmorCategory)` (FR-046),
  `GreatWeaponFighting.Treat(rolled int)` (FR-047), `FreeCastingRule.Resolve(SlotLevel,
  roll)` (FR-053; `FreeCasting` is the `FeatEffect` variant only).
- `FeatUses`: `Expend(CharacterFeats, FeatID)`, `Renew(CharacterFeats, FeatRenewal)`
  (FR-032, FR-050 to FR-052). `MagicInitiateSpellChange.Replace(CharacterFeats, FeatID
  instance, oldSpell, newSpell, LevelGained)` (FR-034 to FR-037).
  `FeatReplacement.Replace(...)` (FR-082), returning `FeatTaking`.

**Declared effects (closed `FeatEffect`, published language).** The 19 variants
are those of table E (field types there): immutable value objects of structured
facts, no display text, interpreted by `combat` or the host, never by `d20`.
New exported types, enums: `FeatTriggerAction` (`attackAction`, `magicAction`,
`bonusAction`, `reaction`, `action`); `HandsUsed` (`twoHands`);
`FeatInitiativeTiming` (`afterRollingInitiative`); `FeatSwapPartner`
(`willingAllyInSameCombat`); `FeatTrigger` (`weaponHit`, `natural20OnAttackRoll`);
`FeatSizeRelation` (`sameSizeOrSmaller`); `FeatDestination`
(`unoccupiedSpaceYouCanSee`). Closed (interface with an unexported marker method): `FeatDamageTypeRule` (`SameAsAttackDamageType()`,
`SpecificDamageType(DamageType)`; the closed `DamageType` cannot hold "the type of
the attack"); `FeatUsageLimit` (`NoUsageLimit`, `OncePerTurn`,
`OnceUntilStartOfYourNextTurn`, `UseRenewedBy([]FeatRenewal)`). Light levels are
`LightLevel` of `content`, defined once (shared-types-and-ownership.md FR-053).

**Events** (past tense, FR-063, FR-080): closed `FeatEvent` satisfies
`character.Event`: `FeatTaken`, `FeatSpellReplaced`, `FeatUseExpended` (the event; the
state is `FeatUseIsExpended`), `FeatUsesRenewed`, `FeatReplaced` (FR-082). Increases reuse
`AbilityScoreChanged` and `HitPointMaximumChanged` (character-aggregate.md).

**Failures.** `FeatRejection` (closed): `FeatBehaviourUnknown`, `FeatOutsideGrant`,
`FeatNotRepeatable`, `FeatPrerequisiteNotMet(unmetClauses)`,
`FeatChoicesInvalid(problems)`, `FeatIncreaseRejected(AbilityScoreIncreaseRejection)`,
`SpellChangeNotAllowedNow`, `SpellChangeAlreadyUsed`, `FeatUseAlreadyExpended`,
`FeatNotHeld`, `FeatHasNoLimitedUse`, `FeatNotReplaceable`. Invalid values (a score below 1, a slot level
outside 1 to 9, a blank feature name, a choices shape not matching its feat id) are
returned `srderr.ErrInvalidInput` errors from `NewX` (`MustX` panics; architecture.md §6). The domain does no I/O, so it has no
`*srderr.PortError`, and imports only the kernel packages and package `content` and its own
context.

## 6. Interface

| Use case | Command | Result (closed) | Ports |
|---|---|---|---|
| `TakeFeat` | `Character`, `CharacterFeats`, `FeatID`, `FeatGrant`, `FeatChoices`, `HeldFeatures` | `TakeFeatResult`: `TakeFeatSucceeded` (`Character`, `CharacterFeats`, `ProficiencyGrants`, `FeatGrantedSpells`, events), `TakeFeatRejected` (`FeatRejection`), `TakeFeatContentMissing` (`MissingContent` list) | `content.Source` (feat, spells, tools in the choices) |
| `ChangeMagicInitiateSpell` | `Character`, `CharacterFeats`, the instance (its list), old and new `SpellID`, the `LevelGained` event | `ChangeMagicInitiateSpellResult`: `...Succeeded` (`CharacterFeats`, `FeatSpellReplaced`), `...Rejected`, `...ContentMissing` | `content.Source` for the new spell |
| `ExpendFeatUse` | `Character`, `CharacterFeats`, `FeatID` | `ExpendFeatUseResult`: `...Succeeded` (`CharacterFeats`, `FeatUseExpended`), `...Rejected` | none |
| `RenewFeatUses` | `Character`, `CharacterFeats`, `FeatRenewal` | `RenewFeatUsesResult`: `...Succeeded` (`CharacterFeats`, events, possibly none) | none |
| `RollFreeCasting` | `Character`, `CharacterFeats`, `SlotLevel` (1 to 9) | `RollFreeCastingResult`: `...Resolved` (roll, `FreeCastingOutcome`: `FreeCastingSpared` or `FreeCastingSpent`), `...NotApplicable` (`slotAboveFour`), `...Rejected` (`FeatNotHeld`) | `Roller` (in-memory, synchronous) |

Each use case is a type built by `NewX(opts ...Option)` (`NewRollFreeCasting(roller Roller, opts ...Option)`, `NewTakeFeat(source content.Source, opts ...Option)`) with one method `Run(ctx context.Context, cmd <UseCase>Command) (<UseCase>Result, error)`; the `error` return carries only port failures. Each command is `<UseCase>Command`; rejected variants carry a `FeatRejection`; variants
carry the operation prefix (architecture.md §4, shared-types-and-ownership.md FR-017).
Missing content is `MissingContent(kind, id)` (FR-018 there; character-aggregate.md
FR-077). Content is addressed by the `content.Key` of kind Feat with `featID`, the key of kind Spell, and
the `content.Key` of a `Tool` entry (content-equipment.md); the use case passes the found
`Feat` and `Spell` values to `FeatTakingRules`, so the domain stays synchronous. Ports:
`content.Source` (a failure is returned wrapped once, FR-068), the consumer-side `Roller` interface declared in `character`'s `ports.go` (only
`RollFreeCasting`; satisfied structurally by `dice.Roller` implementations) and the telemetry bundle (zero-value `telemetry.Noop` by default); no new port, double or contract suite.

**Exports.** Package `character` exports every
type named in sections 3 to 6 with its variants (no re-export package); the package `doc.go`
lists each. Every name is unique in the exported-name scan (architecture.md §9);
`BardFeatChoice` (class-bard.md) and `FeatChoiceSlot` (class-druid.md) are not defined
here (Q3).

## 7. Observability

The domain is **silent by design**: it returns events, rejections and effect values.
The five use cases are instrumented (observability.md §2) with signals owned by
character-aggregate.md; this spec adds values and two attributes (Q21), no new counter. Failures: missing content (`srd5e.outcome=failure`, `result=content_missing`,
span error status); port or dice failure (span error status, `error.type`); rejection
(`failure`, `result=rejected`, `srd5e.character.rejection`, span status not error);
invalid arguments are bugs. Ids and scores stay in results and events, not attributes.
A Constitution increase is an `AbilityScoreChanged` event, not counted again.
`srd5e.outcome` and `error.type` come from the attribute-key constants of package `telemetry` (errors-and-results.md
section 7; a returned failure has no `error.type`). No logs, no per-effect or per-feat signal.

- `srd5e.character.operation`: `take_feat`, `change_magic_initiate_spell`,
  `expend_feat_use`, `renew_feat_uses`, `roll_free_casting`.
- `srd5e.character.result` (to `srd5e.outcome`): `taken` (`success`; `take_feat`),
  `changed` (`success`; the other non-dice operations, including a no-op),
  `resolved` (`success`; a roll), `not_applicable` (`success`; `slotAboveFour`),
  `rejected` (`failure`; includes `feat_not_held`), `content_missing` (`failure`).
- `srd5e.character.rejection`: the snake_case of each `FeatRejection` variant
  (`feat_behaviour_unknown` to `feat_not_replaceable`, section 5).
- Optional attributes on the counter and spans of `take_feat` only (not the
  histogram): `srd5e.character.feat_category` (`origin`, `general`,
  `fighting_style`, `epic_boon`; omitted when the feat is not found) and
  `srd5e.character.feat_source` (`background`, `species_trait`, `class_feature`,
  `eldritch_invocation`, `bonus_feat`; a custom background is `background`).

In the table, bare attribute names stand for `srd5e.character.<name>`, except `outcome` (`srd5e.outcome`).

| Signal | Type, unit | Attributes | Emitted by | Purpose | Monitoring |
|---|---|---|---|---|---|
| `srd5e.character.operations` | counter, `{operation}` | `operation`, `outcome`, `result`, `rejection`, `feat_category`, `feat_source` | all five | Existing; this spec adds values. Question 4: what is counted | A rise in `feat_prerequisite_not_met` or `feat_outside_grant` (a host UI offering feats a character cannot take); any `feat_behaviour_unknown` or `feat_not_held` |
| `srd5e.character.duration` | histogram, `s` | `operation` only | all five | Existing; this spec adds values. Question 4: what is timed | Latency regression within one operation |
| `srd5e.character.take_feat` | span | `outcome`, `result` (`taken`, `rejected`, `content_missing`), `feat_category`, `feat_source`, `rejection` | `TakeFeat` | Question 3: which rule or missing entry stopped it | Error status rate above 0 sustained |
| `srd5e.character.change_magic_initiate_spell` | span | `outcome`, `result` (`changed`, `rejected`, `content_missing`), `rejection` | `ChangeMagicInitiateSpell` | Diagnosis of a refused Spell Change | As above |
| `srd5e.character.expend_feat_use` | span | `outcome`, `result` (`changed`, `rejected`), `rejection` | `ExpendFeatUse` | Diagnosis of a refused expend | Counter only |
| `srd5e.character.renew_feat_uses` | span | `outcome`, `result` (`changed`) | `RenewFeatUses` | Confirms renewals ran | Counter only |
| `srd5e.character.roll_free_casting` | span | `outcome`, `result` (`resolved`, `not_applicable`, `rejected`), `rejection` (`feat_not_held`); error status on a dice failure | `RollFreeCasting` | Diagnosis of a dice failure or a wrongly offered Free Casting | Error status rate above 0 sustained |

Attribute values are low-cardinality; no ids, spell names, scores or levels
(observability.md §5). Entries feed `doc/observability/signals.md`.

## 8. Test plan

Unit for domain rules (`character/feat_*_test.go`, package `character_test`, table-driven), application for the five use
cases (`character/*_feat_usecase_test.go`), `Fuzz*` tests for `NewAbilityScoreImprovementChoices` and `NewSkilledChoices`, `Example*` functions for `TakeFeat`; no contract suite.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-004, FR-076 | unit | Table G, one test per row; each scope with a feat inside and outside it; all four categories under `AnyFeat` |
| FR-005 to FR-016 | unit | Tables P and N, one test per row; hold then retake from each source, with a feat built with `repeatable: true`; every-unmet-clause assertion for FR-012 |
| FR-017 to FR-019 | unit | Table C, one test per row; problem lists asserted whole |
| FR-020, FR-023, FR-024, FR-026, FR-079 | unit | Table A, one test per row; `NewX` returns an error exactly for the invalid inputs; maximum 20 or 30 per feat; rejection leaves state unchanged |
| FR-027 to FR-037 | unit | Table M; `LevelGained` events for the right and wrong character and level |
| FR-038 to FR-043 | unit | Table S, one test per row; `ToTakeFeat()`; a General feat in a `FightingStyleChoice` gives `FeatOutsideGrant` |
| FR-044 to FR-049 | unit | Table E, one test per feat, one construction test per variant; the boundary cases listed in FR-045 to FR-047; Overwhelming Strike uses the score |
| FR-031 to FR-033, FR-050 to FR-052 | unit | Table U, one test per row: granted spells, slot cast leaves free-cast state, expend twice, each renewal trigger, no-op renewal, `FeatNotHeld`, `FeatHasNoLimitedUse` |
| FR-053 to FR-056, FR-077, FR-078 | unit + application | Table F, one test per row; `FixedDiceRoller`, one case recording no call for levels 5 to 9; a failing roller's error is returned wrapped once, recorded once |
| FR-057 to FR-065 | unit | Mutating every returned collection changes nothing; value equality; inputs unchanged; table O; `NewX` returns an `srderr.ErrInvalidInput` error; `TakenFrom`; event fields; determinism |
| FR-066 to FR-075 | application | `contenttest.FakeSource` (all found, some missing, one failing with an error) for `TakeFeat`, and for `ChangeMagicInitiateSpell` (FR-070: replacement found, missing, rejected); lookups attempted before returning; sync use cases use no port; inputs unchanged; recording the telemetry bundle: span names, attributes, counter, histogram, no log, `srd5e.outcome` only `success` or `failure`, `error.type` only for thrown exceptions, failures recorded once, rejection not an error status |
| FR-080, FR-081 | unit + architecture | Result roots closed, variants prefixed; exhaustive type switch over `FeatEvent`; no generic variant name; `TakeFeat` records exactly its own spans and counter; no `*_usecase.go` file calls another use case |
| FR-082 | unit | Replace a Fighting Style feat by another (old effects gone, `FeatReplaced` then `FeatTaken`); a Magic Initiate replaced loses its use state; replace with a feat not held gives `FeatNotHeld`; replace Ability Score Improvement gives `FeatNotReplaceable`; the new feat repeating the old one is accepted (old removed first); new feat failing table O gives that rejection and no change; FR-083: `replacing` with old not held, a new feat equal to another held non-Repeatable feat (`FeatNotRepeatable`), a Repeatable one, Fighting Style and `AnyFeat` cases; FR-084: `NoBonusFeats` 0, `BonusFeatsFromXp` at 355,000 / 384,999 / 385,000 / 415,000 with 0, 1 and 2 `BonusFeat` feats held (never negative) |
| (architecture) | architecture | domain files import only `content`, the kernel packages and own context (check A1); no duplicate exported name (`FeatUseExpended`, `FeatReplaced`, `SlotNotExpended`, `FreeCasting`, `FeatTaking`, `FeatChoiceProblem`, `RequiredSpellList`, the effect field types); signal names are constants matching `doc/observability/signals.md` |

Withdrawn ids need no tests. Doubles: `FakeContentSource`, `FixedDiceRoller`,
the recording doubles of `telemetry/telemetrytest` (shared-types-and-ownership.md FR-038).
A scenario test (a Soldier takes Savage Attacker, gains ASI at level 4) is
recommended once character creation exists. Mutation testing: every changed
domain file, threshold 80 (testing.md §8; `gomu` runs only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`).

## 9. Open questions and decisions

No question is open or escalated. Settled, one line each (the "amend" notes are follow-up edits to other specs, which this spec assumes applied):

- Q1: checks belong to `FeatTakingRules` (FR-001 to FR-007, FR-043); class specs keep the level gate, "only once" and replacement timing; amend class-paladin.md FR-038 (use `FeatOutsideGrant`) and class-ranger.md FR-051 (drop `FightingStyleClassOption`).
- Q7: Skilled accepts a pick the character already holds (FR-040; interpretation: p. 87 says "any combination of three skills or tools" and states no exclusion); a tool pick must be a `Tool` entry the source holds, a tool family is the host's concern; `ProficiencyGrants` is the handoff.
- Q8: Magic Initiate spells belong to no class: spellcasting.md casts them as `CastMode.FeatureGranted` using `FeatGrantedSpells.Ability` instead of a `SpellcastingClass` (p. 87); the free cast is gated by the host calling `ExpendFeatUse` first, since no use case calls another (FR-033, FR-081); amend spellcasting.md.
- Q10: per-turn limits are `combat`'s turn state, which records uses of a `FeatUsageLimit` (FR-049); amend initiative-and-actions.md and encounter-aggregate.md.
- Q11: the Alert Initiative Swap (p. 87) is a `SwapInitiative` use case in initiative-and-actions.md consuming `InitiativeSwap`, rejected when the character or the ally is Incapacitated; amend initiative-and-actions.md.
- Q17: the caller builds the `BackgroundFeat`/`SpeciesTraitFeat` grant per table G: the host calls `TakeFeat` after `CreateCharacter`, or a downstream use case composes `FeatTakingRules` (FR-081); class-bard.md FR-093 (decided) builds the `ClassFeatureFeat`/`AnyFeat` grant and composes `FeatTakingRules.Take`; amend class-ranger.md FR-049 to say so.
- Q19: the host builds `HeldFeatures` as the union of `FeatureID`s from each class spec's held-features query, matched by printed name per table N; amend content-classes.md so a class feature's `FeatureID` resolves to its printed name.
- Q20 (settled): replacing a held feat (Fighter p. 47; Wish p. 175) is `CharacterFeats.Replacing` (FR-083) inside `FeatReplacement` (FR-082); Wish uses `AnyFeat`, the Fighter swap `FeatFromCategory(Fighting Style)`. Interpretation: an increase cannot be undone (no score reduction exists), so a feat that gave one is `FeatNotReplaceable`; class-fighter.md and wish-spell.md consume it (spell-effects.md does not mention Wish).
- Q21: amend character-aggregate.md section 7: add `TakeFeat` and `ChangeMagicInitiateSpell` to the `load_content` emitters, `taken`, `resolved`, `not_applicable` to `result`, and `feat_category`, `feat_source` (`take_feat` only) to the counter and span attributes; histogram stays operation-only.

- Q2: `FightingStyleChoice` is only the feat selection (shared-types-and-ownership.md FR-020; FR-042, FR-043). Q4: `CharacterFeats` lives beside the `Character` (FR-022 there). Q3: `FeatGrant` is the one handoff (table G); class specs derive used grants as owed minus `TakenFrom` (FR-062); `TakeFeat` does not verify entitlement; class-bard.md's local choice type is `BardFeatChoice`.
- Q5: "Spellcasting Feature" excludes "Pact Magic" (FR-011). Q6: no partial increase (level-advancement-and-multiclassing.md FR-032, FR-033; FR-079).
- Q9: the host calls `RenewFeatUses` after each rest (`RestApplied`, rests-and-hit-dice.md FR-060); `combat` calls it with `initiativeRolled` after `RollInitiative`.
- Q12: Overwhelming Strike adds the score (FR-048). Q13: a house feat gets `FeatBehaviourUnknown` (FR-018). Q14: Warlock's different-Origin-feat rule stays in class-warlock.md (FR-030 still applies).
- Q22: rank 15 in `shared-types-and-ownership.md` 4.11, so both imports are listed in "Depends on". Q15: a feat held from two sources counts once (FR-005). Q16: Spell Change keeps the free-cast state (FR-037). Q18: "Level 4+" is total level (FR-009).

## 10. Changelog impact

Added, under `Unreleased` (new API, not breaking before 0.1.0): the five feat use
cases, `CharacterFeats`, `FeatGrant`, `FeatChoices`, `FeatEffect`, the feat events,
`FightingStyleChoice`, the signals of section 7 and the new values and optional
attributes of `srd5e.character.operation`, `.result` and `.rejection`. Not
**BREAKING**. Follow-up edits to other specs (Q1, Q3, Q8 to Q11, Q17, Q20, Q21)
become changelog entries when their code lands.
