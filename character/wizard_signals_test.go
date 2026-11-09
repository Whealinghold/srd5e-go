package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-wizard.md section 7, FR-097, FR-098 - the Wizard span names and the srd5e.class, srd5e.feature and rejection values are a public contract: their strings are pinned here, each span appears in
// doc/observability/signals.md and each value in signals.md or doc/rules/observability.md; every rejection Kind has its snake_case name there.
// Path: happy.
func TestWizardSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile("../doc/rules/observability.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ got, want string }{
		{character.SpanChooseWizardSkills, "srd5e.character.choose_wizard_skills"},
		{character.SpanChooseWizardEquipment, "srd5e.character.choose_wizard_equipment"},
		{character.SpanChooseWizardCantrips, "srd5e.character.choose_wizard_cantrips"},
		{character.SpanChooseScholar, "srd5e.character.choose_scholar_expertise"},
		{character.SpanStartSpellbook, "srd5e.character.start_spellbook"},
		{character.SpanAddLevelUpSpells, "srd5e.character.add_level_up_spells"},
		{character.SpanCopySpellIntoBook, "srd5e.character.copy_spell_into_book"},
		{character.SpanCopySpellbook, "srd5e.character.copy_spellbook"},
		{character.SpanReplaceLostSpellbook, "srd5e.character.replace_lost_spellbook"},
		{character.SpanUseArcaneRecovery, "srd5e.character.use_arcane_recovery"},
		{character.SpanMemorizeSpell, "srd5e.character.memorize_spell"},
		{character.SpanCheckRitualAdept, "srd5e.character.check_ritual_adept"},
		{character.SpanChooseSpellMastery, "srd5e.character.choose_spell_mastery"},
		{character.SpanReplaceSpellMastery, "srd5e.character.replace_spell_mastery"},
		{character.SpanChooseSignatureSpells, "srd5e.character.choose_signature_spells"},
		{character.ValueClassWizard, "wizard"},
		{character.ValueFeatureWizardSkills, "skills"},
		{character.ValueFeatureWizardEquipment, "equipment"},
		{character.ValueFeatureWizardCantrips, "cantrips"},
		{character.ValueFeatureScholar, "scholar"},
		{character.ValueFeatureSpellbookStart, "spellbook_start"},
		{character.ValueFeatureSpellbookLevelUp, "spellbook_level_up"},
		{character.ValueFeatureSpellbookCopyIn, "spellbook_copy_in"},
		{character.ValueFeatureSpellbookCopyBook, "spellbook_copy_book"},
		{character.ValueFeatureSpellbookReplace, "spellbook_replace"},
		{character.ValueFeatureArcaneRecovery, "arcane_recovery"},
		{character.ValueFeatureMemorizeSpell, "memorize_spell"},
		{character.ValueFeatureRitualAdept, "ritual_adept"},
		{character.ValueFeatureSpellMastery, "spell_mastery"},
		{character.ValueFeatureSpellMasteryReplace, "spell_mastery_replace"},
		{character.ValueFeatureSignatureSpells, "signature_spells"},
		{character.ValueRejectionNotAWizard, "not_a_wizard"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for _, name := range wzKindNames() {
		if !strings.Contains(string(rules), "`"+name+"`") {
			t.Errorf("rejection value %q is not in observability.md", name)
		}
	}
}

// wzKindNames are the String of every Wizard rejection Kind.
func wzKindNames() []string {
	var out []string
	for k := character.SkillsWrongSkillCount; k <= character.SkillsSkillNotAllowed; k++ {
		out = append(out, k.String())
	}
	out = append(out, character.EquipmentUnknownEquipmentOption.String())
	for k := character.CantripChoiceWrongCantripCount; k <= character.CantripChoiceNotOnWizardList; k++ {
		out = append(out, k.String())
	}
	for k := character.SpellbookFeatureNotAvailable; k <= character.SpellbookNotInSourceSpellbook; k++ {
		out = append(out, k.String())
	}
	for k := character.ArcaneRecoveryNotAfterShortRest; k <= character.ArcaneRecoveryExceedsMaximum; k++ {
		out = append(out, k.String())
	}
	for k := character.ScholarFeatureNotAvailable; k <= character.ScholarNotProficient; k++ {
		out = append(out, k.String())
	}
	for k := character.MemorizeFeatureNotAvailable; k <= character.MemorizeNewSpellAlreadyPrepared; k++ {
		out = append(out, k.String())
	}
	for k := character.SpellMasteryFeatureNotAvailable; k <= character.SpellMasteryCastingTimeNotAction; k++ {
		out = append(out, k.String())
	}
	for k := character.SignatureSpellsFeatureNotAvailable; k <= character.SignatureSpellsNotInSpellbook; k++ {
		out = append(out, k.String())
	}
	for k := character.RitualAdeptSpellbookUnavailable; k <= character.RitualAdeptNotARitualSpell; k++ {
		out = append(out, k.String())
	}
	return out
}
