# Monster limited use and Legendary Actions

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `observability-ports.md`, `content-monsters.md`, `rests-and-hit-dice.md`, `combat-foundations.md`, `conditions.md`, `initiative-and-actions.md` |

Types ship with the `combat` phase (`shared-types-and-ownership.md` FR-014; see [../roadmap.md](../roadmap.md)).

## 1. Purpose and scope

Parts of a monster's stat block can be restricted in how often they are used:
`X/Day`, `Recharge X-Y` and `Recharge after a Short or Long Rest`
(SRD 5.2.1, "Limited Usage", p. 257). A monster with Legendary Actions can take
one only immediately after another creature's turn, from a limited number of
uses that come back at the start of its own turn ("Legendary Actions", p. 257).
No other spec owns these rules (`shared-types-and-ownership.md` FR-060).
`content-monsters.md` holds the printed notation as data (`PartUsage`,
`LegendaryActions`); `initiative-and-actions.md` leaves Legendary Action timing
to this spec.

The spec is a set of pure state-in, state-out rules. One value object,
`LimitedUseState`, holds what changes for one monster combatant: uses left of
each `X/Day` part, whether each `Recharge` part is ready, and the Legendary
Action uses left, and uses left of each `N/Day` spell of a Spellcasting trait.
Each operation returns a new state and events. One use case,
`ApplyMonsterTurnStart`, exists only because rolling the Recharge die needs the
`dice.Roller` port (`shared-types-and-ownership.md` FR-044, FR-023). The host stores
the state (Q-5).

**Out of scope:**

- **Lair actions.** SRD 5.2.1 has none: no Lair Action rule, no Initiative count
  20 action, no regional effect. The only lair content is the "in Lair" numbers
  (`Legendary Resistance (3/Day, or 4/Day in Lair)`, `Legendary Action Uses: 3
  (4 in Lair)`, pp. 256-258; `content-monsters.md` FR-032, FR-035, FR-040). Left
  out per `AGENTS.Md` and flagged (Q-3). Only the `LairStatus` input is modelled.
- **"Running a Monster"** (p. 255): advice to the Game Master, not a rule. No
  requirement; recorded in `rules-glossary-coverage.md` (adjunct row GL-A10).
- **What a used part does** (damage, a Legendary Resistance's effect, a
  Legendary Action's text, riders such as "can't take this action again until the
  start of its next turn", "Solar", p. 327): stat-block text resolved by the
  host and the attack, saving throw and condition specs. This spec counts uses.
- **Spell-list frequencies:** `N/Day` and `N/Day Each` uses are counted (section
  4.9, Q-4); `At Will` has no limit, so no entry. Which spell a monster casts and
  its effect: the cast rules are owned by `monster-spellcasting.md` (ownership rule:
  `shared-types-and-ownership.md` FR-048) and the spell specs. Per-creature setup is
  `monster-combatant-setup.md`, a sibling; neither depends on the other.
- **Turn order and the Action economy:** `initiative-and-actions.md`. **Rest
  rules:** `rests-and-hit-dice.md`. **Where a lair is:** the host reports
  `LairStatus`; the engine has no map.

## 2. Source

- SRD 5.2.1, "Monsters", "Limited Usage", p. 257; "Legendary Actions", p. 257;
  "Running a Monster", p. 255.
- "Aboleth", p. 258 (`Legendary Action Uses: 3 (4 in Lair)`, `Legendary Resistance
  (3/Day, or 4/Day in Lair)`, `Dominate Mind (2/Day)`); "Air Elemental", pp.
  258-259 (Whirlwind, `Recharge 4-6`); "Ankheg", p. 259 (Acid Spray, `Recharge
  6`); "Cloaker", p. 274 (`Recharge after a Short or Long Rest`); "Solar", p.
  327 (`Legendary Action Uses: 3`).
- Rules Glossary, "Incapacitated", p. 184; "Short Rest", p. 187; "Long Rest", p. 185.
- Package design: `doc/rules/architecture.md` §3, §4, §6; `doc/rules/observability.md`
  §2, §3, §5; `doc/rules/testing.md` §4, §5, §8; `combat-foundations.md`.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| X/Day | `PartPerDay` (`content-monsters.md` FR-035), tracked as an entry of `LimitedUseState` | Long Rest regains uses (p. 257). |
| Recharge X-Y | `PartRecharge(low, high)` (`content-monsters.md` FR-035), `RechargeReadiness` (`ready`, `spent`) | 1d6 at the start of each of the monster's turns; regain on a roll in the range (p. 257). |
| Recharge after a Short or Long Rest | `PartRechargeAfterRest` | Once, then until a Short or Long Rest (p. 257). |
| Lair | `LairStatus` (`inLair`, `outsideLair`) | Selects "in Lair" numbers. Host input. |
| Legendary Actions, Legendary Action Uses | `LegendaryActions` (`content-monsters.md` FR-040), `LegendaryTrack` (closed: `LegendaryNone`, `LegendaryTracked(maximum, remaining)`) | p. 257. |
| "immediately after another creature's turn" | `LegendaryActionWindow(endedTurnOf, round)` | Package term; built from `TrackerTurnEnded` (`initiative-and-actions.md` FR-028). |
| Incapacitated | `ConditionActionAvailability` (`conditions.md` FR-064) | `none` means unable to take actions (p. 184). |
| N/Day, N/Day Each, At Will (Spellcasting groups) | `SpellFrequencyPerDay`, `SpellFrequencyPerDayEach`, `SpellFrequencyAtWill` (`content-monsters.md` FR-045), tracked as `SpellUseEntry` | Long Rest regains uses (p. 257). |
| Short Rest, Long Rest | `RestKind` (`rests-and-hit-dice.md` FR-001) | Reused. |
| (a limited-use subject) | `LimitedUseSubject` (closed: `PartSubject(LimitedUsePartRef)`, `SpellSubject(SpellUseRef)`) | Package term: what an event names. |
| (a stat-block part) | `LimitedUsePartRef(section, index)`, `MonsterPartSection` (`traits`, `actions`, `bonusActions`, `reactions`, `legendaryActions`) | Package term: section and printed position (`content-monsters.md` FR-033). |

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. "Printed order" is
`MonsterPartSection` declaration order, then position in the section's list.

### 4.1 Starting state

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `LimitedUseState.Start(Monster monster, {required CombatantID combatant, required LairStatus lair})` MUST hold one entry for each part whose `PartUsage` is `PartPerDay`, `PartRecharge` or `PartRechargeAfterRest`, in printed order, and no entry for `PartUnlimited` or `PartUsageOther` (usage the SRD gives no rule for, such as "Requires Soul Bag"). Spell entries follow (FR-043). | SRD 5.2.1, "Limited Usage", p. 257 | happy |
| FR-002 | A `PartPerDay(count, inLairCount?)` entry MUST start with `count` uses remaining when `lair` is `outsideLair`, and with `inLairCount` when `lair` is `inLair` and the part has one; with no `inLairCount` it keeps `count` in the lair. Legendary Resistance (3/Day, or 4/Day in Lair) is 3 outside and 4 in the lair; Dominate Mind (2/Day) is 2 in both. | SRD 5.2.1, "Aboleth", p. 258; "Limited Usage", p. 257 | happy |
| FR-003 | A `PartRecharge` or `PartRechargeAfterRest` entry MUST start `ready` ("can use the stat block part once"). | SRD 5.2.1, "Limited Usage", p. 257 | happy |
| FR-004 | A monster with `LegendaryActions` MUST start with `LegendaryTracked(maximum, remaining: maximum)`, `maximum` being `usesInLair` when `lair` is `inLair` and the monster has one, else `uses`; a monster without them MUST have `LegendaryNone`. The Aboleth is 3 outside and 4 in the lair; the Solar is 3 in both. | SRD 5.2.1, "Legendary Actions", p. 257; "Aboleth", p. 258; "Solar", p. 327 | happy |
| FR-005 | `LimitedUsePartRef` MUST return an error matching `srderr.ErrInvalidInput` for a negative `index`. Two refs with equal section and index MUST be equal. | architecture.md §6 | unhappy (invalid input) |

### 4.2 X/Day

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-006 | `spend(part)` on a `PartPerDay` entry with at least 1 use remaining MUST return `LimitedUseSpent` with the entry reduced by exactly 1 and one `LimitedUsePartSpent` event carrying the uses remaining. | SRD 5.2.1, "Limited Usage" ("X/Day"), p. 257 | happy |
| FR-007 | `spend(part)` with 0 uses remaining MUST return `LimitedUseSpendRefused(exhausted)`, with the state unchanged and no event. A `1/Day` part is usable exactly once until a Long Rest (FR-026). | SRD 5.2.1, "Limited Usage" ("a Reaction that includes “1/Day” means the Reaction can be taken once"), p. 257 | unhappy (boundary) |
| FR-008 | A Short Rest MUST NOT restore uses of an `X/Day` part; only a Long Rest does (FR-026). | SRD 5.2.1, "Limited Usage" ("must finish a Long Rest to regain expended uses"), p. 257 | unhappy (rule interaction) |

### 4.3 Recharge

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-009 | `spend(part)` on a `ready` `PartRecharge` or `PartRechargeAfterRest` entry MUST return `LimitedUseSpent` with the entry `spent` and one `LimitedUsePartSpent` event (uses remaining 0); on a `spent` entry it MUST return `LimitedUseSpendRefused(exhausted)`. | SRD 5.2.1, "Limited Usage" ("can use the stat block part once"), p. 257 | happy |
| FR-010 | `RechargeRollsNeeded` MUST list, in printed order, exactly the `spent` `PartRecharge` entries: one roll per part. A `ready` part, a `PartRechargeAfterRest` part and every `X/Day` part need no roll; a state with no spent `PartRecharge` needs none (empty list). | SRD 5.2.1, "Limited Usage" ("At the start of each of the monster's turns, roll 1d6"), p. 257 | unhappy (empty and minimal) |
| FR-011 | A spent `PartRecharge(low, high)` MUST become `ready` for a 1d6 face `f` with `low <= f <= high` (both ends inclusive) and stay `spent` otherwise. "Recharge 5-6" regains on 5 and 6 and not on 4; "Recharge 4-6" on 4; "Recharge 6" only on 6. All rows of table RC-1 MUST hold. | SRD 5.2.1, "Limited Usage" ("Recharge 5-6" example), p. 257; "Air Elemental", pp. 258-259; "Ankheg", p. 259 | unhappy (boundary) |
| FR-012 | `StartOfOwnTurn(Map<LimitedUsePartRef, DieRoll> rolls)` MUST return an error matching `srderr.ErrInvalidInput` unless its keys are exactly `RechargeRollsNeeded` and every `DieRoll` is of a d6 (`Die.D6`). A caller error, not content: the use case builds the map (FR-036). | architecture.md §6; SRD 5.2.1, "Limited Usage" (1d6), p. 257 | unhappy (invalid input) |
| FR-013 | A spent `PartRechargeAfterRest` MUST NOT be rolled for and MUST become `ready` only when the monster finishes a Short or Long Rest (FR-025, FR-026). | SRD 5.2.1, "Limited Usage" ("Recharge after a Short or Long Rest"), p. 257; "Cloaker", p. 274 | unhappy (rule interaction) |
| FR-014 | A spent `PartRecharge` MUST also become `ready` when the monster finishes a Short or Long Rest, with no roll. | SRD 5.2.1, "Limited Usage" ("also recharges when the monster finishes a Short or Long Rest"), p. 257 | unhappy (rule interaction) |

### 4.4 Legendary Actions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-015 | `TakeLegendaryAction(LegendaryActionWindow window, ConditionActionAvailability availability)` MUST, when no refusal of FR-021 applies, return `LegendaryActionAllowed` with `remaining` reduced by exactly 1, the window recorded as used (FR-018) and one `LegendaryActionTaken` event. Every Legendary Action costs one use: SRD 5.2.1 stat blocks print no other cost. | SRD 5.2.1, "Legendary Actions" ("expends one use whenever it takes a Legendary Action"), p. 257; "Aboleth", p. 258 | happy |
| FR-016 | A window whose `endedTurnOf` equals the monster's own `combatant` MUST be refused with `ownTurn`: a Legendary Action is taken "only after another creature's turn ends". `round` MUST be at least 1, else an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Legendary Actions", p. 257 | unhappy (rule interaction) |
| FR-017 | `availability` of `none` (the Incapacitated condition, or the host's report that the monster is otherwise unable to take actions) MUST be refused with `incapacitated`. | SRD 5.2.1, "Legendary Actions" ("can't take a Legendary Action if it has the Incapacitated condition or is otherwise unable to take actions"), p. 257; "Incapacitated", p. 184 | unhappy (state error) |
| FR-018 | At most one Legendary Action MUST be taken in one window: after `LegendaryActionAllowed` for a window, a second request for an equal window (same `endedTurnOf` and `round`) MUST be refused with `windowAlreadyUsed`, and a request for a different window MUST be judged on its own. The state records the last used window as a closed `LegendaryWindowRecord` (`LegendaryWindowNoneTaken`, `LegendaryWindowTaken(window)`). Package reading of "Only one of these actions can be taken at a time" (Q-1). | SRD 5.2.1, "Legendary Actions" ("Only one of these actions can be taken at a time"), p. 257 | unhappy (rule interaction) |
| FR-019 | With 0 uses remaining a request MUST be refused with `noUsesRemaining`. | SRD 5.2.1, "Legendary Actions" ("limited number of Legendary Action uses"), p. 257 | unhappy (boundary) |
| FR-020 | A monster with `LegendaryNone` MUST be refused with `noLegendaryActions`. | SRD 5.2.1, "Legendary Actions" ("If the monster has Legendary Action options"), p. 257 | unhappy (empty and minimal) |
| FR-021 | When several refusals apply, the order MUST be `noLegendaryActions`, `ownTurn`, `incapacitated`, `windowAlreadyUsed`, `noUsesRemaining`; the first is returned, the state unchanged, no event. Package design, so a result does not depend on evaluation order. | package design; SRD 5.2.1, p. 257 | unhappy (rule interaction) |
| FR-022 | `StartOfOwnTurn` MUST set `remaining` to `maximum` ("regains all expended uses at the start of each of its turns") and, when `remaining` was below `maximum`, add one `LegendaryActionUsesRegained` event with the number regained. At `maximum` no event is added. | SRD 5.2.1, "Legendary Actions", p. 257 | happy |
| FR-023 | A Short Rest and a Long Rest MUST leave the Legendary Action uses unchanged: the SRD restores them only at the start of the monster's turn (Q-6). | SRD 5.2.1, "Legendary Actions", p. 257 | unhappy (rule interaction) |

### 4.5 Rests, lair and turn start

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-024 | `StartOfOwnTurn(rolls)` MUST apply FR-011 to every roll and FR-022, returning a `LimitedUseTransition`. Its events MUST be in this order: one `MonsterRechargeRolled` per roll in printed order (the face, the part and a `RechargeRollResult` of `regained` or `notRegained`), then `LegendaryActionUsesRegained` when FR-022 adds it. With no rolls and nothing to regain it MUST return an equal state and no events. | SRD 5.2.1, "Limited Usage", "Legendary Actions", p. 257 | happy |
| FR-025 | `AfterRest(RestKind.ShortRest)` MUST make every spent `PartRecharge` and `PartRechargeAfterRest` entry `ready` and change nothing else (FR-008, FR-023). | SRD 5.2.1, "Limited Usage", p. 257 | happy |
| FR-026 | `AfterRest(RestKind.LongRest)` MUST do what FR-025 does and also set every `X/Day` entry to its maximum for the current `LairStatus` (FR-002) and every spell entry to its maximum (FR-047). | SRD 5.2.1, "Limited Usage" ("must finish a Long Rest to regain expended uses"), p. 257 | happy |
| FR-027 | `AfterRest` MUST add one `LimitedUsesRestored` event listing, in printed order (parts, then spells), exactly the `LimitedUseSubject`s that changed, and none when nothing changed. | SRD 5.2.1, "Limited Usage", p. 257 | unhappy (empty and minimal) |
| FR-028 | `ChangeLair(LairStatus status)` MUST recompute each `X/Day` maximum and the Legendary maximum for `status` (FR-002, FR-004) and keep the number of uses already spent: `remaining` becomes `max(0, newMaximum - spent)`, where `spent = oldMaximum - remaining`. It MUST add one `MonsterLairStatusChanged` event when the status differs and return an equal state with no event when it does not. Package design (Q-2). | SRD 5.2.1, "Aboleth", p. 258 ("3 (4 in Lair)") | unhappy (rule interaction) |

### 4.6 Queries, values and events

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-029 | `AvailabilityOf(part)` MUST return `LimitedUseAvailability.Available` for a ready or not-yet-exhausted entry, `exhausted` for a spent or zero-use entry, and `notLimited` for a ref that has no entry (an unlimited part, a `PartUsageOther` part, or a position not in the stat block). It MUST NOT return an error for a valid ref. | SRD 5.2.1, "Limited Usage", p. 257 | unhappy (empty and minimal) |
| FR-030 | `spend(part)` for a ref with no entry MUST return `LimitedUseSpendRefused(notLimitedUse)`, the state unchanged. | SRD 5.2.1, "Limited Usage", p. 257 | unhappy (missing content) |
| FR-031 | Every operation MUST return a new state and leave the receiver and its inputs unchanged. Entry lists, `RechargeRollsNeeded` and `LimitedUseTransition.Events` MUST be copies (mutating a returned slice does not change the state); events are in the order they happened. | architecture.md §3, §4; `combat-foundations.md` FR-021 | unhappy (immutability) |
| FR-032 | `LimitedUseState`, its entries and the closed `LegendaryTrack` and `LegendaryWindowRecord` MUST be value objects: equal fields give equal values; any differing field makes them unequal. | architecture.md §3 | happy |
| FR-033 | The events `LimitedUsePartSpent`, `MonsterRechargeRolled`, `LimitedUsesRestored`, `LegendaryActionTaken`, `LegendaryActionUsesRegained` and `MonsterLairStatusChanged` MUST be the members of one closed `MonsterLimitedUseEvent` that implements `CombatEvent`, (`LimitedUsePartSpent` and `LimitedUsesRestored` name `LimitedUseSubject`s, FR-050), each with a constant unique `kind` (`limited_use_part_spent`, `monster_recharge_rolled`, `limited_uses_restored`, `legendary_action_taken`, `legendary_action_uses_regained`, `monster_lair_status_changed`) and `combatants` equal to `[combatant]`. | `combat-foundations.md` FR-015 to FR-020 | happy |
| FR-034 | Every operation MUST return `LimitedUseSpent`, `LimitedUseSpendRefused` (also from `SpendSpell`), `LegendaryActionAllowed`, `LegendaryActionRefused` or `LimitedUseTransition`; none MUST return an error or panic for a refusal. Only FR-005, FR-012, FR-016 (round) and FR-044 return an error matching `srderr.ErrInvalidInput`; every other call returns no error. | architecture.md §6 | unhappy (invalid input) |

### 4.7 The use case

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-035 | `ApplyMonsterTurnStart` MUST take `dice.Roller` and an optional `telemetry.Telemetry` (default `telemetry.Telemetry.none`) through its constructor and return the closed `ApplyMonsterTurnStartResult`, whose only variant `MonsterTurnStartApplied` holds the new state, the events and the rolls made. It MUST contain no rule: it calls `RechargeRollsNeeded` and `StartOfOwnTurn` (FR-010, FR-024). | `shared-types-and-ownership.md` FR-017, FR-023, FR-044; architecture.md §4 | happy |
| FR-036 | For each part of `RechargeRollsNeeded`, in order, the use case MUST roll `dice.NewExpression(1, dice.D6(), 0)` once through `dice.Roll`, then call `StartOfOwnTurn` with the faces. It MUST make no roll when the list is empty. Rolls are not made for a part the state does not need (FR-010). | SRD 5.2.1, "Limited Usage" ("roll 1d6"), p. 257; `dice-and-rolls.md` FR-045, FR-050 | happy |
| FR-037 | A `dice.Roller` error MUST be returned wrapped by `srderr.WrapPort` (a `*srderr.PortError` for port `dice`; `errors.Is` finds the cause), with no state or event returned. The span and the `srd5e.combat.errors` count record it once (`combat-foundations.md` FR-032). | `errors-and-results.md` FR-034; `combat-foundations.md` FR-032 | unhappy (failure) |
| FR-038 | The use case MUST NOT call or hold another use case, and MUST behave identically with the zero-value (Noop) and a recording `telemetrytest` double. | `shared-types-and-ownership.md` FR-044; observability.md §9 | unhappy (rule interaction) |
| FR-039 | The use case MUST record the shared `combat` signals through the `combat` telemetry helper over `telemetry.Telemetry` under the operation `apply_monster_turn_start` (span `srd5e.combat.apply_monster_turn_start`) and count each roll in `srd5e.combat.monster_recharge_rolls` (section 7). Each result is class `applied` (no refusal exists); `srd5e.combat.result` is `turn_start_recharged` (at least one roll regained), `turn_start_not_recharged` (rolls made, none regained) or `turn_start_no_rolls`. The counter is incremented only when the result is returned, one count per roll made; on a returned port failure it is not incremented. | `combat-foundations.md` FR-027 to FR-032, FR-039, FR-040; observability.md §5 | happy |

### 4.8 Placement and API

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-040 | The state, entries, events and rules MUST live in the `limited_use*.go` files of package `combat`, be synchronous and deterministic (no `math/rand`, `time.Now`, goroutines, `context`, file or network I/O) and contain no `Logger`, `Tracer`, `Meter` or signal constant. The use case MUST live in `apply_monster_turn_start.go`. Imports are limited to the packages `content` and `character` (`Monster`, `PartUsage`, `LegendaryActions`, `RestKind`), the kernel packages (`dice` and `telemetry` for the use case only) and `combat` files to the left in the order of `combat-foundations.md` FR-024 (enforced by `internal/archtest`). | architecture.md §1, §3, §9; observability.md §3 | happy |
| FR-041 | The types of sections 5 and 6 MUST be exported from package `combat` with a godoc each; no exported name MAY duplicate another exported name in the package, and no exported name is generic (`Applied`, `Refused`). `LegendaryActionWindow` and `ApplyMonsterTurnStartCommand` are the handoff types a host or the encounter builds. | architecture.md §4, §7; `combat-foundations.md` FR-022, FR-024 | happy |
| FR-042 | Nothing in the code, tests or data of this spec MAY name a consuming project or use content from outside SRD 5.2.1. | AGENTS.Md ("Hard constraints") | happy |

### 4.9 Spell frequencies

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-043 | `start` MUST add, after the part entries, one `SpellUseEntry(ref, maximum, remaining)` for each spell of each Spellcasting group whose `SpellFrequency` is `SpellFrequencyPerDay(n)` or `SpellFrequencyPerDayEach(n)`, in printed order (group, then spell), with `maximum` and `remaining` equal to `n`. A `SpellFrequencyAtWill` spell and a monster without Spellcasting add no entry. | SRD 5.2.1, "Spellcasting", p. 257; "Archmage", p. 305 | happy |
| FR-044 | `SpellUseRef(groupIndex, spellIndex)` MUST return an error matching `srderr.ErrInvalidInput` for a negative index; equal indexes give equal values. | architecture.md §6 | unhappy (invalid input) |
| FR-045 | `SpendSpell(SpellUseRef)` MUST, with at least 1 use remaining, return `LimitedUseSpent` with the entry reduced by exactly 1 and one `LimitedUsePartSpent` event; with 0 remaining `LimitedUseSpendRefused(exhausted)`; for a ref with no entry (At Will, or a position not in the stat block) `LimitedUseSpendRefused(notLimitedUse)`; state unchanged on refusal. | SRD 5.2.1, "Limited Usage" ("X/Day"), "Spellcasting", p. 257 | unhappy (boundary) |
| FR-046 | `N/Day` and `N/Day Each` MUST both give every listed spell its own `n` uses, never a pool shared by the group. Interpretation: every printed `N/Day` group in SRD 5.2.1 lists a single spell (so the readings coincide there), and the reading with independent uses matches common practice for stat-block spell lists. | SRD 5.2.1, "Spellcasting", p. 257; "Priest", p. 316 | unhappy (rule interaction) |
| FR-047 | A Long Rest MUST restore every spell entry to its maximum; a Short Rest MUST NOT. | SRD 5.2.1, "Limited Usage" ("must finish a Long Rest to regain expended uses"), p. 257 | unhappy (rule interaction) |
| FR-048 | `AvailabilityOfSpell(SpellUseRef)` MUST return `available`, `exhausted` or `notLimited` as FR-029 does for parts, and MUST NOT return an error for a valid ref. | SRD 5.2.1, "Spellcasting", p. 257 | unhappy (empty and minimal) |
| FR-049 | `ChangeLair` and `StartOfOwnTurn` MUST leave spell entries unchanged: the SRD gives no in-Lair figure and no turn-start regain for spell uses. | SRD 5.2.1, "Aboleth", p. 258; "Limited Usage", p. 257 | unhappy (rule interaction) |
| FR-050 | `LimitedUsePartSpent` and `LimitedUsesRestored` MUST name their subject as a closed `LimitedUseSubject` (`PartSubject(LimitedUsePartRef)`, `SpellSubject(SpellUseRef)`); `Spend` and `SpendSpell` MUST NOT accept each other's refs (separate parameters). Spell entries MUST NOT appear in `RechargeRollsNeeded`. | package design; `combat-foundations.md` FR-015 | happy |

### Cases RC-1: Recharge (one test per row; FR-011)

Face is the d6 rolled for a spent part.

| Label | Part (low-high) | Face | Expected | Source | Path |
|---|---|---|---|---|---|
| 5-6, face 4 | `PartRecharge(5, 6)` | 4 | stays spent | "Limited Usage", p. 257 | unhappy (boundary) |
| 5-6, face 5 | `PartRecharge(5, 6)` | 5 | ready | p. 257 | unhappy (boundary) |
| 5-6, face 6 | `PartRecharge(5, 6)` | 6 | ready | p. 257 | happy |
| 5-6, face 1 | `PartRecharge(5, 6)` | 1 | stays spent | p. 257 | unhappy (boundary) |
| 4-6, face 3 | `PartRecharge(4, 6)` Whirlwind | 3 | stays spent | "Air Elemental", p. 259 | unhappy (boundary) |
| 4-6, face 4 | `PartRecharge(4, 6)` | 4 | ready | "Air Elemental", p. 259 | unhappy (boundary) |
| 6 only, face 5 | `PartRecharge(6, 6)` Acid Spray | 5 | stays spent | "Ankheg", p. 259 | unhappy (boundary) |
| 6 only, face 6 | `PartRecharge(6, 6)` | 6 | ready | "Ankheg", p. 259 | unhappy (boundary) |
| widest range, face 1 | synthetic `PartRecharge(1, 6)` | 1 | ready | `content-monsters.md` FR-035 (lowest allowed bound) | unhappy (boundary) |
| single middle value, face 2 | synthetic `PartRecharge(3, 3)` | 2 | stays spent | FR-011 | unhappy (boundary) |
| single middle value, face 3 | synthetic `PartRecharge(3, 3)` | 3 | ready | FR-011 | unhappy (boundary) |
| single middle value, face 4 | synthetic `PartRecharge(3, 3)` | 4 | stays spent | FR-011 | unhappy (boundary) |
| ready part, no roll | `PartRecharge(5, 6)`, `ready` | none | not in `RechargeRollsNeeded` | FR-010 | unhappy (empty and minimal) |
| rest-recharge part, no roll | `PartRechargeAfterRest`, `spent` | none | not in `RechargeRollsNeeded`; stays spent at turn start | "Cloaker", p. 274; FR-013 | unhappy (rule interaction) |
| two spent parts | `PartRecharge(5, 6)` then `PartRecharge(6, 6)`, faces 5 then 5 | 5, 5 | first ready, second spent | FR-024 | unhappy (rule interaction) |
| wrong die | roll map holding a d8 | n/a | `srderr.ErrInvalidInput` error | FR-012 | unhappy (invalid input) |
| missing roll | two spent parts, one roll | n/a | `srderr.ErrInvalidInput` error | FR-012 | unhappy (invalid input) |
| extra roll | one spent part, rolls for it and a ready part | n/a | `srderr.ErrInvalidInput` error | FR-012 | unhappy (invalid input) |

### Cases XD-1: X/Day and starting state (one test per row; FR-001 to FR-008, FR-026, FR-027, FR-029, FR-030)

| Label | Monster and lair | Action | Expected | Source | Path |
|---|---|---|---|---|---|
| per day, outside | Aboleth, `outsideLair`: Legendary Resistance (3/Day, or 4/Day in Lair) | start | 3 remaining | "Aboleth", p. 258 | happy |
| per day, in lair | Aboleth, `inLair` | start | 4 remaining | "Aboleth", p. 258 | happy |
| per day, no lair figure | Aboleth `inLair`: Dominate Mind (2/Day) | start | 2 remaining | "Aboleth", p. 258 | unhappy (rule interaction) |
| spend to zero | Dominate Mind 2/Day | spend, spend, spend | `LimitedUseSpent` (1), `LimitedUseSpent` (0), `LimitedUseSpendRefused(exhausted)` | p. 257 | unhappy (boundary) |
| 1/Day | synthetic Reaction 1/Day | spend twice | `LimitedUseSpent` (0), then refused | "Limited Usage", p. 257 | unhappy (boundary) |
| unlimited part | Aboleth Tentacle (`PartUnlimited`) | spend its ref | `LimitedUseSpendRefused(notLimitedUse)` | FR-030 | unhappy (missing content) |
| other usage | Night Hag part (1/Day; Requires Soul Bag) | start | no entry; `AvailabilityOf` is `notLimited` | `content-monsters.md` FR-035 | unhappy (rule interaction) |
| ref past the list | position 99 of Actions | `AvailabilityOf` | `notLimited` | FR-029 | unhappy (boundary) |
| negative index | `LimitedUsePartRef(actions, -1)` | construct | `srderr.ErrInvalidInput` error | FR-005 | unhappy (invalid input) |
| no limited parts | monster with only unlimited parts | start | no entries, `LegendaryNone` | FR-001, FR-004 | unhappy (empty and minimal) |
| short rest keeps uses | Dominate Mind at 0 | `AfterRest(shortRest)` | still 0, no event | FR-008 | unhappy (rule interaction) |
| long rest restores | Dominate Mind at 0, Legendary Resistance at 1 | `AfterRest(longRest)` | 2 and 3 (outside), one `LimitedUsesRestored` listing both | FR-026, FR-027 | happy |
| long rest in lair | same monster `inLair`, Legendary Resistance at 1 | `AfterRest(longRest)` | 4 | FR-026 | unhappy (rule interaction) |
| rest with nothing spent | full state | `AfterRest(longRest)` | equal state, no event | FR-027 | unhappy (empty and minimal) |

### Cases SP-1: Spell frequencies (one test per row; FR-043 to FR-050)

| Label | Monster and input | Expected | Source | Path |
|---|---|---|---|---|
| per day each | synthetic group `1/Day Each: A, B` (the shape SRD 5.2.1 prints in "Spellcasting", p. 257) | two entries, 1 each | "Spellcasting", p. 257 | happy |
| per day single | Priest `2/Day: Command` | one entry, 2 | "Priest", p. 316 | happy |
| at will | `At Will: Detect Magic` | no entry; `AvailabilityOfSpell` is `notLimited` | p. 257 | unhappy (rule interaction) |
| spend to zero | `1/Day` spell, spend twice | `LimitedUseSpent` (0), then refused `exhausted` | p. 257 | unhappy (boundary) |
| independent uses | synthetic `2/Day: A, B`, spend A twice | A 0, B 2 | FR-046 | unhappy (rule interaction) |
| ref past the list | group 9 | refused `notLimitedUse` | FR-045 | unhappy (boundary) |
| negative index | `SpellUseRef(-1, 0)` | `srderr.ErrInvalidInput` error | FR-044 | unhappy (invalid input) |
| short rest | spell at 0 | still 0, no event | FR-047 | unhappy (rule interaction) |
| long rest | spell at 0 and Dominate Mind at 0 | both restored, one event listing a `PartSubject` then a `SpellSubject` | FR-047, FR-027 | happy |
| lair change and turn start | spell at 0 | unchanged | FR-049 | unhappy (rule interaction) |
| no spellcasting | Ankheg | no spell entries | FR-043 | unhappy (empty and minimal) |

### Cases LA-1: Legendary Actions (one test per row; FR-004, FR-015 to FR-023, FR-028)

Window W1 is `(endedTurnOf: B, round: 1)`, W2 is `(endedTurnOf: C, round: 1)`; the Aboleth is combatant `A`.

| Label | State and input | Expected | Source | Path |
|---|---|---|---|---|
| start outside lair | Aboleth `outsideLair` | `LegendaryTracked(3, 3)` | "Aboleth", p. 258 | happy |
| start in lair | Aboleth `inLair` | `LegendaryTracked(4, 4)` | p. 258 | happy |
| no in-lair figure | Solar `inLair` | `LegendaryTracked(3, 3)` | "Solar", p. 327 | unhappy (rule interaction) |
| taken after another's turn | 3 left, W1, `available` | `LegendaryActionAllowed`, 2 left, W1 recorded, `LegendaryActionTaken` | p. 257 | happy |
| last use | 1 left, W1 | allowed, 0 left | p. 257 | unhappy (boundary) |
| none left | 0 left, W1 | refused `noUsesRemaining` | p. 257 | unhappy (boundary) |
| own turn | W with `endedTurnOf: A` | refused `ownTurn` | p. 257 | unhappy (rule interaction) |
| incapacitated | 3 left, W1, `none` | refused `incapacitated`, state unchanged | "Incapacitated", p. 184 | unhappy (state error) |
| second in same window | W1 used, 2 left, W1 again | refused `windowAlreadyUsed` | FR-018 | unhappy (rule interaction) |
| second window | W1 used, W2 | allowed, 1 left | FR-018 | unhappy (rule interaction) |
| same combatant, next round | W1 used, `(B, round 2)` | allowed | FR-018 | unhappy (boundary) |
| no legendary actions | Ankheg state, W1 | refused `noLegendaryActions` | p. 257 | unhappy (empty and minimal) |
| precedence, own turn and incapacitated | own-turn window, `none` | `ownTurn` | FR-021 | unhappy (rule interaction) |
| precedence, incapacitated and used | W1 used, `none` | `incapacitated` | FR-021 | unhappy (rule interaction) |
| precedence, used and none left | W1 used, 0 left | `windowAlreadyUsed` | FR-021 | unhappy (rule interaction) |
| precedence, no legendary and own turn | `LegendaryNone`, own turn | `noLegendaryActions` | FR-021 | unhappy (rule interaction) |
| round below 1 | `LegendaryActionWindow(B, 0)` | `srderr.ErrInvalidInput` error | FR-016 | unhappy (invalid input) |
| turn start regains | 0 of 3 | `StartOfOwnTurn`: 3 of 3, `LegendaryActionUsesRegained(3)` | p. 257 | happy |
| turn start partial | 2 of 3 | 3 of 3, `LegendaryActionUsesRegained(1)` | p. 257 | unhappy (boundary) |
| turn start full | 3 of 3 | 3 of 3, no event | p. 257 | unhappy (empty and minimal) |
| rest leaves uses | 0 of 3, `AfterRest(longRest)` | still 0 | FR-023 | unhappy (rule interaction) |
| enter lair | Aboleth outside, 1 of 3 left (2 spent) | `ChangeLair(inLair)`: 2 of 4, `MonsterLairStatusChanged` | FR-028 | unhappy (rule interaction) |
| leave lair, spent exceeds | Aboleth in lair, 0 of 4 | `ChangeLair(outsideLair)`: 0 of 3 | FR-028 | unhappy (boundary) |
| lair unchanged | `inLair`, `ChangeLair(inLair)` | equal state, no event | FR-028 | unhappy (empty and minimal) |

## 5. Model

- **Value objects:** `LimitedUseState` (`combatant`, `monsterID`, `lair`, ordered
  `entries`, `legendary`, `windowRecord`), closed `LimitedUseEntry`
  (`PerDayEntry(part, maximum, maximumInLair?, remaining)`,
  `RechargeEntry(part, low, high, readiness)`, `RestRechargeEntry(part,
  readiness)`), `LimitedUsePartRef` (the position of any stat-block part;
  `monster-stat-block-actions.md` reuses it, no second ref type),
  `MonsterPartSection`, `LairStatus`, `RechargeReadiness`, `SpellUseEntry(ref, maximum, remaining)` (the sixth `LimitedUseEntry` variant), `SpellUseRef`, `LimitedUseSubject`, `LegendaryTrack`, `LegendaryWindowRecord`,
  `LegendaryActionWindow`, `LimitedUseAvailability` (`available`, `exhausted`,
  `notLimited`), `LimitedUseTransition` (`state`, `events`).
- **Results (closed, operation-prefixed):** `LimitedUseSpendOutcome`
  (`LimitedUseSpent(transition)`, `LimitedUseSpendRefused(reason)`),
  `LimitedUseRefusal` (`exhausted`, `notLimitedUse`), `LegendaryActionOutcome`
  (`LegendaryActionAllowed(transition)`, `LegendaryActionRefused(reason)`),
  `LegendaryActionRefusal` (`noLegendaryActions`, `ownTurn`, `incapacitated`,
  `windowAlreadyUsed`, `noUsesRemaining`).
- **Events:** closed `MonsterLimitedUseEvent` (FR-033) and enum `RechargeRollResult`.
- **Use case:** `ApplyMonsterTurnStart`, `ApplyMonsterTurnStartCommand(state)`,
  closed `ApplyMonsterTurnStartResult` (`MonsterTurnStartApplied`).
- **Self-contained state:** each entry copies the printed numbers it needs, so no
  operation takes the `Monster` again. `monsterID` is copied from `monster.id` by
  `start` and is kept only for the host's own checks (a state for the wrong stat
  block); no operation reads it.
- **Invariants:** `0 <= remaining <= maximum` (also `LegendaryTracked`); entries in
  printed order with distinct refs (constructor, `srderr.ErrInvalidInput` error).
- **Failures:** no error type is defined here beyond the `srderr.ErrInvalidInput` errors above; a `dice.Roller` error is returned by the use case wrapped as a `*srderr.PortError` (FR-037).

## 6. Interface

```go
// Package combat, file limited_use_state.go (indicative signatures).
type LimitedUseState struct{ /* unexported */ }

// Errors below match srderr.ErrInvalidInput (untrusted input); refusals are values.
func StartLimitedUse(m content.Monster, combatant CombatantID, lair LairStatus) (LimitedUseState, error) // FR-001..FR-004
func (s LimitedUseState) AvailabilityOf(part LimitedUsePartRef) LimitedUseAvailability                   // FR-029
func (s LimitedUseState) RechargeRollsNeeded() []LimitedUsePartRef                                      // FR-010
func (s LimitedUseState) Spend(part LimitedUsePartRef) LimitedUseSpendOutcome                           // FR-006..FR-009, FR-030
func (s LimitedUseState) SpendSpell(spell SpellUseRef) LimitedUseSpendOutcome                           // FR-045
func (s LimitedUseState) AvailabilityOfSpell(spell SpellUseRef) LimitedUseAvailability                  // FR-048
func (s LimitedUseState) TakeLegendaryAction(w LegendaryActionWindow,
	a ConditionActionAvailability) (LegendaryActionOutcome, error)                                  // FR-015..FR-021
func (s LimitedUseState) StartOfOwnTurn(rolls map[LimitedUsePartRef]dice.DieRoll) (LimitedUseTransition, error) // FR-011, FR-022, FR-024
func (s LimitedUseState) AfterRest(rest RestKind) LimitedUseTransition                                  // FR-025..FR-027
func (s LimitedUseState) ChangeLair(status LairStatus) LimitedUseTransition                             // FR-028

type LairStatus int          // InLair, OutsideLair
type MonsterPartSection int  // Traits, Actions, BonusActions, Reactions, LegendaryActions
type SpellUseRef struct{ /* groupIndex, spellIndex */ }       // comparable value
type LimitedUsePartRef struct{ /* section, index */ }          // comparable value, usable as a map key
type LegendaryActionWindow struct{ /* endedTurnOf CombatantID, round int */ } // NewLegendaryActionWindow validates round (FR-016)
type LimitedUseTransition struct{ /* state, events */ }        // State() and Events() (a copied slice)

// Closed variant sets: interface with an unexported marker method.
type LimitedUseSpendOutcome interface{ /* LimitedUseSpent | LimitedUseSpendRefused(LimitedUseRefusal) */ }
type LegendaryActionOutcome interface{ /* LegendaryActionAllowed | LegendaryActionRefused(LegendaryActionRefusal) */ }

// Package combat, file apply_monster_turn_start.go.
type ApplyMonsterTurnStart struct{ /* unexported */ }
func NewApplyMonsterTurnStart(r dice.Roller, opts ...Option) ApplyMonsterTurnStart // Option: WithTelemetry (zero value = Noop)
func (u ApplyMonsterTurnStart) Do(ctx context.Context, c ApplyMonsterTurnStartCommand) (ApplyMonsterTurnStartResult, error) // FR-035..FR-039

type ApplyMonsterTurnStartCommand struct{ State LimitedUseState }
type ApplyMonsterTurnStartResult interface{ /* MonsterTurnStartApplied */ } // closed variant set
type MonsterTurnStartApplied struct{ /* State(), Events(), Rolls() []MonsterRechargeRolled (copied) */ }
```

- **Driving it:** on `TrackerTurnStarted` for a monster: `ApplyMonsterTurnStart`.
  On `TrackerTurnEnded` for any creature: build a `LegendaryActionWindow` for each
  monster with Legendary Actions and offer `TakeLegendaryAction`
  (`initiative-and-actions.md` FR-028, FR-029). On a finished rest: `AfterRest`.
  The host stores the state and makes these calls (Q-5).
- **Synchronous:** `dice.Roller` is synchronous and takes no `ctx` (`dice-and-rolls.md` FR-045), so
  `ApplyMonsterTurnStart.Do` returns its result directly; it takes `ctx` first only because it starts a span (decision 2b).
- **Exports:** every type of sections 5 and 6 from package `combat` (no facade or re-export package). `dice/dicetest` and `telemetry/telemetrytest` gain nothing (`dicetest.FixedRoller` and the
  recording `telemetrytest` double suffice).

## 7. Observability

The domain is **silent by design** (observability.md §3). Only
`ApplyMonsterTurnStart` emits, through the `combat` telemetry helper over `telemetry.Telemetry`. The four shared
`combat` signals (calls, duration, errors, result) are owned by
`combat-foundations.md` (its FR-037); this spec owns only the two below.
Evaluation (observability.md §2):

1. Failure modes: a dice adapter that returns an error (`srd5e.combat.errors`, `error.type`
   `dice_roller_error`, span error status); one that is stuck so a Recharge
   part never or always regains (the `regained` share of the counter); a host
   that passes a state for the wrong monster (not detectable; `monsterID` is for
   the host's own checks).
2. Signals that show it: the span error status and `errors` count; the counter's
   `regained` share.
3. Attributes: bounded values only; faces and part names are in the returned
   events, not in attributes (`combat-foundations.md` FR-033).
4. What to count: each Recharge roll and its outcome, once per roll in a returned
   result (FR-039).
5. Existing instrumentation: the four shared signals are unchanged and owned by
   `combat-foundations.md` FR-037.
6. Logs: none (`combat-foundations.md` FR-034).
7. Cardinality: the counter has two attribute values.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.apply_monster_turn_start` | span | `srd5e.combat.operation`, `srd5e.outcome`, `srd5e.combat.result` (`turn_start_recharged`, `turn_start_not_recharged`, `turn_start_no_rolls`), `srd5e.combat.result_class` (`applied`) | The path of one monster turn start | Sample traces; alert on error-status spans (a port failure) |
| `srd5e.combat.monster_recharge_rolls` | counter `{roll}` | `srd5e.combat.recharge_outcome` (`regained`, `not_regained`) | Rule outcome of every Recharge roll, so a stuck or biased dice adapter shows up | Compare the `regained` share with the printed ranges (5-6 is 1 in 3, 4-6 is 1 in 2, 6 is 1 in 6); near 0% or 100% over at least 200 rolls points at the roller adapter |

Signal constants (`CombatSignals`, `combat-foundations.md` FR-039, FR-040):
`applyMonsterTurnStartOperationValue`, `applyMonsterTurnStartSpan`,
`monsterRechargeRollsMetric`, `rechargeOutcomeAttribute`,
`rechargeOutcomeRegainedValue`, `rechargeOutcomeNotRegainedValue`, and the three `srd5e.combat.result` values
`turnStartRechargedValue`, `turnStartNotRechargedValue` and
`turnStartNoRollsValue`, each with a doc comment naming `srd5e.combat.result`. These rows feed `doc/observability/signals.md`.

## 8. Test plan

Tests are in package `combat_test` (`combat/limited_use_*_test.go`) and `combat/apply_monster_turn_start_test.go`; fixtures in `combat/testdata/`. Unit tests use synthetic and bundled-shape stat-block
parts, no ports or doubles. Each row of a case table is one table-driven test.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-005 | unit | `limited_use_state_start_test.go`: XD-1 and LA-1 start rows; refs; unlimited and `PartUsageOther` rows |
| FR-006 to FR-008 | unit | `per_day_test.go`: XD-1 spend, exhaust and Short Rest rows |
| FR-009 to FR-014 | unit | `recharge_test.go`: RC-1, all rows including invalid-roll-map rows |
| FR-015 to FR-023 | unit | `legendary_action_test.go`: LA-1 allowed, refusal, precedence, turn-start regain and rest rows |
| FR-024 | unit | `start_of_own_turn_test.go`: event order with two rolls then a regain; no-op turn |
| FR-025 to FR-028 | unit | `rest_test.go`, `lair_test.go`: Short and Long Rest rows, lair maximum, event list, nothing-changed, the three lair rows, a part with no lair figure |
| FR-029 to FR-032 | unit | `availability_test.go`, `immutability_test.go`: the three values, ref past the list, inputs unchanged, mutating each returned slice leaves the state unchanged, equality and comparison |
| FR-033, FR-034 | unit, architecture | `events_test.go`: kinds, `combatants`, exhaustive type switch; `internal/archtest`: kind uniqueness and the `CombatEvent` family scan (`combat-foundations.md` FR-016, FR-017); no operation returns an error or panics for a refusal |
| FR-035 to FR-039 | application | `apply_monster_turn_start_test.go` with `dicetest.FixedRoller` and the recording `telemetrytest` double: faces in printed order; no roll when none needed; a roller error (also on the second of two rolls) is returned wrapped as a `*srderr.PortError` with one `errors` increment, no `monster_recharge_rolls` increment and no log; equal results with no-op and recording `telemetrytest` double; span name, result values, `monster_recharge_rolls` counts, no id in any attribute |
| FR-043 to FR-050 | unit | `spell_use_test.go`: SP-1 rows, subject names in events |
| FR-040 to FR-042 | architecture | `internal/archtest`: file purity, import allow-list, no signal code in the pure files, exported-name scan (exports, godoc, duplicate or generic names), consumer-name scan |

- **Doubles and suites:** none new.
- **Mutation testing:** the `limited_use*.go` files of package `combat`, with gomu run only inside Docker
  with `--cpus` and `--workers` capped at 1/4 of `nproc` (testing.md §8; never run locally), watching `<=` versus `<` in FR-011
  at both ends, `max(0, ...)` in FR-028, the refusal order of FR-021, and
  `maximum` versus `maximumInLair` in FR-002. `ApplyMonsterTurnStart` SHOULD be
  mutation tested.
- **Not covered:** nothing.

## 9. Open questions and decisions

Settled, one line each:

- Q-1: one Legendary Action per window (FR-018); the SRD's "Only one of these
  actions can be taken at a time" is read as excluding two in one gap
  (interpretation: p. 257 also ties each action to a creature's turn ending).
- Q-2: lair change keeps uses already spent (FR-028); the SRD gives only the numbers.
- Q-3: Lair actions are not in SRD 5.2.1 (section 1); left out, flagged here.
- Q-4: spell-list `N/Day` and `N/Day Each` uses are counted now (section 4.9,
  FR-043 to FR-050), each spell with its own `n` uses (interpretation, FR-046);
  `At Will` needs no entry. Chosen over a later closed-subject extension because
  p. 257 puts "X/Day" and Spellcasting frequencies under the same Long Rest rule.
- Q-5: the host stores one `LimitedUseState` per monster combatant and calls
  `ApplyMonsterTurnStart` on `TrackerTurnStarted`, builds a `LegendaryActionWindow`
  on `TrackerTurnEnded` and calls `AfterRest` on a finished rest. `Encounter` and
  `Combatant` do not hold it (`encounter-aggregate.md` FR-082, FR-083 add no field):
  package design, state in/out with no hidden owner (architecture.md §3, §4).
- Q-6: rests do not refill Legendary Action uses (FR-023).
- Q-7: the host decides which `RestKind` it reports (`rests-and-hit-dice.md` Q-8, Q-16).
- Q-8: this spec uses `ConditionActionAvailability` (`conditions.md` FR-064, the
  owner; `combat-action-resolution.md` already uses it).

Open: none. Escalated: none.

## 10. Changelog impact

`Added` (combat): `LimitedUseState` and its rules (X/Day, spell N/Day, Recharge X-Y, Recharge
after a Short or Long Rest, Legendary Action timing and uses), `ApplyMonsterTurnStart`,
the `MonsterLimitedUseEvent` family, and the signals `srd5e.combat.apply_monster_turn_start`
and `srd5e.combat.monster_recharge_rolls`. Not **BREAKING**: nothing is released.
