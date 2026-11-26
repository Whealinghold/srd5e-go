package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// monsterSpellcastingFiles are the production files of monster-spellcasting.md.
func monsterSpellcastingFiles() []string {
	var out []string
	for _, n := range []string{"caster", "state", "notes", "request", "event", "outcome", "rules"} {
		out = append(out, "combat/monster_spellcasting_"+n+".go")
	}
	return out
}

// Requirement: doc/specs/monster-spellcasting.md FR-036, observability.md section 3 - the monster_spellcasting*.go files import only ability, character, content, d20, modifier and the standard library's math, slices, strings and unicode; never telemetry, context, dice, srderr or a use-case package, and name no signal constant; no file with that prefix is missing from the list.
// Path: unhappy (forbidden edge).
func TestMonsterSpellcastingFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	allowed := []string{"ability", "character", "content", "d20", "modifier"}
	std := []string{"math", "slices", "strings", "unicode"}
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if strings.HasPrefix(f.Path, "combat/monster_spellcasting") && !slices.Contains(monsterSpellcastingFiles(), f.Path) {
			t.Errorf("%s is a monster spellcasting file missing from the list", f.Path)
		}
		if !slices.Contains(monsterSpellcastingFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				if !slices.Contains(allowed, p) {
					t.Errorf("%s imports %s, which monster-spellcasting.md does not allow", f.Path, p)
				}
				continue
			}
			if !slices.Contains(std, imp.Path) {
				t.Errorf("%s imports %s, which monster-spellcasting.md does not allow", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				for _, prefix := range []string{"metric", "span", "attr", "value", "msg"} {
					if strings.HasPrefix(id.Name, prefix) && len(id.Name) > len(prefix) && id.Name[len(prefix)] >= 'A' && id.Name[len(prefix)] <= 'Z' {
						t.Errorf("%s names the signal constant %s: the spec is silent (FR-036)", f.Path, id.Name)
					}
				}
			}
			return true
		})
	}
	if seen != len(monsterSpellcastingFiles()) {
		t.Fatalf("found %d of %d monster spellcasting files", seen, len(monsterSpellcastingFiles()))
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-034 - exactly the five spellcasting events embed the family base, all declared in monster_spellcasting_event.go.
// Path: unhappy (duplicate).
func TestMonsterSpellcastingEventFamilyHasExactlyFiveMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "monsterSpellcastingEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/monster_spellcasting_event.go" {
							t.Errorf("%s is declared in %s, want combat/monster_spellcasting_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	want := []string{"MonsterCastingFailed", "MonsterCastingStarted", "MonsterConcentrationBegan", "MonsterConcentrationEnded", "MonsterSpellCast"}
	if !slices.Equal(members, want) {
		t.Errorf("spellcasting event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/monster-spellcasting.md section 5 - the closed sets have exactly the variants the spec lists.
// Path: unhappy (duplicate).
func TestMonsterSpellcastingClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isMonsterCastRequest": {"MonsterListCast", "MonsterPartCast"},
		"isMonsterCastOutcome": {"MonsterCastPerformed", "MonsterCastRefused", "MonsterCastStarted"},
		"isComponentWaiver":    {"ComponentStatementUnrecognized", "ComponentsWaived"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-037 - package combat exports the spellcasting types, rules and events once each, declares no use case or port for them, and no generic variant name.
// Path: happy.
func TestCombatExportsMonsterSpellcastingTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		MonsterSpellcastingRules MonsterSpellcaster NewMonsterSpellcaster MonsterSpellcasterForParts MonsterSpellcastingState MonsterSpellcastingStateOption
		WithMonsterConcentration WithMonsterCasting NewMonsterSpellcastingState MonsterCastRequest MonsterListCast NewMonsterListCast MonsterPartCast NewMonsterPartCast
		PartSpellcasting PartSpellcastingFields NewPartSpellcasting MonsterCastContext NewMonsterCastContext MonsterSpellRestrictions ComponentWaiver ComponentsWaived
		ComponentStatementUnrecognized ParseComponentWaiver MonsterCastReport TargetLimit TargetLimitSelfOnly TargetLimitUnrestricted MonsterCastOutcome
		MonsterCastPerformed MonsterCastStarted MonsterCastRefused MonsterCastRefusal MonsterSpellcastingTransition MonsterSpellcastingEvent MonsterSpellCast
		MonsterCastingStarted MonsterConcentrationBegan MonsterConcentrationEnded MonsterCastingFailed`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Refused", "Applied", "Performed", "Started", "StateOption", "ListCast", "PartCast"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-037)", generic)
		}
	}
	for _, name := range []string{"NewMonsterSpellcasting", "MonsterSpellcastingUseCase", "MonsterSpellcastingPort"} {
		if len(have[name]) != 0 {
			t.Errorf("combat declares %s: monster spellcasting has no use case or port (FR-036)", name)
		}
	}
}
