package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/spell-outcome-tables.md FR-091, FR-103 - a roller failure at ANY roll of a table, not only the first or the second, is returned as a *srderr.PortError with a nil result, and no roll is requested after the failing one (mutation gap: the error branches of the later rolls, such as the Mishap damage, the Off Target distance and direction, the Confusion pick and the roll-again chain, survived).
// Path: unhappy (failure of collaborator, every roll).
func TestResolveSpellOutcomeTableFailsAtEveryRoll(t *testing.T) {
	t.Parallel()
	threeCreatures := luMust(combat.NewTeleportingCreatures(otIDs("caster", "ally", "mule")))
	twelve := otRepeat(12, 3)
	cases := []struct {
		name  string
		cmd   combat.ResolveSpellOutcomeTableCommand
		faces []int // nil answers every roll with its minimum
	}{
		{"teleport Mishap then Off Target", combat.RollTeleportOutcomeCommand{Subject: threeCreatures, Familiarity: combat.ViewedOnceOrDescribed},
			[]int{43, 1, 2, 3, 4, 5, 6, 7, 8, 9, 54, 12, 11, 5}},
		{"teleport Off Target", combat.RollTeleportOutcomeCommand{Subject: threeCreatures, Familiarity: combat.VeryFamiliar}, []int{14, 3, 4, 2}},
		{"confusion attacks one of three", combat.RollConfusionBehaviorCommand{CreaturesWithinReach: otIDs("a", "b", "c")}, []int{7, 2}},
		{"confusion moves", combat.RollConfusionBehaviorCommand{}, []int{1, 3}},
		{"blink", combat.RollBlinkTurnEndCommand{Location: combat.BlinkOnOtherPlane}, []int{4}},
		{"reincarnation after two rerolls", combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid}, []int{1, 1, 4}},
		{"prismatic spray with two rays", combat.RollPrismaticSprayCommand{SaveDC: 15, Targets: otTargets("a", "b")},
			slices.Concat([]int{8, 3, 5}, twelve, []int{5}, twelve, []int{15})},
		{"prismatic wall", combat.ResolvePrismaticWallCrossingCommand{SaveDC: 15, Creature: combat.PrismaticSaveTarget{ID: "a", Attempt: otAttempt()}}, nil},
	}
	cause := errors.New("roller down")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for failAt := 0; ; failAt++ {
				calls := 0
				var base combat.Roller = dicetest.NewMinimumRoller()
				if c.faces != nil {
					base = dicetest.NewFixedRoller(c.faces...)
				}
				roller := dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
					calls++
					if calls-1 == failAt {
						return dice.Result{}, cause
					}
					return base.Roll(e)
				})
				res, err := otUse(t, roller).Resolve(context.Background(), c.cmd)
				if err == nil {
					if calls > failAt {
						t.Fatalf("roll %d was requested but the failure was not returned", failAt)
					}
					if failAt == 0 {
						t.Fatal("the command made no roll")
					}
					return // every roll of the command has been failed once
				}
				var pe *srderr.PortError
				if !errors.As(err, &pe) || !errors.Is(err, cause) || res != nil || calls != failAt+1 {
					t.Fatalf("failing roll %d: err = %v, result = %v, %d rolls; want a PortError wrapping the cause, a nil result and no roll after the failure", failAt, err, res, calls)
				}
			}
		})
	}
}
