package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: doc/specs/monster-combatant-setup.md FR-011, FR-005 - a roll of another expression or no choice gives no maximum and only the problem, and a valid choice gives no problem.
// Path: unhappy (invalid input).
func TestMonsterMaximumOnAProblemReturnsOnlyTheProblem(t *testing.T) {
	t.Parallel()
	rat := dice.MustExpression(2, dice.D6(), 0)
	entry, err := content.NewHitPointsEntry(7, rat)
	if err != nil {
		t.Fatal(err)
	}
	other := dice.MustExpression(1, dice.D4(), 0)
	roll, err := dice.NewResult(other, []dice.DieRoll{mustDieRoll(t, dice.D4(), 2)})
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		hp   ResolvedMonsterHitPoints
		want MonsterSetupProblem
	}{"mismatch": {RolledMonsterHitPoints{roll}, RolledExpressionMismatch}, "nil": {nil, SetupArgumentInvalid}} {
		if m, raised, p := monsterMaximum(entry, c.hp); m != 0 || raised || p != c.want {
			t.Errorf("%s: monsterMaximum = %d, %v, %v, want 0, false, %v", name, m, raised, p, c.want)
		}
	}
	if m, raised, p := monsterMaximum(entry, PrintedAverageHitPoints{}); m != 7 || raised || p != 0 {
		t.Errorf("average: monsterMaximum = %d, %v, %v", m, raised, p)
	}
}

func mustDieRoll(t *testing.T, d dice.Die, face int) dice.DieRoll {
	t.Helper()
	r, err := dice.NewDieRoll(d, face)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
