package character

// Spans, the log event and the attribute values of the Fighter use cases. They
// record on the shared class feature instruments MetricFeatureUsed,
// MetricFeatureUsedDuration and MetricFeatureUsedErrors (see feature_signals.go),
// which every class reuses. The rejection values are the String of FighterReason.
const (
	// SpanUseSecondWind wraps one UseSecondWind.Run call.
	SpanUseSecondWind = "srd5e.character.use_second_wind"
	// SpanUseTacticalMind wraps one UseTacticalMind.Run call.
	SpanUseTacticalMind = "srd5e.character.use_tactical_mind"
	// SpanUseIndomitable wraps one UseIndomitable.Run call.
	SpanUseIndomitable = "srd5e.character.use_indomitable"
	// SpanUseActionSurge wraps one UseActionSurge.Run call.
	SpanUseActionSurge = "srd5e.character.use_action_surge"
	// SpanApplyFighterOption wraps one ApplyFighterOptionChoice.Run call, lookups included.
	SpanApplyFighterOption = "srd5e.character.apply_fighter_option"
	// LogClassFeatureResolved is the FINE log event of one class feature call that
	// ended in a success, a rule rejection or a still-failed result.
	LogClassFeatureResolved = "srd5e.character.class_feature_resolved"
)

// Attribute values of the Fighter use cases.
const (
	ValueClassFighter = "fighter"

	ValueFeatureSecondWind       = "second_wind"
	ValueFeatureTacticalMind     = "tactical_mind"
	ValueFeatureActionSurge      = "action_surge"
	ValueFeatureIndomitable      = "indomitable"
	ValueFeatureFightingStyle    = "fighting_style"
	ValueFeatureWeaponMastery    = "weapon_mastery"
	ValueRejectionContentMissing = "content_missing"
)
