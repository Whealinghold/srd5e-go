package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// hidingFiles are the production files of hiding-searching-and-senses.md: pure domain files, no use case.
func hidingFiles() []string {
	return []string{"combat/light.go", "combat/special_senses.go", "combat/sight.go", "combat/hiding.go",
		"combat/hiding_event.go", "combat/search.go", "combat/study.go"}
}

// Requirement: doc/specs/hiding-searching-and-senses.md section 6, section 7 - the sight, hiding, search and study files are
// silent domain code: they import only content, d20, ability, modifier and srderr of this module (no telemetry, dice,
// character), and name no signal constant.
// Path: unhappy (forbidden edge).
func TestHidingFilesAreSilentDomainCode(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	allowed := []string{"content", "d20", "ability", "modifier", "srderr"}
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(hidingFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which hiding-searching-and-senses.md does not allow", f.Path, p)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr")) {
				t.Errorf("%s names the signal constant %s: the spec is silent by design (section 7)", f.Path, id.Name)
			}
			return true
		})
	}
	if seen != len(hidingFiles()) {
		t.Fatalf("found %d of %d hiding files", seen, len(hidingFiles()))
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-092 - exactly CreatureHid and HidingEnded embed the hiding event base,
// both declared once in hiding_event.go.
// Path: unhappy (duplicate).
func TestHidingEventFamilyHasExactlyTwoMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "hidingEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/hiding_event.go" {
							t.Errorf("%s is declared in %s, want combat/hiding_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if want := []string{"CreatureHid", "HidingEnded"}; !slices.Equal(members, want) {
		t.Errorf("hiding event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md section 6 - package combat exports the sight, hiding, search and study
// types, results, events and services once each, and declares no generic variant name.
// Path: happy.
func TestCombatExportsHidingTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		SpaceVisibility NewSpaceVisibility MustSpaceVisibility SpaceVisibilityReason SpaceVisibilityError SightCheckKind SightCheckEffect LightRules
		SenseRange NewSenseRange MustSenseRange SenseRangeReason SenseRangeError SenseRanges NewSenseRanges SenseReach ColorPerception PerceivedLight
		TremorsenseContact DetectedThing TremorsenseResult TruesightScene NewTruesightScene IllusionAppearance IllusionSave TrueFormPerception
		EtherealView IllusionFacts TrueFormFacts SpecialSenseRules SightScene NewSightScene SightRules
		HideCoverDegree EnemySighting HideScene NewHideScene HideBlock HideCheckSpec HideAssessment HideAvailable HideBlocked HiddenState NewHiddenState
		HidingStatus HidingStatusHidden HidingStatusNotHidden HideResolution HideSucceeded HideCheckFailed HideNotRolled HidingActivity HidingEffect
		HidingFinder HidingChange HideRules HidingEvent CreatureHid HidingEnded NewCreatureHid NewHidingEnded
		SearchTarget SearchProximity SearchReveal SearchSkillAssessment SuggestedSearchSkill OffTableSearchSkill InvalidSearchSkill SearchRules
		KnowledgeTopic StudySubject StudyTopic StudyCreature StudySkillAssessment SuggestedStudySkill InvalidStudySkill StudyRules`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
}
