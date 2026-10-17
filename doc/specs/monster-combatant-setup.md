# Monster combatant setup

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `ability-scores.md`, `observability-ports.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `content-monsters.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md`, `initiative-and-actions.md` |

Rank 26 in `shared-types-and-ownership.md` 4.11 (amendment A-1), beside `monster-limited-use.md`. `encounter-aggregate.md` (rank 28) consumes this spec and is therefore not listed (FR-056). The amendments A-1 to A-3 are defined in section 9.

## 1. Purpose and scope

A `Monster` (`content-monsters.md`) is static content. A fight needs per-creature inputs built from it. This spec is the one place that turns a stat block into those inputs, for **one creature**:

- Hit Points: the printed average **or** a roll of the Hit Point Dice, never both (SRD p. 255), as a `HitPointStatus` (`hit-points-and-damage.md` FR-087);
- the `ZeroHitPointsRule`: monsters die at 0 Hit Points unless the host chooses otherwise (p. 17);
- Resistances, Vulnerabilities and damage Immunities as `DamageAffinities`; condition Immunities as a `ConditionSet`'s immunities;
- Initiative: the modifier to roll, or the printed score to use as the count (p. 254);
- the Proficiency Bonus from Challenge Rating, and the saving-throw and skill `RollModifiers` that reproduce the printed numbers (pp. 255-256);
- the chosen size, the Speeds as `MovementSpeeds`, the `CombatantRef` and the `CombatantProfile`.

One pure value object, `MonsterSetup`, holds the result. One use case, `SetUpMonsterCombatant`, exists only because rolling Hit Point Dice needs the `dice.Roller` port (`shared-types-and-ownership.md` FR-044). Package design adds the names; no rule is added.

**Out of scope:**

- Rolling Initiative, ordering, ties (`initiative-and-actions.md` owns `RollInitiative`; a use case never calls another). This spec supplies the modifier or the fixed count.
- Building the `Combatant` entity, assigning a side or a `CombatantID`, adding it to an `Encounter` (`encounter-aggregate.md`; the host supplies every id, `combat-foundations.md` FR-009).
- Traits, Actions, Reactions, Legendary Actions, Spellcasting, limited use and Recharge (`monster-limited-use.md`, `content-monsters.md`); attack bonuses and save DCs, used as printed.
- Senses, Languages, Gear, Skills beyond the modifier maths, Alignment, XP and CR for encounter planning (`toolbox-combat-encounter-difficulty.md`); attuning a monster to items (p. 255; GM advice).
- "Running a Monster" (p. 255): advice to the Game Master, recorded in `rules-glossary-coverage.md`; no requirement.
- Qualified and described defences, and speeds with a qualifier: reported, not applied (FR-021, FR-022, FR-042).
- Anything not in SRD 5.2.1. Nothing core to this topic was left out.

## 2. Source

Printed page equals PDF page.

- SRD 5.2.1, "Stat Block Overview" and "Parts of a Stat Block": "Size", "Initiative", "Armor Class", p. 254; "Hit Points" and "Hit Dice by Size", "Speed", "Ability Scores", "Skills", "Resistances and Vulnerabilities", "Immunities", p. 255; "Challenge Rating", "Proficiency Bonus" and its table, p. 256.
- SRD 5.2.1, "Damage and Healing": "Resistance and Vulnerability", "No Stacking", "Immunity", "Monster Death", "Hit Point Maximum of 0", p. 17; "Hit Points", p. 16.
- SRD 5.2.1, "Rules Glossary": "Initiative", p. 184 (Initiative score); "Immunity", p. 183; "Ability Modifiers", p. 6; "Expertise", p. 182; "Skill List", p. 9.
- Worked blocks: "Aboleth", p. 258; "Assassin", p. 260; "Gray Ooze", p. 293; "Invisible Stalker", p. 301; "Archmage", p. 305; "Bat", p. 345; "Giant Rat", p. 353; "Air Elemental", p. 258; "Werewolf", p. 339; "Swarm of Insects", p. 361; "Half-Dragon", p. 297.
- Package design: `doc/rules/architecture.md` §2-§6; `doc/rules/observability.md` §2-§7; `doc/rules/testing.md` §4-§5, §8; `shared-types-and-ownership.md`; `combat-foundations.md`.

## 3. Vocabulary

Owned elsewhere and reused: `Monster`, `HitPointsEntry`, `DefenseEntry`, `SpeedEntry`, `StatBlockInitiative`, `ChallengeRating`, `AbilityBlock` (`content-monsters.md`); `CreatureSize`, `DamageType`, `ConditionKind`, `Skill`, `MonsterID` (`content-schema-and-decoder.md`); `HitPointStatus`, `ZeroHitPointsRule` (`DiesInstantly`, `FallsUnconscious`), `DamageAffinities`, `DamageOfType` (`hit-points-and-damage.md`); `ConditionSet` (`conditions.md`); `InitiativeCount`, `MovementSpeeds`, `CombatantProfile` (`initiative-and-actions.md`); `CombatantID`, `CombatantRef`, `MonsterCombatant` (`combat-foundations.md`); `RollModifiers` (`d20-tests.md`); `dice.Roller`, `dice.Expression`, `dice.Result` (`dice-and-rolls.md`); `Modifier`, `Ability` (kernel).

| SRD term | Type defined here | Notes |
|---|---|---|
| (a monster prepared for a fight) | `MonsterSetup` | Not an SRD term. Immutable, no identity, no `CombatantID`. |
| Hit Points: number or dice, "don't use both" | `MonsterHitPointsMethod` (closed: `UseAverageHitPoints`, `RollHitPointDice`); `ResolvedMonsterHitPoints` (closed: `PrintedAverageHitPoints`, `RolledMonsterHitPoints`) | p. 255. |
| Initiative modifier / Initiative score | `MonsterInitiativeMode` (`rolled`, `fixed`); `MonsterInitiativePlan` (closed: `RolledMonsterInitiative`, `FixedMonsterInitiative`) | p. 254. |
| Monster Death | the default of `ZeroHitPointsRule` | p. 17. |
| Proficiency Bonus (by CR) | `MonsterSetup.ProficiencyBonus` (`Modifier`) | p. 256. Not `character`'s `ProficiencyBonus` (levels only, +2 to +6). |
| a defence not applied | `UnappliedDefense`, `UnappliedDefenseReason` (`qualified`, `described`), `MonsterDefenseKind` | Package names. |
| a speed not applied | `UnappliedSpeed` | Package name. |
| (setup problems) | `MonsterSetupProblem`, `MonsterSetupParse` (closed: `MonsterSetupParsed`, `MonsterSetupRejected`) | Package names. |

Names without an SRD source add no rules (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. One behaviour each. Each row of a case table is one table-driven test.

### 4.1 Size, identity and construction

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `MonsterSetup` MUST be an immutable value object built by `NewMonsterSetup(monster, hitPoints, opts...) (MonsterSetup, error)` with the options `WithSize`, `WithZeroHitPointsRule`, `WithInitiativeMode` and `WithExtraAffinities` (more than three optional values, so functional options). A rejection is a `*MonsterSetupRejected` error matching `srderr.ErrInvalidInput`; `MustMonsterSetup` (same arguments) panics with that error and is for literals and tests only. `hitPoints` is a required `ResolvedMonsterHitPoints`. It holds the `monster`, `size`, `hitPointStatus`, `hitPointsRaisedToOne`, `conditions`, `initiative`, `proficiencyBonus`, `speeds`, `unappliedDefenses` and `unappliedSpeeds` (the section 5 list), and is compared by value through an `Equal` method over all of them. | SRD 5.2.1, "Stat Block Overview", p. 254; architecture §3, §6 | happy |
| FR-002 | A monster with exactly one size option MUST take that size when `size` is absent. A supplied `size` equal to it is accepted. | SRD 5.2.1, "Size", p. 254 | happy |
| FR-003 | A monster with several size options ("Medium or Small") and no `size` MUST be rejected with `MonsterSetupProblem.SizeChoiceRequired`; the engine does not pick (the SRD: "you choose the creature's size"). | SRD 5.2.1, "Size", p. 254; "Assassin", p. 260 | unhappy (invalid input) |
| FR-004 | A `size` not among the monster's options MUST be rejected with `sizeNotOffered`, including any other size for a single-option monster. | SRD 5.2.1, "Size", p. 254 | unhappy (boundary) |
| FR-005 | `NewMonsterSetup` MUST NOT panic for any argument values and MUST report every problem found, once each, in enum order `sizeChoiceRequired`, `sizeNotOffered`, `rolledExpressionMismatch`, in one `*MonsterSetupRejected` (it holds a copied, non-empty slice of `MonsterSetupProblem`s and is never built empty). A rejection is returned if and only if no `MonsterSetup` is built, and `MustMonsterSetup` panics exactly then. | `errors-and-results.md` FR-023 to FR-026; architecture §6 | unhappy (invalid input) |
| FR-006 | `MonsterSetup.Ref(CombatantID id)` MUST return `CombatantRef(id, MonsterCombatant(monster.id))`. The setup holds no `CombatantID`, so one setup value MAY be reused for several creatures only where the host wants identical Hit Points (FR-013). | `combat-foundations.md` FR-010, FR-013; `hit-points-and-damage.md` FR-104 | happy |

### 4.2 Hit Points

A creature's Hit Points come from exactly one of two sources, chosen by the host (interpretation: "don't use both" is a choice rule, not a sum rule).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-007 | `MonsterHitPointsMethod` MUST be a closed variant set (interface with an unexported marker method) with exactly `UseAverageHitPoints` and `RollHitPointDice`, and `ResolvedMonsterHitPoints` closed with exactly `PrintedAverageHitPoints` and `RolledMonsterHitPoints(dice.Result)`. No type or member combines the average and a roll. | SRD 5.2.1, "Hit Points", p. 255 ("don't use both") | happy |
| FR-008 | With `PrintedAverageHitPoints` the Hit Point maximum MUST equal the printed average (`HitPointsEntry.Average`), not recomputed from the dice or the Constitution modifier. | SRD 5.2.1, "Hit Points", p. 255; `content-monsters.md` FR-015 | happy |
| FR-009 | With `RolledMonsterHitPoints(roll)` the maximum MUST be `roll.total` (dice plus the printed flat modifier) when that is at least 1, and 1 otherwise. Table 4.2a. | SRD 5.2.1, "Hit Points", p. 255; "Hit Point Maximum of 0", p. 17 | unhappy (boundary) |
| FR-010 | A rolled total below 1 MUST be raised to 1 and `MonsterSetup.HitPointsRaisedToOne` MUST be true; otherwise false (always false for the average). Interpretation: the SRD says a creature whose maximum reaches 0 dies (p. 17), and the rolled block "1 (1d4 − 1)" prints an average of 1, so a creature set up for a fight starts alive with at least 1 Hit Point (Q-1). | SRD 5.2.1, "Hit Point Maximum of 0", p. 17; "Bat", p. 345 | unhappy (boundary) |
| FR-011 | `RolledMonsterHitPoints.Roll.expression` MUST equal the monster's Hit Point Dice expression (count, die, modifier); otherwise `NewMonsterSetup` MUST report `rolledExpressionMismatch`. | SRD 5.2.1, "Hit Points", p. 255 | unhappy (invalid input) |
| FR-012 | The `HitPointStatus` MUST be created through `NewHitPointStatus(maximum, zeroHitPointsRule, affinities)`: current equals maximum, no Temporary Hit Points, `Alive`. `Restore` is never used. The maximum is at least 1 by FR-008 and FR-009, so the constructor cannot fail. | `hit-points-and-damage.md` FR-087; SRD 5.2.1, "Hit Points", p. 16 | happy |
| FR-013 | One `MonsterSetup` is one creature. Several creatures of one stat block are several setups; with `RollHitPointDice` each is rolled separately (interpretation: p. 255 says to roll to determine "the monster's Hit Points", singular; Initiative rolls for identical creatures are shared by `initiative-and-actions.md` FR-008, a separate rule). | SRD 5.2.1, "Hit Points", p. 255; "Initiative", p. 13 | unhappy (rule interaction) |
| FR-014 | The Hit Dice by Size table MUST NOT be enforced: the die of a stat block that differs from its size's die is accepted unchanged. The table says "typically". | SRD 5.2.1, "Hit Dice by Size", p. 255 | unhappy (rule interaction) |

Table 4.2a, maximum Hit Points (one test per row; faces in roll order; `+m` is the printed flat modifier).

| Label | Stat block | Method | Expected maximum; raised | Source | Path |
|---|---|---|---|---|---|
| printed average | Aboleth, HP 150 (20d10 + 40) | average | 150; no | p. 258 | happy |
| rolled, all ones | Aboleth | roll, twenty 1s | 60 (20 + 40); no | p. 255 | unhappy (boundary) |
| rolled, all tens | Aboleth | roll, twenty 10s | 240; no | p. 255 | unhappy (boundary) |
| rolled, ordinary | Giant Rat, HP 7 (2d6) | roll 3, 4 | 7; no | p. 353 | happy |
| rolled to zero | Bat, HP 1 (1d4 − 1) | roll 1 | 1; yes (total 0) | p. 345; p. 17 | unhappy (boundary) |
| rolled to one | Bat | roll 2 | 1; no (total 1) | p. 345 | unhappy (boundary) |
| rolled to three | Bat | roll 4 | 3; no | p. 345 | unhappy (boundary) |
| average of a minus block | Bat | average | 1; no | p. 345 | unhappy (minimal) |
| minus block, all ones | a 4d8 − 4 block | roll 1, 1, 1, 1 | 1; yes (total 0) | p. 255; p. 17 | unhappy (boundary) |
| wrong expression | Bat | roll result for 2d6 | rejected `rolledExpressionMismatch` | p. 255 | unhappy (invalid input) |
| wrong modifier | Giant Rat | roll result for 2d6 + 1 | rejected `rolledExpressionMismatch` | p. 255 | unhappy (invalid input) |

### 4.3 When it drops to 0 Hit Points

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-015 | The `zeroHitPointsRule` argument MUST default to `DiesInstantly` ("A monster dies the instant it drops to 0 Hit Points"), and the `HitPointStatus` MUST carry the rule given, unchanged. | SRD 5.2.1, "Monster Death", p. 17 | happy |
| FR-016 | A host MAY pass `FallsUnconscious` for an individual monster ("a Game Master can ignore this rule for an individual monster and treat it like a character"); the status then makes Death Saving Throws at 0 Hit Points (`hit-points-and-damage.md` FR-042, FR-096). | SRD 5.2.1, "Monster Death", p. 17 | unhappy (rule interaction) |
| FR-017 | The rule MUST NOT be inferred from the monster's type, size, Challenge Rating or traits; every monster gets the same default. This default is a `monster-combatant-setup` argument default; `NewHitPointStatus` keeps no default (`hit-points-and-damage.md` Q-9). | SRD 5.2.1, "Monster Death", p. 17; architecture §3 | unhappy (rule interaction) |

### 4.4 Defences

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | Each unqualified damage-type entry of Resistances MUST become `DamageOfType(type)` in the `DamageAffinities` resistances; of Vulnerabilities, a vulnerability to the type; of Immunities, an immunity to the type. A monster without such entries gets empty sets. | SRD 5.2.1, "Resistances and Vulnerabilities", "Immunities", p. 255; "Resistance and Vulnerability", "Immunity", p. 17 | happy |
| FR-019 | Each unqualified condition entry of Immunities MUST become an immunity of the `ConditionSet`; the set MUST hold no explicit condition. Exhaustion is a condition entry like any other. | SRD 5.2.1, "Immunities", p. 255; "Immunity", pp. 17, 183; `conditions.md` FR-021, FR-025 | happy |
| FR-020 | Damage and condition entries MUST NOT cross: a damage entry never reaches the condition set, a condition entry never reaches the affinities. Entries are applied once each; a type listed in two entries of one list counts once (`hit-points-and-damage.md` FR-091). A Resistance and a Vulnerability to the same type are both kept (`hit-points-and-damage.md` FR-020). | SRD 5.2.1, "No Stacking", p. 17 | unhappy (rule interaction) |
| FR-021 | An entry with a qualifier ("Charmed (with Mind Blank)", "Charmed (except from its vampire master)", Piercing "from weapons wielded by creatures under the effect of a Bless spell") MUST NOT be applied. It MUST be listed in `unappliedDefenses` with reason `qualified`. Interpretation: the qualifier makes the defence conditional on a circumstance the engine does not track; applying it unconditionally would over-apply it, dropping it would lose it (Q-3). | SRD 5.2.1, "Immunities", p. 255; "Archmage", p. 305; "Vampire Familiar", p. 334; "Rakshasa", p. 317 | unhappy (rule interaction) |
| FR-022 | A described entry (printed text naming neither a damage type nor a condition, "Damage type chosen for the Draconic Origin trait") MUST be listed in `unappliedDefenses` with reason `described`. | SRD 5.2.1, "Resistances and Vulnerabilities", p. 255; "Half-Dragon", p. 297 | unhappy (rule interaction) |
| FR-023 | `unappliedDefenses` MUST be a copied slice ordered by kind (`vulnerability`, `resistance`, `immunity`, as the stat block's entries are read) and then by printed order. Each holds its `MonsterDefenseKind`, the `DefenseEntry` and the reason. | SRD 5.2.1, "Stat Block Overview", p. 254; architecture §4 | unhappy (boundary) |
| FR-024 | `extraAffinities` (a `DamageAffinities`, default empty) MUST be merged into the stat block's affinities by union (duplicates collapse). It is how a host resolves a described or qualified defence it has decided applies (a chosen Damage Type, a Bless-affected hit). | SRD 5.2.1, "Immunities", p. 255; "Immunity", p. 17; architecture §3 (extension point) | unhappy (rule interaction) |

Table 4.4a, defences (one test per row).

| Label | Printed | Expected | Source | Path |
|---|---|---|---|---|
| plain Resistances | Air Elemental: Bludgeoning, Lightning, Piercing, Slashing | four resistances | p. 258 | happy |
| damage then conditions | Air Elemental Immunities: Poison, Thunder; Exhaustion, Grappled, Paralyzed, Petrified, Poisoned, Prone, Restrained, Unconscious | immune to two types; eight condition immunities; nothing explicit | p. 258 | happy |
| no defence entries | Giant Rat | empty affinities, empty immunities, no unapplied | p. 353 | unhappy (empty and minimal) |
| qualified condition | Archmage: Psychic; Charmed (with Mind Blank) | Psychic applied; Charmed unapplied (`qualified`, immunity) | p. 305 | unhappy (rule interaction) |
| qualified damage | Rakshasa Vulnerabilities: Piercing from weapons wielded by creatures under a Bless spell | no vulnerability; one unapplied (`qualified`, vulnerability) | p. 317 | unhappy (rule interaction) |
| described | Half-Dragon: Damage type chosen for the Draconic Origin trait | one unapplied (`described`, resistance) | p. 297 | unhappy (rule interaction) |
| host supplies the choice | Half-Dragon plus `extraAffinities` resistance to Fire | resistance to Fire applied; the described entry still listed | p. 297; FR-024 | unhappy (rule interaction) |
| extra duplicates the block | Air Elemental plus extra resistance to Piercing | four resistances, not five | FR-020, FR-024 | unhappy (rule interaction) |
| order of unapplied | a block with qualified Immunity then qualified Vulnerability | vulnerability listed first | FR-023 | unhappy (boundary) |
| extra contradicts the block | Air Elemental (Resistance to Piercing) plus extra immunity to Piercing | resistance and immunity both kept, no problem | FR-020, FR-024; `hit-points-and-damage.md` FR-020 | unhappy (rule interaction) |

### 4.5 Initiative

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-025 | `MonsterInitiativeMode` MUST have exactly `rolled` and `fixed` and default to `rolled`; the setup's `initiative` MUST be a closed `MonsterInitiativePlan`: `RolledMonsterInitiative(modifier)` for `rolled`, `FixedMonsterInitiative(count)` for `fixed`, each carrying only its own datum. | SRD 5.2.1, "Initiative", p. 254 | happy |
| FR-026 | The rolled plan's `modifier` MUST be the printed Initiative modifier as a `Modifier`, not recomputed from Dexterity ("typically equal", but a block may add Proficiency Bonus or more). The host passes it to `RollInitiative` as the participant's Initiative modifier and adds the roll-only modifier, Surprise and condition contributions there. | SRD 5.2.1, "Initiative", p. 254; `content-monsters.md` FR-014; `initiative-and-actions.md` FR-001, FR-111 | happy |
| FR-027 | The fixed plan's `count` MUST be the printed Initiative score as an `InitiativeCount`, not "10 plus the modifier": scores that differ from it (Advantage built in) are used as printed. Table 4.5a. The host passes the count to `EstablishInitiativeOrder`. | SRD 5.2.1, "Initiative", p. 254 ("use the Initiative score ... in combat"); "Initiative", p. 184 | unhappy (rule interaction) |
| FR-028 | The setup MUST roll no Initiative die and MUST NOT call `RollInitiative`; the plan only hands over data. | `shared-types-and-ownership.md` FR-044 | unhappy (forbidden edge) |

Table 4.5a, Initiative (one test per row).

| Label | Printed | Mode | Expected | Source | Path |
|---|---|---|---|---|---|
| typical, rolled | Aboleth `+7 (17)` | rolled | modifier +7 | p. 258 | happy |
| typical, fixed | Aboleth `+7 (17)` | fixed | count 17 | p. 258 | happy |
| score above 10 plus modifier | Invisible Stalker `+7 (22)` | fixed | count 22 (not 17) | p. 301 | unhappy (rule interaction) |
| negative modifier, score above | Gray Ooze `−2 (13)` | fixed | count 13 (not 8) | p. 293 | unhappy (rule interaction) |
| negative modifier, rolled | Gray Ooze | rolled | modifier −2 | p. 293 | unhappy (boundary) |
| default mode | Bat `+2 (12)` | not given | rolled, modifier +2 | p. 345 | happy |

### 4.6 Proficiency Bonus, saving throws and skills

A stat block prints the final numbers. The setup keeps them exactly and only labels how they are made up, so a D20 Test breakdown can show the Proficiency Bonus.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-029 | `MonsterSetup.ProficiencyBonus` MUST be the Proficiency Bonus of the monster's Challenge Rating (`Monster.ProficiencyBonus`, `content-monsters.md` FR-031, table 4.5a there: +2 for CR 0-4, +3 for 5-8, +4 for 9-12, +5 for 13-16, +6 for 17-20, +7 for 21-24, +8 for 25-28, +9 for 29-30), as a `Modifier`. It MUST NOT use `character`'s `ProficiencyBonus`, which stops at +6. | SRD 5.2.1, "Proficiency Bonus", p. 256 | happy |
| FR-030 | `savingThrow(Ability)` MUST return `RollModifiers` whose `ability` slot is the ability modifier and whose total (ability, Proficiency Bonus slot, others) equals the printed save modifier. With `d` the printed save minus the ability modifier: `d == 0` gives no Proficiency Bonus; `d` equal to the Proficiency Bonus puts it in the slot; any other `d` gives no slot and `others` of `[Modifier(d)]` (the block is authoritative: "Other modifiers might apply"). Table 4.6a. | SRD 5.2.1, "Ability Scores", "Proficiency Bonus", pp. 255-256; "The Bonus Doesn't Stack", p. 8 | unhappy (rule interaction) |
| FR-031 | `skillCheck(Skill)` for a skill the block lists MUST use the ability modifier of `Skill.Ability` and the same decomposition, with one more case: `d` equal to twice the Proficiency Bonus puts the doubled value in the slot (the bonus is "doubled", as Expertise, p. 182; `derived-stats.md` FR-066 maps an `expertise` term to the same slot). For a skill the block does not list the result MUST be the ability modifier alone. | SRD 5.2.1, "Skills", p. 255; "Skill List", p. 9; "Expertise", p. 182 | unhappy (rule interaction) |
| FR-032 | `abilityCheck(Ability)` MUST return the ability modifier alone: a stat block has no ability-check proficiency except through Skills. | SRD 5.2.1, "Skills", p. 255 | happy |
| FR-033 | For every ability and every listed skill, the sum of the returned slots MUST equal the printed value, and the returned `RollModifiers` MUST hold at most one Proficiency Bonus (the shape guarantees it). The ability modifier is `AbilityBlock.AbilityModifier` and is never stored. | `d20-tests.md` FR-017, FR-018; `content-monsters.md` FR-019 | unhappy (rule interaction) |

Table 4.6a, modifiers (PB from the CR; `A` ability modifier, `P` slot, `O` others; one test per row).

| Label | Block and request | Printed | Expected | Source | Path |
|---|---|---|---|---|---|
| save, not proficient | Aboleth Strength save (+5, A+5) | +5 | `A+5` | p. 258 | happy |
| save, proficient | Aboleth Dexterity save (A−1, PB +4) | +3 | `A−1 P+4` | p. 258 | happy |
| skill, doubled | Aboleth Perception (Wisdom A+2) | +10 | `A+2 P+8` (twice +4) | p. 258; p. 182 | unhappy (rule interaction) |
| skill, doubled, other ability | Aboleth History (Intelligence A+4) | +12 | `A+4 P+8` | p. 258 | unhappy (rule interaction) |
| skill not listed | Aboleth Arcana | none | `A+4` | p. 255 | unhappy (minimal) |
| ability check | Aboleth Charisma check | none | `A+4` | p. 255 | happy |
| a difference that is neither | synthetic save, printed A+3, PB +2, printed +6 | +6 | `A+3 O[+3]`, no slot | p. 255 | unhappy (invalid input) |
| a negative difference | synthetic skill, A+3, printed +1 | +1 | `A+3 O[−2]` | p. 255 | unhappy (boundary) |
| bands | CR 0, 4, 5, 8, 9, 12, 13, 16, 17, 20, 21, 24, 25, 28, 29, 30 | PB | +2, +2, +3, +3, +4, +4, +5, +5, +6, +6, +7, +7, +8, +8, +9, +9 | p. 256 | unhappy (boundary) |

### 4.7 Size, Speeds and profile

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-034 | `speeds` MUST be a `MovementSpeeds` built from the unqualified, single-mode `SpeedEntry`s: walk, and the special speeds burrow, climb, fly and swim, in feet. `MovementSpeeds` has no hover slot, so the `hover` flag is not copied; nothing is lost, because the printed `SpeedEntry` (with `hover`, `content-monsters.md` FR-016, FR-017) stays readable from `MonsterSetup.Monster`. When there is no such walk entry the walking Speed MUST be 0. | SRD 5.2.1, "Speed", p. 255; Glossary "Speed", p. 188 | happy |
| FR-035 | An entry with a qualifier ("wolf form only", "GM's choice") or with more than one mode ("Climb or Fly") MUST NOT be applied and MUST be listed in `unappliedSpeeds` in printed order. Interpretation: it holds only in a form or by a choice the engine does not track; the host applies it with `UpdateMovementSpeeds` (`initiative-and-actions.md` FR-116) when it does. | SRD 5.2.1, "Speed", p. 255; "Werewolf", p. 339; "Swarm of Insects", p. 361 | unhappy (rule interaction) |
| FR-036 | `profile(CombatantID id)` MUST return `CombatantProfile(id, size, speeds)` with the chosen size and the FR-034 speeds, for `EstablishInitiativeOrder`. | `initiative-and-actions.md` FR-025 (profile per count; FR-024 for the counts); SRD 5.2.1, "Size", p. 254 | happy |

Table 4.7a, Speeds (one test per row).

| Label | Printed | Expected | Source | Path |
|---|---|---|---|---|
| walk and swim | Aboleth: 10 ft., Swim 40 ft. | walk 10, swim 40 | p. 258 | happy |
| hover not copied | Air Elemental: 10 ft., Fly 90 ft. (hover) | walk 10, fly 90; `setup.monster` speeds still show fly 90 with hover true; no unapplied entry | p. 258 | happy |
| form qualifier | Werewolf: 30 ft., 40 ft. (wolf form only) | walk 30; unapplied: walk 40 (wolf form only) | p. 339 | unhappy (rule interaction) |
| alternative modes | Swarm of Insects: 20 ft., Climb or Fly 20 ft. (GM's choice) | walk 20; unapplied: {climb, fly} | p. 361 | unhappy (rule interaction) |
| no walk entry | synthetic: Fly 30 ft. only | walk 0, fly 30 | p. 255 | unhappy (boundary) |
| zero Speed | synthetic: walk 0 | walk 0 | p. 255 | unhappy (boundary) |
| no Speed entries | synthetic: empty speed list | walk 0, no special speeds, nothing unapplied | p. 255 | unhappy (empty and minimal) |

### 4.8 The use case

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-037 | `SetUpMonsterCombatant` MUST take a `SetUpMonsterCombatantCommand` (`monster`, optional `size`, `hitPoints` as a `MonsterHitPointsMethod`, `zeroHitPointsRule` default `DiesInstantly`, `initiativeMode` default `rolled`, `extraAffinities` default empty) and return a closed `SetUpMonsterCombatantResult`: `SetUpMonsterCombatantApplied(setup)` or `SetUpMonsterCombatantInvalid(problems)` (`combat-foundations.md` FR-022, FR-023). | architecture §4; `shared-types-and-ownership.md` FR-017 | happy |
| FR-038 | With `UseAverageHitPoints` the use case MUST make no request to the `dice.Roller`. With `RollHitPointDice` it MUST make exactly one request, for the monster's printed Hit Point Dice expression including its flat modifier, through `dice.Roll`, and wrap the result as `RolledMonsterHitPoints`. | SRD 5.2.1, "Hit Points", p. 255; `dice-and-rolls.md` FR-050 | happy |
| FR-039 | A size problem (FR-003, FR-004) MUST be found before any roll: a command with a size problem MUST return `SetUpMonsterCombatantInvalid` and consume no die. | architecture §4 (no wasted effects) | unhappy (invalid input) |
| FR-040 | A port failure from `dice.Roll` (a roller error, or the `srderr.ErrContractViolation` that `dice.Roll` reports for an invalid result) MUST be returned wrapped as a `*srderr.PortError` for port `dice` (`errors.Is` finds the cause) with no result and no new state. A `rolledExpressionMismatch` cannot arise from the use case (it requests the expression itself). A roller returning a total that disagrees with its faces is the same contract violation from `dice.Roll` and is returned the same way; a Hit Point Dice expression that cannot be rolled is excluded by `content-monsters.md` FR-015 (`HitPointsEntry` only holds d4 to d20 dice with count at least 1), so the use case adds no check. | `dice-and-rolls.md` FR-051, FR-052; testing §4 | unhappy (collaborator failure) |
| FR-041 | Given the same command and the same roll the use case MUST return an equal result; the domain reads no clock and no randomness. Neither the command nor a returned collection is mutated; returned slices are copies (mutating one does not change the result). | architecture §3; testing §4 | unhappy (determinism, immutability) |
| FR-042 | `SetUpMonsterCombatantApplied` is class `applied`; `SetUpMonsterCombatantInvalid` is class `invalid` (host input the engine cannot use, `combat-foundations.md` FR-031): `srd5e.outcome` `failure`, span error status, no `srd5e.combat.errors`, no `error.type`, no log. | `combat-foundations.md` FR-031, FR-032, FR-034 | unhappy (telemetry) |
| FR-043 | The use case MUST run through the `combat` telemetry helper over `telemetry.Telemetry` and MUST NOT call, hold or accept another use case; it composes the pure `MonsterSetup`. No event is produced (setup is not a happening in the fight). | `shared-types-and-ownership.md` FR-044; `combat-foundations.md` FR-029, FR-035 | unhappy (forbidden edge) |
| FR-044 | With the zero-value (Noop) and with the recording `telemetrytest` double the use case MUST return equal results. | observability §9 | unhappy (determinism) |

### 4.9 Layering and exports

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-045 | Everything but `SetUpMonsterCombatant` and its command and result MUST be in package `combat` files that are pure, synchronous and silent (none imports `telemetry`, `dice` or `context`). Those files MAY import only the kernel packages, `content`, `d20`, and the files of `combat` specs of lower rank; never a `combat` spec of rank 26 or higher, nor `toolbox` (checked by `internal/archtest`). | `combat-foundations.md` FR-014, FR-026; architecture §1, §2 | unhappy (forbidden edge) |
| FR-046 | Package `combat` MUST export every type of section 5 with a godoc comment and no generic name (`Rejected`, `Applied`). No consumer-specific name appears. | architecture §4, §7, §9 | happy |
| FR-047 | The setup MUST NOT hold a `Monster` copy in mutable form, a `*rand.Rand`, or any display text; `UnappliedDefense` and `UnappliedSpeed` carry the content value objects, not strings. | architecture §3, §10 | unhappy (immutability) |

### 4.10 Telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | The use case MUST record the shared four signals of `combat-foundations.md` FR-029 under operation `set_up_monster_combatant` and span `srd5e.combat.set_up_monster_combatant`, with `srd5e.combat.result` one of `set_up_average`, `set_up_rolled` (class `applied`), `size_choice_required`, `size_not_offered` (class `invalid`), chosen by the first matching rule of table 4.10a. | `combat-foundations.md` FR-030, FR-040 | happy |
| FR-049 | An applied result MUST add the extra attribute `srd5e.combat.monster_hit_points_method` (`average` or `rolled`) through `attributes` (FR-041 there), on the span and the `operations` increment only. | `combat-foundations.md` FR-041 | happy |
| FR-050 | When a rolled total was raised to 1 (FR-010) the use case MUST increment `srd5e.combat.monster_hit_points_clamped` once. No other path increments it. | observability §5 | unhappy (boundary) |
| FR-051 | No attribute MUST carry a `MonsterID`, `CombatantID`, Hit Point value or free text (`combat-foundations.md` FR-033). | observability §5 | unhappy (telemetry) |

Table 4.10a, `srd5e.combat.result` (first match wins).

| Variant and condition | Value | Class |
|---|---|---|
| `Applied`, method `UseAverageHitPoints` | `set_up_average` | applied |
| `Applied`, method `RollHitPointDice` | `set_up_rolled` | applied |
| `Invalid` holding `sizeChoiceRequired` | `size_choice_required` | invalid |
| `Invalid` holding `sizeNotOffered` | `size_not_offered` | invalid |

### Withdrawn ids

None.

## 5. Model

- **Value objects** (immutable, unexported fields, value receivers): `MonsterSetup` (`monster`, `size`, `hitPointStatus`, `hitPointsRaisedToOne`, `conditions` (a `ConditionSet`), `initiative` (`MonsterInitiativePlan`), `proficiencyBonus`, `speeds`, `unappliedDefenses`, `unappliedSpeeds`; methods `ref`, `profile`, `savingThrow`, `skillCheck`, `abilityCheck`); `MonsterHitPointsMethod`, `ResolvedMonsterHitPoints`, `MonsterInitiativeMode`, `MonsterInitiativePlan`, `UnappliedDefense`, `UnappliedDefenseReason`, `MonsterDefenseKind`, `UnappliedSpeed`, `MonsterSetupProblem`, `MonsterSetupParse`.
- **Invariants:** a maximum of at least 1 (FR-008, FR-009); a chosen size among the options (FR-002 to FR-004); a rolled result for the printed expression (FR-011); a non-empty problem list in a rejection (FR-005). Not enforced: Hit Dice by size (FR-014), a qualified defence's circumstance (FR-021).
- **Failures:** expected host mistakes are returned as `NewMonsterSetup` errors or as `SetUpMonsterCombatantInvalid` (a result variant, not an error); `MustMonsterSetup` panics only for programmer error; a `*srderr.PortError` for port `dice` is the only port failure.
- **Domain service:** `MonsterSetup` itself is the pure service; no other class is needed. The Proficiency Bonus lookup and the ability modifier are owned by `content-monsters.md`.
- **Files:** one package, `combat`. `monster_setup*.go`: all value objects, silent. `set_up_monster_combatant.go`: `SetUpMonsterCombatant`, `SetUpMonsterCombatantCommand`, `SetUpMonsterCombatantResult`, the only file touching `telemetry.Telemetry`.

## 6. Interface

```go
func NewMonsterSetup(m content.Monster, hp ResolvedMonsterHitPoints, opts ...SetupOption) (MonsterSetup, error)
func MustMonsterSetup(m content.Monster, hp ResolvedMonsterHitPoints, opts ...SetupOption) MonsterSetup

// Options; defaults: size absent, DiesInstantly, MonsterInitiativeRolled, no extra affinities.
func WithSize(s CreatureSize) SetupOption
func WithZeroHitPointsRule(r ZeroHitPointsRule) SetupOption
func WithInitiativeMode(m MonsterInitiativeMode) SetupOption
func WithExtraAffinities(a DamageAffinities) SetupOption

func (s MonsterSetup) Ref(id CombatantID) CombatantRef
func (s MonsterSetup) Profile(id CombatantID) CombatantProfile
func (s MonsterSetup) SavingThrow(a ability.Ability) d20.RollModifiers
func (s MonsterSetup) SkillCheck(sk content.Skill) d20.RollModifiers
func (s MonsterSetup) AbilityCheck(a ability.Ability) d20.RollModifiers
func (s MonsterSetup) Equal(o MonsterSetup) bool

// *MonsterSetupRejected holds a non-empty []MonsterSetupProblem; it matches srderr.ErrInvalidInput.
type MonsterSetupProblem int // SizeChoiceRequired, SizeNotOffered, RolledExpressionMismatch

type SetUpMonsterCombatant struct{ /* unexported */ }
func NewSetUpMonsterCombatant(r dice.Roller, opts ...Option) SetUpMonsterCombatant // Option: WithTelemetry (zero value = Noop)
func (u SetUpMonsterCombatant) Do(ctx context.Context, c SetUpMonsterCombatantCommand) (SetUpMonsterCombatantResult, error)
```

Every use-case type in this spec exposes one exported entry point, `Do`; `SetUpMonsterCombatantResult` is a closed variant set (interface with an unexported marker method; variants `Applied` and `Invalid`, prefixed in code as listed in section 4). The `error` is non-nil only for a port failure or a port that breaks its contract.

- **Synchronous:** `dice.Roller` is synchronous and takes no `context.Context` (`dice-and-rolls.md` FR-045); the use case takes `ctx` first only because it starts a span (decision 2b).
- **Port failure contract:** a `*srderr.PortError` for port `dice` only (FR-040). No new port, no double: `dicetest.FixedRoller` and the recording `telemetrytest` double suffice.
- **Driving it:** per creature: call `SetUpMonsterCombatant`, take `setup.HitPointStatus()`, `setup.Conditions()` and `setup.Ref(id)` to build a `Combatant`, `setup.Profile(id)` and the initiative plan for `RollInitiative` or `EstablishInitiativeOrder`, and the `SavingThrow`, `SkillCheck` and `AbilityCheck` modifiers for the `d20` resolver. The host stores the setup if it wants the unapplied lists.
- **Exports:** the types of section 5 and the three use-case types from package `combat` (no facade or re-export package).

## 7. Observability

`MonsterSetup` is **silent by design** (observability §3): pure, deterministic, nothing to count or time. Only the use case emits, through the `combat` telemetry helper over `telemetry.Telemetry`. Evaluation (observability §2):

1. **What can go wrong?** A dice adapter that returns an error or the wrong expression; a host that omits or mis-states the size; rolled totals that fall to 0 (the "minus" blocks) hiding in the clamp.
2. **How would an operator know?** Port failure: `srd5e.combat.errors` with `error.type` `dice_roller_error` plus span error status. Size mistakes: `operations` with `result_class` `invalid`. Clamping: the counter below.
3. **To diagnose:** the method attribute and the result value; Hit Point values and ids stay in the returned setup (FR-051).
4. **Count or time:** the shared signals; the clamp.
5. **Existing instrumentation:** the four shared signals are unchanged (`combat-foundations.md` FR-037); amendment A-3 adds the one extra key to their Attributes cell.
6. **Logs:** none (`combat-foundations.md` FR-034).
7. **Noise:** none; one counter with no attributes.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.set_up_monster_combatant` | span | `srd5e.combat.operation`, `srd5e.outcome`, `srd5e.combat.result` (`set_up_average`, `set_up_rolled`, `size_choice_required`, `size_not_offered`), `srd5e.combat.result_class` (`applied`, `invalid`), `srd5e.combat.monster_hit_points_method` (`average`, `rolled`) | The path of one setup | Sample traces; alert on error-status spans (a port failure or a sustained rise in `invalid`) |
| `srd5e.combat.monster_hit_points_clamped` | counter `{roll}` | none | Rolled Hit Point totals raised to 1 (FR-010) | Expect about a quarter of rolls for a `1d4 − 1` block (a d4 face of 1) and none for blocks with a positive minimum; a rate far from that over 200 or more rolls points at the roller adapter |

Signal constants (`CombatSignals`, `combat-foundations.md` FR-039, FR-040): `SetUpMonsterCombatantOperationValue`, `SetUpMonsterCombatantSpan`, `MonsterHitPointsClampedMetric`, `MonsterHitPointsMethodAttribute`, `MonsterHitPointsMethodAverageValue`, `MonsterHitPointsMethodRolledValue`, and the four result values `SetUpAverageValue`, `SetUpRolledValue`, `SizeChoiceRequiredValue`, `SizeNotOfferedValue` (each with a doc comment naming `srd5e.combat.result`). These rows feed `doc/observability/signals.md`.

## 8. Test plan

Unit tests live in `combat/monster_setup_test.go` and its siblings (package `combat_test`; unexported edge logic in `package combat` files), use-case tests in `combat/set_up_monster_combatant_test.go`. Stat-block text used as input lives only in `combat/testdata/srd/` (`content-monsters.md` FR-092); hand-built monsters use synthetic names. Tests cite their FR id.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-006 | unit | `monster_setup_creation_test.go`: single size defaults, chosen size, the five size rows (Aboleth Large; Assassin Medium or Small with none, Small, Large; Aboleth Medium), problems in enum order and each once, `MustMonsterSetup` panics iff `NewMonsterSetup` returns an error, a `*MonsterSetupRejected` is never empty, `ref` kind and id, equality over each field, copied slices |
| FR-007 to FR-014 | unit | `monster_hit_points_test.go`: table 4.2a, closed type switch exhaustive, current equals maximum and Alive, no Temporary Hit Points, `hitPointsRaisedToOne` both ways, wrong count/die/modifier each rejected, a Gargantuan block keeps a non-d20 die, two setups of one block with different rolls differ |
| FR-015 to FR-017 | unit | `zero_hit_points_rule_test.go`: default `DiesInstantly`; `FallsUnconscious` kept; the resulting status Dead on 0 damage versus Dying (through `applyDamage`); same default for CR 0 and CR 30 |
| FR-018 to FR-024 | unit | `monster_defenses_test.go`: table 4.4a; damage and condition entries never cross; duplicate and Resistance-with-Vulnerability cases; unapplied order; `extraAffinities` union |
| FR-025 to FR-028 | unit, architecture | `monster_initiative_plan_test.go`: table 4.5a, default mode, exhaustive type switch; architecture: no `RollInitiative`, `dice.Roller` or `math/rand` use in the pure setup files |
| FR-029 to FR-033 | unit | `monster_modifiers_test.go`: table 4.6a, PB for all 34 CRs by band, property over every ability and listed skill of the fixture blocks (slots sum to the printed value), at most one PB slot |
| FR-034 to FR-036 | unit | `monster_speeds_profile_test.go`: table 4.7a; profile id, size, speeds |
| FR-037 to FR-044 | application | `set_up_monster_combatant_test.go` with `dicetest.FixedRoller`, `dicetest.CallbackRoller` and the recording `telemetrytest` double: average makes no request; roll makes one request for the printed expression (`requests`); size problem consumes no die (`remaining` unchanged); an error from the roller, a wrong-expression result and a total inconsistent with its faces (`invalidResult`) are returned wrapped by `srderr.WrapPort` (`errors.Is` finds the original) with one `errors` increment, no result; equal results with no-op and recording `telemetrytest` double; `invalid` has error status and no `errors` increment; no event counted; mutating a returned slice does not change the result |
| FR-045 to FR-047 | architecture | import allowlist, exported-name scan (exports, godoc, generic names), consumer-name scan, no `math/rand` or `time.Now` in the files |
| FR-048 to FR-051 | application, architecture | result values per table 4.10a; span name; the method extra on span and `operations` only; the clamp counter increments once on the Bat row and not on the Giant Rat or average rows; no attribute equals a supplied id or Hit Point value; catalogue rows and `CombatSignals` constants match |
| errors-and-results.md FR-031, FR-034, FR-051 | application | The port failure of FR-040 is returned as a `*srderr.PortError` wrapping the original, with no result; a recorded failure sets `error.type` from `srderr.ErrorType(err)`; a panicking roller is recovered by `srderr.Guard` into a `*srderr.PortError`. |

- **Doubles and contract suites:** none new.
- **Mutation testing:** the `monster_setup*.go` files of package `combat` with gomu run only inside Docker with `--cpus` and `--workers` capped at 1/4 of `nproc` (`doc/rules/testing.md` §8; never run locally), watching `< 1` versus `<= 1` in FR-009, the three-way split of FR-030 and FR-031 (`d == 0`, `== PB`, `== 2 * PB`), and the qualified check in FR-021. `SetUpMonsterCombatant` SHOULD be mutation tested.
- **Not covered:** nothing.

## 9. Open questions and decisions

Settled, one line each. Open: none. Escalated: none.

- Q-1 (rolled total of 0 or less): raised to 1, flagged (FR-009, FR-010). The SRD says Hit Point maximum 0 means death (p. 17) and offers no floor for a rolled block; the SRD itself prints "1 (1d4 − 1)" blocks. Web research found the usual ruling is a minimum of 1 Hit Point (community consensus, no official text): interpretation, and the least surprising reading. Flagged by `hitPointsRaisedToOne` and a counter so a host can choose differently.
- Q-2 (default `ZeroHitPointsRule`): `DiesInstantly` for a monster (p. 17); the host overrides per monster. `hit-points-and-damage.md` Q-9 (no default on `HitPointStatus`) stands; this default is on the setup argument.
- Q-3 (qualified and described defences): reported, not applied (FR-021, FR-022); `extraAffinities` (FR-024) lets the host apply what holds. Package design: a general extension point, no consumer name.
- Q-4 (how to split a printed modifier): ability plus one Proficiency Bonus slot, doubled for skills, otherwise `others` (FR-030, FR-031). Printed totals win; the labels are interpretation of p. 255 ("Other modifiers might apply").
- Q-5 (fixed Initiative): the printed score, never "10 plus modifier" (FR-027); two SRD blocks print a score five above it.
- Q-6 (several identical creatures): separate setups and separate Hit Point rolls (FR-013); Initiative sharing stays with `initiative-and-actions.md` FR-008.
- Q-7 (qualified Speeds): not applied; walk 0 when no unqualified walk (FR-034, FR-035).
- Q-8 (why not depend on `encounter-aggregate.md`): the candidate listed it, but it is rank 28 and builds `Combatant` from these inputs; FR-056 forbids a higher-rank dependency. `Combatant` takes a `HitPointStatus` and `ConditionSet` already (`encounter-aggregate.md` section 3); this spec supplies them (A-2).
- Q-9 (wrong-size choice): the engine never picks a size (FR-003); `invalid`, not `refused`.
- Q-10 (Hit Dice by Size): not enforced (FR-014).

Amendments to other documents, required when this spec is accepted (this spec does not edit them):

| Id | Target | Edit |
|---|---|---|
| A-1 | `shared-types-and-ownership.md` 4.11; `documentation_index.md` | Add `monster-combatant-setup` to the rank-26 row; add an index entry for `specs/monster-combatant-setup.md` (stat block to per-creature fight inputs; read when you set up a monster for an encounter). |
| A-2 | `encounter-aggregate.md` section 3 | Name this spec as the source of the `HitPointStatus` and `ConditionSet` a `Combatant` takes for a monster (`MonsterSetup.HitPointStatus`, `conditions`). |
| A-3 | `combat-foundations.md` (signal Attributes cell) and `doc/observability/signals.md` | Add the extra key `srd5e.combat.monster_hit_points_method` to the shared signals' attributes; add the two section 7 rows (marked pending until acceptance). |

## 10. Changelog impact

`Added` (combat): `MonsterSetup`, `SetUpMonsterCombatant` and its command and result, `MonsterHitPointsMethod`, `ResolvedMonsterHitPoints`, `MonsterInitiativePlan`, `UnappliedDefense`, `UnappliedSpeed`, and the signals `srd5e.combat.set_up_monster_combatant` and `srd5e.combat.monster_hit_points_clamped`. Not **BREAKING**: nothing is released.
