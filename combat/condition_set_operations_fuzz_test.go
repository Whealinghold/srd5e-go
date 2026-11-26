package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/conditions.md FR-012, FR-013, FR-020, FR-035, FR-124, testing.md section 10 - any sequence of
// Apply, Remove and ReduceExhaustion never panics, keeps one explicit condition per kind, keeps Exhaustion within 1 to 6,
// yields effective kinds in order without repeats, and returns a set equal to one rebuilt from its parts.
// Path: unhappy (invalid input).
func FuzzConditionSetOperations(f *testing.F) {
	f.Add([]byte{0, 3, 1, 3, 2, 14, 1, 14})
	f.Add([]byte{0, 3, 0, 3, 0, 3, 0, 3, 0, 3, 0, 3, 0, 3, 2, 0})
	f.Add([]byte{0, 8, 0, 9, 1, 14, 3, 200})
	f.Fuzz(func(t *testing.T, ops []byte) {
		s := condSet(t, nil, cD)
		all := allFifteen()
		for i := 0; i+1 < len(ops); i += 2 {
			arg := int(ops[i+1])
			switch ops[i] % 4 {
			case 0:
				s = s.Apply(subject(), all[arg%len(all)]).Set()
			case 1:
				s = s.Remove(subject(), content.ConditionKind(arg%17)).Set()
			case 2:
				if res, err := s.ReduceExhaustion(subject(), arg%8-1); err == nil {
					s = res.Set()
				}
			default:
				s = s.Apply(subject(), condLevel(arg%6+1)).Set()
			}
			seen := map[content.ConditionKind]bool{}
			for _, c := range s.Explicit() {
				if seen[c.Kind()] {
					t.Fatalf("two explicit conditions of kind %v", c.Kind())
				}
				seen[c.Kind()] = true
				if e, ok := c.(combat.ExhaustionCondition); ok && (e.Level.Value() < 1 || e.Level.Value() > 6) {
					t.Fatalf("Exhaustion level %d", e.Level.Value())
				}
			}
			eff := s.Effective()
			if !slices.IsSorted(eff) || len(slices.Compact(slices.Clone(eff))) != len(eff) {
				t.Fatalf("effective kinds %v are not ordered and unique", eff)
			}
			rebuilt := condSet(t, s.Explicit(), s.Immunities()...)
			if !rebuilt.Equal(s) {
				t.Fatalf("a set rebuilt from its parts differs")
			}
		}
	})
}
