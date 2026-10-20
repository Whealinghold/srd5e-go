# Bundled content thin slice

| | |
|---|---|
| Status | partially implemented |
| Context | `content` |
| Roadmap phase | Types 2; slice data 2; bulk data 5 |
| Depends on | `content-schema-and-decoder.md`, `content-source-port.md` |

Roadmap detail: [../roadmap.md](../roadmap.md). The slice data ships in phase 2. Class types ship in phase 2
(shared-types-and-ownership.md FR-014); bulk class data is phase 3
(content-classes.md FR-003; bundled by content-classes-data.md; shared-types-and-ownership.md FR-025); bulk species,
spells and the rest are phase 5.

## 1. Purpose and scope

Phase 2 ships "a thin slice of bundled SRD data (a few species, one class, a
few spells) to validate the schema before bulk entry" (`doc/roadmap.md`). This
spec fixes that slice: which SRD 5.2.1 entries are bundled, what the data must
carry, how it is exposed, and why these entries. It is small enough to
transcribe and review against the PDF and varied enough that every structural
shape of the bulk data appears at least once. It is **self-consistent under
whole-set validation** (shared-types-and-ownership.md FR-054): every id it
references is an entry it contains, so the merged slice passes
`content.ValidateSet` and no reference site opts out.

**Ownership.** This spec owns the slice's membership, the pinned values of its
species and spells, the Wizard's place in it, and its loading and telemetry
pins. It owns no type and no adapter (shared-types-and-ownership.md FR-025,
FR-035, FR-046, FR-047); section 5 lists the owners. The data is written to
their shapes and this spec depends on none of the data specs (FR-047 there).

**Oracle.** Tables T1 to T4, T6, T7, the pinned phrases and the section 2
citation rows are SRD text, so they live as Go fixture values (or JSON files) under
`testdata/srd/thin_slice/` (shared-types-and-ownership.md FR-033), imported
by the data-pin tests. The banned-term list of FR-008 is not SRD text and stays
in test code.

**Closure.** The Elf's three lineages name nine spells (`levelBenefits`
`spellId`, a registered reference site, content-schema-and-decoder.md FR-126), so
the slice bundles all nine: Prestidigitation, Detect Magic, Misty Step (High
Elf) and Dancing Lights, Darkness, Druidcraft, Faerie Fire, Longstrider, Pass
without Trace.

**Text pins.** The owners declined typed trait facts, choice picks, lineage
grant semantics and a structured `SpellScaling` (section 5.1). Species traits
are `NamedText` and spell effects are faithful text. Where a requirement says a
trait or spell "MUST state" a fact, the pin is that the entry's text contains
the SRD's wording, as phrases the test author picks from the cited page; the
data holds no typed copy (content-species-backgrounds-feats.md Q15). Structure
is pinned only for names, membership, `creatureType`, `sizes`, `sizeText`,
`speed`, the SX-1 `choices` and `levelBenefits` (`damageType`, `spellId`) and
the spell fields of content-spells.md. FR-067, FR-078 and the pins that read
entries are enabled when the decoder implements the owners' rows
(R1-species-backgrounds-feats, R1-classes, R1-spells in
content-schema-and-decoder.md) in the same change as the bundled data
(content-spells.md FR-092).

**Wizard.** The slice's Wizard is content-classes.md's entry (core traits,
Wizard Features table, ten feature entries); its value pins (FR-033 to FR-047
here) are withdrawn to it. This spec keeps membership (FR-002) and citation
(FR-007). Its `subclasses` list is empty until content-classes-data.md bundles Evoker.

**Lifecycle.** These phase-2 pins lapse when the bulk specs land, replaced by
their own inventory pins (as content-equipment.md Q2 does): membership FR-001 to
FR-004, closure FR-063, absent-id pins FR-064 and FR-065, the count of FR-067
(17), the per-kind counts of FR-078 (4, 1, 12) and the matching advice in
section 7. `characterClass` pins lapse in phase 3 (content-classes-data.md bundles
all twelve classes and their subclasses), species and spell pins in phase 5. FR-066 does
not lapse. Species value pins (FR-009 to FR-032) are superseded by the case
tables of content-species-backgrounds-feats.md (Q15); the spell pins FR-003 and
FR-007 are replaced, and FR-060 with T7 reduces to presence checks of
`cantripUpgrade` and `higherLevel`, when content-spells.md bundles (FR-094,
FR-115).

The slice:

| Kind | Entries | Count |
|---|---|---|
| `species` | Dragonborn, Dwarf, Elf, Human | 4 |
| `characterClass` | Wizard (class, class features, Wizard Features table; no subclass) | 1 |
| `spell` | Fire Bolt, Prestidigitation, Detect Magic, Magic Missile, Misty Step, Fireball; Dancing Lights, Darkness, Druidcraft, Faerie Fire, Longstrider, Pass without Trace | 12 |

### Rationale

The slice covers every structural shape of the bulk data: flat species (Dwarf);
an enumerated choice table (Dragonborn's ten Draconic Ancestors);
character-level thresholds kept as text (5, 11, 17 in Dragonborn and Fire
Bolt); spells granted at levels 1, 3, 5 through `levelBenefits` and lineage
text overriding a base value (Elf); size as a choice (Human); picks from absent
kinds kept as text; a class with a 20-row slot table and level-keyed features
(Wizard); spell levels 0 to 3, all casting-time, range, duration and component
forms, scaling as `cantripUpgrade`, `higherLevel` or neither, class lists of
size 1, 2, 3, 4 and 8; resolving references and lookups outside the slice.

**Why Wizard, not Fighter.** The class links the class kind to the spell kind
(spell list, pp. 79 to 82, derived by content-spells.md FR-120) and carries the
heaviest table (20 rows, slot columns); a non-spellcaster (Fighter, p. 47)
would leave those unvalidated. Wizard has the plainest full-caster table, with
no class-specific resource column (unlike Bard, Cleric, Druid, Sorcerer,
Warlock); those arrive with bulk class data.

**Why these spells.** The first six are on the Wizard list and cover the shapes
above; Prestidigitation, Detect Magic and Misty Step are the High Elf lineage
spells. The other six are the Drow and Wood Elf lineage spells, present so every
reference resolves. Magic Missile and Detect Magic are two of the six level 1
spells the Wizard's Spellbook text recommends (p. 78).

**Out of scope:** other species, backgrounds, feats, equipment, magic items,
monsters; other classes and every subclass including Evoker (p. 82;
content-classes.md); other spells and the Wizard spell list tables (derived by
`classSpellList`); character creation (p. 77) and rules maths (applied by
`character` and `combat`); parsing prose (FR-062); the decoder, port,
adapters and notice (content-schema-and-decoder.md FR-120 to FR-124;
content-source-port.md FR-059 to FR-064; content-licensing-and-notice.md).
Nothing encountered looked like core fifth-edition material absent from the PDF.

## 2. Source

Printed page equals PDF page (PDF page 5 prints "5", page 123 prints "123").

| Entry | SRD 5.2.1 section | Printed page(s) |
|---|---|---|
| Species, general rules | "Character Species", "Parts of a Species" | 83, 84 |
| Dragonborn | "Character Species", "Dragonborn" | 84 |
| Dwarf | "Character Species", "Dwarf" | 84 |
| Elf | "Character Species", "Elf" | 84, 85 |
| Human | "Character Species", "Human" | 86 |
| Wizard core traits and Wizard Features table | "Wizard", "Core Wizard Traits", "Wizard Features" | 77 |
| Wizard Level 1 Spellcasting | "Wizard Class Features", "Level 1: Spellcasting" | 77, 78 |
| Wizard Ritual Adept, Arcane Recovery, Scholar | "Level 1: Ritual Adept", "Level 1: Arcane Recovery", "Level 2: Scholar" | 78 |
| Wizard spell list (cross-check only) | "Wizard Spell List" | 79 to 82 |
| Spell entry layout and rules | "Spells", "Casting Spells" (Spell Level, School of Magic, Class Spell Lists, Casting Time, Range, Components, Duration, Using a Higher-Level Spell Slot) | 104 to 106 |
| Dancing Lights | "Spell Descriptions", "Dancing Lights" | 121 |
| Darkness | "Spell Descriptions", "Darkness" | 122 |
| Detect Magic | "Spell Descriptions", "Detect Magic" | 123 |
| Druidcraft | "Spell Descriptions", "Druidcraft" | 126 |
| Faerie Fire | "Spell Descriptions", "Faerie Fire" | 129 |
| Fireball | "Spell Descriptions", "Fireball" | 131 |
| Fire Bolt | "Spell Descriptions", "Fire Bolt" | 132 |
| Longstrider | "Spell Descriptions", "Longstrider" | 145 |
| Magic Missile | "Spell Descriptions", "Magic Missile" | 146 |
| Misty Step | "Spell Descriptions", "Misty Step" | 150 |
| Pass without Trace | "Spell Descriptions", "Pass without Trace" | 151 |
| Prestidigitation | "Spell Descriptions", "Prestidigitation" | 154 |
| Attribution statement | "Legal Information" | 1 |

### Entry citations (FR-007)

The envelope's `source` is `{document, section, page}` with one integer `page`
(content-schema-and-decoder.md FR-034, FR-090): the first printed page of the
entry. `document` is `SRD 5.2.1` for every entry. The `section` strings follow
the owners (content-species-backgrounds-feats.md FR-007; content-spells.md
FR-008; content-classes.md FR-009). Exact values:

| Entry id | `section` | `page` |
|---|---|---|
| `dragonborn`, `dwarf`, `elf` | `Dragonborn`, `Dwarf`, `Elf` | 84 each |
| `human` | `Human` | 86 |
| `wizard` | `Wizard` | 77 |
| each of the 12 spells | `Spell Descriptions` | the page of its heading in the table above: `dancing_lights` 121, `darkness` 122, `detect_magic` 123, `druidcraft` 126, `faerie_fire` 129, `fireball` 131, `fire_bolt` 132, `longstrider` 145, `magic_missile` 146, `misty_step` 150, `pass_without_trace` 151, `prestidigitation` 154 |

The `wizard` entry uses the class's first page (p. 77). A `FeatureEntry` and a
`ProgressionRow` carry no `source`; the class-level citation covers them.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Species | Content kind (`species`) | Never "Race". Parts: Creature Type, Size, Speed, Special Traits (pp. 83 to 84). |
| Class | Content kind (`characterClass`) | Wizard. Core Wizard Traits and Wizard Features: content-classes.md (p. 77). Hit Point Die "D6 per Wizard level" is the `HitPointDie` member `d6` (content-schema-and-decoder.md FR-132; JSON "D6"), not the kernel `Die`; the value pin is content-classes.md FR-017. |
| Spell | Content kind (`spell`) | Level, School of Magic, class list, Casting Time, Range, Components, Duration, effect text (pp. 104 to 106). |
| Schools of Magic | Closed set | Abjuration, Conjuration, Divination, Enchantment, Evocation, Illusion, Necromancy, Transmutation (p. 105). |
| Casting Time, Ritual | `Spell.castingTimes` and `Spell.ritual` | "Action or Ritual" (p. 123); Ritual rule under "Casting without Slots" (p. 104). |
| Cantrip Upgrade | `Spell.cantripUpgrade` (text) | "levels 5 (2d10), 11 (3d10), and 17 (4d10)" (p. 132). They are character levels: SRD 5.2.1, "Multiclassing" (Spellcasting, Cantrips), p. 25: the increase "is based on your total character level, not your level in a particular class, unless the spell says otherwise". Text only (content-spells.md FR-115). |
| Using a Higher-Level Spell Slot | `Spell.higherLevel` (text) | By slot level (pp. 131, 146). Text only (content-spells.md FR-115). |
| Ability, Skill, Damage Type, Size, Creature Type, Condition | Typed vocabulary | `Ability` from `the kernel packages` (ability-scores.md FR-001); `Skill`, `DamageType`, `CreatureSize`, `CreatureType`, `ConditionKind` from content-schema-and-decoder.md (FR-106 to FR-110). SRD spellings, never raw strings. The slice uses no `RechargeTrigger`. |
| Proficiency Bonus, Long Rest, Bonus Action, Resistance, Advantage, Darkvision, Tremorsense | Rules terms | "Rules Glossary"; verbatim trait text only (section 5.1, U1). |
| absent spell id | Package term | A `SpellID` no slice entry has. The slice has none in a structured field: every lineage `levelBenefits[].spellId` is a registered reference site (content-schema-and-decoder.md FR-126) and resolves (FR-063). The term is only for a lookup of an unbundled SRD spell (FR-064), which is `LookupNotFound`, never `unresolvedReference`. A spell's class list holds `ClassID` identity values, not references. |

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids.

Entry ids: `dragonborn`, `dwarf`, `elf`, `human`, `wizard`, `fire_bolt`,
`prestidigitation`, `detect_magic`, `magic_missile`, `misty_step`, `fireball`,
`dancing_lights`, `darkness`, `druidcraft`, `faerie_fire`, `longstrider`,
`pass_without_trace`: the SRD name in lowercase with underscores, matching
`[a-z][a-z0-9_]*` (shared-types-and-ownership.md FR-007; content-schema-and-decoder.md
FR-030), in the JSON and in `SpeciesID`, `ClassID` and `SpellID`.

**Loading unit.** Three documents, one per kind, as JSON files under `content/data/`
embedded directly with `go:embed` (no generator step, shared-types-and-ownership.md FR-030): stems `species`, `spells`, `class_wizard`
(content-schema-and-decoder.md FR-120; content-species-backgrounds-feats.md
FR-053; content-classes-data.md FR-083), files `species.json`, `spells.json`,
`class_wizard.json`. Bulk
data replaces these files (content-spells.md FR-094). `content.Decoder.Decode`
and `content.LoadSet` take one document each (content-schema-and-decoder.md
FR-077, FR-078), so a full load is three decodes. Pins that read entries use a
`data.BundledSource` built from exactly the three texts
(content-source-port.md FR-059). The duplicate-id check is per document and kind
(content-schema-and-decoder.md FR-031).

### 4.1 Membership, provenance and fidelity

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | The bundled `species` entries MUST be exactly `dragonborn`, `dwarf`, `elf` and `human`: none missing, none extra. | SRD 5.2.1, "Character Species", pp. 84 to 86 | happy |
| FR-002 | The bundled `characterClass` entries MUST be exactly `wizard`. No subclass entry (including Evoker) is bundled: `Lookup` of `Key(characterClass, evoker)` and of `Key(characterClass, fighter)` MUST be `LookupNotFound`. Lapses in phase 3. | SRD 5.2.1, "Wizard", p. 77 | unhappy (boundary: nothing beyond the slice) |
| FR-003 | The bundled `spell` entries MUST be exactly `fire_bolt`, `prestidigitation`, `detect_magic`, `magic_missile`, `misty_step`, `fireball`, `dancing_lights`, `darkness`, `druidcraft`, `faerie_fire`, `longstrider` and `pass_without_trace`: none missing, none extra. | SRD 5.2.1, "Spell Descriptions", pp. 121, 122, 123, 126, 129, 131, 132, 145, 146, 150, 151, 154 | happy |
| FR-004 | The slice MUST contain no entry of any `Kind` other than `species`, `characterClass` and `spell`: `ListKind` returns an empty slice for each other kind (today `background`, `feat`, `equipment`, `magicItem`, `monster`; the `poison`, `trap` and `contagion` kinds of content-schema-and-decoder.md FR-129 join the pin when declared). | none (spec scope decision) | unhappy (empty case) |
| FR-005 | Each entry MUST be addressable by its id under its kind as a `Key`. | content-source-port.md FR-008 | happy |
| FR-006 | Building a `data.BundledSource` from the three slice texts (17 entries) MUST complete without returning a `*data.BundledInvalidError` (a duplicate key would be a `duplicate` reference problem, content-source-port.md FR-061) and without the `*content.DuplicateKeyError` of content-source-port.md FR-021 returned by a bare `content.Catalogue`: no two entries share a kind and id. | content-source-port.md FR-021, FR-061; testing.md §4 (duplicate entries) | unhappy (duplicate) |
| FR-007 | Every entry MUST carry the `source` value (`document`, `section`, `page`) of the "Entry citations" table of section 2. | AGENTS.md, "Licensing and source material"; content-schema-and-decoder.md FR-034, FR-090; content-species-backgrounds-feats.md FR-007; content-spells.md FR-008; content-classes.md FR-009 | happy |
| FR-008 | Entry text (trait, feature and spell descriptions, table cell text) MUST contain the pinned phrases that this section quotes for that entry, and MUST NOT contain any whole-word, case-insensitive banned term: `race`, `races`, `subrace`, `subraces` (word boundaries, so "trace" and "grace" never match). The list is a test constant; extending it is a test change, not a spec change. | AGENTS.md, "Licensing and source material" | unhappy (invalid input: non-SRD terms) |

### 4.2 Species: Dwarf

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-009 | `dwarf` MUST have Creature Type Humanoid, Size Medium and Speed 30 feet. | SRD 5.2.1, "Dwarf", p. 84 | happy |
| FR-010 | `dwarf` MUST have exactly four traits, named Darkvision, Dwarven Resilience, Dwarven Toughness and Stonecunning, and the Darkvision text MUST state a range of 120 feet. | SRD 5.2.1, "Dwarf", p. 84 | happy |
| FR-011 | The Dwarven Resilience text MUST state Resistance to Poison damage and Advantage on saving throws to avoid or end the Poisoned condition. | SRD 5.2.1, "Dwarf", p. 84 | happy |
| FR-012 | The Dwarven Toughness text MUST state a Hit Point maximum increase of 1 and a further increase of 1 for each level gained. | SRD 5.2.1, "Dwarf", p. 84 | happy |
| FR-013 | The Stonecunning text MUST state: Bonus Action, Tremorsense range 60 feet, duration 10 minutes, uses equal to Proficiency Bonus, all expended uses regained on a Long Rest, and the condition that the Tremorsense works only while on or touching a stone surface (natural or worked). | SRD 5.2.1, "Dwarf", p. 84 | happy |

### 4.3 Species: Dragonborn

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `dragonborn` MUST have Creature Type Humanoid, Size Medium, Speed 30 feet and exactly five traits named Draconic Ancestry, Breath Weapon, Damage Resistance, Darkvision and Draconic Flight; the Darkvision text MUST state a range of 60 feet. | SRD 5.2.1, "Dragonborn", p. 84 | happy |
| FR-015 | `dragonborn` MUST have one `choices` element (SX-1) whose `traitName` is Draconic Ancestry and whose options are exactly the ten dragon kinds of table T1, each a `DamageTypeOption` with its `damageType`. | SRD 5.2.1, "Dragonborn", "Draconic Ancestors", p. 84; content-species-backgrounds-feats.md FR-016, FR-060 | per case (T1): happy / unhappy (boundary) |
| FR-016 | The Breath Weapon text MUST state: usable when taking the Attack action, replacing one attack; area a 15-foot Cone or a 30-foot Line 5 feet wide, shape chosen each use; a Dexterity saving throw with DC 8 plus Constitution modifier plus Proficiency Bonus; damage of the type set by Draconic Ancestry on a failed save and half on a success. | SRD 5.2.1, "Dragonborn", p. 84 | happy |
| FR-017 | The Breath Weapon text MUST state the damage by character level of table T2: 1d10 at level 1, 2d10 at 5, 3d10 at 11, 4d10 at 17. The thresholds stay in the text (content-species-backgrounds-feats.md Q16). | SRD 5.2.1, "Dragonborn", p. 84 | per case (T2): happy / unhappy (boundary) |
| FR-018 | The Breath Weapon text MUST state uses equal to Proficiency Bonus, with all expended uses regained on a Long Rest. | SRD 5.2.1, "Dragonborn", p. 84 | happy |
| FR-019 | The Damage Resistance text MUST state Resistance to the damage type chosen for Draconic Ancestry, and MUST NOT name a fixed damage type. | SRD 5.2.1, "Dragonborn", p. 84 | unhappy (rule interaction: depends on a choice) |
| FR-020 | The Draconic Flight text MUST state: available from character level 5, Bonus Action, 10 minutes or until retracted or the Incapacitated condition, Fly Speed equal to Speed, usable once per Long Rest. | SRD 5.2.1, "Dragonborn", p. 84 | happy |

### 4.4 Species: Elf

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-021 | `elf` MUST have Creature Type Humanoid, Size Medium and Speed 30 feet. | SRD 5.2.1, "Elf", p. 84 | happy |
| FR-022 | `elf` MUST have exactly five traits named Darkvision, Elven Lineage, Fey Ancestry, Keen Senses and Trance; the Darkvision text MUST state a range of 60 feet. | SRD 5.2.1, "Elf", pp. 84 to 85 | happy |
| FR-023 | `elf` MUST have one `choices` element (SX-1) whose `traitName` is Elven Lineage and whose options are exactly the three `LineageOption`s of table T3, each with `levelBenefits` for levels 1, 3 and 5 (content-species-backgrounds-feats.md FR-018, FR-061): the verbatim Level 1 text and the `spellId` of its cantrip, then the level 3 and level 5 `spellId`. | SRD 5.2.1, "Elf", "Elven Lineages", p. 85 | per case (T3): happy / unhappy (boundary) |
| FR-024 | The level 3 and level 5 `spellId` of each lineage MUST be the T3 spells. The grant semantics (always prepared, castable once without a spell slot, that free cast regained when a Long Rest is finished, also castable with a slot, and the choice of Intelligence, Wisdom or Charisma as spellcasting ability) MUST be stated in the Elven Lineage trait text and are not structured data (content-species-backgrounds-feats.md Q17). Applying them (always prepared, free cast, chosen ability) is species-trait-behaviour.md FR-004, FR-021 and FR-024. | SRD 5.2.1, "Elf", pp. 84 to 85 | happy |
| FR-025 | A lineage's Level 1 benefit MUST be stored as that option's text, and MUST NOT alter the `elf` entry's base `speed` (30 feet) or Darkvision text (60 feet): reading `elf` with no lineage applied returns the base values. The Drow text states Darkvision 120 feet and the Wood Elf text Speed 35 feet (content-species-backgrounds-feats.md Q18). Applying the overrides (Darkvision 120, Speed 35) is species-trait-behaviour.md FR-015. | SRD 5.2.1, "Elf", pp. 84 to 85 | unhappy (rule interaction: override versus base) |
| FR-026 | The Fey Ancestry text MUST state Advantage on saving throws to avoid or end the Charmed condition. | SRD 5.2.1, "Elf", p. 85 | happy |
| FR-027 | The Keen Senses text MUST state proficiency in one skill chosen from Insight, Perception or Survival. | SRD 5.2.1, "Elf", p. 85 | happy |
| FR-028 | The Trance text MUST state: no need to sleep, magic cannot put the Elf to sleep, and a Long Rest can be finished in 4 hours. | SRD 5.2.1, "Elf", p. 85 | happy |

### 4.5 Species: Human

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-029 | `human` MUST have Creature Type Humanoid and Speed 30 feet; its `sizes` MUST be the set {Medium, Small}, a choice made when the species is selected, and its `sizeText` MUST carry the printed heights (about 4 to 7 feet tall for Medium, about 2 to 4 feet tall for Small). | SRD 5.2.1, "Human", p. 86 | unhappy (boundary: size is a set) |
| FR-030 | `human` MUST have exactly three traits: Resourceful (Heroic Inspiration whenever a Long Rest is finished), Skillful (proficiency in one skill of the player's choice) and Versatile (an Origin feat of the player's choice, Skilled recommended), each in its verbatim text. | SRD 5.2.1, "Human", p. 86 | happy |
| FR-031 | A trait that asks for a pick from a kind absent from the slice (Versatile's Origin feat, Skillful's skill) stays verbatim text with no reference, so it MUST NOT make the slice fail to build or decode. | SRD 5.2.1, "Human", p. 86 | unhappy (missing content: choice target not bundled) |

### 4.6 Choices across the slice

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-032 | Only Draconic Ancestry and Elven Lineage MUST be structured `choices` elements (SX-1), as FR-015 and FR-023 state. Every other selection in the species traits (rows of table T4: breath weapon shape, lineage spellcasting ability, keen senses skill, high elf cantrip replacement, human size, skillful skill, versatile feat) MUST be present in the verbatim trait text, or in `sizes` for the Human size, and MUST NOT be a second structured copy. T4 lists the SRD facts of every selection, so the test author checks each row is stated in the text. Class selections belong to content-classes.md. Applying the selections is species-trait-behaviour.md. | SRD 5.2.1, pp. 84 to 86; content-species-backgrounds-feats.md Q19 | per case (T4): happy / unhappy (boundary) |

### 4.7 Spells

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | Each of the 12 slice spells MUST have exactly the level, School of Magic, class list, Casting Time, Range, Components and Duration of table T6. | SRD 5.2.1, "Spell Descriptions", pp. 121 to 154 | per case (T6): happy / unhappy (boundary) |
| FR-050 | The Ritual tag is the `Spell.ritual` flag beside `castingTimes` (content-spells.md): Detect Magic has a single `CastingTimeMagicAction` and `ritual` true, Fire Bolt has `CastingTimeMagicAction` and `ritual` false, Misty Step has `CastingTimeBonusAction` and `ritual` false. | SRD 5.2.1, "Casting without Slots", p. 104; "Casting Time", p. 105; "Detect Magic", p. 123 | unhappy (rule interaction: action plus Ritual) |
| FR-051 | A Duration MUST distinguish Instantaneous, Concentration up to a span, and a time span: Detect Magic is Concentration up to 10 minutes, Prestidigitation is "Up to 1 hour", Fire Bolt, Fireball, Magic Missile, Misty Step and Druidcraft are Instantaneous, Longstrider is a time span of 1 hour without Concentration, and the other lineage spells follow T6 (Concentration up to 1 minute, 10 minutes, 1 minute and 1 hour for Dancing Lights, Darkness, Faerie Fire and Pass without Trace). | SRD 5.2.1, "Duration", p. 106 | happy |
| FR-052 | Components MUST record Verbal, Somatic and Material separately, and a Material component MUST carry its text: Fireball carries "a ball of bat guano and sulfur"; Misty Step and Faerie Fire have Verbal only; Druidcraft has Verbal and Somatic without Material; the Material text of the lineage spells is as in T6. | SRD 5.2.1, "Components", p. 105; "Fireball", p. 131; "Misty Step", p. 150 | unhappy (boundary: optional Material text) |
| FR-053 | A Range MUST distinguish Self from a distance in feet: Self for Detect Magic and Misty Step; 10, 120, 120 and 150 feet for Prestidigitation, Fire Bolt, Magic Missile and Fireball; the lineage spells as in T6 (Touch for Longstrider, Self for Pass without Trace, feet for the rest). | SRD 5.2.1, "Range", p. 105 | happy |
| FR-054 | Fire Bolt MUST carry: a ranged spell attack; on a hit 1d10 Fire damage; a flammable object hit and not worn or carried starts burning. | SRD 5.2.1, "Fire Bolt", p. 132 | happy |
| FR-055 | Magic Missile MUST carry: three darts, each striking a creature of the caster's choice that can be seen within range, each dealing 1d4 + 1 Force damage, all striking simultaneously. | SRD 5.2.1, "Magic Missile", p. 146 | happy |
| FR-056 | Fireball MUST carry: a 20-foot-radius Sphere centred on a point within range, a Dexterity saving throw, 8d6 Fire damage on a failed save and half as much on a success, and flammable objects in the area not worn or carried start burning. | SRD 5.2.1, "Fireball", p. 131 | happy |
| FR-057 | Misty Step MUST carry: teleport up to 30 feet to an unoccupied space the caster can see, with no attack roll, saving throw or damage. | SRD 5.2.1, "Misty Step", p. 150 | happy |
| FR-058 | Detect Magic MUST carry: for the duration, sense magical effects within 30 feet; the Magic action reveals a faint aura and the school of a spell-created effect; blocked by 1 foot of stone, dirt or wood, 1 inch of metal or a thin sheet of lead. | SRD 5.2.1, "Detect Magic", p. 123 | happy |
| FR-059 | Prestidigitation MUST carry six named effects (Sensory Effect, Fire Play, Clean or Soil, Minor Sensation, Magic Mark, Minor Creation) and the limit of three non-instantaneous effects active at a time. | SRD 5.2.1, "Prestidigitation", p. 154 | happy |
| FR-060 | Spell scaling MUST be recorded per table T7 as text only (content-spells.md FR-115): Fire Bolt has `cantripUpgrade` (thresholds 5, 11, 17 are character levels by SRD 5.2.1, "Multiclassing", p. 25) and no `higherLevel`; Magic Missile (one more dart per level above 1), Fireball (1d6 more per level above 3) and Longstrider (one additional creature per level above 1) have `higherLevel` and no `cantripUpgrade`; Detect Magic, Misty Step, Prestidigitation, Dancing Lights, Darkness, Druidcraft, Faerie Fire and Pass without Trace have neither field (absent, not an empty list, which content-spells.md rejects). The data holds no structured mechanism; the Mechanism column of T7 is for the test author. | SRD 5.2.1, "Fire Bolt", p. 132; "Multiclassing" (Spellcasting, Cantrips), p. 25; "Magic Missile", p. 146; "Fireball", p. 131 | per case (T7): happy / unhappy (boundary) |
| FR-061 | For every slice spell the class list MUST equal the set of T6 (membership and size 1, 2, 3, 4 and 8, each non-empty); order is not asserted. `Spell.classes` is an immutable set of `ClassID` ignoring listing order (a defensively copied slice, compared as a set) (content-spells.md FR-017). The class names are `ClassID` values (shared-types-and-ownership.md FR-005), not references. The non-empty rule is content-spells.md FR-018; this pins only the five sizes. | SRD 5.2.1, "Class Spell Lists", p. 105 | unhappy (boundary: lists of 1, 2, 3, 4 and 8 classes) |
| FR-083 | The six lineage spells MUST carry these facts in their text: Dancing Lights, up to four torch-size lights shedding Dim Light in a 10-foot radius, combinable into one Medium humanlike form, with a Bonus Action to move them up to 60 feet; Darkness, magical Darkness filling a 15-foot-radius Sphere that Darkvision cannot see through, or a 15-foot Emanation on an object, dispelling an overlapping Bright Light or Dim Light of a spell of level 2 or lower; Druidcraft, four named effects (Weather Sensor, Bloom, Sensory Effect, Fire Play); Faerie Fire, a 20-foot Cube, outlined in blue, green or violet light, a Dexterity saving throw, and Advantage on attack rolls against an affected creature or object the attacker can see; Longstrider, a touched creature's Speed increases by 10 feet; Pass without Trace, a 30-foot Emanation, a +10 bonus to Dexterity (Stealth) checks and no tracks left. | SRD 5.2.1, "Dancing Lights", p. 121; "Darkness", p. 122; "Druidcraft", p. 126; "Faerie Fire", p. 129; "Longstrider", p. 145; "Pass without Trace", p. 151 | happy |

### 4.8 References and missing content

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-062 | A reference MUST be a `SpellID` (whose `key` is a `Key` of kind `spell`) in a structured field (the `spellId` of a lineage's `levelBenefits`). A name inside description text, and a class name in a spell's class list (a `ClassID` value), MUST NOT be a reference. | architecture.md §5 (anti-corruption); content-spells.md; shared-types-and-ownership.md FR-005 | unhappy (invalid input: prose is not data) |
| FR-063 | **The slice is closed under its references** (shared-types-and-ownership.md FR-054). Every lineage `spellId` MUST name an entry the slice contains: `dancing_lights`, `prestidigitation`, `druidcraft` (Level 1), `faerie_fire`, `detect_magic`, `longstrider` (Level 3) and `darkness`, `misty_step`, `pass_without_trace` (Level 5), exactly the nine of T3. The lineage `spellId` is a registered reference site (content-schema-and-decoder.md FR-126; row R1-species-backgrounds-feats) with no scope and no opt-out, so `content.ValidateSet` over the three merged documents (what `data.BundledSource` runs, content-schema-and-decoder.md FR-125 to FR-127, FR-140) MUST return `ValidateSet result` with no `unresolvedReference` problem. Both the set of collected `spellId`s and the set of bundled spell ids it must be a subset of are asserted. The slice defines no absent id and no pin depends on a reference being unresolved. | SRD 5.2.1, "Elven Lineages", p. 85; shared-types-and-ownership.md FR-054; content-schema-and-decoder.md FR-125 to FR-127 | unhappy (dangling id: none permitted) |
| FR-064 | Looking up a spell that is in SRD 5.2.1 but not in the slice, for example `Key(spell, light)` or `Key(spell, ray_of_frost)` (both named in the Wizard's Spellcasting text, FR-062), MUST return `LookupNotFound` carrying that key and a nil error, and MUST NOT panic. This is a lookup result, not a reference problem. | architecture.md §4, §6; content-source-port.md FR-009 | unhappy (missing content) |
| FR-065 | Looking up a species that is in SRD 5.2.1 but not in the slice, `Key(species, gnome)`, MUST complete with `LookupNotFound`. The bundle MUST NOT supply it from any other source. | SRD 5.2.1, "Gnome", p. 85 | unhappy (missing content: real SRD entry not bundled) |
| FR-066 | Looking up an id that exists only under another kind, `Key(species, fireball)` and `Key(spell, wizard)`, MUST complete with `LookupNotFound`. | content-source-port.md FR-010 | unhappy (rule interaction: kind mismatch) |

### 4.9 Loading and exposure

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-067 | Each of the three embedded documents ("Loading unit") MUST decode through `content.Decoder` with zero problems, producing 4, 1 and 12 entries; a `data.BundledSource` built from exactly those texts (content-source-port.md FR-059) MUST hold exactly the 17 entries of FR-001 to FR-003. The duplicate-id check is per document (content-schema-and-decoder.md FR-031); a merged set holds no duplicate key (FR-006). Whole-set validation over the merged documents (content-schema-and-decoder.md FR-125 to FR-127) MUST pass because the slice is closed (FR-063). The check runs against the embedded strings only, through the same decode path as host JSON; comparing the files under `data/srd_5_2_1/` to the embedded strings is the architecture test of content-schema-and-decoder.md FR-104 (rule FR-089) and the drift check of architecture-guardrails.md FR-078, not a data pin here. | architecture.md §5; shared-types-and-ownership.md FR-031; content-schema-and-decoder.md FR-077, FR-089 | happy |
| FR-078 | Loading the three embedded slice documents through `LoadSet` (three calls, one per document) with a recording `telemetry.Telemetry` MUST add to `srd5e.content.entries_loaded`, across the calls, 4 for `srd5e.content.kind=species`, 1 for `srd5e.content.kind=character_class` and 12 for `srd5e.content.kind=spell` (nothing for other kinds), MUST add nothing to `srd5e.content.decode_errors`, and MUST add three `srd5e.content.loads` with `srd5e.outcome=success`, one per call (kind value: snake_case of the kind name, content-schema-and-decoder.md FR-101; outcome values: shared-types-and-ownership.md FR-041). The per-kind counts lapse as section 1 says. The use case returns its own closed `LoadSetResult` (content-schema-and-decoder.md FR-077; shared-types-and-ownership.md FR-017). Each call is a separate top-level call by the test; no use case calls another (shared-types-and-ownership.md FR-044). | content-schema-and-decoder.md FR-079 to FR-082; observability.md §4; shared-types-and-ownership.md FR-017, FR-041, FR-044 | happy |
| FR-079 | Verbatim fidelity of the long texts (trait, feature and spell descriptions) to the cited SRD page MUST be reviewed by a person against the PDF before merge, with the result recorded in the PR checklist. This is a process requirement; CI does not claim to prove it (the PDF is not in the repo). | AGENTS.md, "Licensing and source material" | n/a (process) |
| FR-084 | Self-consistency under whole-set validation is checked from the decoded data, not a hand-kept list: the test collects every registered reference site (content-schema-and-decoder.md reference-site table: the `species` lineage `spellId` and `spellIds` sites and the `characterClass` `AlwaysPreparedSpells` sites) from the three decoded documents and MUST find each target in the merged set. The slice has no `AlwaysPreparedSpells` fact, no `background` and no `equipment`, so the sites found are exactly the nine lineage `spellId`s. A future entry adding a site whose target is outside the slice fails this pin. | shared-types-and-ownership.md FR-054; content-schema-and-decoder.md FR-126, FR-140 | unhappy (dangling id: a new site with an absent target) |

### 4.10 Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-033 | Withdrawn: moved to content-classes.md (FR-016; Hit Point Die FR-017) |
| FR-034 | Withdrawn: moved to content-classes.md (FR-018) |
| FR-035 | Withdrawn: moved to content-classes.md (FR-020) |
| FR-036 | Withdrawn: moved to content-classes.md (FR-022; weapon proficiencies FR-021) |
| FR-037 | Withdrawn: moved to content-classes.md (FR-024; item names FR-025) |
| FR-038 | Withdrawn: moved to content-classes.md (FR-033); table T5 moved with it |
| FR-039 | Withdrawn: moved to content-classes.md (FR-031) |
| FR-040 | Withdrawn: moved to content-classes.md (FR-033; monotonic columns FR-034) |
| FR-041 | Withdrawn: moved to content-classes.md (FR-046); the ten feature entries are referenced by table rows, no name-only features |
| FR-042 | Withdrawn: moved to content-classes.md (FR-029) |
| FR-043 | Withdrawn: moved to content-classes.md (FR-054) |
| FR-044 | Withdrawn: moved to content-classes.md (FR-054) |
| FR-045 | Withdrawn: moved to content-classes.md (FR-054) |
| FR-046 | Withdrawn: moved to content-classes.md (FR-037) |
| FR-047 | Withdrawn: moved to content-spells.md (FR-120) |
| FR-049 | Withdrawn: moved to content-spells.md (FR-009) |
| FR-068 | Withdrawn: moved to content-source-port.md (FR-059; no file access FR-060) |
| FR-069 | Withdrawn: moved to content-source-port.md (FR-062) |
| FR-070 | Withdrawn: moved to content-species-backgrounds-feats.md (FR-056); spell and class collections: content-spells.md FR-017, FR-058; content-classes.md FR-008 |
| FR-071 | Withdrawn: moved to content-source-port.md (FR-047) |
| FR-072 | Withdrawn: moved to content-schema-and-decoder.md (FR-016) |
| FR-073 | Withdrawn: moved to content-schema-and-decoder.md (FR-120); covered paths: content-licensing-and-notice.md FR-005, FR-008 |
| FR-074 | Withdrawn: moved to architecture-guardrails.md (FR-026; `content/data/` FR-025) |
| FR-075 | Withdrawn: moved to content-source-port.md (FR-061) |
| FR-076 | Withdrawn: moved to content-classes.md (FR-019); the species `choices` element has no picks count (content-species-backgrounds-feats.md Q19) |
| FR-077 | Withdrawn: moved to content-species-backgrounds-feats.md (FR-012) |
| FR-080 | Withdrawn: moved to architecture-guardrails.md (FR-078); generator: content-schema-and-decoder.md FR-121 to FR-124 |
| FR-081 | Withdrawn: moved to content-licensing-and-notice.md (FR-033) |
| FR-082 | Withdrawn: moved to content-schema-and-decoder.md (FR-106); the slice data uses `DamageType`, `Skill`, `CreatureSize`, `CreatureType`, `ClassID`, `SpeciesID`, `SpellID`, `FeatureID`, and no `RechargeTrigger` or `FeatID` |

### Tables of cases

Each row becomes one table-driven test. "Path" is `happy` unless stated.

**T1. Draconic Ancestors (FR-015), SRD 5.2.1, "Draconic Ancestors", p. 84**

| Label | Dragon | Damage type | Path |
|---|---|---|---|
| black | Black | Acid | happy |
| blue | Blue | Lightning | happy |
| brass | Brass | Fire | happy |
| bronze | Bronze | Lightning | happy |
| copper | Copper | Acid | happy |
| gold | Gold | Fire | happy |
| green | Green | Poison | happy |
| red | Red | Fire | happy |
| silver | Silver | Cold | happy |
| white | White | Cold | happy |
| exactly ten ancestors | (count of rows) | 10 | unhappy (boundary: none missing, none extra) |

**T2. Breath Weapon damage by character level (FR-017), SRD 5.2.1, "Dragonborn", p. 84**

| Label | Threshold level | Damage dice | Path |
|---|---|---|---|
| base | 1 | 1d10 | happy |
| first upgrade | 5 | 2d10 | unhappy (boundary: lowest threshold) |
| second upgrade | 11 | 3d10 | unhappy (boundary) |
| third upgrade | 17 | 4d10 | unhappy (boundary: highest threshold) |
| exactly three upgrades | (count of thresholds above level 1) | 3 | unhappy (boundary: none missing, none extra) |

**T3. Elven Lineages (FR-023, FR-063), SRD 5.2.1, "Elven Lineages", p. 85.** All nine spells are bundled (T6).

| Label | Lineage | Level 1 benefit | Level 3 spell | Level 5 spell | Path |
|---|---|---|---|---|---|
| drow | Drow | Darkvision range increases to 120 feet; knows the Dancing Lights cantrip | Faerie Fire | Darkness | happy |
| high elf | High Elf | Knows the Prestidigitation cantrip; after a Long Rest can replace it with a different cantrip from the Wizard spell list | Detect Magic | Misty Step | happy |
| wood elf | Wood Elf | Speed increases to 35 feet; knows the Druidcraft cantrip | Longstrider | Pass without Trace | happy |
| exactly three lineages | (count) | 3 | | | unhappy (boundary) |

**T4. Choices in the species (FR-032), SRD 5.2.1, pp. 84 to 86.** "Picks" and "Option set" are SRD facts for the test author; "Held as" says where the data holds the selection (section 4, "Text pins").

| Label | Owner | Option set | Picks | Held as | Source | Path |
|---|---|---|---|---|---|---|
| draconic ancestry | dragonborn, Draconic Ancestry | the ten dragon kinds of T1 | 1 | `choices` (10 `DamageTypeOption`s) | p. 84 | happy |
| breath weapon shape | dragonborn, Breath Weapon | 15-foot Cone; 30-foot Line, 5 feet wide | 1, each use | trait text | p. 84 | happy |
| elven lineage | elf, Elven Lineage | Drow, High Elf, Wood Elf | 1 | `choices` (3 `LineageOption`s) | p. 85 | happy |
| lineage spellcasting ability | elf, Elven Lineage | Intelligence, Wisdom, Charisma | 1, when the lineage is selected | trait text | p. 85; applied by species-trait-behaviour.md FR-004 | happy |
| keen senses skill | elf, Keen Senses | Insight, Perception, Survival | 1 | trait text | p. 85 | happy |
| high elf cantrip replacement | elf, High Elf | any cantrip on the Wizard spell list | 1, after each Long Rest | option text | p. 85 | unhappy (missing content: option set is a derived list, mostly not bundled) |
| human size | human, Size | Medium, Small | 1, when the species is selected | `sizes` | p. 86 | unhappy (boundary: size is a choice) |
| skillful skill | human, Skillful | any skill | 1 | trait text | p. 86 | unhappy (missing content: skills are not a bundled kind) |
| versatile feat | human, Versatile | any Origin feat | 1 | trait text | p. 86 | unhappy (missing content: no feats bundled) |

**T6. Spell headers (FR-048, FR-061), SRD 5.2.1, "Spell Descriptions".** (T5 was the Wizard Features table, moved to content-classes.md with FR-038.)

| Label | Id | Level and School | Class list (a set; order not significant) | Casting Time | Range | Components | Duration | Page | Path |
|---|---|---|---|---|---|---|---|---|---|
| fire bolt | fire_bolt | 0, Evocation | Sorcerer, Wizard | Action | 120 feet | V, S | Instantaneous | 132 | happy |
| prestidigitation | prestidigitation | 0, Transmutation | Bard, Sorcerer, Warlock, Wizard | Action | 10 feet | V, S | Up to 1 hour | 154 | unhappy (boundary: lowest level, time-span duration) |
| detect magic | detect_magic | 1, Divination | Bard, Cleric, Druid, Paladin, Ranger, Sorcerer, Warlock, Wizard | Action or Ritual | Self | V, S | Concentration, up to 10 minutes | 123 | unhappy (boundary: longest class list; ritual) |
| magic missile | magic_missile | 1, Evocation | Sorcerer, Wizard | Action | 120 feet | V, S | Instantaneous | 146 | unhappy (boundary: two-class list) |
| misty step | misty_step | 2, Conjuration | Sorcerer, Warlock, Wizard | Bonus Action | Self | V | Instantaneous | 150 | happy |
| fireball | fireball | 3, Evocation | Sorcerer, Wizard | Action | 150 feet | V, S, M (a ball of bat guano and sulfur) | Instantaneous | 131 | happy |
| dancing lights | dancing_lights | 0, Illusion | Bard, Sorcerer, Wizard | Action | 120 feet | V, S, M (a bit of phosphorus) | Concentration, up to 1 minute | 121 | happy |
| darkness | darkness | 2, Evocation | Sorcerer, Warlock, Wizard | Action | 60 feet | V, M (bat fur and a piece of coal) | Concentration, up to 10 minutes | 122 | unhappy (boundary: Material without Somatic) |
| druidcraft | druidcraft | 0, Transmutation | Druid | Action | 30 feet | V, S | Instantaneous | 126 | unhappy (boundary: one-class list; not on the Wizard list) |
| faerie fire | faerie_fire | 1, Evocation | Bard, Druid | Action | 60 feet | V | Concentration, up to 1 minute | 129 | happy |
| longstrider | longstrider | 1, Transmutation | Bard, Druid, Ranger, Wizard | Action | Touch | V, S, M (a pinch of dirt) | 1 hour | 145 | unhappy (boundary: Touch range; time span without Concentration) |
| pass without trace | pass_without_trace | 2, Abjuration | Druid, Ranger | Action | Self | V, S, M (ashes from burned mistletoe) | Concentration, up to 1 hour | 151 | happy |

**T7. Spell scaling (FR-060).** The Mechanism column is the SRD's rule, for the test author; the data holds the text form only (`cantripUpgrade`, `higherLevel`, or neither).

| Label | Spell | Mechanism | Rule | Source | Path |
|---|---|---|---|---|---|
| fire bolt upgrade | fire_bolt | character level (SRD 5.2.1, "Multiclassing", p. 25) | 1d10; 2d10 at level 5; 3d10 at 11; 4d10 at 17, as printed "levels 5 (2d10), 11 (3d10), and 17 (4d10)" | pp. 25, 132 | happy |
| fire bolt thresholds match breath weapon | fire_bolt, dragonborn | character level | the thresholds 5, 11 and 17 equal those of T2; both are character levels (Dragonborn wording p. 84; cantrip rule p. 25) | pp. 25, 84, 132 | unhappy (rule interaction: one scaling shape for two kinds) |
| magic missile slot | magic_missile | spell slot level | one more dart for each spell slot level above 1 | p. 146 | happy |
| fireball slot | fireball | spell slot level | damage increases by 1d6 for each spell slot level above 3 | p. 131 | happy |
| detect magic none | detect_magic | none | no scaling | p. 123 | unhappy (empty case) |
| misty step none | misty_step | none | no scaling | p. 150 | unhappy (empty case) |
| prestidigitation none | prestidigitation | none | no scaling | p. 154 | unhappy (empty case) |
| longstrider slot | longstrider | spell slot level | one additional creature for each spell slot level above 1 | p. 145 | happy |
| lineage spells none | dancing_lights, darkness, druidcraft, faerie_fire, pass_without_trace | none | no scaling | pp. 121, 122, 126, 129, 151 | unhappy (empty case) |

## 5. Model

This spec defines no type. Owners (shared-types-and-ownership.md FR-005, FR-025,
FR-035, FR-046, FR-047): species types and the SX-1 `choices` element,
content-species-backgrounds-feats.md; `Spell` and its parts, content-spells.md;
`ClassDefinition`, `ProgressionRow`, `FeatureEntry`, `ClassSpellList`,
content-classes.md; the envelope, decoder, `content.Problem`, revision table,
`NamedText`, `SourceCitation`, vocabulary types and typed ids,
content-schema-and-decoder.md (FR-106 to FR-114); `Entry`, `Key`,
`content.Source`, `LookupResult`, `content.InMemorySource`,
`data.BundledSource`, `data.BundledInvalidError`, `MissingContent`,
content-source-port.md; `ability.Ability` and `ability.Score` (the kernel). `ItemID` is
unused: starting-equipment items are names as printed (content-classes.md
FR-025). `CreatureSize` is the type for the SRD field "Size".

Invariants live in the owning constructors (each returns an error matching `srderr.ErrInvalidInput`;
shared-types-and-ownership.md FR-048): spell level 0 to 9 (content-spells.md
FR-009); nine slot counts per row, levels 1 to 20 once each (content-classes.md
FR-031, FR-037); one or two species sizes (content-species-backgrounds-feats.md
FR-012). A non-empty class list is a data pin (FR-061). Whether a reference
resolves is never a constructor check but a whole-set `unresolvedReference`
problem (content-schema-and-decoder.md FR-125); the slice has none (FR-063).

### 5.1 Requests answered by the owners

All four requests of the first draft are answered; none is open. One
`schemaVersion` covers the content format (shared-types-and-ownership.md
FR-034); this spec claims no revision-table row, and `R1-thin-slice` is to be
closed as declined by its owner.

| Id | Request | Answer | Effect |
|---|---|---|---|
| U1 | Typed trait facts on a species trait | Declined: a trait is `NamedText` (content-species-backgrounds-feats.md Q15, Q16) | FR-010 to FR-013, FR-016 to FR-020, FR-022, FR-026 to FR-028, FR-030 are text pins |
| U2 | Choice picks and option-set kinds | Accepted for the five species choices only, as SX-1; no picks count (content-species-backgrounds-feats.md Q19) | FR-015, FR-023 are structure; FR-032 and T4 say what is text |
| U3 | Lineage grant semantics, typed overrides | Declined; typed `spellId` per level accepted (content-species-backgrounds-feats.md Q17, Q18) | FR-024, FR-025 |
| U4 | A structured `SpellScaling` | Declined: scaling is text (content-spells.md FR-115) | FR-060, T7 |

## 6. Interface

No new use case, port, adapter or type: the slice is three JSON documents and
their embedded JSON documents, served by `data.BundledSource`
(content-source-port.md FR-059 to FR-064) and decoded by `content.Decoder`.
`Lookup` and `ListKind` take a `context.Context` and return an `error` (architecture.md §4). The package reads
no file at runtime: the JSON documents under `content/data/` are embedded
with `go:embed` and decoded at construction
(shared-types-and-ownership.md FR-030, FR-032; content-schema-and-decoder.md FR-120 to
FR-124). Data tests use the embedded documents only, with no real filesystem
(testing.md §6); the disk read is the architecture test of
content-schema-and-decoder.md FR-104 and the embedded-data check of
architecture-guardrails.md FR-078. Expected absence is `LookupNotFound`
(FR-064 to FR-066); the in-memory source never returns a `*content.SourceError`;
a document that fails to decode is handled by `data.BundledSource`
(content-source-port.md FR-061). Exports: none of its own (types by the section 5
owners, `data.BundledSource` from `content/data`, doubles from
`content/contenttest`; the embedded documents are never exported).

## 7. Observability

The data is static content and silent by design (observability.md §3). This
spec adds no signal and no catalogue entry, and nothing feeds
`doc/observability/signals.md`. The signals that apply are those of
`LoadSet` (content-schema-and-decoder.md FR-079 to FR-082:
`srd5e.content.load` and `decode` spans, `loads`, `decode_errors`,
`entries_loaded`, `load_duration`); they fire only through `LoadSet`, not
on direct `data.BundledSource` use. Attribute values follow
shared-types-and-ownership.md FR-041 and FR-051. Answers to the seven questions:

1. **What can go wrong?** A transcription error; a decode failure (packaging
   defect); a duplicate id; a missing citation; a dangling reference (FR-063).
   A lookup outside the slice is expected, not a failure.
2. **How would an operator know?** Transcription errors are caught by the pin
   tests. A decode failure shows as `loads` with `srd5e.outcome=failure`,
   non-zero `decode_errors` and an error status on the `srd5e.content.load`
   span; `LoadSetRejected` carries `srd5e.content.problem_kind`
   (content-schema-and-decoder.md FR-100) but no `error.type`. A
   `LookupNotFound` lookup has **no signal**: no lookup use case exists and a
   use case would not call another (shared-types-and-ownership.md FR-044).
3. **Diagnosis?** The failure result lists path and kind per problem
   (content-schema-and-decoder.md FR-016; content-source-port.md FR-061); each
   entry carries its SRD section and page (FR-007).
4. **Counted or timed?** `entries_loaded` per kind (FR-078: species 4, character_class 1, spell 12; no `decode_errors`).
5. **Existing instrumentation still correct?** Yes: `LoadSet` signals are unchanged; FR-078 only asserts values.
6. **Consumer monitoring?** For a slice-only consumer: `decode_errors` above 0,
   `loads` with `outcome=failure`, `entries_loaded` differing from FR-078 (the
   counts lapse by kind as section 1 says). No `LookupNotFound`-rate advice.
7. **Noisy?** No. A `LookupNotFound` lookup MUST NOT be logged at warning level
   (observability.md §7).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none added by this spec | n/a | n/a | n/a | Use the content-schema-and-decoder.md signals; alert on `decode_errors` above zero and compare `entries_loaded` to FR-078. |

## 8. Test plan

Levels: adapter (real decoder over the embedded strings, via `data.BundledSource`
built from the three texts or `content.Decoder`, in `content/data/bundled_source_test.go`),
application (`LoadSet` with a recording `telemetry.Telemetry`, in
`content/load_set_test.go`), scenario (`scenarios/`), architecture. Unit,
contract, embedded-data decode and notice checks belong to the owners (section
4.10). Data pins are table-driven from section 4, written from the PDF
independently of the data files, cite this spec, the FR id and the SRD section
and page, never read the filesystem (testing.md §6), and assert phrases the test
author picks from the cited page. Fixtures: "Oracle", section 1. Doubles: a
recording `telemetry.Telemetry` for FR-078; a fake `content.Decoder`
(shared-types-and-ownership.md FR-038) for a scripted decoder.

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-003 | adapter | `ListKind(species)` and `ListKind(spell)` ids equal the four and twelve ids |
| FR-002 | adapter | `ListKind(characterClass)` is `wizard` only; `evoker` and `fighter` keys are `LookupNotFound` |
| FR-004 | adapter | `ListKind` empty for every other `Kind` (iterate the enum, so later kinds are covered) |
| FR-005, FR-006 | adapter | `Lookup` of each of the 17 keys is `LookupFound` with an equal key; constructing the source from the three texts returns no `*data.BundledInvalidError` and no `*content.DuplicateKeyError` |
| FR-007 | adapter | each entry's citation equals the section 2 row |
| FR-008 | adapter | pinned phrases per entry; word-boundary scan for banned terms (self-test: "trace" and "grace" do not match). Fidelity is FR-079, not a CI claim |
| FR-009 to FR-013 | adapter | Dwarf core fields; trait names; text phrases (the Stonecunning stone-surface condition: a copy without it fails) |
| FR-014, FR-016 to FR-020 | adapter | Dragonborn core, five trait names, Darkvision 60 feet, Breath Weapon, Damage Resistance (no fixed type), Draconic Flight phrases (level gate 5) |
| FR-015, FR-017 | adapter | one test per T1 and T2 row, including count rows; `choices` links to the trait by `traitName` |
| FR-021, FR-022, FR-026 to FR-028 | adapter | Elf core, five trait names, Darkvision 60 feet, Fey Ancestry, Keen Senses, Trance phrases |
| FR-023, FR-024 | adapter | one test per T3 row including count; `levelBenefits` levels 1, 3, 5 with `spellId`; grant-semantics phrases in the trait text |
| FR-025 | adapter | base `elf` unchanged after reading lineages; Drow 120 and Wood Elf 35 in the option text |
| FR-029 to FR-031 | adapter | `sizes` is {Medium, Small}, `sizeText` both heights, Speed 30; three trait names and phrases; slice builds and decodes with feat and skill picks as text only (also FR-067) |
| FR-032 | adapter | one test per T4 row: the two `choices` rows are structure, every other row's phrase is in the text or `sizes`; no second structured copy |
| FR-048, FR-051 to FR-053 | adapter | one test per T6 row; duration forms; Material text for Fireball only, Misty Step V only; Self versus feet |
| FR-050 | adapter | Ritual tag only on Detect Magic |
| FR-054 to FR-059, FR-083 | adapter | effect phrases per spell, including the six effects and limit of three (FR-059) and one test per lineage spell (FR-083) |
| FR-060 | adapter | one test per T7 row, asserting `cantripUpgrade`, `higherLevel` or neither |
| FR-061 | adapter | class sets of size 1, 2, 3, 4, 8 with T6 membership, none empty, compared as sets (the empty-set case is content-spells.md's test) |
| FR-062 | adapter | the Wizard Spellcasting text names Light, Mage Hand and Ray of Frost; none appears as a reference |
| FR-063, FR-084 | adapter | collect every registered reference site from the decoded entries (nine `spellId`s); each is a bundled spell key and equals the T3 spells; `content.ValidateSet` over the merged documents reports no problem; a synthetic entry with a dangling site fails it |
| FR-064 to FR-066 | adapter | `LookupNotFound` carrying the key for `light`, `ray_of_frost`, `gnome`, and the two kind-mismatched keys |
| FR-067 | adapter | decode each embedded string: zero problems, 4, 1 and 12 entries; the source holds 17; whole-set validation passes; no disk read |
| FR-078 | application | `LoadSet` once per document with a recording `telemetry.Telemetry` and the three real texts; assert across the calls the `entries_loaded` counts, no `decode_errors`, three `loads` with `success` |
| FR-079 | doc review | PR checklist item; no automated test |

Also in scope: one scenario test through the public packages and
`content/data` only: build a `data.BundledSource` from the
three texts, look up `elf`, follow every lineage spell of all three lineages
(nine) to `LookupFound`, and look up `light` for `LookupNotFound`. It cites
FR-063 and FR-064.

Mutation testing (testing.md §8): no domain code or adapter, so nothing is
mutated; the pin tables protect the data.

Deliberately not covered: verbatim fidelity beyond pinned phrases (FR-079);
infrastructure failure of the bundled source (no I/O; content-source-port.md
tests the contract and FR-061); a `LookupNotFound` signal (none exists,
section 7).

**Implementation status.**

- Done (Phase 2, step 6, package `content/data`): the three documents `species.json` (4 species), `class_wizard.json` (the Wizard, no subclass) and `spells.json` (12 spells), transcribed from SRD 5.2.1 at the pages of section 2; `bundled_source.go` with `BundledSource`, `BundledInvalidError`, the options and `Files()` (content-source-port.md FR-059 to FR-064). The pins are `data_pin_test.go` (FR-001 to FR-007, FR-008, FR-062 to FR-067, FR-078, FR-084, the file-list pin), `data_species_test.go` (FR-009 to FR-032, T1 to T4), `data_spells_test.go` (FR-048 to FR-061, FR-083, T6, T7) and `data_wizard_test.go` (the Wizard shape, whose value pins are content-classes.md's), driven by the oracle under `content/data/testdata/srd/thin_slice/` (README with the attribution pointer, content-licensing-and-notice.md FR-027). The oracle phrases and tables were checked against the PDF text; the data strings were checked as substrings of the PDF text with line-end hyphens resolved (FR-066 of content-spells.md rule: joined unless a genuine hyphenated form, as in "10-foot").
- Choices the spec left open: the oracle sits in `content/data/testdata/srd/thin_slice/` and is embedded into the tests (`go:embed` cannot reach a `testdata/` at the module root, and tests read no real filesystem); species `sizes` are `[Small, Medium]` for the Human (the order of content-species-backgrounds-feats.md FR-012; compared as a set); trait text is the text after the run-in heading with paragraphs joined by one line feed, so Breath Weapon, Stonecunning and Elven Lineage hold two or three paragraphs; the Elven Lineages table cells are stored as printed (the zero-width space of the PDF is dropped); starting equipment names keep the printed typographic apostrophe ("Scholar’s Pack").
- Not yet implemented: FR-079 (the person's check of every long text against the PDF, recorded in the PR checklist; a process requirement, not claimed by CI).

## 9. Open questions and decisions

No open questions. Settled by: data location, form and decode check,
shared-types-and-ownership.md FR-030 to FR-033 and architecture-guardrails.md
FR-078; id syntax, FR-007 there and content-schema-and-decoder.md FR-030; "—"
as 0 slots, content-classes.md FR-031; reference closure, shared-types-and-ownership.md
FR-054 and content-schema-and-decoder.md FR-126, FR-140; vocabulary types not
content kinds, FR-005, FR-006, FR-046 there; equipment as names and the Wizard
entry, content-classes.md FR-024, FR-025 and shared-types-and-ownership.md
FR-025; class lists as a set of `ClassID`, content-spells.md FR-016, FR-017;
`data.BundledSource` ownership, shared-types-and-ownership.md FR-047 (other data
specs do not list this spec in Depends; row `R1-thin-slice` closes as declined,
section 5.1). The Wizard's recommended spells stay text (FR-062); revisit if a
`character` feature needs them.

## 10. Changelog impact

Under `Unreleased`:

- **Added:** a bundled SRD 5.2.1 content slice (species Dragonborn, Dwarf, Elf
  and Human; the Wizard class; twelve spells including the six Elf lineage
  spells), served by `data.BundledSource` from `content/data`
  with the content types to read it. `NOTICE.md` covers `data/`,
  `content/data/`.

Not **BREAKING**: everything is new. Later corrections of a slice value against
the SRD are **Fixed**; adding, removing or renaming a slice entry id or a
structured field after release is **BREAKING** (an exported content key
changes).
