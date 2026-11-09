package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

var paladinDomainFiles = []string{
	"paladin_level", "paladin_reason", "paladin_progression", "paladin_state", "paladin_event", "paladin_rules", "paladin_advance", "paladin_choices", "paladin_creation_choices",
}

var paladinUseCaseFiles = []string{
	"paladin_usecase", "use_lay_on_hands", "use_paladin_channel_divinity", "use_free_cast", "use_holy_nimbus", "use_apply_smite_of_protection", "use_choose_paladin_option", "use_calculate_paladin_features",
}

// Requirement: doc/specs/class-paladin.md FR-123, FR-124 - the Paladin files define none of the types other specs own, declare the named results, services and events with no generic variant name, none of their exported names
// equals an export of another package of the module, and the feature state key is named paladin_state, which no other production file uses.
// Path: unhappy (name collision).
func TestPaladinNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(paladinDomainFiles, paladinUseCaseFiles, []string{"paladin_signals"}) {
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
	if len(own) < 100 {
		t.Fatalf("only %d exported names found in the Paladin files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SubclassChosen", "RestApplied", "MissingContent", "ProficiencyBonus", "ClassLevels", "SpellSaveDC", "SpellcastingState", "SpellSlotPool", "FightingStyleChoice", "WeaponMasteryChoice", "GrantedCantrip", "ContentDecodeFailed"} {
		if own[owned] {
			t.Errorf("the Paladin files define %s, which another spec owns", owned)
		}
	}
	for _, generic := range []string{"Applied", "Rejected", "NoEffect", "NotAPaladin", "ContentMissing", "ClassDataInvalid", "Available", "Spent", "Accepted", "Summary", "Started"} {
		if own[generic] {
			t.Errorf("the Paladin files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"PaladinLevel", "PaladinProgression", "PaladinState", "PaladinStateKey", "PaladinStateParts", "PaladinCreationChoices", "PaladinReason", "PaladinEvent", "PaladinFeature", "PaladinFightingStyle", "PaladinFightingStyleFeat",
		"BlessedWarrior", "PaladinChoice", "PaladinChoiceRules", "LongRestUse", "FreeCastFeature", "ClassEntry", "PaladinActionCost", "PaladinAttackKind", "DetectionKind", "CoverGrant", "LayOnHandsRules", "LayOnHandsOutcome", "AuraEffects",
		"PaladinAuras", "RadiantStrikes", "DivineSenseEffect", "AbjureFoesEffect", "SacredWeaponEffect", "HolyNimbusEffect", "SmiteOfProtectionEffect", "EndPaladinEffect", "AdvancePaladin", "ApplyPaladinRest", "StartPaladin",
		"AttacksPerAttackAction", "LayOnHandsMaximum", "UseLayOnHands", "UsePaladinChannelDivinity", "UseFreeCast", "UseHolyNimbus", "ApplySmiteOfProtection", "ChoosePaladinOption", "CalculatePaladinFeatures",
		"UseLayOnHandsApplied", "UseLayOnHandsRejected", "UseLayOnHandsNotAPaladin", "UsePaladinChannelDivinityApplied", "UsePaladinChannelDivinityRejected", "UsePaladinChannelDivinityNotAPaladin",
		"UseFreeCastApplied", "UseFreeCastRejected", "UseFreeCastNotAPaladin", "UseHolyNimbusApplied", "UseHolyNimbusRejected", "UseHolyNimbusNotAPaladin", "ApplySmiteOfProtectionApplied",
		"ApplySmiteOfProtectionNoEffect", "ApplySmiteOfProtectionNotAPaladin", "ChoosePaladinOptionAccepted", "ChoosePaladinOptionRejected", "ChoosePaladinOptionNotAPaladin",
		"CalculatePaladinFeaturesSummary", "CalculatePaladinFeaturesNotAPaladin", "LayOnHandsUsed", "PaladinChannelDivinityUsed", "FreeCastSpent", "HolyNimbusActivated", "HolyNimbusRestored",
		"SmiteOfProtectionTriggered", "PaladinEffectEnded", "PaladinChoiceMade", "PaladinLevelOneGained", "PaladinAdvanced", "PaladinResourcesRestored",
	} {
		if !own[want] {
			t.Errorf("the Paladin files declare no %s", want)
		}
	}
	count := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			ast.Inspect(f.AST, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Value == `"paladin_state"` {
					count++
				}
				return true
			})
		}
	}
	if count != 1 {
		t.Errorf("key name paladin_state is used %d times in package character, want 1", count)
	}
}

// Requirement: doc/specs/class-paladin.md FR-108, FR-114, shared-types-and-ownership.md FR-044 - the Paladin domain files import neither telemetry, context, time nor sync, start no goroutine and read no
// ContentSource, so they are pure functions of their arguments (no dice roller either: no Paladin feature rolls dice); the use case files call no other use case (neither a constructor nor a Run).
// Path: unhappy (architecture).
func TestPaladinDomainIsPureAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewTakeFeat", "NewRestTaker", "NewUseSecondWind", "NewUseLayOnHands", "NewUsePaladinChannelDivinity", "NewUseFreeCast",
		"NewUseHolyNimbus", "NewApplySmiteOfProtection", "NewChoosePaladinOption", "NewCalculatePaladinFeatures"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(paladinDomainFiles, name), slices.Contains(paladinUseCaseFiles, name)
			if !domain && !useCase {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if domain && (imp.Path == "context" || imp.Path == "time" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry")) {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
				if strings.HasSuffix(imp.Path, "/dice/randroller") || strings.HasSuffix(imp.Path, "/dice/dicetest") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.Ident:
					if n.Name == "ContentSource" || n.Name == "Roller" {
						t.Errorf("%s names %s", f.Path, n.Name)
					}
				case *ast.CallExpr:
					if id, ok := n.Fun.(*ast.Ident); ok && useCase && slices.Contains(constructors, id.Name) {
						t.Errorf("%s calls %s", f.Path, id.Name)
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && sel.Sel.Name == "Run" {
						t.Errorf("%s calls a Run method", f.Path)
					}
				}
				return true
			})
		}
	}
	if seen != len(paladinDomainFiles)+len(paladinUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(paladinDomainFiles)+len(paladinUseCaseFiles))
	}
}
