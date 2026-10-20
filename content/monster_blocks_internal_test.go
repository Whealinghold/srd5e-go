package content

import "testing"

// Requirement: doc/specs/content-monsters.md FR-024, FR-025 - a damage type entry
// anywhere after a condition entry is a constraintViolation, even with a
// described entry between them.
// Path: unhappy.
func TestDefenseIssuesConditionStaysSeen(t *testing.T) {
	t.Parallel()
	cond, err := NewConditionDefense(ConditionCharmed, "")
	if err != nil {
		t.Fatal(err)
	}
	desc, err := NewDescribedDefense("text")
	if err != nil {
		t.Fatal(err)
	}
	dmg, err := NewDamageTypeDefense(DamageFire, "")
	if err != nil {
		t.Fatal(err)
	}
	got := defenseIssues("immunities", []DefenseEntry{cond, desc, dmg}, true)
	if len(got) != 1 || got[0].kind != ProblemConstraintViolation {
		t.Errorf("issues = %v, want one constraintViolation at the damage entry", got)
	}
}
