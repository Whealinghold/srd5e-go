package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// heroicDomainFiles are the files of package character that hold the Heroic
// Inspiration domain code; the two use case files and signals.go are not among them.
var heroicDomainFiles = []string{"excess_inspiration_choice", "resolve_excess_inspiration", "reroll_target", "rerolled_die"}

var heroicUseCaseFiles = []string{"gain_heroic_inspiration_usecase", "expend_heroic_inspiration_usecase"}

// Requirement: doc/specs/heroic-inspiration.md FR-032 - the Heroic Inspiration files
// declare no event type, and none of their exported names equals an export of another
// package of the module.
// Path: unhappy (name collision).
func TestHeroicInspirationDeclaresNoEventAndNamesAreUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(heroicDomainFiles, heroicUseCaseFiles) {
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
	if len(own) < 30 {
		t.Fatalf("only %d exported names found in the Heroic Inspiration files", len(own))
	}
	for n := range own {
		if strings.HasSuffix(n, "Event") {
			t.Errorf("character.%s looks like an event type; the aggregate owns the events", n)
		}
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-038, FR-037 - the Heroic Inspiration
// domain files import neither telemetry nor context and start no goroutine, and the
// two use case files call no other use case.
// Path: unhappy (architecture).
func TestHeroicInspirationDomainIsSilentAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewCalculateDerivedStats", "NewRestTaker", "NewHitDieSpender", "NewGainHeroicInspiration", "NewExpendHeroicInspiration"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(heroicDomainFiles, name), slices.Contains(heroicUseCaseFiles, name)
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
				case *ast.CallExpr:
					if id, ok := n.Fun.(*ast.Ident); ok && useCase && slices.Contains(constructors, id.Name) {
						t.Errorf("%s calls %s", f.Path, id.Name)
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && slices.Contains([]string{"Run", "TakeRest", "SpendHitPointDie", "Gain", "Expend"}, sel.Sel.Name) {
						t.Errorf("%s calls a %s method", f.Path, sel.Sel.Name)
					}
				}
				return true
			})
		}
	}
	if seen != len(heroicDomainFiles)+len(heroicUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(heroicDomainFiles)+len(heroicUseCaseFiles))
	}
}
