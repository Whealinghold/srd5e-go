# Magic items A–Z: bulk content, extraction and verification

| | |
|---|---|
| Status | partially implemented |
| Context | `content` |
| Roadmap phase | Types 2; bulk data 5 |
| Depends on | `content-schema-and-decoder.md`, `content-source-port.md` |

Phase rule: shared-types-and-ownership.md FR-014; see [../roadmap.md](../roadmap.md).
The description blocks are the neutral text-block family owned by content-schema-and-decoder.md (Q2, section 9), which that spec and `content-spells.md` both cite; this spec has no build-order edge to `content-spells.md`.

## 1. Purpose and scope

Specifies how the SRD 5.2.1 magic item catalogue ("Magic Items A–Z", pp. 209-253) becomes bundled content served by `Source` under `KindMagicItem` (content-source-port.md, kind table: "Magic Items", p. 204). It fixes:

- the `MagicItemEntry` content entry: identity, the header facts the SRD prints as structure (Category with its qualifier, rarity, Attunement) and the description, verbatim SRD text carried as the neutral text blocks of content-schema-and-decoder.md (`TextBlock`; reused, not redefined);
- the member-less `MagicItem` marker implemented by `MagicItemEntry` alone (FR-003); Equipment's four magic gear entries do not implement it (content-equipment.md FR-095) and are matched to the catalogue by id (FR-052);
- the JSON shape (root array `magicItems`, row R2-magic-items of the revision table in content-schema-and-decoder.md) and its decoding, all problems at once. `MagicItemID` is owned by content-schema-and-decoder.md FR-114 and FR-128; this spec only requests it (FR-002, Q1);
- fidelity rules for long, table-rich text and the two stat blocks embedded in item descriptions;
- the extraction tool, the verification suite (golden, PDF page check, counts, order) and the process steps;
- the closed vocabulary and static tables of the chapter rules "Magic Items" (pp. 204-208) and the Equipment chapter's Attunement text (pp. 102-103), as code constants with no wire shape (section 4.8, FR-078 to FR-097). They moved here from magic-item-rules.md, a `character` spec (shared-types-and-ownership.md FR-058), because `content` may not depend on `character`.

The catalogue has **258 entries**, counted twice (by reading the headings on pp. 209-253, and by a text pass matching every heading followed by a header line; the per-page and category counts below are the second pass's). The SRD's contents page (p. 2) lists no index of items, so no printed list is an authority: the golden fixture (FR-074) is. 258 and the tables below are leads to be confirmed by the first extraction run (FR-045).

| Category (printed in the header) | Entries |
|---|---|
| Armor | 19 |
| Potion | 24 |
| Ring | 22 |
| Rod | 7 |
| Scroll | 1 |
| Staff | 12 |
| Wand | 13 |
| Weapon | 33 |
| Wondrous Item | 127 |
| **Total** | **258** |

**Out of scope:**

- **Behaviour of the chapter rules** ("Magic Items", pp. 204-208; Equipment "Magic Items", pp. 102-103): magic-item-rules.md (`character`) covers it (Q9). This spec bundles the catalogue, the vocabulary, and the static tables and facts of section 4.8, and no behaviour.
- **Behaviour of items.** Charges, DCs, bonuses, spells cast from an item and Attunement effects stay verbatim text (schema spec Q7). Structured per-item facts are a host-supplied `MagicItemFacts` value defined here (FR-091), not an entry field (Q18).
- **Sub-variants as entries** (Potion of Healing (greater), the five Belts of Giant Strength, each Ioun Stone, Feather Token and Figurine type, each Horn of Valhalla): they live in their entry's text and tables (Q6).
- **Monster stat blocks** of creatures an item names (bold creature names point to "Monsters", p. 209), and structured parsing of the two embedded blocks (FR-034, Q5).
- **Prices, purchase and sale.** Rarity values are a static table (FR-084); incorporated-item cost (p. 205), halving a consumable's value and what a settlement sells are behaviour or Game Master guidance of magic-item-rules.md.
- **Not in the PDF.** The set is exactly the headings printed on pp. 209-253; no gap was found and none is flagged.

## 2. Source

Printed page numbers equal PDF page numbers (364 pages; PDF page 209 is printed "209"), as checked by content-equipment.md section 2.

- SRD 5.2.1, "Contents", p. 2 ("Magic Items" p. 204; "Magic Items A–Z" p. 209).
- SRD 5.2.1, "Magic Item Categories" (nine categories), p. 204; "Magic Item Rarity" and "Magic Item Rarities and Values" (six rarities), pp. 205-206.
- SRD 5.2.1, for section 4.8: "Armor", "Potions" (Using a Potion, Mixing Potions), "Potion Miscibility", "Rings", "Rods", p. 204; "Scrolls", "Staffs", "Wands", "Weapons", "Wondrous Items", p. 205; "Activating a Magic Item", "Consumable Items", "Charges", "The Next Dawn", "Cursed Items", "Magic Item Resilience", p. 206; "Magic Item Tools", "Magic Item Crafting Time and Cost", p. 207; "Sentient Item's Alignment, Communication, Senses and Special Purpose", "Conflict", p. 208; "Identifying a Magic Item", "Attunement", "No More Than Three Items", "Ending Attunement", pp. 102-103; "Coins", p. 89.
- SRD 5.2.1, sidebar "Magic Item Rules" ("Attunement Prerequisites"), p. 205; "Attunement", Equipment, "Magic Items", p. 102; "Attunement", Rules Glossary, p. 177.
- SRD 5.2.1, "Magic Items A–Z", pp. 209-253: introduction (alphabetical order; bold creature names point to stat blocks), p. 209; Adamantine Armor, p. 209, to Wings of Flying, p. 253; end of section, p. 253, and "Monsters", p. 254.
- SRD 5.2.1, embedded stat blocks: "Giant Fly" in "Figurine of Wondrous Power", p. 222; "Avatar of Death" in "Mysterious Deck", p. 232.
- SRD 5.2.1, "Index of Stat Blocks", pp. 2-4 (checked: neither "Giant Fly" nor "Avatar of Death" is listed).
- SRD 5.2.1, "Potion of Healing", "Spell Scroll", Equipment, p. 99 (gear rows that overlap the catalogue).

Package-design rules derive from `doc/rules/architecture.md` (§2, §3, §5, §6), `doc/rules/observability.md` (§2, §3), `doc/rules/testing.md` (§3 to §5, §8) and `AGENTS.md` ("Licensing and source material").

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Magic Item | `MagicItemEntry` (value object, `Entry`, kind `magicItem`); marker `MagicItem` (interface with one exported marker method) | Marker implemented by `MagicItemEntry` alone (FR-003). |
| (package term) magic item identifier | `MagicItemID` | Owned by content-schema-and-decoder.md FR-114, FR-128; same id format as every content id (shared-types-and-ownership.md FR-007). |
| Magic Item Categories | `MagicItemCategory` (enum) | Armor, Potion, Ring, Rod, Scroll, Staff, Wand, Weapon, Wondrous Item (p. 204; the table prints plurals, headers the singular). |
| Magic Item Rarity | `ItemRarity` (enum) | Common to Artifact (p. 206). |
| Rarity Varies, a rarity per bonus or type | `RaritySpec` (closed: `SingleRarity`, `RarityVaries`, `RarityByVariant`), `RarityOption` | Three header forms (FR-015). |
| Requires Attunement, ... by a Wizard | `Attunement` (closed: `NoAttunement`, `AttunementRequired`, `AttunementRequiredBy`) | Prerequisite is verbatim text (FR-019, Q7). |
| (header) category qualifier | `categoryNote` (optional string; the empty string means absent) | The parenthetical after the category. |
| Description of an item | `[]TextBlock` | Neutral family owned by content-schema-and-decoder.md (Q2); wire discriminators `paragraph`, `entry`, `bulletList`, `table`, `embeddedStatBlock`. |
| Run-in heading ("Curse.") | `TextEntry{name, text}` block | p. 210 and throughout. |
| Stat block inside an item | `TextEmbeddedStatBlock{lines}` block | Giant Fly p. 222, Avatar of Death p. 232 (FR-034). |
| Potion of Healing, Spell Scroll | entries `potions_of_healing`, `spell_scroll`; also gear rows of kind `equipment` | Overlap, FR-052. |
| (package term) heading, header line, golden | the red item title; the italic line below it (grammar FR-063); the independent transcription fixture (FR-074) | Counted per page in FR-046. |
| Section 4.8 types (category defaults, Potion Miscibility, rarity and crafting facts, tools, resilience, attunement facts, charges, `MagicItemFacts`, sentient tables) | listed in FR-097 | FR-078 to FR-096. `Alignment` is owned by content-schema-and-decoder.md (shared-types-and-ownership.md FR-053). Amounts are whole Copper Pieces (FR-084). |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3). The model uses no term from another edition or book.

## 4. Requirements

A "problem" is one `Problem` of the decoder (path and one of the ten `ProblemKind`s of content-schema-and-decoder.md FR-015). "Reject" means the decoder returns the joined `Problem` error containing that problem. Paths are JSON Pointers, for example `/magicItems/3/rarity/options/1/variant`. "Data test" means a test over the bundled data read through the decoder.

### 4.1 Identity and kind

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `MagicItemEntry` MUST implement `Entry` with `key` equal to `Key(KindMagicItem, id)`, so `Source.lookup` and `listKind(magicItem)` serve every catalogue entry as one kind. | SRD 5.2.1, "Magic Items A–Z", p. 209; "Contents", p. 2; content-schema-and-decoder.md FR-142 | happy |
| FR-002 | `MagicItemEntry.id` MUST be a `MagicItemID`, the typed id that content-schema-and-decoder.md FR-114 and FR-128 own (this spec defines no id class); the row R2-magic-items requests it. A `MagicItemID` wraps one `ID`, so it accepts exactly the strings `ID` accepts (lowercase `[a-z][a-z0-9_]*`, at most 64 characters) and rejects any other with a validation error; a `MagicItemID` MUST NOT be accepted where an `ItemID` or another typed id is expected, and an `ItemID` MUST NOT be accepted for `MagicItemEntry.id`. The id-format cases are tested where FR-114 is tested; this spec tests only the entry's use of the type. | shared-types-and-ownership.md FR-005, FR-046; content-schema-and-decoder.md FR-114, FR-128; content-schema-and-decoder.md FR-142 | unhappy (invalid input) |
| FR-003 | `MagicItem` MUST be an interface with exactly one exported marker method (`IsMagicItem()`), implemented by `MagicItemEntry` and by no other type in the module; no flag or getter named for it is added anywhere: callers test `_, ok := entry.(MagicItem)`. Equipment's `MagicGear`, `ScrollFamily` and `ScrollVariant` do not implement it (content-equipment.md FR-095); a caller holding one of those gear entries finds the catalogue entry by id (FR-052). The marker method is exported, so the set is open: a consumer's own entry type MAY implement it in its own package. | content-equipment.md FR-095; shared-types-and-ownership.md FR-047; SRD 5.2.1, "Potion of Healing", "Spell Scroll", p. 99 | unhappy (rule interaction) |
| FR-004 | `MagicItemEntry` MUST return a validation error (from its constructor) a `name` that is empty, whitespace only, or has leading or trailing whitespace, and a `description` that is empty. | architecture.md §3, §6; testing.md §4 (invalid input, empty and minimal) | unhappy (invalid input) |
| FR-005 | An entry's `name` MUST be the heading exactly as printed, including a trailing list of bonuses (`Ammunition, +1, +2, or +3`), a parenthetical (`Stone of Good Luck (Luckstone)`), a plural (`Potions of Healing`), hyphens (`Arrow-Catching Shield`) and typographic apostrophes and capitalisation. A heading that wraps onto two lines is joined with one space. | SRD 5.2.1, "Ammunition, +1, +2, or +3", p. 209; "Stone of Good Luck (Luckstone)", p. 248; "Potions of Healing", p. 236 | happy |
| FR-006 | In the bundled data an `id` MUST be derived from the printed name by the slug rule of content-equipment.md FR-003 (lower-cased, `'` and U+2019 removed, every other run of non-alphanumeric characters one underscore, no leading or trailing underscore, at most 64 characters, content-schema-and-decoder.md FR-030). The slug function lives in `tools/` only and is tested in `tools/` on the case table "Id derivation"; the data test does not import it: it compares the golden's expected `id` strings with the bundled ids and checks the 64-character limit on all 258. | content-equipment.md FR-003; content-schema-and-decoder.md FR-030 | happy |
| FR-007 | Ids MUST be unique within `magicItem`: the later entry with an id already used is rejected at its `id` with `duplicate`; the same id in another kind is accepted. | content-schema-and-decoder.md FR-031, FR-032, FR-098; testing.md §4 (duplicate entries) | unhappy (invalid input) |
| FR-008 | `MagicItemEntry` and every value object of this spec MUST be immutable: every slice they expose is returned as a copy (mutating it leaves the value unchanged, the next read is unchanged) and is copied from the constructor argument. | architecture.md §3, §4; testing.md §4 (immutability) | unhappy (immutability) |
| FR-009 | Two `MagicItemEntry`s MUST be equal under `MagicItemEntry.Equal` (the type holds slices, so it is not comparable with `==`) iff all their fields are equal; changing any single field makes them unequal. | architecture.md §3 (value object) | happy |
| FR-010 | Every bundled entry MUST have a `source` (the `SourceCitation` of content-schema-and-decoder.md FR-115, optional for host content) with `document` exactly `SRD 5.2.1`, `section` exactly `Magic Items A–Z` (en dash, as printed; Q11) and an integer `page` from 209 to 253. This narrows the general bundled-data rule (`document` exactly `SRD 5.2.1`, non-blank `section`, `page` 1 to 364). | content-schema-and-decoder.md FR-034, FR-090, FR-115; AGENTS.md ("cite by section and page"); SRD 5.2.1, p. 209 | unhappy (boundary) |

### 4.2 Header facts: category, rarity, Attunement

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-011 | `MagicItemCategory` MUST have exactly nine values, in the order of the Magic Item Categories table: armor, potion, ring, rod, scroll, staff, wand, weapon, wondrousItem. A `switch` over it MUST be exhaustive. | SRD 5.2.1, "Magic Item Categories", p. 204 | happy |
| FR-012 | On the wire `category` MUST be one of `Armor`, `Potion`, `Ring`, `Rod`, `Scroll`, `Staff`, `Wand`, `Weapon`, `Wondrous Item`, the singular spelling the headers print, matched exactly and case-sensitively. Any other value (including the table's plurals `Potions`, `Wondrous Items`, and `armor`) is `invalidValue` at `/magicItems/<i>/category`; absent is `missingField`. | SRD 5.2.1, "Magic Item Categories", p. 204; "Amulet of Health", p. 209 (header); content-schema-and-decoder.md FR-025 | unhappy (invalid input) |
| FR-013 | `categoryNote` MUST be optional and, when present, the non-blank text inside the header's category parenthetical, outer parentheses removed and inner commas kept (`Any Medium or Heavy, Except Hide Armor`); it is absent when the header prints no parenthetical, and blank is `invalidValue` at `/magicItems/<i>/categoryNote`. The model gives it no meaning and does not resolve it to equipment entries (Q15). | SRD 5.2.1, "Adamantine Armor", p. 209; "Berserker Axe", p. 213 | unhappy (boundary) |
| FR-014 | `ItemRarity` MUST have exactly six values in order: Common, Uncommon, Rare, Very Rare, Legendary, Artifact, with the SRD spelling on the wire. It carries no value or price. | SRD 5.2.1, "Magic Item Rarities and Values", p. 206 | happy |
| FR-015 | `RaritySpec` MUST be closed with exactly `SingleRarity(ItemRarity)` (a header such as `Rare`), `RarityVaries` (the header words `Rarity Varies`) and `RarityByVariant(options)` (a header such as `Uncommon (+1), Rare (+2), or Very Rare (+3)`). No nullable field or flag stands for a state. | SRD 5.2.1, "Amulet of Health", p. 209; "Belt of Giant Strength", p. 213; "Ammunition, +1, +2, or +3", p. 209 | happy |
| FR-016 | `RarityByVariant` MUST hold two or more `RarityOption(rarity, variant)` in printed order, each `variant` the non-blank verbatim text of the parenthetical (`+1`, `Silver or Brass`) and the variants distinct; fewer than two options or a duplicate or blank variant is a validation error. A rarity MAY repeat across options. | SRD 5.2.1, "Horn of Valhalla", p. 226; "Wand of the War Mage, +1, +2, or +3", p. 251 | unhappy (boundary) |
| FR-017 | On the wire `rarity` MUST be an object `{kind: "single", rarity}`, `{kind: "varies"}` or `{kind: "byVariant", options: [{rarity, variant}, ...]}`. Problems: `rarity` absent `missingField`; another `kind` `invalidValue` at `/rarity/kind`; a rarity word outside FR-014 (including `Very rare`) `invalidValue` at the rarity; `rarity` on `varies`, or any unlisted key, `unknownField`; fewer than two options `outOfRange` at `/rarity/options`; a repeated variant `duplicate` at the later `variant`; a blank variant `invalidValue`. | SRD 5.2.1, p. 206; content-schema-and-decoder.md FR-015, FR-022, FR-024, FR-026 | unhappy (invalid input) |
| FR-018 | `Attunement` MUST be closed with exactly `NoAttunement`, `AttunementRequired` and `AttunementRequiredBy(prerequisite)`; `prerequisite` is the non-blank verbatim text after `Requires Attunement by ` (for example `a Wizard`), else a validation error. | SRD 5.2.1, "Attunement", p. 102; "Magic Item Rules" sidebar, p. 205; "Hat of Many Spells", p. 224 | happy |
| FR-019 | On the wire `attunement` MUST be absent for `NoAttunement`, `{}` for `AttunementRequired` and `{by: <text>}` for `AttunementRequiredBy`; a blank `by` is `invalidValue` at `/attunement/by`, `null` or a non-object `wrongType`, another key `unknownField`. The prerequisite is not parsed: `a Dwarf or a Creature Attuned to a Belt of Dwarvenkind` and `a Spellcaster` are not classes (Q7). | SRD 5.2.1, "Dwarven Thrower", p. 219; "Pearl of Power", p. 234; "Magic Item Rules" sidebar, p. 205 | unhappy (invalid input) |
| FR-020 | In the bundled data exactly seven entries MUST be `RarityVaries` (Belt of Giant Strength, Feather Token, Figurine of Wondrous Power, Ioun Stone, Potion of Giant Strength, Potions of Healing, Spell Scroll), exactly six `RarityByVariant` (Ammunition, +1, +2, or +3; Armor, +1, +2, or +3; Horn of Valhalla; Shield, +1, +2, or +3; Wand of the War Mage, +1, +2, or +3; Weapon, +1, +2, or +3) and the other 245 `SingleRarity`. | SRD 5.2.1, the headers on pp. 209, 210, 213, 221, 226, 227, 235, 236, 243, 244, 251, 253 | happy |
| FR-021 | The bundled entries per `MagicItemCategory` MUST equal the table in section 1 (19, 24, 22, 7, 1, 12, 13, 33, 127), and each entry's category MUST equal the golden's (FR-075). | SRD 5.2.1, headers, pp. 209-253 | happy |

### 4.3 Description: fidelity, tables, lists, stat blocks

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-022 | `MagicItemEntry.description` MUST be a non-empty, ordered list of `TextBlock`s in printed order, returned as a copy (`paragraph`, `entry`, `bulletList`, `table`, `embeddedStatBlock`). This spec defines no block type, constructor rule or wire shape; the neutral family in content-schema-and-decoder.md owns them (Q2), and content-spells.md cites the same family. | content-schema-and-decoder.md (text-block family, per Q2); architecture.md §3 | happy |
| FR-023 | Text MUST be verbatim from the PDF. The only permitted changes are collapsing runs of whitespace, replacing line breaks inside a paragraph with single spaces, removing page footers (FR-024), resolving line-end hyphens (FR-065) and rejoining a sentence broken by a column or page. The en dash of numeric ranges (`01–60`), the minus sign U+2212 of `1d20 − 1` (p. 217) and `1d6 − 1` (p. 239), typographic quotation marks and apostrophes and the em dash MUST be kept as printed. No rewording, no spelling or grammar fixes. | AGENTS.md ("keep it faithful"); content-spells.md FR-064; SRD 5.2.1, "Deck of Illusions", p. 217; "Ring of Spell Storing", p. 239 | happy |
| FR-024 | The footer ("System Reference Document 5.2.1" and the printed page number) MUST NOT appear in any text, and a sentence that a column or page break interrupts MUST be rejoined without loss or duplication: "creates an" (p. 234) continues "extradimensional hole 10 feet deep" (p. 235); "cast-off" (p. 218) continues "scales and gift them" (p. 219). | SRD 5.2.1, "Portable Hole", pp. 234-235; "Dragon Scale Mail", pp. 218-219; content-spells.md FR-065 | unhappy (rule interaction: layout artefact) |
| FR-025 | Paragraph order MUST be kept exactly as printed; two printed paragraphs MUST NOT be merged and one MUST NOT be split. A paragraph that begins with a hanging indent (the command words of "Decanter of Endless Water", p. 217; "First Command Word." of "Gem of Brightness", p. 223) is a paragraph of its own. | SRD 5.2.1, "Decanter of Endless Water", p. 217; content-spells.md FR-060 | happy |
| FR-026 | A run-in heading MUST become `entry(name, text)`: a bold or bold-italic run at the start of a paragraph that ends with a full stop, name stored without that full stop and with any parenthetical kept (`Splash`, `Curse`, `Metal Shell`, `Anchor (Uncommon)`, `Bronze Griffon (Rare)`, `Acrobatic Assist (Quarterstaff and 10-Foot Pole Forms Only)`). Cases in table "Text fidelity". | SRD 5.2.1, "Armor of Invulnerability", p. 210; "Decanter of Endless Water", p. 217; "Feather Token", p. 221; "Quarterstaff of the Acrobat", p. 237 | happy |
| FR-027 | Emphasis other than a run-in heading is not carried: italic spell and item names and bold creature names stay as plain words in their paragraph, and a cross-reference to "Monsters" or "Equipment" stays text and is not resolved. | SRD 5.2.1, "Magic Items A–Z", p. 209; "Brazier of Commanding Fire Elementals", p. 214; content-spells.md FR-070 | happy |
| FR-028 | A bulleted list MUST be a `bulletList` block with one item per bullet in order, for example: "Ring of Animal Influence" (3 items, p. 237), "Robe of Useful Items" (6 items, p. 240), "Rod of Alertness" (4 items, p. 241). | SRD 5.2.1, pp. 237, 240, 241 | happy |
| FR-029 | A table MUST be a `table(title, header, rows)` block holding every printed data row and cell in order. `title` is the printed caption (`Cube of Force Faces`, `Feather Tokens`, `Gray Bag of Tricks`, `Wand of Wonder Effects`) and is absent when none is printed (the destination table of "Amulet of the Planes", p. 209). A footnote printed under a table (`*Stat blocks for these creatures ... appear in "Monsters."`) is a `paragraph` block immediately after it, and the asterisk in the header cell stays. | SRD 5.2.1, "Cube of Force", p. 216; "Feather Token", p. 221; "Deck of Illusions", pp. 217-218; "Wand of Wonder", p. 252 | happy |
| FR-030 | A table that continues in the next column or on the next page with its header repeated MUST be one table: the repeated header is dropped and the rows continue without a gap. This applies to "Deck of Illusions" (pp. 217-218), "Mysterious Deck" (p. 231), "Bag of Beans" (p. 211), "Hat of Many Spells" (p. 225), "Spell Scroll" (p. 244), "Staff of the Magi" (pp. 246-247), "Wand of Wonder Effects" (p. 252) and "Robe of Useful Items" (pp. 240-241). | SRD 5.2.1, as listed | unhappy (rule interaction: split table) |
| FR-031 | A table printed as two halves side by side with a repeated header MUST be one table as printed: the header repeats, each printed row holds the cells left to right, and an odd number of entries leaves empty-string cells in the last row (`Staff of Fire`, p. 245: `Fireball`, `3`, ``, ``). It MUST NOT be merged into a taller two-column table (Q12). | SRD 5.2.1, "Armor of Resistance", p. 210; "Staff of Fire", p. 245; "Dragon Scale Mail", p. 219; "Ammunition of Slaying", p. 209 | unhappy (boundary: odd half) |
| FR-032 | A table cell MUST be the printed text with its own line breaks collapsed to single spaces; a printed dash cell (`—`) is kept as `—`; a printed blank cell is the empty string; the header cells of "Mysterious Deck" are `1d100 (13-Card Deck)`, `1d100 (22-Card Deck)`, `Card`. | SRD 5.2.1, "Mysterious Deck", p. 231 | unhappy (boundary) |
| FR-033 | The row count and header of each anchor table in the case table "Anchor tables" MUST be pinned by a data test; the full set of tables (pages and row counts) is read from the PDF when the fixture is written and pinned in `content/testdata/srd/magic_items/`, and a table found by extraction on a page the fixture does not list is a finding. | SRD 5.2.1, pp. 209-252; content-spells.md FR-068 | unhappy (rule interaction: embedded table) |
| FR-034 | The two stat blocks printed inside item descriptions MUST be kept as `embeddedStatBlock` blocks of their owning entry, at their printed position, with every printed line, and MUST NOT become `Monster` entries or entries of any kind: "Giant Fly" inside `figurine_of_wondrous_power` (between the entry `Ebony Fly (Rare)` and the entry `Golden Lions (Rare)`, p. 222) and "Avatar of Death" as the last block of `mysterious_deck` (after the entry `Void`, p. 232). No other embedded block exists in the catalogue. | SRD 5.2.1, "Figurine of Wondrous Power", p. 222; "Mysterious Deck", p. 232; content-monsters.md section 1 (out of scope); content-spells.md FR-069 | unhappy (rule interaction: header-less sub-entry) |
| FR-035 | A stat block's lines MUST be one line per printed row with the tiles of a row joined left to right by single spaces; nothing is added, dropped or recomputed, so printed values that a Monster could not hold stay: Avatar of Death's `HP Half the HP maximum of its summoner`, `CR None (XP 0; PB equals its summoner's)` and `Melee Attack Roll: Automatic hit`, and its alignment as printed. | SRD 5.2.1, "Avatar of Death", p. 232; "Giant Fly", p. 222 | unhappy (boundary: non-numeric values) |
| FR-036 | The bundled monster set MUST NOT change: the bundled magic item document has no `monsters` key and no entry of kind `monster`, and no entry of kind `monster` served by `BundledSource` has the name `Giant Fly` or `Avatar of Death`, so content-monsters.md FR-056 (the set equals the index of stat blocks, pp. 2-4) stays true. The test reads the bundled source through `listKind(KindMonster)`, not another spec's fixture, and passes whether or not the monster data is bundled yet. | SRD 5.2.1, "Index of Stat Blocks", pp. 2-4; content-monsters.md FR-056 | unhappy (rule interaction) |

### 4.4 Decoding and JSON shape

JSON object per entry, keys in report order: `id`, `name`, `source {document, section, page}`, `category`, `categoryNote?`, `rarity {kind, rarity?, options?}`, `attunement? {by?}`, `description [blocks]`. The decoder maps these to the closed types; the domain never sees a flag.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-037 | The root array `magicItems` MUST be registered as the one row R2-magic-items of the revision table in content-schema-and-decoder.md, which lists no reference site (a magic item entry holds no `Key` of another kind, Q10); this spec claims no schema version (one `schemaVersion`; before 0.1.0 every revision lands in it). Until the row is applied, `magicItems` is `unknownField` at `/magicItems`; once applied, an absent `magicItems` means no entries, and a non-array is `wrongType`. Problem order follows the revision table (after `monsters`, Q4). | shared-types-and-ownership.md FR-034, FR-035; content-schema-and-decoder.md FR-008, FR-010, FR-011, FR-103, FR-126 | unhappy (boundary) |
| FR-038 | A `magicItems` element MUST be an object whose keys, in problem-report order, are `id`, `name`, `source`, `category`, `categoryNote`, `rarity`, `attunement`, `description`. Required: `id`, `name`, `category`, `rarity`, `description` (`source` is optional for host content, decoder FR-034, and mandatory for bundled data, FR-010). A key not listed is `unknownField` at `/magicItems/<i>/<key>`. | content-schema-and-decoder.md FR-016, FR-022, FR-034 | unhappy (invalid input) |
| FR-039 | Problems MUST use the ten kinds of content-schema-and-decoder.md FR-015: absent required field `missingField`; wrong JSON type (including `null`) `wrongType`; a word outside a closed set or a blank string `invalidValue`; an array length outside bounds `outOfRange`; a repeated variant `duplicate`; unlisted key `unknownField`. This spec adds no kind. Each kind used MUST be reachable from a test and carry the path of the offending value. | content-schema-and-decoder.md FR-015, FR-019 to FR-028; testing.md §4 | unhappy (invalid input) |
| FR-040 | The smallest valid entry (`id`, `name`, `category`, `rarity` `single`, `description` of one paragraph) MUST decode to an entry with no `categoryNote`, `NoAttunement` and a copied description. | SRD 5.2.1, "Immovable Rod", p. 227; testing.md §4 (empty and minimal) | unhappy (empty and minimal) |
| FR-041 | `description` on the wire MUST be the array of block objects whose shape the neutral text-block family of content-schema-and-decoder.md owns (discriminator `kind`); an empty array is `outOfRange` at `/magicItems/<i>/description`, absent is `missingField`, and a block problem is reported at the block's path by that family's rules. This spec defines no block shape (Q2). | content-schema-and-decoder.md FR-024 and its text-block family (Q2) | unhappy (boundary) |
| FR-042 | The decoder MUST report every problem of every `magicItems` entry in one pass, in the field order of FR-038 and by ascending index, with no cascade (a `rarity` that is a string yields one `wrongType`, not one per option). | content-schema-and-decoder.md FR-012, FR-013, FR-016 | unhappy (rule interaction) |
| FR-043 | Decoding the same text twice MUST give equal entries and equal failures in the same order, and MUST NOT modify its input. | content-schema-and-decoder.md FR-072, FR-076; testing.md §4 (determinism) | unhappy (determinism) |
| FR-044 | The decoder and the constructors MUST agree: the case tables "Decoder rows" and "Header forms" fed to both the decoder (expecting the stated problem) and the matching constructor (expecting a validation error) give the same accept and reject verdicts, and each limit (nine categories, six rarities, at least two options) is defined once and used by both. | content-schema-and-decoder.md FR-093, FR-094; architecture.md §6 | unhappy (rule interaction) |

### 4.5 Bundled set and completeness

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-045 | The bundled set MUST contain exactly the headings printed on pp. 209-253, no entry missing and no extra entry: the golden fixture (FR-074) is the authority and its size MUST equal the number of entries; `listKind(magicItem)` MUST return them ordered by id (ordinal). The count 258 MUST be confirmed or corrected by the first extraction run and the correction recorded here. | SRD 5.2.1, "Magic Items A–Z", pp. 209-253; content-source-port.md FR-011 | unhappy (missing content) |
| FR-046 | The number of bundled entries whose `source.page` is each printed page MUST equal the table "Headings per page" (45 pages, 258 in total; pages 222 and 252 have none). | SRD 5.2.1, pp. 209-253 | unhappy (boundary) |
| FR-047 | The bundled entries in printed order MUST be in order of the key (name lower-cased, every character that is not a letter or digit removed, compared by code unit), with exactly one recorded exception: `Potions of Healing` (p. 236) is printed between `Potion of Growth` and `Potion of Heroism`, as if singular. The exception names the pair, the page and the reason; any other violation, or an exception that no longer applies, fails the suite. | SRD 5.2.1, "Magic Items A–Z" ("presented in alphabetical order"), p. 209; "Potions of Healing", p. 236 | unhappy (rule interaction) |
| FR-048 | Every bundled `source.page` MUST equal the golden's `page` and be the page of the entry's heading, also for an entry that continues on later pages ("Portable Hole" cites 234, "Ring of Elemental Command" 237, "Staff of Power" 245); no entry cites a page before 209 (the chapter rules, pp. 204-208, are not entries) or after 253. | SRD 5.2.1, pp. 209-253; content-spells.md Q7 | unhappy (boundary) |
| FR-049 | The bundled data MUST be one JSON document, `content/data/magic_items.json` (source of truth, current `schemaVersion`, array `magicItems`), embedded directly with `go:embed` (content-schema-and-decoder.md FR-120); `data.Files()` lists the stem `magic_items` (content-schema-and-decoder.md FR-122). This spec adds no generator, drift test, header or README check or `NOTICE.md` path check (layout: architecture-guardrails.md FR-078; header of the Go file: content-licensing-and-notice.md FR-029; notice paths: architecture-guardrails.md FR-079). No `.json` file is placed outside `content/data/` and `testdata/`. One document keeps every id in one decode (Q13). | shared-types-and-ownership.md FR-030, FR-032; content-schema-and-decoder.md FR-120, FR-122, FR-123; content-licensing-and-notice.md FR-008; architecture-guardrails.md FR-078; architecture.md §5 | unhappy (drift) |
| FR-050 | `data.BundledSource` (owned by content-source-port.md FR-059 to FR-061; this spec adds only the embedded file, FR-064 there), built from `data.Files()`, MUST return the found variant for the key of every bundled magic item, the not-found variant for an unknown id, the not-found variant for a bundled magic item id under any other kind, and exactly the golden's entries from `ListKind(ctx, KindMagicItem)` in id order. A decode failure of the bundled document makes the constructor return `BundledContentInvalid` (an error, content-source-port.md FR-061), never a result and never a source failure. | content-source-port.md FR-008, FR-009, FR-010, FR-011, FR-059, FR-061, FR-064; shared-types-and-ownership.md FR-031 | unhappy (missing content: wrong kind) |
| FR-051 | The read-only suite `contentSourceReadOnlyChecks` MUST run against `BundledSource` with `expectedEntries` the golden-checked decoded catalogue, and the shared suite `contentSourceContractChecks` MUST run with `create: (entries) => MemorySource(entries)`, using the decoded catalogue as one fixture input. This spec registers no source of its own. | content-source-port.md FR-028, FR-047, FR-058 | happy |
| FR-052 | The catalogue MUST contain `potions_of_healing` (Potion, `RarityVaries`; table `Potion`, `HP Regained`, `Rarity` with the four rows of the case table "Cross-kind pairs") and `spell_scroll` (Scroll, `RarityVaries`; the ten-row table of "Anchor tables"), although equipment holds the priced gear rows `Potion of Healing` and `Spell Scroll`. Neither kind replaces the other, no cross-kind reference field exists (Q8), and both ids exist under their own kind. | SRD 5.2.1, "Potions of Healing", p. 236; "Spell Scroll", p. 244; "Potion of Healing", "Spell Scroll", p. 99 | unhappy (rule interaction: overlap) |
| FR-053 | A data test MUST scan every text, name, table cell and stat block line of the bundled data and find none of: the footer string, a soft hyphen (U+00AD), a run of two spaces, a leading or trailing space, a blank text, or a table row of a different width than its header. | content-spells.md FR-065, FR-066, FR-059; content-schema-and-decoder.md FR-150; architecture.md §5 | unhappy (invalid input: layout artefact) |
| FR-054 | No bundled entry MAY come from outside SRD 5.2.1: an entry that the PDF check (FR-076) does not find on its cited page, or whose source is another book, SRD 5.1 or memory, MUST be removed, and anything that looks like a magic item but is absent from the PDF MUST be listed in the change summary. Enforced by the PDF check on the real text and by review, not by CI (PR-2, PR-3). | AGENTS.md ("If it is not in that PDF, it does not go in this repository") | n/a (process) |
| FR-055 | Nothing in the data, code, tests or tools of this spec MAY name a consuming project (the guardrail denylist scan over the module, `tools/` and the data). | AGENTS.md ("Hard constraints"); architecture-guardrails.md FR-025, FR-026 | happy |
| FR-056 | The domain types of this spec (`MagicItemEntry`, `MagicItem`, `MagicItemCategory`, `ItemRarity`, `RaritySpec` and its subtypes, `RarityOption`, `Attunement` and its subtypes; not `MagicItemID`, FR-002) MUST live in the magic-item files of package `content`, be synchronous and pure (no `os`, `context`, goroutine, clock, randomness), import nothing from `telemetry` and contain no telemetry call and no use-case type, and be exported with a doc comment on each. Each is declared once and its name is unique across the package; none is a generic variant name (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`, `Found`, `NotFound`), and the closed families `RaritySpec` and `Attunement` are value-object families, not use-case results. The decoder additions stay behind `JSONDecoder` with no new exported name. | architecture.md §2, §3, §5, §7; observability.md §3; shared-types-and-ownership.md FR-017 | happy |
| FR-057 | A consumer importing only the public packages (`content`, `content/data`) MUST be able to build the bundled source, look up the key `(magicItem, bag_of_holding)` and read category `Wondrous Item`, rarity `Uncommon` and no Attunement, and look up `(magicItem, amulet_of_health)` and read `AttunementRequired` (scenario test, `package scenarios_test`). | SRD 5.2.1, "Bag of Holding", p. 212; "Amulet of Health", p. 209; architecture.md §7 | happy |

### 4.6 Extraction tool

A developer aid in `tools/`; its output is a draft that a person reviews, and the reviewed JSON is the source of truth. The process is: draft (FR-059 to FR-073), review (FR-068), independent golden (FR-074, PR-1), CI data tests, PDF check (FR-076, PR-2), page review (PR-4). Completeness has no printed index, so the golden, the per-page heading counts (FR-046) and the order check (FR-047) stand in for one; the PDF check also looks for headings no entry cites.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-058 | The extraction tool and the PDF check (FR-076) MUST live under `tools/` (its own module with its own `go.mod`), MUST NOT be imported by any package of the root module (or by a test other than their own tests in `tools/`), MUST add no dependency to the root `go.mod`, MUST be excluded from the published module (a nested module is not part of the root archive), MUST write any draft to a path outside `content/data/` (the reviewed JSON becomes the data file only after a person has checked it, PR-4), and MUST import from the module only its public packages, never `internal/`. There is no embed generator (content-schema-and-decoder.md FR-120). | architecture.md §5, §9; shared-types-and-ownership.md FR-032; content-equipment.md FR-070; content-spells.md FR-105; content-schema-and-decoder.md FR-124 | unhappy (state error) |
| FR-059 | The tool MUST be given a local PDF path, MUST NOT read the network, and MUST abort before reading further or writing anything unless the PDF is SRD 5.2.1 as expected: 364 pages, the footer of page 209 reads "209", the first heading on page 209 is "Adamantine Armor", the last heading on page 253 is "Wings of Flying", and page 254 begins "Monsters". | `doc/research/sources.md` ("The PDF is not stored in this repository"); content-spells.md FR-071 | unhappy (invalid input) |
| FR-060 | The tool MUST read pages 209-253 twice: through `pdftohtml -xml` (position and emphasis per text run, run in Docker with poppler, not a Go dependency) for the structural reading, and through the default `pdftotext` order for the comparison pass (FR-073); the structural reading MUST take the left column of a page in full before the right column, by coordinates. A page on which a heading's header line is not found directly below it is reported `columnOrderSuspect`. | content-spells.md FR-067, FR-100, Q9; content-equipment.md PR-2 | unhappy (rule interaction: column layout) |
| FR-061 | The tool MUST report the number of headings it found, in total and per page, and compare them with the golden's; a title that no header line follows is `unparsedHeading` and a header line with no title is `unparsedHeader`, each with its page. | SRD 5.2.1, pp. 209-253 | unhappy (missing content) |
| FR-062 | The tool MUST tell an item heading from a stat block title by what follows: a heading is followed by a header line (FR-063); a stat block title is followed by a size, type and alignment line and then `AC`. A stat block belongs to the item whose text precedes it and ends where the next item heading begins. Synthetic fixtures built on the layouts of pp. 222 and 232 MUST give exactly one block in each of the two items and none elsewhere. | SRD 5.2.1, "Giant Fly", p. 222; "Avatar of Death", p. 232 | unhappy (rule interaction: header-less sub-entry) |
| FR-063 | The tool MUST parse a header line, after joining the lines it wraps onto, by one grammar: `<Category>[ (<note>)], <rarity>[ (Requires Attunement[ by <prerequisite>])]`, where `<rarity>` is one rarity word, `Rarity Varies`, or a comma and `or` separated list of `<rarity> (<variant>)`; and map each printed form of the case table "Header forms" to its value. A header matching no form is `unparsedHeader`, and the tool supplies no value for it. | SRD 5.2.1, headers, pp. 209-253 | happy |
| FR-064 | The tool MUST NOT guess: a category outside the nine, a rarity word outside the six, or an Attunement text it cannot read is reported with the item and page and left out of the draft. | AGENTS.md ("never write rules from memory"); content-spells.md FR-074 | unhappy (missing content) |
| FR-065 | A line-end hyphen or soft hyphen MUST be resolved by the corpus rule of content-spells.md FR-066 (joined word attested elsewhere in the PDF text: join; hyphenated form attested: keep; neither: unresolved, a problem cleared only by a correction entry), each resolution listed in the report and the rule tested with a row each for join, keep, unresolved and soft hyphen. Examples: "de-/termines" (p. 209) and "al-/lowing" (p. 210) join. | content-spells.md FR-066; SRD 5.2.1, "Ammunition of Slaying", p. 209; "Apparatus of the Crab", p. 210 | unhappy (boundary: hyphenation) |
| FR-066 | A run-in heading is a bold or bold-italic run at the start of a paragraph that ends with a full stop; an inline bold or italic run elsewhere is ignored; a bold run at the start of a paragraph that does not end with a full stop is reported `ambiguousEmphasis` and resolved only by a correction entry. | content-spells.md FR-100; SRD 5.2.1, "Decanter of Endless Water", p. 217 (bold only); "Armor of Invulnerability", p. 210 (bold italic); "Brazier of Commanding Fire Elementals", p. 214 (inline bold) | unhappy (invalid input: layout cues) |
| FR-067 | The tool MUST capture every table, detect a repeated header at a column or page break (FR-030) and two halves side by side (FR-031), and report as a finding any table whose rows do not all have the header's width; it MUST NOT merge or reorder rows to repair one. | SRD 5.2.1, pp. 210, 217-218, 231, 245 | unhappy (rule interaction: table layout) |
| FR-068 | A hand correction or accepted finding MUST be an explicit log entry (item, field, printed page, reason) that the tool applies; the data file MUST NOT be hand-edited otherwise. The log-applying function is pure, rejects an entry that matches nothing (`staleCorrection`), and is tested on synthetic records. | content-spells.md FR-077; testing.md §5 (keep citations true) | unhappy (state error: unrecorded edit) |
| FR-069 | Extraction MUST be deterministic: the same PDF and tool version give byte-identical files and an identical report, with entries in printed order and no timestamp. | testing.md §4 (determinism); content-spells.md FR-072 | happy |
| FR-070 | The tool MUST report all problems of a run, each with item name (or page), field and kind, MUST exit non-zero if any exists, and MUST write no data file in that case. | architecture.md §5 (every problem, not just the first); content-spells.md FR-073 | unhappy (invalid input) |
| FR-071 | The tool MUST write only through an injected `io.Writer` factory, and only the data file `content/data/magic_items.json` and the report; a request to write any other path, an image, or a page-text dump MUST return an error from the writer factory and write nothing. | `doc/research/sources.md`; content-monsters.md FR-069; AGENTS.md | unhappy (state error) |
| FR-072 | The extraction run MUST NOT read the golden fixture (FR-074): only the verification step (FR-075) does, so the draft and the golden stay independent. | content-equipment.md PR-1 | unhappy (state error) |
| FR-073 | The two readings (FR-060) MUST be compared per item after normalisation (collapse whitespace, resolve hyphens by FR-065, drop footers, drop a checked-in allow-list of known layout artefacts); each difference that survives is reported. The comparison is per item, not per page, so the interleaved columns of the raw order do not mis-assign text. | content-spells.md FR-079 | unhappy (rule interaction: two readings disagree) |

### 4.7 Verification of the bundled set

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-074 | A golden fixture, `content/testdata/srd/magic_items_golden.json`, MUST hold for every heading the name, the page, the category, the `categoryNote`, the rarity and the Attunement (no description text), transcribed independently of the extraction draft (PR-1), plus the row count and header of each anchor table (FR-033). The golden is exempt from testing.md §5 "Cite, do not copy" because a transcription check has to hold the values; it lives only under the covered test path. | testing.md §1, §5, §8; shared-types-and-ownership.md FR-033 | happy |
| FR-075 | The data test MUST fail on any difference between the golden and the bundled data in name, page, category, `categoryNote`, rarity, Attunement and the pinned table shapes, and on any entry in one that is not in the other. | testing.md §8; AGENTS.md | unhappy (missing content) |
| FR-076 | `tools/check_magic_item_pages` MUST take the text of printed pages 209-253 and the bundled document and print one report line `<id> <page> <reason>` for every entry whose printed name is not found, as a heading (a line or a pair of wrapped lines, hyphens joined, whitespace collapsed, case-sensitive), on its cited page, and for every heading on pages 209-253 that no entry cites; it exits non-zero when the report is not empty. It proves presence on the page, not the correctness of values. It is unit-tested on synthetic page text (a missing name, a name wrapped over two lines, a name that occurs only in body text, a heading with no entry). Running it on the real text is PR-2. | AGENTS.md ("If it is not in that PDF, it does not go in this repository"); content-equipment.md FR-073 | unhappy (missing content) |
| FR-077 | The spot-check fixtures of the case table "Spot-check entries" MUST hold: the exact block kinds and counts of the listed entries, and the first and last lines of the two embedded stat blocks. | SRD 5.2.1, as cited per row | happy |

### 4.8 Rules vocabulary and static tables (moved from magic-item-rules.md)

shared-types-and-ownership.md FR-058 moves these here so that magic-item-rules.md is a pure `character` spec. Everything in this section is a code constant or value type of the magic-item facts files of package `content` (static tables are `const` blocks, functions and `switch`es, never package-level variables: plan item P6, architecture.md A4): no wire shape, revision-table row, `Kind` or `schemaVersion` change, never decoded (FR-097). It carries static SRD facts only (shared-types-and-ownership.md FR-006); which fact applies to an item, and every roll, sum or state change, is behaviour in magic-item-rules.md. Amounts are whole Copper Pieces (FR-084; Q21).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-078 | `MagicItemCategoryFacts{}.Of(category)` MUST be total over the nine `MagicItemCategory` values (FR-011) and return, as the static defaults of the category, `consumable` (true for `potion` and `scroll` only), `arcaneFocus` (true for `rod`, `staff` and `wand` only), `nonmagicalQuarterstaff` (true for `staff` only), `mustBeWorn` (true for `armor` and `ring` only) and `activationTime` (an `ActivationTime`, FR-079: `BonusAction` for `potion`, `MagicAction` for every other category), exactly as table "Category facts". A `switch` over the category MUST be exhaustive. These are defaults ("Unless its description notes otherwise"): a departure by one item is an override (`BoolFactOverride` or `ActivationTimeOverride`) in its `MagicItemFacts` (FR-091), and which of the two applies is behaviour of magic-item-rules.md (its FR-124). | SRD 5.2.1, "Armor", "Potions" (Using a Potion), "Rings", "Rods", p. 204; "Scrolls" (Using a Scroll), "Staffs", "Wands", p. 205; "Consumable Items", "Activating a Magic Item", p. 206 | happy |
| FR-079 | `ActivationTime` MUST be a closed enum with exactly five members, `ActivationMagicAction`, `ActivationBonusAction`, `ActivationAction`, `ActivationReaction` and `ActivationDescribedElsewhere` (the `Activation` prefix keeps the generic names `Action` and `Reaction` out of the package namespace, shared-types-and-ownership.md FR-017, FR-021; in the rest of this spec the members are written without it); `DescribedElsewhere` means the item description governs (an oil "might take longer as specified in its description", a Spell Scroll takes the spell's casting time) and MUST be a valid value for any category: the vocabulary does not restrict it to a Potion, because "the description of each item category or individual item details how an item is activated". How `combat` maps these to its action types is not decided here (magic-item-rules.md Q6). | SRD 5.2.1, "Using a Potion", p. 204; "Activating a Magic Item", p. 206 | unhappy (rule interaction) |
| FR-080 | `PotionMiscibilityTable.bands` MUST hold exactly eight `PotionMiscibilityBand(minRoll, maxRoll, effect)` rows that together cover every roll from 1 to 100 exactly once, with 100 as the table's "00": 1 `PotionsExplode`; 2 to 8 `PotionsBecomeIngestedPoison`; 9 to 15 `BothPotionsLoseEffects`; 16 to 25 `OnePotionLosesEffect`; 26 to 35 `PotionEffectsHalved`; 36 to 90 `PotionsWorkNormally`; 91 to 99 `OnePotionEffectsDoubled`; 100 `OnePotionPermanent`. No gap and no overlap exist, and a band with `minRoll` above `maxRoll`, or a roll outside 1 to 100, makes the band constructor return an error wrapping `srderr.ErrInvalidInput`. `PotionMiscibilityTable` holds no state and its zero value is the table (`Bands()`, `ForRoll`, `MustForRoll`). | SRD 5.2.1, "Potion Miscibility", p. 204 | unhappy (boundary) |
| FR-081 | Every `PotionMiscibilityEffect` MUST expose the facts of its row as typed fields, not text, with `bothPotionsLoseEffects` (bool) and `hostChoices` (a `[]MixtureChoice` (a set), the decisions the SRD leaves to "you", which the package never makes). `PotionsExplode`: `bothPotionsLoseEffects` true and an `ExplosionFacts` of a 5-foot-radius Sphere (`radiusFeet` 5, `shape` the single member `ExplosionSphere` of the enum `ExplosionShape`) whose point of origin is the mixture itself (`origin` equal to `ExplosionOriginMixture`, "centered on itself", not the drinker, because potions can be poured into a container), `damage` `4d10` (a `dice.Expression`), `the Force damage type`, affecting each creature in the area. `PotionsBecomeIngestedPoison`: `bothPotionsLoseEffects` true and `hostChoices` `{poison}` ("of your choice"). `BothPotionsLoseEffects`: `bothPotionsLoseEffects` true. `OnePotionLosesEffect`: false and `{losingPotion}`. `PotionEffectsHalved`: numerical effects and durations halved and `potionWithNothingToHalveLosesEffect` true. `PotionsWorkNormally`: no choices. `OnePotionEffectsDoubled`: the numerical effects and duration of ONE potion are doubled, `hostChoices` `{doubledPotion}` and `workNormallyIfNothingToDouble` true. `OnePotionPermanent`: only one potion works with permanent effects, `hostChoices` `{survivingPotion, permanentEffect}` ("choose the simplest effect ... or the one that seems the most fun"). `MixtureChoice` has exactly `poison`, `losingPotion`, `doubledPotion`, `survivingPotion` and `permanentEffect`. | SRD 5.2.1, "Potion Miscibility", p. 204 | happy |
| FR-082 | `PotionMiscibilityTable.ForRoll(roll)` MUST return the `PotionMiscibilityEffect` of the band holding the roll (FR-080); `ForRoll(0)`, `ForRoll(101)` and a negative roll MUST return an error wrapping `srderr.ErrInvalidInput` whose typed cause is `*RollRangeError{Roll, Min, Max}` (a roll can arrive from the host, so there is a single fallible form and no panicking one outside `MustForRoll`). The dice are rolled by `RollPotionMiscibility` of magic-item-rules.md, never here (Q23). | SRD 5.2.1, "Potion Miscibility" (1d100), p. 204 | unhappy (invalid input) |
| FR-083 | `ItemRarityFacts{}.Of(rarity)` MUST be total over the six `ItemRarity` values (FR-014), in the order common, uncommon, rare, veryRare, legendary, artifact, with an exhaustive `switch`, and return that rarity's `value` (FR-084) and its crafting-row lookup (FR-085): table "Rarity facts" has exactly six rows. No type named `MagicItemRarity` exists (Q18). | SRD 5.2.1, "Magic Item Rarity", p. 205; "Magic Item Rarities and Values", p. 206 | happy |
| FR-084 | `MagicItemValue` MUST be a closed interface: `PricedMagicItemValue{copperPieces}` (a non-negative `int64`, so no amount depends on the platform integer width, else `NewPricedMagicItemValue` returns an error wrapping `srderr.ErrInvalidInput`; `PricedMagicItemValueFromGold(n)` is `n` times 100 Copper Pieces, SRD "Coin Values", p. 89, and returns the same error for a negative `n` or one that overflows) and `PricelessMagicItemValue`. `ItemRarityFacts{}.Of(rarity).Value()` MUST be Common 100 GP, Uncommon 400 GP, Rare 4,000 GP, Very Rare 40,000 GP, Legendary 200,000 GP and Artifact `PricelessMagicItemValue`. The value of a consumable item, of a Spell Scroll and of an item that incorporates another is behaviour of magic-item-rules.md (its FR-017 to FR-020). | SRD 5.2.1, "Magic Item Rarities and Values", p. 206; "Coins", p. 89 | happy |
| FR-085 | `MagicItemCraftingRow(rarity, days, workHours, cost)` MUST exist for exactly five rarities, `workHours` always 8 times `days` (a day of crafting is 8 hours of work) and `cost` a `PricedMagicItemValue`: Common 5 days (40 hours) 50 GP; Uncommon 10 days (80 hours) 200 GP; Rare 50 days (400 hours) 2,000 GP; Very Rare 125 days (1,000 hours) 20,000 GP; Legendary 250 days (2,000 hours) 100,000 GP. `ItemRarityFacts{}.Of(artifact).Crafting()` MUST be `MagicItemCraftingRowAbsent` and every other rarity `MagicItemCraftingRowFound` holding its row (only `ItemRarityFacts` builds a `MagicItemCraftingRowFound`); the Artifact has no row and no value is invented. The table's footnote (halving for a consumable item other than a Spell Scroll) is behaviour of magic-item-rules.md (its FR-075). | SRD 5.2.1, "Magic Item Crafting Time and Cost", p. 207; "Work per Day" (Time and Cost), p. 207 | unhappy (missing content) |
| FR-086 | `MagicItemToolTable{}.Of(category)` (a type of its own, because `MagicItemToolRule` is the closed interface and cannot also carry the method) MUST be total over the nine categories and return the closed `MagicItemToolRule` of table "Magic Item Tools": `FixedTools([]ItemID)` for ring (`jewelers_tools`), rod (`woodcarvers_tools`), scroll (`calligraphers_supplies`), staff (`woodcarvers_tools`), wand (`woodcarvers_tools`) and potion (the two alternatives `alchemists_supplies` and `herbalism_kit`, either of which suffices); `ByBaseItemTools` for armor and weapon (the tools "depending on the kind of armor" or "weapon", read from the base item by magic-item-rules.md FR-079); `TinkersOrBaseItemTools` for wondrous item ("Tinker's Tools or the tool required to make the nonmagical item on which the magic item is based"). A `FixedTools` set MUST be non-empty, else a validation error. The ids are the snake_case ids of the `Tool` entries of the equipment data, resolved by id only (this spec does not read that data; a scenario test checks the ids exist when both are bundled). | SRD 5.2.1, "Magic Item Tools", p. 207; "Tools" (Crafting Magic Items), p. 206 | happy |
| FR-087 | `MagicItemResilience` MUST be the enum `resistantToAllDamage`, `nonmagicalDurability` and `impervious`, and `MagicItemResilienceFacts{}.DefaultFor(category, rarity)` MUST return `impervious` for `the Artifact rarity` in every category, else `nonmagicalDurability` for `potion` and `scroll`, else `resistantToAllDamage` ("Most magic items, other than Potions and Scrolls, have Resistance to all damage"; an Artifact is "impervious to damage"), exactly as table "Resilience facts"; the Artifact rule takes precedence over the Potion and Scroll rule. A per-item exception is a `ResilienceOverride` in `MagicItemFacts.resilience` (FR-091) and wins by magic-item-rules.md FR-073. The enum has no behaviour: `combat` maps `impervious` to Immunity to all damage (hit-points-and-damage.md; magic-item-rules.md Q5 settled), and `environment-hazards-and-objects.md` (Objects) reads these facts (Q22). | SRD 5.2.1, "Magic Item Resilience", p. 206 | unhappy (rule interaction) |
| FR-088 | `AttunementPrerequisite` MUST be closed with exactly `NoPrerequisite`, `ClassPrerequisite([]ClassID)` (a non-empty set, else a validation error), `SpellcasterPrerequisite` and `DescribedPrerequisite(text)` (a non-blank verbatim text, for any prerequisite that is neither a class nor a Spellcaster). The class and spellcaster forms are the two the SRD states: a class prerequisite is met by membership of that class and a spellcaster prerequisite by being able to "cast at least one spell using its traits or features, not by using a magic item or the like"; evaluating them, and reading an `Attunement` text into one of these forms, is behaviour of magic-item-rules.md (its FR-044 to FR-046 and FR-110). This type does not replace `Attunement` (FR-018), which keeps the printed text. | SRD 5.2.1, "Magic Item Rules" (Attunement Prerequisites), p. 205; "Attunement", p. 102 | happy |
| FR-089 | `AttunementFacts` MUST expose exactly these constants: `standardLimit` 3 ("no more than three magic items at a time"), `maxCopiesPerItem` 1 ("you can't attune to more than one copy of an item"), `endingDistanceFeet` 100 and `endingHoursAway` 24 (Attunement ends if the item "has been more than 100 feet away for at least 24 hours"). They are numbers only; comparing against them, raising the limit for a feature and ending Attunement are behaviour of magic-item-rules.md (its FR-048 to FR-053). | SRD 5.2.1, "No More Than Three Items", "Ending Attunement", p. 102; "Attunement", p. 177 | happy |
| FR-090 | `ItemRechargeTime` MUST be closed with exactly `RechargesAtNextDawn`, `RechargesAtSpecifiedTime` and `DoesNotRecharge`, and `ChargeRecovery` closed with exactly `RegainsAll`, `RegainsFixed(n)` (n of at least 1, else a validation error), `RegainsRolled(dice.Expression)` and `NoChargeRecovery` (the item with no stated recovery). Both are data only: no member computes an amount, because the amount regained depends on the item's charges, which `content` never reads (`ChargeRecoveryRules` in magic-item-rules.md, its FR-034). The SRD states only that items recharge "at the next dawn or some other specified time"; the amounts are item text (magic-item-rules.md Q13). | SRD 5.2.1, "Charges", "The Next Dawn", p. 206 | happy |
| FR-091 | `MagicItemFacts` MUST be an immutable value object keyed by `MagicItemID` with exactly these fields: `consumable`, `arcaneFocus`, `nonmagicalQuarterstaff`, `mustBeWorn` (each a `BoolFactOverride`); `activationTime` (`ActivationTimeOverride`); `curse` (`CurseStatus`: closed `NotCursed`, `Cursed`); `resilience` (`ResilienceOverride`); `allowedSpells` (a copied `[]SpellID` (a set)); `baseItem` (`BaseItemSpec`: closed `SpecificBaseItems([]ItemID)`, non-empty else a validation error, `UnspecifiedBaseItem`, `NoBaseItem`); `incorporatedItem` (closed `NoIncorporatedItem`, `IncorporatedItem(itemId, costCopperPieces)` with a non-negative cost, else a validation error); `craftingBasis` (`MagicItemCraftingBasis`: closed `TableCrafting`, `EquipmentRulesCrafting(ItemID)`); `rechargeTime` (`ItemRechargeTime`); `chargeRecovery` (`ChargeRecovery`); and `fit` (`ItemFit`: enum `adjusts`, `fixed`). There is no generic `FactOverride[T]` (plan item P8, user decision 2026-10-18; go.md section 6): the three override types are concrete, `BoolFactOverride` (`StandardBoolFact()`, `OverriddenBoolFact(v)`), `ActivationTimeOverride` (`StandardActivationTime()`, `OverriddenActivationTime(v)`) and `ResilienceOverride` (`StandardResilience()`, `OverriddenResilience(v)`), each comparable, with an accessor `Value()` returning the value and whether it is overridden, and the zero value of each is the standard fact, so no nilable field stands for a state; `NewMagicItemFacts` rejects an override of a value that is not a member of its enum. `MagicItemFacts` is built by `NewMagicItemFacts(id, MagicItemFactsFields)` (a validation error for an invalid id, member or set; a zero field takes its default: the standard fact, `NotCursed`, `NoBaseItem`, `NoIncorporatedItem`, `TableCrafting`, `DoesNotRecharge`, `NoChargeRecovery`, `adjusts`). `MagicItemFactsForEntry(entry)` MUST return the defaults of the entry's category (FR-078): every override `StandardFact`, `NotCursed`, no spells, `NoBaseItem`, `NoIncorporatedItem`, `TableCrafting`, `DoesNotRecharge`, `NoChargeRecovery` and `adjusts`. The host supplies richer facts as a value; the package never infers a fact from description text and the entry keeps charges and effects as verbatim text (Q18). The incorporated cost is an integer of Copper Pieces because `Cost` belongs to the equipment data (shared-types-and-ownership.md FR-047 forbids the import); magic-item-rules.md converts a `Cost`. | SRD 5.2.1, "Magic Items", pp. 204-206; "Cursed Items", p. 206; "Magic Item Rules" (Items Made for Specific Creatures), p. 205; "Armor", p. 204; "Weapons", p. 205 | unhappy (missing content) |
| FR-092 | `SentientCommunication` MUST be the enum `transmitsEmotion`, `speaksLanguages` and `speaksLanguagesAndTelepathy`, and `SentientCommunicationForRoll(roll)` MUST map a d10 as 1 to 6 `transmitsEmotion`, 7 to 9 `speaksLanguages` and 10 `speaksLanguagesAndTelepathy`; 0 and 11 MUST return an error wrapping `srderr.ErrInvalidInput`. | SRD 5.2.1, "Sentient Item's Communication", p. 208 | unhappy (boundary) |
| FR-093 | `SentientSenses{rangeFeet, vision}` with `vision` a closed family, `StandardVisionOnly` and `StandardVisionAndSense(SpecialSense)` (the enum owned by content-schema-and-decoder.md FR-136; no local sense enum), MUST be returned by `SentientSensesForRoll(roll)` for a d4 as 1 hearing and standard vision out to 30 feet, 2 to 60 feet, 3 to 120 feet and 4 hearing and standard vision plus `the Darkvision sense` out to 120 feet; 0 and 5 MUST return an error wrapping `srderr.ErrInvalidInput`, and a range of 0 or less makes `NewSentientSenses` return an error wrapping `srderr.ErrInvalidInput`. | SRD 5.2.1, "Sentient Item's Senses", p. 208 | unhappy (boundary) |
| FR-094 | `SpecialPurposeKind` MUST be the ten-value enum, in table order, `aligned`, `bane`, `creatorSeeker`, `destinySeeker`, `destroyer`, `glorySeeker`, `loreSeeker`, `protector`, `soulmateSeeker`, `templar`, and `SpecialPurposeKindForRoll(roll)` MUST map a d10 to them in that order; 0 and 11 MUST return an error wrapping `srderr.ErrInvalidInput`. The payloads of a purpose (an opposed alignment, a creature type, a described target) and their validation are behaviour of magic-item-rules.md (its FR-095, FR-096). | SRD 5.2.1, "Sentient Item's Special Purpose", p. 208 | unhappy (boundary) |
| FR-095 | `SentientAlignmentTableForRoll(roll)` MUST map a d100 (100 is "00") to an `Alignment` by table "Sentient alignment bands" and MUST NOT return `unaligned`; 0 and 101 MUST return an error wrapping `srderr.ErrInvalidInput`. `Alignment` is owned by content-schema-and-decoder.md (shared-types-and-ownership.md FR-053). That a sentient item's alignment is never `unaligned` when host-chosen is magic-item-rules.md FR-091. | SRD 5.2.1, "Sentient Item's Alignment", p. 208 | unhappy (boundary) |
| FR-096 | `SentientDemand` MUST be the four-value enum `chaseMyDreams`, `getRidOfIt`, `itsTimeForAChange` and `keepMeClose` ("Chase My Dreams", "Get Rid of It", "It's Time for a Change", "Keep Me Close"), and `SentientRefusalPower` exactly `preventAttunement`, `suppressProperties` and `attemptControl` ("Make it impossible for its bearer to attune to it", "Suppress one or more of its activated properties", "Attempt to take control of its bearer"), and no others. | SRD 5.2.1, "Conflict", p. 208 | happy |
| FR-097 | The types and tables of this section (`MagicItemCategoryFacts`, `ActivationTime` and its variants, `PotionMiscibilityTable`, `PotionMiscibilityBand`, `PotionMiscibilityEffect` and its eight variants, `ExplosionFacts`, `ExplosionShape`, `ExplosionOrigin`, `MixtureChoice`, `RollRangeError`, `ItemRarityFacts`, `MagicItemValue`, `PricedMagicItemValue`, `PricelessMagicItemValue`, `MagicItemCraftingRow`, `MagicItemCraftingRowLookup`, `MagicItemCraftingRowFound`, `MagicItemCraftingRowAbsent`, `MagicItemToolRule` and its variants, `MagicItemResilience`, `MagicItemResilienceFacts`, `AttunementPrerequisite` and its variants, `AttunementFacts`, `ItemRechargeTime` and its variants, `ChargeRecovery` and its variants, `MagicItemFacts`, `MagicItemFactsFields`, `BoolFactOverride`, `ActivationTimeOverride`, `ResilienceOverride`, `CurseStatus`, `NotCursed`, `Cursed`, `BaseItemSpec` and its variants, `IncorporatedItemSpec` (the closed family of `NoIncorporatedItem` and `IncorporatedItem`), `MagicItemCraftingBasis`, `TableCrafting`, `EquipmentRulesCrafting`, `ItemFit`, `SentientCommunication`, `SentientSenses`, `SentientVision` (the closed family of `StandardVisionOnly` and `StandardVisionAndSense`), `SpecialPurposeKind`, `SentientDemand`, `SentientRefusalPower`) MUST live in the magic-item facts files of package `content`, be immutable (every collection copied, equal by value or by `Equal`), synchronous and pure, import nothing from `telemetry`, contain no telemetry call and no use-case type, have no wire shape (no decoder, no revision-table row, no `Kind`, no schema version) and be exported with a doc comment on each, each name unique across the package. None is a generic variant name (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`, `Found`, `NotFound`), none is a use-case result, and each is declared once, here. The only package type imports are the `content` ids and vocabulary (`ItemID`, `SpellID`, `ClassID`, `DamageType`, `Alignment`, `SpecialSense`), `MagicItemCategory`, `ItemRarity`, `MagicItemID` and the kernel `dice.Expression`. | architecture.md §2, §3, §5, §7; shared-types-and-ownership.md FR-005, FR-006, FR-017, FR-058 | happy |

Table "Category facts" (FR-078). One row per category, one test per row.

| Category | Consumable | Arcane Focus | Quarterstaff | Worn | Activation time | Source |
|---|---|---|---|---|---|---|
| armor | no | no | no | yes | MagicAction | pp. 204, 206 |
| potion | yes | no | no | no | BonusAction | p. 204 |
| ring | no | no | no | yes | MagicAction | p. 204 |
| rod | no | yes | no | no | MagicAction | p. 204 |
| scroll | yes | no | no | no | MagicAction | p. 205 |
| staff | no | yes | yes | no | MagicAction | p. 205 |
| wand | no | yes | no | no | MagicAction | p. 205 |
| weapon | no | no | no | no | MagicAction | p. 205 |
| wondrousItem | no | no | no | no | MagicAction | p. 205 |

Table "Rarity facts" (FR-083 to FR-085). Gold Pieces; hours are 8 per day.

| Rarity | Value | Crafting days | Crafting hours | Crafting cost | Source |
|---|---|---|---|---|---|
| common | 100 | 5 | 40 | 50 | pp. 206, 207 |
| uncommon | 400 | 10 | 80 | 200 | pp. 206, 207 |
| rare | 4,000 | 50 | 400 | 2,000 | pp. 206, 207 |
| veryRare | 40,000 | 125 | 1,000 | 20,000 | pp. 206, 207 |
| legendary | 200,000 | 250 | 2,000 | 100,000 | pp. 206, 207 |
| artifact | priceless | no row | no row | no row | pp. 206, 207 |

Table "Magic Item Tools" (FR-086). Ids are the `ItemID`s of the equipment `Tool` entries.

| Category | Rule | Source |
|---|---|---|
| armor | `ByBaseItemTools` (Leatherworker's, Smith's or Weaver's Tools, by kind of armor) | p. 207 |
| potion | `FixedTools` {`alchemists_supplies`, `herbalism_kit`}, either | p. 207 |
| ring | `FixedTools` {`jewelers_tools`} | p. 207 |
| rod, staff, wand | `FixedTools` {`woodcarvers_tools`} | p. 207 |
| scroll | `FixedTools` {`calligraphers_supplies`} | p. 207 |
| weapon | `ByBaseItemTools` (Leatherworker's, Smith's or Woodcarver's Tools, by kind of weapon) | p. 207 |
| wondrousItem | `TinkersOrBaseItemTools` (`tinkers_tools` or the tool for the base nonmagical item) | p. 207 |

Table "Resilience facts" (FR-087). One test per row; source p. 206.

| Label | Category | Rarity | Expected | Path |
|---|---|---|---|---|
| ring | ring | uncommon | resistantToAllDamage | happy |
| plate armor | armor | rare | resistantToAllDamage | happy |
| potion | potion | common | nonmagicalDurability | unhappy (rule interaction) |
| scroll | scroll | rare | nonmagicalDurability | unhappy (rule interaction) |
| artifact wondrous item | wondrousItem | artifact | impervious | unhappy (boundary) |
| artifact potion | potion | artifact | impervious | unhappy (rule interaction) |

Table "Sentient alignment bands" (FR-095). Test each edge of every band; source p. 208; path unhappy (boundary).

| Roll band | Alignment |
|---|---|
| 1 to 15 | lawfulGood |
| 16 to 35 | neutralGood |
| 36 to 50 | chaoticGood |
| 51 to 63 | lawfulNeutral |
| 64 to 73 | neutral |
| 74 to 85 | chaoticNeutral |
| 86 to 89 | lawfulEvil |
| 90 to 96 | neutralEvil |
| 97 to 100 (the table's "00") | chaoticEvil |
| 0 or 101 | a validation error (unhappy, invalid input) |

### 4.9 Process requirements (not tested by CI)

| Id | Step |
|---|---|
| PR-1 | The golden (FR-074) is transcribed by someone, or something, that did not see the extraction tool's draft. A difference is resolved against the PDF, never by choosing one side. |
| PR-2 | Before the change merges and before each release, `tools/check_magic_item_pages` is run on the real PDF text (produced in Docker with poppler, as content-equipment.md PR-2 describes, so the text never enters the repository) and its empty report is recorded in the change summary. The drift of the data against a fresh extraction run (content-spells.md FR-078) is checked locally the same way. |
| PR-3 | Anything that looks like a magic item but is absent from the PDF is listed in the change summary (FR-054); so are the counts of FR-020, FR-021 and FR-046 if the first extraction run corrected any. |
| PR-4 | A person compares each entry with the rendered PDF page and records the review (item, page, reviewer, date) in the extraction notes under `doc/research/` (Markdown only, carries the attribution and is not a Go package, shared-types-and-ownership.md FR-033), together with every source oddity found (content-spells.md FR-080). |
| PR-5 | The change that adds the bundled data leaves `NOTICE.md` listing exactly the covered path (`content/data/`) and states that magic items were restructured (header split into fields, tables into rows, text into blocks); content-licensing-and-notice.md FR-005 defines the path list. The path-equality check is architecture-guardrails.md FR-079. |

### 4.10 Withdrawn ids

None. (No requirement of this spec has been withdrawn or moved.)

### 4.11 Case tables

#### Cases: Id derivation (one test per row; FR-006)

| Label | Printed name | Expected id | Source | Path |
|---|---|---|---|---|
| plain | Adamantine Armor | `adamantine_armor` | p. 209 | happy |
| bonus list | Ammunition, +1, +2, or +3 | `ammunition_1_2_or_3` | p. 209 | happy |
| bonus list, armor | Armor, +1, +2, or +3 | `armor_1_2_or_3` | p. 210 | happy |
| bonus list, wand | Wand of the War Mage, +1, +2, or +3 | `wand_of_the_war_mage_1_2_or_3` | p. 251 | happy |
| hyphen | Arrow-Catching Shield | `arrow_catching_shield` | p. 211 | happy |
| parenthetical | Stone of Good Luck (Luckstone) | `stone_of_good_luck_luckstone` | p. 248 | unhappy (boundary) |
| plural | Potions of Healing | `potions_of_healing` | p. 236 | happy |
| longest name | Amulet of Proof against Detection and Location | `amulet_of_proof_against_detection_and_location` (46 characters) | p. 209 | unhappy (boundary) |
| article inside | Horseshoes of a Zephyr | `horseshoes_of_a_zephyr` | p. 226 | happy |
| apostrophe (synthetic) | `A’s B` | `as_b` | FR-006 | unhappy (boundary) |
| 65 characters (synthetic) | a name that slugs to 65 characters | the data test fails; a document with that id is `invalidValue` at `/magicItems/0/id` | content-schema-and-decoder.md FR-030 | unhappy (boundary) |

#### Cases: Header forms (one test per row; FR-012 to FR-019, FR-063)

Each row is parsed by the tool grammar (FR-063) and decoded as wire JSON; both give the stated value.

| Label | Item, page | Printed header line | Category / note | Rarity | Attunement | Path |
|---|---|---|---|---|---|---|
| single, no extras | Immovable Rod, p. 227 | Rod, Uncommon | Rod | Single Uncommon | none | happy |
| single, wondrous | Bag of Holding, p. 212 | Wondrous Item, Uncommon | Wondrous Item | Single Uncommon | none | happy |
| attunement | Amulet of Health, p. 209 | Wondrous Item, Rare (Requires Attunement) | Wondrous Item | Single Rare | Required | happy |
| artifact | Dragon Orb, p. 218 | Wondrous Item, Artifact (Requires Attunement) | Wondrous Item | Single Artifact | Required | happy |
| category note | Adamantine Armor, p. 209 | Armor (Any Medium or Heavy, Except Hide Armor), Uncommon (wrapped over two lines) | Armor / "Any Medium or Heavy, Except Hide Armor" | Single Uncommon | none | unhappy (boundary: wrapped header) |
| category note, weapon | Berserker Axe, p. 213 | Weapon (Battleaxe, Greataxe, or Halberd), Rare (Requires Attunement) | Weapon / "Battleaxe, Greataxe, or Halberd" | Single Rare | Required | happy |
| category note, plate | Armor of Invulnerability, p. 210 | Armor (Plate Armor), Legendary (Requires Attunement) | Armor / "Plate Armor" | Single Legendary | Required | happy |
| rarity varies | Belt of Giant Strength, p. 213 | Wondrous Item, Rarity Varies (Requires Attunement) | Wondrous Item | Varies | Required | happy |
| rarity varies, potion | Potions of Healing, p. 236 | Potion, Rarity Varies | Potion | Varies | none | happy |
| rarity varies, scroll | Spell Scroll, p. 244 | Scroll, Rarity Varies | Scroll | Varies | none | happy |
| rarity per bonus | Ammunition, +1, +2, or +3, p. 209 | Weapon (Any Ammunition), Uncommon (+1), Rare (+2), or Very Rare (+3) | Weapon / "Any Ammunition" | By variant: (Uncommon, +1), (Rare, +2), (Very Rare, +3) | none | happy |
| rarity per bonus, armor | Armor, +1, +2, or +3, p. 210 | Armor (Any Light, Medium, or Heavy), Rare (+1), Very Rare (+2), or Legendary (+3) | Armor / "Any Light, Medium, or Heavy" | By variant: (Rare, +1), (Very Rare, +2), (Legendary, +3) | none | happy |
| rarity per bonus, shield | Shield, +1, +2, or +3, p. 243 | Armor (Shield), Uncommon (+1), Rare (+2), or Very Rare (+3) | Armor / "Shield" | By variant | none | happy |
| rarity per type | Horn of Valhalla, p. 226 | Wondrous Item, Rare (Silver or Brass), Very Rare (Bronze), or Legendary (Iron) | Wondrous Item | By variant: (Rare, "Silver or Brass"), (Very Rare, "Bronze"), (Legendary, "Iron") | none | unhappy (boundary: variant text with "or") |
| variants and attunement | Wand of the War Mage, +1, +2, or +3, p. 251 | Wand, Uncommon (+1), Rare (+2), or Very Rare (+3) (Requires Attunement by a Spellcaster) | Wand | By variant | RequiredBy "a Spellcaster" | unhappy (rule interaction: two parentheticals) |
| attunement by a class | Hat of Many Spells, p. 224 | Wondrous Item, Very Rare (Requires Attunement by a Wizard) | Wondrous Item | Single Very Rare | RequiredBy "a Wizard" | happy |
| attunement by several classes | Necklace of Prayer Beads, p. 233 | Wondrous Item, Rare (Requires Attunement by a Cleric, Druid, or Paladin) | Wondrous Item | Single Rare | RequiredBy "a Cleric, Druid, or Paladin" | unhappy (boundary: commas) |
| attunement by six classes | Staff of Charming, p. 245 | Staff, Rare (Requires Attunement by a Bard, Cleric, Druid, Sorcerer, Warlock, or Wizard) | Staff | Single Rare | RequiredBy "a Bard, Cleric, Druid, Sorcerer, Warlock, or Wizard" | unhappy (boundary: long prerequisite) |
| attunement by species | Dwarven Thrower, p. 219 | Weapon (Warhammer), Very Rare (Requires Attunement by a Dwarf or a Creature Attuned to a Belt of Dwarvenkind) | Weapon / "Warhammer" | Single Very Rare | RequiredBy "a Dwarf or a Creature Attuned to a Belt of Dwarvenkind" | unhappy (rule interaction: not a class) |
| attunement by a Paladin | Holy Avenger, p. 226 | Weapon (Any Simple or Martial), Legendary (Requires Attunement by a Paladin) | Weapon / "Any Simple or Martial" | Single Legendary | RequiredBy "a Paladin" | happy |
| name keeps parenthetical | Stone of Good Luck (Luckstone), p. 248 | Wondrous Item, Uncommon (Requires Attunement) | Wondrous Item | Single Uncommon | Required | happy |
| unknown rarity | synthetic | Wondrous Item, Mythic | n/a | n/a | n/a | tool: `unparsedHeader`; decoder: `invalidValue` at `/rarity/rarity` (unhappy, invalid input) |
| missing rarity | synthetic | Wondrous Item | n/a | n/a | n/a | tool: `unparsedHeader` (unhappy, missing content) |
| unknown category | synthetic | Gadget, Rare | n/a | n/a | n/a | tool: `unparsedHeader`; decoder: `invalidValue` at `/category` (unhappy, invalid input) |

#### Cases: Decoder rows (one test per row; FR-012 to FR-019, FR-037 to FR-042)

Paths are relative to `/magicItems/0` unless a different index is named; the entry is otherwise valid.

| Label | Fields | Expected | Source | Path |
|---|---|---|---|---|
| minimal | `id`, `name`, `category: "Rod"`, `rarity: {kind: single, rarity: Uncommon}`, one paragraph | accepted; no note, no Attunement | FR-040 | unhappy (empty and minimal) |
| category missing | no `category` | `missingField` at `category` | FR-012 | unhappy (missing content) |
| category plural | `category: "Wondrous Items"` | `invalidValue` at `category` | FR-012 | unhappy (invalid input) |
| category case | `category: "armor"` | `invalidValue` at `category` | FR-012 | unhappy (invalid input) |
| note blank | `categoryNote: " "` | `invalidValue` at `categoryNote` | FR-013 | unhappy (invalid input) |
| rarity word | `rarity: {kind: single, rarity: "Very rare"}` | `invalidValue` at `rarity/rarity` | FR-017 | unhappy (invalid input) |
| rarity kind | `rarity: {kind: "range"}` | `invalidValue` at `rarity/kind` | FR-017 | unhappy (invalid input) |
| rarity as string | `rarity: "Rare"` | `wrongType` at `rarity` only | FR-017, FR-042 | unhappy (rule interaction) |
| varies with a rarity | `rarity: {kind: varies, rarity: Rare}` | `unknownField` at `rarity/rarity` | FR-017 | unhappy (invalid input) |
| one option | `{kind: byVariant, options: [one]}` | `outOfRange` at `rarity/options` | FR-017 | unhappy (boundary) |
| two options | `{kind: byVariant, options: [(Rare, +1), (Very Rare, +2)]}` | accepted | FR-016 | unhappy (boundary) |
| repeated variant | options `+1`, `+1` | `duplicate` at `rarity/options/1/variant` | FR-017 | unhappy (invalid input) |
| blank variant | option `variant: ""` | `invalidValue` at `rarity/options/0/variant` | FR-017 | unhappy (invalid input) |
| attunement required | `attunement: {}` | accepted as `AttunementRequired` | FR-019 | happy |
| attunement by | `attunement: {by: "a Wizard"}` | accepted as `AttunementRequiredBy` | FR-019 | happy |
| attunement blank | `attunement: {by: " "}` | `invalidValue` at `attunement/by` | FR-019 | unhappy (invalid input) |
| attunement boolean | `attunement: true` | `wrongType` at `attunement` | FR-019 | unhappy (invalid input) |
| description empty | `description: []` | `outOfRange` at `description` | FR-041 | unhappy (boundary) |
| unknown key | `charges: 3` | `unknownField` at `charges` | FR-038 | unhappy (invalid input) |
| duplicate id | entries 0 and 1 with one id | `duplicate` at `/magicItems/1/id` | FR-007 | unhappy (invalid input) |
| same id, two kinds | a spell and a magic item with one id | accepted | FR-007 | unhappy (rule interaction) |
| three problems | entry 0: no `category`, `rarity` kind `range`, `description: []` | the three problems in field order, by path | FR-042 | unhappy (invalid input) |

#### Cases: Text fidelity (one test per row; FR-023 to FR-027)

| Label | Printed | Expected | Source | Path |
|---|---|---|---|---|
| footer removed | the footer "209 System Reference Document 5.2.1" below the last line | absent from every text | p. 209 | unhappy (layout artefact) |
| sentence across a page | "creates an" (p. 234) / "extradimensional hole 10 feet deep." (p. 235) | one sentence, one space | "Portable Hole" | unhappy (rule interaction) |
| line-end hyphen, join | "de-" / "termines" | `determines` | "Ammunition of Slaying", p. 209 | unhappy (boundary) |
| line-end hyphen, join | "al-" / "lowing" | `allowing` | "Apparatus of the Crab", p. 210 | unhappy (boundary) |
| minus sign kept | "1d20 − 1 cards" | text contains U+2212, unchanged | "Deck of Illusions", p. 217 | unhappy (boundary) |
| en dash kept | table cell "01–60" | `01–60` | "Amulet of the Planes", p. 209 | unhappy (boundary) |
| run-in heading, bold only | "Splash." then the text | `entry("Splash", ...)` | "Decanter of Endless Water", p. 217 | happy |
| run-in heading, bold italic | "Curse." then the text | `entry("Curse", ...)` | "Armor of Vulnerability", p. 211 | happy |
| run-in heading with parenthetical | "Anchor (Uncommon)." | `entry("Anchor (Uncommon)", ...)` | "Feather Token", p. 221 | unhappy (boundary) |
| inline bold is not a heading | "summon a **Fire Elemental**." mid-paragraph | stays in the paragraph, plain | "Brazier of Commanding Fire Elementals", p. 214 | unhappy (rule interaction) |
| bold run without a full stop at paragraph start | a bold word at a paragraph start, no full stop | `ambiguousEmphasis` | FR-066 (synthetic) | unhappy (invalid input) |

#### Cases: Anchor tables (one test per row; FR-029 to FR-033)

Row counts are data rows after the header; cells are as printed. The counts are leads read from the pages (Deck of Illusions 33 rows over pp. 217-218, Wand of Wonder Effects 18 rows in two columns of p. 252, and Staff of Fire 2 rows were re-counted against the PDF text); the golden (FR-074) pins them and a difference found at its transcription is corrected here (PR-3).

| Label | Item, page | Header | Rows | Notes | Path |
|---|---|---|---|---|---|
| destination, no title | Amulet of the Planes, p. 209 | `1d100`, `Destination` | 5 | first row `01–60`; last row `91–00`, "Random location on the Astral Plane" | happy |
| halves, even | Ammunition of Slaying, p. 209 | `1d100`, `Creature Type`, `1d100`, `Creature Type` | 10 | first row `01–10`, `Aberrations`, `51–60`, `Fey`; last row `46–50`, `Humanoids`, `91–00`, `Undead` | unhappy (rule interaction: side by side) |
| halves, no title | Armor of Resistance, p. 210 | `1d10`, `Damage Type`, `1d10`, `Damage Type` | 5 | | unhappy (rule interaction: side by side) |
| levers | Apparatus of the Crab, p. 210 | `Lever`, `Up`, `Down` | 10 | title `Apparatus of the Crab Levers` | happy |
| split table | Bag of Beans, p. 211 | `1d100`, `Effect` | 12 | continues in the right column, header repeated and dropped | unhappy (rule interaction: split table) |
| three tables | Bag of Tricks, p. 212 | `1d8`, `Creature`, `1d8`, `Creature` | 4 each | titles `Gray Bag of Tricks`, `Rust Bag of Tricks`, `Tan Bag of Tricks` | happy |
| rarity table | Belt of Giant Strength, p. 213 | `Belt`, `Str.`, `Rarity` | 5 | no title; last row ends `Legendary` | happy |
| faces | Cube of Force, p. 216 | `Spell`, `Charge Cost` | 6 | title `Cube of Force Faces` | happy |
| long split table | Deck of Illusions, pp. 217-218 | `1d100`, `Illusion*` | 33 | header repeated on p. 218 and dropped; last row `97–00`, `The card drawer`; footnote paragraph follows | unhappy (rule interaction: split table, footnote) |
| rarity per token | Feather Token, p. 221 | `1d100`, `Token`, `Rarity` | 6 | title `Feather Tokens` | happy |
| split across pages | Robe of Useful Items, p. 240 (table continues on p. 241) | `1d100`, `Patch` | 13 | 10 rows on p. 240, 3 on p. 241, header repeated and dropped; first row `01–08`, `Bag of 100 GP`; last row `97–00`, `Portable Ram` | unhappy (rule interaction: split table) |
| split, 11 rows | Hat of Many Spells, p. 225 | `1d100`, `Effect` | 11 | first row `01–50`; last row `96–00` | unhappy (rule interaction: split table) |
| four columns | Horn of Valhalla, p. 226 | `1d100`, `Horn Type`, `Spirits`, `Requirement` | 4 | | happy |
| table on the next page | Manual of Golems, p. 229 (table on p. 230) | `1d20`, `Golem`, `Time`, `Cost` | 4 | entry cites p. 229 | unhappy (boundary: page of the heading) |
| three columns with dashes | Mysterious Deck, p. 231 | `1d100 (13-Card Deck)`, `1d100 (22-Card Deck)`, `Card` | 22 | row `Balance`: `—`, `01–05`; row `Talons`: `—`, `88–91` | unhappy (boundary: dash cells) |
| table of healing | Potions of Healing, p. 236 | `Potion`, `HP Regained`, `Rarity` | 4 | see "Cross-kind pairs" | happy |
| odd half | Staff of Fire, p. 245 | `Spell`, `Charge Cost`, `Spell`, `Charge Cost` | 2 | row 2 is `Fireball`, `3`, ``, `` | unhappy (boundary: odd half) |
| ten levels | Spell Scroll, p. 244 | `Spell Level`, `Rarity`, `Save DC`, `Attack Bonus` | 10 | `Cantrip` to `9`; header repeated on the right column and dropped | unhappy (rule interaction: split table) |
| long list | Wand of Wonder, p. 251 (table on p. 252) | `1d100`, `Effect` | 18 | title `Wand of Wonder Effects`; first row `01–20`; last row `98–00` | unhappy (rule interaction: split table) |

#### Cases: Spot-check entries (one test per row; FR-077)

| Label | Entry, page | Blocks | Path |
|---|---|---|---|
| list in the middle | `ring_of_animal_influence`, p. 237 | `paragraph`, `bulletList` of `Animal Friendship`, `Fear (affects Beasts only)`, `Speak with Animals` (2 blocks) | happy |
| run-in headings | `decanter_of_endless_water`, p. 217 | `paragraph`, `paragraph`, `entry("Splash")`, `entry("Fountain")`, `entry("Geyser")` (5 blocks) | happy |
| list between entries | `rod_of_alertness`, p. 241 | `paragraph`, `entry("Alertness")`, `entry("Spells")`, `bulletList` of 4, `entry("Protective Aura")` (5 blocks) | happy |
| stat block in the middle | `figurine_of_wondrous_power`, p. 222 | exactly one `embeddedStatBlock`, after `entry("Ebony Fly (Rare)")` and before `entry("Golden Lions (Rare)")`; its first line `Giant Fly` | unhappy (rule interaction: header-less sub-entry) |
| stat block at the end | `mysterious_deck`, p. 231 | last block an `embeddedStatBlock` whose first line is `Avatar of Death`, after `entry("Void")`; `table` of 22 rows earlier | unhappy (rule interaction: header-less sub-entry) |
| no table, no list | `bag_of_holding`, p. 212 | only `paragraph` blocks | happy |

#### Cases: Cross-kind pairs (one test per row; FR-052)

| Label | Catalogue entry | Equipment entry | Check | Path |
|---|---|---|---|---|
| potion rows | `potions_of_healing`, table rows `Potion of Healing` / `2d4 + 2` / `Common`; `Potion of Healing (greater)` / `4d4 + 4` / `Uncommon`; `... (superior)` / `8d4 + 8` / `Rare`; `... (supreme)` / `10d4 + 20` / `Very Rare` | gear `Potion of Healing` (kind `equipment`, `MagicGear`) | both keys exist under their own kinds when the equipment data is bundled; row 1 names the gear's name | unhappy (rule interaction: overlap) |
| scroll | `spell_scroll`, `RarityVaries`; ten-row table | gear `Spell Scroll` family (kind `equipment`, `ScrollFamily`) and its two rows | both exist; kinds differ | unhappy (rule interaction: overlap) |
| marker implementers | `MagicItemEntry` | `MagicGear`, `ScrollFamily`, `ScrollVariant`, `Weapon`, `PlainGear` | exactly one type in the module implements `MagicItem`, `MagicItemEntry`; none of the equipment types does | unhappy (rule interaction) |

#### Cases: Headings per page (one test per row; FR-046)

| Page | N | Page | N | Page | N | Page | N | Page | N |
|---|---|---|---|---|---|---|---|---|---|
| 209 | 7 | 218 | 5 | 227 | 4 | 236 | 11 | 245 | 5 |
| 210 | 4 | 219 | 7 | 228 | 4 | 237 | 4 | 246 | 3 |
| 211 | 3 | 220 | 6 | 229 | 8 | 238 | 10 | 247 | 3 |
| 212 | 4 | 221 | 5 | 230 | 4 | 239 | 9 | 248 | 8 |
| 213 | 6 | 222 | 0 | 231 | 2 | 240 | 5 | 249 | 5 |
| 214 | 10 | 223 | 8 | 232 | 1 | 241 | 3 | 250 | 9 |
| 215 | 9 | 224 | 8 | 233 | 7 | 242 | 5 | 251 | 9 |
| 216 | 10 | 225 | 2 | 234 | 9 | 243 | 7 | 252 | 0 |
| 217 | 3 | 226 | 7 | 235 | 8 | 244 | 5 | 253 | 6 |

Sum 258. Source for every row: SRD 5.2.1, the page itself. Pages 222 and 252 (no heading: the "Giant Fly" block, the Wand of Wonder Effects table) are unhappy (boundary); the others are happy.

#### Cases: Printed order (one test per row; FR-047)

| Label | Pair (printed order) | Keys | Expected | Path |
|---|---|---|---|---|
| bonus form first | `Armor, +1, +2, or +3`, `Armor of Invulnerability` | `armor12or3`, `armorofinvulnerability` | in order (digits before letters) | unhappy (boundary) |
| shield before scroll | `Spellguard Shield`, `Spell Scroll` | `spellguardshield`, `spellscroll` | in order | unhappy (boundary) |
| article | `Horseshoes of a Zephyr`, `Horseshoes of Speed` | `horseshoesofazephyr`, `horseshoesofspeed` | in order | happy |
| war mage | `Wand of the War Mage, +1, +2, or +3`, `Wand of Web` | `wandofthewarmage12or3`, `wandofweb` | in order | happy |
| before the plural | `Potion of Growth`, `Potions of Healing` | `potionofgrowth`, `potionsofhealing` | in order | happy |
| the exception | `Potions of Healing`, `Potion of Heroism` | `potionsofhealing`, `potionofheroism` | out of order; the recorded exception | unhappy (rule interaction) |
| synthetic violation | two entries swapped | n/a | finding; the suite fails | unhappy (state error) |
| stale exception | the exception's pair now in order | n/a | the suite fails | unhappy (state error) |

## 5. Model

- **Types:** the domain value objects are those of FR-056 and the section 4.8 types of FR-097. `MagicItemEntry` fields: `id` (`MagicItemID`), `name`, `category`, `categoryNote` (nullable text), `rarity` (`RaritySpec`), `attunement` (`Attunement`), `description` (list of blocks), `source` (`SourceCitation`, optional for host content); `key` is `(KindMagicItem, id)`. This spec adds no id, block or citation type.
- **No aggregate, entity or domain event.** An entry is a value object identified by its `Key`; it never changes.
- **Invariants and where enforced:** name non-blank, description non-empty, lists copied on construction and on read (the `MagicItemEntry` constructor); at least two options, distinct non-blank variants, non-blank prerequisite (the `RarityByVariant` and `AttunementRequiredBy` constructors); id format (the `MagicItemID` constructor, content-schema-and-decoder.md FR-114); block rules (`TextBlock` constructors, content-schema-and-decoder.md FR-150; content-spells.md FR-096 is withdrawn); field presence, types, closed sets, ranges, duplicates and unknown keys (the decoder, section 4 paths and kinds); counts, goldens, pages, order, hygiene and stat-block placement are data tests, not constructor invariants, so a host's own magic item is a valid `MagicItemEntry`.
- **Failures:** bad content is a `Problem` in the joined `Problem` error; an absent entry is the port's not-found variant; a violated constructor precondition is a returned validation error (`MustX` panics); nothing here returns a `PortError`. Section 4.8 constructors and `ForRoll` return an error wrapping `srderr.ErrInvalidInput` (typed cause `*RollRangeError` for a roll); a roll can arrive from the host, so the error is returned and never panicked outside `MustX`. No new failure type crosses a port.
- **Purity:** every type is synchronous and deterministic, with no clock, randomness or I/O.

## 6. Interface

- **No new use case and no new port.** Magic items are read through `Source` (context-first, synchronous; content-source-port.md): `Lookup(ctx, Key)` returns the found or not-found variant, and `ListKind(ctx, KindMagicItem)` returns every entry in id order. The decoder (`Decoder`, `JSONDecoder`) is synchronous and returns content problems in its error, never panicking for content.
- **Bundled access.** `data.BundledSource` (content-source-port.md FR-059) serves the catalogue from the embedded file of FR-049; a decode failure is a `BundledContentInvalid` error (FR-050).
- **Exports.** Package `content` exports the types of FR-056 (with `SingleRarity`, `RarityVaries`, `RarityByVariant`, `NoAttunement`, `AttunementRequired`, `AttunementRequiredBy`) and FR-097, each documented; `MagicItemID` is exported by the spec that owns it. `content/data` exports nothing new; the tool and golden are not exported.
- **Test doubles.** None new; no `content/contenttest` addition.
- **Dependencies.** The domain uses only `content`'s own types (typed ids, `SourceCitation`, `DamageType`, `Alignment`, the block types) and the kernel `dice.Expression` (dice-and-rolls.md, reached through the decoder spec and not a header dependency, guardrails FR-099); it does not import `telemetry`, any `Cost` or equipment entry type (shared-types-and-ownership.md FR-047), `character` or `combat`. magic-item-rules.md depends on this spec, not the reverse. Cross-references to other data are by id only and none exists (Q10).

## 7. Observability

**Silent by design.** The entry and header types are domain code, silent by rule (observability.md §3). The decoder additions are an adapter with no retries, fallbacks, partial decodes or timing the use case cannot see; their failures return to `LoadSet`, which records them once. The tool is a developer tool outside the package. This spec owns no signal constant. It adds one catalogue value to a shared signal: `srd5e.content.kind` gains `magic_item` (content-schema-and-decoder.md FR-101); `srd5e.content.entries_loaded` is existing and rises by 258 for that kind. Shared signals: content-schema-and-decoder.md FR-079 to FR-082, FR-100.

The seven evaluation questions (`doc/rules/observability.md` §2):

1. **What can go wrong?** The bundled document fails to decode; a transcription differs from the SRD (a dropped table row, a wrong rarity, text lost across a column break); an entry is not from the SRD; an item is missing or doubled; a host document uses `magicItems` where the version does not register it; a constructor receives an impossible value (a one-option `RarityByVariant`).
2. **How would an operator know?** A rejected host load: the existing `LoadSet` span ends in error status, `srd5e.content.loads` shows `srd5e.outcome=failure` and `srd5e.content.decode_errors` counts problems by `srd5e.content.problem_kind`. A bundled-data failure is the `BundledContentInvalid` error of the `BundledSource` constructor. Transcription errors do not show at run time; the golden, count, order, hygiene and PDF checks catch them before release. A returned validation error is a bug the caller sees as an error.
3. **What is needed to diagnose it?** The the joined `Problem` error problems (path and kind) the consumer holds; paths and ids are never attributes. For a transcription error, the failing golden row or PDF-check report line (entry id and page).
4. **What should be counted or timed?** Nothing new (the duration and outcome counters already cover loading).
5. **Is existing instrumentation still correct?** Yes, with the one attribute value above. No signal is renamed and no problem kind is added.
6. **What should a consumer monitor?** Any `srd5e.outcome=failure` load or non-zero `decode_errors` for shipped data (alert at 1); after an upgrade, `entries_loaded` for `magic_item` away from 258 when the full set loads.
7. **Is anything now noisy?** No. The 258 entries MUST NOT each produce a log line at load time (observability.md §7, "No hot-loop logging").

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.content.entries_loaded` (existing) | counter | `srd5e.content.kind` gains the value `magic_item` | Entries loaded per kind | a change from 258 after an upgrade |

Entries here feed `doc/observability/signals.md`: the one attribute value above.

## 8. Test plan

No double is needed for the domain or decoder. FR-050 and FR-051 use the real `BundledSource` and `MemorySource`, `contentSourceContractChecks` (content-source-port.md FR-028) and `contentSourceReadOnlyChecks` (FR-058 there). Fixtures: the golden under `testdata/srd/` (FR-074), synthetic page text for the tool tests, small synthetic entries. Levels follow testing.md §3.

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002, FR-004, FR-005, FR-008, FR-009, FR-011 to FR-019 | unit | `magic_item_test.go` (`package content_test`): key kind; `id` is a `MagicItemID` and an `ItemID` cannot be passed (compile-time check; the id-format boundaries are the nine-id table of content-schema-and-decoder.md FR-114, not repeated); blank and untrimmed names, empty description, immutability, value equality changing one field at a time; exact enum lists and order; wire strings including rejected plurals and case; options at 1, 2, duplicate and blank variants; Attunement subtypes and wire forms. |
| FR-003 | unit + architecture | Marker has one method and no other member; `internal/archtest`: exactly one implementer in the module (case table "Cross-kind pairs", last row); no `isMagicItem` member. |
| FR-006 | unit + contract | `tools/`: the slug function over "Id derivation", one test per row. Data test: the golden's expected ids equal the bundled ids for all 258 and none exceeds 64 characters; it imports nothing from `tools/`. |
| FR-007, FR-037 to FR-044 | contract | Decoder tests over the bundled source and decoder: one test per row of "Decoder rows" and "Header forms"; duplicate id in one kind rejected and in two kinds accepted; three independent problems in one document; determinism and no input mutation; decoder and constructor agreement over the same rows. |
| FR-010, FR-020, FR-021, FR-045, FR-046, FR-047, FR-048 | contract | Data tests: document, section and page range; heading-page rule on the continuing entries; the seven, six and 245 rarity forms and the category counts by name list; golden size; per-page counts ("Headings per page"); printed order with the recorded exception and a stale-exception row. |
| FR-022 to FR-036, FR-077 | contract | "Text fidelity" rows on synthetic fixtures; "Anchor tables" and "Spot-check entries" asserted on the bundled data (cells, split and side-by-side rows); Giant Fly and Avatar of Death placement and lines; `listKind(KindMonster)` holds no entry named `Giant Fly` or `Avatar of Death`; no `monsters` key and no `monster` entry in the catalogue document. |
| FR-049, FR-050, FR-052 | architecture + contract | `data.Files()` lists the stem `magic_items` and its bytes are the document the data tests decode (no drift test exists: architecture-guardrails.md FR-078; content-schema-and-decoder.md FR-123); no `.json` outside `content/data/` and `testdata/`; `BundledSource` found for each key, not-found for an unknown id and another kind, `ListKind` equal to the golden in id order; `BundledContentInvalid` for corrupted bytes passed through the constructor's documents parameter (content-source-port.md FR-059); "Cross-kind pairs": both ids present under their own kinds, the four potion rows. |
| FR-051 | contract | `contentSourceReadOnlyChecks` over `BundledSource` with the decoded catalogue as `expectedEntries`; `contentSourceContractChecks` with `MemorySource`. |
| FR-053, FR-074, FR-075 | contract | Hygiene scan over every text, cell and line (a synthetic corrupted copy for each item MUST make it fail); the golden against the bundled data (a deliberately altered copy of each compared field MUST fail; missing and extra entries). |
| FR-054, FR-055, FR-056, FR-097 | architecture + unit | FR-054 is review and the PDF check (PR-2); in CI only that the check tool fails on synthetic text with an extra entry (not run on the real PDF: it is not in the repository). Consumer-name scan over the module, `tools/` and the data. Placement, purity and export scans (AST scan for duplicate and generic names, architecture-guardrails.md FR-077); no import of `telemetry`, `character` or `combat`; no telemetry call; no wire shape or revision-table row for the section 4.8 types; the decoder additions add no exported name. The consumer-name scan is an architecture check; FR-057 is the scenario row below. |
| FR-057 | scenario | `scenarios/`, public entry points only: build the bundled source, look up `bag_of_holding` and `amulet_of_health`. |
| FR-058 to FR-073 (tool) | unit | `tools/`: fixtures from pp. 209, 210, 211, 214, 217, 222, 231, 232, 245 as synthetic page text; every report kind (`unparsedHeading`, `unparsedHeader`, `columnOrderSuspect`, `ambiguousEmphasis`, `lineEndHyphen`, unexpected PDF) reachable; byte-identical rerun; no data file on any problem; FR-058 also an architecture check (no dependency added to the root `go.mod`; no import of `tools/` from the root module, architecture-guardrails.md FR-076; the draft path is outside `content/data/`; `tools/` imports only public packages); FR-068 the pure log-applying function (applied, stale `staleCorrection`, unrecorded edit); FR-071 an in-memory writer, any other path is refused by the writer factory with an error; FR-072 a reader double records that the golden path is never opened. |
| FR-076 | unit | `tools/check_magic_item_pages` on synthetic page text (four rows of the requirement); exit status. |
| FR-078 to FR-079, FR-083 to FR-090 | unit | `magic_item_facts_test.go`: "Category facts" one test per row, exhaustive `switch`; `ActivationTime` has five variants and `DescribedElsewhere` is accepted for every category; "Rarity facts" six rows once each in order, `PricedMagicItemValueFromGold` against Copper Pieces, a negative amount returns an error, five crafting rows with `workHours` 8 times `days`, Artifact row absent and value priceless; "Magic Item Tools" one test per category, an empty `FixedTools` set returns an error, scenario (`scenarios/`) that every `FixedTools` id exists as a `Tool` when equipment data is bundled; "Resilience facts" every row, Artifact wins in every category; the four prerequisite forms and constructors (empty class set, blank text), the four `AttunementFacts` constants, `ItemRechargeTime` and `ChargeRecovery` variants, `RegainsFixed(0)` returns an error. |
| FR-080, FR-081, FR-082 | unit | Every band edge (1, 2, 8, 9, 15, 16, 25, 26, 35, 36, 90, 91, 99, 100) plus 0, 101 and -1 (`ForRoll` returns `*RollRangeError`, `MustForRoll` panics with it); exhaustive over 1 to 100 that each roll has exactly one band; a band with `minRoll` above `maxRoll` returns an error; typed facts of each variant (explosion origin `mixture`, radius 5, sphere, `4d10`, Force; `hostChoices` per variant; halved and doubled flags). |
| FR-091 | unit | `MagicItemFactsForEntry` defaults for one entry of each category; each override type, `BaseItemSpec` and `IncorporatedItem` constructor boundary; collections copied on construction and on read; value equality changing one field at a time. |
| FR-092 to FR-096 | unit | Every roll edge of communication (1, 6, 7, 9, 10), senses (1 to 4), special purpose (1 to 10, `glorySeeker` spelling) and alignment tables (every band edge) and out-of-range inputs; `unaligned` is never returned; the exact demand and refusal-power sets. |

Coverage and mutation:

- Every changed line in the domain files must be covered (testing.md §7). The domain files MUST be mutation tested with `gomu` (run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors; testing.md §8) and a score of at least 80; the decoder additions and the tool's parsers SHOULD be.
- Observability tests: none (observability.md §9). Architecture tests: the existing guardrails apply; additions are the marker-implementer scan (FR-003) and the `tools/` reachability checks (FR-058).
- Deliberately not covered by an automated test: PR-1 to PR-5 and the real-PDF run of the checks, which CI cannot see.

**Implementation status.**

- Done (Phase 2, step 5d, package `content`, files `magic_item.go`, `magic_item_category.go`, `magic_item_rarity.go`, `magic_item_attunement.go`, `decode_magic_item.go`, `decode_text_block.go` and the section 4.8 files `magic_item_category_facts.go`, `magic_item_potion_miscibility.go`, `magic_item_rarity_facts.go`, `magic_item_tools.go`, `magic_item_resilience.go`, `magic_item_attunement_facts.go`, `magic_item_recharge.go`, `magic_item_facts.go`, `magic_item_sentient.go`, `magic_item_roll_error.go`): the entry, marker, category, rarity and Attunement types, the `magicItems` decoder row (registered in `rootArrays()` last, after the equipment arrays and `monsters`), the wire form of `TextBlock`, and every type and table of section 4.8. Tested on synthetic entries: FR-001 to FR-005, FR-007 to FR-009, FR-011 to FR-019, FR-022, FR-037 to FR-044, FR-056 and FR-078 to FR-097 (FR-056 and FR-097 by the existing import, variable, doc and variant checks, and FR-003 also by `internal/archtest/magic_item_marker_test.go`).
- Choices the spec left open: the constructors take a keyed fields struct (`MagicItemFields`, `MagicItemFactsFields`); a nil `Attunement` is `NoAttunement` and an empty `CategoryNote` is absent; the constants carry their type's prefix (`MagicItemRod`, `RarityVeryRare`, `ActivationMagicAction`, `ResilienceImpervious`, `SentientDemandKeepMeClose`, `SpecialPurposeGlorySeeker`) so that `Action`, `Reaction` and `Armor` stay out of the package namespace; `ActivationTime`, `ItemRechargeTime`, `CurseStatus`, `ItemFit` and the small vocabularies are enums, the families with payloads are sealed interfaces; the lookup types that the spec spells `X{}.Of(...)` are zero-size structs (`MagicItemCategoryFacts`, `ItemRarityFacts`, `MagicItemResilienceFacts`, `AttunementFacts`, `PotionMiscibilityTable`) except the tools, which are `MagicItemToolTable{}.Of` because `MagicItemToolRule` is the interface; a value that is not a member gets the zero facts (or nil for tools, 0 for the resilience); `ExplosionShape` and `ExplosionOrigin` are one-member enums, `IncorporatedItemSpec` and `SentientVision` name the closed families whose members the spec lists; `SentientAlignmentTable` is the function `SentientAlignmentTableForRoll` only; the explosion damage and all tables are built in functions and `switch`es, never package variables (plan item P6); amounts are `int64` Copper Pieces; `FactOverride[T]` does not exist (plan item P8): `BoolFactOverride`, `ActivationTimeOverride` and `ResilienceOverride` replace it; a rarity option or a table row is checked on the decoder side by the same constructor, and a table row is width-checked only when its header decoded.
- Mutation status (`just mutate "./content/<file>.go"`, one file at a time, capped at a quarter of the processors; score is killed over killed plus survived, not-viable mutants excluded): `decode_magic_item.go` 92.0, `decode_text_block.go` 90.5, `magic_item.go` 88.6, `magic_item_attunement.go` 85.7, `magic_item_attunement_facts.go` 100, `magic_item_category.go` 100, `magic_item_category_facts.go` 100, `magic_item_facts.go` 81.0, `magic_item_potion_miscibility.go` 100, `magic_item_rarity.go` 91.7, `magic_item_rarity_facts.go` 83.6, `magic_item_recharge.go` 93.8, `magic_item_resilience.go` 82.6, `magic_item_roll_error.go` 100, `magic_item_sentient.go` 93.5, `magic_item_tools.go` 100; some of these ran before the last test additions. Several survivors reported by gomu (for example the `&&` chain of `MagicItemFacts.Equal`, the `return false` of the validity helpers) were each broken by hand and failed the suite, so gomu's survivor list is noisy; the rest are equivalent (`==` against 0 or "" and `>=` on the highest enum member, marked in the code).
- Manual-mutation pass (2026-10-19, hand-applied mutants, narrow `go test ./content`): `magic_item_roll_error.go` 11 mutants all killed, `magic_item_category.go` 12 of 12 killed, `magic_item_resilience.go` 16 of 17 killed (survivor `r >= RarityArtifact` is equivalent, marked in code), `magic_item_attunement.go` 11 of 11 killed. Added `content/magic_item_small_test.go` for the resilience String names, the zero value returned with a constructor error, and the RollRangeError text.
- Manual-mutation pass 2 (2026-10-19, hand-applied mutants, `go test ./content`; these four have no same-named test, so gomu gives no output): `magic_item_tools.go` 15 of 15 killed, `magic_item_category_facts.go` 20 of 20 killed (2 more did not compile), `magic_item_attunement_facts.go` 13 of 13 killed, `magic_item_recharge.go` 20 of 20 killed (1 did not compile), after adding `FixedTools.Equal`, name-string, `RegainsFixed.Charges` and `ChargeRecovery` validity tests to `content/magic_item_small_test.go`; no survivors left.
- Mutation (testing.md §8, species and origins 1): `custom_background.go` gomu score 100.0% (43 killed of 49, 6 not viable, 0 survivors). `feat.go`, `species.go`, `decode_species.go` have no same-named test file, so they were mutated manually (about 25 distinct mutations each, one at a time, restored byte-identical): every viable mutation was killed except two that surfaced real gaps, now killed by `TestSpeciesChoicesAreCopiedOnRead` (`Species.Choices` must copy) and the "unknown kind with the fields of every variant" case of `TestDecodeSpeciesChoices` (`o.skip` must name `spellIds`); the remaining `==`/`<=`, `!=`/`>` mutants are the equivalent ones recorded beside them in the source.
- Manual-mutation pass 3 (2026-10-19, group "magic items 3"; gomu v0.2.1 via `just mutate`, killed over killed plus survived): `magic_item.go` 96.3 (78 of 81 killed, 3 survivors), `decode_magic_item.go` 84.6 (22 killed, 4 survivors, 30 not viable). `magic_item_rarity.go` and `magic_item_rarity_facts.go` have no same-named test, so 31 hand-applied mutants over both files (flipped comparisons, dropped branches, changed constants and table rows) ran against `go test ./content`: all killed except `raritySpecEqual` returning true for a non-by-variant versus by-variant pair; `TestMagicItemEntryEqualRarityVariesVersusByVariant` now kills it. Both files restored and verified byte-identical.
- Mutation pass 4 (2026-10-20, group "magic items 4"; gomu v0.2.1 via `just mutate`, run on all three files because their tests live in `magic_item_facts_test.go`; score is killed over killed plus survived): `magic_item_facts.go` 81.0 (132 of 171 killed, 31 survivors, 8 not viable), `magic_item_sentient.go` 93.6 (88 killed, 6 survivors, 3 not viable), `magic_item_potion_miscibility.go` 100 (57 of 57 killed); no test changes needed.
- Moved from T to D, with the reason: the scenario test of FR-086 (the `FixedTools` ids exist as `Tool` entries) because no equipment data is bundled; FR-006, FR-010 and FR-020 to FR-021 (bundled data rules); FR-044 as to the tool grammar of "Header forms" (no tool yet); FR-056's per-name AST scan beyond the existing checks.

Not yet implemented: FR-006, FR-010, FR-020, FR-021, FR-023 to FR-036, FR-045 to FR-055, FR-057 to FR-077 (the bundled data, the golden fixture, the tools under `tools/`, the data and scenario tests), the tool-grammar part of FR-044 and the scenario part of FR-086.

## 9. Open questions and decisions

All questions are settled; none is escalated. Settled (one line each; ids kept because other specs cite them):

| Id | Decision |
|---|---|
| Q2 | Description blocks are not defined here and not spell-named: the block family is a neutral `TextBlock` (paragraph, entry, bulletList, table, embeddedStatBlock) owned by content-schema-and-decoder.md, and content-spells.md re-cites it (design basis: shared-types-and-ownership.md FR-047 and FR-006, data specs share no types with each other, shared shapes sit with the schema owner). This spec depends only on the schema spec already in the header. Amendments requested of both owners. |
| Q5 | The two embedded stat blocks stay verbatim `embeddedStatBlock` text (FR-034, FR-035), as content-spells.md decided for spells. Avatar of Death cannot be held by the `Monster` schema (HP "Half the HP maximum of its summoner", CR "None", alignment "Neutral evil", SRD 5.2.1, p. 232) and Giant Fly is absent from the Index of Stat Blocks (pp. 2-4); promoting either would break content-monsters.md FR-056. |
| Q6 | Variants (Potion of Healing grades, Belts, Ioun Stones, Feather Tokens, Figurines, Horns, Spell Scroll levels) are not entries; the SRD prints each inside its parent item's text and tables (pp. 213, 221, 222, 226, 236, 244). An optional `variants` list is a later, non-breaking addition if a consumer needs it. |
| Q7 | Attunement prerequisite stays verbatim text (FR-019): the SRD prints it as prose including non-class forms ("a Dwarf or a Creature Attuned to a Belt of Dwarvenkind", p. 219). The structured host-side form is `AttunementPrerequisite` (FR-088). |
| Q8 | No cross-kind field between the gear rows and the catalogue entries for Potion of Healing and Spell Scroll (they are separate printed objects, p. 99 and pp. 236, 244); FR-052 pins that both exist, and a resolved reference is a later non-breaking addition (content-schema-and-decoder.md FR-125, FR-126). |
| Q10 | Bold creature names and italic spell and item names are not carried (FR-027, p. 209); resolved references by id are a later non-breaking addition once those sets are bundled. |
| Q13 | One file `magic_items.json` for all 258 entries (the data directory holds one file per document, content-schema-and-decoder.md FR-120; one decode keeps ids unique across the document). |
| Q15 | `categoryNote` stays verbatim text (FR-013): the SRD notes mix item names, groups and prose ("Any Simple or Martial", "Any Light, Medium, or Heavy", pp. 226, 210), which need defined equipment group words before they could be structured. |
| Q1 | `MagicItemID` is owned by content-schema-and-decoder.md FR-114, FR-128 (shared-types-and-ownership.md FR-046) and requested by row R2-magic-items (FR-002). It is already listed there (FR-114, FR-128) and goes live when R2-magic-items is applied, which sets the row to landed. Separately, the schema spec's R2-magic-items row text still calls this spec "an empty placeholder with no citable requirement" and needs updating there. |
| Q3 | Storage and parser placement: shared-types-and-ownership.md FR-030 to FR-032; content-schema-and-decoder.md FR-120 to FR-124 (FR-049, FR-058). |
| Q4 | Row R2-magic-items follows R2-monsters (content-schema-and-decoder.md FR-016); decoded entries are ordered by `Kind` declaration, `magicItem` before `monster` (content-schema-and-decoder.md FR-142; schema FR-071). |
| Q9 | Behaviour and state of the chapter rules are magic-item-rules.md's; this spec holds the vocabulary and static tables (section 4.8, shared-types-and-ownership.md FR-058). |
| Q11 | `source.section` is `Magic Items A–Z` with the printed en dash (FR-010). |
| Q12 | Side-by-side halves are stored as printed (FR-031). |
| Q14 | Completeness without an index: golden, per-page counts, order check, plus the PDF check for headings no entry cites. |
| Q16 | dice-and-rolls.md now cites the trap on p. 200 and "Cloak of Invisibility", p. 215; see Q20. |
| Q17 | Only `MagicItemEntry` implements `MagicItem` (FR-003); equipment declined the marker (content-equipment.md FR-095, Q19). |
| Q18 | One rarity type `ItemRarity`; no `MagicItemRarity`; no `facts` block on the entry; facts are the host-supplied `MagicItemFacts` (FR-091). |
| Q19 | Withdrawn question; id kept so numbering does not move (see Q17). |
| Q20 | Checked: the text `1d3` occurs on pp. 215, 216, 223, 226, 227, 229, 232, 234, 237, 238, 239, 240, 248, 250 and 251, all inside pp. 209-253; p. 200 is the Gameplay Toolbox trap. |
| Q21 | Amounts are Copper Pieces (the unit of `Cost.valueInCopper`, SRD "Coins", p. 89) because shared-types-and-ownership.md FR-047 forbids importing `Cost`; magic-item-rules.md converts. |
| Q22 | Resilience facts (FR-087) live here; magic-item-rules.md records the per-item override (its FR-073); the damage handling is settled in its Q5. |
| Q23 | `ForRoll` here is a total lookup over a static table with a range check, not a rule; the dice, rejection errors and cross-field rules stay in magic-item-rules.md. |

## 10. Changelog impact

`CHANGELOG.md`, under `Unreleased`, **Added**: "Magic item content: `MagicItemEntry`, the `MagicItem` marker, `MagicItemCategory`, `ItemRarity`, `RaritySpec` and `Attunement` (the typed id `MagicItemID` is listed with the other typed ids by content-schema-and-decoder.md); the `magicItems` document array decoded by `JSONDecoder`; the bundled SRD 5.2.1 magic items (258 entries, pp. 209-253); and the static vocabulary and tables of the magic item rules (category facts, rarity values and crafting rows, the Magic Item Tools table, resilience facts, attunement facts, Potion Miscibility, the sentient item tables and `MagicItemFacts`; section 4.8)." This is additive and not **BREAKING** (first release of these types; before 0.1.0 every schema revision lands in the single `schemaVersion`, shared-types-and-ownership.md FR-034). After release, changing the header shape, renaming a category or rarity value, or changing a bundled value for the same item (outside a correction against the SRD PDF) is a breaking change under `architecture.md` §7 and MUST be marked **BREAKING**.
