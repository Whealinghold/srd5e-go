package combat_test

import (
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/combat-foundations.md FR-012 - a CombatantKind variant
// exports only its reference id field and no method.
// Path: per case.
func TestCombatantKindIdentityOnly(t *testing.T) {
	t.Parallel()
	for _, v := range []any{combat.PlayerCharacterCombatant{}, combat.MonsterCombatant{}} {
		typ := reflect.TypeOf(v)
		if typ.NumField() != 1 || typ.Field(0).Name != "ID" || typ.NumMethod() != 0 {
			t.Errorf("%s: want only field ID and no exported method, got %d fields, %d methods",
				typ, typ.NumField(), typ.NumMethod())
		}
	}
}

// Requirement: doc/specs/combat-foundations.md FR-015 - Event declares Kind and
// Combatants (the marker is unexported, so no outside type satisfies it).
// Path: happy.
func TestEventMethodSet(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeFor[combat.Event]()
	var exported []string
	for i := range typ.NumMethod() {
		if m := typ.Method(i); m.IsExported() {
			exported = append(exported, m.Name)
		}
	}
	if !reflect.DeepEqual(exported, []string{"Combatants", "Kind"}) {
		t.Errorf("Event exported methods = %v, want [Combatants Kind]", exported)
	}
}

// Requirement: doc/specs/combat-foundations.md FR-014 - the reference types
// (id, kind, ref files) import no d20, toolbox, combat sibling or internal
// package.
// Path: per file.
func TestReferenceTypesImports(t *testing.T) {
	t.Parallel()
	for _, f := range []string{"combatant_id.go", "combatant_kind.go", "combatant_ref.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			switch p {
			case "strings", "fmt", "github.com/Whealinghold/srd5e-go/srderr",
				"github.com/Whealinghold/srd5e-go/character", "github.com/Whealinghold/srd5e-go/content":
			default:
				t.Errorf("%s imports %s", f, p)
			}
		}
	}
}
