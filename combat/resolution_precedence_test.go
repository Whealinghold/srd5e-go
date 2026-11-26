package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: per case
// Path: per case.
func TestResolutionPreconditions(t *testing.T) {
	t.Parallel()
	healthy := encABC(t)
	deadA, _ := encStart(t, resDead(t, "a", combat.SideA), resPC(t, "b", combat.SideB))
	deadB, _ := encStart(t, resPC(t, "a", combat.SideA), resDead(t, "b", combat.SideB))
	deadBoth, _ := encStart(t, resDead(t, "a", combat.SideA), resDead(t, "b", combat.SideB))
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		ids                     []combat.CombatantID
		rejection               combat.ResolutionRejection
		problem                 combat.ResolutionProblem
	}{
		{"nothing applies", resSpec + "FR-010", "happy", healthy, []combat.CombatantID{cid("a"), cid("b")}, nil, nil},
		{"no ids", resSpec + "FR-010", "unhappy (empty and minimal)", healthy, nil, nil, nil},
		{"self target", resSpec + "FR-013", "unhappy (rule interaction)", healthy, []combat.CombatantID{cid("a"), cid("a")}, nil, nil},
		{"ended and unknown", resSpec + "4.2.1 FR-010 step 1 before step 2", "unhappy (rule interaction)", resEnded(t, healthy),
			[]combat.CombatantID{cid("ghost")}, combat.EncounterStateRejection{Rejection: combat.EncounterAlreadyEndedRejection{}}, nil},
		{"unknown attacker", resSpec + "4.2.1 FR-010 step 2, FR-011", "unhappy (invalid input)", healthy,
			[]combat.CombatantID{cid("ghost"), cid("b")}, nil, combat.UnknownCombatantProblem{ID: cid("ghost")}},
		{"unknown target", resSpec + "4.2.1 FR-010 step 2", "unhappy (invalid input)", healthy,
			[]combat.CombatantID{cid("a"), cid("ghost")}, nil, combat.UnknownCombatantProblem{ID: cid("ghost")}},
		{"two unknown ids name the first", resSpec + "4.2.1 FR-010 step 2", "unhappy (invalid input)", healthy,
			[]combat.CombatantID{cid("zz"), cid("ghost")}, nil, combat.UnknownCombatantProblem{ID: cid("zz")}},
		{"unknown target and dead attacker", resSpec + "4.2.1 FR-010 step 2 before step 3", "unhappy (rule interaction)", deadA,
			[]combat.CombatantID{cid("a"), cid("ghost")}, nil, combat.UnknownCombatantProblem{ID: cid("ghost")}},
		{"dead attacker", resSpec + "4.2.1 FR-012", "unhappy (state)", deadA, []combat.CombatantID{cid("a"), cid("b")},
			combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("a")}}, nil},
		{"dead target", resSpec + "4.2.1 FR-012", "unhappy (state)", deadB, []combat.CombatantID{cid("a"), cid("b")},
			combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("b")}}, nil},
		{"both dead name the first", resSpec + "4.2.1 FR-010 step 3", "unhappy (rule interaction)", deadBoth, []combat.CombatantID{cid("a"), cid("b")},
			combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("a")}}, nil},
		{"both dead, target named first", resSpec + "FR-010 step 3", "unhappy (rule interaction)", deadBoth, []combat.CombatantID{cid("b"), cid("a")},
			combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("b")}}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rej, prob := (combat.ResolutionPrecedence{}).Preconditions(c.e, c.ids...)
			if rej != c.rejection || prob != c.problem {
				t.Errorf("= %#v, %#v, want %#v, %#v [%s]", rej, prob, c.rejection, c.problem, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestResolutionPrecedenceDeathSavingThrow(t *testing.T) {
	t.Parallel()
	dyingActive, _ := encStart(t, resDying(t, "a", combat.SideA), resPC(t, "b", combat.SideB))
	dyingSecond, _ := encStart(t, resPC(t, "a", combat.SideA), resDying(t, "b", combat.SideB))
	made, _ := encStart(t, resPC(t, "a", combat.SideA), resDying(t, "b", combat.SideB).WithDeathSavingThrowMade(true))
	healthy := encABC(t)
	notDying := func(id string) combat.ResolutionRejection {
		return combat.EncounterStateRejection{Rejection: combat.HitPointRuleRejection{ID: cid(id), Reason: combat.NotDying}}
	}
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		id                      string
		want                    combat.ResolutionRejection
	}{
		{"Dying and active", resSpec + "FR-068", "happy", dyingActive, "a", nil},
		{"Dying and not active", resSpec + "FR-010 step 5a", "unhappy (state)", dyingSecond, "b", combat.DeathSavingThrowOutOfTurnRejection{}},
		{"not Dying and active", resSpec + "FR-010 step 3a", "unhappy (state)", healthy, "a", notDying("a")},
		{"not Dying and not active", resSpec + "4.11.1 FR-010 3a before 5a", "unhappy (rule interaction)", healthy, "b", notDying("b")},
		{"throw made and not active", resSpec + "FR-010 5a before 5b", "unhappy (rule interaction)", made, "b", combat.DeathSavingThrowOutOfTurnRejection{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := (combat.ResolutionPrecedence{}).DeathSavingThrow(c.e, cid(c.id)); got != c.want {
				t.Errorf("= %#v, want %#v [%s]", got, c.want, c.requirement)
			}
		})
	}
}
