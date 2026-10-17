# Class: Bard

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `derived-stats.md`, `character-aggregate.md`, `spellcasting.md`, `feats-behaviour.md`, `level-advancement-and-multiclassing.md`, `character-creation.md`, `rests-and-hit-dice.md`, `ability-scores.md`, `d20-tests.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `content-classes.md`, `content-spells.md`, `content-equipment.md`, `content-source-port.md`, `content-schema-and-decoder.md`, `observability-ports.md`, `errors-and-results.md` |

Bard class and subclass data is owned by `content-classes.md`
(shared-types-and-ownership.md FR-025); this spec is behaviour only.
`content-equipment.md` is a data-only reference for the Musical Instrument item
category (FR-122). Countercharm rolls through the `dice.Roller` port and the pure
`d20` domain members (FR-084), so no Bard use case calls another use case.

Sibling specs are consumed unchanged (`adjustedBy` and `resolve` of d20-tests.md,
`SpellcastingState` and the generic validation of spellcasting.md, `levelUp` and
`FeatureStates` of character-aggregate.md, `SubclassChosen` of
level-advancement-and-multiclassing.md). Feats go through `FeatTakingRules`
(FR-093), slots and prepared spells through the `SpellcastingState` transitions and
`SpellPreparation` of spellcasting.md (FR-124 here), and multiclass entry seeds the pool
inside `ApplyBardLevelGain` (FR-121). This spec defines no Bonus Action / Reaction
cost type (section 9).

## 1. Purpose and scope

Specifies the Bard class as SRD 5.2.1 presents it: every Bard class feature and
College of Lore. Content data (traits, progression rows, feature ids, spell list
membership, subclass) is owned by `content-classes.md` and content-spells.md. This
spec owns behaviour: choices at level gain, Bardic Inspiration state and use, rest
and Initiative recovery, Expertise, Jack of All Trades, spell preparation rules,
Cutting Words, Peerless Skill, Countercharm, as pure functions from (character
state, inputs) to (new state, events), with one use case only where a feature
needs dice or content (decisions FR-023). Per-feature state lives in the
aggregate's `FeatureStates` under a typed key (FR-118); creation choices travel in
`CreateCharacter` as `BardCreationChoices` (decisions FR-024).

**Out of scope:**

- Other classes and every subclass other than College of Lore (the only Bard
  subclass in SRD 5.2.1, p. 35).
- The Cleric, Druid and Wizard spell lists (`Spell.classes` in content-spells.md);
  this spec only filters by `Spell.classes` containing the `ClassId`.
- Spell entries and casting resolution; feat definitions, Musical Instrument
  items and the Entertainer's Pack (all `content`); applying a feat and
  qualification (feats-behaviour.md).
- Action economy and geometry (Bonus Action or Reaction availability, "can see or
  hear", 60/30/10 feet): the host or `combat` decides.
- Multiclass spell slot calculation (spellcasting.md, character-aggregate.md);
  Heroic Inspiration (character-aggregate.md); `RestKind` and rest effects
  (rests-and-hit-dice.md, this spec only reacts to a finished rest).
- The non-binding "recommended" spell and feat suggestions in the class text.

## 2. Source

Printed SRD page numbers equal PDF page numbers (offset 0, checked).

- SRD 5.2.1, "Bard", "Core Bard Traits", "Becoming a Bard", p. 31
- SRD 5.2.1, "Bard Features" table, p. 31
- SRD 5.2.1, "Level 1: Bardic Inspiration", pp. 31-32
- SRD 5.2.1, "Level 1: Spellcasting" through "Level 5: Font of Inspiration", p. 32
- SRD 5.2.1, "Level 7: Countercharm" through "Level 20: Words of Creation", p. 33
- SRD 5.2.1, "Bard Spell List", pp. 33-35
- SRD 5.2.1, "Bard Subclass: College of Lore", p. 35
- SRD 5.2.1, "Rules Glossary": "Advantage", p. 176; "Bonus Action", p. 177;
  "Expertise", p. 182; "Initiative", p. 184; "Long Rest", p. 185;
  "Reaction", p. 186; "Round Down", "Short Rest", p. 187;
  "Spellcasting Focus", p. 188
- SRD 5.2.1, "Playing the Game": "Rolling 20 or 1", p. 7; "They Don't Stack"
  and "Interactions with Rerolls", p. 8

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Bard, Core Bard Traits, Bard Features, College of Lore | content | `ClassId` `bard`, `SubclassId` `college_of_lore`, `ClassDefinition`, `ProgressionRow`, `FeatureEntry` in content-classes.md (class data ownership: shared-types-and-ownership.md FR-025); feature ids by `deriveFeatureId` (content-classes.md FR-052), section 5.1 |
| Bardic Inspiration | feature / state | A pool of uses held by the Bard, and a die held by a recipient |
| Expertise, Jack of All Trades | features | Doubling in derived-stats.md; Jack of All Trades is half Proficiency Bonus, round down |
| Spellcasting, Countercharm, Magical Secrets, Words of Creation, Bonus Proficiencies, Cutting Words, Magical Discoveries, Peerless Skill | features | Spellcasting Ability Charisma, Focus a Musical Instrument; levels 7, 10, 20; College of Lore 3, 3, 6, 14 |
| D20 Test, Saving Throw, Ability Check, Attack Roll, DC | `d20` terms | Used, not defined: `CheckResult`, `AttackRollResult` (`adjustedBy`), `D20TestOutcome`, `DifficultyClass`, `Roll`, `RollModifiers`; `RollMode`, `RollCircumstance`, `Modifier` (kernel) |
| Charmed, Frightened | `ConditionKind` values | Closed vocabulary owned by `content` (decisions FR-005); condition behaviour stays in `combat`, unused here |
| Short Rest, Long Rest | `RestKind` | rests-and-hit-dice.md (decisions FR-009); a trigger value here (FR-038); Initiative is the trigger of Superior Inspiration only (`BardRecoveryInitiative`) |
| Bonus Action, Reaction | action types | No `character`-side type (the plug-in types live in downstream `combat`); the cost is stated in each operation's documentation and charged by the host (section 9) |

**Go conventions in this spec** (binding vocabulary: errors-and-results.md,
shared-types-and-ownership.md). Function and method names written lowerCamelCase
below (`rowAt`, `activateRage`) are exported PascalCase in Go (`RowAt`,
`ActivateRage`); enum-like values written lowerCamelCase (`notABarbarian`) are typed
constants with a `String()` method. A "closed" result or choice is an interface with
an unexported marker method, one struct per variant, handled by an exhaustive type
switch; the variant list is in its doc comment. Untrusted input is validated by
`NewX` / `ParseX`, which return `(T, error)` with a typed, `errors.As`-able error and
report every problem at once with `errors.Join`; a port failure is a returned
`*srderr.PortError` (`errors.Is` / `errors.As`); a panic carrying an error value is
reserved for programmer errors (violated precondition) and appears only in `MustX`
helpers and unreachable branches. A function that may do I/O or emit spans takes
`context.Context` first; pure rule functions do not.

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. "Bard level" means
the number of Bard levels the character has. Ids FR-112 to FR-114 were never
allocated.

### 4.1 Core Bard Traits (creation choices)

Bard choice validation is `character` domain code carried by
`BardCreationChoices` and `BardMulticlassChoices` (decisions FR-024). A
rejection is a closed `BardChoiceProblem` value; every problem is listed
together, never just the first. The trait data itself is content-classes.md.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-004 | A `BardCreationChoices` Skill Proficiencies selection of exactly 3 distinct skills MUST be accepted and recorded as proficiencies. | SRD 5.2.1, "Core Bard Traits", p. 31 | happy |
| FR-005 | A Skill Proficiencies selection with fewer or more than 3 skills, or a duplicate skill, MUST be rejected with a result naming the problem. | SRD 5.2.1, "Core Bard Traits", p. 31 | unhappy (invalid input) |
| FR-007 | A `BardCreationChoices` Tool Proficiencies selection of exactly 3 distinct Musical Instruments MUST be accepted. | SRD 5.2.1, "Core Bard Traits", p. 31 | happy |
| FR-008 | A Tool Proficiencies selection with fewer or more than 3 items, a duplicate, or an item that is not a Musical Instrument MUST be rejected with a result naming the problem. | SRD 5.2.1, "Core Bard Traits", p. 31 | unhappy (invalid input) |
| FR-012 | A Starting Equipment selection that picks neither or both options, or picks option A with a Musical Instrument choice that is missing or is not a Musical Instrument, MUST be rejected with a result. | SRD 5.2.1, "Core Bard Traits", p. 31 | unhappy (invalid input) |
| FR-015 | A `BardMulticlassChoices` that does not name exactly one skill and exactly one Musical Instrument MUST be rejected with a result. | SRD 5.2.1, "As a Multiclass Character", p. 31 | unhappy (invalid input) |
| FR-122 | The validators of FR-004, FR-005, FR-007, FR-008, FR-012 and FR-015 are pure functions with stated inputs: the selection; the decoded `SkillChoice` of the Bard `ClassDefinition` (`SkillChoiceAny` with 3 picks at creation, and the multiclass `skillChoice`; option set is every `Skill` for `SkillChoiceAny`, the listed skills for `SkillChoiceListed`); the decoded `ClassToolProficiency` (`ClassToolProficiencyChoice`, `categories` containing "Musical Instrument", `picks` 3 at creation, 1 at multiclass; content-classes.md FR-023); and a caller-supplied `Set<ItemId>` of Musical Instrument items from content-equipment.md item category data. An item is a Musical Instrument exactly when its `ItemId` is in that set; a validator never reads a `ContentSource`. A selection outside the option set of its `SkillChoice` MUST be rejected with a result naming the problem. | n/a (package design, architecture.md section 4; SRD 5.2.1, "Core Bard Traits", p. 31) | unhappy (invalid input) |

### 4.2 Bard Features table and Bard level

"Table 1" in this spec means SRD 5.2.1, "Bard Features", p. 31, as decoded in
content-classes.md (Proficiency Bonus, Bardic Die, cantrips, prepared spells,
slots per level: content-classes.md FR-033, FR-038, FR-041, FR-042). Tests read it
through `BundledContentSource`; this spec keeps no copy.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-022 | A Bard level below 1 or above 20 MUST be rejected: `MustBardLevel` panics with an error value, and `NewBardLevel` for untrusted input returns an error. A `BardLevel` is read from the Bard entry of `ClassLevels`; a character with no Bard entry has no `BardLevel`. Levels 0, 1, 20 and 21 MUST be tested. Use cases never build a `BardLevel` from a raw int; they hold a validated one and compute the next (FR-099). | SRD 5.2.1, "Bard Features", p. 31 | unhappy (boundary) |
| FR-023 | The Proficiency Bonus in the decoded Bard progression (content-classes.md) MUST equal the derived-stats.md `ProficiencyBonus` for the same class level. It needs decoded data and a derived value together, so it is a scenario-level check through `BundledContentSource`. | SRD 5.2.1, "Bard Features", p. 31 | unhappy (rule interaction) |

### 4.3 Bardic Inspiration

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-025 | The Number of Uses of Bardic Inspiration MUST equal the Charisma modifier with a minimum of once (cases below). | SRD 5.2.1, "Bardic Inspiration", "Number of Uses", p. 32 | unhappy (boundary) |
| FR-026 | Granting Bardic Inspiration to another creature MUST expend one use from the Bard's pool and give the recipient one die of the Bard's current Bardic Die size. The Bonus Action cost is documented on the operation for the host to charge (section 9). | SRD 5.2.1, "Using Bardic Inspiration", p. 31; "Bonus Action", p. 177 | happy |
| FR-027 | The recipient is a closed `InspirationRecipient`: `InspirationRecipientSelf` or `InspirationRecipientOther(currentGrant)`, where `currentGrant` is the `BardicInspirationGrant` the other creature already holds, or null. The host decides the variant (a monster or NPC is an `Other`); the domain compares no identities. Granting to `InspirationRecipientSelf` MUST be rejected with a result (`selfTarget`); the SRD says "another creature". | SRD 5.2.1, "Using Bardic Inspiration", p. 31 | unhappy (invalid input) |
| FR-028 | Granting with no uses remaining MUST be rejected with a result and MUST NOT change the pool. | SRD 5.2.1, "Number of Uses", p. 32 | unhappy (state error) |
| FR-029 | Granting to an `InspirationRecipientOther` whose `currentGrant` is present MUST be rejected with a result, MUST NOT expend a use, and MUST NOT replace the held die. | SRD 5.2.1, "Using Bardic Inspiration", pp. 31-32 | unhappy (state error) |
| FR-030 | When the holder of a die fails a D20 Test within the next hour, using it MUST roll the Bardic Die through the `dice.Roller` port (the use case rolls; it decides nothing) and add the number rolled to the d20 of that test (pure `applyBardicInspirationDie`, FR-119). | SRD 5.2.1, "Using Bardic Inspiration", pp. 31-32 | happy |
| FR-031 | The outcome of the adjusted D20 Test MUST be decided by `adjustedBy(rolled)` of the failed `CheckResult` or `AttackRollResult` (d20-tests.md FR-065), which appends `Modifier(rolled)` to `RollModifiers.others` so the total invariant of d20-tests.md FR-016 holds; never re-derived in `character`, never made by rolling the d20 again (cases below). | SRD 5.2.1, "Using Bardic Inspiration", pp. 31-32 | happy |
| FR-104 | The die rolled for Bardic Inspiration, Cutting Words or Peerless Skill is a modifier to the total, not a new d20. A natural 1 and a natural 20 stay judged on the d20 used (d20-tests.md FR-027 to FR-034 and FR-066, applied by `adjustedBy`): an attack roll that was a natural 1 stays a miss when Bardic Inspiration or Peerless Skill is added, and a natural 20 stays a critical hit when Cutting Words is subtracted. Ability checks and saving throws use the adjusted total only. | SRD 5.2.1, "Rolling 20 or 1", p. 7; "Using Bardic Inspiration", pp. 31-32; "Level 3: Cutting Words", "Level 14: Peerless Skill", p. 35; n/a (design decision) | unhappy (boundary) |
| FR-032 | The die MUST be expended when it is rolled, whether or not the adjusted test then succeeds. | SRD 5.2.1, "Using Bardic Inspiration", pp. 31-32 | unhappy (rule interaction) |
| FR-033 | Using a die on a D20 Test that was not failed MUST be rejected with a result, MUST NOT roll, and MUST NOT expend the die. | SRD 5.2.1, "Using Bardic Inspiration", pp. 31-32 | unhappy (state error) |
| FR-034 | Using a die when more than one hour has elapsed since it was granted MUST be rejected with an expired result, MUST NOT roll, and the die is lost: the rejection carries the expired grant to discard and one `BardicInspirationExpired` event (FR-116). | SRD 5.2.1, "Using Bardic Inspiration", pp. 31-32 | unhappy (boundary) |
| FR-116 | The `expired` rejection of FR-034 MUST carry the grant to discard and a `BardicInspirationExpired` event; a later attempt to use the same grant MUST be rejected again. The non-failure rejection (FR-033) carries no grant because the die stays. | n/a (package design, architecture.md section 3) | unhappy (state error) |
| FR-035 | Using a die exactly one hour after it was granted MUST be accepted ("within the next hour"). | SRD 5.2.1, "Using Bardic Inspiration", pp. 31-32 | unhappy (boundary) |
| FR-036 | If the `dice.Roller` port returns a typed `*srderr.PortError` while using, the use case MUST propagate it unchanged, and the die remains unexpended (no new state is returned; the pure functions run only after a successful roll). | n/a (package design, architecture.md section 4) | unhappy (collaborator failure) |
| FR-037 | Granting and using MUST NOT alter the input state or grant; each returns a new state and the events produced. | n/a (package design, architecture.md section 3) | unhappy (immutability) |

Number of Uses cases (FR-025), SRD 5.2.1, p. 32: Charisma modifier -5, -1 and 0 give 1 use (boundary); +1 gives 1; +2 gives 2; +5 gives 5.

Using-a-die cases (FR-030, FR-031, FR-032). Ability check, DC 15, unless stated;
"Roll" is the die roll from a fixed `dice.Roller`. Source SRD 5.2.1, p. 32 (the
natural-1 rows p. 7).

| Label | Die | Original total | Roll | Adjusted total | Expected outcome | Die expended | Path |
|---|---|---|---|---|---|---|---|
| exactly reaches the DC | d6 | 14 | 1 | 15 | success | yes | unhappy (boundary) |
| one short of the DC | d6 | 12 | 2 | 14 | failure | yes | unhappy (boundary) |
| comfortably succeeds | d6 | 12 | 6 | 18 | success | yes | happy |
| maximum d6 just enough | d6 | 9 | 6 | 15 | success | yes | unhappy (boundary) |
| d12 at level 15+ | d12 | 3 | 12 | 15 | success | yes | unhappy (boundary) |
| minimum roll, still fails | d8 | 5 | 1 | 6 | failure | yes | unhappy (boundary) |
| attack roll against AC 14 | d6 | 12 | 2 | 14 | hit | yes | happy |
| attack roll one short of AC 14 | d6 | 12 | 1 | 13 | miss | yes | unhappy (boundary) |
| natural 1 attack stays a miss (FR-104) | d6 | 1 (natural 1, modifier +0) | 6 | 7 against AC 5 | miss | yes | unhappy (boundary) |
| natural 1 ability check uses the total (FR-104; the natural-1 rule is attack-roll only) | d6 | 1 (natural 1, modifier +10) | 6 | 17 against DC 15 | success | yes | unhappy (boundary) |

### 4.4 Recovering Bardic Inspiration

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-038 | Finishing a Long Rest (trigger `BardRecoveryRest(RestKind.longRest)`) MUST restore all expended uses of Bardic Inspiration, at every Bard level. | SRD 5.2.1, "Number of Uses", p. 32; "Long Rest", p. 185 | happy |
| FR-039 | From Bard level 5, finishing a Short Rest (`BardRecoveryRest(RestKind.shortRest)`) MUST restore all expended uses (Font of Inspiration). | SRD 5.2.1, "Level 5: Font of Inspiration", p. 32; "Short Rest", p. 187 | happy |
| FR-040 | Below Bard level 5, finishing a Short Rest MUST NOT restore any use. Levels 4 and 5 MUST be tested. | SRD 5.2.1, "Number of Uses", "Level 5: Font of Inspiration", p. 32 | unhappy (boundary) |
| FR-041 | From Bard level 5, the Bard's choice to expend a spell slot (`ExpendSlotForInspiration`, section 6) MUST spend that slot through the `SpellcastingState` slot transition of FR-124 (spellcasting.md FR-021 to FR-024, never a call to the use case `ExpendSpellSlot`) over the character's `SpellcastingState` and the chosen `SpellSlotPool` (`spellcasting`, or `pactMagic` for a Bard 5 / Warlock multiclass: the caller chooses, the Bard code never infers it), return the new `SpellcastingState`, forward the `SpellSlotExpended` event, and restore exactly one expended use (event `BardicInspirationRestored`); it takes no action. The caller writes back both the `FeatureStates` pool and the `SpellcastingState`. The function re-implements no slot spending. Casting a spell does not restore a use. | SRD 5.2.1, "Level 5: Font of Inspiration", p. 32 | happy |
| FR-042 | `ExpendSlotForInspiration` MUST be rejected with a result when the Bard is below level 5 (`belowRequiredLevel`), has no expended use to restore (`nothingToRestore`, checked before any slot is spent), or spellcasting.md rejects with its `SlotOperationRejection` (`noSlotAvailable`); in each case no slot is spent and the `SpellcastingState` and pool are unchanged. | SRD 5.2.1, "Level 5: Font of Inspiration", p. 32 | unhappy (state error) |
| FR-109 | The `slotLevel` of `ExpendSlotForInspiration` is spellcasting.md's `SlotLevel` (1 to 9), never a raw int: a caller builds it with `NewSlotLevel`, whose out-of-range error (levels 0 and 10) is the caller's to handle, so the function has no Bard `invalidSlotLevel` rejection. A level with no slots in the chosen pool is rejected by the spellcasting operation as `noSlotAvailable` (spellcasting.md FR-023). Any level with an unexpended slot is accepted: the SRD sets no minimum. Levels 0, 1, 9 and 10 MUST be tested (0 and 10 at `NewSlotLevel`, 1 and 9 through the function). | SRD 5.2.1, "Level 5: Font of Inspiration", "Spell Slots", p. 32 | unhappy (boundary) |
| FR-043 | From Bard level 18, when the Bard rolls Initiative (`BardRecoveryInitiative`), expended uses MUST be restored until the Bard has two uses if fewer than two; a Bard with two or more MUST NOT lose any (cases below). | SRD 5.2.1, "Level 18: Superior Inspiration", p. 33; "Initiative", p. 184 | unhappy (boundary) |
| FR-044 | Below Bard level 18, rolling Initiative MUST NOT restore any use. Levels 17 and 18 MUST be tested. | SRD 5.2.1, "Level 18: Superior Inspiration", p. 33 | unhappy (boundary) |
| FR-045 | No recovery MUST raise the uses remaining above the Number of Uses. | SRD 5.2.1, "Number of Uses", p. 32 | unhappy (rule interaction) |
| FR-108 | Where the maximum is below two (Charisma modifier of +1 or less), Superior Inspiration MUST stop at the maximum rather than reach two. | n/a (design decision) | unhappy (rule interaction) |

Superior Inspiration cases (FR-043, FR-044, FR-045, FR-108), SRD 5.2.1, p. 33, as
remaining before to after: level 18, max 4: 0 to 2, 1 to 2, 2 to 2, 3 to 3 (boundary);
level 17: 0 to 0; level 18, max 1: 0 to 1 (design decision).

### 4.5 Spellcasting

Mechanics common to all casters are in spellcasting.md. The generic validation
(list membership, level ceiling, duplicates, prepared limit, always-prepared,
replacement trigger and count, every problem reported) is spellcasting.md
FR-040 to FR-054, used here, not re-specified; the Bard's prepared spells are
its `PreparedSpells`. The FRs below fix the Bard inputs to those rules. The SRD
reserves "spellbook" for the Wizard: no Bard spellbook type.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-046 | The Bard's Spellcasting Ability MUST be Charisma. | SRD 5.2.1, "Spellcasting Ability", p. 32 | happy |
| FR-047 | Cross-reference only: the Bard's Spellcasting Focus kind (Musical Instrument; no other item substitutes) is owned by spellcasting.md FR-068 (Table A). The Bard test asserts only that the Bard row reads Musical Instrument. | SRD 5.2.1, "Spellcasting Focus", p. 32; "Rules Glossary", p. 188 | unhappy (invalid input) |
| FR-049 | At level 1 the Bard MUST know two cantrips of the player's choice from the Bard spell list. | SRD 5.2.1, "Cantrips", p. 32 | happy |
| FR-050 | A cantrip choice with the wrong count, a duplicate, a spell whose `Spell.classes` lacks Bard, or a spell that is not a cantrip MUST be rejected with a result listing every problem (delegated to spellcasting.md FR-052 and FR-054 with the Table 1 count). | SRD 5.2.1, "Cantrips", p. 32 | unhappy (invalid input) |
| FR-051 | On reaching Bard level 4 and level 10, the Bard MUST learn one more cantrip from the Bard spell list, so the count matches Table 1 (3 at level 4, 4 at level 10). No other level changes the count. | SRD 5.2.1, "Cantrips", p. 32 | unhappy (boundary) |
| FR-052 | Whenever the Bard gains a Bard level, the player MAY replace at most one known cantrip with another Bard cantrip; a second replacement at the same level gain MUST be rejected. | SRD 5.2.1, "Cantrips", p. 32 | unhappy (boundary) |
| FR-053 | At level 1 the Bard MUST prepare four level 1 spells from the Bard spell list. | SRD 5.2.1, "Prepared Spells of Level 1+", p. 32 | happy |
| FR-054 | When the Prepared Spells number in Table 1 increases, the Bard MUST choose additional Bard spells until the list matches; each chosen spell MUST be of a level for which the Bard has spell slots (cases below). | SRD 5.2.1, "Prepared Spells of Level 1+", p. 32 | unhappy (boundary) |
| FR-055 | A prepared-spell choice that is not on the allowed lists, has a duplicate, has a spell level without slots, or leaves the list count different from Table 1 MUST be rejected with a result listing every problem (delegated to spellcasting.md FR-040 to FR-043, FR-047, FR-054 with the Table 1 count and the allowed lists of FR-088 and FR-091). | SRD 5.2.1, "Prepared Spells of Level 1+", p. 32 | unhappy (invalid input) |
| FR-056 | Whenever the Bard gains a Bard level, the player MAY replace one prepared spell with another Bard spell for which the Bard has slots; a second replacement at the same level gain MUST be rejected. | SRD 5.2.1, "Changing Your Prepared Spells", p. 32 | unhappy (boundary) |
| FR-057 | Spells the Bard always has prepared through another Bard feature (Magical Discoveries, Words of Creation) MUST NOT count against the Prepared Spells number in Table 1, and MUST otherwise count as Bard spells. | SRD 5.2.1, "Prepared Spells of Level 1+", p. 32 | unhappy (rule interaction) |
| FR-058 | Finishing a Long Rest MUST restore all expended Bard spell slots. Owner: the generic rest recovery of `SpellcastingState` slots in spellcasting.md (`RestoreSpellSlots`); the Bard test only asserts a Bard's slots come back full through it. | SRD 5.2.1, "Spell Slots", p. 32 | happy |
| FR-059 | Finishing a Short Rest MUST NOT restore Bard spell slots. Owner as FR-058. | SRD 5.2.1, "Spell Slots", p. 32 | unhappy (state error) |

Prepared-spell level cases (FR-054, FR-055): the highest spell level a chosen spell may have equals the highest level with slots in Table 1 (SRD 5.2.1, p. 31). Bard level 1 and 2 allow 1; level 3 allows 2 and rejects a level 3 spell (p. 32); level 5 allows 3; level 9 allows 5; level 17 allows 9.

### 4.6 Expertise and Jack of All Trades

Expertise state and doubling are derived-stats.md (its FR-015, FR-016); the
requirements here fix only when the Bard owes the choice and how many skills.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-060 | At Bard level 2 the Bard MUST gain Expertise in two of the skills in which it has proficiency. | SRD 5.2.1, "Level 2: Expertise", p. 32 | happy |
| FR-061 | At Bard level 9 the Bard MUST gain Expertise in two more skills in which it has proficiency. | SRD 5.2.1, "Level 2: Expertise", p. 32 | happy |
| FR-062 | An Expertise choice MUST be rejected with a result listing every problem when it does not name exactly two skills, names a duplicate, names a skill without proficiency, or names a skill that already has Expertise. | SRD 5.2.1, "Level 2: Expertise", p. 32; "Rules Glossary", "Expertise", p. 182 | unhappy (invalid input) |
| FR-064 | Jack of All Trades MUST be a pure domain function returning an optional `Modifier` from (the Bard level or none, the `ProficiencyBonus`, the optional `Skill` of the check or null, the `SkillProficiencies`, and whether the check already uses the Proficiency Bonus through another source such as a proficient tool). It returns half the Proficiency Bonus, rounded down, for an ability check that uses a skill the character lacks and does not otherwise use the Proficiency Bonus, and null otherwise (cases below). The check composer places a present result in `RollModifiers.others`, never in the Proficiency Bonus slot; derived-stats.md has no skill-level feature input (its `FlatBonus` is Armor Class only), so the per-check input lives here. | SRD 5.2.1, "Level 2: Jack of All Trades", p. 32; "Rules Glossary", "Round Down", p. 187 | happy |
| FR-065 | Jack of All Trades MUST NOT apply below Bard level 2, to a skill the character is proficient in (including Expertise), or to an ability check that uses no skill. | SRD 5.2.1, "Level 2: Jack of All Trades", p. 32 | unhappy (boundary) |

Jack of All Trades cases (FR-064, FR-065), SRD 5.2.1, p. 32:

| Label | Bard level | Proficiency Bonus | Skill state | Bonus added | Path |
|---|---|---|---|---|---|
| level 2, lacks Athletics | 2 | +2 | no proficiency | +1 | happy |
| level 5, rounds down (p. 187) | 5 | +3 | no proficiency | +1 | unhappy (boundary) |
| level 9 | 9 | +4 | no proficiency | +2 | happy |
| level 13, rounds down (p. 187) | 13 | +5 | no proficiency | +2 | unhappy (boundary) |
| level 17 | 17 | +6 | no proficiency | +3 | happy |
| proficient skill | 5 | +3 | proficiency | 0 from this feature | unhappy (rule interaction) |
| Expertise skill | 9 | +4 | Expertise | 0 from this feature | unhappy (rule interaction) |
| ability check with no skill | 5 | +3 | not applicable | 0 | unhappy (invalid input) |
| level 1 | 1 | +2 | no proficiency | 0 | unhappy (boundary) |
| skill-lacking check that also uses a proficient tool ("does not otherwise use your Proficiency Bonus"; "The Bonus Doesn't Stack", p. 8) | 5 | +3 | no skill proficiency, proficient tool | 0 from this feature (not added twice) | unhappy (rule interaction) |

### 4.7 Subclass and College of Lore

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-066 | At Bard level 3 the Bard MUST choose a Bard subclass; before level 3 a choice MUST be rejected. The accepted choice is the `SubclassChosen` event and a rejection `SubclassRejected`, both owned by level-advancement-and-multiclassing.md. | SRD 5.2.1, "Level 3: Bard Subclass", p. 32 | unhappy (boundary) |
| FR-067 | A chosen `SubclassId` is resolved from the loaded `ClassDefinition.subclasses` of the Bard (a `SubclassId` has no `ContentKind` or `ContentKey`, so it cannot be looked up through the port or yield a `MissingContent`; content-source-port.md section 3). An id absent from that list MUST be reported as `BardChoiceProblem.unknownSubclass(id)` inside `ApplyBardLevelGainInvalidChoices`, never as `ApplyBardLevelGainContentMissing` and never as an error. | SRD 5.2.1, "Bard Subclass: College of Lore", p. 35 | unhappy (missing content) |
| FR-068 | A Bard that already has a subclass MUST reject a second subclass choice with a result (`SubclassRejected`); the subclass is unchanged. | SRD 5.2.1, "Level 3: Bard Subclass", p. 32 | unhappy (state error) |
| FR-069 | The Bard MUST gain each subclass feature whose level is the Bard level or lower: Bonus Proficiencies and Cutting Words at 3, Magical Discoveries at 6, Peerless Skill at 14. Because levels are gained one at a time (FR-115), a subclass feature never comes owed from a lower level. | SRD 5.2.1, "Level 3: Bard Subclass", p. 32; "College of Lore", p. 35 | unhappy (boundary) |
| FR-070 | College of Lore Bonus Proficiencies MUST grant proficiency in three skills of the player's choice. | SRD 5.2.1, "Level 3: Bonus Proficiencies", p. 35 | happy |
| FR-071 | A Bonus Proficiencies choice that does not name exactly three distinct skills MUST be rejected with a result. | SRD 5.2.1, "Level 3: Bonus Proficiencies", p. 35 | unhappy (invalid input) |
| FR-110 | A Bonus Proficiencies choice (FR-071) or a Multiclass skill choice (FR-015) that names a skill the character is already proficient in MUST be rejected with a result, because a repeat gains nothing. | n/a (design decision) | unhappy (invalid input) |
| FR-072 | Cutting Words MUST let the Bard expend one use of Bardic Inspiration, roll its die through the `dice.Roller` port, and subtract the number from the triggering creature's damage roll, or from an ability check or attack roll that succeeded (the use case rolls; pure `planCuttingWords` and `applyCuttingWords` decide the rest, FR-119). The trigger is a `character` value (section 6): an `AdjustableD20Result`, or a damage total as a non-negative `int` (never a `combat` type, decisions FR-011, FR-013). The Reaction cost is documented on the operation; the result carries no cost field (section 9). | SRD 5.2.1, "Level 3: Cutting Words", p. 35; "Reaction", p. 186 | happy |
| FR-073 | When Cutting Words is applied to a successful ability check or attack roll, the outcome MUST be decided by `adjustedBy(-rolled)` (d20-tests.md FR-065): the success turns into a failure exactly when the resolution says so (cases below; FR-104 for a natural 20). | SRD 5.2.1, "Level 3: Cutting Words", p. 35 | happy |
| FR-074 | Cutting Words MUST be rejected with a result, with no roll and no use spent, when the Bard has no use remaining, is below level 3, has no College of Lore subclass, or the trigger is ineligible: a saving throw (`CheckResult` of kind `savingThrow`), or an ability check or attack roll whose outcome is not `success` (`ineligibleTrigger`). Gates, eligibility and check order are decided by the pure `planCuttingWords`, before any roll. | SRD 5.2.1, "Level 3: Cutting Words", p. 35 | unhappy (state error) |
| FR-075 | Cutting Words MUST use the Bard's current Bardic Die size and MUST spend the use whether or not the success becomes a failure. | SRD 5.2.1, "Level 3: Cutting Words", p. 35; "Using Bardic Inspiration", pp. 31-32 | unhappy (rule interaction) |
| FR-107 | When the number rolled for Cutting Words exceeds a damage total, the reduced damage MUST be zero, never negative; the result records that it was clamped. | n/a (design decision) | unhappy (boundary) |
| FR-076 | At Bard level 6 Magical Discoveries MUST add two spells of the player's choice, each from the Cleric, Druid or Wizard spell list in any combination (three lists; the Bard list is not named, unlike Magical Secrets, FR-088). A spell is on a list when its `Spell.classes` contains that class. | SRD 5.2.1, "Level 6: Magical Discoveries", p. 35 | happy |
| FR-077 | Each Magical Discoveries spell MUST be a cantrip or of a level for which the Bard has slots; a choice that is not exactly two spells, has a duplicate, is on none of the Cleric, Druid and Wizard lists (a Bard-only spell is rejected), or has a level without slots MUST be rejected with a result listing every problem. A spell on the Bard list and also on one of the three lists MUST be accepted. A pure domain rule over `Spell` values. | SRD 5.2.1, "Level 6: Magical Discoveries", p. 35 | unhappy (invalid input) |
| FR-078 | Magical Discoveries spells MUST always be prepared and MUST NOT count against the Prepared Spells number (FR-057); the SRD makes this decided: they "otherwise count as Bard spells for you" (p. 32). | SRD 5.2.1, "Level 6: Magical Discoveries", p. 35; "Prepared Spells of Level 1+", p. 32 | unhappy (rule interaction) |
| FR-079 | Whenever the Bard gains a Bard level, the player MAY replace one Magical Discoveries spell with another meeting the same requirements; a second replacement at the same level gain MUST be rejected. | SRD 5.2.1, "Level 6: Magical Discoveries", p. 35 | unhappy (boundary) |
| FR-080 | If a chosen spell id, or the spell entries needed to evaluate Magical Discoveries (Cleric, Druid, Wizard lists) or Magical Secrets (Bard, Cleric, Druid, Wizard lists), are not found in the content source (`lookup` or `listKind(ContentKind.spell)` filtered by `Spell.classes` containing the `ClassId`: bard, cleric, druid, wizard), the operation MUST return `ApplyBardLevelGainContentMissing` holding the `MissingContent(kind, id)`, and MUST NOT silently restrict the choice to the Bard list. | SRD 5.2.1, "Level 6: Magical Discoveries", p. 35; "Level 10: Magical Secrets", p. 33 | unhappy (missing content) |
| FR-081 | Peerless Skill MUST let the Bard, when it fails an ability check or attack roll, expend one use of Bardic Inspiration, roll the die through the `dice.Roller` port, and add the number to the d20 (cases below). | SRD 5.2.1, "Level 14: Peerless Skill", p. 35 | happy |
| FR-082 | When the Peerless Skill result is still a failure, the use MUST NOT be expended; the outcome is decided by `adjustedBy(rolled)` (d20-tests.md FR-065) on the new total (FR-104 for a natural 1); whether the use is spent is decided by the pure `applyPeerlessSkill`. | SRD 5.2.1, "Level 14: Peerless Skill", p. 35 | unhappy (rule interaction) |
| FR-083 | Peerless Skill MUST be rejected with a result, with no roll, when the Bard has no use remaining, is below level 14, has no College of Lore subclass, or the trigger is ineligible: a saving throw, or an ability check or attack roll whose outcome is not `failure` (`ineligibleTrigger`). Gates and eligibility are decided by the pure `planPeerlessSkill`, before any roll. | SRD 5.2.1, "Level 14: Peerless Skill", p. 35 | unhappy (state error) |
| FR-111 | Magical Secrets MUST widen the list only when the Prepared Spells number increases: at Bard levels 12, 14 and 16 (no increase in Table 1) nothing may be chosen from the Cleric, Druid or Wizard list as an addition; a replacement from those lists remains allowed from level 10 (FR-090). Levels 11 (increase) and 12 (no increase) MUST be tested. | SRD 5.2.1, "Level 10: Magical Secrets", p. 33 | unhappy (boundary) |

Magical Discoveries list cases (FR-076, FR-077), Bard level 6 (slots to level 3), SRD 5.2.1, p. 35:

| Label | Chosen spells (lists) | Expected | Path |
|---|---|---|---|
| one each from the three lists | a Cleric-only, a Druid-only | accepted | happy |
| Bard-only spell | a Bard-only spell and a Wizard-only | rejected (Bard-only spell on none of the three lists) | unhappy (invalid input) |
| Bard and another list | a spell on the Bard and Wizard lists, and a Druid-only | accepted | unhappy (boundary) |
| level 4 spell | a Wizard level 4 spell and a Cleric-only | rejected (no level 4 slots at level 6) | unhappy (boundary) |

Cutting Words cases (FR-072, FR-073, FR-075). Die d8 unless stated; SRD 5.2.1, p. 35 (natural 20 row p. 7).

| Label | Trigger | Original value | Roll | Reduced value | Expected | Use spent | Path |
|---|---|---|---|---|---|---|---|
| success becomes failure | ability check, DC 15 | 15 | 1 | 14 | failure | yes | unhappy (boundary) |
| still meets the DC | ability check, DC 15 | 17 | 2 | 15 | success | yes | unhappy (boundary) |
| comfortably succeeds | ability check, DC 15 | 20 | 4 | 16 | success | yes | happy |
| attack stops hitting | attack roll, AC 13 | 14 | 2 | 12 | miss | yes | unhappy (boundary) |
| attack still hits | attack roll, AC 13 | 14 | 1 | 13 | hit | yes | unhappy (boundary) |
| damage reduced | damage roll | 9 | 4 | 5 | damage 5 | yes | happy |
| roll exceeds the damage (FR-107, design decision) | damage roll | 3 | 8 | 0 (clamped) | damage 0 | yes | unhappy (boundary) |
| natural 20 attack stays a critical hit (FR-104) | attack roll, AC 30 | 20 (natural 20, modifier +5 = 25) | 8 | 17 against AC 30 | critical hit | yes | unhappy (boundary) |

Peerless Skill cases (FR-081, FR-082). Die d10 (Bard level 14); SRD 5.2.1, p. 35 (natural 1 row p. 7).

| Label | Trigger | Original total | Roll | New total | Expected | Use spent | Path |
|---|---|---|---|---|---|---|---|
| reaches the DC | ability check, DC 15 | 10 | 5 | 15 | success | yes | unhappy (boundary) |
| one short of the DC | ability check, DC 15 | 10 | 4 | 14 | failure | no | unhappy (boundary) |
| well over | ability check, DC 15 | 14 | 10 | 24 | success | yes | happy |
| natural 1 attack stays a miss (FR-104) | attack roll, AC 5 | 1 (natural 1, modifier +0) | 9 | 10 against AC 5 | miss | no | unhappy (boundary) |

### 4.8 Countercharm

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-084 | When the Bard or a creature within 30 feet fails a saving throw against an effect that applies the Charmed or Frightened condition, Countercharm MUST cause the save to be rerolled and the new roll has Advantage. The pure `planCountercharm` takes the original save's circumstances plus one Advantage `RollCircumstance` and returns the mode from the kernel `RollMode.resolve`; the use case rolls that many d20 through the `dice.Roller` port (calls no other use case, emits no `srd5e.d20_test.*` signal); the pure `resolveCountercharm` builds the `Roll` from the mode and face values (d20-tests.md FR-055) and resolves it with `CheckResult.resolve` (its FR-063) against the original `RollModifiers` and DC. `character` never picks the higher d20 itself. The Reaction cost is documented on the operation; no cost field (section 9). | SRD 5.2.1, "Level 7: Countercharm", p. 33; "Rules Glossary", "Advantage", p. 176 | happy |
| FR-085 | The rerolled save MUST be resolved with the same modifiers and DC as the original, and the result MUST report both the failed and the rerolled outcome. | SRD 5.2.1, "Level 7: Countercharm", p. 33 | happy |
| FR-105 | The original save's circumstances carry over to the reroll, and the added Advantage is one more source. If the original had Disadvantage from any source and no Advantage, the reroll has neither and MUST be a single d20; if the original had Advantage and no Disadvantage, the reroll is still Advantage (no stacking, two d20s); if the original had both, any Disadvantage source still cancels the Advantage (single d20). Multiple Disadvantage sources still cancel against the added Advantage. The cancellation rule is SRD text; that the original circumstances carry over is an interpretation. | n/a (design decision); SRD 5.2.1, "Rules Glossary", "Advantage", p. 176; "They Don't Stack", p. 8 | unhappy (rule interaction) |
| FR-106 | Countercharm rerolls the whole save as a new roll, not one die of an existing roll, so the "Interactions with Rerolls" one-die rule does not apply. | SRD 5.2.1, "Level 7: Countercharm", p. 33; "Interactions with Rerolls", p. 8 | unhappy (rule interaction) |
| FR-086 | Countercharm MUST be rejected with a result, with no roll, when the Bard is below level 7 (levels 6 and 7 MUST be tested), when the `ConditionKind` set the effect applies contains neither Charmed nor Frightened, or when the original save did not fail. The three gates are decided by the pure `planCountercharm`, in that order, before any roll. | SRD 5.2.1, "Level 7: Countercharm", p. 33 | unhappy (boundary) |
| FR-087 | Countercharm MUST NOT spend a use of Bardic Inspiration or any spell slot; the pool MUST be unchanged. | SRD 5.2.1, "Level 7: Countercharm", p. 33 | unhappy (rule interaction) |

Countercharm roll-mode cases (FR-084, FR-105): `planCountercharm` returns the mode and the d20 count (`diceCount`) the use case requests. Original circumstances none: 2d20 (SRD 5.2.1, p. 33; p. 176). One Disadvantage or two Disadvantage: 1d20 (cancels; p. 8). One Advantage: 2d20 (no stacking). One Advantage and one Disadvantage: 1d20.

### 4.9 Magical Secrets, feats, and level 20

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-088 | From Bard level 10, whenever the Bard reaches a Bard level (including 10) and the Prepared Spells number increases, the new prepared spells MAY be chosen from the Bard, Cleric, Druid and Wizard spell lists (four lists, unlike Magical Discoveries, FR-076). | SRD 5.2.1, "Level 10: Magical Secrets", p. 33 | happy |
| FR-089 | Spells chosen through Magical Secrets MUST count as Bard spells for the Bard. | SRD 5.2.1, "Level 10: Magical Secrets", p. 33 | happy |
| FR-090 | From Bard level 10, a spell replaced from the prepared list MAY be replaced with a spell from any of the four lists. | SRD 5.2.1, "Level 10: Magical Secrets", p. 33 | happy |
| FR-091 | Below Bard level 10 a prepared-spell choice from the Cleric, Druid or Wizard list (outside Magical Discoveries) MUST be rejected. Levels 9 and 10 MUST be tested. | SRD 5.2.1, "Level 10: Magical Secrets", p. 33 | unhappy (boundary) |
| FR-092 | At Bard levels 4, 8, 12 and 16 the Bard MUST owe one feat choice (a `FeatId`; Ability Score Improvement or another feat). Applying the feat and qualification are feats-behaviour.md. | SRD 5.2.1, "Level 4: Ability Score Improvement", p. 32 | happy |
| FR-093 | A feat choice MUST be rejected with a result carrying `MissingContent(kind, id)` (content-source-port.md) when the `FeatId` is not found in the content source; no feat is owed at other Bard levels (for example 5). Qualification and application are decided by the domain service `FeatTakingRules.take` of feats-behaviour.md (FR-081: it composes `FeatQualification` and the ability increase), which `ApplyBardLevelGain` composes and never reaches through `TakeFeat`. The Bard builds the `FeatGrant` (`ClassFeatureFeat`, scope `AnyFeat`, feats-behaviour.md table G; at 19 the grant is the same scope). A `FeatTakingRejected(FeatRejection)` MUST be mapped to `BardChoiceProblem.featRejected(FeatRejection)` (FR-123). | SRD 5.2.1, "Level 4: Ability Score Improvement", p. 32 | unhappy (boundary) |
| FR-094 | At Bard level 19 the Bard MUST owe one feat choice (an Epic Boon feat or another feat); qualification and application are feats-behaviour.md. | SRD 5.2.1, "Level 19: Epic Boon", p. 33 | happy |
| FR-095 | At Bard level 20 the Bard MUST always have Power Word Heal and Power Word Kill prepared; they MUST NOT count against the Prepared Spells number (FR-057); a spell already prepared normally becomes always-prepared, with no duplicate entry and one choice freed at the next level gain. | SRD 5.2.1, "Level 20: Words of Creation", p. 33 | happy |
| FR-096 | `BardFeatures.allowsSecondTarget(SpellId spell)` (a pure query on the `BardLevel`) MUST return true exactly when the Bard is level 20 and the spell is Power Word Heal or Power Word Kill (the second creature is within 10 feet of the first target; geometry is the host's); false below level 20 or for any other spell. Levels 19 and 20 MUST be tested. | SRD 5.2.1, "Level 20: Words of Creation", p. 33 | unhappy (boundary) |

### 4.10 Level gain and package design

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-097 | Gaining a Bard level MUST grant exactly the features of Table 1 for the new level (and the chosen subclass's features of that level), and MUST update Hit Point Die count, Proficiency Bonus, Bardic Die, cantrip, prepared-spell and slot counts to the new row, the last three in the returned `SpellcastingState` (FR-123). | SRD 5.2.1, "Bard Class Features", p. 31 | happy |
| FR-098 | A level gain whose choices are incomplete or invalid (for example level 2 without Expertise, level 3 without a subclass, level 4 without a feat, a prepared-spell list of the wrong size) MUST be rejected with a result listing every problem found, not just the first, and MUST NOT change the state. | SRD 5.2.1, "Bard Class Features", p. 31 | unhappy (invalid input) |
| FR-099 | Gaining a level from Bard level 20 MUST be rejected with `ApplyBardLevelGainAlreadyMaxLevel`: the command holds the current level as a validated optional `BardLevel` (absent exactly when the character has no Bard entry, the multiclass entry of FR-121, whose next level is 1) and computes the next, so FR-022 stays constructor-only and nothing builds `BardLevel(21)`. A command whose level is present but whose `ClassLevels` has no Bard entry (or the reverse) MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error). | SRD 5.2.1, "Bard Features", p. 31 | unhappy (boundary) |
| FR-115 | A level gain MUST raise the Bard level by exactly one; there are no multi-level jumps (a host repeats it, with each level's choices, as character-aggregate.md FR-014 does for `levelUp`). | SRD 5.2.1, "Gaining a Level", p. 23; n/a (package design) | unhappy (boundary) |
| FR-100 | If the Bard class definition, a chosen feat or a chosen spell (kinds with a `ContentKey`) is not found by the `ContentSource` port (`lookup`, or `listKind` for spells filtered by `Spell.classes` containing the `ClassId`: bard, cleric, druid, wizard), the level gain MUST return `ApplyBardLevelGainContentMissing` holding `MissingContent(kind, id)` (content-source-port.md; decisions FR-018) for the first missing entry; it MUST NOT panic. An unknown `SubclassId` is `BardChoiceProblem.unknownSubclass` instead (FR-067). | n/a (package design, architecture.md section 4; content-source-port.md section 3) | unhappy (missing content) |
| FR-101 | If the `ContentSource` port returns a typed `*srderr.PortError`, the level gain MUST propagate it unchanged. | n/a (package design, architecture.md section 4) | unhappy (collaborator failure) |
| FR-102 | A level gain MUST NOT alter the input state; it returns the new `Character`, the new `SpellcastingState` and the events produced (`LevelGained`, `SubclassChosen`, `BardExpertiseGained`, and the preparation and cantrip events of spellcasting.md). Neither input is mutated. | n/a (package design, architecture.md section 3) | unhappy (immutability) |
| FR-117 | The Bard's creation choices MUST be a typed `BardCreationChoices` (skills, instruments, Starting Equipment option and instrument; FR-004 to FR-012) and its multiclass counterpart a typed `BardMulticlassChoices` (FR-015, FR-110), both `ClassCreationChoices` carried by `CreateCharacter` / `ApplyBardLevelGain` at multiclass entry (FR-121). Each MUST implement the two methods of character-aggregate.md FR-094: the seeding method returns the given `FeatureStates` with a full `BardicInspirationPool` written under the FR-118 key (FR-121), and `contentKeys` returns the `ContentKey`s the choices name, if any. The seeding method receives no ability scores, so the choices carry the Charisma `AbilityModifier`. The aggregate stores no Bard-specific field. | n/a (shared-types-and-ownership.md FR-024) | happy |
| FR-118 | Bardic Inspiration state (`BardicInspirationPool`) MUST live in the character's `FeatureStates` under one typed `FeatureStateKey[BardicInspirationPool]` defined here; a character with no Bard levels has no entry, and a missing entry for a character with Bard levels is a broken invariant (panic), never zero uses. The entry is created by FR-121. Tests construct the state directly, without the aggregate. | n/a (shared-types-and-ownership.md FR-022) | unhappy (state error) |
| FR-119 | Every Bard feature rule MUST be a pure domain function over plain values, needing no port and emitting no signal: grant, recovery and slot expenditure (FR-026 to FR-029, FR-038 to FR-045, FR-108, FR-109) and, for the features that roll, a plan step and an apply step. The plan step takes the state and trigger and returns a closed rejection or a plan holding what the use case must roll (the die or d20 count); it decides the level, subclass, condition and use-remaining gates, trigger eligibility (`notAFailure`, `ineligibleTrigger`, `saveNotFailed`, `ineligibleEffect`) and the one-hour expiry (FR-034, FR-035), so a rejection never reaches the roller. The apply step takes the plan and the numbers rolled and returns the new state, the adjusted result and events (spend-versus-no-spend, damage clamp, natural 1 and 20 through `adjustedBy`). The pairs are `planBardicInspirationUse` / `applyBardicInspirationDie`, `planCuttingWords` / `applyCuttingWords`, `planPeerlessSkill` / `applyPeerlessSkill`, `planCountercharm` / `resolveCountercharm`. The use cases `UseBardicInspiration`, `UseCuttingWords`, `UsePeerlessSkill`, `UseCountercharm` only call the plan step, roll through `dice.Roller`, call the apply step and record telemetry; they hold no rule. Only `ApplyBardLevelGain` loads content. These five are the only use cases of this spec. | n/a (shared-types-and-ownership.md FR-023; architecture.md section 4; testing.md section 3) | happy |
| FR-120 | The Bard code MUST NOT import from `combat`, from another class spec's code, or from `*test` packages. Condition identity comes from `content` (`ConditionKind`), a damage amount is a plain `int`, and every Bard result variant name is prefixed with its use case (section 6). | n/a (shared-types-and-ownership.md FR-005, FR-011, FR-013, FR-017) | unhappy (forbidden edge) |
| FR-121 | The Bard state MUST be created by the seeding method of FR-117: `BardicInspirationPool.initial(AbilityModifier charisma)` is a pure function returning a full pool whose maximum is the Charisma modifier with a minimum of 1 (FR-025), using the modifier the choices carry. No other spec calls it and this spec defines no creation command. Multiclass entry: the generic grant seeds no feature state, so `ApplyBardLevelGain` seeds the pool with `initial(charisma)` when the `Character` has no Bard entry in `ClassLevels` and the level gain is the first (command level absent, FR-099), writing it with the `levelUp` result; `BardMulticlassChoices` is validated then (FR-015, FR-110) and its seeding method is the same pure function (FR-117). Every later function that reads or writes the pool takes the current Charisma `AbilityModifier` and first applies `pool.withMaximumFrom(charisma)` (remaining uses are clamped to the new maximum, never raised), so the stored maximum never goes stale after an ability score change. `ApplyBardLevelGain` passes the Charisma modifier of the character it holds. | n/a (package design); SRD 5.2.1, "Number of Uses", p. 32 | happy |
| FR-123 | `ApplyBardLevelGainCommand` MUST carry: the `Character` (holds the validated Bard `ClassLevels` entry, the `FeatureStates` and the ability scores), the `SpellcastingState` the host holds beside it, the `HitPointGain` for the new level (`FixedHitPointGain` or `RolledHitPointGain`, derived-stats.md; the host supplies it, this use case rolls no Hit Point die, and a gain that does not match the Bard die is `levelUp`'s panic, character-aggregate.md FR-035), and the closed Bard choices for the level that follows (FR-115; `BardMulticlassChoices` at multiclass entry, FR-121), and for a feat level the `CharacterFeats`, `HeldFeatures`, `FeatChoices` and the chosen `FeatId` that `FeatTakingRules.take` needs (FR-093). The `Map<SpellId, Spell>` that spellcasting.md's validation needs is not a command field: this use case assembles it from the `ContentSource` port (chosen spells and the lists of FR-080), which is why it is the one use case that takes a `context.Context`. The result carries the new `Character`, the new `SpellcastingState`, the new `CharacterFeats` and the events. Order: (1) load the Bard `ClassDefinition`, feats and spells (FR-100), resolve the subclass from the definition (FR-067); (2) validate every Bard choice with the pure `BardChoiceProblems` over the spellcasting.md validation (FR-098); (3) `levelUp` (character-aggregate.md), yielding the new `ClassLevels` and `LevelGained`; (4) advance spellcasting over the new `ClassLevels`, reporting `spellsToChoose` and `cantripsToChoose` (spellcasting.md FR-047); (5) apply the additional and replaced prepared spells, then the cantrip changes; steps 4 and 5 go through the pure services of FR-124; (6) the Bard-only parts (Expertise, Bonus Proficiencies, `SubclassChosen`, the pool seeding of FR-121, the feat through `FeatTakingRules.take`, FR-093). Any rejection of a delegated step after step 2 MUST be mapped to a `BardChoiceProblem` (`levelUpRejected`, `spellcastingRejected`, `featRejected`) inside `ApplyBardLevelGainInvalidChoices`, and no new state is returned (FR-098). | n/a (package design; character-aggregate.md FR-035; spellcasting.md FR-047; derived-stats.md `HitPointGain`) | happy |
| FR-124 | Slot expenditure (FR-041), additional prepared spells, replacement and cantrip changes (FR-123 steps 4 and 5) MUST be consumed as pure published domain services or `SpellcastingState` transitions of spellcasting.md, as the Concentration service is, never by calling its use cases (`ExpendSpellSlot`, `AdvanceSpellcasting`, `ChooseAdditionalSpells`, `ChangePreparedSpells`; decisions FR-044). The services are the pure `SpellcastingState` transitions (spellcasting.md FR-021 to FR-024 for the slot, FR-028 for the shape of a transition) and `SpellcastingRules` / `SpellPreparation` (spellcasting.md FR-040 to FR-054, section 5), which the use cases wrap. Their outcome types are spellcasting.md's and are mapped, not re-exported; the exact member names are spellcasting.md's to publish (section 9). | n/a (shared-types-and-ownership.md FR-044) | unhappy (forbidden edge) |
| FR-125 | The `BardLevel`-to-`BardicDie` mapping (levels 1 to 4 d6, 5 to 9 d8, 10 to 14 d10, 15 to 20 d12) MUST equal the decoded Bardic Die column (`ColumnDice`) at all 20 levels, and each `BardicDie` MUST map to the kernel `Die` with the same number of sides. A scenario-level check through `BundledContentSource`, like FR-023. | SRD 5.2.1, "Bard Features", p. 31 | unhappy (rule interaction) |

### 4.11 Withdrawn ids

Data and grant requirements moved to their owners; ids are never reused.

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (FR-016) |
| FR-002 | Withdrawn: moved to content-classes.md (FR-017) |
| FR-003 | Withdrawn: moved to content-classes.md (FR-018) |
| FR-006 | Withdrawn: moved to content-classes.md (FR-021) |
| FR-009 | Withdrawn: moved to content-classes.md (FR-022) |
| FR-010 | Withdrawn: moved to content-classes.md (FR-024) |
| FR-011 | Withdrawn: moved to content-classes.md (FR-024) |
| FR-013 | Withdrawn: moved to character-creation.md (grant of Core Traits and level 1 features at creation) |
| FR-014 | Withdrawn: moved to level-advancement-and-multiclassing.md (grant on multiclassing) and content-classes.md (multiclass traits as data); choice validation stays in FR-015 and FR-110 |
| FR-016 | Withdrawn: moved to content-classes.md (FR-038) |
| FR-017 | Withdrawn: moved to content-classes.md (FR-041, FR-042) |
| FR-018 | Withdrawn: moved to content-classes.md (FR-033) |
| FR-019 | Withdrawn: moved to content-classes.md (FR-033) |
| FR-020 | Withdrawn: moved to content-classes.md (FR-033) |
| FR-021 | Withdrawn: moved to content-classes.md (FR-043, FR-045) |
| FR-024 | Withdrawn: moved to content-classes.md (FR-008) |
| FR-048 | Withdrawn: moved to content-spells.md (Bard class list verification against the pp. 33-35 list) |
| FR-063 | Withdrawn: moved to derived-stats.md (FR-015, FR-010, FR-066: Expertise doubling) |
| FR-103 | Withdrawn: moved to content-classes.md (decoder reports every problem in the Bard data; decisions FR-020, FR-035) |
| FR-112 to FR-114 | Never allocated: no requirement ever held these ids (new ids start at FR-115 after FR-111); nothing was withdrawn |

## 5. Model

### 5.1 Content data (owned by `content-classes.md`)

The Bard `ClassDefinition`, College of Lore `SubclassDefinition`, `ProgressionRow`
and `FeatureEntry` values are content-classes.md's, reached through
`BundledContentSource`; this spec requests no revision-table row and claims no
`schemaVersion`. Behaviour keys on the Bard `FeatureId`s (`deriveFeatureId`,
content-classes.md FR-052; owners `bard` and `college_of_lore`, for example
`bard_font_of_inspiration`, `college_of_lore_cutting_words`). The "Subclass
feature" cells (levels 6, 14) and repeated Ability Score Improvement and Expertise
rows are row markers or references, so feat choices (FR-092) and the second
Expertise choice (FR-061) key on the Bard level, not a `FeatureId`. The Bard spell
list is `Spell.classes` membership (content-spells.md).

### 5.2 Character behaviour (`character` package, pure domain files)

All value objects are immutable; state changes return new instances plus events.

- `BardLevel` (1-20): `MustBardLevel` panics, `NewBardLevel` returns an error (FR-022).
  `BardicDie` (d6, d8, d10, d12) maps to the kernel `Die` by sides and is checked against the decoded Bardic Die column (FR-125).
- `BardicInspirationPool`: remaining and maximum, enforcing
  `0 <= remaining <= maximum` (FR-025, FR-045). Methods `spend`, `restoreAll`,
  `restoreOne`, `restoreToTwo`, `withMaximumFrom(AbilityModifier)`; factory
  `initial(AbilityModifier)` (FR-121). A `FeatureState` under
  `FeatureStateKey[BardicInspirationPool]` (FR-118).
- `BardicInspirationGrant`: the die a recipient holds and a host-supplied marker
  of when it was granted. The elapsed `time.Duration` comes from the host in the
  command (the domain never reads a clock; the one-hour rule is
  `planBardicInspirationUse`, FR-034, FR-035). A recipient holds at most one
  grant (FR-029).
- `AdjustableD20Result`: closed, `AdjustableCheck(CheckResult)` |
  `AdjustableAttack(AttackRollResult)`; a `ForcedFailureResult` cannot be one.
  `adjustedBy(int)` delegates to the wrapped result (d20-tests.md FR-065).
- `InspirationRecipient` (FR-027); `BardRecoveryTrigger` closed
  `BardRecoveryRest(RestKind)` | `BardRecoveryInitiative`; `CuttingWordsTrigger`
  closed `CuttingWordsRoll(AdjustableD20Result)` | `CuttingWordsDamage(int)`
  (non-negative).
- Plans (FR-119): `BardicInspirationUsePlan`, `CuttingWordsPlan`,
  `PeerlessSkillPlan` (the die to roll) and `CountercharmPlan` (the `RollMode`
  and the failed save).
- `BardCreationChoices`, `BardMulticlassChoices` (FR-117), closed
  `BardChoiceProblem` (including `unknownSubclass`, `levelUpRejected`,
  `spellcastingRejected`, `featRejected`); `BardFeatures`: features at a level, the choices that
  level owes (`ExpertiseChoice`, `SubclassChoice`, `BardFeatChoice` (not feats-behaviour.md's `FeatChoices`), `SpellChoices`,
  `BonusProficiencyChoice`) and `allowsSecondTarget` (FR-096).
- Prepared spells, cantrips and slots are spellcasting.md's `PreparedSpells`
  (with `alwaysPrepared` for Magical Discoveries and Words of Creation) and
  `SpellcastingState`. Invariant: prepared count (excluding always-prepared)
  equals Table 1 after a level gain (FR-054, FR-057). No Bard spell container.
- Events: they implement `CharacterEvent`, carry `characterId` and form one closed family (decisions FR-019, FR-063; character-aggregate.md FR-120): `BardicInspirationGranted`,
  `BardicInspirationExpended`, `BardicInspirationExpired`,
  `BardicInspirationRestored`, `BardExpertiseGained`. Emitted, owned elsewhere:
  `LevelGained`, `SubclassChosen`, spellcasting.md's preparation and cantrip events.

Failures (architecture.md section 6): results for rejected cases, missing content
and invalid choices; panic for broken invariants; typed `*srderr.PortError`
only from ports, propagated unchanged (FR-036, FR-101). Expertise is
derived-stats.md data (its FR-010, FR-015, FR-066). Jack of All Trades depends on
the individual check, so it is the pure `jackOfAllTrades` (FR-064); its present
`Modifier` goes in `RollModifiers.others` (d20-tests.md FR-016).

## 6. Interface

Domain functions live in the `character` package (pure domain files), use cases in
the `character` package (use-case files). Commands carry state in; results carry new
state and events out. Each use case returns its own closed `<UseCase>Result`
with variants prefixed by the operation, no generic variant name exported
(decisions FR-017); missing content is a `MissingContent(kind, id)` field
(decisions FR-018). "Character state" means the aggregate with the pool in its
`FeatureStates` (FR-118) plus, for slots or prepared spells, the host-held
`SpellcastingState` (FR-041, FR-123); functions take the values they need
(validated `BardLevel`, subclass, pool, Charisma `AbilityModifier`), not the
aggregate. "Bard facts" is the Bard level, subclass, pool and Charisma modifier;
every function that holds a pool applies `withMaximumFrom(charisma)` first
(FR-121).

Pure domain functions (no port, no signal; FR-119), tested with plain values:

| Function | Input | Output | Covers |
|---|---|---|---|
| `GrantBardicInspiration` | Bard level, pool, Charisma modifier, `InspirationRecipient` | new pool, grant and event, or rejection (`selfTarget`, `noUsesRemaining`, `recipientAlreadyInspired`) | FR-026 to FR-029, FR-037 |
| `RecoverBardicInspiration` | Bard level, pool, Charisma modifier, `BardRecoveryTrigger` | new pool and event, or rejection (`belowRequiredLevel`, `nothingToRestore`) | FR-038 to FR-040, FR-043 to FR-045, FR-108 |
| `ExpendSlotForInspiration` | Bard level, pool, Charisma modifier, `SpellcastingState`, `SpellSlotPool`, `SlotLevel` | new pool, new `SpellcastingState`, events (`SpellSlotExpended`, `BardicInspirationRestored`), or rejection (`belowRequiredLevel`, `nothingToRestore`, `slotRejected(SlotOperationRejection)`) | FR-041, FR-042, FR-109 |
| `PlanBardicInspirationUse` | Bard facts, the `BardicInspirationGrant`, elapsed `time.Duration`, `AdjustableD20Result` | `BardicInspirationUsePlan` or rejection (`notAFailure`, or `expired` carrying the grant to discard and `BardicInspirationExpired`) | FR-033 to FR-035, FR-116 |
| `ApplyBardicInspirationDie` | the plan, the number rolled | adjusted result (`adjustedBy(rolled)`), expended grant and event | FR-030 to FR-032, FR-104 |
| `PlanCuttingWords` | Bard facts, pool, `CuttingWordsTrigger` | `CuttingWordsPlan` or rejection (`featureUnavailable`, `noUsesRemaining`, `ineligibleTrigger`) | FR-074 |
| `ApplyCuttingWords` | the plan, the number rolled | adjusted result or reduced damage (clamped at zero, flag set), new pool, event | FR-072, FR-073, FR-075, FR-104, FR-107 |
| `PlanPeerlessSkill` | Bard facts, pool, `AdjustableD20Result` | `PeerlessSkillPlan` or rejection (`featureUnavailable`, `noUsesRemaining`, `ineligibleTrigger`) | FR-083 |
| `ApplyPeerlessSkill` | the plan, the number rolled | adjusted result and, only when it becomes a success, the spent use (new pool, event) | FR-081, FR-082, FR-104 |
| `PlanCountercharm` | Bard level, failed saving-throw `CheckResult`, its `RollCircumstance`s, the `ConditionKind` set | `CountercharmPlan` (the `RollMode`) or rejection (`belowRequiredLevel`, `ineligibleEffect`, `saveNotFailed`) | FR-084, FR-086, FR-105 |
| `ResolveCountercharm` | the plan, the d20 face values | rerolled `CheckResult` (`CheckResult.resolve`), both outcomes; pool unchanged | FR-085, FR-087, FR-106 |
| `JackOfAllTrades` | Bard level or none, `ProficiencyBonus`, optional `Skill`, `SkillProficiencies`, whether the check already uses the Proficiency Bonus | optional `Modifier` | FR-064, FR-065 |

Use cases (the only ones; each needs a port; decisions FR-023, FR-119):

| Use case | Sync / takes ctx | Collaborators | Covers |
|---|---|---|---|
| `UseBardicInspiration` | sync | `dice.Roller`, `telemetry.Telemetry` | FR-030 to FR-037, FR-104, FR-116 (port interaction and telemetry) |
| `UseCuttingWords` | sync | `dice.Roller`, `telemetry.Telemetry` | FR-072 to FR-075, FR-107 |
| `UsePeerlessSkill` | sync | `dice.Roller`, `telemetry.Telemetry` | FR-081 to FR-083 |
| `UseCountercharm` | sync | `dice.Roller`, `telemetry.Telemetry` (no use-case collaborator; does not call `ResolveD20Test`) | FR-084 to FR-087, FR-105, FR-106 |
| `ApplyBardLevelGain` | context-aware | `ContentSource`, `telemetry.Telemetry` | FR-097 to FR-102, FR-115, FR-123, and the choice rules of 4.1, 4.5 to 4.7, 4.9 |

Only `ContentSource` does I/O, so only `ApplyBardLevelGain` returns a.
`UseCountercharm` is the outermost, and only, recorder of its `dice.Roller` failure
(section 7). The die features use `adjustedBy` (d20-tests.md FR-065, FR-066), not
`Roll` reroll operations. `ApplyBardLevelGain` is the Bard part of the
level-gain flow of level-advancement-and-multiclassing.md (FR-123 here), not a second
level-up path.

Results: each function and use case returns its own closed result with variants
`<Op>Granted` / `Recovered` / `Restored` / `Applied` / `Rerolled` / `LeveledUp` and
`<Op>Rejected(reason)` (reasons in the tables above; `UseBardicInspirationRejected`
carries the grant and `BardicInspirationExpired` for `expired`, FR-116).
`ApplyBardLevelGainResult` also has `ApplyBardLevelGainInvalidChoices(problems)`,
`ApplyBardLevelGainContentMissing(missing)` (class entry, `FeatId` or `SpellId`
only) and `ApplyBardLevelGainAlreadyMaxLevel`. The host extracts a Cutting Words
damage total from the `combat` damage roll. Casting is never a recovery trigger.
`ContentSource` and `dice.Roller` failures follow the port contracts (typed
`*srderr.PortError`); no new port, double or contract suite.

Exports: the character package API exports the use cases, domain functions, commands,
results, `BardLevel`, `BardicDie`, `BardicInspirationPool`, `BardicInspirationGrant`,
`AdjustableD20Result`, `BardCreationChoices`, `BardMulticlassChoices`, `BardFeatures`
and the Bard events; the module's public packages re-exports them once each.

## 7. telemetry.Telemetry

Signals follow shared-types-and-ownership.md FR-063: this spec declares no
metric. It uses the shared `srd5e.character.feature_used` counter (attributes
`srd5e.class` = `bard`, `srd5e.feature`) and the shared `.duration` and `.errors`
signals owned by character-aggregate.md. The domain, including every pure
function of FR-119 and the validators, is silent by design.

Failures: a missing class, feat or spell; a failing content or dice port; invalid
choices (an unknown subclass id is a choice problem, not missing content); an
illegal use. Pure rule outcomes are normal. A port error sets span error
status and increments the shared `.errors` with `error.type`
(`dice_roller_error`, `content_error`) once, by the outermost Bard
use case. `ApplyBardLevelGainContentMissing` sets error status on the use-case and
`load_content` spans with rejection `content_missing` and no `error.type`; every
other rejection is a `failure` outcome with no error status. Attributes are
low-cardinality (no character, recipient or content ids). Each use case
increments `feature_used` once per call and records duration with
`telemetry.Timer` (character-aggregate.md FR-118). `UseCountercharm`
emits no `srd5e.d20_test.*` signal. Monitor: any rise in `.errors`; any
`content_missing` on `level_gain`; `invalid_choices` above 20% of calls over 15
minutes. No logs (decisions FR-043).


Signals this spec owns (feed `doc/observability/signals.md`):

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| one span per use case: `srd5e.character.use_bardic_inspiration`, `.use_cutting_words`, `.use_peerless_skill`, `.use_countercharm`, `.apply_bard_level_gain` | span | `srd5e.class`, `srd5e.feature` (`bardic_inspiration`, `cutting_words`, `peerless_skill`, `countercharm`, `level_gain`), `srd5e.outcome` (`success`, `failure`), `srd5e.character.rejection` (`not_a_failure`, `expired`, `feature_unavailable`, `no_uses_remaining`, `ineligible_trigger`, `below_required_level`, `ineligible_effect`, `save_not_failed`, `invalid_choices`, `content_missing`, `already_max_level`; absent unless a rejection), `error.type` on a port failure | Trace one Bard operation | Spans with error status; follow a slow operation |
| `srd5e.character.load_content` (shared, owned by character-aggregate.md; not redeclared) | span (child of `apply_bard_level_gain`) | `srd5e.outcome` | The `ContentSource` lookups (class, feats, spells; a subclass is read from the loaded definition). Error status when a lookup returns not found | Compare with the parent span |

Result variant to attribute mapping:

| Result variant | `srd5e.outcome` | `srd5e.character.rejection` |
|---|---|---|
| `UseBardicInspirationApplied`, `UseCuttingWordsApplied`, `UsePeerlessSkillApplied`, `UseCountercharmRerolled`, `ApplyBardLevelGainLeveledUp` | `success` | absent |
| `...Rejected(reason)`, `ApplyBardLevelGainInvalidChoices` | `failure` | the reason (`invalid_choices` for `InvalidChoices`) |
| `ApplyBardLevelGainAlreadyMaxLevel` | `failure` | `already_max_level` |
| `ApplyBardLevelGainContentMissing` | `failure` | `content_missing`; error status on both spans; no `error.type` |
| a port panics `*srderr.PortError` | none (span error status) | absent; counted once on `.errors` by the outermost Bard use case |

All names and attribute keys are constants in the character context; the shared
keys `srd5e.outcome` and `error.type` are package `telemetry` constants.

## 8. Test plan

Locations (testing.md sections 3 and 10): `character_test` package (`*_test.go`) for unit rows
(plain values, no port or double), `character_test` use-case tests for use-case
rows, `scenarios/` for cross-context checks through the public API.
Withdrawn ids are tested in their owners' specs.

| Id | Level | Notes |
|---|---|---|
| FR-004, FR-005, FR-007, FR-008, FR-012, FR-015, FR-110, FR-122 | unit | Validators over decoded `SkillChoice` / `ClassToolProficiencyChoice` built directly and a `Set<ItemId>`: counts 0, 1, 2, 3, 4, duplicates, skill outside the option set, item not in the instrument set, already-held skill; all problems together |
| FR-022 | unit | `BardLevel` boundaries 0/1/20/21; `ParseX` (returns an error) returns a result |
| FR-023, FR-125 | scenario | `BundledContentSource` progression Proficiency Bonus equals derived-stats `ProficiencyBonus`, and the Bardic Die column equals `BardicDie` and the kernel `Die`, at all 20 levels |
| FR-025, FR-026, FR-027, FR-028, FR-029, FR-037, FR-119, FR-121 | unit | Number of Uses table; `GrantBardicInspiration` over plain values (self rejected, `Other` with and without a current grant); `initial` and `withMaximumFrom` (clamps, never raises); inputs unmodified, no signal |
| FR-030, FR-031, FR-032, FR-033, FR-034, FR-035, FR-104, FR-116 | unit | `planBardicInspirationUse` / `applyBardicInspirationDie` with real results and `adjustedBy`: using-a-die table, one-hour boundary (exactly 1 hour accepted, 1 hour and 1 tick expired), `notAFailure` and `expired` rejections, spend-versus-no-spend, natural 1 rows, total invariant holds |
| FR-030, FR-032, FR-036 | application | `UseBardicInspiration`: `dicetest.FixedRoller` records the die; no roll on a rejected plan; a failing roller propagates, returns no state, recorded once |
| FR-038, FR-039, FR-040, FR-043, FR-044, FR-045, FR-108 | unit | `RecoverBardicInspiration` tables (levels 4/5, 17/18, max below two) |
| FR-041, FR-042, FR-109 | unit | `ExpendSlotForInspiration` over a real `SpellcastingState`: Spellcasting and Pact Magic pools, events, rejections leave state unchanged; `SlotLevel` 0 and 10 at the `New` constructor (returns an error), 1 and 9 through the function |
| FR-046, FR-047, FR-049, FR-050, FR-051, FR-052, FR-053, FR-054, FR-055, FR-056, FR-057 | unit | Spellcasting ability; Bard row of spellcasting.md Table A reads Musical Instrument; Bard counts and level ceiling fed into spellcasting.md validation (one case per Bard parameter, prepared-spell level table, always-prepared exclusion); every problem reported |
| FR-058, FR-059 | unit | Slots through the spellcasting.md rest capability: Long Rest restores, Short Rest does not |
| FR-060, FR-061, FR-062, FR-064, FR-065 | unit | Expertise choices (levels 2 and 9); `jackOfAllTrades` table incl. round down, level 1, tool row, placement in `RollModifiers.others` |
| FR-066, FR-068, FR-069, FR-070, FR-071 | unit | Subclass rules (level 3 minimum, second choice rejected, feature levels); Bonus Proficiencies validation |
| FR-067 | unit + application | `BardChoiceProblems` reports `unknownSubclass`; `ApplyBardLevelGain` with a `FakeContentSource` and unknown subclass gives `InvalidChoices` (never `ContentMissing`); accepted choice emits `SubclassChosen` |
| FR-072, FR-073, FR-074, FR-075, FR-107 | unit | Cutting Words table; ineligible trigger; level, subclass and use gates; natural 20 stays a critical hit; damage clamp |
| FR-076, FR-077, FR-078, FR-079, FR-088, FR-089, FR-090, FR-091, FR-111 | unit | Magical Discoveries over three lists (Bard-only rejected, Bard-and-another accepted); Magical Secrets widening at levels 9/10/11/12; chosen spells count as Bard spells |
| FR-081, FR-082, FR-083 | unit | Peerless Skill table; uses unchanged on a remaining failure; natural 1 stays a miss; saving-throw trigger rejected |
| FR-084, FR-085, FR-086, FR-087, FR-105, FR-106 | unit | Countercharm roll-mode table (mode and `diceCount`), levels 6/7, ineligible effect, save not failed, same modifiers and DC, both outcomes, pool unchanged |
| FR-072, FR-075, FR-081, FR-083, FR-084, FR-085 | application | `UseCuttingWords`, `UsePeerlessSkill`, `UseCountercharm`: die or 1 or 2 d20 requested through `dicetest.FixedRoller`; no roll on a rejected plan; no `srd5e.d20_test.*` signal from Countercharm; a failing roller recorded once (error status, one `.errors` increment with `dice_roller_error`) |
| FR-080, FR-092, FR-093, FR-094, FR-100, FR-101 | application | `FakeContentSource` missing a spell entry, a class's spells, the class, a feat (result holds `MissingContent`) and a failing port; feat owed at 4, 8, 12, 16, 19, not at 5 (`FeatTakingRules` with real feats; rejection maps to `featRejected`) |
| FR-095, FR-096 | unit | Words of Creation at levels 19/20; `allowsSecondTarget` for the two spells, another spell, levels 19/20 |
| FR-097, FR-098, FR-099, FR-102, FR-115, FR-121, FR-123 | application | Levels 1 to 20 one at a time with complete and incomplete choices, `AlreadyMaxLevel` from 20; multiclass entry from another class seeds a full pool and validates `BardMulticlassChoices` (level present without a Bard entry panics with an error value); inputs unmodified; a `HitPointGain` supplied; delegation order to `levelUp` and the spellcasting steps of FR-124; one `LevelGained`; a delegated rejection maps into `InvalidChoices`, no state returned |
| FR-117, FR-118, FR-120, FR-124 | unit + architecture | Typed `FeatureStates` key built directly; missing entry for a Bard panics with an error value; import scan: no `combat`, sibling-class or `testing` import, no `ResolveD20Test` or other use case called from a Bard use case, no generic variant name exported; every Bard event is a `CharacterEvent` with `characterId` and the family is closed (exhaustive switch); `initial` seeded by both choices types through the FR-094 methods |
| Telemetry (observability.md section 9) | application | Per use case: counter, span name, attribute keys, no log; same behaviour with no-op and recording `telemetrytest.Recorder`; port error recorded once; `ContentMissing` sets error status on both spans with `content_missing` and no `error.type`; other rejections set no error status; `srd5e.outcome` only `success` or `failure`; one `feature_used` increment per call and duration through `telemetry.Timer`, no per-feature metric; constants pinned by a test |
| Scenario | scenario | One Bard from level 1 to 20 through the public packages (`github.com/Whealinghold/srd5e-go/...`) with `dicetest.FixedRoller` and `FakeContentSource` (or `BundledContentSource`): Bardic Inspiration across rests, Cutting Words, Peerless Skill |

- Doubles: `dicetest.FixedRoller`, `telemetrytest.Recorder`, `FakeContentSource` from
  the `*test` packages (decisions FR-038, FR-039); no new port, double or contract suite.
- Mutation testing: changed Bard lines in the `character` package (pure domain files) MUST reach
  score 80 with the pinned `gomu` command (testing.md section 8); the use
  cases SHOULD.
- Every live FR id is covered. Not covered: geometry and action economy (out of scope).

## 9. Open questions and decisions

No question is open; none is escalated.

Settled, one line each:

- **Bonus Action / Reaction cost** (was blocked on shared-types-and-ownership.md): no cost type or result field in `character`. The action-economy plug-in types (`BonusActionGrant`, `FeatureReaction`, initiative-and-actions.md FR-112 to FR-114) live in `combat`, downstream of `character`, so importing them would break rank order (FR-120). The cost is documented per operation (Bardic Inspiration grant: Bonus Action, p. 31; Cutting Words and Countercharm: Reaction, pp. 33, 35) and the host charges it, keyed by the Bard `FeatureId`s. SRD 5.2.1, "Bonus Action", p. 177; "Reaction", p. 186.
- **Feat qualification** (was blocked on feats-behaviour.md): `FeatTakingRules.take` and `FeatQualification` are published domain services that a `character` use case may compose (feats-behaviour.md FR-081); the Bard builds the `ClassFeatureFeat` / `AnyFeat` grant (its table G, Q17) and maps a rejection to `featRejected` (FR-093, FR-123).
- **Multiclass seeding of `BardicInspirationPool`** (was blocked on level-advancement-and-multiclassing.md): `ApplyBardLevelGain` seeds it at the first Bard level, as the Sorcerer spec does for its state (FR-121, FR-099); no upstream hook is needed. SRD 5.2.1, "As a Multiclass Character", p. 31.
- **Slot expenditure, added spells and replacement** (was blocked on spellcasting.md): spellcasting.md publishes `SpellcastingState` transitions and `SpellPreparation` as pure services, so FR-124 consumes them and the FR-044 claim stands; only their member names remain spellcasting.md's to fix (amendment returned).

## 10. Changelog impact

`CHANGELOG.md`, Unreleased, **Added**: "Bard class behaviour: Bardic Inspiration
(grant, use, and recovery), Bard spellcasting rules, Expertise, Jack of All
Trades, Countercharm, Cutting Words, Peerless Skill, Magical Secrets, Words of
Creation and level gain, with the Bard creation choices; the Bard class and
College of Lore data is recorded by content-classes.md. New signals: uses the
shared `srd5e.character.feature_used` counter and its shared `.duration` and
`.errors` companions (class `bard`), and adds the
`srd5e.character.load_content` span." Not **BREAKING** (new API and new
signals only).
