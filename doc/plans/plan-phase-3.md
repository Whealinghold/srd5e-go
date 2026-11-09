# Phase 3 plan: the `character` context and the class data

| | |
|---|---|
| Status | Done (step 13 closed the phase; see [../roadmap.md](../roadmap.md) and the "Not yet implemented" lines of the specs). |
| Roadmap phase | 3 ([../roadmap.md](../roadmap.md)) |
| Specs sequenced | `character` (rank 9 to 18): [equipment-rules](../specs/equipment-rules.md), [derived-stats](../specs/derived-stats.md), [character-aggregate](../specs/character-aggregate.md), [rests-and-hit-dice](../specs/rests-and-hit-dice.md), [heroic-inspiration](../specs/heroic-inspiration.md), [weapon-properties-and-mastery](../specs/weapon-properties-and-mastery.md), [social-interaction](../specs/social-interaction.md), [exploration](../specs/exploration.md), [spellcasting](../specs/spellcasting.md), [magic-item-rules](../specs/magic-item-rules.md), [level-advancement-and-multiclassing](../specs/level-advancement-and-multiclassing.md), [casting-and-concentration](../specs/casting-and-concentration.md), [magic-item-effects](../specs/magic-item-effects.md), [tool-proficiency-and-crafting](../specs/tool-proficiency-and-crafting.md), [feats-behaviour](../specs/feats-behaviour.md), [higher-level-start-equipment-and-trinkets](../specs/higher-level-start-equipment-and-trinkets.md), [spell-scroll-use](../specs/spell-scroll-use.md), [character-creation](../specs/character-creation.md), [wish-spell](../specs/wish-spell.md), the twelve `class-*` specs plus [class-wizard-evoker](../specs/class-wizard-evoker.md), [species-trait-behaviour](../specs/species-trait-behaviour.md). `content` (rank 7): [content-classes-data](../specs/content-classes-data.md), and the spell data it needs from [content-spells](../specs/content-spells.md). Rules used: [rule-options](../specs/rule-options.md) FR-029 only. |
| Depends on | Phases 0 to 2 (kernel, `telemetry`, `srderr`, `d20`, `content`, `content/data`, `internal/archtest`) |
| Format | follows [plan-phase-2.md](plan-phase-2.md); there is no `doc/plans/README.md` |

Naming note: `A1` to `A13` always mean the **architecture checks** of
`architecture.md` section 9. Plan items are numbered `P1` onward.

## 1. Goal

Ship the `character` bounded context (one Go package, `architecture.md` §2):
levels and derived stats, the character aggregate and its events, rests, Heroic
Inspiration, spellcasting and Concentration, equipment and magic-item rule maths,
feats, level-up and multiclassing, character creation, and the behaviour of all
twelve classes and the nine species. Also ship the class and subclass **data** for
all twelve classes (roadmap Phase 3), with the spells that data refers to.

This plan sequences decisions already made in the specs and in
[../specs/shared-types-and-ownership.md](../specs/shared-types-and-ownership.md)
(binding). It decides nothing; where a spec and this plan disagree, the spec wins.
Every Phase 3 spec records "no question open" in its section 9, so the gaps below
are the plan's own findings.

## 2. What already exists (reused as-is)

| Asset | Where | Used for |
|---|---|---|
| `d20.ResolveCheck`, `ResolveAttack`, `FailWithoutRoll`, `Roll` (reroll operations), `RollModifiers`, `DifficultyClass`, `Resolver` and its `WithTelemetry` / `WithClock` options | `d20/` | every save, check and attack in `character`; the template for each use case's constructor and options (FR-044: domain functions are composed, `Resolver` is never called) |
| `content.Source`, `Loader`, `Set`, `Catalogue`, `InMemorySource`, `JSONDecoder`, `ValidateSet`, `MissingContent`, `ClassDefinition`, `Species`, `Background`, `Feat`, `EquipmentItem`, `MagicItem`, `Spell` and the vocabulary enums | `content/` | all content access; class tables read through `ClassDefinition.RowAt` and friends, not re-parsed |
| `content/enum.go` shared enum helper | `content/` | pattern for `character`'s own closed enums (it is unexported, so copy the shape, do not import) |
| `contenttest.FakeSource`, `RunSourceContract`; `content.InMemorySource` | `content/contenttest`, `content/` | fixtures for use-case tests |
| `dicetest.FixedRoller`, min/max and callback rollers | `dice/dicetest` | every roll in tests |
| `ability.Score`, `Scores`, `Modifier`; `modifier.Modifier`, `RollCircumstance`, `RollMode`; `dice.Expression`, `Roller` | kernel | derived stats, Hit Dice, Spell Scroll rolls |
| `srderr.WrapPort`, `GuardValue`, `InvalidError` shape (`ability/error.go`) | `srderr/`, kernel | all port failures and validation errors |
| `telemetry.Telemetry`, `Noop`, `StartTimer`; `telemetrytest` doubles | `telemetry/` | every use case's signals and their tests |
| `internal/archtest` `DefaultPolicy` row `character` (imports `content`, `d20`, kernel) | `internal/archtest/policy.go:58` | A1 already allows the package; no new import edge is needed |
| `data.BundledSource`, `species.json`, `spells.json`, `class_wizard.json`, the pin tests | `content/data/` | step 10 and 11 extend these files, they do not add a second loader |

No kernel change is expected. If a step finds one is needed it stops and goes to
the user (FR-001: the kernel stays small).

## 3. Scope fence

**In scope:** the Phase 3 requirements of the specs in the header; the bundled
data of the 11 classes not yet bundled and the full Wizard; the spells those class
files refer to (P2); `NOTICE.md` coverage of whatever they add; the doc, policy and
catalogue edits each step needs.

**Out of scope** (each is tracked where it already lives):

- Bulk data for species, backgrounds, feats, equipment, magic items and monsters,
  and the rest of the 339 spells (Phase 5). Requirements that need them
  ("over the bundled data", all 25 tools, nine species) are classed **D** and
  listed in that spec's "Not yet implemented" line (P4).
- Everything in `combat` (Phase 4): attacks, conditions, initiative, hit point
  damage application, the encounter aggregate. Specs of this plan only produce
  values `combat` consumes (`shared-types-and-ownership.md` FR-011).
- The `toolbox` context (Phase 5).
- The extraction tool `tools/extract_classes` and the PDF drift check
  (content-classes-data FR-086 to FR-092) (P3).
- Any release, tag or version bump (Phase 5).

## 4. Open items, assumptions and spec defects

**Decided**: a spec or the user settled it. **Assumption**: a sequencing choice
the user can override. **Spec defect**: fix in the spec in the same PR as the code
that hits it. **OPEN**: needs the user before the named step.

| # | Item | Class | Handling |
|---|---|---|---|
| P1 | Scope is the roadmap's Phase 3 bullet list plus the type-phase rule (`shared-types-and-ownership.md` FR-014) | decided (roadmap) | section 3 |
| P2 | Class files for Bard, Cleric, Druid, Paladin, Ranger, Sorcerer and Warlock name about 70 spells in `AlwaysPreparedSpells` facts (content-classes-data Table FA); the bundle has 12 spells and `ValidateSet` rejects an unresolved spell id (FR-054) | decided (user, 2026-11-01): bundle the referenced spells | step 10 bundles exactly the spells Table FA names, no others; the rest of the set stays Phase 5. Exact count is derived from the class files by a test, not hard-coded in this plan |
| P3 | content-classes-data FR-086 to FR-092 require a drift-checking extraction tool in `tools/`; Phase 2 also left all tools to Phase 5 | assumption: defer the tool, keep the human check | class and spell JSON is hand-transcribed and checked entry by entry against the PDF by a person, plus the hand-written oracle (FR-093) which lands with each class. Step 9a amends content-classes-data FR-086 to FR-092 to say the tool is Phase 5. **Override this if you want the tool in Phase 3** (adds a nested module and a CI step) |
| P4 | Some requirements cannot be satisfied with the Phase 2 bundle (nine species, all tools, full equipment tables) | assumption: classify each requirement T (types and rules, testable with synthetic fixtures) or D (needs Phase 5 data) per step, as Phase 2 did | the implementer re-checks every id; D ids are named in the PR `Testing` entry and in the spec's "Not yet implemented" line. Fixtures are synthetic or SRD slice entries, never SRD text outside `testdata/srd/` |
| P5 | content-classes-data FR-100 staging pin is written as "bundled spell count still the slice's"; step 10 changes that count | spec defect | step 10 changes the pin to "the full 339-spell set is not yet bundled" and amends FR-100 in the same PR |
| P6 | Specs list follow-up edits to other specs they assume applied (species Q8, feats-behaviour Q1, Q3, Q8 to Q11, Q17, Q20, Q21; level-advancement Q-9, which may be BREAKING once code ships; spell-scroll-use outstanding items; character-aggregate's Sorcerer slot and Wish items) | decided (the specs) | each step applies the edits that concern the spec it implements in the same PR; edits to `combat` specs are spec-only and change no code |
| P7 | `architecture.md` §1 has no `character` row in either table; AGENTS.md "Directory structure" has no `character/` | spec defect | step 0 |
| P8 | `character` is one package that will hold about 12 classes' domain code, so names must stay unique across it (FR-021) and file count is large | decided (`architecture.md` §2: one package per context) | one file per concept (go.md §1); step 13 checks name uniqueness is enforced by the compiler plus the existing A7/A9 checks |
| P9 | Order inside a rank is not fixed by the specs | assumption | the order in section 5 is a suggestion; merge any order that respects `Depends on` (FR-016) |
| P10 | A package-level `var` for progression, slot or cost tables breaks A4 | decided: A4 wins (Phase 2 P6) | `const` blocks, funcs and `switch` only; amend any spec wording that says "table variable" in the PR that hits it |

No step is blocked on a user decision; P3 and P9 are the two overrides worth a look
before work starts.

## 5. Delivery sequence

About 35 reviewable PRs, each green on its own. Order is forced by imports and by
the intra-context order `derived-stats` ← `character-aggregate` ← the rest ← `class-*`
(`shared-types-and-ownership.md` FR-016, rank table 4.11).

| Step | Ships | Rank | Why here |
|---|---|---|---|
| 0 | Groundwork: docs, policy, `character/doc.go` | n/a | the first production file must already be inside the allow-list |
| 1 | equipment-rules | 9 | pure services; `derived-stats` calls Armor Class maths |
| 2 | derived-stats | 10 | `Level`, `ClassLevels`, `ProficiencyBonus`, `HitPointMaximum`, `IsBloodied`; the aggregate stores these |
| 3 | character-aggregate | 11 | `Character`, events, `FeatureStates`, `CharacterRuleOptions`; root of everything below |
| 4a to 4e | rests-and-hit-dice, heroic-inspiration, weapon-properties-and-mastery, social-interaction, exploration | 12 | independent of one another |
| 5a, 5b | spellcasting, magic-item-rules | 13 | independent of one another |
| 6a to 6d | level-advancement-and-multiclassing, casting-and-concentration, magic-item-effects, tool-proficiency-and-crafting | 14 | independent of one another |
| 7a to 7c | feats-behaviour, higher-level-start-equipment-and-trinkets, spell-scroll-use | 15 | independent of one another |
| 8a, 8b | character-creation, wish-spell | 16 | creation needs 7a, 7b; Wish needs 7a, 6b |
| 9a to 9f | Fighter (reference implementation, FR-023), Barbarian, Monk, Rogue, full Wizard, Evoker, each with its class and subclass JSON | 17, 18 | these six classes name no spell, so their data closes without step 10 |
| 10a to 10c | the spells named by the other seven class files, in three PRs by spell level | 6 data | must exist before 11 |
| 11a to 11g | Cleric, Druid, Bard, Sorcerer, Warlock, Paladin, Ranger, each with its JSON | 17 | spell closure from step 10 |
| 12 | species-trait-behaviour | 17 | needs 4a, 4b, 5a, 6b, 7a, 8a; independent of the classes, so it may merge any time after 8a |
| 13 | Whole-phase scenarios, examples, README, CHANGELOG, roadmap, spec status audit | n/a | closes the phase |

Natural stopping points that leave a coherent library: after step 3 (a character
that levels and has derived stats), after step 8 (everything except class
behaviour), after step 9 (six playable classes).

### Process contract for every step

- Read the four rule files first (`AGENTS.md`); never quote a `RULE-TAG` in a plan
  or PR text (A13 requires each value to appear in one place only).
- Every new test cites its requirement (`Requirement:` and `Path:`;
  `Requirement: per case` on tables). `testing.md` §5 is still "planned", so apply
  it to all new tests.
- Each PR carries a **requirement-to-test list**: every T id of the spec maps to a
  test; D ids and any id deferred are named in the `Testing` entry.
- `just check` green. Each changed package at least 90 percent statement
  coverage, reported per package.
- Mutation testing with `gomu`, only inside Docker, `--cpus` and `--workers`
  capped at a quarter of the processors (`just mutate ./character/<file>.go`
  per file, as Phase 2 did; whole-package runs are too slow at this size). Score at
  least 80 for domain files; survivors fixed unless provably equivalent. Where gomu
  cannot parse a file, use the manual-mutation fallback of `testing.md` §8 and list
  the mutations. A file with no same-named `_test.go` gets none from gomu, so name
  test files after the source files.
- `CHANGELOG.md` entry under `Unreleased` (the spec's own section 10 gives the text).
- The PR description carries `Testing` and `Observability` entries. The
  `Observability` entry answers **all seven questions** of `observability.md` §2;
  a bare "none" or "silent by design" is rejected.
- Signals: every new signal is a constant in `character/signals.go`, a row in
  `doc/observability/signals.md` with purpose and suggested monitoring, and a pin
  test. New attribute values are added to `observability.md` §5 in the same PR (tag
  refreshed).
- A spec's `Status` becomes `partially implemented` in the PR that first ships code
  for it, with an `Implementation status` list and a `Not yet implemented:` line.
  It becomes `implemented` only when that line is empty (specs needing Phase 5 data
  stay `partially implemented`).
- Fixtures are synthetic or the bundled slice; nothing from outside SRD 5.2.1; no
  consumer names.
- Pure domain files import no `telemetry` (O1, still review-enforced); use-case
  files hold every `telemetry` call and every `srderr.WrapPort` call (A6).

### 5.0 Step 0: groundwork

- `doc/rules/architecture.md` §1: add the `character` row to the package table and
  to the dependency-rule table (`content`, `d20`, kernel; no `*test` in production
  files). The §2 context map already lists it. Refresh the `RULE-TAG`.
- `AGENTS.md` "Directory structure": add `character/`. Leave the "Read the rules
  first" list untouched.
- `character/doc.go`: package comment naming inputs, outputs and handoff types
  (grows with each step).
- `internal/archtest`: extend the existing `character` policy row only if a step
  later needs a package that is not yet allowed (none expected).

**Tests:** archtest table rows: a `character` production file importing `combat`
(violating) and importing `content` and `d20` (clean).
**Observability:** the seven questions, answered. What can go wrong: a `character`
file imports a downstream context or a `*test` package (A1, A2). Operator signal:
`just arch` fails. Nothing to count or time; no existing signal affected; no
consumer monitoring; nothing noisy.
**Rollback:** revert the PR (docs and an empty package).
**Acceptance:** `just arch` passes with `character/doc.go` only.

### 5.1 Step 1: equipment-rules (rank 9)

Spec `equipment-rules.md`, all T ids: `ArmorRules`, `CarryingCapacity`, `DrawnLoad`,
`Barding`, `ShipRepair`, `Crafting`, `ScrollScribing`, `SpellcastingServices`,
`ArmorTrainingRules`, `ArmorRules.StealthCircumstances`. Stateless zero-size
services over `content` equipment values.
**Reuse:** `content.EquipmentItem` accessors, `modifier.RollCircumstance`,
`ability.Modifier`.
**Tests:** unit only, plain values, table-driven with a boundary row per threshold
(Strength 8 against Plate Armor's speed rule, DEX caps, Carrying Capacity size
multipliers incl. the Gargantuan ceiling, the Q15 rounding cases 15 GP and 1 CP and
5 CP). Immutability: returned slices are copies. D ids: the "over the bundled data"
rows (FR-064, the full-table coverage errors) run on synthetic fixtures and the
Phase 2 slice; the all-rows runs are deferred to Phase 5.
**Observability:** silent: pure domain. Failure modes are returned typed errors or
result values; callers (steps 2, 5a, 6d) count them. Nothing noisy.
**Rollback:** revert the PR.
**Acceptance:** every T id has a test; `just check` green.

### 5.2 Step 2: derived-stats (rank 10)

Spec `derived-stats.md`: `Level`, `ClassLevels` (+`WithLevelIn`),
`ProficiencyBonus`, `HitPointMaximum`, `HitDicePool` (+`Plus`), `HitPointGain`,
`IsBloodied` (FR-049), `ModifierBreakdown.ToRollModifiers()`, and the use case
`CalculateDerivedStats` with signals `srd5e.character.derived_stats.*`.
**Reuse:** step 1 `ArmorRules` for Armor Class; `d20.RollModifiers`; the
`d20.NewResolver` option shape for the use case; `content.ClassDefinition`.
**Tests:** unit for every value object (levels 0, 1, 20, 21; total 20 across
classes; the multi-problem row), application tests for the use case with
`contenttest.FakeSource` returning found, missing content, and a failing source
(`*srderr.PortError` once), `telemetrytest` assertions on the signals, Noop versus
recording bundle give an identical result, an already-cancelled `ctx`. A
`Fuzz*` test for each new `ParseX`.
**Observability:** use case instrumented per spec §7. Failure modes: missing class
content (counted by `srd5e.outcome=failure` and a variant attribute, no `error.type`),
port failure (`error.type=content_error`, span `RecordError`, no log). Consumers
should alert on a sustained non-zero missing-content rate.
**Rollback:** revert the PR.
**Acceptance:** `IsBloodied` is the only Bloodied predicate in the module (grep in
review); `just check` green.

### 5.3 Step 3: character-aggregate (rank 11)

Spec `character-aggregate.md`: `Character`, `CharacterID`, `ExperiencePoints`,
`HeroicInspiration` (enum slot), `FeatureStates` and `FeatureStateKey` (FR-022),
`ClassCreationChoices` (FR-024), `CharacterRuleOptions` (rule-options FR-029),
the seven events and `character.Event`, use cases `LevelUpCharacter`,
`AwardExperience`, `ChangeAbilityScore`, and the shared signals
`srd5e.character.feature_used`, `.duration`, `.errors` that every class reuses
(FR-063).
**Reuse:** step 2 types; `d20` reroll operations take `HeroicInspiration` as input
(FR-010).
**Tests:** unit for the aggregate's invariants and state-in/state-out transitions
(each returns new state plus events; inputs unmodified), `FeatureStates` typed-key
store with two foreign keys proving no `any` leak, use-case tests as step 2. Any
variant added to `BonusFeatsAtLevel20` later is breaking: pinned by a test.
**Observability:** use cases instrumented. Failure modes: rejected level-up
(multiclass prerequisite, level 20 cap) are failure results counted by variant, not
errors; port failure as step 2. Consumers monitor the `rejection` rate by reason.
**Rollback:** revert the PR. After this step every later step builds on this API:
changing it later is a breaking change within the unreleased module.
**Acceptance:** the aggregate has no class-specific field; `just check` green.

### 5.4 Steps 4a to 4e: rank 12 (one PR each)

| Step | Spec | Notes specific to the PR |
|---|---|---|
| 4a | `rests-and-hit-dice.md` | `RestKind`, `TakeRest`, `SpendHitPointDie` (uses `dice.Roller`), `RestApplied`; interruption rows (Long Rest, Trance); Hit Die roll tests use `dicetest` min/max |
| 4b | `heroic-inspiration.md` | owns mechanics, the state slot is step 3's; swap window until used or next Long Rest (no clock) |
| 4c | `weapon-properties-and-mastery.md` | `WeaponMasteryChoice`; weapon proficiency; the property-rules attack plan; mastery effects reported as offers (data, never applied); plan types are `character` published language that `combat` maps in Phase 4 |
| 4d | `social-interaction.md` | attitude (Friendly, Indifferent, Hostile) and the Influence action; the Hesitant-monster check resolved by `d20` domain functions; no state mutation |
| 4e | `exploration.md` | index of the Exploration section; owns Travel Pace (`TravelRules.Circumstances`, `VehicleSpeed`) and light sources; pure functions over host-supplied values, no dice, no port |

**Tests:** unit for rule maths, application tests only where a spec defines a use
case (4a, 4c), boundary and rule-interaction rows per testing.md §4; D ids named.
**Observability:** 4a instruments `TakeRest` and `SpendHitPointDie` (rests,
`hit_point_dice_spent`, durations); a Short Rest with zero Hit Dice available is a
result, not an error. 4b to 4e are pure and silent: failure modes are returned
values, nothing to count here, existing signals unaffected.
**Rollback:** each PR reverts alone.
**Acceptance:** per PR, every T id of the spec has a test.

### 5.5 Steps 5a, 5b: rank 13

- **5a spellcasting.md**: spell slots, Pact Magic, multiclass slots, save DC and
  attack bonus (with `RollModifiers`), prepared spells and cantrips, `Concentration`
  state shape (FR-011), the eight use cases (`CalculateSpellcasting` ... `AdvanceSpellcasting`).
  Slot and preparation tables read from `content.ClassDefinition` rows, never
  duplicated as Go tables (P10).
- **5b magic-item-rules.md**: rarity resolution, Potion Miscibility, activation,
  Attunement, identification, cursed items, resilience, crafting, sentient items;
  host-supplied `MagicItemFacts`; ten use cases.

**Tests:** table-driven for the slot tables at every level 1 to 20 and for
multiclass slot combinations (boundary: total level 20, one level in each caster
type); use cases with `FakeSource` and failing source; attunement limit boundary
(3 and 4); spell-casting rows that need full spell lists are D and use the 12 slice
spells plus synthetic ones.
**Observability:** use cases instrumented; counts by outcome; monitor the
`rejection` reasons (no slot, not prepared, attunement limit). A rejected cast is a
normal outcome, never an error.
**Rollback:** revert the PR; 5a is a dependency of 6a, 6b, 6c, 6d.
**Acceptance:** per PR, every T id tested; slot tables equal the SRD tables for all
levels.

### 5.6 Steps 6a to 6d: rank 14

| Step | Spec | Notes |
|---|---|---|
| 6a | `level-advancement-and-multiclassing.md` | `PreviewLevelAdvancement`, `ChooseSubclass`, `SubclassChosen`, `SubclassRejected`, `AbilityScoreIncrease`, `MulticlassTraining`, `ExtraAttackCombination`; apply Q-9 edits to other specs |
| 6b | `casting-and-concentration.md` | `CastSpell`, Concentration rules, scroll casts (`CastModeSpellScroll`), Casting in Armor via step 1 `ArmorTrainingRules` |
| 6c | `magic-item-effects.md` | numeric effects of items; `RollPotionOfHealing` (dice), charged items, Ring of Spell Storing, extradimensional containers |
| 6d | `tool-proficiency-and-crafting.md` | `ToolProficiencies`, `ResolveTool`, Utilize DCs, `PlanItemCrafting`, `BeginScribing`; all-25-tools rows are D |

**Tests:** as step 5, plus for 6b: Concentration save DC boundary (damage 1, 20, 21,
and the cap), a spell cast with the Concentration state already set (replaced),
`ctx` cancelled; for 6c: dice-driven rows with fixed and min/max rollers.
**Observability:** 6a to 6c use cases instrumented; monitor `rejection` reasons and
unexpected `outcome=failure` after a content change. 6d is pure and silent.
**Rollback:** each reverts alone.
**Acceptance:** per PR, every T id tested.

### 5.7 Steps 7a to 7c: rank 15

- **7a feats-behaviour.md**: five feat use cases, `TakenFeats`, `FeatGrant`,
  `FeatChoices`, `FightingStyleChoice`; Repeatable feats; prerequisites from
  `content.Prerequisite`. Feats beyond the bundled slice are D.
- **7b higher-level-start-equipment-and-trinkets.md**: starting equipment at levels
  above 1, magic item allowance, trinket roll (`dice.Roller`).
- **7c spell-scroll-use.md**: who can read a scroll, the ability check for
  higher-level spells (`d20.ResolveCheck`), DC and attack bonus, post-cast outcome,
  interruption; pure domain functions.

**Tests:** as step 5; 7b's trinket table with a min and max roll; 7c's check
boundary (DC equals, one below).
**Observability:** 7a, 7b instrumented per spec §7; 7c silent (pure; its casts are
counted by 6b). Monitor `rejection` reasons.
**Rollback:** each reverts alone.
**Acceptance:** per PR, every T id tested.

### 5.8 Steps 8a, 8b: rank 16

- **8a character-creation.md**: `CreateCharacter`, ability score generation
  (Standard Array, `RollAbilityScores`, Point Cost), background increases,
  `RollLanguages`, `CreationRecord`; exports `AbilityScoreGenerationMethod`.
- **8b wish-spell.md**: the Wish spell's rule branches as pure functions.

**Tests:** application tests with `FakeSource` carrying a species, background and
class from the bundle; roll-based rows with `dicetest` (4d6 drop-lowest at min,
max, ties); invalid method per `CharacterRuleOptions`; missing content per kind.
**Observability:** `CreateCharacter`, `RollAbilityScores`, `RollLanguages`
instrumented; monitor creation `outcome=failure` by variant (spikes after a content
change point at bad data). 8b silent.
**Rollback:** each reverts alone.
**Acceptance:** per PR, every T id tested; a whole creation scenario for a bundled
Wizard passes with public API only (scenario, few).

### 5.9 Steps 9a to 9f: classes whose data closes without new spells

Order: Fighter (reference implementation for FR-023, copied by the rest), Barbarian,
Monk, Rogue, full Wizard (replaces the slice's `class_wizard.json`, the spec says the
slice entry "becomes the full Wizard entry"), then Evoker (rank 18; its subclass
data lives in the Wizard file).

Each PR ships, from its `class-*.md`: `<Class>Level`, `<Class>Progression`,
`<Class>State` (a `FeatureState` with a typed key), `<Class>CreationChoices`,
domain services as pure functions (FR-023), the use cases it defines, its events,
and `content/data/class_<id>.json` (class and its one SRD subclass) with the rows of
the oracle `content/testdata/srd/classes/oracle.json` for that class (written by hand,
independent of the JSON; FR-093).
**Reuse:** step 3 `FeatureStates`, shared `feature_used` counter (FR-042, FR-063),
steps 4a, 4b, 4c, 5a, 6a, 7a, 8a; `d20` rerolls (Indomitable, Stroke of Luck).
**Tests:** unit for every feature (including limited uses, recharge on the right
`RestKind`), progression table rows for all 20 levels against the oracle,
application tests for each use case, class JSON decodes through the real decoder
(`data` tests), `ValidateSet` over the merged bundle, and the feature-use counter
asserted with `telemetrytest`. Rule interactions named per class spec (for
example Extra Attack with Two-Weapon rules in 6a, Action Surge uses regained on a
Short Rest).
**Observability:** each feature use increments `srd5e.character.feature_used` with
`srd5e.class` and `srd5e.feature` (bounded by the SRD's feature list); rejected uses
(no uses left, wrong rest) are results counted by `rejection`. Consumers monitor the
rejection mix per feature.
**Rollback:** revert the PR (code, JSON, oracle rows). Because JSON files merge into
one bundle, a class PR must not leave the merged set self-inconsistent (P12 in
Phase 2): the file and any shared id it adds land together.
**Stale tests:** 9e replaces the slice Wizard, so the tests pinned to it
(`content/data/data_wizard_test.go`, the thin-slice scenario, the pin test) are
updated in the same PR, not skipped. **Size:** a class spec has up to about 180
requirement ids; if a PR exceeds a reviewable size, split it into progression and
data, then features, then use cases (each still green).
**Acceptance:** the class's levels 1 to 20 match the oracle; the file decodes with
no problem; `data.Files()` lists it; the pin test is updated.

### 5.10 Steps 10a to 10c: spells named by the other class files

Spec `content-spells.md` (data side) and content-classes-data Table FA: bundle
exactly the spells that Bard, Cleric, Druid, Paladin, Ranger, Sorcerer and Warlock
name in `AlwaysPreparedSpells` facts, split in three PRs by spell level so each is
reviewable. They go in `content/data/spells.json`, keep the same shape as the 12
slice spells (the slice defines what a bundled spell must carry), and reuse the
Phase 2 spell types and `spell-effect-data` types; no new decoder row is needed.
P5 amends the FR-100 staging pin here.
**Tests:** `data` tests (every entry decodes, citations are first pages of their
sections), per-spell oracle rows against the printed lists (content-spells FR-023),
`Spell.classes` equals the printed class lists, `ValidateSet` still clean, and a
test that every spell id named in Table FA resolves. Requirements D: distributions
and counts for the full 339 (FR-101 stays skipped behind the pin).
**Observability:** data only. Failure mode: a bad entry. Operator signal:
`srd5e.content.decode_errors` by `problem_kind` and `BundledInvalidError` at
construction (existing signals); consumers should alert on any bundle decode
failure. Nothing new.
**Rollback:** revert the PR; the class PRs of step 11 depend on these.
**Acceptance:** no `unresolvedReference` for a Table FA spell id; **a person has
compared each long text with the PDF** (the open item Phase 2 never closed; record
it in the PR).

### 5.11 Steps 11a to 11g: spell-referencing classes

Order: Cleric, Druid, Bard, Sorcerer, Warlock, Paladin, Ranger. Same contents and
checks as 5.9, plus each class's spell-facing behaviour from `spellcasting.md`
(Sorcerer created slot, Warlock Pact Magic, Paladin Divine Smite as an
always-prepared spell, Druid `WildShapeState` and `KnownForms`).
**Tests / Observability / Rollback / Acceptance:** as 5.9.

### 5.12 Step 12: species-trait-behaviour

Six use cases, `SpeciesTraits`, `SpeciesTraitChoices`, `SpeciesTraitEffect`,
`SpeciesGrantedSpells`, species events, for the nine species of Table C. Four
species are bundled; the other five run on synthetic `Species` fixtures (P4) and
their data requirements stay D.
**Tests:** as 5.9, plus an unknown `SpeciesID` gives `SpeciesBehaviourUnknown` and
changes no state (FR-008), and every species' choice variant rejects the others.
**Observability:** use cases instrumented; the new `resolved` and `not_applicable`
values of `srd5e.character.result` are added to `observability.md` §5 and
`signals.md`. Monitor `rejection` and `not_applicable` shares.
**Rollback:** revert the PR.
**Acceptance:** every T id tested.

### 5.13 Step 13: closing the phase

- Scenarios in `scenarios/` (public API only, few): create a bundled Wizard, cast a
  spell, take a Short Rest, level up and choose a subclass; create a Fighter and
  use Second Wind and Action Surge; a multiclass slot case.
- `Example*` functions for the main use cases (`go test` verified, `dicetest`
  roller).
- README, CHANGELOG (check each step added its entry), `roadmap.md` (Phase 3 status,
  implementation status roll-up), `doc/documentation_index.md`.
- Audit: every spec of the header has an accurate `Status` and "Not yet implemented"
  line; `signals.md` equals the constants (O3, review-enforced); remove the
  class-data staging pins that no longer apply (content-classes-data FR-083, the
  spell pin stays).
- Run `just mutate` per changed file one last time and record the table (as Phase 2
  section 11).

**Tests:** scenarios and examples only; they supplement, never replace, unit tests.
**Observability:** the seven questions over the whole phase: a consolidated
"what to monitor" paragraph in `signals.md` for `character` (rejection rates,
missing-content rate, duration regressions).
**Rollback:** revert the PR; no code depends on it.
**Acceptance:** `just check` green; `GOARCH=386 go test ./content/... ` still green.

## 6. Verification commands

```sh
just check
GOARCH=386 go test ./content/...
go test -race -shuffle=on -coverprofile=coverage.txt ./character/... && go tool cover -func=coverage.txt | tail -n 1
just mutate ./character/<file>.go     # Docker only, capped at a quarter of the processors, one file at a time
```

## 7. Security review

`character` adds no network, storage, identity or secret. Its inputs are **values
the host passes in** (commands) and **content decoded earlier** by the Phase 2
decoder; bundled JSON is the only new untrusted-by-design data.

| Concern | Worst case | Mitigation |
|---|---|---|
| Hostile command values (levels, scores, ids, choices) | an invalid state reaches the aggregate, or a panic crashes the host | every constructor validates and returns a typed error (`srderr.ErrInvalidInput`); exported APIs never panic on input; fuzz each `ParseX`; tests walk the invalid-input list of testing.md §4 |
| Resource exhaustion | huge slices of choices, spells, items or pending choices make a use case slow | every collection in a command has a bound taken from the SRD (20 levels, 3 attuned items, spell list sizes); tests at bound and bound plus one; no recursion on input shape |
| Dice and randomness | a biased or failing roller yields bad rolls silently | only `dice.Roller`; failure wrapped once with `srderr.WrapPort`; contract suite; tests use `dicetest` |
| Port failure and panics | a content source error or panic escapes as a crash | use cases call `srderr.GuardValue` at the port boundary; recorded once (span, counter), not logged |
| State forgery | a host passes a `Character` that breaks an invariant | the zero value is invalid and documented; the aggregate root re-validates on every transition (it does not trust the passed state) |
| Telemetry disclosure | character names or ids in attributes | attributes are low-cardinality enums only (`srd5e.class`, `srd5e.feature`, outcome); never ids or free text (observability.md §5) |
| Bundled data provenance | non-SRD or trademarked text enters `content/data/` | AGENTS.md rule, `NOTICE.md` path list (A12), licensing denylist (A10); a person compares each new entry with the PDF (step 10 and 11 acceptance) |
| Authn, authz, injection, cross-tenant data | none | N/A: a library with no I/O; `os`, `net`, `reflect` denied in `character` (A3) |

## 8. Compatibility and rollout

Before v0.1.0 nothing is published, so every change is an addition and every step
reverts alone. After the first release these are **BREAKING**: a new variant on a
closed result or event set, a new `BonusFeatsAtLevel20` variant, a new
`FeatureStateKey` contract, renaming a signal or attribute, and the spec Q-9 edit of
level-advancement. Because later steps build on earlier ones, the order in section
5 is also the rollout order; no step changes a Phase 1 or Phase 2 public API unless
its spec says so (a step that must stops and goes to the user).

Rule-file edits (`architecture.md` rows, `observability.md` §5 values) land with the
code that needs them and refresh the rule tag; a new archtest check that fails an
existing file is fixed in the same PR, never skipped.

## 9. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Size: about 2,000 requirement ids across 35 PRs | one PR per spec; independent PRs inside a rank; stopping points after steps 3, 8 and 9 |
| T/D classification by judgement | re-checked per PR; moves recorded in the requirement-to-test list |
| Spec defects and follow-up edits found mid-step | fix the spec in the same PR; contradictions needing a decision stop the step and go to the user |
| Transcription error in class or spell data (the largest fidelity risk; the Phase 2 slice's PDF review is still open) | oracle fixtures (FR-093), pin tests, a named person's review recorded in the PR; the open Phase 2 review is not closed by this plan |
| Gomu is slow on a large package | per-file runs only, one at a time, capped; manual fallback documented |
| The class pattern (FR-023) turns out wrong after Fighter | Fighter is the only class PR until reviewed; the rest copy it |
| A4 conflicts with table-like data | constants, funcs and `switch` only (P10) |
| One package grows large | one file per concept; compile and test time watched in CI; splitting a context needs an architecture change and goes to the user |

## 10. Out-of-scope observations (recorded, not acted on)

- Roadmap Phase 3 says class data ships here, but `shared-types-and-ownership.md`
  FR-014 puts bulk data in Phase 5; the plan follows the roadmap (P1) and bounds the
  extra spell data (P2).
- FR-052 (the citation check across specs) and the rank check FR-056 are not
  mechanised; this plan relies on review for them.
- The `Requirement:` comment archtest check (`testing.md` §9) is still planned.

## 11. Mutation results (step 13, 2026-11-05)

Step 13 changed no domain or use-case file of `character`, `content` or `content/data`: it added tests
(`scenarios/phase_3_test.go`, `scenarios/rests_test.go`, `character/level_up_example_test.go`,
`TestClassFeatureTotals`), one test helper (`finishedRest`) and documents. A mutant needs a changed source
file, so no `just mutate` run was made and nothing below was re-measured.

The per-file scores that earlier steps measured with gomu v0.2.1 (capped Docker, one file at a time) are
recorded in the Implementation status of the spec that owns the file, where the step wrote them down:

| Where recorded | Files and scores |
|---|---|
| `specs/class-wizard.md` | `wizard_rules.go` 91.1, `wizard_spellbook.go` 93.2, `wizard_state.go` 87.5, `wizard_choices.go` 89.2, `wizard_features.go` 90.2, `wizard_rejection.go` 100, `wizard_spell_facts.go` 62.5 (3 survivors judged equivalent or killed by hand), `wizard_usecase.go` 100 |
| `specs/class-ranger.md` | `ranger_rules.go` 98.1, `ranger_choices.go` 87.4, `ranger_progression.go` 92.9, `ranger_state.go` 98.1, `ranger_level.go` 96.7, `ranger_reason.go` 97.6, `ranger_creation_choices.go` 85.7, `use_tireless.go` 81.6, `use_choose_ranger_option.go` 95.8, `use_resolve_ranger_features.go` 91.8 |
| `specs/content-classes.md` | the `content` class files (Phase 2): `class_ability.go` 85, `class_ids.go` 86.3, `class_table.go` 89.1, `class_spellcasting.go` 100, `decode_class_features.go` 83.7, `decode_class.go` 92.2, `class_definition.go` 98.0, and manual-fallback results for the files without a same-named test |

Not recorded in the tree: the scores of the other steps (the aggregate, derived stats, rests, spellcasting,
magic items, feats, creation, the other classes, species traits). Those steps reported them in their own
summaries and PR descriptions only, so this table cannot list them, and a final whole-phase re-run (about
300 files, minutes each under the CPU cap) was not made in this step. Anyone who needs the numbers runs
`just mutate ./character/<file>.go` for the file in question.

The maintainer has not accepted this gap. The Phase 3 mutation gate is therefore unrecorded for those files and stays an open item (the roadmap Status line does not claim it as met). Running about 450 files one at a time under the CPU cap was not done here.
