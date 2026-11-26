package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// grappledCrowd is the roster of the event-cap tests: the leading combatants, the grappler "g" and n creatures that
// it grapples. When "g" dies or becomes Incapacitated in a call, each grappled creature adds one grapple check to
// the events of the call, so n sets the event count.
func grappledCrowd(tb testing.TB, n int, lead []combat.Combatant, g combat.Combatant) combat.Encounter {
	tb.Helper()
	cs := append(lead, g)
	for i := range n {
		cs = append(cs, resPC(tb, fmt.Sprintf("t%04d", i), combat.SideA, combat.Grappled{Grappler: cid("g")}))
	}
	return resStartWith(tb, nil, cs...)
}

// Requirement: doc/specs/combat-action-resolution.md FR-098, plan-phase-4.md P10 - one call produces at most 4096 events, in every use case that can reach the cap: exactly 4096 events complete and one more is an invalid result carrying limit_exceeded, with the encounter unchanged and no event. Only a call whose events grow with the roster can reach it (a grappler that dies adds a grapple check per grappled creature); the advance, take-action, move and stabilize calls are bounded by their per-call lists and take no such check (combat-action-resolution.md section 1, Limits).
// Path: per case.
func TestUseCaseEventCap(t *testing.T) {
	t.Parallel()
	dying := func(tb testing.TB) combat.Combatant {
		tb.Helper()
		return encC(tb, "g", combat.SideB, dyingStatus(tb, 20, 0, 2))
	}
	cases := []struct {
		name, requirement, path string
		// run runs the use case against a roster of n grappled creatures and reports the events, the problem of
		// an invalid result (nil for a completed one) and whether the encounter that came back is the one that went in.
		run func(t *testing.T, n int) (events int, problem combat.ResolutionProblem, unchanged bool)
	}{
		{"RollCombatantDeathSavingThrow", resSpec + "FR-098", "unhappy (boundary)", func(t *testing.T, n int) (int, combat.ResolutionProblem, bool) {
			t.Helper()
			e := grappledCrowd(t, n, nil, dying(t))
			res, _, err := resThrow(t, e, "g", nil, nil, 1)
			if err != nil {
				t.Fatal(err)
			}
			if inv, ok := res.(combat.RollCombatantDeathSavingThrowInvalid); ok {
				return len(inv.Events()), inv.Problem(), inv.Encounter().Equal(e)
			}
			return len(resAs[combat.RollCombatantDeathSavingThrowCompleted](t, res).Events()), nil, true
		}},
		{"PerformUnarmedStrike", resSpec + "FR-098", "unhappy (boundary)", func(t *testing.T, n int) (int, combat.ResolutionProblem, bool) {
			t.Helper()
			e := grappledCrowd(t, n, []combat.Combatant{resPC(t, "a", combat.SideA)}, encPC(t, "g", combat.SideB, 1, 1))
			p := resStrikeParams(t, e, "a", "g", combat.UnarmedStrikeDamageOption{ArmorClass: 10})
			cmd, err := combat.NewPerformUnarmedStrikeCommand(p)
			if err != nil {
				t.Fatal(err)
			}
			uc, err := combat.NewPerformUnarmedStrike(newFixed(15))
			if err != nil {
				t.Fatal(err)
			}
			res, err := uc.Run(ctxBackground(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			if inv, ok := res.(combat.PerformUnarmedStrikeInvalid); ok {
				return len(inv.Events()), inv.Problem(), inv.Encounter().Equal(e) && inv.Ledger().Equal(combat.ActionLedger{})
			}
			return len(resAs[combat.PerformUnarmedStrikeCompleted](t, res).Events()), nil, true
		}},
		{"ResolveStatBlockSave", resSpec + "FR-098, FR-166", "unhappy (boundary)", func(t *testing.T, n int) (int, combat.ResolutionProblem, bool) {
			t.Helper()
			e := grappledCrowd(t, n, []combat.Combatant{encPC(t, luSelf, combat.SideA, 20, 20)}, encPC(t, "g", combat.SideB, 5, 5))
			p := sbSaveParams(t, e, nil, 40)
			p.TargetID = cid("g")
			res, _ := sbSaveRun(t, p, 5)
			if inv, ok := res.(combat.ResolveStatBlockSaveInvalid); ok {
				return len(inv.Events()), inv.Problem(), inv.Encounter().Equal(e)
			}
			return len(resAs[combat.ResolveStatBlockSaveCompleted](t, res).Events()), nil, true
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Each grappled creature adds one event, so the count of the smallest crowd gives the rest.
			base, _, _ := c.run(t, 1)
			base--
			if events, problem, _ := c.run(t, 4096-base); events != 4096 || problem != nil {
				t.Errorf("at the cap: %d events, problem %v, want 4096 and none [%s]", events, problem, c.requirement)
			}
			events, problem, unchanged := c.run(t, 4096-base+1)
			if problem != (combat.LimitExceededProblem{}) || events != 0 || !unchanged {
				t.Errorf("cap plus one: %d events, problem %v, unchanged %v, want limit_exceeded, no event, the encounter as it came in [%s]", events, problem, unchanged, c.requirement)
			}
		})
	}
}
