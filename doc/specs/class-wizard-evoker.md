# Class: Wizard, Evoker subclass

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `class-wizard.md`, `character-aggregate.md`, `derived-stats.md`, `d20-tests.md`, `spellcasting.md`, `content-classes.md`, `content-spells.md`, `content-source-port.md`, `dice-and-rolls.md`, `ability-scores.md`, `observability-ports.md`, `errors-and-results.md` |

Binding decisions: [shared-types-and-ownership.md](shared-types-and-ownership.md);
where this spec conflicts with it, that spec wins. Split from `class-wizard.md` to keep
both under the size limit (shared-types-and-ownership.md FR-062); it ranks after that spec
(section 4.11 there, rank 17) because it consumes its types. No `combat` spec is a
dependency (shared-types-and-ownership.md FR-013).

> Go notation used below. A name in lowerCamelCase (a field, command argument or enum member) stands for the exported PascalCase Go identifier (`attackerID` is the field `AttackerID`). A trailing `?` marks an optional value (the zero value or a `(T, bool)` pair). `[]T` is a slice; `set of T` is a collection without duplicates. A `(T, error)` pair replaces an accepted/rejected result pair: the nil-error case is the accepted case and the error is the typed rejection, satisfying `errors.Is(err, srderr.ErrInvalidInput)`; a parser reports every problem at once through `errors.Join`. Rule outcomes (refusals, not-possible stops) stay result values. A use case is a struct with `Run(ctx, cmd)`; its `error` is non-nil only for a port failure.

## 1. Purpose and scope

Specifies the **behaviour** of the Wizard subclass Evoker as SRD 5.2.1 presents it
(p. 82), inside the `character` context: Evocation Savant, Potent Cantrip, Sculpt Spells,
Empowered Evocation and Overchannel. Content is upstream: the Evoker `SubclassDefinition`
and its five features are `content-classes.md`'s (shared-types-and-ownership.md FR-025).
The Wizard state `WizardResources`, `WizardSpellFacts`, `WizardRules`, `SpellbookRules`
types, the Wizard list test and the held-features rule are `class-wizard.md`'s
(FR-106 there). Pattern (shared-types-and-ownership.md FR-023): each feature is a pure
domain function from (state, inputs) to (new state, events), with one use case. Features
resolved in `combat` are returned only as value objects; Potent Cantrip takes
`d20.Outcome` and `d20.Kind` from `d20`.

**Out of scope:** the subclass choice at level 3 (level-advancement-and-multiclassing.md,
shared-types-and-ownership.md FR-020); applying damage, Resistance, Immunity and maximum
damage (`combat`); other Wizard features (class-wizard.md); other subclasses (the SRD has only
Evoker). Nothing outside SRD 5.2.1 was used.

## 2. Source

PDF page numbers equal printed page numbers (checked: PDF page 82).

- SRD 5.2.1, "Wizard Subclass: Evoker", "Level 3: Evocation Savant", "Level 3: Potent
  Cantrip", "Level 6: Sculpt Spells", "Level 10: Empowered Evocation", "Level 14:
  Overchannel", p. 82; "Wizard Features", p. 77; "Round Down", p. 5; "Half Damage", p. 16;
  "Using a Higher-Level Spell Slot", p. 104

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Evoker | `SubclassID` `evoker` | Lookup: `ClassDefinition.Subclass` (`SubclassFound` / `SubclassNotFound`, content-classes.md FR-074). |
| Evocation Savant, Potent Cantrip, Sculpt Spells, Empowered Evocation, Overchannel | Features | Levels from the Evoker `SubclassDefinition` only (class-wizard.md FR-106); Overchannel keeps a use count in `WizardResources`. |

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. Feature levels come only from the
Evoker `SubclassDefinition`; no Evoker level is hard-coded. The host passes the subclass in the
command (the aggregate has no subclass field): the `SubclassDefinition?` itself to the per-cast
use cases and `UseOverchannel`, a `SubclassID?` to `AddEvocationSavantSpells` and
`AddEvokerFreeSpell`, which resolve it with `ClassDefinition.Subclass` (an unlisted id gives
`<UseCase>SubclassUnknown`, FR-018). Any subclass whose id is not `evoker`, or none, is
`NotEvoker`, reported before `FeatureNotAvailable` (class-wizard.md FR-100). Spell and creature
facts are caller-supplied as `WizardSpellFacts` (class-wizard.md section 5). Rejection
variants: section 5.

### 4.1 Evocation Savant and the free spellbook additions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `AddEvocationSavantSpells` MUST accept exactly two Wizard spells of the Evocation school, each of level 1 or 2 (a cantrip is not a spellbook spell: section 9), not already in the spellbook, from an Evoker who holds the feature (class-wizard.md FR-106), and return the spellbook with both added free of cost. | SRD 5.2.1, "Level 3: Evocation Savant", p. 82 | happy |
| FR-002 | `AddEvocationSavantSpells` MUST reject, by cause: a character without the subclass with `NotEvoker`, a level below 3 with `FeatureNotAvailable`, a count other than two with `WrongSpellCount(allowed)`, a repeated spell with `DuplicateSpell(id)`, a cantrip with `IsACantrip(id)`, a spell not on the Wizard list with `NotOnWizardList(id)`, a spell not of the Evocation school with `NotAnEvocationSpell(id)`, a spell above level 2 with `AboveHighestSpellLevel(id, 2)`, and a spell already in the spellbook with `AlreadyInSpellbook(id)`. | SRD 5.2.1, "Level 3: Evocation Savant", p. 82 | unhappy (invalid input) |
| FR-003 | At the Wizard levels where `WizardRules.NewSpellLevelAt` (class-wizard.md FR-018) is true after level 3 (5, 7, 9, 11, 13, 15 and 17), and not at level 3, `AddEvokerFreeSpell` MUST accept one Evocation Wizard spell of a level for which the Evoker has Spell Slots, not already in the spellbook, and return the spellbook with it added free of cost, in addition to the additions of class-wizard.md FR-033 (cases in 4.1.1; level 3 reading in section 9). | SRD 5.2.1, "Level 3: Evocation Savant", p. 82 | happy |
| FR-004 | `AddEvokerFreeSpell` MUST reject, by cause: a character without the subclass with `NotEvoker`, a Wizard level below 3 with `FeatureNotAvailable`, a level that is not one of those of FR-003 with `NoNewSpellLevel`, a cantrip with `IsACantrip(id)`, a spell not on the Wizard list with `NotOnWizardList(id)`, a spell not of the Evocation school with `NotAnEvocationSpell(id)`, a spell above the highest Spell Level with `AboveHighestSpellLevel(id, highest)`, and a spell already in the spellbook with `AlreadyInSpellbook(id)`. | SRD 5.2.1, "Level 3: Evocation Savant", p. 82 | unhappy (state error) |

#### 4.1.1 Evoker free addition cases (one table-driven test per row)

Source for every row: SRD 5.2.1, "Level 3: Evocation Savant", p. 82, and "Wizard
Features", p. 77. "Highest" is the highest Spell Level at the Wizard level (class-wizard.md 4.2.2).

| Label | Wizard level | Highest | Spell | Expected | Path |
|---|---|---|---|---|---|
| FA-01 | 3 | 2 | Evocation, level 2 | `NoNewSpellLevel` (level 3 reading: section 9) | unhappy (boundary) |
| FA-02 | 5 | 3 | Evocation, level 3 | accepted | happy (boundary: first level with a free addition) |
| FA-03 | 5 | 3 | Evocation, level 4 | `AboveHighestSpellLevel(id, 3)` | unhappy (boundary) |
| FA-04 | 6 | 3 | Evocation, level 3 | `NoNewSpellLevel` (level 6 does not raise the highest level) | unhappy (boundary) |
| FA-05 | 17 | 9 | Evocation, level 9 | accepted | happy (boundary: highest level) |
| FA-06 | 5 | 3 | not Evocation, level 3 | `NotAnEvocationSpell(id)` | unhappy (rule interaction) |
| FA-07 | 7 | 4 | Evocation, level 4, already in the spellbook | `AlreadyInSpellbook(id)` | unhappy (invalid input) |
| FA-08 | 5 | 3 | Evocation cantrip | `IsACantrip(id)` | unhappy (invalid input) |

### 4.2 Potent Cantrip (level 3)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-005 | `ResolvePotentCantrip` MUST return `ResolvePotentCantrip` with a nil error carrying `PotentCantripHalfDamage(damage)` with the cantrip's damage halved and rounded down, and no additional effect, when the Evoker casts a damaging cantrip at a creature and either misses with the attack roll or the target succeeds on a saving throw (cases in 4.2.1). The caller supplies the `d20.Kind` (`attackRoll` or `savingThrow`) and the `d20.Outcome` of `d20`: a miss is a `failure` of an attack roll, a successful save a `success` of a saving throw (shared-types-and-ownership.md FR-002, FR-004). When the attack hits or the save fails, the ordinary full effect applies: the use case MUST return `ResolvePotentCantrip` with a nil error carrying `PotentCantripUnchanged`, an accepted outcome, not a rejection. | SRD 5.2.1, "Level 3: Potent Cantrip", p. 82; "Round Down", p. 5; "Half Damage", p. 16 | happy |
| FR-006 | `ResolvePotentCantrip` MUST return an error with a `PotentCantripRejection`: `NotEvoker` for a character without the subclass, `FeatureNotAvailable` below level 3, `UnsupportedTestKind(kind)` for an ability check (the feature names only attack rolls and saving throws), `NotACantrip` for a spell that is not a cantrip and `SpellDealsNoDamage` for a cantrip that deals none. | SRD 5.2.1, "Level 3: Potent Cantrip", p. 82 | unhappy (rule interaction) |

#### 4.2.1 Potent Cantrip cases (one table-driven test per row)

Source for every row: SRD 5.2.1, "Level 3: Potent Cantrip", p. 82. "Damage" is
the cantrip's damage if it were to hit, supplied by the caller. Half is rounded
down (SRD 5.2.1, "Round Down", p. 5; "Half Damage", p. 16). A rejected row asserts
the variant of FR-006; an unchanged row asserts `PotentCantripUnchanged` (FR-005).

| Label | Wizard level | Cantrip | Outcome | Damage | Expected | Path |
|---|---|---|---|---|---|---|
| PCa-01 | 3 | attack roll | the attack roll misses | 8 | half damage (4), no additional effect | happy |
| PCa-02 | 3 | saving throw | the target succeeds | 6 | half damage (3), no additional effect | happy |
| PCa-03 | 3 | attack roll | the attack roll hits | 8 | accepted, `PotentCantripUnchanged` (full damage 8 applies) | happy (rule interaction) |
| PCa-04 | 3 | saving throw | the target fails | 6 | accepted, `PotentCantripUnchanged` (full damage 6 applies) | happy (rule interaction) |
| PCa-05 | 3 | no damage | the target succeeds | none | rejected `SpellDealsNoDamage` | unhappy (empty and minimal) |
| PCa-06 | 2 | attack roll | the attack roll misses | 8 | rejected `FeatureNotAvailable` | unhappy (boundary) |
| PCa-07 | 3 | a level 1 spell | the attack roll misses | 8 | rejected `NotACantrip` | unhappy (rule interaction) |
| PCa-08 | 3, no subclass | attack roll | the attack roll misses | 8 | rejected `NotEvoker` | unhappy (rule interaction) |
| PCa-09 | 3 | attack roll | the attack roll misses | 7 | half damage (3, rounded down), no additional effect | happy (boundary: odd damage) |
| PCa-10 | 3 | saving throw | the target succeeds | 7 | half damage (3, rounded down), no additional effect | happy (boundary: odd damage) |
| PCa-11 | 3 | attack roll | the attack roll misses | 1 | half damage (0, rounded down), no additional effect | happy (boundary: minimum damage) |
| PCa-12 | 3 | ability check | the check fails | 8 | rejected `UnsupportedTestKind` | unhappy (invalid input) |

### 4.3 Sculpt Spells (level 6)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-007 | `ResolveSculptSpells` MUST accept, when the Evoker casts an Evocation spell that affects other creatures it can see, a choice of at most 1 plus the level the spell is cast at (the slot level for a higher-level slot; 0 for a cantrip) and return `SculptedCreatures` marking each chosen creature as automatically succeeding on the saving throw and as taking no damage if it would normally take half (cases in 4.3.1). | SRD 5.2.1, "Level 6: Sculpt Spells", p. 82; "Using a Higher-Level Spell Slot", p. 104 | happy |
| FR-008 | `ResolveSculptSpells` MUST reject, by cause: a character without the subclass with `NotEvoker`, a Wizard level below 6 with `FeatureNotAvailable`, a spell not of the Evocation school with `NotAnEvocationSpell`, a chosen creature that is the caster with `Caster` with a nil error, a creature the command marks not visible or not affected with `CreatureNotEligible(id)` (FR-015), and more creatures than allowed with `TooManyCreatures(allowed)`. | SRD 5.2.1, "Level 6: Sculpt Spells", p. 82 | unhappy (boundary) |
| FR-015 | The creature facts of Sculpt Spells (visible to the caster, affected by the spell, is the caster) MUST be supplied by the caller in the command; `ResolveSculptSpells` MUST reject a chosen creature the command marks not visible or not affected with `CreatureNotEligible(id)`. | SRD 5.2.1, "Level 6: Sculpt Spells" ("other creatures that you can see"), p. 82 | unhappy (invalid input) |

#### 4.3.1 Sculpt Spells cases (one table-driven test per row)

Source for every row: SRD 5.2.1, "Level 6: Sculpt Spells", p. 82. "Level" is the
level the spell is cast at (0 for a cantrip; reading in section 9). A rejected row
asserts the variant of FR-008 or FR-015.

| Label | Wizard level | Spell | Level | Allowed number | Chosen | Expected | Path |
|---|---|---|---|---|---|---|---|
| SC-01 | 6 | Evocation | 1 | 2 | 2 | accepted | happy |
| SC-02 | 6 | Evocation | 1 | 2 | 3 | rejected (above allowed) | unhappy (boundary) |
| SC-03 | 11 | Evocation | 5 | 6 | 6 | accepted | happy |
| SC-04 | 17 | Evocation | 9 | 10 | 10 | accepted | happy (boundary: highest level) |
| SC-05 | 6 | Evocation | 0 (cantrip) | 1 | 1 | accepted | happy (boundary: lowest level) |
| SC-06 | 6 | Evocation | 3 | 4 | 0 | accepted, nothing protected | unhappy (empty and minimal) |
| SC-07 | 6 | not Evocation | 3 | n/a | 1 | rejected | unhappy (rule interaction) |
| SC-08 | 5 | Evocation | 3 | n/a | 1 | rejected (feature not gained) | unhappy (boundary) |
| SC-09 | 6 | Evocation | 3 | 4 | 1, the caster | rejected (not another creature) | unhappy (invalid input) |
| SC-10 | 6 | Evocation | 3 | 4 | 1, not visible to the caster | `CreatureNotEligible` (FR-015) | unhappy (invalid input) |
| SC-11 | 6 | Evocation | 1, cast with a level 3 slot | 4 | 4 | accepted (the spell takes the slot level, p. 104) | happy (boundary: higher-level slot) |

### 4.4 Empowered Evocation (level 10)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-009 | `ResolveEmpoweredEvocation` MUST, when the Evoker chooses to apply it (`applies` true) to a Wizard spell of the Evocation school, return `EmpoweredRoll` carrying the Intelligence modifier as a modifier on one damage roll of that spell, and an `EmpoweredCastState` recording that the cast's damage roll is empowered (cases in 4.4.1). | SRD 5.2.1, "Level 10: Empowered Evocation", p. 82 | happy |
| FR-010 | `ResolveEmpoweredEvocation` MUST reject, by cause: a character without the subclass with `NotEvoker`, a Wizard level below 10 with `FeatureNotAvailable`, a spell that is not a Wizard spell with `NotAWizardSpell`, a spell not of the Evocation school with `NotAnEvocationSpell`, and a cast whose `EmpoweredCastState` already records an empowered damage roll with `AlreadyEmpowered`. | SRD 5.2.1, "Level 10: Empowered Evocation", p. 82 | unhappy (rule interaction) |
| FR-016 | Empowered Evocation is optional: `ResolveEmpoweredEvocation` with `applies` false MUST return `EmpoweredRoll` with no modifier and leave the `EmpoweredCastState` unchanged. | SRD 5.2.1, "Level 10: Empowered Evocation" ("you can add"), p. 82 | happy (boundary) |

#### 4.4.1 Empowered Evocation cases (one table-driven test per row)

Source for every row: SRD 5.2.1, "Level 10: Empowered Evocation", p. 82.

| Label | Wizard level | Spell | Intelligence modifier | Damage roll | Expected | Path |
|---|---|---|---|---|---|---|
| EE-01 | 10 | Wizard Evocation | +3 | first | +3 on that roll | happy |
| EE-02 | 20 | Wizard Evocation | +5 | first | +5 on that roll | happy |
| EE-03 | 10 | Wizard Evocation | 0 | first | +0 (no change) | happy (boundary: zero) |
| EE-04 | 10 | Wizard Evocation | -1 | first | -1 on that roll (reading in section 9) | happy (boundary: negative) |
| EE-05 | 10 | Wizard Evocation | +3 | second damage roll | `AlreadyEmpowered` | unhappy (rule interaction) |
| EE-06 | 10 | Wizard, not Evocation | +3 | first | `NotAnEvocationSpell` | unhappy (rule interaction) |
| EE-07 | 9 | Wizard Evocation | +3 | first | `FeatureNotAvailable` | unhappy (boundary) |
| EE-08 | 10 | Wizard Evocation | +3 | first, the caster chooses not to apply it | no modifier, state unchanged (FR-016) | happy (boundary: optional) |

### 4.5 Overchannel (level 14)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-011 | `UseOverchannel` MUST reject, by cause: a character without the subclass with `NotEvoker`, a Wizard level below 14 with `FeatureNotAvailable`, a spell that is not a Wizard spell with `NotAWizardSpell`, a cast not made with a Spell Slot (a cantrip, a Ritual or a free cast) with `NoSpellSlotUsed`, a slot level outside 1 to 5 with `SlotLevelOutOfRange(level)`, and a spell that does not deal damage with `SpellDealsNoDamage`. A rejected request MUST NOT count as a use. | SRD 5.2.1, "Level 14: Overchannel", p. 82 | unhappy (boundary) |
| FR-012 | The first use of Overchannel before a Long Rest MUST return `OverchannelApplied` exposing maximum damage for the spell on the turn it is cast and no Necrotic damage. | SRD 5.2.1, "Level 14: Overchannel", p. 82 | happy |
| FR-013 | Each further use before a Long Rest MUST return `OverchannelApplied` exposing maximum damage and, immediately after the cast, Necrotic damage of n d12 for each level of the slot, where n is the use number (second use 2d12, third 3d12), that ignores Resistance and Immunity (cases in 4.5.1). The damage is a dice expression rolled through the `dice.Roller`. | SRD 5.2.1, "Level 14: Overchannel", p. 82 | happy (boundary) |
| FR-014 | `UseOverchannel` MUST read the use number from `WizardResources` (reset by a Long Rest and not by a Short Rest, class-wizard.md FR-090) and return the resources with the count raised by one on an accepted request. | SRD 5.2.1, "Level 14: Overchannel", p. 82 | unhappy (state error) |
| FR-017 | With a fixed `dice.Roller`, the Overchannel Necrotic damage MUST be identical every run, including the minimum roll (every d12 shows 1) and the maximum roll (every d12 shows 12). | SRD 5.2.1, "Level 14: Overchannel", p. 82; doc/rules/testing.md §4 | unhappy (determinism) |

#### 4.5.1 Overchannel cases (one table-driven test per row)

Source for every row: SRD 5.2.1, "Level 14: Overchannel", p. 82. "Use" is the
number of this use since the last Long Rest.

| Label | Wizard level | Slot level | Spell deals damage | Use | Expected | Path |
|---|---|---|---|---|---|---|
| OC-01 | 14 | 3 | yes | 1 | maximum damage, no Necrotic damage | happy |
| OC-02 | 14 | 3 | yes | 2 | maximum damage, 6d12 Necrotic damage | happy (boundary: first use after the free one) |
| OC-03 | 14 | 1 | yes | 3 | maximum damage, 3d12 Necrotic damage | happy (boundary: lowest slot level) |
| OC-04 | 14 | 5 | yes | 2 | maximum damage, 10d12 Necrotic damage | happy (boundary: highest slot level) |
| OC-05 | 20 | 2 | yes | 4 | maximum damage, 8d12 Necrotic damage | happy (boundary) |
| OC-06 | 14 | 3 | yes | 1 after a Long Rest | no Necrotic damage again | happy (state) |
| OC-07 | 14 | 3 | yes | 2, after a Short Rest only | 6d12 Necrotic damage (count not reset) | happy (state) |
| OC-08 | 14 | 6 | yes | 1 | rejected (slot level 6) | unhappy (boundary) |
| OC-09 | 14 | 3 | no | 1 | rejected (no damage) | unhappy (rule interaction) |
| OC-10 | 14 | none (cantrip) | yes | 1 | rejected (no Spell Slot) | unhappy (rule interaction) |
| OC-11 | 13 | 3 | yes | 1 | rejected (feature not gained) | unhappy (boundary) |
| OC-12 | 14 | 3 | yes, but not a Wizard spell | 1 | rejected | unhappy (rule interaction) |

### 4.6 Cross-cutting behaviour

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | Every use case here follows class-wizard.md FR-091 (new state and events, no mutation, read-only (accessors return copies) collections, events implementing `CharacterEvent` with the `CharacterID`), FR-094 and FR-096 to FR-098 (port failure propagated unchanged, observability-independent results, one span, counter and duration record per call, pinned constants) and FR-100 (first variant in the order of section 5). `AddEvocationSavantSpells` and `AddEvokerFreeSpell` resolve their `SubclassID?` with `ClassDefinition.Subclass`; on `SubclassNotFound(id)` they MUST return their own `<UseCase>SubclassUnknown` carrying that `SubclassID`, and when the `wizard` class definition is absent their own `<UseCase>ContentMissing` (class-wizard.md FR-093); neither is a rejection or an error. A use case whose command carries `ClassLevels` applied to a character with no Wizard levels returns `<UseCase>NotAWizard` (class-wizard.md FR-095). A spell id with no facts in the command is a programmer error (`srderr.ErrInvalidInput`). | class-wizard.md FR-091, FR-093; doc/rules/architecture.md §4, §6 | unhappy (missing content) |

## 5. Model

Domain (`character/`, pure, synchronous). Types owned by other specs are consumed, not redefined.

- **Spellbook additions**: stateless functions over spellcasting.md's `Spellbook` (`addEvocationSavant`, which takes no
  level because its cap is fixed at 2, and `addEvokerFree`, which takes the plain `int highestSpellLevel` and a
  `bool newSpellLevel`) with the `CharacterID` and the `WizardSpellFacts` they touch. They return class-wizard.md's
  `SpellbookChanged` payload or the rejection `EvokerSpellbookRejection`, and the event
  `SpellsAddedToSpellbook(source)` with source `evocationSavant` or `evokerLevel` (defined by class-wizard.md).
- **Feature functions** (`PotentCantrip`, `SculptSpells`, `EmpoweredEvocation`, `Overchannel`; stateless, called by the
  use cases), each taking the class-wizard.md FR-106 held-feature test as a `bool`. Inputs: `EmpoweredCastState(bool
  damageRollEmpowered)` passed in and returned (FR-010), `SculptCandidate(id, isVisible, isAffected, isCaster)`
  (FR-015), `OverchannelRequest(spell, castSource, slotLevel, resources)`. `PotentCantripOutcome` is closed:
  `PotentCantripHalfDamage(damage)` | `PotentCantripUnchanged`.
- **`WizardResources`** (class-wizard.md) holds the Overchannel use count; `UseOverchannel` returns the new state.
- **Event**: `OverchannelUsed` (past tense, carries the `CharacterID`). The three resolve use cases emit no event (their
  commands carry no `CharacterID`).

**Rejection types** (closed, one variant per cause; in code each variant is prefixed with its type stem, class-wizard.md
FR-108; this spec uses short names; the order is the reporting order):

| Type | Variants in reporting order |
|---|---|
| `EvokerSpellbookRejection` | `NotEvoker`, `FeatureNotAvailable`, `NoNewSpellLevel`, `WrongSpellCount(allowed)`, `DuplicateSpell(id)`, `IsACantrip(id)`, `NotOnWizardList(id)`, `NotAnEvocationSpell(id)`, `AboveHighestSpellLevel(id, highest)`, `AlreadyInSpellbook(id)` |
| `PotentCantripRejection` | `NotEvoker`, `FeatureNotAvailable`, `UnsupportedTestKind(kind)`, `NotACantrip`, `SpellDealsNoDamage` |
| `SculptRejection` | `NotEvoker`, `FeatureNotAvailable`, `NotAnEvocationSpell`, `Caster` with a nil error, `CreatureNotEligible(id)`, `TooManyCreatures(allowed)` |
| `EmpoweredRejection` | `NotEvoker`, `FeatureNotAvailable`, `NotAWizardSpell`, `NotAnEvocationSpell`, `AlreadyEmpowered` |
| `OverchannelRejection` | `NotEvoker`, `FeatureNotAvailable`, `NotAWizardSpell`, `NoSpellSlotUsed`, `SlotLevelOutOfRange(level)`, `SpellDealsNoDamage` |

`srd5e.character.rejection` is the short variant name in snake_case; with the values `content_missing`,
`subclass_unknown` and `not_a_wizard` (class-wizard.md section 7) that is the whole set.

## 6. Interface

Use cases are in `character/`, single-purpose with one method `Run`
(`doc/rules/architecture.md` §4). The two spellbook use cases are `context.Context`-taking, error-returning because they load the Wizard
`ClassDefinition` through the context-taking `content.Source` to resolve the subclass; the per-cast use cases and
`UseOverchannel` take the `SubclassDefinition?` in the command and do no I/O. A use case `X` returns the closed
`XResult` with `X` with a nil error and an error and, for the two spellbook use cases, `XContentMissing(MissingContent)`,
`XSubclassUnknown(SubclassID)` and `XNotAWizard` (class-wizard.md FR-108). The short names in the FR text
(`SpellbookChanged`, `PotentCantripHalfDamage`, `PotentCantripUnchanged`, `SculptedCreatures`, `EmpoweredRoll`,
`OverchannelApplied`) name the payload of `X` with a nil error.

| Use case | Mode | Command carries | `X` with a nil error payload | an error rejection | FRs |
|---|---|---|---|---|---|
| `AddEvocationSavantSpells` | content-loading (class definition, subclass lookup) | `CharacterID`, `ClassLevels`, `SubclassID?`, `Spellbook`, two `WizardSpellFacts` | `SpellbookChanged` | `EvokerSpellbookRejection` | FR-001, FR-002 |
| `AddEvokerFreeSpell` | content-loading (class definition, subclass lookup) | `CharacterID`, `ClassLevels`, `SubclassID?`, `Spellbook`, one `WizardSpellFacts` | `SpellbookChanged` | `EvokerSpellbookRejection` | FR-003, FR-004 |
| `ResolvePotentCantrip` | sync | `ClassLevels`, `SubclassDefinition?`, `WizardSpellFacts`, `d20.Kind`, `d20.Outcome`, damage | `PotentCantripHalfDamage(damage)` or `PotentCantripUnchanged` | `PotentCantripRejection` | FR-005, FR-006 |
| `ResolveSculptSpells` | sync | `ClassLevels`, `SubclassDefinition?`, `WizardSpellFacts`, cast level, `SculptCandidate`s | `SculptedCreatures` | `SculptRejection` | FR-007, FR-008, FR-015 |
| `ResolveEmpoweredEvocation` | sync | `ClassLevels`, `SubclassDefinition?`, `WizardSpellFacts`, Intelligence `AbilityModifier`, `applies`, `EmpoweredCastState` | `EmpoweredRoll(modifier?, state)` | `EmpoweredRejection` | FR-009, FR-010, FR-016 |
| `UseOverchannel` | sync (rolls through `dice.Roller`) | `CharacterID`, `ClassLevels`, `SubclassDefinition?`, `WizardResources`, `OverchannelRequest` | `OverchannelApplied(use number, maximum damage, necrotic?, resources, events)` | `OverchannelRejection` | FR-011 to FR-014, FR-017 |

`UseOverchannel` rolls only the Necrotic expression, through the `dice.Roller`; applying damage and maximum damage are
`combat`'s job, which consumes only the returned value objects. Ports: `content.Source`, `dice.Roller`, `telemetry.Telemetry`
(`telemetry.Noop` default); no new port, double or adapter. The use cases, commands, results, rejection types and
value objects are exported from package `character`.

## 7. Observability

A rejection is a normal outcome, not an error; operators and monitoring are those of class-wizard.md section 7. The
per-cast use cases here are the highest-volume Wizard use cases and carry no per-creature attribute; the domain is
**silent by design**. Signals are the shared class-feature signals owned by character-aggregate.md (FR-118, FR-119;
shared-types-and-ownership.md FR-042, FR-063), emitted as class-wizard.md FR-097 states, with `srd5e.class` =
`wizard` and `srd5e.feature` one of `evocation_savant`, `evoker_free_spell`, `potent_cantrip`, `sculpt_spells`,
`empowered_evocation`, `overchannel`. `PotentCantripUnchanged` is a success. This spec defines no signal beyond:

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.<snake_case use case>` for the six use cases (`add_evocation_savant_spells` ... `use_overchannel`) | span | as in class-wizard.md section 7 | One span per call; the two content-loading ones have the child `srd5e.character.load_content` span; `use_overchannel` includes the roll | Spans with error status; slow traces |

## 8. Test plan

Levels as in class-wizard.md section 8 (recording `telemetry.Telemetry`, `dicetest.FixedRoller`, `contenttest.FakeSource`).

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-004 | unit, application | Evoker features held by level via an in-memory `SubclassDefinition` (class-wizard.md FR-106): none without a subclass, with a non-Evoker definition, or below level 3: `NotEvoker` before `FeatureNotAvailable`; school and level checks; 4.1.1 (levels 3 and 5 pinned); an unlisted id gives `AddEvocationSavantSpellsSubclassUnknown` / `AddEvokerFreeSpellSubclassUnknown` |
| FR-005 to FR-017 | unit, application | 4.2.1, 4.3.1, 4.4.1 and 4.5.1 one test per row (odd damage, `PotentCantripUnchanged` rows, ability-check rejection, visibility and caster facts, `applies` false, state threading); the Overchannel application test rolls with a fixed roller of all 1s and all 12s and resets the count after a Long Rest only |
| FR-018 | application | Class not found gives `XContentMissing`; no Wizard levels gives `XNotAWizard`; a returning double raises its typed `*srderr.PortError`; one span, counter and duration record per call; same result with the no-op and the recording `telemetry.Telemetry` |
| Whole slice | scenario | Through the public packages only (`scenarios/`, package `scenarios_test`): a level 14 Evoker uses Overchannel twice |

Mutation testing: every changed line in `character/` at a score of at least 80
(`doc/rules/testing.md` §8); application lines SHOULD be tested.

## 9. Open questions and decisions

None open. Settled readings: the Evoker free spell at level 3 is not granted beyond the two spells of the first
paragraph, and the "new level of Spell Slots" addition applies at levels 5, 7, 9, 11, 13, 15, 17 (literally level 3 is
also a new level, so this is a reading, not text; FR-003, FA-01 to FA-08). Sculpt Spells' "the spell's level" is the
level the spell is cast at (SRD "Using a Higher-Level Spell Slot", p. 104), a cantrip being 0 (FR-007, SC-05, SC-11).
Potent Cantrip rounds half down (FR-005). Empowered Evocation is optional and adds a negative modifier as is (FR-016,
EE-04). Evocation Savant excludes cantrips because a spellbook holds level 1+ spells only (FR-001, FR-002).
Overchannel's slot level is the slot expended (FR-011).

## 10. Changelog impact

`Added`: the Evoker features (Evocation Savant, Potent Cantrip, Sculpt Spells, Empowered Evocation, Overchannel), their
use cases and results. Not **BREAKING**. The Evoker data comes from content-classes.md (shared-types-and-ownership.md
FR-025). The first code change records the decisions it implements.
