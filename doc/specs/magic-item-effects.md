# Magic item effects: numeric bonuses, set scores, charged spell items, containers

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `magic-item-rules.md`, `content-magic-items.md`, `content-schema-and-decoder.md`, `derived-stats.md`, `spellcasting.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `character-aggregate.md`, `errors-and-results.md`, `observability-ports.md` |

## 1. Purpose and scope

`magic-item-rules.md` owns the general rules of magic items (Attunement, charges, wearing) and leaves what a particular item does as verbatim content text (`content-magic-items.md`, "Behaviour of items"). This spec models the **numeric effects** of the SRD items that are plain numbers, as typed values the host feeds into `derived-stats.md` and `combat`:

1. **Bonus items** (section 4.1): +N Armor, Shield, Weapon and Ammunition, Wand of the War Mage, Ring and Cloak of Protection and their siblings, as typed `MagicItemEffect`s with a condition ("while worn", "while holding").
2. **Ability-score items** (4.2): set-to-score (Belt of Giant Strength, Gauntlets of Ogre Power, Headband of Intellect, Amulet of Health, Potion of Giant Strength) and capped raises (Belt of Dwarvenkind, Might of Giants).
3. **Potions** (4.3): the four Potion of Healing grades and Potion of Resistance.
4. **Charged spell items** (4.4, 4.5): the charge-to-level pattern, charge cost tables, the last-charge roll, Staff of the Magi absorption and Retributive Strike numbers.
5. **Ring of Spell Storing** (4.6).
6. **Extradimensional containers** (4.7): Bag of Holding, Handy Haversack, Portable Hole capacity, overload and nesting destruction. All three fit under the size limit, so none is split out (Q1).

The effect table (FR-013) is a `character` constant table, not a content kind (Q2). No `combat` type is used: amounts, bonuses and resistances are returned as values and the host applies them (shared-types-and-ownership.md FR-013).

**Out of scope:**

- General rules (Attunement, charges, `SpellFromItem`, crafting, sentience): `magic-item-rules.md`; this spec reuses them and restates none.
- Applying healing, Temporary Hit Points, Resistance or a Strength change to a creature, and expiry of "1 hour": `hit-points-and-damage.md` and the host. Casting a spell (slots, Concentration, targets): `casting-and-concentration.md` and the host (`spellcasting.md` excludes casting from items).
- Items with an effect that is not a flat number or a table in this spec: Robe of the Archmagi (base AC 15), Arrow-Catching Shield (AC against ranged attacks), Bracers of Archery and the skill-check bonuses of Gloves of Swimming and Climbing, Gloves of Thievery (+5 to checks), Manuals and Tomes (permanent score raises), Ioun Stones, Dragon Scale Mail and Weapon riders (extra damage). They stay item text; a row is added when a spec needs one. Anything not found in the PDF is not invented.
- Display text, persistence, the clock ("1 hour", "next dawn" arrive as host facts), the 50 percent plane-travel roll of Retributive Strike and the GM's choice among table results.

## 2. Source

SRD 5.2.1 printed page numbers equal PDF page numbers.

- "Armor", "Potions", "Rings" (Resistance not here), p. 204; "Magic Item Rules" sidebar, "Weapons" (Ammunition), "Wands", p. 205; "Charges", "Spells Cast from Items", "The Next Dawn", p. 206.
- "Magic Items A-Z": Amulet of Health, "Ammunition, +1, +2, or +3", p. 209; "Armor, +1, +2, or +3", p. 210; "Bag of Holding", p. 212; "Belt of Dwarvenkind", "Belt of Giant Strength", p. 213; "Bracers of Defense", p. 214; "Chime of Opening", p. 215; "Cloak of Protection", p. 216; "Dwarven Plate", p. 219; "Elven Chain", p. 220; "Gauntlets of Ogre Power", "Gem of Brightness", "Glamoured Studded Leather", p. 223; "Hammer of Thunderbolts" (Giant's Bane, Might of Giants), "Handy Haversack", p. 224; "Headband of Intellect", p. 225; "Portable Hole", pp. 234-235; "Potion of Giant Strength", p. 235; "Potions of Healing", "Potion of Resistance", p. 236; "Ring of Protection", "Ring of Resistance", p. 238; "Ring of Spell Storing", "Ring of Three Wishes", p. 239; "Robe of Stars", p. 240; "Rod of Resurrection", p. 242; "Scarab of Protection", "Shield, +1, +2, or +3", p. 243; "Staff of Charming", "Staff of Fire", "Staff of Frost", "Staff of Healing", "Staff of Power", p. 245; "Staff of Power", "Staff of Striking", "Staff of Swarming Insects", "Staff of the Magi", p. 246; "Staff of the Magi", "Staff of the Woodlands", p. 247; wands, pp. 250-251; "Wand of the War Mage, +1, +2, or +3", p. 251; "Weapon, +1, +2, or +3", p. 253.
- "Equipment", "Potion of Healing", p. 99; "Magic Items" (Attunement), p. 102; "Rules Glossary", "Armor Class", p. 177; "Round Down", p. 5.
- Package design: `doc/rules/architecture.md` sections 2 to 6, `doc/rules/observability.md` sections 2 to 9, `doc/rules/testing.md` section 4, `shared-types-and-ownership.md` FR-013, FR-017, FR-019, FR-044.

## 3. Vocabulary

Types defined here (`character`). Terms use SRD 5.2.1 wording; no type is called `Race`, `Dc` or `MagicItemRarity`.

| SRD term | Type | Notes |
|---|---|---|
| bonus to AC, saving throws, attack and damage rolls, spell attack rolls | `MagicItemEffect` (closed: `ArmorClassBonus`, `SavingThrowBonus`, `AttackAndDamageBonus`, `SpellAttackRollBonus`, `IgnoresHalfCoverOnSpellAttacks`, `SetAbilityScore`, `RaiseAbilityScore`, `BoostBestowedScore`), `ItemEffectCondition` (`WhileWorn`, `WhileHeld`, `WhileOnPerson`, `WhileWornWithoutArmorOrShield`) | FR-003 to FR-013. Each bonus holds a `Modifier` (kernel). `SpellAttackRollBonus` is not `spellcasting.md`'s `SpellAttackBonus`. |
| "bonus is determined by its rarity" | `EnhancementFamily` (`armor`, `shield`, `weapon`, `ammunition`, `warMageWand`), `EnhancementBonusLookup` (`EnhancementBonusFound`, `EnhancementBonusNotListed`) | FR-001, FR-002. |
| which effects are on | `ItemEffectContext(access, usage, wearsArmor, usesShield)`, `ActiveItemEffects`, `MagicItemBonus(modifier, instanceID)` | FR-009 to FR-012. `access` is `MagicalPropertiesAccessLevel` (`magic-item-rules.md`). |
| the effects of an item | `MagicItemEffectTable`, `MagicItemEffectLookup` (`MagicItemEffectsFound`, `MagicItemEffectsNotModelled`) | FR-013. |
| ammunition after a hit | `AmmunitionState` (`MagicalAmmunition`, `NonmagicalAmmunition`) | FR-006. |
| score granted, "increases by", cap | `EffectiveAbilityScores`, `AbilityScoreSource` (`BaseScore`, `SetByItem(instanceID)`, `RaisedByItem(instanceID)`), `GiantStrengthType` (`hill`, `frostOrStone`, `fire`, `cloud`, `storm`) | FR-014 to FR-021. |
| Potion of Healing grades | `PotionOfHealingGrade` (`standard`, `greater`, `superior`, `supreme`); the rolled amount is a plain `int` in `RollPotionOfHealingRolled` (`HitPointsRestored` is `hit-points-and-damage.md`'s and is not redefined) | FR-022 to FR-025. |
| Resistance to one type, 1d10 table | `ResistanceTypeTable`, `GrantedResistance(damageType, duration)` | FR-026, FR-027. `DamageType` is `content`'s. |
| spell cast by expending charges | `ChargedSpellOption` (closed: `FixedChargeCost`, `ChargesPerLevel`), `ItemSpellLevel` (`LowestLevel`, `NamedLevel`), `ItemSpellStats` (`UsersSpellStats`, `FixedSaveDC`), `ChargedSpellPlan`, `ChargedSpellCasting` | FR-028 to FR-036. |
| "roll 1d20. On a 1" | `LastChargeRule` (`NoLastChargeEffect`, `LastChargeAutomatic`, `LastChargeRolled(onOne, onTwenty)`), `LastChargeConsequence` (`none`, `destroyed`, `becomesNonmagical`, `retainsBonusLosesOtherProperties`), `LastChargeRegain` (`NoRegain`, `RegainsDice`) | FR-037 to FR-044. |
| Spell Absorption, Retributive Strike | `StaffSpellAbsorption`, `RetributiveStrike`, `RetributiveStrikeDamage` | FR-045 to FR-047. |
| Ring of Spell Storing | `SpellStorage(capacityLevels, stored)`, `StoredSpell(spellID, slotLevel, saveDC, attackBonus, ability)` | FR-048 to FR-056. |
| extradimensional space | `ExtradimensionalSpace` (`BagOfHoldingSpace`, `HandyHaversackSpace`, `PortableHoleSpace`, `SimilarSpace`), `SpaceLoad(ounces, cubicInches)`, `AstralGate`, `GateArea`, `AirSupply` | FR-057 to FR-065. |
| events | `ChargedItemEvent` (closed family, FR-068) | Implements `CharacterEvent`. |

Reused, not redefined: `ItemRarity`, `MagicItemID`, `MagicItemFacts`, `ChargeRecovery` (`content-magic-items.md`); `DamageType`, `SpellID`, `SpellLevel`, `ContentKind` (`content`); `ItemCharges`, `MagicItemInstanceID`, `MagicalPropertiesAccessLevel`, `SpellFromItem`, `ItemCasterLevel`, `MagicItemChargesSpent` (`magic-item-rules.md`); `Ability`, `AbilityScore`, `AbilityScores` (`ability-scores.md`); `Modifier` (`modifiers-and-advantage.md`); `dice.Expression`, `dice.Roller` (`dice-and-rolls.md`); `SpellSaveDC`, `SpellAttackBonus` (`spellcasting.md`); `CharacterID`, `CharacterEvent` (`character-aggregate.md`).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids.

### 4.1 Bonus items

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `EnhancementBonusLookup.Of(family, rarity)` MUST return `EnhancementBonusFound(Modifier(n))` per table "Enhancement bonuses": armor Rare +1, Very Rare +2, Legendary +3; shield, weapon, ammunition and warMageWand Uncommon +1, Rare +2, Very Rare +3. | SRD 5.2.1, "Armor, +1, +2, or +3", p. 210; "Shield, +1, +2, or +3", p. 243; "Weapon, +1, +2, or +3", p. 253; "Ammunition, +1, +2, or +3", p. 209; "Wand of the War Mage", p. 251 | happy |
| FR-002 | A rarity the header does not list for the family (Common, Artifact, Uncommon armor, Legendary weapon) MUST return `EnhancementBonusNotListed(family, rarity)`; it MUST NOT return an error and MUST NOT guess. | Same headers | unhappy (missing content) |
| FR-003 | A +N Armor MUST yield `ArmorClassBonus(+N, WhileWorn)`: a bonus on top of whatever base the wearer's AC uses, not a base. | SRD 5.2.1, "Armor, +1, +2, or +3", p. 210; "Armor Class", p. 177 | happy |
| FR-004 | A +N Shield MUST yield `ArmorClassBonus(+N, WhileHeld)` as its own term, "in addition to the Shield's normal bonus to AC"; it never replaces or alters the Shield's own bonus, which `derived-stats.md` already counts. | SRD 5.2.1, "Shield, +1, +2, or +3", p. 243 | unhappy (rule interaction) |
| FR-005 | A +N Weapon or +N Ammunition MUST yield one `AttackAndDamageBonus(+N)` that applies to both the attack rolls and the damage rolls made with that weapon or that piece of ammunition, never to other weapons. | SRD 5.2.1, "Weapon, +1, +2, or +3", p. 253; "Ammunition, +1, +2, or +3", p. 209 | happy |
| FR-006 | `MagicAmmunition.AfterAttack(state, hit)` MUST return `NonmagicalAmmunition` (with no bonus) when `hit` is true and the same `MagicalAmmunition` when false ("once it hits a target, the ammunition is no longer magical"); a `NonmagicalAmmunition` stays so for either input. | SRD 5.2.1, "Ammunition, +1, +2, or +3", p. 209 | unhappy (boundary) |
| FR-007 | A Wand of the War Mage +N MUST yield `SpellAttackRollBonus(+N, WhileHeld)` and `IgnoresHalfCoverOnSpellAttacks(WhileHeld)`, and MUST NOT yield a spell save DC bonus (the SRD names attack rolls only). | SRD 5.2.1, "Wand of the War Mage, +1, +2, or +3", p. 251 | unhappy (rule interaction) |
| FR-008 | The fixed-bonus items of table "Fixed bonus effects" MUST yield exactly the effects listed, each with its condition and its attunement requirement as a fact for FR-009. | SRD 5.2.1, as cited per row | happy |
| FR-009 | `ActiveItemEffects.Of(items, context)` MUST include an item's effects only when `ItemEffectContext.Access` is `fullMagical` (`magic-item-rules.md` FR-039) and the effect's `ItemEffectCondition` holds for `usage`; `nonmagicalBenefitsOnly` and `noMagic` MUST yield none ("a magic Shield that requires Attunement" gives only a nonmagical Shield's benefit until attuned). | SRD 5.2.1, "Attunement", p. 102 | unhappy (rule interaction) |
| FR-010 | `WhileWornWithoutArmorOrShield` MUST hold only when `wearsArmor` and `usesShield` are both false (Bracers of Defense); `WhileWorn` requires `usage` worn, `WhileHeld` held, `WhileOnPerson` carried (Scarab of Protection "on your person"). | SRD 5.2.1, "Bracers of Defense", p. 214; "Scarab of Protection", p. 243 | unhappy (rule interaction) |
| FR-011 | Active bonuses from different items MUST add, with no cap; each item's bonus counts once. The SRD has no rule limiting magic item bonuses (its only non-stacking rules are Proficiency Bonus and Advantage), so Ring of Protection plus Cloak of Protection is +2. Interpretation (Q3). Two copies of one attuned item cannot occur (`magic-item-rules.md` FR-050). | SRD 5.2.1, "No More Than Three Items", p. 102; "The Bonus Doesn't Stack", p. 8 (Proficiency Bonus only) | unhappy (rule interaction) |
| FR-012 | `ActiveItemEffects` MUST expose, per kind, a copied slice of `MagicItemBonus(modifier, instanceID)` in input order: `armorClass`, `savingThrows` (a plain value: the host decides which D20 Tests it feeds; Death Saving Throws are saving throws in the SRD but have no ability, so the host chooses, Q12), `spellAttackRolls`, `attackAndDamage` (per instance), plus `ignoresHalfCoverOnSpellAttacks`. The host maps `armorClass` to `derived-stats.md` flat AC bonuses (its FR-038). | SRD 5.2.1, "Ring of Protection", p. 238; `derived-stats.md` FR-038 | happy |
| FR-013 | `MagicItemEffectTable.Lookup(MagicItemID, EnhancementChoice)` MUST return `MagicItemEffectsFound(effects)` for the ids of tables "Fixed bonus effects", "Set and raised scores" and FR-026, and `MagicItemEffectsNotModelled` for any other id (no effect is invented). A +N item needs the resolved bonus (`ChosenRarity` through `magic-item-rules.md` FR-111, then FR-001). Host-defined items build `MagicItemEffect`s directly. | `shared-types-and-ownership.md` principle 2; SRD 5.2.1, as cited per row | unhappy (missing content) |

Table "Enhancement bonuses" (FR-001, FR-002). One test per cell.

| Family | Common | Uncommon | Rare | Very Rare | Legendary | Artifact |
|---|---|---|---|---|---|---|
| armor | not listed | not listed | +1 | +2 | +3 | not listed |
| shield, weapon, ammunition, warMageWand | not listed | +1 | +2 | +3 | not listed | not listed |

Table "Fixed bonus effects" (FR-008).

| Item | Effects | Condition | Source | Path |
|---|---|---|---|---|
| Ring of Protection (Rare, attune) | AC +1, saves +1 | worn | p. 238 | happy |
| Cloak of Protection (Uncommon, attune) | AC +1, saves +1 | worn | p. 216 | happy |
| Bracers of Defense (Rare, attune) | AC +2 | worn, no armor, no Shield | p. 214 | unhappy (rule interaction) |
| Scarab of Protection (Legendary, attune) | AC +1 | on person | p. 243 | happy |
| Robe of Stars (Very Rare, attune) | saves +1 | worn | p. 240 | happy |
| Dwarven Plate | AC +2 | worn | p. 219 | happy |
| Elven Chain; Glamoured Studded Leather | AC +1 each | worn | pp. 220, 223 | happy |
| Staff of Power (Very Rare, attune) | AC +2, saves +2, spell attack rolls +2; as Quarterstaff attack and damage +2 | held | p. 245 | happy |
| Staff of the Magi (Legendary, attune); Staff of the Woodlands (Rare, attune) | spell attack rolls +2 each; as Quarterstaff attack and damage +2 | held | pp. 246, 247 | happy |

Table "Bonus combination" (FR-009 to FR-012).

| Label | Items | Context | Expected | Path |
|---|---|---|---|---|
| ring and cloak | Ring and Cloak of Protection, both attuned | worn | AC +2, saves +2 | unhappy (rule interaction) |
| cloak not attuned | Cloak, `nonmagicalBenefitsOnly` | worn | none | unhappy (rule interaction) |
| bracers with armor | Bracers of Defense | `wearsArmor` true | none | unhappy (boundary) |
| bracers bare | Bracers of Defense | no armor, no Shield | AC +2 | happy |
| +2 armor and +1 shield | Armor +2, Shield +1 | worn, held | AC terms +2 and +1 | happy |
| shield not held | Shield +1 | `usage` worn, not held | none | unhappy (state error) |
| no items | empty | any | empty lists, no flag | unhappy (empty case) |

### 4.2 Ability-score items

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `SetAbilityScore(ability, score)` MUST give an effective score of `max(base, score)`: it sets the score when the base is lower and has no effect when the base is equal or higher ("no effect on you if your Strength without the belt is equal to or greater"). | SRD 5.2.1, "Belt of Giant Strength", p. 213; "Gauntlets of Ogre Power", p. 223 | unhappy (boundary) |
| FR-015 | The set-score items of table "Set and raised scores" MUST be modelled as `SetAbilityScore` with exactly those scores; a Belt of Giant Strength and a Potion of Giant Strength take the score from their `GiantStrengthType`, and `GiantStrengthType` has exactly five values. | SRD 5.2.1, "Belt of Giant Strength", p. 213; "Potion of Giant Strength", p. 235; "Amulet of Health", p. 209; "Headband of Intellect", p. 225 | happy |
| FR-016 | When several `SetAbilityScore` effects name one ability, the effective score MUST be the highest `max(base, scores...)`; they never add, whatever the order (Belt and Gauntlets together: the Belt wins, as 21 is above 19). Interpretation of "your Strength changes to a score granted by the belt" with the no-effect clause (Q4). | SRD 5.2.1, "Belt of Giant Strength", p. 213 | unhappy (rule interaction) |
| FR-017 | Effects on different abilities MUST be independent, and only active effects (FR-009) count. | SRD 5.2.1, "Gauntlets of Ogre Power", p. 223 | unhappy (rule interaction) |
| FR-018 | `RaiseAbilityScore(ability, amount, cap)` MUST apply after all set effects of FR-016 and give `max(current, min(current + amount, cap))`: Belt of Dwarvenkind raises Constitution by 2 to at most 20, and never lowers a score already above the cap. Sets-then-raises is an interpretation (Q11). | SRD 5.2.1, "Belt of Dwarvenkind", p. 213 | unhappy (boundary) |
| FR-019 | `BoostBestowedScore(amount 4, cap 30)` (Hammer of Thunderbolts, Might of Giants) MUST apply only when the Hammer is attuned (`fullMagical`) and a Belt of Giant Strength or Gauntlets of Ogre Power is **worn** (`usage`) and attuned, and its `SetAbilityScore` actually determined Strength (a set above the base); the boost goes on the winning set of FR-016 (Belt and Gauntlets together: the higher one), giving `min(bestowed + 4, 30)`; if the Belt or Gauntlets had no effect (base at or above the score) nothing is bestowed and nothing is added. A Potion of Giant Strength is not named, so it takes no boost. Interpretation of "the Strength score bestowed" (Q5). | SRD 5.2.1, "Hammer of Thunderbolts" (Giant's Bane: "attuned to the weapon and wearing either a Belt of Giant Strength or Gauntlets of Ogre Power to which you are also attuned"), p. 224 | unhappy (rule interaction) |
| FR-020 | `EffectiveAbilityScores.Of(base: AbilityScores, active)` MUST return the six effective `AbilityScore`s (always 1 to 30, `ability-scores.md` FR-025) and, per ability, the `AbilityScoreSource` that decided it (`BaseScore`, `SetByItem`, `RaisedByItem`); the input `AbilityScores` MUST NOT change. The host passes the result to `derived-stats.md` as the ability scores. | SRD 5.2.1, "Ability Scores", p. 5 | happy |
| FR-021 | Effects whose item states a duration (Potion of Giant Strength "for 1 hour") MUST carry it as a `time.Duration` of 60 minutes (data); expiry is the host's, which stops passing the effect. | SRD 5.2.1, "Potion of Giant Strength", p. 235 | happy |

Table "Set and raised scores" (FR-014 to FR-021).

| Item | Effect | Source |
|---|---|---|
| Amulet of Health (Rare, attune) | Constitution 19 | p. 209 |
| Gauntlets of Ogre Power (Uncommon, attune) | Strength 19 | p. 223 |
| Headband of Intellect (Uncommon, attune) | Intelligence 19 | p. 225 |
| Belt of Giant Strength (attune) / Potion of Giant Strength (1 hour) | hill 21 (Rare / Uncommon), frost or stone 23 (Very Rare / Rare), fire 25 (Very Rare / Rare), cloud 27 (Legendary / Very Rare), storm 29 (Legendary / Legendary) | pp. 213, 235 |
| Belt of Dwarvenkind (Rare, attune) | Constitution +2, cap 20 | p. 213 |
| Hammer of Thunderbolts (Legendary, attune) | bestowed Strength +4, cap 30 | p. 224 |

Table "Score cases" (FR-014 to FR-019).

| Label | Base | Items | Expected | Path |
|---|---|---|---|---|
| gauntlets raise | Str 10 | Gauntlets | Str 19 | happy |
| gauntlets, base equal | Str 19 | Gauntlets | 19, `BaseScore` | unhappy (boundary) |
| gauntlets, base higher | Str 20 | Gauntlets | 20 | unhappy (boundary) |
| belt and gauntlets | Str 10 | Belt (hill), Gauntlets | 21, `SetByItem(belt)` | unhappy (rule interaction) |
| storm belt | Str 8 | Belt (storm) | 29 | unhappy (boundary) |
| amulet on 18 | Con 18 | Amulet | 19 | unhappy (boundary) |
| dwarvenkind | Con 17; 19; 20; 22 | Belt of Dwarvenkind | 19; 20; 20; 22 | unhappy (boundary) |
| amulet then dwarvenkind | Con 12 | Amulet, Belt of Dwarvenkind | 19 then 20 (Q11) | unhappy (rule interaction) |
| might of giants on storm belt | Str 10 | Hammer, Belt (storm) | 30 (29 + 4 capped) | unhappy (boundary) |
| might of giants on gauntlets | Str 10; Str 19 | Hammer, Gauntlets | 23; 19 (no effect, no boost) | unhappy (rule interaction) |
| belt and gauntlets with hammer | Str 10 | Hammer, Belt (hill), Gauntlets, all worn | 25 (21 + 4, on the Belt) | unhappy (rule interaction) |
| hammer, belt not worn | Str 10 | Hammer, Belt (hill) carried only | 10 | unhappy (state error) |
| hammer alone | Str 10 | Hammer | 10 | unhappy (empty case) |
| inactive item | Str 10 | Gauntlets, `noMagic` | 10 | unhappy (state error) |

### 4.3 Potions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-022 | `PotionOfHealingGrade` MUST have four values with the dice and rarity of table "Potion of Healing grades"; `PotionOfHealingGrade.ForRarity(ItemRarity)` MUST map Common, Uncommon, Rare and Very Rare to `standard`, `greater`, `superior`, `supreme`. | SRD 5.2.1, "Potions of Healing", p. 236 | happy |
| FR-023 | `ForRarity` for Legendary or Artifact MUST return the closed `PotionOfHealingGradeNotListed(rarity)` and not return an error; `RollPotionOfHealing` then returns `RollPotionOfHealingNotListed(rarity)` with no request. | SRD 5.2.1, "Potions of Healing", p. 236 | unhappy (missing content) |
| FR-024 | `RollPotionOfHealing` (command: `ItemRarity`) MUST, for `greater`, `superior` and `supreme`, make exactly one `dice.Roll` request for the grade's `dice.Expression` (including its fixed modifier) and return `RollPotionOfHealingRolled(grade, amount)` with `amount` an `int` equal to the total (`4d4 + 4` gives 8 to 20); a roller that returns another expression propagates its typed failure unchanged and the input is untouched. The amount is not clamped to a Hit Point maximum; applying it is `hit-points-and-damage.md`. | SRD 5.2.1, "Potions of Healing", p. 236 | happy |
| FR-025 | One roll path: `standard` (Common) MUST NOT be rolled here; `RollPotionOfHealing` returns `RollPotionOfHealingStandardRolledElsewhere(grade)` without a request, because `UsePotionOfHealing` (`item-use-effects.md` FR-040) owns the Common potion's roll. `PotionOfHealingGrade.Standard.dice` is `2d4 + 2`, equal to the gear row, and is the constant that spec consumes (amendment A-5). | SRD 5.2.1, "Potion of Healing", p. 99; "Potions of Healing", p. 236 | unhappy (rule interaction) |
| FR-026 | `ResistanceTypeTable.ForRoll` MUST map 1 to 10 to Acid, Cold, Fire, Force, Lightning, Necrotic, Poison, Psychic, Radiant, Thunder; 0 and 11 MUST return a zero value and an error matching `srderr.ErrInvalidInput` (the signature is `ForRoll(d10 int) (DamageType, error)`; there is no panicking form because the roll is untrusted). The same table serves Potion of Resistance, Ring of Resistance and Armor of Resistance. | SRD 5.2.1, "Potion of Resistance", p. 236; "Ring of Resistance", p. 238; "Armor of Resistance", p. 210 | unhappy (boundary) |
| FR-027 | `GrantedResistance.Chosen(type, duration)` MUST accept only a type from the table's ten ("the GM chooses the type or determines it randomly by rolling on the following table") and return the closed `GrantedResistanceRejected(typeNotListed)` for Bludgeoning, Piercing or Slashing; the potion's duration is 60 minutes, the ring's and armor's is "while wearing". Interpretation (Q6). | SRD 5.2.1, "Potion of Resistance", p. 236 | unhappy (invalid input) |

Table "Potion of Healing grades" (FR-022, FR-024).

| Grade | Dice | Rarity | Range |
|---|---|---|---|
| standard | 2d4 + 2 | Common | 4 to 10 |
| greater | 4d4 + 4 | Uncommon | 8 to 20 |
| superior | 8d4 + 8 | Rare | 16 to 40 |
| supreme | 10d4 + 20 | Very Rare | 30 to 60 |

### 4.4 Charged spell items

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-028 | `ChargedSpellOption` MUST be a closed variant set (interface with an unexported marker method) with exactly `FixedChargeCost(spellID, charges, level, stats)` (`charges` 0 or more) and `ChargesPerLevel(spellID, lowestLevel, maximumCharges, stats)` (`maximumCharges` at least 1), where `level` is `LowestLevel` or `NamedLevel(SpellLevel)`; invalid values yield an error matching `srderr.ErrInvalidInput` from the constructor (`NewFixedChargeCost`, `NewChargesPerLevel`; the `Must` forms panic). | SRD 5.2.1, "Staff of Fire", p. 245; "Wand of Magic Missiles", p. 251 | unhappy (invalid input) |
| FR-029 | For `ChargesPerLevel`, spending `c` charges (1 to `maximumCharges`) MUST cast the spell at level `lowestLevel + c - 1`: "for 1 charge, you cast the level 1 version... increase the spell's level by 1 for each additional charge", "no more than 3 charges". Staff of Healing's Cure Wounds is `ChargesPerLevel(lowest 1, max 4)` ("1 charge per spell level"). | SRD 5.2.1, "Wand of Magic Missiles", "Wand of Lightning Bolts", p. 251; "Staff of Healing", p. 245 | happy |
| FR-030 | `ChargedSpellCasting.Plan` MUST reject with `ChargedSpellPlanRejected(reason)`: `chargesOutOfRange` for 0 or more than `maximumCharges` on `ChargesPerLevel`, `chargesNotAllowed` for any count other than the fixed cost on `FixedChargeCost`; `plan` makes no change to charges. | SRD 5.2.1, "Wand of Magic Missiles", p. 251 | unhappy (boundary) |
| FR-031 | `FixedChargeCost` with `LowestLevel` MUST cast at the spell's lowest level (the caller passes it, so `plan` needs no content lookup) and with `NamedLevel(n)` at that level ("Fireball (level 5 version)": 5 charges). | SRD 5.2.1, "Spells Cast from Items", p. 206; "Staff of Power", p. 246 | happy |
| FR-032 | A `FixedChargeCost` of 0 charges ("Detect Magic 0", "Light 0") MUST plan a cast that spends nothing: `ChargedSpellCasting.Cast` returns the `ItemCharges` unchanged and no `MagicItemChargesSpent` event (`ItemCharges.Spend` rejects 0, `magic-item-rules.md` FR-030). | SRD 5.2.1, "Staff of the Magi", pp. 246-247 | unhappy (boundary) |
| FR-033 | `ChargedSpellCasting.Cast(charges, plan, ids)` MUST spend through `ItemCharges.Spend` and return its `MagicItemInsufficientCharges(current, requested)` unchanged when the item has too few charges (`magic-item-rules.md` FR-031); on success it returns the new charges, the `MagicItemChargesSpent` event and the `SpellFromItem`. | SRD 5.2.1, "Charges", p. 206 | unhappy (state error) |
| FR-034 | The `SpellFromItem` of a plan MUST have delivery `CastFromItem`, default components and terms, `casterLevel` equal to `ItemCasterLevel.Lowest` and `castAtLevel` equal to the level of FR-029 or FR-031, and `userMustConcentrate` per `magic-item-rules.md` FR-026 from the `requiresConcentration` the caller passes. The cast level may exceed the spell's lowest level here only because the item's description names it (amendment A-2 to `magic-item-rules.md` FR-025; **this requirement is blocked on A-2**, because that FR-025 still says `castAtLevel` MUST equal the lowest level). | SRD 5.2.1, "Spells Cast from Items", p. 206 | unhappy (rule interaction) |
| FR-035 | `ItemSpellStats.UsersSpellStats` MUST mean the spell uses the user's spell save DC, spell attack modifier or spellcasting ability modifier, resolved by `ItemSpellcastingAbility` (`magic-item-rules.md` FR-027); `FixedSaveDC(n)` (Wand of Lightning Bolts DC 15, Wand of Polymorph DC 15, Wand of Web DC 13) MUST be used as given; staff spells use "your spell save DC" and Staff of Healing "your spellcasting ability modifier" (`UsersSpellStats`). | SRD 5.2.1, "Spells Cast from Items", p. 206; "Wand of Lightning Bolts", "Wand of Polymorph", "Wand of Web", p. 251; "Staff of Healing", p. 245 | happy |
| FR-036 | `ChargedSpellOption` MUST NOT model spell slots: a cast through charges spends no slot, so `plan` and `cast` accept no `SpellSlots` and the result has no slot event. | SRD 5.2.1, "Spells Cast from Items", p. 206 | unhappy (rule interaction) |

Table "Charged spell rows" (FR-028 to FR-033). Each row is one test; item dice for regained charges are `magic-item-rules.md` FR-034.

| Item | Option | Charges to level | Source |
|---|---|---|---|
| Wand of Magic Missiles | `ChargesPerLevel(magic_missile, 1, 3)` | 1, 2, 3 charges give level 1, 2, 3; 4 rejected | p. 251 |
| Wand of Lightning Bolts | `ChargesPerLevel(lightning_bolt, 3, 3)`, DC 15 | 1, 2, 3 give level 3, 4, 5 | p. 251 |
| Staff of Healing | `ChargesPerLevel(cure_wounds, 1, 4)`; `FixedChargeCost(lesser_restoration, 2)`, `FixedChargeCost(mass_cure_wounds, 5)` | 1 to 4 give level 1 to 4 | p. 245 |
| Staff of Fire | `FixedChargeCost` Burning Hands 1, Fireball 3, Wall of Fire 4 | lowest levels | p. 245 |
| Staff of Power | Fireball `NamedLevel(5)` 5; Lightning Bolt `NamedLevel(5)` 5; Globe of Invulnerability 6; Magic Missile 1 | as listed | p. 246 |
| Staff of the Magi | Arcane Lock 0, Detect Magic 0, Fireball `NamedLevel(7)` 7, Conjure Elemental 7, Plane Shift 7 | zero cost spends nothing | pp. 246-247 |
| Staff of the Woodlands | Animal Friendship 1, Awaken 5, Wall of Thorns 6 | lowest levels | p. 247 |

### 4.5 Last charge, absorption and Retributive Strike

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-037 | `LastChargeRule` MUST be a closed variant set (interface with an unexported marker method): `NoLastChargeEffect`; `LastChargeAutomatic(consequence)` (no roll); `LastChargeRolled(onOne: LastChargeConsequence, onTwenty: LastChargeRegain)`, where `LastChargeConsequence.None` and `NoRegain` mean "no effect" so no nil value stands for a state. The d20 target numbers (1 and 20) are constants, not fields. | SRD 5.2.1, "Wand of Magic Missiles", p. 251; "Staff of Power", p. 246 | happy |
| FR-038 | `ResolveLastCharge` MUST be rejected with `ResolveLastChargeRejected(chargesRemain)` and make no request when `ItemCharges.Current` is above 0; the host calls it right after the spend that emptied the item ("if you expend the last charge"). | SRD 5.2.1, "Wand of Magic Missiles", p. 251 | unhappy (state error) |
| FR-039 | For `LastChargeRolled` with 0 charges, it MUST make exactly one `dice.Roll` request of `1d20`; a natural 1 gives the rule's `onOne` consequence; a natural 20 with `RegainsDice(d)` makes one more request for `d` and gives that many charges (capped at `maximum`, `magic-item-rules.md` FR-034); any other face changes nothing. | SRD 5.2.1, "Staff of Power", p. 246; "Staff of the Magi", p. 247 | unhappy (boundary) |
| FR-040 | A roll of 1 with `onOne` `none` (Staff of the Magi) and a 20 with `NoRegain` (every wand) MUST give no consequence and no regain; the dice are still rolled once. | SRD 5.2.1, "Staff of the Magi", p. 247 | unhappy (rule interaction) |
| FR-041 | `LastChargeAutomatic` MUST apply its consequence with no roll (Ring of Three Wishes and Gem of Brightness `becomesNonmagical`, Scarab of Protection `destroyed`); `NoLastChargeEffect` MUST change nothing. | SRD 5.2.1, "Ring of Three Wishes", p. 239; "Gem of Brightness", p. 223; "Scarab of Protection", p. 243 | happy |
| FR-042 | `ResolveLastCharge` MUST return the new `ItemCharges`, the `LastChargeConsequence` and a `MagicItemLastChargeResolved` event (FR-068); a `destroyed` or `becomesNonmagical` item has lost its magic, which the host records (the item's own text says how: "crumbles", "vanishes in a flash of light"). | SRD 5.2.1, "Staff of Healing", p. 245 | happy |
| FR-043 | The regained dice use `dice.Expression` with a fixed modifier (`1d8 + 2`, `1d12 + 1`), the same type as recharge dice. A `dice.Roller` failure is returned wrapped by `srderr.WrapPort` with the cause preserved and the input is unchanged (`magic-item-rules.md` FR-106). | SRD 5.2.1, "Staff of Power", p. 246 | unhappy (failure) |
| FR-044 | The rows of table "Last charge rules" MUST hold; an item not in it has `NoLastChargeEffect` until a row is added (Q7). | SRD 5.2.1, as cited per row | unhappy (missing content) |
| FR-045 | `StaffSpellAbsorption.Absorb(charges, spellLevel)` (Staff of the Magi) MUST return charges `current + spellLevel` and a `StaffSpellAbsorbed` event while the total is at most the `maximum` (50), and `StaffSpellAbsorptionOverflowed(total)` when it would exceed 50 ("the staff explodes as if you activated its Retributive Strike"); the explosion uses `RetributiveStrike.ForCharges(staff, total)`, the charge count after adding the spell level (interpretation: the staff holds that many when it explodes; 49 + 3 gives 52); absorption is the host's Reaction to a spell that targets only the wearer, and `absorb` assumes it (a multi-target or area spell must not call it); a spell level of 0 or below 0 returns an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Staff of the Magi", p. 247 | unhappy (boundary) |
| FR-046 | `RetributiveStrike.ForCharges(staff, charges)` MUST return `RetributiveStrikeDamage(self, areaFailed, areaSucceeded)` as Force damage: `self` 16 times the charges; `areaFailed` 4 times (Staff of Power) or 6 times (Staff of the Magi) the charges; `areaSucceeded` half of that rounded down; DC 17 Dexterity is a constant. Zero charges gives all zeros; a negative count returns an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Staff of Power", p. 246; "Staff of the Magi", p. 247; "Round Down", p. 5 | unhappy (boundary) |
| FR-047 | `RetributiveStrike` MUST NOT roll the 50 percent travel chance or apply damage; the host does (out of scope). | SRD 5.2.1, "Staff of Power", p. 246 | unhappy (rule interaction) |

Table "Last charge rules" (FR-037 to FR-044). `destroyed` means "the item is gone" whatever the printed wording (interpretation: every wording ends the item). A wand not listed (Wand of Magic Detection, Wand of Secrets, the War Mage wand) prints no last-charge sentence and has `NoLastChargeEffect`.

| Item | Rule | Source |
|---|---|---|
| Wand of Binding; Wand of Enemy Detection; Wand of Fear; Wand of Fireballs | `Rolled(destroyed, NoRegain)` ("crumbles into ashes") | pp. 250-251 |
| Wand of Lightning Bolts; Wand of Magic Missiles; Wand of Paralysis; Wand of Polymorph; Wand of Web | `Rolled(destroyed, NoRegain)` ("crumbles into ashes") | p. 251 |
| Wand of Wonder | `Rolled(destroyed, NoRegain)` ("crumbles into dust") | p. 251 |
| Rod of Resurrection | `Rolled(destroyed, NoRegain)` ("disappears in a harmless burst of radiance") | p. 242 |
| Staff of Charming | `Rolled(destroyed, NoRegain)` ("crumbles to dust") | p. 245 |
| Staff of Frost | `Rolled(destroyed, NoRegain)` ("turns to water") | p. 245 |
| Staff of Fire | `Rolled(destroyed, NoRegain)` ("crumbles into cinders") | p. 245 |
| Staff of Swarming Insects | `Rolled(destroyed, NoRegain)` ("consumes and destroys the staff") | p. 246 |
| Staff of Healing | `Rolled(destroyed, NoRegain)` ("vanishes in a flash of light, lost forever") | p. 245 |
| Staff of Striking; Staff of the Woodlands | `Rolled(becomesNonmagical, NoRegain)` (nonmagical Quarterstaff) | pp. 246, 247 |
| Staff of Power | `Rolled(retainsBonusLosesOtherProperties, RegainsDice(1d8+2))` | p. 246 |
| Staff of the Magi | `Rolled(none, RegainsDice(1d12+1))` | p. 247 |
| Ring of Three Wishes; Gem of Brightness | `Automatic(becomesNonmagical)` | pp. 239, 223 |
| Scarab of Protection | `Automatic(destroyed)` | p. 243 |

### 4.6 Ring of Spell Storing

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | `SpellStorage` MUST hold `capacityLevels` 5 and an ordered copied slice of `StoredSpell`; the used levels are the sum of the stored `slotLevel`s and MUST NOT exceed 5; the constructor returns an error matching `srderr.ErrInvalidInput` otherwise. | SRD 5.2.1, "Ring of Spell Storing", p. 239 | unhappy (boundary) |
| FR-049 | `RollSpellStoringRingLevels` MUST make one `dice.Roll` request of `1d6` and return stored levels `roll - 1` (0 to 5, "1d6 - 1 levels"); a failing `dice.Roller` is returned wrapped by `srderr.WrapPort` with the cause preserved and a roller returning another expression is a typed failure, never a result; `SpellStorage.Initial(levels, spells)` MUST accept spells whose slot levels sum exactly to `levels` (chosen by the GM, the host) and return `SpellStorageRejected(levelsDoNotMatch)` otherwise. | SRD 5.2.1, "Ring of Spell Storing", p. 239 | unhappy (boundary) |
| FR-050 | `SpellStorage.Store(spell)` MUST accept a spell cast into the ring with a slot of level 1 to 5 when used plus `slotLevel` is at most 5, with the slot level (not the spell's own level) deciding the space; it returns the new storage and `RingSpellStored`. | SRD 5.2.1, "Ring of Spell Storing", p. 239 | happy |
| FR-051 | When the ring cannot hold the spell, `store` MUST return `SpellStorageSpellExpended(spell)` with the storage unchanged ("the spell is expended without effect"). A slot level of 0 or above 5 MUST return `SpellStorageNotStorable(slotLevel)`: only "a spell of level 1 through 5" can be cast into the ring (a cantrip is not stored; interpretation, Q8). | SRD 5.2.1, "Ring of Spell Storing", p. 239 | unhappy (rule interaction) |
| FR-052 | `SpellStorage.Cast(index, access)` MUST remove the stored spell at that position, free its levels, and return `StoredSpellCast(spell, storage)` and `RingStoredSpellCast`; a position outside the list returns `SpellStorageNoSuchSpell(index)`. | SRD 5.2.1, "Ring of Spell Storing", p. 239 | happy |
| FR-053 | The cast MUST carry the original caster's slot level, spell save DC, spell attack bonus and spellcasting ability (the `StoredSpell` fields), "otherwise treated as if you cast the spell", so the host resolves it with the original numbers, not the wearer's. | SRD 5.2.1, "Ring of Spell Storing", p. 239 | unhappy (rule interaction) |
| FR-054 | Only "the attuned wearer" can cast: `cast` MUST return `SpellStorageNotUsable(access)` unless `access` is `fullMagical` (`magic-item-rules.md` FR-039), with the storage unchanged. Storing needs no attunement ("any creature can cast a spell... into the ring"). | SRD 5.2.1, "Ring of Spell Storing", p. 239 | unhappy (rule interaction) |
| FR-055 | `StoredSpell` MUST hold the caster's values as supplied (`SpellSaveDC`, `SpellAttackBonus`, `Ability`), never recomputed; slot level 1 to 5 else an error matching `srderr.ErrInvalidInput`. Two stored copies of one spell are allowed. | SRD 5.2.1, "Ring of Spell Storing", p. 239 | unhappy (invalid input) |
| FR-056 | `SpellStorage` MUST be immutable: every operation returns a new instance, the input and the exposed list are unchanged, equal contents are equal. | `doc/rules/architecture.md` sections 3, 4 | unhappy (immutability) |

Table "Ring of Spell Storing" (FR-048 to FR-054).

| Label | Used | Action | Expected | Path |
|---|---|---|---|---|
| fits exactly | 2 | store slot 3 | stored, used 5 | unhappy (boundary) |
| one too many | 3 | store slot 3 | expended, storage unchanged | unhappy (boundary) |
| upcast uses slot level | 0 | level-1 spell with a level-3 slot | uses 3 | unhappy (rule interaction) |
| cantrip | 0 | slot level 0 | `NotStorable` | unhappy (boundary) |
| level 6 | 0 | slot level 6 | `NotStorable` | unhappy (boundary) |
| not attuned | 3 | cast, `nonmagicalBenefitsOnly` | `NotUsable`, unchanged | unhappy (rule interaction) |
| cast frees space | 5 | cast the 2-level spell | used 3 | happy |
| initial roll | | 1d6 of 1; 6 | 0 levels; 5 levels | unhappy (boundary) |

### 4.7 Extradimensional containers

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-057 | `ExtradimensionalSpace` MUST have a capacity per table "Extradimensional spaces": Bag of Holding 500 pounds and 64 cubic feet; each Handy Haversack side pouch 200 pounds and 25 cubic feet, the central pouch 500 and 64; Portable Hole no stated limit. Loads are whole ounces and cubic inches (16 ounces to the pound, 1,728 cubic inches to the cubic foot; package design, so no floating-point comparison). | SRD 5.2.1, "Bag of Holding", p. 212; "Handy Haversack", p. 224; "Portable Hole", pp. 234-235 | happy |
| FR-058 | A pouch is overloaded when its load is above the weight **or** the volume limit; a load exactly at a limit is not overloaded ("up to 500 pounds... not exceeding 64 cubic feet"). `SpaceCheck.Of(space, load)` returns `SpaceWithinCapacity` or `SpaceOverloaded(pouch)`; a negative load returns an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Bag of Holding", p. 212 | unhappy (boundary) |
| FR-059 | An overloaded, pierced or torn Bag of Holding MUST be destroyed with `ContainerContentsScattered` (contents scattered in the Astral Plane); a Handy Haversack with any pouch overloaded, pierced or torn MUST be destroyed with `ContainerContentsLost` (lost forever; an Artifact turns up again somewhere); a Portable Hole has no overload rule and `SpaceCheck` never reports it overloaded. `ContainerDamage.PiercedOrTorn` is a host fact. | SRD 5.2.1, "Bag of Holding", p. 212; "Handy Haversack", p. 224 | unhappy (rule interaction) |
| FR-060 | `ExtradimensionalNesting.Place(placed, into)` MUST return `NestingDestroysBoth(gate)` when a Bag of Holding, Handy Haversack, Portable Hole or `SimilarSpace` item is placed inside an extradimensional space of another such item (any ordered pair of the four kinds, including the same kind twice: Bag in Bag, Haversack in Haversack, Hole in Hole), and `NestingSafe` when the placed object is not an extradimensional item. Both items are destroyed. Same-kind pairs rest on the "similar item" wording (a Hole in a Hole has no explicit wording); interpretation (Q9). | SRD 5.2.1, "Bag of Holding", p. 212; "Handy Haversack", p. 224; "Portable Hole", p. 235 | unhappy (rule interaction) |
| FR-061 | The `AstralGate` MUST originate where the one item was placed inside the other, be one-way, close after creatures are moved and never reopen (`oneWay` true, `reopenable` false), and move each creature in its area to a random location on the Astral Plane (the roll and the move are the host's). | SRD 5.2.1, "Bag of Holding", p. 212; "Portable Hole", p. 235 | happy |
| FR-062 | The `GateArea` MUST follow the **placed** item's own text: placing a Bag of Holding gives a 10-foot-radius Sphere centered on the gate; placing a Handy Haversack or Portable Hole gives each creature within 10 feet of the gate and not behind Total Cover. The texts differ and the SRD does not say which governs when two items meet; the placed item's sentence ("Placing a Bag of Holding inside...") is the one that names the act (interpretation, Q9). Same-kind pairs follow the same rule: Bag in Bag a 10-foot-radius Sphere; Haversack in Haversack and Hole in Hole within 10 feet and not behind Total Cover. A `SimilarSpace` carries its own `GateArea` as data. | SRD 5.2.1, "Bag of Holding", p. 212; "Handy Haversack", p. 224; "Portable Hole", p. 235 | unhappy (rule interaction) |
| FR-063 | `AirSupply.Of(space, breathingCreatures)` MUST return `AirForMinutes(base / n)` (integer division) with base 10 (Bag, each Haversack pouch) or 60 (closed Portable Hole), n at least 1, rounded down; `NoOneBreathing` for 0; a negative count returns an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Bag of Holding", p. 212; "Portable Hole", p. 235; "Round Down", p. 5 | unhappy (boundary) |
| FR-064 | `ExtradimensionalSpace` MUST expose the fixed carried weight as a closed `ContainerWeight`: `FixedWeight(5 pounds)` for the Bag and Haversack ("regardless of its contents") and `WeighsNextToNothing` for the Portable Hole (no number is invented). | SRD 5.2.1, "Bag of Holding", p. 212; "Portable Hole", p. 235 | unhappy (missing content) |
| FR-065 | `PortableHoleSpace` MUST expose its escape facts as data: depth 10 feet, a creature in a folded hole may take an action for a DC 10 Strength (Athletics) check and on a success appears within 5 feet of the hole. The check is made by the host through `d20`. | SRD 5.2.1, "Portable Hole", p. 235 | happy |

Table "Extradimensional spaces" (FR-057 to FR-065).

| Space | Capacity | Overload | Air | Weight |
|---|---|---|---|---|
| Bag of Holding | 500 lb, 64 cu ft | destroyed, scattered | 10 min | 5 lb |
| Handy Haversack, side pouch (two) | 200 lb, 25 cu ft each | any pouch: destroyed, lost | 10 min per pouch | 5 lb |
| Handy Haversack, central pouch | 500 lb, 64 cu ft | as above | 10 min | 5 lb |
| Portable Hole | none stated | none | 60 min when folded | next to nothing |

Table "Nesting and load cases" (FR-058 to FR-063).

| Label | Input | Expected | Path |
|---|---|---|---|
| bag at limit | 500 lb, 64 cu ft | within capacity | unhappy (boundary) |
| bag one ounce over | 500 lb 1 oz, 64 cu ft | overloaded | unhappy (boundary) |
| bag volume over, weight under | 100 lb, 64 cu ft 1 cu in | overloaded | unhappy (boundary) |
| side pouch 201 lb | 201 lb, 1 cu ft | overloaded (pouch named) | unhappy (boundary) |
| bag into haversack | placed Bag, into Haversack | both destroyed, Sphere 10 ft | unhappy (rule interaction) |
| haversack into bag; hole into bag | placed Haversack; placed Hole | both destroyed, within 10 ft and not behind Total Cover | unhappy (rule interaction) |
| bag into bag | placed Bag, into Bag | both destroyed (similar item), Sphere 10 ft | unhappy (rule interaction) |
| haversack into haversack; hole into hole | placed Haversack, into Haversack; placed Hole, into Hole | both destroyed, within 10 ft and not behind Total Cover (Q9) | unhappy (rule interaction) |
| ordinary object | placed Rope | `NestingSafe` | unhappy (empty case) |
| air | Bag, 3 creatures; Hole, 0; Hole, 7 | 3 min; no one breathing; 8 min | unhappy (boundary) |

### 4.8 Cross-cutting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-066 | Every domain value and service of this spec MUST be immutable and pure: no clock, no randomness, no I/O; exposed slices are copies; equal inputs give equal outputs. | `doc/rules/architecture.md` sections 3, 4 | unhappy (immutability) |
| FR-067 | The three use cases (`RollPotionOfHealing` for the grades it rolls, `ResolveLastCharge`, `RollSpellStoringRingLevels`) MUST give the same result with the zero-value (Noop) `telemetry.Telemetry` as with a recording one, record one span and one counter increment per call, and use `srd5e.outcome` `success` or `failure` only (`magic-item-rules.md` FR-105, FR-109). | `doc/rules/observability.md` sections 5, 9 | happy |
| FR-068 | The `ChargedItemEvent` family MUST be a closed variant set (interface with an unexported marker method), implement `CharacterEvent`, and contain exactly `MagicItemLastChargeResolved`, `RingSpellStored`, `RingStoredSpellCast` and `StaffSpellAbsorbed`, each past tense, defined only here, each carrying the `CharacterID` (required input, never defaulted) and `MagicItemInstanceID`; charge spending reuses `MagicItemChargesSpent`. A rejection emits no event. | `shared-types-and-ownership.md` FR-019, FR-021; `magic-item-rules.md` FR-122 | unhappy (missing content) |
| FR-069 | Exported names MUST be the section 3 types, the services of section 5, the three use cases with their results and the events of FR-068, each exported once from package `character` (`github.com/Whealinghold/srd5e-go/character`); no generic name (`Applied`, `Rejected`, `Found`, `NotListed` alone) and every result variant carries its use-case or type prefix. | `shared-types-and-ownership.md` FR-017; `doc/rules/architecture.md` section 9 | unhappy (name collision) |
| FR-070 | Package `character` MUST NOT import package `combat`; healing, Resistance and Strength are returned as values. No use case calls another (`shared-types-and-ownership.md` FR-044). | `shared-types-and-ownership.md` FR-013, FR-044 | unhappy (forbidden edge) |

### 4.9 Withdrawn ids

None. This is a new spec.

## 5. Model

**Context split.** All types are `character` (behaviour and the effect table, Q2). `content` stays free of them: `content-magic-items.md` keeps `MagicItemFacts` and the categories; this spec reads them and `magic-item-rules.md`'s state types.

**Value objects:** `MagicItemEffect` and its variants, `ItemEffectCondition`, `ItemEffectContext`, `MagicItemBonus`, `ActiveItemEffects`, `EffectiveAbilityScores`, `PotionOfHealingGrade`, `GrantedResistance`, `ChargedSpellOption`, `ChargedSpellPlan`, `LastChargeRule`, `RetributiveStrikeDamage`, `SpellStorage`, `StoredSpell`, `SpaceLoad`, `AstralGate`, `GateArea`.

**Domain services** (pure, stateless): `EnhancementBonusLookup`, `MagicAmmunition`, `MagicItemEffectTable`, `ChargedSpellCasting`, `StaffSpellAbsorption`, `RetributiveStrike`, `ResistanceTypeTable`, `SpaceCheck`, `ExtradimensionalNesting`, `AirSupply`.

**Failures.** Expected outcomes are results (`not listed`, `rejected`, `overloaded`, `expended`); untrusted values (a negative load, an invalid slot level, a non-positive `maximumCharges`, a roll outside its table) are rejected by `NewX` / `ParseX`, which return `(T, error)` with the error matching `srderr.ErrInvalidInput`; `MustX` helpers (used only for literals in tables and tests) panic with an error value. Port failures (`dice.Roller`) are returned as `*srderr.PortError` (built with `srderr.WrapPort`), unchanged beneath the wrapper; callers branch with `errors.Is` / `errors.As`.

**Invariants and where enforced:** used levels at most 5 (`SpellStorage` constructor); charges within range (`ItemCharges`, `magic-item-rules.md` FR-029); effective scores 1 to 30 (`AbilityScore`); closed families for every "one of" (`LastChargeRule`, `ChargedSpellOption`, `ContainerWeight`).

## 6. Interface

| Use case | Command | Result variants | Go entry point | Ports |
|---|---|---|---|---|
| `RollPotionOfHealing` | `ItemRarity` | `RollPotionOfHealingRolled(grade, amount)`, `RollPotionOfHealingNotListed(rarity)`, `RollPotionOfHealingStandardRolledElsewhere(grade)` | `func (u PotionRoller) RollPotionOfHealing(ctx context.Context, r ItemRarity) (RollPotionOfHealingResult, error)` | `dice.Roller` |
| `ResolveLastCharge` | `CharacterID`, `MagicItemInstanceID`, `ItemCharges`, `LastChargeRule` | `ResolveLastChargeResolved(charges, consequence, event)`, `ResolveLastChargeRejected(reason)` | `func (u LastChargeResolver) ResolveLastCharge(ctx context.Context, c ResolveLastChargeCommand) (ResolveLastChargeResult, error)` | `dice.Roller` |
| `RollSpellStoringRingLevels` | none | `RollSpellStoringRingLevelsRolled(levels)` | `func (u RingLevelRoller) RollSpellStoringRingLevels(ctx context.Context) (RollSpellStoringRingLevelsResult, error)` | `dice.Roller` |

Everything else is a pure domain service or value-object method (FR-001 to FR-021, FR-026 to FR-036, FR-045 to FR-047, FR-048 to FR-065). Every call is synchronous: the only port is `dice.Roller`, which is in-memory and takes no `context.Context`. The use-case entry points take `ctx` first only because they start a span (decision 2b); the result is a closed variant set (interface with an unexported marker method; the variants in the table are the whole set) and the `error` is non-nil only for a port failure or a port that breaks its contract. Each use case holds its `dice.Roller` and a `telemetry.Telemetry` (zero value usable, Noop) set through functional options at construction; there are no globals.

- **Port failure contract:** a `dice.Roller` failure or a roll of another expression is returned wrapped by `srderr.WrapPort` with the cause preserved; no new port, no new double (`dicetest.FixedRoller` and the recording `telemetrytest` double exist).
- **Exports:** package `character` exports the section 3 types, services, use cases, results and events (no facade or re-export package). Test doubles stay in the existing `dice/dicetest` and `telemetry/telemetrytest` packages; no new `*test` package is added.
- **Dependencies:** package `character` imports only kernel packages (`dice`, `ability`, `modifier`, `srderr`, `telemetry`) and the published API of `content`; this spec uses no `d20` type except through `spellcasting.md`'s `SpellSaveDC`.

## 7. Observability

Domain services and value objects are **silent by design**: they are pure and return what happened as values and events. The three use cases are instrumented.

1. **What can go wrong?** A `dice.Roller` failure or wrong-expression roll; a rule outcome (rarity not listed, charges remaining); programmer errors.
2. **How would an operator know?** A returned failure: span error status and `error.type` `dice_roller_error`. A returned variant: `srd5e.outcome` and `srd5e.character.result`. A rule outcome sets no error status.
3. **What is needed to diagnose?** Operation, outcome, result and, for the last charge, the consequence; ids and amounts are in the returned result.
4. **What is counted or timed?** Calls and duration through the existing `srd5e.character.operations` counter and `srd5e.character.duration` histogram, which gain the three operation values below. No new instrument.
5. **Is existing instrumentation still correct?** Yes; `srd5e.character.result` is not a closed set (`character-aggregate.md` FR-088); this spec adds `rolled`, `resolved`, `not_listed` and `rolled_elsewhere`.
6. **What should a consumer monitor?** Any error status or `error.type`; a rising `rejected` share for `resolve_last_charge` (a host calling too early).
7. **Is anything now noisy?** No; no log is emitted.

`srd5e.outcome`: `success` for `Rolled`, `Resolved`; `failure` for `NotListed`, `StandardRolledElsewhere` and `Rejected`.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.roll_potion_healing` | span | `srd5e.outcome`, `srd5e.character.result` (`rolled`, `not_listed`, `rolled_elsewhere`) | One Potion of Healing roll (greater to supreme) | None: `not_listed` and `rolled_elsewhere` mean the host offered the wrong rarity |
| `srd5e.character.resolve_last_charge` | span | `srd5e.outcome`, `srd5e.character.result` (`resolved`, `rejected`), `srd5e.character.rejection` (`charges_remain`), `srd5e.character.last_charge_consequence` (`none`, `destroyed`, `becomes_nonmagical`, `retains_bonus`) | One last-charge resolution | A rise in `charges_remain` shows hosts calling before the item is empty |
| `srd5e.character.roll_spell_storing_levels` | span | `srd5e.outcome`, `srd5e.character.result` (`rolled`) | One ring contents roll | None |

The shared counter's `srd5e.character.operation` gains `roll_potion_healing`, `resolve_last_charge`, `roll_spell_storing_levels`. Attribute values are low-cardinality; no id is an attribute. Names are added to `doc/observability/signals.md` with this change.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002 | unit | Table "Enhancement bonuses", every cell (6 rarities by 2 rows), not-listed carries family and rarity. |
| FR-003 to FR-008 | unit | One test per effect and per row of "Fixed bonus effects"; Shield term separate from the Shield's own bonus; `afterAttack` hit, miss and already nonmagical; War Mage has attack bonus and Half Cover flag but no save DC. |
| FR-009 to FR-013 | unit | Table "Bonus combination", every row; each `MagicalPropertiesAccessLevel`; each `ItemEffectCondition`; `MagicItemEffectTable.Lookup` for every modelled id, an unknown id, and a +N item with an unresolved bonus. |
| FR-014 to FR-021 | unit | Tables "Set and raised scores" and "Score cases", every row; equal, one below, one above the set score; cap at 20 and 30; input `AbilityScores` unchanged; five giant types; duration data. |
| FR-022 to FR-025 | unit and application | Table "Potion of Healing grades"; `ForRarity` for all six rarities; `RollPotionOfHealing` with `dicetest.FixedRoller` for greater, superior, supreme: one request of the exact expression, minimum and maximum faces give the range ends; Common returns `StandardRolledElsewhere` with no request and `standard.dice` equals `2d4 + 2` (the gear row); failing roller is returned wrapped as a `*srderr.PortError` with the cause preserved and the input is untouched; a roller returning the wrong expression fails typed. |
| FR-026, FR-027 | unit | Rolls 0, 1, 10, 11, -1 (`ForRoll` returns the error); chosen types in and out of the ten; the potion's `time.Duration` of 60 minutes and the ring's and armor's "while wearing" data. |
| FR-028 to FR-036 | unit | Table "Charged spell rows", every row; charges 0, 1, max, max + 1; zero cost spends nothing and emits no event; too few charges returns `MagicItemInsufficientCharges` unchanged; `SpellFromItem` fields and Concentration; no slot parameter exists (compile check by architecture scan). |
| FR-037 to FR-044 | application | `ResolveLastCharge` with `dicetest.FixedRoller`: faces 1, 2, 19, 20 for each rule shape; exactly one `1d20` request, plus one for regain on 20; none for automatic and no-effect rules; rejected with charges above 0 and no request; regain capped at maximum; table "Last charge rules", one assertion per row; failing roller is returned wrapped by `srderr.WrapPort` with the cause preserved; a roller returning the wrong expression fails typed. |
| FR-045 to FR-047 | unit | Absorption at 49 + 1, 50 + 1, 49 + 3 (explosion at 52 charges), level 0; Retributive Strike for 0, 1, 7, 50 charges on both staffs (half rounded down); negative count returns an error. |
| FR-048 to FR-056 | unit and application | Table "Ring of Spell Storing", every row; `RollSpellStoringRingLevels` faces 1 and 6, a failing roller and a wrong-expression roller as above; initial spells matching and not matching; immutability and equality. |
| FR-057 to FR-065 | unit | Tables "Extradimensional spaces" and "Nesting and load cases", every row; every ordered pair of the four space kinds for nesting (16, including the three same-kind pairs, each with its gate area); load one ounce and one cubic inch over; air for 0, 1, 3, 7 creatures; weight variants. |
| FR-066, FR-067 | application | Same results with no-op and recording `telemetrytest` double; one span and one counter increment per call; `srd5e.outcome` only `success` or `failure`; no log. |
| FR-068 to FR-070 | architecture | Closed event family and exact members; exported-name scan for names and prefixes; no `combat` import from package `character`; no use case mentions another. |
| errors-and-results.md FR-031, FR-034, FR-051 | application | A failing roller is returned wrapped as a `*srderr.PortError` with the cause preserved (FR-022, FR-037, FR-048); a recorded failure sets `error.type` from `srderr.ErrorType(err)`; a roller that panics is recovered by `srderr.Guard` into a `*srderr.PortError`. |

- **Doubles and contracts:** `dicetest.FixedRoller` and the recording `telemetrytest` double; no contract suite (no port added).
- **Scenario test** (`scenarios/`, package `scenarios_test`, public API only): a character with a Belt of Giant Strength (storm) and Ring plus Cloak of Protection yields Strength 29 and AC and saves +2 through `derived-stats.md`; a Wand of Magic Missiles emptied to 0 charges rolls its last-charge check; (the AC and saves part is blocked on amendment A-3); a Bag of Holding placed in a Portable Hole destroys both.
- **Mutation testing:** the magic item files of package `character`, run with gomu only inside Docker with `--cpus` and `--workers` capped at 1/4 of `nproc` (`doc/rules/testing.md` section 8; never run locally). Likeliest survivors: `max(base, score)` and the cap (`<` against `<=`), load limits (`>` against `>=`), `lowestLevel + c - 1`, last-charge faces 1 and 20, ring capacity 5, half rounded down.
- **Not covered:** none.

## 9. Open questions and decisions

None escalated. Settled, one line each (the SRD is the authority; readings marked "interpretation" are justified in SRD or package-design terms):

- **Q1** (one spec or three): one; the three parts total under 600 lines and share the item-effect vocabulary and use cases.
- **Q2** (effect table in `content` or `character`): `character`. Only `character` needs these types (`shared-types-and-ownership.md` principle 2), and no content rule consumes a per-item effect table, so moving it to `content` would add a type there that nothing in `content` reads. Same treatment as the constants of `content-magic-items.md` section 4.8 (code constants, no wire shape, no data file).
- **Q3** (do bonuses stack): interpretation. The SRD limits only Proficiency Bonus ("The Bonus Doesn't Stack", p. 8) and copies of one attuned item (p. 102), so effects from different items add (Ring and Cloak of Protection stack).
- **Q4** (Belt and Gauntlets together): interpretation. Each item sets the score and has no effect at or above it, so the highest governs.
- **Q5** (Might of Giants): interpretation. "The Strength score bestowed" needs a bestowing item, so an ineffective Belt or Gauntlets bestows nothing; it applies after sets (FR-018) and is capped at 30.
- **Q6** (Resistance type chosen by the GM): interpretation. The three items list ten types; the SRD names no other, so a chosen type is one of them.
- **Q7** (items with a last-charge sentence not yet in the table): they have `NoLastChargeEffect` until a row is added; no wand is guessed.
- **Q8** (cantrip into the ring): interpretation. "Level 1 through 5" excludes level 0.
- **Q9** (nesting of same-kind pairs and which gate area applies): interpretation. Each text names the other two kinds plus "similar item", so Bag in Bag, Haversack in Haversack and Hole in Hole are covered by "similar item" (a Hole in a Hole has no explicit wording; the reasonable reading is that identical items are similar); the placed item's own sentence governs the area (FR-062). A different reading changes only `GateArea`, which is data on the item kind.
- **Q10** (loads in ounces and cubic inches): package design, so capacity checks are exact integer comparisons.
- **Q11** (sets before raises): interpretation. The SRD gives no order. A "set" item replaces the score it acts on and a raise (Belt of Dwarvenkind, Might of Giants) adds to the resulting score; Con 12 with Amulet (19) and Belt of Dwarvenkind gives 20, whereas raising first gives 19.
- **Q12** (Death Saving Throws and "saving throws" bonuses): interpretation. A Death Saving Throw is a saving throw without an ability (SRD 5.2.1, "Death Saving Throws", p. 17), so whether a flat bonus joins it is the host's choice; `savingThrows` is a plain list and the spec does not decide.

**Amendments** (other specs are not edited here), all landed except the signals catalogue: A-1 rank 14 in `shared-types-and-ownership.md` section 4.11, so FR-056 holds; A-2 `magic-item-rules.md` FR-025 allows `castAtLevel` above the lowest when the description names it, so FR-034 holds; A-3 `derived-stats.md` FR-079 (`flatSavingThrowBonuses`) and FR-038 (`FlatBonusSource`), so the saves half of the scenario test holds; A-4 `documentation_index.md` and `roadmap.md` list the spec, while the `doc/observability/signals.md` entries for section 7 are written with the implementation (as for other planned specs); A-5 `item-use-effects.md` FR-040 takes `PotionOfHealingGrade.Standard.dice` and lists this spec in Depends on, so there is one roll path (FR-025).

## 10. Changelog impact

`CHANGELOG.Md`, under `Unreleased`:

- **Added:** magic item effects in `character`: enhancement bonuses for Armor, Shield, Weapon, Ammunition and the Wand of the War Mage, fixed-bonus items, set-to-score and capped ability-score items, Potion of Healing grades and Potion of Resistance, charged spell items (charge-to-level costs, last-charge roll, Staff of the Magi absorption and Retributive Strike), Ring of Spell Storing, and extradimensional container capacity and nesting destruction. Three use cases (`RollPotionOfHealing` for the greater to supreme grades, `ResolveLastCharge`, `RollSpellStoringRingLevels`) and the signals of section 7. Not **BREAKING**: everything is new.
