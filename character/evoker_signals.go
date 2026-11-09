package character

// Spans and attribute values of the Evoker use cases. They record on the shared class
// feature instruments like the Wizard ones (wizard_signals.go); the two that load the
// Wizard class also look it up in the child span SpanLoadContent. The rejection values are
// the String of the rejection Kind, plus ValueRejectionNotAWizard, ValueRejectionContentMissing
// and ValueRejectionSubclassUnknown.
const (
	SpanAddEvocationSavantSpells  = "srd5e.character.add_evocation_savant_spells"
	SpanAddEvokerFreeSpell        = "srd5e.character.add_evoker_free_spell"
	SpanResolvePotentCantrip      = "srd5e.character.resolve_potent_cantrip"
	SpanResolveSculptSpells       = "srd5e.character.resolve_sculpt_spells"
	SpanResolveEmpoweredEvocation = "srd5e.character.resolve_empowered_evocation"
	SpanUseOverchannel            = "srd5e.character.use_overchannel"
)

// Attribute values of the Evoker use cases (srd5e.class is ValueClassWizard).
const (
	ValueFeatureEvocationSavant    = "evocation_savant"
	ValueFeatureEvokerFreeSpell    = "evoker_free_spell"
	ValueFeaturePotentCantrip      = "potent_cantrip"
	ValueFeatureSculptSpells       = "sculpt_spells"
	ValueFeatureEmpoweredEvocation = "empowered_evocation"
	ValueFeatureOverchannel        = "overchannel"

	ValueRejectionSubclassUnknown = "subclass_unknown"
)
