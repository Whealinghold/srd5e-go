package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

var bardDomainFiles = []string{
	"bard_level", "bard_reason", "bard_state", "bard_creation_choices", "bard_event", "bard_inspiration", "bard_features", "bard_choices", "bard_advance",
}

var bardUseCaseFiles = []string{"bard_usecase", "use_bardic_inspiration", "use_cutting_words", "use_peerless_skill", "use_countercharm"}

// Requirement: doc/specs/class-bard.md FR-118, FR-120 - the Bard files define none of the types other specs own, declare the named results, services and events with no generic variant name, none of their exported names equals an export of another package of the
// module, and the feature state key is named bardic_inspiration_pool, which no other production file uses.
// Path: unhappy (name collision).
func TestBardNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(bardDomainFiles, bardUseCaseFiles, []string{"bard_signals"}) {
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
	if len(own) < 60 {
		t.Fatalf("only %d exported names found in the Bard files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SubclassChosen", "RestApplied", "MissingContent", "ProficiencyBonus", "ClassLevels", "SpellSlotPool", "SlotOperationRejection", "SpellcastingState", "FeatTakingRules", "SkillProficiencies"} {
		if own[owned] {
			t.Errorf("the Bard files define %s, which another spec owns", owned)
		}
	}
	for _, generic := range []string{"Granted", "Applied", "Rejected", "Restored", "Rerolled", "LeveledUp", "NotABard", "ContentMissing", "InvalidChoices", "Pool", "Grant", "Plan"} {
		if own[generic] {
			t.Errorf("the Bard files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"BardLevel", "BardicDie", "BardicInspirationPool", "BardicInspirationGrant", "AdjustableD20Result", "AdjustableCheck", "AdjustableAttack", "InspirationRecipient", "BardCreationChoices", "BardMulticlassChoices",
		"BardEvent", "BardicInspirationGranted", "BardicInspirationExpended", "BardicInspirationExpired", "BardicInspirationRestored", "BardExpertiseGained", "GrantBardicInspiration", "RecoverBardicInspiration",
		"ExpendSlotForInspiration", "PlanBardicInspirationUse", "ApplyBardicInspirationDie", "PlanCuttingWords", "ApplyCuttingWords", "PlanPeerlessSkill", "ApplyPeerlessSkill", "PlanCountercharm", "ResolveCountercharm",
		"JackOfAllTrades", "AllowsSecondTarget", "ChooseBardExpertise", "AdvanceBard", "UseBardicInspiration", "UseCuttingWords", "UsePeerlessSkill", "UseCountercharm", "UseBardicInspirationApplied",
		"UseBardicInspirationRejected", "UseCuttingWordsApplied", "UseCuttingWordsRejected", "UsePeerlessSkillApplied", "UsePeerlessSkillRejected", "UseCountercharmRerolled", "UseCountercharmRejected",
		"AdvanceBardApplied", "AdvanceBardNotABard",
	} {
		if !own[want] {
			t.Errorf("the Bard files declare no %s", want)
		}
	}
	count := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			ast.Inspect(f.AST, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Value == `"bardic_inspiration_pool"` {
					count++
				}
				return true
			})
		}
	}
	if count != 1 {
		t.Errorf(`the literal "bardic_inspiration_pool" is used %d times in package character, want 1`, count)
	}
}

// Requirement: doc/specs/class-bard.md FR-119, FR-120, shared-types-and-ownership.md FR-044 - the Bard domain files import neither telemetry, context nor sync, start no goroutine, read no ContentSource and read no clock (the time package is used for the Duration
// type only), so they are pure functions of their arguments; the use case files call no other use case, neither a constructor nor a Run.
// Path: unhappy (architecture).
func TestBardDomainIsPureAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewTakeFeat", "NewRestTaker", "NewUseSecondWind", "NewResolver"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(bardDomainFiles, name), slices.Contains(bardUseCaseFiles, name)
			if !domain && !useCase {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if domain && (imp.Path == "context" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry")) {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
				if strings.Contains(imp.Path, "combat") || strings.HasSuffix(imp.Path, "test") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.Ident:
					if n.Name == "ContentSource" {
						t.Errorf("%s names ContentSource", f.Path)
					}
				case *ast.CallExpr:
					if id, ok := n.Fun.(*ast.Ident); ok && useCase && slices.Contains(constructors, id.Name) {
						t.Errorf("%s calls %s", f.Path, id.Name)
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
						if pkgID, isPkg := sel.X.(*ast.Ident); isPkg && pkgID.Name == "time" {
							t.Errorf("%s uses time.%s", f.Path, sel.Sel.Name)
						}
						if useCase && sel.Sel.Name == "Run" {
							t.Errorf("%s calls a Run method", f.Path)
						}
					}
				}
				return true
			})
		}
	}
	if seen != len(bardDomainFiles)+len(bardUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(bardDomainFiles)+len(bardUseCaseFiles))
	}
}
