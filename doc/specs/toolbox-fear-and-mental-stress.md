# Fear and Mental Stress

| | |
|---|---|
| Status | draft |
| Context | `toolbox` |
| Roadmap phase | 5 |
| Depends on | `toolbox-hazard-effects.md`, `d20-tests.md`, `hit-points-and-damage.md`, `dice-and-rolls.md`, `combat-foundations.md`, `modifiers-and-advantage.md`, `ability-scores.md`, `content-schema-and-decoder.md`, `errors-and-results.md`, `observability-ports.md` |
| Referenced, not imported | `resolve-d20-test-use-case.md` (the host resolves the save; `shared-types-and-ownership.md` FR-044), `conditions.md` (the host applies and removes handed-back `ConditionKind`s), `spellcasting.md` (host maps a cast spell to `ProlongedCure`, Q-8), `rule-options.md` (option pattern only), `initiative-and-actions.md` (owns the Dash action; `DashTaken` here is a local report value that only reads it), `content-licensing-and-notice.md`, `toolbox-curses-and-contagions.md` (separate cure vocabulary, Q-17) |
| In-context order | Inside `toolbox`: hazard's types, including `ToolboxRuleOptions`, `FearRider`, `FearOptions` and `MentalStressOnSuccess` (hazard FR-099, FR-102 to FR-104); then this spec's value types and use cases, which import them upward. No import cycle (Q-19, settled by hazard Q-20). |

## 1. Purpose and scope

"Fear and Mental Stress" (SRD 5.2.1, p. 196) is an optional Game Master toolkit: "If you plan to use any of these rules, discuss them with your players at the start of the campaign." This spec makes its three parts available as pure rules and small use cases, with nothing switched on globally: (1) **Fear Effects**, the Frightened condition resisted by a Wisdom save against a DC from how terrifying the situation is (Sample Fear DCs 10, 15, 20) with a repeat save at the end of each of the creature's turns, plus three **optional riders** as command options; (2) **Mental Stress Effects**, Psychic damage per the Sample Mental Stress table (DC 10 / 1d6, 15 / 3d6, 20 / 9d6), Wisdom save by default and the GM's choice of half damage on a success; (3) **Prolonged Effects**, short-term (Frightened, Incapacitated or Stunned for 1d10 minutes), long-term (Disadvantage on some or all ability checks for 1d10 x 10 hours) and indefinite effects, suppressed by Calm Emotions and removed by Lesser or Greater Restoration.

**How it fits** (state in, new state and events out, `doc/rules/architecture.md` §4). The host rolls the save with `d20.Resolver` and hands it over as a `SaveGatedSave` (rolled `CheckResult`, `ForcedFailureResult`, or not yet rolled). A use case never calls another use case (`shared-types-and-ownership.md` FR-044). **One effect model, owned by hazard:** fear and mental stress are `SaveGatedEffect`s (hazard FR-008); this spec defines no effect, state, duration, repeat-save, consequence, event or signal type hazard already has. It adds builders producing a `SaveGatedEffect`, the riders and turn check, the long-term Disadvantage scope, the prolonged-effect lifecycle with its cure matrix, and the sample tables. The Resolve use cases call hazard's **domain** planner and evaluator, never its use cases.

**The engine hands conditions back; it never applies them** (hazard FR-046, Q-5): results carry `ConditionKind` values, the host builds the `Condition` (with its source of fear, `conditions.md` FR-003) and calls `ConditionSet.apply` or `.remove` (section 4.9).

**Optional rules are command inputs, never global flags** (`doc/rules/architecture.md` §3, §10). `FearOptions` (with `FearRider`) and the closed interface `MentalStressOnSuccess` are immutable values owned by `toolbox-hazard-effects.md` (FR-102 to FR-104), carried as the fields `fear` and `mentalStressOnSuccess` of its `ToolboxRuleOptions` (FR-099; `shared-types-and-ownership.md` FR-045). Their behaviour is owned here. `ToolboxRuleOptions.standard` gives baseline SRD behaviour.

**Out of scope:**

- The Frightened, Incapacitated and Stunned conditions and applying or removing them: `conditions.md`. This spec decides only **when** the host should.
- Applying Psychic damage (Resistance, Temporary Hit Points, Half Damage arithmetic): `hit-points-and-damage.md` (FR-014, FR-099).
- The spells Calm Emotions, Lesser Restoration and Greater Restoration (casting, slots, targeting): `content-spells.md`, `spellcasting.md`. Only the three statements on p. 196 of what a cure does are modelled (Q-8).
- Turn timing, elapsing minutes and hours (host or `encounter-aggregate.md`); GM advice and flavour; distance and line of sight (the host reports `FearMovement`).
- The Far Realm example's duration ("until the end of that character's next turn"): the row keeps DC and damage only (FR-003).
- `ToolboxRuleOptions` itself and the other Gameplay Toolbox sections.
- Anything not in SRD 5.2.1. Nothing core-looking was found missing from p. 196.

**Supersedes** (`shared-types-and-ownership.md` FR-048): `toolbox-hazard-effects.md` Q-5 (the remark that this spec "adds a typed source to its own command", met by `fear.source`, FR-010) and its section 5.2 row "Fear and Mental Stress save types"; rider and half-damage behaviour lives here (FR-027 to FR-044; the option values moved to hazard, so FR-024 to FR-026 and FR-051 are withdrawn).

## 2. Source

- SRD 5.2.1, "Fear and Mental Stress", "Fear Effects", "Sample Fear DCs", "Mental Stress Effects", "Sample Mental Stress Effects" and "Prolonged Effects" (Short-Term, Long-Term, Indefinite), p. 196; "Saving Throws", p. 7 ("you can choose to fail the save without rolling"), through hazard's `SaveGatedSave`.
- Cross-references only: "Half Damage", p. 16, "Round Down", p. 5, and the Frightened, Incapacitated and Stunned entries, Rules Glossary pp. 182, 184, 189.
- Package design: `doc/rules/architecture.md` §2 to §6, `doc/rules/observability.md` §2 to §7, `doc/rules/testing.md` §4, `shared-types-and-ownership.md`, `toolbox-hazard-effects.md`. Printed page numbers equal PDF page indexes.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Fear Effects | `FearEffect` (source, DC, save `Ability`, default Wisdom) | p. 196. Expressed as a `SaveGatedEffect` (FR-008) |
| Frightened condition; "the source of fear" | `ConditionKind.frightened` handed back with `FearEffect.source` (`CombatantID`) | Owned by `conditions.md`; Q-6 |
| one fear on one creature | `ActiveFear` (`fear`, `active`: `ActiveSaveGatedEffect`) | Package name |
| saving throw | `SaveGatedSave` (`SaveGatedSaveRolled`, `SaveGatedSaveForcedFailure`, `SaveGatedSaveNotRolled`) | Hazard section 5; `d20-tests.md` |
| Sample Fear DCs | `SampleFearDc` (read-only, three `SampleFearDcRow`, returned by a function as a fresh copy), `SampleFearExample` | p. 196: 10, 15, 20. No SRD prose stored (FR-001) |
| optional Frightened examples | `FearRider` (`dashEachTurn`, `attackersHaveAdvantage`, `oneActionKindPerTurn`), `FearOptions` | Three bullets on p. 196; defined by hazard (FR-102, FR-103), carried as `ToolboxRuleOptions.fear`; behaviour here |
| Dash action; move, action, Bonus Action | `DashTaken` (`notTaken`, `asAction`, `asBonusAction`), `TurnUse`, `FearMovement` | The Dash action is owned by `initiative-and-actions.md` (referenced, not imported); `DashTaken` is a local report value, only read here |
| Mental Stress Effects | `MentalStressEffect` (DC, damage `dice.Expression`, save `Ability`) | p. 196. Psychic fixed (FR-006, Q-15). Expressed as a `SaveGatedEffect` (FR-040) |
| Sample Mental Stress Effects | `SampleMentalStress` (read-only, three `SampleMentalStressRow`, returned by a function as a fresh copy), `SampleMentalStressExample` | p. 196: 10 / 1d6, 15 / 3d6, 20 / 9d6 |
| half damage "at your discretion" | `MentalStressOnSuccess` (closed interface: `MentalStressNoDamageOnSuccess`, `MentalStressHalfDamageOnSuccess`); `HalfDamage` | defined by hazard (FR-104), carried as `ToolboxRuleOptions.mentalStressOnSuccess`; mapped here to hazard `SaveGatedSuccess`; `HalfDamage` is owned by `hit-points-and-damage.md` |
| Prolonged Effects | `ProlongedEffect`, `ProlongedTerm` (`shortTerm`, `longTerm`, `indefinite`), `ProlongedStatus` (`active`, `suppressed`) | p. 196; section 5.1 |
| Short-Term Effects | `ShortTermCondition` (`ShortTermFrightened(source)`, `ShortTermIncapacitated`, `ShortTermStunned`) | 1d10 minutes |
| Long-Term and Indefinite Effects | `AbilityCheckScope` (`ProlongedAllAbilityChecks`, `ProlongedSomeAbilityChecks(abilities)`) | Disadvantage on some or all ability checks, 1d10 x 10 hours; indefinite lasts until Greater Restoration |
| Calm Emotions, Lesser Restoration, Greater Restoration | `ProlongedCure` (`calmEmotions`, `lesserRestoration`, `greaterRestoration`) | Spell names as printed; Q-8, Q-17 |

Duration reuses hazard's `SaveGatedDuration` and `SaveGatedFixedTime` (this spec FR-055, FR-093; hazard FR-008, FR-091). Other package names: `ProlongedChange` (`ProlongedResume`, `ProlongedCureWith`), `ProlongedEffectSpec` (`ProlongedShortTermSpec`, `ProlongedLongTermSpec`, `ProlongedIndefiniteSpec`), `ProlongedConditionHandBack`, `ProlongedRejection`, `ProlongedEffectEvent`, `FearRiderEffect`, `FrightenedTurnCheck`. Psychic damage is `DamageType.psychic` (`content`, `shared-types-and-ownership.md` FR-005).

**Names deliberately avoided** (`doc/rules/architecture.md` §9 duplicate-name scan): `ForMinutes`, `ForHours`, `Indefinite`, `AllAbilityChecks`, `SomeAbilityChecks`, `ResumeChange`, `CureChange`, `Frightened` (a `conditions.md` `Condition` variant), hazard's own types (FR-092 here) and `MentalStressAbility` (FR-045); also the `ContagionCuringMagic` family of `toolbox-curses-and-contagions.md`, neither imported nor reused (Q-17). Every type here carries a `Fear`, `MentalStress`, `Prolonged`, `Sample` or `ShortTerm` prefix or a use-case prefix (FR-092).

## 4. Requirements

Stable ids. Never renumber. Unless a row says otherwise, "the save" is a `SaveGatedSave` whose rolled value is a `CheckResult` of kind `D20TestKind.savingThrow` or a `ForcedFailureResult` (`d20-tests.md` FR-035, FR-036; SRD 5.2.1, "Saving Throws", p. 7), validated by the hazard evaluator (hazard FR-039). "Hazard FR-nnn" is a requirement of `toolbox-hazard-effects.md`. Result variants are written with their full use-case prefix; a table caption says when it abbreviates.

### 4.1 Sample tables and custom values

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `SampleFearDc` MUST hold exactly three `SampleFearDcRow`s in printed order: `apparitionFromSarcophagus` (DC 10), `illusoryWorstFears` (DC 15), `abyssPortal` (DC 20). A row MUST hold only the example id and the `DifficultyClass`; no SRD example sentence or display string MUST be stored in code (enum ids are identifiers, not prose). | SRD 5.2.1, "Sample Fear DCs", p. 196; `doc/rules/architecture.md` §3; `AGENTS.md` (bundled game content is kept apart from code); Q-4 | happy |
| FR-002 | Each `SampleFearDc` DC MUST equal the matching `DifficultyClass` constant of `d20-tests.md` FR-021 (`easy` 10, `medium` 15, `hard` 20). | SRD 5.2.1, "Sample Fear DCs", p. 196; "Difficulty Class", p. 6 | happy |
| FR-003 | `SampleMentalStress` MUST hold exactly three `SampleMentalStressRow`s in printed order: `hallucinogenicSubstance` (DC 10, 1d6), `fiendishIdol` (DC 15, 3d6), `farRealmTrap` (DC 20, 9d6). A row MUST hold only the example id, `DifficultyClass` and `dice.Expression`: no prose, no duration (the Far Realm duration is out of scope) and no damage type (every sample is Psychic, FR-006). | SRD 5.2.1, "Sample Mental Stress Effects", p. 196; Q-4 | happy |
| FR-004 | Both sample tables MUST be immutable: each is returned by a function as a fresh copy of its row slice (there is no exported mutable package variable), so appending to or modifying a returned slice MUST NOT change the next call's result. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-005 | A `FearEffect` MUST accept any `DifficultyClass`, not only 10, 15 or 20; DC 5 and DC 30 MUST be accepted, with no minimum or maximum. | SRD 5.2.1, "Fear Effects", p. 196; `d20-tests.md` D-02 | unhappy (boundary) |
| FR-006 | A `MentalStressEffect` MUST accept any `DifficultyClass` and any `dice.Expression` (validity is `dice-and-rolls.md` FR-007, FR-008), and the damage type of the effect built from it MUST be Psychic with no input that sets another. This is a **package design decision**, not an SRD prohibition (p. 196 calls Psychic damage "the best way to emulate" such an effect, Q-15). | package design (Q-15); SRD 5.2.1, "Mental Stress Effects", p. 196 | unhappy (boundary) |
| FR-007 | `FearEffect` MUST default its save ability to Wisdom and MUST accept any other `Ability`. | SRD 5.2.1, "Fear Effects", p. 196 ("typically require a Wisdom saving throw") | happy |

### 4.2 Fear Effects: the initial save (use case `ResolveFearEffect`)

Command: `target` (`CombatantID`), `fear` (`FearEffect`), `save` (`SaveGatedSave`), `options` (`ToolboxRuleOptions`, default `ToolboxRuleOptions.standard`; only `options.fear` is read). There is no `SaveGatedSubject` (FR-104). The use case builds `fear.saveGatedEffect` (FR-008), evaluates it with the hazard planner and evaluator at `SaveGatedInitialExposure(effect, target)` and maps the outcome. Result `ResolveFearEffectResult`: `ResolveFearEffectSaveRequired`, `ResolveFearEffectResisted`, `ResolveFearEffectFrightened`, `ResolveFearEffectRejected(reason)` (payloads in section 6).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-008 | `FearEffect.saveGatedEffect` MUST be a valid `SaveGatedEffect` with: the requirement `SaveGatedSaveRequired(fear.ability, fear.dc)` (hazard FR-008, FR-105); no Disadvantage circumstances; an empty automatic-success set; `SaveGatedOnsetImmediate`; a failure outcome of exactly one `SaveGatedCondition(frightened, SaveGatedAnyFailure)`, no damage, no Exhaustion, `healable`; `SaveGatedNoEffectOnSuccess`; `SaveGatedRepeatSaves(SaveGatedRepeatAtTurnEnd, endsAfterSuccesses: 1, SaveGatedRepeatFailureNoFurtherEffect)`; duration `SaveGatedDurationUntilEnded`. It MUST satisfy hazard FR-009 to FR-023. A failed save MUST return `ResolveFearEffectFrightened` with the consequences of hazard FR-036. | SRD 5.2.1, "Fear Effects", p. 196 ("baseline effect"; "repeats the saving throw at the end of each of its turns, ending the effect on itself on a success"); hazard FR-008, FR-027, FR-036 | happy |
| FR-009 | A successful save MUST return `ResolveFearEffectResisted` with empty consequences (hazard FR-033), no dice requested and no events. | SRD 5.2.1, "Fear Effects", p. 196 | happy |
| FR-010 | `ResolveFearEffectFrightened` MUST carry `fear.source` next to the handed-back `ConditionKind.frightened`, so the host builds `Frightened(source)` itself. The engine MUST NOT construct a combat `Condition`. | SRD 5.2.1, "Fear Effects", p. 196; hazard FR-046; `conditions.md` FR-003 | happy |
| FR-011 | The engine MUST NOT read the target's `ConditionSet`. If the creature already is Frightened the host's `ConditionSet.apply` returns `ConditionAlreadyHeld` (`conditions.md` FR-018, FR-019); the host keeps the original source and discards the returned `ActiveFear` (table 4.9b). `ResolveFearEffectFrightened` never claims the condition was applied. | hazard FR-046; `conditions.md` FR-018, FR-019; SRD 5.2.1, "Condition", p. 179 | unhappy (rule interaction) |
| FR-012 | A creature immune to Frightened is the host's `ConditionImmune` outcome (`conditions.md` FR-021, FR-125); the host discards the returned `ActiveFear` (table 4.9b). | hazard FR-046; `conditions.md` FR-021; SRD 5.2.1, "Immunity", p. 183 | unhappy (rule interaction) |
| FR-013 | A failed save MUST give consequences whose conditions are exactly `[ConditionKind.frightened]`, no damage, no Exhaustion, healing `healable`, whatever else is true of the target. | hazard FR-036, FR-044, FR-045 | unhappy (empty and minimal) |
| FR-014 | A rolled save MUST decide only from `save.outcome`, never from a total, DC or natural value. A `CheckResult` with total equal to the DC and outcome `success` MUST give `ResolveFearEffectResisted`; total one below the DC and outcome `failure` MUST give `ResolveFearEffectFrightened`. | SRD 5.2.1, "Fear Effects", p. 196; `d20-tests.md`; hazard FR-037 | unhappy (boundary) |
| FR-015 | A rolled save whose target DC differs from `fear.dc` MUST return `ResolveFearEffectRejected(SaveGatedRejection.saveDoesNotMatchEffect)` with nothing rolled, no consequences and no events (hazard FR-039, run by the evaluator; untrusted input is a result). | hazard FR-039; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-016 | A rolled save whose kind is not `D20TestKind.savingThrow` MUST return `ResolveFearEffectRejected(saveDoesNotMatchEffect)`, as FR-015. | hazard FR-039; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-017 | The use case MUST NOT alter the `FearEffect`, `ToolboxRuleOptions` or save passed in. | `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-088 | A `SaveGatedSaveForcedFailure` of kind `savingThrow` MUST give `ResolveFearEffectFrightened` exactly as a rolled failure does; the DC check (FR-015) is skipped because a forced failure has no total or target (a forced failure of another kind: FR-109). | SRD 5.2.1, "Saving Throws", p. 7; hazard FR-036, FR-038, FR-039; `d20-tests.md` FR-035, FR-036 | unhappy (rule interaction) |
| FR-098 | A `SaveGatedSaveNotRolled` MUST return `ResolveFearEffectSaveRequired` carrying the hazard `SaveGatedRollPlan` (the fear's ability, DC and empty roll circumstances) and request no dice. | SRD 5.2.1, "Saving Throws", p. 7; hazard FR-030, FR-031 | happy |
| FR-104 | `ResolveFearEffect`, `ResolveFearRepeatSave` and `ResolveMentalStress` MUST take no `SaveGatedSubject`. Each hands the hazard planner and evaluator one fixed subject: `DamageAffinities` with empty resistance, vulnerability and immunity sets, no `SaveGatedCircumstance`, and `SaveGatedNoExemption`. Hazard's `ResolveSaveGatedEffectAutoSucceeded` (hazard FR-024, FR-032, FR-076) and a repeat save counted as `autoSucceeded` (hazard FR-060) are therefore unreachable, so the result types have no variant for them and the mappings of FR-014, FR-018, FR-019 and FR-089 are total over the variants that can occur. A creature's own Resistance, Immunity or exemption is not an input of these rules. | package design; hazard FR-024, FR-032, FR-060, FR-076; `doc/rules/architecture.md` §3 (illegal states unrepresentable) | unhappy (rule interaction) |

### 4.3 Fear Effects: the repeat save (use case `ResolveFearRepeatSave`)

Command: `state` (the `ActiveFear` of FR-008), `save`; no subject (FR-104). Result `ResolveFearRepeatSaveResult`: `ResolveFearRepeatSaveSaveRequired`, `ResolveFearRepeatSaveContinues`, `ResolveFearRepeatSaveEnded`, `ResolveFearRepeatSaveRejected(reason)`. It calls the hazard domain repeat-save evaluator (hazard FR-055 to FR-063).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | A successful repeat save on an active fear MUST return `ResolveFearRepeatSaveEnded` with the state in `PhaseEnded(successesReached)`, `fear.source`, the handed-back `ConditionKind.frightened` to remove, and the hazard events `SaveGatedEffectRepeatSaveMade(result: succeeded)` then `SaveGatedEffectEnded` (granted conditions `[frightened]`, hazard FR-067), in that order (hazard FR-084). The engine does not remove the condition and emits no `ConditionRemoved` (the host's, `conditions.md` FR-129). | SRD 5.2.1, "Fear Effects", p. 196; hazard FR-056, FR-067, FR-084 | happy |
| FR-019 | A failed repeat save MUST return `ResolveFearRepeatSaveContinues` with empty consequences (`SaveGatedRepeatFailureNoFurtherEffect`, hazard FR-059), the state unchanged and one `SaveGatedEffectRepeatSaveMade(result: failed)`. | SRD 5.2.1, "Fear Effects", p. 196; hazard FR-059 | happy |
| FR-020 | A repeat save on an `ActiveFear` whose phase is `PhaseEnded` (or `PhaseAwaitingOnset`, which a fear never has) MUST return `ResolveFearRepeatSaveRejected(effectNotActive)` with nothing rolled and nothing changed. | hazard FR-061; `doc/rules/testing.md` §4 | unhappy (state error) |
| FR-021 | `ResolveFearRepeatSaveEnded` MUST name `fear.source`; the host MUST remove a Frightened only if the one the target holds has that source (pre-check after table 4.9a). A Frightened from a different source is left. Two fears are two values and one never ends the other; `ConditionSet.remove(kind)` takes no source (`conditions.md` FR-031), so the check is the host's. | SRD 5.2.1, "Fear Effects", p. 196; `conditions.md` FR-019; hazard FR-046 | unhappy (rule interaction) |
| FR-022 | An `ActiveFear` MUST be rejected with `ErrInvalidInput` error at construction when `active.effect` differs from `fear.saveGatedEffect`. A save of the wrong DC or kind returns `ResolveFearRepeatSaveRejected(saveDoesNotMatchEffect)` (hazard FR-039, FR-062) as FR-015, FR-016, and a rejected state wins over a mismatched save (hazard FR-085). | `doc/rules/architecture.md` §6; hazard FR-039, FR-062, FR-085 | unhappy (invalid input) |
| FR-023 | The use case MUST decide only from `save.outcome` (as FR-014) and MUST NOT alter its inputs (as FR-017). | `d20-tests.md`; `doc/rules/testing.md` §4 | unhappy (boundary) |
| FR-089 | A `SaveGatedSaveForcedFailure` of kind `savingThrow` on a repeat save MUST give `ResolveFearRepeatSaveContinues` (the effect never ends on it). **Interpretation:** hazard FR-062 covers only no-save and mismatch; this follows hazard FR-059 and FR-038, and Q-16 (f) asks hazard to add the row. A `SaveGatedSaveNotRolled` MUST give `ResolveFearRepeatSaveSaveRequired` with the plan (hazard FR-062). | SRD 5.2.1, "Saving Throws", p. 7; hazard FR-038, FR-059, FR-062; Q-16 (f) | unhappy (rule interaction) |

### 4.4 Optional Frightened riders (`FearOptions`)

p. 196: "At your discretion, a Frightened creature might be subject to other effects as long as the Frightened condition lasts." The riders are the three examples, passed per command as `ToolboxRuleOptions.fear`. The enum and the options value are defined by hazard (FR-102, FR-103; value-object rules for all 8 subsets, copying and equality are hazard FR-103). They are not hazard consequences (not conditions; hazard Q-11 puts non-condition riders outside its model), so they are new here.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-027 | `fearRiderEffects(options)` MUST return one effect per enabled rider and no others, in declaration order `dashEachTurn`, `attackersHaveAdvantage`, `oneActionKindPerTurn`: `MustDashAwayFromSource`; `AttackersHaveAdvantage` (attack rolls against the Frightened creature have Advantage); `OneOfMoveActionOrBonusAction`. | SRD 5.2.1, "Fear Effects", p. 196 | happy |
| FR-028 | `ResolveFearEffectFrightened` MUST carry `fearRiderEffects(options)`. `ResolveFearEffectResisted` and `ResolveFearEffectSaveRequired` MUST carry no rider effects. | SRD 5.2.1, "Fear Effects", p. 196 ("as long as the Frightened condition lasts") | unhappy (rule interaction) |
| FR-029 | Options MUST affect only the command they are passed in: two calls in the same process with different `FearOptions` MUST give independent results, and no static, top-level or environment setting may change any result. | `doc/rules/architecture.md` §3, §10; `toolbox-hazard-effects.md` FR-101 | unhappy (rule interaction) |
| FR-030 | A rider effect MUST be an immutable value equal by value; `AttackersHaveAdvantage` is data for the host to merge into attack-roll Advantage sources (`modifiers-and-advantage.md`), not applied here. | `doc/rules/architecture.md` §4; `shared-types-and-ownership.md` FR-026 | happy |

Table 4.4a: rider subsets (`D` = `dashEachTurn`, `A` = `attackersHaveAdvantage`, `O` = `oneActionKindPerTurn`); source p. 196 for every row; one test per row (FR-027).

| Case | Riders | Expected `fearRiderEffects` | Path |
|---|---|---|---|
| none | {} | [] | unhappy (empty and minimal) |
| D only | {D} | [MustDashAwayFromSource] | happy |
| A only | {A} | [AttackersHaveAdvantage] | happy |
| O only | {O} | [OneOfMoveActionOrBonusAction] | happy |
| D and A | {D, A} | [MustDashAwayFromSource, AttackersHaveAdvantage] | unhappy (rule interaction) |
| D and O | {D, O} | [MustDashAwayFromSource, OneOfMoveActionOrBonusAction] | unhappy (rule interaction) |
| A and O | {A, O} | [AttackersHaveAdvantage, OneOfMoveActionOrBonusAction] | unhappy (rule interaction) |
| all, supplied O, A, D | {O, A, D} | declaration order D, A, O | unhappy (ordering) |

### 4.5 Checking a Frightened creature's turn against the riders

Pure function `checkFrightenedTurn(options, turn)`. `TurnUse` holds `dash` (`DashTaken`), `movement` (`FearMovement`: `didNotMove`, `movedAwayFromSource`, `movedOtherwise`), `tookAction`, `tookBonusAction`. Result `FrightenedTurnCheck`: `FrightenedTurnAllowed` or `FrightenedTurnBreaksRiders(riders)` naming every broken rider.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | `TurnUse` MUST return an error wrapping `srderr.ErrInvalidInput` when `dash` is `asAction` and `tookAction` is false, or `asBonusAction` and `tookBonusAction` is false. | package design (internal consistency of the host's report); `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-032 | With no riders enabled, every `TurnUse` MUST give `FrightenedTurnAllowed`. | SRD 5.2.1, "Fear Effects", p. 196 | unhappy (empty and minimal) |
| FR-033 | With `dashEachTurn`, a turn without the Dash action (as action or Bonus Action) MUST break `dashEachTurn`. | SRD 5.2.1, "Fear Effects", p. 196 ("must take the Dash action on each of its turns") | happy |
| FR-034 | With `dashEachTurn`, a turn with the Dash taken but movement other than `movedAwayFromSource` MUST break `dashEachTurn`. | SRD 5.2.1, "Fear Effects", p. 196 ("uses its movement to get farther away from the source of its fear") | unhappy (boundary) |
| FR-035 | With `dashEachTurn`, a turn with the Dash taken and `movedAwayFromSource` MUST give `FrightenedTurnAllowed` (when no other rider is enabled). | SRD 5.2.1, "Fear Effects", p. 196 | happy |
| FR-036 | With `oneActionKindPerTurn`, a turn using exactly one of (any movement, an action, a Bonus Action), or none, MUST be allowed. | SRD 5.2.1, "Fear Effects", p. 196 ("only one of the following") | unhappy (boundary) |
| FR-037 | With `oneActionKindPerTurn`, a turn using two or all three MUST break it; movement counts as used for `movedAwayFromSource` and `movedOtherwise`, not for `didNotMove`. | SRD 5.2.1, "Fear Effects", p. 196 | unhappy (boundary) |
| FR-038 | With both `dashEachTurn` and `oneActionKindPerTurn`, each rider MUST be judged independently and `FrightenedTurnBreaksRiders` MUST name every broken one. A Dash taken as an action plus movement away MUST satisfy `dashEachTurn` and break `oneActionKindPerTurn` (table 4.5a; Q-7). | SRD 5.2.1, "Fear Effects", p. 196 | unhappy (rule interaction) |
| FR-039 | `attackersHaveAdvantage` MUST never be broken by a turn's content (it constrains attackers); with only that rider every `TurnUse` MUST give `FrightenedTurnAllowed`. | SRD 5.2.1, "Fear Effects", p. 196 | unhappy (rule interaction) |

Table 4.5a: `checkFrightenedTurn` cases (source p. 196 for every row; one test per row). `Allowed` is `FrightenedTurnAllowed`, `Breaks {..}` is `FrightenedTurnBreaksRiders`.

| Case | Riders | dash | movement | action | bonus action | Expected | Path |
|---|---|---|---|---|---|---|---|
| Dash action and flee | D | asAction | away | yes | no | Allowed | happy |
| Dash as Bonus Action and flee | D | asBonusAction | away | no | yes | Allowed | happy |
| no Dash | D | notTaken | away | no | no | Breaks {D} | unhappy (boundary) |
| Dash but stands still | D | asAction | didNotMove | yes | no | Breaks {D} | unhappy (boundary) |
| Dash but moves otherwise | D | asAction | movedOtherwise | yes | no | Breaks {D} | unhappy (boundary) |
| idle turn | O | notTaken | didNotMove | no | no | Allowed | unhappy (empty and minimal) |
| move only | O | notTaken | movedOtherwise | no | no | Allowed | happy |
| action only | O | notTaken | didNotMove | yes | no | Allowed | happy |
| Bonus Action only | O | notTaken | didNotMove | no | yes | Allowed | happy |
| move and action | O | notTaken | movedOtherwise | yes | no | Breaks {O} | unhappy (boundary) |
| action and Bonus Action | O | notTaken | didNotMove | yes | yes | Breaks {O} | unhappy (boundary) |
| move, action and Bonus Action | O | notTaken | away | yes | yes | Breaks {O} | unhappy (boundary) |
| Dash flee under both | D, O | asAction | away | yes | no | Breaks {O} | unhappy (rule interaction) |
| idle under both | D, O | notTaken | didNotMove | no | no | Breaks {D} | unhappy (rule interaction) |
| only the Advantage rider | A | notTaken | movedOtherwise | yes | yes | Allowed | unhappy (rule interaction) |

### 4.6 Mental Stress Effects (use case `ResolveMentalStress`)

Command: `target`, `stress` (`MentalStressEffect`), `save`, `options` (`ToolboxRuleOptions`; only `options.mentalStressOnSuccess` is read, default `MentalStressNoDamageOnSuccess`), `damageAmount` (hazard `SaveGatedDamageAmount`, default rolled; printed mode is rejected, FR-108). No `subject` (FR-104). The use case builds `stress.saveGatedEffect` (FR-040), evaluates it with the hazard domain planner and evaluator and maps the outcome. Result `ResolveMentalStressResult`: `ResolveMentalStressSaveRequired`, `ResolveMentalStressAvoided`, `ResolveMentalStressHalved`, `ResolveMentalStressFull`, `ResolveMentalStressRejected(reason)` (payloads in section 6).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | `MentalStressEffect.saveGatedEffect` (taking the `MentalStressOnSuccess`) MUST be a valid `SaveGatedEffect` with: the requirement `SaveGatedSaveRequired(stress.ability, stress.dc)` (hazard FR-008, FR-105); no Disadvantage circumstances and an empty automatic-success set; `SaveGatedOnsetImmediate`; a failure outcome of one `SaveGatedDamage(stress.damage, psychic)` and no conditions or Exhaustion; `SaveGatedNoEffectOnSuccess` for `MentalStressNoDamageOnSuccess` and `SaveGatedHalfDamageOnSuccess` for `MentalStressHalfDamageOnSuccess`; `SaveGatedNoRepeat`; `SaveGatedDurationInstantaneous`. A failed save MUST return `ResolveMentalStressFull` with the `Damage` hazard FR-040 rolls (total of the dice, type Psychic) and the events of FR-102. The damage is `SaveGatedDamage(stress.damage, psychic, SaveGatedNoPrintedAverage)` because the SRD prints no average for the sample rows (hazard FR-018, RE-M). | SRD 5.2.1, "Mental Stress Effects", p. 196; hazard FR-008, FR-018, FR-036, FR-040, FR-041 | happy |
| FR-041 | A successful save with `MentalStressNoDamageOnSuccess` MUST return `ResolveMentalStressAvoided` and MUST NOT call the `Roller`. | SRD 5.2.1, "Mental Stress Effects", p. 196 ("instead of no damage"); hazard FR-033 | happy |
| FR-042 | A successful save with `MentalStressHalfDamageOnSuccess` MUST roll the damage once and return `ResolveMentalStressHalved` with the full rolled `Damage` and the `HalfDamage` adjustment; the rounded-down arithmetic stays with `ApplyDamage` (`hit-points-and-damage.md` FR-014, FR-099). | SRD 5.2.1, "Mental Stress Effects", p. 196 ("half as much damage ... at your discretion"); "Half Damage", p. 16; hazard FR-034, FR-035 | happy |
| FR-043 | A failed save with `MentalStressHalfDamageOnSuccess` enabled MUST still return `ResolveMentalStressFull` with no adjustment (the option only changes a success). | SRD 5.2.1, "Mental Stress Effects", p. 196; hazard FR-036 | unhappy (rule interaction) |
| FR-044 | The `Damage` of every result variant that carries one MUST have damage type Psychic. | package design (FR-006); SRD 5.2.1, "Mental Stress Effects", p. 196 | happy |
| FR-045 | `MentalStressEffect` MUST take a plain `Ability` for its save, default Wisdom, and accept all six abilities: p. 196 says stress "can usually be resisted with a successful Wisdom save, but sometimes an Intelligence or Charisma save is more appropriate", a suggestion, not a restriction. No `MentalStressAbility` type exists and nothing errors or panics for Strength, Dexterity or Constitution. | SRD 5.2.1, "Mental Stress Effects", p. 196; FR-007 | unhappy (boundary) |
| FR-046 | A rolled save whose target DC differs from `stress.dc`, or whose kind is not a saving throw, MUST return `ResolveMentalStressRejected(saveDoesNotMatchEffect)` with nothing rolled (hazard FR-039), as FR-015. A `SaveGatedDamageShared` list whose length or `DamageType`s differ from the effect's single Psychic instance MUST return `ResolveMentalStressRejected(sharedDamageMismatch)` with nothing rolled and no events (hazard FR-042). | hazard FR-039, FR-042; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-047 | When damage is rolled the use case MUST request exactly one roll, `stress.damage` unchanged, through `dice.Roll` (hazard FR-070); when none is rolled (`ResolveMentalStressAvoided`, `ResolveMentalStressSaveRequired`, a valid shared `damageAmount`, a rejected call) it MUST NOT call the roller. A shared `damageAmount` supplies the `Damage` unchanged (hazard FR-042); `printed` is rejected (FR-108). | `dice-and-rolls.md` FR-050; hazard FR-040, FR-042, FR-043, FR-070, FR-089; `doc/rules/architecture.md` §4 | unhappy (determinism) |
| FR-048 | A dice port failure (the roller returned an error) or a contract violation found by `dice.Roll` (a result for another expression) MUST be returned wrapped once as `*srderr.PortError` with a nil result (hazard FR-069). | `dice-and-rolls.md` FR-051, FR-052; hazard FR-069; `doc/rules/testing.md` §4 | unhappy (failure) |
| FR-049 | Damage totals MUST follow the roller exactly: with a fixed roller returning the minimum and maximum of each sample row (1d6: 1 and 6; 3d6: 3 and 18; 9d6: 9 and 54) the `Damage` amounts MUST be those totals (table 4.6a). | SRD 5.2.1, "Sample Mental Stress Effects", p. 196 | unhappy (determinism) |
| FR-050 | The returned `Damage` MUST be accepted unchanged by `ApplyDamage` (a rolled `Damage` is reusable, `hit-points-and-damage.md` FR-100), and Half Damage for a rolled 9 MUST be 4 (rounded down) once applied. Verified at scenario level. | SRD 5.2.1, "Half Damage", p. 16; "Round Down", p. 5 | unhappy (boundary) |
| FR-052 | Resolving MUST NOT alter the command's inputs and MUST be deterministic: the same command with a fixed roller returning the same faces gives an equal result. | `doc/rules/testing.md` §4 | unhappy (determinism) |
| FR-090 | A `SaveGatedSaveForcedFailure` of kind `savingThrow` MUST give `ResolveMentalStressFull`, even with `MentalStressHalfDamageOnSuccess` (a forced failure is a failure), and the DC check is skipped. | SRD 5.2.1, "Saving Throws", p. 7; hazard FR-036, FR-038, FR-039 | unhappy (rule interaction) |
| FR-099 | A `SaveGatedSaveNotRolled` MUST return `ResolveMentalStressSaveRequired` with the plan (stress ability, DC, empty roll circumstances) and request no dice. | SRD 5.2.1, "Saving Throws", p. 7; hazard FR-030, FR-031 | happy |
| FR-102 | A failed or forced-failed save MUST give `ResolveMentalStressFull` with events exactly `[SaveGatedEffectStarted(target), SaveGatedEffectEnded(instantaneous)]` in that order and the `active` state in `SaveGatedPhaseEnded(instantaneous)`; the `SaveGatedEffectEnded` event lists no conditions and 0 Exhaustion levels. The hazard evaluator emits both events for `SaveGatedDurationInstantaneous` (hazard FR-053, FR-083); this spec adds none. `ResolveMentalStressHalved`, `ResolveMentalStressAvoided` and `ResolveMentalStressSaveRequired` emit no event (hazard FR-047). | package design (mirrors hazard FR-083); SRD 5.2.1, "Mental Stress Effects", p. 196; hazard FR-047, FR-053 | happy |

Table 4.6a: sample rows with a fixed roller. "Started/Ended" abbreviates events `[Started, Ended(instantaneous)]`. Source p. 196 unless noted. One test per row; variants are shown without the `ResolveMentalStress` prefix.

| Case | DC | Dice | Save | Options | Roller returns | Expected | Path |
|---|---|---|---|---|---|---|---|
| hallucinogen, fail, minimum | 10 | 1d6 | failure | default | 1 | Full, Started/Ended, 1 Psychic | unhappy (boundary) |
| hallucinogen, fail, maximum | 10 | 1d6 | failure | default | 6 | Full, Started/Ended, 6 Psychic | unhappy (boundary) |
| idol, fail, minimum | 15 | 3d6 | failure | default | 1,1,1 | Full, Started/Ended, 3 Psychic | unhappy (boundary) |
| idol, fail, maximum | 15 | 3d6 | failure | default | 6,6,6 | Full, Started/Ended, 18 Psychic | unhappy (boundary) |
| Far Realm, fail, minimum | 20 | 9d6 | failure | default | nine 1s | Full, Started/Ended, 9 Psychic | unhappy (boundary) |
| Far Realm, fail, maximum | 20 | 9d6 | failure | default | nine 6s | Full, Started/Ended, 54 Psychic | unhappy (boundary) |
| idol, success, default | 15 | 3d6 | success | default | not called | Avoided, no events | happy |
| idol, success, half | 15 | 3d6 | success | half | 4,4,4 | Halved, 12 Psychic and HalfDamage, no events | happy |
| idol, fail, half enabled | 15 | 3d6 | failure | half | 4,4,4 | Full, Started/Ended, 12 Psychic, no adjustment | unhappy (rule interaction) |
| idol, chosen failure, half enabled | 15 | 3d6 | forced failure | half | 4,4,4 | Full, Started/Ended, 12 Psychic, no adjustment, no DC check (p. 7) | unhappy (rule interaction) |
| Charisma save | 15 | 3d6 | failure | default | 2,2,2 | Full, Started/Ended, 6 Psychic (Charisma) | happy |
| Strength save | 10 | 1d6 | failure | default | 3 | Full, Started/Ended, 3 Psychic (Strength accepted) | unhappy (boundary) |
| no save yet | 15 | 3d6 | not rolled | default | not called | SaveRequired, Wisdom, DC 15 (p. 7) | happy |
| shared damage, fail | 15 | 3d6 | failure | `damageAmount` shared `[12 Psychic]` | not called | Full, Started/Ended, the supplied 12 Psychic (p. 16) | unhappy (rule interaction) |
| shared damage, wrong type | 15 | 3d6 | failure | shared `[12 Fire]` | not called | Rejected(sharedDamageMismatch), no events | unhappy (invalid input) |
| shared damage, wrong length | 15 | 3d6 | failure | shared `[]` | not called | Rejected(sharedDamageMismatch), no events | unhappy (invalid input) |
| printed average, fail | 15 | 3d6 | failure | `damageAmount` printed | not called | Rejected(noPrintedAverage), no events | unhappy (invalid input) |
| printed average, no save yet | 15 | 3d6 | not rolled | `damageAmount` printed | not called | SaveRequired (the save is checked first) | unhappy (invalid input) |
| printed average, success, default | 15 | 3d6 | success | `damageAmount` printed | not called | Avoided (no damage returned, so not rejected) | unhappy (rule interaction) |

### 4.7 Prolonged Effects

`ProlongedEffect` is an immutable value: `target` (`CombatantID`, set by `StartProlongedEffect` and carried unchanged, FR-103), `term`, `duration` (hazard `SaveGatedDuration`: the resolved starting span, or `SaveGatedDurationUntilEnded` for indefinite), `remaining` (a `time.Duration`, absent exactly for indefinite, FR-093), `status` and the variant detail (`ShortTermCondition` for short-term; `AbilityCheckScope` for long-term and indefinite). A host holds one per affected creature. The use cases are thin: they call the pure domain functions `prolongedDiceRequired`, `ProlongedEffect.start`, `.elapse`, `.suppress`, `.resume`, `.remove`, `decideProlongedCure` and the query `ProlongedEffect.rollCircumstanceFor`, and add telemetry.

#### 4.7.1 Starting an effect (use case `StartProlongedEffect`)

Command: `target`, `spec` (closed interface `ProlongedEffectSpec`: `ProlongedShortTermSpec(condition)`, `ProlongedLongTermSpec(scope)`, `ProlongedIndefiniteSpec(scope)`); no `ConditionSet` (FR-057). Result: `StartProlongedEffectStarted(effect, conditionToApply, events)`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-053 | `ProlongedTerm` MUST have exactly three values: `shortTerm`, `longTerm`, `indefinite`. | SRD 5.2.1, "Prolonged Effects", p. 196 | happy |
| FR-107 | `prolongedDiceRequired(spec)` MUST return the ordered `dice.Expression`s `StartProlongedEffect` rolls: `[1d10]` for short-term, `[1d10]` for long-term, `[]` for indefinite. The use case MUST roll exactly that list through `dice.Roll` and MUST decide nothing else about dice (as hazard FR-087). | SRD 5.2.1, "Prolonged Effects", p. 196; hazard FR-087; `doc/rules/architecture.md` §4, §10 | happy |
| FR-054 | A short-term effect MUST roll `1d10` and last that many minutes: a roll of n (1 to 10) gives `SaveGatedDurationFor(SaveGatedFixedTime(n, DurationUnit.minutes))` (table 4.7a). | SRD 5.2.1, "Short-Term Effects", p. 196 | happy |
| FR-055 | A long-term effect MUST roll `1d10` and last that roll multiplied by 10 hours: n gives `SaveGatedDurationFor(SaveGatedFixedTime(10 x n, DurationUnit.hours))`, 10 to 100 hours. The domain multiplies the roll's `total`, because a multiplied roll is not a `dice.Expression` (`dice-and-rolls.md`, "Multiplied dice") and hazard's `SaveGatedRolledTime` holds a `dice.Expression`. | SRD 5.2.1, "Long-Term Effects", p. 196 | happy |
| FR-056 | An indefinite effect MUST NOT call the `Roller` and MUST have `SaveGatedDurationUntilEnded` and no `remaining`. | SRD 5.2.1, "Indefinite Effects", p. 196 | happy |
| FR-057 | A short-term effect MUST hand back exactly one of Frightened, Incapacitated or Stunned as `conditionToApply` (`ProlongedConditionNamed(kind, source)`); the engine MUST NOT apply it or read a `ConditionSet`. The host applies it; Stunned gives an effective Incapacitated through `conditions.md` FR-026 without an extra event. | SRD 5.2.1, "Short-Term Effects", p. 196; hazard FR-046; `conditions.md` FR-026 | happy |
| FR-058 | `ShortTermFrightened` MUST carry a source of fear (`CombatantID`) and the handed-back condition MUST be `frightened` with that source; `ShortTermIncapacitated` and `ShortTermStunned` carry no creature and their handed-back source is absent. | SRD 5.2.1, "Short-Term Effects", p. 196; `conditions.md` FR-003 | happy |
| FR-059 | If the short-term condition is already held or the target is immune, the host's `ConditionSet.apply` returns `ConditionAlreadyHeld` or `ConditionImmune` (`conditions.md` FR-018, FR-021, FR-125); the host MUST discard the returned `ProlongedEffect` and MUST NOT publish its `ProlongedEffectStarted` (table 4.9b; Q-9). The engine returns no variant for it. | `conditions.md` FR-018, FR-021; SRD 5.2.1, "Condition", p. 179; Q-9 | unhappy (rule interaction) |
| FR-060 | A long-term or indefinite effect MUST hand back `ProlongedConditionNone`. | SRD 5.2.1, "Long-Term Effects", p. 196 | unhappy (rule interaction) |
| FR-061 | `AbilityCheckScope` MUST be `ProlongedAllAbilityChecks` or `ProlongedSomeAbilityChecks` over a non-empty set of `Ability`; an empty set MUST return an error wrapping `srderr.ErrInvalidInput`, and the set is copied and defensively copied. | SRD 5.2.1, "Long-Term Effects", p. 196 ("Disadvantage on some or all ability checks") | unhappy (empty and minimal) |
| FR-062 | `ProlongedEffect.rollCircumstanceFor(Ability a)` MUST return a `Disadvantage` for a long-term or indefinite effect exactly when its scope is `ProlongedAllAbilityChecks` or its `ProlongedSomeAbilityChecks` contains `a`, and only while `status` is `active`; otherwise (short-term, suppressed, ability outside the scope) it reports none (`ok` false). It never returns an `Advantage`; the circumstance is data for the host to merge (it does not stack, `modifiers-and-advantage.md`) and is not applied here. | SRD 5.2.1, "Long-Term Effects", p. 196; `shared-types-and-ownership.md` FR-026 | unhappy (boundary) |
| FR-063 | A newly started effect MUST have status `active`, and `StartProlongedEffectStarted` MUST carry one `ProlongedEffectStarted` event naming the target and term. | SRD 5.2.1, "Prolonged Effects", p. 196 | happy |
| FR-064 | A dice port failure MUST be returned wrapped once as `*srderr.PortError` with a nil result (as FR-048). | `dice-and-rolls.md` FR-051, FR-052; hazard FR-069 | unhappy (failure) |

Table 4.7a: duration rolls (source p. 196).

| Case | Term | Roller returns | Expected duration | Expected `remaining` | Path |
|---|---|---|---|---|---|
| short, minimum | shortTerm | 1 | 1 minutes | 1 minute | unhappy (boundary) |
| short, middle | shortTerm | 5 | 5 minutes | 5 minutes | happy |
| short, maximum | shortTerm | 10 | 10 minutes | 10 minutes | unhappy (boundary) |
| long, minimum | longTerm | 1 | 10 hours | 10 hours | unhappy (boundary) |
| long, middle | longTerm | 5 | 50 hours | 50 hours | happy |
| long, maximum | longTerm | 10 | 100 hours | 100 hours | unhappy (boundary) |
| indefinite | indefinite | not called | until ended | absent | happy |

#### 4.7.2 Time passing (use case `ElapseProlongedEffect`)

Command: `effect`, `elapsed` (a `Duration`, never read from a clock). Result: `ElapseProlongedEffectContinues(effect, events)`, `ElapseProlongedEffectEnded(conditionToRemove, events)`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-065 | Elapsed time shorter than `remaining` MUST return `ElapseProlongedEffectContinues` with `remaining` reduced by exactly `elapsed` (FR-093), no events and the other fields unchanged. | SRD 5.2.1, "Short-Term Effects", "Long-Term Effects", p. 196 | happy |
| FR-066 | Elapsed time equal to or greater than `remaining` MUST end the effect: `ElapseProlongedEffectEnded` with one `ProlongedEffectEnded` event. One microsecond below `remaining` MUST continue (table 4.7b). | SRD 5.2.1, "Prolonged Effects", p. 196 | unhappy (boundary) |
| FR-067 | When an `active` short-term effect ends, the result MUST hand back its condition in `conditionToRemove` (Frightened with its source, FR-021) besides the event; the engine does not call `ConditionSet.remove`. A long-term effect ending MUST hand back `ProlongedConditionNone`. Host `remove` outcomes are table 4.9a. | SRD 5.2.1, "Short-Term Effects", p. 196; `conditions.md` FR-031 to FR-033; hazard FR-046 | happy |
| FR-068 | An indefinite effect MUST never end through elapsed time, whatever the value. | SRD 5.2.1, "Indefinite Effects", p. 196 | unhappy (boundary) |
| FR-069 | An `elapsed` of zero MUST change nothing (the effect returned equals the input) and a negative `elapsed` MUST return an error wrapping `srderr.ErrInvalidInput` at construction. | `doc/rules/architecture.md` §6; `doc/rules/testing.md` §4 | unhappy (boundary) |
| FR-070 | A suppressed effect MUST still lose remaining duration as time passes (Q-10); if it ends while suppressed the result MUST hand back `ProlongedConditionNone` (the condition was already taken off, FR-077) and the host removes nothing. | SRD 5.2.1, "Prolonged Effects", p. 196 (silent on pausing); "Antimagic Field", p. 109 ("the time it spends suppressed counts against its duration"); Q-10 | unhappy (rule interaction) |
| FR-071 | `ElapseProlongedEffect` MUST NOT alter its inputs and MUST be deterministic (no clock, no dice). | `doc/rules/architecture.md` §3 | unhappy (immutability) |
| FR-093 | `Remaining` MUST be a `time.Duration` (read as `(time.Duration, bool)`) holding the starting span converted exactly (`DurationUnit.minutes` = 60 s, `DurationUnit.hours` = 3600 s) and reduced by the exact `elapsed` with no rounding; present (`ok` true) for short-term and long-term, absent (`ok` false) for indefinite (never a sentinel such as zero). `duration` keeps the resolved starting value for display. | package design; SRD 5.2.1, "Short-Term Effects", "Long-Term Effects", p. 196 | unhappy (boundary) |
| FR-094 | `ProlongedEffect` MUST return an error wrapping `srderr.ErrInvalidInput` at construction for: `shortTerm` with an `AbilityCheckScope` detail, or `longTerm`/`indefinite` with a `ShortTermCondition` detail; `indefinite` with a `SaveGatedDurationFor` or a present `remaining`; `shortTerm`/`longTerm` with `SaveGatedDurationUntilEnded` or an absent `remaining`; a `remaining` of zero or negative. `target` is required with no further invariant. | package design | unhappy (invalid input) |
| FR-103 | `ProlongedEffect` MUST hold the `target` (`CombatantID`) of the creature it is on. `StartProlongedEffect` MUST set it from the command's `target`; `.elapse`, `.suppress`, `.resume` and `.remove` MUST return effects with the same `target`; every `ProlongedEffectEvent` MUST name the `target` of the effect it was built from. `ElapseProlongedEffectCommand` (`effect`, `elapsed`) and `ChangeProlongedEffectCommand` (`effect`, `change`) need no separate target field. | package design (FR-085); `doc/rules/architecture.md` §3 | unhappy (rule interaction) |

Table 4.7b: elapsing (source p. 196; one test per row). `Continues` and `Ended` are `ElapseProlongedEffectContinues` and `ElapseProlongedEffectEnded`.

| Case | Term, duration | Status | Elapsed | Expected | Path |
|---|---|---|---|---|---|
| short, one minute short | shortTerm, 5 min | active | 4 min | Continues, remaining 1 min | unhappy (boundary) |
| short, sub-minute | shortTerm, 5 min | active | 30 s | Continues, remaining 4 min 30 s | unhappy (boundary) |
| short, one microsecond short | shortTerm, 5 min | active | 5 min minus 1 microsecond | Continues, remaining 1 microsecond | unhappy (boundary) |
| short, exactly | shortTerm, 5 min | active | 5 min | Ended, condition handed back | unhappy (boundary) |
| short, over | shortTerm, 5 min | active | 6 min | Ended | unhappy (boundary) |
| short, accumulated | shortTerm, 5 min | active | 30 s then 30 s (two calls) | second call Continues, remaining 4 min | unhappy (rule interaction) |
| long, mixed units | longTerm, 10 h | active | 599 min | Continues, remaining 1 min | unhappy (boundary) |
| long, sub-minute | longTerm, 10 h | active | 599 min 30 s | Continues, remaining 30 s | unhappy (boundary) |
| long, exactly | longTerm, 10 h | active | 10 h | Ended, `ProlongedConditionNone` | unhappy (boundary) |
| indefinite | indefinite | active | 100,000 h | Continues, unchanged | unhappy (boundary) |
| zero elapsed | shortTerm, 5 min | active | 0 | Continues, effect equal to the input | unhappy (empty and minimal) |
| suppressed short, ends | shortTerm, 5 min | suppressed | 5 min | Ended, `ProlongedConditionNone` | unhappy (rule interaction) |

#### 4.7.3 Suppressing, resuming and curing (use case `ChangeProlongedEffect`)

Command: `effect`, `change` (closed interface `ProlongedChange`: `ProlongedResume`, `ProlongedCureWith(cure)`). Result: `ChangeProlongedEffectSuppressed(effect, conditionToRemove, events)`, `ChangeProlongedEffectResumed(effect, conditionToApply, events)`, `ChangeProlongedEffectRemoved(conditionToRemove, events)`, `ChangeProlongedEffectRejected(reason)` with `ProlongedRejection` (`cureNotEffective`, `alreadySuppressed`, `notSuppressed`). The SRD names which cure does what to which term (table 4.7c); every other pair is not modelled. `decideProlongedCure(term, cure)` returns `ProlongedCureSuppress`, `ProlongedCureRemove` or `ProlongedCureNotEffective`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-072 | `decideProlongedCure` MUST give `ProlongedCureSuppress` for `calmEmotions` on every term, and `ChangeProlongedEffect` MUST then suppress an `active` effect: status `suppressed` and one `ProlongedEffectSuppressed` event. The SRD says "can be suppressed with the Calm Emotions spell" (short- and long-term) and "can be suppressed by a Calm Emotions spell" (indefinite), with no "usually", so the matrix has no probabilistic or exception clause. | SRD 5.2.1, "Prolonged Effects", p. 196 | happy |
| FR-073 | `decideProlongedCure` MUST give `ProlongedCureRemove` for `lesserRestoration` on `shortTerm` and `longTerm`, and the effect (`active` or `suppressed`) MUST be removed: `ChangeProlongedEffectRemoved` with one `ProlongedEffectRemoved` event. | SRD 5.2.1, "Short-Term Effects", "Long-Term Effects", p. 196 ("removed by the Lesser Restoration spell") | happy |
| FR-074 | `decideProlongedCure` MUST give `ProlongedCureRemove` for `greaterRestoration` on `indefinite`, and the effect (`active` or `suppressed`) MUST be removed with one `ProlongedEffectRemoved` event. | SRD 5.2.1, "Indefinite Effects", p. 196 | happy |
| FR-075 | **Interpretation:** `lesserRestoration` on `indefinite` MUST give `ProlongedCureNotEffective` and `ChangeProlongedEffectRejected(cureNotEffective)`, the effect unchanged, no events. p. 196 calls an indefinite effect "a long-term effect (see above)" that "lasts until removed by a Greater Restoration spell", while long-term effects can be "removed by the Lesser Restoration spell"; the package reads the indefinite sentence as naming only Greater Restoration (Q-8, settled). | SRD 5.2.1, "Long-Term Effects", "Indefinite Effects", p. 196; Q-8 | unhappy (rule interaction) |
| FR-076 | Any pair not named in FR-072 to FR-074 (Greater Restoration on a short- or long-term effect) MUST give `ProlongedCureNotEffective` and `ChangeProlongedEffectRejected(cureNotEffective)` with nothing changed; the package invents no effect the page does not state. The matrix is decided before the status check, so a not-effective pair is `cureNotEffective` whatever the status. | SRD 5.2.1, "Prolonged Effects", p. 196; Q-8 | unhappy (rule interaction) |
| FR-077 | Suppressing a short-term effect MUST hand back its condition in `conditionToRemove` (Frightened with its source); suppressing a long-term or indefinite effect MUST hand back `ProlongedConditionNone` (Q-10). The engine does not call `ConditionSet.remove` (table 4.9a). | SRD 5.2.1, "Prolonged Effects", p. 196; hazard FR-046; Q-10 | happy |
| FR-078 | Suppression happens only through `ProlongedCureWith(calmEmotions)`; `ProlongedResume` MUST set status `active` and, for a short-term effect, hand back its condition in `conditionToApply` (with the Frightened source) with one `ProlongedEffectResumed` event. | SRD 5.2.1, "Prolonged Effects", p. 196; Q-10 | happy |
| FR-079 | Suppressing an already `suppressed` effect MUST return `ChangeProlongedEffectRejected(alreadySuppressed)`; resuming an `active` one MUST return `ChangeProlongedEffectRejected(notSuppressed)`; both with nothing changed and no events. | `doc/rules/testing.md` §4 | unhappy (state error) |
| FR-080 | When the host's re-application of a resumed short-term condition is blocked (`ConditionAlreadyHeld` or `ConditionImmune`), the host MUST keep its stored `suppressed` effect, discard the returned `ChangeProlongedEffectResumed` effect and not publish `ProlongedEffectResumed` (table 4.9b). The engine has no `ConditionBlocked` variant because it never sees the `ConditionSet`. | `conditions.md` FR-018, FR-021, FR-125; hazard FR-046 | unhappy (rule interaction) |
| FR-081 | Removing an `active` short-term effect MUST hand back its condition in `conditionToRemove`; removing a `suppressed` one MUST hand back `ProlongedConditionNone`. | SRD 5.2.1, "Short-Term Effects", p. 196; `conditions.md` FR-031, FR-032 | unhappy (rule interaction) |
| FR-082 | `ChangeProlongedEffect` MUST NOT alter its inputs. | `doc/rules/testing.md` §4 | unhappy (immutability) |

Table 4.7c: cure matrix, one test per cell (`—` = `ProlongedCureNotEffective`; source p. 196).

| Cure | shortTerm | longTerm | indefinite |
|---|---|---|---|
| Calm Emotions | suppress | suppress | suppress |
| Lesser Restoration | remove | remove | — |
| Greater Restoration | — | — | remove |

### 4.8 Cross-cutting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-083 | Every command, result, effect, event and value in this spec MUST be immutable and equal by value; reachable collections MUST be defensively copied. | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |
| FR-084 | The domain types and functions (`SampleFearDc`, `SampleMentalStress`, `FearEffect.saveGatedEffect`, `MentalStressEffect.saveGatedEffect`, `FearOptions`, `fearRiderEffects`, `checkFrightenedTurn`, `decideProlongedCure`, `prolongedDiceRequired`, `ProlongedEffect.rollCircumstanceFor` and the `ProlongedEffect` rules) MUST be pure, synchronous and silent: no `math/rand`, `time.Now`, `context.Context` or telemetry. | `doc/rules/architecture.md` §3; `doc/rules/observability.md` §3 | happy |
| FR-085 | `ProlongedEffectStarted`, `Suppressed`, `Resumed`, `Ended` and `Removed` (each `ProlongedEffect...`) MUST be immutable past-tense value objects carrying the target `CombatantID` and `ProlongedTerm`, variants of one closed interface `ProlongedEffectEvent` that embeds `ToolboxEvent` (defined once by hazard FR-086; `shared-types-and-ownership.md` FR-019). As hazard FR-086 does for `SaveGatedEvent`, the family is declared in one file (`toolbox/prolonged_effect_event.go`) with an unexported marker method and a doc comment listing the five variants, so the host's exhaustive type switch compiles. Fear and mental stress define no event of their own: they use hazard's `SaveGatedEffectStarted`, `SaveGatedEffectRepeatSaveMade` and `SaveGatedEffectEnded`. | `doc/rules/architecture.md` §3; `shared-types-and-ownership.md` FR-019; hazard section 5 | happy |
| FR-086 | The use cases MUST import only the published API of upstream contexts (`combat`, `d20`, `content`, `character` as needed) and the kernel packages (the roller port is a consumer-side interface), and no other context's `internal/` packages; within `toolbox` they may use hazard's **domain** types and services but MUST NOT call a hazard use case or any other use case (`shared-types-and-ownership.md` FR-044). | `doc/rules/architecture.md` §1, §2, §9; `architecture-guardrails.md` FR-018, FR-019 (import legality), FR-097; `shared-types-and-ownership.md` FR-044, FR-045 | unhappy (forbidden edge) |
| FR-087 | The use cases MUST take no content from a `ContentSource`: every table is held in the domain, and no `MissingContent` result exists. | `shared-types-and-ownership.md` FR-018 (n/a here); Q-4 | happy |
| FR-092 | This spec MUST define no type already defined by hazard (`SaveGatedEffect`, `SaveGatedSave`, `SaveGatedDamageAmount`, `SaveGatedConsequences`, `ActiveSaveGatedEffect`, `SaveGatedDuration`, `SaveGatedTime` variants, `SaveGatedRollPlan`, `SaveGatedRejection`, and the option types `FearRider`, `FearOptions`, `MentalStressOnSuccess`, hazard FR-102 to FR-104) and no name in the "names deliberately avoided" list of section 3. The exported-name scan MUST find each exported name exactly once in package `toolbox`, none generic, including the functions `fearRiderEffects`, `checkFrightenedTurn`, `decideProlongedCure`, `prolongedDiceRequired` and the query `ProlongedEffect.rollCircumstanceFor`. | `doc/rules/architecture.md` §9; hazard section 3, FR-006; `shared-types-and-ownership.md` FR-017 | unhappy (name collision) |
| FR-095 | The `toolbox` published language of this spec MUST be exported from package `toolbox`, each name once with a doc comment starting with its name, **in the same change** that adds the use cases (as hazard FR-006), so the scenario test of FR-050 imports only the public API of the module's packages. | hazard FR-006; `doc/rules/testing.md` §3; `doc/rules/architecture.md` §4, §7 | happy |
| FR-100 | `ResolveFearEffectCommand` and `ResolveMentalStressCommand` MUST carry their rule options as one `ToolboxRuleOptions` (default `standard`); each use case MUST read only its own field (`fear`, `mentalStressOnSuccess`) and MUST give the same result as a command whose other field differs. No other command takes rule options. | `shared-types-and-ownership.md` FR-045; `toolbox-hazard-effects.md` FR-099, FR-101 | unhappy (rule interaction) |
| FR-101 | `ResolveFearEffectRejected`, `ResolveMentalStressRejected` and `ResolveFearRepeatSaveRejected` MUST carry the hazard `SaveGatedRejection` as `reason`: `ResolveFearEffectRejected` only `saveDoesNotMatchEffect`; `ResolveFearRepeatSaveRejected` `effectNotActive` or `saveDoesNotMatchEffect`; `ResolveMentalStressRejected` `saveDoesNotMatchEffect`, `sharedDamageMismatch` (FR-046) or `noPrintedAverage` (FR-108). None can return `noRepeatSaves` or `effectNotAwaitingOnset`, and the fear use cases never return `sharedDamageMismatch`. | hazard FR-039, FR-042, FR-061, FR-085, FR-089; `shared-types-and-ownership.md` FR-018 | unhappy (invalid input) |
| FR-106 | `ChangeProlongedEffectRejected(cureNotEffective)` is a **normal rule outcome, not an error** (a legal cast of a real spell the p. 196 matrix does not list, FR-075, FR-076). Its telemetry MUST be `srd5e.outcome=success`, `srd5e.toolbox.result=cure_not_effective`, no span error status, no log and no `error.type`, counted once by result. `alreadySuppressed` and `notSuppressed` remain state errors: `srd5e.outcome=failure`, span error status, results `rejected_already_suppressed` / `rejected_not_suppressed`. | `doc/rules/observability.md` §7; `doc/rules/testing.md` §4; hazard FR-073 | unhappy (rule interaction) |
| FR-108 | With a `damageAmount` of `SaveGatedDamagePrintedAverage`, `ResolveMentalStress` MUST map hazard's `Rejected(noPrintedAverage)` to `ResolveMentalStressRejected(noPrintedAverage)` for a failed or forced-failed save and for a successful save with `MentalStressHalfDamageOnSuccess` (the sample rows print no number): nothing rolled, no damage, no events. The check follows save validation (hazard FR-039): a `SaveGatedSaveNotRolled` still gives `ResolveMentalStressSaveRequired`, a mismatched save `ResolveMentalStressRejected(saveDoesNotMatchEffect)`, and a successful save with `MentalStressNoDamageOnSuccess` gives `ResolveMentalStressAvoided` (no damage is returned, so nothing is rejected). | package design; SRD 5.2.1, "Sample Mental Stress Effects", p. 196; hazard FR-018, FR-039, FR-089 | unhappy (invalid input) |
| FR-109 | A `SaveGatedSaveForcedFailure` whose kind is not `D20TestKind.savingThrow` MUST return the use case's `...Rejected(saveDoesNotMatchEffect)` (`ResolveFearEffectRejected`, `ResolveFearRepeatSaveRejected`, `ResolveMentalStressRejected`) with nothing rolled and no events; a forced failure carries no DC, so only its kind is checked. On a repeat save an ended or awaiting state still wins (FR-022). | SRD 5.2.1, "Saving Throws", p. 7; hazard FR-039, FR-085; `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-110 | the `toolbox` signal constants MUST gain the constants for this spec's six `srd5e.toolbox.operation` values, its `result` values (section 7 table, including the `rejected_...` values, `cure_not_effective` and `error` for `resolve_mental_stress` and `start_prolonged_effect`), the shared attribute key `srd5e.toolbox.prolonged_term` (`short_term`, `long_term`, `indefinite`; only on the three prolonged spans), use of the shared `srd5e.toolbox.save_ability` key on the save spans, `saves` and `mental_stress.damage`, and the two signals of section 7, in the change that adds the use cases; no second signals class. | `toolbox-hazard-effects.md` section 7 (shared contract), FR-006; `doc/rules/observability.md` §5 | happy |

### 4.9 Host protocol for conditions handed back

The engine hands back a `ProlongedConditionHandBack` (`ProlongedConditionNone` or `ProlongedConditionNamed(kind, source)`, `source` present exactly for `frightened`) or, for a fear, `ConditionKind.frightened` plus `fear.source`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-096 | The lifecycle outcome of an engine result (a fear ended, a prolonged effect ended, suppressed or removed) MUST NOT depend on the outcome of the host's `ConditionSet.remove`: the effect is over whatever `remove` returns, and the result is the same for every outcome in table 4.9a. | hazard FR-046, FR-067; `conditions.md` FR-125 | unhappy (rule interaction) |
| FR-097 | The engine MUST emit no `ConditionApplied`, `ConditionRemoved` or `ConditionLevelReduced` event; those are the host's (`conditions.md` FR-129). When `remove` returns `ConditionRemovalNotPresent` or `ConditionRemovalImpliedBy` no `ConditionRemoved` exists and the host MUST NOT publish one. | `conditions.md` FR-115, FR-129; hazard FR-046 | unhappy (rule interaction) |

Table 4.9a: the host's `ConditionSet.remove(kind)` outcome after a fear ended, a short-term effect ended, was suppressed or removed (`conditions.md` FR-031 to FR-033, FR-125). Engine result unchanged in every row.

| Host `remove` outcome | Meaning | Host action | Source |
|---|---|---|---|
| `ConditionCleared` | explicit condition removed | publish the one `ConditionRemoved` | `conditions.md` FR-031 |
| `ConditionRemovalNotPresent` | already ended by other means | nothing; no `ConditionRemoved` | `conditions.md` FR-032 |
| `ConditionRemovalImpliedBy(kinds)` | effective only through another explicit condition (for example Incapacitated while Paralyzed) | nothing; no `ConditionRemoved`; the implying condition stays | `conditions.md` FR-033 |

Pre-check (FR-021): before calling `remove` for a Frightened handed back with a `source`, the host checks that the Frightened the target holds has that source; if not, another fear holds the condition and the host MUST NOT call `remove` (`conditions.md` FR-019, FR-031).

Table 4.9b: the host's `ConditionSet.apply(condition)` outcome after a fear failed, a short-term effect started or one was resumed. Engine result unchanged in every row.

| Host `apply` outcome | Host action | Source |
|---|---|---|
| `ConditionAdded` | persist the returned `ActiveFear` or `ProlongedEffect`; publish its events | `conditions.md` FR-016 |
| `ConditionAlreadyHeld` | discard the returned state (for a resume keep the stored `suppressed` effect); the original Frightened source is kept | `conditions.md` FR-018, FR-019 |
| `ConditionImmune` | discard the returned state (for a resume keep the stored `suppressed` effect) | `conditions.md` FR-021 |

### 4.10 Withdrawn ids

| Withdrawn id | Moved to |
|---|---|
| FR-024, FR-025, FR-026 | `toolbox-hazard-effects.md` FR-103 |
| FR-051 | `toolbox-hazard-effects.md` FR-104 |
| FR-105 | `toolbox-hazard-effects.md` FR-089 |
| FR-091 | never allocated here (section 3 cites hazard's own FR-091) |

## 5. Model

Everything in the domain files of package `toolbox` for this spec (not `*_usecase.go`) is pure, synchronous and silent. Hazard's types and `ToolboxRuleOptions` are referenced, not defined. The value objects, enums, closed interface types and domain functions are those named in sections 3 and 4; there are no entities or aggregates (the host holds the `ActiveFear` and `ProlongedEffect` values it gets back, one per affected creature). Where the rules live: the two `saveGatedEffect` builders, the pure functions `fearRiderEffects`, `checkFrightenedTurn`, `decideProlongedCure`, `prolongedDiceRequired` and `ProlongedEffect.start` (takes `target`, spec and the already-rolled `1d10` total), `.elapse`, `.suppress`, `.resume`, `.remove` and `rollCircumstanceFor`, with no `ConditionSet` knowledge. "Failed save gives Frightened", "repeat save ends after one success" and "half damage only on a success with the option" are the hazard evaluator's decisions.

**Results versus errors** (`doc/rules/architecture.md` §6). Failed or resisted saves, repeat saves that continue or end and a cure that does not apply (a normal outcome, FR-106) are results; `...Rejected` results cover state and input errors (FR-015, FR-016, FR-020, FR-046, FR-079, FR-108, FR-109). `ErrInvalidInput` error (programmer error, at construction): FR-022, FR-031, FR-061, FR-069, FR-094. A `Roller` failure is returned as a `*srderr.PortError` (a nil result).

### 5.1 Why a parallel type stays: `ProlongedEffect`

`FearEffect` and `MentalStressEffect` are inputs to builders returning a `SaveGatedEffect`; the lifecycle is hazard's. `ProlongedEffect` stays because hazard's cannot express it:

1. **No saving throw.** `SaveGatedEffect` requires an `Ability` and a `DifficultyClass` (hazard FR-008); p. 196 links prolonged effects to "exposure to mental stress", not to a save (hazard Q-10; Q-11).
2. **A multiplied duration.** "1d10 x 10 hours" is not a `dice.Expression`, so `SaveGatedRolledTime` cannot hold it (FR-055).
3. **Suppression.** Hazard's phases have no "suppressed then resumed" (Q-10; SRD 5.2.1, "Antimagic Field", p. 109, is the one SRD rule for it).
4. **A non-condition outcome.** Disadvantage on ability checks is outside hazard's model (hazard Q-11).

Even if hazard later adds `SaveGatedNoSave`, `ProlongedEffect` stays as specified (Q-16 (d)): long-term scope, the multiplied duration and suppression need it, and one type for all three terms is simpler for hosts.

## 6. Interface

Six use cases, each a struct built with `NewX(...)` and options `WithTelemetry` (default `telemetry.Noop`) and `WithClock`, and called as `Run(ctx context.Context, cmd) (<UseCase>Result, error)` (`ctx` first because each emits a span; the `error` carries only a port failure or a cancelled `ctx`): `ResolveFearEffect`, `ResolveFearRepeatSave`, `ResolveMentalStress` (takes a `Roller`), `StartProlongedEffect` (takes a `Roller`), `ElapseProlongedEffect` and `ChangeProlongedEffect` (no dice, no clock). `Roller` is the consumer-side interface of toolbox-hazard-effects.md FR-001. Signatures are illustrative; the requirements fix behaviour.

- **Sync.** Every port used is synchronous (`doc/rules/architecture.md` §4).
- **Options.** Only `ResolveFearEffectCommand` and `ResolveMentalStressCommand` carry `options: ToolboxRuleOptions` (FR-100).
- **Ports.** `Roller` for `ResolveMentalStress` and `StartProlongedEffect`; `telemetry.Telemetry` for all; no `ContentSource`. A dice port failure is returned wrapped once as `*srderr.PortError` and recorded once where it leaves srd5e (hazard FR-069, FR-074); there is no expected-absence result because there is no lookup.
- **Results.** One closed interface `<UseCase>Result` per use case with operation-prefixed variants, no generic variant name (`shared-types-and-ownership.md` FR-017) and the rejection `reason` as a field.
- **Host protocol.** Fear: call `ResolveFearEffect` with `SaveGatedSaveNotRolled`; on `SaveRequired` roll with `d20.Resolver` using the plan and call again; on `Frightened` call `ConditionSet.apply` (table 4.9b) and persist the `ActiveFear`; call `ResolveFearRepeatSave` at the end of each of the creature's turns; on `Ended` call `ConditionSet.remove` (table 4.9a). Prolonged: `Start` and apply the hand-back, `Elapse` as time passes, `Change` when a cure applies.
- **Exports.** Package `toolbox` exports the use cases, commands, results, values, events and pure functions in the same change (FR-095); there is no re-export package. No new test double (`dicetest.FixedRoller` and the `telemetrytest` recorder exist, `shared-types-and-ownership.md` FR-038).

### Result payloads

A test asserts exactly these fields. "events" is an defensively copied `[]ToolboxEvent`.

| Result variant | Fields |
|---|---|
| `ResolveFearEffectSaveRequired`, `ResolveFearRepeatSaveSaveRequired`, `ResolveMentalStressSaveRequired` | `plan` (`SaveGatedRollPlan`: ability, DC, empty circumstances) |
| `ResolveFearEffectResisted` | `saveResult` (`succeeded`); no events |
| `ResolveFearEffectFrightened` | `consequences` (conditions `[frightened]`, no damage, no adjustments, no Exhaustion, `healable`); `active` (`ActiveFear`); `source`; `riders` (`[]FearRiderEffect`); `saveResult` (`failed`); `events` (`[SaveGatedEffectStarted]`) |
| `ResolveFearEffectRejected` | `reason` (`saveDoesNotMatchEffect`, FR-101) |
| `ResolveFearRepeatSaveContinues` | `state` (`ActiveFear`, successes unchanged); `saveResult` (`failed`); `events` (`[SaveGatedEffectRepeatSaveMade(failed)]`) |
| `ResolveFearRepeatSaveEnded` | `state` (`PhaseEnded(successesReached)`); `source`; `conditionToRemove` (`frightened`); `events` (`[SaveGatedEffectRepeatSaveMade(succeeded), SaveGatedEffectEnded]`) |
| `ResolveFearRepeatSaveRejected` | `reason` (`effectNotActive` or `saveDoesNotMatchEffect`, FR-101) |
| `ResolveMentalStressAvoided` | `saveResult` (`succeeded`); no damage; no events |
| `ResolveMentalStressHalved` | `saveResult` (`succeeded`); `damage` (one Psychic, full rolled amount); `adjustments` (`[HalfDamage]`); no events |
| `ResolveMentalStressFull` | `saveResult` (`failed`); `damage` (one Psychic, rolled or shared amount); `adjustments` (empty); `active` (`SaveGatedPhaseEnded(instantaneous)`); `events` (FR-102) |
| `ResolveMentalStressRejected` | `reason` (`saveDoesNotMatchEffect`, `sharedDamageMismatch` or `noPrintedAverage`, FR-046, FR-101, FR-108); nothing rolled; no events |
| `StartProlongedEffectStarted` | `effect` (status `active`); `conditionToApply`; `events` (`[ProlongedEffectStarted]`) |
| `ElapseProlongedEffectContinues` | `effect` (remaining reduced, FR-065); no events |
| `ElapseProlongedEffectEnded` | `conditionToRemove` (the condition for an active short-term effect, else none); `events` (`[ProlongedEffectEnded]`) |
| `ChangeProlongedEffectSuppressed` | `effect` (status `suppressed`); `conditionToRemove`; `events` (`[ProlongedEffectSuppressed]`) |
| `ChangeProlongedEffectResumed` | `effect` (status `active`); `conditionToApply`; `events` (`[ProlongedEffectResumed]`) |
| `ChangeProlongedEffectRemoved` | `conditionToRemove`; `events` (`[ProlongedEffectRemoved]`) |
| `ChangeProlongedEffectRejected` | `reason` (`ProlongedRejection`); nothing changed; no events (telemetry per FR-106) |

## 7. Observability

Hazard's section 7 owns `srd5e.toolbox.operations`, `operation_duration`, `errors`, `saves`, `damage_rolled`, the `toolbox` signal constants and the shared rules (hazard FR-073, FR-078 to FR-081); this spec lists its own six operations and results below and adds their constants to the `toolbox` signal constants itself (FR-110; hazard's tables no longer receive them, Q-16 (c)), plus only the signals below.

1. **What can go wrong?** A dice port error; a host-built bad value (`ErrInvalidInput` error at construction, before any use case runs, so no signal); a wrong-DC or wrong-kind save, a non-fitting shared damage list, a printed amount where the SRD prints none, a repeat save on an ended fear, a double suppression or a resume of an active effect (counted `...Rejected` results); outcome-rate drift from a wrong DC or biased roller. An unlisted cure is not an error (FR-106).
2. **How would an operator know?** Port failure: span error status, `srd5e.toolbox.errors` with `error.type=dice_roller_error`, result `error`. Rejections: `srd5e.outcome=failure`, span error status, a `rejected_...` result, no `error.type`. Failed saves, Frightened targets and `cure_not_effective` are normal outcomes in `srd5e.toolbox.result`.
3. **To diagnose?** Operation, result, `srd5e.toolbox.save_ability` (all six values; save spans and `saves`), `srd5e.toolbox.prolonged_term` (`short_term`, `long_term`, `indefinite`; the three prolonged spans only; a shared key registered here, FR-110) and the rider count (Frightened counter only); DC, totals and damage are in the returned values.
4. **Counted or timed?** Hazard's metrics cover throughput, latency, saves and errors; new here: Psychic damage size and Frightened results by rider count.
5. **Existing instrumentation still correct?** Yes; `srd5e.outcome` keeps hazard's meaning (use-case outcome), not the save's (`srd5e.toolbox.save_result`).
6. **Monitor or alert?** A `rejected_effect_not_active` share above 5% of `resolve_fear_repeat_save`; as a dashboard only, a rising `cure_not_effective` share of `change_prolonged_effect`.
7. **Noisy?** No logs, no child spans, no per-rider or per-die signals.

`srd5e.toolbox.operation` gains `resolve_fear_effect`, `resolve_fear_repeat_save`, `resolve_mental_stress`, `start_prolonged_effect`, `elapse_prolonged_effect` and `change_prolonged_effect` (spans `srd5e.toolbox.<operation>` with `srd5e.outcome` and `srd5e.toolbox.result`). Allowed `srd5e.toolbox.result` values:

| Use case | `srd5e.toolbox.result` values | `srd5e.outcome` |
|---|---|---|
| `resolve_fear_effect` | `save_required`, `resisted`, `frightened` | `success` |
| `resolve_fear_effect` | `rejected_save_does_not_match_effect` | `failure` |
| `resolve_fear_repeat_save` | `save_required`, `continues`, `ended` | `success` |
| `resolve_fear_repeat_save` | `rejected_effect_not_active`, `rejected_save_does_not_match_effect` | `failure` |
| `resolve_mental_stress` | `save_required`, `avoided`, `halved`, `full` | `success` |
| `resolve_mental_stress` | `rejected_save_does_not_match_effect`, `rejected_shared_damage_mismatch`, `rejected_no_printed_average` | `failure` |
| `start_prolonged_effect` | `started` | `success` |
| `elapse_prolonged_effect` | `continues`, `ended` | `success` |
| `change_prolonged_effect` | `suppressed`, `resumed`, `removed`, `cure_not_effective` (FR-106) | `success` |
| `change_prolonged_effect` | `rejected_already_suppressed`, `rejected_not_suppressed` | `failure` |
| `resolve_mental_stress`, `start_prolonged_effect` | `error` (a dice port error left the call, hazard FR-078) | `failure` |

The other four operations roll nothing and have no `error` result. Hazard FR-079 and FR-080 apply: `saves` once per decided save (a forced failure included), never for `save_required`, rejections or port failures (`auto_succeeded` cannot occur, FR-104); `damage_rolled` once per `resolve_mental_stress` call returning damage (shared amount included).

Constants live in the `toolbox` signal-name constants (`toolbox/signals.go`, `doc/rules/observability.md` §5); this table feeds `doc/observability/signals.md`. Each signal carries only the attributes in its row.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.toolbox.fear.frightened` | counter, unit `{creature}` | `srd5e.toolbox.fear.riders` (0 to 3) | Frightened results by riders in play | Dashboard rate by riders; a sudden shift suggests hosts passing the wrong `FearOptions` |
| `srd5e.toolbox.mental_stress.damage` | histogram, unit `{hit_point}` | `srd5e.toolbox.save_ability` | Psychic damage rolled by `ResolveMentalStress`, for `Halved` and `Full` only when this call rolled it (not a shared amount); in addition to hazard's `damage_rolled`, which `resolve_mental_stress` MUST also record (hazard FR-080); a second histogram because it carries `save_ability`, which hazard's `damage_rolled` does not (Q-16 (c): kept; hazard is not asked to add the attribute) | Compare the mean to the expression's expected value; investigate drift over 20 percent |

The prolonged term is an attribute (key registered here, FR-110) of the `start_`, `elapse_` and `change_prolonged_effect` spans, not a signal of its own. No attribute holds an id, free text or an unbounded number. Every `...Rejected` result sets `srd5e.outcome=failure` and span error status and is counted once, not logged, **except** `cureNotEffective` (FR-106); a failed save is a normal outcome (`doc/rules/observability.md` §6, §7). A dice port failure is recorded once where it leaves srd5e: `span.RecordError` (error status) and one `srd5e.toolbox.errors` increment, never a log. The domain emits nothing (FR-084).

## 8. Test plan

Decisions live in the domain and are proved at unit level (`doc/rules/testing.md` §3); application tests keep ports, telemetry and the wiring of the hazard evaluator.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-007, FR-045 | unit | Table-driven over the sample rows (id, DC, dice; no string field); mutating a returned table slice leaves the next call unchanged; boundary DCs 5 and 30; Psychic fixed; Wisdom default; all six abilities accepted; no `MentalStressAbility` exported (exported-name scan). |
| FR-008 to FR-010, FR-013, FR-014, FR-088, FR-098 | unit | `FearEffect.saveGatedEffect` equals the expected `SaveGatedEffect` and passes hazard's validity constructors; hazard evaluator with rolled (equal-to-DC, one-below), forced and not-rolled saves; `source` carried. |
| FR-017, FR-023, FR-052, FR-082 | unit | Inputs unchanged; deterministic. |
| FR-018 to FR-020, FR-022, FR-089 | unit + application | Mapping (ended, continues, rejected, forced failure, `ActiveFear` mismatch) at unit level; application level only for variant wiring and recorded signals. |
| FR-027 to FR-039 | unit | `fearRiderEffects` ordering, table 4.4a, independence of two option values in one process (option equality and copy semantics are hazard FR-103); `TurnUse` invariant and every row of table 4.5a. |
| FR-040 to FR-044, FR-090, FR-099, FR-102 | unit | `MentalStressEffect.saveGatedEffect` and the evaluator with fixed dice; one test per row of table 4.6a; Psychic on every damage; failures assert events `[Started, Ended(instantaneous)]` in order, no conditions, 0 Exhaustion, `SaveGatedPhaseEnded(instantaneous)`; `Halved`, `Avoided`, `SaveRequired` assert no events. |
| FR-047 to FR-049, FR-064 | application | `dicetest.FixedRoller`: not called on avoided, shared or printed (rejected); called once with the exact expression; both dice port failure kinds are returned wrapped once and recorded once (also for `StartProlongedEffect`); totals for the six min/max rows of table 4.6a. |
| FR-015, FR-016, FR-046, FR-100, FR-101, FR-104, FR-108, FR-109 | unit + application | A wrong DC, a wrong kind (rolled, or forced failure of another kind) and a misfit shared list return the exact `...Rejected` variant with nothing rolled and no events; printed mode gives `Rejected(noPrintedAverage)` after save validation; each command reads only its own `ToolboxRuleOptions` field (FR-100); rejection reasons per use case; results `rejected_save_does_not_match_effect`, `rejected_shared_damage_mismatch`, `rejected_no_printed_average`; the three save commands have no `subject` field (exported-name scan); the built subject has empty affinities, no circumstances, `SaveGatedNoExemption`; no `AutoSucceeded` variant reachable; mappings are exhaustive switches. Application level asserts `srd5e.outcome=failure`, span error status, the `rejected_...` result, no `error.type`. |
| FR-011, FR-012, FR-021, FR-050, FR-059, FR-080, FR-096, FR-097 | scenario | Through the public API of the module's packages only: `ResolveMentalStress` with 9 on 9d6 then `ApplyDamage` with `HalfDamage` gives 4; a Frightened round trip with a real `ConditionSet` (`ResolveFearEffect`, host `apply`, `ResolveFearRepeatSave` ending, host `remove`) with the already-held, immune, wrong-source, `ConditionRemovalNotPresent` and `ConditionRemovalImpliedBy` rows of tables 4.9a and 4.9b; a blocked resume keeps the stored suppressed effect; no condition event from the engine. |
| FR-053 to FR-063, FR-094, FR-103, FR-107 | unit | `prolongedDiceRequired` gives `[1d10]`, `[1d10]`, `[]`; `ProlongedEffect.start` with rolled totals: every row of table 4.7a; status `active`; one event; hand-back per short-term condition with the Frightened source, none for long-term and indefinite; scope validity and copy; `rollCircumstanceFor` per ability and status, none (`ok` false) otherwise; `ProlongedEffect` invariants; `target` set by start, kept through every operation and named by every event. |
| FR-065 to FR-071, FR-093 | unit | `ProlongedEffect.elapse`: every row of table 4.7b; a negative elapsed returns an `ErrInvalidInput` error; zero returns an equal effect; suppressed effect ends with no hand-back; indefinite never ends; exact `Duration` arithmetic. |
| FR-072 to FR-079, FR-081, FR-106 | unit + application | `decideProlongedCure` for every cell of table 4.7c; suppress then resume; already-suppressed and not-suppressed rejections; removal of active versus suppressed short-term; not-effective pair rejected whatever the status; application: `cureNotEffective` records `success`, `cure_not_effective`, no error status, no log; the two state errors record `failure`, error status and `rejected_*`. |
| FR-083 to FR-087, FR-092, FR-095, FR-110 | unit + architecture | the `toolbox` signal-name constants gain exactly the FR-110 constants (one file, no second signals file, shared keys reused); Equality, defensively copied collections; `internal/archtest`: no `math/rand`, `time.Now`, `context.Context` or telemetry in the domain files; event fields, past-tense names, closed interface `ProlongedEffectEvent` (exhaustive type switch compiles) embedding `ToolboxEvent`; import-direction and cycle checks, no call to a hazard use case; no `ContentSource` parameter or `MissingContent` variant; exported-name scan: each exported name once in package `toolbox`, none generic, none in the "names deliberately avoided" list, the four functions and the query once each. |

- **Telemetry.** Application tests assert the signals of section 7 with the recording doubles: operation, result and attributes per the tables; `srd5e.outcome=failure` with error status for each `...Rejected` result except `cureNotEffective`; `saves` once per decided save and never for `save_required`, rejections or a roller failure; `damage_rolled` once per call returning damage; a roller failure records one `errors` increment, result `error`, no log; each use case behaves identically with the no-op and a recording `telemetrytest` bundle.
- **Doubles.** `dicetest.FixedRoller` and the recording observability double; no new port, so no new double or contract suite.
- **Mutation testing.** `gomu` on package `toolbox` (the fear and stress files) at threshold 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (command in `doc/rules/testing.md` §8); the `*_usecase.go` files SHOULD be. Hotspots: table 4.5a boundaries, `remaining <= elapsed`, `10 x n` and the cure matrix cells.
- **Deliberately not covered.** The spell text of the three cures; the SRD's GM advice.

## 9. Open questions and decisions

Everything below is closed. Settled, one line each:

- **Context and options.** `toolbox` context and options bundle (`shared-types-and-ownership.md` FR-045; `toolbox-hazard-effects.md` FR-099 to FR-101); conditions are handed back, never applied (hazard Q-5); the option types are hazard's and this spec imports upward (Q-19, hazard Q-20); in-context order in the header (Q-18).
- **Tables and scope.** Sample tables are read-only in the domain with no SRD prose or display string (Q-4: numbers and dice only, precedent `content-monsters.md`'s XP-by-CR table); riders judged independently (Q-7, FR-038); scope by `Ability`, a skill scope can be added later (Q-12); a cornered creature is the host's ruling (Q-13); the repeat-save cadence is the host's (Q-14); Psychic fixed and ability unnarrowed (Q-15, FR-006, FR-045).
- **Q-6, source of fear.** `CombatantID`, as `conditions.md` D-2 settled (a host with a non-creature source such as the Abyss portal or the trap on p. 196 assigns it an id); no `FearSource` type (design basis: `doc/rules/architecture.md` §3, minimal surface). FR-010, FR-021 stand.
- **Q-9, already held or immune.** The effect does not start: "the character has the ... condition for 1d10 minutes" (p. 196) cannot hold for a condition the creature cannot gain, and a second timer has no SRD basis. The engine never sees the `ConditionSet`, so the host discards the returned effect (FR-059, table 4.9b). Package design.
- **Q-10, suppression versus the clock.** The clock keeps running while suppressed (FR-070): SRD 5.2.1, "Antimagic Field", p. 109, says "the time it spends suppressed counts against its duration", the only SRD rule on suppression, and p. 196 pauses nothing (interpretation: the same word, the same meaning). A suppressed short-term effect hands back its condition for removal and, on `ProlongedResume`, for re-application (FR-077, FR-078). The host calls `ProlongedResume` when the suppressing cast ends (for Calm Emotions, p. 114, the end of the spell, during which the creature also has Immunity to Frightened); a resume after the effect expired in the meantime never happens because `ElapseProlongedEffect` already ended it (FR-070).
- **Q-11, trigger of a prolonged effect.** The host decides: p. 196 says only "Exposure to mental stress can cause prolonged effects" and names no save, so `StartProlongedEffect` starts one on command (section 5.1). The host may run `ResolveMentalStress` first.
- **Q-8, cures.** Table 4.7c models only the pairs p. 196 states; every other pair is `cureNotEffective` (FR-075, FR-076). Interpretations: an indefinite effect "lasts until removed by a Greater Restoration spell" (so Lesser Restoration does not remove it, though it is "a long-term effect"), and p. 196 names no cure but Greater Restoration for it and none but Lesser for the others. The Greater Restoration spell text (p. 137) lists neither Frightened nor Disadvantage on checks, so the SRD gives no basis to widen the matrix. Suppression has no "usually", so no exception clause (FR-072). Mapping a cast spell to `ProlongedCure` stays the host's task; a `SpellID` mapping is added only when a consumer needs it.
- **Q-17, cure vocabularies.** `ProlongedCure` stays separate from `ContagionCuringMagic`: different rules for the same spell (p. 196 versus p. 194), and the host maps one cast to whichever applies. A shared enum is hoisted by amendment only if a third spec needs one.
- **Q-16, hazard amendments.** (a) and (b) done (hazard FR-018, FR-089, FR-086). (c) settled by hazard Q-15: this spec's section 7 owns the six operations, their results and the shared key `srd5e.toolbox.prolonged_term`, and adds the the `toolbox` signal constants constants (FR-110); hazard section 7 receives none of them; `cure_not_effective` is `success` (FR-106); the `mental_stress.damage` histogram stays here. Hazard FR-008 now holds a `SaveGatedRequirement`, so FR-008 and FR-040 build `SaveGatedSaveRequired`. (d) `ProlongedEffect` stays whatever hazard adds (section 5.1). (e) hazard Q-5 superseded (section 1). (f) owed: hazard FR-062 adds "a forced failure on a repeat save counts as a failed repeat save" (FR-089 interprets until then). This spec edits only its own file.

## 10. Changelog impact

Under `Unreleased` > **Added**: the `toolbox` "Fear and Mental Stress" use cases (`ResolveFearEffect`, `ResolveFearRepeatSave`, `ResolveMentalStress`, `StartProlongedEffect`, `ElapseProlongedEffect`, `ChangeProlongedEffect`), the rider and half-damage behaviour for the `ToolboxRuleOptions` fields, the sample tables, the pure functions `fearRiderEffects`, `checkFrightenedTurn`, `decideProlongedCure` and `prolongedDiceRequired`, the five prolonged-effect events (closed interface family `ProlongedEffectEvent`) and the toolbox signals of section 7. Not **BREAKING**: everything is new and no existing export, result or signal changes.
