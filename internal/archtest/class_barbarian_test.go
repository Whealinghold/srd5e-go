package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// barbarianDomainFiles are the files of package character that hold the Barbarian domain code; the use case files, feature_signals.go and
// barbarian_signals.go are not among them.
var barbarianDomainFiles = []string{
	"barbarian_level", "barbarian_reason", "barbarian_progression", "barbarian_state", "barbarian_turn", "barbarian_rage", "barbarian_effects",
	"barbarian_attack_rules", "barbarian_berserker", "barbarian_relentless_rage", "barbarian_choice_rules", "barbarian_event", "barbarian_usecase",
}

var barbarianUseCaseFiles = []string{"use_relentless_rage", "apply_barbarian_option_choice"}

// Requirement: doc/specs/class-barbarian.md FR-105, FR-106, FR-107, FR-123 - the Barbarian files define none of the types other specs own (SubclassChosen, WeaponMasteryChoice, WeaponMasteryAccess, RestApplied, MissingContent, Roll,
// ProficiencyBonus, ClassLevels, AttackFacts), declare the named domain-service and result types and no generic variant name, and none of their exported names equals an export of another package of the module; the Rage variants carry
// the operation prefix.
// Path: unhappy (name collision).
func TestBarbarianNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(barbarianDomainFiles, barbarianUseCaseFiles) {
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
		t.Fatalf("only %d exported names found in the Barbarian files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SubclassChosen", "WeaponMasteryChoice", "WeaponMasteryAccess", "RestApplied", "MissingContent", "Roll", "ProficiencyBonus", "ClassLevels", "AttackFacts", "TakenFeats"} {
		if own[owned] {
			t.Errorf("the Barbarian files define %s, which another spec owns", owned)
		}
	}
	for _, generic := range []string{"Available", "Unavailable", "Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound", "Active", "Inactive", "TurnEnded", "RageState"} {
		if own[generic] {
			t.Errorf("the Barbarian files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"BarbarianRage", "RageEntry", "RageEntered", "RageNotEntered", "RageEndCause", "RageTurnEnded", "RageHeavyArmorDonned", "RageBecameIncapacitated", "RageBecameUnconscious", "RageElapsedTime",
		"BarbarianEffects", "BarbarianAttackFacts", "BarbarianHostFacts", "BarbarianTurnState", "BarbarianAttackRules", "BerserkerRules", "RelentlessRage", "PersistentRage", "BarbarianEvent",
		"BrutalStrikeEffect", "RelentlessRageAvailability", "RelentlessRageResolution", "BarbarianChoice", "BarbarianChoiceRules", "UseRelentlessRage", "ApplyBarbarianOptionChoice",
	} {
		if !own[want] {
			t.Errorf("the Barbarian files declare no %s", want)
		}
	}
}

// Requirement: doc/specs/class-barbarian.md FR-038, FR-105, FR-111 - the Barbarian domain files import neither telemetry, context nor sync, read no clock (no call of time.Now, time.Since or time.Until), start no goroutine and import no
// combat package, so they are pure functions of their arguments; the two use case files call no other use case (neither a constructor nor a Run).
// Path: unhappy (architecture).
func TestBarbarianDomainIsPureAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewUseSecondWind", "NewUseTacticalMind", "NewUseIndomitable", "NewUseActionSurge", "NewApplyFighterOptionChoice", "NewUseRelentlessRage", "NewApplyBarbarianOptionChoice"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(barbarianDomainFiles, name), slices.Contains(barbarianUseCaseFiles, name)
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
	if seen != len(barbarianDomainFiles)+len(barbarianUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(barbarianDomainFiles)+len(barbarianUseCaseFiles))
	}
}
