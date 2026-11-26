package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A grappler holds a target, the target fails then succeeds an escape check, and the
// host ends the grapple on the target's conditions.
func ExampleEscapeGrappleFromRecord() {
	ctx := context.Background()
	orc, hero := combat.MustCombatantID("orc"), combat.MustCombatantID("hero")
	dc := combat.GrappleRules{}.GrappleDC(3, 2) // 8 + Strength modifier + Proficiency Bonus

	begun, _ := combat.NewBeginGrapple().Run(ctx, combat.BeginGrappleCommand{
		Grappler: orc, Target: hero, Part: combat.MustGraspingHand(0), EscapeDC: dc, RangeFeet: combat.GrappleUnarmedStrikeRangeFeet,
	})
	grapples := begun.(combat.BeginGrappleApplied).Grapples()
	conditions := combat.MustConditionSet([]combat.Condition{combat.Grappled{Grappler: orc}}, nil)

	attempt := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(2))}
	escape, _ := combat.NewEscapeGrappleFromRecord(dicetest.NewFixedRoller(5, 14))
	for range 2 {
		cmd, _ := combat.NewEscapeGrappleFromRecordCommand(grapples, hero, combat.GrappleEscapeOptionStrengthAthletics,
			combat.ConditionActionAvailabilityAvailable, attempt)
		res, _ := escape.Run(ctx, cmd)
		switch r := res.(type) {
		case combat.EscapeGrappleFromRecordHeld:
			fmt.Println("held, action spent:", r.ActionSpent())
		case combat.EscapeGrappleFromRecordEscaped:
			ended := combat.GrappleRules{}.EndOnSet(conditions, hero, r.Ending().Cause()).(combat.GrappleEndOnSetApplied)
			for _, e := range ended.Events() {
				fmt.Println(e.Kind())
			}
			fmt.Println("grapples left:", r.Grapples().Len())
		}
	}
	// Output:
	// held, action spent: true
	// condition_removed
	// grapple_ended
	// grapples left: 0
}
