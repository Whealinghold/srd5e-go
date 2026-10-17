# Shared types and ownership

| | |
|---|---|
| Status | partially implemented |
| Context | `package` |
| Roadmap phase | 0 (binding on every later spec) |
| Depends on | none |

## 1. Purpose and scope

The specs were drafted in parallel and disagreed about who owns which type, which
context may depend on which, and how bundled data reaches consumers. This spec
settles those questions once. Where another spec conflicts with it, **this spec
wins** and the other spec changes.

Principles used to decide, in priority order:

1. **Decoupled and testable first.** Prefer the design in which each unit is
   testable with plain values and no collaborators (pure functions, state in and
   state out, small ports, doubles in one place).
2. **Dependency inversion.** A type lives in the most upstream context that
   needs it, never in a downstream one. Contexts depend on published language
   (value objects, ids, events), not on each other's use cases.
3. **Identity apart from behaviour.** A closed SRD vocabulary (`DamageType`)
   is data owned upstream. The behaviour that interprets it (`ConditionEffects`)
   is owned where the rules are applied.
4. **Make illegal states unrepresentable.** Structured composites and closed
   variant sets (interface with an unexported marker method) instead of flags,
   nil and "validate later".
5. **One owner per type.** Every other spec references it by file and id.
   Redefining a type is a defect.

**Out of scope:** any SRD rule. This is package design; it adds no game content.

## 2. Source

`n/a (package design)`. Derived from `doc/rules/architecture.md` §1–§6, `doc/rules/go.md`
and `doc/rules/observability.md` §4, and from the consistency review of the first 38
specs. The rule-file amendments in §5 are part of this change.

## 3. Vocabulary

| Term | Meaning here |
|---|---|
| Vocabulary type | A closed set of SRD names with static SRD facts only (typed enum or typed string id) |
| Behaviour owner | The spec whose domain code interprets a vocabulary type |
| Published language | The value objects, ids and events exported from a context package |
| Type phase | The roadmap phase in which a spec's *types* ship; bulk *data* may ship later |

## 4. Requirements

Each row is a decision. Ids are stable.

### 4.1 Ownership of shared types

| Id | Decision | Why | Path |
|---|---|---|---|
| FR-001 | The kernel packages own: `dice` (`Die`, `Expression`, `Roll`, `Result`, the `Roller` port, the `SeededRoller` adapter); `modifier` (`Modifier`, a plain signed integer value object with no kind tag; `RollMode` with `Normal`, `WithAdvantage`, `WithDisadvantage`; `RollCircumstance`, whose values include `Advantage` and `Disadvantage`); `ability` (`Ability`, the six abilities; `Score`; the derived modifier, `Score.Modifier() ability.Modifier`; `Scores`; the rejection error `InvalidError` with a `Reason` enum, single definition, in `ability-scores.md`); `srderr` (errors-and-results.md); `telemetry` (observability-ports.md). There is no kernel `DomainEvent` type: each context declares its own `Event` interface (FR-019). | Used by every context and by `d20`, which has no upstream. Kernel packages are small and import only each other as `architecture.md` §1 lists. | happy |
| FR-002 | `d20` owns: `Roll` (the d20 or d20s rolled, which die was kept, natural value) **including the reroll and replace-a-die operations**; `RollModifiers`; `DifficultyClass`; `Kind` (the D20 Test kind); `Outcome`; `AttackRollResult`; `ArmorClass`; `CheckResult`; `ForcedFailureResult`; `ForcedFailureReason`; `RerollDecision`; the pure domain functions `ResolveCheck`, `ResolveAttack` and `FailWithoutRoll` (FR-044); the `Resolver` use case (`Resolve(ctx, Command)`, which owns the roller and telemetry). All live in package `d20` and are written without the package stutter (`d20.Outcome`, not `d20.D20TestOutcome`). | Rerolls (Indomitable, Heroic Inspiration, similar features) act on a roll, so the roll type and its operations live together upstream of `character` and `combat`. | happy |
| FR-003 | `RollModifiers` is a **structured composite** (one ability modifier, an optional Proficiency Bonus, a list of other `Modifier`s), not a bare list. Any "at most one of X" rule the SRD states is enforced by the shape, so no runtime check is needed. | Illegal states unrepresentable. | happy |
| FR-004 | `d20.Outcome` is `Success` or `Failure` (a typed enum with `String()`) for every test kind. `AttackRollResult` adds `IsCriticalHit()` and exposes `Hit()` as `Outcome() == Success`. No parallel enums (`AttackOutcome`, `AttackConclusion`, `AttackRollOutcome`) exist. | One vocabulary; combat reuses d20's types. | happy |
| FR-005 | `content` owns every closed SRD vocabulary type and id: `DamageType`, `ConditionKind`, `Skill` (with its `Ability`), `CreatureSize`, `CreatureType`, `Language`, `RechargeTrigger`, `WeaponProperty`, `MasteryProperty`, and the typed ids `ClassID`, `SubclassID`, `SpeciesID`, `BackgroundID`, `FeatID`, `SpellID`, `ItemID`, `MonsterID`, `FeatureID` (`type ClassID string`, initialisms capitalised). All in package `content`. Their owning specs are the content specs. | `content` is upstream of `character` and `combat`; moving these there removes every content→combat and content→character dependency. | happy |
| FR-006 | Vocabulary types carry static SRD facts only (`Skill.Ability()`). Behaviour is owned elsewhere: `ConditionEffects` in `conditions.md`, damage handling in `hit-points-and-damage.md`, weapon mastery behaviour in `weapon-properties-and-mastery.md`. | Identity apart from behaviour. | happy |
| FR-007 | Content ids are lowercase `snake_case` matching `[a-z][a-z0-9_]*`, in JSON and in typed ids; `content.ParseClassID` and the like return `(ClassID, error)` for untrusted input (errors-and-results.md FR-027). | Matches the schema spec; one convention. | happy |
| FR-008 | `character` owns, in `derived-stats.md`: `Level`, `ClassLevels` (ordered; the first entry is the initial class), `ProficiencyBonus`, `HitPointMaximum`, `HitDicePool`, `HitPointGain` (package `character`, no stutter). The aggregate (`character-aggregate.md`) depends on `derived-stats.md`, never the reverse. **`LevelHistory` does not exist**: the aggregate stores state and emits events; hosts keep history if they want it. | Breaks the mutual deferral; state-in/state-out stays small. | happy |
| FR-009 | `rests-and-hit-dice.md` owns `RestKind` and rest effects. `content` owns only `RechargeTrigger` (data); `character` maps `RestKind` to triggers. | `content` must not depend on `character`. | happy |
| FR-010 | `character-aggregate.md` owns `HeroicInspiration` as a typed enum (`Inspired`, `NotInspired`); `d20` reroll operations take the decision as input. | No boolean standing in for a named state. | happy |
| FR-011 | `spellcasting.md` owns the `Concentration` state shape (held in `SpellcastingState`); `casting-and-concentration.md` owns the Concentration rules and the save-DC rule, which takes a plain `int` damage amount, never a `combat` event. `combat-action-resolution.md` reads the character's concentration state and calls it. | `character` may not depend on `combat`. | happy |
| FR-012 | `combat-foundations.md` (new) owns `CombatantID`, the combatant reference types, and the in-context naming conventions. See FR-021. | The bottom of the combat dependency order. | happy |

### 4.2 Dependency rules

| Id | Decision | Why | Path |
|---|---|---|---|
| FR-013 | The context map in `architecture.md` §2 is binding on specs, not only on code. A spec may not list, import or assume a type from a downstream context. If it needs one, the type moves upstream under FR-001–FR-012. | The first review found `character`→`combat` and `content`→`combat` edges. | unhappy (forbidden edge) |
| FR-014 | A spec's **types** ship in its context's phase; bulk **data** may ship later. A spec may depend only on specs whose types ship in the same or an earlier phase. Phases: shared and `d20` types 0–1; `content` types and vocabulary 2; `character` 3; `combat` 4; bulk data and release 5. | Fixes the phase inversions without reshuffling content volume. | happy |
| FR-015 | Inside `combat`, specs form a strict order: `combat-foundations` ← `hit-points-and-damage` ← `conditions` ← `attacks` ← `initiative-and-actions` ← `encounter-aggregate` ← `combat-action-resolution`. A spec may depend only on specs to its left. | Removes the `conditions`/`initiative`/`encounter` cycle. | unhappy (cycle) |
| FR-016 | Inside `character`, order: `derived-stats` ← `character-aggregate` ← `rests-and-hit-dice`, `spellcasting`, `feats-behaviour`, `weapon-properties-and-mastery`, `level-advancement-and-multiclassing`, `character-creation` ← `class-*`. | Same reason. | unhappy (cycle) |

### 4.3 Results, events and naming

| Id | Decision | Why | Path |
|---|---|---|---|
| FR-017 | Every use case returns its own closed result named for the operation (`<Op>Result`): a typed enum for a single named state, a result struct for one shape of data, or an interface with an unexported marker method plus one struct type per variant for different data per variant (documented, switched exhaustively; errors-and-results.md FR-018). Variants are prefixed with the operation: `DamageApplied`, `DamageRejected`, `CreateCharacterContentMissing`; a prefix that only repeats the package name is dropped. Generic names (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`, `Found`, `NotFound`) are never exported from a context package. They are returned values, never `error`s. | Removes name collisions inside a context package. | unhappy (name collision) |
| FR-018 | Shared failure detail is a **value object held as a field**, not a shared variant: `MissingContent(kind, id)` owned by `content`. Variants of different results each carry one. | A closed variant set cannot span packages (its marker method is unexported); fields can. | happy |
| FR-019 | Each context declares its own `Event` interface (`character.Event`, `combat.Event`) with an unexported marker method and no kernel base type; a host that wants one stream defines its own consumer-side interface. Each spec's own event family is a closed variant set (its own sub-interface or documented list), so a host switches exhaustively per family. Events are past-tense value objects describing something that happened (never `...Requested`, `...Required` or `...Needed`, and `...Due` only as `...BecameDue`, `combat-foundations.md` FR-020; result values such as `ContainmentDamageDue` are not events). A named event has exactly one defining spec; others reference it. | One marker per context keeps the set closed to outsiders while each spec adds its own families. | happy |
| FR-020 | `SubclassChosen`, `SubclassRejected` live in `level-advancement-and-multiclassing.md`; `FightingStyleChoice` in `feats-behaviour.md`; `WeaponMasteryChoice` in `weapon-properties-and-mastery.md`; `RestApplied` in `rests-and-hit-dice.md`; `ClassDataInvalid` is replaced by the decoder's failure type. Class specs reference these and define none of them. | Class specs duplicated them. | unhappy (duplicate) |
| FR-021 | Command and event names are unique across the package. Where two specs named the same thing (`ApplyDamageCommand`, `Damage`/`DamageInstance`), `hit-points-and-damage.md` keeps the plain name and `encounter-aggregate.md` uses the `...ToCombatant` form. | Collision list from the review. | unhappy (name collision) |

### 4.4 Class features and state

| Id | Decision | Why | Path |
|---|---|---|---|
| FR-022 | The character aggregate holds `FeatureStates`: an immutable typed-key store. A class spec defines its `FeatureState` implementations and typed `FeatureStateKey`s (a small interface plus keys comparable by value, no `any` in the public signature). The aggregate defines the slot and the read/write API only. | Open for extension without a closed variant set spanning files; testable by constructing states directly. | happy |
| FR-023 | Class feature logic is a **pure domain function** from (character state, inputs) to (new state, events). A feature needing dice or content gets one use case (in a file named for it, `go.md` §1) that supplies them through ports. No class has a different pattern. | Uniform, testable. | happy |
| FR-024 | Class creation choices travel in `CreateCharacter` as a typed `ClassCreationChoices` carried by the class spec that defines them; the aggregate stores no class-specific field. | No hook proliferation. | happy |
| FR-025 | `ClassDefinition`, `SubclassDefinition`, `ProgressionRow`, `FeatureEntry` and bulk class and subclass data (all 12 classes, including the Evoker subclass) are owned by a new spec, `content-classes.md`. `class-*.md` specs own behaviour only. Primary-ability requirements are `AbilityRequirement`, a closed variant set `AllOf` / `AnyOf`. | Nine specs referenced a class content spec that did not exist; Fighter's "or" needs `AnyOf`. | happy |

### 4.5 Combat wiring

| Id | Decision | Why | Path |
|---|---|---|---|
| FR-026 | `conditions.md` publishes a pure function `ConditionEffects` returning `RollContributions` (roll-mode sources, automatic failure, `AutomaticCritical`). `attacks.md` is pure: it takes `RollModifiers`, roll-mode sources and an `AutomaticCritical` input already computed. `combat-action-resolution.md` is the only place that calls both. | Attacks testable with plain values; no cycle. | happy |
| FR-027 | `attacks.md` defines no `Proficiency` type; the Proficiency Bonus arrives inside `RollModifiers` (FR-003). | Duplicated `ProficiencyBonus`. | unhappy (duplicate) |
| FR-028 | Cover (AC and Dexterity-save bonus) is owned by `attacks.md`. `derived-stats.md` does not compute it. | One owner. | happy |
| FR-029 | `grappling.md` owns grapple **mechanics** (the grapple option, escape, movement while grappling, ending the grapple, the `GrappleEnded` event). `conditions.md` owns only the Grappled **condition entry** (its effects row) and references `grappling.md`. `attacks.md` owns only the Unarmed Strike attack and references `grappling.md` for the grapple option. `encounter-aggregate.md` handles `GrappleEnded`. See FR-048 for the general rule. | Three-way split resolved; the topic-specific spec wins. | happy |

### 4.6 Content delivery

| Id | Decision | Why | Path |
|---|---|---|---|
| FR-030 | The JSON files under `content/data/` are the only copy and the source of truth. They are embedded **directly** with `go:embed` (JSON files themselves, no generator step, no generated Go). No other data directory exists. | `os` is forbidden in the core; `go:embed` reads at build time and stays platform-neutral. | happy |
| FR-031 | There is **one decode path**: embedded strings go through the same `ContentSetDecoder` as any host-supplied JSON. `BundledContentSource` is exported from `content/data` (package `data`, the only package allowed to import `embed`) and implements the `content` source port. | Bundled data exercises the real decoder; no second loader to test. | happy |
| FR-032 | The embedded `content/data/` JSON is the source of truth, so no drift check is needed; instead a test decodes every embedded file through the real decoder. Extraction tooling, if any, lives outside the module root packages and is never imported. | Source of truth stays authoritative. | unhappy (drift) |
| FR-033 | `NOTICE.md` covers exactly one path: `content/data/` (matching AGENTS.md and architecture.md A12). SRD text in tests lives only under a `testdata/srd/` directory, which is test input that carries the attribution pointer (content-licensing-and-notice.md FR-027) but is not a `NOTICE.md` covered path. `doc/research/` notes that hold SRD text are Markdown only, carry the attribution, and are not Go packages. | One covered-path list; CC-BY changes are indicated. | happy |
| FR-034 | There is **one** `schemaVersion` integer for the whole content format. The decoder accepts exactly the current version. `content-schema-and-decoder.md` owns a revision table. Other content specs request additions as rows there; none claims a version number. Before 0.1.0 all revisions land as one version. | Stops v2 collisions. | unhappy (version mismatch) |
| FR-035 | `content-schema-and-decoder.md` owns the envelope, ids, decoder and failure type. Each kind's shape is owned by its content spec (`Spell` by `content-spells.md`, and so on) and is registered in the revision table. | Stops "supersedes" disputes. | happy |

### 4.7 Ports, doubles and adapters

| Id | Decision | Why | Path |
|---|---|---|---|
| FR-036 | No-op implementations (`telemetry.NoopLogger`, `NoopTracer`, `NoopMeter`, and the bundle `Noop`) are zero-value-usable types in package `telemetry` itself. They are not adapters. Ports, no-ops and the shared vocabulary live in the port's own package; doubles live in `<pkg>test`. | A use case needs the default and must not import an adapter. | happy |
| FR-037 | Generic adapters live in the package of the port they implement (`dice.SeededRoller`, which takes a caller-supplied `*rand.Rand` and never creates one) or, when they need a non-kernel import, in a sub-package (`telemetry/slogtelemetry`, `dice/randroller`, stdlib only). An adapter imports only its port package and the kernel. No facade or barrel exists. | One consistent rule. | happy |
| FR-038 | All test doubles live in the `<pkg>test` package of the port they double (`dice/dicetest`, `telemetry/telemetrytest`, `content/contenttest` with `FakeSource`). Only `_test.go` files import them. Doubles import only their port package and the kernel. A fake content source is a double, not an adapter. | One place; layering clear. | happy |
| FR-039 | Every port ships its contract suite as an exported function in its `*test` package (`dicetest.RunRollerContract(t, factory)`), so consumer adapters reuse it. Importing `testing` there is fine because only test files import `*test` packages. | Dependency rule. | happy |

### 4.8 Telemetry conventions shared by all specs

| Id | Decision | Why | Path |
|---|---|---|---|
| FR-040 | `ResolveD20Test` has one authority for signals: `resolve-d20-test-use-case.md`. `d20-tests.md` is domain-only and silent. | Two authorities disagreed. | unhappy (duplicate) |
| FR-041 | Attribute values: `srd5e.outcome` = `success` or `failure`; `srd5e.d20_test.kind` = `ability_check`, `saving_throw` or `attack_roll`; `srd5e.d20_test.roll_mode` = `normal`, `advantage` or `disadvantage`; `srd5e.d20_test.critical_hit` = `true` or `false` (attack rolls only); `error.type` = one value per port (`<port>_error`: `dice_roller_error`, `content_error`), defined in `errors-and-results.md` section 7 and produced by `srderr.ErrorType`. | One set of values across metrics, spans and logs. | happy |
| FR-042 | Class features share one counter, `srd5e.character.feature_used`, with attributes `srd5e.class` and `srd5e.feature`. No per-feature metric names. | Bounded signal count. | happy |
| FR-043 | The d20 test emits no log (spans and metrics already cover it). | `observability.md` §7. | happy |

### 4.9 Further decisions (second consistency pass)

| Id | Decision | Why | Path |
|---|---|---|---|
| FR-044 | **A use case never calls another use case.** It composes pure domain services that upstream contexts publish in their package API (`d20.ResolveCheck`, `ResolveAttack`, `FailWithoutRoll`; `d20.Resolver` is itself a use case and is never called by another), supplying dice and content through its own ports. Each use case records only its own signals, taking attribute values from the domain result. `attacks` therefore does not build a nested `ResolveD20Test` and does not use a silenced `Observability`. | No hidden nested telemetry, no double counting, and the domain service is testable with plain values. | unhappy (nested telemetry) |
| FR-045 | A new bounded context **`toolbox`** holds the Gameplay Toolbox rules (SRD pp. 192-203). It is downstream of `content`, `d20`, `character` and `combat` and no context depends on it, so the map stays acyclic. Its type phase is 5. It hands results back to the host (an Exhaustion level to apply, a condition to apply); it does not mutate character or combat state. Named catalogue entries (poisons, traps, contagions) are content kinds registered in the schema revision table; small rules tables are constants in the toolbox domain. Optional Game Master riders are explicit `ToolboxRuleOptions` fields in commands, never global flags. "Creating a Background" is a validation rule in `content-species-backgrounds-feats.md`, not a toolbox spec. | Nearly every modellable rule is "save, then damage / condition / Exhaustion", which needs `combat` types that `character` cannot import. | happy |
| FR-046 | The owning spec for the FR-005 vocabulary is `content-schema-and-decoder.md`: `DamageType`, `ConditionKind`, `Skill`, `CreatureSize`, `CreatureType`, `Language`, `RechargeTrigger`, `WeaponProperty`, `MasteryProperty` and every typed id including `MonsterID`. Bulk content specs reference them and define none. | `content-monsters.md` and `content-equipment.md` each re-defined some. | unhappy (duplicate) |
| FR-047 | Content **data** specs (`content-thin-slice`, `content-species-backgrounds-feats`, `content-equipment`, `content-spells`, `content-monsters`, `content-classes`, `content-magic-items`) depend only on `content-schema-and-decoder` and `content-source-port`, never on each other. The `toolbox-*` catalogue specs are behaviour specs (FR-045, FR-056), not bound by this rule; their entries are only data kinds in the schema revision table. Cross-references between data are by id, checked by the decoder's whole-set cross-reference validation, which `content-schema-and-decoder.md` owns. `BundledContentSource` is owned by `content-source-port.md`; each data spec adds its embedded library, and `content-thin-slice.md` supplies only the first entries. | Removes the data-spec cycle; keeps content testable one kind at a time. | unhappy (cycle) |
| FR-048 | **General ownership rule:** when two specs cover one topic, the topic-specific spec owns the mechanics and the broader spec keeps only its entry and a reference. (`grappling.md` over `conditions.md`; `heroic-inspiration.md` owns the mechanics while `character-aggregate.md` owns the state slot; `attack-and-help-actions.md` over `initiative-and-actions.md` for those actions; `magical-effects.md` and `spell-effects.md` over `spellcasting.md` and `content-spells.md` for effect resolution.) The broader spec's duplicate requirements become "Withdrawn: moved to X". | Prevents the parallel-drafting overlaps. | unhappy (overlap) |
| FR-049 | `IsBloodied(current, HitPointMaximum)` is a pure function in `derived-stats.md`. `hit-points-and-damage.md` and the class specs reference it and define no other Bloodied predicate. | `character` cannot import the `combat` definition. | happy |
| FR-050 | The mechanical check that `NOTICE.md`'s covered paths equal the FR-033 list is owned by `architecture-guardrails.md` FR-079. `content-licensing-and-notice.md` defines the list and does not withdraw the check. | A delegation loop left no owner. | unhappy (no owner) |
| FR-051 | The rule amendments that specs mark "blocked" or "outstanding" are accepted as written, provided they weaken no MUST: `errors-and-results.md` OQ-5, OQ-9, OQ-11 and FR-043 (`PortError` is one concrete type; `recover` only in `srderr.Guard`; a returned failure result carries no `error.type`), `observability-ports.md` FR-059 (span-ending via `defer`, closure-free attributes, zero-value no-op exemption) and `combat-foundations.md` FR-032 / D-9. They are applied to the rule files in the same change, with fresh tags. | The specs depended on rule text the rules lacked. | happy |
| FR-052 | Cross-references are checked mechanically: every `<spec>.md FR-nnn` citation must name an existing spec and an FR id that exists and is not withdrawn. The check lives in `internal/archtest` (spec set) and is run before any spec change is accepted. | Dangling citations were the commonest drift. | unhappy (dangling reference) |

### 4.10 Third-pass decisions (ownership gaps, header meaning, size)

| Id | Decision | Why | Path |
|---|---|---|---|
| FR-053 | `content-schema-and-decoder.md` also owns these vocabulary types: `HitPointDie`, `LightLevel`, `DarknessKind`, `Obscurement`, `SpecialSense`, `Alignment`, `SourceCitation`, and the Creature Size and Space table as static facts of `CreatureSize` (SRD p. 14, p. 188). The specs that used them (`hiding-searching-and-senses`, `content-monsters`, `content-spells`, `exploration`, `magic-item-rules`, `derived-stats`, `rests-and-hit-dice`) reference them and define none. | These had no owner, a ghost owner, or two owners. | unhappy (duplicate) |
| FR-054 | Cross-reference validation is **whole-set**: after decoding, the decoder checks every id reference across the full content set. A reference site never opts out. The thin slice must be self-consistent (it may reference only ids it contains). Specs that said "sameDocument" or "the decoder must not check" change. | One rule; the first slice fails fast. | unhappy (dangling id) |
| FR-055 | `PortError`, the guards and the shared sentinels live in the kernel package `srderr`. `dice-and-rolls.md` and any spec saying `shared_kernel/errors/` or `Srd5eException` is corrected. Kernel constructors return typed errors wrapping `srderr.ErrInvalidInput`; they never panic outside `MustX`. | The kernel depends on nothing in the package. | unhappy (kernel dependency) |
| FR-056 | **"Depends on" means one thing:** the specs that must be implemented before this one can be, listed only as `` `name.md` `` tokens separated by commas, no prose in the cell. Consumers are never listed. A prose mention is not a dependency. Every dependency listed must have a **strictly lower rank** in section 4.11; `shared-types-and-ownership.md` (rank 0) is always allowed and need not be listed. Specs of rank 90 or higher may depend on any spec. | An acyclic graph by construction; checkable by script. | unhappy (cycle) |
| FR-057 | `rule-options.md` is a package-level spec defining only the option **pattern** (immutable value object, in the command, never global) and the shared rules. Each context's options are owned by its own spec: `CharacterRuleOptions` by `character-aggregate.md`, `CombatRuleOptions` by `combat-foundations.md`, `ToolboxRuleOptions` by `toolbox-hazard-effects.md`. | Removes the character-to-toolbox edge and phase inversion. | unhappy (forbidden edge) |
| FR-058 | `magic-item-rules.md` is a `character` spec (behaviour and state). The closed vocabulary and static tables it needs (rarity, category, attunement facts, Magic Item Resilience facts) move to `content-magic-items.md` as content types. Magic Item Resilience facts are recorded in `magic-item-rules.md`; `environment-hazards-and-objects.md` (Objects) references them. | Single context per spec; no content→character edge. | unhappy (mixed context) |
| FR-059 | `spell-effects.md` is split. The new `spell-effect-data.md` is a `content` spec (effect data model, facts, decoder shape, bundling, rank 7). `spell-effects.md` stays `combat` (resolver, use case, telemetry). Requirement ids stay with their current file where they are; `spell-effects.md` Q3's FR-to-part assignment says which ids move, and moved ids are marked withdrawn with the new owner id. | Mixed-context spec; its own Q3 asked for the split. | unhappy (mixed context) |
| FR-060 | New spec `monster-limited-use.md` (`combat`): Recharge X–Y, X/Day, Legendary Action timing and Lair actions (SRD pp. 254-257), as pure state-in/state-out rules over the monster stat block. Other unowned items: Armor Training Disadvantage (SRD pp. 92, 177) is owned by `equipment-rules.md` (rule) and applied in `derived-stats.md` (roll circumstances); the Creature Size and Space table is a static fact of `CreatureSize` (FR-053); "Rhythm of Play" and "Exceptions Supersede General Rules" (p. 5) are recorded in `rules-glossary-coverage.md` as narrative guidance with no requirement. | Closes the SRD ownership gaps. | happy |
| FR-061 | `architecture-guardrails.md` depends on every spec it verifies (rank 90). Other specs reference it in prose only. `observability-ports.md` therefore does not list it in "Depends on". | Removes the mutual dependency. | unhappy (cycle) |
| FR-062 | **Size discipline.** A spec stays under 600 lines. Withdrawn requirements are collapsed into one compact "Withdrawn ids" table at the end of section 4 (`| FR-nnn | Withdrawn: moved to <spec>.md (<id>) |`, one line each) and their test-plan rows are deleted. The test plan groups consecutive FR ids that share a level and a test file into one row. Section 9 lists only open questions; settled ones become one line citing the decision. A spec does not restate another spec's requirement; it cites it. Live FR ids are never changed or dropped. | The first reconciliation passes grew the specs to 63,000 lines. | unhappy (bloat) |
| FR-063 | Class specs do not define their own feature-use signals. They use the shared `srd5e.character.feature_used` counter and the shared `.duration` and `.errors` signals owned by `character-aggregate.md`, with attributes `srd5e.class` and `srd5e.feature`. The owner of emitting them is the use case (FR-023). `StrokeOfLuckUsed` and every character event implement `character.Event` and carry `CharacterID`. | Nine class specs each re-declared signals. | unhappy (duplicate) |
| FR-064 | `content-schema-and-decoder.md` owns the content identity types `ID` (content id; `content.ID`), `IDProblem`, `Key`, `Kind`, `Entry` and `MissingContent` (all package `content`, no stutter). `content-source-port.md` keeps only the `Source` port (`content.Source`), the content source port's `Port`, failure sentinels and kind mapping (failures are `*srderr.PortError`), and depends on the schema. | The decoder used identity types from the port while the port listed the decoder: a real cycle. | unhappy (cycle) |
| FR-065 | `DurationUnit` is a vocabulary type owned by `content-schema-and-decoder.md` (FR-149). `content-spells.md` and the toolbox specs use it and define none. | `toolbox-poison` and `toolbox-traps` imported it from a data spec. | unhappy (duplicate) |
| FR-066 | A class spell list is derived in `content-spells.md` from `Spell.classes`. `content-classes.md` defines no `ClassSpellListMember` interface and asks nothing of `content-spells.md`. | Avoids content-spells importing content-classes. | unhappy (cycle) |
| FR-067 | FR-047 is amended: a **bundle** spec at rank 7 (`content-classes-data`) may depend on the rank 6 data specs it validates against. A rank 6 data spec never depends on another rank 6 spec; the check that species data needs spell data is part of whole-set validation (FR-054), not a spec dependency. | `content-species-backgrounds-feats` and the class data needed cross-checks. | unhappy (cycle) |
| FR-068 | **Go package names.** The Dart contexts and kernel map to packages: `shared_kernel` to `dice`, `ability`, `modifier`; error vocabulary to `srderr`; observability ports to `telemetry`; `d20_test` to `d20`; `content`, `character`, `combat`, `toolbox` keep their names. Spec text that says `shared_kernel`, `shared_ports`, `lib/src/...` or a barrel is read as the Go package that owns the concept. Types are named without the package stutter (`dice.Expression`, `d20.Outcome`). | One naming rule for the whole spec set. | happy |
| FR-069 | **Where ports, no-ops, doubles and contract suites live (architecture.md §4).** A port is defined by its consumer; a port shared by several packages sits next to the type defining its vocabulary (`dice.Roller`). A context declares its own small consumer-side interface for what it needs and is not coupled to implementations. No-ops (`telemetry.Noop*`) are zero-value-usable and live in the port package. Test doubles and the exported contract suite live in `<pkg>test` (`dice/dicetest`, `telemetry/telemetrytest`); only `_test.go` files import them. Adapters carry `var _ Port = (*Adapter)(nil)`. | Replaceable modules; doubles never mixed into production packages. | happy |
| FR-070 | **Results, errors, panics.** Expected rule outcomes are returned values (typed enum, result struct, or a closed variant set: interface with an unexported marker method, one struct type per variant, documented, switched exhaustively). Untrusted-input failures are `(T, error)` from `NewX` / `ParseX` with typed errors or sentinels, decoders joining all problems with `errors.Join`. Port failures are returned `error`s wrapped once by the caller with `srderr.WrapPort`. Panics only in `MustX` and unreachable branches. Specs that say "sealed result", `tryCreate`, `Srd5eException` or `ArgumentError` use these forms. | Matches errors-and-results.md. | happy |
| FR-071 | **Context.** `context.Context` is the first parameter of anything that may do I/O, block or emit spans (content sources, use-case entry points that call telemetry). Pure domain functions and `dice.Roller` take none. | Matches architecture.md §4. | happy |
| FR-072 | **Immutability and zero values.** Value objects have unexported fields, validating constructors, value receivers and no setters; each type documents whether its zero value is useful or invalid. No package-level mutable state, `init()` side effects, or direct clock/randomness. | Matches go.md §5. | happy |
| FR-073 | **Go notation.** A lowerCamelCase name in a spec (field, method, function, enum member) stands for the exported PascalCase Go identifier (`rowAt` is `RowAt`, `notABarbarian` is `NotABarbarian`). Enum members are typed constants carrying the type name as prefix where go.md §1 requires it, and `String()` returns the short name. Specs need not repeat this note. | The s-z specs read as unexported or Dart names without it. | happy |

### 4.11 Spec rank table

A spec's "Depends on" may list only specs of strictly lower rank (FR-056).

| Rank | Specs |
|---|---|
| 0 | `shared-types-and-ownership` |
| 1 | `errors-and-results` |
| 2 | `dice-and-rolls`, `modifiers-and-advantage`, `ability-scores` |
| 3 | `observability-ports`, `rule-options` |
| 4 | `d20-tests`, `content-schema-and-decoder` |
| 5 | `resolve-d20-test-use-case`, `content-source-port`, `content-licensing-and-notice` |
| 6 | `content-thin-slice`, `content-classes`, `content-species-backgrounds-feats`, `content-equipment`, `content-spells`, `content-monsters`, `content-magic-items` |
| 7 | `spell-effect-data`, `content-classes-data` |
| 9 | `equipment-rules` |
| 10 | `derived-stats` |
| 11 | `character-aggregate` |
| 12 | `rests-and-hit-dice`, `heroic-inspiration`, `weapon-properties-and-mastery`, `social-interaction`, `exploration` |
| 13 | `spellcasting`, `magic-item-rules` |
| 14 | `level-advancement-and-multiclassing`, `casting-and-concentration`, `magic-item-effects`, `tool-proficiency-and-crafting` |
| 15 | `feats-behaviour`, `higher-level-start-equipment-and-trinkets`, `spell-scroll-use` |
| 16 | `character-creation`, `wish-spell` |
| 17 | `class-barbarian`, `class-bard`, `class-cleric`, `class-druid`, `class-fighter`, `class-monk`, `class-paladin`, `class-ranger`, `class-rogue`, `class-sorcerer`, `class-warlock`, `class-wizard`, `species-trait-behaviour` |
| 18 | `class-wizard-evoker` |
| 20 | `combat-foundations` |
| 21 | `hit-points-and-damage` |
| 22 | `conditions` |
| 23 | `grappling`, `hiding-searching-and-senses`, `environment-hazards-and-objects`, `telepathy` |
| 24 | `attacks`, `attack-and-help-actions`, `mounted-and-underwater-combat`, `deployed-and-restraining-gear`, `monster-engulf-and-swallow` |
| 25 | `initiative-and-actions`, `item-use-effects` |
| 26 | `monster-limited-use`, `magical-effects`, `movement-lifting-and-flight`, `grid-positions-and-movement`, `readying-a-spell`, `monster-combatant-setup` |
| 27 | `spell-effects`, `returning-to-life`, `monster-stat-block-actions`, `monster-traits`, `monster-spellcasting` |
| 28 | `encounter-aggregate`, `active-effects`, `spell-granted-modifiers`, `persistent-spell-zones`, `dispel-and-counterspell`, `shape-changing-spells`, `revival-and-restoration-spells`, `spell-outcome-tables` |
| 29 | `combat-action-resolution` |
| 30 | `combatant-roster-changes` |
| 31 | `summoned-and-companion-creatures` |
| 40 | `toolbox-hazard-effects` |
| 41 | `toolbox-travel-pace`, `toolbox-environmental-effects`, `toolbox-poison`, `toolbox-traps`, `toolbox-curses-and-contagions`, `toolbox-fear-and-mental-stress`, `toolbox-combat-encounter-difficulty` |
| 90 | `architecture-guardrails` |
| 91 | `rules-glossary-coverage` |
| 99 | `release-0-1-0` |

## 5. Model

Rule-file rewrites made together with this spec (read-receipt tags replaced):

- `architecture.md` is rewritten for Go: flat public packages and an import
  allow-list (§1), the context map as a table of allowed package imports (§2),
  closed results, ports and concurrency (§4), the failure model (§6),
  versioning (§7), and the architecture test checks A1 to A13 (§9). FR-005,
  FR-017, FR-036 to FR-039 are realised there.
- `go.md` is new: naming, doc comments, errors, context, zero values, generics,
  interfaces, compatibility.
- `observability.md` §4: no-op defaults are zero-value-usable and live with the
  port; the attribute value sets in FR-041 are the shared convention.
- `testing.md` §7: contract suites are exported from the `*test` package
  (FR-039).

`AGENTS.md` and `doc/roadmap.md` are updated to match (class data in Phase 3,
type-phase rule FR-014, specs list).

## 6. Interface

None of its own. Consumers of this spec are the other specs, which cite it as
`shared-types-and-ownership.md FR-nnn`.

## 7. Observability

Silent by design: this spec defines conventions, not code. FR-041 and FR-042 fix
attribute values and names so every spec's signal table agrees. Failure mode of
the conventions: two specs defining the same signal differently; the guardrail
check on `doc/observability/signals.md` consistency catches it.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-013, FR-038, FR-068, FR-069 | architecture | Checks A1, A2, A11 in `internal/archtest` (import allow-list, `*test` isolation, cycles) |
| FR-017, FR-021, FR-070 | architecture | Check A7: undocumented variant sets and generic exported variant names fail |
| FR-030–FR-033 | unit + architecture | Embedded `content/data/` decodes through the real decoder; `NOTICE.md` path list (A12); no non-`.json` data files |
| FR-034 | unit | Decoder rejects any other `schemaVersion` |
| FR-015, FR-016, FR-052, FR-056 | architecture | Spec order, citation and rank checks in `internal/archtest` (spec set); not yet implemented |
| FR-036, FR-037 | architecture | Package-location and import rules (A1, A2) |
| FR-070, FR-071, FR-072 | architecture | Checks A4 to A6 (no hidden effects, panic discipline, port use confined) |
| FR-001–FR-012, FR-014, FR-018–FR-029, FR-035, FR-039–FR-067, FR-073 | doc review | Verified by the spec-consistency critic; each row is checked against the FR table in §4 |

**Implementation status.**

- Done: the kernel ownership it decides (`dice`, `modifier`, `ability`, `srderr`, `telemetry`, `d20`) exists in code. `internal/archtest` implements checks A1 to A13 (`architecture.md` §9), which enforce FR-013, FR-017, FR-021, FR-036 to FR-038, FR-068 to FR-072 and the embedded-data rules of FR-030 to FR-033 (A12 is vacuous until `content/data/` exists).
- Not done: FR-015 and FR-016 (intra-context spec order) and FR-056 (rank rule) are not mechanised; FR-052 (cross-reference check in `internal/archtest`) is not implemented; FR-034 needs the decoder; the `content`, `character`, `combat` and `toolbox` contexts are not built. Rows verified by doc review stay judged in review.

Not yet implemented: FR-015, FR-016, FR-034, FR-052, FR-056, and the content, character, combat and toolbox contexts.

## 9. Open questions and decisions

All questions raised by the consistency review are settled above. None open.

## 10. Changelog impact

No separate migration: the code exists. Each code change records the decisions it implements.
