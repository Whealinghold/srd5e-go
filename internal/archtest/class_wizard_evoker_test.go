package archtest_test

import "testing"

// evokerFiles are the files of package character that hold the Evoker code.
var evokerFiles = []string{"evoker_rules", "evoker_rejection", "evoker_event", "evoker_signals", "use_evoker_spellbook", "use_evoker_spells"}

// Requirement: doc/specs/class-wizard-evoker.md FR-018, class-wizard.md FR-108 - the Evoker files declare the named rules, use cases and results, no generic variant name (Applied, Rejected, ContentMissing,
// NotAWizard), none of the types other specs own, and none of their exported names equals an export of another package of the module.
// Path: unhappy (name collision).
func TestEvokerNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range evokerFiles {
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
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SpellbookChanged", "SpellsAddedToSpellbook", "WizardResources", "SubclassChosen", "CastSource"} {
		if own[owned] {
			t.Errorf("the Evoker files define %s, which another spec owns", owned)
		}
	}
	for _, generic := range []string{"Applied", "Rejected", "ContentMissing", "NotAWizard", "Accepted", "SubclassUnknown"} {
		if own[generic] {
			t.Errorf("the Evoker files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"EvokerRules", "EvokerGate", "EvokerSubclassID", "OverchannelUsed", "OverchannelRejection", "EvokerSpellbookRejection", "PotentCantripRejection", "SculptRejection", "EmpoweredRejection",
		"AddEvocationSavantSpells", "AddEvokerFreeSpell", "ResolvePotentCantrip", "ResolveSculptSpells", "ResolveEmpoweredEvocation", "UseOverchannel", "AddEvocationSavantSpellsSubclassUnknown",
		"AddEvokerFreeSpellContentMissing", "UseOverchannelNotAWizard", "OverchannelApplied", "SculptedCreatures", "EmpoweredRoll", "PotentCantripHalfDamage", "PotentCantripUnchanged",
	} {
		if !own[want] {
			t.Errorf("the Evoker files declare no %s", want)
		}
	}
}
