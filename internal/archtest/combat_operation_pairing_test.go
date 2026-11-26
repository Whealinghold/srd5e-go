package archtest_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

// Requirement: doc/rules/architecture.md section 9, rule combat-operation-pairing - a combatOperation literal pairs valueOperationX with spanX, both declared in combat/signals.go.
// Path: per case.
func TestCombatOperationPairingScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatOperationPairing
	const sig = "package combat\n\nconst (\n\tvalueOperationA = \"a\"\n\tspanA = \"s.a\"\n\tvalueOperationB = \"b\"\n\tspanB = \"s.b\"\n)\n"
	use := func(signals, body string) map[string]string {
		files := map[string]string{"combat/a.go": "package combat\n\nvar _ = " + body + "\n"}
		if signals != "" {
			files["combat/signals.go"] = signals
		}
		return files
	}
	runRuleCases(t, rule, []ruleCase{
		{"a matching pair", "rule", "happy", use(sig, "combatOperation{valueOperationA, spanA}"), ""},
		{"no signals file in the fixture", "rule", "happy", use("", "combatOperation{valueOperationA, spanA}"), ""},
		{"parameters of a helper are no hit", "rule", "happy", use(sig, "combatOperation{value, span}"), ""},
		{"keyed fields are no hit", "rule", "happy", use(sig, "combatOperation{value: v, spanName: s}"), ""},
		{"another type is no hit", "rule", "happy", use(sig, "other{valueOperationA, spanB}"), ""},
		{"a span of another operation", "rule", "unhappy (invalid input)", use(sig, "combatOperation{valueOperationA, spanB}"), rule},
		{"a value of another operation", "rule", "unhappy (invalid input)", use(sig, "combatOperation{valueOperationB, spanA}"), rule},
		{"a literal argument", "rule", "unhappy (invalid input)", use(sig, "combatOperation{\"a\", spanA}"), rule},
		{"wrong prefixes", "rule", "unhappy (invalid input)", use(sig, "combatOperation{attrA, spanA}"), rule},
		{"a constant missing from signals.go", "rule", "unhappy (invalid input)", use(sig, "combatOperation{valueOperationC, spanC}"), rule},
	})
}
