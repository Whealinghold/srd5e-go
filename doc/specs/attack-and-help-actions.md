# Attack action detail, Extra Attack across sources, and the Help action

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `modifiers-and-advantage.md`, `content-schema-and-decoder.md` |

## 1. Purpose and scope

The parts of the Attack action and the Help action that no existing spec
specifies, as pure domain logic over immutable values: how many attacks of this
Attack action have been made against the allowance, whether a weapon may be drawn
or stowed now, whether the combatant is between two attacks, and what benefit
Help leaves behind, who may use it and when it goes away.

Decisions applied (`shared-types-and-ownership.md`, which wins over this spec):
FR-005, FR-012 to FR-015, FR-017, FR-019, FR-021, FR-026, FR-038, FR-039,
FR-041, FR-044, FR-046, FR-048, FR-052. FR-036 and FR-037 do not apply to a
domain-only spec.

Imports (FR-057): the `content` package (`Skill`, `ItemID`, `FeatureID`, owned by
`content-schema-and-decoder.md`), the kernel (`Advantage`, `modifier.RollCircumstance`,
`modifiers-and-advantage.md` FR-042) and, inside `combat`, only
`combat-foundations.md` (`CombatantID`, `CombatEvent`, FR-015 to FR-024).

Referenced in prose only, never imported (nothing from them is needed; the host
passes plain integers and values; some are of higher rank or in another
context): `level-advancement-and-multiclassing.md`, `class-warlock.md`,
`weapon-properties-and-mastery.md`, `initiative-and-actions.md`, `attacks.md`,
`hit-points-and-damage.md`, `derived-stats.md`, and
`combat-action-resolution.md`, the only consumer (its FR-112 to FR-123 own the
use cases that call these types; they call no other use case, shared-types
FR-044). Position in the `combat` order: Q-06 (decided). Package `combat`
exports every type in section 5, the variants of FR-059, the groups of FR-071
and the results of FR-072; there is no facade or re-export package.

Ownership:

- `initiative-and-actions.md` records that Attack or Help was taken and used the
  action (its FR-050, FR-114); it resolves no effect. This spec supplies the
  effect side and redefines no `StandardAction`, movement, free interaction
  (its FR-058) or tracker event.
- `attacks.md` resolves one attack however granted; this spec never calls it.
  The host or `combat-action-resolution.md` resolves with `attacks.md`, then
  records here. `AttackMeans` is bookkeeping of the kind of attack, not
  `AttackForm` (`attacks.md` FR-087); `combat-action-resolution.md` FR-141 and
  FR-142 own the rule that the two must match.
- **The number of attacks is not computed here.** The multiclass Extra Attack
  combination is `level-advancement-and-multiclassing.md`
  (`ExtraAttackCombination`, FR-054 to FR-058), from the counts the class specs report (`class-fighter.md` FR-026, FR-086;
  `class-barbarian.md` FR-056; `class-monk.md` FR-075; `class-paladin.md`
  FR-085; `class-ranger.md` FR-065, FR-068). The pact weapon count, with
  Thirsting Blade and Devouring Blade, is `class-warlock.md`
  (`PactWeaponAttackCount`, FR-079, FR-155). This spec takes both as plain
  integers in `AttackAllowance(general, pact)` (FR-066).
- `weapon-properties-and-mastery.md` owns weapon properties, `WeaponTurnState`
  and `WeaponWielding`. `AttackActionProgress` counts attacks and slots;
  `WeaponTurnState` records only the per-turn Light, Cleave and Nick facts;
  neither reads the other (FR-067). `AttackActionWeaponChange` never collides
  with `WeaponWielding` (shared-types FR-021).
- **Topic ownership (shared-types FR-048).** This spec owns the Attack action
  mechanics (counting, replaced attacks, weapon changes, moving between attacks)
  and the Help action (giving, using, expiring, dropping). The older specs
  already reduce to references; a later restatement is withdrawn "moved to
  attack-and-help-actions.md" by its own reconcile.
- The Advantage Help grants is a `modifier.RollCircumstance`; this spec returns it and
  resolves no roll mode.

**Out of scope:**

- Resolving an attack roll or damage; the Unarmed Strike's damage, Grapple and
  Shove (`attacks.md`, `d20-tests.md`).
- Opportunity Attacks and the Reaction economy ("Opportunity Attacks", p. 185).
  An Opportunity Attack is never recorded here (FR-024).
- The Bonus Action economy and weapon property rules (Light, two-weapon
  attacks, Martial Arts). The Nick extra attack is made "as part of the Attack
  action" (p. 90): an extra attack outside the allowance, recorded in
  `WeaponTurnState` (FR-063, Q-16).
- Spell attack rolls (casting is the Magic action; p. 177).
- The effect of a replaced attack (Breath Weapon, thrown Acid, the familiar's
  attack under Pact of the Chain, ending Quivering Palm): this spec records only
  that one attack was replaced or forgone (FR-062); the effect belongs to the
  owning specs (`content-species-backgrounds-feats.md`, `content-equipment.md`, `item-use-effects.md` (thrown Acid, Alchemist's Fire, Holy Water, Oil, Net),
  `class-warlock.md` FR-091, `class-monk.md` FR-118). Action Surge's limits
  (p. 48) beyond FR-023 belong to `class-fighter.md`.
- Monsters' Multiattack (`monster-stat-block-actions.md`).
- First aid: SRD "Stabilizing a Character", p. 18, and "Knocking Out a
  Creature", p. 17, owned by `hit-points-and-damage.md` (FR-069) and
  `combat-action-resolution.md` (`AttemptToStabilizeCombatant`). The Help
  [Action] entry (p. 182) lists only the two options modelled here.
- Allegiance, distances, "near enough ... to assist" and the Game Master's call
  on whether assistance is possible: typed host input (FR-036, FR-039, FR-040,
  Q-07).
- Help outside an encounter (no tracker; the SRD's example is p. 10): the host
  ends benefits with `DropAll` and `DropFor` (FR-069, Q-14).
- Display text.

## 2. Source

The printed page equals the PDF page. SRD 5.2.1: "Attack [Action]", "Equipping
and Unequipping Weapons", "Moving between Attacks", p. 177; "Multiclassing",
"Extra Attack", p. 25; "Fighter" (Extra Attack, Two Extra Attacks, Three Extra
Attacks, Action Surge), p. 48; "Warrior of the Open Hand", "Quivering Palm", p.
52; "Equipment Proficiencies" (Tools), p. 9; "Warlock", "Devouring Blade", p. 72,
"Thirsting Blade", "Pact of the Chain", p. 74; "Breath Weapon", p. 84; "Acid",
"Alchemist's Fire", p. 94; "Holy Water", p. 97; "Net", "Oil", p. 98; "Nick", p.
90; "Light", p. 89; "Help [Action]", pp. 182-183; "Ally", p. 176; "Enemy", p.
181; Actions table, pp. 9-10; "Stabilizing a Character", p. 18 and "Knocking
Out a Creature", p. 17 (only to say where first aid lives); "They Don't Stack",
p. 8; "Interacting with Things", p. 13; "Opportunity Attacks", p. 185.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Attack [Action]: "one attack roll with a weapon or an Unarmed Strike" | `AttackMeans` (enum: `PactWeapon`, `OtherWeapon`, `UnarmedStrike`) | `PactWeapon` exists because Thirsting Blade's extra attack is "for your pact weapon only" (p. 74). A spell is not representable (FR-001). Not `AttackForm`. |
| "replace one of your attacks"; "forgo one of your own attacks" | `AttackReplacement` with `AttackReplacementKind` (`replace`, `forgo`) and `AttackReplacementSource` (closed: `ReplacementByFeature(FeatureID)`, `ReplacementByItem(ItemID)`) | Breath Weapon (`replace`, p. 84), thrown Acid, Alchemist's Fire, Holy Water, Net, Oil (`replace`, pp. 94, 97, 98), Pact of the Chain and Quivering Palm (`forgo`, pp. 74, 52). |
| (package term) the count the owners give | `AttackAllowance` | `GeneralAttacks` (any means) and `PactWeaponAttacks` (an action made entirely with the pact weapon, replacements allowed, FR-016). |
| Attack action in progress | `AttackActionProgress` | `attacker` (FR-077), allowance, attacks so far (means or replacements), used weapon-change slots. |
| Equipping and Unequipping Weapons | `AttackActionWeaponChange` (`DrawFromSheath`, `PickUp`, `EquipOther`, `sheathe`, `stow`, `drop`, `UnequipOther`) with `WeaponChangeKind` (`equip`, `unequip`) | Drawing and picking up are equipping; sheathing, stowing, dropping are unequipping (p. 177). |
| (package term) before or after the attack | `AttackActionWeaponChangeTiming` (`BeforeAttack`, `AfterAttack`) | |
| Moving between Attacks | `AttackActionProgress.IsBetweenAttacks` | A query. |
| Help: Assist an Ability Check | `AbilityCheckHelp` (a `HelpBenefit`) | Helper, ally, `AssistedProficiency`. `NewAbilityCheckHelp` returns a validation error when helper is ally (FR-074). |
| Help: Assist an Attack Roll | `AttackRollHelp` (a `HelpBenefit`) | Helper and enemy. `NewAttackRollHelp` returns a validation error when helper is enemy (FR-074). |
| "skill or tool proficiencies" | `AssistedProficiency` (closed: `AssistedSkill(Skill)`, `AssistedTool(ItemID)`) | `Skill` and `ItemID` are `content` vocabulary (shared-types FR-005, FR-046). |
| (package term) proficiencies of the creature that took Help | `HelpGiverProficiencies` | Skills and tools the creature is proficient in. A monster has no `SkillProficiencies` (`derived-stats.md` FR-016), so the caller builds this from either source (Q-15). |
| (package term) "helper" | prose and parameter names only | The creature that took Help; no type name contains `Helper` (architecture.md section 3). |
| Ally (p. 176); Enemy (p. 181) | `AlliedHelpers`, `AttackAssistTarget` | The host decides allegiance (Q-07). |
| the Game Master's final say | `AssistanceJudgement` (`possible`, `NotPossible`) | Host input. |
| the enemy within 5 feet | `AttackAssistTarget` (`EnemyWithinFiveFeet`, `EnemyBeyondFiveFeet`, `NotAnEnemy`) | Host input; no map or allegiance model. |
| the benefits Help has left | `HelpBenefits` | Immutable collection of pending `HelpBenefit`s. |
| "expires if unused before the start of your next turn" | `HelpBenefits.ExpireAtStartOfTurn(CombatantID)` returning `HelpBenefitsExpiry(benefits, events)`; `DropFor` / `DropAll` returning `HelpBenefitsDrop(benefits, events)` | Called when `TrackerTurnStarted` (`initiative-and-actions.md` FR-125) names the helper (FR-072). |

## 4. Requirements

### 4.1 Attack means and the baseline

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `AttackMeans` MUST have exactly three values: `PactWeapon`, `OtherWeapon` and `UnarmedStrike`. No value MUST stand for a spell attack. An attack that replaces or forgoes an attack is an `AttackReplacement` (FR-062), not an `AttackMeans`. An exhaustive `switch` over `AttackMeans` MUST compile with no default. | SRD 5.2.1, "Attack [Action]", p. 177 | happy |
| FR-002 | `AttackAllowance.baseline` MUST be one attack for every means (`GeneralAttacks` 1, `PactWeaponAttacks` 1). | SRD 5.2.1, "Attack [Action]", p. 177 | unhappy (empty case) |

### 4.2 The allowance (input, not computed here)

Table A. Each row is one table-driven test of the `NewAttackAllowance` constructor
and names the owner whose output the host passes in. Written `(general, pact)`.

| Row | `GeneralAttacks` | `PactWeaponAttacks` | Produced by | Source | Path |
|---|---|---|---|---|---|
| A01 | 1 | 1 | no Extra Attack (`baseline`) | "Attack [Action]", p. 177 | unhappy (empty case) |
| A02 | 2 | 2 | `ExtraAttackCombination` gives 2 | "Level 5: Extra Attack", p. 48 | happy |
| A05 | 3 | 3 | `ExtraAttackCombination` gives 3 | "Level 11: Two Extra Attacks", p. 48 | happy |
| A06 | 4 | 4 | `ExtraAttackCombination` gives 4 | "Level 20: Three Extra Attacks", p. 48 | happy |
| A10 | 1 | 2 | `PactWeaponAttackCount` with Thirsting Blade | "Thirsting Blade", p. 74 | happy |
| A15 | 1 | 3 | `PactWeaponAttackCount` with Thirsting Blade and Devouring Blade (`class-warlock.md` FR-079) | "Devouring Blade", p. 72; "Thirsting Blade", p. 74 | happy |
| A16 | 0 | 1 | invalid | package design | unhappy (invalid input) |
| A17 | 2 | 1 | invalid (pact below general) | package design | unhappy (invalid input) |
| A18 | 2 | 3 | Extra Attack 2 and `PactWeaponAttackCount` 3; reachable only if `class-warlock.md` Q-16 / TB-07 allows it | "Thirsting Blade", p. 74; "Level 5: Extra Attack", p. 48 | happy (boundary: pact above general, both above 1) |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | `NewAttackAllowance(generalAttacks, pactWeaponAttacks int) (AttackAllowance, error)` MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) unless `GeneralAttacks` is at least 1 and `PactWeaponAttacks` is at least `GeneralAttacks` (A16, A17). | package design (invariant) | unhappy (invalid input) |
| FR-066 | `NewAttackAllowance(generalAttacks, pactWeaponAttacks)` is the typed input of `StartAttackActionProgress`. The host builds it from `ExtraAttackCombination.AttacksPerAttackAction` (1 with no Extra Attack) and `PactWeaponAttackCount`, which already includes Thirsting Blade and Devouring Blade (A10, A15). Package `combat` MUST export no source, scope or combine type or function for attack counts. | SRD 5.2.1, "Extra Attack" under "Multiclassing", p. 25; "Thirsting Blade", p. 74; "Devouring Blade", p. 72 | unhappy (architecture) |

### 4.3 The Attack action in progress

Table B. Each row is one table-driven test of `RecordAttack` (or
`RecordReplacement` where the next entry is `R`). Allowance `(general, pact)`;
`P` is `PactWeapon`, `O` `OtherWeapon`, `U` `UnarmedStrike`, `R` an
`AttackReplacement`.

| Row | Allowance | History | Next | Outcome | Path |
|---|---|---|---|---|---|
| B01 | (1,1) | none | O | recorded, attack 1 | happy |
| B02 | (1,1) | O | O | refused `AttackLimitReached` | unhappy (boundary) |
| B03 | (2,2) | O | U | recorded, attack 2 | happy |
| B04 | (2,2) | O, U | O | refused `AttackLimitReached` | unhappy (boundary) |
| B05 | (1,2) | none | P | recorded, attack 1 | happy |
| B06 | (1,2) | P | P | recorded, attack 2 | happy |
| B07 | (1,2) | P, P | P | refused `AttackLimitReached` | unhappy (boundary) |
| B08 | (1,2) | P | O | refused `AttackLimitReached` | unhappy (rule interaction) |
| B09 | (1,2) | O | P | refused `AttackLimitReached` | unhappy (rule interaction) |
| B10 | (1,2) | none | O | recorded, attack 1 | happy |
| B11 | (1,2) | P | U | refused `AttackLimitReached` | unhappy (rule interaction) |
| B12 | (3,3) | O, O | P | recorded, attack 3 | happy |
| B13 | (3,3) | O, O, P | O | refused `AttackLimitReached` | unhappy (boundary) |
| B14 | (4,4) | U, O, O, O | O | refused `AttackLimitReached` | unhappy (boundary) |
| B15 | (1,3) | none | P | recorded, attack 1 | happy |
| B16 | (1,3) | P, P | P | recorded, attack 3 | happy |
| B17 | (1,3) | P, P, P | P | refused `AttackLimitReached` | unhappy (boundary) |
| B18 | (1,3) | P, P | O | refused `AttackLimitReached` | unhappy (rule interaction) |
| B19 | (2,2) | R | O | recorded, attack 2 | happy |
| B20 | (2,2) | R, O | O | refused `AttackLimitReached` | unhappy (boundary) |
| B21 | (1,1) | R | O | refused `AttackLimitReached` | unhappy (boundary) |
| B22 | (1,2) | R | P | recorded, attack 2 | unhappy (rule interaction) |
| B23 | (1,2) | R | R | refused `AttackLimitReached` | unhappy (rule interaction) |
| B24 | (1,2) | P | R | recorded, attack 2 | unhappy (rule interaction) |
| B25 | (1,2) | P, R | P | refused `AttackLimitReached` | unhappy (boundary) |
| B26 | (1,2) | O | R | refused `AttackLimitReached` | unhappy (rule interaction) |
| B27 | (2,2) | none | R | recorded, attack 1 | happy |
| B28 | (2,3) | none | P | recorded, attack 1 | happy |
| B29 | (2,3) | P, P | P | recorded, attack 3 (pact limit 3) | happy |
| B30 | (2,3) | P, P, P | P | refused `AttackLimitReached` | unhappy (boundary) |
| B31 | (2,3) | P, P | O | refused `AttackLimitReached` (non-pact attack: general limit 2) | unhappy (rule interaction) |
| B32 | (2,3) | P | O | recorded, attack 2 (general limit 2) | unhappy (rule interaction) |
| B33 | (2,3) | P, O | O | refused `AttackLimitReached` (general limit 2) | unhappy (boundary) |
| B34 | (2,3) | O | P | recorded, attack 2 (general limit 2) | unhappy (rule interaction) |
| B35 | (2,3) | O, P | P | refused `AttackLimitReached` (general limit 2) | unhappy (boundary) |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `StartAttackActionProgress(CombatantID, AttackAllowance)` MUST return a progress with no attack recorded, no weapon change used, and an `AttacksRemaining` equal to `PactWeaponAttacks` (the most the allowance can ever give). | SRD 5.2.1, "Attack [Action]", p. 177 | happy |
| FR-015 | `RecordAttack(AttackMeans)` MUST return `AttackActionAttackRecorded` for every Table B row marked recorded with a means: a new progress with the means appended and an `AttackMadeInAttackAction` event naming the combatant, the means and the 1-based attack number; and `AttackActionAttackRefused(AttackRefusal.AttackLimitReached)` with the progress unchanged for every row marked refused. | SRD 5.2.1, "Attack [Action]", "Moving between Attacks", p. 177 | per row: happy / unhappy (boundary) |
| FR-016 | The limit for the next attack (a means or a replacement) MUST be `PactWeaponAttacks` when every attack recorded so far and the next one is a `PactWeapon` attack or a replacement, and at least one of them is a `PactWeapon` attack; and `GeneralAttacks` otherwise (B22 to B26; B28 to B35 for an allowance whose two limits differ and both exceed 1, where the second attack of B32 and B34 is allowed only by `GeneralAttacks` 2 and the third attack of B29 only by `PactWeaponAttacks` 3). | SRD 5.2.1, "Thirsting Blade", p. 74 | unhappy (rule interaction) |
| FR-017 | An `UnarmedStrike` MUST count as one attack against the same limit as a weapon attack, and MAY be mixed with weapon attacks within the allowance (B03). | SRD 5.2.1, "Attack [Action]", p. 177 ("with a weapon or an Unarmed Strike") | happy |
| FR-018 | `AttacksRemaining` MUST be `PactWeaponAttacks` minus the attacks recorded when every recorded attack is a `PactWeapon` attack or a replacement (an empty history included), and `GeneralAttacks` minus the attacks recorded for any other history; it MUST never be negative. | SRD 5.2.1, "Attack [Action]", p. 177 | unhappy (boundary) |
| FR-019 | It MUST NOT be possible to build a progress that has recorded more attacks than the limit: the `NewAttackActionProgress` constructor MUST enforce FR-016 on every prefix of the history (each attack checked against the limit for the attacks before it) and return a validation error otherwise; (1,3) with [R, R, P] returns an error, [R, P, P] builds. | architecture.md section 3 (invariants) | unhappy (invalid input) |
| FR-020 | `RecordAttack` and every other operation MUST return a result whose progress is a new instance that differs from the one it was called on when something was recorded, and a progress equal by value to it when nothing changes (a refusal; the same instance is permitted); the instance it was called on MUST stay equal to what it was; the `Attacks()` view MUST be returned as a copy. | architecture.md section 3 | unhappy (immutability) |
| FR-021 | Two `AttackActionProgress` values with equal fields MUST be equal with `==` (or an `Equal` method where the type holds a slice). | architecture.md section 3 | unhappy (boundary) |
| FR-023 | Each Attack action MUST have its own progress. A second Attack action in the same turn (for example from Action Surge) MUST start a new progress with its own count and its own weapon-change slots, unaffected by the first. | SRD 5.2.1, "Attack [Action]", p. 177; "Level 2: Action Surge", p. 48 ("one additional action, except the Magic action") | unhappy (state error) |
| FR-024 | The public API of `AttackActionProgress` MUST have no member for an Opportunity Attack or any other Reaction attack: the architecture scan lists its public members and fails on a name containing `Opportunity` or `Reaction`. | SRD 5.2.1, "Opportunity Attacks", p. 185; "Attack [Action]", p. 177 | unhappy (rule interaction) |
| FR-062 | `RecordReplacement(AttackReplacement)` MUST count one attack of the allowance against the limit of FR-016 and return `AttackActionAttackRecorded` with an `AttackReplacedInAttackAction` event naming the combatant, the `AttackReplacementKind`, the `AttackReplacementSource` and the 1-based attack number, or `AttackActionAttackRefused(AttackRefusal.AttackLimitReached)` with the progress unchanged (B19 to B27). For the weapon-change slots (FR-025 to FR-028) a replacement counts as an attack made. A `BreathWeaponEffect` with `ReplacesOneAttack` (`species-trait-behaviour.md` FR-034) is recorded by the host as `AttackReplacement(replace, ReplacementByFeature(id))` with the Breath Weapon trait's `FeatureID`; no new `AttackReplacementSource` variant is needed (species traits are features, `content-schema-and-decoder.md` FR-114), and the effect's Dexterity save, damage and DC are resolved by the owning spec, not here. Whether Breath Weapon, thrown Acid, Alchemist's Fire, Holy Water, Net or Oil, Pact of the Chain or Quivering Palm may be used is the owning spec's rule: `item-use-effects.md` for the thrown items (the Net: `deployed-and-restraining-gear.md` ThrowNet, NetRules FR-035 to FR-039), `class-warlock.md`, `class-monk.md`. | SRD 5.2.1, "Breath Weapon", p. 84; "Acid", "Alchemist's Fire", p. 94; "Holy Water", p. 97; "Net", "Oil", p. 98; "Pact of the Chain", p. 74; "Quivering Palm", p. 52 | per row: happy / unhappy (boundary) |
| FR-063 | The Nick extra attack ("as part of the Attack action instead of as a Bonus Action") is an extra attack outside the allowance: `AttackActionProgress` MUST have no operation for it and MUST report the same `AttacksRemaining` and `IsBetweenAttacks` before and after the host records it in `WeaponTurnState`. This spec reads the Nick attack as having no weapon-change slot of its own, so the slots are reported unchanged too (Q-16). Its cost is `ExtraAttackCost.PartOfAttackAction` (`weapon-properties-and-mastery.md` FR-057, FR-063, FR-064). | SRD 5.2.1, "Nick", p. 90; "Light", p. 89; "Equipping and Unequipping Weapons", p. 177 (Q-16) | unhappy (rule interaction) |
| FR-064 | `NewAttackActionProgress` MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) for a used weapon-change slot that is below 1, that appears twice, or that is above the number of attacks recorded plus one; the slot after the last attack recorded (made + 1) is valid only while `AttacksRemaining` is at least 1. | architecture.md section 3 (invariants); package design (FR-025) | unhappy (invalid input) |

### 4.4 Equipping and unequipping weapons

The SRD allows one equip or unequip "when you make an attack as part of this
action", before or after the attack. This spec reads that as one change per
attack, using that attack's slot (Q-02, settled). A slot is the 1-based number of an
attack. A change before the next attack uses slot *made + 1*; a change after the
last attack made uses slot *made*.

Table C. Each row is one table-driven test of `HandleWeapon`. `B` is a change
before the next attack, `A` a change after the last attack made, `atk` a
recorded attack.

| Row | Allowance | Sequence so far | Request | Outcome | Path |
|---|---|---|---|---|---|
| C01 | (1,1) | none | B, `DrawFromSheath` | recorded, slot 1 | happy |
| C02 | (1,1) | none | B, `sheathe` | recorded, slot 1 | happy |
| C03 | (1,1) | none | A, `stow` | refused `NoAttackMade` | unhappy (state error) |
| C04 | (1,1) | atk | A, `drop` | recorded, slot 1 | happy |
| C05 | (1,1) | atk | B, `PickUp` | refused `NoAttackRemaining` | unhappy (boundary) |
| C06 | (1,1) | B | B | refused `WeaponChangeAlreadyUsedForThisAttack` | unhappy (boundary) |
| C07 | (1,1) | B, atk | A | refused `WeaponChangeAlreadyUsedForThisAttack` | unhappy (boundary) |
| C08 | (1,1) | atk, A | A | refused `WeaponChangeAlreadyUsedForThisAttack` | unhappy (boundary) |
| C09 | (2,2) | atk, A | B (before attack 2) | recorded, slot 2 | happy |
| C10 | (2,2) | atk, A, B | B | refused `WeaponChangeAlreadyUsedForThisAttack` | unhappy (boundary) |
| C11 | (2,2) | atk, atk | A | recorded, slot 2 | happy |
| C12 | (2,2) | atk, atk, A | A | refused `WeaponChangeAlreadyUsedForThisAttack` | unhappy (boundary) |
| C13 | (2,2) | B, atk, atk | A (attack 2 slot unused) | recorded, slot 2 | happy |
| C14 | (2,2) | atk, atk | B | refused `NoAttackRemaining` | unhappy (boundary) |
| C15 | (1,2) | P | B (pact weapon, so one more attack is possible) | recorded, slot 2 | happy |
| C16 | (1,2) | O | B (no further attack is possible after an other-weapon attack) | refused `NoAttackRemaining` | unhappy (rule interaction) |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-025 | `HandleWeapon(AttackActionWeaponChange, AttackActionWeaponChangeTiming)` MUST return `AttackActionWeaponChangeRecorded` for every Table C row marked recorded: a new progress with that attack slot marked used and a `WeaponChangedInAttackAction` event naming the combatant, the change, its `change`, the timing and the slot; and `AttackActionWeaponChangeRefused` carrying a `WeaponChangeRefusal` (`NoAttackMade`, `NoAttackRemaining`, `WeaponChangeAlreadyUsedForThisAttack`) for every row marked refused, with the progress unchanged. | SRD 5.2.1, "Equipping and Unequipping Weapons", p. 177 | per row: happy / unhappy (boundary) |
| FR-026 | A change before an attack MUST be refused with `NoAttackRemaining` when `AttacksRemaining` is 0. | SRD 5.2.1, "Equipping and Unequipping Weapons", p. 177 ("when you make an attack") | unhappy (boundary) |
| FR-027 | A change after an attack MUST be refused with `NoAttackMade` when no attack has been recorded. | SRD 5.2.1, "Equipping and Unequipping Weapons", p. 177 | unhappy (state error) |
| FR-028 | A second change in the slot of the same attack, before or after it, MUST be refused with `WeaponChangeAlreadyUsedForThisAttack`. | SRD 5.2.1, "Equipping and Unequipping Weapons", p. 177 ("equip or unequip one weapon") | unhappy (rule interaction) |
| FR-029 | `AttackActionWeaponChange.change` (a `WeaponChangeKind`) MUST be `equip` for `DrawFromSheath`, `PickUp` and `EquipOther`, and `unequip` for `sheathe`, `stow`, `drop` and `UnequipOther`. | SRD 5.2.1, "Equipping and Unequipping Weapons", p. 177 ("Equipping a weapon includes drawing it from a sheath or picking it up. Unequipping a weapon includes sheathing, stowing, or dropping it.") | happy |
| FR-030 | A weapon equipped before an attack MUST NOT restrict the next attack's means: `RecordAttack` MUST accept any `AttackMeans` the limit allows, whatever was equipped. The progress MUST NOT record which weapon is held. | SRD 5.2.1, "Equipping and Unequipping Weapons", p. 177 ("you don't have to use it for that attack") | happy |
| FR-031 | A change under this spec uses neither the turn's free interaction nor movement: no public member of `AttackActionProgress` MUST take or return a type of `initiative-and-actions.md` (`TurnTracker`, `InteractWithObject`, movement), checked by the architecture scan, so a combatant that equips here still has its free interaction (`initiative-and-actions.md` FR-058). | SRD 5.2.1, "Equipping and Unequipping Weapons", p. 177; "Interacting with Things", p. 13 | unhappy (rule interaction) |

### 4.5 Moving between attacks

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-032 | `IsBetweenAttacks` MUST be true exactly when at least one attack is recorded and `AttacksRemaining` is at least 1. | SRD 5.2.1, "Moving between Attacks", p. 177 | happy |
| FR-033 | `IsBetweenAttacks` MUST be false before the first attack and after the last attack the history allows, including for an allowance of one attack (B01, B02): with one attack allowed there is no attack to move between. | SRD 5.2.1, "Moving between Attacks", p. 177 ("a feature, such as Extra Attack, that gives you more than one attack") | unhappy (boundary) |
| FR-034 | `IsBetweenAttacks` MUST be a pure query: repeated calls leave the progress equal to what it was and the next `RecordAttack` and `HandleWeapon` give the same result as on a control progress never queried. The progress has no operation for movement; its cost and legality stay with `initiative-and-actions.md` (FR-059). | SRD 5.2.1, "Moving between Attacks", p. 177 ("use some or all of that movement to move between those attacks") | unhappy (rule interaction) |

### 4.6 Giving Help

Table D. Each row is one table-driven test of `HelpBenefits.GiveAbilityCheckHelp`.
Helper `H`, ally `Y`. "Lacks" means the helper's `HelpGiverProficiencies` does not
contain the chosen skill or tool.

| Row | Helper and ally | Proficiency | Judgement | Outcome | Path |
|---|---|---|---|---|---|
| D01 | H, Y | skill, held | possible | given | happy |
| D02 | H, Y | tool, held | possible | given | happy |
| D03 | H, Y | skill, lacks | possible | refused `HelperLacksProficiency` | unhappy (invalid input) |
| D04 | H, Y | tool, lacks | possible | refused `HelperLacksProficiency` | unhappy (invalid input) |
| D05 | H, Y | skill, held | notPossible | refused `AssistanceNotPossible` | unhappy (state error) |
| D06 | H, H | skill, held | possible | refused `CannotHelpSelf` | unhappy (invalid input) |
| D07 | H, Y | skill, lacks | notPossible | refused `HelperLacksProficiency` | unhappy (rule interaction) |
| D08 | H, H | skill, lacks | notPossible | refused `CannotHelpSelf` | unhappy (rule interaction) |
| D09 | H, Y | none held at all (empty `HelpGiverProficiencies`) | possible | refused `HelperLacksProficiency` | unhappy (empty case) |

Table E. Each row is one table-driven test of `HelpBenefits.GiveAttackRollHelp`.

| Row | `AttackAssistTarget` | Outcome | Path |
|---|---|---|---|
| E01 | `EnemyWithinFiveFeet` | given | happy |
| E02 | `EnemyBeyondFiveFeet` | refused `EnemyNotWithinFiveFeet` | unhappy (boundary) |
| E03 | `NotAnEnemy` | refused `TargetNotAnEnemy` | unhappy (invalid input) |
| E04 | `EnemyWithinFiveFeet`, helper's own empty `HelpGiverProficiencies` | given | unhappy (empty case) |
| E05 | `EnemyWithinFiveFeet`, the named enemy is the helper itself | refused `TargetNotAnEnemy` | unhappy (invalid input) |
| E06 | `EnemyBeyondFiveFeet`, the named enemy is the helper itself | refused `TargetNotAnEnemy` (not `EnemyNotWithinFiveFeet`; FR-075) | unhappy (rule interaction) |
| E07 | `NotAnEnemy`, the named enemy is the helper itself | refused `TargetNotAnEnemy` | unhappy (rule interaction) |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-035 | `GiveAbilityCheckHelp` MUST return `AbilityCheckHelpGiven` for Table D rows marked given: a new `HelpBenefits` with an `AbilityCheckHelp` naming the helper, the ally and the chosen `AssistedProficiency`, and a `HelpGiven` event; and `AbilityCheckHelpRefused` carrying an `AbilityCheckHelpRefusal` (`HelperLacksProficiency`, `CannotHelpSelf`, `AssistanceNotPossible`) for rows marked refused, with the `HelpBenefits` unchanged. | SRD 5.2.1, "Help [Action]", "Assist an Ability Check", p. 182 | per row: happy / unhappy (boundary) |
| FR-036 | The chosen skill or tool MUST be one the helper is proficient in; otherwise `HelperLacksProficiency`. A tool and a skill MUST NOT match each other, by type. | SRD 5.2.1, "Help [Action]", "Assist an Ability Check", p. 182 ("Choose one of your skill or tool proficiencies") | unhappy (invalid input) |
| FR-037 | The Help action's ally MUST be another creature: a helper naming itself MUST be refused with `CannotHelpSelf`. | SRD 5.2.1, "Actions", Actions table, p. 10 ("Help another creature's ability check or attack roll"); "Help [Action]", p. 182 ("one ally"); "Ally", p. 176 | unhappy (invalid input) |
| FR-038 | When more than one refusal applies to ability-check help, the reason MUST be the first of `CannotHelpSelf`, `HelperLacksProficiency`, `AssistanceNotPossible` (D07, D08). | package design (the SRD gives no order); architecture.md section 3 (determinism) | unhappy (rule interaction) |
| FR-039 | An `AssistanceJudgement` of `NotPossible` MUST refuse with `AssistanceNotPossible` and leave the `HelpBenefits` unchanged. | SRD 5.2.1, "Help [Action]", "Assist an Ability Check", p. 182 ("The GM has final say on whether your assistance is possible") | unhappy (state error) |
| FR-040 | `GiveAttackRollHelp` MUST return `AttackRollHelpGiven` for `EnemyWithinFiveFeet`: a new `HelpBenefits` with an `AttackRollHelp` naming the helper and the enemy, and a `HelpGiven` event; and `AttackRollHelpRefused` carrying an `AttackRollHelpRefusal`: `EnemyNotWithinFiveFeet` for `EnemyBeyondFiveFeet` and `TargetNotAnEnemy` for `NotAnEnemy`, with the `HelpBenefits` unchanged (precedence and the helper-is-the-enemy case: FR-075). | SRD 5.2.1, "Help [Action]", "Assist an Attack Roll", p. 183 ("an enemy within 5 feet of you"); "Enemy", p. 181 | per row: happy / unhappy (boundary) |
| FR-041 | `GiveAttackRollHelp` MUST NOT need or check the helper's proficiencies (E04), and `GiveAbilityCheckHelp` MUST NOT need an enemy. | SRD 5.2.1, "Help [Action]", pp. 182-183 | unhappy (empty case) |
| FR-042 | Giving Help MUST return, in the `...Given` variant, a new `HelpBenefits` that holds the added benefit after the existing ones; the one it was called on MUST be unchanged, and its `Benefits()` view MUST be returned as a copy. A refusal MUST carry a `HelpBenefits` equal by value to the one it was called on (the same instance is permitted). A new `HelpBenefits` MUST start empty. | architecture.md section 3 | unhappy (immutability) |
| FR-065 | `AssistedSkill`, `AssistedTool`, `AbilityCheckHelp`, `AttackRollHelp`, `HelpBenefits` and `HelpGiverProficiencies` MUST have value equality (comparable with `==`, or an `Equal` method where the type holds a slice), because FR-044 and FR-048 match a benefit to a roll by it. `HelpBenefits.benefits` and the skill and tool views of `HelpGiverProficiencies` MUST be copied (mutating the returned slice leaves the value unchanged); `HelpGiverProficiencies` MUST hold each skill and tool once, so a duplicate entry collapses to one. | architecture.md section 3 | unhappy (immutability) |
| FR-074 | `NewAbilityCheckHelp` MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) when the helper equals the ally, and `NewAttackRollHelp` MUST return a validation error (`errors.Is` `srderr.ErrInvalidInput`) when the helper equals the enemy, so that no benefit that can never be used, and no `HelpGiven` or `HelpBenefitUsed` event with a duplicate combatant (`combat-foundations.md` FR-018; FR-068 here), can be built directly. `ConsumeForAttackRoll` MUST treat an attacker that is the target as no benefit: it MUST return `NoAttackRollHelp` with the `HelpBenefits` unchanged (G09), because a creature is not its own enemy. | architecture.md section 3 (invariants); combat-foundations.md FR-018; SRD 5.2.1, "Help [Action]", pp. 182-183; "Enemy", p. 181 | unhappy (invalid input) |
| FR-075 | `GiveAttackRollHelp` MUST refuse with `TargetNotAnEnemy`, before any other check, when the named enemy is the helper itself (E05, E06, E07). When more than one refusal applies the reason MUST be the first of `TargetNotAnEnemy`, `EnemyNotWithinFiveFeet`. | package design (the SRD gives no order); SRD 5.2.1, "Help [Action]", "Assist an Attack Roll", p. 183; "Enemy", p. 181; architecture.md section 3 (determinism) | unhappy (rule interaction) |

### 4.7 Using the benefit

Table F. Each row is one table-driven test of
`HelpBenefits.ConsumeForAbilityCheck(checker, AssistedProficiency)`. Pending
benefits `[helper→ally, proficiency]`; `Ath` is `AssistedSkill` Athletics, `Ste`
Stealth, `Thv` an `AssistedTool`.

| Row | Pending | Check made | Result | Benefits left | Path |
|---|---|---|---|---|---|
| F01 | [H→Y, Ath] | Y, Ath | Advantage | none | happy |
| F02 | [H→Y, Thv] | Y, Thv | Advantage | none | happy |
| F03 | [H→Y, Ath] | Y, Ste | none | [H→Y, Ath] | unhappy (rule interaction) |
| F04 | [H→Y, Ath] | Z, Ath | none | [H→Y, Ath] | unhappy (rule interaction) |
| F05 | [H→Y, Ath] after F01 | Y, Ath again | none | none | unhappy (state error) |
| F06 | [H→Y, Ath], [K→Y, Ath] | Y, Ath | one Advantage | none | unhappy (rule interaction) |
| F07 | [H→Y, Ath], [K→Y, Ste] | Y, Ath | Advantage | [K→Y, Ste] | unhappy (rule interaction) |
| F08 | none | Y, Ath | none | none | unhappy (empty case) |
| F09 | [H→Y, Ath] | Y, Thv (a tool, not the skill) | none | [H→Y, Ath] | unhappy (rule interaction) |

Table F2. Each row is one table-driven test of
`ConsumeForAbilityCheckUsing(checker, []AssistedProficiency)` (FR-073), for a
check that uses a skill and a tool together (SRD 5.2.1, "Equipment
Proficiencies", p. 9).

| Row | Pending | Check made | Result | Benefits left | Path |
|---|---|---|---|---|---|
| F10 | [H→Y, Ath], [K→Y, Thv] | Y, {Ath, Thv} | one Advantage, two `HelpBenefitUsed` (in the order given) | none | happy |
| F11 | [H→Y, Ath], [K→Y, Thv] | Y, {Ath} (skill only) | one Advantage, one `HelpBenefitUsed` | [K→Y, Thv] | unhappy (rule interaction) |
| F12 | [H→Y, Ath], [K→Y, Thv] | Y, {Thv} (tool only) | one Advantage, one `HelpBenefitUsed` | [H→Y, Ath] | unhappy (rule interaction) |
| F13 | [H→Y, Ath], [K→Y, Ste] | Y, {Ath, Thv} | one Advantage | [K→Y, Ste] | unhappy (rule interaction) |
| F14 | [H→Y, Ath] | Y, {} (a check that uses no proficiency) | none | [H→Y, Ath] | unhappy (empty case) |
| F15 | [H→Y, Ath], [K→Z, Thv] | Y, {Ath, Thv} | one Advantage (Z's benefit is for another ally) | [K→Z, Thv] | unhappy (rule interaction) |
| F16 | [H→Y, Ath] | Y, {Ath} | the same as F01 (the one-proficiency form equals the set `{Ath}`) | none | happy |

Table G. Each row is one table-driven test of
`ConsumeForAttackRoll(attacker, target, alliedHelpers)`; `AlliedHelpers` is the
set of helpers the host says are allies of the attacker. Pending `[helper→enemy]`.

| Row | Pending | Attacker, target, `AlliedHelpers` | Result | Benefits left | Path |
|---|---|---|---|---|---|
| G01 | [H→E] | A, E, {H} | Advantage | none | happy |
| G02 | [H→E] | A, F (another enemy), {H} | none | [H→E] | unhappy (rule interaction) |
| G03 | [H→E] | A, E, {} (the host says A is not H's ally) | none | [H→E] | unhappy (rule interaction) |
| G04 | [H→E] | H, E, {H} (the helper's own attack) | none | [H→E] | unhappy (rule interaction) |
| G05 | [H→E] after G01 | A, E again, {H} | none | none | unhappy (state error) |
| G06 | [H→E], [K→E] | A, E, {H, K} | one Advantage | none | unhappy (rule interaction) |
| G07 | [H→E], [K→E] | A, E, {H} | Advantage | [K→E] | unhappy (rule interaction) |
| G08 | none | A, E, {H} | none | none | unhappy (empty case) |
| G09 | [H→E] | E, E, {H} (the attacker is the target, FR-074) | none | [H→E] | unhappy (invalid input) |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-043 | `ConsumeForAbilityCheck` and `ConsumeForAbilityCheckUsing` (FR-073) MUST return `AbilityCheckHelpUsed` for Table F and F2 rows with a result: a new `HelpBenefits` without the consumed benefits, the `[]modifier.RollCircumstance` (a copy) containing one `Advantage` (FR-076), and one `HelpBenefitUsed` event for each consumed benefit; and `NoAbilityCheckHelp` with the `HelpBenefits` unchanged otherwise. | SRD 5.2.1, "Help [Action]", "Assist an Ability Check", p. 182 | per row: happy / unhappy (boundary) |
| FR-044 | For a check that uses one proficiency (`ConsumeForAbilityCheck(checker, AssistedProficiency)`), a benefit MUST apply only to the named ally, and only to an ability check that uses the same skill or the same tool as the benefit. A check with another skill or tool MUST NOT use it up (F03, F09). A check that uses a skill and a tool together is FR-073. | SRD 5.2.1, "Help [Action]", "Assist an Ability Check", p. 182 ("the next ability check they make with the chosen skill or tool") | unhappy (rule interaction) |
| FR-045 | A benefit MUST be used by one check only: a later matching check MUST get no Advantage from it (F05). | SRD 5.2.1, "Help [Action]", pp. 182-183 ("the next") | unhappy (state error) |
| FR-046 | Several pending benefits that match the same check MUST all be consumed by it, and the result MUST hold exactly one `Advantage` (F06). | SRD 5.2.1, "They Don't Stack", p. 8; "Help [Action]", p. 182 (Q-04) | unhappy (rule interaction) |
| FR-047 | `ConsumeForAttackRoll` MUST return `AttackRollHelpUsed` for Table G rows with a result, shaped as in FR-043, and `NoAttackRollHelp` with the `HelpBenefits` unchanged otherwise. | SRD 5.2.1, "Help [Action]", "Assist an Attack Roll", p. 183 | per row: happy / unhappy (boundary) |
| FR-048 | A benefit MUST apply only to the next attack roll against the enemy it names (G02), and only to an attacker in `AlliedHelpers` for that benefit's helper (G03). | SRD 5.2.1, "Help [Action]", "Assist an Attack Roll", p. 183 ("the next attack roll by one of your allies against that enemy"); "Ally", p. 176; "Enemy", p. 181 | unhappy (rule interaction) |
| FR-049 | The helper's own attack MUST NOT use its own `AttackRollHelp` (G04), because the benefit is for "one of your allies". | SRD 5.2.1, "Help [Action]", "Assist an Attack Roll", p. 183; "Ally", p. 176 (Q-05) | unhappy (rule interaction) |
| FR-050 | A benefit MUST be used up by the attack roll that it applies to, whether or not that roll ends with Advantage, Disadvantage or neither once the caller resolves the roll mode. `HelpBenefits` MUST return the `Advantage` as a circumstance only and MUST NOT resolve the roll mode (the kernel's rule: `modifiers-and-advantage.md` FR-013 to FR-015). | SRD 5.2.1, "They Don't Stack", p. 8; "Help [Action]", p. 183 (Q-04) | unhappy (rule interaction) |
| FR-052 | Using a benefit MUST NOT change any other benefit and MUST NOT mutate the `HelpBenefits` it was called on. | architecture.md section 3 | unhappy (immutability) |
| FR-073 | `HelpBenefits.ConsumeForAbilityCheckUsing(checker, proficiencies []AssistedProficiency)` MUST take every proficiency the one ability check uses (a skill, a tool, or both: the Tools entry says "you can benefit from both skill proficiency and tool proficiency on the same ability check"). A pending benefit MUST match when it names the checker as ally and its `AssistedProficiency` is in the slice (read as a set; duplicates are harmless); every matching benefit MUST be consumed, the result MUST hold exactly one `Advantage` and one `HelpBenefitUsed` per consumed benefit, in the order the benefits were given, and no match gives `NoAbilityCheckHelp` with the `HelpBenefits` unchanged (F10 to F15). An empty or nil slice matches nothing (F14). `ConsumeForAbilityCheck(checker, proficiency)` MUST give the same result as `ConsumeForAbilityCheckUsing(checker, []AssistedProficiency{proficiency})` (F16). | SRD 5.2.1, "Equipment Proficiencies", Tools, p. 9; "Help [Action]", "Assist an Ability Check", p. 182; "They Don't Stack", p. 8 | unhappy (rule interaction) |

### 4.8 Expiry and dropping

Table H. Each row is one table-driven test of `ExpireAtStartOfTurn(combatant)`.

| Row | Pending | Turn that starts | Events | Benefits left | Path |
|---|---|---|---|---|---|
| H01 | [H→Y, Ath], [H→E] | H | two `HelpBenefitExpired` | none | happy |
| H02 | [H→Y, Ath], [K→E] | H | one, for H's benefit | [K→E] | unhappy (rule interaction) |
| H03 | [H→Y, Ath] | Y (the ally's own turn) | none | [H→Y, Ath] | unhappy (boundary) |
| H04 | [H→Y, Ath] | K (another combatant) | none | [H→Y, Ath] | unhappy (boundary) |
| H05 | none | H | none | none | unhappy (empty case) |
| H06 | [H→Y, Ath] used by Y earlier | H | none | none | unhappy (state error) |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-053 | `ExpireAtStartOfTurn(combatant)` MUST remove every benefit whose helper is that combatant and return a `HelpBenefitsExpiry` (FR-072) holding one `HelpBenefitExpired` event for each, in the order the benefits were given, and MUST keep every other benefit (H01, H02). | SRD 5.2.1, "Help [Action]", pp. 182-183 ("expires if the ally doesn't use it before the start of your next turn"; "expires at the start of your next turn") | happy |
| FR-054 | A start of any other combatant's turn, including the ally's own, MUST expire nothing (H03, H04). | SRD 5.2.1, "Help [Action]", pp. 182-183 | unhappy (boundary) |
| FR-055 | A benefit used before the helper's next turn MUST NOT produce an expiry event later (H06), and expiring with nothing pending MUST return a `HelpBenefitsExpiry` whose `benefits` is equal by value to the one called on and whose `events` is empty (H05). | SRD 5.2.1, "Help [Action]", pp. 182-183 | unhappy (state error) |
| FR-056 | `ExpireAtStartOfTurn` MUST always return a `HelpBenefitsExpiry`; its `benefits` MUST be a new `HelpBenefits` when a benefit was removed and equal by value otherwise (the same instance is permitted), and the one it was called on MUST be left unchanged. | architecture.md section 3 | unhappy (immutability) |

Table H2. Each row is one table-driven test of `DropFor(combatant)` or
`DropAll()`. A benefit is dropped for a combatant that is its helper, ally or
enemy. "none" in Events is an empty slice (returned as a copy).

| Row | Pending | Call | Events | Benefits left | Path |
|---|---|---|---|---|---|
| H07 | [H→Y, Ath], [K→E] | `DropFor(E)` | one `HelpBenefitDropped` (K→E) | [H→Y, Ath] | happy |
| H08 | [H→Y, Ath], [K→E] | `DropFor(H)` | one (H→Y) | [K→E] | unhappy (rule interaction) |
| H09 | [H→Y, Ath], [K→E] | `DropAll()` | two, in the order given | none | happy |
| H10 | none | `DropAll()` | none | none | unhappy (empty case) |
| H11 | [H→Y, Ath] | `DropFor(Z)` (Z named by no benefit) | none | [H→Y, Ath] | unhappy (boundary) |
| H12 | [H→Y, Ath], [H→E] | `DropFor(Y)` | one (H→Y) | [H→E] | unhappy (rule interaction) |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-069 | `DropFor(combatant)` MUST remove every benefit that names the combatant as helper, ally or enemy, and `DropAll()` MUST remove every benefit; each MUST always return a `HelpBenefitsDrop` (FR-072) holding one `HelpBenefitDropped` event per removed benefit in the order the benefits were given, and a `HelpBenefits` that is new when a benefit was removed and equal by value to the one called on, with an empty `events` list, when nothing is removed (H10, H11). The host calls `DropFor` when a combatant leaves the encounter and `DropAll` when the encounter or the exploration ends; the choice of when stays the host's (Q-14). | SRD 5.2.1, "Help [Action]", pp. 182-183 (the benefit only lasts to "the next" roll or "your next turn"); package design (Q-14) | per row: happy / unhappy (boundary) |

### 4.9 Package rules

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-057 | Every type in this spec MUST live in package `combat` (domain files, none named `*_usecase.go`), be immutable, synchronous and deterministic, and import no adapter, `*test` package, `sync` or `time` call, or another context's sub-package. It MUST import only the `content` package (`Skill`, `ItemID`, `FeatureID`), the kernel (`modifier.RollCircumstance`) and, inside `combat`, only the types of `combat-foundations.md` (`CombatantID`, `Event`), so no import points to the right in the order of shared-types FR-015. This spec has no slot in that order (Q-06), so an architecture check MUST also assert that no file of package `combat` other than the files of `combat-action-resolution.md` mentions this spec's types. | architecture.md sections 1, 2 and 9; shared-types-and-ownership.md FR-013, FR-015; combat-foundations.md FR-026 | unhappy (architecture) |
| FR-058 | This spec MUST define no use case, command or port, and no type of it MUST import or call a use case of `attacks.md`, `initiative-and-actions.md` or any other spec, or `ResolveD20Test`; the caller wires them. | shared-types-and-ownership.md FR-026, FR-044; architecture.md sections 2 and 4 | unhappy (architecture) |
| FR-059 | Every refusal MUST be a closed outcome variant carrying a reason enum, never an `error` or a panic. Only a malformed value (FR-010, FR-019, FR-064) returns a validation error from its `NewX` constructor. The variant names (`AttackActionAttackRecorded`, `AttackActionAttackRefused`, `AttackActionWeaponChangeRecorded`, `AttackActionWeaponChangeRefused`, `AbilityCheckHelpGiven`, `AbilityCheckHelpRefused`, `AttackRollHelpGiven`, `AttackRollHelpRefused`, `AbilityCheckHelpUsed`, `NoAbilityCheckHelp`, `AttackRollHelpUsed`, `NoAttackRollHelp`), the closed groups (FR-071) and the result values `HelpBenefitsExpiry` and `HelpBenefitsDrop` (FR-072; in no group) MUST be unique among the exported members of package `combat`, and no generic name (`Applied`, `Rejected`, `Found`) MUST be exported. `AttackActionWeaponChange` and `WeaponWielding` MUST be two distinct exported names. These are outcomes of domain operations, so the `<UseCase>Result` name of shared-types FR-017 does not apply (FR-070). | architecture.md sections 4 and 6; shared-types-and-ownership.md FR-017, FR-021 | unhappy (architecture) |
| FR-060 | `AttackMadeInAttackAction`, `AttackReplacedInAttackAction`, `WeaponChangedInAttackAction`, `HelpGiven`, `HelpBenefitUsed`, `HelpBenefitExpired` and `HelpBenefitDropped` MUST be past-tense immutable value objects, each defined here once. The first three MUST belong to the closed sub-family `AttackActionEvent` and the last four to the closed sub-family `HelpEvent`, and both sub-families MUST embed `combat.Event` (`combat-foundations.md` FR-015, FR-016); no event implements `combat.Event` directly. | combat-foundations.md FR-015, FR-016, FR-020; shared-types-and-ownership.md FR-019; architecture.md section 3 | unhappy (architecture) |
| FR-061 | Nothing here MUST produce display text, a random number, a clock read or a log; all results are structured data, and the same call sequence from the same `start` or the same `HelpBenefits` gives the same result on every run. | architecture.md sections 3 and 6; observability.md section 3 | unhappy (determinism) |
| FR-067 | `AttackActionProgress` and `HelpBenefits` MUST NOT import or reference `WeaponTurnState`, `WeaponWielding` or any other type of `weapon-properties-and-mastery.md`, and that spec's types MUST NOT be needed to build them. The host records an attack in `AttackActionProgress` and, separately, in `WeaponTurnState`; the two cannot disagree about the allowance because only this one holds it. | shared-types-and-ownership.md FR-026; architecture.md section 2 | unhappy (architecture) |
| FR-068 | Each event MUST have a constant `kind` matching `[a-z][a-z0-9_]*`, unique in the package, and a `Combatants()` list (a copy on each read) that has no duplicates and is in the order given here, and each MUST be equal by value including `kind` and `combatants`: `AttackMadeInAttackAction` `attack_made_in_attack_action`, `AttackReplacedInAttackAction` `attack_replaced_in_attack_action` and `WeaponChangedInAttackAction` `weapon_changed_in_attack_action`, each with the combatant alone; `HelpGiven` `help_given`, `HelpBenefitExpired` `help_benefit_expired` and `HelpBenefitDropped` `help_benefit_dropped`, each with the helper, then the ally or the enemy; `HelpBenefitUsed` `help_benefit_used` with the helper, then the ally who made the check, or for an attack roll the helper, then the attacker, then the enemy. | combat-foundations.md FR-017 to FR-019 | unhappy (name collision) |
| FR-070 | The refusal reasons MUST be four named enums exported from package `combat`: `AttackRefusal` (`AttackLimitReached`), `WeaponChangeRefusal` (`NoAttackMade`, `NoAttackRemaining`, `WeaponChangeAlreadyUsedForThisAttack`), `AbilityCheckHelpRefusal` (`CannotHelpSelf`, `HelperLacksProficiency`, `AssistanceNotPossible`) and `AttackRollHelpRefusal` (`EnemyNotWithinFiveFeet`, `TargetNotAnEnemy`). These are outcomes of domain value-object operations, not use case results, so `combat-foundations.md` FR-022 and FR-023 (`<Operation>Rejected` holding a `<Subject>Rejection`) govern the wrapping use cases (the `...Rejected` variants of `combat-action-resolution.md` carry a `LedgerRejection` holding a `LedgerRefusal`), not these types. | combat-foundations.md FR-022, FR-023; architecture.md section 4 | unhappy (architecture) |
| FR-071 | The closed outcome types that group the variants of FR-059 MUST be named `AttackRecordOutcome` (`AttackActionAttackRecorded`, `AttackActionAttackRefused`), `WeaponChangeOutcome` (`AttackActionWeaponChangeRecorded`, `AttackActionWeaponChangeRefused`), `AbilityCheckHelpOutcome` (`AbilityCheckHelpGiven`, `AbilityCheckHelpRefused`), `AttackRollHelpOutcome` (`AttackRollHelpGiven`, `AttackRollHelpRefused`), `AbilityCheckHelpUseOutcome` (`AbilityCheckHelpUsed`, `NoAbilityCheckHelp`) and `AttackRollHelpUseOutcome` (`AttackRollHelpUsed`, `NoAttackRollHelp`), so a host switches exhaustively on each. `WeaponChangeKind` (`equip`, `unequip`) is the enum of FR-029. All are exported from package `combat` and are unique package-wide. | shared-types-and-ownership.md FR-017, FR-019, FR-021; architecture.md sections 3 and 4 | unhappy (name collision) |
| FR-072 | `ExpireAtStartOfTurn` MUST return a `HelpBenefitsExpiry{Benefits, Events}` and `DropFor` and `DropAll` MUST return a `HelpBenefitsDrop{Benefits, Events}`. Both are immutable value structs exported from package `combat`, equal by value (`Equal` method, since they hold slices), with `Benefits` a `HelpBenefits` and `Events` a `[]HelpEvent` returned as a copy (FR-076) holding only `HelpBenefitExpired` (in `HelpBenefitsExpiry`) or only `HelpBenefitDropped` (in `HelpBenefitsDrop`) in the order the benefits were given. The result is always a record, also when nothing is removed (then `events` is empty and `benefits` is equal by value to the instance called on), so a caller never branches on a nil value; this keeps FR-055, FR-056 and FR-069 consistent. They are result values, not outcome variants and not use case results, so they sit in no FR-071 group. | combat-foundations.md FR-021; architecture.md sections 3 and 4; `combat-action-resolution.md` FR-118, FR-123 call these operations | unhappy (architecture) |
| FR-076 | Every `events` list of an outcome variant or result value of this spec (`AttackActionAttackRecorded`, `AttackActionWeaponChangeRecorded`, `AbilityCheckHelpGiven`, `AttackRollHelpGiven`, `AbilityCheckHelpUsed`, `AttackRollHelpUsed`, `HelpBenefitsExpiry`, `HelpBenefitsDrop`; an empty one for a refusal or a `No...Help` variant) and the `circumstances` list of `AbilityCheckHelpUsed` and `AttackRollHelpUsed` (FR-043, FR-047) MUST be returned as a copy (mutating the returned slice leaves the value unchanged). An `events` list MUST be in the order the events happened (for a consume, the order the benefits were given); a recorded or given variant holds exactly one event. | combat-foundations.md FR-021; architecture.md section 4 ("MUST NOT expose internal collections for mutation") | unhappy (immutability) |
| FR-077 | `AttackActionProgress` MUST expose the combatant it was started for as the public getter `attacker` (`CombatantID`, the first argument of `start`, FR-014), included in equality (FR-021). A host holding one progress per ledger can then check, before `RecordAttack`, that the progress is that attacker's; the roster removal of `combatant-roster-changes.md` FR-025 leaves it untouched because the leaver is not the active combatant. | `combat-action-resolution.md` FR-113; `combatant-roster-changes.md` FR-025; architecture.md section 3 | happy |

### 4.10 Withdrawn ids

Ids stay unused. Table A rows A03, A04, A07 to A09, A11 to A14 are withdrawn too.

| Id | Status |
|---|---|
| FR-003 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-054) |
| FR-004 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-054) |
| FR-005 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-055) |
| FR-006 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-057) |
| FR-007 | Withdrawn: moved to class-warlock.md (FR-079) |
| FR-008 | Withdrawn: moved to class-warlock.md (FR-155) |
| FR-009 | Withdrawn: dropped, no owner needed (class-warlock.md FR-079) |
| FR-011 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-058) |
| FR-012 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-058) |
| FR-013 | Withdrawn: moved to class-fighter.md (FR-026) |
| FR-022 | Withdrawn: moved to attack-and-help-actions.md (FR-061) |
| FR-051 | Withdrawn: moved to modifiers-and-advantage.md (FR-013) |

## 5. Model

- **Value objects:** `AttackMeans`, `AttackReplacement` (`AttackReplacementKind`,
  `AttackReplacementSource`), `AttackAllowance`, `AttackActionWeaponChange`,
  `AttackActionWeaponChangeTiming`, `WeaponChangeKind`, `AssistedProficiency`,
  `HelpGiverProficiencies`, `AssistanceJudgement`, `AttackAssistTarget`,
  `HelpBenefit` (closed: `AbilityCheckHelp`, `AttackRollHelp`).
- **State holders (immutable values):** `AttackActionProgress` and
  `HelpBenefits`. The host keeps them in the `ActionLedger` of
  `combat-action-resolution.md` (FR-112, Q-03).
- **Outcomes and events:** the six closed variant groups of FR-071, the results
  `HelpBenefitsExpiry` and `HelpBenefitsDrop` (FR-072), the closed
  `AttackActionEvent` and `HelpEvent` (FR-060, FR-068) and the four refusal enums (FR-070).
- **Operations (pure):** the operations named in section 4 and the queries
  `AttacksRemaining` and `IsBetweenAttacks`.
- **Invariants:** `NewAttackAllowance` (FR-010); `NewAttackActionProgress` (FR-019, FR-064); `NewAbilityCheckHelp` and `NewAttackRollHelp` (FR-074); equality and copied views (FR-065, FR-076). Value objects have unexported fields, value receivers and no setters; the zero value of each is invalid and documented as such.
- **Failures:** expected refusals are closed outcome variants (an interface with an unexported marker method plus one struct type per variant, listed in the interface's doc comment and switched exhaustively); malformed values are validation errors from `NewX` wrapping `srderr.ErrInvalidInput`, and
  `MustX` helpers panic with that error. All input arrives typed from the host, so there is no untrusted-input parser or decoder path.
- **Worked example.** Fighter 5 / Barbarian 5: the host combines the counts [2, 2] with `ExtraAttackCombination`, builds `AttackAllowance(2, 2)` (A02), starts a progress, draws a weapon before attack 1 (C13), attacks, is between attacks, attacks again and the allowance is spent (B04).

## 6. Interface

- Domain only: no use case, port, command or `<UseCase>Result`. Operations are synchronous pure methods on the value objects returning the closed outcomes of FR-059 (state in, new state and events out). Composition (`ActionLedger`,
  `PerformAttack`, `TakeCombatantAction`, `AdvanceCombatTurn`; its FR-117 consumes, FR-118 and FR-123 expire and drop) is `combat-action-resolution.md` (section 1). Exports: section 1 (package `combat`). Pure operations take no
  `context.Context` (go.md §4).
- No test doubles, contract suites or bundled content (shared-types FR-030 to FR-035, FR-038, FR-039 do not apply).

## 7. Observability

Silent by design (observability.md section 3): domain code with no use case,
port or signal, and no entry in `doc/observability/signals.md`. The wrapping
use cases run through `combat-action-resolution.md` (FR-099
to FR-106), which take the `telemetry` ports through options, emit the shared `srd5e.combat.*` signals and count every returned
event in `srd5e.combat.events`; each records only its own signals (shared-types
FR-044). The seven questions (observability.md section 2):

1. **What can go wrong?** An attack past the limit, a second weapon change for
   one attack, a Help the helper cannot give: typed refusals; a malformed value
   (FR-010, FR-019, FR-064) is a returned validation error.
2. **How would an operator know?** The reason enum is carried in a
   `LedgerRejection` and classified `refused` / `ledger_refused` by
   `combat-action-resolution.md` FR-100 (shared-types FR-041,
   `combat-foundations.md` FR-030). A refusal is a normal outcome: no error
   status, `error.type` or log.
3. **Diagnosis?** The outcome carries reason, allowance, history and slots; events
   name the combatant, means or benefit.
4. to 7. **Counted or timed? Existing instrumentation? Monitor? Noisy?** Nothing
   is emitted here and no signal is added or changed (`ActionTaken` is
   `initiative-and-actions.md` FR-103); the wrapping counters count Attack
   actions, weapon changes and Help benefits; a rise in `ledger_refused` points
   to a host sending impossible requests.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none | | | | |

## 8. Test plan

Doubles and contract suites: none. Unit tests: `test/combat/domain/`; architecture
tests: `test/architecture/`.

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002 | unit | Exhaustive `switch` over `AttackMeans` compiles with three cases; `baseline` is (1,1), A01 |
| FR-010 | unit | A01, A02, A05, A06, A10, A15, A18 construct; A16 (general 0) and A17 (pact below general) return a validation error; boundary at general 1 and pact equal to general |
| FR-014 | unit | `start` with each Table A allowance (A18 included); `AttacksRemaining` equals `PactWeaponAttacks` |
| FR-015 | unit | Table B, one test per row with a means, asserting the event's means and number |
| FR-016 | unit | B06, B08, B09, B11, B22 to B26, B28 to B35: B29 and B30 separate `PactWeaponAttacks` 3 from 2, B32, B34 and B35 separate `GeneralAttacks` 2 from `GeneralAttacks` minus 1 (a mutant using 1 refuses B32 and B34) |
| FR-017, FR-018 | unit | B03 and an unarmed-first sequence; `AttacksRemaining` after each step of B06, B08, B12, B16, B22; for (2,3): 3 at the start, 2 after P, 1 after P,P, 0 after P,P,P, 0 after P,O, 1 after O; empty history gives `PactWeaponAttacks`; never below zero |
| FR-019, FR-064 | unit | A history longer than the limit returns an error, exactly at the limit builds; (1,3) with [R, R, P] returns an error and [R, P, P] builds; slot 0, a duplicate slot, slot made + 2 return an error; slot 1 with one attack and slot made + 1 with `AttacksRemaining` 1 build; slot made + 1 with `AttacksRemaining` 0 returns an error |
| FR-020, FR-021, FR-077 | unit | `attacker` returns the id passed to `start` and is unchanged by `RecordAttack`; Original progress equals its earlier value after `RecordAttack`; result differs after a recorded attack and equals by value after a refusal (no identity assertion); mutating the returned slice leaves the value unchanged; equal progresses compare equal, ones differing in one slot (`attacker` included) differ |
| FR-023 | unit | Two progresses for the same combatant are independent; a second `start` has 0 attacks and no slot used |
| FR-024, FR-031 | architecture | Public member names of `AttackActionProgress` contain neither `Opportunity` nor `Reaction`, and no member takes or returns a type of `initiative-and-actions.md`; the tracker scenario is `combat-action-resolution.md` FR-118 |
| FR-025 to FR-030 | unit | Table C, one test per row, asserting event change, `change`, timing and slot; FR-026 C05, C14, C16; FR-027 C03; FR-028 C06, C07, C08, C10, C12 (each refusal reachable alone); FR-029 one assertion per value; FR-030 after `DrawFromSheath` before attack 1, `RecordAttack` accepts `UnarmedStrike` and `OtherWeapon` |
| FR-032 to FR-034 | unit | (2,2): false at 0 attacks, true at 1, false at 2; (1,1) false at 0 and 1; (1,2) after an other-weapon attack false; repeated queries leave the progress equal, and the next `RecordAttack` and `HandleWeapon` equal a control run (tracker half: `combat-action-resolution.md` FR-118) |
| FR-035 to FR-042 | unit | Table D and E, one test per row; D03, D04, D09 and an `AssistedTool` not matching an `AssistedSkill` of the same name; D06; D07, D08; D05; E04 and ability-check help with no enemy argument; original `HelpBenefits` unchanged after giving, mutating the returned slice leaves the value unchanged, a refusal carries an equal-by-value `HelpBenefits`, a new one is empty |
| FR-043 to FR-052, FR-073 | unit | Tables F, F2, G, one test per row, asserting the circumstance list and one `HelpBenefitUsed` per consumed benefit; F03, F04, F09; F05; F06, F07 (circumstances length 1); G02, G03; G04; FR-050: a benefit consumed when the caller also holds a `Disadvantage` is gone and the circumstances hold only `Advantage`; unrelated benefits and the input instance unchanged; F10 asserts two events in the order given; F16 equals F01 |
| FR-053 to FR-056, FR-069 | unit | Tables H and H2, one test per row, with event order, asserting a `HelpBenefitsExpiry` or `HelpBenefitsDrop`; H05, H10, H11 assert empty events and `benefits` equal by value (not identical); input instance unchanged; `benefits` differs after H01 and equals after H03 |
| FR-057, FR-058, FR-067 | architecture | Import scan against architecture.md section 9 (checks A1, A2, A4): the only `combat` types used are `combat-foundations`'s, no `*_usecase.go` file, no use case or `weapon-properties-and-mastery.md` type mentioned, no command, use case or port declared; a second scan fails on any file of package `combat` outside `combat-action-resolution.md`'s that mentions this spec's types |
| FR-059, FR-070, FR-071 | architecture, unit | Package API scan for duplicate and generic names (`HelpBenefitsExpiry`, `HelpBenefitsDrop` included), `AttackActionWeaponChange` distinct from `WeaponWielding`, the four refusal enums with their values, the six closed groups and `WeaponChangeKind` each once; an exhaustive type `switch` over each group lists both variants and no default (the `gochecksumtype` lint passes); every FR-059 variant is in exactly one group; a refusal never returns an error or panics |
| FR-060, FR-068 | architecture, unit | Only `AttackActionEvent` and `HelpEvent` embed `combat.Event` directly, both closed, each event in exactly one; each event's `Kind()`, `Combatants()` order, equality and copied slice; kind format and uniqueness across the package |
| FR-061 | architecture, unit | Scan for `math/rand`, `time.Now(`, `fmt.Print*`; the same call sequence run twice gives equal results |
| FR-062, FR-063 | unit, architecture | B19 to B27 assert the event's kind, source and number; `AttacksRemaining`, `IsBetweenAttacks` and slots are equal before and after a Nick extra attack is recorded in `WeaponTurnState`; no operation for it on the progress |
| FR-065, FR-074, FR-075 | unit | Equal and unequal pairs for each type, mutating a returned slice leaves the value unchanged, `HelpGiverProficiencies` with a duplicate skill holds it once; `AbilityCheckHelp` with helper equal to ally and `AttackRollHelp` with helper equal to enemy return an error, distinct combatants build; G09; no event holds a duplicate in `combatants`; E05, E06, E07 (`TargetNotAnEnemy` wins) |
| FR-066 | architecture | The package exports no source, scope or combine type for attack counts; Table A rows pass through the constructor only |
| FR-072, FR-076 | unit, architecture | H01 to H12 return the right result type including H05, H10, H11; equality; mutating the returned `Events()` slice leaves every named variant unchanged, a refusal has an empty `Events()`; event order for F10 and H09; both result names appear once and in no FR-071 group |

- Mutation testing: `gomu` on the files of this spec in package `combat`, threshold 80,
  run only inside Docker with `--cpus` and `--workers` capped at a quarter of the
  host's processors (testing.md section 8; never as a bare host process). Likely survivors: the limit logic (FR-016), slot arithmetic
  (FR-025 to FR-028, FR-064) and match logic (FR-044, FR-048, FR-073); the
  tables kill them (B02, B07, B17, B29 to B35, C06 to C14, F10 to F15).
- Composition with `initiative-and-actions.md` and `attacks.md` is tested in
  `combat-action-resolution.md` section 4.18 (FR-113 to FR-118).

## 9. Open questions and decisions

Settled, one clause per question (shared-types FR-062); none escalated:

- Q-01: counts are owned by `level-advancement-and-multiclassing.md` and
  `class-warlock.md` (FR-066). Q-03: ledger state is `combat-action-resolution.md` FR-112. Q-04: one roll spends all matching benefits, even if Disadvantage
  cancels it (FR-046, FR-050). Q-05: the helper is not its own ally (FR-049).
  Q-06: a leaf with no slot in the FR-015 order (FR-057). Q-07: allegiance,
  distance and the GM call are host-typed input. Q-09, Q-10: fewer attacks than
  the allowance are allowed; Unarmed Strike mixes with weapons (FR-017). Q-11:
  first aid is not modelled. Q-12: ownership per shared-types FR-048. Q-13:
  replaced-attack effects stay with `class-warlock.md` FR-091 and
  `class-monk.md` FR-118. Q-15: `HelpGiverProficiencies` (a monster has no
  `SkillProficiencies`); `combat-action-resolution.md` FR-119 must adopt the name.
- Q-02 (interpretation): one weapon change per attack, in that attack's slot, so
  Extra Attack gives more changes; "when you make an attack as part of this
  action" (p. 177) recurs per attack, and published rules commentary agrees
  (FR-025 to FR-028).
- Q-08 (interpretation): the pact-only extra attack needs every attack to be a
  pact weapon attack or a replacement, at least one a pact weapon attack; Thirsting
  Blade is Extra Attack "for your pact weapon only", "attack twice with the
  weapon" (p. 74), so a mixed action gets the general limit (FR-016).
  `class-warlock.md` Q-16 is not decided here.
- Q-14: Help outside an encounter has no tracker, so no turn expiry; the host
  ends benefits with `DropFor` and `DropAll` (FR-069; pure domain, FR-057).
- Q-16 (interpretation): the Nick extra attack has no weapon-change slot; it is
  outside the allowance (p. 90) and a slot numbers an attack counted against it
  (FR-063, FR-067). If a table rules otherwise, add `RecordExtraAttack` here and
  change FR-063 and FR-064.

## 10. Changelog impact

Under `Unreleased`, **Added**: the attack and Help types, outcomes, refusal
enums and events named in sections 3 to 5 and FR-059 to FR-076 (including
`ConsumeForAbilityCheckUsing`, FR-073). Not **BREAKING**: nothing is released yet.
