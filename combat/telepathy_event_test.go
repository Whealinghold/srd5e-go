package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/telepathy.md FR-021 and table 4.2 - the three events are TelepathyEvent members about [telepath, contacted] as a fresh copy, with the kinds telepathic_contact_made, _broken and _ended, unique, and equal by value.
// Path: unhappy (immutability).
func TestTelepathyEvents(t *testing.T) {
	t.Parallel()
	made, err1 := combat.NewTelepathicContactMade(gid("t"), gid("c"))
	broken, err2 := combat.NewTelepathicContactBroken(gid("t"), gid("c"), combat.BreakCauseContactedLeftRange)
	ended, err3 := combat.NewTelepathicContactEnded(gid("t"), gid("c"))
	if err := errors.Join(err1, err2, err3); err != nil {
		t.Fatal(err)
	}
	var _ combat.TelepathyEvent = made
	var _ combat.TelepathyEvent = broken
	var _ combat.TelepathyEvent = ended
	combattest.CheckEvent(t, made, "telepathic_contact_made", gid("t"), gid("c"))
	combattest.CheckEvent(t, broken, "telepathic_contact_broken", gid("t"), gid("c"))
	combattest.CheckEvent(t, ended, "telepathic_contact_ended", gid("t"), gid("c"))
	combattest.CheckEventKinds(t, []combat.Event{made, broken, ended})
	other, _ := combat.NewTelepathicContactBroken(gid("t"), gid("c"), combat.BreakCauseIncapacitated)
	again, _ := combat.NewTelepathicContactBroken(gid("t"), gid("c"), combat.BreakCauseContactedLeftRange)
	if broken.Cause() != combat.BreakCauseContactedLeftRange || broken == other || broken != again {
		t.Error("equality and Cause must follow every field")
	}
}

// Requirement: doc/specs/telepathy.md FR-021 - the event constructors reject a blank id, equal ids and (broken) an undefined cause with ErrInvalidInput.
// Path: unhappy (invalid input).
func TestTelepathyEventConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	checks := map[string]error{
		"made blank telepath":  second(combat.NewTelepathicContactMade("", gid("c"))),
		"made blank contacted": second(combat.NewTelepathicContactMade(gid("t"), "")),
		"made same":            second(combat.NewTelepathicContactMade(gid("t"), gid("t"))),
		"ended blank":          second(combat.NewTelepathicContactEnded("", gid("c"))),
		"ended same":           second(combat.NewTelepathicContactEnded(gid("t"), gid("t"))),
		"broken blank":         second(combat.NewTelepathicContactBroken("", gid("c"), combat.BreakCauseIncapacitated)),
		"broken same":          second(combat.NewTelepathicContactBroken(gid("t"), gid("t"), combat.BreakCauseIncapacitated)),
		"broken cause 0":       second(combat.NewTelepathicContactBroken(gid("t"), gid("c"), 0)),
		"broken cause 4":       second(combat.NewTelepathicContactBroken(gid("t"), gid("c"), 4)),
	}
	for name, err := range checks {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}
