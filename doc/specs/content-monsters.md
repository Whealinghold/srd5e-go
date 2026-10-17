# Monsters and animals: stat-block schema, extraction and verification

| | |
|---|---|
| Status | draft |
| Context | `content` |
| Roadmap phase | Types 2; bulk data 5 |
| Depends on | `content-schema-and-decoder.md`, `content-source-port.md`, `errors-and-results.md` |

## 1. Purpose and scope

Defines how every stat block in the SRD monster chapter ("Monsters A-Z" and
"Animals", pp. 258-364; about 330 blocks, authoritative list in FR-056) becomes
bundled content readable through `Source`:

- the `Monster` content entry and its stat-block value objects, detailed enough
  that `combat` reads AC, Hit Points, Speed, abilities, defences, Challenge
  Rating and each part's notation without re-parsing prose;
- the JSON shape and monster-entry decoding, as a request to
  content-schema-and-decoder.md (a `monsters` root array registered in its
  revision table; no version number claimed, shared-types-and-ownership.md
  FR-034, FR-035). It reuses that spec's `Decoder`, `Problem`,
  `ProblemKind` and failure result (FR-017) and reports all problems;
- extraction tooling in `tools/` (shared-types-and-ownership.md FR-032): a
  stat-block text parser, an overrides file and a report;
- offline verification: completeness against the contents pages, schema
  validity and consistency. SRD-stated rules are errors (section 4.11); checks
  inferred from data or worded "typically" are advisory.

**Out of scope:** executable effects (riders stay verbatim text; the typed data of FR-098 to FR-101 is a transcription, not behaviour); "Running a
Monster" (p. 255), summoned-monster XP (p. 256), XP awards and encounter
building (pp. 192-203), Attunement for monsters (pp. 255-256); querying beyond
`Source`; stat blocks outside pp. 254-364 (spell-summoned creatures pp.
109, 131, 136, 166 and magic items pp. 222, 232 belong to their content specs,
which may reuse this schema); Lair Actions or regional effects (only the
"in Lair" counts and XP, FR-040, FR-032); display strings, localisation, any
monster not on the contents page (nothing was found to flag); `NOTICE.md`
wording and `LoadSet`.

Notice and licensing for the bundled data are owned by content-licensing-and-notice.md, which this spec cites in prose and does not depend on (content-schema-and-decoder.md and content-source-port.md are its only content dependencies, shared-types-and-ownership.md FR-047).

## 2. Source

- SRD 5.2.1, "Stat Block Overview" and "Parts of a Stat Block", pp. 254-257 (header, AC, Initiative p. 254; Hit Points, Speed, Ability Scores, Skills, defences, Gear p. 255; Senses, Languages, CR, XP, Proficiency Bonus, Traits, Actions, Attack and Saving Throw Effect Notation p. 256; Damage Notation, Multiattack, Spellcasting, Bonus Action, Reactions, Legendary Actions, Limited Usage p. 257); "Monsters A-Z" and "Animals", pp. 258-364 (blocks cited by name and page).
- SRD 5.2.1, "Rules Glossary": Blindsight p. 177, Darkvision p. 180, Tremorsense and Truesight p. 190, Burrow p. 178, Climb p. 179, Fly p. 182, Hover p. 183, Swim p. 189, Passive Perception p. 186, Telepathy p. 190, Initiative p. 184, conditions pp. 177-191; "Damage Types" p. 180; "Ability Modifiers" p. 6; "Skill List" p. 9; "The Nine Alignments" p. 21; "Unaligned Creatures" p. 22.
- SRD 5.2.1 contents pages, pp. 2-4, "Index of Stat Blocks" (Aboleth to Zombie; section starts Monsters A-Z p. 258, Animals p. 344). `doc/research/sources.md` records only the section table, so cite the contents pages for the index.
- Package design: `doc/rules/architecture.md` §2-§6, `doc/rules/observability.md` §2-§3, §7, `doc/rules/testing.md` §4-§5, §8, `doc/roadmap.md`, and `doc/specs/shared-types-and-ownership.md`.

PDF page N is printed page N. Counts (FR-032, FR-037, FR-039, FR-065, FR-072, FR-088) come from a drafting-time extraction; re-confirm against the PDF.

## 3. Vocabulary

Terms owned elsewhere are reused, not redefined (FR-089): `CreatureSize`,
`CreatureType`, `DamageType`, `ConditionKind`, `Skill`, `Alignment`,
`SpecialSense`, `MonsterID` (content-schema-and-decoder.md; shared-types-and-ownership.md
FR-005, FR-046, FR-053), `ability.Ability`, `ability.Score`, `ability.Scores`,
`ability.Modifier`, `dice.Expression` (kernel package `dice`, FR-001); `SourceCitation` (FR-004).

| SRD term | Type defined here | Notes |
|---|---|---|
| Stat block | `Monster` | p. 254 |
| Swarm | `SwarmComposition` | Swarm size plus member size (pp. 278, 361). |
| Initiative | `StatBlockInitiative` | Modifier and score as printed (p. 254). |
| Hit Points, Hit Point Dice | `HitPointsEntry` | Printed average plus `dice.Expression` on SRD die faces (p. 255). |
| Speed | `SpeedEntry`, `SpeedMode` | p. 255; Glossary pp. 178-189. |
| Ability Scores, MOD, SAVE | `AbilityBlock` | Kernel `ability.Scores` plus a save modifier per ability (p. 255). |
| Vulnerabilities, Resistances, Immunities | `DefenseEntry` | Damage types and (Immunities) conditions; no condition behaviour (pp. 255-256). |
| Senses, Passive Perception | `SenseEntry` | p. 256 |
| Languages, telepathy | `LanguagesEntry` | p. 256 |
| Challenge Rating | `ChallengeRating` | 0, 1/8, 1/4, 1/2, 1-30 (p. 256). |
| Traits, Actions, Bonus Actions, Reactions, Legendary Actions | `StatBlockPart`, `LegendaryActions` | Printed order; sections may be absent; "3 (4 in Lair)" (pp. 254-258, 327). |
| Attack, saving throw, damage notation | `StatBlockAttack`, `StatBlockAttackRange`, `EffectClause`, `StatBlockSavingThrow`, `StatBlockDamage` | pp. 256-257 |
| Spellcasting; At Will, N/Day, N/Day Each | `MonsterSpellcasting`, `SpellFrequency` | pp. 257, 305 |
| X/Day, Recharge X-Y, Recharge after rest | `PartUsage` closed | Limited Usage (p. 257). |

Proficiency Bonus is derived from CR (p. 256), not stored. No term from outside
SRD 5.2.1 is used (`doc/rules/architecture.md` §3; AGENTS.md).

## 4. Requirements

"Printed" means as in the stat block; pages are printed pages. "validation error"
is a failed `NewX` / `ParseX` precondition (`architecture.md` §6); "problem" is a
`Problem` (content-schema-and-decoder.md FR-001, FR-015). A table row with `/` alternatives is one test per alternative.

### 4.1 Identity and provenance

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `Monster` MUST implement `Entry`; its `id` MUST be a `MonsterID` (content-schema-and-decoder.md FR-114; shared-types-and-ownership.md FR-005, FR-046) derived by FR-002 and FR-093, and its `key` MUST be that id's own `key` (`Key(KindMonster, id.id)`, content-source-port.md section 3). `Monster` never builds a `Key` from a bare string. | SRD 5.2.1, "Stat Block Overview", p. 254 | happy |
| FR-002 | The `MonsterID` MUST be the printed name lower-cased, apostrophes (`'`, `’`) removed, each run of characters outside ASCII `a-z0-9` replaced by one underscore, leading/trailing underscores removed (table 4.1a). The result matches `^[a-z][a-z0-9_]*$` (content-schema-and-decoder.md FR-030; shared-types-and-ownership.md FR-007; all 330 index names are unique, at most 26 characters, within the 64-character `ID` limit of content-schema-and-decoder.md FR-143). A non-ASCII letter is a separator. Exposed as `DeriveMonsterID` (FR-093). The JSON entry carries an explicit `id` equal to this derivation (else `constraintViolation`). `NewMonster` MUST return an error wrapping `srderr.ErrInvalidInput` for an `id` differing from `DeriveMonsterID(name)`. | SRD 5.2.1, "Monsters A-Z", p. 258 | happy |
| FR-003 | `Monster` MUST return a validation error (from its constructor) a name that is empty, whitespace-only, or has leading or trailing whitespace. | architecture.md §3, §6 | unhappy (invalid input) |
| FR-004 | `Monster.source` MUST be the schema-owned `SourceCitation` (content-schema-and-decoder.md FR-115, FR-034, FR-090; `document` "SRD 5.2.1"); `Monster` defines no copy (shared-types-and-ownership.md FR-053). `source.page` MUST be the printed page on which the block begins, within 258-364 inclusive, and `source.section` "Monsters A-Z" or "Animals", else a validation error. | SRD 5.2.1, contents pages, pp. 2-4 (starts: Monsters A-Z p. 258, Animals p. 344) | unhappy (boundary) |
| FR-005 | `Monster` and every value object in it MUST be immutable; exposed collections MUST be returned as a copy views, and mutating a collection passed to a constructor afterwards MUST NOT change the instance. | architecture.md §3, §4 ("Handoffs") | unhappy (immutability) |
| FR-006 | `Monster` MUST have value equality via `Monster.Equal` (it holds slices, so it is not comparable with `==`): equal fields give equal monsters; changing any one field makes them unequal. | architecture.md §3 | happy |

Table 4.1a, id derivation (one test per row):

| Label | Printed name | Expected `MonsterID` | Source | Path |
|---|---|---|---|---|
| single word | Aboleth | `aboleth` | p. 258 | happy |
| spaces | Adult Red Dragon | `adult_red_dragon` | contents pages, pp. 2-4 | happy |
| hyphen becomes underscore | Saber-Toothed Tiger | `saber_toothed_tiger` | contents pages, pp. 2-4 | happy |
| hyphen in name | Half-Dragon | `half_dragon` | p. 297 | happy |
| apostrophe removed (curly or straight) | Will-o’-Wisp / Will-o'-Wisp | `will_o_wisp` | p. 341 | unhappy (boundary) |
| repeated separators collapse | `Giant  Rat` (two spaces) | `giant_rat` | architecture.md §3 | unhappy (boundary) |
| non-ASCII letter is a separator (synthetic) | `Zoé` | `zo` | FR-002 | unhappy (boundary) |
| id `giant_rat`, name `Giant Bat` | (constructor) | a returned validation error (`MustX` panics) | FR-002 | unhappy (invalid input) |
| no letters or digits | `' - '` | `DeriveMonsterID` returns an error wrapping `srderr.ErrInvalidInput` with `*IDError` `empty` as cause; `MustDeriveMonsterID` panics | architecture.md §6 | unhappy (invalid input) |

### 4.2 Header: Size, Creature Type, Alignment

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-008 | A `Monster` MUST hold one or more size options in printed order, no duplicates ("Medium or Small" is `[Medium, Small]`). An empty or duplicated list MUST return an error wrapping `srderr.ErrInvalidInput`. | SRD 5.2.1, "Size", p. 254; "Assassin", p. 260 | unhappy (boundary) |
| FR-009 | A `Monster` MAY hold a `SwarmComposition` naming the size of the swarm's members (for example Tiny); then its size options hold the swarm's own size (Medium or Large). A non-swarm monster MUST have none. | SRD 5.2.1, "Swarm of Crawling Claws", p. 278; "Swarm of Bats", p. 361 | happy |
| FR-011 | Descriptive tags MUST be an ordered list of zero or more non-blank, trimmed strings as printed (for example Dinosaur, Chromatic, Metallic, Devil, Demon, Goblinoid, Lycanthrope, Titan, Angel, Genie, Wizard, Cleric, Druid); blank or untrimmed tags MUST return an error wrapping `srderr.ErrInvalidInput`. Tags MUST NOT change any behaviour. | SRD 5.2.1, "Descriptive Tags", p. 254 | unhappy (invalid input) |

### 4.3 Combat highlights: Armor Class, Initiative, Hit Points, Speed

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-013 | Armor Class MUST be stored as printed and be a non-negative integer; negative MUST return an error wrapping `srderr.ErrInvalidInput`. | SRD 5.2.1, "Armor Class", p. 254 | unhappy (boundary) |
| FR-014 | `StatBlockInitiative` MUST hold the printed modifier (signed integer) and printed score (integer), neither MUST be recomputed from the other (consistency check: FR-072). | SRD 5.2.1, "Initiative", p. 254 | happy |
| FR-015 | `HitPointsEntry` MUST hold the printed average (integer, at least 1) and Hit Point Dice as a `dice.Expression`: count at least 1, die one of d4, d6, d8, d10, d12, d20 (a restriction of the monster types, not of `dice.Expression`), flat modifier any integer. A count or average below 1, or any other die, MUST return an error wrapping `srderr.ErrInvalidInput`. Average and dice are separate members with no combining member (p. 255: "don’t use both"). | SRD 5.2.1, "Hit Points", "Hit Dice by Size", p. 255 | unhappy (boundary) |
| FR-016 | `SpeedEntry` MUST hold a non-empty set of `SpeedMode`s (closed enum: walk, burrow, climb, fly, swim), a feet value of at least 0, a `hover` flag, and an optional non-blank qualifier. `hover` is valid only when the mode set contains fly. A monster MUST have at least one entry; no two entries share a mode set and qualifier. Violations MUST return an error wrapping `srderr.ErrInvalidInput`. Cases: table 4.3a. | SRD 5.2.1, "Speed", p. 255; "Hover", p. 183 | unhappy (invalid input) |
| FR-017 | An unlabelled first speed MUST be walk; labelled speeds ("Burrow", "Climb", "Fly", "Swim") map to their modes; "(hover)" sets `hover`; a parenthetical such as "(wolf form only)" or "(GM’s choice)" becomes the qualifier of the speed it follows. Also: (a) a second unlabelled speed ("Speed 30 ft., 40 ft. (wolf form only)") is a second walk entry with its own qualifier; (b) "Climb or Fly 20 ft. (GM’s choice)" is one entry with mode set {climb, fly}; (c) a qualifier on a labelled speed ("Climb 30 ft. (bear form only)") attaches to that entry. | SRD 5.2.1, "Speed", p. 255; "Werewolf", p. 339; "Werebear", p. 337; "Swarm of Insects", p. 361 | happy |

Table 4.3a, Speed (one test per row):

| Label | Printed Speed | Expected entries | Source | Path |
|---|---|---|---|---|
| walk only | Speed 25 ft. | walk 25 | "Animated Armor", p. 259 | happy |
| walk and climb | Speed 30 ft., Climb 30 ft. | walk 30; climb 30 | "Ape", p. 344 | happy |
| walk and swim | Speed 10 ft., Swim 40 ft. | walk 10; swim 40 | "Aboleth", p. 258 | happy |
| hover | Speed 10 ft., Fly 90 ft. (hover) | walk 10; fly 90 hover | "Air Elemental", p. 258 | happy |
| form qualifier | Speed 30 ft., 40 ft. (wolf form only) | walk 30; walk 40 qualifier "wolf form only" | "Werewolf", p. 339 | happy |
| alternative modes | Speed 20 ft., Climb or Fly 20 ft. (GM’s choice) | walk 20; {climb, fly} 20 qualifier "GM’s choice" | "Swarm of Insects", p. 361 | happy |
| qualifier on labelled speed | Speed 30 ft., 40 ft. (bear form only), Climb 30 ft. (bear form only) | walk 30; walk 40 "bear form only"; climb 30 "bear form only" | "Werebear", p. 337 | happy |
| zero speed value | walk 0 | accepted | p. 255 | unhappy (boundary) |
| negative speed | walk -5 | a returned validation error (`MustX` panics) | p. 255 | unhappy (invalid input) |
| hover without fly | swim 30 hover | a returned validation error (`MustX` panics) | p. 183 | unhappy (invalid input) |
| no entries | (empty) | a returned validation error (`MustX` panics) | p. 255 | unhappy (empty) |
| duplicate entry | walk 30; walk 30 | a returned validation error (`MustX` panics) | p. 255 | unhappy (invalid input) |

### 4.4 Ability scores, skills, defences, gear

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | `AbilityBlock` MUST compose the kernel's `ability.Scores` (ability-scores.md owns "exactly one `AbilityScore`, 1-30, per `Ability`"; not reimplemented here) plus one printed saving-throw modifier (signed integer) per ability. A missing ability or out-of-range score MUST return an error wrapping `srderr.ErrInvalidInput` (from `ability.Scores`); a missing save modifier MUST return an error wrapping `srderr.ErrInvalidInput` from `AbilityBlock`. | SRD 5.2.1, "Ability Scores", p. 255; "Ability Modifiers", p. 6 | unhappy (boundary) |
| FR-019 | `AbilityBlock.AbilityModifier(ability)` MUST delegate to `ability.Score.Modifier` (ability-scores.md FR-012 owns the table and its 30 score cases); no band is restated and the modifier MUST NOT be stored. | ability-scores.md FR-012; SRD 5.2.1, "Ability Modifiers", p. 6 | happy |
| FR-020 | An entry MUST report `proficientInSave` true exactly when its save modifier differs from its ability modifier ("Int 18 +4 +8" proficient, "Str 21 +5 +5" not). | SRD 5.2.1, "Ability Scores", p. 255; "Aboleth", p. 258 | happy |
| FR-021 | Skills MUST be a map from the `Skill` enum (18 skills) to a signed integer bonus; an unknown skill name MUST be an `invalidValue` problem. A duplicate skill is not detectable in decoded JSON (`encoding/json` keeps the last key, content-schema-and-decoder.md Q5), so no rule is stated. | SRD 5.2.1, "Skills", p. 255; "Skill List", p. 9 | unhappy (invalid input) |
| FR-024 | Vulnerabilities, Resistances and Immunities MUST each be an ordered list of `DefenseEntry`: a damage type with optional qualifier, a condition with optional qualifier (Immunities only), or a described entry for printed text naming neither (table 4.4a). Conditions in Vulnerabilities or Resistances MUST return an error wrapping `srderr.ErrInvalidInput`. | SRD 5.2.1, "Resistances and Vulnerabilities", "Immunities", p. 255 | unhappy (invalid input) |
| FR-025 | In Immunities, damage-type entries MUST precede condition entries as printed; a condition before a damage-type entry MUST return an error wrapping `srderr.ErrInvalidInput`. | SRD 5.2.1, "Immunities", p. 255 | unhappy (rule interaction) |
| FR-026 | `Gear` MUST be an ordered list of non-blank item names as printed; duplicates allowed; a blank name MUST return an error wrapping `srderr.ErrInvalidInput`. Absent Gear and an empty list are indistinguishable. | SRD 5.2.1, "Gear", p. 255; "Assassin", p. 260 | unhappy (empty) |

Table 4.4a, defence entries (one test per row):

| Label | Printed text | Expected | Source | Path |
|---|---|---|---|---|
| plain damage types | Resistances Bludgeoning, Lightning, Piercing, Slashing | four damage-type entries | "Air Elemental", p. 258 | happy |
| damage then conditions | Immunities Poison, Thunder; Exhaustion, Grappled, Paralyzed, Petrified, Poisoned, Prone, Restrained, Unconscious | damage entries Poison, Thunder, then the eight condition entries in that order | "Air Elemental", p. 258 | happy |
| condition qualifier | Immunities Psychic; Charmed (with Mind Blank) / Immunities Charmed (except from its vampire master) | Psychic; Charmed with qualifier "with Mind Blank" / Charmed with qualifier "except from its vampire master" | "Archmage", p. 305; "Vampire Familiar", p. 334 | happy |
| damage qualifier | Vulnerabilities Piercing damage from weapons wielded by creatures under the effect of a Bless spell | Piercing with qualifier "from weapons wielded by creatures under the effect of a Bless spell" | "Rakshasa", p. 317 | happy |
| described entry | Resistances Damage type chosen for the Draconic Origin trait below | one described entry | "Half-Dragon", p. 297 | happy |
| condition among Resistances | Resistances Poisoned | a returned validation error (`MustX` panics) | p. 255 | unhappy (invalid input) |
| order reversed | Immunities Exhaustion; Poison | a returned validation error (`MustX` panics) | p. 255 | unhappy (rule interaction) |

### 4.5 Senses, Languages, Challenge Rating

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-027 | Senses MUST hold an ordered list of `SenseEntry` (a `SpecialSense`: Blindsight, Darkvision, Tremorsense, Truesight, content-schema-and-decoder.md FR-136; range in feet at least 0; optional non-blank qualifier such as "unimpeded by magical Darkness") and a required Passive Perception integer. No special senses gives an empty list and still a Passive Perception. | SRD 5.2.1, "Senses", p. 256; Rules Glossary pp. 177, 180, 186, 190; p. 267 | happy |
| FR-028 | A `SenseEntry` with a negative range or unknown kind MUST return an error wrapping `srderr.ErrInvalidInput` (constructor) or be an `outOfRange` (range) or `invalidValue` (kind) problem (decoder). | SRD 5.2.1, "Senses", p. 256 | unhappy (invalid input) |
| FR-029 | `LanguagesEntry` MUST hold an ordered list of printed language phrases (verbatim, parentheses kept), an optional telepathy range in feet, and an optional `telepathyQualifier` (verbatim parenthetical after the range). A semicolon outside parentheses separates the language list from a telepathy clause; within the list a comma outside parentheses separates phrases, except a phrase beginning "Understands" runs to its "but can’t speak" ending (optional "them") as one phrase. A semicolon-separated part that is not telepathy is a further phrase. "None" is the empty list. Telepathy MUST NOT appear as a phrase. `telepathyQualifier` stays verbatim text and is never interpreted here: a host maps a qualifier such as the Otyugh's no-response clause to `ReplyPolicy.contactedMayNotReply` (telepathy.md FR-023). Cases: table 4.5b. | SRD 5.2.1, "Languages", p. 256; "Telepathy", p. 256, Rules Glossary p. 190; "Aboleth", p. 258; "Kraken", p. 303; "Dretch", p. 281; "Otyugh", p. 313 | happy |
| FR-030 | `ParseChallengeRating(printed string) (ChallengeRating, error)` MUST accept only the 34 spellings "0", "1/8", "1/4", "1/2" and "1" through "30" (no sign, padding or whitespace); anything else ("1/3", "31", "-1", "0.5", "01", "") MUST return an error wrapping `srderr.ErrInvalidInput` whose typed cause carries the rejected input (`MustChallengeRating` panics with the same error). It never panics for any `string`. `String()` round-trips the spelling. The JSON `challengeRating` field carries the same string. | SRD 5.2.1, "Experience Points by Challenge Rating", p. 256 | unhappy (boundary) |
| FR-097 | `ChallengeRating` MUST be comparable with `==` and ordered by `CompareChallengeRating(a, b) int` (like `cmp.Compare`) with total order 0 < 1/8 < 1/4 < 1/2 < 1 < 2 < ... < 30, expose `IsZero()` (true only for CR 0), and have value equality by `==`. This is the amendment toolbox-combat-encounter-difficulty.md FR-031 and Q-3 require. | SRD 5.2.1, "Experience Points by Challenge Rating", p. 256; toolbox-combat-encounter-difficulty.md FR-031 | happy |
| FR-031 | The Proficiency Bonus MUST be derived from CR by table 4.5a and MUST NOT be stored. `Monster.proficiencyBonus` returns a plain `int` (2 to 9); a consumer needing a kernel `Modifier` converts it (monster-combatant-setup.md FR-029). | SRD 5.2.1, "Proficiency Bonus", p. 256 | happy |
| FR-032 | A monster MUST store printed XP (integer, at least 0), and where a block prints a second figure "in lair" MUST store `experiencePointsInLair` (greater than printed XP, else a validation error). Constructor and decoder MUST NOT check XP against the CR row: that is a verification-layer check (FR-078), because a printed value differing from table 4.5a ("Archmage", p. 305: CR 12, XP 8,000) MUST still construct, decode and bundle as printed (FR-082). The four blocks printing "(N XP; PB +n)" instead of "(XP N; PB +n)" decode to the same fields. | SRD 5.2.1, "Experience Points", p. 256; "Aboleth", p. 258; "Archmage", p. 305 | unhappy (boundary) |

Table 4.5b, languages (one test per row):

| Label | Printed Languages | Expected | Source | Path |
|---|---|---|---|---|
| none | None | empty list, no telepathy | "Eagle", p. 348 | happy |
| phrase with parentheses | Primordial (Auran) | one phrase `Primordial (Auran)` | "Air Elemental", p. 258 | happy |
| language and telepathy | Deep Speech; telepathy 120 ft. | phrase Deep Speech; telepathy 120 | "Aboleth", p. 258 | happy |
| understands but can’t speak | Understands Abyssal, Celestial, Infernal, and Primordial but can’t speak; telepathy 120 ft. | one phrase "Understands Abyssal, Celestial, Infernal, and Primordial but can’t speak"; telepathy 120 | "Kraken", p. 303 | happy |
| telepathy qualifier | Abyssal; telepathy 60 ft. (works only with creatures that understand Abyssal) / Otyugh; telepathy 120 ft. (doesn’t allow the receiving creature to respond telepathically) | phrase Abyssal, telepathy 60, qualifier "works only with creatures that understand Abyssal" / phrase Otyugh, telepathy 120, qualifier "doesn’t allow the receiving creature to respond telepathically" | "Dretch", p. 281; "Otyugh", p. 313 | happy |
| understands after semicolon | Celestial; understands Common, Elvish, and Sylvan but can’t speak them | phrases: Celestial; "understands Common, Elvish, and Sylvan but can’t speak them" | "Giant Elk", p. 351 | unhappy (rule interaction) |

Table 4.5a, Challenge Rating (one test per CR; XP and PB are the expected lookups):

| CR | XP | PB | CR | XP | PB | CR | XP | PB |
|---|---|---|---|---|---|---|---|---|
| 0 | 0 or 10 | +2 | 9 | 5,000 | +4 | 21 | 33,000 | +7 |
| 1/8 | 25 | +2 | 10 | 5,900 | +4 | 22 | 41,000 | +7 |
| 1/4 | 50 | +2 | 11 | 7,200 | +4 | 23 | 50,000 | +7 |
| 1/2 | 100 | +2 | 12 | 8,400 | +4 | 24 | 62,000 | +7 |
| 1 | 200 | +2 | 13 | 10,000 | +5 | 25 | 75,000 | +8 |
| 2 | 450 | +2 | 14 | 11,500 | +5 | 26 | 90,000 | +8 |
| 3 | 700 | +2 | 15 | 13,000 | +5 | 27 | 105,000 | +8 |
| 4 | 1,100 | +2 | 16 | 15,000 | +5 | 28 | 120,000 | +8 |
| 5 | 1,800 | +3 | 17 | 18,000 | +6 | 29 | 135,000 | +9 |
| 6 | 2,300 | +3 | 18 | 20,000 | +6 | 30 | 155,000 | +9 |
| 7 | 2,900 | +3 | 19 | 22,000 | +6 |  |  |  |
| 8 | 3,900 | +3 | 20 | 25,000 | +6 |  |  |  |

Source: SRD 5.2.1, "Experience Points by Challenge Rating" and "Proficiency Bonus
by Challenge Rating", p. 256. Path: happy; PB band edges (CR 4/5, 8/9, 12/13, 16/17,
20/21, 24/25, 28/29) unhappy (boundary); CR 1/3, 31, -1, 0.5 a validation error (invalid input).

### 4.6 Traits, Actions, Bonus Actions, Reactions, Legendary Actions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-033 | A `Monster` MUST hold five ordered `StatBlockPart` lists (Traits, Actions, Bonus Actions, Reactions, Legendary Actions), each possibly empty, in printed order. | SRD 5.2.1, "Stat Block Overview", p. 254; "Traits" p. 256; "Bonus Action", "Reactions", "Legendary Actions", p. 257 | happy |
| FR-034 | A `StatBlockPart` MUST have a non-blank name, a `PartUsage`, and non-blank verbatim text; blank name or text MUST return an error wrapping `srderr.ErrInvalidInput`. Which trait a part is (recognition) belongs to monster-traits.md; the model stores no structured trait parameters (monster-traits.md Q-7), so no schema revision. | SRD 5.2.1, "Traits", p. 256 | unhappy (invalid input) |
| FR-035 | `PartUsage` MUST be closed: `PartUnlimited`, `PartPerDay(count, inLairCount?)`, `PartRecharge(low, high)`, `PartRechargeAfterRest`, or `PartUsageOther(text)` for any other parenthetical. `PartPerDay` counts MUST be at least 1; `PartRecharge` MUST satisfy 1 <= low <= high <= 6; "Recharge 6" is `PartRecharge(6, 6)`. The parser accepts the printed en dash ("Recharge 4–6") and a hyphen as the same separator and stores integers. Cases: table 4.6a. | SRD 5.2.1, "Limited Usage", p. 257; "Air Elemental", pp. 258-259 (Whirlwind, p. 259) | unhappy (boundary) |
| FR-036 | An action named exactly `Multiattack` MUST be kept as an ordinary part with its prose and exposed as `Monster.multiattack`; at most one per monster (a second MUST return an error wrapping `srderr.ErrInvalidInput`). The prose is never interpreted by the model or decoder; the attacks it names are the optional structured `plan` (FR-098, FR-099). | SRD 5.2.1, "Multiattack", p. 257 | happy |
| FR-037 | An attack part MUST expose `StatBlockAttack`: a `StatBlockAttackKind` (melee, ranged, meleeOrRanged), the printed signed bonus, an optional qualifier (the parenthetical after the bonus, such as "with Advantage if the target is Grappled by the ankheg"), reach in feet when present, and a `StatBlockAttackRange` (normal feet, optional long feet) when present: "range 80/320 ft." is 80 and 320; "range 60 ft." (18 ranged attacks) is 60 and no long range. "reach 5 feet"/"reach 5 ft." and "range 120 feet"/"range 120 ft." MUST decode to the same value. Reach MUST be at least 0, the range normal at least 1, and a long range greater than the normal; violations return an error wrapping `srderr.ErrInvalidInput` (decoder `outOfRange` at the number; `constraintViolation` at `.../range/long` for the ordering). Cases: table 4.6b. | SRD 5.2.1, "Attack Notation", p. 256; "Bite", "Ankheg", p. 259; "Priest", p. 316; "Djinni", p. 280 | happy |
| FR-038 | A part's `EffectClause` list MUST preserve the printed label as an `EffectClauseLabel` (closed enum) with verbatim text: `Hit`, `Miss`, `HitOrMiss` (attacks; "Miss:" is allowed by the notation but not printed in the monster chapter) and `Failure`, `Success`, `FailureOrSuccess`, `FirstFailure`, `SecondFailure`, `SubsequentFailures`, `FailureByFiveOrMore` (saves). A part may hold several failure labels ("Basilisk"). A label twice in one part MUST return an error wrapping `srderr.ErrInvalidInput` and is an `invalidValue` problem in the decoder; a label outside the enum is an `invalidValue` problem in the decoder only. | SRD 5.2.1, "Attack Notation", "Saving Throw Effect Notation", p. 256; "Basilisk", p. 262; "Death Dog", p. 279; "Homunculus", p. 299 | unhappy (invalid input) |
| FR-039 | A `StatBlockSavingThrow` MUST hold the save's ability, the printed DC (integer; below 1 MUST return an error wrapping `srderr.ErrInvalidInput`) and an optional target description ("one creature the aboleth can see within 30 feet"; empty means implied, 14 printed saves have none). Printed text before the notation ("Response—Wisdom Saving Throw: DC 15, the triggering creature.") is kept verbatim as optional `prefix`. Attack-roll and saving-throw notation MAY co-occur in one part ("Death Dog" Bite). The text of a Success clause (including "Half damage only") stays verbatim and may carry a `SaveSuccessRider` (FR-100). | SRD 5.2.1, "Saving Throw Effect Notation", p. 256; "Dominate Mind", p. 258; "Death Dog", p. 279; "Chain Devil", p. 273 | happy |
| FR-040 | `LegendaryActions` MUST hold the printed use count (at least 1), an optional "in Lair" count (at least 1 when present) and a non-empty ordered action list; zero actions or zero uses MUST return an error wrapping `srderr.ErrInvalidInput`. "3 (4 in Lair)" gives uses 3, in-lair 4; "3." gives uses 3 and none. | SRD 5.2.1, "Legendary Actions", p. 257; "Aboleth", p. 258; p. 327 | unhappy (boundary) |
| FR-041 | A Reaction MUST keep its trigger inside its verbatim text; the model MUST NOT have a separate trigger field. | SRD 5.2.1, "Reactions", p. 257 | happy |
| FR-098 | The `Multiattack` part MAY carry `plan`, a `MultiattackPlan`: one or more `MultiattackVariant`s. A variant has an optional `requiresUsed` (printed name of a part the monster used this turn), one or more `AttackSlotGroup(count >= 1, parts)` (non-empty set of part names), zero or more `AdditionalUse(options)` (one or more part names) and zero or more `ReplacementRule(count >= 1, scope, options, limit)`: `scope` is closed `ReplacementScope` (`anyAttack`, `oneAttackPart(name)`), `options` are one or more `ReplacementOption(part, optional spell {name, castAtLevel 1-9})`, `limit` is `ReplacementLimit` (`once`, `anyNumber`). Groups of one variant have pairwise disjoint part sets; `oneAttackPart` names a part of some group of that variant; no name repeats within one part set or options list; a variant with `requiresUsed` needs at least one sibling variant without it (so a Multiattack is always available). `plan` on a part not named `Multiattack` is invalid. Violations MUST return an error wrapping `srderr.ErrInvalidInput`. These are the same invariants as monster-stat-block-actions.md FR-011, which consumes the plan; the model and decoder never read the prose, which stays verbatim beside it. Cases: table 4.6c. | SRD 5.2.1, "Multiattack", p. 257; "Clay Golem", p. 274; "Chimera", p. 273; "Young Brass Dragon", p. 268; "Doppelganger", p. 280 | unhappy (invalid input) |
| FR-099 | A `Monster` MUST return a validation error (from its constructor) a plan whose part names do not resolve: each group, additional-use, `oneAttackPart` and replacement-option name MUST be exactly one Actions part other than `Multiattack`; `requiresUsed` MUST be exactly one part of Actions, Bonus Actions or Reactions (the Clay Golem's Hasten is a Bonus Action). Absent, in another section, or shared by two parts in the allowed sections is unresolved. A replacement spell name is not checked here (monster-stat-block-actions.md FR-021 resolves it against the Spellcasting groups). | SRD 5.2.1, "Multiattack", p. 257; "Clay Golem", p. 274 | unhappy (missing content) |
| FR-100 | A `Success` `EffectClause` MAY carry one `SaveSuccessRider`, closed: `HalfDamageOnly`, `HalfDamage`, `ImmuneToPartForHours(hours >= 1)` (the part is the one holding the clause: "this harpy’s Luring Song"). A rider on any other label MUST return an error wrapping `srderr.ErrInvalidInput`; hours below 1 likewise. The clause text stays verbatim; the rider is data set by the extractor (FR-102) and carries no behaviour here. `HalfDamageOnly` is the p. 256 phrase; `HalfDamage` is other text beginning "Half damage" (the plain half-damage success of p. 16). | SRD 5.2.1, "Saving Throw Effect Notation", p. 256; "Half Damage", p. 16; "Harpy", p. 297; "Cloaker", p. 274 | unhappy (boundary) |

Tables 4.6a, 4.6b, 4.7a: "Printed" is extractor input (FR-062 to FR-070); the
decoder sees only structured JSON (section 5). A row expecting a decode problem
is tested at the "JSON equivalent", and the extractor reports the printed form
as `unparsedBlock`. No notation-specific problem kind exists (FR-049).

Table 4.6a, usage notation (one test per row):

| Label | Printed parenthetical | Expected | Source | Path |
|---|---|---|---|---|
| per day | (3/Day) | `PartPerDay(3)` | p. 257 | happy |
| per day with lair | (3/Day, or 4/Day in Lair) | `PartPerDay(3, inLair: 4)` | "Aboleth", p. 258 | happy |
| recharge range | (Recharge 4–6) | `PartRecharge(4, 6)` | "Air Elemental", pp. 258-259 | happy |
| recharge range, hyphen | (Recharge 4-6) | `PartRecharge(4, 6)` | p. 257 | unhappy (boundary) |
| single-value recharge | (Recharge 6) | `PartRecharge(6, 6)` | "Ankheg", p. 259 | unhappy (boundary) |
| rest recharge | (Recharge after a Short or Long Rest) | `PartRechargeAfterRest` | "Cloaker", p. 274 | happy |
| other condition | (1/Day; Requires Soul Bag) | `PartUsageOther("1/Day; Requires Soul Bag")` | "Night Hag", p. 311 | unhappy (rule interaction) |
| no parenthetical | Amphibious | `PartUnlimited` | "Aboleth", p. 258 | happy |
| zero uses | (0/Day) | `unparsedBlock`; JSON equivalent: `outOfRange` at `<part>/usage/count` | p. 257 | unhappy (boundary) |
| recharge low above high | (Recharge 5–4) | `unparsedBlock`; JSON equivalent: `constraintViolation` at `<part>/usage` | p. 257 | unhappy (boundary) |
| recharge outside die | (Recharge 7) / (Recharge 0–6) | `unparsedBlock`; JSON equivalent: `outOfRange` at `<part>/usage/high` / `<part>/usage/low` | p. 257 | unhappy (boundary) |

Table 4.6b, attack notation (one test per row):

| Label | Printed | Expected `StatBlockAttack` | Source | Path |
|---|---|---|---|---|
| melee | Melee Attack Roll: +9, reach 15 ft. | melee, +9, reach 15 | "Aboleth", p. 258 | happy |
| ranged with range | Ranged Attack Roll: +7, range 80/320 ft. | ranged, +7, range 80/320 | "Assassin", p. 260 | happy |
| melee or ranged | Melee or Ranged Attack Roll: +5, reach 10 ft. or range 30/120 ft. | melee-or-ranged, +5, reach 10, range 30/120 | Javelin of "Bugbear Stalker", p. 271 | happy |
| single range | Ranged Attack Roll: +4, range 60 ft. | ranged, +4, range 60, no long range | "Priest", p. 316 | unhappy (boundary) |
| attack qualifier | Melee Attack Roll: +5 (with Advantage if the target is Grappled by the ankheg), reach 5 ft. | melee, +5, reach 5, qualifier "with Advantage if the target is Grappled by the ankheg" | "Ankheg", p. 259 | happy |
| "feet" spelling | Melee Attack Roll: +4, reach 5 feet / Ranged Attack Roll: +9, range 120 feet | same value as "reach 5 ft." / "range 120 ft." | "Eagle", p. 348; "Djinni", p. 280 | unhappy (boundary) |
| attack and save in one part | Melee Attack Roll: +4, reach 5 ft. Hit: 4 (1d4 + 2) Piercing damage. ... Constitution Saving Throw: DC 12. First Failure: ... | attack and saving throw both present; clauses Hit, FirstFailure, SubsequentFailures | "Death Dog", p. 279 | happy |
| save without target | Constitution Saving Throw: DC 12. Failure: ... | DC 12, empty target description | "Homunculus", p. 299 | unhappy (minimal) |
| save with prefix | Response—Wisdom Saving Throw: DC 15, the triggering creature. Failure: ... | prefix "Response—", target "the triggering creature" | "Chain Devil", p. 273 | happy |
| failure labels | First Failure: ... Second Failure: ... / Failure: ... Failure by 5 or More: ... | clauses FirstFailure, SecondFailure / Failure, FailureByFiveOrMore | "Basilisk", p. 262; "Homunculus", p. 299 | happy |
| negative bonus | Melee Attack Roll: -1, reach 5 ft. | bonus -1 | p. 256 | unhappy (boundary) |
| missing bonus | Melee Attack Roll: , reach 5 ft. | `unparsedBlock` (FR-062); JSON equivalent: `missingField` at `<part>/attack/bonus` | p. 256 | unhappy (invalid input) |

Table 4.6c, Multiattack plan, success rider and damage flag (FR-098 to FR-101; "Printed" is extractor input, FR-102; expected problems are decoder verdicts, paths relative to the Multiattack part's `plan` or to `<part>/clauses/<i>`; one test per row):

| Label | Printed or JSON equivalent | Expected | Source | Path |
|---|---|---|---|---|
| any combination | three attacks, using Foreleg or Poison Burst in any combination | one group, count 3, parts {Foreleg, Poison Burst} | "Drider", p. 281 | happy |
| replace named part | one Ram, one Bite, one Claw; replace the Claw with Fire Breath if available | three groups of count 1; replacement count 1, `oneAttackPart(Claw)`, [Fire Breath], `once` | "Chimera", p. 273 | happy |
| additional use | two Slam attacks and uses Unsettling Visage if available | group count 2 {Slam}; one `AdditionalUse` [Unsettling Visage] | "Doppelganger", p. 280 | happy |
| conditional variant | two Slam attacks, or three if it used Hasten this turn | variant 1 count 2; variant 2 `requiresUsed` Hasten (a Bonus Action), count 3 | "Clay Golem", p. 274 | happy |
| part or spell option | replace one attack with a use of (A) Sleep Breath or (B) Spellcasting to cast Scorching Ray | `anyAttack`, count 1, options Sleep Breath; Spellcasting with spell Scorching Ray | "Young Brass Dragon", p. 268 | happy |
| replace two | replace two attacks with a use of Sleep Breath | `ReplacementRule` count 2 | p. 268 | unhappy (boundary) |
| any attack, repeatable | two Bite attacks; replace any attack with a use of Poisonous Spittle | count 1, `anyAttack`, limit `anyNumber` | "Guardian Naga", p. 296 | happy |
| no variants | `variants: []` | `constraintViolation` at `/variants` | FR-098 | unhappy (empty) |
| conditional without sibling | one variant with `requiresUsed` | `constraintViolation` at `/variants/0/requiresUsed` | FR-098 | unhappy (rule interaction) |
| overlapping groups | groups {Bite} and {Bite, Claw} in one variant | `constraintViolation` at `/variants/<v>/groups/1` | FR-098 | unhappy (rule interaction) |
| repeated name | parts [Bite, Bite] | `duplicate` at `.../parts/1` | FR-098 | unhappy (invalid input) |
| unresolved name | group naming a Bonus Action / unknown part / a name shared by two Actions parts | `constraintViolation` at the name | FR-099 | unhappy (missing content) |
| replace part not in a group | `oneAttackPart(Tail)`, no group holds Tail | `constraintViolation` at `.../scope/part` | FR-098 | unhappy (rule interaction) |
| zero count / spell level | group count 0 / `castAtLevel` 10 | `outOfRange` at `.../count` / `.../spell/castAtLevel` | FR-098 | unhappy (boundary) |
| plan on other part | `plan` on `Bite` | `constraintViolation` at `<part>/plan` | FR-098 | unhappy (invalid input) |
| half damage only | Success: Half damage only. | `HalfDamageOnly` | "Steam Mephit", p. 308 | happy |
| plain half damage | Success: Half damage, and the target moves to an unoccupied space within 5 feet | `HalfDamage` | "Gelatinous Cube", p. 287 | happy |
| immunity | Success: The target is immune to this harpy’s Luring Song for 24 hours. | `ImmuneToPartForHours(24)` | "Harpy", p. 297 | happy |
| hours below 1 / rider on Failure | `hours: 0` / rider on a `Failure` clause | `outOfRange` at `.../successRider/hours` / `constraintViolation` at `.../successRider` | FR-100 | unhappy (boundary) |
| underwater flag | Bite ... plus 7 (2d6) Fire damage. Being underwater doesn’t grant Resistance to this Fire damage. | Fire instance flagged; Piercing instance not | "Dragon Turtle", p. 281 | happy |
| flag in a Failure or Success clause | Failure: 56 (16d6) Fire damage. ... Failure or Success: Being underwater doesn’t grant Resistance to this Fire damage. | the Failure clause’s Fire instance flagged | "Dragon Turtle", p. 281; "Steam Mephit", p. 308 | happy |
| flag on non-Fire | flag with Cold damage | a validation error; decoder `constraintViolation` at `.../damage/<k>/underwaterResistanceDenied` | FR-101 | unhappy (invalid input) |

### 4.7 Damage and Spellcasting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-042 | A `StatBlockDamage` MUST hold the printed fixed number, an optional `dice.Expression` (faces as in FR-043) and a `DamageTypeRef`: `SpecificDamageType` (a `DamageType`) or `DescribedDamageType` (verbatim text, for example "damage of the type chosen for the Draconic Origin trait"). A fixed number with no dice ("1 Slashing damage") is valid. The number is stored as printed, not recomputed. Each clause (FR-038) exposes its damage instances in printed order ("5 (1d6 + 2) Piercing damage plus 3 (1d6) Poison damage" yields two). Cases: table 4.7a. | SRD 5.2.1, "Damage Notation", p. 257; "Awakened Shrub", p. 260; "Half-Dragon", p. 297 | happy |
| FR-043 | A damage die count below 1, faces other than 4, 6, 8, 10, 12, 20, a negative number, or an unknown damage type word MUST return an error wrapping `srderr.ErrInvalidInput` (constructor) or be `invalidValue` (unknown word) or `outOfRange` (count, faces, number) in the decoder. | SRD 5.2.1, "Damage Notation", p. 257; "Damage Types", p. 180 | unhappy (invalid input) |
| FR-044 | A part named `Spellcasting` MUST expose a `MonsterSpellcasting`: the ability (one of six), the spell save DC and spell attack bonus when printed, any printed components statement (such as "requiring no Material components") verbatim, and one or more spell groups; zero groups return an error wrapping `srderr.ErrInvalidInput` (decoder `constraintViolation` at `.../spellcasting/groups`). | SRD 5.2.1, "Spellcasting", p. 257; "Archmage", p. 305 | happy |
| FR-045 | A spell group MUST have a `SpellFrequency`: `SpellFrequencyAtWill`, `SpellFrequencyPerDay(n)` ("N/Day:") or `SpellFrequencyPerDayEach(n)` ("N/Day Each:"), n at least 1, and an ordered spell list. A group with no spells or n below 1 MUST return an error wrapping `srderr.ErrInvalidInput`. | SRD 5.2.1, "Spellcasting", p. 257; "Archmage", p. 305; "Priest", p. 316 | unhappy (boundary) |
| FR-046 | A spell entry MUST hold the spell name verbatim and an optional note (parenthetical such as "included in AC" or "cast before combat"). A note "(level N version)" also sets `castAtLevel` to N (1 to 9); otherwise `castAtLevel` is absent (lowest possible level). | SRD 5.2.1, "Spellcasting", p. 257; "Archmage", p. 305 | unhappy (boundary) |
| FR-047 | Spell entries MUST be split on commas outside parentheses only: "Shapechange (Beast or Humanoid form only, no Temporary Hit Points gained from the spell, and no Concentration or Temporary Hit Points required to maintain the spell)" is one entry with the whole parenthetical as note. | SRD 5.2.1, "Spellcasting", p. 257; "Adult Brass Dragon", pp. 268-269 (entry on p. 269) | unhappy (rule interaction) |
| FR-101 | A `StatBlockDamage` MUST carry a flag `underwaterResistanceDenied` (default false; JSON `underwaterResistanceDenied: true`, absent when false) that is true only on a `SpecificDamageType` of Fire; true on any other type MUST return an error wrapping `srderr.ErrInvalidInput`. It records "Being underwater doesn’t grant Resistance to this Fire damage" (the sentence stays verbatim in the clause text) so a host can pass `UnderwaterResistanceGrant.withheldByEffect` to mounted-and-underwater-combat.md FR-044; nothing here applies it. The model reads no prose: the extractor sets it (FR-102). | SRD 5.2.1, "Dragon Turtle", p. 281; "Steam Mephit", p. 308; "Fire Resistance" (Underwater Combat), p. 16 | unhappy (rule interaction) |

Table 4.7a, damage notation (one test per row):

| Label | Printed | Expected instances | Source | Path |
|---|---|---|---|---|
| number and dice | 5 (1d6 + 2) Bludgeoning damage | 5; 1d6+2; Bludgeoning | "Animated Armor", p. 259 | happy |
| no flat modifier | 10 (3d6) Psychic damage | 10; 3d6+0; Psychic | "Aboleth", p. 258 | happy |
| fixed only | 1 Slashing damage | 1; no dice; Slashing | "Awakened Shrub", p. 260 | unhappy (minimal) |
| two instances | 7 (1d6 + 4) Piercing damage plus 17 (5d6) Poison damage | two instances in order | "Assassin", p. 260 | happy |
| split across lines | 10 (2d6 + 3) with the "+ 3" wrapped onto the next line | one instance 10; 2d6+3 | "Animated Rug of Smothering", p. 259 | unhappy (rule interaction) |
| described type | 7 (2d6) damage of the type chosen for the Draconic Origin trait | described type | "Half-Dragon", p. 297 | happy |
| unknown type | 4 (1d4) Arcane damage | `unparsedBlock`; JSON equivalent: `invalidValue` at `.../damage/<k>/damageType` | p. 180 | unhappy (invalid input) |
| bad dice | 4 (0d4) Fire damage / 4 (1d7) Fire damage | `unparsedBlock`; JSON equivalent: `outOfRange` at `.../dice/count` / `.../dice/faces` | p. 257 | unhappy (boundary) |
| negative flat modifier (U+2212, FR-064; also Hit Points, p. 302) | 1 (1d4 − 1) Piercing damage | 1; 1d4-1; Piercing | "Baboon", p. 345 | happy |

### 4.8 Decoder and JSON shape

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | Monsters MUST be decoded by `JSONDecoder` (content-schema-and-decoder.md FR-001, FR-078) from the `monsters` root array; this spec adds `MonsterDecoder`, an internal collaborator, not a second public decoder or result type. Every problem is a `Problem` with path `/monsters/<index>/...`, all problems reported (ordering per content-schema-and-decoder.md FR-016). The domain MUST NOT see a raw `map[string]any`. | content-schema-and-decoder.md FR-001, FR-015, FR-016; architecture.md §5, §6 | happy |
| FR-049 | Monster problems MUST use `ProblemKind` (content-schema-and-decoder.md FR-015): absent required field `missingField`; wrong JSON type `wrongType`; number or count out of range `outOfRange`; word outside a closed set, unparsable id, unknown skill/damage type/sense kind `invalidValue`; unknown key `unknownField`; later duplicate id or repeated member of a set array (content-schema-and-decoder.md FR-026) `duplicate`; id differing from FR-002 and any rule among members of one entry (table 4.8a) `constraintViolation`. This spec adds **no** kind: the ten kinds of its FR-015 and the ten `problem_kind` values of its FR-100 stand, and `malformedNotation` is dropped. The decoder never sees printed notation, so unparsable notation is an extractor result (`unparsedBlock`, FR-062). Table 4.8a gives the verdict for every constructor-only rule for the agreement test (content-schema-and-decoder.md FR-094, section 8); an unlisted rule follows its first row. Each kind used MUST be reachable from a test and carry the JSON path. | content-schema-and-decoder.md FR-015, FR-026, FR-094, FR-100; testing.md §4 ("Missing or bad content") | unhappy (invalid input) |
| FR-050 | The decoder MUST NOT coerce or guess: a creature type, size, alignment, skill, damage type, condition, ability or sense outside the closed sets (FR-089; skills FR-021; sense kinds FR-027) MUST be `invalidValue`, including words not in SRD 5.2.1 ("not_an_alignment", "not_a_creature_type"). | SRD 5.2.1, "Creature Type" p. 254; "The Nine Alignments", p. 21; AGENTS.md | unhappy (invalid input) |
| FR-051 | An absent required field MUST be `missingField`, a wrong JSON type `wrongType`. Required: `id`, `name`, `source` (`document`, `section`, `page`; content-schema-and-decoder.md FR-090), `sizes`, `type`, `alignment`, `armorClass`, `initiative`, `hitPoints`, `speeds`, all six abilities, `senses` (with `passivePerception`), `languages`, `challengeRating`, `experiencePoints`. Absent optional fields (skills, defences, gear, parts, legendary actions, swarm, tags) decode to the empty value ("if a monster lacks those details, entries for them don't appear"). | SRD 5.2.1, "Stat Block Overview", p. 254 | unhappy (boundary) |
| FR-052 | The smallest valid monster (one size, type, alignment, AC, Initiative, Hit Points, one Speed, six abilities, Passive Perception only, empty `languages`, CR 0 with XP 0, no parts) MUST decode with every optional collection empty (returned as copies). | SRD 5.2.1, "Stat Block Overview", p. 254 | unhappy (empty) |
| FR-053 | `monsters` is a root array registered as a row in the revision table of content-schema-and-decoder.md (shared-types-and-ownership.md FR-034, FR-035); a monster entry carries no `schemaVersion`; a document with a non-current version is `unsupportedVersion` (content-schema-and-decoder.md FR-005 to FR-007) and `monsters` in a document that does not list it is `unknownField` at `/monsters` (content-schema-and-decoder.md FR-010). An unknown key inside a monster MUST be `unknownField` (strict decoding; the data is generated, so an unknown key is a bug and consumers cannot extend entries in this version). | content-schema-and-decoder.md FR-008 to FR-010; shared-types-and-ownership.md FR-034; testing.md §4 | unhappy (invalid input) |
| FR-054 | Decoding the same JSON twice MUST give equal monsters and equal failures in the same order, without modifying the input. | architecture.md §3; testing.md §4 | unhappy (determinism) |
| FR-055 | A canonical encoder (with the extractor, outside the decoder's public API) re-encoding a decoded monster and decoding again MUST give an equal monster, and its output MUST be byte-identical every run (stable key order, no whitespace variation). | architecture.md §3; testing.md §4 | happy |

Table 4.8a, decoder verdict for each constructor rule (content-schema-and-decoder.md
FR-094: each row fed to the decoder as JSON expecting the problem, and to the
constructor expecting a validation error; paths relative to `/monsters/<index>`; one
test per row):

| Label | Constructor rule | Decoder problem (kind at path) | Path |
|---|---|---|---|
| default mapping | numeric limit broken | `outOfRange` at the number; blank or malformed text, or a word outside a closed set, `invalidValue` at the value; a rule between members of one entry `constraintViolation` at the later offending member; a repeated member of a set array `duplicate` at the later one | unhappy (invalid input) |
| blank name | FR-003 | `invalidValue` at `/name` | unhappy (invalid input) |
| id mismatch | FR-002 | `constraintViolation` at `/id` | unhappy (invalid input) |
| page out of range | FR-004 | `outOfRange` at `/source/page`; section outside the two names `invalidValue` at `/source/section` | unhappy (boundary) |
| empty sizes | FR-008 | `constraintViolation` at `/sizes` | unhappy (empty) |
| duplicate sizes | FR-008 | `duplicate` at `/sizes/<later>` | unhappy (invalid input) |
| blank tag | FR-011 | `invalidValue` at `/tags/<i>` | unhappy (invalid input) |
| negative AC | FR-013 | `outOfRange` at `/armorClass` | unhappy (boundary) |
| hit point dice | FR-015 | `outOfRange` at `/hitPoints/average`, `/hitPoints/count` or `/hitPoints/faces` | unhappy (boundary) |
| negative speed | FR-016 | `outOfRange` at `/speeds/<i>/feet` | unhappy (boundary) |
| speed rules | FR-016 | hover without fly `constraintViolation` at `/speeds/<i>/hover`; repeated mode in one entry `duplicate` at `/speeds/<i>/modes/<later>`; duplicate entry `constraintViolation` at `/speeds/<later>`; no entries `constraintViolation` at `/speeds` | unhappy (invalid input, empty) |
| ability block | FR-018 | `missingField` at `/abilities/<ability>` or `/abilities/<ability>/save`; `outOfRange` at `/abilities/<ability>/score` | unhappy (boundary) |
| condition under Resistances or Vulnerabilities | FR-024 | `invalidValue` at `/resistances/<i>` (or `/vulnerabilities/<i>`) | unhappy (invalid input) |
| condition before damage type in Immunities | FR-025 | `constraintViolation` at `/immunities/<i>` (the damage entry that follows a condition) | unhappy (rule interaction) |
| blank gear item | FR-026 | `invalidValue` at `/gear/<i>` | unhappy (invalid input) |
| sense | FR-028 | `outOfRange` at `/senses/special/<i>/feet`; `invalidValue` at `/senses/special/<i>/kind` | unhappy (invalid input) |
| invalid Challenge Rating | FR-030 | `wrongType` for a non-string; otherwise `invalidValue` at `/challengeRating` | unhappy (boundary) |
| XP | FR-032 | negative `outOfRange` at `/experiencePoints`; lair XP not above printed `constraintViolation` at `/experiencePointsInLair` | unhappy (boundary) |
| blank part name or text | FR-034 | `invalidValue` at `<part list>/<i>/name` or `.../text` | unhappy (invalid input) |
| usage count or recharge bounds | FR-035 | `outOfRange` at `<part>/usage/count`, `.../low` or `.../high`; low above high `constraintViolation` at `<part>/usage` | unhappy (boundary) |
| second Multiattack | FR-036 | `constraintViolation` at `/actions/<later>/name` | unhappy (rule interaction) |
| Multiattack plan | FR-098, FR-099 | the verdicts of table 4.6c at `/actions/<i>/plan/...`: counts and `castAtLevel` `outOfRange`; a repeated name `duplicate`; empty variants, conditional variant without sibling, overlapping groups, `oneAttackPart` outside the groups, unresolved name, `plan` on another part `constraintViolation` | unhappy (rule interaction) |
| success rider / underwater flag | FR-100, FR-101 | hours below 1 `outOfRange` at `<part>/clauses/<i>/successRider/hours`; unknown rider kind `invalidValue`; rider on a non-Success clause `constraintViolation` at `.../successRider`; flag on a non-Fire type `constraintViolation` at `.../damage/<k>/underwaterResistanceDenied` | unhappy (rule interaction) |
| repeated clause label | FR-038 | `invalidValue` at `<part>/clauses/<later>/label` | unhappy (invalid input) |
| save DC below 1 | FR-039 | `outOfRange` at `<part>/savingThrow/dc` | unhappy (boundary) |
| empty legendary actions | FR-040 | `constraintViolation` at `/legendaryActions/actions`; zero uses `outOfRange` at `/legendaryActions/uses` | unhappy (empty) |
| damage dice or number | FR-043 | `outOfRange` at `.../damage/<k>/dice/count`, `.../faces` or `.../number`; unknown damage type `invalidValue` | unhappy (boundary) |
| empty spell group or frequency below 1 | FR-045 | `constraintViolation` at `.../spellcasting/groups/<g>/spells`; `outOfRange` at `.../frequency/count` | unhappy (boundary) |
| `castAtLevel` out of range | FR-046 | `outOfRange` at `.../spells/<s>/castAtLevel` | unhappy (boundary) |

### 4.9 Bundled set and completeness

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-056 | The bundled set MUST contain exactly the stat blocks in the "Index of Stat Blocks", pp. 2-4, for "Monsters A-Z" and "Animals" (pp. 258-364): none missing, none extra. The committed fixture `monster_index` (printed name, page and section, each copied from the index) is the authority and the set's size MUST equal its size. The section cannot be derived from the page: p. 344 holds "Ogre Zombie" (Monsters A-Z) before the "Animals" heading. | SRD 5.2.1, contents pages, pp. 2-4 | unhappy (missing content) |
| FR-057 | Every bundled monster's `source.page`, `source.section` and name MUST equal the index values exactly (case, hyphens, apostrophes). "Ogre Zombie" (p. 344, "Monsters A-Z") and "Allosaurus" (p. 344, "Animals") are the test rows for the shared page. | SRD 5.2.1, contents pages, pp. 2-4 | unhappy (missing content) |
| FR-058 | Every bundled `MonsterID` MUST be unique within `KindMonster`; within the single monsters document (FR-060) this is content-schema-and-decoder.md FR-031 and FR-098 unchanged: the later entry with a used valid id gets `duplicate` at its own `/monsters/<index>/id`. Those rules are per document, so FR-060 requires exactly one monsters document; a duplicate across documents would surface only as a validation error from `ContentCatalogue` when `BundledSource` is built (content-source-port.md FR-021). | content-schema-and-decoder.md FR-031, FR-098; content-source-port.md FR-021; architecture.md §5; testing.md §4 | unhappy (duplicate) |
| FR-059 | The access point MUST be `data.BundledSource` (package `content/data`; owned by content-source-port.md; shared-types-and-ownership.md FR-031, FR-047). It decodes the embedded monster JSON (FR-060) through the same `Decoder` as host-supplied JSON; there is no second loader. A bad bundle is a packaging error: the constructor returns a `BundledContentInvalid` error carrying the decoder's problems (content-source-port.md FR-059 to FR-062, FR-064); `data.Files()` is the only embedded access (content-schema-and-decoder.md FR-122); no consumer-visible signal exists. Guards: the every-bundled-file-decodes test (content-schema-and-decoder.md FR-089, FR-104) and the layout check (FR-091). Seeded with the bundled monsters it MUST return found for any bundled id, not-found for an unknown id, not-found for a bundled monster id under another `Kind`, and exactly the `monster_index` entries ordered by `MonsterID` from `ListKind(ctx, KindMonster)`. Over the real embedded data the suite is `contenttest.RunSourceContract` (content-source-port.md FR-058, FR-047; exported from `content/contenttest`, shared-types-and-ownership.md FR-039), with the decoded monsters as the expected entries. | content-source-port.md FR-011, FR-047, FR-058 to FR-062, FR-064; shared-types-and-ownership.md FR-031, FR-039, FR-047; architecture.md §4 | happy |
| FR-060 | The source of truth MUST be exactly one monsters document, `content/data/monsters.json`, with the root of FR-053, embedded directly with `go:embed` (shared-types-and-ownership.md FR-030); the core reads no file at runtime. `NOTICE.md` covers exactly `content/data/` and `testdata/srd/` (FR-033; content-licensing-and-notice.md). Extraction inputs and report sit in the extractor's own `testdata/srd/` directory, outside `content/data/` (so the every-file-decodes walk, content-schema-and-decoder.md FR-089, skips them): `tools/extract_monsters/testdata/srd/monster_overrides.json` (FR-066) and `tools/extract_monsters/testdata/srd/monster_extraction_report.txt` (FR-069). The data MUST contain only SRD monster content and no monster absent from the index (FR-056). | AGENTS.md ("Licensing and source material"); shared-types-and-ownership.md FR-030, FR-033 | unhappy (invalid input) |

### 4.10 Extraction

The extractor (pure Go text parser plus a thin command-line wrapper reading the
PDF text) lives in `tools/extract_monsters` (its own module), is never imported by the root module and is not public API
(shared-types-and-ownership.md FR-032); it may import the module's public packages.
The PDF is never stored in the repository (`doc/research/sources.md`).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-061 | The extractor MUST read each page's two columns separately, left before right; raw page order interleaves columns (observed pp. 258-364) and MUST be rejected. A header line (name, then size/type/alignment line) not followed by the `AC`, `Initiative`, `HP`, `Speed` sequence within its next 8 lines marks the page `columnOrderSuspect`; given interleaved text the parser MUST report it rather than produce JSON. | SRD 5.2.1, "Monsters A-Z", pp. 258-364 (layout) | unhappy (invalid input) |
| FR-062 | The parser MUST split text into blocks at the header line (size, optional "or" size or "Swarm of Tiny", type, tags, alignment) and take the name from the line above. The block count MUST be reported; a block with no parsable header or no `CR` line MUST be `unparsedBlock` with its page and first lines. | SRD 5.2.1, "Stat Block Overview", p. 254 | unhappy (invalid input) |
| FR-063 | The parser MUST accept an ability grid fragmented by layout ("Str 5 / Int 1 ... -3 -3 / -5 -5", "Giant Centipede", p. 349; "Con22 +6 +6", "Adult White Dragon", p. 340) and report per block that it did (`repairedTile`). A block whose six abilities cannot be recovered is `unparsedBlock`. A printed sign omission is not fragmentation and is kept as printed (FR-082, "Young White Dragon", p. 340, "Int 6 −2 2"). | SRD 5.2.1, "Giant Centipede", p. 349; "Adult White Dragon", p. 340 | unhappy (invalid input) |
| FR-064 | The parser MUST convert U+2212 to a numeric minus; accept the en dash (U+2013) in "Recharge 4–6" and "Recharge 5–6" (FR-088); keep em dashes in notation names ("Response—Wisdom Saving Throw"), curly apostrophes and punctuation as printed; and remove a soft hyphen (U+00AD) inside a word. | SRD 5.2.1, "Initiative", p. 254; "Will-o’-Wisp", p. 341; "Chain Devil", p. 273 | happy |
| FR-065 | The parser MUST join wrapped prose lines with one space and resolve a line-end hyphen deterministically: (a) next line starts lowercase and the fragment is not on the compound allowlist: join without the hyphen ("sur-" + "prise" is "surprise"); (b) fragment on the allowlist (true compounds such as "5-foot-", "spider-"): join without a space, keep the hyphen; (c) next line starts with a capital or digit: ambiguous, listed as `lineEndHyphen` with page and context and resolved only by an overrides entry (FR-066). The allowlist lives in the overrides file; the report counts joins per rule. An unresolved (c) case fails the run. About 418 line-end hyphens occur on pp. 258-364, most (a). | SRD 5.2.1, wrapped prose, pp. 258-364 | unhappy (rule interaction) |
| FR-066 | The overrides file `data/extraction/monster_overrides.json` (input only; holds SRD text, FR-060) MUST be the only way to differ from a mechanical parse. Each override carries the monster name, JSON path, the value the parser produced, the value to use, the printed page and a reason. A stale override (produced value no longer matches, or matches nothing) MUST fail the run (`staleOverride`). | doc/rules/testing.md §5 ("Keep citations true") | unhappy (state error) |
| FR-067 | The extractor MUST check each printed ability modifier against the Ability Modifiers table and the printed Proficiency Bonus against the CR table, reporting `printedMismatch` with page and values, and keep printed XP, Initiative and other values as printed (FR-014, FR-032). | SRD 5.2.1, "Ability Modifiers", p. 6; "Proficiency Bonus", p. 256 | unhappy (rule interaction) |
| FR-068 | Extraction MUST be deterministic: same text and overrides give byte-identical JSON and report (ordering by name, no timestamps). | testing.md §6 | unhappy (determinism) |
| FR-069 | The extractor MUST write only through an `ExtractionOutput` interface (`Write(relativePath, text string) error`, declared in `tools/`; the command supplies `os`, tests an in-memory one). The only files written are `content/data/monsters.json` and the report `tools/extract_monsters/testdata/srd/monster_extraction_report.txt` (committed, deterministic, FR-068); a write to any other path (another `*.json` under `content/data/`, the overrides file), an image or a page-text dump MUST be refused with an error and write nothing. | `doc/research/sources.md`; AGENTS.md; shared-types-and-ownership.md FR-032 | unhappy (state error) |
| FR-070 | The parser MUST normalise equivalent printings to one value: "reach 5 feet"/"reach 5 ft."; "range 120 feet"/"range 120 ft."; "(N XP; PB +n)"/"(XP N; PB +n)"; a trailing tab after "AC n"; "AC n Initiative +m (k)" on one line. It MUST NOT otherwise rewrite prose. | SRD 5.2.1, pp. 258-364; "Djinni", p. 280 | happy |
| FR-083 | The parser MUST remove page footers ("258 System Reference Document 5.2.1") and the running head ("Monsters A–Z") wherever they appear, including mid-sentence at a column break (p. 258 "Consume Memories"; p. 295 "Green Hag" Spellcasting), and rejoin the sentence. No text is lost or duplicated. | SRD 5.2.1, pp. 258, 295 (layout) | unhappy (rule interaction) |
| FR-084 | The parser MUST recognise group headings between blocks ("Animated Objects" p. 259, "Awakened Plants" p. 260, "Bugbears" p. 271) and not append them to the previous block's prose. A heading is a line not followed by a header line that follows a block's last section; recognised headings are listed in the report. | SRD 5.2.1, pp. 259, 260, 271 | unhappy (rule interaction) |
| FR-085 | The printed name appears twice in the layout ("Aboleth" "Aboleth"); the parser MUST collapse an immediately repeated name line. | SRD 5.2.1, "Aboleth", p. 258 | unhappy (rule interaction) |
| FR-086 | A block continuing into the next column or page ("Giant Centipede", p. 349 to p. 350) MUST be read as one block with `source.page` the header's page. | SRD 5.2.1, "Giant Centipede", pp. 349-350 | unhappy (rule interaction) |
| FR-087 | Each of FR-083 to FR-086 and FR-088 MUST have at least one fixture row: p. 258 footer/head mid-sentence ("Consume Memories"), p. 295 ("Green Hag"), headings on pp. 259, 260, 271, the doubled "Aboleth", the block spanning pp. 349-350, and "Recharge 4–6" with an en dash, each asserting the expected JSON value or report line. | doc/rules/testing.md §5 | unhappy (rule interaction) |
| FR-088 | The parser MUST accept both en dash and hyphen in a recharge range and store integers (FR-035); the en dash is the printed form (72 ranges on pp. 258-364: 5 of "Recharge 4–6", 67 of "Recharge 5–6"). | SRD 5.2.1, "Air Elemental", pp. 258-259 | happy |
| FR-102 | The extractor MUST fill the structured data of FR-098, FR-100, FR-101 from printed phrases only: `plan` from the Multiattack prose by a deterministic sentence-pattern recogniser (table 4.6c rows are its fixtures); `successRider` `HalfDamageOnly` from "Half damage only", else `HalfDamage` from a Success text beginning "Half damage", `ImmuneToPartForHours(n)` from "immune to this X’s Y for [the next] n hours" (n digits); `underwaterResistanceDenied` on every damage instance of type T in the same part for the exact sentence "Being underwater doesn’t grant Resistance to this T damage." A Multiattack the recogniser does not match is `unplannedMultiattack` (page, name, text) and fails the run unless an overrides entry (FR-066) supplies the plan; every bundled Multiattack part MUST carry a plan (data-level test; the plan is optional in the schema, not in the bundle). A near miss (the sentence naming a non-Fire type, a rider phrase outside a Success clause) is `unparsedBlock`. The report lists each plan beside its prose for review; no mechanical prose-versus-plan check exists, so the plan is a reviewed transcription. | SRD 5.2.1, "Multiattack", p. 257; "Saving Throw Effect Notation", p. 256; pp. 268, 273, 274, 280, 281, 296, 308 | unhappy (rule interaction) |

### 4.11 Verification of the bundled set

Offline checks over the committed data (no PDF): pure functions from one
`Monster` (plus a name set where stated) to findings, in
`monster_checks_test.go` (a test file, not shipped), unit-tested with hand-built
monsters, plus one data-level test over the bundled set. **Errors** (SRD-stated
rules): FR-072 (pp. 184, 254), FR-073 (p. 255), FR-074 (p. 255), FR-075 (p. 186),
FR-078 (p. 256). **Advisory** (report lines only): FR-096 ("typically"), FR-076,
FR-077 ("usually", p. 257), FR-079. An error not in the recorded exceptions list
fails the suite; an exception names the monster, check, printed value, page and
reason, and is itself tested (a stale one fails). The list covers errors only.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-071 | Every bundled monster MUST round-trip. This is a data-level test over (JSON text of one entry, decoded `Monster`) pairs, not a check function, because it needs the raw text and the canonical encoder (FR-055): the entry decodes without problems and re-encodes byte-identically. | architecture.md §5 | happy |
| FR-072 | Error: Initiative score MUST equal 10 plus the Initiative modifier (the score replaces a roll, p. 254; "Your Initiative score equals 10 plus your Dexterity modifier", Rules Glossary p. 184; the modifier is Dexterity plus extras such as Proficiency Bonus, p. 254). 328 of 330 blocks agree; the two others are exceptions (FR-082). | SRD 5.2.1, "Initiative", Rules Glossary p. 184, and p. 254 | unhappy (rule interaction) |
| FR-073 | Error: the Hit Points flat modifier MUST equal Hit Point Dice count times the Constitution modifier, and the printed average MUST equal `dice.Expression.Average` (count times (faces + 1) / 2 plus the modifier, rounded down; the Hit Dice by Size table gives 2½, 3½, 4½, 5½, 6½, 10½ per die). | SRD 5.2.1, "Hit Points", "Hit Dice by Size", p. 255 | unhappy (rule interaction) |
| FR-074 | Error: each saving-throw modifier MUST equal the ability modifier or the ability modifier plus the Proficiency Bonus. | SRD 5.2.1, "Ability Scores", p. 255; "Proficiency Bonus", p. 256 | unhappy (rule interaction) |
| FR-075 | Error: Passive Perception MUST equal 10 plus the Perception skill bonus when listed, else 10 plus the Wisdom modifier. | SRD 5.2.1, "Senses", p. 256; "Passive Perception", p. 186 | unhappy (rule interaction) |
| FR-076 | Advisory: each attack bonus is expected to equal Proficiency Bonus plus one ability modifier, and each save DC 8 plus Proficiency Bonus plus one ability modifier. Neither is stated for monsters (the "8 plus" form appears only for character features and equipment, pp. 30, 50, 63, 84, 90, 94; p. 256 gives only the notation), so differences are report lines. | SRD 5.2.1, "Attack Notation", "Saving Throw Effect Notation", p. 256 (notation only) | unhappy (rule interaction) |
| FR-077 | Advisory: each damage instance's number is expected to equal `dice.Expression.Average` of its dice; the SRD gives it alongside the dice ("usually", p. 257) without stating it is the average, so differences are report lines. | SRD 5.2.1, "Damage Notation", p. 257 | unhappy (rule interaction) |
| FR-078 | Error: printed XP MUST be allowed by the CR row (table 4.5a; where FR-032's check lives, with the exceptions list), and printed lair XP MUST equal the standard XP of the next CR up. | SRD 5.2.1, "Experience Points by Challenge Rating", p. 256; "Aboleth", p. 258 | unhappy (rule interaction) |
| FR-079 | Advisory: skill bonuses SHOULD equal the ability modifier plus Proficiency Bonus ("other modifiers might apply"); differences are report lines. | SRD 5.2.1, "Skills", p. 255 | unhappy (rule interaction) |
| FR-080 | Report-only: `unresolvedNames(monster, spellNames, equipmentNames)` returns one line (`unresolvedSpell` or `unresolvedGear`, with monster name, page, name) per Spellcasting spell not in `spellNames` and Gear item not in `equipmentNames`; with a name set not supplied it returns one `skipped` line for that kind. It never fails the suite. | SRD 5.2.1, "Spellcasting", p. 257; "Gear", p. 255 | unhappy (missing content) |
| FR-081 | The unit tests of the error checks MUST include a deliberately corrupted hand-built monster for each of FR-072, FR-073, FR-074, FR-075 and FR-078 that makes the check return a finding. | testing.md §8 ("A test that cannot fail is a defect") | unhappy (invalid input) |
| FR-082 | The recorded exceptions list MUST contain these printed disagreements, kept as printed and not corrected: "Gray Ooze" p. 293, Initiative -2 (13) (FR-072); "Invisible Stalker" p. 301, Initiative +7 (22) (FR-072); "Archmage" p. 305, XP 8,000 at CR 12 (table 4.5a: 8,400) (FR-078); "Young White Dragon" p. 340, "Int 6 −2 2", save 2 being neither -2 nor -2 plus Proficiency Bonus (FR-074; kept as printed, no override, so FR-020 reports `proficientInSave` true for its Int). Differences from the advisory FR-076 are not exceptions but expected report lines: "Adult Bronze Dragon" p. 270 (spell save DC 17, also disagreeing with its +10 to hit), "Sphinx of Valor" p. 328 (DC 16), "Ancient White Dragon" p. 341 (DC 20), "Swarm of Ravens" p. 362 (Cacophony, Wisdom Saving Throw DC 10, which no PB +2 and modifier combination gives). | SRD 5.2.1, pages as listed | unhappy (rule interaction) |
| FR-096 | Advisory: the Hit Point die is expected to be the die of one of the monster's size options (Hit Dice by Size); p. 255 says only "typically", so a difference is a report line. | SRD 5.2.1, "Hit Points", "Hit Dice by Size", p. 255 | unhappy (rule interaction) |

### 4.12 Ownership, delivery and fixtures

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-089 | `Monster` fields MUST use the vocabulary types owned by `content` and MUST NOT define or export a copy: `CreatureSize` (FR-008, FR-009), `CreatureType` (14), `DamageType` (13), `ConditionKind` (15), `Skill`, `Alignment` (10), `SpecialSense` (4, FR-027) and `MonsterID`, all owned by content-schema-and-decoder.md (FR-105 to FR-110, FR-114, FR-128, FR-136, FR-137; shared-types-and-ownership.md FR-046, FR-053). A test asserts only that the monster accepts every value of each type and that the module declares each once; the owners test the value lists. | shared-types-and-ownership.md FR-005, FR-006, FR-013, FR-046; SRD 5.2.1, pp. 180, 254, 255, 177-191 | happy |
| FR-090 | No package of the root module MAY import `tools/` (a nested module), and the parser, overrides loader, canonical encoder and `ExtractionOutput` MUST NOT be reachable from any public package. `Monster` and its value objects depend only on the kernel packages and `content`. | shared-types-and-ownership.md FR-013, FR-032 | unhappy (forbidden edge) |
| FR-091 | The embedded monster data MUST be the committed `content/data/monsters.json` itself (no generated copy, content-schema-and-decoder.md FR-120); the layout check (content-schema-and-decoder.md FR-104, architecture.md check A12) MUST fail when `content/data/` holds any file other than `.json` files and the embed Go file, monsters included. | shared-types-and-ownership.md FR-030, FR-032 | unhappy (drift) |
| FR-092 | SRD text in this spec's tests MUST live only under `testdata/srd/` (covered by `NOTICE.md`): the `monster_index` fixture, the parser text fixtures of FR-087, and every verbatim printed phrase used as an input row of tables 4.3a, 4.4a, 4.5b, 4.6a, 4.6b, 4.7a. Each fixture is a file under `content/testdata/srd/` read by the test through `os.ReadFile` or `embed` (testing.md §6) and the test names it by a constant. A table row refers to its phrase by a named constant. Hand-built `Monster` builders carry no SRD text beyond names and numbers a case needs. | shared-types-and-ownership.md FR-033 | happy |
| FR-093 | The FR-002 derivation MUST be a pure function in package `content`: `DeriveMonsterID(name string) (MonsterID, error)` (returns an error wrapping `srderr.ErrInvalidInput`, with the `*IDError` of content-schema-and-decoder.md FR-143 as typed cause, when the name derives to nothing or to an invalid id; `MustDeriveMonsterID` panics with that error). `MonsterID`, its parser, `Key()` and `KeyedID` are owned by content-schema-and-decoder.md FR-114, FR-128; conversion to a `Key` is `MonsterID.Key()`. The decoder uses `DeriveMonsterID` for the `id` field (FR-002, FR-049), so constructor and decoder cannot disagree. Cases: table 4.1a. | content-schema-and-decoder.md FR-114, FR-128; content-schema-and-decoder.md FR-144; shared-types-and-ownership.md FR-005, FR-007 | happy |
| FR-095 | The domain types expose their limits as exported constants in a documented `MonsterLimits` value (page range 258-364, Hit Point Dice faces, allowed Challenge Ratings, castAtLevel 1-9, recharge die 1-6, ...), exported from package `content`. Constructors and decoder both use them, with no numeric literal for a limit that has a constant (content-schema-and-decoder.md FR-094). | content-schema-and-decoder.md FR-094; architecture.md §6 | happy |

### Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-007 | Withdrawn: moved to content-schema-and-decoder.md (FR-109) |
| FR-010 | Withdrawn: moved to content-schema-and-decoder.md (FR-108) |
| FR-012 | Withdrawn: moved to content-schema-and-decoder.md (FR-137) |
| FR-022 | Withdrawn: moved to content-schema-and-decoder.md (FR-107) |
| FR-023 | Withdrawn: moved to content-schema-and-decoder.md (FR-110) |
| FR-094 | Withdrawn: moved to content-schema-and-decoder.md (FR-110) |

## 5. Model

`Monster` is an immutable content entry in the monster files of package `content`,
identified by its `MonsterID` (FR-001, FR-093); not an aggregate. Fields are
those of the JSON shape below (identity, header, combat highlights, abilities,
other details, parts); `source` is a `SourceCitation` (FR-004), not a local copy. Derived accessors: `ProficiencyBonus()`, `Multiattack()`,
`AbilityModifier(a)`, `ProficientInSave(a)`, `Spellcasting()`; nothing
derived is stored. Reused types (section 3, FR-089) are exported once, by their
owners (a second declaration would collide, architecture.md §2); `ConditionEffects`
is owned by conditions.md. Exported names carry the stat-block or monster concept
(AST scan, architecture.md §9).

| Group | Types defined here |
|---|---|
| Entry and header | `Monster`, `MonsterLimits`, `DeriveMonsterID`, `ChallengeRating`, `ParseChallengeRating`, `SwarmComposition` |
| Combat highlights | `StatBlockInitiative`, `HitPointsEntry`, `SpeedEntry`, `SpeedMode` (walk, burrow, climb, fly, swim) |
| Abilities and details | `AbilityBlock`, `DefenseEntry` (closed: `DamageTypeDefense`, `ConditionDefense`, `DescribedDefense`), `SenseEntry`, `LanguagesEntry` |
| Parts | `StatBlockPart`, `PartUsage` (closed: `PartUnlimited`, `PartPerDay`, `PartRecharge`, `PartRechargeAfterRest`, `PartUsageOther`), `LegendaryActions` |
| Attack and save notation | `StatBlockAttack`, `StatBlockAttackKind` (melee, ranged, meleeOrRanged), `StatBlockAttackRange`, `StatBlockSavingThrow`, `EffectClause`, `EffectClauseLabel` (Hit, Miss, HitOrMiss, Failure, Success, FailureOrSuccess, FirstFailure, SecondFailure, SubsequentFailures, FailureByFiveOrMore), `SaveSuccessRider` (closed: `HalfDamageOnly`, `HalfDamage`, `ImmuneToPartForHours`) |
| Multiattack plan | `MultiattackPlan`, `MultiattackVariant`, `AttackSlotGroup`, `AdditionalUse`, `ReplacementRule`, `ReplacementOption`, `ReplacementScope` (closed: anyAttack, `oneAttackPart`), `ReplacementLimit` (once, anyNumber) |
| Damage | `StatBlockDamage` (with `underwaterResistanceDenied`), `DamageTypeRef` (closed: `SpecificDamageType`, `DescribedDamageType`) |
| Spellcasting | `MonsterSpellcasting`, `MonsterSpellGroup`, `MonsterSpellEntry`, `SpellFrequency` (closed: `SpellFrequencyAtWill`, `SpellFrequencyPerDay`, `SpellFrequencyPerDayEach`) |

Generic names (`DamageInstance`, `Size`, `Spellcasting`, `Recharge`, `Other`, ...) are not exported.

**Invariants.** Each section 4 rule is enforced by the named type's constructor
(a validation error). The decoder checks the same ranges first, with the same
`MonsterLimits` constants (FR-095), to report every problem; table 4.8a and the
agreement test (section 8) enforce the correspondence.

| Failure | Mechanism |
|---|---|
| Invalid constructor argument (FR-002, 003, 004, 008, 011, 013, 015, 016, 018, 024, 025, 026, 028, 030, 032 (negative XP, lair XP not above printed XP; XP is not checked against the CR row), 034, 035, 036, 037, 038, 039, 040, 043, 044, 045, 046, 098, 099, 100, 101) | a returned validation error (`MustX` panics) |
| Untrusted JSON that fails decoding (FR-048 to FR-053) | Decoder error joining `Problem`s (never a panic) |
| Lookup miss or source failure | The port's not-found variant / a `*srderr.PortError` (content-source-port.md FR-015, FR-016, FR-017) |

**JSON shape (`monsters` root array).** One object per monster, canonical key
order: `id` (FR-002), `name`, `source {document, section, page}`, `sizes`,
`swarmOf?`, `type`, `tags`, `alignment`, `armorClass`, `initiative {modifier,
score}`, `hitPoints {average, count, faces, modifier}`, `speeds [{modes, feet,
hover?, qualifier?}]`, `abilities {str, dex, con, int, wis, cha: {score, save}}`,
`skills {name: bonus}`, `vulnerabilities`, `resistances`, `immunities` (entries
`{damageType|condition|text, qualifier?}`), `gear`, `senses {special: [{kind,
feet, qualifier?}], passivePerception}`, `languages {phrases, telepathyFeet?,
telepathyQualifier?}`, `challengeRating` (string "0", "1/8", ... "30"),
`experiencePoints`, `experiencePointsInLair?`, `traits`, `actions`,
`bonusActions`, `reactions`, `legendaryActions {uses, usesInLair?, actions}`.
Each part: `{name, usage, text, attack?, savingThrow?, clauses?, damage?,
spellcasting?, plan?}`, all structured, never notation strings: `usage` is `{kind:
"unlimited" | "per_day" | "recharge" | "recharge_after_rest" | "other", count?,
inLairCount?, low?, high?, text?}`; `attack` is `{kind, bonus, qualifier?,
reach?, range {normal, long?}?}`; `savingThrow` is `{ability, dc, target?,
prefix?}`; `clauses` is `[{label, text, successRider? {kind: "half_damage_only" | "half_damage" | "immune_to_part", hours?}}]`; `damage` is `[{number, dice {count,
faces, modifier}?, damageType | text, underwaterResistanceDenied?}]`; `spellcasting` is `{ability, dc?,
attackBonus?, components?, groups [{frequency {kind, count?}, spells [{name,
note?, castAtLevel?}]}]}`; `plan` (Multiattack part only) is `{variants [{requiresUsed?, groups [{count, parts}], additionalUses [{options}], replacements [{count, scope {kind: "any_attack" | "one_attack_part", part?}, options [{part, spell {name, castAtLevel?}?}], limit: "once" | "any_number"}]}]}`. Omitted optional keys mean "entry does not appear".

Parser, overrides loader, canonical encoder and `ExtractionOutput` live in `tools/extract_monsters`
(FR-090). The unexported monster decoder (FR-048), in the `decode*.go` files of package `content`, is the only
monster-entry decoding code. Verification (FR-071 to FR-082): test-file checks.

## 6. Interface

- **No new use case or port.** Consumers use `Source` (content-source-port.md): `Lookup(ctx, Key)` returns the lookup result (callers type-switch to `Monster`; FR-007, FR-015, FR-016, Q14), `ListKind` returns all entries; failure is a `*srderr.PortError` (FR-017).
- **Decoder.** Monsters decode inside the synchronous `Decoder.Decode` (content-schema-and-decoder.md FR-001, FR-078); the monster decoder is unexported (FR-048). Problems are returned, never panicked.
- **Bundled set.** `data.BundledSource` from package `content/data` (shared-types-and-ownership.md FR-031) serves monsters (FR-059, FR-060, FR-091).
- **Exports.** Package `content` exports `Monster` and the section 5 types (no use cases); `content/data` adds nothing; `tools/` is not part of the module. Nothing in the package uses `os`.

## 7. Observability

Domain and decoder code emit nothing (`doc/rules/observability.md` §3, §7). Failure modes: bad bundled data, duplicate id, unsupported version, source I/O error, extraction errors (offline, in `tools/`). Nothing is counted or timed here (no clock; decoding is in-memory), so no timer is warranted. No signal is added; no problem kind is added (FR-049), so the ten `problem_kind` values of content-schema-and-decoder.md FR-100 stand.

| Signal | Change | Purpose | Suggested monitoring |
|---|---|---|---|
| `srd5e.content.entries_loaded` | attribute `srd5e.content.kind` gains `monster` (FR-101) | entries loaded per kind | none |
| `srd5e.content.decode_errors` | none | host-supplied monster JSON failing to decode | alert at 1 for host-supplied JSON; none for bundled |

- **Operator signal:** host-supplied monster JSON loaded through `LoadSet` records `srd5e.content.load`, `.loads` and `.decode_errors` (content-schema-and-decoder.md FR-077, FR-082, FR-100, FR-101; shared-types-and-ownership.md FR-041). Bundled monsters decode inside `BundledSource` (FR-059), emitting nothing; a bad bundle returns a `BundledContentInvalid` error (content-source-port.md FR-061), guarded by content-schema-and-decoder.md FR-089, FR-104 and the layout check (FR-091).
- **Diagnosis:** problem path and kind in the failure result; the monster id stays out of attributes. Not-found is the consumer's result, so no alert. The 330 entries MUST NOT each log at load (observability.md §7).

## 8. Test plan

No doubles are needed. FR-059 uses the real `BundledSource` and `contenttest.RunSourceContract` (content-source-port.md FR-058, FR-047; `content/contenttest`, shared-types-and-ownership.md FR-039). Fixtures (files under `testdata/srd/`, FR-092): `monster_index`, hand-built `Monster` builders, parser text from the pages cited in FR-061 to FR-070.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-006, FR-093 | unit | `monster_test.go` (`package content_test`); page boundaries 257, 258, 364, 365; immutability test mutates the passed-in slice and a returned slice; table 4.1a one test per row for `DeriveMonsterID` and `MustDeriveMonsterID` (FR-093). |
| FR-008, FR-009, FR-011, FR-013 to FR-017 | unit | Size, swarm, tag cases; AC -1/0; HP count 0/1, each die, average 0/1, negative modifier; table 4.3a. |
| FR-018 to FR-021, FR-024 to FR-026 | unit | Score boundaries 0, 1, 30, 31; one delegation test for FR-019; save proficiency true/false; unknown skill `invalidValue`; table 4.4a. |
| FR-027 to FR-032, FR-097 | unit | Empty senses, "None" languages, table 4.5b; table 4.5a plus invalid-CR rows (`ParseChallengeRating` returns an error, `MustChallengeRating` panics for the same inputs; `FuzzParseChallengeRating` never panics); ordering over all 34 values, `IsZero` only for CR 0; negative XP and lair XP not above printed XP return errors; Archmage XP 8,000 at CR 12 constructs and decodes. |
| FR-033 to FR-041, FR-098 to FR-100 | unit | Tables 4.6a, 4.6b, 4.6c; Multiattack recognised, duplicate rejected; empty legendary section rejected; each plan invariant and each name-resolution case; rider on each label. |
| FR-042 to FR-047, FR-101 | unit | Table 4.7a (Baboon negative modifier); flag on Fire, Cold, a described type; each frequency label; `castAtLevel` 1, 9, 0, 10; comma inside parentheses. |
| FR-048 to FR-055, FR-058 | unit | `decode_monster_test.go`: one test per problem kind used (FR-049) and per row of table 4.8a; three independent problems all reported in content-schema-and-decoder.md FR-016 order; minimal monster; non-current `schemaVersion` is `unsupportedVersion`; `monsters` absent from the root listing is `unknownField`; FR-058 two-entry synthetic document gives one `duplicate`. Encoder round-trip and byte-identical output (FR-055) in `tools/`. |
| FR-049, FR-095 | adapter | Agreement test (content-schema-and-decoder.md FR-094): every invalid row of tables 4.1a, 4.3a, 4.4a, 4.5a, 4.5b, 4.6a, 4.6b, 4.6c, 4.7a, 4.8a goes to the decoder (as JSON equivalent, expecting the kind and path of table 4.8a) and the constructor (expecting a validation error); valid rows to both (expecting acceptance). A lint-style test asserts the decoder uses `MonsterLimits`, with no numeric literal for a limit that has one. |
| FR-056, FR-057, FR-060 | scenario | `scenarios/`, public API only: `data.BundledSource` compared with `monster_index` (the p. 344 rows assert `source.section`). |
| FR-059 | contract | Real embedded data through `contenttest.RunSourceContract` (decoded monsters as the expected entries), plus found/not-found/wrong-kind and `ListKind(ctx, KindMonster)` order. |
| FR-061 to FR-068, FR-070, FR-083 to FR-088, FR-102 | unit | `tools/`; fixtures from pp. 258, 259, 260, 271, 295, 340, 349-350 and a deliberately interleaved page; each hyphen rule (a), (b), (c); stale override; unresolved line-end hyphen; each table 4.6c printed row; `unplannedMultiattack`. |
| FR-069 | unit | `tools/extract_monsters`: a run against an in-memory `ExtractionOutput` writes only `monsters.json` and the report; any other write is refused with an error. |
| FR-072 to FR-081, FR-096 | unit | `monster_checks_test.go` check functions with hand-built monsters (conforming, boundary, corrupted per FR-081); FR-076, FR-077, FR-079, FR-080, FR-096 return report lines, not failures. |
| FR-071 to FR-082 | scenario | One data-level test runs every check over the bundled set (FR-071 over (JSON text, `Monster`) pairs; every Multiattack part carries a plan, FR-102); the exceptions list is asserted entry by entry (a stale one fails); FR-076 differences are asserted as report lines. |
| FR-089 | unit + architecture | `monster_test.go`: `Monster` accepts every value of each published type; `internal/archtest`: the module declares each once (shared-types-and-ownership.md FR-017, FR-021). |
| FR-090 to FR-092 | architecture (FR-091: architecture + unit) | `internal/archtest`: no root-module import of `tools/`, `Monster` imports only the kernel packages and `content`; the layout check fails on any stray file in `content/data/`, monsters included; SRD text in tests only under `testdata/srd/`. |

Coverage and mutation: every changed domain line is covered (testing.md §7); section 5 domain files MUST be mutation tested with `gomu` (only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors; testing.md §8) at a score of at least 80, the decoder and parser SHOULD be. No observability tests (observability.md §9).

## 9. Open questions and decisions

Settled, one line each: PDF text via poppler `pdftotext` in Docker, cropped at 297 pt into halves (FR-061). `PartRechargeAfterRest` stays local, not `RechargeTrigger`. Languages stay verbatim, not `Language` (FR-029). R2-monsters row text is "monsters array and the monster kind; no new problem kind; no Set change" and `malformedNotation` is dropped (FR-049), landing with the first bulk data file. Spell and Gear names stay verbatim with the advisory FR-080 check, no `Key` fields (targets owned by other specs, shared-types-and-ownership.md FR-047; resolved keys can be added later, architecture.md §7). The FR-076 differences (pp. 270, 328, 341, 362) are printed in the PDF, not artefacts, and stay as printed (FR-076, FR-082). The Depends-on cycle and ability-scores.md requests are settled.

Amendments from consumer specs, applied: mounted-and-underwater-combat.md Q-20 (flag, FR-101); telepathy.md FR-023 and monster-traits.md Q-7 (notes under FR-029, FR-034); monster-combatant-setup.md FR-029 (`proficiencyBonus` is an `int`, FR-031); monster-stat-block-actions.md A-2 and A-3 (FR-098 to FR-100, FR-102).

Decisions on those amendments (all package design or SRD-based; no escalation):
- **Rider kinds.** The relayed A-3 listed two kinds; the consumer's own A-3 and its FR-056 need a third, `HalfDamage`, for the far more common plain "Success: Half damage" (SRD 5.2.1, "Half Damage", p. 16). Three kinds stand, so the consumer never parses prose.
- **`requiresUsed`.** The relayed text checks it as an Actions part, but the one SRD case, the Clay Golem's Hasten (p. 274), is a Bonus Action; it resolves against Actions, Bonus Actions and Reactions (FR-099).
- **Problem kinds.** A repeated name in one part set is `duplicate` and a rider on a non-Success clause is `constraintViolation`, following the default mapping of FR-049 and table 4.8a (the relayed text said `constraintViolation` and `invalidValue`); the verdicts are still decoder problems at the member, nothing new.
- **Underwater flag.** Valid only on Fire because the SRD grants only Fire Resistance underwater (p. 16); the flag sits on the damage instance, not the part, so a part with Fire and another type is exact.
- **Spellcasting restrictions (monster-spellcasting.md A4, optional): rejected.** `selfOnly`, `waivesConcentration` and component waivers stay verbatim notes and statements; monster-spellcasting.md Q-5 recognises them with a refuse-never-guess grammar, and a second structured copy would give two sources of truth for one printed sentence. The SRD prints these only as free text (p. 257). Revisit only if that spec moves the parsers.

Escalated: none. No open questions.

## 10. Changelog impact

`CHANGELOG.md`, under `Unreleased`, **Added**: "Monster content: `Monster`, its stat-block value objects, the `monsters` document array decoded by `JSONDecoder`, the Multiattack `plan`, `SaveSuccessRider` and `underwaterResistanceDenied` data, and the bundled SRD 5.2.1 monsters and animals (330 stat blocks)." Additive, not **BREAKING**. Once released, changing the content format version, renaming a problem kind, or changing a bundled value (outside a correction against the SRD PDF) is **BREAKING** (`architecture.md` §7). No schema version is claimed (shared-types-and-ownership.md FR-034).
