# Readying a spell

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `initiative-and-actions.md`, `content-schema-and-decoder.md`, `content-spells.md` |

Split from `magical-effects.md` (its FR-130 to FR-136, FR-138 to FR-142, withdrawn
there in favour of FR-001 to FR-012 here). Rank 26 (`shared-types-and-ownership.md`
4.11): right of `initiative-and-actions` (25), whose `ReadiedAction`, `ReadyTriggerID`,
`ReactionTaken` and `ReadiedActionExpired` it imports, with `CombatantID` and
`CombatEvent` (`combat-foundations.md`), `SpellID` (`content-schema-and-decoder.md`)
and `CastingTime` (`content-spells.md`). It has no port and returns no
`*srderr.PortError`, so `errors-and-results.md` and `observability-ports.md` are not
dependencies; `spellcasting.md` (Concentration) is prose. Under decisions FR-048
this spec owns the casting-time check and the events; Concentration stays
`spellcasting.md`'s. `combat-action-resolution.md` will consume the three use cases of
section 7 (an amendment not made here).

## 1. Purpose and scope

The spell half of the Rules Glossary "Ready [Action]" (SRD 5.2.1, pp. 186-187).
The Ready action is `initiative-and-actions.md`: the readied action
(`ReadiedAction`, FR-056), trigger match (FR-072, `NoMatchingReadiedAction`), expiry
(FR-038, `ReadiedActionExpired`) and the Reaction (FR-073, `ReactionTaken`). `rules-glossary-coverage.md` Q-6 (settled) gives GL-117b to this spec, FR-001 to FR-012.

**RA-1 (Concentration, change request to `spellcasting.md`, not made here).** A
held spell has a `SpellID`, not a `FeatureID`: the owner is asked for a
Concentration subject that records it, or to say the host uses the `FeatureID`
subject. Until then FR-002 to FR-012 state only the request (`ConcentrationHold`).
**RA-2 (change request to `initiative-and-actions.md`, not made here).** A
`ReadiedAction` gains a closed `HeldSpellInput` interface field (`HoldsSpell{ReadiedSpell}`
or `HoldsNoSpell`, no null); until then the host holds it beside the
`ReadiedAction` and both operations take it as an argument.

**Out of scope:** casting, slot expenditure and Concentration state
(`spellcasting.md`); taking the Ready action and spending the Reaction
(`initiative-and-actions.md`); a monster's Concentration.

## 2. Source

- SRD 5.2.1, "Ready [Action]", pp. 186-187; "Magic [Action]", p. 185;
  "Concentration", p. 179; "No Concentration", p. 184.
- Rule files: `doc/rules/architecture.md` §3, §6; `doc/rules/testing.md` §4, §5;
  `doc/rules/observability.md` §2, §3.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| ready a spell | `ReadiedSpell{Spell, Caster, Trigger}`, `HeldSpellInput` (`HoldsSpell`, `HoldsNoSpell`), `ConcentrationHold{Spell SpellID}`, `ReleaseHoldRequest{Spell SpellID}`, `DissipationCause` (`ConcentrationBroken`, `ReadyElapsed`) | The Ready action is `initiative-and-actions.md`'s. |
| casting time | `CastingTime` from `content-spells.md` FR-026 | Reused. |

## 4. Requirements

Ids are stable.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `ReadiedSpell{Spell SpellID, Caster CombatantID, Trigger ReadyTriggerID}` and the closed `HeldSpellInput` interface (`HoldsSpell{ReadiedSpell}`, `HoldsNoSpell`) MUST be the only data this spec adds to a readied action; the trigger match, expiry and Reaction are `initiative-and-actions.md` FR-072, FR-038 and FR-073 and are not redefined here. No operation takes a null spell. | SRD 5.2.1, "Ready [Action]", p. 187 | happy |
| FR-002 | `ReadySpell(caster, trigger, spell, castingTime)` with a spell whose casting time is `MagicAction` MUST return `ReadySpellApplied` carrying a `ReadiedSpell` and a `ConcentrationHold{Spell SpellID}`, with one `SpellHeld` event. `ConcentrationHold` is the request the caller passes to `spellcasting.md` to begin the held-spell Concentration. | SRD 5.2.1, "Ready [Action]" ("a spell must have a casting time of an action"), p. 187; "Magic [Action]", p. 185 | happy |
| FR-003 | `ReadySpell` with `BonusAction`, `Reaction`, `Minutes(n)` or `Hours(n)` MUST return `ReadySpellNotAnAction` carrying that `CastingTime`, and no events (one test per casting time, `Minutes(1)` and `Hours(1)` included). | SRD 5.2.1, "Ready [Action]", p. 187 | unhappy (state error) |
| FR-004 | `ReadySpellApplied` MUST carry a `ConcentrationHold` even for a spell that does not itself require Concentration ("holding on to the spell's magic requires Concentration"). | SRD 5.2.1, "Ready [Action]", p. 187 | unhappy (rule interaction) |
| FR-005 | The events of a `ReadySpell` call MUST be exactly `[SpellHeld]`: no `SpellCast` and no slot-spending event. The cast and its resources are `spellcasting.md`'s, done before this call ("expending any resources used to cast it"). | SRD 5.2.1, "Ready [Action]", p. 187 | unhappy (rule interaction) |
| FR-006 | `ReadySpell` MUST NOT end any Concentration itself. Beginning the held-spell Concentration while another is in progress ends the other with the reason `AnotherEffect` (`casting-and-concentration.md` FR-119, FR-120; interpretation, settled Q-1). | SRD 5.2.1, "Concentration" ("Another Concentration Effect"), p. 179; "Ready [Action]", p. 187 | unhappy (rule interaction) |
| FR-007 | `ReleaseHeldSpell(held HeldSpellInput, reaction ReactionTaken)` MUST check in this order: (1) a `reaction` that is not the `ReactionTaken` of a `ReadiedActionReaction` returns `ReleaseHeldSpellNotAReadiedReaction` (FR-009); (2) `HoldsNoSpell` returns `ReleaseHeldSpellNoHeldSpell` (FR-008); (3) a `HoldsSpell(spell)` whose `Spell.Caster` differs from `reaction.Combatant`, or whose `Spell.Trigger` differs from the reaction's `ReadyTriggerID`, returns `ReleaseHeldSpellReactionMismatch` and no events; (4) otherwise it MUST return `ReleaseHeldSpellApplied` with one `HeldSpellReleased` event and a `ReleaseHoldRequest{Spell SpellID}`. The caller ends the hold with `BreakConcentration` (`Ended`, `casting-and-concentration.md` FR-077) and, if the spell requires Concentration, begins it as `casting-and-concentration.md` FR-075 describes. A matching reaction means equal caster and trigger ids (`initiative-and-actions.md` FR-072). | SRD 5.2.1, "Ready [Action]" ("which you release with your Reaction when the trigger occurs"), p. 187; "Concentration", p. 179 | happy |
| FR-008 | `ReleaseHeldSpell` given `HoldsNoSpell` and a readied-action `ReactionTaken` MUST return `ReleaseHeldSpellNoHeldSpell` and no events. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-009 | A held spell MUST be released only with the caster's Reaction: `ReleaseHeldSpell` takes the `ReactionTaken` event as a required parameter. A `ReactionTaken` of another kind (an Opportunity Attack, a `FeatureReaction`) MUST return `ReleaseHeldSpellNotAReadiedReaction` and no events. This spec spends no Reaction itself. | SRD 5.2.1, "Ready [Action]" ("which you release with your Reaction when the trigger occurs"), p. 187 | unhappy (rule interaction) |
| FR-010 | `DissipateHeldSpell(held HeldSpellInput, cause DissipationCause)` with `HoldsSpell` and cause `ConcentrationBroken` (the host saw `spellcasting.md`'s `ConcentrationEnded` for the hold) MUST return `DissipateHeldSpellApplied` with one `HeldSpellDissipated` event: the spell takes no effect and no resource is refunded. | SRD 5.2.1, "Ready [Action]" ("If your Concentration is broken, the spell dissipates without taking effect"), p. 187 | unhappy (state error) |
| FR-011 | `DissipateHeldSpell` with `HoldsSpell` and cause `ReadyElapsed` (the host saw `ReadiedActionExpired`, `initiative-and-actions.md` FR-038, for a readied action holding a spell) MUST behave as FR-010; the caller ends the hold with `BreakConcentration` (`Ended`). `HoldsNoSpell` with either cause MUST return `DissipateHeldSpellNoHeldSpell` and no events. | SRD 5.2.1, "Ready [Action]" ("maintain up to the start of your next turn"), p. 187; `initiative-and-actions.md` FR-038 | unhappy (state error) |
| FR-012 | A second held spell MUST be handled thus: `initiative-and-actions.md` FR-056 refuses a second Ready (`ActionAlreadyTaken`) and FR-038 expires the first at the start of the caster's next turn, so two holds overlap only if the host skipped FR-011; then the first hold's Concentration ends with the reason `AnotherEffect` (`casting-and-concentration.md` FR-119) and the host calls `DissipateHeldSpell` (`ConcentrationBroken`) for the first. | SRD 5.2.1, "Concentration", p. 179; "Ready [Action]", p. 187 | unhappy (rule interaction) |
| FR-013 | Layering, events and telemetry follow `combat-foundations.md` FR-015 to FR-019, FR-026 and FR-029 to FR-035: types live in package `combat`, are pure and import only the kernel packages and the published API of upstream contexts (`content`: `SpellID`, `CastingTime`); no import of `character`'s concentration rules. Events form a closed `HeldSpellEvent` interface (unexported marker method; variants `SpellHeld`, `HeldSpellReleased`, `HeldSpellDissipated`) whose variants also satisfy `CombatEvent`: `SpellHeld` (`spell_held`), `HeldSpellReleased` (`held_spell_released`), `HeldSpellDissipated` (`held_spell_dissipated`), each with `Combatants()` the caster; the use cases are `ready_combatant_spell`, `release_combatant_spell`, `dissipate_combatant_spell`. | `doc/rules/architecture.md` §1, §2, §9; `shared-types-and-ownership.md` FR-019, FR-044 | unhappy (forbidden edge) |

## 5. Model

- **Referenced, not defined here:** `CombatantID`, `CombatEvent`; `ReadiedAction`,
  `ReadyTriggerID`, `ReactionTaken`, `ReadiedActionExpired`
  (`initiative-and-actions.md`); `SpellID` (`content-schema-and-decoder.md`);
  `CastingTime` (`content-spells.md`). Named in prose, not imported:
  `Concentration`, `BreakConcentration`, `ConcentrationEnded` (`spellcasting.md`).
- **Value objects:** the types of section 3. **Pure functions** (no service type, no state):
  `ReadySpell`, `ReleaseHeldSpell`, `DissipateHeldSpell`. **Failures:** closed result interfaces with
  the variants of section 6 (an exhaustive type switch); no error return and no panic: every refusal is a variant.

## 6. Interface

Signatures (package `combat`, pure, no `context.Context`, no error return):

```go
func ReadySpell(caster CombatantID, trigger ReadyTriggerID, spell SpellID, castingTime CastingTime) ReadySpellResult
func ReleaseHeldSpell(held HeldSpellInput, reaction ReactionTaken) ReleaseHeldSpellResult
func DissipateHeldSpell(held HeldSpellInput, cause DissipationCause) DissipateHeldSpellResult
```

Result interfaces are closed (unexported marker method):

- `ReadySpellResult`: `ReadySpellApplied{Spell ReadiedSpell, Hold ConcentrationHold, Events []HeldSpellEvent}`,
  `ReadySpellNotAnAction{CastingTime CastingTime}`.
- `ReleaseHeldSpellResult`: `ReleaseHeldSpellApplied{Release ReleaseHoldRequest, Events []HeldSpellEvent}`,
  `ReleaseHeldSpellNoHeldSpell`, `ReleaseHeldSpellNotAReadiedReaction`,
  `ReleaseHeldSpellReactionMismatch`.
- `DissipateHeldSpellResult`: `DissipateHeldSpellApplied{Events []HeldSpellEvent}`,
  `DissipateHeldSpellNoHeldSpell`.

Refusal variants carry no events.

- No use case, port or async: synchronous pure functions over host-held values.
- Exported from package `combat` only (no facade, no re-export). The
  architecture scan finds no Concentration name defined here. No test doubles.

## 7. Observability

Silent by design (`doc/rules/observability.md` §3); no signal of its own. The
use cases of the `combat-action-resolution.md` amendment record through `CombatTelemetry` (`combat-foundations.md` FR-029 to FR-035).

| Operation | Variant | `srd5e.combat.result` | Class |
|---|---|---|---|
| `ready_combatant_spell` | `ReadySpellApplied` | `completed` | applied |
| | `ReadySpellNotAnAction` | `ready_not_an_action` | refused |
| `release_combatant_spell` | `ReleaseHeldSpellApplied` | `completed` | applied |
| | `ReleaseHeldSpellNoHeldSpell` | `release_no_held_spell` | refused |
| | `ReleaseHeldSpellNotAReadiedReaction` | `release_not_a_readied_reaction` | refused |
| | `ReleaseHeldSpellReactionMismatch` | `release_reaction_mismatch` | refused |
| `dissipate_combatant_spell` | `DissipateHeldSpellApplied` | `completed` | applied |
| | `DissipateHeldSpellNoHeldSpell` | `dissipate_no_held_spell` | refused |

Seven questions: (1) the host calls in the wrong state or passes a wrong casting
time (`refused`). (2) By `srd5e.combat.operations` once those use cases land. (3) Results
carry the casting time that failed. (4) Each call by operation and result.
(5) Concentration signals keep their meaning: this spec defines no Concentration
state. (6) Sustained `release_no_held_spell` and `dissipate_no_held_spell` are
host state bugs. (7) Nothing is noisy: no log.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-006 | unit | One test per casting time; a spell with no Concentration still yields a hold; events are exactly `[SpellHeld]`; no `ConcentrationEnded` in the result |
| FR-007 to FR-012 | unit | One test per outcome, including the check order of FR-007 and the `HoldsNoSpell` input |
| FR-013 | architecture | Import direction; no duplicate or generic variant name |

No doubles or contract suites. Mutation testing (`doc/rules/testing.md` §8): all
changed lines of the `combat` readying files.

## 9. Open questions and decisions

**Settled** (one line each): Q-1 holding a readied spell ends Concentration on another effect, reason `AnotherEffect` (FR-006): the SRD says holding the spell "requires Concentration" (p. 187) and Concentration allows one effect at a time (p. 179), so beginning the hold is "Another Concentration Effect"; interpretation, matching community consensus and `casting-and-concentration.md` FR-119 and FR-120; no new `ConcentrationBreakReason`. RA-1 (held-spell Concentration subject) and RA-2 (`HeldSpellInput` field) stay change requests to their owners with the host-held fallbacks of section 1; neither blocks this spec.

## 10. Changelog impact

- **Added:** readying a spell (the spell half of "Ready [Action]") in `combat`
  (`doc/specs/readying-a-spell.md`).
- **BREAKING:** no. Everything is new.
