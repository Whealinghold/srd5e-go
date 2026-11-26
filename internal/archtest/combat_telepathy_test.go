package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// telepathyFiles are the production files of telepathy.md: pure domain code, no use case file.
func telepathyFiles() []string {
	return []string{"combat/telepathy.go", "combat/telepathy_event.go", "combat/telepathy_rules.go"}
}

// Requirement: doc/specs/telepathy.md FR-012, FR-021 and section 7 - the telepathy files are silent domain code: they import
// only srderr of this module (no telemetry, dice, content, d20), name no signal constant, and name none of ActionChoice,
// StandardAction or ReactionTaken (starting and ending contact take no action).
// Path: unhappy (forbidden edge).
func TestTelepathyFilesAreSilentDomainCode(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(telepathyFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && p != "srderr" {
				t.Errorf("%s imports %s, which telepathy.md does not allow", f.Path, p)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr") {
				t.Errorf("%s names the signal constant %s: the spec is silent by design (section 7)", f.Path, id.Name)
			}
			if slices.Contains([]string{"ActionChoice", "StandardAction", "ReactionTaken"}, id.Name) {
				t.Errorf("%s names %s, but telepathy takes no action [FR-012]", f.Path, id.Name)
			}
			return true
		})
	}
	if seen != len(telepathyFiles()) {
		t.Fatalf("found %d of %d telepathy files", seen, len(telepathyFiles()))
	}
}

// Requirement: doc/specs/telepathy.md FR-021 - exactly TelepathicContactMade, TelepathicContactBroken and TelepathicContactEnded
// embed the telepathy event base, all declared in telepathy_event.go.
// Path: unhappy (duplicate).
func TestTelepathyEventFamilyHasExactlyThreeMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "telepathyEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/telepathy_event.go" {
							t.Errorf("%s is declared in %s, want combat/telepathy_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if want := []string{"TelepathicContactBroken", "TelepathicContactEnded", "TelepathicContactMade"}; !slices.Equal(members, want) {
		t.Errorf("telepathy event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/telepathy.md section 6 - package combat exports the telepathy types, results, events and functions once each.
// Path: happy.
func TestCombatExportsTelepathyTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		ErrTelepathyRange TelepathyCapability NewTelepathic MustTelepathic ReplyPolicy ContactedMayReply ContactedMayNotReply
		ContactedUnderstanding ContactedUnderstandingNeither ContactedUnderstandingALanguage ContactedUnderstandingTelepathic
		TelepathicContactRejection BreakCause BreakCauseIncapacitated BreakCauseContactedLeftRange BreakCauseTelepathContactedAnother
		TelepathicContactState InContact TelepathyEvent TelepathicContactMade TelepathicContactBroken TelepathicContactEnded
		NewTelepathicContactMade NewTelepathicContactBroken NewTelepathicContactEnded
		InitiateTelepathicContactCommand InitiateTelepathicContactResult InitiateTelepathicContactMade InitiateTelepathicContactRejected
		InitiateTelepathicContactAlreadyMade InitiateTelepathicContact EndTelepathicContactResult EndTelepathicContactApplied
		EndTelepathicContactNoContact EndTelepathicContact ReportTelepathIncapacitatedResult ReportTelepathIncapacitatedBroken
		ReportTelepathIncapacitatedUnchanged ReportTelepathIncapacitated ReportTelepathDistanceResult ReportTelepathDistanceBroken
		ReportTelepathDistanceUnchanged ReportTelepathDistance TelepathicReplyResult TelepathicReplyAllowed TelepathicReplyNotPermitted
		TelepathicReplyNoContact TelepathicReplyNotContacted TelepathicReply`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
}
