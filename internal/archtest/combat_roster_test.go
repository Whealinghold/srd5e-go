package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// rosterDomainFiles and rosterUseCaseFiles are the production files of combatant-roster-changes.md.
func rosterDomainFiles() []string {
	return []string{"combat/roster_position.go", "combat/roster_reason.go", "combat/roster_rejection.go", "combat/roster_event.go",
		"combat/roster_step.go", "combat/encounter_roster.go"}
}

func rosterUseCaseFiles() []string { return []string{"combat/roster_usecase.go"} }

// Requirement: doc/specs/combatant-roster-changes.md FR-046, FR-014, FR-049, FR-041, observability.md section 3 - the roster domain files import only content of this module (no context, time, math/rand, telemetry, dice or srderr), name no signal constant and no ActionLedger or Help type; the use-case file adds context and telemetry only, names no Roller, Containments, ReleaseOccupants or AdvanceCombatTurn and calls no Run.
// Path: unhappy (forbidden edge).
func TestRosterFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		domain := slices.Contains(rosterDomainFiles(), f.Path)
		if !domain && !slices.Contains(rosterUseCaseFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				allowed := []string{"content"}
				if !domain {
					allowed = append(allowed, "telemetry")
				}
				if !slices.Contains(allowed, p) {
					t.Errorf("%s imports %s, which combatant-roster-changes.md does not allow", f.Path, p)
				}
				continue
			}
			switch imp.Path {
			case "slices":
			case "context":
				if domain {
					t.Errorf("%s (domain) imports context (FR-046)", f.Path)
				}
			default:
				t.Errorf("%s imports %s, which combatant-roster-changes.md does not allow (FR-046)", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.Ident:
				if domain && (strings.HasPrefix(v.Name, "metric") || strings.HasPrefix(v.Name, "span") || strings.HasPrefix(v.Name, "attr")) {
					t.Errorf("%s names the signal constant %s: the domain is silent", f.Path, v.Name)
				}
				if domain && (v.Name == "ActionLedger" || strings.HasPrefix(v.Name, "Help")) {
					t.Errorf("%s names %s: the Help clean-up is the use case's (FR-025)", f.Path, v.Name)
				}
				if !domain && slices.Contains([]string{"Roller", "Containments", "ReleaseOccupants", "AdvanceCombatTurn", "Roll"}, v.Name) {
					t.Errorf("%s names %s: the roster use cases roll nothing, hold no containment and call no other use case (FR-014, FR-041, FR-049)", f.Path, v.Name)
				}
			case *ast.SelectorExpr:
				if !domain && v.Sel.Name == "Run" {
					t.Errorf("%s calls a Run method: no use case calls another (FR-041)", f.Path)
				}
			}
			return true
		})
	}
	if want := len(rosterDomainFiles()) + len(rosterUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d roster files", seen, want)
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-001, FR-033, FR-036, FR-039, FR-041, FR-045 - the closed sets have exactly the variants the spec lists (the cap added the sixth rejection), the rejection classes end in Rejection, and the roster transition is not exported.
// Path: unhappy (duplicate).
func TestRosterClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isRosterPosition": {"RosterAfterCombatant", "RosterAtInitiative"},
		"isRosterRejection": strings.Fields(`RosterEncounterEndedRejection RosterUnknownCombatantRejection RosterDuplicateCombatantRejection
			RosterCombatantIsActiveRejection RosterWouldEmptySideRejection RosterLimitExceededRejection`),
		"isRosterStep":                           {"RosterStepApplied", "RosterStepRejected"},
		"isRosterTransition":                     {"rosterApplied", "rosterRejected"},
		"isAddCombatantToEncounterResult":        {"AddCombatantToEncounterApplied", "AddCombatantToEncounterRejected"},
		"isRemoveCombatantFromEncounterResult":   {"RemoveCombatantFromEncounterApplied", "RemoveCombatantFromEncounterRejected"},
		"isRepositionCombatantInEncounterResult": {"RepositionCombatantInEncounterApplied", "RepositionCombatantInEncounterRejected"},
	}
	for marker, names := range want {
		slices.Sort(names)
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
	for _, name := range markerReceivers(t, "isRosterRejection") {
		if !strings.HasSuffix(name, "Rejection") {
			t.Errorf("%s does not end in Rejection (FR-045)", name)
		}
	}
	have := declared(combatPackage(t))
	if len(have["RosterTransition"]) != 0 {
		t.Errorf("RosterTransition is exported (FR-036)")
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-033, FR-035 - exactly CombatantJoined, CombatantRemoved and CombatantRepositioned embed the roster event base, all declared once in roster_event.go; none implements Event on its own.
// Path: unhappy (duplicate).
func TestRosterEventFamilyHasExactlyThreeMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "rosterEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/roster_event.go" {
							t.Errorf("%s is declared in %s, want combat/roster_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if want := []string{"CombatantJoined", "CombatantRemoved", "CombatantRepositioned"}; !slices.Equal(members, want) {
		t.Errorf("roster event members = %v, want %v", members, want)
	}
	if got := markerReceivers(t, "rosterEvent"); !slices.Equal(got, []string{"rosterEventBase"}) {
		t.Errorf("rosterEvent is implemented by %v, want only the base", got)
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-045, section 6 (Exports) - package combat exports the use cases, commands, results, positions, reasons, rejections, step and events once each, with no generic name, and its package comment names each of them.
// Path: happy.
func TestCombatExportsRosterTypesAndDocumentsThem(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		AddCombatantToEncounter NewAddCombatantToEncounter AddCombatantToEncounterCommand NewAddCombatantToEncounterCommand AddCombatantToEncounterResult
		AddCombatantToEncounterApplied AddCombatantToEncounterRejected
		RemoveCombatantFromEncounter NewRemoveCombatantFromEncounter RemoveCombatantFromEncounterCommand NewRemoveCombatantFromEncounterCommand
		RemoveCombatantFromEncounterResult RemoveCombatantFromEncounterApplied RemoveCombatantFromEncounterRejected
		RepositionCombatantInEncounter NewRepositionCombatantInEncounter RepositionCombatantInEncounterCommand NewRepositionCombatantInEncounterCommand
		RepositionCombatantInEncounterResult RepositionCombatantInEncounterApplied RepositionCombatantInEncounterRejected
		RosterPosition RosterAfterCombatant RosterAtInitiative TiePlacement TiePlacementBeforeTied TiePlacementAfterTied JoinReason JoinReasonSummoned
		JoinReasonArrived RemovalCause RemovalCauseFled RemovalCauseSurrendered RemovalCauseDisappeared RemovalCauseDeparted RosterEvent CombatantJoined
		CombatantRemoved CombatantRepositioned RosterRejection RosterEncounterEndedRejection RosterUnknownCombatantRejection RosterDuplicateCombatantRejection
		RosterCombatantIsActiveRejection RosterWouldEmptySideRejection RosterLimitExceededRejection RosterStep RosterStepApplied RosterStepRejected`)
	enumMembers := strings.Fields(`TiePlacementBeforeTied TiePlacementAfterTied JoinReasonSummoned JoinReasonArrived RemovalCauseFled RemovalCauseSurrendered
		RemovalCauseDisappeared RemovalCauseDeparted`)
	have := declared(combatPackage(t))
	doc := ""
	for _, f := range combatPackage(t).Production() {
		if f.Path == "combat/doc.go" && f.AST.Doc != nil {
			doc = f.AST.Doc.Text()
		}
	}
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
		if !strings.Contains(doc, "["+name+"]") && !strings.HasPrefix(name, "New") && !slices.Contains(enumMembers, name) {
			t.Errorf("the package comment of combat/doc.go does not name [%s] (FR-045)", name)
		}
	}
	for _, generic := range []string{"Applied", "Rejected", "Joined", "Removed", "Repositioned"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-045)", generic)
		}
	}
}
