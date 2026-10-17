# Rules Glossary coverage

| | |
|---|---|
| Status | draft |
| Context | `package` |
| Roadmap phase | 5 |
| Depends on | `architecture-guardrails.md` |

## 1. Purpose and scope

The Rules Glossary (SRD 5.2.1, pp. 176-191) is the reference the other rules
cite. [../roadmap.md](../roadmap.md) (Phase 5) puts it in scope "as the
reference for terms the other specs cite". The glossary adds no subsystem: its
entries are rules that other parts of the SRD also state, and the topic specs
each took some and excluded others, so an entry can fall between two specs (no
owner) or be claimed by two (two owners).

This is a **traceability spec**. Section 4.3 has one row per glossary entry, or
per aspect of an entry with more than one owner, naming the single owning spec
and the printed pages. The requirements keep the table true:

- every entry has exactly one owning spec for each of its aspects;
- an owner is an existing spec, a planned spec registered in section 4.5, or
  `none` with a stated reason (vocabulary or notation, no engine rule);
- an entry nobody owns is a `gap` with an open question, never hidden;
- a documentation test (section 8) compares the table with a fixture of the
  printed headings and with the files in `doc/specs/`, so a new, renamed,
  withdrawn or missing owner breaks the build.

The spec defines no domain type, use case, port or event and changes nothing in
any Go package. It owns no rule mechanics, so under `shared-types-and-ownership.md`
FR-048 it **supersedes no requirement of any other spec**; each row names the
topic spec that owns the mechanics (FR-030).

**Out of scope:**

- **Any glossary rule itself.** The table holds names, pages, file names and
  requirement ids; the owner specifies the behaviour.
- **Writing the planned owners** (section 4.5).
- **Whether an owner implements an entry correctly.** The check proves a named
  owner exists and cites a requirement that exists, not that it matches the SRD.
- **The Gameplay Toolbox (pp. 192-203) and Magic Items (pp. 204-253).** Separate
  specs (`shared-types-and-ownership.md` FR-045). They appear here only as
  adjunct rows (section 4.4, FR-029) and add no glossary entry.
- **The SRD's index** ("No Obsolete Terms", p. 176).
- **Material from outside SRD 5.2.1.** Every entry is on pp. 176-191 of the
  PDF. Lair Actions are not in the PDF; left out and flagged (Q-20).

## 2. Source

- SRD 5.2.1, "Rules Glossary", "Glossary Conventions" and "Rules Definitions",
  pp. 176-191: every heading of the two-column entry list.
- Adjunct sources (section 4.4): "The Six Abilities" sidebar "Round Down", p. 5;
  the "Heroic Inspiration" sidebar, p. 8, and "Interactions with Rerolls", p. 8;
  "Curses and Magical Contagions" (including "Environmental Curses", Demonic
  Possession), pp. 193-194; "Underwater Combat" ("Impeded Weapons", "Fire
  Resistance"), p. 16; "Rhythm of Play" and the sidebar "Exceptions Supersede
  General Rules", p. 5; "Creature Size and Space" table, p. 14; "Armor" ("Armor
  Training", "One at a Time", "Strength", "Stealth"), p. 92; "Playing on a Grid",
  "Ranges", p. 13; "Monsters", "Running a Monster", p. 255, "Attack Notation",
  "Saving Throw Effect Notation", p. 256, "Damage Notation", "Multiattack",
  "Bonus Actions", "Reactions", "Spellcasting" (with "Spell Components" and "Casting Times of 1+ Minutes"), "Limited Usage" and "Legendary Actions", p. 257.
- Package inputs: `doc/roadmap.md` (Phase 5), `doc/research/sources.md`,
  `doc/rules/architecture.md` section 2, `doc/rules/testing.md` section 9,
  `AGENTS.md` ("Licensing and source material").

## 3. Vocabulary

| SRD term or package term | Type | Notes |
|---|---|---|
| Rules Glossary | doc concept | pp. 176-191: conventions on p. 176, then definitions in alphabetical order. |
| Tags in Brackets | doc concept | Five tags follow an entry name: Action, Area of Effect, Attitude, Condition, Hazard (p. 176). |
| Entry | doc concept | One glossary heading. 155 entries (FR-001). |
| Unit | doc concept | An entry, or one aspect of an entry owned by more than one spec. A row of the table (FR-004). |
| Aspect | doc concept | A separable part of an entry, e.g. "Cantrip" is data in one spec and casting without a slot in another. |
| Owner | doc concept | The one spec responsible for a unit (FR-005): a file name in `doc/specs/`. |
| Owner state | doc concept | `exists`, `planned` (registered in section 4.5), `none` (no engine rule) or `gap` (an open question names who should own it). |
| Vocabulary-only entry | doc concept | An entry naming a thing with no applicable rule (e.g. "Adventure"). Owner `none` (FR-006). |
| Adjunct row | doc concept | A rule stated outside pp. 176-191 related to an entry (section 4.4). Not among the 155. |
| Narrative guidance | doc concept | SRD text on how to play that states no engine rule ("Rhythm of Play", p. 5). Adjunct row, owner `none`, Where `narrative` (FR-031). |
| Violation | test value | The `Violation` type of `architecture-guardrails.md` (in package `internal/archtest`), not redefined here (FR-019). |
| Warning | test output | A line printed by the check (FR-022); never fails the build. |
| Coverage fixture | test data | `internal/archtest/testdata/srd/rules_glossary_entries.json` (FR-018). |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` section 3).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids.

### 4.1 The table and its rows

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | The coverage table (section 4.3) MUST contain one entry for each heading of the Rules Glossary's "Rules Definitions" and for no other heading: exactly 155 entries, numbered GL-001 to GL-155 in printed order. The "Glossary Conventions" column (p. 176) is row GL-C01 and is not an entry. | SRD 5.2.1, "Rules Glossary", pp. 176-191 | happy |
| FR-002 | An entry's name MUST equal its printed heading without the bracketed tag, and the Tag column MUST hold the printed tag (`Action`, `Area of Effect`, `Attitude`, `Condition` or `Hazard`) or be empty when there is none. | SRD 5.2.1, "Glossary Conventions", "Tags in Brackets", p. 176 | happy |
| FR-003 | An entry's Page cell MUST give the printed page where the entry begins, and the last page where it ends when it continues. Every page MUST be in 176..191. The six continuing entries are Breaking Objects (177-178), Climbing (178-179), Help (182-183), Long Jump (184-185), Ready (186-187) and Stat Block (188-189). The constant list of these six lives in the test and is compared with the fixture; the table never supplies it. A page of 175 or 192 MUST be rejected. | SRD 5.2.1, "Rules Glossary", pp. 176-191 | unhappy (boundary) |
| FR-004 | Every entry MUST have at least one unit (row). An entry whose rule text is divided between specs MUST be split into units `GL-nnna`, `GL-nnnb`, and so on, each with a one-line Aspect. A unit MUST NOT list two owners. | SRD 5.2.1, "Rules Glossary", pp. 176-191; `doc/specs/shared-types-and-ownership.md` FR-005 (one owner per type), FR-048 (the topic-specific spec owns the mechanics) | unhappy (duplicate ownership) |
| FR-005 | A unit's Owner cell MUST hold exactly one of: a file name present in `doc/specs/` (state `exists`); a file name registered in section 4.5 that is not yet present (state `planned`); or the word `none` (state `none`). State `gap` MUST name the interim owner as for `planned` or `exists` and FR-009 applies. | n/a (package design) | unhappy (invalid input) |
| FR-006 | Owner `none` MUST be used only for an entry or aspect that is vocabulary or notation with no engine rule, and the Where cell MUST say `vocabulary` or `notation`. No other reason is accepted, except that an adjunct row (section 4.4) MAY say `narrative` (FR-031). | SRD 5.2.1, "Glossary Conventions", p. 176 (the "You", "See Also" and abbreviation conventions are notation) | unhappy (invalid input) |
| FR-007 | Every planned owner MUST appear in section 4.5 with its target file name, its assumed scope and, if a draft exists under a different file name, that name. When the file `doc/specs/<name>.md` exists, the same change MUST set the state of every row naming it to `exists`. | n/a (package design) | unhappy (state error) |
| FR-008 | The Where cell of a unit with state `exists` MUST name a section number or one or more FR ids of the owner (or the word `scope` where the owner states the unit only in its section 1 or section 3). Every cited FR id MUST exist in the owner's requirements table and MUST NOT be a withdrawn id (a row whose Requirement cell begins `Withdrawn:`, `architecture-guardrails.md` FR-087). A cited section number MUST be a heading of the owner (a requirements section with live FR rows, or a vocabulary or scope section). A Q id written in a Where cell of a unit with state `exists` (for example "section 9, Q-23 of the owner") is the owner's and is not resolved by FR-009, which resolves Q ids only for state `gap` in section 9 of this spec. A unit with state `planned` MUST say `planned`. | `doc/specs/architecture-guardrails.md` FR-071 (what a requirements table is) | unhappy (missing content) |
| FR-009 | A unit with state `gap` MUST cite an open question of section 9 in its Where cell, and that question MUST exist. The number of `gap` units MUST be published in section 4.6. An "exported rule" is a unit whose interim or planned owner spec has Status `accepted` or `implemented`; a release (`doc/specs/release-0-1-0.md`) MUST NOT be tagged while such a unit is a `gap`. The gate is `release-0-1-0.md` FR-097. | n/a (package design) | unhappy (state error) |
| FR-010 | Where an existing spec states, in its purpose, scope or out-of-scope list, that a unit belongs to a different owner than the table shows, the disagreement MUST be recorded in section 9 and MUST NOT be resolved by redefining the unit in this spec or by editing the other spec. A unit that is only a reference to an existing spec's requirement MUST cite it instead of defining it again. FR-028 checks this mechanically. | `doc/specs/shared-types-and-ownership.md`, section 1 (one owner per type; redefining is a defect) | unhappy (rule interaction) |
| FR-011 | Rows MUST appear in the printed order of the entries (page, then position on the page), units of one entry adjacent and in alphabetical suffix order. The order MUST equal the order of the coverage fixture. | SRD 5.2.1, "Rules Glossary", pp. 176-191 | unhappy (state error) |
| FR-012 | Table cells MUST hold names, tags, pages, file names, requirement ids, section numbers and one-line aspects only. No cell MAY reproduce rule text, and no Where or Aspect cell MAY exceed 200 characters. | `AGENTS.md`, "Licensing and source material" (cite, do not copy); `doc/rules/testing.md` section 5 | unhappy (invalid input) |

### 4.2 Tag families

The five tags mark families that must each be owned consistently.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-013 | The Action family MUST be exactly twelve entries tagged `Action`: Attack, Dash, Disengage, Dodge, Help, Hide, Influence, Magic, Ready, Search, Study and Utilize. The constant list of the twelve names lives in the test and is compared with the entries the coverage fixture (FR-018) tags `Action`; the table never supplies it. | SRD 5.2.1, "Action", p. 176 | happy |
| FR-014 | The Condition family MUST be exactly fifteen entries tagged `Condition`: Blinded, Charmed, Deafened, Exhaustion, Frightened, Grappled, Incapacitated, Invisible, Paralyzed, Petrified, Poisoned, Prone, Restrained, Stunned and Unconscious. Each unit of each of those entries, and of "Condition" (GL-032), MUST have owner `conditions.md`. | SRD 5.2.1, "Condition", p. 179 | happy |
| FR-015 | The Area of Effect family MUST be exactly six entries tagged `Area of Effect`: Cone, Cube, Cylinder, Emanation, Line and Sphere. The vocabulary unit "Area of Effect" (GL-008a) and all six MUST have the same owner, `spell-effect-data.md` (FR-013, FR-014), which defines the six shapes (moved there from `spell-effects.md` by `shared-types-and-ownership.md` FR-059). The origin placement and unseen-origin rule (GL-008b) is a different unit with its own owner (Q-14). | SRD 5.2.1, "Area of Effect", p. 177 | happy |
| FR-016 | The Hazard family MUST be exactly five entries tagged `Hazard`: Burning, Dehydration, Falling, Malnutrition and Suffocation. "Hazard" (GL-073) and all five MUST have the same owner. | SRD 5.2.1, "Hazard", p. 182 | happy |
| FR-017 | The Attitude family MUST be exactly three entries tagged `Attitude`: Friendly, Hostile and Indifferent. "Attitude" (GL-013) and all three MUST have the same owner. | SRD 5.2.1, "Attitude", p. 177 | happy |

### 4.3 The coverage table

Columns: unit id, entry, tag, printed page(s), aspect (blank for the whole
entry), owner file, where in the owner, state. Section numbers and ids were read
from the owners' files when this spec was reconciled with
`shared-types-and-ownership.md`; every owner file exists.

| Id | Entry | Tag | Page | Aspect | Owner | Where | State |
|---|---|---|---|---|---|---|---|
| GL-C01 | Glossary Conventions; Abbreviations table | | 176 | | none | notation | none |
| GL-001 | Ability Check | | 176 | | d20-tests.md | FR-023, FR-024 | exists |
| GL-002 | Ability Score and Modifier | | 176 | | ability-scores.md | FR-017, FR-024 | exists |
| GL-003 | Action | | 176 | the list of twelve and recording that one was taken | initiative-and-actions.md | FR-044, FR-050 | exists |
| GL-004 | Advantage | | 176 | | modifiers-and-advantage.md | FR-011, FR-015 | exists |
| GL-005 | Adventure | | 176 | | none | vocabulary | none |
| GL-006a | Alignment | | 176 | the `Alignment` values (the stat block field is part of GL-136) | content-schema-and-decoder.md | FR-137 | exists |
| GL-006b | Alignment | | 176 | a player character's alignment | character-creation.md | FR-067 | exists |
| GL-007 | Ally | | 176 | | none | vocabulary | none |
| GL-008a | Area of Effect | | 177 | the six shapes, their dimensions, origin included or not | spell-effect-data.md | FR-013, FR-014 | exists |
| GL-008b | Area of Effect | | 177 | origin placement, the unseen origin, a line blocked only by Total Cover | magical-effects.md | FR-016 to FR-018, FR-024 | exists |
| GL-009a | Armor Class | | 177 | the target number of an attack roll | d20-tests.md | FR-022, FR-050 | exists |
| GL-009b | Armor Class | | 177 | the base calculation from armor, Dexterity and Shield | equipment-rules.md | FR-001 to FR-005 | exists |
| GL-010a | Armor Training | | 177 | Armor Class with and without training (Shield bonus) | derived-stats.md | FR-035, FR-039 | exists |
| GL-010b | Armor Training | | 177 | no spellcasting in untrained armor | casting-and-concentration.md | FR-063 | exists |
| GL-010c | Armor Training | | 177 | the rule and its roll-circumstance output: Disadvantage on Strength and Dexterity D20 Tests (derived-stats.md applies it) | equipment-rules.md | FR-071 to FR-078 | exists |
| GL-011 | Attack | Action | 177 | one attack; Equipping and Unequipping Weapons; Moving between Attacks | attack-and-help-actions.md | FR-015, FR-023, FR-025 to FR-034 | exists |
| GL-012 | Attack Roll | | 177 | | attacks.md | section 4.1 | exists |
| GL-013 | Attitude | | 177 | | social-interaction.md | FR-001 to FR-005 | exists |
| GL-014 | Attunement | | 177 | | magic-item-rules.md | FR-038, FR-048 | exists |
| GL-015 | Blinded | Condition | 177 | | conditions.md | section 4.7 | exists |
| GL-016 | Blindsight | | 177 | | hiding-searching-and-senses.md | FR-028 to FR-032 | exists |
| GL-017 | Bloodied | | 177 | | derived-stats.md | FR-071 | exists |
| GL-018 | Bonus Action | | 177 | | initiative-and-actions.md | section 4.5 | exists |
| GL-019 | Breaking Objects | | 177-178 | Object AC and Hit Points tables, damage types and objects, Damage Threshold, no ability scores | environment-hazards-and-objects.md | FR-066 to FR-073, FR-087, FR-088 | exists |
| GL-020 | Bright Light | | 178 | the rule (`ObscurementOf`); the `LightLevel` member is content-schema-and-decoder.md FR-133 | hiding-searching-and-senses.md | FR-003 | exists |
| GL-021 | Burning | Hazard | 178 | | environment-hazards-and-objects.md | FR-005 to FR-013 | exists |
| GL-022a | Burrow Speed | | 178 | using the speed to move, and refusal without the speed | initiative-and-actions.md | FR-082 | exists |
| GL-022b | Burrow Speed | | 178 | sand, earth, mud or ice only; solid rock needs a trait | movement-lifting-and-flight.md | FR-048 to FR-051 | exists |
| GL-023 | Campaign | | 178 | | none | vocabulary | none |
| GL-024a | Cantrip | | 178 | a level 0 spell as data | content-spells.md | FR-011 | exists |
| GL-024b | Cantrip | | 178 | cast without a spell slot | casting-and-concentration.md | FR-062 | exists |
| GL-025a | Carrying Capacity | | 178 | the table by size and Strength, the Drag/Lift/Push maximum, reporting the 5-foot limit | equipment-rules.md | FR-011 to FR-020 | exists |
| GL-025b | Carrying Capacity | | 178 | applying the limit to a Speed (the arithmetic) | movement-lifting-and-flight.md | FR-001, FR-058 | exists |
| GL-026 | Challenge Rating | | 178 | | content-monsters.md | section 4.5 | exists |
| GL-027 | Character Sheet | | 178 | | none | vocabulary | none |
| GL-028 | Charmed | Condition | 178 | | conditions.md | section 4.7 | exists |
| GL-029a | Climbing | | 178-179 | extra movement cost | initiative-and-actions.md | FR-077 | exists |
| GL-029b | Climbing | | 178-179 | the optional DC 15 Strength (Athletics) check | movement-lifting-and-flight.md | FR-031 | exists |
| GL-030 | Climb Speed | | 179 | | initiative-and-actions.md | FR-077, FR-082 | exists |
| GL-031a | Concentration | | 179 | Concentration on a spell or on a non-spell effect: the rule that ends it (Incapacitated, Dead, a second effect), the damage save, one at a time | casting-and-concentration.md | FR-075 to FR-079, FR-099 to FR-101, FR-119, FR-120 | exists |
| GL-031b | Concentration | | 179 | only the reported Incapacitated flag, a fact and not the ending rule | conditions.md | FR-065 | exists |
| GL-032 | Condition | | 179 | definition and no stacking with itself | conditions.md | FR-012 | exists |
| GL-033 | Cone | Area of Effect | 179 | | spell-effect-data.md | FR-013, FR-014 | exists |
| GL-034a | Cover | | 179 | the three degrees and their bonuses | attacks.md | section 4.2 | exists |
| GL-034b | Cover | | 179 | Total Cover blocks a line of an area of effect | magical-effects.md | FR-021, FR-022 | exists |
| GL-035 | Crawling | | 179 | | initiative-and-actions.md | FR-077 | exists |
| GL-036 | Creature | | 179 | | none | vocabulary | none |
| GL-037 | Creature Type | | 179 | | content-schema-and-decoder.md | FR-105 | exists |
| GL-038 | Critical Hit | | 179 | | attacks.md | FR-008, FR-010, FR-056 to FR-059 | exists |
| GL-039 | Cube | Area of Effect | 179 | | spell-effect-data.md | FR-013, FR-014 | exists |
| GL-040 | Curses | | 179 | the general rule and how a curse ends | magical-effects.md | FR-027 to FR-033 | exists |
| GL-041 | Cylinder | Area of Effect | 180 | | spell-effect-data.md | FR-013, FR-014 | exists |
| GL-042 | D20 Test | | 180 | | d20-tests.md | FR-016 | exists |
| GL-043 | Damage | | 180 | | hit-points-and-damage.md | section 4.2 | exists |
| GL-044 | Damage Roll | | 180 | | attacks.md | FR-051 to FR-062 | exists |
| GL-045 | Damage Threshold | | 180 | | environment-hazards-and-objects.md | FR-080 to FR-085, FR-112 | exists |
| GL-046 | Damage Types | | 180 | the list of types | content-schema-and-decoder.md | FR-105 | exists |
| GL-047 | Darkness | | 180 | the rule (`ObscurementOf`); the `LightLevel` member is content-schema-and-decoder.md FR-133 | hiding-searching-and-senses.md | FR-005 | exists |
| GL-048 | Darkvision | | 180 | | hiding-searching-and-senses.md | FR-021 to FR-027 | exists |
| GL-049 | Dash | Action | 180 | | initiative-and-actions.md | section 4.4 | exists |
| GL-050a | Dead | | 180 | no Hit Points, terminal | hit-points-and-damage.md | section 4.6 | exists |
| GL-050b | Dead | | 180 | return to life: revived Hit Points, conditions and curses persist, Exhaustion, Attunement | returning-to-life.md | FR-001 to FR-013 | exists |
| GL-051 | Deafened | Condition | 181 | | conditions.md | section 4.7 | exists |
| GL-052 | Death Saving Throw | | 181 | | hit-points-and-damage.md | section 4.7 | exists |
| GL-053 | Dehydration | Hazard | 181 | | environment-hazards-and-objects.md | FR-027, FR-029 to FR-035 | exists |
| GL-054 | Difficult Terrain | | 181 | | initiative-and-actions.md | FR-078, FR-079 | exists |
| GL-055 | Difficulty Class | | 181 | | d20-tests.md | section 4.4 | exists |
| GL-056 | Dim Light | | 181 | the rule (`ObscurementOf`); the `LightLevel` member is content-schema-and-decoder.md FR-133 | hiding-searching-and-senses.md | FR-004 | exists |
| GL-057 | Disadvantage | | 181 | | modifiers-and-advantage.md | FR-012, FR-015 | exists |
| GL-058 | Disengage | Action | 181 | | initiative-and-actions.md | FR-053 | exists |
| GL-059 | Dodge | Action | 181 | | initiative-and-actions.md | FR-037, FR-054, FR-055 | exists |
| GL-060 | Emanation | Area of Effect | 181 | | spell-effect-data.md | FR-013, FR-014 | exists |
| GL-061 | Encounter | | 181 | | encounter-aggregate.md | scope | exists |
| GL-062 | Enemy | | 181 | | none | vocabulary | none |
| GL-063 | Exhaustion | Condition | 181 | | conditions.md | section 4.6 | exists |
| GL-064 | Experience Points | | 182 | | character-aggregate.md | sections 4.2, 4.4 | exists |
| GL-065 | Expertise | | 182 | | derived-stats.md | section 4.3 | exists |
| GL-066 | Falling | Hazard | 182 | including falling into a liquid | environment-hazards-and-objects.md | FR-014 to FR-026 | exists |
| GL-067 | Flying | | 182 | falling when Incapacitated, Prone or Fly Speed reduced to 0; staying aloft by hovering | movement-lifting-and-flight.md | FR-036 to FR-046 | exists |
| GL-068 | Fly Speed | | 182 | | initiative-and-actions.md | FR-082 | exists |
| GL-069 | Friendly | Attitude | 182 | | social-interaction.md | FR-001 to FR-005 | exists |
| GL-070 | Frightened | Condition | 182 | | conditions.md | section 4.7 | exists |
| GL-071 | Grappled | Condition | 182 | | conditions.md | section 4.7 | exists |
| GL-072 | Grappling | | 182 | the record of holder, target and part, the escape check, release, range, and ending the grapple (the Grappled condition entry is GL-071) | grappling.md | FR-006, FR-012, FR-013, FR-018, FR-022 to FR-026, FR-029, FR-033, FR-035 to FR-043, FR-049, FR-050, FR-094 to FR-096 | exists |
| GL-073 | Hazard | | 182 | the tag and its list | environment-hazards-and-objects.md | FR-001 to FR-004 | exists |
| GL-074 | Healing | | 182 | | hit-points-and-damage.md | section 4.5 | exists |
| GL-075 | Heavily Obscured | | 182 | the rule (`ObscurementOf`); the `Obscurement` value is content-schema-and-decoder.md FR-135 (shared-types-and-ownership.md FR-053) | hiding-searching-and-senses.md | FR-011 | exists |
| GL-076 | Help | Action | 182-183 | | attack-and-help-actions.md | FR-035 to FR-050, FR-052 to FR-056 | exists |
| GL-077a | Heroic Inspiration | | 183 | the held or not held state, gaining it with only one at a time, expending it | character-aggregate.md | FR-089 to FR-091 | exists |
| GL-077b | Heroic Inspiration | | 183 | the reroll of one die after it is expended | d20-tests.md | FR-012 to FR-015, FR-059 | exists |
| GL-077c | Heroic Inspiration | | 183 | the instrumented use cases, and giving it to another character | heroic-inspiration.md | FR-004, FR-005, FR-007, FR-031 | exists |
| GL-078 | Hide | Action | 183 | | hiding-searching-and-senses.md | FR-051 to FR-063 | exists |
| GL-079 | High Jump | | 183 | | movement-lifting-and-flight.md | FR-009, FR-013, FR-014, FR-029 | exists |
| GL-080a | Hit Point Dice | | 183 | the pool and the maximum | derived-stats.md | section 4.6 | exists |
| GL-080b | Hit Point Dice | | 183 | spending them on a Short Rest | rests-and-hit-dice.md | scope | exists |
| GL-081 | Hit Points | | 183 | | hit-points-and-damage.md | section 4.1 | exists |
| GL-082 | Hostile | Attitude | 183 | | social-interaction.md | FR-001 to FR-005 | exists |
| GL-083 | Hover | | 183 | | movement-lifting-and-flight.md | FR-042 to FR-044 | exists |
| GL-084 | Illusions | | 183 | | magical-effects.md | FR-042 to FR-046 | exists |
| GL-085a | Immunity | | 183 | to a damage type | hit-points-and-damage.md | section 4.3 | exists |
| GL-085b | Immunity | | 183 | to a condition, taken as an input | conditions.md | section 4.3 | exists |
| GL-086 | Improvised Weapons | | 183 | | attacks.md | FR-046 to FR-049, FR-100 | exists |
| GL-087 | Incapacitated | Condition | 184 | | conditions.md | section 4.7 | exists |
| GL-088 | Indifferent | Attitude | 184 | | social-interaction.md | FR-001 to FR-005 | exists |
| GL-089 | Influence | Action | 184 | the action and the Influence Checks table | social-interaction.md | FR-006 to FR-011 | exists |
| GL-090 | Initiative | | 184 | | initiative-and-actions.md | section 4.1 | exists |
| GL-091 | Invisible | Condition | 184 | | conditions.md | section 4.7 | exists |
| GL-092 | Jumping | | 184 | | movement-lifting-and-flight.md | FR-007 to FR-022 | exists |
| GL-093 | Knocking Out a Creature | | 184 | | hit-points-and-damage.md | FR-050 | exists |
| GL-094 | Lightly Obscured | | 184 | | hiding-searching-and-senses.md | FR-004, FR-008 to FR-010 | exists |
| GL-095 | Line | Area of Effect | 184 | | spell-effect-data.md | FR-013, FR-014 | exists |
| GL-096 | Long Jump | | 184-185 | | movement-lifting-and-flight.md | FR-008, FR-011, FR-012, FR-023 to FR-028 | exists |
| GL-097 | Long Rest | | 185 | | rests-and-hit-dice.md | FR-001 to FR-004 | exists |
| GL-098 | Magic | Action | 185 | casting a spell that has a casting time of an action | spellcasting.md | scope | exists |
| GL-099 | Magical Effect | | 185 | | magical-effects.md | FR-001 to FR-005 | exists |
| GL-100 | Malnutrition | Hazard | 185 | | environment-hazards-and-objects.md | FR-028, FR-036 to FR-045 | exists |
| GL-101 | Monster | | 185 | | content-monsters.md | FR-001 | exists |
| GL-102 | Nonplayer Character | | 185 | | none | vocabulary | none |
| GL-103 | Object | | 185 | | environment-hazards-and-objects.md | FR-072 to FR-078 | exists |
| GL-104a | Occupied Space | | 185 | a space with a creature in it | initiative-and-actions.md | FR-081 | exists |
| GL-104b | Occupied Space | | 185 | a space completely filled by objects | environment-hazards-and-objects.md | section 9, Q-23 of the owner | exists |
| GL-105 | Opportunity Attacks | | 185 | | initiative-and-actions.md | section 4.6 | exists |
| GL-106 | Paralyzed | Condition | 186 | | conditions.md | section 4.7 | exists |
| GL-107 | Passive Perception | | 186 | | derived-stats.md | section 4.5 | exists |
| GL-108 | Per Day | | 186 | | rests-and-hit-dice.md | scope | exists |
| GL-109 | Petrified | Condition | 186 | | conditions.md | section 4.7 | exists |
| GL-110 | Player Character | | 186 | | none | vocabulary | none |
| GL-111 | Poisoned | Condition | 186 | | conditions.md | section 4.7 | exists |
| GL-112a | Possession | | 186 | prevention by Protection from Evil and Good, ending by Dispel Evil and Good | magical-effects.md | FR-037, FR-039 | exists |
| GL-112b | Possession | | 186 | the possessing effect defines how the possession operates (Demonic Possession) | toolbox-curses-and-contagions.md | FR-006 to FR-030, FR-108 | exists |
| GL-113 | Proficiency | | 186 | | derived-stats.md | sections 4.2, 4.3 | exists |
| GL-114 | Prone | Condition | 186 | | conditions.md | section 4.7 | exists |
| GL-115 | Reach | | 186 | | attacks.md | FR-030 to FR-032 | exists |
| GL-116 | Reaction | | 186 | | initiative-and-actions.md | section 4.6 | exists |
| GL-117a | Ready | Action | 186-187 | readied actions and readied moves | initiative-and-actions.md | FR-038, FR-056, FR-057, FR-072 to FR-074 | exists |
| GL-117b | Ready | Action | 186-187 | readying a spell: held energy, Concentration to the next turn | readying-a-spell.md | FR-001 to FR-012 | exists |
| GL-118 | Resistance | | 187 | | hit-points-and-damage.md | section 4.3 | exists |
| GL-119 | Restrained | Condition | 187 | | conditions.md | section 4.7 | exists |
| GL-120 | Ritual | | 187 | | casting-and-concentration.md | FR-073 | exists |
| GL-121 | Round Down | | 187 | the general convention; each owner applies it (dice-and-rolls.md FR-012, conditions.md FR-086, hit-points-and-damage.md FR-014) | none | notation | none |
| GL-122 | Save | | 187 | | d20-tests.md | FR-025, FR-026 | exists |
| GL-123a | Saving Throw | | 187 | the D20 Test, deciding to fail, automatic failure without the score | d20-tests.md | FR-025, FR-026, FR-035, FR-036 | exists |
| GL-123b | Saving Throw | | 187 | the modifier of a character's saving throws | derived-stats.md | section 4.4 | exists |
| GL-124 | Search | Action | 187 | the action and the Search table | hiding-searching-and-senses.md | FR-074 to FR-082 | exists |
| GL-125a | Shape-Shifting | | 187 | the general rule: carry-over and revert (Wild Shape is a class feature) | magical-effects.md | FR-047 to FR-055 | exists |
| GL-125b | Shape-Shifting | | 187 | the four shape-shifting spells once cast: form, Temporary Hit Points, ending | shape-changing-spells.md | FR-024, FR-026, FR-044, FR-048 | exists |
| GL-126 | Short Rest | | 187 | | rests-and-hit-dice.md | FR-001, FR-002 | exists |
| GL-127 | Simultaneous Effects | | 187 | | magical-effects.md | FR-091 to FR-096 | exists |
| GL-128a | Size | | 188 | the six size categories | content-schema-and-decoder.md | FR-105, FR-109 | exists |
| GL-128b | Size | | 188 | the Creature Size and Space table: the space of each size, as static facts of `CreatureSize` (shared-types-and-ownership.md FR-053) | content-schema-and-decoder.md | FR-139 | exists |
| GL-128c | Size | | 188 | an object's size affects its Hit Points | environment-hazards-and-objects.md | FR-067, FR-069 | exists |
| GL-129a | Skill | | 188 | the list of skills and the associated ability | content-schema-and-decoder.md | FR-105, FR-106 | exists |
| GL-129b | Skill | | 188 | proficiency in a skill | derived-stats.md | section 4.3 | exists |
| GL-130 | Speed | | 188 | special speeds, switching speeds, changes to your speeds | initiative-and-actions.md | FR-076, FR-109, section 4.7 | exists |
| GL-131 | Spell | | 188 | | content-spells.md | scope | exists |
| GL-132 | Spell Attack | | 188 | | attacks.md | FR-087 to FR-089, FR-101 | exists |
| GL-133 | Spellcasting Focus | | 188 | | spellcasting.md | FR-068 | exists |
| GL-134 | Sphere | Area of Effect | 188 | | spell-effect-data.md | FR-013, FR-014 | exists |
| GL-135 | Stable | | 188 | | hit-points-and-damage.md | section 4.8 | exists |
| GL-136 | Stat Block | | 188-189 | | content-monsters.md | sections 4.2 to 4.7 | exists |
| GL-137 | Study | Action | 189 | the action and the Areas of Knowledge table | hiding-searching-and-senses.md | FR-083 to FR-091 | exists |
| GL-138 | Stunned | Condition | 189 | | conditions.md | section 4.7 | exists |
| GL-139 | Suffocation | Hazard | 189 | | environment-hazards-and-objects.md | FR-046 to FR-059 | exists |
| GL-140 | Surprise | | 189 | | initiative-and-actions.md | FR-003 | exists |
| GL-141a | Swimming | | 189 | extra movement cost | initiative-and-actions.md | FR-077 | exists |
| GL-141b | Swimming | | 189 | the optional DC 15 Strength (Athletics) check in rough water | movement-lifting-and-flight.md | FR-031 | exists |
| GL-142 | Swim Speed | | 189 | | initiative-and-actions.md | FR-077, FR-082 | exists |
| GL-143 | Target | | 190 | | none | vocabulary | none |
| GL-144 | Telepathy | | 190 | the rule (a monster's range is data) | telepathy.md | FR-001 to FR-020 | exists |
| GL-145a | Teleportation | | 190 | provokes no Opportunity Attack | initiative-and-actions.md | section 4.6 | exists |
| GL-145b | Teleportation | | 190 | the rest of the rule (no movement spent, equipment, occupied destination; where the Teleport spell's arrival table lands is `spell-outcome-tables.md` FR-010 to FR-017) | magical-effects.md | FR-056 to FR-063, FR-066 to FR-070 | exists |
| GL-146 | Temporary Hit Points | | 190 | | hit-points-and-damage.md | section 4.4 | exists |
| GL-147 | Tremorsense | | 190 | | hiding-searching-and-senses.md | FR-033 to FR-039 | exists |
| GL-148 | Truesight | | 190 | | hiding-searching-and-senses.md | FR-040 to FR-048 | exists |
| GL-149a | Unarmed Strike | | 190 | the attack, the Damage and Shove options and the choice of option | attacks.md | section 4.4 | exists |
| GL-149b | Unarmed Strike | | 190 | the Grapple option: DC, saves, size and free-hand pre-checks | grappling.md | FR-094 to FR-096 | exists |
| GL-150 | Unconscious | Condition | 191 | | conditions.md | section 4.7 | exists |
| GL-151a | Unoccupied Space | | 191 | a space with no creature in it | initiative-and-actions.md | FR-081 | exists |
| GL-151b | Unoccupied Space | | 191 | a space not completely filled by objects | environment-hazards-and-objects.md | section 9, Q-23 of the owner | exists |
| GL-152 | Utilize | Action | 191 | the free interaction and the action economy; effects of the action are the host's | initiative-and-actions.md | FR-050 | exists |
| GL-153 | Vulnerability | | 191 | | hit-points-and-damage.md | section 4.3 | exists |
| GL-154 | Weapon | | 191 | the Simple and Martial categories | content-equipment.md | FR-101 | exists |
| GL-155 | Weapon Attack | | 191 | | attacks.md | FR-039 to FR-042, FR-087 | exists |

Notes on the choices above (everything else follows from the rows):

- **Closed vocabularies** (Damage Types, Size, Creature Type, Skill; also LightLevel FR-133, DarknessKind FR-134, Obscurement FR-135, SpecialSense FR-136, Alignment FR-137, HitPointDie FR-132) are owned by
  `content-schema-and-decoder.md` (`shared-types-and-ownership.md` FR-005, FR-046, FR-053); the behaviour that
  reads them is elsewhere. Skill is split: the 18 skills and `Skill.ability` are
  `content-schema-and-decoder.md` FR-106; `derived-stats.md` section 4.3 only
  imports `Skill`.
- **Round Down** (GL-121) keeps owner `none`: each owner applies the convention in
  its own requirement (ids in its Aspect cell); no spec owns a general rounding
  function.
- **Area of Effect**: the six shapes are `spell-effect-data.md` (`AreaShape`);
  only origin placement and the unseen origin (GL-008b) are `magical-effects.md`,
  which defines no second shape type (Q-14).
- **Grappling** (GL-072) is one unit owned by `grappling.md`; `conditions.md` keeps
  only the Grappled entry (GL-071) and `attacks.md` only the Unarmed Strike
  Damage and Shove options (GL-149a); the Grapple option is GL-149b
  (`shared-types-and-ownership.md` FR-029, FR-048).
- **Armor Class and Armor Training**: base AC is `equipment-rules.md` FR-001 to
  FR-005; the Shield bonus and gaps are GL-010a; the casting ban GL-010b; the
  untrained-armor rule and its Disadvantage output GL-010c (decided by
  `shared-types-and-ownership.md` FR-060; `derived-stats.md` applies it).
- **Shape-Shifting** (GL-125) is split: the general carry-over and revert rule is `magical-effects.md` (GL-125a); the four spells' own behaviour once cast is `shape-changing-spells.md` (GL-125b; `shared-types-and-ownership.md` FR-048).
- **Possession** (GL-112) is split by what is stated: prevention and ending by
  named spells are `magical-effects.md`; the possessing effect is the Toolbox
  rule in `toolbox-curses-and-contagions.md`.
- **Concentration** (GL-031a) also covers a non-spell effect (`spellcasting.md`
  FR-118; `casting-and-concentration.md` FR-119, FR-120).
- **Bloodied** is `derived-stats.md` FR-071, not `hit-points-and-damage.md` (`shared-types-and-ownership.md` FR-049). **Knocking Out a Creature** is `hit-points-and-damage.md` FR-050; `attacks.md` FR-111 only exposes the melee flag.
- **Surprise** and **Utilize** each
  have one owner for the engine part; `initiative-and-actions.md` FR-050 records
  Utilize without resolving the object half (the host reports it).
- **Alignment, LightLevel, DarknessKind, Obscurement, SpecialSense, HitPointDie** (FR-132 to FR-137) are `content-schema-and-decoder.md` vocabulary (`shared-types-and-ownership.md` FR-053).
- **Monster** is the `content-monsters.md` entity; **Nonplayer Character** only
  names a monster with a name and personality, so it is `none`.

### 4.4 Adjunct rows

Rules stated outside pp. 176-191 related to a glossary entry. Not entries and
not counted in the 155 (FR-001). Page is the printed page of the non-glossary
text.

| Id | Rule | Page | Related entry | Owner | Where | State |
|---|---|---|---|---|---|---|
| GL-A01 | Round Down (sidebar) | 5 | GL-121 | none | notation | none |
| GL-A02 | Heroic Inspiration sidebar, "Only One at a Time" and "Gaining Heroic Inspiration" | 8 | GL-077a | character-aggregate.md | FR-089 to FR-091 | exists |
| GL-A03 | Underwater Combat, Impeded Weapons | 16 | GL-155 | attacks.md | FR-039 to FR-042 | exists |
| GL-A04 | Underwater Combat, Fire Resistance | 16 | GL-118 | mounted-and-underwater-combat.md | FR-039 to FR-049, FR-070 | exists |
| GL-A05 | Advantage and Disadvantage, "Interactions with Rerolls" | 8 | GL-004, GL-057, GL-077b | d20-tests.md | FR-012 to FR-015, FR-059 | exists |
| GL-A06 | Curses and Magical Contagions, the curse potency benchmark | 193 | GL-040 | toolbox-curses-and-contagions.md | FR-001 to FR-005 | exists |
| GL-A07 | Environmental Curses, Demonic Possession | 193 | GL-112b | toolbox-curses-and-contagions.md | FR-006 to FR-030 | exists |
| GL-A08 | Monsters, Limited Usage: X/Day, Recharge X-Y, Recharge after a Short or Long Rest | 257 | GL-108, GL-136 | monster-limited-use.md | FR-001 to FR-014 | exists |
| GL-A09 | Monsters, Legendary Actions: timing, uses and regaining them | 257 | GL-087, GL-136 | monster-limited-use.md | FR-015 to FR-023 | exists |
| GL-A10 | Monsters, "Running a Monster" sidebar (guidance on using limited abilities, Multiattack, Bonus Actions, Reactions and Legendary Actions) | 255 | GL-136 | none | narrative | none |
| GL-A11 | Playing the Game, Creature Size and Space table | 14 | GL-128b | content-schema-and-decoder.md | FR-139 | exists |
| GL-A12 | Armor, "Armor Training" and "One at a Time" (the p. 92 text the glossary entry repeats) | 92 | GL-010a, GL-010b, GL-010c | equipment-rules.md | FR-071 to FR-078 | exists |
| GL-A13 | Playing the Game, "Rhythm of Play" (the three pillars and the three-step pattern of a session) | 5 | none | none | narrative | none |
| GL-A14 | Playing the Game, sidebar "Exceptions Supersede General Rules" | 5 | none | none | narrative | none |
| GL-A15 | Armor, "Strength" (the Speed reduction of 10 feet: the rule and output here, the arithmetic on a Speed in the second row) | 92 | GL-009b | equipment-rules.md | FR-007 to FR-010 | exists |
| GL-A15b | Armor, "Strength" (applying the 10-foot reduction to every Speed, floor 0) | 92 | GL-009b | movement-lifting-and-flight.md | FR-060 | exists |
| GL-A16 | Armor, "Stealth" (Disadvantage on Dexterity (Stealth) checks) | 92 | GL-009b | equipment-rules.md | FR-079 | exists |
| GL-A17 | Monsters, "Attack Notation" | 256 | GL-011 | monster-stat-block-actions.md | FR-022 to FR-025, FR-053 | exists |
| GL-A18 | Monsters, "Saving Throw Effect Notation" | 256 | GL-123a | monster-stat-block-actions.md | FR-032 to FR-043, FR-054 to FR-056 | exists |
| GL-A19 | Monsters, "Damage Notation" | 257 | GL-044 | monster-stat-block-actions.md | FR-026 to FR-028 | exists |
| GL-A20 | Monsters, "Multiattack" | 257 | GL-011 | monster-stat-block-actions.md | FR-010 to FR-021 | exists |
| GL-A21 | Monsters, "Bonus Actions" | 257 | GL-018 | monster-stat-block-actions.md | FR-003 | exists |
| GL-A22 | Monsters, "Reactions" | 257 | GL-116 | monster-stat-block-actions.md | FR-004 | exists |
| GL-A23 | Playing the Game, "Playing on a Grid": Squares, Speed, Entering a Square, Corners (optional rule) | 13 | GL-054 | grid-positions-and-movement.md | FR-020 to FR-026, FR-028 to FR-032 | exists |
| GL-A24 | Playing the Game, "Ranges" on a grid | 13 | GL-115 | grid-positions-and-movement.md | FR-010 to FR-016 | exists |
| GL-A25 | Playing the Game, Creature Size and Space table, the footprint on a grid (the table's facts stay GL-A11) | 14 | GL-128b | grid-positions-and-movement.md | FR-004 to FR-006 | exists |
| GL-A26 | Monsters, "Spellcasting": a spell of level 1+ is cast at its lowest possible level unless noted, and never higher; printed spell save DC and spell attack bonus | 257 | GL-131, GL-132 | monster-spellcasting.md | FR-003 to FR-012 | exists |
| GL-A27 | Monsters, "Spellcasting": a spell's special restrictions ("self only") | 257 | GL-131 | monster-spellcasting.md | FR-013 to FR-015 | exists |
| GL-A28 | Monsters, "Spellcasting", "Spell Components": waived components, Verbal and Somatic signalling, Material components always held | 257 | GL-133 | monster-spellcasting.md | FR-016 to FR-019, FR-039 | exists |
| GL-A29 | Monsters, "Spellcasting", "Casting Times of 1+ Minutes": Magic action each turn and Concentration | 257 | GL-031a, GL-098 | monster-spellcasting.md | FR-022 to FR-028 | exists |
| GL-A30 | Equipment, "Tools": Tool Proficiency on a check, the Utilize entries and their DCs | 93 | GL-113, GL-152 | tool-proficiency-and-crafting.md | FR-001 to FR-021 | exists |
| GL-A31 | Equipment, "Crafting Nonmagical Items" and scribing a Spell Scroll: tool, workers, days, prepared spell | 103 | GL-113 | tool-proficiency-and-crafting.md | FR-023 to FR-043 | exists |
| GL-A32 | Adventuring Gear, thrown Acid, Alchemist's Fire, Holy Water and Oil on a creature or object (replaces one attack) | 94-98 | GL-152 | item-use-effects.md | FR-001 to FR-019 | exists |
| GL-A33 | Adventuring Gear, Basic Poison, Antitoxin, Potion of Healing and Healer's Kit | 94-98 | GL-152 | item-use-effects.md | FR-020 to FR-026, FR-030 to FR-034, FR-040 to FR-045, FR-050 to FR-054 | exists |
| GL-A34 | Adventuring Gear, set-out gear: Caltrops, Ball Bearings, Oil on a 5-foot square, Hunting Trap (area entry once per turn) | 96-98 | GL-152 | deployed-and-restraining-gear.md | FR-004 to FR-025 | exists |
| GL-A35 | Adventuring Gear, holds: Net, Chain, Manacles, Rope (Restrained, escape, burst, unlock) | 96-98 | GL-119 | deployed-and-restraining-gear.md | FR-026 to FR-058 | exists |
| GL-A36 | Magic Items, "Spell Scroll": reading, the level check, the cast and the scroll afterwards | 204 | GL-131 | spell-scroll-use.md | FR-003 to FR-022 | exists |
| GL-A37 | Spells, "Combining Spell Effects" | 106 | GL-127 | active-effects.md | FR-013 to FR-019 | exists |
| GL-A38 | Monsters, "Hit Points": average or rolled, never both | 255 | GL-081 | monster-combatant-setup.md | FR-007 to FR-014 | exists |
| GL-A39 | Monsters, "Initiative": the modifier or the printed score | 254 | GL-090 | monster-combatant-setup.md | FR-025 to FR-028 | exists |
| GL-A40 | Monsters, "Proficiency Bonus", saving throws and skills | 255-256 | GL-113, GL-123b | monster-combatant-setup.md | FR-029 to FR-033 | exists |
| GL-A41 | Monsters, "Traits": recurring traits as typed facts | 256 | GL-136 | monster-traits.md | FR-001 to FR-008, FR-010 to FR-014, FR-020 to FR-022, FR-030 to FR-037, FR-040 to FR-047, FR-050 to FR-052, FR-060 to FR-062, FR-070 to FR-078, FR-090 to FR-095 | exists |
| GL-A42 | Monsters, stat-block Swallow, Engulf and Whelm actions | 263 | GL-071, GL-136 | monster-engulf-and-swallow.md | FR-021 to FR-063, FR-078 to FR-080 | exists |
| GL-A43 | Playing the Game, "Mounted Combat" | 15-16 | GL-090 | mounted-and-underwater-combat.md | FR-001 to FR-006, FR-008 to FR-023, FR-027 to FR-038 | exists |

Narrative rows (GL-A10, GL-A13, GL-A14) record SRD text with **no requirement**
(FR-031): Rhythm of Play (p. 5) is the shape of a session, not an engine rule;
Exceptions Supersede General Rules (p. 5) has no precedence function, since each
override is a requirement of the spec that owns it; Running a Monster (p. 255)
is play advice (`monster-limited-use.md` `AvailabilityOf` gives a host the
facts).

### 4.5 Planned owners

A planned owner is a spec the table names that is not yet in `doc/specs/`
(FR-007). **There are none:** every row is `exists` (Q-1). A new one is added
here as "target file, assumed scope, draft file name, entries it owns".
`content-magic-items.md`, `level-advancement-and-multiclassing.md`,
`feats-behaviour.md`, `weapon-properties-and-mastery.md`, `exploration.md`,
`combat-action-resolution.md`, `content-classes.md` and the other `toolbox-*.md`
specs own no glossary unit (Q-9, settled).

### 4.6 Totals

The test (FR-021) recomputes these and fails if the table disagrees.

| Measure | Count |
|---|---|
| Entries (GL-001 to GL-155) | 155 |
| Units (rows of 4.3 excluding GL-C01) | 183 |
| Units with state `exists` | 173 |
| Units with state `planned` | 0 |
| Units with state `none` | 10 |
| Units with state `gap` | 0 |
| Entries split into more than one unit | 25 |

### 4.7 Verification

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | The coverage fixture `internal/archtest/testdata/srd/rules_glossary_entries.json` MUST list, in printed order, every entry heading, its tag and its first and last printed page (155 records). It MUST hold no rule text. It is under a `testdata/srd/` directory, which carries the attribution pointer of `content-licensing-and-notice.md` FR-027 and is not a `NOTICE.md`-covered path (`shared-types-and-ownership.md` FR-033). | SRD 5.2.1, "Rules Glossary", pp. 176-191 | happy |
| FR-019 | The check, a function in package `internal/archtest` (`rules_glossary_coverage.go`) with `rules_glossary_coverage_test.go` and `rules_glossary_coverage_selftest_test.go` beside it (as `architecture-guardrails.md` section 5 lays out checks; stdlib only, no new dependency), MUST fail the build when any of FR-001 to FR-018 is broken, reporting each problem as the `Violation` of `architecture-guardrails.md` section 3 (check id, path, line, detail) and defining no second Violation shape: check id is the spec-qualified FR id of this spec that is broken (`rules-glossary-coverage.md#FR-013`), so it cannot collide with a guardrails check id; path is `doc/specs/rules-glossary-coverage.md`, line is the row of the table, detail is the unit id and a short reason. Problems: a missing, extra, renamed or reordered entry; a wrong tag or page; a unit with no owner or two owners; an owner that is neither a file in `doc/specs/` nor registered in section 4.5; a cited FR id that is absent or withdrawn; a `gap` without an open question; a tag family not owned consistently; a fixture record of the wrong shape or a page outside 176..191 (FR-018). | `doc/specs/architecture-guardrails.md` section 3 (Violation), FR-004, FR-011; `doc/rules/testing.md` section 9 | unhappy (state error) |
| FR-020 | The check MUST have a self-test that runs it on fixture tables, each with one injected fault, and asserts the exact Violation (check id and detail): a duplicate owner on one unit, a page of 175, a page of 192, a missing entry, an extra entry, an out-of-order row, an unregistered planned owner, a withdrawn FR id, an unknown file name, a `gap` with no open question. A clean fixture MUST yield zero Violations. The self-test is this spec's own duty: its check ids (`rules-glossary-coverage.md#FR-nnn`) are not registered in the registry of `architecture-guardrails.md` FR-005 (that spec declines it, section 9 there; Q-13, Q-17). | `doc/specs/architecture-guardrails.md` FR-005 (self-tests with fixtures) | unhappy (invalid input) |
| FR-021 | The totals of section 4.6 MUST equal the counts the check computes from the table, per state, and the count of split entries. A mismatch is a Violation. | n/a (package design) | unhappy (boundary) |
| FR-022 | For each unit with state `exists`, the check SHOULD look for the entry name or its printed page in the owner's `## 2. Source` section. A miss is a Warning: a line printed in the test output, not a Violation and not a check of the guardrails registry. It does not fail the build until promoted (Q-8). | n/a (package design) | unhappy (missing content) |
| FR-023 | A missing `doc/specs/rules-glossary-coverage.md`, a missing coverage fixture or a table row that does not parse MUST be reported as a Violation naming the file and row. An owner file that cannot be read is already one Violation per file under `architecture-guardrails.md` FR-010 and is not handled again here; the units that name it are not checked further. None of these MAY panic or abort the run: each is returned as a Violation. | `doc/rules/architecture.md` section 6 (expected failures are values); `doc/specs/architecture-guardrails.md` FR-010 | unhappy (failure) |
| FR-024 | An empty table MUST report 155 missing entries and no other Violation per entry beyond "missing". A fixture of one entry and one row MUST pass. | `doc/rules/testing.md` section 4 (empty and minimal cases) | unhappy (empty and minimal) |
| FR-025 | The check MUST be deterministic and read-only: it MUST read files only, MUST not use the network, MUST not write, and its Violations MUST be ordered as `architecture-guardrails.md` FR-011 orders them (path, then line, then the spec-qualified check id), so the result does not depend on directory listing order. | `doc/rules/testing.md` section 6 (deterministic, independent) | unhappy (determinism) |
| FR-026 | A change that adds, renames, deletes or withdraws a spec in `doc/specs/`, or withdraws an FR id the table cites, MUST update the table in the same change. This is enforced by FR-019 for existing owners and checked in review for planned ones. | n/a (package design) | unhappy (state error) |
| FR-027 | Before this spec is accepted, and in the same change as any SRD version change (`AGENTS.md`: "A new SRD version is a deliberate change"), a person MUST re-read pp. 176-191 against the fixture and state the entry count and the date in the pull request. The fixture and the table change together with `doc/research/sources.md`. | SRD 5.2.1, "Rules Glossary", pp. 176-191 | unhappy (state error) |
| FR-028 | The check MUST scan the `Out of scope` list of section 1 of each owner file named by a unit with state `exists` for the unit's entry name, matched only as a quoted heading (straight or curly double quotes, for example "Hit Point Dice") or as the name followed by its glossary page in a parenthesis (for example "Hit Point Dice, p. 183"); every other mention, such as the bare word action or line, is ignored. A unit whose entry name matches there is a Violation, unless a question of section 9 names the unit id (the disagreement is recorded, FR-010). | `doc/specs/shared-types-and-ownership.md`, section 1 (one owner per type); FR-010 | unhappy (rule interaction) |
| FR-029 | An entry that points to a rule in the Gameplay Toolbox (pp. 192-203, the `toolbox` context, `shared-types-and-ownership.md` FR-045) MUST keep its glossary owner for the glossary text, and the Toolbox rule MUST appear as an adjunct row (section 4.4) whose Owner is a `toolbox-*.md` spec. A unit MUST NOT be owned by a `toolbox-*.md` spec unless the glossary text itself defers to it (GL-112b). | SRD 5.2.1, "Curses", p. 179; "Possession", p. 186; "Environmental Curses", p. 193 | unhappy (rule interaction) |
| FR-030 | Where two existing specs cover one unit, the owner MUST be the topic-specific spec and the broader spec MUST have withdrawn its duplicate requirements to it (`shared-types-and-ownership.md` FR-048). A row whose Where cell cites a requirement that is withdrawn (FR-008) is the failing case: the table is changed to the new owner in the same change. | `doc/specs/shared-types-and-ownership.md` FR-048 | unhappy (overlap) |
| FR-031 | An adjunct row (section 4.4) MAY have owner `none` and Where `narrative`, with a related entry or `none`, for SRD text that guides the Game Master or players on how to play and states no rule an engine could apply ("Rhythm of Play", the sidebar "Exceptions Supersede General Rules", "Running a Monster"). Such a row MUST NOT cite a requirement id, and a note under the table MUST say why no requirement exists. A glossary unit of section 4.3 MUST NOT use `narrative`; it is owned, or `none` for `vocabulary` or `notation` (FR-006). A rule that merely sits in a narrative-looking sidebar (the Heroic Inspiration sidebar, GL-A02) is not narrative. | SRD 5.2.1, "Rhythm of Play", p. 5; "Exceptions Supersede General Rules", p. 5; "Running a Monster", p. 255 | unhappy (rule interaction) |
| FR-032 | A rule stated outside pp. 176-191 that no spec owns (equipment or item use p. 98, for example) MUST be recorded as an open question of section 9 (Q-22, Q-23), not as an adjunct row with an invented owner and not as a `gap` unit. It is outside the 155 entries, so it is not counted in 4.6 and does not block a release (FR-009). The question is closed, and an adjunct row added, in the change that adds the owning spec. | SRD 5.2.1, "Oil", p. 98; n/a (package design) | unhappy (state error) |

The check reads Markdown tables the way `architecture-guardrails.md` FR-071 and
FR-094 read specs. Rows of 4.3 start with `GL-`, so they cannot be mistaken for
an FR row.

### 4.8 Withdrawn ids

| Id | Withdrawn |
|---|---|
| none | This spec has moved no requirement. |

## 5. Model

No domain types, and no code in any library package. The data shapes are the table and the fixture.

- **Coverage row** (4.3): `id` (`GL-` plus three digits or `GL-C01`, plus an
  optional lower-case letter), `entry`, `tag` (one of five or empty), `pages`
  (`n` or `n-m`, in 176..191), `aspect`, `owner` (file name or `none`), `where`,
  `state` (`exists`, `planned`, `none`, `gap`).
- **Adjunct row** (4.4): the same with a page outside 176..191 and a related
  entry (or `none` for a narrative row).
- **Fixture record**: `{ "entry": string, "tag": string or "", "firstPage": int,
  "lastPage": int }` (decoded with `encoding/json` into a Go struct with the same four fields, `DisallowUnknownFields`), in printed order, 155 records.
- **Invariants:** FR-001 to FR-017, enforced after the fact; no constructor, so
  no `srderr.ErrInvalidInput`. **Failures** are `Violation`s (FR-019), returned as a slice, never panics or errors that stop the run
  (FR-023); I/O errors reading a file become a Violation naming the file; the only other output is the Warning of FR-022.

## 6. Interface

No command, use case, port, event, result or export; every library package is
unchanged. The only interface is the check function in
`internal/archtest/rules_glossary_coverage.go` (unexported outside the module root tree, since `internal/` is never imported elsewhere) with its test and
self-test, and support code in the same package reusing the spec
reader of `architecture-guardrails.md` (FR-071, FR-094). Its spec-qualified check
ids stay spec-qualified and are not registered with that spec (Q-13, Q-17). No port, so no double or
contract suite.

## 7. Observability

Silent by design: documentation and a test, no use case, adapter or runtime path
(`doc/rules/observability.md` section 3). This spec owns no signal and adds no
catalogue entry; `doc/observability/signals.md` is unchanged.

1. **What can go wrong?** The table drifts from the PDF, an owner is renamed,
   deleted or has a requirement withdrawn, a planned owner is written but its
   rows stay `planned`, two specs claim one unit, or a unit has no owner.
2. **How would an operator know?** The check of section 8 fails the build with one
   Violation per unit; the Violation text is the signal.
3. **What to diagnose it?** Unit id, FR id broken, owner file, cited id (all in
   the Violation).
4. **Counted or timed?** Nothing at runtime; the totals of 4.6 are the standing
   measure.
5. **Existing instrumentation correct?** Yes, unchanged.
6. **What should a consumer monitor?** Nothing. A contributor watches the `gap`
   count of 4.6 (0 now; any gap needs an open question).
7. **Anything noisy?** The FR-022 Warnings, if many owners do not cite an entry
   in their Source section; Q-8 decides whether to promote them.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 | architecture | Real table against the fixture: 155 entries, GL-001 to GL-155, none extra or missing. Self-test: a dropped entry, an added entry. |
| FR-002 | architecture | Name and tag equal the fixture's. Self-test: a renamed entry, a wrong tag, a tag on an untagged entry. |
| FR-003 | architecture | Pages 176 and 191 accepted; 175 and 192 rejected; a range whose last page is before its first; the six continuing entries as a constant list in the test, compared with the fixture's entries whose pages differ. |
| FR-004, FR-005, FR-006 | architecture | One unit with two file names (Violation); an entry split into `a` and `b` with distinct owners passes. Owner existing, registered planned, `none`, an unknown file name (Violation), an empty cell (Violation), a `gap` with an interim owner. `none` with `vocabulary` or `notation` passes; another reason or blank is a Violation. |
| FR-007, FR-008, FR-009 | architecture | A planned owner not in 4.5; a planned owner whose file now exists (state must be `exists`). Real owners: every cited FR id exists and is live; a missing id and a withdrawn id fail; `scope` accepted for an existing owner and rejected for a planned one. `gap` with an existing Q id passes, a missing Q id and a `gap` with no Where fail. The release gate is covered by `release-0-1-0.md` FR-097. |
| FR-010, FR-026 | architecture | Mechanical part is FR-028; rename, deletion and withdrawal cases each have a self-test (with FR-008, FR-019). Review items: every conflict question (Q-3, Q-5, Q-6, Q-7, Q-14) is listed in section 9 as settled; planned owners. |
| FR-011, FR-012 | architecture | Two rows swapped; units of one entry non-adjacent; `b` before `a`. A cell of 201 characters fails, 200 passes; a Where cell with a full sentence fails. |
| FR-013 to FR-017 | architecture | Counts from the fixture against constant name lists held in the test: 12 Action (self-test: missing Study), 15 Condition (each owner `conditions.md`; one condition owned elsewhere), seven Area of Effect rows sharing `spell-effect-data.md` (GL-008b may differ; Sphere owned elsewhere), six Hazard rows (Falling owned elsewhere), four Attitude rows (Hostile owned elsewhere). |
| FR-018 | architecture | Fixture shape and count; no field beyond the four; a record with page 192 (Violation `rules-glossary-coverage.md#FR-018`). Written from the PDF by hand (FR-027). |
| FR-019, FR-020 | architecture | The real run: zero Violations. Each class of problem has a self-test fixture; each Violation carries the spec-qualified check id, this file's path, the row line and the unit id. The ten injected faults give exactly one Violation each with the exact check id and unit id; a clean fixture gives none. |
| FR-021, FR-024, FR-025 | architecture | Totals per state and the split count recomputed; a total off by one fails. Empty table: 155 "missing" Violations; a one-entry fixture passes. Run twice with a shuffled file listing: identical output ordered by path, line, check id; read-only fixture directory shows no write and no network. |
| FR-022, FR-023 | architecture | An owner whose Source names the entry (no Warning), a page only (no Warning), neither (one Warning, build passes). An absent spec file, an absent fixture, a malformed row: each a Violation, none a panic; an unreadable owner gives only the one Violation of `architecture-guardrails.md` FR-010. |
| FR-027 | not automated | A process step recorded in the pull request (entry count and date); needs the PDF, which is not stored in the repository (`doc/research/sources.md`). |
| FR-032 | architecture | A section 9 question for an unowned outside-glossary rule passes; an adjunct row owned by `none` with Where other than `narrative` fails (FR-006); no such question is counted as a `gap`. |
| FR-028 to FR-031 | architecture | An owner whose `Out of scope` names the entry as a quoted heading fails, passes with a section 9 question naming the unit; a bare common word such as action or line is not a hit. A `none` row with another reason is a Violation (FR-006). A unit owned by a `toolbox-*.md` spec without the Demonic Possession exception fails; an adjunct row owned by one passes. A row citing an id the broader spec moved fails through FR-008, passes once re-pointed (real run covers Grappling, Armor Class, Bloodied). An adjunct row `none`/`narrative` with no id passes; with an FR id fails; a 4.3 unit with `narrative` fails; the real run covers GL-A10, GL-A13, GL-A14. |

- **Levels:** architecture only, with a self-test per `architecture-guardrails.md`
  FR-005; no library code, so no unit, application, contract or scenario test. Tests are table-driven with `t.Run`, use `testing/fstest` or `t.TempDir` fixtures for the self-tests, and run with `go test -race -shuffle=on`.
- **Mutation testing:** not applicable (`doc/rules/testing.md` section 8 scopes it
  to the domain files of the context packages; `gomu` is in any case run only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`); the self-test fixtures play that role for
  FR-001 to FR-017.
- **Not covered:** FR-027 (needs the SRD PDF).

## 9. Open questions and decisions

Open: none; no glossary unit is a `gap`. Q-23 and Q-24 are decided below (nothing escalated).

- Q-23 (outside-glossary gear still unowned, SRD 5.2.1, "Adventuring Gear", p. 98): Grappling Hook, Climber's Kit and Lock and Thieves' Tools on a Lock. Decided: no owner; the host narrates them. When a sibling spec takes one, add an adjunct row in the same change.
- Q-24 (Armor "Strength" on a Speed, `equipment-rules.md` Q20), decided: GL-A15 is `equipment-rules.md` FR-007 to FR-010 (the rule and the 10-foot output) and GL-A15b is `movement-lifting-and-flight.md` FR-060 (`ArmorSpeedRules.apply`, via `MovementSpeeds.adjustedBy`, SRD 5.2.1, "Changes to Your Speeds", p. 188); the host only calls it.

Settled, one line each:

- Q-2 (`magical-effects.md` owns 11 units after Q-6): stays one spec while it is general stateless rules; split only if it outgrows one reviewable PR, and a split changes only Owner cells (design basis: `doc/rules/architecture.md` size rules, FR-062 of `shared-types-and-ownership.md`).
- Q-5 (Curses, GL-040): `magical-effects.md` owns the general rule and how a curse ends (FR-027 to FR-033); the potency benchmark is `toolbox-curses-and-contagions.md` (GL-A06); `magic-item-rules.md` owns the item-specific state (`CurseBreak`, FR-057, FR-115) and cites by id (SRD 5.2.1, "Curses", p. 179).
- Q-6 (Readying a spell, GL-117b): owned by `readying-a-spell.md` FR-001 to FR-012, split from `magical-effects.md` whose FR-130 to FR-139 are withdrawn; the row cited those withdrawn ids and now cites the new owner. Settled as an interpretation (the SRD's Ready text, p. 187, covers one readied spell, which is a distinct subject from the action economy), pending RA-1: the Concentration subject is `readying-a-spell.md` RA-1 to `spellcasting.md` (until answered, `spellcasting.md` FR-118 and `casting-and-concentration.md` FR-119, FR-120).
- Q-7 (`ability-scores.md` cites "Breaking Objects", "No Ability Scores", p. 178): the table gives it to `environment-hazards-and-objects.md`; `ability-scores.md` keeps the citation as context only, and FR-022 does not treat a citation as ownership.
- Q-9 (specs owning no unit, "Bright Light", "Dim Light", "Darkness" pointing to "Exploration", pp. 178, 180, 181): a rule stays with its current owner; when a new spec's scope states one, the row changes in that same change (FR-026). Not an open item.
- Q-11 (Hit Point Dice for monsters): the SRD gives a monster Hit Point Dice as stat-block data (`content-monsters.md` FR-015), and spending them is a character rest rule; GL-080b stays with `rests-and-hit-dice.md` (its Q-8 excludes monster rests), no row is added.
- Q-1 (eight planned owners now exist): `shared-types-and-ownership.md` FR-052.
- Q-3 (Armor Training Disadvantage): `shared-types-and-ownership.md` FR-060.
- Q-4 (grapple ending): `shared-types-and-ownership.md` FR-029, FR-048.
- Q-8 (FR-022 starts as a Warning): promote to a failure in the change that sets
  this spec to `accepted`, and no later than 2026-12-31.
- Q-10 (Page column is glossary pages only; sidebars are adjunct rows), Q-12
  (Abbreviations table is `none`, FR-006), Q-21 (narrative rows, FR-031).
- Q-13, Q-17 (declined by `architecture-guardrails.md` section 9; ids stay spec-qualified so bare ids cannot collide with guardrails FR-013 to FR-030): test in `internal/archtest/`, check ids for FR-001
  to FR-018, FR-021, FR-023, FR-024, FR-028 to FR-031).
- Q-14 (six shapes with `spell-effect-data.md`; origin and unseen origin with
  `magical-effects.md`): a `content` to `combat` edge is forbidden by
  `doc/rules/architecture.md` section 2.
- Q-15 (space filled by objects not modelled): `environment-hazards-and-objects.md` Q-23.
- Q-16 (Creature Size and Space table): `shared-types-and-ownership.md` FR-053, FR-060; space facts are `content-schema-and-decoder.md` FR-139.
- Q-19 (stale "planned" text in `mounted-and-underwater-combat.md` and `heroic-inspiration.md`): removed by the owners.
- Q-18 (the Grapple DC, saves, size and free-hand rules are live only in `grappling.md` FR-094 to FR-096; no duplicate remains in `attacks.md`).
- Q-22 (Oil on a space, thrown gear, Basic Poison, Antitoxin): closed by `item-use-effects.md` and `deployed-and-restraining-gear.md` (GL-A32 to GL-A35). The Net is in both: `item-use-effects.md` FR-003, FR-014 give the thrown save, `deployed-and-restraining-gear.md` FR-035 to FR-044 the hold; GL-A35 names the hold, and the overlap is for those two specs to settle.
- Q-20 (Lair actions are not in the PDF; `monster-limited-use.md` models only the
  `LairStatus` input).

## 10. Changelog impact

No user-visible entry: adds a spec and a test, changes nothing in any
library package, its exports or any signal. Not **BREAKING**. A one-line `Added` entry is optional
(`doc/rules/architecture.md` section 7 asks only for user-visible changes).
