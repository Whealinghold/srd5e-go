package character

// Spans and attribute values of the Bard use cases. They record on the shared class feature instruments
// MetricFeatureUsed, MetricFeatureUsedDuration and MetricFeatureUsedErrors (see feature_signals.go), which every class
// reuses. The rejection values are the String of BardReason.
const (
	// SpanUseBardicInspiration wraps one UseBardicInspiration.Run call, a dice roll included.
	SpanUseBardicInspiration = "srd5e.character.bard.use_bardic_inspiration"
	// SpanUseCuttingWords wraps one UseCuttingWords.Run call, a dice roll included.
	SpanUseCuttingWords = "srd5e.character.bard.use_cutting_words"
	// SpanUsePeerlessSkill wraps one UsePeerlessSkill.Run call, a dice roll included.
	SpanUsePeerlessSkill = "srd5e.character.bard.use_peerless_skill"
	// SpanUseCountercharm wraps one UseCountercharm.Run call, the d20 rolls included.
	SpanUseCountercharm = "srd5e.character.bard.use_countercharm"
)

// Attribute values of the Bard use cases.
const (
	ValueClassBard = "bard"

	ValueFeatureBardicInspiration = "bardic_inspiration"
	ValueFeatureCuttingWords      = "cutting_words"
	ValueFeaturePeerlessSkill     = "peerless_skill"
	ValueFeatureCountercharm      = "countercharm"
)
