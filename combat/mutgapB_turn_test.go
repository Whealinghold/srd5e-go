package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/initiative-and-actions.md FR-155 - AdditionalActionGrant.Equal compares the zero grant, the feature and the scope kind: a zero grant equals only another zero grant (and never panics against a built one), and grants of different features or scope kinds differ whichever feature sorts first.
// Path: unhappy (rule interaction).
func TestMutgapBAdditionalActionGrantEqual(t *testing.T) {
	t.Parallel()
	scope := func(only bool) combat.AdditionalActionScope {
		if only {
			return luMust(combat.NewOnlyActions(combat.StandardActionAttack))
		}
		return luMust(combat.NewAnyActionBut(combat.StandardActionAttack))
	}
	grant := func(feature string, only bool) combat.AdditionalActionGrant {
		return luMust(combat.NewAdditionalActionGrant(content.FeatureID(feature), scope(only)))
	}
	zero := combat.AdditionalActionGrant{}
	low, high := grant("action_surge", false), grant("haste", false)
	cases := []struct {
		name string
		a, b combat.AdditionalActionGrant
		want bool
	}{
		{"zero equals zero", zero, zero, true},
		{"zero differs from a built grant", zero, low, false},
		{"a built grant differs from zero", low, zero, false},
		{"same grant", low, grant("action_surge", false), true},
		{"the lesser feature against the greater", low, high, false},
		{"the greater feature against the lesser", high, low, false},
		{"same feature and actions, other scope kind", low, grant("action_surge", true), false},
	}
	for _, c := range cases {
		if got := c.a.Equal(c.b); got != c.want {
			t.Errorf("%s: Equal = %v, want %v [FR-155]", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-157 - a OncePerTurn feat use of any combatant ends when the next turn starts, not only the use of the combatant whose turn starts: C's use made during A's turn can be recorded again once B's turn has started.
// Path: unhappy (state error).
func TestMutgapBPerTurnFeatUseOfAnotherCombatantEndsWithTheTurn(t *testing.T) {
	t.Parallel()
	per := character.OncePerTurn{}
	tr, _ := trkApplied(t)(trkABC(t).RecordFeatUse(trkC, "savage_attacker", per))
	if r := trkReason(t)(tr.RecordFeatUse(trkC, "savage_attacker", per)); r == nil {
		t.Fatal("a second use in the same turn was accepted")
	}
	tr, _ = trkAdvance(t, tr) // B's turn starts; C's turn has not
	trkApplied(t)(tr.RecordFeatUse(trkC, "savage_attacker", per))
}

// Requirement: doc/specs/initiative-and-actions.md FR-069, FR-118 - a provoker that Disengaged this turn is judged only when it is the active combatant: another combatant's Opportunity Attack on a seen provoker is accepted when a different combatant (whose identifier sorts before the provoker's) is the one that Disengaged.
// Path: unhappy (rule interaction).
func TestMutgapBDisengageOfAnotherCombatantDoesNotShieldTheProvoker(t *testing.T) {
	t.Parallel()
	tr := trkTake(t, trkABC(t), trkA, trkDisenga) // A is active and Disengaged
	oa := combat.OpportunityAttackReaction{Provoker: trkB, Means: combat.DepartureBySpeed, Visibility: combat.VisibilitySeen}
	if _, ev := trkApplied(t)(tr.TakeReaction(trkC, trkAvail, oa)); len(ev) != 1 || ev[0].Kind() != "reaction_taken" {
		t.Errorf("events = %v, want one reaction_taken", trkKinds(ev))
	}
	own := combat.OpportunityAttackReaction{Provoker: trkA, Means: combat.DepartureBySpeed, Visibility: combat.VisibilitySeen}
	if r := trkReason(t)(tr.TakeReaction(trkC, trkAvail, own)); r != (combat.TurnProvokerDisengagedRejection{}) {
		t.Errorf("reason = %#v, want TurnProvokerDisengagedRejection", r)
	}
}
