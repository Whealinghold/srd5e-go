package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-010, FR-066, Table A - the constructor takes the plain counts the owners
// give: A01 to A18; general 0 and pact below general are rejected.
// Path: per case.
func TestNewAttackAllowance(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		general, pact           int
		ok                      bool
	}{
		{"A01 no Extra Attack", `SRD 5.2.1, "Attack [Action]", p. 177`, "unhappy (empty case)", 1, 1, true},
		{"A02 Extra Attack", `SRD 5.2.1, "Level 5: Extra Attack", p. 48`, "happy", 2, 2, true},
		{"A05 Two Extra Attacks", `SRD 5.2.1, "Level 11: Two Extra Attacks", p. 48`, "happy", 3, 3, true},
		{"A06 Three Extra Attacks", `SRD 5.2.1, "Level 20: Three Extra Attacks", p. 48`, "happy", 4, 4, true},
		{"A10 Thirsting Blade", `SRD 5.2.1, "Thirsting Blade", p. 74`, "happy", 1, 2, true},
		{"A15 Thirsting and Devouring Blade", `SRD 5.2.1, "Devouring Blade", p. 72`, "happy", 1, 3, true},
		{"A16 general zero", "package design", "unhappy (invalid input)", 0, 1, false},
		{"A17 pact below general", "package design", "unhappy (invalid input)", 2, 1, false},
		{"A18 pact above general, both above 1", `SRD 5.2.1, "Thirsting Blade", p. 74`, "happy (boundary)", 2, 3, true},
		{"negative general", "package design", "unhappy (invalid input)", -1, 5, false},
		{"both zero", "package design", "unhappy (invalid input)", 0, 0, false},
		{"largest counts", "plan P10 cap", "unhappy (boundary)", 256, 256, true},
		{"counts past the cap", "plan P10 cap plus one", "unhappy (boundary)", math.MaxInt, math.MaxInt, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a, err := combat.NewAttackAllowance(c.general, c.pact)
			if (err == nil) != c.ok || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) || a.Valid() != c.ok {
				t.Fatalf("NewAttackAllowance(%d, %d) = %v, %v, want ok %v [%s]", c.general, c.pact, a, err, c.ok, c.requirement)
			}
			if c.ok && (a.GeneralAttacks() != c.general || a.PactWeaponAttacks() != c.pact) {
				t.Errorf("counts = %d, %d, want %d, %d", a.GeneralAttacks(), a.PactWeaponAttacks(), c.general, c.pact)
			}
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-002 - the baseline is one attack for every means; the zero allowance is invalid.
// Path: unhappy (empty case).
func TestBaselineAttackAllowance(t *testing.T) {
	t.Parallel()
	b := combat.BaselineAttackAllowance()
	if b.GeneralAttacks() != 1 || b.PactWeaponAttacks() != 1 || !b.Valid() || b != allowanceOf(t, 1, 1) {
		t.Errorf("baseline = %d, %d", b.GeneralAttacks(), b.PactWeaponAttacks())
	}
	if (combat.AttackAllowance{}).Valid() {
		t.Error("the zero allowance is valid")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-010 - the constructor never panics and keeps its invariant for any pair of integers.
// Path: unhappy (invalid input).
func FuzzNewAttackAllowance(f *testing.F) {
	for _, s := range [][2]int{{1, 1}, {0, 0}, {2, 1}, {math.MaxInt, math.MinInt}, {math.MinInt, math.MaxInt}, {1, 3}, {256, 256}, {1, 257}, {257, 257}} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, general, pact int) {
		a, err := combat.NewAttackAllowance(general, pact)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || general >= 1 && pact >= general && pact <= 256 { // 256 is the plan P10 cap (LimitExceededError)
				t.Fatalf("NewAttackAllowance(%d, %d) rejected a valid pair: %v", general, pact, err)
			}
			return
		}
		if !a.Valid() || a.GeneralAttacks() < 1 || a.PactWeaponAttacks() < a.GeneralAttacks() {
			t.Fatalf("NewAttackAllowance(%d, %d) built %v", general, pact, a)
		}
	})
}
