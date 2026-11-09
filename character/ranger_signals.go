package character

// Spans and attribute values of the Ranger use cases. They record on the shared class feature instruments MetricFeatureUsed,
// MetricFeatureUsedDuration and MetricFeatureUsedErrors (see feature_signals.go), which every class reuses. The rejection values
// are the String of RangerReason.
const (
	// SpanChooseRangerOption wraps one ChooseRangerOption.Run call.
	SpanChooseRangerOption = "srd5e.character.choose_ranger_option"
	// SpanResolveRangerFeatures wraps one ResolveRangerFeatures.Run call.
	SpanResolveRangerFeatures = "srd5e.character.resolve_ranger_features"
	// SpanUseTireless wraps one UseTireless.Run call.
	SpanUseTireless = "srd5e.character.use_tireless"
)

// Attribute values of the Ranger use cases. ChooseRangerOption reports the feature chosen; the Fighter's weapon_mastery and
// fighting_style values are reused for the same features.
const (
	ValueClassRanger = "ranger"

	ValueFeatureTireless            = "tireless"
	ValueFeatureRangerFeatures      = "ranger_features"
	ValueFeatureDeftExplorer        = "deft_explorer"
	ValueFeatureRangerExpertise     = "expertise"
	ValueFeatureHuntersPrey         = "hunters_prey"
	ValueFeatureDefensiveTactics    = "defensive_tactics"
	ValueFeatureRangerFeatGrant     = "feat_grant"
	ValueFeatureRangerWeaponMastery = ValueFeatureWeaponMastery
	ValueFeatureRangerFightingStyle = ValueFeatureFightingStyle
)
