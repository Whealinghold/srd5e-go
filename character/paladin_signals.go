package character

// Spans and attribute values of the Paladin use cases. They record on the shared class feature instruments MetricFeatureUsed,
// MetricFeatureUsedDuration and MetricFeatureUsedErrors (see feature_signals.go), which every class reuses. The rejection values
// are the String of PaladinReason.
const (
	// SpanUseLayOnHands wraps one UseLayOnHands.Run call.
	SpanUseLayOnHands = "srd5e.character.paladin.use_lay_on_hands"
	// SpanUsePaladinChannelDivinity wraps one UsePaladinChannelDivinity.Run call.
	SpanUsePaladinChannelDivinity = "srd5e.character.paladin.use_channel_divinity"
	// SpanUseFreeCast wraps one UseFreeCast.Run call.
	SpanUseFreeCast = "srd5e.character.paladin.use_free_cast"
	// SpanUseHolyNimbus wraps one UseHolyNimbus.Run call.
	SpanUseHolyNimbus = "srd5e.character.paladin.use_holy_nimbus"
	// SpanApplySmiteOfProtection wraps one ApplySmiteOfProtection.Run call.
	SpanApplySmiteOfProtection = "srd5e.character.paladin.apply_smite_of_protection"
	// SpanChoosePaladinOption wraps one ChoosePaladinOption.Run call.
	SpanChoosePaladinOption = "srd5e.character.paladin.choose_option"
	// SpanCalculatePaladinFeatures wraps one CalculatePaladinFeatures.Run call.
	SpanCalculatePaladinFeatures = "srd5e.character.paladin.calculate_features"
)

// Attribute values of the Paladin use cases.
const (
	ValueClassPaladin = "paladin"

	ValueFeatureLayOnHands        = "lay_on_hands"
	ValueFeatureDivineSense       = "divine_sense"
	ValueFeatureAbjureFoes        = "abjure_foes"
	ValueFeatureSacredWeapon      = "sacred_weapon"
	ValueFeaturePaladinsSmite     = "paladins_smite"
	ValueFeatureFaithfulSteed     = "faithful_steed"
	ValueFeatureHolyNimbus        = "holy_nimbus"
	ValueFeatureSmiteOfProtection = "smite_of_protection"
	ValueFeatureChooseOption      = "choose_option"
	ValueFeatureCalculateFeatures = "calculate_features"
)
