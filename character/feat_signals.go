package character

// Spans of the feat use cases. They record on the shared MetricOperations counter
// and MetricDuration histogram; TakeFeat and ChangeMagicInitiateSpell look content
// up in the child span SpanLoadContent.
const (
	// SpanTakeFeat wraps one TakeFeat.Run call.
	SpanTakeFeat = "srd5e.character.take_feat"
	// SpanChangeMagicInitiateSpell wraps one ChangeMagicInitiateSpell.Run call.
	SpanChangeMagicInitiateSpell = "srd5e.character.change_magic_initiate_spell"
	// SpanExpendFeatUse wraps one ExpendFeatUse.Run call.
	SpanExpendFeatUse = "srd5e.character.expend_feat_use"
	// SpanRenewFeatUses wraps one RenewFeatUses.Run call.
	SpanRenewFeatUses = "srd5e.character.renew_feat_uses"
	// SpanRollFreeCasting wraps one RollFreeCasting.Run call.
	SpanRollFreeCasting = "srd5e.character.roll_free_casting"
)

// Attribute keys of TakeFeat: on its span and its operations counter, not on the
// duration histogram.
const (
	// AttrFeatCategory is the FeatCategory of the feat: a ValueFeatCategory* value; omitted when the feat is not found.
	AttrFeatCategory = "srd5e.character.feat_category"
	// AttrFeatSource is the kind of FeatSource of the grant: a ValueFeatSource* value.
	AttrFeatSource = "srd5e.character.feat_source"
)

// Attribute values of the feat use cases. The rejection values are the String of
// FeatRejection (feat_behaviour_unknown to feat_not_replaceable). ValueResultChanged,
// ValueResultResolved, ValueResultRejected and ValueResultContentMissing are shared.
const (
	ValueOperationTakeFeat                 = "take_feat"
	ValueOperationChangeMagicInitiateSpell = "change_magic_initiate_spell"
	ValueOperationExpendFeatUse            = "expend_feat_use"
	ValueOperationRenewFeatUses            = "renew_feat_uses"
	ValueOperationRollFreeCasting          = "roll_free_casting"

	ValueResultTaken         = "taken"
	ValueResultNotApplicable = "not_applicable"

	ValueFeatCategoryOrigin        = "origin"
	ValueFeatCategoryGeneral       = "general"
	ValueFeatCategoryFightingStyle = "fighting_style"
	ValueFeatCategoryEpicBoon      = "epic_boon"

	ValueFeatSourceBackground         = "background"
	ValueFeatSourceSpeciesTrait       = "species_trait"
	ValueFeatSourceClassFeature       = "class_feature"
	ValueFeatSourceEldritchInvocation = "eldritch_invocation"
	ValueFeatSourceBonusFeat          = "bonus_feat"
)
