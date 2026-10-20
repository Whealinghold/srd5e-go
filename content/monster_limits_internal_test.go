package content

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// Requirement: doc/specs/content-monsters.md FR-095 - the decoder uses the limit constants and has no numeric
// literal for a limit that has one: the only integer literals of the monster decoder files are 0 and 1 (an
// index, a count, or a lower bound of one).
// Path: unhappy (rule interaction).
func TestMonsterDecoderHasNoLimitLiterals(t *testing.T) {
	t.Parallel()
	for _, file := range []string{"decode_monster.go", "decode_monster_parts.go"} {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.INT && lit.Value != "0" && lit.Value != "1" {
				t.Errorf("%s: integer literal %s at %v; use a monster limit constant [FR-095]", file, lit.Value, lit.Pos())
			}
			return true
		})
	}
}
