package character

// Spans and the attribute values of the Monk use cases. They record on the shared class
// feature instruments MetricFeatureUsed, MetricFeatureUsedDuration and
// MetricFeatureUsedErrors (see feature_signals.go), which every class reuses, and on the log
// event LogClassFeatureResolved. The rejection values are the String of MonkReason.
const (
	// SpanUseFocusFeature wraps one UseFocusFeature.Run call.
	SpanUseFocusFeature = "srd5e.character.use_focus_feature"
	// SpanUseUncannyMetabolism wraps one UseUncannyMetabolism.Run call.
	SpanUseUncannyMetabolism = "srd5e.character.use_uncanny_metabolism"
	// SpanResolveDeflectAttacks wraps one ResolveDeflectAttacks.Run call.
	SpanResolveDeflectAttacks = "srd5e.character.resolve_deflect_attacks"
	// SpanRedirectDeflectedAttack wraps one RedirectDeflectedAttack.Run call.
	SpanRedirectDeflectedAttack = "srd5e.character.redirect_deflected_attack"
	// SpanUseWholenessOfBody wraps one UseWholenessOfBody.Run call.
	SpanUseWholenessOfBody = "srd5e.character.use_wholeness_of_body"
	// SpanEndQuiveringPalm wraps one EndQuiveringPalm.Run call.
	SpanEndQuiveringPalm = "srd5e.character.end_quivering_palm"
)

// Attribute values of the Monk use cases. The Focus features are reported under their own
// name: flurry_of_blows, patient_defense or step_of_the_wind (the String of MonkFocusFeature).
const (
	ValueClassMonk = "monk"

	ValueFeatureFlurryOfBlows     = "flurry_of_blows"
	ValueFeaturePatientDefense    = "patient_defense"
	ValueFeatureStepOfTheWind     = "step_of_the_wind"
	ValueFeatureUncannyMetabolism = "uncanny_metabolism"
	ValueFeatureDeflectAttacks    = "deflect_attacks"
	ValueFeatureDeflectRedirect   = "deflect_redirect"
	ValueFeatureWholenessOfBody   = "wholeness_of_body"
	ValueFeatureQuiveringPalmEnd  = "quivering_palm_end"
)
