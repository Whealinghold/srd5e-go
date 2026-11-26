package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// Requirement: doc/specs/revival-and-restoration-spells.md FR-030, FR-046 - the RestorationEvent family has the three members HitPointMaximumIncreased,
// HitPointMaximumIncreaseEnded and DeathWardSpent, each with a constant snake_case kind, the creature as its only combatant (a fresh copy) and the values the
// operation gave it; kinds are unique across the family and the events HitPointMaximumRestored and Died of other families are not in it.
// Path: unhappy (name collision).
func TestRestorationEventFamily(t *testing.T) {
	t.Parallel()
	inc := rrIncreased(t, pcStatus(t, 15, 20, 0), combat.HitPointMaximumIncreases{}, rrIncrease(t, "a", "aid", 5)).Events()[0].(combat.HitPointMaximumIncreased)
	ended := rrEnded(t, pcStatus(t, 25, 25, 0), rrLedger(t, rrIncrease(t, "a", "aid", 5)), "a").Events()[0].(combat.HitPointMaximumIncreaseEnded)
	res, err := combat.NegateInstantKill(gid("hero"), pcStatus(t, 5, 20, 0), combat.WardActive)
	if err != nil {
		t.Fatal(err)
	}
	spent := res.(combat.GuardedDamageApplied).Events()[0].(combat.DeathWardSpent)
	family := []combat.RestorationEvent{inc, ended, spent}
	var events []combat.Event
	for _, e := range family {
		events = append(events, e)
	}
	combattest.CheckEvent(t, inc, "hit_point_maximum_increased", gid("hero"))
	combattest.CheckEvent(t, ended, "hit_point_maximum_increase_ended", gid("hero"))
	combattest.CheckEvent(t, spent, "death_ward_spent", gid("hero"))
	combattest.CheckEventKinds(t, events)
	incResult := rrIncreased(t, pcStatus(t, 15, 20, 0), combat.HitPointMaximumIncreases{}, rrIncrease(t, "a", "aid", 5))
	combattest.CheckEventsCopy(t, incResult.Events)
	combattest.CheckEventsCopy(t, res.(combat.GuardedDamageApplied).Events)
	if inc.Maximum() != 25 || inc.Increase() != 5 || ended.Maximum() != 20 || ended.Drop() != 5 || ended.HitPointsLost() != 5 ||
		spent.Reason() != combat.InstantKill {
		t.Errorf("event values = %+v %+v %+v", inc, ended, spent)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md section 3 - the names of the Death Ward triggers.
// Path: happy.
func TestDeathWardTriggerNames(t *testing.T) {
	t.Parallel()
	for tr, want := range map[combat.DeathWardTrigger]string{combat.DropToZero: "drop_to_zero", combat.InstantKill: "instant_kill", 0: "unknown", 3: "unknown"} {
		if tr.String() != want {
			t.Errorf("DeathWardTrigger(%d).String() = %q, want %q", int(tr), tr.String(), want)
		}
	}
}
