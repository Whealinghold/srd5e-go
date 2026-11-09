package character

// Spans of the magic item use cases. They record on the shared MetricOperations
// counter and MetricDuration histogram; PlanMagicItemCrafting looks content up in
// the child span SpanLoadContent.
const (
	// SpanAttuneToMagicItem wraps one AttuneToMagicItem.Do call.
	SpanAttuneToMagicItem = "srd5e.character.attune_magic_item"
	// SpanEndMagicItemAttunement wraps one EndMagicItemAttunement.Do call.
	SpanEndMagicItemAttunement = "srd5e.character.end_magic_item_attunement"
	// SpanRechargeMagicItem wraps one RechargeMagicItem.Do call.
	SpanRechargeMagicItem = "srd5e.character.recharge_magic_item"
	// SpanMixPotions wraps one RollPotionMiscibility.Do call.
	SpanMixPotions = "srd5e.character.mix_potions"
	// SpanCheckCraftingMaterials wraps one CheckCraftingMaterials.Do call.
	SpanCheckCraftingMaterials = "srd5e.character.check_crafting_materials"
	// SpanPlanMagicItemCrafting wraps one PlanMagicItemCrafting.Do call.
	SpanPlanMagicItemCrafting = "srd5e.character.plan_magic_item_crafting"
	// SpanGenerateSentientTraits wraps one GenerateSentientItemTraits.Do call.
	SpanGenerateSentientTraits = "srd5e.character.generate_sentient_traits"
	// SpanResolveSentientConflict wraps one ResolveSentientItemConflict.Do call.
	SpanResolveSentientConflict = "srd5e.character.resolve_sentient_conflict"
	// SpanAttemptSentientControl wraps one AttemptSentientItemControl.Do call.
	SpanAttemptSentientControl = "srd5e.character.attempt_sentient_control"
	// SpanRepeatSentientControlSave wraps one RepeatSentientControlSave.Do call.
	SpanRepeatSentientControlSave = "srd5e.character.repeat_sentient_control_save"
	// SpanRollPotionOfHealing wraps one RollPotionOfHealing.Do call.
	SpanRollPotionOfHealing = "srd5e.character.roll_potion_healing"
	// SpanResolveLastCharge wraps one ResolveLastCharge.Do call.
	SpanResolveLastCharge = "srd5e.character.resolve_last_charge"
	// SpanRollSpellStoringLevels wraps one RollSpellStoringRingLevels.Do call.
	SpanRollSpellStoringLevels = "srd5e.character.roll_spell_storing_levels"
)

// AttrRechargeMoment is the RechargeMoment.String reported to RechargeMagicItem:
// a span attribute of that use case only.
const AttrRechargeMoment = "srd5e.character.recharge_moment"

// AttrLastChargeConsequence is the LastChargeConsequence.String a ResolveLastCharge call resolved to: a span attribute of that use case only.
const AttrLastChargeConsequence = "srd5e.character.last_charge_consequence"

// Attribute values of the magic item use cases. The rejection values of the
// attunement use cases are the String of AttuneRejection and EndRejection; those
// of crafting the String of ShortfallKind.
const (
	ValueOperationAttuneMagicItem         = "attune_magic_item"
	ValueOperationEndMagicItemAttunement  = "end_magic_item_attunement"
	ValueOperationRechargeMagicItem       = "recharge_magic_item"
	ValueOperationMixPotions              = "mix_potions"
	ValueOperationCheckCraftingMaterials  = "check_crafting_materials"
	ValueOperationPlanMagicItemCrafting   = "plan_magic_item_crafting"
	ValueOperationGenerateSentientTraits  = "generate_sentient_traits"
	ValueOperationResolveSentientConflict = "resolve_sentient_conflict"
	ValueOperationAttemptSentientControl  = "attempt_sentient_control"
	ValueOperationRepeatSentientControl   = "repeat_sentient_control_save"
	ValueOperationRollPotionOfHealing     = "roll_potion_healing"
	ValueOperationResolveLastCharge       = "resolve_last_charge"
	ValueOperationRollSpellStoringLevels  = "roll_spell_storing_levels"

	ValueResultRolled            = "rolled"
	ValueResultPlanned           = "planned"
	ValueResultGenerated         = "generated"
	ValueResultResisted          = "resisted"
	ValueResultDemandsMade       = "demands_made"
	ValueResultUnchanged         = "unchanged"
	ValueResultNotDue            = "not_due"
	ValueResultUnavailable       = "unavailable"
	ValueResultContentIncomplete = "content_incomplete"
	ValueResultResolved          = "resolved"
	ValueResultNotListed         = "not_listed"
	ValueResultRolledElsewhere   = "rolled_elsewhere"

	ValueRejectionTooFewPotions                          = "too_few_potions"
	ValueRejectionRarityUnresolved                       = "rarity_unresolved"
	ValueRejectionPurposeDetailMissing                   = "purpose_detail_missing"
	ValueRejectionUnalignedNotAllowed                    = "unaligned_not_allowed"
	ValueRejectionAlignedPurposeInvalid                  = "aligned_purpose_invalid"
	ValueRejectionAlignedPurposeNeedsNonNeutralAlignment = "aligned_purpose_needs_non_neutral_alignment"
	ValueRejectionDemandsInvalid                         = "demands_invalid"
	ValueRejectionPowerSpent                             = "power_spent"
	ValueRejectionChargesRemain                          = "charges_remain"
)
