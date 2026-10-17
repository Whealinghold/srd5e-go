# srd5e Roadmap

Status: Phase 1 is implemented in Go and Phase 0 is partly implemented; later phases are proposed.
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
`implemented`. `errors-and-results` and `observability-ports` are partially
implemented (a lint marker and some doc-comment tests are still owed) and
`architecture-guardrails` is partially implemented: `internal/archtest`
enforces architecture.md A1 to A13, and the requirements the spec lists under
"Not yet implemented" (FR-021, FR-022, FR-032 and so on) are still to do. Each spec's `Implementation status` list names the
ids done and not done.

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

**Status: not started** (no `content` package yet).

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

**Status: not started.**

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

**Status: not started.**

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
change as the code. Only the kernel packages and `d20` exist; there are no
`content`, `character`, `combat` or `toolbox` packages yet.

| Status | Count | Specs |
|---|---|---|
| implemented | 4 | `ability-scores`, `d20-tests`, `dice-and-rolls`, `modifiers-and-advantage` |
| partially implemented | 5 | `errors-and-results`, `observability-ports`, `architecture-guardrails`, `shared-types-and-ownership`, `resolve-d20-test-use-case` |
| accepted | 0 | none |
| draft | 92 | every other spec in `specs/`, including `release-0-1-0` |

## Planning notes

1. **Specs first.** Tests must cite requirement ids, so every unit of work has
   a spec in `doc/specs/` (SRD sections and pages, plus FR-ids) before it is
   coded, in `_template.md` shape. Every unit of work now has a spec: the
   foundation, content, character, combat and `toolbox` specs, the class
   behaviour specs, the Rules Glossary coverage table and the spell, monster,
   magic item, summoned creature and grid specs added by the later reconciliation
   passes (listed in [documentation_index.md](documentation_index.md)). All of
   them except the nine implemented or partially implemented specs are drafts.
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
