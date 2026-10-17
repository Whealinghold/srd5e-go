# Class: Wizard

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `character-aggregate.md`, `derived-stats.md`, `spellcasting.md`, `rests-and-hit-dice.md`, `level-advancement-and-multiclassing.md`, `character-creation.md`, `ability-scores.md`, `content-classes.md`, `content-spells.md`, `content-source-port.md`, `content-schema-and-decoder.md`, `observability-ports.md`, `errors-and-results.md` |

Binding decisions: [shared-types-and-ownership.md](shared-types-and-ownership.md);
where this spec conflicts with it, that spec wins. In-context order is its FR-016.
`feats-behaviour.md`, `hit-points-and-damage.md` and every other `combat` spec
are not dependencies (FR-013 there).

> Go notation used below. A name in lowerCamelCase (a field, command argument or enum member) stands for the exported PascalCase Go identifier (`attackerID` is the field `AttackerID`). A trailing `?` marks an optional value (the zero value or a `(T, bool)` pair). `[]T` is a slice; `set of T` is a collection without duplicates. A `(T, error)` pair replaces an accepted/rejected result pair: the nil-error case is the accepted case and the error is the typed rejection, satisfying `errors.Is(err, srderr.ErrInvalidInput)`; a parser reports every problem at once through `errors.Join`. Rule outcomes (refusals, not-possible stops) stay result values. A use case is a struct with `Run(ctx, cmd)`; its `error` is non-nil only for a port failure.

## 1. Purpose and scope

Specifies the **behaviour** of the Wizard class as SRD 5.2.1 presents it (pp. 77-82), inside the `character` context:
choices at creation and level up, the spellbook operations, Ritual Adept eligibility, Arcane Recovery, Scholar,
Memorize Spell, Spell Mastery and Signature Spells. The Evoker subclass features (p. 82) are specified in
`class-wizard-evoker.md`, split off for size (shared-types-and-ownership.md FR-062).

**Content is upstream** (shared-types-and-ownership.md FR-025, FR-005). The Wizard `ClassDefinition`, the Evoker
`SubclassDefinition`, the Features table (rows, `rowAt` bounds, features per level, feat grants, subclass features,
agreement with the Proficiency Bonus and spell tables) and the Core Wizard Traits belong to `content-classes.md`; this
spec adds only the rule maths of `WizardRules` (FR-018, FR-029). The Wizard spell list is derived from `Spell.Classes`
(`classSpellList`, content-spells.md FR-120; content-spells.md FR-016, FR-022), never stored. `Spell`/`SpellSchool`
are content-spells.md's (content-spells.md FR-012); `ClassLevels` is derived-stats.md's (derived-stats.md FR-058);
`RestKind` is rests-and-hit-dice.md's (rests-and-hit-dice.md FR-001); the `Spellbook` and prepared-list rules are
spellcasting.md's (this spec adds Wizard operations on that type, never a second one). Features resolved in `combat`
are returned only as value objects; no `combat` spec or type is used. Pattern (shared-types-and-ownership.md FR-023):
each feature is a pure domain function from (state, inputs) to (new state, events), with one use case where content is
needed; state is `WizardResources` in `FeatureStates` (FR-107).

**Out of scope:** general spellcasting and the multiclass Spell Slot rules (spellcasting.md; FR-020 and FR-021 record
only what a multiclass Wizard gains); the spells (pp. 104-175, content-spells.md); equipment definitions (lines and
gold are used as held, FR-009); the Ability Score Improvement and Epic Boon feats (`featGrantsAt`, content-classes.md
FR-049); the level 3 subclass choice (level-advancement-and-multiclassing.md, shared-types-and-ownership.md FR-020;
features held are FR-106); rest effects and `RestApplied` (rests-and-hit-dice.md; only `afterRest` is here);
`CreateCharacter` (character-creation.md; FR-105); recommended spells (advice, not rules); definitions of Expertise,
Ritual, Resistance, Immunity and the rests; Identify; Spell Scrolls (`spell-scroll-use.md`); other subclasses (the SRD has only Evoker).
Nothing outside SRD 5.2.1 was used.

## 2. Source

PDF page numbers equal printed page numbers (checked: PDF pages 77 and 82).

- SRD 5.2.1, "Wizard", "Core Wizard Traits", "Becoming a Wizard ...", "As a Level 1
  Character", "As a Multiclass Character", "Wizard Features", p. 77
- SRD 5.2.1, "Level 1: Spellcasting" (Cantrips begin p. 77; Spellbook, Spell Slots,
  Prepared Spells of Level 1+), "Level 1: Ritual Adept", "Level 1: Arcane Recovery",
  "Level 2: Scholar", "Level 3: Wizard Subclass" (begins), "Expanding and Replacing
  a Spellbook", p. 78
- SRD 5.2.1, "Level 3: Wizard Subclass" (continues), "Level 4: Ability Score
  Improvement", "Level 5: Memorize Spell", "Level 18: Spell Mastery", "Level 19:
  Epic Boon", "Level 20: Signature Spells", p. 79
- SRD 5.2.1, "Wizard Spell List", pp. 79-82 (content-owned; Special legend only); the Evoker, p. 82, is
  class-wizard-evoker.md's

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Wizard | `ClassID` `wizard` (content) | Definition: `ClassDefinition`, content-classes.md; Primary Ability is Intelligence. |
| Skill Proficiencies | Value (selection) | Choose 2 from a fixed list of 7. |
| Starting Equipment | Value (choice of A or B) | (A) 2 Daggers, Arcane Focus (Quarterstaff), Robe, Spellbook, Scholar's Pack, 5 GP; or (B) 55 GP. Held as `{"items": [{"name", "quantity"}], "gold": n}` (content-classes.md FR-024); no `ItemID` (section 9). |
| Spellbook | Value object (set of `SpellID`), spellcasting.md | The level 1+ Wizard spells known; operations FR-030 to FR-039 are `SpellbookRules` functions. The Wizard list is a derived view (FR-099). |
| `WizardSpellFacts` | Value object (input) | A spell's id, level, `SpellSchool`, `ritual`, `classes`, `castingTimes` from the content `Spell`, plus caller-supplied `dealsDamage`. |
| `WizardResources` | `FeatureState` | Feature state under `WizardResourcesKey` (`wizard_resources`; FR-107). |
| `WizardCreationChoices` | `ClassCreationChoices` | Level 1 choices carried in `CreateCharacter`; seeds `FeatureStates` (FR-105). |
| Class features | SRD names | Ritual Adept, Arcane Recovery (state: used), Scholar (choice), Memorize Spell, Spell Mastery (state: two spells), Signature Spells (state: two spells, uses). |
| Evoker | `SubclassID` `evoker` | Its features are class-wizard-evoker.md's; `WizardResources` holds the Overchannel use count. |

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. Withdrawn ids are
listed at the end of this section.

### 4.1 Core Wizard Traits (choices; the traits are content)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-004 | `ChooseWizardSkills` MUST accept exactly 2 distinct `Skill`s (shared-types-and-ownership.md FR-005) from Arcana, History, Insight, Investigation, Medicine, Nature and Religion (the Wizard definition's list, pinned by content-classes.md). | SRD 5.2.1, "Core Wizard Traits", p. 77 | happy |
| FR-005 | `ChooseWizardSkills` MUST reject a count other than 2 (0, 1, 3) with `WrongSkillCount(count)`, and a skill named twice with `DuplicateSkill(skill)`. | SRD 5.2.1, "Core Wizard Traits", p. 77 ("Choose 2") | unhappy (invalid input) |
| FR-006 | `ChooseWizardSkills` MUST reject a skill outside the seven (for example Athletics) with `SkillNotAllowed(skill)` naming the first offender. | SRD 5.2.1, "Core Wizard Traits", p. 77 | unhappy (invalid input) |
| FR-009 | `ChooseWizardEquipment` option A MUST resolve to the definition's option `A` exactly as held: the `{name, quantity}` lines (2 Daggers, an Arcane Focus (Quarterstaff), a Robe, a Spellbook, a Scholar's Pack) and 5 GP (content-classes.md FR-024). It MUST NOT invent `ItemID`s. | SRD 5.2.1, "Core Wizard Traits", p. 77 | happy |
| FR-010 | `ChooseWizardEquipment` option B MUST resolve to the definition's option `B`: no item lines and 55 GP. | SRD 5.2.1, "Core Wizard Traits", p. 77 | happy |
| FR-011 | `ChooseWizardEquipment` takes the option as a raw `string` and MUST reject any value other than `A` or `B` with `UnknownEquipmentOption(option)`, not an error. | SRD 5.2.1, "Core Wizard Traits", p. 77 ("Choose A or B") | unhappy (invalid input) |

### 4.2 Wizard rule maths over the Features table

Row contents and the `rowAt` bounds (content-classes.md FR-047) are not re-tested here.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | `WizardRules.HighestSpellLevel(ClassDefinition wizard, int level)` MUST be derived from the Spell Slots of the row of that level (the highest Spell Level with at least one slot) and equal the values in 4.2.2. | SRD 5.2.1, "Wizard Features", p. 77 | happy |

#### 4.2.2 Highest Spell Level with Spell Slots (one table-driven test per row)

Source for every row: SRD 5.2.1, "Wizard Features", p. 77. "New" marks a level at
which the highest Spell Level rises; `WizardRules.NewSpellLevelAt(wizard, level)`
reports this column (true at level 1 and wherever the highest level rises). The

| Label | Wizard level | Highest Spell Level | New | Path |
|---|---|---|---|---|
| H-01 | 1 | 1 | yes (first) | happy (boundary) |
| H-02 | 2 | 1 | no | happy |
| H-03 | 3 | 2 | yes | happy (boundary) |
| H-04 | 4 | 2 | no | happy |
| H-05 | 5 | 3 | yes | happy (boundary) |
| H-06 | 7 | 4 | yes | happy (boundary) |
| H-07 | 9 | 5 | yes | happy (boundary) |
| H-08 | 11 | 6 | yes | happy (boundary) |
| H-09 | 13 | 7 | yes | happy (boundary) |
| H-10 | 15 | 8 | yes | happy (boundary) |
| H-11 | 16 | 8 | no | happy |
| H-12 | 17 | 9 | yes | happy (boundary) |
| H-13 | 20 | 9 | no | happy (boundary: maximum level) |

### 4.3 Becoming a Wizard

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-019 | A character taking Wizard as a level 1 character MUST gain the Core Wizard Traits the definition lists (Primary Ability, Hit Point Die, Saving Throw Proficiencies, Skill Proficiencies, Weapon Proficiencies, Armor Training, Starting Equipment), the choices of FR-004, FR-009 and FR-010 (FR-005, FR-006 and FR-011 are rejections, not traits) and the level 1 features (Spellcasting, with its Spellcasting Ability, Ritual Adept, Arcane Recovery). | SRD 5.2.1, "As a Level 1 Character", p. 77; "Level 1: Spellcasting", p. 78 | happy |
| FR-020 | A character gaining Wizard as a multiclass MUST gain exactly the Hit Point Die and the level 1 features (Spellcasting, Ritual Adept, Arcane Recovery). | SRD 5.2.1, "As a Multiclass Character", p. 77 | happy |
| FR-021 | A multiclass Wizard MUST NOT gain the Saving Throw Proficiencies, Skill Proficiencies, Weapon Proficiencies or Starting Equipment. Its Spell Slots follow the multiclassing rules (spellcasting.md). | SRD 5.2.1, "As a Multiclass Character", p. 77 | unhappy (rule interaction) |
| FR-109 | Whether Wizard is the initial class is read from `ClassLevels` (the first entry, shared-types-and-ownership.md FR-008). FR-019 applies when Wizard is the first entry; FR-020 and FR-021 when it is not; no level history is read. | shared-types-and-ownership.md FR-008; SRD 5.2.1, "Becoming a Wizard ...", p. 77 | unhappy (rule interaction) |

### 4.4 Wizard spell list (derived) and cantrips

Per-level counts, Schools and the C, R and M flags are content facts
(content-spells.md FR-022, FR-024, FR-025, FR-097).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-027 | `ChooseWizardCantrips` MUST accept a choice whose size equals the Cantrips column for the Wizard level (3 at level 1) and whose members are distinct level 0 spells on the Wizard list (FR-099), returning `WizardCantrips` with a nil error. | SRD 5.2.1, "Level 1: Spellcasting", Cantrips, p. 77; "Wizard Features", p. 77 | happy |
| FR-028 | `ChooseWizardCantrips` MUST reject, by cause: a wrong size with `WrongCantripCount(allowed)`, a repeated cantrip with `DuplicateCantrip(id)`, a spell that is not level 0 with `NotACantrip(id)`, and a cantrip not on the Wizard list with `NotOnWizardList(id)`. | SRD 5.2.1, "Level 1: Spellcasting", Cantrips, p. 77 | unhappy (invalid input) |
| FR-029 | `WizardRules.CantripsGainedAt(wizard, level)` MUST return the rise of the Cantrips column at that level (3 at level 1, 1 at levels 4 and 10, 0 elsewhere), and `cantripsPending(wizard, level, knownCount)` MUST return the column minus `knownCount`, never below 0. Nothing is stored: a pending choice is derived and filled under FR-027 and FR-028. Cantrip replacement after a Long Rest is spellcasting.md's. | SRD 5.2.1, "Level 1: Spellcasting", Cantrips, p. 78; "Wizard Features", p. 77 | happy (boundary) |
| FR-099 | A spell MUST count as on the Wizard list if and only if its `WizardSpellFacts.Classes` (the content `Spell.Classes`) contains `ClassID` `wizard`, the same test as `classSpellList` (content-spells.md FR-120). No separate Wizard list MUST be stored or decoded in `character`. | content-spells.md FR-120; content-spells.md FR-022 | happy |

### 4.5 Spellbook

Operations are the `SpellbookRules` functions of section 5. Rejection variants are
those of `SpellbookRejection`; several causes at once are reported in that type's
order (FR-100), not the order listed below.
Copying from a Spell Scroll is `spell-scroll-use.md` FR-028 to FR-031; the host runs `CopySpellIntoBook` eligibility (FR-035, FR-036) before the scroll attempt (that spec FR-037).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-030 | `StartSpellbook` MUST accept exactly six distinct level 1 spells on the Wizard list and return `SpellbookChanged` with those six and a `SpellsAddedToSpellbook` event. | SRD 5.2.1, "Level 1: Spellcasting", Spellbook, p. 78 | happy |
| FR-031 | `StartSpellbook` MUST reject, by cause: a count other than six with `WrongSpellCount(allowed)`, a repeated spell with `DuplicateSpell(id)`, a cantrip with `IsACantrip(id)`, a spell above level 1 with `AboveHighestSpellLevel(id, highest)`, and a spell not on the list with `NotOnWizardList(id)`. | SRD 5.2.1, "Level 1: Spellcasting", Spellbook, p. 78 | unhappy (invalid input) |
| FR-032 | No `SpellbookRules` operation MUST return a `Spellbook` that contains a cantrip, a spell not on the Wizard list, or a repeated spell. An empty `Spellbook` MUST remain representable. spellcasting.md's `Spellbook` constructor is unchanged; the operations that receive facts enforce the Wizard rules. | SRD 5.2.1, "Level 1: Spellcasting", Spellbook ("contains the level 1+ spells you know"), p. 78 | unhappy (empty and minimal) |
| FR-033 | A Wizard level after 1 owes the spellbook exactly two Wizard spells (`WizardRules.SpellsOwedAtLevelUp(level)` is 2 for levels 2 to 20 and 0 at level 1; nothing is stored, because copied and found spells (FR-035) make the book size an unreliable record). `AddLevelUpSpells` MUST accept two distinct spells not already in the spellbook, each of a level up to the highest Spell Level at the new Wizard level (4.2.2), and return the spellbook with both added (cases in 4.5.1). | SRD 5.2.1, "Level 1: Spellcasting", Spellbook, p. 78 | happy |
| FR-034 | `AddLevelUpSpells` MUST reject, by cause: a request at Wizard level 1 with `FeatureNotAvailable`, a count other than two with `WrongSpellCount(allowed)`, a repeated spell with `DuplicateSpell(id)`, a spell already in the book with `AlreadyInSpellbook(id)`, a cantrip with `IsACantrip(id)`, a spell not on the list with `NotOnWizardList(id)`, and a spell above the highest Spell Level with `AboveHighestSpellLevel(id, highest)`. | SRD 5.2.1, "Level 1: Spellcasting", Spellbook, p. 78 | unhappy (invalid input) |
| FR-035 | `CopySpellIntoBook` MUST accept a spell of level 1 or higher on the Wizard list, not already in the spellbook, up to the highest Spell Level at the Wizard level; it MUST return the spellbook with the spell added and a cost of 2 hours and 50 GP for each level of the spell (cases in 4.5.2). | SRD 5.2.1, "Expanding and Replacing a Spellbook", Copying a Spell into the Book, p. 78 | happy |
| FR-036 | `CopySpellIntoBook` MUST reject, by cause: a cantrip with `IsACantrip(id)`, a spell not on the list with `NotOnWizardList(id)`, a spell above the highest Spell Level with `AboveHighestSpellLevel(id, highest)`, and a spell already in the book with `AlreadyInSpellbook(id)`. A rejection MUST carry no cost. | SRD 5.2.1, "Expanding and Replacing a Spellbook", p. 78 | unhappy (invalid input) |
| FR-037 | `CopySpellbook` MUST accept spells that are in the source spellbook and return a cost of 1 hour and 10 GP for each level of each copied spell (cases in 4.5.2); copying no spells MUST cost nothing and return an empty spellbook. | SRD 5.2.1, "Expanding and Replacing a Spellbook", Copying the Book, p. 78 | happy |
| FR-038 | `CopySpellbook` MUST reject a spell that is not in the source spellbook with `NotInSourceSpellbook(id)`. | SRD 5.2.1, "Expanding and Replacing a Spellbook", Copying the Book, p. 78 | unhappy (invalid input) |
| FR-039 | `ReplaceLostSpellbook` MUST transcribe the Wizard spells the character has prepared into a new spellbook at the Copying the Book cost (FR-037), leaving the rest empty to be filled by finding new spells (FR-035); with no prepared spells it MUST return an empty spellbook at no cost. | SRD 5.2.1, "Expanding and Replacing a Spellbook", p. 78 ("the same procedure" is Copying the Book: section 9) | unhappy (empty and minimal) |

#### 4.5.1 Level-up addition cases (one table-driven test per row)

Source for every row: SRD 5.2.1, "Level 1: Spellcasting", Spellbook, p. 78, and
"Wizard Features", p. 77. "Chosen" gives the Spell Levels of the two added spells.

| Label | New Wizard level | Highest Spell Level | Chosen | Already in book | Expected | Path |
|---|---|---|---|---|---|---|
| LU-01 | 2 | 1 | 1, 1 | no | accepted | happy |
| LU-02 | 2 | 1 | 1, 2 | no | rejected (level 2 above highest) | unhappy (boundary) |
| LU-03 | 3 | 2 | 1, 2 | no | accepted | happy (boundary: new highest level) |
| LU-04 | 3 | 2 | 2, 2 | no | accepted | happy |
| LU-05 | 3 | 2 | 3, 1 | no | rejected (level 3 above highest) | unhappy (boundary) |
| LU-06 | 5 | 3 | 3, 3 | no | accepted | happy |
| LU-07 | 17 | 9 | 9, 9 | no | accepted | happy (boundary: highest level) |
| LU-08 | 20 | 9 | 9 | no | rejected (one spell, not two) | unhappy (boundary) |
| LU-09 | 4 | 2 | 1, 1 with the same spell twice | no | rejected (repeats a spell) | unhappy (invalid input) |
| LU-10 | 4 | 2 | 1, 2 | one of them | rejected (already in the spellbook) | unhappy (invalid input) |
| LU-11 | 4 | 2 | 0, 1 (a cantrip) | no | rejected (cantrip) | unhappy (invalid input) |
| LU-12 | 1 | 1 | 1, 1 | no | rejected (no addition at level 1) | unhappy (state error) |

#### 4.5.2 Copy cost cases (one table-driven test per row)

Source for every row: SRD 5.2.1, "Expanding and Replacing a Spellbook", p. 78.

| Label | Operation | Spell Level | Wizard level | Expected cost | Path |
|---|---|---|---|---|---|
| CC-01 | copy a spell into the book | 1 | 1 | 2 hours, 50 GP | happy (boundary: lowest level) |
| CC-02 | copy a spell into the book | 3 | 5 | 6 hours, 150 GP | happy |
| CC-03 | copy a spell into the book | 5 | 9 | 10 hours, 250 GP | happy |
| CC-04 | copy a spell into the book | 9 | 17 | 18 hours, 450 GP | happy (boundary: highest level) |
| CC-05 | copy a spell into the book | 2 | 2 | rejected (level 2 above highest, 1) | unhappy (boundary) |
| CC-06 | copy a spell into the book | 0 | 5 | rejected (cantrip) | unhappy (boundary) |
| CC-07 | copy the book | 1 | any | 1 hour, 10 GP | happy (boundary: lowest level) |
| CC-08 | copy the book | 3 | any | 3 hours, 30 GP | happy |
| CC-09 | copy the book | 9 | any | 9 hours, 90 GP | happy (boundary: highest level) |
| CC-10 | copy the book | 1, 2 and 3 together | any | 6 hours, 60 GP | happy (cost sums per spell) |

### 4.6 Ritual Adept (level 1)

Ritual casting of a spellbook Ritual spell that is not prepared is spellcasting.md's
(FR-073, FR-114; `CastSpell` with `CastMode.Ritual` already passes it, with no
`CastingContext` field). `CheckRitualAdept` is a query that adds the one fact
spellcasting.md lacks: whether the spellbook can be read from.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-041 | `CheckRitualAdept` MUST return `RitualAdeptEligible` for a spell whose `ritual` flag is set and that is in the spellbook when the spellbook is available, whether or not the spell is prepared. | SRD 5.2.1, "Level 1: Ritual Adept", p. 78 | happy |
| FR-042 | `CheckRitualAdept` MUST return `NotARitualSpell` for a spell without the Ritual tag, even when it is in the spellbook. | SRD 5.2.1, "Level 1: Ritual Adept", p. 78 | unhappy (rule interaction) |
| FR-043 | `CheckRitualAdept` MUST return `NotInSpellbook` for a spell that is not in the spellbook, even when it has the Ritual tag. | SRD 5.2.1, "Level 1: Ritual Adept", p. 78 | unhappy (rule interaction) |
| FR-044 | `CheckRitualAdept` MUST return `SpellbookUnavailable` when the command says the spellbook cannot be read from. | SRD 5.2.1, "Level 1: Ritual Adept", p. 78 ("you must read from the book") | unhappy (state error) |
| FR-102 | `CheckRitualAdept` MUST NOT be a precondition `CastSpell` reads: the host calls it to decide whether to offer a Wizard Ritual cast, then calls `CastSpell` with `CastMode.Ritual` (casting-and-concentration.md FR-106, FR-114). This spec asserts no `CastingContext` field, and `CastSpell` does not reject a Ritual cast for an unreadable spellbook: that fact is the host's, so the query stays advisory (settled, section 9). | SRD 5.2.1, "Level 1: Ritual Adept", p. 78; "Casting without Slots", p. 104 | unhappy (rule interaction) |

### 4.7 Arcane Recovery (level 1)

Rejection variants are those of `ArcaneRecoveryRejection` (section 5).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-045 | The most Spell Slot levels Arcane Recovery can recover MUST equal half the Wizard level rounded up (1 at levels 1-2, 2 at levels 3-4, up to 10 at levels 19-20; cases in 4.7.1). | SRD 5.2.1, "Level 1: Arcane Recovery", p. 78 | happy |
| FR-046 | On finishing a Short Rest, `UseArcaneRecovery` MUST accept a choice of expended Spell Slots whose combined level does not exceed the maximum of FR-045, none of level 6 or higher, and return `ArcaneRecoveryApplied` with the recovered slots, Arcane Recovery marked used and an `ArcaneRecoveryUsed` event. | SRD 5.2.1, "Level 1: Arcane Recovery", p. 78 | happy |
| FR-047 | A choice that includes a Spell Slot of level 6 or higher MUST be rejected with `SlotLevelTooHigh(level)`, even when the combined level is within the maximum. | SRD 5.2.1, "Level 1: Arcane Recovery", p. 78 ("none of the slots can be level 6 or higher") | unhappy (boundary) |
| FR-048 | A choice that names more slots of a level than are expended MUST be rejected with `SlotNotExpended(level)`. | SRD 5.2.1, "Level 1: Arcane Recovery", p. 78 ("expended spell slots") | unhappy (invalid input) |
| FR-049 | A choice whose combined level exceeds the maximum MUST be rejected with `ExceedsMaximum(max)`. | SRD 5.2.1, "Level 1: Arcane Recovery", p. 78 | unhappy (boundary) |
| FR-050 | After use, Arcane Recovery MUST be rejected with `AlreadyUsed` until a Long Rest is finished; a Short Rest MUST NOT make it available again, and a Long Rest (`WizardResources.AfterRest`) MUST. | SRD 5.2.1, "Level 1: Arcane Recovery", p. 78 | unhappy (state error) |
| FR-051 | Arcane Recovery MUST be accepted only when the trigger is the `RestApplied` of a finished Short Rest (FR-110: `benefitsOf` `shortRest`, `source` `finished`); any other trigger, including a Long Rest or the Short Rest benefits of an interrupted Long Rest, MUST be rejected with `NotAfterShortRest`. | SRD 5.2.1, "Level 1: Arcane Recovery", p. 78 | unhappy (state error) |
| FR-052 | An empty choice MUST be accepted as a no-op that recovers nothing and MUST NOT mark Arcane Recovery used (settled, section 9). | SRD 5.2.1, "Level 1: Arcane Recovery" ("you can choose"), p. 78 | unhappy (empty and minimal) |

#### 4.7.1 Arcane Recovery cases (one table-driven test per row)

Source for every row: SRD 5.2.1, "Level 1: Arcane Recovery", p. 78. "Max" is half
the Wizard level rounded up. A rejected row asserts the variant of FR-047 to
FR-051 (AR-02, AR-05 and AR-12 `ExceedsMaximum`, AR-06 `SlotNotExpended`, AR-07
`AlreadyUsed`, AR-09 `SlotLevelTooHigh`, AR-14 and AR-15 `NotAfterShortRest`).

| Label | Wizard level | Max | Expended slots | Chosen slots | Used since Long Rest | Expected | Path |
|---|---|---|---|---|---|---|---|
| AR-01 | 1 | 1 | one level 1 | one level 1 | no | recovers one level 1 slot | happy (boundary: minimum level) |
| AR-02 | 2 | 1 | two level 1 | two level 1 | no | rejected (2 exceeds max 1) | unhappy (boundary) |
| AR-03 | 3 | 2 | one level 2 | one level 2 | no | recovers one level 2 slot | happy (rounding up from 1.5) |
| AR-04 | 4 | 2 | two level 1 | two level 1 | no | recovers two level 1 slots | happy |
| AR-05 | 4 | 2 | one level 2, one level 1 | one level 2, one level 1 | no | rejected (3 exceeds max 2) | unhappy (boundary) |
| AR-06 | 4 | 2 | one level 1 | two level 1 | no | rejected (only one expended) | unhappy (invalid input) |
| AR-07 | 4 | 2 | one level 2 | one level 2 | yes | rejected (already used) | unhappy (state error) |
| AR-08 | 9 | 5 | one level 5 | one level 5 | no | recovers one level 5 slot | happy (boundary: level 5 slot allowed) |
| AR-09 | 11 | 6 | one level 6 | one level 6 | no | rejected (level 6 slot) | unhappy (boundary) |
| AR-10 | 11 | 6 | one level 5, one level 1 | one level 5, one level 1 | no | recovers both (6 equals max) | happy (boundary: equals max) |
| AR-11 | 20 | 10 | two level 5 | two level 5 | no | recovers both (10 equals max) | happy (boundary: maximum level) |
| AR-12 | 20 | 10 | two level 5, one level 1 | two level 5, one level 1 | no | rejected (11 exceeds max 10) | unhappy (boundary) |
| AR-13 | 5 | 3 | none | none | no | no-op, stays available | unhappy (empty and minimal) |
| AR-14 | 5 | 3 | one level 3 | one level 3, trigger is a Long Rest | no | rejected (not a Short Rest) | unhappy (state error) |
| AR-15 | 5 | 3 | one level 3 | one level 3, trigger is `shortRest` with `source` `longRestInterrupted` | no | rejected (`NotAfterShortRest`; not a finished Short Rest) | unhappy (state error) |

### 4.8 Scholar (level 2)

Rejection variants are those of `ScholarRejection` (section 5).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-053 | `ChooseScholarExpertise` MUST accept, at Wizard level 2 or higher, one skill in which the character has proficiency from Arcana, History, Investigation, Medicine, Nature and Religion, and return `ScholarExpertise` with a nil error with the Expertise as an active effect on that skill. | SRD 5.2.1, "Level 2: Scholar", p. 78 | happy |
| FR-054 | A Scholar choice naming a skill outside the six (for example Insight, which is on the level 1 list but not this one) MUST be rejected with `SkillNotEligible(skill)`. | SRD 5.2.1, "Level 2: Scholar", p. 78 | unhappy (invalid input) |
| FR-055 | A Scholar choice naming a listed skill without proficiency MUST be rejected with `NotProficient(skill)`. | SRD 5.2.1, "Level 2: Scholar", p. 78 ("in which you have proficiency") | unhappy (invalid input) |
| FR-056 | A Scholar choice below Wizard level 2 MUST be rejected with `FeatureNotAvailable`, and a second choice once one is held with `Already` with a nil error. | SRD 5.2.1, "Level 2: Scholar", p. 78 | unhappy (state error) |

### 4.9 Subclass features held

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-106 | The subclass features a Wizard holds MUST be `SubclassDefinition.FeaturesAtOrBelow(wizardLevel)` (content-classes.md FR-075; for the Evoker the entries at levels 3, 6, 10 and 14, none below level 3). This spec defines no function for it. The Evoker use cases (class-wizard-evoker.md FR-001 to FR-011) MUST decide `FeatureNotAvailable` by testing that the `FeatureEntry` of the feature is in that list for the subclass they were given. | SRD 5.2.1, "Level 3: Wizard Subclass", pp. 78-79; content-classes.md FR-075; shared-types-and-ownership.md FR-020, FR-025 | happy |

### 4.10 Memorize Spell (level 5)

The prepared-list rules (replacement trigger, Wizard level 5, spell in the
spellbook) are spellcasting.md FR-050 and FR-091. `MemorizeSpell` reuses those pure domain rules from the spellcasting
package, adds the level checks and never calls the `ChangePreparedSpells` use case (shared-types-and-ownership.md FR-044).
Rejection variants: `MemorizeRejection` (section 5).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-061 | On finishing a Short Rest, from Wizard level 5, `MemorizeSpell` MUST accept the replacement of exactly one prepared level 1+ Wizard spell with another level 1+ spell from the spellbook that is of a level for which the character has Spell Slots and is not already prepared, and return `MemorizeApplied` with the prepared list and a `SpellMemorized` event. | SRD 5.2.1, "Level 5: Memorize Spell", p. 79; "Level 1: Spellcasting", Prepared Spells of Level 1+, p. 78 | happy |
| FR-062 | `MemorizeSpell` MUST reject, by cause: a new spell that is a cantrip with `NewSpellIsCantrip(id)`, not in the spellbook with `NewSpellNotInSpellbook(id)`, above the highest Spell Level with `NewSpellAboveHighestLevel(id, highest)`, or already prepared with `NewSpellAlreadyPrepared(id)`; and a spell to replace that is not currently prepared with `NotPrepared(id)`. | SRD 5.2.1, "Level 5: Memorize Spell", p. 79 | unhappy (invalid input) |
| FR-063 | `MemorizeSpell` MUST reject a Wizard level below 5 with `FeatureNotAvailable` and a trigger other than a finished Short Rest (FR-110) with `NotAfterShortRest`. The command carries one spell to replace and one new spell, so "more than one replacement" cannot be expressed and has no variant. | SRD 5.2.1, "Level 5: Memorize Spell", p. 79 | unhappy (state error) |
| FR-064 | A spell that is always prepared through Spell Mastery or Signature Spells MUST be reported by `MemorizeSpell` as `AlwaysPreparedNotReplaceable(id)`, mapping spellcasting.md FR-046; the Wizard's contribution is only the always-prepared set (FR-070). `MemorizeSpell` reuses the pure prepared-list domain rules published in the spellcasting package and never calls the `ChangePreparedSpells` use case (shared-types-and-ownership.md FR-044). | SRD 5.2.1, "Level 5: Memorize Spell", p. 79; "Always-Prepared Spells", p. 104 | unhappy (rule interaction) |

### 4.11 Spell Mastery (level 18)

Rejection variants: `SpellMasteryRejection` (section 5). A spell "with a casting
time of an Action" has at least one option in `Spell.CastingTimes` equal to
`CastingTime.MagicAction()` (content-spells.md FR-026).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-065 | At level 18 `ChooseSpellMastery` MUST accept exactly one level 1 spell and one level 2 spell in the spellbook, each with a casting time of an Action, return `SpellMastery` with a nil error, and expose the two spells as always prepared through `WizardAlwaysPrepared` (FR-070). | SRD 5.2.1, "Level 18: Spell Mastery", p. 79 | happy |
| FR-066 | `ChooseSpellMastery` MUST reject, by cause: a spell not in the spellbook with `NotInSpellbook(id)`, the same spell twice with `DuplicateSpell(id)`, levels that are not one of 1 and one of 2 (two level 1 spells, a level 3 spell) with `WrongSpellLevels(levels)`, and a casting time that is not an Action with `CastingTimeNotAction(id)`. | SRD 5.2.1, "Level 18: Spell Mastery", p. 79 | unhappy (invalid input) |
| FR-067 | `ChooseSpellMastery` MUST reject a choice below Wizard level 18 with `FeatureNotAvailable`, and a second choice while one is held (other than the replacement of FR-069) with `Already` with a nil error. | SRD 5.2.1, "Level 18: Spell Mastery", p. 79 | unhappy (state error) |
| FR-068 | `SpellMastery.CastRequirement(spell, castAtLevel)` MUST report a Spell Mastery spell as castable without expending a Spell Slot at its lowest level, and as requiring a Spell Slot when cast at a higher level; the use is unlimited and MUST NOT be tracked. | SRD 5.2.1, "Level 18: Spell Mastery", p. 79 | happy |
| FR-069 | After finishing a Long Rest `ReplaceSpellMastery` MUST accept the replacement of one of the two Spell Mastery spells with an eligible spell of the same level (in the spellbook, casting time of an Action, not the other chosen spell). It MUST reject, by cause: a trigger other than the `RestApplied` of a Long Rest (`benefitsOf` `longRest`, FR-110) with `NotAfterLongRest`, no Spell Mastery held with `NotHeld`, a spell to replace that is not one of the two with `NotAMasterySpell(id)`, a new spell of a different level with `WrongSpellLevels(levels)`, and an ineligible new spell with `NotInSpellbook(id)`, `DuplicateSpell(id)` or `CastingTimeNotAction(id)`. | SRD 5.2.1, "Level 18: Spell Mastery", p. 79 | unhappy (state error) |
| FR-070 | `WizardAlwaysPrepared.Of(resources)` MUST return the ids of the Spell Mastery and Signature Spells spells, which the host passes as the always-prepared set to spellcasting.md; counting them outside the prepared limit and refusing their replacement is spellcasting.md FR-045 and FR-046. | SRD 5.2.1, "Level 18: Spell Mastery" ("always have those spells prepared"), p. 79; "Always-Prepared Spells", p. 104 | unhappy (rule interaction) |

### 4.12 Signature Spells (level 20)

Rejection variants: `SignatureSpellsRejection` (section 5).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-071 | At level 20 `ChooseSignatureSpells` MUST accept exactly two distinct level 3 spells in the spellbook, return `SignatureSpells` with a nil error, and expose both as always prepared through `WizardAlwaysPrepared` (FR-070). | SRD 5.2.1, "Level 20: Signature Spells", p. 79 | happy |
| FR-072 | `ChooseSignatureSpells` MUST reject, by cause: a count other than two with `WrongSpellCount(allowed)`, the same spell twice with `DuplicateSpell(id)`, a spell not of level 3 with `NotLevelThree(id)`, and a spell not in the spellbook with `NotInSpellbook(id)`. | SRD 5.2.1, "Level 20: Signature Spells", p. 79 | unhappy (invalid input) |
| FR-073 | `SignatureSpells.CastRequirement(spell, castAtLevel, resources)` MUST report each Signature Spell as castable once at level 3 without a Spell Slot; once the free cast is recorded, a second cast of the same spell before a Short or Long Rest MUST need a Spell Slot, while the other Signature Spell stays free. | SRD 5.2.1, "Level 20: Signature Spells", p. 79 | unhappy (state error) |
| FR-074 | `WizardResources.AfterRest` for a Short Rest or a Long Rest MUST make both Signature Spells free again; casting one at a level above 3 MUST require a Spell Slot and MUST NOT use up the free cast. | SRD 5.2.1, "Level 20: Signature Spells", p. 79 | happy (boundary) |
| FR-075 | `ChooseSignatureSpells` MUST reject a choice below Wizard level 20 with `FeatureNotAvailable`, and a second choice once one is held with `Already` with a nil error (the SRD gives no way to change it; settled, section 9). | SRD 5.2.1, "Level 20: Signature Spells", p. 79 | unhappy (state error) |

### 4.13 Overchannel use count (state held for the Evoker)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-090 | `WizardResources.AfterRest` for a Long Rest MUST reset the Overchannel use count so that the next use is a first use; for a Short Rest it MUST NOT. The feature itself is class-wizard-evoker.md's. | SRD 5.2.1, "Level 14: Overchannel", p. 82 | unhappy (state error) |

### 4.14 Cross-cutting behaviour

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-091 | Every Wizard state transition MUST return a new state and its events (per transition in section 5) and MUST NOT mutate the input state; exposed collections, the spellbook and active effects MUST be read-only (accessors return copies). Events are past-tense value objects that implement `CharacterEvent` directly, the open interface of character-aggregate.md FR-085 (outside its closed `CharacterRecordEvent` family), and so `DomainEvent` (shared-types-and-ownership.md FR-019). Every Wizard event MUST carry the `CharacterID`, so every command that produces an event carries one (section 6); the host supplies it, including at creation. | doc/rules/architecture.md §3; character-aggregate.md FR-085 | unhappy (immutability) |
| FR-093 | When the `wizard` class definition is absent from the `content.Source`, every use case that loads it MUST return its own `<UseCase>ContentMissing` variant carrying `MissingContent(kind, id)` (shared-types-and-ownership.md FR-017, FR-018), not an error. Subclass lookup is class-wizard-evoker.md FR-018. A spell absent from content is the caller's concern: the caller resolves spells and passes `WizardSpellFacts`; a spell id with no facts in the command is a programmer error (`srderr.ErrInvalidInput`). | doc/rules/architecture.md §4, §6; content-classes.md FR-074 | unhappy (missing content) |
| FR-094 | When the `content.Source` returns its typed `*srderr.PortError`, every content-loading use case MUST propagate it unchanged and MUST NOT return a partial result. | doc/rules/architecture.md §4 | unhappy (collaborator failure) |
| FR-095 | A use case whose command carries `ClassLevels` (all except `ChooseWizardSkills` and `ChooseWizardEquipment`, which are creation choices) applied to a character with no Wizard levels MUST return its own `<UseCase>NotAWizard` variant, not an error. | SRD 5.2.1, "Wizard Class Features", p. 77 | unhappy (state error) |
| FR-096 | Every use case MUST produce identical results with the no-op `telemetry.Telemetry` and with a recording one. | doc/rules/observability.md §9 | unhappy (determinism) |
| FR-097 | Each use case call MUST emit exactly one span named per section 7, increment `srd5e.character.feature_used` once and record `srd5e.character.feature_used.duration` once (character-aggregate.md FR-118, FR-119), with the attributes of section 7. A rejection or other returned failure (an error, `XContentMissing`, `XSubclassUnknown`, `XNotAWizard`) MUST set `srd5e.outcome` to `failure` and `srd5e.character.rejection` to the snake_case variant, MUST NOT set `error.type` (errors-and-results.md FR-043) and MUST NOT increment `srd5e.character.feature_used.errors`; the span error status follows doc/rules/observability.md §6 (set for content missing, not for a rule rejection). An accepted result sets `srd5e.outcome` to `success`. A port failure (returned `*srderr.PortError`) MUST be recorded once and then propagated: span error status, one `feature_used` increment with `srd5e.outcome=failure`, no `error.type` and no `srd5e.character.rejection`, one `srd5e.character.feature_used.errors` increment with `error.type`, and no log (character-aggregate.md FR-118). Exactly one `LevelDebug` summary log MUST be emitted for every call that returns a result and none when an error propagates. | doc/rules/observability.md §5, §7, §9; errors-and-results.md FR-043 | unhappy (collaborator failure) |
| FR-098 | Every signal name and attribute key and value in section 7 MUST be a constant in the character telemetry constants, and a test MUST pin each string so a rename fails. Shared values (`srd5e.outcome`, `error.type`) are those of shared-types-and-ownership.md FR-041. | doc/rules/observability.md §5, §9 | happy |
| FR-100 | When several causes of one rejection type apply to a request, the use case MUST report the first variant in the order of that type's row in section 5, and for per-spell or per-skill causes the first offending item in command order. | doc/rules/testing.md §4 | unhappy (invalid input) |
| FR-105 | `WizardCreationChoices` MUST implement both methods of `ClassCreationChoices` (character-aggregate.md FR-094). The seeding method returns the given `FeatureStates` seeded with the initial `WizardResources` under `WizardResourcesKey` (nothing used, nothing chosen). The key-listing method returns an empty read-only (accessors return copies) list, because the Wizard's choices name no `ContentKey`. `CreateCharacter` (character-creation.md FR-102, FR-103) calls only those two methods. Validation is the pure `NewWizardCreationChoices(...)` (section 5), which the host calls before building the `CreateCharacter` command: it takes the Wizard `ClassDefinition`, the two skills, the equipment option string, the three cantrips and the six spellbook spells as `WizardSpellFacts`, reuses the domain functions behind FR-004 to FR-011, FR-027 and FR-030, and returns the closed `WizardCreationParse`: `WizardCreation` with a nil error or an error (WizardCreationRejection) naming the first failing part under FR-100 (`skills`, `equipment`, `cantrips`, `spellbook`) with that part's rejection. A rejection never travels through `CreateCharacter`, which needs no rejection channel. | character-aggregate.md FR-094; shared-types-and-ownership.md FR-024; SRD 5.2.1, "As a Level 1 Character", p. 77 | unhappy (invalid input) |
| FR-107 | `WizardResources` MUST implement the aggregate's `FeatureState` and be read and written under `WizardResourcesKey`, a `FeatureStateKey[WizardResources]` named `wizard_resources` (matching `[a-z][a-z0-9_]*`, character-aggregate.md FR-093). A use case whose command lists `WizardResources` (section 6) takes the current state, and one that changes it returns the new one in its result. This spec defines the state and the key; the aggregate defines the slot and the read/write API. A character with Wizard levels and no stored `WizardResources` reads as the initial state (no uses spent, nothing chosen). | shared-types-and-ownership.md FR-022, FR-023; character-aggregate.md FR-093 | happy |
| FR-110 | The host feeds rests in as the `RestApplied` event of rests-and-hit-dice.md (FR-036, FR-060, FR-061; its Q-1): `WizardResources.AfterRest` takes `RestApplied.BenefitsOf` (any `source`, because the resets are recharges) and no generic `FeatureState` refresh hook is needed. The windows take the whole `RestApplied`: Arcane Recovery and Memorize Spell (Table W W-14, W-3) accept only `benefitsOf` `shortRest` with `source` `finished`, so an interrupted Long Rest (`longRestInterrupted`) recharges Signature Spells but opens neither window nor resets Arcane Recovery (R-28 to R-30); Spell Mastery replacement (W-2) accepts only `benefitsOf` `longRest`. The wrong trigger gives the use case's own `NotAfter...Rest` variant. | rests-and-hit-dice.md FR-061, Table W, Table R; SRD 5.2.1, "Level 1: Arcane Recovery", p. 78 | unhappy (rule interaction) |
| FR-108 | Every exported result variant MUST be prefixed with its use case name (`StartSpellbook` with a nil error, an error, `StartSpellbookContentMissing`, `StartSpellbookNotAWizard`) and every rejection variant with its rejection type stem (`SpellbookNotOnWizardList`, `ArcaneRecoveryAlreadyUsed`). Parse results are named by their type (`SpellMastery` with a nil error, an error, `WizardCreation` with a nil error). The short names in this spec denote the payloads and variants of section 5; section 6 gives the exported names. No generic name (`Applied`, `Rejected`, `ContentMissing`, `NotInSpellbook`) is exported, and no name collides with another spec. | shared-types-and-ownership.md FR-017, FR-021 | unhappy (name collision) |

### Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (FR-002), Primary Ability |
| FR-002 | Withdrawn: moved to content-classes.md (FR-002), Hit Point Die |
| FR-003 | Withdrawn: moved to content-classes.md (FR-002), Saving Throw Proficiencies |
| FR-007 | Withdrawn: moved to content-classes.md (FR-002), Weapon Proficiencies |
| FR-008 | Withdrawn: moved to content-classes.md (FR-022), Armor Training |
| FR-012 | Withdrawn: moved to content-classes.md (FR-035), Spellcasting Ability and Focus agreement; `SpellcastingAbilityMismatch` no longer exists |
| FR-013 | Withdrawn: moved to content-classes.md (FR-035), the 20 Features table rows |
| FR-014 | Withdrawn: moved to content-classes.md (FR-047), `rowAt` bounds |
| FR-015 | Withdrawn: moved to content-classes.md (FR-035), Proficiency Bonus agreement |
| FR-016 | Withdrawn: moved to content-classes.md (FR-035), spell column agreement |
| FR-017 | Withdrawn: moved to content-classes-data.md (FR-090), no `WizardProgression`, no content-invalid result here |
| FR-022 | Withdrawn: moved to content-classes.md (FR-048), `entriesGainedAt`, `rowFeatures` |
| FR-023 | Withdrawn: moved to content-spells.md (FR-022), spells per level of the Wizard list |
| FR-024 | Withdrawn: moved to content-spells.md (FR-024), School and Special flags per entry |
| FR-025 | Withdrawn: moved to content-spells.md (FR-024), sample entries, False Life en dash p. 79 |
| FR-026 | Withdrawn: moved to content-schema-and-decoder.md (FR-012), spell list decode failures |
| FR-040 | Withdrawn: moved to content-equipment.md (FR-074), Spellbook item facts |
| FR-057 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-041), `Subclass` with a nil error; features held are FR-106 |
| FR-058 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-042), an error |
| FR-059 | Withdrawn: moved to content-classes.md (FR-049), `featGrantsAt`, Ability Score Improvement at 4, 8, 12, 16 |
| FR-060 | Withdrawn: moved to content-classes.md (FR-049), `featGrantsAt`, Epic Boon at 19 |
| FR-076 | Withdrawn: moved to content-classes.md (FR-075), Evoker `SubclassDefinition` and five features; consumed through FR-106 |
| FR-077 | Withdrawn: moved to class-wizard-evoker.md (FR-001) |
| FR-078 | Withdrawn: moved to class-wizard-evoker.md (FR-002) |
| FR-079 | Withdrawn: moved to class-wizard-evoker.md (FR-003) |
| FR-080 | Withdrawn: moved to class-wizard-evoker.md (FR-004) |
| FR-081 | Withdrawn: moved to class-wizard-evoker.md (FR-005) |
| FR-082 | Withdrawn: moved to class-wizard-evoker.md (FR-006) |
| FR-083 | Withdrawn: moved to class-wizard-evoker.md (FR-007) |
| FR-084 | Withdrawn: moved to class-wizard-evoker.md (FR-008) |
| FR-085 | Withdrawn: moved to class-wizard-evoker.md (FR-009) |
| FR-086 | Withdrawn: moved to class-wizard-evoker.md (FR-010) |
| FR-087 | Withdrawn: moved to class-wizard-evoker.md (FR-011) |
| FR-088 | Withdrawn: moved to class-wizard-evoker.md (FR-012) |
| FR-089 | Withdrawn: moved to class-wizard-evoker.md (FR-013) |
| FR-092 | Withdrawn: moved to class-wizard-evoker.md (FR-017) |
| FR-101 | Withdrawn: moved to content-classes.md (FR-035), no Wizard content cross-check, so no `ContentInvalid` variant |
| FR-103 | Withdrawn: moved to class-wizard-evoker.md (FR-015) |
| FR-104 | Withdrawn: moved to class-wizard-evoker.md (FR-016) |

## 5. Model

Domain (`character/`, pure, synchronous). Types owned by other specs are consumed, not redefined.

- **`WizardSpellFacts`**: record `(SpellID id, int level, SpellSchool school, bool ritual, set of ClassID classes,
  []CastingTime castingTimes, bool dealsDamage)`, built by the caller from the content `Spell` plus
  `dealsDamage`. "An Action" is a `castingTimes` option equal to `CastingTime.MagicAction()`.
- **`WizardRules`**: stateless functions of a Wizard `ClassDefinition` (or `ProgressionRow`) and an `int` level,
  bounds delegated to `ClassDefinition.RowAt`: `highestSpellLevel`, `newSpellLevelAt`, `cantripsGainedAt`,
  `cantripsPending`, `spellsOwedAtLevelUp`. No `WizardProgression` exists.
- **`SpellbookRules`**: stateless functions over spellcasting.md's `Spellbook` (`start`, `addLevelUp`, `copyIn`,
  `copyOut`, `replaceLost`) taking the `CharacterID` where an event is
  produced, the `WizardSpellFacts` they touch and the plain `int highestSpellLevel` (`copyOut` takes no level and no
  `CharacterID`; the Evoker's additions are class-wizard-evoker.md's). They return the unexported closed `SpellbookOutcome`: `SpellbookChanged(Spellbook,
  SpellCopyCost? cost, []CharacterEvent)` | an error (SpellbookRejection). `SpellCopyCost` is a value
  object (hours, GP).
- **`NewArcaneRecoveryChoice(chosen, expended, maxLevels)`** (slot level to count maps) returns closed
  `ArcaneRecoveryChoiceParse` (`...Parsed(choice)` | `...ParseRejected(ArcaneRecoveryRejection)`);
  `ArcaneRecovery.MaxLevels(level)`.
- **`WizardResources`**: immutable `FeatureState` under `WizardResourcesKey` (FR-107): Arcane Recovery used,
  Overchannel use count, Signature Spells free casts spent, `SpellMastery?`, `SignatureSpells?`, the Scholar skill,
  whether a Spell Mastery replacement is available; no pending addition (derived) and no subclass id. Transitions:
  `afterRest(RestKind)` returning only the new state (FR-050, FR-069, FR-074, FR-090) and
  `recordSignatureSpellCast(SpellID)`. Rest handling is a pure transition, not a use case; the rest event is
  `RestApplied`.
- **`WizardCreationChoices`** (FR-105): implements both methods of `ClassCreationChoices` (seeding `FeatureStates`;
  an empty key list) and exposes the validated `skills`, `equipment`, `cantrips` and `spellbook`, which the
  host passes on to `CreateCharacter` (class skills, equipment option letter) and `CreateSpellcasting` (starting
  cantrips, spellbook; settled, section 9). `NewWizardCreation(wizard, skills, equipmentOption, cantrips, spellbook)` returns `(WizardCreation, error)`: the value with a nil error, or an error joining every `WizardCreationRejection` (errors.Is `srderr.ErrInvalidInput`).
- **`SpellMastery`**, **`SignatureSpells`**: `NewSpellMastery(chosen []WizardSpellFacts, book Spellbook)` and `NewSignatureSpells(...)` enforce the level and count rules (FR-065 to FR-075) and return `(SpellMastery, error)` / `(SignatureSpells, error)` (the error carries the `<Type>Rejection`, errors.Is `srderr.ErrInvalidInput`). The use cases make the state checks
  (`FeatureNotAvailable`, `Already` with a nil error, `NotAfterLongRest`, `NotHeld`) first. `castRequirement(...)` reports
  whether a cast needs a Spell Slot (FR-068, FR-073). `WizardAlwaysPrepared.Of(WizardResources)` gives the
  always-prepared ids (FR-070).

**Rejection types.** Each is closed with one variant per cause (data in brackets). In code every variant name is
prefixed with its type stem (FR-108; for example `SpellMasteryNotInSpellbook`); this spec uses short names. The
order below is the reporting order (FR-100). `srd5e.character.rejection` is the short variant name in snake_case,
so the set is this table plus `content_missing`, `subclass_unknown` and `not_a_wizard`.

| Type | Variants in reporting order |
|---|---|
| `SkillsRejection` | `WrongSkillCount(count)`, `DuplicateSkill(skill)`, `SkillNotAllowed(skill)` |
| `EquipmentRejection` | `UnknownEquipmentOption(option)` (raw `string`) |
| `CantripChoiceRejection` | `WrongCantripCount(allowed)`, `DuplicateCantrip(id)`, `NotACantrip(id)`, `NotOnWizardList(id)` |
| `SpellbookRejection` | `FeatureNotAvailable`, `WrongSpellCount(allowed)`, `DuplicateSpell(id)`, `IsACantrip(id)`, `NotOnWizardList(id)`, `AboveHighestSpellLevel(id, highest)`, `AlreadyInSpellbook(id)`, `NotInSourceSpellbook(id)` |
| `ArcaneRecoveryRejection` | `NotAfterShortRest`, `AlreadyUsed`, `SlotLevelTooHigh(level)`, `SlotNotExpended(level)`, `ExceedsMaximum(max)` |
| `ScholarRejection` | `FeatureNotAvailable`, `Already` with a nil error, `SkillNotEligible(skill)`, `NotProficient(skill)` |
| `WizardCreationRejection` | `SkillsInvalid(SkillsRejection)`, `EquipmentInvalid(EquipmentRejection)`, `CantripsInvalid(CantripChoiceRejection)`, `SpellbookInvalid(SpellbookRejection)` |
| `MemorizeRejection` | `FeatureNotAvailable`, `NotAfterShortRest`, `NotPrepared(id)`, `AlwaysPreparedNotReplaceable(id)`, `NewSpellIsCantrip(id)`, `NewSpellNotInSpellbook(id)`, `NewSpellAboveHighestLevel(id, highest)`, `NewSpellAlreadyPrepared(id)` |
| `SpellMasteryRejection` | `FeatureNotAvailable`, `NotAfterLongRest`, `NotHeld`, `Already` with a nil error, `NotAMasterySpell(id)`, `NotInSpellbook(id)`, `DuplicateSpell(id)`, `WrongSpellLevels(levels)`, `CastingTimeNotAction(id)` |
| `SignatureSpellsRejection` | `FeatureNotAvailable`, `Already` with a nil error, `WrongSpellCount(allowed)`, `DuplicateSpell(id)`, `NotLevelThree(id)`, `NotInSpellbook(id)` |
| `RitualAdeptRejection` | `SpellbookUnavailable`, `NotInSpellbook`, `NotARitualSpell` |

**Events** (past tense), one per state transition, each carrying the `CharacterID` (FR-091): `WizardSkills` with a nil error,
`WizardEquipment` with a nil error, `WizardCantrips` with a nil error, `ScholarExpertise` with a nil error, `SpellsAddedToSpellbook(source)`
(`start`, `levelUp`, `copy`, `evocationSavant`, `evokerLevel`, `replacement`; the last two are the Evoker's), `ArcaneRecoveryUsed`,
`SpellMemorized`, `SpellMastery` with a nil error, `SpellMasteryReplaced`, `SignatureSpells` with a nil error. No
Wizard subclass or rest event exists. **No event:** `CopySpellbook`, `CheckRitualAdept`, `recordSignatureSpellCast`
(the cast event is spellcasting.md's `SpellCast`) (Evoker events: class-wizard-evoker.md). Invariants: no cantrip, off-list or repeated spell in a returned spellbook (FR-032); uses never
above the maximum nor below zero (`WizardResources` constructor, `srderr.ErrInvalidInput`). Failure channels are FR-093.

## 6. Interface

Use cases are in `character/`, single-purpose with one method `Run`, taking the current state in the
command and returning the new state and events (`doc/rules/architecture.md` §4). A use case is `context.Context`-taking, error-returning
only when it loads the Wizard `ClassDefinition` through `content.Source`; the others do no I/O. Spells are never loaded: the caller
passes `WizardSpellFacts`.

A use case `X` returns `(X, error)` with `X` with a nil error, an error and, as applicable, `XContentMissing`
(every content-loading use case, FR-093), `XNotAWizard`
(commands carrying `ClassLevels`, FR-095); the table lists the rest (shared-types-and-ownership.md FR-017, FR-018,
FR-021; FR-108). The short names in the FR text (`SpellbookChanged`, `ArcaneRecoveryApplied`, `MemorizeApplied`,
`RitualAdeptEligible`) name the payload of `X` with a nil error. Commands whose use case produces an event
carry the `CharacterID`; those listing `WizardResources` take it and return the new one when it changes (FR-091,
FR-107). A "rest trigger" is the `RestApplied` event (FR-110); `afterRest` takes its `benefitsOf` `RestKind`.

| Use case | Ports used | Command carries | `X` with a nil error payload | an error rejection | FRs |
|---|---|---|---|---|---|
| `ChooseWizardSkills` | takes ctx; class-definition loader | `CharacterID`, selected `Skill`s | `WizardSkills` with a nil error event | `SkillsRejection` | FR-004 to FR-006 |
| `ChooseWizardEquipment` | takes ctx; class-definition loader | `CharacterID`, option as a raw `string` | `WizardEquipment` with a nil error | `EquipmentRejection` | FR-009 to FR-011 |
| `ChooseWizardCantrips` | takes ctx; class-definition loader | `CharacterID`, `ClassLevels`, current cantrips, chosen `WizardSpellFacts` | `WizardCantrips` with a nil error | `CantripChoiceRejection` | FR-027 to FR-029, FR-099 |
| `ChooseScholarExpertise` | takes ctx; class-definition loader | `CharacterID`, `ClassLevels`, `WizardResources`, proficient skills, skill | `ScholarExpertise` with a nil error | `ScholarRejection` | FR-053 to FR-056 |
| `StartSpellbook` | takes ctx; class-definition loader | `CharacterID`, `ClassLevels`, six `WizardSpellFacts` | `SpellbookChanged` | `SpellbookRejection` | FR-030 to FR-032 |
| `AddLevelUpSpells` | takes ctx; class-definition loader | `CharacterID`, `ClassLevels`, `Spellbook`, two `WizardSpellFacts` | `SpellbookChanged` | `SpellbookRejection` | FR-033, FR-034 |
| `CopySpellIntoBook` | takes ctx; class-definition loader | `CharacterID`, `ClassLevels`, `Spellbook`, one `WizardSpellFacts` | `SpellbookChanged(cost)` | `SpellbookRejection` (no cost) | FR-035, FR-036 |
| `CopySpellbook` | no ports (pure, no ctx) | `ClassLevels`, source `Spellbook`, named `WizardSpellFacts` | `SpellbookChanged(cost)` | `SpellbookRejection` | FR-037, FR-038 |
| `ReplaceLostSpellbook` | no ports (pure, no ctx) | `CharacterID`, `ClassLevels`, prepared `WizardSpellFacts` | `SpellbookChanged(cost)` | none (no an error) | FR-039 |
| `UseArcaneRecovery` | takes ctx; class-definition loader | `CharacterID`, `ClassLevels`, `WizardResources`, expended slots, chosen slots, rest trigger | `ArcaneRecoveryApplied(recovered, resources, events)` | `ArcaneRecoveryRejection` | FR-045 to FR-052 |
| `MemorizeSpell` | takes ctx; class-definition loader | `CharacterID`, `ClassLevels`, `WizardResources` (read only, for the always-prepared set), prepared list, `Spellbook`, spell to replace, new `WizardSpellFacts`, rest trigger | `MemorizeApplied(prepared list, events)` | `MemorizeRejection` | FR-061 to FR-064 |
| `CheckRitualAdept` | no ports (pure, no ctx) | `ClassLevels`, `Spellbook`, `WizardSpellFacts`, whether the spellbook is available | `RitualAdeptEligible` | `RitualAdeptRejection` | FR-041 to FR-044, FR-102 |
| `ChooseSpellMastery` | takes ctx; class-definition loader | `CharacterID`, `ClassLevels`, `WizardResources`, `Spellbook`, two `WizardSpellFacts` | `SpellMastery` with a nil error | `SpellMasteryRejection` | FR-065 to FR-068, FR-070 |
| `ReplaceSpellMastery` | takes ctx; class-definition loader | `CharacterID`, `ClassLevels`, `WizardResources`, `Spellbook`, spell to replace, new `WizardSpellFacts`, rest trigger | `SpellMasteryReplaced(resources, events)` | `SpellMasteryRejection` | FR-069 |
| `ChooseSignatureSpells` | takes ctx; class-definition loader | `CharacterID`, `ClassLevels`, `WizardResources`, `Spellbook`, two `WizardSpellFacts` | `SignatureSpells` with a nil error | `SignatureSpellsRejection` | FR-071 to FR-075 |

`CreateCharacter` calls only the two methods of `WizardCreationChoices` (FR-105); no `ChooseWizardSubclass` or
creation use case exists here. No use case here rolls dice; `combat` consumes only the returned value objects. Ports: `content.Source` (content-loading; `ContentNotFound` for absence, `*srderr.PortError` for
infrastructure failure), `telemetry.Telemetry` (`telemetry.Noop` default); no new port, double or adapter.
Package `character` exports the use cases, commands, results (FR-108), rejection and parse
types, `WizardSpellFacts`, `WizardRules`, `WizardResources`, `WizardResourcesKey`, `WizardCreationChoices`,
`SpellMastery`, `SignatureSpells`, `WizardAlwaysPrepared`, the events and published-language value objects;
(`doc/rules/architecture.md` §9).

## 7. Observability

Evaluation questions (`doc/rules/observability.md` §2). A rejection is a normal outcome, not an error. Operators see
span error status and the `srd5e.character.feature_used.errors` counter for port failures, and `srd5e.outcome` with
`srd5e.character.rejection` for returned failures, which carry no `error.type` (errors-and-results.md FR-043,
governing until observability.md §7 is amended per its OQ-11). Diagnosis uses the low-cardinality attributes below
and the result (slots, spells, cost, use number, dice); no spell id, character id or free text is an attribute.
Monitor non-zero `.errors`, `content_missing` and `subclass_unknown`, and a sudden rise in `arcane_recovery` or
spellbook failures (possibly a host bug, such as always passing a Long Rest as the trigger);
Noise: one
`LevelDebug` log per call that returns a result and none when a port error propagates; no `LevelInfo`, no domain logging;
no per-creature attribute.

The domain is **silent by design**; only use cases emit signals (`telemetry.StartTimer`,
`Tracer.Start` and `Span.End`; observability-ports.md FR-013, FR-029, FR-031). **Shared signals used, not
defined here** (shared-types-and-ownership.md FR-042, FR-063; character-aggregate.md FR-118, FR-119): the counter
`srd5e.character.feature_used`, the histogram `srd5e.character.feature_used.duration` (`srd5e.class` and
`srd5e.feature` only), the counter `srd5e.character.feature_used.errors` (`srd5e.class`, `srd5e.feature`,
`error.type`) and the child lookup span `srd5e.character.load_content`, with `srd5e.class` = `wizard` and
`srd5e.feature`.

`srd5e.feature` values (15, one per use case; the Evoker's six are class-wizard-evoker.md's): `skills`, `equipment`,
`cantrips`, `scholar`, `spellbook_start`, `spellbook_level_up`, `spellbook_copy_in`, `spellbook_copy_book`,
`spellbook_replace`, `arcane_recovery`, `memorize_spell`, `ritual_adept`, `spell_mastery`, `spell_mastery_replace`,
`signature_spells`. Attributes on every
span, the counter and the log: `srd5e.class`, `srd5e.feature`, `srd5e.outcome`; on failures also
`srd5e.character.rejection` (the snake_case short variant of section 5, plus `content_missing`,
`subclass_unknown`, `not_a_wizard`); the histogram carries `srd5e.class` and `srd5e.feature` only.

Signals this spec owns (constants in `character/telemetry.go` or character-aggregate.md's
shared file, pinned by a test, FR-098):

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.<snake_case use case>` for the 12 content-loading use cases (`choose_wizard_skills` ... `choose_signature_spells`) and the 3 pure ones (`copy_spellbook`, `replace_lost_spellbook`, `check_ritual_adept`) | span | as above (`error.type` on a port failure) | One span per call; content-loading ones have the child `srd5e.character.load_content` span | Spans with error status; slow traces |
| `srd5e.character.operation_completed` | log event, `LevelDebug` | all but `error.type` | One summary per call that returns a result; none on a propagated error | Enable at `LevelDebug` only when diagnosing |

## 8. Test plan

Levels: unit (`character/*_test.go` (package `character_test`)), application (`character/*_test.go` (use-case tests), recording `telemetry.Telemetry`,
`dicetest.FixedRoller`, `contenttest.FakeSource`), architecture, scenario (`scenarios/`, public
API only). The shared `content.Source` contract suite covers the port; no double or contract suite is added here.

| Id | Level | Notes |
|---|---|---|
| FR-004 to FR-006, FR-009 to FR-011 | unit | Skill counts 0, 1, 2, 3; duplicates; Athletics; option A gives the definition's lines and 5 GP, B gives 55 GP and no lines; `C`, `a` and empty give `UnknownEquipmentOption`; no `ItemID` |
| FR-018 | unit | `highestSpellLevel` and `newSpellLevelAt` over 4.2.2 with the real Wizard `ClassDefinition`; levels 0 and 21 propagate `srderr.ErrInvalidInput` from `rowAt` (one test) |
| FR-019 to FR-021, FR-109 | unit | `ClassLevels` with Wizard first (FR-019 grants) and second (FR-020, FR-021 only) |
| FR-027 to FR-029, FR-099 | unit | Cantrip sizes 2, 3, 4 and each variant; `cantripsGainedAt` at levels 1, 3, 4, 5, 10; `cantripsPending` below, at and above the column; facts with and without `wizard` in `classes`; agreement with `classSpellList` |
| FR-030 to FR-039 | unit | Six-spell book (five, seven, repeats, cantrip, level 2, off-list, empty book representable); 4.5.1 and 4.5.2 one test per row; `spellsOwedAtLevelUp` 0 at level 1 and 2 at 2 to 20; rejected copies carry no cost |
| FR-041 to FR-044, FR-102 | unit | Eligibility matrix of the four conditions; a scenario passes `CastMode.Ritual` to `CastSpell` for an unprepared spellbook Ritual spell and asserts `CastSource.Ritual` (casting-and-concentration.md FR-114), no new `CastingContext` field |
| FR-045 to FR-056, FR-110 | unit, application | 4.7.1 one test per row (AR-15 interrupted Long Rest), `afterRest` fed each `benefitsOf`; variants asserted, use-case results in application; Scholar: Insight, non-proficient skill, below level 2, second choice |
| FR-106 | unit | Evoker `featuresAtOrBelow(wizardLevel)` at Wizard levels 1, 2 (none), 3, 5, 6, 10, 13, 14, 20 gives the entries at 3, 6, 10, 14 only; a feature absent from the list is the `FeatureNotAvailable` case |
| FR-090 | unit | Long Rest resets the Overchannel count to a first use; a Short Rest leaves it |
| FR-061 to FR-075 | unit, application | Memorize replacement cases and each variant (the application test updates the prepared list); Spell Mastery and Signature Spells level and count combinations, `magicAction` versus other casting time, no-slot at lowest level versus slot at higher, Long Rest replacement, `WizardAlwaysPrepared` contents, per-spell once, Short and Long Rest reset, higher-level cast leaves the free cast unused |
| FR-091 | unit | Inputs not mutated; mutating a returned lists leaves the stored value unchanged (accessors return copies); the events of section 5 carry the command's `CharacterID` and are a `CharacterEvent` (a test-only switch outside the closed `CharacterRecordEvent` family compiles) and a `DomainEvent`; two characters' events are distinguishable |
| FR-093 to FR-096 | application | Class not found gives `XContentMissing(MissingContent)`; a returning double raises its typed `*srderr.PortError`; no Wizard levels gives `XNotAWizard` for every use case with `ClassLevels`; same result with the no-op and the recording `telemetry.Telemetry` |
| FR-097, FR-098 | application, unit | Per use case: one span with the section 7 name, one `feature_used` increment and one `feature_used.duration` record; a returned failure sets `srd5e.character.rejection`, no `error.type`, no `.errors` increment; content missing sets span error status; a port failure gives one `feature_used` increment (`failure`, no `error.type`, no rejection), one `feature_used.errors` increment, error span status and no log; every returned result emits one `LevelDebug` log and a propagated error none; every name, key and value string is pinned |
| FR-100, FR-105, FR-107 | unit | Two causes at once report the first in the type's order, for each rejection type; the `New` constructor with one defect per part and two at once; the seeding method returns `FeatureStates` with the initial `WizardResources` under `wizard_resources` and leaves other keys intact; the key list is empty and read-only (accessors return copies); a character with Wizard levels and no stored state reads as the initial state |
| FR-108 | architecture | Package scan: no generic or duplicate exported name; every result, parse and rejection variant carries its prefix |
| Whole slice | scenario | Through the public packages only (`scenarios/`, package `scenarios_test`): a level 4 Wizard builds its spellbook, finishes a Short Rest and uses Arcane Recovery |

Mutation testing: every changed line in `character/` at a score of at least 80
(`doc/rules/testing.md` §8); application lines SHOULD be tested. Withdrawn ids are covered by their new owners.

## 9. Open questions and decisions

Open: none.

Settled, one line each:
- Lost spellbook cost: "the same procedure" follows Copying the Book, so 1 hour and 10 GP per spell level of each prepared spell (FR-039); the fast rate is justified by the SRD's "you already know how to cast the spell" (SRD 5.2.1, "Expanding and Replacing a Spellbook", p. 78; interpretation, the nearest preceding procedure).
- `CastSpell` rejecting a Ritual cast with an unreadable spellbook (former CS-1): no. Whether the book can be read is a host circumstance, not state; `CheckRitualAdept` stays advisory and `CastSpell` needs no new input (FR-102, FR-044). No amendment.
- Applying `afterRest` on `RestApplied`: the host passes `RestApplied.BenefitsOf`; windows take the whole event (FR-110; rests-and-hit-dice.md FR-060, FR-061, Q-1). No generic refresh hook.
- Starting items and `ItemID`: lines are used as held (`{name, quantity}`), never typed here; typing is content-equipment's concern (FR-009).
- Creation-time cantrips and spellbook: the host passes the validated values to `CreateSpellcasting` (spellcasting.md FR-092 takes starting cantrips and a Wizard's spellbook); skills and equipment go in the `CreateCharacter` fields; `ClassCreationChoices` seeds only `FeatureStates` (FR-105).

Settled readings: pending cantrip choices and level-up spells are derived, not stored, because copied and found spells
change the book size (FR-029, FR-033); there is no spellbook size limit, as the SRD gives no page cost per spell
(FR-032). An empty Arcane Recovery choice is a no-op and does not use the feature, because the SRD says "you can
choose" (FR-052). Signature Spells cannot be changed, because the SRD gives no replacement rule, unlike Spell Mastery
(FR-075). The Evoker readings are in class-wizard-evoker.md section 9. Settled elsewhere: Evoker data ownership,
`WizardResources` in `FeatureStates`, shared telemetry names (shared-types-and-ownership.md FR-025, FR-022, FR-042,
FR-063); returned failures are not errors (errors-and-results.md FR-043); the Wizard list is 218 spells
(content-spells.md FR-022, FR-097).

## 10. Changelog impact

`Added`: the Wizard behaviour, resource state, use cases and the signals in section 7. Not **BREAKING**. Class and
spell data come from content-classes.md and content-spells.md (shared-types-and-ownership.md FR-033, FR-034). The
first code change records the decisions it implements.
