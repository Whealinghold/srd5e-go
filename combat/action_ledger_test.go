package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/combat-action-resolution.md FR-112 - a new ledger (the zero value) holds no Attack action and no Help benefit; NewActionLedger restores saved state and refuses a state that is not one of the two variants or holds a zero progress.
// Path: per case.
func TestNewActionLedger(t *testing.T) {
	t.Parallel()
	progress, err := combat.StartAttackActionProgress(cid("a"), allowanceOf(t, 2, 2))
	if err != nil {
		t.Fatal(err)
	}
	help, _ := combat.NewAttackRollHelp(cid("a"), cid("b"))
	helps, err := combat.NewHelpBenefits([]combat.HelpBenefit{help})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, requirement, path string
		state                   combat.AttackActionState
		helps                   combat.HelpBenefits
		ok                      bool
	}{
		{"nil state", resSpec + "FR-112", "unhappy (empty and minimal)", nil, combat.HelpBenefits{}, true},
		{"no Attack action", resSpec + "FR-112", "happy", combat.NoAttackActionInProgress{}, combat.HelpBenefits{}, true},
		{"Attack action in progress", resSpec + "FR-112", "happy", combat.AttackActionInProgress{Progress: progress}, helps, true},
		{"zero progress", resSpec + "FR-112", "unhappy (invalid input)", combat.AttackActionInProgress{}, combat.HelpBenefits{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			l, err := combat.NewActionLedger(c.state, c.helps)
			if c.ok != (err == nil) || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Fatalf("error = %v, want ok=%v [%s]", err, c.ok, c.requirement)
			}
			if c.ok && (!l.HelpBenefits().Equal(c.helps)) {
				t.Errorf("benefits = %v, want %v", l.HelpBenefits().Benefits(), c.helps.Benefits())
			}
		})
	}
	zero := combat.ActionLedger{}
	if _, none := zero.AttackAction().(combat.NoAttackActionInProgress); !none || zero.HelpBenefits().Len() != 0 || !zero.Equal(combat.ActionLedger{}) {
		t.Errorf("the zero ledger = %#v, want no Attack action and no benefit [FR-112]", zero)
	}
	in, _ := combat.NewActionLedger(combat.AttackActionInProgress{Progress: progress}, helps)
	same, _ := combat.NewActionLedger(combat.AttackActionInProgress{Progress: progress}, helps)
	if !in.Equal(same) || in.Equal(zero) || zero.Equal(in) {
		t.Errorf("Equal does not tell ledgers with the same and with other state apart")
	}
	var bad combat.AttackActionState = struct{ combat.AttackActionState }{}
	if _, err := combat.NewActionLedger(bad, combat.HelpBenefits{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a state of another type gives %v, want ErrInvalidInput", err)
	}
}
