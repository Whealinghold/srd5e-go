package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// restDomainFiles are the files of package character that hold the rest and Hit
// Point Dice domain code; the two use case files and signals.go are not among them.
var restDomainFiles = []string{
	"rest_kind", "long_rest_spacing", "rest_start_check", "rest_resolution", "rest_benefits",
	"hit_points_regained", "spent_hit_dice", "rest_event",
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-059 - no exported name of the rest
// and Hit Point Dice files (domain and use cases) equals an export of another
// package of the module, and no RestKind member is spelled short or long.
// Path: unhappy (name collision).
func TestRestNamesAreUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{"character/take_rest.go": true, "character/spend_hit_point_die.go": true}
	for _, f := range restDomainFiles {
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
		t.Fatalf("only %d exported names found in the rest files", len(own))
	}
	for n := range own {
		// RollRangeError is declared in content too, in another package and for another
		// rule; the spec names it in both places.
		if n == "RollRangeError" {
			continue
		}
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, bad := range []string{"short", "long", "Short", "Long"} {
		if own[bad] {
			t.Errorf("character exports %s, a RestKind member spelled like the wire form", bad)
		}
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-062 - the rest and Hit Point Dice
// domain files import neither telemetry nor context and start no goroutine
// (observability.md section 3, architecture.md section 3).
// Path: unhappy (architecture).
func TestRestDomainFilesAreSilentAndPure(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			if !slices.Contains(restDomainFiles, name) {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if imp.Path == "context" || imp.Path == "time" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				if g, ok := n.(*ast.GoStmt); ok {
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(g.Pos()))
				}
				return true
			})
		}
	}
	if seen != len(restDomainFiles) {
		t.Errorf("found %d of %d domain files", seen, len(restDomainFiles))
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-057, shared-types-and-ownership.md
// FR-044 - the rest use case files call no other use case: no Run method and no
// constructor of another use case.
// Path: unhappy (nested telemetry).
func TestRestUseCasesDoNotCallEachOther(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{"character/take_rest.go": true, "character/spend_hit_point_die.go": true}
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewCalculateDerivedStats", "NewRestTaker", "NewHitDieSpender"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			if !files[f.Path] {
				continue
			}
			seen++
			ast.Inspect(f.AST, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch fn := call.Fun.(type) {
				case *ast.Ident:
					if slices.Contains(constructors, fn.Name) {
						t.Errorf("%s calls %s", f.Path, fn.Name)
					}
				case *ast.SelectorExpr:
					if fn.Sel.Name == "Run" || fn.Sel.Name == "TakeRest" || fn.Sel.Name == "SpendHitPointDie" {
						t.Errorf("%s calls a %s method", f.Path, fn.Sel.Name)
					}
				}
				return true
			})
		}
	}
	if seen != len(files) {
		t.Errorf("found %d of %d use case files", seen, len(files))
	}
}
