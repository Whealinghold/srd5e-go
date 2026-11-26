package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/hiding-searching-and-senses.md FR-092, FR-093, FR-094 and 4.13a - CreatureHid and HidingEnded are members of
// the HidingEvent family, about [hider] as a fresh copy, with the constant kinds creature_hid and hiding_ended, equal by value
// (hider, find DC or activity) and never equal to each other.
// Path: unhappy (immutability).
func TestHidingEvents(t *testing.T) {
	t.Parallel()
	hid, err := combat.NewCreatureHid(gid("h"), 16)
	if err != nil {
		t.Fatal(err)
	}
	ended, err := combat.NewHidingEnded(gid("h"), combat.HidingActivityAttackRoll)
	if err != nil {
		t.Fatal(err)
	}
	var _ combat.HidingEvent = hid
	var _ combat.HidingEvent = ended
	combattest.CheckEvent(t, hid, "creature_hid", gid("h"))
	combattest.CheckEvent(t, ended, "hiding_ended", gid("h"))
	combattest.CheckEventKinds(t, []combat.Event{hid, ended})
	if hid.FindDC() != 16 || ended.Activity() != combat.HidingActivityAttackRoll {
		t.Errorf("accessors = %v, %v", hid.FindDC(), ended.Activity())
	}
	again, _ := combat.NewCreatureHid(gid("h"), 16)
	otherDC, _ := combat.NewCreatureHid(gid("h"), 17)
	otherHider, _ := combat.NewCreatureHid(gid("g"), 16)
	otherActivity, _ := combat.NewHidingEnded(gid("h"), combat.HidingActivityFoundByEnemy)
	if hid != again || hid == otherDC || hid == otherHider || ended == otherActivity || combat.Event(hid) == combat.Event(ended) {
		t.Error("event equality must follow every field [FR-094]")
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-092 - the event constructors reject a blank hider, a find DC below 15 and
// an undefined activity with ErrInvalidInput.
// Path: unhappy (invalid input).
func TestHidingEventConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	checks := map[string]error{
		"hid blank":         second(combat.NewCreatureHid("", 16)),
		"hid DC 14":         second(combat.NewCreatureHid(gid("h"), 14)),
		"ended blank":       second(combat.NewHidingEnded("", combat.HidingActivityAttackRoll)),
		"ended no activity": second(combat.NewHidingEnded(gid("h"), 0)),
		"ended bad":         second(combat.NewHidingEnded(gid("h"), 8)),
	}
	for name, err := range checks {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, err := combat.NewCreatureHid(gid("h"), 15); err != nil {
		t.Errorf("DC 15 rejected: %v", err)
	}
}
