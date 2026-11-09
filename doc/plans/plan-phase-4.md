# Phase 4 plan: the `combat` context

| | |
|---|---|
| Status | Proposed. Nothing is built. |
| Roadmap phase | 4 ([../roadmap.md](../roadmap.md)) |
| Specs sequenced | `combat` (rank 20 to 31, 36 specs): [combat-foundations](../specs/combat-foundations.md), [hit-points-and-damage](../specs/hit-points-and-damage.md), [conditions](../specs/conditions.md), [grappling](../specs/grappling.md), [hiding-searching-and-senses](../specs/hiding-searching-and-senses.md), [environment-hazards-and-objects](../specs/environment-hazards-and-objects.md), [telepathy](../specs/telepathy.md), [attacks](../specs/attacks.md), [attack-and-help-actions](../specs/attack-and-help-actions.md), [mounted-and-underwater-combat](../specs/mounted-and-underwater-combat.md), [deployed-and-restraining-gear](../specs/deployed-and-restraining-gear.md), [monster-engulf-and-swallow](../specs/monster-engulf-and-swallow.md), [initiative-and-actions](../specs/initiative-and-actions.md), [item-use-effects](../specs/item-use-effects.md), [monster-limited-use](../specs/monster-limited-use.md), [magical-effects](../specs/magical-effects.md), [movement-lifting-and-flight](../specs/movement-lifting-and-flight.md), [grid-positions-and-movement](../specs/grid-positions-and-movement.md), [readying-a-spell](../specs/readying-a-spell.md), [monster-combatant-setup](../specs/monster-combatant-setup.md), [spell-effects](../specs/spell-effects.md), [returning-to-life](../specs/returning-to-life.md), [monster-stat-block-actions](../specs/monster-stat-block-actions.md), [monster-traits](../specs/monster-traits.md), [monster-spellcasting](../specs/monster-spellcasting.md), [encounter-aggregate](../specs/encounter-aggregate.md), [active-effects](../specs/active-effects.md), [spell-granted-modifiers](../specs/spell-granted-modifiers.md), [persistent-spell-zones](../specs/persistent-spell-zones.md), [dispel-and-counterspell](../specs/dispel-and-counterspell.md), [shape-changing-spells](../specs/shape-changing-spells.md), [revival-and-restoration-spells](../specs/revival-and-restoration-spells.md), [spell-outcome-tables](../specs/spell-outcome-tables.md), [combat-action-resolution](../specs/combat-action-resolution.md), [combatant-roster-changes](../specs/combatant-roster-changes.md), [summoned-and-companion-creatures](../specs/summoned-and-companion-creatures.md). Rank table: `shared-types-and-ownership.md` §4.11. |
| Depends on | Phases 0 to 2, and Phase 3 steps 0 to 7c (see P1); step 12 also needs Phase 3 steps 11b (Druid), 11e (Warlock) and the step 10 spells (P1) |
| Format | follows [plan-phase-3.md](plan-phase-3.md) |

Naming note: `A1` to `A13` mean the **architecture checks** of `architecture.md`
section 9. Plan items are numbered `P1` onward. Steps are numbered by rank group.

## 1. Goal

Ship the `combat` bounded context (one Go package, `architecture.md` §2):
combatant identity and the shared event and telemetry conventions, hit points
and damage, conditions, attacks and the attack-adjacent actions, initiative and
the turn tracker, movement, spell effects on combatants, monster setup and
monster stat-block behaviour, the encounter aggregate, whole-action resolution,
roster changes and summoned creatures. Also the Phase 4 monster abilities that
need `combat` types: limited use and monster spellcasting (roadmap Phase 4).

This plan sequences decisions already made in the specs and in
[../specs/shared-types-and-ownership.md](../specs/shared-types-and-ownership.md)
(binding). It decides nothing; where a spec and this plan disagree, the spec
wins. Every Phase 4 spec records "no question open" (or only settled items) in
its section 9, so the gaps below are the plan's own findings.

## 2. What already exists (reused as-is)

| Asset | Where | Used for |
|---|---|---|
| `d20.ResolveCheck`, `ResolveAttack`, `Roll`, reroll operations, `RollModifiers`, `DifficultyClass`, `Resolver` and its `WithTelemetry` / `WithClock` options | `d20/` | every save, check, attack and Death Saving Throw (hit-points-and-damage Q-4: a Death Saving Throw is a D20 Test); the constructor and option shape for each `combat` use case |
| The `character` published language (verified present): `Identifier` (the character id, `character/identifier.go`), `Character`, `HitPointMaximum`, `IsBloodied(currentHitPoints, maximum)` (`character/bloodied.go`), `ProficiencyBonus.Value()` (`character/proficiency_bonus.go:22`), `ModifierBreakdown.ToRollModifiers`, `RestApplied`, weapon-property attack plans, the function `ExtraAttackCombination(counts []int) (int, error)` (`character/extra_attack_combination.go`), Concentration state shape, spell slots and save DC. `character.Event` is only the character-side event family: `combat` declares its own sealed `Event` (combat-foundations FR-025 reserves the name) and does not reuse it | `character/` (Phase 3 steps 2 to 7) | combatant state is built from these by id and value, never by reaching into the aggregate (`architecture.md` §2); the plan types of step 4c are what `attacks` maps |
| `modifier.RollCircumstance`, `modifier.RollMode` | `modifier/` | the condition effects of step 3 feed `d20.RollModifiers` through these |
| Carrying and speed helpers: `NewCarryingCapacity`, `CarryHalfPounds`, `DragLiftPushPounds`, `Assess`, `CanCarry` (`character/equipment_carrying.go`), `ArmorRules.SpeedPenaltyFeet` (`character/equipment_armor_rules.go`), `Barding`, `DrawnLoad` | `character/` | steps 5c (mounted) and 7c (lifting, flight) call these instead of re-deriving weights and speed penalties |
| `content.Monster`, `Spell`, `EquipmentItem`, `ConditionKind`, `DamageType`, `CreatureSize`, `WeaponProperty` and the vocabulary enums, `Source`, `InMemorySource`, `JSONDecoder` | `content/` | all content access; vocabulary types are owned here and only interpreted in `combat` |
| `content/contenttest.FakeSource`, `dice/dicetest` rollers, `telemetry/telemetrytest` doubles | test support | every use-case test |
| `srderr.WrapPort`, `GuardValue`, the `InvalidError` shape (`ability/error.go`) | kernel | port failures and validation errors |
| `telemetry.StartTimer`, `telemetry.Telemetry.WithDefaults()` (a zero bundle becomes Noop), `telemetry.AttrOutcome`, `OutcomeSuccess/Failure`, `AttrErrorType`; the unexported `character/operation_signals.go` (`operationRun`, `record`, `portFailure`) and `d20/signals.go` as patterns | `telemetry/`, `character/`, `d20/` | `combat.Instrumenter` (combat-foundations FR-035) is a deliberate sibling of `operationRun`, not an import of it (it is unexported and tied to `character`'s result switch): it reuses the `telemetry` helpers above and copies the `record`/`portFailure` logic by pattern, with its own shape (generic `Run`, result class, event counting). `WithDefaults` implements FR-035's "zero bundle defaults to Noop". Signal rows follow the `srd5e.character.*` catalogue convention |
| Exhaustion | `character/rest_benefits.go` only has the Long Rest reduction effect | the Exhaustion D20 Test penalty, Speed reduction and the "at least two sizes" threshold (initiative-and-actions FR-080, compared through the `content.CreatureSize` ordinal) are net-new in `combat`; no `character` helper backs them |
| `internal/archtest` `DefaultPolicy` row `combat` (imports `character`, `d20`, `content`, kernel) | `internal/archtest/policy.go:59` | A1 already allows the package. The `combat/combattest` helper package has no row yet (step 0) |
| `character/` pure-domain plus use-case file split, one file per concept, `Must*` helpers, closed variant sets with doc-listed variants (A7) | `character/` | the template for every `combat` file; do not invent a second layout |

No kernel change is expected. If a step finds one is needed it stops and goes to
the user (FR-001: the kernel stays small). No `character` public API changes
either: a spec that needs one is an amendment request, recorded in the spec and
sent to the user, not made in a `combat` PR.

## 3. Scope fence

**In scope:** the Phase 4 requirements of the 36 specs in the header; the
`combat/combattest` helper package named by combat-foundations (event-family
checks); the doc, policy and catalogue edits each step needs; the spec-only
amendments the specs request of one another (P5).

**Out of scope** (each is tracked where it already lives):

- Bulk data: the full monster set (`content-monsters`), the full 339 spells and
  equipment tables (Phase 5). Requirements that need them ("over the bundled
  data", every SRD spell, every stat block) are classed **D** and named in that
  spec's "Not yet implemented" line (P3).
- The `toolbox` context, ranks 40 to 41, and every `toolbox-*` spec (Phase 5).
- Class and species behaviour (Phase 3 steps 9 to 12). `combat` specs name class
  specs in prose only; requirement ids that need a class feature's own code are
  **D** until that class merges (P1).
- `release-0-1-0`, any tag or version bump, and `rules-glossary-coverage`
  (Phase 5).
- Any change to a Phase 1, 2 or 3 public API (stop and go to the user).

## 4. Open items, assumptions and spec defects

**Decided**: a spec or the user settled it. **Assumption**: a sequencing choice
the user can override. **Spec defect**: fix in the spec in the same PR as the
code that hits it. **OPEN**: needs the user before the named step.

| # | Item | Class | Handling |
|---|---|---|---|
| P1 | Which Phase 3 work must be merged before Phase 4 starts. Checked for all 36 specs' "Depends on" lines: only `summoned-and-companion-creatures` lists class specs (`class-warlock.md`, `class-druid.md`); every other `combat` spec cites class and species specs in prose only (`attack-and-help-actions` for Extra Attack counts, `shape-changing-spells`) | assumption | Steps 0 to 11 may start once Phase 3 steps 0 to 7c are merged (ranks up to 15: `initiative-and-actions` needs `feats-behaviour` (7a), `spell-effects` needs `spellcasting` (5a), `combat-action-resolution`, `monster-spellcasting` and `active-effects` need `casting-and-concentration` (6b), `item-use-effects` needs `magic-item-effects` (6c), `monster-limited-use` needs `rests-and-hit-dice` (4a), `movement-lifting-and-flight` needs `equipment-rules` (1)). **Step 12 is also gated on Phase 3 steps 11b (Druid) and 11e (Warlock), and therefore on the step 10 spell closure**, because FR-056 makes a listed spec a hard dependency. Prose-only class references in other specs stay D and are re-run in step 13. Phase 4 does not close "implemented" while step 12 or any D id of this kind is open. Override: wait for all of Phase 3 steps 9 to 11 before starting Phase 4 |
| P2 | The scope is the 36 specs of ranks 20 to 31 (P2 is the roadmap Phase 4 bullet list, which names only monster-limited-use and monster-spellcasting explicitly) | decided (roadmap, `shared-types-and-ownership.md` §4.11 and FR-014) | section 3 |
| P3 | Some requirements cannot be satisfied with the Phase 2 bundle (full monster set, all spells) | assumption: classify each requirement T (types and rules, testable with synthetic fixtures or the 17-entry slice) or D (needs Phase 5 data) per step, as Phases 2 and 3 did | the implementer re-checks every id; D ids are named in the PR `Testing` entry and the spec's "Not yet implemented" line. Fixtures are synthetic or the bundled slice, never SRD text outside `testdata/srd/` |
| P4 | `combat/combattest` (event-family checks, combat-foundations FR-018 to FR-021) is a new package with no row in `architecture.md` §1 or `DefaultPolicy` | spec defect | step 0 adds the §1 row, the dependency row (imports `combat`, `srderr`; no other `*test` package) and the policy row, with archtest cases |
| P5 | The specs request amendments of one another (encounter-aggregate from initiative-and-actions, hit-points-and-damage Q-3, combat-action-resolution Q-4 and Q-51, attacks' `...NotPossible` names recorded in combat-foundations FR-023, `ProficiencyBonus.Value()` from `derived-stats` FR-059, mounted-and-underwater-combat F-1, summoned-and-companion-creatures A3 to A6) | decided (the specs) | each step applies the edits that concern the spec it implements in the same PR; edits to other specs are spec-only. Spec-only edits never change the Status of a Phase 1 to 3 spec. A needed `character` API amendment goes to the user (`ProficiencyBonus.Value()` already exists, so none is known today) |
| P6 | A package-level `var` for tables breaks A4 | decided: A4 wins (Phase 2 P6, Phase 3 P10) | `const`, funcs and `switch` only; amend spec wording that says "table variable" in the PR that hits it |
| P7 | `combat` is one package holding 36 specs' worth of names, and combat-foundations FR-025 reserves names against other specs | decided (`architecture.md` §2: one package per context) | one file per concept; the compiler enforces uniqueness; step 1 adds an archtest case for the FR-025 reserved list; step 13 re-checks collisions across the whole package |
| P8 | Order inside a rank is not fixed by the specs | assumption | the order in section 5 is a suggestion; merge any order that respects "Depends on" (FR-056) |
| P9 | Result classification: a refusal (`...NotPossible`, `Rejected`) is not an error status (combat-foundations D-4, FR-031). It is counted by `srd5e.outcome` plus the attribute `srd5e.combat.result_class` (`applied`, `refused`, `invalid`); `srd5e.combat.result` is the finer per-use-case value (FR-030). `outcome=failure` therefore also covers every refusal and is not by itself an alarm | decided (combat-foundations) | every use case routes through `Instrumenter` (FR-035); no use case records telemetry by hand |

| P10 | **Library-level limits.** The specs set no cap on roster size, active effects, summons per summoner, readied actions, events per call, dice per expression, or the depth of a reaction, counterspell, containment, mount or summon chain (checked: encounter-aggregate, active-effects, combatant-roster-changes, summoned-and-companion-creatures, combat-action-resolution, initiative-and-actions, conditions, persistent-spell-zones). Limits that do exist: SRD-derived ones, and `grid-positions-and-movement` D-8 (coordinates within +/-1,000,000, footprints 1,000 per side). Hostile or corrupt saved state could otherwise reach an O(n squared) path | **OPEN** (needs the user) | The plan does not decide. **Proposed defaults, for the user to approve or change** (each is a design number, not an SRD rule): combatants per encounter 256; active effects per creature 64; summons per summoner 32; readied actions per creature 1 (derived from the SRD\x27s one reaction); events per call 4,096 (this cap governs events; the cascade budget governs steps, and one step may emit several events); cascade step budget 1,000 per call; dice per expression 1,000; link cycles rejected at link creation (a proposal); grid per D-8, plus an arithmetic (not enumerating) overlap test. Each cap is written into the spec of the step that builds the collection, by an amendment in that PR. **Which steps are affected:** every step that builds a collection, cascade or link graph, which in practice is every step from 3 onward (for example 3 conditions per creature; 4a grapples; 4b scenes and sightings; 4c hazards and objects; 5c the mount chain; 5d gear sets; 5e containment; 6a and 6b turn resources and ready chains; 7a to 7f limited-use ledgers, effects, boards, paths, readied spells and monster numbers (P13); 8a to 8e; 9a to 9h; 10; 11; 12). Each affected step lists in its `Testing` entry which cap and budget tests it holds back. The gate is **per test, not per step**: an affected step ships without its cap and budget tests, says so in its `Testing` entry, and they land when the numbers are set. Validation runs once at entry and once at exit of a call, never inside the cascade loop, so the cost is roster plus effects, not that product times the budget (a proposal, tested by operation count with a roster, effects and cascade all at their caps). Waiving the gate (no library caps; document the risk) is the user's call |
| P11 | Superseded signal names. combat-foundations section 5 and FR-028 list the signal names of earlier combat specs that are superseded and must not be in the catalogue (for example `srd5e.combat.use_case`, `srd5e.operation`); that list is authoritative and this plan does not restate it. Some names of those specs survive (the `conditions` histogram `srd5e.combat.condition.active` with `srd5e.combat.condition.kind`, hit-points' own counters, initiative's `initiative_ties` and `turn_movement_spent`) | spec defect | step 1 adds the catalogue check of combat-foundations FR-028's test row, reading its list; each later step removes or migrates the superseded names its spec still carries in the PR that implements it; step 13 audits `signals.md` |
| P12 | `d20` inside `combat`. combat-foundations FR-027 says `combat` reads `d20`'s exported signal constants for attribute keys it needs. `resolve-d20-test-use-case` FR-057 forbids a use case calling `Resolver`, and attacks FR-109, FR-110 and FR-127 compose `d20` domain functions as pure services | decided (the specs) | a `combat` call composes `d20.ResolveCheck`, `ResolveAttack` and the reroll operations; no `srd5e.d20_test.*` signal fires inside a `combat` call, and the `d20` extras are omitted when no d20 was rolled (attacks FR-118). Step 2 pins this with a recording test. Spec defect: attacks FR-109 and FR-127 name `d20.Resolver` where FR-057 and FR-110 forbid it; step 5a corrects that wording in the attacks spec |
| P13 | Content-derived numbers (multiattack counts, recharge and limited-use maxima, spell and action list lengths, dice count and sides) were validated by Phase 2 only structurally | assumption (a new requirement the user can drop) | `combat` re-validates them at `monster-combatant-setup` (step 7f) and in the use cases that read them; a bad value is an `invalid` result. Ranges come from the specs; any content number a spec gives no range for gets a P10 cap before its cap tests land |
| P14 | State validation versus the encounter spec: encounter-aggregate FR-009 says the root accepts only a tracker from `InitiativeOrderEstablished` and does not re-sort or re-validate it, and FR-083 says the `Combatant` constructor adds no invariant. A forged tracker or `Combatant` therefore passes through the root unvalidated | **OPEN** (needs the user if a spec amendment is wanted) | The spec wins and the plan does not override it. The plan covers forgery to the extent the specs allow (section 7). If the user wants the tracker and `Combatant` validated at the boundary, that is a spec amendment to encounter-aggregate, initiative-and-actions and hit-points-and-damage made before step 9a |

Two items need a user decision: **P10** (library limits) and, only if wanted,
**P14**. P1 and P8 are the overrides worth a look before work starts. Steps 0 to 3 and then 4a to 4d can start now: P10 only holds their cap tests, not the steps.

## 5. Delivery sequence

At least 38 reviewable PRs (the splits allowed for steps 7b and 10 add more), each green on its own. Order is forced by imports and by
the intra-context order of `shared-types-and-ownership.md` FR-015 and the rank
table 4.11.

| Step | Ships | Rank | Why here |
|---|---|---|---|
| 0 | Groundwork: docs, policy, `combat/doc.go`, `combat/combattest` policy | n/a | the first production file must be inside the allow-list |
| 1 | combat-foundations | 20 | `CombatantID`, `CombatantRef`, `Event`, `Instrumenter`, `CombatRuleOptions`; everything else uses these |
| 2 | hit-points-and-damage | 21 | `HitPointStatus`, `ApplyDamage`, Death Saving Throws; conditions and attacks need it |
| 3 | conditions | 22 | `ConditionSet`, effective Speed, Exhaustion |
| 4a to 4d | grappling, hiding-searching-and-senses, environment-hazards-and-objects, telepathy | 23 | independent of one another (4a, 4b and 4d need steps 1 and 3; 4c needs steps 1 to 3) |
| 5a to 5e | attacks, attack-and-help-actions, mounted-and-underwater-combat, deployed-and-restraining-gear, monster-engulf-and-swallow | 24 | `attacks` needs `grappling` (4a); 5b needs only step 1; 5c needs step 2; 5d needs 4c; 5e needs 4a. Within the rank 5a first is only a suggestion (P8), since 5b to 5e do not depend on it |
| 6a, 6b | initiative-and-actions, item-use-effects | 25 | `TurnTracker`; 6b needs 4c, 5a, 5b and Phase 3 `magic-item-effects` |
| 7a to 7f | monster-limited-use, magical-effects, movement-lifting-and-flight, grid-positions-and-movement, readying-a-spell, monster-combatant-setup | 26 | independent of one another; each needs the earlier ranks its spec lists (7a, 7c, 7d, 7e and 7f need 6a; 7b needs 5a) |
| 8a to 8e | spell-effects, returning-to-life, monster-stat-block-actions, monster-traits, monster-spellcasting | 27 | 8a needs 5a and Phase 3 `spellcasting` (not 7b); 8b needs 7b; 8c needs 5a, 6a, 7a; 8d needs 4c, 6a, 7a; 8e needs 7a and Phase 3 `casting-and-concentration` |
| 9a to 9h | encounter-aggregate, active-effects, spell-granted-modifiers, persistent-spell-zones, dispel-and-counterspell, shape-changing-spells, revival-and-restoration-spells, spell-outcome-tables | 28 | 9a first (aggregate root; needs 4a and 6a); 9b needs 6a and 8a; 9d and 9e need 7b and 8a; 9f needs 7b; 9g needs 8a and 8b; 9h needs 8a; 9c needs 5a and 6a |
| 10 | combat-action-resolution | 29 | composes the domain functions of everything to its left; the largest spec (165 ids) |
| 11 | combatant-roster-changes | 30 | needs 4a, 5b, 6a, 9a and 10 |
| 12 | summoned-and-companion-creatures | 31 | needs 5c, 11 **and Phase 3 steps 11b (Druid), 11e (Warlock) and 10** (P1) |
| 13 | Whole-phase scenarios, examples, README, CHANGELOG, roadmap, spec status audit | n/a | closes the phase |

Natural stopping points that leave a coherent library: after step 3 (hit points,
damage and conditions on a combatant), after step 6 (attacks, initiative and
turns), after step 10 (a whole encounter can be run through the public API).

### Process contract for every step

- Read the four rule files first (`AGENTS.md`); never quote a `RULE-TAG` in a
  plan or PR text (A13 requires each value to appear in one place only).
- Every new test cites its requirement (`Requirement:` and `Path:`;
  `Requirement: per case` on tables). Apply `testing.md` §5 to all new tests.
- Each PR carries a **requirement-to-test list**: every T id of the spec maps to
  a test; D ids and any id deferred are named in the `Testing` entry.
- `just check` green. Each changed package at least 90 percent statement
  coverage, reported per package.
- Mutation testing with `gomu`, only inside Docker, `--cpus` and `--workers`
  capped at a quarter of the processors (`just mutate ./combat/<file>.go`, one
  file at a time). Score at least 80 for domain files; survivors fixed unless
  provably equivalent. Where gomu cannot parse a file, use the manual-mutation
  fallback of `testing.md` §8 and list the mutations. Name test files after the
  source files so gomu pairs them.
- `CHANGELOG.md` entry under `Unreleased`.
- The PR description carries `Testing` and `Observability` entries. The
  `Observability` entry answers **all seven questions** of `observability.md`
  §2; a bare "none" or "silent by design" is rejected.
- Signals: every new signal is a constant in `combat/signals.go`, a row in
  `doc/observability/signals.md` with purpose and suggested monitoring, and a pin
  test. New attribute values go into `observability.md` §5 in the same PR (tag
  refreshed).
- A spec's `Status` becomes `partially implemented` in the PR that first ships
  code for it, with an `Implementation status` list and a `Not yet implemented:`
  line; it becomes `implemented` only when that line is empty.
- Pure domain files import no `telemetry` (O1, review-enforced); use-case files
  hold every `telemetry` call, every `srderr.WrapPort` call and the `Instrumenter`
  use (A6).
- A use case never calls another use case (`architecture.md` §4). `combat-action-resolution`
  composes domain functions, not use cases.
- **Unhappy paths in every use-case step** (`testing.md` §4), **where the step's
  spec has them** (the spec\x27s own rows win over this checklist, but every step keeps at least the boundary, empty or nil, immutability and cancelled-`ctx` items of `testing.md` §4 unless it says why not): an unknown `CombatantID` (an `invalid` result)
  and a dead, knocked-out or not-active combatant (a `refused` result);
  a failing roller and a panicking roller (`*srderr.PortError` once,
  `srderr.GuardValue`); a failing and a panicking content source; a cancelled
  `ctx`; the input state unchanged after every refusal and after a panic; a
  returned slice or event list that is a copy; Advantage and Disadvantage
  together and stacked conditions. Each step names its own boundary rows.
- **Fuzzing:** every new `NewX` or `ParseX` that takes untrusted text or numbers
  has a `Fuzz*` test (`testing.md` §10), not only step 1. Each PR's
  requirement-to-test list names its `Fuzz*` targets, or says there are none
  because the step adds no such constructor.
- **Limits (assumption, depends on P10):** once the user confirms or changes the
  caps proposed in P10 and they are written into the spec, every collection a
  step builds gets a typed `invalid` rejection (one shared `limit_exceeded`
  result value) and tests at the cap and cap plus one. Until then the step ships
  without those tests and says so. Arithmetic on host or content numbers
  (damage multipliers, Temporary Hit Points sums, durations in rounds, grid
  coordinates and squared distances, weight sums) gets overflow tests at the
  limits of the quantity; the content suite also runs at 32-bit width today, so
  step 13 runs `GOARCH=386 go test ./combat/...` for these (an assumption of this plan, since the existing 32-bit run covers `content` only). Content ids and
  free-form keys never become telemetry attribute values; the foundation spec's
  recording-double test (FR-027/FR-033) is reused per use-case step.
- **Observability entry per step follows its spec's own section 7.** The PR
  reproduces that spec's signals, `srd5e.combat.result` values and extras,
  failure modes and monitoring, and answers the seven questions. This plan lists
  no per-step signal names, because a plan decides nothing and the specs are
  authoritative (several signal names are superseded, P11). A step whose spec is
  silent by design (for example `hiding-searching-and-senses`) says so and which
  use-case step counts its failures.
- **Acceptance bar for every step:** every T id has a test, the unhappy paths
  above are covered or justified, and D ids are listed.
- Fixtures are synthetic or the bundled slice; nothing from outside SRD 5.2.1;
  no consumer names.

### 5.0 Step 0: groundwork

- `doc/rules/architecture.md` §1: the `combat` row already exists in the §2
  context map and the policy table; add the missing `combat` row to the §1
  package table and dependency-rule table (`character`, `d20`, `content`, kernel;
  no `*test` in production files), and the `combat/combattest` row (imports
  `combat`, `srderr`). Edit existing rows rather than duplicating them. Refresh
  the `RULE-TAG`.
- `AGENTS.md` "Directory structure": add `combat/` and `combat/combattest/`; in
  "Bounded contexts" change `combat/ (planned)` to in progress. Leave "Read the
  rules first" untouched.
- `combat/doc.go`: package comment naming inputs, outputs and handoff types
  (grows with each step).
- `internal/archtest` `DefaultPolicy`: add a `combat/combattest` row of
  `KindSupport` with `Allow: combat, srderr`, mirroring `content/contenttest`
  (A1, A2, A11), and extend the real-module test.

**Tests:** archtest cases: a `combat` production file importing `toolbox`
(violating) and `character`, `content`, `d20` (clean); a `combattest` file
importing a context other than `combat` (violating).
**Observability:** the seven questions, answered. What can go wrong: a `combat`
file imports a downstream context or a `*test` package (A1, A2). Operator
signal: `just arch` fails. Nothing to count or time; no existing signal
affected; no consumer monitoring; nothing noisy.
**Rollback:** revert the PR (docs and an empty package).
**Acceptance:** `just arch` passes with `combat/doc.go` only.

### 5.1 Step 1: combat-foundations (rank 20)

`CombatantID`, `CombatantKind` (`PlayerCharacterCombatant`, `MonsterCombatant`),
`CombatantRef`, the sealed `Event` interface and `Event.Kind()`, the transport
envelope marker, `CombatOperation`, `CombatResultClass`, `CombatRuleOptions` and
`PlayingOnAGrid`, `Instrumenter` with `NewInstrumenter(t, clock)`, and the shared
signals `srd5e.combat.*`. Also `combat/combattest` event-family checks.
**Reuse:** `character.Identifier`, `content.MonsterID`, `telemetry.StartTimer`,
the `d20.NewResolver` option shape, `rule-options.md` pattern.
**Tests:** unit for every value object (blank id per FR-002/FR-006, no trimming,
equal ids), table rows for the result-class mapping, the FR-025 reserved-name
list as an archtest case, a self-test of the `combattest` event-family helper
against a fake family (good family passes; aliased slice, duplicate, unordered
list and wrong marker fail). `Instrumenter` tested with `telemetrytest`: one
counter increment per call, refusal gives no error status, a `*srderr.PortError`
is recorded once under `srd5e.combat.errors` with `error.type`, Noop versus
recording bundle give identical results. `Fuzz*` for `NewCombatantID`.
Also pinned here: the catalogue check of P11 (no superseded signal name), and
that shared keys in a use case's extras are rejected (FR-041).
**Observability:** the shared signals and attribute values are defined here and
catalogued (`srd5e.combat.operations`, `operation_duration`, `events`, `errors`;
attributes `srd5e.combat.operation`, `result`, `result_class`); every later step
reuses them. Failure modes and who records them: a port failure (`error.type`,
span `RecordError`, `srd5e.combat.errors` once, no log); a panic from domain code
(a bug: the span ends, no `errors` increment, the panic propagates, FR-035). An
invalid `CombatantID` is rejected by `NewCombatantID` before any use case runs, so
nothing is counted; an unknown id passed to a use case later is an `invalid`
result. No logs at Info or above (FR-034). Existing instrumentation: the
superseded names of P11 are removed. Spec alert guidance: any increment of
`srd5e.combat.errors`; operation duration p99 above 10 ms for 5 minutes.
**Rollback:** revert the PR. Every later step builds on this API; changing it
later is a breaking change within the unreleased module, and after v0.1.0 a
signal rename is BREAKING for consumers' dashboards (section 8).
**Acceptance:** `just check` green; the `combattest` helper is covered by its own
test.

### 5.2 Step 2: hit-points-and-damage (rank 21)

`HitPointStatus`, temporary Hit Points, `ApplyDamage` and healing, Resistance,
Vulnerability and Immunity order, Massive Damage, Dying, Stable, Death Saving
Throws (a D20 Test through `d20`), knock-out, `Bloodied` via the single
`character` predicate.
**Reuse:** `d20.ResolveCheck` and reroll operations, `dice.Roller`,
`character.IsBloodied` (the only Bloodied predicate; do not add another),
`content.DamageType`.
**Tests:** unit table rows for every boundary: damage 0, equal to Hit Points, one
over, Temporary Hit Points absorbing the remainder (Q-1), Massive Damage at
exactly the maximum, a Critical Hit at 0 Hit Points (two failures), a Stable
creature taking damage (Q-3: Dying and one failure), healing from 0, natural 1
and 20 on a Death Saving Throw, Advantage and Disadvantage and the Exhaustion
reduction. Application tests for each use case with `dicetest` and
`telemetrytest`, a failing roller and a panicking roller (`*srderr.PortError` once, no result committed), a
cancelled `ctx`, a dead or absent target, the P12 test (a recording `telemetry` double sees no
`srd5e.d20_test.*` signal during a combat call), and overflow rows for
damage multipliers and Temporary Hit Points sums at `math.MaxInt`.
**Observability:** use cases via `Instrumenter`, following
`hit-points-and-damage` section 7 and its surviving own signals (damage resolved,
hit points lost and restored, death saves, refusals and deaths, with their
attributes). Failure modes: a roller failing mid-Death-Saving-Throw (port failure:
`error.type`, once), a refusal such as a dead target, a creature not Dying or not
Stable (a `refused` result with its own `result` value, not an error). New
`result` values are added to `observability.md` §5, or that section points at the
spec to keep its line reviewable, in the same PR. Monitor the refused share per
operation and any `errors` increment.
**Rollback:** revert the PR.
**Acceptance:** every T id has a test; `IsBloodied` stays the only predicate.

### 5.3 Step 3: conditions (rank 22)

`ConditionSet` and its sources (a `CombatantID` for fear, D-2), every SRD
condition's effects as data and pure functions, effective Speed floored at 0
(D-8), Exhaustion capped at 6 with `IsDeadFromExhaustion` (D-3), Immunity
suppressing rather than deleting (D-5), no duplicate source (D-9).
**Reuse:** `modifier.RollCircumstance`, `content.ConditionKind`. The Exhaustion
D20 Test penalty and Speed reduction are net-new here (`character` only has the
Long Rest reduction effect).
**Tests:** unit for every condition's effect on attacks, saves and checks,
stacking and removal, Unconscious ending with Prone persisting (D-7), Exhaustion
at 0, 1, 5, 6 and 7, order independence of sources, immutability (copies), a
duplicate source rejected (D-9), Speed forced below 0 floored at 0 (D-8),
Immunity suppressing an existing condition without deleting it (D-5), a
conditions-per-creature bound at the cap and cap plus one (gated, P10), and `Fuzz*` for
any new parser.
**Observability:** `conditions` defines three use cases (`apply_condition_to_set`,
`remove_condition_from_set`, `reduce_exhaustion_on_set`, conditions FR-119 to
FR-122) with their own result values, three event kinds and the surviving
histogram `srd5e.combat.condition.active` with `srd5e.combat.condition.kind`; the
step follows that section. `ReduceExhaustion` errors are validation errors
returned unchanged, not port failures. The pure rule functions behind them are
silent, with failure modes as returned values; a wrong condition effect is not
detectable at runtime, so the guard is the per-condition table tests and mutation
score. The superseded names are not added (P11). Nothing noisy.
**Rollback:** revert the PR.
**Acceptance:** all 142 ids classified and tested or named D.

### 5.4 Steps 4a to 4d: rank 23 (one PR each)

| Step | Spec | Notes specific to the PR |
|---|---|---|
| 4a | `grappling.md` | Grappled, escape and shove contests resolved through `d20`; `Grapples` record held by the host (see encounter-aggregate settled item) |
| 4b | `hiding-searching-and-senses.md` | Hide, search, senses, Invisible and Obscured interactions; class references stay D |
| 4c | `environment-hazards-and-objects.md` | objects, Damage Threshold evaluated before `ApplyDamage` (hit-points-and-damage Q-11), hazards; needed by 5d, 6b and 8d |
| 4d | `telepathy.md` | small value-type spec; no dice |

**Tests:** unit for rule maths with boundary and rule-interaction rows
(`testing.md` §4); application tests only where a spec defines a use case, with
the unhappy paths of the process contract. Named boundaries: 4a contest ties and
the DC boundary, escape and shove outcomes, a grappler that becomes Incapacitated;
4b a hidden creature found at the DC and one below, Invisible against Obscured;
4c Damage Threshold at equal, one below and one above (hit-points-and-damage
Q-11), an object at 0 Hit Points; 4d the small value rules of telepathy range.
**Observability:** each follows its spec's section 7. 4a (`grappling`) has its own
signals (grapple ended, in force, end reason); 4b (`hiding-searching-and-senses`)
is silent by design, its failure modes being returned values counted by the
use-case steps that call it; 4c follows its spec; 4d (`telepathy`) is pure and
silent, and the use case `ChangeTelepathicContact` that wraps it is recorded by
step 10. A roller failure is `error.type` once; a refusal is a `refused` result.
**Rollback:** each PR reverts alone.
**Acceptance:** per PR, every T id of the spec has a test.

### 5.5 Steps 5a to 5e: rank 24

| Step | Spec | Notes |
|---|---|---|
| 5a | `attacks.md` | Attack plans from `character` step 4c mapped into combat attack data; `...NotPossible` refusal names (FR-120), critical range (FR-134), cover and range |
| 5b | `attack-and-help-actions.md` | Attack action, Extra Attack combination read from `character` (`ExtraAttackCombination`); counts reported by class specs are D until Phase 3 step 9 |
| 5c | `mounted-and-underwater-combat.md` | mount and underwater rules; `SpendMovement` not added (F-1) |
| 5d | `deployed-and-restraining-gear.md` | needs 4c |
| 5e | `monster-engulf-and-swallow.md` | needs 4a; uses `dice.Roller`; state in and out |

**Tests:** as 5.4, plus dice-driven rows with fixed and min/max rollers, natural
1 and 20, Advantage and Disadvantage together. Named boundaries: 5a the
`...NotPossible` refusals (FR-120), critical range (FR-134), cover, and range at
the long-range edge and one past it; 5c a mount and rider that cannot form a
cycle; 5e engulf and swallow entry, exit and invalid transitions, and a
containment link cycle rejected iteratively with a visited set (P10).
**Observability:** each follows its spec's section 7. 5a records the attack
extras its spec defines; 5c has use cases (`mount_creature`, `dismount_creature`,
`resolve_falling_off`, with a dice roll) beside silent rule functions; 5e records
containment results. A roller failure is `error.type` once and every
`...NotPossible` is a `refused` result. Monitor the refused share per operation
and reason after a content change.
**Rollback:** each reverts alone. 5a first is only a suggestion (P8).
**Acceptance:** per PR, every T id tested.

### 5.6 Steps 6a, 6b: rank 25

- **6a initiative-and-actions.md**: `TurnTracker`, `InitiativeCount`, turn
  advance, per-turn resources, the "at least two sizes" threshold (FR-080), readied
  actions. Trackers compare by value (encounter-aggregate settled item).
- **6b item-use-effects.md**: using items in combat; needs 4c.

**Tests:** as 5.5; tracker value-equality rows, initiative ties, advance across
rounds, an effect on its last round, a ready-action chain stopped by the step budget (gated, P10), and the out-of-turn and wrong-owner refusals.
**Observability:** 6a follows `initiative-and-actions` section 7, including its
surviving signals (`initiative_ties`, `initiative_counts`, `turn_movement_spent`)
and its ban on per-turn counters such as `turns_started` (FR-103). 6b follows its
own spec. A refused out-of-turn action is `refused`; a roller failure is
`error.type` once; monitor the refused share by reason.
**Rollback:** each reverts alone. **Acceptance:** per PR, every T id tested and
the process-contract unhappy paths covered.

### 5.7 Steps 7a to 7f: rank 26

| Step | Spec | Notes |
|---|---|---|
| 7a | `monster-limited-use.md` | uses `rests-and-hit-dice` recharge mapping; roadmap Phase 4 item |
| 7b | `magical-effects.md` | needs `attacks` (5a); large (148 ids), split into types then rules if it exceeds a reviewable size |
| 7c | `movement-lifting-and-flight.md` | uses `character` equipment rule maths |
| 7d | `grid-positions-and-movement.md` | reads `CombatRuleOptions.Grid` from step 1 |
| 7e | `readying-a-spell.md` | small |
| 7f | `monster-combatant-setup.md` | `SetUpMonsterCombatant`; other files stay pure (FR-045); bundled monsters beyond the slice are D |

**Tests:** as 5.6, plus missing-content rows for the content-reading steps (an
unknown monster or spell id, a duplicate entry, content that fails to decode, a
failing and a panicking content source) and P13 re-validation of content numbers
in 7f (a multiattack, recharge or dice count out of range is an `invalid`
result).
Named boundaries: 7a a limited-use ledger at its last use and one past it, recharge at the boundary roll; 7b a magical effect on its last round; 7c carry and lift at the weight threshold, one below and one above, and a Strength of 0 or the maximum; 7d coordinates at +/-1,000,000 and one beyond, a footprint of 1,000 and 1,001 per side, and the squared-distance arithmetic at the limit; 7e a readied spell released or dissipated on the same turn; 7f a multiattack, recharge or dice count at its range and one outside it (P13).
**Observability:** each follows its spec's section 7 and says which of its files
are silent. 7e (`readying-a-spell`) is silent in its own files: its three use
cases are recorded by the `combat-action-resolution` amendment (step 10). 7f
monitors the `invalid` share after a content change (bad monster data).
**Rollback:** each reverts alone. **Acceptance:** per PR, every T id tested and
the process-contract unhappy paths covered.

### 5.8 Steps 8a to 8e: rank 27

| Step | Spec | Notes |
|---|---|---|
| 8a | `spell-effects.md` | needs `spellcasting`; spells beyond the bundle are D |
| 8b | `returning-to-life.md` | needs 7b |
| 8c | `monster-stat-block-actions.md` | needs 5a, 6a, 7a |
| 8d | `monster-traits.md` | needs 4c, 6a, 7a |
| 8e | `monster-spellcasting.md` | roadmap Phase 4 item; needs `casting-and-concentration` |

**Tests:** as 5.7, with synthetic stat blocks and spells; every monster-facing
requirement over the full SRD set is D. Missing-content rows for each (unknown
monster, spell or form id).
Named boundaries: 8a a spell whose table entry is first or last; 8b revival at the edge of its time window and one past it; 8c an action at its limited-use boundary; 8d a trait at the threshold of its trigger; 8e a spell slot at its last use.
**Observability:** each follows its spec's section 7 (operations, `result` values
and extras, for example the spell table in 8a and the spellcasting kind in 8e),
counts port failures once, and monitors `invalid` and `refused` shares after a
content change.
**Rollback:** each reverts alone. **Acceptance:** per PR, every T id tested and
the process-contract unhappy paths covered.

### 5.9 Steps 9a to 9h: rank 28

`encounter-aggregate` first (9a): the `Encounter` root, `Combatant`,
`applyTurnChange` (FR-096), per-turn Death Saving Throw flag, `KnockedOut`
marker. Then, independently: 9b `active-effects`, 9c `spell-granted-modifiers`,
9d `persistent-spell-zones`, 9e `dispel-and-counterspell`, 9f
`shape-changing-spells`, 9g `revival-and-restoration-spells`, 9h
`spell-outcome-tables`.
**Tests:** 9a, explicitly: a forged `Encounter` built as a struct literal or from
the zero value is invalid (tracker and `Combatant` forgery only as far as encounter-aggregate FR-009 and FR-083 allow, P14) and cannot skip invariants (the type is opaque: unexported
fields, copies in and out, so a host alias cannot change state mid-resolution);
a duplicate `CombatantID` rejected; nil versus empty rosters; a roster at the
cap and cap plus one (gated, P10); the cost of per-call validation stated and tested by
operation count with roster, effects and cascade at their caps (gated, P10); state-in/state-out transitions (inputs unmodified); equal-encounter rows
(table 4.11.1); the per-turn Death Saving Throw flag. 9b to 9h: concentration
and duration expiry on the last round; dispel with no matching effect, a
counterspell with no slot or the wrong level, a dispel DC at the boundary; a
zone expiring on its last round; rule interactions between active effects and
conditions; effect lists at the cap and cap plus one (gated, P10); `Fuzz*` for any new parser.
Named boundaries for 9b to 9h: 9b an active effect on its last round and the round after; 9c a granted modifier stacking at the cap of the spec; 9d a zone entered and left on the same turn; 9e a dispel DC and a counterspell level at the boundary; 9f a shape-change on its last round and at 0 Hit Points; 9g revival at the edge of its window; 9h a table roll at its first and last entries.
**Observability:** each follows its spec's section 7. 9a keeps the aggregate's
surviving signals and its `invalid` results (`unknown_combatant`,
`invalid_encounter`, encounter-aggregate FR-095), and removes the superseded
`rejection` and `error_kind` names (P11). 9b to 9h list their operations and
`result` values (expiry, dispel, zone entry). `outcome=failure` is not by itself
an alarm; monitor the `invalid` share and any `errors` increment, and refused
shares by reason.
**Rollback:** each reverts alone. After 9a every later step builds on the
aggregate; changing it is a breaking change within the unreleased module.
**Acceptance:** the aggregate has no class-specific field (a review item, not a test); per PR, every T id
tested.

### 5.10 Step 10: combat-action-resolution (rank 29)

The largest spec (165 ids): resolve a whole combat action by composing the
domain functions of steps 1 to 9 (attacks, damage, conditions, effects, death
saving throws). Size: split into types, then the main resolution paths, then the
spell and Concentration paths (each green) if it exceeds a reviewable size.
**Reuse:** composes domain functions only; calls no use case (§4).
**Tests:** application tests per action kind with fixed rollers; the full
state-in/state-out loop with events; a Death Saving Throw once per turn start
(`DeathSavingThrowAlreadyMade`); spec amendment Q-51 recorded. Unhappy paths,
explicitly: a dead, knocked-out, not-active or unknown `CombatantID` acting (each
refusal asserted by exact value); a cancelled `ctx`; a failing and a panicking
roller and content source (`*srderr.PortError` once, the input `Encounter`
unchanged); Advantage and Disadvantage together and stacked conditions; a
reaction, counterspell or readied-action cascade stopped at the step budget and at budget plus one (gated, P10), with a typed `invalid` result carrying the shared `limit_exceeded` value (if P10 adopts it); a cycle in
containment, mount or summon links rejected iteratively. The plan's scripted
round runs as a scenario in this step, not step 13.
**Observability:** the seven questions, answered from the spec\x27s section 7. Failure
modes: a port failure mid-resolution (`error.type`, once, state unchanged), a
refused action (`refused` class with its result value), an unknown combatant
(`invalid`), a step-budget breach (the shared `limit_exceeded` result value, if P10 adopts it) and
an invalid mid-resolution state. Each entry point is one `Instrumenter`
operation. Step 10 also records the use cases that amendments to
`readying-a-spell` and `telepathy` assign to `combat-action-resolution`
(`ready_combatant_spell`, `release_combatant_spell`, `dissipate_combatant_spell` and `change_telepathic_contact`), so their signals appear
here, not in steps 4d or 7e. New `result` values and event kinds
(`srd5e.combat.events`) are catalogued in the same PR. Monitor the refused share
per action kind and the `errors` counter.
**Rollback:** revert the PR; nothing depends on it until step 11.
**Acceptance:** a scripted round of a two-combatant fight runs through the
public API in a scenario, including one refused action and a death with the turn
advancing; every T id tested and the process-contract unhappy paths covered.

### 5.11 Steps 11 and 12

- **11 combatant-roster-changes.md** (rank 30): add and remove combatants,
  tracker and encounter roster operations; needs 4a, 5b, 6a, 9a and 10.
- **12 summoned-and-companion-creatures.md** (rank 31): `SummonedRoster`,
  `SummonedLink`, `SummonCreature`, familiar rules. Its "Depends on" lists
  `class-warlock.md` and `class-druid.md`, so it does not start until Phase 3
  steps 11b (Druid), 11e (Warlock) and 10 are merged (P1); the Pact of the Chain and Wild
  Companion riders are then T, not D.

**Tests:** step 11 named cases: removing the combatant whose turn it is,
removing the last combatant, adding a duplicate id, adding mid-round, the tracker
index adjusted after a removal, a roster at the cap and cap plus one (gated, P10). Step
12 named cases: a summoned creature outliving its summoner, a duplicate
`SummonedLink`, a link cycle (A summons B summons A) rejected iteratively, summons
per summoner at the cap and cap plus one (gated, P10), and a refused Attack action for a
familiar (FR-048). Both add the process-contract unhappy paths.
**Observability:** each follows its spec's section 7 (operations, `result` values
and extras such as `roster_reason`, `summon_kind` and `summon_end_cause` as closed
enums), counts port failures once, and monitors the refused share by reason and
any `limit_exceeded` result.
**Rollback:** each reverts alone. **Acceptance:** per PR, every T id tested and
the process-contract unhappy paths covered.

### 5.12 Step 13: closing the phase

- Scenarios in `scenarios/` (public API only, few; step 10 already ships the
  scripted-round scenario): a conditioned combatant; a mounted combatant; a spell
  effect with Concentration broken by damage; and one unhappy scenario, a refused
  action followed by a legal one.
- `Example*` functions for the main use cases (`dicetest` roller).
- README, CHANGELOG (check each step added its entry), `roadmap.md` (Phase 4
  status, implementation status roll-up), `doc/documentation_index.md`.
- Re-run every class-dependent D id (P1) against the merged Phase 3 classes and
  move it to T. If those classes are not merged, this step ships without them
  and the affected specs stay `partially implemented`; Phase 4 is not recorded as
  closed while any such id is open.
- Audit: every spec of the header has an accurate `Status` and "Not yet
  implemented" line; `signals.md` equals the constants (O3 is review-enforced,
  so the reviewer diffs it by hand) and holds none of the superseded names
  (P11); name collisions across the package (P7) re-checked beyond the FR-025
  archtest case.
- Record the per-file mutation table from the step PRs in the roadmap; re-run a
  file only if its code changed after its PR.

**Tests:** scenarios and examples only; they supplement, never replace, unit
tests.
**Observability:** the seven questions over the whole phase: a consolidated
"what to monitor" paragraph in `signals.md` for `combat`, using the thresholds
combat-foundations already gives as suggestions (any increment of
`srd5e.combat.errors`; `invalid` above 1% of an operation's calls over 15
minutes; operation duration p99 above 10 ms for 5 minutes), plus the refused
share per operation and reason and any `limit_exceeded` result. `outcome=failure`
alone is not an alarm, because it includes every refusal.
**Rollback:** revert the PR; no code depends on it.
**Acceptance:** `just check` green; `GOARCH=386 go test ./content/... ./combat/...`
green.

## 6. Verification commands

```sh
just check
GOARCH=386 go test ./content/... ./combat/...
go test -race -shuffle=on -coverprofile=coverage.txt ./combat/... && go tool cover -func=coverage.txt | tail -n 1
just mutate ./combat/<file>.go     # Docker only, capped at a quarter of the processors, one file at a time
```

## 7. Security review

`combat` adds no network, storage, identity or secret. Its inputs are **values the
host passes in** (commands, encounter state, combatant ids) and **content decoded
earlier** by the Phase 2 decoder. `combat` is the first context to interpret
monster content, so content-derived numbers are re-validated rather than trusted
(P13); saved encounter state is treated as untrusted input.

| Concern | Worst case | Recommended handling (the specs win; items marked P10 or assumption are proposals) |
|---|---|---|
| Hostile command values (ids, damage, counts, positions) | an invalid state reaches the aggregate, or a panic crashes the host | every constructor validates and returns a typed error (`srderr.ErrInvalidInput`); exported APIs never panic on input; fuzz every new `NewX`/`ParseX` (process contract); tests walk the invalid-input list of `testing.md` §4 |
| Arithmetic overflow | a damage multiplier, Temporary Hit Points sum, duration in rounds, grid distance (`x*x+y*y`) or weight sum wraps and yields a wrong result | grid coordinates are bounded by `grid-positions-and-movement` D-8; bounds on durations and damage are P10 proposals; overflow-safe arithmetic with tests at the limits of each quantity (process contract, "Limits") |
| Resource exhaustion: collections | a host or corrupt saved state passes a huge roster, effect list, summon set, event list or grid footprint into an O(n squared) path (targeting, dispel planning, duplicate-id checks; a 1,000 by 1,000 footprint copied per call) | **P10 (open):** a cap per collection, proposed in P10 for the user to approve or change and then written into the spec, with a typed `invalid` rejection and tests at cap and cap plus one. Until then the affected steps ship without their cap tests. A cap breach uses one shared `limit_exceeded` result value so the `result` set does not multiply per collection |
| Resource exhaustion: cascades and cycles | reaction, counterspell-on-counterspell, readied-action and summon-triggers-summon chains, or a grapple, engulf, mount or summon-link cycle, make one call run unbounded | **P10 (open):** an iterative step budget per call with a typed refusal, and cycle rejection **at link creation** (an invariant of the registry; a saved state that already holds a cycle is rejected by the validation of a registry the root owns, as far as P14 allows; this is a proposal). No recursion on input shape. Tested at budget and budget plus one |
| Dice and randomness | a biased, failing or panicking roller; a roller returning 0, a negative or a value above the die sides; a single expression with a huge dice count | only `dice.Roller`; failure wrapped once with `srderr.WrapPort`; a panic mid-resolution leaves the input state unchanged (state in, state out), tested. **Assumption (A-roll):** an out-of-range roll is treated as a port failure, as `dice` contract tests already require of a roller; hit-points-and-damage treats the d20 face 0 or 21 as invalid input, and the spec wins where the two differ. A dice-count cap joins P10 and P13 |
| Port failure and panics | a content source error or panic escapes as a crash | use cases call `srderr.GuardValue` at the port boundary; recorded once (span, counter), not logged |
| State forgery | a host passes an `Encounter` or registry that breaks an invariant (a duplicate id, a zero value, nil collections, an alias mutated mid-resolution), or a tracker or `Combatant` that was never validated | **Bounded by the specs (P14).** The root validates what it owns (unique `CombatantID`s, its own markers, link graphs at creation) and exposes only opaque values (unexported fields, copies in and out, zero value invalid and documented). encounter-aggregate FR-009 and FR-083 say the root does not re-validate a tracker from `InitiativeOrderEstablished` and that `Combatant` adds no invariant; those specs win. Assumption: validation runs once at entry and once at exit of a call, never inside the cascade loop, at a stated cost tested by operation count with roster, effects and cascade at their caps, not by timing |
| Acting out of turn or for another owner | a command names a combatant that is not the active one, is dead or absent, or a reaction by a non-owner: the library's privilege-escalation analogue | an unknown `CombatantID` is an `invalid` result; a dead, absent-from-turn or not-active combatant is a `refused` result (combat-foundations FR-031 draws the line); never an error, asserted by exact value in steps 6a, 9a and 10 |
| Content-derived numbers | a decoded `Monster` carries hostile counts (a multiattack of 10^9, recharge maxima, long action or spell lists, huge dice expressions) that Phase 2 validated only structurally | P13 (assumption, droppable by the user): `combat` re-validates at `monster-combatant-setup` and where it reads them; a bad value is an `invalid` result. Any content number a spec gives no range for gets a P10 cap before its cap tests land |
| Events | an unbounded or aliasing event list | events per call are bounded by the events cap of P10 (a proposal); returned slices are copies and events hold no pointers into aggregate state (checked by the `combattest` event-family helper) |
| Telemetry disclosure | combatant ids, content ids or free-form keys (for example `OtherEffectOrigin`) in attributes, or high-cardinality values | attributes are low-cardinality enums only (`srd5e.combat.operation`, `result`, `result_class`, outcome); content ids and free-form keys never become attribute values; size histograms use fixed buckets |
| Data provenance | non-SRD or trademarked text enters tests or fixtures | AGENTS.md rule, A10 denylist; fixtures synthetic |
| Authn, injection, cross-tenant data | none | N/A: a library with no I/O; `os`, `net`, `reflect` denied in `combat` (A3) |

## 8. Compatibility and rollout

Before v0.1.0 nothing is published, so every change is an addition and every step
reverts alone. After the first release these are **BREAKING**: a new variant on a
closed result, event or condition set, a new `ConditionEffect` variant, renaming
a signal or attribute, and any change to the `Event` family contract. Because
later steps build on earlier ones, the order in section 5 is also the rollout
order. No step changes a Phase 1, 2 or 3 public API unless its spec says so (a
step that must stops and goes to the user).

Rule-file edits (`architecture.md` rows, `observability.md` §5 values) land with
the code that needs them and refresh the rule tag; a new archtest check that fails
an existing file is fixed in the same PR, never skipped.

## 9. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Size: about 2,700 requirement rows across 36 specs and about 38 PRs | one PR per spec; independent PRs inside a rank; stopping points after steps 3, 6 and 10 |
| `combat-action-resolution` and `magical-effects` are too large for one PR | split into types, rules and spell paths, each green (steps 5.7, 5.10) |
| T/D classification by judgement | re-checked per PR; moves recorded in the requirement-to-test list; class-dependent ids revisited in step 13 |
| Cross-spec amendments requested of one another (P5) | applied in the PR that implements the requesting spec; contradictions needing a decision stop the step and go to the user |
| A late `character` API gap found in Phase 4 | amendment request to the user, never a silent `character` edit in a `combat` PR |
| Gomu is slow on a large package | per-file runs only, one at a time, capped; manual fallback documented |
| One package grows large (36 specs' names) | one file per concept; FR-025 reserved names pinned by archtest; compile time watched in CI; splitting a context is an architecture change and goes to the user |
| Phase 3 not yet finished when Phase 4 starts | P1 gate: Phase 3 steps 0 to 7c merged first; step 12 also waits for Phase 3 steps 11b, 11e and 10 |
| No library-level limits in the specs (P10) | the user approves or changes the proposed caps; each step ships without its cap and budget tests until then and says so; each PR amends its spec when the numbers are set |
| Superseded signal names left in the catalogue (P11) | catalogue check in step 1, per-step migration, audit in step 13 |
| A forged tracker or `Combatant` passes the encounter root unvalidated (P14) | the specs win (encounter-aggregate FR-009 and FR-083); the user decides before step 9a whether to amend them; section 7 covers forgery only as far as the specs allow |

## 10. Out-of-scope observations (recorded, not acted on)

- The `Requirement:` comment archtest check (`testing.md` §9) and the spec-set
  checks FR-052 and FR-056 are still unmechanised; this plan relies on review.
- `toolbox` specs (ranks 40 to 41) depend on `combat` and ship in Phase 5.
- `combat-action-resolution` Q-51 notes that Death Ward, Hit Point maximum and
  revival operations act on host-held values until `encounter-aggregate` gains
  matching methods; that amendment is a follow-up spec change, not part of this
  plan.
