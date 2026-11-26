package combat_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Requirement: doc/specs/spell-outcome-tables.md FR-110, FR-111, FR-073, FR-093, FR-094 - the pure spell_outcome_*.go files import no telemetry, no srderr and no dice.Roller, and name neither the Roller port nor a use case; the use case file imports no character, ability or toolbox package, so it never rolls the Indigo Constitution or Violet Wisdom save, and it constructs no other use case.
// Path: unhappy (forbidden edge).
func TestSpellOutcomeFilesKeepTheirLayers(t *testing.T) {
	t.Parallel()
	pure, err := filepath.Glob("spell_outcome_*.go")
	if err != nil || len(pure) < 6 {
		t.Fatalf("found %d spell_outcome files (%v), want the pure model files", len(pure), err)
	}
	const prefix = "github.com/Whealinghold/srd5e-go/"
	check := func(file string, banned []string, bannedIdents []string) {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			for _, b := range banned {
				if p == prefix+b {
					t.Errorf("%s imports %s [FR-110, FR-111]", file, p)
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				for _, b := range bannedIdents {
					if id.Name == b {
						t.Errorf("%s names %s [FR-073, FR-093]", file, b)
					}
				}
			}
			return true
		})
	}
	for _, file := range pure {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		check(file, []string{"telemetry", "srderr", "character"}, []string{"Roller", "Telemetry", "Constitution", "Wisdom"})
	}
	check("resolve_spell_outcome_table_usecase.go", []string{"character", "ability"}, []string{
		"Constitution", "Wisdom", "NewApplyDamage", "ApplyDamage", "ApplyConditionToSet", "NewResolveSpellEffects", "Event"})
}
