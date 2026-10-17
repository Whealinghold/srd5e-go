# Equipment rules (rule maths for equipment)

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | Types 3; bulk data 5 |
| Depends on | `content-equipment.md`, `content-schema-and-decoder.md`, `content-spells.md`, `ability-scores.md`, `modifiers-and-advantage.md` |


## 1. Purpose and scope

[content-equipment.md](content-equipment.md) put the rule maths that interprets
equipment (Armor Class from the Armor table, the speed penalty for a missing
Strength score, drawn loads, barding, ship repair, crafting, scribing and
spellcasting fees) in `content` (planned `content/domain` files). `doc/rules/architecture.md` §2 says
`content` carries static SRD facts only, so this spec owns those rules in
`character` (package `character`, files `equipment_*.go`, Q3) and adds Carrying Capacity (p. 178). It is a leaf spec within `character`: it imports
no other `character` spec (FR-070). It is rank 9 in shared-types-and-ownership.md 4.11, below `derived-stats.md` (rank 10), which lists it (Q18).
Entry types, bundled data and the decoder stay in `content` (entry types ship in
phase 2, shared-types-and-ownership.md FR-014; FR-064, FR-065, FR-068 wait for
the phase 5 bulk data). All services are pure: stateless zero-size service types and immutable
value objects over loaded content values and plain numbers; no use case, port
or I/O.

**Content types used** (entry types only): `Armor`, `ArmorCategory`, `BaseRule`
and its closed Dexterity treatment (content-equipment.md FR-093), `Mount`, `Tool`
with its Craft list (`CraftReference`, the weapon and armor sets and their
`except`), `SpellScrollCost`, `SpellcastingService`, `Cost`, `Coin`, `Weight`,
`WeightSpec`, `Availability`, `SettlementSize`, `ItemID`, `CreatureSize`,
`SpellLevel`, `Ability`, `AbilityScore`, `AbilityModifier`, `RollCircumstance`.
Consumers, not dependencies: `derived-stats.md` (`ArmorRules.BaseArmorClass`,
`ArmorBase`, the Armor Training output, `StealthCircumstances`, FR-079),
`movement-lifting-and-flight.md` (`LimitFeet`, FR-019; `SpeedPenaltyFeet`, Q20), `magic-item-rules.md`
(`Crafting.RequirementsFor`), `exploration.md` (`CarryingCapacity`),
`casting-and-concentration.md` (`untrainedArmorWorn`, FR-075; spellcasting.md lists this spec for it).

**Relocated from content-equipment.md** (shared-types-and-ownership.md FR-048): the
old-id to new-id table is that spec's withdrawn-ids table and Q18; those ids are
not ids of this spec (the FR-083 row ending section 4 is for the checker). Not
relocated: `Coin`, `Cost`, `ArmorCategory` and donning times, Craft lists, the
Potion of Healing override, `SettlementSize`, `Availability` and all data.
Superseded: the Armor Class evaluation of `derived-stats.md` and the Carrying
Capacity table of `movement-lifting-and-flight.md` (its Q1).

**Out of scope:**

- Applying the armor speed penalty to a Speed (floor 0): `movement-lifting-and-flight.md`, through `MovementSpeeds.AdjustedBy(-penalty)` (Q20).
- Applying the 5-foot drag, lift or push limit to a Speed:
  `movement-lifting-and-flight.md` FR-001, FR-002, FR-005 (Q19).
- Saddles rule, p. 100: `mounted-and-underwater-combat.md`.
- Tool Proficiency, Utilize DCs (p. 93), helpers' proficiency and crafting
  prerequisites with assisted time (p. 103): `tool-proficiency-and-crafting.md`
  (Q13). Scribing prerequisites and scribed-scroll stats: `spell-scroll-use.md`
  FR-023 to FR-027 (assisted time stays out of scope).
- Item-use effects in gear descriptions, pp. 94-99, none owned here:
  `item-use-effects.md` (thrown gear, Oil coat, Basic Poison p. 99, Antitoxin
  pp. 95-96 reusing the coating idea and `poisonedAtStake` of `toolbox-poison.md`
  FR-019, Q-19; Potion of Healing; Healer's Kit); `deployed-and-restraining-gear.md`
  (Caltrops, Ball Bearings, Oil on a space, Hunting Trap, Net, Chain, Manacles, Rope).
- Applying Armor Training: `derived-stats.md` FR-039, FR-035 (Shield AC loss), `casting-and-concentration.md` FR-063 (via FR-075).
- Resolving a Dexterity (Stealth) check: the consumer applies the circumstance of FR-079 (Q20).
- Armor Class basis, Shield bonus, flat bonuses: `derived-stats.md`.
- Coins and trade, deliberately (Q23): make-change, paying a Cost, Selling Equipment (p. 89), coin weight (50 coins per pound, p. 89), inventory totals; magic items (pp. 102-103); using a Spell Scroll (p. 99).
- Vehicle rules (Large Vehicles Speed, Crew, Damage Threshold, berthing, pp. 100-101);
  Barding's Armor Class and which mount may wear it (p. 100, not in the SRD);
  Hireling, Lifestyle and Food, Drink, and Lodging totals and the length of a
  week or month (consumer multiplication); encumbrance beyond the 5-foot limit.

## 2. Source

SRD 5.2.1 pp. 8 ("Advantage/Disadvantage"), 13 ("Initiative"), 89 ("Coins"), 92
("Armor", Armor table, "Strength", "Armor Training", "One at a Time"), 93-94
("Tools", Craft), 99 ("Potion of Healing", "Spell Scroll"), 100 ("Mounts and
Cargo", "Barding", Mounts and Tack tables), 101 ("Ship Repair", Vehicles table),
102 ("Spellcasting", Spellcasting Services table), 103 ("Crafting Nonmagical
Items", "Brewing Potions of Healing", "Scribing Spell Scrolls"), 177 ("Armor
Class", "Armor Training"), 178 ("Carrying Capacity"). Printed page equals PDF
page. Package-design rules: `doc/rules/architecture.md`, `observability.md`,
`testing.md` (sections cited inline).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Armor Class (AC), "base AC" | `ArmorBase` | Base AC of a body armor for a Dexterity modifier, with its parts (p. 92). Not `ArmorClass` (`d20`) nor `ArmorClassBasis` (`derived-stats.md`). |
| Armor (Light, Medium, Heavy) | `Armor`, `ArmorClassRule`, `BaseRule` | content-equipment.md (FR-093); used, not redefined. Only a `BaseRule` is evaluated here. |
| Armor Training | `ArmorTrainingRules`, `ArmorTrainingEffects` | Collection of `ArmorCategory` trained against the set worn (pp. 92, 177). Plain `[]ArmorCategory` inputs treated as sets, so no `character` import (FR-070). |
| Disadvantage on a D20 Test that involves Strength or Dexterity | `RollCircumstance` (`Disadvantage`) | Returned by `RollCircumstancesFor` (FR-072); applied by the consumer. |
| One at a Time | an `srderr.ErrInvalidInput` error in `Assess` (FR-074) | "only one suit of armor at a time and ... only one Shield at a time" (p. 92). |
| Strength, Stealth (Armor table columns) | `Armor.StrengthRequirement`, `Armor.StealthDisadvantage` | Content-owned; used by `SpeedPenaltyFeet` and `StealthCircumstances`. |
| Carrying Capacity, Carry, Drag/Lift/Push | `CarryingCapacity` | From size and Strength score (p. 178). |
| Creature Size | `CreatureSize` | `content-schema-and-decoder.md` (FR-109; shared-types-and-ownership.md FR-005, FR-046); initiative-and-actions.md FR-092 forbids a second definition. Small and Medium share a row. |
| Speed rule of the Carrying Capacity entry | `LoadAssessment` (`LoadWithinCarry`, `LoadSpeedLimited`, `LoadBeyondDragLiftPush`) | `LoadSpeedLimited` carries the 5-foot limit only; applying it is `LoadSpeedRules.Limit`. |
| Mounts and Cargo; Mounts and Other Animals, Carrying Capacity | `DrawnLoad`; `Mount.CarryingCapacityPounds` (content-owned) | Five times the animals' base carrying capacity, vehicle included (p. 100). |
| Barding | `Barding`, `BardingOutcome` (`BardingDerived`, `BardingWeightUnknown`) | Four times the cost, twice the weight (p. 100). |
| Ship Repair | `ShipRepair`, `ShipRepairOutcome` (`ShipRepairQuoted`, `ShipRepairExceedsShip`), `SuppliesAndLabor` (`notAbundant`, `abundant`) | 1 Hit Point per day and 20 GP; halved "where supplies and skilled labor are abundant, such as a city shipyard" (p. 101). |
| Crafting Nonmagical Items | `Crafting`, `CraftingOutcome` (`CraftingRequirements`, `NotCraftable`), `NotCraftableReason` | Tools, Raw Materials, Time (p. 103); `days` counts "working 8 hours a day", need not be consecutive. The Copper Piece rounding unit and the fractional-GP reading of Time are package-design interpretations of a silent SRD (Q15); the Potion of Healing `craftingOverride` is content data (FR-041). |
| Scribing Spell Scrolls, Spell Scroll Costs; Spellcasting Services | `ScrollScribing`; `SpellcastingServices` | Ten `SpellScrollCost` rows (p. 103); seven `SpellcastingService` rows (p. 102); the class of that name in content-equipment.md is withdrawn (content-equipment.md Q18). |
| (package term) table coverage | `TableCoverageError` | Whether decoded rows cover each spell level exactly once. |

## 4. Requirements

Stable ids. Costs are compared by `valueInCopper` unless a row says the printed
form is kept. A "load" is whole pounds (Q11). **Case tables** (testing.md
sections 5, 6): each `<br>`-separated entry is one table-driven test; label =
entry text, requirement = the FR ids and page in the heading (or the Group
cell), path = the Path cell (a two-path cell tags each entry inline). Inside an
entry `;` and `:` are part of its format.

### Armor Class from a body armor

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `ArmorRules.BaseArmorClass(BaseRule rule, AbilityModifier dexterity)` MUST, for a full Dexterity treatment (Light), return the listed base plus the whole modifier, with no cap and a negative modifier lowering the result. All Light rows of "Armor base" MUST hold. | SRD 5.2.1, "Armor", Armor table, p. 92; "Armor Class", p. 177 | happy |
| FR-002 | For a capped treatment (Medium, maximum 2) the contribution MUST be the modifier limited to the maximum from above only: at or below it applies in full, above it gives the maximum, a negative modifier is never raised. All Medium rows MUST hold. | SRD 5.2.1, Armor table ("Dex modifier (max 2)"), p. 92 | unhappy (boundary) |
| FR-003 | For no Dexterity treatment (Heavy) the result MUST be the listed base for every modifier from -5 to +10. All Heavy rows MUST hold. | SRD 5.2.1, Armor table, p. 92 | unhappy (rule interaction: Dexterity ignored) |
| FR-004 | `BaseArmorClass` MUST return an `ArmorBase` with `listedBase`, the signed `dexterityContribution` (0 for none) and `total` equal to their sum; equal fields MUST be equal under `==`. | SRD 5.2.1, "Armor", p. 92 ("your base AC is 11 plus your Dexterity modifier"); architecture.md section 6 | happy |
| FR-005 | `ArmorRules` MUST offer no evaluation taking an `Armor` or `ArmorClassRule`, only a `BaseRule`, so a Shield (a `BonusRule`) cannot be evaluated as a base AC: unrepresentable, not thrown. The closed `ArmorClassRule` has exactly `BaseRule` and `BonusRule`; a `BaseRule` carries a closed treatment `FullDexterity`, `CappedDexterity(maximum)` or `NoDexterity` (content-equipment.md FR-093), switched over exhaustively with no default. | SRD 5.2.1, Armor table (Shield, "+2"), p. 92; content-equipment.md FR-024 | unhappy (rule interaction) |
| FR-006 | A capped rule with maximum 0 MUST contribute 0 for a positive modifier and the modifier itself for a negative one. | SRD 5.2.1, Armor table ("Dex modifier (max 2)"), p. 92 (capping at its lowest maximum, on a synthetic rule) | unhappy (boundary) |

### Speed penalty and Stealth Disadvantage

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-007 | `ArmorRules.SpeedPenaltyFeet(Armor armor, AbilityScore strength)` MUST return 10 when the armor has a Strength requirement and the score is lower than it. | SRD 5.2.1, "Armor", Strength, p. 92 | unhappy (boundary) |
| FR-008 | It MUST return 0 when the score equals or exceeds the requirement. | SRD 5.2.1, "Armor", Strength, p. 92 | unhappy (boundary) |
| FR-009 | It MUST return 0 for every score 1 to 30 when the armor has no requirement. | SRD 5.2.1, Armor table (Strength "—"), p. 92 | unhappy (empty and minimal) |
| FR-010 | For a requirement `r` the penalty MUST be 10 for scores 1 to `r - 1` and 0 for `r` to 30 (a sweep for requirements 13 and 15). | SRD 5.2.1, "Armor", Strength, p. 92 | unhappy (boundary) |
| FR-079 | `ArmorRules.StealthCircumstances(worn []Armor)` MUST return a one-element fresh slice holding a `Disadvantage` when any worn armor has `stealthDisadvantage`, else a nil slice; never more than one element (Disadvantage does not stack). The consumer applies it to Dexterity (Stealth) checks only. It does not depend on training: the SRD attaches it to "the wearer" by the Stealth column, not to Armor Training (**interpretation**, p. 92 states no training condition). A Shield and an empty `worn` give the empty list. No check is resolved here. | SRD 5.2.1, "Armor", Stealth, p. 92; "They Don't Stack", p. 8 | unhappy (rule interaction) |

### Armor Training

Added by shared-types-and-ownership.md FR-060. Inputs are plain
`[]ArmorCategory` (treated as a set: order and duplicates are irrelevant) (`light`, `medium`, `heavy`, `shield`; content-equipment.md
FR-104), so no `character` spec is imported (FR-070).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-071 | `ArmorTrainingRules.Assess(worn []ArmorCategory, trained []ArmorCategory)` MUST return an `ArmorTrainingEffects` whose `Untrained` holds exactly the categories in `worn` not in `trained` (a Light, Medium or Heavy armor "and lack training with it", or a Shield used untrained). A trained or unworn category MUST NOT appear. All "Armor Training" rows MUST hold. | SRD 5.2.1, "Armor Training", p. 177; "Armor", p. 92 ("only those with training can use them effectively") | happy |
| FR-072 | `ArmorTrainingEffects.RollCircumstancesFor(involved []ability.Ability)` MUST return exactly one `Disadvantage` when `Untrained` contains `light`, `medium` or `heavy` and `involved` contains `Ability.Strength` or `Ability.Dexterity` ("any D20 Test that involves Strength or Dexterity": an ability check, saving throw or attack roll using either, and Initiative, a Dexterity check), and an empty list otherwise; never more than one element. A test with no ability (a death saving throw) or only other abilities MUST give the empty list. | SRD 5.2.1, "Armor Training", p. 177; "Light, Medium, or Heavy Armor", p. 92; "Combat Step by Step", p. 13 | unhappy (rule interaction) |
| FR-073 | A Shield alone in `Untrained` MUST give the empty list for every `involved` set: its only consequence is losing its AC bonus (`derived-stats.md` FR-035), reported through `Untrained`. | SRD 5.2.1, "Shield", p. 92; "Armor Training", p. 177 | unhappy (rule interaction) |
| FR-074 | `Assess` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when `worn` holds more than one of `light`, `medium`, `heavy` ("only one suit of armor at a time"); a `Set` holds a Shield once ("only one Shield at a time"). An empty `worn` MUST be valid with an empty `Untrained`. | SRD 5.2.1, "One at a Time", p. 92; architecture.md section 6 | unhappy (invalid input) |
| FR-075 | `WearsUntrainedBodyArmor` MUST be true exactly when `Untrained` contains `light`, `medium` or `heavy`, false for an untrained Shield alone. It is the fact `casting-and-concentration.md` FR-063 reads (`untrainedArmorWorn`); no spellcasting rule is applied here. | SRD 5.2.1, "Armor Training", p. 177 | unhappy (rule interaction) |
| FR-076 | `ArmorTrainingRules.MonsterTraining(statBlockArmor []ArmorCategory)` MUST return `statBlockArmor` ("A monster has training with any armor in its stat block"); `Assess(A, MonsterTraining(A))` MUST have an empty `Untrained` and an empty list for every `involved` set. | SRD 5.2.1, "Armor", p. 92 | happy |
| FR-077 | The list from `RollCircumstancesFor` MUST be accepted by `modifier.Resolve` (`modifiers-and-advantage.md` FR-042) with other circumstances: `Disadvantage` alone gives `disadvantage`, with one `Advantage` it gives `normal` (`modifiers-and-advantage.md` FR-015). No D20 Test is resolved and no source recorded here. | SRD 5.2.1, "Advantage/Disadvantage", "They Don't Stack", p. 8; "Armor Training", p. 177 | unhappy (rule interaction) |
| FR-078 | `ArmorTrainingEffects` MUST be an immutable value object, equal (under `Equal`) when `Untrained()` is equal. `Untrained()` MUST return a fresh slice (mutation changes nothing); `Assess` MUST copy its inputs, MUST NOT alter them, and MUST NOT depend on their iteration order. | architecture.md section 3; testing.md section 4 | unhappy (immutability) |

### Carrying Capacity

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-011 | `NewCarryingCapacity(size content.CreatureSize, strength ability.Score)` MUST give a Carry maximum of Strength times 7.5 (Tiny), 15 (Small, Medium), 30 (Large), 60 (Huge), 120 (Gargantuan) pounds, exposed as `CarryHalfPounds` (an `int`, so Tiny's 7.5 is exact). All "Carrying Capacity" rows MUST hold. | SRD 5.2.1, "Carrying Capacity", p. 178 | happy |
| FR-012 | It MUST give a Drag/Lift/Push maximum of Strength times 15 (Tiny), 30 (Small, Medium), 60 (Large), 120 (Huge), 240 (Gargantuan) pounds as `DragLiftPushPounds`. | SRD 5.2.1, "Carrying Capacity", p. 178 | happy |
| FR-013 | Small and Medium MUST give equal capacities for every score. | SRD 5.2.1, Carrying Capacity table ("Small/Medium"), p. 178 | unhappy (rule interaction) |
| FR-014 | A Tiny creature's Carry maximum for an odd score MUST end in a half pound exactly (Strength 3 gives 22.5 lb., `CarryHalfPounds` 45), tested as the integer of each odd-score row. | SRD 5.2.1, Carrying Capacity table (Tiny, "Str. x 7.5 lb."), p. 178 | unhappy (boundary) |
| FR-015 | `CanCarry(loadPounds int)` MUST be true iff the load does not exceed the Carry maximum (equal true, one pound above false; Tiny compared in half pounds). | SRD 5.2.1, "Carrying Capacity", p. 178 | unhappy (boundary) |
| FR-016 | `Assess(loadPounds int)` MUST return `LoadWithinCarry` for a load not exceeding the Carry maximum. | SRD 5.2.1, "Carrying Capacity", p. 178 | happy |
| FR-017 | `Assess` MUST return `LoadSpeedLimited` with `LimitFeet` 5 for a load above Carry and not above Drag/Lift/Push. | SRD 5.2.1, "Carrying Capacity" ("your Speed can be no more than 5 feet"), p. 178 | unhappy (boundary) |
| FR-018 | `Assess` MUST return `LoadBeyondDragLiftPush` for a load above the Drag/Lift/Push maximum (Q10). | SRD 5.2.1, Carrying Capacity table, p. 178 (the maximum only); package design (Q10: the state and variant are interpretation) | unhappy (boundary) |
| FR-019 | `LoadSpeedLimited` MUST expose `LimitFeet` (always 5) and no method applying it to a Speed. The arithmetic is only `LoadSpeedRules.Limit` (movement-lifting-and-flight.md FR-001, FR-002, FR-005); none is duplicated here (Q19). | SRD 5.2.1, "Carrying Capacity", p. 178 | unhappy (boundary) |
| FR-020 | `CanCarry` and `Assess` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a negative load; 0 MUST be `LoadWithinCarry`. A load is a caller-computed number, so a negative one is a programmer error (section 5; Q16). | SRD 5.2.1, "Carrying Capacity", p. 178; architecture.md section 6 | unhappy (invalid input) |
| FR-084 | `NewCarryingCapacity` MUST take the size used for capacity as given and read no species type (FR-070). For a Goliath the host passes `SpeciesPassives.CarryingCapacitySize(currentSize)` (`species-trait-behaviour.md` FR-013, Powerful Build): one `CreatureSize` larger than the current size, Gargantuan staying Gargantuan, so a Medium Goliath uses the Large row and one in Large Form the Huge row, for Carry and Drag/Lift/Push alike. Interpretation: "carrying capacity" in the trait covers the whole table. | SRD 5.2.1, "Powerful Build", p. 86; "Carrying Capacity", p. 178 | unhappy (rule interaction) |

### Drawn load

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-021 | `DrawnLoad.LimitPounds(animals)` MUST return five times the sum of the animals' carrying capacities. All "Drawn load" rows MUST hold. | SRD 5.2.1, "Mounts and Cargo", p. 100 | happy |
| FR-022 | `DrawnLoad.CanMove(animals, vehiclePounds, cargoPounds)` MUST be true iff `vehiclePounds + cargoPounds` does not exceed the limit (equal true, one over false). | SRD 5.2.1, "Mounts and Cargo" ("including the weight of the vehicle"), p. 100 | unhappy (boundary) |
| FR-023 | With no animals the limit MUST be 0 and `CanMove` true only when vehicle and cargo weigh 0. | SRD 5.2.1, "Mounts and Cargo", p. 100 | unhappy (empty and minimal) |
| FR-024 | A negative `vehiclePounds` or `cargoPounds` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. | architecture.md section 6 | unhappy (invalid input) |
| FR-025 | The limit MUST NOT depend on animal order, MUST be repeatable and MUST NOT alter the iterable passed in. | architecture.md section 3; testing.md section 4 | unhappy (immutability) |

### Barding

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-026 | `DeriveBarding(Armor armor)` MUST return `BardingDerived` holding a `Barding` costing four times the armor's cost in the same Coin and weighing twice its weight, for every armor on the Armor table including the Shield (Q9). All "Barding" rows MUST hold. | SRD 5.2.1, "Barding", p. 100 | happy |
| FR-027 | An armor whose weight is `NoWeight` or `WeightVaries` MUST give `BardingWeightUnknown` carrying the armor's id: no throw, no guessed weight. | SRD 5.2.1, "Barding" ("twice as much"), p. 100; architecture.md section 6 | unhappy (missing content) |
| FR-028 | `Barding` MUST be a value object (comparable with `==`: equal cost and weight) keeping the printed cost form (4 x 1,500 GP is 6,000 GP, not 600,000 CP). | architecture.md section 3; SRD 5.2.1, "Barding", p. 100 | unhappy (rule interaction) |

### Ship repair

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-029 | `QuoteShipRepair(hitPoints int, supplies SuppliesAndLabor)` with `notAbundant` MUST return 20 GP and one day per Hit Point, time in `HalfDays` (2 per Hit Point). All "Ship repair" rows MUST hold. | SRD 5.2.1, "Ship Repair" ("1 day and costs 20 GP"), p. 101 | happy |
| FR-030 | With `abundant` cost and time MUST both be halved: 10 GP and 1 `halfDay` per Hit Point, so halving never rounds. | SRD 5.2.1, "Ship Repair" ("the repair time and cost are halved"), p. 101 | unhappy (boundary) |
| FR-031 | `hitPoints` below 1 MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. | SRD 5.2.1, "Ship Repair" ("Repairing 1 Hit Point of damage"), p. 101 | unhappy (invalid input) |
| FR-032 | `QuoteShipRepairFor(ship content.AirborneOrWaterborneVehicle, hitPoints int, supplies SuppliesAndLabor)` MUST return `ShipRepairQuoted` (the `ShipRepair` of FR-029/FR-030) for `hitPoints` up to and including the ship's Hit Points, and `ShipRepairExceedsShip` (requested and ship Hit Points) above them; below 1 is FR-031. **Package design:** the SRD states no upper bound (the damage taken is unknown here), so the ship's Hit Points from the Vehicles table bound it; a caller who knows the real damage uses `ShipRepair.Of`. | Package design; SRD 5.2.1, Airborne and Waterborne Vehicles table (HP), p. 101 (the figure only) | unhappy (boundary) |

### Crafting Nonmagical Items

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-033 | `Crafting.RawMaterials(Cost cost)` MUST return half the cost in Copper Pieces, rounded down, as a Copper Piece `Cost` (1,500 GP gives 750 GP; 15 GP gives 7 GP 5 SP; 1 CP gives 0 CP; 5 CP gives 2 CP). The SRD prints "half its purchase cost (round down)" with no denomination; rounding in Copper Pieces (the smallest denomination, the unit of `valueInCopper`) is package design (Q15), and the rows depending on it are marked so. All "Crafting" rows MUST hold. | SRD 5.2.1, "Crafting Nonmagical Items", Raw Materials, p. 103 (halving, 1,500 GP example); package design (Q15) | unhappy (boundary) |
| FR-034 | `RawMaterials` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a cost with a rate (`per` Day or Mile) and return 0 Copper Pieces for a cost of 0 (Q7). | SRD 5.2.1, "Raw Materials" ("purchase cost"), p. 103; architecture.md section 6 | unhappy (invalid input) |
| FR-035 | `Crafting.Days(Cost cost)` MUST return the cost in GP divided by 10, rounded up to a whole day: the ceiling of `valueInCopper` / 1,000 (10 GP 1 day; 10 GP 1 CP 2; 15 GP 2; 45 GP 5; 20 GP 2; 1 CP 1). Silver and Copper Pieces are read as exact fractions of a GP, not truncated; the 10 GP 1 CP and 1 CP rows are package design (Q15). | SRD 5.2.1, "Crafting Nonmagical Items", Time ("round a fraction up to a day"), p. 103; package design (Q15) | unhappy (boundary) |
| FR-036 | `days` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a cost of 0 and for a cost with a rate. | architecture.md section 6; SRD 5.2.1, "Time" ("purchase cost"), p. 103 | unhappy (invalid input) |
| FR-037 | `Crafting.ToolsFor(item EquipmentEntry, tools []Tool)` MUST return the `ItemID` of every tool whose Craft list names the item by id. For a `FocusVariant` or `AmmunitionVariant` it MUST also return tools naming the variant's family (`variantOf`: Arcane Focus, Crystal matches "Arcane Focus"). A `ScrollVariant` is not matched through its family (scribing has its own path, FR-042; its tool prerequisites are Q13's). A `ToolVariant` has no Craft list and matches nothing (content-equipment.md FR-030, FR-033). | SRD 5.2.1, "Tools" (Craft), pp. 93-94 (Jeweler's Tools: "Arcane Focus, Holy Symbol"; Woodcarver's Tools: "Arcane Focus ... Druidic Focus"); "Crafting Nonmagical Items", Tools, p. 103 | happy |
| FR-038 | It MUST also return every tool whose Craft list names a set including the item (`MeleeWeapons` or `RangedWeapons` by `weaponType`, `MediumArmor` or `HeavyArmor` by armor category) unless the set's `except` lists the item's id; a set never matches an entry that is not a weapon or armor of that group. | SRD 5.2.1, "Smith's Tools", "Woodcarver's Tools" (Craft), pp. 93-94 | unhappy (rule interaction) |
| FR-039 | The result MUST be ordered by id (ordinal), hold each id once whatever the input order, and be empty when no tool names the item (the GM assigns the tools). The ids are **alternatives**: any one tool of the list suffices (Q8). | SRD 5.2.1, "Tools" ("The GM assigns required tools for items not listed there"), p. 103 | unhappy (empty and minimal) |
| FR-040 | `Crafting.RequirementsFor(item EquipmentEntry, tools []Tool, scribing ScrollScribing)` for an item with a `FixedCost` without a rate, amount above 0 and no override MUST return `CraftingRequirements` with `RawMaterials(cost)`, `days(cost)`, `ToolsFor(item, tools)` and `basis` `CraftingBasis.GenericRule`. All "generic" rows MUST hold. | SRD 5.2.1, "Crafting Nonmagical Items", p. 103 | happy |
| FR-041 | For an item with a `craftingOverride` (a `MagicGear`), it MUST return the override's raw materials, days and tools with `basis` `itemOverride`, not FR-033 or FR-035 applied to the price (Potion of Healing: 25 GP, 1 day, Herbalism Kit; the generic rule would give 25 GP and 5 days). | SRD 5.2.1, "Brewing Potions of Healing", p. 103; "Potion of Healing", p. 99 | unhappy (rule interaction) |
| FR-042 | For a `ScrollVariant` it MUST return the `scribing` row for the scroll's `SpellLevel`: raw materials the row's cost, days the row's days, no tools, `basis` `scrollRow` (Spell Scroll (Cantrip) sells for 30 GP; scribing is 15 GP and 1 day, where the generic rule would give 15 GP and 3 days). | SRD 5.2.1, "Scribing Spell Scrolls", Spell Scroll Costs, p. 103; "Spell Scroll", p. 99 | unhappy (rule interaction) |
| FR-043 | For a `ScrollVariant` with no `scribing` it MUST return `NotCraftable` with reason `scribingTableMissing`: no throw, not the generic rule. | architecture.md section 6; SRD 5.2.1, "Scribing Spell Scrolls", p. 103 | unhappy (missing content) |
| FR-044 | It MUST return `NotCraftable(priceVaries)` for an entry priced `CostVaries` (tool family, gear family, Spell Scroll family). | SRD 5.2.1, Adventuring Gear table ("Varies"), p. 95; "Gaming Set", p. 94; Spell Scroll family: package design (content-equipment.md Q17: the SRD prints 30 GP and 50 GP, pp. 95, 99, never "Varies") | unhappy (empty and minimal) |
| FR-045 | It MUST return `NotCraftable(ratedPrice)` for a price with a rate (Feed, Stabling per day). | SRD 5.2.1, Tack, Harness, and Drawn Vehicles table, p. 100 | unhappy (boundary) |
| FR-046 | It MUST return `NotCraftable(freeOfCharge)` for goods whose `FixedCost` amount is 0. | architecture.md section 6; SRD 5.2.1, "Raw Materials", p. 103 | unhappy (boundary) |
| FR-047 | It MUST return `NotCraftable(notAnItem)` for a `Mount`, `LifestyleExpense`, `FoodDrinkOrLodging`, `Hireling`, `SpellcastingService` or `SpellScrollCost`: services, expenses and table rows are not goods. | SRD 5.2.1, "Crafting Nonmagical Items" ("To craft a nonmagical item"), p. 103 | unhappy (empty and minimal) |
| FR-048 | When several cases apply the order MUST be: `notAnItem` and `noCraftingRule` (by entry type, FR-047, FR-066), the override (FR-041), the scroll row (FR-042, FR-043), `priceVaries`, `ratedPrice`, `freeOfCharge`, then the generic rule (FR-040); Wretched (a lifestyle row at cost 0) is `notAnItem`, not `freeOfCharge`. | SRD 5.2.1, "Crafting Nonmagical Items", p. 103 | unhappy (rule interaction) |

### Scribing Spell Scrolls

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-049 | `ScrollScribing.ForSpellLevel(SpellLevel level)` MUST return the `SpellScrollCost` row for every level 0 to 9. All "Scroll scribing" rows MUST hold. | SRD 5.2.1, "Scribing Spell Scrolls", Spell Scroll Costs, p. 103 | happy |
| FR-050 | The `ScrollScribing` constructor MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` unless its rows cover each level 0 to 9 exactly once (a level outside 0 to 9 is impossible: `SpellLevel` rejects it, content-spells.md FR-009). | SRD 5.2.1, Spell Scroll Costs, p. 103; architecture.md section 6 | unhappy (invalid input) |
| FR-051 | `NewScrollScribing(rows)` MUST return the table and a nil error for a complete set and otherwise (including empty) a `*TableCoverageError` (satisfying `errors.Is(err, srderr.ErrInvalidInput)`) with the missing and repeated levels, each ordered by level, without panicking; `MustScrollScribing` panics with that error iff `NewScrollScribing` returns one. | architecture.md sections 5, 6; SRD 5.2.1, "Scribing Spell Scrolls", p. 103 | unhappy (missing content) |

### Spellcasting services

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-052 | `SpellcastingServices.ForSpellLevel(SpellLevel level)` MUST return the row whose range contains the level (Cantrip 30 GP, 1 50 GP, 2 200 GP, 3 300 GP, 4 to 5 2,000 GP, 6 to 8 20,000 GP, 9 100,000 GP). All "Spellcasting fee by level" rows MUST hold. | SRD 5.2.1, Spellcasting Services table, p. 102 | happy |
| FR-053 | The `SpellcastingServices` constructor MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` unless the ranges cover each level 0 to 9 exactly once (no gap, no overlap, `spellLevelMin` not above `spellLevelMax`). | SRD 5.2.1, Spellcasting Services table, p. 102; architecture.md section 6 | unhappy (invalid input) |
| FR-054 | `NewSpellcastingServices(rows)` MUST return the table and a nil error for a complete set and otherwise (including empty) a `*TableCoverageError` with the uncovered and overlapping levels, each ordered by level, without throwing. | architecture.md sections 5, 6; SRD 5.2.1, Spellcasting Services table, p. 102 | unhappy (missing content) |
| FR-055 | `SpellcastingServices.IsAvailable(SpellcastingService row, SettlementSize size)` MUST be true for Village, Town and City when Availability is "Village, town, or city", for Town and City when "Town or city only", and for City only when "City only". | SRD 5.2.1, Spellcasting Services table (Availability), p. 102 | unhappy (rule interaction) |
| FR-056 | `SpellcastingServices.TotalCost(SpellcastingService row, Cost componentCost)` MUST return the listed cost plus the expensive components' cost as a Copper Piece `Cost` (level 3 with 100 GP of components gives 400 GP; no components gives the listed cost). | SRD 5.2.1, "Spellcasting" ("add the cost of those components to the cost listed"), p. 102 | unhappy (rule interaction) |
| FR-057 | `TotalCost` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a `componentCost` with a rate. | architecture.md section 6; SRD 5.2.1, "Spellcasting", p. 102 | unhappy (invalid input) |

### Placement, purity and API

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-058 | Every type and function here MUST live in package `character` (files `equipment_*.go`), be synchronous and deterministic, import no `os`, `net`, `telemetry`, `context` or `*test` package, spawn no goroutine, and use no `math/rand`, `time.Now` or `time.Since`. | architecture.md sections 3, 9 | happy |
| FR-059 | The only module imports MUST be the kernel packages (`ability`, `modifier`, `srderr`) and the published API of `content`; it MUST NOT import `d20` internals, `combat`, `toolbox` or any context not upstream of `character` (shared-types-and-ownership.md FR-013). | architecture.md sections 1, 2, 9 | happy |
| FR-060 | This code MUST contain no `telemetry.Logger`, `Tracer`, `Meter` or signal constant. | observability.md section 3 | happy |
| FR-061 | The types of section 6 MUST be exported from package `character` (no re-export package), each with a godoc comment starting with its name, no exported name duplicating another in the package, and no exported signature containing a JSON key name or `map[string]any`. | architecture.md sections 2, 5, 7 | happy |
| FR-062 | Nothing in the code, tests or data of this spec MAY name a consuming project. | AGENTS.md ("Hard constraints") | happy |
| FR-063 | Every slice in a result (`CraftingRequirements.Tools`, the slices of the two `TableCoverageError` kinds) MUST be a copy: mutating a returned slice leaves the next read unchanged. | architecture.md section 4 | unhappy (immutability) |
| FR-064 | A consumer importing only public packages of the module MUST be able to load the bundled equipment, look up Leather Armor and compute its base AC for Dexterity +3 (14), and look up Plate Armor and compute its speed penalty for Strength 8 (10 feet). Phase 5 data (the thin slice has none, content-thin-slice.md FR-004); the maths is phase 3 and covered on synthetic entries by FR-001 to FR-010. | architecture.md section 7; testing.md section 3; SRD 5.2.1, pp. 92, 100, 103 | happy |
| FR-065 | Over the bundled data, `Crafting.ToolsFor` MUST satisfy every row of "Which tools craft an item" (phase 5 data). | SRD 5.2.1, "Crafting Nonmagical Items", Tools, p. 103; Craft entries, pp. 93-94 | unhappy (rule interaction) |
| FR-066 | `RequirementsFor` MUST treat the remaining entry types thus. An `AirborneOrWaterborneVehicle` gives `NotCraftable(noCraftingRule)` (the SRD has no vehicle-building rule; a Galley at 30,000 GP does not get the generic 15,000 GP and 3,000 days; Q17). A `TackHarnessOrDrawnVehicle` with a rate-free `FixedCost` above 0 (Cart, Saddle) follows the generic path with the tools of `ToolsFor` (empty: the GM assigns); with a rate (Feed, Stabling) it is `ratedPrice` (FR-045). A `ToolVariant` with a `FixedCost` above 0 (Gaming Set, dragonchess, 1 GP) follows the generic path from its own price with no tools. A `FocusVariant` and `AmmunitionVariant` follow the generic path with the tools of FR-037. | SRD 5.2.1, "Crafting Nonmagical Items", p. 103; Tack table, p. 100; Vehicles table, p. 101; package design (Q17) | unhappy (rule interaction) |
| FR-067 | `NewScrollScribing`, `NewSpellcastingServices` and `CraftingRequirements` MUST copy the slice they are given, so later mutation of the source changes nothing (lookups, equality). `IsAvailable` and `TotalCost` are package functions and use no table state. | architecture.md section 3; testing.md section 4 | unhappy (immutability) |
| FR-068 | Over the bundled data, a consumer importing only public packages MUST derive Plate Armor's barding (6,000 GP, 130 lb.) and compute Plate Armor's crafting requirements (750 GP, 150 days, `smiths_tools`) (phase 5 data; the armor part is FR-064). | architecture.md section 7; testing.md section 3; SRD 5.2.1, pp. 100, 103 | happy |
| FR-069 | No use case MAY call another use case to obtain a result of this spec. A use case needing one (`CalculateDerivedStats`, `PlanMagicItemCrafting`) MUST call these services and value objects directly with content loaded through its own `content.Source` port, recording only its own signals (shared-types-and-ownership.md FR-044). This spec defines no use case and no port. | shared-types-and-ownership.md FR-044; architecture.md section 4 | happy |
| FR-070 | Code in the `equipment_*.go` files MUST import no other file of another `character` spec, so no cycle arises whatever its in-context place (Q18). Its package imports are those of FR-059. | architecture.md section 9; shared-types-and-ownership.md FR-016 | unhappy (cycle) |

### Cases: Armor base (FR-001 to FR-004, FR-006; Armor table, p. 92)

Entry: rule, Dexterity: `listedBase` + contribution = `total`.

| Group | Entries | Path |
|---|---|---|
| Light, positive | Leather Armor (11, full), +3: 11 + 3 = 14 | happy |
| Light, zero | Leather Armor, 0: 11 + 0 = 11 | unhappy (boundary) |
| Light, negative, lowest, no cap | Padded Armor (11, full), -1: 11 - 1 = 10<br>Padded Armor, -5: 11 - 5 = 6<br>Studded Leather Armor (12, full), +5: 12 + 5 = 17<br>Studded Leather Armor, +10: 12 + 10 = 22 | unhappy (boundary) |
| Medium (max 2) | Hide Armor (12), +1: 12 + 1 = 13<br>Hide Armor, +2: 12 + 2 = 14<br>Hide Armor, +3: 12 + 2 = 14<br>Half Plate Armor (15), +5: 15 + 2 = 17<br>Chain Shirt (13), 0: 13 + 0 = 13<br>Breastplate (14), -2: 14 - 2 = 12<br>Scale Mail (14), -5: 14 - 5 = 9 | unhappy (boundary) |
| Heavy (none) | Ring Mail (14), +5: 14 + 0 = 14<br>Chain Mail (16), -3: 16 + 0 = 16<br>Splint Armor (17), +10: 17 + 0 = 17 | unhappy (rule interaction) |
| Heavy, zero | Plate Armor (18), 0: 18 + 0 = 18 | happy |
| synthetic cap 0 (FR-006) | (10, max 0), +3: 10 + 0 = 10<br>(10, max 0), -1: 10 - 1 = 9 | unhappy (boundary) |

### Cases: Strength requirement and Stealth (FR-007 to FR-010, FR-079; Armor table, p. 92)

Entry: armor (Strength), score: penalty in feet.

| Group | Entries | Path |
|---|---|---|
| boundary | Chain Mail (13), 12: 10<br>Chain Mail (13), 13: 0<br>Chain Mail (13), 14: 0<br>Chain Mail (13), 1: 10<br>Chain Mail (13), 30: 0<br>Splint Armor (15), 14: 10<br>Splint Armor (15), 15: 0<br>Plate Armor (15), 8: 10<br>Plate Armor (15), 15: 0<br>Padded Armor (none), 30: 0 | unhappy (boundary) |
| no requirement | Ring Mail (none), 1: 0<br>Padded Armor (none), 1: 0 | unhappy (empty and minimal) |
| Stealth (FR-079; entry: worn armor: list, [D] = one `Disadvantage`) | Padded Armor: [D]<br>Plate Armor: [D]<br>Leather Armor: []<br>Chain Mail + Shield: [D]<br>Scale Mail + Plate Armor: [D] (one element)<br>Shield alone: []<br>no armor: [] | unhappy (rule interaction) |

### Cases: Armor Training (FR-071 to FR-078; "Armor Training", p. 177; Shield and One at a Time, p. 92; Initiative, p. 13; They Don't Stack, p. 8)

Entry: `worn` / `trained` / `involved`: `Untrained`; `RollCircumstancesFor`. H = `heavy`,
M = `medium`, L = `light`, S = `shield`; "[D]" is the one-element list holding a
`Disadvantage`, "[]" the empty list; Initiative is a Dexterity check.

| Group | Entries | Path |
|---|---|---|
| trained or untrained | `{H}` / `{L, M, S}` / `{strength}`: `{H}`; [D]<br>`{H}` / `{L, M, S}` / `{dexterity}`: `{H}`; [D]<br>`{H}` / `{H}` / `{strength}`: `{}`; [] | happy |
| Initiative and abilities | `{H}` / `{L, M, S}` / `{dexterity}` (Initiative): `{H}`; [D]<br>`{M}` / `{L}` / `{strength}`: `{M}`; [D]<br>`{L}` / `{}` / `{dexterity}`: `{L}`; [D]<br>`{H}` / `{L, M, S}` / `{constitution}`: `{H}`; []<br>`{H}` / `{L, M, S}` / `{charisma}` (attack roll): `{H}`; [] | unhappy (rule interaction) |
| minimal | `{H}` / `{L, M, S}` / `{}` (death saving throw): `{H}`; []<br>`{}` / `{}` / `{strength}`: `{}`; [] | unhappy (empty and minimal) |
| both abilities | `{H}` / `{}` / `{strength, dexterity}`: `{H}`; [D] (one element) | unhappy (boundary) |
| Shield (FR-073, FR-075) | `{S}` / `{}` / `{strength}`: `{S}`; []<br>`{S}` / `{}` / `{dexterity}`: `{S}`; [] and `WearsUntrainedBodyArmor` false (`casting-and-concentration.md` FR-063)<br>`{H, S}` / `{}` / `{strength}`: `{H, S}`; [D] and `WearsUntrainedBodyArmor` true<br>`{H, S}` / `{H}` / `{strength}`: `{S}`; []<br>`{H, S}` / `{S}` / `{dexterity}`: `{H}`; [D] | unhappy (rule interaction) |
| One at a Time (FR-074) | `{L, H}` / `{}`: an `srderr.ErrInvalidInput` error (unhappy: invalid input)<br>`{L, M, H}` / `{L, M, H}`: an `srderr.ErrInvalidInput` error (unhappy: boundary) | unhappy (invalid input, boundary) |
| monster (FR-076) | `{M}` / `MonsterTraining({M})` / `{strength}`: `{}`; [] (happy)<br>`{H, S}` / `MonsterTraining({H, S})` / `{strength, dexterity}`: `{}`; [] (unhappy: rule interaction) | happy, unhappy (rule interaction) |
| with Advantage (FR-077) | `{H}` / `{}` / `{strength}`: [D] with one `Advantage` resolves `normal`; alone `disadvantage` | unhappy (rule interaction) |
| immutability (FR-078) | `{H}` mutated after the call: result unchanged and passed sets keep their contents<br>mutating the slice from `Untrained()` leaves the next read unchanged | unhappy (immutability) |

### Cases: Carrying Capacity (FR-011 to FR-014; p. 178)

Entry: size, Strength: Carry lb. (`CarryHalfPounds`) / Drag/Lift/Push lb.

| Group | Entries | Path |
|---|---|---|
| happy | Tiny, 10: 75 (150) / 150<br>Small, 10: 150 (300) / 300<br>Medium, 10: 150 (300) / 300<br>Medium, 15: 225 (450) / 450<br>Large, 20: 600 (1200) / 1,200<br>Huge, 25: 1,500 (3000) / 3,000 | happy |
| boundary | Tiny, 1: 7.5 (15) / 15<br>Tiny, 3: 22.5 (45) / 45<br>Tiny, 15: 112.5 (225) / 225<br>Medium, 1: 15 (30) / 30<br>Gargantuan, 30: 3,600 (7200) / 7,200 | unhappy (boundary) |

### Cases: Load assessment (FR-015 to FR-020; p. 178)

Capacity Medium, Str 10 (carry 150, drag 300) unless stated; Tiny Str 15 has carry
112.5 and drag 225. Entry: load (lb.): `CanCarry`; `Assess`.

| Group | Entries | Path |
|---|---|---|
| empty | 0: true; `LoadWithinCarry` | unhappy (empty and minimal) |
| Medium boundaries | 150: true; `LoadWithinCarry`<br>151: false; `LoadSpeedLimited` (5)<br>300: false; `LoadSpeedLimited` (5)<br>301: false; `LoadBeyondDragLiftPush` | unhappy (boundary) |
| Tiny, Str 15 | 112: true; `LoadWithinCarry`<br>113 (over by a half pound): false; `LoadSpeedLimited` (5)<br>225: false; `LoadSpeedLimited` (5)<br>226: false; `LoadBeyondDragLiftPush` | unhappy (boundary) |
| negative | -1: an `srderr.ErrInvalidInput` error | unhappy (invalid input) |
| limit in feet (FR-019) | the `LoadSpeedLimited` of load 151: `LimitFeet` is 5 (Speed arithmetic: movement-lifting-and-flight.md FR-001, FR-002, FR-005) | unhappy (boundary) |

### Cases: Drawn load (FR-021 to FR-025; capacities from Mounts and Other Animals, vehicle weights from Tack, Harness, and Drawn Vehicles, p. 100)

Entry: animals (capacity), vehicle (lb.), cargo (lb.): limit; can move.

| Group | Entries | Path |
|---|---|---|
| boundary | Horse, Draft (540), Cart (200), 2,500: 2,700; yes<br>Horse, Draft, Cart, 2,501: 2,700; no<br>Elephant (1,320), Carriage (600), 6,000: 6,600; yes<br>Pony (225), Sled (300), 825: 1,125; yes<br>Mastiff (195), Sled (300), 675: 975; yes<br>Mastiff, Sled, 676: 975; no | unhappy (boundary) |
| animals add | Mule (420) x2, Wagon (400), 3,800: 4,200; yes<br>Horse, Riding (480) + Horse, Draft (540), Carriage (600), 4,500: 5,100; yes<br>Camel (450) x2 + Mule (420), Wagon (400), 5,200: 6,600; yes | unhappy (rule interaction) |
| other order | Mule (420), Camel (450), Camel (450), Wagon (400), 5,201: 6,600; yes | unhappy (immutability) |
| no animals | none, Cart (200), 0: 0; no<br>none, 0, 0: 0; yes | unhappy (empty and minimal) |
| negative (FR-024) | Mule, Cart, cargo -1: an `srderr.ErrInvalidInput` error<br>Mule, vehicle -1, cargo 0: an `srderr.ErrInvalidInput` error | unhappy (invalid input) |

### Cases: Barding (FR-026 to FR-028; "Barding", p. 100; Armor table, p. 92)

Entry: armor (cost; weight): expected cost; weight.

| Group | Entries | Path |
|---|---|---|
| table | Padded Armor (5 GP; 8 lb.): 20 GP; 16 lb.<br>Leather Armor (10 GP; 10 lb.): 40 GP; 20 lb.<br>Chain Shirt (50 GP; 20 lb.): 200 GP; 40 lb.<br>Half Plate Armor (750 GP; 40 lb.): 3,000 GP; 80 lb. | happy |
| most expensive | Plate Armor (1,500 GP; 65 lb.): 6,000 GP; 130 lb. | unhappy (boundary) |
| rule interaction | Shield, on the Armor table (Q9) (10 GP; 6 lb.): 40 GP; 12 lb.<br>coin kept (FR-028), Studded Leather Armor (45 GP; 13 lb.): 180 GP; 26 lb. (not 18,000 CP) | unhappy (rule interaction) |
| no weight (FR-027) | synthetic armor, `NoWeight` (10 GP; none): `BardingWeightUnknown` (armor id) | unhappy (missing content) |

### Cases: Ship repair (FR-029 to FR-032; "Ship Repair" and Vehicles table, p. 101: Rowboat 50 HP, Sailing Ship 300 HP)

Entry: Hit Points, supplies: cost; time.

| Group | Entries | Path |
|---|---|---|
| many points | 50, `notAbundant`: 1,000 GP; 100 half-days<br>50, `abundant`: 500 GP; 50 half-days | happy |
| boundaries | 1, `notAbundant`: 20 GP; 2 half-days<br>1, `abundant`: 10 GP; 1 half-day<br>3, `abundant`: 30 GP; 3 half-days | unhappy (boundary) |
| `ForShip` (FR-032) | Rowboat 50, `notAbundant`: `ShipRepairQuoted` 1,000 GP; 100 half-days<br>Rowboat 51, `notAbundant`: `ShipRepairExceedsShip` (requested 51, ship 50; package design)<br>Sailing Ship 300, `abundant`: `ShipRepairQuoted` 3,000 GP; 300 half-days | unhappy (boundary) |
| invalid (FR-031) | 0, `notAbundant`: an `srderr.ErrInvalidInput` error<br>-1, `abundant`: an `srderr.ErrInvalidInput` error | unhappy (invalid input) |

### Cases: Crafting (FR-033 to FR-036; p. 103)

Entry: item cost: raw materials (Copper Piece amount); days. Entries marked (Q15)
depend on the Copper Piece rounding unit or the exact-fraction reading of Time,
which the SRD does not state; they are package design, not SRD facts.

| Group | Entries | Path |
|---|---|---|
| SRD examples | Plate Armor 1,500 GP: 750 GP (75,000 CP); 150<br>Heavy Crossbow 50 GP: 25 GP (2,500 CP); 5 | happy |
| days boundaries | 10 GP: 5 GP (500 CP); 1<br>20 GP: 10 GP (1,000 CP); 2<br>45 GP: 22 GP 5 SP (2,250 CP); 5<br>2 GP: 1 GP (100 CP); 1<br>15 GP: 7 GP 5 SP (750 CP); 2 (the 5 SP half is Q15) | unhappy (boundary) |
| Silver and Copper (Q15; "round down") | 10 GP 1 CP (1,001 CP): 500 CP; 2<br>1 SP: 5 CP; 1<br>1 CP: 0 CP; 1<br>5 CP: 2 CP; 1 | unhappy (boundary) |
| printed form not kept (FR-033, Q15) | 5 SP: `Cost` of 25 CP, not 2 SP 5 CP; 1 | unhappy (rule interaction) |
| invalid (FR-034, FR-036) | 0 CP: 0 CP; an `srderr.ErrInvalidInput` error<br>5 SP per Day: an `srderr.ErrInvalidInput` error; an `srderr.ErrInvalidInput` error | unhappy (invalid input) |

### Cases: Crafting requirements (FR-040 to FR-048, FR-066)

Entry: item: expected (raw materials; days; tools; basis, or `NotCraftable` reason).

| Group | Entries | Path |
|---|---|---|
| generic (p. 103) | Plate Armor (1,500 GP): 750 GP; 150 days; `smiths_tools`; `genericRule` | happy |
| generic, no tool | Shield (10 GP): 5 GP; 1 day; no tools (the GM assigns); `genericRule`<br>Cart (15 GP, p. 100): 7 GP 5 SP (5 SP half is Q15); 2 days; no tools; `genericRule`<br>Gaming Set, dragonchess (1 GP, p. 94): 50 CP; 1 day; no tools; `genericRule` | unhappy (empty and minimal) |
| generic, focus variant | Arcane Focus, Crystal: the generic answer for its own price, tools `jewelers_tools`, `woodcarvers_tools` (FR-037) | unhappy (rule interaction) |
| override beats generic (FR-041, p. 103) | Potion of Healing (50 GP, `MagicGear` with override): 25 GP; 1 day; `herbalism_kit`; `itemOverride` (generic would give 25 GP and 5 days)<br>synthetic gear, price 50 GP, override 3 GP and 2 days: 3 GP; 2 days; `itemOverride`<br>synthetic gear with a rate and an override: the override; `itemOverride` (FR-048) | unhappy (rule interaction) |
| scroll row (FR-042, p. 103) | Spell Scroll (Cantrip) (30 GP) with scribing rows: 15 GP; 1 day; no tools; `scrollRow` (generic would give 15 GP and 3 days)<br>Spell Scroll (Level 1) (50 GP): 25 GP; 1 day; no tools; `scrollRow` | unhappy (rule interaction) |
| scroll without rows (FR-043) | Spell Scroll (Cantrip), no `scribing`: `NotCraftable(scribingTableMissing)` | unhappy (missing content) |
| price varies (FR-044) | Gaming Set (tool family, p. 94): `NotCraftable(priceVaries)`<br>Spell Scroll family (package design, content-equipment.md Q17; p. 99 prints 30 GP and 50 GP only): `NotCraftable(priceVaries)` | unhappy (empty and minimal) |
| not goods (FR-047) | Horse, Draft (p. 100): `NotCraftable(notAnItem)`<br>Spellcasting Services, 3 (p. 102): `NotCraftable(notAnItem)` | unhappy (empty and minimal) |
| rated or free (FR-045, FR-046) | Feed per day (p. 100): `NotCraftable(ratedPrice)`<br>synthetic gear at 0 GP: `NotCraftable(freeOfCharge)` | unhappy (boundary) |
| order (FR-048, FR-066) | Wretched (cost 0, p. 101): `NotCraftable(notAnItem)`, entry type first<br>Galley (30,000 GP): `NotCraftable(noCraftingRule)`, not 15,000 GP and 3,000 days (Q17) | unhappy (rule interaction) |

### Cases: Which tools craft an item (FR-037 to FR-039 on a synthetic tool set mirroring the Craft lists; FR-065 over bundled data; Craft entries, pp. 93-94)

Entry: item: expected tool ids (ordered).

| Group | Entries | Path |
|---|---|---|
| named or set | Club: `carpenters_tools`, `woodcarvers_tools`<br>Longsword (melee set): `smiths_tools`<br>Chain Shirt (medium set): `smiths_tools`<br>Plate Armor (heavy set): `smiths_tools`<br>Longbow (ranged set minus firearms and Sling): `woodcarvers_tools`<br>Arcane Focus: `jewelers_tools`, `woodcarvers_tools` | happy |
| exclusions and direct names | Whip (excluded from melee set): `leatherworkers_tools`<br>Hide Armor (excluded from medium set): `leatherworkers_tools`<br>Musket (named directly, excluded from Woodcarver's set): `tinkers_tools`<br>Sling (named directly): `leatherworkers_tools` | unhappy (rule interaction) |
| variants and sets (FR-037, FR-038) | Arcane Focus, Crystal: `jewelers_tools`, `woodcarvers_tools` (through `variantOf`)<br>synthetic Martial Melee weapon not in the bundled set: `smiths_tools` (a set expands by `weaponType`)<br>a Tool entry against `MeleeWeapons`: not matched | unhappy (rule interaction) |
| empty (FR-039) | Gaming Set, dragonchess (a variant has no Craft list<br>content-equipment.md FR-033): empty<br>Shield (the GM assigns the tool, p. 103): empty<br>Club with no tools: empty | unhappy (empty and minimal) |
| ordering (FR-039) | Club, tool list reversed: `carpenters_tools`, `woodcarvers_tools`<br>Club, a tool twice in the input: each id once | unhappy (immutability) |

### Cases: Scroll scribing (FR-049 to FR-051; Spell Scroll Costs, p. 103)

| Group | Entries (spell level: time, cost) | Path |
|---|---|---|
| levels 0 to 8 | 0: 1 day, 15 GP<br>1: 1 day, 25 GP<br>2: 3 days, 100 GP<br>3: 5 days, 150 GP<br>4: 10 days, 1,000 GP<br>5: 25 days, 1,500 GP<br>6: 40 days, 10,000 GP<br>7: 50 days, 12,500 GP<br>8: 60 days, 15,000 GP | happy |
| highest | 9: 120 days, 50,000 GP | unhappy (boundary) |
| out of range | `NewSpellLevel(-1)` and `(10)` are rejected before this service (content-spells.md FR-010) | unhappy (invalid input) |
| `NewScrollScribing` (FR-051) | rows 0 to 8 only: `*TableCoverageError`, missing [9]<br>level 3 twice, 4 absent: missing [4], repeated [3]<br>`NewScrollScribing(nil)`: missing [0 to 9] (the last is empty and minimal) | unhappy (missing content) |
| constructor (FR-050) | `ScrollScribing(rows 0 to 8)`: an `srderr.ErrInvalidInput` error | unhappy (invalid input) |

### Cases: Spellcasting fee by level (FR-052 to FR-057; Spellcasting Services, p. 102)

| Group | Entries (input: expected) | Path |
|---|---|---|
| rows | 0: 30 GP, Village, town, or city<br>1: 50 GP, Village, town, or city<br>2: 200 GP, Village, town, or city<br>9: 100,000 GP, City only | happy |
| range boundaries | 3: 300 GP, Town or city only<br>4: 2,000 GP, Town or city only<br>5: 2,000 GP, Town or city only<br>6: 20,000 GP, City only<br>8: 20,000 GP, City only | unhappy (boundary) |
| out of range | `NewSpellLevel(-1)` and `(10)` are rejected before this service (content-spells.md FR-010) | unhappy (invalid input) |
| total cost (FR-056) | level 3, components 100 GP: 400 GP (40,000 CP)<br>level 3, no components: 300 GP | unhappy (rule interaction) |
| rated components (FR-057) | level 3, components 5 SP per Day: an `srderr.ErrInvalidInput` error | unhappy (invalid input) |
| availability (FR-055) | level 2 in a Village: available<br>3 in a Village: not<br>5 in a Town: available<br>6 in a Town: not<br>9 in a City: available | unhappy (rule interaction) |
| `NewSpellcastingServices` (FR-054) | rows 0 to 3 and 5 to 9: `*TableCoverageError`, uncovered [4]<br>rows 0 to 4 and 4 to 9: overlapping [4]<br>`NewSpellcastingServices(nil)`: uncovered [0 to 9] | unhappy (missing content) |
| constructor (FR-053) | `SpellcastingServices(rows 0 to 4, 4 to 9)`: an `srderr.ErrInvalidInput` error | unhappy (invalid input) |

### Withdrawn ids

| Id | Status |
|---|---|
| FR-083 | Withdrawn: moved to equipment-rules.md (FR-065) |
| FR-080 to FR-082 | Never allocated (ids skipped; not withdrawn rows). Not the FR-080 or FR-081 of `derived-stats.md`, nor FR-081 or FR-082 of `content-equipment.md`, which are those specs' own. |

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

- **Value objects** (immutable, by value, collections copied on construction and on read): the types of
  section 6. **Domain services** (stateless zero-size types): `ArmorRules`,
  `ArmorTrainingRules`, `DrawnLoad`, `Crafting`, `DeriveBarding`, `ShipRepair`,
  `NewCarryingCapacity`, `SpellcastingServices.IsAvailable`/`TotalCost`. No
  aggregates, entities, events or ports.
- **Rows, not a `ContentSet`:** the services take decoded rows and `Tool` entries,
  never the `ContentSet` aggregate or a lookup (Q6).
- **Invariants:** table coverage (each level once) is enforced by the two
  constructors (`NewScrollScribing`, `NewSpellcastingServices`, which return `TableCoverageError`); `LoadSpeedLimited`
  always has `LimitFeet` 5; `ArmorBase.Total` equals `listedBase + dexterityContribution`.
- **D1.** Static SRD facts stay in `content`; arithmetic over them lives here
  (shared-types-and-ownership.md FR-006). **D2.** Carrying Capacity multipliers
  are domain constants switched on `CreatureSize` (like `Coin` ratios,
  content-equipment.md D1). **D3.** `ArmorRules` takes a `BaseRule`, not an
  `Armor`, so the Shield case is unrepresentable (content-equipment.md FR-024).

## 6. Interface

`go
// package character, files equipment_*.go (indicative signatures)
type ArmorRules struct{}                                                       // stateless; zero value ready
func (ArmorRules) BaseArmorClass(rule content.BaseRule, dexterity modifier.Modifier) ArmorBase // FR-001..FR-006
func (ArmorRules) SpeedPenaltyFeet(armor content.Armor, strength ability.Score) int            // FR-007..FR-010
func (ArmorRules) StealthCircumstances(worn []content.Armor) []modifier.RollCircumstance       // FR-079: nil or [Disadvantage], fresh slice

type ArmorBase struct{ /* unexported */ }                                      // comparable value
func (ArmorBase) ListedBase() int
func (ArmorBase) DexterityContribution() int
func (ArmorBase) Total() int

type ArmorTrainingRules struct{}                                               // FR-071..FR-078
func (ArmorTrainingRules) Assess(worn, trained []content.ArmorCategory) (ArmorTrainingEffects, error) // error: two body armors
func (ArmorTrainingRules) MonsterTraining(statBlockArmor []content.ArmorCategory) []content.ArmorCategory

type ArmorTrainingEffects struct{ /* unexported */ }
func (ArmorTrainingEffects) Untrained() []content.ArmorCategory                // fresh slice
func (ArmorTrainingEffects) WearsUntrainedBodyArmor() bool                     // the fact casting-and-concentration.md reads (FR-075)
func (ArmorTrainingEffects) RollCircumstancesFor(involved []ability.Ability) []modifier.RollCircumstance // nil or [Disadvantage] (FR-072)

type CarryingCapacity struct{ /* unexported */ }                               // FR-011..FR-020
func NewCarryingCapacity(size content.CreatureSize, strength ability.Score) CarryingCapacity
func (CarryingCapacity) CarryHalfPounds() int
func (CarryingCapacity) DragLiftPushPounds() int
func (CarryingCapacity) CanCarry(loadPounds int) (bool, error)
func (CarryingCapacity) Assess(loadPounds int) (LoadAssessment, error)

// LoadAssessment is the closed set of results of Assess. Variants:
// LoadWithinCarry, LoadSpeedLimited, LoadBeyondDragLiftPush.
//
//sumtype:decl
type LoadAssessment interface{ isLoadAssessment() }
type LoadSpeedLimited struct{ /* LimitFeet() is always 5 */ } // no Apply method: LoadSpeedRules.Limit (movement spec) applies it

type DrawnLoad struct{}                                                        // FR-021..FR-025
func (DrawnLoad) LimitPounds(animals []content.Mount) int
func (DrawnLoad) CanMove(animals []content.Mount, vehiclePounds, cargoPounds int) (bool, error)

type Barding struct{ /* Cost() content.Cost; Weight() content.Weight */ }       // FR-026..FR-028
func DeriveBarding(armor content.Armor) BardingOutcome                         // BardingDerived{Barding} | BardingWeightUnknown{ArmorID}

type SuppliesAndLabor int                                                      // NotAbundant, Abundant (typed enum with String())
type ShipRepair struct{ /* Cost() content.Cost; HalfDays() int */ }            // FR-029..FR-032
func QuoteShipRepair(hitPoints int, supplies SuppliesAndLabor) (ShipRepair, error)
func QuoteShipRepairFor(ship content.AirborneOrWaterborneVehicle, hitPoints int, supplies SuppliesAndLabor) (ShipRepairOutcome, error) // ShipRepairQuoted{Repair} | ShipRepairExceedsShip{RequestedHitPoints, ShipHitPoints}

type Crafting struct{}                                                         // FR-033..FR-048
func (Crafting) RawMaterials(cost content.Cost) (content.Cost, error)
func (Crafting) Days(cost content.Cost) (int, error)
func (Crafting) ToolsFor(item content.EquipmentEntry, tools []content.Tool) []content.ItemID
func (Crafting) RequirementsFor(item content.EquipmentEntry, tools []content.Tool, scribing ScrollScribing) (CraftingOutcome, error) // the zero ScrollScribing means "no table"
// CraftingOutcome: CraftingRequirements{RawMaterials, Days, Tools, Basis} | NotCraftable{Reason}
type CraftingBasis int            // GenericRule, ItemOverride, ScrollRow
type NotCraftableReason int       // PriceVaries, RatedPrice, FreeOfCharge, NotAnItem, NoCraftingRule, ScribingTableMissing

type ScrollScribing struct{ /* unexported */ }                                 // FR-049..FR-051, FR-067; zero value = no table (IsZero)
func NewScrollScribing(rows []content.SpellScrollCost) (ScrollScribing, error) // copies the rows; *TableCoverageError{Missing, Repeated []SpellLevel}, satisfies ErrInvalidInput, unless 0..9 once
func MustScrollScribing(rows []content.SpellScrollCost) ScrollScribing
func (ScrollScribing) ForSpellLevel(level content.SpellLevel) content.SpellScrollCost

type SpellcastingServices struct{ /* unexported */ }                           // FR-052..FR-057, FR-067
func NewSpellcastingServices(rows []content.SpellcastingService) (SpellcastingServices, error) // copies the rows; *TableCoverageError{Uncovered, Overlapping []SpellLevel}
func MustSpellcastingServices(rows []content.SpellcastingService) SpellcastingServices
func (SpellcastingServices) ForSpellLevel(level content.SpellLevel) content.SpellcastingService
func IsAvailable(row content.SpellcastingService, size content.SettlementSize) bool
func TotalCost(row content.SpellcastingService, componentCost content.Cost) (content.Cost, error)
```



- **Sync, no `ctx`:** the services are pure CPU functions over loaded values and take no `context.Context` (architecture.md section 4); loading is the
  `content.Source` port's concern (architecture.md section 4); failure contract Q16.
- **Exports** (FR-061; package `character`, no re-export package,
  each with a godoc comment): `ArmorRules`, `ArmorBase`, `ArmorTrainingRules`,
  `ArmorTrainingEffects`, `CarryingCapacity`, `LoadAssessment`, `LoadWithinCarry`,
  `LoadSpeedLimited`, `LoadBeyondDragLiftPush`, `DrawnLoad`, `Barding`,
  `BardingOutcome`, `BardingDerived`, `BardingWeightUnknown`, `SuppliesAndLabor`,
  `ShipRepair`, `ShipRepairOutcome`, `ShipRepairQuoted`, `ShipRepairExceedsShip`,
  `Crafting`, `CraftingOutcome`, `CraftingRequirements`, `NotCraftable`,
  `CraftingBasis`, `NotCraftableReason`, `ScrollScribing`, `NewScrollScribing`, `MustScrollScribing`,
  `SpellcastingServices`, `NewSpellcastingServices`, `MustSpellcastingServices`,
  `TableCoverageError`. No `*test` package gains anything.
- **Name stewardship.** `Barding`, `DrawnLoad`, `ShipRepair`, `Crafting`, `CraftingOutcome`, `CraftingRequirements`, `NotCraftable`, `NotCraftableReason`, `ScrollScribing` and `SpellcastingServices` are defined only here (content-equipment.md Q18).

## 7. Observability

**Silent by design** (observability.md section 3): every type is domain code
with no collaborator, no use case and no port, so nothing is emitted and this
spec adds no row to `doc/observability/signals.md`. Results carry the what and
why (`ArmorBase` parts, `CraftingRequirements.Basis`, `NotCraftable` and the
`TableCoverageError` naming the reason).

1. **What can go wrong?** A bad value, incomplete decoded content (missing spell level, armor with no weight, no scribing rows), a rule applied to the wrong entry, a rounding mistake.
2. **How would an operator know?** a returned `srderr.ErrInvalidInput` error for programmer errors; incomplete content returns `scribingTableMissing`, `BardingWeightUnknown` or a `TableCoverageError`, recorded only by calling use cases (FR-069); decode failures are recorded by `LoadContentSet`.
3. **Diagnosis:** the returned breakdown; no attributes.
4. **Count or time?** Nothing: no I/O, constant or linear work on small tables.
5. **Existing instrumentation still correct?** Yes (content-equipment.md section 7).
6. **What should a consumer monitor?** Existing content-load guidance. The first use case calling `Crafting`, `Barding` or the table constructors MUST record `scribingTableMissing`, `BardingWeightUnknown` and the `TableCoverageError` results under its own span (FR-069; shared-types-and-ownership.md FR-044).
7. **Anything noisy?** No.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none defined by this spec | n/a | n/a | n/a | n/a |

## 8. Test plan

Test files are under `character/` (package `character_test`, table-driven, `t.Run` and `t.Parallel` where safe) unless stated. Unit tests use
small synthetic entries and no ports, doubles or I/O.

| Ids | Level | File and notes |
|---|---|---|
| FR-001 to FR-004, FR-006 | unit | `armor_rules_test.go`: "Armor base" rows (Heavy: every modifier -5 to +10 gives the listed base); `ArmorBase` parts, equality under `==` for full, capped-above-cap and none rules; the two synthetic cap-0 rows |
| FR-005 | unit, architecture | `armor_rules_test.go`: an exhaustive type switch (no `default`) over closed `ArmorClassRule` (the `BonusRule` arm has no evaluation) and over the Dexterity treatment with no default; `internal/archtest`: the exported-name scan lists exactly `BaseArmorClass`, `SpeedPenaltyFeet` and `StealthCircumstances` as `ArmorRules` members. The Shield's `BonusRule` data check stays in content-equipment.md FR-024 |
| FR-007 to FR-010, FR-079 | unit | `armor_rules_test.go`: "Strength requirement" rows; sweeps of scores 1 to 30 for no requirement and for requirements 13 and 15; "Stealth Disadvantage" rows (all thirteen armors against the seven flagged ones; one element for several flagged armors; mutating the returned slice leaves the next read unchanged) |
| FR-011 to FR-014, FR-084 | unit | `carrying_capacity_test.go`: Goliath Strength 15: Medium row gives Large (450 lb. Carry), Large Form gives Huge (900 lb.), Gargantuan stays; "Carrying Capacity" rows (Carry and `CarryHalfPounds`, Drag/Lift/Push); Small equals Medium for scores 1 to 30; Tiny odd scores (1, 3, 15) give 15, 45, 225 |
| FR-015 to FR-020 | unit | same file: "Load assessment" rows (within, speed-limited, beyond, negative, load 0, `LimitFeet`); an architecture check that `LoadSpeedLimited` has no member taking a Speed |
| FR-021 to FR-025 | unit | `drawn_load_test.go`: "Drawn load" rows (limits, at and over, no animals, negative weights); other-order row, passed list unchanged, two calls equal |
| FR-026 to FR-028 | unit | `barding_test.go`: "Barding" rows; synthetic `NoWeight` armor gives `BardingWeightUnknown` with its id; equality and the coin-kept row |
| FR-029 to FR-032 | unit | `ship_repair_test.go`: "Ship repair" rows (`notAbundant`, `abundant` incl. odd points, zero and negative, `ForShip` at, above the ship's Hit Points, Sailing Ship) |
| FR-033 to FR-036 | unit | `crafting_test.go`: "Crafting" rows (raw materials, days incl. 10 GP 1 CP and 20 GP); rated cost an `srderr.ErrInvalidInput` error; zero cost gives 0 CP for `RawMaterials` and an `srderr.ErrInvalidInput` error for `days` |
| FR-037 to FR-039 | unit | `crafting_tools_test.go`: item named directly, sets by `weaponType` and armor category with `except`, a non-matching entry type, an item outside the bundled set; order by id, duplicates once, reversed and empty input |
| FR-040 to FR-048, FR-066 | unit | `crafting_requirements_test.go`: "Crafting requirements" rows (generic, Potion of Healing override, scroll rows, missing `scribing`, Gaming Set, Spell Scroll family, Feed, zero-cost goods, non-goods entries, Wretched, override-beats-rated, Galley, Cart, dragonchess, Arcane Focus, Crystal) |
| FR-049 to FR-051 | unit | `scroll_scribing_test.go`: ten rows; `NewScrollScribing` with a gap, a repeat and no rows (the `TableCoverageError` fields); `content.NewSpellLevel(-1)` and `(10)` assert upstream rejection |
| FR-052 to FR-057 | unit | `spellcasting_services_test.go`: nine level rows; `NewSpellcastingServices` with a gap, an overlap, an inverted range, and empty rows; five availability rows; components added, none and rated |
| FR-058 to FR-062, FR-069, FR-070 | architecture | `internal/archtest` (check A1): the `equipment_*.go` files import no `os`, `net`, `context`, `telemetry` or `*test` package and no other `character` spec file (shared-types-and-ownership.md FR-016); only the kernel packages and `content`; no `combat` import; no `go` statement, `math/rand`, `time.Now`, `time.Since`, `telemetry.Logger`, `Tracer`, `Meter` or signal constant; the consumer-name scan over the whole module and data (architecture-guardrails.md FR-025, FR-026); every type of section 6 exported from package `character`; duplicate-name scan; godoc present; no JSON key or `map[string]any` in a signature; no use case, command, port or `...Result` type; `derived-stats.md` and `magic-item-rules.md` use cases construct no other use case |
| FR-063, FR-067 | unit | `crafting_requirements_test.go`, `scroll_scribing_test.go`, `spellcasting_services_test.go`: mutate each returned slice (unchanged next read); build from a mutable slice, mutate it, assert lookups and equality unchanged; `IsAvailable` and `TotalCost` called as package functions |
| FR-064, FR-065, FR-068 | scenario | `scenarios/equipment_rules_test.go` (package `scenarios_test`) via public packages only: Leather Armor +3 gives 14; Plate Armor Strength 8 gives 10; the whole "Which tools craft an item" table; Plate Armor barding 6,000 GP and 130 lb. and crafting 750 GP, 150 days, `smiths_tools` (phase 5 data; until then FR-001 to FR-010, FR-026 to FR-039 cover the behaviour on synthetic entries) |
| FR-071 to FR-078 | unit | `armor_training_test.go`: "Armor Training" rows (`Untrained` and `RollCircumstancesFor` for Strength, Dexterity, Initiative, Constitution, Charisma, the empty and two-ability sets; at most one element; Shield rows give `[]` over all six abilities and the empty set; two- and three-body-armor rows return an `srderr.ErrInvalidInput` error, empty `worn` valid; `WearsUntrainedBodyArmor`; monster rows over every `involved` set); `modifier.Resolve` with and without one `Advantage` (the one place this suite imports the kernel's `RollMode`); equality, hash code, `UnsupportedError` on `Untrained` and the list, mutated inputs, input order irrelevant |

- **Table-driven tests:** one test per `<br>`-separated entry of every case table, named from its entry text, with its requirement and path (testing.md section 5). Unit tests use synthetic entries with the printed numbers; the scenario test uses bundled data.
- **Mutation testing:** the `equipment_*.go` files at least 80 with `gomu` (testing.md section 8; run only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`), attending to the cap in FR-002, `<` vs `<=` in FR-007/FR-008, `>` vs `>=` in FR-015, FR-017, FR-018, FR-022, the five-fold factor, the floor in FR-033, the ceiling in FR-035, the half-pound doubling in FR-011, the guard order in FR-048, in FR-072 the body-armor test and the Strength-or-Dexterity test, and in FR-079 the any-flagged test and the single element.
- **Doubles, suites, telemetry tests:** none. **Not covered:** nothing; FR-064, FR-065, FR-068 are added with the phase 5 data.

## 9. Open questions and decisions

No question is open or escalated. Settled (one line each):

- **Q3.** `character` (`character/equipment_*.go`): the only context besides `combat` that may depend on `content`, and it already consumes Armor Class (architecture.md section 2); the architecture.md and AGENTS.md wording edits are scheduled in section 10 and are not part of this spec's acceptance of the design.
- **Q8.** Alternatives: any one listed tool suffices. SRD p. 103 says "You must use the required tool" (singular) and each Craft entry names what that tool can make, so an item under several tools is made with one of them; `magic-item-rules.md` FR-079 already reads alternatives the same way. FR-039 states it; `ToolsFor` still only returns ids.
- **Q13.** Owner: `tool-proficiency-and-crafting.md` (needs character state; FR-070 forbids importing it here); it consumes `CraftingRequirements.Tools`. Scribing prerequisites: `spell-scroll-use.md` FR-027. `Crafting.Days` stays an `int` (FR-035), `ShipRepair.HalfDays` separate (FR-030). **Q23.** Coins and trade (gap sweep): not folded in; no purse aggregate is planned and `Cost.ValueInCopper` already converts denominations (content-equipment.md). The SRD gives no change-making policy, no rounding for half of an odd copper count and no list of full-value "trade goods and valuables" (content-equipment.md Q5), so only a host policy could implement them. Coin weight (a coin about a third of an ounce, fifty to the pound, p. 89) is a host input to an inventory total, and Q11 keeps loads in whole pounds. A future inventory spec owns all of it.
- **Q15.** Interpretation (the SRD gives no denomination; web sources only repeat the p. 103 text): round Raw Materials down in Copper Pieces, the smallest coin and the unit of `valueInCopper` (15 GP gives 7 GP 5 SP, 1 CP gives 0, 5 CP gives 2); Time is the ceiling of `valueInCopper` / 1,000 (10 GP 1 CP is 2 days). SRD examples are unaffected; dependent rows are marked "package design (Q15)"; a consumer wanting whole-GP rounding wraps the call. **Q20.** The Speed penalty: `SpeedPenaltyFeet` stays the integer output; the Speed arithmetic is `movement-lifting-and-flight.md`'s, as `MovementSpeeds.AdjustedBy(-penalty)` (a decrease reaches special speeds equally, floor 0: SRD p. 188 "Changes to Your Speeds"; `character` cannot import `combat`'s `MovementSpeeds`). The Stealth Disadvantage is FR-079 here, following the Armor Training pattern; `derived-stats.md` section 1 and the glossary are re-pointed by amendment.
- **Q1, Q2, Q4, Q5, Q12.** shared-types-and-ownership.md FR-048, FR-014, FR-005, FR-046; content-equipment.md FR-093. **Q21.** `CraftingBasis` is a named enum; an absent `scribing` table yields `scribingTableMissing`, never a nil value standing in for a state (FR-040, FR-043).
- **Q6.** Domain code takes `[]content.Tool` and rows, not the `ContentSet`; lookup is the port's job (architecture.md section 2). **Q14.** No use case here (shared-types-and-ownership.md FR-044). **Q7.** `RawMaterials(0)` is 0 Copper Pieces; only `days` rejects 0 (FR-034, FR-036). **Q10.** Above the Drag/Lift/Push maximum the load cannot be moved: `LoadBeyondDragLiftPush` (FR-018).
- **Q9.** Absent weight is `BardingWeightUnknown`; the Shield is on the Armor table so it is included (content-equipment.md Q4); no barding AC (FR-026, FR-027). **Q11.** Whole pounds for loads, half pounds for Tiny Carry, integer Copper Pieces for sums, half days for ship time; a `Pounds` quarter-pound type only when an inventory total needs it.
- **Q16.** Results for content-derived or domain outcomes, a returned `srderr.ErrInvalidInput` error for caller-computed numbers (loads, weights, hit points, rates): these are host-computed sums, not parsed text, so architecture.md section 6's untrusted-input row does not apply; a host taking such a number from outside validates at its own boundary. **Q17.** Vehicle: `NotCraftable(noCraftingRule)`; plain-priced Tack row, tool, focus or ammunition variant: generic rule (FR-066); package design.
- **Q19.** `movement-lifting-and-flight.md` owns the Speed limit arithmetic (FR-001, FR-002, FR-005, FR-058; `LoadSpeedRules.Limit` reads `LimitFeet`, FR-006, FR-059); this spec only reports 5 feet (rules-glossary-coverage.md GL-025a, GL-025b). **Q18.** Rank 9 in shared-types-and-ownership.md 4.11 (dependencies top out at rank 6); `derived-stats.md` (rank 10) lists this spec; FR-070 holds for any position.
- **Q22.** shared-types-and-ownership.md FR-060: this spec owns Armor Training (FR-071 to FR-078), `derived-stats.md` applies it (its FR-039); Shield AC loss is `derived-stats.md` FR-035, the spellcasting ban `casting-and-concentration.md` FR-063 (fed by FR-075). `derived-stats.md` lists this spec in its header (FR-056).

## 10. Changelog impact

Under `Unreleased`:

- **Added:** equipment rule services in `character`: `ArmorRules`, `CarryingCapacity`, `DrawnLoad`, `Barding`, `ShipRepair`, `Crafting`, `ScrollScribing`, `SpellcastingServices` and `ArmorTrainingRules` (untrained armor and the Strength and Dexterity Disadvantage as a `RollCircumstance` for the consumer), plus the Stealth-column Disadvantage as a `RollCircumstance` (`ArmorRules.StealthCircumstances`).
- **Documentation (with acceptance):** the `character` purpose row in architecture.md section 2 and the `character/` line of "Current contexts" in AGENTS.md say it also hosts the equipment rule maths (Q3).
- **Changed:** these rules leave the planned `content` API (content-equipment.md), which keeps entry types and data; nothing is released, so not **BREAKING**.
