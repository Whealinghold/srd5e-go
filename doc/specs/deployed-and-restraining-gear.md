# Deployed and restraining gear

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `observability-ports.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `d20-tests.md`, `resolve-d20-test-use-case.md`, `content-schema-and-decoder.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md`, `environment-hazards-and-objects.md` |

Dependency notes (prose, not dependencies): rank 24 in table 4.11 of `shared-types-and-ownership.md` (above `environment-hazards-and-objects.md`, rank 23; all dependencies are rank 23 or lower and no edge goes to rank 25). Uses `CombatantID`, `combat.Event`, `CombatTelemetry`, `CombatSignals` (`combat-foundations.md`); `Damage`, `DamageAffinities` (`hit-points-and-damage.md`); `ConditionSet`, `ConditionKind`, `ConditionActionAvailability` (`conditions.md`); `ObjectStatus`, `ObjectAffinities` (`environment-hazards-and-objects.md` FR-072 to FR-075); the pure `d20.ResolveCheck`, `d20.FailWithoutRoll` and the attempt types of `resolve-d20-test-use-case.md` (never `ResolveD20Test`, `shared-types-and-ownership.md` FR-044). Hand-off by fact only, no import: `initiative-and-actions.md` (round and active combatant for `TurnStamp`, Utilize and improvised actions), `attack-and-help-actions.md` FR-062 (the Net replaces an attack), `toolbox-traps.md` (Falling Net, rank 41, downstream: a `combat` spec cannot depend on `toolbox`).

## 1. Purpose and scope

Gear that is set out on the ground or used to bind a creature, from "Adventuring Gear" (SRD 5.2.1, pp. 96-99):

- **Deployed gear:** Caltrops, Ball Bearings, Oil poured on a 5-foot square (lit and burning) and the Hunting Trap: placement as a Utilize action, the area-entry rule ("enters this area for the first time on a turn", "only once per turn"), the save and its consequences, burning duration, recovery.
- **Holds:** the thrown Net, Chain, Manacles, Rope and the Hunting Trap's grip on a creature: the bind precondition (Grappled, Incapacitated or Restrained), the binder's check, which holds give the Restrained condition, the escape, burst, destroy and unlock routes with their DCs, the Manacles' Disadvantage on attack rolls, the Hunting Trap's chain limit and failure damage.

State in, new state and events out (`doc/rules/architecture.md` §4). Use cases **return consequences and never apply them**: `Damage` goes to `ApplyDamage`, `Restrained` and `Prone` to `ConditionSet.Apply`, a Net's damage to `ApplyDamageToObject`. The host stores `DeployedGears`, `EntryLedger` and `GearHolds`. Distances, sight, level ground and willingness are host facts (the engine models no map, `grappling.md` Q-2).

**Complements `toolbox-traps.md` Q-11.** The Net's statistics are `NetProfile` here (FR-044); a Falling Net hold is begun with `GearHolds.Begin` using the trap's escape DC (FR-033). `content-equipment.md` keeps the gear rows and verbatim text.

**Out of scope:**

- **Other thrown or applied gear** (Acid, Alchemist's Fire, Holy Water; Oil thrown at a creature or object, "covered in oil", p. 98; Basic Poison; Antitoxin; Grappling Hook; Climber's Kit; Lock and Thieves' Tools on a Lock, p. 98). No spec owns them yet (Amendment A-4).
- **Falling Net and Hunting-Trap-like traps of the Toolbox** (`toolbox-traps.md`); **Restrained, Prone, Grappled effects** (`conditions.md`); **damage handling** (`hit-points-and-damage.md`); **Utilize and Attack-action bookkeeping** (`initiative-and-actions.md`, `attack-and-help-actions.md`); **the Grapple itself** (`grappling.md`).
- **Binding a willing creature, untying Rope or Chain, bursting by a helper:** the SRD states no rule (Q-6, Q-7).
- **Anything not in SRD 5.2.1.** No core-looking rule was missing from the PDF.

## 2. Source

- SRD 5.2.1, "Adventuring Gear": Ball Bearings, Caltrops, Chain, p. 96; Hunting Trap (start), p. 97; Hunting Trap (end), Manacles, Net, Oil, p. 98; Rope, p. 99. Printed page equals PDF page.
- SRD 5.2.1, "D20 Tests", p. 6; "Saving Throws", p. 7; "Advantage/Disadvantage", p. 8; "Damage Rolls", p. 16; "Immunity", p. 183; "Round Down", p. 187; "Creature Size", p. 14; "Condition", p. 179; "Grappled", p. 182; "Incapacitated", p. 184; "Prone", "Restrained", pp. 186-187; "Unconscious", p. 191; "Utilize [Action]", p. 191; "Breaking Objects", p. 178; "Thieves' Tools", p. 94.
- Web research (informs interpretation only; nothing copied): community and publisher reading of "first time on a turn" as once per creature per turn, and of Rope binding as using the knot check (Q-1, Q-4).
- Package design: `doc/rules/architecture.md` §§2-6; `doc/rules/observability.md` §§2-7; `doc/rules/testing.md` §4.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Caltrops, Ball Bearings, Oil (dousing a space), Hunting Trap | `DeployedGearKind` (`DeployedGearKindCaltrops`, `DeployedGearKindBallBearings`, `DeployedGearKindPouredOil`, `DeployedGearKindHuntingTrap`) | Each has an `ItemID` (`content`, FR-114 there). |
| (a deployed item) | `DeployedGear`, `DeployedGearID`, `DeployedGears` | Package terms. Id is host-supplied, non-blank. |
| "5-foot-square area", "within 5 feet of yourself" | `DeployedGearProfile.AreaSideFeet`, `DeployReach` (`ReachWithin(feet)`, `ReachUnstated`) | Table 4.9a. |
| "level ground" | `GroundLevel` (`GroundLevelLevel`, `GroundLevelNotLevel`) | Host input. |
| "enters this area", "ends its turn there" | `AreaTrigger` (`AreaTriggerEnters`, `AreaTriggerEndsTurn`), `TurnStamp(round, combatant)`, `EntryLedger` | `TurnStamp` is the turn in progress (any creature's). |
| "If lit", "burns until the end of the turn 2 rounds from when the oil was lit" | `PouredOilState` (`Unlit`, `Burning(burnsThrough)`), `TurnStamp` | FR-016. |
| Net, Chain, Manacles, Rope, Hunting Trap (the grip) | `HoldKind` (`HoldKindNet`, `HoldKindChain`, `HoldKindManacles`, `HoldKindRope`, `HoldKindHuntingTrap`), `GearHold`, `HoldID`, `GearHolds` | Table 4.9b. |
| "bind an unwilling creature" | `BindingKind` (`chain`, `manacles`, `rope`), `Willingness` | Subset of `HoldKind`. |
| "If the creature's legs are bound", "attached to a chain or hook fixed in place" | `HoldRestraint` (`HoldRestraintRestrains`, `HoldRestraintDoesNotRestrain`) | Host states it; FR-040. |
| "Escaping", "Bursting", pick the lock, key | `HoldReleaseMethod` (`HoldReleaseMethodEscape`, `...Burst`, `...PickLock`, `...Key`), `HoldEndCause` (`HoldEndCauseEscaped`, `...Burst`, `...Destroyed`, `...Unlocked`) | |
| "the target or a creature within 5 feet of it" | `ReleaseActor` (`ActorIsHeld`, `ActorIsHelper{WithinReach bool}`) | Host states the reach. |
| Net "AC 10; 5 HP; Immunity to Bludgeoning, Poison, and Psychic" | `NetProfile` | FR-044. |
| Speed 0 "until the start of its next turn" | `GearSpeedZeroEffect` | Effect data; no condition. |

Package names, no SRD term: `GearEffect` and variants, `GearEntryOutcome`, `GearRejection`, `GearProblem`, `GearEvent`, the use cases and results. `GearHold.movementLimitFeet` is the "length of the chain (typically 3 feet)", so 3 is a default, not a rule value.

## 4. Requirements

Stable ids. "Profile" means table 4.9a or 4.9b. Distances are whole feet.

### 4.1 Profiles

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `DeployedGearKind` MUST have exactly four members (`DeployedGearKindCaltrops`, `DeployedGearKindBallBearings`, `DeployedGearKindPouredOil`, `DeployedGearKindHuntingTrap`) and `HoldKind` exactly five (`HoldKindNet`, `HoldKindChain`, `HoldKindManacles`, `HoldKindRope`, `HoldKindHuntingTrap`); `BindingKind` MUST be exactly `BindingKindChain`, `BindingKindManacles`, `BindingKindRope`, so no other kind can be expressed. | SRD 5.2.1, "Adventuring Gear", pp. 96-99 | unhappy (invalid input made unrepresentable) |
| FR-002 | `DeployedGearProfileFor(kind)` MUST return the facts of table 4.9a, and `HoldProfileFor(kind)` those of table 4.9b, as typed constants (no strings); a DC MUST be held exactly as printed and not bounded. | SRD 5.2.1, pp. 96-99 | happy |
| FR-003 | Each kind MUST expose its `ItemID` (`caltrops`, `ball_bearings`, `oil`, `hunting_trap`, `net`, `chain`, `manacles`, `rope`); a contract test checks each is an `equipment` entry of the bundled set. `huntingTrap` serves both enums. | SRD 5.2.1, pp. 96-99; `content-schema-and-decoder.md` FR-114 | happy |

### 4.2 Deploying and maintaining

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-004 | `DeployedGear` MUST carry id, kind, deployer and state and compare by value on all four. `NewDeployedGearID` MUST reject an empty or blank string with an error matching `srderr.ErrInvalidInput` (`MustDeployedGearID` panics with it). | SRD 5.2.1, "Caltrops", p. 96; architecture §6 | unhappy (invalid input) |
| FR-005 | `DeployedGears.Deploy` MUST add a gear when the id is unused, the distance is within the profile's reach and the ground rule holds, returning the new set and one `DeployedGearPlaced`. Initial state: `Armed` for the Hunting Trap, `Unlit` for Oil, `InForce` for Caltrops and Ball Bearings. | SRD 5.2.1, "Caltrops", "Ball Bearings", p. 96; "Hunting Trap", p. 97; "Oil", p. 98 | happy |
| FR-006 | Reach: `ReachWithin(n)` MUST accept a distance of 0 to n and refuse n + 1 (`outOfReach`): Caltrops 5, Ball Bearings 10, Oil 5. The Hunting Trap has `ReachUnstated`, so any distance of 0 or more is accepted. A negative distance is `GearProblem.negativeDistance` (invalid), never a refusal. | SRD 5.2.1, pp. 96-98 ("within 5 feet", "within 10 feet of yourself") | unhappy (boundary) |
| FR-007 | Ball Bearings and Oil MUST refuse `GroundLevel.notLevel` with `groundNotLevel` ("a level, 10-foot-square area"; "on level ground"). Caltrops and the Hunting Trap MUST accept either value (the SRD states no such rule; interpretation, Q-2). | SRD 5.2.1, "Ball Bearings", p. 96; "Oil", p. 98 | unhappy (rule interaction) |
| FR-008 | `AreaSideFeet` MUST be 5 for Caltrops and Oil, 10 for Ball Bearings, and the Hunting Trap has no area (it is triggered on its pressure plate). | SRD 5.2.1, pp. 96-98 | happy |
| FR-009 | `Deploy` with an id already in the set MUST refuse with `idInUse`, the set unchanged and no event. When several refusals apply the order is `idInUse`, `outOfReach`, `groundNotLevel` (package design). | package design | unhappy (state error) |
| FR-010 | `Recover` MUST remove a Caltrops or Ball Bearings gear and emit one `DeployedGearRecovered`; the profile carries `recoveryMinutes` 10 as data and the engine reads no clock. Any other kind MUST refuse with `notRecoverable`. | SRD 5.2.1, "Caltrops", "Ball Bearings", p. 96 ("It takes 10 minutes to recover") | happy; unhappy (state error) |
| FR-011 | `Light(gearID, stamp)` on `Unlit` Oil MUST make it `Burning(burnsThrough)` with `burnsThrough = TurnStamp(stamp.round + 2, stamp.combatant)` and emit `PouredOilLit`. Already lit: `alreadyLit`. Not Oil: `notPouredOil`. Unknown id: `unknownGear`. | SRD 5.2.1, "Oil", "Dousing a Space", p. 98 ("until the end of the turn 2 rounds from when the oil was lit (or 12 seconds)") | happy; unhappy (state error) |
| FR-012 | `Expire(endedTurn)` MUST burn out every `Burning` oil whose `burnsThrough` equals `endedTurn` or whose round is below `endedTurn.round`, removing it and emitting one `PouredOilBurnedOut` each in deployment order; unaffected gear is unchanged and an empty set returns itself with no events. | SRD 5.2.1, "Oil", p. 98; package design (determinism) | unhappy (boundary, empty and minimal) |
| FR-013 | A hunting trap that is `Sprung` and holds no creature MAY be reset by `Reset(gearID, holds)`, giving `Armed` and `HuntingTrapReset`; `Armed` refuses with `notSprung`; a `Sprung` trap is occupied, refusing with `trapOccupied`, exactly when some hold in `holds` has that `gearID` (FR-026; the engine derives it, no host boolean). Interpretation: setting it is again a Utilize action (Q-3). | SRD 5.2.1, "Hunting Trap", p. 97 ("you can set a Hunting Trap") | unhappy (state error) |
| FR-014 | `DeployedGears` MUST reject duplicate ids in its constructor with an error matching `srderr.ErrInvalidInput`, copy its input, expose views that are copies, compare equal regardless of insertion order and iterate in deployment order. | architecture §3; testing §4 | unhappy (immutability) |
| FR-015 | `NewTurnStamp(round, combatant)` MUST return an error matching `srderr.ErrInvalidInput` for a round below 1; equality is by both fields. | Package design (a round is a 1-based count); SRD 5.2.1, "Combat Step by Step", p. 13 (the round concept only) | unhappy (boundary) |

### 4.3 Area entry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-016 | `AreaEntryRules.Assess(gear, creature, trigger, stamp, ledger)` MUST be pure. For Caltrops and Ball Bearings only `AreaTrigger.enters` applies; `endsTurn` refuses with `triggerNotApplicable`. For lit Oil both apply. For the Hunting Trap only the step onto the plate applies, modelled as `enters`. | SRD 5.2.1, pp. 96-98 | happy |
| FR-017 | Caltrops and Ball Bearings: the first `enters` by a creature on a `TurnStamp` MUST require the save; a second by the same creature on the same stamp MUST refuse `alreadyTriggeredThisTurn`; the same creature on another stamp (another round, or another combatant's turn) MUST require it again. Interpretation: "a turn" is the turn in progress, whoever's (Q-1). | SRD 5.2.1, "Caltrops", "Ball Bearings", p. 96 ("for the first time on a turn") | unhappy (rule interaction) |
| FR-018 | The ledger key MUST be gear id, creature and stamp. Two creatures, or two gears, on one stamp MUST NOT share an entry. `EntryLedger.Record` returns a new ledger; the ledger is copied (returned slices are copies), value-equal and independent of insertion order. | SRD 5.2.1, p. 96; architecture §3 | unhappy (rule interaction, immutability) |
| FR-019 | Caltrops: save Dexterity DC 15; failure MUST give `Damage(1, piercing)` and `GearSpeedZeroEffect`; success MUST give no effect. | SRD 5.2.1, "Caltrops", p. 96 | happy |
| FR-020 | Ball Bearings: save Dexterity DC 10; failure MUST give `GearConditionEffect(ConditionKind.prone)`; success none. If the creature already has Prone the host's `ConditionSet.Apply` reports it (`conditions.md` FR-018); no rule here. | SRD 5.2.1, "Ball Bearings", p. 96 | happy; unhappy (rule interaction) |
| FR-021 | Lit Oil: a creature that enters, or ends its turn in, the area MUST take `Damage(5, fire)` with no save; "only once per turn" MUST share one ledger entry across `enters` and `endsTurn` on a stamp. An `endsTurn` trigger whose stamp combatant is not the creature MUST be `GearProblem.endTurnStampMismatch` (invalid). | SRD 5.2.1, "Oil", "Dousing a Space", p. 98 | happy; unhappy (rule interaction, invalid input) |
| FR-022 | `Unlit` Oil, Oil whose `burnsThrough.round` is below the stamp's round, and a Hunting Trap already `Sprung` MUST refuse with `oilNotBurning`, `oilNotBurning` and `trapAlreadySprung`; nothing is rolled or recorded. Precondition: oil lit at (r, A) burns through the end of A's turn in round r + 2, so an entry at (r + 2, B) after A's turn is `oilNotBurning` only if the host has called `Expire((r + 2, A))`; without it the oil is still in the set and still counts as burning (Q-9). | SRD 5.2.1, "Oil", p. 98 ("If lit"); "Hunting Trap", p. 97 | unhappy (state error) |
| FR-023 | Hunting Trap: save Dexterity DC 13. Failure MUST give `Damage(d4 rolled, piercing)`, `GearSpeedZeroEffect` and `GearCaughtEffect{Hold}` where `hold` is `HoldKind.huntingTrap` on that creature with the trap's `gearID` (escape DC 13, `doesNotRestrain`, `movementLimitFeet` defaulting to 3, the SRD's "typically 3 feet", a default the host may override as FR-033 allows for a hold's escape DC); success none. Either way the trap becomes `Sprung` with one `HuntingTrapSprung`. Interpretation: the plate "snaps shut" on any step (Q-3). | SRD 5.2.1, "Hunting Trap", pp. 97-98 | happy; unhappy (boundary) |
| FR-024 | A save total equal to the DC MUST succeed and one below MUST fail; the comparison is `d20.ResolveCheck`'s. `ChooseToFail` and `AutomaticFailure` attempts MUST fail with a `ForcedFailureResult` and draw no d20. | SRD 5.2.1, "D20 Tests", p. 6; "Saving Throws", p. 7 | unhappy (boundary, rule interaction) |
| FR-025 | The dice requested MUST be only the save's d20s (`RollMode.resolve`, `resolve-d20-test-use-case.md` FR-007 shape) and, for a failed Hunting Trap save, exactly one `1d4` after them; a saved Hunting Trap or Caltrops request no damage die. | SRD 5.2.1, "Hunting Trap", pp. 97-98; "Advantage/Disadvantage", p. 8 | unhappy (determinism) |

### 4.4 Holds

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-026 | `GearHold` MUST carry id, kind, target, escape DC, `HoldRestraint`, `gearID` and `movementLimitFeet` (both Hunting Trap only: required there, else absent) and, for a Net, its `ObjectStatus`; it MUST compare by value on all. Construction MUST return an error matching `srderr.ErrInvalidInput` for a Net with `doesNotRestrain`, a Hunting Trap with `restrains`, a negative limit, a status on a non-Net, and a missing `gearID` on a Hunting Trap or one on any other kind; `NewGearHold` returns an error carrying `GearProblem` exactly then. | SRD 5.2.1, "Net", p. 98; "Hunting Trap", pp. 97-98 | unhappy (invalid input) |
| FR-027 | `GearHolds` MUST allow one hold per target and kind; a second hold of the same kind on a target MUST return an error matching `srderr.ErrInvalidInput` in the constructor and `begin` MUST refuse it with `targetAlreadyHeld`. Exception: the key for a Hunting Trap hold is (target, trap `gearID`), so a creature caught in one trap may also be caught by a different trap (Q-10). Different kinds on one target are allowed (a Net and Manacles). Hold ids are unique. A `GearCaughtEffect{Hold}` therefore always begins, since the same trap cannot spring twice (FR-022). | SRD 5.2.1, "Condition", p. 179 (no stacking of the same condition); package design | unhappy (state error) |
| FR-028 | `Begin(hold)` MUST return `GearHoldsApplied(holds, events, restraintBegins)` with one `GearHoldStarted`. `restraintBegins` MUST be true exactly when the hold restrains and no other hold in the set already restrains that target; the host then applies `ConditionKind.restrained`. | SRD 5.2.1, "Net", "Chain", pp. 96, 98 ("Restrained condition until it escapes") | happy |
| FR-029 | `End(holdID, cause)` MUST remove the hold, emit one `GearHoldEnded(cause)` and set `restraintEnds` true exactly when the hold restrained and no other hold in the set still restrains the target; the host then removes Restrained through `ConditionSet.Remove`. Another Restrained source outside the set is the host's concern (Q-5). Unknown id: `unknownHold`. | SRD 5.2.1, "Net", p. 98 ("until it escapes"); "Restrained", p. 187 | happy; unhappy (rule interaction) |
| FR-030 | `GearHolds.RestrainsTarget(target)` MUST be true when any hold on the target restrains. A hold with `doesNotRestrain` (Rope or Chain on arms, loose Manacles, the Hunting Trap) MUST NOT. | SRD 5.2.1, "Chain", "Rope", "Manacles", pp. 96, 98, 99 | happy |
| FR-031 | `AttackRollCircumstances(target)` MUST return `[Disadvantage]` while a `manacles` hold is on the target (restraining or not) and an empty list otherwise; it adds nothing for the other kinds. `combat-action-resolution.md` merges it with `ConditionEffects` through `RollMode.resolve`. | SRD 5.2.1, "Manacles", p. 98 ("While bound, a creature has Disadvantage on attack rolls") | happy; unhappy (rule interaction) |
| FR-032 | `Reanchor(holdID, restraint)` MUST change a Manacles hold's `HoldRestraint`, emit `GearHoldRestraintChanged`, and report `restraintBegins` or `restraintEnds` by FR-028 and FR-029. Other kinds refuse `methodNotAvailable`; no change refuses nothing and emits nothing. | SRD 5.2.1, "Manacles", p. 98 ("if the Manacles are attached to a chain or hook that is fixed in place") | unhappy (state error) |
| FR-033 | A hold's escape DC MUST default to the profile and MAY be supplied (a Falling Net hold uses the trap's DC, `toolbox-traps.md` FR-030). A DC of any integer MUST be held as given. | SRD 5.2.1, "Net", p. 98; "Falling Net", p. 199 | unhappy (boundary) |
| FR-034 | `GearHolds` MUST copy its input, expose views that are copies, compare equal regardless of insertion order, iterate in the order begun and answer `HoldsOn(target)` and `HoldOf(id)` (the latter returns the hold and an `ok` boolean) (the zero `GearHold` is invalid, so absence is the `ok` boolean, never a nil pointer). | architecture §3, §4 | unhappy (immutability) |

### 4.5 Thrown Net

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-035 | `NetRules.Possibility(distanceFeet, sight)` MUST refuse `targetNotSeen` when the thrower cannot see the target and `outOfReach` beyond 15 feet; 15 feet is accepted. `targetNotSeen` is reported before `outOfReach`. The attack replacement is recorded by `AttackActionProgress.recordReplacement` (`attack-and-help-actions.md` FR-062); the host owns "you can see". | SRD 5.2.1, "Net", p. 98 ("a creature you can see within 15 feet") | unhappy (boundary, rule interaction) |
| FR-036 | `NetRules.SaveDC(abilityModifier, proficiencyBonus)` MUST return `DifficultyClass(8 + abilityModifier + proficiencyBonus)` with no cap or floor; the host passes the thrower's Dexterity modifier. | SRD 5.2.1, "Net", p. 98 ("DC 8 plus your Dexterity modifier and Proficiency Bonus") | happy; unhappy (boundary) |
| FR-037 | The target's saving throw is Dexterity. A failure MUST produce a Net hold (`restrains`, escape DC 10 unless overridden); a success MUST produce none. | SRD 5.2.1, "Net", p. 98 | happy |
| FR-038 | A target of size Huge or Gargantuan MUST auto-succeed: no d20, any supplied attempt ignored, no hold. Large and smaller MUST save. | SRD 5.2.1, "Net", p. 98 ("The target succeeds automatically if it is Huge or larger"); "Creature Size", p. 14 | unhappy (boundary) |
| FR-039 | A target already held by a Net MUST refuse the throw with `targetAlreadyHeld` before any roll. | FR-027; package design | unhappy (state error) |
| FR-040 | The `HoldRestraint` of a new hold MUST come from the command: Chain and Rope `restrains` only when the host states the legs are bound; Manacles `restrains` only when the host states they are attached to a fixed chain or hook; a Net always `restrains` (FR-026). | SRD 5.2.1, "Chain", p. 96; "Manacles", p. 98; "Rope", p. 99 | happy; unhappy (rule interaction) |
| FR-041 | The Net escape: the held creature or a creature within 5 feet of it MUST make a Strength (Athletics) check, DC 10, as an action; success ends the hold with `escaped`. A helper beyond 5 feet (`ActorIsHelper{WithinReach: false}`) MUST be refused `helperOutOfReach`. | SRD 5.2.1, "Net", p. 98 | happy; unhappy (boundary) |
| FR-042 | Destroying the Net: `GearHolds.RecordNetDamage(holdID, status)` MUST store an updated, undestroyed status, and for a destroyed status MUST end the hold with `destroyed` (`restraintEnds` by FR-029). A non-Net hold refuses `notDestructible`. Damage itself is `ApplyDamageToObject`'s. | SRD 5.2.1, "Net", p. 98 ("Destroying the Net ... also frees the target"); "Breaking Objects", p. 178 | happy; unhappy (state error) |
| FR-043 | A stored status for a Net hold MUST be a destroyed-or-not `ObjectStatus`; recording a destroyed status twice, or on an absent hold, MUST refuse `unknownHold` the second time. | `environment-hazards-and-objects.md` FR-077 | unhappy (state error) |
| FR-044 | `NetProfile.Status()` MUST return an `ObjectStatus` with Armor Class 10, current and maximum Hit Points 5, Temporary Hit Points 0, no Damage Threshold, and affinities `ObjectAffinities` built by its standard constructor with the ruling Immunity to Bludgeoning (so Bludgeoning, Poison and Psychic; `environment-hazards-and-objects.md` FR-074). The size is Medium; no Net rule reads it (interpretation, Q-8). A contract test MUST check `NetProfile` equals the `net` entry of `content-equipment.md` once that entry exists (one source of truth for the numbers; `toolbox-traps.md` Q-11). | SRD 5.2.1, "Net", p. 98 ("AC 10; 5 HP; Immunity to Bludgeoning, Poison, and Psychic damage"); "Breaking Objects", p. 178 | happy |

### 4.6 Binding with Chain, Manacles and Rope

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-045 | `BindingRules.Assess` MUST refuse in this order (package design; the SRD gives none): `targetWilling` (the rules say "unwilling"; it wins over every later refusal, also for a willing Unconscious target), `outOfReach` (beyond 5 feet; Chain and Manacles only, FR-049), `sizeNotAllowed` (Manacles: not Small or Medium), `targetNotHeldDown`, `targetAlreadyHeld`. Otherwise the binder's check is made. | SRD 5.2.1, "Chain", "Manacles", "Rope", pp. 96, 98, 99 | unhappy (rule interaction) |
| FR-046 | `targetConditions` MUST be the target's effective `ConditionSet` (`conditions.md`), whose implied kinds are already expanded by the set. `targetNotHeldDown` MUST hold unless the target's effective conditions include Grappled, Incapacitated or Restrained; Unconscious, Paralyzed, Petrified and Stunned qualify through their implied Incapacitated. | SRD 5.2.1, "Chain", p. 96; "Condition", p. 179; `conditions.md` FR-026 | happy; unhappy (rule interaction) |
| FR-047 | The binder's check MUST be Strength (Athletics) DC 13 for Chain, Dexterity (Sleight of Hand) DC 13 for Manacles and Dexterity (Sleight of Hand) DC 10 for Rope; a success MUST begin a hold, a failure none. Interpretation for Rope: binding a creature is tying a knot (Q-4). | SRD 5.2.1, "Chain", p. 96; "Manacles", p. 98; "Rope", p. 99 | happy; unhappy (boundary) |
| FR-048 | Size: Manacles MUST accept Small and Medium and refuse Tiny, Large, Huge and Gargantuan. Chain and Rope have no size rule. | SRD 5.2.1, "Manacles", p. 98 ("unwilling Small or Medium creature"); "Creature Size", p. 14 | unhappy (boundary) |
| FR-049 | Bind reach: for Chain and Manacles 5 feet is accepted and 6 refused. Rope states no distance (p. 99), so the engine does not check one for Rope (`distanceFeet` ignored, host fact; interpretation, Q-13). | SRD 5.2.1, "Chain", "Manacles", pp. 96, 98 ("within 5 feet of yourself"); "Rope", p. 99 | unhappy (boundary) |

### 4.7 Escaping, bursting and unlocking

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-050 | `HoldRules.Available(kind)` MUST give: `escape` for all five; `burst` for Chain, Manacles and Rope only; `pickLock` and `key` for Manacles only. Others refuse `methodNotAvailable`. | SRD 5.2.1, pp. 96-99 | unhappy (invalid input) |
| FR-051 | Escape and burst checks MUST be (ability, skill, DC), table 4.9b: escape Net Strength (Athletics) 10, Chain Dexterity (Acrobatics) 18, Manacles Dexterity (Sleight of Hand) 20, Rope Dexterity (Acrobatics) 15, Hunting Trap Strength (Athletics) 13; burst Chain 20, Manacles 25, Rope 20 (all Strength (Athletics)); pick the lock Dexterity (Sleight of Hand) 15. The ability is `Skill.ability`; this spec holds no second mapping. | SRD 5.2.1, pp. 96-99 | happy |
| FR-052 | Escape and burst take an action. The actor MUST be the held creature, except for a Net and a Hunting Trap where another creature within reach may act (FR-041; Hunting Trap: "freeing itself or another creature within its reach"). Burst and escape by a helper for Chain, Manacles and Rope are refused `helperNotAllowed` (Q-7). The Rope burst states no action cost (p. 99); treating it as an action like Chain and Manacles is an interpretation (Q-7). | SRD 5.2.1, "Net", "Hunting Trap", p. 98; "Chain", p. 96 ("as an action"); "Manacles", p. 98; "Rope", p. 99 | unhappy (rule interaction) |
| FR-053 | An actor whose `ConditionActionAvailability` is `none` MUST be refused `noAction` for escape, burst, pickLock and key, with no die rolled. `key` takes no check; the SRD is silent on its action cost, so requiring an actor able to act is an interpretation (Q-12). | SRD 5.2.1, "Incapacitated", "Inactive", p. 184 | unhappy (state error) |
| FR-054 | The check composes `d20.ResolveCheck` (with `d20.AbilityCheck`; a forced failure is `d20.FailWithoutRoll`) over the roll, modifiers and DC with the hold's DC and the attempt's modifiers unchanged; the use case adds no modifier. Success ends the hold (`escaped`, `burst`, `unlocked`); a total equal to the DC succeeds. | SRD 5.2.1, "D20 Tests", p. 6 | happy; unhappy (boundary) |
| FR-055 | `key` MUST end a Manacles hold as `unlocked` with no check and no die; `pickLock` makes the DC 15 check by any creature the host states has Thieves' Tools (`hasThievesTools`), else it MUST refuse `toolsMissing` with no die rolled; the tool's proficiency arrives only through the attempt's modifiers. Who holds the key is the host's (Q-12). | SRD 5.2.1, "Manacles", p. 98 ("comes with a key"; "Thieves' Tools to pick the Manacles' lock") | happy |
| FR-056 | A failed attempt MUST leave the hold unchanged with `actionSpent` true. A failed Hunting Trap escape MUST also return `Damage(1, piercing)` for the trapped creature. A failed check on any other hold returns no damage. | SRD 5.2.1, "Hunting Trap", pp. 97-98 ("Each failed check deals 1 Piercing damage") | unhappy (boundary) |
| FR-057 | An `AutomaticFailure` attempt MUST fail with the `ForcedFailureResult`, draw no d20, spend the action and (Hunting Trap) still deal the failure damage. | SRD 5.2.1, "D20 Tests", p. 6 | unhappy (rule interaction) |
| FR-058 | `actionSpent` MUST be true for a rolled or forced attempt (including the Rope burst, FR-052) and false for a refusal or a `key`; how the host spends the action is not decided here (improvised action, `grappling.md` Q-4). | SRD 5.2.1, "Chain", p. 96; "Rope", p. 99 (interpretation) | happy |

### 4.8 Use cases, events, telemetry

Telemetry follows `combat-foundations.md` FR-027 to FR-041 through `CombatTelemetry`; the domain is silent.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-059 | Six use cases: `DeployGear`, `MaintainDeployedGear` (closed `GearMaintenance`: `LightOil`, `RecoverGear`, `ResetHuntingTrap`, `ExpireBurningOil`), `ResolveGearEntry`, `ThrowNet`, `BindCreature`, `ReleaseFromHold`. Each takes `<UseCase>Command` and returns its own closed result with operation-prefixed variants and reasons as fields (`GearRejection`, `GearProblem`). No use case calls another (`shared-types-and-ownership.md` FR-044). | `shared-types-and-ownership.md` FR-017; `combat-foundations.md` FR-022, FR-023 | happy |
| FR-060 | Results: `DeployGearPlaced`, `...Rejected`, `...Invalid`; `MaintainDeployedGearApplied`, `...Rejected`; `ResolveGearEntrySaved`, `...Failed`, `...Damaged`, `...Rejected`, `...Invalid`; `ThrowNetHeld`, `...Saved`, `...AutoSucceeded`, `...Rejected`; `BindCreatureBound`, `...Failed`, `...Rejected`; `ReleaseFromHoldReleased`, `...Held`, `...Rejected`. Each carries the new state values it changes, `events`, and (rolling ones) the `CheckResult` or `ForcedFailureResult`. | `combat-foundations.md` FR-022 | happy |
| FR-061 | `ResolveGearEntry`, `ThrowNet`, `BindCreature` and `ReleaseFromHold` take a `dice.Roller`; the other two take only `telemetry.Telemetry`. A returned `*srderr.PortError` MUST propagate unchanged with no partial result, recorded once by `CombatTelemetry`; a wrong result is a `*srderr.PortError` matching `srderr.ErrContractViolation` from `dice.Roll`. | `dice-and-rolls.md` FR-050 to FR-053; `combat-foundations.md` FR-032 | unhappy (failure of collaborator) |
| FR-062 | Equal commands with equal fixed dice MUST give equal results and leave the command unchanged; results MUST be equal with the zero-value no-op and a recording `telemetry.Telemetry`. | testing §4; observability §9 | unhappy (determinism) |
| FR-063 | Events (closed `GearEvent` implementing `combat.Event`, constant `kind`, unique, immutable, value-equal, past tense): `DeployedGearPlaced` `deployed_gear_placed`, `DeployedGearRecovered` `deployed_gear_recovered`, `PouredOilLit` `poured_oil_lit`, `PouredOilBurnedOut` `poured_oil_burned_out`, `HuntingTrapSprung` `hunting_trap_sprung`, `HuntingTrapReset` `hunting_trap_reset`, `GearHoldStarted` `gear_hold_started`, `GearHoldRestraintChanged` `gear_hold_restraint_changed`, `GearHoldEnded` `gear_hold_ended`. `combatants` is the creatures concerned, deployer first, copied (returned slices are copies), no duplicates; events lists are copied (returned slices are copies). | `combat-foundations.md` FR-015 to FR-021; `shared-types-and-ownership.md` FR-019 | happy |
| FR-064 | Operations (`snake_case`, span `srd5e.combat.<operation>`): `deploy_gear`, `maintain_deployed_gear`, `resolve_gear_entry`, `throw_net`, `bind_creature`, `release_from_hold`. `CombatSignals` gains an operation-name constant and span-name constant each. | `combat-foundations.md` FR-028, FR-040 | happy |
| FR-065 | Result classes: the result variants (each exported with its operation prefix, errors-and-results.md FR-041: `DeployGearPlaced`, `ThrowNetFailed`, and so on) `Placed`, `Applied`, `Saved`, `Failed`, `Damaged`, `Held`, `AutoSucceeded`, `Bound`, `Released` are `applied` (a saved or failed roll is a normal outcome); `Rejected` is `refused`; `Invalid` is `invalid`. `srd5e.outcome` is `success` or `failure` only. No new shared counter is defined. | `combat-foundations.md` FR-030, FR-031; `shared-types-and-ownership.md` FR-041 | happy |
| FR-066 | Each `GearEntry` assessed to a roll or a damage MUST increment `srd5e.combat.gear.entries` once with `srd5e.combat.gear.kind` and `srd5e.combat.gear.entry` (`save_failed`, `save_passed`, `damaged`); a refusal increments it with `already_triggered`, `not_active` or `not_applicable`. | `doc/rules/observability.md` §5 | happy |
| FR-067 | Each `GearHoldEnded` returned by a use case MUST increment `srd5e.combat.gear.hold_ended` once with `srd5e.combat.gear.kind` and `srd5e.combat.gear.end_cause`; each result carrying `GearHolds` MUST record its size in `srd5e.combat.gear.holds_in_force` with `srd5e.combat.operation`. | `doc/rules/observability.md` §§4, 5 | happy |
| FR-068 | Attributes MUST come from closed sets and carry no ids, DCs, distances or free text; refusals and `Invalid` are not errors (no `srd5e.combat.errors`, no log above `FINE`); at most one `FINE` log `'gear operation'` per call. | `combat-foundations.md` FR-033, FR-034; observability §7 | unhappy (telemetry) |
| FR-069 | The gear domain files MUST contain no logging, metric or span call and no observability import; they MUST import only the kernel, the `content` and `d20` packages and the `combat` domain files of the specs in "Depends on". | architecture §9; `shared-types-and-ownership.md` FR-013 | unhappy (forbidden edge) |
| FR-070 | Package `combat` MUST export every type of section 5 and 6 with a doc comment and no generic variant name; names are unique across the package (`Net` alone is not exported: it is `NetProfile`, `NetRules`). | architecture §§4, 7, 9 | happy |
| FR-072 | The signals catalogue `doc/observability/signals.md` MUST gain an entry for each of `srd5e.combat.gear.entries`, `.hold_ended`, `.holds_in_force` and the six operation spans in the same change as the code; a doc-consistency check (architecture test, if one exists for the catalogue) MUST find them. | `doc/rules/observability.md` §2; `combat-foundations.md` FR-027 | happy |
| FR-071 | Each constructor (`NewDeployedGearID`, `NewGearHold`, `NewTurnStamp`) MUST never panic for any input and MUST return an error (matching `srderr.ErrInvalidInput`) for exactly the invalid cases; the `MustX` form panics with that error. `GearProblem` is `negativeDistance`, `negativeRound`, `blankId`, `endTurnStampMismatch`, `holdKindMismatch`, `negativeMovementLimit`, `gearIdMismatch`; `GearProblem` is the typed reason carried by the returned `*GearError` and by the `...Invalid` result variants. | `errors-and-results.md` FR-025, FR-035 | unhappy (invalid input) |

### 4.9 Tables

**4.9a Deployed gear** (FR-002, FR-005 to FR-011, FR-016 to FR-023; one row one test)

| Kind | Reach | Side | Level ground | Trigger | Save | Failure | Recover | Source |
|---|---|---|---|---|---|---|---|---|
| `caltrops` | within 5 | 5 | no | `enters`, first on a turn | Dexterity 15 | 1 Piercing; Speed 0 until start of its next turn | 10 min | p. 96 |
| `ballBearings` | within 10 | 10 | yes | `enters`, first on a turn | Dexterity 10 | Prone | 10 min | p. 96 |
| `pouredOil` | within 5 | 5 | yes | lit: `enters` or `endsTurn`, once per turn | none | 5 Fire | no | p. 98 |
| `huntingTrap` | unstated | none | no | step on plate (`enters`) | Dexterity 13 | 1d4 Piercing; Speed 0 until start of its next turn; caught | no | pp. 97-98 |

**4.9b Holds** (FR-026, FR-040 to FR-052)

| Kind | Applied by | Escape (actor) | Burst | Restrains | Extra | Source |
|---|---|---|---|---|---|---|
| `net` | target Dexterity save, DC 8 + Dex + PB; Huge or larger auto | Strength (Athletics) 10 (held or within 5 ft) | none | always | AC 10, 5 HP; destroying frees | p. 98 |
| `chain` | Strength (Athletics) 13 | Dexterity (Acrobatics) 18 (held) | Strength (Athletics) 20 | if legs bound | needs Grappled, Incapacitated or Restrained | p. 96 |
| `manacles` | Dexterity (Sleight of Hand) 13; Small or Medium | Dexterity (Sleight of Hand) 20 (held) | Strength (Athletics) 25 | if attached to fixed chain or hook | Disadvantage on attack rolls; key or pick DC 15 | p. 98 |
| `rope` | Dexterity (Sleight of Hand) 10 (interpretation); no reach check (FR-049); burst costs an action (interpretation) | Dexterity (Acrobatics) 15 (held) | Strength (Athletics) 20 | if legs bound | needs Grappled, Incapacitated or Restrained | p. 99 |
| `huntingTrap` | the entry save (FR-023) | Strength (Athletics) 13 (held or within reach) | none | never | failed check 1 Piercing; chain limit typically 3 ft (default, host may override) | pp. 97-98 |

**4.9.1 Once-per-turn (FR-017, FR-018, FR-021)**

| Label | Gear | Sequence (creature, stamp) | Expected | Path |
|---|---|---|---|---|
| first entry | caltrops | A enters at (1, A) | save required | happy |
| second entry same turn | caltrops | A enters, leaves, enters at (1, A) | second `alreadyTriggeredThisTurn` | unhappy (rule interaction) |
| next round | caltrops | A at (1, A), then (2, A) | save both times | unhappy (boundary) |
| other creature's turn | caltrops | A at (1, A), then (1, B) | save both times | unhappy (rule interaction) |
| two creatures | caltrops | A and B at (1, A) | both save | unhappy (rule interaction) |
| oil enter then end | lit oil | A enters then ends turn at (1, A) | 5 Fire once | unhappy (rule interaction) |
| oil, other's turn then own | lit oil | A enters at (1, B), ends turn at (1, A) | 5 Fire twice | unhappy (rule interaction) |
| caltrops end turn | caltrops | A `endsTurn` | `triggerNotApplicable` | unhappy (invalid input) |
| oil unlit | oil | A enters | `oilNotBurning` | unhappy (state error) |
| oil expired | oil burns through (3, A) | A enters at (4, B) | `oilNotBurning` | unhappy (boundary) |
| oil, after lighter's turn, expired | oil burns through (3, A); `Expire((3, A))` called | C enters at (3, B) | `oilNotBurning` | unhappy (boundary) |
| oil, after lighter's turn, host forgot | oil burns through (3, A); no `Expire` | C enters at (3, B) | 5 Fire (still burning) | unhappy (state error) |
| second trap | trap 1 holds A; A steps on trap 2 and fails | trap 2 hold begins (key target + `gearID`) | both holds in force | unhappy (rule interaction) |
| reset occupied / free | hunting trap | `Sprung`, hold with its `gearID` / none | `trapOccupied` / `Armed` | unhappy (state error) |

**4.9.2 Burning and boundaries (FR-006, FR-011, FR-012, FR-036, FR-038, FR-048, FR-049)**

| Label | Input | Expected | Path |
|---|---|---|---|
| lit in round 1 by A | `light` at (1, A) | `burnsThrough` (3, A) | happy |
| expire at (3, A) | `Expire((3, A))` | burnt out | unhappy (boundary) |
| expire at (3, B) | `Expire((3, B))` | still burning | unhappy (boundary) |
| expire at (4, B) | `Expire((4, B))` | burnt out | unhappy (boundary) |
| reach 5 / 6 | caltrops distance 5 / 6 | placed / `outOfReach` | unhappy (boundary) |
| reach 10 / 11 | ball bearings | placed / `outOfReach` | unhappy (boundary) |
| not level | ball bearings, `notLevel` | `groundNotLevel` | unhappy (state error) |
| net DC | Dex +3, PB +2 / Dex -1, PB +2 | DC 13 / DC 9 | happy; unhappy (boundary) |
| net size | Large / Huge / Gargantuan | save / auto / auto | unhappy (boundary) |
| manacles size | Tiny / Small / Medium / Large | refused / ok / ok / refused | unhappy (boundary) |
| net 15 / 16 ft | distance | ok / `outOfReach` | unhappy (boundary) |

## 5. Model

Domain and use cases in package `combat`, one file per concept (`gear_*.go` for the domain; `*_usecase.go` for the use cases). All domain types are immutable, pure, synchronous, silent.

- **Value objects:** `DeployedGearID`, `DeployedGear`, `DeployedGears`, `PouredOilState` (closed `Unlit`, `Burning`), `DeployedGearState` (closed `InForce`, `Armed`, `Sprung`, plus the oil states), `TurnStamp`, `EntryLedger`, `GearHold`, `HoldID`, `GearHolds`, `NetProfile`, profile types, `GearEffect` (closed `GearDamageEffect(Damage)`, `GearConditionEffect(ConditionKind)`, `GearSpeedZeroEffect`, `GearCaughtEffect(GearHold)`).
- **Enums:** `DeployedGearKind`, `HoldKind`, `BindingKind`, `GroundLevel`, `AreaTrigger`, `HoldRestraint`, `HoldReleaseMethod`, `HoldEndCause`, `Willingness`, `GearRejection` (`outOfReach`, `groundNotLevel`, `idInUse`, `unknownGear`, `unknownHold`, `notRecoverable`, `notPouredOil`, `alreadyLit`, `notSprung`, `trapOccupied`, `alreadyTriggeredThisTurn`, `oilNotBurning`, `trapAlreadySprung`, `triggerNotApplicable`, `targetNotSeen`, `targetAlreadyHeld`, `targetWilling`, `sizeNotAllowed`, `targetNotHeldDown`, `methodNotAvailable`, `helperNotAllowed`, `helperOutOfReach`, `noAction`, `notDestructible`, `toolsMissing`), `GearProblem` (FR-071).
- **Domain services (stateless; zero-value structs):** `AreaEntryRules` (FR-016 to FR-025), `NetRules` (FR-035 to FR-039), `BindingRules` (FR-045 to FR-049), `HoldRules` (FR-050 to FR-058).
- **Closed outcomes:** `GearEntryOutcome` (`GearEntrySaved`, `GearEntryFailed(effects)`, `GearEntryDamaged(effects)`); transitions `GearHoldsApplied`, `DeployedGearsApplied`, each with `...Rejected(GearRejection)`.
- **Invariants and where enforced:** unique ids and one hold per target and kind (collection constructors); kind-consistent `HoldRestraint` (`GearHold`); round at least 1 (`TurnStamp`). Constructors return an error matching `srderr.ErrInvalidInput`; every host-supplied value goes through a `NewX` constructor.
- **Go naming:** members written lowerCamel above (`areaSideFeet`, `movementLimitFeet`, `gearID`, `actionSpent`, `restraintBegins`, `restraintEnds`, `burnsThrough`) are exported read-only accessors or fields in Go (`AreaSideFeet`, `MovementLimitFeet`, `GearID`); operations written lowerCamel (`deploy`, `recover`, `light`, `expire`, `reset`, `assess`, `record`, `begin`, `end`, `reanchor`, `possibility`, `saveDc`, `available`, `restrainsTarget`, `holdsOn`, `holdOf`, `recordNetDamage`) are exported methods (`Deploy`, `SaveDC`, `RestrainsTarget`); `X.of(kind)` is the function `XFor(kind)`. Enum members and rejection reasons written lowerCamel (`caltrops`, `ballBearings`, `outOfReach`, `targetAlreadyHeld`, `doesNotRestrain`, `notLevel`) are exported typed constants prefixed by their type (`DeployedGearKindCaltrops`, `GearRejectionOutOfReach`, `HoldRestraintDoesNotRestrain`) with a `String()` method, and snake_case only in attribute values and `ItemID`s. Function-call notation with named arguments (`f(a: x)`) means positional arguments or a command struct; sum-type constructors written `Name(x)` are exported struct types (`ActorIsHelper{WithinReach: false}`, `GearCaughtEffect{Hold}`). A "null" part is the zero value, documented invalid-or-absent. Telemetry getters: a recorded failure sets `error.type` from the error type name.
- **Failures:** every rule outcome is a result; `*srderr.PortError` arises only from `dice.Roller`.

## 6. Interface

Use cases take a `context.Context` first because they emit spans, and are otherwise synchronous. Only the `dice.Roller` (shared) is a port; no new port, double or contract suite.

- `DeployGear` (built with `combat.WithTelemetry`; `Run(ctx, DeployGearCommand)`): command fields `Gears, ID, Kind, Deployer, DistanceFeet, Ground`.
- `MaintainDeployedGear` (same construction; `Run(ctx, cmd)`): command `Gears, Holds, Maintenance`; `holds` is read only by `ResetHuntingTrap` (FR-013). Host duty: call `ExpireBurningOil(endedTurn)` at the end of every turn, or oil stays burning past its time (FR-022, Q-9).
- `ResolveGearEntry` (built with a roller and `combat.WithTelemetry`): command `Gears, Ledger, GearID, Creature, Trigger, Stamp, HoldID, Attempt` (a `SaveAttempt`); results carry the new `Gears` and `Ledger`, the `GearEntryOutcome`, `events`.
- `ThrowNet` (roller and telemetry): command `Holds, HoldID, Thrower, Target, TargetSize, DistanceFeet, Sight, SaveDC, Attempt` (a `SaveAttempt`), `EscapeDCOverride`.
- `BindCreature` (roller and telemetry): command `Holds, HoldID, Kind` (a `BindingKind`), `Binder, Target, TargetSize, TargetConditions` (a `ConditionSet`), `Willingness, DistanceFeet` (ignored for Rope), `Restraint, Attempt` (a `CheckAttempt`).
- `ReleaseFromHold` (roller and telemetry): command `Holds, HoldID, Method, Actor` (a `ReleaseActor`), `Actions, HasThievesTools, Attempt` (a `CheckAttempt`); `Released` and `Held` carry `actionSpent`, the check and any `Damage`.
- Hosts that skip use cases call `DeployedGears`, `GearHolds`, `AreaEntryRules`, `NetRules`, `BindingRules`, `HoldRules` directly (no telemetry).
- **Exports:** every name above from package `combat`; imported types (`Damage`, `ConditionSet`, `ConditionKind`, `ObjectStatus`, `SaveAttempt`, `CheckAttempt`, `CreatureSize`, `DifficultyClass`) are exported by their owners.

## 7. Observability

Shared conventions: `combat-foundations.md` FR-027 to FR-041.

1. **What can go wrong?** Refusals (hold already present, trap sprung, oil unlit, nothing to escape with), a dice roller failing, inconsistent host facts (`Invalid`). 2. **How would an operator know?** `srd5e.combat.result` and `result_class` on the shared counter; roller failure raises `srd5e.combat.errors`. 3. **Diagnosis:** operation, result, gear kind, entry result, end cause, holds in force; ids and DCs stay in results. 4. **Counted here:** entries, hold endings, holds in force, one span per operation. 5. **Existing instrumentation:** unaffected; no `srd5e.d20_test.*` signal fires (FR-044 of the shared decisions). 6. **Monitor:** a large `already_triggered` share (host re-reports entries); `burst` or `escaped` ratios far from the DCs (dice or modifiers); `holds_in_force` that only grows (host never ends holds). 7. **Noise:** none; one `FINE` log per call.

Closed sets: `srd5e.combat.gear.kind`: `caltrops`, `ball_bearings`, `poured_oil`, `hunting_trap`, `net`, `chain`, `manacles`, `rope`. `srd5e.combat.gear.entry`: `save_failed`, `save_passed`, `damaged`, `already_triggered`, `not_active`, `not_applicable`. `srd5e.combat.gear.end_cause`: `escaped`, `burst`, `destroyed`, `unlocked`. `srd5e.combat.result` per use case: one value per variant or reason, each mapping to one class (FR-065).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.gear.entries` | counter `{entry}` | `srd5e.combat.gear.kind`, `srd5e.combat.gear.entry` | Area-entry outcomes (FR-066) | `already_triggered` above 20 percent: host re-reports entries |
| `srd5e.combat.gear.hold_ended` | counter `{hold}` | `srd5e.combat.gear.kind`, `srd5e.combat.gear.end_cause` | How holds end (FR-067) | `escaped` share far from DC expectation |
| `srd5e.combat.gear.holds_in_force` | histogram `{hold}` | `srd5e.combat.operation` | Size of `GearHolds` in a result | p99 above 50 per encounter |
| `srd5e.combat.deploy_gear`, `...maintain_deployed_gear`, `...resolve_gear_entry`, `...throw_net`, `...bind_creature`, `...release_from_hold` | span | `srd5e.combat.operation`, `srd5e.combat.result`, `srd5e.combat.result_class`, `srd5e.outcome` | One call (FR-064) | Shared, `combat-foundations.md` |

## 8. Test plan

Doubles: `dicetest.FixedRoller`, recording `telemetry.Telemetry`. No contract suite. Tables 4.9a, 4.9b, 4.9.1, 4.9.2 are table-driven.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-003 | unit, contract | exact member sets; profile constants against table 4.9a and 4.9b; each `ItemID` is an `equipment` entry of the bundled set |
| FR-004 to FR-015 | unit | reset with and without an occupying hold; `deploy` rows (reach 5/6, 10/11, level ground, id in use, refusal order), recover, light, expire (table 4.9.2), reset, collection invariants, `TurnStamp` round 0 and 1 |
| FR-016 to FR-025 | unit, application | tables 4.9.1 and 4.9a with the real `d20.ResolveCheck`; DC boundary totals; forced failure draws no d20; dice requested (d20s, then `1d4`) |
| FR-026 to FR-034 | unit | hold construction errors, one hold per target and kind, `restraintBegins` and `restraintEnds` with two restraining holds, Manacles Disadvantage, `reanchor`, DC override, immutability |
| FR-035 to FR-044 | unit, application | `NetRules` rows (15/16 ft, sight order, DC 13/9, sizes), auto-success draws no die, `recordNetDamage` with a real `ApplyDamageToObject` result, `NetProfile` equals an `ObjectStatus` of AC 10, HP 5 and the three Immunities |
| FR-045 to FR-049 | unit, application | rejection order rows (incl. willing Unconscious target: `targetWilling`); Rope at any distance;  implied Incapacitated sources; checks per kind; size and reach boundaries |
| FR-050 to FR-058 | unit, application | availability matrix; every (ability, skill, DC) in 4.9b; helper rules; `noAction` (also for key); `toolsMissing`; key; Hunting Trap failure damage; forced failure; `actionSpent` |
| FR-059 to FR-068 | application | variant sets, roller failure propagation (set unchanged), determinism with both `telemetry.Telemetry` types, event kinds and order, spans and class mapping, counters once, no id in any attribute, no log at `INFO` |
| FR-069 to FR-072 | architecture, unit | `signals.md` lists the new signals; import scan; exported-name scan and doc comment check; constructors never panic and `MustX` matches `NewX` |
| scenario | scenario | public API only: a Hunting Trap is set, a creature steps on it, fails the save, is caught, fails then passes the escape check; Caltrops entered twice in one turn; Manacles bound to a Grappled creature, Disadvantage merged with conditions; Net thrown at a Large creature, destroyed through `ApplyDamageToObject` |
| errors-and-results.md FR-031, FR-034, FR-051 | application | The roller failure of FR-061 propagates unchanged with no partial result; a recorded failure sets `error.type` from the error type name; a leaked non-`*srderr.PortError` propagates uncounted. |

- **Mutation testing:** package `combat` gear files, `gomu` threshold 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (`doc/rules/testing.md` §8); the boundary rows kill comparison mutants and the once-per-turn rows kill ledger-key mutants.
- **Not covered:** none; every FR appears above.

## 9. Open questions and decisions

Open: none. Nothing is escalated. Settled by SRD text and research; interpretations are named.

- **Q-1 ("a turn"):** the SRD says "first time on a turn" and "once per turn" without saying whose turn; the common reading is any turn in progress, so a creature entering on another creature's turn and again on its own is affected twice (FR-017, FR-021; interpretation).
- **Q-2 (level ground for Caltrops and Hunting Trap):** only Ball Bearings and Oil state it, so no check for the others (FR-007).
- **Q-3 (Hunting Trap):** the plate "snaps shut" on any step, so the trap is `Sprung` even when the save succeeds; only a failure catches the creature; resetting is a Utilize action like setting (FR-013, FR-023; interpretation).
- **Q-4 (Rope):** binding a creature uses the DC 10 knot check, as Chain and Manacles state their own checks (FR-047; interpretation).
- **Q-5 (other Restrained sources):** `restraintEnds` considers only holds in the set; the host also checks spells or effects (FR-029).
- **Q-6 (willing creature, untying):** no rule; a willing target is refused `targetWilling` so the host decides; no untie method exists (FR-045).
- **Q-7 (who may burst or escape):** the held creature only, for Chain, Manacles and Rope; a Net and a Hunting Trap admit a helper because the text says so. Rope's burst states no action cost; it is treated as an action like Chain and Manacles (FR-052; interpretation).
- **Q-8 (Net size):** the SRD gives none and no rule reads it; Medium is a placeholder (FR-044).
- **Q-9 (turn comparison):** no initiative order is needed; `Expire` is told which turn ended (FR-012). Entries later in the lighter's final round depend on the host having called `Expire` for the lighter's turn (FR-022; table 4.9.1).
- **Q-10 (second Hunting Trap):** a creature limited to 3 feet may step on another trap; the hold key for traps includes the trap's `gearID`, so both holds exist (FR-027). The host applies the smaller `movementLimitFeet` (interpretation).
- **Q-11 (reset with creature held):** `trapOccupied` is derived from `GearHold.gearID`, not a host boolean (FR-013, FR-026).
- **Q-12 (key and Thieves' Tools):** the SRD says a creature "can use Thieves' Tools" (p. 98) so `pickLock` needs the host fact `hasThievesTools`; using a key states no cost, so it takes no check and no spent action but needs an actor able to act (interpretation).
- **Q-13 (Rope reach):** Rope states no distance (p. 99), unlike Chain and Manacles; the engine does not check one for Rope and leaves adjacency to the host (interpretation).

## 10. Changelog impact

Under `Unreleased`, "Added": deployed and restraining gear in `combat`: `DeployedGears`, `EntryLedger`, `GearHolds`, `NetProfile`, the rule services, the closed `GearEvent` family, and the use cases `DeployGear`, `MaintainDeployedGear`, `ResolveGearEntry`, `ThrowNet`, `BindCreature`, `ReleaseFromHold`, with signals `srd5e.combat.gear.entries`, `...hold_ended`, `...holds_in_force` and six operations, each added to `doc/observability/signals.md` (FR-072). Not breaking (new API); the closed result and `GearEvent` families make later variants breaking after release.
