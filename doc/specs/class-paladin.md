# Class: Paladin

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `derived-stats.md`, `character-aggregate.md`, `spellcasting.md`, `rests-and-hit-dice.md`, `feats-behaviour.md`, `weapon-properties-and-mastery.md`, `level-advancement-and-multiclassing.md`, `character-creation.md`, `content-classes.md`, `content-spells.md`, `content-equipment.md`, `content-source-port.md`, `content-schema-and-decoder.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `ability-scores.md`, `d20-tests.md`, `errors-and-results.md`, `observability-ports.md` |

## 1. Purpose and scope

Specifies the behaviour of the Paladin class of SRD 5.2.1. The bundled Paladin
data (class entry, feature table and texts, spell list, Oath of Devotion) is
owned by `content-classes.md` (shared-types-and-ownership.md FR-025); see the
withdrawn ids (section 4.12).

**Character behaviour.** Pure rules for the Paladin-specific choices, Lay On
Hands and Restoring Touch, Channel Divinity and its options, the free casts, the
auras, Extra Attack and Radiant Strikes data, Holy Nimbus, feat entitlements,
rest recharge and level-up resource changes. Each behaviour is a pure domain
function from (state, inputs) to (new state, events)
(shared-types-and-ownership.md FR-023); effects are typed data (radius, bonus,
DC, duration in minutes) for the host and `combat` to apply.

**Out of scope:**

- Generic class grants and choices (`character-creation.md`,
  `level-advancement-and-multiclassing.md`: grants, prerequisite, level-up
  transaction, subclass choice, advance rejections); `FightingStyleChoice`
  (`feats-behaviour.md`); `WeaponMasteryChoice`
  (`weapon-properties-and-mastery.md`); `RestKind`, `RestApplied` and rest effects
  (`rests-and-hit-dice.md`); Fixed Hit Points, p. 23 (`character-aggregate.md`).
- The spells themselves (spell ids only), casting mechanics, slots, prepared
  limits, save DC, multiclass slots (`spellcasting.md`); how Extra Attack combines
  across classes; the Cleric spell list (Blessed Warrior reads `Spell.classes`,
  content-spells.md FR-022); any other Paladin subclass (the SRD has one).
- Armor Class (derived-stats.md); attacks, damage and Critical Hits (attacks.md,
  hit-points-and-damage.md); conditions, Immunity and Half Cover (conditions.md);
  saving throws (d20-tests.md).
- Positions ("within" an Emanation, "ally", "that you can see") and the clock: the
  host supplies those facts and reports that a duration elapsed (FR-076).
- Feat effects and qualification; "Breaking Your Oath" (p. 54, advice with no
  rule); optional rules (none named, so no `RuleOptions`); display strings and Oath
  flavour prose beyond the tenets.

## 2. Source

PDF page numbers equal printed page numbers in SRD 5.2.1 (no offset).

- SRD 5.2.1, "Paladin", "Core Paladin Traits", "Paladin Features", p. 53;
  "Paladin Class Features" (Level 1 Lay On Hands, Spellcasting, Weapon Mastery;
  Level 2 Fighting Style, Paladin's Smite; Level 3 Channel Divinity, Paladin
  Subclass; Level 4 Ability Score Improvement; Level 5 Extra Attack, Faithful
  Steed; Level 6 Aura of Protection; Level 9 Abjure Foes; Level 10 Aura of
  Courage; Level 11 Radiant Strikes; Level 14 Restoring Touch; Level 18 Aura
  Expansion; Level 19 Epic Boon), pp. 53-55 (Divine Sense on p. 55); "Paladin
  Spell List", pp. 55-56.
- "Paladin Subclass: Oath of Devotion": tenets, Oath of Devotion Spells, Sacred
  Weapon, Aura of Devotion, p. 56; Smite of Protection, Holy Nimbus, p. 57.
- "Becoming a Paladin ..." (As a Level 1 Character, As a Multiclass Character),
  p. 53; "Equipment", "Weapon Properties", "Thrown", p. 90: a Thrown Melee weapon
  keeps its Melee type (FR-086).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Paladin, Core Paladin Traits, Paladin Features, Oath of Devotion | `ClassDefinition`, `ProgressionRow`, `SubclassDefinition` (content-classes.md); `SubclassId` `oath_of_devotion` | Data only; not the state type `PaladinState` |
| Lay On Hands | `LayOnHandsPool` | Hit Points, five times Paladin level (p. 53) |
| Weapon Mastery; Fighting Style | `WeaponMasteryChoice` (weapon-properties-and-mastery.md); `FightingStyleChoice` (feats-behaviour.md), wrapped as `PaladinFightingStyle` (FR-038) | Mastery not held here; Blessed Warrior is a variant `BlessedWarrior` (two Cleric cantrips, p. 54) |
| Paladin's Smite, Faithful Steed | `FreeCastFeature.paladinsSmite`, `.faithfulSteed` (in `FreeCasts`, each a `LongRestUse`) | Divine Smite / Find Steed once per Long Rest without a slot |
| Channel Divinity; Divine Sense, Abjure Foes, Sacred Weapon | `PaladinChannelDivinityUses`; `PaladinChannelDivinityOption` (enum) | 2 uses, 3 from level 11 |
| Auras of Protection, Courage, Devotion; Aura Expansion | `AuraEffects` | Emanation, radius in feet; 30 from level 18 |
| Extra Attack; Radiant Strikes; Restoring Touch | `attacksPerAttackAction`; `RadiantStrikes` (extra 1d8 Radiant); level gate in `LayOnHandsRules` | |
| Holy Nimbus; Smite of Protection | `HolyNimbusEffect`; `SmiteOfProtectionEffect` | Level 20 and level 15 subclass features |
| Ability Score Improvement, Epic Boon | `ClassDefinition.featGrantsAt(level)`, a `FeatCategory` list (content-classes.md FR-049) | No Paladin-local entitlement type |
| Bright Light, Dim Light, Half Cover | `LightLevel.brightLight`, `.dimLight` (content-schema-and-decoder.md FR-133, referenced); Paladin-local `CoverGrant` {half} | Cover is owned by attacks.md; only `radiusFeet` is data |
| Melee weapon, Ranged weapon, Unarmed Strike, spell attack | `PaladinAttackKind` {meleeWeapon, rangedWeapon, unarmedStrike, spellAttack}; `.forWeapon(WeaponType)` (content-equipment.md FR-101) | Input of `RadiantStrikes.appliesTo` (FR-086) |
| Holy Nimbus use, free casts | `LongRestUse` {available, spent} | One named state instead of a flag (FR-111) |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

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

Stable ids; never renumber. A choice or use command carries the nullable
`PaladinState` (from `FeatureStates`, FR-121) and the host facts it needs;
untrusted-input failures are rejection results with typed reason enums, never
errors. Withdrawn ids: section 4.12.

### 4.1 Class data guard and Oath spells

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `NewPaladinProgression` takes the decoded `ClassDefinition` (content-classes.md) and MUST reject one whose id is not `paladin`, that has no Channel Divinity column, or that lacks a `FeatureEntry` for any feature whose level gates behaviour here (Channel Divinity, Paladin's Smite, Faithful Steed, Extra Attack, Aura of Protection, Abjure Foes, Aura of Courage, Radiant Strikes, Restoring Touch, Aura Expansion; and, when the subclass is given, `SubclassDefinition` entries Sacred Weapon, Aura of Devotion, Smite of Protection, Holy Nimbus). It MUST list every problem found, not only the first, as `PaladinDataProblem` values (a Paladin-local value object: a `PaladinDataProblemKind` {wrongClass, missingChannelDivinityColumn, missingFeatureEntry} and the feature name). The use case maps them to `ContentProblem` values and wraps them in `ContentDecodeFailed` (content-schema-and-decoder.md); `domain/` cannot import that `application/` type. There is no Paladin-local `ClassDataInvalid` type (shared-types-and-ownership.md FR-020). The guard does not repeat what the decoder guarantees: 20 rows in order is content-classes.md FR-037; the absence of a Cantrips column and the slot and prepared columns is content-classes.md FR-033. | `doc/rules/architecture.md` §1, §5, §6; content-classes.md FR-033, FR-037 | unhappy (invalid input) |
| FR-023 | A spell granted by Oath of Devotion that is not on the Paladin list (Beacon of Hope, Freedom of Movement, Guardian of Faith, Commune, Flame Strike) MUST still be returned by `alwaysPrepared` (FR-058): the output is not filtered by membership of the Paladin spell list, and is passed to spellcasting.md as always-prepared input. Whether such a spell counts as a Paladin spell is spellcasting.md FR-045. | SRD 5.2.1, "Level 1: Spellcasting" ("those spells otherwise count as Paladin spells"), p. 54; "Oath of Devotion Spells", p. 56 | unhappy (rule interaction) |

### 4.2 Fighting Style and Blessed Warrior

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-038 | A Fighting Style choice MUST be accepted only at Paladin level 2 or higher and only once, and MUST be a `PaladinFightingStyle`: either `PaladinFightingStyleFeat` wrapping a `FightingStyleChoice` (feats-behaviour.md; a feat of category Fighting Style, whose own effects and qualification are that spec's) or `BlessedWarrior`. A feat of another category MUST be rejected with `FeatOutsideGrant` (feats-behaviour.md FR-043, delegated; no local category check); a repeat `alreadyChosen`; level 1 `levelTooLow`. `PaladinFightingStyle` is a Paladin-local sum type because a closed hierarchy cannot span libraries (shared-types-and-ownership.md FR-018, FR-020) and Blessed Warrior is not a feat. | SRD 5.2.1, "Level 2: Fighting Style", p. 54 | unhappy (state error) |
| FR-039 | Blessed Warrior MUST take exactly 2 distinct cantrips, each of spell level 0 and with Cleric in its class list. Fewer, more, a duplicate, a non-cantrip or a non-Cleric spell MUST be rejected (`wrongCount`, `duplicate`, `notACantrip`, `notOnClericList`). | SRD 5.2.1, "Level 2: Fighting Style" (Blessed Warrior), p. 54 | unhappy (invalid input) |
| FR-040 | The Blessed Warrior cantrips MUST count as Paladin spells for the character, with Charisma as their spellcasting ability; they MUST NOT count against the prepared limit, and MUST be handed to spellcasting.md as one `GrantedCantrip(spellId, SpellcastingClass.paladin, ClassId cleric)` each (spellcasting.md FR-097: the casting class whose ability and DC apply, and the class list the feature names). | SRD 5.2.1, "Level 2: Fighting Style" (Blessed Warrior), p. 54; spellcasting.md FR-097 | happy |
| FR-041 | Whenever a Paladin level is gained, one Blessed Warrior cantrip MAY be replaced by another Cleric cantrip. This spec rejects only what is Paladin-specific: a replacement without Blessed Warrior (`notBlessedWarrior`) and a replacement with a cantrip already held (`duplicate`); the replacement also fails the FR-039 checks (`notACantrip`, `notOnClericList`). The replacement count (at most one) and the trigger (a gained level) are spellcasting.md FR-098, not repeated here. Replacing none MUST be accepted. | SRD 5.2.1, "Level 2: Fighting Style" (Blessed Warrior), p. 54; spellcasting.md FR-098 | unhappy (boundary) |
| FR-042 | `pendingChoices(level, features)` MUST list, in the fixed order below, every Paladin-owned choice the level grants and the character has not made: fighting style (2+), then one feat selection per entitlement of FR-088 not yet used. Skills and Starting Equipment, weapon masteries and the subclass are listed by their owners. A level 1 Paladin MUST list none; a level 2 Paladin with no fighting style MUST list it; a fully chosen level 20 Paladin MUST list none. | SRD 5.2.1, "Paladin Features", "Paladin Class Features", pp. 53-55 | unhappy (empty and minimal) |

### 4.3 Lay On Hands and Restoring Touch

Pool: `LayOnHandsPool(maximum, remaining)`. Table H gives the maximum by level.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-043 | The pool maximum MUST be 5 times the Paladin level (Table H). A Paladin level outside 1..20 MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error) (programmer error: the level type is validated by the domain, not host input). | SRD 5.2.1, "Level 1: Lay On Hands", p. 53; "Paladin Features", p. 53 | per case (Table H): happy / unhappy (boundary) |
| FR-044 | A Paladin who gains the feature MUST start with `remaining` equal to `maximum`. | Design decision D1 (the SRD gives only the maximum, "Level 1: Lay On Hands", p. 53) | happy |
| FR-045 | `LayOnHandsPool` MUST enforce `0 <= remaining <= maximum` and `maximum >= 5`; a violation panics with an error value. | `doc/rules/architecture.md` §3, §6 | unhappy (invalid input) |
| FR-046 | Finishing a Long Rest MUST set `remaining` to `maximum`; finishing a Short Rest MUST leave it unchanged. | SRD 5.2.1, "Level 1: Lay On Hands" ("replenishes when you finish a Long Rest"), p. 53 | happy |
| FR-047 | Lay On Hands MUST restore a requested number of Hit Points, from 1 up to `remaining`, to the touched creature (which may be the Paladin) and reduce `remaining` by exactly that number. | SRD 5.2.1, "Level 1: Lay On Hands", p. 54 | happy |
| FR-048 | A request above `remaining` MUST be rejected with `exceedsPool`, with the pool unchanged; a request equal to `remaining` MUST succeed and leave the pool at 0. | SRD 5.2.1, "Level 1: Lay On Hands" ("up to the maximum amount remaining in the pool"), p. 54 | unhappy (boundary) |
| FR-049 | The restore amount is host input, so it MUST NOT panic. A request that restores 0 Hit Points and removes no condition MUST be rejected with `emptyRequest`; a negative restore amount with `invalidAmount`. A use with the pool at 0 and a positive request MUST be rejected with `exceedsPool`. Panic is kept for programmer errors only (an inconsistent `PaladinState`, FR-045). | `doc/rules/architecture.md` §6; SRD 5.2.1, p. 54 | unhappy (empty and minimal) |
| FR-050 | A use MAY remove the Poisoned condition by expending 5 Hit Points from the pool; those 5 points MUST NOT also restore Hit Points. It MUST be available at every Paladin level 1 to 20, and rejected with `exceedsPool` when `remaining` is below 5. | SRD 5.2.1, "Level 1: Lay On Hands", p. 54 | unhappy (boundary: pool 4 and 5) |
| FR-051 | Restoring Touch MUST be available from level 14 (the level of its `FeatureEntry`, FR-125): with a use of Lay On Hands the Paladin MAY also remove Blinded, Charmed, Deafened, Frightened, Paralyzed or Stunned. Below level 14, requesting any of these MUST be rejected with `restoringTouchNotAvailable`. | SRD 5.2.1, "Level 14: Restoring Touch", p. 55 | unhappy (boundary: levels 13 and 14) |
| FR-052 | Each condition removed by Restoring Touch MUST cost 5 Hit Points from the pool, and those points MUST NOT also restore Hit Points. | SRD 5.2.1, "Level 14: Restoring Touch", p. 55 | happy |
| FR-053 | One use MAY combine restoring Hit Points, removing Poisoned and removing Restoring Touch conditions. The total cost (restore amount plus 5 for each condition removed, Poisoned included) MUST NOT exceed `remaining`; otherwise the whole use MUST be rejected with `exceedsPool` and the pool unchanged. | SRD 5.2.1, "Level 1: Lay On Hands", "Level 14: Restoring Touch", pp. 54-55 | unhappy (rule interaction: atomic cost) |
| FR-054 | A condition outside Poisoned and the six of FR-051 (for example Prone or Petrified) MUST be rejected with `conditionNotRemovable`. | SRD 5.2.1, "Level 1: Lay On Hands", "Level 14: Restoring Touch", pp. 54-55 | unhappy (invalid input) |
| FR-055 | When several rejection reasons apply, all MUST be reported, in this fixed order: `invalidAmount`, `emptyRequest`, `conditionNotRemovable`, `restoringTouchNotAvailable`, `exceedsPool`. | `doc/rules/testing.md` §4 (determinism) | unhappy (rule interaction) |
| FR-056 | `LayOnHandsRules.use` (domain) MUST, for an accepted use, return the new `LayOnHandsPool`, the Hit Points to restore, the conditions to remove (`ConditionKind`, FR-120), the action cost (`PaladinActionCost.bonusAction`) and one `LayOnHandsUsed` event carrying the `characterId`, the Hit Points, the conditions and the pool remaining. The use case wraps that in `UseLayOnHandsApplied`. The host applies Hit Points and conditions and owns action economy. | `doc/rules/architecture.md` §4 | happy |

Table H: pool maximum (FR-043; one test per row).

| Label | Paladin level | Maximum | Path |
|---|---|---|---|
| lowest level | 1 | 5 | unhappy (boundary) |
| level 2 | 2 | 10 | happy |
| level 5 | 5 | 25 | happy |
| level 10 | 10 | 50 | happy |
| level 11 | 11 | 55 | happy |
| level 20 | 20 | 100 | unhappy (boundary) |
| below the range | 0 | panic | unhappy (invalid input) |
| above the range | 21 | panic | unhappy (invalid input) |

Table U: uses (FR-047 to FR-055; one test per row; conditions are removals requested).

| Label | Level | Pool remaining | Request | Expected | FR | Path |
|---|---|---|---|---|---|---|
| restore part | 5 | 25 | restore 10 | accepted, remaining 15 | FR-047 | happy |
| restore all | 5 | 25 | restore 25 | accepted, remaining 0 | FR-048 | unhappy (boundary) |
| restore one too many | 5 | 25 | restore 26 | `exceedsPool`, unchanged | FR-048 | unhappy (boundary) |
| pool empty | 5 | 0 | restore 1 | `exceedsPool` | FR-049 | unhappy (empty and minimal) |
| cure Poisoned exactly | 1 | 5 | Poisoned | accepted, remaining 0, restores 0 | FR-050 | unhappy (boundary) |
| cure Poisoned one short | 1 | 4 | Poisoned | `exceedsPool` | FR-050 | unhappy (boundary) |
| Restoring Touch at 13 | 13 | 65 | Blinded | `restoringTouchNotAvailable` | FR-051 | unhappy (boundary) |
| Restoring Touch at 14 | 14 | 70 | Blinded | accepted, remaining 65 | FR-051, FR-052 | unhappy (boundary) |
| two conditions | 14 | 70 | Blinded, Stunned | accepted, remaining 60 | FR-052 | happy |
| combined, exact | 14 | 20 | restore 5, Blinded, Stunned | accepted (5 + 10 = 15), remaining 5 | FR-053 | happy |
| combined, one over | 14 | 14 | Blinded, Stunned, Frightened | `exceedsPool` (15 > 14), unchanged | FR-053 | unhappy (rule interaction) |
| Poisoned and Charmed | 14 | 20 | Poisoned, Charmed | accepted (5 + 5), remaining 10 | FR-053 | happy |
| not removable | 14 | 70 | Prone | `conditionNotRemovable` | FR-054 | unhappy (invalid input) |
| two reasons | 13 | 4 | Prone, Blinded, restore 1 | `conditionNotRemovable`, `restoringTouchNotAvailable`, `exceedsPool` (in that order) | FR-055 | unhappy (rule interaction) |
| empty request | 5 | 25 | restore 0, no condition | `emptyRequest`, unchanged | FR-049 | unhappy (empty and minimal) |
| negative amount | 5 | 25 | restore -1 | `invalidAmount`, unchanged | FR-049 | unhappy (invalid input) |
| negative and not removable | 5 | 25 | restore -1, Prone | `invalidAmount`, `conditionNotRemovable` (in that order) | FR-055 | unhappy (rule interaction) |

### 4.4 Spellcasting inputs, free casts, always-prepared spells

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-057 | The Paladin's spellcasting data MUST be: spellcasting ability Charisma, Spellcasting Focus Holy Symbol, prepared-list change after a Long Rest, one spell replaceable. These MUST agree with the Paladin row of spellcasting.md. | SRD 5.2.1, "Level 1: Spellcasting", p. 54 | happy |
| FR-058 | `alwaysPrepared(level, subclass)` MUST read the grants from the loaded definitions, not from constants: the `AlwaysPreparedSpells` facts (content-classes.md FR-058) of the Paladin's Smite and Faithful Steed entries of the `ClassDefinition` (Divine Smite from level 2, Find Steed from level 5 in the SRD data) and of the Oath of Devotion Spells entry of the chosen `SubclassDefinition` (every grant whose level is at or below the Paladin level), the subclass part only when Oath of Devotion is chosen. Table A is the fixture for the SRD data, not the source of the behaviour; a test with a modified definition proves the function follows the definition. | SRD 5.2.1, "Level 2: Paladin's Smite", "Level 5: Faithful Steed", pp. 54-55; "Level 3: Oath of Devotion Spells", p. 56 | per case (Table A): happy / unhappy (boundary) |
| FR-059 | The output of `alwaysPrepared` MUST be passed to spellcasting.md as its always-prepared set. That an always-prepared spell is not counted against the prepared limit is spellcasting.md FR-045 and FR-047; this spec adds no counting rule. | SRD 5.2.1, "Level 1: Spellcasting" ("those spells don't count against the number of spells you can prepare"), p. 54 | unhappy (rule interaction) |
| FR-060 | Paladin's Smite MUST be available from level 2 (the level of its `FeatureEntry`, FR-125): Divine Smite may be cast once without expending a spell slot, and its use is `LongRestUse.available` on gaining the feature (FR-098). Before level 2 a free cast MUST be rejected with `featureNotGained`; availability is a named state (`LongRestUse`), the gate is the level, and the two are never mixed. The cast itself is not performed here: `UseFreeCast` only spends the availability and returns the spell id, and the host then casts it with casting-and-concentration.md `CastSpell` and `CastMode.featureGranted` (its FR-121; no slot spent). This spec imports nothing from that spec. | SRD 5.2.1, "Level 2: Paladin's Smite", p. 54; "Casting without Slots", p. 104 | unhappy (boundary: levels 1 and 2) |
| FR-061 | Faithful Steed MUST be available from level 5 (its `FeatureEntry` level, FR-125): Find Steed may be cast once without expending a spell slot, and its use is `LongRestUse.available` on gaining the feature (FR-098). Before level 5 a free cast MUST be rejected with `featureNotGained`. The cast is made by the host as in FR-060. | SRD 5.2.1, "Level 5: Faithful Steed", p. 55; "Casting without Slots", p. 104 | unhappy (boundary: levels 4 and 5) |
| FR-062 | A free cast whose use is `LongRestUse.spent` MUST NOT be spent again before a Long Rest: it MUST be rejected with `alreadyUsedSinceLongRest`. | SRD 5.2.1, "Level 2: Paladin's Smite" ("you must finish a Long Rest before you can cast it this way again"), p. 54; "Level 5: Faithful Steed", p. 55 | unhappy (state error) |
| FR-063 | Spending a free cast MUST change only that free cast: the other free cast, the Channel Divinity uses, the Lay On Hands pool and the spell slots MUST be unchanged. The two free casts MUST be independent. | SRD 5.2.1, "Level 2: Paladin's Smite", p. 54; "Level 5: Faithful Steed", p. 55 | unhappy (rule interaction) |
| FR-064 | Finishing a Long Rest MUST make both free casts available again; finishing a Short Rest MUST NOT. | SRD 5.2.1, "Level 2: Paladin's Smite", p. 54; "Level 5: Faithful Steed" ("you regain the ability to do so when you finish a Long Rest"), p. 55 | happy |

Table A: always-prepared spells (FR-058; one test per row; subclass is Oath of
Devotion unless "none").

| Label | Level | Subclass | Always prepared | Path |
|---|---|---|---|---|
| level 1 | 1 | none | none | unhappy (empty case) |
| Paladin's Smite | 2 | none | Divine Smite | unhappy (boundary) |
| level 3, subclass not yet chosen | 3 | none | Divine Smite | unhappy (state error) |
| level 3, Oath | 3 | Oath | Divine Smite, Protection from Evil and Good, Shield of Faith | unhappy (boundary) |
| level 4, Oath | 4 | Oath | same as level 3 | happy |
| level 5, no subclass | 5 | none | Divine Smite, Find Steed | unhappy (rule interaction) |
| level 5, Oath | 5 | Oath | Divine Smite, Find Steed, Protection from Evil and Good, Shield of Faith, Aid, Zone of Truth | happy |
| level 9, Oath | 9 | Oath | level 5 set, plus Beacon of Hope, Dispel Magic | happy |
| level 13, Oath | 13 | Oath | level 9 set, plus Freedom of Movement, Guardian of Faith | happy |
| level 16, Oath | 16 | Oath | same as level 13 | unhappy (boundary: one below 17) |
| level 17, Oath | 17 | Oath | level 13 set, plus Commune, Flame Strike (12 spells) | unhappy (boundary) |
| level 20, Oath | 20 | Oath | same 12 spells as level 17 | unhappy (boundary: maximum level) |

### 4.5 Channel Divinity and its options

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-065 | The maximum Channel Divinity uses MUST be 0 below level 3, 2 at levels 3 to 10 and 3 at levels 11 to 20, read from the Channel Divinity column of the table. | SRD 5.2.1, "Paladin Features", p. 53; "Level 3: Channel Divinity", p. 54 | unhappy (boundary: levels 2, 3, 10, 11) |
| FR-066 | A Paladin gaining Channel Divinity at level 3 MUST start with remaining uses equal to the maximum; reaching level 11 MUST raise the maximum and the remaining uses by 1 each (never above the maximum). | SRD 5.2.1, "Level 3: Channel Divinity" ("You gain an additional use when you reach Paladin level 11"), p. 54, for the maximum only; the rise of the remaining uses is design decision D1 | unhappy (boundary: level 11) |
| FR-067 | Finishing a Short Rest MUST regain one expended use, never above the maximum; finishing a Long Rest MUST regain all expended uses. A Short Rest at maximum uses MUST change nothing. | SRD 5.2.1, "Level 3: Channel Divinity" ("You regain one of its expended uses when you finish a Short Rest, and you regain all expended uses when you finish a Long Rest"), p. 54 | unhappy (boundary: 0, 1 and maximum uses) |
| FR-068 | Using a Channel Divinity effect MUST spend exactly one use. With 0 uses remaining it MUST be rejected with `noUsesRemaining`; below level 3 with `featureNotGained`; both leave the state unchanged. | SRD 5.2.1, "Level 3: Channel Divinity", p. 54 | unhappy (state error) |
| FR-069 | The options MUST be: Divine Sense from level 3, Abjure Foes from level 9, and Sacred Weapon from level 3 with Oath of Devotion chosen (each level is that of the feature's `FeatureEntry`, FR-125). An option not yet available MUST be rejected with `optionNotAvailable`. | SRD 5.2.1, "Level 3: Channel Divinity", p. 54; "Level 9: Abjure Foes", p. 55; "Level 3: Sacred Weapon", p. 56 | unhappy (boundary: levels 8 and 9; no subclass) |
| FR-070 | A Channel Divinity effect that requires a saving throw MUST use as its DC the `SpellSaveDc` (spellcasting.md; it wraps a `DifficultyClass`) supplied in the command; the use case MUST NOT recompute it, and the effect data carries that `SpellSaveDc` unchanged. | SRD 5.2.1, "Level 3: Channel Divinity" ("the DC equals the spell save DC"), p. 55 | happy |
| FR-071 | Divine Sense MUST return effect data: Bonus Action; 10 minutes or until the Incapacitated condition; radius 60 feet; detects, as typed values, creature types {Celestial, Fiend, Undead} (`CreatureType` set) with detection kinds {location, creatureType} (`DetectionKind`), and places or objects consecrated or desecrated, as with the Hallow spell (`DetectionKind.consecratedOrDesecrated`). | SRD 5.2.1, "Level 3: Channel Divinity", "Divine Sense", p. 55 | happy |
| FR-072 | Abjure Foes MUST return effect data: Magic action; targets up to the larger of 1 and the Charisma `AbilityModifier` among creatures the Paladin can see within 60 feet; Wisdom saving throws at the `SpellSaveDc` of FR-070; on a failure Frightened for 1 minute or until it takes damage; while Frightened the target can do only one of: move, take an action, take a Bonus Action. The number of targets MUST follow Table B. | SRD 5.2.1, "Level 9: Abjure Foes", p. 55 | per case (Table B): happy / unhappy (boundary) |
| FR-073 | Sacred Weapon MUST be rejected unless the host reports that the Paladin is taking the Attack action (`notTakingAttackAction`) and holding a Melee weapon (`noMeleeWeaponHeld`). | SRD 5.2.1, "Level 3: Sacred Weapon", p. 56 | unhappy (state error) |
| FR-074 | Sacred Weapon MUST return effect data: the chosen Melee weapon; 10 minutes or until used again; attack roll bonus equal to the larger of 1 and the Charisma modifier; damage of its normal type or Radiant; `LightLevel.brightLight` radius 20 feet and `LightLevel.dimLight` a further 20 feet (content-schema-and-decoder.md FR-133). The bonus MUST follow Table W. | SRD 5.2.1, "Level 3: Sacred Weapon", p. 56 | per case (Table W): happy / unhappy (boundary) |
| FR-075 | Using Sacred Weapon while it is active MUST spend another use, replace the earlier imbuement and leave exactly one imbued weapon; the replaced weapon's effect ends with reason `usedAgain`. | SRD 5.2.1, "Level 3: Sacred Weapon" ("or until you use this feature again"), p. 56 | unhappy (rule interaction) |
| FR-076 | `PaladinEffects.end` (a synchronous pure domain function: it needs no content, FR-114) MUST end Divine Sense for `elapsed`, `incapacitated`; Sacred Weapon for `elapsed`, `endedEarly` (no action), `notCarried`, `usedAgain`; Holy Nimbus for `elapsed`, `endedEarly`. Ending an effect that is not active, or with a reason the effect does not have, MUST be rejected with `notActive` or `reasonNotApplicable`. Ending MUST NOT refund a Channel Divinity use. | SRD 5.2.1, "Divine Sense", p. 55; "Level 3: Sacred Weapon", p. 56; "Level 20: Holy Nimbus", p. 57 | unhappy (state error) |

Table B: Abjure Foes target count (FR-072) and Table W: Sacred Weapon attack
bonus (FR-074); one test per row.

| Label | Charisma modifier | Abjure Foes targets | Sacred Weapon bonus | Path |
|---|---|---|---|---|
| lowest modifier | -5 | 1 | n/a | unhappy (boundary) |
| negative | -2 | n/a | +1 | unhappy (boundary) |
| negative | -1 | 1 | n/a | unhappy (boundary) |
| zero | 0 | 1 | +1 | unhappy (boundary) |
| one | +1 | 1 | +1 | unhappy (boundary) |
| two | +2 | 2 | n/a | happy |
| three | +3 | n/a | +3 | happy |
| five | +5 | n/a | +5 | happy |
| highest modifier | +10 | 10 | n/a | unhappy (boundary) |

### 4.6 Auras

`AuraEffects(radiusFeet, saveBonus, immunities, ...)` (the save bonus is a
`Modifier`) is computed from the Paladin level, the subclass, the Charisma
`AbilityModifier` and the host fact `hasIncapacitatedCondition`. The level gates
(6, 7, 10, 15, 18) are `FeatureEntry` levels (FR-125); the radii, the larger-of-1
rule and the immunities are SRD formulas kept as constants.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-077 | Aura of Protection MUST be absent below level 6 and present from level 6. | SRD 5.2.1, "Level 6: Aura of Protection", p. 55 | unhappy (boundary: levels 5 and 6) |
| FR-078 | The aura MUST be a 10-foot Emanation at levels 6 to 17 and a 30-foot Emanation from level 18 (Aura Expansion). | SRD 5.2.1, "Level 6: Aura of Protection", "Level 18: Aura Expansion", p. 55 | unhappy (boundary: levels 17 and 18) |
| FR-079 | The Paladin and allies in the aura MUST gain a bonus to saving throws equal to the larger of 1 and the Paladin's Charisma modifier. | SRD 5.2.1, "Level 6: Aura of Protection", p. 55 | per case (Table AU): happy / unhappy (boundary) |
| FR-080 | The aura MUST be inactive, with no radius, bonus or immunity, while the Paladin has the Incapacitated condition. | SRD 5.2.1, "Level 6: Aura of Protection", p. 55 | unhappy (state error) |
| FR-081 | A creature MUST benefit from only one Aura of Protection at a time, the one it chooses. `chooseAura(candidates, chosen)` MUST return the chosen bonus; a chosen aura not among the candidates MUST be rejected with `notACandidate`; no candidates MUST give no bonus. | SRD 5.2.1, "Level 6: Aura of Protection" ("a creature can benefit from only one Aura of Protection at a time"), p. 55 | unhappy (rule interaction) |
| FR-082 | Aura of Courage MUST give the Paladin and allies Immunity to the Frightened condition while in an active Aura of Protection, from level 10; below level 10 it MUST give none. The suspension of a Frightened condition for an ally in the aura is expressed only as the `immunities` set of `AuraEffects`, which the host applies while an ally is in the aura; this spec models no creature entering the aura. | SRD 5.2.1, "Level 10: Aura of Courage", p. 55 | unhappy (boundary: levels 9 and 10) |
| FR-083 | Aura of Devotion MUST give the same kind of Immunity to Charmed from level 7 with Oath of Devotion chosen, and none at level 6 or without the subclass; as in FR-082 it is expressed as the `immunities` set the host applies. | SRD 5.2.1, "Level 7: Aura of Devotion", p. 56 | unhappy (boundary: levels 6 and 7) |
| FR-084 | Smite of Protection MUST give the Paladin and allies Half Cover while in an active aura, until the start of the Paladin's next turn, whenever Divine Smite is cast, from level 15 with Oath of Devotion; the `ApplySmiteOfProtection` use case (FR-112) takes the host fact `divineSmiteCast` and MUST return `SmiteOfProtectionEffect` data (`CoverGrant.half`) with one `SmiteOfProtectionTriggered` event, and give no effect (`ApplySmiteOfProtectionNoEffect`, no event) below level 15, without the subclass, while the aura is inactive, or when `divineSmiteCast` is false. A no-effect outcome is a normal success of the evaluation, not a failure (FR-115, FR-117). | SRD 5.2.1, "Level 15: Smite of Protection", p. 57 | unhappy (boundary: levels 14 and 15) |

Table AU: auras (FR-077 to FR-083; one test per row; "none" means absent or
inactive; Oath = Oath of Devotion).

| Label | Level | Subclass | Charisma modifier | Incapacitated | Radius (ft) | Save bonus | Immunities | FR | Path |
|---|---|---|---|---|---|---|---|---|---|
| before the feature | 5 | none | +3 | no | none | none | none | FR-077 | unhappy (boundary) |
| first level | 6 | none | +3 | no | 10 | +3 | none | FR-077, FR-079 | unhappy (boundary) |
| modifier -1 | 6 | none | -1 | no | 10 | +1 | none | FR-079 | unhappy (boundary) |
| modifier 0 | 6 | none | 0 | no | 10 | +1 | none | FR-079 | unhappy (boundary) |
| modifier +1 | 6 | none | +1 | no | 10 | +1 | none | FR-079 | unhappy (boundary) |
| Charmed one level early | 6 | Oath | +2 | no | 10 | +2 | none | FR-083 | unhappy (boundary) |
| Charmed | 7 | Oath | +2 | no | 10 | +2 | Charmed | FR-083 | happy |
| Frightened one level early | 9 | none | +2 | no | 10 | +2 | none | FR-082 | unhappy (boundary) |
| Frightened | 10 | none | +2 | no | 10 | +2 | Frightened | FR-082 | happy |
| both | 10 | Oath | +2 | no | 10 | +2 | Charmed, Frightened | FR-082, FR-083 | happy |
| Incapacitated | 10 | Oath | +2 | yes | none | none | none | FR-080 | unhappy (state error) |
| last small radius | 17 | Oath | +4 | no | 10 | +4 | Charmed, Frightened | FR-078 | unhappy (boundary) |
| Aura Expansion | 18 | Oath | +4 | no | 30 | +4 | Charmed, Frightened | FR-078 | unhappy (boundary) |
| maximum level | 20 | Oath | +5 | no | 30 | +5 | Charmed, Frightened | FR-078, FR-079 | unhappy (boundary) |

Table SP: Smite of Protection (FR-084, FR-112; one test per row).

| Label | Level | Subclass | Divine Smite cast | Aura active | Expected | Path |
|---|---|---|---|---|---|---|
| before the feature | 14 | Oath | yes | yes | no effect | unhappy (boundary) |
| first level | 15 | Oath | yes | yes | Half Cover until the start of the Paladin's next turn | unhappy (boundary) |
| no cast | 15 | Oath | no | yes | no effect | unhappy (state error) |
| no subclass | 15 | none | yes | yes | no effect | unhappy (rule interaction) |
| Incapacitated | 15 | Oath | yes | no | no effect | unhappy (state error) |

### 4.7 Extra Attack, Radiant Strikes, feat entitlements

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-085 | `attacksPerAttackAction` MUST be 1 below the level of the Extra Attack `FeatureEntry` (5 in the SRD data, FR-125) and 2 from it. Combining with another class's Extra Attack is out of scope. | SRD 5.2.1, "Level 5: Extra Attack", p. 55 | unhappy (boundary: levels 4 and 5) |
| FR-086 | `RadiantStrikes.appliesTo(level, PaladinAttackKind)` (a pure domain function; the gate is the Radiant Strikes `FeatureEntry` level, 11 in the SRD data, FR-125) MUST be true, from that level, for `meleeWeapon` and `unarmedStrike` and false for `rangedWeapon`, `spellAttack`, or below that level. The host builds the `PaladinAttackKind` of a weapon attack with `PaladinAttackKind.forWeapon(WeaponType)` from the weapon's own `WeaponType` (content-equipment.md FR-101), so the weapon's own Melee or Ranged type decides, not how it was used: a thrown Javelin is `WeaponType.melee` and qualifies (Table X). The attack itself is resolved by attacks.md, which is downstream; this spec only answers the question. | SRD 5.2.1, "Level 11: Radiant Strikes", p. 55; "Weapon Properties", "Thrown", p. 90 | unhappy (boundary: levels 10 and 11; rule interaction by attack kind) |
| FR-087 | Radiant Strikes MUST expose its extra damage as data: 1d8, Radiant, once per qualifying hit. | SRD 5.2.1, "Level 11: Radiant Strikes", p. 55 | happy |
| FR-088 | `featEntitlements(level)` MUST be derived from `ClassDefinition.featGrantsAt(level)` (content-classes.md FR-049), a list of `FeatCategory`, minus the grants of category Fighting Style (the level 2 Fighting Style is also a `GrantsFeat` fact there, but it is chosen under FR-038, so FR-042 lists it once and it is not counted here), with no Paladin-local entitlement type: in the SRD data one grant at each of levels 4, 8, 12, 16 (Ability Score Improvement, or another feat for which the character qualifies) and one at level 19 (Epic Boon, or another feat for which the character qualifies), and none at any other level. The total by level 20 MUST be five. Table E is the fixture for that data. content-classes.md FR-049 is asked to name the Epic Boon (level 19) among its grants; it names only the Ability Score Improvement levels today. | SRD 5.2.1, "Level 4: Ability Score Improvement", "Level 19: Epic Boon", p. 55 | per case (Table E): happy / unhappy (boundary) |

Table X: Radiant Strikes by attack (FR-086; one test per row).

| Label | Level | Attack | Applies | Path |
|---|---|---|---|---|
| below the feature | 10 | Longsword hit | no | unhappy (boundary) |
| Melee weapon | 11 | Longsword hit | yes | unhappy (boundary) |
| Unarmed Strike | 11 | Unarmed Strike hit | yes | happy |
| thrown Javelin | 11 | Javelin thrown, hit | yes (a Melee weapon with the Thrown property) | unhappy (rule interaction) |
| Ranged weapon | 11 | Longbow hit | no | unhappy (rule interaction) |
| spell attack | 11 | spell attack hit | no | unhappy (rule interaction) |

Table E: feat entitlements (FR-088; one test per row).

| Label | Paladin level | New entitlement | Total so far | Path |
|---|---|---|---|---|
| Fighting Style | 2 | none | 0 | unhappy (rule interaction: Fighting Style is FR-038) |
| none yet | 3 | none | 0 | unhappy (boundary) |
| first | 4 | Ability Score Improvement | 1 | unhappy (boundary) |
| gap | 7 | none | 1 | unhappy (rule interaction: a "Subclass feature" row grants none) |
| second | 8 | Ability Score Improvement | 2 | happy |
| third | 12 | Ability Score Improvement | 3 | happy |
| fourth | 16 | Ability Score Improvement | 4 | happy |
| Epic Boon | 19 | Epic Boon | 5 | unhappy (boundary) |
| last level | 20 | none | 5 | unhappy (boundary) |

### 4.8 Holy Nimbus (Oath of Devotion, level 20)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-089 | Holy Nimbus MUST be rejected with `featureNotGained` below the level of its `FeatureEntry` (20 in the SRD data, FR-125) or without Oath of Devotion chosen. | SRD 5.2.1, "Level 20: Holy Nimbus", p. 57 | unhappy (boundary: levels 19 and 20) |
| FR-090 | Activating Holy Nimbus MUST return effect data: Bonus Action; 10 minutes or until ended (no action); Holy Ward (Advantage on any saving throw the Paladin is forced to make by a creature of a type in the typed `CreatureType` set {Fiend, Undead}); Radiant Damage (a creature that is an enemy and starts its turn in the aura takes Radiant damage equal to the Charisma `AbilityModifier` plus the `ProficiencyBonus`); Sunlight (the aura is filled with `LightLevel.brightLight` that is sunlight). A successful activation sets the use to `LongRestUse.spent` and returns one `HolyNimbusActivated` event. | SRD 5.2.1, "Level 20: Holy Nimbus", p. 57 | happy |
| FR-091 | Once used, Holy Nimbus MUST NOT be activated again until a Long Rest is finished; a second activation MUST be rejected with `alreadyUsedSinceLongRest`. | SRD 5.2.1, "Level 20: Holy Nimbus" ("you can't use it again until you finish a Long Rest"), p. 57 | unhappy (state error) |
| FR-092 | The use MAY be restored by expending one level 5 spell slot, with no action. The command carries the character's `SpellcastingState` and the `SpellSlotPool` (`spellcasting` or `pactMagic`) to draw from, because a Paladin and Warlock multiclass may hold a level 5 slot only in the Pact Magic pool. The use case MUST reject it with `notExpended` when the use is `LongRestUse.available`, and with `noLevel5SlotAvailable` when the chosen pool has no level 5 slot (the `NoSlotAvailable` outcome of spellcasting.md FR-109); on success it MUST expend exactly one level 5 slot of the chosen pool through the domain operation behind spellcasting.md FR-109 `ExpendSpellSlot` (not the use case, so errors count once), return the new `SpellcastingState`, set the use to `available` and return one `HolyNimbusRestored` event. | SRD 5.2.1, "Level 20: Holy Nimbus", p. 57 | unhappy (state error) |
| FR-093 | The Radiant Damage value MUST be the Charisma modifier plus the Proficiency Bonus (Table N). | SRD 5.2.1, "Level 20: Holy Nimbus", p. 57 | per case (Table N): happy / unhappy (boundary) |
| FR-094 | Holy Nimbus MUST NOT spend a Channel Divinity use and MUST NOT depend on how many remain. | SRD 5.2.1, "Level 20: Holy Nimbus" (a separate use of its own), p. 57 | unhappy (rule interaction) |
| FR-095 | While the Paladin has the Incapacitated condition the Aura of Protection is inactive, and the Holy Nimbus effect data MUST report its aura-based effects (Radiant Damage, Sunlight, Holy Ward) as inactive (interpretation, section 9: Holy Nimbus imbues the Aura of Protection, which is inactive while Incapacitated). | SRD 5.2.1, "Level 6: Aura of Protection", p. 55; "Level 20: Holy Nimbus", p. 57 | unhappy (rule interaction) |

Table N: Holy Nimbus Radiant damage (FR-093; one test per row; Proficiency Bonus
at level 20 is +6).

| Label | Charisma modifier | Damage | Path |
|---|---|---|---|
| lowest modifier | -5 | 1 | unhappy (boundary) |
| zero | 0 | 6 | unhappy (boundary) |
| three | +3 | 9 | happy |
| five | +5 | 11 | happy |

### 4.9 Rests and level advancement (Paladin resources only)

Rests and level advancement are class-agnostic and sit to the left of `class-*`
(shared-types-and-ownership.md FR-016, FR-024): the host applies the pure
functions below to `PaladinState` after `RestApplied` or the level-advanced event
(Q13, settled).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-096 | Rest recharge is a pure domain function `PaladinRests.apply(state, RestKind)`. `character` maps `RestKind` to the content `RechargeTrigger` data (shared-types-and-ownership.md FR-009). It MUST affect the Paladin resources as Table R shows and nothing else. Active effects (Divine Sense, Sacred Weapon, Holy Nimbus) MUST be left unchanged by a rest; the host ends them through `PaladinEffects.end` (`elapsed`). A rest has no connection to Weapon Mastery changes (weapon-properties-and-mastery.md). | SRD 5.2.1, "Level 1: Lay On Hands", p. 53; "Level 3: Channel Divinity", p. 54; "Level 2: Paladin's Smite", p. 54; "Level 5: Faithful Steed", p. 55; "Level 20: Holy Nimbus", p. 57 | per case (Table R): happy / unhappy (boundary) |
| FR-098 | Level-up MUST be a pure domain function `PaladinAdvancement.resourcesFor(state, newLevel)` that raises the Lay On Hands maximum by 5 and its remaining by 5, applies FR-066 to Channel Divinity, and sets the use of a feature gained at `newLevel` to `LongRestUse.available`: Paladin's Smite at the level of its `FeatureEntry` (2), Faithful Steed (5) and Holy Nimbus (20) (design decision D1: a newly gained resource starts available, so a Paladin reaching level 2 can use the free cast at once; the `PaladinState` invariant of section 5 makes a not-yet-gained use `available`, so this never un-spends a spent use). It MUST NOT refill any other resource. The level-up transaction, its rejections and the report of features gained are in the Withdrawn ids table (FR-097, FR-099, FR-100), owned by level-advancement-and-multiclassing.md; the host applies this function after the level-advanced event. `newLevel` outside 1..20 is a programmer error (panic) here. | SRD 5.2.1, "Level 1: Lay On Hands", p. 53, for the maximum only; "Level 2: Paladin's Smite", p. 54; "Level 5: Faithful Steed", p. 55; "Level 20: Holy Nimbus", p. 57; the rise of the remaining amount is design decision D1 | unhappy (rule interaction: no refill beyond the gain) |

Table R: rests (FR-096; one test per row; "—" means unchanged; active effects
are unchanged in every row).

| Label | Rest | Lay On Hands pool | Channel Divinity | Paladin's Smite | Faithful Steed | Holy Nimbus | Path |
|---|---|---|---|---|---|---|---|
| Long Rest, all spent | Long | to maximum | to maximum | `available` | `available` | `available` | happy |
| Long Rest, nothing spent | Long | — | — | — | — | — | unhappy (empty and minimal) |
| Long Rest, effect active | Long | to maximum | to maximum | available | available | available; active Sacred Weapon stays active | unhappy (rule interaction) |
| Short Rest, none remaining | Short | — | regain 1 | — | — | — | unhappy (boundary: 0 uses) |
| Short Rest, 1 of 2 remaining | Short | — | regain 1 (now 2) | — | — | — | unhappy (boundary) |
| Short Rest, at maximum | Short | — | — | — | — | — | unhappy (boundary: no change) |
| Short Rest, level 11, 0 of 3 | Short | — | regain 1 (now 1) | — | — | — | unhappy (boundary) |
| Short Rest, spent free casts | Short | — | — | not restored | not restored | not restored | unhappy (rule interaction) |

Table G: advancement (FR-098; one test per row). Only the Resource changes
column is tested here; Features reported is a fixture for
level-advancement-and-multiclassing.md (which owns `maximumLevel` and
`notNextLevel`: level 20 to 21, 4 to 6 and 4 to 4 are rejected there).

| Label | From | To | Subclass | Features reported | Resource changes | Path |
|---|---|---|---|---|---|---|
| level 2 | 1 | 2 | none | Fighting Style, Paladin's Smite | Lay On Hands 5 to 10; Paladin's Smite `available` (usable at once) | happy |
| level 3, no subclass yet | 2 | 3 | none | Channel Divinity, Paladin Subclass | Channel Divinity 0 to 2 uses, 2 remaining | unhappy (state error: choice pending) |
| level 3, subclass chosen at once | 2 | 3 | Oath | Channel Divinity, Paladin Subclass, Oath of Devotion Spells, Sacred Weapon | as above | happy |
| level 5 | 4 | 5 | any | Extra Attack, Faithful Steed | Lay On Hands 20 to 25; Faithful Steed `available` (usable at once) | happy |
| level 7 | 6 | 7 | Oath | Aura of Devotion | none beyond Lay On Hands | happy |
| level 11 | 10 | 11 | any | Radiant Strikes | Channel Divinity 2 to 3, remaining +1 | unhappy (boundary) |
| level 13 | 12 | 13 | any | (none) | Lay On Hands only | unhappy (empty case) |
| level 15 | 14 | 15 | Oath | Smite of Protection | Lay On Hands only | happy |
| level 17 | 16 | 17 | any | (none) | Lay On Hands only | unhappy (empty case) |
| level 20 | 19 | 20 | Oath | Holy Nimbus | Lay On Hands 95 to 100; Holy Nimbus `available` (usable at once) | unhappy (boundary) |

### 4.10 Cross-cutting behaviour

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-101 | The use cases `ChoosePaladinOption`, `UseLayOnHands`, `UsePaladinChannelDivinity`, `UseFreeCast`, `UseHolyNimbus`, `ApplySmiteOfProtection` and `CalculatePaladinFeatures` take an optional `PaladinState` (read by `paladinStateKey`, FR-121; null when the character has no Paladin levels) and MUST return their `...NotAPaladin` variant and no events for null. The pure domain functions `PaladinEffects.end`, `PaladinRests.apply` and `PaladinAdvancement.resourcesFor` take a present `PaladinState` and have no not-a-Paladin outcome; `PaladinFirstLevel.start` takes none (it rejects an existing state, FR-111). `...NotAPaladin` is a returned failure (FR-118), not an error. | `doc/rules/architecture.md` §6 | unhappy (state error) |
| FR-102 | When the `paladin` class entry is not in the content source, a use case that loads content MUST return its `<UseCase>ContentMissing` variant carrying a `MissingContent(kind, id)` field naming the key (content-schema-and-decoder.md FR-147, shared-types-and-ownership.md FR-018), built from `ContentNotFound.missing` or the id's `ContentKeyedId.missing`; when a feat or spell id in a command is not found, the same variant naming that key (the only ids a Paladin command carries are the feat of a Fighting Style and the Blessed Warrior cantrips; no command takes a weapon id). When both `<UseCase>ContentMissing` and a rejection reason would apply, `<UseCase>ContentMissing` MUST win and no rejection reasons are returned; the `<UseCase>ClassDataInvalid` variant then ranks above `<UseCase>ContentMissing`. | `doc/rules/architecture.md` §4 | unhappy (missing content) |
| FR-103 | When the `ContentSource` port returns its typed `*srderr.PortError`, the use case MUST propagate it unchanged. | `doc/rules/architecture.md` §4 (failure contract) | unhappy (failure of collaborator) |
| FR-104 | When the class entry exists but `NewPaladinProgression` rejects it (FR-014), the use case MUST return its `...ClassDataInvalid` variant carrying every problem found in the decoder's failure type. | `doc/rules/architecture.md` §5 | unhappy (missing or bad content) |
| FR-105 | A rejected request MUST return the rejection reasons and no events; the input state MUST be unchanged. This is tested at the domain level and again at application level through each use case. | `doc/rules/architecture.md` §3 | unhappy (state error) |
| FR-106 | For every use case other than Lay On Hands (FR-055), when several rejection reasons apply, all MUST be reported in the order the values appear in the declaration of that use case's rejection reason enum. Rejections are typed reason enums (one per use case) carried by that use case's `...Rejected` variant, not a closed class hierarchy; FR-049 reasons are such enum values. | `doc/rules/testing.md` §4 (determinism) | unhappy (rule interaction) |
| FR-107 | Every state type (`PaladinState`, `LayOnHandsPool`, `PaladinChannelDivinityUses`, the choices, the effect data) MUST be immutable: a transition leaves its input unchanged, and every exposed collection MUST be a defensive copy (exposed slices are defensive copies, so a caller cannot mutate state). | `doc/rules/architecture.md` §3 | unhappy (immutability) |
| FR-108 | Every operation MUST be deterministic and pure: same inputs, same outputs, no `dice.Roller`, clock or `Random`. No Paladin feature in this spec rolls dice. | `doc/rules/architecture.md` §3 (purity) | unhappy (determinism) |
| FR-109 | A use case MUST give the same result with the no-op `telemetry.Telemetry` as with a recording one. | `doc/rules/observability.md` §9 | happy |
| FR-110 | A returned `*srderr.PortError` leaving a use case MUST be recorded once: error span status, one increment of the shared `srd5e.character.feature_used.errors` counter (owned by `character-aggregate.md`, shared-types-and-ownership.md FR-063) with `error.type` set to `srderr.ErrorType(err)` (errors-and-results.md FR-034), no log; the call also increments `srd5e.character.feature_used` once with `srd5e.outcome` = `failure`, no `error.type` and no rejection (character-aggregate.md FR-118). A returned failure result is recorded in one of two ways. (a) `<UseCase>ContentMissing` and `<UseCase>ClassDataInvalid` set the span to error status (observability.md §6), with `srd5e.outcome` = `failure` and `srd5e.character.rejection` = `content_missing` or `class_data_invalid`, and no `error.type`, no `.errors` increment and no log (errors-and-results.md FR-043). (b) `<UseCase>NotAPaladin` and every `<UseCase>Rejected` are normal rule outcomes: `srd5e.outcome` = `failure`, `srd5e.character.rejection` set, no error span status, no `error.type`, no `.errors` increment. Every returned failure counts once in `feature_used` (FR-117). | `doc/rules/observability.md` §7 (errors: who records what) | unhappy (telemetry on failure) |

### 4.11 First level, Smite of Protection, ownership, telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-111 | `PaladinFirstLevel.start(characterId, entry)` (`entry` is a `ClassEntry` {firstLevel, multiclass}, the SRD's two entry modes, not a flag) is a pure synchronous domain function (it needs no content, FR-023). It MUST return a new `PaladinState` holding the `CharacterId` at Paladin level 1 with the Lay On Hands pool at its maximum of 5 (FR-043, FR-044), Channel Divinity uses 0, both free casts and the Holy Nimbus use `LongRestUse.available` (inert until the feature is gained: the gate is the level, FR-060, FR-061, FR-089), no choices made, and one `PaladinLevelOneGained(characterId, entry)` event. It MUST reject a request when a `PaladinState` already exists (`alreadyAPaladin`). `PaladinLevelOneGained` is emitted only on the `multiclass` path, where the host (or its orchestrating use case) calls `start` after the level-advanced event; on the `firstLevel` path the character is created through `PaladinCreationChoices.seed` (FR-122), which emits no event, and `CharacterCreated` is the record. The generic first-level and multiclass grants are character-creation.md's and level-advancement-and-multiclassing.md's. | SRD 5.2.1, "Becoming a Paladin ...", p. 53; design decision D1 for the full pool | happy |
| FR-112 | `ApplySmiteOfProtection` MUST take the optional `PaladinState`, optional `SubclassId`, the host fact `divineSmiteCast`, the Charisma `AbilityModifier` and the host fact `hasIncapacitatedCondition`, and return `ApplySmiteOfProtectionApplied` carrying `SmiteOfProtectionEffect` (`CoverGrant.half`, until the start of the Paladin's next turn) and one `SmiteOfProtectionTriggered` event per FR-084. With `divineSmiteCast` false, or any other FR-084 condition unmet, it MUST return `ApplySmiteOfProtectionNoEffect`, a returned success outcome with no event (FR-115, FR-117). | SRD 5.2.1, "Level 15: Smite of Protection", p. 57 | unhappy (boundary: levels 14 and 15; flag false) |
| FR-114 | Behaviour that loads no content MUST NOT be a use case: `PaladinEffects.end`, `PaladinRests.apply`, `PaladinAdvancement.resourcesFor` and `PaladinFirstLevel.start` are synchronous pure domain functions with no telemetry (FR-108; shared-types-and-ownership.md FR-023). Use cases that load the Paladin class entry (or feats, weapons, spells) MUST be context-aware (section 6). | `doc/rules/architecture.md` §4 | happy |
| FR-115 | Each use case MUST record exactly one span, named per section 7, with the attributes `srd5e.class` (`paladin`), `srd5e.outcome` (`success` or `failure`, shared-types-and-ownership.md FR-041) and `srd5e.feature` (the feature, or `choose_option` or `calculate_features` for the use cases that are not class features); a failure additionally sets `srd5e.character.rejection` to the reason, or to `content_missing`, `class_data_invalid` or `not_a_paladin`. The values of `srd5e.outcome` are exactly: `success` for an accepted use, a calculation, and `ApplySmiteOfProtectionNoEffect`; `failure` for a rejection, `...NotAPaladin`, `...ContentMissing` and `...ClassDataInvalid`. `srd5e.character.rejection` has no `no_effect` value. | `doc/rules/observability.md` §9 | happy |
| FR-116 | Each use case MUST record its duration through the `Meter` timing operation into the shared histogram `srd5e.character.feature_used.duration`, which this spec does not declare (owned by `character-aggregate.md`, shared-types-and-ownership.md FR-063). This spec only emits it, with the shared attributes `srd5e.class` (`paladin`) and `srd5e.feature` (the feature's snake_case name, or the operation name for a use case that is not a class feature). The outcome and rejection are on the span (FR-115), not on the histogram. | `doc/rules/observability.md` §4 | happy |
| FR-117 | Every call of a use case that is a class feature (Lay On Hands, Divine Sense, Abjure Foes, Sacred Weapon, Paladin's Smite, Faithful Steed, Holy Nimbus, Smite of Protection) MUST increment the shared counter `srd5e.character.feature_used` once, accepted or not, with `srd5e.class` = `paladin`, `srd5e.feature` = the feature's snake_case name, `srd5e.outcome` and, on a failure, `srd5e.character.rejection` (character-aggregate.md FR-118; shared-types-and-ownership.md FR-042). There are no per-feature or per-option metric names; the Hit Points spent and the option are in the result, not in a metric. `ApplySmiteOfProtectionNoEffect` increments it with `success` and no rejection. A returned `*srderr.PortError` increments it once with `failure`, no `error.type` and no rejection, and `.errors` once (FR-110). | `doc/rules/observability.md` §9 | happy |
| FR-118 | `<UseCase>NotAPaladin` and every `<UseCase>Rejected` MUST be a returned failure that is a normal rule outcome: `srd5e.outcome` = `failure`, no error span status, no `srd5e.character.feature_used.errors` increment and no `error.type` (errors-and-results.md FR-043), counted once in `feature_used` (FR-117). `<UseCase>ContentMissing` and `<UseCase>ClassDataInvalid` are returned failures that also set error span status (FR-110), still with no `error.type`, no counter increment and no log. `ApplySmiteOfProtectionNoEffect` is `srd5e.outcome` = `success`, no rejection attribute, one `feature_used` increment. `srd5e.outcome` takes no value other than `success` and `failure` (shared-types-and-ownership.md FR-041). | `doc/rules/observability.md` §7 | unhappy (telemetry on rejection) |
| FR-119 | Signal names and attribute keys MUST be constants in the character telemetry constants file, and a test MUST pin each name and key to the string of section 7. | `doc/rules/observability.md` §9 | happy |
| FR-120 | Lay On Hands removals and aura immunities MUST use `ConditionKind` from `content` (shared-types-and-ownership.md FR-005); creature types use `CreatureType` and damage types `DamageType`, also from `content`. There is no `PaladinCondition` enum and no mapping by the host. A `ConditionKind` outside Poisoned and the six of FR-051 is `conditionNotRemovable`. | shared-types-and-ownership.md FR-005, FR-013; SRD 5.2.1, "Level 1: Lay On Hands", p. 54 | unhappy (invalid input: any other condition) |
| FR-121 | `PaladinState` MUST implement `FeatureState` and be stored in the character aggregate's `FeatureStates` under a typed `FeatureStateKey[PaladinState]` named `paladinStateKey`, both defined in this spec. The aggregate has no Paladin-specific field. | shared-types-and-ownership.md FR-022 | happy |
| FR-122 | The Paladin's creation choices travel in `CreateCharacter` as `PaladinCreationChoices`, an implementation of `ClassCreationChoices` defined in this spec. It carries only what is Paladin-specific: the `CharacterId` of the character being created (the events of FR-124 carry it, and `ClassCreationChoices.seed` receives only a `FeatureStates`; the host passes the same id as in `CreateCharacterCommand`, a mismatch being a host programmer error) and the `WeaponMasteryChoice` (type owned by weapon-properties-and-mastery.md). Skills and Starting Equipment are generic choices in character-creation.md. `PaladinCreationChoices.seed(FeatureStates)` is pure and emits no event (character-aggregate.md FR-094). It MUST return the given `FeatureStates` with two entries added: the `PaladinState` of `PaladinFirstLevel.start(characterId, ClassEntry.firstLevel)` under `paladinStateKey` (its event is discarded, FR-111), and the `WeaponMasteryChoice` under the key weapon-properties-and-mastery.md defines. All other entries are kept and the input is unchanged. A `FeatureStates` that already holds a `PaladinState` is a programmer error (panic): `CreateCharacter` starts from empty feature states. | shared-types-and-ownership.md FR-024 | happy |
| FR-123 | Every use case MUST return its own closed result named `<UseCase>Result`, with variants prefixed by the use case (the model is `CreateCharacterContentMissing`): `UseLayOnHandsApplied`, `UseLayOnHandsRejected`, `UseLayOnHandsContentMissing`, `UseLayOnHandsClassDataInvalid` and `UseLayOnHandsNotAPaladin`, and likewise for each use case with the exact names in section 6. An accepted variant carries the new `PaladinState` as `state` and the `events`. No generic variant name (`Applied`, `Rejected`, `ContentMissing`, `NoEffect`) is exported. | shared-types-and-ownership.md FR-017, FR-018 | unhappy (name collision) |
| FR-124 | Paladin events (`LayOnHandsUsed`, `PaladinChannelDivinityUsed`, `FreeCastSpent`, `HolyNimbusActivated`, `HolyNimbusRestored`, `SmiteOfProtectionTriggered`, `PaladinEffectEnded`, `PaladinChoiceMade`, `PaladinLevelOneGained`) MUST be past-tense value objects that implement `CharacterEvent` (character-aggregate.md FR-085, an open interface that implements `DomainEvent`) and carry the `characterId`, as the other class specs do; they are outside the closed `CharacterRecordEvent`. The id comes from the `PaladinState` they were produced from (it holds the `CharacterId`, set once by `PaladinFirstLevel.start` and never changed, as spellcasting.md FR-127 does). Each is defined only here. Events defined by other specs (`SubclassChosen`, `RestApplied`, the level-advanced event) are referenced, not redefined. Every accepted use or choice carries at least one event; a rejection carries none, and neither does a feature calculation. | shared-types-and-ownership.md FR-019, FR-020; character-aggregate.md FR-085 | happy |
| FR-125 | Level gates and derived lists MUST come from the loaded definition, not constants: the level at which a feature becomes available is the `FeatureEntry.level` of the named feature in the `ClassDefinition` (or the chosen `SubclassDefinition`) held by `PaladinProgression` (FR-014), through `PaladinProgression.levelOf(featureName)`. This covers FR-051, FR-060, FR-061, FR-065, FR-069, FR-077, FR-078, FR-082 to FR-086 and FR-089; `alwaysPrepared` (FR-058) and `featEntitlements` (FR-088) read their own facts as stated there. Only the effect values that are formulas or numbers in the SRD feature text stay constants: 5 times Paladin level, 5 Hit Points per condition, the larger of 1 and the Charisma modifier, the 10-foot and 30-foot radii, 60 feet, 10 minutes, 1 minute, 1d8. | shared-types-and-ownership.md FR-025; content-classes.md FR-049, FR-058 | unhappy (rule interaction: a definition whose Extra Attack is at level 6 gives 2 attacks from 6, not 5) |

### 4.12 Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (Paladin class entry) |
| FR-002 | Withdrawn: moved to content-classes.md (Paladin class entry) |
| FR-003 | Withdrawn: moved to content-classes.md (Paladin class entry) |
| FR-004 | Withdrawn: moved to content-classes.md (Paladin class entry) |
| FR-005 | Withdrawn: moved to content-classes.md (Paladin class entry) |
| FR-006 | Withdrawn: moved to content-classes.md (Paladin class entry, Starting Equipment A) |
| FR-007 | Withdrawn: moved to content-classes.md (Paladin class entry, Starting Equipment B) |
| FR-008 | Withdrawn: moved to content-classes.md (FR-037) |
| FR-009 | Withdrawn: moved to content-classes.md (Paladin Features table, p. 53, transcribed there) |
| FR-010 | Withdrawn: moved to content-classes.md (Paladin Features table) |
| FR-011 | Withdrawn: moved to content-classes.md (FR-033) |
| FR-012 | Withdrawn: moved to content-classes.md (Paladin Features table) |
| FR-013 | Withdrawn: moved to content-classes.md (FR-033) |
| FR-015 | Withdrawn: moved to content-classes.md (Paladin feature entries) |
| FR-016 | Withdrawn: moved to content-classes.md (feature text fidelity); that spec has no numeric-term check, so the checklist stays here as a fixture: 5 times Paladin level, 5 Hit Points, 60 feet, 10 minutes, 1 minute, 1d8, 10-foot and 30-foot Emanations, 20-foot Bright Light and 20-foot Dim Light, level 5 spell slot, the Level 11 additional use, minimum of one creature, minimum bonus +1 (SRD pp. 53-57) |
| FR-017 | Withdrawn: moved to content-classes.md (feature entries) |
| FR-018 | Withdrawn: the content model has no non-binding recommendation field, so the SRD recommendations are intentionally not modelled; fixture for any owner: Heroism and Searing Smite (Prepared Spells), Guidance and Sacred Flame (Blessed Warrior), Boon of Truesight (Epic Boon); a choice ignoring one is accepted (SRD pp. 54-55) |
| FR-019 | Withdrawn: moved to content-spells.md (FR-023; spell list pp. 55-56 transcribed there) |
| FR-020 | Withdrawn: moved to content-spells.md (FR-024, FR-025) |
| FR-021 | Withdrawn: moved to content-spells.md (FR-023 to FR-025) |
| FR-022 | Withdrawn: moved to content-spells.md (FR-024, FR-025) |
| FR-024 | Withdrawn: moved to content-classes.md (Oath of Devotion subclass entry) |
| FR-025 | Withdrawn: moved to content-classes.md (Oath of Devotion subclass entry) |
| FR-026 | Withdrawn: moved to content-classes.md (Oath of Devotion Spells, FR-058) |
| FR-027 | Withdrawn: moved to content-classes.md (Oath of Devotion subclass entry); the model has no tenets field, so the tenets are intentionally not modelled; fixture, in printed order: "Let your word be your promise." / "Protect the weak and never fear to act." / "Let your honorable deeds be an example." (SRD p. 56) |
| FR-028 | Withdrawn: moved to level-advancement-and-multiclassing.md (which features a level grants) |
| FR-029 | Withdrawn: moved to character-creation.md (first-level grants) |
| FR-030 | Withdrawn: moved to level-advancement-and-multiclassing.md (multiclass grants) |
| FR-031 | Withdrawn: moved to level-advancement-and-multiclassing.md (multiclass prerequisite) |
| FR-032 | Withdrawn: moved to character-creation.md (skill choice) |
| FR-033 | Withdrawn: moved to weapon-properties-and-mastery.md (WeaponMasteryChoice) |
| FR-034 | Withdrawn: moved to weapon-properties-and-mastery.md (WeaponMasteryChoice change rule) |
| FR-035 | Withdrawn: moved to weapon-properties-and-mastery.md (canUseMastery) |
| FR-036 | Withdrawn: moved to character-creation.md (Starting Equipment choice) |
| FR-037 | Withdrawn: moved to level-advancement-and-multiclassing.md (SubclassChosen, SubclassRejected) |
| FR-097 | Withdrawn: moved to level-advancement-and-multiclassing.md (features gained at a level) |
| FR-099 | Withdrawn: moved to level-advancement-and-multiclassing.md (AdvanceRejected) |
| FR-100 | Withdrawn: moved to level-advancement-and-multiclassing.md (the level-advanced event) |
| FR-113 | Withdrawn: moved to content (ConditionKind, shared-types-and-ownership.md FR-005; see FR-120) |

## 5. Model

Domain (`character` package, pure domain files; pure and synchronous):

- **`PaladinProgression`**: built from the decoded `ClassDefinition` by
  the `New` constructor (returns an error) (FR-014); gives the table row of a level, the Channel Divinity
  maximum and `levelOf` (FR-125). Numbers come from content, except SRD formulas;
  grants come from `AlwaysPreparedSpells` facts and `featGrantsAt`.
- **`PaladinState`**: immutable `FeatureState` (FR-121) under `paladinStateKey`.
  Fields: `CharacterId`, Paladin level, optional `PaladinFightingStyle`,
  `LayOnHandsPool`, `PaladinChannelDivinityUses`, `FreeCasts` (a `LongRestUse`
  per `FreeCastFeature`), `holyNimbus` (a `LongRestUse`), `ActiveEffects`
  (Divine Sense, Sacred Weapon with its `ItemId`, Holy Nimbus), feat selections
  used. "Not yet gained" is the level gate (FR-125), not a third `LongRestUse`
  state. The `SubclassId` and Weapon Mastery choice are held by their owners.
- **Value objects**: `LayOnHandsPool` (`maximum`, `remaining`, `spend`),
  `PaladinChannelDivinityUses` (`spend`, `shortRest`, `longRest`,
  `raiseMaximum`), `PaladinFightingStyle` (closed:
  `PaladinFightingStyleFeat(FightingStyleChoice)` | `BlessedWarrior(cantrips)`),
  `PaladinCreationChoices` (FR-122), and the effect data `DivineSenseEffect`,
  `AbjureFoesEffect`, `SacredWeaponEffect`, `HolyNimbusEffect`, `AuraEffects`,
  `SmiteOfProtectionEffect`, `RadiantStrikes`. Distances are integers in feet,
  durations integers in minutes, damage a `dice.Expression` plus a `DamageType`,
  bonuses a `Modifier`, a save DC a `SpellSaveDc`. Paladin-local enums:
  `DetectionKind` {location, creatureType, consecratedOrDesecrated},
  `PaladinActionCost` {bonusAction, magicAction; Lay On Hands, Divine Sense, Holy
  Nimbus: bonus action; Abjure Foes: magic action}, `CoverGrant`,
  `PaladinAttackKind`, `ClassEntry` {firstLevel, multiclass}, `LongRestUse`
  {available, spent}, `FreeCastFeature` {paladinsSmite, faithfulSteed}. Light is
  `LightLevel` from `content` (content-schema-and-decoder.md FR-133).

Invariants: pool bounds (`LayOnHandsPool` constructor, FR-045); uses within
`0..maximum` (`PaladinChannelDivinityUses` constructor); a Fighting Style only at
level 2 or higher; a free cast or Holy Nimbus use of a feature not yet gained at
the state's level is `LongRestUse.available` (what lets FR-098 make a newly
gained use usable at once); at most one active Sacred Weapon. `PaladinState` and
`ActiveEffects` `MustX` constructors panic with an error value for violations.

## 6. Interface

Use cases are in the `character` package (use-case files), one class each, single
`call`; all load the class entry (and feats, spells) through the context-aware
`ContentSource` port and are context-aware (`doc/rules/architecture.md` §4). The
pure functions that need no content (FR-114) are not use cases and have no
telemetry; the host applies them after `RestApplied` or the level-advanced event
(section 4.9). None uses a `dice.Roller`.

| Use case | Command carries | Other variants (`<UseCase>Result`) | Requirements |
|---|---|---|---|
| `ChoosePaladinOption` | optional `PaladinState`, a closed `PaladinChoice` (`ChoosePaladinFightingStyle`, `ChooseBlessedWarriorCantrips`, `ReplaceBlessedWarriorCantrip`), optional `SubclassId`, change trigger | `ChoosePaladinOptionAccepted(state, events)`, `ChoosePaladinOptionRejected(reasons)` | FR-038 to FR-042, FR-101 to FR-106 |
| `UseLayOnHands` | optional `PaladinState`, Hit Points to restore, `ConditionKind`s to remove | `UseLayOnHandsApplied(state, hitPoints, conditions, actionCost, events)`, `UseLayOnHandsRejected(reasons)` | FR-043 to FR-056 |
| `UsePaladinChannelDivinity` | optional `PaladinState`, `PaladinChannelDivinityOption`, optional `SubclassId`, Charisma modifier, `SpellSaveDc`, host facts `takingAttackAction`, `holdingMeleeWeapon` | `UsePaladinChannelDivinityApplied(state, effect, events)`, `UsePaladinChannelDivinityRejected(reasons)` | FR-065 to FR-075 |
| `UseFreeCast` | optional `PaladinState`, `FreeCastFeature` | `UseFreeCastApplied(state, spell, events)`, `UseFreeCastRejected(reasons)` | FR-060 to FR-064 |
| `UseHolyNimbus` | optional `PaladinState`, optional `SubclassId`, action (`activate` or `restoreWithSlot`), Charisma modifier, Proficiency Bonus, `hasIncapacitatedCondition`, for `restoreWithSlot` the `SpellcastingState` and the `SpellSlotPool` | `UseHolyNimbusApplied(state, effect, spellcastingState, events)` (`effect` only for `activate`, `spellcastingState` only for `restoreWithSlot`), `UseHolyNimbusRejected(reasons)` | FR-089 to FR-095 |
| `ApplySmiteOfProtection` | optional `PaladinState`, optional `SubclassId`, `divineSmiteCast`, Charisma modifier, `hasIncapacitatedCondition` | `ApplySmiteOfProtectionApplied(effect, events)`, `ApplySmiteOfProtectionNoEffect` | FR-084, FR-112 |
| `CalculatePaladinFeatures` | optional `PaladinState`, optional `SubclassId`, Charisma modifier, Proficiency Bonus, `hasIncapacitatedCondition` | `CalculatePaladinFeaturesSummary(auras, attacksPerAttackAction, radiantStrikes, alwaysPrepared, featGrants, pendingChoices)` (no `Rejected`) | FR-042, FR-058, FR-077 to FR-083, FR-085 to FR-088, FR-095, FR-125 |

Notes:

- `UseHolyNimbus` `restoreWithSlot` uses the domain operation behind
  spellcasting.md FR-109 `ExpendSpellSlot`, not a spellcasting use case, so
  errors count once (FR-092). Free casts only track availability; the host casts
  the returned spell with `CastMode.featureGranted` (FR-060).
- The package API the `character` package exports the use cases, commands,
  results, state and key, choices, value objects (including `LongRestUse`,
  `FreeCastFeature`, `ClassEntry`, `PaladinChannelDivinityOption`,
  `PaladinActionCost`, `PaladinAttackKind`, `DetectionKind`, `CoverGrant`), effect
  data, pure domain functions, events and rejection enums; the module's public packages
  re-exports them.

## 7. telemetry.Telemetry

The shared signals `srd5e.character.feature_used`, `.duration`, `.errors` and the
child span `srd5e.character.load_content` are declared by `character-aggregate.md`
(FR-118; shared-types-and-ownership.md FR-042, FR-063); this spec only emits them
(FR-110, FR-115 to FR-118). The domain is silent and nothing logs (observability.md
§7). Names are constants (FR-119) and feed `doc/observability/signals.md`.

- **What can go wrong, who records what:** FR-110 and FR-118.
- **Attributes:** `srd5e.class` (`paladin`), `srd5e.feature` (`lay_on_hands`,
  `divine_sense`, `abjure_foes`, `sacred_weapon`, `paladins_smite`,
  `faithful_steed`, `holy_nimbus`, `smite_of_protection`; `choose_option` and
  `calculate_features` for the two use cases that are not class features),
  `srd5e.outcome`, `srd5e.character.rejection` (failure only; snake_case, so
  `exceedsPool` is `exceeds_pool`). No ids or values go on signals.
- **Monitor:** any non-zero returned-error rate on `.errors`; a rise in
  `content_missing` or `class_data_invalid` rejections (a content deployment
  fault); `exceeds_pool` failures of `lay_on_hands` (a host not showing the
  remaining pool); a p99 regression of the shared duration histogram after a
  content-source change.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.use_lay_on_hands`, `use_channel_divinity`, `use_free_cast`, `use_holy_nimbus`, `apply_smite_of_protection` | span | `srd5e.class`, `srd5e.feature` (for `use_channel_divinity` the option; for `use_free_cast` `paladins_smite` or `faithful_steed`), `srd5e.outcome` (`success` also for Smite of Protection no effect), `srd5e.character.rejection` (failure only) | One use of the feature, with the shared `load_content` child span | error status; slower than baseline |
| `srd5e.character.choose_paladin_option`, `calculate_paladin_features` | span | `srd5e.class`, `srd5e.feature` (`choose_option` or `calculate_features`), `srd5e.outcome`, `srd5e.character.rejection` (failure only) | One Paladin choice or summary calculation | error status; slower than baseline |

## 8. Test plan

Levels: unit (`character_test` package (`*_test.go`)), application (`character_test` use-case tests,
`telemetrytest.Recorder`, `FakeContentSource`), scenario (`scenarios/`,
public API only), architecture. Withdrawn-id data checks belong to their owners.

| Id | Level | Notes |
|---|---|---|
| FR-014 | unit, application | the `New` constructor (returns an error) over `ClassDefinition` fixtures with several defects at once (wrong id, no Channel Divinity column, missing gating feature entries); all `PaladinDataProblem` values listed; the use case wraps them in `ContentDecodeFailed` as `<UseCase>ClassDataInvalid` |
| FR-023 | unit | An always-prepared spell not on the Paladin list (Beacon of Hope at level 9 with Oath of Devotion) is returned by `alwaysPrepared` and passed on unfiltered |
| FR-038 to FR-042 | unit, application | Fighting Style and Blessed Warrior validation, reasons and pending choices; `GrantedCantrip(spellId, SpellcastingClass.paladin, cleric)` per cantrip; replacement rejects only `notBlessedWarrior` and `duplicate` here; `ContentMissing` for unknown ids |
| FR-043 to FR-056 | unit, application | Tables H and U per row (levels 0 and 21 give panic); starts full; constructor panic; Long and Short Rest; `LayOnHandsRules.use` returns the new pool, Hit Points, conditions, Bonus Action cost and `LayOnHandsUsed` with its `characterId`, and `UseLayOnHandsApplied` carries them |
| FR-057 to FR-059 | unit | Table A per row with Oath grants inside a `SubclassDefinition`, plus a modified definition (Divine Smite at level 3) proving the grants are read from the definition; set passed to spellcasting.md unchanged; Paladin row agrees with spellcasting.md |
| FR-060 to FR-064 | unit | Level gating; `available` on gaining at 2 and 5 and usable at once; spent twice; independence; rests |
| FR-065 to FR-070 | unit, application | Maximum at levels 2, 3, 10, 11; start full; Short Rest at 0, 1, maximum; option gating by level and subclass; DC taken from the command, never recomputed |
| FR-071 to FR-076 | unit | Effect data; Tables B and W per row; replacement while active; end reasons through `PaladinEffects.end` |
| FR-077 to FR-088 | unit | Tables AU, SP, X and E per row; Radiant Strikes data 1d8 Radiant; `attacksPerAttackAction` at levels 4 and 5; a definition whose level 2 row has a Fighting Style `GrantsFeat` gives no entitlement at 2 and total 5 by 20 (`forWeapon(WeaponType.melee)` for a thrown Javelin); aura choice; Half Cover data; a definition with Extra Attack at level 6 and one with an extra feat grant at level 6 show gates and lists follow the definition (FR-125) |
| FR-089 to FR-095 | unit, application | Table N per row; activation sets `LongRestUse.spent` and returns `HolyNimbusActivated`; restore through the slot operation for the `spellcasting` pool and for a level 5 slot held only in `pactMagic`; `noLevel5SlotAvailable` for each pool empty and for the wrong pool chosen; `notExpended` while `available`; Channel Divinity unchanged |
| FR-096, FR-098 | unit | Table R through `PaladinRests.apply`; Table G (Resource changes) through `PaladinAdvancement.resourcesFor`: Paladin's Smite `available` at level 2 without a Long Rest, Faithful Steed at 5, Holy Nimbus at 20; no other resource refilled; a spent use stays spent when an unrelated level is gained |
| FR-101 to FR-104 | application | Null state for each use case of FR-101; `<UseCase>ContentMissing` wins over a rejection reason, built from `ContentNotFound.missing`; failing double raising its typed `*srderr.PortError`; malformed entry mapped to `ContentProblem` values in `ContentDecodeFailed` |
| FR-105 to FR-108 | unit, application | Input unchanged and no events, at domain level and through each use case; fixed rejection order; `a defensive copy` on mutating returned lists; same inputs same output |
| FR-109, FR-110 | application | Same result with no-op and recording `telemetrytest.Recorder`; a returned error gives one `feature_used.errors` increment with `error.type`, one `feature_used` with `failure`, error span status, no log; `ContentMissing` and `ClassDataInvalid` give `failure`, error span status, no `error.type`, no counter, no log; `Rejected` and `NotAPaladin` give `failure`, no error span status, no `error.type` |
| FR-111, FR-112, FR-114 | unit, application | `PaladinFirstLevel.start` (pool 5, uses `available`, no choices, `alreadyAPaladin`, event carries `characterId` and `entry`); Table SP per row, including `ApplySmiteOfProtectionNoEffect` with no event; no-content behaviour callable without a `ContentSource` |
| FR-115 to FR-119 | application | `telemetrytest.Recorder`: one span per call with pinned attributes; `feature_used` once per call with `srd5e.class`, `srd5e.feature`, `srd5e.outcome` and, on a failure, `srd5e.character.rejection`; the shared duration histogram (`telemetrytest.Meter.timedDuration`) once per call, carrying only `srd5e.class` and `srd5e.feature`; `ApplySmiteOfProtectionNoEffect` is `success` with no rejection attribute; rejections and `NotAPaladin` are `failure` with no error span status and no `error.type`; a returned error gives one `feature_used` (`failure`, no `error.type`) and one `.errors`; constants pinned; no log in any path |
| FR-120 to FR-122 | unit, architecture | `ConditionKind` is the only condition type used; `PaladinState` round-trips through `FeatureStates` by `paladinStateKey`; `PaladinCreationChoices.seed` on empty `FeatureStates` writes the level 1 `PaladinState` (pool 5, no event) and the mastery choice under the key weapon-properties-and-mastery.md defines; keeps other entries; panics with an error value when a `PaladinState` is already held; input unchanged |
| FR-123 to FR-125 | architecture, unit | Package API scan: every variant prefixed by its use case, no generic name exported, none of `UsePaladinChannelDivinity*`, `PaladinChannelDivinityUsed`, `PaladinChannelDivinityUses`, `ChoosePaladinFightingStyle`, `PaladinActionCost`, `PaladinAttackKind` collides with another class spec; events implement `CharacterEvent` and `DomainEvent`, carry `characterId`, defined once. FR-125: definitions with moved feature levels move the gates |
| Whole slice | scenario | Through the public packages (`github.com/Whealinghold/srd5e-go/...`): a level 1 Paladin is created through `PaladinCreationChoices.seed` (or `PaladinFirstLevel.start`), advances to 3 (the host applies `PaladinAdvancement.resourcesFor` after each level-advanced event) and picks the Oath, reaches 6 with the aura, uses Lay On Hands and Channel Divinity, then takes a Short and a Long Rest (the host applies `PaladinRests.apply` after each `RestApplied`), proving the specs to the left of `class-*` need no Paladin call |

Mutation testing: changed lines in the `character` package (pure domain files) at a score of at
least 80 (`doc/rules/testing.md` §8); application lines SHOULD be.

## 9. Open questions and decisions

No question is open.

Settled, one line each:

- Q7 (free cast of Divine Smite or Find Steed): casting-and-concentration.md FR-121 already provides `CastMode.featureGranted` for casting without a slot (SRD p. 104); `UseFreeCast` tracks availability and returns the spell id (FR-060, FR-061).
- Q14 (`seed` receives no `CharacterId`): `PaladinCreationChoices` takes the id at construction (FR-122); package design, no other spec needs to change.
- D1 (what a level-up does to the remaining Lay On Hands or Channel Divinity amount, and to a newly gained use): interpretation. The SRD gives maximums only and says Lay On Hands "replenishes when you finish a Long Rest" (p. 53); raising `remaining` by the gain and starting a new feature `available` is the reading in which a level-up neither wastes the gain nor grants a free refill (FR-044, FR-066, FR-098).
- D4 (Holy Nimbus while Incapacitated): interpretation. Holy Nimbus imbues the Aura of Protection, which "is inactive while you have the Incapacitated condition" (p. 55), so its aura-based effects report inactive (FR-095).
- Q15 (shared action-cost type): keep `PaladinActionCost`, class-prefixed like Ranger and Druid (shared-types-and-ownership.md FR-021, unique names); a shared type is a later refactor, not a blocker.
- Q4, Q5, Q6, Q9, Q10 by shared-types-and-ownership.md FR-022, FR-005, FR-063 and spellcasting.md FR-097, FR-098; Q8 by SRD p. 90 (FR-086); Q13 (section 4.9); D2 by FR-048; D3 moved to level-advancement-and-multiclassing.md; Q11, Q12 by FR-076, FR-073.

## 10. Changelog impact

`Added`: Paladin behaviour (state, use cases, pure functions, effect data) and the
signals in section 7. Not **BREAKING**. The bundled Paladin data and its
`NOTICE.md` coverage are recorded by `content-classes.md`.
