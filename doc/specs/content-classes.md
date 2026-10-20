# Content: classes and subclasses

| | |
|---|---|
| Status | partially implemented |
| Context | `content` |
| Roadmap phase | Types 2; bulk data 3 |
| Depends on | `content-schema-and-decoder.md`, `content-source-port.md`, `errors-and-results.md` |

## 1. Purpose and scope

Single owner of the class content model (shared-types-and-ownership.md FR-025) and of the
bulk class and subclass data (phase split: FR-014; see [../roadmap.md](../roadmap.md)). It
defines:

- `ClassDefinition`, `SubclassDefinition`, `ProgressionRow`, `FeatureEntry`,
  `AbilityRequirement` (closed `AllOf` / `AnyOf`) and their small value objects;
- the JSON shape of the `classes` root array, with decoder rules that report every problem
  (schema FR-012);
- the Features table (typed core, typed spellcasting part, per-class declared columns);
- how a feature is stored: stable id, three **structured facts** other contexts read as
  data, structured options and tables, and the faithful SRD text;
- the bulk data for the twelve SRD classes and twelve subclasses, their extraction and
  verification and the oracle tables are in content-classes-data.md (split, FR-062);
- the revision-table rows needed in content-schema-and-decoder.md (shared-types-and-ownership.md
  FR-034, FR-035).

The `class-*.md` specs own **behaviour only** and register no content shape. The class spell list is
not defined here: it is derived from `Spell.classes` by content-spells.md (FR-120).

**Out of scope:**

- Feature behaviour, resources, per-class state and creation choices (`class-*.md`;
  shared-types-and-ownership.md FR-022 to FR-024).
- Level-up grants and multiclass decision logic (`level-advancement-and-multiclassing.md`,
  `character-creation.md`); this spec stores the data they read.
- Spell, feat, equipment, species and background entries. Starting-equipment item names are
  printed text, not item ids.
- Machine-readable mechanics beyond the three facts (schema Q7).
- Anything not in SRD 5.2.1. The PDF has twelve classes and exactly one subclass each
  (table SRC); nothing that looked like core class material was found missing.

Notice and licensing for the bundled data are owned by content-licensing-and-notice.md, which this spec cites in prose and does not depend on (content-schema-and-decoder.md and content-source-port.md are its only content dependencies, shared-types-and-ownership.md FR-047).


## 2. Source

SRD 5.2.1 printed page numbers equal PDF page numbers.

- "Classes", pp. 28-82: Barbarian 28-30, Bard 31-35, Cleric 36-40, Druid 41-46, Fighter
  47-49, Monk 49-52, Paladin 53-56, Ranger 57-61, Rogue 61-64, Sorcerer 64-69, Warlock
  70-76, Wizard 77-82.
- Core Traits tables and "Becoming a ..." lists: pp. 28, 31, 36, 41, 47, 49, 53, 57, 61, 64,
  70, 77. Features tables: pp. 28, 31, 36, 41, 47, 50, 53, 58, 62, 65, 71, 77.
- Class spell lists: pp. 33-35 (Bard), 38-40 (Cleric), 44-45 (Druid), 55-56 (Paladin), 60
  (Ranger), 67-69 (Sorcerer), 74-76 (Warlock), 79-82 (Wizard). Subclasses: table SRC.
- Embedded tables: pp. 42, 46, 66. Option lists: pp. 66-67 (Metamagic), 72-74 (Invocations).
- "Character Creation", "Class Overview" (Primary Ability), p. 19; "Parts of a Feat:
  Category", p. 87.

Rules derive from shared-types-and-ownership.md (FR-005 to FR-007, FR-014, FR-017, FR-020,
FR-022 to FR-025, FR-030 to FR-035), `doc/rules/architecture.md` (§2, §3, §5, §6),
`doc/rules/observability.md` (§2, §3, §7) and `doc/rules/testing.md` (§3 to §5, §8, §10).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Class | `ClassDefinition` (`Entry`, key kind `characterClass`), `ClassID` | Owned here. |
| Subclass | `SubclassDefinition`, `SubclassID` | Nested inside its class entry (D1). |
| Primary Ability | `AbilityRequirement`, closed `AllOf` / `AnyOf` | "Strength and Charisma" is `AllOf`; Fighter's "Strength or Dexterity" is `AnyOf`. |
| Hit Point Die | `HitPointDie` (schema FR-132) | `d12`; JSON `"D12"`. Not the kernel `Die` (shared-types-and-ownership.md FR-053). |
| Saving Throw Proficiencies | `savingThrowProficiencies`, `[]ability.Ability` | Printed order; SRD row name is the field name (architecture.md §3). |
| Skill Proficiencies | `SkillChoice`, closed `SkillChoiceListed` / `SkillChoiceAny` | "Choose 2: ..."; Bard's "Choose any 3 skills". |
| Weapon Proficiencies | `[]WeaponProficiency` (`WeaponTrainingCategory` plus optional `WeaponProperty` filter) | `WeaponTrainingCategory` (`simple`, `martial`) is defined here (shared-types-and-ownership.md FR-047); `WeaponProperty` is schema FR-113. |
| Tool Proficiencies | `[]ClassToolProficiency`, closed `ClassToolProficiencySpecific` / `ClassToolProficiencyChoice` | Class-side type, distinct from the species spec's `ToolProficiency` (D10). |
| Armor Training | `[]ArmorTrainingCategory` (a set) | `light`, `medium`, `heavy`, `shield` (JSON `Light`, ...), defined here; "None" is an explicit empty set. |
| Starting Equipment | `ClassStartingEquipment` (`a`, `b`, `c?`) of `ClassEquipmentOption` of `ClassEquipmentItem` | Defined here; data specs do not depend on each other (shared-types-and-ownership.md FR-047). |
| Becoming a ... As a Multiclass Character | `MulticlassGrants` | What a first level in the class grants beyond the level 1 features. |
| Spellcasting, Pact Magic | `ClassSpellcastingTrait`, closed `SlotSpellcastingTrait` / `PactSpellcastingTrait` | `...Trait` because `ClassSpellcasting` is spellcasting.md state (shared-types-and-ownership.md FR-021). |
| Features table row | `ProgressionRow` | One per level 1 to 20. |
| Class-specific column | `ClassColumn`, `ClassColumnID`, `ColumnValue` | Open set declared per class. |
| "Level N: Name" | `FeatureEntry`, `FeatureID` | Class and subclass features. |
| "Subclass feature" cell | `RowSubclassFeature` | Marker; the subclass supplies the feature. |
| Always-prepared spells | `AlwaysPreparedSpells` (a `FeatureFact`) | Divine Smite; Life Domain Spells table. |
| Ability Score Improvement, Epic Boon, Fighting Style | `GrantsFeat(FeatCategory)` (a `FeatureFact`) | `FeatCategory`: schema FR-117. |
| Metamagic Options, Eldritch Invocation Options, Primal Order | `FeatureOption`, `FeatureOptionID` | Catalogue of named, chosen options. |
| Beast Shapes, Creating Spell Slots, Nature's Ward | `FeatureTable` | Embedded tables no fact captures. |
| (package term) oracle | the tables of content-classes-data.md and the fixture mirroring them | Independent of the bundled JSON. |

JSON values for SRD enums use the SRD spelling ("Sleight of Hand", "Martial"); structural
`kind` tags are lower camel case; ids match `[a-z][a-z0-9_]*` (shared-types-and-ownership.md
FR-007). Package types named only in section 5 (`ResolvedRow`, `SpellGrant`,
`FeatureEntryLookup`, `OptionPrerequisite`, `TableCell`) are defined there.

## 4. Requirements

A "problem" is one `Problem` in the joined `Problem` error (schema FR-014). Paths are
JSON Pointers from the document root (`/classes/0/progression/3/features/1/featureId`).
"Reject" means the decoder returns the joined `Problem` error with the stated problem;
constructors return an error wrapping `srderr.ErrInvalidInput` for the same violations (schema FR-093, FR-094). Each
rule lives once, as a constant or predicate on the domain type. Every "bundled" requirement
is checked against the oracle tables that follow the requirements. Pages are printed pages.

### 4.1 Entry, identity and document shape

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `ClassDefinition` MUST be an immutable `Entry` whose key is (`KindCharacterClass`, its `ClassID`). A lookup of that key returns it inside `LookupFound`; an absent class is `LookupNotFound` holding `MissingContent(characterClass, id)`, never an error. | shared-types-and-ownership.md FR-005, FR-018, FR-025; content-source-port.md | happy |
| FR-002 | A class entry MUST have the fields `id`, `name`, `source?`, `primaryAbility`, `hitPointDie`, `savingThrowProficiencies`, `skillChoice`, `weaponProficiencies`, `toolProficiencies`, `armorTraining`, `startingEquipment`, `multiclassGrants`, `spellcasting?`, `columns`, `progression`, `features`, `subclasses`, in that order (the problem order of FR-005). A required field that is absent is `missingField`; a key not listed is `unknownField` (schema FR-019, FR-022). `toolProficiencies`, `armorTraining`, `columns` and `subclasses` are required and MAY be empty arrays: absence is never the way to say "none". | SRD 5.2.1, "Core Wizard Traits" (Armor Training "None"), p. 77; architecture.md §3 | unhappy (missing content) |
| FR-003 | The bundled `classes` MUST come from the twelve entries of table SRC: ids `barbarian`, `bard`, `cleric`, `druid`, `fighter`, `monk`, `paladin`, `ranger`, `rogue`, `sorcerer`, `warlock`, `wizard`, with the SRD class name as `name`. **Staging (shared-types-and-ownership.md FR-014):** classes ship one at a time (Wizard already in the phase 2 slice), so until all twelve are bundled the check asserts only that the bundled classes are a subset of table SRC (no class outside it) and that each present class matches its rows of every table; a missing class is not yet a failure. When the twelfth class lands, the same change replaces the subset assertion with "exactly these twelve" (a thirteenth or a missing one fails) and removes the staging pin (Q4). | SRD 5.2.1, "Classes", pp. 28-82 | happy |
| FR-004 | Three id spaces MUST be unique across the whole document, each decided on the lookup set of schema FR-095 and reported at the later entry's `id` with `duplicate`: `ClassID` (schema FR-031, reported first), then `SubclassID` across all classes, then `FeatureID` across all class and subclass features, in document order (classes by index, class features before subclass features). The same string in two different id spaces is accepted. A subclass or feature whose own id is invalid takes no part in the check. | schema FR-031, FR-095, FR-098; shared-types-and-ownership.md FR-007 | unhappy (invalid input) |
| FR-005 | Problems inside one class entry MUST be reported in a deterministic order: field problems in the order of FR-002, descending into arrays by ascending index and objects in declared field order, unknown keys after the known fields of the same object; then the cross-field problems of the entry in the order spellcasting rows (FR-030), progression references (FR-044, FR-045), subclass level checks (FR-071, FR-073); then, after all entry-level problems of the document (every class entry, not one), the duplicate problems of FR-004 in the order class ids, subclass ids, feature ids (schema FR-016). The same text always gives an equal result. The cross-field problems of an entry (and of a feature entry's parts) are reported only when every field of it decoded with no problem, so a part dropped for a problem cannot cascade into false reference problems. The field order of an object is the order in which its fields are listed in the requirement or section 5 "Types" entry that defines it; unknown keys follow the known fields in ordinal order. | schema FR-016; testing.md §4 (determinism) | unhappy (rule interaction) |
| FR-006 | A value of the wrong type for a container (a `progression` that is a string, a `features` entry that is not an object) MUST produce one `wrongType` problem at that path and no problem for its children. | schema FR-013 | unhappy (rule interaction) |
| FR-007 | Every value object in this spec MUST validate in its constructor and return an error wrapping `srderr.ErrInvalidInput` for each invalid instance the rules below exclude; the decoder MUST reuse the same limits as public constants and predicates; the table-driven agreement test of schema FR-094 MUST feed every row of this spec's case tables to both paths. | architecture.md §6; schema FR-093, FR-094 | unhappy (invalid input) |
| FR-008 | Every type in this spec MUST be immutable with value equality; every exposed collection MUST be returned as a copy (mutating it leaves the value unchanged) and later mutation of a caller's input list MUST NOT change an instance. Two instances built from the same data are equal. | architecture.md §3, §4 (handoffs); testing.md §4 (immutability) | unhappy (immutability) |
| FR-009 | `source` MUST follow schema FR-034 (all three parts or absent). Every bundled class and subclass MUST carry a `source` with document `SRD 5.2.1`, the section named in table SRC and the first page of that section; schema FR-090 also applies. | SRD 5.2.1, "Classes", pp. 28-82; AGENTS.md (cite section and page) | happy |
| FR-010 | The types of this spec MUST live in the class files (`class_*.go`) of package `content`, be exported, be documented with a doc comment starting with the type name, and import nothing from `os` or another context; only the decoder files (`decode*.go`) import `encoding/json` (schema FR-073, FR-075; architecture.md §9). There is no facade or re-export. | architecture.md §1, §2, §7, §9; schema FR-088 | happy |

### 4.2 AbilityRequirement

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-011 | `AbilityRequirement` MUST be a closed interface with exactly `AllOf` and `AnyOf`, exhaustive in a `switch` without a default. JSON is `{"kind": "allOf" or "anyOf", "abilities": [...]}`; a `kind` of any other value is `invalidValue` at `kind`. | shared-types-and-ownership.md FR-025; SRD 5.2.1, "Class Overview", p. 19 | happy |
| FR-012 | `AllOf` MUST hold at least one ability and no ability twice. An empty list is `outOfRange` at `abilities`; a repeated ability is `duplicate` at the later index; the constructor returns an error wrapping `srderr.ErrInvalidInput`. A single primary ability (Strength for the Barbarian) is `AllOf` of one. | SRD 5.2.1, "Core Barbarian Traits", p. 28 | unhappy (boundary) |
| FR-013 | `AnyOf` MUST hold at least two distinct abilities (`AnyOf` of one is the same fact as `AllOf` of one and is not representable). One ability is `outOfRange`; a repeated one is `duplicate`. | SRD 5.2.1, "Core Fighter Traits", p. 47 | unhappy (boundary) |
| FR-014 | `IsMetBy(meeting []ability.Ability)` MUST return true for `AllOf` when every listed ability is in `meeting`, and for `AnyOf` when at least one is. Abilities in `meeting` that the requirement does not list are ignored. The threshold that decides which abilities "meet" the requirement belongs to the rule owner (character-aggregate.md), never to this type. | SRD 5.2.1, "Multiclassing", "Prerequisites", p. 24; "Class Overview", p. 19 | per case (table AR): happy / unhappy (boundary) |
| FR-015 | `abilities` MUST preserve the printed order, be returned as a copy by `Abilities()`, and accept only SRD ability names with SRD spelling (`invalidValue` otherwise; schema FR-025). | SRD 5.2.1, "The Six Abilities", p. 5 | unhappy (invalid input) |
| FR-016 | Each bundled `primaryAbility` MUST equal table CT1, and agree with the Class Overview table on p. 19 (one oracle entry per class from each page, both compared). | SRD 5.2.1, "Class Overview", p. 19; Core Traits tables, pp. 28-77 | per case (table CT1): happy / unhappy (boundary) |

### 4.3 Core traits and training

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-017 | `hitPointDie` MUST be a `HitPointDie`, defined by content-schema-and-decoder.md FR-132 (this spec does not define it). JSON is the string `"D6"`, `"D8"`, `"D10"` or `"D12"`: a non-string is `wrongType`, any other string (`"d8"`, `"D4"`) `invalidValue`. Each bundled value MUST equal table CT1. | SRD 5.2.1, Core Traits tables, pp. 28-77; content-schema-and-decoder.md FR-132 | per case (table CT1): happy / unhappy (boundary) |
| FR-018 | `savingThrowProficiencies` MUST hold from one to six distinct abilities (empty is `outOfRange`, a repeat `duplicate`). Each bundled class MUST have exactly the two abilities of table CT1, in printed order. | SRD 5.2.1, "Saving Throw Proficiencies", Core Traits tables | per case (table CT1): happy / unhappy (boundary) |
| FR-019 | `skillChoice` MUST be `{"picks": n, "options": {"kind": "listed", "skills": [...]}}` (`SkillChoiceListed`) or `{"picks": n, "options": {"kind": "any"}}` (`SkillChoiceAny`). `picks` is at least 1; a listed set is non-empty, has distinct `Skill` values and at least `picks` of them (`outOfRange`, `duplicate`, `invalidValue` for an unknown skill name). | SRD 5.2.1, "Core Bard Traits" ("Choose any 3 skills"), p. 31; "Core Barbarian Traits", p. 28 | unhappy (boundary) |
| FR-020 | Each bundled `skillChoice` MUST equal table CT1: its number of picks and, for a listed choice, exactly its skills. | SRD 5.2.1, Core Traits tables, pp. 28-77 | per case (table CT1): happy / unhappy (boundary) |
| FR-021 | `weaponProficiencies` MUST be an array of `{"category": "Simple" or "Martial" (a `WeaponTrainingCategory`), "requiresAnyProperty"?: [...]}` with at most one element per category (`duplicate`). A present `requiresAnyProperty` is non-empty and holds distinct `WeaponProperty` values; absent means every weapon of the category. The array MAY be empty and then means no weapon proficiency. Each bundled value MUST equal table CT2. | SRD 5.2.1, "Core Monk Traits" ("Martial weapons that have the Light property"), p. 49; "Core Rogue Traits" (Finesse or Light), p. 61 | per case (table CT2): happy / unhappy (boundary) |
| FR-022 | `armorTraining` MUST be an array of distinct `ArmorTrainingCategory` values (`Light`, `Medium`, `Heavy`, `Shield`; another spelling is `invalidValue`; `duplicate` for a repeat); `[]` is the explicit "None" and is not the same as an absent field (FR-002). Each bundled value MUST equal table CT2. | SRD 5.2.1, "Core Wizard Traits" (Armor Training None), p. 77; "Core Barbarian Traits", p. 28 | per case (table CT2): happy / unhappy (boundary) |
| FR-023 | `toolProficiencies` MUST be an array of `{"kind": "specific", "name"}` (non-blank name) or `{"kind": "choice", "categories": [...], "picks": n}` (categories non-empty, distinct, non-blank; picks at least 1). `[]` is the explicit "none". Each bundled value MUST equal table CT2. | SRD 5.2.1, "Core Bard Traits" (Choose 3 Musical Instruments), p. 31; "Core Monk Traits", p. 49 | per case (table CT2): happy / unhappy (boundary) |
| FR-024 | `startingEquipment` MUST have the keys `A` and `B`, and MAY have `C` (a Fighter option); any other key is `unknownField`, a missing `A` or `B` is `missingField`. Each option is a `ClassEquipmentOption` (`{"items": [{"name", "quantity"}], "gold": n}`, `items` possibly empty, each `quantity` at least 1 and `gold` at least 0, `outOfRange`), with `ClassEquipmentItem` as its element; both are defined here, and the background spec's same-shaped types share nothing with them (shared-types-and-ownership.md FR-047). The composite `ClassStartingEquipment` has `a`, `b` and optional `c` (JSON keys `A`, `B`, `C`). Each bundled value MUST equal table CT3. | SRD 5.2.1, "Core Fighter Traits: Starting Equipment" (A, B or C), p. 47; "Core Barbarian Traits", p. 28 | per case (table CT3): happy / unhappy (boundary) |
| FR-025 | An item name MUST be stored as printed, in the singular where the SRD prints a count ("2 Daggers" is `{"name": "Dagger", "quantity": 2}`), and a parenthetical stays in the name ("Arcane Focus (Quarterstaff)"). The two lines that name a choice rather than an item are stored verbatim after the count is removed: the Bard's "Musical Instrument of your choice" and the Monk's "Artisan's Tools or Musical Instrument chosen for the tool proficiency above". Names are not resolved to item ids. | SRD 5.2.1, "Core Bard Traits", p. 31; "Core Monk Traits", p. 49 | unhappy (boundary) |
| FR-026 | `multiclassGrants` MUST be `{"weaponProficiencies": [...], "armorTraining": [...], "toolProficiencies": [...], "skillChoice"?: ...}` using the shapes of FR-019 and FR-021 to FR-023, each array possibly empty. It MUST NOT exceed the class's core traits: its weapon training, armor training and listed skills MUST be subsets of the class's own (a `SkillChoiceAny` grant needs `SkillChoiceAny` core), its tool choices MUST name only categories of a core choice or a core specific tool, and its picks MUST NOT exceed the core picks. A violation is `constraintViolation` at the offending element. The Hit Point Die and the level 1 features are always granted and are not stored here. | SRD 5.2.1, "Becoming a ... As a Multiclass Character", pp. 28, 31, 36, 41, 47, 49, 53, 57, 61, 64, 70, 77 | unhappy (rule interaction) |
| FR-027 | Each bundled `multiclassGrants` MUST equal table MC. | SRD 5.2.1, "As a Multiclass Character", pp. 28-77 | per case (table MC): happy / unhappy (boundary) |

### 4.4 Spellcasting data

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-028 | `spellcasting`, when present, MUST be `{"kind": "slots", "ability": ..., "cantrips": bool}` (`SlotSpellcastingTrait`) or `{"kind": "pact", "ability": ...}` (`PactSpellcastingTrait`); the closed `ClassSpellcastingTrait` is exhaustive in a `switch`. An unknown `kind` is `invalidValue`. | SRD 5.2.1, "Level 1: Spellcasting" headings, pp. 32, 36, 41, 54, 57, 64, 77; "Level 1: Pact Magic", p. 71 | happy |
| FR-029 | The bundled `spellcasting` values MUST equal table SC: absent for Barbarian, Fighter, Monk and Rogue; `slots` with the table's ability and `cantrips` flag for the other seven slot casters; `pact` for the Warlock only. | SRD 5.2.1, "Spellcasting Ability" sentences, pp. 32, 37, 42, 54, 58, 65, 72, 78 | per case (table SC): happy / unhappy (boundary) |
| FR-030 | Every row of the progression MUST carry the spellcasting part that matches the class: a `SlotSpellcastingRow` for `slots`, a `PactSpellcastingRow` for `pact`, and none when the class has no `spellcasting`. A row with the wrong variant, a row part on a non-caster, or a missing part on a caster is `constraintViolation` at the row's `spellcasting` (missing: `missingField`). On the wire a row part with a `slotLevel` key is the Pact Magic variant and any other is the slot variant, so a variant that does not suit the class is a rule between parts and not a type problem. | SRD 5.2.1, Features tables, pp. 31, 36, 41, 53, 58, 65, 71, 77 | unhappy (rule interaction) |
| FR-031 | A `SlotSpellcastingRow` MUST be `{"cantrips"?: n, "preparedSpells": n, "spellSlots": [nine integers]}`: `cantrips` present exactly when the class's `cantrips` flag is true (missing is `missingField`, present on a class without the flag `constraintViolation`); `preparedSpells` at least 0; `spellSlots` exactly nine non-negative integers for spell levels 1 to 9, where a printed "—" is stored as 0 and a half caster's columns beyond level 5 are 0. A length other than nine is `outOfRange`. | SRD 5.2.1, Features tables, pp. 31, 36, 41, 53, 58, 65, 77 | unhappy (boundary) |
| FR-032 | A `PactSpellcastingRow` MUST be `{"cantrips": n, "preparedSpells": n, "spellSlots": n, "slotLevel": n}` with `spellSlots` at least 1 and `slotLevel` from 1 to 9; 0 and 10 are `outOfRange`. | SRD 5.2.1, "Warlock Features", p. 71 | unhappy (boundary) |
| FR-033 | Each bundled spellcasting row MUST equal the oracle: the slot rows of table FS (Bard, Cleric, Druid, Sorcerer, Wizard), table HS (Paladin, Ranger), and the Cantrips, Prepared Spells and Pact Magic sequences of table SQ. | SRD 5.2.1, Features tables, pp. 31, 36, 41, 53, 58, 65, 71, 77 | per case (tables FS, HS, SQ): happy / unhappy (boundary) |
| FR-034 | In the bundled data, every spellcasting column MUST be non-decreasing with level (cantrips, prepared spells, slots of each spell level, Pact slot count, Pact slot level), a sanity check that does not depend on the oracle and catches a transposed row. | SRD 5.2.1, Features tables | unhappy (rule interaction) |
| FR-035 | The slot, cantrip and prepared-spell columns of the bundled data (FR-033) are the authoritative copy of the Features-table values **for `content` consumers**. spellcasting.md stays authoritative for spellcasting behaviour until it adopts `ProgressionRow.spellcasting` and keeps its own code tables (FR-010, FR-011, FR-014, FR-015, FR-111 there) until then; both are checked against the same PDF pages. This spec does **not** require agreement with any `character` type (shared-types-and-ownership.md FR-013). The cross-copy agreement scenario compares the bundled values against spellcasting.md's tables and derived-stats.md's `ProficiencyBonus`, naming class, level and column; it is a requirement of those specs (pending there; Q5). | SRD 5.2.1, Features tables; "Proficiency Bonus", p. 8; spellcasting.md FR-010, FR-011, FR-014, FR-015, FR-111 | unhappy (rule interaction) |

### 4.5 The Features table: columns, rows and accessors

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-036 | `columns` MUST be an array of `{"id", "header", "kind"}`: `id` a `ClassColumnID` (`snake_case`, unique within the class, `duplicate` for a repeat), `header` a non-blank string equal to the printed column heading, `kind` one of `count`, `dice`, `bonus`, `feetBonus`. The array MAY be empty (Wizard). Its order is the printed left-to-right order of the class-specific columns. | SRD 5.2.1, Features tables, pp. 28-77 | unhappy (empty case) |
| FR-037 | `progression` MUST hold exactly 20 rows and row `i` MUST have `level` `i + 1`. A length other than 20 is `outOfRange` at `progression`; a row whose `level` is not its position plus one is `invalidValue` at that row's `level`. | SRD 5.2.1, Features tables (levels 1 to 20) | unhappy (boundary) |
| FR-038 | A row's `proficiencyBonus` MUST be an integer of at least 1. Each bundled row MUST be +2 at levels 1 to 4, +3 at 5 to 8, +4 at 9 to 12, +5 at 13 to 16 and +6 at 17 to 20, the value changing exactly at levels 5, 9, 13 and 17. | SRD 5.2.1, Features tables, pp. 28-77; "Proficiency Bonus", p. 8 | unhappy (boundary) |
| FR-039 | A row's `values` MUST be an object with exactly one key per declared column: a missing declared column is `missingField` at `values/<id>`, an undeclared key `unknownField`. A class with no columns has `"values": {}`. | SRD 5.2.1, Features tables | unhappy (missing content) |
| FR-040 | A column value MUST be a closed `ColumnValue`: `ColumnCount(n)` (JSON integer, n at least 0), `ColumnDice(count, sides)` (JSON `{"count": c, "sides": s}`, c at least 1, `sides` a valid `Die`), `ColumnBonus(n)` (JSON integer, may be 0 or more), `ColumnFeetBonus(n)` (JSON integer, feet), or `ColumnNone` (the JSON string `"—"`, U+2014, exactly as printed). The JSON type must suit the declared kind (`wrongType`; any string other than `"—"` is `invalidValue`); `ColumnNone` is allowed for every kind. | SRD 5.2.1, "Monk Features" (Focus Points "—"), p. 50; "Cleric Features" (Channel Divinity "—"), p. 36 | per case (table CV): happy / unhappy (boundary) |
| FR-041 | The bundled column declarations MUST equal table CC (ids, headers, kinds), and Wizard MUST declare none. | SRD 5.2.1, Features tables, pp. 28-77 | per case (table CC): happy / unhappy (boundary) |
| FR-042 | Every bundled column value MUST equal the sequences of table CC for levels 1 to 20, including every printed "—". | SRD 5.2.1, Features tables, pp. 28-77 | per case (table CC): happy / unhappy (boundary) |
| FR-043 | A row's `features` MUST be an array whose elements are `{"featureId": "...", "annotation"?: "..."}` (`RowClassFeature`) or `{"kind": "subclassFeature"}` (`RowSubclassFeature`), in the printed order of the cell. A printed "—" is `[]`. A row MAY hold at most one `RowSubclassFeature` (`duplicate`). An `annotation` is the printed parenthetical without its brackets and is non-blank ("one use"). | SRD 5.2.1, "Fighter Features" ("Action Surge (one use)"), p. 47; "Warlock Features" ("Mystic Arcanum (level 6 spell)"), p. 71 | unhappy (empty case) |
| FR-044 | Every `featureId` of a row MUST name a feature entry of the same class, else `unresolvedReference` at that `featureId`; and that entry's `level` MUST NOT exceed the row's level, else `constraintViolation`. | SRD 5.2.1, Features tables | unhappy (missing content) |
| FR-045 | Every class feature entry MUST be referenced by the row at its own `level`. An entry that is not listed there is `constraintViolation` at the entry's `level`. | SRD 5.2.1, Features tables ("Class Features" column) | unhappy (rule interaction) |
| FR-046 | The bundled rows' `features` MUST equal the oracle: the entries of table F for the row's level in printed order, plus the references of table RR, the `RowSubclassFeature` rows of table SF, and `[]` for the rows table RR lists as "—". | SRD 5.2.1, Features tables, pp. 28-77 | per case (tables F, RR, SF): happy / unhappy (boundary) |
| FR-047 | `ClassDefinition.RowAt(level)` MUST return the row of that level and return an error wrapping `srderr.ErrInvalidInput` for 0, 21 and any value outside 1 to 20. | SRD 5.2.1, Features tables; architecture.md §6 | unhappy (boundary) |
| FR-048 | `ClassDefinition.EntriesGainedAt(level)` MUST return the class feature entries whose `level` equals the argument, in printed order, and `[]` for a level with none (for example Barbarian level 6). `RowFeatures(level)` MUST return a `ResolvedRow` (`Entries()`: the row's `RowClassFeature`s resolved to their entries in printed order, an entry possibly older than the row, Ability Score Improvement at level 8; `HasSubclassFeature()`: whether the row holds a subclass marker). Both return an error wrapping `srderr.ErrInvalidInput` outside 1 to 20. | SRD 5.2.1, "Becoming a ...", e.g. p. 28 ("Gain the Barbarian's level 1 features") | per case (table RR): happy / unhappy (boundary) |
| FR-049 | `ClassDefinition.FeatGrantsAt(level)` MUST return the `FeatCategory` of every `GrantsFeat` fact among the row's resolved features: Ability Score Improvement (General) at 4, 8, 12 and 16 for most classes, at 6 and 14 as well for Fighter and at 10 for Rogue. | SRD 5.2.1, "Level 4: Ability Score Improvement" and the Fighter and Rogue tables, pp. 47-48, 62-63 | per case (table FA): happy / unhappy (boundary) |
| FR-050 | `ClassDefinition.Column(ClassColumnID)` and `ProgressionRow.Value(ClassColumnID)` MUST return the declared column and its value; an undeclared id returns an error wrapping `srderr.ErrInvalidInput`. | architecture.md §6 | unhappy (invalid input) |
| FR-099 | `ClassDefinition.Feature(FeatureID)` MUST return the closed `FeatureEntryLookup`: `FeatureEntryFound(entry)` for a class feature of this class, `FeatureEntryNotFound(id)` otherwise (a subclass feature id is `FeatureEntryNotFound`; subclass features are reached through `Subclass(id)`). The variants carry their data and are not generic names (shared-types-and-ownership.md FR-017). | architecture.md §4, §6; FR-074 | unhappy (missing content) |
| FR-102 | `entry(FeatureID).name` MUST be the feature's printed SRD name, exactly (case-sensitive; no `level`, `_level_<n>` suffix or annotation), so a host maps a held class `FeatureID` to its printed name by `ClassDefinition.entry`. Bundled names are table F's: the Fighter, Paladin and Ranger "Fighting Style" and the Bard, Cleric, Druid, Paladin, Ranger, Sorcerer and Wizard "Spellcasting" are each named exactly so (distinct ids, same name), and the Warlock's is "Pact Magic", never "Spellcasting". The id stays the identity; two features may share a name (FR-053). No new API: feats-behaviour.md FR-011 and Q19 match held features by this name. | SRD 5.2.1, "Fighting Style", pp. 47, 54, 59; "Spellcasting", pp. 31-77; "Pact Magic", p. 71 | per case (table F): happy / unhappy (boundary) |

### 4.6 Features: ids, text, facts, options and tables

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-051 | A `FeatureEntry` MUST have `id`, `name`, `level` (1 to 20, `outOfRange` otherwise), `text`, and the optional arrays `facts`, `options` and `tables` (absent means none). The entries of a class are ordered by ascending level, then by printed order; the decoder reports but never reorders. | SRD 5.2.1, "Barbarian Class Features", pp. 28-30 | happy |
| FR-052 | A feature `id` MUST start with its owner's id and an underscore (the `ClassID` for a class feature, the `SubclassID` for a subclass feature), else `constraintViolation`. The bundled ids MUST be exactly `DeriveFeatureID(owner, name)`: the owner's id, an underscore, then the slug of the name (D4: lower-case, U+0027 and U+2019 deleted, each run of other characters one underscore, edge underscores trimmed; defined here in `DeriveFeatureID` until content-source-port.md defines a shared `DeriveContentID`, Q7), and every derived id MUST satisfy schema FR-030 (at most 64 characters; a longer result returns an error wrapping `srderr.ErrInvalidInput`). | schema FR-030; shared-types-and-ownership.md FR-007 | per case (table ID): happy / unhappy (boundary) |
| FR-053 | When one owner prints two features with the same name (the Barbarian's Improved Brutal Strike at levels 13 and 17), each id MUST end in `_level_<n>` for its own level; a name that is unique takes no suffix. | SRD 5.2.1, "Level 13: Improved Brutal Strike", "Level 17: Improved Brutal Strike", p. 30 | unhappy (name collision) |
| FR-054 | `text` MUST be the faithful SRD text of the feature, kept character for character (schema FR-028, FR-029): without the heading ("Level 1: Rage"), with bold lead-in words kept as plain text, with paragraphs separated by one line feed, and without any embedded table (captured by FR-066) or option list (FR-061). Sidebars that are not feature text ("Breaking Your Oath", "Expanding and Replacing a Spellbook") are not stored. | SRD 5.2.1, "Barbarian Class Features", pp. 28-30; "Breaking Your Oath", p. 54; "Expanding and Replacing a Spellbook", p. 78; schema FR-029 | happy |
| FR-055 | `FeatureFact` MUST be a closed interface with exactly `GrantsSubclass`, `GrantsFeat` and `AlwaysPreparedSpells`, exhaustive in a `switch`. JSON is `{"kind": "grantsSubclass"}`, `{"kind": "grantsFeat", "category": ...}` or `{"kind": "alwaysPreparedSpells", "grants": [...]}`; another `kind` is `invalidValue`. Facts on one entry MUST be distinct (`duplicate`). | shared-types-and-ownership.md FR-022, FR-025; schema Q7 | happy |
| FR-056 | A class MUST have exactly one class feature carrying `GrantsSubclass`; `ClassDefinition.subclassLevel` is that entry's `level`. Zero or two is `constraintViolation` (reported at the class's `features`); a subclass feature MUST NOT carry the fact. | SRD 5.2.1, "Level 3: Barbarian Subclass" and the other eleven, pp. 29, 32, 37, 43, 48, 51, 55, 59, 62, 66, 72, 78 | unhappy (rule interaction) |
| FR-057 | `GrantsFeat.category` MUST be a `FeatCategory` with its SRD spelling (`Origin`, `General`, `Fighting Style`, `Epic Boon`; `invalidValue` otherwise). The bundled facts MUST equal table FA. | SRD 5.2.1, "Parts of a Feat: Category", p. 87; "Level 4: Ability Score Improvement", p. 29; "Level 1: Fighting Style", p. 47 | per case (table FA): happy / unhappy (boundary) |
| FR-058 | `AlwaysPreparedSpells.grants` MUST be a non-empty array of `{"level": n, "spellIds": [...]}`: `level` from 1 to 20 and strictly ascending (`invalidValue` for an out-of-order level, `outOfRange` outside the range), `spellIds` non-empty with distinct valid `SpellID`s (`duplicate`, `invalidValue`), and `level` at least the owning feature's level (the `level` of the `FeatureEntry` that holds the fact, or of the entry that holds the option): a grant level below it is `invalidValue` at `grants/<i>/level`. Each grant is a `SpellGrant` (`level`, `spellIds`). | SRD 5.2.1, "Level 3: Life Domain Spells", p. 40; "Level 2: Paladin's Smite", p. 54 | unhappy (boundary) |
| FR-059 | Every spell id inside an `AlwaysPreparedSpells` fact (class feature, subclass feature or option) is a reference to a `spell` entry and MUST be a registered reference site of row R1-classes-spell-sites (section 5). It is checked by the whole-set cross-reference validation of content-schema-and-decoder.md (FR-125, FR-126; shared-types-and-ownership.md FR-054), which reports `unresolvedReference` naming the class, feature and id; the site has no scope and no opt-out. Decoding one class document alone reports nothing about it, because one document cannot know the others (schema FR-035). **Staging:** a class file ships in the change that bundles every spell its facts name (table FA), so the bundled set is always self-consistent; bulk spells are phase 5, so a class with such facts ships no earlier (Q4). | shared-types-and-ownership.md FR-047, FR-054; content-schema-and-decoder.md FR-035, FR-125, FR-126 | unhappy (missing content) |
| FR-060 | The bundled facts MUST be exactly those of table FA (kind, owner, level, category or spell grants) and no other entry MUST carry a fact: an absent fact is part of the oracle. | SRD 5.2.1, class and subclass features, pp. 28-82 | per case (table FA): happy / unhappy (boundary) |
| FR-061 | `options` MUST be an array of `{"id", "name", "text", "prerequisite"?, "pointCost"?, "repeatable"?, "facts"?}`: `id` a `FeatureOptionID` (`snake_case` of the printed name, unique within the feature, `duplicate`), `name` and `text` non-blank and faithful (FR-054). An option list is a catalogue; how many a character holds is stated by the feature text or a column. | SRD 5.2.1, "Level 1: Primal Order", p. 42; "Metamagic Options", pp. 66-67 | happy |
| FR-062 | An option `prerequisite` MUST be `{"minClassLevel"?: n, "requiresOptionIds"?: [...], "note"?: "..."}` with at least one part present (`constraintViolation` for `{}`); `minClassLevel` from 1 to 20; a present `requiresOptionIds` is non-empty (`outOfRange`) and distinct (`duplicate`); every required id MUST name a different option of the same feature (`unresolvedReference`; a self reference is `constraintViolation`) and the references MUST NOT form a cycle (`constraintViolation`); `note` holds any remaining printed requirement verbatim ("a Warlock Cantrip That Deals Damage"). | SRD 5.2.1, "Eldritch Invocation Options", pp. 72-74 | unhappy (rule interaction) |
| FR-063 | `pointCost`, when present, MUST be an integer of at least 1 ("Cost: 2 Sorcery Points"); `repeatable`, when present, MUST be `true` and means the SRD prints "Repeatable" for the option; both are absent when the SRD prints neither. | SRD 5.2.1, "Metamagic Options", pp. 66-67; "Agonizing Blast", p. 72 | unhappy (boundary) |
| FR-064 | An option MAY carry only an `AlwaysPreparedSpells` fact in `facts` (for the Circle of the Land's four land types); any other fact kind on an option is `constraintViolation`. | SRD 5.2.1, "Level 3: Circle of the Land Spells", p. 46 | unhappy (invalid input) |
| FR-065 | The bundled option lists MUST equal table OP (feature, option ids in printed order, `pointCost`, `repeatable`, prerequisites, option facts); a feature not listed there has no `options`. | SRD 5.2.1, class features and "Metamagic Options", "Eldritch Invocation Options", pp. 42, 66-67, 72-74 | per case (table OP): happy / unhappy (boundary) |
| FR-066 | A `FeatureTable` MUST be `{"name", "columns": [...], "rows": [[...]]}`: non-blank printed `name`, one or more non-blank column headers, one or more rows, every row as long as the header list (`outOfRange`), every cell a JSON string or integer and every column consistent in kind (all text or all integers; `wrongType` at the first deviating cell). A table the SRD prints inside a feature and that no fact captures MUST be a `FeatureTable` and MUST NOT be left in `text`. A table printed as side-by-side column pairs (Nature's Ward prints Land Type and Resistance twice, in two rows) is **normalised to one pair per row**, taking the left group of columns top to bottom and then the right group; this is the only restructuring and no cell is changed. | SRD 5.2.1, "Beast Shapes", p. 42; "Creating Spell Slots", p. 66 | unhappy (invalid input) |
| FR-067 | The bundled tables MUST be exactly the three of table TB (Beast Shapes on Wild Shape, Creating Spell Slots on Font of Magic, Nature's Ward on Nature's Ward), with their printed headers and cells, except that Nature's Ward is normalised as FR-066 states (printed as four columns in two rows, stored as two columns in four rows); no other feature has `tables`. | SRD 5.2.1, pp. 42, 46, 66 | per case (table TB): happy / unhappy (boundary) |
| FR-068 | The bundled class features and subclass features MUST equal table F exactly: every id (derived by FR-052 and FR-053), name, level and order, with nothing added and nothing missing. | SRD 5.2.1, "Class Features" headings, pp. 28-82 | per case (table F): happy / unhappy (boundary) |
| FR-069 | Once all twelve classes are bundled, the bundled data MUST contain 174 class feature entries and 58 subclass feature entries (the column totals of table F), a count asserted separately from the per-class rows so a lost or duplicated entry is reported as a total as well. **Staging:** until then the check asserts that the counts of the bundled classes equal the sum of their rows in table F (for example Wizard alone: 10 class entries and, until Evoker lands, no subclass entries); the totals are asserted in the change that lands the twelfth class (Q4). | SRD 5.2.1, pp. 28-82 | happy |

### 4.7 Subclasses

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-070 | A `SubclassDefinition` MUST be `{"id", "name", "source"?, "features": [...]}` with a `SubclassID`, a non-blank name and a non-empty feature list (`outOfRange` when empty); features use the shape of FR-051. A class MAY have no subclasses (`"subclasses": []`). | SRD 5.2.1, "Barbarian Subclass: Path of the Berserker", p. 30 | unhappy (empty case) |
| FR-071 | Every subclass feature MUST have a `level` of at least the class's `subclassLevel` and at most 20, and the subclass MUST have at least one feature at exactly `subclassLevel`. A lower level, or none at the subclass level, is `constraintViolation` at the offending feature (or at the subclass's `features`). | SRD 5.2.1, "Level 3: Barbarian Subclass", p. 29 | unhappy (boundary) |
| FR-072 | A subclass feature id MUST be prefixed with the subclass id (FR-052) and unique in the document (FR-004). | SRD 5.2.1, subclass sections, pp. 30-82 | unhappy (invalid input) |
| FR-073 | `subclasses` MAY be non-empty only when FR-056 holds for the class (every class has its subclass feature), and a `SubclassDefinition` constructed outside a class MUST be validated against the class's `subclassLevel` when added: `ClassDefinition`'s constructor returns an error wrapping `srderr.ErrInvalidInput` for an invalid combination. | SRD 5.2.1, "Level 3: ... Subclass", pp. 29-78 | unhappy (rule interaction) |
| FR-074 | `ClassDefinition.Subclass(SubclassID)` MUST return the closed `SubclassLookup`: `SubclassFound(subclass)` for a subclass of this class, `SubclassNotFound(id)` otherwise. Neither variant is generic (shared-types-and-ownership.md FR-017). | architecture.md §4, §6 | unhappy (missing content) |
| FR-075 | `SubclassDefinition.FeaturesAtOrBelow(level)` MUST return the subclass features whose `level` does not exceed the argument, in printed order ("you gain each of your subclass's features that are of your class level or lower"), and `[]` below the subclass level; it returns an error wrapping `srderr.ErrInvalidInput` outside 1 to 20. | SRD 5.2.1, "Level 3: Barbarian Subclass", p. 29 | per case (table SF): happy / unhappy (boundary) |
| FR-076 | The bundled subclasses MUST be exactly the twelve of table SRC, one per class, with the features of table F. | SRD 5.2.1, subclass sections, pp. 30-82 | per case (tables SRC, F): happy / unhappy (boundary) |
| FR-077 | For every bundled class, the levels above the subclass level at which each subclass has a feature MUST equal the levels of the class's `RowSubclassFeature` rows (table SF): no subclass feature at a level whose table row has no "Subclass feature" cell, and none missing at a level that has one. | SRD 5.2.1, Features tables ("Subclass feature"), pp. 28-77 | unhappy (rule interaction) |

### 4.9 Revision-table rows and traceability

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-095 | content-schema-and-decoder.md MUST register the rows of section 5 ("Revision rows requested") in its revision table in the same change that lands the class decoder; this spec claims no version number (shared-types-and-ownership.md FR-034). | shared-types-and-ownership.md FR-034, FR-035; schema FR-103 | happy |
| FR-096 | The format addition MUST add no `ProblemKind` and no `Kind`: every rule above uses the existing kinds `missingField`, `unknownField`, `wrongType`, `outOfRange`, `invalidValue`, `duplicate`, `unresolvedReference` and `constraintViolation`, and the entries stay in `KindCharacterClass`. | schema FR-015, FR-142 | unhappy (rule interaction) |
| FR-097 | The withdrawn rows of the schema spec MUST each be satisfied by a row of this spec, as in the traceability table of section 5 (FR-033, FR-046 to FR-054, FR-091, FR-092 there). | schema section 4 ("Class", "Additions") | happy |
| FR-098 | The class behaviour specs MUST NOT define a `ClassDefinition`, `ProgressionRow`, `SubclassDefinition` or `FeatureEntry`, nor a second copy of any table of this spec; they cite this spec's ids, tables and `FeatureID`s. A spec that does is a defect found by the spec-consistency review. | shared-types-and-ownership.md FR-020, FR-025 | unhappy (duplicate) |

### Withdrawn ids and oracle tables

The bundled files, the extraction and verification process and the oracle tables SRC, CT1 to
CT3, MC, SC, FS, HS, SQ, CC, F, RR, SF, FA, OP and TB are owned by content-classes-data.md.
"Table X" in a requirement of this spec means that table there.

| Id | Requirement |
|---|---|
| FR-078 | Withdrawn: moved to content-spells.md (FR-120): the class spell list is derived from `Spell.classes`; `ClassSpellListMember` does not exist (shared-types-and-ownership.md FR-066) |
| FR-079 | Withdrawn: moved to content-spells.md (FR-121) |
| FR-080 | Withdrawn: moved to content-spells.md (FR-121) |
| FR-081 | Withdrawn: moved to content-classes-data.md (FR-100), a bundle check across classes and spells (shared-types-and-ownership.md FR-067) |
| FR-082 | Withdrawn: moved to content-classes-data.md (FR-101) |

FR-100 and FR-101 of this spec were never issued: the gap is reserved group spacing, not lost requirements. (The FR-100 and FR-101 cited above are content-classes-data.md's.)
| FR-083 | Withdrawn: moved to content-classes-data.md (FR-083) |
| FR-084 | Withdrawn: moved to content-classes-data.md (FR-084), the cross-file duplicate-id check (content-source-port.md FR-059) |
| FR-085 | Withdrawn: moved to content-classes-data.md (FR-085) |
| FR-086 | Withdrawn: moved to content-classes-data.md (FR-086) |
| FR-087 | Withdrawn: moved to content-classes-data.md (FR-087) |
| FR-088 | Withdrawn: moved to content-classes-data.md (FR-088) |
| FR-089 | Withdrawn: moved to content-classes-data.md (FR-089) |
| FR-090 | Withdrawn: moved to content-classes-data.md (FR-090) |
| FR-091 | Withdrawn: moved to content-classes-data.md (FR-091) |
| FR-092 | Withdrawn: moved to content-classes-data.md (FR-092) |
| FR-093 | Withdrawn: moved to content-classes-data.md (FR-093), the oracle fixture |
| FR-094 | Withdrawn: moved to content-classes-data.md (FR-094) |

### Cases AR: `isMetBy` (one test per row)

| Label | Requirement | `meeting` | Expected | Source | Path |
|---|---|---|---|---|---|
| single, met | `AllOf` Strength | Strength | true | FR-014 | happy |
| single, not met | `AllOf` Strength | none | false | FR-014 | unhappy (empty case) |
| both required, one met | `AllOf` Dexterity, Wisdom | Dexterity | false | FR-014 | unhappy (boundary) |
| both required, both met | `AllOf` Dexterity, Wisdom | Dexterity, Wisdom | true | FR-014 | happy |
| either, first | `AnyOf` Strength, Dexterity | Strength | true | FR-014 | happy |
| either, second | `AnyOf` Strength, Dexterity | Dexterity | true | FR-014 | happy |
| either, neither | `AnyOf` Strength, Dexterity | Constitution | false | FR-014 | unhappy (boundary) |
| either, none | `AnyOf` Strength, Dexterity | none | false | FR-014 | unhappy (empty case) |
| extra ignored | `AllOf` Charisma | all six | true | FR-014 | unhappy (rule interaction) |

### Cases ID: feature id derivation (one test per row)

| Label | Owner, name, other | Expected id | Source | Path |
|---|---|---|---|---|
| plain | `barbarian`, Rage | `barbarian_rage` | FR-052 | happy |
| spaces | `bard`, Jack of All Trades | `bard_jack_of_all_trades` | FR-052 | happy |
| apostrophe deleted | `monk`, Monk's Focus | `monk_monks_focus` | FR-052 | unhappy (boundary) |
| plural possessive | `rogue`, Thieves' Cant | `rogue_thieves_cant` | FR-052 | unhappy (boundary) |
| hyphen | `thief`, Second-Story Work | `thief_second_story_work` | FR-052 | unhappy (boundary) |
| subclass owner | `fiend_patron`, Dark One's Own Luck | `fiend_patron_dark_ones_own_luck` | FR-052 | happy |
| duplicate name, first | `barbarian`, Improved Brutal Strike at level 13 | `barbarian_improved_brutal_strike_level_13` | FR-053 | unhappy (name collision) |
| duplicate name, second | `barbarian`, Improved Brutal Strike at level 17 | `barbarian_improved_brutal_strike_level_17` | FR-053 | unhappy (name collision) |
| wrong prefix | owner `fighter`, id `barbarian_rage` | `constraintViolation` | FR-052 | unhappy (invalid input) |
| 64 characters | owner plus name giving 64 characters | accepted | FR-052 | unhappy (boundary) |
| 65 characters | owner plus name giving 65 characters | `invalidValue` | FR-052 | unhappy (boundary) |

### Cases CV: column values (one test per row)

| Label | Column kind and value | Expected | Source | Path |
|---|---|---|---|---|
| count | `count`, 2 | `ColumnCount(2)` | FR-040 | happy |
| count zero | `count`, 0 | accepted | FR-040 | unhappy (boundary) |
| count negative | `count`, -1 | `outOfRange` | FR-040 | unhappy (boundary) |
| count as string | `count`, "2" | `invalidValue` (any string other than "—", FR-040) | FR-040 | unhappy (invalid input) |
| dice | `dice`, count 1 sides 6 | `ColumnDice(1, 6)` | FR-040 | happy |
| dice zero count | `dice`, count 0 sides 6 | `outOfRange` | FR-040 | unhappy (boundary) |
| dice one side | `dice`, count 1 sides 1 | `outOfRange` | FR-040 | unhappy (boundary) |
| integer for dice | `dice`, 6 | `wrongType` | FR-040 | unhappy (invalid input) |
| bonus | `bonus`, 2 | `ColumnBonus(2)` | FR-040 | happy |
| feet bonus | `feetBonus`, 10 | `ColumnFeetBonus(10)` | FR-040 | happy |
| none, count | `count`, "—" | `ColumnNone` | FR-040 | unhappy (empty case) |
| none, dice | `dice`, "—" | `ColumnNone` | FR-040 | unhappy (empty case) |
| hyphen for none | `count`, "-" | `invalidValue` | FR-040 | unhappy (invalid input) |
| object for count | `count`, count 1 sides 6 | `wrongType` | FR-040 | unhappy (invalid input) |

### Cases PR: progression, rows and references (one test per row)

| Label | Input (otherwise valid) | Expected problems | Source | Path |
|---|---|---|---|---|
| exactly 20 | 20 rows, levels 1 to 20 | none | FR-037 | happy |
| 19 rows | row of level 20 missing | `/classes/0/progression`, `outOfRange` | FR-037 | unhappy (boundary) |
| 21 rows | an extra row | `/classes/0/progression`, `outOfRange` | FR-037 | unhappy (boundary) |
| level out of order | row 4 has level 5 | `/classes/0/progression/3/level`, `invalidValue` | FR-037 | unhappy (invalid input) |
| missing column | row 0 lacks `rages` | `/classes/0/progression/0/values/rages`, `missingField` | FR-039 | unhappy (missing content) |
| undeclared column | row 0 has `luck` | `/classes/0/progression/0/values/luck`, `unknownField` | FR-039 | unhappy (invalid input) |
| unresolved feature | row 0 cites `barbarian_nothing` | `.../features/0/featureId`, `unresolvedReference` | FR-044 | unhappy (missing content) |
| entry later than row | row 0 cites a level 5 entry | `.../features/0/featureId`, `constraintViolation` | FR-044 | unhappy (boundary) |
| entry never listed | a level 2 entry absent from row 2 | `/classes/0/features/<i>/level`, `constraintViolation` | FR-045 | unhappy (rule interaction) |
| two subclass markers | row 5 has two | `.../features/1`, `duplicate` | FR-043 | unhappy (boundary) |
| empty cell | row 8 `features` is `[]` | none | FR-043 | unhappy (empty case) |
| blank annotation | `"annotation": ""` | `.../annotation`, `invalidValue` | FR-043 | unhappy (invalid input) |
| wrong container | `progression` is a string | `/classes/0/progression`, `wrongType` only | FR-006 | unhappy (rule interaction) |

### Cases SP: spellcasting rows (one test per row)

| Label | Input (otherwise valid) | Expected problems | Source | Path |
|---|---|---|---|---|
| nine slots | `spellSlots` of nine integers | none | FR-031 | happy |
| eight slots | eight integers | `.../spellcasting/spellSlots`, `outOfRange` | FR-031 | unhappy (boundary) |
| negative slot | one slot -1 | `.../spellSlots/0`, `outOfRange` | FR-031 | unhappy (boundary) |
| cantrips missing | class `cantrips: true`, row lacks it | `.../spellcasting/cantrips`, `missingField` | FR-031 | unhappy (missing content) |
| cantrips on Paladin | class `cantrips: false`, row has it | `.../spellcasting/cantrips`, `constraintViolation` | FR-031 | unhappy (rule interaction) |
| pact row on slot class | variant `pact` fields on a `slots` class | `.../spellcasting`, `constraintViolation` | FR-030 | unhappy (rule interaction) |
| row part on a non-caster | Fighter row with `spellcasting` | `.../spellcasting`, `constraintViolation` | FR-030 | unhappy (rule interaction) |
| slot level 0 | pact `slotLevel` 0 | `.../slotLevel`, `outOfRange` | FR-032 | unhappy (boundary) |
| slot level 10 | pact `slotLevel` 10 | `.../slotLevel`, `outOfRange` | FR-032 | unhappy (boundary) |
| slot level 9 | pact `slotLevel` 9 | none | FR-032 | unhappy (boundary) |

### Cases FT: facts, options, tables and equipment (one test per row)

| Label | Input (otherwise valid) | Expected problems | Source | Path |
|---|---|---|---|---|
| two subclass facts | two features with `grantsSubclass` | `/classes/0/features`, `constraintViolation` | FR-056 | unhappy (rule interaction) |
| no subclass fact | none carries it | `/classes/0/features`, `constraintViolation` | FR-056 | unhappy (missing content) |
| unknown category | `grantsFeat`, "Epic" | `.../facts/0/category`, `invalidValue` | FR-057 | unhappy (invalid input) |
| grants out of order | grants at 5 then 3 | `.../grants/1/level`, `invalidValue` | FR-058 | unhappy (boundary) |
| grant below entry | entry level 3, grant at 1 | `.../grants/0/level`, `invalidValue` | FR-058 | unhappy (boundary) |
| empty spell ids | `spellIds` is `[]` | `.../spellIds`, `outOfRange` | FR-058 | unhappy (empty case) |
| unknown spell id | a set in which no spell has the granted id | none from the decoder alone; `unresolvedReference` from the whole-set validation | FR-059 | unhappy (missing content) |
| duplicate option id | two options with one id | `.../options/1/id`, `duplicate` | FR-061 | unhappy (invalid input) |
| empty prerequisite | `"prerequisite": {}` | `.../prerequisite`, `constraintViolation` | FR-062 | unhappy (empty case) |
| prerequisite on a missing option | `requiresOptionIds` naming no option | `.../requiresOptionIds/0`, `unresolvedReference` | FR-062 | unhappy (missing content) |
| self prerequisite | an option requires itself | `.../requiresOptionIds/0`, `constraintViolation` | FR-062 | unhappy (rule interaction) |
| prerequisite cycle | two options require each other | `.../requiresOptionIds/0`, `constraintViolation` | FR-062 | unhappy (rule interaction) |
| zero point cost | `pointCost` 0 | `.../pointCost`, `outOfRange` | FR-063 | unhappy (boundary) |
| false repeatable | `repeatable` false | `.../repeatable`, `invalidValue` | FR-063 | unhappy (invalid input) |
| subclass fact on option | `grantsSubclass` on an option | `.../facts/0`, `constraintViolation` | FR-064 | unhappy (invalid input) |
| ragged table | a row shorter than the header | `.../rows/1`, `outOfRange` | FR-066 | unhappy (boundary) |
| mixed column | text then integer in one column | `.../rows/1/0`, `wrongType` | FR-066 | unhappy (invalid input) |
| table with no rows | `rows` is `[]` | `.../rows`, `outOfRange` | FR-066 | unhappy (empty case) |
| side-by-side pairs | Nature's Ward as printed: Arid/Fire/Temperate/Lightning, then Polar/Cold/Tropical/Poison | normalised to the four rows of table TB in the order Arid, Polar, Temperate, Tropical | FR-066, FR-067 | unhappy (rule interaction) |
| option C | `startingEquipment` with `A`, `B`, `C` | none | FR-024 | happy |
| option D | `startingEquipment` with `D` | `.../startingEquipment/D`, `unknownField` | FR-024 | unhappy (invalid input) |
| missing B | `startingEquipment` with `A` only | `.../startingEquipment/B`, `missingField` | FR-024 | unhappy (missing content) |
| quantity zero | item `quantity` 0 | `.../items/0/quantity`, `outOfRange` | FR-024 | unhappy (boundary) |
| negative gold | `gold` -1 | `.../gold`, `outOfRange` | FR-024 | unhappy (boundary) |
| empty items | option `B` with `items` `[]`, `gold` 75 | none | FR-024 | unhappy (empty case) |
| multiclass exceeds core | multiclass grants Heavy armor to a Light class | `.../multiclassGrants/armorTraining/0`, `constraintViolation` | FR-026 | unhappy (rule interaction) |
| any-skill grant on a listed core | multiclass `any` on a class with a listed choice | `.../multiclassGrants/skillChoice`, `constraintViolation` | FR-026 | unhappy (rule interaction) |
| primary ability repeated | `AllOf` Strength, Strength | `.../abilities/1`, `duplicate` | FR-012 | unhappy (boundary) |
| any-of of one | `AnyOf` Strength | `.../abilities`, `outOfRange` | FR-013 | unhappy (boundary) |

## 5. Model

### Types (package `content`, class files)

- **`ClassDefinition`** (value object, `Entry`): the fields of FR-002 plus pure
  accessors `SubclassLevel`, `RowAt`, `RowFeatures` (`ResolvedRow`), `EntriesGainedAt`,
  `FeatGrantsAt`, `Column`, `Feature(FeatureID)` (`FeatureEntryLookup`, FR-099) and
  `Subclass(SubclassID)` (`SubclassLookup`, FR-074).
- **`SubclassDefinition`**: `id`, `name`, `source?`, `features`; `FeaturesAtOrBelow(level)`.
- **`ProgressionRow`**: `level`, `proficiencyBonus`, `features` (closed `RowFeature`:
  `RowClassFeature{FeatureID, annotation?}`, `RowSubclassFeature`), `spellcasting?` (closed
  `SpellcastingRow`: `SlotSpellcastingRow`, `PactSpellcastingRow`), `values` (map from
  `ClassColumnID` to closed `ColumnValue`: `ColumnCount`, `ColumnDice`, `ColumnBonus`,
  `ColumnFeetBonus`, `ColumnNone`); `Value(ClassColumnID)`.
- **`ClassColumn`**: `id`, `header`, `kind` (`count`, `dice`, `bonus`, `feetBonus`).
- **`ResolvedRow`**: `entries` (the row's class features resolved, printed order) and
  `hasSubclassFeature`.
- **`FeatureEntry`**: `id`, `name`, `level`, `text`, `facts` (closed `FeatureFact`),
  `options`, `tables`. **`SpellGrant`**: `level` (1 to 20), `spellIds` (non-empty, distinct).
- **`FeatureEntryLookup`** (closed `FeatureEntryFound(entry)`, `FeatureEntryNotFound(id)`);
  **`SubclassLookup`** (closed `SubclassFound`, `SubclassNotFound`).
- **`OptionPrerequisite`**: `minClassLevel?`, `requiresOptionIds`, `note?` (FR-062).
  **`FeatureOption`**: `id`, `name`, `text`, `prerequisite?`, `pointCost?`, `repeatable`,
  `facts`. **`FeatureTable`**: `name`, `columns`, `rows` of `TableCell` (closed
  `TableCellText`, `TableCellInt`), side-by-side pairs normalised (FR-066).
- **`AbilityRequirement`**, **`SkillChoice`**, **`WeaponProficiency`**,
  **`ArmorTrainingCategory`**, **`ClassToolProficiency`**, **`ClassStartingEquipment`**,
  **`MulticlassGrants`**, **`ClassSpellcastingTrait`**: all defined here, with `WeaponTrainingCategory`, `ClassEquipmentOption`, `ClassEquipmentItem`, `SkillChoiceListed`, `SkillChoiceAny`, `ClassToolProficiencySpecific` and `ClassToolProficiencyChoice`. No `EquipmentLine`; no
  equipment, weapon, armor, feat or spell entry type is reused.
- Ids: `ClassID`, `SubclassID`, `FeatureID` are defined with the other content ids
  (content-source-port.md; shared-types-and-ownership.md FR-005); `FeatureOptionID` and
  `ClassColumnID` are defined string types here with the same id rule, each with a `ParseX` parser and `MustX` (schema FR-114).

No aggregate, entity, event or service beyond the pure function `DeriveFeatureID` (which carries
the slug rule of FR-052). Class definitions
are reference data compared by value (architecture.md §3).

### Failures

Untrusted JSON problems are `Problem`s in the joined `Problem` error; a constructor given an invalid value returns an error wrapping `srderr.ErrInvalidInput` (`MustX` panics); no `PortError`: nothing here calls a port. Absence at lookup is `MissingContent`,
`SubclassNotFound` or `FeatureEntryNotFound`; an unresolved spell id is a whole-set problem (FR-059).

### Revision rows requested (shared-types-and-ownership.md FR-034, FR-035)

To be added to the revision table of content-schema-and-decoder.md in the change that lands
the class decoder (FR-095). None claims a version number.

| Row | Adds | Notes |
|---|---|---|
| R1-classes | The `classes` entry shape of section 4 (FR-002 to FR-077), `subclasses` nested | Replaces the status "requested" of the existing row; root key `classes` is in the baseline (schema FR-008). |
| R1-classes-ids | Uniqueness of `SubclassID` and `FeatureID` across a document, reported after the class-id duplicates (FR-004) | Extends schema FR-031 and FR-016 by two groups; no new `ProblemKind`. The cross-file check (content-classes-data.md FR-084) is `BundledSource` construction and is not part of this row; the schema spec's row text "includes the cross-file assembly check" is requested to be corrected to match. |
| R1-classes-spell-sites | Reference sites `(characterClass, classes/*/features/*/facts/*/grants/*/spellIds/*, spell)`, the same under `classes/*/subclasses/*/features/*` and `.../options/*/facts/*` (FR-059); no constraint | Registered per schema FR-126, checked by `ValidateSet` with no scope and no opt-out (shared-types-and-ownership.md FR-054). |

### Traceability from the schema spec's withdrawn rows (FR-097)

Schema FR-033 is satisfied by FR-004, FR-072 (cross-file: content-classes-data.md FR-084);
FR-046 by FR-011 to FR-017; FR-047 by FR-018; FR-048 by FR-019, FR-020; FR-049 by FR-021 to
FR-023; FR-050, FR-091, FR-092 and FR-045 (traceability only) by FR-024, FR-025; FR-051, FR-052
by FR-036 to FR-046; FR-053 by FR-044, FR-045, FR-051 to FR-068; FR-054 by FR-070 to FR-077.

The thin slice's Wizard entry is this spec's Wizard entry: ten feature entries referenced by
the rows (FR-044, FR-045, table F), `subclasses` possibly `[]` until Evoker lands.

## 6. Interface

No use case and no new port. Classes reach consumers through `Source`
(`Lookup(ctx, NewKey(KindCharacterClass, id))` returns the found variant holding the `ClassDefinition`, content-source-port.md), the
decoder (`JSONDecoder`) and `BundledSource` (content-source-port.md FR-059).

```go
// AbilityRequirement is closed: AllOf | AnyOf (FR-011).
type AbilityRequirement interface {
	Abilities() []ability.Ability              // printed order, a copy
	IsMetBy(meeting []ability.Ability) bool    // threshold belongs to the rule owner
}
func NewAllOf(abilities ...ability.Ability) (AllOf, error) // >= 1, no repeat
func NewAnyOf(abilities ...ability.Ability) (AnyOf, error) // >= 2, no repeat

type ClassDefinition struct{ /* unexported; implements Entry */ }
func (ClassDefinition) SubclassLevel() int                          // the GrantsSubclass entry's level
func (ClassDefinition) RowAt(level int) (ProgressionRow, error)     // error outside 1..20
func (ClassDefinition) EntriesGainedAt(level int) ([]FeatureEntry, error)
func (ClassDefinition) RowFeatures(level int) (ResolvedRow, error)
func (ClassDefinition) FeatGrantsAt(level int) ([]FeatCategory, error)
func (ClassDefinition) Column(id ClassColumnID) (ClassColumn, error) // error when undeclared
func (ClassDefinition) Feature(id FeatureID) FeatureEntryLookup
func (ClassDefinition) Subclass(id SubclassID) SubclassLookup

type SubclassDefinition struct{ /* unexported */ }
func (SubclassDefinition) FeaturesAtOrBelow(level int) ([]FeatureEntry, error)

// level is non-zero only for duplicate names (FR-053); error when the id exceeds 64 bytes.
func DeriveFeatureID(ownerID, name string, level int) (FeatureID, error)
```

Everything is synchronous and pure; class rules are added to `JSONDecoder` in
the decoder files of package `content`. Exports: package `content` exports every
type named in section 5 and above, `DeriveFeatureID` and the `ClassColumns`
constants for table CC; `content/contenttest` gains nothing.

## 7. Observability

Silent by design (observability.md §3): the domain types and
`DeriveFeatureID` emit nothing, and the class decoder rules live in the silent
`JSONDecoder` (schema FR-083) whose outcomes the existing `LoadSet`
records. **No signal is added or changed**; `doc/observability/signals.md` is unchanged.

1. **What can go wrong?** A malformed class document, a subclass at the wrong level, a granted spell id naming no spell, bundled data drifting from the PDF or lost in assembly (content-classes-data.md), or a consumer reading an undeclared column (a returned validation error, a bug).
2. **How would an operator know?** A rejected document is a `LoadSetRejected`: the `srd5e.content.load` span ends in error, `srd5e.content.loads` shows `srd5e.outcome=failure`, and `srd5e.content.decode_errors` counts problems by kind.
3. **What is needed to diagnose it?** Path and kind of each problem in the returned the joined `Problem` error (never in attributes, schema FR-085).
4. **Counted or timed?** Already: loads by outcome, problems by kind, entries by kind, load duration; per-class or per-feature counters would be high cardinality and actionless.
5. **Is the existing instrumentation still correct?** Yes: `srd5e.content.kind` already lists `character_class`, subclasses and features are nested and not counted entries, and `problem_kind` is unchanged (FR-096).
6. **What to monitor?** `entries_loaded{srd5e.content.kind=character_class}` equals 12 for the bundled set (alert on any drop); any `loads{srd5e.outcome=failure}` or non-zero `decode_errors` for shipped data (alert at one).
7. **Is anything now noisy?** No: no log is added and nothing fires per class, feature or row.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none added or changed | n/a | n/a | n/a | existing schema signals suffice |

## 8. Test plan

Levels (testing.md §3): **unit** (`class_*_test.go`, `package content_test`), **adapter** (class rules
of `JSONDecoder`, JSON text in `decode_class_test.go`, also on 32-bit per schema
FR-099), **architecture** (bundled data, layout, `internal/archtest`), **tool**
(`tools/extract_classes/*_test.go`, offline, synthetic input), **scenario** (public API, `scenarios/`). No
application-level test: no use case is added. "Cases X" are the tables of section 4, one
test per row.

| Ids | Level | Notes |
|---|---|---|
| FR-001 | unit, scenario | `Entry` key; bundled `barbarian` through `BundledSource`; a missing id gives the not-found variant holding `NewMissingContent(KindCharacterClass, id)` |
| FR-002 | adapter | One test per required field removed and per unknown key; empty arrays accepted for the four listed fields |
| FR-003, FR-069 | architecture | Staged: bundled ids a subset of SRC, each present class matches its rows, counts equal the sum of present rows of F; the pin fails when the twelfth lands (exactly twelve, totals 174 and 58) |
| FR-004 | adapter | Duplicate class, subclass and feature ids; same string in two id spaces; invalid id takes no part |
| FR-005, FR-006 | adapter | One document with problems in several parts, full ordered list, repeat equal; Cases PR (wrong container) |
| FR-007, FR-008 | unit, adapter | Agreement test of schema FR-094 over every case row; equality, mutating a returned slice leaves the value unchanged, input slice mutated after construction |
| FR-009 | adapter, architecture | `source` rules; `source` and `source.page` of every bundled class and subclass equal the first page in SRC |
| FR-010 | architecture | File placement, doc-comment lint, only `decode*.go` import `encoding/json`, none imports `os`, no other-context import |
| FR-011 to FR-015 | unit, adapter | Exhaustive `switch`; unknown `kind`; Cases AR; Cases FT (repeated, any-of of one, empty); order, copy on read, near-miss spelling |
| FR-016, FR-020, FR-027, FR-029, FR-038, FR-041, FR-042, FR-046, FR-057, FR-060, FR-065, FR-067, FR-068, FR-076, FR-077 | architecture | Bundled data against the oracle tables (CT1 skills and the p. 19 Class Overview rows, MC, SC, CC, F/RR/SF, FA incl. bundled categories per FA, OP, TB, SRC; the bundled Proficiency Bonus +2/+3/+4/+5/+6 changing at 5, 9, 13, 17), one test per class (and per level or column where stated); no other fact, option or table exists; subclass feature levels equal SF (a fixture with an extra level fails) |
| FR-017 to FR-019, FR-021 to FR-025 | unit, adapter, architecture | `D6` to `D12` accepted, `d8`, `D4`, `D20`, number rejected; zero/one/six/seven saves; skill variants (picks 0, too many, duplicate, unknown); weapon, armor, tool forms (empty filter, `[]` versus absent, picks 0, empty categories); Cases FT (equipment); singular and parenthetical names; bundled per CT1 to CT3 |
| FR-026 | unit, adapter | Cases FT (multiclass rows); subset check for each grant part |
| FR-028, FR-030 to FR-032 | unit, adapter | Both variants, unknown kind, ability spelling; Cases SP |
| FR-033, FR-034 | architecture | Bundled slot, cantrip, prepared values against FS, HS, SQ per class and column; monotonic check (a fixture with a transposed row fails naming class, level, column) |
| FR-035 | doc review | No test here; the agreement scenario is a row of spellcasting.md and derived-stats.md (decided, Q5) |
| FR-036 to FR-045 | unit, adapter | Empty columns (Wizard), duplicate id, blank header, unknown kind; Cases PR (19/20/21 rows, level order, missing/undeclared column, markers, empty cell, annotation, unresolved, later entry, never listed); bonus 0 rejected, 1 accepted, changes at 5, 9, 13, 17; Cases CV |
| FR-047 to FR-050, FR-099 | unit | Levels 0, 1, 20, 21, negative; `EntriesGainedAt` Barbarian 1, 6, 17; `RowFeatures` ASI at 8 resolves to the level 4 entry and sets the `hasSubclassFeature` flag on marker rows; `FeatGrantsAt` Fighter 6, Rogue 10, Wizard 5 (none), Barbarian 19; declared/undeclared column ids; `Feature` found, unknown id and subclass feature id not found |
| FR-102 | architecture | Every bundled class feature `name` equals table F; the "Fighting Style" and "Spellcasting" entries resolve by `Feature(id)` name for each owner; the Warlock has no "Spellcasting" |
| FR-051 to FR-055 | unit, adapter, architecture | Level 0/21 rejected, 1/20 accepted; Cases ID; every bundled id equals the derivation incl. the Barbarian pair; verbatim text round trip (apostrophe, line feed), no heading prefix or table in bundled text; exhaustive `switch`, unknown kind, duplicate fact |
| FR-056 to FR-058, FR-061 to FR-064, FR-066, FR-070 to FR-073 | unit, adapter | Cases FT (subclass fact, a subclass feature carrying the fact rejected, category, grants, options, prerequisites, point cost, option fact, tables); empty subclass feature list rejected, `subclasses: []` accepted; subclass feature at levels 2 and 3, none at subclass level; prefix and document-wide duplicate; `ClassDefinition` constructor returns an error for a low subclass or subclasses with no subclass fact |
| FR-059 | adapter, architecture | Decoder alone accepts an unknown spell id; `ValidateSet` reports `unresolvedReference` naming class, feature, id (fixture); the bundled set resolves every id; pin: a class with such facts is not bundled before its spells |
| FR-074, FR-075 | unit | Found/not found carry data; `FeaturesAtOrBelow` at 1, 3, 6, 20 for a bundled subclass (SF) |
| FR-095, FR-097 | doc review | The schema revision table holds the three rows R1-classes, R1-classes-ids and R1-classes-spell-sites by these names, and no version number; each withdrawn schema row maps to a row above |
| FR-096, FR-098 | architecture | `ProblemKind` and `Kind` unchanged (AST scan); no class spec defines `ClassDefinition`, `ProgressionRow`, `SubclassDefinition` or `FeatureEntry`, no duplicate exported names |

- **Doubles and contract suites:** none (no port). Fixtures are synthetic classes from a test builder.
- **Mutation testing:** the class files of package `content` at least 80 with `gomu`, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors
  (testing.md §8): `AllOf`/`AnyOf`, row and level bounds, the FR-026 subset rule, the
  spellcasting-variant match, the prerequisite cycle check; mutate the decoder rules and
  tool too, with attention to index and level boundaries.
- **Telemetry tests:** none added; one `LoadSet` test (schema FR-079 to FR-086) loads a
  rejected class document and asserts one `decode_errors` increment per problem kind.
- **Not covered:** prose faithfulness beyond what content-classes-data.md FR-089 can compare
  (review task); agreement with spellcasting.md tables and `ProficiencyBonus` (FR-035,
  downstream owners); cross-file duplicate ids and the oracle fixture (content-classes-data.md
  FR-084, FR-093; tested in its section 8; BundledSource coverage is
  content-source-port.md FR-059).

**Implementation status.**

- Done (Phase 2, step 5b, package `content`, files `class_ids.go`, `class_ability.go`, `class_training.go`, `class_spellcasting.go`, `class_progression.go`, `class_feature.go`, `class_definition.go`, `class_check.go`, `class_table.go`, `class_entry.go`, `decode_class.go`, `decode_class_rows.go`, `decode_class_features.go`): the entry type `ClassDefinition` (an `Entry` that implements `References`) and `SubclassDefinition` with constructors and `MustClassDefinition`; `AbilityRequirement` (`AllOf`, `AnyOf`), `SkillChoice`, `WeaponProficiency`, `ClassToolProficiency`, `ClassStartingEquipment`, `MulticlassGrants`, `ClassSpellcastingTrait`, `SpellcastingRow`, `ClassColumn`, `ColumnValue`, `RowFeature`, `ProgressionRow`, `FeatureFact`, `SpellGrant`, `FeatureOption`, `OptionPrerequisite`, `FeatureTable`, `TableCell`, `FeatureEntry`, the lookups `FeatureEntryLookup` and `SubclassLookup`, `ResolvedRow`, the ids `ClassColumnID` and `FeatureOptionID`, and `DeriveFeatureID`; the decoder row registered as `classes`; the reference sites of FR-059; and FR-001, FR-002, FR-004 to FR-008, FR-010 to FR-015, FR-017 to FR-019, FR-021 to FR-026, FR-028, FR-030 to FR-032, FR-036, FR-037, FR-039, FR-040, FR-043 to FR-045, FR-047 to FR-053, FR-055, FR-056, FR-058, FR-059, FR-061 to FR-064, FR-066, FR-070 to FR-075, FR-095, FR-096, FR-099 (tested on synthetic classes: a full-caster shape with 20 rows and ten feature entries, a Pact Magic class, a class with one column of each kind), and the decoder halves of FR-009 and FR-038.
- Choices the spec left open: the prerequisite cycle check is one strongly-connected-components pass per entry and every repeat check uses a set, so the cost is linear in the size of the document (the FR-045 check builds one set of the class features each row names, and `TestDecodeClassManyFeaturesInOneRow` guards it; `FeatureEntry.valid` is constant time: only the zero value is invalid); the rules between parts live once, as `ClassDefinition.problems` and `FeatureEntry.problems`, and the constructor returns the first as a `BlockError` while the decoder reports all of them with their paths; the decoder reports them only when every field of the entry decoded cleanly (FR-005); the order of the cross-field problems is multiclass grants, feature id prefixes, row levels and values, spellcasting rows, row references, then the subclass rules; a feature entry's own cross problems (a spell grant below the entry level, an option prerequisite naming a missing option, itself or a cycle) follow that entry's fields; every count a class stores is bounded by `ClassMaxCount` (2^31 - 1), so a value is the same on every platform integer width; a column value that is a string other than "—" is `invalidValue` for every kind (FR-040 over the Cases CV row, which said `wrongType` for "2"); a repeated weapon category is `duplicate` at the later element; a present `requiresOptionIds` is non-empty and distinct; a subclass or feature id that repeats one of the same id space is reported once per document by the decoder (FR-004), and `NewClassDefinition` does not reject repeated feature ids inside one class; `Feature(id)` returns the first entry with the id; the row part of a Pact class is recognised by its `slotLevel` key; `BlockProblem` is reused unchanged (`BlockRaggedRow` for a short table row, which the decoder reports as `outOfRange` at the row as FR-066 says); the types hold slices and a map and have no `Equal` method, so compare classes with `Set.Equal`.
- Done (Phase 2, step 6, `content/data/class_wizard.json`): the Wizard of the thin slice as a `ClassDefinition` (core traits, the 20-row Wizard Features table, ten feature entries with the facts of table FA, no subclasses), served by `data.BundledSource`; its shape is pinned by `content/data/data_wizard_test.go` against the printed table. The other eleven classes, the extraction tool and the full oracle tables stay in Phase 3.
- Mutation (gomu 0.2.1, per file, package tests): `class_ability.go` 85 (1 survivor, `len > 0` vs `!= 0`, equivalent); `class_ids.go` 86.3 (8 survivors, all annotated equivalent in the source); `class_table.go` 89.1 (2 survivors, `len == 0` vs `<= 0`, equivalent); `class_spellcasting.go` 100 (no survivors).
- Mutation (manual fallback, no same-named `_test.go`; one mutation at a time, `go test ./content`, file restored and diffed identical): `decode_class_rows.go` 31/31 killed (score 100; one gap, `o.skip` on a wrong row feature kind, closed by a test); `class_entry.go` 41/41 killed (100; `issue.blockError` mapping closed by `TestIssueBlockError`); `class_check.go` 53/53 killed (100; id prefix separator, first-wins feature lookup and tool pick maxima closed by new cases); `class_feature.go` 44/47 killed (93.6, 3 survivors all equivalent: `factKey` prefix "p", and `Repeatable` and `Prerequisite` in `FeatureOption.valid`, which `NewFeatureOption` never checks; annotated in the source).
- Mutation (gomu 0.2.1, per file): `decode_class_features.go` 83.7 (15 survivors, all annotated equivalent in the source; first run 79.1, closed by `TestDecodeTableRowsDropsDeviatingRows`); `decode_class.go` 92.2 (8 survivors, 1 error).
- Mutation (manual fallback, 16 mutations each, `go test ./content`, restored and diffed identical): `class_progression.go` 16/16 killed after `TestColumnNoneFitsOnlyValidKinds` closed `ColumnNone.fits` (`k.Valid()` to `true`); `class_training.go` 16/16 killed.
- Mutation (gomu 0.2.1, per file): `class_definition.go` 98.0 (159 mutants, 148 killed, 3 survived, 8 not viable); score above the 80 gate, survivors not itemised by the tool output and left unclosed.

Not yet implemented: FR-003 (FR-069 and FR-102 go with it: the plan's subset assertions for 003, 009, 038 and 102 and the staged count check of 069 move to step 6, because no class is bundled in 5b so each would be vacuous), FR-025 (the bundled choice lines "Musical Instrument of your choice" and the Monk's; the storage rule itself is tested), FR-009 (bundled sources), FR-016, FR-020, FR-027, FR-029, FR-033 to FR-035, FR-038 (bundled values), FR-041, FR-042, FR-046, FR-054 (bundled text), FR-057 (bundled categories), FR-060, FR-065, FR-067 to FR-069, FR-076, FR-077, FR-102 (the bundled data against the oracle tables, step 6 and Phase 3, in content-classes-data.md), the `ClassColumns` constants for table CC and the side-by-side normalisation of FR-066 (the extraction tool, Phase 3), the AST scans of FR-096 and FR-098 (no archtest row added), and FR-097 (doc review, done by the revision rows of the schema spec).

## 9. Open questions and decisions

Decided, with amendments requested of other specs (nothing escalated):

| Question | Decision | Amendment requested |
|---|---|---|
| Q5. Where does agreement between the bundled Features-table values and spellcasting.md / derived-stats.md live? | Decided: in those specs, as a scenario row each (shared-types-and-ownership.md FR-013); this spec's bundled columns are the authoritative copy (FR-035) and both copies are checked against the PDF independently. | spellcasting.md and derived-stats.md add the agreement scenario row (or adopt `ProgressionRow.spellcasting` and drop their tables). |
| Q6. Who checks cross-file duplicate ids in the bundled set? | Decided: `BundledSource` construction (content-source-port.md FR-059, which exists) by a returned construction error wrapping `srderr.ErrInvalidInput`; the requirement is content-classes-data.md FR-084 (exists, with its test row). | None; content-source-port.md FR-059 needs no change. |
| Q7. Who owns a shared `DeriveContentID`? | Decided: none exists; this spec's `DeriveFeatureID` (FR-052) stays the only derivation until another spec needs one; the apostrophe divergence is settled in content-species-backgrounds-feats.md FR-004. | If a shared helper is wanted, content-source-port.md may add `DeriveContentID` with the FR-052 slug rule and a case table, and `DeriveFeatureID` then delegates to it. |

Settled (one line each):

- D1 subclasses are nested in the class entry (`Kind` has no subclass kind,
  schema FR-142; FR-071, FR-074); promoting `subclass` to a kind would be a
  breaking change.
- D2 class spell lists are derived from `Spell.classes` in content-spells.md (FR-120), never stored.
- D3 only three facts are structured; everything else is text keyed by `FeatureID`
  (shared-types-and-ownership.md FR-023; schema Q7).
- D4 feature ids are `<owner>_<slug>` by the rule of FR-052 in `DeriveFeatureID`, with
  `_level_<n>` only for a repeated name (FR-053). No shared `DeriveContentID` exists (Q7);
  content-species-backgrounds-feats.md FR-004 differs for apostrophes (`dragon_s_breath`),
  and no bundled species, background or feat id contains one.
- D5 declared columns, not typed per-class fields (FR-036 to FR-042).
- D6 nine slot columns for every slot caster, "—" being 0 (FR-031); D7 "—" is the string
  `"—"` in column values, `[]` in a feature cell, 0 in a slot count, never null (schema FR-020).
- D8 rows reference entries by id (FR-044, FR-045); D15 Ability Score Improvement and Epic
  Boon live once at the first level and later rows reference them (FR-049).
- D9 choice-naming item lines are stored verbatim (FR-025); D10 `ClassToolProficiency` is a
  separate class-side type (FR-023), and `ArmorTrainingCategory` and `WeaponTrainingCategory`
  deliberately mirror content-equipment.md's `ArmorCategory` (FR-104) and `WeaponCategory`
  (FR-101) under shared-types-and-ownership.md FR-047; the consumer (character-creation,
  equipment-rules) maps them by name; D14 `AnyOf` of one is not representable (FR-013).
- D11 the decoder checks the owner prefix and uniqueness, not exact derivation (FR-052,
  FR-004); D12 and D13 (oracle fixture, one file per class) moved to content-classes-data.md.
- Q1 the twelve "Level 3: <Class> Subclass" features are ordinary entries carrying
  `GrantsSubclass` (FR-056). Q2 232 "Level N:" headings were counted in pp. 28-82; the tool
  re-checks (content-classes-data.md FR-086, FR-088). Q3 no embedded table beyond TB exists; the tool reports any
  other. Q4 staging of the all-twelve and full-spell-set checks: FR-003, FR-069
  stage them (and content-classes-data.md FR-083, FR-100); FR-059 needs no pin (shared-types-and-ownership.md FR-054).

## 10. Changelog impact

Under `Unreleased`:

- **Added:** every type and function named in section 5 and section 6 (phase 2); the
  `classes` entry shape and its decoder rules; the bundled content is content-classes-data.md.
- **Changed:** content-schema-and-decoder.md gains R1-classes, R1-classes-ids and
  R1-classes-spell-sites; the thin slice's Wizard becomes the full Wizard entry.

Not **BREAKING** while unreleased. After first release, **BREAKING**: adding a variant to
`FeatureFact`, `ColumnValue`, `RowFeature`, `SpellcastingRow`, `ClassSpellcastingTrait` or
`AbilityRequirement`; adding or removing a bundled feature, option, column or id; changing a
bundled value (unless it fixes a transcription bug against the PDF); changing the id
derivation.
