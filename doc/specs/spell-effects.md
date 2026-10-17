# Spell effects: resolution

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `observability-ports.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `resolve-d20-test-use-case.md`, `content-spells.md`, `spell-effect-data.md`, `spellcasting.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md`, `attacks.md` |

Resolution half of the original spell-effects design (split under shared-types-and-ownership.md FR-059): resolver, use case, plan and telemetry. The effect data model, facts, decoder shape and bundling are `spell-effect-data.md` (`content`). Rank 27 in table 4.11 of shared-types-and-ownership.md, after `attacks` and `initiative-and-actions`, before `encounter-aggregate` (FR-014; [../roadmap.md](../roadmap.md)). Nothing is needed in `character`.

## 1. Purpose and scope

Nothing in the package resolves what a spell does. `content-spells.md` keeps effects as faithful text (its Q5 asks for this spec). `spellcasting.md` stops at the cast (slot, Concentration, a `CastReport` with the spell save DC, spell attack modifiers and scaling levels) and lists targets, saving throws and areas as out of scope. `attacks.md` resolves one spell attack roll and its damage. `hit-points-and-damage.md` and `conditions.md` apply finished damage and conditions. Nothing owns "this spell forces a Dexterity save, deals 8d6 Fire damage, half on a success, plus 1d6 per slot level above 3".

Spell effects are **data** (`spell-effect-data.md`, its section 5.1: data now, no strategy interface per spell). This spec is the interpreter, a **pure domain service and one use case** in `combat`:

- `ResolveSpellEffects` takes the `Spell`, the `CastReport`, the host's chosen targets and each target's save and attack facts. It rolls through the `dice.Roller`, makes every save and attack roll by **composing pure domain services** (the pure rules functions of `d20`: `ResolveCheck`, `ResolveAttack`, `FailWithoutRoll`, and the attack domain functions of attacks.md), never `d20.Resolver` or `ResolveAttack` (FR-096; shared-types-and-ownership.md FR-044), applies scaling, and returns a **plan** (the `Damage`, healing, Temporary Hit Points and conditions each target is due) for `ApplyDamage`, `RestoreHitPoints`, `GainTemporaryHitPoints` and `ApplyConditionToSet`. It changes no state (Q2).
- A spell with no structured data resolves to "text only"; a `partial` one resolves what the model covers and leaves the rest to the GM ("any outcomes beyond those effects are under the GM's purview", p. 106).

**Out of scope:**

- Choosing targets, area membership, line of sight, visibility, range checks: the host supplies the creatures (as in `attacks.md`); shapes are data so the host can compute membership (spell-effect-data.md FR-013; FR-015 here).
- Applying the plan and wiring it into an encounter: the existing use cases and combat-action-resolution.md (Q2).
- Anything that lasts or recurs: durations, Concentration of an effect, repeated saves, riders (Hunter's Mark, Hex), delayed damage (Acid Arrow, Delayed Blast Fireball), and "Combining Spell Effects" (p. 106). They need an active-effects model ([active-effects.md](active-effects.md); [persistent-spell-zones.md](persistent-spell-zones.md) for zones; Q11). Duration is carried only as a fact (spell-effect-data.md FR-046; FR-040 here).
- Casting (`spellcasting.md`), the attack roll (`attacks.md`), the saving throw (`d20-tests.md`), damage handling (`hit-points-and-damage.md`), what a condition does (`conditions.md`).
- Hit Point maximum changes (Aid, Harm) and removing conditions (Heal): see `revival-and-restoration-spells.md`. Wish: `wish-spell.md`. Summoned creatures, terrain, light, objects, illusions, divinations, mode choice (Eyebite, Command): all stay text (table E of spell-effect-data.md). The die-chosen tables of Teleport, Confusion, Prismatic Spray, Prismatic Wall, Reincarnate and Blink are also text here and are rolled by `spell-outcome-tables.md`.
- Spell Scrolls and monster Spellcasting traits (`monster-spellcasting.md`, whose report converts to a `CastReport`; FR-050); "Awareness of Being Targeted" and "Identifying an Ongoing Spell" (p. 106, GM adjudication).
- Rule options: the SRD names none here (`doc/rules/architecture.md` §3).
- Anything outside SRD 5.2.1. The SRD gives no data format for spell effects; the structure is package design. Nothing core was found missing from the PDF.

**Ownership (shared-types-and-ownership.md FR-048).** This spec owns what a spell does once cast: target validity, the rolls, the plan and its telemetry. `spell-effect-data.md` owns the model, scaling and area shapes and facts, and lists what it supersedes (section 1). Nothing in spellcasting.md is superseded; its out-of-scope list is the entry this spec fills. magical-effects.md reads `AreaShape` from spell-effect-data.md; this spec does not depend on it (cycle) and names it only as where a host may compute an area's targets (FR-015).

## 2. Source

SRD 5.2.1 (`doc/research/sources.md`). Printed page equals PDF page (checked on pp. 104-107, 129-132, 139-141, 158-160).

- "Spells", "Effects", "Targets" ("A Clear Path to the Target", "Targeting Yourself", "Areas of Effect"), "Invalid Targets", "Awareness of Being Targeted", "Saving Throws", "Attack Rolls", "Combining Spell Effects", p. 106; "Spell Level", "Spell Slots", "Casting without Slots", "Using a Higher-Level Spell Slot", pp. 104-105; "Range", p. 105.
- "Character Creation", "Multiclassing", "Spellcasting", "Cantrips" (increase based on total character level), p. 25.
- "Damage Rolls", "Critical Hits", "Saving Throws and Damage" ("Damage against Multiple Targets", "Half Damage"), p. 16; "Damage Types", pp. 16-17; "Healing", p. 17; "Temporary Hit Points", p. 18; "Melee Attacks", "Ranged Attacks", p. 15; "Saving Throws", "Attack Rolls", p. 7.
- "Rules Glossary": "Area of Effect", p. 177; "Spell Attack", p. 188; "Condition", "Concentration", p. 179; "Damage Types" table, p. 180; "Magical Effect", p. 185.
- Spell entries (page of each heading): Acid Splash 107; Chromatic Orb 115; Cure Wounds 121; Eldritch Blast 127; False Life 129; Fireball 131; Fire Bolt 132; Flame Strike 132; Harm 139-140; Healing Word 139; Hold Person 141; Ice Knife 141; Magic Missile 146; Ray of Sickness 158. The pilot set is listed in spell-effect-data.md section 2.

Package rules: `doc/rules/architecture.md` §2-§6; `doc/rules/observability.md` §2 to §7; `doc/rules/testing.md` §4, §5, §8. Binding decisions: shared-types-and-ownership.md FR-001, FR-005, FR-013, FR-015, FR-017, FR-018, FR-019, FR-041, FR-044, FR-046, FR-048, FR-059.

## 3. Vocabulary

Effect model types (`Spell.effects`, `SpellEffectModel`, `ModelledEffects`, `TextOnlyEffects`, `EffectCoverage`, `SpellTargeting`, `AreaShape`, `SelfTargetPolicy`, `PartResolution`, `SpellOutcome`, `SpellAmount`, `DamageTypeSelection`, `ConditionDuration`, scaling types, `SpellEffectFacts`) are spell-effect-data.md section 3. SRD wording is used exactly; the SRD says "Using a Higher-Level Spell Slot", not "upcast".

| SRD term | Type | Notes |
|---|---|---|
| a target the host names | `SpellTargetRef` (`CreatureTargetRef(CombatantID)`, `ObjectTargetRef(SpellObjectID)`) | FR-093. `SpellObjectID` is an opaque, host-supplied id; the package tracks no object state (as attacks.md does for the target's AC). |
| Attack Rolls (of a spell) | `SpellAttackFacts` | The target-side facts of an attack part. The roll is made by composing the pure `d20` rules functions and the attack domain services of attacks.md, not by a use case (FR-022, FR-096, FR-097). |
| Spell save DC | `SpellSaveDc` (spellcasting.md) in `CastReport` | Wraps a `DifficultyClass` (d20-tests.md). |
| spell attack modifier | `RollModifiers` from `SpellcastingStats.AttackRollModifiers` (spellcasting.md FR-116) | Passed to `d20.ResolveAttack` unchanged (FR-022). |
| Using a Higher-Level Spell Slot, Cantrip Upgrade | `CastReport.CastAtLevel`, `CastReport.ScalingLevel` | The resolver derives n from them (FR-043); the scaling shapes are spell-effect-data.md's. Cantrip Upgrade uses total character level. |
| Half Damage | `HalfDamage` adjustment of `PlannedDamage` | p. 16. Rounding is hit-points-and-damage.md's (FR-026). |
| Invalid Targets | `TargetValidity`, `InvalidTargetPerception` | p. 106. |
| A Clear Path to the Target | `PathToTarget` | p. 106. |
| D20 Test resolution | the pure `d20` rules functions | The pure rules functions of `d20` (d20-tests.md FR-069). Composed here; `d20.Resolver` is never called. |
| (what the resolver returns) | `SpellEffectsResolved`, `PlannedDamage`, `TargetEffects`, `ConditionToApply` | Package terms; a plan, not a state change (Q2). |
| creature in a fight | `CombatantID` | Owned by combat-foundations.md (FR-001; shared-types-and-ownership.md FR-012). |

## 4. Requirements

Stable ids. "The model" is `SpellEffectModel`, "a part" a `SpellEffectPart` (spell-effect-data.md). Withdrawn ids are in the table at the end of this section. Case-table labels are not ids.

### 4.1 Targeting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-015 | The resolver MUST NOT decide who is inside an area of effect, in range, in line of sight or visible. The command lists the targets the host chose, in a fixed order, and a `SpellTargetRef` listed twice MUST be rejected with an error that is `errors.Is` `ErrInvalidSpellEffectsCommand`. For a `TargetsInArea` spell the list is the host's; the host MAY compute it with `AreaMembership.assess` of magical-effects.md, which this spec does not call, and the creator's choice to include the origin of a Cone, Cube, Emanation or Line is that spec's `CreatorIncludesOrigin`, not a field here (spell-effect-data.md FR-014). | SRD 5.2.1, "Area of Effect" (blocked lines need Total Cover), p. 177; attacks.md section 1 (the host supplies the facts the GM decides) | unhappy (invalid input) |
| FR-017 | A target that is invalid (outside the creature type restriction, an `ObjectTargetRef` under an `OnlyCreatureTypes` restriction with the subject `CreatureOrObject`, which has no `CreatureType`, or reported by the host as `TargetValidity.CannotBeAffected`) MUST receive no outcome, no roll and no `Damage`. Its result MUST carry an `InvalidTargetPerception`: `AppearedToSucceed` when the part is a `SavingThrowResolution` whose success effect is `NothingOnSuccess`; `SpellDidNothing` otherwise. The slot is not refunded here (the cast is `spellcasting.md`'s). See table I. | SRD 5.2.1, "Invalid Targets", p. 106 | unhappy (rule interaction) |
| FR-018 | A `TargetsChosen` target the host reports as `PathToTarget.BlockedByTotalCover` MUST be rejected with the reason `NoClearPath`; for `TargetsInArea` the host's list already reflects blocked lines and no `PathToTarget` is read. | SRD 5.2.1, "A Clear Path to the Target", p. 106; "Area of Effect", p. 177 | unhappy (state error) |
| FR-019 | The number of targets MUST match the model after scaling: `ExactlyTargets(n)` requires exactly n targets and `UpToTargets(n)` requires 1 to n, else the reason `WrongTargetCount`; `TargetsSelf` requires exactly one target, the caster's own `CreatureTargetRef`: an empty list, a list of more than one, a creature other than the caster and an `ObjectTargetRef` are all `WrongTargetCount` (R26 to R28); `TargetsByInstance` is checked by FR-031. `TargetsInArea` has no count rule (FR-020). | SRD 5.2.1, "Targets", p. 106; "Hold Person" (one additional Humanoid per slot level above 2), p. 141 | unhappy (boundary) |
| FR-020 | A `TargetsInArea` cast with no creature in the area MUST be valid: it MUST return `SpellEffectsResolved` with no target effects, roll no saving throw and request no die at all: a damage roll that no creature takes affects nothing, and requesting it would only move the host's roller sequence. This is a package decision (the SRD is silent on casting at an empty area), recorded as Q17; the same rule applies when every listed target is invalid (FR-017). | SRD 5.2.1, "Areas of Effect", p. 106 (an area "determines what the spell targets"); package decision (Q17) | unhappy (empty and minimal) |

### 4.2 Resolution methods and outcomes

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-022 | For an `AttackRollResolution` (its `SpellAttackKind`, `melee` or `ranged`, is spell-effect-data.md FR-042) the roll MUST be made without calling a use case (FR-096), in this order: the matching `MeleeSpellAttack` or `RangedSpellAttack` form (attacks.md FR-087) and the target's `SpellAttackFacts` (FR-097) go to `CircumstanceCollector` (attacks.md FR-125), which returns an `AttackPlan`; for `AttackPlanProceeds` the use case calls `modifier.Resolve` once on its circumstances, requests the d20 or d20s from the `dice.Roller` through `dice.Roll`, builds the `Roll` from the mode and the faces (d20-tests.md FR-055), and passes it with the caster's `RollModifiers` from `SpellcastingStats.AttackRollModifiers` (spellcasting.md FR-116) and the effective AC to `d20.ResolveAttack` (d20-tests.md FR-069). Cover comes first: `CoverResolver` (attacks.md FR-017, FR-018) picks the applicable cover from `SpellAttackFacts.CoverSources`, and the effective AC is the target's `ArmorClass` plus that cover's `AcBonus` (attacks.md FR-004, FR-015). Applicable Total Cover gives the rejection `AttackNotPossible` (attacks.md FR-019) before any die is requested. The cover bonus is carried in the evidence (FR-042). This spec MUST NOT compute cover, range, reach, Advantage or the natural 20 itself. `AttackPlanAutomaticMiss` is a miss with no d20 rolled and applies `OnMiss`; `AttackPlanNotPossible` is the rejection `AttackNotPossible` (FR-057). `PathToTarget.BlockedByTotalCover` (FR-018) governs `TargetsChosen` targeting only (`NoClearPath`); the Total Cover derived from `SpellAttackFacts.CoverSources` serves only the effective AC and `AttackNotPossible`. When both apply to one target, both reasons are reported in FR-057 order. | SRD 5.2.1, "Attack Rolls" (spell), p. 106; "Spell Attack", p. 188; "Melee Attacks", "Ranged Attacks", p. 15; shared-types-and-ownership.md FR-044 | happy |
| FR-023 | On a hit, including a Critical Hit, the `OnHit` outcomes MUST apply (non-empty); on a miss the `OnMiss` outcomes MUST apply (MAY be empty). The final hit and Critical Hit are derived as attacks.md derives them (the `AttackRollResult` plus the `AutomaticCritical` input of `SpellAttackFacts`). For an `AttackRollResolution` each `SpellAmount` of `OnHit` MUST be mapped to a `DamageSpec` of `DiceDamage` components (the dice and flat modifier of its `dice.Expression`, `AddsAbilityModifier` from its `ModifierAddition`) and the `AttackDamageComposer` alone produces the `Damage` (attacks.md FR-056 to FR-059, FR-114; `HitKind`): it doubles the dice on a Critical Hit, never a flat modifier, and applies the one ability modifier and the floor at 0. This MUST NOT be reimplemented, and the resolver's own modifier and floor step of FR-032 applies only to save and automatic parts. See O12. | SRD 5.2.1, "Critical Hits", p. 16; "Attack Rolls", p. 7 | happy |
| FR-024 | Each valid target's save of a `SavingThrowResolution` (its `Ability` is spell-effect-data.md FR-042) MUST be resolved by the pure `d20` rules functions, without calling a use case (FR-096): for a `RollAttempt` the use case calls `modifier.Resolve` once on the attempt's circumstances, requests the d20 or d20s through `dice.Roll`, builds the `Roll` (d20-tests.md FR-055) and calls `d20.ResolveCheck` with the `SavingThrow` kind, the roll, the attempt's `RollModifiers` and the `CastReport`'s spell save DC; for a `ChooseToFail` attempt it calls `d20.FailWithoutRoll` with `ForcedFailureReason.ChosenToFail`, and for an `AutomaticFailure` attempt with `ForcedFailureReason.automatic` (d20-tests.md FR-035, FR-036), and rolls no d20. | SRD 5.2.1, "Saving Throws", p. 106 ("the spell specifies the ability that the target uses for the save"); "Saving Throws", p. 7; shared-types-and-ownership.md FR-044 | happy |
| FR-025 | On a failed save the `OnFailure` outcomes MUST apply (non-empty). On a success `NothingOnSuccess` applies nothing; `HalfDamageOnSuccess` applies every `SpellDealsDamage` of `OnFailure`, each marked `HalfDamage` (FR-026), and none of the other outcomes of `OnFailure` (Harm: "it takes half as much damage only"). No per-outcome mark exists in the model: the halving applies to all of the damage outcomes of the part, so a spell that wants only one of several damages halved is not expressible and stays `partial`. See table S (S8, S13). | SRD 5.2.1, "Saving Throws" (spell), p. 106; "Half Damage", p. 16; "Harm", pp. 139-140 | happy |
| FR-026 | The resolver MUST NOT compute the halved amount. Each damage of a halved success (every `SpellDealsDamage` of `OnFailure`, FR-025) MUST be reported at the full rolled amount inside a `PlannedDamage` carrying the `HalfDamage` adjustment of hit-points-and-damage.md (FR-099 there), so the rounding down and the order of application stay in one place. | SRD 5.2.1, "Half Damage", p. 16; "Order of Application", p. 17; hit-points-and-damage.md FR-014, FR-099 | unhappy (boundary) |
| FR-028 | The damage of a `SavingThrowResolution` MUST be rolled once per part for all of its targets: every target of the part gets the same rolled amount, and the dice are requested before the first save (FR-041). The dice are requested only when the part has at least one valid target (FR-020). | SRD 5.2.1, "Damage against Multiple Targets", p. 16 | happy |
| FR-029 | An `AutomaticResolution` MUST roll nothing to hit and MUST apply its outcomes to the targets of each instance; an amount in it MUST be rolled once per instance and shared by that instance's targets. Rolling once is a package decision for healing and Temporary Hit Points, where the SRD states a rule only for damage (Q9). | SRD 5.2.1, "Magic Missile", p. 146; "Cure Wounds", p. 121; "Damage against Multiple Targets", p. 16 (damage only); package decision (Q9) | happy |
| FR-030 | An `AttackRollResolution` MUST roll each hit's damage separately (one damage roll per instance that hits), never once for several attack rolls. | SRD 5.2.1, "Eldritch Blast" ("Make a separate attack roll for each beam"), p. 127; "Damage Rolls", p. 16 | unhappy (rule interaction) |
| FR-031 | `PartInstances` MUST give the number of times the part happens (a dart, ray or beam). Each instance is resolved on its own and is assigned one target. When the command has exactly one target, assignment is implicit. Otherwise a part that is an attack roll, or has more than one instance, MUST have an `InstanceAssignment` listing one target id per instance, each id among the command's targets; any other count, an unknown id, or a listed target that none of the part's instances is assigned to MUST be rejected with `InstanceAssignmentMismatch` (a host that does not want to target a creature does not list it). A part with one instance that is not an attack roll applies once to every target. | SRD 5.2.1, "Magic Missile" ("you can direct them to hit one creature or several"), p. 146; "Eldritch Blast", p. 127 | unhappy (rule interaction) |
| FR-032 | A `SpellAmount` MUST be evaluated as: roll a `RolledSpellAmount`'s `dice.Expression` through `dice.Roll` (its flat modifier is part of the expression), or take a `FixedSpellAmount` as is; then add the casting class's spellcasting ability modifier once when the outcome says `ModifierAddition.SpellcastingAbilityModifier`; then raise a negative total to 0 (for the parts of FR-023 the `AttackDamageComposer` does these two steps instead). The use case only requests the dice; the `SpellEffectResolver` adds the modifier and applies the floor (section 5.1). The floor at 0 is the SRD's for damage ("possible to deal 0 damage but not negative"); applying it to healing and Temporary Hit Points is a package decision (Q10). | SRD 5.2.1, "Damage Rolls", p. 16; "Cure Wounds", p. 121 ("2d8 plus your spellcasting ability modifier") | unhappy (boundary) |
| FR-033 | For a spell with a `ChosenDamageType` (spell-effect-data.md FR-044) the command's `ChosenDamageType` MUST be present (else the reason `DamageTypeNotChosen`) and in the set (else `DamageTypeNotAllowed`). A `ChosenDamageType` supplied for a spell with no `ChosenDamageType` set MUST be rejected with an error that is `errors.Is` `ErrInvalidSpellEffectsCommand`. | SRD 5.2.1, "Chromatic Orb" ("Choose Acid, Cold, Fire, Lightning, Poison, or Thunder"), p. 115; "Damage Types", p. 180 | unhappy (invalid input) |
| FR-035 | A `SpellDealsDamage` MUST produce `Damage` values of hit-points-and-damage.md (an amount and a `DamageType`) inside a `PlannedDamage`, grouped per part, target and `HitKind` (FR-065). The resolver MUST NOT apply Resistance, Vulnerability, Immunity, Temporary Hit Points or Hit Point arithmetic. | SRD 5.2.1, "Damage Types", p. 16; "Order of Application", p. 17; hit-points-and-damage.md FR-085, FR-086 | happy |
| FR-036 | A `SpellRestoresHitPoints` MUST produce a healing amount for `RestoreHitPoints`. The resolver MUST NOT cap it at the Hit Point maximum (that is hit-points-and-damage.md's). An amount of 0 after FR-032 MUST produce no healing outcome. | SRD 5.2.1, "Healing", p. 17; "Healing Word", p. 139 | unhappy (boundary) |
| FR-037 | A `SpellGrantsTemporaryHitPoints` MUST produce an amount for `GainTemporaryHitPoints`; an amount of 0 after FR-032 MUST produce no outcome, because that use case rejects an amount below 1 (hit-points-and-damage.md FR-030). Choosing between old and new Temporary Hit Points is the host's (`TemporaryHitPointsChoice`). | SRD 5.2.1, "Temporary Hit Points" ("They Don't Stack"), p. 18; "False Life", p. 129 | unhappy (boundary) |
| FR-038 | A `SpellAppliesCondition` (a `ConditionKind` and a `ConditionDuration`, spell-effect-data.md FR-028) MUST produce a `ConditionToApply` naming the target, the kind, the caster as source (`CombatantID`) and the duration. `ConditionToApply.ToCondition()` MUST give the `Condition` of conditions.md that the host passes to `ApplyConditionToSet`: for `Charmed`, `Frightened` and `Grappled`, the variant with the caster's `CombatantID` as its source (conditions.md FR-003); for `Exhaustion`, `Exhaustion(1)`, because conditions.md FR-123 ignores the level of an applied `Exhaustion` and adds exactly one level; for every other kind, the variant of that kind. The duration is not an input of `ApplyConditionToSet` (conditions.md has no duration); it is handed on as data to the host and to the later active-effects spec (Q11). The resolver MUST NOT apply the condition (that is `ApplyConditionToSet`), and "a condition doesn't stack with itself" stays `ConditionSet`'s rule. | SRD 5.2.1, "Condition", p. 179; "Ray of Sickness", p. 158; "Hold Person", p. 141 | happy |
| FR-040 | The resolver MUST NOT track durations, Concentration, repeated saves or expiry. It MUST hand the `ConditionDuration` on as data. The result MUST be identical for any `Spell.duration` and any `RequiresConcentration` value, all else equal (a test builds the same model on spells that differ only in those fields and compares the results). | SRD 5.2.1, "Concentration", p. 179; spellcasting.md (Concentration is owned there); Q11 | unhappy (state error: out of this spec) |
| FR-041 | For a fixed `dice.Roller` sequence the roll order MUST be fixed: parts in order; inside a saving-throw part, the damage dice of its outcomes in outcome order, then each valid target's save in command order; inside an attack part, per instance in order, the d20 (`1d20`, or `2d20` under Advantage or Disadvantage) and then, on a hit, the damage dice as the `AttackDamageComposer` states them (doubled on a Critical Hit); inside an automatic part, per instance in order, the amounts in outcome order. See table O. | `doc/rules/testing.md` §4 (determinism); package design | unhappy (determinism) |
| FR-042 | `SpellEffectsResolved` MUST carry the evidence for each instance in `ResolvedPart` and `ResolvedInstance` values: for an attack roll the `AttackRollResult` (which holds the `Roll` with its mode, the modifiers, the total and the outcome) unchanged, with the cover bonus added to the target's AC (FR-022), or the `AttackPlanAutomaticMiss` reason for an automatic miss; for a saving throw the `CheckResult` or `ForcedFailureResult` unchanged; and for every instance the amounts rolled (each rolled die) and the outcomes applied, so a consumer can render the breakdown and a host can reroll a die (FR-095). They hold domain values only (no use case result), so they live in `combat/domain/` (FR-087). | `doc/rules/architecture.md` §6 (results explain themselves) | happy |

### 4.3 Scaling from the cast

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-043 | Slot scaling MUST use n = the larger of 0 and (`CastReport.CastAtLevel` minus `AboveLevel`). A cast with no `CastAtLevel` MUST use n = 0: casting-and-concentration.md FR-082 leaves `CastAtLevel` absent for a Ritual (which also has no `ScalingLevel`) and for a cantrip; a cantrip never has slot scaling (spell-effect-data.md FR-009), so for it n is not used and its table is selected by `ScalingLevel` (spell-effect-data.md FR-018). The resolver passes n, and `ScalingLevel` as the total character level, to the scaling queries of spell-effect-data.md FR-041; the scaling shapes (steps, tables) are that spec's. A cast at exactly `AboveLevel` MUST use the base value. See table N. | SRD 5.2.1, "Using a Higher-Level Spell Slot", pp. 104-105; "Multiclassing", "Spellcasting", p. 25 ("you can use the enhanced effect as normal") | unhappy (boundary) |
| FR-050 | `ResolveSpellEffects.Do` MUST return a nil result and an error that is `errors.Is` `ErrInvalidSpellEffectsCommand` (wrapping `srderr.ErrInvalidInput`) when the `CastReport`'s spell id differs from the `Spell`'s, when `CastAtLevel` is present and below the spell's level, or when a model needs `ScalingLevel` (a character-level table, which only a cantrip has, spell-effect-data.md FR-009) and the report has none, or when a part needs the spell save DC or a spell attack modifier and the report's value is absent (a monster report built from printed values may omit either; `monster-spellcasting.md` FR-030). A cantrip report has no `CastAtLevel` and has `ScalingLevel`; a Ritual report has neither (casting-and-concentration.md FR-082) and uses the base values (FR-043). | SRD 5.2.1, "Spell Level", "Spell Slots", p. 104; casting-and-concentration.md FR-059, FR-082 | unhappy (invalid input) |

### 4.4 Command, result and rejection

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-054 | `ResolveSpellEffects` MUST return the closed `ResolveSpellEffectsResult` with exactly the variants `SpellEffectsResolved`, `SpellEffectsTextOnly` and `SpellEffectsRejected`, each prefixed with the operation. No generic variant name is exported. | `doc/rules/architecture.md` §4; shared-types-and-ownership.md FR-017 | happy |
| FR-055 | A `Spell` with `TextOnlyEffects` MUST return `SpellEffectsTextOnly` (the spell id and the effective level) without rolling any die and without validating targets. It is an expected outcome, not a rejection or an error. | SRD 5.2.1, "Effects", p. 106 (GM's purview); `doc/rules/architecture.md` §6 | unhappy (missing content) |
| FR-056 | `SpellEffectsResolved` MUST report the model's `EffectCoverage`. With `partial`, the host MUST be able to tell that part of the description is unresolved. | SRD 5.2.1, "Effects", p. 106 | happy |
| FR-057 | `SpellEffectsRejected` MUST list every problem found, as values of `SpellEffectsRejectionKind` in this fixed order: `WrongTargetCount`, `NoClearPath`, `InstanceAssignmentMismatch`, `MissingSaveAttempt`, `MissingAttackFacts`, `DamageTypeNotChosen`, `DamageTypeNotAllowed`, `CasterNotAllowed` (FR-092), `WrongTargetSubject` (FR-094), `AttackNotPossible` (FR-022: an `AttackPlanNotPossible` for any target of an attack part). A reason is reported once however many targets cause it. See table R. Each reason has one `CombatResultClass` under combat-foundations.md FR-031, and a rejection takes the most severe class of its reasons (FR-080). | `doc/rules/architecture.md` §5, §6; `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-058 | A rejection MUST be returned before any die is requested: a recording `dice.Roller` MUST see no request. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-059 | For every valid target, a part that needs a saving throw MUST find a `SaveAttempt` for its ability in the target's saves, else `MissingSaveAttempt`; an attack part MUST find the target's attack facts, else `MissingAttackFacts`. Invalid targets (FR-017) are exempt. | d20-tests.md (`SaveAttempt`); attacks.md section 6 | unhappy (missing content) |
| FR-060 | Commands, models and results MUST NOT be mutated by the call, and every slice or map a result exposes MUST be a copy (copy on construction and on read). Results MUST have value equality (an `Equal` method where the type is not comparable). | `doc/rules/architecture.md` §4 (handoffs); `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-061 | A roller failure MUST be returned wrapped exactly once as a `*srderr.PortError` for `dice.Port` (`errors.Is` the original cause), the use case MUST return no partial result, and it MUST NOT recover a panic itself (`dice.Roll` already recovers a roller panic as `srderr.ErrPortPanic`). | `doc/rules/architecture.md` §6; errors-and-results.md; observability.md §7 | unhappy (collaborator failure) |
| FR-062 | Contradictory command input that no rule gives meaning to (FR-015, FR-033, FR-050, FR-098) MUST return a nil result and an error that is `errors.Is` `ErrInvalidSpellEffectsCommand` (which wraps `srderr.ErrInvalidInput`), never a result variant and never a panic. | `doc/rules/architecture.md` §6; `errors-and-results.md` FR-025 to FR-027 | unhappy (invalid input) |
| FR-063 | The same command and the same roller sequence MUST give an equal result, . | `doc/rules/architecture.md` §3 (Purity); `doc/rules/testing.md` §4 (determinism) | unhappy (determinism) |
| FR-064 | The result is a plan: `ResolveSpellEffects` MUST change no `HitPointStatus`, `ConditionSet` or other state, and MUST return exactly these things for the four existing use cases: per target, its `PlannedDamage` values (FR-065), the healing amounts, the Temporary Hit Points amounts and the `ConditionToApply` values, each with the part it came from, plus the evidence of FR-042. Which use case applies what, and in which order, is the host's (Q2). | `doc/rules/architecture.md` §4 (state in, new state and events out is held by the owners of that state); package design (Q2) | happy |
| FR-065 | `PlannedDamage` MUST be one per damage event: one per damage-bearing part, instance and target. It carries the `Damage` values of that instance's `SpellDealsDamage` outcomes (kept separate, in outcome order: Flame Strike's Fire and Radiant are one `PlannedDamage`), the optional `HalfDamage` adjustment, the `HitKind` (`CriticalHit` only for an attack instance that scored a Critical Hit) and its part and instance indices. Instances are never merged: two beams or three darts at one target give two or three `PlannedDamage` values, and a save part's one roll gives one per target. hit-points-and-damage.md FR-085 allows one `HitKind` and one adjustment list per `ApplyDamage` and applies the Death Saving Throw failure once per command, so the host passes one `PlannedDamage` per `ApplyDamage` command (settled, Q8). See table O (O5, O10). Spell damage carries no `DamageDelivery` (hit-points-and-damage.md FR-085): the host passes it, `OtherSource` by default, and for a `melee` `SpellAttackKind` the host chooses between `MeleeAttackWithoutKnockOut` and `MeleeAttackKnockOut` ("Knocking Out a Creature", p. 17). | SRD 5.2.1, "Damage at 0 Hit Points", p. 18 ("If you take any damage"); hit-points-and-damage.md FR-085, FR-099; Q8 | unhappy (rule interaction) |

### 4.5 telemetry.Telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-080 | `ResolveSpellEffects` MUST record through the combat telemetry helper (combat-foundations.md FR-035) with the operation `resolve_spell_effects`, so each call runs in the span `srd5e.combat.resolve_spell_effects` and records the shared `srd5e.combat.operations`, `srd5e.combat.operation_duration` and `srd5e.combat.errors` signals exactly as combat-foundations.md FR-027 to FR-032 define them. This spec MUST NOT define a span, throughput counter, duration histogram or error counter of its own. Its result values are classified (`CombatResultClass`) as: `SpellEffectsResolved` and `SpellEffectsTextOnly` are `applied`; a `SpellEffectsRejected` is `invalid` (span error status) when any of its reasons is `WrongTargetCount`, `InstanceAssignmentMismatch`, `MissingSaveAttempt`, `MissingAttackFacts`, `DamageTypeNotChosen` or `DamageTypeNotAllowed` (host input the engine cannot use), and `refused` (no error status) when every reason is `NoClearPath`, `CasterNotAllowed`, `WrongTargetSubject` or `AttackNotPossible` (legal input that the spell's rules do not allow). `srd5e.outcome` is `success` or `failure` only, from the class. | combat-foundations.md FR-027 to FR-032, FR-035; `doc/rules/observability.md` §4, §5, §6; shared-types-and-ownership.md FR-041 | happy |
| FR-081 | `srd5e.combat.result` (combat-foundations.md FR-030: a closed set owned by this spec, each value mapping to exactly one `CombatResultClass`) MUST be `resolved` and `text_only` (class `applied`), `rejected_invalid` (class `invalid`) and `rejected_refused` (class `refused`, FR-080); these four values are added to the catalogue. The use case MUST add these extra low-cardinality attributes through the `attributes` closure of the combat telemetry helper's `run` (combat-foundations.md FR-041), so they appear on the span and on the `srd5e.combat.operations` increment of the call, and nowhere else: `srd5e.combat.spell_effects.coverage` (`complete`, `partial`, `text_only`) and, for a rejection only, `srd5e.combat.spell_effects.rejection` (the first reason in FR-057 order, as the snake_case value: `wrong_target_count`, `no_clear_path`, `instance_assignment_mismatch`, `missing_save_attempt`, `missing_attack_facts`, `damage_type_not_chosen`, `damage_type_not_allowed`, `caster_not_allowed`, `wrong_target_subject`, `attack_not_possible`). Both keys and their closed value sets MUST be listed in the Attributes cell of the shared `srd5e.combat.operations` row (combat-foundations.md FR-037); the added cardinality is at most 3 coverage values times 10 rejection reasons. It MUST increment `srd5e.combat.spell_effects.resolutions` once for every call that is not rejected (`resolved` and `text_only`), with `srd5e.combat.spell_effects.coverage` and `srd5e.combat.spell_effects.level` (`cantrip` or `1` to `9`). The effective level is `CastReport.CastAtLevel`, else the spell's own level (a Ritual, which has no `CastAtLevel`, so reports its own level; level 0 is `cantrip`); the same value is the `EffectiveLevel` of the results (section 6). `srd5e.combat.spell_effects.targets` (the number of valid targets) MUST be recorded only for `resolved` calls; neither signal is recorded for a rejected call. The `result` values are deliberately no finer than the class for the two rejections (`rejected_invalid`, `rejected_refused`): the reason is the `rejection` attribute, which is a deviation from the "finer than `result_class`" wording of combat-foundations.md FR-030, kept so the closed set stays small. No attribute carries a spell id, a combatant id or an amount (combat-foundations.md FR-033). | combat-foundations.md FR-029, FR-030, FR-033; `doc/rules/observability.md` §5 | happy |
| FR-082 | Each attack roll and each saving throw of a `resolved` call MUST increment `srd5e.combat.spell_effects.rolls` once, with `srd5e.combat.spell_effects.roll` (`attack`, `save`, `save_forced`) and `srd5e.outcome` (`success`, `failure`; an attack hit, Critical Hit included, is `success`; a saved target is `success`) and, for `save` and `save_forced`, the shared `srd5e.ability` (six lowercase ability names), read from the constant that the `d20` signal constants (`d20.AttrAbility`) exports (resolve-d20-test-use-case.md FR-038; combat-foundations.md FR-027), not a combat-local key. The closed value sets of this requirement and of FR-081 are added to the catalogue. A `ChooseToFail` or `AutomaticFailure` save (no d20 rolled) MUST be counted as `save_forced` with `failure`, so it does not skew the save success share of rolled saves. An `AttackPlanAutomaticMiss` instance (FR-022) rolled no attack roll and MUST record no `rolls` increment, so it does not skew the attack hit share either (R24). An invalid target (FR-017), a `text_only` call and a `rejected` call roll nothing and record no roll. | `doc/rules/observability.md` §5, §8; shared-types-and-ownership.md FR-041 | happy |
| FR-083 | A roller failure MUST be recorded once, by the combat telemetry helper of this use case, which calls no other use case (FR-096) (span error status and `RecordError`, one increment of the shared `srd5e.combat.errors` with `error.type` from `srderr.ErrorType(err)`), and MUST NOT be logged; an invalid-command error (FR-062) MUST be recorded on the span only. The use case MUST NOT log at any level for a normal outcome or a rejection (combat-foundations.md FR-034). | combat-foundations.md FR-032, FR-034; `doc/rules/observability.md` §7 ("Errors: who records what") | unhappy (collaborator failure) |
| FR-084 | Because the use case calls no other use case and builds none (FR-096), one call MUST record only the shared `combat` signals and the `srd5e.combat.spell_effects.*` signals of FR-081 and FR-082 (`srd5e.ability` is the `d20` key, used only on the `rolls` counter). It fires no signal of the `ResolveAttack` family (the span `srd5e.combat.resolve_attack` and the shared `srd5e.combat.operations` row with that operation) and none of the `srd5e.d20_test.*` family: those belong to the use cases that make them, which are not called. No `telemetry.Telemetry` is silenced to hide a nested call. The use case produces no `combat.Event`, so it increments no `srd5e.combat.events`. | shared-types-and-ownership.md FR-044; combat-foundations.md FR-029, FR-032; `doc/rules/observability.md` §7 ("Use cases never call one another") | unhappy (rule interaction) |
| FR-085 | The use case MUST return an equal result with the zero-value no-op `telemetry.Telemetry` and with the recording `telemetrytest` doubles. All constants (the operation name, attribute keys and the closed value sets of this section) MUST be added to the exported constants of `combat/signals.go` (combat-foundations.md FR-027), with no string literal at a call site. | `doc/rules/observability.md` §9; combat-foundations.md FR-027 | happy |

### 4.6 Layering and exports

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-087 | The interpreter (`SpellEffectResolver`, an unexported stateless type or set of pure functions), the plan value objects (`TargetEffects`, `PlannedDamage`, `ConditionToApply`, `InvalidTargetPerception`), the evidence types `ResolvedPart` and `ResolvedInstance` (FR-042) and the input value objects (`SpellTarget`, `SpellAttackFacts`) hold only domain values and MUST live in package `combat`, in files named `spell_effects*.go`, pure and silent (no `srderr`, `telemetry`, `dice.Roller` or `context` reference). `ResolveSpellEffects`, its command, result and rejection types MUST live in `resolve_spell_effects_usecase.go` and be synchronous, because the only port it calls, `dice.Roller`, is synchronous; `ctx` is its first parameter because it starts a span, and it is never stored (`doc/rules/architecture.md` §1, §9). It MUST import `character`, `d20` and `content` only through their exported API, and uses the pure `d20` rules functions and the attack domain functions of attacks.md (same context, to its left in the order). | `doc/rules/architecture.md` §1, §2, §4; observability.md §3; shared-types-and-ownership.md FR-044 | happy |
| FR-088 | No new port is introduced, so no new test double or contract suite is needed. Tests MUST use `dicetest.FixedRoller` (dice-and-rolls.md FR-054), `dicetest.CallbackRoller (returning an error)` for FR-061 and the recording `telemetrytest` doubles. | `doc/rules/architecture.md` §4, §7 | happy |
| FR-089 | Every exported name of section 6 MUST be unique across the module's packages, none may be a generic variant name (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`, `Found`, `NotFound`), and no exported name may refer to a consuming project. | `doc/rules/architecture.md` §4, §9; shared-types-and-ownership.md FR-017, FR-021 | unhappy (name collision) |
| FR-090 | The dependency directions MUST hold: the model (spell-effect-data.md FR-039) imports nothing from `combat` or `character`; the interpreter imports `character`, `d20` and `content` and never another context's internals or use cases; `content` and `character` import nothing from this spec's `combat` files. | `doc/rules/architecture.md` §2 (context map); shared-types-and-ownership.md FR-013 | unhappy (rule interaction) |

### 4.7 Targets, self-targeting and composition

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-091 | The combat part MUST use `CombatantID` of combat-foundations.md for every creature (the caster and each creature target) and MUST define no creature id. | combat-foundations.md FR-001, FR-025; shared-types-and-ownership.md FR-012 | happy |
| FR-092 | A command listing the caster among the targets of a `TargetsChosen` spell whose `SelfTargetPolicy` is `ExcludesCaster` (the field and its meaning are spell-effect-data.md FR-040) MUST be rejected with `CasterNotAllowed`. Under `MayIncludeCaster` the caster MAY be listed and is validated like any other target. `TargetsSelf` is unaffected, and a `TargetsInArea` spell affects the caster when the host lists the caster (the area decides, p. 106). That a creature really is Hostile is the host's fact and is not modelled (spell-effect-data.md FR-004). | SRD 5.2.1, "Targeting Yourself", p. 106 | unhappy (rule interaction) |
| FR-093 | A listed target MUST be a closed variant set `SpellTargetRef` (an interface with an unexported marker method): `CreatureTargetRef(CombatantID)` or `ObjectTargetRef(SpellObjectID)`. `SpellObjectID` is an opaque, non-blank string the host supplies, a defined string type (`type SpellObjectID string`) with no behaviour; `ParseSpellObjectID(string) (SpellObjectID, error)` MUST return an error that is `errors.Is` `ErrBlankSpellObjectID` (wrapping `srderr.ErrInvalidInput`) for a blank host-supplied value, and `MustSpellObjectID`, for a value the caller has already checked, MUST panic with that error on a blank string. The duplicate-target (FR-015) and unexpected `ChosenDamageType` (FR-033) cases are decided to be invalid-input errors returned by the use case (FR-062), not result variants: the host builds the command from values it already holds, and no host number or text is parsed there. The package tracks no object state, so an object target has no `CreatureType` and no `CombatantID`; for an attack part the host supplies its armor class and the other target-side facts in the `SpellAttackFacts` of FR-097, as for any target (attacks.md has no target type, because a target's kind changes no rule). A saving throw part with an object target MUST be rejected with `MissingSaveAttempt`, because the host has no `SaveAttempt` for an object. | SRD 5.2.1, "Targets" ("creatures, objects, or something else"), p. 106; "Fire Bolt", p. 132; "Eldritch Blast", p. 127; attacks.md section 1 (the host supplies the facts about a target) | unhappy (invalid input) |
| FR-094 | A creature target for a spell whose subject is `object`, and an object target for a spell whose subject is `creature`, MUST be rejected with `WrongTargetSubject`. `CreatureOrObject` accepts both. | SRD 5.2.1, "Targets", p. 106; "Ice Knife" ("one creature"), p. 141 | unhappy (invalid input) |
| FR-095 | A class-feature rider on a spell's damage roll (Agonizing Blast adds the Charisma modifier to a cantrip's damage, class-warlock.md FR-055; Potent Spellcasting, class-druid.md; Empowered Spell rerolls damage dice, class-sorcerer.md) is not applied by this use case. The host MUST apply a flat bonus to the `Damage` amount of a `PlannedDamage` before it calls `ApplyDamage`, and adds it once per damage roll, never doubled by a Critical Hit (the Critical Hit doubles dice only, hit-points-and-damage.md and attacks.md FR-056 to FR-059). The evidence of FR-042 carries each rolled die so a host can reroll. Because a `PlannedDamage` is one damage roll (FR-065), "once per damage roll" is once per `PlannedDamage`: Agonizing Blast adds to each beam. No first-class extension point exists (settled, Q18). | SRD 5.2.1, "Damage Rolls", "Critical Hits", p. 16; class-warlock.md FR-055; Q18 | unhappy (rule interaction) |
| FR-096 | `ResolveSpellEffects` MUST NOT call, construct, hold or import any use case (`ResolveAttack`, `d20.Resolver`, `ApplyDamage` and the rest). Its constructor `NewResolveSpellEffects(roller dice.Roller, opts ...Option) (ResolveSpellEffects, error)` takes the roller (nil is rejected with an error `errors.Is` `srderr.ErrInvalidInput`) and the option `WithTelemetry(telemetry.Telemetry)` (default: silent no-ops) and nothing else; it declares its own one-method consumer-side roller interface so it is not coupled to a `dice` implementation. It obtains every D20 Test result from the pure `d20` rules functions of `d20` (FR-022, FR-024) and every cover, attack plan, circumstance and damage composition from the domain services of attacks.md (`CoverResolver`, `CircumstanceCollector`, `AttackDamageComposer`), and it records only its own signals (FR-084). | `doc/rules/architecture.md` §4; shared-types-and-ownership.md FR-044; architecture-guardrails.md FR-097 | unhappy (rule interaction: nested telemetry) |
| FR-097 | For each target of an attack part the host MUST supply a `SpellAttackFacts` value object: the target's `ArmorClass` (`d20`), the target-side inputs of attacks.md's `CoverResolver` (the `CoverSource` list, which feeds the effective AC and Total Cover, FR-022) and of its `CircumstanceCollector` (distance, `AttackRange`, `Reach`, the nearby enemies, `AttackSituation`; cover sources are not collector inputs) and the `AutomaticCritical` input already computed by conditions.md. It holds no kind and no damage specification, because this spec fills those from the model. This spec MUST NOT call `ConditionEffects`: every condition consequence arrives as input (`SpellAttackFacts`, the `SaveAttempt`), and combat-action-resolution.md is the one place that computes it. A target of an attack part without `SpellAttackFacts` is `MissingAttackFacts` (FR-059). | shared-types-and-ownership.md FR-026, FR-044; attacks.md FR-119, FR-125; Q7 (settled) | unhappy (missing content) |
| FR-098 | `ResolveSpellEffectsCommand` MAY carry `PartIndexes` (a non-empty set of indexes into the model's parts) and `InstanceCount` (at least 1), so one call re-applies the parts a `ZoneEffectDue` names (persistent-spell-zones.md FR-054): only those parts are resolved, `InstanceCount` replaces their `PartInstances` (for `TravelsWithin`, one instance per `units`), and the target-count and self-target checks (FR-019, FR-092) are replaced by "at least one creature target". An index out of range, an empty set, or `InstanceCount` without `PartIndexes` MUST be rejected with an error that is `errors.Is` `ErrInvalidSpellEffectsCommand` (FR-062). When `PartIndexes` is absent, a part named by a recurring rule of the spell's `ZoneFacts` (spell-effect-data.md) is skipped, so a cast applies only its immediate parts. Each call rolls on its own: the damage dice of a due effect are rolled for that call's creature (interpretation: the effect happens anew when triggered). Nothing else changes: rolls, evidence, plan and telemetry are those of a normal call. | SRD 5.2.1, "Spirit Guardians" ("When a creature enters... or ends its turn there"), pp. 164-165; "Effects", p. 106; persistent-spell-zones.md FR-054 | unhappy (rule interaction) |

### Table N: slot steps from the cast (FR-043, FR-050; one test per row)

`n` is the larger of 0 and `CastAtLevel` minus `AboveLevel`. The amounts and
counts that follow from `n` are table L of spell-effect-data.md.

| Label | Spell and page | `AboveLevel` | `CastAtLevel` | n | Path |
|---|---|---|---|---|---|
| N1 | Fireball, p. 131 | 3 | 3 | 0 | unhappy (boundary) |
| N2 | Fireball, p. 131 | 3 | 4 | 1 | happy |
| N3 | Fireball, p. 131 | 3 | 9 | 6 | unhappy (boundary) |
| N4 | Cure Wounds, p. 121 | 1 | 1 | 0 | unhappy (boundary) |
| N5 | Hold Person, p. 141 | 2 | 3 | 1 | happy |
| N6 | A synthetic level 1 spell with the Ritual tag (casting-and-concentration.md FR-074; Fireball has no Ritual tag and is rejected as `NotARitual`), cast as a Ritual: no `CastAtLevel` and no `ScalingLevel` (casting-and-concentration.md FR-082) | 1 | absent | 0 (base values, 2d6 for a 2d6 base) | unhappy (rule interaction) |
| N7 | Fire Bolt, p. 132, cast as a cantrip: the report has `ScalingLevel` 5 and no `CastAtLevel` (casting-and-concentration.md FR-082) | n/a | absent | n/a (no slot scaling; the character-level table applies: 2d10) | unhappy (rule interaction) |
| N8 | A cantrip model, report without `ScalingLevel` | n/a | absent | `ErrInvalidSpellEffectsCommand` (FR-050) | unhappy (invalid input) |

### Table S: saving throw outcomes (FR-024, FR-025, FR-026, FR-028, FR-017; one test per row)

Amounts are the rolled totals of the fixed roller in the row. "Plan" is what the
result carries; halving itself is hit-points-and-damage.md's (FR-026).

| Label | Spell and page | Save | Plan | Path |
|---|---|---|---|---|
| S1 | Fireball, p. 131, damage rolls 28 | failed | `PlannedDamage`: `Damage(28, Fire)`, no adjustment | happy |
| S2 | Fireball | succeeded | `PlannedDamage`: `Damage(28, Fire)` with `HalfDamage` | happy |
| S3 | Fireball, two targets, damage rolls 28 | one fails, one succeeds | both plans use 28 (rolled once); the saver's carries `HalfDamage` | unhappy (rule interaction) |
| S4 | Acid Splash, p. 107, damage rolls 4 | failed | `Damage(4, Acid)` | happy |
| S5 | Acid Splash | succeeded | no outcome (`NothingOnSuccess`) | unhappy (boundary) |
| S6 | Hold Person, p. 141 | failed | `ConditionToApply`: Paralyzed, `ForSpellDuration`, source the caster | happy |
| S7 | Hold Person | succeeded | no outcome | unhappy (boundary) |
| S8 | Harm, p. 139, damage rolls 49 | succeeded | `Damage(49, Necrotic)` with `HalfDamage`; no other outcome | unhappy (rule interaction) |
| S9 | Fireball | `ChooseToFail` | failure, no d20 rolled; full plan | unhappy (rule interaction) |
| S10 | Fireball | `AutomaticFailure` | failure, no d20 rolled; full plan | unhappy (rule interaction) |
| S11 | Flame Strike, p. 132, rolls 20 and 17 | failed | two `Damage` values: `(20, Fire)` and `(17, Radiant)` | happy |
| S12 | Hold Person, p. 141, a Beast target | n/a | no roll; `InvalidTargetPerception` per table I | unhappy (rule interaction) |
| S13 | Flame Strike, p. 132, rolls 20 and 17 | succeeded | two `PlannedDamage` damages, `(20, Fire)` and `(17, Radiant)`, each with `HalfDamage` (every `SpellDealsDamage` of `OnFailure` is halved, FR-025) | unhappy (rule interaction) |

### Table I: invalid targets (FR-017, FR-094; one test per row)

| Label | Spell and page | Target | Perception | Path |
|---|---|---|---|---|
| I1 | Hold Person, p. 141 (a Humanoid) | a Beast | `AppearedToSucceed` (save part, `NothingOnSuccess`) | unhappy (rule interaction) |
| I2 | Fireball, p. 131 (half on a success) | host reports `CannotBeAffected` | `SpellDidNothing` | unhappy (rule interaction) |
| I3 | Fire Bolt, p. 132 (attack part) | host reports `CannotBeAffected` | `SpellDidNothing` | unhappy (rule interaction) |
| I4 | Cure Wounds, p. 121 (automatic) | host reports `CannotBeAffected` | `SpellDidNothing` | unhappy (rule interaction) |
| I5 | Hold Person | a Humanoid | no perception; the save is rolled | happy |
| I8 | A part with subject `CreatureOrObject` and `OnlyCreatureTypes({Humanoid})` | an `ObjectTargetRef` | invalid target (FR-017): `AppearedToSucceed` for a save part with `NothingOnSuccess`, else `SpellDidNothing` | unhappy (rule interaction) |
| I9 | Ice Knife, p. 141 (subject `creature`) | an `ObjectTargetRef` | rejected `WrongTargetSubject` (FR-094) | unhappy (invalid input) |

Restriction construction is T1 to T3 of `spell-effect-data.md`.

### Table R: rejections and their order (FR-018, FR-019, FR-020, FR-022, FR-031, FR-033, FR-057, FR-058, FR-059, FR-092, FR-093, FR-094, FR-097; one test per row)

| Label | Command | Reasons, in order | Dice requested | Path |
|---|---|---|---|---|
| R1 | Hold Person at level 3 with 3 targets (up to 2) | `WrongTargetCount` | none | unhappy (boundary) |
| R2 | Hold Person at level 3 with 2 targets | none: resolved | as the case | unhappy (boundary) |
| R3 | Fire Bolt, target `BlockedByTotalCover` | `NoClearPath` | none | unhappy (state error) |
| R4 | Magic Missile at level 2 (4 darts), two targets, 3 assignments | `InstanceAssignmentMismatch` | none | unhappy (invalid input) |
| R5 | Magic Missile, assignment naming a creature not in the targets | `InstanceAssignmentMismatch` | none | unhappy (invalid input) |
| R6 | Fireball, target has no Dexterity `SaveAttempt` | `MissingSaveAttempt` | none | unhappy (missing content) |
| R7 | Fire Bolt, target without attack facts | `MissingAttackFacts` | none | unhappy (missing content) |
| R8 | Chromatic Orb, no chosen type | `DamageTypeNotChosen` | none | unhappy (invalid input) |
| R9 | Chromatic Orb, chosen type Radiant | `DamageTypeNotAllowed` | none | unhappy (invalid input) |
| R10 | Hold Person at level 2 with 3 targets, one `BlockedByTotalCover`, no Wisdom save | `WrongTargetCount`, `NoClearPath`, `MissingSaveAttempt` | none | unhappy (rule interaction) |
| R11 | Fireball with a `ChosenDamageType` supplied | `ErrInvalidSpellEffectsCommand` (FR-033) | none | unhappy (invalid input) |
| R12 | Fireball with one creature listed twice | `ErrInvalidSpellEffectsCommand` (FR-015) | none | unhappy (invalid input) |
| R13 | Fireball, no creature in the area | resolved, empty plan; no die requested, no save (FR-020, Q17) | none | unhappy (empty and minimal) |
| R14 | Fireball with the `CastReport` of another spell | `ErrInvalidSpellEffectsCommand` (FR-050) | none | unhappy (invalid input) |
| R15 | Fireball with `CastAtLevel` 2 | `ErrInvalidSpellEffectsCommand` (FR-050) | none | unhappy (boundary) |
| R16 | Magic Missile at level 2 (4 darts), targets A, B and C, assignments A, A, B, B | `InstanceAssignmentMismatch` (C is listed and unassigned, FR-031) | none | unhappy (rule interaction) |
| R17 | A synthetic `TargetsChosen` spell with `ExcludesCaster`, the caster listed | `CasterNotAllowed` (FR-092) | none | unhappy (rule interaction) |
| R18 | Cure Wounds, p. 121 (`MayIncludeCaster`), the caster listed as the target | none: resolved, the caster is validated like any target | as the case | happy |
| R19 | Fireball, every listed target reported `CannotBeAffected` | resolved, no target effects; no die requested (FR-020) | none | unhappy (empty and minimal) |
| R20 | Fire Bolt (`CreatureOrObject`) with an `ObjectTargetRef` and the host's attack facts | none: resolved | as the case | happy |
| R21 | A synthetic save part with subject `CreatureOrObject` and an `ObjectTargetRef` | `MissingSaveAttempt` (FR-093) | none | unhappy (missing content) |
| R22 | Ice Knife, p. 141 (`creature`) with an `ObjectTargetRef` | `WrongTargetSubject` (FR-094) | none | unhappy (invalid input) |
| R23 | Fire Bolt, the target's `SpellAttackFacts` make `CircumstanceCollector` return `AttackPlanNotPossible`, or its `CoverSources` resolve to Total Cover | `AttackNotPossible` (FR-022); class `refused` (FR-080) | none | unhappy (state error) |
| R24 | Fire Bolt, the target's facts make `CircumstanceCollector` return `AttackPlanAutomaticMiss` | none: resolved; `OnMiss` applies; no d20 requested for that instance (FR-022) | none for that instance | unhappy (rule interaction) |
| R25 | Fire Bolt, target's `CoverSources` resolve to Half Cover | none: resolved; the attack roll is compared with AC plus 2 (FR-022) | as the case | unhappy (rule interaction) |
| R26 | False Life, p. 129 (`TargetsSelf`), empty target list | `WrongTargetCount` (FR-019) | none | unhappy (empty and minimal) |
| R27 | False Life, the target list is a creature other than the caster (or an `ObjectTargetRef`) | `WrongTargetCount` (FR-019) | none | unhappy (invalid input) |
| R28 | False Life, the caster only | none: resolved | as the case | happy |

### Table O: roll order with a fixed roller (FR-041, FR-065, FR-020; one test per row)

| Label | Cast | Dice requests, in order | Path |
|---|---|---|---|
| O1 | Fireball, targets A and B | `8d6`, `1d20` (A), `1d20` (B) | happy |
| O2 | Flame Strike, one target | `5d6` (Fire), `5d6` (Radiant), `1d20` | unhappy (rule interaction) |
| O3 | Eldritch Blast, `ScalingLevel` 5, beams assigned to A then B, both hit | `1d20` (A), `1d10` (A), `1d20` (B), `1d10` (B) | happy |
| O4 | Eldritch Blast, `ScalingLevel` 5, first beam misses | `1d20` (A), `1d20` (B), `1d10` (B) | unhappy (rule interaction) |
| O5 | Magic Missile, level 2, darts assigned A, A, B, B | four requests of `1d4+1`, in dart order; four `PlannedDamage` values (two for A, two for B), so a target at 0 Hit Points takes one failure per dart (FR-065) | happy |
| O6 | Cure Wounds, one target | one request of `2d8` | happy |
| O7 | Hold Person, one target | `1d20` | happy |
| O8 | Fire Bolt, one target, hit | `1d20`, `1d10` | happy |
| O9 | Fire Bolt, one target, Critical Hit | `1d20`, then the doubled damage expression as attacks.md rolls it (its FR-056 to FR-059) | unhappy (boundary) |
| O10 | Eldritch Blast, `ScalingLevel` 5, both beams assigned to A, beam 1 a normal hit, beam 2 a Critical Hit | `1d20` (beam 1), `1d10`, `1d20` (beam 2), then the doubled damage expression as attacks.md rolls it. The plan holds two `PlannedDamage` values for A: one `NormalHit` with `Damage(x, Force)` and one `CriticalHit` with the doubled `Damage`, in instance order (FR-065) | unhappy (rule interaction) |
| O11 | Fireball, no creature in the area | none (FR-020) | unhappy (empty and minimal) |
| O12 | A synthetic spell attack whose `OnHit` damage is `1d8+2`, Critical Hit | `1d20`, then the dice doubled and the flat `+2` once, as the `AttackDamageComposer` states them (FR-023); no second floor or modifier step by the resolver | unhappy (boundary) |

### Withdrawn ids

Moved to `spell-effect-data.md` (shared-types-and-ownership.md FR-059). FR-092 stays here; its `SelfTargetPolicy` field is spell-effect-data.md FR-040.

| Id | Status |
|---|---|
| FR-001 | Withdrawn: moved to spell-effect-data.md (FR-001) |
| FR-002 | Withdrawn: moved to spell-effect-data.md (FR-002) |
| FR-003 | Withdrawn: moved to spell-effect-data.md (FR-003) |
| FR-004 | Withdrawn: moved to spell-effect-data.md (FR-004) |
| FR-005 | Withdrawn: moved to spell-effect-data.md (FR-005) |
| FR-006 | Withdrawn: moved to spell-effect-data.md (FR-006) |
| FR-007 | Withdrawn: moved to spell-effect-data.md (FR-007) |
| FR-008 | Withdrawn: moved to spell-effect-data.md (FR-008) |
| FR-009 | Withdrawn: moved to spell-effect-data.md (FR-009) |
| FR-010 | Withdrawn: moved to spell-effect-data.md (FR-010) |
| FR-011 | Withdrawn: moved to spell-effect-data.md (FR-011) |
| FR-012 | Withdrawn: moved to spell-effect-data.md (FR-012) |
| FR-013 | Withdrawn: moved to spell-effect-data.md (FR-013) |
| FR-014 | Withdrawn: moved to spell-effect-data.md (FR-014) |
| FR-016 | Withdrawn: moved to spell-effect-data.md (FR-015) |
| FR-021 | Withdrawn: moved to spell-effect-data.md (FR-042) |
| FR-027 | Withdrawn: moved to spell-effect-data.md (FR-043) |
| FR-034 | Withdrawn: moved to spell-effect-data.md (FR-045) |
| FR-039 | Withdrawn: moved to spell-effect-data.md (FR-046) |
| FR-044 | Withdrawn: moved to spell-effect-data.md (FR-016) |
| FR-045 | Withdrawn: moved to spell-effect-data.md (FR-017) |
| FR-046 | Withdrawn: moved to spell-effect-data.md (FR-018) |
| FR-047 | Withdrawn: moved to spell-effect-data.md (FR-019) |
| FR-048 | Withdrawn: moved to spell-effect-data.md (FR-020) |
| FR-049 | Withdrawn: moved to spell-effect-data.md (FR-021) |
| FR-051 | Withdrawn: moved to spell-effect-data.md (FR-022) |
| FR-052 | Withdrawn: moved to spell-effect-data.md (FR-023) |
| FR-053 | Withdrawn: moved to spell-effect-data.md (FR-024) |
| FR-066 | Withdrawn: moved to spell-effect-data.md (FR-025) |
| FR-067 | Withdrawn: moved to spell-effect-data.md (FR-026) |
| FR-068 | Withdrawn: moved to spell-effect-data.md (FR-027) |
| FR-069 | Withdrawn: moved to spell-effect-data.md (FR-028) |
| FR-070 | Withdrawn: moved to spell-effect-data.md (FR-029) |
| FR-071 | Withdrawn: moved to spell-effect-data.md (FR-030) |
| FR-072 | Withdrawn: moved to spell-effect-data.md (FR-031) |
| FR-073 | Withdrawn: moved to spell-effect-data.md (FR-032) |
| FR-074 | Withdrawn: moved to spell-effect-data.md (FR-033) |
| FR-075 | Withdrawn: moved to spell-effect-data.md (FR-034) |
| FR-076 | Withdrawn: moved to spell-effect-data.md (FR-035) |
| FR-077 | Withdrawn: moved to spell-effect-data.md (FR-036) |
| FR-078 | Withdrawn: moved to spell-effect-data.md (FR-037) |
| FR-079 | Withdrawn: moved to spell-effect-data.md (FR-038) |
| FR-086 | Withdrawn: moved to spell-effect-data.md (FR-039) |

## 5. Model

Notation: `Name(a, b)` lists the fields of a struct type `Name`; a "closed" type is an interface with an unexported marker method, a doc comment naming every variant, and one struct per variant, switched exhaustively (`errors-and-results.md` FR-018). Constructors are `NewX` returning `(T, error)` with errors satisfying `errors.Is(err, srderr.ErrInvalidInput)`; `MustX` panics with that error. Slices and maps are copied on construction and on read.

The content model is spell-effect-data.md section 5.2; this spec uses it through the exported API of package `content` and holds only the combat-side types.

**Domain** (package `combat`, files `spell_effects*.go`; pure, no `srderr`, `telemetry` or `context` reference):

- **`SpellEffectResolver`** (stateless domain service): given the model, the `CastReport`, the `dice.Result`s of the dice the use case requested and the d20 results (`AttackRollResult`, `CheckResult`, `ForcedFailureResult`), it computes scaled amounts and counts (FR-043, with spell-effect-data.md FR-041), evaluates each `SpellAmount` (modifier, floor at 0, FR-032), builds the `Damage`, healing and Temporary Hit Points amounts, target validity (FR-017), part outcomes (FR-024, FR-025) and the plan (FR-064). The use case only requests dice and passes results on; it holds no rules logic (`doc/rules/architecture.md` §4).
- **Plan types:** `TargetEffects(target, plannedDamage, healing, temporaryHitPoints, conditions, perception)`; `PlannedDamage(partIndex, instanceIndex, damages, adjustment, hitKind)` (FR-065); `ConditionToApply(target, kind, source, duration)` with `ToCondition()` (FR-038); `InvalidTargetPerception` (`AppearedToSucceed`, `SpellDidNothing`).
- **Evidence** (FR-042): `ResolvedPart`, `ResolvedInstance`, holding the d20 results unchanged beside the amounts rolled.
- **Inputs:** `SpellTarget(ref, creatureType, validity, path, saves, attackFacts)` with `ref` a `SpellTargetRef` (FR-093) and `AttackFacts` a `SpellAttackFacts` (FR-097); `SpellObjectID` (`type SpellObjectID string`; `ParseSpellObjectID` returns `(SpellObjectID, error)`); `TargetValidity` (`valid`, `CannotBeAffected`); `PathToTarget` (`clear`, `BlockedByTotalCover`); `InstanceAssignment(partIndex, targets)`; `SpellEffectsRejectionKind` (the reasons of FR-057).

**Use case** (`resolve_spell_effects_usecase.go`): `ResolveSpellEffects`, `ResolveSpellEffectsCommand`, `ResolveSpellEffectsResult` and variants (section 6).

**Reused, not defined here:** `Damage`, `HitKind`, `HalfDamage` (hit-points-and-damage.md); `CircumstanceCollector`, `AttackPlan`, `AttackDamageComposer`, `AttackForm`, `CoverResolver`, `CoverSource`, `AttackRange`, `Reach`, `NearbyEnemy`, `AttackSituation` (attacks.md); `AutomaticCritical`, `Condition` (conditions.md); the pure `d20` rules functions, `Roll`, `DifficultyClass`, `ArmorClass`, `CheckResult`, `ForcedFailureResult`, `AttackRollResult` (d20-tests.md), `SaveAttempt`, `RollAttempt`, `ChooseToFail`, `AutomaticFailure`, `ForcedFailureReason` (resolve-d20-test-use-case.md; the use case itself is not used); `CastReport`, `SpellcastingStats` (spellcasting.md); `CombatantID`, the combat telemetry helper, the `combat` signal constants, `CombatResultClass` (combat-foundations.md).

**Failures:** rejections are a closed result with an enum of reasons in a value object (shared-types-and-ownership.md FR-018); contradictory input is a returned error (`ErrInvalidSpellEffectsCommand`, wrapping `srderr.ErrInvalidInput`); a roller failure is the only port failure and is returned as a `*srderr.PortError`.

## 6. Interface

Synchronous (`dice.Roller` is; telemetry calls do not block). `ctx` is the first parameter because the use case starts a span. Illustrative: the requirements fix behaviour, not spelling.

```go
// resolve_spell_effects_usecase.go
type ResolveSpellEffects struct{ /* roller, telemetry */ }

func NewResolveSpellEffects(roller dice.Roller, opts ...Option) (ResolveSpellEffects, error)
func (u ResolveSpellEffects) Do(ctx context.Context, cmd ResolveSpellEffectsCommand) (ResolveSpellEffectsResult, error)
```


- **`ResolveSpellEffectsCommand`:** the `Spell`; the `CastReport` (casting-and-concentration.md FR-082); the caster's `CombatantID`; the caster's attack `RollModifiers` (spellcasting.md FR-116) and spellcasting ability modifier; the ordered `SpellTarget`s; the `InstanceAssignment`s; the optional `ChosenDamageType`; the optional `PartIndexes` and `InstanceCount` (FR-098). Each `SpellTarget` carries its `SpellTargetRef`, `CreatureType` (creatures only), `TargetValidity`, `PathToTarget`, a `map[ability.Ability]SaveAttempt` (copied on construction and on read), and the `SpellAttackFacts` for an attack part (FR-097, which settles Q7).
- **`ResolveSpellEffectsResult`** (closed): `SpellEffectsResolved(spellID, effectiveLevel, coverage, parts, targets)`; `SpellEffectsTextOnly(spellID, effectiveLevel)`; `SpellEffectsRejected(SpellEffectsRejection)` holding the reasons list (FR-057).
- **Flow:** (1) `TextOnlyEffects` returns `SpellEffectsTextOnly`. (2) Validate (FR-015 to FR-019, FR-031, FR-033, FR-050, FR-057, FR-059); a rejection returns before any die. (3) Per part in order, roll and resolve as FR-041 fixes. (4) Assemble the plan; nothing is applied.
- **Ports:** only `dice.Roller` and `telemetry.Telemetry`, no other use case (FR-096); no new port, double or contract suite. The `content.Source` port is not called: the caller passes the `Spell` (as spellcasting.md does).
- **Exports.** Model types: spell-effect-data.md section 6. Package `combat` exports `ResolveSpellEffects`, `ResolveSpellEffectsCommand`, `ResolveSpellEffectsResult`, `SpellEffectsResolved`, `SpellEffectsTextOnly`, `SpellEffectsRejected`, `SpellEffectsRejection`, `SpellEffectsRejectionKind`, `SpellTarget`, `SpellAttackFacts`, `SpellTargetRef`, `CreatureTargetRef`, `ObjectTargetRef`, `SpellObjectID`, `ParseSpellObjectID`, `MustSpellObjectID`, `ErrBlankSpellObjectID`, `TargetValidity`, `PathToTarget`, `InstanceAssignment`, `TargetEffects`, `PlannedDamage`, `ConditionToApply`, `InvalidTargetPerception`, `ResolvedPart`, `ResolvedInstance`. Signal constants go into the `combat` signal constants. `SpellEffectResolver` stays internal. Every export has a godoc comment starting with its name.
- **Position.** Depends on `combat-foundations`, `hit-points-and-damage`, `conditions`, `attacks` and nothing later; rank 27.

## 7. telemetry.Telemetry

The domain files are silent (pure). `ResolveSpellEffects` emits through the combat telemetry helper (combat-foundations.md FR-035), which owns the span, throughput counter, duration histogram and error counter (FR-080). Shared signals used unchanged: span `srd5e.combat.resolve_spell_effects`, `srd5e.combat.operations`, `srd5e.combat.operation_duration`, `srd5e.combat.errors`; new `srd5e.combat.result` values `resolved`, `text_only`, `rejected_invalid`, `rejected_refused` (FR-080, FR-081).

1. **What can go wrong?** The `dice.Roller` fails (a `*srderr.PortError`); host facts do not fit the spell (wrong count, missing save, unassigned dart); the host treats a `text_only` or `partial` result as complete; a model disagrees with its text (bundle defect, caught offline by spell-effect-data.md FR-033); an invalid command.
2. **How would an operator know?** `srd5e.combat.operations` with `srd5e.combat.operation` = `resolve_spell_effects`, `srd5e.combat.result` and `srd5e.combat.result_class`; the `resolutions` counter adds coverage, so a high `partial` or `text_only` share shows as a content gap. A roller failure is an error span plus `srd5e.combat.errors`; an invalid command is on the span only.
3. **Diagnosis:** the rejection reason (span attribute), effective level, roll kind and outcome, and the result's evidence (FR-042). Spell ids are not attributes.
4. **Counted or timed:** calls by result and class; resolutions by coverage and level; rolls by outcome and ability; targets per call.
5. **Existing instrumentation still correct?** Yes. No other use case is called (FR-096), so `resolve_attack` and `srd5e.d20_test.*` do not count rolls made here and nothing is counted twice. Attack dashboards do not include spell attacks made here. `srd5e.character.*` signals of `CastSpell` are unaffected.
6. **Monitor or alert on:** any non-zero error rate; the `invalid` share (wrong host wiring if above a few percent); the `text_only` plus `partial` share; the rolled-save success share per ability (see the table); a sudden `text_only` or `partial` rise (a bundle or decoder regression, or hosts casting spells the model does not cover); latency on the shared histogram.
7. **Noise?** No log at all: a rejection is an expected outcome (observability.md §7; combat-foundations.md FR-034).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.resolve_spell_effects` (extra attributes via `attributes`, FR-041: also on the shared `srd5e.combat.operations` increment, listed in its Attributes cell) | span | `srd5e.combat.spell_effects.coverage` (`complete`, `partial`, `text_only`); `srd5e.combat.spell_effects.rejection` (first reason in FR-057 order, rejected only) | Path of one call, on the shared span | Any error status |
| `srd5e.combat.spell_effects.resolutions` | counter, `{resolution}` | `srd5e.combat.spell_effects.coverage`; `srd5e.combat.spell_effects.level` (`cantrip`, `1` to `9`, the effective level); `resolved` and `text_only` only | Content-gap measure by coverage and level | `text_only` plus `partial` share rising against the 7-day baseline |
| `srd5e.combat.spell_effects.rolls` | counter, `{roll}` | `srd5e.combat.spell_effects.roll` (`attack`, `save`, `save_forced`); `srd5e.outcome` (`success`, `failure` only; an attack hit, Critical Hit included, is `success`; a forced failure is `failure`); `srd5e.ability` (six abilities, the `d20` constant; saves and forced saves only); no increment for an automatic miss | Aggregate attack and save outcomes of spells, which no `srd5e.d20_test.*` or `resolve_attack` signal counts | Over at least 1000 rolled saves per ability, a success share more than 10 percentage points from the host's baseline for the same DCs (wrong DCs or modifiers, or a roller bug; forced failures are counted apart, FR-082) |
| `srd5e.combat.spell_effects.targets` | histogram, `{target}` | none; `resolved` only | Valid targets per call | p99 above 3 times the 7-day baseline (wrong area membership) |

Entries feed `doc/observability/signals.md`; shared-name rows stay with combat-foundations.md (its FR-037). Attribute keys other than `telemetry.AttrOutcome` and `telemetry.AttrErrorType` are exported constants in `combat/signals.go` (FR-085); renaming any is **BREAKING**.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-015, FR-017, FR-018, FR-019, FR-020, FR-091, FR-092, FR-093, FR-094 | unit and application | Perception rules (table I rows I1 to I5, I8, I9); `SpellObjectID` boundaries (blank: `ParseSpellObjectID` returns `ErrBlankSpellObjectID`, `MustSpellObjectID` panics with it); target count, total cover, empty area, all-invalid targets (R1 to R3, R13, R19, R26 to R28); caster and object targets (R17, R18, R20 to R22). |
| FR-022, FR-023, FR-024, FR-025, FR-026 | unit and application | Plan assembly (S1 to S13); real the pure `d20` rules functions and attack services with `dicetest.FixedRoller`; Fire Bolt hit, miss, Critical Hit, automatic miss, not possible, Half Cover (R23 to R25); a flat modifier on a Critical Hit (O12); the save routine equals `AttackFlow.ResolveSave` (attacks.md FR-133) for equal attempts and dice; forced failures (`ChosenToFail`, `automatic`). |
| FR-028, FR-029, FR-030, FR-031, FR-041, FR-042 | application | Table O rows O1 to O12 assert the exact request sequence; R4, R5, R16; evidence carried unchanged. |
| FR-032, FR-033, FR-035, FR-036, FR-037, FR-038, FR-040 | unit and application | Negative spellcasting modifier (floor at 0; zero healing and Temporary Hit Points omitted); chosen damage type (R8, R9, R11); condition plan; identical result for two spells differing only in `Spell.duration` and `RequiresConcentration`. |
| FR-043, FR-050 | unit and application | Table N rows N1 to N8; error cases through the use case (R14, R15, N8). Scaled amounts are tested in spell-effect-data.md (tables L and C). |
| FR-054 to FR-059 | application | Result variants; text-only consumes no die; every rejection reason and the fixed order (R1 to R10); a recording roller sees no request on rejection; missing save attempt and attack facts. |
| FR-060 to FR-065 | application | Immutability; A roller failure is returned as a `*srderr.PortError` with no partial result; invalid-command errors (R11, R12, R14); equal results for equal sequences; `PlannedDamage` per part, instance and target with the right `HalfDamage` and `HitKind` (O5, O10); no state returned. |
| FR-080 to FR-085 | application | Recording `telemetrytest` doubles through the combat telemetry helper: shared span and counters once per call; `result` and `result_class` per variant and rejection kind (`invalid` versus `refused`, including `AttackNotPossible`); `resolutions` for `resolved` and `text_only` only; roll counter including `save_forced` and `srd5e.ability`; no `rolls` increment for an automatic miss (R24); `coverage` and `rejection` on the span and the `operations` increment only; no roll or targets signal for `text_only` and rejected calls; no `srd5e.combat.events`; error recorded once, no log; no `resolve_attack` or `srd5e.d20_test.*` signal; identical result with the zero-value no-op. |
| FR-087 to FR-090 | architecture | File and import rules (evidence types in the pure `spell_effects*.go` files), export scan for duplicate and generic names (including `TurnOwner` against `TurnHolder`), context-map direction. |
| FR-096, FR-097 | architecture, application | No file of the use case or resolver mentions a use case class (architecture-guardrails.md FR-097 scan); constructor takes only `dice.Roller` and the telemetry option; a recording `telemetrytest` double sees no `srd5e.d20_test.*` signal; attack part with `SpellAttackFacts` (R7, R23 to R25, O8, O9). |
| FR-098, FR-050 | unit and application | `PartIndexes` resolves only the named parts; `InstanceCount` 3 gives three instances for one creature; out-of-range index, empty set, `InstanceCount` alone: `ErrInvalidSpellEffectsCommand`; a cast skips a recurring-rule part; a part needing an absent save DC or attack modifier: `ErrInvalidSpellEffectsCommand`. |
| FR-060 to FR-065 | application | Cancellation (testing.md §4; errors-and-results.md FR-054): `Do` with an already-cancelled context and a roller double that honours `ctx` returns an error satisfying `errors.Is(err, context.Canceled)`, no partial result, and no result variant or failure outcome on the telemetry double. |
| FR-095 | scenario | A flat bonus added to the `Damage` of a `PlannedDamage` before `ApplyDamage` is not doubled on a Critical Hit (host-side riders, settled Q18). |

- **Levels:** unit (`combat/spell_effects*_test.go`, package `combat_test`), application (`combat/resolve_spell_effects_usecase_test.go`), architecture (`internal/archtest`), one scenario. No contract suite (no new port). Model tests are in spell-effect-data.md; withdrawn ids are tested there.
- **Doubles:** `dicetest.FixedRoller` and the recording `telemetrytest` doubles; the real the pure `d20` rules functions and attack services; no use case built or called (FR-096); no mocks of the domain.
- **Scenario (`scenarios/`, package `scenarios_test`, public API only):** a character casts Fireball with `CastSpell` at a level 4 slot (fixture under `testdata/srd/`); `ResolveSpellEffects` resolves two targets, one failed and one successful save; the `PlannedDamage` goes to `ApplyDamage` with `HalfDamage` for the saver; final Hit Points are asserted. A second row casts Fire Bolt at character level 5.
- **Mutation testing (`doc/rules/testing.md` §8):** the `spell_effects*.go` files at score >= 80, the use case file SHOULD be mutated too, using `gomu` run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors. Most valuable: `n` in FR-043, `HalfDamage` marking in FR-025/FR-026, the floor in FR-032.

## 9. Open questions and decisions

Settled by decision: the split (shared-types-and-ownership.md FR-059); rank 27; `ConditionKind` owner (FR-005, FR-046); target-side attack facts (FR-097); `CombatantID` (combat-foundations.md FR-001; objects use `SpellObjectID`, FR-093; magical-effects.md Q-8 stays open for curses); Concentration save DC (casting-and-concentration.md FR-079, shared-types-and-ownership.md FR-011). Data-side questions (Q1, Q12 to Q14, Q16): spell-effect-data.md section 9.

Package decisions where the SRD is silent (changing any is **BREAKING**, flagged for review):

- Q3: the FR-to-part assignment (which ids moved to spell-effect-data.md) is the "Withdrawn ids" table at the end of section 4, binding through shared-types-and-ownership.md FR-059; each row names the new owner id.

- Q2: the resolver returns a plan, not applied state (FR-064); ordering is combat-action-resolution.md's (shared-types-and-ownership.md FR-015, FR-026). Cost: up to four host calls per target.
- Q9: one roll per instance shared by its targets, per hit for attack parts (FR-029, FR-030); the SRD states "roll the damage once for all the targets" only for save damage (p. 16).
- Q10: the floor at 0 also applies to healing and Temporary Hit Points (FR-032).
- Q17: no die is requested when no creature takes the damage (FR-020, FR-028); rejected alternative: roll once to keep roller sequences aligned.

- Q8: one `PlannedDamage` per damage event (part, instance, target), one `ApplyDamage` command each (FR-065). The SRD gives a Death Saving Throw failure for each instance of damage taken at 0 Hit Points (p. 18), and the usual ruling counts each dart or beam as its own instance, so per-dart and per-beam commands give the right count while one roll with several damage types stays one command (hit-points-and-damage.md FR-085, no change needed).
- Q11: durations, repeated saves, delayed damage and riders are out of this spec (section 1); such parts stay text and make a spell `partial` (FR-056). [active-effects.md](active-effects.md) consumes `ConditionDuration` and `Spell.duration` and owns lifecycle and riders (Hex, Hunter's Mark, Bless, Bane); [spell-granted-modifiers.md](spell-granted-modifiers.md) owns the numeric modifiers an active spell grants; and [persistent-spell-zones.md](persistent-spell-zones.md) hands due effects back through FR-098; no other change here.
- Q18: class riders on spell damage are host-side: a flat bonus added to a `PlannedDamage`'s same-type `Damage`, never doubled on a Critical Hit (FR-095), which the class specs already compute. Empowered Spell's reroll uses the dice in the evidence (FR-042). A first-class input is added only if a class spec needs one.

Done: RA-6 (attacks.md FR-115 no longer nests a use case; this spec composes the same domain services and not the internal `AttackFlow`, attacks.md FR-127; the save routine of FR-024 follows `AttackFlow.ResolveSave`, attacks.md FR-133). Data-model change requests are RA-1 to RA-4 of spell-effect-data.md.

## 10. Changelog impact

Under `Unreleased`:

- **Added:** `ResolveSpellEffects` with its command, result and plan types, the `srd5e.combat.spell_effects.*` signals and the `srd5e.combat.result` values `resolved`, `text_only`, `rejected_invalid`, `rejected_refused` (added to `doc/observability/signals.md`). The model and pilot annotations are spell-effect-data.md's.
- **Not BREAKING while unreleased.** After the first release, adding a variant to `ResolveSpellEffectsResult`, changing a modelled spell's result for the same input other than to fix a transcription error (including Q9, Q10, Q17), or renaming a signal or attribute is **BREAKING**.
