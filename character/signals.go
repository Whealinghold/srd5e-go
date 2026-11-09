package character

// The signals the character use cases emit, defined once. The names and
// attribute keys are a public contract: renaming one breaks consumers'
// dashboards and alerts. They are catalogued in doc/observability/signals.md.
// The shared keys telemetry.AttrOutcome and telemetry.AttrErrorType are used as
// they are.
const (
	// SpanDerivedStatsCalculate wraps one CalculateDerivedStats.Run call.
	SpanDerivedStatsCalculate = "srd5e.character.derived_stats.calculate"
	// SpanDerivedStatsLoadContent wraps the content lookups of one call.
	SpanDerivedStatsLoadContent = "srd5e.character.derived_stats.load_content"
	// MetricDerivedStatsCalls counts calls that return a result or a port failure ({call}).
	MetricDerivedStatsCalls = "srd5e.character.derived_stats.calls"
	// MetricDerivedStatsErrors counts port failures leaving the use case ({error}).
	MetricDerivedStatsErrors = "srd5e.character.derived_stats.errors"
	// MetricDerivedStatsDuration records call durations (s), failed calls included.
	MetricDerivedStatsDuration = "srd5e.character.derived_stats.duration"
	// MetricDerivedStatsLevels records the total level of successful calls ({level}).
	MetricDerivedStatsLevels = "srd5e.character.derived_stats.levels"
)

// Attribute keys.
const (
	// AttrMulticlass is a bool: whether the character holds more than one class.
	AttrMulticlass = "srd5e.character.multiclass"
	// AttrDerivedStatsFailure is the variant of a returned failure result: a
	// ValueDerivedStats* value; failure results only.
	AttrDerivedStatsFailure = "srd5e.character.derived_stats.failure"
)

// Attribute values.
const (
	ValueDerivedStatsContentMissing        = "content_missing"
	ValueDerivedStatsBasisNotApplicable    = "armor_class_basis_not_applicable"
	ValueDerivedStatsEquipmentSlotMismatch = "equipment_slot_mismatch"
)

// Spans and metrics of the aggregate use cases LevelUpCharacter,
// AwardExperience and ChangeAbilityScore, and the class feature signals every
// class feature use case reuses.
const (
	// SpanLevelUp wraps one LevelUpCharacter.Run call.
	SpanLevelUp = "srd5e.character.level_up"
	// SpanAwardExperience wraps one AwardExperience.Run call.
	SpanAwardExperience = "srd5e.character.award_experience"
	// SpanChangeAbilityScore wraps one ChangeAbilityScore.Run call.
	SpanChangeAbilityScore = "srd5e.character.change_ability_score"
	// SpanLoadContent wraps the content lookups of a character use case.
	SpanLoadContent = "srd5e.character.load_content"
	// MetricOperations counts character use case calls ({operation}).
	MetricOperations = "srd5e.character.operations"
	// MetricDuration records character use case durations (s).
	MetricDuration = "srd5e.character.duration"
	// MetricFeatureUsed counts class feature uses ({use}).
	MetricFeatureUsed = "srd5e.character.feature_used"
	// MetricFeatureUsedDuration records class feature use durations (s).
	MetricFeatureUsedDuration = "srd5e.character.feature_used.duration"
	// MetricFeatureUsedErrors counts port failures leaving a class feature use case ({error}).
	MetricFeatureUsedErrors = "srd5e.character.feature_used.errors"
)

// Attribute keys of the aggregate use cases and of class features.
const (
	// AttrOperation is the use case: a ValueOperation* value; operations counter and duration histogram.
	AttrOperation = "srd5e.character.operation"
	// AttrResult is the result variant: a ValueResult* value.
	AttrResult = "srd5e.character.result"
	// AttrRejection is the rule rejection: a ValueRejection* value; rejections only.
	AttrRejection = "srd5e.character.rejection"
	// AttrHitPointChoice is how the level's Hit Points were chosen: a ValueHitPointChoice* value.
	AttrHitPointChoice = "srd5e.character.hit_point_choice"
	// AttrClass is the class of a class feature signal; the value is the class's snake_case name, added by the class spec.
	AttrClass = "srd5e.class"
	// AttrFeature is the feature of a class feature signal; values are added by the class specs.
	AttrFeature = "srd5e.feature"
)

// Attribute values.
const (
	ValueOperationLevelUp            = "level_up"
	ValueOperationAwardExperience    = "award_experience"
	ValueOperationChangeAbilityScore = "change_ability_score"

	ValueResultChanged        = "changed"
	ValueResultRejected       = "rejected"
	ValueResultContentMissing = "content_missing"

	ValueRejectionMaximumLevel                  = "maximum_level"
	ValueRejectionInsufficientExperience        = "insufficient_experience"
	ValueRejectionMulticlassPrerequisitesNotMet = "multiclass_prerequisites_not_met"
	ValueRejectionExceedsMaximumScore           = "exceeds_maximum_score"
	ValueRejectionConstitutionModifierDecrease  = "constitution_modifier_decrease"

	ValueHitPointChoiceRolled = "rolled"
	ValueHitPointChoiceFixed  = "fixed"
)

// Spans and metrics of the rest use cases TakeRest and SpendHitPointDie.
const (
	// SpanTakeRest wraps one RestTaker.TakeRest call.
	SpanTakeRest = "srd5e.character.take_rest"
	// SpanSpendHitPointDie wraps one HitDieSpender.SpendHitPointDie call.
	SpanSpendHitPointDie = "srd5e.character.spend_hit_point_die"
	// MetricRests counts TakeRest calls ({rest}).
	MetricRests = "srd5e.character.rests"
	// MetricHitPointDiceSpent counts SpendHitPointDie calls ({spend}).
	MetricHitPointDiceSpent = "srd5e.character.hit_point_dice_spent"
	// MetricHitDieHitPoints records the Hit Points regained per spent die ({hit_point}).
	MetricHitDieHitPoints = "srd5e.character.hit_die_hit_points"
	// MetricRestOperationDuration records the duration of each rest use case (s).
	MetricRestOperationDuration = "srd5e.character.rest_operation_duration"
)

// Attribute keys of the rest use cases.
const (
	// AttrRestKind is the kind of rest reported: RestKind.String, "short" or "long".
	AttrRestKind = "srd5e.character.rest_kind"
	// AttrRestBenefits is the benefits a TakeRest result granted: a ValueRestBenefits* value.
	AttrRestBenefits = "srd5e.character.rest_benefits"
	// AttrHitDieFaces is the faces of the Hit Point Die spent, an int: 6, 8, 10 or 12.
	AttrHitDieFaces = "srd5e.character.hit_die_faces"
	// AttrRestOperation is the rest use case: a ValueRestOperation* value; the duration histogram only.
	AttrRestOperation = "srd5e.character.rest_operation"
)

// Attribute values of the rest use cases. The other rejection values
// (none_remaining, not_in_pool, not_a_short_rest_benefit, spent_exceeds_pool,
// short_rest_cannot_be_resumed) are the String of HitDieSpendRejection and
// RestReportRejection.
const (
	ValueRestOperationTakeRest         = "take_rest"
	ValueRestOperationSpendHitPointDie = "spend_hit_point_die"

	ValueResultApplied                  = "applied"
	ValueResultAppliedAfterInterruption = "applied_after_interruption"
	ValueResultStopped                  = "stopped"
	ValueResultIncomplete               = "incomplete"
	ValueResultNotStarted               = "not_started"
	ValueResultSpent                    = "spent"

	ValueRestBenefitsShort = "short"
	ValueRestBenefitsLong  = "long"
	ValueRestBenefitsNone  = "none"

	ValueRejectionNoHitPoints          = "no_hit_points"
	ValueRejectionExhaustionLevelSix   = "exhaustion_level_six"
	ValueRejectionTooSoonAfterLongRest = "too_soon_after_long_rest"
)

// Spans of the Heroic Inspiration use cases. They record on the shared
// MetricOperations counter and MetricDuration histogram.
const (
	// SpanGainHeroicInspiration wraps one GainHeroicInspiration.Gain call.
	SpanGainHeroicInspiration = "srd5e.character.gain_heroic_inspiration"
	// SpanExpendHeroicInspiration wraps one ExpendHeroicInspiration.Expend call.
	SpanExpendHeroicInspiration = "srd5e.character.expend_heroic_inspiration"
)

// Attribute values of the Heroic Inspiration use cases.
const (
	ValueOperationGain   = "gain"
	ValueOperationExpend = "expend"

	ValueResultGained      = "gained"
	ValueResultExcessLost  = "excess_lost"
	ValueResultExcessGiven = "excess_given"
	ValueResultExpended    = "expended"

	ValueRejectionNotInspired = "not_inspired"
)

// Span of the Topple save use case. It records on the shared MetricOperations
// counter and MetricDuration histogram.
const (
	// SpanResolveToppleSave wraps one ToppleSaveResolver.ResolveToppleSave call.
	SpanResolveToppleSave = "srd5e.character.resolve_topple_save"
)

// Attribute values of the Topple save use case.
const (
	ValueOperationResolveToppleSave = "resolve_topple_save"

	ValueResultTargetProne    = "target_prone"
	ValueResultTargetResisted = "target_resisted"
)

// Signals of the Influence use case ResolveInfluence. It owns its counters and
// duration histogram (it is the outermost use case and records its own errors).
const (
	// SpanResolveInfluence wraps one InfluenceResolver.ResolveInfluence call.
	SpanResolveInfluence = "srd5e.character.resolve_influence"
	// MetricInfluenceResolved counts completed Influence calls ({influence}).
	MetricInfluenceResolved = "srd5e.character.influence_resolved"
	// MetricInfluenceErrors counts port failures leaving ResolveInfluence ({error}).
	MetricInfluenceErrors = "srd5e.character.influence_errors"
	// MetricInfluenceDuration records ResolveInfluence durations (s), failed calls included.
	MetricInfluenceDuration = "srd5e.character.influence_duration"
)

// Attribute keys of the Influence use case. The shared keys
// telemetry.AttrOutcome, telemetry.AttrErrorType and d20.AttrRollMode are used
// as they are.
const (
	// AttrInfluenceApproach is the InfluenceApproach.String of a hesitant call.
	AttrInfluenceApproach = "srd5e.influence.approach"
	// AttrInfluenceAttitude is the Attitude.String of a hesitant call.
	AttrInfluenceAttitude = "srd5e.influence.attitude"
	// AttrInfluenceResponse is the result variant: a ValueInfluence* value.
	AttrInfluenceResponse = "srd5e.influence.response"
)

// Attribute values of AttrInfluenceResponse. Both failed-check variants
// (rolled and forced) map to ValueInfluenceCheckFailed.
const (
	ValueInfluenceFulfilled      = "fulfilled"
	ValueInfluenceRefused        = "refused"
	ValueInfluenceCheckSucceeded = "check_succeeded"
	ValueInfluenceCheckFailed    = "check_failed"
)

// Spans and metrics of the spellcasting use cases. The use cases CalculateSpellcasting,
// CreateSpellcasting, ExpendSpellSlot, RegainSpellSlot and AdvanceSpellcasting
// record on the shared MetricOperations counter; all eight record on the shared
// MetricDuration histogram.
const (
	// SpanCalculateSpellcasting wraps one CalculateSpellcasting.Do call.
	SpanCalculateSpellcasting = "srd5e.character.calculate_spellcasting"
	// SpanCreateSpellcasting wraps one CreateSpellcasting.Do call.
	SpanCreateSpellcasting = "srd5e.character.create_spellcasting"
	// SpanChooseAdditionalSpells wraps one ChooseAdditionalSpells.Do call.
	SpanChooseAdditionalSpells = "srd5e.character.choose_additional_spells"
	// SpanChangePreparedSpells wraps one ChangePreparedSpells.Do call.
	SpanChangePreparedSpells = "srd5e.character.change_prepared_spells"
	// SpanExpendSpellSlot wraps one ExpendSpellSlot.Do call.
	SpanExpendSpellSlot = "srd5e.character.expend_spell_slot"
	// SpanRegainSpellSlot wraps one RegainSpellSlot.Do call.
	SpanRegainSpellSlot = "srd5e.character.regain_spell_slot"
	// SpanRestoreSpellSlots wraps one RestoreSpellSlots.Do call.
	SpanRestoreSpellSlots = "srd5e.character.restore_spell_slots"
	// SpanAdvanceSpellcasting wraps one AdvanceSpellcasting.Do call.
	SpanAdvanceSpellcasting = "srd5e.character.advance_spellcasting"
	// MetricPreparedSpellChanges counts ChooseAdditionalSpells and ChangePreparedSpells calls ({change}).
	MetricPreparedSpellChanges = "srd5e.character.prepared_spell_changes"
	// MetricSpellSlotsRestored counts RestoreSpellSlots calls ({rest}).
	MetricSpellSlotsRestored = "srd5e.character.spell_slots_restored"
	// MetricSpellSlotLevel records the level of each slot expended by ExpendSpellSlot ({level}).
	MetricSpellSlotLevel = "srd5e.character.spell_slot_level"
)

// AttrPool is the slot pool of a slot operation: a ValuePool* value.
const AttrPool = "srd5e.character.pool"

// Attribute values of the spellcasting use cases. The rejection values are the
// String of PreparationRejectionKind, SlotOperationRejection and
// SpellcastingAdvanceRejection; the class value is SpellcastingClass.String.
const (
	ValueOperationCalculateSpellcasting  = "calculate_spellcasting"
	ValueOperationCreateSpellcasting     = "create_spellcasting"
	ValueOperationChooseAdditionalSpells = "choose_additional_spells"
	ValueOperationChangePreparedSpells   = "change_prepared_spells"
	ValueOperationExpendSpellSlot        = "expend_spell_slot"
	ValueOperationRegainSpellSlot        = "regain_spell_slot"
	ValueOperationRestoreSpellSlots      = "restore_spell_slots"
	ValueOperationAdvanceSpellcasting    = "advance_spellcasting"

	ValueResultCalculated = "calculated"
	ValueResultCreated    = "created"
	ValueResultRegained   = "regained"
	ValueResultRestored   = "restored"
	ValueResultAdvanced   = "advanced"

	ValuePoolSpellcasting = "spellcasting"
	ValuePoolPactMagic    = "pact_magic"
)

// Spans of the level advancement use cases PreviewLevelAdvancement and
// ChooseSubclass. They record on the shared MetricOperations counter and
// MetricDuration histogram.
const (
	// SpanAdvancementPreview wraps one PreviewLevelAdvancement.Run call.
	SpanAdvancementPreview = "srd5e.character.advancement.preview"
	// SpanAdvancementChooseSubclass wraps one ChooseSubclass.Run call.
	SpanAdvancementChooseSubclass = "srd5e.character.advancement.choose_subclass"
	// SpanAdvancementLoadContent wraps the content lookups of one level advancement call.
	SpanAdvancementLoadContent = "srd5e.character.advancement.load_content"
)

// Attribute values of the level advancement use cases. The preview rejection
// values are those of LevelUpRejection (maximum_level, insufficient_experience,
// multiclass_prerequisites_not_met); the subclass rejection values are the
// String of SubclassRejectionReason. ValueResultUnavailable is shared with the
// magic item use cases.
const (
	ValueOperationPreview        = "preview"
	ValueOperationChooseSubclass = "choose_subclass"

	ValueResultAvailable = "available"
	ValueResultChosen    = "chosen"

	ValueRejectionNotInClass         = "not_in_class"
	ValueRejectionBelowSubclassLevel = "below_subclass_level"
	ValueRejectionAlreadyChosen      = "already_chosen"
	ValueRejectionUnknownSubclass    = "unknown_subclass"
)

// Spans and metrics of the casting use cases CastSpell, CompleteCasting,
// BreakConcentration and BeginConcentration. All four record on the shared
// MetricDuration histogram; BreakConcentration and BeginConcentration record on the
// shared MetricOperations counter, CastSpell and CompleteCasting on MetricSpellsCast
// and, for a slot spell, on MetricSpellSlotLevel.
const (
	// SpanCastSpell wraps one CastSpell.Do call.
	SpanCastSpell = "srd5e.character.cast_spell"
	// SpanCompleteCasting wraps one CompleteCasting.Do call.
	SpanCompleteCasting = "srd5e.character.complete_casting"
	// SpanBreakConcentration wraps one BreakConcentration.Do call.
	SpanBreakConcentration = "srd5e.character.break_concentration"
	// SpanBeginConcentration wraps one BeginConcentration.Do call.
	SpanBeginConcentration = "srd5e.character.begin_concentration"
	// MetricSpellsCast counts CastSpell and CompleteCasting calls ({cast}).
	MetricSpellsCast = "srd5e.character.spells_cast"
	// LogSpellCastResolved is the FINE log event of one CastSpell call.
	LogSpellCastResolved = "srd5e.character.spell_cast_resolved"
)

// Attribute keys of the casting use cases.
const (
	// AttrCastSource is the CastSource.String of a cast: a ValueCastSource* value.
	AttrCastSource = "srd5e.character.cast_source"
	// AttrConcentrationReason is the ConcentrationBreakReason.String of a BreakConcentration call.
	AttrConcentrationReason = "srd5e.character.concentration_reason"
)

// Attribute values of the casting use cases. The rejection values are the String
// of CastRejectionKind (the first reason), the cast source values are the String
// of CastSource and the concentration reasons the String of
// ConcentrationBreakReason.
const (
	ValueOperationCastSpell          = "cast_spell"
	ValueOperationCompleteCasting    = "complete_casting"
	ValueOperationBreakConcentration = "break_concentration"
	ValueOperationBeginConcentration = "begin_concentration"

	ValueResultCast          = "cast"
	ValueResultStarted       = "started"
	ValueResultEnded         = "ended"
	ValueResultBegan         = "began"
	ValueResultNone          = "none"
	ValueResultCastingFailed = "casting_failed"
)
