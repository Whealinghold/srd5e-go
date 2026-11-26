package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// grappleFiles are the production files of grappling.md: domain files (grapple_*.go) and use-case files.
func grappleFiles(t *testing.T) (domain, useCase map[string]bool) {
	t.Helper()
	domain, useCase = map[string]bool{}, map[string]bool{}
	for _, f := range combatPackage(t).Production() {
		switch name := f.Path[len("combat/"):]; name {
		case "begin_grapple.go", "release_grapple.go", "escape_grapple_from_record.go", "review_grapples.go",
			"drop_grapple_record.go", "grapple_usecase.go":
			useCase[f.Path] = true
		default:
			if strings.HasPrefix(name, "grapple_") {
				domain[f.Path] = true
			}
		}
	}
	return domain, useCase
}

// Requirement: doc/specs/grappling.md FR-079, FR-097 - the grapple domain files import only content, d20, ability and
// srderr of this module (no telemetry, dice, modifier, character or later combat spec's code); the use-case files add
// dice, modifier and telemetry.
// Path: unhappy (forbidden edge).
func TestGrappleFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	domain, useCase := grappleFiles(t)
	if len(domain) < 8 || len(useCase) != 6 {
		t.Fatalf("found %d domain and %d use-case grapple files", len(domain), len(useCase))
	}
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"content", "d20", "ability", "srderr"}
		switch {
		case useCase[f.Path]:
			allowed = append(allowed, "dice", "modifier", "telemetry")
		case !domain[f.Path]:
			continue
		}
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which grappling.md FR-097 does not allow", f.Path, p)
			}
		}
	}
}

// Requirement: doc/specs/grappling.md FR-081, FR-084 - exactly GrappleStarted, GrappleEnded and GrappleRecordDropped embed
// the grapple event base, and they are declared once, in grapple_event.go; GrappleEnded, GrappleEndCause, GrappleHold,
// GrappleProblem and GrappleRejection are each declared once.
// Path: unhappy (duplicate).
func TestGrappleEventFamilyHasExactlyThreeMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "grappleEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/grapple_event.go" {
							t.Errorf("%s is declared in %s, want combat/grapple_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if want := []string{"GrappleEnded", "GrappleRecordDropped", "GrappleStarted"}; !slices.Equal(members, want) {
		t.Errorf("grapple event members = %v, want %v", members, want)
	}
	d := declared(combatPackage(t))
	for _, name := range []string{"GrappleEnded", "GrappleEndCause", "GrappleHold", "GrappleProblem", "GrappleRejection"} {
		if len(d[name]) != 1 {
			t.Errorf("%s is declared %d times, want once", name, len(d[name]))
		}
	}
}

// Requirement: doc/specs/grappling.md FR-070, FR-072, FR-073, plan-phase-4.md P11 - the signals file pins the five operation
// values and spans, the two own metrics and the end-reason attribute at their catalogue values, and no domain file names
// a signal constant.
// Path: unhappy (invalid input).
func TestGrappleSignalNames(t *testing.T) {
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
		"valueOperationBeginGrapple": "begin_grapple", "spanBeginGrapple": "srd5e.combat.begin_grapple",
		"valueOperationReleaseGrapple": "release_grapple", "spanReleaseGrapple": "srd5e.combat.release_grapple",
		"valueOperationEscapeGrappleFromRecord": "escape_grapple_from_record", "spanEscapeGrappleFromRecord": "srd5e.combat.escape_grapple_from_record",
		"valueOperationReviewGrapples": "review_grapples", "spanReviewGrapples": "srd5e.combat.review_grapples",
		"valueOperationDropGrappleRecord": "drop_grapple_record", "spanDropGrappleRecord": "srd5e.combat.drop_grapple_record",
		"metricGrappleEnded": "srd5e.combat.grapple.ended", "metricGrappleInForce": "srd5e.combat.grapple.in_force",
		"attrGrappleEndReason": "srd5e.combat.grapple.end_reason", "valueGrappleEndConditionEnded": "condition_ended",
		"valueResultApplied": "applied", "valueResultEscaped": "escaped", "valueResultHeld": "held",
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
	for name, value := range got {
		if slices.Contains(supersededSignals(), value) {
			t.Errorf("signals.go constant %s holds the superseded name %s", name, value)
		}
	}
	domain, _ := grappleFiles(t)
	for _, f := range combatPackage(t).Production() {
		if !domain[f.Path] {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr")) {
				t.Errorf("%s names the signal constant %s: the domain is silent (FR-079)", f.Path, id.Name)
			}
			return true
		})
	}
}

// Requirement: doc/specs/grappling.md section 6 - package combat exports the grapple types, results, events and use cases
// once each.
// Path: happy.
func TestCombatExportsGrappleTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		GraspingPart GraspingHand GraspingBodyPart NewGraspingHand MustGraspingHand NewGraspingBodyPart MustGraspingBodyPart
		Grapple NewGrapple MustGrapple GrappleUnarmedStrikeRangeFeet Grapples NewGrapples MustGrapples GrappleOfTarget InGrapple NotInGrapple
		PartState HandAvailability GrappleHold GrappleEndCause GrappleNotPossibleReason GrappleEscapeOption GrappleProblem GrappleProblemError GrappleRejection
		GrappleEnding GrappleTransition GrappleTransitionApplied GrappleTransitionRejected GrappleTransitionInvalid
		GrappleObservation NewGrappleObservation MustGrappleObservation GrappleOption WithGrapplerDead
		GrappleRules GrapplePossibility GrapplePossible GrappleNotPossible GrappleEndOnSetResult GrappleEndOnSetApplied GrappleEndOnSetNotGrappled
		GrappleEvent GrappleStarted GrappleEnded GrappleRecordDropped NewGrappleStarted NewGrappleEnded NewGrappleRecordDropped
		BeginGrapple BeginGrappleCommand BeginGrappleResult BeginGrappleApplied BeginGrappleRejected BeginGrappleInvalid NewBeginGrapple
		ReleaseGrapple ReleaseGrappleCommand ReleaseGrappleResult ReleaseGrappleApplied ReleaseGrappleRejected ReleaseGrappleInvalid NewReleaseGrapple
		EscapeGrappleFromRecord EscapeGrappleFromRecordCommand EscapeGrappleFromRecordResult EscapeGrappleFromRecordEscaped EscapeGrappleFromRecordHeld
		EscapeGrappleFromRecordRejected NewEscapeGrappleFromRecord NewEscapeGrappleFromRecordCommand GrappleEscapeCheck GrappleEscapeRolled GrappleEscapeForced
		ReviewGrapples ReviewGrapplesCommand ReviewGrapplesResult ReviewGrapplesApplied ReviewGrapplesInvalid NewReviewGrapples
		DropGrappleRecord DropGrappleRecordCommand DropGrappleRecordResult DropGrappleRecordApplied DropGrappleRecordRejected NewDropGrappleRecord`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, banned := range []string{"Applied", "Rejected", "Invalid", "Found", "NotFound", "Succeeded", "Grappled"} {
		if banned != "Grappled" && len(have[banned]) != 0 {
			t.Errorf("combat declares the generic name %s", banned)
		}
	}
}
