package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// returningToLifeFiles are the production files of returning-to-life.md: a pure function and one event family, no use case file.
func returningToLifeFiles() []string {
	return []string{"combat/returning_to_life.go", "combat/revival_event.go"}
}

// Requirement: doc/specs/returning-to-life.md FR-016 and section 7 - the returning-to-life files are silent domain code: they import
// only content and srderr of this module (no telemetry, dice, d20, character), take no context and name no signal constant.
// Path: unhappy (forbidden edge).
func TestReturningToLifeFilesAreSilentDomainCode(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(returningToLifeFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && p != "srderr" && p != "content" {
				t.Errorf("%s imports %s, which returning-to-life.md FR-016 does not allow", f.Path, p)
			}
			if imp.Path == "context" {
				t.Errorf("%s imports context, but Revive is a pure function with no ctx (section 6)", f.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if ok && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr")) {
				t.Errorf("%s names the signal constant %s: the spec owns no signal (section 7)", f.Path, id.Name)
			}
			return true
		})
	}
	if seen != len(returningToLifeFiles()) {
		t.Fatalf("found %d of %d returning-to-life files", seen, len(returningToLifeFiles()))
	}
}

// Requirement: doc/specs/returning-to-life.md FR-016 - exactly CreatureReturnedToLife embeds the revival event base, declared in revival_event.go.
// Path: unhappy (duplicate).
func TestRevivalEventFamilyHasExactlyOneMember(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "revivalEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/revival_event.go" {
							t.Errorf("%s is declared in %s, want combat/revival_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	if want := []string{"CreatureReturnedToLife"}; !slices.Equal(members, want) {
		t.Errorf("revival event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/returning-to-life.md section 5, section 6 - package combat declares the exported names of the spec once each, and
// declares the names of its use case (FR-016: ReviveCombatant, its command and constructor) once each; ReviveBySpell and its types are revival-and-restoration-spells.md and are checked in combat_revival_restoration_test.go.
// Path: unhappy (duplicate).
func TestCombatExportsReturningToLifeTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		RevivalConsent SpiritAccepts SpiritRefuses ReviveCreatureCommand ReviveCreatureResult ReviveCreatureApplied ReviveCreatureNotDead
		ReviveCreatureRefused ReviveCreatureHitPointsBelowOne ReviveCreatureMaximumBelowOne Revive RevivalEvent CreatureReturnedToLife
		ReviveCombatant ReviveCombatantCommand NewReviveCombatant`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
}
