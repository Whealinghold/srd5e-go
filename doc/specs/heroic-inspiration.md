# Heroic Inspiration

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `character-aggregate.md`, `dice-and-rolls.md`, `observability-ports.md`, `errors-and-results.md` |

## 1. Purpose and scope

The instrumented use cases and host-boundary mechanics for Heroic Inspiration.
`shared-types-and-ownership.md` FR-048 makes this the topic-specific spec: it
owns the mechanics, the use cases and their signals; `character-aggregate.md`
(D-15 there) keeps the state slot. It adds no state, no events and no rule
logic beyond one pure domain function (FR-038).

Owned by `character-aggregate.md` (decided by `shared-types-and-ownership.md`
FR-010) and not redefined here: `HeroicInspiration` (`Inspired`,
`NotInspired`), `Character.HeroicInspiration` (FR-089),
`GrantHeroicInspiration()` (FR-090), `ExpendHeroicInspiration()` (FR-091),
`HeroicInspirationGained`, `HeroicInspirationExpended` and the rejection
`HeroicInspirationNotAvailable` (a `HeroicInspirationRejection`; FR-085 for
the event family). This spec adds:

1. **Gaining at the host boundary.** One operation for any grant (GM award,
   Human Resourceful, Champion Heroic Warrior), including the choice when the
   holder already has it: lose the new one or give it to a player character who
   lacks it (SRD 5.2.1, "Heroic Inspiration", p. 8, "Only One at a Time";
   "Rules Glossary", "Heroic Inspiration", p. 183).
2. **Expending, with the new face.** The aggregate transition spends the state;
   this use case also rolls the single die being rerolled through the
   `dice.Roller` and returns the new face (p. 8; p. 183). The die may be any die,
   so the face is drawn here (D-14).
3. **Telemetry** for both (section 7).

The d20 reroll rule is specified in `d20-tests.md` FR-012 to FR-015
(`Roll.ReplaceDie`) and FR-059 (`RerollDecision`, `ApplyReroll`); they take
the new face as input and this spec supplies it (D-2). Use cases are state in,
new state and events out (`doc/rules/architecture.md` section 4).

**Placement.** Context `character`, a leaf after `character-aggregate.md` and
before every `class-*` spec, alongside `rests-and-hit-dice.md`. No `class-*`,
`combat`, `toolbox` or `d20` type is imported (FR-034). Related, not
dependencies: `d20-tests.md`, `class-fighter.md` (FR-093, FR-135),
`content-thin-slice.md` (FR-030).

**Supersedes** (those specs reconcile on their own; this spec edits none):
`character-aggregate.md` FR-090 last sentence (excess instance) and FR-091 last
sentence (new face, reroll, use cases), now owned by FR-003 to FR-011, FR-038,
FR-039 and FR-012 to FR-019 here; `character-aggregate.md` D-15; the owner
named by `rules-glossary-coverage.md` GL-077c. No other spec defines a Heroic
Inspiration use case, command, result, event or signal.

**Out of scope:**

- **What grants Heroic Inspiration:** the GM award (p. 8), Human Resourceful
  (p. 86) and Champion Heroic Warrior (p. 49) are content or class inputs. The
  host calls `GainHeroicInspiration` with the result; no use case calls it
  (`shared-types-and-ownership.md` FR-044, FR-037).
- **Recomputing a D20 Test after a replaced d20:** the host builds
  `RerollDie(position)` (FR-034).
- **Adding a rerolled die to damage or totals:** the host or damage-roll owner.
- **"Immediately after rolling":** no clock, turn or roll history is modelled
  (D-8). **"In your group":** a host fact; the command carries the recipient
  `Character` only.
- **Monsters and non-player characters:** the use cases take a `Character`
  (p. 183). **Persistence:** the host's concern.
- **Reroll effects of classes and spells** (Indomitable, Empowered Spell and so
  on): owned by their specs. **Bardic Inspiration:** `class-bard.md`.
- Any rule from SRD 5.1 or another book, or not on pp. 8 and 183 (or the grant
  sites above) of SRD 5.2.1.

## 2. Source

- SRD 5.2.1, "Heroic Inspiration" (sidebar), "Only One at a Time", "Gaining
  Heroic Inspiration", p. 8.
- SRD 5.2.1, "Rules Glossary", "Heroic Inspiration", p. 183.
- SRD 5.2.1, "Advantage/Disadvantage", "Interactions with Rerolls", p. 8.
- Grant sites only: "Level 10: Heroic Warrior", p. 49; Human "Resourceful", p. 86.
- Package rules: `doc/rules/architecture.md` sections 3, 4, 6;
  `doc/rules/observability.md` sections 2 to 7; `doc/rules/testing.md` section 4.

## 3. Vocabulary

Types owned elsewhere are listed for reference and not defined or exported here.

| SRD term | Type | Notes |
|---|---|---|
| Heroic Inspiration | `HeroicInspiration` (`Inspired`, `NotInspired`) | `character-aggregate.md` FR-089; read as `Character.HeroicInspiration` |
| gain and expend events | `HeroicInspirationGained`, `HeroicInspirationExpended` | `character-aggregate.md` section 5; `CharacterEvent`s carrying `CharacterID` |
| "nothing to expend" | `HeroicInspirationNotAvailable` | `character-aggregate.md` FR-091; reused as the expend rejection |
| "gain Heroic Inspiration" | `GainHeroicInspiration` (use case) | New. Same operation whoever grants it (p. 8) |
| "give it to a player character in your group who lacks it" | `ExcessInspirationChoice` (closed variant set): `LoseExcessInspiration`, `GiveExcessInspirationTo{Recipient Character}` | New |
| "expend it to reroll any die" | `ExpendHeroicInspiration` (use case) | New. Rolls one die through `dice.Roller` |
| "any die", "one die, not both" | `RerollTarget` (`Die()`, `DiceRolled()`, `Position()`) | New. `DiceRolled` is how many dice were rolled together (2 for Advantage or Disadvantage); `Position` is the die chosen (D-13) |
| "the new roll" | `RerolledDie` (`Die()`, `Position()`, `NewValue()`) | New. No alternative to the new value |

`Inspired` and `NotInspired` are package names, not SRD terms.

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. "Rejected" in a
result means a closed rejection variant, not an `error`. An invalid
constructor argument returns an error matching `srderr.ErrInvalidInput` (`architecture.md` section 6).

### 4.1 Delegation

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | The use cases MUST delegate every state change to `Character.GrantHeroicInspiration()` and `Character.ExpendHeroicInspiration()` (FR-090, FR-091 there). The gain-with-excess decision is delegated to `ResolveExcessInspiration` (FR-038). No second implementation of any of these rules exists: no state is derived from anything but those transitions and `Character.HeroicInspiration`, and no use case branches on the holder's state, the choice or the recipient. | n/a (single owner, `shared-types-and-ownership.md` FR-010) | happy |
| FR-032 | The events of every result MUST be the `HeroicInspirationGained` and `HeroicInspirationExpended` events produced by those transitions, unchanged. This spec defines no event type and requires no amendment to `character-aggregate.md` section 5 (D-11). | n/a (`character-aggregate.md` FR-085; `shared-types-and-ownership.md` FR-019) | happy |

### 4.2 Gaining Heroic Inspiration

`GainHeroicInspiration` takes the holder `Character` and an
`ExcessInspirationChoice`. It has no parameter for the source of the grant. The
rules of FR-003 to FR-010 live in the pure `ResolveExcessInspiration(holder,
choice)` (FR-038); the use case maps its result (FR-039) and adds telemetry, so
FR-003 to FR-010 are unit tests of that function.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-003 | When the holder is `NotInspired`, gaining MUST return `GainHeroicInspirationReceived` whose holder is `Inspired`, with the one `HeroicInspirationGained` event of `GrantHeroicInspiration()` carrying the holder's id. | SRD 5.2.1, "Heroic Inspiration", p. 8 (Gaining Heroic Inspiration); "Rules Glossary", p. 183 | happy |
| FR-004 | When the holder is `Inspired` and the choice is `LoseExcessInspiration`, gaining MUST return `GainHeroicInspirationExcessLost` whose holder equals the input holder, with no events (as `character-aggregate.md` FR-090; D-11). | SRD 5.2.1, "Rules Glossary", "Heroic Inspiration", p. 183 ("it's lost unless you give it"); p. 8 ("Only One at a Time") | unhappy (rule interaction) |
| FR-005 | When the holder is `Inspired` and the choice is `GiveExcessInspirationTo` a different character whose state is `NotInspired`, gaining MUST return `GainHeroicInspirationExcessGiven` with the holder unchanged, the recipient `Inspired` (via the recipient's `GrantHeroicInspiration()`), and the recipient's one `HeroicInspirationGained` event, carrying the recipient's id. The holder emits no event. | SRD 5.2.1, "Heroic Inspiration", "Only One at a Time", p. 8 ("you can give it to a player character in your group who lacks it"); "Rules Glossary", p. 183 | happy |
| FR-006 | When the holder is `Inspired` and the recipient is `Inspired`, gaining MUST return `GainHeroicInspirationRejected(RecipientAlreadyInspired)`, with no state change and no events. The holder MAY resubmit with `LoseExcessInspiration`. | SRD 5.2.1, "Rules Glossary", p. 183 ("a player character who lacks it"); p. 8 | unhappy (state error) |
| FR-007 | When the holder is `Inspired` and the recipient's id equals the holder's id, gaining MUST return `GainHeroicInspirationRejected(RecipientIsHolder)`, with no state change and no events. | SRD 5.2.1, "Rules Glossary", p. 183 (the holder has it) | unhappy (invalid input) |
| FR-008 | When both conditions of FR-006 and FR-007 hold, the reason MUST be `RecipientIsHolder` (check order: `RecipientIsHolder`, then `RecipientAlreadyInspired`). | n/a (package design; deterministic result) | unhappy (rule interaction) |
| FR-009 | When the holder is `NotInspired`, the `ExcessInspirationChoice` MUST NOT be consulted: a `GiveExcessInspirationTo` naming the holder, or an `Inspired` recipient, MUST still return `GainHeroicInspirationReceived`, with the recipient untouched and no event for the recipient. | SRD 5.2.1, "Rules Glossary", p. 183 (the choice arises only when "you already have it"); p. 8 | unhappy (rule interaction) |
| FR-010 | After gaining, the number of `Inspired` characters among the holder and the recipient MUST rise by one for G1 to G4 and G6, and MUST be unchanged for G5 and G7 to G9. In no case may it rise by more than one. | SRD 5.2.1, "Heroic Inspiration", "Only One at a Time", p. 8 | unhappy (rule interaction) |
| FR-011 | A rejected gain MUST leave the supplied `Character` values unchanged and produce no events; an accepted gain MUST NOT mutate the supplied values. The `events` list of every gain variant that carries one (`Received`, `ExcessLost`, `ExcessGiven`, of both `ExcessResolution` and `GainHeroicInspirationResult`), including the empty list of `ExcessLost`, MUST be defensively copied. A `Rejected` variant has no `events` member. | n/a (architecture.md section 3, immutability) | unhappy (immutability) |

Gaining cases (FR-003 to FR-010). Holder H, recipient R (a different character
unless stated). "Change" is the change in the count of `Inspired` among H and R.

| Label | Holder | Choice | Recipient | Result variant | Holder after | Recipient after | Change | Events | Source | Path |
|---|---|---|---|---|---|---|---|---|---|---|
| G1 not held, lose | `NotInspired` | `LoseExcessInspiration` | n/a | Received | `Inspired` | n/a | +1 | `HeroicInspirationGained(H)` | p. 183 | happy |
| G2 not held, give (choice not consulted) | `NotInspired` | give to R | `NotInspired` | Received | `Inspired` | `NotInspired` (untouched) | +1 | `HeroicInspirationGained(H)` | p. 183 | unhappy (rule interaction) |
| G3 not held, give to holder's own id | `NotInspired` | give to H | `NotInspired` | Received | `Inspired` | n/a | +1 | `HeroicInspirationGained(H)` | p. 183 | unhappy (rule interaction) |
| G4 not held, recipient already inspired | `NotInspired` | give to R | `Inspired` | Received | `Inspired` | `Inspired` (untouched) | +1 (H 0 to 1, R unchanged) | `HeroicInspirationGained(H)` | p. 8 | unhappy (rule interaction) |
| G5 held, lose excess | `Inspired` | `LoseExcessInspiration` | n/a | ExcessLost | `Inspired` | n/a | 0 | none | p. 183 | unhappy (rule interaction) |
| G6 held, give to one who lacks it | `Inspired` | give to R | `NotInspired` | ExcessGiven | `Inspired` | `Inspired` | +1 | `HeroicInspirationGained(R)` | p. 8, p. 183 | happy |
| G7 held, recipient already inspired | `Inspired` | give to R | `Inspired` | Rejected(`RecipientAlreadyInspired`) | `Inspired` | `Inspired` | 0 | none | p. 183 | unhappy (state error) |
| G8 held, give to self | `Inspired` | give to H | `NotInspired` | Rejected(`RecipientIsHolder`) | `Inspired` | n/a | 0 | none | p. 183 | unhappy (invalid input) |
| G9 held, give to self, both conditions | `Inspired` | give to H | `Inspired` | Rejected(`RecipientIsHolder`) | `Inspired` | n/a | 0 | none | n/a (check order) | unhappy (rule interaction) |

### 4.3 Expending Heroic Inspiration

`ExpendHeroicInspiration` takes the `Character` and a `RerollTarget`. It calls
`ExpendHeroicInspiration()`, then rolls exactly one die through `dice.Roller`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-012 | When the character is `Inspired`, expending MUST return `ExpendHeroicInspirationRerolled` whose character is `NotInspired`, with the one `HeroicInspirationExpended` event of `ExpendHeroicInspiration()` (character id only), and a `RerolledDie` carrying the target's die, position and the new value. | SRD 5.2.1, "Rules Glossary", "Heroic Inspiration", p. 183 ("you can expend it"); "Heroic Inspiration", p. 8 | happy |
| FR-013 | When the character is `NotInspired`, expending MUST return `ExpendHeroicInspirationRejected` carrying the aggregate's `HeroicInspirationNotAvailable`, MUST NOT call the `dice.Roller`, MUST produce no event, and MUST leave the character unchanged. | SRD 5.2.1, "Rules Glossary", p. 183 | unhappy (state error) |
| FR-014 | Expending MUST make exactly one request to the `dice.Roller`, for one die of the target's `die` (`dice.NewExpression(1, die)`), through `dice.Roll`, and MUST return that face as `RerolledDie.NewValue`. | SRD 5.2.1, "Heroic Inspiration", p. 8 ("reroll any die"); n/a (`dice-and-rolls.md` section 4.5) | happy |
| FR-015 | The result MUST carry the die, the position and the new value, and MUST NOT carry or offer the original value or any choice between the two. The command MUST NOT carry the original value. (That the new roll must be used is stated in the godoc of `RerolledDie`; only its existence is machine-checked, by the exported-comment lint rule; the content is reviewed.) | SRD 5.2.1, "Heroic Inspiration", p. 8 ("you must use the new roll"); "Rules Glossary", p. 183 | happy |
| FR-016 | The new value MUST be returned unchanged at both extremes of the die: a roller serving 1 gives `NewValue` 1, and one serving the highest face gives that face. A low face is never replaced or rejected. | SRD 5.2.1, "Heroic Inspiration", p. 8 | unhappy (boundary) |
| FR-017 | Expending MUST work for any die, not only the d20: every `Die` named in the case table below MUST be rerolled with a request for one die of that size. | SRD 5.2.1, "Rules Glossary", "Heroic Inspiration", p. 183 ("reroll any die") | happy |
| FR-018 | When the target has `DiceRolled` of 2 (Advantage or Disadvantage), expending MUST reroll only the die at `position` and make one request for one die, never two. The use case MUST NOT choose the position: the result's `position` MUST equal the command's, for 0 or 1. It is indifferent to which of Advantage or Disadvantage applied (FR-027). | SRD 5.2.1, "Advantage/Disadvantage", "Interactions with Rerolls", p. 8 ("reroll or replace only one die, not both. You choose which one") | unhappy (rule interaction) |
| FR-019 | A command MUST carry exactly one `RerollTarget`, and a single expenditure MUST make one request for one die whatever `DiceRolled` is: `DiceRolled` 8 still produces one request. No operation rerolls several dice for one use. | SRD 5.2.1, "Heroic Inspiration", p. 8 ("reroll any die"; singular) | unhappy (boundary) |
| FR-020 | `NewRerollTarget` MUST return a `*RerollTargetError` (matching `srderr.ErrInvalidInput`) carrying the argument name and invalid value for: a `DiceRolled` below 1 or above `dice.MaxCount` (`dice-and-rolls.md`), and a `position` below 0 or at or above `DiceRolled`. `position` 0 with `DiceRolled` 1, and the last index, MUST be accepted. | n/a (package design; testing.md section 4, boundaries; SRD 5.2.1, p. 183 "any die immediately after rolling it") | unhappy (boundary) |
| FR-021 | `NewRerollTarget(die, diceRolled, position)` MUST NOT panic for any ints and any `dice.Die`, including the zero `Die`. It returns a `*RerollTargetError` (`errors.As`-able, `errors.Is` `srderr.ErrInvalidInput`) if and only if FR-020 rejects the arguments, otherwise an equal `RerollTarget` and a nil error. `RerollTargetError.Reason` is `RerollTargetDiceRolledOutOfRange` or `RerollTargetPositionOutOfRange` (check order: `diceRolled`, then `position`; FR-033). `MustRerollTarget` panics with that error for the same arguments (`errors-and-results.md` FR-026). | n/a (`architecture.md` section 6; untrusted host input) | unhappy (invalid input) |
| FR-022 | When `dice.Roll` returns a `*srderr.PortError` for `dice.Port` (the roller failed or panicked), expending MUST return it unchanged as the error, with a zero result, under a recording and the zero-value no-op `telemetry.Telemetry` alike. The caller's `Inspired` character is untouched: no use is spent. | n/a (`dice-and-rolls.md` FR-052; `architecture.md` section 4) | unhappy (failure) |
| FR-023 | When the `dice.Roller` returns a result for a different expression, expending MUST return the `*srderr.PortError` matching `srderr.ErrContractViolation` that `dice.Roll` produces, and spend nothing. A roller panic is recovered by `dice.Roll` into a `*srderr.PortError` matching `srderr.ErrPortPanic`; it is not re-panicked. | n/a (`dice-and-rolls.md` FR-051, FR-053) | unhappy (failure) |
| FR-024 | Feeding the `NotInspired` character of an accepted expenditure into a second `ExpendHeroicInspiration` MUST return `ExpendHeroicInspirationRejected` and MUST NOT call the roller: one Heroic Inspiration allows one reroll. | SRD 5.2.1, "Heroic Inspiration", "Only One at a Time", p. 8; "Rules Glossary", p. 183 | unhappy (state error) |
| FR-025 | Expending MUST be deterministic and read no clock: the same command and a `dicetest.FixedRoller` serving the same values give equal results. | n/a (testing.md section 4, determinism) | unhappy (determinism) |
| FR-026 | Expending MUST NOT mutate the command, its `RerollTarget` or the supplied `Character`; the `events` list of `ExpendHeroicInspirationRerolled` (the only variant with events) MUST be defensively copied. | n/a (architecture.md section 4; testing.md section 4, immutability) | unhappy (immutability) |
| FR-027 | Gaining and expending MUST NOT be affected by Advantage or Disadvantage state of a roll except through `RerollTarget`: the `d20` roll mode is not an input of either use case. | SRD 5.2.1, "Heroic Inspiration", p. 8 | unhappy (rule interaction) |
| FR-033 | `RerollTargetReason` MUST be a typed enum with a `String()` method and exactly `RerollTargetDiceRolledOutOfRange` and `RerollTargetPositionOutOfRange`; `RerollTargetError` carries one `Reason`. There is no creation result type: `NewRerollTarget` returns `(RerollTarget, error)` (`errors-and-results.md` FR-035). No generic name is used (`architecture.md` section 4). | n/a (`dice-and-rolls.md` FR-085) | happy |
| FR-034 | The new face MUST be applicable to a d20 roll without translation loss: a host builds `RerollDie(position)` from `RerollTarget.Position()` and calls `Roll.ApplyReroll(decision, rerolled.NewValue())` (`d20-tests.md` FR-059). This spec imports nothing from `d20`. | SRD 5.2.1, "Interactions with Rerolls", p. 8; d20-tests.md FR-012, FR-059 | happy |
| FR-035 | The expend rejection MUST be the aggregate's `HeroicInspirationNotAvailable`; this spec defines no `ExpendHeroicInspirationRejection` enum. The gain rejection is the new `GainHeroicInspirationRejection` typed enum (`RecipientIsHolder`, `RecipientAlreadyInspired`), which has no aggregate counterpart because gaining with a recipient is a host-boundary concept. | n/a (D-11) | unhappy (state error) |

Expending cases (FR-012 to FR-020). `dicetest.FixedRoller` supplies the listed face.
Two-dice rows are labelled by position, not mode: the SRD example (p. 8) rolls
a 3 and an 18 under either Advantage or Disadvantage and lets the holder choose
either die (FR-018, FR-027).

| Label | State | Target (`die`, `DiceRolled`, `position`) | Roller serves | Requests | Result | State after | Source | Path |
|---|---|---|---|---|---|---|---|---|
| E1 d20, single roll | `Inspired` | d20, 1, 0 | 14 | one `1d20` | Rerolled, `NewValue` 14, position 0 | `NotInspired` | p. 8, p. 183 | happy |
| E2 two d20s (`[18, 3]`), reroll the 3 | `Inspired` | d20, 2, 1 | 11 | one `1d20` | Rerolled, `NewValue` 11, position 1 | `NotInspired` | p. 8 ("3 and an 18") | unhappy (rule interaction) |
| E3 two d20s (`[18, 3]`), reroll the 18 | `Inspired` | d20, 2, 0 | 6 | one `1d20` | Rerolled, `NewValue` 6, position 0 | `NotInspired` | p. 8 | unhappy (rule interaction) |
| E4 damage die d6 | `Inspired` | d6, 3, 2 | 4 | one `1d6` | Rerolled, `NewValue` 4, position 2 | `NotInspired` | p. 183 ("any die") | happy |
| E5 damage die d4 | `Inspired` | d4, 1, 0 | 1 | one `1d4` | Rerolled, `NewValue` 1 | `NotInspired` | p. 183 | unhappy (boundary) |
| E6 d8, d10, d12 | `Inspired` | d8 / d10 / d12, 1, 0 | highest face (8 / 10 / 12) | one die of that size | Rerolled, `NewValue` the highest face | `NotInspired` | p. 183 | unhappy (boundary) |
| E7 d100 | `Inspired` | d100, 1, 0 | 100 | one `1d100` | Rerolled, `NewValue` 100 | `NotInspired` | p. 183 | unhappy (boundary) |
| E8 not inspired | `NotInspired` | d20, 1, 0 | none | none | Rejected(`HeroicInspirationNotAvailable`) | `NotInspired` | p. 183 | unhappy (state error) |
| E9 second use | `NotInspired` (from E1) | d20, 1, 0 | none | none | Rejected(`HeroicInspirationNotAvailable`) | `NotInspired` | p. 8 | unhappy (state error) |
| E10 roller fails | `Inspired` | d20, 1, 0 | roller returns an error | one `1d20` | error `*srderr.PortError` returned, zero result | caller's `Inspired` unchanged | n/a | unhappy (failure) |
| E11 position out of range | any | d20, 2, 2 | none | none | `NewRerollTarget` returns `*RerollTargetError` | n/a | n/a | unhappy (boundary) |
| E12 no dice rolled | any | d20, 0, 0 | none | none | `NewRerollTarget` returns `*RerollTargetError` | n/a | n/a | unhappy (boundary) |
| E13 many dice, one rerolled | `Inspired` | d6, 8, 7 | 3 | one `1d6` | Rerolled, `NewValue` 3, position 7 | `NotInspired` | p. 8 | unhappy (boundary) |

### 4.4 Telemetry and boundary behaviour

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-028 | Each use case entry point takes `ctx` first and MUST run inside one span, record one throughput counter increment per call that returns a result or a `*srderr.PortError`, and one duration (section 7), through the injected `telemetry.Telemetry` (option `WithTelemetry`, defaulting to the zero-value no-op; decisions FR-036). A cancelled `ctx` is returned as `ctx.Err()` with no counter increment (`errors-and-results.md` FR-054). The duration is recorded with `telemetry.StartTimer` and `Timer.Stop` with the `srd5e.character.operation` attribute only. Attributes known only after the body (outcome, result, rejection) are set with `Span.SetAttributes` before `Span.End`, which is deferred, and on the counter after the call (a `telemetrytest.Tracer` flags writes after `End`, `observability-ports.md` FR-023). A panic in a telemetry backend propagates unrecovered (`errors-and-results.md` FR-051). Results MUST be identical under the no-op and a recording `telemetry.Telemetry`. | `doc/rules/observability.md` sections 5, 6, 9 | happy |
| FR-029 | Every rejection variant (FR-006, FR-007, FR-008, FR-013, FR-024) MUST be recorded as `srd5e.outcome` `failure` with `srd5e.character.result` `rejected` and `srd5e.character.rejection` on the counter and the span (the duration histogram carries the operation only); the span status MUST NOT be error, with no `error.type` and no log (`errors-and-results.md` FR-043; `character-aggregate.md` FR-072). Every accepted gain or expenditure (including `excess_lost`) MUST be `success`. No other `srd5e.outcome` value is used. The `srd5e.character.result` value follows the variant: `Received` gives `gained`, `ExcessLost` gives `excess_lost`, `ExcessGiven` gives `excess_given`, `ExpendHeroicInspirationRerolled` gives `expended`, every `Rejected` gives `rejected`. | `doc/rules/observability.md` sections 5 to 7; errors-and-results.md FR-043 | unhappy (state error) |
| FR-030 | A `*srderr.PortError` for `dice.Port` leaving `ExpendHeroicInspiration` MUST be recorded exactly once: `Span.RecordError` and error status, one counter increment with `error.type` = `srderr.ErrorType(err)` (`errors-and-results.md` FR-034), `srd5e.outcome` `failure` and no result attribute, and no log. | `doc/rules/observability.md` section 7; `character-aggregate.md` FR-088 | unhappy (failure) |
| FR-036 | The use cases MUST record on the shared `srd5e.character.operations` counter and `srd5e.character.duration` histogram of `character-aggregate.md` (FR-103, D-21: values only, no new attribute, no spec-local metric). The catalogue (section 7) MUST list the values added to existing keys: `srd5e.character.operation` gains `gain` and `expend`; `srd5e.character.result` gains `gained`, `excess_lost`, `excess_given`, `expended` (`rejected` exists); `srd5e.character.rejection` gains `recipient_is_holder`, `recipient_already_inspired` and `not_inspired`. | `doc/observability/signals.md` key-and-value check (`architecture-guardrails.md` FR-048, FR-085) | happy |
| FR-037 | Neither use case MUST call another use case, `ResolveD20Test` or a `D20TestResolver` (decisions FR-044). `GainHeroicInspiration` composes only `GrantHeroicInspiration()` on the holder and, for a give, the recipient; `ExpendHeroicInspiration` composes `ExpendHeroicInspiration()` and one `dice.Roller` call. Each records only its own signals, so a gain then an expend in one flow gives two independent spans and increments, never nested ones. | `doc/rules/architecture.md` section 4; `doc/rules/observability.md` sections 7, 11; decisions FR-044 | unhappy (nested telemetry) |

### 4.5 Domain function and value validation

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-038 | Package `character` MUST contain the pure function `ResolveExcessInspiration(holder Character, choice ExcessInspirationChoice) ExcessResolution`; `ExcessResolution` is a closed variant set (interface with an unexported marker method, documented) with exactly `ExcessResolutionReceived{Holder, Events}`, `ExcessResolutionExcessLost{Holder, Events}`, `ExcessResolutionExcessGiven{Holder, Recipient, Events}` and `ExcessResolutionRejected{Reason GainHeroicInspirationRejection}`. It MUST implement FR-003 to FR-010 exactly, calling `GrantHeroicInspiration()` on the holder and, for a give, the recipient, and changing nothing else. Order: (1) holder `NotInspired` gives `Received` without reading the choice (FR-009); (2) a give naming the holder's id gives `Rejected(RecipientIsHolder)` (FR-007); (3) a give to an `Inspired` recipient gives `Rejected(RecipientAlreadyInspired)` (FR-006); (4) `LoseExcessInspiration` gives `ExcessLost`; (5) otherwise `ExcessGiven`. Step 2 precedes step 3 (FR-008). The function is silent, takes no `ctx`, reads no port and no clock, and is deterministic. | SRD 5.2.1, "Heroic Inspiration", "Only One at a Time", p. 8; "Rules Glossary", p. 183; n/a (package design for the check order) | happy |
| FR-039 | `GainHeroicInspiration` MUST call `ResolveExcessInspiration` once and map each `ExcessResolution` variant to the `GainHeroicInspirationResult` variant of the same name suffix (`Received`, `ExcessLost`, `ExcessGiven`, `Rejected`) with the same holder, recipient, events and reason. It MUST NOT inspect the holder's state, the choice or the recipient, and MUST NOT call `GrantHeroicInspiration()` itself. Its remaining work is telemetry (FR-028 to FR-030). | `doc/rules/architecture.md` section 4 (no rules logic in a use case) | happy |
| FR-040 | `NewRerolledDie(die, position, newValue)` MUST return an error matching `srderr.ErrInvalidInput` (a `*RerolledDieError` carrying the argument name and invalid value) for a `newValue` below 1 or above `die.Sides()`, and for a `position` below 0. Accepted: `newValue` 1 and `die.Sides()`, `position` 0. (No `diceRolled`, so no upper bound for `position`; `RerollTarget` has one, FR-020.) | n/a (testing.md section 4, boundaries; `dice-and-rolls.md` FR-031, FR-032) | unhappy (boundary) |

### 4.6 Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to character-aggregate.md (FR-089) |
| FR-002 | Withdrawn: moved to character-aggregate.md (FR-055, FR-089) |

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

- **No new aggregate, state or event** (FR-031, FR-032; D-12).
- **Domain function:** `ResolveExcessInspiration` (FR-038) in
  `character`, returning the closed `ExcessResolution`
  (`ExcessResolutionReceived`, `ExcessResolutionExcessLost`,
  `ExcessResolutionExcessGiven`, `ExcessResolutionRejected`). It holds the
  check order and the count invariant (FR-008 to FR-010) and only composes the
  aggregate's `GrantHeroicInspiration()`.
- **Value objects (pure, immutable, unexported fields):** `ExcessInspirationChoice` (closed variant set:
  `LoseExcessInspiration`, `GiveExcessInspirationTo{Recipient Character}`),
  `RerollTarget`, `RerolledDie`, built by `NewRerollTarget` and `NewRerolledDie`
  (with panicking `MustRerollTarget`, `MustRerolledDie`). **Rejection:**
  `*RerollTargetError` carrying the typed enum `RerollTargetReason`
  (`RerollTargetDiceRolledOutOfRange`, `RerollTargetPositionOutOfRange`; FR-033).
- **Rejection reasons (closed enums):** `GainHeroicInspirationRejection`;
  expending reuses `HeroicInspirationRejection` (FR-035).
- **Invariants:** `1 <= diceRolled <= dice.MaxCount` and
  `0 <= position < diceRolled` (FR-020); `1 <= newValue <= die.Sides()` and
  `0 <= position` (FR-040).
- **Failures:** rule outcomes are closed results; untrusted target numbers
  return `(T, error)` from `NewX`; the error matches `srderr.ErrInvalidInput`
  and is `errors.As`-able; a roller failure, panic or contract-breaking
  answer is a `*srderr.PortError` for `dice.Port`. No exported API panics on
  untrusted input; panics exist only in `MustX` helpers.

## 6. Interface

Both use cases are synchronous calls taking `ctx` first (`dice.Roller` itself
takes none; gaining touches no port but emits spans). They live in package
`character` (files `gain_heroic_inspiration_usecase.go` and
`expend_heroic_inspiration_usecase.go`), take `telemetry.Telemetry` through the
`WithTelemetry` option defaulting to the zero-value no-op (decisions
FR-036), call no other use case (FR-037), and return a closed
`<UseCase>Result` (interface with an unexported marker method) with
operation-prefixed variants (decisions FR-017). Constructors return concrete
types and each use case declares the one-method `Roller` it consumes
(any `dice.Roller` satisfies it).

| Use case | Collaborators | Command | Result |
|---|---|---|---|
| `GainHeroicInspiration` | `telemetry.Telemetry` | `GainHeroicInspirationCommand{Holder Character, Excess ExcessInspirationChoice}` | `GainHeroicInspirationReceived{Holder, Events}`, `GainHeroicInspirationExcessLost{Holder, Events}`, `GainHeroicInspirationExcessGiven{Holder, Recipient, Events}`, `GainHeroicInspirationRejected{Reason GainHeroicInspirationRejection}` (no `Events`); signature `Gain(ctx, cmd) GainHeroicInspirationResult` |
| `ExpendHeroicInspiration` | `Roller` (`dice.Roller`), `telemetry.Telemetry` | `ExpendHeroicInspirationCommand{Character Character, Target RerollTarget}` | `ExpendHeroicInspirationRerolled{Character, Rerolled RerolledDie, Events}`, `ExpendHeroicInspirationRejected{Reason HeroicInspirationRejection}` (no `Events`); signature `Expend(ctx, cmd) (ExpendHeroicInspirationResult, error)` |

- A `*srderr.PortError` for `dice.Port` is returned as the error (FR-022, FR-023); an expected absence is the
  `HeroicInspirationNotAvailable` rejection value. No content source is used.
- Exports (each once, with a godoc comment, from package `character`; no facade and no re-export package): `ExcessInspirationChoice`,
  `LoseExcessInspiration`, `GiveExcessInspirationTo`, `RerollTarget`,
  `RerollTargetError`, `RerollTargetReason`, `NewRerollTarget`,
  `MustRerollTarget`, `RerolledDie`, `NewRerolledDie`, `ResolveExcessInspiration`,
  `ExcessResolution` (variants `ExcessResolutionReceived`,
  `ExcessResolutionExcessLost`, `ExcessResolutionExcessGiven`,
  `ExcessResolutionRejected`), `GainHeroicInspiration`,
  `GainHeroicInspirationCommand`, `GainHeroicInspirationResult` (variants
  `GainHeroicInspirationReceived`, `GainHeroicInspirationExcessLost`,
  `GainHeroicInspirationExcessGiven`, `GainHeroicInspirationRejected`),
  `GainHeroicInspirationRejection`, `ExpendHeroicInspiration`,
  `ExpendHeroicInspirationCommand`, `ExpendHeroicInspirationResult`,
  `ExpendHeroicInspirationRerolled`, `ExpendHeroicInspirationRejected`. The
  aggregate's types are defined elsewhere in the same package. No generic
  variant name is exported (`architecture.md` section 4). Each closed
  interface's doc comment lists its variants (`errors-and-results.md` FR-044).

## 7. Observability

Domain types are silent. The use cases are instrumented.

What can go wrong, how an operator knows, what is counted and what a consumer
monitors are FR-028 to FR-030 and FR-036: roller failures are span error status
plus `error.type` on the counter; rejections are `failure` with `rejected` and
a rejection value; programmer errors (`MustX` panics) are bugs, propagate and are not recorded.
Ids, the new face and die size are in results and events, not attributes
(cardinality). Monitor any `error.type` increment, a rise of `not_inspired` (a
host offering a reroll without it) or `recipient_already_inspired`, and a
duration regression. No logging. `srd5e.d20_test.*` is unaffected;
`character-aggregate.md` section 7 needs no amendment (FR-103 there allows
operation values).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.operations` (shared) | counter, unit `{operation}` | adds `srd5e.character.operation` values `gain`, `expend`; `srd5e.character.result` values `gained`, `excess_lost`, `excess_given`, `expended`; `srd5e.character.rejection` values `recipient_is_holder`, `recipient_already_inspired`, `not_inspired`; `error.type` = `srderr.ErrorType(err)` (port failures only; result then absent) | Throughput and outcome mix; exposes failing roller adapters | Alert on any `error.type`; watch `failure` share per rejection |
| `srd5e.character.duration` (shared) | histogram, unit `s` | `srd5e.character.operation` only (`gain`, `expend` added) | Latency via `telemetry.StartTimer` (also on failure) | p95 regression; dominated by the roller adapter |
| `srd5e.character.gain_heroic_inspiration` | span | `srd5e.outcome`, `srd5e.character.result`, `srd5e.character.rejection` (rejections only) | Path of one gain | Error status rate above 0 |
| `srd5e.character.expend_heroic_inspiration` | span | same as above | Path of one expenditure, including the roller call | Error status rate above 0 |

Entries feed `doc/observability/signals.md`. `srd5e.outcome` uses only
`success` and `failure` (`shared-types-and-ownership.md` FR-041;
`errors-and-results.md` FR-043).

## 8. Test plan

Gain rules (FR-003 to FR-010) are unit tests of `ResolveExcessInspiration`;
application tests assert only mapping and telemetry. Levels: unit
(`character/*_test.go`), application (`character/*_test.go`),
scenario (`scenarios/`), architecture (`internal/archtest/`). Every test
carries `Requirement:` and `Path:` comments (`testing.md` section 5); case
tables are table-driven, one test per row.

| Id | Level | Notes |
|---|---|---|
| FR-003 to FR-010 | unit | On `ResolveExcessInspiration`: G1 (event carries holder id); G5 (holder equals input, no events); G6 (event carries recipient id, holder unchanged); G7, G8; G9 (check order); G2 to G4 (choice not consulted when not held); count change of G1 to G9 |
| FR-011 | unit, application | Inputs unchanged after accepted and rejected gains; `Rejected` has no `Events` field; the `Events` slice of `Received`, `ExcessLost` (empty) and `ExcessGiven` is a copy (mutating it leaves the value unchanged), for `ExcessResolution` (unit) and `GainHeroicInspirationResult` (application) |
| FR-012 to FR-014, FR-016 to FR-019 | application | E1 (event contents: character id only); E8 (no roller request, no event, rejection is `HeroicInspirationNotAvailable`); E1, E4, E5, E6, E7 (one request for one die of the target's size, through `dice.Roll`, across d4 to d100); E5, E6, E7 (served face also returned unchanged); E2, E3 (position equals command's for 0 and 1); E13 (one request; command accepts one `RerollTarget`) |
| FR-015 | unit, application | Result exposes no original value; command has no original-value parameter (constructor signature exercised); godoc statement left to code review |
| FR-020, FR-021 | unit | Boundary table: `diceRolled` 0, 1, `dice.MaxCount`, `dice.MaxCount + 1`; `position` -1, 0, `diceRolled - 1`, `diceRolled`; E11, E12; `errors.As` a `*RerollTargetError` with its `Reason` and `errors.Is(err, srderr.ErrInvalidInput)`, never message text; `NewRerollTarget` errors iff FR-020 rejects, over a range of ints, with reasons and check order; `MustRerollTarget` panics with that error iff `NewRerollTarget` fails |
| FR-022 to FR-025, FR-027 | application | E10 with a `dicetest.CallbackRoller` that returns an error and one that panics, under recording and zero-value no-op `telemetry.Telemetry` (`errors.Is` reaches the cause, `errors.As` finds the `*srderr.PortError`, zero result); roller returning a different expression gives `srderr.ErrContractViolation`; E9 (feed E1's output back, no further request); repeat call with equal `dicetest.FixedRoller` supplies gives equal results; neither command has a roll-mode parameter |
| FR-026 | unit, application | Command, target and character unchanged; the `Events` slice of `Rerolled` is a copy; `Rejected` has no `Events` field |
| FR-028 to FR-030 | application | Cancelled `ctx`: `ctx.Err()` returned, no counter increment; a panicking `telemetrytest` double propagates and `End` still runs; span attributes set before `End` (`telemetrytest.Tracer.Violations()` empty); `telemetry.StartTimer` used around the call; recording and no-op give equal results; names and keys pinned (shared counter and histogram, no spec-local metric). Rejections G7, G8, G9, E8, E9: `failure`, `rejected`, rejection string, span not in error, no `error.type`, no log; G1 to G6 and E1 assert `success` and the FR-029 variant-to-result mapping; no other `srd5e.outcome` value. E10 with `telemetrytest.Recorder`: span error status once, one increment with `error.type` = `srderr.ErrorType(err)`, `failure`, no result attribute, no log |
| FR-031, FR-032, FR-039 | application | Results equal direct `Character` transition calls (differential over G1 to G9 and E1, E8) and map each `ExcessResolution` variant to the same-named result variant with equal fields; use case source has no branch on holder state, choice or recipient; events are the aggregate's (architecture check: no event type declared here) |
| FR-033, FR-038 | unit | Exhaustive type switch over `ExcessResolution` (no `default`; `gochecksumtype` lint) and `RerollTargetReason`; table G1 to G9 on `ResolveExcessInspiration` (variants, event contents, check order, choice not read when not held); function is pure |
| FR-034 | scenario | Composition through the public API: dice `[18, 3]`, `RerollDie(1)`, `ApplyReroll` with the face from the use case |
| FR-035 | unit, application | Expend rejection is `HeroicInspirationNotAvailable`; gain rejection enum has exactly two values |
| FR-036 | architecture | Catalogue key and value strings in `signals.md` match the constants |
| FR-037 | application | No use case among constructor parameters; `ExpendHeroicInspiration` records one span, one increment and one duration; a gain then an expend records two separate spans, no nesting |
| FR-040 | unit | Boundary table: `newValue` 0, 1, `die.Sides()`, `die.Sides() + 1`; `position` -1, 0; `errors.Is` `srderr.ErrInvalidInput` and `errors.As` the typed error with the argument name and value |
| (all) | scenario | Through the public packages only (`package scenarios_test`): gain, roll two d20s showing 18 and 3, expend to reroll the 3 (position 1), try again and be refused; a second gain while holding it is given to a party member |
| (guardrail) | architecture | The `internal/archtest` allow-list, context-map and domain-purity checks apply; no new rule |

- Doubles (decisions FR-038; none defined here): `dicetest.FixedRoller`,
  `dicetest.CallbackRoller` (throwing and wrong-expression rollers,
  `dice-and-rolls.md` section 4.6), `telemetrytest.Recorder`. No new port, double
  or contract suite.
- Mutation testing (`testing.md` section 8): `RerollTarget`, `RerolledDie` and
  `ResolveExcessInspiration`, `gomu` incremental, threshold 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (never on the host); targets are the
  bounds of FR-020 and FR-040, the check order of FR-008, the FR-009 branch and
  the FR-010 invariant. The `*_usecase.go` files SHOULD be mutation tested for FR-039.

## 9. Open questions and decisions

No open questions. Settled decisions:

- D-1: state ownership is `character-aggregate.md` (`shared-types-and-ownership.md` FR-010).
- D-2: `d20` applies the face (`shared-types-and-ownership.md` FR-002; FR-034).
- D-3: `srd5e.outcome` is `success` or `failure` only (`shared-types-and-ownership.md` FR-041).
- D-4: the Human trait text (p. 86) governs over the p. 8 sidebar; the host calls gain after the rest.
- D-5: `class-fighter.md` FR-135 returns `HeroicWarriorNotAvailable(AlreadyHasInspiration)` when held, so gain is never called then.
- D-6: a give to a recipient who has it is rejected, not silently lost (FR-006).
- D-7: a give is modelled only when a second instance would be gained (p. 8, p. 183).
- D-8: "immediately" is not enforced; the package has no clock (`architecture.md` section 3).
- D-9: no refusal when a die was already replaced by another effect; each effect replaces one die and no state is kept (`d20-tests.md` FR-013).
- D-10: one die per use (FR-019).
- D-11: no excess-lost or given event is added; the aggregate is unchanged (FR-004, FR-005).
- D-12: this spec owns the use cases (`shared-types-and-ownership.md` FR-048); no `HeroicInspirationRules` service (FR-031, FR-038).
- D-13: `RerollTarget` rather than `RerollDecision`, which is d20-only and carries no die size (FR-034).
- D-14: the new face is drawn in `character` because it may be any die; applying it to a d20 stays in `d20` (FR-059 there).

## 10. Changelog impact

`CHANGELOG.md`, Unreleased, **Added**: the `GainHeroicInspiration` and
`ExpendHeroicInspiration` use cases with their commands and results, the
`ExcessInspirationChoice` and `RerollTarget` inputs, `ResolveExcessInspiration`
with `ExcessResolution`, and the telemetry in section 7, recorded on the
shared `srd5e.character.operations` and `srd5e.character.duration` (value
additions to `srd5e.character.operation`, `srd5e.character.result` and
`srd5e.character.rejection`, FR-036). Not **BREAKING**. No `character-aggregate.md` type or event changes
(D-11); no edit to `class-fighter.md`.
