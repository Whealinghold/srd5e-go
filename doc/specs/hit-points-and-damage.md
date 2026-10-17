# Hit points and damage

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `dice-and-rolls.md`, `observability-ports.md`, `errors-and-results.md`, `d20-tests.md`, `ability-scores.md`, `derived-stats.md`, `content-schema-and-decoder.md` |

Downstream specs (`conditions`, `attacks`, `encounter-aggregate`, `combat-action-resolution`) consume this one and are
not listed; the in-context order is `shared-types-and-ownership.md` FR-015. Also used: `IsBloodied`
(`derived-stats.md` FR-071; `shared-types-and-ownership.md` FR-049), `HitPointMaximum` (`derived-stats.md` FR-060),
`DamageType` (`content-schema-and-decoder.md` FR-107) and `Skill` (FR-106; `shared-types-and-ownership.md` FR-005,
FR-046).

## 1. Purpose and scope

The rules of "Damage and Healing": Hit Points and maximum, applying instances of damage, half damage on a successful
save, Resistance, Vulnerability and Immunity, Temporary Hit Points, Healing, dropping to 0 Hit Points (instant death,
falling Unconscious, the optional knock-out), Death Saving Throws, stabilizing and the Stable state. The state is a
creature's hit-point status (state in, new state and events out); hosts decide when things happen (turn start, elapsed
time, whether an attack was a Critical Hit). Damage rolling (dice, Critical Hit doubling, modifiers, flooring, fixed
damage, the Damage Type) is owned by `attacks.md`; this spec receives finished `Damage` values.

**Out of scope:**

- Deciding that an attack hit or is a Critical Hit, rolling damage (`attacks.md`), and dice and damage values of
  weapons, spells and monsters (content).
- The Unconscious, Prone and Incapacitated conditions (`conditions.md`, downstream): here Unconscious appears only as
  the `FellUnconscious` and `KnockedOut` events and the `IsUnconscious` predicate. Deciding that Exhaustion reached
  level 6 (`IsDeadFromExhaustion`); this spec records the death (FR-095).
- The Concentration save ("Concentration", p. 179): `spellcasting.md` owns it and takes a plain `int` damage amount
  (`shared-types-and-ownership.md` FR-011); this spec only emits `DamageTaken` with plain `int` amounts (FR-049,
  FR-084, FR-102).
- The `DamageType` and `Skill` vocabularies (owner above); this spec owns only the behaviour that interprets
  `DamageType` (FR-016 to FR-025).
- Rest mechanics, Hit Point Dice and how a maximum is derived (`rests-and-hit-dice.md`, `derived-stats.md`); this spec
  only exposes the operation that ends Temporary Hit Points at the end of a Long Rest. `HitPoints.maximum` is a plain
  `int`; `HitPointMaximum` is built from it only for `BecameBloodied` (FR-101), a `combat` to `character` edge the
  context map allows (`doc/rules/architecture.md` §2).
- Revival of a Dead creature (for example Raise Dead, "Dead", p. 180): Dead is terminal. Objects: Object Hit Points,
  Object Armor Class, Damage Threshold ("Breaking Objects", pp. 177-178; "Damage Threshold", p. 180; Q-11). Hazard
  damage, condition Immunity, and effects that reduce a maximum (content; this spec provides the operation).
- Aid and Heroes' Feast maximum increases, maximum restoration and Death Ward: `revival-and-restoration-spells.md`,
  composed from FR-109 operations (`Restore`, `ReduceMaximum`, `ApplyDamage` with `MeleeAttackKnockOut`) and the
  FR-087 `RestoreHitPointStatus` parser; no requirement here changes for them.
- Anything not found in SRD 5.2.1. Nothing core to this topic was left out.

## 2. Source

Printed page equals PDF page; "Damage Rolls" and "Critical Hits" are both on p. 16.

- `SRD 5.2.1, "Hit Points", p. 16`; `"Damage Rolls", p. 16`; `"Saving Throws and Damage" ("Damage against Multiple
  Targets", "Half Damage"), p. 16`
- `SRD 5.2.1, "Damage Types", pp. 16-17`; "Rules Glossary", "Damage Types", p. 180
- `SRD 5.2.1, "Resistance and Vulnerability" ("No Stacking", "Order of Application"), p. 17`; `"Immunity", p. 17`;
  `"Healing", p. 17`
- `SRD 5.2.1, "Knocking Out a Creature" (sidebar), p. 17`; `"Dropping to 0 Hit Points" ("Instant Death", "Falling
  Unconscious"), p. 17`
- `SRD 5.2.1, "Death Saving Throws", pp. 17-18`; `"Stabilizing a Character", p. 18`
- `SRD 5.2.1, "Temporary Hit Points" ("Lose Temporary Hit Points First", "Duration", "They Don't Stack", "They're Not
  Hit Points or Healing"), p. 18`
- Glossary: "Bloodied", p. 177; "Critical Hit", p. 179; "Death Saving Throw", p. 181; "Hit Points", p. 183;
  "Immunity", p. 183; "Resistance", p. 187; "Round Down", p. 187; "Stable", p. 188; "Temporary Hit Points", p. 190;
  "Unconscious", p. 191; "Vulnerability", p. 191.
- Package decisions (`shared-types-and-ownership.md`): FR-004, FR-005, FR-006, FR-011, FR-015, FR-017 to FR-021,
  FR-036 to FR-044, FR-046, FR-049, FR-051, FR-052. Rules: `doc/rules/architecture.md` (§1 to §4, §6),
  `doc/rules/observability.md` (§2-§6), `doc/rules/testing.md` (§3-§5).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Hit Points, Hit Point maximum, Temporary Hit Points | value object `HitPoints` (`current`, `maximum`, `temporary`) | Current is 0 to the maximum. `maximum` is a plain `int`; `HitPointMaximum` (at least 1) is built from it only for `BecameBloodied` (FR-101). Temporary Hit Points are a buffer, not Hit Points. |
| Bloodied | `IsBloodied` (`derived-stats.md` FR-071) | Called as `IsBloodied(current, HitPointMaximum(maximum))` for `BecameBloodied`. This spec defines no Bloodied predicate. |
| Critical Hit | `HitKind.CriticalHit` | Here it only decides how many Death Saving Throw failures damage at 0 Hit Points adds (FR-063). `HitKind` (`NormalHit`, `CriticalHit`) is the named-state form (`doc/rules/architecture.md` §3) of `AttackRollResult.IsCriticalHit` (`shared-types-and-ownership.md` FR-004): the one deliberate exception to "no parallel enums", a damage-application input only (FR-111). |
| Damage Types | `DamageType` (owner above) | The thirteen Damage Types of `content-schema-and-decoder.md` FR-107 (SRD 5.2.1, "Damage Types", p. 180). |
| instance of damage | `Damage` | An amount of one Damage Type; Resistance and Vulnerability apply once per instance. Keeps the plain names `Damage` and `ApplyDamageCommand` (`shared-types-and-ownership.md` FR-021). |
| Resistance, Vulnerability, Immunity | `DamageAffinities.resistances` (a `DamageScope`: one type or all damage), `.vulnerabilities`, `.immunities` | Immunity to conditions is out of scope. |
| Healing; Instant Death | `RestoreHitPoints`; `DeathCause` | Instant death: Monster Death, Hit Point Maximum of 0, Massive Damage ("Damage at 0 Hit Points", p. 18, also applies). |
| Unconscious; Stable; Dead | `IsUnconscious`; `LifeState.stable`; `LifeState.dead` | The Unconscious condition is owned by `conditions.md`. Stable: 0 Hit Points, no Death Saving Throws. |
| Death Saving Throw | `MakeDeathSavingThrow` | A saving throw and so a D20 Test (Glossary "D20 Test", p. 180) with no ability modifier or proficiency: the host states Advantage or Disadvantage and any Exhaustion reduction (p. 181); FR-061, Q-4. |
| player character / monster | `ZeroHitPointsRule` | Chooses instant death or Death Saving Throws. |

These names have **no SRD source**; they are descriptive package names and add no rules: `HitPointStatus`,
`LifeState`, `Alive` ("not at 0 Hit Points, or knocked out"), `Dying` ("at 0 Hit Points and making Death Saving
Throws"), `DamageAffinities`, `HitKind`, `KnockedOut` (from the sidebar title), `MeleeAttackKnockOut`,
`DamageDelivery`, `HitPointRejection`, `HitPointsError` / `HitPointsProblem`, `HitPointStatusError` /
`HitPointStatusProblem` (FR-105), `DeathCause.DeathSavingThrowFailures` and `DeathCause.Exhaustion`. `HitPointStatus`
is an immutable value object with no identity; other specs refer to the creature by `CombatantID`
(`combat-foundations.md` FR-001).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. No requirement is provisional: section 9 has no open question.

### 4.1 Hit Points

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `HitPoints(current, maximum, temporary)` MUST be constructible when `0 <= current <= maximum` and `temporary >= 0`, and MUST expose the three values. | SRD 5.2.1, "Hit Points", p. 16; Glossary "Hit Points", p. 183 | happy |
| FR-002 | `NewHitPoints(current, maximum, temporary)` MUST return `(HitPoints, error)`: a `*HitPointsError` (matching `srderr.ErrInvalidInput`) holding every violated `HitPointsProblem` (FR-105), for `current < 0`, `current > maximum`, `maximum < 0` or `temporary < 0`; it never panics. `MustHitPoints` panics with that error. | SRD 5.2.1, "Hit Points", p. 16; Glossary "Hit Points", p. 183 | unhappy (invalid input) |

### 4.2 Damage values and half damage

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | Damage dealt via a successful saving throw that the effect halves MUST be half the damage that would be dealt on a failed save, rounded down (table 4.2b). It is applied as the `HalfDamage` adjustment (FR-099). | SRD 5.2.1, "Half Damage", p. 16; Glossary "Round Down", p. 187 | unhappy (boundary) |
| FR-015 | Every `Damage` MUST carry exactly one `DamageType`. | SRD 5.2.1, "Damage Types", pp. 16-17 | happy |
| FR-086 | `Damage(amount, type)` MUST accept an amount of 0 or more and MUST return an error matching `srderr.ErrInvalidInput` for a negative amount. Each per-component result of `attacks.md` converts to exactly one `Damage` with the same amount and type. | SRD 5.2.1, "Damage Rolls", p. 16 | unhappy (invalid input) |
| FR-100 | One `Damage` value MUST be usable unchanged for several creatures (it is immutable): applying it to two creatures with different `DamageAffinities` MUST give each its own result and MUST NOT alter the `Damage`. The damage is rolled once by the caller. | SRD 5.2.1, "Damage against Multiple Targets", p. 16 | happy |

Table 4.2b, half damage (one test per case; SRD 5.2.1, "Half Damage", p. 16): 14 on a failed save gives 7 (even,
happy); 15 gives 7 (odd rounds down, boundary); 1 gives 0 (boundary); 0 gives 0 (empty and minimal).

### 4.3 Resistance, Vulnerability and Immunity

Pipeline for one `Damage`, in order: (1) adjustments (flat bonuses and penalties, then `HalfDamage`), floored at 0;
(2) Resistance; (3) Vulnerability; Immunity overrides all.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-016 | Flat adjustments (bonuses and penalties) MUST be applied first, and a result below 0 MUST be treated as 0 before Resistance and Vulnerability. | SRD 5.2.1, "Order of Application", p. 17; "Damage Rolls", p. 16 | unhappy (boundary) |
| FR-017 | Resistance to a Damage Type MUST halve damage of that type, rounded down. | SRD 5.2.1, "Resistance and Vulnerability", p. 17; Glossary "Resistance", p. 187 | happy |
| FR-018 | Vulnerability to a Damage Type MUST double damage of that type. | SRD 5.2.1, "Resistance and Vulnerability", p. 17; Glossary "Vulnerability", p. 191 | happy |
| FR-019 | Damage MUST be resolved in the order adjustments, then Resistance, then Vulnerability (table 4.3a, including the worked example of 28 Fire damage becoming 22). | SRD 5.2.1, "Order of Application", p. 17 | unhappy (rule interaction) |
| FR-020 | A creature with both Resistance and Vulnerability to the same Damage Type MUST have both applied (halved, rounded down, then doubled). | SRD 5.2.1, "Order of Application", p. 17 | unhappy (rule interaction) |
| FR-021 | Multiple instances of Resistance, or of Vulnerability, affecting the same Damage Type MUST count as one, and each of Resistance and Vulnerability MUST be applied at most once to an instance of damage. | SRD 5.2.1, "No Stacking", p. 17; Glossary "Resistance", p. 187, "Vulnerability", p. 191 | unhappy (rule interaction) |
| FR-022 | Resistance to all damage MUST apply to damage of every Damage Type, and together with Resistance to one type MUST halve once. | SRD 5.2.1, "No Stacking", p. 17 | unhappy (rule interaction) |
| FR-023 | Immunity to a Damage Type MUST make the damage of that type 0. | SRD 5.2.1, "Immunity", p. 17; Glossary "Immunity", p. 183 | happy |
| FR-024 | Immunity MUST override any adjustment, Resistance or Vulnerability to the same type, so the result is 0 in every combination. | SRD 5.2.1, "Immunity", p. 17 | unhappy (rule interaction) |
| FR-025 | The damage resolution MUST report the incoming amount, the amount after adjustments, after Resistance, after Vulnerability, and whether Immunity applied, so a consumer can show the breakdown. A package-design requirement; the SRD defines only the order. | `doc/rules/architecture.md` §6 (design); SRD 5.2.1, "Order of Application", p. 17 | happy |
| FR-099 | Inside the adjustment step, flat adjustments MUST be applied first (floored at 0) and then `HalfDamage` (rounded down). The damage that "would be dealt on a failed save" is read as the amount after the flat adjustments and before Resistance and Vulnerability, because the SRD places adjustments and multipliers in one step before Resistance (Q-5). | SRD 5.2.1, "Order of Application", p. 17; "Half Damage", p. 16 | unhappy (rule interaction) |
| FR-112 | `DamageAffinities` is a struct, so it MUST expose three exported constructor functions (a struct value cannot be a Go constant): `NoAffinities()` (all sets empty; equal to the zero value, which is also valid), `ResistantToAllDamage()` (resistances `{AllDamage}`) and `ImmuneToAllDamage()` (immunities every `DamageType`); each call returns an independent value (FR-091). The mapping from a magic item's resilience (`content-magic-items.md` FR-087, resolved by `magic-item-rules.md` FR-073) is: `ResistantToAllDamage` to `ResistantToAllDamage()`, `NonmagicalDurability` to `NoAffinities()`, `impervious` to `ImmuneToAllDamage()`. The host or `magic-item-rules.md` performs the switch: `combat` declares and imports no `MagicItemResilience` (`environment-hazards-and-objects.md` FR-123); the values add no pipeline behaviour (FR-017, FR-022, FR-023). | SRD 5.2.1, "Magic Item Resilience", p. 206; "Resistance and Vulnerability", "Immunity", p. 17 | happy |
| FR-113 | `FlatAdjustment(amount, scope)` MUST take an optional `DamageScope` (default `AllDamage`, so an unscoped adjustment keeps applying to every `Damage`); a scoped one MUST change only a `Damage` whose type is within its scope, and leave other instances of the same command unchanged. The floor at 0 (FR-016) applies per instance. | SRD 5.2.1, "Order of Application", p. 17 (an adjustment such as a reduction of one type's damage precedes Resistance) | unhappy (rule interaction) |
| FR-114 | `ApplyDamage` MUST also accept an `unpreventable` list of `Damage` values (default empty; `damages` and `unpreventable` together non-empty, else an invalid-input error (`srderr.ErrInvalidInput`)). An unpreventable instance MUST skip flat adjustments, `HalfDamage`, Resistance and Immunity, still take Vulnerability, and then be summed with the other instances (FR-085) and absorbed by Temporary Hit Points first. The host maps `UnpreventableDamage(type, amount)` (`wish-spell.md` FR-044) to `Damage(amount, type)`; this spec imports nothing from `character`. Interpretation (Q-19): "can't be reduced or prevented" bars only decreases, and Vulnerability is an increase; Temporary Hit Points are lost first by rule, not by reducing the damage. | SRD 5.2.1, "Wish", p. 175; "Order of Application", p. 17; "Lose Temporary Hit Points First", p. 18 | unhappy (rule interaction) |

Table 4.3a, damage pipeline (one test per row; "res" Resistance, "vuln" Vulnerability):

| Label | Incoming | Type | Flat adjustment | Affinities | Expected | Source | Path |
|---|---|---|---|---|---|---|---|
| worked example | 28 | Fire | -5 | res Fire, vuln Fire | 22 (28, 23, 11, 22) | SRD 5.2.1, "Order of Application", p. 17 | unhappy (rule interaction) |
| resistance only | 15 | Fire | 0 | res Fire | 7 | "Resistance and Vulnerability", p. 17 | happy |
| resistance rounds to zero | 1 | Fire | 0 | res Fire | 0 | same | unhappy (boundary) |
| vulnerability only | 15 | Cold | 0 | vuln Cold | 30 | same | happy |
| both on odd amount | 7 | Fire | 0 | res Fire, vuln Fire | 6 (7, 3, 6) | "Order of Application", p. 17 | unhappy (rule interaction) |
| unrelated type | 9 | Acid | 0 | res Fire | 9 | "Resistance and Vulnerability", p. 17 | unhappy (rule interaction) |
| resistance to all damage | 9 | Necrotic | 0 | res all damage | 4 | "No Stacking", p. 17 | happy |
| resistance to type and to all | 9 | Necrotic | 0 | res Necrotic, res all damage | 4 | "No Stacking", p. 17 | unhappy (rule interaction) |
| bonus before resistance | 15 | Fire | +3 | res Fire | 9 | "Order of Application", p. 17 | unhappy (rule interaction) |
| penalty floors at zero | 3 | Fire | -5 | none | 0 | "Damage Rolls", p. 16 | unhappy (boundary) |
| penalty floors before vulnerability | 3 | Fire | -5 | vuln Fire | 0 | "Order of Application", p. 17 | unhappy (boundary) |
| zero damage | 0 | Fire | 0 | vuln Fire | 0 | "Damage Rolls", p. 16 | unhappy (empty and minimal) |
| immunity | 12 | Fire | 0 | immune Fire | 0 | "Immunity", p. 17 | happy |
| immunity beats vulnerability | 12 | Fire | +4 | immune Fire, vuln Fire | 0 | "Immunity", p. 17 | unhappy (rule interaction) |
| half damage then resistance | 15 | Fire | half (save) | res Fire | 3 (15, 7, 3) | "Half Damage", p. 16; Q-5 | unhappy (rule interaction) |
| half damage with flat adjustment | 15 | Fire | +3 then half | none | 9 (15, 18, 9) | FR-099 | unhappy (rule interaction) |
| half damage with vulnerability | 15 | Fire | half | vuln Fire | 14 (15, 7, 14) | "Order of Application", p. 17 | unhappy (rule interaction) |
| half damage with immunity | 15 | Fire | half | immune Fire | 0 | "Immunity", p. 17 | unhappy (rule interaction) |
| half of one | 1 | Fire | half | vuln Fire | 0 (1, 0, 0) | "Half Damage", p. 16 | unhappy (boundary) |

### 4.4 Temporary Hit Points

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-026 | Damage MUST reduce Temporary Hit Points first, and any leftover damage MUST carry over to current Hit Points (table 4.6a, 5 temporary and 7 damage leaves 0 temporary and loses 2 Hit Points). | SRD 5.2.1, "Lose Temporary Hit Points First", p. 18 | happy |
| FR-027 | Damage less than or equal to the Temporary Hit Points MUST leave current Hit Points unchanged. | SRD 5.2.1, "Lose Temporary Hit Points First", p. 18 | unhappy (boundary) |
| FR-028 | Gaining Temporary Hit Points while having none MUST set them to the amount gained. | SRD 5.2.1, "Temporary Hit Points", p. 18 | happy |
| FR-029 | Gaining Temporary Hit Points while already having some MUST NOT add them: the result MUST be exactly the current amount or the new amount, as the `TemporaryHitPointsChoice` states (10 held and 12 offered gives 12 or 10, never 22). An offer equal to the amount held, under either choice, MUST leave the state unchanged, emit no event and give result `no_effect`. | SRD 5.2.1, "They Don't Stack", p. 18 | unhappy (rule interaction) |
| FR-030 | Gaining Temporary Hit Points with an amount below 1 MUST return an error matching `srderr.ErrInvalidInput` (a host precondition, FR-094). | SRD 5.2.1, "Temporary Hit Points", p. 18 | unhappy (invalid input) |
| FR-031 | Temporary Hit Points MUST be gainable at full Hit Points, MUST NOT raise current Hit Points, and MUST NOT be reported as Healing. | SRD 5.2.1, "They're Not Hit Points or Healing", p. 18 | unhappy (rule interaction) |
| FR-032 | Healing MUST NOT restore or increase Temporary Hit Points. | SRD 5.2.1, "They're Not Hit Points or Healing", p. 18 | unhappy (rule interaction) |
| FR-033 | Receiving Temporary Hit Points at 0 Hit Points MUST NOT change the creature's `LifeState`; it stays Unconscious. | SRD 5.2.1, "They're Not Hit Points or Healing", p. 18 | unhappy (state error) |
| FR-034 | Finishing a Long Rest MUST end Temporary Hit Points (set them to 0), via an explicit operation the host calls, and depleting them by damage MUST leave 0. | SRD 5.2.1, "Duration", p. 18 | happy |
| FR-098 | `EndTemporaryHitPoints` MUST reject only a Dead creature (`CreatureIsDead`); for a creature with 0 Temporary Hit Points it MUST return `TemporaryHitPointsEndApplied` with the state unchanged and no event (result `no_effect`). | SRD 5.2.1, "Duration", p. 18; Glossary "Dead", p. 180 | unhappy (state error) |
| FR-084 | Damage that is greater than 0 after the pipeline and is fully absorbed by Temporary Hit Points MUST count as damage taken: it MUST emit `DamageTaken` with 0 Hit Points lost and the amount absorbed, and `TemporaryHitPointsEnded` if the Temporary Hit Points reach 0. It MUST NOT add a Death Saving Throw failure, MUST NOT change `LifeState`, and MUST NOT trigger Massive Damage (the remainder is 0). Result `absorbed`. | SRD 5.2.1, "Hit Points", p. 16 ("Whenever you take damage"); "Lose Temporary Hit Points First", p. 18; Q-2 | unhappy (rule interaction) |

### 4.5 Healing

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-035 | Healing MUST add the restored Hit Points to current Hit Points, never above the Hit Point maximum, and the result MUST report the amount actually regained (table 4.5a: 8 of healing at 14 of 20 regains 6). | SRD 5.2.1, "Healing", p. 17 | happy |
| FR-036 | Healing a creature at its Hit Point maximum MUST leave it unchanged and report 0 regained. | SRD 5.2.1, "Healing", p. 17 | unhappy (boundary) |
| FR-037 | Healing with a negative amount MUST return an error matching `srderr.ErrInvalidInput` (FR-094); Healing of 0 MUST change nothing, including not restoring a creature at 0 Hit Points. | SRD 5.2.1, "Healing", p. 17; "Falling Unconscious", p. 17 | unhappy (invalid input) |
| FR-038 | Regaining any Hit Points while at 0 Hit Points (Dying or Stable) MUST make the creature Alive, no longer Unconscious through the 0-Hit-Point rules, and MUST reset the Death Saving Throw tallies. | SRD 5.2.1, "Falling Unconscious", p. 17; "Death Saving Throws", p. 17 | happy |
| FR-039 | Healing a Dead creature MUST return a rejection (`CreatureIsDead`) and leave the state unchanged. | Glossary "Dead", p. 180 (also Q-7) | unhappy (state error) |
| FR-115 | Healing MUST take an optional `HealingCeiling` (closed: `NoHealingCeiling`, the default, and `HealingCeilingAt(HitPoints)`, 0 or more else an invalid-input error (`srderr.ErrInvalidInput`)). Current Hit Points MUST NOT rise above `min(maximum, ceiling)`: regained is `max(0, min(amount, cap - current))`, a current already at or above the cap is never lowered, and a cap of 0 at 0 Hit Points regains 0 and leaves Dying or Stable unchanged (FR-036 applies). The host sets it while an effect makes damage unhealable (`toolbox-poison.md` FR-047 `HealingCeiling`) and clears it when the effect ends; this spec imports nothing from `toolbox`. Interpretation (Q-20): the ceiling formula is the host's; this is only a generic cap. | SRD 5.2.1, "Pale Tincture" (damage "can't be healed by any means while the creature remains Poisoned"), p. 198; "Healing", p. 17 | unhappy (boundary) |

Table 4.5a, Healing (one test per case; SRD 5.2.1, "Healing", p. 17; maximum 20; format: current, healing gives new
current, regained): worked example 14, 8 gives 20, 6 (happy); exactly to maximum 14, 6 gives 20, 6 (boundary); one
short 14, 5 gives 19, 5 (boundary); already full 20, 3 gives 20, 0 (boundary); zero healing at 0 gives 0, still Dying,
0 ("Falling Unconscious", p. 17; boundary); one Hit Point at 0 gives 1, Alive, 1 (boundary).

### 4.6 Dropping to 0 Hit Points

A creature's `ZeroHitPointsRule` (`FallsUnconscious` or `DiesInstantly`) is supplied by the host when the
`HitPointStatus` is created (FR-087); there is no package default (Q-9). The remainder in Massive Damage is the total
damage after the pipeline, minus the Temporary Hit Points absorbed, minus the current Hit Points used up (Q-1).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | When damage reduces a creature with the `FallsUnconscious` rule to 0 Hit Points without killing it, the creature MUST be Dying with 0 successes and 0 failures, and the result MUST contain `DroppedToZeroHitPoints` and `FellUnconscious` events (table 4.9a). | SRD 5.2.1, "Falling Unconscious", p. 17 | happy |
| FR-041 | A creature with the `DiesInstantly` rule that drops to 0 Hit Points MUST be Dead with cause `MonsterDeath`. | SRD 5.2.1, "Monster Death", p. 17 | happy |
| FR-042 | A creature given the `FallsUnconscious` rule MUST be treated like a character whatever it is (a GM may treat a monster like a character): Dying at 0 Hit Points and making Death Saving Throws. Death Saving Throws follow the rule, not the kind of creature (FR-096). | SRD 5.2.1, "Monster Death", p. 17 | unhappy (rule interaction) |
| FR-043 | When damage reduces a creature with the `FallsUnconscious` rule to 0 Hit Points and damage remains, the creature MUST die with cause `MassiveDamage` if the remainder equals or exceeds the Hit Point maximum (table 4.6a: maximum 12, current 6, 18 damage kills). | SRD 5.2.1, "Massive Damage", p. 17 | unhappy (boundary) |
| FR-044 | A remainder one below the Hit Point maximum, and a remainder of 0, MUST leave the character Dying, not Dead. | SRD 5.2.1, "Massive Damage", p. 17 | unhappy (boundary) |
| FR-045 | Massive Damage MUST use the damage left after Temporary Hit Points and after Resistance, Vulnerability and Immunity, never the incoming amount. | SRD 5.2.1, "Lose Temporary Hit Points First", p. 18; "Order of Application", p. 17 | unhappy (rule interaction) |
| FR-046 | A creature whose Hit Point maximum reaches 0 MUST be Dead with cause `HitPointMaximumZero`, in any `LifeState` other than Dead. A `HitPointStatus` with a maximum of 0 and any other `LifeState` cannot exist (FR-106). | SRD 5.2.1, "Hit Point Maximum of 0", p. 17 | unhappy (boundary) |
| FR-047 | Reducing the Hit Point maximum by an amount MUST lower the maximum (not below 0), MUST lower current Hit Points to the new maximum if it is lower, and a negative reduction MUST return an error matching `srderr.ErrInvalidInput` (FR-094). | SRD 5.2.1, "Hit Point Maximum of 0", p. 17; "Hit Points", p. 16 | unhappy (invalid input) |
| FR-048 | Damage that is 0 after the pipeline, summed over all instances in the command (including by Immunity), MUST change nothing: no Hit Point loss, no `DamageTaken` event, and no Death Saving Throw failure. The result still carries the `DamageResolution`s. | SRD 5.2.1, "Immunity", p. 17; "Damage at 0 Hit Points", p. 18 | unhappy (empty and minimal) |
| FR-049 | Damage that is greater than 0 after the pipeline MUST emit `DamageTaken` carrying, per instance, the `DamageResolution`, and in total the amount absorbed by Temporary Hit Points and the Hit Points lost (plain `int`s, FR-102); a creature that becomes Bloodied by it MUST also emit `BecameBloodied` (FR-101). Absorbed damage is covered by FR-084. | SRD 5.2.1, "Hit Points", p. 16 ("might trigger other game effects") | happy |
| FR-050 | When a melee attack is delivered as `MeleeAttackKnockOut` and the damage, after the pipeline and after Temporary Hit Points, would reduce a creature to 0 Hit Points, the creature MUST instead have 1 Hit Point and stay Alive, a `KnockedOut` event MUST be emitted (the consumer applies the Unconscious condition), and Massive Damage MUST NOT be checked. This holds whatever the creature's `ZeroHitPointsRule`. | SRD 5.2.1, "Knocking Out a Creature", p. 17 | happy |
| FR-051 | `MeleeAttackKnockOut` MUST have no effect when the damage would not reduce the creature to 0 Hit Points, or when the creature is already at 0 Hit Points; the ordinary rules apply in both cases. | SRD 5.2.1, "Knocking Out a Creature", p. 17 | unhappy (boundary) |
| FR-052 | Damage delivered as any other source, or as a melee attack without knock-out, MUST follow FR-040 to FR-046. | SRD 5.2.1, "Knocking Out a Creature", p. 17 | unhappy (rule interaction) |
| FR-085 | `ApplyDamage` MUST take one or more `Damage` values from one source, with one `HitKind` and one `DamageDelivery`. Each `Damage` MUST go through the pipeline separately (Resistance and Vulnerability once per instance, adjustments applied to each). The post-pipeline amounts MUST be summed; the sum goes against Temporary Hit Points, then current Hit Points. The zero-Hit-Point rules (the Death Saving Throw failure of FR-062 and FR-063, and the Massive Damage test of FR-043 and FR-064) MUST be applied once per command, not once per instance, against the sum. An empty list MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Damage at 0 Hit Points", p. 18 ("If you take any damage"); "Massive Damage", p. 17; Q-6 | unhappy (rule interaction) |
| FR-087 | A `HitPointStatus` MUST be created from a maximum, an explicit `ZeroHitPointsRule` and `DamageAffinities` (no default). The initial state MUST be current = maximum, 0 Temporary Hit Points and `Alive`. `NewHitPointStatus` MUST return a `*HitPointStatusError` (matching `srderr.ErrInvalidInput`) holding `HitPointStatusProblem.MaximumBelowOne` for a maximum below 1 (FR-105). `RestoreHitPointStatus` takes `HitPoints`, a `LifeState`, the rule and the affinities, and enforces FR-089, FR-090 and FR-106 by returning a `*HitPointStatusError`; FR-088 is enforced by `NewDying` itself. | `doc/rules/architecture.md` §3, §6 (design); SRD 5.2.1, "Hit Point Maximum of 0", p. 17 | happy |
| FR-088 | `NewDying` MUST return an error matching `srderr.ErrInvalidInput` for a success or failure tally outside 0 to 2. | SRD 5.2.1, "Death Saving Throws", p. 17 ("third success", "third failure") | unhappy (invalid input) |
| FR-089 | A `HitPointStatus` with `Dead`, `Dying` or `Stable` and `current > 0` MUST be rejected: `RestoreHitPointStatus` returns a `*HitPointStatusError` with `DeadWithHitPoints`, `DyingWithHitPoints` or `StableWithHitPoints`. | SRD 5.2.1, "Falling Unconscious", p. 17; "Stabilizing a Character", p. 18; Glossary "Dead", p. 180 | unhappy (invalid input) |
| FR-090 | A `HitPointStatus` with `Alive` and `current == 0` MUST be rejected: `RestoreHitPointStatus` returns a `*HitPointStatusError` with `AliveAtZeroHitPoints`. A knocked-out creature is `Alive` at 1 Hit Point. | SRD 5.2.1, "Falling Unconscious", p. 17; "Knocking Out a Creature", p. 17 | unhappy (invalid input) |
| FR-091 | `DamageAffinities` MUST accept empty slices; a duplicate Damage Type or scope MUST collapse to one entry (FR-021); `resistances`, `vulnerabilities` and `immunities` MUST be defensively copied views; two equal sets MUST compare equal (`Equal`, order-independent). | `doc/rules/architecture.md` §3 (value object); SRD 5.2.1, "No Stacking", p. 17 | unhappy (empty and minimal) |
| FR-093 | `ReduceHitPointMaximum` with an amount of 0 MUST return `HitPointMaximumReductionApplied` with the state unchanged and no event (result `no_effect`). | SRD 5.2.1, "Hit Point Maximum of 0", p. 17 | unhappy (boundary) |
| FR-095 | `RecordExhaustionDeath` MUST make a creature that is not Dead Dead with cause `exhaustion` and emit `Died(exhaustion)`; for a Dead creature it MUST return `CreatureIsDead`. The host calls it when `conditions.md` FR-040 (`IsDeadFromExhaustion`) says Exhaustion reached level 6. | SRD 5.2.1, "Exhaustion", p. 181 (level 6 death); `conditions.md` FR-040 | happy |
| FR-096 | A `HitPointStatus` MUST expose `MakesDeathSavingThrows`, true if and only if `LifeState` is Dying. A consumer decides that a Death Saving Throw is due from this predicate, not from the kind of creature: a monster treated like a character is due one. `encounter-aggregate.md` FR-018 produces `DyingCombatantTurnStarted` from it. | SRD 5.2.1, "Monster Death", p. 17; "Death Saving Throws", p. 17 | unhappy (rule interaction) |
| FR-101 | `BecameBloodied` MUST be emitted when damage takes a creature from not Bloodied to Bloodied, both judged by `IsBloodied(current, HitPointMaximum(maximum))` (`derived-stats.md` FR-071) on the before and after current Hit Points, and the creature is not Dead afterwards. The domain builds `HitPointMaximum(maximum)` from package `character` only on this path: the creature is not Dead afterwards, so by FR-106 its maximum is at least 1 and the constructor cannot throw (Q-18). Temporary Hit Points never change the judgement. It MUST NOT be emitted when the creature was already Bloodied, when it dies, or for Healing (there is no "stopped being Bloodied" event: consumers read the predicate). | Glossary "Bloodied", p. 177; "Hit Points", p. 16 | unhappy (rule interaction) |
| FR-102 | `DamageTaken` MUST expose `HitPointsLost`, `AbsorbedByTemporary` and `total` (their sum) as plain `int`s. `total`, which includes the amount absorbed, is the damage taken, and is the one amount `combat-action-resolution.md` passes as a plain `int` to the Concentration rule in `spellcasting.md` (`shared-types-and-ownership.md` FR-011; Q-2). This spec imports nothing from `spellcasting.md`. | `shared-types-and-ownership.md` FR-011 (design); SRD 5.2.1, "Hit Points", p. 16 | happy |
| FR-103 | Every `HitPointEvent` MUST implement `CombatEvent` (`combat-foundations.md` FR-015 to FR-019): a constant snake_case `Kind()` string, unique in the package, equal to the snake_case of its class name (`damage_taken`, `dropped_to_zero_hit_points`, `fell_unconscious`, `knocked_out`, `became_bloodied`, `hit_points_restored`, `temporary_hit_points_gained`, `temporary_hit_points_ended`, `hit_point_maximum_reduced`, `hit_point_maximum_restored`, `death_saving_throw_made`, `became_stable`, `lost_stability`, `recovered`, `died`); and a defensively copied `Combatants()` slice that is exactly `[subject]` (FR-104), never empty and never holding a duplicate. Events are immutable values, comparable by value including `Kind()` and `Combatants()`, and carry no timestamp or display text. | `combat-foundations.md` FR-015, FR-017 to FR-019; `doc/rules/architecture.md` §3 | unhappy (immutability) |
| FR-104 | Every use case that takes a `HitPointStatus` MUST also take, in its command, the `CombatantID` of the creature (`subject`), and every domain operation that returns events (FR-109) MUST take the same `CombatantID` as an argument and stamp it on each event as `combatants == [subject]`. `HitPointStatus` itself holds no id, `RollStableRecoveryDelay` has no creature and no event, and the id never reaches a telemetry attribute (FR-082). Chosen over an optional id because `combat-foundations.md` FR-018 allows empty `combatants` only for events about no creature (Q-17). | `combat-foundations.md` FR-018; `doc/rules/architecture.md` §3 (typed ids), §4 (handoffs) | happy |
| FR-105 | The parsers (`NewHitPoints`, `NewHitPointStatus`, `RestoreHitPointStatus`) MUST return `(T, error)` with typed errors, never display text, and never panic for any input (`errors-and-results.md` FR-025; `combat-foundations.md` FR-023). They are aggregate parsers, so each lists **every** problem found in a deterministic order (`errors-and-results.md` FR-023): `NewHitPoints` returns a `*HitPointsError` whose `Problems` is a non-empty defensively copied slice of the enum `HitPointsProblem` in the fixed order `CurrentBelowZero`, `CurrentAboveMaximum`, `MaximumBelowZero`, `TemporaryBelowZero`. `NewHitPointStatus` and `RestoreHitPointStatus` return a `*HitPointStatusError` with the enum `HitPointStatusProblem` (`MaximumBelowOne` for `NewHitPointStatus`; `DeadWithHitPoints`, `DyingWithHitPoints`, `StableWithHitPoints`, `AliveAtZeroHitPoints`, `HitPointMaximumZeroNotDead` for `RestoreHitPointStatus`). Each such error matches `srderr.ErrInvalidInput` and is never built with an empty problem list (`errors-and-results.md` FR-024). The `MustX` helpers panic with the same error if and only if the parser fails (`errors-and-results.md` FR-026), except that a `Dying` tally outside 0 to 2 is rejected by `NewDying` itself (FR-088). These names differ from `HitPointRejection`, the operation-state rejection (`CreatureIsDead`, `NotDying`, `NotStable`), which names no field. | `doc/rules/architecture.md` §5, §6; `errors-and-results.md` FR-023 to FR-026; `combat-foundations.md` FR-023 | unhappy (invalid input) |
| FR-106 | A `HitPointStatus` with `maximum == 0` MUST be `Dead` with cause `HitPointMaximumZero`. `NewHitPointStatus` already rejects a maximum below 1 (FR-087); `RestoreHitPointStatus` MUST return a `*HitPointStatusError` with `HitPointMaximumZeroNotDead` for a maximum of 0 with any other `LifeState` or any other death cause. This keeps FR-046 true for every constructible status. | SRD 5.2.1, "Hit Point Maximum of 0", p. 17 | unhappy (invalid input) |

Table 4.6a, damage to a creature (one test per row; PC means a creature with `FallsUnconscious`, damage values are
after the pipeline, "other" is `OtherSource`):

| Label | Kind | Max | Current | Temp | Damage | Delivery | Expected | Source | Path |
|---|---|---|---|---|---|---|---|---|---|
| ordinary hit | PC | 20 | 20 | 0 | 5 | other | 15, Alive | "Hit Points", p. 16 | happy |
| exactly 0, no remainder | PC | 20 | 5 | 0 | 5 | other | 0, Dying | "Falling Unconscious", p. 17 | happy |
| worked example | PC | 12 | 6 | 0 | 18 | other | 0, Dead (massive) | "Massive Damage", p. 17 | unhappy (boundary) |
| remainder one below max | PC | 12 | 6 | 0 | 17 | other | 0, Dying | same | unhappy (boundary) |
| remainder 1 | PC | 12 | 6 | 0 | 7 | other | 0, Dying | same | unhappy (boundary) |
| temporary absorbs part | PC | 20 | 10 | 5 | 7 | other | temp 0, 8 Hit Points | "Lose Temporary Hit Points First", p. 18 | happy |
| damage equals temp | PC | 20 | 10 | 5 | 5 | other | temp 0, 10 Hit Points (FR-084) | same | unhappy (boundary) |
| damage below temp | PC | 20 | 10 | 5 | 4 | other | temp 1, 10 Hit Points (FR-084) | same | unhappy (boundary) |
| temp then drop | PC | 20 | 3 | 5 | 12 | other | 0, Dying (remainder 4) | same; "Massive Damage", p. 17 | unhappy (rule interaction) |
| temp then massive | PC | 10 | 3 | 5 | 18 | other | 0, Dead (remainder 10) | same | unhappy (rule interaction) |
| minimal maximum | PC | 1 | 1 | 0 | 1 | other | 0, Dying | "Falling Unconscious", p. 17 | unhappy (empty and minimal) |
| monster at 0 | `DiesInstantly` | 20 | 5 | 0 | 5 | other | 0, Dead (MonsterDeath) | "Monster Death", p. 17 | happy |
| monster treated like character | `FallsUnconscious` | 20 | 5 | 0 | 5 | other | 0, Dying | same | unhappy (rule interaction) |
| knock-out | PC | 20 | 5 | 0 | 5 | melee knock-out | 1, Alive, `KnockedOut` | "Knocking Out a Creature", p. 17 | happy |
| knock-out not needed | PC | 20 | 5 | 0 | 4 | melee knock-out | 1, no `KnockedOut` | same | unhappy (boundary) |
| knock-out beats massive | PC | 20 | 5 | 0 | 100 | melee knock-out | 1, `KnockedOut` | same | unhappy (rule interaction) |
| knock-out ignored at 0 | PC, Dying | 20 | 0 | 0 | 3 | melee knock-out | one failure | same; "Damage at 0 Hit Points", p. 18 | unhappy (state error) |
| ranged does not knock out | PC | 20 | 5 | 0 | 5 | other | 0, Dying | same | unhappy (rule interaction) |
| knock-out after temporary absorbs part | PC | 20 | 5 | 3 | 8 | melee knock-out | temp 0, 1, `KnockedOut` (leftover 5 would reach 0) | "Knocking Out a Creature", p. 17; "Lose Temporary Hit Points First", p. 18 | unhappy (rule interaction) |
| knock-out fully absorbed | PC | 20 | 5 | 10 | 8 | melee knock-out | temp 2, 5 Hit Points, no `KnockedOut` | same | unhappy (rule interaction) |
| knock-out of a monster | `DiesInstantly` | 20 | 5 | 0 | 5 | melee knock-out | 1, Alive, `KnockedOut`, no death | "Knocking Out a Creature", p. 17 | unhappy (rule interaction) |
| two types summed | PC | 12 | 6 | 0 | 10 Fire and 8 Cold | other | 0, Dead (massive; remainder 12 from the sum) | FR-085; "Massive Damage", p. 17 | unhappy (rule interaction) |
| two types, no massive | PC | 20 | 10 | 0 | 6 Fire and 5 Cold | other | 0, Dying (remainder 1, one drop) | FR-085 | unhappy (rule interaction) |

### 4.7 Death Saving Throws

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-053 | A Death Saving Throw MUST be accepted only for a Dying creature; for an Alive, Stable or Dead creature it MUST return a rejection (`NotDying` or `CreatureIsDead`) and leave the state unchanged. | SRD 5.2.1, "Death Saving Throws", p. 17; "Stable", p. 188 | unhappy (state error) |
| FR-054 | A Death Saving Throw whose natural d20 is 2 to 19 MUST be a success when its total (FR-061) is 10 or higher and a failure otherwise (table 4.7a). With no modifier the total is the natural d20, so 10 to 19 succeed and 2 to 9 fail. The SRD's "10 or higher" is overridden for a natural 20 by FR-056, and a natural 1 is FR-055. | SRD 5.2.1, "Death Saving Throws", p. 17; "Rolling a 1 or 20", p. 18; Q-4 | happy |
| FR-055 | A natural 1 on the d20 MUST count as two failures whatever the total. | SRD 5.2.1, "Rolling a 1 or 20", p. 18 | unhappy (boundary) |
| FR-056 | A natural 20 MUST give the creature 1 Hit Point, make it Alive and reset the tallies whatever the total, and MUST NOT record a success (a 20 never moves the creature towards Stable). | SRD 5.2.1, "Rolling a 1 or 20", p. 18 | unhappy (boundary) |
| FR-057 | The third success MUST make the creature Stable and reset both tallies to 0. | SRD 5.2.1, "Three Successes/Failures", p. 17 | unhappy (boundary) |
| FR-058 | The third failure, including failures from a natural 1 that take the tally to 3 or beyond, MUST make the creature Dead with cause `DeathSavingThrowFailures`. | SRD 5.2.1, "Three Successes/Failures", p. 17; "Rolling a 1 or 20", p. 18 | unhappy (boundary) |
| FR-059 | Successes and failures MUST accumulate without needing to be consecutive, and a success or failure MUST have no other effect by itself. | SRD 5.2.1, "Three Successes/Failures", p. 17 | unhappy (rule interaction) |
| FR-060 | Both tallies MUST be reset to 0 when the creature regains any Hit Points or becomes Stable. | SRD 5.2.1, "Three Successes/Failures", p. 17 | unhappy (state error) |
| FR-061 | A Death Saving Throw is a saving throw and so a D20 Test, but "isn't tied to an ability score": it adds no ability modifier or proficiency. Its **natural** is the kept d20 (the roll under Advantage or Disadvantage, kernel `RollMode.select`) and its **total** is the natural plus a host-supplied `Modifier` (default 0), which carries the Exhaustion reduction (`-2 x level`, p. 181) and any other effect the host states; the total is not clamped. The natural decides FR-055 and FR-056, the total decides FR-054. `combat-action-resolution.md` FR-069 builds the same total. The resolver MUST return an error matching `srderr.ErrInvalidInput` for a natural outside 1 to 20 as its own argument check; it is the only face check in this spec, because `DieRoll` returns an error matching `srderr.ErrInvalidInput` for an out-of-range face (`dice-and-rolls.md` FR-032). | SRD 5.2.1, "Death Saving Throws", p. 17; "D20 Test", p. 180; "Exhaustion", p. 181; Q-4 | unhappy (invalid input) |
| FR-062 | Damage taken at 0 Hit Points while Dying MUST add one failure, when Hit Points are lost after Temporary Hit Points absorb (FR-084 covers fully absorbed damage). | SRD 5.2.1, "Damage at 0 Hit Points", p. 18; Q-2 | happy |
| FR-063 | Damage taken at 0 Hit Points from a Critical Hit (`HitKind.CriticalHit`) MUST add two failures instead, once per command (FR-085), and after Temporary Hit Points. | SRD 5.2.1, "Damage at 0 Hit Points", p. 18 | unhappy (boundary) |
| FR-064 | Damage taken at 0 Hit Points whose remainder after Temporary Hit Points (summed per command, FR-085) equals or exceeds the Hit Point maximum MUST kill the creature with cause `MassiveDamage`, even when it would add fewer than three failures. | SRD 5.2.1, "Damage at 0 Hit Points", p. 18 | unhappy (boundary) |
| FR-065 | Damage that takes the failure tally to 3 or beyond MUST make the creature Dead with cause `DeathSavingThrowFailures`. | SRD 5.2.1, "Damage at 0 Hit Points", p. 18; "Three Successes/Failures", p. 17 | unhappy (boundary) |
| FR-066 | Damage to a Dead creature MUST return a rejection (`CreatureIsDead`) and leave the state unchanged. | Glossary "Dead", p. 180 | unhappy (state error) |

Table 4.7a, Death Saving Throw (one test per row; S = successes, F = failures; "d20" is the natural and the total equals it
unless a total is given):

| Label | d20 | Before | After | Source | Path |
|---|---|---|---|---|---|
| Exhaustion turns a success into a failure | 11, total 7 (level 2) | S0 F0 | S0 F1 | FR-061; "Exhaustion", p. 181 | unhappy (rule interaction) |
| total of 10 exactly after reduction | 14, total 10 | S0 F0 | S1 F0 | FR-054 | unhappy (boundary) |
| natural 20 ignores a low total | 20, total 4 | S0 F2 | 1 Hit Point, Alive, S0 F0 | FR-056 | unhappy (boundary) |
| natural 1 with a high total | 1, total 12 | S0 F0 | S0 F2 | FR-055 | unhappy (boundary) |
| Advantage keeps the higher die | dice 4 and 15, `advantage`, total 15 | S0 F0 | S1 F0 | FR-061; "Roll Two D20s", p. 8 | unhappy (rule interaction) |
| minimum failure range | 2 | S0 F0 | S0 F1 | "Death Saving Throws", p. 17 | unhappy (boundary) |
| highest failure | 9 | S0 F0 | S0 F1 | same | unhappy (boundary) |
| lowest success | 10 | S0 F0 | S1 F0 | same | unhappy (boundary) |
| high success | 19 | S1 F1 | S2 F1 | same | happy |
| natural 1 | 1 | S0 F0 | S0 F2 | "Rolling a 1 or 20", p. 18 | unhappy (boundary) |
| natural 1 at one failure | 1 | S0 F1 | Dead | same | unhappy (boundary) |
| natural 1 at two failures | 1 | S0 F2 | Dead | same | unhappy (boundary) |
| natural 20 | 20 | S1 F2 | 1 Hit Point, Alive, S0 F0 | same | unhappy (boundary) |
| natural 20 records no success | 20 | S2 F0 | 1 Hit Point, Alive, S0 F0, not Stable, no `BecameStable` | FR-056 | unhappy (boundary) |
| third success | 10 | S2 F2 | Stable, S0 F0 | "Three Successes/Failures", p. 17 | unhappy (boundary) |
| third failure | 9 | S2 F2 | Dead | same | unhappy (boundary) |
| interleaved | 12, 4, 15, 3, 18 | S0 F0 | Stable after the fifth roll | same | unhappy (rule interaction) |
| face out of range (resolver) | 0 and 21 | any | an invalid-input error (`srderr.ErrInvalidInput`) | n/a | unhappy (invalid input) |

Table 4.7b, damage at 0 Hit Points (one test per row; maximum 20, Dying; damage is after the pipeline, per command):

| Label | Damage | Critical Hit | Before | After | Source | Path |
|---|---|---|---|---|---|---|
| any damage | 3 | no | S0 F0 | S0 F1 | "Damage at 0 Hit Points", p. 18 | happy |
| Critical Hit | 3 | yes | S0 F0 | S0 F2 | same | unhappy (boundary) |
| Critical Hit at two failures | 3 | yes | S0 F2 | Dead | same | unhappy (boundary) |
| third failure | 1 | no | S0 F2 | Dead | same | unhappy (boundary) |
| one below maximum | 19 | no | S0 F0 | S0 F1 | same | unhappy (boundary) |
| equals maximum | 20 | no | S0 F0 | Dead (massive) | same | unhappy (boundary) |
| exceeds maximum | 25 | no | S0 F0 | Dead (massive) | same | unhappy (boundary) |
| successes kept | 3 | no | S2 F0 | S2 F1 | "Three Successes/Failures", p. 17 | unhappy (rule interaction) |
| fully absorbed by Temporary Hit Points | 3 (temp 5) | no | S0 F0 | unchanged, temp 2, `DamageTaken` with 0 lost, no failure (FR-084) | "Lose Temporary Hit Points First", p. 18; Q-2 | unhappy (rule interaction) |
| partly absorbed | 8 (temp 5) | no | S0 F0 | temp 0, S0 F1 (leftover 3) | same; FR-062 | unhappy (rule interaction) |
| Massive Damage uses the leftover | 24 (temp 5) | no | S0 F0 | S0 F1 (leftover 19 is below 20) | FR-064 | unhappy (boundary) |
| Massive Damage after temporary | 25 (temp 5) | no | S0 F0 | Dead (massive; leftover 20) | FR-064 | unhappy (boundary) |
| Immunity | 0 | no | S0 F0 | unchanged | "Immunity", p. 17 | unhappy (empty and minimal) |
| two types, Critical Hit | 3 Fire and 4 Cold | yes | S0 F0 | S0 F2 (not four) | FR-085; FR-063 | unhappy (rule interaction) |
| two types, summed Massive Damage | 12 Fire and 8 Cold | no | S0 F0 | Dead (massive; sum 20) | FR-085; FR-064 | unhappy (rule interaction) |

### 4.8 Stabilizing and Stable

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-067 | Stabilizing a Dying creature MUST make it Stable with both tallies reset to 0. The success of the Help-action check is the host's input, not decided here. | SRD 5.2.1, "Stabilizing a Character", p. 18 | happy |
| FR-068 | Stabilizing a creature that is Alive or Stable MUST return `NotDying`, and a Dead one `CreatureIsDead`, leaving the state unchanged. | SRD 5.2.1, "Stabilizing a Character", p. 18; Glossary "Stable", p. 188 | unhappy (state error) |
| FR-069 | The package MUST publish the stabilizing check, `StabilizingCheck`, as DC 10 (`DifficultyClass`, `d20-tests.md`), Wisdom (`Ability`, `ability-scores.md`), Medicine (`Skill`, `content-schema-and-decoder.md` FR-106, with its `Ability`). | SRD 5.2.1, "Stabilizing a Character", p. 18 | happy |
| FR-070 | A Stable creature MUST have 0 Hit Points, make no Death Saving Throws, and still be Unconscious; a Dying creature MUST also be Unconscious; an Alive or Dead creature MUST NOT report Unconscious from this state. A knocked-out creature (1 Hit Point, Alive) is Unconscious through the `KnockedOut` event and `conditions.md`, not through this predicate. | SRD 5.2.1, "Stabilizing a Character", p. 18; "Falling Unconscious", p. 17 | unhappy (state error) |
| FR-071 | Damage that takes Hit Points from a Stable creature MUST make it Dying (no longer Stable) with the tallies at 0 before the damage is counted, and the damage MUST then count under FR-062 or FR-063, emitting `LostStability`; Massive Damage (FR-064) applies too (Q-3, interpretation: the creature "takes any damage while you have 0 Hit Points", so both texts apply). | SRD 5.2.1, "Stabilizing a Character", p. 18; "Damage at 0 Hit Points", p. 18 | unhappy (rule interaction) |
| FR-072 | `RollStableRecoveryDelay` MUST request exactly `1d4` through the `dice.Roller` and return `StableRecoveryDelayRolled` holding a `StableRecoveryDelay` whose `hours` is the face, 1 to 4 (table 4.8a). | SRD 5.2.1, "Stabilizing a Character", p. 18 | unhappy (boundary) |
| FR-073 | Recovering a Stable creature MUST set it to 1 Hit Point and Alive; applied to a creature that is not Stable it MUST return `NotStable`, or `CreatureIsDead` for a Dead one. | SRD 5.2.1, "Stabilizing a Character", p. 18 | unhappy (state error) |
| FR-074 | Healing a Stable creature MUST follow FR-038, so it can no longer be recovered under FR-073. | SRD 5.2.1, "Stabilizing a Character", p. 18 | unhappy (rule interaction) |
| FR-075 | A Dead creature MUST reject temporary-Hit-Point gain, Hit Point maximum reduction, stabilizing, Death Saving Throws, stable recovery and `RecordExhaustionDeath` with `CreatureIsDead`, leaving the state unchanged (Q-7). | Glossary "Dead", p. 180 | unhappy (state error) |

Table 4.8a, stable recovery delay (one test per case; SRD 5.2.1, "Stabilizing a Character", p. 18; boundary): d4 face
1 gives 1 hour; d4 face 4 gives 4 hours.

### 4.9 Events per transition

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-097 | Each operation MUST emit exactly the events of table 4.9a, in that order (one test per transition: each alternative separated by `/` or `;` in a row is its own test). Items in brackets are conditional. | SRD 5.2.1, "Hit Points", p. 16; `doc/rules/architecture.md` §3 (domain events) | happy |

Table 4.9a (events in order; brackets are conditional):

| Transition | Events in order |
|---|---|
| damage, Hit Points lost, creature stays above 0 | `DamageTaken`, [`TemporaryHitPointsEnded`], [`BecameBloodied`] |
| damage fully absorbed by Temporary Hit Points (FR-084) | `DamageTaken` (0 lost), [`TemporaryHitPointsEnded`] |
| damage to 0, falls unconscious | `DamageTaken`, [`TemporaryHitPointsEnded`], [`BecameBloodied`], `DroppedToZeroHitPoints`, `FellUnconscious` |
| damage to 0, dies (monster death or Massive Damage) | `DamageTaken`, [`TemporaryHitPointsEnded`], `DroppedToZeroHitPoints`, `Died` |
| damage to 0, knocked out | `DamageTaken`, [`TemporaryHitPointsEnded`], [`BecameBloodied`], `KnockedOut` |
| damage at 0, Dying: a failure added / dies (third failure or Massive Damage) | `DamageTaken`, [`TemporaryHitPointsEnded`] / `DamageTaken`, [`TemporaryHitPointsEnded`], `Died` |
| damage at 0, Stable | `DamageTaken`, [`TemporaryHitPointsEnded`], `LostStability`, [`Died`] |
| damage 0 after the pipeline; Healing with nothing regained; gain Temporary Hit Points with current kept or an equal amount (FR-029); reduce maximum by 0; restore a maximum at or below the current one | none |
| Healing, Hit Points regained: creature above 0 / at 0 | `HitPointsRestored` / `HitPointsRestored`, `Recovered` |
| Death Saving Throw: still Dying / third success | `DeathSavingThrowMade` / `DeathSavingThrowMade`, `BecameStable` |
| Death Saving Throw: third failure or natural 1 beyond 2 / natural 20 | `DeathSavingThrowMade`, `Died` / `DeathSavingThrowMade`, `HitPointsRestored` (1), `Recovered` |
| stabilize; stable recovery | `BecameStable`; `HitPointsRestored` (1), `Recovered` |
| gain Temporary Hit Points, amount differs from the held amount and is kept (FR-029); end Temporary Hit Points, some held | `TemporaryHitPointsGained`; `TemporaryHitPointsEnded` |
| reduce maximum by more than 0, maximum stays above 0 / reaches 0; record Exhaustion death; restore a maximum raised above the current one | `HitPointMaximumReduced` / `HitPointMaximumReduced`, `Died`; `Died`; `HitPointMaximumRestored` |

### 4.10 Purity, ports and telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-076 | Every operation, and every use case, MUST return a new state plus events and MUST NOT mutate its inputs; returned event lists and collections MUST be defensively copied. | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |
| FR-077 | Given the same state, command and fixed rolls, every operation and use case MUST return an equal result; the domain MUST NOT read the clock or randomness. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (determinism) |
| FR-078 | Use cases MUST obtain every roll through `dice.Roll`. When the `dice.Roller` returns a `*srderr.PortError` for `dice.Port` (reason `unavailable`), the use case MUST propagate it unchanged (the same instance) and return no new state. | `doc/rules/architecture.md` §4; `doc/rules/testing.md` §4; `dice-and-rolls.md` FR-049, FR-052 | unhappy (collaborator failure) |
| FR-092 | When the `dice.Roller` returns a result for the wrong expression (a different count, die or modifier), `MakeDeathSavingThrow` and `RollStableRecoveryDelay` MUST returns a `*srderr.PortError` for `dice.Port` with reason `InvalidResult` and return no new state (`dice.Roll` compares the expression only, `dice-and-rolls.md` FR-050, FR-051). A face outside the die is unrepresentable (`dice-and-rolls.md` FR-032), so it is not a case here. A fixed roller MUST show that `MakeDeathSavingThrow` requested exactly `1d20` once (`2d20` once under Advantage or Disadvantage, FR-061; the modifier is never part of the expression) and `RollStableRecoveryDelay` exactly `1d4` once. | `doc/rules/testing.md` §4 ("collaborator returns invalid data"); `dice-and-rolls.md` FR-032, FR-050, FR-051 | unhappy (collaborator failure) |
| FR-094 | Command amounts (Healing, Temporary Hit Points, a maximum reduction, a `Damage` amount) are programmer-supplied preconditions, not untrusted input: a violation returns an error matching `srderr.ErrInvalidInput` (FR-030, FR-037, FR-047, FR-085 empty `Damage` list, FR-086). A host that passes a rolled total, which can be negative (for example `1d4 - 5`), MUST clamp it to 0 first; Healing and a maximum reduction of 0 are valid no-ops, and Temporary Hit Points of 0 are skipped by the host. | `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-079 | Each use case MUST run its work through `CombatTelemetry` (`combat-foundations.md` FR-035) with its `CombatOperation` (the snake_case use-case name, FR-028 there). `CombatTelemetry` records, once per call, the shared span `srd5e.combat.<operation>`, `srd5e.combat.operations`, `srd5e.combat.operation_duration`, `srd5e.combat.events` per event, with the shared attributes `srd5e.combat.operation`, `srd5e.outcome`, `srd5e.combat.result` and `srd5e.combat.result_class` (`combat-foundations.md` FR-029, FR-030). This spec defines none of those and no span of its own; names are constants in `CombatSignals` (FR-027 there). Its own signals are in section 7. | `doc/rules/observability.md` §5, §6; `combat-foundations.md` FR-027 to FR-030, FR-035 | happy |
| FR-080 | A refused result (FR-107: `CreatureIsDead`, `NotDying`, `NotStable`) is a normal rule outcome, not an error (`doc/rules/observability.md` §7; `combat-foundations.md` FR-031, D-4). It MUST be recorded with `srd5e.outcome` = `failure`, `srd5e.combat.result_class` = `refused` and **no** span error status, and MUST NOT increment `srd5e.combat.errors` or carry an `error.type` (`combat-foundations.md` FR-032; Q-16). A returned `*srderr.PortError` for `dice.Port` MUST be recorded once on the span and once in `srd5e.combat.errors` with `error.type` = `srderr.ErrorType(err)` (`errors-and-results.md` FR-005, FR-034; `dice-and-rolls.md` FR-049), and MUST NOT also be logged. No use case in this spec emits a log (`combat-foundations.md` FR-034; `shared-types-and-ownership.md` FR-043). | `doc/rules/observability.md` §6, §7; `combat-foundations.md` FR-031, FR-032; `shared-types-and-ownership.md` FR-041 | unhappy (telemetry on failure) |
| FR-107 | Every result variant MUST be classified for `CombatTelemetry` as one `CombatResultClass`, and MUST carry a `srd5e.combat.result` value from the closed `snake_case` set of table 4.10a, one set per use case, each value mapped to exactly one class (`combat-foundations.md` FR-030, FR-031). The value is not derived from the variant class name and is finer than `result_class`; the use case obtains it from the variant and its events with the first matching rule of the table. No variant here is `invalid`: unusable host inputs are programmer errors (an invalid-input error (`srderr.ErrInvalidInput`), FR-094). Every `...Rejected` variant, which holds a `HitPointRejection`, is `refused`. | `combat-foundations.md` FR-030, FR-031; `doc/rules/architecture.md` §6 | unhappy (rule interaction) |
| FR-081 | Every use case MUST return identical results with the default zero-value `telemetry.Noop` (`shared-types-and-ownership.md` FR-036) and with a recording one. | `doc/rules/observability.md` §9 | unhappy (determinism) |
| FR-082 | Attributes MUST be low-cardinality: Damage Type, outcome, result, result class, event kind, refusal reason, death cause, operation name; never creature ids (`CombatantID`), amounts or free text (`combat-foundations.md` FR-033). | `doc/rules/observability.md` §5 | unhappy (telemetry) |
| FR-083 | Package `combat` MUST export every use case and every handoff type listed in section 6, each with a godoc comment. The architecture test derives its expected export list from sections 5 and 6 together with the variants named in FR-103, FR-105 and FR-109. | `doc/rules/architecture.md` §4, §7 | happy |
| FR-108 | Each use case MUST emit its own signals (section 7) once per call and only for a returned result (never on a returned error); attribute values are the lowercase `snake_case` forms below. `srd5e.combat.damage_resolved`: `ApplyDamage` increments once per `Damage` instance of an applied command, with `srd5e.combat.damage_type` and `srd5e.combat.damage_effect` chosen by the first matching rule of table 4.10b from that instance's `DamageResolution` (FR-025). `srd5e.combat.hit_points_lost`: `ApplyDamage` records one sample per applied command: the `HitPointsLost` of the `DamageTaken` event, or 0 when the command has no `DamageTaken` (FR-048); none for a rejection. `srd5e.combat.hit_points_restored`: `RestoreHitPoints` records one sample per applied command: the amount regained (0 included, FR-036); none for a rejection. `srd5e.combat.death_saves`: `MakeDeathSavingThrow` increments once per applied call, with `srd5e.combat.death_save_result` mapped from the `DeathSavingThrowOutcome`: `DeathSavingThrowSuccess` to `success`, `DeathSavingThrowFailure` to `failure`, `DeathSavingThrowTwoFailures` to `critical_failure`, `DeathSavingThrowRegainedHitPoint` to `critical_success`. `srd5e.combat.hit_point_refusals`: every use case that can return a `...Rejected` increments once per rejection, with `srd5e.combat.operation` and `srd5e.combat.refusal_reason` mapped from the `HitPointRejection`: `CreatureIsDead` to `creature_is_dead`, `NotDying` to `not_dying`, `NotStable` to `not_stable`. `srd5e.combat.deaths`: increments once per `Died` event in an applied result, with `srd5e.combat.death_cause` mapped from `DeathCause`: `MonsterDeath` to `monster_death`, `HitPointMaximumZero` to `hit_point_maximum_zero`, `MassiveDamage` to `massive_damage`, `DeathSavingThrowFailures` to `death_saving_throw_failures`, `exhaustion` to `exhaustion`. No attribute carries an amount or a `CombatantID` (FR-082). | `doc/rules/observability.md` §5, §9; `combat-foundations.md` FR-029, FR-030, FR-033 | happy |
| FR-109 | The rules MUST be available as **pure domain operations** on `HitPointStatus`, exported from `combat`, so downstream domain code composes them without calling a use case (`doc/rules/architecture.md` §2, §4; `shared-types-and-ownership.md` FR-044). Each takes the state, the `subject` (FR-104) and its inputs, and returns a `HitPointTransition`: closed, `HitPointTransitionApplied(status, events, resolutions, HitPointsRegained)` (the new status, the defensively copied ordered events of table 4.9a, the per-instance `DamageResolution`s for damage and an empty list otherwise, and the Hit Points actually regained for Healing and 0 otherwise) or `HitPointTransitionRefused(rejection)` (a `HitPointRejection`, state unchanged). The operations are `ApplyDamage(subject, damages, adjustments, delivery, HitKind, unpreventable)` (FR-085, FR-114), `Restore(subject, amount, ceiling)` (FR-115), `RestoreMaximum(subject, NormalMaximum)` (FR-116, no use case), `GainTemporary(subject, amount, choice)`, `EndTemporary(subject)`, `ApplyDeathSave(subject, natural, total)` (the kept d20, 1 to 20, and the total, FR-061), `Stabilize(subject)`, `RecoverStable(subject)`, `ReduceMaximum(subject, amount)` and `RecordExhaustionDeath(subject)`. Each enforces exactly the requirements of the matching use case (`RestoreMaximum` has none); every use case is a thin wrapper that obtains the roll through `dice.Roller` where it needs one, calls the operation and classifies the result (FR-107). `DamageResolver` (the pipeline, FR-016 to FR-025 and FR-099) stays internal and is what `ApplyDamage` calls; the "Hit Point arithmetic" that `attacks.md` expects is `HitPointStatus.ApplyDamage`. `encounter-aggregate.md` FR-021 replaces the status with the `status` of the transition and wraps its `events`. | `doc/rules/architecture.md` §2, §3, §4; `shared-types-and-ownership.md` FR-044 | happy |
| FR-110 | No use case of this spec MUST call another use case, hold one, or accept one as a collaborator: each composes the pure operations of FR-109 and its own ports, and records only its own signals. A downstream spec composes the FR-109 operations, never these use cases. | `shared-types-and-ownership.md` FR-044; `doc/rules/architecture.md` §4 | unhappy (forbidden edge) |
| FR-116 | `RestoreMaximum(subject, NormalMaximum)` MUST return a `HitPointTransition` that returns a reduced Hit Point maximum to the host-supplied normal maximum: for a Dead creature `HitPointTransitionRefused(CreatureIsDead)`; for `NormalMaximum` at or below the current maximum, `Applied` with the state unchanged and no event (a maximum is never lowered, so one raised by an effect is not cut); otherwise the maximum becomes `NormalMaximum`, current Hit Points, `LifeState` and Temporary Hit Points stay as they are (current is already within the new maximum) and one `HitPointMaximumRestored` event (`PreviousMaximum`, `maximum`) is emitted. `NormalMaximum` below 1 MUST return an error matching `srderr.ErrInvalidInput` (FR-094). It is the inverse of `ReduceMaximum` for `rests-and-hit-dice.md` FR-027; it has no use case, signal or `srd5e.combat.result` value, and the host composes it (FR-110). Interpretation (Q-21): the SRD says only "returns to normal", so the host states the normal value and a Long Rest does not also heal here. | SRD 5.2.1, "Long Rest" (Regain All HP), p. 185; "Hit Point Maximum of 0", p. 17 | unhappy (boundary) |
| FR-117 | The following MUST be exported by `combat` for consumers that rebuild states and events (`monster-traits.md` FR-030, FR-034, FR-045, FR-046). (1) `DamageResolution` exposes `Type()` (the `DamageType`), `Dealt()` (the amount after the pipeline, 0 under Immunity) and `Unpreventable()` besides the breakdown of FR-025. (2) `DroppedToZeroHitPoints`, `Died`, `FellUnconscious`, `BecameBloodied` and every other `HitPointEvent` have public constructors taking `subject` (FR-104) and their data, building the same events the operations emit. (3) `RestoreHitPointStatus` (FR-087) can build `Alive`, `Dying`, `Stable` and `Dead` statuses from given `HitPoints`, within FR-089, FR-090 and FR-106. | `monster-traits.md` FR-030, FR-034; `doc/rules/architecture.md` §3, §4 (design) | happy |
| FR-118 | `combat` MUST NOT import species behaviour; three handoffs from `species-trait-behaviour.md` are host mappings. (1) Each `DamageType` of `SpeciesPassives.resistances` (FR-010 there) becomes a `DamageOfType` entry of `DamageAffinities.resistances`, merged with other sources (duplicates collapse: FR-021, FR-091). (2) A `TemporaryHitPointsGrant(amount)` (Adrenaline Rush) becomes `GainTemporaryHitPoints` with that amount; the host picks the `TemporaryHitPointsChoice` and FR-029 applies. (3) For `UseRelentlessEndurance` the host maps the `DamageApplied` events to `ZeroHitPointsOutcome`: any `Died` (`MonsterDeath`, `HitPointMaximumZero`, `MassiveDamage`) gives `ZeroHitPointsKilledOutright`; `DroppedToZeroHitPoints` without `Died` gives `ZeroHitPointsNotKilled`; a `KnockedOut` (1 Hit Point already) never reaches the trait. When the trait is used the host rebuilds the status as `Alive` with 1 Hit Point through `restore` (FR-117). Interpretation: all three `Died` causes are "Instant Death" (p. 17), the SRD's "killed outright". | SRD 5.2.1, "Resistance and Vulnerability", "No Stacking", "Instant Death", p. 17; "They Don't Stack", p. 18; "Relentless Endurance", p. 86 | unhappy (rule interaction) |
| FR-111 | `HitKind` MUST have exactly two members, `NormalHit` and `CriticalHit`, exhaustive in a `switch`, and is the only place in `combat` that restates `AttackRollResult.IsCriticalHit` for damage. It MUST NOT be used as an attack outcome. `attacks.md` and `combat-action-resolution.md` map `IsCriticalHit` to `HitKind` once, when they hand damage to the pure `ApplyDamage` operation; only a host calling the `ApplyDamage` use case directly hands the `HitKind` to it (`shared-types-and-ownership.md` FR-004). | `shared-types-and-ownership.md` FR-004; SRD 5.2.1, "Damage at 0 Hit Points", p. 18 | happy |

Table 4.10a, `srd5e.combat.result` per use case. Rules are listed in order and the first match wins; every rule gives
class `applied` (`success`) except a `...Rejected` variant, which gives class `refused` (`failure`, no error status):

| Use case | Ordered rules: variant and condition -> `srd5e.combat.result` |
|---|---|
| `apply_damage` | `DamageApplied` with `Died` -> `killed`; with `KnockedOut` -> `knocked_out`; with `DroppedToZeroHitPoints` -> `dropped_to_zero`; with `LostStability` -> `lost_stability`; with `DamageTaken` and 0 Hit Points lost (FR-084) -> `absorbed`; with `DamageTaken` and more than 0 lost -> `damaged`; no `DamageTaken` (FR-048) -> `no_effect`; `DamageRejected` -> `rejected_dead` (refused) |
| `restore_hit_points` | `HealingApplied` with `Recovered` -> `recovered`; with `HitPointsRestored` -> `healed`; nothing regained (FR-036, FR-037) -> `no_effect`; `HealingRejected` -> `rejected_dead` (refused) |
| `gain_temporary_hit_points` | `TemporaryHitPointsGainApplied` with `TemporaryHitPointsGained` -> `gained`; current kept or an equal amount offered, no event (FR-029) -> `no_effect`; `TemporaryHitPointsGainRejected` -> `rejected_dead` (refused) |
| `end_temporary_hit_points` | `TemporaryHitPointsEndApplied` with `TemporaryHitPointsEnded` -> `ended`; none held (FR-098) -> `no_effect`; `TemporaryHitPointsEndRejected` -> `rejected_dead` (refused) |
| `make_death_saving_throw` | `DeathSavingThrowApplied` with `Died` -> `killed`; with `Recovered` (natural 20) -> `recovered`; with `BecameStable` -> `stabilized`; outcome `DeathSavingThrowSuccess` -> `succeeded`; outcome `DeathSavingThrowFailure` or `DeathSavingThrowTwoFailures` -> `failed`; `DeathSavingThrowRejected` holding `NotDying` -> `not_dying` (refused); holding `CreatureIsDead` -> `rejected_dead` (refused) |
| `stabilize_creature` | `StabilizationApplied` -> `stabilized`; `StabilizationRejected` holding `NotDying` -> `not_dying` (refused); holding `CreatureIsDead` -> `rejected_dead` (refused) |
| `roll_stable_recovery_delay` | `StableRecoveryDelayRolled` -> `rolled` |
| `recover_stable_creature` | `StableRecoveryApplied` -> `recovered`; `StableRecoveryRejected` holding `NotStable` -> `not_stable` (refused); holding `CreatureIsDead` -> `rejected_dead` (refused) |
| `reduce_hit_point_maximum` | `HitPointMaximumReductionApplied` with `Died` -> `killed`; with `HitPointMaximumReduced` -> `reduced`; amount 0 (FR-093) -> `no_effect`; `HitPointMaximumReductionRejected` -> `rejected_dead` (refused) |
| `record_exhaustion_death` | `ExhaustionDeathApplied` -> `killed`; `ExhaustionDeathRejected` -> `rejected_dead` (refused) |

Table 4.10b, `srd5e.combat.damage_effect` per instance of damage (first matching rule wins; "Resistance took effect"
means the amount after Resistance is below the amount after adjustments, "Vulnerability took effect" means the amount
after Vulnerability is above the amount after Resistance):

| Rule | Value |
|---|---|
| Incoming amount is 0 (nothing to change, even under Immunity) | `unchanged` |
| Immunity applied (FR-023) | `immune` |
| Final amount is 0 and the incoming amount was above 0 (including a penalty that floored at 0 and Resistance rounding down) | `reduced_to_zero` |
| Resistance and Vulnerability both took effect | `resisted_and_vulnerable` |
| Resistance took effect | `resisted` |
| Vulnerability took effect | `vulnerable` |
| Otherwise (no affinity matched, or only flat adjustments or `HalfDamage` changed the amount) | `unchanged` |

### 4.11 Withdrawn ids

Ids are never reused. Tables for FR-006 to FR-013 moved with them; the Bloodied boundary table moved to
`derived-stats.md`.

| Id | Withdrawn |
|---|---|
| FR-003 | Withdrawn: moved to derived-stats.md (FR-071) |
| FR-004 | Withdrawn: moved to derived-stats.md (FR-071) |
| FR-005 | Withdrawn: moved to content-schema-and-decoder.md (FR-107) |
| FR-006 | Withdrawn: moved to attacks.md (FR-051) |
| FR-007 | Withdrawn: moved to attacks.md (FR-055) |
| FR-008 | Withdrawn: moved to attacks.md (FR-056) |
| FR-009 | Withdrawn: moved to attacks.md (FR-057) |
| FR-010 | Withdrawn: moved to attacks.md (FR-054) |
| FR-011 | Withdrawn: moved to attacks.md (FR-051) |
| FR-012 | Withdrawn: moved to attacks.md (FR-062) |
| FR-013 | Withdrawn: moved to attacks.md (FR-051) |

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

Everything in `combat` is pure, synchronous and silent. Referenced, not defined here: `DamageType`,
`Skill`, `Ability`, `DifficultyClass`, `IsBloodied`, `HitPointMaximum`, `CombatantID`, `CombatEvent`,
`CombatTelemetry`, `CombatSignals`, `dice.Roller`, the zero-value `telemetry.Noop`, `AttackRollResult.IsCriticalHit` (owners in the
header and section 3).

- **Value objects:** `HitPoints`, `Damage`, `DamageAdjustment` (closed: `FlatAdjustment(int, DamageScope)`, `HalfDamage`), `HealingCeiling` (closed: `NoHealingCeiling`, `HealingCeilingAt`),
  `DamageScope` (closed: `AllDamage`, `DamageOfType`), `DamageAffinities` (resistances as `DamageScope`s,
  vulnerabilities and immunities as `DamageType`s), `DamageResolution`, `HitKind`, `DamageDelivery` (closed:
  `MeleeAttackWithoutKnockOut`, `MeleeAttackKnockOut`, `OtherSource`), `TemporaryHitPointsChoice` (closed:
  `KeepCurrent`, `AcceptNew`), `ZeroHitPointsRule` (closed: `DiesInstantly`, `FallsUnconscious`),
  `DeathSavingThrowOutcome` (closed: `DeathSavingThrowSuccess`, `DeathSavingThrowFailure`,
  `DeathSavingThrowTwoFailures`, `DeathSavingThrowRegainedHitPoint`), `StableRecoveryDelay` (`hours`, 1 to 4),
  `StabilizingCheck`, `HitPointRejection` (closed: `CreatureIsDead`, `NotDying`, `NotStable`), `HitPointTransition`
  (FR-109), `HitPointsError` with `HitPointsProblem`, `HitPointStatusError` with `HitPointStatusProblem` (FR-105).
- **`LifeState`** (closed): `Alive`, `Dying(successes, failures)` with both in 0 to 2, `Stable`, `Dead(cause)`.
  `DeathCause` enum: `MonsterDeath`, `HitPointMaximumZero`, `MassiveDamage`, `DeathSavingThrowFailures`, `exhaustion`.
  **`HitPointStatus`** (immutable, no identity; the host holds one per creature, created through FR-087): `HitPoints`,
  `LifeState`, `ZeroHitPointsRule`, `DamageAffinities`, `IsUnconscious`, `MakesDeathSavingThrows`. Invariants: FR-002,
  FR-088 to FR-090, FR-106.
- **Events:** closed family `HitPointEvent` implementing `CombatEvent` (the classes and `kind`s of FR-103, with
  `combatants` per FR-104; `DeathSavingThrowMade` carries the outcome, `Died` the cause); each is defined only here.
- **Pure operations** (FR-109, published) and **internal services** (`DamageResolver`, `DeathSavingThrowResolver`).
- **Failures:** expected rejections are `refused` results (FR-107); untrusted values in the factories are
  `HitPointsRejected` or `HitPointStatusRejected` (FR-105); precondition violations are an invalid-input error (`srderr.ErrInvalidInput`) (FR-094, plus
  a d20 face outside 1 to 20 in the resolver); port failure is a `*srderr.PortError` for `dice.Port` (FR-078, FR-092).

## 6. Interface

Use cases are synchronous structs built by `NewX(opts ...Option)`, each with one method `Run(ctx, cmd)` (`ctx` first, because every use case emits a span). Each takes an optional `telemetry.Telemetry` (the `WithTelemetry` option; default: the zero-value `telemetry.Noop`) handed to
`CombatTelemetry` (FR-079) and a `<UseCase>Command` (carrying `subject` when it takes a status, FR-104), and returns
its own closed `<UseCase>Result` with operation-prefixed variants (`shared-types-and-ownership.md` FR-017, FR-018,
FR-021): an applied variant holding the new status and events, or a rejected variant holding a `HitPointRejection`.
Each wraps the pure operation of the same name (FR-109) and calls no other use case (FR-110). The only port is
`dice.Roller` (declared as a one-method `Roller` interface in the consuming files); there is no new port, adapter, double or contract suite. A use case returns `(Result, error)`: expected rejections are in the result, a port failure is the `*srderr.PortError` error.

| Use case | Command | Result | Port |
|---|---|---|---|
| `ApplyDamage` | status, `Damage` list and optional unpreventable list (together non-empty), adjustments (applied to each), `DamageDelivery`, `HitKind` | `ApplyDamageResult`: `DamageApplied(status, events, resolutions)` or `DamageRejected(reason)` | none |
| `RestoreHitPoints` | status, amount, optional `HealingCeiling` | `RestoreHitPointsResult`: `HealingApplied` (with amount regained) or `HealingRejected` | none |
| `GainTemporaryHitPoints` | status, amount, choice | `GainTemporaryHitPointsResult`: `TemporaryHitPointsGainApplied` or `TemporaryHitPointsGainRejected` | none |
| `EndTemporaryHitPoints` | status (host calls when a Long Rest is finished) | `EndTemporaryHitPointsResult`: `TemporaryHitPointsEndApplied` or `TemporaryHitPointsEndRejected` (FR-098) | none |
| `MakeDeathSavingThrow` | status, `RollMode` (default `normal`), `Modifier` (default 0; FR-061) | `MakeDeathSavingThrowResult`: `DeathSavingThrowApplied(outcome, status, events)` or `DeathSavingThrowRejected` | `dice.Roller` |
| `StabilizeCreature` | status | `StabilizeCreatureResult`: `StabilizationApplied` or `StabilizationRejected` | none |
| `RollStableRecoveryDelay` | empty `RollStableRecoveryDelayCommand` | `RollStableRecoveryDelayResult`: `StableRecoveryDelayRolled(delay)`, `delay` a `StableRecoveryDelay` (hours 1 to 4) | `dice.Roller` |
| `RecoverStableCreature` | status | `RecoverStableCreatureResult`: `StableRecoveryApplied` or `StableRecoveryRejected` | none |
| `ReduceHitPointMaximum` | status, amount | `ReduceHitPointMaximumResult`: `HitPointMaximumReductionApplied` or `HitPointMaximumReductionRejected` | none |
| `RecordExhaustionDeath` | status | `RecordExhaustionDeathResult`: `ExhaustionDeathApplied` or `ExhaustionDeathRejected` (FR-095) | none |

**Exports of package `combat`** (FR-083 derives from this): the ten use cases
variants; every value object, closed family, enum and event of section 5 (including `StabilizingCheck`,
`HitPointTransition` with its two variants, the parser errors and problem enums); and the `HitPointStatus` operations.
`DamageResolver` and `DeathSavingThrowResolver` stay internal. `DamageType` is consumed from
`content-schema-and-decoder.md`, not re-exported.

## 7. Observability

The seven questions (`doc/rules/observability.md` §2). **Counted or timed:** the shared operations, events and duration come from `combat-foundations.md`; this spec adds damage instances by effect, Hit Points lost and restored, refusals by reason, death saves by result and deaths by cause. **Existing instrumentation:** no earlier combat signals exist; the earlier own operations, duration and error rows and the refusal-as-error-status rule were superseded by `combat-foundations.md` FR-029, FR-031 and FR-032. **What can go wrong:** the `dice.Roller` fails (`unavailable`)
or returns the wrong expression (`InvalidResult`); a caller operates on a creature in the wrong state (a refusal); a
programmer passes invalid arguments (an invalid-input error (`srderr.ErrInvalidInput`), on the span only); a wrong rule result such as a missed
Resistance. **How an operator knows:** refusals are `result_class` = `refused`, `srd5e.outcome` = `failure`, no span
error status (`combat-foundations.md` FR-031), reason in `srd5e.combat.hit_point_refusals`; a port failure is in
`srd5e.combat.errors` with `error.type`. **To diagnose:** the attributes below; amounts and the `CombatantID` live in
results and events, never in attributes. **Monitor:** any non-zero `srd5e.combat.errors` over 5 minutes; a `refused`
share above 5% of an operation's calls; the death-save mix; p99 duration above its 7-day baseline. **Noisy:** no; no
log in the normal path, failures are recorded once and not logged (`doc/rules/observability.md` §7). `srd5e.outcome`
keeps only `success` and `failure` (`shared-types-and-ownership.md` FR-041).

The shared signals (span `srd5e.combat.<operation>`, `operations`, `operation_duration`, `errors`, `events`) and
attributes (`operation`, `outcome`, `result`, `result_class`, `event`) are owned by `combat-foundations.md` (FR-022 to
FR-030); `result` values are table 4.10a (FR-107), `event` values the `kind`s of FR-103. Emission rules and value
mappings below are FR-108. Names are constants in `CombatSignals`; entries feed `doc/observability/signals.md`; where
foundations differs, it wins. `srd5e.combat.deaths` is the one package death counter; the encounter
(`encounter-aggregate.md` FR-081) and `combat-action-resolution.md` (FR-102) increment it once per wrapped `Died`, and
no death is counted twice because no use case calls another (FR-110).

| Signal | Type | Attributes | Emitted by | Purpose | Suggested monitoring |
|---|---|---|---|---|---|
| `srd5e.combat.damage_resolved` | counter `{instance}` | `srd5e.combat.damage_type`; `srd5e.combat.damage_effect` (`unchanged`, `resisted`, `vulnerable`, `resisted_and_vulnerable`, `immune`, `reduced_to_zero`) | `ApplyDamage` | Pipeline effect per instance of damage | Dashboard by effect; a sudden `immune` share means an affinity bug |
| `srd5e.combat.hit_points_lost` | histogram `{hit_point}` | none | `ApplyDamage` | Size of Hit Point loss per command after the pipeline | p99 above 2x its 7-day baseline; a collapse to 0 means an Immunity or Resistance bug |
| `srd5e.combat.hit_points_restored` | histogram `{hit_point}` | none | `RestoreHitPoints` | Size of Hit Points actually regained | Compare with the Healing offered |
| `srd5e.combat.death_saves` | counter `{roll}` | `srd5e.combat.death_save_result` (`success`, `failure`, `critical_failure`, `critical_success`) | `MakeDeathSavingThrow` | Death Saving Throw results | Expected mix: `success` about 50%, `failure` about 40%, `critical_failure` about 5%, `critical_success` about 5%; investigate a 10-point drift over 1,000 rolls |
| `srd5e.combat.hit_point_refusals` | counter `{refusal}` | `srd5e.combat.operation`; `srd5e.combat.refusal_reason` (`creature_is_dead`, `not_dying`, `not_stable`) | every use case that can return a `...Rejected` | Why an operation was refused | A `refused` share above 5% of an operation's calls means the host calls in the wrong state |
| `srd5e.combat.deaths` | counter `{creature}` | `srd5e.combat.death_cause` (`monster_death`, `hit_point_maximum_zero`, `massive_damage`, `death_saving_throw_failures`, `exhaustion`) | `ApplyDamage`, `MakeDeathSavingThrow`, `ReduceHitPointMaximum`, `RecordExhaustionDeath`; and the use cases of `encounter-aggregate.md` (FR-081) and `combat-action-resolution.md` (FR-102) whose result holds a wrapped `Died` | Why creatures die | A non-zero `hit_point_maximum_zero` rate when the consumer applies no maximum reduction |

## 8. Test plan

Levels: unit (`combat/*_test.go`), application (`combat/*_test.go`, real domain with `dicetest.FixedRoller`,
`dicetest.CallbackRoller` and a recording `telemetry.Telemetry` from the `<pkg>test` packages, `shared-types-and-ownership.md` FR-038),
scenario (`scenarios/`), architecture. Table-driven tests cite per row with `Requirement: per case`.

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002 | unit | `NewHitPoints` and `MustHitPoints`; one row per violated field |
| FR-014, FR-015, FR-086 | unit | Table 4.2b; `Damage` constructor: type carried, negative amount returns an error, 0 valid |
| FR-100 | application | One `Damage` applied to two creatures with different affinities; the `Damage` is unchanged |
| FR-016 to FR-025, FR-099 | unit | Table 4.3a; FR-025 asserts the breakdown fields; FR-099 asserts the flat-then-half order |
| FR-026 to FR-034, FR-084, FR-098 | unit | Tables 4.6a and 4.7b temporary-absorption rows; the choice (10 held and 12 offered, none held, equal); Healing leaves temporary unchanged; state unchanged at 0; end operation incl. 0 held and Dead |
| FR-035 to FR-039 | unit | Table 4.5a plus state rows |
| FR-040 to FR-052, FR-085 | unit | Table 4.6a, exact event lists; `BecameBloodied` ignores Temporary Hit Points |
| FR-087 to FR-091, FR-106 | unit | Creation and restore (reject Dying with tally 3, Dead with current above 0, Alive at 0, maximum 0 with `Dying`, `Stable` or `Alive`; accept maximum 0 only with `Dead(HitPointMaximumZero)`); `DamageAffinities` empty, duplicates, defensively copied views, equality |
| FR-105 | unit | `NewHitPoints`, `NewHitPointStatus` and `RestoreHitPointStatus`: each problem alone and combined (fixed order), no error on valid input, extreme values never panic, `MustX` panics iff the parser errors (one input table through both); problems slice defensively copied; never an empty problem list (`errors-and-results.md` FR-024); fuzz test over arbitrary ints |
| FR-103, FR-104 | unit | Per event class: `Kind()` equals the FR-103 constant, unique, matches `[a-z][a-z0-9_]*`; `Combatants()` is `[subject]`, defensively copied, no duplicate; value equality; table 4.9a events carry the `subject`. Architecture: no `HitPointStatus` field of type `CombatantID` |
| FR-107 | application | Table 4.10a, one row per variant and rule: class and `result` value; never `invalid`; each value maps to one class |
| FR-093, FR-095, FR-096 | unit | Reduction of 0: state equal, no events; Alive, Dying, Stable to Dead(exhaustion), Dead rejected; `MakesDeathSavingThrows` per `LifeState`, a `FallsUnconscious` monster is Dying and due one |
| FR-097, FR-101 | unit | Table 4.9a, exact ordered lists; Bloodied rows (already Bloodied, dies, Healing, Temporary Hit Points present) |
| FR-053 to FR-066 | unit | Tables 4.7a (FR-053 to FR-061) and 4.7b (FR-062 to FR-066) |
| FR-067 to FR-071, FR-073 to FR-075 | unit | State-transition tests; FR-069 pins `StabilizingCheck` (DC 10, Wisdom, Medicine) |
| FR-072 | application | Table 4.8a with `dicetest.FixedRoller`; `Requests()` is exactly `[1d4]` |
| FR-076, FR-077 | unit, application | Inputs compared before and after; equal results for equal inputs, also through each use case |
| FR-078, FR-092 | application | `dicetest.CallbackRoller` returning an error: `errors.Is` reaches the cause through the `*srderr.PortError`, no new state. Wrong expression (count, die, modifier): `srderr.ErrContractViolation` (a `*srderr.PortError`), no new state. `Requests()` is exactly `[1d20]` (`[2d20]` under Advantage or Disadvantage) for `MakeDeathSavingThrow`, `[1d4]` for `RollStableRecoveryDelay` |
| FR-094 | unit | Preconditions return an error matching `srderr.ErrInvalidInput`; 0 amounts are valid no-ops |
| FR-079 to FR-082 | application | Recording `telemetry.Telemetry`: shared signals once per call as in FR-107; a refusal is `failure`, `refused`, no span error status, no `errors` increment, no `error.type`; a `*srderr.PortError` for `dice.Port` is recorded once on the span and once in `errors` with `error.type` = `srderr.ErrorType(err)`, no log; no attribute equals the `CombatantID`; same result with no-op; signal names pinned |
| FR-108 | application | One row per signal: `damage_resolved` once per instance with the table 4.10b value (incoming 0 under Immunity `unchanged`, Immunity `immune`, 1 Fire under Resistance `reduced_to_zero`, Resistance only, Vulnerability only, both, flat adjustment only `unchanged`); `hit_points_lost` one sample per applied command (absorbed and zero damage give 0, a rejection none); `hit_points_restored` (amount, 0 when full, none on rejection); `death_saves` with four results; `hit_point_refusals` with each reason; `deaths` once per `Died` with each cause, never on a rejection or returned error |
| FR-109, FR-110, FR-111 | unit, architecture | One row per operation: `HitPointTransitionApplied` with status and exact table 4.9a events, or `HitPointTransitionRefused` (`CreatureIsDead`, `NotDying`, `NotStable`); inputs not mutated; each use case returns the operation's status and events unchanged; no use case holds or calls another; `HitKind` has exactly two members and `CriticalHit` at 0 Hit Points adds two failures (FR-063); no other `combat` type restates `IsCriticalHit` for damage; operations exported |
| FR-118 | unit | Host mappings: two sources of the same Resistance collapse; grant 5 against 3 held with each choice; `Died` of each cause, a drop without `Died` and a knock-out map as FR-118 states |
| FR-112 to FR-117 | unit | Affinity constructors (including zero value equal to `NoAffinities()`) and the resilience mapping; scoped `FlatAdjustment` changes only its type, unscoped unchanged; unpreventable skips flat, half, Resistance, Immunity, takes Vulnerability and Temporary Hit Points; ceiling cases (none, below, at, above maximum, cap 0 at 0 Hit Points, negative returns an error); `RestoreMaximum` (Dead, at or below, above, below 1 returns an error; current kept); public event constructors and `DamageResolution.Type()` and `Dealt()` |
| FR-102 | unit | `DamageTaken`: `total == HitPointsLost + AbsorbedByTemporary` for rows of tables 4.6a and 4.7b; plain `int`s |
| FR-083 | architecture | Export check derived from section 6, plus the signal catalogue consistency check |

- **Scenario:** one test through the public packages: a Critical Hit drops a character to 0, Death Saving
  Throws fail and succeed, the character is stabilized and healed; and a second with Temporary Hit Points, Resistance
  and Vulnerability. **Doubles:** `dicetest.FixedRoller` (with `requests`), `dicetest.CallbackRoller` and a recording
  `telemetry.Telemetry` from the `<pkg>test` packages; no new port or contract suite.
- **Mutation testing:** all of `combat`, score at least 80 (`gomu`, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors; `doc/rules/testing.md` §8); boundary
  mutations (the 10 threshold, the 20 and 1 exceptions, remainder against maximum) first. **Not covered:** none
  deliberately; FR-015 is partly a compile-time property.

## 9. Open questions and decisions

| Question | Recommendation | Status |
|---|---|---|
No open or escalated questions.

Settled: Q-3 (a Stable creature that takes damage becomes Dying and also suffers the failure, two on a Critical Hit,
and the Massive Damage test: FR-071; interpretation, it has 0 Hit Points and takes damage, so "Stabilizing a
Character" and "Damage at 0 Hit Points", p. 18, both apply); Q-4 (a Death Saving Throw is a D20 Test, so Advantage,
Disadvantage and the Exhaustion reduction apply; natural decides 1 and 20, total decides 10: FR-054 to FR-056, FR-061;
pp. 17, 18, 180, 181; matches `combat-action-resolution.md` FR-069); Q-11 (Damage Threshold and objects belong to
`environment-hazards-and-objects.md`, evaluated before `ApplyDamage`); Q-1 (remainder after Temporary Hit Points:
FR-043 to FR-045, FR-064); Q-2 (absorbed damage at 0 Hit Points is damage taken, no failure, `total` feeds
Concentration: FR-084, FR-062, FR-102); Q-5 (flat, then half, then Resistance: FR-099); Q-6 (one command, summed:
FR-085); Q-7 (Dead is rejected: FR-075; revival is a separate spec); Q-8 (the rest of the knock-out sidebar belongs to
`conditions.md` and the rest specs); Q-9 (per-creature `ZeroHitPointsRule`, no default: FR-087, FR-096); Q-10, Q-14,
Q-15 (`shared-types-and-ownership.md` FR-005, FR-006, FR-021, FR-046, FR-049); Q-12 (the host owns time); Q-13
(preconditions, the host clamps: FR-094); Q-16 (refusals are `refused`: FR-080, FR-107); Q-17 (`subject` argument:
FR-104); Q-18 (`HitPointMaximum` built only where the creature is not Dead: FR-101, FR-106); Q-19 (interpretation:
Wish stress damage skips reductions and Immunity, takes Vulnerability and Temporary Hit Points: FR-114; the host maps
`UnpreventableDamage` to `Damage` so this spec does not depend on the later-ranked `wish-spell.md`); Q-20
(interpretation: a generic healing ceiling, the host computes it, `toolbox-poison.md` Q-11 stays valid: FR-115); Q-21
(interpretation: `RestoreMaximum` never lowers a maximum: FR-116); Q-22 (magic item resilience is three `DamageAffinities`
constructors, switched by the host, no `combat` import of the content enum: FR-112). Naming: the pure operation is the method `HitPointStatus.Restore` and the parser is the function `RestoreHitPointStatus`, so the two never clash (FR-109).

## 10. Changelog impact

Added: `combat` hit points and damage (Hit Points, applying damage, Resistance, Vulnerability and Immunity, Temporary
Hit Points, Healing, dropping to 0 Hit Points, Death Saving Throws, stabilizing), with telemetry under
`srd5e.combat.*` through `CombatTelemetry` (FR-107, FR-080, FR-108), with scoped adjustments, unpreventable damage, a healing ceiling and `RestoreMaximum`. Every `HitPointEvent` carries `Kind()` and`combatants`. The rules are published as pure `HitPointStatus` operations returning a `HitPointTransition`, which the
use cases wrap; `NewHitPoints`, `NewHitPointStatus` and `RestoreHitPointStatus` return typed errors. Not **BREAKING** (new API, no earlier combat
release).
