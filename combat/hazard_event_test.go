package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/environment-hazards-and-objects.md FR-105, FR-106 and section 5 - the seven creature events carry exactly the target and the four object events none, each with its constant snake_case Kind, copied Combatants, accessors for its data and equality by value.
// Path: per case.
func TestEnvironmentEvents(t *testing.T) {
	t.Parallel()
	id := subject()
	burn, _ := combat.NewBurningEnded(id, combat.ExtinguishMethodSubmerged)
	dehy, _ := combat.NewDehydrationRecovered(id, 2)
	mal, _ := combat.NewMalnutritionRecovered(id, 3)
	held, _ := combat.NewBreathHeldStarted(id)
	ran, _ := combat.NewRanOutOfBreath(id)
	choke, _ := combat.NewBeganChoking(id)
	rest, _ := combat.NewBreathRestored(id, 0)
	dmg, _ := combat.NewObjectDamaged(4)
	for _, c := range []struct {
		event combat.EnvironmentEvent
		kind  string
		ids   []combat.CombatantID
	}{
		{burn, "burning_ended", []combat.CombatantID{id}}, {dehy, "dehydration_recovered", []combat.CombatantID{id}},
		{mal, "malnutrition_recovered", []combat.CombatantID{id}}, {held, "breath_held_started", []combat.CombatantID{id}},
		{ran, "ran_out_of_breath", []combat.CombatantID{id}}, {choke, "began_choking", []combat.CombatantID{id}},
		{rest, "breath_restored", []combat.CombatantID{id}}, {dmg, "object_damaged", nil},
		{combat.ObjectDamageDeflected{}, "object_damage_deflected", nil}, {combat.ObjectDestroyed{}, "object_destroyed", nil},
		{combat.ObjectCollapsed{}, "object_collapsed", nil},
	} {
		combattest.CheckEvent(t, c.event, c.kind, c.ids...)
		if _, ok := any(c.event).(combat.Event); !ok {
			t.Errorf("%s is not an Event", c.kind)
		}
	}
	if burn.Method() != combat.ExtinguishMethodSubmerged || dehy.Levels() != 2 || mal.Levels() != 3 || rest.LevelsRemoved() != 0 || dmg.Amount() != 4 {
		t.Error("an event accessor is wrong")
	}
	other, _ := combat.NewBurningEnded(id, combat.ExtinguishMethodDoused)
	again, _ := combat.NewBurningEnded(id, combat.ExtinguishMethodSubmerged)
	if burn == other || burn != again || dehy == (combat.DehydrationRecovered{}) {
		t.Error("events do not compare by value")
	}
	if got := (combat.ObjectDamageDeflected{}).Combatants(); got == nil || len(got) != 0 {
		t.Errorf("object event Combatants = %#v, want an empty non-nil slice", got)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-105 - the event constructors reject a zero target, a method or an amount that is not valid, with an ErrInvalidInput error.
// Path: unhappy (invalid input).
func TestEnvironmentEventConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	id := subject()
	_, e1 := combat.NewBurningEnded("", combat.ExtinguishMethodDoused)
	_, e2 := combat.NewBurningEnded(id, 0)
	_, e3 := combat.NewDehydrationRecovered("", 1)
	_, e4 := combat.NewDehydrationRecovered(id, 0)
	_, e5 := combat.NewMalnutritionRecovered("", 1)
	_, e6 := combat.NewMalnutritionRecovered(id, 0)
	_, e7 := combat.NewBreathHeldStarted("")
	_, e8 := combat.NewRanOutOfBreath("")
	_, e9 := combat.NewBeganChoking("")
	_, e10 := combat.NewBreathRestored("", 0)
	_, e11 := combat.NewBreathRestored(id, -1)
	_, e12 := combat.NewObjectDamaged(0)
	for i, err := range []error{e1, e2, e3, e4, e5, e6, e7, e8, e9, e10, e11, e12} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("constructor %d error = %v, want ErrInvalidInput", i+1, err)
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-105 - the least valid values are accepted: one unlocked level, 0 levels removed and 1 Hit Point of object damage.
// Path: unhappy (boundary).
func TestEnvironmentEventConstructorsAcceptTheLeastValidValue(t *testing.T) {
	t.Parallel()
	id := subject()
	d, e1 := combat.NewDehydrationRecovered(id, 1)
	m, e2 := combat.NewMalnutritionRecovered(id, 1)
	r, e3 := combat.NewBreathRestored(id, 0)
	if more, err := combat.NewBreathRestored(id, 3); err != nil || more.LevelsRemoved() != 3 {
		t.Errorf("NewBreathRestored(id, 3) = %v, %v", more, err)
	}
	o, e4 := combat.NewObjectDamaged(1)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil || d.Levels() != 1 || m.Levels() != 1 || r.LevelsRemoved() != 0 || o.Amount() != 1 {
		t.Errorf("= %v %v %v %v, %v %v %v %v", d, m, r, o, e1, e2, e3, e4)
	}
}
