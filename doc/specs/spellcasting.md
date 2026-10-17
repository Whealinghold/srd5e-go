# Spellcasting

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `errors-and-results.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `observability-ports.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `content-spells.md`, `derived-stats.md`, `equipment-rules.md`, `character-aggregate.md`, `rests-and-hit-dice.md` |

Dependency notes (prose only; FR-056): `character-aggregate.md` gives `character.Event` and `CharacterID` only; `content-schema-and-decoder.md` the typed ids and `MissingContent`; `equipment-rules.md` (its FR-075) the Armor Training result from which the caller sets `UntrainedArmorWorn`. No `combat` spec is listed (shared-types-and-ownership.md FR-013 to FR-016). `resolve-d20-test-use-case.md` is cited in prose only: this context never rolls. No port or `content.Source` is used. The casting rules and Concentration transitions are `casting-and-concentration.md` (rank 14), which depends on this spec and is not listed here.

## 1. Purpose and scope

Models how a character with the Spellcasting or Pact Magic class feature gains and prepares spells and keeps its slots: the spellcasting ability, spell save DC and spell attack bonus, the Spell Slots of each class (and of a multiclass character), the cantrips and Prepared Spells each class level allows, how those lists change, the standalone slot operations, and the `SpellcastingState` the caller holds beside the `Character` aggregate (not inside it; section 9). It also owns the *shapes* of the state's `Concentration` (FR-118), `CastingInProgress`, `CastSource` and `SpellSlotChoice`. The rules that cast a spell (components, casting time, Rituals, casting in armor, feature-granted casts) and the Concentration transitions and save DC after damage (shared-types-and-ownership.md FR-011) are `casting-and-concentration.md`; every id that moved there keeps its number (section 4, "Withdrawn ids"). A bare `FR-nnn` is defined in exactly one of the two specs; a withdrawn row says which.

State in, new state and events out (`doc/rules/architecture.md` §4). Reused, not redefined (architecture.md §2; shared-types-and-ownership.md FR-001 to FR-012): `Ability`, `AbilityScore`, `AbilityModifier`, `Modifier` (ability-scores.md; modifiers-and-advantage.md FR-001, FR-043); `ClassLevels`, `CharacterLevel`, `ProficiencyBonus` (derived-stats.md FR-058, FR-059); `DifficultyClass` (d20-tests.md FR-021), `RollModifiers` (d20-tests.md section 4.3); `ClassID`, `SpellID`, `FeatureID` (content-schema-and-decoder.md FR-128); `MissingContent` (content-schema-and-decoder.md FR-147); `character.Event`, `CharacterID` (character-aggregate.md FR-085); `RestKind` (rests-and-hit-dice.md FR-001); from content-spells.md `Spell`, `SpellLevel`, `CastingTimeMode`.

**Out of scope:**

- The spells themselves (lists, descriptions, effects, "Combining Spell Effects", "Invalid Targets", "Awareness of Being Targeted", SRD 5.2.1, p. 106): the published `Spell` (content-spells.md).
- Class features other than Spellcasting and Pact Magic (Magical Cunning, Arcane Recovery, Font of Magic, Sorcerous Restoration, Metamagic, Innate Sorcery, Mystic Arcanum, Magical Secrets, Spell Mastery, Signature Spells, Favored Enemy, Eldritch Invocation spells, Contact Patron, subclass spell grants, Draconic Spells). A feature granting an always-prepared spell, or extra or granted cantrips (Divine Order: Thaumaturge, Primal Order: Magician, Blessed Warrior, Druidic Warrior), enters only as an input (FR-045, FR-046, FR-096 to FR-098). Ritual Adept and Memorize Spell: FR-091, FR-114.
- Casting through Spell Scrolls (`spell-scroll-use.md`; only the `CastSource` shape, FR-148) and magic items (`magic-item-rules.md`; the lowest-level reading is FR-147) ("Casting without Slots", p. 104); spellbook management (p. 78; the spellbook is an input set, FR-050); `RestKind`; any `combat` type; monster Spellcasting traits; Spellcasting Services prices (p. 102); "Identifying an Ongoing Spell" (p. 106); level-up as a whole (only the spellcasting changes, FR-029, FR-047, FR-111, FR-112).
- Optional rules: the SRD names none for spellcasting, so no `RuleOptions`. Barbarian, Fighter, Monk and Rogue have no Spellcasting feature in SRD 5.2.1 (checked in the class sections, pp. 28-82); nothing here applies to them.

## 2. Source

PDF page numbers equal printed page numbers in SRD 5.2.1 (page 104 is printed 104).

- SRD 5.2.1, "Spells": "Gaining Spells", "Preparing Spells", "Always-Prepared Spells", "Spell Level", "Spell Slots", p. 104; "Saving Throws" (spell save DC), "Attack Rolls" (spell attack modifier), p. 106. The rest of "Spells" is `casting-and-concentration.md`.
- SRD 5.2.1, "Character Creation", "Create Your Character", "Spellcasting", pp. 22-23; "Level Advancement", Character Advancement table, p. 23; "Multiclassing" ("Prerequisites", "Proficiency Bonus", "Spellcasting": Spells Prepared, Cantrips, Spell Slots, Pact Magic), pp. 24-26, and the Multiclass Spellcaster table, p. 26.
- SRD 5.2.1, "Classes": Spellcasting or Pact Magic and Features table of Bard (pp. 31-32), Cleric (pp. 36-37), Druid (pp. 41-42), Paladin (pp. 53-54), Ranger (pp. 57-58), Sorcerer (pp. 64-65), Warlock (pp. 71-72), Wizard (pp. 77-78); Wizard "Memorize Spell" (p. 79); Cleric "Divine Order" (p. 37), Druid "Primal Order" (p. 42), Paladin "Blessed Warrior" (p. 54), Ranger "Druidic Warrior" (p. 59); Warlock "Pact of the Blade" and "Pact of the Tome" (Spellcasting Focus), p. 74 (FR-068, FR-129).
- SRD 5.2.1, "Rules Glossary": "Armor Training", p. 177; "Cantrip", p. 178; "Concentration", p. 179 (casting spec).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Spellcasting / Pact Magic (class feature) | `FeatureSpellcasting` / `FeaturePactMagic` (typed enum `SpellcastingFeature`) | Pact Magic: Warlock only |
| spellcasting ability, modifier | `Ability`, `AbilityModifier` (kernel package `ability`) | Fixed per class (table A); modifier supplied by the caller |
| spell save DC | `SpellSaveDC` (wraps a `d20.DifficultyClass`) | 8 + spellcasting ability modifier + Proficiency Bonus (`ProficiencyBonus`, derived-stats.md) |
| spell attack bonus ("spell attack modifier") | `SpellAttackBonus` | Modifier + Proficiency Bonus; also `RollModifiers` (FR-116) |
| Spell Level, Cantrip | `SpellLevel` 0-9 (content-spells.md), `IsCantrip()` | Level 0 is cast without a slot |
| Spell Slots, Pact Magic slots | `SpellSlots`, `PactMagicSlots` | Per level 1-9 max and remaining; Pact Magic: a count, all one level |
| Prepared Spells, Always-Prepared | `PreparedSpells`, `PreparedSpells.AlwaysPrepared` | Always-prepared do not count against the limit |
| Extra and granted cantrips | `ClassSpellcasting.ExtraCantrips`, `GrantedCantrip` | Inputs from class features (FR-096 to FR-098) |
| Spell Preparation by Class | table I | Trigger and number of *replacements*; additions are FR-089 |
| Long Rest, Short Rest | `RestKind` (rests-and-hit-dice.md) | Reported by the caller; also the `TriggerShortRest` replacement trigger (FR-091) |
| Spellcasting Focus | `SpellcastingFocusKind` | Table A; features may allow another kind (FR-068, FR-129) |
| Spellbook | `Spellbook` (set of `SpellID`) | Wizard only (FR-050) |
| Armor Training | caller input `UntrainedArmorWorn` | From equipment-rules.md FR-075; used by the casting spec |
| Spell Slot level (Warlock "Slot Level") | `SlotLevel` | 1-9 (FR-024, FR-107) |
| Multiclass Spellcaster table | table C (FR-039 checks it against an independent fixture) | Spell Slots per Spell Level by total level |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3). The SRD's "level 1+ spells" and "spells of level 1 or higher" are written here as "level 1+ spells".

## 4. Requirements

Stable ids; never renumber. Source cites are by section and printed page. "The class" means one of the eight classes of FR-001. Data tables A to T follow the table (labels kept; C, D, E and G share one table; K, L, N, S and T are in the casting spec). Result shorthand (`NoSlotAvailable`, `AdvanceRejected(...)`, the PascalCase rejection kinds) maps to exported names in section 5 ("Naming of results"); requirement text is not renamed (shared-types-and-ownership.md FR-017, FR-018). New requirements of the pair take ids from FR-141 on, so a number is never in both specs.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | The model MUST represent exactly eight spellcasting classes: Bard, Cleric, Druid, Paladin, Ranger, Sorcerer, Warlock and Wizard. Warlock uses Pact Magic; the other seven use Spellcasting. | SRD 5.2.1, "Classes" (Bard, Cleric, Druid, Paladin, Ranger, Sorcerer, Warlock, Wizard), pp. 31-78 | happy |
| FR-002 | Each class MUST have the spellcasting ability in table A (Charisma for Bard, Paladin, Sorcerer and Warlock; Wisdom for Cleric, Druid and Ranger; Intelligence for Wizard). | SRD 5.2.1, class "Spellcasting Ability" entries, pp. 32, 37, 42, 54, 58, 65, 72, 78 | happy |
| FR-003 | Spell save DC MUST equal 8 + spellcasting ability modifier + Proficiency Bonus (table B). | SRD 5.2.1, "Spells", "Saving Throws", p. 106; "Create Your Character", p. 23 | happy |
| FR-004 | Spell attack bonus MUST equal spellcasting ability modifier + Proficiency Bonus (table B). | SRD 5.2.1, "Spells", "Attack Rolls", p. 106; "Create Your Character", p. 23 | happy |
| FR-005 | The spellcasting ability modifier used for a spell MUST be that of the class whose feature gives the ability to cast the spell, so a multiclass character gets a DC and attack bonus per class. | SRD 5.2.1, "Create Your Character", p. 23; "Multiclassing", "Spellcasting" ("Each spell you prepare is associated with one of your classes"), p. 25 | happy |
| FR-006 | DC and attack bonus MUST be correct at the extremes: the lowest modifier (-5) with Proficiency Bonus +2 gives DC 5 and attack bonus -3; the highest (+10) with +6 gives DC 24 and attack bonus +16; a modifier of 0 gives DC 8 + Proficiency Bonus. | SRD 5.2.1, "Spells", p. 106; Ability Modifiers table (ability-scores.md) | unhappy (boundary) |
| FR-007 | The stats result MUST carry the ability, ability modifier and Proficiency Bonus used, not only the totals. | `doc/rules/architecture.md` §6; SRD 5.2.1, "Spells", p. 106 | happy |
| FR-008 | Class levels and character level are owned upstream by derived-stats.md (shared-types-and-ownership.md FR-008): `ClassLevels` (ordered; the first entry is the initial class) and `CharacterLevel` (1 to 20, built by `NewCharacterLevel`). This spec MUST reuse them and MUST NOT define its own class-level or character-level type. Every use case that takes `ClassLevels` MUST ignore a `ClassID` that is not one of the eight spellcasting classes for spellcasting purposes (it still counts toward `ClassLevels.TotalLevel()`). Order in `ClassLevels` has no effect on any spellcasting result. | SRD 5.2.1, Character Advancement table, p. 23; `doc/rules/architecture.md` §2 (the upstream context owns a shared concept) | happy |
| FR-009 | Cross-reference only: the 20-level total, the at-least-level-1 rule and any duplicate-class rule are invariants of `ClassLevels` and `CharacterLevel` (derived-stats.md) and of the `Character` aggregate (character-aggregate.md); they are package invariants of those types where the SRD is silent. This spec adds no second validation; table H rows are expressed as `ClassLevels` over `ClassID`. | SRD 5.2.1, "Level Advancement", Character Advancement table, p. 23; "Multiclassing", p. 24 (rule owned by derived-stats.md) | unhappy (invalid input) |
| FR-010 | Bard, Cleric, Druid, Sorcerer and Wizard MUST have the Spell Slots of table C at each class level 1-20 (single-class). | SRD 5.2.1, Bard Features p. 31, Cleric Features p. 36, Druid Features p. 41, Sorcerer Features p. 65, Wizard Features p. 77 | happy |
| FR-011 | Paladin and Ranger MUST have the Spell Slots of table D at each class level 1-20 (single-class). | SRD 5.2.1, Paladin Features p. 53, Ranger Features p. 58 | happy |
| FR-012 | Warlock MUST have the Pact Magic slot count and slot level of table E at each class level 1-20. | SRD 5.2.1, Warlock Features, p. 71 | happy |
| FR-013 | A spell level with no slots at the current level MUST read as zero maximum slots (the SRD's "-"), not as an error, for every level 1-9. | SRD 5.2.1, Features tables (the "-" cells), pp. 31, 36, 41, 53, 58, 65, 77 | unhappy (boundary) |
| FR-014 | Cantrips known MUST follow table F for Bard, Cleric, Druid, Sorcerer, Warlock and Wizard at each class level 1-20. | SRD 5.2.1, Features tables, pp. 31, 36, 41, 65, 71, 77 | happy |
| FR-015 | The number of prepared level 1+ spells MUST follow table G for each class and class level 1-20. | SRD 5.2.1, Features tables ("Prepared Spells" column), pp. 31, 36, 41, 53, 58, 65, 71, 77 | happy |
| FR-016 | A Warlock's maximum preparable spell level MUST equal the Slot Level of table E for the Warlock level (not the highest level 1-5 the Warlock has ever had slots for). | SRD 5.2.1, "Warlock", Pact Magic, "Prepared Spells of Level 1+", p. 71 | happy |
| FR-017 | For every other class, the maximum preparable spell level MUST be the highest spell level for which the class has slots in its own table (C or D) at its class level; this holds in a multiclass as if the class were single-classed. | SRD 5.2.1, "Bard", "Prepared Spells of Level 1+", p. 32 (and the same entry for each class); "Multiclassing", "Spells Prepared", p. 25 | happy |
| FR-018 | Paladin and Ranger MUST have zero cantrips known from Spellcasting at every level (their Features tables have no Cantrips column). Cantrips that a feature grants them (Blessed Warrior, Druidic Warrior) are separate inputs (FR-097) and do not change this count. | SRD 5.2.1, Paladin Features p. 53, Ranger Features p. 58 | unhappy (empty and minimal) |
| FR-019 | `SpellSlots` MUST hold, for each spell level 1-9, a maximum and a remaining count; `NewSpellSlots` MUST return an error (`errors.Is` `srderr.ErrInvalidInput`) if remaining is negative or exceeds maximum (`MustSpellSlots` panics with it). | SRD 5.2.1, "Spell Slots", p. 104; `doc/rules/architecture.md` §3 | unhappy (invalid input) |
| FR-020 | A newly derived `SpellSlots` MUST have remaining equal to maximum at every level. | SRD 5.2.1, "Spell Slots", p. 104 | happy |
| FR-021 | Expending a slot of level n MUST decrease the remaining count at level n by exactly one, leave all other levels unchanged, and emit a `SpellSlotExpended` event. | SRD 5.2.1, "Spell Slots" ("you expend a slot"), p. 104 | happy |
| FR-022 | Expending a slot at a level whose remaining count is zero MUST return a `NoSlotAvailable` result and leave the state unchanged. | SRD 5.2.1, "Spell Slots", p. 104 | unhappy (state error) |
| FR-023 | Expending a slot at a level whose maximum is zero (for example level 3 for a level 2 Wizard) MUST return `NoSlotAvailable`, the same as FR-022. | SRD 5.2.1, Features tables, pp. 31, 77 | unhappy (boundary) |
| FR-024 | A slot level outside 1-9 MUST be rejected by `NewSlotLevel` with an error that is `errors.Is` `ErrSlotLevelRange` (wrapping `srderr.ErrInvalidInput`; level 0 is a cantrip and never uses a slot). | SRD 5.2.1, "Spell Level", "Casting without Slots" ("A cantrip is cast without a spell slot"), p. 104 | unhappy (invalid input) |
| FR-025 | Finishing a Long Rest MUST restore every expended spell slot (both the Spellcasting pool and the Pact Magic pool) and emit a `SpellSlotsRestored` event with the number restored per pool. | SRD 5.2.1, "Spell Slots", p. 104; "Warlock", Pact Magic, p. 71 | happy |
| FR-026 | Finishing a Short Rest MUST restore expended Pact Magic slots only and MUST leave the Spellcasting pool unchanged. | SRD 5.2.1, "Warlock", Pact Magic, "Spell Slots", p. 71; Wizard "Spell Slots", p. 78 (Long Rest only) | unhappy (rule interaction) |
| FR-027 | Restoring when nothing is expended MUST return an equal state and no events. | SRD 5.2.1, "Spell Slots", p. 104 | unhappy (empty and minimal) |
| FR-028 | State transitions MUST NOT mutate their input, and every slice or map a handoff type exposes MUST be a copy (copy on construction and on read). | `doc/rules/architecture.md` §3, §4; `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-029 | When a class level increases, recomputed Spellcasting-pool maximums MUST come from the tables; remaining slots at a level MUST increase by the increase in maximum at that level (never above the new maximum) and never decrease. Pact Magic slots follow FR-111. This is a package decision where the SRD is silent (section 9). | Package design (section 9); SRD 5.2.1, Features tables | happy |
| FR-030 | A character with the Spellcasting feature from exactly one class MUST use that class's own slots (table C or D), even if they also have levels in a class without Spellcasting. | SRD 5.2.1, "Multiclassing", "Spellcasting" ("If you multiclass but have the Spellcasting feature from only one class, follow the rules for that class"), p. 25 | happy |
| FR-031 | A character with the Spellcasting feature from two or more classes MUST derive Spell Slots by looking up the *total spellcaster level* in the Multiclass Spellcaster table (table C). | SRD 5.2.1, "Multiclassing", "Spellcasting", "Spell Slots", pp. 25-26 | happy |
| FR-032 | Total spellcaster level MUST equal all levels in Bard, Cleric, Druid, Sorcerer and Wizard, plus half of the combined Paladin and Ranger levels rounded up (decision on the combined reading in section 9). | SRD 5.2.1, "Multiclassing", "Spell Slots" (bulleted list), p. 25; interpretation: Paladin and Ranger are one list item, so their levels are summed, halved once and rounded up (per-class rounding is the alternative; section 9, Table H4) | happy |
| FR-033 | Levels in classes without the Spellcasting feature and Warlock levels MUST NOT count toward total spellcaster level. | SRD 5.2.1, "Multiclassing", "Spell Slots" (list names only the five full and two half classes), p. 25 | unhappy (rule interaction) |
| FR-034 | Pact Magic slots MUST be a separate pool, tracked beside the Spellcasting pool, never merged into it, and a Warlock with another Spellcasting class MUST have both. | SRD 5.2.1, "Multiclassing", "Pact Magic", p. 26 | unhappy (rule interaction) |
| FR-035 | A character with Pact Magic and the Spellcasting feature MUST be able to spend a slot from either pool for a prepared spell of any of their classes. | SRD 5.2.1, "Multiclassing", "Pact Magic", p. 26 | happy |
| FR-036 | Prepared-spell limits, cantrip counts and maximum preparable levels MUST be computed per class from that class's level alone, as if single-classed. | SRD 5.2.1, "Multiclassing", "Spells Prepared", p. 25 | happy |
| FR-037 | Each prepared-list entry MUST belong to exactly one class, and casting it MUST use that class's ability (FR-005). The same spell id MAY appear in the prepared lists of two classes, as separate entries. | SRD 5.2.1, "Multiclassing", "Spells Prepared" ("Each spell you prepare is associated with one of your classes"), p. 25 | happy |
| FR-038 | The level used to scale a cantrip MUST be the total character level, not the class level, unless the spell says otherwise. A cantrip cast MUST report it as `ScalingLevel` in the `SpellCast` result (casting spec FR-082), taken from `ClassLevels.TotalLevel()` in the state (derived-stats.md FR-058; there is no `CharacterLevel` member). | SRD 5.2.1, "Multiclassing", "Cantrips", p. 25; "Acid Splash" Cantrip Upgrade, p. 107 | happy |
| FR-039 | The Multiclass Spellcaster table (p. 26) MUST agree with the single-class full-caster slots of table C for every total level 1-20. The test MUST compare the production tables with an independent fixture, transcribed separately from the Multiclass Spellcaster table on p. 26, and with each full-caster class's own Features table transcribed separately (not imported from the production code), so the check can fail. | SRD 5.2.1, Multiclass Spellcaster table, p. 26; Features tables | happy |
| FR-040 | A spell MUST be on the class's spell list (`Spell.Classes`, a set of `ClassID`, contains the casting class's `ClassID`; content-spells.md FR-016) to be prepared by that class, otherwise preparation is rejected with `NotOnClassList`. A granted cantrip (FR-097) is checked against the list its feature names, held as a `ClassID` on the `GrantedCantrip`, not against the casting class's own list. | SRD 5.2.1, "Bard", "Prepared Spells of Level 1+", p. 32 (and each class); "Class Spell Lists", p. 105; content-spells.md FR-016 | unhappy (invalid input) |
| FR-041 | A spell above the class's maximum preparable level (FR-016, FR-017) MUST be rejected with `SpellLevelTooHigh`; a spell exactly at the maximum MUST be accepted. | SRD 5.2.1, "Bard", p. 32 ("must be of a level for which you have spell slots"); "Warlock", p. 71 | unhappy (boundary) |
| FR-042 | A prepared list longer than the class's limit (table G) MUST be rejected with `TooManyPrepared`; a list exactly at the limit MUST be accepted. | SRD 5.2.1, Features tables, "Prepared Spells" column, pp. 31-77 | unhappy (boundary) |
| FR-043 | A spell listed twice in one prepared list MUST be rejected with `DuplicateSpell`. | Package design (`doc/rules/architecture.md` §6, contradictory data; section 9); SRD 5.2.1, "Preparing Spells" speaks of a list of spells, p. 104 | unhappy (invalid input) |
| FR-044 | A level 0 spell in a prepared list MUST be rejected with `CantripNotPreparable`; cantrips are known, not prepared. | SRD 5.2.1, "Preparing Spells" ("list of level 1+ spells you prepare"), p. 104 | unhappy (invalid input) |
| FR-045 | Always-prepared spells MUST count as prepared for casting and MUST NOT count against the class's prepared limit; they otherwise count as spells of that class. | SRD 5.2.1, "Always-Prepared Spells", p. 104; "Bard", p. 32 | happy |
| FR-046 | An always-prepared spell MUST NOT be removed or replaced by a change to the prepared list; an attempt MUST be rejected with `AlwaysPreparedNotReplaceable`. | SRD 5.2.1, "Always-Prepared Spells" ("you always have prepared"), p. 104 | unhappy (state error) |
| FR-047 | When a class level increases the prepared limit, `AdvanceSpellcasting` MUST report `SpellsToChoose` per class: the new limit minus the number currently prepared (always-prepared excluded), never below zero. It MUST also report `CantripsToChoose` per class: the Table F count plus the extra-cantrip allowance (FR-096) minus the cantrips already known, never below zero; granted cantrips (FR-097) are not counted. A class that enters with this level (FR-112) has empty lists, so both numbers are the full headroom. The additions themselves are made with FR-089 for every class. | SRD 5.2.1, "Bard", "Prepared Spells of Level 1+" ("Whenever that number increases, choose additional spells ... "), p. 32; same entry pp. 37, 42, 54, 58, 65, 71, 78 | happy |
| FR-048 | A *replacement* (FR-090) MUST be accepted only on a trigger in Table I: after finishing a Long Rest for Cleric, Druid, Paladin, Ranger and Wizard; on gaining a level for Bard, Sorcerer and Warlock; after finishing a Short Rest for a Wizard of level 5 or higher (Memorize Spell, FR-091). A replacement on any other trigger MUST be rejected with `ChangeNotAllowedNow`. Additions (FR-089) are not triggered and are not rejected by this requirement. | SRD 5.2.1, "Preparing Spells", Spell Preparation by Class table, p. 104; Wizard "Memorize Spell", p. 79 | unhappy (state error) |
| FR-049 | A change MUST replace at most the number in table I for the class and trigger: any number for Cleric, Druid and Wizard on a Long Rest; one for Bard, Paladin, Ranger, Sorcerer and Warlock; one for a Wizard on a Short Rest (FR-091). Replacements are counted as in FR-090. More MUST be rejected with `TooManyReplacements`; replacing none MUST be accepted. | SRD 5.2.1, Spell Preparation by Class table, p. 104; "Changing Your Prepared Spells" entries, pp. 32, 37, 42, 54, 58, 65, 72, 78; "Memorize Spell", p. 79 | unhappy (boundary) |
| FR-050 | A Wizard MUST prepare only spells in their spellbook; any other spell MUST be rejected with `NotInSpellbook`. | SRD 5.2.1, "Wizard", "Prepared Spells of Level 1+" ("choose four spells from your spellbook"), p. 78 | unhappy (invalid input) |
| FR-051 | An empty prepared list (zero spells) MUST be valid for every class and level. | SRD 5.2.1, "Preparing Spells", p. 104 | unhappy (empty and minimal) |
| FR-052 | Cantrips known (those chosen, not granted) MUST be level 0 spells on the class's list, no more than the class's count in table F plus its extra-cantrip allowance (FR-096), without duplicates; violations MUST be rejected with the matching kind (`NotOnClassList`, `TooManyCantrips`, `DuplicateSpell`, `NotACantrip`). | SRD 5.2.1, "Bard", "Cantrips", p. 32 (and each class); "Divine Order", p. 37; "Primal Order", p. 42 | unhappy (invalid input) |
| FR-053 | A cantrip replacement MUST be accepted at the right trigger and for at most one cantrip: on gaining a level for Bard, Cleric, Druid, Sorcerer and Warlock; on finishing a Long Rest for Wizard. | SRD 5.2.1, "Bard", "Cantrips", p. 32; "Cleric", p. 36; "Druid", p. 42; "Sorcerer", p. 64; "Warlock", p. 71; "Wizard", p. 77 | unhappy (state error) |
| FR-054 | A preparation or cantrip rejection MUST list every problem found, in a fixed order, and leave the caster's state unchanged. | `doc/rules/architecture.md` §6; `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-068 | The Spellcasting Focus kinds that can satisfy a Material component MUST be the class kinds of the casting class plus the kinds a caller-reported feature allows (`CastingContext.FeatureFocusKinds` of the casting spec, FR-129). The class kinds are: `FocusMusicalInstrument` (Bard), `FocusHolySymbol` (Cleric, Paladin), `FocusDruidic` (Druid, Ranger), `FocusArcane` (Sorcerer, Warlock, Wizard), plus `FocusSpellbook` for Wizard (table A). A held focus whose kind is neither a class kind of the casting class nor in `FeatureFocusKinds` MUST NOT substitute. | SRD 5.2.1, class "Spellcasting Focus" entries, pp. 32, 37, 42, 54, 58, 65, 72, 78; "Components" ("can substitute a Spellcasting Focus if the caster has a feature that allows that substitution"), p. 106; Warlock "Pact of the Blade", "Pact of the Tome", p. 74 | unhappy (invalid input) |
| FR-084 | Every operation in this spec MUST be deterministic and use no dice: the same state, command and spell MUST give an equal result. The domain services MUST be tested for this at unit level and each use case at application level. Concentration saves and spell attack rolls are rolled through the pure `d20` rules functions (d20-tests.md FR-069; a use case composing them) or `d20.Resolver` (resolve-d20-test-use-case.md; a host), never by this context. | `doc/rules/architecture.md` §3 (Purity); `doc/rules/testing.md` §4 | unhappy (determinism) |
| FR-085 | Every use case in section 6, and every use case of the casting spec, MUST record one span, one outcome counter increment (on the counter section 7 names for it) and one duration, with the attributes in section 7. A rule rejection MUST set `srd5e.outcome` to `failure` (the shared values are `success` and `failure`, shared-types-and-ownership.md FR-041) and MUST NOT set error status. | `doc/rules/observability.md` §5, §6 | happy |
| FR-086 | Each use case MUST return an equal result with the zero-value no-op `telemetry.Telemetry` and with the recording `telemetrytest` doubles. | `doc/rules/observability.md` §9 | happy |
| FR-087 | An invalid-command error returned by a use case (FR-126, FR-129) MUST be returned unchanged, be recorded on the span only (`RecordError`), increment no outcome counter and be logged nowhere; use cases never recover a panic. | `doc/rules/observability.md` §7 ("Errors: who records what") | unhappy (state error) |
| FR-088 | A use case MUST NOT log at `telemetry.LevelInfo` or above for a normal outcome, a rejection included; at most one `telemetry.LevelDebug` summary per call. | `doc/rules/observability.md` §7 | happy |
| FR-089 | `ChooseAdditionalSpells` MUST let any of the eight classes add level 1+ spells and cantrips to its lists without a trigger, up to the headroom: level 1+ spells up to `SpellsToChoose` (FR-047), cantrips up to the Table F count plus the extra-cantrip allowance (FR-096) minus the cantrips already known. Each addition MUST pass the checks of FR-040, FR-041, FR-043, FR-044, FR-050 and FR-052. It is the operation for the initial list at level 1 and for the additional spells chosen when a level raises the limit, including for Cleric, Druid, Paladin, Ranger and Wizard. More than the headroom MUST be rejected with `TooManyPrepared` or `TooManyCantrips`; an empty addition MUST be accepted with no events. | SRD 5.2.1, "Bard", "Prepared Spells of Level 1+" ("To start, choose ..."; "Whenever that number increases, choose additional spells"), p. 32, and the same entries pp. 37, 42, 54, 58, 65, 71, 78; "Cantrips" entries | happy |
| FR-090 | A *replacement* MUST be defined as a spell in the class's current prepared list (not always-prepared) that is absent from the new prepared list; the number of replacements is the count of such spells. A spell in the new list and not in the old is an *addition*, subject to FR-089 headroom and not counted as a replacement. A change that both replaces and adds MUST be accepted when the replacements fit FR-049 and the new list fits FR-042. | SRD 5.2.1, "Changing Your Prepared Spells" entries, pp. 32, 37, 42, 54, 58, 65, 72, 78 ("replace one spell on your list with another"); package design for the count (section 9) | unhappy (boundary) |
| FR-091 | A Wizard of level 5 or higher MUST be able to replace one prepared level 1+ spell with another level 1+ spell from the spellbook on the `TriggerShortRest` trigger (Memorize Spell). A Wizard below level 5, or any other class, on `TriggerShortRest` MUST be rejected with `ChangeNotAllowedNow`. The new spell MUST be in the spellbook (FR-050). | SRD 5.2.1, Wizard "Level 5: Memorize Spell", p. 79 | unhappy (state error) |
| FR-092 | `CreateSpellcasting` MUST build the initial `SpellcastingState` from the `CharacterID` (FR-127), `ClassLevels`, the always-prepared sets, the extra-cantrip allowances, the granted cantrips, the starting cantrips, the starting prepared lists and, for a Wizard, the starting spellbook. It MUST apply the same checks as FR-040 to FR-052, MUST derive full slots (FR-020) and a full Pact Magic pool, and MUST emit no events. | SRD 5.2.1, "Create Your Character", "Spellcasting", pp. 22-23; "Bard", "Cantrips", "Prepared Spells of Level 1+" ("To start, choose"), p. 32; "Wizard", p. 78 | happy |
| FR-093 | `NewClassSpellcasting` MUST return `(ClassSpellcasting, error)` for untrusted input: a valid `ClassSpellcasting`, or an error listing every problem in a fixed order (FR-054) as an `errors.Join` of typed `*PreparationProblem` values, `errors.Is` `srderr.ErrInvalidInput`, and never panic. `MustClassSpellcasting` MUST panic with that same error for the same inputs. | `doc/rules/architecture.md` §6 (untrusted input) | unhappy (invalid input) |
| FR-096 | Each class MAY have an extra-cantrip allowance (`ExtraCantrips`, default zero), supplied by a feature: one for a Cleric with Thaumaturge and for a Druid with Magician. The cap in FR-052 is the Table F count plus the allowance. The extra cantrip MUST be a cantrip of the class's own list. | SRD 5.2.1, Cleric "Divine Order: Thaumaturge", p. 37; Druid "Primal Order: Magician", p. 42 | happy |
| FR-097 | A granted cantrip (`GrantedCantrip`: spell id, the `SpellcastingClass` whose ability and DC it uses, and the `ClassID` of the spell list its feature names, supplied by a feature) MUST count as a known cantrip of that class for casting (FR-055) and for its DC and attack bonus (FR-005); it MUST NOT count against the Table F count plus allowance, nor against the prepared limit; and it MUST NOT be checked against the casting class's own list. | SRD 5.2.1, Paladin "Blessed Warrior", p. 54 ("count as Paladin spells for you"); Ranger "Druidic Warrior", p. 59 | happy |
| FR-098 | On the `TriggerGainedLevel` trigger, one granted cantrip MAY be replaced by another cantrip the host supplies as eligible for that feature; replacing more than one in a change MUST be rejected with `TooManyReplacements`, and a replacement on another trigger with `ChangeNotAllowedNow`. | SRD 5.2.1, Paladin "Blessed Warrior", p. 54; Ranger "Druidic Warrior", p. 59 | unhappy (state error) |
| FR-109 | `ExpendSpellSlot` MUST expend one slot of a given level from the chosen pool (Spellcasting or Pact Magic), with the outcomes of FR-021 to FR-024: `SpellSlotExpended`, or `NoSlotAvailable` with the state unchanged; it wraps `SpellcastingState.ExpendSlot` (FR-143). It MUST NOT apply the one-slot-per-turn rule (that is a casting rule). | SRD 5.2.1, "Spell Slots", p. 104; Druid "Nature Magician", p. 43 (needs a standalone slot operation) | happy |
| FR-110 | `RegainSpellSlot` MUST restore expended slots of a given level in the chosen pool, up to a requested count and never above the maximum, and emit `SpellSlotsRestored`; when no slot of that level is expended it MUST return `NoExpendedSlot` and leave the state unchanged. | SRD 5.2.1, "Spell Slots", p. 104; Warlock "Magical Cunning", p. 72 (needs a partial restore); package design for the result | unhappy (state error) |
| FR-111 | When a Warlock level changes the Pact Magic slot count and the Slot Level, remaining slots MUST become `min(newMaximum, oldRemaining + max(0, newMaximum - oldMaximum))` slots, all at the new Slot Level. This is a package decision where the SRD is silent (section 9). | Package design (section 9); SRD 5.2.1, Warlock Features, p. 71 | happy |
| FR-112 | `AdvanceSpellcasting` MUST validate the new `ClassLevels` against the state's: it MUST reject (state unchanged) with `LevelLowered` when any class level is lower, `ClassRemoved` when a class in the state is missing, `LevelsSkipped` when the total rises by more than one, and `NoChange` when nothing changed. A new spellcasting class MUST enter with empty lists, full headroom (FR-089) and no cantrips. | Package design (section 9); SRD 5.2.1, "Level Advancement", p. 23; "Multiclassing", p. 24 | unhappy (state error) |
| FR-116 | `SpellcastingStats` MUST expose the spell attack roll's modifiers as the `d20` type `RollModifiers` (`NewRollModifiers(ability).WithProficiencyBonus(pb)`, `Others()` empty) (shared-types-and-ownership.md FR-003; d20-tests.md section 4.3): the spellcasting ability modifier and the Proficiency Bonus, with `others` empty. The slots of `RollModifiers` hold kernel `modifier.Modifier`s, not `AbilityModifier` or `ProficiencyBonus` (d20-tests.md D-16), so this spec converts: the ability slot is `modifier.New(abilityModifier.Value())` and the proficiency slot is `modifier.New(proficiencyBonus.Value())` (the integer value of each). The caller passes the result to the pure `d20` rules functions (a use case composing them) or to `d20.Resolver` (a host) and does not recompute it. The sum of its parts equals the `SpellAttackBonus` of FR-004. | SRD 5.2.1, "Spells", "Attack Rolls", p. 106; shared-types-and-ownership.md FR-003; d20-tests.md D-16 | happy |
| FR-117 | A spell id named in a command (a prepared, cantrip, spellbook or granted-cantrip list) but absent from the `map[content.SpellID]content.Spell` the caller supplied MUST NOT return an error or panic. `CreateSpellcasting`, `ChooseAdditionalSpells` and `ChangePreparedSpells` MUST return their operation-prefixed content-missing variant (`CreateSpellcastingContentMissing`, `ChooseAdditionalSpellsContentMissing`, `ChangePreparedSpellsContentMissing`; shared-types-and-ownership.md FR-017), each holding one `content.MissingContent{Kind: spell, ID}` (shared-types-and-ownership.md FR-018) per absent id, in the order the ids appear in the command (prepared, cantrips, spellbook, granted cantrips), duplicates removed. It takes precedence over every rule rejection: no other check is evaluated, because the checks need the `Spell`, and the state MUST stay unchanged. Telemetry for such a result is FR-132. This is a package decision (section 9). | Package design (section 9); shared-types-and-ownership.md FR-017, FR-018; content-schema-and-decoder.md FR-147; `doc/rules/architecture.md` §4 (operation-prefixed variants), §6 (expected absence is a result) | unhappy (invalid input) |
| FR-118 | `Concentration` MUST hold a closed `ConcentrationSubject` (an interface with an unexported marker method), either a spell (`ConcentrationSubjectSpell(SpellID)`) or a non-spell effect (`ConcentrationSubjectFeature(FeatureID)`), and a `ConcentrationKind` (`ConcentrationEffect` or `ConcentrationCasting`). The kind `ConcentrationCasting` applies to a spell subject only (casting spec FR-101); a feature subject is always `ConcentrationEffect`. Events carry the subject, not a bare spell id. The SRD's Concentration rules apply to "spells and other effects", so a host can begin and end Concentration for a feature (casting spec FR-119, FR-120). | SRD 5.2.1, "Concentration" ("Some spells and other effects require Concentration"; "activate another effect that requires Concentration"), p. 179 | happy |
| FR-126 | A command that names a class the `SpellcastingState` does not hold MUST be rejected with an error that is `errors.Is` `ErrClassNotHeld` (wrapping `srderr.ErrInvalidInput`; a returned error, never a panic and never a result variant): `CastSpell` (casting spec FR-108), `ChooseAdditionalSpells`, `ChangePreparedSpells`, and a `GrantedCantrip` whose casting class is not held. Both pools always exist in the state (a pool with no slots has maximum zero), so naming `PoolSpellcasting` or `PoolPactMagic` (values of `SpellSlotPool`) in `ExpendSpellSlot` or `RegainSpellSlot` is never an error: an empty pool gives `NoSlotAvailable` (FR-109) or `NoExpendedSlot` (FR-110). | `doc/rules/architecture.md` §6; `errors-and-results.md` FR-025 to FR-027; FR-023, FR-107 | unhappy (invalid input) |
| FR-127 | `SpellcastingState` MUST hold the `CharacterID` of the character it belongs to, supplied to `CreateSpellcasting` and never changed by a transition. Every event in this spec (`SpellSlotExpended`, `SpellSlotsRestored`, `PreparedSpellsChanged`) MUST implement `character.Event` (character-aggregate.md FR-085) and carry that `CharacterID`; they form their own closed variant family `SpellcastingEvent`, which implements `character.Event` directly and is outside the closed `CharacterRecordEvent` of character-aggregate.md (as that spec's FR-085 allows). The casting and Concentration events are the sibling family `CastingEvent` (FR-141). Events from two characters are distinguishable. | character-aggregate.md FR-085; shared-types-and-ownership.md FR-019 | happy |
| FR-129 | `SpellcastingFocusKind` MUST be a typed enum (with `String()`) with exactly these values: the class kinds `FocusMusicalInstrument`, `FocusHolySymbol`, `FocusDruidic`, `FocusArcane` and `FocusSpellbook`, and the feature-granted kinds `FocusPactWeapon` (Pact of the Blade) and `FocusBookOfShadows` (Pact of the Tome). `CastingContext.FeatureFocusKinds` (a set of `SpellcastingFocusKind`, default empty) lists the kinds a feature the caller reports allows the caster to use as a Spellcasting Focus. It MUST hold only feature-granted kinds; `NewCastingContext` MUST reject a class kind in it with an error that is `errors.Is` `srderr.ErrInvalidInput`. The caller derives it from the feature's own state (the pact weapon bond of class-warlock.md FR-084, the Book of Shadows of class-warlock.md FR-100) and sets `HeldFocus` to the kind actually held; this spec tracks no bond and no book. | SRD 5.2.1, "Components", p. 106; Warlock "Pact of the Blade", "Pact of the Tome", p. 74 | unhappy (invalid input) |
| FR-132 | A `CreateSpellcastingContentMissing`, `ChooseAdditionalSpellsContentMissing` or `ChangePreparedSpellsContentMissing` result MUST be recorded exactly once: `srd5e.outcome` `failure`, `srd5e.character.result` `content_missing`, span error status (a returned failure result sets error status, observability.md §6), one increment of the counter section 7 names for the use case, no `srd5e.character.rejection`, no `error.type` (a returned failure carries none, errors-and-results.md FR-043) and no log. This is the recording of character-aggregate.md FR-071 and character-creation.md FR-098, so a consumer alerting on `result=content_missing` sees spellcasting too. | `doc/rules/observability.md` §6, §7 ("Errors: who records what"); character-aggregate.md FR-071; errors-and-results.md FR-043 | unhappy (telemetry on failure) |
| FR-136 | Every type that a command or a result of section 6 names MUST be defined in section 5 and exported from package `character`, including `Spellbook`, `CantripsKnown`, `ClassSpellcasting` `SpellcastingFeature`, `SpellcastingFocusKind`, `CastSource`, `SpellSlotChoice`, `Concentration`, `CastingInProgress`, `SpellcastingSummary`, `SpellSlotPools`, `SlotPoolView`, `FeatureSpellAbility` and the outcome types of FR-143, so that no consumer needs an unexported type to read a result (`doc/rules/architecture.md` §4, handoffs; §7, exports). The types of other specs are not re-exported (section 6). | `doc/rules/architecture.md` §4 (handoffs), §7 (public API) | happy |
| FR-143 | The pure transitions MUST be exported, so a `character` use case of another spec composes them without calling a use case (shared-types-and-ownership.md FR-044), each returning a new state with events and recording nothing. `SpellcastingState.ExpendSlot(SpellSlotPool, SlotLevel)` returns the closed `SlotExpenditure` (an interface with an unexported marker method): `SlotExpenditureExpended(state, event)` (a `SpellSlotExpended`) or `SlotExpenditureRejected(SlotOperationRejection)` (`NoSlotAvailable`); `RegainSlot(pool, SlotLevel, count)` returns `SlotRecovery` (`SlotRecoveryRegained(state, event)` or `SlotRecoveryRejected`); `Advance(ClassLevels)` returns `SpellcastingLevelChange` (`SpellcastingLevelChanged(state, spellsToChoose, cantripsToChoose, events)` or `SpellcastingLevelChangeRejected(SpellcastingAdvanceRejection)`). `ChoosePreparation(...)` (additions and cantrips, FR-089) and `ChangePreparation(...)` (replacements, cantrip and granted-cantrip changes, FR-090 to FR-098) take the state, class, trigger where one applies, the lists and the caller's `map[content.SpellID]content.Spell`, and return the exported closed `PreparationOutcome`: `PreparationAccepted(state, events)`, `PreparationRefused(PreparationRejection)` or `PreparationContentMissing([]content.MissingContent)` (FR-117). `ExpendSpellSlot` (FR-109), `RegainSpellSlot` (FR-110), `AdvanceSpellcasting`, `ChooseAdditionalSpells` and `ChangePreparedSpells` wrap these and map each outcome to their own `<UseCase>Result`; the rules are written once (FR-021 to FR-024, FR-047, FR-089). Same exported names as section 5; FR-084 and FR-028 hold for them. | Package design (architecture.md §4, §7; shared-types-and-ownership.md FR-044); SRD 5.2.1, "Spell Slots", p. 104 | happy |
| FR-144 | `SpellcastingState.Slots(SpellSlotPool)` MUST return a read-only `SlotPoolView` with `Remaining(SlotLevel)`, `Maximum(SlotLevel)` and `Expended(SlotLevel)` (maximum minus remaining), usable without `ExpendSpellSlot`, `RegainSpellSlot` or any rejection path. For the Spellcasting pool each is the per-level count of `SpellSlots`; for the Pact Magic pool the counts are those of the one Slot Level and zero at every other level (FR-013); an empty pool reads zero everywhere. The view exposes no setter and no mutable slice or map (FR-028). | Package design (architecture.md §4, handoffs); SRD 5.2.1, "Spell Slots", p. 104 | happy |
| FR-145 | A spell granted by a feat, a species or other source that belongs to no class (the Magic Initiate spells: two cantrips and one always-prepared level 1 spell cast with the ability the character chose; the lineage, legacy and Gnomish Lineage spells, which use a chosen ability the same way, `SpeciesGrantedSpells.Ability`) MUST get its ability, DC and attack bonus from a `FeatureSpellAbility` (a value holding one `Ability`) through `StatsForAbility(FeatureSpellAbility, AbilityModifier, ProficiencyBonus)`, which returns the same `SpellcastingStats` as FR-003, FR-004 and FR-116. It adds nothing to `ClassSpellcasting`, prepared limits or cantrip counts, and a Spellcasting Focus comes only from `FeatureFocusKinds` (FR-068: no class kinds). This spec does not import `feats-behaviour.md` (rank 15 against rank 13): the host builds the value from the feat's `FeatGrantedSpells.Ability` or the species' `SpeciesGrantedSpells.Ability`, and a cast takes the resulting stats in place of a class (`CastModeFeatureGranted`, casting-and-concentration.md FR-121 to FR-124, FR-149). The free cast is gated by the host through `ExpendFeatUse` or `ExpendSpeciesTraitUse` before the cast; their always-prepared marking stays with the owner (not in `PreparedSpells`); a cast using a slot is `CastModeNormal` with a `SpellSlotChoice` and touches no feat state (feats-behaviour.md FR-033). | SRD 5.2.1, "Magic Initiate", p. 87 ("is your spellcasting ability for this feat's spells"; "You can also cast the spell using any spell slots you have"); "Always-Prepared Spells", p. 104; Q8 of feats-behaviour.md; "Elven Lineage", p. 84; "Gnomish Lineage", p. 85; "Fiendish Legacy", p. 86 | happy |
| FR-146 | `SlotsFor(ClassLevels)` MUST return the exported value type `SpellSlotPools`: `Spellcasting` (a `SpellSlots`, the single-class table C or D or the Multiclass Spellcaster table, FR-010, FR-011, FR-030 to FR-033; all maximums zero when no class has Spellcasting) and `PactMagic` (a `PactMagicSlots` with a `HasPactMagic()` accessor, FR-012; the zero value, with `HasPactMagic()` false, when the character has no Warlock level, FR-034), at full maximums with remaining equal to maximum (FR-020). It spends nothing and reads no state. A `SpellcastingState` built from it holds an empty `PactMagicSlots` (count zero) where `PactMagic` is the zero value, so both pools exist (FR-126). `SpellSlotPools` is also the `SpellSlotsAfter` type of level-advancement-and-multiclassing.md FR-018. | SRD 5.2.1, "Multiclassing", "Spellcasting", pp. 25-26; package design for the value type | happy |
| FR-147 | A spell cast from a magic item uses no spell slot of the user's, no `ClassSpellcasting` entry and no `CastSource` (its entry is magic-item-rules.md `SpellFromItem`). Interpretation of `ItemCasterLevel.lowest` (magic-item-rules.md FR-025): the spell is cast at the spell's lowest possible level (its own `SpellLevel`) and, for a cantrip, at its base (unscaled) tier, so no Cantrip Upgrade applies and FR-038's total character level is not used. Reasonable reading: "lowest possible spell and caster level" has no other floor for a cantrip, whose scaling is by character level (p. 107). | SRD 5.2.1, "Spells Cast from Items", p. 206 ("at the lowest possible spell and caster level, doesn't expend any of the user's spell slots"); "Acid Splash" Cantrip Upgrade, p. 107 | unhappy (rule interaction) |
| FR-148 | `CastSource` MUST have a fifth value `scroll` (exported `CastSourceScroll`), for a spell cast from a Spell Scroll: it expends no slot and is not prepared or known by the caster. The cast rules (`CastModeSpellScroll`, derivation by `CastSpell`, FR-106) are the casting spec's follow-up of spell-scroll-use.md Q10. | SRD 5.2.1, "Casting without Slots", p. 104; spell-scroll-use.md Q10 | happy |
| FR-149 | `CastSource` MUST have a sixth variant `duplicated` (exported `CastSourceDuplicated`), for a spell that Wish duplicates: it expends no slot and needs no list, preparation, components or casting time. The cast rules (`CastModeDuplicated`, derivation by `CastSpell`) are casting-and-concentration.md FR-150 (wish-spell.md A1). `CastingInProgress` never holds it (a duplicated cast is immediate). | SRD 5.2.1, "Wish", p. 175 | happy |

### Data tables

All values are read from the Features tables cited with each table. `-` means
no slots of that level (FR-013).

#### Table A: spellcasting ability and focus (FR-001, FR-002, FR-068, FR-129)
| Label | Class | Feature | Spellcasting ability | Spellcasting Focus | Source | Path |
|---|---|---|---|---|---|---|
| A1 | Bard | Spellcasting | Charisma | Musical Instrument | SRD 5.2.1, "Bard", p. 32 | happy |
| A2 | Cleric | Spellcasting | Wisdom | Holy Symbol | SRD 5.2.1, "Cleric", p. 37 | happy |
| A3 | Druid | Spellcasting | Wisdom | Druidic Focus | SRD 5.2.1, "Druid", p. 42 | happy |
| A4 | Paladin | Spellcasting | Charisma | Holy Symbol | SRD 5.2.1, "Paladin", p. 54 | happy |
| A5 | Ranger | Spellcasting | Wisdom | Druidic Focus | SRD 5.2.1, "Ranger", p. 58 | happy |
| A6 | Sorcerer | Spellcasting | Charisma | Arcane Focus | SRD 5.2.1, "Sorcerer", p. 65 | happy |
| A7 | Warlock | Pact Magic | Charisma | Arcane Focus | SRD 5.2.1, "Warlock", p. 72 | happy |
| A8 | Wizard | Spellcasting | Intelligence | Arcane Focus or spellbook | SRD 5.2.1, "Wizard", p. 78 | happy |

The focus column lists the class kinds of `SpellcastingFocusKind` XX`FocusDruidic`, `FocusArcane`, `FocusSpellbook`). The feature-granted kinds `FocusPactWeapon` and `FocusBookOfShadows`
(SRD 5.2.1, Warlock, p. 74) belong to no class row: a caller reports them in
`CastingContext.FeatureFocusKinds` (FR-129), and they then work for any casting class (FR-068).

#### Table B: spell save DC and spell attack bonus (FR-003, FR-004, FR-006)
| Label | Ability modifier | Proficiency Bonus | DC | Attack bonus | Source | Path |
|---|---|---|---|---|---|---|
| B1 | -5 | +2 | 5 | -3 | SRD 5.2.1, "Spells", p. 106 | unhappy (boundary) |
| B2 | -1 | +2 | 9 | +1 | SRD 5.2.1, "Spells", p. 106 | unhappy (boundary) |
| B3 | 0 | +2 | 10 | +2 | SRD 5.2.1, "Spells", p. 106 | unhappy (boundary) |
| B4 | +3 | +2 | 13 | +5 | SRD 5.2.1, "Spells", p. 106 | happy |
| B5 | +3 | +3 | 14 | +6 | SRD 5.2.1, "Spells", p. 106 | happy |
| B6 | +4 | +4 | 16 | +8 | SRD 5.2.1, "Spells", p. 106 | happy |
| B7 | +5 | +6 | 19 | +11 | SRD 5.2.1, "Spells", p. 106 | happy |
| B8 | +10 | +6 | 24 | +16 | SRD 5.2.1, "Spells", p. 106 | unhappy (boundary) |

#### Tables C, D, E and G: slots, Pact Magic and prepared spells by class level (FR-010 to FR-017, FR-039)

Source: Features tables, pp. 31, 36, 41, 53, 58, 65, 71, 77, and the Multiclass Spellcaster
table, p. 26 (table C is also indexed by total spellcaster level, FR-031). Path: happy,
except levels 1 and 20 which are unhappy (boundary). Slot cells (C, D) list the slots of
spell levels 1, 2, 3, ... in order; every spell level not listed has no slots and reads as
zero maximum, the SRD's "-" (FR-013). C: Bard, Cleric, Druid, Sorcerer, Wizard (levels 1-9).
D: Paladin and Ranger (levels 1-5). E: Warlock Pact Magic, as slots x Slot Level, all slots of
the Slot Level (FR-012, FR-016). G: prepared level 1+ spells (FR-015).

| Level | C slots | D slots | E (slots x Slot Level) | G Bard, Cleric, Druid | G Paladin, Ranger | G Sorcerer | G Warlock | G Wizard |
|---|---|---|---|---|---|---|---|---|
| 1 | 2 | 2 | 1 x 1 | 4 | 2 | 2 | 2 | 4 |
| 2 | 3 | 2 | 2 x 1 | 5 | 3 | 4 | 3 | 5 |
| 3 | 4 2 | 3 | 2 x 2 | 6 | 4 | 6 | 4 | 6 |
| 4 | 4 3 | 3 | 2 x 2 | 7 | 5 | 7 | 5 | 7 |
| 5 | 4 3 2 | 4 2 | 2 x 3 | 9 | 6 | 9 | 6 | 9 |
| 6 | 4 3 3 | 4 2 | 2 x 3 | 10 | 6 | 10 | 7 | 10 |
| 7 | 4 3 3 1 | 4 3 | 2 x 4 | 11 | 7 | 11 | 8 | 11 |
| 8 | 4 3 3 2 | 4 3 | 2 x 4 | 12 | 7 | 12 | 9 | 12 |
| 9 | 4 3 3 3 1 | 4 3 2 | 2 x 5 | 14 | 9 | 14 | 10 | 14 |
| 10 | 4 3 3 3 2 | 4 3 2 | 2 x 5 | 15 | 9 | 15 | 10 | 15 |
| 11 | 4 3 3 3 2 1 | 4 3 3 | 3 x 5 | 16 | 10 | 16 | 11 | 16 |
| 12 | 4 3 3 3 2 1 | 4 3 3 | 3 x 5 | 16 | 10 | 16 | 11 | 16 |
| 13 | 4 3 3 3 2 1 1 | 4 3 3 1 | 3 x 5 | 17 | 11 | 17 | 12 | 17 |
| 14 | 4 3 3 3 2 1 1 | 4 3 3 1 | 3 x 5 | 17 | 11 | 17 | 12 | 18 |
| 15 | 4 3 3 3 2 1 1 1 | 4 3 3 2 | 3 x 5 | 18 | 12 | 18 | 13 | 19 |
| 16 | 4 3 3 3 2 1 1 1 | 4 3 3 2 | 3 x 5 | 18 | 12 | 18 | 13 | 21 |
| 17 | 4 3 3 3 2 1 1 1 1 | 4 3 3 3 1 | 4 x 5 | 19 | 14 | 19 | 14 | 22 |
| 18 | 4 3 3 3 3 1 1 1 1 | 4 3 3 3 1 | 4 x 5 | 20 | 14 | 20 | 14 | 23 |
| 19 | 4 3 3 3 3 2 1 1 1 | 4 3 3 3 2 | 4 x 5 | 21 | 15 | 21 | 15 | 24 |
| 20 | 4 3 3 3 3 2 2 1 1 | 4 3 3 3 2 | 4 x 5 | 22 | 15 | 22 | 15 | 25 |

#### Table F: cantrips known (FR-014, FR-018)

Source: Features tables, "Cantrips" column, pp. 31, 36, 41, 65, 71, 77. Each
band is constant across its levels. Paladin and Ranger: 0 at every level.

| Label | Class levels | Bard | Cleric | Druid | Sorcerer | Warlock | Wizard | Path |
|---|---|---|---|---|---|---|---|---|
| F1 | 1-3 | 2 | 3 | 2 | 4 | 2 | 3 | unhappy (boundary) |
| F2 | 4-9 | 3 | 4 | 3 | 5 | 3 | 4 | happy |
| F3 | 10-20 | 4 | 5 | 4 | 6 | 4 | 5 | happy |

Boundary tests pin the first and last level of each band (3 to 4, 9 to 10).

#### Table H: total spellcaster level and slots (FR-030 to FR-034)

Source: SRD 5.2.1, "Multiclassing", "Spellcasting", p. 25 and the Multiclass
Spellcaster table, p. 26. "Slots" are the Spellcasting pool; Pact Magic is
separate (FR-034).

| Label | Classes and levels | Total spellcaster level | Spellcasting slots (levels 1, 2, 3...) | Pact Magic | Path |
|---|---|---|---|---|---|
| H1 | Ranger 4, Sorcerer 3 | 2 + 3 = 5 | 4, 3, 2 | none | happy (SRD example, p. 25-26) |
| H2 | Paladin 1 only | n/a, single class: table D | 2 | none | happy |
| H3 | Paladin 1, Wizard 1 | 1 + 1 = 2 | 3 | none | unhappy (rule interaction: differs from H2) |
| H4 | Paladin 1, Ranger 1 | combined 2, half rounded up = 1 | 2 | none | unhappy (boundary, rounding) |
| H5 | Paladin 3, Ranger 3 | combined 6, half = 3 | 4, 2 | none | happy |
| H6 | Paladin 2, Fighter 3 | Fighter has no Spellcasting: single class, table D at 2 | 2 | none | unhappy (rule interaction) |
| H7 | Wizard 5, Fighter 4 | single Spellcasting class: table C at 5 | 4, 3, 2 | none | happy |
| H8 | Wizard 3, Cleric 3 | 6 | 4, 3, 3 | none | happy |
| H9 | Warlock 5, Wizard 3 | Wizard only counts: 3 | 4, 2 | 2 slots of level 3 | happy; `SlotsFor` returns `SpellSlotPools` (`spellcasting` 4, 2; `PactMagic` 2 x level 3), remaining = maximum (FR-146) |
| H10 | Warlock 5 only | n/a, Pact Magic: table E at 5 | none | 2 slots of level 3 | happy |
| H11 | Bard 10, Cleric 10 | 20 | 4, 3, 3, 3, 3, 2, 2, 1, 1 | none | unhappy (boundary) |
| H12 | Paladin 20 only | n/a, single class: table D at 20 | 4, 3, 3, 3, 2 | none | unhappy (boundary) |
| H13 | Fighter 5 only | no Spellcasting | all zero | none (`SpellSlotPools.PactMagic` zero value) | unhappy (empty and minimal; FR-146) |
| H14 | Wizard 5 / Fighter 2 | single Spellcasting class: table C at 5 | 4, 3, 2 | none (`PactMagic` zero value) | unhappy (rule interaction; FR-146) |

#### Table I: Spell Preparation by Class (FR-048, FR-049, FR-053, FR-091)

Source: SRD 5.2.1, "Preparing Spells", p. 104, the class entries cited at
FR-053 and FR-049, and Wizard "Memorize Spell", p. 79. The table gives the
trigger and the number of *replacements* (FR-090); additions are FR-089 and
have no trigger. Cantrip column: the trigger for replacing one cantrip.

| Class | Prepared list replacements when you... | Number of spells | Cantrip replacement trigger |
|---|---|---|---|
| Bard | gain a level | one | gain a level |
| Cleric | finish a Long Rest | any | gain a level |
| Druid | finish a Long Rest | any | gain a level |
| Paladin | finish a Long Rest | one | none (no cantrips from Spellcasting) |
| Ranger | finish a Long Rest | one | none (no cantrips from Spellcasting) |
| Sorcerer | gain a level | one | gain a level |
| Warlock | gain a level | one | gain a level |
| Wizard | finish a Long Rest | any | finish a Long Rest |
| Wizard, level 5+ (Memorize Spell) | finish a Short Rest | one | none |

#### Table J: highest preparable spell level, derived (FR-016, FR-017)

Derived from tables C, D and E; one boundary row per band edge.

| Label | Class and level | Highest preparable level | Path |
|---|---|---|---|
| J1 | Wizard 1, Wizard 2 | 1 | unhappy (boundary) |
| J2 | Wizard 3 | 2 | unhappy (boundary) |
| J3 | Wizard 17 | 9 | unhappy (boundary) |
| J4 | Paladin 4 | 1 | unhappy (boundary) |
| J5 | Paladin 5 | 2 | unhappy (boundary) |
| J6 | Paladin 17 | 5 | unhappy (boundary) |
| J7 | Warlock 2 | 1 | unhappy (boundary) |
| J8 | Warlock 3 | 2 | unhappy (boundary) |
| J9 | Warlock 9 | 5 | unhappy (boundary) |
| J10 | Wizard 3 in a Wizard 3 / Cleric 6 build | 2 (the Wizard's own table, FR-036) | unhappy (rule interaction) |

#### Table M: extra and granted cantrips (FR-052, FR-096 to FR-098)
| Label | Case | Result | FR | Path |
|---|---|---|---|---|
| M1 | Cleric 1 with Thaumaturge (allowance 1), 4 cantrips chosen | accepted (3 + 1) | FR-096 | unhappy (boundary) |
| M2 | Cleric 1 with Thaumaturge, 5 cantrips chosen | `TooManyCantrips` | FR-052 | unhappy (boundary) |
| M3 | Cleric 1 without Thaumaturge, 4 cantrips chosen | `TooManyCantrips` | FR-052 | unhappy (boundary) |
| M4 | Druid 4 with Magician, 4 cantrips chosen (table 3 plus 1); then 5 chosen | accepted; `TooManyCantrips` | FR-096 | unhappy (boundary) |
| M5 | Paladin 2 with Blessed Warrior, two granted Cleric cantrips, zero chosen | valid; both castable with the Paladin DC | FR-018, FR-097 | happy |
| M6 | Wizard 1 with 3 chosen cantrips and 2 granted cantrips | accepted: granted do not count | FR-097 | unhappy (rule interaction) |
| M7 | granted cantrip not on the casting class's list | accepted (checked against the feature's list) | FR-040, FR-097 | unhappy (rule interaction) |
| M8 | Ranger 2 with Druidic Warrior, replace one granted cantrip on `TriggerGainedLevel` | accepted | FR-098 | happy |
| M9 | replace two granted cantrips on `TriggerGainedLevel` | `TooManyReplacements` | FR-098 | unhappy (boundary) |
| M10 | replace one granted cantrip on `TriggerLongRest` | `ChangeNotAllowedNow` | FR-098 | unhappy (state error) |
| M11 | granted cantrip counted against the prepared limit | no: prepared list of 4 at Wizard 1 stays valid | FR-097 | unhappy (rule interaction) |

#### Table O: level change for Pact Magic and validation (FR-029, FR-111, FR-112)
| Label | Case | Result | FR | Path |
|---|---|---|---|---|
| O1 | Wizard 4 (1 used level 1 slot) to Wizard 5 | level 1: 3 of 4; level 2: 3 of 3; level 3: 2 of 2 | FR-029 | happy |
| O2 | Warlock 1 (slot spent) to 2 | 1 of 2 remaining at level 1 | FR-111 | unhappy (boundary) |
| O3 | Warlock 2 (1 remaining) to 3 | 1 of 2 remaining at level 2 (count unchanged, level changes) | FR-111 | unhappy (rule interaction) |
| O4 | Warlock 10 (2 of 2) to 11 | 3 of 3 remaining at level 5 | FR-111 | unhappy (boundary) |
| O5 | Warlock 4 (0 remaining) to 5 | 0 of 2 remaining at level 3 | FR-111 | unhappy (boundary) |
| O6 | same `ClassLevels` | `AdvanceRejected(noChange)` | FR-112 | unhappy (empty and minimal) |
| O7 | Wizard 5 to Wizard 4 | `AdvanceRejected(levelLowered)` | FR-112 | unhappy (state error) |
| O8 | Wizard 5 / Fighter 1 to Wizard 5 | `AdvanceRejected(classRemoved)` | FR-112 | unhappy (state error) |
| O9 | Wizard 5 to Wizard 7 | `AdvanceRejected(levelsSkipped)` | FR-112 | unhappy (boundary) |
| O10 | Wizard 5 to Wizard 5 / Cleric 1 | new Cleric entry: empty lists, Cleric `SpellsToChoose` 4, `CantripsToChoose` 3 | FR-112, FR-047 | happy |
| O11 | Wizard 3 (3 cantrips known) to Wizard 4 | Wizard `CantripsToChoose` 1 (Table F: 3 to 4) | FR-047 | unhappy (boundary) |
| O12 | Wizard 4 (4 cantrips known) to Wizard 5 | Wizard `CantripsToChoose` 0 (same band) | FR-047 | unhappy (boundary) |
| O13 | Cleric 3 with Thaumaturge (allowance 1, 4 cantrips known) to Cleric 4 | Cleric `CantripsToChoose` 1 (4 + 1 allowance = 5, minus 4 known) | FR-047, FR-096 | unhappy (rule interaction) |
| O14 | Paladin 2 with Blessed Warrior (2 granted cantrips) to Paladin 3 | Paladin `CantripsToChoose` 0 (granted cantrips not counted, no Table F cantrips) | FR-047, FR-018, FR-097 | unhappy (rule interaction) |

#### Table P: additions, replacements, creation (FR-047 to FR-049, FR-089 to FR-092)
| Label | Case | Result | FR | Path |
|---|---|---|---|---|
| P1 | Wizard 5, 9 prepared, replace one on `TriggerShortRest` from the spellbook | accepted | FR-091 | happy |
| P2 | Wizard 4, replace one on `TriggerShortRest` | `ChangeNotAllowedNow` | FR-091 | unhappy (state error) |
| P3 | Wizard 5, replace two on `TriggerShortRest` | `TooManyReplacements` | FR-049 | unhappy (boundary) |
| P4 | Wizard 5, `TriggerShortRest`, new spell not in the spellbook | `NotInSpellbook` | FR-050 | unhappy (invalid input) |
| P5 | Ranger 5, replace one on `TriggerShortRest` | `ChangeNotAllowedNow` | FR-091 | unhappy (state error) |
| P6 | Cleric 3 to 4 (6 to 7 prepared), add one | accepted via `ChooseAdditionalSpells` | FR-089 | happy |
| P7 | Cleric 3 to 4, add two | `TooManyPrepared` | FR-089 | unhappy (boundary) |
| P8 | Cleric on `TriggerGainedLevel`, replace one | `ChangeNotAllowedNow` | FR-048 | unhappy (state error) |
| P9 | Bard 2 to 3, replace one and add one | accepted | FR-090 | unhappy (rule interaction) |
| P10 | Bard on `TriggerGainedLevel`, replace two | `TooManyReplacements` | FR-049 | unhappy (boundary) |
| P11 | Paladin on `TriggerLongRest`, replace one | accepted | FR-049 | happy |
| P12 | Paladin on `TriggerLongRest`, replace two | `TooManyReplacements` | FR-049 | unhappy (boundary) |
| P13 | replace an always-prepared spell | `AlwaysPreparedNotReplaceable` | FR-046 | unhappy (state error) |
| P14 | `CreateSpellcasting`, Wizard 1, 3 cantrips, 4 prepared from a spellbook of 6 | valid state, full slots, no events | FR-092 | happy |
| P15 | `CreateSpellcasting`, Wizard 1, 5 prepared | failure with `TooManyPrepared` | FR-092, FR-093 | unhappy (boundary) |
| P16 | `CreateSpellcasting`, Warlock 1, 2 prepared, 2 cantrips | valid state; Pact pool 1 slot of level 1 | FR-092 | happy |
| P17 | Cleric: add zero spells | accepted, no events | FR-089 | unhappy (empty and minimal) |
| P18 | `NewClassSpellcasting` with a duplicate and a level 0 prepared spell | error listing `DuplicateSpell` and `CantripNotPreparable` in the fixed order | FR-093, FR-054 | unhappy (invalid input) |
| P19 | Wizard 5 / Cleric 5: the same spell prepared by both | two entries, each with its own class DC | FR-037 | unhappy (rule interaction) |
| P20 to P22 | `ChooseAdditionalSpells` or `ChangePreparedSpells` for a class the state does not hold, or with a `GrantedCantrip` whose casting class is not held | error `ErrClassNotHeld` | FR-126 | unhappy (invalid input) |

#### Table Q: slot operations (FR-109, FR-110)
| Label | Case | Result | FR | Path |
|---|---|---|---|---|
| Q1 | `ExpendSpellSlot` level 2, Wizard 5 | `SpellSlotExpended`, 2 of 3 | FR-109 | happy |
| Q2 | `ExpendSpellSlot` level 2 with 0 remaining | `NoSlotAvailable` | FR-109 | unhappy (state error) |
| Q3 | `ExpendSpellSlot` Pact pool, Warlock 5 | `SpellSlotExpended(pact_magic)` | FR-109 | happy |
| Q4 | `ExpendSpellSlot` with `SlotSpentThisTurn` true in the caller's context | still expended: not a casting | FR-109 | unhappy (rule interaction) |
| Q5 | `RegainSpellSlot` level 1, count 1, 2 expended | one restored | FR-110 | happy |
| Q6 | `RegainSpellSlot` level 1, count 3, 2 expended | two restored, capped | FR-110 | unhappy (boundary) |
| Q7 | `RegainSpellSlot` level 1 with none expended | `NoExpendedSlot`, unchanged | FR-110 | unhappy (empty and minimal) |
| Q8, Q9 | `ExpendSpellSlot` or `RegainSpellSlot` on the Pact pool, Wizard 5 only (the pool exists and is empty) | `NoSlotAvailable` or `NoExpendedSlot`, no error | FR-109, FR-110, FR-126 | unhappy (boundary) |

#### Table R: stats as `RollModifiers`, and missing content (FR-116, FR-117, FR-132)
| Label | Case | Result | FR | Path |
|---|---|---|---|---|
| R1 | Wizard 5, Intelligence modifier +3, Proficiency Bonus +3 | `AttackRollModifiers`: ability modifier +3, Proficiency Bonus +3, no other modifier; sum +6 equals `SpellAttackBonus` | FR-116 | happy |
| R2 | ability modifier -5, Proficiency Bonus +2 | sum -3 equals `SpellAttackBonus` (table B1) | FR-116, FR-006 | unhappy (boundary) |
| R3 | `CreateSpellcasting`, a prepared spell id absent from the supplied map | `CreateSpellcastingContentMissing` holding `MissingContent(spell, id)`; no state; result `content_missing`, span error status | FR-117, FR-132 | unhappy (invalid input) |
| R4 | `ChangePreparedSpells`, a new list names an id absent from the map and also exceeds the limit | `ChangePreparedSpellsContentMissing` holding only the missing id; `TooManyPrepared` is not evaluated; state unchanged | FR-117 | unhappy (rule interaction) |
| R5 | `ChooseAdditionalSpells`, one id absent from the prepared additions and one from the cantrips, the prepared id repeated | `ChooseAdditionalSpellsContentMissing` holding both ids once, prepared list first, then cantrips | FR-117 | unhappy (boundary) |

### Withdrawn ids

| Id | Status |
|---|---|
| FR-055 | Withdrawn: moved to casting-and-concentration.md (FR-055) |
| FR-056 | Withdrawn: moved to casting-and-concentration.md (FR-056) |
| FR-057 | Withdrawn: moved to casting-and-concentration.md (FR-057) |
| FR-058 | Withdrawn: moved to casting-and-concentration.md (FR-058) |
| FR-059 | Withdrawn: moved to casting-and-concentration.md (FR-059) |
| FR-060 | Withdrawn: moved to casting-and-concentration.md (FR-060) |
| FR-061 | Withdrawn: moved to casting-and-concentration.md (FR-061) |
| FR-062 | Withdrawn: moved to casting-and-concentration.md (FR-062) |
| FR-063 | Withdrawn: moved to casting-and-concentration.md (FR-063) |
| FR-064 | Withdrawn: moved to casting-and-concentration.md (FR-064) |
| FR-065 | Withdrawn: moved to casting-and-concentration.md (FR-065) |
| FR-066 | Withdrawn: moved to casting-and-concentration.md (FR-066) |
| FR-067 | Withdrawn: moved to casting-and-concentration.md (FR-067) |
| FR-069 | Withdrawn: moved to casting-and-concentration.md (FR-069) |
| FR-070 | Withdrawn: moved to casting-and-concentration.md (FR-070) |
| FR-071 | Withdrawn: moved to casting-and-concentration.md (FR-071) |
| FR-072 | Withdrawn: moved to casting-and-concentration.md (FR-072) |
| FR-073 | Withdrawn: moved to casting-and-concentration.md (FR-073) |
| FR-074 | Withdrawn: moved to casting-and-concentration.md (FR-074) |
| FR-075 | Withdrawn: moved to casting-and-concentration.md (FR-075) |
| FR-076 | Withdrawn: moved to casting-and-concentration.md (FR-076) |
| FR-077 | Withdrawn: moved to casting-and-concentration.md (FR-077) |
| FR-078 | Withdrawn: moved to casting-and-concentration.md (FR-078) |
| FR-079 | Withdrawn: moved to casting-and-concentration.md (FR-079) |
| FR-080 | Withdrawn: moved to casting-and-concentration.md (FR-080) |
| FR-081 | Withdrawn: moved to casting-and-concentration.md (FR-081) |
| FR-082 | Withdrawn: moved to casting-and-concentration.md (FR-082) |
| FR-083 | Withdrawn: moved to casting-and-concentration.md (FR-083) |
| FR-094 | Withdrawn: moved to casting-and-concentration.md (FR-094) |
| FR-095 | Withdrawn: moved to casting-and-concentration.md (FR-095) |
| FR-099 | Withdrawn: moved to casting-and-concentration.md (FR-099) |
| FR-100 | Withdrawn: moved to casting-and-concentration.md (FR-100) |
| FR-101 | Withdrawn: moved to casting-and-concentration.md (FR-101) |
| FR-102 | Withdrawn: moved to casting-and-concentration.md (FR-102) |
| FR-103 | Withdrawn: moved to casting-and-concentration.md (FR-103) |
| FR-104 | Withdrawn: moved to casting-and-concentration.md (FR-104) |
| FR-105 | Withdrawn: moved to casting-and-concentration.md (FR-105) |
| FR-106 | Withdrawn: moved to casting-and-concentration.md (FR-106) |
| FR-107 | Withdrawn: moved to casting-and-concentration.md (FR-107) |
| FR-108 | Withdrawn: moved to casting-and-concentration.md (FR-108) |
| FR-113 | Withdrawn: moved to casting-and-concentration.md (FR-113) |
| FR-114 | Withdrawn: moved to casting-and-concentration.md (FR-114) |
| FR-115 | Withdrawn: moved to casting-and-concentration.md (FR-115) |
| FR-119 | Withdrawn: moved to casting-and-concentration.md (FR-119) |
| FR-120 | Withdrawn: moved to casting-and-concentration.md (FR-120) |
| FR-121 | Withdrawn: moved to casting-and-concentration.md (FR-121) |
| FR-122 | Withdrawn: moved to casting-and-concentration.md (FR-122) |
| FR-123 | Withdrawn: moved to casting-and-concentration.md (FR-123) |
| FR-124 | Withdrawn: moved to casting-and-concentration.md (FR-124) |
| FR-125 | Withdrawn: moved to casting-and-concentration.md (FR-125) |
| FR-128 | Withdrawn: moved to casting-and-concentration.md (FR-128) |
| FR-130 | Withdrawn: moved to casting-and-concentration.md (FR-130) |
| FR-131 | Withdrawn: moved to casting-and-concentration.md (FR-131) |
| FR-133 | Withdrawn: moved to casting-and-concentration.md (FR-133) |
| FR-134 | Withdrawn: moved to casting-and-concentration.md (FR-134) |
| FR-135 | Withdrawn: moved to casting-and-concentration.md (FR-135) |
| FR-137 | Withdrawn: moved to casting-and-concentration.md (FR-137) |
| FR-138 | Withdrawn: moved to casting-and-concentration.md (FR-138) |
| FR-139 | Withdrawn: moved to casting-and-concentration.md (FR-139) |
| FR-140 | Withdrawn: moved to casting-and-concentration.md (FR-140) |
| FR-141 | Never allocated here: the id was used by casting-and-concentration.md (FR-141, event contract) |
| FR-142 | Never allocated here: the id was used by casting-and-concentration.md (FR-142, type export rule); this spec's equivalent is FR-136 |

## 5. Model

Everything here is package `character`, in files named `spellcasting*.go`, per `doc/rules/architecture.md` §3: immutable (unexported fields, value receivers, accessors without a Get prefix), pure, synchronous, free of dice and time. The variant notation `Name(a, b)` used in this spec lists the fields of a struct type `Name`; a closed result is an interface with an unexported marker method, a doc comment naming every variant, and one struct per variant, switched exhaustively. Optional values are a documented zero value or a (value, ok) accessor. Upstream types are listed in section 1 and not redefined; there is no `SpellListClass` (a class's list is its `ClassID` in `Spell.Classes`).

**Value objects**

- `SpellcastingClass`: enum of the eight classes (FR-001) with `feature`, `ability` (FR-002), `focuses` (class `SpellcastingFocusKind`s, FR-068) and `ClassID` (the key in `Spell.Classes`, FR-040). A view of a class; levels come from `ClassLevels` (FR-008).
- Typed enums (each with `String()`): `SpellcastingFeature` (`FeatureSpellcasting`, `FeaturePactMagic`); `SpellcastingFocusKind` (class kinds `FocusMusicalInstrument`, `FocusHolySymbol`, `FocusDruidic`, `FocusArcane`, `FocusSpellbook`; feature-granted `FocusPactWeapon`, `FocusBookOfShadows`, FR-129, with `IsFeatureGranted()`); `PreparationTrigger` (`TriggerLongRest`, `TriggerShortRest`, `TriggerGainedLevel`); `SpellSlotPool` (`PoolSpellcasting`, `PoolPactMagic`); `ConcentrationKind` (`ConcentrationEffect`, `ConcentrationCasting`).
- `SlotLevel` (1-9, FR-024): `NewSlotLevel(int) (SlotLevel, error)` (error `errors.Is` `ErrSlotLevelRange`) and `MustSlotLevel` (panics with that error).
- `SpellSlots` (per-level maximum and remaining, FR-019; `SpellSlots.ForLevels` derives it from `ClassLevels`, FR-010, FR-011, FR-030 to FR-032); `PactMagicSlots` (count, level, remaining; FR-012, FR-034, FR-111); `SpellSlotPools` (`Spellcasting` `SpellSlots`, `PactMagic` `PactMagicSlots`, zero value when absent; the return of `SlotsFor`, FR-146); `SlotPoolView` (`Remaining`, `Maximum`, `Expended` per `SlotLevel`; FR-144); `FeatureSpellAbility` (one `Ability`; FR-145).
- `SpellcastingStats`: ability, modifier, Proficiency Bonus, `SpellSaveDC`, `SpellAttackBonus`, `AttackRollModifiers` (`RollModifiers`; FR-003 to FR-007, FR-116).
- `PreparedSpells` (ordered `SpellID`s plus the always-prepared set, FR-045), `CantripsKnown` (FR-052), `Spellbook` (FR-050; used by class-wizard.md), `GrantedCantrip` (spell id, `SpellcastingClass`, the `ClassID` of the list its feature names; FR-097).
- `ClassSpellcasting`: one class, its `CantripsKnown`, `ExtraCantrips` (FR-096), `GrantedCantrip` set, `PreparedSpells` and for a Wizard the `Spellbook`. Enforces FR-014, FR-015, FR-041 to FR-044 at construction: `NewClassSpellcasting` returns `(ClassSpellcasting, error)` (FR-093). `SpellsToChoose` is derived (FR-047).
- `SpellcastingSummary` (value of `CalculateSpellcastingResult`): per class stats, cantrip limit, prepared limit, maximum preparable level (FR-014 to FR-017); the `SpellSlotPools` (Spellcasting `SpellSlots` and optional `PactMagicSlots` (zero value when absent), FR-030 to FR-034, FR-146).
- State-side shapes whose rules are in `casting-and-concentration.md`: `Concentration` (closed `ConcentrationSubject`, `ConcentrationSubjectSpell(SpellID)` | `ConcentrationSubjectFeature(FeatureID)`, and a `ConcentrationKind`; FR-118); `CastSource` (typed enum `CastSourceSlot`, `CastSourceCantrip`, `CastSourceRitual`, `CastSourceFeatureGranted`, `CastSourceScroll`, `CastSourceDuplicated`; FR-148, FR-149); `SpellSlotChoice` (closed: `SpellSlotChoiceSpellcasting(level)` | `SpellSlotChoicePactMagic`); `CastingInProgress` (spell id, `CastSource`, class, chosen `CastingTimeMode`, `SpellSlotChoice` (both optional), `TotalMinutes`, whole minutes of at least 1, set by the casting spec's FR-131). Exported variants carry the type prefix, so no generic name is exported from the package.

**Aggregate part.** `SpellcastingState` is an immutable value the host holds *beside* the `Character` aggregate: character-aggregate.md keeps spellcasting out of scope and this spec does not amend it; it is not in `FeatureStates` because a multiclass pool belongs to no single class (section 9). It holds the `CharacterID` (FR-127), `ClassLevels`, per-class `ClassSpellcasting`, Spellcasting `SpellSlots`, `PactMagicSlots`, `Concentration` and `CastingInProgress` (both optional, zero value meaning none). Every transition returns a new state plus events (FR-028). Exported pure methods: `ExpendSlot`, `RegainSlot`, `Advance` (FR-143) and the read accessor `Slots(pool)` (FR-144).

**Pure functions** (stateless, plain values only, so unit tested with no port, double or telemetry; `doc/rules/testing.md` §3): `Stats(...)`, `StatsForAbility(...)` (FR-145), `SlotsFor(ClassLevels)` returning `SpellSlotPools` (FR-146), `PreparedLimit`, `CantripLimit`, `MaxPreparableLevel`, and `ChoosePreparation` / `ChangePreparation` (additions, replacements, cantrip changes; FR-040 to FR-054, FR-089 to FR-093, FR-096 to FR-098, FR-117), returning the exported `PreparationOutcome` (FR-143); with the `SpellcastingState` transitions of FR-143 they are exported from package `character` so another spec's use case composes them (shared-types-and-ownership.md FR-044). Each use case maps its outcome to its own `<UseCase>Result` (shared-types-and-ownership.md FR-017). The casting and Concentration rules are in the casting spec.

**Rejections.** Reasons are enum values in rejection value objects, so kinds in two lists never collide: each kind constant carries its type prefix (`PreparationNotOnClassList`, `SpellcastingAdvanceLevelLowered`, `SlotOperationNoSlotAvailable`); the unprefixed shorthand of section 4 (`NotOnClassList`, `LevelLowered`, `NoSlotAvailable`) names the same values. `PreparationRejection` holds an ordered slice of `PreparationRejectionKind` in this order: `NotOnClassList`, `SpellLevelTooHigh`, `TooManyPrepared`, `DuplicateSpell`, `CantripNotPreparable`, `AlwaysPreparedNotReplaceable`, `ChangeNotAllowedNow`, `TooManyReplacements`, `NotInSpellbook`, `TooManyCantrips`, `NotACantrip`; each entry carries the spell id and class where it applies. Absent content is not a kind: it is the operation-prefixed `...ContentMissing` variant with a `[]content.MissingContent` (FR-117), which takes precedence over every rejection. `SpellcastingAdvanceRejection` holds one `SpellcastingAdvanceRejectionKind` (`LevelLowered`, `ClassRemoved`, `LevelsSkipped`, `NoChange`, checked in that order, FR-112); the `Spellcasting` prefix avoids a clash with level-advancement-and-multiclassing.md. `SlotOperationRejection` holds `NoSlotAvailable` or `NoExpendedSlot` (FR-109, FR-110). The kinds of section 4 are the enum values without their type prefix.

**Naming of results.** Every result is a closed variant set (interface with an unexported marker method) named `<UseCase>Result`.

| Use case | Result and variants (exported) | Shorthand in section 4 and the tables |
|---|---|---|
| `CalculateSpellcasting` | `CalculateSpellcastingCalculated(summary)` | n/a |
| `CreateSpellcasting` | `CreateSpellcastingCreated(state)`, `CreateSpellcastingRejected(rejection)`, `CreateSpellcastingContentMissing(missing)` | "valid state", "failure with `TooManyPrepared`" |
| `ChooseAdditionalSpells` | `ChooseAdditionalSpellsChosen(state, events)`, `...Rejected(rejection)`, `...ContentMissing(missing)` | "accepted", `TooManyPrepared` |
| `ChangePreparedSpells` | `ChangePreparedSpellsChanged(state, events)`, `...Rejected(rejection)`, `...ContentMissing(missing)` | same |
| `ExpendSpellSlot` | `ExpendSpellSlotExpended(state, event)`, `ExpendSpellSlotRejected(SlotOperationRejection)` | `SpellSlotExpended`, `NoSlotAvailable` |
| `RegainSpellSlot` | `RegainSpellSlotRegained(state, event)`, `RegainSpellSlotRejected(SlotOperationRejection)` | `SpellSlotsRestored`, `NoExpendedSlot` |
| `RestoreSpellSlots` | `RestoreSpellSlotsRestored(state, events)` (empty events when nothing was expended, FR-027) | n/a |
| `AdvanceSpellcasting` | `AdvanceSpellcastingAdvanced(state, spellsToChoose, cantripsToChoose, events)` (both per class), `AdvanceSpellcastingRejected(SpellcastingAdvanceRejection)` | `AdvanceRejected(kind)` |

**Events** (past tense): members of the closed variant family `SpellcastingEvent`, which implements `character.Event` directly (character-aggregate.md FR-085 leaves it open for this; its closed `CharacterRecordEvent` is that spec's own; shared-types-and-ownership.md FR-019, no kernel base type). Each carries the state's `CharacterID` (FR-127) and is defined here only; `RestApplied` is rests-and-hit-dice.md's: `SpellSlotExpended(level, pool)`, `SpellSlotsRestored(spellcasting, pactMagic)` and `PreparedSpellsChanged(class)`. The casting events are the family `CastingEvent`.

**Failures.** Rule outcomes (rejections, `NoSlotAvailable`, `NoExpendedSlot`, `...ContentMissing` (FR-117)) are closed results. Constructor violations and contradictory command input (FR-019, FR-024, FR-126, FR-129) are returned errors, each `errors.Is` `srderr.ErrInvalidInput` (never a panic outside `Must...` helpers). No `*srderr.PortError` exists here (no port does I/O).

## 6. Interface

Synchronous use cases (a struct per use case with `Do(ctx context.Context, cmd <UseCase>Command) (<UseCase>Result, error)`, built by `New<UseCase>(opts ...Option)` with the option `WithTelemetry(telemetry.Telemetry)`; `ctx` is first because each call starts a span and is never stored; the `error` is non-nil only for an invalid command such as `ErrClassNotHeld`, FR-126), each with an exported immutable `<UseCase>Command` value object (architecture.md §4):

| Use case | Command | Result |
|---|---|---|
| `CalculateSpellcasting` | `ClassLevels`, the six ability scores, `ProficiencyBonus` | `SpellcastingSummary` with per-class stats, slots by pool, cantrip and prepared limits, max preparable level (FR-001 to FR-018, FR-030 to FR-039, FR-116) |
| `CreateSpellcasting` | `CharacterID`, `ClassLevels`, per class: starting cantrips, prepared list, always-prepared set, `ExtraCantrips`, `GrantedCantrip`s, a Wizard's `Spellbook`; `map[content.SpellID]content.Spell` | the `SpellcastingState`, a rejection listing every problem, or `...ContentMissing` (FR-092, FR-093, FR-117) |
| `ChooseAdditionalSpells` | `SpellcastingState`, class, spells and cantrips to add, `map[content.SpellID]content.Spell` | new state and events, a rejection, or `...ContentMissing` (FR-089, FR-117) |
| `ChangePreparedSpells` | `SpellcastingState`, class, `PreparationTrigger`, new prepared list and/or cantrip list and/or replacement granted cantrip, `map[content.SpellID]content.Spell` | new state and events, a rejection, or `...ContentMissing` (FR-040 to FR-054, FR-090, FR-091, FR-098, FR-117) |
| `ExpendSpellSlot` | `SpellcastingState`, `SpellSlotPool`, `SlotLevel` | new state and `SpellSlotExpended`, or `NoSlotAvailable` (FR-109) |
| `RegainSpellSlot` | `SpellcastingState`, `SpellSlotPool`, `SlotLevel`, count | new state and `SpellSlotsRestored`, or `NoExpendedSlot` (FR-110) |
| `RestoreSpellSlots` | `SpellcastingState`, `RestKind` | new state and `SpellSlotsRestored` (FR-025 to FR-027) |
| `AdvanceSpellcasting` | `SpellcastingState`, new `ClassLevels` | new state, `SpellsToChoose` and `CantripsToChoose` per class, or a `SpellcastingAdvanceRejection` (FR-029, FR-047, FR-111, FR-112) |

The spell input is the content context's `Spell`, passed in the command with the caller's `map[content.SpellID]content.Spell`; a spell id named but absent from the map is a `...ContentMissing` result (FR-117). **Ports.** None new; the use cases stay synchronous and depend on content only through its exported API. The telemetry ports arrive through the `WithTelemetry` option and default to the zero-value no-ops of package `telemetry` (shared-types-and-ownership.md FR-036; observability.md §4); no global exists.

**Exports** (FR-136), from package `character`: every type that a command or result of this section names, so no consumer needs an unexported type: the use cases, their `<UseCase>Command` and `<UseCase>Result` types with variants (incl. the three `...ContentMissing`), `SpellcastingState`, the value types of section 5, the rejection value objects and kinds, and `SpellcastingEvent` with its events. Not re-exported: other specs' types (section 1). Telemetry name constants live in one file (`character/signals.go`, observability.md §5); `character/charactertest` gains no double (no port).

## 7. Observability

The pure domain files are silent; the use cases (`*_usecase.go`) emit the signals below (`doc/rules/observability.md` §2). FR-085 to FR-088 and FR-132 hold for the use cases of both specs.

- **1-2. What can go wrong:** rule rejections (normal outcomes); invalid commands (a returned error, FR-087); a missing spell in the caller's map (`content_missing`, a host or content bug, FR-117). No port, so no I/O failure, no `*srderr.PortError`, no `error.type` (errors-and-results.md FR-043).
- **6. Operator view:** rejections and successes are values of `srd5e.outcome` on one counter beside the finer `srd5e.character.result`. Span error status is set only by an invalid-command error and by a `...ContentMissing` result (FR-132; character-aggregate.md FR-071). No spell id, character id or free text is an attribute. Monitor the share of `failure` by `srd5e.character.rejection`, error-status spans and duration regressions.
- **3, 7. Diagnostics and logs:** no `INFO` or above (FR-088); spans and `rejection` values locate a failure. **4. Counted or timed:** preparation changes, slot levels, rests, per-operation counts and durations. **5. Existing instrumentation:** `operations` and `duration` stay correct; this spec adds operation values only.

Shared signals this spec adds values to: `srd5e.character.operations` and `.duration` (character-aggregate.md section 7): the operation values `calculate_spellcasting`, `create_spellcasting`, `expend_spell_slot`, `regain_spell_slot`, `advance_spellcasting` on the counter and all eight use-case names here (plus the four of the casting spec) on the histogram. It uses `srd5e.outcome` (shared-types-and-ownership.md FR-041) and `srd5e.class` (FR-042) and emits no `srd5e.character.feature_used`. No use case calls another, so no span nests and nothing is recorded twice (shared-types-and-ownership.md FR-044). Every use case has a span `srd5e.character.<operation>` (snake case of its name) with `srd5e.outcome`, `srd5e.character.result`, `srd5e.character.operation` and the attributes below. `result` values: `rejected` and `content_missing` are `failure`; `calculated`, `created`, `changed`, `expended`, `regained`, `restored`, `advanced` are `success`. A rule rejection sets no error status (FR-085).

| Use case | Span suffix | `result` values | Counter | Extra attributes |
|---|---|---|---|---|
| `CalculateSpellcasting` | `calculate_spellcasting` | `calculated` | `operations` | none |
| `CreateSpellcasting` | `create_spellcasting` | `created`, `rejected`, `content_missing` | `operations` | `rejection` (rejections only) |
| `ChooseAdditionalSpells` | `choose_additional_spells` | `changed`, `rejected`, `content_missing` | `prepared_spell_changes` | `srd5e.class`, `rejection` |
| `ChangePreparedSpells` | `change_prepared_spells` | `changed`, `rejected`, `content_missing` | `prepared_spell_changes` | `srd5e.class`, `rejection` |
| `ExpendSpellSlot` | `expend_spell_slot` | `expended`, `rejected` | `operations` | `pool`, `rejection` |
| `RegainSpellSlot` | `regain_spell_slot` | `regained`, `rejected` | `operations` | `pool`, `rejection` |
| `RestoreSpellSlots` | `restore_spell_slots` | `restored` | `spell_slots_restored` | `rest_kind` |
| `AdvanceSpellcasting` | `advance_spellcasting` | `advanced`, `rejected` | `operations` | `rejection` |

`srd5e.character.rejection` (rejections only) is the snake_case of the *first* reason in the fixed order of section 5 (`no_slot_available`, `no_expended_slot`; `CreateSpellcasting` and the preparation use cases use the preparation kinds); `srd5e.character.pool`: `spellcasting`, `pact_magic`; `srd5e.character.rest_kind`: `short`, `long`; `srd5e.class` on preparation spans and counters. All are low cardinality.

| Signal | Type | Unit | Attributes | Emitted by | Purpose | Suggested monitoring |
|---|---|---|---|---|---|---|
| `srd5e.character.<operation>` spans | Span | n/a | as above | each use case | Follow one call, including its duration | Error status is a caller bug or `content_missing`; a rejection is not an error |
| `srd5e.character.spell_slot_level` | Histogram | `{level}` | `pool` | `ExpendSpellSlot`, `CastSpell`, `CompleteCasting` (not for a cantrip, Ritual or feature-granted cast) | Slot levels spent | Dashboard only; no alert |
| `srd5e.character.prepared_spell_changes` | Counter | `{change}` | `srd5e.outcome`, `srd5e.character.result`, `srd5e.class`, `srd5e.character.rejection` (only when `rejected`) | `ChooseAdditionalSpells`, `ChangePreparedSpells` | Which limit or trigger rejects a change | A spike of `change_not_allowed_now` means the host times changes wrongly; any `content_missing` means an incomplete spell map |
| `srd5e.character.spell_slots_restored` | Counter | `{rest}` | `srd5e.outcome` (`success`), `rest_kind` | `RestoreSpellSlots` | Rest traffic | A zero rate beside casts suggests the host never reports rests; no alert |
| `srd5e.character.operations` | Counter | `{operation}` | `srd5e.character.operation`, `srd5e.outcome`, `srd5e.character.result`, `srd5e.character.rejection` (only when `rejected`) | every use case without its own counter | Owned by character-aggregate.md; this spec adds operation values | A rise in `failure` for `expend_spell_slot`, `regain_spell_slot` or `advance_spellcasting` means the host sends rejected calls |
| `srd5e.character.duration` | Histogram | `s` | `srd5e.character.operation` only (character-aggregate.md FR-103) | every use case | Owned by character-aggregate.md; this spec adds operation values | Alert on a p99 regression per operation |

These entries feed `doc/observability/signals.md`.

## 8. Test plan

Doubles: the recording doubles of `telemetry/telemetrytest` (shared-types-and-ownership.md FR-038) only; no port, so no contract suite (FR-039). Domain tests (the large majority) import no port or double and build `Spell`s with small builders. Tests are `*_test.go` files in package `character_test` beside the code (internal test files only for unexported edge logic); every test and subtest name carries its requirement id and path, and table-driven tests cite them per row (`doc/rules/testing.md` §5). The tables of this spec are the rows. SRD text transcribed for independent fixtures lives only under `testdata/srd/` (shared-types-and-ownership.md FR-033).

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-009, FR-068, FR-129 | unit | Table A (one test per row); Table B plus a Wizard 3 / Cleric 3 case for FR-005; non-casters (Fighter, Barbarian) ignored for spellcasting but counted in `TotalLevel`, reordering `ClassLevels` changes nothing (validation tests live in derived-stats.md); `SpellcastingFocusKind` has exactly seven values (L50 to L55 and L54 of the casting spec cover FR-068 and the `CastingContext` constructor). `spellcasting_class_test.go`, `spellcasting_stats_test.go`, `class_levels_use_test.go` |
| FR-010 to FR-018 | unit | Tables C, D, E, F, G, J, one test per row, band edges for F. `slot_tables_test.go`, `prepared_limits_test.go`. FR-118: `Concentration` holds a closed subject (spell or feature) and a kind; `casting` only with a spell subject (N19 to N22 of the casting spec) |
| FR-039 | unit | `multiclass_table_fixture_test.go`: independent fixtures of the Multiclass Spellcaster table (p. 26) and of each full-caster Features table in `testdata/srd/`, not imported from production code; every total level 1-20 compared with the production lookup |
| FR-019 to FR-028 | unit | `spell_slots_test.go` (levels 0, 1, 9, 10; remaining above maximum); `rest_test.go` (Warlock plus Wizard: a Short Rest restores Pact Magic only); immutability across transitions |
| FR-029, FR-111, FR-112 | unit | Table O. `level_up_test.go` |
| FR-030 to FR-038 | unit and application | Table H, `multiclass_slots_test.go`; Warlock 5 / Wizard 3 spending from each pool (L35 of the casting spec) and P19; a cantrip cast reports `ScalingLevel` (L7 of the casting spec) |
| FR-040 to FR-054 | unit | `spell_preparation_test.go`: limits at, below, above; several errors at once (FR-054); Tables I and P; Bard 2 to 3 reports `SpellsToChoose` 1; O10 to O14 for `CantripsToChoose` |
| FR-089 to FR-093, FR-096 to FR-098 | unit | Tables M and P; additions alone, replacements alone, both. `granted_cantrips_test.go` |
| FR-084 to FR-088 | application | Same input twice gives equal results per service and use case; recording `telemetrytest` doubles versus the zero-value no-op; keys pinned per use case incl. closed `rejection` values; a rejection is `failure` without error status; an invalid-command error recorded once on the span, not logged; nothing at `telemetry.LevelInfo` or above |
| FR-109, FR-110, FR-143, FR-144 | unit | Table Q. `slot_operations_test.go`: the use cases and the public transitions give equal outcomes for every Q row; `slots(pool)` reads remaining, maximum and expended (Pact pool: zero off its Slot Level; empty pool: zero); `PreparationOutcome` variants incl. content missing |
| FR-145 to FR-149 | unit | `feature_spell_ability_test.go` (`StatsForAbility` equals Table B for every ability; no `ClassSpellcasting` change); `slots_for_test.go` (H9, H13, H14: `SpellSlotPools`, `PactMagic` zero value without a Warlock level, remaining = maximum); an item-cast, a `CastSourceScroll` and a `CastSourceDuplicated` shape test; export scan (FR-136) |
| FR-116, FR-117, FR-132 | unit and application | Table R; R1, R2 assert the `Modifier` slots with `others` empty; one application test per use case taking a spell map: `failure`, `content_missing`, error status, one increment, no `rejection`, no `error.type`, no log |
| FR-126, FR-127 | unit | P20 to P22, Q8, Q9 (`ErrClassNotHeld` for a class not held, no error for an empty pool); every event implements `character.Event` and `SpellcastingEvent`, exhaustive switch |
| FR-136, architecture | architecture | Export scan (`internal/archtest`): every type named by a command or result of section 6 is exported, none generic (shared-types-and-ownership.md FR-013, FR-017); no `combat` import. |
| scenario | scenario | `scenarios/spellcasting_test.go` (package `scenarios_test`), public API only: Wizard 5 creates, prepares, rests; Ranger 4 / Sorcerer 3 (H1) shared pool; Warlock short rest; a level-up that raises the prepared limit. Cites FR-021, FR-025, FR-026, FR-031 |

- Not covered: none. The ids in section 4's "Withdrawn ids" are tested in `casting-and-concentration.md`.
- Mutation testing: the `spellcasting*.go` domain files (`doc/rules/testing.md` §8; `gomu`, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors; threshold 80); table-driven tests kill lookup mutants, boundary rows in F, J, O kill `<` versus `<=`. Use cases SHOULD be mutation tested too.

## 9. Open questions and decisions

Open: none. Two cross-spec follow-ups were settled as amendments to other specs (this spec edits none of them; it is written as if they are applied):

- Specs citing contracts that moved or changed: class-paladin.md, class-ranger.md, class-warlock.md, class-cleric.md, class-sorcerer.md, class-wizard.md re-point withdrawn ids (FR-121 to FR-124, FR-114, FR-130 and every other id in "Withdrawn ids") to `casting-and-concentration.md`, same number; class-warlock.md reports `FocusPactWeapon` and `FocusBookOfShadows` through `FeatureFocusKinds` (FR-129); class-sorcerer.md FR-035 is met by FR-047; character-aggregate.md adds the operation values of section 7; magical-effects.md's `ConcentrationRules.SaveDC` becomes `ConcentrationSaveDC.ForDamage`; combat-action-resolution.md's break transition is `ConcentrationRules.BreakConcentration`.
- Rank: `casting-and-concentration.md` imports this spec (rank 13), so it sits at rank 14 in shared-types-and-ownership.md table 4.11 (already listed there), and FR-011 of that spec names it, not this spec, for Concentration.

Package decisions flagged for review (SRD silent or ambiguous): half-caster rounding (below); `...ContentMissing` for absent spells (FR-117, FR-132). The casting-time and Concentration decisions are in `casting-and-concentration.md`.

Settled (the cited requirement or decision holds the rule):

- Half-caster rounding (FR-032, Table H4): interpretation, combined. "Half your levels (round up) in the Paladin and Ranger classes" (SRD 5.2.1, "Multiclassing", "Spell Slots", p. 25) is the second item of a list whose first item ("All your levels in the Bard, Cleric, ...") is one aggregate, and the next sentence sums the list before the table lookup, so Paladin and Ranger levels are summed, halved once, rounded up. The p. 25 example (Ranger 4 gives 2) fits both readings. Web sources are split and one unsourced answer favoured per-class rounding; they differ only for two odd levels (Paladin 1 / Ranger 1: combined 1, per-class 2). Not escalated: it is a one-line change to FR-032 and Table H4 if the SRD is corrected.
- Slot growth FR-029; Pact Magic level change FR-111; replacement counting FR-090; prepared list is "at most" the table count FR-042, FR-047, FR-089; always-prepared set kept in state FR-045, FR-046; slot operations FR-109, FR-110; event family FR-127; unheld pools FR-126; focus kinds FR-068, FR-129.
- A Shield worn without training does not stop casting (SRD 5.2.1, "Armor Training", p. 177); the character spec must not set `UntrainedArmorWorn` for a Shield.
- The caller passes the `Spell` in the command, not a `content.Source` lookup (content-spells.md FR-016). `SpellcastingState` is held beside the aggregate (shared-types-and-ownership.md FR-008) and not in `FeatureStates` (shared-types-and-ownership.md FR-022): multiclass pools fit no single class key.
- A type-only edge to the sibling rests-and-hit-dice.md for `RestKind` is allowed (shared-types-and-ownership.md FR-016); that spec must not depend on this one.
- Telemetry values and keys: shared-types-and-ownership.md FR-041, FR-042; character-aggregate.md section 7. Naming: shared-types-and-ownership.md FR-017, FR-018.
- Amendments of other specs, applied: public transitions and outcomes (FR-143, class-bard.md), slot read accessors (FR-144, class-druid.md), `SpellSlotPools` (FR-146, level-advancement-and-multiclassing.md), item-cast reading of `ItemCasterLevel.lowest` (FR-147, Q10 of magic-item-rules.md; interpretation, no slot and base cantrip tier), `CastSource.scroll` (FR-148, spell-scroll-use.md). The feats-behaviour.md request (accept `FeatGrantedSpells`) is applied as `FeatureSpellAbility` (FR-145) because importing `FeatGrantedSpells` would make rank 13 depend on rank 15 (shared-types-and-ownership.md FR-056). Follow-ups for other owners: casting-and-concentration.md `CastSpell` accepts a `FeatureSpellAbility` in place of a `SpellcastingClass` and adds `CastModeSpellScroll` (FR-106 derives `scroll`); class-bard.md FR-124 names the members of FR-143 here (its line 21 reads "spellcasting.md (FR-124)", which is its own id, not an id of this spec).
- Bard Magical Secrets is a class feature; `Spell.Classes` lets a later spec add the list.

## 10. Changelog impact

Under `Unreleased`, **Added** (not breaking, first release of this area): "Spellcasting: spell slots, Pact Magic, multiclass slots, spell save DC and attack bonus (with its `RollModifiers`), prepared spells and cantrips, and the use cases `CalculateSpellcasting`, `CreateSpellcasting`, `ChooseAdditionalSpells`, `ChangePreparedSpells`, `ExpendSpellSlot`, `RegainSpellSlot`, `RestoreSpellSlots` and `AdvanceSpellcasting`, each returning its own `<UseCase>Result`." Casting and Concentration are the entry of `casting-and-concentration.md`. New signals (section 7) go to `doc/observability/signals.md`; renaming any later is **BREAKING**.
