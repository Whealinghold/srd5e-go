# Rule options

| | |
|---|---|
| Status | draft |
| Context | `package` |
| Roadmap phase | 0 |
| Depends on | `errors-and-results.md` |

## 1. Purpose and scope

Defines the **pattern** for the immutable option value objects that let a
consumer choose among the alternatives that SRD 5.2.1 itself presents to the
Game Master or the table, as `doc/rules/architecture.md` §3 ("Optional rules
and extension points") requires, and the rules every such object shares.
Options are plain values carried in the state passed to a command. They are
never global settings.

This spec defines no option type and has no phase of its own. Each context's
options are owned by that context's spec and ship in its phase
(`shared-types-and-ownership.md` FR-014, FR-045, FR-057):

| Options object | Owner spec | Context and type phase |
|---|---|---|
| `CharacterRuleOptions` | `character-aggregate.md` (section 4.10) | `character`, 3 |
| `CombatRuleOptions` | `combat-foundations.md` (section 4.6) | `combat`, 4 |
| `ToolboxRuleOptions` | `toolbox-hazard-effects.md` (section 4.9) | `toolbox`, 5 |

Finding from reading the whole PDF: SRD 5.2.1 contains no section titled
"optional rules" or "variant rules". It does, however, offer alternatives to
the table in two ways. The core rules say where the GM or the table may choose
a different approach to a rule that applies to the whole table (the first four
options below), and the Gameplay Toolbox's "Fear and Mental Stress" (p. 196) is
an explicitly optional Game Master toolkit: "If you plan to use any of these
rules, discuss them with your players at the start of the campaign", followed
by the Frightened riders ("Consider these examples") and half damage on a
successful Mental Stress save ("at your discretion"). Each option is cited.
Anything weaker (a general "the GM decides" remark, a player's own choice
during play, a per-creature or per-attack choice) is not an option; see Out of
scope, which lists the alternatives found and the spec that owns each.

The options, each verified in the PDF:

| Option | Where the SRD offers it | Options object | Consumer (spec and FR that reads it) |
|---|---|---|---|
| Ability score generation methods the table permits | "Generate Your Scores", p. 21 | `CharacterRuleOptions` | `character-creation.md` FR-054 |
| Starting level | "Starting at Higher Levels", p. 24 | `CharacterRuleOptions` | `character-creation.md` FR-071 |
| Bonus Feats at Level 20 | "Bonus Feats at Level 20", p. 24 | `CharacterRuleOptions` | `feats-behaviour.md` (applies feats). The consuming FR is not yet named. Deferred, FR-029. |
| Playing on a Grid | "Playing on a Grid", p. 13 | `CombatRuleOptions` | `grid-positions-and-movement.md` FR-001 (reads `CombatRuleOptions.Grid`; `initiative-and-actions.md` stays in feet). |
| Fear Effects riders (Dash each turn, Advantage against the creature, one kind of action per turn) and half damage on a successful Mental Stress save | "Fear Effects", "Mental Stress Effects", p. 196 | `ToolboxRuleOptions` | `toolbox-fear-and-mental-stress.md` FR-027, FR-028 (riders), FR-040, FR-043, FR-051 (Mental Stress), FR-029 (independence), FR-100 (both commands carry one `ToolboxRuleOptions`). |

The consumers are not dependencies of this spec. The types, defaults,
validation and small rule maths of an option (bonus feats earned from XP; Speed
in grid squares) live in the owner specs; this spec does not apply the options.
The mechanics of each toolbox rider stay in
`toolbox-fear-and-mental-stress.md` (`shared-types-and-ownership.md` FR-048).

**Out of scope:**

- Monster Death ("Instant Death", p. 17: "a Game Master can ignore this rule
  for an individual monster and treat it like a character"). The SRD text is
  per individual monster, so it is not a table-wide option. It is modelled as
  the per-creature `ZeroHitPointsRule` (`DiesInstantly` / `FallsUnconscious`)
  owned by `hit-points-and-damage.md` (its Q-9, decided). There is no
  `MonsterDeath` type.
- The Initiative method ("Initiative", Rules Glossary, p. 184: "Sometimes a GM
  might have combatants use their Initiative scores instead of rolling
  Initiative"; score is 10 plus Dexterity modifier, plus 5 with Advantage,
  minus 5 with Disadvantage). A GM-chosen alternative, but already modelled as
  the `InitiativeMethod` (`Rolled` / `Score`) input of the command in
  `initiative-and-actions.md` (FR-010); a second owner would describe the same
  SRD option twice. If a table-wide default is wanted later, that spec adds it.
- Knocking Out a Creature ("Knocking Out a Creature" sidebar, p. 17). The
  attacker chooses it per attack. The choice is the `MeleeAttackKnockOut`
  `DamageDelivery` input owned by `hit-points-and-damage.md` (FR-050, FR-051),
  which `attacks.md` only exposes through its delivery (FR-111).
- Multiclassing. The SRD describes it as a rule you have "the option" of
  using (p. 24), but the option is the player's, made at each level
  advancement, and nothing says the table may switch the rule off. It is a
  command input to level advancement
  (`level-advancement-and-multiclassing.md`).
- Rolling or taking the fixed Hit Points when gaining a level ("Instead of
  rolling, you can use the fixed value", p. 23): a per-level choice by the
  player, a `HitPointChoice` of `LevelUpCharacter` in
  `character-aggregate.md` (FR-033, FR-034; the `HitPointGain` type is owned by
  `derived-stats.md`, shared-types-and-ownership.md FR-008).
- Starting equipment ("The GM decides whether your character starts with more
  than the standard equipment", p. 24) and extra backgrounds ("your GM might
  offer additional backgrounds as options", "Choose a Background", p. 19). The
  first is the Starting Equipment at Higher Levels table
  (`higher-level-start-equipment-and-trinkets.md` FR-081 to FR-089 own the table;
  `character-creation.md` FR-079 carries the choice; `content-equipment.md`
  supplies the items); the second is content supplied through `ContentSource`
  (`doc/rules/architecture.md` §3, house rules as content).
- Heroic Inspiration awards (p. 8; mechanics in `heroic-inspiration.md`),
  objects too fragile to need a roll (p. 177), "At the GM's option" Athletics
  checks when climbing a slippery surface ("Climbing", Rules Glossary, p. 179;
  offered as values by `movement-lifting-and-flight.md` FR-033) or moving
  through rough water ("Swimming", p. 189), the GM disallowing a head-severing
  effect (a magic item property, p. 250) and similar rulings. These are GM
  adjudication in the moment, not a stated alternative rule.
- Starting experience points for a higher starting level
  (`higher-level-start-equipment-and-trinkets.md` FR-065, composed by
  `character-creation.md`) and the Character Advancement table (p. 23)
  (`character-aggregate.md` FR-004).
- Movement costs on a grid (entering squares, Difficult Terrain, corners) and
  range counting (p. 13): owned by `grid-positions-and-movement.md`, whose
  FR-001 consumes `CombatRuleOptions.Grid`; `initiative-and-actions.md`
  FR-077 keeps per-foot costs.
- Decoding options from JSON or another external format. A consumer builds the
  value objects directly or through their `NewX` constructors.
- A partial dose of an Ingested poison ("You may decide that a partial dose has
  a reduced effect", p. 197): a per-exposure ruling owned by
  `toolbox-poison.md` (its Q-6), an explicit command field
  (`PoisonPartialDoseRuling`), which satisfies shared-types-and-ownership.md
  FR-045 ("explicit fields in commands, never global flags").
- Prolonged Effects ("Consider the following possibilities", p. 196): a
  per-event Game Master choice, modelled as `ProlongedEffect` in
  `toolbox-fear-and-mental-stress.md`.
- Commonly used optional rules that SRD 5.2.1 does not contain. None of them
  is modelled (`AGENTS.md`, "Licensing and source material").
- Consumer-specific toggles of any kind.

## 2. Source

Page numbering: the PDF page index equals the printed page number (checked:
PDF pages 5, 13, 17, 24 and 196 carry the printed numbers in their footers).
No offset.

- SRD 5.2.1, "Generate Your Scores" ("Your GM might prefer you to use a
  particular one"), p. 21; "Starting at Higher Levels", p. 24; "Bonus Feats at
  Level 20", p. 24; "Playing on a Grid", p. 13; "Fear and Mental Stress"
  ("If you plan to use any of these rules, discuss them with your players at
  the start of the campaign"; "Fear Effects"; "Mental Stress Effects",
  "at your discretion"), p. 196. The owner specs cite the same passages with
  the detail they model.
- Out of scope citations: "Instant Death", p. 17; "Knocking Out a Creature",
  p. 17; "Prolonged Effects", p. 196; "Initiative", p. 184; "Climbing", p. 179;
  "Swimming", p. 189; "Choose a Background", p. 19; "Level Advancement" step 2 (fixed Hit
  Points), p. 23; "Poison", Ingested, p. 197; "Breaking Objects", p. 177; the
  head-severing property (Vorpal Sword), p. 250; Heroic Inspiration, p. 8.
- Package-design requirements derive from `doc/rules/architecture.md` (§3
  optional rules, immutability, purity; §6 errors and results; §7 exports;
  §10 "Global rule toggles"), `doc/rules/observability.md` (§3, domain is
  silent), `doc/rules/testing.md` (§4 checklist) and `errors-and-results.md`.

## 3. Vocabulary

| Term | Meaning here |
|---|---|
| Option | A table-wide alternative that SRD 5.2.1 itself offers to the GM or the table, with a page citation. Not an SRD term: the words "variant" and "optional" are not SRD terms for these rules and do not appear in type names. |
| Options object | One immutable value per context (`CharacterRuleOptions`, `CombatRuleOptions`, `ToolboxRuleOptions`); the package-level concept is `RuleOptions` as named in `doc/rules/architecture.md` §3 and the roadmap. |
| `standard` | The options object whose every field is the SRD's own baseline; in Go the constructors `StandardCharacterRuleOptions()`, `StandardCombatRuleOptions()` and `StandardToolboxRuleOptions()` return it (the zero value of a field-bearing options struct is not assumed to be the baseline). |

Type names follow SRD headings where the SRD has one (`doc/rules/architecture.md`
§3: "Do not invent synonyms"). Names with no SRD source (`NoBonusFeats`,
`NoGrid`) are marked as package terms in their owner specs.

## 4. Requirements

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-023 | Option types (`CharacterRuleOptions`, `CombatRuleOptions`, `ToolboxRuleOptions`, `BonusFeatsAtLevel20` and `PlayingOnAGrid` with their variants, `FearOptions`, `FearRider` and `MentalStressOnSuccess` with its variants (`MentalStressNoDamageOnSuccess`, `MentalStressHalfDamageOnSuccess`; owned by `toolbox-hazard-effects.md` FR-102 to FR-104), and the result types) MUST be immutable: unexported fields, value receivers, no setters, collections copied on construction and on read. The `standard` values and every variant MUST NOT be changeable, and no package-level mutable variable, setter, singleton, `init` function or environment lookup MUST select options. Options reach rules only as a field of a command or state. Every closed variant set is an interface with an unexported marker method and one struct type per variant; the interface doc comment names every variant (`errors-and-results.md` FR-044). The closed variants, `CombatRuleOptions` and `ToolboxRuleOptions` are comparable with `==`; `CharacterRuleOptions` holds a set of methods, so it is compared with its `Equal` method (`character-aggregate.md` FR-116). | architecture.md §3, §10 ("Global rule toggles") | unhappy (immutability, state) |
| FR-029 | An option type MUST NOT be exported from its package until a consuming spec names the FR that reads it. Until then the type is unexported (lower-case identifier) or does not exist. `CharacterRuleOptions` has consumers (`character-creation.md` FR-054, FR-071). The recorded exception is that exporting it also exports `BonusFeatsAtLevel20` and its variants, which have no consuming FR yet (Q-17 settled: an accepted exception, also recorded in `character-creation.md`); the exception closes when `feats-behaviour.md` names one. `ToolboxRuleOptions` is consumed by `toolbox-fear-and-mental-stress.md` FR-100 (with FR-051 and FR-029), so package `toolbox` exports it. `CombatRuleOptions` is consumed by `grid-positions-and-movement.md` FR-001, so exporting that FR's types also exports it and `PlayingOnAGrid`. | architecture.md §7 ("Export deliberately. Anything exported is a compatibility promise."); §3 (`RuleOptions` names each option the SRD itself presents). A package-design rule. | unhappy (scope) |
| FR-033 | Option types MUST NOT import or name any type of a downstream context. `CharacterRuleOptions` imports at most kernel packages (`ability`, `modifier`, `srderr`), `content`, and `derived-stats.md` and `character-aggregate.md` types in package `character`; `CombatRuleOptions` imports only package `combat` and the published API of upstream packages; `ToolboxRuleOptions` imports only package `toolbox` and the published API of upstream packages. No type of the `character` or `combat` options imports `toolbox`, and no context depends on `toolbox`. Consumers import these types, never the reverse. | shared-types-and-ownership.md FR-013, FR-014, FR-015, FR-016, FR-045, FR-057 | unhappy (forbidden edge) |
| FR-034 | The `character`, `combat` and `toolbox` package doc comments (`doc.go`) MUST each state that an option is a value in the command, never a global setting. | architecture.md §3, §10 ("Global rule toggles") | happy |
| FR-038 | No option type, function or factory MUST define or call a use case; the consuming use cases read the options as plain input, so shared-types-and-ownership.md FR-044 is met by construction. | shared-types-and-ownership.md FR-044 | unhappy (scope) |
| FR-039 | Every options object MUST follow one shape: an immutable value with a `Standard...` constructor returning the SRD baseline (every default is the SRD's own, so the standard value yields exactly the SRD rule and a consumer opts into an alternative explicitly, by value, in the command); equality by value (`==` where comparable, otherwise an `Equal` method); `With...` methods (one per field) that return a new value changing only that field, leave the receiver unchanged and apply the constructor's validation; no boolean flag standing in for a named state (alternatives are typed enums or closed interfaces, branched on by a type switch that lists every variant). An options object that can hold an invalid state has a validating constructor `NewX(...) (X, error)` whose error satisfies `errors.Is(err, srderr.ErrInvalidInput)` and, when several things are wrong, is `errors.Join` of typed problems listing every one (errors-and-results.md FR-023, FR-024, FR-035); `MustX` panics with that error and is for test literals only; one that cannot hold an invalid state (every field already a validated value) has a constructor that returns no error. The zero value of an options struct that can be invalid is documented as invalid. | architecture.md §3, §6; errors-and-results.md FR-023, FR-024, FR-035 | happy / unhappy (invalid input) |
| FR-040 | Each options object MUST be defined in the owner spec of section 1 and in no other. A new option MUST be one that SRD 5.2.1 itself names, with a section and page citation, and is added as a field of its context's options object by amending the owner spec. Every field MUST have a consuming FR before its object is exported, except the recorded `BonusFeatsAtLevel20` exception in FR-029. Anything from outside SRD 5.2.1 is never an option (`AGENTS.md`, "Licensing and source material"). | shared-types-and-ownership.md FR-048, FR-057; `AGENTS.md` | unhappy (scope, duplicate) |

### Withdrawn ids

Moved by `shared-types-and-ownership.md` FR-057; the live requirement keeps its
text and test rows in the owner spec. The ids are not reused.

| Id | Requirement |
|---|---|
| FR-001 | Withdrawn: moved to character-aggregate.md (FR-104) |
| FR-002 | Withdrawn: moved to character-aggregate.md (FR-105) |
| FR-003 | Withdrawn: moved to character-aggregate.md (FR-105) |
| FR-004 | Withdrawn: moved to character-aggregate.md (FR-106) |
| FR-005 | Withdrawn: moved to character-aggregate.md (FR-106) |
| FR-006 | Withdrawn: moved to character-aggregate.md (FR-106) |
| FR-007 | Withdrawn: moved to character-aggregate.md (FR-108) |
| FR-008 | Withdrawn: moved to character-aggregate.md (FR-109) |
| FR-009 | Withdrawn: moved to character-aggregate.md (FR-109) |
| FR-010 | Withdrawn: moved to character-aggregate.md (FR-111) |
| FR-011 | Withdrawn: moved to character-aggregate.md (FR-112) |
| FR-012 | Withdrawn: moved to character-aggregate.md (FR-112) |
| FR-013 | Withdrawn: moved to character-aggregate.md (FR-113) |
| FR-014 | Withdrawn: moved to character-aggregate.md (FR-114) |
| FR-015 | Withdrawn: moved to character-aggregate.md (FR-008) (`ExperiencePoints` cannot be negative) |
| FR-016 | Withdrawn: moved to combat-foundations.md (FR-042) |
| FR-017 | Withdrawn: moved to hit-points-and-damage.md (FR-087) (Monster Death) |
| FR-018 | Withdrawn: moved to combat-foundations.md (FR-043) |
| FR-019 | Withdrawn: moved to combat-foundations.md (FR-044) |
| FR-020 | Withdrawn: moved to combat-foundations.md (FR-044) |
| FR-021 | Withdrawn: moved to combat-foundations.md (FR-045) |
| FR-022 | Withdrawn: moved to combat-foundations.md (FR-042) |
| FR-024 | Withdrawn: moved to character-aggregate.md (FR-115) |
| FR-025 | Withdrawn: moved to character-aggregate.md (FR-110) |
| FR-026 | Withdrawn: moved to character-aggregate.md (FR-115) |
| FR-027 | Withdrawn: moved to character-aggregate.md (FR-109) |
| FR-028 | Withdrawn: unused id, no destination (removed in draft before review) |
| FR-030 | Withdrawn: moved to character-aggregate.md (FR-116) |
| FR-031 | Withdrawn: moved to character-aggregate.md (FR-107) |
| FR-032 | Withdrawn: moved to character-aggregate.md (FR-117) |
| FR-035 | Withdrawn: moved to toolbox-hazard-effects.md (FR-099) |
| FR-036 | Withdrawn: moved to toolbox-hazard-effects.md (FR-100) |
| FR-037 | Withdrawn: moved to toolbox-hazard-effects.md (FR-101) |

## 5. Model

`doc/rules/architecture.md` §3 permits "one `RuleOptions` value object (or one
per context)". The kernel packages are reserved for concepts every context needs, so
there is one options object per context, owned by that context
(`shared-types-and-ownership.md` FR-057):

- `CharacterRuleOptions{AbilityScoreMethods, StartingLevel, BonusFeatsAtLevel20}` (a non-empty set of `AbilityScoreGenerationMethod`, an `int`, a `BonusFeatsAtLevel20` variant; unexported fields with accessors of those names)
  in `character-aggregate.md`.
- `CombatRuleOptions{Grid}` (a `PlayingOnAGrid` variant) in `combat-foundations.md`.
- `ToolboxRuleOptions{Fear, MentalStressOnSuccess}` (a `FearOptions` and a `MentalStressOnSuccess` variant)
  in `toolbox-hazard-effects.md`. It sits downstream of every other context,
  which is why it is a separate type and not a field of the other two. Its two
  field types are owned by `toolbox-hazard-effects.md` (FR-102 to FR-104), and
  `toolbox-fear-and-mental-stress.md` imports them (hazard Q-20).

`combat` already depends on `character` in the context map (architecture.md §2),
so a consumer that wants a single bag may compose them in its own code. The
module ships no aggregate `RuleOptions` that would need a context depending on
both. There are no domain events and no aggregate: options are values, not
entities. Nothing here carries bulk data.

Placement (shared-types-and-ownership.md FR-015, FR-016): each object sits in
its owner spec, which precedes its consumers (`character-creation.md`,
`feats-behaviour.md`, `initiative-and-actions.md`). No context cycle arises, and
FR-033 forbids the `character` and `combat` options from importing `toolbox`.
Every default is the SRD's own baseline (FR-039).

## 6. Interface

- No use case, port or adapter. This spec adds no code. No use case calls
  another (`shared-types-and-ownership.md` FR-044); option types define or
  call none (FR-038).
- Use cases in other specs take the relevant options object as a field of
  their command (state in, state out; architecture.md §4). Rules code that
  varies by option uses a type switch listing every variant of the closed option type, or
  reads the permitted-methods set or a `ToolboxRuleOptions` field. No boolean
  flags.
- Exports (FR-029: only once a consuming spec exists for the option): package
  `character` exports `CharacterRuleOptions`,
  `AbilityScoreGenerationMethod`, `BonusFeatsAtLevel20` and its variants and
  the result and problem types; package `combat` exports `CombatRuleOptions`,
  `PlayingOnAGrid` and its variants (consumed by `grid-positions-and-movement.md` FR-001); package `toolbox` exports
  `ToolboxRuleOptions` and the `FearOptions` family (defined by
  `toolbox-hazard-effects.md` FR-102 to FR-104). No facade or re-export package exists; each type is imported from its own package.
  Every exported member has a doc comment starting with its name, and the package
  comment (`doc.go`) lists these as handoff types.

## 7. Observability

Silent by design: the pattern defines no code, and the option types in the
owner specs are pure domain value objects, which `doc/rules/observability.md`
§3 keeps silent. Answers to the seven questions in §2 of that file:

1. **What can go wrong?** A consumer builds invalid options (empty method set,
   level out of range, negative Speed) or forgets non-standard options and gets
   the SRD default. Neither is a package failure.
2. **How would an operator know?** Invalid values surface at once as an
   error from the `NewX` constructor satisfying `errors.Is(err, srderr.ErrInvalidInput)` (caller input error), which for `CharacterRuleOptions` is the joined typed problems.
3. **What would someone need to diagnose it?** The returned error lists the parameter and kind of every
   problem (`errors.Join` of typed problems). Consuming specs decide whether their results echo the options.
4. **What should be counted or timed?** Nothing; constructing a value is not a unit of work.
5. **Is the existing instrumentation still correct?** Yes: this spec adds and changes no signal. Whether a use case signal
   carries an option attribute is decided by that spec (the toolbox fear use
   cases already carry the `srd5e.toolbox.fear.riders` attribute,
   `toolbox-fear-and-mental-stress.md` section 7).
6. **What should a consumer monitor or alert on?** Nothing from here; alert on the consuming use case.
7. **Is anything now noisy?** No: nothing is emitted.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none | n/a | n/a | n/a | n/a |

No entries are added to `doc/observability/signals.md`.

## 8. Test plan

Doubles and contract suites: none. The behavioural tests of each options object
(its constructor, `With...` methods, rule maths and boundary tables) are in the owner
specs' test plans (`character-aggregate.md` section 8, `combat-foundations.md`
section 8, `toolbox-hazard-effects.md` section 8). Tests here are architecture
scans over the option files only, in `internal/archtest` (stdlib `go/parser`, no new dependency).

| Id | Level | Notes |
|---|---|---|
| FR-023 | architecture | Scan the files of packages `character`, `combat` and `toolbox` declaring the types listed in FR-023. Confirms every closed variant set is an interface with an unexported marker method whose doc comment names every variant. Banned patterns: an exported field of a struct type that is not documented read-only (options types use unexported fields), a package-level `var` that is not an error sentinel, a setter method (`Set...`), an `init` function. Fails on any match. (`os` and environment lookups are already banned by the architecture checks.) Plus a unit test that two values built from the same inputs of each closed variant (including the `FearOptions`, `FearRider` and `MentalStressOnSuccess` types), of `CombatRuleOptions` and of `ToolboxRuleOptions` are equal with `==`, and `CharacterRuleOptions.Equal` holds for equal method sets given in different order. |
| FR-029 | architecture | An option type is not exported from its package unless a consuming spec names its FR; the scan lists `CharacterRuleOptions` as consumed (`character-creation.md` FR-054, FR-071) and `ToolboxRuleOptions` as consumed (`toolbox-fear-and-mental-stress.md` FR-100, FR-029, and `toolbox-hazard-effects.md` FR-104), and `CombatRuleOptions` as consumed (`grid-positions-and-movement.md` FR-001), with `BonusFeatsAtLevel20` as the recorded exception, and is updated in the same PR as each further consuming spec. |
| FR-033 | architecture | Import-direction check: the option files import nothing from a downstream context, the `character` and `combat` option files import nothing from `toolbox`, and no context imports `toolbox` (part of the shared-types-and-ownership.md FR-013 check; architecture-guardrails.md FR-098). |
| FR-034 | architecture | Scan: the `character`, `combat` and `toolbox` package doc comments (`doc.go`) contain the required statement. |
| FR-038 | architecture | Import scan: the option files declare and call no `*_usecase.go` function or type. |
| FR-039 | unit, architecture | For each options object: `Standard...()` equals a value built from the SRD baseline inputs; equality and each `With...` method per field (tested in the owner specs); a scan that no option type has a `bool` field standing in for a named state. |
| FR-040 | doc review | Verified by the spec-consistency critic: each options object appears in one owner spec, and every option cites a page of SRD 5.2.1; the table in section 1 is the checklist. |

Mutation testing: all changed lines of the option types in the
owner specs, score at least 80 (`doc/rules/testing.md` §8; `gomu` is run only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`, and is not run by this spec). Every live FR id
appears above; the withdrawn ids have their test rows in the owner specs.
Scenario tests that exercise the options through use cases belong to the specs
that consume them.

## 9. Open questions and decisions

Open: none in this spec. The header phase 0 means the pattern needs no code and
ships with the package skeleton; the option types ship in their owner contexts'
phases (3, 4, 5). Settled: one options object per context, owned by
its context's spec (shared-types-and-ownership.md FR-057); type phase is the
context's phase (FR-014, FR-045). External follow-up, not blocking:
`roadmap.md` still lists `RuleOptions` under Phase 5 (move `CharacterRuleOptions`
to Phase 3, `CombatRuleOptions` to Phase 4, `ToolboxRuleOptions` to Phase 5;
the decisions win until then; export waits for a consuming FR, FR-029). The rank of the `ToolboxRuleOptions` field types
is open question Q-20 in `toolbox-hazard-effects.md`.

Other settled items:

- The permitted ability score methods are a non-empty set (`character-aggregate.md` FR-105).
- `StartingLevel` stays an `int` and delegates its range to `CharacterLevel`;
  `FeatsEarned` takes `ExperiencePoints` (`character-aggregate.md` FR-107, FR-114).

## 10. Changelog impact

`CHANGELOG.md`, under Unreleased, **Added**, when each options object is
exported (FR-029): "Rule options for the optional choices SRD 5.2.1 itself
presents: `CharacterRuleOptions` (ability score generation methods, starting
level, bonus feats at level 20), `CombatRuleOptions` (playing on a grid) and
`ToolboxRuleOptions` (Fear Effects riders, half damage on a successful Mental
Stress save)." The owner specs record their own entries. Not **BREAKING**:
purely additive, and the `standard` options reproduce the SRD baseline, so no
existing rule result changes. Adding a variant to a closed option type later, or
removing one, would be breaking (callers' exhaustive type switches change) (architecture.md §7).
