# Class: Sorcerer

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `derived-stats.md`, `character-aggregate.md`, `spellcasting.md`, `rests-and-hit-dice.md`, `level-advancement-and-multiclassing.md`, `character-creation.md`, `feats-behaviour.md`, `content-classes.md`, `content-spells.md`, `content-source-port.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `d20-tests.md`, `errors-and-results.md`, `observability-ports.md` |

Roadmap: [../roadmap.md](../roadmap.md); type phase 3 (shared-types-and-ownership.md FR-014). No `combat` dependency (FR-013); last in the `character` order (FR-016).

> Go notation used below. A name in lowerCamelCase (a field, command argument or enum member) stands for the exported PascalCase Go identifier (`attackerID` is the field `AttackerID`). A trailing `?` marks an optional value (the zero value or a `(T, bool)` pair). `[]T` is a slice; `set of T` is a collection without duplicates. A `(T, error)` pair replaces an accepted/rejected result pair: the nil-error case is the accepted case and the error is the typed rejection, satisfying `errors.Is(err, srderr.ErrInvalidInput)`; a parser reports every problem at once through `errors.Join`. Rule outcomes (refusals, not-possible stops) stay result values. A use case is a struct with `Run(ctx, cmd)`; its `error` is non-nil only for a port failure.

## 1. Purpose and scope

The Sorcerer class as SRD 5.2.1 presents it (printed pp. 64-70): Core Sorcerer Traits, the Features table, every class feature, the
ten Metamagic options, the spell list and the one SRD subclass (Draconic Sorcery). The **data** is owned by content-classes.md
(shared-types-and-ownership.md FR-025; the list is derived from `Spell.Classes`, FR-078 there). This spec transcribes the SRD
tables for tests and owns the **character-context behaviour**: pure domain logic, dice and time are host input. Feature logic is a
pure function from (state, inputs) to (new state, events) (FR-023); four features that need content get one use case each (section 6).
Generic spellcasting is cited from spellcasting.md by id. "Table C" here is the Creating Spell Slots table.

**Out of scope:**

- Class, subclass, row and feature shapes and bulk data: content-classes.md. Subclass choice: level-advancement-and-multiclassing.md
  (shared-types-and-ownership.md FR-020). Rests: rests-and-hit-dice.md (shared-types-and-ownership.md FR-009). Feats and Epic Boons: feats-behaviour.md. Class skill and equipment
  validation: character-creation.md. Aggregate, level-up flow, Hit Points: character-aggregate.md, derived-stats.md.
- Generic spellcasting (spellcasting.md): save DC and attack bonus (FR-003, FR-004), ability and Focus (FR-002, FR-068),
  slot tables (table C, FR-010), cantrip and prepared counts (FR-014, FR-015), selection validation (FR-040 to FR-044,
  FR-052 to FR-054), always-prepared spells (FR-045, FR-046), replacement (FR-048, FR-049, FR-053), Long Rest slots
  (FR-025), slot expenditure (FR-021), multiclass slots (FR-031). Individual spell and feat text.
- Spell school and Concentration/Ritual/Material markers of the list; SRD recommendations (Light, Prestidigitation, Shocking
  Grasp, Sorcerous Burst, Burning Hands, Detect Magic, Boon of Dimensional Travel), which are not rules. Time keeping: "1 minute",
  "1 hour", "each of your turns" are host-issued commands. No other subclass or feature; nothing outside SRD 5.2.1 or from SRD 5.1.

## 2. Source

- SRD 5.2.1, "Sorcerer" (Core Sorcerer Traits), p. 64; "Sorcerer Features", "Sorcerer Class Features" (Spellcasting), pp. 64-65;
  "Level 1: Innate Sorcery", pp. 65-66; "Level 2: Font of Magic", "Creating Spell Slots", "Level 2: Metamagic", p. 66.
- SRD 5.2.1, "Level 3: Sorcerer Subclass", "Level 4: Ability Score Improvement", "Level 5: Sorcerous Restoration", "Level 7: Sorcery Incarnate", "Level 19: Epic Boon", "Level 20: Arcane Apotheosis", p. 66.
- SRD 5.2.1, "Metamagic Options", pp. 66-67; "Sorcerer Spell List", pp. 67-69; "Sorcerer Subclass: Draconic Sorcery", pp. 69-70.
- Package design: `doc/rules/architecture.md` §3 to §6, `doc/rules/testing.md` §4, §5, `doc/rules/observability.md` §5 to §7, shared-types-and-ownership.md; this spec's FR-099 to FR-101 and FR-124 to FR-137 derive from these. Page offset: none.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Sorcerer, Core Sorcerer Traits, Features table, subclass | `ClassDefinition`, `ProgressionRow`s, `SubclassDefinition` (content-classes.md) | `ClassID` `sorcerer`; subclass at level 3 via level-advancement-and-multiclassing.md |
| Creating Spell Slots | content `FeatureTable` (content-classes.md FR-066, FR-067) | Bonus Action; held as a created slot in `SorcererState` (FR-136) |
| Metamagic option | `FeatureOption` with `pointCost` (content-classes.md FR-061 to FR-065) plus closed `MetamagicEffect` | ids `careful_spell`, `distant_spell`, `empowered_spell`, `extended_spell`, `heightened_spell`, `quickened_spell`, `seeking_spell`, `subtle_spell`, `transmuted_spell`, `twinned_spell` |
| Sorcerer spell list | derived content (content-spells.md FR-120) | Authoritative membership is the header-derived 140; the 138 printed are the oracle only (content-spells.md FR-022, FR-023, FR-097; 4.11) |
| Sorcerer spell | derived set | Prepared or known spell from the derived list plus, per FR-103, any always-prepared spell a Sorcerer feature grants (spellcasting.md FR-037) |
| `MetamagicSpellFacts`, `MetamagicCastContext`, `SorcererProfile`, `SorcererEvent` | handoff values; closed event family implementing `CharacterEvent` | 4.8, FR-110, FR-137, FR-131 |
| `RestKind`, `DamageType`, `Skill`, `SpellID`, `FeatID`, `ItemID`, Concentration | owned elsewhere | rests-and-hit-dice.md FR-001; `content` (shared-types-and-ownership.md FR-005); spellcasting.md (state shape) and casting-and-concentration.md (transitions; shared-types-and-ownership.md FR-011) |

## 4. Requirements

Ids are stable and never renumbered; withdrawn ids are in the last table of this section; new ids are FR-128 and above (FR-119,
FR-121, FR-122 are unused). Each case id is one test (a row naming several ids is one test per id). A PascalCase rejection name in prose (`CountOutOfRange`) is shorthand
for the constant of `SorcererRejectionKind` or `MetamagicIneligibility` (section 5), a typed enum value; it is not a separate type.

### 4.1 Becoming a Sorcerer

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | A character taking Sorcerer as a level 1 character MUST gain all Core Sorcerer Traits and the level 1 features (Spellcasting, Innate Sorcery). The grant is applied by character-creation.md from the `ClassDefinition`; this spec supplies the seeded `SorcererState` through the no-argument `SorcererCreationChoices` (FR-129). | SRD 5.2.1, "As a Level 1 Character", p. 64 | happy |
| FR-011 | A multiclass Sorcerer MUST gain the Hit Point Die and the level 1 features, and MUST NOT gain saving throw, skill or weapon proficiencies, starting equipment or any other trait. The generic grant seeds no feature state, so `ApplySorcererLevelUp` MUST seed the level 1 `SorcererState` (FR-129) when the `LevelGained` event is for `sorcerer` at class level 1 and no `SorcererState` exists; at any other level with none it MUST return `NotASorcerer`. | SRD 5.2.1, "As a Multiclass Character", p. 64 | happy |

### 4.2 Progression by level

Class Features and Sorcery Points are content data (content-classes.md, with their row tests); the other columns are spellcasting.md's (tables C, F, G; see this spec's FR-102).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-017 | Pointer, not a second owner. The Proficiency Bonus column of Table P MUST agree with `ProficiencyBonus` (derived-stats.md) at the total level equal to each Sorcerer level. Tested once for every class by content-classes.md FR-035; Table P is the Sorcerer row set of that test. | SRD 5.2.1, "Sorcerer Features", p. 65; content-classes.md FR-035 | happy |
| FR-018 | A Sorcerer level below 1 or above 20 MUST be rejected (`srderr.ErrInvalidInput` from the value-object constructor; a non-returning parse for untrusted input). | SRD 5.2.1, "Sorcerer Features", p. 65 | unhappy (boundary) |
| FR-019 | `sorcererFeatureAvailability(SorcererProfile)` MUST report at Sorcerer level 1 0 Sorcery Points and no Font of Magic or Metamagic, available from level 2 (Sorcerous Restoration from 5, Sorcery Incarnate from 7, Arcane Apotheosis at 20). | SRD 5.2.1, "Sorcerer Features", p. 65 | unhappy (empty/minimal) |
| FR-102 | Pointer, not a second owner. The slot columns of Table P and its Cantrips and Prepared Spells columns MUST equal spellcasting.md table C (FR-010), table F (FR-014) and table G (FR-015) at every Sorcerer level 1-20 (single-class values; multiclass: spellcasting.md FR-031, FR-036). Tested once for every class by the scenario test of content-classes.md FR-035; this spec adds no Sorcerer-only test (a decoder in `content` cannot read the spellcasting tables, shared-types-and-ownership.md FR-013). Spellcasting tables remain the only runtime source. | SRD 5.2.1, "Sorcerer Features", p. 65; "Level 1: Spellcasting", pp. 64-65; content-classes.md FR-035; spellcasting.md FR-010, FR-014, FR-015, FR-031, FR-036 | happy |

**Table P: progression by Sorcerer level, with the spell slot columns ("Table S").** Source for every row: SRD 5.2.1, "Sorcerer
Features", p. 65; path happy, rows 1 and 20 also boundaries. Proficiency Bonus: FR-017; Cantrips, Prepared Spells, slots: FR-102.

| Lvl | PB | Class Features | SP | Cant. | Prep. | 1st | 2nd | 3rd | 4th | 5th | 6th | 7th | 8th | 9th |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | +2 | Spellcasting, Innate Sorcery | 0 (dash) | 4 | 2 | 2 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| 2 | +2 | Font of Magic, Metamagic | 2 | 4 | 4 | 3 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| 3 | +2 | Sorcerer Subclass | 3 | 4 | 6 | 4 | 2 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| 4 | +2 | Ability Score Improvement | 4 | 5 | 7 | 4 | 3 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| 5 | +3 | Sorcerous Restoration | 5 | 5 | 9 | 4 | 3 | 2 | 0 | 0 | 0 | 0 | 0 | 0 |
| 6 | +3 | Subclass feature | 6 | 5 | 10 | 4 | 3 | 3 | 0 | 0 | 0 | 0 | 0 | 0 |
| 7 | +3 | Sorcery Incarnate | 7 | 5 | 11 | 4 | 3 | 3 | 1 | 0 | 0 | 0 | 0 | 0 |
| 8 | +3 | Ability Score Improvement | 8 | 5 | 12 | 4 | 3 | 3 | 2 | 0 | 0 | 0 | 0 | 0 |
| 9 | +4 | none | 9 | 5 | 14 | 4 | 3 | 3 | 3 | 1 | 0 | 0 | 0 | 0 |
| 10 | +4 | Metamagic | 10 | 6 | 15 | 4 | 3 | 3 | 3 | 2 | 0 | 0 | 0 | 0 |
| 11 | +4 | none | 11 | 6 | 16 | 4 | 3 | 3 | 3 | 2 | 1 | 0 | 0 | 0 |
| 12 | +4 | Ability Score Improvement | 12 | 6 | 16 | 4 | 3 | 3 | 3 | 2 | 1 | 0 | 0 | 0 |
| 13 | +5 | none | 13 | 6 | 17 | 4 | 3 | 3 | 3 | 2 | 1 | 1 | 0 | 0 |
| 14 | +5 | Subclass feature | 14 | 6 | 17 | 4 | 3 | 3 | 3 | 2 | 1 | 1 | 0 | 0 |
| 15 | +5 | none | 15 | 6 | 18 | 4 | 3 | 3 | 3 | 2 | 1 | 1 | 1 | 0 |
| 16 | +5 | Ability Score Improvement | 16 | 6 | 18 | 4 | 3 | 3 | 3 | 2 | 1 | 1 | 1 | 0 |
| 17 | +6 | Metamagic | 17 | 6 | 19 | 4 | 3 | 3 | 3 | 2 | 1 | 1 | 1 | 1 |
| 18 | +6 | Subclass feature | 18 | 6 | 20 | 4 | 3 | 3 | 3 | 3 | 1 | 1 | 1 | 1 |
| 19 | +6 | Epic Boon | 19 | 6 | 21 | 4 | 3 | 3 | 3 | 3 | 2 | 1 | 1 | 1 |
| 20 | +6 | Arcane Apotheosis | 20 | 6 | 22 | 4 | 3 | 3 | 3 | 3 | 2 | 2 | 1 | 1 |

### 4.3 Subclass

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-024 | `subclassFeaturesAvailable(SorcererProfile, SubclassDefinition?)` MUST return each subclass feature whose level is at most the profile's Sorcerer level (Draconic Sorcery: 3, 6, 14, 18) and none above it, and none while no subclass is chosen. Levels come from the chosen `SubclassDefinition`; the level and the chosen `SubclassID?` are explicit inputs (FR-137), not read from `SorcererState`. | SRD 5.2.1, "Level 3: Sorcerer Subclass", p. 66 | unhappy (boundary) |

### 4.4 Spellcasting (owned by spellcasting.md)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-032 | The always-prepared status of Draconic Spells MUST be honoured as spellcasting.md defines it: they do not count against the Prepared Spells number (FR-045) and replacing one MUST be rejected with `AlwaysPreparedNotReplaceable` (FR-046). The SRD states the not-counted part (p. 65); non-removal is an inference from "you thereafter always have the listed spells prepared" (p. 70). | SRD 5.2.1, "Prepared Spells of Level 1+", p. 65; "Draconic Spells", p. 70 (inference); spellcasting.md FR-045, FR-046 | unhappy (rule interaction) |
| FR-035 | Gaining a Sorcerer level MUST report the number of Metamagic options still to choose (the new level's entitlement of FR-055 minus the options known). Cantrips and prepared spells still to choose are reported by spellcasting.md `AdvanceSpellcasting` (FR-029, FR-047). | SRD 5.2.1, "Level 2: Metamagic", p. 66; "Level 1: Spellcasting", pp. 64-65 | happy |
| FR-103 | A spell a Sorcerer feature always prepares (the Draconic Spells) MUST count as a Sorcerer spell for the character: Charisma is its spellcasting ability, an Arcane Focus works for it, the Innate Sorcery DC increase and Advantage apply (FR-037) and Metamagic applies (section 4.8), even when the spell is not on the Sorcerer list (Command, Arcane Eye, Legend Lore, Summon Dragon). | SRD 5.2.1, "Prepared Spells of Level 1+" ("those spells otherwise count as Sorcerer spells for you"), p. 65 | unhappy (rule interaction) |

### 4.5 Innate Sorcery

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-037 | Activating Innate Sorcery (a Bonus Action) MUST mark it active and, while active, raise the spell save DC of Sorcerer spells (including always-prepared ones, FR-103) by 1 and give Advantage on their attack rolls. The domain publishes both as the pure function `innateSorceryEffects` (FR-133): a `Modifier` of +1 and an `Advantage` (the `RollCircumstance` of modifiers-and-advantage.md FR-042). The host adds the modifier to the DC `CalculateSpellcasting` returns and passes the circumstance to the attack roll (FR-133). Nothing here reads a `combat` type. | SRD 5.2.1, "Level 1: Innate Sorcery", pp. 65-66 | happy |
| FR-038 | Innate Sorcery MUST have 2 uses; each activation by the normal route MUST consume one. | SRD 5.2.1, "Level 1: Innate Sorcery", p. 66 | happy |
| FR-039 | Activating with no uses left below Sorcerer level 7 MUST be refused with `InnateSorceryNoUsesLeft`, state unchanged. | SRD 5.2.1, "Level 1: Innate Sorcery", p. 66 | unhappy (state) |
| FR-040 | At level 7 or higher with no uses left, the character MUST be able to activate Innate Sorcery by spending 2 Sorcery Points. | SRD 5.2.1, "Level 7: Sorcery Incarnate", p. 66 | happy |
| FR-041 | The 2 Sorcery Point route MUST be refused when uses remain (`InnateSorceryUsesRemain`), with fewer than 2 points (`SorceryPointsInsufficient`), or below level 7 (`SorceryIncarnateUnavailable`). | SRD 5.2.1, "Level 7: Sorcery Incarnate", p. 66 | unhappy (boundary) |
| FR-042 | Finishing a Long Rest (`RestKind.LongRest`) MUST restore both uses. | SRD 5.2.1, "Level 1: Innate Sorcery", p. 66 | happy |
| FR-043 | The host MUST be able to end Innate Sorcery (its 1 minute duration), after which the DC increase and Advantage no longer apply; ending an inactive one MUST be refused with `InnateSorceryNotActive`. | SRD 5.2.1, "Level 1: Innate Sorcery", p. 65 | unhappy (state) |
| FR-044 | Activating while already active MUST be refused with `InnateSorceryAlreadyActive` (it wastes a use and cannot extend the minute). Package decision, SRD silent. | SRD 5.2.1, "Level 1: Innate Sorcery" ("unleash that magic for 1 minute"), p. 65, interpreted | unhappy (state) |
| FR-045 | From level 7, while Innate Sorcery is active, up to two Metamagic options MUST be usable on each spell cast (ordering and counts: FR-111, FR-112). | SRD 5.2.1, "Level 7: Sorcery Incarnate", p. 66 | happy |
| FR-046 | From level 20, while Innate Sorcery is active, one Metamagic option per turn MUST be usable without spending Sorcery Points; the free use MUST NOT be available when inactive, below level 20 or after one free use this turn. The flag clears when the host starts the next turn (FR-109). | SRD 5.2.1, "Level 20: Arcane Apotheosis", p. 66 | unhappy (boundary) |

### 4.6 Font of Magic

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-047 | From level 2 the character MUST have Sorcery Points with the maximum of Table P; on gaining the feature current equals the maximum (2). On each later level gain current MUST rise by the increase of the maximum, never above the new maximum. Package decision, SRD silent. | SRD 5.2.1, "Level 2: Font of Magic" ("you gain more as you reach higher levels"), p. 66, interpreted | happy |
| FR-048 | Current Sorcery Points MUST never exceed the maximum for the Sorcerer level nor go below 0. | SRD 5.2.1, "Level 2: Font of Magic", p. 66 | unhappy (boundary) |
| FR-049 | Finishing a Long Rest MUST restore all expended Sorcery Points. | SRD 5.2.1, "Level 2: Font of Magic", p. 66 | happy |
| FR-050 | Converting a spell slot of level L (1-9) MUST consume the slot and add L Sorcery Points, needing no action. An ordinary slot is consumed through spellcasting.md FR-021 (`ConvertOrdinarySlot`, returning the new `SpellcastingState`); a created slot (FR-136) from `SorcererState.CreatedSlots` (`ConvertCreatedSlot`). A slot level outside 1-9 MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (FR-114). | SRD 5.2.1, "Converting Spell Slots to Sorcery Points", p. 66 | happy |
| FR-051 | Conversion MUST be refused when no such slot is available (`NoSlotToConvert`) or the result would exceed the maximum (`SorceryPointsWouldExceedMaximum`); it MUST NOT cap the gain, so a spell slot is never silently lost. Package decision. | SRD 5.2.1, "Level 2: Font of Magic" ("can't have more Sorcery Points than the number shown"), p. 66, interpreted | unhappy (state) |
| FR-052 | Creating a spell slot of level 1-5 (a Bonus Action) MUST spend the Table C cost and add one created slot of that level to `SorcererState.CreatedSlots` (FR-054), only at or above the table's Min. Sorcerer Level. Cost and minimum are read from the class definition's Creating Spell Slots `FeatureTable` (content-classes.md FR-066, FR-067). | SRD 5.2.1, "Creating Spell Slots", p. 66 | happy |
| FR-053 | Creating a slot MUST be refused with a result for levels 6-9 (`SlotLevelNotCreatable`), fewer points than the cost (`SorceryPointsInsufficient`) or Sorcerer level below the minimum (`SorcererLevelBelowMinimum`); a level below 1 or above 9 MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (FR-114). Refusal leaves state unchanged. | SRD 5.2.1, "Creating Spell Slots", p. 66 | unhappy (boundary) |
| FR-054 | A slot created with Font of Magic MUST vanish when the character finishes a Long Rest. Created slots are counts per level in `SorcererState.CreatedSlots`, not in spellcasting.md `SpellSlots` (FR-136), so the ordinary slot count is untouched. `applySorcererRest` with `RestKind.LongRest` removes them (event `CreatedSlotsExpired`); `RestKind.ShortRest` MUST leave them. | SRD 5.2.1, "Creating Spell Slots" ("vanishes when you finish a Long Rest"), p. 66; rests-and-hit-dice.md FR-001, FR-060 | happy |
| FR-114 | Slot levels follow spellcasting.md FR-024: levels 6-9 are valid slot levels this feature cannot create (Create refuses with a result); a level below 1 or above 9 is a programmer error (Convert and Create returns an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`). | `doc/rules/architecture.md` §6; spellcasting.md FR-024; SRD 5.2.1, "Creating Spell Slots" ("no higher than level 5"), p. 66 | unhappy (boundary) |

**Table C: Creating Spell Slots.** C-1 to C-5: SRD 5.2.1, "Creating Spell Slots", p. 66; C-6 to C-12 apply it; C-13 to C-16 apply FR-136, FR-054.

| Case | Spell slot level | Cost | Min. Sorcerer Level | FR | Path |
|---|---|---|---|---|---|
| C-1 | 1 | 2 | 2 | FR-052 | happy |
| C-2 | 2 | 3 | 3 | FR-052 | happy |
| C-3 | 3 | 5 | 5 | FR-052 | happy |
| C-4 | 4 | 6 | 7 | FR-052 | happy |
| C-5 | 5 | 7 | 9 | FR-052 | happy |
| C-6 | 6 | n/a | n/a | FR-053 | unhappy (boundary): refused `SlotLevelNotCreatable` |
| C-7 | 4 at Sorcerer level 6 | 6 | 7 | FR-053 | unhappy (boundary): refused |
| C-8 | 3 with 4 Sorcery Points | 5 | 5 | FR-053 | unhappy (boundary): refused |
| C-9 | 3 with exactly 5 Sorcery Points | 5 | 5 | FR-053 | unhappy (boundary): accepted, 0 left |
| C-10, C-11 | 0; 10 | n/a | n/a | FR-114 | unhappy (invalid input): `srderr.ErrInvalidInput` |
| C-12 | 9 | n/a | n/a | FR-053 | unhappy (boundary): refused `SlotLevelNotCreatable` |
| C-13 | created level 3 slot spent on a level 3 spell | n/a | n/a | FR-136 | happy: one removed, `CreatedSlotSpent` |
| C-14 | created level 3 slot spent on a level 1 spell | n/a | n/a | FR-136 | unhappy (boundary): accepted, cast level 3 |
| C-15 | created level 2 slot spent when none exists; level 1 slot on a level 3 spell; `slotSpentThisTurn` true | n/a | n/a | FR-136 | unhappy (state): `CreatedSlotUnavailable`; `CreatedSlotLevelTooLow`; `CreatedSlotSpentThisTurn` |
| C-16 | created slot at a Long Rest and at a Short Rest | n/a | n/a | FR-054 | unhappy (state): removed by `longRest` (`CreatedSlotsExpired`), kept by `shortRest` |

**Table V: Converting Spell Slots to Sorcery Points.** Source: SRD 5.2.1, "Converting Spell Slots to Sorcery Points", "Level 2: Font of Magic", p. 66.

| Case | Slot level | State | Expected | FR | Path |
|---|---|---|---|---|---|
| V-1 | 1 | slot available, 0 of maximum points | slot consumed, +1 point | FR-050 | happy |
| V-2 | 9 | slot available, 0 of maximum 20 | slot consumed, +9 points | FR-050 | happy |
| V-3, V-4 | 0; 10 | any | `srderr.ErrInvalidInput` | FR-114 | unhappy (invalid input) |
| V-5 | 5 | slot available, 18 of 20 points | refused `SorceryPointsWouldExceedMaximum`, slot kept | FR-051 | unhappy (boundary) |
| V-6 | 5 | slot available, 15 of 20 points | accepted, 20 points | FR-050 | unhappy (boundary) |
| V-7 | 3 | no level 3 slot remaining | refused `NoSlotToConvert` | FR-051 | unhappy (state) |
| V-8 | 3 | a created level 3 slot, 0 of maximum points | created slot consumed, +3 points | FR-050 | happy |

### 4.7 Metamagic (general)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-055 | Metamagic options known MUST be 2 from level 2, 4 from level 10, 6 from level 17, and 0 at level 1. | SRD 5.2.1, "Level 2: Metamagic", p. 66 | happy |
| FR-056 | A Metamagic selection MUST be accepted when it has the entitled count of distinct options from the Metamagic Options list. | SRD 5.2.1, "Level 2: Metamagic", p. 66 | happy |
| FR-057 | A selection MUST be rejected, with every problem listed, if it is the wrong size, duplicates an option or names an option not in content. | SRD 5.2.1, "Level 2: Metamagic", p. 66 | unhappy (invalid input) |
| FR-058 | On gaining a Sorcerer level at most one known option MUST be replaceable with one not already known; a second replacement MUST be refused. | SRD 5.2.1, "Level 2: Metamagic", p. 66 | happy |
| FR-059 | Using a known option MUST spend its Sorcery Point cost and report the new total. | SRD 5.2.1, "Level 2: Metamagic", p. 66 | happy |
| FR-060 | Using an option MUST be refused when current points are below its cost (one below refused, exactly the cost accepted). | SRD 5.2.1, "Level 2: Metamagic", p. 66 | unhappy (boundary) |
| FR-061 | Using an option the character does not know MUST be refused with `MetamagicOptionNotKnown`. | SRD 5.2.1, "Level 2: Metamagic", p. 66 | unhappy (state) |
| FR-062 | A second option on the same casting MUST be refused unless Sorcery Incarnate applies (FR-045) or the second is Empowered Spell or Seeking Spell (FR-063). | SRD 5.2.1, "Level 2: Metamagic", p. 66 | unhappy (rule interaction) |
| FR-063 | Empowered Spell and Seeking Spell MUST be usable on a casting that already has a different option; the exception covers only these two alongside a different option, and a later option that is neither MUST still be refused (ordering: FR-111). | SRD 5.2.1, "Empowered Spell", "Seeking Spell", p. 67 | unhappy (rule interaction) |
| FR-109 | A `startSorcererTurn` input (the host's turn boundary) MUST clear every per-turn flag: free Metamagic use spent (FR-046), Quickened Spell used this turn (FR-075). No other input clears them; ending Innate Sorcery does not. | `doc/rules/architecture.md` §3 (Purity); SRD 5.2.1, "Level 20: Arcane Apotheosis" ("on each of your turns"), "Quickened Spell", pp. 66-67 | unhappy (state) |
| FR-110 | Every Metamagic command MUST carry a `MetamagicCastContext`: the options already applied to this casting (in order), whether the free use was already spent this turn, `levelOnePlusSpellCastThisTurn` (FR-113). The domain keeps no per-casting state. | `doc/rules/architecture.md` §3, §4; SRD 5.2.1, "Level 2: Metamagic" ("on a spell when you cast it"), p. 66 | unhappy (state) |
| FR-111 | Ordering. An option other than Empowered Spell and Seeking Spell MUST be refused when any option was already applied, unless Sorcery Incarnate allows a second (FR-045). Empowered or Seeking MUST be accepted after any other option: Distant then Empowered is accepted, Empowered then Distant is refused. Package decision (literal reading of "even if you've already used a different Metamagic option"). | SRD 5.2.1, "Level 2: Metamagic", p. 66; "Empowered Spell", "Seeking Spell", p. 67, interpreted | unhappy (rule interaction) |
| FR-112 | Counts. Without Sorcery Incarnate a casting allows one option plus any of Empowered and Seeking. With Innate Sorcery active at level 7+ it allows two options plus any of Empowered and Seeking (up to four; refusal only for a duplicate or a non-exception option past the cap). The same option MUST NOT be applied twice. Package decision. | SRD 5.2.1, "Level 2: Metamagic"; "Level 7: Sorcery Incarnate", p. 66, interpreted | unhappy (rule interaction) |
| FR-113 | Quickened Spell MUST read `MetamagicCastContext.LevelOnePlusSpellCastThisTurn`, a host fact true after any level 1+ spell cast this turn, slot-free casts included (Dragon Companion, a created slot, other features). It is not `CastingContext.SlotSpentThisTurn`, which limits slots only (casting-and-concentration.md FR-061, FR-123) and would miss slot-free casts. The domain keeps no copy; the host derives it from its `SpellCast` events. | SRD 5.2.1, "Quickened Spell" ("cast a level 1+ spell on the current turn"), p. 67; "One Spell with a Spell Slot per Turn", p. 105 | unhappy (rule interaction) |

### 4.8 Metamagic options

Content carries each option's id and cost (content-classes.md FR-061 to FR-065); the ten effects are a closed domain `MetamagicEffect`.

**Spell facts.** `MetamagicSpellFacts` is built by the host from the content-spells.md `Spell` (`SpellRange`, `SpellDuration`,
`CastingTime`, `SpellComponents` with consumed and costed Material flags) plus host-supplied mechanics (saving throw and number of
other creatures affected, spell attack roll as an `AttackRollResult` of d20-tests.md, damage types, higher-level-slot extra
target; decided). A fact not supplied MUST give a `SpellFactUnknown` refusal (FR-123).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-064 | Careful Spell (1 point) MUST be accepted for a spell that forces other creatures to save and report the creatures protected: a count from 1 up to the Charisma modifier (minimum one) and up to the number of other creatures forced to save; each succeeds automatically and takes no damage if it would take half on a success. | SRD 5.2.1, "Careful Spell", pp. 66-67 | happy |
| FR-065 | Careful Spell MUST be refused for a spell that forces no saving throw and for a count above the cap; Charisma modifier 0 or below still allows 1 creature (Table M). | SRD 5.2.1, "Careful Spell", pp. 66-67 | unhappy (boundary) |
| FR-066 | Distant Spell (1 point) MUST report double the range for a range of at least 5 feet (miles double in miles) and 30 feet for Touch. | SRD 5.2.1, "Distant Spell", p. 67 | happy |
| FR-067 | Distant Spell MUST be decided over every `SpellRange` variant (FR-117): refused for Self and for a distance below 5 feet. | SRD 5.2.1, "Distant Spell", p. 67 | unhappy (boundary) |
| FR-068 | Empowered Spell (1 point) MUST accept a reroll count of 1 up to the Charisma modifier (minimum one) of the spell's damage dice and require the new rolls to be used. | SRD 5.2.1, "Empowered Spell", p. 67 | happy |
| FR-069 | Empowered Spell MUST be refused for a count of 0 or below (FR-115), above the cap, or above the number of damage dice rolled. | SRD 5.2.1, "Empowered Spell", p. 67 | unhappy (boundary) |
| FR-070 | Extended Spell (1 point) MUST double a duration of 1 minute or longer, to a maximum of 24 hours, and for a Concentration spell give Advantage on saves to maintain Concentration (state shape: spellcasting.md; save and transitions: casting-and-concentration.md; shared-types-and-ownership.md FR-011). | SRD 5.2.1, "Extended Spell", p. 67 | happy |
| FR-071 | Extended Spell MUST be decided over every `SpellDuration` variant (FR-116): refused below 1 minute; a doubled duration over 24 hours is reported as 24 hours (Table M). | SRD 5.2.1, "Extended Spell", p. 67 | unhappy (boundary) |
| FR-072 | Heightened Spell (2 points) MUST be accepted for a spell that forces a saving throw and report Disadvantage on the saves of one chosen target; refused for a spell with no saving throw (`NoSavingThrow`). | SRD 5.2.1, "Heightened Spell", p. 67 | happy |
| FR-073 | Quickened Spell (2 points) MUST change a casting time of an action to a Bonus Action for that casting. | SRD 5.2.1, "Quickened Spell", p. 67 | happy |
| FR-074 | Quickened Spell MUST be refused for a casting time that is not an action: a Bonus Action, a Reaction, 1 minute or more. | SRD 5.2.1, "Quickened Spell", p. 67 | unhappy (invalid input) |
| FR-075 | Quickened Spell MUST be refused if a level 1+ spell was already cast this turn (FR-113), and after it is used a level 1+ spell cast the same turn MUST be refused by `checkQuickenedRestriction`; cantrips are unaffected. | SRD 5.2.1, "Quickened Spell", p. 67 | unhappy (rule interaction) |
| FR-076 | Seeking Spell (1 point) MUST be accepted only for a spell attack roll that missed, and require the new d20 to be used. The reroll is the `Roll` reroll of d20-tests.md (shared-types-and-ownership.md FR-002); the effect carries the instruction and the host supplies the die. | SRD 5.2.1, "Seeking Spell", p. 67 | happy |
| FR-077 | Seeking Spell MUST be refused when the roll did not miss (the supplied `AttackRollResult` has `hit` true) or the spell has no attack roll. | SRD 5.2.1, "Seeking Spell", p. 67 | unhappy (state) |
| FR-078 | Subtle Spell (1 point) MUST remove every Verbal, Somatic and Material component except Material components consumed by the spell or with a specified cost, and MUST be refused (`NothingToRemove`) when nothing could be removed (package decision: no-effect use). | SRD 5.2.1, "Subtle Spell", p. 67; package decision | happy |
| FR-079 | Transmuted Spell (1 point) MUST change a damage type of Acid, Cold, Fire, Lightning, Poison or Thunder to a different type from that list. | SRD 5.2.1, "Transmuted Spell", p. 67 | happy |
| FR-080 | Transmuted Spell MUST be refused when the spell's damage type or the target type is not in that list, or the target equals the original. | SRD 5.2.1, "Transmuted Spell", p. 67 | unhappy (invalid input) |
| FR-081 | Twinned Spell (1 point) MUST be accepted for a spell that can be cast with a higher-level slot to target an additional creature, and report the effective level increased by 1 (host-supplied fact). | SRD 5.2.1, "Twinned Spell", p. 67 | happy |
| FR-082 | Twinned Spell MUST be refused for a spell that cannot target an additional creature with a higher-level slot (host-supplied fact). | SRD 5.2.1, "Twinned Spell", p. 67 | unhappy (invalid input) |
| FR-115 | A chosen count for Careful (creatures) or Empowered (rerolls) of 0 or below MUST be refused with a result (`CountOutOfRange`), not an error: counts are host input. The minimum-of-one rule applies to the cap, never to the chosen count. | `doc/rules/architecture.md` §6; SRD 5.2.1, "Careful Spell", "Empowered Spell" ("minimum of one"), pp. 66-67 | unhappy (invalid input) |
| FR-116 | Extended Spell MUST be decided over every `SpellDuration` variant. `timed`: converted to seconds (1 round is 6 seconds, so 10 rounds qualify), refused below 1 minute, doubled and capped at 24 hours otherwise, refused `DurationAlreadyAtMaximum` when already 24 hours or longer (never shortened). `instantaneous`, `untilDispelled`, `untilDispelledOrTriggered`, `special`: refused `DurationNotEligible` (no doubling in the SRD). Package decision for the 24 hour and non-timed cases. | SRD 5.2.1, "Extended Spell" ("1 minute or longer ... a maximum duration of 24 hours"), p. 67, interpreted | unhappy (boundary) |
| FR-117 | Distant Spell MUST be decided over every `SpellRange` variant. `distance` of 5 feet or more (or any miles): doubled; below 5 feet: refused; `touch`: 30 feet; `self`: refused; `sight`, `unlimited`, `special`: refused `RangeNotEligible` (no distance to double). | SRD 5.2.1, "Distant Spell", p. 67, interpreted | unhappy (boundary) |
| FR-118 | Transmuted Spell on a spell dealing several damage types MUST change only the type the caller names, leaving the others; naming a type the spell does not deal is refused (`DamageTypeNotTransmutable`). A target equal to another type the spell already deals is accepted. | SRD 5.2.1, "Transmuted Spell" ("change that damage type to one of the other listed types"), p. 67, interpreted | unhappy (rule interaction) |
| FR-120 | Careful Spell MUST be refused when the chosen count exceeds the number of other creatures forced to save, as Empowered is above the dice (FR-069). | SRD 5.2.1, "Careful Spell" ("choose a number of those creatures"), pp. 66-67 | unhappy (boundary) |
| FR-123 | Every Metamagic eligibility check MUST take a `MetamagicSpellFacts`; a fact marked unknown MUST give a `SpellFactUnknown` refusal naming it (`MetamagicIneligibility.SpellFactUnknown`). No silent default. | `doc/rules/architecture.md` §4 (handoffs), §6; content-spells.md `Spell`; spellcasting.md `CastingContext` | unhappy (missing content) |

**Table M: Metamagic option cases.** Each row is one test. Source: the option's entry in SRD 5.2.1, "Metamagic Options",
pp. 66-67 (Careful pp. 66-67, others p. 67); rows M-53 to M-60 also "Level 2: Metamagic", "Level 7: Sorcery Incarnate", p. 66.

| Case | Option | Input | Expected | FR | Path |
|---|---|---|---|---|---|
| M-1 | Careful | Charisma modifier +3, 3 creatures, 5 other creatures save | accepted | FR-064 | happy |
| M-2 | Careful | modifier +3, 4 creatures | refused (over cap) | FR-065 | unhappy (boundary) |
| M-3, M-5 | Careful | modifier 0 or -1, 1 creature | accepted (minimum one) | FR-065 | unhappy (boundary) |
| M-4 | Careful | modifier 0, 2 creatures | refused | FR-065 | unhappy (boundary) |
| M-6 | Empowered | modifier +2, rerolling 2 of 8 dice | accepted | FR-068 | happy |
| M-7 | Empowered | modifier +2, rerolling 3 | refused | FR-069 | unhappy (boundary) |
| M-8 | Empowered | modifier 0, rerolling 1 | accepted | FR-068 | unhappy (boundary) |
| M-9 | Empowered | rerolling 2 of 1 die | refused | FR-069 | unhappy (boundary) |
| M-10 | Distant | range 5 feet | 10 feet | FR-066 | unhappy (boundary) |
| M-11 | Distant | range 4 feet | refused | FR-067 | unhappy (boundary) |
| M-12 | Distant | Touch | 30 feet | FR-066 | happy |
| M-13 | Distant | Self | refused | FR-067 | unhappy (invalid input) |
| M-14 | Extended | duration 1 minute | 2 minutes | FR-070 | unhappy (boundary) |
| M-15 | Extended | 1 round (6 seconds) | refused | FR-071 | unhappy (boundary) |
| M-16, M-17 | Extended | 12 hours; 16 hours | 24 hours (maximum) | FR-071 | unhappy (boundary) |
| M-18 | Quickened | only a cantrip cast earlier this turn (`levelOnePlusSpellCastThisTurn` false) | accepted | FR-075 | unhappy (rule interaction) |
| M-19 | Quickened | a level 1 spell cast earlier this turn (flag true), with a slot or feature-granted | refused | FR-075, FR-113 | unhappy (rule interaction) |
| M-20 | Quickened | level 1 spell attempted after Quickened this turn | refused by `checkQuickenedRestriction` | FR-075 | unhappy (rule interaction) |
| M-21 | Transmuted | Fire to Thunder | accepted | FR-079 | happy |
| M-22, M-23 | Transmuted | Fire to Fire; Force to Fire | refused | FR-080 | unhappy (invalid input) |
| M-24 | Careful | modifier +3, 3 creatures, only 2 other creatures save | refused | FR-120 | unhappy (boundary) |
| M-25, M-26 | Careful | count 0 or -1 | refused `CountOutOfRange` | FR-115 | unhappy (boundary; invalid input) |
| M-27 | Careful | spell forces no saving throw | refused | FR-065 | unhappy (invalid input) |
| M-28, M-29 | Empowered | rerolling 0 or -1 | refused `CountOutOfRange` | FR-115 | unhappy (boundary; invalid input) |
| M-30 | Extended | 10 minutes | 20 minutes | FR-070 | happy |
| M-31 | Extended | 10 rounds (1 minute) | 2 minutes | FR-116 | unhappy (boundary) |
| M-32 to M-35 | Extended | Instantaneous; Until dispelled; Until dispelled or triggered; Special | refused `DurationNotEligible` | FR-116 | unhappy (boundary) |
| M-36 | Extended | 24 hours | refused `DurationAlreadyAtMaximum` | FR-116 | unhappy (boundary) |
| M-37 | Extended | 10 days | refused, never shortened | FR-116 | unhappy (boundary) |
| M-38 | Extended | Concentration spell, 1 minute | 2 minutes and Advantage on Concentration saves | FR-070 | happy |
| M-39 to M-41 | Distant | Sight; Unlimited; Special | refused `RangeNotEligible` | FR-117 | unhappy (boundary) |
| M-42 | Distant | 1 mile | 2 miles | FR-117 | unhappy (boundary) |
| M-43 | Heightened | spell forces a saving throw | accepted, one target has Disadvantage | FR-072 | happy |
| M-44 | Heightened | spell forces no saving throw | refused `NoSavingThrow` | FR-072 | unhappy (invalid input) |
| M-45 | Subtle | Verbal and Somatic components | accepted, both removed | FR-078 | happy |
| M-46 | Subtle | Verbal and a Material component with a cost | accepted, Verbal removed, Material kept | FR-078 | unhappy (boundary) |
| M-47 | Subtle | only a consumed Material component | refused `NothingToRemove` | FR-078 | unhappy (boundary) |
| M-48 | Subtle | no components | refused `NothingToRemove` | FR-078 | unhappy (empty/minimal) |
| M-49 to M-51 | Quickened | casting time Bonus Action; Reaction; 1 minute | refused | FR-074 | unhappy (invalid input) |
| M-52 | Quickened | casting time an action, no level 1+ spell this turn | accepted | FR-073 | happy |
| M-53 | Distant then Empowered | one option applied, then Empowered | accepted | FR-111 | unhappy (rule interaction) |
| M-54 | Empowered then Distant | no Sorcery Incarnate | refused | FR-111 | unhappy (rule interaction) |
| M-55 | Distant then Extended | no Sorcery Incarnate | refused | FR-062 | unhappy (rule interaction) |
| M-56 | Distant then Extended | Sorcery Incarnate (level 7, Innate Sorcery active) | accepted | FR-045 | happy |
| M-57 | Distant, Extended, then Empowered | Sorcery Incarnate | accepted (Empowered rides on top) | FR-112 | unhappy (rule interaction) |
| M-58 | Distant, Extended, then Quickened | Sorcery Incarnate | refused (third non-exception option) | FR-112 | unhappy (rule interaction) |
| M-59 | Distant, Empowered, then Seeking | no Sorcery Incarnate | accepted | FR-112 | unhappy (rule interaction) |
| M-60 | Empowered then Empowered | any | refused (duplicate) | FR-112 | unhappy (rule interaction) |
| M-61 | Careful on Command (always prepared, not on the Sorcerer list) | Draconic Spells granted Command | accepted as a spell the character casts | FR-103 | unhappy (rule interaction) |
| M-62 | any | a required spell fact marked unknown | refused `SpellFactUnknown` | FR-123 | unhappy (missing content) |
| M-63 | Transmuted | spell deals Fire and Cold, naming Fire, to Lightning | accepted, Cold unchanged | FR-118 | unhappy (rule interaction) |
| M-64 | Transmuted | spell deals Fire, naming Acid | refused `DamageTypeNotTransmutable` | FR-118 | unhappy (invalid input) |
| M-65 | free use | requested at level 20, Innate Sorcery inactive | refused, points spent normally | FR-046 | unhappy (state) |

### 4.9 Sorcerous Restoration, Ability Score Improvement, Epic Boon

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-083 | On finishing a Short Rest (`RestKind.ShortRest`, FR-107) at level 5+, the character MUST be able to regain Sorcery Points up to half its Sorcerer level (rounded down), never above the maximum (Table R). | SRD 5.2.1, "Level 5: Sorcerous Restoration", p. 66; rests-and-hit-dice.md FR-001, FR-060 | happy |
| FR-084 | Sorcerous Restoration MUST be refused below level 5, if already used since the last Long Rest, and for an amount above the cap or above the points missing (FR-108); a Long Rest MUST make it available again. A refused attempt MUST NOT consume the use (FR-106). | SRD 5.2.1, "Level 5: Sorcerous Restoration", p. 66 | unhappy (state) |
| FR-085 | The Sorcerer MUST be granted one feat choice at levels 4, 8, 12 and 16 (Ability Score Improvement or another feat it qualifies for) and none elsewhere. The grant is data (`GrantsFeat`, `FeatCategory` General, `ClassDefinition.FeatGrantsAt`, content-classes.md FR-049) reported by `ApplySorcererLevelUp` in `SorcererLevelGrants.FeatGrants` (FR-134); applying the feat is feats-behaviour.md's. | SRD 5.2.1, "Level 4: Ability Score Improvement", p. 66; content-classes.md FR-049 | happy |
| FR-086 | One Epic Boon feat choice MUST be granted at level 19 and none at levels 1-18 or 20; reported as `FeatCategory` Epic Boon in `SorcererLevelGrants.FeatGrants` (FR-134). | SRD 5.2.1, "Level 19: Epic Boon", p. 66; content-classes.md FR-049 | happy |
| FR-106 | A refused operation MUST leave the whole Sorcerer state unchanged, including once-per-rest flags: refused Sorcerous Restoration, Dragon Wings restore or Dragon Companion cast do not consume their use, and a zero-gain attempt is a refusal. Package decision. | `doc/rules/testing.md` §4 (state errors); SRD 5.2.1, "Level 5: Sorcerous Restoration" ("Once you use this feature"), p. 66, interpreted | unhappy (state) |
| FR-107 | The rest MUST be the rests-and-hit-dice.md `RestKind` (shared-types-and-ownership.md FR-009), values exactly `shortRest` and `longRest` (rests-and-hit-dice.md FR-001, FR-059). `applySorcererRest` MUST accept both (FR-060 there): `shortRest` runs the regain; `longRest` MUST NOT regain points through this feature (FR-049 restores all) but MUST make it available again, restore Innate Sorcery (FR-042), Dragon Wings and Dragon Companion (FR-093, FR-095) and remove every created slot (FR-054); asking for the regain with `longRest` MUST be refused `RestorationNotAShortRest`. | rests-and-hit-dice.md FR-001, FR-059, FR-060; shared-types-and-ownership.md FR-009; SRD 5.2.1, "Level 5: Sorcerous Restoration", p. 66 | unhappy (state) |
| FR-108 | A requested Restoration amount above the points missing MUST be refused, never clamped (as FR-051); a request of 0 or below, or above the cap (half the level, rounded down), MUST also be refused. Package decision. | SRD 5.2.1, "Level 5: Sorcerous Restoration" ("no more than ... half your Sorcerer level"), p. 66, interpreted | unhappy (boundary) |

**Table R: Sorcerous Restoration cap.** Source: SRD 5.2.1, "Level 5: Sorcerous Restoration", p. 66; R-6 and R-7 apply FR-108.

| Case | Sorcerer level | Points missing | Requested | Expected | FR | Path |
|---|---|---|---|---|---|---|
| R-1 | 5 | 5 | 2 | regains 2 | FR-083 | happy |
| R-2 | 5 | 5 | 3 | refused (cap 2) | FR-084 | unhappy (boundary) |
| R-3 | 4 | 4 | 2 | refused (feature not gained) | FR-084 | unhappy (boundary) |
| R-4 | 7 | 7 | 3 | regains 3 (half of 7 rounds down) | FR-083 | unhappy (boundary) |
| R-5 | 20 | 20 | 10 | regains 10 | FR-083 | unhappy (boundary) |
| R-6 | 20 | 4 | 10 | refused (above the 4 missing; not clamped), use not consumed | FR-108 | unhappy (boundary) |
| R-7 | 5 | 0 | 1 | refused (nothing missing), use not consumed | FR-108 | unhappy (state) |
| R-8 | 20 | 4 | 4 | regains 4 | FR-083 | unhappy (boundary) |
| R-9 | 5 | 5 | 0 | refused (below 1) | FR-108 | unhappy (boundary) |
| R-10 | 5 | 5 | 2 with `RestKind.LongRest` | refused `RestorationNotAShortRest` | FR-107 | unhappy (state) |
| R-11 | 5 | 5 | 2, then 1 after a refused attempt, same rest | refused attempt leaves the use available; a later valid request accepted | FR-106 | unhappy (state) |
| R-12 | 5 | 5 | 2 accepted, then 1 before a Long Rest | refused `RestorationAlreadyUsed` | FR-084 | unhappy (state) |

### 4.10 Draconic Sorcery

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-087 | `draconicResilienceBonus()` MUST return the `HitPointMaximumBonus` (derived-stats.md FR-029, FR-051) with `flat` 3, `perLevel` 1, `grantedAtLevel` 3 and class scope `sorcerer`: 3 added on gaining the feature at level 3, 1 more per further Sorcerer level (total equals the level, 3 at level 3). `ApplySorcererLevelUp` MUST report it in `SorcererLevelGrants` for a Draconic Sorcerer of level 3+ (FR-134); the caller supplies it to derived-stats.md. | SRD 5.2.1, "Level 3: Draconic Resilience", p. 69 | happy |
| FR-088 | `draconicArmorClassBasis()` MUST return the `AbilityBasedArmorClassBasis` candidate (10 + Dexterity modifier + Charisma modifier, `abilities` {Charisma}, `shieldAllowed` true, `id` the `FeatureID` `draconic_resilience`) for `alternativeBases` (derived-stats.md FR-036, FR-037). It applies only with no body armor worn; a wielded Shield does not disable it (decided in derived-stats.md; its row "Sorcerer (Draconic), trained Shield" = 17). | SRD 5.2.1, "Level 3: Draconic Resilience", p. 69; derived-stats.md FR-036 | unhappy (rule interaction) |
| FR-089 | Draconic Spells MUST add Alter Self, Chromatic Orb, Command and Dragon's Breath at level 3, Fear and Fly at 5, Arcane Eye and Charm Monster at 7, Legend Lore and Summon Dragon at 9, as always-prepared spells (FR-032, FR-103). They are content: the `AlwaysPreparedSpells` fact (content-classes.md FR-055 to FR-059, Table FA). `ApplySorcererLevelUp` MUST report the `SpellID`s newly granted at the gained level in `SorcererLevelGrants` (FR-134), for spellcasting.md. | SRD 5.2.1, "Draconic Spells", p. 70 | happy |
| FR-090 | Elemental Affinity (level 6) MUST record exactly one damage type from Acid, Cold, Fire, Lightning, Poison (a domain constant); a choice outside the list or a second choice MUST be refused. | SRD 5.2.1, "Level 6: Elemental Affinity", p. 70 | unhappy (invalid input) |
| FR-091 | With a chosen type the character MUST have Resistance to it (published in `SorcererActiveEffects.Resistances`, FR-135), and when casting a spell that deals damage of that type MUST be able to add the Charisma modifier to one damage roll of that spell; a spell dealing no damage of that type, or a second addition to the same spell, MUST be refused. Independent of the spell's class list. | SRD 5.2.1, "Level 6: Elemental Affinity", p. 70 | happy |
| FR-092 | Dragon Wings (level 14, a Bonus Action) MUST grant a Fly Speed of 60 feet until dismissed (no action) or 1 hour ends, as host-issued commands; the Fly Speed is published in `SorcererActiveEffects.FlySpeedFeet` while active (FR-135). | SRD 5.2.1, "Level 14: Dragon Wings", p. 70 | happy |
| FR-093 | Dragon Wings MUST be refused after use until a Long Rest unless the character spends 3 Sorcery Points (no action) to restore the use; a Long Rest MUST restore it. | SRD 5.2.1, "Level 14: Dragon Wings", p. 70 | unhappy (state) |
| FR-094 | The 3 point restoration MUST be refused with fewer than 3 points (2 refused, 3 accepted), when the use is not expended, and below level 14. | SRD 5.2.1, "Level 14: Dragon Wings", p. 70 | unhappy (boundary) |
| FR-095 | Dragon Companion (level 18) MUST let the character cast Summon Dragon without a Material component, and once without a spell slot until the next Long Rest; a second slot-free cast MUST be refused until a Long Rest. The host casts with `CastMode.FeatureGranted` (casting-and-concentration.md FR-121); this function tracks only the use. | SRD 5.2.1, "Level 18: Dragon Companion", p. 70 | happy |
| FR-096 | Casting Summon Dragon with Dragon Companion, the character MUST be able to choose that it does not require Concentration, its duration then 1 minute for that casting (Concentration: casting-and-concentration.md, shared-types-and-ownership.md FR-011). | SRD 5.2.1, "Level 18: Dragon Companion", p. 70 | happy |
| FR-104 | Elemental Affinity MUST apply to a spell the character casts that is not on the Sorcerer list (a Wizard-list spell through multiclass levels, another feature's spell) when it deals damage of the chosen type, regardless of spellcasting ability or class list. | SRD 5.2.1, "Level 6: Elemental Affinity" ("when you cast a spell that deals damage of that type"), p. 70 | unhappy (boundary) |
| FR-105 | The damage type compared with the chosen type MUST be the type the damage roll actually deals, after Transmuted Spell: a Fire spell Transmuted to Cold gains the bonus for a Cold affinity, not a Fire one. Package decision (the SRD does not order them). | SRD 5.2.1, "Level 6: Elemental Affinity"; "Transmuted Spell", pp. 67, 70, interpreted | unhappy (rule interaction) |

**Table E: Elemental Affinity cases.** Source: SRD 5.2.1, "Level 6: Elemental Affinity", p. 70 (E-4, E-5 also "Transmuted Spell", p. 67).

| Case | Chosen type | Spell and damage | Expected | FR | Path |
|---|---|---|---|---|---|
| E-1 | Fire | Sorcerer-list spell dealing Fire | Charisma modifier added to one damage roll | FR-091 | happy |
| E-2 | Fire | spell dealing Cold | refused `ElementalAffinityTypeMismatch` | FR-091 | unhappy (invalid input) |
| E-7 | Fire | spell dealing no damage | refused `ElementalAffinityTypeMismatch` | FR-091 | unhappy (empty/minimal) |
| E-3 | Fire | Wizard-list spell dealing Fire, cast via multiclass Wizard levels | added | FR-104 | unhappy (boundary) |
| E-4 | Fire | Fire spell Transmuted to Cold | refused (type after change is Cold) | FR-105 | unhappy (rule interaction) |
| E-5 | Cold | Fire spell Transmuted to Cold | added | FR-105 | unhappy (rule interaction) |
| E-6 | Fire | second addition to the same spell | refused `ElementalAffinityAlreadyApplied` | FR-091 | unhappy (state) |
| E-8 | none chosen | Fire spell | refused (no chosen type) | FR-090 | unhappy (state) |

### 4.11 Sorcerer spell list, content and cross-cutting behaviour

The list is **derived, not stored** (content-spells.md FR-120; oracle: content-spells.md FR-022, FR-023, FR-097). The authoritative
membership is the header-derived 140 (content-classes-data.md FR-101); Mind Spike and Phantasmal Force are on no printed list. The table
is the printed list (SRD 5.2.1, "Sorcerer Spell List", pp. 67-69, **138 spells**: cantrips 16, then levels 1-9: 21, 25, 21, 13, 12,
11, 8, 6, 5), an oracle only, never a rule that the list is exactly these. No second copy outside `testdata/srd/` (FR-033).

| Level | Spells |
|---|---|
| Cantrip | Acid Splash, Chill Touch, Dancing Lights, Elementalism, Fire Bolt, Light, Mage Hand, Mending, Message, Minor Illusion, Poison Spray, Prestidigitation, Ray of Frost, Shocking Grasp, Sorcerous Burst, True Strike |
| 1 | Burning Hands, Charm Person, Chromatic Orb, Color Spray, Comprehend Languages, Detect Magic, Disguise Self, Expeditious Retreat, False Life, Feather Fall, Fog Cloud, Grease, Ice Knife, Jump, Mage Armor, Magic Missile, Ray of Sickness, Shield, Silent Image, Sleep, Thunderwave |
| 2 | Alter Self, Blindness/Deafness, Blur, Darkness, Darkvision, Detect Thoughts, Dragon's Breath, Enhance Ability, Enlarge/Reduce, Flame Blade, Flaming Sphere, Gust of Wind, Hold Person, Invisibility, Knock, Levitate, Magic Weapon, Mirror Image, Misty Step, Scorching Ray, See Invisibility, Shatter, Spider Climb, Suggestion, Web |
| 3 | Blink, Clairvoyance, Counterspell, Daylight, Dispel Magic, Fear, Fireball, Fly, Gaseous Form, Haste, Hypnotic Pattern, Lightning Bolt, Major Image, Protection from Energy, Sleet Storm, Slow, Stinking Cloud, Tongues, Vampiric Touch, Water Breathing, Water Walk |
| 4 | Banishment, Blight, Charm Monster, Confusion, Dimension Door, Dominate Beast, Fire Shield, Greater Invisibility, Ice Storm, Polymorph, Stoneskin, Vitriolic Sphere, Wall of Fire |
| 5 | Animate Objects, Arcane Hand, Cloudkill, Cone of Cold, Creation, Dominate Person, Hold Monster, Insect Plague, Seeming, Telekinesis, Teleportation Circle, Wall of Stone |
| 6 | Chain Lightning, Circle of Death, Disintegrate, Eyebite, Flesh to Stone, Freezing Sphere, Globe of Invulnerability, Mass Suggestion, Move Earth, Sunbeam, True Seeing |
| 7 | Delayed Blast Fireball, Etherealness, Finger of Death, Fire Storm, Plane Shift, Prismatic Spray, Reverse Gravity, Teleport |
| 8 | Demiplane, Dominate Monster, Earthquake, Incendiary Cloud, Power Word Stun, Sunburst |
| 9 | Gate, Meteor Swarm, Power Word Kill, Time Stop, Wish |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-099 | Each use case needing the Sorcerer definition MUST load it through `content.Source` by `ClassID`. A `ClassID` content cannot find MUST yield the use case's own `<UseCase>ContentMissing` variant carrying `MissingContent(kind, id)` (shared-types-and-ownership.md FR-017, FR-018), never a shared variant or an error. It covers only that lookup: `SubclassID` and `FeatureID` have no `ContentKind`, and the Metamagic `FeatureOption`s and Creating Spell Slots `FeatureTable` are nested in the loaded class, so cannot be absent (the decoder rejects such data, content-classes.md FR-065 to FR-067). A Metamagic option id the host supplies that the definition does not list MUST be `MetamagicOptionNotKnown` (`UseMetamagic`) or `MetamagicSelectionInvalid` (`ChooseMetamagicOptions`). A content source returning a typed `*srderr.PortError` MUST propagate, state unchanged. | `doc/rules/architecture.md` §4, §6; content-source-port.md; content-classes.md FR-065 to FR-067; shared-types-and-ownership.md FR-017, FR-018 | unhappy (collaborator failure) |
| FR-100 | Every operation MUST return a new state and events and leave the input state unchanged; collections exposed on the state MUST be read-only (accessors return copies). | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-101 | Every operation MUST be deterministic: same state and inputs give the same result and events, with no dice, clock or global state; rolls (Empowered, Seeking) arrive as host values. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (determinism) |

### 4.12 Use cases, level-up and telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-124 | Each of the four use cases (section 6) MUST record one span `srd5e.character.<use_case>`, one increment of the shared counter `srd5e.character.feature_used` and one measurement of the shared histogram `srd5e.character.feature_used.duration` (via `telemetry.StartTimer`), signals owned by character-aggregate.md FR-118 (shared-types-and-ownership.md FR-063). Attributes: `srd5e.class` `sorcerer`, `srd5e.feature` (section 7), and on span and counter `srd5e.outcome` (`success` or `failure`; FR-041); the histogram carries class and feature only. Every returned result also sets `srd5e.character.result` on the span: `changed` (`...Applied`), `rejected` (`...Rejected`), `content_missing` (`...ContentMissing`). A rejection sets outcome `failure`, result `rejected` and `srd5e.character.rejection` (snake_case of the `SorcererRejectionKind` member) on span and counter, and no error status. A `...ContentMissing` result sets outcome `failure`, result `content_missing`, span error status and rejection `content_missing`, no `error.type` (errors-and-results.md FR-043). The domain functions are silent. | `doc/rules/observability.md` §5, §6, §7; shared-types-and-ownership.md FR-041, FR-042, FR-063; character-aggregate.md FR-071, FR-072, FR-118 | happy |
| FR-125 | A panic in a use case (a broken invariant from a `MustX` helper or an unreachable branch) MUST propagate unchanged (never recovered into a result), be recorded on the span, increment no outcome counter and be logged nowhere. A port failure (`*srderr.PortError`) MUST be recorded once: span error status with the error, one increment of `feature_used` with outcome `failure` (no `error.type`, no rejection, no result) and one increment of `srd5e.character.feature_used.errors` with `error.type` set to its `errorType` (`content_error`). It MUST NOT be logged or double counted, the duration MUST still be recorded, and it MUST propagate. | `doc/rules/observability.md` §5, §7; character-aggregate.md FR-118; errors-and-results.md section 7 | unhappy (collaborator failure) |
| FR-126 | Each use case MUST return an equal result with the no-op `telemetry.Telemetry` and a recording one, and emit no log (shared-types-and-ownership.md FR-043). | `doc/rules/observability.md` §7, §9; spellcasting.md FR-086, FR-088 | unhappy (determinism) |
| FR-127 | Applying a Sorcerer level MUST reuse the aggregate's `LevelGained` event (character-aggregate.md FR-025, FR-040) as trigger and emit no second level event. `LevelUpCharacter` drives the level and Hit Points; `ApplySorcererLevelUp` then applies the Sorcerer results: Sorcery Points maximum and current (FR-047), owed Metamagic options (FR-035) and the `SorcererLevelGrants` of FR-134 (FR-087, FR-085, FR-086, FR-089). The subclass choice due at level 3 is reported by level-advancement-and-multiclassing.md. | character-aggregate.md FR-025, FR-040, FR-092; `doc/rules/architecture.md` §4 | happy |
| FR-128 | `SorcererState` MUST be a `FeatureState` stored in the character's `FeatureStates` under one typed `FeatureStateKey[SorcererState]` named `sorcerer`; the aggregate does not interpret it (character-aggregate.md FR-092, FR-093). Every operation is a pure function from `SorcererState` and explicit inputs to a new state and events; the caller writes it back with `Character.WithFeatureState`. No `SorcererState` field is added to `Character`, and no command carries one beside the `Character`. | shared-types-and-ownership.md FR-022, FR-023; character-aggregate.md FR-092 | happy |
| FR-129 | `SorcererCreationChoices` MUST be a zero-field `ClassCreationChoices` (character-aggregate.md FR-094) whose pure seeding method returns the `FeatureStates` with a level 1 `SorcererState` (0 Sorcery Points, 2 Innate Sorcery uses, no Metamagic, no created slots, no once-per-rest use). Sorcerer has no class-specific creation choice, so it carries no skills, equipment or parse result (character-creation.md FR-016 to FR-021, FR-102, FR-104 own those and store the choices uninterpreted). A multiclass Sorcerer is seeded by FR-011. | shared-types-and-ownership.md FR-024; character-aggregate.md FR-094; character-creation.md FR-102, FR-104 | happy |
| FR-130 | Each use case MUST return its own closed `<UseCase>Result` whose variants carry the operation prefix (`UseMetamagicApplied`, an error, `UseMetamagicContentMissing`). A rejection is a `SorcererRejection` field of the `...Rejected` variant; a missing definition a `MissingContent` field of `...ContentMissing`. `SorcererRejection` is one value object with enum `SorcererRejectionKind` and typed payload, so no rejection variant name is exported (unprefixed names such as `SkillsInvalid`, `CastingTimeNotAction` collide with class-wizard.md); `MetamagicNotEligible` reasons are the enum `MetamagicIneligibility`. Domain functions return the closed `SorcererTransition` (`SorcererChanged(state, events)` or `SorcererRefused(rejection)`). Members of `InnateSorceryPayment`, `DragonWingsAction`, `SlotConversionSource` are prefixed with their operation. No generic variant name and no name defined by another spec is exported. | shared-types-and-ownership.md FR-017, FR-018, FR-021; `doc/rules/architecture.md` §9 | unhappy (name collision) |
| FR-131 | This spec MUST define no `Subclass` with a nil error, an error, `RestApplied`, `FightingStyleChoice`, `WeaponMasteryChoice`, `ClassDataInvalid` or `ContentMissing` type; it references the owners. Its events (section 5) are past-tense value objects in one closed family `SorcererEvent` implementing `CharacterEvent` (character-aggregate.md FR-085, an open interface declaring `characterID`). Every `SorcererEvent` MUST carry the `CharacterID` on the closed supertype. The family is closed within its own library (class-cleric.md FR-136 settles the same case). | shared-types-and-ownership.md FR-019, FR-020; character-aggregate.md FR-085 | unhappy (duplicate) |
| FR-132 | This spec MUST NOT claim a content `schemaVersion`. Every shape the Sorcerer needs is registered by content-classes.md: Metamagic `FeatureOption` with `pointCost` (FR-061 to FR-065), the Creating Spell Slots `FeatureTable` (FR-066, FR-067), the `AlwaysPreparedSpells` fact (FR-055 to FR-059, Table FA). The Elemental Affinity types are a domain constant (D3 there allows three fact kinds); per-spell Metamagic facts are host input. Bundled data reaches consumers only through `BundledContentSource` and the one `ContentSetDecoder` (shared-types-and-ownership.md FR-030, FR-031). | shared-types-and-ownership.md FR-030, FR-031, FR-034, FR-035; content-classes.md FR-055 to FR-067 | happy |
| FR-133 | `innateSorceryEffects(SorcererState)` MUST be a pure function returning `InnateSorceryEffects`: while active a spell save DC `Modifier` of +1 and an `Advantage` (`RollCircumstance`, modifiers-and-advantage.md FR-042) for attack rolls of Sorcerer spells (FR-103); inactive, a `Modifier` of 0 and no circumstance. The host applies the effects to the spells FR-103 names and adds the modifier to the DC `CalculateSpellcasting` returns (it accepts none; settled, section 9). | SRD 5.2.1, "Level 1: Innate Sorcery", pp. 65-66; modifiers-and-advantage.md FR-042 | happy |
| FR-134 | `sorcererLevelGrants` MUST be a pure function returning `SorcererLevelGrants` for the level just gained: the feat grants (`FeatCategory` list from `ClassDefinition.FeatGrantsAt`, content-classes.md FR-049: General at 4, 8, 12, 16, Epic Boon at 19), the Draconic Resilience `HitPointMaximumBonus` (FR-087) for a Draconic Sorcerer of level 3+, and the `SpellID`s the subclass's `AlwaysPreparedSpells` fact newly grants at that level (FR-089). `ApplySorcererLevelUp` carries it in its `Applied` variant, so every grant is a named field. | content-classes.md FR-049, FR-055 to FR-059; derived-stats.md FR-029, FR-051 | happy |
| FR-135 | `sorcererActiveEffects(SorcererState)` MUST be a pure function returning `SorcererActiveEffects`: the `DamageType`s resisted (the chosen Elemental Affinity type, FR-091; empty before a choice) and the Fly Speed in feet (60 while Dragon Wings is active, FR-092; absent otherwise). Nothing here reads a `combat` type. | SRD 5.2.1, "Level 6: Elemental Affinity", "Level 14: Dragon Wings", p. 70 | happy |
| FR-136 | A created slot (FR-052, FR-054) MUST be a count per level in `SorcererState.CreatedSlots`, never in `SpellSlots` (whose `remaining` cannot exceed `maximum`, spellcasting.md FR-019). It is a real spell slot, so it can cast a spell of its level or lower, at the slot's level, and the one-slot-per-turn rule applies. `spendCreatedSlot(state, castingContext, createdSlotLevel, spellLevel)` (spell level 1-9, else `srderr.ErrInvalidInput`, FR-114) MUST remove one created slot of that level and emit `CreatedSlotSpent`; refused `CreatedSlotUnavailable` when none exists, `CreatedSlotLevelTooLow` when the spell's level exceeds the slot's, `CreatedSlotSpentThisTurn` when `slotSpentThisTurn` is true. The host then casts with `CastMode.FeatureGranted`, `featureCastLevel` = the created slot's level and `countsAsSlotSpell` = true (casting-and-concentration.md FR-144), and sets `slotSpentThisTurn` from the report. `convertSpellSlot` MAY convert a created slot (`ConvertCreatedSlot`). | SRD 5.2.1, "Creating Spell Slots", p. 66; "One Spell with a Spell Slot per Turn", "Using a Higher-Level Spell Slot", p. 105; spellcasting.md FR-019 | unhappy (state) |
| FR-137 | A function that depends on the Sorcerer level or chosen subclass MUST take both as an explicit `SorcererProfile` (a `SorcererLevel` from `ClassLevels` and a `SubclassID?` from `ChosenSubclass`, level-advancement-and-multiclassing.md Q-10); `SorcererState` holds neither, so each has one owner. Functions needing neither (`endInnateSorcery`, `startSorcererTurn`, `checkQuickenedRestriction`, `spendCreatedSlot`) take no profile. | shared-types-and-ownership.md FR-008, FR-020, FR-023; derived-stats.md `ClassLevels`; level-advancement-and-multiclassing.md Q-10 | happy |

**Withdrawn ids.** A retired id duplicated spellcasting.md (see FR-102).

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (Sorcerer `ClassDefinition`: Primary Ability Charisma) |
| FR-002 | Withdrawn: moved to content-classes.md (Sorcerer `ClassDefinition`: Hit Point Die D6) |
| FR-003 | Withdrawn: moved to content-classes.md (Sorcerer `ClassDefinition`: saving throws Constitution and Charisma) |
| FR-004 | Withdrawn: moved to character-creation.md (FR-016 to FR-019; 2 distinct skills from Arcana, Deception, Insight, Intimidation, Persuasion, Religion) |
| FR-005 | Withdrawn: moved to character-creation.md (FR-016 to FR-018; invalid skill choice refused there) |
| FR-006 | Withdrawn: moved to content-classes.md (Sorcerer `ClassDefinition`: Simple weapons, no Armor Training) |
| FR-007 | Withdrawn: moved to content-classes.md (Sorcerer `ClassDefinition`: starting equipment option A) |
| FR-008 | Withdrawn: moved to content-classes.md (Sorcerer `ClassDefinition`: starting equipment option B, 50 GP) |
| FR-009 | Withdrawn: moved to character-creation.md (FR-020; unknown equipment option refused there) |
| FR-012 | Withdrawn: moved to content-classes.md (Sorcerer `ProgressionRow`s, Class Features column of Table P) |
| FR-013 | Withdrawn: moved to content-classes.md (Sorcerer `ProgressionRow`s, Sorcery Points column of Table P) |
| FR-014 | Withdrawn: moved to spellcasting.md (FR-014; retired duplicate, see FR-102) |
| FR-015 | Withdrawn: moved to spellcasting.md (FR-015; retired duplicate, see FR-102) |
| FR-016 | Withdrawn: moved to spellcasting.md (FR-010; retired duplicate, see FR-102) |
| FR-020 | Withdrawn: moved to level-advancement-and-multiclassing.md (subclass choice, `Subclass` with a nil error) |
| FR-021 | Withdrawn: moved to level-advancement-and-multiclassing.md (pending subclass choice) |
| FR-022 | Withdrawn: moved to level-advancement-and-multiclassing.md (an error for level 1 or 2) |
| FR-023 | Withdrawn: moved to level-advancement-and-multiclassing.md (a missing subclass id is a `MissingContent` result) |
| FR-025 | Withdrawn: moved to spellcasting.md (FR-002; Charisma as spellcasting ability) |
| FR-026 | Withdrawn: moved to spellcasting.md (FR-068; Arcane Focus as Spellcasting Focus) |
| FR-027 | Withdrawn: moved to spellcasting.md (FR-052, FR-054; cantrip selection validity) |
| FR-028 | Withdrawn: moved to spellcasting.md (FR-052, FR-054; cantrip rejection with all problems) |
| FR-029 | Withdrawn: moved to spellcasting.md (FR-053, FR-014; cantrip replacement and counts) |
| FR-030 | Withdrawn: moved to spellcasting.md (FR-040 to FR-042; prepared list validity) |
| FR-031 | Withdrawn: moved to spellcasting.md (FR-040 to FR-044, FR-054; prepared list rejection) |
| FR-033 | Withdrawn: moved to spellcasting.md (FR-048, FR-049; one prepared-spell replacement per level) |
| FR-034 | Withdrawn: moved to spellcasting.md (FR-040, FR-041, FR-046, FR-049; replacement rejections) |
| FR-036 | Withdrawn: moved to spellcasting.md (FR-025; Long Rest restores every slot) |
| FR-097 | Withdrawn: moved to content-spells.md (FR-097, printed-list oracle; FR-120, derived list) and content-classes-data.md (FR-101, counts) |
| FR-098 | Withdrawn: moved to content-classes.md (class decoder reports every problem at once; all-class check is FR-035 there) |
| FR-119, FR-121, FR-122 | Never allocated: no requirement ever held these ids; nothing was withdrawn |

## 5. Model

**Content (context `content`, content-classes.md).** This spec defines no content type; it states what the data carries
(shared-types-and-ownership.md FR-025; this spec FR-132): the Sorcerer `ClassDefinition` (traits, 20 `ProgressionRow`s, Metamagic
`FeatureOption`s with `pointCost`, the Creating Spell Slots `FeatureTable`, subclass ids) and the Draconic Sorcery
`SubclassDefinition` with its `AlwaysPreparedSpells` fact. The spell list is not carried. The Elemental Affinity types and
the Transmuted Spell list are domain constants. Vocabulary and ids are `content`'s, in `snake_case` (FR-005, FR-007).

**Character context (domain)**

- `SorcererLevel`, `SorceryPoints` (current, maximum): value objects with constructor invariants (FR-018, FR-048).
  `SorcererProfile` (FR-137): level plus `SubclassID?`, built by the caller, never stored.
- `SorcererState` (a `FeatureState`, FR-128, typed key `sorcerer`): known Metamagic option ids, Sorcery Points, Innate
  Sorcery (uses, active), Sorcerous Restoration used, `createdSlots` by level (FR-054, FR-136), chosen Elemental Affinity
  `DamageType`, Dragon Wings (used, active), Dragon Companion free cast used, per-turn flags (free Metamagic spent,
  Quickened used). It holds no level, subclass, cantrips or prepared spells (`ClassLevels`, `ChosenSubclass`, spellcasting.md
  `SpellcastingState`). Compares by value (character-aggregate.md FR-093). Seeded at creation by `SorcererCreationChoices`
  (FR-129) and for a multiclass entrant by `ApplySorcererLevelUp` (FR-011).
- `SorcererRejection` (FR-130): `kind` (`SorcererRejectionKind`) plus payload (`required`, `available`, `cap`, `missing`,
  `feature`, `problems`, `ineligibility`); a field of results, not a variant. `SorcererTransition` (closed):
  `SorcererChanged(state, events)` / `SorcererRefused(rejection)`; `SorcererSlotTransition`: `SorcererSlotsChanged(state,
  spellcasting, events)` / `SorcererSlotsRefused(rejection)` for `convertSpellSlot`, which can also change the
  `SpellcastingState` (spellcasting.md FR-021).
- Handoff values: `MetamagicSpellFacts` (4.8), `MetamagicCastContext` (FR-110), `MetamagicSelection`, `SorcererLevelGrants`
  (FR-134), `InnateSorceryEffects` (FR-133), `SorcererActiveEffects` (FR-135). `MetamagicEffect` (closed, ten variants):
  `Careful`, `Distant`, `Empowered`, `Extended`, `Heightened`, `Quickened`, `Seeking`, `Subtle`, `Transmuted`, `Twinned`.
- Services: `SorcererProgression`, `MetamagicRules` (eligibility, cost; `MetamagicAllowed(effect, cost)` / `MetamagicRefused(rejection)`), `NewMetamagicSelection`.
- Events (`SorcererEvent`, FR-131): `InnateSorceryActivated`, `InnateSorceryEnded`, `SpellSlotConvertedToSorceryPoints`,
  `SpellSlotCreated`, `CreatedSlotSpent`, `CreatedSlotsExpired`, `MetamagicUsed`, `SorceryPointsRestored`,
  `MetamagicOptions` with a nil error, `ElementalAffinity` with a nil error, `DragonWingsActivated`, `DragonWingsEnded`,
  `DragonWingsUseRestored`, `DragonCompanionCast`, `ElementalAffinityApplied`. The level gain is the existing `LevelGained` (FR-127).

**Invariants and failures.** Level 1-20 (`SorcererLevel`), slot level 1-9 (spellcasting.md `SlotLevel`): `srderr.ErrInvalidInput`
(FR-018, FR-114). 0 <= points <= maximum (FR-048). Metamagic selections within entitlement: `MetamagicSelection`, a closed
result as the input is untrusted (FR-057). `MissingContent(kind, id)` only for the `ClassID` lookup (FR-099, FR-130);
`*srderr.PortError` only from the content port, propagated. Rejection kinds (snake_case in telemetry):

| Member | Operation | FRs |
|---|---|---|
| `ElementalAffinityChoiceInvalid`, `ElementalAffinityTypeMismatch`, `ElementalAffinityAlreadyApplied` | `chooseElementalAffinity`, `applyElementalAffinity` | FR-090, FR-091, FR-104, FR-105 |
| `MetamagicSelectionInvalid` (`problems`), `MetamagicReplacementInvalid` | `ChooseMetamagicOptions`, `ApplySorcererLevelUp` | FR-057, FR-058 |
| `SorcererFeatureNotGained` (`feature`) | any feature below its level | FR-019, FR-084 |
| `InnateSorceryAlreadyActive`, `InnateSorceryNoUsesLeft`, `InnateSorceryUsesRemain`, `SorceryIncarnateUnavailable`, `InnateSorceryNotActive` | `activateInnateSorcery`, `endInnateSorcery` | FR-039, FR-041, FR-043, FR-044 |
| `SorceryPointsInsufficient` (`required`, `available`) | several | FR-041, FR-053, FR-060, FR-094 |
| `NoSlotToConvert`, `SorceryPointsWouldExceedMaximum` | `convertSpellSlot` | FR-051 |
| `SlotLevelNotCreatable`, `SorcererLevelBelowMinimum` | `CreateSpellSlot` | FR-053 |
| `CreatedSlotUnavailable`, `CreatedSlotLevelTooLow`, `CreatedSlotSpentThisTurn` | `spendCreatedSlot` | FR-136 |
| `MetamagicOptionNotKnown`, `MetamagicCombinationRefused`, `QuickenedRestrictionApplies` | `UseMetamagic`, `checkQuickenedRestriction` | FR-061 to FR-063, FR-075, FR-111, FR-112 |
| `MetamagicNotEligible` (`ineligibility`: enum `MetamagicIneligibility` = `NoSavingThrow`, `RangeNotEligible`, `DurationNotEligible`, `DurationAlreadyAtMaximum`, `CastingTimeNotAction`, `NoMissedAttackRoll`, `NothingToRemove`, `DamageTypeNotTransmutable`, `NoHigherSlotTarget`, `CountOutOfRange`, `SpellFactUnknown`) | `UseMetamagic` | FR-065, FR-067, FR-069, FR-071, FR-072, FR-074, FR-077, FR-078, FR-080, FR-082, FR-115 to FR-118, FR-120, FR-123 |
| `RestorationAlreadyUsed`, `RestorationAmountInvalid` (`cap`, `missing`), `RestorationNotAShortRest` | `applySorcererRest` | FR-084, FR-107, FR-108 |
| `DragonWingsAlreadyUsed`, `DragonWingsUseNotExpended`, `DragonWingsNotActive`, `DragonCompanionFreeCastUsed` | `useDragonWings`, `useDragonCompanion` | FR-093 to FR-095 |
| `NotASorcerer` | the four use cases (no `SorcererState`, not the seeding case of FR-011) | FR-010, FR-011 |

`MetamagicIneligibility` reasons are telemetry-silent: the rejection attribute carries only `metamagic_not_eligible`.

## 6. Interface

Use cases take the `Character` (whose `FeatureStates` hold the `SorcererState`) and return the new `Character` and events
(`doc/rules/architecture.md` §4). Only a feature that needs content gets a use case (FR-023); these four read the `ClassDefinition`
through `content.Source`, so take a `context.Context`; each builds the `SorcererProfile` (FR-137) from `ClassLevels` and `ChosenSubclass`, then calls
the pure function of the same name. SRD constants (2 uses, 2 and 3 point costs, half-level Restoration) live in the domain.

| Use case | Command carries | `<UseCase>Result` variants | Reads | FRs |
|---|---|---|---|---|
| `ChooseMetamagicOptions` | `Character`, chosen option ids | `...Applied(character, events)` / `...Rejected(rejection)` / `...ContentMissing(missing)` | Metamagic `FeatureOption`s | FR-035, FR-055 to FR-057 |
| `ApplySorcererLevelUp` | `Character`, the `LevelGained` event, Metamagic replacement and additions | `...Applied(character, owedMetamagicOptions, grants, events)` (`grants` = `SorcererLevelGrants`) / `...Rejected` / `...ContentMissing` | Table P row, feat grants | FR-011, FR-013, FR-018, FR-035, FR-047, FR-058, FR-085 to FR-087, FR-089, FR-127, FR-134 |
| `CreateSpellSlot` | `Character`, slot level | `...Applied(character, events)` / `...Rejected` / `...ContentMissing` | Creating Spell Slots `FeatureTable` | FR-052 to FR-054, FR-114 |
| `UseMetamagic` | `Character`, option id, `MetamagicSpellFacts`, `MetamagicCastContext`, host parameters (count, target, damage type) | `...Applied(character, effect, cost, events)` / `...Rejected` / `...ContentMissing` | option `pointCost` | FR-045, FR-046, FR-059 to FR-082, FR-110 to FR-118, FR-120, FR-123 |

**Domain functions (no use case, silent, synchronous).** Each takes the `SorcererState` and explicit inputs and returns a
`SorcererTransition` unless stated; those gating on level or subclass take a `SorcererProfile` (FR-137).

| Function | Inputs | Result | FRs |
|---|---|---|---|
| `sorcererFeatureAvailability`, `subclassFeaturesAvailable` | profile (and chosen `SubclassDefinition?`) | `SorcererFeatureAvailability`; `FeatureID`s at or below the level, empty if none chosen | FR-019, FR-024 |
| `sorcererLevelGrants` | profile (new level), `ClassDefinition`, chosen `SubclassDefinition?` | `SorcererLevelGrants` | FR-085 to FR-087, FR-089, FR-134 |
| `draconicResilienceBonus`, `draconicArmorClassBasis` | none | `HitPointMaximumBonus`; `AbilityBasedArmorClassBasis` candidate | FR-087, FR-088 |
| `chooseElementalAffinity` | state, profile, `DamageType` | transition | FR-090 |
| `activateInnateSorcery` | state, profile, `InnateSorceryPayment` (`InnateSorceryNormalUse`, `InnateSorceryPaidWithPoints`) | transition | FR-037 to FR-041, FR-044 to FR-046 |
| `endInnateSorcery`, `startSorcererTurn` | state | transition | FR-043, FR-046, FR-109 |
| `innateSorceryEffects`, `sorcererActiveEffects` | state | `InnateSorceryEffects`; `SorcererActiveEffects` | FR-037, FR-091, FR-092, FR-133, FR-135 |
| `convertSpellSlot` | state, profile, `SlotConversionSource` (`ConvertOrdinarySlot(SpellcastingState)`, `ConvertCreatedSlot`), slot level | `SorcererSlotTransition` | FR-050, FR-051, FR-114 |
| `spendCreatedSlot` | state, `CastingContext`, created slot level, spell level | transition (`CreatedSlotSpent`); host casts at the slot's level (FR-136) | FR-054, FR-136 |
| `checkQuickenedRestriction` | state, spell level about to be cast | `QuickenedCastAllowed` / `QuickenedCastRefused(rejection)` | FR-075 |
| `applySorcererRest` | state, profile, `RestKind`, requested Restoration amount (short rest only) | transition (`SorceryPointsRestored`, `CreatedSlotsExpired`) | FR-042, FR-049, FR-054, FR-083, FR-084, FR-093, FR-095, FR-106 to FR-108 |
| `applyElementalAffinity` | state, profile, Charisma modifier, `DamageType` dealt (after Transmuted), already-applied flag | transition (`ElementalAffinityApplied`) | FR-091, FR-104, FR-105 |
| `useDragonWings` | state, profile, `DragonWingsAction` (`ActivateDragonWings`, `DismissDragonWings`, `RestoreDragonWingsUse`) | transition | FR-092 to FR-094 |
| `useDragonCompanion` | state, profile, whether Concentration is waived | transition (`DragonCompanionCast`) | FR-095, FR-096 |

- Save DC, attack bonus, slots and casting are spellcasting.md's. `applySorcererRest` is called beside `RestoreSpellSlots`
  with the same `RestKind`; it never touches a `SpellcastingState` and is the one function that removes created slots (FR-054).
- Package: `character` exports the use cases, commands, results, `SorcererState` and its key, the value
  objects, rejection, transition, event, effect and handoff types named in section 5 and the domain functions. No generic variant name and no rejection class is exported (FR-130).
- A consumer needing another subclass supplies it as content. Metamagic is fixed to the SRD's ten effects (costs may differ
  in content, effects may not): a deliberate limit.

## 7. Observability

The domain is **silent by design** (architecture.md §3, observability.md §3); telemetry comes only from the four use cases.
This spec owns only the four spans `srd5e.character.choose_metamagic_options`, `srd5e.character.apply_sorcerer_level_up`,
`srd5e.character.create_spell_slot`, `srd5e.character.use_metamagic` (attributes in FR-124, with the content lookup as a
child span). The counter `srd5e.character.feature_used`, its `.duration` histogram and its `.errors` counter are owned by
character-aggregate.md FR-118 (shared-types-and-ownership.md FR-042, FR-063). Values this spec adds: `srd5e.class`
`sorcerer`; `srd5e.feature` `metamagic` (`ChooseMetamagicOptions`, `UseMetamagic`), `font_of_magic` (`CreateSpellSlot`),
`sorcerer_level_up` (`ApplySorcererLevelUp`); `srd5e.character.rejection` the snake_case `SorcererRejectionKind` (about
thirty, bounded) plus `content_missing` on a `...ContentMissing` result. Names are constants in the character telemetry constants file; entries feed `doc/observability/signals.md`.

Seven questions (`doc/rules/observability.md` §2):

1. **What can go wrong?** Class definition missing; `content.Source` failing; a request illegal in the state; invalid host input;
   programmer errors (`srderr.ErrInvalidInput`). A missing Metamagic option or slot table cannot occur (decoder, FR-099).
2. **How would an operator know?** Port failure: span error status, `feature_used` failure and `.errors` with `error.type`.
   `...ContentMissing`: span error status, result `content_missing`, rejection `content_missing`. Rejection: outcome `failure`, result `rejected`, rejection attribute.
3. **What to diagnose with?** The feature, result and rejection attributes; results carry cost, cap, points, level, problems, events.
4. **What to count or time?** Uses and latency per feature through the shared signals only.
5. **Is existing instrumentation still correct?** Yes; the old `class_feature.*` names are withdrawn (shared-types-and-ownership.md FR-042).
6. **What to monitor?** Spans with error status; a sustained rise in `failure` for `metamagic` with rejection `sorcery_points_insufficient`.
7. **Is anything noisy?** No. No log at any level.

## 8. Test plan

Levels: unit (`character/*_test.go` (package `character_test`)), application (`character/*_test.go` (use-case tests): real domain, recording `telemetry.Telemetry`,
`contenttest.FakeSource` and one scripted to return a `*srderr.PortError`, from the `<pkg>test` packages), scenario
(`scenarios/`, public entry points only), architecture (guardrails). Table-driven tests (Tables C, V, M, R, E) take
requirement and path from each row; Table P is fixture data for content-classes.md FR-035. Withdrawn ids have no test here.

| Id | Level | Notes |
|---|---|---|
| FR-010, FR-129 | unit | `SorcererCreationChoices` seeds the level 1 `SorcererState` |
| FR-011 | application | `ApplySorcererLevelUp` seeds level 1 for `sorcerer` at class level 1 with no state; `NotASorcerer` at class level 2 with no state |
| FR-017, FR-102 | none here | Sorcerer rows of the all-class scenario test of content-classes.md FR-035 |
| FR-018, FR-019, FR-024 | unit | Levels 0, 1, 20, 21, -1; availability at 1 and 2; `subclassFeaturesAvailable` at 2, 3, 5, 6, 13, 14, 17, 18 and none while no subclass |
| FR-032, FR-103 | unit | Draconic Spells not counted, not replaceable; always-prepared Command counts for the Innate Sorcery DC and Metamagic (M-61) |
| FR-035, FR-055 to FR-058 | unit | Options owed at levels 1, 2, 9, 10, 16, 17, 20; selection validation; second replacement refused |
| FR-037 to FR-044, FR-133 | unit | Innate Sorcery uses, level 7 payment, Long Rest, ending; `innateSorceryEffects` active gives `Modifier(1)` and `Advantage`, inactive neither |
| FR-045, FR-046, FR-059 to FR-063, FR-109 to FR-113 | unit | Two options; free use (level 19/20, inactive, second use, cleared by `startSorcererTurn`); cost, shortage by one, unknown option; Table M rows M-53 to M-60; Quickened reads `levelOnePlusSpellCastThisTurn` (feature-granted level 1+ cast counts) |
| FR-047 to FR-054, FR-114, FR-136 | unit | Sorcery Points maximum, rise, rest; Tables V and C one test per case id; `spendCreatedSlot` (higher slot on lower spell, too low, turn limit); `applySorcererRest` `longRest` removes created slots, `shortRest` keeps them |
| FR-064 to FR-082, FR-115 to FR-118, FR-120, FR-123 | unit | Table M one test per case id (M-1 to M-52, M-61 to M-65); Seeking with an `AttackRollResult` that hit and one that missed |
| FR-083, FR-084, FR-106 to FR-108 | unit | Table R one test per case id |
| FR-085 to FR-089, FR-134 | unit | `sorcererLevelGrants` at every level 1-20 from a decoded `ClassDefinition`; `draconicResilienceBonus` at 3, 4, 20; `draconicArmorClassBasis` (armor worn or not, Shield, derived-stats.md row = 17); Draconic Spells per level |
| FR-090 to FR-096, FR-104, FR-105, FR-135 | unit | Table E one test per case id; each type, outside type, second choice; `sorcererActiveEffects`; Dragon Wings and Dragon Companion transitions, 2 versus 3 points |
| FR-099 | application | Per use case: `...ContentMissing` for an absent `ClassID`; scripted `*srderr.PortError` propagated with state unchanged; unlisted option id is `MetamagicOptionNotKnown` |
| FR-100, FR-101, FR-128, FR-137 | unit | Input unchanged, read-only (accessors return copies) collections, repeated call equal; `SorcererState` through its `FeatureStateKey`, value equality, no level or subclass held; aggregate leaves it unchanged on other transitions (character-aggregate.md FR-097) |
| FR-124 to FR-127 | application | Per use case: span, `feature_used` and duration with the FR-124 attributes; rejection and `ContentMissing` as specified; a port failure records `feature_used` failure and one `.errors` increment with `error.type`, error status, no log; equal results with no-op and recording `telemetry.Telemetry`; `LevelGained` in, grants out, no second level event |
| FR-130 to FR-132 | architecture | Package scan: no generic or duplicate exported name; every `SorcererEvent` carries a `CharacterID`; no `schemaVersion` literal; bundled data only via `BundledContentSource` |
| Whole slice | scenario | `scenarios/sorcerer_test.go` via the public entry points: Sorcerer 1 to 20 (points, slots, Metamagic milestones) and a multiclass Ranger 4 / Sorcerer 3 (caster level 5) with Innate Sorcery, a created slot spent via `CastMode.FeatureGranted` and a Metamagic use |

- SRD text in tests lives only under `testdata/srd/` (shared-types-and-ownership.md FR-033). Mutation testing: Sorcerer files in
  `character/` MUST score at least 80 at `gomu --incremental --base-branch=origin/main`, Docker only with `--cpus` and `--workers` capped at a quarter of the processors (testing.md §8), pinning every table row. No test needs randomness.

## 9. Open questions and decisions

No question is escalated. Settled, one line each (design basis in the FR): created slot is a real slot, spell level at most the slot's, one-slot-per-turn applies, cast at the slot's level (FR-136; SRD 5.2.1 "Creating Spell Slots", "One Spell with a Spell Slot per Turn", pp. 66, 105); Quickened Spell reads its own host fact `levelOnePlusSpellCastThisTurn`, so no spellcasting.md change is needed (FR-113); Innate Sorcery's +1 DC is added by the host from `innateSorceryEffects`, `CalculateSpellcasting` stays unchanged (FR-037, FR-133). Earlier package decisions: FR-044, FR-047, FR-051, FR-078, FR-105, FR-108, FR-111, FR-112, FR-116, FR-117, FR-123.

One cross-spec request remains, with a fallback (FR-136): casting-and-concentration.md should let `CastMode.FeatureGranted` take an optional cast level at or above the spell's own and a flag making the cast slot-limited (rejects `SlotAlreadySpentThisTurn`, sets `slotSpentThisTurn`), amending its FR-123 and FR-124. Fallback: the host casts at the spell's own level and sets `slotSpentThisTurn` itself.

## 10. Changelog impact

`CHANGELOG.md`, under Unreleased / Added: "Sorcerer class behaviour in the character context: Innate Sorcery, Font of Magic, Metamagic, Sorcerous Restoration and the Draconic Sorcery features, as a feature state, pure functions and four use cases. The Sorcerer class data is added with content-classes.md." Not **BREAKING**: all new API.
