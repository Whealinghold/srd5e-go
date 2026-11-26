package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// mountingDomainFiles and mountingUseCaseFiles are the production files of mounted-and-underwater-combat.md.
func mountingDomainFiles() []string {
	return []string{"combat/mount_state.go", "combat/mounting_rejection.go", "combat/mounting_rules.go", "combat/mount_event.go",
		"combat/falling_off.go", "combat/underwater_resistance.go"}
}

func mountingUseCaseFiles() []string {
	return []string{"combat/mount_creature.go", "combat/dismount_creature.go", "combat/resolve_falling_off.go", "combat/mounting_usecase.go"}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-050, FR-071, observability.md section 3 - the mounting domain files import only content, d20 and srderr of this module and name no signal constant (they are silent, pure, with no telemetry or dice); the use-case files add dice, modifier and telemetry.
// Path: unhappy (forbidden edge).
func TestMountingFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"content", "d20", "srderr"}
		domain := slices.Contains(mountingDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(mountingUseCaseFiles(), f.Path):
			allowed = append(allowed, "dice", "modifier", "telemetry")
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which mounted-and-underwater-combat.md does not allow", f.Path, p)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if domain && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr")) {
				t.Errorf("%s names the signal constant %s: the domain is silent (FR-050)", f.Path, id.Name)
			}
			if slices.Contains([]string{"TurnTracker", "TurnResources", "CombatantProfile", "ConditionActionAvailability", "ConditionSet", "ConditionEffects"}, id.Name) {
				t.Errorf("%s names %s: mounting reads no tracker, resource or condition (FR-018, FR-023, FR-066)", f.Path, id.Name)
			}
			if f.Path == "combat/resolve_falling_off.go" && id.Name == "MountingTurnFacts" {
				t.Errorf("%s names MountingTurnFacts: the save does not depend on the turn (FR-036)", f.Path)
			}
			return true
		})
	}
	if want := len(mountingDomainFiles()) + len(mountingUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d mounting files", seen, want)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-018, FR-023, FR-036, FR-066 - no struct of the mounting files has a field of type TurnTracker, TurnResources, MountingTurnFacts (the save command), ConditionActionAvailability, ConditionSet or ConditionKind.
// Path: unhappy (forbidden edge).
func TestMountingStructsDeclareNoForbiddenField(t *testing.T) {
	t.Parallel()
	forbidden := []string{"TurnTracker", "TurnResources", "ConditionActionAvailability", "ConditionSet", "ConditionKind", "CombatantProfile"}
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(mountingDomainFiles(), f.Path) && !slices.Contains(mountingUseCaseFiles(), f.Path) {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, fld := range st.Fields.List {
				ast.Inspect(fld.Type, func(m ast.Node) bool {
					name := ""
					switch x := m.(type) {
					case *ast.Ident:
						name = x.Name
					case *ast.SelectorExpr:
						name = x.Sel.Name
					}
					if slices.Contains(forbidden, name) || (ts.Name.Name == "ResolveFallingOffCommand" && name == "MountingTurnFacts") {
						t.Errorf("%s: %s has a field of type %s", f.Path, ts.Name.Name, name)
					}
					return true
				})
			}
			return true
		})
	}
}

// markerReceivers returns the receiver type names of the methods called marker, sorted.
func markerReceivers(t *testing.T, marker string) []string {
	t.Helper()
	var out []string
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv != nil && fn.Name.Name == marker {
				if id, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
					out = append(out, id.Name)
				}
			}
		}
	}
	slices.Sort(out)
	return out
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-053, FR-054, FR-073 - the rejection sets, the three results and FallingOffSave have exactly the variants the spec lists.
// Path: unhappy (duplicate).
func TestMountingClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isMountRejection": strings.Fields(`MountingRiderNotActiveRejection MountingRiderAlreadyMountedRejection MountingMountUnwillingRejection
			MountingMountNotLargerRejection MountingMountAnatomyRejection MountingExoticSaddleRequiredRejection MountingOutOfReachRejection
			MountingMountCannotBeControlledRejection MountingNoMovementRemainingRejection MountingInsufficientMovementRejection`),
		"isDismountRejection":       strings.Fields(`MountingRiderNotActiveRejection MountingRiderNotMountedRejection MountingNoMovementRemainingRejection MountingInsufficientMovementRejection`),
		"isFallingOffRejection":     {"MountingRiderNotMountedRejection"},
		"isMountCreatureResult":     {"MountCreatureApplied", "MountCreatureRejected"},
		"isDismountCreatureResult":  {"DismountCreatureApplied", "DismountCreatureRejected"},
		"isResolveFallingOffResult": {"ResolveFallingOffFellOff", "ResolveFallingOffRejected", "ResolveFallingOffStayedMounted"},
		"isFallingOffSave":          {"FallingOffSaveForced", "FallingOffSaveRolled"},
		"isMountInitiativeChange":   {"MountInitiativeMatchesRider", "MountInitiativeUnchanged"},
	}
	for marker, names := range want {
		slices.Sort(names)
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-055, FR-068 - exactly CreatureMounted, CreatureDismounted and FellOffMount embed the mount event base, all declared once in mount_event.go.
// Path: unhappy (duplicate).
func TestMountEventFamilyHasExactlyThreeMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "mountEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/mount_event.go" {
							t.Errorf("%s is declared in %s, want combat/mount_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if want := []string{"CreatureDismounted", "CreatureMounted", "FellOffMount"}; !slices.Equal(members, want) {
		t.Errorf("mount event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-071 - NewResolveFallingOff takes a Roller and options only; no mounting use case constructor takes anything else.
// Path: unhappy (forbidden edge).
func TestMountingConstructorsTakeOnlyRollerAndOptions(t *testing.T) {
	t.Parallel()
	want := map[string]string{"NewResolveFallingOff": "Roller,...Option", "NewMountCreature": "...Option", "NewDismountCreature": "...Option"}
	found := 0
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || want[fn.Name.Name] == "" {
				continue
			}
			found++
			var params []string
			for _, p := range fn.Type.Params.List {
				switch x := p.Type.(type) {
				case *ast.Ident:
					params = append(params, x.Name)
				case *ast.Ellipsis:
					params = append(params, "..."+x.Elt.(*ast.Ident).Name)
				}
			}
			if got := strings.Join(params, ","); got != want[fn.Name.Name] {
				t.Errorf("%s takes (%s), want (%s)", fn.Name.Name, got, want[fn.Name.Name])
			}
		}
	}
	if found != len(want) {
		t.Errorf("found %d of %d constructors", found, len(want))
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-058, FR-061, FR-064, plan-phase-4.md P11 - signals.go pins the three operation values and spans, the movement histogram and the two attribute keys at their catalogue values, with no superseded name.
// Path: unhappy (invalid input).
func TestMountingSignalNames(t *testing.T) {
	t.Parallel()
	got := map[string]string{}
	for _, f := range combatPackage(t).Production() {
		if f.Path != "combat/signals.go" {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if v, ok := n.(*ast.ValueSpec); ok && len(v.Values) == 1 {
				if lit, ok := v.Values[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					got[v.Names[0].Name], _ = strconv.Unquote(lit.Value)
				}
			}
			return true
		})
	}
	want := map[string]string{
		"metricMountingMovementSpent": "srd5e.combat.mounting_movement_spent", "attrMountControl": "srd5e.combat.mount_control",
		"attrFallingOffTrigger": "srd5e.combat.falling_off_trigger", "valueResultStayedMounted": "stayed_mounted", "valueResultFellOff": "fell_off",
	}
	for _, op := range strings.Fields("MountCreature:mount_creature DismountCreature:dismount_creature ResolveFallingOff:resolve_falling_off") {
		name, value, _ := strings.Cut(op, ":")
		want["valueOperation"+name] = value
		want["span"+name] = "srd5e.combat." + value
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md section 6 (Exports) - package combat exports the use cases, commands, results, statuses, facts, rejections, events and the underwater function once each.
// Path: happy.
func TestCombatExportsMountingTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		MountCreature NewMountCreature MountCreatureInput MountCreatureCommand NewMountCreatureCommand MountCreatureResult MountCreatureApplied MountCreatureRejected
		DismountCreature NewDismountCreature DismountCreatureCommand NewDismountCreatureCommand DismountCreatureResult DismountCreatureApplied DismountCreatureRejected
		ResolveFallingOff NewResolveFallingOff ResolveFallingOffCommand NewResolveFallingOffCommand ResolveFallingOffResult ResolveFallingOffStayedMounted
		ResolveFallingOffFellOff ResolveFallingOffRejected RiderStatus NotMounted MountedOn MountControl MountControlControlled MountControlIndependent
		MountFacts NewMountFacts MountWillingness MountAnatomy MountTraining MountHabitat ExoticSaddle RiderTurnState MountingTurnFacts NewMountingTurnFacts
		MountInitiativeChange MountInitiativeMatchesRider MountInitiativeUnchanged MountTurn MountTurnOnRidersTurn MountTurnOnOwnTurn FallingOffTrigger
		LandingSpace FallingOffDifficultyClass FallingOffSave FallingOffSaveRolled FallingOffSaveForced MountRejection DismountRejection FallingOffRejection
		MountingRiderNotActiveRejection MountingRiderAlreadyMountedRejection MountingMountUnwillingRejection MountingMountNotLargerRejection
		MountingMountAnatomyRejection MountingExoticSaddleRequiredRejection MountingOutOfReachRejection MountingMountCannotBeControlledRejection
		MountingNoMovementRemainingRejection MountingInsufficientMovementRejection MountingRiderNotMountedRejection MountEvent CreatureMounted
		CreatureDismounted FellOffMount UnderwaterFireResistanceAffinities UnderwaterStatus UnderwaterResistanceGrant`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, unexported := range []string{"mountRefusal", "dismountRefusal", "movementShortfall"} {
		if len(have[unexported]) != 1 {
			t.Errorf("the mounting rule %s must be declared (unexported) once", unexported)
		}
	}
	for _, generic := range []string{"Applied", "Refused", "Rejected", "TurnStep"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-054)", generic)
		}
	}
}
