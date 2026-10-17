# Class: Cleric

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | Types 3; bulk data 5 |
| Depends on | `content-schema-and-decoder.md`, `derived-stats.md`, `character-aggregate.md`, `spellcasting.md`, `character-creation.md`, `level-advancement-and-multiclassing.md`, `rests-and-hit-dice.md`, `content-classes.md`, `content-spells.md`, `d20-tests.md`, `ability-scores.md`, `dice-and-rolls.md`, `observability-ports.md`, `errors-and-results.md` |

The Cleric class data ships in Phase 3 with content-classes.md, which owns it (shared-types-and-ownership.md FR-014, FR-025). The consistency tests FR-015 and FR-017 read only the bundled Cleric Features table and run in Phase 3 with content-classes.md; the FR-133 id-existence test reads bundled spell ids and is a Phase 5 test (after the bulk spell data of content-spells.md; the thin slice holds none of those spells). No dependency on any `combat` spec (shared-types-and-ownership.md FR-013). resolve-d20-test-use-case.md (saving throws) and the decoder of content-schema-and-decoder.md are cited in prose only; its vocabulary types (`DamageType`, `ConditionKind`, `CreatureType`, `Skill`, typed ids) are used directly.

## 1. Purpose and scope

Specifies the Cleric class as SRD 5.2.1 presents it (pp. 36-40): the Core Cleric Traits, the Cleric Features table, every class feature from level 1 to 20, the Cleric spell list, and the one subclass the SRD gives, Life Domain. Per shared-types-and-ownership.md FR-025 this spec owns **behaviour only**:

1. **Content data is not owned here.** `ClassDefinition`, `SubclassDefinition`, `ProgressionRow`, `FeatureEntry` and the bulk Cleric and Life Domain data (Cleric Features table, Life Domain Spells `AlwaysPreparedSpells` fact, oracle tables FS, SQ, CC, F, FA and the fixture of FR-093) belong to content-classes.md. Spell-list membership, school, Concentration and Ritual marks are `Spell.classes`, `Spell.school`, duration and `ritual` (content-spells.md FR-022 to FR-025, "Distribution by class", Cleric 109). `character` never decodes content (content-classes.md FR-098). Checks of the bundled data against the SRD are withdrawn to those specs (4.15).
2. **Behaviour in the `character` context**: the Divine Order choice, Channel Divinity uses and its three effects, Sear Undead, Blessed Strikes, Divine Intervention and Greater Divine Intervention, and the Life Domain healing features. Feature logic is a pure domain function; only Channel Divinity and Divine Intervention, which need dice, also have a use case (shared-types-and-ownership.md FR-023). State lives in the aggregate's `FeatureStates` (shared-types-and-ownership.md FR-022; character-aggregate.md FR-092, FR-093); creation choices travel in `CreateCharacter` (shared-types-and-ownership.md FR-024; character-aggregate.md FR-094).

Spellcasting is **modelled by spellcasting.md**; this spec states only the Thaumaturge's extra-cantrip allowance (FR-141) and the always-prepared Life Domain spells read from content (FR-098). Effects resolved against other creatures (damage, healing, Temporary Hit Points, Frightened and Incapacitated, movement) belong to the downstream `combat` context; this spec returns amounts, DCs, `DamageType`s, `ConditionKind`s, durations and early endings as data.

**Out of scope:**

- Spellcasting rules, the spells themselves, and the definitions of Starting Equipment items, armor, weapons and the Holy Symbol (spellcasting.md, content specs); this spec stores content ids. Holding a Holy Symbol for Divine Spark and Turn Undead is the host's.
- Feats (Ability Score Improvement, Epic Boon, the recommendation of Boon of Fate; content-species-backgrounds-feats.md) and the pending feat and subclass choices (level-advancement-and-multiclassing.md FR-016, FR-017).
- Class skill selection and the Starting Equipment option: character-creation.md (FR-015 to FR-020, FR-070). Subclass choice and level validation: level-advancement-and-multiclassing.md (shared-types-and-ownership.md FR-020). Multiclassing prerequisites and what a first level in a new class grants: character-aggregate.md, level-advancement-and-multiclassing.md (FR-108, FR-110), character-creation.md (FR-104); only the Divine Order on a multiclass first Cleric level stays here (FR-022).
- Applying damage, healing and Temporary Hit Points, their stacking, Bloodied as a creature state, Critical Hit doubling, Short and Long Rest rules (rests-and-hit-dice.md), the conditions, actions and movement of a frightened creature (`combat`). Creature facts arrive as plain values.
- Converting "1 minute" to rounds (no clock); line of sight (host-supplied, FR-132); ability score generation; loading and decoding content (`MissingContent` is the loader's).
- Other Cleric subclasses (SRD 5.2.1 contains only Life Domain, p. 40). Nothing outside SRD 5.2.1 pp. 36-40 was used, and nothing that looked like core fifth-edition Cleric material was missing from the PDF for the features listed.

## 2. Source

PDF page numbers equal printed page numbers (checked: PDF page 36 shows printed page 36, the start of "Cleric").

- SRD 5.2.1, "Cleric", "Core Cleric Traits", "Becoming a Cleric ...", "As a Level 1 Character", "As a Multiclass Character", "Cleric Features" table, p. 36
- SRD 5.2.1, "Cleric Class Features": "Level 1: Spellcasting" ("Cantrips" p. 36; "Prepared Spells of Level 1+" p. 37), "Level 1: Divine Order", "Level 2: Channel Divinity", "Level 3: Cleric Subclass", "Level 4: Ability Score Improvement", "Level 5: Sear Undead", p. 37; "Level 7: Blessed Strikes", "Level 10: Divine Intervention", "Level 14: Improved Blessed Strikes", "Level 19: Epic Boon", "Level 20: Greater Divine Intervention", p. 38
- SRD 5.2.1, "Cleric Spell List", pp. 38-40; "Cleric Subclass: Life Domain" (Disciple of Life, Life Domain Spells, Preserve Life, Blessed Healer, Supreme Healing), p. 40
- SRD 5.2.1, "Ability Modifiers", p. 6; "Proficiency", p. 8; "Multiclassing" ("Proficiency Bonus", "Proficiencies", "Class Features"), p. 25; "Hit Points" (Bloodied), "Damage Rolls", p. 16; "Healing", p. 17; "Temporary Hit Points", p. 18; "Bloodied", p. 177; "Creature Type", p. 179; "Casting without Slots", p. 104
- Package design: shared-types-and-ownership.md; `doc/rules/architecture.md` (§2, §3, §4, §6), `doc/rules/observability.md` (§2, §5, §6, §7), `doc/rules/testing.md` (§4).

## 3. Vocabulary

The Core Cleric Traits (Wisdom, D8, saves Wisdom and Charisma, skills, Simple weapons, Light and Medium armor and Shields, Starting Equipment A or B), the Cleric Features table and the 109-spell list are **content data** (content-classes.md, content-spells.md; `ClassId` `cleric`, shared-types-and-ownership.md FR-007), validated elsewhere.

| SRD term | Type | Notes |
|---|---|---|
| `ClericRuleData` | Value object (SRD defaults, injectable) | Rule data the content format keeps only as text: the recommended spells (FR-027) and the extra Divine Intervention grants (Wish, level 20, 2d4; FR-094). FR-133. |
| `ClericRuleData`'s companions | `ClericCreationChoices`, `ClericChoices`, `ClericResources`, `ClericPendingChoice` | Creation choices (FR-135, FR-140); `FeatureState`s under `ClericFeatureKeys` (FR-139) holding the Divine Order and Blessed Strikes option, and Channel Divinity uses and Divine Intervention state; the enum `divineOrder`, `blessedStrikesOption` (FR-021). |
| Divine Order | Closed `DivineOrder` | `DivineOrderProtector` (Martial weapons, Heavy armor) or `DivineOrderThaumaturge` (one extra cantrip, `extraCantrips` 1, FR-141; Intelligence (Arcana or Religion) bonus). |
| Channel Divinity | `ClericChannelDivinityUses` (maximum, remaining) | Effects: Divine Spark (30 feet, healing or Constitution save damage), Turn Undead (Wisdom save, Frightened and Incapacitated), Preserve Life (Life Domain). Sear Undead modifies Turn Undead; optional, `SearUndeadChoice` (FR-131). |
| Blessed Strikes | Closed `BlessedStrikesOption` | `BlessedStrikesDivineStrike` (extra 1d8, 2d8 from level 14) or `BlessedStrikesPotentSpellcasting` (Wisdom modifier on Cleric cantrip damage; from level 14 also Temporary Hit Points). |
| Divine Intervention | Closed `DivineInterventionState` | `DivineInterventionAvailable`, `DivineInterventionUnavailableUntilLongRest`, `DivineInterventionRecharging`. A `CastMode.featureGranted` cast (casting-and-concentration.md FR-121), once per Long Rest. |
| Life Domain | `SubclassId` `life_domain` (content-classes.md table SRC) | Membership is `SubclassId` equality with `LifeDomain.subclassId`, tested against the bundled subclass (FR-098). Features: Disciple of Life, Life Domain Spells (read, not copied), Preserve Life, Blessed Healer, Supreme Healing. |
| Bloodied | Glossary term (p. 177) | Decided by derived-stats.md `isBloodied` (shared-types-and-ownership.md FR-049), never recomputed here (FR-106). |

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

Stable ids. Never renumber; new requirements get new ids. "Cleric level" means the character's level in the Cleric class, not total character level, unless stated (FR-117). Numbers 4.1, 4.3, 4.4 and 4.9 are unused (they held only withdrawn ids, see 4.15); 4.14.1 to 4.14.4 are the case tables of 4.13. Each merged case label (for example AC-11, AC-13, AC-15) expands to one test per listed level.

### 4.2 Cleric Features table, becoming a Cleric, spellcasting data
The Cleric Features table is the `ProgressionRow`s of the `cleric` `ClassDefinition` (content-classes.md); this spec holds **no copy**. The oracle is content-classes.md tables FS, SQ, CC and F (fixture `testdata/srd/classes/oracle.json`, FR-093); `rowAt` and `entriesGainedAt` are content-classes.md FR-047 and FR-048. FR-015 and FR-017 are `character`-side consistency tests (`character_test` package (`*_test.go`)), **not decode failures**: `content` never imports `character`, so no runtime path rejects content because of them. Spellcasting mechanics, the extra-cantrip cap and the cantrip headroom are spellcasting.md's (FR-096, FR-052, FR-047); the Cleric spell list is the set of spells whose `Spell.classes` contains the Cleric `ClassId`, with oracle content-spells.md FR-023 (`testdata/srd/spells/class_list_oracle.json`).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-015 | The Proficiency Bonus column of the bundled Cleric Features table MUST equal the general Proficiency Bonus for the same level; a disagreement MUST fail the `character`-side consistency test (not a decode). | SRD 5.2.1, "Cleric Features", p. 36; "Proficiency", p. 8 | unhappy (rule interaction) |
| FR-017 | The Cantrips, Prepared Spells and Spell Slots columns of the bundled table MUST agree with the Cleric values that spellcasting.md derives for the same level; a disagreement MUST fail the `character`-side consistency test (not a decode). | SRD 5.2.1, "Cleric Features", p. 36 | unhappy (rule interaction) |
| FR-021 | The features a Cleric gains at level L (1 to 20) are `ClassDefinition.entriesGainedAt(L)` of the `cleric` definition (content-classes.md FR-048), as `FeatureId`s; this spec adds no second reader and no progression service. The only pending choices a Cleric level adds are the enum `ClericPendingChoice`: `divineOrder` at Cleric level 1 and `blessedStrikesOption` at Cleric level 7, never strings. The subclass and feat choices are level-advancement-and-multiclassing.md (FR-016, FR-017) and the cantrip headroom is spellcasting.md (FR-047). | SRD 5.2.1, "Cleric Class Features", pp. 36-38 | happy |
| FR-022 | A multiclass Cleric MUST make the Divine Order choice of FR-029 on gaining its first Cleric level, because Divine Order is a level 1 feature; choosing Protector MUST then grant Martial weapon proficiency and Heavy armor training (FR-031). | SRD 5.2.1, "As a Multiclass Character", p. 36; "Level 1: Divine Order", p. 37 | unhappy (rule interaction) |
| FR-027 | `ClericRuleData.recommendedCantrips` MUST return the content ids of Guidance, Sacred Flame and Thaumaturgy and `ClericRuleData.recommendedLevel1Spells` those of Bless, Cure Wounds, Guiding Bolt and Shield of Faith; no use case MUST reject a selection for differing from them. | SRD 5.2.1, "Level 1: Spellcasting" ("Cantrips" p. 36; "Prepared Spells of Level 1+" p. 37) ("are recommended") | unhappy (rule interaction) |

### 4.5 Divine Order (level 1)
| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-029 | The Divine Order choice MUST accept exactly one of Protector or Thaumaturge (`DivineOrderProtector` or `DivineOrderThaumaturge`). | SRD 5.2.1, "Level 1: Divine Order" ("one of the following sacred roles"), p. 37 | happy |
| FR-030 | A Divine Order choice that is missing, names an unknown order, or names both orders MUST be rejected with `ChooseDivineOrderRejected` carrying reason `order_missing`, `order_unknown` or `both_orders` (precedence: FR-128). | SRD 5.2.1, "Level 1: Divine Order", p. 37 | unhappy (invalid input) |
| FR-031 | Protector MUST grant proficiency with Martial weapons and training with Heavy armor, exposed as additions to the Core Cleric Traits. | SRD 5.2.1, "Level 1: Divine Order" (Protector), p. 37 | happy |
| FR-032 | Thaumaturge MUST NOT grant Martial weapon proficiency or Heavy armor training, and Protector MUST NOT grant the extra cantrip or the check bonus of FR-035. | SRD 5.2.1, "Level 1: Divine Order", p. 37 | unhappy (rule interaction) |
| FR-033 | The cap on cantrips known and the rule that the extra cantrip is a cantrip of the Cleric list are spellcasting.md's (FR-096, FR-052); this spec does not restate them. The extra cantrip, passed in the command as a `Spell` (content-spells.md), is checked at choice time with the same list check so a bad choice is rejected before it is stored: it MUST be a level 0 spell with Cleric in `Spell.classes` that is not already known. Otherwise the choice MUST be rejected with `ChooseDivineOrderRejected` carrying `extra_cantrip_missing` (Thaumaturge without a `Spell`), `extra_cantrip_not_cleric_cantrip` (not level 0, or not on the Cleric list) or `extra_cantrip_already_known`. The allowance itself is FR-141. | SRD 5.2.1, "Level 1: Divine Order" (Thaumaturge), p. 37 | unhappy (invalid input) |
| FR-034 | A Protector MUST supply no extra-cantrip allowance (`extraCantrips` 0, the spellcasting.md FR-096 default), so the number of cantrips known stays the Table F count. | SRD 5.2.1, "Level 1: Divine Order" (Protector), p. 37; "Cleric Features", p. 36 | unhappy (rule interaction) |
| FR-035 | Thaumaturge MUST give a bonus (a `Modifier`, FR-137) to Intelligence (Arcana) and Intelligence (Religion) checks equal to the Wisdom modifier, with a minimum of +1 (cases in 4.5.1). | SRD 5.2.1, "Level 1: Divine Order" (Thaumaturge), p. 37; "Ability Modifiers", p. 6 | happy |
| FR-036 | The bonus of FR-035 MUST NOT apply to any other ability check, including other Intelligence skills, or to a Protector. | SRD 5.2.1, "Level 1: Divine Order" (Thaumaturge), p. 37 | unhappy (rule interaction) |
| FR-037 | A second Divine Order choice for a character that already holds one MUST be rejected with `ChooseDivineOrderRejected(already_chosen)` and MUST leave state unchanged (D-4). | SRD 5.2.1, "Level 1: Divine Order" ("You have dedicated yourself"), p. 37 | unhappy (state error) |

#### 4.5.1 Thaumaturge check bonus cases (one test per row; source: SRD 5.2.1, "Level 1: Divine Order", p. 37, and "Ability Modifiers", p. 6)
| Label | Order | Wisdom score | Wisdom modifier | Check | Expected bonus | Path |
|---|---|---|---|---|---|---|
| TB-01 | Thaumaturge | 14 | +2 | Arcana | +2 | happy |
| TB-02 | Thaumaturge | 14 | +2 | Religion | +2 | happy |
| TB-03 | Thaumaturge | 12 | +1 | Arcana | +1 | unhappy (boundary: equals the minimum) |
| TB-04 | Thaumaturge | 10 | +0 | Religion | +1 | unhappy (boundary: raised to the minimum) |
| TB-05 | Thaumaturge | 8 | -1 | Arcana | +1 | unhappy (boundary: negative raised to the minimum) |
| TB-06 | Thaumaturge | 1 | -5 | Religion | +1 | unhappy (boundary: lowest score) |
| TB-07 | Thaumaturge | 20 | +5 | Arcana | +5 | happy |
| TB-08 | Thaumaturge | 30 | +10 | Religion | +10 | unhappy (boundary: highest score) |
| TB-09 | Thaumaturge | 14 | +2 | History | none | unhappy (rule interaction) |
| TB-10 | Thaumaturge | 14 | +2 | Intelligence (Investigation) | none | unhappy (rule interaction) |
| TB-11 | Protector | 14 | +2 | Arcana | none | unhappy (rule interaction) |

### 4.6 Channel Divinity (level 2)
| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-038 | The maximum Channel Divinity uses at Cleric level L is the value of the Channel Divinity column of the level's `ProgressionRow`, `ProgressionRow.value(ClassColumnId channel_divinity)` (content-classes.md FR-050, table CC); this spec holds no maximum literal of its own, so a house-rule table changes the maximum with no code change. The SRD values are the oracle and the consistency test (as FR-015 and FR-017): 0 below level 2, 2 at levels 2 to 5, 3 at levels 6 to 17 and 4 at levels 18 to 20; a disagreement between the bundled column and those values MUST fail the `character`-side consistency test (not a decode). | SRD 5.2.1, "Level 2: Channel Divinity", p. 37; "Cleric Features", p. 36 | happy |
| FR-039 | Using Channel Divinity MUST require choosing exactly one available effect, expend exactly one use, and return the effect's data (FR-048 onward) with a `ClericChannelDivinityUsed` event. | SRD 5.2.1, "Level 2: Channel Divinity" ("Each time you use it, choose which Channel Divinity effect"), p. 37 | happy |
| FR-040 | Using Channel Divinity with no remaining use MUST be rejected with `UseClericChannelDivinityRejected(no_uses_remaining)` and MUST NOT change state. | SRD 5.2.1, "Level 2: Channel Divinity" ("expended uses"), p. 37 | unhappy (state error) |
| FR-041 | Using Channel Divinity below Cleric level 2 MUST be rejected with `UseClericChannelDivinityRejected(below_level_2)`. | SRD 5.2.1, "Level 2: Channel Divinity", p. 37 | unhappy (boundary) |
| FR-042 | Finishing a Short Rest MUST regain one expended use, never exceeding the maximum; with no use expended it changes nothing. | SRD 5.2.1, "Level 2: Channel Divinity" ("You regain one of its expended uses when you finish a Short Rest"), p. 37 | unhappy (boundary) |
| FR-043 | Finishing a Long Rest MUST regain all expended uses. | SRD 5.2.1, "Level 2: Channel Divinity", p. 37 | happy |
| FR-044 | `AdvanceCleric` reaching a level that raises the Channel Divinity maximum (levels 2, 6 and 18 for the SRD data, read from the Channel Divinity column of the `ProgressionRow` passed in, FR-038) MUST raise the maximum and MUST raise the remaining uses by the same number, never above the maximum; a level that does not raise it MUST change neither (D-2). `ClericResources` holds the maximum, so no Cleric operation loads content. | SRD 5.2.1, "Level 2: Channel Divinity" ("You gain additional uses when you reach certain Cleric levels"), p. 37 | unhappy (boundary) |
| FR-045 | When a Channel Divinity effect requires a saving throw, the effect data MUST carry a DC equal to the `SpellSaveDc` supplied in the command (spellcasting.md), carried as the `DifficultyClass` it wraps (d20-tests.md). The effect data carries the roll's outcome as data (damage applying on a failed save); the saving throw itself is resolved elsewhere (section 6 notes). | SRD 5.2.1, "Level 2: Channel Divinity" ("the DC equals the spell save DC"), p. 37 | happy |
| FR-046 | Naming an effect the character does not have (an unknown effect, or Preserve Life without the Life Domain subclass or below level 3) MUST be rejected with `UseClericChannelDivinityRejected(effect_not_available)` naming the effect and MUST NOT expend a use. | SRD 5.2.1, "Level 2: Channel Divinity", p. 37; "Level 3: Preserve Life", p. 40 | unhappy (state error) |
| FR-047 | `ClericChannelDivinityUses` MUST reject (panic) a negative remaining value or a remaining value above the maximum. | n/a (constructor invariant); SRD 5.2.1, "Level 2: Channel Divinity", p. 37 | unhappy (invalid input) |

### 4.7 Divine Spark (level 2)
| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | Divine Spark MUST be exposed as a Magic action that targets one creature the Cleric can see within 30 feet, using the Cleric's Holy Symbol, as either healing or a Constitution saving throw dealing damage. | SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37 | happy |
| FR-049 | The number of d8s rolled MUST be 1 at Cleric levels 2 to 6, 2 at levels 7 to 12, 3 at levels 13 to 17 and 4 at levels 18 to 20 (cases in 4.7.1). | SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37 | unhappy (boundary) |
| FR-050 | The Divine Spark total MUST equal the sum of the d8s plus the Wisdom modifier (cases in 4.7.1). | SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37 | happy |
| FR-051 | In healing mode the effect MUST report the Hit Points to restore as the total, not below 0. Capping at the target's Hit Point maximum belongs to `combat` (it receives this amount as a plain `int`). | SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37; "Healing", p. 17 | happy |
| FR-052 | In damage mode the effect MUST report a Constitution saving throw at the supplied DC, damage on a failed save equal to the total, and damage on a successful save equal to half the total rounded down, of a type the Cleric chooses between Necrotic and Radiant; any other damage type MUST be rejected with `UseClericChannelDivinityRejected(damage_type_not_allowed)`. | SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37 | happy |
| FR-053 | A total below 0 (a negative Wisdom modifier above a low roll) MUST yield an amount of 0 in both modes and 0 damage on a successful save, never a negative amount. | SRD 5.2.1, "Damage Rolls" ("possible to deal 0 damage but not negative damage"), p. 16; healing side: D-3 | unhappy (boundary) |
| FR-054 | Roll values outside 1 to 8, or a count of roll values different from FR-049's number of dice, MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error) from the domain service. | n/a (constructor invariant); SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37 | unhappy (invalid input) |
| FR-055 | Except where Supreme Healing applies (FR-114, FR-115: Life Domain Cleric of level 17 or higher, healing mode), in which case the use case MUST NOT call the `dice.Roller`, `UseClericChannelDivinity` MUST obtain exactly the number of d8 values of FR-049 through `dice.Roller.rollChecked` and pass them to the domain; the domain MUST NOT roll. | doc/rules/architecture.md §3, §4; SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37 | happy |
| FR-056 | A target farther than 30 feet MUST be rejected with `UseClericChannelDivinityRejected(target_out_of_range)`; a target at exactly 30 feet MUST be accepted (cases in 4.7.2). | SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark, "within 30 feet"), p. 37 | unhappy (boundary) |
| FR-057 | A target that is the Cleric itself MUST be rejected with `UseClericChannelDivinityRejected(target_is_self)` and MUST NOT expend a use. | SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark, "another creature"), p. 37 | unhappy (invalid input) |

#### 4.7.1 Divine Spark cases (one test per row; source: SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37)
"Fail" is the damage on a failed Constitution save; "Pass" on a success (half, rounded down).

| Label | Cleric level | d8s | Wisdom modifier | Mode | Total | Healing | Fail | Pass | Path |
|---|---|---|---|---|---|---|---|---|---|
| DS-01 | 2 | 1d8 = 5 | +3 | healing | 8 | 8 | n/a | n/a | happy |
| DS-02 | 2 | 1d8 = 5 | +3 | damage | 8 | n/a | 8 | 4 | happy |
| DS-03 | 6 | 1d8 = 4 | +2 | damage | 6 | n/a | 6 | 3 | unhappy (boundary: last level at 1d8) |
| DS-04 | 7 | 2d8 = 3, 8 | +4 | damage | 15 | n/a | 15 | 7 | unhappy (boundary: first level at 2d8; odd total rounds down) |
| DS-05 | 12 | 2d8 = 1, 1 | +0 | healing | 2 | 2 | n/a | n/a | unhappy (boundary: last level at 2d8) |
| DS-06 | 13 | 3d8 = 8, 8, 8 | +5 | damage | 29 | n/a | 29 | 14 | unhappy (boundary: first level at 3d8; maximum rolls) |
| DS-07 | 17 | 3d8 = 2, 2, 2 | +1 | healing | 7 | 7 | n/a | n/a | unhappy (boundary: last level at 3d8) |
| DS-08 | 18 | 4d8 = 1, 1, 1, 1 | +5 | damage | 9 | n/a | 9 | 4 | unhappy (boundary: first level at 4d8; minimum rolls) |
| DS-09 | 20 | 4d8 = 8, 8, 8, 8 | +5 | damage | 37 | n/a | 37 | 18 | happy |
| DS-10 | 2 | 1d8 = 1 | +0 | damage | 1 | n/a | 1 | 0 | unhappy (boundary: half of 1 rounds down to 0) |
| DS-11 | 2 | 1d8 = 1 | -1 | healing | 0 | 0 | n/a | n/a | unhappy (boundary: total exactly 0) |
| DS-12 | 2 | 1d8 = 2 | -5 | damage | -3 | n/a | 0 | 0 | unhappy (boundary: negative total gives 0, FR-053) |
| DS-13 | 2 | 1d8 = 9 | +3 | healing | n/a | n/a | n/a | n/a | unhappy (invalid input): panic |
| DS-14 | 7 | 1d8 = 4 (one value for two dice) | +3 | damage | n/a | n/a | n/a | n/a | unhappy (invalid input): panic |

#### 4.7.2 Divine Spark target cases (one test per row; source: SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark, "another creature you can see within 30 feet"), p. 37)
All rows: Cleric level 2 with a use remaining; reasons and precedence are FR-128.

| Label | Target | Expected | Path |
|---|---|---|---|
| DT-01 | another creature, visible, 30 feet | accepted | unhappy (boundary: exactly 30 feet) |
| DT-02 | another creature, visible, 31 feet | rejected `target_out_of_range` | unhappy (boundary: one above) |
| DT-03 | another creature, visible, 0 feet | accepted | unhappy (boundary: distance 0) |
| DT-04 | the Cleric itself | rejected `target_is_self` | unhappy (invalid input) |
| DT-05 | another creature, not visible, 10 feet | rejected `target_not_visible` (FR-132) | unhappy (invalid input) |
| DT-06 | the Cleric itself, not visible, 40 feet | rejected `target_is_self` (first in the order of FR-128) | unhappy (rule interaction: several apply) |
| DT-07 | another creature, not visible, 40 feet | rejected `target_not_visible` (before range) | unhappy (rule interaction: several apply) |

### 4.8 Turn Undead and Sear Undead (levels 2 and 5)
| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-058 | Turn Undead MUST be exposed as a Magic action affecting each Undead of the Cleric's choice within 30 feet, each making a Wisdom saving throw at the supplied DC (FR-045). | SRD 5.2.1, "Level 2: Channel Divinity" (Turn Undead), p. 37 | happy |
| FR-059 | On a failed save the effect data MUST give the creature the Frightened and Incapacitated conditions (`ConditionKind.frightened` and `ConditionKind.incapacitated`; their effects belong to `combat`) for 1 minute, and report that on its turns it tries to move as far from the Cleric as it can. | SRD 5.2.1, "Level 2: Channel Divinity" (Turn Undead), p. 37 | happy |
| FR-060 | The effect data MUST list its early endings: the creature takes any damage, the Cleric has the Incapacitated condition, or the Cleric dies. | SRD 5.2.1, "Level 2: Channel Divinity" (Turn Undead), p. 37 | happy |
| FR-061 | A chosen target whose `CreatureType` is not Undead MUST be rejected with `UseClericChannelDivinityRejected(target_not_undead)` naming it, and no use MUST be expended. | SRD 5.2.1, "Level 2: Channel Divinity" (Turn Undead, "Each Undead"), p. 37; "Creature Type", p. 179 | unhappy (invalid input) |
| FR-062 | A target farther than 30 feet MUST be rejected with `UseClericChannelDivinityRejected(target_out_of_range)`; a target at exactly 30 feet MUST be accepted. | SRD 5.2.1, "Level 2: Channel Divinity" (Turn Undead, "within 30 feet"), p. 37 | unhappy (boundary) |
| FR-063 | Turn Undead with no chosen target MUST be accepted, expend one use and report an empty target list (D-6). | SRD 5.2.1, "Level 2: Channel Divinity" (Turn Undead, "Each Undead of your choice"), p. 37 | unhappy (empty and minimal) |
| FR-064 | From Cleric level 5, when the command's `SearUndeadChoice` is `use` (FR-131), Turn Undead MUST also report one roll shared by all targets: a number of d8s equal to the Wisdom modifier, minimum 1d8, added together (cases in 4.8.1). The roll is optional in the SRD ("you can roll"); `skip` rolls nothing and reports no Sear Undead data. | SRD 5.2.1, "Level 5: Sear Undead", p. 37 | happy |
| FR-065 | The effect data MUST carry the Sear Undead total as Radiant damage that applies to each Undead that fails its Wisdom saving throw (damage conditional on a failed save; saves are resolved elsewhere, so this use case does not know who fails), and MUST report that this damage does not end the Turn Undead effect, overriding the early ending "takes any damage" of FR-060. | SRD 5.2.1, "Level 5: Sear Undead", p. 37 | unhappy (rule interaction) |
| FR-066 | Below Cleric level 5 no Sear Undead data MUST be reported and `SearUndeadChoice.use` MUST be rejected with `UseClericChannelDivinityRejected(sear_undead_not_gained)` without expending a use; at level 4 the dice count MUST be absent and at level 5 present when `use` is requested. | SRD 5.2.1, "Level 5: Sear Undead", p. 37 | unhappy (boundary) |
| FR-067 | Sear Undead roll values outside 1 to 8, or a count of values different from FR-064's number of dice, MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error); when `use` is requested the use case MUST roll the dice through `dice.Roller.rollChecked`, once, whatever the number of targets (including none). | n/a (constructor invariant); doc/rules/architecture.md §3, §4; SRD 5.2.1, "Level 5: Sear Undead", p. 37 | unhappy (invalid input) |
| FR-131 | `UseClericChannelDivinity` for Turn Undead takes `searUndead: SearUndeadChoice` (`use` or `skip`, default `skip`). With `skip`, at any level, the use case MUST NOT call the `dice.Roller` and the effect data MUST carry no Sear Undead dice or damage. The decision to roll is the host's, as the SRD says "you can roll" (cases SU-09 to SU-12; D-17). | SRD 5.2.1, "Level 5: Sear Undead" ("you can roll"), p. 37 | unhappy (rule interaction) |

#### 4.8.1 Sear Undead dice cases (one test per row; source: SRD 5.2.1, "Level 5: Sear Undead", p. 37, and "Ability Modifiers", p. 6)
| Label | Cleric level | Wisdom modifier | Expected d8s | Rolls | Expected total | Path |
|---|---|---|---|---|---|---|
| SU-01 | 5 | +3 | 3 | 2, 4, 6 | 12 | happy |
| SU-02 | 5 | +1 | 1 | 8 | 8 | unhappy (boundary: modifier equals the minimum) |
| SU-03 | 5 | +0 | 1 | 1 | 1 | unhappy (boundary: raised to the minimum) |
| SU-04 | 5 | -2 | 1 | 5 | 5 | unhappy (boundary: negative raised to the minimum) |
| SU-05 | 12 | +5 | 5 | 8, 8, 8, 8, 8 | 40 | happy (maximum rolls) |
| SU-06 | 5 | +2 | 2 | 1, 1 | 2 | unhappy (boundary: minimum rolls) |
| SU-07 | 4 | +3 | none | n/a | none | unhappy (boundary: feature not yet gained) |
| SU-08 | 5 | +3 | 3 | 2, 4 | n/a | unhappy (invalid input): panic |
| SU-09 | 5, `skip` | +3 | none | no roll | no Sear Undead data | unhappy (rule interaction: host opts out; zero `dice.Roller` calls) |
| SU-10 | 4, `use` | +3 | none | no roll | rejected `sear_undead_not_gained`; no use expended | unhappy (boundary: not yet gained) |
| SU-11 | 5, `use`, no targets | +3 | 3 | 2, 4, 6 | 12 (applies to nobody; one roll) | unhappy (empty and minimal) |
| SU-12 | 5, `use`, three Undead targets | +3 | 3 | 2, 4, 6 | 12 as Radiant damage on each failed save; one roll | unhappy (rule interaction: one shared roll) |

### 4.10 Blessed Strikes (levels 7 and 14)
| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-075 | At Cleric level 7 the character MUST choose exactly one of Divine Strike or Potent Spellcasting; a missing, unknown or double choice MUST be rejected with `ChooseBlessedStrikesRejected` carrying `option_missing`, `unknown_option` or `both_options`, and a choice below level 7 with `below_level_7` (precedence: FR-128). | SRD 5.2.1, "Level 7: Blessed Strikes", p. 38 | happy |
| FR-076 | A second Blessed Strikes choice for a character that already holds one MUST be rejected with `ChooseBlessedStrikesRejected(already_chosen)` (D-7). | SRD 5.2.1, "Level 14: Improved Blessed Strikes" ("The option you chose"), p. 38 | unhappy (state error) |
| FR-077 | With Divine Strike, once on each of the Cleric's turns when it hits a creature with an attack roll using a weapon, the effect data MUST give an extra 1d8 damage of a type the Cleric chooses between Necrotic and Radiant, exposed as a dice expression (the host rolls it with the weapon's damage). | SRD 5.2.1, "Level 7: Blessed Strikes" (Divine Strike), p. 38 | happy |
| FR-078 | From Cleric level 14 the Divine Strike extra damage MUST be 2d8; at level 13 it MUST be 1d8. | SRD 5.2.1, "Level 14: Improved Blessed Strikes" (Divine Strike), p. 38 | unhappy (boundary) |
| FR-079 | Divine Strike MUST NOT apply to a second hit in the same turn (the turn state is supplied as `available` or `spent`), to an attack roll not using a weapon (a spell attack), or with a damage type other than Necrotic or Radiant; each MUST be reported as `ResolveBlessedStrikesNotApplicable` with reason `turn_spent`, `not_weapon_attack` or `damage_type_not_allowed` respectively (never returned as an error, never a rejection). | SRD 5.2.1, "Level 7: Blessed Strikes" (Divine Strike), p. 38 | unhappy (state error) |
| FR-080 | With Potent Spellcasting, the damage dealt with a Cleric cantrip MUST be increased by the Wisdom modifier (cases in 4.10.1). | SRD 5.2.1, "Level 7: Blessed Strikes" (Potent Spellcasting), p. 38 | happy |
| FR-081 | Potent Spellcasting MUST NOT increase damage from a cantrip without Cleric in its `Spell.classes` or from a spell that is not a cantrip; the command carries the cast `Spell` (content-spells.md) and the result is `ResolveBlessedStrikesNotApplicable(not_cleric_cantrip)`. | SRD 5.2.1, "Level 7: Blessed Strikes" (Potent Spellcasting, "any Cleric cantrip"), p. 38 | unhappy (rule interaction) |
| FR-082 | A Cleric holding Divine Strike MUST get no Potent Spellcasting effect, and a Cleric holding Potent Spellcasting MUST get no Divine Strike effect: `ResolveBlessedStrikesNotApplicable(other_option_chosen)`. | SRD 5.2.1, "Level 7: Blessed Strikes" ("one of the following options of your choice"), p. 38 | unhappy (rule interaction) |
| FR-083 | From Cleric level 14, with Potent Spellcasting, when the Cleric casts a Cleric cantrip and deals damage to a creature with it, the effect data MUST offer Temporary Hit Points equal to twice the Wisdom modifier to the Cleric or another creature within 60 feet; a modifier of 0 or less MUST yield 0 (D-3; cases in 4.10.1). | SRD 5.2.1, "Level 14: Improved Blessed Strikes" (Potent Spellcasting), p. 38; "Temporary Hit Points", p. 18 | unhappy (boundary) |
| FR-084 | The Temporary Hit Points of FR-083 MUST NOT be offered when the cantrip dealt no damage, when the recipient is farther than 60 feet (exactly 60 feet is accepted), below Cleric level 14, or to a holder of Divine Strike; the result is `ResolveBlessedStrikesNotApplicable` with reason `no_damage_dealt`, `recipient_out_of_range`, `below_level_14` or `other_option_chosen` (the bonus damage of FR-080 is unaffected). | SRD 5.2.1, "Level 14: Improved Blessed Strikes" (Potent Spellcasting), p. 38 | unhappy (boundary) |
| FR-085 | Below Cleric level 7 no Blessed Strikes effect of either option MUST be reported: `ResolveBlessedStrikesNotApplicable(below_level_7)`; a Cleric of level 7 or higher with no option chosen gets `ResolveBlessedStrikesNotApplicable(no_option_chosen)`. | SRD 5.2.1, "Level 7: Blessed Strikes", p. 38 | unhappy (boundary) |

#### 4.10.1 Potent Spellcasting cases (one test per row; source: SRD 5.2.1, "Level 7: Blessed Strikes" and "Level 14: Improved Blessed Strikes" (Potent Spellcasting), p. 38)
Cantrips named are test fixtures (Sacred Flame is on the Cleric list, p. 38); the table makes no claim about their damage.

| Label | Cleric level | Wisdom modifier | Cantrip | Damage dealt | Expected extra damage | Expected Temporary Hit Points | Path |
|---|---|---|---|---|---|---|---|
| PS-01 | 7 | +3 | Sacred Flame (on the Cleric list) | yes | +3 | none | happy |
| PS-02 | 7 | +0 | Sacred Flame | yes | +0 | none | unhappy (boundary: zero modifier) |
| PS-03 | 7 | -1 | Sacred Flame | yes | -1 (literal Wisdom modifier, D-3) | none | unhappy (boundary: negative modifier) |
| PS-04 | 7 | +3 | a cantrip not on the Cleric list | yes | none | none | unhappy (rule interaction) |
| PS-05 | 7 | +3 | a level 1 spell on the Cleric list | yes | none | none | unhappy (rule interaction) |
| PS-06 | 13 | +3 | Sacred Flame | yes | +3 | none | unhappy (boundary: before Improved Blessed Strikes) |
| PS-07 | 14 | +3 | Sacred Flame | yes | +3 | 6 | unhappy (boundary: first level with Temporary Hit Points) |
| PS-08 | 14 | +1 | Sacred Flame | yes | +1 | 2 | happy |
| PS-09 | 14 | +0 | Sacred Flame | yes | +0 | 0 | unhappy (boundary: zero modifier) |
| PS-10 | 14 | -2 | Sacred Flame | yes | -2 | 0 | unhappy (boundary: negative modifier gives 0) |
| PS-11 | 14 | +3 | a Cleric cantrip fixture that dealt no damage | no | none | none | unhappy (rule interaction) |
| PS-12 | 20 | +5 | Sacred Flame | yes | +5 | 10 | happy |

### 4.11 Divine Intervention and Greater Divine Intervention (levels 10 and 20)
| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-086 | Divine Intervention MUST be exposed as a Magic action in which the Cleric chooses a Cleric spell of level 5 or lower and casts it without expending a spell slot and without needing Material components; the result MUST say so, and MUST NOT expend any Spell Slot. `UseDivineInterventionApplied` carries the free-cast request (spell id, chosen `CastingTime`, no slot, no Material components) and is the published handoff: the host passes it to spellcasting.md `CastSpell` with `CastMode.featureGranted` and no `SpellSlotChoice` (casting-and-concentration.md FR-121 to FR-124), which applies the casting checks that do not concern a slot (casting-and-concentration.md FR-122) and derives `CastSource.featureGranted` (FR-106 there). Entitlement (the level, the once-per-Long-Rest state, the list and level checks) stays here, because casting-and-concentration.md leaves it to the feature owner (its FR-122; D-21). | SRD 5.2.1, "Level 10: Divine Intervention", p. 38; "Casting without Slots", p. 104 | happy |
| FR-087 | A chosen `Spell` (content-spells.md) that does not have Cleric in `Spell.classes`, and is not an unlocked extra grant (FR-094), MUST be rejected with `UseDivineInterventionRejected(not_cleric_spell)`. Membership is derived from `Spell.classes`, never from a host-supplied flag. | SRD 5.2.1, "Level 10: Divine Intervention" ("any Cleric spell"), p. 38 | unhappy (invalid input) |
| FR-088 | A chosen Cleric spell of level 5 MUST be accepted and of level 6 MUST be rejected with `UseDivineInterventionRejected(spell_level_too_high)` (cases in 4.11.1). | SRD 5.2.1, "Level 10: Divine Intervention" ("level 5 or lower"), p. 38 | unhappy (boundary) |
| FR-089 | A chosen spell whose chosen `CastingTime` is a `CastingTimeReaction` (content-spells.md FR-026) MUST be rejected with `UseDivineInterventionRejected(reaction_casting_time)`, and one with another casting time accepted. The command carries the chosen `CastingTime`, defaulting to the first of `Spell.castingTimes` as in casting-and-concentration.md FR-115. | SRD 5.2.1, "Level 10: Divine Intervention" ("doesn't require a Reaction to cast"), p. 38 | unhappy (rule interaction) |
| FR-090 | A cantrip with Cleric in `Spell.classes` MUST be accepted (level 0 is "level 5 or lower"). | SRD 5.2.1, "Level 10: Divine Intervention", p. 38 | unhappy (boundary) |
| FR-091 | After a use, Divine Intervention MUST be `DivineInterventionUnavailableUntilLongRest` until a Long Rest is finished; a second use before then MUST be rejected with `UseDivineInterventionRejected(not_available)` carrying the state; finishing a Short Rest MUST NOT make it available. | SRD 5.2.1, "Level 10: Divine Intervention" ("You can't use this feature again until you finish a Long Rest"), p. 38 | unhappy (state error) |
| FR-092 | `ApplyClericRest` with a Long Rest MUST move a Divine Intervention in the `DivineInterventionUnavailableUntilLongRest` state to `DivineInterventionAvailable`; a state already `DivineInterventionAvailable` MUST stay so; the `DivineInterventionRecharging` state is governed by FR-095. | SRD 5.2.1, "Level 10: Divine Intervention", p. 38 | happy |
| FR-093 | Divine Intervention MUST be rejected below Cleric level 10 with `UseDivineInterventionRejected(below_level_10)`. | SRD 5.2.1, "Level 10: Divine Intervention", p. 38 | unhappy (boundary) |
| FR-094 | `ClericRuleData` MUST carry the extra Divine Intervention grants as data (`ExtraDivineInterventionGrant`: spell id, minimum Cleric level, recharge dice); the SRD default is one grant: Wish, level 20, 2d4. At or above a grant's minimum level the Cleric MUST be allowed to choose that spell, exempt from the list-membership and level 5 checks of FR-087 and FR-088; the choice MUST roll the grant's recharge dice through `dice.Roller.rollChecked` and set the state to `DivineInterventionRecharging` with that many Long Rests. The rule logic names no spell id and no level literal. | SRD 5.2.1, "Level 20: Greater Divine Intervention", p. 38 | happy |
| FR-095 | After a grant use that rolled N (2 to 8 for the 2d4 default), the state is `DivineInterventionRecharging(longRestsRemaining: N)`; each finished Long Rest MUST count one down, Divine Intervention MUST be unavailable until the count reaches 0 and available at that Long Rest, and a Short Rest MUST NOT count (cases in 4.11.2). | SRD 5.2.1, "Level 20: Greater Divine Intervention" ("until you finish 2d4 Long Rests"), p. 38 | unhappy (boundary) |
| FR-096 | Choosing a granted spell (Wish by default) below the grant's minimum Cleric level MUST be rejected with `UseDivineInterventionRejected(not_cleric_spell)` (it is not on the Cleric list and is above level 5). | SRD 5.2.1, "Level 10: Divine Intervention", "Level 20: Greater Divine Intervention", p. 38 | unhappy (boundary) |
| FR-097 | The recharge roll values outside the die's faces (1 to 4 for 2d4) or a count different from the grant's dice MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error); the domain MUST NOT roll. | n/a (constructor invariant); doc/rules/architecture.md §3; SRD 5.2.1, "Level 20: Greater Divine Intervention", p. 38 | unhappy (invalid input) |

#### 4.11.1 Divine Intervention spell-choice cases (one test per row; source: SRD 5.2.1, "Level 10: Divine Intervention", p. 38, and the Cleric Spell List, pp. 38-40)
A "Reaction" row means the chosen `CastingTime` is a `CastingTimeReaction` (content-spells.md FR-026). Casting times are test fixtures, not claims about the named spells.

| Label | Cleric level | Spell | Casting time (fixture) | Expected | Path |
|---|---|---|---|---|---|
| DI-01 | 10 | Cure Wounds (level 1) | Magic action | accepted | happy |
| DI-02 | 10 | Sacred Flame (cantrip) | Magic action | accepted | unhappy (boundary: level 0) |
| DI-03 | 10 | Flame Strike (level 5) | Magic action | accepted | unhappy (boundary: level 5) |
| DI-04 | 10 | Heal (level 6) | Magic action | rejected `spell_level_too_high` | unhappy (boundary: level 6) |
| DI-05 | 10 | a level 5 Cleric spell | Reaction | rejected `reaction_casting_time` | unhappy (rule interaction) |
| DI-06 | 10 | a Cleric spell with a Bonus Action casting time | Bonus Action | accepted | unhappy (rule interaction) |
| DI-07 | 10 | a spell not on the Cleric list | Magic action | rejected `not_cleric_spell` | unhappy (invalid input) |
| DI-08 | 9 | Cure Wounds | Magic action | rejected `below_level_10` | unhappy (boundary) |
| DI-09 | 19 | Wish | Magic action | rejected `not_cleric_spell` (grant needs level 20) | unhappy (boundary) |
| DI-10 | 20 | Wish | Magic action | accepted; roll the grant dice (2d4) | happy |
| DI-11 | 20 | Cure Wounds | Magic action | accepted; no roll | happy |
| DI-12 | 10 | Cure Wounds, Divine Intervention already used | Magic action | rejected `not_available` | unhappy (state error) |
| DI-13 | 10 | Heal (level 6), Divine Intervention already used | Magic action | rejected `not_available` (state before spell checks, FR-128) | unhappy (rule interaction: several apply) |
| DI-14 | 9 | a spell not on the Cleric list, level 6 | Reaction | rejected `below_level_10` (first in the order) | unhappy (rule interaction: several apply) |

#### 4.11.2 Greater Divine Intervention recharge cases (one test per row; source: SRD 5.2.1, "Level 20: Greater Divine Intervention", p. 38)
| Label | 2d4 rolled | Event | Expected state | Path |
|---|---|---|---|---|
| GD-01 | 1, 1 (N = 2) | right after Wish | unavailable; 2 Long Rests to go | happy |
| GD-02 | 1, 1 (N = 2) | one Long Rest finished | unavailable; 1 to go | unhappy (boundary: one before) |
| GD-03 | 1, 1 (N = 2) | two Long Rests finished | available | unhappy (boundary: exactly N) |
| GD-04 | 4, 4 (N = 8) | seven Long Rests finished | unavailable; 1 to go | unhappy (boundary: maximum N, one before) |
| GD-05 | 4, 4 (N = 8) | eight Long Rests finished | available | unhappy (boundary: maximum N) |
| GD-06 | 2, 3 (N = 5) | three Short Rests finished | unavailable; 5 to go | unhappy (rule interaction) |
| GD-07 | 2, 3 (N = 5) | a use attempt after two Long Rests | rejected `not_available` (3 Long Rests to go) | unhappy (state error) |
| GD-08 | 5, 1 | n/a | panic (5 outside 1-4) | unhappy (invalid input) |

### 4.12 Life Domain
| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-098 | `LifeDomainSpells.grantedAt(SubclassDefinition? held, int clericLevel)` MUST return the always-prepared spells of a Life Domain Cleric by reading the `AlwaysPreparedSpells` fact of the held subclass's "Life Domain Spells" `FeatureEntry` (content-classes.md FR-055, FR-058; the SRD grants are table FA of content-classes.md: level 3 Aid, Bless, Cure Wounds, Lesser Restoration; level 5 Mass Healing Word, Revivify; level 7 Aura of Life, Death Ward; level 9 Greater Restoration, Mass Cure Wounds). This spec holds no copy of those grants (content-classes.md FR-098, shared-types-and-ownership.md FR-025). The caller passes the `SubclassDefinition` it already holds (found with `ClassDefinition.subclass(SubclassId)`, content-classes.md FR-074), or null; Life Domain membership is the `SubclassId` equality `held.id == LifeDomain.subclassId` (the SRD id `life_domain`, content-classes.md table SRC), and any other subclass or none gives the empty set (cases in 4.12.1). | SRD 5.2.1, "Level 3: Life Domain Spells", p. 40 | happy |
| FR-099 | The always-prepared set at Cleric level L MUST be the union of the grants whose `level` is at or below L, in grant order: empty below the first grant level (3 for the SRD data) and for a Cleric without the Life Domain subclass. | SRD 5.2.1, "Level 3: Life Domain Spells" ("When you reach a Cleric level specified in the Life Domain Spells table"), p. 40 | unhappy (boundary) |
| FR-100 | The Life Domain Spells MUST be handed to spellcasting.md as always-prepared spells of the Cleric class, so that they do not count against the number of Prepared Spells and otherwise count as Cleric spells. | SRD 5.2.1, "Level 1: Spellcasting" ("If another Cleric feature gives you spells you always have prepared"), p. 37; "Level 3: Life Domain Spells", p. 40 | happy |
| FR-101 | With Disciple of Life, when a spell the Cleric casts with a spell slot restores Hit Points to a creature, that creature MUST be reported as regaining additional Hit Points equal to 2 plus the slot's level, on the turn the spell is cast (cases in 4.12.2). | SRD 5.2.1, "Level 3: Disciple of Life", p. 40 | happy |
| FR-102 | The slot level of FR-101 MUST be the level of the slot actually spent, so a spell cast at a higher level gives a larger bonus. | SRD 5.2.1, "Level 3: Disciple of Life" ("the spell slot's level"), p. 40 | unhappy (boundary) |
| FR-103 | Disciple of Life MUST give each creature the spell restores Hit Points to its own bonus, whatever the number of creatures. | SRD 5.2.1, "Level 3: Disciple of Life" ("restores Hit Points to a creature, that creature regains"), p. 40 | unhappy (rule interaction) |
| FR-104 | Disciple of Life MUST give no bonus when the spell was cast without a spell slot (a Ritual, a cantrip, or a spell cast through Divine Intervention), for healing from a Channel Divinity effect (Divine Spark or Preserve Life), below Cleric level 3, or to a Cleric without the Life Domain subclass (no subclass, or any other subclass id) (D-8; cases DL-13 to DL-15). | SRD 5.2.1, "Level 3: Disciple of Life" ("cast with a spell slot"), p. 40 | unhappy (rule interaction) |
| FR-105 | Preserve Life MUST be exposed as a Magic action that expends a use of Channel Divinity and yields a pool of Hit Points equal to 5 times the Cleric level, to be divided among chosen Bloodied creatures within 30 feet of the Cleric, who can include the Cleric (cases in 4.12.3). | SRD 5.2.1, "Level 3: Preserve Life", p. 40 | happy |
| FR-106 | A target that is not Bloodied (more than half its Hit Point maximum remaining) MUST be rejected with `UseClericChannelDivinityRejected(target_not_bloodied)` naming it; a target at exactly half MUST be accepted. Bloodied MUST be decided by calling derived-stats.md `isBloodied(currentHitPoints, HitPointMaximum)` (shared-types-and-ownership.md FR-049) on the plain values the command carries for each target; this spec defines no Bloodied predicate of its own (D-22). | SRD 5.2.1, "Level 3: Preserve Life" ("Bloodied creatures"), p. 40; "Bloodied", p. 177 | unhappy (boundary) |
| FR-107 | A target farther than 30 feet MUST be rejected with `UseClericChannelDivinityRejected(target_out_of_range)`; a target at exactly 30 feet, or the Cleric itself, MUST be accepted. | SRD 5.2.1, "Level 3: Preserve Life" ("within 30 feet of yourself (which can include you)"), p. 40 | unhappy (boundary) |
| FR-108 | The sum of the Hit Points allocated to the targets MUST NOT exceed the pool; a sum exactly equal to the pool MUST be accepted, a sum above it rejected with `UseClericChannelDivinityRejected(allocation_exceeds_pool)` carrying the pool and the sum, and a sum below it accepted (the unspent Hit Points are lost, D-9). | SRD 5.2.1, "Level 3: Preserve Life" ("divide those Hit Points among them"), p. 40 | unhappy (boundary) |
| FR-109 | An allocation that would restore a target to more than half its Hit Point maximum MUST be rejected with `UseClericChannelDivinityRejected(allocation_exceeds_room)` carrying the target's room, where room is the whole-number half of its Hit Point maximum rounded down, minus its current Hit Points (D-9). | SRD 5.2.1, "Level 3: Preserve Life" ("no more than half its Hit Point maximum"), p. 40 | unhappy (boundary) |
| FR-110 | A negative allocation MUST be rejected with `UseClericChannelDivinityRejected(negative_allocation)` and the same target named twice with `duplicate_target`; an allocation of 0 to a target MUST be accepted. | SRD 5.2.1, "Level 3: Preserve Life", p. 40; contradictory data | unhappy (invalid input) |
| FR-111 | Preserve Life with no chosen target MUST be accepted, expend one use of Channel Divinity and report an empty allocation (D-6). | SRD 5.2.1, "Level 3: Preserve Life", p. 40 | unhappy (empty and minimal) |
| FR-112 | With Blessed Healer, immediately after the Cleric casts a spell with a spell slot that restores Hit Points to one or more creatures other than itself, the Cleric MUST be reported as regaining Hit Points equal to 2 plus the slot's level, once per cast whatever the number of creatures healed (cases in 4.12.2). | SRD 5.2.1, "Level 6: Blessed Healer", p. 40 | happy |
| FR-113 | Blessed Healer MUST give nothing when the only creature healed is the Cleric, when the spell was cast without a spell slot, below Cleric level 6, or to a Cleric without the Life Domain subclass (no subclass, or any other subclass id; cases DL-13 to DL-15). | SRD 5.2.1, "Level 6: Blessed Healer" ("creatures other than yourself", "with a spell slot"), p. 40 | unhappy (rule interaction) |
| FR-114 | For a Cleric with the Life Domain subclass, from Cleric level 17, when healing dice of a spell or of Channel Divinity would be rolled, the result MUST be the highest number possible on each die instead of a roll (2d6 gives 12); below level 17, and for a Cleric without the Life Domain subclass (no subclass, or any other subclass id) at any level, the dice MUST be rolled (cases SH-09, SH-10). | SRD 5.2.1, "Level 17: Supreme Healing", p. 40 | happy |
| FR-115 | Supreme Healing MUST apply to Divine Spark in healing mode (the d8s count as 8 each, then the Wisdom modifier is added; `UseClericChannelDivinity` MUST NOT call the `dice.Roller` in that case, FR-055) and MUST NOT apply to damage dice, including Divine Spark in damage mode (cases in 4.12.4). | SRD 5.2.1, "Level 17: Supreme Healing" ("to restore Hit Points"), p. 40; "Level 2: Channel Divinity" (Divine Spark), p. 37 | unhappy (rule interaction) |
| FR-116 | A healing dice expression with fewer than 1 die or fewer than 2 sides MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error). | n/a (constructor invariant); SRD 5.2.1, "Level 17: Supreme Healing", p. 40 | unhappy (invalid input) |

#### 4.12.1 Life Domain Spells cases (one test per row; source: SRD 5.2.1, "Level 3: Life Domain Spells", p. 40)
The grant rows are those of content-classes.md table FA (levels 3, 5, 7 and 9), read from the bundled subclass or a fixture `SubclassDefinition`; this table repeats no spell id.

| Label | Cleric level | Subclass | Expected always-prepared spells | Path |
|---|---|---|---|---|
| LS-01 | 2 | Life Domain (not yet available) | none | unhappy (boundary: one below level 3) |
| LS-02 | 3 | Life Domain | the level 3 grant | happy |
| LS-03 | 4 | Life Domain | the level 3 grant | unhappy (boundary: no new row) |
| LS-04 | 5 | Life Domain | the grants of levels 3 and 5 | happy |
| LS-05 | 7 | Life Domain | the grants of levels 3, 5 and 7 | happy |
| LS-06 | 9 | Life Domain | the grants of levels 3, 5, 7 and 9 | happy |
| LS-07 | 20 | Life Domain | the grants of levels 3, 5, 7 and 9 | unhappy (boundary: maximum level) |
| LS-08 | 9 | none chosen (null) | none | unhappy (state error) |
| LS-09 | 9 | a subclass other than Life Domain (fixture id) | none | unhappy (rule interaction) |
| LS-10 | 5 | a fixture `SubclassDefinition` with id `life_domain` whose grant rows are different from the SRD | the fixture's rows at or below 5 | happy (proves the grants are read, not held here) |
| LS-11 | 3 | Life Domain definition with no `AlwaysPreparedSpells` fact on any entry | none | unhappy (missing content: no fact) |

#### 4.12.2 Disciple of Life and Blessed Healer cases (one test per row; source: SRD 5.2.1, "Level 3: Disciple of Life" and "Level 6: Blessed Healer", p. 40)
Unless a row says otherwise the Cleric holds the Life Domain subclass.

| Label | Cleric level | Slot spent | Creatures healed | Cast with slot | Expected Disciple of Life (each creature) | Expected Blessed Healer (Cleric) | Path |
|---|---|---|---|---|---|---|---|
| DL-01 | 3 | level 1 | one other creature | yes | +3 | none (level 6 needed) | happy |
| DL-02 | 6 | level 1 | one other creature | yes | +3 | 3 | happy |
| DL-03 | 6 | level 3 (Cure Wounds cast at level 3) | one other creature | yes | +5 | 5 | unhappy (boundary: higher slot) |
| DL-04 | 17 | level 9 | one other creature | yes | +11 | 11 | unhappy (boundary: highest slot level) |
| DL-05 | 6 | level 3 | three other creatures | yes | +5 to each of three | 5 once | unhappy (rule interaction) |
| DL-06 | 6 | level 2 | the Cleric only | yes | +4 to the Cleric | none | unhappy (rule interaction) |
| DL-07 | 6 | level 2 | the Cleric and one other creature | yes | +4 to each | 4 once | unhappy (rule interaction) |
| DL-08 | 6 | none (Ritual) | one other creature | no | none | none | unhappy (rule interaction) |
| DL-09 | 6 | none (Divine Intervention) | one other creature | no | none | none | unhappy (rule interaction) |
| DL-10 | 6 | none (Preserve Life) | Bloodied creatures | no | none | none | unhappy (rule interaction) |
| DL-11 | 2 | level 1 | one other creature | yes | none (level 3 needed) | none | unhappy (boundary) |
| DL-12 | 5 | level 1 | one other creature | yes | +3 | none (level 6 needed) | unhappy (boundary: one below level 6) |
| DL-13 | 3 | level 1 | one other creature | yes, no subclass chosen | none (no Life Domain) | none | unhappy (state error) |
| DL-14 | 6 | level 3 | one other creature | yes, a subclass other than Life Domain (fixture id) | none | none | unhappy (rule interaction) |
| DL-15 | 6 | level 1 | one other creature | yes, no subclass chosen | none | none | unhappy (state error) |

#### 4.12.3 Preserve Life cases (one test per row; source: SRD 5.2.1, "Level 3: Preserve Life", p. 40, and "Bloodied", p. 177)
"Room" is half the maximum rounded down, minus the current Hit Points (FR-109).

| Label | Cleric level | Pool | Target (current / maximum) | Allocation | Expected | Path |
|---|---|---|---|---|---|---|
| PL-01 | 3 | 15 | 4 / 20 | 6 | accepted (room 6) | happy |
| PL-02 | 3 | 15 | 4 / 20 | 7 | rejected `allocation_exceeds_room` (room 6) | unhappy (boundary: one above room) |
| PL-03 | 3 | 15 | 4 / 20 | 5 | accepted | unhappy (boundary: one below room) |
| PL-04 | 3 | 15 | 10 / 20 | 1 | target accepted (FR-106, Bloodied at exactly half); allocation of 1 rejected `allocation_exceeds_room` (FR-109, room 0) | unhappy (boundary: exactly half) |
| PL-05 | 3 | 15 | 10 / 20 | 0 | accepted (0 to a target) | unhappy (boundary) |
| PL-06 | 3 | 15 | 11 / 20 | 1 | rejected `target_not_bloodied` | unhappy (boundary: one above half) |
| PL-07 | 3 | 15 | 10 / 21 | 1 | target accepted; allocation of 1 rejected `allocation_exceeds_room` (room 0 since half of 21 rounds down to 10) | unhappy (boundary: odd maximum) |
| PL-08 | 3 | 15 | 0 / 21 | 10 | accepted (room 10) | unhappy (boundary: zero Hit Points) |
| PL-09 | 3 | 15 | two targets, 0 / 40 and 0 / 40 | 8 and 7 | accepted (sum 15 equals the pool) | unhappy (boundary: sum equals the pool) |
| PL-10 | 3 | 15 | two targets, 0 / 40 and 0 / 40 | 8 and 8 | rejected `allocation_exceeds_pool` (sum 16 above the pool) | unhappy (boundary: one above the pool) |
| PL-11 | 3 | 15 | one target, 0 / 40 | 14 | accepted (unspent Hit Point lost) | unhappy (boundary: one below the pool) |
| PL-12 | 20 | 100 | five targets, each 0 / 80 | 20 each | accepted (sum 100) | happy (maximum level) |
| PL-13 | 3 | 15 | the Cleric, 2 / 30 | 13 | accepted (room 13; the Cleric is a valid target) | happy |
| PL-14 | 3 | 15 | the same target twice | any | rejected `duplicate_target` | unhappy (invalid input) |
| PL-15 | 3 | 15 | 4 / 20, 35 feet away | 1 | rejected `target_out_of_range` | unhappy (boundary) |
| PL-16 | 3 | 15 | none | n/a | accepted; use expended; empty allocation | unhappy (empty and minimal) |

#### 4.12.4 Supreme Healing cases (one test per row; source: SRD 5.2.1, "Level 17: Supreme Healing", p. 40)
Unless a row says otherwise the Cleric holds the Life Domain subclass (FR-114).

| Label | Cleric level | Healing dice | Source of healing | Expected | Path |
|---|---|---|---|---|---|
| SH-01 | 17 | 2d6 | a spell | 12, no roll | happy |
| SH-02 | 17 | 1d4 | a spell | 4, no roll | unhappy (boundary: one die) |
| SH-03 | 16 | 2d6 | a spell | rolled | unhappy (boundary: one below level 17) |
| SH-04 | 17 | 3d8 + Wisdom modifier +3 | Divine Spark, healing mode | 27, no roll | happy |
| SH-05 | 17 | 3d8 + Wisdom modifier +3 | Divine Spark, damage mode | rolled normally | unhappy (rule interaction) |
| SH-06 | 20 | 4d8 | Divine Spark, healing mode | 32 plus the Wisdom modifier | unhappy (boundary: maximum level) |
| SH-07 | 17 | 0d6 | a spell | n/a | unhappy (invalid input): panic |
| SH-08 | 17 | 1d1 | a spell | n/a | unhappy (invalid input): panic |
| SH-09 | 17 | 2d6 | a spell, no subclass chosen | rolled | unhappy (state error: not Life Domain) |
| SH-10 | 17 | 3d8 + Wisdom modifier +3 | Divine Spark, healing mode, a subclass other than Life Domain (fixture id) | rolled; the `dice.Roller` is called | unhappy (rule interaction) |
| SH-11 | 17 | 3d8 + Wisdom modifier +3 | Divine Spark, healing mode, Life Domain | 27; zero `dice.Roller` calls (FR-055) | unhappy (rule interaction: no roll) |
| SH-12 | 17 | 3d8 + Wisdom modifier +3 | Divine Spark, damage mode, Life Domain | rolled through the `dice.Roller` (3 d8s) | unhappy (rule interaction: damage is not healing) |

### 4.13 Cross-cutting behaviour and operation contracts (4.14.1 to 4.14.4 are its case tables)
| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-117 | Every level-gated Cleric feature (the uses and dice of FR-038 and FR-049, Sear Undead, Blessed Strikes, Divine Intervention, the Preserve Life pool, the subclass features) MUST use the Cleric level, not total character level; the Proficiency Bonus and thus the Spell Save DC MUST use total character level. A level 3 Cleric / level 5 Fighter has a Preserve Life pool of 15, one Divine Spark die, and Proficiency Bonus +3. | SRD 5.2.1, "Preserve Life" ("five times your Cleric level"), p. 40; "Proficiency Bonus" under Multiclassing, p. 25; "Cleric Class Features", pp. 36-38 | unhappy (rule interaction) |
| FR-118 | Every Cleric state transition MUST return a new state and the events it produced and MUST NOT mutate the input state; collections exposed by the state, the progression, the spell list and effect data MUST be a defensive copy. | doc/rules/architecture.md §3 | unhappy (immutability) |
| FR-119 | With a fixed `dice.Roller`, each rolling use case (Divine Spark, Turn Undead with `SearUndeadChoice.use`, a Divine Intervention that chooses a granted spell such as Wish) MUST return the same result every run, including minimum rolls and maximum rolls. | SRD 5.2.1, "Level 2: Channel Divinity", "Level 5: Sear Undead", "Level 20: Greater Divine Intervention"; doc/rules/testing.md §4 | unhappy (determinism) |
| FR-122 | A Cleric operation whose inputs carry the `Character` (every operation except `ApplyClericRest`) applied to a character with no Cleric levels MUST return its own `<Operation>NotACleric` variant (for example `UseClericChannelDivinityNotACleric`), not panic. `AdvanceCleric` is called after level-advancement-and-multiclassing.md has recorded the new level, so the `Character` it receives already holds the new Cleric level (the previous level is passed separately); `AdvanceClericNotACleric` is returned only when it holds no Cleric level at all. | n/a (package design); doc/rules/architecture.md §4, §6; shared-types-and-ownership.md FR-017 | unhappy (state error) |
| FR-123 | Each of the two use cases (`UseClericChannelDivinity`, `UseDivineIntervention`) MUST produce identical results with the no-op `telemetry.Telemetry` and with a recording one. Domain functions take no `telemetry.Telemetry`; they are asserted silent by the architecture tests, with no telemetry test of their own. | doc/rules/observability.md §9 | unhappy (determinism) |
| FR-124 | Applies to the two use cases; the domain functions are silent. They emit the shared class-feature signals owned by character-aggregate.md FR-118 (counter `srd5e.character.feature_used`, histogram `.feature_used.duration`, counter `.feature_used.errors`; decisions FR-063) with `srd5e.class` = `cleric` and `srd5e.feature`, plus one span each (section 7). Rejections of every kind (a rule or state rejection such as `no_uses_remaining`, a rejection of invalid host input such as a bad target, and `NotACleric`) MUST be recorded with `srd5e.outcome` = `failure` and the reason in `srd5e.character.rejection`, with no error status, no `error.type`, no `.errors` increment and no log (as character-aggregate.md FR-072). Only a failure that leaves srd5e as a returned `*srderr.PortError` (a `dice.Roller` failure) MUST be recorded as an error, once: span error status and `error.type` (the same constant, as character-aggregate.md FR-088), one `feature_used` increment with `srd5e.outcome` = `failure` and no `error.type`, and one `.errors` increment with `error.type` (`srderr.ErrorType(err)`, shared-types-and-ownership.md FR-041); not logged. A programmer error (panic) propagates and the span records it as it propagates, with no `error.type` and no counter increment (character-aggregate.md FR-088; observability.md §7). Every call also records its latency in `.feature_used.duration` through the `Meter` timing operation (observability.md §5). | doc/rules/observability.md §6, §7, §9; shared-types-and-ownership.md FR-041, FR-042, FR-063 | unhappy (telemetry on failure) |
| FR-125 | Each successful transition MUST produce the matching domain events: `ClericOptionChosen` (a Divine Order or Blessed Strikes choice; the subclass choice is `SubclassChosen` of level-advancement-and-multiclassing.md), `ClericChannelDivinityUsed`, `ClericChannelDivinityRegained`, `DivineInterventionUsed`, `DivineInterventionRecharged`, and `ClericAdvanced`. `ClericAdvanced` MUST be emitted on every successful `AdvanceCleric`, exactly once, whatever the level, carrying the new level, the features gained, the pending choices, the Channel Divinity uses gained and the always-prepared spells granted (a level with none of those has empty collections; cases in 4.14.3; D-18). A rejection MUST produce none. | doc/rules/architecture.md §3 (domain events) | happy |
| FR-126 | No Cleric operation loads content: every operation, domain function or use case, is synchronous and MUST NOT call a `ContentSource`. It takes the facts it needs as values (`Spell`, `ProgressionRow`, levels, `ClericChoices`, `ClericResources`, the skill choices of the `ClassDefinition`). Only `UseClericChannelDivinity` and `UseDivineIntervention` are use cases (they need the `dice.Roller` and emit telemetry); every other operation is a pure domain function (shared-types-and-ownership.md FR-023). | doc/rules/architecture.md §4 ("Do not make a use case context-aware speculatively"); shared-types-and-ownership.md FR-023; spellcasting.md (facts in the command) | happy |
| FR-128 | When several rejection conditions hold, the reason MUST be the first in the order of table 4.14.1 for that use case (state before input; the target list is checked in command order, the first failing target is reported); the reason values are exactly the values in that table, and these are the values of the `srd5e.character.rejection` attribute (section 7; character-aggregate.md FR-072). | n/a (determinism, as character-aggregate.md FR-043; observability.md §2) | unhappy (rule interaction) |
| FR-130 | A `*srderr.PortError` (port `dice`) from the roller (reason `unavailable`, or `invalidResult` raised by `rollChecked`) MUST propagate unchanged from `UseClericChannelDivinity` and from `UseDivineIntervention`, MUST leave the input state unchanged (no use expended, no recharge started, no events), and MUST be recorded once (`error.type=dice_roller_error`, the constant errors-and-results.md defines for `*srderr.PortError` (port `dice`), on `.feature_used.errors`; FR-124). Roll values outside the die's faces from a roller that returned the right expression MUST surface as the panic of FR-054, FR-067 or FR-097, which propagates with no `error.type` and no counter increment; the span records it as it propagates (FR-124; character-aggregate.md FR-088; D-19). Cases in 4.14.4. | doc/rules/testing.md §4 (collaborator failure and invalid data); dice-and-rolls.md FR-049 to FR-052; doc/rules/observability.md §7 | unhappy (collaborator failure) |
| FR-132 | Divine Spark takes a host-supplied `visible` fact for its target (the Cleric "can see" it); a target that is not visible MUST be rejected with `UseClericChannelDivinityRejected(target_not_visible)` and MUST NOT expend a use (cases in 4.7.2). | SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark, "another creature you can see"), p. 37 | unhappy (invalid input) |
| FR-133 | `ClericRuleData` MUST hold the Cleric rule data that the content format keeps only as feature text (content-classes.md): the recommended spells (FR-027) and the extra Divine Intervention grants (FR-094). It holds no Life Domain data: the Life Domain Spells and the Life Domain `SubclassId` are structured content read through `LifeDomainSpells` (FR-098; content-classes.md FR-055, FR-058, FR-098). Its default MUST be the SRD 5.2.1 values; a host MAY pass another instance (house rules). Rule logic reads it and holds no spell-id or level literal of its own; ids are typed (`SpellId`, lowercase snake_case, shared-types-and-ownership.md FR-005, FR-007). A test MUST check that the remaining default ids (Guidance, Sacred Flame, Thaumaturgy, Bless, Cure Wounds, Guiding Bolt, Shield of Faith and the grant's Wish) exist in the bundled content (content-spells.md dependent). | n/a (doc/rules/architecture.md §3: new content and house rules arrive as data); SRD 5.2.1, "Level 1: Spellcasting", "Level 20: Greater Divine Intervention", pp. 36-38 | happy |
| FR-134 | `LifeDomainSpells.grantedAt(held, L)` (FR-098) makes the handoff of FR-100 complete at level 3 when the caller reacts to the `SubclassChosen` event of level-advancement-and-multiclassing.md. `AdvanceCleric` takes the held optional `SubclassDefinition` and MUST return the spells newly granted at the new level, those of `grantedAt(held, L)` not in `grantedAt(held, L - 1)` (levels 5, 7 and 9 for the SRD data; none for a Cleric without the Life Domain). | SRD 5.2.1, "Level 3: Life Domain Spells", p. 40 | happy |
| FR-135 | The Cleric's state and creation choices follow shared-types-and-ownership.md FR-022 and FR-024: `ClericChoices` (Divine Order, Blessed Strikes option) and `ClericResources` (Channel Divinity uses, Divine Intervention state) are `FeatureState` implementations held under the typed keys `ClericFeatureKeys.choices` and `ClericFeatureKeys.resources` in the aggregate's `FeatureStates` (character-aggregate.md FR-092, FR-093); the Cleric-specific creation choices travel in `CreateCharacter` as a typed `ClericCreationChoices` (a `ClassCreationChoices`, character-aggregate.md FR-094; what it seeds and names is FR-140); the class skills and the Starting Equipment option are `CreateCharacterCommand` fields owned by character-creation.md (FR-015 to FR-020, FR-070). The aggregate stores no Cleric-specific field. | n/a (package design); shared-types-and-ownership.md FR-022, FR-023, FR-024; character-aggregate.md FR-094 | happy |
| FR-136 | `ClericEvent` is a `closed` family and every Cleric event is a variant of it. `ClericEvent` implements `CharacterEvent`, the open `interface` of character-aggregate.md FR-085 (which implements `DomainEvent` and declares `characterId`), so each Cleric event carries the `CharacterId` of the character it concerns. `ClericEvent` sits outside the closed `CharacterRecordEvent` family, as shared-types-and-ownership.md FR-019 prescribes. Each Cleric event has exactly one defining spec (this one); `SubclassChosen`, `SubclassRejected` and `RestApplied` are referenced, never defined, here. Both Channel Divinity classes prefix their names (`UseClericChannelDivinity`, `ClericChannelDivinityUses`, `ClericChannelDivinityUsed`, `ClericChannelDivinityRegained` here; `UsePaladinChannelDivinity`, `PaladinChannelDivinityUses`, `PaladinChannelDivinityUsed` in class-paladin.md, FR-124 there), and so are the state and option variants (`DivineOrderProtector`, `DivineOrderThaumaturge`, `BlessedStrikesDivineStrike`, `BlessedStrikesPotentSpellcasting`, `DivineInterventionAvailable`, `DivineInterventionUnavailableUntilLongRest`, `DivineInterventionRecharging`), so the one `character` package API has no collision (shared-types-and-ownership.md FR-017, FR-021). Test: `ClericEvent` is a `CharacterEvent` (and so a `DomainEvent`); every Cleric event is a `ClericEvent`, carries the id and is switchable exhaustively over `ClericEvent`; a test-only `CharacterEvent` implementor compiles outside the family; and the package API scan finds none of `Available`, `Unavailable`, `Recharging`, `Protector`, `Thaumaturge`, `DivineStrike` or `PotentSpellcasting` exported, and none of the Paladin's names. | n/a (package design); shared-types-and-ownership.md FR-017, FR-019, FR-020, FR-021; character-aggregate.md FR-085 | happy |
| FR-137 | The Thaumaturge check bonus MUST be exposed as a `Modifier` (a plain signed integer, no kind tag) for the other-modifiers list of `RollModifiers`; the Wisdom modifier input is an `AbilityModifier` (shared-types-and-ownership.md FR-001, FR-003). | n/a (package design); shared-types-and-ownership.md FR-001, FR-003 | happy |
| FR-138 | When a Cleric operation needs `ClericChoices` or `ClericResources` and the `FeatureStates` hold none (a multiclass Cleric whose character never went through `CreateCharacter` with Cleric as the initial class, so `ClassCreationChoices` never seeded them), the first `AdvanceCleric` to level 1 MUST supply the initial states: `ClericChoices` empty (no Divine Order, no Blessed Strikes option), `ClericResources` with Channel Divinity 0 of 0 and Divine Intervention `DivineInterventionAvailable`. The ongoing transitions then follow FR-044. | n/a (package design); shared-types-and-ownership.md FR-022, FR-024 | happy |
| FR-139 | The two keys MUST be named `cleric_choices` and `cleric_resources` (`ClericFeatureKeys`), matching `[a-z][a-z0-9_]*`; because same-named keys are equal across classes (character-aggregate.md FR-093), a test MUST assert that no other class spec's key name equals either. | n/a (package design); character-aggregate.md FR-093 | unhappy (name collision) |
| FR-140 | `ClericCreationChoices` MUST implement both pure methods of `ClassCreationChoices` (character-aggregate.md FR-094), emitting no event. It is built only by `NewClericCreationChoices(order names, extra cantrip optional `Spell`, known cantrips)`, which applies the `ChooseDivineOrder` rules (FR-029 to FR-033; reasons and order FR-128, without `already_chosen`) and returns `(ClericCreationChoices, error)` where the error is a typed rejection reason. (a) `seed(FeatureStates)` MUST return the given states with two entries added and every other entry kept, the input unchanged: `ClericChoices` under `ClericFeatureKeys.choices`, holding the validated Divine Order (for a Thaumaturge with the extra cantrip's `SpellId`) and no Blessed Strikes option; and the initial `ClericResources` for Cleric level 1 under `ClericFeatureKeys.resources` (Channel Divinity 0 of 0, `DivineInterventionAvailable`, as FR-138). A `FeatureStates` that already holds either key is a programmer error (panic; `CreateCharacter` starts from empty states, character-creation.md FR-102). (b) `contentKeys()` MUST return an unmodifiable list holding the `ContentKey` of the extra cantrip for a Thaumaturge and none for a Protector, so `CreateCharacter` looks it up and reports an absent one in `CreateCharacterContentMissing` (character-creation.md FR-006, FR-103). A rejection at creation is reported by the `New` constructor (returns an error), before the command exists, with a `ClericCreationChoicesRejected` carrying the `ChooseDivineOrderRejected` reason; it is never a `CreateCharacterRejected` problem, because `CreateCharacter` interprets no class choice beyond the seeding and content-key calls of character-creation.md FR-102 and FR-103 and compares no class id: the class spec validates its choices when it builds them. The `seed` and `contentKeys` of the same choices are therefore always consistent. | n/a (package design); shared-types-and-ownership.md FR-022, FR-024; character-aggregate.md FR-094; character-creation.md FR-102, FR-103 | happy |
| FR-141 | A Thaumaturge MUST supply an extra-cantrip allowance of 1 for the Cleric class to spellcasting.md (`ClassSpellcasting.extraCantrips`, FR-096 there), and a Protector 0: `ChooseDivineOrderChosen` and `ClericChoices` expose `extraCantrips` (1 or 0) and the host passes it as the Cleric's `extraCantrips` to `CreateSpellcasting` (spellcasting.md FR-092). The cap on cantrips known (Table F count plus the allowance, FR-052 there), the rule that the extra cantrip is a cantrip of the Cleric list (FR-096 there) and the headroom reported on a level gain (`cantripsToChoose`, FR-047 there) belong to spellcasting.md; this spec states none of them. | SRD 5.2.1, "Level 1: Divine Order" (Thaumaturge), p. 37; "Cleric Features", p. 36 | happy |

#### 4.14.1 Rejection reasons and their order (one test per row of 4.14.2)
First listed is first reported (FR-128). Each reason is a value of the closed rejection type named in the first column and the snake_case `srd5e.character.rejection` attribute value shown. Class skill and equipment reasons (`classSkillCountWrong`, `classSkillNotOffered`, `classSkillRepeated`, `classEquipmentOptionUnknown`), `SubclassRejected` and level validation are owned by character-creation.md and level-advancement-and-multiclassing.md; `AdvanceCleric` has no rejection.

| Result type (use case) | Reasons in order |
|---|---|
| `ChooseDivineOrderRejected` (`ChooseDivineOrder`) | `already_chosen`, `order_missing`, `both_orders`, `order_unknown`, `extra_cantrip_missing`, `extra_cantrip_not_cleric_cantrip`, `extra_cantrip_already_known` (a Protector choice ignores any extra cantrip supplied) |
| `ChooseBlessedStrikesRejected` (`ChooseBlessedStrikes`) | `below_level_7`, `already_chosen`, `option_missing`, `both_options`, `unknown_option` |
| `UseClericChannelDivinityRejected` (`UseClericChannelDivinity`) | `below_level_2`, `effect_not_available`, `no_uses_remaining`, then by effect. Divine Spark: `target_is_self`, `target_not_visible`, `target_out_of_range`, `damage_type_not_allowed`. Turn Undead: `sear_undead_not_gained`, then per target in command order `target_not_undead`, `target_out_of_range`. Preserve Life: `duplicate_target`, then per target in command order `target_out_of_range`, `target_not_bloodied`, `negative_allocation`, `allocation_exceeds_room`, and last `allocation_exceeds_pool` |
| `UseDivineInterventionRejected` (`UseDivineIntervention`) | `below_level_10`, `not_available`, `not_cleric_spell`, `spell_level_too_high`, `reaction_casting_time` |
| `ResolveBlessedStrikesNotApplicable` (`ResolveBlessedStrikes`), by path | all paths: `below_level_7`, `no_option_chosen`, `other_option_chosen`; Divine Strike: `turn_spent`, `not_weapon_attack`, `damage_type_not_allowed`; Potent Spellcasting damage: `not_cleric_cantrip`; Temporary Hit Points: `not_cleric_cantrip`, `no_damage_dealt`, `below_level_14`, `recipient_out_of_range` |

`NotACleric` is not a reason of table 4.14.1 but a variant of each operation's own result (FR-122), recorded with `srd5e.character.rejection` = `not_a_cleric`. There is no `ContentMissing` variant (FR-126).

#### 4.14.2 Rejection order cases (one test per row; source: n/a, package design, FR-128; the SRD rule behind each reason is cited on its FR; RJ-13, RJ-15 and RJ-18 were subclass, class skill and level rows owned by other specs)
| Label | Use case and facts | Expected reason | Path |
|---|---|---|---|
| RJ-01 | `UseClericChannelDivinity`, Cleric level 1, no uses, Turn Undead on a non-Undead 35 feet away | `below_level_2` | unhappy (rule interaction: several apply) |
| RJ-02 | level 2, no uses, Preserve Life without Life Domain | `effect_not_available` | unhappy (rule interaction: several apply) |
| RJ-03 | level 2, no uses, Turn Undead on a non-Undead 35 feet away | `no_uses_remaining` | unhappy (rule interaction: several apply) |
| RJ-04 | level 2, 1 use, Turn Undead on a non-Undead 35 feet away | `target_not_undead` | unhappy (rule interaction: several apply) |
| RJ-05 | level 2, 1 use, Turn Undead on an Undead 31 feet away | `target_out_of_range` | unhappy (boundary) |
| RJ-06 | level 4, Turn Undead with `use` Sear Undead, a non-Undead target | `sear_undead_not_gained` | unhappy (rule interaction: several apply) |
| RJ-07 | level 3 Life Domain, Preserve Life, one target named twice, one 35 feet away | `duplicate_target` | unhappy (rule interaction: several apply) |
| RJ-08 | Preserve Life, a target not Bloodied and 35 feet away | `target_out_of_range` | unhappy (rule interaction: several apply) |
| RJ-09 | Preserve Life, a target not Bloodied with allocation -1 | `target_not_bloodied` | unhappy (rule interaction: several apply) |
| RJ-10 | Preserve Life, a Bloodied target with allocation -1 | `negative_allocation` | unhappy (invalid input) |
| RJ-11 | Preserve Life, pool 15, one Bloodied target of room 6, allocation 16 | `allocation_exceeds_room` (before the pool) | unhappy (rule interaction: several apply) |
| RJ-12 | `ChooseDivineOrder`, an order already held, both orders named | `already_chosen` | unhappy (rule interaction: several apply) |
| RJ-14 | `ChooseBlessedStrikes`, level 6, both options named | `below_level_7` | unhappy (rule interaction: several apply) |
| RJ-16 | `ResolveBlessedStrikes`, level 7, Potent Spellcasting held, a weapon hit | `other_option_chosen` | unhappy (rule interaction) |
| RJ-17 | `ResolveBlessedStrikes`, Divine Strike, turn spent, a spell attack | `turn_spent` | unhappy (rule interaction: several apply) |

#### 4.14.3 AdvanceCleric cases (one test per row; source: n/a for validation rows, package design; SRD 5.2.1, "Cleric Features", p. 36, and "Level 3: Life Domain Spells", p. 40, for payload rows)
Level-validation rows (AV-03 to AV-09) belong to level-advancement-and-multiclassing.md (this spec's withdrawn FR-129 moved there: FR-011, FR-024). Validation rows left: **AV-01** (previous 0, new 1): applied with `ClericAdvanced`; the `Character` passed already holds Cleric level 1 (FR-122) and the initial `ClericChoices` and `ClericResources` are supplied (FR-138); happy (boundary: first level). **AV-02** (19 to 20): applied; happy (boundary: last level).

Features gained are `FeatureId`s and pending choices are `ClericPendingChoice` values (FR-021); the names are display labels. Pending subclass and feat choices are not payload (level-advancement-and-multiclassing.md FR-016, FR-017) and the cantrip headroom is `cantripsToChoose` (spellcasting.md FR-047). "Life Domain spells" applies to a Life Domain Cleric only. Payload rows, each a successful advance from the level above (one `ClericAdvanced` event every time, FR-125):

| Label | New level | Features gained | Pending choices | Uses gained | Life Domain spells granted |
|---|---|---|---|---|---|
| AC-01 | 1 | Spellcasting, Divine Order | `divineOrder` | 0 | none |
| AC-02 | 2 | Channel Divinity | none | 2 (maximum 0 to 2) | none |
| AC-03 | 3 | Cleric Subclass | none | 0 | none (granted by `LifeDomainSpells.grantedAt` when the caller reacts to `SubclassChosen`, FR-134) |
| AC-04, AC-08, AC-12, AC-16 | 4, 8, 12, 16 | Ability Score Improvement | none | 0 | none |
| AC-05 | 5 | Sear Undead | none | 0 | the level 5 grant row (content-classes.md table FA) |
| AC-06 | 6 | Subclass feature | none | 1 (maximum 2 to 3) | none |
| AC-07 | 7 | Blessed Strikes | `blessedStrikesOption` | 0 | the level 7 grant row |
| AC-09 | 9 | none | none | 0 | the level 9 grant row |
| AC-10 | 10 | Divine Intervention | none | 0 | none |
| AC-11, AC-13, AC-15 | 11, 13, 15 | none | none | 0 | none |
| AC-14, AC-17, AC-19, AC-20 | 14, 17, 19, 20 | 14: Improved Blessed Strikes; 17: Subclass feature; 19: Epic Boon; 20: Greater Divine Intervention | none | 0 | none |
| AC-18 | 18 | none | none | 1 (maximum 3 to 4) | none |

#### 4.14.4 Collaborator failure cases (one test per row; source: doc/rules/testing.md §4, doc/rules/observability.md §7; CF-05 to CF-07 were content and class skill rows owned by other specs, FR-126)
`.errors` is `srd5e.character.feature_used.errors` (character-aggregate.md FR-118).

| Label | Use case and double | Expected | Path |
|---|---|---|---|
| CF-01 | `UseClericChannelDivinity`, Divine Spark; `dice.Roller` throws `*srderr.PortError` (port `dice`) | the same instance propagates; no result, no use expended, no events; error span status with `error.type=dice_roller_error`; one `feature_used` increment with `srd5e.outcome=failure` and no `error.type`; one `.errors` increment with `error.type=dice_roller_error`; one duration sample; no log | unhappy (collaborator failure) |
| CF-02 | `UseClericChannelDivinity`, Turn Undead with `use`, level 5; roller returns a result for a different expression | `*srderr.PortError` (port `dice`) from `rollChecked` propagates; state unchanged; error span status with `error.type=dice_roller_error`; one `.errors` increment with `error.type=dice_roller_error` | unhappy (invalid data from a collaborator) |
| CF-03 | `UseDivineIntervention`, Wish at level 20; roller returns `*srderr.PortError` (port `dice`) | propagates; `DivineInterventionState` unchanged (not `DivineInterventionRecharging`); error span status with `error.type=dice_roller_error`; one `feature_used` increment with `srd5e.outcome=failure` and no `error.type`; one `.errors` increment with `error.type=dice_roller_error` | unhappy (collaborator failure) |
| CF-04 | `UseClericChannelDivinity`, Divine Spark at level 2; roller returns the right expression with a face of 9 | panic (FR-054) propagates; state unchanged; the span records it as it propagates; no `error.type`; no counter increment | unhappy (invalid data from a collaborator) |
| CF-08 | `UseClericChannelDivinity`, Divine Spark healing at level 17, Life Domain; roller double that panics if called | no error; zero roller calls (FR-055) | unhappy (rule interaction: no roll) |

### 4.15 Withdrawn ids
Ids are kept so nothing is renumbered. Their tests live in the named specs.

| Id | Requirement |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (FR-003, FR-016 to FR-023) and content-classes-data.md (FR-085) |
| FR-002 | Withdrawn: moved to content-classes.md (FR-003, FR-016 to FR-023) and content-classes-data.md (FR-085) |
| FR-003 | Withdrawn: moved to content-classes.md (FR-003, FR-016 to FR-023) and content-classes-data.md (FR-085) |
| FR-004 | Withdrawn: moved to character-creation.md (FR-015, FR-019, FR-020, FR-070) and content-classes.md (FR-020, FR-024, FR-025) |
| FR-005 | Withdrawn: moved to character-creation.md (FR-016, FR-018, FR-019) and content-classes.md (FR-020) |
| FR-006 | Withdrawn: moved to character-creation.md (FR-017) and content-classes.md (FR-020) |
| FR-007 | Withdrawn: moved to content-classes.md (FR-003, FR-016 to FR-023) and content-classes-data.md (FR-085) |
| FR-008 | Withdrawn: moved to content-classes.md (FR-003, FR-016 to FR-023) and content-classes-data.md (FR-085) |
| FR-009 | Withdrawn: moved to character-creation.md (FR-020, FR-070) and content-classes.md (FR-024, FR-025) |
| FR-010 | Withdrawn: moved to character-creation.md (FR-020, FR-070) and content-classes.md (FR-024, FR-025) |
| FR-011 | Withdrawn: moved to character-creation.md (FR-020) and content-classes.md (FR-024) |
| FR-012 | Withdrawn: moved to content-classes.md (FR-003, FR-016 to FR-023) and content-classes-data.md (FR-085) |
| FR-013 | Withdrawn: moved to content-classes.md (FR-033) and content-classes-data.md (FR-090) |
| FR-014 | Withdrawn: moved to content-classes.md (FR-047) |
| FR-016 | Withdrawn: moved to content-classes-data.md (FR-085) and content-schema-and-decoder.md (FR-001, FR-012) |
| FR-018 | Withdrawn: moved to character-creation.md (FR-102, FR-104) |
| FR-019 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-108) |
| FR-020 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-110) |
| FR-023 | Withdrawn: moved to content-classes.md (FR-029) |
| FR-024 | Withdrawn: moved to content-spells.md (FR-023 to FR-025) and content-classes-data.md (FR-100, FR-101) |
| FR-025 | Withdrawn: moved to content-spells.md (FR-023 to FR-025) and content-classes-data.md (FR-100, FR-101) |
| FR-026 | Withdrawn: moved to content-spells.md (FR-023 to FR-025) and content-classes-data.md (FR-100, FR-101) |
| FR-028 | Withdrawn: moved to spellcasting.md (FR-047) |
| FR-068 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-041, FR-044, FR-049) |
| FR-069 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-045) |
| FR-070 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-016, FR-017) |
| FR-071 | Withdrawn: moved to content-classes.md (FR-075) |
| FR-072 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-016) |
| FR-073 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-016) |
| FR-074 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-016) |
| FR-120 | Withdrawn: moved to character-creation.md (FR-006) and level-advancement-and-multiclassing.md (FR-020, FR-050) |
| FR-121 | Withdrawn: moved to character-creation.md (FR-006) and level-advancement-and-multiclassing.md (FR-022, FR-050) |
| FR-127 | Withdrawn: moved to content-classes.md (FR-047) |
| FR-129 | Withdrawn: moved to character-aggregate.md (FR-043) and level-advancement-and-multiclassing.md (FR-011, FR-024) |

## 5. Model

Domain (`character` package, pure domain files; pure, synchronous). Per shared-types-and-ownership.md FR-023 all Cleric feature logic is a pure domain function from (character state, inputs) to (new state, events); the two dice-using features also have a use case in `application/`. No operation loads content: callers (character-creation.md, level-advancement-and-multiclassing.md) pass the facts as values (a `ProgressionRow`, the held `SubclassDefinition`). The domain reads content types from content-classes.md (FR-025, FR-047, FR-048, FR-050, FR-055), `Spell` (content-spells.md), the `content` vocabulary types and ids (shared-types-and-ownership.md FR-005), `shared_kernel` types (FR-001), `DifficultyClass` and `RollModifiers` (d20-tests.md), `SpellSaveDc` (spellcasting.md), `RestKind` (rests-and-hit-dice.md, FR-009) and `FeatureState` / `FeatureStateKey` (character-aggregate.md FR-092, FR-093). It defines no class-definition, progression-row, spell-list or subclass type and never decodes.

- **Types:** `ClericRuleData` (FR-133, with `ExtraDivineInterventionGrant`: `SpellId`, minimum Cleric level, recharge dice); `LifeDomain` and `LifeDomainSpells` (FR-098, FR-134); `ClericCreationChoices` (FR-135, FR-140); `ClericChoices` (each option set once, FR-037, FR-076) and `ClericResources` (FR-038, FR-042 to FR-044, FR-047, FR-091 to FR-095), seeded at creation or supplied at the first advance of a multiclass Cleric (FR-138). The subclass is held by the aggregate and passed in.
- **Domain services** (stateless): `DivineSpark` (FR-049 to FR-054), `TurnUndead` and `SearUndead` (FR-058 to FR-066, FR-131), `ThaumaturgeBonus` (FR-035), `BlessedStrikes` (FR-077 to FR-085), `DivineIntervention` (FR-086 to FR-090, FR-096), `ExtraDivineInterventionRecharge` (FR-094, FR-097), `DiscipleOfLife`, `BlessedHealer` (FR-101 to FR-104, FR-112, FR-113), `PreserveLife` (FR-105 to FR-111), `SupremeHealing` (FR-114 to FR-116).
- **Handoff values** (immutable): `ClericEffect` data per Channel Divinity effect (action, range, save `Ability`, DC, amounts, `DamageType`s, `ConditionKind`s, duration, early endings, Sear Undead damage on a failed save), `SparkTarget`, `UndeadTarget`, `HealingTarget` (opaque host key, facts, distance), `HealedCreature` and `HealingCast`. Creatures are host keys, Hit Point amounts plain `int`s; no `combat` type is used (shared-types-and-ownership.md FR-011, FR-013).
- **Events** (`ClericEvent` family, FR-136): `ClericOptionChosen`, `ClericAdvanced`, `ClericChannelDivinityUsed`, `ClericChannelDivinityRegained`, `DivineInterventionUsed`, `DivineInterventionRecharged`. `SubclassChosen` and `RestApplied` (rests-and-hit-dice.md FR-036) are defined elsewhere. The result variant `AdvanceClericApplied` is a different type from the `ClericAdvanced` event it carries.
- **Invariants:** uses within 0 and the maximum (FR-047); roll values within the die's faces and the right count (FR-054, FR-067, FR-097); healing dice of at least one die with two sides (FR-116); a Preserve Life allocation within the pool, per-target room and distinct targets (`NewPreserveLifeAllocation`, FR-108 to FR-110).
- **Failures:** untrusted host input returns a closed rejection (table 4.14.1), never an error; `MustX` constructors panic with an error value for broken invariants; missing content is the loading caller's `MissingContent` (shared-types-and-ownership.md FR-018); port failure is `*srderr.PortError` and propagates (`doc/rules/architecture.md` §6).

## 6. Interface

Domain functions are in the `character` package (pure domain files), the two use cases in the `character` package (use-case files) (`doc/rules/architecture.md` §4). Each operation takes the current state and returns the new state and events in its own closed `<Operation>Result` with operation-prefixed variants (shared-types-and-ownership.md FR-017), so no generic `Rejected`, `NotACleric`, `NotApplicable` or `ContentMissing` name is exported. Each operation that carries the `Character` has a `<Operation>NotACleric` variant (FR-122). None loads content or is context-aware (FR-126).

| Operation | Kind | Inputs | Result variants | Requirements |
|---|---|---|---|---|
| `ChooseBlessedStrikes` | domain function | character, `ClericChoices`, the requested option names | `ChooseBlessedStrikesChosen(choices, events)` / `ChooseBlessedStrikesRejected(reason)` / `ChooseBlessedStrikesNotACleric` | FR-075, FR-076 |
| `ChooseDivineOrder` | domain function | character, `ClericChoices`, the requested order names, the extra cantrip as an optional `Spell` and the known cantrips | `ChooseDivineOrderChosen(choices, granted proficiencies, extra cantrip, extraCantrips allowance 1 or 0, check bonus, events)` / `ChooseDivineOrderRejected(reason)` / `ChooseDivineOrderNotACleric` | FR-022, FR-029 to FR-037, FR-141 |
| `AdvanceCleric` | domain function | character (already holding the new Cleric level, FR-122), `ClericChoices` and `ClericResources` (or absent, FR-138), the held optional `SubclassDefinition` (FR-134), the previous Cleric level and the new level's `ProgressionRow` (level validity: level-advancement-and-multiclassing.md) | `AdvanceClericApplied(features gained as List<FeatureId>, pending choices as List<ClericPendingChoice>, resources, always-prepared spells, the ClericAdvanced event)` / `AdvanceClericNotACleric` | FR-021, FR-044, FR-098, FR-099, FR-125, FR-134, FR-138 |
| `UseClericChannelDivinity` | use case (sync; `dice.Roller`, `telemetry.Telemetry`) | character, `ClericResources`, `ClericChoices`, the optional `SubclassId` held, the effect (Divine Spark, Turn Undead, Preserve Life) with its targets, `SpellSaveDc` (the effect data carries its `DifficultyClass`), Wisdom `AbilityModifier`, `SearUndeadChoice` | `UseClericChannelDivinityApplied(effect data, resources, events)` / `UseClericChannelDivinityRejected(reason)` / `UseClericChannelDivinityNotACleric` | FR-038 to FR-067, FR-105 to FR-111, FR-115, FR-130 to FR-132 |
| `UseDivineIntervention` | use case (sync; `dice.Roller`, `telemetry.Telemetry`) | character, `ClericResources`, `ClericRuleData`, the chosen `Spell` (content-spells.md) and its chosen `CastingTime` | `UseDivineInterventionApplied(free-cast request for spellcasting.md CastSpell with CastMode.featureGranted: spell id, CastingTime, no slot choice, no Material components; resources, events, rechargeRoll (optional))` / `UseDivineInterventionRejected(reason)` / `UseDivineInterventionNotACleric` | FR-086 to FR-097, FR-130 |
| `ResolveClericHealing` | domain function | character, `ClericChoices`, the optional `SubclassId` held, `HealingCast` (slot level or none, creatures healed), healing dice (optional) | `ResolveClericHealingResolved(Disciple of Life bonuses, Blessed Healer amount, resolved dice)` / `ResolveClericHealingNotACleric` | FR-101 to FR-104, FR-112 to FR-116 |
| `ResolveBlessedStrikes` | domain function | character, `ClericChoices`, the attack or cantrip facts (the cast optional `Spell`), turn state | `ResolveBlessedStrikesResolved(extra dice or bonus, Temporary Hit Points)` / `ResolveBlessedStrikesNotApplicable(reason)` / `ResolveBlessedStrikesNotACleric` | FR-077 to FR-085 |
| `ApplyClericRest` | domain function | `ClericResources`, `RestKind` taken from `RestApplied.benefitsOf` (rests-and-hit-dice.md FR-036, FR-060: an interrupted Long Rest reports `shortRest`, so the benefit is read from the event) | `ApplyClericRestApplied(resources, events)` (no rejection, no `NotACleric`: it takes no `Character`) | FR-042, FR-043, FR-091, FR-092, FR-095 |

- `UseClericChannelDivinity` rolls the d8s of Divine Spark (not for Supreme Healing, FR-055) and of Sear Undead (only for `use`) through `dice.Roller.rollChecked`; `UseDivineIntervention` rolls the grant's recharge dice only for a granted spell (Wish); `ResolveClericHealing` rolls nothing. `*srderr.PortError` (port `dice`) propagates (FR-130).
- Target saving throws (Constitution, Wisdom) are resolved through `ResolveD20Test` (resolve-d20-test-use-case.md); this spec returns the DC, ability and consequences. `combat` reads this spec's published values (shared-types-and-ownership.md FR-013). Spell Save DC, Proficiency Bonus and the Wisdom modifier are inputs from derived-stats.md and spellcasting.md, never recomputed.
- `UseDivineInterventionApplied` is a request for spellcasting.md `CastSpell` with `CastMode.featureGranted` (casting-and-concentration.md FR-121 and FR-124; FR-106 derives `CastSource.featureGranted`); that cast spends no slot, applies the non-slot casting checks (FR-122) and emits `SpellCast` (FR-123). A use case never calls another (shared-types-and-ownership.md FR-044), so this one owns entitlement and the host performs the cast (D-21).
- Ports: `dice.Roller` and `telemetry.Telemetry` (package `telemetry`, sync, zero-value `telemetry.Noop*` defaults, shared-types-and-ownership.md FR-036). No `ContentSource`, no new port. The package API the `character` package export the operations, commands, results, the types of section 5, the prefixed variants, `ClericPendingChoice`, effect and handoff values, `ClericEvent` and the events; the package API's library doc comment lists inputs and outputs.

## 7. telemetry.Telemetry

Answers to the seven evaluation questions (`doc/rules/observability.md` §2). Only the two use cases emit signals; the domain is silent by design (FR-123).

1. **What can go wrong?** The `dice.Roller` failing or returning bad data (FR-130); a request illegal in the current state (no uses left, Divine Intervention not recharged, an unavailable effect, a non-Undead target, an over-large Preserve Life allocation, a Reaction spell); invalid host input; programmer errors (out-of-range roll values), which propagate as panic. A target failing its save is a normal outcome resolved elsewhere; content problems are the loader's.
2. **How would an operator know?** Span error status and a `.errors` increment with `error.type` for port failures; `srd5e.outcome` on `feature_used` separates `success` from `failure`; a rejection is an outcome, not an error; a programmer error appears only as the error recorded on the span (character-aggregate.md FR-088).
3. **What would someone need to diagnose it?** `srd5e.class` (`cleric`), `srd5e.feature` (`divine_spark`, `turn_undead`, `preserve_life`, `divine_intervention`), `srd5e.outcome` and, on a rejection, `srd5e.character.rejection` (the values of table 4.14.1 plus `not_a_cleric`). The result carries roll, DC, amounts and remaining uses. No creature key, spell id, character id or free text is an attribute.
4. **What should be counted or timed?** Uses per feature and outcome, and latency per call, through the shared signals below. No per-feature metric names.
5. **Is existing instrumentation still correct?** Yes. There is no `srd5e.character.cleric.*` metric or attribute namespace (shared-types-and-ownership.md FR-042); only the two span names are class-qualified, because class-paladin.md's span is the unprefixed `srd5e.character.use_channel_divinity`. `srd5e.d20_test.resolved` is unaffected: this context returns the DC and does not resolve the save.
6. **What should a consumer monitor or alert on?** Any `.errors` increment with `dice_roller_error`, or a span ending in a panic (a defect in the host or roller); a jump in `failure` outcomes for `divine_intervention` or `divine_spark` (a host not tracking rests).
7. **Is anything now noisy?** No. The use cases emit no log.

**Shared signals, not redefined here.** The counters `srd5e.character.feature_used` and `.feature_used.errors` and the histogram `.feature_used.duration` are owned by character-aggregate.md (FR-118; decisions FR-063); the Cleric adds only the attribute values above. This spec owns two spans:

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.cleric.use_channel_divinity` | span | `srd5e.class`, `srd5e.feature`, `srd5e.outcome`, `srd5e.character.rejection` (rejections only) | Trace one `UseClericChannelDivinity` call, including the dice roll | Spans with error status; traces slower than baseline |
| `srd5e.character.cleric.use_divine_intervention` | span | same | Trace one `UseDivineIntervention` call, including a recharge roll | Spans with error status |

Failure recording is FR-124. Signal names are constants in the `character` telemetry constants and attribute keys use the attribute-key constants of package `telemetry` (observability-ports.md FR-052). Entries feed `doc/observability/signals.md`.

## 8. Test plan

Levels: unit (`character_test` package (`*_test.go`)), application (`character_test` use-case tests, real domain, recording `telemetrytest.Recorder` and fixed `dice.Roller` from the `*test` packages), contract (none: no new port), scenario (`scenarios/`, public API only), architecture (existing guardrails). Every test and group carries `Requirement:` and `Path:` comments; table-driven tests cite per row (`doc/rules/testing.md` §5). Tables 4.5.1 to 4.14.4 are the rows. Domain functions have no telemetry tests (observability.md §9). Behaviour tests build a minimal in-memory `ClassDefinition` or `SubclassDefinition` with only the columns and facts under test, or read the bundled content, and copy no SRD text beyond `testdata/srd/` (shared-types-and-ownership.md FR-033). There is no decoder test in this spec.

| Id | Level | Notes |
|---|---|---|
| FR-015, FR-017 | unit (consistency) | `cleric_consistency_test.go`: bundled table against the Proficiency Bonus rule and spellcasting.md (Phase 3 class data of content-classes.md; no spell ids read); not a decode |
| FR-021, FR-022, FR-134 | unit | Pending choices per level (`divineOrder` at 1, `blessedStrikesOption` at 7, none elsewhere); features gained and newly granted always-prepared spells asserted in AC-01 to AC-20; multiclass Protector |
| FR-027, FR-133 | unit | Recommended ids (differing selections not rejected); defaults equal the SRD values; each default id (Guidance, Sacred Flame, Thaumaturgy, Bless, Cure Wounds, Guiding Bolt, Shield of Faith, Wish) exists in the bundled content once content-spells.md ships it |
| FR-029 to FR-037, FR-141 | unit | `divine_order_test.go`: extra cantrip valid, level 1 spell, already known, off-list, missing; second choice; reasons per FR-128; FR-035 and FR-036 over 4.5.1; allowance 1 or 0 fed to a `ClassSpellcasting` fixture (cap and headroom are spellcasting.md FR-096, case O13, cited not re-tested) |
| FR-038 to FR-047 | unit, application | `channel_divinity_uses_test.go`: maximum at levels 1, 2, 5, 6, 17, 18, 20; gains 1 to 2, 5 to 6, 17 to 18; constructor bounds; transitions in unit, results and DC carry-through in application |
| FR-048 to FR-057, FR-132 | unit, application | `divine_spark_test.go`: over 4.7.1 and 4.7.2; the use case rolls exactly the number of d8s in one call per level band, zero calls for Supreme Healing (SH-11, CF-08) |
| FR-058 to FR-067, FR-131 | unit, application | `turn_undead_test.go`: non-Undead target, 30 and 31 feet, empty list; over 4.8.1; one roll whatever the target count, none for `skip` |
| FR-075 to FR-085 | unit | `blessed_strikes_test.go`: levels 6, 7, 13, 14, double and second choice, other option, spent turn, spell attack, damage types; over 4.10.1 incl. 60 and 61 feet |
| FR-086 to FR-097 | unit, application | `divine_intervention_test.go`: over 4.11.1 and 4.11.2; second use rejected, Short Rest no effect, Long Rest from each state; grant dice through a fixed `dice.Roller` at 1, 1 and 4, 4; a non-SRD `ClericRuleData` grant proves no literal id |
| FR-098, FR-099, FR-100 | unit | Over 4.12.1 (LS-02 to LS-07 read the bundled Life Domain, which also checks `LifeDomain.subclassId`; LS-10 and LS-11 use fixtures); the handoff is asserted as the always-prepared input of spellcasting.md |
| FR-101 to FR-116 | unit | `life_domain_healing_test.go` over 4.12.2; `preserve_life_test.go` over 4.12.3; `supreme_healing_test.go` over 4.12.4, one test per row, incl. the no-subclass and other-subclass rows |
| FR-117, FR-118, FR-119 | unit, application | Cleric 3 / Fighter 5: pool 15, one die, Proficiency Bonus +3; inputs not mutated, returned slices are defensive copies (mutating one leaves the state unchanged); fixed roller at the minimum and maximum faces for each rolling use case |
| FR-122, FR-126, FR-138, FR-139 | unit, application, architecture | No Cleric levels for each operation carrying the `Character`; `AdvanceCleric` 0 to 1 supplies the initial states; no `ContentSource` import or `context.Context`-taking; key names collide with no other class key |
| FR-123, FR-124, FR-130 | application | Same result with no-op and recording `telemetrytest.Recorder`; span, counters, histogram, attribute keys and values pinned; CF-01 to CF-04 with `dicetest.FixedRoller` and `CallbackDiceRoller`; rejections set no error status and no `error.type`; a panic leaves no counter increment |
| FR-125, FR-128 | unit, application | AV-01, AV-02 and the AC rows (one `ClericAdvanced` per advance) and 4.14.2; every reason of 4.14.1 is hit at least once; rejection attribute strings pinned |
| FR-135, FR-136, FR-137, FR-140 | unit, architecture | `ClericCreationChoices` accepted by `CreateCharacter` and stored under `ClericFeatureKeys`, no Cleric field in the aggregate; the `New` constructor (returns an error) rejections (each reason except `already_chosen`); `seed` adds both states, keeps others, leaves the input unchanged, panics with an error value on a held key; `contentKeys` names the extra cantrip for a Thaumaturge only and a double source missing it gives `CreateCharacterContentMissing` (character-creation.md FR-006, FR-103); the bonus is a `Modifier`; `ClericEvent` checks and package API scan of FR-136 |
| Whole slice | scenario | `scenarios/cleric_test.go`, public API only: a level 7 Life Domain Cleric uses Divine Spark, Turn Undead with Sear Undead and Preserve Life, rests, and later a level 20 Cleric uses Divine Intervention with Wish. Cites FR-039, FR-052, FR-064, FR-095, FR-105 |

- **Doubles:** `dicetest.FixedRoller`, `CallbackDiceRoller` and the recording `telemetrytest.Recorder` from the `*test` packages (shared-types-and-ownership.md FR-038); no new port, so no contract suite.
- **Mutation testing:** changed lines in the `character` package (pure domain files) at a score of at least 80 (`doc/rules/testing.md` §8, `gomu` in Docker with the CPU caps, `--base-branch`); the boundary rows (levels 6/7, 13/14, 17/18, 30/31 feet, 60/61 feet, half-maximum, room 6/7, pool 15/16) kill `<` versus `<=` mutants. Application lines SHOULD be mutation tested too.
- **Not covered:** no live FR lacks a test here; withdrawn ids (4.15) are covered by the specs named there.

## 9. Open questions and decisions

No question is open or escalated.

Decisions taken here (all decided, draft), referenced from the requirements:

- **D-2:** when the Channel Divinity maximum rises the remaining uses rise by the same number, never above the maximum (FR-044); matches Spell Slots in spellcasting.md; changing it later is breaking.
- **D-3:** the SRD gives no minimum for a negative Wisdom modifier. Damage and healing amounts are never negative (FR-053, "Damage Rolls" p. 16); Potent Spellcasting adds the literal modifier and the host's damage rules floor the total at 0 (PS-03); Temporary Hit Points "twice your Wisdom modifier" is 0 at a modifier of 0 or less (FR-083). Revisit if an erratum says otherwise.
- **D-4, D-7:** the Divine Order and the Blessed Strikes option cannot change once chosen (FR-037, FR-076): "You have dedicated yourself", "The option you chose". **D-6:** a Turn Undead or Preserve Life use with no chosen target is legal and expends the use (FR-063, FR-111); a host may refuse it in its UI. **D-8:** Disciple of Life and Blessed Healer need "a spell you cast with a spell slot", so Channel Divinity healing and Divine Intervention casts get nothing (FR-104, FR-113).
- **D-9:** Preserve Life rejects an over-large allocation with the room as data (FR-109); a partial allocation is accepted and the rest lost (FR-108); "no more than half" is half rounded down (PL-07).
- **D-15, D-17, D-18:** Turn Undead movement is effect data for the host (FR-059); the Sear Undead roll is optional, `SearUndeadChoice` defaults to `skip` and the roll happens once even with no target (FR-064, FR-131); `ClericAdvanced` fires on every successful `AdvanceCleric` with a possibly empty payload (FR-125, 4.14.3).
- **Settled elsewhere:** D-25 (the Cleric rest result name: rests-and-hit-dice.md FR-036 and its consumer table no longer name `ClericRestApplied`; this spec keeps `ApplyClericRestApplied`, shared-types-and-ownership.md FR-017); D-19 (a programmer error is recorded on the span only) by shared-types-and-ownership.md FR-041 and character-aggregate.md FR-088; D-21 (Divine Intervention free cast as a `CastSpell` request) by casting-and-concentration.md FR-121; D-22 (Bloodied is `isBloodied`) by shared-types-and-ownership.md FR-049; the subclass change rule, state placement, shared telemetry (character-aggregate.md FR-118), Critical Hit doubling, the Channel Divinity names and event base types by shared-types-and-ownership.md FR-013, FR-017, FR-019 to FR-022, FR-042, FR-063; the Life Domain Spells are structured content (content-classes.md FR-055, FR-058); Wish is bundled by content-spells.md and tested by FR-133.

## 10. Changelog impact

Under `Unreleased`, **Added**: the Cleric behaviour (Divine Order, Channel Divinity with Divine Spark, Turn Undead, Preserve Life and Sear Undead, Blessed Strikes, Divine Intervention and Greater Divine Intervention, the Life Domain healing features), `ClericCreationChoices`, `ClericChoices`, `ClericResources`, the two use cases, the domain functions of section 6, the `ClericEvent` family, `ClericRuleData`, `LifeDomain`, `LifeDomainSpells`, and the two span signals of section 7. Not **BREAKING**. The class and subclass data and the spell list are deliverables of content-classes.md and content-spells.md (the `NOTICE.md` coverage list is shared-types-and-ownership.md FR-033); the FR-133 id-existence test waits for the bulk spell data (Phase 5); FR-015 and FR-017 run in Phase 3. Type phase 3 (shared-types-and-ownership.md FR-014).
