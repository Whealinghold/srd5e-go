package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// gridFiles are the production files of grid-positions-and-movement.md: pure domain files, no use case.
func gridFiles() []string {
	return []string{"combat/grid_square.go", "combat/grid_footprint.go", "combat/grid_board.go", "combat/grid_rules.go", "combat/grid_step.go", "combat/grid_path.go"}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-040, section 7 - the grid files are silent, pure domain code: they import only content from this module
// (no telemetry, dice, ports or other context), no context, time, randomness or file package, and name no signal constant.
// Path: unhappy (forbidden edge).
func TestGridFilesAreSilentPureDomainCode(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(gridFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && p != "content" {
				t.Errorf("%s imports %s, which grid-positions-and-movement.md does not allow", f.Path, p)
			}
			if slices.Contains([]string{"context", "math/rand", "math/rand/v2", "time", "os", "sync"}, imp.Path) {
				t.Errorf("%s imports %s: the grid rules are pure (FR-040)", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr")) {
				t.Errorf("%s names the signal constant %s: the spec is silent by design (section 7)", f.Path, id.Name)
			}
			return true
		})
	}
	if seen != len(gridFiles()) {
		t.Fatalf("found %d of %d grid files", seen, len(gridFiles()))
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-001 - every exported GridRules method takes a SquareGrid as its first parameter (the argument that proves the grid is
// played on), and package combat exports RuleOptions and PlayingOnAGrid.
// Path: unhappy (scope).
func TestEveryGridRulesOperationTakesASquareGrid(t *testing.T) {
	t.Parallel()
	methods := 0
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 || !fn.Name.IsExported() {
				continue
			}
			if id, ok := fn.Recv.List[0].Type.(*ast.Ident); !ok || id.Name != "GridRules" {
				continue
			}
			methods++
			first := ""
			if ps := fn.Type.Params.List; len(ps) > 0 {
				if id, ok := ps[0].Type.(*ast.Ident); ok {
					first = id.Name
				}
			}
			if first != "SquareGrid" {
				t.Errorf("GridRules.%s takes %q first, want SquareGrid (FR-001)", fn.Name.Name, first)
			}
		}
	}
	if methods != 7 {
		t.Errorf("GridRules has %d exported methods, want the 7 of the spec (RangeInSquares, RangeInFeet, IsWithinFeet, DistancesFeetFrom, LeavesReach, AssessStep, PlanPath)", methods)
	}
	decl := declared(combatPackage(t))
	for _, name := range []string{"RuleOptions", "PlayingOnAGrid", "SquareGrid", "NoGrid"} {
		if len(decl[name]) != 1 {
			t.Errorf("%s declared in %v, want exactly one file", name, decl[name])
		}
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-004, D-8, plan 5.7 step 7d - the overlap, containment and range tests are arithmetic on the corners, not an enumeration of
// squares: Overlaps, Contains, rangeSquares and squaresBetween hold no loop and never call Squares, so a footprint of 1,000 by 1,000 squares costs the same as one square.
// Path: unhappy (boundary).
func TestGridOverlapAndRangeAreArithmetic(t *testing.T) {
	t.Parallel()
	want := map[string]bool{"Overlaps": false, "Contains": false, "rangeSquares": false, "squaresBetween": false}
	for _, path := range []string{"combat/grid_footprint.go", "combat/grid_rules.go"} {
		for _, d := range movementFile(t, path).Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if _, tracked := want[fn.Name.Name]; !tracked {
				continue
			}
			want[fn.Name.Name] = true
			ast.Inspect(fn, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.ForStmt, *ast.RangeStmt:
					t.Errorf("%s holds a loop: the test must be arithmetic (D-8)", fn.Name.Name)
				case *ast.SelectorExpr:
					if v.Sel.Name == "Squares" {
						t.Errorf("%s calls Squares: the test must not enumerate squares (D-8)", fn.Name.Name)
					}
				}
				return true
			})
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("function %s not found", name)
		}
	}
}
