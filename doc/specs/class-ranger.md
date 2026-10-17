# Class: Ranger

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `ability-scores.md`, `observability-ports.md`, `content-schema-and-decoder.md`, `content-source-port.md`, `content-classes.md`, `content-spells.md`, `content-monsters.md`, `derived-stats.md`, `character-aggregate.md`, `rests-and-hit-dice.md`, `feats-behaviour.md`, `weapon-properties-and-mastery.md`, `spellcasting.md`, `level-advancement-and-multiclassing.md`, `character-creation.md` |

## 1. Purpose and scope

Specifies the Ranger as SRD 5.2.1 presents it (pp. 57-61): Core Ranger Traits,
the Features table, every class feature from level 1 to 20, the spell list and
the one subclass, Hunter. Two parts:

1. **Content data** (lands in phase 3 with content-classes.md, shared-types-and-ownership.md FR-014; definition, traits, Features table, feature entries, spell
   list membership, Hunter) is owned by content-classes.md
   (shared-types-and-ownership.md FR-025); requirements that only describe or
   validate it are in "Withdrawn ids" (4.12). T6, T7 and T10 stay as the SRD
   transcription the behaviour requirements read.
2. **Behaviour in the `character` context**: value objects, pure domain
   functions and three use cases (FR-111) for Favored Enemy uses, Weapon Mastery
   parameters, Deft Explorer, Expertise, Fighting Style (including Druidic
   Warrior), feat grants, Roving, Tireless, Relentless Hunter, Nature's Veil,
   Precise Hunter, Feral Senses, Foe Slayer and the Hunter features.

Effects resolved by another context (Resistance, Temporary Hit Points, Invisible,
moving, resolving an attack) are only structured data here (a grant, a rider, an
event); applying them is specified where that rule lives.

**Out of scope:**

- Spell text, including *Hunter's Mark*, *Cure Wounds*, *Ensnaring Strike*,
  *Guidance*, *Starry Wisp* (pp. 104-175). Used from the Ranger pages only: the
  id, always prepared, a creature can be marked, the d6 (Foe Slayer, p. 59),
  Concentration can be held. The "recommended" spells, cantrips and feat are
  recommendations, not rules.
- Slots, Prepared Spells, Spellcasting Focus and the Concentration state shape:
  spellcasting.md; casting and Concentration transitions: casting-and-concentration.md
  (shared-types-and-ownership.md FR-011). This spec passes on the
  always-prepared fact (FR-031) and the Druidic Warrior cantrips (FR-052).
  FR-078 is a rule for combat-action-resolution.md. Half-caster arithmetic is
  spellcasting.md FR-032 (H4); FR-030 only points to it.
- The Druid cantrip list, language tables, feat descriptions (pp. 87-88),
  weapons, armor and starting items: content specs (ids and names only here).
  General multiclassing and combining Extra Attack (p. 25):
  level-advancement-and-multiclassing.md, derived-stats.md (FR-068).
- Rest, Speed, Temporary Hit Points, Exhaustion, Invisible, Opportunity Attack,
  action economy, Resistance and Concentration save rules, positions and
  visibility: inputs and outputs only. "Bonus Feats at Level 20" (p. 24); other
  subclasses (only Hunter, p. 61); anything not in SRD 5.2.1.

## 2. Source

PDF page numbers equal printed page numbers (checked: PDF pages 28 and 57-61).

- SRD 5.2.1, "Core Ranger Traits", "Becoming a Ranger ...", "Ranger Class
  Features", "Level 1: Spellcasting", p. 57
- "Ranger Features" table; "Level 1: Spellcasting"; "Level 1: Favored Enemy";
  "Level 1: Weapon Mastery" (begins), p. 58
- "Level 1: Weapon Mastery" (continues); Levels 2 Deft Explorer, 2 Fighting
  Style, 3 Ranger Subclass, 4 Ability Score Improvement, 5 Extra Attack, 6
  Roving, 9 Expertise, 10 Tireless, 13 Relentless Hunter, 14 Nature's Veil, 17
  Precise Hunter, 18 Feral Senses, 19 Epic Boon, 20 Foe Slayer, p. 59
- "Ranger Spell List", p. 60; "Ranger Subclass: Hunter", p. 61
- Cross-checks only: "Character Advancement", p. 23; "Multiclassing", p. 25;
  "Spells", pp. 104-175 (class lists, T7); "Hunter's Mark", p. 141 (d6); Rules
  Glossary, "Grappled", p. 182 (FR-070); "Feat Descriptions", pp. 87-88

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Ranger, Hunter | `ClassId` `ranger`, `SubclassId` `hunter` (content) | Definitions are content-classes.md data. |
| Ranger spell list, Favored Enemy, *Hunter's Mark* | Spell membership (T7); feature with uses; `SpellId` `hunters_mark` | 48 spells; *Hunter's Mark* always prepared with free casts; spell content out of scope. |
| Weapon Mastery | `WeaponMasteryChoice` (weapon-properties-and-mastery.md) | Two kinds; this spec supplies count two and the Long Rest moment. |
| Deft Explorer, Expertise | Feature (choices) | One skill and two languages; Expertise at 2 and 9. |
| Fighting Style | Feature (choice) | `RangerFightingStyleFeat(featId)` (the `FeatId` of a `FightingStyleChoice`; handoff `toTakeFeat()`), or `DruidicWarrior(cantrips)`, a Ranger-owned case of the closed `RangerFightingStyle`; feats-behaviour.md has no counterpart. |
| Ability Score Improvement, Epic Boon, Tireless, Nature's Veil | Feature (feat grant; uses) | Levels 4, 8, 12, 16 and 19; Temporary Hit Points and Decrease Exhaustion; Invisible until end of next turn. |
| Hunter's Prey, Defensive Tactics | Subclass feature (choice) | Content option ids `colossus_slayer`, `horde_breaker`; `escape_the_horde`, `multiattack_defense` (content-classes.md table OP). No Ranger enum. |
| Magic action, Bonus Action, Reaction | `RangerActionCost` | Ranger-prefixed: the class-specific prefix keeps the package API names unique (shared-types-and-ownership.md FR-017, FR-021); cf. `DruidActionCost` in class-druid.md. |
| Other Ranger features and glossary terms | | Extra Attack, Roving, Relentless Hunter, Precise Hunter, Feral Senses, Foe Slayer, Hunter's Lore, Superior Hunter's Prey and Defense: SRD names; glossary terms are used by name only. |

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

Stable ids; never renumber. "Typed result" means a closed result variant, never
an error (`doc/rules/architecture.md` §6). Untrusted host choices are
validated by a `New` constructor returning `(T, error)`; `MustX` helpers
panic for programmer errors.

Naming (shared-types-and-ownership.md FR-017): the short rejection names
(`NoUsesRemaining`, `WrongMoment`, `SameOption`, `FeatureNotAvailable`,
`AlreadyChosen`, `NothingToReplace`, `UnknownOption`, `NotProficient`,
`AlreadyHasExpertise`, `NoGrantRemaining`, `InvalidChoice`, `NotHunter`) are
values of the single `RangerRejection` enum (`RangerRejection.noUsesRemaining`,
...). Reasons feats-behaviour.md or weapon-properties-and-mastery.md decide
(category, repeatability, count, eligibility, moment) are mapped from their
rejection types into `InvalidChoice` or `WrongMoment`, never re-decided.
`content_missing` and `not_a_ranger` come from the `...ContentMissing` and
`...NotARanger` variants. The short result names (`NoneApply`, `Reveals`,
`NotMarked`, `Ineligible`, `ContentMissing`, `NotARanger`) stand for the
operation-prefixed variants of section 5.

### 4.1 Features table and spell list

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-023 | Favored Enemy (read from the row's Favored Enemy column through `ClassDefinition.rowAt` and `column`, content-classes.md FR-047, FR-050, never re-stored here) MUST rise exactly at Ranger levels 5, 9, 13 and 17. Prepared Spells MUST rise exactly at levels 2, 3, 4, 5, 7, 9, 11, 13, 15, 17 and 19 (from level 1 as the starting row) and be unchanged at 6, 8, 10, 12, 14, 16, 18 and 20 (T2). | SRD 5.2.1, "Ranger Features", p. 58 | unhappy (boundary) |
| FR-030 | Ranger levels are counted with Paladin levels in the half-caster sum of spellcasting.md FR-032 (table H4). This spec defines no half-level arithmetic; it requires only that the Ranger `ClassId` is supplied to that calculation as a half-caster class. | SRD 5.2.1, "Multiclassing", "Spell Slots", p. 25 | happy (consumer-side) |

#### T1: Favored Enemy and Prepared Spells by level (reference for FR-023, FR-032)

Source: SRD 5.2.1, "Ranger Features", p. 58. The Proficiency Bonus, Class Features and Spell Slots columns are content-classes.md data. Levels 1 to 20: Favored Enemy 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 6, 6, 6, 6; Prepared Spells 2, 3, 4, 5, 6, 6, 7, 7, 9, 9, 10, 10, 11, 11, 12, 12, 14, 14, 15, 15.

#### T2: Favored Enemy and Prepared Spells steps (FR-023, one test per row)

Source: SRD 5.2.1, "Ranger Features", p. 58. Every row is unhappy (boundary). Each row is from-level to to-level, Favored Enemy, Prepared Spells: FE-01 4 to 5, 2 to 3, 5 to 6; FE-02 8 to 9, 3 to 4, 7 to 9; FE-03 12 to 13, 4 to 5, 10 to 11; FE-04 16 to 17, 5 to 6, 12 to 14; FE-05 1 to 2, 2 to 2, 2 to 3; FE-06 5 to 6, 3 to 3, 6 to 6; FE-07 19 to 20, 6 to 6, 15 to 15; FE-08 6 to 7, 3 to 3, 6 to 7; FE-09 18 to 19, 6 to 6, 14 to 15.

#### T7: Ranger spell list (reference for FR-031, FR-051)

Source: SRD 5.2.1, "Ranger Spell List", p. 60. 48 spells; no cantrip. This spec needs only `hunters_mark`.

| Level | Spells |
|---|---|
| 1 | Alarm, Animal Friendship, Cure Wounds, Detect Magic, Detect Poison and Disease, Ensnaring Strike, Entangle, Fog Cloud, Goodberry, Hunter's Mark, Jump, Longstrider, Speak with Animals |
| 2 | Aid, Animal Messenger, Barkskin, Darkvision, Enhance Ability, Find Traps, Gust of Wind, Lesser Restoration, Locate Animals or Plants, Locate Object, Magic Weapon, Pass without Trace, Protection from Poison, Silence, Spike Growth |
| 3 | Conjure Animals, Daylight, Dispel Magic, Meld into Stone, Nondetection, Plant Growth, Protection from Energy, Revivify, Speak with Plants, Water Breathing, Water Walk, Wind Wall |
| 4 | Conjure Woodland Beings, Dominate Beast, Freedom of Movement, Locate Creature, Stoneskin |
| 5 | Commune with Nature, Greater Restoration, Tree Stride |

### 4.2 Favored Enemy

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | From Ranger level 1 the character MUST always have *Hunter's Mark* (`hunters_mark`) prepared. This spec defines no always-prepared type: the grant is the `AlwaysPreparedSpells` fact on the Favored Enemy entry (content-classes.md FR-055, FR-058). The Ranger passes its spell ids to spellcasting.md (`PreparedSpells.alwaysPrepared`), where it does not count against the Prepared Spells number but is otherwise a Ranger spell. | SRD 5.2.1, "Level 1: Favored Enemy"; "Level 1: Spellcasting", p. 58 | happy |
| FR-032 | The number of times *Hunter's Mark* can be cast without a spell slot MUST equal the Favored Enemy column: 2 at levels 1-4, 3 at 5-8, 4 at 9-12, 5 at 13-16, 6 at 17-20. | SRD 5.2.1, "Level 1: Favored Enemy"; "Ranger Features", p. 58 | happy (T1, T2) |
| FR-033 | The pure function `useFavoredEnemy` (no use case, no dice, no content) with at least one use remaining MUST return `FavoredEnemyUseAccepted`: expended uses rise by one and a `FavoredEnemyCastSpent` event is returned. The last remaining use is accepted. After `FavoredEnemyUseAccepted` the host casts with `CastMode.featureGranted` (casting-and-concentration.md FR-121); after `FavoredEnemyUseRejected` a slot cast is spellcasting.md's. | SRD 5.2.1, "Level 1: Favored Enemy", p. 58 | happy |
| FR-034 | `useFavoredEnemy` with no use remaining MUST return `FavoredEnemyUseRejected` with `NoUsesRemaining` and leave the resources unchanged; casting with a slot is then spellcasting.md's concern. | SRD 5.2.1, "Level 1: Favored Enemy", p. 58 | unhappy (state error) |
| FR-035 | Uses remaining MUST be the level's maximum minus uses expended, never below zero. When a level raises the maximum (5, 9, 13, 17), the expended count MUST be unchanged, so remaining rises by the increase. | SRD 5.2.1, "Level 1: Favored Enemy", p. 58 | unhappy (boundary) |
| FR-036 | Favored Enemy recharges on `RechargeTrigger` long rest (`RestKind` and its mapping are rests-and-hit-dice.md's, shared-types-and-ownership.md FR-009): a Long Rest MUST reset expended uses to zero; a Short Rest MUST leave them unchanged. | SRD 5.2.1, "Level 1: Favored Enemy", p. 58 | unhappy (rule interaction) |

### 4.3 Weapon Mastery

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-037 | The initial Weapon Mastery choice (made at Ranger level 1, no moment or rest) MUST be built by weapon-properties-and-mastery.md `NewWeaponMasteryChoice` with `allowedCount` 2 and `ProficientWeaponEligibility` (FR-014, FR-019 there). The Ranger supplies the count, the eligibility rule and the call; it re-implements no count, distinctness or proficiency check, and maps any mastery problem to `InvalidChoice`. | SRD 5.2.1, "Level 1: Weapon Mastery", pp. 58-59 | happy |
| FR-040 | After the initial choice, the two kinds MAY be replaced only when the character finishes a Long Rest: the Ranger replacement moment `finishedLongRest` maps to `MasteryChangeMoment.longRest` and a Short Rest to `MasteryChangeMoment.shortRest` (`WeaponMasteryChoice.change`, weapon-properties-and-mastery.md FR-024); `gainedRangerLevel` or no moment is rejected in the Ranger with `RangerRejection.wrongMoment` before `change` is called, and the owner's `wrongMoment` maps to `RangerRejection.wrongMoment`. The initial choice (FR-037) needs no rest. | SRD 5.2.1, "Level 1: Weapon Mastery", pp. 58-59 | unhappy (state error) |
| FR-041 | A change after a Long Rest is `WeaponMasteryChoice.change` with `ReplaceAnyKinds` (either or both kinds) and the same proficiency eligibility; the previous pair MUST be returned unchanged on rejection (weapon-properties-and-mastery.md FR-029). | SRD 5.2.1, "Level 1: Weapon Mastery", pp. 58-59 | unhappy (invalid input) |

### 4.4 Deft Explorer and Expertise

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-042 | From Ranger level 2, Deft Explorer MUST be accepted with Expertise in one skill in which the character has proficiency and lacks Expertise. | SRD 5.2.1, "Level 2: Deft Explorer", p. 59 | happy |
| FR-043 | A Deft Explorer skill without proficiency MUST be rejected with `NotProficient`. | SRD 5.2.1, "Level 2: Deft Explorer", p. 59 | unhappy (invalid input) |
| FR-044 | A Deft Explorer skill that already has Expertise MUST be rejected with `AlreadyHasExpertise`. | SRD 5.2.1, "Level 2: Deft Explorer", p. 59 | unhappy (rule interaction) |
| FR-045 | Deft Explorer MUST be accepted with exactly two distinct `Language` values (the closed content vocabulary, shared-types-and-ownership.md FR-005); one, three or a repeated language MUST be rejected with a typed result. A value outside the vocabulary cannot be constructed. | SRD 5.2.1, "Level 2: Deft Explorer" (Languages), p. 59 | unhappy (invalid input) |
| FR-046 | Deft Explorer, Fighting Style and Expertise MUST be rejected with `FeatureNotAvailable` below Ranger levels 2, 2 and 9. | SRD 5.2.1, "Ranger Features", p. 58 | unhappy (boundary) |
| FR-047 | From Ranger level 9, Expertise MUST be accepted with exactly two distinct skills in which the character has proficiency and lacks Expertise, counting Expertise from Deft Explorer. | SRD 5.2.1, "Level 9: Expertise", p. 59 | happy |
| FR-048 | An Expertise choice of one skill, three skills, a repeated skill, a skill without proficiency, or a skill that already has Expertise (including the Deft Explorer skill) MUST be rejected with a typed result naming the reason. | SRD 5.2.1, "Level 9: Expertise", p. 59 | unhappy (invalid input) |

### 4.5 Fighting Style and Druidic Warrior

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-049 | From Ranger level 2, Fighting Style MUST be accepted as a feat (`RangerFightingStyleFeat(featId)`) after `ChooseRangerOption` validates it by composing the domain services `FeatTakingRules` and `FeatQualification` (feats-behaviour.md FR-081, FR-043; a use case never calls a use case, shared-types-and-ownership.md FR-044). It records the choice and returns the `FightingStyleChoice.toTakeFeat()` handoff (FR-042 there); the host calls `TakeFeat` afterwards. The `ClassFeatureFeat` grant (`FeatFromCategory(Fighting Style)` here, `AnyFeat` for FR-062 and FR-064) is built by the caller per feats-behaviour.md table G, not by a Ranger type. Category, qualification and repeatability are decided in feats-behaviour.md (FR-002, FR-005, FR-042, FR-043), not here. The Fighting Style feature satisfies the "Fighting Style Feature" prerequisite those feats name. | SRD 5.2.1, "Level 2: Fighting Style", p. 59; "Fighting Style Feats", pp. 87-88 | happy |
| FR-051 | Instead of a feat, Fighting Style MUST accept `DruidicWarrior` (a Ranger-owned case of `RangerFightingStyle`; feats-behaviour.md defines no counterpart) with exactly two distinct Druid cantrips (`SpellId`s; a Druid cantrip is a cantrip whose class list names Druid, read from content-spells.md, not class-druid.md); a count other than two, a repeated cantrip, a non-cantrip, or a cantrip not on the Druid list MUST be rejected with a typed result. | SRD 5.2.1, "Level 2: Fighting Style" (Druidic Warrior), p. 59 | unhappy (invalid input) |
| FR-052 | The Druidic Warrior cantrips MUST be exposed to spellcasting.md as `GrantedCantrip` values (spellcasting.md FR-097; class `ranger`) so they are Ranger spells with Wisdom as the spellcasting ability, and MUST NOT count toward Prepared Spells or a cantrips-known limit (the Ranger has none, spellcasting.md FR-018). | SRD 5.2.1, "Level 2: Fighting Style" (Druidic Warrior), p. 59; "Ranger Features", p. 58 | unhappy (rule interaction) |
| FR-053 | On gaining a Ranger level, one Druidic Warrior cantrip MAY be replaced with another Druid cantrip not already chosen; replacing two at once, replacing with a non-Druid or non-cantrip spell, or replacing at any other moment (a rest, no trigger) MUST be rejected with a typed result. | SRD 5.2.1, "Level 2: Fighting Style", p. 59 | unhappy (state error) |
| FR-054 | Once Fighting Style is chosen (feat or Druidic Warrior), choosing again MUST be rejected with `AlreadyChosen`; the SRD gives no way to change it. | SRD 5.2.1, "Level 2: Fighting Style", p. 59 | unhappy (state error) |
| FR-055 | A character who chose a feat MUST have no Druidic Warrior cantrips, and one who chose Druidic Warrior MUST have no Fighting Style feat from this feature. This holds by shape: the choice is the closed one-of `RangerFightingStyleFeat(featId)` or `DruidicWarrior(cantrips)`, with no wrapper of `FightingStyleChoice`. | SRD 5.2.1, "Level 2: Fighting Style", p. 59 | unhappy (rule interaction) |

### 4.6 Feat grants

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-062 | The Epic Boon grant is available exactly when `featGrantsAt` (content-classes.md FR-049) reports an Epic Boon category for a level at most the Ranger level (level 19; the feat is validated by `FeatTakingRules`/`FeatQualification` and applied by the host's `TakeFeat` handoff, feats-behaviour.md FR-081); before level 19 a choice MUST be rejected with `FeatureNotAvailable`. | SRD 5.2.1, "Level 19: Epic Boon", p. 59; "Epic Boon Feats", p. 88 | happy |
| FR-064 | Choosing a feat with no grant left MUST be rejected with `NoGrantRemaining` (the feat is then validated and handed off as in FR-049, not applied by a use case); unspent grants stay pending and are never lost. Grants made are counted against the sum of `featGrantsAt` for levels 1 to L (T6). | Package design (grant bookkeeping); SRD 5.2.1, "Level 4: Ability Score Improvement" (levels 8, 12, 16) and "Level 19: Epic Boon", p. 59 | unhappy (state error) |

#### T6: feat grants by Ranger level (FR-062, FR-064, one test per row)

Expected sums of `featGrantsAt`; content-classes.md tests the per-level data. Source: SRD 5.2.1, "Ranger Features", p. 58; Levels 4 and 19, p. 59. Each row is level, Ability Score Improvement grants, Epic Boon: FG-01 3, 0, no (unhappy, one below); FG-02 4, 1, no (happy); FG-03 7, 1, no (unhappy); FG-04 8, 2, no (happy); FG-05 12, 3, no (happy); FG-06 16, 4, no (happy); FG-07 18, 4, no (unhappy, one below); FG-08 19, 4, yes (happy); FG-09 20, 4, yes (unhappy, maximum level).

### 4.7 Extra Attack and Roving

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-065 | At Ranger level 5 and above the character MUST have two attacks per Attack action from this feature; below level 5 it has one. | SRD 5.2.1, "Level 5: Extra Attack", p. 59 | happy |
| FR-066 | Roving MUST add 10 feet to Speed at Ranger level 6 and above while the character is not wearing Heavy armor and the Speed is not flagged unable to increase (FR-070; T3). | SRD 5.2.1, "Level 6: Roving", p. 59 | happy (T3) |
| FR-067 | Roving MUST NOT add to Speed while Heavy armor is worn, and MUST NOT apply below Ranger level 6. | SRD 5.2.1, "Level 6: Roving", p. 59 | unhappy (rule interaction) |
| FR-068 | Extra Attack MUST be exposed as a total number of attacks (2), not a +1 bonus, as the per-class count fed to `ExtraAttackCombination` (level-advancement-and-multiclassing.md FR-054 to FR-058), which takes the greatest count ("the features don't stack"); combining is that spec's. | SRD 5.2.1, "Level 5: Extra Attack", p. 59; "Multiclassing", "Extra Attack", p. 25 | unhappy (rule interaction) |
| FR-069 | From Ranger level 6 the character MUST have a Climb Speed and a Swim Speed each equal to its Speed, including the Roving increase when it applies (also under Heavy armor, without the +10); below level 6 neither from this feature (T3). | SRD 5.2.1, "Level 6: Roving", p. 59 | happy (T3) |
| FR-070 | The Speed input MUST be a base Speed plus a flag `canIncrease` (false when a condition such as Grappled sets Speed to 0, which "can't increase"). A negative base Speed MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error). With `canIncrease` false Roving MUST NOT apply, so a condition-zeroed Speed stays 0 (walk, Climb, Swim). With `canIncrease` true a genuine base Speed of 0 MUST be accepted and yield walk 10 at level 6 without Heavy armor and 0 with it. | SRD 5.2.1, "Level 6: Roving", p. 59; Rules Glossary, "Grappled", p. 182; doc/rules/testing.md §4 | unhappy (boundary) |

#### T3: Roving cases (FR-066, FR-067, FR-069, FR-070, one test per row)

Source: SRD 5.2.1, "Level 6: Roving", p. 59; RV-09 and RV-10 also Rules
Glossary, "Grappled", p. 182. "Can increase" is yes except RV-09 and RV-10.

| Label | Level | Base Speed (ft) | Can increase | Heavy armor | Walk | Climb | Swim | Requirement | Path |
|---|---|---|---|---|---|---|---|---|---|
| RV-01 | 5 | 30 | yes | no | 30 | none | none | FR-066 | unhappy (boundary: one below) |
| RV-02 | 6 | 30 | yes | no | 40 | 40 | 40 | FR-066, FR-069 | happy |
| RV-03 | 6 | 30 | yes | yes | 30 | 30 | 30 | FR-067, FR-069 | unhappy (rule interaction) |
| RV-04 | 20 | 35 | yes | no | 45 | 45 | 45 | FR-066, FR-069 | happy |
| RV-05 | 6 | 25 | yes | no | 35 | 35 | 35 | FR-066, FR-069 | happy |
| RV-06 | 6 | 0 | yes | no | 10 | 10 | 10 | FR-070 | unhappy (boundary: base Speed 0) |
| RV-07 | 6 | 0 | yes | yes | 0 | 0 | 0 | FR-070 | unhappy (boundary: zero Speed) |
| RV-08 | 6 | -5 | yes | no | panic | | | FR-070 | unhappy (invalid input) |
| RV-09 | 6 | 0 | no | no | 0 | 0 | 0 | FR-070 | unhappy (Grappled Speed 0) |
| RV-10 | 6 | 30 | no | no | 30 | 30 | 30 | FR-070 | unhappy (Speed can't increase) |

### 4.8 Tireless, Relentless Hunter, Nature's Veil, Precise Hunter, Feral Senses, Foe Slayer

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-071 | `UseTireless` (the one use case here that needs dice) MUST return `UseTirelessRejected` with `FeatureNotAvailable` below Ranger level 10. | SRD 5.2.1, "Level 10: Tireless", p. 59 | unhappy (boundary) |
| FR-072 | `UseTireless` MUST roll 1d8 through the `dice.Roller` and return Temporary Hit Points equal to the roll plus the greater of 1 and the Wisdom modifier, with action cost Magic action (T4). The minimum of 1 applies to the modifier, not the total. | SRD 5.2.1, "Level 10: Tireless", p. 59 (interpretation: the parenthetical attaches to the Wisdom modifier, as in the same feature's "Wisdom modifier (minimum of once)"; see T4 notes) | happy (T4) |
| FR-073 | Tireless uses MUST be the greater of 1 and the Wisdom modifier (`AbilityModifier`). A use with uses remaining MUST return `UseTirelessAccepted` and emit `TirelessUsed`; with none remaining `UseTirelessRejected` with `NoUsesRemaining` and no dice rolled. | SRD 5.2.1, "Level 10: Tireless", p. 59 | unhappy (boundary) |
| FR-074 | Remaining Tireless uses MUST be maximum minus expended, never below zero, so a Wisdom decrease below the expended count leaves zero. | SRD 5.2.1, "Level 10: Tireless", p. 59 | unhappy (boundary) |
| FR-075 | Tireless recharges on `RechargeTrigger` long rest (rests-and-hit-dice.md; shared-types-and-ownership.md FR-009): a Long Rest MUST reset expended uses to zero; a Short Rest MUST leave them unchanged. | SRD 5.2.1, "Level 10: Tireless", p. 59 | unhappy (rule interaction) |
| FR-076 | The lowercase pure function `applyRangerRest` (the `ApplyRangerRest` of rests-and-hit-dice.md means this function) for a Short Rest at Ranger level 10 or above with Exhaustion level 1 or more MUST return an `ExhaustionReductionGranted` event of exactly 1 in the `events` of its `RangerRestOutcome` (no separate reduction field; applying it is rests-and-hit-dice.md's); with Exhaustion level 0, or below level 10, none. A negative Exhaustion level MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error). | SRD 5.2.1, "Level 10: Tireless" (Decrease Exhaustion), p. 59 | unhappy (boundary) |
| FR-077 | A Tireless roll outside 1 to 8 MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error) from the domain service. | doc/rules/testing.md §4; SRD 5.2.1, "Level 10: Tireless" (1d8), p. 59 | unhappy (invalid input) |
| FR-078 | From Ranger level 13, damage MUST NOT break Concentration on *Hunter's Mark*: the pure rule `damageCanBreakConcentration(spell, level)` returns false for `hunters_mark`, and true for any other spell and at any level below 13. It takes only ids and the level. It is read by the host, not imported by `combat` (no class-specific coupling): the host passes the answer to combat-action-resolution.md as the closed Concentration-damage fact of `ConcentrationInput`, which skips the Constitution save when damage cannot break it, before `ConcentrationSaveDc.forDamage` (casting-and-concentration.md FR-079; shared-types-and-ownership.md FR-011); casting-and-concentration.md does not consult it (section 9). Other reasons Concentration ends are unaffected. | SRD 5.2.1, "Level 13: Relentless Hunter", p. 59 | unhappy (rule interaction) |
| FR-079 | `useNaturesVeil` MUST return `NaturesVeilRejected` with `FeatureNotAvailable` below Ranger level 14. At level 14 or above with uses remaining it MUST return `NaturesVeilAccepted` with an Invisible grant (`ConditionGrant(ConditionKind.invisible, EndOfNextTurn)`) lasting until the end of the character's next turn, action cost Bonus Action, and emit `NaturesVeilUsed`. | SRD 5.2.1, "Level 14: Nature's Veil", p. 59 | happy |
| FR-080 | Nature's Veil uses MUST be the greater of 1 and the Wisdom modifier, remaining never below zero; none remaining MUST be rejected with `NoUsesRemaining`; a Long Rest (`RechargeTrigger` long rest, as FR-075) MUST reset expended uses and a Short Rest MUST NOT. | SRD 5.2.1, "Level 14: Nature's Veil", p. 59 | unhappy (boundary) |
| FR-081 | From Ranger level 17, an attack roll against the creature currently marked by the character's *Hunter's Mark* MUST carry a `RollCircumstance` (modifiers-and-advantage.md) of Advantage from Precise Hunter; against an unmarked creature, or below level 17, it MUST NOT. | SRD 5.2.1, "Level 17: Precise Hunter", p. 59 | happy |
| FR-082 | From Ranger level 18 the character MUST have a `SenseGrant(SpecialSense.blindsight, 30)` (30 feet) from Feral Senses; below level 18 none from this feature. | SRD 5.2.1, "Level 18: Feral Senses", p. 59 | happy |
| FR-083 | The damage die of *Hunter's Mark* MUST be d10 at Ranger level 20 and the spell's own die (d6) below level 20 (T5). | SRD 5.2.1, "Level 20: Foe Slayer", p. 59 | happy (T5) |

#### T4: Tireless Temporary Hit Points and uses (FR-072, FR-073, one test per row)

Source: SRD 5.2.1, "Level 10: Tireless", p. 59. Temporary Hit Points = roll + max(1, Wisdom modifier); uses = max(1, Wisdom modifier).

| Label | Roll (d8) | Wisdom score | Modifier | Temporary Hit Points | Uses | Path |
|---|---|---|---|---|---|---|
| TL-01 | 5 | 16 | +3 | 8 | 3 | happy |
| TL-02 | 8 | 20 | +5 | 13 | 5 | unhappy (boundary: maximum roll) |
| TL-03 | 1 | 10 | +0 | 2 | 1 | unhappy (boundary: zero modifier, minimum roll) |
| TL-04 | 1 | 8 | -1 | 2 | 1 | unhappy (boundary: negative modifier) |
| TL-05 | 8 | 1 | -5 | 9 | 1 | unhappy (boundary: lowest score) |
| TL-06 | 4 | 12 | +1 | 5 | 1 | unhappy (boundary: modifier equals the minimum) |
| TL-07 | 4 | 14 | +2 | 6 | 2 | happy |
| TL-08 | 3 | 30 | +10 | 13 | 10 | unhappy (boundary: highest score) |

#### T5: Foe Slayer cases (FR-083, one test per row)

Source: SRD 5.2.1, "Level 20: Foe Slayer", p. 59. Each row is Ranger level, spell die in, expected die: FS-01 20, d6, d10 (happy); FS-02 19, d6, d6 (unhappy, one below); FS-03 1, d6, d6 (unhappy, minimum level); FS-04 21, d6, panic (unhappy, invalid input).

### 4.9 Hunter subclass

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-084 | Hunter's Lore (Hunter, level 3+): while a creature is marked by the character's *Hunter's Mark*, the character MUST learn whether it has any Immunities, Resistances or Vulnerabilities and what they are. The caller supplies the creature's three `DefenseEntry` lists (content-monsters.md FR-024); the result is `NoneApply` (`HuntersLoreNoneApply`, all three empty) or `Reveals` (`HuntersLoreReveals`) carrying the three lists unchanged, so qualifiers are kept and none is flattened to a bare `DamageType` or `ConditionKind` set. | SRD 5.2.1, "Level 3: Hunter's Lore", p. 61 | happy |
| FR-085 | Hunter's Lore MUST reveal nothing for a creature not marked (`HuntersLoreNotMarked`), for a character who has not chosen Hunter (`HuntersLoreNotHunter`, `RangerRejection.notHunter`), and below Ranger level 3 (`HuntersLoreFeatureNotAvailable`); never an empty reveal. | SRD 5.2.1, "Level 3: Hunter's Lore", p. 61 | unhappy (state error) |
| FR-086 | Hunter's Prey (Hunter, level 3) MUST accept exactly one of the content option ids `colossus_slayer` and `horde_breaker` (content-classes.md table OP); any other id MUST be rejected with `UnknownOption`, and a second initial choice (once an option is held) with `AlreadyChosen`; replacement is FR-087. | SRD 5.2.1, "Level 3: Hunter's Prey", p. 61 | happy |
| FR-087 | The Hunter's Prey and Defensive Tactics option MAY be replaced with the other option only when the character finishes a Short Rest or a Long Rest; at any other moment the replacement MUST be rejected with `WrongMoment`. Replacing with the option already held MUST be rejected with `SameOption`. The initial choice (FR-086) needs no rest. | SRD 5.2.1, "Level 3: Hunter's Prey"; "Level 7: Defensive Tactics", p. 61 | unhappy (state error) |
| FR-088 | Colossus Slayer (Ranger level 3+, Hunter chosen, `colossus_slayer` held): when the character hits a creature with a weapon, that creature is missing any of its Hit Points, and the extra damage has not yet been dealt this turn, a rider of 1d8 extra damage MUST be returned. The SRD states no damage type, so the rider carries `UnstatedDamageType`. | SRD 5.2.1, "Level 3: Hunter's Prey" (Colossus Slayer), p. 61 | happy (T8) |
| FR-089 | Colossus Slayer MUST return no rider when the creature is at its Hit Point maximum, the damage was already dealt this turn, the hit was not made with a weapon, or the character holds Horde Breaker instead (T8). | SRD 5.2.1, "Level 3: Hunter's Prey" (Colossus Slayer), p. 61 | unhappy (rule interaction) |
| FR-090 | Horde Breaker (Ranger level 3+, Hunter chosen, `horde_breaker` held, attack made with a weapon): once per turn when the character attacks with a weapon, it MUST be allowed one further attack with the same weapon against a different creature within 5 feet of the original target, within the weapon's range, not yet attacked this turn; each failing precondition or condition, or the option already used this turn, MUST yield `Ineligible` carrying that reason (T9). | SRD 5.2.1, "Level 3: Hunter's Prey" (Horde Breaker), p. 61 | unhappy (boundary, T9) |
| FR-091 | Defensive Tactics (Hunter, level 7) MUST accept exactly one of the content option ids `escape_the_horde` and `multiattack_defense` (table OP), be rejected below Ranger level 7 with `FeatureNotAvailable`, and reject any other option with `UnknownOption`. | SRD 5.2.1, "Level 7: Defensive Tactics", p. 61 | happy |
| FR-092 | Escape the Horde (Ranger level 7+, Hunter chosen, `escape_the_horde` held): an Opportunity Attack against the character MUST carry a `RollCircumstance` of Disadvantage; any other attack roll against the character MUST NOT get one from this option. | SRD 5.2.1, "Level 7: Defensive Tactics" (Escape the Horde), p. 61 | happy |
| FR-093 | Multiattack Defense (Ranger level 7+, Hunter chosen, `multiattack_defense` held): after a creature hits the character with an attack roll, that creature MUST have a `RollCircumstance` of Disadvantage on every other attack roll against the character for the rest of that turn, and no other creature MUST; the record MUST be empty at the start of the next turn and the attack roll that hit MUST NOT itself get Disadvantage. | SRD 5.2.1, "Level 7: Defensive Tactics" (Multiattack Defense), p. 61 | unhappy (rule interaction) |
| FR-094 | Superior Hunter's Prey (Hunter chosen, Ranger level 11+; below 11 `Ineligible(featureNotAvailable)`): once per turn, when the character deals damage to a creature marked by its *Hunter's Mark*, a rider of the spell's extra damage (the die of FR-083) MUST be allowed against a different creature the character can see within 30 feet of the first; each failing condition, or the rider already used this turn, MUST yield `Ineligible` carrying that reason (T9). | SRD 5.2.1, "Level 11: Superior Hunter's Prey", p. 61 | unhappy (boundary, T9) |
| FR-095 | Superior Hunter's Defense (Hunter chosen, Ranger level 15+): when the character takes damage and has a Reaction available, `useSuperiorHuntersDefense` MUST return `SuperiorHuntersDefenseAccepted` with a Resistance grant for that `DamageType`, applying to the triggering damage and any other damage of the same type, expiring (`EndOfCurrentTurn`) at the end of the current turn, with action cost Reaction. Without a Reaction, below level 15, or without Hunter it MUST be rejected with a typed result. | SRD 5.2.1, "Level 15: Superior Hunter's Defense", p. 61 | happy |

#### T8: Colossus Slayer cases (FR-088, FR-089, one test per row)

Source: SRD 5.2.1, "Level 3: Hunter's Prey" (Colossus Slayer), p. 61.

| Label | Option | Weapon hit | Target Hit Points | Dealt this turn | Expected | Path |
|---|---|---|---|---|---|---|
| CS-01 | Colossus Slayer | yes | 19 of 20 | no | rider 1d8 | happy |
| CS-02 | Colossus Slayer | yes | 20 of 20 | no | none | unhappy (boundary: not missing any) |
| CS-03 | Colossus Slayer | yes | 1 of 20 | no | rider 1d8 | unhappy (boundary: one remaining) |
| CS-04 | Colossus Slayer | yes | 10 of 20 | yes | none | unhappy (state error) |
| CS-05 | Colossus Slayer | no (spell attack) | 10 of 20 | no | none | unhappy (rule interaction) |
| CS-06 | Horde Breaker | yes | 10 of 20 | no | none | unhappy (rule interaction) |
| CS-07 | none chosen | yes | 10 of 20 | no | none | unhappy (missing choice) |
| CS-08 | Colossus Slayer, level 2 | yes | 10 of 20 | no | none | unhappy (boundary: below level 3) |

#### T9: Horde Breaker and Superior Hunter's Prey cases (FR-090, FR-094, one test per row)

Source: SRD 5.2.1, "Level 3: Hunter's Prey" (Horde Breaker); "Level 11: Superior
Hunter's Prey", p. 61. Distance in feet from the first target. "Range/seen" is
weapon range (HB) or visibility (SP); "Attacked" is already attacked this turn.

| Label | Feature | Different creature | Distance | Range/seen | Attacked | Used this turn | Expected | Path |
|---|---|---|---|---|---|---|---|---|
| HB-01 | Horde Breaker | yes | 5 | in range | no | no | eligible | happy |
| HB-02 | Horde Breaker | yes | 6 | in range | no | no | `Ineligible(tooFar)` | unhappy (boundary: one over) |
| HB-03 | Horde Breaker | yes | 0 | in range | no | no | eligible | unhappy (boundary: zero) |
| HB-04 | Horde Breaker | no (same) | 0 | in range | no | no | `Ineligible(sameCreature)` | unhappy (rule interaction) |
| HB-05 | Horde Breaker | yes | 5 | out of range | no | no | `Ineligible(outOfRange)` | unhappy (boundary) |
| HB-06 | Horde Breaker | yes | 5 | in range | yes | no | `Ineligible(alreadyAttacked)` | unhappy (rule interaction) |
| HB-07 | Horde Breaker | yes | 5 | in range | no | yes | `Ineligible(usedThisTurn)` | unhappy (state error) |
| HB-08 | Horde Breaker, spell attack | yes | 5 | in range | no | no | `Ineligible(notWeaponAttack)` | unhappy (rule interaction) |
| HB-09 | Colossus Slayer held | yes | 5 | in range | no | no | `Ineligible(optionNotHeld)` | unhappy (missing choice) |
| HB-10 | Horde Breaker, level 2 | yes | 5 | in range | no | no | `Ineligible(featureNotAvailable)` | unhappy (boundary: below level 3) |
| SP-01 | Superior Hunter's Prey | yes | 30 | visible | n/a | no | eligible | happy |
| SP-02 | Superior Hunter's Prey | yes | 31 | visible | n/a | no | `Ineligible(tooFar)` | unhappy (boundary: one over) |
| SP-03 | Superior Hunter's Prey | yes | 10 | not visible | n/a | no | `Ineligible(notVisible)` | unhappy (invalid input) |
| SP-04 | Superior Hunter's Prey | no (same) | 0 | visible | n/a | no | `Ineligible(sameCreature)` | unhappy (rule interaction) |
| SP-05 | Superior Hunter's Prey | yes | 30 | visible | n/a | yes | `Ineligible(usedThisTurn)` | unhappy (state error) |
| SP-06 | SHP, first target not marked | yes | 10 | visible | n/a | no | `Ineligible(notMarked)` | unhappy (rule interaction) |
| SP-07 | SHP, level 20 | yes | 10 | visible | n/a | no | eligible, die d10 | unhappy (Foe Slayer) |
| SP-08 | SHP, level 10 | yes | 10 | visible | n/a | no | `Ineligible(featureNotAvailable)` | unhappy (boundary: one below 11) |
| SP-09 | SHP, level 11 | yes | 10 | visible | n/a | no | eligible, die d6 | unhappy (boundary: minimum level) |

#### T10: Hunter features held by Ranger level (expected values of content-classes.md FR-075)

Source: SRD 5.2.1, "Ranger Subclass: Hunter", p. 61. None held (unhappy): level 3 with no subclass chosen; level 2 with Hunter requested (a request below level 3 is level-advancement-and-multiclassing.md `SubclassRejected(belowSubclassLevel)`, FR-044 there). Hunter at levels 3 to 6: Hunter's Lore, Hunter's Prey; 7 to 10: and Defensive Tactics; 11 to 14: and Superior Hunter's Prey; 15 to 20: all five.

### 4.10 Resolving the Ranger's features and choices

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-096 | `ResolveRangerFeatures` MUST return the effects that apply at the Ranger level and choices: attacks per Attack action, Speed profile, senses, always-prepared spells, granted cantrips, Hunter's Mark die, Concentration rule, feature uses maximum, and the subclass features held. Effects of a choice not yet made MUST be absent. | SRD 5.2.1, "Ranger Class Features", pp. 57-59 | happy |
| FR-097 | The same result MUST list the pending choices owed at that level and not yet made (T11): Weapon Mastery (1); Deft Explorer and Fighting Style (2); Ranger Subclass (3); Hunter's Prey (3, Hunter); Ability Score Improvement (each of 4, 8, 12, 16); Defensive Tactics (7, Hunter); Expertise (9); Epic Boon (19). Class skills and Starting Equipment are character-creation.md FR-015 to FR-018 and `EquipmentOptionLetter`; the multiclass single skill is level-advancement-and-multiclassing.md's. A multiclass entry still owes Weapon Mastery. The Ranger Subclass choice is cleared by the `SubclassChosen` event of level-advancement-and-multiclassing.md. | SRD 5.2.1, "Core Ranger Traits", p. 57; "Ranger Features", p. 58; "Ranger Subclass: Hunter", p. 61 | unhappy (choices versus level) |
| FR-098 | Consumer-side check: Ranger functions and use cases take the Ranger level as the per-class entry of `ClassLevels` (1 to 20; derived-stats.md, shared-types-and-ownership.md FR-008); this spec defines no level type, and a level outside 1 to 20 is rejected before it reaches a Ranger function. | SRD 5.2.1, "Ranger Features", p. 58 | unhappy (boundary) |

#### T11: pending choices by Ranger level (FR-097, one test per row)

Source: SRD 5.2.1, "Ranger Features", p. 58; Levels 1 to 19, pp. 57-59 and 61. Each row assumes no choice made yet; "the above" is the previous row. Level, subclass, pending choices: PC-01 1, n/a, Weapon Mastery (happy); PC-02 2, n/a, the above and Deft Explorer, Fighting Style (happy); PC-03 3, none chosen, the above and Ranger Subclass (happy); PC-04 3, Hunter, the above and Hunter's Prey (happy); PC-05 4, Hunter, the above and one Ability Score Improvement (unhappy, boundary); PC-06 7, Hunter, the above and Defensive Tactics (unhappy, boundary); PC-07 9, Hunter, the above and Expertise (unhappy, boundary); PC-08 19, Hunter, the above, four Ability Score Improvement grants and Epic Boon (unhappy, boundary); PC-09 20, Hunter with all choices made, none (unhappy, empty and minimal); PC-10 1 as a multiclass entry, n/a, Weapon Mastery only (the one skill is level-advancement-and-multiclassing.md's; no Starting Equipment) (unhappy, multiclass entry).

### 4.11 Cross-cutting behaviour

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-099 | Every Ranger state transition MUST return a new state and the `RangerEvent`s it produced, MUST NOT mutate the input state, and every collection exposed by the state and results MUST be a defensive copy. | doc/rules/architecture.md §3 | unhappy (immutability) |
| FR-100 | With a fixed `dice.Roller`, `UseTireless` MUST return the same result every run, including a natural 1 and a natural 8. | doc/rules/testing.md §4; SRD 5.2.1, "Level 10: Tireless", p. 59 | unhappy (determinism) |
| FR-101 | When the `ranger` class definition (or the chosen subclass) is absent from the `ContentSource`, `ChooseRangerOption` and `ResolveRangerFeatures` MUST return their own `ChooseRangerOptionContentMissing` or `ResolveRangerFeaturesContentMissing` variant holding a `MissingContent(kind, id)` (shared-types-and-ownership.md FR-017, FR-018) naming the id, not panic. `UseTireless` and the pure functions load no content and have no such variant. | doc/rules/architecture.md §4, §6 | unhappy (missing content) |
| FR-102 | When the `ContentSource` throws its typed `*srderr.PortError`, `ChooseRangerOption` and `ResolveRangerFeatures` MUST propagate it unchanged and return no partial result. | doc/rules/architecture.md §4 | unhappy (collaborator failure) |
| FR-103 | `ChooseRangerOption` and `ResolveRangerFeatures`, applied to a character with no Ranger levels, MUST return their own `ChooseRangerOptionNotARanger` or `ResolveRangerFeaturesNotARanger` variant, not panic. `UseTireless` and the pure functions take a Ranger level and state, not a character, and have no such variant. | SRD 5.2.1, "Ranger Class Features", p. 57 | unhappy (state error) |
| FR-104 | Each of the three use cases MUST produce identical results with the no-op `telemetry.Telemetry` and a recording one. The pure functions take no `telemetry.Telemetry`. | doc/rules/observability.md §9 | unhappy (determinism) |
| FR-105 | Each of the three use cases MUST emit one span and one `srd5e.character.feature_used` increment per call with the attributes of section 7 (`srd5e.class`, `srd5e.feature`, `srd5e.outcome` with the shared-types-and-ownership.md FR-041 values), record a content failure result exactly once (span error status and one `feature_used` increment with `srd5e.outcome=failure`, no `error.type`) and a returned port failure exactly once (span error status, one `feature_used` increment with `srd5e.outcome=failure`, no `error.type` and no rejection, and one increment of the shared `srd5e.character.feature_used.errors` with `error.type`; character-aggregate.md FR-118, shared-types-and-ownership.md FR-063), and MUST NOT log it. | doc/rules/observability.md §6, §7, §9; shared-types-and-ownership.md FR-041, FR-042 | unhappy (telemetry on failure) |
| FR-106 | Nothing in the Ranger results, events or state MUST contain a pre-formatted display string, name a consuming project, or rely on a rule from outside SRD 5.2.1. The decoded definition (content-classes.md) MAY carry content names and feature text, which are content data. | AGENTS.md, "Hard constraints"; doc/rules/architecture.md §3, §10 | unhappy (invalid input) |
| FR-107 | When the `dice.Roller` throws its typed `*srderr.PortError`, `UseTireless` MUST propagate it unchanged, return no partial result and leave the passed `RangerResources` unchanged (immutable; no use spent). | doc/rules/architecture.md §4; doc/rules/testing.md §4 | unhappy (collaborator failure) |
| FR-108 | A Ranger choice is either an initial choice (made when the feature is gained, no moment) or a replacement of a choice already made (with the moment of FR-040, FR-053 or FR-087). The initial choice MUST be accepted without a rest, a replacement without the right moment MUST be rejected, and a replacement when no choice exists MUST be rejected with `NothingToReplace`. | SRD 5.2.1, "Level 1: Weapon Mastery"; "Level 2: Fighting Style"; "Level 3: Hunter's Prey"; "Level 7: Defensive Tactics", pp. 59, 61 | unhappy (state error) |
| FR-109 | Ranger creation choices travel in `CreateCharacter` as a `RangerCreationChoices` value; the aggregate stores no Ranger-specific field. It holds only the initial Weapon Mastery pair (FR-037) and implements `ClassCreationChoices` (character-aggregate.md FR-094; stored unchanged, character-creation.md FR-102): a pure method taking a `FeatureStates` and returning it seeded with `RangerChoices` (the pair) and the initial `RangerResources` under the keys of FR-110, and a pure method returning the `ContentKey`s the pair names. The interface carries no class id. Class skills and the Starting Equipment option are `CreateCharacterCommand` fields validated by character-creation.md FR-015 to FR-018. Its the `New` constructor (returns an error) delegates the pair to `NewWeaponMasteryChoice` (FR-037) and returns `(RangerCreationChoices, error)`. | shared-types-and-ownership.md FR-024; character-aggregate.md FR-094; SRD 5.2.1, "Level 1: Weapon Mastery", pp. 58-59 | happy |
| FR-110 | Ranger state (`RangerChoices`, `RangerResources`) MUST be held as `FeatureState` implementations in the aggregate's `FeatureStates` under two typed keys defined here: `FeatureStateKey[RangerChoices] named `ranger_choices`` and `FeatureStateKey[RangerResources] named `ranger_resources`` (names match `[a-z][a-z0-9_]*`, character-aggregate.md FR-093). The aggregate interprets neither and `FeatureStates.read` returns the zero value and false for a never-set key (FR-092 there). The default is Ranger-side: `RangerChoicesOf(states)` returns the stored value, or `RangerChoices` initial state when the key is unset, and `RangerResourcesOf(states)` returns the stored value, or `RangerResources` initial state when the key is unset (zero expended). | shared-types-and-ownership.md FR-022; character-aggregate.md FR-092, FR-093 | unhappy (empty and minimal) |
| FR-111 | Every Ranger feature other than the three use cases of section 6 MUST be a pure domain function from (state, inputs) to (new state, events), with no `telemetry.Telemetry`, port or content access; only a feature needing dice (`UseTireless`) or content (`ChooseRangerOption`, `ResolveRangerFeatures`) has a use case. | shared-types-and-ownership.md FR-023 | happy |
| FR-112 | No export named `Applied`, `Rejected`, `ContentMissing`, `RestApplied`, `SubclassChosen`, `SubclassRejected`, `WeaponMasteryChoice`, `FightingStyleChoice` or `ClassDataInvalid` is defined by this spec; each is owned as in shared-types-and-ownership.md FR-017, FR-020, FR-021. | shared-types-and-ownership.md FR-017, FR-020, FR-021 | unhappy (name collision) |
| FR-113 | Ranger events MUST be the variants of one closed `RangerEvent` family declared in one library and implementing the `DomainEvent` marker (the per-class family accepted for shared-types-and-ownership.md FR-019, as class-cleric.md FR-136 does). They carry no `CharacterId`; the host or aggregate use case attributes each to a character. They are not members of `CharacterEvent` or `CharacterRecordEvent` (character-aggregate.md FR-085). | shared-types-and-ownership.md FR-019; character-aggregate.md FR-085 | happy |

### 4.12 Withdrawn ids

Moved to the named spec; ids are never reused.

| Id | Moved |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (class definition data) |
| FR-002 | Withdrawn: moved to content-classes.md (class definition data) |
| FR-003 | Withdrawn: moved to content-classes.md (class definition data) |
| FR-004 | Withdrawn: moved to character-creation.md (FR-015) |
| FR-005 | Withdrawn: moved to character-creation.md (FR-016, FR-018) |
| FR-006 | Withdrawn: moved to character-creation.md (FR-017) |
| FR-007 | Withdrawn: moved to content-classes.md (class definition data) |
| FR-008 | Withdrawn: moved to content-classes.md (class definition data) |
| FR-009 | Withdrawn: moved to content-classes.md (class definition data) |
| FR-010 | Withdrawn: moved to content-classes.md (FR-024) |
| FR-011 | Withdrawn: moved to content-classes.md (FR-024) |
| FR-012 | Withdrawn: moved to character-creation.md (`EquipmentOptionLetter` field) and content-classes.md (FR-024) |
| FR-013 | Withdrawn: moved to content-classes.md (level 1 trait data) and character-creation.md |
| FR-014 | Withdrawn: moved to content-classes.md (multiclass trait data) and level-advancement-and-multiclassing.md |
| FR-015 | Withdrawn: moved to content-classes.md (multiclass trait data) and level-advancement-and-multiclassing.md |
| FR-016 | Withdrawn: moved to level-advancement-and-multiclassing.md (multiclass single skill) |
| FR-017 | Withdrawn: moved to content-classes.md (decoder table checks) |
| FR-018 | Withdrawn: moved to content-classes.md (bulk class data; T1 is its transcription) |
| FR-019 | Withdrawn: moved to content-classes.md (decoder table checks) |
| FR-020 | Withdrawn: moved to content-classes.md (no Cantrips column; zero known is spellcasting.md FR-018) |
| FR-021 | Withdrawn: moved to content-classes.md (FR-047) |
| FR-022 | Withdrawn: moved to content-classes.md (FR-048, FR-075) |
| FR-024 | Withdrawn: moved to content-classes.md (`FeatureEntry` data) |
| FR-025 | Withdrawn: moved to content-classes.md (`SubclassDefinition` data, Hunter) |
| FR-026 | Withdrawn: moved to content-classes.md and content-spells.md (spell list; T7 is its transcription) |
| FR-027 | Withdrawn: moved to content-spells.md and content-classes.md (list versus class lists) |
| FR-028 | Withdrawn: moved to content-spells.md and content-classes.md (content check) |
| FR-029 | Withdrawn: moved to content-classes.md (Spellcasting feature data) |
| FR-038 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-015, FR-016) |
| FR-039 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-019, FR-020) |
| FR-050 | Withdrawn: moved to feats-behaviour.md (FR-002, FR-005) |
| FR-056 | Withdrawn: moved to level-advancement-and-multiclassing.md (`SubclassChosen`) |
| FR-057 | Withdrawn: moved to level-advancement-and-multiclassing.md (`SubclassRejected`) |
| FR-058 | Withdrawn: moved to level-advancement-and-multiclassing.md (`SubclassRejected`) |
| FR-059 | Withdrawn: moved to content-classes.md (FR-075; T10 is its expected values) |
| FR-060 | Withdrawn: moved to content-classes.md (FR-049; counting is FR-064, T6) |
| FR-061 | Withdrawn: moved to feats-behaviour.md (applying a feat, qualification) |
| FR-063 | Withdrawn: moved to feats-behaviour.md (duplicate and Repeatable rules) |

## 5. Model

Domain (`character` package, pure domain files; pure, synchronous, silent):

- **The Ranger definition is content** (`ClassDefinition`, `SubclassDefinition`,
  `ProgressionRow`, `FeatureEntry`; content-classes.md, shared-types-and-ownership.md
  FR-025, FR-031). `character` imports only the `content` package API. There is no
  `RangerProgression`: rows, features, feat grants and subclass features come
  from content-classes.md FR-047 to FR-049, FR-075; only `favoredEnemyUses(level)`
  (FR-023, FR-032) stays, a one-line read of the row.
- **Names owned elsewhere**: `Skill`, `Language`, `DamageType`, `ConditionKind`,
  `SpellId`, `ClassId`, `SubclassId`, `FeatureId`, `RechargeTrigger`,
  `MissingContent` (content, shared-types-and-ownership.md FR-005, FR-018);
  `DefenseEntry` (content-monsters.md FR-024); `SpecialSense`
  (content-schema-and-decoder.md FR-136); `RollCircumstance`;
  `GrantedCantrip`, `PreparedSpells.alwaysPrepared` (spellcasting.md FR-097);
  `AlwaysPreparedSpells`, `rowAt`, `featGrantsAt`, `featuresAtOrBelow`
  (content-classes.md); `ExtraAttackCombination`, `SubclassChosen`,
  `SubclassRejected` (level-advancement-and-multiclassing.md);
  `ClassCreationChoices`, `FeatureStates`, `FeatureStateKey`
  (character-aggregate.md); `ClassLevels` (derived-stats.md); `RestKind`
  (rests-and-hit-dice.md); `WeaponMasteryChoice`; `FightingStyleChoice`.
- **`RangerCreationChoices`** (FR-109): holds the initial Weapon Mastery pair;
  `NewRangerCreationChoices` returns `(RangerCreationChoices, error)`.
- **`RangerChoices`**: `FeatureState` (FR-110) holding the Weapon Mastery pair,
  `DeftExplorerChoice` (`Skill`, two `Language`s), the Fighting Style (closed
  `RangerFightingStyle`: `RangerFightingStyleFeat(featId)` | `DruidicWarrior(cantrips)`),
  Hunter's Prey and Defensive Tactics as content option ids, Expertise skills
  and feat grants made. The subclass id lives in the `SubclassChosen` state.
  the `New` constructor (returns an error) factories return closed results for untrusted input (FR-045,
  FR-048, FR-051), delegating mastery checks to `WeaponMasteryChoice` (FR-037) and feat checks to the composed `FeatTakingRules`/`FeatQualification` (FR-049).
- **`RangerResources`**: `FeatureState` (`.of(states)`, `.initial`) with expended
  counts for Favored Enemy, Tireless and Nature's Veil (never negative);
  remaining is maximum minus expended, clamped at zero (FR-035, FR-074).
- **Pure functions** (FR-111): `useFavoredEnemy`, `useNaturesVeil`,
  `useSuperiorHuntersDefense`, `applyRangerRest`, `resolveRangerAttackFeatures`
  (section 6), and the services `RangerSpeed` (FR-066 to FR-070),
  `TirelessTemporaryHitPoints` (FR-072, FR-073, FR-077), `HuntersMarkDie`
  (FR-083), `RangerConcentrationRule` (FR-078), `HuntersLore` (FR-084, FR-085),
  `colossusSlayerRider` (FR-088, FR-089), `hordeBreakerEligibility` (FR-090),
  `SuperiorHuntersPrey` (FR-094), `MultiattackDefenseRecord` (FR-093).
- **Handoff values**: `RangerEffects` (FR-096), `AttackRider` (dice and closed `RiderDamageType`: `UnstatedDamageType` | `StatedDamageType(DamageType)`), `ResistanceGrant(DamageType, GrantExpiry)`, `ConditionGrant(ConditionKind, GrantExpiry)`, the closed Ranger-side `GrantExpiry` (`EndOfCurrentTurn` | `EndOfNextTurn`; FR-079, FR-095), `SenseGrant(SpecialSense, rangeFeet)`, `RangerChoiceRequest` (closed: the initial and replacement variants of section 6), the enum `RangerReplacementMoment` (`finishedLongRest`, `finishedShortRest`, `gainedRangerLevel`), `SpeedProfile`, `SpeedInput(base, canIncrease)` (FR-070), `RangerActionCost`, and `AttackerKey` (opaque defined type for Multiattack Defense, FR-093; `combat` maps its `CombatantId` to it, shared-types-and-ownership.md FR-012, FR-013). Granted cantrips are `GrantedCantrip` (FR-052); the always-prepared spell is the content fact (FR-031).
- **Closed results** (shared-types-and-ownership.md FR-017): `ChooseRangerOptionResult`
  = `ChooseRangerOptionAccepted(choices, events)` | `...Rejected(reason)` |
  `...ContentMissing(missing)` | `...NotARanger`; `ResolveRangerFeaturesResult` =
  `ResolveRangerFeaturesResolved(effects)` | `...ContentMissing(missing)` |
  `...NotARanger`; `UseTirelessResult` = `UseTirelessAccepted(temporaryHitPoints,
  actionCost, resources, events)` | `UseTirelessRejected(reason)`. Pure-function
  results: `FavoredEnemyUseResult`, `NaturesVeilResult`,
  `SuperiorHuntersDefenseResult` (each `...Accepted` | `...Rejected`),
  `HuntersLoreResult` (`HuntersLoreNoneApply` | `HuntersLoreReveals(immunities,
  resistances, vulnerabilities)` | `HuntersLoreNotMarked` | `HuntersLoreNotHunter`
  | `HuntersLoreFeatureNotAvailable`), `RangerRiderEligibility`
  (`RangerRiderEligible` | `RangerRiderIneligible(reason)`). Each `missing` is a
  `MissingContent(kind, id)`.
- **Events** (FR-113): `FavoredEnemyCastSpent`, `TirelessUsed`, `NaturesVeilUsed`,
  `ExhaustionReductionGranted`, `RangerChoiceMade`, `RangerChoiceReplaced`. The
  rest event is `RestApplied` of rests-and-hit-dice.md.

Invariants and enforcing constructor: set cardinalities (`RangerChoices`, FR-045,
FR-048); uses never negative (`RangerResources`, FR-035); level 1 to 20
(`ClassLevels`, FR-098); Tireless roll 1 to 8 (FR-077); Exhaustion level and base
Speed not negative (FR-070, FR-076); at most one Fighting Style (FR-054).

Failures: untrusted host input is a closed rejection from the `New` constructor (returns an error); programmer
errors panics with an error value; illegal-in-this-state requests are closed results
carrying a `RangerRejection`; missing content is the `...ContentMissing` variant
(FR-101); port failure is `*srderr.PortError` (`doc/rules/architecture.md` §6).

## 6. Interface

Per shared-types-and-ownership.md FR-023 a feature is a pure domain function; only
a feature needing content or dice gets a use case in
the `character` package (use-case files), collaborators injected as ports
(`doc/rules/architecture.md` §4). `ChooseRangerOption` and `ResolveRangerFeatures`
load content through the context-aware `ContentSource`, take the
character and have the `...ContentMissing` and `...NotARanger` variants.
`UseTireless` needs only the synchronous `dice.Roller`: it is synchronous, takes
no character and has neither variant.

| Use case (collaborators) | Command carries | Result variants | Requirements |
|---|---|---|---|
| `ChooseRangerOption` (`ContentSource`, `telemetry.Telemetry`) | character, `RangerChoices`, one `RangerChoiceRequest`: **initial** variants with no moment (Deft Explorer, Expertise, Fighting Style, Hunter's Prey, Defensive Tactics, feat grant) and **replacement** variants each with its moment (Weapon Mastery after `finishedLongRest`; Druidic Warrior cantrip on `gainedRangerLevel`; Hunter's Prey and Defensive Tactics on `finishedShortRest` or `finishedLongRest`). The initial mastery pair travels in `RangerCreationChoices`; class skills and equipment are character-creation.md's; the subclass is level-advancement-and-multiclassing.md's | `ChooseRangerOptionAccepted(choices, events)` / `...Rejected(reason)` / `...ContentMissing(missing)` / `...NotARanger` | FR-037, FR-040, FR-041, FR-042 to FR-049, FR-051 to FR-055, FR-062, FR-064, FR-086, FR-087, FR-091, FR-108 |
| `ResolveRangerFeatures` (`ContentSource`, `telemetry.Telemetry`) | character, `RangerChoices`, `SpeedInput`, Heavy armor worn, Wisdom modifier | `ResolveRangerFeaturesResolved(effects)` / `...ContentMissing(missing)` / `...NotARanger` | FR-032, FR-065 to FR-070, FR-078, FR-082, FR-083, FR-096, FR-097 |
| `UseTireless` (`dice.Roller`, `telemetry.Telemetry`) | `RangerResources`, Ranger level, Wisdom modifier | `UseTirelessAccepted(temporaryHitPoints, actionCost, resources, events)` / `UseTirelessRejected(reason)`; a `dice.Roller` failure propagates (FR-107) | FR-071 to FR-075, FR-077, FR-100, FR-107 |

**Pure functions** (no use case, no `telemetry.Telemetry`, FR-111):

| Function | Inputs | Returns | Requirements |
|---|---|---|---|
| `UseFavoredEnemy` | `RangerResources`, level (maximum is `favoredEnemyUses(level)`) | `FavoredEnemyUseAccepted(resources, events)` / `FavoredEnemyUseRejected(NoUsesRemaining)` | FR-033 to FR-035 |
| `UseNaturesVeil` | `RangerResources`, level, Wisdom modifier | `NaturesVeilAccepted(invisibleGrant, actionCost, resources, events)` / `NaturesVeilRejected(reason)` | FR-079, FR-080 |
| `UseSuperiorHuntersDefense` | level, Hunter chosen, `DamageType`, Reaction available | `SuperiorHuntersDefenseAccepted(grant, actionCost: Reaction)` / `...Rejected(reason)` | FR-095 |
| `ResolveRangerAttackFeatures` | level, `RangerChoices`, the spell's own die, an attack context (target marked, weapon used, target Hit Points, per-turn usage record, second-target facts, Multiattack Defense record, Opportunity Attack against the character) | `RangerAttackFeatures(riders, advantage, disadvantage, hordeBreaker, hunterLore, record)` | FR-081, FR-084 to FR-094 |
| `ApplyRangerRest` | `RangerResources`, `RestKind`, level, Exhaustion level | `RangerRestOutcome(resources, events)` | FR-036, FR-075, FR-076, FR-080 |

- Ports: `ContentSource` (context-aware; `ContentFound` | `ContentNotFound` of
  content-source-port.md, `*srderr.PortError` for failure), `dice.Roller` (sync),
  `telemetry.Telemetry` (sync, `const` no-op default). No new port.
- Spell facts, feat qualification, weapon proficiency and the target's defenses
  arrive as caller facts: small records and closed states (`MarkStatus`:
  `MarkedByYourHuntersMark` | `NotMarked`), never loose flags. The Ranger returns
  rider dice expressions; it does not roll them.
- **Exports** from the `character` package: the use
  cases, pure functions, commands, results, `RangerCreationChoices`,
  `RangerChoices`, `RangerResources`, `RangerEffects`, rider, grant, `AttackerKey`,
  `RangerEvent` types and `RangerRejection`; all names operation-prefixed
  (FR-112). Telemetry constants live in one context file
  (`doc/rules/observability.md` §5).

## 7. telemetry.Telemetry

Class specs define no feature-use signals of their own (shared-types-and-ownership.md
FR-042, FR-063). The three use cases use the shared signals owned by
character-aggregate.md: counter `srd5e.character.feature_used`, histogram
`srd5e.character.feature_used.duration` and counter
`srd5e.character.feature_used.errors`. The domain and the five pure functions are
**silent by design**: they return results and events. Values this spec adds:

- `srd5e.class` = `ranger`; `srd5e.feature` = `tireless` (`UseTireless`),
  `ranger_features` (`ResolveRangerFeatures`), and for `ChooseRangerOption` the
  feature chosen: `weapon_mastery`, `deft_explorer`, `fighting_style`,
  `expertise`, `hunters_prey`, `defensive_tactics`, `feat_grant`.
- `srd5e.outcome`: `success` for the accepted or resolved variant; `failure` for
  a rejected variant, `...ContentMissing`, `...NotARanger`, and also a returned port
  failure (one `feature_used` increment, no `error.type`, no rejection; plus one
  `.errors` increment).
- Spans: `srd5e.character.choose_ranger_option`, `srd5e.character.resolve_ranger_features`, `srd5e.character.use_tireless`. Each call records its latency in `srd5e.character.feature_used.duration` through `telemetry.Timer` (character-aggregate.md FR-118).
- `srd5e.character.rejection`: `no_uses_remaining`, `feature_not_available`,
  `wrong_moment`, `same_option`, `nothing_to_replace`, `already_chosen`,
  `unknown_option`, `not_proficient`, `already_has_expertise`,
  `no_grant_remaining`, `invalid_choice` (counts, distinctness or membership, and
  any rejection mapped from `WeaponMasteryChoice`, `FeatTakingRules` or `FeatQualification`), `not_hunter`,
  plus `content_missing` and `not_a_ranger`.

Evaluation (`doc/rules/observability.md` §2): failures are missing content, a
`ContentSource` or `dice.Roller` failure, an illegal request, invalid host input
and programmer errors (a roll outside 1 to 8). A returned port failure is span error
status, one `feature_used` increment (`outcome=failure`) plus one `.errors` increment with `error.type`; missing content is a
failure result (span error, `content_missing`, no `error.type`); a rejection is
`outcome=failure` with normal span status; a low roll is not a failure. Roll,
resources, rider and grant are in the result, never in attributes. Alert on any
non-zero `.errors` rate and review `content_missing`; watch `no_uses_remaining`
for `tireless` and `wrong_moment` for `weapon_mastery`, `hunters_prey`,
`defensive_tactics`. Nothing is noisy: the use cases log nothing and rejections
are not warnings. Entries feed `doc/observability/signals.md`.

## 8. Test plan

Levels: unit (`character_test` package (`*_test.go`)), application (`character_test` use-case tests,
recording `telemetrytest.Recorder`, fixed `dice.Roller`, `FakeContentSource` from
the `*test` packages), scenario, architecture. No new port, no new contract suite.
Withdrawn ids (4.12) are tested by their new owners.

| Id | Level | Notes |
|---|---|---|
| FR-023, FR-030 to FR-032 | unit | T2 per row; FR-030 consumer-side (H4 tested in spellcasting.md); `hunters_mark` fact at levels 1 and 20; free casts by T1, T2 |
| FR-033 to FR-036, FR-075, FR-080 | unit | `useFavoredEnemy`: last use accepted then rejected; level-up keeps expended; `applyRangerRest`: Long Rest resets, Short Rest does not |
| FR-037, FR-040, FR-041, FR-108 | unit, application | Count 2 and eligibility passed to `WeaponMasteryChoice`; initial choice with no moment; `finishedLongRest` accepted, Short Rest, level gain, none rejected with `WrongMoment`; `NothingToReplace` for each replaceable choice |
| FR-042 to FR-048 | unit | Deft Explorer, language counts 1, 2, 3 and duplicate, levels 1 and 2; Expertise at 9, 8 rejected, Deft Explorer skill excluded, counts 1, 2, 3 |
| FR-049, FR-051 to FR-055 | unit | Feat validated through `FeatTakingRules`/`FeatQualification`, `toTakeFeat()` handoff returned; Druidic Warrior and bad cantrips; `GrantedCantrip`; replacement on gaining a level only; second choice rejected; exclusivity by shape |
| FR-062, FR-064 | unit | Epic Boon from level 19; no grant left; T6 sums |
| FR-065 to FR-070 | unit | Attacks 1 at level 4, 2 at 5 and 20, a total; T3 per row |
| FR-071 to FR-074, FR-077, FR-100, FR-107 | unit, application | T4 per row with a fixed roller (1 and 8); Wisdom drop below expended; rolls 0, 1, 8, 9 (a double returning 9 and 0 propagates panic, spends no use); throwing `dice.Roller`: propagated, resources unchanged, one `feature_used` increment (`outcome=failure`) and one `.errors` increment, no log |
| FR-076 | unit | `applyRangerRest`: Exhaustion 0, 1, 6, negative; levels 9 and 10 |
| FR-078 to FR-083 | unit | Levels 12, 13 with `hunters_mark` and another spell (consumer tested in combat-action-resolution.md); 13, 14 Invisible expiry and Bonus Action; 16, 17, 18 marked and unmarked; T5 per row |
| FR-084 to FR-095 | unit | Marked, unmarked, no Hunter, level 2; `DefenseEntry` qualifiers kept; options and replacement moments, `SameOption`; T8 and T9 per row; Opportunity Attack versus other attack; `AttackerKey`, next turn empties the record; Reaction or not, levels 14, 15, expiry |
| FR-096 to FR-098 | unit, application | Effects by level; T11 per row; FR-098 consumer-side |
| FR-099, FR-106, FR-111 to FR-113 | unit, architecture | Inputs not mutated, returned slices are defensive copies (mutating one leaves the state unchanged); consumer-name guardrail; package API scan (no generic or duplicate name, pure functions import no `telemetry.Telemetry`); events are `RangerEvent` variants, no `CharacterId`, none a `CharacterEvent` |
| FR-101 to FR-105 | application | Not found yields the prefixed `...ContentMissing`; failing double propagates; no Ranger levels; per use case: same result with no-op and recording `telemetrytest.Recorder`, one counter increment, error status, no log; span names, duration recorded, attribute values pinned |
| FR-109, FR-110 | unit | Seeding writes both keys and leaves others; content keys name the pair; the `New` constructor (returns an error) delegates; `.of(states)` on empty states returns `initial`, raw `read` is null; round-trip |
| Whole slice | scenario | A level 11 Hunter chooses Colossus Slayer, spends Favored Enemy uses, takes a Short and a Long Rest; a level 10 Ranger uses Tireless with a fixed roller |

Mutation testing: changed lines in the `character` package (pure domain files) at a score of at
least 80 (`doc/rules/testing.md` §8), run with the capped Docker `gomu` command. Not covered: none.

## 9. Open questions and decisions

Open: none. Escalated: none.

Settled (one line each; cited by requirements):

| Decision | Reading |
|---|---|
| Tireless minimum of 1 (FR-072, T4) | Interpretation: the parenthetical "(minimum of 1)" (SRD 5.2.1 p. 59) sits next to the Wisdom modifier, as in the same feature's uses ("your Wisdom modifier (minimum of once)"); web sources split, so the SRD wording decides. This differs from Wholeness of Body (p. 52), whose minimum is stated on the total ("minimum of 1 Hit Point regained"); the Tireless text states no total, so the modifier reading holds (TL-04, TL-05 cover negative modifiers) |
| Roving Climb and Swim under Heavy armor (FR-069); Speed 0 from a condition (FR-070) | They apply without the +10; `canIncrease` flag, a genuine base 0 yields 10 (SRD p. 59: "Speed increases by 10 feet while you aren.t wearing Heavy armor") |
| Colossus Slayer damage type (FR-088) | `UnstatedDamageType`; not inferred |
| Deft Explorer known languages (FR-045) | Repeats within the pair only; character-creation.md FR-049 makes an already-known language a no-op |
| Fighting Style change (FR-054); Favored Enemy level-up (FR-035) | No SRD way to change; track expended uses (SRD pp. 57-59 give no change rule) |
| `DamageCanBreakConcentration` consumer (FR-078) | Host-supplied fact to combat-action-resolution.md `ConcentrationInput`; no `combat` import of a Ranger function (architecture.md §2) |
| *Hunter's Mark* base die (FR-083) | d6, SRD 5.2.1 p. 141; an input from spell content |
| Ownership of names, state, use-case split, telemetry, events | shared-types-and-ownership.md FR-005, FR-006, FR-012, FR-013, FR-019, FR-022, FR-023, FR-041, FR-042; extra attack stacking level-advancement-and-multiclassing.md FR-054 to FR-058; mastery eligibility weapon-properties-and-mastery.md FR-019; `DefenseEntry` content-monsters.md FR-024; half-casters spellcasting.md FR-032 |

## 10. Changelog impact

`Added`: the Ranger behaviour in `character` (types, pure feature functions, the
use cases `ChooseRangerOption`, `ResolveRangerFeatures` and `UseTireless`) on the
shared signals of section 7. Not **BREAKING**. Ranger class data is recorded in
content-classes.md; `NOTICE.md` coverage (shared-types-and-ownership.md FR-033) is
unchanged.
