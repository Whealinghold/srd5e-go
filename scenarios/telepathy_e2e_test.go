package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpATelepathyKinds returns the Kind of each telepathy event.
func grpATelepathyKinds(events []combat.TelepathyEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// grpAChange runs one telepathic contact change.
func grpAChange(t *testing.T, req combat.TelepathyRequest) combat.ChangeTelepathicContactResult {
	t.Helper()
	res, err := combat.NewChangeTelepathicContact().Run(context.Background(), combat.ChangeTelepathicContactCommand{Request: req})
	if err != nil {
		t.Fatalf("ChangeTelepathicContact: %v", err)
	}
	return res
}

// Requirement: doc/specs/telepathy.md FR-002, FR-003, FR-004, FR-013, FR-014, FR-017, FR-018, FR-012, FR-019 - a telepath with a 120-foot range contacts a scout at exactly 120 feet, who may reply; contacting a guard instead breaks the first contact and makes the new one, in that order; contacting the guard again changes nothing; the telepath ends the contact, and ending it again finds none.
// Path: happy.
func TestTelepathyContactLifecycleScenario(t *testing.T) {
	t.Parallel()
	id := combat.MustCombatantID
	sage, scout, guard := id("sage"), id("scout"), id("guard")
	initiate := func(state combat.TelepathicContactState, contacted combat.CombatantID, feet int) combat.ChangeTelepathicContactResult {
		return grpAChange(t, combat.InitiateContact{Command: combat.InitiateTelepathicContactCommand{
			State: state, Telepath: sage, Contacted: contacted, Capability: combat.MustTelepathic(120),
			DistanceFeet: feet, Understanding: combat.ContactedUnderstandingALanguage}})
	}

	made, ok := initiate(combat.TelepathicContactState{}, scout, 120).(combat.InitiateTelepathicContactMade)
	if !ok || !made.State().IsInContact() || made.State().RangeFeet() != 120 || made.State().ReplyPolicy() != combat.ContactedMayReply ||
		len(made.Events()) != 1 || made.Events()[0].Kind() != "telepathic_contact_made" {
		t.Fatalf("contact at exactly the range = %#v, want InContact over 120 feet, replies allowed, one telepathic_contact_made", made)
	}
	if got := grpAChange(t, combat.ReplyToContact{State: made.State(), Replier: scout}); got != combat.ChangeTelepathicContactResult(combat.TelepathicReplyAllowed{}) {
		t.Errorf("the contacted creature's reply = %#v, want allowed", got)
	}

	switched, ok := initiate(made.State(), guard, 30).(combat.InitiateTelepathicContactMade)
	if !ok || !made.State().IsInPair(scout) || !switched.State().IsInPair(guard) || switched.State().IsInPair(scout) {
		t.Fatalf("contacting another creature = %#v, want the new pair held in place of the old (the old state unchanged)", switched)
	}
	if got := grpATelepathyKinds(switched.Events()); len(got) != 2 || got[0] != "telepathic_contact_broken" || got[1] != "telepathic_contact_made" {
		t.Errorf("switch events = %v, want broken then made", got)
	}
	if b := switched.Events()[0].(combat.TelepathicContactBroken); b.Cause() != combat.BreakCauseTelepathContactedAnother {
		t.Errorf("break cause = %v, want telepath_contacted_another", b.Cause())
	}
	if got := initiate(switched.State(), guard, 30); got != combat.ChangeTelepathicContactResult(combat.InitiateTelepathicContactAlreadyMade{}) {
		t.Errorf("contacting the same creature again = %#v, want AlreadyMade", got)
	}

	ended, ok := grpAChange(t, combat.EndContact{State: switched.State(), Telepath: sage}).(combat.EndTelepathicContactApplied)
	if !ok || ended.State().IsInContact() || len(ended.Events()) != 1 || ended.Events()[0].Kind() != "telepathic_contact_ended" {
		t.Fatalf("end = %#v, want NoContact and one telepathic_contact_ended", ended)
	}
	if got := grpAChange(t, combat.EndContact{State: ended.State(), Telepath: sage}); got != combat.ChangeTelepathicContactResult(combat.EndTelepathicContactNoContact{}) {
		t.Errorf("ending with no contact = %#v, want NoContact", got)
	}
	if got := grpAChange(t, combat.ReplyToContact{State: ended.State(), Replier: guard}); got != combat.ChangeTelepathicContactResult(combat.TelepathicReplyNoContact{}) {
		t.Errorf("a reply with no contact = %#v, want NoContact", got)
	}
}

// Requirement: doc/specs/telepathy.md FR-001, FR-003, FR-005, FR-006, FR-008 to FR-011, FR-024 - the first rejection in order applies: not telepathic, initiator Incapacitated, contacted Incapacitated, out of range by one foot, contacted creature understands nothing; a telepathic creature that speaks no language is fine; a malformed command is an invalid result, not an error.
// Path: unhappy (rule interaction).
func TestTelepathyRejectionsScenario(t *testing.T) {
	t.Parallel()
	id := combat.MustCombatantID
	stunned := combat.MustConditionSet([]combat.Condition{combat.Stunned{}}, nil)
	base := combat.InitiateTelepathicContactCommand{Telepath: id("sage"), Contacted: id("scout"), Capability: combat.MustTelepathic(60),
		DistanceFeet: 60, Understanding: combat.ContactedUnderstandingALanguage}
	with := func(f func(*combat.InitiateTelepathicContactCommand)) combat.InitiateTelepathicContactCommand {
		c := base
		f(&c)
		return c
	}
	cases := map[string]struct {
		cmd  combat.InitiateTelepathicContactCommand
		want combat.TelepathicContactRejection
	}{
		"initiator is not telepathic":             {with(func(c *combat.InitiateTelepathicContactCommand) { c.Capability = combat.TelepathyCapability{} }), combat.TelepathicContactRejectionInitiatorNotTelepathic},
		"initiator is stunned (so Incapacitated)": {with(func(c *combat.InitiateTelepathicContactCommand) { c.TelepathConditions = stunned }), combat.TelepathicContactRejectionInitiatorIncapacitated},
		"contacted creature is Incapacitated":     {with(func(c *combat.InitiateTelepathicContactCommand) { c.ContactedConditions = stunned }), combat.TelepathicContactRejectionContactedIncapacitated},
		"one foot beyond the range":               {with(func(c *combat.InitiateTelepathicContactCommand) { c.DistanceFeet = 61 }), combat.TelepathicContactRejectionContactedOutOfRange},
		"the contacted creature understands nothing": {with(func(c *combat.InitiateTelepathicContactCommand) {
			c.Understanding = combat.ContactedUnderstandingNeither
		}), combat.TelepathicContactRejectionContactedCannotUnderstand},
		"not telepathic outranks Incapacitated": {with(func(c *combat.InitiateTelepathicContactCommand) {
			c.Capability, c.TelepathConditions, c.DistanceFeet = combat.TelepathyCapability{}, stunned, 500
		}), combat.TelepathicContactRejectionInitiatorNotTelepathic},
		"Incapacitated outranks out of range": {with(func(c *combat.InitiateTelepathicContactCommand) {
			c.ContactedConditions, c.DistanceFeet, c.Understanding = stunned, 500, combat.ContactedUnderstandingNeither
		}), combat.TelepathicContactRejectionContactedIncapacitated},
	}
	for name, c := range cases {
		rej, ok := grpAChange(t, combat.InitiateContact{Command: c.cmd}).(combat.InitiateTelepathicContactRejected)
		if !ok || rej.Reason != c.want {
			t.Errorf("%s: result = %#v, want rejection %v", name, rej, c.want)
		}
	}
	telepathic := with(func(c *combat.InitiateTelepathicContactCommand) {
		c.Understanding = combat.ContactedUnderstandingTelepathic
	})
	if _, ok := grpAChange(t, combat.InitiateContact{Command: telepathic}).(combat.InitiateTelepathicContactMade); !ok {
		t.Errorf("a telepathic creature that speaks no language must be contactable")
	}
	atRange := with(func(c *combat.InitiateTelepathicContactCommand) { c.DistanceFeet = 60 })
	if _, ok := grpAChange(t, combat.InitiateContact{Command: atRange}).(combat.InitiateTelepathicContactMade); !ok {
		t.Errorf("a distance equal to the range must be contactable")
	}
	for name, bad := range map[string]combat.InitiateTelepathicContactCommand{
		"a negative distance":       with(func(c *combat.InitiateTelepathicContactCommand) { c.DistanceFeet = -1 }),
		"a contact with itself":     with(func(c *combat.InitiateTelepathicContactCommand) { c.Contacted = c.Telepath }),
		"a blank contacted id":      with(func(c *combat.InitiateTelepathicContactCommand) { c.Contacted = combat.CombatantID("") }),
		"an undefined reply policy": with(func(c *combat.InitiateTelepathicContactCommand) { c.ReplyPolicy = combat.ReplyPolicy(99) }),
	} {
		rej, ok := grpAChange(t, combat.InitiateContact{Command: bad}).(combat.InitiateTelepathicContactRejected)
		if !ok || rej.Reason != combat.TelepathicContactRejectionInvalidCommand {
			t.Errorf("%s: result = %#v, want InvalidCommand", name, rej)
		}
	}
	if _, err := combat.NewTelepathic(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a range of 0 err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/telepathy.md FR-015, FR-016, FR-020, FR-023 - the host reports facts after a contact is made: at the range the contact holds, one foot beyond it breaks (left range), the contacted creature becoming Incapacitated breaks it, a creature outside the pair changes nothing; with ContactedMayNotReply the contacted creature may not reply, and a creature that is not the contacted one cannot reply at all.
// Path: unhappy (state error).
func TestTelepathyReportsScenario(t *testing.T) {
	t.Parallel()
	id := combat.MustCombatantID
	sage, scout, guard := id("sage"), id("scout"), id("guard")
	state := combat.InContact(sage, scout, 120, combat.ContactedMayReply)

	if got := grpAChange(t, combat.ReportDistance{State: state, DistanceFeet: 120}); got != combat.ChangeTelepathicContactResult(combat.ReportTelepathDistanceUnchanged{}) {
		t.Errorf("a distance equal to the range = %#v, want unchanged", got)
	}
	left, ok := grpAChange(t, combat.ReportDistance{State: state, DistanceFeet: 121}).(combat.ReportTelepathDistanceBroken)
	if !ok || left.State().IsInContact() || len(left.Events()) != 1 || left.Events()[0].(combat.TelepathicContactBroken).Cause() != combat.BreakCauseContactedLeftRange {
		t.Errorf("a distance one foot beyond = %#v, want broken, cause contacted_left_range", left)
	}
	stunned := combat.MustConditionSet([]combat.Condition{combat.Stunned{}}, nil)
	broken, ok := grpAChange(t, combat.ReportIncapacitated{State: state, Creature: scout, Conditions: stunned}).(combat.ReportTelepathIncapacitatedBroken)
	if !ok || broken.State().IsInContact() || broken.Events()[0].(combat.TelepathicContactBroken).Cause() != combat.BreakCauseIncapacitated {
		t.Errorf("the contacted creature stunned = %#v, want broken, cause incapacitated", broken)
	}
	for name, r := range map[string]combat.ChangeTelepathicContactResult{
		"a creature outside the pair":  grpAChange(t, combat.ReportIncapacitated{State: state, Creature: guard, Conditions: stunned}),
		"a pair member in good health": grpAChange(t, combat.ReportIncapacitated{State: state, Creature: scout, Conditions: combat.ConditionSet{}}),
	} {
		if r != combat.ChangeTelepathicContactResult(combat.ReportTelepathIncapacitatedUnchanged{}) {
			t.Errorf("%s: result = %#v, want unchanged", name, r)
		}
	}

	silent := combat.InContact(sage, scout, 120, combat.ContactedMayNotReply)
	if got := grpAChange(t, combat.ReplyToContact{State: silent, Replier: scout}); got != combat.ChangeTelepathicContactResult(combat.TelepathicReplyNotPermitted{}) {
		t.Errorf("a reply under ContactedMayNotReply = %#v, want not permitted", got)
	}
	if got := grpAChange(t, combat.ReplyToContact{State: state, Replier: guard}); got != combat.ChangeTelepathicContactResult(combat.TelepathicReplyNotContacted{}) {
		t.Errorf("a reply by a creature not contacted = %#v, want not contacted", got)
	}
	if _, err := combat.NewChangeTelepathicContact().Run(context.Background(), combat.ChangeTelepathicContactCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a command with no request err = %v, want ErrInvalidInput", err)
	}
}
