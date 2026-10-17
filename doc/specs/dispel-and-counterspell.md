# Dispel Magic, Counterspell and Antimagic Field

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `d20-tests.md`, `content-schema-and-decoder.md`, `content-spells.md`, `spellcasting.md`, `casting-and-concentration.md`, `combat-foundations.md`, `magical-effects.md`, `spell-effects.md` |
| Consumed by | `combat-action-resolution.md` (the three use cases of section 7, an amendment not made here) |

Rank 28 (`shared-types-and-ownership.md` 4.11): right of `spell-effects` (27) and `magical-effects` (26), level with `encounter-aggregate` (28), which it does not import; left of `combat-action-resolution` (29). Imports (FR-056): the published value types of `d20-tests.md` (`CheckResult`, `ForcedFailureResult`, `d20.Kind`, `DifficultyClass`, `RollModifiers`); `SpellID`, `ConditionKind` (`content-schema-and-decoder.md`); `SpellLevel`, `SpellComponents`, `SpellDuration` (`content-spells.md`); `SlotLevel`, `SpellSaveDC`, `CastSource` (`spellcasting.md`); `d20.Outcome` (`d20-tests.md`); `CombatantID`, `CombatEvent` (`combat-foundations.md`); `OngoingEffect`, `OngoingEffectID`, `SpellSource` (`magical-effects.md`); `SpellObjectID` (`spell-effects.md`). Prose only, not dependencies: `readying-a-spell.md` (see FR-040 here), `conditions.md` (FR-021), `initiative-and-actions.md` (RA-2), `hit-points-and-damage.md`. This spec has no port and returns no `*srderr.PortError`, so `errors-and-results.md` and `observability-ports.md` are not dependencies.

## 1. Purpose and scope

Three SRD spells interact with spells that are already cast or being cast, and nothing in the package models them: Dispel Magic (SRD 5.2.1, p. 124) ends ongoing spells, Counterspell (p. 120) stops a cast, and Antimagic Field (p. 109) switches magic off in an area. `magical-effects.md` holds a creature's `OngoingEffects` but not a spell's level, remaining duration or dispel exceptions; `casting-and-concentration.md` has no "countered" outcome; `spell-effects.md` explicitly defers anything that lasts (its Q11). This spec adds:

1. A **read model** of the live spells: `OngoingSpell` (an `OngoingEffect` of category `spell` plus the level it was cast at, its caster, origin, remaining time and dispel restriction) and the immutable `OngoingSpellRegistry` view (section 4.1). The one live lifetime store is `active-effects.md`; this spec holds no clock and ends spells only through `EndActiveEffects` with `RemovedByEffect` (its FR-044, RA-6).
2. `DispelMagic`: a plan and a conclude step around the ability checks (section 4.2).
3. `Counterspell`: eligibility and the outcome of the Constitution save (section 4.3).
4. `AntimagicField`: which ongoing spells are suppressed and which activities are blocked (section 4.4).

All pure, synchronous domain code over host-held values. The host rolls through `dice.Roller` and the D20 resolver; this spec never rolls (`shared-types-and-ownership.md` FR-044).

**Change requests (not made here).** **RA-1 (`casting-and-concentration.md`):** a pure "countered" transition. The cast must stop before any slot is expended, yet Concentration on another effect is still lost "the moment you start casting" a Concentration spell (SRD p. 179; Q-2). Ask for `ConcentrationRules.CounterCast(SpellcastingState, Spell)`: ends Concentration in progress with reason `anotherEffect` when `Spell.RequiresConcentration`, leaves slots and `slotSpentThisTurn` equal, begins no Concentration, and emits one `CastingCountered` (a `CastingEvent`) after any `ConcentrationEnded`. Until then the host discards the uncommitted `CastSpell` result (FR-034) and calls `BreakConcentration(anotherEffect)` itself. A countered longer cast (casting in progress) is already covered: `BreakConcentration(ended)` clears it with no slot spent (casting-and-concentration.md FR-072). **RA-2 (`initiative-and-actions.md`):** Counterspell is a spell cast with the Reaction; `ReactionChoice` has no variant for it (only `FeatureReaction`, which needs a `FeatureID`). Ask for `SpellReaction{SpellID}` variant with a `SpellReactionRecord`. Until then the host records Counterspell's Reaction as it records any spell cast with the Reaction. **RA-3 (`shared-types-and-ownership.md` 4.11):** add this spec at rank 28. **RA-4 (`combat-action-resolution.md`):** the three use cases and telemetry of section 7.

**Out of scope:** casting itself and slots (`casting-and-concentration.md`); Light and Darkness dispelling each other by overlapping areas, and any "is dispelled" wording inside a spell description (spell text, GM adjudication); ending a lasting magic-item effect "at your discretion" (SRD p. 204); suspending or ending a magic item's properties (Dispel Magic does neither in the SRD); the Hit Points and conditions a spell confers (the host applies `ConditionsToRemove` through `RemoveConditionFromSet`, `conditions.md` FR-117); range, line of sight and area membership (host geometry; `magical-effects.md` for shapes); "Identifying an Ongoing Spell" (p. 106, a Study action); portals and Areas of Effect that "can't extend into the aura" (geometry, FR-050); Spell Scroll and monster spell rules. Nothing core was found missing from the PDF.

## 2. Source

- SRD 5.2.1, "Dispel Magic", p. 124; "Counterspell", p. 120; "Antimagic Field", p. 109.
- "Spells" (Using a Higher-Level Spell Slot, "the spell takes on the higher level for that casting", p. 104; Duration, Combining Spell Effects, p. 106); "Concentration" (Another Concentration Effect), p. 179; "Ready [Action]", p. 187; "Reaction", p. 186; "Exceptions Supersede General Rules", p. 5; "Saving Throws" and "Ability Checks", p. 6; "Subtle Spell", p. 67.
- Exceptions to Dispel Magic: "Forcecage" p. 134, "Guards and Wards" p. 138, "Imprisonment" p. 142, "Prismatic Wall" p. 155, "Wall of Force" p. 172, "Wall of Stone" p. 173.
- Rule files: `doc/rules/architecture.md` §3, §4, §6; `doc/rules/observability.md` §2, §3; `doc/rules/testing.md` §4, §5.
- Web research informed only the interpretations marked in section 9. No outside text is used.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| ongoing spell | `OngoingSpell(effect, CastAtLevel, caster, origin, remaining, restriction, PartOf, conditions, RequiresConcentration)` | `effect` is an `OngoingEffect` of category `spell` (`magical-effects.md` FR-006, FR-007). `CastAtLevel` is a `SpellLevel` (0 to 9). `RequiresConcentration` is set by the host from `Spell.RequiresConcentration`. |
| spell part | `SpellPart` (`PrismaticLayer(1..7)`, `GuardsAndWardsEffect`) | Closed type naming which part of a multi-part spell an entry is (FR-017). Package name. |
| spell's level (when cast higher) | `OngoingSpell.CastAtLevel` | "The spell takes on the higher level for that casting" (p. 104). A cantrip is level 0. |
| ongoing spells held by one target | `OngoingSpellRegistry`, `DispelTarget` (`DispelCreature(CombatantID)`, `DispelObject(SpellObjectID)`, `DispelEffect(OngoingEffectID)`) | "Choose one creature, object, or magical effect" (p. 124). Package names. |
| duration left | `SpellTimeRemaining` (`TimedRemaining(seconds)`, `OpenEnded`) | A round is 6 seconds. `OpenEnded` covers `untilDispelled`, `untilDispelledOrTriggered`, `special`. |
| Artifact or a deity | `SpellOrigin` (`ordinary`, `artifactOrDeity`) | Antimagic Field's exception (p. 109). |
| "can't be dispelled" | `DispelRestriction` (`Unrestricted`, `NeverDispellable`, `RequiresSlotLevel(SlotLevel)`, `AfterPartsGone([]OngoingEffectID)`), `DispelRestrictions.ForSpell(SpellID, ForSpellOptions{Permanent, Part, Siblings})` | Spell-specific exceptions (section 4.2, table T-R). |
| Dispel Magic | `DispelMagic`, `DispelMagicPlan`, `DispelStep` (`DispelEnds`, `DispelNeedsCheck`, `DispelBlocked`), `DispelBlockReason`, `DispelMagicConclusion` | Sections 4.2, 6. |
| Counterspell | `Counterspell`, `CastObservation`, `UsedComponents`, `CounterspellEligible`, `CounterspellIneligible(CounterspellRefusal)`, `CounterspellSave` (`CounterspellRolledSave`, `CounterspellForcedFailure`), `WastedActionCost` | Sections 4.3, 6. |
| Antimagic Field | `AntimagicField`, `SpellFunctioning` (`functioning`, `suppressed`), `AntimagicActivity`, `AntimagicVerdict` (`AntimagicPermits`, `AntimagicBlocks(AntimagicBlock)`) | Section 4.4. |
| ability check using your spellcasting ability | `CheckResult` of kind `abilityCheck` | `d20-tests.md`; the modifier is the spellcasting ability modifier only. |

Terms use SRD 5.2.1 wording (`doc/rules/architecture.md` §3). Package names with no SRD source: every type above except the spell names.

## 4. Requirements

Ids are stable. Case-table rows are one test each, citing the FR in the row.

### 4.1 Ongoing spells and the registry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `OngoingSpell` MUST be an immutable value object equal by value over all fields. Its `effect` MUST be an `OngoingEffect` of category `spell` whose source is a `SpellSource`, else an `srderr.ErrInvalidInput` error (it reuses `magical-effects.md` FR-007). | SRD 5.2.1, "Spell", p. 188; `doc/rules/architecture.md` §3 | happy; unhappy (invalid input) |
| FR-002 | `OngoingSpell.CastAtLevel` MUST be the level of the slot the spell was cast with when that is higher than the spell's own level, and the spell's own level otherwise (a Ritual, a cantrip or a feature-granted cast uses the spell's own level; a cantrip is 0). The host sets it from `CastReport.CastAtLevel`; this spec never recomputes it. | SRD 5.2.1, "Using a Higher-Level Spell Slot", p. 104 | happy; unhappy (boundary) |
| FR-003 | `TimedRemaining(seconds)` MUST require at least 1 second (an `srderr.ErrInvalidInput` error otherwise); `OpenEnded` has no number. `SpellTimeRemaining.FromDuration(SpellDuration)` MUST convert a `SpellDurationTimed` to seconds (rounds x 6, minutes x 60, hours x 3600, days x 86400, using the amount, and for `upTo`/`concentrationUpTo` the maximum) and `untilDispelled`, `untilDispelledOrTriggered` and `special` to `OpenEnded`. `SpellDurationInstantaneous` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`: the effect happens once, so it is never an ongoing spell and cannot be registered. | SRD 5.2.1, "Duration" (Instantaneous), p. 106 | happy; unhappy (boundary); unhappy (invalid input) |
| FR-004 | A part MUST name its parent through `PartOf`; a spell MUST NOT name itself (an `srderr.ErrInvalidInput` error). `conditions` is the set of `ConditionKind`s the spell currently imposes on its holder, empty by default and unmodifiable. `RequiresConcentration` (bool) is part of the value and of equality; a Concentration spell SHOULD carry its `caster`. | `doc/rules/architecture.md` §3 | unhappy (invalid input) |
| FR-005 | `OngoingSpellRegistry` MUST be an immutable read model, built by the host from `ActiveEffects` (`active-effects.md`), mapping each `DispelTarget` to an ordered, unmodifiable list of `OngoingSpell`s. It MAY be empty and MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when two spells anywhere in it share an `OngoingEffectID`. It is not a lifetime store. | `doc/rules/testing.md` §4 (empty, immutability); `shared-types-and-ownership.md` FR-048; `active-effects.md` §1 | unhappy (invalid input) |
| FR-006 | `spellsOn(target)` MUST return the target's spells in registry order, and an empty list for a target the registry does not hold. The registry has no `with`, `without` or `advanced`: adding, ending and ageing a spell is `active-effects.md`'s, and the host rebuilds the view after it. | `doc/rules/architecture.md` §4 | unhappy (immutability); unhappy (empty and minimal) |
| FR-008 | A spell that Antimagic Field suppresses keeps losing time exactly like a functioning one; nothing here reads suppression to stop the clock, and the host MUST advance it through `AdvanceActiveEffects` regardless ("the time it spends suppressed counts against its duration"). | SRD 5.2.1, "Antimagic Field", p. 109 | unhappy (rule interaction) |
| FR-009 | When a removed entry has `PartOf` naming a parent still in the registry on the same target and no other entry shares that parent, `DispelMagic.Conclude` (FR-024) MUST also end the parent with cause `allPartsEnded`; the same on expiry is `active-effects.md`'s. | SRD 5.2.1, "Guards and Wards" ("If all four are dispelled, Guards and Wards ends"), p. 138 | unhappy (rule interaction) |
| FR-010 | Combining spells MUST be left to the host: the registry holds each casting as its own entry, and holds two castings of the same spell as two entries. Which one applies is "Combining Spell Effects" (p. 106), not decided here. | SRD 5.2.1, "Combining Spell Effects", p. 106 | unhappy (rule interaction) |

### 4.2 Dispel Magic

`DispelMagic.Plan(registry, target, castLevel, abilityModifier, modifiers)` decides each spell on the target; the use case rolls one ability check per `DispelNeedsCheck` (which carries `abilityModifier` and `modifiers`); `DispelMagic.Conclude(plan, checks)` applies them. `castLevel` is the `SlotLevel` the Dispel Magic was cast with (3 to 9). `modifiers` is a `RollModifiers` (default none).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-011 | `Plan` MUST consider every spell the registry holds for the target and no other, in registry order, and MUST return one `DispelStep` per spell. A target with no spells MUST return an empty plan. | SRD 5.2.1, "Dispel Magic" ("Any ongoing spell ... on the target"), p. 124 | happy; unhappy (empty and minimal) |
| FR-012 | A spell whose `CastAtLevel` is 3 or lower, with no restriction blocking it, MUST be `DispelEnds` whatever the cast level (table T-D). | SRD 5.2.1, "Dispel Magic", p. 124 | happy |
| FR-013 | A spell of level 4 or higher with no restriction blocking it MUST be `DispelEnds` when its level is equal to or less than `castLevel`, and `DispelNeedsCheck` otherwise (table T-D). | SRD 5.2.1, "Dispel Magic" ("Using a Higher-Level Spell Slot"), p. 124 | happy; unhappy (boundary) |
| FR-014 | `DispelNeedsCheck.Dc` MUST be 10 plus the spell's `CastAtLevel` as a `DifficultyClass` (14 for level 4 up to 19 for level 9). | SRD 5.2.1, "Dispel Magic" ("DC 10 plus that spell's level"), p. 124 | happy; unhappy (boundary) |
| FR-015 | `castLevel` below 3 MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (name `castLevel`): Dispel Magic is a level 3 spell and cannot be cast lower. A monster's or item's at-will cast uses level 3. | SRD 5.2.1, "Dispel Magic" (Level 3), p. 124 | unhappy (invalid input) |
| FR-016 | A spell's `DispelRestriction` MUST take precedence over FR-012 and FR-013: `NeverDispellable` gives `DispelBlocked(neverDispellable)`; `RequiresSlotLevel(n)` gives `DispelBlocked(slotLevelTooLow)` when `castLevel` is below n, and otherwise follows FR-013; `AfterPartsGone(ids)` gives `DispelBlocked(partsRemain)` while any listed id is still in the registry on that target, and otherwise follows FR-013. A blocked spell is never given a check. | SRD 5.2.1, "Exceptions Supersede General Rules", p. 5; spell entries pp. 134, 138, 142, 155, 172, 173 | unhappy (rule interaction) |
| FR-017 | `DispelRestrictions.ForSpell(SpellID, ForSpellOptions{Permanent bool, Part SpellPart, Siblings []OngoingEffectID})` MUST give the restriction of table T-R and `Unrestricted` for any other id. `Part` names which part of a multi-part spell the entry is (the zero `SpellPart` for a whole spell or a parent); `Siblings` is the id slice the host builds from the registry (the other parts of the same parent) and is used only for the violet layer. Wall of Stone depends only on `Permanent`. `PrismaticLayer` outside 1 to 7 MUST make `ForSpell` return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (the `PrismaticLayer` constructor itself is `NewPrismaticLayer(n) (PrismaticLayer, error)` with the same rule). | SRD 5.2.1, as table T-R | happy; unhappy (invalid input: unknown id, layer 0 or 8) |
| FR-018 | The Antimagic Field spell MUST be a registry entry on its caster, `NeverDispellable`, so `Plan` returns `DispelBlocked(neverDispellable)` for it (a second field's too). | SRD 5.2.1, "Antimagic Field" ("Dispel Magic has no effect on the aura"), p. 109 | unhappy (rule interaction) |
| FR-019 | A target the Antimagic Field covers MUST NOT be planned for: `Plan` takes no aura fact. The host asks `AntimagicField.Blocks` (FR-045) with the cast first and does not call `Plan` when blocked. | SRD 5.2.1, "Antimagic Field" ("can't target or otherwise affect anything inside it"), p. 109 | unhappy (rule interaction) |
| FR-020 | `abilityModifier` MUST be the spellcasting ability modifier and nothing else: the Proficiency Bonus is NOT added (the use case must not use `SpellcastingStats.AttackRollModifiers`). The check is a plain ability check; a natural 20 or 1 has no special effect. Extra `RollModifiers` the host holds (for example a die from another feature) are the `modifiers` argument of `Plan`, carried on `DispelNeedsCheck` and applied by the use case. | SRD 5.2.1, "Dispel Magic" ("an ability check using your spellcasting ability"), p. 124; "Ability Checks", p. 6 | unhappy (rule interaction) |
| FR-021 | `conclude(plan, checks)` MUST take a map from `OngoingEffectID` to the `CheckResult` the use case resolved through `d20.ResolveCheck` for each `DispelNeedsCheck`. A `d20.Success` ends the spell; a `failure` leaves it. A total equal to the DC succeeds. | SRD 5.2.1, "Dispel Magic" ("On a successful check, the spell ends"), p. 124; "Compare the Total to a Target Number", p. 6 | happy; unhappy (boundary) |
| FR-022 | `Conclude` MUST return `DispelMagicInvalid(problem)` and change nothing when: a needed check is missing (`missingCheck`); a check names an id the plan did not ask for (`unexpectedCheck`); a check's `target` differs from the step's DC (`checkTargetMismatch`); or a check's kind is not `abilityCheck` (`notAnAbilityCheck`); or FR-059 (`planMismatch`, checked first). The first problem in plan order is reported. | `doc/rules/architecture.md` §6; `magical-effects.md` FR-144 | unhappy (invalid input) |
| FR-023 | `Conclude` MUST return `DispelMagicApplied(registry, ended, survived, events)`: `ended` holds the `DispelEnds` spells and the spells whose check succeeded, in plan order; `survived` the checked spells that failed, then the blocked ones. With nothing ended the result is still `DispelMagicApplied` with an equal registry and no events: the spell was cast and nothing changed. | SRD 5.2.1, "Dispel Magic", p. 124 | happy; unhappy (rule interaction) |
| FR-024 | Each ended spell MUST be removed from the registry and produce one `OngoingSpellEnded` (cause `dispelled`), then, for a part (FR-009), the parent with cause `allPartsEnded` directly after its last part. Order of events: plan order, each parent after its last part. | SRD 5.2.1, "Guards and Wards", p. 138 | happy; unhappy (rule interaction) |
| FR-025 | `OngoingSpellEnded` MUST carry the target, the `OngoingEffectID`, the `SpellID`, the `CastAtLevel`, the cause, the spell's `conditions` as `ConditionsToRemove`, and `EndsCasterConcentration` (true when `RequiresConcentration` is true and `caster` is known). It removes no condition and ends no Concentration itself. | SRD 5.2.1, "Concentration" ("If the effect's creator loses Concentration, the effect ends"), p. 179 | happy |
| FR-026 | When `EndsCasterConcentration` is true the host MUST call `BreakConcentration(ended)` for the caster (`casting-and-concentration.md` FR-119) once, however many spells of that caster ended; this spec imports nothing of `character`. Interpretation (Q-3): nothing is left to concentrate on. | SRD 5.2.1, "Concentration", p. 179 | unhappy (rule interaction) |
| FR-027 | `Plan` and `Conclude` MUST NOT change their inputs, MUST return equal results for equal inputs, and every list in a result MUST be unmodifiable. | `doc/rules/testing.md` §4 (determinism, immutability) | unhappy (immutability) |
| FR-028 | Several spells from different casters on one target MUST each be planned separately: one check each, none shared. | SRD 5.2.1, "Dispel Magic" ("For each ongoing spell"), p. 124 | unhappy (rule interaction) |
| FR-029 | A creature that is the caster of a Concentration spell on another target is NOT a "spell on the target": only spells whose registry holder is the chosen target are planned. | SRD 5.2.1, "Dispel Magic" ("on the target"), p. 124 | unhappy (boundary) |
| FR-058 | A magical effect that is not a creature or object (a wall, a zone) MUST be keyed by `DispelEffect(its OngoingEffectID)` in the registry; a spell on a creature or object by `DispelCreature` or `DispelObject`. The host picks the key from what the player chose. Interpretation: "creature, object, or magical effect" (p. 124) names three kinds of target, and the registry key is the target. | SRD 5.2.1, "Dispel Magic", p. 124 | unhappy (boundary) |
| FR-059 | `Conclude` MUST return `DispelMagicInvalid(planMismatch)` when the plan carries a registry identity (target and the id set of its steps) that the given registry does not hold. The plan stores the target and its step ids, not the registry. | `doc/rules/testing.md` §4 (invalid input) | unhappy (invalid input) |
| FR-060 | `Imprisonment` keeps `RequiresSlotLevel(9)` and the SRD's further condition that the check target "the prison or the component" is a fact about the chosen target, which the host holds: the spec does not model it. Interpretation: no registry key distinguishes prison from component and the SRD gives no different number. | SRD 5.2.1, "Imprisonment", p. 142 | n/a (documented exclusion) |
| FR-030 | A held readied spell is not an ongoing spell and MUST NOT be in the registry: it has no target yet, and only Concentration or the Ready action's expiry end it (`readying-a-spell.md` FR-010, FR-011). | SRD 5.2.1, "Ready [Action]", p. 187 | unhappy (rule interaction) |

#### Table T-D: the plan for one spell (FR-012 to FR-016; one test per row)

| Case | `castLevel` | Spell `CastAtLevel` | Restriction | Expected | Source | Path |
|---|---|---|---|---|---|---|
| cantrip with a duration | 3 | 0 | none | `DispelEnds` | p. 124 | happy |
| level 1 | 3 | 1 | none | `DispelEnds` | p. 124 | happy |
| level 3 at the threshold | 3 | 3 | none | `DispelEnds` | p. 124 | unhappy (boundary) |
| level 4, slot 3 | 3 | 4 | none | `DispelNeedsCheck` DC 14 | p. 124 | unhappy (boundary) |
| level 5, slot 4 | 4 | 5 | none | `DispelNeedsCheck` DC 15 | p. 124 | happy |
| level 4, slot 4 | 4 | 4 | none | `DispelEnds` | p. 124 | unhappy (boundary) |
| level 5, slot 5 | 5 | 5 | none | `DispelEnds` | p. 124 | unhappy (boundary) |
| level 6, slot 5 | 5 | 6 | none | `DispelNeedsCheck` DC 16 | p. 124 | happy |
| level 9, slot 8 | 8 | 9 | none | `DispelNeedsCheck` DC 19 | p. 124 | unhappy (boundary) |
| level 9, slot 9 | 9 | 9 | none | `DispelEnds` | p. 124 | unhappy (boundary) |
| level 2 spell upcast to 5, slot 3 | 3 | 5 | none | `DispelNeedsCheck` DC 15 | p. 104, p. 124 | unhappy (rule interaction) |
| level 1 cast with a slot of 4, slot 3 | 3 | 4 | none | `DispelNeedsCheck` DC 14 | p. 104 | unhappy (rule interaction) |
| level 2, `NeverDispellable` | 9 | 2 | never | `DispelBlocked(neverDispellable)` | p. 5 | unhappy (rule interaction) |
| level 9, `RequiresSlotLevel(9)`, slot 8 | 8 | 9 | slot 9 | `DispelBlocked(slotLevelTooLow)` | p. 142 | unhappy (boundary) |
| level 9, `RequiresSlotLevel(9)`, slot 9 | 9 | 9 | slot 9 | `DispelEnds` | p. 142 | happy |
| Antimagic Field spell, level 8 | 9 | 8 | never | `DispelBlocked(neverDispellable)` | p. 109 | unhappy (rule interaction) |
| `AfterPartsGone({a})`, a present | 5 | 2 | after a | `DispelBlocked(partsRemain)` | p. 155 | unhappy (rule interaction) |
| `AfterPartsGone({a})`, a absent | 5 | 2 | after a | `DispelEnds` | p. 155 | happy |

#### Table T-R: `DispelRestrictions.ForSpell` (FR-017)

| `SpellID` and part | Restriction | Source | Path |
|---|---|---|---|
| `forcecage` | `NeverDispellable` | "Forcecage", p. 134 | happy |
| `wall_of_force` | `NeverDispellable` | "Wall of Force", p. 172 | happy |
| `wall_of_stone`, `Permanent` true | `NeverDispellable` | "Wall of Stone", p. 173 | happy |
| `wall_of_stone`, `Permanent` false | `Unrestricted` | "Wall of Stone" ("Otherwise, the wall disappears when the spell ends"), p. 173 | unhappy (boundary) |
| `antimagic_field` | `NeverDispellable` | "Antimagic Field", p. 109 | happy |
| `guards_and_wards`, `Part` zero (the parent) | `NeverDispellable` | "Guards and Wards" ("no effect on Guards and Wards itself"), p. 138 | happy |
| `guards_and_wards`, `GuardsAndWardsEffect` | `Unrestricted` (each of the four effects "can be dispelled"; the parent ends by FR-009) | "Guards and Wards", p. 138 | unhappy (rule interaction) |
| `imprisonment` | `RequiresSlotLevel(9)` | "Imprisonment", p. 142 | happy |
| `prismatic_wall`, `Part` zero (the parent) | `NeverDispellable`; the parent is the registry entry that FR-009 removes when its last layer is gone | "Prismatic Wall" ("Dispel Magic can affect only the violet layer"), p. 155 | unhappy (rule interaction) |
| `prismatic_wall`, `PrismaticLayer(1..6)` (red to indigo) | `NeverDispellable` | "Prismatic Wall", p. 155 | happy |
| `prismatic_wall`, `PrismaticLayer(7)` (violet), `Siblings` S | `AfterPartsGone(S)` (interpretation: layers fall "in order from red to violet", so violet is reachable only when the others are gone; Q-8) | "Prismatic Wall", p. 155 | unhappy (rule interaction) |
| `prismatic_wall`, `PrismaticLayer(8)` | an `srderr.ErrInvalidInput` error | n/a | unhappy (invalid input) |
| `fireball`, `no_such_spell` | `Unrestricted` | n/a | unhappy (invalid input) |

### 4.3 Counterspell

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | `Counterspell.Assess(CastObservation, counterspeller, distanceFeet, seesCaster, SpellSaveDC)` MUST return `CounterspellEligible` (which carries the `SpellSaveDC` as the save target) only when the counterspeller sees the casting creature, the creature is within 60 feet, and the spell uses at least one Verbal, Somatic or Material component (table T-C). Otherwise it MUST return `CounterspellIneligible` with the first reason in the order `casterNotSeen`, `outOfRange`, `noComponents`. | SRD 5.2.1, "Counterspell" (Casting Time, Range), p. 120 | happy; unhappy (boundary) |
| FR-032 | `UsedComponents(verbal, somatic, material)` MUST be the components the cast actually uses, not the spell's printed ones: a spell cast without components (for example by Subtle Spell, which keeps consumed or costed Material components) is `noComponents`. The host supplies it. | SRD 5.2.1, "Counterspell", p. 120; "Subtle Spell", p. 67 | unhappy (rule interaction) |
| FR-033 | The save MUST be the casting creature's Constitution saving throw against the counterspeller's spell save DC. `Counterspell.Conclude(eligible, save)` takes a `CounterspellSave`: `CounterspellRolledSave(CheckResult)` (kind `savingThrow`, resolved through `d20.ResolveCheck` by the use case, target equal to the DC `CounterspellEligible` carries) or `CounterspellForcedFailure(ForcedFailureResult)`. | SRD 5.2.1, "Counterspell" ("The creature makes a Constitution saving throw"), p. 120; "Saving Throws", p. 6 | happy |
| FR-034 | A failed save (or a forced failure) MUST return `CounterspellCountered` and a success `CounterspellResisted`. On `CounterspellCountered` the host MUST discard the uncommitted `CastSpell` result: no slot is expended, no `SpellCast`, no effect, and `slotSpentThisTurn` is unchanged. The result carries `slotNotExpended` (true when the `CastSource` is `slot`), `priorConcentrationEnds` (the spell requires Concentration, FR-036) and `wasted`. | SRD 5.2.1, "Counterspell" ("the spell dissipates with no effect"), p. 120 | happy |
| FR-035 | `wasted` MUST be the action cost of the cast: `magicAction`, `bonusAction` or `reaction` for those casting times, and none for a casting of one minute or longer, which the host ends with `BreakConcentration(ended)`. The cost is "wasted": this spec does not refund it, and `initiative-and-actions.md` already recorded it. | SRD 5.2.1, "Counterspell" ("the action, Bonus Action, or Reaction used to cast it is wasted"), p. 120 | unhappy (rule interaction) |
| FR-036 | `priorConcentrationEnds` MUST be true exactly when the countered spell requires Concentration: Concentration on another effect is lost "the moment you start casting", and a countered cast still started. It is false for any other spell. Interpretation (Q-2). | SRD 5.2.1, "Concentration" (Another Concentration Effect), p. 179 | unhappy (rule interaction) |
| FR-037 | Only the slot is restored: a countered cast with `CastSource` `cantrip`, `ritual` or `featureGranted` MUST report `slotNotExpended` false, and nothing here refunds a feature's daily use (the feature owner's state). Interpretation (Q-4). | SRD 5.2.1, "Counterspell" ("If that spell was cast with a spell slot, the slot isn't expended"), p. 120 | unhappy (rule interaction) |
| FR-038 | Casting Counterspell MUST be the host's ordinary `CastSpell` of a level 3 or higher slot (or a feature-granted cast) with `triggerMet` true (`casting-and-concentration.md` FR-069, FR-082); a higher slot changes nothing: the spell has no higher-level text. | SRD 5.2.1, "Counterspell", p. 120 | unhappy (rule interaction) |
| FR-039 | A Counterspell cast in response to a Counterspell MUST be resolved first (its target is the first Counterspell's cast). If the first Counterspell is countered it is not cast, so the original spell is not countered by it. The host orders the calls; each is its own pure conclude. Interpretation (Q-5). | SRD 5.2.1, "Counterspell" (Reaction, Components: S), p. 120 | unhappy (rule interaction) |
| FR-040 | A spell readied with the Ready action is countered, if at all, at the moment the Ready action casts it; releasing the held spell with the Reaction is not a casting and `Counterspell.Assess` MUST NOT be called for it. Interpretation (Q-6). | SRD 5.2.1, "Ready [Action]" ("you cast the spell as normal ... and hold its energy"), p. 187 | unhappy (rule interaction) |
| FR-041 | `Conclude` MUST return `CounterspellInvalid(problem)` when the rolled save's `kind` is not `savingThrow` (`notASavingThrow`) or its target differs from the eligible DC (`saveTargetMismatch`). | `doc/rules/architecture.md` §6; `magical-effects.md` FR-144 | unhappy (invalid input) |
| FR-042 | `CounterspellCountered` and `CounterspellResisted` MUST each produce one event (`SpellCountered`, `CounterspellResisted`) carrying the counterspeller, the casting creature and the `SpellID`; `Assess`, `Conclude` MUST be deterministic and mutate nothing. | `doc/rules/testing.md` §4 | happy; unhappy (determinism) |

#### Table T-C: eligibility (FR-031, FR-032; one test per row)

| Case | Sees caster | Distance (ft) | Components used | Expected | Path |
|---|---|---|---|---|---|
| all three present | yes | 30 | V, S, M | `CounterspellEligible` | happy |
| only Verbal | yes | 60 | V | `CounterspellEligible` (60 is within) | unhappy (boundary) |
| only Somatic | yes | 10 | S | `CounterspellEligible` | happy |
| only Material | yes | 10 | M | `CounterspellEligible` | happy |
| 61 feet | yes | 61 | V | `CounterspellIneligible(outOfRange)` | unhappy (boundary) |
| not seen | no | 10 | V | `CounterspellIneligible(casterNotSeen)` | unhappy (rule interaction) |
| no components (Subtle Spell) | yes | 10 | none | `CounterspellIneligible(noComponents)` | unhappy (rule interaction) |
| not seen and far and no components | no | 90 | none | `casterNotSeen` (first reason) | unhappy (rule interaction) |
| negative distance | yes | -1 | V | an `srderr.ErrInvalidInput` error naming `distanceFeet` | unhappy (invalid input) |
| DC carried | yes | 30 | V | `CounterspellEligible.Dc` equals the `SpellSaveDC` passed (FR-031); a rolled save with another target gives `saveTargetMismatch` on `Conclude` (FR-041) | unhappy (rule interaction) |

### 4.4 Antimagic Field

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-043 | `AntimagicField.Status(spell, {insideAura})` MUST return `suppressed` exactly when `insideAura` is true, the spell's `origin` is `ordinary`, and the spell is not exempt (FR-044); otherwise `functioning` (table T-A). | SRD 5.2.1, "Antimagic Field" ("Ongoing spells, except those cast by an Artifact or a deity, are suppressed in the area"), p. 109 | happy; unhappy (boundary) |
| FR-044 | A spell of `antimagic_field` (any caster) and of `prismatic_wall` MUST be exempt from suppression: auras of different Antimagic Fields "don't nullify each other", and the field "has no effect on the wall". | SRD 5.2.1, "Antimagic Field", p. 109; "Prismatic Wall", p. 155 | unhappy (rule interaction) |
| FR-045 | `AntimagicField.Blocks(activity, actorInside, targetsInside)` MUST return `AntimagicBlocks` with the reason `actorInsideAura` when `actorInside` is true, otherwise `targetInsideAura` when `targetsInside` is not empty (carrying the blocked targets), otherwise `AntimagicPermits`. `AntimagicActivity` MUST be `castSpell`, `takeMagicAction`, `createMagicalEffect`, `useMagicItemProperty`, `teleport` (table T-B). | SRD 5.2.1, "Antimagic Field" ("No one can cast spells, take Magic actions, or create other magical effects inside the aura, and those things can't target or otherwise affect anything inside it"), p. 109 | happy; unhappy (boundary) |
| FR-046 | For `useMagicItemProperty`, `actorInside` MUST mean the item is inside the aura or on anything inside it; the item's properties do not work then. A target list is ignored for this activity. | SRD 5.2.1, "Antimagic Field" ("Magical properties of magic items don't work inside the aura or on anything inside it"), p. 109 | unhappy (rule interaction) |
| FR-047 | For `teleport`, `actorInside` MUST mean the origin or the destination is inside the aura: no one teleports into or out of it. Planar travel is the same case. | SRD 5.2.1, "Antimagic Field" ("no one can teleport into or out of it or use planar travel there"), p. 109 | unhappy (rule interaction) |
| FR-048 | A cast blocked by the field MUST NOT reach Counterspell or Dispel Magic: the host asks `Blocks` first (FR-019). A Counterspell cast inside the aura is a spell cast and is blocked like any other. | SRD 5.2.1, "Antimagic Field", p. 109 | unhappy (rule interaction) |
| FR-049 | A suppressed spell stays in the registry with `remaining` falling (FR-008) and is not ended: when the aura ends or the holder leaves it, `Status` returns `functioning` again and the spell resumes with the shorter time. Suppression holds no state. | SRD 5.2.1, "Antimagic Field" ("While an effect is suppressed, it doesn't function"), p. 109 | unhappy (state error) |
| FR-050 | Areas of effect that "can't extend into the aura" and portals that close are geometry: the host removes the aura's volume from membership (`magical-effects.md` FR-019 to FR-021). This spec states no rule for them and returns no result. | SRD 5.2.1, "Antimagic Field", p. 109 | n/a (documented exclusion) |
| FR-051 | An ongoing spell whose `origin` is `artifactOrDeity` MUST be `functioning` inside the aura, and Dispel Magic treats it like any other spell (FR-011). | SRD 5.2.1, "Antimagic Field", p. 109; "Dispel Magic", p. 124 | unhappy (rule interaction) |
| FR-052 | The field's own Concentration is the caster's: when it ends, the host calls `Status` again; nothing in this spec changes a spell's state. | SRD 5.2.1, "Antimagic Field" (Duration: Concentration), p. 109 | unhappy (state error) |

#### Table T-A: `AntimagicField.Status` (FR-043, FR-044, FR-051)

| Case | Spell | Origin | Inside | Expected | Path |
|---|---|---|---|---|---|
| ordinary spell inside | `bless` | ordinary | yes | `suppressed` | happy |
| ordinary spell outside | `bless` | ordinary | no | `functioning` | happy |
| Concentration spell inside | `haste` | ordinary | yes | `suppressed` | happy |
| Artifact or deity | `bless` | artifactOrDeity | yes | `functioning` | unhappy (rule interaction) |
| a second field's aura | `antimagic_field` | ordinary | yes | `functioning` | unhappy (rule interaction) |
| Prismatic Wall | `prismatic_wall` | ordinary | yes | `functioning` | unhappy (rule interaction) |
| cantrip with a duration | `light` | ordinary | yes | `suppressed` | unhappy (boundary) |

#### Table T-B: `AntimagicField.Blocks` (FR-045 to FR-047)

| Case | Activity | Actor inside | Targets inside | Expected | Path |
|---|---|---|---|---|---|
| cast from inside | `castSpell` | yes | none | `AntimagicBlocks(actorInsideAura)` | happy |
| cast at a target inside | `castSpell` | no | one | `AntimagicBlocks(targetInsideAura)` | happy |
| both inside | `castSpell` | yes | one | `actorInsideAura` (first) | unhappy (rule interaction) |
| cast, nothing inside | `castSpell` | no | none | `AntimagicPermits` | happy |
| Magic action inside | `takeMagicAction` | yes | none | `AntimagicBlocks(actorInsideAura)` | happy |
| item property on something inside | `useMagicItemProperty` | yes | any | `AntimagicBlocks(actorInsideAura)` | unhappy (rule interaction) |
| item outside, target inside | `useMagicItemProperty` | no | one | `AntimagicPermits` | unhappy (boundary) |
| teleport out of | `teleport` | yes | none | `AntimagicBlocks(actorInsideAura)` | happy |
| duplicate target ids | `castSpell` | no | two equal | an `srderr.ErrInvalidInput` error | unhappy (invalid input) |

### 4.5 Layering, events and telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-053 | Every type here MUST live in package `combat` (domain files, one file per concept), be pure and synchronous, and import only the kernel packages (`srderr`, `dice` for `dice.Roller` types, `modifier`) and the published language of the specs in "Depends on" (ids, value objects, events, `d20.CheckResult`). It MUST NOT import a domain service or use case of `character` (no `ConcentrationRules`) or `toolbox`, MUST NOT import any `*test` package from production files, and MUST NOT depend on the types of `encounter-aggregate.md`. Composing `d20.ResolveCheck` belongs to the use cases of RA-4. | `doc/rules/architecture.md` §1, §2, §9; `combat-foundations.md` FR-026 | unhappy (forbidden edge) |
| FR-054 | Events MUST form the closed `SpellEndingEvent` (an interface with an unexported marker method that also satisfies `combat.Event`) (`combat-foundations.md` FR-015 to FR-019): `OngoingSpellEnded` (`ongoing_spell_ended`), `SpellCountered` (`spell_countered`), `CounterspellResisted` (`counterspell_resisted`), each an immutable value, past tense, equal by value, with `combatants` per table 4.5a. Every `kind` MUST be unique in the package. | `combat-foundations.md` FR-015 to FR-020; `shared-types-and-ownership.md` FR-019 | unhappy (name collision) |
| FR-055 | Results MUST be closed variant sets (interface with an unexported marker method, one struct type per variant) with operation-prefixed variants (`DispelMagicApplied`, `DispelMagicInvalid`, `CounterspellEligible`, `CounterspellIneligible`, `CounterspellCountered`, `CounterspellResisted`, `CounterspellInvalid`). No generic `Applied`, `Invalid` or `Rejected` is exported. Each reason is a value object or enum named `<Subject>Rejection` or `<Subject>Problem` held by the variant. | `combat-foundations.md` FR-022, FR-023; `shared-types-and-ownership.md` FR-017, FR-018 | unhappy (name collision) |
| FR-056 | Classes (`combat-foundations.md` FR-031): `DispelMagicApplied`, `CounterspellCountered`, `CounterspellResisted`, `CounterspellEligible` and `AntimagicPermits` are `applied` (a completed rule answer); `CounterspellIneligible` and `AntimagicBlocks` are `refused` (legal input the rules do not allow in this state); `DispelMagicInvalid` and `CounterspellInvalid` are `invalid`. | `combat-foundations.md` FR-030, FR-031 | happy |
| FR-057 | Telemetry is the use cases' (RA-4), through `CombatTelemetry` (over the `telemetry` ports) (`combat-foundations.md` FR-029 to FR-035). This spec adds no metric or span family; it adds the `srd5e.combat.result` values of section 7 as `CombatSignals` `...Value` constants, each mapped to one class (`combat-foundations.md` FR-030, FR-039). The domain records nothing. | `combat-foundations.md` FR-029 to FR-035; `shared-types-and-ownership.md` FR-044 | happy |

#### Table 4.5a: event kinds and combatants (FR-054)

| Event | `kind` | `combatants` order |
|---|---|---|
| `OngoingSpellEnded` | `ongoing_spell_ended` | the holder if it is a creature, then the caster if known and different |
| `SpellCountered` | `spell_countered` | the counterspeller, then the casting creature |
| `CounterspellResisted` | `counterspell_resisted` | the counterspeller, then the casting creature |

### Withdrawn ids

| Id | Reason |
|---|---|
| FR-007 | The registry's `advanced(elapsedSeconds)` and `OngoingSpellExpired` are replaced by `AdvanceActiveEffects` in `active-effects.md` (RA-6); expiry events are its. |

## 5. Model

- **Reused, not defined here:** `CombatantID`, `CombatEvent`; `OngoingEffect`, `OngoingEffectID`, `SpellSource`; `SpellObjectID`; `SpellID`, `ConditionKind`; `SpellLevel`, `SpellDuration`; `SlotLevel`; `CastSource`; `CheckResult`, `ForcedFailureResult`, `d20.Kind`, `d20.Outcome`, `DifficultyClass`, `RollModifiers`.
- **Value objects:** `OngoingSpell`, `SpellTimeRemaining`, `SpellOrigin`, `DispelRestriction`, `DispelTarget`, `CastObservation(spell, caster, source, used UsedComponents, RequiresConcentration, castingTime)`, `WastedActionCost`, `CounterspellSave`, `AntimagicActivity`.
- **Aggregate-like state:** `OngoingSpellRegistry` (immutable, host-held beside the encounter; no aggregate root, no id).
- **Domain services (stateless: zero-size struct types or plain functions; no constructor needed):** `DispelMagic` (`Plan`, `Conclude`), `DispelRestrictions` (`forSpell`), `Counterspell` (`Assess`, `Conclude`), `AntimagicField` (`Status`, `Blocks`).
- **Invariants (constructors):** FR-001, FR-003, FR-004, FR-005, FR-015, `distanceFeet` at least 0, no duplicate target ids in `targetsInside`.
- **Go naming:** member names written lowerCamel above (`remaining`, `origin`, `caster`, `conditions`, `distanceFeet`) are exported read-only accessors or fields in Go (`Remaining()`, `Origin()`); enum values written lowerCamel (`casterNotSeen`, `neverDispellable`) are exported typed constants prefixed by their type with a `String()` method; "null" parts are the zero value of the type, documented as invalid-or-absent.
- **Failures:** host input that depends on state is a result (`DispelMagicInvalid`, `CounterspellInvalid`, `CounterspellIneligible`); constructor and argument misuse is a returned error satisfying `errors.Is(err, srderr.ErrInvalidInput)`, a typed error carrying the argument name and invalid value; `MustX` helpers panic with that error. No `*srderr.PortError`.

## 6. Interface

- No port, no I/O: pure, synchronous functions over host-held values (no `ctx`). The use cases of RA-4 take a `context.Context` first and add the dice (`dice.Roller`), the resolver and telemetry. They are named here so the amendment is exact: `DispelOngoingSpells` (plan, roll one ability check per `DispelNeedsCheck` through the `dice.Roller` port, resolve with `d20.ResolveCheck`, conclude), `CounterCastSpell` (assess, roll the save, conclude), and `AssessAntimagicActivity`. `DispelOngoingSpells` ends spells by passing the ended ids to `EndActiveEffects` with `RemovedByEffect`; ageing is `AdvanceActiveEffects` (`active-effects.md`), not a use case here. None calls another use case.
- Package: every type of sections 3 and 4 is exported from package `combat` (import path `github.com/Whealinghold/srd5e-go/combat`); there is no re-export package. No test double is needed: all types are values.

## 7. Observability

Domain code is silent by design (`doc/rules/observability.md` §3): it is pure and records nothing. The signals below belong to the RA-4 use cases, which run in one `CombatTelemetry.Run` each (`combat-foundations.md` FR-029 to FR-035); `srd5e.outcome` is `success` or `failure` only. `doc/observability/signals.md` gets the three operation names and the closed `srd5e.combat.result` values (`dispel_nothing_ended`, `dispel_invalid_checks`, `counterspell_resisted`, `counterspell_ineligible`, `counterspell_invalid_save`, `antimagic_blocked`, each a `CombatSignals` `...Value` constant added by this spec); no new metric or span family. Result values carry no ids, levels or totals.

| Operation | Variant | `srd5e.combat.result` | Class |
|---|---|---|---|
| `dispel_ongoing_spells` | `DispelMagicApplied` (one or more ended) | `completed` | applied |
| | `DispelMagicApplied` (nothing ended) | `dispel_nothing_ended` | applied |
| | `DispelMagicInvalid` | `dispel_invalid_checks` | invalid |
| `counter_cast_spell` | `CounterspellCountered` | `completed` | applied |
| | `CounterspellResisted` | `counterspell_resisted` | applied |
| | `CounterspellIneligible` | `counterspell_ineligible` | refused |
| | `CounterspellInvalid` | `counterspell_invalid_save` | invalid |
| `assess_antimagic_activity` | `AntimagicPermits` | `completed` | applied |
| | `AntimagicBlocks` | `antimagic_blocked` | refused |

The seven questions (`doc/rules/observability.md` §2):

1. **What can go wrong?** The host passes a missing or mismatched check or save (`invalid`), asks for Counterspell on an ineligible cast or an action Antimagic Field blocks (`refused`), or builds an invalid value (a returned `srderr.ErrInvalidInput` error).
2. **How would an operator know?** `srd5e.combat.operations` by operation and result; an `invalid` result sets the span error status, a `refused` one does not. `srd5e.combat.errors` counts only port errors, and this spec has none.
3. **What would someone need to diagnose it?** Results carry the first problem, the refusal reason, the step list and the events; events carry the ids and the cause.
4. **What should be counted or timed?** Each call by operation and result, and each event by kind, through `CombatTelemetry`.
5. **Is existing instrumentation still correct?** Yes: no Concentration state or existing signal changes. `srd5e.character.result` `casting_failed` stays `character`'s.
6. **What should a consumer monitor?** Sustained `dispel_invalid_checks` or `counterspell_invalid_save` means the host builds the check or save wrongly. A high `counterspell_ineligible` share is a UI problem.
7. **Is anything now noisy?** No: no log is added, and the `dispel_nothing_ended` share is a game outcome.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-006 | unit | Source and category invariants; `CastAtLevel` for a cantrip, a ritual, an upcast and a feature-granted cast; `FromDuration` over all five `SpellDuration` variants and every unit, `SpellDurationInstantaneous` returns an error; duplicate ids; absent target; immutable lists; `RequiresConcentration` in equality |
| FR-008 to FR-010 | unit | suppression is not read by this spec; parent ended after the last part; two castings of one spell kept (FR-007 withdrawn) |
| FR-011 to FR-016 | unit | Table T-D, one test per row; empty plan; registry order; `castLevel` 2 returns an error; restriction precedence |
| FR-017, FR-018 | unit | Table T-R, one test per row; unknown id; an exhaustive type switch over `DispelRestriction` |
| FR-019, FR-020, FR-027 to FR-030 | unit | `Plan` takes no aura fact; Proficiency Bonus is not in the modifier; input unchanged; equal results; per-caster separate steps; a Concentration caster's other spells are not planned; no held spell in the registry |
| FR-058, FR-059 | unit | Wall keyed by `DispelEffect`; plan from another registry gives `planMismatch`; `modifiers` reach `DispelNeedsCheck` (FR-020) |
| FR-021 to FR-026 | unit | Check total at DC and DC minus 1; natural 20 on a failed total fails; each `DispelMagicInvalid` problem and their order; nothing ended; event order with a parent; `EndsCasterConcentration` false for a non-Concentration spell or an unknown caster, set once for several spells |
| FR-031 to FR-033, FR-041 | unit | Table T-C; save kind and target mismatch; forced failure |
| FR-034 to FR-040, FR-042 | unit | Countered: each `CastSource`, each action cost, Concentration and non-Concentration spell; resisted; chain order built by two calls; no `Assess` for a release; events and determinism |
| FR-060 | unit | `Imprisonment` keeps `RequiresSlotLevel(9)` (pinned); the prison-or-component choice is the host's (documented exclusion, no test) |
| FR-043 to FR-052 | unit | Table T-A and table T-B, one test per row; suppressed spell loses time and resumes; artifact origin; nothing returned for FR-050 (documented exclusion, no test) |
| FR-053 to FR-057 | architecture | Import direction (no `character` service, no `toolbox`); `SpellEndingEvent` closed, `kind` unique; exported-name scan for generic variant names (check A7); class of each variant |
| Scenario | scenario | Public API only: Dispel Magic at slot 3 on a creature with a level 2 buff, a level 5 buff and a Forcecage; Counterspell on a Concentration cast; a spell expiring inside an Antimagic Field |
| errors-and-results.md FR-031, FR-034, FR-051 | application | A failing roller double (`dicetest.CallbackRoller (returning an error)`) makes the use cases return the error wrapped once by `srderr.WrapPort`, with no partial result; a recorded failure sets `error.type` from `srderr.ErrorType(err)`; a panic in a telemetry double propagates uncounted. |

No doubles or contract suites. Mutation testing (`doc/rules/testing.md` §8, `gomu` inside Docker only): all changed domain lines, with `castLevel` versus spell level, the DC of 10, the 60 feet and 6 seconds per round as the boundary mutants to kill.

Deliberately not covered: FR-007, FR-050, FR-060 (path n/a) - withdrawn or documented exclusions with no behaviour to test.

## 9. Open questions and decisions

None escalated.

**Settled** (one line each): Q-1 spell level for Dispel Magic is the level the spell took when cast (FR-002): the SRD says a spell cast with a higher slot "takes on the higher level for that casting" (p. 104), and community consensus agrees. Q-2 a countered Concentration spell still ends prior Concentration (FR-036): the SRD says Concentration is lost "the moment you start casting" (p. 179), and community consensus holds the same; RA-1 asks `character` for the transition. Q-3 Dispelling a Concentration spell ends its caster's Concentration (FR-026): the SRD ties the effect to the creator's Concentration (p. 179) and has no reason for it to continue; interpretation, reason `ended` reused, no new reason. Q-4 only the slot is restored (FR-037): the SRD says only that (p. 120); interpretation, consistent with community readings that a daily feature use is still spent. Q-5 a Counterspell in response resolves first (FR-039): a Reaction interrupts its trigger; interpretation. Q-6 a readied spell is countered when readied, not when released (FR-040): the SRD says the spell is cast at Ready and only its release waits (p. 187), and the designers' public ruling says Counterspell stops casting, not release; interpretation. Q-7 the dispel check adds no Proficiency Bonus (FR-020): p. 124 says "your spellcasting ability" and no skill applies. Q-8 parts of Guards and Wards and Prismatic Wall take the parent's `CastAtLevel` and the violet layer waits for the others (FR-009, table T-R): the SRD gives no level for parts and says layers fall "in order from red to violet" (p. 155); interpretation. Q-9 Imprisonment follows its own text: Dispel Magic ends it only from a level 9 slot (p. 142), which then ends it with no check since 9 is not above 9. Q-10 suppression holds no state (FR-049): it is derived from the aura each time, so the registry cannot disagree with the geometry. Q-11 the registry sits beside the encounter and is not an aggregate (`encounter-aggregate.md` unchanged): no invariant spans it and a creature. Q-12 time is whole seconds (FR-003): rounds are 6 seconds (p. 13) and no unit finer than a round exists. Q-13 Dispel Magic ending conditions is the host's job (FR-025): `conditions.md` owns removal and may hold the same condition from another source.

## 10. Changelog impact

- **Added:** Dispel Magic, Counterspell and Antimagic Field rules, the ongoing-spell read model and the three events in `combat` (`doc/specs/dispel-and-counterspell.md`).
- **BREAKING:** no. Everything is new; the amendments RA-1 to RA-4 are requests, not changes.
