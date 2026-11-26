package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/dice"
)

// luEvents returns one event of each of the six kinds, in the order of the spec.
func luEvents(t *testing.T) []combat.MonsterLimitedUseEvent {
	t.Helper()
	s := luStart(t, luDrake(t), combat.InLair)
	spent := s.Spend(luMind(t)).(combat.LimitedUseSpent).Transition.Events()
	s = luSpent(t, luSpent(t, s, luSpray(t)), luMind(t))
	turn, err := s.StartOfOwnTurn(map[combat.LimitedUsePartRef]dice.DieRoll{luSpray(t): luRoll(6)})
	if err != nil {
		t.Fatal(err)
	}
	out, _ := s.TakeLegendaryAction(luWindow(t, "b", 1), combat.ConditionActionAvailabilityAvailable)
	taken := out.(combat.LegendaryActionAllowed).Transition.Events()
	spentLegend := luTake(t, luTake(t, s, luWindow(t, "b", 1)), luWindow(t, "c", 1))
	regained, _ := spentLegend.StartOfOwnTurn(map[combat.LimitedUsePartRef]dice.DieRoll{luSpray(t): luRoll(1)})
	return []combat.MonsterLimitedUseEvent{
		spent[0], turn.Events()[0], s.AfterRest(character.LongRest).Events()[0], taken[0], regained.Events()[1],
		s.ChangeLair(combat.OutsideLair).Events()[0],
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-033 - the six events are the members of one closed MonsterLimitedUseEvent that implements Event, each with its constant unique kind and Combatants equal to the one monster; events are values.
// Path: happy.
func TestMonsterLimitedUseEventKindsAndCombatants(t *testing.T) {
	t.Parallel()
	kinds := []string{"limited_use_part_spent", "monster_recharge_rolled", "limited_uses_restored", "legendary_action_taken", "legendary_action_uses_regained", "monster_lair_status_changed"}
	events := luEvents(t)
	var all []combat.Event
	for i, e := range events {
		combattest.CheckEvent(t, e, kinds[i], combat.MustCombatantID(luSelf))
		all = append(all, e)
		switch e.(type) { // exhaustive over the closed family
		case combat.LimitedUsePartSpent, combat.MonsterRechargeRolled, combat.LimitedUsesRestored,
			combat.LegendaryActionTaken, combat.LegendaryActionUsesRegained, combat.MonsterLairStatusChanged:
		default:
			t.Errorf("event %T is not one of the six", e)
		}
	}
	combattest.CheckEventKinds(t, all)
	combattest.CheckEventsCopy(t, func() []combat.Event {
		return []combat.Event{events[0], events[1]}
	})
}

// Requirement: doc/specs/monster-limited-use.md FR-033, FR-050, FR-024 - each event reports its data: the spent subject and uses left, the part, face and result of a Recharge roll, the restored subjects, the window and uses left of a Legendary Action, the number regained, and the two lair statuses.
// Path: happy.
func TestMonsterLimitedUseEventData(t *testing.T) {
	t.Parallel()
	ev := luEvents(t)
	spent := ev[0].(combat.LimitedUsePartSpent)
	if spent.Subject() != (combat.PartSubject{Part: luMind(t)}) || spent.Remaining() != 1 {
		t.Errorf("spent = %v, %d, want Mind with 1 left", spent.Subject(), spent.Remaining())
	}
	rolled := ev[1].(combat.MonsterRechargeRolled)
	if rolled.Part() != luSpray(t) || rolled.Face() != 6 || rolled.Result() != combat.RechargeRegained {
		t.Errorf("rolled = %v %d %v, want Spray 6 regained", rolled.Part(), rolled.Face(), rolled.Result())
	}
	if got := ev[2].(combat.LimitedUsesRestored).Subjects(); !slices.Equal(got, []combat.LimitedUseSubject{combat.PartSubject{Part: luMind(t)}, combat.PartSubject{Part: luSpray(t)}}) {
		t.Errorf("restored subjects = %v, want the changed parts", got)
	}
	taken := ev[3].(combat.LegendaryActionTaken)
	if taken.Window() != luWindow(t, "b", 1) || taken.Remaining() != 3 {
		t.Errorf("taken = %v with %d left, want the window b/1 with 3 left", taken.Window(), taken.Remaining())
	}
	if regained := ev[4].(combat.LegendaryActionUsesRegained); regained.Count() != 2 {
		t.Errorf("regained = %d, want 2", regained.Count())
	}
	if lair := ev[5].(combat.MonsterLairStatusChanged); lair.From() != combat.InLair || lair.To() != combat.OutsideLair {
		t.Errorf("lair change = %v to %v, want in_lair to outside_lair", lair.From(), lair.To())
	}
}
