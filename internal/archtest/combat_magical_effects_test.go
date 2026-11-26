package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// magicalEffectsFiles are the production files of magical-effects.md: pure domain code, no use case file.
func magicalEffectsFiles() []string {
	return []string{
		"combat/magical_effect_source.go", "combat/ongoing_effect.go", "combat/area_origin.go", "combat/area_membership.go",
		"combat/curse_removal.go", "combat/possession.go", "combat/illusion.go", "combat/shape_shifting.go",
		"combat/teleportation.go", "combat/simultaneous_effects.go", "combat/magical_effect_event.go",
	}
}

// Requirement: doc/specs/magical-effects.md FR-143, FR-064 and section 7 - the magical-effects files are silent domain code: they import
// only content and srderr of this module (no telemetry, dice, d20, character, toolbox), name no signal constant and no
// ConcentrationRules, and the teleportation file neither imports nor names DepartureMeans (teleportation provokes no Opportunity Attack).
// Path: unhappy (forbidden edge).
func TestMagicalEffectsFilesAreSilentDomainCode(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(magicalEffectsFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && p != "srderr" && p != "content" {
				t.Errorf("%s imports %s, which magical-effects.md FR-143 does not allow", f.Path, p)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr") {
				t.Errorf("%s names the signal constant %s: the spec owns no signal (section 7)", f.Path, id.Name)
			}
			if id.Name == "ConcentrationRules" || (id.Name == "DepartureMeans" && f.Path == "combat/teleportation.go") {
				t.Errorf("%s names %s, which the spec forbids [FR-143, FR-064]", f.Path, id.Name)
			}
			return true
		})
	}
	if seen != len(magicalEffectsFiles()) {
		t.Fatalf("found %d of %d magical-effects files", seen, len(magicalEffectsFiles()))
	}
}

// Requirement: doc/specs/magical-effects.md FR-147 - exactly CurseRemoved, ShapeShifted, RevertedToTrueForm and CreatureTeleported embed the
// magical-effect event base, all declared in magical_effect_event.go.
// Path: unhappy (duplicate).
func TestMagicalEffectEventFamilyHasExactlyFourMembers(t *testing.T) {
	t.Parallel()
	var members []string
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			if st, ok := ts.Type.(*ast.StructType); ok {
				for _, fld := range st.Fields.List {
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "magicalEffectEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/magical_effect_event.go" {
							t.Errorf("%s is declared in %s, want combat/magical_effect_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if want := []string{"CreatureTeleported", "CurseRemoved", "RevertedToTrueForm", "ShapeShifted"}; !slices.Equal(members, want) {
		t.Errorf("magical-effect event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/magical-effects.md section 6, FR-143, FR-145 - package combat declares the exported names of the spec once each, and
// declares none of the names that belong to other specs: the area shapes (content), TurnOwner (spell-effect-data), NotPossessed,
// PossessionState and DemonicPossessionState (toolbox-curses-and-contagions), and no Concentration name.
// Path: unhappy (duplicate).
func TestCombatExportsMagicalEffectsTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		EffectSourceKind EffectSource SpellSource MagicItemSource LabeledMagicalSource NonMagicalSource
		OngoingEffectID NewOngoingEffectID OngoingEffectCategory OngoingEffect NewOngoingEffect OngoingEffects NewOngoingEffects
		ConeWidthAt OverridableAreaShape OverridableCone OverridableCube OverridableEmanation OverridableLine OriginInclusionDecision
		ShapeDefaultOrigin CreatorIncludesOrigin EmanationEffectTiming OriginPlacement AreaOriginResolution AreaOrigin
		ShapeContainment AreaExclusion AreaLocationFacts OriginLocationFacts NewOriginLocationFacts OtherLocationFacts NewOtherLocationFacts
		AreaMembershipResult AreaMembershipIncluded AreaMembershipExcluded AreaMembershipNoLinesConsidered AreaMembership
		CurseRemovalMeans CurseRemovalMeansFromSpell MagicThatEndsCurses RemoveCombatantCurseResult RemoveCurseApplied RemoveCurseNotPresent
		RemoveCurseNotACurse CurseRemoval PossessionPrevention PossessionEnding PossessionPreventionFromSpell PossessionEndingFromSpell
		IllusionSenseID NewIllusionSenseID IllusionBodyFacts EnvironmentAppearance IllusionManifestation NotInSpace ManifestsInSpace
		NewManifestsInSpace IllusionTraits NewIllusionTraits ShapeShiftState TrueForm ShiftedForm CarryOverRule CarryOverAll CarryOverExcept
		NewCarryOverExcept NotCarriedOver ShiftShapeCommand ShiftShapeResult ShiftShapeApplied ShiftShapeExceptionNotHeld ShapeShifting
		RevertToTrueFormCommand RevertToTrueFormResult RevertToTrueFormApplied RevertToTrueFormExceptionNotHeld RevertOnDeathResult
		RevertOnDeathApplied RevertOnDeathNotDead DestinationSpace DestinationSight TouchedCreaturePolicy TeleportFields TeleportCommand
		NewTeleportCommand TeleportCombatantResult TeleportArrivedAtDestination TeleportArrivedNearDestination TeleportDestinationNotSeen
		TeleportInvalidTouchedSet Teleportation TurnHolder OrderingAuthority PendingEffectID NewPendingEffectID OrderProblemKind OrderProblem
		ResolveEffectOrderResult OrderSimultaneousEffectsOrdered OrderSimultaneousEffectsRejected SimultaneousEffects
		MagicalEffectEvent CurseRemoved ShapeShifted RevertCause RevertedToTrueForm TeleportArrival CreatureTeleported`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, name := range strings.Fields(`AreaShape ConeArea CubeArea CylinderArea SphereArea EmanationArea LineArea TurnOwner NotPossessed
		PossessionState DemonicPossessionState PossessionEndingMagic ConcentrationState ConcentrationRules DepartureMeansTeleport`) {
		if len(have[name]) != 0 {
			t.Errorf("combat declares %s, which belongs to another spec [FR-143, FR-145]", name)
		}
	}
}
