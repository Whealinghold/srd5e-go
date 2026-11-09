package character

// Spans and attribute values of the Sorcerer use cases. They record on the shared class feature instruments
// MetricFeatureUsed, MetricFeatureUsedDuration and MetricFeatureUsedErrors (see feature_signals.go), which every class
// reuses, and look the class up in the child span SpanLoadContent. The rejection values are the String of
// SorcererRejectionKind, plus ValueRejectionContentMissing.
const (
	// SpanChooseMetamagicOptions wraps one ChooseMetamagicOptions.Run call.
	SpanChooseMetamagicOptions = "srd5e.character.choose_metamagic_options"
	// SpanApplySorcererLevelUp wraps one ApplySorcererLevelUp.Run call.
	SpanApplySorcererLevelUp = "srd5e.character.apply_sorcerer_level_up"
	// SpanCreateSpellSlot wraps one CreateSpellSlot.Run call.
	SpanCreateSpellSlot = "srd5e.character.create_spell_slot"
	// SpanUseMetamagic wraps one UseMetamagic.Run call.
	SpanUseMetamagic = "srd5e.character.use_metamagic"
)

// Attribute values of the Sorcerer use cases.
const (
	ValueClassSorcerer = "sorcerer"

	ValueFeatureMetamagic       = "metamagic"
	ValueFeatureFontOfMagic     = "font_of_magic"
	ValueFeatureSorcererLevelUp = "sorcerer_level_up"
)
