package archtest

import (
	"go/ast"
	"go/token"
	"strings"
)

// RuleCombatOperationPairing names the scan of the operation literals of package combat: every
// combatOperation{valueOperationX, spanX} pairs identifiers with the same suffix X, and when
// combat/signals.go is present both constants are declared there.
const RuleCombatOperationPairing = "combat-operation-pairing"

// signalConstNames returns every constant declared in combat/signals.go.
func signalConstNames(files []File) (names map[string]bool, found bool) {
	names = map[string]bool{}
	for _, f := range files {
		if f.Path != "combat/signals.go" {
			continue
		}
		found = true
		for _, d := range f.AST.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.CONST {
				for _, s := range g.Specs {
					for _, n := range s.(*ast.ValueSpec).Names {
						names[n.Name] = true
					}
				}
			}
		}
	}
	return names, found
}

// checkOperationPairing reports a combatOperation literal whose value and span constants differ in suffix
// or are not declared in combat/signals.go. It guards against a copy-pasted use case that records another
// operation's signals.
func checkOperationPairing(files []File) []Violation {
	consts, haveSignals := signalConstNames(files)
	var vs []Violation
	for _, f := range files {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isIdent(lit.Type, "combatOperation") || len(lit.Elts) != 2 {
				return true
			}
			v, ok1 := lit.Elts[0].(*ast.Ident)
			s, ok2 := lit.Elts[1].(*ast.Ident)
			if (!ok1 || !signalName(v.Name, "valueOperation")) && (!ok2 || !signalName(s.Name, "span")) {
				return true // a constructor or helper building the literal from its own parameters or fields
			}
			switch {
			case !ok1 || !ok2:
				vs = append(vs, at(f, lit.Pos(), RuleCombatOperationPairing, "combatOperation takes a valueOperation... and a span... constant"))
			case !signalName(v.Name, "valueOperation") || !signalName(s.Name, "span") ||
				strings.TrimPrefix(v.Name, "valueOperation") != strings.TrimPrefix(s.Name, "span"):
				vs = append(vs, at(f, lit.Pos(), RuleCombatOperationPairing, "%s and %s do not name the same operation", v.Name, s.Name))
			case haveSignals && (!consts[v.Name] || !consts[s.Name]):
				vs = append(vs, at(f, lit.Pos(), RuleCombatOperationPairing, "%s or %s is not declared in combat/signals.go", v.Name, s.Name))
			}
			return true
		})
	}
	return vs
}

// signalName reports whether name is prefix followed by a capitalised suffix, as valueOperationX and spanX are.
func signalName(name, prefix string) bool {
	rest, ok := strings.CutPrefix(name, prefix)
	return ok && rest != "" && rest[0] >= 'A' && rest[0] <= 'Z'
}
