# Item use effects (thrown, coated and consumable gear)

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `ability-scores.md`, `observability-ports.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `resolve-d20-test-use-case.md`, `content-equipment.md`, `derived-stats.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `environment-hazards-and-objects.md`, `attacks.md`, `attack-and-help-actions.md`, `magic-item-effects.md` |

## 1. Purpose and scope

Rank 25 (`shared-types-and-ownership.md` 4.11, amendment A-1, made): right of `attacks` and `attack-and-help-actions` (24) and `environment-hazards-and-objects` (23), equal to `initiative-and-actions`, which it does not use. It is the owner of "the owning spec's rule" that `attack-and-help-actions.md` FR-062 and `combat-action-resolution.md` FR-122 leave open for thrown gear, and it closes `toolbox-poison.md` Q-19. It merges the candidates thrown-and-coated-gear, consumable-gear-use and the poison part of item-use effects.

Item-use effects in the Adventuring Gear entries of SRD 5.2.1, pp. 94-99, as pure rules and two small use cases:

- **Thrown gear** that replaces one attack: Acid, Alchemist's Fire, Holy Water, Oil (douse a creature or object) and Net. The save DC (8 + Dexterity modifier + Proficiency Bonus) as a value, target validity (kind, range, sight, Total Cover), the Dexterity save, the damage or condition as data, Burning started, the item consumed.
- **Oil coat** on a creature or object: the 1-minute drying clock and the extra 5 Fire damage.
- **Basic Poison** coating: Bonus Action, one weapon or up to three pieces of ammunition, extra 1d4 Poison damage on Piercing or Slashing damage, potent 1 minute or until delivered.
- **Antitoxin**: Bonus Action, 1 hour of Advantage on saves to avoid or end the Poisoned condition.
- **Potion of Healing**: Bonus Action, drink or administer within 5 feet, 2d4 + 2.
- **Healer's Kit**: ten uses, a Utilize action stabilizes an Unconscious creature at 0 Hit Points with no Wisdom (Medicine) check.

Effects are handed on as data (`Damage`, `BurningState`, a condition kind, a Hit Point transition); the host applies damage with `ApplyDamage` or `ApplyDamageToObject` and conditions with `ApplyConditionToSet`. Action costs are reported as data (`ItemActionCost`), not spent (Q-2).

**Out of scope:**

- Oil "Dousing a Space" (Utilize, a lit 5-foot square, 5 Fire damage on entering or ending a turn, once per turn) and Oil "as fuel" (p. 98): a persistent area with per-creature per-turn tracking; a gap recorded for `rules-glossary-coverage.md` (Q-1, amendment A-4).
- Other Utilize gear (Caltrops, Ball Bearings, Hunting Trap, Rope, Pole, Crowbar, ...), Spell Scroll, and the magic-item rules of `magic-item-rules.md` (a Potion of Healing is a magic item by its own text, p. 99, but needs no Attunement and is consumed; none of those rules applies; only the Common potion's roll constant comes from `magic-item-effects.md`, FR-040).
- Sample Poisons, the Poison types and harvesting (`toolbox-poison.md`); Burning damage and extinguishing (`environment-hazards-and-objects.md` FR-005 to FR-013); the Poisoned condition's effects (`conditions.md`); stabilizing by the Help action (`hit-points-and-damage.md` FR-067 to FR-069).
- Inventory counts, price, weight, crafting (`content-equipment.md`, `equipment-rules.md`): the engine holds no stock; a result names what was consumed and the host decrements.
- Enforcing the action economy (Q-2) and any consumer-specific rule.

## 2. Source

- SRD 5.2.1, "Equipment", "Adventuring Gear": "Acid", "Alchemist's Fire", p. 94; "Antitoxin", p. 96; "Healer's Kit", "Holy Water", p. 97; "Net", "Oil", p. 98; "Poison, Basic", "Potion of Healing", p. 99; table p. 95.
- "Attack [Action]", p. 177 (replacing one attack); "Breaking Objects", pp. 177-178 and "Burning [Hazard]", p. 178; "Damage Types" and "Damage Roll", p. 180; "Poisoned [Condition]", p. 186; "Restrained [Condition]", p. 187; "Stable", p. 188; "Utilize [Action]", p. 191; "Unconscious [Condition]", p. 191.
- "Playing the Game": "Saving Throws", p. 7; "Critical Hits", "Damage Rolls", "Healing", pp. 16-17; "Stabilizing a Character", p. 18; "Cover", p. 15; "They Don't Stack", p. 8.
- "Toolbox": "Poison" (the coating idea only, p. 197).
- Rule files: `doc/rules/architecture.md` §1 to §6; `observability.md` §2, §3; `testing.md` §4, §5, §8.
- No source outside SRD 5.2.1 informs this spec. Where the SRD text does not force a reading, the decision is marked interpretation and justified from SRD text alone (section 9).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| "DC 8 plus your Dexterity modifier and Proficiency Bonus" | `ItemSaveDCOf(AbilityModifier, ProficiencyBonus)` returning `DifficultyClass` | A value (FR-001). |
| thrown Acid, Alchemist's Fire, Holy Water, Oil, Net | `ThrownItemKind` (`ThrownAcid`, `ThrownAlchemistsFire`, `ThrownHolyWater`, `ThrownOil`, `ThrownNet`), `ThrownItemFacts` | Static table, FR-003. Ids `acid`, `alchemists_fire`, `holy_water`, `oil`, `net` (`content-equipment.md` FR-003). |
| "replace one of your attacks" | `ItemActionCost.ReplacesOneAttack`; `ThrownItemRules.Replacement(kind)` | An `AttackReplacement(replace, ReplacementByItem(ItemID))` (`attack-and-help-actions.md` FR-062). |
| "As a Bonus Action", "As a Utilize action" | `ItemActionCost` (`BonusAction`, `UtilizeAction`, `ReplacesOneAttack`) | Data, FR-060. |
| target one creature or object you can see within 20 feet | `ThrownTarget` (closed: `ThrownAtCreature(CombatantID, CreatureSize, CreatureType)`, `ThrownAtObject`), `ThrownSight` (`seen`, `unseen`), `ThrownCover` (`targetable`, `TotalCover`), `DistanceFeet` | Host facts (allegiance, line of sight, `CoverResolver` of `attacks.md` FR-017 to FR-019). |
| (package) outcome of a throw | `ThrownSave` (`ThrownSaveRolled{DC: dc, Result: result}`, `ThrownSaveSkipped{Reason: reason}`), `ThrownEffect` (`ThrownDamageDealt(Damage)`, `ThrownDamageAndBurningStarted(Damage)`, `ThrownOilCovered(OilCoat)`, `ThrownRestrained(NetRestraint)`, `ThrownNoEffect(reason)`), `ItemConsumption` (`ItemConsumed(ItemID, count)`, `ItemNotConsumed`) | `result` is the published `CheckResult` or `ForcedFailureResult` of `d20-tests.md`. |
| "covered in oil", "before the oil dries (after 1 minute)" | `OilCoat(elapsed, state)`; `OilCoatState` (`wet`, `dried`, `BurnedOff`) | FR-016 to FR-018. |
| Net: "Restrained until it escapes", escape check, "Destroying the Net" | `NetRestraint`, `NetEscapeCheck(dc, ability, skill)`, `NetStatistics` | FR-014. |
| Basic Poison: coat, "retains potency for 1 minute or until its damage is dealt" | `BasicPoisonCoating(owner, object, elapsed, state)`, `BasicPoisonObject` (`weapon`, `AmmunitionPiece`), `BasicPoisonState` (`potent`, `expired`, `delivered`) | FR-020 to FR-026. |
| Antitoxin: "Advantage on saving throws to avoid or end the Poisoned condition for 1 hour" | `AntitoxinEffect(drinker, elapsed)`, `PoisonedSaveIntent` (`AvoidPoisoned`, `EndPoisoned`, `OtherSave`) | FR-030 to FR-034. |
| Potion of Healing: "drink it or administer it ... within 5 feet", 2d4 + 2 | `PotionUse` (`DrinkPotion`, `AdministerPotion(DistanceFeet)`) | FR-040 to FR-045. |
| Healer's Kit: "ten uses" | `HealersKit(UsesRemaining)` | FR-050 to FR-054. |

No SRD source (descriptive package names): every `...Rejection`, `...Reason`, `ItemConsumption`, `ThrownSave`, `ThrownEffect`, `ItemUseEvent` and its events, `ItemUseRules`.

## 4. Requirements

Ids are stable. Ids are allocated in blocks; an id absent from this section and not in the Withdrawn ids was never used by this spec.

### 4.1 Save DC, facts and replacement

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `ItemSaveDCOf(DexterityModifier, ProficiencyBonus)` MUST return `DifficultyClass(8 + DexterityModifier.value + ProficiencyBonus.value)` with no cap or floor (table 4.1a). | SRD 5.2.1, "Acid", "Alchemist's Fire", p. 94; "Holy Water", p. 97; "Net", "Oil", p. 98 | happy |
| FR-002 | The Proficiency Bonus MUST always be added: the thrower needs no proficiency, since the item is not a weapon and the text says "Proficiency Bonus" without a condition. A character who lacks it (a monster has its own) is the caller's input. | SRD 5.2.1, "Acid", p. 94 (interpretation, Q-3) | unhappy (rule interaction) |
| FR-003 | `ThrownItemFactsOf(kind)` MUST return exactly: Acid (creature or object; range 20; sight required; Dexterity save; 2d6 Acid); Alchemist's Fire (creature or object; 20; sight; Dexterity; 1d4 Fire and starts burning); Holy Water (creature only; 20; sight; Dexterity; 2d8 Radiant, only to a Fiend or an Undead); Oil (creature or object; 20; sight not required; Dexterity; covered in oil); Net (creature only; range 15; sight; Dexterity; Restrained). `ThrownItemKind` MUST have exactly those five members. | SRD 5.2.1, pp. 94, 97, 98 | happy |
| FR-004 | `ThrownItemRules.Replacement(kind)` MUST return `AttackReplacement(AttackReplacementKind.replace, ReplacementByItem(id))` with the kind's `ItemID`. This spec never records the replacement: the host passes it to `RecordReplacement` (`attack-and-help-actions.md` FR-062) or `ReplaceAttack` (`combat-action-resolution.md` FR-122) and the allowance limit is theirs. | SRD 5.2.1, "Attack [Action]", p. 177; "Acid", p. 94 | happy |
| FR-005 | The ids `acid`, `alchemists_fire`, `holy_water`, `oil`, `net`, `poison_basic`, `antitoxin`, `potion_of_healing`, `healers_kit` MUST each resolve to a bundled equipment entry (data test over `data.BundledContentSource`); this spec carries the numbers, since `content-equipment.md` keeps gear mechanics as verbatim text. | SRD 5.2.1, table p. 95; `content-equipment.md` Out of scope | happy |

Table 4.1a, save DC (`ItemSaveDC`; one test per row; format: Dexterity modifier, Proficiency Bonus, DC): 0, +2 gives 10 (happy); +3, +2 gives 13 (happy); +5, +6 gives 19 (boundary, high); -1, +2 gives 9 (boundary, negative modifier); -5, +2 gives 5 (boundary, no floor); +0, +0 gives 8 (empty and minimal; the Proficiency Bonus is never below 2 for a character, so 0 is the minimal input).

### 4.2 Resolving a thrown item

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-006 | `ResolveThrownItem` MUST check target validity in this order, no die rolled and nothing consumed on a refusal: (1) a `ThrownAtObject` for Holy Water or Net returns `ThrownItemNotPossible(TargetNotACreature)`; (2) `ThrownCover.TotalCover` returns `TotalCover`; (3) `DistanceFeet` above the kind's range returns `BeyondRange` (a distance equal to the range is valid); (4) `ThrownSight.unseen` for a kind that requires sight returns `TargetNotSeen`. | SRD 5.2.1, "Acid", p. 94; "Holy Water", p. 97; "Net", "Oil", p. 98; "Cover", p. 15 (Total Cover "can't be targeted", interpretation Q-4) | unhappy (boundary, rule interaction) |
| FR-007 | A negative `DistanceFeet` MUST make `NewResolveThrownItemCommand` return a typed error (matching `srderr.ErrInvalidInput`) that carries the violated field (`DistanceFeet`) and never panics, per `attacks.md` FR-132; `MustResolveThrownItemCommand` panics with that error. | `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-008 | Oil for a valid target MUST be valid at any `ThrownSight`. For Acid, Alchemist's Fire, Holy Water and Net, `unseen` MUST be refused. | SRD 5.2.1, "Oil", p. 98 (no sight clause) vs pp. 94, 97, 98 | unhappy (rule interaction) |
| FR-009 | For a valid throw the target MUST make a Dexterity saving throw against `ItemSaveDCOf(command dexterity modifier, command Proficiency Bonus)`, resolved through the shared save routine (`attacks.md` FR-133) over the command's `SaveAttempt`, never through `d20.Resolver.Resolve` (`shared-types-and-ownership.md` FR-044). A total at or above the DC succeeds. A `ChooseToFail` or `AutomaticFailure` attempt draws no d20 and fails. The result carries the DC and the published result in `ThrownSaveRolled`. | SRD 5.2.1, "Acid", p. 94; "Saving Throws", p. 7; "D20 Tests", p. 6 | happy; unhappy (boundary) |
| FR-010 | Cover's Dexterity bonus, an Advantage or Disadvantage, and any other modifier on the target's save MUST be supplied inside the `SaveAttempt` by the caller; the use case adds nothing to it. | SRD 5.2.1, "Cover", p. 15; `attacks.md` FR-094 | unhappy (rule interaction) |
| FR-011 | Acid on a failed save MUST return `ThrownDamageDealt(Damage(2d6 total, Acid))`; on a success `ThrownNoEffect(SaveSucceeded)` and no damage roll (no half damage). Holy Water on a failed save by a Fiend or an Undead MUST return `ThrownDamageDealt(Damage(2d8 total, Radiant))`; on a success `ThrownNoEffect(SaveSucceeded)`. The `Damage` is unadjusted: Resistance, Immunity and Temporary Hit Points are `ApplyDamage`'s (`hit-points-and-damage.md` FR-016 to FR-024). | SRD 5.2.1, "Acid", p. 94; "Holy Water", p. 97 ("or take 2d8 Radiant damage if it is a Fiend or an Undead") | happy |
| FR-012 | Holy Water thrown at a creature whose type is neither Fiend nor Undead MUST return `ThrownSaveSkipped(NotFiendOrUndead)` and `ThrownNoEffect(NotFiendOrUndead)` with no d20 and no damage, and MUST still consume the flask. | SRD 5.2.1, "Holy Water", p. 97 (interpretation, Q-6) | unhappy (rule interaction) |
| FR-013 | Alchemist's Fire on a failed save MUST return `ThrownDamageAndBurningStarted(Damage(1d4 total, Fire))`: the host applies the damage and sets `BurningState.Burning` (creature or object); on a success `ThrownNoEffect(SaveSucceeded)`. A target already burning stays burning. Its later 1d4 at each turn start is `RollBurningDamage`'s. | SRD 5.2.1, "Alchemist's Fire", p. 94; "Burning [Hazard]", p. 178 | happy; unhappy (state error) |
| FR-014 | Net on a failed save MUST return `ThrownRestrained(NetRestraint)`: the host applies Restrained; the `NetEscapeCheck` is Strength (Athletics) at DC 10, taken as an action by the target or a creature within 5 feet; `NetStatistics` is Armor Class 10, 5 Hit Points, Immunity to Bludgeoning, Poison and Psychic. A target of size Huge or Gargantuan MUST return `ThrownSaveSkipped(TargetTooLarge)` and `ThrownNoEffect(TargetTooLarge)` with no d20. `NetRestraint.escape(d20.Outcome)` MUST return `NetEscaped` on `success` and `NetHoldsFast` on `failure`; `NetRestraint.destroyed()` MUST return `NetEscaped`. | SRD 5.2.1, "Net", p. 98 ("succeeds automatically if it is Huge or larger"); "Restrained [Condition]", p. 187 | happy; unhappy (boundary) |
| FR-015 | Damage dice MUST be rolled once, only on a failed save, through the `dice.Roller` and `dice.Roll` as one expression (`2d6`, `1d4`, `2d8`), after the save's d20s; a success, a skipped save and a refusal request no damage die. Dice requested: a save with Advantage or Disadvantage `2d20`, otherwise `1d20`, then the damage expression. There is no Critical Hit. | SRD 5.2.1, "Damage Rolls", p. 16; "Critical Hits", p. 16 (an attack roll only) | unhappy (rule interaction) |
| FR-019 | A valid throw MUST report `ItemConsumed(ItemID, 1)` for Acid, Alchemist's Fire, Holy Water and Oil whatever the save, including a skipped save; a refusal and a Net MUST report `ItemNotConsumed` (the Net stays an object; whether it is recoverable is the host's, Q-7). | SRD 5.2.1, pp. 94, 97, 98 (interpretation, Q-7) | unhappy (state error) |

Table 4.2a, thrown outcomes (one test per row; fixed rolls; thrower Dexterity +3, Proficiency Bonus +2, so DC 13; target a Medium Humanoid unless stated):

| Label | Item, target | Save d20 (total) | Expected | Req. | Path |
|---|---|---|---|---|---|
| acid fail | Acid, creature 20 ft, seen | 12 (12 < 13) | `ThrownDamageDealt(2d6 Acid)`, consumed 1 | FR-009, FR-011 | happy |
| acid exact | Acid, creature | 13 (13 = DC) | `ThrownNoEffect(SaveSucceeded)`, no damage die, consumed 1 | FR-009 | unhappy (boundary) |
| acid object | Acid, `ThrownAtObject`, 20 ft | 5 | `ThrownDamageDealt`; the object takes it through `ApplyDamageToObject` | FR-006, FR-011 | happy |
| fire fail | Alchemist's Fire, creature | 3 | `ThrownDamageAndBurningStarted(1d4 Fire)` | FR-013 | happy |
| fire save | Alchemist's Fire | 18 | `ThrownNoEffect(SaveSucceeded)`, still consumed | FR-013, FR-019 | unhappy (boundary) |
| holy fiend | Holy Water, Fiend | 4 | `ThrownDamageDealt(2d8 Radiant)` | FR-011 | happy |
| holy human | Holy Water, Humanoid | none | `ThrownSaveSkipped(NotFiendOrUndead)`, consumed | FR-012 | unhappy (rule interaction) |
| holy object | Holy Water, `ThrownAtObject` | none | `ThrownItemNotPossible(TargetNotACreature)`, not consumed | FR-006 | unhappy (invalid input) |
| oil fail | Oil, creature, `unseen` | 2 | `ThrownOilCovered(OilCoat(0, wet))` | FR-008, FR-016 | happy |
| net fail | Net, Large creature, 15 ft | 6 | `ThrownRestrained`, not consumed | FR-014 | happy |
| net huge | Net, Huge creature | none | `ThrownSaveSkipped(TargetTooLarge)` | FR-014 | unhappy (boundary) |
| net range | Net, 16 ft; Acid, 20 ft; Acid, 21 ft | none; 5; none | `BeyondRange`; valid; `BeyondRange` | FR-006 | unhappy (boundary) |
| total cover | Acid, `TotalCover` | none | `TotalCover`, no die | FR-006 | unhappy (state error) |
| nat 20 fail | Acid, d20 20 with a modifier total below 13 (e.g. -9 modifier, total 11) | 20 (11 < 13) | `ThrownDamageDealt`; no automatic success on a natural 20 | FR-009 | unhappy (boundary) |
| nat 1 success | Acid, d20 1 with a modifier total of 13 or more (+12, total 13) | 1 (13 = DC) | `ThrownNoEffect(SaveSucceeded)`; no automatic failure on a natural 1 | FR-009 | unhappy (boundary) |
| forced fail | Acid, target Paralyzed (`AutomaticFailure`) | none | fails with no d20, `2d6` rolled | FR-009 | unhappy (rule interaction) |
| disadvantage | Acid, `Disadvantage()` | 17 and 9 | `2d20` requested, 9 kept, fails | FR-009, FR-015 | unhappy (rule interaction) |

### 4.3 Oil on a creature or object

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-016 | A failed Oil save MUST produce `OilCoat{Elapsed: 0, State: Wet}`. A new failed save on a target already carrying a `wet` or `dried` coat MUST yield a fresh `OilCoat(zero, wet)` replacing it, never a second coat (one extra 5 Fire damage at most; interpretation, Q-8). `OilCoat.Advance(Duration)` MUST add to `elapsed` and MUST change `wet` to `dried` once `elapsed` is at least 1 minute; a negative `time.Duration` MUST return an error matching `srderr.ErrInvalidInput`; `dried` and `BurnedOff` MUST NOT change on `Advance`. | SRD 5.2.1, "Oil", p. 98 ("before the oil dries (after 1 minute)"; boundary is interpretation, Q-8) | happy; unhappy (boundary) |
| FR-017 | `OilCoat.OnFireDamage(DamageTaken)` on a `wet` coat and `DamageTaken` of 1 or more MUST return `OilIgnited(Damage(5, Fire), coat in BurnedOff)`: the extra 5 Fire damage is a second `Damage` instance, so Resistance and Vulnerability apply to it once. `DamageTaken` is `DamageTaken.total` of the Fire damage (`hit-points-and-damage.md` FR-102), so Temporary Hit Points that absorb it count; Immunity that reduces it to 0 does not. A negative value MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Oil", p. 98 ("If the target takes Fire damage ... an extra 5 Fire damage from burning oil"); `toolbox-poison.md` FR-012 (same `DamageTaken` reading) | happy; unhappy (boundary) |
| FR-018 | `OnFireDamage` on a `dried` or `BurnedOff` coat, or with `DamageTaken` 0, MUST return `OilNotIgnited(reason)` (`dried`, `BurnedOff`, `NoFireDamageTaken`) and an unchanged coat. The extra 5 damage is not itself "Fire damage taken" for a second ignition; a coat burns off once (interpretation, Q-8). | SRD 5.2.1, "Oil", p. 98 | unhappy (state error) |

Table 4.3a, oil coat (one test per row): a second failed Oil save on a `wet` coat at 30 s gives a fresh `OilCoat(0, wet)`; `Advance(59 s)` stays `wet` (boundary); `Advance(60 s)` gives `dried` (boundary); two advances of 30 s give `dried`; Fire damage taken 1 on `wet` gives `OilIgnited(5 Fire)`; Fire damage 0 gives `NoFireDamageTaken`; Fire damage on `dried` gives `dried`; a second Fire damage after ignition gives `BurnedOff`; `Advance(-1 s)` returns an error matching `srderr.ErrInvalidInput`.

### 4.4 Basic Poison coating

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-020 | `BasicPoisonCoating.ApplyToWeapon(owner)` MUST return one `potent` coating (`weapon`, elapsed zero) and `ApplyToAmmunition(owner, pieces)` MUST return `pieces` independent `potent` coatings (`AmmunitionPiece`), for `pieces` 1 to 3; any other count MUST return an error matching `srderr.ErrInvalidInput` that carries the argument name and invalid value as fields. Each application reports `ItemActionCost.BonusAction` and `ItemConsumed(poison_basic, 1)` and emits one `BasicPoisonApplied`. | SRD 5.2.1, "Poison, Basic", p. 99 ("one weapon or up to three pieces of ammunition") | happy; unhappy (boundary) |
| FR-021 | A coating MUST be potent while its state is `potent` and `elapsed` is under 1 minute. `Advance(time.Duration)` MUST add to `elapsed` and move a `potent` coating to `expired` (event `BasicPoisonSpent(expired)`) once `elapsed` is at least 1 minute; a negative `time.Duration` MUST return an error matching `srderr.ErrInvalidInput`. `expired` and `delivered` MUST NOT change on `Advance`. | SRD 5.2.1, "Poison, Basic", p. 99 ("retains potency for 1 minute"; boundary is interpretation, Q-8) | unhappy (boundary) |
| FR-022 | `ExtraDamage(WeaponDamageType)` on a potent coating MUST return `BasicPoisonExtraDamage(DiceDamage(1d4, Poison))` when the weapon's damage type is Piercing or Slashing, and `BasicPoisonNoExtraDamage(NotPiercingOrSlashing)` for any other type. The host adds the component to the attack's `DamageSpec`, so the die is rolled with the attack's damage and doubled on a Critical Hit (`attacks.md` FR-057; Q-5); the coating is not spent by this query. | SRD 5.2.1, "Poison, Basic", p. 99 ("takes an extra 1d4 Poison damage"); "Critical Hits", p. 16 | happy; unhappy (rule interaction) |
| FR-023 | `ExtraDamage` on a coating that is not potent MUST return `BasicPoisonNoExtraDamage(NotPotent)`. | SRD 5.2.1, "Poison, Basic", p. 99 | unhappy (state error) |
| FR-024 | `AfterWeaponDamage(WeaponDamageType, DamageTaken)` on a potent coating MUST return `delivered` (event `BasicPoisonSpent(delivered)`) when the type is Piercing or Slashing and `DamageTaken` is 1 or more, counted as `DamageTaken.total` (Temporary Hit Points that absorb the hit count); otherwise it MUST return the coating unchanged (Bludgeoning, or damage reduced to 0). The host calls it only after a hit; a miss never calls it, so a missed coated weapon or arrow stays potent. A negative `DamageTaken` MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Poison, Basic", p. 99 ("Once applied, the poison retains potency for 1 minute or until its damage is dealt, whichever comes first"); `toolbox-poison.md` FR-012 | unhappy (boundary, rule interaction) |
| FR-025 | `AfterWeaponDamage` and `Advance` on a spent coating (`expired` or `delivered`) MUST return it unchanged with no event. | SRD 5.2.1, "Poison, Basic", p. 99 | unhappy (state error) |
| FR-026 | Coatings MUST be independent values: spending, expiring or advancing one piece of ammunition leaves the others unchanged; the coating holds no weapon identity (the host pairs coating and weapon) and no save, `PoisonType` or Sample Poison (no save: p. 99; `toolbox-poison.md` Q-19). | SRD 5.2.1, "Poison, Basic", p. 99 | unhappy (rule interaction) |

Table 4.4a, Basic Poison (one test per row): weapon, Slashing hit damage taken 4 gives `delivered`, extra `1d4 Poison`; Piercing hit, damage taken 1 gives `delivered` (boundary); Bludgeoning hit leaves `potent`; Piercing hit absorbed whole by Temporary Hit Points (taken 5) gives `delivered`; Piercing damage reduced to 0 leaves `potent`; 3 arrows, one fired and hitting: that piece `delivered`, two `potent`; 4 arrows returns an error matching `srderr.ErrInvalidInput`; 0 arrows likewise; `Advance(59 s)` `potent`, `Advance(60 s)` `expired`; fired after expiry `NotPotent`.

### 4.5 Antitoxin

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-030 | `AntitoxinEffect.drink(drinker)` MUST return an effect with `elapsed` zero, `ItemActionCost.BonusAction`, `ItemConsumed(antitoxin, 1)` and one `AntitoxinDrunk` event. | SRD 5.2.1, "Antitoxin", p. 96 | happy |
| FR-031 | The effect MUST be active while `elapsed` is under 1 hour; `Advance(time.Duration)` MUST add to `elapsed` and, on reaching 1 hour, emit one `AntitoxinWoreOff`; a negative `time.Duration` MUST return an error matching `srderr.ErrInvalidInput`; an expired effect is unchanged by `Advance`. | SRD 5.2.1, "Antitoxin", p. 96 ("for 1 hour"; boundary is interpretation, Q-8) | unhappy (boundary) |
| FR-032 | `AdvantageFor(PoisonedSaveIntent)` MUST return `AntitoxinGivesAdvantage(Advantage())` for `AvoidPoisoned` and `EndPoisoned` while active, and `AntitoxinGivesNothing(reason)` for `OtherSave` (`NotAPoisonedSave`) or when expired (`expired`). The host maps a save to an intent from `PoisonedAtStake` (`toolbox-poison.md` FR-019, prose) or from its own knowledge of any source that gives Poisoned; the Advantage applies to a save against any such source, not only to poison. | SRD 5.2.1, "Antitoxin", p. 96 ("saving throws to avoid or end the Poisoned condition") | happy; unhappy (rule interaction) |
| FR-033 | Drinking while active MUST restart `elapsed` at zero (a fresh vial consumed); the Advantage does not stack with Dwarven Resilience's or any other Advantage (`RollMode`, `modifiers-and-advantage.md` FR-039). | SRD 5.2.1, "Antitoxin", p. 96; "They Don't Stack", p. 8 (interpretation, Q-9) | unhappy (rule interaction) |
| FR-034 | Antitoxin MUST NOT remove, apply or end any condition and gives no immunity: no operation takes or returns a `ConditionSet`, and an existing Poisoned condition stays until ended by its own rule. Disadvantage on a Poisoned creature's attacks and checks is `conditions.md`'s. | SRD 5.2.1, "Antitoxin", p. 96; "Poisoned [Condition]", p. 186 | unhappy (rule interaction) |

Table 4.5a, Antitoxin (one test per row): fresh, `AvoidPoisoned` gives Advantage; `EndPoisoned` gives Advantage; `OtherSave` gives `NotAPoisonedSave`; `Advance(59 min 59 s)` active; `Advance(60 min)` expired, `expired`; drink at 30 min restarts to zero; Advantage plus an Advantage from another source resolves `RollMode.advantage` (not doubled).

### 4.6 Potion of Healing

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | `UsePotionOfHealing` MUST roll exactly one expression, `PotionOfHealingGrade.standard.dice` (`magic-item-effects.md` FR-025, `2d4+2`; one definition, not restated here; dice-and-rolls `dice.Expression`, modifier 2) through `dice.Roll`, giving a healing amount of 4 to 10, call `Restore(subject, amount, ceiling)` on the patient's `HitPointStatus` (`hit-points-and-damage.md` FR-109), and return `PotionOfHealingUsed(status, events, amount, HitPointsRegained, roll, ItemConsumed(potion_of_healing, 1), ItemActionCost.BonusAction)`. | SRD 5.2.1, "Potion of Healing", p. 99; "Healing", p. 17 | happy |
| FR-041 | Whether the user can act (Incapacitated, Unconscious, Dead) is the host's input: this use case does not refuse `DrinkPotion` by a patient at 0 Hit Points. `PotionUse` MUST be `DrinkPotion` (the user is the patient) or `AdministerPotion(DistanceFeet)` (another creature "within 5 feet of yourself"); `DistanceFeet` above 5 MUST return `PotionOfHealingNotPossible(NotWithinFiveFeet)` (5 is valid) and a negative value MUST return an error matching `srderr.ErrInvalidInput`. The command names `user` and `patient` (`CombatantID`); `DrinkPotion` with `user != patient` MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Potion of Healing", p. 99 | unhappy (boundary, invalid input) |
| FR-042 | A Dead patient MUST return `PotionOfHealingNotPossible(PatientIsDead)` before any die is rolled, the potion not consumed and the status unchanged. | SRD 5.2.1, "Dead", p. 180; `hit-points-and-damage.md` FR-039 | unhappy (state error) |
| FR-043 | Healing a patient at 0 Hit Points MUST make it Alive and reset its Death Saving Throw tallies (the transition's own rule), whether the patient was Dying or Stable; an Unconscious patient MAY be administered the potion (the potion's text sets no condition on the patient). | SRD 5.2.1, "Potion of Healing", p. 99; `hit-points-and-damage.md` FR-038, FR-074 | unhappy (rule interaction) |
| FR-044 | A patient at its Hit Point maximum MUST regain 0 and the potion MUST still be consumed; the amount rolled is reported unclamped. | SRD 5.2.1, "Healing", p. 17 (interpretation, Q-10) | unhappy (boundary) |
| FR-045 | The healing MUST be reported through the `HitPointsRestored` event of the transition and `PotionOfHealingConsumed` (one `ItemUseEvent`, FR-072), so the host sees both the Hit Point change and the potion use. | `hit-points-and-damage.md` FR-103; package design | happy |

Table 4.6a, Potion of Healing (one test per row; maximum 20): rolls 1,1 (amount 4) at 14 regains 4, 18; rolls 4,4 (amount 10) at 14 regains 6, 20 (boundary); at 20 regains 0, consumed (boundary); at 0 Dying, rolls 2,2 regains 6, Alive, tallies 0; at 0 Stable regains and is Alive; Dead refused; administer at 5 ft valid, at 6 ft refused; administer to an Unconscious patient valid. Drink while at 0 Hit Points (user = patient, Dying) is allowed by this use case: heals and is Alive (host gates actions).

### 4.7 Healer's Kit

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-050 | `HealersKit(UsesRemaining)` MUST accept 0 to 10 and returns an error matching `srderr.ErrInvalidInput` outside it; `HealersKit.fresh` MUST hold 10; it is an immutable value equal by `UsesRemaining`. No operation adds uses. | SRD 5.2.1, "Healer's Kit", p. 97 ("ten uses") | unhappy (boundary) |
| FR-051 | `UseToStabilize(kit, subject, PatientStatus)` on a Dying patient MUST return `HealersKitUsed(kit with one fewer use, status, events)` where the status is the result of `Stabilize(subject)` (Stable, tallies reset), with `ItemActionCost.UtilizeAction`, no die and no Wisdom (Medicine) check, and one `HealersKitUseSpent` event. | SRD 5.2.1, "Healer's Kit", p. 97; "Stabilizing a Character", p. 18; "Utilize [Action]", p. 191 | happy |
| FR-052 | A kit with `UsesRemaining` 0 MUST return `HealersKitEmpty` before the patient is examined, the patient unchanged. | SRD 5.2.1, "Healer's Kit", p. 97 | unhappy (state error) |
| FR-053 | A patient that is not Dying MUST be refused with the `HitPointRejection` from `Stabilize` (`NotDying` for an Alive or Stable patient, `CreatureIsDead`), and the kit MUST keep its use: a use is expended "to stabilize". | SRD 5.2.1, "Healer's Kit", p. 97; `hit-points-and-damage.md` FR-068 (interpretation, Q-11) | unhappy (state error) |
| FR-054 | The kit MUST NOT need or consume the Help action and MUST NOT grant or spend Help benefits; the `StabilizingCheck` path (`hit-points-and-damage.md` FR-069) stays available when no kit is used. | SRD 5.2.1, "Healer's Kit", p. 97 vs "Stabilizing a Character", p. 18 | unhappy (rule interaction) |

Table 4.7a, Healer's Kit (one test per row): fresh kit, Dying patient gives 9 uses, Stable; 1 use left gives 0 uses; 0 uses gives `HealersKitEmpty`, patient Dying; Stable patient gives `NotDying`, 10 uses; Alive patient gives `NotDying`; Dead patient gives `CreatureIsDead`; kit of 11 or -1 returns an error matching `srderr.ErrInvalidInput`.

### 4.8 Cost, layering, results, events, telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-060 | `ItemUseRules.CostOf(ItemUseKind)` MUST return `ReplacesOneAttack` for the five thrown kinds, `BonusAction` for Basic Poison, Antitoxin and Potion of Healing, and `UtilizeAction` for the Healer's Kit; `ItemUseKind` has exactly those nine members. Each result also carries its cost. | SRD 5.2.1, pp. 94-99 | happy |
| FR-061 | No operation of this spec MUST take a `TurnTracker`, an `ConditionActionAvailability` or a `BonusActionGrant`, or spend an action: the cost is data and the host spends it (`initiative-and-actions.md` FR-050, FR-060). Amendment A-2 (made) added `BonusActionFromItem`. | `shared-types-and-ownership.md` FR-044, FR-048; `toolbox-poison.md` Q-13 | unhappy (forbidden edge) |
| FR-070 | Domain types MUST live in `combat` and be pure and synchronous. They MAY import `combat`'s domain files of ranks 20 to 24, package `content` (`ItemID`, `DamageType`, `CreatureSize`, `CreatureType`), `d20` and `character` (`ProficiencyBonus`), and the kernel packages; they MUST NOT import `toolbox`, `telemetry` or any adapter package (only the `*_usecase.go` files import `telemetry`). | `doc/rules/architecture.md` §1, §2; `combat-foundations.md` FR-014, FR-026 | unhappy (forbidden edge) |
| FR-071 | Exactly two use cases exist, `ResolveThrownItem` and `UsePotionOfHealing`, each built by `NewX(roller Roller, opts ...Option)` (a one-method `Roller` interface that any `dice.Roller` satisfies; `WithTelemetry` option, default zero-value `telemetry.Noop`) with one method `Run(ctx, cmd) (Result, error)`; neither calls, builds or holds a use case, and each composes pure operations and the shared save routine (`attacks.md` FR-133). Coatings, Antitoxin, Oil and the kit are pure domain operations the host calls directly. | `shared-types-and-ownership.md` FR-044; `attacks.md` FR-109 | unhappy (rule interaction) |
| FR-072 | Events MUST form one closed family `ItemUseEvent implements CombatEvent`: `ThrownItemUsed` (`thrown_item_used`; thrower, plus the creature target), `BasicPoisonApplied` (`basic_poison_applied`; owner), `BasicPoisonSpent` (`basic_poison_spent`; owner; cause `delivered` or `expired`), `AntitoxinDrunk` (`antitoxin_drunk`), `AntitoxinWoreOff` (`antitoxin_wore_off`), `PotionOfHealingConsumed` (`potion_of_healing_used`; user, patient), `HealersKitUseSpent` (`healers_kit_used`; user, patient); event class names differ from the result variants `PotionOfHealingUsed` and `HealersKitUsed` so every exported name is unique. They are immutable, past tense, equal by value including `kind` and `combatants`, with a defensively copied duplicate-free list. | `combat-foundations.md` FR-015 to FR-021; `shared-types-and-ownership.md` FR-019 | unhappy (immutability) |
| FR-073 | Results MUST be `ResolveThrownItemResult` (`ThrownItemResolved`, `ThrownItemNotPossible(reason)`) and `UsePotionOfHealingResult` (`PotionOfHealingUsed`, `PotionOfHealingNotPossible(reason)`), reasons as fields (`ThrownItemNotPossibleReason`: `TargetNotACreature`, `TotalCover`, `BeyondRange`, `TargetNotSeen`; `PotionNotPossibleReason`: `NotWithinFiveFeet`, `PatientIsDead`). No generic variant name is exported, and no event shares a name with a result variant. | `combat-foundations.md` FR-022, FR-023 | unhappy (name collision) |
| FR-074 | A `*srderr.PortError` for `dice.Port` MUST be returned unchanged as the error with no partial result; a roll for another expression than requested MUST be returned as `srderr.ErrContractViolation` (a `*srderr.PortError`) by `dice.Roll`. A wrong-size roll never yields a `Damage`. | `dice-and-rolls.md` FR-050; `attacks.md` FR-112 | unhappy (failure of collaborator) |
| FR-075 | No operation MUST alter its inputs; every command, coating, effect, kit and `HitPointStatus` passed in is unchanged after the call, each state change returns a new value, and every collection in a result or event is defensively copied. Two calls with equal inputs and equal fixed rolls MUST give equal results. | `doc/rules/architecture.md` §3 | unhappy (immutability, determinism) |
| FR-076 | Each use case MUST run through `CombatTelemetry` as in section 7, return equal results with the no-op and a recording `telemetry.Telemetry`, and emit no log. The pure operations are silent. | `combat-foundations.md` FR-029 to FR-036; `doc/rules/observability.md` §3 | unhappy (telemetry) |
| FR-077 | Package `combat` MUST export every use case, command, result variant, event and type of sections 3, 5 and 6, each with a godoc comment; no exported name duplicates another in the package. | `doc/rules/architecture.md` §4, §7; `combat-foundations.md` FR-024 | happy |

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

- **Value objects:** `ItemSaveDC`, `ThrownItemFacts`, `ThrownTarget` variants, `ThrownSave`, `ThrownEffect` and `ThrownNoEffectReason` (`SaveSucceeded`, `NotFiendOrUndead`, `TargetTooLarge`), `ItemConsumption`, `ItemActionCost`, `ItemUseKind`, `OilCoat`, `NetRestraint`, `NetEscapeCheck`, `NetStatistics`, `BasicPoisonCoating`, `AntitoxinEffect`, `PoisonedSaveIntent`, `HealersKit`, `PotionUse`. All immutable, equal by value.
- **Domain services** (stateless): `ThrownItemRules` (replacement, validity FR-006), `ItemUseRules` (FR-060). **Entities and aggregates:** none; the host holds one value per coating, effect, coat and kit.
- **Outcome types:** `BasicPoisonExtra` (`BasicPoisonExtraDamage(DiceDamage)`, `BasicPoisonNoExtraDamage(reason)`), `OilIgnition` (`OilIgnited`, `OilNotIgnited`), `AntitoxinAdvantage` (`AntitoxinGivesAdvantage(Advantage)`, `AntitoxinGivesNothing(reason)`), `HealersKitOutcome` (`HealersKitUsed`, `HealersKitEmpty`, `HealersKitRefused(HitPointRejection)`), `NetOutcome` (`NetEscaped`, `NetHoldsFast`). An optional part is a named state, never null.
- **Invariants and failures:** a negative `DistanceFeet`, `time.Duration`, `DamageTaken` or amount, an ammunition count outside 1 to 3 and a kit outside 0 to 10 make the constructor return an error matching `srderr.ErrInvalidInput` (panicking `MustX` forms exist for literals); a rule outcome is a closed result; only a `*srderr.PortError` for `dice.Port` is returned by a port. The two command constructors follow `attacks.md` FR-132.
- **Referenced, not defined here:** `Damage`, `DamageType`, `HitPointStatus`, `HitPointRejection`, `Restore`, `Stabilize` (`hit-points-and-damage.md`); `BurningState`, `ObjectStatus` (`environment-hazards-and-objects.md`); `DiceDamage`, `DamageSpec`, `AttackFlow` (`attacks.md`); `AttackReplacement`, `ReplacementByItem` (`attack-and-help-actions.md`); `SaveAttempt`, `DifficultyClass`, `CheckResult`, `ForcedFailureResult`, `d20.Outcome` (`d20-tests.md`, `resolve-d20-test-use-case.md`); `Advantage`; `CreatureSize`, `CreatureType`, `ItemID`; `CombatantID`, `CombatEvent`.

## 6. Interface

Use cases are synchronous calls taking `ctx` first (they emit spans); the `dice.Roller` port itself is synchronous. Port failure contract: a roller failure is returned as a `*srderr.PortError` for `dice.Port`; expected absence and refusals are closed results.

| Use case | Command | Result | Port |
|---|---|---|---|
| `ResolveThrownItem` | kind, thrower id, Dexterity `AbilityModifier`, `ProficiencyBonus`, `ThrownTarget`, `DistanceFeet`, `ThrownSight`, `ThrownCover`, `SaveAttempt` | `ThrownItemResolved(item, cost, save, effect, consumption, events)` or `ThrownItemNotPossible(reason)` | `dice.Roller` |
| `UsePotionOfHealing` | user, patient, `HitPointStatus`, `PotionUse` | `PotionOfHealingUsed(...)` or `PotionOfHealingNotPossible(reason)` | `dice.Roller` |

Pure operations, exported from `combat`: `ItemSaveDCOf`, `ThrownItemRules`, `ItemUseRules`, `OilCoat.Advance`/`OnFireDamage`, `NetRestraint.Escape`/`Destroyed`, `BasicPoisonCoating.ApplyToWeapon`/`ApplyToAmmunition`/`Advance`/`ExtraDamage`/`AfterWeaponDamage`, `AntitoxinEffect.Drink`/`Advance`/`AdvantageFor`, `HealersKit.UseToStabilize`. No new port, adapter, test double or contract suite; `dicetest.FixedRoller` and `telemetrytest.Recorder` serve. The package exports all of the above (FR-077).

## 7. Observability

The domain is silent by design: pure functions with no port and nothing to measure. The two use cases run through `CombatTelemetry` (FR-076) and record on the shared signals of `combat-foundations.md` (FR-029 to FR-032): spans `srd5e.combat.resolve_thrown_item` and `srd5e.combat.use_potion_of_healing`, with `operations`, `events`, `operation_duration`, `errors`. One extra attribute is added through `CombatTelemetry.run`'s `attributes` (FR-041 there): `srd5e.combat.item_use.item` (`acid`, `alchemists_fire`, `holy_water`, `oil`, `net`, `potion_of_healing`), a `CombatSignals` attribute constant on the span and the `operations` increment only; it is listed in the Attributes cell of the shared `operations` row.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.resolve_thrown_item` | span | `srd5e.combat.operation`, `srd5e.outcome`, `srd5e.combat.result`, `srd5e.combat.result_class`, `srd5e.combat.item_use.item` | One thrown item resolved | Error status only for a returned port failure; slow spans point at the roller |
| `srd5e.combat.use_potion_of_healing` | span | the same, item `potion_of_healing` | One potion used | As above |
| `srd5e.combat.item_use.item` | attribute on the shared `operations` counter | closed set above | Which item drives throughput and refusals | `refused` over 30% for one item over 1 hour with 100 calls suggests the host passes a wrong distance, sight or cover fact |

| Operation | applied results | refused results |
|---|---|---|
| `resolve_thrown_item` | `damaged`, `damaged_and_burning`, `covered_in_oil`, `restrained`, `saved`, `no_effect` | `target_not_a_creature`, `total_cover`, `beyond_range`, `target_not_seen` |
| `use_potion_of_healing` | `healed` | `not_within_five_feet`, `patient_is_dead` |

`saved` is a rolled save that succeeded; `no_effect` is a skipped save (FR-012, FR-014). A refusal records `failure` with no error status; no result is `invalid`.

Evaluation (`doc/rules/observability.md` §2). **Can go wrong:** the roller fails or answers the wrong expression; the host passes wrong target facts (refusals); a wrong DC or damage die (visible in the returned result, not an attribute). **Operator sees:** refusals by `result_class`, port failures on `srd5e.combat.errors`. **Counted or timed:** the shared counters and duration. **Diagnosis:** operation, result name, item; rolls and amounts are in the result, never attributes (`combat-foundations.md` FR-033). **Existing instrumentation:** still correct; `attacks` and hazard signals are untouched because nothing here calls them. **Monitor:** the refused share per item. **Noise:** no log.

## 8. Test plan

Doubles: `dicetest.FixedRoller`, `dicetest.CallbackRoller`, the recording and no-op `telemetry.Telemetry` (the `<pkg>test` packages); real `HitPointStatus`, `SaveAttempt` and `BurningState` values. No new port, double or contract suite.

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002 | unit | Table 4.1a one row per test; Proficiency Bonus present once |
| FR-003 to FR-005 | unit, contract (adapter data) | Facts table exact; five members; replacement value; the nine ids resolve in `data.BundledContentSource` |
| FR-006 to FR-008 | unit, application | Table 4.2a refusal rows; order of checks with two faults at once; boundary 20/21 and 15/16 ft; Oil unseen valid; constructor error parity with `MustResolveThrownItemCommand` |
| FR-009, FR-010, FR-015 | application | Fixed rolls: `1d20`/`2d20` then damage expression; success requests no damage die; forced failure draws no d20; no modifier added by the use case |
| FR-011 to FR-014, FR-019 | application | Table 4.2a rows; Holy Water on Fiend, Undead, Humanoid, object; Net on Huge, Gargantuan, Large; escape and destroy |
| FR-016 to FR-018 | unit | Table 4.3a at 59/60 s, damage 0/1, state sequence, negative inputs |
| FR-020 to FR-026 | unit | Table 4.4a; independence of three arrows; spent coatings unchanged; events |
| FR-030 to FR-034 | unit | Table 4.5a; restart; Advantage not stacked via `RollMode`; no condition parameter (exported-signature scan) |
| FR-040 to FR-045 | application | Table 4.6a; rolls 1,1 and 4,4; refusal before roll (roller never called for Dead); events |
| FR-050 to FR-054 | unit | Table 4.7a with real `HitPointStatus` Dying, Stable, Alive, Dead |
| FR-060, FR-061 | unit, architecture | Cost table; no `TurnTracker`, `ConditionActionAvailability` or `BonusActionGrant` in any signature |
| FR-070, FR-071 | architecture | `internal/archtest` import scan (domain files import no `toolbox`, no `telemetry`); no use case imports another |
| FR-072, FR-073, FR-077 | unit, architecture | One closed event family; `kind`s unique and matching the pattern; `combatants` defensively copied; export list derived from sections 3, 5, 6; no generic variant name |
| FR-074, FR-075 | application | a `*srderr.PortError` for `dice.Port` propagates (same instance); wrong expression gives `srderr.ErrContractViolation`; inputs unchanged; equal inputs equal results |
| FR-076 | application | Span per use case, one `operations`, `operation_duration`, per-event `events`; item attribute on the span and `operations` only; refused is `failure` without error status; equal results with no-op |
| (scenario) | scenario | Public API only: a Fighter's Attack action replaced by Alchemist's Fire (replacement recorded, DC 13, failed save, 1d4 Fire through `ApplyDamage`, Burning, `RollBurningDamage`); Basic Poison on three arrows, one hits (spent) and the other two expire after 1 minute; a Dying ally stabilized by a Healer's Kit then healed by a Potion of Healing |
| errors-and-results.md FR-031, FR-034, FR-051 | application | The typed roller failure of FR-074 is returned unchanged (`errors.As` finds the same `*srderr.PortError`); a recorded failure sets `error.type` from `srderr.ErrorType(err)`; a panicking telemetry double propagates uncounted. |

Mutation testing (`doc/rules/testing.md` §8): all item-use lines of `combat` with `gomu` (score at least 80; run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors, never on the host), watching `<` against `<=` at the 5-foot, range, 1-minute, 1-hour and `DamageTaken` 1 boundaries and the `8 +` in the DC. Every FR id appears above; none is deliberately uncovered.

## 9. Open questions and decisions

Nothing is escalated. **Settled** (one line each; "interpretation" marks a reading the SRD text does not force):

- **Q-1** Oil "Dousing a Space" and "as fuel" are deferred (Out of scope): a persistent area needs a per-creature per-turn model no sibling provides; amendment A-4 records the gap.
- **Q-2** Action costs are data, not spent (FR-060, FR-061): `BonusActionGrant` names a `FeatureID` (`initiative-and-actions.md` FR-060, FR-151) and the SRD's "As a Bonus Action" items grant it by their own text; amendment A-2 (made) gives the item source `BonusActionFromItem`, and the host spends the action meanwhile (as `toolbox-poison.md` Q-13).
- **Q-3** (interpretation) Proficiency Bonus is always added to the item DC (FR-002): the text says "your Dexterity modifier and Proficiency Bonus" with no proficiency clause; matches the Unarmed Strike DC (`attacks.md` FR-074).
- **Q-4** (interpretation) Total Cover refuses a throw (FR-006): the target "can't be targeted directly" (p. 15) and the item says "Target one creature".
- **Q-5** (interpretation) The Basic Poison 1d4 doubles on a Critical Hit (FR-022): it is a die of the weapon's damage with no save between hit and damage, and the SRD doubles "the attack's damage dice" (p. 16); the extra damage is dealt by the attack, not by a save.
- **Q-6** (interpretation) Holy Water thrown at a non-Fiend, non-Undead creature has no save and no damage and uses the flask (FR-012): the damage clause is conditional on type, so the save has nothing to avoid.
- **Q-7** (interpretation) A used flask or vial is consumed whatever the save; a Net is not reported consumed because it persists as an object (FR-019).
- **Q-8** (interpretation) Timed items expire when elapsed time reaches the stated span (oil dries "after 1 minute"; potency "for 1 minute"; "for 1 hour"), so exactly 60 s is expired; the clock is a host-supplied `time.Duration` since nothing in the domain reads time. A coat burns off after one ignition (FR-017, FR-018).
- **Q-9** (interpretation) A second Antitoxin restarts the hour and does not stack (FR-033).
- **Q-10** (interpretation) A potion drunk at full Hit Points is still consumed (FR-044); a Dead patient refuses before use (FR-042).
- **Q-11** (interpretation) A Healer's Kit use is expended only when it stabilizes (FR-053).
- **Q-12** `toolbox-poison.md` Q-19 asked to reuse `PoisonCoating` and `PoisonedAtStake`; `toolbox` is downstream of `combat` and `combat` may not import it (`shared-types-and-ownership.md` FR-013, FR-045), so this spec has its own `BasicPoisonCoating` (no save, one-minute clock, up to three pieces) and takes a `PoisonedSaveIntent` the host derives from `PoisonedAtStake`. Net effect: no `toolbox` edge.
- **Q-13** Net is included although the candidate list omitted it: it is a thrown item that replaces an attack (`attack-and-help-actions.md` FR-062 names it) and `toolbox-traps.md` holds only its statistics by `ItemID`.

**Amendments for other specs:** A-1 (made) `shared-types-and-ownership.md` 4.11 rank 25 lists `item-use-effects`. The host supplies a valid target; self-targeting by the thrower is not refused. A-2 (made) `initiative-and-actions.md` FR-158: `BonusActionFromItem(ItemID)`, for Basic Poison, Antitoxin and Potion of Healing. A-3 `toolbox-poison.md` Q-19 and `equipment-rules.md` (the "no spec yet" line, section 1) point to this spec; `attack-and-help-actions.md` FR-062 and `combat-action-resolution.md` FR-122 replace "the owning spec's rule" with `item-use-effects.md`. A-4 `rules-glossary-coverage.md` records Oil "Dousing a Space" and "as fuel" as an open gap.

## 10. Changelog impact

`CHANGELOG.md`, Unreleased, **Added**: item-use effects in `combat` (`ResolveThrownItem`, `UsePotionOfHealing`, the item save DC, `OilCoat`, `NetRestraint`, `BasicPoisonCoating`, `AntitoxinEffect`, `HealersKit`, the `ItemUseEvent` family) and the `srd5e.combat.item_use.item` attribute with two spans. **Not BREAKING**: new exports only.
