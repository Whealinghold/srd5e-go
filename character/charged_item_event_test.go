package character_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/magic-item-effects.md FR-068 - the ChargedItemEvent family has exactly MagicItemLastChargeResolved, RingSpellStored,
// RingStoredSpellCast and StaffSpellAbsorbed; each is an Event that carries the CharacterID passed in and the instance and its own fields.
// Path: happy.
func TestChargedItemEventsCarryTheCharacterID(t *testing.T) {
	t.Parallel()
	ring := ringWith(t, 2)
	stored := must(ring.Store(holder, inst("ring-1"), intoRing(1))).(character.SpellStorageStored).Event()
	cast := must(ring.Cast(holder, inst("ring-1"), 0, character.AccessFullMagical)).(character.StoredSpellCast).Event()
	absorbed := must((character.StaffSpellAbsorption{}).Absorb(holder, inst("magi-1"), character.MustItemCharges(50, 0), 4)).(character.StaffSpellAbsorptionAbsorbed).Event()
	rule := must(character.NewLastChargeAutomatic(character.LastChargeDestroyed))
	last := must(must(character.NewResolveLastCharge(dicetest.NewFixedRoller())).Do(context.Background(), lastChargeCmd(character.MustItemCharges(3, 0), rule))).(character.ResolveLastChargeResolved).Event()
	events := []character.ChargedItemEvent{stored, cast, absorbed, last}
	var seen []string
	for _, e := range events {
		var asEvent character.Event = e
		if asEvent.CharacterID() != holder {
			t.Errorf("%T carries %v", e, asEvent.CharacterID())
		}
		switch e := e.(type) {
		case character.MagicItemLastChargeResolved:
			seen = append(seen, "last")
			if e.InstanceID() != inst("wand-1") || e.Consequence() != character.LastChargeDestroyed {
				t.Errorf("last = %+v", e)
			}
		case character.RingSpellStored:
			seen = append(seen, "stored")
			if e.InstanceID() != inst("ring-1") || e.SlotLevel() != lvl(1) || e.SpellID() != sp("shield") {
				t.Errorf("stored = %+v", e)
			}
		case character.RingStoredSpellCast:
			seen = append(seen, "cast")
			if e.InstanceID() != inst("ring-1") || e.SlotLevel() != lvl(2) {
				t.Errorf("cast = %+v", e)
			}
		case character.StaffSpellAbsorbed:
			seen = append(seen, "absorbed")
			if e.InstanceID() != inst("magi-1") || e.Level() != 4 {
				t.Errorf("absorbed = %+v", e)
			}
		}
	}
	if len(seen) != 4 {
		t.Errorf("seen %v", seen)
	}
}
