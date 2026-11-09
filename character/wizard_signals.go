package character

// Spans and attribute values of the Wizard use cases. They record on the shared
// class feature instruments MetricFeatureUsed, MetricFeatureUsedDuration and
// MetricFeatureUsedErrors (see feature_signals.go) and look the class up in the child
// span SpanLoadContent. The rejection values are the String of the rejection Kind
// (the short snake_case variant), plus ValueRejectionContentMissing and
// ValueRejectionNotAWizard.
const (
	SpanChooseWizardSkills    = "srd5e.character.choose_wizard_skills"
	SpanChooseWizardEquipment = "srd5e.character.choose_wizard_equipment"
	SpanChooseWizardCantrips  = "srd5e.character.choose_wizard_cantrips"
	SpanChooseScholar         = "srd5e.character.choose_scholar_expertise"
	SpanStartSpellbook        = "srd5e.character.start_spellbook"
	SpanAddLevelUpSpells      = "srd5e.character.add_level_up_spells"
	SpanCopySpellIntoBook     = "srd5e.character.copy_spell_into_book"
	SpanCopySpellbook         = "srd5e.character.copy_spellbook"
	SpanReplaceLostSpellbook  = "srd5e.character.replace_lost_spellbook"
	SpanUseArcaneRecovery     = "srd5e.character.use_arcane_recovery"
	SpanMemorizeSpell         = "srd5e.character.memorize_spell"
	SpanCheckRitualAdept      = "srd5e.character.check_ritual_adept"
	SpanChooseSpellMastery    = "srd5e.character.choose_spell_mastery"
	SpanReplaceSpellMastery   = "srd5e.character.replace_spell_mastery"
	SpanChooseSignatureSpells = "srd5e.character.choose_signature_spells"
)

// Attribute values of the Wizard use cases.
const (
	ValueClassWizard = "wizard"

	ValueFeatureWizardSkills        = "skills"
	ValueFeatureWizardEquipment     = "equipment"
	ValueFeatureWizardCantrips      = "cantrips"
	ValueFeatureScholar             = "scholar"
	ValueFeatureSpellbookStart      = "spellbook_start"
	ValueFeatureSpellbookLevelUp    = "spellbook_level_up"
	ValueFeatureSpellbookCopyIn     = "spellbook_copy_in"
	ValueFeatureSpellbookCopyBook   = "spellbook_copy_book"
	ValueFeatureSpellbookReplace    = "spellbook_replace"
	ValueFeatureArcaneRecovery      = "arcane_recovery"
	ValueFeatureMemorizeSpell       = "memorize_spell"
	ValueFeatureRitualAdept         = "ritual_adept"
	ValueFeatureSpellMastery        = "spell_mastery"
	ValueFeatureSpellMasteryReplace = "spell_mastery_replace"
	ValueFeatureSignatureSpells     = "signature_spells"

	ValueRejectionNotAWizard = "not_a_wizard"
)
