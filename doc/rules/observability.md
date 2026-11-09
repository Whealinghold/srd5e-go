# srd5e Observability Rules

Normative. MUST / MUST NOT / SHOULD are binding on all code and review.
Read together with [architecture.md](architecture.md); observability follows
its layering and dependency rules.

## 1. Principles

1. **The goal is good observability.** Someone running a system built on srd5e
   MUST be able to tell what it is doing, whether it is healthy, and why it
   gave a particular result, without changing the code. Observability is a
   feature of every change, not an afterthought and not optional polish.
2. **The library emits; the consumer decides where it goes.** srd5e defines
   observability ports in `telemetry`. Consumers inject OpenTelemetry, zap,
   Sentry or anything else through their own adapter. srd5e never chooses a
   backend.
3. **Instrument for diagnosis.** Emit the signals someone would need to debug
   a wrong result, find a slow path, or understand how a rule resolved. The
   evaluation in §2 decides which. Do not wait for an incident to add them.
4. **Signal over noise.** Emit what helps, at the right level, once. Volume is
   a cost for every consumer.
5. **Zero cost when unused.** With no telemetry injected, everything is a
   no-op and behaviour is identical.
6. **Monitoring is part of the design.** A signal nobody can act on is
   incomplete. Each signal states what it is for and what a consumer should
   watch or alert on (see §2).

---

## 2. Evaluating observability is mandatory

Agents and contributors MUST actively evaluate observability and monitoring
for **every change**: a new use case, changed behaviour, a new port or adapter,
a new failure mode, a performance-sensitive path, or a bug fix. A change is
not done until this evaluation has been performed, acted on and reported.
Skipping it, or answering it with a bare "not applicable", is a defect.

### The evaluation

Before finishing, answer each of these for the change. Think about it, do not
just tick it off.

1. **What can go wrong here?** Failure modes: bad content, a port failing,
   invalid input, an invariant violated, unexpected rule interaction.
2. **How would an operator know it went wrong?** What signal (metric, span,
   log) surfaces each failure mode? Is it distinguishable from the normal
   outcome?
3. **What would someone need to diagnose it?** The identifiers, inputs summary,
   intermediate values and outcome, as low-cardinality attributes or in the
   returned result.
4. **What should be counted or timed?** Throughput, outcomes, latency, sizes,
   errors (§5).
5. **Is the existing instrumentation still correct?** Does the change make an
   existing metric, span or log misleading, stale or incomplete? Update it.
6. **What should a consumer monitor or alert on?** Which signal, what
   condition, and why it matters (e.g. any content decode failure; a latency
   regression; an outcome rate that suddenly shifts).
7. **Is anything now noisy?** Does the change add volume without value, or log
   at the wrong level?

### The outcome

- **Add or change the instrumentation** the evaluation calls for, following the
  rest of this document.
- **Update the signal catalogue** (below) for every metric, span or
  log event added, changed or removed.
- **Report it.** The PR description (see `.github/pull_request_template.md`)
  and the agent's final summary MUST contain an `Observability` entry saying
  what was added, changed or removed, and what a consumer should monitor.
- **If you add nothing, justify it specifically.** State why no failure mode,
  diagnostic need or measurement applies to this change (for example: "pure
  rename, no behavioural change; existing `srd5e.d20_test.resolved` unaffected").
  "N/A" or "no changes" alone is not acceptable.
- **Bug fixes:** also add the signal that would have exposed the bug sooner,
  or state why none could have.
- **Reviewers MUST reject** a change that lacks the evaluation or its report.

### The signal catalogue

`doc/observability/signals.md` is the single list of every signal srd5e emits.
It MUST stay in sync with the code. For each signal it records:

| Field | Meaning |
|---|---|
| Name | The metric, span or log-event name, exactly the string value of the Go constant |
| Go constant | The exported constant that holds it, e.g. `d20.SpanResolve` |
| Type | Counter, histogram, gauge, span or log event (with level) |
| Unit | Unit of measure, where applicable |
| Attributes | Allowed attribute keys and their possible values |
| Emitted by | The use case or adapter that produces it (`package.Type.Method`) |
| Purpose | What question it answers or what failure it exposes |
| Suggested monitoring | What consumers should dashboard or alert on, and the threshold or condition to start from |

srd5e is a library and cannot run alerts. The catalogue is how it tells
consumers what to monitor, so the "Suggested monitoring" column MUST be filled
in for every signal where an operator could act on it.

---

## 3. Where observability lives

| Layer | Observability rule |
|---|---|
| Domain types and pure functions (value objects, rules, `dice`, `ability`, `modifier`) | **Silent.** No logging, metrics or spans, and no `telemetry` import. They expose what happened through return values. |
| Use cases (the entry points of a context package, e.g. `d20.Resolver.Resolve`) | **Instrumented.** Emit metrics, spans and logs through the injected ports. This is the main place telemetry is produced. |
| Adapters | **Instrumented only for what the use case cannot see**: retries, fallbacks, partial decodes, I/O timing. Emitted through the same ports, injected the same way. Failures the adapter returns are recorded by the use case, not by the adapter (§7, "Errors: who records what"). |

A context package holds domain and use case in one package (architecture.md
§1), so the split is by file: telemetry calls appear only in use-case files
(`d20/resolver.go`, `signals.go`) (and in adapter packages). Domain files in the same package MUST NOT
import `telemetry` or take a telemetry value.

- Result types already carry the *what* and *why* (modifiers applied, roll
  made, DC, outcome). Use cases translate those into telemetry. They MUST NOT
  re-derive them.
- A use case MUST NOT reach into domain internals to emit telemetry. If the
  domain holds information worth observing, it returns it.

---

## 4. Ports

Defined once in the kernel package `telemetry` (every context uses them; see
[architecture.md](architecture.md) §1). `telemetry` imports no other srd5e
package and only the standard library (`context`, `time`).

| Port | Responsibility |
|---|---|
| `Logger` | Emit a log event: `context.Context`, level (a typed enum), constant message, structured attributes supplied lazily; an error is passed as the `telemetry.Err` attribute. Exposes `Enabled(ctx, level) bool`. |
| `Tracer` | `Start(ctx, name, ...)` returns a derived `context.Context` and a `Span`. `Span` has `SetAttributes`, `RecordError`, `SetStatus` and `End`. |
| `Meter` | Create counters, histograms and gauges; each instrument records a value with attributes |

Each interface is small (1 to 3 methods where possible), carries no vendor
types, and is shaped so an OpenTelemetry adapter is a thin translation. The
exact method sets are defined by the Go declarations; this document owns the
rules, not the signatures.

A use case accepts the ports it needs, either as individual options or as a
bundle struct (`telemetry.Telemetry`) holding a `Logger`, `Tracer` and
`Meter`. The constructor replaces any nil field with the matching Noop, so the
use case never branches on "is telemetry enabled".

**Timing.** Domain and use-case code may not read the clock (A4 in
architecture.md §9: no `time.Now`, `time.Since`). Durations are therefore
measured by the adapters: a span's duration by the `Tracer` implementation, and
duration histograms through a timing helper in `telemetry` that takes an
injected clock (`func() time.Time`) supplied through the use case's options.
The default is the function value `time.Now` (referenced, never called directly, so A4 holds; reference: `d20.NewResolver`), overridden with
`WithClock`. Tests inject a deterministic clock.

**Context propagation.** Trace context travels in the `context.Context` that
`Tracer.Start` returns. Every use case that emits telemetry takes `ctx` as its
first parameter, passes the context returned by `Start` to every port call it
makes (content sources, nested spans, `Logger` calls) and never stores it in a
struct. srd5e does not parse, inject or propagate trace headers: linkage
between parent and child spans is entirely the consumer's `Tracer` adapter
working on that `ctx`.

Rules:

- **No-op by default.** `telemetry.Noop` implementations (`NoopLogger`,
  `NoopTracer`, `NoopMeter`, and a `Telemetry` bundle) hold no state and allocate nothing on the
  hot path. The zero `Telemetry` has nil fields and is valid input to a
  constructor, which calls `WithDefaults`; use `telemetry.Noop{}` for a
  ready-to-use value. They live in `telemetry`, not in an adapter package, so use cases
  can default to them.
- **Dependency injection only.** No package-level logger, tracer or meter, no
  global registries, no `slog.Default()` / `otel.Tracer(...)` lookups, no
  `init()` registration. A zero-value Noop is stateless and is not a global.
- **No vendor types.** Ports use standard-library types only: strings,
  numbers, booleans, `time.Duration`, attributes as plain key/value pairs.
  No OpenTelemetry, zap, Sentry or other SDK type may appear in a port or
  anywhere in this module.
- **Shaped for OpenTelemetry.** Port semantics MUST map cleanly onto the OTel
  model (spans with attributes and status, counters, histograms, severity
  levels) so a consumer's adapter is a thin translation.
- **Small.** Add a port method only when an instrumentation point needs it.
  Ports are interfaces defined next to their vocabulary in `telemetry`; a use
  case that needs only a counter MAY declare its own one-method consumer-side
  interface instead of taking the whole `Meter`.
- **No error returns.** Telemetry methods return nothing. A backend that
  fails must handle it internally; a backend that panics is the host's bug
  and the panic propagates (architecture.md §6). Telemetry never changes a
  use case's result.
- **Synchronous.** Ports are synchronous and non-blocking in contract. A
  backend that buffers or exports does so on its own goroutines; srd5e starts
  none.
- **Test doubles in `telemetry/telemetrytest`**: a recording `Logger`,
  `Tracer` and `Meter` (and a fake clock). Production packages never contain
  doubles.
- **Contract suite.** `telemetrytest` exports `RunLoggerContract`,
  `RunTracerContract` and `RunMeterContract` (each taking `*testing.T` and a
  factory). Every adapter and double MUST pass them: Noop, the recording
  doubles, `slogtelemetry.Logger`, and any adapter a consumer writes.
  The suite checks that methods accept a nil attribute list, tolerate a
  cancelled `ctx`, that `End` is idempotent, that a returned `ctx` still
  carries the caller's values, and that nothing panics for any level.
- **One adapter ships**: `telemetry/slogtelemetry` maps `telemetry.Logger` onto
  a caller-supplied `*slog.Logger` (stdlib only). It never calls
  `slog.Default()` and never creates a handler. Bridges for OpenTelemetry and
  other SDKs are **not** shipped (they would add a dependency for every
  consumer); show them as `Example*` functions in docs and the README.

---

## 5. Metrics

> **Rule: anything worth counting, timing or measuring in aggregate is a
> metric through the `Meter` port,** never only a log line (§8). Consumers
> route metrics to OpenTelemetry through their adapter. srd5e itself never
> imports an OTel package.

### What to record

- **Throughput:** a counter per use case invocation, e.g. a resolved check, an
  applied condition, a computed derived stat.
- **Outcomes:** counters broken down by result, e.g. outcome, rule
  outcome, validation failure, decode failure.
- **Latency:** a duration histogram for each use case, recorded through the
  timing helper (§4).
- **Sizes:** histograms of things that grow, e.g. number of modifiers applied,
  size of an encounter, number of content entries loaded.
- **Errors:** a counter of returned port failures (a non-nil `error` from a
  port) by error type. A failure **result** (content not found, decode
  problems reported as a value) is counted by outcome (`srd5e.outcome` =
  `failure`) plus a variant attribute defined by the use case's spec, and
  carries no `error.type`.
- **Content:** counters or gauges for content loaded, missing and rejected.

### Naming

- Names are lowercase, dot-separated and prefixed: `srd5e.<context>.<noun>` and
  an appropriate suffix, e.g. `srd5e.d20_test.resolved`,
  `srd5e.d20_test.duration`, `srd5e.content.decode_errors`. The wire name uses
  snake_case segments even where they differ from the Go package name (`d20`
  emits `srd5e.d20_test.*`).
- Units follow UCUM as OpenTelemetry does, and are stated in the instrument
  definition, not in the name. Durations are always seconds (`s`, as a
  `float64`); counts of things use an annotation such as `{modifier}`.
- Names (metric, span, log event, attribute key) are exported or package-level
  `const` strings, defined in one `signals.go` per context, or in `telemetry`
  for keys shared across contexts. No string literals at call sites.
  Constants follow Go naming: `SpanResolve`, `MetricResolved`, `AttrOutcome`.
- Renaming or removing a signal or attribute key is a breaking change for
  consumers' dashboards and alerts. Mark it **BREAKING** in `CHANGELOG.md`
  ([architecture.md](architecture.md) §7).

### Attributes

- Attributes MUST be **low-cardinality**: rule names, outcome,
  ability, context name, error type. Never ids, free text, whole
  objects, or unbounded numbers.
- Attributes MUST NOT contain personal or consumer-identifying data.
- Attributes are typed key/value pairs built with the `telemetry` constructors
  (`telemetry.String`, `Int`, `Bool`, `Float`) from constant keys. A value that
  belongs to a closed set is produced from a typed enum's `String()`, never
  from a free string.
- Use consistent attribute keys across metrics, spans and logs
  (`srd5e.outcome`, `srd5e.d20_test.kind`).
- Attribute **values** are shared too, so metrics, spans and logs agree:
  `srd5e.outcome` is `success` or `failure`; `srd5e.d20_test.kind` is
  `ability_check`, `saving_throw` or `attack_roll`;
  `srd5e.d20_test.roll_mode` is `normal`, `advantage` or `disadvantage`;
  `srd5e.d20_test.critical_hit` is `true` or `false` (attack rolls only);
  `srd5e.d20_test.roller_failure` is `unavailable` or `invalid_result`;
  `srd5e.d20_test.forced_failure_reason` is `chosen_to_fail` or `automatic`;
  `srd5e.content.problem_kind` is the snake_case of a `content.ProblemKind`: `syntax`, `unsupported_version`, `missing_field`, `unknown_field`, `wrong_type`, `out_of_range`, `invalid_value`, `duplicate`, `unresolved_reference` or `constraint_violation`;
  `srd5e.content.kind` is the snake_case of a registered `content.Kind`: `species`, `background`, `character_class`, `feat`, `equipment`, `spell`, `magic_item` or `monster` (one value is added per kind a revision row registers);
  `srd5e.character.multiclass` is `true` or `false` (a character with more than one class);
  `srd5e.character.derived_stats.failure` is `content_missing`, `armor_class_basis_not_applicable` or `equipment_slot_mismatch` (a returned failure result of `character.CalculateDerivedStats`);
  `srd5e.character.operation` is `level_up`, `award_experience`, `change_ability_score`, `gain`, `expend`, `resolve_topple_save`, `calculate_spellcasting`, `create_spellcasting`, `choose_additional_spells`, `change_prepared_spells`, `expend_spell_slot`, `regain_spell_slot`, `restore_spell_slots`, `advance_spellcasting`, `attune_magic_item`, `end_magic_item_attunement`, `recharge_magic_item`, `mix_potions`, `check_crafting_materials`, `plan_magic_item_crafting`, `generate_sentient_traits`, `resolve_sentient_conflict`, `attempt_sentient_control`, `repeat_sentient_control_save`, `roll_potion_healing`, `resolve_last_charge`, `roll_spell_storing_levels`, `preview`, `choose_subclass`, `cast_spell`, `complete_casting`, `break_concentration`, `begin_concentration`, `take_feat`, `change_magic_initiate_spell`, `expend_feat_use`, `renew_feat_uses`, `roll_free_casting`, `start_at_higher_level`, `starting_money`, `trinket`, `create`, `roll_ability_scores`, `roll_languages`, `resolve_wish`, `roll_wish_stress_damage`, `expend_species_trait_use`, `renew_species_trait_uses`, `use_relentless_endurance`, `roll_halfling_luck`, `roll_stones_endurance` or `replace_high_elf_cantrip` (the `character` use case that ran; sibling specs add values);
  `srd5e.character.result` is `changed`, `rejected` or `content_missing` (the returned result variant of an aggregate use case), or, for the rest use cases, `applied`, `applied_after_interruption`, `stopped`, `incomplete`, `not_started`, `rejected` (`TakeRest`) or `spent`, `rejected` (`SpendHitPointDie`), or `gained`, `excess_lost`, `excess_given`, `rejected` (`GainHeroicInspiration`) or `expended`, `rejected` (`ExpendHeroicInspiration`), or `target_prone`, `target_resisted` (`ResolveToppleSave`), or `calculated`, `created`, `regained`, `restored`, `advanced` and (with `changed`, `rejected`, `content_missing`, `expended`) the spellcasting use cases, or, for the magic item use cases, `rolled` (`RollPotionMiscibility`), `planned`, `content_incomplete` (`PlanMagicItemCrafting`), `generated` (`GenerateSentientItemTraits`), `resisted`, `demands_made` (the sentient conflict and control use cases), `unchanged` (`RepeatSentientControlSave`), `not_due` (`RechargeMagicItem`), `unavailable` (`CheckCraftingMaterials`), `rolled` (`RollPotionOfHealing`, `RollSpellStoringRingLevels`), `not_listed` or `rolled_elsewhere` (`RollPotionOfHealing`, failure outcomes) or `resolved` (`ResolveLastCharge`, with `rejected`), or, for level advancement, `available` or `unavailable` (`PreviewLevelAdvancement`, `unavailable` being a failure outcome there) and `chosen` (`ChooseSubclass`, with `rejected` and `content_missing`), or, for the casting use cases, `cast`, `started`, `rejected` (`CastSpell`; `CompleteCasting` with `cast` and `rejected`) and `ended`, `none`, `casting_failed` (`BreakConcentration`) or `began`, `casting_failed` (`BeginConcentration`; `casting_failed` is a success, the dependent casting failing as the rules say); for the feat use cases `taken` (`TakeFeat`), `resolved` and `not_applicable` (`RollFreeCasting`, a slot above level 4, a success) for the higher-level start use cases `started` (`StartAtHigherLevel`), `rolled` (`RollHigherLevelStartingMoney`, `RollTrinket`) and `not_applicable` (`RollHigherLevelStartingMoney`, starting level 1, a success) with `changed`, `rejected` and `content_missing` shared, for the creation use cases `created` (`CreateCharacter`), `rolled` (`RollAbilityScores`, `RollLanguages`) and `duplicate` (`RollLanguages`, two rolls giving one language, a failure outcome) with `rejected` and `content_missing` shared, for the Wish use cases `duplicated`, `applied` (`ResolveWish`) and `damage`, `none` (`RollWishStressDamage`) with `rejected` shared, for the species trait use cases `changed` (`ExpendSpeciesTraitUse`, `RenewSpeciesTraitUses`, `UseRelentlessEndurance`), `resolved` (`RollHalflingLuck`, `RollStonesEndurance`: the new roll) and `not_applicable` (`RollHalflingLuck` for a die that is not a 1, `UseRelentlessEndurance` for an Orc killed outright: both successes) with `rejected` and `content_missing` (`ReplaceHighElfCantrip`) shared; absent for a returned port failure;
  `srd5e.character.rejection` is the snake_case of a rule rejection: `maximum_level`, `insufficient_experience`, `multiclass_prerequisites_not_met` (a `LevelUpRejection`), `exceeds_maximum_score` or `constitution_modifier_decrease` (an `AbilityScoreChangeRejection`), or, for a rest, `no_hit_points`, `exhaustion_level_six`, `too_soon_after_long_rest` (the first `RestStartBlocker`), `short_rest_cannot_be_resumed` (a `RestReportRejection`), `none_remaining`, `not_in_pool`, `not_a_short_rest_benefit` or `spent_exceeds_pool` (a `HitDieSpendRejection`), or `recipient_is_holder` or `recipient_already_inspired` (a `GainHeroicInspirationRejection`) or `not_inspired` (the `HeroicInspirationNotAvailable` rejection of an expenditure); or, for spellcasting, a preparation kind (`not_on_class_list`, `spell_level_too_high`, `too_many_prepared`, `duplicate_spell`, `cantrip_not_preparable`, `always_prepared_not_replaceable`, `change_not_allowed_now`, `too_many_replacements`, `not_in_spellbook`, `too_many_cantrips` or `not_a_cantrip`: the first reason of a `PreparationRejection`), `no_slot_available` or `no_expended_slot` (a `SlotOperationRejection`) or `level_lowered`, `class_removed`, `levels_skipped` or `no_change` (a `SpellcastingAdvanceRejection`); or, for the magic item use cases, an `AttuneRejection` (`wrong_rest_purpose`, `rest_not_focused`, `already_attuned`, `duplicate_copy`, `prerequisite_judgement_missing`, `prerequisite_not_met`, `limit_reached`, `rest_interrupted`), an `EndRejection` (`curse_blocks_ending`, `not_attuned`, `not_cursed`, `spell_does_not_end_attunement`, `rest_interrupted`, `rest_not_focused`, `wrong_rest_purpose`, `ending_condition_not_met`), `too_few_potions`, `power_spent`, `charges_remain` (`ResolveLastCharge`), `demands_invalid`, a crafting `ShortfallKind` (`arcana`, `tool`, `spell_not_prepared`, `too_many_assistants`), `rarity_unresolved` or a `GenerateRejection` (`purpose_detail_missing`, `unaligned_not_allowed`, `aligned_purpose_invalid`, `aligned_purpose_needs_non_neutral_alignment`), or, for `ChooseSubclass`, the `String` of a `SubclassRejectionReason` (`not_in_class`, `below_subclass_level`, `already_chosen`, `unknown_subclass`; `PreviewLevelAdvancement` uses the `LevelUpRejection` values above); present only on a rejection (a class feature failure also uses it); for the casting use cases it is the snake_case of the first `CastRejectionKind` in the fixed reporting order: `cannot_cast_spells`, `spell_unavailable`, `untrained_armor`, `casting_already_in_progress`, `not_known`, `not_prepared`, `not_in_spellbook`, `spellbook_not_readable`, `not_a_ritual`, `trigger_not_met`, `missing_component_verbal`, `missing_component_somatic`, `missing_component_material`, `slot_already_spent_this_turn`, `slot_level_too_low`, `no_slot_available` or `casting_not_in_progress`; or, for the feat use cases, the snake_case of a `FeatRejection`: `feat_behaviour_unknown`, `feat_outside_grant`, `feat_not_repeatable`, `feat_prerequisite_not_met`, `feat_choices_invalid`, `feat_increase_rejected`, `spell_change_not_allowed_now`, `spell_change_already_used`, `feat_use_already_expended`, `feat_not_held`, `feat_has_no_limited_use` `feat_not_replaceable`, or, for `StartAtHigherLevel`, the snake_case of a `HigherLevelStartRejection`: `plan_length_mismatch`, `creation_invalid`, `level_up_rejected`, `subclass_rejected`, `increase_rejected`, `increase_not_granted` or `subclass_required`, or, for `CreateCharacter`, the snake_case of the first `CreateCharacterProblemKind` (`class_skill_count_wrong`, `class_skill_not_offered`, `class_skill_repeated`, `class_tool_count_wrong`, `class_tool_not_in_category`, `class_equipment_option_unknown`, `equipment_placeholder_unresolved`, `background_increase_ability_not_listed`, `background_increase_exceeds_maximum`, `background_tool_not_allowed`, `background_tool_missing`, `background_tool_not_in_category`, `background_equipment_option_unknown`, `background_feat_rejected`, `size_required`, `size_not_offered`, `species_choice_missing`, `species_choice_option_unknown`, `species_choice_not_offered`, `species_choice_repeated`, `species_grant_not_offered`, `species_feat_not_origin`, `species_ability_not_offered`, `species_feat_rejected`, `language_count_wrong`, `language_not_standard`, `language_repeated`, `ability_score_method_not_permitted`, `alignment_not_choosable`, `level_advancement_count_wrong`, `level_advancement_rejected`), or, for `ResolveWish`, the snake_case of a `ResolveWishRejectionKind` (`cannot_cast_wish_again`, `spell_level_too_high`, `object_value_too_high`, `object_is_magic_item`, `object_too_large`, `too_many_creatures`, `roll_not_recent`, `feat_required_by_another`, `feat_replacement_rejected`), or, for the Fighter use cases, the snake_case of a `FighterReason` (`not_a_fighter`, `feature_not_gained`, `no_uses_remaining`, `not_an_ability_check`, `check_not_failed`, `not_a_saving_throw`, `die_not_rolled`, `save_not_failed`, `not_own_turn`, `already_used_this_turn`, `fighting_style_already_chosen`, `no_style_to_replace`, `event_not_for_character`, `no_level_gained`, `additional_style_already_chosen`, `not_a_fighting_style_feat`, `same_feat`, `feat_not_repeatable`, `wrong_mastery_count`, `duplicate_mastery`, `not_simple_or_martial_weapon`, `drops_existing_mastery`, `no_long_rest_finished`, `not_currently_chosen`, `already_chosen`, `mastery_not_usable`, `not_in_combat`, `already_has_inspiration`, `no_hit_points`, `not_bloodied`) or, for the Barbarian use cases, the snake_case of a `BarbarianReason` (`not_a_barbarian`, `feature_not_gained`, `not_a_berserker`, `rage_already_active`, `rage_not_active`, `wearing_heavy_armor`, `incapacitated`, `no_bonus_action`, `no_reaction`, `no_rage_uses_remaining`, `rage_began_this_turn`, `rage_already_extended`, `rage_lasts_without_extension`, `not_first_attack_roll`, `reckless_not_declared`, `not_a_strength_attack`, `attack_missed`, `attack_has_disadvantage`, `effect_not_unlocked`, `duplicate_effect`, `too_many_effects`, `frenzy_target_already_hit`, `not_dropped_to_zero`, `persistent_rage_spent`, `presence_not_restored`, `presence_not_used`, `source_not_within_five_feet`, `wrong_mastery_count`, `duplicate_mastery`, `not_simple_or_martial_melee_weapon`, `drops_existing_mastery`, `no_long_rest_finished`, `not_currently_chosen`, `already_chosen`; the ones the Fighter list shares keep one meaning) or, for the Monk use cases, a `MonkReason` (`not_a_monk`, `level_too_low`, `not_eligible`, `already_used`, `no_uses_remaining`, `not_enough_focus_points`; the first that applies, in that order) or or `content_missing` (an `ApplyFighterOptionChoice` call whose feat or weapon the source lacks, a failure result with error status);
  `srd5e.character.ability_score_method` is `standard_array`, `random_generation` or `point_cost` (the method of the assignment in a `CreateCharacter` command; its span and counter, every outcome);
  `srd5e.character.wish_effect` is `duplicate`, `object_creation`, `instant_health`, `resistance`, `spell_immunity`, `sudden_learning`, `roll_redo`, `reshape_reality` (the choice of a `ResolveWish` call; its span and counter, every outcome) and `srd5e.character.wish_lockout` is `true` or `false` (whether the stress locked the creature out of Wish for good; its span and counter, applied results only);
  `srd5e.character.feat_category` is `origin`, `general`, `fighting_style` or `epic_boon` and `srd5e.character.feat_source` is `background` (a custom background too), `species_trait`, `class_feature`, `eldritch_invocation` or `bonus_feat` (`TakeFeat` only, on its counter and span; the category is omitted when the feat is not found);
  `srd5e.character.cast_source` is `slot`, `cantrip`, `ritual`, `feature_granted`, `scroll` or `duplicated` (the snake_case of `CastSource`; `CastSpell` and `CompleteCasting`);
  `srd5e.character.concentration_reason` is `incapacitated`, `dead`, `ended`, `failed_save` or `another_effect` (the snake_case of `ConcentrationBreakReason`; `BreakConcentration`);
  `srd5e.character.last_charge_consequence` is `none`, `destroyed`, `becomes_nonmagical` or `retains_bonus` (the snake_case of `LastChargeConsequence`; the span of `ResolveLastCharge` only);
  `srd5e.character.recharge_moment` is `next_dawn` or `specified_time` (the moment reported to `RechargeMagicItem`; its span only);
  `srd5e.character.pool` is `spellcasting` or `pact_magic` (the slot pool of `ExpendSpellSlot` and `RegainSpellSlot`);
  `srd5e.character.hit_point_choice` is `rolled` or `fixed` (`LevelUpCharacter` only);
  `srd5e.character.rest_kind` is `short` or `long` (the snake_case of `RestKind`; `TakeRest`, `RestoreSpellSlots`);
  `srd5e.character.rest_benefits` is `short`, `long` or `none` (the benefits a `TakeRest` result granted);
  `srd5e.character.hit_die_faces` is the integer `6`, `8`, `10` or `12` (the Hit Point Die of a spend);
  `srd5e.character.rest_operation` is `take_rest` or `spend_hit_point_die` (the duration histogram of the rest use cases only);
  `srd5e.influence.approach` is `deceive`, `intimidate`, `amuse`, `persuade` or `coax` and `srd5e.influence.attitude` is `friendly`, `indifferent` or `hostile` (`character.InfluenceResolver` hesitant calls only);
  `srd5e.influence.response` is `fulfilled`, `refused`, `check_succeeded` or `check_failed` (the returned result variant of `ResolveInfluence`; a forced failure is `check_failed`);
  `srd5e.class` and `srd5e.feature` are the snake_case class and feature of a class feature signal (`srd5e.class` is `fighter`, `barbarian`, `monk`, `rogue`, `wizard`, `cleric`, `druid`, `bard`, `sorcerer`, `warlock`, `paladin` or `ranger`; `srd5e.feature` is `second_wind`, `tactical_mind`, `action_surge`, `indomitable`, `fighting_style` or `weapon_mastery` for the Fighter, the last two by option, `relentless_rage` or `weapon_mastery` for the Barbarian, and `flurry_of_blows`, `patient_defense`, `step_of_the_wind`, `uncanny_metabolism`, `deflect_attacks`, `deflect_redirect`, `wholeness_of_body` or `quivering_palm_end` for the Monk, and `skills`, `equipment`, `cantrips`, `scholar`, `spellbook_start`, `spellbook_level_up`, `spellbook_copy_in`, `spellbook_copy_book`, `spellbook_replace`, `arcane_recovery`, `memorize_spell`, `ritual_adept`, `spell_mastery`, `spell_mastery_replace`, `signature_spells`, `evocation_savant`, `evoker_free_spell`, `potent_cantrip`, `sculpt_spells`, `empowered_evocation` or `overchannel` for the Wizard,, `divine_spark`, `turn_undead`, `preserve_life` or `divine_intervention` for the Cleric, `bardic_inspiration`, `cutting_words`, `peerless_skill` or `countercharm` for the Bard, and `wild_shape`, `wild_companion`, `wild_resurgence`, `lands_aid`, `natural_recovery`, `natures_sanctuary`, `nature_magician` or `primal_strike` for the Druid, mapped by `DruidRules.FeatureAttribute` for the host to record, since the Druid code emits nothing itself, and `metamagic` (`ChooseMetamagicOptions`, `UseMetamagic`), `font_of_magic` (`CreateSpellSlot`) or `sorcerer_level_up` (`ApplySorcererLevelUp`) for the Sorcerer, `eldritch_invocations` (`ChooseWarlockInvocations`), `mystic_arcanum` (`ChooseWarlockMysticArcanum`), `invocation_spell` (`CastInvocationSpell`), `familiar_form` (`ChooseWarlockFamiliarForm`), `book_of_shadows` (`ConjureBookOfShadows`), `pact_weapon` (`BondPactWeapon`), `lifedrinker` (`UseLifedrinker`) or `dark_ones_own_luck` (`UseDarkOnesOwnLuck`) for the Warlock, and `lay_on_hands` (`UseLayOnHands`), `divine_sense`, `abjure_foes` or `sacred_weapon` (the options of `UsePaladinChannelDivinity`), `paladins_smite` or `faithful_steed` (`UseFreeCast`), `holy_nimbus` (`UseHolyNimbus`), `smite_of_protection` (`ApplySmiteOfProtection`), `choose_option` (`ChoosePaladinOption`) or `calculate_features` (`CalculatePaladinFeatures`) for the Paladin, and `tireless` (`UseTireless`), `ranger_features` (`ResolveRangerFeatures`) or, for `ChooseRangerOption`, the feature chosen: `weapon_mastery`, `deft_explorer`, `fighting_style`, `expertise`, `hunters_prey`, `defensive_tactics` or `feat_grant` for the Ranger); each class spec adds its values;
  for the Cleric `srd5e.character.rejection` is the snake_case `ClericReason`: `not_a_cleric`, `below_level_2`, `effect_not_available`, `no_uses_remaining`, `target_is_self`, `target_not_visible`, `target_out_of_range`, `damage_type_not_allowed`, `sear_undead_not_gained`, `target_not_undead`, `duplicate_target`, `target_not_bloodied`, `negative_allocation`, `allocation_exceeds_room`, `allocation_exceeds_pool`, `already_chosen`, `order_missing`, `both_orders`, `order_unknown`, `extra_cantrip_missing`, `extra_cantrip_not_cleric_cantrip`, `extra_cantrip_already_known`, `below_level_7`, `option_missing`, `both_options`, `unknown_option`, `below_level_10`, `not_available`, `not_cleric_spell`, `spell_level_too_high`, `reaction_casting_time`, `no_option_chosen`, `other_option_chosen`, `turn_spent`, `not_weapon_attack`, `not_cleric_cantrip`, `no_damage_dealt`, `below_level_14` and `recipient_out_of_range` (the two use cases record the reasons of their rows in table 4.14.1 of class-cleric.md; the others are results of the domain functions and are not attributes);
  for the Bard `srd5e.character.rejection` is the snake_case `BardReason`: `not_a_bard`, `self_target`, `no_uses_remaining`, `recipient_already_inspired`, `below_required_level`, `nothing_to_restore`, `slot_rejected`, `not_a_failure`, `expired`, `feature_unavailable`, `ineligible_trigger`, `ineligible_effect` and `save_not_failed` (the four Bard use cases record the reasons of the rows of class-bard.md section 7: `not_a_failure` and `expired`; `not_a_bard`, `feature_unavailable`, `no_uses_remaining` and `ineligible_trigger`; `below_required_level`, `ineligible_effect` and `save_not_failed`; the others are results of the domain functions and are not attributes);
  for the Sorcerer `srd5e.character.rejection` is the snake_case `SorcererRejectionKind`: `not_a_sorcerer`, `feature_not_gained`, `elemental_affinity_choice_invalid`, `elemental_affinity_type_mismatch`, `elemental_affinity_already_applied`, `metamagic_selection_invalid`, `metamagic_replacement_invalid`, `innate_sorcery_already_active`, `innate_sorcery_no_uses_left`, `innate_sorcery_uses_remain`, `sorcery_incarnate_unavailable`, `innate_sorcery_not_active`, `sorcery_points_insufficient`, `no_slot_to_convert`, `sorcery_points_would_exceed_maximum`, `slot_level_not_creatable`, `sorcerer_level_below_minimum`, `created_slot_unavailable`, `created_slot_level_too_low`, `created_slot_spent_this_turn`, `metamagic_option_not_known`, `metamagic_combination_refused`, `quickened_restriction_applies`, `metamagic_not_eligible`, `restoration_already_used`, `restoration_amount_invalid`, `restoration_not_a_short_rest`, `dragon_wings_already_used`, `dragon_wings_use_not_expended`, `dragon_wings_not_active`, `dragon_companion_free_cast_used` (the four Sorcerer use cases; the Metamagic eligibility reasons are not attributes, a spell the option does not suit is `metamagic_not_eligible`);
  for the Warlock `srd5e.character.rejection` is the snake_case `WarlockRejectionKind`: `not_a_warlock`, `feature_not_gained`, `invocation_selection_invalid`, `invocation_not_known`, `not_a_spell_grant`, `light_level`, `already_cast`, `magical_cunning_already_used`, `rite_too_short`, `contact_patron_already_used`, `no_arcanum_at_level`, `wrong_spell_level`, `not_on_warlock_list`, `arcanum_too_many_replacements`, `not_an_arcanum_spell`, `arcanum_already_cast`, `not_fiend_patron`, `not_an_enemy`, `target_too_far`, `force_not_allowed`, `not_after_rest`, `not_a_hit`, `already_used_this_turn`, `hurl_through_hell_already_spent`, `hurl_through_hell_use_not_spent`, `no_pact_magic_slot`, `prone_not_allowed_for_size`, `no_uses_left`, `already_used_on_this_roll`, `roll_kind_not_allowed`, `not_melee_simple_or_martial`, `attuned_by_another`, `bonded_by_another_warlock`, `damage_type_not_offered`, `no_hit_point_die`, `familiar_form_not_offered`, `no_familiar`, `no_attack_to_forgo`, `reaction_spent`, `no_bonus_action`, `damage_not_changeable`, `target_type_not_offered`, `not_willing`, `not_touched`, `not_on_same_plane`, `link_ended`, `cantrip_count_wrong`, `spell_count_wrong`, `book_spell_refused`, `no_page`, `permission_not_given`, `no_action_available`, `page_full`, `name_already_on_page`, `name_not_on_page`, `no_magic_action`, `gift_of_the_protectors_already_spent` and `killed_outright` (the eight Warlock use cases record the reasons of the invocation choice, of the Mystic Arcanum choice, of the invocation spell, of the familiar form, of the Book of Shadows, of the pact weapon bond, of Lifedrinker and of Dark One's Own Luck; the others are results of the domain functions and are not attributes);
  for the Paladin `srd5e.character.rejection` is the snake_case `PaladinReason`, the first that holds when a use case reports several: `not_a_paladin`, `invalid_amount`, `empty_request`, `condition_not_removable`, `restoring_touch_not_available`, `exceeds_pool`, `feature_not_gained`, `option_not_available`, `not_taking_attack_action`, `no_melee_weapon_held`, `no_uses_remaining`, `already_used_since_long_rest`, `not_expended`, `no_level_5_slot_available`, `level_too_low`, `already_chosen`, `not_a_fighting_style_feat`, `feat_not_repeatable`, `wrong_count`, `duplicate`, `not_a_cantrip`, `not_on_cleric_list`, `not_blessed_warrior`, `not_held`, `not_active` and `reason_not_applicable` (`not_a_candidate` and `not_active` are results of domain functions and are not attributes);
  for the Ranger `srd5e.character.rejection` is the snake_case `RangerReason`, the first that holds when a use case reports several: `not_a_ranger`, `feature_not_available`, `not_hunter`, `no_uses_remaining`, `wrong_moment`, `nothing_to_replace`, `already_chosen`, `same_option`, `unknown_option`, `not_proficient`, `already_has_expertise`, `no_grant_remaining`, `invalid_choice` and `no_reaction` (the three Ranger use cases record the reasons of the rows of class-ranger.md section 7; `no_reaction` and `same_option` are results of domain functions and of `ChooseRangerOption`, `UseTireless` never reports them);
  for the species trait use cases `srd5e.character.rejection` is the snake_case `SpeciesTraitRejection`, the first that holds in the order of table O of species-trait-behaviour.md: `species_behaviour_unknown`, `species_use_not_offered`, `species_use_has_own_use_case`, `species_use_not_yet_available`, `not_enough_space_for_large_form`, `species_use_exhausted`, `species_choices_invalid` (a High Elf replacement cantrip that is not a Wizard cantrip or is the one held), `cantrip_window_closed`, `die_already_rerolled` and `character_mismatch` (a rest event of another character);
  for the Wizard `srd5e.character.rejection` is the snake_case cause of the rejection type of the use case: `wrong_skill_count`, `duplicate_skill`, `skill_not_allowed`, `unknown_equipment_option`, `wrong_cantrip_count`, `duplicate_cantrip`, `not_a_cantrip`, `not_on_wizard_list`, `feature_not_available`, `wrong_spell_count`, `duplicate_spell`, `is_a_cantrip`, `above_highest_spell_level`, `already_in_spellbook`, `not_in_source_spellbook`, `not_after_short_rest`, `already_used`, `slot_level_too_high`, `slot_not_expended`, `exceeds_maximum`, `already`, `skill_not_eligible`, `not_proficient`, `not_prepared`, `always_prepared_not_replaceable`, `new_spell_is_cantrip`, `new_spell_not_in_spellbook`, `new_spell_above_highest_level`, `new_spell_already_prepared`, `not_after_long_rest`, `not_held`, `not_a_mastery_spell`, `not_in_spellbook`, `wrong_spell_levels`, `casting_time_not_action`, `not_level_three`, `spellbook_unavailable`, `not_a_ritual_spell`, and for the Evoker `not_evoker`, `no_new_spell_level`, `not_an_evocation_spell`, `unsupported_test_kind`, `spell_deals_no_damage`, `caster`, `creature_not_eligible`, `too_many_creatures`, `not_a_wizard_spell`, `already_empowered`, `no_spell_slot_used` or `slot_level_out_of_range`, plus `content_missing`, `subclass_unknown` (an Evoker spellbook use case given a subclass id the Wizard does not list) and `not_a_wizard`;
  `error.type` is `<port>_error` from `srderr.ErrorType` for a `*srderr.PortError` (one value per port, for example `dice_roller_error`; [errors-and-results.md](../specs/errors-and-results.md)). Nothing is emitted for other error types; a future non-port error type must first add its own constant to this list and the catalogue. It is set only for an error that left the use case, never for a returned
  failure result. A new value for an existing key is added to this list and to
  the catalogue.

---

## 6. Traces

- Wrap each top-level use case in a span through the `Tracer` port. The span
  name is the use case, e.g. `srd5e.d20_test.resolve`. Start it first, `defer
  span.End()` immediately, and pass the returned `ctx` on (§4).
- Add child spans only for steps that are slow or that cross a port (content
  lookup, decoding). Do not span trivial functions.
- Attach the same low-cardinality attributes as the matching metrics.
- On a port failure, call `span.RecordError(err)` and `span.SetStatus` with
  error status, and return the error. Do not swallow it. A failure **result**
  (failed decode, content not found) also sets error status; a normal rule
  outcome (failed check) does not.
- A panic that `srderr.Guard` recovers from a port is converted into a
  `*srderr.PortError` and then follows the port-failure path above. Any other
  panic is a bug: the deferred `span.End()` still runs, and the panic
  propagates.
- srd5e does not manage trace context beyond passing `ctx` (§4). Parent/child
  linkage is the consumer adapter's job.

---

## 7. Logging

### Levels

Use the lowest level that serves the reader. `telemetry.Level` maps to
`log/slog` levels and to OpenTelemetry severity.

| Level | Use for | Examples |
|---|---|---|
| `LevelTrace` | Step-by-step detail useful only when chasing a specific bug. Off by default; MUST be cheap when off. | Each modifier considered and why it was dropped by stacking rules |
| `LevelDebug` | Normal operation details a developer would want while debugging | Use case started with inputs summary; content source selected; result summary |
| `LevelInfo` | Significant lifecycle events, rare. At most a handful per process, never per call. | Content set loaded (name, count); engine configured |
| `LevelWarn` | Something is wrong but handled; the result is still valid but possibly not what was intended | Unknown optional field ignored while decoding; deprecated content format; falling back to a default |
| `LevelError` | An operation failed and the failure is **not** surfaced to the caller as a returned error or failure result (so a log is the only trace of it) | A best-effort step that failed and was skipped |

There is no fatal level; this package never terminates the process.

- **A library MUST NOT log at `LevelInfo` per use case call.** Per-call detail
  is `LevelDebug` or below. Consumers enable it when needed.
- A rule outcome the player simply failed (a failed check, an illegal but
  expected action) is **not** a warning or an error. It is a normal result.
  Log at `LevelDebug` at most.
- `LevelError` is rare in a library. If the failure reaches the caller as a
  result or error, the caller decides whether to log it, so srd5e does not.

### Errors: who records what

Each failure is recorded once, by the right owner, in the right signal:

| Failure | Recorded by | How |
|---|---|---|
| Returned out of the use case as an `error` (port failure, already wrapped with `srderr.WrapPort`) | The use case at whose boundary it leaves srd5e | `span.RecordError` + error status; increment the error counter with an `error.type` attribute. **No log.** |
| Returned out of the use case as a failure result (decode failure, content not found) | The use case that returns it | Span error status; count it with `srd5e.outcome` = `failure` plus a variant attribute defined by the use case's spec. **No `error.type`, no log.** A result the rules merely refuse is a normal outcome (last row), not this one. |
| Returned by an adapter to a use case | The use case (as above), not the adapter | Adapter emits nothing for it |
| Handled inside srd5e and not surfaced (fallback, skipped best-effort step, ignored unknown field) | The code that handled it | `LevelWarn` log (or `LevelError` if the operation could not complete) + counter |
| Programmer error (panic from a `MustX` helper or unreachable branch) | Nobody in srd5e | The panic propagates; it is a bug for the caller to see. The deferred `span.End()` still runs. |
| Normal rule outcome (failed check, illegal action) | The use case | Outcome attribute on the counter and span. Not an error. |

Use cases never call one another (architecture.md §4), so none records another's
signals and an error is counted once, by the use case it left. Attribute values
come from the result value.

### Log intelligently

- **Log at boundaries, not in the middle.** Spans already record the start,
  end and duration of a use case, so do not log those. At most one
  `LevelDebug` summary per use case call. Log at a port call only for what a
  span cannot hold. Do not narrate internal steps.
- **One event per occurrence.** Do not log the same fact at several layers,
  and never log an error that is also returned (see the table above).
- **Do not log what the result already says.** If the result type carries the
  breakdown, the log line refers to the outcome, not a copy of it.
- **Structured, not prose.** Short constant message, details in attributes:
  `log.Log(ctx, telemetry.LevelDebug, MsgResolved, func() []telemetry.Attr {...})`,
  not `fmt.Sprintf("the check by %s against DC %d rolled %d", ...)`.
- **No hot-loop logging.** Never log inside per-modifier or per-combatant loops
  above `LevelTrace`. Aggregate instead: log counts, or record a metric.
- **Cheap when disabled.** Do not build attribute slices eagerly. Attributes are
  supplied as a closure the `Logger` calls only when the level is enabled;
  there is no eager form. Messages are constants, so never need building.
- **Actionable messages.** An error or warning says what failed, with enough
  attributes to find it (context, rule or content identifier, port), and what
  the consumer can do.
- **No secrets or personal data** in messages or attributes.
- **No printing.** No `fmt.Print*`, `println` or standard-library `log` in any
  non-test file (architecture.md §9, A3 and A4). Anything that would be printed
  is a `LevelDebug` or `LevelTrace` log.

---

## 8. Logs, metrics and traces together

- Pick the right signal. Counting or timing something is a **metric**. Seeing
  the path of one operation is a **trace**. Explaining one unusual event is a
  **log**.
- If a log line would be useful in aggregate (you would grep and count it), it
  should also be a metric. Do not make consumers parse logs to count things.
- The same fact uses the same attribute keys across all three signals so a
  consumer can correlate them.

---

## 9. Testing

- Use cases MUST be tested for their emitted telemetry using the
  `telemetrytest` recording doubles: the expected metrics, span names,
  attribute keys and log levels.
- A test MUST assert that a use case returns an identical result with the
  zero-value Noop and with a recording bundle. Table-driven, run with
  `t.Parallel()` where the doubles are not shared.
- Failure tests assert the error is recorded exactly once (span status, one
  counter increment) and is not also logged.
- Pure domain code has no telemetry tests, because it has no telemetry.
- Metric names and attribute keys are part of the public contract. Tests that
  pin them (comparing each constant to its literal catalogue value) guard
  against accidental renames.
- Time-dependent signals use an injected fake clock; no test sleeps.
- Adapters (`slogtelemetry`, Noop, doubles) run the exported contract suites
  from §4 in their own tests, plus their specific behaviour.

---

## 10. Enforcement

The guardrails below are enforced by `internal/archtest` (architecture.md §9) and
the contract suites, which fail `go test`, only where an item names its check
as implemented (A1 to A13 today). Items marked "Not yet enforced" are
specified and rest on review until their check lands. The checks that apply to
observability are:

- **Imports (A1, A3).** `telemetry` imports no srd5e package; no file in the
  module imports an observability or error-tracking SDK (`go.opentelemetry.io`,
  `github.com/getsentry`, `go.uber.org/zap`, `github.com/rs/zerolog`, ...), the
  standard `log` package, and `go.mod` has no `require`. A3 enforces the `log` ban in kernel
  and context packages and any third-party import anywhere (which covers the SDK ban). Confining
  `log/slog` to `telemetry/slogtelemetry` is not yet enforced; review covers it, and test files such as
  `scenarios/example_test.go` may use `log/slog` for examples.
- **Domain files silent (O1). Not yet enforced.** A non-test file that is not a use-case file or `signals.go` (and
  not under `telemetry/`) importing `telemetry`.
- **No globals (A4).** A package-level `var` other than `_` assertions and
  `Err*` sentinels, which covers static loggers, tracers and meters; `init`
  functions.
- **No printing (A4).** `fmt.Print*`, `println`.
- **No literal names (O2). Not yet enforced.** A string literal passed as the name argument of a
  `Tracer.Start`, `Meter` instrument constructor or `Logger` call, or as an
  attribute key, instead of a defined constant.
- **Catalogue matches code (O3). Not yet enforced.** Every signal constant in a `signals.go` file
  appears in `doc/observability/signals.md` with the same value, and every
  catalogue row's Go constant exists. Entries with an empty "Purpose" or
  "Suggested monitoring" cell fail.
- **Closed value sets (O4). Not yet enforced.** A shared attribute value (the sets in §5
  "Attributes") used in code or listed in the catalogue that is outside its
  defined set.
- **No-op shape (O5). Not yet enforced.** A Noop type that has fields, is not usable as a zero
  value, or lives outside `telemetry`.
- **Test-support isolation (A2).** Only `_test.go` files import `*test` packages (enforced). That doubles live only in `*test` packages is not yet enforced; review covers it.
- **Contract suites.** `telemetrytest` suites run against Noop, the recording
  doubles and `slogtelemetry` in their own packages' tests. A new adapter
  without a contract test fails review and the coverage target.

O1 to O5 will be lexical or AST-level like the A-checks, each with violating and
clean in-memory fixtures (`testing/fstest` overlays in `internal/archtest`). They are specified here
and added to the check table in architecture.md §9 when implemented; until a
check exists, review enforces its rule.

Process checks (evaluation performed, `Observability` entry reported) cannot be
fully automated; they are enforced in review (§2).

---

## 11. Anti-patterns (prohibited)

- **Instrumenting the domain.** It breaks purity and couples rules to a backend.
- **Vendor lock-in:** SDK types in ports, or SDK imports anywhere in the module.
- **Package-level loggers or global tracers**, including reading
  `slog.Default()` or an SDK's global provider.
- **Storing `context.Context` in a struct** to emit telemetry later, or
  dropping the `ctx` returned by `Tracer.Start`.
- **Reading the clock** in domain or use-case code to measure duration.
- **Log spam:** per-call `LevelInfo`, logging in tight loops, logging the same
  event at several layers.
- **Log-and-return:** logging an error and then returning it.
- **Double counting:** an error counted by both an adapter and the use case,
  or a use case that calls another use case and so records its signals twice.
- **Wrong severity:** a failed check recorded as an error, or a handled
  fallback logged at `LevelDebug` instead of `LevelWarn`.
- **High-cardinality attributes:** ids, free text or unbounded values.
- **Prose logs** with the data interpolated into the message.
- **Counting by log parsing:** a countable event that exists only as a log line.
- **Unnamed constants:** metric and attribute names scattered as literals.
- **Telemetry that changes behaviour:** a use case whose result, error or
  control flow depends on whether telemetry is enabled.
- **Skipping the evaluation:** finishing a change without assessing
  observability, or reporting "N/A" without a specific reason.
- **Signals nobody can use:** emitting telemetry with no stated purpose or
  monitoring guidance, or leaving the catalogue out of date.
- **Instrument-after-incident:** waiting for a production problem before adding
  the signal that would have exposed it.

---

## Checklist before submitting

- [ ] I evaluated observability and monitoring for this change (§2), thinking
      through all seven questions.
- [ ] The commit/PR and my final summary include an `Observability` entry
      (signals added, changed, removed, and what to monitor) or a specific
      reason none were needed.
- [ ] `doc/observability/signals.md` is updated, with Go constant, purpose and
      suggested monitoring filled in.
- [ ] For a bug fix: I added the signal that would have exposed it, or said why
      none could.
- [ ] Telemetry is produced in use-case files (and adapters only for what
      the use case cannot see), never in domain files.
- [ ] Each failure is recorded once, by its owner, per "Errors: who records
      what"; nothing is logged and also returned.
- [ ] Everything goes through injected ports with Noop defaults; no vendor
      imports, no globals, no stored `ctx`.
- [ ] The `ctx` from `Tracer.Start` is passed to every downstream port call.
- [ ] Anything countable or timeable is a metric, not only a log line.
- [ ] Durations are recorded in seconds through the injected clock.
- [ ] Signal and attribute names are defined constants, correctly prefixed.
- [ ] Attributes are low-cardinality and free of personal data.
- [ ] Log levels are the lowest that serve the reader; nothing noisy at
      `LevelInfo`.
- [ ] No duplicate logging; no log-and-return; no printing.
- [ ] Telemetry is covered by tests using the `telemetrytest` doubles; new
      adapters pass the contract suites.
- [ ] `go test -race -shuffle=on ./...` passes, including `internal/archtest`.

---

## Read receipt

RULE-TAG: `observability-45ede554bcb1`

This tag proves you read this document in this session. Quote it verbatim as
required by `AGENTS.md`. Do not copy it anywhere else. Whoever edits this
document MUST replace the tag with a fresh random value.
