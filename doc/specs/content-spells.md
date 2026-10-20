# Content: all SRD spells

| | |
|---|---|
| Status | partially implemented |
| Context | `content` |
| Roadmap phase | Types 2; bulk data 5 |
| Depends on | `errors-and-results.md`, `content-schema-and-decoder.md`, `content-source-port.md`, `spell-effect-data.md` |

Phase rule: shared-types-and-ownership.md FR-014; see [../roadmap.md](../roadmap.md). content-thin-slice.md (same rank) is cited in prose only (FR-094), not a dependency.

## 1. Purpose and scope

Defines the `Spell` content entry (a `Entry` of kind `spell`, identified by a `SpellID`),
the class spell lists derived from `Spell.classes` (FR-120, FR-121; shared-types-and-ownership.md
FR-066), and the bulk bundling of every spell in SRD 5.2.1 ("Spell Descriptions", pp. 107-175):
the spell JSON shape and decoder verdicts (section 5.2, FR-107 to FR-111), the extraction tool
`tools/extract_spells` (a Go `main` package), and verification (pinned counts, cross-checks against the eight class
spell lists, spot-check fixtures, human review) of the 339 entries.

**Out of scope:**

- Preparing, casting and expending slots (spellcasting.md); the rules text on pp. 104-107 is used
  only for vocabulary and the closed sets of schools, levels and class names.
- Structured mechanics (damage dice, save abilities, areas, conditions applied): they stay in
  the description text except the optional `Spell.effects`, owned by spell-effect-data.md (FR-122 here, Q5); no `SpellScaling` (FR-115). The four embedded stat blocks are
  verbatim text, not monsters (Q4; content-monsters.md).
- Class and subclass content (content-classes.md; the class spell-list pages are only a second
  source for `Spell.classes`), Spell Scrolls and other magic items, the Component Pouch and
  Spellcasting Focus entries.
- Anything not in SRD 5.2.1: spells, names or class lists from another book or edition. Entry
  names are the SRD headings, with no aliases; no spell in the PDF was left out.
- Owned elsewhere: the envelope, `schemaVersion`, revision table, decoder, failure type and common
  `source` object (content-schema-and-decoder.md FR-034), `SourceCitation` (FR-115, FR-138),
  `SpellID`, `ClassID` (FR-114) and `DurationUnit` (FR-149); this spec owns only the `Spell`
  shape, row R1-spells (shared-types-and-ownership.md FR-034, FR-035). The `go:embed`
  wiring, its decode test and `data.BundledSource`: shared-types-and-ownership.md FR-030 to
  FR-032.

Notice and licensing for the bundled data are owned by content-licensing-and-notice.md, which this spec cites in prose and does not depend on (content-schema-and-decoder.md and content-source-port.md are its only content dependencies, shared-types-and-ownership.md FR-047, and from FR-122 spell-effect-data.md).


## 2. Source

SRD 5.2.1 (`doc/research/sources.md`); package decisions in shared-types-and-ownership.md. The
PDF page index equals the printed page number; pages cited are printed pages.

- "Spell Descriptions", pp. 107-175: 339 entries (Acid Arrow, p. 107, to Zone of Truth, p. 175).
  Rules vocabulary: "Spell Level" and "Spell Slots", "Casting without Slots", "Using a
  Higher-Level Spell Slot", pp. 104-105; "Schools of Magic", "Class Spell Lists", "Casting Time",
  "Range", "Components" (the tail of "Material (M)" is on p. 106), p. 105; "Duration", p. 106.
- Verification oracle, the class spell lists: Bard p. 33, Cleric p. 38, Druid p. 44, Paladin
  p. 55, Ranger p. 60, Sorcerer p. 67, Warlock p. 74, Wizard p. 79, each with School and
  Special columns (C Concentration, R Ritual, M a specific Material component).

The first extraction run MUST confirm every count (`higherLevel` was corrected from 106 to 109 in
review); a difference is investigated against the PDF page, never forced to match (FR-075, FR-084).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Spell | `Spell` (value object, `Entry`) | Key `(Kind.spell, SpellID)`; `SpellID` per content-schema-and-decoder.md FR-114 (shared-types-and-ownership.md FR-005, FR-046). |
| Spell Level, Cantrip | `SpellLevel` (0-9) | Level 0 is a cantrip, printed "<School> Cantrip" instead of "Level N <School>". |
| School of Magic | `SpellSchool` (enum) | Abjuration, Conjuration, Divination, Enchantment, Evocation, Illusion, Necromancy, Transmutation. |
| Class Spell Lists | `Spell.classes` (set of `ClassID`); `ClassSpellList`, derived by `DeriveClassSpellList` (FR-120) | The eight ids `bard`, `cleric`, `druid`, `paladin`, `ranger`, `sorcerer`, `warlock`, `wizard`. |
| Casting Time | `CastingTime` (closed, five variants, FR-026) | Printed "Action" is the Magic action (p. 105). The lower-case case-table labels are the factory constructors (FR-114). |
| (package) casting time options | `SpellCastingTimes` (closed: `Single`, `Modes`) | One time, or at least two labelled modes (Plant Growth, FR-033). |
| Ritual | `Spell.ritual` | Printed "or Ritual" in the Casting Time entry. |
| Reaction and Bonus Action Triggers | `CastingTime` trigger text | Verbatim text after "which you take". |
| Range | `SpellRange` (closed, six variants, FR-036) | Feet or miles, Self, Touch, Sight, Unlimited, Special, as printed. |
| Components, Material (M) | `SpellComponents`, `MaterialComponent` | V, S, M; M has verbatim description, stated GP minimums, whether materials are consumed. |
| Duration | `SpellDuration` (closed, five variants, FR-050) | A Time Span carries a `DurationLimit` and the schema's `DurationUnit` (FR-119). |
| Concentration | `DurationLimit.concentrationUpTo` | Printed "Concentration, up to ..."; the other limits are `fixed` ("1 hour") and `upTo` ("Up to 1 hour"). |
| Using a Higher-Level Spell Slot; Cantrip Upgrade | `Spell.higherLevel`; `Spell.cantripUpgrade` | Text only. |

## 4. Requirements

Stable ids, never renumbered; withdrawn ids are in the table at the end of this section.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | The bundled spell set MUST contain exactly 339 `Spell` entries: one for every spell description in the "Spell Descriptions" section, none duplicated. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-002 | The bundled spell set MUST contain no entry whose name is not a spell description heading in pp. 107-175, and no spell from any other source; a test MUST pin the full list of 339 names. | `AGENTS.md`, "Licensing and source material"; SRD 5.2.1, "Spell Descriptions", pp. 107-175 | unhappy (missing content: non-SRD entry) |
| FR-003 | The tool MUST derive a `SpellID` from a spell name by: lower-casing; deleting apostrophes (U+2019 and U+0027); replacing every maximal run of other non-`[a-z0-9]` characters with one underscore; trimming leading and trailing underscores. The result MUST be passed to `ParseSpellID` (content-schema-and-decoder.md FR-114), so the pattern `^[a-z][a-z0-9_]*$` and the 64-character limit live in that one type and are not repeated in the tool (FR-105); an error from `ParseSpellID`, or a name yielding an empty string, MUST be reported as a problem, not defaulted. Id convention: shared-types-and-ownership.md FR-007 (lowercase `snake_case`; content-schema-and-decoder.md FR-030; `fire_bolt` as in content-thin-slice.md FR-003). | `doc/rules/architecture.md` §3 (typed ids); shared-types-and-ownership.md FR-005, FR-007; content-schema-and-decoder.md FR-114, FR-030; SRD 5.2.1, "Spell Descriptions", pp. 107-175 (names) | happy |
| FR-004 | Every bundled spell's id MUST equal the id derived from its name by FR-003. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-005 | No two bundled spells MAY share an id, and no two distinct names MAY derive the same id; the extraction tool MUST report a collision. The data is ten documents (Q10), so one decode cannot see a duplicate across files: it MUST be rejected as a duplicate key when the decoded documents are combined into one content set (the in-memory source behind `data.BundledSource`), and a test MUST cover that construction. | `doc/rules/testing.md` §4 (duplicate entries) | unhappy (missing content: duplicate) |
| FR-006 | A bundled spell's name MUST equal the heading as printed, including capitalisation, spaces, "/" and the typographic apostrophe, with surrounding whitespace removed. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-007 | `Spell` MUST carry an optional `SourceCitation` (`document`, `section`, `page`; content-schema-and-decoder.md FR-115). A `Spell` without a citation MUST be valid, so host-supplied content is not forced to claim an SRD page. The JSON `source` object is the common one of content-schema-and-decoder.md FR-034 and decodes into `SourceCitation` (FR-107). | `doc/rules/architecture.md` §3 (house rules arrive as content); content-licensing-and-notice.md (every bundled entry cites section and page); content-schema-and-decoder.md FR-034 | unhappy (invalid input) |
| FR-008 | Every bundled spell MUST carry the citation document "SRD 5.2.1" (content-schema-and-decoder.md FR-090), the section "Spell Descriptions" and a page from 107 to 175 inclusive; the page MUST be the one on which the spell's heading is printed. | SRD 5.2.1, "Spell Descriptions", pp. 107-175; content-licensing-and-notice.md | happy |
| FR-009 | `SpellLevel` MUST accept the integers 0 to 9 and MUST reject -1, 10 and any other value outside the range with an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Spell Level", p. 104 | unhappy (boundary) |
| FR-010 | `NewSpellLevel(n)` MUST return `(SpellLevel, error)`: the level for 0 to 9 and otherwise a `*SpellLevelError` (FR-119; errors-and-results.md FR-027, FR-035) carrying the rejected integer and matching `srderr.ErrInvalidInput`, and MUST NOT panic; `MustSpellLevel` panics with that error. | `doc/rules/architecture.md` §6 (untrusted input) | unhappy (invalid input) |
| FR-011 | `SpellLevel.IsCantrip` MUST be true for level 0 and false for levels 1 to 9. | SRD 5.2.1, "Spell Level", p. 104 | unhappy (boundary) |
| FR-012 | `SpellSchool` MUST enumerate exactly the eight schools in the Schools of Magic table and nothing else. | SRD 5.2.1, "Schools of Magic", p. 105 | happy |
| FR-013 | Extraction MUST read level and school from the header line: "Level N <School>" gives level N, "<School> Cantrip" gives level 0. Any other header form MUST be reported as a problem (table: header forms). | SRD 5.2.1, "Spell Descriptions", p. 107 (Acid Arrow, Acid Splash) | unhappy (invalid input) |
| FR-014 | The bundled set MUST contain the per-level counts in the table "Distributions". | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-015 | The bundled set MUST contain the per-school counts in the table "Distributions". | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-016 | The class lists of a bundled spell MUST use only the eight `ClassID`s `bard`, `cleric`, `druid`, `paladin`, `ranger`, `sorcerer`, `warlock`, `wizard`; the closed set is a bundle-verification and extraction rule (a host-built `Spell` MAY hold any `ClassID`, and the decoder accepts any valid id, FR-110). This spec defines no class vocabulary type. | SRD 5.2.1, "Class Spell Lists", p. 105; class spell lists, pp. 33, 38, 44, 55, 60, 67, 74, 79 | happy |
| FR-017 | `Spell.classes` MUST be a defensively copied set: mutating it MUST NOT change the spell, later mutation of the caller's input MUST NOT change the spell, and two spells whose classes differ only in listing order MUST be equal. | `doc/rules/architecture.md` §4 (handoffs); `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-018 | Every bundled spell MUST list at least one class. (`Spell` itself MAY have an empty set, since a host's content may be granted only by features.) | SRD 5.2.1, "Class Spell Lists", p. 105 | unhappy (empty and minimal) |
| FR-019 | Extraction MUST read the class names from the parentheses after the school (each maps to its `ClassID` by lower-casing), including when the header wraps onto a second line (for example Aid, p. 107). | SRD 5.2.1, "Class Spell Lists", p. 105; "Spell Descriptions", p. 107 (Aid) | unhappy (boundary: wrapped header) |
| FR-020 | A class name in a header that is not one of the eight (FR-016) MUST be reported as a problem, not dropped. | `doc/rules/architecture.md` §5 (decoders report problems) | unhappy (invalid input) |
| FR-021 | The same class named twice in one header MUST be reported as a problem. | `doc/rules/testing.md` §4 (contradictory data) | unhappy (invalid input) |
| FR-022 | The header parentheses are authoritative for `Spell.classes`; the eight class spell lists are the verification oracle (FR-023), not the source. The bundled set MUST contain the per-class counts in the table "Distributions", which are header-derived. The class lists give Bard 129, Sorcerer 138 and Wizard 217; the differences are the four findings of FR-097. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-023 | Verification MUST compare each class's membership in `Spell.classes` with that class's printed spell list (pp. 33, 38, 44, 55, 60, 67, 74, 79): a spell present in one source and absent in the other, or listed with a different school, MUST be reported as a finding. A finding MAY be accepted only by an entry in the reviewed exceptions log (FR-077) citing the page of each side. On the PDF the comparison MUST yield exactly the four findings of FR-097 and no others. | SRD 5.2.1, class spell lists, pp. 33, 38, 44, 55, 60, 67, 74, 79 | unhappy (rule interaction: two sources disagree) |
| FR-024 | Verification MUST compare each class list's C and R flags with the spell: C present iff `SpellDuration.concentration`, R present iff `ritual`. The dash glyphs "—" and "–" in the Special column both mean no flag. | SRD 5.2.1, "Wizard Spell List", p. 79 (the legend; same on each list) | unhappy (rule interaction: two sources disagree) |
| FR-025 | Verification MUST compare each class list's M flag with whether the spell's Material component states a GP cost (`costsGp` non-empty) and report every disagreement as a finding; an accepted disagreement is recorded as in FR-023. On the PDF the comparison MUST yield exactly the three Gentle Repose findings of FR-097 and no others (Q12). | SRD 5.2.1, "Wizard Spell List", p. 79 (legend: M means a specific Material component) | unhappy (rule interaction: two sources disagree) |
| FR-026 | `CastingTime` MUST be closed with exactly the five exported variants `CastingTimeMagicAction`, `CastingTimeBonusAction`, `CastingTimeReaction`, `CastingTimeMinutes` and `CastingTimeHours`, so a `switch` is exhaustive without a default. The factories `NewCastingTimeMagicAction()`, `bonusAction([trigger])`, `reaction(trigger)`, `minutes(n)` and `hours(n)` return them (FR-114). | SRD 5.2.1, "Casting Time", p. 105 | happy |
| FR-027 | `reaction` MUST require a non-empty trigger text; an empty or whitespace-only trigger MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Reaction and Bonus Action Triggers", p. 105 | unhappy (invalid input) |
| FR-028 | `bonusAction` MUST accept no trigger and a non-empty trigger; an empty trigger string MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Reaction and Bonus Action Triggers", p. 105 | unhappy (boundary) |
| FR-029 | `minutes(n)` and `hours(n)` MUST require n of at least 1; 0 and negative values MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Longer Casting Times", p. 105 | unhappy (boundary) |
| FR-030 | `Spell.castingTimes` MUST be a `SpellCastingTimes`, a closed type with exactly two variants, so the label rule is carried by the type: `SpellCastingTimesSingle(time)` (one `CastingTime`, no label) and `NewSpellCastingTimesModes(modes)` (a defensively copied list of at least `MinCastingTimeModes` = 2 `NewCastingTimeMode(label, time)` values, each with a non-blank label). Fewer than 2 modes, or a blank label, MUST return an error matching `srderr.ErrInvalidInput`. Zero options, and a single option carrying a label, are unrepresentable. | `doc/rules/architecture.md` §3 (invariants by the type); shared-types-and-ownership.md principle 4 | unhappy (invalid input) |
| FR-031 | `Spell.ritual` MUST be true iff the printed Casting Time entry contains "or Ritual"; the casting time options MUST NOT include the extra 10 minutes (derived by spellcasting). | SRD 5.2.1, "Casting without Slots" (Rituals), p. 104 | happy |
| FR-032 | Extraction MUST map each printed Casting Time form in the table "Cases: Casting Time forms" to the stated model value, and the bundled set MUST contain the stated count of each. | SRD 5.2.1, "Casting Time", p. 105; "Spell Descriptions", pp. 107-175 | happy |
| FR-033 | Plant Growth (p. 153) MUST have exactly two labelled casting time options: "Overgrowth" (Magic action) and "Enrichment" (8 hours). No other bundled spell has more than one option. | SRD 5.2.1, "Plant Growth", p. 153 | unhappy (rule interaction: two casting modes) |
| FR-034 | A trigger that wraps onto following lines in the PDF MUST be stored as one string with line breaks replaced by single spaces; the four Reaction spells and the four Bonus Action spells with a trigger MUST carry a non-empty trigger. | SRD 5.2.1, "Reaction and Bonus Action Triggers", p. 105 | unhappy (boundary: wrapped text) |
| FR-035 | A Casting Time text that matches no form in FR-032 MUST be reported as a problem and MUST NOT be guessed. | `doc/rules/architecture.md` §5 | unhappy (invalid input) |
| FR-036 | `SpellRange` MUST be closed with exactly the six exported variants `SpellRangeSelf`, `SpellRangeTouch`, `SpellRangeDistance(amount, unit)`, `SpellRangeSight`, `SpellRangeUnlimited` and `SpellRangeSpecial`. Factories `NewSpellRangeSelf()`, `touch()`, `distance(amount, unit)`, `sight()`, `unlimited()` and `special()` return them (FR-114). | SRD 5.2.1, "Range", p. 105; entries on pp. 126, 149, 161 (Special, Sight, Unlimited) | happy |
| FR-037 | `distance` MUST require an amount of at least 1 and a `DistanceUnit` of `feet` or `miles` (FR-119); 0 and negative amounts MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Range" (rules text says only "expressed in feet"), p. 105; the entries that use miles: "Clairvoyance", p. 115 (1 mile), "Project Image", p. 156 (500 miles) | unhappy (boundary) |
| FR-038 | Extraction MUST map each printed Range form in the table "Cases: Range forms" to the stated model value, and the bundled set MUST contain the stated count of each. | SRD 5.2.1, "Range", p. 105; "Spell Descriptions", pp. 107-175 | happy |
| FR-039 | A Range text that matches no form in FR-038 MUST be reported as a problem. | `doc/rules/architecture.md` §5 | unhappy (invalid input) |
| FR-040 | Extraction MUST read the Components entry as a comma-separated sequence of the letters V, S and M in that order; any other letter, a repeated letter, or an empty entry MUST be reported as a problem. | SRD 5.2.1, "Components", p. 105 | unhappy (invalid input) |
| FR-041 | A Components entry containing M MUST have a non-empty parenthesised material description, and a parenthesised description with no M MUST be reported as a problem. `MaterialComponent` MUST reject an empty description with an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Material (M)", pp. 105-106 | unhappy (invalid input) |
| FR-042 | The material description MUST be stored verbatim from the parentheses, with line breaks inside it replaced by single spaces. | SRD 5.2.1, "Material (M)", pp. 105-106 | happy |
| FR-043 | `MaterialComponent.costsGp` MUST list, in printed order, every figure written as "N+ GP" in the description, with thousands separators removed (table: Cases: Material components). A description with no such figure has an empty list. | SRD 5.2.1, "Material (M)", pp. 105-106; entries on pp. 111, 115, 144, 160 | happy |
| FR-044 | `MaterialComponent.consumesMaterial` MUST be true iff the description states that the spell consumes any of the materials (the word "consumes"); partial consumption is not recorded separately and the verbatim description keeps the detail. | SRD 5.2.1, "Material (M)" (not consumed unless the description states otherwise), p. 105 | happy |
| FR-045 | A cost stated in a currency other than GP MUST NOT be added to `costsGp`: Gentle Repose (p. 135) has `costsGp` empty and `consumesMaterial` true. | SRD 5.2.1, "Gentle Repose", p. 135 | unhappy (rule interaction: non-GP cost) |
| FR-046 | Extraction MUST accept both the labels "Components:" and "Component:" (12 spells print the singular, for example Barkskin p. 112 and Guidance p. 138) and MUST record each such spell in the source notes. | SRD 5.2.1, "Barkskin", p. 112; "Guidance", p. 138 | unhappy (invalid input: source typo) |
| FR-047 | The bundled set MUST contain the component-combination counts in the table "Cases: Components combinations". | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-048 | The bundled set MUST contain 189 spells with a Material component, 56 of them with a non-empty `costsGp`, and 31 with `consumesMaterial` true. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-049 | `SpellComponents` MUST expose verbal, somatic and material (`MaterialComponent?`) such that a spell with M has a present material and a spell without M has absent. | SRD 5.2.1, "Components", p. 105 | happy |
| FR-050 | `SpellDuration` MUST be closed with exactly the five exported variants `SpellDurationInstantaneous`, `SpellDurationTimed`, `SpellDurationUntilDispelled`, `SpellDurationUntilDispelledOrTriggered` and `SpellDurationSpecial`. Factories `NewSpellDurationInstantaneous()`, `timed(amount, unit, limit)`, `untilDispelled()`, `untilDispelledOrTriggered()` and `special()` return them (FR-114). | SRD 5.2.1, "Duration", p. 106; entries on pp. 121, 128, 169 | happy |
| FR-051 | `timed` MUST require an amount of at least 1 and a `DurationUnit` (content-schema-and-decoder.md FR-149) of `rounds`, `minutes`, `hours` or `days` (FR-119); 0 and negative amounts MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Duration" (Time Span), p. 106; the entries that use days: "Mirage Arcane", p. 149 (10 days), "Geas", p. 135 (30 days) | unhappy (boundary) |
| FR-052 | `SpellDurationTimed` MUST carry a `DurationLimit` (enum: `fixed`, `upTo`, `concentrationUpTo`; FR-119) in place of two flags, so "Concentration without a cap" cannot be expressed. A duration printed "Concentration, up to N unit" is `concentrationUpTo`. | SRD 5.2.1, "Duration" (Concentration), p. 106 | happy |
| FR-053 | A duration printed "Concentration up to 10 minutes" (no comma, Protection from Evil and Good, p. 157) MUST be read as `concentrationUpTo` with a 10 minute amount and recorded in the source notes. | SRD 5.2.1, "Protection from Evil and Good", p. 157 | unhappy (invalid input: source typo) |
| FR-054 | A duration printed "Up to N unit" without Concentration (Etherealness p. 128, Thaumaturgy p. 169, and one entry on p. 154) MUST be read as `SpellDurationTimed` with limit `upTo`. | SRD 5.2.1, "Etherealness", p. 128; "Thaumaturgy", p. 169 | unhappy (rule interaction: cap without Concentration) |
| FR-055 | Extraction MUST map each printed Duration form in the table "Cases: Duration forms" to the stated model value, and the bundled set MUST contain the stated count of each. | SRD 5.2.1, "Duration", p. 106; "Spell Descriptions", pp. 107-175 | happy |
| FR-056 | A Duration text that matches no form in FR-055 MUST be reported as a problem. Creation (p. 121) is `special`, and its details stay in the description. | SRD 5.2.1, "Creation", p. 121 | unhappy (invalid input) |
| FR-057 | `Spell.requiresConcentration` MUST be true iff the duration is `SpellDurationTimed` with limit `concentrationUpTo`, and exactly 133 bundled spells MUST require it. | SRD 5.2.1, "Duration" (Concentration), p. 106 | happy |
| FR-058 | `Spell.description` MUST be a non-empty, defensively copied, ordered list of text blocks; an empty list MUST return an error matching `srderr.ErrInvalidInput`. | `doc/rules/architecture.md` §3 | unhappy (empty and minimal) |
| FR-059 | Description, `higherLevel` and `cantripUpgrade` blocks MUST be the neutral `TextBlock` family owned by content-schema-and-decoder.md (`paragraph`, `entry`, `bulletList`, `table`, `embeddedStatBlock`; its block shape, constructor rules and wire shape are not restated here, Q16). This spec keeps the spell-specific rules: a run-in heading becomes an `entry` by FR-100, and every table in the section MUST be captured (ten table pages, FR-068). | SRD 5.2.1, "Alarm", p. 107 (run-in entries); the table pages listed in FR-068; shared-types-and-ownership.md FR-047 | happy |
| FR-060 | Extraction MUST keep paragraph order exactly as printed and MUST NOT merge two printed paragraphs or split one; boundaries are detected by FR-100. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-061 | `Spell.higherLevel` MUST hold the text that follows the run-in heading "Using a Higher-Level Spell Slot." with the heading removed; exactly 109 bundled spells have it (one run-in heading per spell, none with two, none among the cantrips), distributed by level as in the table "Distributions". | SRD 5.2.1, "Using a Higher-Level Spell Slot", pp. 104-105; "Acid Arrow", p. 107 | happy |
| FR-062 | `Spell.cantripUpgrade` MUST hold the text that follows the run-in heading "Cantrip Upgrade." with the heading removed; exactly 15 bundled spells have it. | SRD 5.2.1, "Acid Splash", p. 107 | happy |
| FR-063 | In the bundled set, no level 0 spell MAY have `higherLevel` and no level 1+ spell MAY have `cantripUpgrade`; a violation MUST be reported as a finding. | SRD 5.2.1, "Acid Splash" and "Acid Arrow", p. 107 | unhappy (rule interaction) |
| FR-064 | Description text MUST be verbatim from the PDF. The only permitted changes are: collapsing runs of whitespace, replacing line breaks inside a paragraph with single spaces, removing page headers and footers (FR-065), and resolving line-end hyphenation (FR-066). No rewording, no spelling or grammar fixes: Animal Friendship (heading p. 107; the quoted sentence is on p. 108) is kept as printed, "the spells ends" included, and recorded in the source notes (Q6). | `AGENTS.md`, "Licensing and source material"; SRD 5.2.1, "Animal Friendship", pp. 107-108 | unhappy (invalid input: source typo) |
| FR-065 | The page footer ("System Reference Document 5.2.1" and the printed page number) MUST NOT appear inside any text field. | SRD 5.2.1, every page footer, pp. 107-175 | unhappy (invalid input: layout artefact) |
| FR-066 | A line-end hyphen (about 607 in the section) or soft hyphen (U+00AD; one) MUST be resolved against one corpus, the whole PDF text (pp. 1-364, line-end hyphens not counted as evidence), with three outcomes. (1) The joined word is attested elsewhere: join. (2) The hyphenated form is attested elsewhere: keep the hyphen. (3) Neither is attested: **unresolved**, a problem (FR-073) cleared only by an FR-077 log entry (spell, field, printed page, decision "join" or "keep"). A soft hyphen is removed before the lookup. If both forms are attested, the joined form wins. Each resolution and each unresolved case MUST be listed in the tool's report. Review found 27 unresolved cases (the first run MUST confirm the list): broken words such as "indef-initely" need "join", real compounds such as "long-dead" need "keep". | SRD 5.2.1, "Spell Descriptions", pp. 107-175 (one soft hyphen is present in the extracted text) | unhappy (boundary: hyphenation) |
| FR-067 | Where the PDF lays two spells out interleaved across columns (for example Animal Messenger and Animate Dead, p. 108), every header field and text block MUST be assigned to the spell whose heading it follows in the printed columns. Verification MUST include spot checks of such pages. | SRD 5.2.1, "Animal Messenger" and "Animate Dead", p. 108 | unhappy (rule interaction: column layout) |
| FR-068 | Every table embedded in a description MUST be captured as a `table` block with every printed row and cell in order, and the row count MUST be asserted in a test per table. The tables are on ten pages: 111 (Augury, Omens table, 4 rows), 117, 120, 121 (Creation, Materials table, 5 rows), 125, 154, 155, 158, 160 (Scrying, two save-modifier tables, 3 rows each) and 168. Row counts for the other pages are read from the PDF when the fixtures are written and pinned in `test/`; a table found on any other page is a finding. | SRD 5.2.1, "Spell Descriptions", pp. 111, 117, 120, 121, 125, 154, 155, 158, 160, 168 | unhappy (rule interaction: embedded table) |
| FR-069 | The four stat blocks embedded in Animate Objects, Find Steed, Giant Insect and Summon Dragon MUST be kept inside the owning spell as `embeddedStatBlock` blocks with every printed line, and MUST NOT become separate `Spell` entries. | SRD 5.2.1, "Animate Objects", "Find Steed", "Giant Insect", "Summon Dragon", pp. 108-109, 130-131, 135-136 and 166 | unhappy (rule interaction: header-less sub-entry) |
| FR-070 | Bulleted lists MUST be captured as `bulletList` blocks with one item per bullet, in order, and cross-references (for example to "Monsters" or "Rules Glossary") MUST stay as text and not be resolved. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-071 | The extraction tool MUST be given a local PDF path, MUST NOT read from the network, and MUST abort before writing anything if the PDF is not SRD 5.2.1 as expected: 364 pages, page 107 footer "107", first heading "Acid Arrow" on page 107, last heading "Zone of Truth" on page 175. | `doc/research/sources.md` ("The PDF is not stored in this repository") | unhappy (invalid input) |
| FR-072 | The tool MUST be deterministic: the same PDF and tool version MUST produce byte-identical files, with entries in printed order. | `doc/rules/testing.md` §4 (determinism) | happy |
| FR-073 | The tool MUST report all problems it finds in one run, each with the spell name (or page when the name is unknown), the field and the kind, MUST exit non-zero, and MUST write no data file when any problem exists. | `doc/rules/architecture.md` §5 (all problems, not just the first) | unhappy (invalid input) |
| FR-074 | The tool MUST NOT supply a value for a missing field: a missing Casting Time, Range, Components or Duration entry, or a header it cannot parse, MUST be a problem. | `AGENTS.md`, "Licensing and source material" (never write rules from memory) | unhappy (missing content) |
| FR-075 | The tool MUST self-check that the number of spell headers equals the number of Casting Time, Range, Components (or Component) and Duration entries (339 each), and report any mismatch. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | unhappy (missing content) |
| FR-076 | The tool MUST verify printed order: page numbers non-decreasing, and names in non-decreasing case-insensitive order with spaces and punctuation ignored; each violation is a finding for review, not silently reordered. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 (alphabetical) | unhappy (state error: out-of-order extraction) |
| FR-077 | A hand correction or accepted finding MUST be recorded as an explicit log entry (spell, field, printed page, reason) that the tool applies; data files MUST NOT be hand-edited otherwise. The log-applying logic (read the log, apply each entry to a parsed record, reject an entry that matches nothing) is a pure function and MUST be tested offline on synthetic records, independently of the PDF check (FR-078). | `doc/rules/testing.md` §5 (keep citations true) | unhappy (state error: unrecorded edit) |
| FR-078 | A PDF check MUST re-run the tool on the PDF and compare with the committed JSON spell files under `content/data/`; any difference not explained by the log (FR-077) MUST fail. It runs locally where the PDF is available, not in CI, and is distinct from the CI decode test of the embedded documents (FR-106). | `doc/research/sources.md` | unhappy (state error: stale data) |
| FR-079 | A second, independent text pass (different reading order, for example the PDF's raw order against its column-aware order) MUST be compared per spell, and each difference that survives normalisation reported. Normalisation: collapse whitespace, resolve hyphens by FR-066, drop page headers and footers (FR-065), drop the entries of a checked-in allow-list of known layout artefacts (small-caps heading differences, bullet glyphs). Interleaved spells (p. 108) are separated by the column-aware order, so the comparison is on per-spell text blocks, not page by page. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | unhappy (rule interaction: two readings disagree) |
| FR-080 | Before the bundled set is marked complete, a person MUST compare each spell with the rendered PDF page and record the review (spell, page, reviewer, date) in the extraction notes. | `AGENTS.md`, "Licensing and source material" | unhappy (missing content: unreviewed entry) |
| FR-081 | The generic rule that every bundled JSON file decodes with no problems is content-schema-and-decoder.md FR-089 and is not restated here. This spec adds only that the ten spell files under `data/srd_5_2_1/` (and their embedded form, FR-030) are among the files that check reads, and that their shape is the one in section 5.2 (FR-107). | content-schema-and-decoder.md FR-089; `doc/rules/architecture.md` §5 | happy |
| FR-082 | The bundled spells, served by `data.BundledSource` (FR-104), MUST satisfy a pin test: a `Lookup` of every one of the 339 bundled spell keys returns `LookupFound` whose entry has an equal key. The shared `content.Source` contract suites run on `data.BundledSource` as content-source-port.md FR-047 arranges (suites exported from `content/contenttest`: shared-types-and-ownership.md FR-039). | `doc/rules/architecture.md` §4, §9; content-source-port.md FR-047 | happy |
| FR-083 | `ListKind(Kind.spell)` on the bundled source MUST return 339 entries ordered by `SpellID` string value using ordinal comparison, and a `Lookup` of a spell id under any other kind MUST return `LookupNotFound`. | `doc/rules/architecture.md` §4; content-source-port.md FR-011, FR-009 | unhappy (missing content: wrong kind) |
| FR-084 | Tests MUST pin every distribution table in section 4 (levels, schools, classes, casting times, ranges, durations, components, Concentration 133 and Ritual 29, `higherLevel` 109 per level and `cantripUpgrade` 15), one test per value. | SRD 5.2.1, "Spell Descriptions", pp. 107-175 | happy |
| FR-085 | Spot-check fixtures (SRD text only under `testdata/srd/`, shared-types-and-ownership.md FR-033) MUST assert every checked field of each spell in the table "Cases: Spot-check spells". | SRD 5.2.1, as cited per row | happy |
| FR-086 | A `Spell` MUST be immutable: every list in it (casting times, description blocks, `costsGp`, table rows, stat block lines) MUST be defensively copied, so a returned slice is a copy and the spell is copied from its inputs. | `doc/rules/architecture.md` §3, §4; `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-087 | Two `Spell`s MUST be equal, with equal hash codes, iff all their fields, `effects` included, are equal, and `Spell.key` MUST equal `(Kind.spell, id)`. | `doc/rules/architecture.md` §3 (value object) | happy |
| FR-088 | Spell value types MUST NOT import `context`, start goroutines, or import `content/data`, `content/contenttest` or any port-implementation package. | `doc/rules/architecture.md` §3, §9 | happy |
| FR-089 | The spell types MUST live under package `content` and be exported from package `content`; the extraction tool MUST live under `tools/`, MUST NOT be exported and MUST NOT be imported from any public package (shared-types-and-ownership.md FR-032). | `doc/rules/architecture.md` §2, §7; shared-types-and-ownership.md FR-032 | happy |
| FR-090 | Bundled spell data MUST live only where shared-types-and-ownership.md FR-030 and FR-033 put it: the ten spell JSON files under `content/data/`, embedded directly with `go:embed` (no generator step, no hand edits beyond the correction log of FR-077; no `data_embedded/` directory). SRD text in tests lives only under `testdata/srd/`. No code or test MAY name a consuming project; decoder tests use clearly synthetic spells, and only the bundled-data tests use SRD spells. | `AGENTS.md`, "Hard constraints"; shared-types-and-ownership.md FR-030, FR-033 | unhappy (invalid input) |
| FR-091 | The change that adds the bundled spells MUST leave `NOTICE.md` covering exactly `data/`, `content/data/` and `testdata/srd/` (shared-types-and-ownership.md FR-033) and stating the changes made to the SRD content, as content-licensing-and-notice.md requires. | `AGENTS.md`, "The notice" | happy |
| FR-092 | This spec owns the `Spell` shape (casting-time options, `SpellRange`, `SpellDuration`, text-block `description`, `higherLevel`, `cantripUpgrade`, `ClassID` class lists; shared-types-and-ownership.md FR-035), registered as row R1-spells of the revision table in content-schema-and-decoder.md (status requested). It claims no `schemaVersion` number: there is one `schemaVersion` for the whole format, the decoder accepts exactly the current version, and before 0.1.0 all revisions land as one version (shared-types-and-ownership.md FR-034; content-schema-and-decoder.md FR-007, FR-103). The change that accepts this spec MUST set R1-spells to landed, in the same change as the decoder support (FR-107 to FR-111) and the bundled data. | shared-types-and-ownership.md FR-034, FR-035; content-schema-and-decoder.md FR-007, FR-103 and its revision table; `doc/rules/architecture.md` §7 (versioning) | unhappy (state error: two specs disagree) |
| FR-094 | The 339 bundled spells replace the twelve spells of the thin slice (content-thin-slice.md FR-003: `fire_bolt`, `prestidigitation`, `detect_magic`, `magic_missile`, `misty_step`, `fireball`, `dancing_lights`, `darkness`, `druidcraft`, `faerie_fire`, `longstrider`, `pass_without_trace`). Each of the twelve ids MUST appear exactly once in the final set with the id FR-003 derives, and the thin slice's data for them MUST be removed or regenerated, not duplicated. The thin-slice spell pins (content-thin-slice.md FR-003 "exactly" twelve, FR-078 `entries_loaded` spell=12, FR-007, FR-060 and table T7) MUST be replaced in the same change: the twelve become twelve of the 339, and the scaling pins reduce to the presence of the `higherLevel` and `cantripUpgrade` text blocks (FR-115). | content-thin-slice.md FR-003, FR-007, FR-060; `doc/rules/testing.md` §4 (duplicate entries, stale tests) | unhappy (missing content: duplicate) |
| FR-097 | Verification (FR-023, FR-025) MUST be pinned to the findings from cross-checking the header lists against the eight class lists. Class membership, four findings, accepted by log entries, the header being authoritative (FR-022): "Phantasmal Force" (p. 151) lists Bard, Sorcerer and Wizard in its header but is on none of the class lists (Bard, p. 33; Sorcerer, p. 67; Wizard, p. 79); "Mind Spike" (p. 149) lists Sorcerer in its header but is absent from the Sorcerer list (pp. 67-73) while present on the Warlock list (p. 75) and the Wizard list (p. 80). M flag, three findings: "Gentle Repose" (p. 135) is flagged M on the Cleric list (p. 38), the Paladin list (p. 56) and the Wizard list (p. 80), but its Material component states a cost in Copper Pieces, not GP, so `costsGp` is empty (FR-045). C and R flags disagree nowhere. | SRD 5.2.1, "Phantasmal Force", p. 151; "Mind Spike", p. 149; "Gentle Repose", p. 135; class spell lists, pp. 33, 38, 56, 67-73, 75, 79, 80 | unhappy (rule interaction: two sources disagree) |
| FR-098 | `Spell.name` MUST be non-empty with no leading or trailing whitespace (`ValidSpellName`), else an error matching `srderr.ErrInvalidInput`; the decoder's name rule (content-schema-and-decoder.md FR-027) and this constructor therefore agree (FR-111). | `doc/rules/testing.md` §4 (invalid input); content-schema-and-decoder.md FR-027 | unhappy (invalid input) |
| FR-100 | The tool reads the PDF through `pdftohtml -xml` (poppler), which carries per-run position and emphasis that `pdftotext` default output drops (Q9). A run-in heading is a text run wrapped in bold-italic markup (`<i><b>`) at the start of a line, with the printed full stop kept in the name's text; the name is stored without the full stop. A new printed paragraph starts at a text run that begins with a first-line indent (leading spaces in the run text, or a `left` offset greater than the column's modal `left` by the indent threshold calibrated on the page 107 fixture); a line with neither is a continuation. A line that matches no run-in heading, paragraph start or continuation rule is a problem. | SRD 5.2.1, "Acid Arrow" and "Alarm", p. 107 (run-in headings printed bold italic after a first-line indent) | unhappy (invalid input: layout cues) |
| FR-101 | `MaterialComponent.costsGp` entries MUST each be at least 1, else an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Material (M)", pp. 105-106 ("N+ GP" costs) | unhappy (boundary) |
| FR-102 | When `higherLevel` or `cantripUpgrade` is present it MUST be a non-empty list of blocks (copied on construction and on read, FR-086); an empty list MUST return an error matching `srderr.ErrInvalidInput`. | `doc/rules/testing.md` §4 (empty and minimal) | unhappy (empty and minimal) |
| FR-103 | Extraction tests MUST include a fixture (under `testdata/srd/`) for a spell with run-in headings (Alarm, p. 107) and one for a column-interleaved page (p. 108), each with the expected blocks, so FR-060, FR-067 and FR-100 do not rest on the full PDF alone. | SRD 5.2.1, "Alarm", p. 107; "Animal Messenger" and "Animate Dead", p. 108 | unhappy (rule interaction: layout) |
| FR-104 | The bundled spells reach consumers only through the single decode path: the embedded JSON strings are decoded by the same `content.Decoder` as host JSON and served by `data.BundledSource`, exported from `content/data`. This spec adds no loader, no second decoder and no spell-specific adapter. | shared-types-and-ownership.md FR-030, FR-031 | happy |
| FR-105 | The tool's default `--out` MUST be `content/data/` (ten JSON files, Q10); it MUST NOT write to any Go source file. The public packages never import `tools/` (shared-types-and-ownership.md FR-032; architecture.md §9). The reverse is this spec's choice: the tool MAY import the public packages, so it builds ids and values with the real `ParseSpellID`, `NewSpellLevel` and constructors and the id pattern exists once (FR-003, FR-111). | shared-types-and-ownership.md FR-030, FR-032; `doc/rules/architecture.md` §9 | happy |
| FR-106 | There is no generator and no second copy to drift (shared-types-and-ownership.md FR-030, FR-032); instead a test MUST assert that the spell JSON files embedded for kind `spell` decode through the real decoder to 339 entries and equal the pinned ids of FR-002. It reads the files under `content/data/` with `os`, so it lives in `internal/archtest/` (FR-117). | shared-types-and-ownership.md FR-032 | unhappy (stale data) |
| FR-107 | The spell object in a content document MUST have the keys, in this declared order (the order problems are reported, content-schema-and-decoder.md FR-016): `id`, `name`, `source?`, `level`, `school`, `classes`, `castingTimes`, `ritual`, `range`, `components`, `duration`, `description`, `higherLevel?`, `cantripUpgrade?`, `effects?` (FR-122). A key not marked `?` is required (absent: `missingField`); an optional key is expressed by absence, never by `absent` (content-schema-and-decoder.md FR-020); any other key is `unknownField` (content-schema-and-decoder.md FR-022), so the old shape's `higherLevelSlot` and `text` are rejected. `source` is the common object `{document, section, page}` (content-schema-and-decoder.md FR-034) and decodes to a `SourceCitation` (FR-115 there). Nested objects, key order and spellings are in section 5.2. | content-schema-and-decoder.md FR-016, FR-019 to FR-022, FR-034; shared-types-and-ownership.md FR-035; `doc/rules/architecture.md` §5 | unhappy (missing content, invalid input) |
| FR-108 | Each closed type is written as a JSON object with a string discriminator `kind` and the exact, case-sensitive spellings of section 5.2 (`magicAction`, `bonusAction`, `reaction`, `minutes`, `hours`; `self`, `touch`, `distance`, `sight`, `unlimited`, `special`; `instantaneous`, `timed`, `untilDispelled`, `untilDispelledOrTriggered`, `special`; the text-block spellings are the schema spec's). Unit and limit values are `feet`/`miles`, `rounds`/`minutes`/`hours`/`days` and `fixed`/`upTo`/`concentrationUpTo`; `school` uses the SRD spelling ("Evocation"); a value outside the closed set is `invalidValue` (content-schema-and-decoder.md FR-025), with no trimming or case-folding. Discriminators are the lower camel case of the variant names (content-schema-and-decoder.md section 3). | content-schema-and-decoder.md FR-025, section 3; SRD 5.2.1, "Schools of Magic", p. 105 | unhappy (invalid input) |
| FR-109 | The decoder MUST report each invalid value as the problem in the table "Cases: spell JSON" (path from the document root, kind from content-schema-and-decoder.md): a required key absent is `missingField`; a wrong JSON type `wrongType`; an integer outside its range or an array outside its length bounds `outOfRange` (a whole number above 2^53 - 1 included, content-schema-and-decoder.md FR-021); a string outside a closed set, a blank text or a malformed id `invalidValue`; a repeated `classes` entry `duplicate` at the later index (content-schema-and-decoder.md FR-026); a rule between sibling values (a label on the only casting-time option, a ragged table row) `constraintViolation`; an undeclared key `unknownField`. A key valid for one `kind` and not another (an `amount` on `magicAction`) is `unknownField`. All problems in one spell are reported, in the order of FR-107. | content-schema-and-decoder.md FR-019 to FR-028, FR-036; `doc/rules/architecture.md` §5 (all problems, path and kind) | unhappy (invalid input) |
| FR-110 | `classes` MUST be an array of plain `ClassID` strings (each matching the id pattern, `invalidValue` otherwise), of any length including zero. The entries are NOT reference sites: R1-spells registers none for `classes` (content-schema-and-decoder.md FR-126; the whole-set validation of shared-types-and-ownership.md FR-054 checks only registered sites), because a host may supply spells without the class entries, so the decoder MUST NOT report `unresolvedReference` for them. That a bundled spell uses only the eight spell-list ids is bundle verification (FR-016). | content-schema-and-decoder.md FR-126, FR-114; shared-types-and-ownership.md FR-054 | unhappy (rule interaction: not a reference) |
| FR-111 | The domain types MUST expose their limits as public constants and predicates, and the decoder MUST use them instead of repeating literals (content-schema-and-decoder.md FR-094): `MinSpellLevel` (0), `MaxSpellLevel` (9), `ValidSpellLevel(int) bool`; `MinCastingTimeAmount` (1), `ValidCastingTimeAmount(int) bool`; `MinCastingTimeModes` (2); `MinSpellRangeDistance` (1), `ValidSpellRangeDistance(int) bool`; `MinSpellDurationAmount` (1), `ValidSpellDurationAmount(int) bool`; `MinMaterialCostGP` (1), `ValidMaterialCostGP(int) bool`; `ValidSpellName(string) bool`. `SourceCitation` limits are content-schema-and-decoder.md FR-115. The table "Cases: spell JSON" gives, per row, the JSON, the decoder problem and the constructor verdict, and is registered with the agreement test of content-schema-and-decoder.md FR-094. | content-schema-and-decoder.md FR-094; `doc/rules/architecture.md` §6 | unhappy (rule interaction: decoder and constructor agree) |
| FR-113 | `Spell.id` MUST be a `SpellID` as defined in content-schema-and-decoder.md FR-114 (a comparable struct over a validated `ID`, `content.ParseSpellID(string) (SpellID, error)`, and a `Key()` method equal to `Key{Kind: Spell, ID: id}`); this spec defines no id type. `Spell.key` MUST equal `id.Key()`. The decoder MUST accept an `id` exactly when `ParseSpellID` accepts it (agreement row in FR-111). | content-schema-and-decoder.md FR-114; shared-types-and-ownership.md FR-005, FR-046 | happy |
| FR-114 | The exported class names are exactly those in FR-026, FR-030, FR-036, FR-050, FR-119 and FR-120, each prefixed with its family (`CastingTime...`, `SpellRange...`, `SpellDuration...`) so no exported name collides on `Special`, `Reaction`, `Entry` or `Table` (shared-types-and-ownership.md FR-017, FR-021; architecture.md §9). Variant constructors validate and return an error matching `srderr.ErrInvalidInput`; the closed interface is built through the factory functions with the names used in the case tables (`NewCastingTimeReaction(trigger)`, `NewSpellRangeDistance(amount, unit)`, `NewSpellDurationTimed(amount, unit, limit)`, and the rest), which return the variant. The exported-name scan MUST find no duplicate and no generic name, and a type switch over each closed type MUST be exhaustive without a default. | shared-types-and-ownership.md FR-017, FR-021; `doc/rules/architecture.md` §4, §9 | unhappy (name collision) |
| FR-115 | This spec provides no `SpellScaling` and no structured `scaling` field: scaling stays text in `higherLevel` and `cantripUpgrade` (Q5). content-thin-slice.md's `SpellScaling` (its FR-060 and table T7) is therefore not a type of `content`; FR-094 replaces those pins with presence checks of the two text blocks. | content-thin-slice.md FR-060, T7; shared-types-and-ownership.md FR-035 | unhappy (state error: two specs disagree) |
| FR-116 | SRD-derived test artefacts live under `testdata/srd/spells/` (shared-types-and-ownership.md FR-033) as Go fixture files (`.go` values or JSON), so tests that read them are platform-independent: `spell_names.go` (the 339 names, FR-002), `class_list_oracle.go` (the class-list expectation, FR-023 to FR-025, Q11) and `table_row_counts.go` (the ten table pages, FR-068), plus the fixtures of FR-085 and FR-103. Both files are to be created (not yet present); creating either MUST add its row to `doc/documentation_index.md` in the same change. The hand-correction and exceptions log (FR-066, FR-077) is `doc/research/spells-extraction-log.json` (to be created); the source notes and review log (FR-046, FR-053, FR-064, FR-080) are `doc/research/spells-extraction-notes.md` (to be created). The change that creates either also adds its row to `doc/documentation_index.md` (architecture-guardrails.md FR-102). Both quote SRD text and are not distributed (`doc/` is in `.pubignore`; shared-types-and-ownership.md FR-033). | shared-types-and-ownership.md FR-033; `doc/rules/testing.md` §6 | happy |
| FR-117 | A check that reads the real filesystem with `os` MUST live in `internal/archtest/` and run (it is not run in the pure unit suite): the embedded-documents decode test (FR-106), the notes-file check (FR-046), the exceptions-log check (FR-025, FR-097), the ten-files-present check (FR-081) and the import scan (FR-105). This is a stated exception to "no real filesystem" (`doc/rules/testing.md` §6), as in content-schema-and-decoder.md FR-089, because the files are the artefact under test. `content/data/bundled_spells_test.go` reads only the embedded documents and the fixtures of FR-116, so it stays platform-independent (content-schema-and-decoder.md FR-099). | `doc/rules/testing.md` §3, §6; content-schema-and-decoder.md FR-089, FR-099 | happy |
| FR-118 | Tests of the extraction tool live in the `tools/` module tests (beside the code) as unit tests using only synthetic records and the fixtures of FR-116, importing no port or double. The `tools/` module is not listed in `doc/rules/testing.md` §10, so the accepting change MUST amend that section (replacing its RULE-TAG with a fresh random value) and keep the "every test cites a requirement" architecture check covering it. | `doc/rules/testing.md` §3, §9, §10 | happy |
| FR-119 | The model MUST name and export (no generic variant names, shared-types-and-ownership.md FR-017): `DistanceUnit` (enum: `feet`, `miles`), `DurationLimit` (enum: `fixed`, `upTo`, `concentrationUpTo`) and the `NewSpellLevel` rejection (a `*SpellLevelError` carrying the rejected integer, matching `srderr.ErrInvalidInput`, and `errors.As`-able). `NewSpellRangeDistance(amount, unit)` takes a `DistanceUnit`; `NewSpellDurationTimed(amount, unit, limit)` takes the `DurationUnit` owned by content-schema-and-decoder.md FR-149 and a `DurationLimit`. This spec defines no `DurationUnit`. | `doc/rules/architecture.md` §3, §6; SRD 5.2.1, "Range", p. 105; "Duration", p. 106; shared-types-and-ownership.md FR-065 | happy |
| FR-120 | A class spell list MUST be derived from `Spell.classes`, never stored on a class entry: the pure function `DeriveClassSpellList(ClassID, []Spell)` (Go cannot export both a type and a function named `ClassSpellList`; plan item P7) returns a `ClassSpellList` holding the `ClassID` and, for each `SpellLevel` 0 to 9, a defensively copied list of the `SpellID`s of the spells whose `classes` contain the id (`IDsAt(level)`). A `ClassID` listed by no spell, and a non-caster, yields empty groups. `ClassSpellList` and `DeriveClassSpellList` are defined here in package `content`; content-classes.md defines neither and asks nothing of this spec. | SRD 5.2.1, "Class Spell Lists", p. 105; shared-types-and-ownership.md FR-066 | happy |
| FR-121 | Within a group the `SpellID`s MUST be ordered by string value with ordinal comparison, so the result is deterministic and independent of input order; the list MUST be defensively copied and the function MUST NOT alter its input. `DeriveClassSpellList` MUST NOT depend on a `ClassDefinition`, and a spell that appears twice in its input is listed once: a list for a `ClassID` held by no loaded class (a host's house class) is derived the same way, since membership is a fact of the spell (FR-016, FR-110). | `doc/rules/testing.md` §4 (determinism, immutability); `doc/rules/architecture.md` §2, §3 | unhappy (determinism) |
| FR-122 | `Spell` MUST carry `effects`, a `SpellEffectModel` defined by spell-effect-data.md (not here), defaulting to its zero-value `TextOnlyEffects`; a `Spell` without it is valid (FR-018). The optional last JSON key `effects` is read by that spec's decoder shape (its FR-030; revision row R2-spell-effects in content-schema-and-decoder.md); absence decodes to `TextOnlyEffects`. This spec defines no effect type; it imports the model from spell-effect-data.md (a lower rank, listed in Depends on), which MUST NOT import `Spell`, so imports stay acyclic (Q17). The `Spell` constructor calls that spec's level check (its FR-009) with the spell level. | spell-effect-data.md FR-001, FR-030, Q5; shared-types-and-ownership.md FR-035, FR-059 | happy |

#### Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-093 | Withdrawn: moved to content-schema-and-decoder.md (FR-115, `SourceCitation`); that every bundled `document` is exactly `SRD 5.2.1` stays in FR-008 |
| FR-095 | Withdrawn: `DurationUnit` moved to content-schema-and-decoder.md (FR-149; shared-types-and-ownership.md FR-065); `DistanceUnit`, `DurationLimit` and `SpellLevelParse` moved to FR-119 here |
| FR-099 | Withdrawn: superseded by FR-052 (`DurationLimit` makes the two-flag combination unrepresentable; limits `fixed`, `upTo` per FR-054, `concentrationUpTo`) |
| FR-112 | Withdrawn: moved to content-schema-and-decoder.md (FR-115, with FR-138; shared-types-and-ownership.md FR-053) |
| FR-096 | Withdrawn: block constructor rules moved to content-schema-and-decoder.md (`TextBlock`, Q16); the block case rows cite FR-059 |

### Cases: spell JSON (FR-107 to FR-111; one test per row, registered with content-schema-and-decoder.md FR-094)

"Otherwise valid" means a document with one valid spell at `/spells/0`. The constructor verdict is
what the matching `NewX` constructor does with the same value; "unrepresentable" means the
type cannot hold it, so only the decoder guards it.

| Label | JSON (change to an otherwise valid spell 0) | Decoder problem (path, kind) | Constructor verdict | Source | Path |
|---|---|---|---|---|---|
| valid minimal | all required keys, `"level":0`, one `magicAction`, `"classes":[]`, `"description"` with one paragraph | none | accepted | FR-107, FR-018 | happy |
| level too high | `"level":10` | `/spells/0/level`, `outOfRange` | `NewSpellLevel(10)` returns an error matching `srderr.ErrInvalidInput` | FR-009, FR-109 | unhappy (boundary) |
| level negative | `"level":-1` | `/spells/0/level`, `outOfRange` | `NewSpellLevel(-1)` returns an error | FR-009 | unhappy (boundary) |
| level 9 | `"level":9` | none | accepted | FR-009 | unhappy (boundary) |
| level a string | `"level":"3"` | `/spells/0/level`, `wrongType` | unrepresentable (not an `int`) | FR-109 | unhappy (invalid input) |
| level absent | no `level` | `/spells/0/level`, `missingField` | unrepresentable | FR-107 | unhappy (missing content) |
| unknown school | `"school":"Lore"` | `/spells/0/school`, `invalidValue` | unrepresentable (closed enum) | FR-108 | unhappy (invalid input) |
| school wrong case | `"school":"evocation"` | `/spells/0/school`, `invalidValue` | unrepresentable | FR-108 | unhappy (invalid input) |
| class repeated | `"classes":["wizard","wizard"]` | `/spells/0/classes/1`, `duplicate` | unrepresentable (a `Set`) | FR-109 | unhappy (invalid input) |
| class not an id | `"classes":["Wizard"]` | `/spells/0/classes/0`, `invalidValue` | `NewClassID("Wizard")` returns an error | FR-110 | unhappy (invalid input) |
| class unknown to the document | `"classes":["unknown_class"]` | none (not a reference site) | accepted | FR-110 | unhappy (rule interaction) |
| no casting time | `"castingTimes":[]` | `/spells/0/castingTimes`, `outOfRange` | `NewSpellCastingTimesModes([])` returns an error | FR-030, FR-109 | unhappy (empty and minimal) |
| unknown casting kind | `"castingTimes":[{"kind":"bonus"}]` | `/spells/0/castingTimes/0/kind`, `invalidValue` | unrepresentable (closed) | FR-108 | unhappy (invalid input) |
| reaction without trigger | `{"kind":"reaction"}` | `/spells/0/castingTimes/0/trigger`, `missingField` | unrepresentable (the factory requires it) | FR-027 | unhappy (missing content) |
| reaction blank trigger | `{"kind":"reaction","trigger":" "}` | `/spells/0/castingTimes/0/trigger`, `invalidValue` | `NewCastingTimeReaction(" ")` returns an error | FR-027 | unhappy (invalid input) |
| minutes zero | `{"kind":"minutes","amount":0}` | `/spells/0/castingTimes/0/amount`, `outOfRange` | `NewCastingTimeMinutes(0)` returns an error | FR-029 | unhappy (boundary) |
| amount on an action | `{"kind":"magicAction","amount":5}` | `/spells/0/castingTimes/0/amount`, `unknownField` | unrepresentable | FR-109 | unhappy (invalid input) |
| label on the only option | one option with `"label":"Solo"` | `/spells/0/castingTimes/0/label`, `constraintViolation` | unrepresentable (`SpellCastingTimesSingle` has no label) | FR-030 | unhappy (rule interaction) |
| second mode unlabelled | two options, the second without `label` | `/spells/0/castingTimes/1/label`, `missingField` | unrepresentable (`CastingTimeMode` requires it) | FR-030 | unhappy (missing content) |
| blank mode label | two options, the second `"label":""` | `/spells/0/castingTimes/1/label`, `invalidValue` | `NewCastingTimeMode("", time)` returns an error | FR-030 | unhappy (invalid input) |
| distance zero | `{"kind":"distance","amount":0,"unit":"feet"}` | `/spells/0/range/amount`, `outOfRange` | `NewSpellRangeDistance(0, DistanceUnit.feet)` returns an error | FR-037 | unhappy (boundary) |
| distance unit unknown | `"unit":"furlongs"` | `/spells/0/range/unit`, `invalidValue` | unrepresentable (enum) | FR-108 | unhappy (invalid input) |
| distance without unit | `{"kind":"distance","amount":5}` | `/spells/0/range/unit`, `missingField` | unrepresentable | FR-109 | unhappy (missing content) |
| unit on `self` | `{"kind":"self","unit":"feet"}` | `/spells/0/range/unit`, `unknownField` | unrepresentable | FR-109 | unhappy (invalid input) |
| verbal not a boolean | `"verbal":"yes"` | `/spells/0/components/verbal`, `wrongType` | unrepresentable | FR-109 | unhappy (invalid input) |
| empty material | `"material":{"description":"","costsGp":[],"consumesMaterial":false}` | `/spells/0/components/material/description`, `invalidValue` | `NewMaterialComponent("", ...)` returns an error | FR-041 | unhappy (invalid input) |
| cost zero | `"costsGp":[0]` | `/spells/0/components/material/costsGp/0`, `outOfRange` | `MaterialComponent` returns an error | FR-101 | unhappy (boundary) |
| timed without amount | `{"kind":"timed","unit":"hours","limit":"fixed"}` | `/spells/0/duration/amount`, `missingField` | unrepresentable | FR-109 | unhappy (missing content) |
| timed zero | `"amount":0` | `/spells/0/duration/amount`, `outOfRange` | `NewSpellDurationTimed(0, ...)` returns an error | FR-051 | unhappy (boundary) |
| concentration without a cap | `"limit":"concentration"` | `/spells/0/duration/limit`, `invalidValue` | unrepresentable (`DurationLimit` has no such value) | FR-052 | unhappy (rule interaction) |
| amount on `instantaneous` | `{"kind":"instantaneous","amount":1}` | `/spells/0/duration/amount`, `unknownField` | unrepresentable | FR-109 | unhappy (invalid input) |
| description empty | `"description":[]` | `/spells/0/description`, `outOfRange` | `Spell` returns an error (FR-058) | FR-058 | unhappy (empty and minimal) |
| blank paragraph | `{"kind":"paragraph","text":"  "}` | `/spells/0/description/0/text`, `invalidValue` | `NewTextBlockParagraph("  ")` returns an error | FR-059 | unhappy (invalid input) |
| unknown block kind | `{"kind":"quote","text":"x"}` | `/spells/0/description/0/kind`, `invalidValue` | unrepresentable | FR-059 | unhappy (invalid input) |
| ragged table | header of 2, a row of 3 | `/spells/0/description/0/rows/1`, `constraintViolation` | `TextBlock.table` returns an error | FR-059 | unhappy (rule interaction) |
| table without rows | `"rows":[]` | `/spells/0/description/0/rows`, `outOfRange` | `TextBlock.table` returns an error | FR-059 | unhappy (empty and minimal) |
| empty higher-level text | `"higherLevel":[]` | `/spells/0/higherLevel`, `outOfRange` | `Spell` returns an error | FR-102 | unhappy (empty and minimal) |
| old shape | `"higherLevelSlot":"..."` | `/spells/0/higherLevelSlot`, `unknownField` | unrepresentable | FR-107 | unhappy (invalid input) |
| id not valid | `"id":"Acid-Arrow"` | `/spells/0/id`, `invalidValue` | `NewSpellID("Acid-Arrow")` returns an error | FR-113 | unhappy (invalid input) |
| name padded | `"name":" Acid Arrow"` | `/spells/0/name`, `invalidValue` | `Spell` returns an error (FR-098) | FR-098 | unhappy (invalid input) |
| citation page zero | `"source":{"document":"d","section":"s","page":0}` | `/spells/0/source/page`, `outOfRange` | `NewSourceCitation(page: 0)` returns an error | content-schema-and-decoder.md FR-115 | unhappy (boundary) |
| citation section absent | `"source":{"document":"d","page":1}` | `/spells/0/source/section`, `missingField` | unrepresentable | content-schema-and-decoder.md FR-115 | unhappy (missing content) |
| citation blank document | `"document":" "` | `/spells/0/source/document`, `invalidValue` | `SourceCitation` returns an error | content-schema-and-decoder.md FR-115 | unhappy (invalid input) |
| citation absent | no `source` | none | accepted | FR-007 | unhappy (empty and minimal) |
| two problems, one spell | `level` 10 and `school` unknown | `/spells/0/level`, then `/spells/0/school` | n/a | FR-107, FR-109 | unhappy (rule interaction) |

### Cases: id derivation (FR-003; one test per row)

| Label | Name | Expected id | Path |
|---|---|---|---|
| plain | Acid Arrow | `acid_arrow` | happy |
| typographic apostrophe | Hunter’s Mark | `hunters_mark` | happy |
| plural possessive | Heroes’ Feast | `heroes_feast` | happy |
| possessive, three words | Arcanist’s Magic Aura | `arcanists_magic_aura` | happy |
| slash | Blindness/Deafness | `blindness_deafness` | unhappy (boundary) |
| slash, other | Antipathy/Sympathy | `antipathy_sympathy` | unhappy (boundary) |
| short words | Create or Destroy Water | `create_or_destroy_water` | happy |
| ASCII apostrophe | Hunter's Mark | `hunters_mark` | unhappy (boundary) |
| repeated separators | `Acid  --  Arrow` | `acid_arrow` | unhappy (boundary) |
| leading and trailing separators | `/Acid Arrow/` | `acid_arrow` | unhappy (boundary) |
| no letters or digits | `’/’` | problem reported | unhappy (invalid input) |
| empty | `` | problem reported | unhappy (invalid input) |
| starts with a digit | `1st Spell` | problem reported (id would not match `^[a-z][a-z0-9_]*$`) | unhappy (invalid input) |

### Cases: `SpellLevel` (FR-009 to FR-011; one test per row)

| Label | Input | Expected | Path |
|---|---|---|---|
| one below minimum | -1 | a `*SpellLevelError` carrying -1 (matches `srderr.ErrInvalidInput`) | unhappy (boundary) |
| minimum | 0 | accepted; `IsCantrip` true | unhappy (boundary) |
| first level 1+ | 1 | accepted; `IsCantrip` false | unhappy (boundary) |
| maximum | 9 | accepted | unhappy (boundary) |
| one above maximum | 10 | a `*SpellLevelError` carrying 10 (matches `srderr.ErrInvalidInput`) | unhappy (boundary) |

### Cases: header forms (FR-013, FR-019 to FR-021; one test per row)

| Label | Header text | Expected | Source | Path |
|---|---|---|---|---|
| level 1+ | `Level 2 Evocation (Wizard)` | level 2, Evocation, [Wizard] | FR-013 | happy |
| cantrip | `Evocation Cantrip (Sorcerer, Wizard)` | level 0, Evocation, [Sorcerer, Wizard] | FR-013 | happy |
| wrapped class list | `Level 2 Abjuration (Bard, Cleric, Druid, Paladin,` then `Ranger)` | level 2, Abjuration, five classes | FR-019 | unhappy (boundary) |
| level out of range | `Level 10 Evocation (Wizard)` | problem reported | FR-013 | unhappy (boundary) |
| level 0 spelled as a level | `Level 0 Evocation (Wizard)` | problem reported | FR-013 | unhappy (invalid input) |
| unknown school | `Level 2 Lore (Wizard)` | problem reported | FR-013 | unhappy (invalid input) |
| unknown class | `Level 2 Evocation (Unknown Class)` | problem reported | FR-020 | unhappy (invalid input) |
| duplicate class | `Level 2 Evocation (Wizard, Wizard)` | problem reported | FR-021 | unhappy (invalid input) |
| no class list | `Level 2 Evocation` | problem reported | FR-013 | unhappy (missing content) |

### Distributions (FR-014, FR-015, FR-022, FR-061; one test per value)

Counted from the PDF. Both level rows sum to 339 and 109 (FR-001, FR-061); no cantrip and no level 9
spell has `higherLevel` (a cantrip with it is the FR-063 finding). Schools sum to 339. Path: happy
for every value except the zero cells of the `higherLevel` row (unhappy (boundary); the cantrip cell
is the FR-063 rule interaction). Requirement per cell: the FR in the column or row label.

| Level | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | Total |
|---|---|---|---|---|---|---|---|---|---|---|---|
| spells (FR-014) | 27 | 57 | 57 | 42 | 34 | 38 | 31 | 20 | 17 | 16 | 339 |
| with `higherLevel` (FR-061) | 0 | 27 | 21 | 16 | 14 | 15 | 11 | 3 | 2 | 0 | 109 |

| School (FR-015) | Abjuration | Conjuration | Divination | Enchantment | Evocation | Illusion | Necromancy | Transmutation | Total |
|---|---|---|---|---|---|---|---|---|---|
| spells | 49 | 54 | 31 | 34 | 54 | 29 | 27 | 61 | 339 |

A spell can be on several lists, so class counts sum to more than 339. These header-derived
counts are authoritative (FR-022); the printed lists give Bard 129, Sorcerer 138, Wizard 217,
the rest equal (the four findings of FR-097).

| Class (FR-022) | Bard | Cleric | Druid | Paladin | Ranger | Sorcerer | Warlock | Wizard |
|---|---|---|---|---|---|---|---|---|
| spells listing it | 130 | 109 | 124 | 38 | 48 | 140 | 72 | 218 |

### Cases: Casting Time forms (FR-032; one test per row; counts sum to 339 and 29 forms contain "or Ritual", 16 + 11 + 1 + 1)

| Label | Printed text | Model | Count | Source | Path |
|---|---|---|---|---|---|
| action | `Action` | `magicAction` | 240 | FR-032 | happy |
| action or ritual | `Action or Ritual` | `magicAction`, ritual | 16 | FR-031, FR-032 | happy |
| bonus action | `Bonus Action` | `bonusAction`, no trigger | 19 | FR-028, FR-032 | happy |
| bonus action with trigger | `Bonus Action, which you take immediately after hitting ...` | `bonusAction` with trigger | 4 | FR-028, FR-034 | happy |
| reaction | `Reaction, which you take when ...` | `reaction` with trigger | 4 | FR-027, FR-034 | happy |
| 1 minute | `1 minute` | `minutes(1)` | 17 | FR-032 | happy |
| 1 minute or ritual | `1 minute or Ritual` | `minutes(1)`, ritual | 11 | FR-031 | happy |
| 10 minutes | `10 minutes` | `minutes(10)` | 12 | FR-032 | happy |
| 10 minutes or ritual | `10 minutes or Ritual` | `minutes(10)`, ritual | 1 | FR-031 | happy |
| 1 hour | `1 hour` | `hours(1)` | 10 | FR-032 | happy |
| 1 hour or ritual | `1 hour or Ritual` | `hours(1)`, ritual | 1 | FR-031 | happy |
| 8 hours | `8 hours` | `hours(8)` | 1 | FR-032 | unhappy (boundary) |
| 12 hours | `12 hours` | `hours(12)` | 1 | FR-032 | unhappy (boundary) |
| 24 hours | `24 hours` | `hours(24)` | 1 | FR-032 | unhappy (boundary) |
| two modes | `Action (Overgrowth) or 8 hours (Enrichment)` | two labelled options | 1 | FR-033 | unhappy (rule interaction) |
| zero minutes | `0 minutes` | problem reported | 0 | FR-029, FR-035 | unhappy (boundary) |
| unknown unit | `1 fortnight` | problem reported | 0 | FR-035 | unhappy (invalid input) |
| empty | `` | problem reported | 0 | FR-074 | unhappy (missing content) |

### Cases: Range forms (FR-038; one test per row; counts sum to 339)

| Label | Printed text | Model | Count | Source | Path |
|---|---|---|---|---|---|
| self | `Self` | `self` | 72 | FR-038 | happy |
| touch | `Touch` | `touch` | 66 | FR-038 | happy |
| 5 feet | `5 feet` | `distance(5, feet)` | 1 | FR-038 | unhappy (boundary) |
| 10 feet | `10 feet` | `distance(10, feet)` | 11 | FR-038 | happy |
| 15 feet | `15 feet` | `distance(15, feet)` | 1 | FR-038 | happy |
| 30 feet | `30 feet` | `distance(30, feet)` | 47 | FR-038 | happy |
| 60 feet | `60 feet` | `distance(60, feet)` | 65 | FR-038 | happy |
| 90 feet | `90 feet` | `distance(90, feet)` | 12 | FR-038 | happy |
| 100 feet | `100 feet` | `distance(100, feet)` | 2 | FR-038 | happy |
| 120 feet | `120 feet` | `distance(120, feet)` | 34 | FR-038 | happy |
| 150 feet | `150 feet` | `distance(150, feet)` | 12 | FR-038 | happy |
| 300 feet | `300 feet` | `distance(300, feet)` | 6 | FR-038 | happy |
| 500 feet | `500 feet` | `distance(500, feet)` | 2 | FR-038 | happy |
| 1 mile | `1 mile` | `distance(1, miles)` | 4 | FR-038 | unhappy (boundary) |
| 500 miles | `500 miles` | `distance(500, miles)` | 1 | FR-038 | unhappy (boundary) |
| sight | `Sight` | `sight` | 1 | FR-036 | unhappy (boundary) |
| unlimited | `Unlimited` | `unlimited` | 1 | FR-036 | unhappy (boundary) |
| special | `Special` | `special` | 1 | FR-036 | unhappy (boundary) |
| zero feet | `0 feet` | problem reported | 0 | FR-037, FR-039 | unhappy (boundary) |
| unknown unit | `30 furlongs` | problem reported | 0 | FR-039 | unhappy (invalid input) |

### Cases: Duration forms (FR-055; one test per row; counts sum to 339, the seven Concentration rows plus the missing-comma row sum to 133, FR-057)

| Label | Printed text | Model | Count | Source | Path |
|---|---|---|---|---|---|
| instantaneous | `Instantaneous` | `instantaneous` | 101 | FR-055 | happy |
| concentration 1 minute | `Concentration, up to 1 minute` | timed 1 minute, `concentrationUpTo` | 66 | FR-052 | happy |
| concentration 10 minutes | `Concentration, up to 10 minutes` | timed 10 minutes, `concentrationUpTo` | 36 | FR-052 | happy |
| concentration 1 hour | `Concentration, up to 1 hour` | timed 1 hour, `concentrationUpTo` | 24 | FR-052 | happy |
| concentration 8 hours | `Concentration, up to 8 hours` | timed 8 hours, `concentrationUpTo` | 2 | FR-052 | happy |
| concentration 1 day | `Concentration, up to 1 day` | timed 1 day, `concentrationUpTo` | 2 | FR-052 | happy |
| concentration 2 hours | `Concentration, up to 2 hours` | timed 2 hours, `concentrationUpTo` | 1 | FR-052 | happy |
| concentration 6 rounds | `Concentration, up to 6 rounds` | timed 6 rounds, `concentrationUpTo` | 1 | FR-052 | unhappy (boundary) |
| missing comma | `Concentration up to 10 minutes` | timed 10 minutes, `concentrationUpTo` | 1 | FR-053 | unhappy (invalid input) |
| up to, no concentration | `Up to 8 hours` | timed 8 hours, `upTo` | 1 | FR-054 | unhappy (rule interaction) |
| up to 1 minute | `Up to 1 minute` | timed 1 minute, `upTo` | 1 | FR-054 | unhappy (rule interaction) |
| up to 1 hour | `Up to 1 hour` | timed 1 hour, `upTo` | 1 | FR-054 | unhappy (rule interaction) |
| 1 round | `1 round` | timed 1 round, `fixed` | 4 | FR-055 | unhappy (boundary) |
| 1 minute | `1 minute` | timed 1 minute, `fixed` | 15 | FR-055 | happy |
| 10 minutes | `10 minutes` | timed 10 minutes, `fixed` | 7 | FR-055 | happy |
| 1 hour | `1 hour` | timed 1 hour, `fixed` | 26 | FR-055 | happy |
| 8 hours | `8 hours` | timed 8 hours, `fixed` | 14 | FR-055 | happy |
| 24 hours | `24 hours` | timed 24 hours, `fixed` | 13 | FR-055 | happy |
| 1 day | `1 day` | timed 1 day, `fixed` | 1 | FR-055 | happy |
| 7 days | `7 days` | timed 7 days, `fixed` | 1 | FR-055 | happy |
| 10 days | `10 days` | timed 10 days, `fixed` | 5 | FR-055 | happy |
| 30 days | `30 days` | timed 30 days, `fixed` | 1 | FR-055 | happy |
| until dispelled | `Until dispelled` | `untilDispelled` | 12 | FR-055 | happy |
| until dispelled or triggered | `Until dispelled or triggered` | `untilDispelledOrTriggered` | 2 | FR-055 | happy |
| special | `Special` | `special` | 1 | FR-056 | unhappy (boundary) |
| zero rounds | `0 rounds` | problem reported | 0 | FR-051, FR-056 | unhappy (boundary) |
| unknown form | `Until the next dawn` | problem reported | 0 | FR-056 | unhappy (invalid input) |

### Cases: Components combinations (FR-040, FR-046, FR-047; one test per row; counts sum to 339 and give 189 spells with M, 176 + 7 + 6)

| Label | Printed entry | Model | Count | Source | Path |
|---|---|---|---|---|---|
| V, S, M | `Components: V, S, M (...)` | V, S, material | 176 | FR-047 | happy |
| V, S | `Components: V, S` | V, S | 114 | FR-047 | happy |
| V | `Components: V` | V | 32 | FR-047 | unhappy (boundary) |
| V, M | `Components: V, M (...)` | V, material | 7 | FR-047 | unhappy (boundary) |
| S, M | `Components: S, M (...)` | S, material | 6 | FR-047 | unhappy (boundary) |
| S | `Components: S` | S | 4 | FR-047 | unhappy (boundary) |
| singular label | `Component: V, S` | V, S, recorded in notes | included above (12 in all) | FR-046 | unhappy (invalid input) |
| wrong order | `Components: S, V` | problem reported | 0 | FR-040 | unhappy (invalid input) |
| repeated letter | `Components: V, V` | problem reported | 0 | FR-040 | unhappy (invalid input) |
| unknown letter | `Components: V, X` | problem reported | 0 | FR-040 | unhappy (invalid input) |
| empty | `Components:` | problem reported | 0 | FR-040 | unhappy (empty and minimal) |
| M without text | `Components: V, S, M` | problem reported | 0 | FR-041 | unhappy (invalid input) |
| text without M | `Components: V, S (a feather)` | problem reported | 0 | FR-041 | unhappy (invalid input) |

### Cases: Material components (FR-042 to FR-045; one test per row)

Descriptions are as printed in the cited entry.

| Label | Description (as printed) | `costsGp` | `consumesMaterial` | Source | Path |
|---|---|---|---|---|---|
| no cost | `a morsel of food` | [] | false | Animal Friendship, p. 107 | happy |
| cost, kept | `a diamond worth 50+ GP` | [50] | false | Chromatic Orb, p. 115 | happy |
| cost, consumed | `gold dust worth 25+ GP, which the spell consumes` | [25] | true | Arcane Lock, p. 110 | happy |
| thousands separator | `an agate worth 1,000+ GP, which the spell consumes` | [1000] | true | Awaken, p. 111 | unhappy (boundary) |
| two costs, all consumed | `for each of the spell’s targets, one jacinth worth 1,000+ GP and one silver bar worth 100+ GP, all of which the spell consumes` | [1000, 100] | true | Astral Projection, p. 111 | unhappy (rule interaction) |
| two costs, partly consumed | `incense worth 250+ GP, which the spell consumes, and four ivory strips worth 50+ GP each` | [250, 50] | true | Legend Lore, p. 144 | unhappy (rule interaction) |
| two costs, none consumed | `a chest, 3 feet by 2 feet by 2 feet, constructed from rare materials worth 5,000+ GP, and a Tiny replica of the chest made from the same materials worth 50+ GP` | [5000, 50] | false | Secret Chest, p. 160 | unhappy (rule interaction) |
| cost inside a phrase | `one 150+ GP black onyx stone for each corpse` | [150] | false | Create Undead, p. 121 | unhappy (boundary) |
| non-GP cost | `2 Copper Pieces, which the spell consumes` | [] | true | Gentle Repose, p. 135 | unhappy (rule interaction) |
| empty description | `` | an error matching `srderr.ErrInvalidInput` | n/a | FR-041 | unhappy (invalid input) |

### Cases: Spot-check spells (FR-085, one test per row, source FR-085)

A fixture asserts the fields shown (checked in full against the PDF), the page, the block kinds
present and the first and last words of the description (read from the PDF when the fixture is
written). Scrying (p. 160, two tables of 3 rows) and Augury (p. 111, 4 rows) join FR-068.

| Label | Page | Level, school, classes | Casting Time | Range | Components | Duration | Notes | Path |
|---|---|---|---|---|---|---|---|---|
| Acid Arrow | 107 | 2 Evocation; Wizard | Action | 90 feet | V, S, M (powdered rhubarb leaf) | Instantaneous | has `higherLevel` | happy |
| Acid Splash | 107 | 0 Evocation; Sorcerer, Wizard | Action | 60 feet | V, S | Instantaneous | has `cantripUpgrade`; level 0 | happy |
| Aid | 107 | 2 Abjuration; Bard, Cleric, Druid, Paladin, Ranger | Action | 30 feet | V, S, M (a strip of white cloth) | 8 hours | header wraps (FR-019) | unhappy (boundary) |
| Alarm | 107 | 1 Abjuration; Ranger, Wizard | 1 minute, ritual | 30 feet | V, S, M (a bell and silver wire) | 8 hours | ritual; run-in entries "Audible Alarm", "Mental Alarm" | happy |
| Alter Self | 107 | 2 Transmutation; Sorcerer, Wizard | Action | Self | V, S | Concentration, up to 1 hour | run-in entries | happy |
| Animal Friendship | 107 | 1 Enchantment; Bard, Druid, Ranger | Action | 30 feet | V, S, M (a morsel of food) | 24 hours | text kept as printed (FR-064); heading p. 107, quoted sentence p. 108 | unhappy (invalid input: source typo) |
| Animal Messenger | 108 | 2 Enchantment; Bard, Druid, Ranger | Action, ritual | 30 feet | V, S, M (a morsel of food) | 24 hours | interleaved with Animate Dead (FR-067) | unhappy (rule interaction: column layout) |
| Animate Dead | 108 | 3 Necromancy; Cleric, Wizard | 1 minute | 10 feet | V, S, M (a drop of blood, a piece of flesh, and a pinch of bone dust) | Instantaneous | interleaved with Animal Messenger | unhappy (rule interaction: column layout) |
| Creation | 121 | 5 Illusion; Sorcerer, Wizard | 1 minute | 30 feet | V, S, M (a paintbrush) | Special | `special` duration (FR-056); Materials table, 5 rows (FR-068) | unhappy (boundary) |
| Dream | 126 | 5 Illusion; Bard, Warlock, Wizard | 1 minute | Special | V, S, M (a handful of sand) | 8 hours | `special` range | unhappy (boundary) |
| Gentle Repose | 135 | 2 Necromancy; Cleric, Paladin, Wizard | Action, ritual | Touch | V, S, M (2 Copper Pieces, which the spell consumes) | 10 days | non-GP cost; flagged M on three class lists (FR-097) | unhappy (rule interaction) |
| Mirage Arcane | 149 | 7 Illusion; Bard, Druid, Wizard | 10 minutes | Sight | V, S | 10 days | `sight` range; days unit | unhappy (boundary) |
| Plant Growth | 153 | 3 Transmutation; Bard, Druid, Ranger | Action ("Overgrowth") or 8 hours ("Enrichment") | 150 feet | V, S | Instantaneous | two options (FR-033) | unhappy (rule interaction) |
| Sending | 161 | 3 Divination; Bard, Cleric, Wizard | Action | Unlimited | V, S, M (a copper wire) | Instantaneous | `unlimited` range | unhappy (boundary) |
| Word of Recall | 175 | 6 Conjuration; Cleric | Action | 5 feet | V | Instantaneous | smallest distance; last-page entry | unhappy (boundary) |
| Zone of Truth | 175 | 2 Enchantment; Bard, Cleric, Paladin | Action | 60 feet | V, S | 10 minutes | last entry in the section | happy |
| Barkskin | 112 | 2 Transmutation; Druid, Ranger | Bonus Action | Touch | V, S, M (a handful of bark) | 1 hour | prints "Component:" (FR-046) | unhappy (invalid input: source typo) |

## 5. Model

- **Value objects (package `content`):** `Spell`, `SpellLevel`, `SpellSchool`,
  `SpellCastingTimes` with `CastingTimeMode`, `CastingTime`, `SpellRange`, `SpellComponents`,
  `MaterialComponent`, `SpellDuration` with their variants (FR-026, FR-030,
  FR-036, FR-050) and `TextBlock` (used, FR-059), `DistanceUnit`, `DurationLimit`, `SpellLevelParse` (FR-119) and the
  derived `ClassSpellList` (FR-120); names and factories: FR-114. `SourceCitation`, `SpellID`,
  `ClassID` and `DurationUnit` are used, not defined (FR-113, FR-119). The pure function
  `DeriveClassSpellList` (FR-120, FR-121) is the only domain service; there is no aggregate, entity
  or event.
- **`Spell` fields:** `id`, `name`, `level`, `school`, `classes` (set of `ClassID`),
  `castingTimes`, `ritual`, `range`, `components`, `duration`, `description`, `higherLevel` and
  `cantripUpgrade` (optional block lists), `source` (optional), `effects` (FR-122, default `TextOnlyEffects`). `key` is `id.Key()`;
  `requiresConcentration` is derived.
- **Invariants:** constructors return an error matching `srderr.ErrInvalidInput` (FR-009, FR-027 to FR-029, FR-037, FR-041,
  FR-051, FR-058, FR-086, FR-098, FR-101, FR-102); the label rule and "Concentration
  without a cap" are unrepresentable by type (FR-030, FR-052). Completeness, counts, class-list
  agreement, citations within pp. 107-175 and "level 0 has no higher-level text" are **bundle
  verification**, not constructor invariants (FR-007, FR-018, FR-063).
- **Failures:** an invalid value passed to a `NewX` constructor is an error matching `srderr.ErrInvalidInput` (`MustX` panics with it); untrusted JSON is a decoder result listing
  every problem (FR-109; `NewSpellLevel`, `ParseSpellID`: FR-010, FR-113); the tool reports a problem list and exits
  non-zero (FR-073); an absent spell is `LookupNotFound` from the port.
- **Purity:** no goroutine, `context`, clock, random or I/O (FR-088). The schema spec's
  earlier spell rows are withdrawn there and specified here (FR-092, FR-094, FR-115); ids are
  `acid_arrow`, never `acid-arrow` (FR-003).

### 5.2 JSON shape (normative; FR-107 to FR-111)

Key order is the declared order and the problem-reporting order. `?` means
optional, by absence. Strings in quotes are the exact accepted values.

| Object | Keys in order | Rules |
|---|---|---|
| spell | `id`, `name`, `source?`, `level`, `school`, `classes`, `castingTimes`, `ritual`, `range`, `components`, `duration`, `description`, `higherLevel?`, `cantripUpgrade?`, `effects?` | `id`: the id pattern (FR-113). `name`: FR-098. `level`: integer 0 to 9. `school`: "Abjuration", "Conjuration", "Divination", "Enchantment", "Evocation", "Illusion", "Necromancy", "Transmutation". `classes`: array of ids, any length, no repeats (FR-110). `ritual`: boolean. `description`: array of text blocks, at least 1. `higherLevel`, `cantripUpgrade`: array of text blocks, at least 1 when present. `effects`: the spell-effect-data.md object (FR-122). |
| source | `document`, `section`, `page` | Non-blank strings; integer at least 1 (content-schema-and-decoder.md FR-115). All three required when `source` is present. |
| castingTimes element | `kind`, `amount?`, `trigger?`, `label?` | `kind`: "magicAction", "bonusAction", "reaction", "minutes", "hours". `amount`: integer at least 1, required for "minutes" and "hours", otherwise not allowed. `trigger`: non-blank, required for "reaction", optional for "bonusAction", otherwise not allowed. `label`: non-blank; required on every element when the array has 2 or more, not allowed when it has 1. |
| range | `kind`, `amount?`, `unit?` | `kind`: "self", "touch", "distance", "sight", "unlimited", "special". `amount` (integer at least 1) and `unit` ("feet", "miles") are required for "distance" and not allowed otherwise. |
| components | `verbal`, `somatic`, `material?` | Booleans; `material` present iff the spell has M (FR-049). |
| material | `description`, `costsGp`, `consumesMaterial` | `description` non-blank; `costsGp` array of integers at least 1, possibly empty; `consumesMaterial` boolean. |
| duration | `kind`, `amount?`, `unit?`, `limit?` | `kind`: "instantaneous", "timed", "untilDispelled", "untilDispelledOrTriggered", "special". `amount` (integer at least 1), `unit` ("rounds", "minutes", "hours", "days") and `limit` ("fixed", "upTo", "concentrationUpTo") are required for "timed" and not allowed otherwise. |
| text block | `kind` plus the keys below | `kind`: "paragraph" (`text`), "entry" (`name`, `text`), "bulletList" (`items`: array of non-blank strings, at least 1), "table" (`title?` non-blank, `header`: array of non-blank strings, at least 1; `rows`: array of at least 1 arrays of strings, each as long as `header`, cells may be empty), "embeddedStatBlock" (`lines`: array of non-blank strings, at least 1). `text` and `name` follow content-schema-and-decoder.md FR-027 and FR-028. |

## 6. Interface

- **No new port and no use case.** Spells are reached through `content.Source.Lookup` and
  `ListKind` (content-source-port.md); the port failure contract is unchanged. A `LookupFound`
  carrying a non-`Spell` entry under key kind `spell` is a content-modelling bug. All types and
  `DeriveClassSpellList` are synchronous.
- **Exports:** package `content` exports the types of section 5
  with doc comment (names in FR-026, FR-030, FR-036, FR-050, FR-114, FR-119, FR-120): the
  published language of `content` that `character` may use. `SourceCitation`, `SpellID`, `ClassID`
  and `DurationUnit` come from the schema spec's entries; `data.BundledSource` is not specified
  here; `content/contenttest` adds nothing (tests build `Spell` values with a builder in the test package).
- **Extraction tool** (dev tooling, not exported): `tools/extract_spells` (a Go `main` package) with `--pdf <path>`,
  `--out <dir>` (default `content/data/`, FR-105) and `--check` (FR-078). It needs
  poppler (`pdftohtml -xml`, `pdftotext`; FR-100, FR-079), installed only in the Docker container
  that runs it (Q9). Output: the JSON files, a report (counts, hyphen resolutions, singular-label
  spells, findings) and the exit code of FR-073; the corrections log and notes are FR-116's.

## 7. Observability

Silent by design (`doc/rules/observability.md` §3): `Spell`, its parts and `ClassSpellList` are
pure domain values with no use case, port or adapter, and the bundled data is inert. The
extraction tool is dev tooling outside the public packages (FR-073), so it is not in
`doc/observability/signals.md`. Decode failures, entries loaded and lookups by outcome are the
signals of content-schema-and-decoder.md and content-source-port.md (section 7 of each). Wrong
bundled data is caught before release by the verification FRs; each spell cites its page (FR-008).

## 8. Test plan

Levels are those of `doc/rules/testing.md` §3. "adapter" means contract-level tests in
`content/data/bundled_spells_test.go`, reading the bundled data through
`data.BundledSource` and the decoder from the embedded documents and the FR-116 fixtures only, so they
run on any platform; checks using `os` are "architecture" tests (FR-117).

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002, FR-004, FR-006, FR-008, FR-082, FR-083, FR-085 | adapter | 339 decoded spells; pinned name list (set equality both ways); every id equals the derivation of its name; exact names; every citation has document `SRD 5.2.1`, the section text and a page 107 to 175; `Lookup` of each of the 339 keys returns `LookupFound` with an equal key (contract-suite registration is content-source-port.md FR-047's); `ListKind` size and order; wrong-kind `LookupNotFound`; spot-check table, one test per row |
| FR-003, FR-005, FR-071 to FR-077 | unit | `tools` tests: id-derivation table (tool and `SpellID` agree, no pattern copy), collision report on synthetic names; preflight on wrong page count, anchors, footer; pure transform twice compares bytes; seeded problems all reported and no file written; missing entry, unparsable header, mismatched entry counts, out-of-order pages and names; pure log applier (entry applied, unmatched entry rejected); the bundle has no id collision |
| FR-007, FR-009 to FR-012, FR-017, FR-018, FR-058, FR-086, FR-087, FR-098, FR-101, FR-102, FR-122 | unit | `Spell` with an absent and a valid citation, any `ClassID`, an empty class set; `SpellLevel` table, `NewSpellLevel` error cases, `IsCantrip`; `SpellSchool` has 8 pinned names (a `String()` test over every value); mutating each returned slice and each input after construction; order-insensitive set equality; equality over each field (`Equal` where a slice is held); `Spell` with empty description and empty, whitespace, valid name; text block rows (blank paragraph, entry name or text, empty and blank-item bullet list, ragged table, empty header, zero rows, blank title, empty stat block, valid minimal of each kind); `costsGp` 0, -1, 1; empty `higherLevel`/`cantripUpgrade` returns an error, absent and non-empty accepted; `Spell.effects` defaults to the zero-value `TextOnlyEffects` (equal), a given value is kept, and an absent `effects` key decodes to it (FR-122) |
| FR-013, FR-019 to FR-021, FR-100, FR-103 | unit | `tools/` header tests (`spell_header_test.go`) header-form rows; run-in heading and indent detection on the Alarm fixture and a synthetic page (a line matching no rule is a problem); column-interleaved p. 108 fixture |
| FR-014 to FR-016, FR-022 to FR-025, FR-048, FR-084, FR-097 | adapter | Distributions, components counts (189, 56, 31), one test per value; every bundled class id is one of the eight and every spell has a non-empty class set; Cross-check against `testdata/srd/spells/class_list_oracle.go` (checked in, derived once from the PDF and reviewed; the tool regenerates it under `--check`): C, R, M flags; exactly the four membership and three M-flag findings, any other fails. A log entry per finding is an architecture check (FR-117) |
| FR-026 to FR-039 | unit, adapter | A `switch` with class patterns over each closed type returns a distinct label per variant (5, 6, 5, 5 and, for `SpellCastingTimes`, 2) and the factories return the variants (FR-114); `reaction`, `bonusAction`, `minutes`/`hours` and `distance` with 0, -1, 1, empty, whitespace, valid; `SpellCastingTimes` with zero, one (returns an error), two, three modes and a blank label; Casting Time and Range tables (parser rows, problem rows, adapter-level count per row, 29 with Ritual); Plant Growth and no other spell with two options; wrapped trigger and the eight triggered spells |
| FR-040 to FR-047, FR-049 | unit, adapter | Components and Material tables: order, repeats, unknown letter, empty, M without text, text without M, empty `MaterialComponent`; verbatim with joined line breaks; `costsGp`; `consumesMaterial`; Gentle Repose; combination counts one per row; `SpellComponents` with and without material; both labels parse, (FR-117, `os` test): the 12 spells are in the notes |
| FR-050 to FR-057 | unit, adapter | `timed` with 0, -1, 1 and each unit; `DurationLimit.values` pinned, `timed` has no flag parameters; Duration table: parser, problem and Concentration rows; adapter-level counts; Protection from Evil and Good, the three "Up to" spells, 133 requiring concentration, Creation |
| FR-059, FR-060, FR-064 to FR-070 | unit, adapter | Parser on synthetic multi-paragraph, bullet, cross-reference, table and interleaved pages; normalisation allows only the permitted changes; footer removal; hyphen rows (joined attested: join; hyphenated attested: keep; both: join; neither: unresolved problem, cleared by a log entry; soft hyphen removed first); adapter-level: Animal Friendship text as printed, no field containing the footer, ten table pages with row counts (Augury 4, Creation 5, Scrying 3 each), four stat blocks inside their spells |
| FR-061 to FR-063 | adapter | 109 `higherLevel` spells per level and 15 `cantripUpgrade` spells; Acid Arrow and Acid Splash; level versus text-kind check over the bundle; unit test of the heading strip |
| FR-078 to FR-080 | process (manual; outside testing.md §3) | Local run with the PDF, and human review; see "Not covered" |
| FR-081, FR-088 to FR-091, FR-104 to FR-106, FR-116 to FR-118 | architecture | where `os` is used: the ten spell files are among those read (decoder FR-089 covers the rest); domain purity; placement, export and `tools/` not under a public package; data-path and consumer-name scan; `NOTICE.md` data paths and changes statement; no spell-specific loader; tool default output path and imports; embedded JSON documents decode to the 339 entries; fixture libraries and log files exist, the module's release file list excludes `doc/research/`; no `os` in `content/data` tests; `tools` tests exists with `Requirement:` comments (and the testing.md §10 amendment, doc review) |
| FR-092, FR-094, FR-115 | doc review, adapter | R1-spells is in the revision table and set to landed by the accepting change; the decoder accepts the current `schemaVersion` only and rejects `higherLevelSlot` as `unknownField`; the twelve thin-slice ids each appear once and its pins are replaced; no `SpellScaling` and no `scaling` key (exported-name and key-list scan) |
| FR-107 to FR-111, FR-113 | adapter, unit | The JSON `content.Decoder` on a valid spell with every optional key and on the rows of "Cases: spell JSON" (path, kind, problem order), platform-independent (no OS-specific behaviour, content-schema-and-decoder.md FR-099); every spelling of section 5.2 decodes, wrong case is `invalidValue`; the same table drives the agreement test of content-schema-and-decoder.md FR-094; constants and predicates pinned at their boundaries; `Spell.id` is a `SpellID`, `Spell.key` equals `id.Key()` |
| FR-114, FR-119 | architecture, unit | Exported-name scan: the exported names exist, none duplicated or generic; factories return the variants; unit and limit enum values pinned (`DistanceUnit`, `DurationLimit`); `NewSpellLevel` results for -1, 0, 9, 10 |
| FR-120, FR-121 | unit, adapter | Derivation over a synthetic spell set (non-caster and unknown `ClassID` give empty groups; shuffled input equal; ordinal order; defensively copied; input unchanged); over the bundle the eight lists have the sizes of the class row of "Distributions" |

- **Doubles and suites:** none new; the bundled source is registered with the contract suites
  (content-source-port.md FR-047). **Mutation testing:** package `content` MUST reach at
  least 80 with `gomu` (run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors; testing.md §8); the tool parsers (`tools` tests) SHOULD be mutated too.
  **Telemetry tests:** none (section 7). Each test and group carries a `Requirement:` and `Path:`
  comment.
- **Not covered by automated tests:** FR-078 to FR-080 need the PDF (not in the repository) or a
  person; they are enforced by `--check` run locally, the review log in the extraction notes, and
  review of any change to bundled spells. Every other FR has a test.

**Implementation status.**

- Done (Phase 2, step 5g, package `content`, files `spell.go`, `spell_level.go`, `spell_enums.go`, `spell_casting_time.go`, `spell_range.go`, `spell_duration.go`, `spell_components.go`, `spell_list.go`, `decode_spell.go`, `decode_spell_effects.go`, `spell_effects_parse.go`): the types and constructors `Spell` (`NewSpell`, `MustSpell`, `SpellFields`, `ValidSpellName`), `SpellLevel` (`NewSpellLevel`, `MustSpellLevel`, `SpellLevelError`, `IsCantrip`, the constants and `ValidSpellLevel`), `SpellSchool`, `DistanceUnit`, `DurationLimit` (each with `ParseX`), the closed `CastingTime` (five variants), `CastingTimeMode`, the closed `SpellCastingTimes` (`SpellCastingTimesSingle`, `SpellCastingTimesModes`), the closed `SpellRange` (six variants), the closed `SpellDuration` (five variants), `MaterialComponent`, `SpellComponents` and `DeriveClassSpellList` with `ClassSpellList` (plan item P7); the decoder row of `spells` (registered after `feats`, its place in the revision table) with the shape of section 5.2, the `effects` key of spell-effect-data.md section 6.1, the agreement rows of FR-111 and the fuzz seeds. Tested ids: FR-007, FR-009 to FR-012, FR-017, FR-026 to FR-030, FR-036, FR-037, FR-041 and FR-049 (types and decoder), FR-050 to FR-052, FR-057 (the method), FR-058, FR-059, FR-086 to FR-089, FR-092 (R1-spells), FR-098, FR-101, FR-102, FR-107 to FR-111, FR-113 to FR-115, FR-119 to FR-122. The exported names of FR-114 and FR-119 exist once each and the module has no `SpellScaling` (`internal/archtest/spell_types_test.go`, FR-114, FR-115). FR-088 and FR-089 are pinned by `TestSpellFilesArePureAndToolsIsNotImported` in the same file (the `tools/` half holds vacuously until the extraction tool exists). Every closed set is built from value variants only: a nil or a pointer variant is rejected with an error (`pointer_variants_test.go`).
- Choices the spec left open: the function of FR-120 is `DeriveClassSpellList` and the group accessor `IDsAt` (a type and a function cannot both be `ClassSpellList`, plan item P7); the zero `SpellLevel` is the cantrip level and is valid, so `NewSpell` checks the level it is given; the factories with an optional or required argument are `NewCastingTimeBonusAction()` (no trigger) and `NewCastingTimeBonusActionTriggered(trigger)`; `SpellComponents` is built by `NewSpellComponents(verbal, somatic)` and `WithMaterial(m)`; `Spell.Classes()` returns the classes in ordinal order, so two spells that differ only in listing order are equal (FR-017), and an empty list of classes, `costsGp` or optional text is stored as none (as the other entries do); a casting time label follows the name rule (schema FR-027); the decoder reads integers up to 2^31 - 1 (outOfRange above); `castingTimes` with one element is a single time (a `label` there is `constraintViolation` at the label) and with two or more is a list of modes, each with a required label; the Spell level check of FR-122 is reported at the `effects` key (spell-effect-data.md section 6.1). Plan item P2 (the rank table and this spec's header omitting spell-effect-data.md) is fixed in the `Depends on` row; the rank table of shared-types-and-ownership.md section 4.11 now ranks spell-effect-data at 5.
- Done (Phase 2, step 6, `content/data/spells.json`): twelve spells of the thin slice (Fire Bolt, Prestidigitation, Detect Magic, Magic Missile, Misty Step, Fireball and the six Elf lineage spells), transcribed from the PDF with run-in headings as `entry` blocks, `higherLevel` and `cantripUpgrade` as text, no `effects`, served by `data.BundledSource` and pinned by content-thin-slice.md (the slice pins FR-003, FR-007, FR-060 are replaced by FR-094 when the 339 land). The extraction tool, the other 327 spells and the counts are not implemented.

Not yet implemented: everything that needs the bundled data or the extraction tool: FR-001 to FR-006, FR-008, FR-013 to FR-016, FR-018 to FR-025, FR-031 to FR-035, FR-038 to FR-040, FR-042 to FR-048, FR-053 to FR-056, FR-060 to FR-085, FR-090, FR-091, FR-094, FR-097, FR-100, FR-103 to FR-106 and FR-116 to FR-118; the extraction half of FR-041 and FR-049 and the count of FR-057; the bundle half of FR-092 (R1-spells is `partially implemented`); the mutation score of the 5g files.
- Mutation (vocabulary-3, manual fallback, 21 mutations): `content/spell_level.go` 21/21 caught after `TestSpellLevelErrorContract` pinned the value returned beside the error, the panic value and the nil-target and nil-receiver `Is` cases; no survivors.
- Mutation (spells-1, manual fallback, 15 to 35 mutations per file, none of the four files has a same-named test): `content/spell_list.go` all caught after `TestClassSpellListEqualComparesEveryLevel` and `TestDeriveClassSpellListSortsTopLevel`; one survivor is equivalent (dropping the `s.level.Valid()` guard in `DeriveClassSpellList`: `NewSpell` rejects an invalid level, so no `Spell` can hold one). `content/spell_range.go`, `content/spell_duration.go` and `content/spell_components.go` all caught after `TestCheckSpellRangeAndDurationRecheckEveryMember` and `TestSpellComponentsMaterialIsRecheckedAndCopied` (internal tests that build the variants by hand); no survivors.
- Mutation (spells-2): `content/decode_spell.go` gomu 78.0 then 81.6 (40 killed, 9 survived) after `TestDecodeSpellCases` pinned that an unknown casting kind hides the label rule; the survivors are equivalent: `reqBool` absent returning true and `decodeMaterial` non-object returning true (the problem is already recorded, so the entry is dropped); the `levelOK` branch (the failed `integer` returns 0, the zero level); `return "", nil` mutated to itself (two); `kind == ""` to `<=` (identical for strings); and `err == nil && clean` to `||` in range, duration and material (a constructor error is always preceded by a recorded problem). Manual fallback, 50 mutations over `content/spell_enums.go`, `content/spell_casting_time.go` and `content/spell.go` (no same-named tests): all caught after `TestSpellVocabularySpellings` and `TestSpellCastingTimeZeroVariantsAndEqualityEdges`, except two equivalent survivors in `spell.go`: `s.id == o.id` in the zero-Spell branch of `Equal` (both ids are zero there) and the `cloneCastingTimes` copy (the modes slice is already copied by its constructor and by `Modes()`).

## 9. Open questions and decisions

No open questions. Settled, referenced by label in the text: Q4 embedded stat blocks stay verbatim blocks, a monster
spec may parse them later (FR-069); Q5 no structured mechanics and no `SpellScaling`, later optional keys are not
breaking (FR-115); Q6 source oddities are kept as printed and listed in the extraction notes (FR-064, FR-116); Q9 text
extraction uses poppler in Docker, outside `go.mod` (FR-100, FR-079); Q10 one data file per level (ten files),
sorted by id (FR-005, FR-105); Q11 the class-list oracle is a checked-in derived Go fixture (FR-116); Q12 the M
flag equals a stated GP cost except Gentle Repose (FR-025, FR-097); Q13 `consumesMaterial` is one flag plus the
verbatim description, not per cost item (FR-044); Q14 class spell lists are derived here from `Spell.classes`
(FR-120; shared-types-and-ownership.md FR-066); Q15 `DurationUnit` is the schema's (FR-119; FR-065 there). Q16 text blocks are the neutral `TextBlock` of the schema spec, no spell-named family (FR-059; content-magic-items.md Q2). Q17 `Spell.effects` (FR-122) is typed by spell-effect-data.md, which does not import `Spell` and ranks below this spec (rank 5 in shared-types-and-ownership.md section 4.11). No Q7 exists here. Q8 (cited by content-equipment.md) is the testing.md §10 `tools` tests amendment, recorded at FR-118; the other specs should cite FR-118 and drop Q7.

## 10. Changelog impact

Under `Unreleased`:
- **Added (Phase 2, types):** every type and function named in section 5 (`Spell`, `SpellLevel`,
  `SpellSchool`, the casting time, range and duration families, `ClassSpellList`,
  `DeriveClassSpellList`; the public packages), plus decoder support for the spell JSON shape
  (row R1-spells, FR-092; no separate `schemaVersion`). Nothing is BREAKING (unpublished 0.0.1).
- **Added (Phase 5, data):** the 339 SRD 5.2.1 spells as bundled content (`NOTICE.md` covers
  `data/`, `content/data/`, `testdata/srd/`), replacing the twelve thin-slice spells
  with unchanged ids (FR-094).

After the first release, adding a value to `SpellSchool`, `DurationLimit`, `CastingTime`,
`SpellRange` or `SpellDuration`, or changing a bundled spell's field values
other than to fix a transcription error, is **BREAKING** (`doc/rules/architecture.md` §7).
