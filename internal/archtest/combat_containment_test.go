package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// containmentDomainFiles and containmentUseCaseFiles are the production files of monster-engulf-and-swallow.md.
func containmentDomainFiles() []string {
	return []string{"combat/containment_terms.go", "combat/containment_reasons.go", "combat/containment_event.go", "combat/containment.go", "combat/containment_rules.go"}
}

func containmentUseCaseFiles() []string {
	return []string{"combat/containment_usecase.go", "combat/enter_container.go", "combat/exit_container.go", "combat/advance_container_turn.go",
		"combat/resolve_regurgitation.go", "combat/release_occupants.go"}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-074, FR-075, observability.md section 3 - the containment domain files import only the kernel, content and d20 (no telemetry, context or signal constant); the use-case files add modifier, telemetry and the dice port; none rolls or applies damage.
// Path: unhappy (forbidden edge).
func TestContainmentFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"ability", "content", "d20", "dice", "srderr"}
		domain := slices.Contains(containmentDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(containmentUseCaseFiles(), f.Path):
			allowed = append(allowed, "modifier", "telemetry")
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which monster-engulf-and-swallow.md does not allow", f.Path, p)
			}
			if domain && (strings.HasSuffix(imp.Path, "/telemetry") || strings.HasSuffix(imp.Path, "/context") || strings.HasSuffix(imp.Path, "/modifier")) {
				t.Errorf("%s imports %s: the containment domain is silent and pure (FR-074)", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if domain && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr") || strings.HasPrefix(id.Name, "msg")) {
				t.Errorf("%s names the signal constant %s: the domain is silent (FR-074)", f.Path, id.Name)
			}
			if id.Name == "ApplyDamage" || id.Name == "rollDamage" || id.Name == "rollTrapDamage" || id.Name == "DamageSpec" {
				t.Errorf("%s names %s: containment rolls and applies no damage (FR-075)", f.Path, id.Name)
			}
			return true
		})
	}
	if want := len(containmentDomainFiles()) + len(containmentUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d containment files", seen, want)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-005, FR-008, FR-017, FR-066 - the closed sets have exactly the variants the spec lists: the entry terms, the death rules, the placements, the lookup, the transition, the tests, the exit methods, the release triggers and the five use case results.
// Path: unhappy (duplicate).
func TestContainmentClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isEntryTerms":                 {"EntryAutomatic", "EntryBySave"},
		"isContainerDeathRule":         {"EscapeFromCorpse", "ReleaseOnDeath"},
		"isExitPlacement":              {"LandingWithin", "NearestUnoccupiedSpace"},
		"isContainmentOfOccupant":      {"InContainer", "NotContained"},
		"isContainmentTransition":      {"ContainmentTransitionApplied", "ContainmentTransitionRejected"},
		"isContainmentTest":            {"ContainmentTestForced", "ContainmentTestNotRequired", "ContainmentTestRolled"},
		"isExitContainerMethod":        {"ExitByPull", "ExitBySelf"},
		"isReleaseTrigger":             {"ContainerDied", "EscapedCorpse", "GrappleReview", "OccupantGone"},
		"isEnterContainerResult":       {"EnterContainerContained", "EnterContainerDamagedOnly", "EnterContainerInvalid", "EnterContainerRejected", "EnterContainerResisted"},
		"isExitContainerResult":        {"ExitContainerExited", "ExitContainerHeld", "ExitContainerRejected"},
		"isAdvanceContainerTurnResult": {"AdvanceContainerTurnAdvanced", "AdvanceContainerTurnInvalid"},
		"isResolveRegurgitationResult": {"ResolveRegurgitationHeld", "ResolveRegurgitationInvalid", "ResolveRegurgitationNotDue", "ResolveRegurgitationRegurgitated"},
		"isReleaseOccupantsResult":     {"ReleaseOccupantsInvalid", "ReleaseOccupantsRejected", "ReleaseOccupantsReleased"},
		"containmentEvent":             {"containmentEventBase"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-064 - exactly the three containment events embed the containment event base, all declared in containment_event.go.
// Path: unhappy (duplicate).
func TestContainmentEventFamilyHasExactlyThreeMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "containmentEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/containment_event.go" {
							t.Errorf("%s is declared in %s, want combat/containment_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if want := []string{"ContainerTrapped", "OccupantContained", "OccupantReleased"}; !slices.Equal(members, want) {
		t.Errorf("containment event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-065 - the three rolling use case constructors take a Roller and options only and the two others options only.
// Path: unhappy (forbidden edge).
func TestContainmentConstructorsTakeOnlyRollerAndOptions(t *testing.T) {
	t.Parallel()
	want := map[string]string{"NewEnterContainer": "Roller,...Option", "NewExitContainer": "Roller,...Option", "NewResolveRegurgitation": "Roller,...Option",
		"NewAdvanceContainerTurn": "...Option", "NewReleaseOccupants": "...Option"}
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

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-067, FR-069, FR-073, section 7, plan-phase-4.md P11 - signals.go pins the five operation values and spans, the two metrics, the end-reason attribute, the debug log message and the new result values at their catalogue values, and signals.md has a row for each metric and span.
// Path: unhappy (invalid input).
func TestContainmentSignalNamesAndCatalogue(t *testing.T) {
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
		"metricContainmentEnded": "srd5e.combat.containment.ended", "metricContainmentOccupants": "srd5e.combat.containment.occupants",
		"attrContainmentEndReason": "srd5e.combat.containment.end_reason", "msgContainmentOperation": "containment operation",
		"valueResultContained": "contained", "valueResultResisted": "resisted", "valueResultDamagedOnly": "damaged_only", "valueResultExited": "exited",
		"valueResultRegurgitated": "regurgitated", "valueResultNotDue": "not_due",
	}
	doc := catalogue(t)
	for _, op := range strings.Fields("EnterContainer:enter_container ExitContainer:exit_container AdvanceContainerTurn:advance_container_turn ResolveRegurgitation:resolve_regurgitation ReleaseOccupants:release_occupants") {
		name, value, _ := strings.Cut(op, ":")
		want["valueOperation"+name] = value
		want["span"+name] = "srd5e.combat." + value
		if !strings.Contains(doc, "| `srd5e.combat."+value+"` |") {
			t.Errorf("signals.md has no row for the span srd5e.combat.%s", value)
		}
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
		if strings.HasPrefix(name, "metric") && !strings.Contains(doc, "| `"+value+"` |") {
			t.Errorf("signals.md has no row for the metric %s", value)
		}
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-076, FR-077, section 6 (Exports) - package combat exports the containment types and use cases once each, exactly two reason enums, and no generic variant name.
// Path: happy.
func TestCombatExportsContainmentTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		ContainmentMode SuffocationRule ContainmentTiming ContainmentActionKind ContainmentCapacity DamageTerm EntryTerms EntryAutomatic EntryBySave PeriodicDamage Regurgitation
		ContainerDeathRule ReleaseOnDeath EscapeFromCorpse ExitTerms GrappleHoldTerms ContainmentTermsInput ContainmentTerms NewContainmentTerms MustContainmentTerms
		ContainmentPhase ContainmentInput Containment NewContainment Containments NewContainments MustContainments ContainmentOfOccupant InContainer NotContained
		ContainmentTransition ContainmentTransitionApplied ContainmentTransitionRejected ContainerTurnBoundary ContainmentRules ContainmentRelease OccupantTrapped
		ContainmentRejection ContainmentProblem ContainmentError ContainmentLeaveCause DamagedOnlyReason ContainmentCover ContainedRestrictions ContainmentEffects
		ContainmentDamageCause ContainmentDamageDue ContainedPlacement RelocationDue ExitPlacement NearestUnoccupiedSpace LandingWithin ConditionImposition ImposedCondition
		ContainmentEvent OccupantContained OccupantReleased ContainerTrapped ContainmentTest ContainmentTestRolled ContainmentTestForced ContainmentTestNotRequired
		EntryCircumstance EnterContainerInput EnterContainerCommand NewEnterContainerCommand EnterContainerResult EnterContainerContained EnterContainerResisted EnterContainerDamagedOnly
		EnterContainerRejected EnterContainerInvalid EnterContainer NewEnterContainer
		ExitContainerMethod ExitBySelf ExitByPull ExitContainerInput ExitContainerCommand NewExitContainerCommand ExitContainerResult ExitContainerExited ExitContainerHeld ExitContainerRejected ExitContainer NewExitContainer
		AdvanceContainerTurnInput AdvanceContainerTurnCommand NewAdvanceContainerTurnCommand AdvanceContainerTurnResult AdvanceContainerTurnAdvanced AdvanceContainerTurnInvalid AdvanceContainerTurn NewAdvanceContainerTurn
		ResolveRegurgitationInput ResolveRegurgitationCommand NewResolveRegurgitationCommand ResolveRegurgitationResult ResolveRegurgitationRegurgitated ResolveRegurgitationHeld
		ResolveRegurgitationNotDue ResolveRegurgitationInvalid ResolveRegurgitation NewResolveRegurgitation
		ReleaseTrigger ContainerDied EscapedCorpse GrappleReview OccupantGone ReleaseOccupantsInput ReleaseOccupantsCommand NewReleaseOccupantsCommand ReleaseOccupantsResult
		ReleaseOccupantsReleased ReleaseOccupantsRejected ReleaseOccupantsInvalid ReleaseOccupants NewReleaseOccupants`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Contained", "Released", "Held", "Exited", "Resisted", "DamagedOnly", "Advanced", "Regurgitated", "NotDue", "Rejected", "Invalid", "ContainmentRejected"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-076)", generic)
		}
	}
	for _, banned := range []string{"Grappled", "ConditionSet", "CreatureSize", "DamageType", "Expression"} {
		for _, f := range have[banned] {
			if strings.Contains(f, "containment") || strings.Contains(f, "_container") || strings.Contains(f, "occupants") {
				t.Errorf("%s is re-declared in %s (FR-076)", banned, f)
			}
		}
	}
}
