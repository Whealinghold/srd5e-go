package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
)

// trkAllEvents produces one event of each of the 16 kinds through the public operations.
func trkAllEvents(t *testing.T) (map[string]combat.TurnEvent, []combat.TurnEvent) {
	t.Helper()
	want := map[string]combat.TurnEvent{}
	add := func(events []combat.TurnEvent) {
		for _, e := range events {
			want[e.Kind()] = e
		}
	}
	tr := trkABC(t)
	grant, _ := combat.NewAdditionalActionGrant("surge", combat.AnyActionBut{})
	bg, _ := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "b"}, combat.NoStandardAction{})
	est := trkEstablish(t, trkStandings(20, 10), nil).(combat.InitiativeOrderEstablished)
	add(est.Events())
	_, ev := trkApplied(t)(tr.GrantAdditionalAction(trkA, grant))
	add(ev)
	_, ev = trkApplied(t)(tr.RecordFeatUse(trkA, "f", character.OncePerTurn{}))
	add(ev)
	_, ev = trkApplied(t)(tr.TakeAction(trkA, trkAvail, combat.ReadyChoice{Action: trkReadied(t, "t", combat.ReadiedActionResponse{Choice: trkAttack})}))
	add(ev)
	_, ev = trkApplied(t)(tr.TakeBonusAction(trkA, trkAvail, combat.EntitledToBonusAction{Grant: bg}))
	add(ev)
	_, ev = trkApplied(t)(tr.TakeReaction(trkB, trkAvail, combat.SpellReaction{Spell: "shield"}))
	add(ev)
	_, ev = trkApplied(t)(tr.Move(trkA, []combat.MoveSegment{trkWalk(t, 5)}))
	add(ev)
	_, ev = trkApplied(t)(tr.InteractWithObject(trkA))
	add(ev)
	_, ev = trkApplied(t)(tr.ChangePosture(trkA, combat.DropProneChange{}))
	add(ev)
	_, ev = trkApplied(t)(tr.ChangePosture(trkA, combat.StandUpChange{Movement: combat.CrawlOrRightItself{CostFeet: 15}}))
	add(ev)
	_, ev = trkApplied(t)(tr.UpdateMovementSpeeds(trkA, trkSpeeds(t, 5)))
	add(ev)
	ready := trkTake(t, tr, trkA, combat.ReadyChoice{Action: trkReadied(t, "t", combat.ReadiedActionResponse{Choice: trkAttack})})
	_, ev = trkAdvance(t, advN(t, ready, 2), combat.TurnAdvance{SharedSpaceOccupants: []content.CreatureSize{content.SizeMedium}})
	add(ev)
	swap, err := combat.NewSwapInitiativeCommand(trkStandings(20, 10), "a", "b", character.InitiativeSwap{}, trkAvail, trkAvail, combat.AllyConsentWilling)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewSwapInitiative().Run(context.Background(), swap)
	if err != nil {
		t.Fatal(err)
	}
	add(res.(combat.SwapInitiativeApplied).Events())
	var list []combat.TurnEvent
	for _, e := range want {
		list = append(list, e)
	}
	return want, list
}

// Requirement: doc/specs/initiative-and-actions.md FR-125, FR-128, FR-129, FR-130 - the 16 events are TurnEvents and Events with unique constant snake_case kinds and Combatants lists of the right creatures in the right order, each a fresh copy.
// Path: per case.
func TestTurnEventFamily(t *testing.T) {
	t.Parallel()
	byKind, list := trkAllEvents(t)
	kinds := []string{"tracker_round_started", "tracker_turn_started", "tracker_turn_ended", "additional_action_granted", "feat_use_recorded",
		"initiative_swapped", "action_taken", "bonus_action_used", "reaction_taken", "movement_spent", "object_interacted", "dropped_prone",
		"stood_up", "fell_prone_in_shared_space", "readied_action_expired", "movement_speeds_updated"}
	if len(byKind) != 16 {
		t.Fatalf("got %d kinds, want the 16 of FR-125: %v", len(byKind), list)
	}
	var events []combat.Event
	for _, k := range kinds {
		e, ok := byKind[k]
		if !ok {
			t.Fatalf("no event of kind %s", k)
		}
		events = append(events, e)
		var want []combat.CombatantID
		switch k {
		case "tracker_round_started":
			want = nil
		case "initiative_swapped":
			want = []combat.CombatantID{"a", "b"}
		case "reaction_taken":
			want = []combat.CombatantID{trkB}
		case "tracker_turn_ended", "fell_prone_in_shared_space": // C ends its turn in the Advance of the fixture
			want = []combat.CombatantID{trkC}
		default:
			want = []combat.CombatantID{trkA}
		}
		combattest.CheckEvent(t, e, k, want...)
	}
	combattest.CheckEventKinds(t, events)
	// FR-130: an Opportunity Attack names observer then provoker.
	_, ev := trkApplied(t)(trkABC(t).TakeReaction(trkB, trkAvail, trkOA(trkA, combat.DepartureBySpeed, combat.VisibilitySeen)))
	combattest.CheckEvent(t, ev[0], "reaction_taken", trkB, trkA)
}

// Requirement: doc/specs/initiative-and-actions.md FR-128 - a result hands out its events as a copy, in order.
// Path: unhappy (immutability).
func TestTurnResultEventsAreCopies(t *testing.T) {
	t.Parallel()
	tr := trkABC(t)
	step, _ := tr.Advance(combat.TurnAdvance{})
	combattest.CheckEventsCopy(t, func() []combat.Event {
		var out []combat.Event
		for _, e := range step.(combat.TrackerStepApplied).Events() {
			out = append(out, e)
		}
		return out
	})
	cmd, _ := combat.NewAdvanceTrackerTurnCommand(tr, combat.TurnAdvance{})
	res, _ := combat.NewAdvanceTrackerTurn().Run(context.Background(), cmd)
	ev := res.(combat.AdvanceTrackerTurnApplied).Events()
	ev[0] = nil
	if res.(combat.AdvanceTrackerTurnApplied).Events()[0] == nil {
		t.Error("the result shares its events")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-131, FR-152 - events carry the fields FR-152 lists, with no display text.
// Path: happy.
func TestTurnEventFields(t *testing.T) {
	t.Parallel()
	byKind, _ := trkAllEvents(t)
	if e := byKind["initiative_swapped"].(combat.InitiativeSwapped); e.FirstCount() != 10 || e.SecondCount() != 20 {
		t.Errorf("InitiativeSwapped counts = %d, %d, want 10 and 20", e.FirstCount(), e.SecondCount())
	}
	if e := byKind["stood_up"].(combat.StoodUp); e.CostFeet() != 15 || e.MovedFeet() != 15 {
		t.Errorf("StoodUp = %+v", e)
	}
	if e := byKind["movement_speeds_updated"].(combat.MovementSpeedsUpdated); e.Speeds().Walk() != 5 {
		t.Errorf("MovementSpeedsUpdated = %+v", e)
	}
	if e := byKind["feat_use_recorded"].(combat.FeatUseRecorded); e.Feat() != "f" {
		t.Errorf("FeatUseRecorded = %+v", e)
	}
	if e := byKind["action_taken"].(combat.ActionTaken); e.Choice().(combat.ReadyChoice).Action.Trigger() != "t" {
		t.Errorf("ActionTaken = %+v", e)
	}
	if e := byKind["reaction_taken"].(combat.ReactionTaken); e.Record() != (combat.SpellReactionRecord{Spell: "shield"}) {
		t.Errorf("ReactionTaken = %+v", e)
	}
	// A readied move records its charges and total and emits no MovementSpent.
	resp, _ := combat.NewReadiedMoveResponse([]combat.MoveSegment{trkWalk(t, 10), trkMode(t, 5, combat.MovementModeFly)})
	tr := trkTracker(t, trkProfile(t, trkA, content.SizeMedium, trkSpeeds(t, 30)), trkProfile(t, trkB, content.SizeMedium, trkWalkFly(t)))
	tr, _ = trkAdvance(t, tr)
	tr = trkTake(t, tr, trkB, combat.ReadyChoice{Action: trkReadied(t, "go", resp)})
	tr, _ = trkAdvance(t, tr)
	_, ev := trkApplied(t)(tr.TakeReaction(trkB, trkAvail, combat.ReadiedActionReaction{Trigger: "go"}))
	mo := ev[0].(combat.ReactionTaken).Record().(combat.ReadiedActionRecord).Outcome.(combat.ReadiedMoveOutcome)
	if len(ev) != 1 || mo.TotalFeet() != 15 || len(mo.Charges()) != 2 || mo.Charges()[1] != (combat.MovementCharge{Mode: combat.MovementModeFly, Feet: 5}) {
		t.Errorf("events %v outcome %+v, want one ReactionTaken with two charges and total 15 [FR-152]", trkKinds(ev), mo)
	}
	mo.Charges()[0].Feet = 0
	if mo.Charges()[0].Feet != 10 {
		t.Error("ReadiedMoveOutcome shares its charges")
	}
}
