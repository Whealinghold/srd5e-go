package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// monkDomainFiles are the files of package character that hold the Monk domain code; the use case files, feature_signals.go and
// monk_signals.go are not among them.
var monkDomainFiles = []string{
	"monk_level", "monk_reason", "monk_progression", "monk_state", "monk_availability", "monk_usecase", "monk_focus", "monk_uncanny", "monk_deflect",
	"monk_stunning", "monk_effects", "monk_defense", "monk_open_hand", "monk_quivering_palm", "monk_event",
}

var monkUseCaseFiles = []string{
	"use_monk", "use_focus_feature", "use_uncanny_metabolism", "use_resolve_deflect_attacks", "use_redirect_deflected_attack", "use_wholeness_of_body", "use_end_quivering_palm",
}

// Requirement: doc/specs/class-monk.md FR-146 - the Monk files define none of the types other specs own (SubclassChosen, WeaponMasteryChoice, RestApplied, MissingContent, Roll, ProficiencyBonus, ClassLevels,
// TakenFeats), declare the named domain-service and result types and no generic variant name (MonkRejected, ContentMissing, Applied, Rejected), and none of their exported names equals an export of
// another package of the module.
// Path: unhappy (name collision).
func TestMonkNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(monkDomainFiles, monkUseCaseFiles) {
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
		t.Fatalf("only %d exported names found in the Monk files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SubclassChosen", "WeaponMasteryChoice", "RestApplied", "MissingContent", "Roll", "ProficiencyBonus", "ClassLevels", "TakenFeats", "MonkRejected", "ContentMissing"} {
		if own[owned] {
			t.Errorf("the Monk files define %s, which another spec owns or which is not exported", owned)
		}
	}
	for _, generic := range []string{"Available", "Unavailable", "Applied", "Rejected", "Succeeded", "Found", "NotFound", "Resolved", "Offered"} {
		if own[generic] {
			t.Errorf("the Monk files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"MonkFocus", "UncannyMetabolism", "PerfectFocus", "DeflectAttacks", "DeflectRedirect", "StunningStrike", "DisciplinedSurvivor", "SuperiorDefense", "OpenHandRules", "WholenessOfBody",
		"QuiveringPalm", "MonkEffects", "MonkAvailability", "MonkEvent", "StunApplied", "StunResisted", "UseFocusFeature", "UseUncannyMetabolism", "ResolveDeflectAttacks", "RedirectDeflectedAttack",
		"UseWholenessOfBody", "EndQuiveringPalm", "FocusFeatureApplied", "FocusFeatureNotAvailable", "UncannyMetabolismApplied", "RedirectApplied", "QuiveringPalmEndApplied",
	} {
		if !own[want] {
			t.Errorf("the Monk files declare no %s", want)
		}
	}
}

// Requirement: doc/specs/class-monk.md FR-127, FR-129, FR-145 - the Monk domain files import neither telemetry, context nor sync, read no clock (no call of time.Now, time.Since or time.Until), start no goroutine and import
// no combat package, so they are pure functions of their arguments; the use case files call no other use case (neither a constructor nor a Run).
// Path: unhappy (architecture).
func TestMonkDomainIsPureAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewUseSecondWind", "NewUseTacticalMind", "NewUseIndomitable", "NewUseActionSurge", "NewApplyFighterOptionChoice", "NewUseRelentlessRage",
		"NewApplyBarbarianOptionChoice", "NewUseFocusFeature", "NewUseUncannyMetabolism", "NewResolveDeflectAttacks", "NewRedirectDeflectedAttack", "NewUseWholenessOfBody", "NewEndQuiveringPalm"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(monkDomainFiles, name), slices.Contains(monkUseCaseFiles, name)
			if !domain && !useCase {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if strings.HasSuffix(imp.Path, "/combat") || (domain && (imp.Path == "context" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry"))) {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.CallExpr:
					if id, ok := n.Fun.(*ast.Ident); ok && useCase && slices.Contains(constructors, id.Name) {
						t.Errorf("%s calls %s", f.Path, id.Name)
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
						if useCase && sel.Sel.Name == "Run" {
							t.Errorf("%s calls a Run method", f.Path)
						}
						if x, isPkg := sel.X.(*ast.Ident); isPkg && x.Name == "time" && (sel.Sel.Name == "Now" || sel.Sel.Name == "Since" || sel.Sel.Name == "Until") {
							t.Errorf("%s reads the clock with time.%s", f.Path, sel.Sel.Name)
						}
					}
				}
				return true
			})
		}
	}
	if seen != len(monkDomainFiles)+len(monkUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(monkDomainFiles)+len(monkUseCaseFiles))
	}
}
