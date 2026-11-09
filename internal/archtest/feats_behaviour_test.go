package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// featsDomainFiles are the files of package character that hold the feat rules;
// none imports telemetry or context. The use case files hold every telemetry call.
var featsDomainFiles = []string{"feat_choices", "feat_grant", "feat_shape", "feat_rejection", "feat_event", "feat_uses", "taken_feats", "feat_qualification",
	"feat_taking", "feat_magic_initiate", "feat_replacement", "feat_free_casting", "feat_effect", "bonus_feats"}

var featsUseCaseFiles = []string{"take_feat_usecase", "change_magic_initiate_spell_usecase", "expend_feat_use_usecase", "renew_feat_uses_usecase", "roll_free_casting_usecase"}

// featsRequiredNames are types the spec requires package character to export (FR-080, FR-081, section 6).
var featsRequiredNames = []string{"TakenFeats", "TakenFeat", "FeatSource", "BackgroundFeat", "CustomBackgroundFeat", "SpeciesTraitFeat", "ClassFeatureFeat", "EldritchInvocationFeat",
	"BonusFeat", "FeatGrant", "FeatGrantScope", "SpecifiedFeat", "FeatFromCategory", "AnyFeat", "RequiredSpellList", "HeldFeatures", "ProficiencyGrants", "FeatGrantedSpells",
	"FeatUseState", "FeatRenewal", "FeatChoices", "NoFeatChoices", "SingleAbilityIncrease", "AbilityScoreImprovementChoices", "MagicInitiateChoices", "SkilledChoices",
	"FightingStyleChoice", "FeatChoiceProblem", "FeatChoicesError", "FeatQualification", "FeatTakingRules", "FeatTaking", "FeatEffects", "FeatUses", "FeatReplacement",
	"MagicInitiateSpellChange", "FeatEffect", "FeatDamageTypeRule", "FeatUsageLimit", "FeatEvent", "FeatTaken", "FeatSpellReplaced", "FeatUseExpended", "FeatUsesRenewed",
	"FeatReplaced", "FeatRejection", "FreeCastingRule", "GreatWeaponFighting", "BonusFeats", "TakeFeat", "TakeFeatResult", "TakeFeatSucceeded", "TakeFeatRejected",
	"TakeFeatContentMissing", "ChangeMagicInitiateSpell", "ChangeMagicInitiateSpellResult", "ExpendFeatUse", "ExpendFeatUseResult", "RenewFeatUses", "RenewFeatUsesResult",
	"RollFreeCasting", "RollFreeCastingResult", "RollFreeCastingResolved", "RollFreeCastingNotApplicable", "RollFreeCastingRejected"}

// featsForbiddenNames are generic variant names and types other specs own (FR-080).
var featsForbiddenNames = []string{"Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound", "BardFeatChoice", "FeatChoiceSlot", "FightingStyleClassOption"}

// Requirement: doc/specs/feats-behaviour.md FR-071, FR-081 and section 8 "architecture" - the feat domain files import neither telemetry nor context nor sync nor time, start no goroutine, call no use case and declare no type alias; the use case files call no other use case.
// Path: unhappy (architecture).
func TestFeatsDomainIsSilentAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewCalculateDerivedStats", "NewRestTaker", "NewHitDieSpender", "NewGainHeroicInspiration",
		"NewExpendHeroicInspiration", "NewPreviewLevelAdvancement", "NewChooseSubclass", "NewTakeFeat", "NewChangeMagicInitiateSpell", "NewExpendFeatUse", "NewRenewFeatUses", "NewRollFreeCasting"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(featsDomainFiles, name), slices.Contains(featsUseCaseFiles, name)
			if !domain && !useCase {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if domain && (imp.Path == "context" || imp.Path == "time" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry")) {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.TypeSpec:
					if n.Assign.IsValid() {
						t.Errorf("%s: type alias %s", f.Path, n.Name.Name)
					}
				case *ast.CallExpr:
					if id, ok := n.Fun.(*ast.Ident); ok && useCase && slices.Contains(constructors, id.Name) {
						t.Errorf("%s calls %s", f.Path, id.Name)
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && (sel.Sel.Name == "Run" || sel.Sel.Name == "Do") {
						t.Errorf("%s calls a %s method", f.Path, sel.Sel.Name)
					}
				}
				return true
			})
		}
	}
	if want := len(featsDomainFiles) + len(featsUseCaseFiles); seen != want {
		t.Errorf("found %d of %d files", seen, want)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-080 and section 6 "Exports" - package character exports every type of the spec, none of the feat files exports a name another package of the module also exports, and none uses a generic variant name or a type another spec owns.
// Path: unhappy (name collision).
func TestFeatsExportsAreOwnedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(featsDomainFiles, featsUseCaseFiles) {
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
	for _, n := range featsRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s", n)
		}
	}
	for _, n := range featsForbiddenNames {
		if own[n] {
			t.Errorf("character exports %s, a generic name or a type another spec owns", n)
		}
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}
