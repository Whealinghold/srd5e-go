# Grappling

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `conditions.md`, `d20-tests.md`, `resolve-d20-test-use-case.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `content-schema-and-decoder.md`, `ability-scores.md`, `derived-stats.md`, `observability-ports.md`, `errors-and-results.md` |

Dependency notes (prose, not dependencies): this spec uses `CombatantID`, `Event` and the telemetry types (`CombatSignals`, `CombatTelemetry`, `CombatResultClass`) of `combat-foundations.md`; `Grappled`, `ConditionSet.Remove`, `ConditionRemoved`, `ConditionEffects.IsIncapacitated`, `ConditionActionAvailability` and `ConditionPermission` of `conditions.md`; the pure `d20.ResolveCheck` and `d20.FailWithoutRoll` functions of `d20-tests.md`; the attempt types `CheckAttempt`, `RollAttempt` and `AutomaticFailure` of `resolve-d20-test-use-case.md` (this spec never calls `d20.Resolver.Resolve`, shared-types-and-ownership.md FR-044); `CreatureSize` and `Skill` from package `content`. Context order is shared-types-and-ownership.md FR-015 and table 4.11 (rank 23, below `attacks.md`, rank 24, which imports `GrappleRules` and `HandAvailability`, its FR-128). The specs that use this one (attacks, initiative-and-actions, encounter-aggregate, combat-action-resolution) are not dependencies.

## 1. Purpose and scope

The Grappling rules of SRD 5.2.1 ("Grappling", p. 182), the Grapple option of the
Unarmed Strike (p. 190) and the grappler-side behaviour of the Grappled
condition ("Grappled [Condition]", p. 182).

**Ownership** (shared-types-and-ownership.md FR-029, FR-048: the topic-specific spec owns the mechanics). This spec owns every grapple **mechanic**: the grapple option's rules (DC, the saves the target may choose, size and free hand); the record of who holds whom with which part (`Grapples`); the escape check and its two options; ending a grapple (release, escape, an Incapacitated grappler, a target beyond the range) and the hold rule that decides the last two with its precedence; the extra movement cost of dragging or carrying; and the events `GrappleEnded`, `GrappleStarted`, `GrappleRecordDropped` with `GrappleEndCause` and `GrappleHold`.

`conditions.md` keeps only the Grappled **condition entry** (Speed 0, Disadvantage on attack rolls against any target other than the grappler) and the plain `ConditionSet.Remove`. `attacks.md` keeps only the **Unarmed Strike attack** (option choice, attack roll, the target's save with its Cover bonus, the result it returns, including the size predicate Shove also uses and the order of impossibility reasons that FR-096 follows) and asks `GrappleRules` for the grapple option's DC and pre-checks. The DC, option rule and size and free-hand rule it once held are withdrawn there in favour of this spec's FR-094 to FR-096. `encounter-aggregate.md` ends a grapple through this spec's `GrappleRules.Hold` (FR-087) and `GrappleRules.EndOnSet` (FR-089).

Withdrawn elsewhere in favour of this spec: `conditions.md` (FR-063, FR-132 to FR-138 (withdrawn): escape options, escape, release, hold rule, `GrappleEnded`, extra movement cost; D-6), `rules-glossary-coverage.md` Q-4 and GL-072.

It covers:

- `Grapple` (grappler, target, body part, escape DC, range) and the immutable `Grapples` collection with the SRD invariants: one grapple per part, and a part holding a creature cannot target another;
- the stateless domain service `GrappleRules` (DC and pre-checks, escape options, hold rule, end operation on a `ConditionSet`, movement cost);
- recording a successful grapple (an Unarmed Strike, or a monster's special attack the host has resolved) and naming the `Grappled` condition the host applies;
- removing the record when a grapple ends and handing the host a `GrappleEnding` for `GrappleRules.EndOnSet`, which emits the one `GrappleEnded`;
- five use cases (`BeginGrapple`, `ReleaseGrapple`, `EscapeGrappleFromRecord`, `ReviewGrapples`, `DropGrappleRecord`) with telemetry through `CombatTelemetry`.

State in, new state and events out (architecture section 4): nothing here holds hidden state. The `Grapples` value is the host's to store; the record is the host's and the condition is the encounter's, and `GrappleRecordDropped` (FR-060) keeps the record from outliving the condition.

**Out of scope:**

- **The Unarmed Strike attack**: roll, Cover and the save belong to `attacks.md`. This spec consumes a grapple that succeeded and never re-checks size or the attack's free hand (it does check that the named part is not already holding someone, FR-024). A monster's special grapple attack is resolved by the host from the stat block (`content-monsters.md`).
- **The effects of the Grappled condition on its bearer**: `conditions.md` FR-060 to FR-062 and FR-128.
- **Applying and removing the Grappled condition.** `Grapples` never touches a `ConditionSet`; the host applies `Grapple.Condition` and removes it with `GrappleRules.EndOnSet` (FR-089). A plain `ConditionSet.Remove` emits no `GrappleEnded`.
- **Spending the escape action in the turn tracker** (Q-4) and **positions and distance** (the engine models no map; the host reports distance, Q-2).
- **What occupies a creature's hands** and how many parts a creature has: the host names the part and states whether a hand is free (`HandAvailability`, FR-096). `IsHolding` (FR-018) only says whether the part already holds a grappled creature.
- **The Grappler feat** ("General Feats", p. 87): `content-species-backgrounds-feats.md`.
- **Removing a combatant** (death, fleeing): the SRD gives no rule beyond the endings above (Q-10).
- **Anything not in SRD 5.2.1.** No rule that looked like core fifth-edition material and was missing from the PDF was needed.

## 2. Source

- SRD 5.2.1, "Grappling", p. 182 (the whole entry: "Grappled Condition", "One Grapple per Hand", "Ending a Grapple").
- SRD 5.2.1, "Grappled [Condition]", p. 182 ("Speed 0", "Attacks Affected", "Movable").
- SRD 5.2.1, "Unarmed Strike", p. 190 (the Grapple option: the target's Strength or Dexterity saving throw, the DC "equals 8 plus your Strength modifier and Proficiency Bonus", "any escape attempts" use the same DC, "no more than one size larger than you", "a hand free to grab it", a target "within 5 feet of you").
- SRD 5.2.1, "Skills", Skills table, p. 9 (Acrobatics is Dexterity, Athletics is Strength).
- SRD 5.2.1, "D20 Tests", p. 6 (a total equal to or above the DC succeeds; two d20s with Advantage or Disadvantage).
- SRD 5.2.1, "Advantage/Disadvantage", p. 8 (starts at the foot of p. 7; both together cancel and one d20 is rolled).
- SRD 5.2.1, "Incapacitated [Condition]", "Inactive", p. 184 (through `conditions.md` FR-064).
- SRD 5.2.1, "Creature Size", p. 14; "Dead", p. 180 and "Unconscious [Condition]", p. 191 (Q-10); "Condition", p. 179 ("doesn't stack with itself"; FR-014, FR-025).
- Package design: `doc/rules/architecture.md` sections 2, 3, 4, 6; `doc/rules/observability.md` sections 2 to 7; `doc/rules/testing.md` section 4.

Printed pages equal PDF pages (checked on pages 182 and 190 again in this revision; earlier on pages 6, 8, 9, 180 and 191).

## 3. Vocabulary

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` section 3).

| SRD term | Type | Notes |
|---|---|---|
| Grapple, "grapple another creature" | `Grapple` | One creature holding another with one part (p. 182). Grappler and target are `CombatantID`s (`combat-foundations.md`). |
| Grappled condition | `Grappled(grappler)`, `ConditionKind.Grappled` | Entry in `conditions.md`; imported, never redefined. |
| "a hand", "a tentacle, a maw, or another body part" | `GraspingPart` (`GraspingHand(index)`, `GraspingBodyPart(name)`) | The engine does not model anatomy; the host names the part. |
| "a hand free to grab it" | `HandAvailability` (`handFree`, `noHandFree`) | Host input to `GrappleRules.Possibility` (FR-096). |
| escape DC, "any escape attempts" | `Grapple.EscapeDC` | For an Unarmed Strike `GrappleRules.GrappleDC` (FR-094); for a monster, the stat block. |
| "the grapple's range" | `Grapple.RangeFeet` | Distance beyond which the grapple ends (p. 182); `Grapple.UnarmedStrikeRangeFeet` is 5 (FR-011). |
| "the distance between the Grappled target and the grappler" | host input (`GrappleObservation.DistanceFeet`) | Not held here (Q-2). |
| Incapacitated, exceeds the range | `GrappleHold` (`holds`, `grapplerIncapacitated`, `outOfRange`) | FR-087. |
| Strength (Athletics), Dexterity (Acrobatics) check | `GrappleEscapeOption` | FR-035; ability from `Skill.Ability`. |
| "release the target at any time (no action required)" | `Grapples.Release`, `ReleaseGrapple` | No action. |
| "Ending a Grapple" | `GrappleEnded`, `GrappleEndCause` (`escaped`, `grapplerIncapacitated`, `outOfRange`, `released`) | FR-085, FR-086; produced only by `GrappleRules.EndOnSet`. |
| "Movable", "extra foot" | `GrappleRules.MovementCost`, `Grapples.ExtraMovementCostPerFoot` | Extra cost per foot moved while dragging or carrying. |
| Grapple option's size and hand rule | `GrapplePossibility`, `GrappleNotPossibleReason` (`targetTooLarge`, `noFreeHand`) | FR-096. |
| (state of a part) | `PartState` (`holding`, `notHoldingGrapple`) | Whether the part holds a grappled creature, not whether the SRD's "hand free" is met. |

Package names that are not SRD terms: `GrappleRules`, `Grapples`, `GraspingPart`, `GrappleObservation`, `GrappleEnding`, `GrappleEvent`, `GrappleEndOnSetResult`, `GrappleEscapeCheck`, `GrappleRejection`, `GrappleProblem`, `PartState`, `HandAvailability`, `GrapplePossibility`, the use cases and their results. There is no `EscapeGrappleResult`: the use case is `EscapeGrappleFromRecord`, so the package keeps one plain name per concept (`combat-foundations.md` FR-024).

## 4. Requirements

Stable ids. Never renumber. "Part" means a `GraspingPart`. Distances and ranges
are whole feet. In rows, "set" is a `Grapples` value; an event name without
comment is a domain event of this spec.

### 4.1 Parts and the grapple record

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `NewGraspingHand(index)` MUST accept any index of 0 or more. | SRD 5.2.1, "Grappling", "One Grapple per Hand", p. 182 | happy |
| FR-002 | `NewGraspingHand` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a negative index. | n/a (architecture section 6) | unhappy (boundary) |
| FR-003 | `NewGraspingBodyPart(name)` MUST accept a name that is non-empty and has no leading or trailing whitespace (the stat block's own word, for example a tentacle or a maw). | SRD 5.2.1, "Grappling", p. 182 ("a tentacle, a maw, or another body part") | happy |
| FR-004 | `NewGraspingBodyPart` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for an empty name, a name of only whitespace, and a name with leading or trailing whitespace. | n/a (architecture section 6) | unhappy (invalid input) |
| FR-005 | `GraspingPart` values MUST be comparable with `==` and equal when they are the same variant with the same index or name; the hands of index 0 and 1 MUST differ, and a `GraspingHand` MUST NOT equal a `GraspingBodyPart`. | SRD 5.2.1, "Grappling", p. 182 | unhappy (boundary) |
| FR-006 | `Grapple` MUST carry grappler, target, part, escape DC (`DifficultyClass`) and range in feet, and MUST compare by value on all five fields (equal when all are equal, unequal when any one differs). | SRD 5.2.1, "Grappling", p. 182 | happy |
| FR-007 | `NewGrapple` MUST return an error when grappler and target are the same `CombatantID`: a `*GrappleProblemError` carrying `GrappleProblem.SameGrapplerAndTarget` that satisfies `errors.Is(err, srderr.ErrInvalidInput)` (FR-083). | SRD 5.2.1, "Grappling" ("grapple another creature"), p. 182 | unhappy (invalid input) |
| FR-008 | `Grapple` MUST accept a range of 0 or more feet; `NewGrapple` MUST return a `*GrappleProblemError` carrying `GrappleProblem.NegativeRange` (satisfying `errors.Is(err, srderr.ErrInvalidInput)`) for a negative range (FR-083). | n/a (architecture section 6); "Grappling", p. 182 | unhappy (boundary) |
| FR-009 | `Grapple` MUST hold the escape DC exactly as given and MUST NOT bound it (the SRD bounds no DC). | SRD 5.2.1, "Grappling", p. 182; d20-tests.md D-02 (`DifficultyClass` is unbounded) | unhappy (boundary) |
| FR-010 | `Grapple.Condition` MUST return `Grappled(grappler)` of `conditions.md`, so the condition's source is the grappler and no second grappler type exists. | SRD 5.2.1, "Grappling" ("Grappled Condition"), p. 182; "Grappled [Condition]", p. 182 | happy |
| FR-011 | `Grapple.UnarmedStrikeRangeFeet` MUST equal 5. Package design (interpretation): the SRD never defines "the grapple's range" for an Unarmed Strike; p. 190 only says the attack's target is "within 5 feet of you". The constant is a default the host may override by passing another `RangeFeet`. | Package design, interpreting SRD 5.2.1, "Unarmed Strike", p. 190 ("a target within 5 feet of you") and "Grappling", p. 182 ("the grapple's range") | happy |

### 4.2 The `Grapples` collection

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-012 | `Grapples.Empty` MUST hold no grapple; on it every part MUST be `PartState.NotHoldingGrapple` (FR-018), no creature MUST be grappled (FR-020) and `HeldBy` MUST be empty (FR-021). | SRD 5.2.1, "Grappling", p. 182 | unhappy (empty and minimal) |
| FR-013 | `NewGrapples` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when two grapples have the same grappler and the same part ("whatever part a grappler uses, it can grapple only one creature at a time with that part"). | SRD 5.2.1, "Grappling", "One Grapple per Hand", p. 182 | unhappy (invalid input) |
| FR-014 | `NewGrapples` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when two grapples have the same target (a condition "doesn't stack with itself"; one source is kept, `conditions.md` FR-019). | SRD 5.2.1, "Condition", p. 179; package design (`conditions.md` D-9) | unhappy (invalid input) |
| FR-015 | The `Grapples` constructor MUST accept several grapples with the same grappler on different parts and different targets, and several grapples with different grapplers on equal parts (a part belongs to its creature). | SRD 5.2.1, "Grappling", p. 182 | unhappy (boundary) |
| FR-016 | `Grapples` MUST copy its input on construction and return copies from its accessors; mutating the input slice after construction or a returned slice MUST NOT change the value. | `doc/rules/architecture.md` section 4 (handoffs) | unhappy (immutability) |
| FR-017 | `Grapples.Equal` MUST report equal when two values hold the same grapples regardless of insertion order, and unequal when any grapple differs (`Grapples` holds a slice, so it is compared with `Equal`, not `==`). Iteration MUST follow the order in which grapples were begun. | `doc/rules/architecture.md` section 3 (value objects, determinism) | happy |
| FR-018 | `IsHolding(grappler, part)` MUST return `PartState.Holding` when a grapple has that grappler and part, otherwise `PartState.NotHoldingGrapple`; the same part value on a different grappler MUST be `notHoldingGrapple`. It does not say a hand is free in the SRD's sense (FR-096 and section 1). | SRD 5.2.1, "Grappling", "One Grapple per Hand" ("it can grapple only one creature at a time with that part"), p. 182 | happy |
| FR-019 | `CanTarget(grappler, part, target)` MUST return `ConditionPermission.Forbidden` when the part is holding a creature other than `target`, and `ConditionPermission.Allowed` when the part is not holding a grapple or is holding `target`. | SRD 5.2.1, "Grappling", p. 182 ("the grappler can't use that part to target another creature unless it ends the grapple") | unhappy (rule interaction) |
| FR-020 | `GrappleOf(target)` MUST return the `Grapple` holding that target, or `NotInGrapple` when none does (a closed lookup, never a nil pointer). | SRD 5.2.1, "Grappling", p. 182 | happy |
| FR-021 | `HeldBy(grappler)` MUST return a fresh slice of the grapples that grappler holds, in the order begun, and an empty list for a grappler holding none. | SRD 5.2.1, "Grappling", p. 182 | happy |

### 4.3 Beginning a grapple

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-022 | `Begin(grapple)` on a set where the part is not holding a grapple and the target is not held MUST return `GrappleTransitionApplied` with a new set containing the grapple, exactly one event `GrappleStarted(grappler, target, part)` and no `GrappleEnding`. | SRD 5.2.1, "Grappling" ("Grappled Condition"), p. 182 | happy |
| FR-023 | `Begin` MUST allow a grappler that already holds a creature with one part to hold another creature with a different part. | SRD 5.2.1, "Grappling", "One Grapple per Hand", p. 182 | unhappy (rule interaction) |
| FR-024 | `Begin` on a part that is already holding a creature (the same target or another) MUST return `GrappleTransitionRejected` with rejection `GrappleRejection.PartInUse`. | SRD 5.2.1, "Grappling", "One Grapple per Hand", p. 182 | unhappy (state error) |
| FR-025 | `Begin` on a target already held by any grappler (including the same grappler through another part) MUST return `GrappleTransitionRejected` with rejection `GrappleRejection.TargetAlreadyGrappled`. | SRD 5.2.1, "Condition", p. 179; package design (`conditions.md` D-9, Q-6) | unhappy (state error) |
| FR-026 | When both FR-024 and FR-025 apply, the reason MUST be `partInUse`. | package design (the SRD gives no order) | unhappy (rule interaction) |
| FR-027 | `Begin` MUST NOT depend on the grappler's own conditions, on sizes or on the target's distance: a grappler that is itself held in a grapple MAY begin one. Whether the attack that produced it was possible is FR-096 and `attacks.md`'s rule (FR-072). | SRD 5.2.1, "Grappling", p. 182 (no such restriction stated) | unhappy (rule interaction) |
| FR-028 | `Begin` MUST NOT change its input set, whether it applies or rejects; a rejection MUST carry no events. | `doc/rules/architecture.md` section 3 | unhappy (immutability) |

### 4.4 Releasing

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-029 | `Release(grappler, target)` where that grappler holds that target MUST return `GrappleTransitionApplied` with the grapple removed, no event, and one `GrappleEnding(grapple, GrappleEndCause.Released)` for the host to pass to `GrappleRules.EndOnSet(set, target, GrappleEndCause.Released)` (FR-089), which emits `ConditionRemoved` then the one `GrappleEnded`. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182 | happy |
| FR-030 | After a release the grapple's part MUST be `PartState.NotHoldingGrapple` and the target MUST be `NotInGrapple`, so the same part MAY `Begin` another grapple. | SRD 5.2.1, "Grappling", p. 182 | unhappy (rule interaction) |
| FR-031 | `Release` where the named grappler does not hold the named target (the target is free, or held by another grappler) MUST return `GrappleTransitionRejected` with rejection `GrappleRejection.NotHeld` and leave the set unchanged. | SRD 5.2.1, "Grappling", p. 182 | unhappy (state error) |
| FR-032 | Release MUST cost no action: `ReleaseGrapple` MUST take no action availability, MUST work for a grappler that is Incapacitated, and its results MUST carry no action cost field. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182 ("no action required") | unhappy (boundary) |

### 4.5 Escaping

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-033 | `Escape(target)` on the set MUST remove the grapple holding that target and return one `GrappleEnding(grapple, GrappleEndCause.Escaped)` and no event (the host passes it to `GrappleRules.EndOnSet(set, target, GrappleEndCause.Escaped)`, FR-089). The roll is the use case's (FR-037); the domain operation applies a success it is told about. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182 | happy |
| FR-034 | `Escape(target)` for a target that is not held MUST return `GrappleTransitionRejected` with rejection `GrappleRejection.NotGrappled` and leave the set unchanged. | SRD 5.2.1, "Grappling", p. 182 | unhappy (state error) |
| FR-035 | `GrappleEscapeOption` MUST be an enum with exactly two members, `strengthAthletics` (`Skill.Athletics`) and `dexterityAcrobatics` (`Skill.Acrobatics`), exposed as `GrappleRules.EscapeOptions`; the command carries one, so no other option can be expressed. | SRD 5.2.1, "Grappling", p. 182 ("Strength (Athletics) or Dexterity (Acrobatics)") | unhappy (invalid input made unrepresentable) |
| FR-036 | `GrappleEscapeOption.Ability` MUST be its skill's `Skill.Ability` (`Ability.Strength` for Athletics, `Ability.Dexterity` for Acrobatics, `content-schema-and-decoder.md` FR-106); this spec keeps no second mapping (table 4.10.1). | SRD 5.2.1, "Skills", Skills table, p. 9; "Grappling", p. 182 | happy |
| FR-037 | `EscapeGrappleFromRecord` MUST make the check by composing the pure `d20.ResolveCheck` and `d20.FailWithoutRoll` functions (`d20-tests.md` FR-069) with its own `dice.Roller`, never through `d20.Resolver.Resolve` (shared-types-and-ownership.md FR-044): for a `RollAttempt` it resolves the roll mode once with `modifier.Resolve` (`modifiers-and-advantage.md` FR-010 to FR-017), requests `dice.NewExpression(mode.DiceCount(), dice.D20(), 0)` through the injected roller (the error wrapped once with `srderr.WrapPort`) (`resolve-d20-test-use-case.md` FR-007 gives the request shape), builds the `Roll` and calls `d20.ResolveCheck(d20.AbilityCheck, roll, modifiers, escapeDC)` with the grapple's own `EscapeDC` and the attempt's modifiers unchanged. It MUST add no modifier of its own: the option names the ability (and the skill) whose modifier and Proficiency Bonus the host puts in the attempt, and this spec does not verify them. | SRD 5.2.1, "Grappling", p. 182; "D20 Tests", p. 6; "Skills", p. 9 | happy |
| FR-038 | When the check succeeds, `EscapeGrappleFromRecord` MUST return `EscapeGrappleFromRecordEscaped` carrying the new set without that grapple, the `GrappleEscapeCheck`, the `GrappleEnding(grapple, escaped)` and `ActionSpent` true; the host passes the ending to `GrappleRules.EndOnSet` (FR-089), which removes the Grappled condition and emits `GrappleEnded`. | SRD 5.2.1, "Grappling", "Ending a Grapple" ("ending the condition on itself on a success"), p. 182 | happy |
| FR-039 | When the check fails, `EscapeGrappleFromRecord` MUST return `EscapeGrappleFromRecordHeld` carrying the unchanged set, the check, `ActionSpent` true and no events or endings. | SRD 5.2.1, "Grappling", p. 182; "D20 Tests", p. 6 | unhappy (boundary) |
| FR-040 | The comparison of total and DC (total equal to or above the DC succeeds) MUST be `d20.ResolveCheck`'s and not recomputed here; both result variants MUST carry the resolver's result unchanged inside `GrappleEscapeCheck`. | SRD 5.2.1, "D20 Tests", p. 6 | unhappy (boundary) |
| FR-041 | Every `EscapeGrappleFromRecordResult` variant MUST carry `ActionSpent`: true for `EscapeGrappleFromRecordEscaped` and `EscapeGrappleFromRecordHeld`, false for `EscapeGrappleFromRecordRejected`, where no die is rolled. The escape result has no `Invalid` variant: every input of the command is a typed value. The godoc states that a true value means the Grappled creature's action is spent; how the host spends it is Q-4. | SRD 5.2.1, "Grappling", "Ending a Grapple" ("can use its action to make"), p. 182 | happy |
| FR-042 | `EscapeGrappleFromRecord` MUST return `EscapeGrappleFromRecordRejected` with `GrappleRejection.NoAction` when the Grappled creature's `ConditionActionAvailability` is `none` (it is Incapacitated and can take no action); no die is rolled. | SRD 5.2.1, "Grappling", p. 182; "Incapacitated [Condition]", "Inactive", p. 184 (`conditions.md` FR-064) | unhappy (state error) |
| FR-043 | `EscapeGrappleFromRecord` MUST return `EscapeGrappleFromRecordRejected` with `GrappleRejection.NotGrappled` when no grapple holds the named target; no die is rolled. | SRD 5.2.1, "Grappling", p. 182 | unhappy (state error) |
| FR-044 | When both FR-042 and FR-043 apply, the reason MUST be `notGrappled` (a creature that is not grappled has nothing to escape; the availability of its action is not consulted). | package design (the SRD gives no order) | unhappy (rule interaction) |
| FR-045 | An `AutomaticFailure` attempt MUST give `EscapeGrappleFromRecordHeld` carrying the `ForcedFailureResult` that `d20.FailWithoutRoll(d20.AbilityCheck, d20.Automatic)` returns (`d20-tests.md` FR-036), drawing no die. | d20-tests.md (forced failure); SRD 5.2.1, "D20 Tests", p. 6; "Grappling", p. 182 | unhappy (rule interaction) |
| FR-046 | Advantage and Disadvantage in the attempt MUST reach the roll through `modifier.Resolve` unchanged: with Disadvantage two d20s are drawn and the lower is used; with both, neither applies and one d20 is drawn. | SRD 5.2.1, "Advantage/Disadvantage", p. 8 (neither applies when both); "D20 Tests", p. 6 (roll two d20s) (via d20-tests.md) | unhappy (rule interaction) |
| FR-047 | When the `dice.Roller` returns an error, `EscapeGrappleFromRecord` MUST return it wrapped once by `srderr.WrapPort(dice.Port, "roll", err)` (a `*srderr.PortError`, `errors.Is` reaching the cause) and return no result (nothing to apply). | `doc/rules/architecture.md` section 4 (port failure contract) | unhappy (failure of collaborator) |
| FR-105 | The escape check is a "check made to end the Grappled condition", so a Goliath's `SpeciesAdvantage` handoffs reach it only through the attempt's `circumstances` (`resolve-d20-test-use-case.md` FR-005), which the host fills from the ability-check form of `SpeciesPassives` (`species-trait-behaviour.md` FR-013, FR-039): Powerful Build gives Advantage on either escape option, and Large Form gives Advantage on the Strength (Athletics) option only (a Strength check). Both are `Advantage` circumstances, so together they are still one Advantage (FR-046; `modifier.Resolve`). This spec reads no species type and adds no modifier (FR-037). Powerful Build counts as one size larger for carrying capacity only, so the size predicate of FR-062 to FR-065 and FR-093 uses the actual current size (Large while Large Form lasts), supplied by the host. | SRD 5.2.1, "Powerful Build", "Large Form", p. 86; "Grappling", p. 182; "Advantage/Disadvantage", p. 8 | unhappy (rule interaction) |

### 4.6 Ending a grapple by state: Incapacitated and out of range

The hold rule is this spec's (shared-types-and-ownership.md FR-029): the host or
the encounter calls `GrappleRules.Hold` per grapple (FR-087) and `Review` is
bookkeeping over that rule.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | `GrappleObservation` (built by `NewGrappleObservation(grappler, target, grapplerConditions, distanceFeet)`) MUST carry the grappler's `ConditionSet` and the distance in feet between target and grappler. `NewGrappleObservation` MUST return a `*GrappleProblemError` (satisfying `errors.Is(err, srderr.ErrInvalidInput)`) for the same grappler and target (`GrappleProblem.SameGrapplerAndTarget`) and for a negative distance (`GrappleProblem.NegativeDistance`). | SRD 5.2.1, "Grappling", "Ending a Grapple" ("the distance between the Grappled target and the grappler"), p. 182; architecture section 6 | unhappy (boundary) |
| FR-049 | `Review` MUST end a grapple whose observation gives `GrappleRules.Hold(grapplerConditions, distanceFeet, grapple.RangeFeet)` equal to `GrappleHold.GrapplerIncapacitated`, returning `GrappleEnding(grapple, GrappleEndCause.GrapplerIncapacitated)`. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182 | happy |
| FR-050 | `Review` MUST end a grapple whose hold is `GrappleHold.OutOfRange`, returning `GrappleEnding(grapple, GrappleEndCause.OutOfRange)`. | SRD 5.2.1, "Grappling", "Ending a Grapple" ("exceeds the grapple's range"), p. 182 | happy |
| FR-051 | `Review` MUST keep a grapple whose hold is `GrappleHold.Holds` (a distance equal to the range holds, FR-087, table 4.10.2). | SRD 5.2.1, "Grappling", p. 182 ("exceeds") | unhappy (boundary) |
| FR-052 | One observation yields one hold, so one grapple yields at most one ending; the precedence when both causes apply is FR-088. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182; package design (FR-088) | unhappy (rule interaction) |
| FR-053 | A grapple with no observation MUST be left unchanged. | package design | unhappy (empty and minimal) |
| FR-054 | `Review` MUST return `GrappleTransitionInvalid` (`ReviewGrapples`: `ReviewGrapplesInvalid`, FR-068) with `GrappleProblem.UnknownGrapple` for an observation naming a grappler and target that no grapple in the set holds; it MUST NOT panic. | `doc/rules/architecture.md` section 6; `combat-foundations.md` FR-031 (unknown combatant is `invalid`) | unhappy (invalid input) |
| FR-055 | `Review` MUST return `GrappleTransitionInvalid` with `GrappleProblem.DuplicateObservation` for two observations of the same grapple; it MUST NOT panic. When both FR-054 and FR-055 apply, FR-103 decides. | `doc/rules/architecture.md` section 6; `combat-foundations.md` FR-031 | unhappy (invalid input) |
| FR-056 | `Review` with no observations, or on an empty set, MUST return the same set, no events and no endings. | package design | unhappy (empty and minimal) |
| FR-057 | When several grapples end in one `Review`, the endings MUST follow the order in which the grapples were begun, not the order of the observations. | `doc/rules/architecture.md` section 3 (determinism) | unhappy (rule interaction) |
| FR-058 | A grapple whose hold is `GrappleHold.Holds` MUST be unchanged by `Review`, and `Review` MUST return no ending for it. | SRD 5.2.1, "Grappling", p. 182 | happy |
| FR-059 | A grapple ended by any route (FR-029, FR-033, FR-049, FR-050, FR-060) MUST free its part and its target in the returned set. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182 | unhappy (rule interaction) |
| FR-060 | `ConditionEnded(target)` MUST remove the grapple holding that target and return one event `GrappleRecordDropped(grappler, target)` and no `GrappleEnding` (the condition is already gone, so there is nothing for `GrappleRules.EndOnSet` to do); it is for the Grappled condition ending by some other rule (an effect of a spell, say) so the record never outlives the condition. The use case that exposes it is `DropGrappleRecord` (FR-098). | SRD 5.2.1, "Grappling" ("Grappled Condition"), p. 182; package design (Q-9) | happy |
| FR-061 | `ConditionEnded(target)` for a target that is not held MUST return `GrappleTransitionRejected` with `GrappleRejection.NotGrappled` and leave the set unchanged. | package design | unhappy (state error) |

### 4.7 Dragging and carrying

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-062 | `ExtraMovementCostPerFoot(grappler, grapplerSize, carried, sizes)` MUST return the sum, over each creature in `carried`, of `GrappleRules.MovementCost(grapplerSize, size of that creature)` (FR-093). It is a pure query and changes no grapple (table 4.10.3). The per-creature sum is package design (interpretation, Q-5): the SRD states the cost in each Grappled creature's entry and does not say how several add up. | SRD 5.2.1, "Grappled [Condition]", "Movable", p. 182; "Creature Size", p. 14; package design for the sum | happy |
| FR-063 | With nothing carried, the extra cost MUST be 0, whatever the grappler holds. | SRD 5.2.1, "Grappled [Condition]", "Movable" ("can drag or carry"), p. 182 | unhappy (empty and minimal) |
| FR-064 | `ExtraMovementCostPerFoot` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when a carried creature is not held by that grappler. | `doc/rules/architecture.md` section 6 | unhappy (invalid input) |
| FR-065 | `ExtraMovementCostPerFoot` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when `sizes` lacks the size of the grappler's carried creature (the `sizes` map is for the carried creatures; the grappler's size is `grapplerSize`). | `doc/rules/architecture.md` section 6 | unhappy (invalid input) |

### 4.8 Use cases, events and telemetry

Telemetry follows `combat-foundations.md` FR-027 to FR-037: every use case runs
through `CombatTelemetry`, which owns the span `srd5e.combat.<operation>`, the
shared counters and the duration; this spec adds the two signals of
FR-072 and FR-073, the five `<operation>OperationValue`/`<operation>Span` pairs (FR-070) and the `LogEvent` of FR-078 as constants in `CombatSignals`. The domain (`Grapples`,
`GrappleRules`) is silent.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-066 | `BeginGrapple` MUST take the current `Grapples` and the new grapple's data, build the `Grapple` with `NewGrapple`, call `Begin`, and return `BeginGrappleApplied(grapples, grapple, events)`, `BeginGrappleRejected(rejection)` or `BeginGrappleInvalid(problem)`. It MUST roll no die, apply no condition and consume no port but the telemetry bundle; the host applies `Grapple.Condition`. | `doc/rules/architecture.md` section 4; SRD 5.2.1, "Grappling" ("Grappled Condition"), p. 182 | happy |
| FR-067 | `ReleaseGrapple` MUST return `ReleaseGrappleApplied(grapples, ending)`, `ReleaseGrappleRejected(rejection)` or `ReleaseGrappleInvalid(problem)` (same grappler and target). | SRD 5.2.1, "Grappling", p. 182 | happy |
| FR-068 | `ReviewGrapples` MUST return `ReviewGrapplesApplied(grapples, endings)` (endings empty when nothing ended) or `ReviewGrapplesInvalid(problem)` (the mapping of `GrappleTransitionInvalid`, FR-054, FR-055, FR-103). | SRD 5.2.1, "Grappling", p. 182 | happy |
| FR-069 | A use case called twice with equal commands (and the same fixed dice) MUST return equal results, and MUST NOT alter its command. | `doc/rules/testing.md` section 4 (determinism, immutability) | unhappy (determinism) |
| FR-070 | The operations (`CombatOperation`) MUST be `begin_grapple`, `release_grapple`, `escape_grapple_from_record`, `review_grapples` and `drop_grapple_record`. Each call MUST run in the span `srd5e.combat.<operation>` through `CombatTelemetry`, with `srd5e.combat.operation`, `srd5e.combat.result` (a closed set per use case, one value per outcome or reason, section 7; finer than the class) and `srd5e.combat.result_class`. The escape has no `applied` value: its values are `escaped`, `held`, `not_grappled` and `no_action`. `CombatSignals` gains one `<operation>OperationValue` and one `<operation>Span` constant per operation (`combat-foundations.md` FR-037, FR-040); a test pairs each value with its span. | `combat-foundations.md` FR-027 to FR-030; `doc/rules/observability.md` section 6 | happy |
| FR-071 | Each result variant MUST be classified as one `CombatResultClass`: `Applied`, `Escaped` and `Held` are `applied` (a failed escape is a normal outcome), `Rejected` is `refused`, `Invalid` is `invalid`; every `srd5e.combat.result` value of section 7 maps to exactly one class (span error status). `srd5e.outcome` MUST be `success` or `failure` only, from the class. The shared counters `srd5e.combat.operations`, `srd5e.combat.operation_duration`, `srd5e.combat.events` and `srd5e.combat.errors` are recorded once by `CombatTelemetry`; this spec defines no counter, histogram or attribute value that repeats them. | `combat-foundations.md` FR-029 to FR-031, FR-037; `shared-types-and-ownership.md` FR-041 | happy |
| FR-072 | Each `GrappleEnding` returned by a use case, and each `GrappleRecordDropped` returned by `DropGrappleRecord` (FR-098), MUST increment `srd5e.combat.grapple.ended` once with `srd5e.combat.grapple.end_reason` (`released`, `escaped`, `grappler_incapacitated`, `out_of_range`, `condition_ended`; the last for the dropped record). | `doc/rules/observability.md` section 5 | happy |
| FR-073 | Each call that returns a result carrying a set (`BeginGrappleApplied`, `ReleaseGrappleApplied`, `EscapeGrappleFromRecordEscaped`, `EscapeGrappleFromRecordHeld`, `ReviewGrapplesApplied`, `DropGrappleRecordApplied`) MUST record the size of that set in `srd5e.combat.grapple.in_force`, with attribute `srd5e.combat.operation`. A result with no set (`...Rejected`, `...Invalid`) records nothing, and a call that throws records nothing here (only the shared duration, `combat-foundations.md` FR-029). | `doc/rules/observability.md` sections 4, 5 | happy |
| FR-074 | A rejection (`partInUse`, `targetAlreadyGrappled`, `notHeld`, `notGrappled`, `noAction`) and a held escape MUST be normal outcomes: span status not error, no error counter, no log above `FINE`. An `Invalid` result sets the span error status but does not increment `srd5e.combat.errors` and has no `error.type`. | `combat-foundations.md` FR-031, FR-032, FR-034; `doc/rules/observability.md` sections 6, 7 | unhappy (state error) |
| FR-075 | When the `dice.Roller` returns an error inside `EscapeGrappleFromRecord`, `CombatTelemetry` MUST record it once (span error status, `srd5e.combat.errors` once with `error.type` from `srderr.ErrorType`) and return it, logging nothing. Any other error (for example `srderr.ErrInvalidInput`) MUST be returned unchanged with the span recording it; no counter is incremented and nothing is logged. Because the escape composes `d20.ResolveCheck` and calls no use case (shared-types-and-ownership.md FR-044), no `srd5e.d20_test.*` signal fires, no nested `telemetry bundle` is built and no error can be counted twice. | `combat-foundations.md` FR-032, FR-035; `shared-types-and-ownership.md` FR-044; `doc/rules/observability.md` section 7 ("Errors: who records what") | unhappy (failure of collaborator) |
| FR-076 | Every use case MUST return equal results with the zero-value `telemetry.Noop` and with a recording double, and its `NewX` constructor MUST work with no telemetry option. | `doc/rules/observability.md` section 9; `combat-foundations.md` FR-036 | unhappy (determinism) |
| FR-077 | Attributes MUST NOT contain combatant ids, part names, DCs or distances; only the closed sets of section 7. | `combat-foundations.md` FR-033; `doc/rules/observability.md` section 5 (low cardinality) | unhappy (invalid input) |
| FR-078 | A call MUST NOT log at `INFO` or above; it MAY emit at most one `FINE` log (`'grapple operation'`, a `LogEvent` constant in `CombatSignals`, `combat-foundations.md` FR-039) with `srd5e.combat.operation` and `srd5e.combat.result`, supplied lazily; nothing on failure. | `combat-foundations.md` FR-034; `doc/rules/observability.md` section 7 | happy |
| FR-079 | The `grappling` domain files (including `GrappleRules`) MUST contain no logging, metric or span call and no import of an observability port. | `doc/rules/architecture.md` section 9; `doc/rules/observability.md` section 3 | unhappy (state error) |
| FR-080 | Package `combat` MUST export the types of section 6, each with a godoc comment starting with its name, and no generic variant name (`Applied`, `Rejected`, `Found`, `NotFound`, `Succeeded`, `Invalid`); the package MUST NOT re-export `Grappled`, `DifficultyClass`, `CreatureSize`, `Skill`, `ProficiencyBonus` or `ConditionSet` (they belong to their owners), and every exported name MUST be unique across the package. | `doc/rules/architecture.md` sections 4, 7, 9; `combat-foundations.md` FR-022, FR-024 | happy |
| FR-081 | `GrappleStarted`, `GrappleEnded` and `GrappleRecordDropped` MUST be the only events defined here; all three MUST satisfy `combat.Event` through one closed family `GrappleEvent` (an interface with an unexported marker method), be immutable values, equal by value, and named in the past tense. Their `kind` MUST be the constants `grapple_started`, `grapple_ended` and `grapple_record_dropped`, unique across the package. No other spec defines `GrappleStarted`, `GrappleEnded` or `GrappleRecordDropped`; `GrappleEventOccurred` (encounter-aggregate.md) only wraps them (`combat-foundations.md` FR-038). | `combat-foundations.md` FR-015 to FR-020; `shared-types-and-ownership.md` FR-019, FR-029 | happy |
| FR-082 | `Combatants()` MUST return a fresh slice `[target, grappler]` without duplicates for all three events, and the `Events` of every transition and result MUST be a `[]Event` that accessors copy; mutating a returned slice MUST NOT change the value. | `combat-foundations.md` FR-018, FR-021 | unhappy (immutability) |
| FR-083 | `NewGraspingHand`, `NewGraspingBodyPart`, `NewGrapple` and `NewGrappleObservation` MUST never panic for any input of their parameter types and MUST return a non-nil `*GrappleProblemError` (carrying one `GrappleProblem`, satisfying `errors.Is(err, srderr.ErrInvalidInput)`) if and only if the input is invalid (FR-002, FR-004, FR-007, FR-008, FR-048); where an input has two problems the order of FR-099 to FR-102 decides. Each has a `MustX` form that panics with that error. `GrappleProblem` is one enum for the spec: `negativeHandIndex`, `blankPartName`, `sameGrapplerAndTarget`, `negativeRange`, `negativeDistance`, `unknownGrapple`, `duplicateObservation`. | `errors-and-results.md` FR-025, FR-035; `combat-foundations.md` FR-005, FR-023 | unhappy (invalid input) |
| FR-084 | There are two reason enums. The reason of a refusal MUST be `GrappleRejection` (`partInUse`, `targetAlreadyGrappled`, `notHeld`, `notGrappled`, `noAction`) held as a field of each `...Rejected` variant and of `GrappleTransitionRejected`; the reason of unusable input MUST be `GrappleProblem`, held as a field of each `...Invalid` variant and of `GrappleTransitionInvalid`. No third reason enum exists. | `combat-foundations.md` FR-022, FR-023 | happy |

### 4.9 Rules taken over from `conditions.md` and `attacks.md`

Ids continue after FR-084; none is renumbered. `GrappleRules` is a stateless domain service (a zero-size struct) in package `combat` (files `grapple_*.go`); its members are
pure, synchronous and silent. Together they replace the rules the older specs
held (section 1, "Withdrawn elsewhere in favour of this spec").

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-085 | `GrappleEndCause` MUST be an enum with exactly four members, `escaped`, `grapplerIncapacitated`, `outOfRange` and `released`, one per way the SRD ends a grapple. It is owned here and no other spec defines it. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182 | happy |
| FR-086 | `GrappleEnded(target, grappler, cause)` MUST be the event for a grapple that ended; it states the fact and is produced only by `GrappleRules.EndOnSet` (FR-089). Two events are equal when target, grappler and cause are equal. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182; `shared-types-and-ownership.md` FR-019, FR-029 | happy |
| FR-087 | `GrappleRules.Hold(grapplerConditions, distanceFeet, rangeFeet)` MUST return `GrappleHold.GrapplerIncapacitated` when the grappler's effective conditions include Incapacitated (`ConditionEffects.IsIncapacitated`, `conditions.md` FR-064), otherwise `GrappleHold.OutOfRange` when `DistanceFeet` is greater than `RangeFeet`, otherwise `GrappleHold.Holds`. A distance equal to the range holds. It MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a negative distance or range. | SRD 5.2.1, "Grappling", "Ending a Grapple" ("the grappler has the Incapacitated condition or if the distance ... exceeds the grapple's range"), p. 182 | happy; unhappy (boundary) |
| FR-088 | When the grappler is Incapacitated and the target is also beyond the range, `GrappleRules.Hold` MUST return `grapplerIncapacitated`. Package design (interpretation): the SRD lists both endings and gives no order, and a grapple ends once, so the order is fixed here for determinism. | SRD 5.2.1, "Grappling", p. 182 (no order stated); package design | unhappy (rule interaction) |
| FR-089 | `GrappleRules.EndOnSet(set, target, cause)` MUST, when `set` holds the explicit condition `Grappled(grappler)`, return `GrappleEndOnSetApplied` carrying the new set without Grappled (through `ConditionSet.Remove`, `conditions.md` FR-031) and the events `ConditionRemoved(target, ConditionKind.Grappled)` then `GrappleEnded(target, grappler, cause)`, in that order. The grappler in the event is the source the removed condition held; `EndOnSet` reads it from the explicit `Grappled(grappler)` by filtering the unmodifiable view of the set's explicit conditions (`conditions.md` FR-012, FR-014, one per kind) for the `Grappled` variant; the getter is named `ConditionSet.Explicit` (amendment requested of `conditions.md`; this spec's Q-16). The event `ConditionRemoved(target, ConditionKind.Grappled)` carries no source, so the grappler cannot be recovered from it. Every `GrappleEndCause` MUST be accepted and carried unchanged. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182; `conditions.md` FR-019, FR-031 | happy |
| FR-090 | `GrappleRules.EndOnSet` on a set that does not hold Grappled MUST return `GrappleEndOnSetNotGrappled`, with no event; the input set is unchanged in every case. | `doc/rules/testing.md` section 4 (state errors, immutability) | unhappy (state error) |
| FR-091 | A plain `ConditionSet.Remove` of Grappled MUST NOT be how a grapple ends: it emits `ConditionRemoved` only (`conditions.md` FR-031). The godoc of `GrappleRules.EndOnSet` states that a host or the encounter ends a grapple through it so `GrappleEnded` always follows `ConditionRemoved`. | `shared-types-and-ownership.md` FR-029; `conditions.md` FR-031 | unhappy (rule interaction) |
| FR-092 | `GrappleRules.Possibility`, `GrappleRules.Hold`, `GrappleRules.MovementCost` and `GrappleRules.GrappleDC` MUST NOT read or change any `Grapples` value; the record and the rules are independent so each is testable alone. | `shared-types-and-ownership.md` principle 1 (testable with plain values) | happy |
| FR-093 | `GrappleRules.MovementCost(grapplerSize, grappledSize)` MUST return 0 when the Grappled creature is Tiny or is two or more sizes smaller than the grappler (sizes in the declaration order of `CreatureSize`), otherwise 1: the extra feet each foot of movement costs the grappler. | SRD 5.2.1, "Grappled [Condition]", "Movable", p. 182; "Creature Size", p. 14 | happy; unhappy (boundary) |
| FR-094 | `GrappleRules.GrappleDC(abilityModifier, proficiencyBonus)` MUST return `DifficultyClass(8 + abilityModifier + proficiencyBonus)`, with no cap or floor. The Unarmed Strike passes its Strength modifier (a Monk's Martial Arts may pass the Dexterity modifier, `class-monk.md` FR-030); the same value is the escape DC of the grapple it starts. | SRD 5.2.1, "Unarmed Strike", Grapple, p. 190 ("8 plus your Strength modifier and Proficiency Bonus", "any escape attempts") | happy; unhappy (boundary) |
| FR-095 | `GrappleRules.OptionSaveAbilities` MUST be exactly `Ability.Strength` and `Ability.Dexterity`: the target chooses which saving throw to make, and a failure gives it the Grappled condition (data the Unarmed Strike result carries, `attacks.md` FR-077). Making the save (the dice, the Dexterity-save Cover bonus) is `attacks.md`'s (FR-092 to FR-094). | SRD 5.2.1, "Unarmed Strike", Grapple, p. 190 | happy |
| FR-096 | `GrappleRules.Possibility(attackerSize, targetSize, hand)` MUST return `GrappleNotPossible(GrappleNotPossibleReason.TargetTooLarge)` when the target is more than one size larger than the attacker (more than one entry after it in the `CreatureSize` order), otherwise `GrappleNotPossible(GrappleNotPossibleReason.NoFreeHand)` when `hand` is `HandAvailability.NoHandFree`, otherwise `GrapplePossible`. When both fail the reason is `targetTooLarge`, the order of `attacks.md` FR-105. Whether a hand is free is the host's statement; the engine models no held items. | SRD 5.2.1, "Unarmed Strike", Grapple, p. 190 ("no more than one size larger than you", "a hand free to grab it"); "Creature Size", p. 14; "Grappling", "One Grapple per Hand", p. 182 | happy; unhappy (boundary, rule interaction) |
| FR-097 | The grapple domain files MUST import only `combat-foundations.md`, `conditions.md`, `content`, `d20`, `character` and kernel published language and no file of `attacks.md`, `initiative-and-actions.md`, `encounter-aggregate.md` or `combat-action-resolution.md` (the specs that use this one, section 1); the application files MUST NOT call another use case. | `shared-types-and-ownership.md` FR-013, FR-015, FR-044 | unhappy (forbidden edge) |
| FR-098 | `DropGrappleRecord` MUST take the current `Grapples` and a target `CombatantID`, call `ConditionEnded` (FR-060, FR-061) and return `DropGrappleRecordApplied(grapples, events)` (one `GrappleRecordDropped`) or `DropGrappleRecordRejected(rejection)` (`GrappleRejection.NotGrappled`, FR-061). It MUST roll no die, touch no `ConditionSet` and consume no port but `telemetry bundle`. It exists so the host has a use case, with signals, for the two routes by which a record outlives its condition: the Grappled condition ended by some other rule, and a failed `apply` of `Grapple.Condition` after `BeginGrapple` recorded the grapple (`ConditionImmune`, `ConditionAlreadyHeld`; Q-9). Operation `drop_grapple_record`. | SRD 5.2.1, "Grappling" ("Grappled Condition"), p. 182; package design (Q-9) | happy |
| FR-099 | `NewGrapple` with two problems MUST report `GrappleProblem.SameGrapplerAndTarget` before `GrappleProblem.NegativeRange`. | `errors-and-results.md` FR-035; `doc/rules/testing.md` section 4 | unhappy (rule interaction) |
| FR-100 | `NewGrappleObservation` with two problems MUST report `GrappleProblem.SameGrapplerAndTarget` before `GrappleProblem.NegativeDistance`. | `errors-and-results.md` FR-035; `doc/rules/testing.md` section 4 | unhappy (rule interaction) |
| FR-101 | `GrappleRules.Hold` with a negative distance and a negative range MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` naming the distance (the distance is checked before the range). | `doc/rules/architecture.md` section 6; `doc/rules/testing.md` section 4 | unhappy (rule interaction) |
| FR-102 | `NewGrapple`'s problems are exactly `sameGrapplerAndTarget` and `negativeRange`, and `NewGrappleObservation`'s are exactly `sameGrapplerAndTarget` and `negativeDistance`; the other members of `GrappleProblem` belong to the part parsers (`negativeHandIndex`, `blankPartName`) and to `Review` (FR-054, FR-055). | `errors-and-results.md` FR-035 | unhappy (invalid input) |
| FR-103 | `Review` with an observation of an unknown grapple and a duplicate observation MUST return `GrappleProblem.UnknownGrapple`: every observation is checked for an unknown grapple, in the order given, before any duplicate is reported. Among several unknown grapples the first in the order given is reported. | `errors-and-results.md` FR-035; `doc/rules/testing.md` section 4 | unhappy (rule interaction) |
| FR-104 | A dead grappler has no actions, as `ConditionActionAvailability.None` does. `GrappleObservation` MUST carry `GrapplerDead` (`bool`, default false) and `GrappleRules.Hold` MUST take the option `WithGrapplerDead(bool)` (default false); with it true `Hold` MUST return `GrappleHold.GrapplerIncapacitated` (before the range test, FR-088), and `Review` passes the observation's value, so a dead grappler whose observed `ConditionSet` lacks Incapacitated still ends the grapple with `GrappleEndCause.GrapplerIncapacitated`. With the default, FR-087 is unchanged. | SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182; "Dead", p. 180 (interpretation: a dead creature can take no action, so it holds nothing); `combat-action-resolution.md` Q-15, FR-120 | unhappy (rule interaction) |

### 4.10 Case tables

Each row becomes one table-driven test.

**4.10.1 Escape check option (FR-035, FR-036)**

| Label | Option | Expected | Req. | Source | Path |
|---|---|---|---|---|---|
| Athletics | `strengthAthletics` | Strength check | FR-036 | p. 9, p. 182 | happy |
| Acrobatics | `dexterityAcrobatics` | Dexterity check | FR-036 | p. 9, p. 182 | happy |
| only two options | `GrappleEscapeOption.Values` | exactly the two above, in this order | FR-035 | p. 182 | unhappy (boundary) |

**4.10.2 Range and Incapacitated (`GrappleRules.Hold` composed with `Review`; FR-049 to FR-052, FR-058, FR-087, FR-088)**

| Label | Grapple range | Distance | Grappler | `Hold` | `Review` | Req. | Source | Path |
|---|---|---|---|---|---|---|---|---|
| adjacent, can act | 5 | 0 | can act | `holds` | kept | FR-051, FR-058 | p. 182 | happy |
| at range | 5 | 5 | can act | `holds` | kept | FR-051 | p. 182 | unhappy (boundary) |
| one foot past range | 5 | 6 | can act | `outOfRange` | ended, `outOfRange` | FR-050 | p. 182 | unhappy (boundary) |
| far beyond range | 5 | 60 | can act | `outOfRange` | ended, `outOfRange` | FR-050 | p. 182 | unhappy (boundary) |
| range 0, distance 0 | 0 | 0 | can act | `holds` | kept | FR-051 | p. 182 | unhappy (boundary) |
| range 0, distance 1 | 0 | 1 | can act | `outOfRange` | ended, `outOfRange` | FR-050 | p. 182 | unhappy (boundary) |
| long reach at limit | 15 | 15 | can act | `holds` | kept | FR-051 | p. 182 | unhappy (boundary) |
| long reach one past | 15 | 16 | can act | `outOfRange` | ended, `outOfRange` | FR-050 | p. 182 | unhappy (boundary) |
| Incapacitated, adjacent | 5 | 0 | Incapacitated | `grapplerIncapacitated` | ended, `grapplerIncapacitated` | FR-049 | p. 182 | happy |
| Unconscious (implies Incapacitated) | 5 | 0 | Unconscious | `grapplerIncapacitated` | ended, `grapplerIncapacitated` | FR-049, FR-087 | p. 182, p. 191 | unhappy (rule interaction) |
| Incapacitated and out of range | 5 | 6 | Incapacitated | `grapplerIncapacitated` | one ending, `grapplerIncapacitated` | FR-052, FR-088 | p. 182 | unhappy (rule interaction) |
| dead, not Incapacitated | 5 | 0 | can act, `GrapplerDead` | `grapplerIncapacitated` | ended, `grapplerIncapacitated` | FR-104 | p. 180, p. 182 | unhappy (rule interaction) |
| negative distance | 5 | -1 | can act | an `srderr.ErrInvalidInput` error (`GrappleObservation` rejects it, FR-048) | n/a | FR-048, FR-087 | n/a | unhappy (invalid input) |
| no observation | 5 | (none) | (none) | (none) | kept | FR-053 | n/a | unhappy (empty and minimal) |
| unknown grapple | n/a | n/a | n/a | n/a | `ReviewGrapplesInvalid(unknownGrapple)` | FR-054 | n/a | unhappy (invalid input) |
| same grapple twice | n/a | n/a | n/a | n/a | `ReviewGrapplesInvalid(duplicateObservation)` | FR-055 | n/a | unhappy (invalid input) |

**4.10.3 Extra movement cost per foot (FR-062, FR-063, FR-093; summing is package design, Q-5)**

| Label | Grappler size | Carried | Expected | Req. | Source | Path |
|---|---|---|---|---|---|---|
| carries same size | Medium | one Medium | 1 | FR-093 | p. 182 | happy |
| carries one size smaller | Large | one Medium | 1 | FR-093 | p. 182 | unhappy (boundary) |
| carries two sizes smaller | Large | one Small | 0 | FR-093 | p. 182 | unhappy (boundary) |
| carries Tiny | Medium | one Tiny | 0 | FR-093 | p. 182 | unhappy (boundary) |
| carries Tiny, Tiny grappler | Tiny | one Tiny | 0 | FR-093 | p. 182 | unhappy (boundary) |
| carries larger | Medium | one Large | 1 | FR-093 | p. 182 | unhappy (boundary) |
| two carried, both costly | Large | two Medium | 2 | FR-062, Q-5 | p. 182 | unhappy (rule interaction) |
| two carried, one free | Large | Medium and Small | 1 | FR-062, Q-5 | p. 182 | unhappy (rule interaction) |
| holds two, carries one | Large | holds Medium and Medium, carries one | 1 | FR-062 | p. 182 | unhappy (rule interaction) |
| carries none | Medium | none, holds one | 0 | FR-063 | p. 182 | unhappy (empty and minimal) |
| carries a creature not held | Medium | an unheld id | an `srderr.ErrInvalidInput` error | FR-064 | n/a | unhappy (invalid input) |
| size missing | Medium | one held, no size entry | an `srderr.ErrInvalidInput` error | FR-065 | n/a | unhappy (invalid input) |

**4.10.4 Beginning (FR-022 to FR-027; rejected = `GrappleTransitionRejected`)**

| Label | State | Command | Expected | Req. | Source | Path |
|---|---|---|---|---|---|---|
| first grapple | empty | A holds B with hand 0 | applied, `GrappleStarted` | FR-022 | p. 182 | happy |
| second part, second target | A holds B with hand 0 | A holds C with hand 1 | applied | FR-023 | p. 182 | unhappy (rule interaction) |
| second part, same target | A holds B with hand 0 | A holds B with hand 1 | rejected `targetAlreadyGrappled` | FR-025 | p. 179 | unhappy (state error) |
| same part, other target | A holds B with hand 0 | A holds C with hand 0 | rejected `partInUse` | FR-024 | p. 182 | unhappy (state error) |
| same part, same target | A holds B with hand 0 | A holds B with hand 0 | rejected `partInUse` | FR-024, FR-026 | p. 182 | unhappy (rule interaction) |
| tentacle then maw | A holds B with a tentacle | A holds C with a maw | applied | FR-023 | p. 182 | unhappy (rule interaction) |
| two grapplers, one target | A holds B | D holds B with its hand 0 | rejected `targetAlreadyGrappled` | FR-025 | p. 179 | unhappy (state error) |
| same part, two creatures | A holds B with hand 0 | D holds C with hand 0 | applied | FR-015 | p. 182 | unhappy (boundary) |
| grappler itself held | D holds A | A holds B with hand 0 | applied | FR-027 | p. 182 | unhappy (rule interaction) |
| released part reused | A holds B, then released | A holds C with hand 0 | applied | FR-030 | p. 182 | unhappy (rule interaction) |

**4.10.5 Escape (FR-037 to FR-046; Dexterity check, escape DC 13, fixed d20 and modifier)**

| Label | d20 | Modifier | Total | Expected | Req. | Source | Path |
|---|---|---|---|---|---|---|---|
| total equals DC | 10 | +3 | 13 | escaped, `GrappleEnding(escaped)`, `ActionSpent` | FR-038, FR-040, FR-041 | p. 6, p. 182 | unhappy (boundary) |
| total one below DC | 9 | +3 | 12 | held, unchanged set, no events, `ActionSpent` | FR-039, FR-040 | p. 6, p. 182 | unhappy (boundary) |
| total well above | 20 | +3 | 23 | escaped | FR-038 | p. 6, p. 182 | happy |
| Disadvantage | 17 and 4 | +0 | 4 | held; two d20 drawn | FR-046 | p. 6, p. 8 | unhappy (rule interaction) |
| Advantage and Disadvantage | 12 | +1 | 13 | one d20 drawn | FR-046 | p. 8 | unhappy (rule interaction) |
| Goliath sources | 17 and 4 (Advantage) | +0 | 17 | Powerful Build plus Large Form: two d20s, one Advantage; Dexterity option without Large Form: Powerful Build only | FR-105 | p. 86 | unhappy (rule interaction) |
| automatic failure | (none) | | | held (`ForcedFailureResult`), no die drawn | FR-045 | p. 6 | unhappy (rule interaction) |
| Incapacitated escaper | (none) | | | rejected `noAction`, no die, `ActionSpent` false | FR-042, FR-041 | p. 184 | unhappy (state error) |
| not grappled | (none) | | | rejected `notGrappled`, no die | FR-043 | p. 182 | unhappy (state error) |
| not grappled and Incapacitated | (none) | | | rejected `notGrappled` | FR-044 | package | unhappy (rule interaction) |
| roller fails | (returns an error) | | | error returned wrapped once, set unchanged | FR-047 | architecture 4 | unhappy (failure of collaborator) |

**4.10.6 Option rules and the end operation (FR-089 to FR-096)**

| Label | Input | Expected | Req. | Source | Path |
|---|---|---|---|---|---|
| DC, ordinary | Strength modifier +3, Proficiency Bonus +2 | DC 13 | FR-094 | p. 190 | happy |
| DC, negative modifier | modifier -1, Proficiency Bonus +2 | DC 9 | FR-094 | p. 190 | unhappy (boundary) |
| size, same | Medium attacker, Medium target | `GrapplePossible` | FR-096 | p. 190 | happy |
| size, one larger | Medium attacker, Large target | `GrapplePossible` | FR-096 | p. 190 | unhappy (boundary) |
| size, two larger | Medium attacker, Huge target | `GrappleNotPossible(targetTooLarge)` | FR-096 | p. 190 | unhappy (boundary) |
| size, smaller | Large attacker, Tiny target | `GrapplePossible` | FR-096 | p. 190 | unhappy (boundary) |
| no hand free | Medium and Medium, `noHandFree` | `GrappleNotPossible(noFreeHand)` | FR-096 | p. 190 | unhappy (state error) |
| both fail | Medium attacker, Huge target, `noHandFree` | `GrappleNotPossible(targetTooLarge)` | FR-096 | p. 190 | unhappy (rule interaction) |
| end, released | set holds `Grappled(A)` on B | set without Grappled; `ConditionRemoved`, `GrappleEnded(B, A, released)` | FR-089 | p. 182 | happy |
| end, each cause | the four causes | event carries the cause unchanged | FR-089 | p. 182 | unhappy (boundary) |
| end, not grappled | empty set | `GrappleEndOnSetNotGrappled`, no event, set equal | FR-090 | p. 182 | unhappy (state error) |
| plain remove | `ConditionSet.Remove(Grappled)` | `ConditionRemoved` only, no `GrappleEnded` | FR-091 | conditions.md FR-031 | unhappy (rule interaction) |

**4.10.7 Precedence of problems (FR-099 to FR-103)**

| Label | Input | Expected | Req. | Source | Path |
|---|---|---|---|---|---|
| grapple, both problems | same grappler and target, range -1 | `sameGrapplerAndTarget` | FR-099 | n/a | unhappy (rule interaction) |
| grapple, range only | different ids, range -1 | `negativeRange` | FR-099, FR-102 | n/a | unhappy (boundary) |
| observation, both problems | same ids, distance -1 | `sameGrapplerAndTarget` | FR-100 | n/a | unhappy (rule interaction) |
| observation, distance only | different ids, distance -1 | `negativeDistance` | FR-100, FR-102 | n/a | unhappy (boundary) |
| hold, negative range only | distance 0, range -1 | an `srderr.ErrInvalidInput` error | FR-087 | n/a | unhappy (invalid input) |
| hold, both negative | distance -1, range -1 | an `srderr.ErrInvalidInput` error naming the distance | FR-101 | n/a | unhappy (rule interaction) |
| review, unknown and duplicate | one unknown grapple observed twice | `unknownGrapple` | FR-103 | n/a | unhappy (rule interaction) |
| review, duplicate then unknown | a known grapple observed twice, then an unknown one | `unknownGrapple` | FR-103 | n/a | unhappy (rule interaction) |
| review, duplicate only | a known grapple observed twice | `duplicateObservation` | FR-055 | n/a | unhappy (invalid input) |

### 4.11 Withdrawn ids

None.

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

Context `combat`: one Go package, domain files `grapple_*.go` and use-case files `*_grapple_usecase.go` (go.md section 1). All domain types are immutable values (unexported fields, value receivers), pure, synchronous and silent.

**Domain**

- `GraspingPart` (closed interface with an unexported marker method; variants `GraspingHand` and `GraspingBodyPart`), built by `NewGraspingHand(index int) (GraspingHand, error)` and `NewGraspingBodyPart(name string) (GraspingBodyPart, error)`, each with a `MustX` form (FR-001 to FR-005, FR-083).
- `Grapple` (grappler, target, part, `EscapeDC`, `RangeFeet`; `Condition()` accessor; `UnarmedStrikeRangeFeet`; built by `NewGrapple(grappler, target, part, escapeDC, rangeFeet) (Grapple, error)`; FR-006 to FR-011).
- `Grapples`: the grapples in force. Invariants FR-013, FR-014; queries `IsHolding`, `CanTarget`, `GrappleOf`, `HeldBy`, `ExtraMovementCostPerFoot`; transitions `Begin`, `Release`, `Escape`, `Review`, `ConditionEnded`, each returning `GrappleTransition`.
- `GrappleRules`: stateless (zero-size struct, usable as `GrappleRules{}`): `GrappleDC`, `OptionSaveAbilities`, `Possibility`, `EscapeOptions`, `Hold`, `EndOnSet`, `MovementCost` (FR-085 to FR-096). Value types: `GrappleEscapeOption`, `GrappleHold`, `GrappleEndCause`, `HandAvailability`, `GrapplePossibility` (`GrapplePossible`, `GrappleNotPossible(reason)`) with `GrappleNotPossibleReason`, `GrappleEndOnSetResult` (`GrappleEndOnSetApplied(conditions, events)`, `GrappleEndOnSetNotGrappled`).
- `GrappleTransition`: closed, `GrappleTransitionApplied(grapples, events, endings)`, `GrappleTransitionRejected(rejection)`, `GrappleTransitionInvalid(problem)`; `Events` is a `[]Event` that accessors copy (FR-082). `GrappleRejection`: `partInUse`, `targetAlreadyGrappled`, `notHeld`, `notGrappled`, `noAction` (the last for the escape use case only). `GrappleProblem` (FR-083, FR-084).
- `GrappleEnding(grapple, cause)`: data for the host to pass to `GrappleRules.EndOnSet`; not an event.
- `GrappleOfTarget`: closed, `InGrapple(grapple)`, `NotInGrapple` (FR-020). `PartState` (`holding`, `notHoldingGrapple`). `ConditionPermission` and `ConditionActionAvailability` come from `conditions.md`.
- `GrappleEvent`: closed, satisfies `combat.Event` (FR-081, FR-082): `GrappleStarted(grappler, target, part)`, `GrappleEnded(target, grappler, cause)`, `GrappleRecordDropped(grappler, target)`. The encounter aggregate wraps them with its envelope as it does the condition events (`conditions.md` D-12).
- `GrappleObservation`, built by `NewGrappleObservation(grappler, target, grapplerConditions, distanceFeet)` with `WithGrapplerDead(bool)` as the one option (default false) (FR-048, FR-104).
- **Invariants and where enforced:** part constructors (index, name); `Grapple` constructor (grappler is not target, range not negative); `Grapples` constructor (FR-013, FR-014, FR-016); `GrappleObservation` constructor. Each `NewX` returns a typed error satisfying `errors.Is(err, srderr.ErrInvalidInput)`; every host-supplied value goes through a `NewX` (FR-083).
- **Failures:** expected rule outcomes (part in use, target already held, release of something not held, an escape by a creature that is not grappled or cannot act, a held escape, a grapple that is not possible) are closed results. Host input the engine cannot use is an `Invalid` result with a `GrappleProblem` (`combat-foundations.md` FR-031). a returned `srderr.ErrInvalidInput` error remains only for value-object constructors, pure queries over host-built maps (FR-064, FR-065) and `GrappleRules.Hold` with a negative number (FR-087). No `*srderr.PortError` arises in the domain.

**Application**

- Use cases `BeginGrapple`, `ReleaseGrapple`, `EscapeGrappleFromRecord`, `ReviewGrapples`, `DropGrappleRecord`, each with a `<UseCase>Command` and a closed `<UseCase>Result` with operation-prefixed variants (shared-types-and-ownership.md FR-017): `Applied`/`Rejected`/`Invalid` for Begin and Release, `Escaped`/`Held`/`Rejected` for the escape, `Applied`/`Invalid` for Review, `Applied`/`Rejected` for Drop. Every result classifies as a `CombatResultClass` (FR-071).
- `GrappleEscapeCheck`: closed, `GrappleEscapeRolled(CheckResult)`, `GrappleEscapeForced(ForcedFailureResult)` (FR-040, FR-045).
- A use case contains no rules logic: it calls the domain (the escape composes `d20.ResolveCheck` with the roller) and maps the `GrappleTransition` into its own result. It calls no other use case (FR-097).

## 6. Interface

Every use case is a type built by `NewX(opts ...Option)` with one method `Run(ctx context.Context, cmd XCommand) (XResult, error)`; `ctx` is first because the call emits a span (go.md section 4), and the `error` return carries only port failures. Calls are synchronous. Only `EscapeGrappleFromRecord` uses a port: the consumer-side interface `Roller` declared in `combat`'s `ports.go` (`Roll(dice.Expression) (dice.Result, error)`, satisfied structurally by `dice.Roller`); the others take only the optional telemetry bundle (`WithTelemetry(telemetry.Telemetry)`, default the zero-value `telemetry.Noop`, shared-types-and-ownership.md FR-036). No new port, test double or contract suite.

- **`NewBeginGrapple(opts ...Option)`**: `BeginGrappleCommand{Grapples, Grappler, Target, Part, EscapeDC, RangeFeet}` (raw values; the use case builds the `Grapple` with `NewGrapple`, so bad values give `BeginGrappleInvalid`). The host calls it when a Grapple option's result carries the Grappled-applied effect data (`attacks.md` FR-077; the DC came from `GrappleRules.GrappleDC`; the range defaults to `Grapple.UnarmedStrikeRangeFeet`), or after resolving a monster's special grapple attack.
- **`NewReleaseGrapple(opts ...Option)`**: `ReleaseGrappleCommand{Grapples, Grappler, Target}`.
- **`NewEscapeGrappleFromRecord(roller Roller, opts ...Option)`**: `EscapeGrappleFromRecordCommand{Grapples, Target, Option, Actions, Attempt}` with `option` a `GrappleEscapeOption`, `actions` the escaper's `ConditionActionAvailability`, `attempt` a `CheckAttempt`. `Escaped` and `Held` carry the `GrappleEscapeCheck`, the `Grapple` and `ActionSpent`.
- **`NewDropGrappleRecord(opts ...Option)`**: `DropGrappleRecordCommand{Grapples, Target}`. The host calls it whenever it removes `Grappled` for a reason other than a `GrappleEnding` it was given, and when applying `Grapple.Condition` failed after `BeginGrapple` (FR-098, Q-9).
- **`NewReviewGrapples(opts ...Option)`**: `ReviewGrapplesCommand{Grapples, Observations}`; each `GrappleObservation` carries the grappler's `ConditionSet`, the distance the host reports and whether the grappler is dead (FR-104).
- **`GrappleRules`** (for hosts that skip the use cases, and the one the encounter aggregate calls): `GrappleDC(abilityModifier, proficiencyBonus) DifficultyClass`; `OptionSaveAbilities() []ability.Ability`; `Possibility(attacker, target content.CreatureSize, HandAvailability) GrapplePossibility`; `EscapeOptions() []GrappleEscapeOption`; `Hold(grapplerConditions ConditionSet, distanceFeet, rangeFeet int, opts ...HoldOption) (GrappleHold, error)` (error for a negative number, FR-087, FR-101); `EndOnSet(set ConditionSet, target CombatantID, cause GrappleEndCause) GrappleEndOnSetResult`; `MovementCost(grapplerSize, grappledSize content.CreatureSize) int`.
- **Port failure contract:** the roller's `Roll` returns an `error`; `Run` wraps it once with `srderr.WrapPort` (or runs the call under `srderr.GuardValue`), `CombatTelemetry` records it once, and `Run` returns it with no result; callers branch with `errors.Is` / `errors.As` (FR-047, FR-075).
- **Exports** (package `combat`, no re-export package): every type named in section 5, the five use cases with their commands and results, and the constants added to `CombatSignals`; imported types (`CombatantID`, `Event`, `DifficultyClass`, `Grappled`, `ConditionSet`, `ConditionActionAvailability`, `ConditionPermission`, `CreatureSize`, `Skill`, `ProficiencyBonus`) are exported by their owning packages and imported by consumers from there (FR-080).

## 7. Observability

Shared conventions: `combat-foundations.md` FR-027 to FR-037 (spans, counters `srd5e.combat.operations`, `srd5e.combat.operation_duration`, `srd5e.combat.events`, `srd5e.combat.errors`; `srd5e.outcome` and `error.type` via the `telemetry` attribute-key constants, `observability-ports.md` FR-052). This section lists only what this spec adds.

1. **What can go wrong?** Expected refusals (part already holding, target already held, releasing what is not held, escaping while not grappled or unable to act); a dice roller failing during an escape; a host reporting grapples inconsistently (an `Invalid` result).
2. **How would an operator know?** `srd5e.combat.result` and `srd5e.combat.result_class` on the shared operations counter; `Invalid` gives span error status; a roller failure raises `srd5e.combat.errors` (once). A returned `srderr.ErrInvalidInput` error is recorded only by the span (`observability.md` section 7).
3. **What to diagnose with?** Operation, result, class, end reason, grapples in force. Ids, part names, DCs and distances stay in results, not attributes (FR-077).
4. **Counted or timed here?** Grapples ended by reason, grapples in force and one span per operation (table below); the rest is shared.
5. **Existing instrumentation still correct?** Yes. The escape composes `d20.ResolveCheck` and calls no use case, so no `srd5e.d20_test.*` signal fires and no error is counted twice (FR-075). The domain is silent.
6. **What should a consumer monitor?** The `refused` share for `begin_grapple` by result (a high share means the host calls `Begin` without consulting `IsHolding` and `GrappleOf`); the `ended` mix by reason (a large `out_of_range` share with few `released` or `escaped` suggests wrong distance reports); any error increment.
7. **Noisy?** No: at most one `FINE` log per call (FR-078).

Closed value sets: `srd5e.combat.operation`: `begin_grapple`, `release_grapple`, `escape_grapple_from_record`, `review_grapples`, `drop_grapple_record`. `srd5e.combat.result_class`: `applied`, `refused`, `invalid`. `srd5e.combat.result` (one class each, `combat-foundations.md` FR-030): `begin_grapple`: `applied`; `part_in_use`, `target_already_grappled` (refused); `same_grappler_and_target`, `negative_range` (invalid). `release_grapple`: `applied`; `not_held` (refused); `same_grappler_and_target` (invalid). `escape_grapple_from_record`: `escaped`, `held` (both applied); `not_grappled`, `no_action` (refused); no `applied`, no `invalid`. `review_grapples`: `applied`; `unknown_grapple`, `duplicate_observation` (invalid). `drop_grapple_record`: `applied`; `not_grappled` (refused). `srd5e.combat.grapple.end_reason`: `released`, `escaped`, `grappler_incapacitated`, `out_of_range`, `condition_ended` (the dropped record).

Rows to add to `doc/observability/signals.md`:

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.grapple.ended` | counter, `{grapple}` | `srd5e.combat.grapple.end_reason` | How grapples end (one per `GrappleEnding` and per `GrappleRecordDropped`) | `out_of_range` above 50 percent of endings over a day: distance reports are probably wrong; an `escaped` ratio far from what the escape DCs predict suggests a dice or modifier problem |
| `srd5e.combat.grapple.in_force` | histogram, `{grapple}` | `srd5e.combat.operation` | Size of the `Grapples` value in a result that carries one (FR-073) | p99 above 50 grapples in one encounter (a host that never ends grapples) |
| `srd5e.combat.begin_grapple`, `srd5e.combat.release_grapple`, `srd5e.combat.escape_grapple_from_record`, `srd5e.combat.review_grapples`, `srd5e.combat.drop_grapple_record` | span (one row each, owned here) | `srd5e.combat.operation`, `srd5e.combat.result`, `srd5e.combat.result_class`, `srd5e.outcome` | One call of that use case (FR-070) | Shared: `combat-foundations.md` section 5 |

Thresholds are starting values to be replaced by baselines. The constants added to `CombatSignals` are the five `...OperationValue`/`...Span` pairs, `grappleEndedMetric`, `grappleInForceMetric`, `grappleEndReasonAttribute` and the `LogEvent` for `'grapple operation'`; no second constants class exists.

## 8. Test plan

Doubles: the fixed `dicetest.FixedRoller` and the failing `dicetest.CallbackRoller (returning an error)` from `dice/dicetest`, and the recording telemetry doubles from `telemetry/telemetrytest`. Tests are table-driven in package `combat_test` (one test per table row, `t.Run`, `t.Parallel` where safe), with `Fuzz*` tests for the part and grapple constructors and `Example*` functions for the public flow. No new port, so no contract suite. Tables 4.10.2 and 4.10.6 build real `ConditionSet`s; the escape tests use the real `d20.ResolveCheck` with the fixed roller.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-021 | unit | parts and equality (hand 0 against 1, hand against body part); `Grapple` fields (one test per field differing), self-grapple, range 0 and -1, DC held unchanged (no bound), `Condition`, constant 5; `Grapples` invariants, empty set, copied accessors, order-insensitive `Equal`, iteration order; `IsHolding`, `CanTarget` (not holding, same target, other target), `GrappleOf`, `HeldBy` |
| FR-022 to FR-028, FR-030 | unit | table 4.10.4; rejection order; input untouched; part reuse after release |
| FR-029, FR-031 to FR-036 | unit, application | release returns `GrappleEnding(released)`, `notHeld`; `ReleaseGrapple` works for an Incapacitated grappler with no availability; domain `Escape`; table 4.10.1; the enum has exactly two members and each `GrappleEscapeOption.Ability` equals `Skill.Ability` |
| FR-037 to FR-047, FR-105 | application | table 4.10.5 with the fixed roller: DC and attempt passed through, draws counted, no draw on a rejection, `ActionSpent` per FR-041, a failing roller's error returned wrapped once and no result; an architecture test shows the use case imports no use case |
| FR-048 to FR-061 | unit | table 4.10.2; `GrappleObservation`; `Review` rules and ending order; `ConditionEnded`; unknown and duplicate observations |
| FR-062 to FR-065, FR-093 | unit | tables 4.10.3 with the real `MovementCost`; size boundaries (Tiny, one, two sizes smaller, larger, Tiny grappler) |
| FR-066 to FR-069, FR-098 | application | each use case returns its closed result; equal commands give equal results; command unchanged; `Invalid` for same grappler and target and negative range; `DropGrappleRecord` applied (one `GrappleRecordDropped`, `condition_ended` increment) and rejected, no `ConditionSet` touched |
| FR-070 to FR-078 | application | recording telemetry double: span name, attribute keys and closed values, `srd5e.outcome` only `success` or `failure`, shared counters recorded once and none repeated, one `ended` increment per ending and per dropped record, result values each mapping to one class, `in_force` only for results with a set and not on a returned error, no error counter on a rejection or `Invalid`, one error record on a returned `*srderr.PortError` and none on an `srderr.ErrInvalidInput` error, no `srd5e.d20_test.*` signal from an escape, no log at `INFO`, identical results with the zero-value `telemetry.Noop`, no id or part name in any attribute, each `...OperationValue` paired with its `...Span` and a catalogue row per span |
| FR-079, FR-097 | architecture | domain files contain no logging, metric or span call; import scan of the grapple domain files against the header order |
| FR-080 to FR-084 | public API, unit, architecture | an external test (`package combat_test`) importing only public packages builds each command and switches exhaustively over each result and each `GrappleEvent` (no `default`); godoc on every export; the exported-name scan (check A7) finds exactly one `GrappleEnded`, `GrappleEndCause` and `GrappleHold` and no banned generic name; `GrappleEvent` is the only closed `Event` family here; `kind` constants unique; `combatants` is `[target, grappler]` and immutable, mutating a returned `Events` slice does not change the value; each `NewX` never panics (fuzz) and returns an error exactly for the invalid inputs, `MustX` panics exactly then; exactly two reason enums |
| FR-085 to FR-092, FR-095, FR-104 | unit | the four causes; `GrappleEnded` equality; table 4.10.2 with real `GrappleRules.Hold`; table 4.10.6: end for each cause, event order, grappler read from the condition, not-grappled case, plain `remove` emits no `GrappleEnded`, input set unchanged; `OptionSaveAbilities`; `Hold` with a negative range alone and with negative numbers (FR-087); `Hold` with `GrapplerDead` true, with and without Incapacitated and beyond the range, returns `grapplerIncapacitated` before the range test, and the default leaves FR-087 unchanged (FR-104; table 4.10.2 dead row through `Review`) |
| FR-094, FR-096 | unit | table 4.10.6 DC values (including a negative modifier) and the `Possibility` rows with the order of reasons |
| FR-099 to FR-103 | unit | table 4.10.7: each pair of problems in the stated order, for each `NewX` |
| scenario | scenario | through public packages only (`package scenarios_test`): an Unarmed Strike Grapple (`attacks.md`) whose save fails, `BeginGrapple`, the host applies `Grapple.Condition`, the grappler drags the target, the target fails then succeeds an escape check, the host passes the `GrappleEnding` to `GrappleRules.EndOnSet` and sees `ConditionRemoved` then one `GrappleEnded`; a second scenario in which the grappler becomes Incapacitated and `ReviewGrapples` ends the grapple |

- **Mutation testing:** the `grapple_*.go` domain files, `gomu` run only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`, threshold 80 (`doc/rules/testing.md` section 8). The boundary rows (range 5 against distances 5 and 6; range 0 against 0 and 1; DC 13 against totals 12 and 13; size one against two larger) kill comparison mutants; the `partInUse` against `targetAlreadyGrappled` rows and the both-fail rows kill order mutants; the two-carried rows kill a sum-versus-max mutant.
- **Not covered:** none. Every FR id appears above.

## 9. Open questions and decisions

Open: none. Nothing is escalated.

Settled, one line each:

- **Q-2 (distance):** the host reports it in feet in `GrappleObservation` (FR-048) or to `GrappleRules.Hold` (FR-087), only for grapples it chooses to check (FR-053); the engine models no map (architecture section 2). `Grapple.RangeFeet` is the grappler's range; the 5 foot default for an Unarmed Strike is package design (FR-011); a monster's range comes from its stat block (`content-monsters.md` FR-037). A later positions spec may call `ReviewGrapples` itself.
- **Q-4 (spending the escape action):** the escape is an improvised action: "can use its action to make a ... check" (p. 182) names no Standard action, and "you can improvise other actions" (p. 10) covers it, so the host spends `ImprovisedActionChoice` (`initiative-and-actions.md` FR-049, FR-150) when `ActionSpent` is true (FR-041). `EscapeGrappleFromRecord` never touches the tracker. It is not mapped to Utilize because the SRD does not say it is one (interpretation). `combat-action-resolution.md` Q-17 composes the pieces.
- **Q-5 (dragging several creatures):** sum over the creatures carried (FR-062). Interpretation: each Grappled condition states its own "every foot of movement costs it 1 extra foot" (p. 182) and the grappler is the one creature named by each, so the costs add. A host that reads it as "once per move" passes one carried creature.
- **Q-10 (death, fleeing, leaving):** only Incapacitated and distance end a grapple (p. 182); Unconscious includes Incapacitated (p. 191). The SRD ("Dead", p. 180) says a dead creature has no Hit Points and its spirit has left, and does not name Incapacitated, so `Hold` reads the condition literally, and the host reports death separately as `GrapplerDead` (FR-104). Interpretation, consistent with `combat-action-resolution.md` Q-15 (a dead grappler has `ConditionActionAvailability.None`) and `encounter-aggregate.md` FR-086: a dead creature cannot hold anyone, so `Review` ends the grapple. A dead target is handled the same way. A fleeing target is out of range (FR-050) once the host reports its distance.
- **Q-13 (monster escape DC field):** not this spec's; the host passes the DC from the stat block (FR-006, FR-009) and typing it on content is `content-monsters.md`'s concern.
- **Q-16 (reading the grappler in `EndOnSet`):** from the explicit `Grappled(grappler)` in the set's explicit-conditions view (`conditions.md` FR-012, FR-014), named `ConditionSet.Explicit`, because `ConditionRemoved` carries no source (FR-089). The signature `EndOnSet(set, target, cause)` is kept, so the encounter-aggregate calls (FR-086, FR-093, FR-094) stand. Amendment to `conditions.md` returned.

Settled earlier: Q-1 (host stores `Grapples`, section 1); Q-3, Q-14 and Q-12 (shared-types-and-ownership.md FR-029, FR-048, principle 4); Q-6 (one source, FR-014, FR-025); Q-7 and Q-8 (shared-types-and-ownership.md FR-012, FR-005, FR-046); Q-11 (`Invalid` results, `combat-foundations.md` FR-031); Q-15 (rank 23, table 4.11). Q-9: the record also ends when the Grappled condition is removed by a route this spec does not model (a spell, a Long Rest) or when applying `Grapple.Condition` after `BeginGrapple` fails (`ConditionImmune`, `ConditionAlreadyHeld`); the SRD says only that grappling "gives it the Grappled condition" (p. 182), so this is package design (FR-060, FR-061, FR-098).

## 10. Changelog impact

Under `Unreleased`, "Added":

- Grappling in the `combat` context: `Grapple`, `Grapples`, `GraspingPart` and its variants, `GrappleTransition` and its variants, `GrappleOfTarget`, `PartState`, `GrappleEnding`, `GrappleObservation`, the domain service `GrappleRules` (with `GrappleEscapeOption`, `GrappleHold`, `GrappleEndCause`, `HandAvailability`, `GrapplePossibility`, `GrappleEndOnSetResult`), the events in the closed family `GrappleEvent`, and the use cases `BeginGrapple`, `ReleaseGrapple`, `EscapeGrappleFromRecord`, `ReviewGrapples` and `DropGrappleRecord`.
- Telemetry signals `srd5e.combat.grapple.ended` and `srd5e.combat.grapple.in_force`, plus the five operations on the shared combat signals.

Not breaking: all new API in an unreleased package. `GrappleEvent` and the result types are closed; adding a variant after release would be breaking for consumers that switch over them (architecture section 7).
