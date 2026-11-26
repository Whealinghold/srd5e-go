package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: doc/specs/monster-combatant-setup.md FR-008 to FR-011, table 4.2a - the maximum is the printed average, or the rolled total of the printed dice and flat modifier raised to 1 when it is below 1 (flagged), and a roll of another expression is rejected (SRD 5.2.1, "Hit Points", p. 255; "Hit Point Maximum of 0", p. 17).
// Path: per case.
func TestMonsterSetupHitPoints(t *testing.T) {
	t.Parallel()
	aboleth := dice.MustExpression(20, dice.D10(), 40)
	rat := dice.MustExpression(2, dice.D6(), 0)
	bat := dice.MustExpression(1, dice.D4(), -1)
	minus := dice.MustExpression(4, dice.D8(), -4)
	ones := func(n int) []int { return slices.Repeat([]int{1}, n) }
	cases := []struct {
		name, requirement, path string
		monster                 content.Monster
		choice                  combat.ResolvedMonsterHitPoints
		want                    int
		raised, rejected        bool
	}{
		{"printed average", "FR-008", "happy", msHitPoints(t, 150, aboleth), combat.PrintedAverageHitPoints{}, 150, false, false},
		{"rolled, all ones", "FR-009", "unhappy (boundary)", msHitPoints(t, 150, aboleth), msRolled(t, aboleth, ones(20)...), 60, false, false},
		{"rolled, all tens", "FR-009", "unhappy (boundary)", msHitPoints(t, 150, aboleth), msRolled(t, aboleth, slices.Repeat([]int{10}, 20)...), 240, false, false},
		{"rolled, ordinary", "FR-009", "happy", msHitPoints(t, 7, rat), msRolled(t, rat, 3, 4), 7, false, false},
		{"rolled to zero", "FR-010", "unhappy (boundary)", msHitPoints(t, 1, bat), msRolled(t, bat, 1), 1, true, false},
		{"rolled to one", "FR-010", "unhappy (boundary)", msHitPoints(t, 1, bat), msRolled(t, bat, 2), 1, false, false},
		{"rolled to three", "FR-010", "unhappy (boundary)", msHitPoints(t, 1, bat), msRolled(t, bat, 4), 3, false, false},
		{"average of a minus block", "FR-008", "unhappy (empty and minimal)", msHitPoints(t, 1, bat), combat.PrintedAverageHitPoints{}, 1, false, false},
		{"minus block, all ones", "FR-010", "unhappy (boundary)", msHitPoints(t, 14, minus), msRolled(t, minus, 1, 1, 1, 1), 1, true, false},
		{"wrong dice", "FR-011", "unhappy (invalid input)", msHitPoints(t, 1, bat), msRolled(t, rat, 1, 1), 0, false, true},
		{"wrong modifier", "FR-011", "unhappy (invalid input)", msHitPoints(t, 7, rat), msRolled(t, dice.MustExpression(2, dice.D6(), 1), 1, 1), 0, false, true},
		{"wrong die", "FR-011", "unhappy (invalid input)", msHitPoints(t, 7, rat), msRolled(t, dice.MustExpression(2, dice.D8(), 0), 1, 1), 0, false, true},
		{"wrong count", "FR-011", "unhappy (invalid input)", msHitPoints(t, 7, rat), msRolled(t, dice.MustExpression(3, dice.D6(), 0), 1, 1, 1), 0, false, true},
		{"the zero roll", "FR-011", "unhappy (invalid input)", msHitPoints(t, 7, rat), combat.RolledMonsterHitPoints{}, 0, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := setupSpec + c.requirement
			s, err := combat.NewMonsterSetup(c.monster, c.choice)
			if c.rejected {
				var rej *combat.MonsterSetupError
				if !errors.As(err, &rej) || !slices.Equal(rej.Problems(), []combat.MonsterSetupProblem{combat.RolledExpressionMismatch}) {
					t.Errorf("err = %v, want a rejection with RolledExpressionMismatch [%s]", err, req)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewMonsterSetup: %v [%s]", err, req)
			}
			hp := s.HitPointStatus().HitPoints()
			if hp.Maximum() != c.want || hp.Current() != c.want || hp.Temporary() != 0 || s.HitPointsRaisedToOne() != c.raised {
				t.Errorf("Hit Points = %d/%d (+%d), raised %v, want %d/%d (+0), raised %v [%s]",
					hp.Current(), hp.Maximum(), hp.Temporary(), s.HitPointsRaisedToOne(), c.want, c.want, c.raised, req)
			}
			if _, alive := s.HitPointStatus().LifeState().(combat.Alive); !alive {
				t.Errorf("life state = %T, want Alive (FR-012)", s.HitPointStatus().LifeState())
			}
		})
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-013, FR-014 - one setup is one creature: two setups of one block with different rolls differ; the Hit Dice by Size table is not enforced, so a Gargantuan block with a d6 is accepted unchanged (SRD 5.2.1, "Hit Points", "Hit Dice by Size", p. 255).
// Path: unhappy (rule interaction).
func TestMonsterSetupCreaturesRollSeparatelyAndHitDiceBySizeIsNotEnforced(t *testing.T) {
	t.Parallel()
	rat := dice.MustExpression(2, dice.D6(), 0)
	m := msHitPoints(t, 7, rat)
	a := luMust(combat.NewMonsterSetup(m, msRolled(t, rat, 1, 2)))
	b := luMust(combat.NewMonsterSetup(m, msRolled(t, rat, 5, 6)))
	if a.Equal(b) || a.HitPointStatus().HitPoints().Maximum() != 3 || b.HitPointStatus().HitPoints().Maximum() != 11 {
		t.Errorf("two creatures of one block rolled %v and %v, want 3 and 11 (FR-013)", a.HitPointStatus().HitPoints(), b.HitPointStatus().HitPoints())
	}
	gargantuan := msMonster(t, func(f *content.MonsterFields) {
		f.Sizes = []content.CreatureSize{content.SizeGargantuan}
		f.HitPoints = luMust(content.NewHitPointsEntry(70, dice.MustExpression(20, dice.D6(), 0)))
	})
	s, err := combat.NewMonsterSetup(gargantuan, combat.PrintedAverageHitPoints{})
	if err != nil || s.Size() != content.SizeGargantuan || s.Monster().HitPoints().Dice().Die() != dice.D6() {
		t.Errorf("a Gargantuan block with d6 dice = %v, %v, want it accepted unchanged (FR-014)", s.Size(), err)
	}
}
