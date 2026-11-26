package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// Requirement: doc/specs/readying-a-spell.md FR-013 - combat/held_spell.go is pure and silent: it imports only content, never telemetry, context or character, and names no signal constant.
// Path: unhappy (forbidden edge).
func TestHeldSpellFileImportsOnlyContent(t *testing.T) {
	t.Parallel()
	seen := false
	for _, f := range combatPackage(t).Production() {
		if f.Path != "combat/held_spell.go" {
			continue
		}
		seen = true
		for _, imp := range f.Imports() {
			if imp.Path != "github.com/Whealinghold/srd5e-go/content" {
				t.Errorf("%s imports %s, which readying-a-spell.md FR-013 does not allow", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				for _, prefix := range []string{"metric", "span", "attr", "value", "msg"} {
					if strings.HasPrefix(id.Name, prefix) && len(id.Name) > len(prefix) && id.Name[len(prefix)] >= 'A' && id.Name[len(prefix)] <= 'Z' {
						t.Errorf("%s names the signal constant %s: the spec is silent (section 7)", f.Path, id.Name)
					}
				}
			}
			return true
		})
	}
	if !seen {
		t.Fatal("combat/held_spell.go not found")
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-013, section 6 - the closed sets have exactly the variants the spec lists, exactly three events embed the held-spell event base, and no Concentration state type or operation of spellcasting.md is defined in combat.
// Path: unhappy (duplicate).
func TestHeldSpellClosedSetsAndNames(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isReadyCombatantSpellResult":     {"ReadySpellApplied", "ReadySpellNotAnAction"},
		"isReleaseCombatantSpellResult":   {"ReleaseHeldSpellApplied", "ReleaseHeldSpellNoHeldSpell", "ReleaseHeldSpellNotAReadiedReaction", "ReleaseHeldSpellReactionMismatch"},
		"isDissipateCombatantSpellResult": {"DissipateHeldSpellApplied", "DissipateHeldSpellInvalidCause", "DissipateHeldSpellNoHeldSpell"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
	var members []string
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					for _, fld := range st.Fields.List {
						if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "heldSpellEventBase" {
							members = append(members, ts.Name.Name)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if !slices.Equal(members, []string{"HeldSpellDissipated", "HeldSpellReleased", "SpellHeld"}) {
		t.Errorf("held-spell event members = %v", members)
	}
	d := declared(combatPackage(t))
	for _, name := range []string{"Concentration", "BreakConcentration", "ConcentrationEnded", "BeginConcentration"} {
		if len(d[name]) != 0 {
			t.Errorf("%s is declared in %v: Concentration is spellcasting.md's", name, d[name])
		}
	}
}
