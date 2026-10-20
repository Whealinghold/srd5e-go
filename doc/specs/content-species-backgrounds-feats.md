# Bundled content: species, backgrounds and feats

| | |
|---|---|
| Status | partially implemented |
| Context | `content` |
| Roadmap phase | Types 2; bulk data 5 |
| Depends on | `content-schema-and-decoder.md`, `content-source-port.md` |

Phase rule: shared-types-and-ownership.md FR-014; see [../roadmap.md](../roadmap.md).
The species spell references are checked by whole-set validation (FR-023), not by a spec
dependency (shared-types-and-ownership.md FR-054, FR-067).

## 1. Purpose and scope

Defines the bulk **data** for the "Character Origins" and "Feats" categories of
SRD 5.2.1: every species, background and feat, as bundled JSON documents of the
current content schema version (shared-types-and-ownership.md FR-034) that
decode into `content` entry types. It specifies the exact bundled set (9
species, 4 backgrounds, 17 feats) with ids and pages; how each SRD description
maps onto schema fields (structured versus verbatim); the schema extensions SX-1
to SX-4 (section 5); the transcription process and fidelity controls; and the
validation of a custom background a host authors: the five steps (FR-072) and
the equipment-package check (FR-073), SRD 5.2.1 pp. 192-193.

**This spec owns the shapes of the three kinds** (section 5): `Species`,
`Background`, `Feat` and the supporting types only they use (`SpeciesChoice`,
`ChoiceOption` and variants, `LevelBenefit`, `BackgroundFeatReference`,
`EquipmentOptions`, `EquipmentOption`, `EquipmentItem`, `FeatBenefit`)
(shared-types-and-ownership.md FR-035). The envelope, ids, `source`, decoder and
failure type stay with content-schema-and-decoder.md ("schema" below); the shapes
are registered there as revision-table row `R1-species-backgrounds-feats`
(FR-066).

**Owned elsewhere, only referenced** (shared-types-and-ownership.md FR-046,
FR-048): `NamedText`, `FeatCategory`, `Prerequisite`, `ToolProficiency` (schema
FR-116 to FR-119); `CreatureSize`, `CreatureType`, `Skill`, `DamageType` (FR-106
to FR-109); `SpeciesID`, `BackgroundID`, `FeatID`, `SpellID` (FR-114); `Ability`
(`the kernel packages`, ability-scores.md). `EquipmentOption` and `EquipmentItem` are
defined here and shared with no other spec (section 9).

**Thin slice.** content-thin-slice.md is not a dependency
(shared-types-and-ownership.md FR-047). The nine species here replace its four;
`species.json` (FR-053) is the one species document and the thin-slice owner
removes its copy in the same change. Where it pins facts this spec keeps as
verbatim trait text, this spec's case tables win (section 5, "Declined
requests").

**Out of scope:**

- The `content.Source` port, `content.InMemorySource`, the decoder, `LoadSet`
  and its telemetry (content-source-port.md, schema); the data directory's layout
  and notice text (content-licensing-and-notice.md).
- Applying any of these rules to a character (ability increases, proficiencies,
  taking a feat, Breath Weapon damage, casting a lineage spell, choosing a size,
  uses per Long Rest): `character` and `combat`. The exception is the validation
  of a custom background (FR-072, FR-073), which checks authored data.
- The introductory prose of "Character Backgrounds" and "Character Species"
  (p. 83) beyond vocabulary; spells, equipment, tools, classes and class spell
  lists these entries mention (referenced by key); rendering or localisation of
  text; telemetry (section 7).
- Any species, background or feat not on pp. 83-88. Material that looks like core
  fifth-edition content but is not printed there stays out and is flagged in the
  change summary.

## 2. Source

- `SRD 5.2.1, "Contents", p. 2`: page ranges (PDF page = printed page; every citation is a printed page).
- Backgrounds: `"Character Backgrounds", "Background Descriptions", p. 83`.
  Species: `"Character Species", "Parts of a Species", pp. 83-84` (Creature Type
  and Size p. 83; Speed and Special Traits p. 84) and `"Species Descriptions",
  pp. 84-86` (Dragonborn, Dwarf, Elf start on p. 84; Elven Lineages table at the
  top of p. 85; Goliath starts p. 85; Fiendish Legacies table p. 86).
- `"Creating a Background", pp. 192-193` (heading and step 1 on p. 192; steps 2 to
  5 on p. 193); `"Coins", p. 89`: 100 CP to 1 GP (FR-073).
- Feats: `"Parts of a Feat", p. 87` (Category, Prerequisite, Benefit, Repeatable);
  `"Origin Feats"` and `"General Feats"` p. 87; `"Fighting Style Feats"`
  pp. 87-88; `"Epic Boon Feats"` p. 88.
- Vocabulary only: `"The Six Abilities", p. 5`; `"Skills", p. 9`; `"Damage
  Types", p. 180`; `"Tools", p. 93`.
- `doc/research/sources.md`, `AGENTS.md`, `doc/rules/architecture.md` §3, §5, §6,
  `observability.md` §2, §3, `testing.md` §4, §5, §8, §9.


## 3. Vocabulary

Types are those of section 5 unless an owner is named. Ids are lowercase
`snake_case` (shared-types-and-ownership.md FR-007). Terms use SRD 5.2.1 wording
(`doc/rules/architecture.md` §3): "Species", never "Race". Identity types and
vocabulary are listed in section 1; `ConditionKind` is schema FR-110.

| SRD term | Type | Notes |
|---|---|---|
| special trait | `NamedText` in `traits` (FR-014) | Name and verbatim text (p. 84). |
| lineage, legacy, ancestry | `SpeciesChoice` in `choices`; closed `ChoiceOption` (`DamageTypeOption`, `LineageOption`, `TextOption`); `LevelBenefit` (SX-1) | Draconic Ancestry, Elven Lineage, Gnomish Lineage, Giant Ancestry, Fiendish Legacy. |
| Feat (background) | `BackgroundFeatReference` `{featId, choice?}` (FR-025, FR-026) | Not `BackgroundFeat`, a `FeatSource` variant of feats-behaviour.md (FR-071). |
| Equipment (background) | `EquipmentOptions` `A`, `B` of `EquipmentOption{items, gold}` of `EquipmentItem{name, quantity}` (FR-029, FR-030) | "Choose A or B" (p. 83). |
| Repeatable (feat) | `repeatable` + `repeatableText` (SX-2, FR-039) | p. 87. |
| source citation | `source: {document, section, page}` (schema FR-034, FR-090) | `page` is the printed start page. |


## 4. Requirements

Ids are stable and never renumbered. "Schema FR-nnn" cites
content-schema-and-decoder.md; a problem is a (path, `ProblemKind`) pair
with JSON Pointer paths. A "bundled-data check" is a **data pin** (section 8) over
the decoded bundled set, never a decoder rule. "Constructor rejects" means
`NewX` returns an error matching `srderr.ErrInvalidInput` (FR-069). The JSON files under `content/data/` are the source of truth;
bundled entries reach tests and consumers as `go:embed` documents decoded by the one
`content.Decoder` (shared-types-and-ownership.md FR-030, FR-031). Lookups yield
`LookupFound` or `LookupNotFound`; a decode failure is `DecodeFailed` from
the port and `LoadSetRejected` from the use case (schema FR-102).


### 4.1 Set membership and identity

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | The bundled species set MUST contain exactly nine entries (Dragonborn, Dwarf, Elf, Gnome, Goliath, Halfling, Human, Orc, Tiefling) and no others. | SRD 5.2.1, "Species Descriptions", pp. 84-86 | happy |
| FR-002 | The bundled background set MUST contain exactly four entries (Acolyte, Criminal, Sage, Soldier) and no others. | SRD 5.2.1, "Background Descriptions", p. 83 | happy |
| FR-003 | The bundled feat set MUST contain exactly 17 entries and no others: four Origin (Alert, Magic Initiate, Savage Attacker, Skilled), two General (Ability Score Improvement, Grappler), four Fighting Style (Archery, Defense, Great Weapon Fighting, Two-Weapon Fighting) and seven Epic Boon (Boon of Combat Prowess, Boon of Dimensional Travel, Boon of Fate, Boon of Irresistible Offense, Boon of Spell Recall, Boon of the Night Spirit, Boon of Truesight). | SRD 5.2.1, "Feat Descriptions", pp. 87-88 | happy |
| FR-004 | An entry's `id` MUST be derived from its SRD name by this one rule (content-spells.md applies the same rule to spell names): lowercase; delete apostrophes (U+0027, U+2019); replace each maximal run of other characters outside `a-z0-9` with one `_`; trim a leading or trailing `_`. The same name always gives the same id (`boon_of_the_night_spirit`, `two_weapon_fighting`). No bundled name has an apostrophe. The id satisfies schema FR-030 (`^[a-z][a-z0-9_]*$`). | schema FR-030; SRD 5.2.1, names as printed pp. 83-88 | happy |
| FR-005 | Within one kind, entry ids MUST be unique: a repeat is rejected with (`/<array>/<index of the later entry>/id`, `duplicate`). | schema FR-031; testing.md §4 (duplicate entries) | unhappy (invalid input) |
| FR-006 | The same id in two different kinds (a feat and a background both `sage`) MUST be accepted: the key includes the kind. | schema FR-032; content-source-port.md | unhappy (rule interaction) |
| FR-007 | Every bundled entry MUST carry a `source` whose `document` is exactly `SRD 5.2.1`, whose `section` is the SRD heading of the entry and whose `page` is the printed start page. An absent `source` is accepted by the decoder (schema FR-034) and MUST fail the bundled-data check (schema FR-090). | AGENTS.md, "Licensing and source material"; schema FR-034, FR-090 | unhappy (missing content) |
| FR-008 | The decoder MUST reject a blank `source.section` with (`.../source/section`, `invalidValue`) and a `source.page` below 1 with (`.../source/page`, `outOfRange`). These are the schema's field rules; this spec adds no constructor. | schema FR-023, FR-027, FR-034 | unhappy (invalid input) |
| FR-009 | `source.page` of every bundled entry MUST equal its start page and lie in its category's printed range (backgrounds 83; species 84-86; feats 87-88). Start pages: backgrounds 83; Dragonborn, Dwarf, Elf 84; Gnome, Goliath 85; Halfling, Human, Orc, Tiefling 86; Alert, Magic Initiate, Savage Attacker, Skilled, Ability Score Improvement, Grappler, Archery 87; Defense, Great Weapon Fighting, Two-Weapon Fighting and the seven Epic Boons 88. A bundled-data check, not a decoder rule. | SRD 5.2.1, "Contents", p. 2 | unhappy (boundary) |
| FR-059 | All 30 ids MUST satisfy the schema id rule (at most 64 characters) and FR-004 applied to each SRD name MUST give exactly that id: species `dragonborn`, `dwarf`, `elf`, `gnome`, `goliath`, `halfling`, `human`, `orc`, `tiefling`; backgrounds `acolyte`, `criminal`, `sage`, `soldier`; feats `alert`, `magic_initiate`, `savage_attacker`, `skilled`, `ability_score_improvement`, `grappler`, `archery`, `defense`, `great_weapon_fighting`, `two_weapon_fighting`, `boon_of_combat_prowess`, `boon_of_dimensional_travel`, `boon_of_fate`, `boon_of_irresistible_offense`, `boon_of_spell_recall`, `boon_of_the_night_spirit`, `boon_of_truesight`. | schema FR-030 | happy |

### 4.2 Species

Fields, order and required status: section 5; extensions `sizeText` and `choices`
(SX-1, SX-3).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | A species missing any of `creatureType`, `sizes`, `speed` or `traits` MUST be rejected with (`/species/<i>/<field>`, `missingField`) for each missing field. | SRD 5.2.1, "Parts of a Species", pp. 83-84; schema FR-019 | unhappy (missing content) |
| FR-011 | Every bundled species MUST have `creatureType` Humanoid. | SRD 5.2.1, "Creature Type", p. 83 | happy |
| FR-012 | `sizes` has one or two distinct `CreatureSize` values, each Small or Medium. Zero or three entries is `outOfRange` at `/species/<i>/sizes` (then no element problems); a repeat is `duplicate` at the later index; any other size (Large) is `invalidValue` at `/species/<i>/sizes/<k>`. The `Species` constructor returns an error matching `srderr.ErrInvalidInput` for each. Dragonborn, Dwarf, Elf, Goliath, Orc: `[Medium]`; Gnome, Halfling: `[Small]`; Human, Tiefling: `[Small, Medium]` ("chosen when you select this species"). | SRD 5.2.1, pp. 83-86 | unhappy (boundary) |
| FR-013 | `speed` is an integer of 0 or more to the decoder and constructor: negative is `outOfRange`; a non-integer (`30.5`, `"30"`) is `wrongType` at `/species/<i>/speed` (schema FR-021). The data pin requires `speed` positive: 35 for Goliath, 30 for every other bundled species. | SRD 5.2.1, pp. 84-86; schema FR-021 | unhappy (boundary) |
| FR-014 | `traits` MUST be a non-empty list of `NamedText` in printed order with unique names (`duplicate`); a blank `name` or `text` is `invalidValue` (schema FR-027, FR-028); an empty list is `outOfRange`. | SRD 5.2.1, "Special Traits", p. 84 | unhappy (invalid input) |
| FR-015 | Each bundled species MUST contain exactly these trait names, in this order: Dragonborn: Draconic Ancestry; Breath Weapon; Damage Resistance; Darkvision; Draconic Flight. Dwarf: Darkvision; Dwarven Resilience; Dwarven Toughness; Stonecunning. Elf: Darkvision; Elven Lineage; Fey Ancestry; Keen Senses; Trance. Gnome: Darkvision; Gnomish Cunning; Gnomish Lineage. Goliath: Giant Ancestry; Large Form; Powerful Build. Halfling: Brave; Halfling Nimbleness; Luck; Naturally Stealthy. Human: Resourceful; Skillful; Versatile. Orc: Adrenaline Rush; Darkvision; Relentless Endurance. Tiefling: Darkvision; Fiendish Legacy; Otherworldly Presence. A fixture species with one trait dropped fails the check naming species and trait. | SRD 5.2.1, pp. 84-86 | happy |
| FR-016 | A trait asking the player to choose from a closed list MUST be a `choices` element (SX-1) whose options are the closed `ChoiceOption` family: `DamageTypeOption` (Draconic Ancestry, 10: `name` and `damageType`, no text, because the SRD table prints none), `LineageOption` (Elven Lineage, 3, and Fiendish Legacy, 3: `name` and `levelBenefits`; the Level 1 cell is `levelBenefits[0].text`, no other text) and `TextOption` (Gnomish Lineage, 2, and Giant Ancestry, 6: `name` and the verbatim `text` printed after the heading). Every option of one choice has the same variant. The decoder rejects an empty `options` (`outOfRange`), a repeated option name (`duplicate` at the later index), a differing variant (`constraintViolation` at `.../options/<j>/kind`), an unknown `kind` (`invalidValue`), a key outside the variant (`unknownField`) and a missing variant field (`missingField`) at `/species/<i>/choices/<j>/options`; the constructors throw an error matching `srderr.ErrInvalidInput` for the same cases. | SRD 5.2.1, "Dragonborn" p. 84, "Elf" p. 85, "Gnome" p. 85, "Goliath" pp. 85-86, "Tiefling" p. 86 | unhappy (invalid input) |
| FR-017 | Each Draconic Ancestry option MUST be a `DamageTypeOption` in this order (the two-column table read column by column): Black, Blue, Brass, Bronze, Copper, Gold, Green, Red, Silver, White, with `damageType` Acid for Black and Copper; Lightning for Blue and Bronze; Fire for Brass, Gold and Red; Poison for Green; Cold for Silver and White. An unknown damage type ("Sonic") is `invalidValue` at `.../options/<j>/damageType`; a `text` key is `unknownField`; one damage and one text option is the mixed-variant rule of FR-016. | SRD 5.2.1, FR-017 p. 84 | happy |
| FR-018 | Each Elven Lineage option MUST be a `LineageOption` with Level 1, 3 and 5 `levelBenefits` as in the Elven rows of "Elven Lineages and Fiendish Legacies": `levelBenefits[0].text` is the verbatim Level 1 cell and its `spellId` the cantrip; `[1]` and `[2]` hold the printed Level 3 and 5 cells as `text` and the spell they name as `spellId`. | SRD 5.2.1, "Elven Lineages", p. 85 | happy |
| FR-019 | Each Fiendish Legacy option MUST be a `LineageOption` with Level 1, 3, 5 `levelBenefits` as in the Fiendish rows of that table, held as in FR-018. | SRD 5.2.1, "Fiendish Legacies", p. 86 | happy |
| FR-020 | The Gnomish Lineage options MUST be exactly Forest Gnome and Rock Gnome (`TextOption`s) with these spell ids on the option: Forest Gnome `minor_illusion`, `speak_with_animals`; Rock Gnome `mending`, `prestidigitation`. The Giant Ancestry options MUST be exactly six `TextOption`s whose `name` is the printed heading, descriptor included, in printed order (Cloud’s Jaunt (Cloud Giant), Fire’s Burn (Fire Giant), Frost’s Chill (Frost Giant), Hill’s Tumble (Hill Giant), Stone’s Endurance (Stone Giant), Storm’s Thunder (Storm Giant)), each with the text printed after the heading and empty `spellIds`. There is no name/descriptor split (`Cloud Giant (Cloud’s Jaunt)` fails the golden comparison, FR-062). | SRD 5.2.1, "Gnome", p. 85; "Goliath", pp. 85-86 | happy |
| FR-021 | A spell or cantrip named inside a species choice option MUST be stored as a typed reference beside the verbatim text (`LevelBenefit.spellId` or `TextOption.spellIds`, a `SpellID` in a `Key` of kind `spell`); a damage type in a Draconic Ancestry option is a `DamageType`. A skill or condition named in species text (Keen Senses, Dwarven Resilience, Fey Ancestry) stays verbatim: `NamedText` carries no references and no rule needs them; `ConditionKind` and `Skill` are `content` vocabulary (section 9). | architecture.md §2, §3; shared-types-and-ownership.md FR-005; SRD 5.2.1, pp. 84-86 | happy |
| FR-022 | The spell ids referenced by the bundled species MUST be exactly these 20: `dancing_lights`, `faerie_fire`, `darkness`, `prestidigitation`, `detect_magic`, `misty_step`, `druidcraft`, `longstrider`, `pass_without_trace`, `minor_illusion`, `speak_with_animals`, `mending`, `poison_spray`, `ray_of_sickness`, `hold_person`, `chill_touch`, `false_life`, `ray_of_enfeeblement`, `fire_bolt`, `hellish_rebuke`. A data pin asserts the set. Thaumaturgy (Otherworldly Presence) and other spells named only in trait text stay text. | SRD 5.2.1, pp. 84-86 | happy |
| FR-023 | A spell reference in species data is a reference site (schema FR-126) registered in `R1-species-backgrounds-feats`: `(species, choices/<j>/options/<k>/levelBenefits/<m>/spellId, spell, none)` and `(species, choices/<j>/options/<k>/spellIds/<n>, spell, none)`. Every site is checked over the whole set (shared-types-and-ownership.md FR-054): a single-document decode never resolves it; `content.ValidateSet` (schema FR-125) reports `unresolvedReference` with the owning species key, the field and the missing spell key. `Species` implements the schema spec's references interface and lists exactly these sites. Consequence: `data.BundledSource` runs the whole-set validation over the merged embedded documents (schema FR-104), so a bundled set that holds `species.json` without the 20 spells of FR-022 fails that validation with `unresolvedReference`; that check, not a spec dependency, is what ties the species data to the spell data (shared-types-and-ownership.md FR-067), and this spec reads no spell entry. It adds no bespoke lookup pin. | shared-types-and-ownership.md FR-047, FR-054; schema FR-035, FR-104, FR-125, FR-126; testing.md §4 (missing content) | unhappy (missing content) |
| FR-060 | A `choices` element's `traitName` MUST equal the `name` of a trait of the same species, else (`/species/<i>/choices/<j>/traitName`, `constraintViolation`). The bundled links are Draconic Ancestry, Elven Lineage, Gnomish Lineage, Giant Ancestry and Fiendish Legacy; each is both a trait (the verbatim introduction) and a choice (its options). | SRD 5.2.1, pp. 84-86; SX-1 | unhappy (rule interaction) |
| FR-061 | A `LineageOption`'s `levelBenefits` MUST be exactly three entries with levels 1, 3, 5 in that order, each with `text` and `spellId`. A length other than three is `outOfRange` at `.../levelBenefits` (no element problems then); a `level` outside {1, 3, 5} is `invalidValue`; a repeated level is `duplicate` at the later index; distinct valid levels in another order are `constraintViolation` at `.../levelBenefits`; a missing `spellId` is `missingField`. The constructor returns an error matching `srderr.ErrInvalidInput` for each, so a Draconic option carrying levels is unrepresentable. | SRD 5.2.1, "Elven Lineages", p. 85; "Fiendish Legacies", p. 86 | unhappy (boundary) |
| FR-074 | Every bundled species MUST have a `sizeText` (SX-3) equal, after the normalisation of FR-045, to the Size sentence printed after "Size:" in the golden (FR-062) and the PDF (FR-044); its hash is in the manifest under key `sizeText` (FR-050). A mis-transcribed height range fails the golden comparison, the PDF check or the hash check, naming the species id and `sizeText`. The decoder accepts absence (SX-3); the data pin fails a bundled species without it. | SRD 5.2.1, "Species Descriptions", "Size", pp. 84-86 | unhappy (missing content) |

### 4.3 Backgrounds

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-024 | `abilityScores` MUST be exactly three distinct `Ability` values in printed order. Two or four is `outOfRange`; a repeat is `duplicate`, at `/backgrounds/<i>/abilityScores`. | SRD 5.2.1, "Ability Scores", p. 83 | unhappy (boundary) |
| FR-025 | A background's `feat.featId` (`FeatID` in a `BackgroundFeatReference`) MUST name a bundled feat of category `Origin`. It is the reference site `(background, feat/featId, feat, TargetHasFeatCategory(Origin))` (schema FR-126), listed by `Background` and registered in `R1-species-backgrounds-feats`, checked by the whole-set validation (schema FR-125): an id naming no feat is (`/backgrounds/<i>/feat/featId`, `unresolvedReference`); a feat of another category is (same path, `constraintViolation`). The constraint reads the target through `HasFeatCategory`, which `Feat` implements (`featCategory` = `category`). A single-document decode of backgrounds alone does not resolve it. | SRD 5.2.1, "Feat", p. 83; schema FR-035, FR-036, FR-126 | unhappy (rule interaction) |
| FR-026 | A feat qualifier MUST be stored in `feat.choice` as the printed word, separate from `featId`; a blank `choice` is `invalidValue`. Acolyte has "Cleric", Sage "Wizard"; Criminal and Soldier none. | SRD 5.2.1, "Acolyte", "Sage", p. 83 | happy |
| FR-027 | `skillProficiencies` MUST be exactly two distinct `Skill` values: one or three is `outOfRange`, a repeat `duplicate`, an unknown skill `invalidValue`, at `/backgrounds/<i>/skillProficiencies`. | SRD 5.2.1, "Skill Proficiencies", p. 83; "Skills", p. 9 | unhappy (boundary) |
| FR-028 | `toolProficiency` has the shape of closed `ToolProficiency` (schema FR-119: `{kind: "specific", name}` or `{kind: "choice", category}`; another `kind` is `invalidValue`, that spec's rule). Bundled values: Acolyte and Sage `specific` Calligrapher's Supplies; Criminal `specific` Thieves' Tools; Soldier `choice`, category `Gaming Set`. | SRD 5.2.1, "Tool Proficiency", "Soldier", p. 83; schema FR-119 | happy |
| FR-029 | `equipmentOptions` MUST have exactly the keys `A` and `B`. `A.items` lists the printed items and `A.gold` is as in "Background cases"; `B.items` is empty and `B.gold` is 50 for every background. A missing or extra key is `missingField` or `unknownField`. | SRD 5.2.1, "Equipment", p. 83 | happy |
| FR-030 | An equipment item is `{name, quantity}`. A leading numeral in the printed text is the `quantity`, not part of `name` (`20 Arrows` is "Arrows", 20); a parenthetical stays in `name` as printed ("Book (prayers)", "Parchment (10 sheets)", "Gaming Set (same as above)"); no numeral means quantity 1. A `quantity` below 1 is (`.../items/<k>/quantity`, `outOfRange`). | SRD 5.2.1, "Background Descriptions", p. 83 | unhappy (boundary) |
| FR-031 | The background entries MUST match "Background cases" field for field (abilities, feat, skills, tool, option A items and gold). | SRD 5.2.1, p. 83 | happy |
| FR-032 | The increases of "Ability Scores" (+2 and +1, or +1 to all three) MUST NOT be data on any background: `Background` has no member for them and a key such as `abilityIncreases` is `unknownField` (schema FR-022). The rule sentence is not stored as a constant, export, display string or golden: it is game content, and no distributed path other than the three covered by `NOTICE.md` may hold an SRD rule sentence. The `Background` doc comment cites SRD 5.2.1, "Ability Scores", p. 83 without quoting it. | SRD 5.2.1, "Ability Scores", p. 83; architecture.md §3; content-licensing-and-notice.md FR-005 | happy |

### 4.4 Feats

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-033 | A feat MUST have `category`, `benefits` (one or more) and `repeatable` (boolean); `prerequisites` is optional. No `benefits` is (`/feats/<i>/benefits`, `outOfRange`); a missing `repeatable` is `missingField`. `Feat` has no whole-description field: the description is its benefits, prerequisites and repeatable text. | SRD 5.2.1, "Parts of a Feat", p. 87 | unhappy (missing content) |
| FR-034 | The category of each bundled feat MUST be as in "Feat cases", and a feat has exactly one. The members and spellings of `FeatCategory` are schema FR-117; this pins only the bundled values. | SRD 5.2.1, pp. 87-88; schema FR-117 | happy |
| FR-035 | `prerequisites`, when present, is a non-empty list of the closed `Prerequisite` family of schema FR-118 (`level`, `abilityScore`, `feature`, any combination; clause shapes, ranges and rejections are that requirement's). A feat with none omits the field; an empty list is `outOfRange` at `/feats/<i>/prerequisites` (SX-4). | SRD 5.2.1, "Prerequisite", p. 87; schema FR-118 | unhappy (boundary) |
| FR-038 | The prerequisites of the bundled feats MUST be as in "Feat cases". Origin feats have none. Pinned: `level` 4 (Ability Score Improvement, Grappler), `level` 19 (every Epic Boon), `abilityScore` `[Strength, Dexterity]` minimum 13 (Grappler), `feature` "Fighting Style Feature" (the four Fighting Style feats), `feature` "Spellcasting Feature" (Boon of Spell Recall). | SRD 5.2.1, pp. 87-88 | happy |
| FR-039 | `repeatable` is `true` exactly for Magic Initiate, Skilled and Ability Score Improvement, `false` for the other 14. A repeatable feat MUST carry `repeatableText` (SX-2), the printed "Repeatable" sentence, verbatim; `repeatableText` with `repeatable: false` is `constraintViolation` at `/feats/<i>/repeatableText`. | SRD 5.2.1, "Repeatable", p. 87 | happy |
| FR-040 | A feat's named benefits (for example "Initiative Proficiency", "Two Cantrips", "Peerless Aim") MUST each be a `{name, text}` benefit with verbatim text, in printed order. A feat printed as one block has exactly one benefit with no `name`: Savage Attacker, Skilled, Archery, Defense, Great Weapon Fighting, Two-Weapon Fighting, Ability Score Improvement. | SRD 5.2.1, "Parts of a Feat", pp. 87-88 | happy |
| FR-041 | The named benefit names of the bundled feats MUST be, in order: Alert: Initiative Proficiency; Initiative Swap. Magic Initiate: Two Cantrips; Level 1 Spell; Spell Change. Grappler: Ability Score Increase; Punch and Grab; Attack Advantage; Fast Wrestler. Each Epic Boon starts with Ability Score Increase, then: Combat Prowess: Peerless Aim; Dimensional Travel: Blink Steps; Fate: Improve Fate; Irresistible Offense: Overcome Defenses, Overwhelming Strike; Spell Recall: Free Casting; the Night Spirit: Merge with Shadows, Shadowy Form; Truesight: Truesight. The unnamed-benefit feats of FR-040 have one benefit and no name. | SRD 5.2.1, pp. 87-88 | happy |
| FR-043 | Numeric limits quoted in feat text (Ability Score Improvement "above 20", Grappler "maximum of 20", the maximum of 30 of every Epic Boon ability increase) MUST appear in the verbatim text exactly as printed. | SRD 5.2.1, pp. 87-88 | happy |
| FR-063 | The decoder MUST reject a feat benefit with a blank `name` or `text` with `invalidValue` at `/feats/<i>/benefits/<k>/name` or `.../text` (schema FR-027, FR-028). A data pin MUST fail a bundled feat whose named benefits repeat a name. | SRD 5.2.1, "Parts of a Feat", p. 87; schema FR-027, FR-028 | unhappy (invalid input) |

### 4.5 Transcription and verification

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-044 | An on-demand PDF check (in `tools/`, run in Docker with `pdftotext` in reading-order mode, without `-layout`, which interleaves the two columns of pp. 84-88; given the path of the SRD PDF, which is not in the repository) MUST find, for every bundled entry, the printed name in the text of page `source.page`, and each `sizeText`, trait, choice option, benefit and repeatable text, after the normalisation of FR-045, in the text of the start page or the next (Elf and Goliath cross a page). It MUST report each entry id and field not found; one report line fails the change. A name, number or sentence not on pp. 83-88 MUST be removed and flagged in the change summary. | AGENTS.md, "Licensing and source material" | unhappy (missing content) |
| FR-045 | The comparison normalisation is a pure function in `tools/` from PDF text-layer output (or data text) to a comparison string. It MUST expand ligatures, remove soft hyphens and zero-width spaces (U+00AD, U+200B), collapse every whitespace run (including newlines inside a paragraph) to one space, and trim; it MUST keep curly quotes and apostrophes, case, punctuation and every hyphen. It does not decide hyphens: a line-end hyphen is resolved by the transcriber against the printed word and recorded in the manifest (FR-047, key `hyphenation`) as (printed form, resolved form), where the printed form is the string this function yields (`Versa- tile`, one space); the PDF check applies the recorded resolutions before comparing. | SRD 5.2.1, pp. 83-88; AGENTS.md ("keep it faithful to the source") | happy |
| FR-046 | The bundled data MUST be produced by the process in section 5 (read from the PDF, structured by hand, independently re-read), and each of the 30 entries MUST have a verification record in the manifest. | shared-types-and-ownership.md FR-030 to FR-032 | happy |
| FR-047 | The verification manifest is `doc/research/srd-species-backgrounds-feats-manifest.json`, outside the public packages, with the key set in section 5. It is to be created (not yet present); creating it MUST add its row to `doc/documentation_index.md` in the same change. For each of the 30 entries it MUST record `kind`, `id`, `name`, `page`, `transcribedBy`, `verifiedBy`, `verifiedOn` (ISO 8601), `hyphenation` and `hashes`. A missing record, blank role or date, or `transcribedBy` equal to `verifiedBy` MUST fail the completeness check. The check proves a record names two different roles, not that the verifier was independent: the re-read is a fresh agent session or person with no access to the transcription, and a human spot check follows an agent re-read (section 9). | testing.md §4 (missing content) | unhappy (missing content) |
| FR-048 | A completeness comparison, a pure function over the decoded set (or a fixture set) and the inventory of FR-001 to FR-003, MUST fail naming the id and kind of every missing and every extra entry. | testing.md §4 | unhappy (missing content) |
| FR-049 | A data pin MUST assert every row of the case tables against the decoded bundled entries, one test per row, so a transcription error in a structured field fails a named case. Expected SRD text is read from the goldens under `testdata/srd/` (FR-062); only ids, counts, names and numbers are inline. | testing.md §5; shared-types-and-ownership.md FR-033; content-licensing-and-notice.md FR-033 | happy |
| FR-050 | The manifest's `hashes` (SHA-256 of the normalised text of each `sizeText`, trait, option, benefit and repeatable text, keyed by field path) is a change detector only. The check MUST fail, naming entry id and field, when bundled text differs from its recorded hash, until a reviewer updates the manifest. A hash says nothing about whether the first transcription was right; that is the job of the goldens (FR-062) and the PDF check (FR-044). | testing.md §8 | unhappy (invalid input) |
| FR-051 | No bundled entry, manifest, fixture, comment or commit message may name a consuming project or the organisation hosting this repository, and no entry text may add attribution to Wizards or its parent or affiliates beyond the verbatim `NOTICE.md` statement. The consumer-name check is architecture-guardrails.md FR-025 and FR-026 (and release-0-1-0.md FR-072), whose Module `FS` includes the manifest and fixtures; this spec adds no scan. | AGENTS.md, "Hard constraints", "Trademarks and attribution"; architecture-guardrails.md FR-025, FR-026 | unhappy (invalid input) |
| FR-053 | The source-of-truth JSON MUST be **two** files under `content/data/`: `species.json` (`species` array) and `origins.json` (`backgrounds` and `feats` arrays, as schema FR-008 lets one document carry several). The background `feat.featId` is a whole-set reference site (FR-025), so no file is required to decode alone with zero reference problems (schema FR-089, FR-104). Embedded documents are one per data file (schema FR-120): `species.json`, `origins.json`. `species.json` supersedes the thin slice's species document; a data pin MUST fail if any species id occurs in two embedded documents. File placement and the `NOTICE.md` paths hold by schema FR-089, FR-104 and content-licensing-and-notice.md FR-005; the path-list equality check is shared-types-and-ownership.md FR-050. This spec adds only the two file names. | AGENTS.md, "The notice"; shared-types-and-ownership.md FR-030, FR-033, FR-050; schema FR-008, FR-035, FR-120 | happy |
| FR-062 | Each category MUST have a golden fixture transcribed independently of the data (by someone or something that did not see it): `testdata/srd/golden_species.json`, `golden_backgrounds.json`, `golden_feats.json` (SRD text in tests lives only under `testdata/srd/`, shared-types-and-ownership.md FR-033). Each holds per entry the id, name, start page and verbatim text of every `sizeText` (species), trait, option (each Level 1, 3 and 5 cell and each Giant Ancestry heading and text), benefit and repeatable text, and the lead-in sentences that are not stored (FR-070). A data pin MUST fail on any difference between golden and bundled data, comparing text after FR-045 and naming entry id and field. | testing.md §1, §8; AGENTS.md ("keep it faithful to the source") | unhappy (invalid input) |

### 4.6 Behaviour of the bundled source

`data.BundledSource` (exported from `content/data`, owned by
content-source-port.md; shared-types-and-ownership.md FR-031, FR-047) decodes
the embedded strings through the one `content.Decoder` and delegates `Lookup`
and `ListKind` to an `content.InMemorySource`; this spec adds only its two
embedded JSON documents (`species.json`, `origins.json`, FR-053). How it reports
embedded data that fails to decode is the owner's rule.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-054 | `Lookup` on the bundled source MUST return `LookupFound` for each of the 30 keys of FR-001 to FR-003 and `LookupNotFound` for a name not in the SRD and for a real id under the wrong kind (`Key(feat, sage)`); it never reports a decode problem. A data pin, not a contract suite: the contract registrations of `data.BundledSource` are content-source-port.md FR-047's and are neither restated nor changed. | content-source-port.md FR-009, FR-047; SRD 5.2.1, pp. 83-88 | unhappy (missing content) |
| FR-055 | `ListKind` for `species`, `background` and `feat` MUST return 9, 4 and 17 entries, ordered by `ID`, identical on every call (data pins; the generic ordering rule is the shared suite's). | content-source-port.md FR-014; testing.md §4 (determinism) | happy |
| FR-056 | A decoded entry MUST be immutable: its lists and sets are defensively copied (mutating a returned slice does not change the entry) and decoding the same text twice yields entries equal by value. | schema FR-071, FR-072, FR-076; architecture.md §3 | unhappy (immutability) |
| FR-057 | Decoding a malformed entry MUST report every problem found (path and kind), not only the first, in the field order of section 5 (schema FR-016): a species with a missing `speed`, a blank trait name and an unknown `creatureType` yields (`/species/0/creatureType`, `invalidValue`), (`/species/0/speed`, `missingField`), (`/species/0/traits/0/name`, `invalidValue`). | schema FR-012, FR-016; architecture.md §5 | unhappy (invalid input) |
| FR-058 | A key not declared for an entry object (for example `race`, `abilityIncreases`) MUST be (`/<array>/<i>/<key>`, `unknownField`); never silently dropped, never a warning. | schema FR-022, FR-074 | unhappy (invalid input) |
| FR-064 | Species, background and feat entries of the bundled source MUST come through exactly one decode path: the embedded strings go through the same `content.Decoder` as host-supplied JSON (schema FR-104). No adapter or bundled source reads files at run time, and this spec's data pins read the embedded documents, not the filesystem; architecture-level reads of the data files belong to schema FR-089, FR-104 and the decode test of FR-065. | shared-types-and-ownership.md FR-031; schema FR-089, FR-104 | happy |
| FR-065 | The embedded JSON documents holding the three kinds under `content/data/` ARE the source of truth: there is no generator and no second copy, so no drift can occur (shared-types-and-ownership.md FR-030, FR-032). Instead a test decodes every embedded file of these kinds through the real decoder with zero problems (architecture-guardrails.md FR-078). Verification tooling lives only in `tools/` and is never imported from the public packages. | shared-types-and-ownership.md FR-030, FR-032; architecture-guardrails.md FR-078 | unhappy (drift) |
| FR-066 | The shapes of section 5 and SX-1 to SX-4 MUST be carried by revision-table row `R1-species-backgrounds-feats` of content-schema-and-decoder.md (no other row adds them), whose description lists them and the reference sites of FR-023 and FR-025; they land with the other revisions as one `schemaVersion`. This spec claims no version number and no bundled document declares a different one; the decoder accepts exactly the current version. | shared-types-and-ownership.md FR-034, FR-035; schema FR-007, FR-103 | unhappy (version mismatch) |
| FR-067 | The `content` code and data of this spec MUST NOT import or name anything from `character`, `combat`, `d20` or `toolbox`. Ids, vocabulary, `Ability` and the building blocks are those named in section 1, owned by content-schema-and-decoder.md (FR-106 to FR-109, FR-114, FR-116 to FR-119), and none is redefined here. | shared-types-and-ownership.md FR-005, FR-013, FR-046, FR-048 | unhappy (forbidden edge) |
| FR-068 | The verification manifest and any `doc/research/` note holding SRD text are not distributed: content-licensing-and-notice.md FR-005, FR-033 and shared-types-and-ownership.md FR-033, FR-050 own this (the `NOTICE.md` path list and the `.pubignore` exclusion of `doc/`); no second check here. Data-specific consequence: the manifest holds hashes and short hyphenation forms only, and SRD text in tests lives only under `testdata/srd/`. | shared-types-and-ownership.md FR-033, FR-050; content-licensing-and-notice.md FR-005, FR-033 | happy |
| FR-069 | Every value-object constructor of section 5 MUST return an error matching `srderr.ErrInvalidInput` for each invalid case the case tables list (blank text, a size outside Small and Medium, a set of the wrong size, a duplicate, a number out of range, a lineage whose levels are not 1, 3, 5, an empty option list, mixed variants, and so on). Limits are public constants and predicates on the domain types, used by the decoder instead of repeating literals; the agreement test of schema FR-094 runs every row of this spec's case tables through both the decoder (the stated problem) and the constructor (an error matching `srderr.ErrInvalidInput` or acceptance). | schema FR-093, FR-094; architecture.md §6 | unhappy (invalid input) |
| FR-070 | The lead-in sentences ("As a Dwarf, you have these special traits." and its siblings, with the Tiefling variant "you have the following special traits"; "You gain the following benefits." wherever printed) are presentation and MUST NOT be stored in any `traits`, `options`, `levelBenefits` or `benefits` text. The golden lists them and the comparison fails if one appears in stored text. | schema anti-corruption rule 4; SRD 5.2.1, pp. 84-88 | unhappy (invalid input) |
| FR-071 | The types of section 5 (`Species`, `Background`, `Feat` and supporting types) MUST live in package `content`, be exported from package `content` with doc comment, and be defined nowhere else (the exported-name scan, shared-types-and-ownership.md FR-021). This spec's type is `BackgroundFeatReference`, not `BackgroundFeat`, which feats-behaviour.md exports as a `FeatSource` variant; the two names are distinct and need no change in those specs (section 9). The ids, vocabulary and building blocks of FR-067 MUST NOT be re-declared in any file of this spec's code. | shared-types-and-ownership.md FR-005, FR-046, FR-048 | unhappy (duplicate) |

### 4.7 Custom backgrounds ("Creating a Background")

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-072 | A custom background (one a host authors) is valid only if it satisfies the five steps of the SRD: (1) three abilities, (2) one feat of the Origin category, (3) two skill proficiencies, (4) one tool proficiency, (5) equipment worth 50 GP including unspent gold, with no Martial weapons or armor. Steps 1, 3 and 4 are the `Background` shape (FR-024, FR-027, FR-028), enforced by constructor and decoder. Step 2 needs the referenced feat's category, which a constructor cannot see, so the decoder and whole-set validation enforce it (FR-025; schema FR-036), not construction: a host-built `Background` does not enforce step 2 until validated as part of a set. No second code path re-checks any step. Step 5 is FR-073. | SRD 5.2.1, "Creating a Background", pp. 192-193; shared-types-and-ownership.md FR-045 | unhappy (rule interaction) |
| FR-073 | Step 5 is the pure function `CustomBackgroundEquipmentProblems(option EquipmentOption, valuation EquipmentValuation) []CustomBackgroundEquipmentProblem` in package `content`, returning a possibly empty slice of the closed `CustomBackgroundEquipmentProblem`: `EquipmentValueNotFiftyGold(copperPieces)` (sum of `quantity` times the item's `copperPieces`, plus `gold` times 100, is not 5000; 100 CP is 1 GP), `EquipmentItemUnpriced(itemName)` (valuation reported no price) and `EquipmentItemExcluded(itemName)` (valuation marks the item a Martial weapon or armor). **If any item is unpriced, the total check is skipped** and no `EquipmentValueNotFiftyGold` is reported; an unpriced item is not counted as 0 CP. An excluded item that is priced still counts toward the total. `EquipmentValuation` is a caller-supplied function `func(EquipmentItem) (EquipmentItemValue, bool)` whose `false` means "no price" and whose `EquipmentItemValue` carries `CopperPieces` and `IsMartialWeaponOrArmor`, so this spec imports nothing from content-equipment.md (shared-types-and-ownership.md FR-047). It applies to each of options A and B of a custom background, reports every problem in item order, then the total, and does not run on the four SRD entries. | SRD 5.2.1, "Creating a Background", p. 193; "Coins", p. 89; shared-types-and-ownership.md FR-047 | unhappy (boundary) |

### Cases: custom background equipment (one test per row)

Valuations are fixture functions; no equipment content is read.

| Label | Option | Expected | Path |
|---|---|---|---|
| 50 GP by gold alone | no items, gold 50 | no problems | happy |
| items plus gold | one item x2 at 1000 CP, gold 30; one item x3 at 1000 CP, gold 20 | no problems (2000 + 3000; 3000 + 2000) | happy |
| 49 and 51 GP | no items, gold 49; gold 51 | `EquipmentValueNotFiftyGold(4900)`; `(5100)` | unhappy (boundary) |
| unpriced item | one item, valuation reports no price, gold 50 | `EquipmentItemUnpriced(name)` only; total check skipped | unhappy (missing content) |
| unpriced and wrong total | one unpriced item, one priced at 1000 CP, gold 10 | `EquipmentItemUnpriced(name)` only (not counted as 0 CP) | unhappy (rule interaction) |
| Martial item | one flagged item, total 50 GP | `EquipmentItemExcluded(name)` only | unhappy (rule interaction) |
| Martial item and wrong total | one flagged item at 1000 CP, gold 10 | `EquipmentItemExcluded(name)` then `EquipmentValueNotFiftyGold(2000)`: an excluded item still counts | unhappy (rule interaction) |
| zero-cost item | item at 0 CP, gold 50 | no problems | unhappy (boundary) |
| negative cost | `EquipmentItemValue{CopperPieces: -1}` | constructor returns an error matching `srderr.ErrInvalidInput` | unhappy (invalid input) |
| steps 1 to 4 | two abilities, a General feat, one skill or no tool | rejected by FR-024, FR-025, FR-027, FR-028, not by this function (FR-072) | unhappy (rule interaction) |

All rows cite FR-073 (the last FR-072).

### Cases: species header (one test per row)

| Label | Expected | Source | Path |
|---|---|---|---|
| bundled sizes and speed | each of the nine equals FR-012 (sizes) and FR-013 (30 ft; Goliath 35); Creature Type Humanoid (FR-011) | FR-011 to FR-013; pp. 84-86 | happy |
| decoder: sizes | `[]`, three sizes, a repeat, `[Large]`: the problems of FR-012; constructor returns an error matching `srderr.ErrInvalidInput` | FR-012 | unhappy (boundary) |
| decoder: speed | 1 and 30 accepted; 0 accepted by the decoder, fails the data pin; -5 and 30.5: schema FR-021 | FR-013 | unhappy (boundary) |
| bundled: `sizeText` present | equals the golden Size sentence after normalisation (height ranges are verbatim text, not numeric) | FR-074 | happy |
| bundled: `sizeText` altered | one changed character fails the golden comparison naming species id and `sizeText`, and the hash check | FR-074, FR-050, FR-062 | unhappy (invalid input) |
| bundled: `sizeText` absent | decoder accepts (SX-3); the data pin fails the fixture | FR-074 | unhappy (missing content) |

### Cases: Elven Lineages and Fiendish Legacies (one test per row)

Each is a `LineageOption`; the Level 1 text is read from the golden (FR-049).
`[1]` and `[2]` are the printed Level 3 and 5 spell names with their `spellId`.
The PDF text layer has a zero-width space after "feet." in the Drow cell; FR-045
removes it.

| Label | Option | Level 1 cantrip id | Level 3 id | Level 5 id | Source | Path |
|---|---|---|---|---|---|---|
| drow | Drow | `dancing_lights` | `faerie_fire` | `darkness` | FR-018; p. 85 | happy |
| high elf | High Elf | `prestidigitation` | `detect_magic` | `misty_step` | FR-018; p. 85 | happy |
| wood elf | Wood Elf | `druidcraft` | `longstrider` | `pass_without_trace` | FR-018; p. 85 | happy |
| abyssal | Abyssal | `poison_spray` | `ray_of_sickness` | `hold_person` | FR-019; p. 86 | happy |
| chthonic | Chthonic | `chill_touch` | `false_life` | `ray_of_enfeeblement` | FR-019; p. 86 | happy |
| infernal | Infernal | `fire_bolt` | `hellish_rebuke` | `darkness` | FR-019; p. 86 | happy |
| level errors | levels 1, 3 only; 1, 3, 3; a level 2; 1, 5, 3 | `outOfRange`; `duplicate` at `.../levelBenefits/2/level`; `invalidValue`; `constraintViolation`; each an error matching `srderr.ErrInvalidInput` (FR-061) | | | FR-061 | unhappy (boundary) |
| missing parts | no `levelBenefits`; a `text` key; a benefit with no `spellId` | `missingField`; `unknownField`; `missingField` (FR-016, FR-061) | | | FR-016, FR-061 | unhappy (missing content) |

### Cases: Background cases (one test per row)

| Label | Background | Abilities | `feat` | Skills | Tool | Option A items (name, quantity) | A GP | B GP | Source | Path |
|---|---|---|---|---|---|---|---|---|---|---|
| acolyte | Acolyte | Intelligence, Wisdom, Charisma | `magic_initiate`, choice Cleric | Insight, Religion | specific: Calligrapher's Supplies | Calligrapher's Supplies x1, Book (prayers) x1, Holy Symbol x1, Parchment (10 sheets) x1, Robe x1 | 8 | 50 | FR-024 to FR-031; p. 83 | happy |
| criminal | Criminal | Dexterity, Constitution, Intelligence | `alert` | Sleight of Hand, Stealth | specific: Thieves' Tools | Daggers x2, Thieves' Tools x1, Crowbar x1, Pouches x2, Traveler's Clothes x1 | 16 | 50 | FR-024 to FR-031; p. 83 | happy |
| sage | Sage | Constitution, Intelligence, Wisdom | `magic_initiate`, choice Wizard | Arcana, History | specific: Calligrapher's Supplies | Quarterstaff x1, Calligrapher's Supplies x1, Book (history) x1, Parchment (8 sheets) x1, Robe x1 | 8 | 50 | FR-024 to FR-031; p. 83 | happy |
| soldier | Soldier | Strength, Dexterity, Constitution | `savage_attacker` | Athletics, Intimidation | choice: category Gaming Set | Spear x1, Shortbow x1, Arrows x20, Gaming Set (same as above) x1, Healer's Kit x1, Quiver x1, Traveler's Clothes x1 | 14 | 50 | FR-024 to FR-031; p. 83 | happy |
| abilities and skills | two or four abilities; Strength twice; one or three skills; Stealth twice | | | | | | | | `outOfRange`; `duplicate` at `.../abilityScores/1`; `outOfRange`; `duplicate` at `.../skillProficiencies/1` (FR-024, FR-027) | unhappy (boundary) |
| unresolved feat | any | | `featId` `no_such_feat` | | | | | | (`.../feat/featId`, `unresolvedReference`) | unhappy (missing content) |
| non-origin feat | any | | `featId` `grappler` (General) | | | | | | (`.../feat/featId`, `constraintViolation`) | unhappy (rule interaction) |
| item quantity | any | | | | | Daggers x0; Daggers x1 | | | (`.../items/0/quantity`, `outOfRange`); accepted | unhappy (boundary) |
| missing option B | any | | | | | `equipmentOptions` with only `A` | | | (`.../equipmentOptions/B`, `missingField`) | unhappy (missing content) |
| increase field | any | `abilityIncreases` key | | | | | | | (`.../abilityIncreases`, `unknownField`) | unhappy (rule interaction) |

### Cases: Feat cases (one test per row)

Prerequisite notation: "level 4" is `{kind: level, minimum: 4}`, "STR or DEX 13"
is `{kind: abilityScore, abilities: [Strength, Dexterity], minimum: 13}`,
"feature X" is `{kind: feature, name: X}`. A row naming several feats is one test
per feat. Clause ranges and rejections are schema FR-118's, not repeated here.

| Label | Feat | Category | Prerequisites | Repeatable | Source | Path |
|---|---|---|---|---|---|---|
| origin, not repeatable | Alert, Savage Attacker | Origin | none | no | FR-034, FR-038, FR-039; p. 87 | happy |
| origin, repeatable | Magic Initiate, Skilled | Origin | none | yes | p. 87 | happy |
| ability score improvement | Ability Score Improvement | General | level 4 | yes | p. 87 | happy |
| grappler | Grappler | General | level 4; STR or DEX 13 | no | p. 87 | happy |
| fighting style | Archery (p. 87), Defense, Great Weapon Fighting, Two-Weapon Fighting (p. 88) | Fighting Style | feature "Fighting Style Feature" | no | pp. 87-88 | happy |
| epic boon | Boon of Combat Prowess, Boon of Dimensional Travel, Boon of Fate, Boon of Irresistible Offense, Boon of the Night Spirit, Boon of Truesight | Epic Boon | level 19 | no | p. 88 | happy |
| boon of spell recall | Boon of Spell Recall | Epic Boon | level 19; feature "Spellcasting Feature" | no | p. 88 | happy |
| empty prerequisites | any | | `prerequisites: []`: (`/feats/0/prerequisites`, `outOfRange`) | | FR-035 (SX-4) | unhappy (invalid input) |
| repeatable text without repeatable | any | | `repeatable: false` with `repeatableText`: (`/feats/0/repeatableText`, `constraintViolation`) | | FR-039 (SX-2) | unhappy (rule interaction) |
| repeatable without text | any | | `repeatable: true`, no `repeatableText`: (`/feats/0/repeatableText`, `missingField`) | | FR-039 (SX-2) | unhappy (missing content) |
| no benefits | any | | `benefits: []`: (`/feats/0/benefits`, `outOfRange`) | | FR-033 | unhappy (empty case) |

### Cases: id derivation and normalisation (one test per row)

Id rows cite FR-004 (the last also FR-059); normalisation rows cite FR-045.

| Label | Input | Expected | Path |
|---|---|---|---|
| names | `Dwarf`; `Great Weapon Fighting`; `Two-Weapon Fighting`; `Boon of the Night Spirit` | `dwarf`; `great_weapon_fighting`; `two_weapon_fighting`; `boon_of_the_night_spirit`; the same name twice gives equal ids | happy |
| punctuation | `a - b`; `  Elf! `; `Dragon's Breath`; `Dragon’s Breath` (U+2019) | `a_b`; `elf`; `dragons_breath`; `dragons_breath` | unhappy (boundary) |
| all 30 | each of the 30 SRD names | the id of FR-059, accepted by the id rule | happy |
| ligature, soft hyphen | `ﬁre`; `Dar­kvision` | `fire`; `Darkvision` | happy |
| zero-width space | `120 feet.​ You` (Drow cell, p. 85) | `120 feet. You` | happy |
| whitespace | `ability\nscore`; `a   b\n\n c` | `ability score`; `a b c` | happy |
| kept characters | `Calligrapher’s`; `Two-Handed` (p. 88) | unchanged | happy |
| line-end hyphen | `Versa-\ntile` (p. 88); `ignite-extin-\nguish` (p. 85) | `Versa- tile`; `ignite-extin- guish`; resolved only through a manifest `hyphenation` entry whose `printed` is that string | unhappy (rule interaction) |
| hyphen without manifest entry | `Pierc-\ning` (p. 88) | the PDF check reports the field not found until `Pierc-ing` to `Piercing` is recorded (FR-044, FR-047) | unhappy (missing content) |

### Withdrawn ids

| Id | Status |
|---|---|
| FR-036 | Withdrawn: moved to content-schema-and-decoder.md (FR-118): the `level` clause range; bundled levels are FR-038 |
| FR-037 | Withdrawn: moved to content-schema-and-decoder.md (FR-118): the `abilityScore` clause set and range; Grappler's clause is FR-038 |
| FR-042 | Withdrawn: dropped, feat choices stay verbatim text covered by FR-062 (section 9, feat choices) |
| FR-052 | Withdrawn: moved to content-licensing-and-notice.md (FR-016): the trademark identifier scan |

## 5. Model

### Entry types

The value types below live in package `content` (exported, section 6),
immutable (unexported fields, value receivers, no setters), compared by value (`Equal` where a type holds a slice), with defensively copied collections. Members are
listed in their JSON names; in Go each is an exported accessor method (`Species.ID()`), and a member marked `?` is optional and its
accessor returns a second `ok` result or a documented zero value. Each type has a validating constructor `NewX` (and `MustX` for constants); every `NewX`
returns an error matching `srderr.ErrInvalidInput` for exactly the invalid cases the case tables list
(FR-069); limits are public constants and predicates the decoder reuses (schema
FR-093, FR-094). Used, not owned: the types listed in section 1, `Entry`,
`Key`, `Kind`, `ID` (content-source-port.md) and the common
`source` object (schema FR-115).

| Type | Members | Invariants |
|---|---|---|
| `Species` (a `Entry`, kind `species`; implements the schema spec's references interface, schema FR-125) | `id`, `name`, `source?`, `creatureType`, `sizes`, `sizeText?`, `speed`, `traits` (`[]NamedText`), `choices` (`[]SpeciesChoice`, possibly empty), `references` (derived: the sites of FR-023) | FR-012 to FR-014, FR-060 |
| `SpeciesChoice` | `traitName`, `options` (`[]ChoiceOption`) | FR-016 |
| `ChoiceOption` (closed) | `name` | variants below |
| `DamageTypeOption` | `name`, `damageType` | FR-017 |
| `LineageOption` | `name`, `levelBenefits` (three `LevelBenefit`) | FR-061 |
| `TextOption` | `name`, `text`, `spellIds` (`[]SpellID`, possibly empty) | FR-020 |
| `LevelBenefit` | `level`, `text`, `spellId` | FR-061 |
| `Background` (`Entry`, kind `background`; implements the schema spec's references interface) | `id`, `name`, `source?`, `abilityScores`, `feat` (`BackgroundFeatReference`), `skillProficiencies`, `toolProficiency`, `equipmentOptions`, `references` (derived: the site of FR-025) | FR-024 to FR-030, FR-032 |
| `BackgroundFeatReference` | `featId`, `choice?` | FR-026; the Origin rule is FR-025 |
| `EquipmentOptions` | `a`, `b` (JSON keys `A`, `B`) | FR-029 |
| `EquipmentOption` | `items`, `gold` | `gold` at least 0 |
| `EquipmentItem` | `name`, `quantity` | FR-030 |
| `Feat` (`Entry`, kind `feat`; implements `HasFeatCategory`, `featCategory` = `category`, schema FR-126) | `id`, `name`, `source?`, `category`, `prerequisites`, `benefits` (`[]FeatBenefit`), `repeatable`, `repeatableText?` | FR-033 to FR-040 |
| `FeatBenefit` | `name?`, `text` | FR-063 |
| `EquipmentItemValue` | `copperPieces`, `isMartialWeaponOrArmor` | `copperPieces` at least 0 (FR-073) |
| `CustomBackgroundEquipmentProblem` (closed) | `EquipmentValueNotFiftyGold(copperPieces)`, `EquipmentItemUnpriced(itemName)`, `EquipmentItemExcluded(itemName)` | returned by the pure `CustomBackgroundEquipmentProblems` (FR-073) |

**Field order (JSON)** is the declaration order schema FR-016 uses to order
problems: `id`, `name`, `source?`, then the members in the order of the table
above (`sizeText?` after `sizes`, `choices?` last; `repeatableText?` last).
Absent required fields are `missingField` (schema FR-019); undeclared keys are
`unknownField` (schema FR-022).

There is no aggregate, event, use case or stateful domain service (the one
function of FR-073 is pure; shared-types-and-ownership.md FR-044 does not
arise). The cross-entry checks are the schema spec's (`Set`, whole-set
validation, schema FR-035, FR-036, FR-125) plus the data pins of section 4.

### Schema extensions

The shapes of this section and SX-1 to SX-4 are carried by revision-table row
`R1-species-backgrounds-feats` (FR-066), marked landed in the same change as this
data; its reference sites are the three of FR-023 and FR-025 (schema FR-126), and
`Feat` implements `HasFeatCategory`, which closes schema Q19 on this side. The
schema owner is asked to narrow `R1-thin-slice` so that, for these kinds, it
points at SX-1 and marks structured trait facts, level-gated threshold lists,
lineage spell-grant semantics and lineage overrides as declined (below); and to
correct row R1 ("keep three files" is two, FR-053; this spec has SX-1 to SX-4,
not SX-5). This spec claims no `schemaVersion` number
(shared-types-and-ownership.md FR-034).

| Id | Extension | Decoder rules |
|---|---|---|
| SX-1 | Species field `choices?`: a list of `{traitName, options}`. Each option has a `kind` of `damageType`, `lineage` or `text`, then `name`, then the variant's fields: `damageType` (`damageType`); `lineage` (`levelBenefits`, a list of `{level, text, spellId}`); `text` (`text`, optional `spellIds`). The variants are the closed `ChoiceOption` family. | The rules of FR-016, FR-017, FR-060, FR-061; `spellId`/`spellIds` match schema FR-030, decode to `SpellID` in `Key(spell, id)` and are resolved by the whole-set validation (FR-023). Field order: `traitName`, `options`; `kind`, `name`, then `damageType` or `levelBenefits` or `text`, `spellIds`; `level`, `text`, `spellId`. |
| SX-2 | Feat field `repeatableText?`: a non-blank string (schema FR-028) | Required when `repeatable` is `true` (`missingField`), forbidden when `false` (`constraintViolation`). |
| SX-3 | Species field `sizeText?`: a non-blank string, the printed Size sentence after "Size:" | Stored verbatim (schema FR-029); FR-074 requires it on every bundled species, equal to the golden's Size sentence, covered by the PDF check (FR-044) and manifest hashes (FR-050). |
| SX-4 | Feat `prerequisites`, when present, MUST be non-empty | Empty list is `outOfRange` at `prerequisites` (schema FR-024). |

SX-1 to SX-4 add no `ProblemKind` and need no `Set` change.

**Declined requests from the thin slice** (content-thin-slice.md section 5.1; its
owner changes that spec). **Structured trait facts** (a sense with a range, a
Resistance, a resource and so on): declined, because a structured fact beside its
text is a second copy that can drift; revisit with a new SX row when a behaviour
spec names a fact it cannot get from text. **Level-gated tables** (Breath Weapon
thresholds, Draconic Flight): declined, they stay in the verbatim text; only the
three-level lineage table is accepted (SX-1). **Choices with option sets**:
accepted for the five species choices only; Keen Senses, Skillful, Versatile and
a feat's choices stay text, class-feature choices belong to content-classes.md.
**Lineage spell grants**: the typed `SpellID` per level is accepted; grant
semantics and the spellcasting-ability choice are declined as data. **Lineage
overrides** (Drow Darkvision 120, Wood Elf Speed 35): declined; applying them is
character behaviour.

**Failures.** Every problem with a bundled or consumer document is a
`Problem` in the decode-failure variant of `LoadSetResult`
(shared-types-and-ownership.md FR-017, FR-018). An error from a `NewX` constructor is for a
programmer building an invalid instance. Nothing here returns a `*srderr.PortError`.


### Data format and process

Two JSON files under `content/data/` (FR-053) are the source of truth and are embedded directly with `go:embed`
(no generator, no second copy, schema FR-120); a test decodes every embedded file and the bundled
source reads those documents through the one decoder (shared-types-and-ownership.md
FR-030 to FR-032).

**Process (in order).** (1) **Read** each page from the PDF as images and, where
it helps, from the text layer in reading order (`pdftotext` without `-layout`,
in Docker), recording the printed page. (2) **Structure by hand** the structured
fields and enter every trait, benefit and option as verbatim text. (3)
**Script-assist only mechanical steps** (raw-text extraction, id derivation
FR-004); a script MUST NOT infer or fill a field the PDF does not state. (4)
**Golden:** a second, independent transcription (FR-062). (5) **Independent
re-read** by a different person or agent, recorded in the manifest (FR-046,
FR-047). (6) **Test:** data tests on every change; the PDF check on demand and
before each release (FR-044). Helpers (FR-004, FR-044, FR-045, FR-047, FR-048)
live in `tools/`, are tested in `tools` tests, are not API and are never imported
by the public packages.

**Verification manifest.** `doc/research/srd-species-backgrounds-feats-manifest.json`,
outside the public packages, not bundled, not decoded by the schema's data test, not shipped. To be created; its row goes into `doc/documentation_index.md` in the same change.
Shape: `{"manifestVersion": 1, "entries": [...]}`; each entry has `kind`, `id`,
`name`, `page`, `transcribedBy`, `verifiedBy`, `verifiedOn`, `hyphenation` (list of
`{page, printed, resolved}`), `notes` (free text, for example a PDF typo kept as
printed) and `hashes` (field path to hex SHA-256, for example `benefits/0/text`; a
species has `sizeText` beside its `traits/<n>/text`, option and level-benefit
paths). Roles are role names, not people.

**Fidelity rules.** Text is stored verbatim after the normalisation of FR-045;
nothing is paraphrased, corrected or "modernised" (architecture.md §3, §6).
Structured fields are redundant, not a replacement: where a fact is both
structured and in the text (speed, prerequisite, Gold Piece amounts) the case
tables and goldens assert they agree. Lead-in sentences are not stored (FR-070).

## 6. Interface

- **Ports and types.** No new port and no use case (shared-types-and-ownership.md
  FR-044); FR-073 takes its valuation from the caller. The section 5 types are
  synchronous `content` value objects served through `content.Source`.
- **Loading.** `data.BundledSource` (`content/data`) decodes the embedded
  strings through `content.Decoder` (FR-064), reads no file and uses no
  `os`; a decode failure cannot surface through the port. A malformed bundled
  document is caught before release by the data pins and the decode test (FR-065).
- **Exports** (package `content`, the public packages): the types of
  section 5 except those owned elsewhere, plus `EquipmentValuation` and
  `CustomBackgroundEquipmentProblems`; no constant or text of SRD rules (FR-032).
  `content/data`: `data.BundledSource`. `content/contenttest`: no new double.
  Every exported member has doc comment; the embedded JSON documents, goldens and
  manifest are not API.


## 7. Observability

**Silent by design** (observability.md §3): this spec adds data and value types,
no signal and no entry to `doc/observability/signals.md`. Loading and counting
use the shared `LoadSet` signals (schema FR-079 to FR-085;
shared-types-and-ownership.md FR-041), which stay correct (schema FR-100, FR-101);
with this data loaded alone `srd5e.content.entries_loaded` reads 9, 4 and 17 for
`species`, `background` and `feat`. A mis-transcription, a missing or extra entry
or a dangling spell reference is caught at build time by the data tests, goldens,
hash check and decode data test (schema FR-089) and by the PDF check before
release; a consumer sees `srd5e.content.loads{srd5e.outcome=failure}`. A
non-structural transcription error is invisible at run time, so the verification
process is the control. There is no content-owned lookup counter
(content-source-port.md Q11).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none defined by this spec | n/a | n/a | n/a | n/a |

## 8. Test plan

Levels (testing.md §3): `unit`, `adapter` (decoder rules driven with JSON text),
`contract`, `architecture`, `scenario`. A **data pin** is a unit-style test with
no doubles in `content/data/` that decodes the embedded strings through the
real `content.Decoder` and asserts printed values. A **tool unit** test covers
a pure helper in `tools/`, in `tools` tests (an addition to the testing.md §10
layout, recorded by this spec's change). The only real-file reads are in one
architecture test, `internal/archtest/species_backgrounds_feats_data_test.go`
(manifest and `content/data/` via `os`, the exception of schema FR-104).
A decoder-rule `unit` row runs twice (constructor and decoder), tied by the
agreement test (FR-069). Case-table tests are one test per row, with SRD text
from the goldens (FR-049). Doubles: `contenttest.FakeSource`,
a fake `content.Decoder`, `content.InMemorySource` only.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-003, FR-011, FR-015, FR-022 | data pin | decoded ids equal the nine species, four backgrounds, 17 feats (per category 4/2/4/7); every species Humanoid; trait names of FR-015; the referenced spell ids equal the 20 (`content/data/species_test.go`) |
| FR-004, FR-059 | tool unit | "id derivation" rows against the `tools/` helper; each of the 30 names derives its id and passes the id rule |
| FR-005 to FR-008, FR-010, FR-014 | unit | duplicate id in a kind and the same id under two kinds; `source` absent accepted by the decoder, blank section `invalidValue`, page 0 and -1 `outOfRange`; one test per missing required species field; blank trait name or text, duplicate trait name, order preserved |
| FR-007, FR-009 | data pin | every entry has `source` with document `SRD 5.2.1` and non-blank section; `source.page` equals the start page; boundary rows backgrounds 82/83/84, species 83/84/86/87, feats 86/87/88/89 (rejected by the check helper on fixtures) |
| FR-012, FR-013 | unit, adapter, data pin | "Species header" rows (constructor, decoder, bundled) |
| FR-016 to FR-020 | unit, adapter, data pin | option counts 10/3/2/6/3; the three variants; empty options, duplicate option name, mixed variants and a wrong-variant key rejected; the Draconic, Elven/Fiendish, Gnomish and Giant Ancestry options |
| FR-021, FR-023 | unit, architecture | `LevelBenefit.spellId` and `TextOption.spellIds` decode to `SpellID` in `Key(spell, id)`, text unchanged. `Species.references` over a synthetic species lists exactly one spell site per `spellId` and `spellIds` element; `content.ValidateSet` over a species set and a spells set lacking one spell gives `unresolvedReference` naming species key, field and spell key, over a complete set `ValidateSet result`; a single-document decode of the species alone reports none. No bespoke lookup pin. Whole-set: the merged bundled set (schema FR-104) passes `content.ValidateSet`, which resolves the 20 spell ids of FR-022 against the spell entries; this spec imports no spell type |
| FR-024, FR-027, FR-030 | unit | "Background cases" rows: abilities, skills, quantity 0 and 1, parenthetical kept in `name`, `20 Arrows` is "Arrows" x 20 |
| FR-025 | unit, adapter | unresolved and non-Origin feat rows, reported by `content.ValidateSet` over backgrounds and feats (not by a single-document decode); `Background.references` lists the `feat/featId` site with `TargetHasFeatCategory(Origin)`; `content.ValidateSet` reports `constraintViolation` for a General feat because `Feat` implements `HasFeatCategory` |
| FR-026, FR-028, FR-029, FR-031 | data pin, unit | Acolyte Cleric, Sage Wizard, blank `choice` `invalidValue`; tool proficiency per background including the Gaming Set choice (shape tested by schema FR-119); option A gold, B 50 with no items, missing B row; "Background cases" table |
| FR-032 | unit, adapter | `abilityIncreases` is `unknownField`; no constant, export or golden carries the rule sentence |
| FR-033, FR-035 | unit | no benefits; missing `repeatable`; optional `prerequisites`; empty list `outOfRange` (clause shapes and ranges are tested by schema FR-118) |
| FR-034, FR-038 to FR-041, FR-043 | data pin, unit | "Feat cases" category and prerequisites columns (`FeatCategory` is tested by schema FR-117); exactly three repeatable feats and `repeatableText` rows; named and unnamed benefits; benefit names of FR-041; "above 20", "maximum of 20" and "maximum of 30" in the texts as printed |
| FR-044 | on demand (outside the default suite) | Docker `pdftotext` in reading-order mode over the PDF path; a fixture page text proves it reports a missing name, a missing text and a wrong `sizeText`; run before release and when data changes |
| FR-045 to FR-048 | tool unit, architecture | normalisation rows including the real hyphen examples; completeness comparison on a fixture set with a missing and an extra entry names the id; record-completeness over fixture manifests with a missing record, blank role, blank date or equal roles fails naming the id. Architecture: the one `os` test requires a record for each of the 30 entries in the real manifest |
| FR-049, FR-050, FR-062, FR-070, FR-074 | data pin | all case-table tests as a group with text from the goldens; one changed character of a trait, option, benefit, repeatable text or `sizeText` fails with entry id and field, a normalisation-only change passes; golden comparison per category on normalised text; the golden lists the lead-in sentences and none appears in stored text; the altered and absent `sizeText` rows |
| FR-051 | architecture | delegated: the scans of architecture-guardrails.md FR-025, FR-026 run over the manifest and fixtures; no scan or denylist here |
| FR-053 | data pin | data pins read `species` and `origins` through the embedded index; no species id in two embedded documents; `origins` holds both arrays and decodes alone with zero `unresolvedReference` for the four `featId`s |
| FR-054, FR-055 | data pin | each of the 30 keys is `LookupFound`; a wrong-kind and an unknown key are `LookupNotFound`; `ListKind` counts 9/4/17, ordered by `ID`, repeat equal |
| FR-056 to FR-058, FR-060, FR-061, FR-063 | unit, data pin | returned slices are copies (mutating one leaves the entry unchanged), double decode equal by value; a species with three independent problems yields three in order; unknown key `unknownField`; link rows; level rows, bundled options have exactly levels 1, 3, 5; blank benefit name and text rows; a duplicate benefit name fixture fails the check |
| FR-064, FR-065 | data pin, architecture | data pins decode the embedded strings through the real decoder and the bundled source returns the same entries; no `os` in `content/data` or `content/data/`. The embedded documents are the source of truth (shared-types-and-ownership.md FR-032); here only that the three kinds decode through the real decoder |
| FR-066 | unit, doc review | decoder rejects any `schemaVersion` other than the current (schema FR-007); doc review: the revision table has `R1-species-backgrounds-feats` listing the shapes, SX-1 to SX-4 and the sites of FR-023 and FR-025, no version number, no other row adding these shapes, and `R1-thin-slice` pointing at SX-1 |
| FR-067, FR-071 | architecture | import-direction scan over `content` (no `character`, `combat`, `d20`, `toolbox`); exported-name scan: the types are exported from package `content` with doc comments, defined once; `BackgroundFeatReference` and `BackgroundFeat` are different names; no FR-067 owner type is declared here |
| FR-068 | tool unit | holds by content-licensing-and-notice.md FR-005, FR-033 and shared-types-and-ownership.md FR-033, FR-050; a manifest-shape test in `tools` tests asserts hashes and short hyphenation forms only, no SRD sentence |
| FR-069 | unit, adapter | the agreement test of schema FR-094 over this spec's case tables: each invalid row gives the stated problem and an error matching `srderr.ErrInvalidInput`, each valid row is accepted by both; a lint-style test asserts the decoder uses the domain constants |
| FR-072, FR-073 | unit | steps 1, 3, 4 are the FR-024, FR-027, FR-028 rows; step 2 is the FR-025 row (decoder and whole-set validation, not the constructor); "Cases: custom background equipment", one test per row; mutation tested (domain); doc review that the rule is stated here and in no toolbox spec |

- **Mutation testing** (testing.md §8): the package `content` constructors and
  predicates score at least 80; decoder rules and `tools/` helpers SHOULD. Data
  files are not code: the Testing entry records one manual mutation of a data
  value and the row or golden entry that caught it. **Not automated:** whether a
  first transcription is correct; only the goldens, the PDF check and the
  re-read show it.

- **Scenario:** one test through the public packages and
  `content/data` looks up a species, a background and a feat from
  `data.BundledSource` and checks one fact in each (a background's origin feat
  is an `Origin` feat). It runs against a bundled set that passes the whole-set validation (FR-023).
- Every live FR id appears above. Tests in other specs made false by this data
  (a thin-slice pin of four species) are updated by their owners.


**Implementation status.**

- Done (Phase 2, step 5a, package `content`, files `species.go`, `species_choice.go`, `background.go`, `feat.go`, `custom_background.go`, `decode_species.go`, `decode_origins.go`): the entry types `Species`, `Background`, `Feat` and their supporting types (`SpeciesChoice`, the closed `ChoiceOption` with `DamageTypeOption`, `LineageOption`, `TextOption`, `LevelBenefit`, `BackgroundFeatReference`, `EquipmentOptions`, `EquipmentOption`, `EquipmentItem`, `FeatBenefit`), their constructors and `MustX` forms (FR-069), the decoder rows registered as `species`, `backgrounds` and `feats` (FR-005, FR-006, FR-008, FR-010, FR-012 to FR-014, FR-016, FR-017, FR-021, FR-024 to FR-030, FR-032, FR-033, FR-035, FR-039, FR-056 to FR-058, FR-060, FR-061, FR-063), the reference sites of FR-023 and FR-025 (`Species.References`, `Background.References`, `Feat` implements `HasFeatCategory`), SX-1 to SX-4 (FR-066), the custom-background check (FR-072, FR-073: `CustomBackgroundEquipmentProblems`, `EquipmentValuation`, `EquipmentItemValue` and the closed `CustomBackgroundEquipmentProblem`), and the agreement rows of FR-069. FR-067 and FR-071 hold through the existing import-direction and exported-name scans.
- Choices the spec left open: an ability is spelled on the wire as the kernel spells it (`ability.Ability.String()`, lowercase, for example `"strength"`), exactly and case sensitively; a repeated trait name or option name is `duplicate` at the later element's `name` (not at the element); the problems of FR-016 for an option are reported at the option's own path (`./options/<k>/...`), not at `options`; when a species' `traits` has a problem the `traitName` link of FR-060 is not checked, so one bad trait does not cascade; the limits the SRD does not state are `SpeciesMaxSpeed`, `MaxEquipmentQuantity` and `MaxEquipmentGold` (2^31 - 1, so a value is the same on every platform integer width) and the saturation of the custom-background total at 2^61 Copper Pieces; `Feat` exposes its category as `FeatCategory()` only (the method of `HasFeatCategory`); `Species.SizeText`, `BackgroundFeatReference.Choice`, `FeatBenefit.Name` and `Feat.RepeatableText` return "" when absent, and a `Feat` built with a nil `Prerequisites` has none while a non-nil empty one is rejected (SX-4); the slice-holding types (`Species`, `Background`, `Feat`, `SpeciesChoice`, `TextOption`, `EquipmentOption`, `EquipmentOptions`) have an `Equal` method (go.md section 5), and an empty list of choices, spell ids or items is normalised to nil on construction so constructed and decoded entries agree under `Equal` and `Set.Equal`; `NewEquipmentOptions` cannot fail because both options are already valid; `BlockProblem` gained `BlockMissing` and `BlockConstraint` for the entry constructors.
- Mutation (testing.md §8, species and origins 1): `custom_background.go` gomu score 100.0% (43 killed of 49, 6 not viable, 0 survivors). `feat.go`, `species.go`, `decode_species.go` have no same-named test file, so they were mutated manually (about 25 distinct mutations each, one at a time, restored byte-identical): every viable mutation was killed except two that surfaced real gaps, now killed by `TestSpeciesChoicesAreCopiedOnRead` (`Species.Choices` must copy) and the "unknown kind with the fields of every variant" case of `TestDecodeSpeciesChoices` (`o.skip` must name `spellIds`); the remaining `==`/`<=`, `!=`/`>` mutants are the equivalent ones recorded beside them in the source.
- Mutation (testing.md §8, species and origins 2): `background.go`, `species_choice.go` and `decode_origins.go` have no same-named test file, so they were mutated manually (28, 43 and 44 distinct mutations, one at a time, restored byte-identical). Every viable mutation was killed except: in `decode_origins.go` eight that surfaced real gaps, now killed by `TestDecodeBackgroundRuleChoices` (id registers for the duplicate check, qualifier is a line, tool name is a name, category is a text, unknown tool kind skips both variants' keys, feat object rejects extra keys, gold and quantity caps are inclusive); and in `species_choice.go` one equivalent mutant, the index `-1` of `lineageLevelProblem`'s `BlockConstraint` result, which both callers discard.
- Done (Phase 2, step 6, `content/data/species.json`): the four species of the thin slice (Dragonborn, Dwarf, Elf, Human) as `Species` entries with their traits, `sizeText`, the two `choices` and the nine lineage `spellId`s, served by `data.BundledSource` (FR-064 holds) and pinned by content-thin-slice.md. Only that slice is bundled; the remaining species, backgrounds, feats, goldens, manifest and tooling are not.

Not yet implemented: FR-001 to FR-004, FR-007, FR-009, FR-011, FR-015, FR-018 to FR-020, FR-022 (the bundled data and its pins, step 6), the data-pin parts of FR-023 (the merged bundled set), FR-026, FR-028, FR-029, FR-031, FR-034, FR-038, FR-040, FR-041, FR-043 and FR-074 (bundled values), FR-044 to FR-053, FR-059, FR-062, FR-068, FR-070 (verification tooling, manifest, goldens, `species.json` and `origins.json`), FR-054 and FR-055 (the nine species, not the four of the slice), FR-065 (the embedded decode test of the full data); moved from T to D in the plan: FR-022 and FR-064.

## 9. Open questions and decisions

No question is open. Settled, one line each: **Feat choices** (Magic Initiate list, Skilled proficiencies, Ability Score Improvement) stay verbatim text covered by FR-062 (FR-042 withdrawn): applying a feat is feats-behaviour.md's, and `content` must not depend on it; a new FR id adds structure if that spec asks. **Skill and condition names in species text** stay text (FR-021): no rule needs references. **Two-person re-read by one agent** is a fresh session with no memory of the transcription, roles differing (FR-047), plus a human spot check (design basis: the check proves two roles, not independence). **Equipment item and `feature` prerequisite names** stay printed strings (`EquipmentItem{name, quantity}`, FR-030, FR-035) until content-equipment.md and content-classes.md publish ids; a new FR id then types them. **Custom background "worth 50 GP (including unspent gold)"** means exactly 5000 CP (FR-073): the gold absorbs any remainder, so a package below 50 GP is padded with unspent gold and one above is overspent; interpretation, because "including unspent gold" only makes sense as a fixed total and every printed option totals 50 GP (SRD 5.2.1, "Creating a Background", step 5, p. 193). **`EquipmentOption`/`EquipmentItem` versus content-classes.md** stay separate types (data specs do not depend on each other, shared-types-and-ownership.md FR-047); revisit only if the schema adds shared building blocks. **`BackgroundFeat` naming**: this spec's type is `BackgroundFeatReference` (FR-071); `BackgroundFeat` stays the `FeatSource` variant of feats-behaviour.md, so there is no collision and no amendment. **Question ids cited by other specs** (the section 5 "Declined requests from the thin slice" paragraph holds the reasoning): Q15 structured trait facts, declined (a trait stays `NamedText`); Q16 level-gated tables such as Breath Weapon thresholds, declined (they stay in the text); Q17 lineage spell-grant semantics and per-entry grant data (which species offers which choice), declined as data (typed `SpellID` per level accepted, FR-023); Q18 lineage overrides (Drow Darkvision 120, Wood Elf Speed 35), declined (applying them is character behaviour); Q19 the species `choices` element has no picks count (SX-1) and `Feat` implements `HasFeatCategory` (FR-025). Also settled: data format, schema change, shape and vocabulary
ownership, source name and decode-failure reporting (shared-types-and-ownership.md
FR-001, FR-005, FR-006, FR-030, FR-031, FR-034, FR-035, FR-046, FR-047); species spell references checked
by whole-set validation (FR-023, FR-054); manifest location (FR-068); verbatim wording is content
(architecture.md §3, §6); heights stay `sizeText` (SX-3); thin-slice requests
(section 5); equipment prices from the caller (FR-073); Human trait spellings
follow the printed page (FR-062); `Feat` implements `HasFeatCategory` (FR-025).

## 10. Changelog impact

Under `Unreleased`:

- **Added:** bundled SRD 5.2.1 species (9), backgrounds (4) and feats (17)
  (`data/srd_5_2_1/`, embedded under `content/data/`, served by
  `data.BundledSource`); the entry types `Species`, `Background`, `Feat` with
  their supporting types and ids; the schema extensions of SX-1 to SX-4
  (revision-table row `R1-species-backgrounds-feats`, one pre-0.1.0
  `schemaVersion`); and the custom-background validation
  (`CustomBackgroundEquipmentProblems` and its types), SRD 5.2.1 p. 193.
- **Changed:** `NOTICE.md` covers `data/`, `content/data/` and
  `testdata/srd/`; bundled species grow from the thin slice's four to nine.

Not **BREAKING** before the first release. After it, adding or removing a bundled
entry changes `ListKind` and is **BREAKING**; correcting a transcription error is
a fix (architecture.md §7).
