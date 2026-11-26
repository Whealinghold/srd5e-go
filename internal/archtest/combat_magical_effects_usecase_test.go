package archtest_test

import (
	"go/ast"
	"slices"
	"testing"
)

// Requirement: doc/specs/magical-effects.md FR-148, FR-143 and doc/rules/architecture.md section 4 - the magical-effects use-case file names no other use case and no pure service of another spec: it composes CurseRemoval, ShapeShifting, Teleportation, SimultaneousEffects, AreaMembership and AreaOrigin only, rolls no die and asks no ConditionEffects.
// Path: unhappy (forbidden edge).
func TestMagicalEffectsUseCaseCallsNoUseCase(t *testing.T) {
	t.Parallel()
	banned := []string{"ConditionEffects", "RollStatBlockDamage", "EnterContainer", "StartActiveEffect", "AdvanceActiveEffects", "EndActiveEffects", "MakeDeathSavingThrow",
		"ResolveD20Test", "BeginShapeShifting", "ReviveCreatureBySpell", "ConcentrationRules", "DepartureMeans", "Roller", "WrapPort"}
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if f.Path != "combat/magical_effects_usecase.go" {
			continue
		}
		seen++
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && slices.Contains(banned, id.Name) {
				t.Errorf("%s names %s: a use case composes the services of its spec and calls no use case (FR-148)", f.Path, id.Name)
			}
			return true
		})
	}
	if seen != 1 {
		t.Fatalf("found %d magical-effects use-case files, want 1", seen)
	}
}

// Requirement: doc/specs/magical-effects.md FR-148, doc/specs/combat-action-resolution.md FR-127 - the requests and results of the magical-effects use cases are closed sets with exactly the variants of their owners, with no wrapper type.
// Path: unhappy (duplicate).
func TestMagicalEffectsClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isShapeShiftingRequest": {"RevertOnDeathRequest", "RevertRequest", "ShiftRequest"},
		"isShiftCombatantShapeResult": {"RevertOnDeathApplied", "RevertOnDeathNotDead", "RevertToTrueFormApplied", "RevertToTrueFormExceptionNotHeld",
			"ShiftShapeApplied", "ShiftShapeExceptionNotHeld"},
		"isAreaLocationRequest":      {"AssessMembershipRequest", "ResolveOriginRequest"},
		"isAssessAreaLocationResult": {"AreaMembershipExcluded", "AreaMembershipIncluded", "AreaMembershipNoLinesConsidered", "AreaOriginResolution"},
	}
	for marker, names := range want {
		slices.Sort(names)
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}
