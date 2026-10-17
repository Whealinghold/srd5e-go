# Content: class and subclass bundled data and oracle

| | |
|---|---|
| Status | draft |
| Context | `content` |
| Roadmap phase | 3 |
| Depends on | `content-classes.md`, `content-schema-and-decoder.md`, `content-source-port.md`, `content-spells.md` |

## 1. Purpose and scope

The verification and data half of `content-classes.md`, split off by shared-types-and-ownership.md
FR-062. It owns the bundled class files (twelve classes, twelve subclasses), the extraction and
verification process, and the hand-transcribed **oracle tables** (SRC to TB) that the "bundled"
requirements of `content-classes.md` cite by table name. It is a bundle spec (rank 7): it also holds the
checks of the class files against the bundled spells (FR-100, FR-101; shared-types-and-ownership.md
FR-067). The content model, decoder rules and accessors stay in `content-classes.md`.

**Out of scope:** any type, JSON shape or decoder rule (`content-classes.md`); class behaviour
(`class-*.md`); anything not in SRD 5.2.1.

## 2. Source

SRD 5.2.1, "Classes", pp. 28-82 (printed page = PDF page); the page map is Table SRC below and
section 2 of `content-classes.md`. Rules derive from shared-types-and-ownership.md (FR-030,
FR-032, FR-033, FR-047), `doc/rules/testing.md` (§3, §6, §8) and `doc/rules/architecture.md` (§5).

## 3. Vocabulary

| Term | Meaning |
|---|---|
| oracle | The hand-transcribed tables below and the fixture mirroring them, independent of the bundled JSON. |
| drift check | The local re-run of the extraction tool against the PDF (FR-092). |
| correction log | The list of hand corrections and accepted findings the tool applies (FR-091). |

Unprefixed `FR-001` to `FR-082` and `FR-095` to `FR-099` inside the oracle tables and notes mean
requirements of `content-classes.md`; `FR-083` to `FR-094` and `FR-100` onward are this spec's.

## 4. Requirements

Paths are JSON Pointers as in `content-classes.md` section 4. Pages are printed pages. Ids are allocated in blocks; an id absent from this section and not in the Withdrawn ids was never used by this spec.

### 4.1 Bundled data and verification

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-083 | Bundled class data MUST be twelve JSON files with **flat stems**, `content/data/class_<class_id>.json` (for example `class_barbarian.json`), each a complete document (`schemaVersion` and a `classes` array of exactly one entry), embedded directly with `go:embed` as content-schema-and-decoder.md FR-120 requires (flat `<stem>.json` names only; no subdirectory, no generated Go). The thin slice's single `classes.json` (content-thin-slice.md) is **replaced** by `class_wizard.json` in the change that moves Wizard to its per-class file, which also removes `classes.json`. | shared-types-and-ownership.md FR-030, FR-032; content-spells.md FR-090 | happy |
| FR-084 | The assembly step is owned by the construction of `data.BundledSource` in package `content/data` (content-source-port.md FR-059; shared-types-and-ownership.md FR-047). It MUST reject a `ClassID`, `SubclassID` or `FeatureID` that appears in two decoded documents by returning a construction error wrapping `srderr.ErrInvalidInput` whose typed cause names the id space and the id (fields, not message text): no `Problem` is produced, because a single decode never sees the other files and a duplicate is a packaging defect, not untrusted input. A duplicate `ClassID` is also a duplicate `Key` and is caught by the content catalogue; `SubclassID` and `FeatureID` are not `Key`s, so the construction MUST check them itself. The same string in two id spaces is accepted (as content-classes.md FR-004). The matching requirement is requested of content-source-port.md (content-classes.md section 9, Q6). | content-source-port.md FR-059; FR-004 | unhappy (invalid input) |
| FR-085 | Each of the twelve files MUST decode to a nil error with no problem, the bundled-data check (schema FR-089, FR-090, FR-104) reads them, and every bundled `source.page` MUST be the first page of its section named in table SRC (content-classes.md FR-009 states the exact rule; there is no range). | schema FR-089, FR-090, FR-104; SRD 5.2.1, pp. 28-82 | happy |
| FR-086 | The extraction tool `tools/extract_classes` is a drafting aid and MUST live in `tools/` (its own module, schema FR-124), never be imported by a package of the root module, and take a local PDF path with no network access. Its output goes to a path **outside** `content/data/` (schema FR-124) and becomes committed JSON only after a person has checked it entry by entry against the cited PDF page. It MUST abort before writing anything unless the PDF is SRD 5.2.1 as expected: 364 pages, the footer of page 28 reading "28", the first heading of page 28 "Barbarian" and the last heading of page 82 "Overchannel". | content-spells.md FR-071; `doc/research/sources.md` | unhappy (invalid input) |
| FR-087 | The tool MUST be deterministic (same PDF and tool version, byte-identical files), MUST report every problem in one run with class, page, field and kind, and MUST write no file when any problem exists. | content-spells.md FR-072, FR-073; architecture.md §5 | unhappy (determinism) |
| FR-088 | The tool MUST NOT supply a value it cannot read from the PDF. Its output is a skeleton: the Core Traits values, the Features table values, and the feature entries with `id`, `name`, `level` and `text`, with `facts`, `options` and `tables` left absent for a person to author. Moving the output into `content/data/` is the human check of schema FR-124, never a step of the tool. A missing Core Traits row, a table with other than 20 levels or a feature heading it cannot parse is a problem. | AGENTS.md ("If it is not in that PDF, it does not go in this repository"); content-spells.md FR-074 | unhappy (missing content) |
| FR-089 | A second, independent text pass (a different reading order of the same PDF) MUST be compared with the committed `text` of every feature and option; each difference that survives normalisation (collapse whitespace, resolve end-of-line hyphens, drop page headers and footers, drop the entries of a checked-in allow-list of known layout artefacts) is a finding, accepted only through the correction log of FR-091. | content-spells.md FR-079 | unhappy (rule interaction) |
| FR-090 | A numeric pass MUST extract each Features table from the PDF and compare its Proficiency Bonus, column values and cell lists with the committed progression and rows, and the spell-slot, Cantrips and Prepared Spells values with table SQ; any difference fails and names class, level and column. | SRD 5.2.1, Features tables | unhappy (state error: stale data) |
| FR-091 | A hand correction or accepted finding MUST be a log entry (class, field, printed page, reason) that the tool applies; data files MUST NOT be hand-edited except for the hand-authored parts of FR-088. The log-applying logic MUST be a pure function tested offline on synthetic records, and an entry that matches nothing is rejected. | content-spells.md FR-077 | unhappy (state error: stale log) |
| FR-092 | A PDF drift check (the tool re-run against the PDF compared with the committed JSON) MUST fail on any difference the log does not explain, **comparing only the fields the tool produces**: the Core Traits values, the Features-table values, and each entry's `id`, `name`, `level` and `text`. The hand-authored `facts`, `options` and `tables` (FR-088) are ignored by this comparison; it runs locally where the PDF is available, not in CI. It is distinct from the CI check of the embedded directory (schema FR-104; there is no generated copy to drift, shared-types-and-ownership.md FR-032). | content-spells.md FR-078; shared-types-and-ownership.md FR-032 | unhappy (state error: stale data) |
| FR-093 | The oracle MUST be a hand-transcribed fixture `content/testdata/srd/classes/oracle.json` mirroring tables SRC, CT1 to CT3, MC, SC, FS, HS, SQ, CC, F, RR, SF, FA, OP and TB, written independently of the JSON in `content/data/` (not generated from it) so that two transcriptions must agree. `testdata/srd/` is a path covered by `NOTICE.md` (shared-types-and-ownership.md FR-033). | testing.md §6; shared-types-and-ownership.md FR-033 | unhappy (rule interaction) |
| FR-094 | The bundled text MUST come from SRD 5.2.1 only; nothing from another edition or book enters a feature, option or table. Faithfulness is enforced by FR-089 and FR-090 and remains a review task where a tool cannot decide (schema section 5, anti-corruption rule 6). | AGENTS.md ("Licensing and source material") | unhappy (missing content) |
| FR-100 | Once all twelve classes and the full spell set are bundled, the bundled-data check MUST assert, using `classSpellList` (content-spells.md FR-120): every `ClassID` named by a bundled spell's `classes` is a bundled class with `spellcasting`; each of the eight spellcasting classes has at least one spell of every spell level it can cast and of level 0 when it has cantrips; the four non-casters have none. It is a bundle check across the class and spell data, never a decoder rule or a dependency of `content-classes.md` (shared-types-and-ownership.md FR-054, FR-067). **Staging:** until both sets are bundled the check is skipped behind an explicit pin (bundled spell count still the slice's) that fails when the full set lands, in the change that enables this check; the classes bundled so far are checked only for "every `ClassID` named by a bundled spell is a table SRC caster" (content-classes.md Q4). | SRD 5.2.1, "Class Spell Lists", p. 105; Features tables | unhappy (rule interaction) |
| FR-101 | The membership counts per class of the derived lists MUST equal the class row of the "Distributions" table of content-spells.md (Bard 130, Cleric 109, Druid 124, Paladin 38, Ranger 48, Sorcerer 140, Warlock 72, Wizard 218; content-spells.md FR-022), checked by a scenario test through the public API once all eight spellcasting classes and the full spell set are bundled (until then the counts are not asserted, behind the pin of FR-100). The printed lists are the verification oracle owned by content-spells.md FR-023; this spec adds no second copy. | content-spells.md FR-022, FR-023; SRD 5.2.1, class spell lists, pp. 33-82 | per case (content-spells.md "Distributions") |

## 5. Model

### Data format and process (phase 3)

Each class is one document `content/data/class_<class_id>.json` (FR-083):
`{"schemaVersion": 1, "classes": [<one entry in the field order of content-classes.md FR-002>]}`.
The files are drafted by `tools/extract_classes` (FR-086 to FR-088), structured by hand
(`facts`, `options`, `tables`) and verified by FR-089, FR-090, the oracle of FR-093 and the
bundled-data checks of schema FR-089, FR-090 and FR-104. A class ships in the change that adds its
`class-*.md` behaviour. The `go:embed` rule is shared (shared-types-and-ownership.md
FR-030).

### Table SRC: classes and subclasses

| Class id | Name | Class page | Features table page | Subclass id | Subclass name | Subclass section and page |
|---|---|---|---|---|---|---|
| `barbarian` | Barbarian | 28 | 28 | `path_of_the_berserker` | Path of the Berserker | "Barbarian Subclass: Path of the Berserker", 30 |
| `bard` | Bard | 31 | 31 | `college_of_lore` | College of Lore | "Bard Subclass: College of Lore", 35 |
| `cleric` | Cleric | 36 | 36 | `life_domain` | Life Domain | "Cleric Subclass: Life Domain", 40 |
| `druid` | Druid | 41 | 41 | `circle_of_the_land` | Circle of the Land | "Druid Subclass: Circle of the Land", 46 |
| `fighter` | Fighter | 47 | 47 | `champion` | Champion | "Fighter Subclass: Champion", 49 |
| `monk` | Monk | 49 | 50 | `warrior_of_the_open_hand` | Warrior of the Open Hand | "Monk Subclass: Warrior of the Open Hand", 52 |
| `paladin` | Paladin | 53 | 53 | `oath_of_devotion` | Oath of Devotion | "Paladin Subclass: Oath of Devotion", 56 |
| `ranger` | Ranger | 57 | 58 | `hunter` | Hunter | "Ranger Subclass: Hunter", 61 |
| `rogue` | Rogue | 61 | 62 | `thief` | Thief | "Rogue Subclass: Thief", 64 |
| `sorcerer` | Sorcerer | 64 | 65 | `draconic_sorcery` | Draconic Sorcery | "Sorcerer Subclass: Draconic Sorcery", 69 |
| `warlock` | Warlock | 70 | 71 | `fiend_patron` | Fiend Patron | "Warlock Subclass: Fiend Patron", 76 |
| `wizard` | Wizard | 77 | 77 | `evoker` | Evoker | "Wizard Subclass: Evoker", 82 |

The class section is the class name. Source pages are the first pages of the sections.

### Table CT1: Core Traits (ability, die, saves, skills)

| Class | Primary Ability | Hit Point Die | Saving throws | Skill choice (picks, options in printed order) | Source |
|---|---|---|---|---|---|
| Barbarian | `AllOf` Strength | 12 | Strength, Constitution | 2 of Animal Handling, Athletics, Intimidation, Nature, Perception, Survival | p. 28 |
| Bard | `AllOf` Charisma | 8 | Dexterity, Charisma | 3 of any skill | p. 31 |
| Cleric | `AllOf` Wisdom | 8 | Wisdom, Charisma | 2 of History, Insight, Medicine, Persuasion, Religion | p. 36 |
| Druid | `AllOf` Wisdom | 8 | Intelligence, Wisdom | 2 of Animal Handling, Arcana, Insight, Medicine, Nature, Perception, Religion, Survival | p. 41 |
| Fighter | `AnyOf` Strength, Dexterity | 10 | Strength, Constitution | 2 of Acrobatics, Animal Handling, Athletics, History, Insight, Intimidation, Persuasion, Perception, Survival | p. 47 |
| Monk | `AllOf` Dexterity, Wisdom | 8 | Strength, Dexterity | 2 of Acrobatics, Athletics, History, Insight, Religion, Stealth | p. 49 |
| Paladin | `AllOf` Strength, Charisma | 10 | Wisdom, Charisma | 2 of Athletics, Insight, Intimidation, Medicine, Persuasion, Religion | p. 53 |
| Ranger | `AllOf` Dexterity, Wisdom | 10 | Strength, Dexterity | 3 of Animal Handling, Athletics, Insight, Investigation, Nature, Perception, Stealth, Survival | p. 57 |
| Rogue | `AllOf` Dexterity | 8 | Dexterity, Intelligence | 4 of Acrobatics, Athletics, Deception, Insight, Intimidation, Investigation, Perception, Persuasion, Sleight of Hand, Stealth | p. 61 |
| Sorcerer | `AllOf` Charisma | 6 | Constitution, Charisma | 2 of Arcana, Deception, Insight, Intimidation, Persuasion, Religion | p. 64 |
| Warlock | `AllOf` Charisma | 8 | Wisdom, Charisma | 2 of Arcana, Deception, History, Intimidation, Investigation, Nature, Religion | p. 70 |
| Wizard | `AllOf` Intelligence | 6 | Intelligence, Wisdom | 2 of Arcana, History, Insight, Investigation, Medicine, Nature, Religion | p. 77 |

The Class Overview table (p. 19) agrees on every Primary Ability (FR-016). The Hit Point Die column is the `HitPointDie` member `d<n>` (JSON `"D<n>"`, FR-017).

### Table CT2: Core Traits (training)

| Class | Weapon Proficiencies | Tool Proficiencies | Armor training |
|---|---|---|---|
| Barbarian | Simple; Martial | none | Light, Medium, Shield |
| Bard | Simple | choice of Musical Instrument, 3 picks | Light |
| Cleric | Simple | none | Light, Medium, Shield |
| Druid | Simple | specific Herbalism Kit | Light, Shield |
| Fighter | Simple; Martial | none | Light, Medium, Heavy, Shield |
| Monk | Simple; Martial requiring Light | choice of Artisan's Tools or Musical Instrument, 1 pick | none |
| Paladin | Simple; Martial | none | Light, Medium, Heavy, Shield |
| Ranger | Simple; Martial | none | Light, Medium, Shield |
| Rogue | Simple; Martial requiring Finesse or Light | specific Thieves' Tools | Light |
| Sorcerer | Simple | none | none |
| Warlock | Simple | none | Light |
| Wizard | Simple | none | none |

### Table CT3: Starting Equipment

Items are `name x quantity`; "GP" is the option's `gold`.

| Class | Option A | Option B | Option C | Source |
|---|---|---|---|---|
| Barbarian | Greataxe x1, Handaxe x4, Explorer's Pack x1; 15 | no items; 75 | none | p. 28 |
| Bard | Leather Armor x1, Dagger x2, Musical Instrument of your choice x1, Entertainer's Pack x1; 19 | no items; 90 | none | p. 31 |
| Cleric | Chain Shirt x1, Shield x1, Mace x1, Holy Symbol x1, Priest's Pack x1; 7 | no items; 110 | none | p. 36 |
| Druid | Leather Armor x1, Shield x1, Sickle x1, Druidic Focus (Quarterstaff) x1, Explorer's Pack x1, Herbalism Kit x1; 9 | no items; 50 | none | p. 41 |
| Fighter | Chain Mail x1, Greatsword x1, Flail x1, Javelin x8, Dungeoneer's Pack x1; 4 | Studded Leather Armor x1, Scimitar x1, Shortsword x1, Longbow x1, Arrow x20, Quiver x1, Dungeoneer's Pack x1; 11 | no items; 155 | p. 47 |
| Monk | Spear x1, Dagger x5, Artisan's Tools or Musical Instrument chosen for the tool proficiency above x1, Explorer's Pack x1; 11 | no items; 50 | none | p. 49 |
| Paladin | Chain Mail x1, Shield x1, Longsword x1, Javelin x6, Holy Symbol x1, Priest's Pack x1; 9 | no items; 150 | none | p. 53 |
| Ranger | Studded Leather Armor x1, Scimitar x1, Shortsword x1, Longbow x1, Arrow x20, Quiver x1, Druidic Focus (sprig of mistletoe) x1, Explorer's Pack x1; 7 | no items; 150 | none | p. 57 |
| Rogue | Leather Armor x1, Dagger x2, Shortsword x1, Shortbow x1, Arrow x20, Quiver x1, Thieves' Tools x1, Burglar's Pack x1; 8 | no items; 100 | none | p. 61 |
| Sorcerer | Spear x1, Dagger x2, Arcane Focus (crystal) x1, Dungeoneer's Pack x1; 28 | no items; 50 | none | p. 64 |
| Warlock | Leather Armor x1, Sickle x1, Dagger x2, Arcane Focus (orb) x1, Book (occult lore) x1, Scholar's Pack x1; 15 | no items; 100 | none | p. 70 |
| Wizard | Dagger x2, Arcane Focus (Quarterstaff) x1, Robe x1, Spellbook x1, Scholar's Pack x1; 5 | no items; 55 | none | p. 77 |

### Table MC: Multiclass grants (beyond the Hit Point Die and the level 1 features)

| Class | Weapon | Armor | Skill | Tool |
|---|---|---|---|---|
| Barbarian | Martial | Shield | none | none |
| Bard | none | Light | any, 1 pick | choice of Musical Instrument, 1 pick |
| Cleric | none | Light, Medium, Shield | none | none |
| Druid | none | Light, Shield | none | none |
| Fighter | Martial | Light, Medium, Shield | none | none |
| Monk | none | none | none | none |
| Paladin | Martial | Light, Medium, Shield | none | none |
| Ranger | Martial | Light, Medium, Shield | listed (the Ranger skill list), 1 pick | none |
| Rogue | none | Light | listed (the Rogue skill list), 1 pick | specific Thieves' Tools |
| Sorcerer | none | none | none | none |
| Warlock | none | Light | none | none |
| Wizard | none | none | none | none |

### Table SC: Spellcasting

| Class | `spellcasting` | Ability | Cantrips column |
|---|---|---|---|
| Barbarian, Fighter, Monk, Rogue | absent | n/a | n/a |
| Bard | `slots` | Charisma | yes |
| Cleric | `slots` | Wisdom | yes |
| Druid | `slots` | Wisdom | yes |
| Paladin | `slots` | Charisma | no |
| Ranger | `slots` | Wisdom | no |
| Sorcerer | `slots` | Charisma | yes |
| Warlock | `pact` | Charisma | yes (a Cantrips column) |
| Wizard | `slots` | Intelligence | yes |

### Table FS: slot rows for Bard, Cleric, Druid, Sorcerer and Wizard

Spell slots for spell levels 1 to 9 at each class level; a printed "—" is 0.

| Level | Slots 1 to 9 | Level | Slots 1 to 9 |
|---|---|---|---|
| 1 | 2,0,0,0,0,0,0,0,0 | 11 | 4,3,3,3,2,1,0,0,0 |
| 2 | 3,0,0,0,0,0,0,0,0 | 12 | 4,3,3,3,2,1,0,0,0 |
| 3 | 4,2,0,0,0,0,0,0,0 | 13 | 4,3,3,3,2,1,1,0,0 |
| 4 | 4,3,0,0,0,0,0,0,0 | 14 | 4,3,3,3,2,1,1,0,0 |
| 5 | 4,3,2,0,0,0,0,0,0 | 15 | 4,3,3,3,2,1,1,1,0 |
| 6 | 4,3,3,0,0,0,0,0,0 | 16 | 4,3,3,3,2,1,1,1,0 |
| 7 | 4,3,3,1,0,0,0,0,0 | 17 | 4,3,3,3,2,1,1,1,1 |
| 8 | 4,3,3,2,0,0,0,0,0 | 18 | 4,3,3,3,3,1,1,1,1 |
| 9 | 4,3,3,3,1,0,0,0,0 | 19 | 4,3,3,3,3,2,1,1,1 |
| 10 | 4,3,3,3,2,0,0,0,0 | 20 | 4,3,3,3,3,2,2,1,1 |

### Table HS: slot rows for Paladin and Ranger

Spell levels 6 to 9 are 0 at every level.

| Level | Slots 1 to 5 | Level | Slots 1 to 5 |
|---|---|---|---|
| 1 | 2,0,0,0,0 | 11 | 4,3,3,0,0 |
| 2 | 2,0,0,0,0 | 12 | 4,3,3,0,0 |
| 3 | 3,0,0,0,0 | 13 | 4,3,3,1,0 |
| 4 | 3,0,0,0,0 | 14 | 4,3,3,1,0 |
| 5 | 4,2,0,0,0 | 15 | 4,3,3,2,0 |
| 6 | 4,2,0,0,0 | 16 | 4,3,3,2,0 |
| 7 | 4,3,0,0,0 | 17 | 4,3,3,3,1 |
| 8 | 4,3,0,0,0 | 18 | 4,3,3,3,1 |
| 9 | 4,3,2,0,0 | 19 | 4,3,3,3,2 |
| 10 | 4,3,2,0,0 | 20 | 4,3,3,3,2 |

### Table SQ: Cantrips, Prepared Spells and Pact Magic by level 1 to 20

| Class | Cantrips | Prepared Spells |
|---|---|---|
| Bard | 2,2,2,3,3,3,3,3,3,4,4,4,4,4,4,4,4,4,4,4 | 4,5,6,7,9,10,11,12,14,15,16,16,17,17,18,18,19,20,21,22 |
| Cleric | 3,3,3,4,4,4,4,4,4,5,5,5,5,5,5,5,5,5,5,5 | 4,5,6,7,9,10,11,12,14,15,16,16,17,17,18,18,19,20,21,22 |
| Druid | 2,2,2,3,3,3,3,3,3,4,4,4,4,4,4,4,4,4,4,4 | 4,5,6,7,9,10,11,12,14,15,16,16,17,17,18,18,19,20,21,22 |
| Paladin | no column | 2,3,4,5,6,6,7,7,9,9,10,10,11,11,12,12,14,14,15,15 |
| Ranger | no column | 2,3,4,5,6,6,7,7,9,9,10,10,11,11,12,12,14,14,15,15 |
| Sorcerer | 4,4,4,5,5,5,5,5,5,6,6,6,6,6,6,6,6,6,6,6 | 2,4,6,7,9,10,11,12,14,15,16,16,17,17,18,18,19,20,21,22 |
| Warlock | 2,2,2,3,3,3,3,3,3,4,4,4,4,4,4,4,4,4,4,4 | 2,3,4,5,6,7,8,9,10,10,11,11,12,12,13,13,14,14,15,15 |
| Wizard | 3,3,3,4,4,4,4,4,4,5,5,5,5,5,5,5,5,5,5,5 | 4,5,6,7,9,10,11,12,14,15,16,16,17,18,19,21,22,23,24,25 |

Warlock Pact Magic by level 1 to 20: Spell Slots 1,2,2,2,2,2,2,2,2,2,3,3,3,3,3,3,4,4,4,4;
Slot Level 1,1,2,2,3,3,4,4,5,5,5,5,5,5,5,5,5,5,5,5.

### Table CC: class-specific columns (id, header, kind, values by level 1 to 20)

| Class | Column id, header, kind | Values by level |
|---|---|---|
| Barbarian | `rages`, Rages, `count` | 2,2,3,3,3,4,4,4,4,4,4,5,5,5,5,5,6,6,6,6 |
| Barbarian | `rage_damage`, Rage Damage, `bonus` | +2 at 1-8, +3 at 9-15, +4 at 16-20 |
| Barbarian | `weapon_mastery`, Weapon Mastery, `count` | 2 at 1-3, 3 at 4-9, 4 at 10-20 |
| Bard | `bardic_die`, Bardic Die, `dice` | 1d6 at 1-4, 1d8 at 5-9, 1d10 at 10-14, 1d12 at 15-20 |
| Cleric | `channel_divinity`, Channel Divinity, `count` | none at 1, 2 at 2-5, 3 at 6-17, 4 at 18-20 |
| Druid | `wild_shape`, Wild Shape, `count` | none at 1, 2 at 2-5, 3 at 6-16, 4 at 17-20 |
| Fighter | `second_wind`, Second Wind, `count` | 2 at 1-3, 3 at 4-9, 4 at 10-20 |
| Fighter | `weapon_mastery`, Weapon Mastery, `count` | 3 at 1-3, 4 at 4-9, 5 at 10-15, 6 at 16-20 |
| Monk | `martial_arts`, Martial Arts, `dice` | 1d6 at 1-4, 1d8 at 5-10, 1d10 at 11-16, 1d12 at 17-20 |
| Monk | `focus_points`, Focus Points, `count` | none at 1, then the level (2 at 2 up to 20 at 20) |
| Monk | `unarmored_movement`, Unarmored Movement, `feetBonus` | none at 1, +10 at 2-5, +15 at 6-9, +20 at 10-13, +25 at 14-17, +30 at 18-20 |
| Paladin | `channel_divinity`, Channel Divinity, `count` | none at 1-2, 2 at 3-10, 3 at 11-20 |
| Ranger | `favored_enemy`, Favored Enemy, `count` | 2 at 1-4, 3 at 5-8, 4 at 9-12, 5 at 13-16, 6 at 17-20 |
| Rogue | `sneak_attack`, Sneak Attack, `dice` | 1d6 at 1-2, 2d6 at 3-4, 3d6 at 5-6, 4d6 at 7-8, 5d6 at 9-10, 6d6 at 11-12, 7d6 at 13-14, 8d6 at 15-16, 9d6 at 17-18, 10d6 at 19-20 |
| Sorcerer | `sorcery_points`, Sorcery Points, `count` | none at 1, then the level (2 at 2 up to 20 at 20) |
| Warlock | `eldritch_invocations`, Eldritch Invocations, `count` | 1,3,3,3,5,5,6,6,7,7,7,8,8,8,9,9,9,10,10,10 |
| Barbarian, Bard, Cleric, Druid, Fighter, Monk, Paladin, Ranger, Rogue, Sorcerer, Warlock | no other columns | n/a |
| Wizard | none | `columns` is `[]` |

A class lists its columns in the order of this table; the Warlock's Cantrips, Prepared
Spells, Spell Slots and Slot Level are spellcasting parts (table SQ), not columns.

### Table F: feature entries by class and subclass (level: name, in printed order)

Ids are `deriveFeatureId(ownerId, name)` (FR-052); the two Improved Brutal Strike entries
are `barbarian_improved_brutal_strike_level_13` and `..._level_17` (FR-053).

| Owner | Entries | Count |
|---|---|---|
| Barbarian (pp. 28-30) | 1: Rage, Unarmored Defense, Weapon Mastery; 2: Danger Sense, Reckless Attack; 3: Barbarian Subclass, Primal Knowledge; 4: Ability Score Improvement; 5: Extra Attack, Fast Movement; 7: Feral Instinct, Instinctive Pounce; 9: Brutal Strike; 11: Relentless Rage; 13: Improved Brutal Strike; 15: Persistent Rage; 17: Improved Brutal Strike; 18: Indomitable Might; 19: Epic Boon; 20: Primal Champion | 20 |
| Path of the Berserker (p. 30) | 3: Frenzy; 6: Mindless Rage; 10: Retaliation; 14: Intimidating Presence | 4 |
| Bard (pp. 31-33) | 1: Bardic Inspiration, Spellcasting; 2: Expertise, Jack of All Trades; 3: Bard Subclass; 4: Ability Score Improvement; 5: Font of Inspiration; 7: Countercharm; 10: Magical Secrets; 18: Superior Inspiration; 19: Epic Boon; 20: Words of Creation | 12 |
| College of Lore (p. 35) | 3: Bonus Proficiencies, Cutting Words; 6: Magical Discoveries; 14: Peerless Skill | 4 |
| Cleric (pp. 36-38) | 1: Spellcasting, Divine Order; 2: Channel Divinity; 3: Cleric Subclass; 4: Ability Score Improvement; 5: Sear Undead; 7: Blessed Strikes; 10: Divine Intervention; 14: Improved Blessed Strikes; 19: Epic Boon; 20: Greater Divine Intervention | 11 |
| Life Domain (p. 40) | 3: Disciple of Life, Life Domain Spells, Preserve Life; 6: Blessed Healer; 17: Supreme Healing | 5 |
| Druid (pp. 41-43) | 1: Spellcasting, Druidic, Primal Order; 2: Wild Shape, Wild Companion; 3: Druid Subclass; 4: Ability Score Improvement; 5: Wild Resurgence; 7: Elemental Fury; 15: Improved Elemental Fury; 18: Beast Spells; 19: Epic Boon; 20: Archdruid | 13 |
| Circle of the Land (p. 46) | 3: Circle of the Land Spells, Land's Aid; 6: Natural Recovery; 10: Nature's Ward; 14: Nature's Sanctuary | 5 |
| Fighter (pp. 47-48) | 1: Fighting Style, Second Wind, Weapon Mastery; 2: Action Surge, Tactical Mind; 3: Fighter Subclass; 4: Ability Score Improvement; 5: Extra Attack, Tactical Shift; 9: Indomitable, Tactical Master; 11: Two Extra Attacks; 13: Studied Attacks; 19: Epic Boon; 20: Three Extra Attacks | 15 |
| Champion (p. 49) | 3: Improved Critical, Remarkable Athlete; 7: Additional Fighting Style; 10: Heroic Warrior; 15: Superior Critical; 18: Survivor | 6 |
| Monk (pp. 50-52) | 1: Martial Arts, Unarmored Defense; 2: Monk's Focus, Unarmored Movement, Uncanny Metabolism; 3: Deflect Attacks, Monk Subclass; 4: Ability Score Improvement, Slow Fall; 5: Extra Attack, Stunning Strike; 6: Empowered Strikes; 7: Evasion; 9: Acrobatic Movement; 10: Heightened Focus, Self-Restoration; 13: Deflect Energy; 14: Disciplined Survivor; 15: Perfect Focus; 18: Superior Defense; 19: Epic Boon; 20: Body and Mind | 22 |
| Warrior of the Open Hand (p. 52) | 3: Open Hand Technique; 6: Wholeness of Body; 11: Fleet Step; 17: Quivering Palm | 4 |
| Paladin (pp. 53-55) | 1: Lay On Hands, Spellcasting, Weapon Mastery; 2: Fighting Style, Paladin's Smite; 3: Channel Divinity, Paladin Subclass; 4: Ability Score Improvement; 5: Extra Attack, Faithful Steed; 6: Aura of Protection; 9: Abjure Foes; 10: Aura of Courage; 11: Radiant Strikes; 14: Restoring Touch; 18: Aura Expansion; 19: Epic Boon | 17 |
| Oath of Devotion (pp. 56-57) | 3: Oath of Devotion Spells, Sacred Weapon; 7: Aura of Devotion; 15: Smite of Protection; 20: Holy Nimbus | 5 |
| Ranger (pp. 57-59) | 1: Spellcasting, Favored Enemy, Weapon Mastery; 2: Deft Explorer, Fighting Style; 3: Ranger Subclass; 4: Ability Score Improvement; 5: Extra Attack; 6: Roving; 9: Expertise; 10: Tireless; 13: Relentless Hunter; 14: Nature's Veil; 17: Precise Hunter; 18: Feral Senses; 19: Epic Boon; 20: Foe Slayer | 17 |
| Hunter (p. 61) | 3: Hunter's Lore, Hunter's Prey; 7: Defensive Tactics; 11: Superior Hunter's Prey; 15: Superior Hunter's Defense | 5 |
| Rogue (pp. 61-63) | 1: Expertise, Sneak Attack, Thieves' Cant, Weapon Mastery; 2: Cunning Action; 3: Rogue Subclass, Steady Aim; 4: Ability Score Improvement; 5: Cunning Strike, Uncanny Dodge; 7: Evasion, Reliable Talent; 11: Improved Cunning Strike; 14: Devious Strikes; 15: Slippery Mind; 18: Elusive; 19: Epic Boon; 20: Stroke of Luck | 18 |
| Thief (p. 64) | 3: Fast Hands, Second-Story Work; 9: Supreme Sneak; 13: Use Magic Device; 17: Thief's Reflexes | 5 |
| Sorcerer (pp. 64-66) | 1: Spellcasting, Innate Sorcery; 2: Font of Magic, Metamagic; 3: Sorcerer Subclass; 4: Ability Score Improvement; 5: Sorcerous Restoration; 7: Sorcery Incarnate; 19: Epic Boon; 20: Arcane Apotheosis | 10 |
| Draconic Sorcery (pp. 69-70) | 3: Draconic Resilience, Draconic Spells; 6: Elemental Affinity; 14: Dragon Wings; 18: Dragon Companion | 5 |
| Warlock (pp. 70-72) | 1: Eldritch Invocations, Pact Magic; 2: Magical Cunning; 3: Warlock Subclass; 4: Ability Score Improvement; 9: Contact Patron; 11: Mystic Arcanum; 19: Epic Boon; 20: Eldritch Master | 9 |
| Fiend Patron (p. 76) | 3: Dark One's Blessing, Fiend Spells; 6: Dark One's Own Luck; 10: Fiendish Resilience; 14: Hurl Through Hell | 5 |
| Wizard (pp. 77-79) | 1: Spellcasting, Ritual Adept, Arcane Recovery; 2: Scholar; 3: Wizard Subclass; 4: Ability Score Improvement; 5: Memorize Spell; 18: Spell Mastery; 19: Epic Boon; 20: Signature Spells | 10 |
| Evoker (p. 82) | 3: Evocation Savant, Potent Cantrip; 6: Sculpt Spells; 10: Empowered Evocation; 14: Overchannel | 5 |
| Totals | 12 classes: 174 entries; 12 subclasses: 58 entries | 232 |

### Table RR: row cells that are not first-level entries (FR-046)

Each class's row cell at a level is the entries of table F at that level in printed order,
plus the references below (annotation in quotes), `RowSubclassFeature` where table SF says
so, and `[]` where listed as "—". Reference entries are the older entry of the same name.

| Class | References (level: cell) | "—" rows |
|---|---|---|
| Barbarian | 8, 12, 16: Ability Score Improvement | none |
| Bard | 8, 12, 16: Ability Score Improvement; 9: Expertise | 11, 13, 15, 17 |
| Cleric | 8, 12, 16: Ability Score Improvement | 9, 11, 13, 15, 18 |
| Druid | 8, 12, 16: Ability Score Improvement | 9, 11, 13, 17 |
| Fighter | 6, 8, 12, 14, 16: Ability Score Improvement; own entries with annotation: 2: Action Surge "one use", 9: Indomitable "one use"; 13: Indomitable "two uses", then Studied Attacks; 17: Action Surge "two uses", then Indomitable "three uses" | none |
| Monk | 8, 12, 16: Ability Score Improvement | none |
| Paladin | 8, 12, 16: Ability Score Improvement | 13, 17 |
| Ranger | 8, 12, 16: Ability Score Improvement | none |
| Rogue | 6: Expertise; 8, 10, 12, 16: Ability Score Improvement | none |
| Sorcerer | 8, 12, 16: Ability Score Improvement; 10, 17: Metamagic | 9, 11, 13, 15 |
| Warlock | 8, 12, 16: Ability Score Improvement; 11: Mystic Arcanum "level 6 spell" (own entry); 13: Mystic Arcanum "level 7 spell"; 15: Mystic Arcanum "level 8 spell"; 17: Mystic Arcanum "level 9 spell" | 5, 7, 18 |
| Wizard | 8, 12, 16: Ability Score Improvement | 7, 9, 11, 13, 15, 17 |

The Fighter's row at level 13 lists Indomitable before Studied Attacks and at level 9
lists Indomitable before Tactical Master, as printed. The Barbarian's rows 13 and 17
reference their own level's Improved Brutal Strike entry.

### Table SF: rows with a "Subclass feature" cell

| Class | Levels of `RowSubclassFeature` |
|---|---|
| Barbarian | 6, 10, 14 |
| Bard | 6, 14 |
| Cleric | 6, 17 |
| Druid | 6, 10, 14 |
| Fighter | 7, 10, 15, 18 |
| Monk | 6, 11, 17 |
| Paladin | 7, 15, 20 |
| Ranger | 7, 11, 15 |
| Rogue | 9, 13, 17 |
| Sorcerer | 6, 14, 18 |
| Warlock | 6, 10, 14 |
| Wizard | 6, 10, 14 |

Each subclass has entries at the subclass level 3 and at exactly these levels (FR-077).

### Table FA: structured facts

All facts not listed are absent. Spell ids use the derivation of content-spells.md FR-003.

| Owner and feature | Fact | Detail |
|---|---|---|
| each of the 12 classes: "<Class> Subclass" (level 3) | `GrantsSubclass` | none |
| each of the 12 classes: Ability Score Improvement (level 4) | `GrantsFeat` | category `General` |
| each of the 12 classes: Epic Boon (level 19) | `GrantsFeat` | category `Epic Boon` |
| Fighter: Fighting Style (level 1) | `GrantsFeat` | category `Fighting Style` |
| Paladin: Fighting Style (level 2) | `GrantsFeat` | category `Fighting Style` |
| Ranger: Fighting Style (level 2) | `GrantsFeat` | category `Fighting Style` |
| Champion: Additional Fighting Style (level 7) | `GrantsFeat` | category `Fighting Style` |
| Bard: Words of Creation (20) | `AlwaysPreparedSpells` | 20: power_word_heal, power_word_kill |
| Druid: Druidic (1) | `AlwaysPreparedSpells` | 1: speak_with_animals |
| Paladin: Paladin's Smite (2) | `AlwaysPreparedSpells` | 2: divine_smite |
| Paladin: Faithful Steed (5) | `AlwaysPreparedSpells` | 5: find_steed |
| Ranger: Favored Enemy (1) | `AlwaysPreparedSpells` | 1: hunters_mark |
| Warlock: Contact Patron (9) | `AlwaysPreparedSpells` | 9: contact_other_plane |
| Life Domain: Life Domain Spells (3) | `AlwaysPreparedSpells` | 3: aid, bless, cure_wounds, lesser_restoration; 5: mass_healing_word, revivify; 7: aura_of_life, death_ward; 9: greater_restoration, mass_cure_wounds |
| Oath of Devotion: Oath of Devotion Spells (3) | `AlwaysPreparedSpells` | 3: protection_from_evil_and_good, shield_of_faith; 5: aid, zone_of_truth; 9: beacon_of_hope, dispel_magic; 13: freedom_of_movement, guardian_of_faith; 17: commune, flame_strike |
| Draconic Sorcery: Draconic Spells (3) | `AlwaysPreparedSpells` | 3: alter_self, chromatic_orb, command, dragons_breath; 5: fear, fly; 7: arcane_eye, charm_monster; 9: legend_lore, summon_dragon |
| Fiend Patron: Fiend Spells (3) | `AlwaysPreparedSpells` | 3: burning_hands, command, scorching_ray, suggestion; 5: fireball, stinking_cloud; 7: fire_shield, wall_of_fire; 9: geas, insect_plague |
| Circle of the Land: Circle of the Land Spells (3), one fact on each option | `AlwaysPreparedSpells` | arid: 3: blur, burning_hands, fire_bolt; 5: fireball; 7: blight; 9: wall_of_stone. polar: 3: fog_cloud, hold_person, ray_of_frost; 5: sleet_storm; 7: ice_storm; 9: cone_of_cold. temperate: 3: misty_step, shocking_grasp, sleep; 5: lightning_bolt; 7: freedom_of_movement; 9: tree_stride. tropical: 3: acid_splash, ray_of_sickness, web; 5: stinking_cloud; 7: polymorph; 9: insect_plague |

The Wizard's Spell Mastery and Signature Spells, the Cleric's and Paladin's chosen
cantrips and every other "choose" are not facts: they are text, and behaviour (class-wizard.md,
class-cleric.md) owns them.

### Table OP: option lists

| Feature | Options (id: notes) |
|---|---|
| Cleric: Divine Order | protector, thaumaturge |
| Cleric: Blessed Strikes | divine_strike, potent_spellcasting |
| Druid: Primal Order | magician, warden |
| Druid: Elemental Fury | potent_spellcasting, primal_strike |
| Ranger Hunter: Hunter's Prey | colossus_slayer, horde_breaker |
| Ranger Hunter: Defensive Tactics | escape_the_horde, multiattack_defense |
| Circle of the Land: Circle of the Land Spells | arid, polar, temperate, tropical (each with its `AlwaysPreparedSpells` fact of table FA) |
| Sorcerer: Metamagic | careful_spell (1), distant_spell (1), empowered_spell (1), extended_spell (1), heightened_spell (2), quickened_spell (2), seeking_spell (1), subtle_spell (1), transmuted_spell (1), twinned_spell (1), the number being `pointCost` |
| Warlock: Eldritch Invocations | 28 options in printed order: agonizing_blast (min 2, note "a Warlock Cantrip That Deals Damage", repeatable), armor_of_shadows, ascendant_step (min 5), devils_sight (min 2), devouring_blade (min 12, requires thirsting_blade), eldritch_mind, eldritch_smite (min 5, requires pact_of_the_blade), eldritch_spear (min 2, note "a Warlock Cantrip That Deals Damage", repeatable), fiendish_vigor (min 2), gaze_of_two_minds (min 5), gift_of_the_depths (min 5), gift_of_the_protectors (min 9, requires pact_of_the_tome), investment_of_the_chain_master (min 5, requires pact_of_the_chain), lessons_of_the_first_ones (min 2, repeatable), lifedrinker (min 9, requires pact_of_the_blade), mask_of_many_faces (min 2), master_of_myriad_forms (min 5), misty_visions (min 2), one_with_shadows (min 5), otherworldly_leap (min 2), pact_of_the_blade, pact_of_the_chain, pact_of_the_tome, repelling_blast (min 2, note "a Warlock Cantrip That Deals Damage via an Attack Roll", repeatable), thirsting_blade (min 5, requires pact_of_the_blade), visions_of_distant_realms (min 9), whispers_of_the_grave (min 7), witch_sight (min 15) |

Option names are the printed names ("Devil's Sight"); ids are their derivation without
an owner prefix (apostrophes deleted). A printed "Prerequisite: Level N+ Warlock" is
`minClassLevel` N; a printed invocation prerequisite is `requiresOptionIds`; the remaining
printed words are `note`.

### Table TB: embedded tables

| Feature | Table name and headers | Rows |
|---|---|---|
| Druid: Wild Shape (p. 42) | Beast Shapes: Druid Level, Known Forms, Max CR, Fly Speed | 2, 4, "1/4", "No"; 4, 6, "1/2", "No"; 8, 8, "1", "Yes" |
| Sorcerer: Font of Magic (p. 66) | Creating Spell Slots: Spell Slot Level, Sorcery Point Cost, Min. Sorcerer Level | 1, 2, 2; 2, 3, 3; 3, 5, 5; 4, 6, 7; 5, 7, 9 |
| Circle of the Land: Nature's Ward (p. 46) | Nature's Ward: Land Type, Resistance | "Arid", "Fire"; "Polar", "Cold"; "Temperate", "Lightning"; "Tropical", "Poison" |

## 6. Interface

None. No type, port or use case: the files are data, the tool lives in `tools/` and is not imported by any package.

## 7. Observability

Silent by design: bundled data and the offline tool emit nothing; loading is recorded by the
existing `LoadSet` signals (content-classes.md section 7). **No signal is added or changed.**

1. **What can go wrong?** A file drifting from the PDF, a lost or duplicated id in assembly, a transcription error in the oracle.
2. **How would an operator know?** A rejected bundled set is a `LoadSetRejected` (failed load span, `loads{srd5e.outcome=failure}`); drift is caught earlier by tests and the local PDF check.
3. **What is needed to diagnose it?** Class, level, column and page in the tool and test reports (FR-087, FR-090); problem paths in the joined `Problem` error.
4. **Counted or timed?** Already, by the schema signals; nothing per class or per row.
5. **Is the existing instrumentation still correct?** Yes: `srd5e.content.kind` already lists `character_class`; `problem_kind` is unchanged.
6. **What to monitor?** `entries_loaded{srd5e.content.kind=character_class}` equals 12 for the bundled set.
7. **Is anything now noisy?** No: no log, nothing per class, feature or row.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none added or changed | n/a | n/a | n/a | existing schema signals suffice |

## 8. Test plan

Levels as in `content-classes.md` section 8: **architecture** (`internal/archtest`), **adapter**,
**tool** (`tools/extract_classes/*_test.go`, offline, synthetic input).

| Ids | Level | Notes |
|---|---|---|
| FR-083 | architecture | Each `class_<id>.json` is one valid embedded document listed by `data.Files()`, no subdirectory, `classes.json` absent once `class_wizard.json` exists (staging pins: content-classes.md FR-003, FR-069) |
| FR-084 | adapter | `BundledSource` construction with two documents sharing a class, subclass or feature id returns an error wrapping `srderr.ErrInvalidInput` naming id space and id (one test each); same string in two id spaces accepted; no `Problem` |
| FR-085 | adapter, architecture | Decode every present file; `source` and `source.page` equal the first page in SRC |
| FR-086 to FR-088 | tool | Wrong page count, footer, first heading each abort before writing; output outside `content/data/`; same input twice byte-identical; all problems reported; a missing Core Traits row, 19 levels, unparsable heading reported; no value invented |
| FR-089 to FR-092, FR-094 | tool, doc review | Text pass on synthetic text (normalisation rows, allow-list); numeric pass (a changed digit reported with class, level, column); pure log applier (unmatched entry rejected); local-only drift check ignores hand-authored fields, skipped from CI by location; review checks the cited page |
| FR-093 | architecture | Oracle fixture under `content/testdata/srd/classes/` is the only source of oracle values, not generated from `content/data/`; `NOTICE.md` lists the path |
| FR-100, FR-101 | architecture, scenario | Class ids on bundled spells; coverage by spell level; non-casters empty; skipped behind the staging pin until the full spell set and twelve classes are bundled; counts 130, 109, 124, 38, 48, 140, 72, 218 through the public API |

- **Doubles and contract suites:** none. **Mutation testing:** the log applier of FR-091, with `gomu` run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (doc/rules/testing.md).
- **Not covered:** prose faithfulness beyond what FR-089 can compare (review task).

## 9. Open questions and decisions

Open: none.

Settled:

- D12 the oracle is a hand-written fixture (FR-093); D13 one file per class (FR-083).

## 10. Changelog impact

Under `Unreleased`, **Added:** the twelve SRD classes and twelve subclasses as bundled content
(phase 3, one entry per class as its behaviour lands). Not **BREAKING** while unreleased; after
first release, adding or removing a bundled feature, option, column or id is **BREAKING**
(content-classes.md section 10).
