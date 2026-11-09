package character

// Spans and attribute values of the Warlock use cases. They record on the shared class feature instruments MetricFeatureUsed,
// MetricFeatureUsedDuration and MetricFeatureUsedErrors (see feature_signals.go), which every class reuses, and look the class up in
// the child span SpanLoadContent. The rejection values are the String of WarlockRejectionKind, plus ValueRejectionContentMissing.
const (
	// SpanChooseWarlockInvocations wraps one ChooseWarlockInvocations.Run call.
	SpanChooseWarlockInvocations = "srd5e.character.choose_warlock_invocations"
	// SpanUseDarkOnesOwnLuck wraps one UseDarkOnesOwnLuck.Run call.
	SpanUseDarkOnesOwnLuck = "srd5e.character.use_dark_ones_own_luck"
	// SpanChooseWarlockMysticArcanum wraps one ChooseWarlockMysticArcanum.Run call.
	SpanChooseWarlockMysticArcanum = "srd5e.character.choose_mystic_arcanum"
	// SpanChooseWarlockFamiliarForm wraps one ChooseWarlockFamiliarForm.Run call.
	SpanChooseWarlockFamiliarForm = "srd5e.character.choose_warlock_familiar_form"
	// SpanConjureBookOfShadows wraps one ConjureBookOfShadows.Run call.
	SpanConjureBookOfShadows = "srd5e.character.conjure_book_of_shadows"
	// SpanBondPactWeapon wraps one BondPactWeapon.Run call.
	SpanBondPactWeapon = "srd5e.character.bond_pact_weapon"
	// SpanCastInvocationSpell wraps one CastInvocationSpell.Run call.
	SpanCastInvocationSpell = "srd5e.character.cast_invocation_spell"
	// SpanUseLifedrinker wraps one UseLifedrinker.Run call.
	SpanUseLifedrinker = "srd5e.character.use_lifedrinker"
)

// Attribute values of the Warlock use cases.
const (
	ValueClassWarlock = "warlock"

	ValueFeatureEldritchInvocations = "eldritch_invocations"
	ValueFeatureDarkOnesOwnLuck     = "dark_ones_own_luck"
	ValueFeatureMysticArcanum       = "mystic_arcanum"
	ValueFeatureFamiliarForm        = "familiar_form"
	ValueFeatureBookOfShadows       = "book_of_shadows"
	ValueFeaturePactWeapon          = "pact_weapon"
	ValueFeatureInvocationSpell     = "invocation_spell"
	ValueFeatureLifedrinker         = "lifedrinker"
)
