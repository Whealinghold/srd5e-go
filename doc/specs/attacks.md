# Attacks

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md`, `grappling.md`, `d20-tests.md`, `resolve-d20-test-use-case.md`, `derived-stats.md`, `content-equipment.md`, `content-schema-and-decoder.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `ability-scores.md`, `errors-and-results.md`, `observability-ports.md` |

`grappling.md` is a real dependency: this spec imports the pure `GrappleRules` and
`HandAvailability` (FR-128). `attack-and-help-actions.md` is related by reference only (no import).
`conditions.md` and `resolve-d20-test-use-case.md` are imported for published
value types only (`AutomaticCritical`; attempt types and the `d20` signal constants keys).

## 1. Purpose and scope

Defines how the `combat` context resolves one attack: Cover, range and reach,
the circumstances that give Advantage or Disadvantage, the attack roll made with
the pure `d20.Resolver` of `d20`, hit, miss or Critical Hit, and the
attack's damage (with the Critical Hit rule). It also defines the Unarmed Strike
and its three options (Damage, Grapple, Shove) as an attack.

The engine is state in, result out (`doc/rules/architecture.md` §4). The host
supplies the facts the GM decides (cover, sight, distance, the target's Armor
Class). This spec never decides line of sight, geometry or hiddenness.

**Ownership split (decided).**

- `d20` owns the attack roll arithmetic, `Roll`, `RollModifiers`,
  `d20.Outcome`, `AttackRollResult` and `d20.Resolver` (shared-types-and-ownership.md
  FR-002, FR-004; `d20-tests.md` FR-069). No use case calls another
  (shared-types-and-ownership.md FR-044): `ResolveAttack` and `ResolveUnarmedStrike`
  roll through their own `dice.Roller`, hand the `Roll` and effective AC to
  `d20.Resolver` and record only their own signals. No parallel outcome enum.
- This spec owns cover (including the Dexterity save bonus,
  shared-types-and-ownership.md FR-028), range, reach, circumstances, forced
  misses, the automatic Critical Hit input, damage rolling and the Critical Hit dice
  rule, and the Unarmed Strike attack side: option choice, Cover check (FR-093),
  the target's save with its Cover bonus (FR-092, FR-094, FR-095) and the result
  (FR-077). The Shove option is wholly this spec's (FR-073 to FR-075).
- `grappling.md` owns the Grapple option's DC, the saves the target may choose, the
  size and free-hand checks (`GrappleRules`: `GrappleDC` FR-094, `OptionSaveAbilities`
  FR-095, `possibility` FR-096; shared-types-and-ownership.md FR-029, FR-048), the
  grapple record, escape, movement and end of a grapple. A succeeded Grapple option
  is handed by the caller to its `BeginGrapple` (`grappling.md` FR-066, FR-022).
  The Grappled condition entry is `conditions.md`'s.
- `attack-and-help-actions.md` owns the Attack action, Extra Attack and the Help
  action (shared-types-and-ownership.md FR-048). This spec resolves one attack
  however granted and never counts it against an allowance.
- Pure with respect to conditions (shared-types-and-ownership.md FR-026): it takes
  `RollModifiers`, circumstances and `AutomaticCritical` already computed
  (FR-119).
- `hit-points-and-damage.md` owns `Damage`, `HitKind`, `DamageResolver`; `content`
  owns `DamageType` and `CreatureSize` (shared-types-and-ownership.md FR-005,
  FR-046). This spec's own service is `AttackDamageComposer`. It defines no
  `Proficiency` type (shared-types-and-ownership.md FR-027).
- `combat-foundations.md` owns the combat conventions (FR-022 to FR-037): naming,
  the combat signal-name constants, the shared use-case recorder, the shared signals (§7).

**Out of scope:**

- Applying damage, Resistance, Vulnerability, Immunity, Bloodied, 0 Hit Points.
- Which conditions exist and what they do (`conditions.md`; SRD 5.2.1, "Rules
  Glossary", pp. 176-191). The caller passes the consequences in: circumstances
  (FR-044), the automatic Critical Hit (FR-090), automatic failure or Disadvantage on
  a save as the `SaveAttempt` (FR-094, FR-095).
- The Attack action, Extra Attack, Help (`attack-and-help-actions.md`); Bonus Action
  and Reaction economy and Opportunity Attacks (`initiative-and-actions.md`; SRD
  5.2.1, "Rules Glossary", "Attack [Action]", p. 177, "Opportunity Attacks", p. 185).
- The grapple itself and `GrappleEnded` (SRD 5.2.1, "Grappling", p. 182).
- Knocking Out a Creature (SRD 5.2.1, p. 17): this spec only tells the host whether
  the attack was melee (FR-111) so it can choose `MeleeAttackKnockOut`
  (`hit-points-and-damage.md` FR-050).
- Weapon data and properties (Weapons table, mastery, Ammunition, Loading, Light,
  Heavy, Versatile, Two-Handed). The caller supplies ability modifier, damage dice,
  range and reach; which ability a weapon uses and the same-modifier rule are
  `weapon-properties-and-mastery.md` FR-031 to FR-033 (FR-007, FR-053).
- Armor Class, Proficiency Bonus, ability modifiers (`derived-stats.md`,
  `ability-scores.md`); spell attack mechanics beyond the roll; area of effect,
  Mounted Combat, Falling Off, grid distance, line of sight, Hide.
- Anything not in SRD 5.2.1 (no feats, fighting styles or class features beyond the
  SRD's own example names in the Critical Hit text).

## 2. Source

- `SRD 5.2.1, "D20 Tests", p. 6`
- `SRD 5.2.1, "Attack Rolls" (Ability Modifier, Proficiency Bonus, Armor Class, Rolling 20 or 1), p. 7`; `"Saving Throws", p. 7`
- `SRD 5.2.1, "Advantage/Disadvantage", pp. 7-8`
- `SRD 5.2.1, "Creature Size", "Unseen Attackers and Targets", p. 14`
- `SRD 5.2.1, "Making an Attack", pp. 14-15` (heading p. 14, three steps p. 15)
- `SRD 5.2.1, "Cover", "Ranged Attacks", "Range", "Ranged Attacks in Close Combat", "Melee Attacks", "Reach", p. 15`
- `SRD 5.2.1, "Underwater Combat" (Impeded Weapons), "Damage Rolls", "Critical Hits", "Damage Types", p. 16` ("Damage Types" continues on p. 17)
- `SRD 5.2.1, "Knocking Out a Creature", p. 17` (hand-off only)
- `SRD 5.2.1, "Rules Glossary": "Attack Roll", p. 177; "Cover", p. 179; "Critical Hit", p. 179; "Grappling", p. 182; "Improvised Weapons", p. 183; "Reach", p. 186; "Paralyzed", p. 186; "Spell Attack", p. 188; "Unarmed Strike", p. 190; "Unconscious", p. 191; "Weapon Attack", p. 191`
- `SRD 5.2.1, "Equipment", "Weapon Proficiency", p. 89; "Range", "Reach", "Thrown", p. 90`

Printed page equals PDF page; no offset. Rule files: `doc/rules/architecture.md`
§2 to §4, §6; `observability.md` §2, §5 to §7; `testing.md` §4, §5.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| attack, attack roll | D20 Test | Against the target's AC; resolved by `d20.Resolver`, composed here, not through its use case. |
| Armor Class (AC) | `ArmorClass` | Owned by `d20` (`d20-tests.md` D-03, FR-022); input only. |
| natural 20, natural 1 | Roll fact | The d20 kept after Advantage or Disadvantage. |
| Critical Hit | `AttackRollResult.IsCriticalHit` | Natural 20, or a hit upgraded by an automatic Critical Hit (FR-090). |
| hit, miss | `d20.Outcome` | `success` (hit) or `failure` (miss); hit: total equals or exceeds AC, or natural 20; miss: otherwise or natural 1. |
| Melee, Ranged attack | `AttackDelivery` | Melee: target within reach. Ranged: target within a range. |
| weapon, spell attack, Unarmed Strike | `AttackSource` | Roll made with a weapon, as part of a spell, with the body. |
| Cover | `Cover` (closed) | `HalfCover` (+2 AC and Dexterity saves), `ThreeQuartersCover` (+5), `TotalCover` (can't be targeted directly). |
| range, normal range, long range | `AttackRange` | A single range, or normal and long, in feet. |
| reach | `Reach` | 5 feet unless a rule says otherwise. |
| Advantage, Disadvantage | `modifier.RollCircumstance`, `RollMode` | Kernel types (`modifiers-and-advantage.md`; shared-types-and-ownership.md FR-001). |
| Proficiency Bonus | `ProficiencyBonus` (`derived-stats.md`) as a kernel `Modifier` in the optional slot of `RollModifiers` | `d20` cannot import `character` (`d20-tests.md` D-16; shared-types-and-ownership.md FR-003, FR-027); the caller converts (FR-124). |
| ability modifier | `ability.Modifier` | Enters `RollModifiers` as `Modifier(abilityModifier.value)` (`ability-scores.md` FR-027; `modifiers-and-advantage.md` FR-043). |
| damage dice, modifiers | `DamageComponent` | Dice or fixed amount of one Damage Type plus its modifiers. |
| Damage Types | `DamageType`, `Damage` | Owned by `content` and `hit-points-and-damage.md`; reused. |
| improvised weapon | `WeaponUse` | Built from `content` vocabulary (`content-equipment.md` FR-101; `content-schema-and-decoder.md` FR-113) plus `WeaponUseManner`. |
| Incapacitated, Swim Speed | Facts | Inputs (a flag on `NearbyEnemy`; the underwater rule). |
| saving throw, DC | D20 Test, `DifficultyClass` | Grapple and Shove saves via `d20.Resolver.ResolveCheck`. |
| size (Tiny to Gargantuan) | `CreatureSize` | Owned by `content-schema-and-decoder.md` (FR-109); imported, ordered smallest to largest. |
| hand free, size limit | `HandAvailability`, `GrapplePossibility` | Owned by `grappling.md` (FR-096); carried and called here (FR-128). |

Package terms (none collides with an SRD term): `AttackForm` (named so as not to
collide with `BarbarianAttackKind` and `PaladinAttackKind`,
shared-types-and-ownership.md FR-021), `AttackDelivery`, `AttackSource`,
`MissOverride`, `AutomaticMissReason`, `AttackSituation`, `TargetVisibility`,
`NearbyEnemy`, `WeaponUse`, `WeaponUseManner`, `CircumstanceCollector`, `AttackPlan`
(`AttackPlanProceeds`, `AttackPlanNotPossible`, `AttackPlanAutomaticMiss`),
`CoverResolver`, `AttackDamageComposer`, `UnarmedStrikeModifiers`, `attackFlow`,
`AttackNotPossible`, `UnarmedStrikeNotPossible`, `AttackNotPossibleReason`, the two
use cases with their `<UseCase>Result` types, and the parse types of FR-107.

## 4. Requirements

Stable ids; never renumbered. A bare "FR-nnn" is this spec's; another spec's id is
written with its file name. Withdrawn ids are in section 4.6.

### 4.1 Attack roll, outcome and circumstances

The roll arithmetic is `d20`'s (`d20-tests.md` FR-069, FR-064, FR-027 to
FR-033); FR-002 to FR-011 are the mapping contract. Natural 20 and 1 maths is tested
once, in `d20`; combat tests it only through the real `d20.Resolver` with a
fixed `dice.Roller`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-002 | The use case MUST roll the d20 through its own `dice.Roller`, build the `Roll` and call `d20.Resolver.ResolveAttackRoll(roll, modifiers, armorClass)` (`d20-tests.md` FR-069, FR-064). The modifiers are the command's `RollModifiers` unchanged (ability modifier, optional Proficiency Bonus, every other `Modifier`, each once); the total MUST be the resolver's. Combat adds no arithmetic and MUST NOT call or construct the `d20` use case (shared-types-and-ownership.md FR-044). | SRD 5.2.1, "D20 Tests", p. 6; "Attack Rolls", p. 7; shared-types-and-ownership.md FR-002, FR-003 | happy |
| FR-003 | With ability modifier 0, no Proficiency Bonus, no other `Modifier` and no circumstances, the `RollModifiers` and circumstances used MUST be exactly the empty values and the total equals the natural d20. | SRD 5.2.1, "D20 Tests", p. 6 | unhappy (empty and minimal case) |
| FR-004 | The target number MUST be the effective AC (FR-015); a total above it is a hit (`d20-tests.md` FR-027). | SRD 5.2.1, "Attack Rolls", p. 7 | happy |
| FR-005 | A total equal to the effective AC is a hit (`d20-tests.md` FR-027); combat MUST map it unchanged. | SRD 5.2.1, "Attack Rolls", p. 7 | unhappy (boundary) |
| FR-006 | A total one below the effective AC with a natural d20 of neither 1 nor 20 is a miss (`d20-tests.md` FR-028) and combat MUST map it unchanged. | SRD 5.2.1, "Attack Rolls", p. 7 | unhappy (boundary) |
| FR-007 | An Unarmed Strike's ability modifier MUST default to Strength (the command field is documented as the Strength modifier, FR-129); a spell attack has no default ability (the caller supplies it inside `RollModifiers`). For a weapon attack this spec chooses no ability (`weapon-properties-and-mastery.md` FR-031, FR-032; FR-053). `AttackForm` carries no default ability. | SRD 5.2.1, "Attack Rolls", Attack Roll Abilities table, p. 7; "Unarmed Strike", p. 190 | happy |
| FR-008 | A natural 20 is a Critical Hit that hits regardless of modifiers and AC, including AC raised by cover (`d20-tests.md` FR-029), unless a critical-range input lowers the threshold (FR-134; a natural 20 stays a Critical Hit) or FR-036 or FR-041 forces a miss (FR-091). | SRD 5.2.1, "Attack Rolls", "Rolling 20 or 1", p. 7; "Critical Hit", p. 179 | unhappy (boundary: natural 20, total below AC) |
| FR-009 | A natural 1 MUST be a miss regardless of modifiers and AC (`d20-tests.md` FR-030), reported with the natural-1 reason. | SRD 5.2.1, "Attack Rolls", "Rolling 20 or 1", p. 7 | unhappy (boundary: natural 1, total at or above AC) |
| FR-010 | With no critical-range input (FR-134), only a natural 20 is a Critical Hit by the roll: a natural 19 that hits is an ordinary hit (`d20-tests.md` FR-033). | SRD 5.2.1, "Rolling 20 or 1", p. 7; "Critical Hit", p. 179 | unhappy (boundary: one below 20) |
| FR-011 | Combat MUST resolve the roll mode once with `RollMode.resolve` (`modifiers-and-advantage.md` FR-010 to FR-017), roll `mode.diceCount` d20s, build the `Roll` (`d20-tests.md` FR-055; shared-types-and-ownership.md FR-002) and MUST NOT re-select the die ("natural" is the kept d20, `d20-tests.md` FR-032). Advantage with dice 1 and 20 is a Critical Hit; Disadvantage with 1 and 20 is a miss. | SRD 5.2.1, "D20 Tests", p. 6; "Roll Two D20s", p. 8 | unhappy (rule interaction) |
| FR-134 | `ResolveAttackCommand` and the Damage option of `ResolveUnarmedStrikeCommand` MUST each accept an optional `CriticalThreshold` (whole number 18 to 20, default 20) and pass it unchanged to the `d20` attack roll resolver's critical-range input (an amendment `d20-tests.md` takes from `class-fighter.md`). A kept d20 at or above it hits and is a Critical Hit regardless of modifiers and AC (so FR-056 doubles the dice); a natural 20 always is; a natural 1 is still a miss; FR-036 and FR-041 still win (FR-091). A caller may supply `ChampionEffects.CriticalHitThreshold` (`class-fighter.md` FR-087, FR-090) for weapon attacks and Unarmed Strikes; this spec reads no class feature and applies no value of its own (`doc/rules/architecture.md` §3). A value outside 18 to 20 is a validation error naming `CriticalThreshold` (FR-033), a `NewResolveAttackCommand` problem field (FR-132). Interpretation: "Critical Hit" is defined as hitting regardless of AC, so a threshold roll hits at any total. | SRD 5.2.1, "Rolling 20 or 1", p. 7; "Critical Hit", p. 179; "Level 3: Improved Critical", p. 49 | unhappy (boundary: natural 19 with and without the input) |
| FR-012 | The result MUST carry the breakdown: dice rolled, d20 kept, each modifier, AC, cover bonus, effective AC, circumstances and outcome (the `AttackRollResult` unchanged, with any `MissOverride`, FR-121). | `doc/rules/architecture.md` §6 (package design) | happy |
| FR-034 | When the attacker can't see the target (including guessing its location or hearing it), the attack roll MUST have Disadvantage. | SRD 5.2.1, "Unseen Attackers and Targets", p. 14 | unhappy (rule interaction) |
| FR-035 | When the target can't see the attacker, the attack roll MUST have Advantage. | SRD 5.2.1, "Unseen Attackers and Targets", p. 14 | unhappy (rule interaction) |
| FR-036 | When the attacker can't see the target and the target is not in the targeted location, the outcome MUST be a miss whatever the roll. The SRD says only "you miss"; that the roll is still made and reported is a package decision (the attacker does not know the target moved, so the same flow runs). | SRD 5.2.1, "Unseen Attackers and Targets", p. 14 (for the miss) | unhappy (state error) |
| FR-037 | When the attacker is hidden, the result MUST flag that the attacker's location is given away, for a hit and a miss alike. | SRD 5.2.1, "Unseen Attackers and Targets", p. 14 | unhappy (rule interaction) |
| FR-038 | When both FR-034 and FR-035 apply, the roll MUST end with neither Advantage nor Disadvantage, one d20. | SRD 5.2.1, "They Don't Stack", p. 8 | unhappy (rule interaction) |
| FR-039 | A melee weapon attack roll underwater by a creature without a Swim Speed MUST have Disadvantage unless the weapon deals Piercing damage. | SRD 5.2.1, "Underwater Combat", "Impeded Weapons", p. 16 | unhappy (rule interaction) |
| FR-040 | A melee weapon attack roll underwater MUST NOT get that Disadvantage when the attacker has a Swim Speed or the weapon deals Piercing damage. A spell attack underwater, melee or ranged, MUST NOT get it. | SRD 5.2.1, "Underwater Combat", "Impeded Weapons", p. 16 | unhappy (boundary) |
| FR-041 | A ranged weapon attack underwater against a target beyond normal range (but not beyond long range, FR-025) MUST be a miss with no attack roll and no dice. The SRD gives the outcome ("automatically misses"); that no roll is made is a package decision (the outcome follows from host facts, a roll could change nothing). | SRD 5.2.1, "Underwater Combat", "Impeded Weapons", p. 16 (for the miss) | unhappy (rule interaction) |
| FR-042 | A ranged weapon attack roll underwater against a target within normal range MUST have Disadvantage, with or without a Swim Speed. | SRD 5.2.1, "Underwater Combat", "Impeded Weapons", p. 16 | unhappy (rule interaction) |
| FR-043 | The use case MUST pass every Advantage and Disadvantage it determines (FR-024, FR-027, FR-034, FR-035, FR-039, FR-042) to the normal combination rule (several of one kind count once; both kinds cancel). | SRD 5.2.1, "They Don't Stack", p. 8 | unhappy (rule interaction) |
| FR-044 | The command MUST accept additional caller-supplied `modifier.RollCircumstance`s (computed with `ConditionEffects`) and combine them with FR-043's; it never queries conditions (FR-119). | SRD 5.2.1, "Making an Attack", step 2, pp. 14-15; shared-types-and-ownership.md FR-026 | happy |
| FR-045 | `AttackNotPossible` (beyond reach, beyond long range, behind Total Cover) MUST be decided before any die is rolled and take precedence over an automatic miss. | SRD 5.2.1, "Range", p. 15; "Cover", p. 15 | unhappy (rule interaction) |
| FR-090 | The command MUST accept the `AutomaticCritical` input: the `conditions.md` enum (`automatic`, `NotAutomatic`) already computed by the caller (shared-types-and-ownership.md FR-026). When `automatic`, an attack that hits MUST be a Critical Hit before damage is rolled, so the dice are doubled (FR-056); a miss, an automatic miss and `AttackNotPossible` are unchanged. `combat-action-resolution.md` derives it with `ConditionEffects.CriticalHit(target, attackerDistanceFeet)` (`conditions.md` FR-077 Paralyzed, FR-103 Unconscious: attacker within 5 feet); this spec does not. | SRD 5.2.1, "Paralyzed", p. 186; "Unconscious", p. 191; "Critical Hit", p. 179 | unhappy (rule interaction) |
| FR-091 | A forced miss (FR-036, FR-041) MUST win over a natural 20: a miss, no damage; where the roll was made (FR-036) the d20 is still reported. | SRD 5.2.1, "Unseen Attackers and Targets", p. 14; "Underwater Combat", p. 16 | unhappy (rule interaction: natural 20 against a vacated location) |
| FR-103 | "The weapon deals Piercing damage" (FR-039, FR-040) MUST be true exactly when at least one component of the attack's `DamageSpec` is Piercing, false with no `DamageSpec`; `AttackSituation` has no Piercing flag. | SRD 5.2.1, "Underwater Combat", p. 16 | unhappy (rule interaction) |
| FR-104 | Whether the attacker can see the target MUST be the closed `TargetVisibility`: `TargetSeen` or `TargetUnseen(targetAtTargetedLocation)`; "seen" with "not at the targeted location" cannot be built. | SRD 5.2.1, "Unseen Attackers and Targets", p. 14 | unhappy (invalid input) |
| FR-113 | An Unarmed Strike is not a weapon attack: it MUST NOT get the Impeded Weapons Disadvantage underwater, with or without a Swim Speed. (Package reading: the rule says "with a weapon", p. 16, and "Weapon Attack" is defined as made with a weapon, p. 191.) | SRD 5.2.1, "Underwater Combat", p. 16; "Weapon Attack", p. 191 | unhappy (rule interaction) |

### 4.2 Cover, range and reach

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-013 | Half Cover MUST give +2 to AC and to Dexterity saving throws, Three-Quarters Cover +5 to both; each `Cover` MUST expose both bonuses. The Dexterity bonus is used by the Grapple and Shove saves (FR-092). | SRD 5.2.1, "Cover", p. 15; "Cover", p. 179 | happy |
| FR-014 | Total Cover MUST have no AC or Dexterity save bonus, because the target can't be targeted directly. | SRD 5.2.1, "Cover", p. 15 | happy |
| FR-015 | The effective AC MUST be the target's AC plus the AC bonus of the applicable cover. | SRD 5.2.1, "Cover", p. 15 | happy |
| FR-016 | A target with no applicable cover MUST have its AC unchanged. | SRD 5.2.1, "Cover", p. 15 | unhappy (empty and minimal case) |
| FR-017 | Only the most protective applicable cover counts; degrees are never added (Half plus Three-Quarters is Three-Quarters; two Half is +2). Order: Total over Three-Quarters over Half. | SRD 5.2.1, "Cover", p. 15; "Cover", p. 179 | unhappy (rule interaction) |
| FR-018 | A cover source counts only when the attack originates on the opposite side of it; others MUST be ignored. | SRD 5.2.1, "Cover", p. 15 | unhappy (rule interaction) |
| FR-019 | With applicable Total Cover the attack MUST NOT be made: `AttackNotPossible` with reason Total Cover, no dice. | SRD 5.2.1, "Cover", p. 15 | unhappy (state error) |
| FR-020 | A natural 20 against Half or Three-Quarters Cover MUST still be a Critical Hit. | SRD 5.2.1, "Rolling 20 or 1", p. 7; "Cover", p. 15 | unhappy (rule interaction) |
| FR-021 | A ranged attack with a single range MUST be possible within that range (distance up to and including it). | SRD 5.2.1, "Range", p. 15 | happy |
| FR-022 | A single-range ranged attack beyond it MUST give `AttackNotPossible` (beyond range), no dice. | SRD 5.2.1, "Range", p. 15 | unhappy (boundary) |
| FR-023 | A ranged attack with normal and long range, at a distance up to and including normal range, MUST NOT get Disadvantage from range. | SRD 5.2.1, "Range", p. 15; "Range", p. 90 | happy |
| FR-024 | A ranged attack beyond normal range and up to and including long range MUST get Disadvantage. | SRD 5.2.1, "Range", p. 15; "Range", p. 90 | unhappy (boundary) |
| FR-025 | A ranged attack beyond long range MUST give `AttackNotPossible` (beyond range), no dice. | SRD 5.2.1, "Range", p. 15; "Range", p. 90 | unhappy (boundary) |
| FR-026 | A ranged attack at distance 0 MUST be treated as within normal range. | SRD 5.2.1, "Range", p. 15 | unhappy (boundary: zero) |
| FR-027 | A ranged attack roll (weapon, spell or other) MUST get Disadvantage when the attacker is within 5 feet of an enemy who can see the attacker and is not Incapacitated. | SRD 5.2.1, "Ranged Attacks in Close Combat", p. 15 | unhappy (rule interaction) |
| FR-028 | The close-combat Disadvantage MUST NOT apply when no enemy meets all three conditions (within 5 feet, can see the attacker, not Incapacitated); an enemy at exactly 5 feet counts. | SRD 5.2.1, "Ranged Attacks in Close Combat", p. 15 | unhappy (boundary) |
| FR-029 | A melee attack MUST NOT get the close-combat Disadvantage or the long-range rules. | SRD 5.2.1, "Melee Attacks", "Ranged Attacks in Close Combat", p. 15 | unhappy (rule interaction) |
| FR-030 | A melee attack MUST be possible within the attacker's reach, default 5 feet. | SRD 5.2.1, "Reach", p. 15; "Reach", p. 186 | happy |
| FR-031 | A melee attack beyond reach MUST give `AttackNotPossible` (beyond reach), no dice; a distance equal to reach is possible, one foot beyond is not. | SRD 5.2.1, "Reach", p. 15; "Reach", p. 186 | unhappy (boundary) |
| FR-032 | A supplied reach greater than 5 feet MUST replace the default for a weapon attack or melee spell attack; an Unarmed Strike ignores any supplied reach (FR-106). | SRD 5.2.1, "Reach", p. 15; "Reach", p. 90 | happy |
| FR-033 | Every host-supplied number MUST be whole feet or a whole count, validated by its `NewX` constructor, which returns a validation error (`errors.Is` `srderr.ErrInvalidInput`) carrying the field as a typed value (FR-107): range (single, or normal and long) positive, long larger than normal, reach positive, distance and every `NearbyEnemy.distance` not negative, `FixedDamage.amount` not negative, dice count at least 1, a die face count `dice-and-rolls.md` accepts. | `doc/rules/architecture.md` §3, §6 (package design) | unhappy (invalid input) |
| FR-087 | An `AttackForm` MUST be one of five legal pairs of delivery and source: `MeleeWeaponAttack`, `RangedWeaponAttack`, `MeleeSpellAttack`, `RangedSpellAttack`, `UnarmedStrikeAttack` (always melee); each exposes delivery and source. | SRD 5.2.1, "Melee Attacks", "Ranged Attacks", p. 15; "Spell Attack", p. 188 | happy |
| FR-088 | A melee spell attack MUST use the reach rules (FR-030 to FR-032), no close-combat Disadvantage or range rules (FR-029), and no underwater Disadvantage (FR-040). | SRD 5.2.1, "Melee Attacks", p. 15 ("A few spells also involve melee attacks") | unhappy (rule interaction) |
| FR-089 | A ranged spell attack MUST use the range rules (FR-021 to FR-026) and the close-combat Disadvantage (FR-027) and MUST NOT get the underwater Disadvantage or automatic miss (FR-040 to FR-042 are weapons only). | SRD 5.2.1, "Ranged Attacks in Close Combat", p. 15; "Underwater Combat", p. 16 | unhappy (rule interaction) |
| FR-105 | When several reasons make an attack impossible, the one reported MUST follow this order: Total Cover (FR-019, FR-093), beyond range or reach (FR-022, FR-025, FR-031, FR-064), target too large, no free hand (the last two in the order of `GrappleRules.possibility`, `grappling.md` FR-096, mapped by FR-128). | `doc/rules/architecture.md` §6 (package design) | unhappy (rule interaction) |

### 4.3 Weapons, proficiency and damage

A `DamageSpec` is a list of `DamageComponent`s. Modifiers belong to components only
(FR-097); the command's one ability modifier goes to exactly one flagged component
(FR-096); no attack-level modifier exists (FR-098).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-046 | An improvised weapon attack roll MUST NOT include the Proficiency Bonus: a command whose `WeaponUse` is improvised and whose `RollModifiers` carry one MUST be rejected by the command constructor with a validation error (`ImprovisedProficiencyBonus`, FR-132). "Weapon Equivalents" is a non-improvised `WeaponUse` (FR-100). | SRD 5.2.1, "Improvised Weapons", p. 183 | unhappy (invalid input) |
| FR-047 | An improvised weapon's default damage MUST be 1d4, type supplied by the caller (the GM chooses). | SRD 5.2.1, "Improvised Weapons", p. 183 | happy |
| FR-048 | A thrown improvised weapon's default range MUST be normal 20 feet, long 60 feet. | SRD 5.2.1, "Improvised Weapons", p. 183 | happy |
| FR-049 | `WeaponUse.IsImprovised` MUST be derived, never supplied: true when `category` is the zero value (none), when `manner` is `MeleeAttackWithRanged`, or when `manner` is `ThrownAttack` for a Melee weapon whose `properties` lack Thrown; false for `WieldedAsDesigned` and a thrown Melee weapon with Thrown. `MeleeAttackWithRanged` on a non-Ranged `WeaponType` MUST be rejected with a validation error. | SRD 5.2.1, "Improvised Weapons", p. 183; "Weapon", p. 191; "Thrown", p. 90 | unhappy (boundary) |
| FR-050 | A weapon attack's proficiency MUST be expressed by the caller through the optional Proficiency Bonus of `RollModifiers` (shared-types-and-ownership.md FR-003, FR-027), added only when supplied, exactly once. | SRD 5.2.1, "Weapon Proficiency", p. 89; "Attack Rolls", "Proficiency Bonus", p. 7 | happy |
| FR-100 | A `WeaponUse` supplied as non-improvised is a Simple or Martial weapon's use whatever the object ("Weapon Equivalents"); only then may its `RollModifiers` carry a Proficiency Bonus. | SRD 5.2.1, "Improvised Weapons", "Weapon Equivalents", p. 183 | unhappy (rule interaction) |
| FR-101 | A spell attack MUST always add the Proficiency Bonus: a spell attack command whose `RollModifiers` carry none MUST be rejected by the command constructor with a validation error (`SpellAttackProficiencyBonus`, FR-132). | SRD 5.2.1, "Attack Rolls", "Proficiency Bonus", p. 7 | unhappy (invalid input) |
| FR-051 | On a hit the use case MUST roll the damage dice of every component, add that component's modifiers and report a `Damage(amount, type)` per component. | SRD 5.2.1, "Making an Attack", step 3, p. 15; "Damage Rolls", p. 16 | happy |
| FR-052 | On a miss, an automatic miss or `AttackNotPossible`, no damage dice MUST be rolled and no damage reported. | SRD 5.2.1, "Making an Attack", step 3, pp. 14-15 | unhappy (state error) |
| FR-053 | The ability modifier added to damage (to the component flagged by FR-096) MUST be the supplied `RollModifiers.ability`, the very value used for the attack roll, unchanged; damage has no ability modifier of its own. Weapon-side rules: `weapon-properties-and-mastery.md` FR-031 to FR-033. | SRD 5.2.1, "Damage Rolls", p. 16; "Finesse", p. 89 | happy |
| FR-054 | A fixed damage amount MUST NOT get the ability modifier unless it is the flagged component (for example the Unarmed Strike, FR-067). | SRD 5.2.1, "Damage Rolls", p. 16 | unhappy (rule interaction) |
| FR-055 | A component's total MUST NOT be negative: a penalty below 0 gives 0 damage, floored per component (equal to the SRD floor because a penalty is attached to one component, FR-097, FR-098). | SRD 5.2.1, "Damage Rolls", p. 16 | unhappy (boundary) |
| FR-056 | On a Critical Hit every damage die of the attack MUST be rolled twice and added, then modifiers added once. | SRD 5.2.1, "Critical Hits", p. 16; "Critical Hit", p. 179 | happy |
| FR-057 | On a Critical Hit, dice from other sources in the same attack (for example Sneak Attack, which the SRD names) MUST also be rolled twice; they are supplied as additional dice components. | SRD 5.2.1, "Critical Hits", p. 16 | unhappy (rule interaction) |
| FR-058 | On a Critical Hit modifiers MUST NOT be doubled. | SRD 5.2.1, "Critical Hits", p. 16 | unhappy (boundary) |
| FR-059 | On a Critical Hit a fixed amount with no dice MUST be unchanged (package reading: no dice to roll twice; the SRD says no more). | SRD 5.2.1, "Critical Hits", p. 16 | unhappy (boundary) |
| FR-060 | On a hit that is not a Critical Hit, damage dice MUST be rolled once. | SRD 5.2.1, "Damage Rolls", p. 16 | happy |
| FR-061 | Each component MUST carry its `DamageType` to the result. | SRD 5.2.1, "Damage Types", p. 16 | happy |
| FR-062 | A `DamageSpec` MUST have at least one component (empty is rejected by `NewDamageSpec` with a validation error). An attack that deals no damage is modelled by passing no `DamageSpec` at all. | SRD 5.2.1, "Making an Attack", step 3, p. 15 | unhappy (empty and minimal case) |
| FR-096 | At most one component of a `DamageSpec` MAY be flagged `AddsAbilityModifier`, and it alone receives the ability modifier, once; two or more flagged are rejected by `NewDamageSpec` with a validation error naming the field (`DamageSpecField` `AddsAbilityModifier`); none flagged adds nothing. Which component is a package decision (the SRD names the floor, not how to split a modifier between types). | SRD 5.2.1, "Damage Rolls", p. 16 | unhappy (invalid input) |
| FR-097 | A component's flat modifiers MUST attach to that component only; there is no attack-level damage modifier (package decision). | SRD 5.2.1, "Damage Rolls", p. 16 | unhappy (rule interaction) |
| FR-098 | A negative flat modifier on one component MUST NOT reduce another: with a penalty on the first of two, the second is unchanged and the first is floored at 0 alone. | SRD 5.2.1, "Damage Rolls", p. 16 | unhappy (boundary) |
| FR-114 | Each component result MUST convert to exactly one `Damage` (amount 0 or more, one `DamageType`, `hit-points-and-damage.md` FR-015, FR-086) and the result MUST expose a `HitKind` (`NormalHit`, `CriticalHit`) for a hit. | SRD 5.2.1, "Damage Types", p. 16; `hit-points-and-damage.md` FR-015, FR-086; shared-types-and-ownership.md FR-005 | happy |
| FR-111 | `AttackResolved` MUST expose the attack's delivery so the host can choose `MeleeAttackKnockOut`; this spec does not apply the knock-out. | SRD 5.2.1, "Knocking Out a Creature", p. 17 | happy |

### 4.4 Unarmed Strike

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-063 | An Unarmed Strike MUST be a melee attack against a target within 5 feet, and each use MUST choose exactly one option: Damage, Grapple or Shove (closed set). | SRD 5.2.1, "Unarmed Strike", p. 190 | happy |
| FR-064 | A target beyond 5 feet MUST give `AttackNotPossible` (beyond reach), no dice. | SRD 5.2.1, "Unarmed Strike", p. 190 | unhappy (boundary) |
| FR-065 | The Damage option MUST make an attack roll whose bonus is the ability modifier used (Strength unless the caller supplies another, FR-129) plus the Proficiency Bonus, built by `UnarmedStrikeModifiers` (FR-124). | SRD 5.2.1, "Unarmed Strike", p. 190 | happy |
| FR-066 | The Damage option's Proficiency Bonus MUST be added without a proficiency check. | SRD 5.2.1, "Unarmed Strike", p. 190 | unhappy (rule interaction) |
| FR-067 | On a hit the Damage option MUST deal Bludgeoning damage equal to 1 plus the ability modifier used (FR-129), minimum 0: a fixed component flagged as adding the ability modifier (FR-054). | SRD 5.2.1, "Unarmed Strike", p. 190; "Damage Rolls", p. 16 | unhappy (boundary) |
| FR-068 | The Damage option MUST follow the attack-roll rules of §4.1 to §4.3: natural 20 and 1, cover, Advantage and Disadvantage, automatic Critical Hit (FR-090) and the 5-foot reach check (FR-106). | SRD 5.2.1, "Unarmed Strike", p. 190; "Attack Roll", p. 177 | unhappy (rule interaction) |
| FR-069 | A Critical Hit with the Damage option MUST be a hit whose damage is unchanged (1 plus the modifier has no dice, FR-059). | SRD 5.2.1, "Critical Hits", p. 16; "Unarmed Strike", p. 190 | unhappy (boundary) |
| FR-073 | The Shove option MUST require the target to make a Strength or Dexterity saving throw (its choice); on a failure the attacker's pre-declared effect is pushing the target 5 feet or giving it Prone (the effect is in the command; a two-step flow can be added later without breaking it). | SRD 5.2.1, "Unarmed Strike", p. 190 | happy |
| FR-074 | The Shove DC MUST equal 8 plus the ability modifier used (FR-129) plus the Proficiency Bonus, no cap or floor. | SRD 5.2.1, "Unarmed Strike", p. 190 | happy |
| FR-075 | A Shove MUST be possible only when the target is no more than one size larger than the attacker (else `AttackNotPossible`, target too large); no free hand is required. | SRD 5.2.1, "Unarmed Strike", p. 190 | unhappy (boundary) |
| FR-076 | A save total at or above the DC MUST succeed (no effect) and one below MUST fail (effect applies), as decided by `d20.Resolver.ResolveCheck` (`d20-tests.md` FR-025, FR-026). | SRD 5.2.1, "D20 Tests", p. 6 | unhappy (boundary) |
| FR-077 | Grapple and Shove results MUST describe the effect (Grappled, Prone or push 5 feet) as data, MUST NOT apply any condition or movement, and MUST carry the DC, the chosen save ability and the published saving throw result (`CheckResult`, or `ForcedFailureResult` when no d20 was rolled). | `doc/rules/architecture.md` §4, §6 (package design) | happy |
| FR-078 | "No more than one size larger" MUST mean the target's `CreatureSize` is not more than one entry after the attacker's in the order Tiny, Small, Medium, Large, Huge, Gargantuan. This spec defines the predicate only; `content` owns the enumeration. | SRD 5.2.1, "Creature Size", p. 14 | unhappy (boundary) |
| FR-092 | When the target's chosen save ability is Dexterity, the Grapple or Shove save MUST add the applicable cover's Dexterity bonus (FR-017, FR-018) as a modifier; a Strength save MUST NOT. | SRD 5.2.1, "Cover", p. 15 (an attack "or other effect") | unhappy (rule interaction) |
| FR-093 | A Grapple or Shove against applicable Total Cover MUST give `AttackNotPossible` (Total Cover), no dice. | SRD 5.2.1, "Cover", p. 15 | unhappy (state error) |
| FR-094 | The command MUST carry the save as a `SaveAttempt` (`resolve-d20-test-use-case.md` FR-001) resolved with `d20.Resolver` and the use case's own `dice.Roller`, never the `d20` use case: for a `RollAttempt` it resolves the roll mode from the attempt's circumstances, rolls that many d20s, builds the `Roll` and calls `ResolveCheck(D20TestKind.SavingThrow, roll, modifiers, dc)`. The attempt's modifiers are unchanged except the single cover modifier of FR-092, appended once to `RollModifiers.others` of a `RollAttempt` only when the save ability is Dexterity and the cover gives a bonus, never to `ChooseToFail` or `AutomaticFailure`. All other caller modifiers and circumstances (for example Disadvantage on a Restrained target's Dexterity save, read by the caller via `conditions.md` FR-094) are used as supplied. | SRD 5.2.1, "Saving Throws", p. 7; "Restrained", p. 187; "Cover", p. 15 | happy |
| FR-095 | A `SaveAttempt` of `ChooseToFail` or `AutomaticFailure` MUST resolve through `d20.Resolver.ForcedFailure(D20TestKind.SavingThrow, reason)` (`d20-tests.md` FR-035, FR-036) to a failed save with no d20 and the effect applying, carrying the `ForcedFailureReason` (a target that chooses to fail, or automatically fails Strength and Dexterity saves: Paralyzed, Unconscious). | SRD 5.2.1, "Saving Throws", p. 7; "Paralyzed", p. 186; "Unconscious", p. 191 | unhappy (rule interaction) |
| FR-106 | An Unarmed Strike MUST use reach 5 feet and ignore any reach supplied elsewhere; its command has no reach field. | SRD 5.2.1, "Unarmed Strike", p. 190 | unhappy (boundary) |
| FR-124 | `UnarmedStrikeModifiers` (pure domain function) MUST build the `RollModifiers` from the `ability.Modifier` used (FR-129) and the `ProficiencyBonus`: ability slot `modifier.New(abilityModifier.Value())` (`ability-scores.md` FR-017, FR-027; `modifiers-and-advantage.md` FR-043), Proficiency Bonus slot `modifier.New(proficiencyBonus.Value())`, no others, so the bonus is present exactly once (FR-066). `ProficiencyBonus.value` (an `int` getter) is the accessor; `derived-stats.md` FR-059 gains it (§9, settled). The same values feed the Shove DC (FR-074), `GrappleRules.GrappleDC` (FR-128) and the damage amount (FR-067). | `d20-tests.md` D-16 and section 4.3; shared-types-and-ownership.md FR-003 | happy |
| FR-128 | For the Grapple option the use case MUST take the DC from `GrappleRules.GrappleDC(abilityModifier, proficiencyBonus)` (`grappling.md` FR-094), the save abilities from `OptionSaveAbilities` (FR-095) and the pre-checks from `possibility(attackerSize, targetSize, hand)` (FR-096), `hand` being the command's `HandAvailability` (`HandFree`, `NoHandFree`), and MUST NOT recompute them. `GrappleNotPossible(reason)` MUST become `UnarmedStrikeNotPossible` with `TargetTooLarge` and `NoFreeHand` mapped one to one to `AttackNotPossibleReason`, no die rolled; FR-105 gives the order. The result carries the DC returned (FR-077). | SRD 5.2.1, "Unarmed Strike", p. 190; "Grappling", p. 182; `grappling.md` FR-094 to FR-096; shared-types-and-ownership.md FR-029, FR-048 | happy; unhappy (boundary, rule interaction) |
| FR-129 | `ResolveUnarmedStrikeCommand` MUST carry the `ability.Modifier` used as `ability.Modifier`, documented as the Strength modifier (not a field named for Strength). Roll bonus (FR-065), damage (FR-067), Shove DC (FR-074) and the modifier given to `GrappleDC` (FR-128) MUST all use that one value. The engine knows no class feature; a caller applying one (for example `class-monk.md` FR-029, FR-030: Dexterity) supplies that modifier, a general extension point with the SRD rule as default (`doc/rules/architecture.md` §3). The Proficiency Bonus is always added (FR-066). | SRD 5.2.1, "Unarmed Strike", p. 190; `doc/rules/architecture.md` §3 | unhappy (rule interaction) |
| FR-132 | `ResolveAttackCommand` and `ResolveUnarmedStrikeCommand` MUST each be built by a validating constructor (`NewResolveAttackCommand`, `NewResolveUnarmedStrikeCommand`) that never panics and returns `(command, error)`; the error is the `errors.Join` of every violated field, each a `*<Subject>Problem{Field}` with `Field` a typed `<Subject>Field` value (errors-and-results.md FR-023, FR-024): attack command `distance` (negative, FR-033), `ImprovisedProficiencyBonus` (FR-046), `SpellAttackProficiencyBonus` (FR-101), `CriticalThreshold` (FR-134); Unarmed Strike command `distance` (negative), `CriticalThreshold` (Damage option), `SaveAbility` (neither Strength nor Dexterity; `GrappleRules.OptionSaveAbilities`, FR-073). An error is returned if and only if at least one field is violated, and `MustResolveAttackCommand` / `MustResolveUnarmedStrikeCommand` panic with that error (FR-107; `doc/rules/architecture.md` §6). Option variants (the closed `UnarmedStrikeOption` set) carry their own inputs (`DamageOption{ArmorClass, Circumstances, CriticalThreshold}`, `GrappleOption{SaveAbility, Attempt}`, `ShoveOption{Effect, SaveAbility, Attempt}`). | `doc/rules/architecture.md` §6; combat-foundations.md FR-022, FR-023 | unhappy (invalid input) |

### 4.5 Types, results, hand-offs and cross-cutting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-107 | `AttackRange`, `Reach`, `NearbyEnemy`, `DamageComponent` (`NewDiceDamage`, `NewFixedDamage`) and `DamageSpec` MUST be built by validating constructors (`NewAttackRange`, `NewReach`, `NewNearbyEnemy`, `NewDiceDamage`, `NewFixedDamage`, `NewDamageSpec`) that never panic and return `(value, error)` for untrusted input; the error is the `errors.Join` of every violated field as `*<Subject>Problem{Field}` with `Field` a typed `<Subject>Field` value (for example `AttackRangeFieldNormal`, `AttackRangeFieldLong`, `AttackRangeFieldSingle`; `DamageSpecFieldComponents`, `DamageSpecFieldAddsAbilityModifier`), and each `MustX` panics with that error (errors-and-results.md FR-023, FR-026; combat-foundations.md FR-022, FR-023). An error is returned if and only if a field is violated (`hit-points-and-damage.md` FR-002). `Problems()` on the error returns a copy. | `doc/rules/architecture.md` §4, §6; `errors-and-results.md`; combat-foundations.md FR-022, FR-023 | unhappy (invalid input) |
| FR-112 | A `dice.Roller` result for a different expression than requested (attack d20, damage die, save d20) MUST make the use case return a `*srderr.PortError` for the dice port wrapping `srderr.ErrContractViolation`, through `dice.Roll` (which already checks the answer), with no partial result and no damage. | `doc/rules/testing.md` §4; `resolve-d20-test-use-case.md` FR-026, FR-027 | unhappy (failure of collaborator) |
| FR-119 | Attack resolution MUST be pure with respect to conditions: neither use case nor any `combat/domain` type of this spec imports or calls `ConditionEffects`; every consequence arrives as input (`RollModifiers`, `modifier.RollCircumstance`s, `AutomaticCritical`, `SaveAttempt`). | shared-types-and-ownership.md FR-013, FR-015, FR-026 | unhappy (forbidden edge) |
| FR-120 | Each use case MUST return its own closed `<UseCase>Result` with operation-prefixed variants: `AttackResolved`, `AttackAutomaticallyMissed`, `AttackNotPossible`; `UnarmedStrikeDamageResolved`, `UnarmedStrikeGrappleResolved`, `UnarmedStrikeShoveResolved`, `UnarmedStrikeNotPossible`. No variant is shared and none has a generic name (`Applied`, `Rejected`, `Succeeded`, `Miss`, `Hit`). The not-possible reason is an `AttackNotPossibleReason` field of each not-possible variant; the `...NotPossible` suffix is kept because `combat-action-resolution.md` consumes it (settled exception to combat-foundations.md FR-023, §9). | shared-types-and-ownership.md FR-017, FR-018, FR-021 | unhappy (name collision) |
| FR-121 | The result MUST reuse the `d20` vocabulary: a rolled attack carries the published `AttackRollResult` unchanged; the final outcome is derived from it plus a `MissOverride` (`none`, `TargetNotAtLocation`, FR-036) and `AutomaticCritical` (FR-090). A miss decided without a roll (FR-041) is `AttackAutomaticallyMissed` carrying an `AutomaticMissReason`, never an `AttackRollResult`. | shared-types-and-ownership.md FR-004 | unhappy (duplicate) |
| FR-123 | This spec MUST NOT define or export `Proficiency`, `AttackConclusion`, `AutomaticCriticalHitPolicy`, a `Dc` or `DamageType`, nor re-export `CreatureSize`. An architecture test fails on an exported name that duplicates one of an upstream package (`d20`, `content`) and when a file of this spec mentions the `d20` use case or one use case mentions the other (FR-109, FR-127). | shared-types-and-ownership.md FR-004, FR-005, FR-021, FR-027 | unhappy (duplicate) |
| FR-125 | `CircumstanceCollector` (domain service) MUST NOT return or import an application type. It MUST return the closed `AttackPlan`: `AttackPlanProceeds(circumstances, ...)`, `AttackPlanNotPossible(AttackNotPossibleReason)` or `AttackPlanAutomaticMiss(AutomaticMissReason)`; the use case maps it to the result variants. The Unarmed Strike size and free-hand checks (`GrappleRules.possibility` via FR-128; FR-075) return the same `AttackPlan` from a domain function. | `doc/rules/architecture.md` §1, §9 | unhappy (rule interaction) |
| FR-126 | `WeaponUse` MUST be `NewWeaponUse(category WeaponCategory, weaponType WeaponType, properties []WeaponProperty, manner WeaponUseManner)` with `content` types and the typed enum `WeaponUseManner` (`WieldedAsDesigned`, `ThrownAttack`, `MeleeAttackWithRanged`). A zero `category` is an object that is not a Simple or Martial weapon (always improvised; `type` and `properties` unread). `properties` MUST be copied on construction and on read; no weapon vocabulary is re-declared and no `thrown` boolean exists. | `doc/rules/architecture.md` §3; shared-types-and-ownership.md FR-005 | unhappy (invalid input) |
| FR-127 | `attackFlow` (constructed from a `Roller`) MUST be an unexported helper type of package `combat` with methods `resolveAttack(ResolveAttackCommand)`, `resolveUnarmedStrike(ResolveUnarmedStrikeCommand)` and the save and check helper of FR-133. It does the shared steps: roll d20s via `dice.Roll`, build the `d20.Roll`, resolve with `d20.Resolver`, roll damage dice, compose `Damage` with `AttackDamageComposer`. It is not a use case or port, takes no `telemetry.Telemetry`, records no signal or log and is exported by no package. `ResolveAttack.Run` MUST be the shared use-case recorder around `attackFlow.resolveAttack` (likewise `ResolveUnarmedStrike`), returning its result unchanged. A roller failure is returned as the `error` (FR-081, FR-112). It MUST NOT mention the `d20` use case or either use case. | `doc/rules/architecture.md` §4; shared-types-and-ownership.md FR-044; `combat-action-resolution.md` section 6 | unhappy (rule interaction) |
| FR-133 | `attackFlow` MUST offer `ResolveSave(SaveAttempt, DifficultyClass)` and `ResolveCheck(CheckAttempt, DifficultyClass)` (attempt types of `resolve-d20-test-use-case.md` FR-001) over one routine. For a `RollAttempt` it resolves the mode once with `RollMode.resolve`, draws `mode.diceCount` d20s through `RollChecked`, builds the `Roll` and calls `d20.Resolver.ResolveCheck(kind, roll, modifiers, target)` (kind `SavingThrow` or `AbilityCheck`). For `ChooseToFail` (saves only) or `AutomaticFailure` it draws no die and returns `ForcedFailure(kind, reason)`. It returns the published `CheckResult` or `ForcedFailureResult` (as an unexported two-variant closed value, since `d20` publishes no common supertype), adds no modifier (the cover bonus is added by `ResolveUnarmedStrike` first) and records no signal. | `d20-tests.md` FR-069; `resolve-d20-test-use-case.md` FR-001, FR-006, FR-007; SRD 5.2.1, "D20 Tests", p. 6; "Saving Throws", p. 7 | happy; unhappy (rule interaction) |
| FR-079 | With the same command and fixed dice a use case MUST return the same result; all randomness MUST come from the `dice.Roller`. | `doc/rules/testing.md` §4; `doc/rules/architecture.md` §3 | unhappy (determinism) |
| FR-080 | Resolving an attack MUST NOT change the command, and any slice in the result MUST be returned as a copy. | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |
| FR-081 | A typed `srderr.PortError` thrown by the `dice.Roller` MUST propagate unchanged, no partial result. | `doc/rules/architecture.md` §4, §6 | unhappy (failure of collaborator) |
| FR-082 | With extreme rolls (all minimum, all maximum) the result MUST equal the table cases for those rolls. | `doc/rules/testing.md` §4 | unhappy (boundary) |
| FR-083 | Each use case MUST behave identically with the no-op and a recording `telemetrytest` bundle. | `doc/rules/observability.md` §9 | happy |
| FR-084 | Each use case MUST emit the signals of §7 through the shared use-case recorder (combat-foundations.md FR-035). The shared rows (span, `srd5e.combat.operations`, `operation_duration`, `errors`, `events`) are owned by `combat-foundations.md`; this spec adds the extras of §7 via the attributes argument of the shared recorder's `Run` (combat-foundations.md FR-041: span and `operations` increment only, keys from the combat or `d20` signal constants, closed values, no shared key of FR-030), the histogram (FR-130) and the `FINE` log (FR-131). | `doc/rules/observability.md` §5, §6, §9; combat-foundations.md FR-027 to FR-032, FR-041 | happy |
| FR-085 | A thrown failure MUST be recorded exactly once, by the use case called: span error status, one `srd5e.combat.errors` increment with `srd5e.combat.operation` and `error.type`, no log. A roller failure in an Unarmed Strike's save d20 gives one error for `resolve_unarmed_strike` and no `srd5e.d20_test.errors`. | `doc/rules/observability.md` §7, §9; combat-foundations.md FR-032, FR-035 | unhappy (failure of collaborator) |
| FR-086 | `AttackNotPossible` and `UnarmedStrikeNotPossible` MUST be `refused` (combat-foundations.md FR-030, FR-031): `srd5e.outcome` `failure`, `result_class` `refused`, no error status, no `errors` increment. A hit, miss, automatic miss and a resisted or applied Grapple or Shove are `applied` (outcome `success`); which one is carried by `srd5e.combat.result` (FR-122). Neither use case returns `invalid`. | `doc/rules/observability.md` §6, §7; combat-foundations.md FR-030, FR-031 | unhappy (state error) |
| FR-108 | Package `combat` MUST export, each once and with a doc comment starting with its name, every type needed to build a command and switch over a result (§6, including the problem types of FR-107 and FR-132); the package doc in `doc.go` lists inputs and outputs. A public-API test in package `scenarios_test`, importing only public packages, builds each command and switches exhaustively over each result. | `doc/rules/architecture.md` §4, §7; `resolve-d20-test-use-case.md` FR-042 | happy |
| FR-109 | `ResolveAttack` and `ResolveUnarmedStrike` MUST be built from a roller (any `dice.Roller`) and optional `telemetry.Telemetry` only and neither takes, builds or calls a use case. Each composes the pure services `d20.Resolver`, `GrappleRules` (FR-128), `CoverResolver`, `CircumstanceCollector`, `AttackDamageComposer`, `UnarmedStrikeModifiers`, sharing steps through `attackFlow` (FR-127). No `telemetry.Noop` silences a nested call (the earlier nested-use-case design is withdrawn). | `shared-types-and-ownership.md` FR-044; `doc/rules/architecture.md` §4; `doc/rules/observability.md` §7 | unhappy (rule interaction) |
| FR-110 | One Unarmed Strike MUST increment `srd5e.combat.operations` once with `resolve_unarmed_strike` and never `resolve_attack`; one `ResolveAttack` call once with `resolve_attack`. Neither touches any `srd5e.d20_test.*` metric or span (the reused keys of FR-118 ride on combat signals). | `doc/rules/observability.md` §6, §7; combat-foundations.md FR-029; shared-types-and-ownership.md FR-044 | unhappy (rule interaction) |
| FR-115 | The constructors MUST be `NewResolveAttack(roller Roller, opts ...Option) (ResolveAttack, error)` and `NewResolveUnarmedStrike(...)` alike, where `Roller` is the consumer-side one-method interface (`Roll(dice.Expression) (dice.Result, error)`) that any `dice.Roller` satisfies, and `WithTelemetry(telemetry.Telemetry)` is the option; omitting it uses `telemetry.Noop` and works. A nil roller returns a validation error. `telemetry.Noop` is the zero-value no-op (shared-types-and-ownership.md FR-036) and is never passed to another use case. | `doc/rules/architecture.md` §4; `observability-ports.md` FR-042; shared-types-and-ownership.md FR-036 | happy |
| FR-122 | Telemetry MUST follow combat-foundations.md FR-028 to FR-033. `srd5e.outcome` is `success` or `failure` only, derived from `CombatResultClass` (`applied` is `success`, `refused` is `failure`), never from hit or miss. Detail is `srd5e.combat.result`, a closed set owned here: `resolve_attack`: `hit`, `critical_hit`, `miss` (including a forced miss), `automatic_miss` (class `applied`), `not_possible` (class `refused`); `resolve_unarmed_strike`: `damage_hit`, `damage_critical_hit`, `damage_miss`, `grapple_resisted`, `grapple_applied`, `shove_resisted`, `shove_applied` (class `applied`), `not_possible` (class `refused`). A miss and a resisted Grapple or Shove are `applied` and `success`. `result_class` is never `invalid`. `error.type` appears only on `srd5e.combat.errors`, the snake_case `srderr.PortError` subtype name (`errors-and-results.md`). `not_possible`, `neither` and `critical_hit` are not values of `srd5e.outcome` or the roll-mode attribute. | combat-foundations.md FR-028 to FR-033; shared-types-and-ownership.md FR-041; `doc/rules/observability.md` §5 | unhappy (rule interaction) |
| FR-130 | Each `ResolveAttack` call that returns a result MUST record one sample in the histogram `srd5e.combat.attack.modifiers` (unit `{modifier}`): the count of modifiers in `RollModifiers` (ability modifier, Proficiency Bonus when present, each of `others`), with `srd5e.combat.attack.form` set to the command's `AttackForm` value. A not-possible call records one sample; a throwing call none; `ResolveUnarmedStrike` none. | `doc/rules/observability.md` §5 ("Sizes") | happy; unhappy (failure of collaborator) |
| FR-131 | Each use case MUST emit at most one `FINE` log per returned call, message `attack resolved` or `unarmed strike resolved`, one attribute `srd5e.combat.result` (FR-122) supplied as a closure. A thrown failure is not logged (FR-085); no `INFO` or higher, no per-modifier log. | `doc/rules/observability.md` §7 | happy; unhappy (failure of collaborator) |
| FR-116 | A call ending in `AttackNotPossible` or `UnarmedStrikeNotPossible` MUST set `srd5e.combat.attack.reason` on the span and the `operations` increment of the use case called, with a value of the closed set, and no reason on any other result, via the attributes argument of the shared recorder. | `doc/rules/observability.md` §5, §6; combat-foundations.md FR-041 | happy |
| FR-117 | The reason values MUST map one to one from `AttackNotPossibleReason`: `TotalCover` to `total_cover`, `BeyondRange` to `beyond_range`, `BeyondReach` to `beyond_reach`, `TargetTooLarge` to `target_too_large`, `NoFreeHand` to `no_free_hand`. | `doc/rules/observability.md` §5 | happy |
| FR-118 | `srd5e.d20_test.roll_mode` and `srd5e.d20_test.critical_hit` (the `d20` signal constants, `resolve-d20-test-use-case.md` FR-040) MUST be omitted when no d20 was rolled (not-possible, the automatic miss of FR-041, `ChooseToFail`, `AutomaticFailure`). Otherwise `roll_mode` is the `RollMode` rolled (attack roll, Damage option, Grapple or Shove save d20); `critical_hit` is set only when an attack roll was made and equals `AttackRollResult.IsCriticalHit` (the kept die at or above the critical range, a natural 20 by default, FR-134; `resolve-d20-test-use-case.md` FR-052), absent for a save; an automatic Critical Hit and a natural 20 turned into a forced miss are not reflected (the final state is `srd5e.combat.result`). No `none` or `neither` value. Both keys are returned through `attributes` (combat-foundations.md FR-041), the map lacking the key when it does not apply. | `doc/rules/observability.md` §5; shared-types-and-ownership.md FR-041; `resolve-d20-test-use-case.md` FR-040, FR-052; combat-foundations.md FR-041 | unhappy (state error) |
| FR-135 | `ResolveAttackCommand` MUST offer an immutable copy operation `WithAdditions(Additions) (ResolveAttackCommand, error)` where `Additions{Circumstances []modifier.RollCircumstance, Modifiers []modifier.Modifier, AutomaticCritical AutomaticCritical}` (zero values add nothing, `NotAutomatic` is the zero `AutomaticCritical`), returning a new command. Circumstances are appended after the existing ones; modifiers are appended to `RollModifiers.others` (the ability modifier and Proficiency Bonus are never replaced, FR-053); the result is `automatic` when either command is (OR, never cleared, FR-090); every other field, including `CriticalThreshold`, is carried unchanged. The original is not altered and every slice is copied. The result is built through the same validation (FR-033, FR-132): the copy returns a validation error exactly where the constructor would. `combat-action-resolution.md` FR-032 uses it for derived inputs; this spec derives nothing. | `combat-action-resolution.md` FR-004, FR-032 (Q-6); `doc/rules/architecture.md` §3 | unhappy (immutability; rule interaction) |

### 4.6 Withdrawn ids

| Id | Status |
|---|---|
| FR-001 | Withdrawn: no `AttackTarget` type, target kind changes no rule (host supplies the AC); not moved |
| FR-070 | Withdrawn: moved to grappling.md (FR-095) |
| FR-071 | Withdrawn: moved to grappling.md (FR-094) |
| FR-072 | Withdrawn: moved to grappling.md (FR-096), Grapple half; the Shove half is FR-075 |
| FR-099 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-031, FR-032) |
| FR-102 | Withdrawn: moved to shared-types-and-ownership.md (FR-027) |

### Case tables

Each row is one table-driven test; "Roll" is the d20 kept. Outcomes use the real
`d20.Resolver` and a fixed `dice.Roller`. Source "pkg" means a package decision.

| Case | Inputs | Expected | FR | Source | Path |
|---|---|---|---|---|---|
| above / equals / one below AC | AC 15, +5: roll 12 / 10 / 9 | Hit 17 / Hit 15 / Miss 14 | FR-004, 005, 006 | "Attack Rolls", p. 7 | happy; unhappy (boundary) |
| natural 20 far below AC | AC 30, roll 20, +0 | Critical Hit | FR-008 | "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| natural 20 behind Half Cover | AC 15, Half, roll 20, -5 | Critical Hit | FR-020 | "Rolling 20 or 1", p. 7; "Cover", p. 15 | unhappy (rule interaction) |
| natural 19 | AC 30 roll 19 +0: Miss; AC 19 roll 19 +0: Hit (not Critical) | as stated | FR-010 | "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| natural 1 above AC | AC 5, roll 1, +20 | Miss | FR-009 | "Rolling 20 or 1", p. 7 | unhappy (boundary) |
| no modifiers | AC 15, roll 15, +0, no Proficiency Bonus | Hit 15 | FR-003 | "D20 Tests", p. 6 | unhappy (empty and minimal case) |
| dice 1 and 20 | Advantage (AC 15): Critical Hit; Disadvantage (AC 5): Miss | as stated | FR-011 | "Roll Two D20s", p. 8 | unhappy (rule interaction) |
| natural 20 against an unseen target | AC 15, roll 20: target not at location / target at location | Miss, no damage / Critical Hit (Disadvantage dice) | FR-091, 036, 034 | "Unseen Attackers and Targets", p. 14 | unhappy (rule interaction) |
| automatic Critical Hit | AC 15 +5: roll 12 / 9 / (AC 5 roll 1 +20) / (roll 20) | Critical Hit, dice doubled / Miss no damage / Miss / Critical Hit once, no further doubling | FR-090 | "Paralyzed", p. 186; "Rolling 20 or 1", p. 7; "Critical Hits", p. 16 | unhappy (rule interaction, boundary) |
| automatic Critical Hit, Total Cover | AC 15, Total | NotPossible, no dice | FR-090, 019 | "Cover", p. 15 | unhappy (state error) |
| cover: none / Half / Three-Quarters | base AC 15 | AC 15, Dex 0 / AC 17, +2 / AC 20, +5 | FR-016, 013, 015 | "Cover", p. 15 | unhappy (empty); happy |
| two Half; Half and Three-Quarters | AC 15 | AC 17, +2; AC 20, +5 | FR-017 | "Cover", p. 15 | unhappy (rule interaction) |
| not on opposite side | Three-Quarters (not opposite): AC 15, 0; Half plus Three-Quarters (not opposite): AC 17, +2 | as stated | FR-018 | "Cover", p. 15 | unhappy (rule interaction) |
| Total; Total with Half | AC 15 | NotPossible both | FR-019, 017 | "Cover", p. 15 | unhappy (state error) |
| effective AC boundary | AC 15, Half (17): roll 10 + 7 / 10 + 6 | Hit / Miss | FR-015 | "Cover", p. 15 | unhappy (boundary) |
| single range 60 | distance 60 / 61 | possible / NotPossible | FR-021, 022 | "Range", p. 15 | unhappy (boundary) |
| range 80/320 | distance 0 / 80 / 81 / 320 / 321 | no effect / no effect / Disadvantage / Disadvantage / NotPossible | FR-026, 023, 024, 025 | "Range", p. 15 | unhappy (boundary) |
| melee reach 5 / 10 | reach 5: distance 5 / 6; reach 10: distance 10 / 11 | possible / NotPossible (each) | FR-030, 031, 032 | "Reach", p. 15 | happy; unhappy (boundary) |
| melee ignores range rules | weapon or spell, reach 5, distance 5, enemy adjacent | possible, no Disadvantage | FR-029, 088 | "Melee Attacks", p. 15 | unhappy (rule interaction) |
| melee spell one beyond reach | reach 5, distance 6 | NotPossible | FR-088, 031 | "Reach", p. 15 | unhappy (boundary) |
| ranged spell range | 60/120 at 61: Disadvantage; single 120 at 121: NotPossible | as stated | FR-089, 024, 022 | "Range", p. 15 | unhappy (boundary) |
| close combat, enemy at 5 / 6 feet | sighted, not Incapacitated | Disadvantage / none | FR-027, 028 | "Ranged Attacks in Close Combat", p. 15 | unhappy (rule interaction, boundary) |
| close combat, cannot see attacker; Incapacitated; no enemies | enemy at 5 | no Disadvantage (each) | FR-028 | "Ranged Attacks in Close Combat", p. 15 | unhappy (rule interaction, empty) |
| close combat, two enemies one qualifies; ranged spell adjacent; melee spell adjacent | at 5 | Disadvantage; Disadvantage (FR-089); none (FR-088) | FR-027, 089, 088 | "Ranged Attacks in Close Combat", "Melee Attacks", p. 15 | unhappy (rule interaction) |
| unseen | `TargetUnseen(true)`: Disadvantage; `TargetUnseen(false)`: roll made, Miss; target cannot see attacker: Advantage; both unseen: `RollMode.normal`, one d20 | as stated | FR-034, 036, 035, 038 | "Unseen Attackers and Targets", p. 14; "They Don't Stack", p. 8 | unhappy (rule interaction, state error) |
| hidden attacker; seen and not at location | hit and miss, target sees attacker; `TargetSeen` with "not at location" | location given away (both); not constructible | FR-037, 104 | "Unseen Attackers and Targets", p. 14 | unhappy (rule interaction, invalid input) |
| underwater melee weapon, no Swim Speed | Slashing | Disadvantage | FR-039 | "Underwater Combat", p. 16 | unhappy (rule interaction) |
| underwater melee, Piercing; Piercing among two; Swim Speed | no Swim Speed Piercing; Slashing and Piercing; Slashing with Swim Speed | no Disadvantage (each) | FR-040, 103 | "Underwater Combat", p. 16 | unhappy (boundary, rule interaction) |
| underwater melee, no damage spec | none | Disadvantage | FR-103 | "Underwater Combat", p. 16 | unhappy (empty and minimal case) |
| underwater spell and Unarmed Strike | melee spell; ranged spell one beyond normal; Unarmed Strike Bludgeoning | no Disadvantage; no automatic miss, Disadvantage from range only; no Disadvantage | FR-040, 088, 089, 113 | "Underwater Combat", p. 16; "Weapon Attack", p. 191 | unhappy (boundary, rule interaction) |
| underwater ranged weapon | at normal, with and without Swim Speed: Disadvantage; one beyond normal within long: Miss, no roll; beyond long: NotPossible (not automatic miss) | as stated | FR-042, 041, 045 | "Underwater Combat", p. 16 (miss); pkg (no roll); "Range", p. 15 | unhappy (boundary, rule interaction) |
| combining; precedence | long range plus close combat: one Disadvantage, one d20; long range plus caller Advantage: `RollMode.normal`, one d20. Precedence: Total Cover and beyond range: Total Cover; distance 6 and two sizes larger (Grapple): beyond reach; two sizes larger and no hand: target too large; Grapple Total and too large: Total Cover | as stated | FR-043, 044, 105 | "They Don't Stack", p. 8; pkg | unhappy (rule interaction) |
| ability defaults | Unarmed Strike: Strength modifier used; spell attack: no default ability | as stated | FR-007, 129 | "Unarmed Strike", p. 190; "Attack Rolls", p. 7 | happy; unhappy (boundary) |
| one ability value for roll and damage | melee weapon, `RollModifiers.ability` `Modifier(2)`, flagged component | attack total and damage both add +2, no other ability value | FR-053 | "Damage Rolls", p. 16 | unhappy (rule interaction) |
| improvised proficiency | improvised with Proficiency Bonus +2: validation error; without: none added; non-improvised `WeaponUse` with +2: added; spell attack without Proficiency Bonus: validation error, with +3: +3 added | as stated | FR-046, 100, 101 | "Improvised Weapons", p. 183 | unhappy (invalid input); happy |
| improvised classification | Ranged weapon in melee; Melee weapon without Thrown, thrown; zero category: improvised. Melee `WeaponType` `ThrownAttack` with Thrown: not improvised. `MeleeAttackWithRanged` on Melee type: validation error; improvised thrown: 1d4, range 20/60 | as stated | FR-049, 126, 047, 048 | "Improvised Weapons", p. 183; "Thrown", p. 90 | unhappy (boundary, invalid input) |
| damage hit, fixed, flag | 1d4 first +2 roll 3: 5; fixed 1 no flag +2: 1; 2d6 first +0 rolls 3 and 4: 7 | as stated | FR-051, 053, 054, 060 | "Damage Rolls", p. 16 | happy; unhappy (rule interaction) |
| penalty floor | 1d4 first roll 2: -3 gives 0 (not -1); -2 gives 0 | as stated | FR-055 | "Damage Rolls", p. 16 | unhappy (boundary) |
| Critical Hit damage | 1d4 first +2 rolls 3 and 4: 9; 1d8 +3 rolls 5 and 6: 14 (modifier not doubled); 1d6 plus extra 2d6, +0, main 1 and 2, extra 3, 4, 5, 6: 21; fixed 5: 5; 1d4 -5 rolls 1 and 1: 0 | as stated | FR-056 to 059, 055 | "Critical Hits", p. 16 | happy; unhappy (boundary, rule interaction) |
| two types | 1d6 slashing and 1d4 fire +1 rolls 3; 2: flag first 4 slashing and 2 fire; no flag 3 and 2; both flagged: validation error | as stated | FR-061, 096 | "Damage Types", p. 16; pkg | happy; unhappy (invalid input) |
| penalty on one of two | 1d6 slashing (-4) and 1d4 fire, rolls 2; 3: 0 and 3; ability -3 flag first, rolls 2; 3: 0 and 3 | as stated | FR-097, 098, 055 | "Damage Rolls", p. 16 | unhappy (boundary) |
| no or empty spec; `Damage` conversion | empty: validation error; none given: hit with no damage; 1d6 slashing roll 4 | as stated; `Damage(4, slashing)`, `HitKind.NormalHit` | FR-062, 114 | pkg; "Making an Attack", pp. 14-15; "Damage Types", p. 16 | unhappy (empty and minimal case); happy |
| invalid die face | 1d6, roller returns 7 | `a port failure (`*srderr.PortError` for the dice port)(invalidResult)`, no result | FR-112 | testing.md §4 | unhappy (failure of collaborator) |
| Damage option | Str +3 PB +2: bonus +5; Str 0: bonus +2, damage 1; -1: 0; -5: 0 (not -4); +4: 5; natural 20: damage 4 unchanged; automatic Critical Hit: damage 4 unchanged | as stated | FR-065, 067, 069, 090 | "Unarmed Strike", p. 190; "Critical Hits", p. 16 | happy; unhappy (boundary) |
| Damage option reach, underwater | target at 6 feet: NotPossible; reach 10 offered with target at 10: NotPossible; underwater: no Disadvantage | as stated | FR-064, 106, 113 | "Unarmed Strike", p. 190; "Underwater Combat", p. 16 | unhappy (boundary, rule interaction) |
| Dexterity modifier supplied; DCs | Dex +4 PB +2: bonus +6, damage 5, Grapple DC 14, Shove DC 14; Str +3 PB +2: Grapple DC 13, Shove DC 13; Str -1: Grapple DC 9 | as stated | FR-129, 065, 067, 128, 074 | "Unarmed Strike", p. 190 | happy; unhappy (rule interaction, boundary) |
| save vs DC | save total equals DC: succeeds, no Grappled; one below: Grappled | as stated | FR-076 | "D20 Tests", p. 6 | unhappy (boundary) |
| Grapple pre-checks | `HandFree`: target smaller or one size larger: possible; two larger: `UnarmedStrikeNotPossible(targetTooLarge)` from `GrappleNotPossible(targetTooLarge)`; `NoHandFree`: `UnarmedStrikeNotPossible(noFreeHand)`; no dice | as stated | FR-128 | "Unarmed Strike", p. 190 | happy; unhappy (boundary, state error) |
| Grapple save ability Wisdom | chosen Wisdom | `NewResolveUnarmedStrikeCommand` returns a joined error holding the `SaveAbility` problem; `MustResolveUnarmedStrikeCommand` panics with it | FR-132, 128 | "Unarmed Strike", p. 190 | unhappy (invalid input) |
| Shove pre-checks, effects | two sizes larger: NotPossible (target too large); `NoHandFree`: possible; failed save: push 5 feet or Prone | as stated | FR-075, 073 | "Unarmed Strike", p. 190 | unhappy (boundary); happy |
| size order | Medium to Huge | two larger | FR-078 | "Creature Size", p. 14 | unhappy (boundary) |
| save cover bonus | Dexterity behind Half: modifiers include +2; Shove Dexterity behind Three-Quarters: +5; Strength behind Half: none; caller +1, Half, d20 10: total 13 (cover once), Strength save 11 | as stated | FR-092, 094 | "Cover", p. 15 | unhappy (rule interaction) |
| Total Cover; Restrained target | Grapple or Shove behind Total; Dexterity save with Disadvantage supplied | NotPossible (Total Cover), no dice; two d20 rolled, lower used | FR-093, 094 | "Cover", p. 15; "Restrained", p. 187 | unhappy (state error, rule interaction) |
| forced failure | Grapple `ChooseToFail`: no d20, Grappled, reason chosen to fail; Shove `AutomaticFailure`: no d20, effect applies, reason automatic; Dexterity `ChooseToFail` with Half Cover: no modifiers appended | as stated | FR-095, 094 | "Saving Throws", p. 7; "Paralyzed", p. 186 | unhappy (rule interaction) |
| `UnarmedStrikeModifiers` | Str +3 PB +2: ability `Modifier(3)`, PB `Modifier(2)`, no others; Str -1: `Modifier(-1)`, `Modifier(2)` | as stated | FR-124, 065 | "Unarmed Strike", p. 190 | happy; unhappy (boundary) |
| roller fails in save d20 | Grapple | the `*srderr.PortError` is returned; one `resolve_unarmed_strike` error, no `resolve_attack` operation, no `d20` signal | FR-081, 085, 110 | observability.md §7 | unhappy (failure of collaborator) |
| no use case collaborator | Damage and Grapple | both built with a roller and optional `telemetry.Telemetry` only; one `operations` increment per call, no `srd5e.d20_test.*` signal | FR-109, 110, 115 | architecture.md §4 | happy |
| `attackFlow` equals use cases | Damage, Grapple, attack | `attackFlow{roller}.resolveUnarmedStrike(command)` equals `ResolveUnarmedStrike.Run(ctx, command)` for equal dice (likewise `ResolveAttack`); helper records no signal | FR-127 | architecture.md §4 | happy |
| `attackFlow` save and check | save `RollAttempt` with Disadvantage: two d20 via `RollChecked`, lower kept, `CheckResult`; save `ChooseToFail`, check `AutomaticFailure`: no die, `ForcedFailureResult` (`ChosenToFail`, `automatic`); check `RollAttempt` normal: one d20, kind `AbilityCheck` | as stated | FR-133 | "Saving Throws", p. 7; "D20 Tests", p. 6 | unhappy (rule interaction); happy |
| modifiers histogram | ability, PB and two others | one sample of 4 with form attribute; none from Unarmed Strike or a failure; one for not-possible | FR-130 | observability.md §5 | happy; unhappy (failure of collaborator) |
| FINE log | attack hit; Unarmed Strike not possible; roller fails | `attack resolved` with `hit`; `unarmed strike resolved` with `not_possible`; no log for the failure | FR-131 | observability.md §7 | happy; unhappy (failure of collaborator) |
| command parse | attack: improvised with PB, negative distance; Unarmed Strike: negative distance, Wisdom save | `...Rejected` listing every violated field; constructor returns a validation error; valid gives `...Parsed` | FR-132 | architecture.md §6 | unhappy (invalid input) |

## 5. Model

All types live in package `combat`, one file per concept. Domain types are
immutable (unexported fields, value receivers) with `NewX` constructors that
return validation errors for untrusted input (`doc/rules/architecture.md` §3,
§6); commands and results sit beside the use cases in `*_usecase.go` files.
**Domain:**

- `AttackForm` (a closed variant set, five variants; `Delivery`: `Melee` | `Ranged`, `Source`:
  `Weapon` | `Spell` | `UnarmedStrike`; no default ability), FR-087.
- `Cover` (closed `HalfCover`, `ThreeQuartersCover`, `TotalCover`, each with
  `ACBonus` and `DexteritySaveBonus`); `CoverSource` (cover plus
  `AttackOriginatesOppositeSide`); `CoverResolver` picks the most protective (FR-013, FR-014, FR-017, FR-018).
- `AttackRange` (`SingleRange{Feet}`, `NormalAndLongRange{Normal, Long}`) with
  `NewAttackRange`; `RangeBand` (`WithinNormal`, `BeyondNormal`, `BeyondLong`); `Reach` (default 5); `NearbyEnemy{Distance, CanSeeAttacker, Incapacitated}`;
  `TargetVisibility` (`TargetSeen`, `TargetUnseen{TargetAtTargetedLocation}`);
  `AttackSituation` (visibility, target can see attacker, attacker hidden, underwater, Swim Speed; Piercing is read from the `DamageSpec`, FR-103).
- `CircumstanceCollector` turns form, range, distance, enemies, situation and damage types into an `AttackPlan` (FR-125, FR-043, FR-045, FR-105).
- `WeaponUse`, `WeaponUseManner` (FR-126, FR-049, FR-100); `UnarmedStrikeModifiers` (FR-124). No `Proficiency` type: `RollModifiers` carries the ability modifier, optional Proficiency Bonus and others; the damage composer reads the ability modifier from it (FR-053).
- `DamageComponent` (closed `DiceDamage{Dice, Type, Modifier, AddsAbilityModifier}`,
  `FixedDamage{Amount, Type, Modifier, AddsAbilityModifier}`), `DamageSpec`, both with validating `NewX` constructors; `AttackDamageComposer` doubles dice on a Critical Hit, adds modifiers and the one ability modifier, floors each component at 0 and converts to `Damage` (FR-055 to FR-059, FR-096, FR-097, FR-114). It is not the
  `DamageResolver`.
- `AutomaticCritical` is `conditions.md`'s enum, read and never derived (FR-090); `CriticalThreshold` is passed through (FR-134).
  `MissOverride` (`none`, `TargetNotAtLocation`) and `AutomaticMissReason` (`UnderwaterBeyondNormalRange`) hold what the d20 did not decide; final hit is
  `roll.Hit() && override == none`, final Critical Hit is `finalHit && (roll.IsCriticalHit() || AutomaticCritical applies)` (FR-121).
- `AttackNotPossibleReason`: `TotalCover`, `BeyondRange`, `BeyondReach`,
  `TargetTooLarge`, `NoFreeHand`; distinct from `grappling.md`'s
  `GrappleNotPossibleReason`, mapped one to one (FR-128).
- Size predicate `IsNoMoreThanOneSizeLarger(attacker, target)` (FR-078, FR-075). Invariants sit in constructors (FR-033, FR-062, FR-096), `TargetVisibility` (FR-104) and command constructors (FR-046, FR-101, FR-132).

**Use cases** (`*_usecase.go`):
- `ResolveAttackCommand`, `ResolveUnarmedStrikeCommand`; `ResolveAttackResult` (`AttackResolved`, `AttackAutomaticallyMissed`, `AttackNotPossible`) and
  `ResolveUnarmedStrikeResult` (`UnarmedStrikeDamageResolved`,
  `UnarmedStrikeGrappleResolved`, `UnarmedStrikeShoveResolved`,
  `UnarmedStrikeNotPossible`), FR-120. `AttackResolved` carries the
  `AttackRollResult`, `MissOverride`, final hit and Critical Hit, AC, effective AC, cover, circumstances, delivery (FR-111), `HitKind`, `Damage` per component and
  `RevealsAttackerLocation`. The Grapple and Shove results carry the DC, the chosen save ability, the saving throw result and the effect as data.
- `UnarmedStrikeOption` (closed `DamageOption`, `GrappleOption`, `ShoveOption`; effect `PushFiveFeet` | `GiveProne`; save ability Strength or Dexterity), FR-132; unexported `attackFlow` (FR-127, FR-133).
- Failures: not possible, a miss and a failed save are result variants, never errors; untrusted bad arguments are validation errors from `NewX` (`errors.Is` `srderr.ErrInvalidInput`; a panic only in `MustX`); a roller failure is the returned `error`, a `*srderr.PortError` wrapped once. No domain events.

## 6. Interface

Both use cases are synchronous calls that take `context.Context` first (they emit spans; `doc/rules/architecture.md` §4, go.md §4).
- `NewResolveAttack(roller Roller, opts ...Option) (ResolveAttack, error)` with
  `Run(ctx, ResolveAttackCommand) (ResolveAttackResult, error)` (FR-115): `ResolveAttackCommand` (form, armor class, `RollModifiers`, cover sources, distance, range or reach, nearby enemies, `AttackSituation`, extra
  `modifier.RollCircumstance`s, `AutomaticCritical`, optional `CriticalThreshold` (FR-134), optional `DamageSpec`, `WeaponUse` where relevant; `NewResolveAttackCommand`, FR-132; `WithAdditions`, FR-135) returning `ResolveAttackResult`. It is
  `attackFlow.resolveAttack` in the shared use-case recorder (FR-127), holds no rules logic and never receives the target's identity.
- `ResolveUnarmedStrike(dice.Roller roller, {telemetry.Telemetry observability = telemetry.Noop})` (FR-115): `ResolveUnarmedStrikeCommand` (option with its inputs, `ability.Modifier` FR-129, `ProficiencyBonus`, attacker and target size, `HandAvailability`, distance, cover sources, `AutomaticCritical`; no reach field, FR-106; `CriticalThreshold` on the Damage option; `NewResolveUnarmedStrikeCommand`) returning
  `ResolveUnarmedStrikeResult`. It is `attackFlow.resolveUnarmedStrike` in the shared recorder, calls `GrappleRules` (FR-128) and never `ResolveAttack` or the `d20` use case.
- Ports: only the consumer-side `Roller` (any `dice.Roller` satisfies it) and the optional
  `telemetry.Telemetry` bundle; no new port or double (existing doubles live in `dicetest` and `telemetrytest`, imported by `_test.go` files only, shared-types-and-ownership.md FR-038). A roller failure comes back as a
  `*srderr.PortError` from `Run`, wrapped once with `srderr.WrapPort` (FR-081); a cancelled `ctx` comes back as `ctx.Err()` and is not a result.
- `attackFlow` is used by both use cases and `combat-action-resolution.md`'s attack operation through the package's own unexported API. No use case calls another (shared-types-and-ownership.md FR-044): each call leaves one span, one `operations` increment and, on a returned failure, one error.
- Exports (FR-108) from package `combat`: both use cases and commands, both results and all variants, `AttackNotPossibleReason`, `MissOverride`,
  `AutomaticMissReason`, `UnarmedStrikeOption` and its variants, `PushFiveFeet`,
  `GiveProne`, `AttackForm` and its five variants, `AttackDelivery`, `AttackSource`,
  `AttackSituation`, `TargetVisibility`, `TargetSeen`, `TargetUnseen`, `NearbyEnemy`,
  `Cover` and its three variants, `CoverSource`, `AttackRange`, `SingleRange`,
  `NormalAndLongRange`, `Reach`, `WeaponUse`, `WeaponUseManner` and its variants,
  `DamageSpec`, `DamageComponent`, `DiceDamage`, `FixedDamage`, and the problem names (`<Subject>Problem`, `<Subject>Field` and its values) for each of
  `AttackRange`, `Reach`, `NearbyEnemy`, `DamageComponent`, `DamageSpec` (FR-107) and the two commands (FR-132). Not re-exported (FR-123):
  `HandAvailability` and `GrappleRules` (`grappling.md`), `d20.Resolver` and the
  `SaveAttempt` family, `RollModifiers`, `AttackRollResult`, `Roll` (`d20`),
  `Damage` and `HitKind` (`hit-points-and-damage.md`), `DamageType` and
  `CreatureSize` (`content`). The shared use-case recorder is unexported.

## 7. Observability

Conventions are `combat-foundations.md`'s (FR-022 to FR-037;
shared-types-and-ownership.md FR-012). Both use cases record through
the shared use-case recorder: the span `srd5e.combat.<operation>`, `operations`,
`operation_duration`, `events` (zero here) and `errors` are shared signals with one
catalogue row each in `combat-foundations.md` (FR-037, FR-040). This spec adds the
extras below, the histogram and the `FINE` logs, with constants in the `combat` signals file (`signals.go`;
combat-foundations.md FR-027): `ValueOperationResolveAttack`, `SpanResolveAttack`, `ValueOperationResolveUnarmedStrike`, `SpanResolveUnarmedStrike` (FR-040), `LogAttackResolved`, `LogUnarmedStrikeResolved`, the `srd5e.combat.attack.*` and `srd5e.combat.unarmed_strike.option` `Attr...` constants and their `Value...` constants.

Evaluation questions (`doc/rules/observability.md` §2): (1) wrong: the roller fails or
returns the wrong shape, impossible host facts, a refused attack the host thought
legal. (2) Known by: a returned port failure is a span error plus `errors`; every rule
outcome (a miss too) is `applied` on `operations`, distinct from `refused` and from
an error; the reason is on counter and span. (3) Diagnose: the result breakdown
(FR-012) plus operation, `srd5e.combat.result`, form, roll mode, natural-20 flag,
cover, reason. (4) Count or time: invocations by operation, result and class, errors,
duration (shared), modifier count. (5) Existing signals unchanged: `srd5e.d20_test.*`
does not fire for attacks made through `combat` (FR-110). (6) Monitor: natural 20
share per roll mode, not-possible by reason, shared errors and duration. (7) Noise:
at most one `FINE` log per call, nothing at `INFO`; the domain is silent.

Operations: `resolve_attack`, `resolve_unarmed_strike`. Closed value sets (shared
keys per combat-foundations.md FR-030 and `observability.md` §5; `srd5e.d20_test.*`
keys reused from the `d20` signal constants, not redefined):

- `srd5e.outcome`, `srd5e.combat.result_class`, `srd5e.combat.result`, `error.type`:
  FR-122, FR-086.
- `srd5e.d20_test.critical_hit` (`true`/`false`, only when an attack roll was made,
  equals `AttackRollResult.IsCriticalHit`) and `srd5e.d20_test.roll_mode` (`normal`,
  `advantage`, `disadvantage`; cancelled Advantage and Disadvantage give `normal`):
  FR-118.
- `srd5e.combat.attack.form` (`resolve_attack` only): `melee_weapon`, `ranged_weapon`,
  `melee_spell`, `ranged_spell`, `unarmed_strike`.
- `srd5e.combat.attack.reason` (only when `not_possible`): `total_cover`,
  `beyond_range`, `beyond_reach`, `target_too_large`, `no_free_hand` (FR-116, FR-117).
- `srd5e.combat.attack.cover` (both operations): `none`, `half`, `three_quarters`, `total`.
- `srd5e.combat.attack.automatic_critical` (both): `true` when the input was
  `automatic`, else `false`.
- `srd5e.combat.unarmed_strike.option` (`resolve_unarmed_strike` only): `damage`,
  `grapple`, `shove`.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.resolve_attack` | span | shared attributes (combat-foundations.md FR-030) plus the extras that apply | Path of one attack; failure record | Any error status |
| `srd5e.combat.resolve_unarmed_strike` | span | same | Path of one Unarmed Strike; failure record | Any error status |
| `srd5e.combat.attack.modifiers` | histogram, `{modifier}` (FR-130) | `srd5e.combat.attack.form` | Modifier set size per returned `ResolveAttack` call | p99 above 20 modifiers per attack (duplicated modifiers) |
| `attack resolved` | log, `FINE` (FR-131) | `srd5e.combat.result` (attack values), closure | One-line outcome for a developer chasing a wrong result | None; enable `FINE` while debugging |
| `unarmed strike resolved` | log, `FINE` (FR-131) | `srd5e.combat.result` (Unarmed Strike values), closure | Same | None |

Extras on the shared `srd5e.combat.operations` row (no second row, FR-041 there):
`srd5e.combat.attack.form`, `.cover`, `.automatic_critical`, `.reason`,
`srd5e.combat.unarmed_strike.option`, `srd5e.d20_test.roll_mode` and
`srd5e.d20_test.critical_hit`. Monitoring to carry on that row: for `resolve_attack`
over an hour with at least 1000 rolled attacks, a `critical_hit` `true` share per
roll mode more than 3 percentage points from 5 percent (`normal`), from 9.75 percent
(`advantage`) or more than 1 percentage point from 0.25 percent (`disadvantage`)
suggests a dice-roller bug; `result_class` `refused` above 10 percent of an
operation's calls for 15 minutes, by `srd5e.combat.attack.reason` (`target_too_large`
and `no_free_hand` point at wrong sizes or hand state, the others at wrong distances
or cover), suggests the host feeds wrong facts; alert on any `errors` increment;
`operation_duration` p99 above 10 ms for 5 minutes or 3 times the 7-day baseline.
Thresholds are starting values. This spec's earlier span and counter names are
superseded (combat-foundations.md FR-029, FR-032) and are not in the catalogue.

The attack-reason, roll-mode and critical-hit requirements are in section 4.5 (FR-116, FR-117, FR-118).

## 8. Test plan

Doubles: fixed `dice.Roller` and recording `telemetrytest` bundle from `dicetest` and `telemetrytest`; no new
port, so no contract suite. Domain tests use no doubles; outcome rows use the real
`d20.Resolver` at use-case level. No use case is faked.

| Id | Level | Notes |
|---|---|---|
| FR-002 to FR-006, FR-008 to FR-012, FR-020 | use case | outcome table with effective AC; exactly the supplied modifiers once, one `1d20` request in `RollMode.normal`; Advantage dice 1 and 20; natural 20 behind cover; breakdown; architecture scan: no file imports the `d20` use case |
| FR-007, FR-129 | unit, use case | Unarmed Strike default is the supplied modifier, spell attack has none; Strength and Dexterity rows for roll, damage, both DCs |
| FR-013 to FR-033 | unit, use case | cover, range, close-combat and reach tables; `AttackNotPossible` draws zero dice; a validation error per invalid value, asserting `errors.Is` `srderr.ErrInvalidInput` and the field with `errors.As` |
| FR-034 to FR-045, FR-103, FR-104, FR-105, FR-113 | unit, use case | circumstance, underwater, combination and precedence tables (Swim Speed with ranged weapon, Unarmed Strike, Piercing in a `DamageSpec`); `TargetVisibility` types |
| FR-046 to FR-050, FR-100, FR-101, FR-126 | unit | `WeaponUse` (derived `IsImprovised`, zero category, manner and type contradiction) and command constructors with `RollModifiers` that do or do not carry a Proficiency Bonus |
| FR-051 to FR-062, FR-096 to FR-098, FR-114 | unit, use case | damage table; fixed roller drives draws; `Damage` conversion |
| FR-063 to FR-069, FR-073 to FR-078, FR-106, FR-124, FR-128 | unit, use case | Unarmed Strike table; `UnarmedStrikeModifiers` rows (unit); `GrappleRules` called with the command's values (DC, `GrappleNotPossible` mapped, no dice); save rolled through `attackFlow` and `d20.Resolver` |
| FR-087 to FR-089, FR-111 | unit, use case | `AttackForm` delivery and source; melee and ranged spell rows; `AttackResolved` delivery |
| FR-090, FR-091 | use case | automatic Critical Hit and forced-miss rows |
| FR-092 to FR-095 | unit, use case | cover appended once and only to a Dexterity `RollAttempt`; Total Cover; other circumstances unchanged; forced failures draw no die |
| FR-107, FR-132 | unit | `NewX` returns every violated field in the joined error (deterministic order), an error if and only if a field is violated; `MustX` panics with that error and never otherwise |
| FR-108, FR-120, FR-123, FR-125 | scenario, architecture | importing only public packages builds each command and switches exhaustively over each result (exhaustive type switch in a scenario test; the planned `gochecksumtype` lint is future work, not yet in `.golangci.yml`), each export with a doc comment; package API scan: no generic or duplicate variant name (`AttackForm` vs `character` kinds, `AttackNotPossibleReason` vs `GrappleNotPossibleReason`, the problem and field names); `CircumstanceCollector` returns `AttackPlan`, mentions no use case |
| FR-079 to FR-082, FR-112 | unit, use case | same command twice; immutability; failing roller double (dicetest.CallbackRoller (returning an error)) in the attack roll, a damage roll and the save d20 (error span, no log on a failure); wrong-shape result; all-minimum and all-maximum rolls |
| FR-083 to FR-086, FR-110, FR-115 to FR-118, FR-122 | use case | recording `telemetrytest` bundle: shared span, `operations`, `operation_duration` once, zero `events`; extras' keys and values; `not_possible` is `refused`, outcome `failure`, no error status; a miss and a resisted Grapple are `applied`; reason only for not-possible; `roll_mode` and `critical_hit` omitted when no d20 was rolled; `critical_hit` equals the natural-20 flag, including a forced miss and an automatic Critical Hit; `roll_mode` is present on the save d20 and absent for `ChooseToFail`; one error for the operation called and no `d20.errors`; every `result` value produced by a table row |
| FR-109, FR-119, FR-127 | architecture, use case | no condition fixture in any attack or Unarmed Strike test; constructors take only a roller and optional `telemetry.Telemetry`; mention scan: no `ConditionEffects`, no the `d20` use case, no use case mentions the other, `attackFlow` mentions no use case or `telemetry.Telemetry` and is exported by no package; `attackFlow` entry points equal the use cases' results and record no signal |
| FR-121, FR-134, FR-135 | unit, use case | threshold 19 and 18 with natural 19 / 18 / 17 (hit and Critical Hit at any total), natural 20 and natural 1 unchanged, forced miss wins, value 17 or 21 rejected; `WithAdditions` appends, ORs, keeps the rest, leaves the original and returns an error exactly where the constructor does; carried `AttackRollResult` equals the `d20` result; `MissOverride`, `AutomaticMissReason` and automatic Critical Hit derived without a second enum |
| FR-130, FR-131, FR-133 | unit, use case | histogram sample count and form attribute (none on a failure or Unarmed Strike); `FINE` message and closure attribute, no log on a failure; `ResolveSave` and `ResolveCheck` rolled, with Disadvantage, forced |
| scenario | scenario | through public packages only (`scenarios_test`): an attack with Half Cover, long range and a Critical Hit with extra dice; a Grapple behind Half Cover against a Dexterity save |

- Mutation testing: `gomu` on package `combat`, threshold 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (`doc/rules/testing.md` §8); never as a bare host process.
- Not covered: a spell attack's default ability beyond "none" (`character`'s); the
  natural 20 and 1 maths (`d20` owns and tests it).
- Every live FR id appears in a row above; withdrawn ids are in §4.6 and have no test.

## 9. Open questions and decisions

Settled by `shared-types-and-ownership.md` (FR-002, FR-003, FR-004, FR-005, FR-012,
FR-026, FR-027, FR-029, FR-044, FR-046, FR-048), `grappling.md` Q-14 and
`combat-foundations.md` FR-041. Package decisions are stated in FR-019, FR-036,
FR-041, FR-059, FR-087, FR-096, FR-097, FR-113 and FR-122.

Settled, one line each (no web research needed; all are package-design questions):

- **`...NotPossible` names.** Kept (FR-120): they are `refused` results (FR-086), not rejected input, so combat-foundations.md FR-023's `<Operation>Rejected` does not fit, and `combat-action-resolution.md` consumes the names. Follow-ups: `combat-foundations.md` FR-023 records the exception; `combat-action-resolution.md` renames `NotPossibleReason` to `AttackNotPossibleReason` (amendments).
- **Small amendments.** `ProficiencyBonus` exposes `Value() int` (FR-124; `derived-stats.md` FR-059 adds it). `attack-and-help-actions.md` corrects its `AttackKind` reference to `AttackForm` (FR-087); `AttackKind` is not a name here. Critical range (FR-134, from `class-fighter.md` FR-090): the SRD defines a Critical Hit as hitting regardless of AC (p. 179), so a threshold roll hits at any total (interpretation); the d20 input itself is `d20-tests.md`'s to add, and until it does FR-134 is blocked on it. The copy operation (FR-135, from `combat-action-resolution.md` Q-6) is a package decision.

No question is escalated.

## 10. Changelog impact

Added: `ResolveAttack`, `ResolveUnarmedStrike`, the attack, cover, range, damage and
Unarmed Strike types and the signals in §7. Not **BREAKING**: new public API and new
signals only.
