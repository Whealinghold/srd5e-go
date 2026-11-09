package character

// Spans and attribute values of the Cleric use cases. They record on the shared class feature
// instruments MetricFeatureUsed, MetricFeatureUsedDuration and MetricFeatureUsedErrors (see
// feature_signals.go), which every class reuses. The rejection values are the String of ClericReason.
const (
	// SpanUseClericChannelDivinity wraps one UseClericChannelDivinity.Run call, a dice roll included.
	SpanUseClericChannelDivinity = "srd5e.character.cleric.use_channel_divinity"
	// SpanUseDivineIntervention wraps one UseDivineIntervention.Run call, a recharge roll included.
	SpanUseDivineIntervention = "srd5e.character.cleric.use_divine_intervention"
)

// Attribute values of the Cleric use cases.
const (
	ValueClassCleric = "cleric"

	ValueFeatureDivineSpark        = "divine_spark"
	ValueFeatureTurnUndead         = "turn_undead"
	ValueFeaturePreserveLife       = "preserve_life"
	ValueFeatureDivineIntervention = "divine_intervention"
)
