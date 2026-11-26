package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/grappling.md FR-081, FR-082, FR-086 - the three events are about [target, grappler] as a fresh
// copy, have constant snake_case kinds, are equal by value (target, grappler, part or cause), and are members of the
// GrappleEvent family.
// Path: happy.
func TestGrappleEvents(t *testing.T) {
	t.Parallel()
	started, err := combat.NewGrappleStarted(gid("a"), gid("b"), hand(1))
	if err != nil {
		t.Fatal(err)
	}
	ended, _ := combat.NewGrappleEnded(gid("b"), gid("a"), combat.GrappleEndCauseOutOfRange)
	dropped, _ := combat.NewGrappleRecordDropped(gid("a"), gid("b"))
	combattest.CheckEvent(t, started, "grapple_started", gid("b"), gid("a"))
	combattest.CheckEvent(t, ended, "grapple_ended", gid("b"), gid("a"))
	combattest.CheckEvent(t, dropped, "grapple_record_dropped", gid("b"), gid("a"))
	combattest.CheckEventKinds(t, []combat.Event{started, ended, dropped})
	if started.Part() != hand(1) || ended.Cause() != combat.GrappleEndCauseOutOfRange {
		t.Errorf("accessors = %v, %v", started.Part(), ended.Cause())
	}
	other, _ := combat.NewGrappleEnded(gid("b"), gid("a"), combat.GrappleEndCauseReleased)
	same, _ := combat.NewGrappleEnded(gid("b"), gid("a"), combat.GrappleEndCauseOutOfRange)
	swapped, _ := combat.NewGrappleEnded(gid("a"), gid("b"), combat.GrappleEndCauseOutOfRange)
	if ended != same || ended == other || ended == swapped {
		t.Error("GrappleEnded equality must follow target, grappler and cause [FR-086]")
	}
	otherPart, _ := combat.NewGrappleStarted(gid("a"), gid("b"), hand(0))
	if started == otherPart {
		t.Error("GrappleStarted ignores the part")
	}
}

// Requirement: doc/specs/grappling.md FR-081, FR-083 - an event constructor rejects a blank id, equal ids, an invalid part and
// a cause that is not one of the four, with ErrInvalidInput.
// Path: unhappy (invalid input).
func TestGrappleEventConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	var none combat.GraspingPart
	checks := map[string]error{
		"started blank grappler": second(combat.NewGrappleStarted("", gid("b"), hand(0))),
		"started same":           second(combat.NewGrappleStarted(gid("a"), gid("a"), hand(0))),
		"started no part":        second(combat.NewGrappleStarted(gid("a"), gid("b"), none)),
		"ended blank target":     second(combat.NewGrappleEnded("", gid("a"), combat.GrappleEndCauseReleased)),
		"ended same":             second(combat.NewGrappleEnded(gid("a"), gid("a"), combat.GrappleEndCauseReleased)),
		"ended zero cause":       second(combat.NewGrappleEnded(gid("b"), gid("a"), 0)),
		"dropped blank":          second(combat.NewGrappleRecordDropped(gid("a"), "")),
		"dropped same":           second(combat.NewGrappleRecordDropped(gid("a"), gid("a"))),
	}
	for name, err := range checks {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if problemOf(t, checks["started same"]) != combat.GrappleProblemSameGrapplerAndTarget || problemOf(t, checks["started blank grappler"]) != combat.GrappleProblemMissingValue {
		t.Error("event problems are not the NewGrapple ones")
	}
}

func second[T any](_ T, err error) error { return err }
