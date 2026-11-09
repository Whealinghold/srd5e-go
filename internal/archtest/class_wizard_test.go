package archtest_test

import "testing"

// wizardFiles are the files of package character that hold the Wizard code.
var wizardFiles = []string{
	"wizard_spell_facts", "wizard_rules", "wizard_rejection", "wizard_choices", "wizard_spellbook", "wizard_state", "wizard_features",
	"wizard_event", "wizard_signals", "wizard_usecase", "use_wizard_choices", "use_wizard_spellbook", "use_wizard_spells",
}

// Requirement: doc/specs/class-wizard.md FR-108 - the Wizard files define none of the types other specs own (RestApplied, Spellbook, ClassLevels,
// FeatureStates, SubclassChosen, SpellCast), declare the named services, use cases and results, no generic variant name (Applied, Rejected,
// ContentMissing, NotInSpellbook), and none of their exported names equals an export of another package of the module.
// Path: unhappy (name collision).
func TestWizardNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range wizardFiles {
		files["character/"+f+".go"] = true
	}
	var own map[string]bool
	others := map[string][]string{}
	for _, pkg := range m.Packages {
		switch pkg.Dir {
		case "character":
			own = exportedNames(pkg, files)
		case "content", "d20", "ability", "dice", "modifier", "srderr", "telemetry":
			for n := range exportedNames(pkg, nil) {
				others[n] = append(others[n], pkg.Dir)
			}
		}
	}
	if len(own) < 150 {
		t.Fatalf("only %d exported names found in the Wizard files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"RestApplied", "Spellbook", "ClassLevels", "FeatureStates", "SubclassChosen", "SpellCast", "RitualAdeptNotInSpellbookRejected"} {
		if own[owned] {
			t.Errorf("the Wizard files define %s, which another spec owns", owned)
		}
	}
	for _, generic := range []string{"Applied", "Rejected", "ContentMissing", "NotInSpellbook", "NotAWizard", "Accepted", "Available", "Unavailable"} {
		if own[generic] {
			t.Errorf("the Wizard files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"WizardClassID", "WizardSpellFacts", "WizardRules", "SpellbookRules", "ArcaneRecovery", "WizardResources", "WizardResourcesKey", "WizardCreationChoices", "SpellMastery",
		"SignatureSpells", "WizardAlwaysPrepared", "WizardEvent", "SpellbookChanged", "ChooseWizardSkills", "ChooseWizardEquipment", "ChooseWizardCantrips", "ChooseScholarExpertise",
		"StartSpellbook", "AddLevelUpSpells", "CopySpellIntoBook", "CopySpellbook", "ReplaceLostSpellbook", "UseArcaneRecovery", "MemorizeSpell", "CheckRitualAdept", "ChooseSpellMastery",
		"ReplaceSpellMastery", "ChooseSignatureSpells", "StartSpellbookNotAWizard", "UseArcaneRecoveryContentMissing", "SpellbookAlreadyInSpellbook", "ArcaneRecoveryAlreadyUsed",
	} {
		if !own[want] {
			t.Errorf("the Wizard files declare no %s", want)
		}
	}
}
