package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"
	"testing"
)

// movementFiles are the production files of movement-lifting-and-flight.md: pure domain files, no use case.
func movementFiles() []string {
	return []string{"combat/movement_load.go", "combat/movement_jump.go", "combat/movement_landing.go",
		"combat/movement_check.go", "combat/movement_flight.go", "combat/movement_burrow.go"}
}

// movementFile returns the production file of combat at path.
func movementFile(t *testing.T, path string) *ast.File {
	t.Helper()
	for _, f := range combatPackage(t).Production() {
		if f.Path == path {
			return f.AST
		}
	}
	t.Fatalf("%s not found", path)
	return nil
}

// structFields returns the field names of the struct type name declared in file, and whether it exists.
func structFields(file *ast.File, name string) ([]string, bool) {
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, s := range gd.Specs {
			if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.Name == name {
				st, isStruct := ts.Type.(*ast.StructType)
				if !isStruct {
					return nil, false
				}
				var out []string
				for _, f := range st.Fields.List {
					for _, n := range f.Names {
						out = append(out, n.Name)
					}
				}
				return out, true
			}
		}
	}
	return nil, false
}

// funcParamTypes returns the parameter type names of the function name declared in file.
func funcParamTypes(t *testing.T, file *ast.File, name string) []string {
	t.Helper()
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == name {
			var out []string
			for _, p := range fn.Type.Params.List {
				typeName := ""
				switch v := p.Type.(type) {
				case *ast.Ident:
					typeName = v.Name
				case *ast.SelectorExpr:
					typeName = v.Sel.Name
				default:
					t.Fatalf("%s has a parameter of a type other than a name", name)
				}
				for range max(1, len(p.Names)) {
					out = append(out, typeName)
				}
			}
			return out
		}
	}
	t.Fatalf("func %s not found", name)
	return nil
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-022, FR-053, section 7 - the movement files are silent, pure domain code: they import
// only ability, character, content, d20 and srderr of this module (no telemetry, dice, modifier or other context), no standard-library
// package of effects, and name no signal constant.
// Path: unhappy (forbidden edge).
func TestMovementFilesAreSilentPureDomainCode(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	allowed := []string{"ability", "character", "content", "d20", "srderr"}
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(movementFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which movement-lifting-and-flight.md does not allow", f.Path, p)
			}
			if slices.Contains([]string{"context", "math/rand", "math/rand/v2", "time", "os", "sync"}, imp.Path) {
				t.Errorf("%s imports %s: the movement rules are pure (FR-053)", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr")) {
				t.Errorf("%s names the signal constant %s: the spec is silent by design (section 7)", f.Path, id.Name)
			}
			return true
		})
	}
	if seen != len(movementFiles()) {
		t.Fatalf("found %d of %d movement files", seen, len(movementFiles()))
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-006, FR-059, FR-060 - movement_load.go holds neither the literal 5 (a cap) nor 10 (a penalty):
// the cap is read from LimitFeet and the penalty from SpeedPenaltyFeet; LimitSpeedsByLoad takes a LoadSpeedLimited only; and package combat declares no
// Carrying Capacity or LoadAssessment type.
// Path: unhappy (duplicate).
func TestMovementLoadReadsTheCapAndPenaltyFromTheCharacterContext(t *testing.T) {
	t.Parallel()
	file := movementFile(t, "combat/movement_load.go")
	called := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.BasicLit:
			if v.Kind == token.INT && (v.Value == "5" || v.Value == "10") {
				t.Errorf("movement_load.go holds the literal %s, a Speed cap or penalty that character owns", v.Value)
			}
		case *ast.SelectorExpr:
			called[v.Sel.Name] = true
		}
		return true
	})
	for _, want := range []string{"LimitFeet", "SpeedPenaltyFeet"} {
		if !called[want] {
			t.Errorf("movement_load.go does not call %s", want)
		}
	}
	if got := funcParamTypes(t, file, "LimitSpeedsByLoad"); !slices.Equal(got, []string{"MovementSpeeds", "LoadSpeedLimited", "LoadLimitScope"}) {
		t.Errorf("LimitSpeedsByLoad parameter types = %v, want a LoadSpeedLimited", got)
	}
	for name, files := range declared(combatPackage(t)) {
		if slices.Contains([]string{"CarryingCapacity", "LoadAssessment", "LoadWithinCarry", "LoadSpeedLimited", "LoadBeyondDragLiftPush"}, name) {
			t.Errorf("combat declares %s in %v: the Carrying Capacity belongs to character (FR-006)", name, files)
		}
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-033, FR-034 - MovementCheckRequirement has the one field kind (no roll, total, outcome or
// stored ability), and no function of the movement files takes a bool parameter (no switch for the GM's option).
// Path: unhappy (rule interaction).
func TestMovementCheckRequirementDescribesWithoutRolling(t *testing.T) {
	t.Parallel()
	if got, ok := structFields(movementFile(t, "combat/movement_check.go"), "MovementCheckRequirement"); !ok || !slices.Equal(got, []string{"kind"}) {
		t.Errorf("MovementCheckRequirement fields = %v (declared %v), want [kind]", got, ok)
	}
	for _, path := range movementFiles() {
		ast.Inspect(movementFile(t, path), func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}
			for _, p := range fn.Type.Params.List {
				if id, ok := p.Type.(*ast.Ident); ok && id.Name == "bool" {
					t.Errorf("%s: %s takes a bool parameter: the movement rules carry no switch for the GM's option (FR-033)", path, fn.Name.Name)
				}
			}
			return true
		})
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-044, FR-045, FR-046 - hover is only a field of FlightFacts, the assessment takes the facts
// alone (no movement, turn or duration), FlightFacts holds exactly the conditions, the base Fly Speed and the hover ability, and FlightFalls carries only its causes.
// Path: unhappy (rule interaction).
func TestFlightDependsOnTheFactsAlone(t *testing.T) {
	t.Parallel()
	file := movementFile(t, "combat/movement_flight.go")
	if got, ok := structFields(file, "FlightFacts"); !ok || !slices.Equal(got, []string{"conditions", "baseFlySpeedFeet", "hover"}) {
		t.Errorf("FlightFacts fields = %v, want conditions, baseFlySpeedFeet, hover", got)
	}
	if got := funcParamTypes(t, file, "AssessFlight"); !slices.Equal(got, []string{"FlightFacts"}) {
		t.Errorf("AssessFlight parameters = %v, want FlightFacts alone", got)
	}
	if got, ok := structFields(file, "FlightFalls"); !ok || !slices.Equal(got, []string{"Causes"}) {
		t.Errorf("FlightFalls fields = %v, want Causes alone", got)
	}
	if got, ok := structFields(file, "FlightStaysAloft"); !ok || !slices.Equal(got, []string{"OverriddenCauses"}) {
		t.Errorf("FlightStaysAloft fields = %v, want OverriddenCauses alone", got)
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-052 - BurrowPermissionFor takes a medium and a solid-rock access and no speed.
// Path: unhappy (rule interaction).
func TestBurrowPermissionReadsNoSpeed(t *testing.T) {
	t.Parallel()
	if got := funcParamTypes(t, movementFile(t, "combat/movement_burrow.go"), "BurrowPermissionFor"); !slices.Equal(got, []string{"BurrowMedium", "SolidRockAccess"}) {
		t.Errorf("BurrowPermissionFor parameters = %v, want BurrowMedium, SolidRockAccess", got)
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-054 - no struct of the movement files has a string field, so no result or effect carries a display string.
// Path: unhappy (invalid input).
func TestMovementTypesCarryNoDisplayString(t *testing.T) {
	t.Parallel()
	for _, path := range movementFiles() {
		ast.Inspect(movementFile(t, path), func(n ast.Node) bool {
			st, ok := n.(*ast.StructType)
			if !ok {
				return true
			}
			for _, f := range st.Fields.List {
				if id, ok := f.Type.(*ast.Ident); ok && id.Name == "string" {
					t.Errorf("%s: a struct has a string field %v (FR-054)", path, f.Names)
				}
			}
			return true
		})
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-055 - package combat exports the movement types and functions once each.
// Path: happy.
func TestCombatExportsMovementTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		LimitSpeedsByLoad LoadLimitScope ApplyArmorSpeedPenalty JumpKind JumpApproach JumpApproachFor JumpMaximumFeet JumpAttempt NewJumpAttempt
		JumpAssessment JumpPermitted JumpRefused JumpRefusal AssessJump JumpLanding JumpLandsClear JumpLandingCheckRequired JumpLandingFor
		JumpObstacle JumpObstacleCheckRequired JumpObstacleNotCovered ObstacleForLongJump HighJumpReachFeet
		MovementCheckKind MovementCheckFailureEffect MovementCheckRequirement RequirementFor
		HoverAbility FlightFallCause FlightFallCauses FlightFacts NewFlightFacts FlightAssessment FlightStaysAloft FlightFalls AssessFlight
		BurrowMedium SolidRockAccess BurrowRefusal BurrowPermission BurrowAllowed BurrowRefused BurrowPermissionFor`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-015, FR-023, FR-026, FR-037, FR-049 - the five closed sets of the movement files have exactly their variants.
// Path: unhappy (duplicate).
func TestMovementClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isJumpAssessment":   {"JumpPermitted", "JumpRefused"},
		"isJumpLanding":      {"JumpLandingCheckRequired", "JumpLandsClear"},
		"isJumpObstacle":     {"JumpObstacleCheckRequired", "JumpObstacleNotCovered"},
		"isFlightAssessment": {"FlightFalls", "FlightStaysAloft"},
		"isBurrowPermission": {"BurrowAllowed", "BurrowRefused"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}
