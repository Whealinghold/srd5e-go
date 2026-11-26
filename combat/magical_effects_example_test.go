package combat_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// ExampleAreaMembership_Assess shows the host reporting a location inside a Sphere whose only line from the origin
// passes through Total Cover: the location is not in the area.
func ExampleAreaMembership_Assess() {
	facts, _ := combat.NewOtherLocationFacts(combat.InsideShape, []combat.Cover{combat.TotalCover})
	switch r := (combat.AreaMembership{}).Assess(facts).(type) {
	case combat.AreaMembershipIncluded:
		fmt.Println("in the area")
	case combat.AreaMembershipExcluded:
		fmt.Println("not in the area:", r.Reason)
	case combat.AreaMembershipNoLinesConsidered:
		fmt.Println("no lines considered")
	}
	// Output: not in the area: all_lines_blocked
}

// ExampleShapeShifting_Shift shows a polymorphed creature keeping a curse unless the effect says otherwise.
func ExampleShapeShifting_Shift() {
	curse, _ := combat.NewOngoingEffect("c1", combat.OngoingEffectCurse, mustLabeled("hag_curse"))
	effects, _ := combat.NewOngoingEffects([]combat.OngoingEffect{curse})
	spell, _ := combat.SpellSource(content.MustSpellID("polymorph"))
	res, _ := (combat.ShapeShifting{}).Shift(combat.ShiftShapeCommand{Creature: combat.MustCombatantID("hero"), Effects: effects, Effect: spell})
	applied := res.(combat.ShiftShapeApplied)
	fmt.Println(applied.State().IsShifted(), applied.Effects().Len(), applied.Events()[0].Kind())
	// Output: true 1 shape_shifted
}

func mustLabeled(id string) combat.EffectSource {
	s, _ := combat.LabeledMagicalSource(content.MustFeatureID(id))
	return s
}
