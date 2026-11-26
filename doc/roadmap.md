# srd5e Roadmap

Status: Phase 1 is implemented in Go. Phase 3 is implemented in Go with open items (see Phase 3 below). Phase 2 is implemented in Go with open items (specs partly; accepted as done by the maintainer on 2026-10-31, see Phase 2 below). Phase 0 is partly implemented. Later phases are proposed.
Phases are ordered by the context map in
[rules/architecture.md](rules/architecture.md) §2, which is itself provisional.
Each phase is a vertical slice that ships tested, instrumented, documented
code, not a layer built in isolation.

[specs/shared-types-and-ownership.md](specs/shared-types-and-ownership.md) is
binding on every phase: it settles type ownership, dependency order and
content delivery, and wins over any other spec.

Every phase follows the process rules in `AGENTS.md`: a spec first, then the
`Testing` and `Observability` evaluations
([rules/testing.md](rules/testing.md) §2,
[rules/observability.md](rules/observability.md) §2), then a `CHANGELOG.md`
entry.

## Phase 0: Foundation (no game rules yet)

- Kernel packages: `dice` (dice expressions and roll results), `modifier`
  (`Modifier`, Advantage / Disadvantage), `ability` (`Ability`, `Score`, `Modifier`, `Scores`), `srderr` (the
  shared error vocabulary). Typed ids arrive with the first context that needs
  them.
- Ports: `dice.Roller` and `telemetry` (`Logger`, `Tracer`, `Meter`) with
  no-op defaults, and the generic `dice.SeededRoller`.
- Test doubles in `<pkg>test` packages: `dice/dicetest` (fixed, callback,
  minimum and maximum rollers), `telemetry/telemetrytest` (recording doubles),
  each with an exported contract suite; `telemetry/slogtelemetry` adapts
  `log/slog`.
- `internal/archtest`: every guardrail from architecture §9 and
  observability §10 that can apply to the code present.
- `doc/observability/signals.md` and the `srderr` error model. A mutation
  testing path (`gomu`, in Docker only) is documented in
  [rules/testing.md](rules/testing.md).

**Outcome:** the rules are enforced by tests before any real logic exists.

**Status: partly implemented in Go.** Three of the six Phase 0 specs
(`dice-and-rolls`, `modifiers-and-advantage`, `ability-scores`) are
`implemented`. `errors-and-results` is partially implemented (its lint marker,
citation scan and a few other ids, 11 in all, are still owed).
`observability-ports` still has the header status `partially implemented`, but
its own "Not yet implemented" line says none, so its status is due an audit and
probably a flip to `implemented`. `architecture-guardrails` is partially
implemented: `internal/archtest` enforces architecture.md A1 to A13, and the
requirements the spec lists under "Not yet implemented" (about 40 ids: FR-021,
FR-022, FR-032 and so on) are still to do. Also not yet mechanised: the
observability checks O1 to O5 and the `Requirement:` comment check
(testing.md section 9); existing tests carry few `Requirement:` comments, so
backfill is pending. Each spec's `Implementation status` list names the ids
done and not done.

Audited against the code on 2026-10-17: `go build`, `go vet`,
`golangci-lint run` and `go test -race -shuffle=on -cover ./...` all pass, and
every package is above the 90% coverage floor.

## Phase 1: `d20` context

**Status: implemented in Go.** `d20` ships ability checks, saving throws
and attack rolls (`Resolver.Resolve`), DCs and ACs, Advantage and
Disadvantage, natural 20 and natural 1 on attack rolls, critical hits, rerolls
and forced failures, with telemetry, table-driven tests and scenarios.

No upstream dependencies, and the core of the ruleset.

- Ability checks, saving throws, attack rolls, DCs, Advantage and
  Disadvantage, success and failure resolution.
- The first use case, with command and result types, telemetry, table-driven
  unit tests and a first scenario test.
- Treat it as the reference implementation that later contexts copy.

## Phase 2: `content` context

**Status: implemented in Go (specs partly), with open items; accepted as done by the maintainer on 2026-10-31 although the mutation gate is unmet on some files.** The `content` package, `content/contenttest` and `content/data` (the 17-entry slice) ship; the specs stay `partially implemented` with their own "Not yet implemented" lines. Mutation results and the open items (the fidelity review of the bundled SRD text against the PDF, which has not been done by a person; the mutation gate unmet on `decode_number.go`, `load_usecase.go` and `catalogue.go`, borderline on `validate_set.go`; killed/total under 80% in the partial run on `class_feature.go`, `class_entry.go` and `class_ability.go`, not investigated; and no mutation run on the per-kind files) are in `doc/plans/plan-phase-2.md` section 11.

- `ContentSource` port (consumer-side interface, `context.Context` first) with a
  `Lookup` result interface whose variants are a closed set.
- JSON decoder reporting all problems at once (`errors.Join`), an in-memory
  source, and a shared contract suite in `content/contenttest`.
- `content` owns every closed SRD vocabulary type and typed id
  (`DamageType`, `ConditionKind`, `Skill`, `ClassID`, ...), so `character` and
  `combat` import them from here.
- A thin slice of bundled SRD data (a few species, one class, a few spells) to
  validate the schema before bulk entry. This adds `NOTICE.md` coverage for the
  bundled content directories.
- `BundledContentSource`, reading the embedded data.

**Settled:** bundled data is JSON under `content/data/` and is embedded
directly with `go:embed` (no generator step, so no drift test); one decoder
serves bundled and host-supplied JSON (shared-types-and-ownership.md FR-030 to
FR-032 already describe this `go:embed` delivery).

## Phase 3: `character` context

**Status: implemented in Go with open items (closed by step 13 of [plans/plan-phase-3.md](plans/plan-phase-3.md)).** The `character` package, the twelve classes with their subclasses (`content/data/class_*.json`) and 73 spells are built; the specs of the phase are `implemented` or `partially implemented` (the table below). What remains open is data, scenarios and the deferred code listed next. Deferred code: none. The Bard `ApplyBardLevelGain` composite of [class-bard.md](specs/class-bard.md) is deliberately not a use case (the host composes `AdvanceBard` with the generic level, spellcasting and feat services, as for the Cleric); its pure Bard rules are built in `character/bard_level_gain.go`. The other class specs' "Not yet implemented" lines were audited: each item is superseded, owned by another spec or the host, or waits on Phase 5 data. Also open: the Phase 5 data (the remaining 266 spells, equipment, feats, backgrounds, magic items, species not yet bundled) that the specs name in their "Not yet implemented" lines, the extraction tool and drift check of content-classes-data.md (FR-086 to FR-092), the full-spell-set class-list coverage (content-classes-data.md FR-100, FR-101), a few unwritten scenarios (a feat taken at level 4, a Wizard scroll copy) and the manual comparison of the class texts against the PDF.

- Proficiency Bonus, level, derived stats (AC, HP, saves,
  skills) and the character aggregate.
- Spellcasting after the basics.
- Level-up and class features, one class at a time. Each class's **data**
  (`ClassDefinition`, progression, features, subclasses; spec
  `content-classes.md`) ships in this phase together with its behaviour spec
  (`class-*.md`), not in Phase 5.
- Species trait behaviour (`species-trait-behaviour.md`): choices, limited
  uses and declared effects of the nine species.
- Spell Scroll use (`spell-scroll-use.md`, rank 15): reading and casting from a
  Spell Scroll, shared with the Rogue and Wizard class specs.
- Magic item effects (`magic-item-effects.md`): typed numeric effects of the
  SRD magic items (bonus items, set ability scores, potions, charged spell
  items, Ring of Spell Storing, extradimensional containers), beside
  `magic-item-rules.md`.

**Type-phase rule (FR-014):** a spec's *types* ship in its context's phase and
bulk *data* may ship later. A spec may depend only on specs whose types ship in
the same or an earlier phase: shared and `d20` types in Phases 0 and 1,
`content` types and vocabulary in 2, `character` in 3, `combat` in 4, `toolbox`
types (after `combat`'s), bulk data and release in 5.

## Phase 4: `combat` context

**Status: partially closed.** Every delivery step of the plan is built (steps 0 to 12, with step 10 in two parts, and step 14) and step 13, the closing audit, is done: the whole-phase scenarios and examples, the README, the CHANGELOG, this roll-up, the spec `Status` audit, the signal catalogue audit and the 32-bit run `GOARCH=386 go test -count=1 ./content/... ./combat/...` (green; `just test-386` and the CI job run both packages). The plan P10 caps are decided and built, each tested at the cap and cap plus one. Of the 36 `combat` specs, 25 are `implemented` and 11 are `partially implemented`; the open items are the D ids that wait for Phase 5 data and the one scenario half named under "Left open" below. Phase 4 is therefore not recorded as closed: it closes with the Phase 5 data. Mutation testing is skipped for this phase by user decision, so no step has a mutation score recorded here, and no step is blocked by that. Plan: [plans/plan-phase-4.md](plans/plan-phase-4.md).


Built (code in `combat/`; each spec's own `Status` line gives its open items):

- Steps 0 to 9c: step 0 groundwork; step 1 `combat-foundations`; step 2 `hit-points-and-damage`; step 3 `conditions`; step 4a `grappling`; step 4b `hiding-searching-and-senses`; step 4c `environment-hazards-and-objects`; step 4d `telepathy`; step 5a `attacks`; step 5b `attack-and-help-actions`; step 5c `mounted-and-underwater-combat`; step 5d `deployed-and-restraining-gear`; step 5e `monster-engulf-and-swallow`; step 6a `initiative-and-actions`; step 6b `item-use-effects`; step 7a `monster-limited-use`; step 7b `magical-effects`; step 7c `movement-lifting-and-flight`; step 7d `grid-positions-and-movement`; step 7e `readying-a-spell`; step 7f `monster-combatant-setup`; step 8a `spell-effects`; step 8b `returning-to-life`; step 8c `monster-stat-block-actions`; step 8d `monster-traits`; step 8e `monster-spellcasting`; step 9a `encounter-aggregate`; step 9b `active-effects`; step 9c `spell-granted-modifiers`.
- Steps 9d to 9h: `persistent-spell-zones`, `dispel-and-counterspell`, `shape-changing-spells`, `revival-and-restoration-spells` and `spell-outcome-tables`. Their pure domain code is in `combat/zone_*.go`, `combat/dispel_*.go`, `combat/shape_shift_*.go`, `combat/revive_*.go` and `combat/spell_outcome_*.go`; the use cases of the zones, shape-changing and revival specs were added by step 10b, and `DispelOngoingSpells`, `CounterCastSpell`, `AssessAntimagicActivity` and `ResolveSpellOutcomeTable` are built with their telemetry.
- Step 10 `combat-action-resolution` (implemented). 10a: the eight use cases `PerformAttack`, `PerformUnarmedStrike`, `TakeCombatantAction`, `MoveCombatant`, `AdvanceCombatTurn`, `RollCombatantDeathSavingThrow`, `AttemptToStabilizeCombatant` and `DealDamageToCombatant`, the ledger and the scripted round of table 8.1. 10b: Concentration, Parry, gear holds, stat-block use and Multiattack, `ResolveStatBlockSave` and the 20 use cases that wrap the pure services of the domain-only specs. FR-109 is built (`scenarios/testdata/golden/combat.txt` and `TestCombatActionResolutionGolden`); the five RA-1 use cases were built by step 14 and the operation lists of FR-168 and FR-170 now name them and `ReviveCombatant` (29 and 30 operations). `DealDamageToCombatant`, `PerformAttack`, `PerformUnarmedStrike`, `RollCombatantDeathSavingThrow` and `ResolveStatBlockSave` are tested at the events cap of 4096 and cap plus one; the other four cannot reach it.
- Step 11 `combatant-roster-changes` (implemented): `AddCombatantToEncounter`, `RemoveCombatantFromEncounter` and `RepositionCombatantInEncounter`, tested at the cap of 256 combatants and cap plus one.
- Step 12 `summoned-and-companion-creatures` (partially implemented, FR-037 data only): `SummonedRoster`, `SummonedLink` and the use cases `SummonCreature` and `ApplySummonedChange`. The class riders need no unbuilt class code (P1), so they are T. Tested at 32 summons per summoner and cap plus one, with link cycles rejected iteratively.
- Step 13 (this audit): `scenarios/` holds the scripted rounds of 10a and 10b, a conditioned combatant (`conditions_test.go`, `encounter_test.go`), a mounted combatant (`mounting_test.go`), a spell effect whose Concentration breaks on damage (`combat_spell_round_test.go`), a hidden attacker (`hiding_test.go`) and one unhappy scenario, a refused command followed by a legal one (`combat_refusal_test.go`); the package examples include `ExamplePerformAttack`, `ExampleTakeCombatantAction`, `ExampleAdvanceCombatTurn`, `ExampleApplyConditionToSet`, `ExampleRollInitiative` and `ExampleResolveAttack`; the "What to monitor for `combat`" section of `doc/observability/signals.md` is written and `TestSignalCatalogueEqualsTheConstants` pins the catalogue to `combat/signals.go`. The class-dependent D ids were re-checked against the merged `character` classes: none needs unbuilt class code, so none moves to T (see below).
- Step 14 `magical-effects` use cases (RA-1, FR-148, FR-144): `RemoveCombatantCurse`, `ShiftCombatantShape`, `TeleportCombatant`, `ResolveEffectOrder` and `AssessAreaLocation` (`combat/magical_effects_usecase.go`), each running in one `CombatTelemetry` operation and returning the owner's own closed result. `magical-effects` is now implemented.

Left open (the Phase 4 close is blocked by these):

- D ids that wait for Phase 5 data (bundled monsters, equipment and spells), named in each spec's `Not yet implemented` line: `deployed-and-restraining-gear` D-1 and D-2; `item-use-effects` D-1 (FR-005); `monster-limited-use` D-1; `monster-combatant-setup` D-1; `monster-stat-block-actions` D-1; `spell-effects` D-1 (D-2 is closed: the scenario casts through the real `CastSpell`); `monster-traits` FR-094; `monster-spellcasting` D, the contract test over the bundled stat blocks; `shape-changing-spells` D-1 (the True Polymorph, Shapechange and Animal Shapes rows); `spell-outcome-tables` FR-054 (five of the nine Reincarnate species are not bundled); `summoned-and-companion-creatures` FR-037 for the Animated Object and the Giant Insect.
- Spec-only amendments that `spell-outcome-tables` section 9 asks of seven other specs (`spell-effects`, `spell-effect-data`, `magical-effects`, `shared-types-and-ownership`, `release-0-1-0`, `architecture-guardrails`, `rules-glossary-coverage`) are not made; they change no code.
- The `character` amendment of `monster-spellcasting` (amendment A1) is built: `character.CastSourceMonster`, the exact constructors `NewSpellSaveDC` and `NewSpellAttackBonus`, and `MonsterCastReport.ToCastReport`, so a monster cast reaches Counterspell and `ResolveSpellEffects`; the user approved it. No D id waits on a `character` amendment now.
- Decided, not open: the variant-prefix half of `combat-foundations` FR-022 is not scanned (a decided waiver, recorded in that spec); `combat-action-resolution` FR-156 is review-only; the operation-count cost test of the whole active-effects registry is not written (`active-effects` section 11). The 32-bit run is no gap: `just test-386` and the CI job `test-386` run `GOARCH=386 go test ./content/... ./combat/...`.
- Mutation testing is skipped for this phase by user decision. No step's mutation score is recorded, and that blocks no step.


Suggested order:

1. Hit points and damage.
2. Attacks (reusing `d20`).
3. Conditions.
4. Initiative and actions.
5. The encounter aggregate.

Domain events such as `DamageTaken` and `ConditionApplied` matter most here.

Monster abilities that need the `combat` types ship here too: limited use
(`monster-limited-use.md`, rank 26) and monster spellcasting
(`monster-spellcasting.md`, rank 27, SRD p. 257).

## Phase 5: Breadth and release

**Status: not started.**

- Remaining bulk data: spells, equipment, monsters and magic items from the
  SRD. (Class data is delivered in Phase 3.)
- Magic Items, the Gameplay Toolbox and the Rules Glossary are in scope, each
  with its own spec before any code: magic items as a `content` kind (SRD
  5.2.1, pp. 204-253), the Gameplay Toolbox (pp. 192-203) as the `toolbox`
  context, and the Rules Glossary (pp. 176-191) as the reference for terms the
  other specs cite. They land in the phase of the context they belong to, under
  the same type-phase rule.
- `toolbox` context (shared-types-and-ownership.md FR-045): downstream of
  `content`, `d20`, `character` and `combat`, with nothing depending on it.
  Its types ship in this phase, after `combat`'s. It hands results to the host
  (an Exhaustion level, a condition to apply) and mutates no character or
  combat state. Named catalogue entries (poisons, traps, contagions) are
  content kinds.
- `RuleOptions` for the optional rules the SRD itself names.
- `Example*` functions as testable docs, including observability bridge examples.
- Tag v0.1.0 (a git tag; the Go module proxy publishes it).

## Implementation status

Roll-up of the `Status` row in each spec header (values in
[specs/_template.md](specs/_template.md)). A spec flips status in the same
change as the code. The kernel packages, `d20`, `content` and `character` exist; `combat` package exists (steps 0 to 14, see the Phase 4 entry); there is no `toolbox` package yet.

| Status | Count | Specs |
|---|---|---|
| implemented | 37 | `ability-scores`, `d20-tests`, `dice-and-rolls`, `modifiers-and-advantage`, the 25 `combat` specs `active-effects`, `attack-and-help-actions`, `attacks`, `combat-action-resolution`, `combat-foundations`, `combatant-roster-changes`, `conditions`, `dispel-and-counterspell`, `encounter-aggregate`, `environment-hazards-and-objects`, `grappling`, `grid-positions-and-movement`, `hiding-searching-and-senses`, `hit-points-and-damage`, `initiative-and-actions`, `magical-effects`, `monster-engulf-and-swallow`, `mounted-and-underwater-combat`, `movement-lifting-and-flight`, `persistent-spell-zones`, `readying-a-spell`, `returning-to-life`, `revival-and-restoration-spells`, `spell-granted-modifiers` and `telepathy`, and the `character` specs `casting-and-concentration`, `class-warlock`, `class-wizard-evoker`, `exploration`, `heroic-inspiration`, `higher-level-start-equipment-and-trinkets`, `social-interaction`, `wish-spell` |

| partially implemented | 53 | `errors-and-results`, `observability-ports`, `architecture-guardrails`, `shared-types-and-ownership`, `resolve-d20-test-use-case`, the 11 `combat` specs `deployed-and-restraining-gear`, `item-use-effects`, `monster-combatant-setup`, `monster-limited-use`, `monster-spellcasting`, `monster-stat-block-actions`, `monster-traits`, `shape-changing-spells`, `spell-effects`, `spell-outcome-tables` and `summoned-and-companion-creatures`, the other `character` specs (`equipment-rules`, `derived-stats`, `character-aggregate`, `character-creation`, `rests-and-hit-dice`, `weapon-properties-and-mastery`, `spellcasting`, `magic-item-rules`, `magic-item-effects`, `level-advancement-and-multiclassing`, `tool-proficiency-and-crafting`, `feats-behaviour`, `spell-scroll-use`, `species-trait-behaviour` and the eleven `class-*` specs except `class-warlock` and the Evoker) and the `content` specs (`content-schema-and-decoder`, `content-source-port`, `content-species-backgrounds-feats`, `content-classes`, `content-classes-data`, `content-equipment`, `content-magic-items`, `content-monsters`, `spell-effect-data`, `content-spells`, `content-thin-slice`, `content-licensing-and-notice`) |

| accepted | 0 | none |
| draft | 11 | `release-0-1-0`, `rule-options`, `rules-glossary-coverage` and the eight `toolbox-*` specs |

## Planning notes

1. **Specs first.** Tests must cite requirement ids, so every unit of work has
   a spec in `doc/specs/` (SRD sections and pages, plus FR-ids) before it is
   coded, in `_template.md` shape. Every unit of work now has a spec: the
   foundation, content, character, combat and `toolbox` specs, the class
   behaviour specs, the Rules Glossary coverage table and the spell, monster,
   magic item, summoned creature and grid specs added by the later reconciliation
   passes (listed in [documentation_index.md](documentation_index.md)). All of
   them except the 90 implemented or partially implemented specs are drafts.
   The remaining work is implementation, phase by phase, each spec flipping to
   `implemented` in the change that ships it. New gaps found while coding get a new spec or a new
   requirement id (never a renumbering), with its rank in
   `shared-types-and-ownership.md` section 4.11.
2. **One plan per phase,** sized to a reviewable PR sequence.
3. **Revisit the context map early.** For example, does `character` need
   `d20` upstream, or only the kernel packages? Changing the map means
   updating the architecture table, `AGENTS.md` and the architecture tests
   together.
4. **Content volume is the biggest risk.** Transcribing SRD content faithfully
   is the largest workload. The data format and delivery are settled (Phase 2,
   FR-030 to FR-034); scale up only through that path.
5. **Start with Phases 0 and 1 together,** since Phase 0 has no value on its
   own.
