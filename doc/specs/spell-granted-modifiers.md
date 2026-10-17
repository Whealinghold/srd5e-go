# Spell-granted modifiers

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `observability-ports.md`, `content-schema-and-decoder.md`, `d20-tests.md`, `derived-stats.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `attacks.md`, `initiative-and-actions.md` |

What an **active** spell contributes to rolls and stats, as inputs that the existing specs already accept (`Modifier`, `RollCircumstance`, flat Armor Class bonuses, `DamageComponent`, `DamageAdjustment`, `MovementSpeeds`). Rank 28 in table 4.11 of shared-types-and-ownership.md (amendment A4), beside `active-effects` and `encounter-aggregate`: after `spell-effects` (27), before `combat-action-resolution` (29), which is the host-facing composer of these inputs. The **lifecycle** of an active spell (start, duration, Concentration, expiry, dismissal) and **Combining Spell Effects** (p. 106) are owned by `active-effects.md` (draft, rank 28; its FR-015 to FR-019 `EffectCombination`), as are the roll and damage riders of Bless, Bane, Hex and Hunter's Mark (its FR-048 to FR-052 `EffectRiders`). This spec does not import it (equal rank, shared-types-and-ownership.md FR-056): the host passes `SpellModifierSource`s taken from the effects that `EffectCombination.Applying` returns (FR-008), and combines the dice of both. `combat-action-resolution.md` (29) does that wiring.

## 1. Purpose and scope

`spell-effects.md` resolves a cast (Q11 there) and `spell-effect-data.md` keeps "riders on later attack rolls or saves" as text (its E16). Nothing says what Bless adds to a roll, what Shield does to Armor Class, or when Hunter's Mark adds damage. This spec is the rule layer for what `active-effects.md` does not model: pure functions from a list of **currently applying** spell sources plus the context of one roll or stat to the contributions, one small use case that rolls bonus dice, and a closed catalogue of 12 SRD spells.

- **Roll contributions** (FR-010 to FR-024): Guidance as a bonus die; Haste, Slow, Bestow Curse, Hunter's Mark (finding it) as Advantage, Disadvantage or a flat penalty. Bless and Bane dice and Hex's check Disadvantage are `EffectRiders` of `active-effects.md` (FR-011, FR-012, FR-015, FR-019 withdrawn here).
- **Armor Class** (FR-040 to FR-046): Shield, Shield of Faith, Haste, Slow bonuses and penalties; Mage Armor's base; Barkskin's floor.
- **Speed** (FR-050 to FR-056): Longstrider, Haste, Slow.
- **Damage** (FR-061 to FR-067): Bestow Curse extra damage; Resistance's reduction of damage taken. Hunter's Mark and Hex extra damage is `EffectRiders.ExtraDamage` (FR-060 withdrawn).
- **Sources** (FR-001 to FR-008): the source value and the closed catalogue. Combining (FR-003 to FR-006) is `active-effects.md`'s and withdrawn here.

**Out of scope:**

- Lifecycle: durations, Concentration, expiry, dismissal, "ends early" triggers, moving a mark to a new creature, repeated saves of Slow, the start-of-turn save of Bestow Curse. The host (later active-effects) passes only sources that are active now. This spec never reads a clock.
- Casting, targets, saves to resist the spell, and the cast-time damage of a spell (`spellcasting.md`, `spell-effects.md`). Spells with an instant effect are not here.
- Action economy of the spells: Haste's extra action, Slow's one-action limit and no Reactions, Slow's 25 percent failure, Haste's lethargy at the end, and the forced Dodge of Bestow Curse. They are returned as named facts (FR-070), never resolved.
- Spells whose effect is a condition (Hold Person), a Temporary Hit Point gain (Heroism), true Resistance or Immunity (Heroes' Feast, Hallow), Hit Point maximum or other stat changes: other specs or text only. Only the 12 spells of table C are modelled; every other id returns `SpellModifiersNotModelled` (FR-002).
- Other fifth-edition material. Nothing core was missing from the PDF for these spells.
- Rule options: the SRD names none here (`doc/rules/architecture.md` §3).

## 2. Source

SRD 5.2.1 (`doc/research/sources.md`); printed page equals PDF page.

- "Combining Spell Effects", p. 106. "D20 Tests", pp. 6-8 (modifiers, Advantage/Disadvantage). "Armor Class", p. 7 and p. 177. "Order of Application", "Death Saving Throws", p. 17. "Damage Rolls", "Critical Hits", p. 16. "Spell Attack", "Speed" ("Changes to Your Speeds"), p. 188. "Rolling 20 or 1", p. 7.
- Spell entries: Bane 112; Barkskin 112; Bestow Curse 112-113; Bless 113; Guidance 138; Haste 139; Hex 140; Hunter's Mark 141; Longstrider 145; Mage Armor 145; Resistance 158; Shield 161-162; Shield of Faith 162; Slow 163.

Package rules: `doc/rules/architecture.md` §2-§6; `doc/rules/observability.md` §2-§7; `doc/rules/testing.md` §4, §5, §8. Binding: shared-types-and-ownership.md FR-001, FR-003, FR-013, FR-015, FR-017, FR-026, FR-044, FR-048, FR-056, FR-062.

Web research informed three judgements, each justified in SRD terms in section 9: Barkskin's floor against bonuses (settled from the SRD text; interpretation), Bless and Bane on Death Saving Throws, and the order of the Resistance cantrip against Resistance (no clarification found; interpretation). No web text is copied.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| (an active spell on a creature) | `SpellModifierSource` | Package term. Supplied by the host; this spec holds no lifecycle. |
| Combining Spell Effects | `EffectCombination` (`active-effects.md`) | FR-008; not defined here. |
| attack roll, saving throw, ability check, Death Saving Throw | `RollContext` (closed variant set) | FR-010. |
| "adds 1d4", "subtract 1d4" | `BonusDie` | A `dice.Expression` with a sign. Guidance here; Bless and Bane come from `EffectRiders`. FR-016, FR-030. |
| bonus / penalty to AC | `FlatBonus` (`derived-stats.md` FR-038) | FR-040. |
| base AC calculation | `AbilityBasedArmorClassBasis` (`derived-stats.md`) | Mage Armor, with a base of 13 (amendment A1). |
| extra damage | `DamageComponent` (`attacks.md`) | `DiceDamage`. FR-060. |
| reduces the damage taken | `DamageAdjustment` (`hit-points-and-damage.md`) | `FlatAdjustment` with a negative amount (amendment A2). |
| Speed | `MovementSpeeds` (`initiative-and-actions.md` FR-075, FR-076, FR-109) | FR-050. |
| Advantage, Disadvantage | `RollCircumstance` | Kernel type. |
| creature | `CombatantID` | Owned by `combat-foundations.md`. |

New package terms: `SpellModifierSource`, `SpellModifierChoice`, the catalogue (`LookupSpellModifiers`), `SpellModifierLookup`, the `Query*` functions, `SpellRollContributions`, `BonusDie`, `SpellArmorClassInputs`, `SpellSpeedEffects`, `SpellDamageReduction`, `SpellEffectFact`, `RollSpellBonusDice` with its command and result. None collides with an SRD term.

## 4. Requirements

Stable ids. Ids are allocated in blocks; an id absent from this section and not in the Withdrawn ids was never used by this spec. Table C is the catalogue; "a source" is a `SpellModifierSource`. Case-table labels are not ids.

### 4.1 Sources and catalogue

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | A `SpellModifierSource` MUST hold the `SpellID`, the casting `CombatantID` (`caster`), the affected creature (`target`), a `CastOrder` (a non-negative `int` used only to order output, FR-023; it need not be unique) and a `SpellModifierChoice` (closed variant set, an interface with an unexported marker method: `NoChoice`, `ChosenSkill(Skill)`, `ChosenAbility(Ability)`, `ChosenDamageType(DamageType)`, `ChosenCurse(CurseEffect)`; `CurseEffect` closed variant set: `CurseAbility(Ability)`, `CurseAttackRolls`, `CurseDodge`, `CurseExtraDamage`). A choice of the wrong variant for the spell (Guidance without a skill, Haste with any choice, Resistance with a type outside the eleven of its list) MUST be rejected by the constructor `NewSpellModifierSource` with an error that is `errors.Is` `srderr.ErrInvalidInput` (the `Must` form panics with it). | SRD 5.2.1, "Guidance", p. 138; "Hex", p. 140; "Resistance", p. 158; "Bestow Curse", pp. 112-113 | unhappy (invalid input) |
| FR-002 | `SpellModifierCatalogue.lookup(SpellID)` MUST return the closed `SpellModifierLookup`: `SpellModifiersKnown(entry)` for the 12 ids of table C, else `SpellModifiersNotModelled(spellID)`. A not-modelled spell contributes nothing to any query and is not an error (the spell may still resolve its cast through `spell-effects.md`). Ids are `snake_case` (`hunters_mark`, `shield_of_faith`; shared-types-and-ownership.md FR-007). | SRD 5.2.1, "Effects", p. 106 (outcomes beyond the stated effects are the GM's); `doc/rules/architecture.md` §6 | unhappy (missing content) |
| FR-007 | Every query MUST be a pure function of the sources and its context. It MUST NOT read a clock, a duration or a Concentration state, and a source the host passes is treated as applying. An empty source list MUST give the neutral result of each query (no dice, no circumstance, no bonus, speed and Armor Class unchanged). The input list MUST NOT be altered. | `doc/rules/architecture.md` §3 (purity); lifecycle is out of scope (section 1) | unhappy (empty and minimal) |
| FR-008 | Sources MUST be the applying effects: the host runs `EffectCombination.Applying` (`active-effects.md` FR-015 to FR-018) and maps each result to a `SpellModifierSource`. Different spells add; same spell and same subject give one source, the latest or most potent. A list holding two sources with the same `SpellID` and `Target` MUST be rejected: every `Query*` function returns an error that is `errors.Is` `ErrDuplicateSource` (wrapping `srderr.ErrInvalidInput`; the host skipped the combining step). Different spells with equal `CastOrder` are legal. Bestow Curse recast by a second caster on the same target is one source (the owner's key); it is the owner's rule, not re-decided here. | SRD 5.2.1, "Combining Spell Effects", p. 106; shared-types-and-ownership.md FR-048 | unhappy (rule interaction; invalid input) |

### 4.2 Contributions to D20 Tests

Withdrawn ids of this section: FR-011, FR-012, FR-015, FR-019 (see the end of section 4). `QueryRollContributions(sources, RollContext)` returns `SpellRollContributions{BonusDice, Modifiers, Circumstances}`. `RollContext` is a closed variant set (an interface with an unexported marker method): `AttackRollContext(attacker, target)`, `SavingThrowContext(creature, ability)`, `DeathSavingThrowContext(creature)`, `AbilityCheckContext(creature, ability, skill?, seeking?)` (`seeking` a `CombatantID` when the check is made to find that creature).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | A source applies to a context only when its subject is the roller: the `target` of the source is the rolling creature for Guidance, Haste, Slow and Bestow Curse (including the attack clause of FR-020, whose extra filter is that `AttackRollContext.target` equals the source's `caster`); the `caster` only for Hunter's Mark's Advantage (FR-021). No source ever affects another creature's roll; the caster attacking the cursed creature gets nothing from Bestow Curse. | SRD 5.2.1, "Bestow Curse", p. 112; "Hunter's Mark", p. 141 | unhappy (rule interaction) |
| FR-013 | Bless and Bane dice (the riders of `active-effects.md`, whose saving-throw rider rolls include a Death Saving Throw; amendment A7) MUST apply to a Death Saving Throw. Interpretation: the SRD calls it a saving throw that "isn't tied to an ability score" (p. 17), so "a saving throw" includes it. The die becomes a `Modifier` in the `Modifier` parameter of `MakeDeathSavingThrow` (`hit-points-and-damage.md` FR-061); a natural 20 or 1 keeps its own rule. | SRD 5.2.1, "Death Saving Throws", p. 17; "Bless", p. 113 | unhappy (rule interaction) |
| FR-014 | One roll gets at most one die per spell (Bless, Bane, Guidance) however many d20s it rolls: a roll with Advantage or Disadvantage (two d20s, p. 8) gets one `1d4`, not two. | SRD 5.2.1, "Bless", p. 113 ("adds 1d4 to the attack roll or save"); "Roll Two D20s", p. 8 | unhappy (rule interaction) |
| FR-016 | Guidance MUST add one `1d4` to an `AbilityCheckContext` whose `skill` equals the chosen `Skill`, whatever the ability used with it, and to no other check, attack roll or save. A check with no skill (a raw ability check) MUST NOT get it. | SRD 5.2.1, "Guidance", p. 138 | unhappy (boundary) |
| FR-017 | Haste MUST add one `Advantage` to a `SavingThrowContext` of Dexterity of the hasted creature and nothing to other saves. | SRD 5.2.1, "Haste", p. 139 | happy |
| FR-018 | Slow MUST add the `Modifier` -2 to a Dexterity `SavingThrowContext` of the slowed creature. It is a penalty, not a Disadvantage. | SRD 5.2.1, "Slow", p. 163 | happy |
| FR-020 | Bestow Curse with `CurseAbility(a)` MUST add one `Disadvantage` to every ability check and saving throw of the cursed creature made with ability `a`; with `CurseAttackRolls` it MUST add one `Disadvantage` to an `AttackRollContext` of the cursed creature whose `target` is the caster of the curse and to no other attack roll. `CurseDodge` and `CurseExtraDamage` add no roll contribution here. A Death Saving Throw has no ability and is unaffected by `CurseAbility`. | SRD 5.2.1, "Bestow Curse", pp. 112-113 | unhappy (rule interaction) |
| FR-021 | Hunter's Mark MUST add one `Advantage` to the marking caster's `AbilityCheckContext` with `skill` Perception or Survival (ability Wisdom) when `seeking` equals the marked `target`, and to no other check. | SRD 5.2.1, "Hunter's Mark", p. 141 | unhappy (boundary) |
| FR-022 | The `circumstances` MUST be handed on for `modifier.Resolve` together with those of conditions and the host's other sources; this spec MUST NOT resolve a mode, so Advantage from Haste and Disadvantage from Hex cancel as a whole (`modifiers-and-advantage.md` FR-015). `modifiers` carry only flat `Modifier`s (Slow's -2). Hex's check Disadvantage arrives from `EffectRiders.RollContributions` as a `RollModeSource` in the same list. | SRD 5.2.1, "They Don't Stack", p. 8; shared-types-and-ownership.md FR-026 | unhappy (rule interaction) |
| FR-023 | `SpellRollContributions` MUST list its dice and circumstances in ascending `CastOrder` of their sources, so equal inputs give equal outputs, and MUST expose defensive copies of slices with value equality. | `doc/rules/architecture.md` §3, §4 | unhappy (determinism; immutability) |
| FR-024 | A context for which no source applies MUST give an empty `SpellRollContributions`; a source of a spell that is not modelled MUST be skipped (FR-002). | `doc/rules/testing.md` §4 (empty and minimal) | unhappy (empty and minimal) |

Table R: roll contexts (one test per row; "none" is an empty result). Rows of withdrawn ids (Bless, Bane, Hex on checks) are tested in `active-effects.md`.

| Label | Active source | Context | Result | Source | Path |
|---|---|---|---|---|---|
| R3 | Bless rider on A (from `EffectRiders`) | A Death Saving Throw | +1d4 | p. 113, p. 17 | unhappy (rule interaction) |
| R9 | Guidance on A, Stealth | A Dexterity (Stealth) check | +1d4 | p. 138 | happy |
| R10 | Guidance on A, Stealth | A Dexterity (Acrobatics) check | none | p. 138 | unhappy (boundary) |
| R11 | Guidance on A, Stealth | A raw Dexterity check, no skill | none | p. 138 | unhappy (boundary) |
| R12 | Haste on A | A Dexterity save | Advantage | p. 139 | happy |
| R13 | Haste on A | A Wisdom save | none | p. 139 | unhappy (boundary) |
| R14 | Slow on A | A Dexterity save | Modifier -2 | p. 163 | happy |
| R15 | Haste and Slow on A | A Dexterity save | Advantage and -2 | p. 106 | unhappy (rule interaction) |
| R18 | Bestow Curse on A, `CurseAbility(wisdom)` | A Wisdom save and a Wisdom check | Disadvantage each | pp. 112-113 | happy |
| R19 | Bestow Curse on A, `CurseAttackRolls`, caster C | A attacks C; A attacks D; C attacks A | Disadvantage; none; none | pp. 112-113 | unhappy (rule interaction) |
| R20 | Hunter's Mark by C on M | C Wisdom (Survival) check seeking M; seeking N | Advantage; none | p. 141 | unhappy (boundary) |
| R21 | Source for a spell id not in table C (Bane) | any | none | FR-002 | unhappy (missing content) |

### 4.3 Rolling the bonus dice

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-030 | `RollSpellBonusDice.Do(ctx, cmd)` MUST roll each `BonusDie` of its command (the host merges `SpellRollContributions.bonusDice` with the signed dice of `EffectRiders.RollContributions`) once through `dice.Roll(roller, expr)` (which rejects a result for another expression), in list order, and return `SpellBonusDiceRolled{Modifiers, Rolls}`: one signed `modifier.Modifier` per die (the rolled total, negated for a penalty die) and the rolled `dice.Result`s as evidence. A roller failure returns a nil result and a `*srderr.PortError` (FR-035). The host appends the `modifiers` to the roll with `d20.RollModifiers.WithOthers` (`d20-tests.md` FR-017) and may pass `circumstances` and `modifiers` of the contributions on unchanged. | SRD 5.2.1, "Bless", p. 113; "Bane", p. 112; "Guidance", p. 138 | happy |
| FR-031 | With no `BonusDie` the use case MUST request no die and return an empty `SpellBonusDiceRolled`. | `doc/rules/testing.md` §4 (empty and minimal) | unhappy (empty and minimal) |
| FR-032 | A Bane die MUST give a negative `Modifier` of the rolled amount (a d4 showing 3 gives -3). The modified total MAY fall below 1 (`modifiers-and-advantage.md` FR-008). | SRD 5.2.1, "Bane", p. 112; "D20 Tests", p. 6 | unhappy (boundary) |
| FR-033 | The dice are modifiers of the total. A natural 20 or natural 1 on an attack roll MUST stay judged on the d20 (`d20-tests.md` FR-066): Bless cannot turn a natural 1 into a hit and Bane cannot turn a natural 20 into a miss. | SRD 5.2.1, "Rolling 20 or 1", p. 7 | unhappy (rule interaction) |
| FR-034 | The host MUST call the use case only for a roll that is made with a d20. A save the creature chooses to fail or fails automatically rolls no d20 and so no bonus die. The dice are rolled **before** the d20; the SRD fixes no order and the total has the same distribution (interpretation). | SRD 5.2.1, "Bless", p. 113 ("makes an attack roll or a saving throw"); "Saving Throws", p. 7 | unhappy (rule interaction) |
| FR-035 | A A roller failure MUST be returned wrapped exactly once as a `*srderr.PortError` for `dice.Port` (`errors.Is` the original cause) with a nil result and no partial result; the call MUST NOT recover a panic itself (`dice.Roll` already recovers a roller panic as `srderr.ErrPortPanic`). The same command and roller sequence MUST give an equal result. | `doc/rules/architecture.md` §6; errors-and-results.md | unhappy (collaborator failure; determinism) |

### 4.4 Armor Class

`QueryArmorClass(sources, wearsArmor)` returns `SpellArmorClassInputs{FlatBonuses, AlternativeBase, Floor}` (the last two optional: a documented zero value or a (value, ok) accessor). The host passes them to `CalculateDerivedStats` (amendment A1); this spec computes no AC.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | Shield MUST give a `FlatBonus` of +5, Shield of Faith +2, Haste +2 and Slow -2. Each carries its spell as source. All of them add (different spells, FR-008) and none is a base: they apply after the base and the worn Shield (`derived-stats.md` FR-038). | SRD 5.2.1, "Shield", pp. 161-162; "Shield of Faith", p. 162; "Haste", p. 139; "Slow", p. 163 | happy |
| FR-041 | Mage Armor MUST give an `AlternativeBase` with base value 13, the Dexterity modifier added as for any base, only when `WearsArmor` is false. With armor worn it MUST give none (the spell "ends early if the target dons armor"; ending it is the host's, so the query is neutral rather than rejecting). Interpretation: a carried Shield (the item) does not stop the spell, since "Armor Training" (p. 177) treats a Shield separately from armor worn. The cast precondition (a willing creature not wearing armor) belongs to the cast (`spell-effects.md` or the host); the query is neutral when `WearsArmor` is true. The base is one candidate of the usual choice: the character "chooses which calculation to use" and uses one (`derived-stats.md` FR-037, FR-054). | SRD 5.2.1, "Mage Armor", p. 145; "Armor Class", p. 177 | unhappy (rule interaction) |
| FR-042 | Barkskin MUST give `floor` 17: the creature's final Armor Class is the larger of its computed value and 17. The floor MUST be applied **after** the base, the Shield and every flat bonus (Shield of Faith, Haste, Shield, Slow), never as a base. Interpretation, argued from the SRD text alone: "has an Armor Class of 17 if its AC is lower than that" is a minimum on the finished AC; a computed AC above 17 is kept, and a bonus matters only when it lifts the total over 17. Rejected: "set AC to 17, then add later bonuses", because it makes "if its AC is lower" meaningless. | SRD 5.2.1, "Barkskin", p. 112; "Armor Class", p. 177 | unhappy (rule interaction) |
| FR-043 | `SpellArmorClassInputs.ApplyFloor(int)` MUST be a pure function returning the larger of its argument and the floor, and the argument unchanged when there is no floor. | SRD 5.2.1, "Barkskin", p. 112 | unhappy (boundary) |
| FR-044 | The Shield spell's +5 applies "including against the triggering attack": `ReevaluateAttack(AttackRollResult, ArmorClass) (AttackRollResult, error)` MUST return the `AttackRollResult` of the same `Roll` and `RollModifiers` judged against the new Armor Class through `d20.ResolveAttack` (the pure rules function; its error is returned unchanged). No die is rolled. It is pure for any new Armor Class, which need not exceed the old AC. A natural 20 stays a hit and a Critical Hit whatever the AC; a total that now misses is a miss. | SRD 5.2.1, "Shield", pp. 161-162; "Rolling 20 or 1", p. 7 | unhappy (boundary; rule interaction) |
| FR-045 | The AC modifiers of FR-040 MUST NOT be clamped: Slow's -2 on a low AC may give any integer (the SRD states no minimum AC). | SRD 5.2.1, "Slow", p. 163; "Armor Class", p. 7 | unhappy (boundary) |
| FR-046 | Two Shield of Faith castings on one creature MUST give one +2 (FR-008: the host passes one source); Shield of Faith and Haste on one creature give +4. | SRD 5.2.1, "Combining Spell Effects", p. 106 | unhappy (rule interaction) |

Table A: Armor Class (one test per row; computed AC means base, Shield item and flat bonuses before spells).

| Label | Computed AC | Active sources | Final AC | Source | Path |
|---|---|---|---|---|---|
| A1 | 12 | Shield spell | 17 | p. 161 | happy |
| A2 | 12 | Shield spell, Shield of Faith, Haste | 21 | pp. 161-162, 139 | unhappy (rule interaction) |
| A3 | 12 | Slow | 10 | p. 163 | happy |
| A4 | 12 | Barkskin | 17 | p. 112 | happy |
| A5 | 19 | Barkskin | 19 | p. 112 | unhappy (boundary) |
| A6 | 12 | Barkskin, Shield of Faith | 17 (14 raised by the floor) | p. 112, FR-042 | unhappy (rule interaction) |
| A7 | 16 | Barkskin, Shield of Faith | 18 (floor not needed) | p. 112, FR-042 | unhappy (rule interaction) |
| A8 | Dex +3, no armor | Mage Armor | base 16 (13 + 3) | p. 145 | happy |
| A9 | Dex +3, armor worn | Mage Armor | no base from the spell | p. 145 | unhappy (state error) |
| A10 | attack total 14 vs AC 12 hit, then Shield | reevaluate against 17 | miss | p. 161 | unhappy (rule interaction) |
| A11 | natural 20 vs AC 12, then Shield | reevaluate against 17 | hit, Critical Hit | p. 161, p. 7 | unhappy (boundary) |
| A12 | 12 | Barkskin, Shield spell | 17 (12 + 5 = 17; the floor adds nothing) | p. 112, p. 161, FR-042 | unhappy (rule interaction) |

### 4.5 Speed

`QuerySpeeds(sources, MovementSpeeds)` returns `MovementSpeeds`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-050 | Longstrider MUST add 10 feet to every Speed (`MovementSpeeds.AdjustedBy`, `initiative-and-actions.md` FR-076): "any special speed you have increases by an equal amount". | SRD 5.2.1, "Longstrider", p. 145; "Speed", "Changes to Your Speeds", p. 188 | happy |
| FR-051 | Haste MUST double every Speed the creature has. | SRD 5.2.1, "Haste", p. 139 | happy |
| FR-052 | Slow MUST halve every Speed, each rounded down (`MovementSpeeds.Halved`, FR-109 there). | SRD 5.2.1, "Slow", p. 163; "Round Down", p. 187 | unhappy (boundary) |
| FR-053 | Order MUST be: additions (Longstrider), then doubling (Haste), then halving (Slow). Interpretation: the SRD fixes no order for Speed changes (p. 188 only says special speeds change by the same effect). Additions before multipliers is the package's chosen reading, so a Longstrider bonus is itself doubled or halved; it is not derived from p. 17. | SRD 5.2.1, "Changes to Your Speeds", p. 188 | unhappy (rule interaction) |
| FR-054 | Haste and Slow on one creature MUST both apply, so the doubled value is halved again, each step rounding down. | SRD 5.2.1, "Combining Spell Effects", p. 106 | unhappy (rule interaction) |
| FR-055 | A Speed MUST NOT fall below 0 and a Speed the creature lacks stays absent. A creature whose Speed is already 0 stays 0 under Haste. | SRD 5.2.1, "Speed", p. 188 | unhappy (boundary) |
| FR-056 | Two Longstrider castings on one creature MUST give one +10 (FR-008: the host passes one source). | SRD 5.2.1, "Combining Spell Effects", p. 106 | unhappy (rule interaction) |

Table S: Speed (one test per row; walk 30 and fly 60 where stated).

| Label | Sources | Speeds in | Speeds out | Source | Path |
|---|---|---|---|---|---|
| S1 | Longstrider | walk 30 | walk 40 | p. 145 | happy |
| S2 | Longstrider | walk 30, fly 60 | walk 40, fly 70 | p. 188 | unhappy (rule interaction) |
| S3 | Haste | walk 30, fly 60 | walk 60, fly 120 | p. 139 | happy |
| S4 | Slow | walk 35 | walk 17 | p. 163, p. 187 | unhappy (boundary) |
| S5 | Longstrider, Haste | walk 30 | walk 80 ((30+10) x 2); interpretation | FR-053 | unhappy (rule interaction) |
| S6 | Haste, Slow | walk 35 | walk 35 (70 halved); interpretation | FR-054 | unhappy (rule interaction) |
| S7 | Slow | walk 0 | walk 0 | p. 188 | unhappy (empty and minimal) |

### 4.6 Damage riders and damage reduction

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-061 | `QueryDamageRiders(sources, attacker, target, DamageEvent)` MUST return the Bestow Curse extra damage as a `DiceDamage` component for the host to append to the damage (`attacks.md` FR-057). `DamageEvent` is a closed variant set (interface with an unexported marker method): `AttackRollHitEvent` (a weapon or spell attack roll hit), `SpellDamageEvent` (damage of a spell that uses a saving throw or no roll) and `OtherDamageEvent` (any other damage, such as a trap or an object). A miss produces no event. No rider applies to `OtherDamageEvent`, to damage dealt by a creature other than the caster, or to a creature other than the cursed one. Hunter's Mark and Hex extra damage on an attack roll hit is `EffectRiders.ExtraDamage` (`active-effects.md` FR-049) and not repeated here. | SRD 5.2.1, "Bestow Curse", p. 113; "Spell Attack", p. 188 | unhappy (rule interaction) |
| FR-062 | The Bestow Curse die is part of the attack's damage, so on a Critical Hit it is rolled twice by the attack's own rule (`attacks.md` FR-056, FR-057) with no rule here; it has no flat modifier. | SRD 5.2.1, "Critical Hits", p. 16 | unhappy (rule interaction) |
| FR-063 | Bestow Curse with `CurseExtraDamage` MUST give a `1d8` Necrotic component whenever the caster deals damage to the cursed creature with an attack roll or a spell: for `AttackRollHitEvent` and `SpellDamageEvent` alike. The extra damage is keyed to the (spell, target) source of FR-008. Interpretation: once per attack that hits and once per cast spell that deals damage to the target, not once per damage type or per target hit by an area. | SRD 5.2.1, "Bestow Curse", p. 113 | unhappy (rule interaction) |
| FR-064 | The Bestow Curse die is Necrotic, fixed by the catalogue. It goes through Resistance, Vulnerability and Immunity as any `Damage` (`hit-points-and-damage.md` FR-019); this spec applies none. | SRD 5.2.1, "Resistance and Vulnerability", p. 17 | unhappy (rule interaction) |
| FR-065 | `QueryDamageReduction(sources, creature, DamageType, alreadyReducedThisTurn)` MUST return `SpellDamageReduction{Dice: 1d4}` for a Resistance source on that creature whose `ChosenDamageType` equals the damage type and `AlreadyReducedThisTurn` is false; otherwise none. "Once per turn" is the host's flag (the turn is not tracked here). | SRD 5.2.1, "Resistance", p. 158 | unhappy (state error) |
| FR-066 | `SpellDamageReduction.ToAdjustment(rolled int) (DamageAdjustment, error)` MUST give a `FlatAdjustment` of `-rolled` scoped to the chosen `DamageType` (amendment A2). It sits in the adjustment step: before Resistance and Vulnerability, floored at 0 (`hit-points-and-damage.md` FR-016, FR-019). `rolled` outside 1..4 MUST return an error that is `errors.Is` `srderr.ErrInvalidInput` (the function never panics). Interpretation (Q3): "reduces the total damage taken" is a reduction of the amount, and p. 17 puts such reductions ("an aura that reduces all damage by 5") before Resistance; with Resistance to the type, 11 damage and a roll of 3 gives 8, halved to 4. The other reading (after Resistance) gives 11 halved to 5, minus 3, is 2. Rolling the d4 is the host's, with `dice.Roller`. | SRD 5.2.1, "Order of Application", p. 17; "Resistance", p. 158 | unhappy (rule interaction) |
| FR-067 | A creature with Resistance to the chosen type, and the spell Resistance, MUST take the reduction first and then the halving (table D, D5). The spell Resistance is never itself "Resistance" of p. 17 and does not count in "No Stacking". | SRD 5.2.1, "No Stacking", "Order of Application", p. 17 | unhappy (rule interaction) |

Table D: damage (one test per row).

| Label | Case | Result | Source | Path |
|---|---|---|---|---|
| D5 | Resistance (Fire) on A, d4 shows 3; A takes 11 Fire and is Resistant to Fire | 11 - 3 = 8, halved to 4 (interpretation, FR-066) | p. 158, p. 17 | unhappy (rule interaction) |
| D6 | Resistance (Fire) on A; A takes Cold damage | none | p. 158 | unhappy (boundary) |
| D7 | Resistance (Fire), already reduced this turn | none | p. 158 | unhappy (state error) |
| D8 | Resistance (Fire), d4 shows 4, damage 3 | adjustment -4, damage 0 (floored) | p. 17 | unhappy (boundary) |
| D9 | Bestow Curse `CurseExtraDamage` by C on M; C casts Fireball damaging M | `1d8` Necrotic added once | p. 113 | unhappy (rule interaction) |
| D10 | Same; C hits M with a Critical Hit | `1d8` rolled twice by `attacks.md` | p. 113, p. 16 | unhappy (boundary) |
| D12 | Bestow Curse by C on M; D hits M; C hits another creature; trap damage to M | none each | p. 113 | unhappy (rule interaction) |
| D13 | `ToAdjustment(0)` and `ToAdjustment(5)` | error (`errors.Is` `srderr.ErrInvalidInput`) | FR-066 | unhappy (invalid input) |

### 4.7 Facts that are not resolved

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-070 | Each catalogue entry MUST expose its non-numeric parts as `SpellEffectFact` values, a closed variant set (interface with an unexported marker method) and descriptive only: `ExtraActionGranted(allowed: attack(1), dash, disengage, hide, utilize)` and `LethargyOnEnd(Incapacitated, Speed 0 until end of next turn)` for Haste; `ReactionsForbidden`, `ActionOrBonusActionNotBoth`, `OneAttackOnly`, `SpellFailureChancePercent(25)` for Slow; `MarkMovesOnZeroHitPoints` for Hunter's Mark and Hex; `ForcedDodgeUnlessWisdomSave` for Bestow Curse `CurseDodge`; `EndsIfArmorDonned` for Mage Armor; `NoDamageFromMagicMissile` for Shield. No fact is resolved here; the host or active-effects applies it (an Incapacitated at the end through `ApplyConditionToSet`). An entry with any fact has coverage `partial`; Hunter's Mark and Hex stay `partial` because of their move clause. Shield's duration (until the start of the caster's next turn) is lifecycle and the host's. | SRD 5.2.1, "Haste", p. 139; "Slow", p. 163; "Hex", p. 140; "Hunter's Mark", p. 141; "Bestow Curse", pp. 112-113; "Mage Armor", p. 145; "Shield", p. 162 | happy |
| FR-071 | A fact MUST NOT change any numeric result of FR-010 to FR-066: a hasted creature's Speed, AC and Dexterity saves are the same whether or not the host handles the extra action. | `doc/rules/architecture.md` §3 (purity) | unhappy (rule interaction) |

### 4.8 Layering, purity and exports

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-080 | The catalogue, sources, queries, contributions, facts and `ReevaluateAttack` MUST live in package `combat`, in files named `spell_modifier*.go`, and be pure, synchronous and silent (no `srderr`, `telemetry`, `dice.Roller` or `context` reference). `RollSpellBonusDice`, its command and result live in `roll_spell_bonus_dice_usecase.go`. | `doc/rules/architecture.md` §1, §9 | happy |
| FR-081 | The catalogue is rule constants in code, one entry per row of table C, because each entry needs a bespoke shape and the set is small (`shared-types-and-ownership.md` FR-045: small rules tables are constants). A test MUST pin every entry to its SRD figures (dice, bonus, type, page). A later content spec MAY move it to data (Q3). | shared-types-and-ownership.md FR-005, FR-045; `doc/rules/testing.md` §4 | happy |
| FR-082 | The domain MUST import `d20`, `character` and `content` only through their exported API and the `combat` specs of lower rank; it MUST NOT import `ConditionEffects` or call a use case, and `ReevaluateAttack` MUST use the pure `d20.ResolveAttack`, not a `d20.Resolver` (a use case). | `doc/rules/architecture.md` §2; shared-types-and-ownership.md FR-013, FR-026, FR-044; combat-foundations.md FR-026 | unhappy (forbidden edge) |
| FR-083 | The package MUST NOT name any consumer. Every exported name of section 6 MUST be unique across the module's packages and not a generic variant name. | AGENTS.md, "Hard constraints"; shared-types-and-ownership.md FR-017, FR-021 | unhappy (name collision) |
| FR-084 | Commands, sources and results MUST be immutable value objects, equal when every field is equal, with slices copied on construction and on read. Out-of-range input (a negative `CastOrder`, a wrong choice variant, two sources with the same spell and target, FR-008) MUST be rejected with an error that is `errors.Is` `srderr.ErrInvalidInput` and never a panic. | `doc/rules/architecture.md` §3, §6 | unhappy (immutability; invalid input) |

### 4.9 Observability

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-090 | `RollSpellBonusDice` MUST record through the combat telemetry helper (combat-foundations.md FR-035) with the operation `roll_spell_bonus_dice` (span `srd5e.combat.roll_spell_bonus_dice`) and define no signal of its own beyond that span row and two `srd5e.combat.result` values: `rolled` (at least one die) and `nothing_to_roll` (none), both class `applied` (outcome `success`). It adds no extra attribute. | combat-foundations.md FR-029 to FR-031, FR-040; `doc/rules/observability.md` §4, §5 | happy |
| FR-091 | A roller failure MUST be recorded once by the combat telemetry helper (span error, one `srd5e.combat.errors` increment with `error.type`) and MUST NOT be logged. The use case MUST NOT log for a normal outcome and MUST emit no `combat.Event`. No spell id, creature id or amount is an attribute. | combat-foundations.md FR-032 to FR-034; `doc/rules/observability.md` §7 | unhappy (collaborator failure) |
| FR-092 | The operation, span and the two result values MUST be exported constants in `combat/signals.go`; the use case MUST return an equal result with the no-op `telemetry.Noop` and the recording `telemetrytest` doubles. The domain queries emit nothing. | `doc/rules/observability.md` §9; combat-foundations.md FR-039 | happy |

### 4.10 Catalogue

Table C: the 12 modelled spells (FR-002, FR-081; one pin test per row). "Subject" is the rolling or affected creature of the source (FR-010). Bless, Bane and the Hex and Hunter's Mark dice are not entries: they are `EffectRiders` (`active-effects.md`). The Hunter's Mark die is a d6; the Ranger's Foe Slayer (SRD 5.2.1, p. 59) makes it a d10. A class-feature die is the host's, set in the template it builds, as `spell-effects.md` Q18 treats class features; no entry here varies.

| Label | Spell id, page | Subject | Contribution | Facts (FR-070) | Requirement |
|---|---|---|---|---|---|
| C1 | `barkskin`, p. 112 | target | AC floor 17 | none | FR-042 |
| C2 | `bestow_curse`, pp. 112-113 | target | one of: Disadvantage on checks and saves with a chosen ability; Disadvantage on attacks against the caster; extra 1d8 Necrotic; forced Dodge | `ForcedDodgeUnlessWisdomSave` | FR-020, FR-063 |
| C3 | `guidance`, p. 138 | target | +1d4 on checks with the chosen skill | none | FR-016 |
| C4 | `haste`, p. 139 | target | Speed x2, AC +2, Advantage on Dexterity saves | extra action, lethargy | FR-017, FR-040, FR-051 |
| C5 | `hex`, p. 140 | target | none here (die and check Disadvantage are riders) | mark moves | FR-070 |
| C6 | `hunters_mark`, p. 141 | caster | Advantage on Wisdom (Perception or Survival) to find it (die is a rider) | mark moves | FR-021, FR-070 |
| C7 | `longstrider`, p. 145 | target | Speed +10 | none | FR-050 |
| C8 | `mage_armor`, p. 145 | target | base AC 13 + Dexterity, no armor | ends if armor donned | FR-041 |
| C9 | `resistance`, p. 158 | target | -1d4 damage of the chosen type, once per turn | none | FR-065 |
| C10 | `shield`, pp. 161-162 | target | AC +5 | no damage from Magic Missile | FR-040, FR-044, FR-070 |
| C11 | `shield_of_faith`, p. 162 | target | AC +2 | none | FR-040 |
| C12 | `slow`, p. 163 | target | Speed halved, AC -2, Dexterity saves -2 | no reactions, one action, one attack, 25% spell failure | FR-018, FR-040, FR-052 |

### 4.11 Withdrawn ids

| Id | Reason |
|---|---|
| FR-003 to FR-006 | Combining Spell Effects is `active-effects.md` FR-015 to FR-019 (shared-types-and-ownership.md FR-048); FR-008 replaces them |
| FR-011, FR-012, FR-015, FR-019 | Bless, Bane dice and Hex check Disadvantage are `EffectRiders.RollContributions` (`active-effects.md` FR-048, FR-050) |
| FR-060 | Hunter's Mark and Hex extra damage is `EffectRiders.ExtraDamage` (`active-effects.md` FR-049) |

## 5. Model

Notation: `Name(a, b)` lists the fields of a struct type `Name`; a "closed" type is an interface with an unexported marker method, a doc comment naming every variant, and one struct per variant, switched exhaustively (`errors-and-results.md` FR-018). Constructors are `NewX` returning `(T, error)` with errors satisfying `errors.Is(err, srderr.ErrInvalidInput)`; `MustX` panics with that error. Slices and maps are copied on construction and on read.

**Domain** (package `combat`, files `spell_modifier*.go`; pure, silent):

- **Inputs:** `SpellModifierSource`, `SpellModifierChoice` and `CurseEffect` (FR-001), `RollContext` (section 4.2), `DamageEvent` (closed variant set: `AttackRollHitEvent`, `SpellDamageEvent`, `OtherDamageEvent`, FR-061).
- **Catalogue:** `LookupSpellModifiers` returns `SpellModifierLookup` (`SpellModifiersKnown`, `SpellModifiersNotModelled`); an entry holds its subject (FR-010), its contributions and its `SpellEffectFact`s.
- **Pure functions:** `QueryRollContributions`, `QueryArmorClass`, `QuerySpeeds`, `QueryDamageRiders`, `QueryDamageReduction`, `LookupSpellModifiers`, `ReevaluateAttack`.
- **Outputs:** `SpellRollContributions`, `BonusDie`, `SpellArmorClassInputs`, `SpellDamageReduction`, `SpellEffectFact`.
- **Invariants:** FR-001 and FR-084 in the constructors (`New...` returning `(T, error)`, `errors.Is` `srderr.ErrInvalidInput`); expected absence (a spell not modelled) is a closed result (FR-002).

**Use case:** `RollSpellBonusDice`, `RollSpellBonusDiceCommand`, `RollSpellBonusDiceResult` (one variant `SpellBonusDiceRolled`, so a plain result struct). No rejection exists because the only input is built from validated values; the only failure is a roller error.

**Reused, not defined here:** `Modifier`, `RollMode`, `RollCircumstance` (kernel); `dice.Expression`, `dice.Result`, `dice.Roller`, `dice.Roll` (`dice-and-rolls.md`); `Roll`, `RollModifiers`, `ArmorClass`, `AttackRollResult`, `ResolveAttack` (`d20-tests.md`, package `d20`); `FlatBonus`, `AbilityBasedArmorClassBasis` (`derived-stats.md`); `DamageComponent`, `DiceDamage` (`attacks.md`); `Damage`, `DamageAdjustment`, `DamageType` (`hit-points-and-damage.md`, `content`); `MovementSpeeds` (`initiative-and-actions.md`); `Skill`, `Ability`, `SpellID` (`content`); `CombatantID`, the combat telemetry helper and signal constants (`combat-foundations.md`).

## 6. Interface

Domain functions are synchronous and pure and take no `context.Context`. The use case is synchronous because `dice.Roller` is; it takes `ctx` first only because it starts a span, and never stores it.

```go
// roll_spell_bonus_dice_usecase.go
type RollSpellBonusDice struct{ /* roller, telemetry */ }

func NewRollSpellBonusDice(roller dice.Roller, opts ...Option) (RollSpellBonusDice, error)
func (u RollSpellBonusDice) Do(ctx context.Context, cmd RollSpellBonusDiceCommand) (RollSpellBonusDiceResult, error)
```


- **Command:** the `SpellRollContributions` for one roll. **Result:** `SpellBonusDiceRolled(modifiers, rolls)`.
- **Host flow for one roll:** (1) `RollContributions(activeSources, context)`; (2) `RollSpellBonusDice` if the roll is made with a d20 (FR-034); (3) append the modifiers to `d20.RollModifiers.Others`, pass the circumstances to `modifier.Resolve`; (4) `d20.Resolver` or `ResolveAttack`. For Armor Class: `ArmorClass` then `CalculateDerivedStats`. For Speed: `speeds`. For damage: `DamageRiders` into the `DamageSpec`, `DamageReduction` into `ApplyDamage`. `combat-action-resolution.md` composes these.
- **Ports:** only `dice.Roller` and the telemetry bundle `telemetry.Telemetry`; no new port, double or contract suite.
- **Exports** from package `combat`, each with a godoc comment starting with its name: `SpellModifierSource`, `SpellModifierChoice` and variants, `CurseEffect` and variants, the catalogue (`LookupSpellModifiers`), `SpellModifierLookup` and variants, the `Query*` functions, `RollContext` and variants, `DamageEvent`, `SpellRollContributions`, `BonusDie`, `SpellArmorClassInputs`, `SpellDamageReduction`, `SpellEffectFact` and variants, `RollSpellBonusDice`, `RollSpellBonusDiceCommand`, `RollSpellBonusDiceResult`, `SpellBonusDiceRolled`. Signal constants go into the `combat` signal constants.

## 7. Observability

The domain is silent by design: pure functions over values emit nothing. `RollSpellBonusDice` emits through the combat telemetry helper (FR-090 to FR-092).

1. **What can go wrong?** The `dice.Roller` fails; a host passes a source that has expired (a stale bonus on rolls); the host forgets the bonus dice (spell effects silently missing); a programmer error.
2. **How would an operator know?** `srd5e.combat.operations` with operation `roll_spell_bonus_dice`; a roller failure is an error span plus `srd5e.combat.errors`. A forgotten or stale source is not visible to the package; it shows as roll outcome drift in the host's own D20 Test signals.
3. **Diagnosis:** the span and the `result` value; the evidence of the result (each rolled die).
4. **Counted or timed:** calls by result and the shared duration histogram.
5. **Existing instrumentation still correct?** Yes. No other use case is called, so `srd5e.d20_test.*` and `resolve_attack` signals are unaffected and nothing is counted twice.
6. **Monitor:** any non-zero error rate; the `nothing_to_roll` share (a high share means hosts call it for rolls with no active source, wasted calls).
7. **Noise?** No log at all (combat-foundations.md FR-034).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.roll_spell_bonus_dice` | span | shared attributes (`srd5e.combat.result` `rolled` or `nothing_to_roll`) | Path of one bonus-dice call, on the shared span | Any error status |

Entries feed `doc/observability/signals.md`; the two `result` values are added to the shared row's Attributes cell (combat-foundations.md FR-037). Renaming any constant is **BREAKING**.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-007 | unit | Wrong choice variants return an error; unknown spell id is `SpellModifiersNotModelled`; R15, R21, A2, S6; FR-008: two sources with the same spell and target return `ErrDuplicateSource`, two different spells with equal `CastOrder` are accepted; negative `CastOrder` returns an error; empty list neutral; input list unchanged. |
| FR-010 to FR-024 | unit | Table R one test per row; Advantage plus Disadvantage cancelling is asserted at `modifier.Resolve` over the returned circumstances; list order by `CastOrder`; immutability. |
| FR-030 to FR-035 | application | `dicetest.FixedRoller`: dice requested in list order, negative modifier for Bane, empty input requests nothing, natural 1 with Bless still a miss and natural 20 with Bane still a hit through the real `d20.Resolver`, a roller failure is returned as a `*srderr.PortError` with no partial result, equal results for equal sequences. |
| FR-040 to FR-046 | unit | Table A one test per row, including the A10 and A11 `ReevaluateAttack` rows; Mage Armor with and without armor; floor applied last (A6, A7); negative AC allowed. |
| FR-050 to FR-056 | unit | Table S one test per row against the real `MovementSpeeds`; order of operations (S5, S6). |
| FR-060 to FR-067 | unit | Table D one test per row (D13 is the `ToAdjustment` range check); D5 against the real `DamageResolver` pipeline through `ApplyDamage` with a scoped `FlatAdjustment`; Critical Hit doubling asserted through `attacks.md` (D10). |
| FR-070, FR-071 | unit | Facts listed per entry, coverage `partial`; numeric results identical with and without facts. |
| FR-080 to FR-084 | architecture | File and import rules (the `spell_modifier*.go` files import no `srderr`, `telemetry` or use case, and no `ConditionEffects`); export scan for duplicate and generic names (`internal/archtest`); catalogue pin test per table C row; value equality, copied slices, the error cases (`errors.Is` `srderr.ErrInvalidInput`). |
| FR-090 to FR-092 | application | Recording `telemetrytest` doubles through the combat telemetry helper: one span, one `operations` increment with result `rolled` or `nothing_to_roll`, one duration; error counted once and not logged; no `events`; identical result with the no-op. |
| errors-and-results.md FR-031, FR-034, FR-051 | application | The roller failure of FR-035 is returned wrapped once as a `*srderr.PortError` with no partial result (`errors.Is` the original cause); the recorded failure's `error.type` is `srderr.ErrorType(err)` (FR-091); a telemetry backend panic propagates uncounted (FR-051). |

- **Levels:** unit (`combat/spell_modifier*_test.go`, package `combat_test`), application (`combat/roll_spell_bonus_dice_usecase_test.go`), architecture (`internal/archtest`), one scenario. Fuzz: none (no parser). No contract suite (no port).
- **Doubles:** `dicetest.FixedRoller`, the recording `telemetrytest` doubles, the real `d20.ResolveAttack`, `MovementSpeeds` and damage pipeline. No mocks of the domain.
- **Scenario (`scenarios/`, package `scenarios_test`, public API only):** a blessed, hasted character with Shield of Faith attacks a creature marked by Hunter's Mark: bonus dice rolled, the Advantage and Disadvantage resolved, a Critical Hit doubling the rider, the target's Shield spell turning a later hit into a miss.
- **Mutation testing (`doc/rules/testing.md` §8):** the `spell_modifier*.go` files at score >= 80, run with `gomu` only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors. Most valuable: the duplicate guard of FR-008, the floor order of FR-042, the speed order of FR-053, the context filters of FR-016 and FR-020.
- Every FR id is covered above; none is deliberately uncovered.

## 9. Open questions and decisions

No question is escalated. Package decisions (changing any is **BREAKING** once released):

- **Q1 (settled): Barkskin is a floor on the finished AC** (FR-042). Interpretation, from the SRD wording alone: "has an Armor Class of 17 if its AC is lower" only makes sense as a minimum, with other bonuses (the Shield spell included) only mattering above 17 (table A rows A6, A12). SRD basis: p. 112 and the single base calculation of p. 177. Rejected: treating it as a base (it would make Shield of Faith add to 17).
- **Q2 (settled): Bless and Bane apply to Death Saving Throws** (FR-013). Researched: the common reading. SRD basis: p. 17 calls it a saving throw (only "not tied to an ability score"). Rejected: excluding it (the SRD has no such exclusion).
- **Q3 (settled, interpretation): the Resistance cantrip reduces before Resistance** (FR-066). SRD basis: p. 17 order of application (the aura example). Web research found no official clarification and no consistent community ruling, so the evidence is split; a recommended default is chosen, not escalated: the d4 is a reduction of the amount, and the only SRD rule that orders such reductions puts them before Resistance. Behaviour difference: 4 against 2 in D5. Catalogue as code (FR-081) rather than content data: 12 bespoke rows; moving to data is deferred until the effect-data spec carries ongoing effects (amendment A5).
- **Q4 (settled): combining** is `active-effects.md`'s (FR-008; its FR-015 to FR-019). Rejected: a second implementation here, and a caster-keyed Bestow Curse or Hunter's Mark (it would diverge from the owner's (spell, subject) key; a change is an amendment to `active-effects.md`, A7).
- **Q5 (settled): bonus dice rolled before the d20** (FR-034): the SRD fixes no order; rejected alternative: rolling after the d20 through `AdjustedBy`, which would need every use case to hand back control mid-roll.
- **Q6 (settled): Speed order** (FR-053, not derived from p. 17) and Bestow Curse extra damage once per attack or spell (FR-063): interpretations stated in the requirements.
- **Q7 (settled): spells not in table C** return `SpellModifiersNotModelled` (FR-002), never an error. Adding a spell to the catalogue is a new row and a pin test, not a breaking change.

### Amendments requested of other specs (not edited here)

| Id | Spec | Change | Why |
|---|---|---|---|
| A1 | `derived-stats.md` | `AbilityBasedArmorClassBasis` gains `BaseValue` (default 10); `FlatBonusSource` accepts a spell id; `CalculateDerivedStatsCommand` gains an `ArmorClassFloor` applied after every flat bonus, shown as a `floor` term | Mage Armor 13, spell bonuses, Barkskin (FR-040 to FR-043) |
| A2 | `hit-points-and-damage.md` | `FlatAdjustment` gains an optional `DamageScope`; an unscoped one keeps applying to every `Damage` | Resistance cantrip reduces one type only (FR-066) |
| A3 | `initiative-and-actions.md` | `MovementSpeeds.Doubled()` beside `AdjustedBy` and `halved` | Haste (FR-051) |
| A4 | `shared-types-and-ownership.md` | Table 4.11 row 28 lists `encounter-aggregate`: add `active-effects` and `spell-granted-modifiers` to it (all three equal rank, none depends on another) | FR-056 |
| A5 | `spell-effect-data.md`, `spell-effects.md` | E16 and Q11 point to `active-effects.md` (lifecycle, riders) and this spec (numeric modifiers) | Ownership |
| A6 | `active-effects.md` | Mention in its Depends-on-free note that `SpellModifierSource`s are mapped from `EffectCombination.Applying` results (FR-008 here); no dependency either way | Lifecycle boundary |
| A7 | `active-effects.md` | `BonusDiceRider` saving-throw rolls include a Death Saving Throw (FR-013); Hunter's Mark advantage on Perception/Survival to find the mark stays here (FR-021), not a rider | One owner |

## 10. Changelog impact

Under `Unreleased`, **Added:** spell-granted modifiers (the `Query*` functions, the catalogue (`LookupSpellModifiers`) for Barkskin, Bestow Curse, Guidance, Haste, Hex, Hunter's Mark, Longstrider, Mage Armor, Resistance, Shield, Shield of Faith, Slow), `RollSpellBonusDice`, the `srd5e.combat.roll_spell_bonus_dice` span and the `rolled` and `nothing_to_roll` result values (added to `doc/observability/signals.md`). Not **BREAKING** while unreleased. After the first release, changing a table C figure other than to fix a transcription error, or renaming a signal or export, is **BREAKING**.
