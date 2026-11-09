package character

// Spans of the species trait use cases. They record on the shared MetricOperations
// counter and MetricDuration histogram; ReplaceHighElfCantrip looks the spell up in
// the child span SpanLoadContent.
const (
	// SpanExpendSpeciesTraitUse wraps one ExpendSpeciesTraitUse.Run call.
	SpanExpendSpeciesTraitUse = "srd5e.character.expend_species_trait_use"
	// SpanRenewSpeciesTraitUses wraps one RenewSpeciesTraitUses.Run call.
	SpanRenewSpeciesTraitUses = "srd5e.character.renew_species_trait_uses"
	// SpanUseRelentlessEndurance wraps one UseRelentlessEndurance.Run call.
	SpanUseRelentlessEndurance = "srd5e.character.use_relentless_endurance"
	// SpanRollHalflingLuck wraps one RollHalflingLuck.Run call.
	SpanRollHalflingLuck = "srd5e.character.roll_halfling_luck"
	// SpanRollStonesEndurance wraps one RollStonesEndurance.Run call.
	SpanRollStonesEndurance = "srd5e.character.roll_stones_endurance"
	// SpanReplaceHighElfCantrip wraps one ReplaceHighElfCantrip.Run call.
	SpanReplaceHighElfCantrip = "srd5e.character.replace_high_elf_cantrip"
)

// Attribute values of the species trait use cases. The rejection values are the String
// of SpeciesTraitRejection (species_behaviour_unknown to character_mismatch).
// ValueResultChanged, ValueResultResolved, ValueResultNotApplicable, ValueResultRejected
// and ValueResultContentMissing are shared.
const (
	ValueOperationExpendSpeciesTraitUse  = "expend_species_trait_use"
	ValueOperationRenewSpeciesTraitUses  = "renew_species_trait_uses"
	ValueOperationUseRelentlessEndurance = "use_relentless_endurance"
	ValueOperationRollHalflingLuck       = "roll_halfling_luck"
	ValueOperationRollStonesEndurance    = "roll_stones_endurance"
	ValueOperationReplaceHighElfCantrip  = "replace_high_elf_cantrip"
)
