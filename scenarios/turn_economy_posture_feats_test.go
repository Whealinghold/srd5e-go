package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpETracker establishes a turn tracker for the given ids, highest Initiative first (the first id acts first), each a Medium creature with
// the walking Speed given, through the public use case.
func grpETracker(t *testing.T, walk int, ids ...combat.CombatantID) combat.TurnTracker {
	t.Helper()
	speeds, err := combat.NewMovementSpeeds(walk)
	if err != nil {
		t.Fatal(err)
	}
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	for i, id := range ids {
		standings = append(standings, combat.InitiativeStanding{Combatant: id, Count: combat.InitiativeCount(20 - 2*i)})
		profiles = append(profiles, combat.CombatantProfile{ID: id, Size: content.SizeMedium, Speeds: speeds})
	}
	cmd, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	return res.(combat.InitiativeOrderEstablished).Tracker()
}

// grpEAdvance ends the active turn and returns the new tracker.
func grpEAdvance(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
	t.Helper()
	cmd, err := combat.NewAdvanceTrackerTurnCommand(tr, combat.TurnAdvance{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewAdvanceTrackerTurn().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	return res.(combat.AdvanceTrackerTurnApplied).Tracker()
}

func grpETurnKinds(events []combat.TurnEvent) []string {
	kinds := make([]string, 0, len(events))
	for _, e := range events {
		kinds = append(kinds, e.Kind())
	}
	return kinds
}

// Requirement: doc/specs/initiative-and-actions.md FR-058, FR-059, FR-090, FR-091, FR-147, FR-148 (SRD 5.2.1, "Your Turn", p. 13; "Dropping
// Prone", p. 14; "Prone [Condition]", p. 186) - on their turn a fighter with Speed 30 drops Prone for free, stands up for 15 feet of movement
// (the cost the conditions rules computed, half the Speed), uses their one free interaction with an object, and is refused a second one.
// Path: happy.
func TestFighterDropsProneStandsUpAndInteractsOnce(t *testing.T) {
	t.Parallel()
	ctx, hero, goblin := context.Background(), combat.MustCombatantID("hero"), combat.MustCombatantID("goblin")
	tr := grpETracker(t, 30, hero, goblin)

	drop, err := combat.NewChangePostureCommand(tr, hero, combat.DropProneChange{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewChangePosture().Run(ctx, drop)
	if err != nil {
		t.Fatal(err)
	}
	dropped := res.(combat.ChangePostureApplied)
	if !slices.Equal(grpETurnKinds(dropped.Events()), []string{"dropped_prone"}) || dropped.Tracker().Resources().Action() != combat.ResourceAvailable ||
		dropped.Tracker().Resources().MovedFeet() != 0 {
		t.Fatalf("drop prone = %v action %v moved %d, want dropped_prone using no action and no movement [FR-090]",
			grpETurnKinds(dropped.Events()), dropped.Tracker().Resources().Action(), dropped.Tracker().Resources().MovedFeet())
	}
	tr = dropped.Tracker()

	cost, err := (combat.ConditionEffects{}).StandUp(combat.MustConditionSet([]combat.Condition{combat.Prone{}}, nil), 30)
	if err != nil {
		t.Fatal(err)
	}
	rises, err := combat.NewChangePostureCommand(tr, hero, combat.StandUpChange{Movement: cost})
	if err != nil {
		t.Fatal(err)
	}
	res, err = combat.NewChangePosture().Run(ctx, rises)
	if err != nil {
		t.Fatal(err)
	}
	stood := res.(combat.ChangePostureApplied)
	up, ok := stood.Events()[0].(combat.StoodUp)
	if !ok || stood.Tracker().Resources().MovedFeet() != 15 || stood.Tracker().Resources().Action() != combat.ResourceAvailable || up.Kind() != "stood_up" {
		t.Fatalf("stand up = %#v moved %d, want stood_up costing 15 feet and no action [FR-091]", stood.Events(), stood.Tracker().Resources().MovedFeet())
	}
	tr = stood.Tracker()

	use := combat.NewInteractWithObject()
	interact, err := combat.NewInteractWithObjectCommand(tr, hero)
	if err != nil {
		t.Fatal(err)
	}
	res2, err := use.Run(ctx, interact)
	if err != nil {
		t.Fatal(err)
	}
	opened := res2.(combat.InteractWithObjectApplied)
	if !slices.Equal(grpETurnKinds(opened.Events()), []string{"object_interacted"}) || opened.Tracker().Resources().Interaction() != combat.ResourceSpent ||
		opened.Tracker().Resources().Action() != combat.ResourceAvailable || opened.Tracker().Resources().MovedFeet() != 15 {
		t.Fatalf("interaction = %v, want object_interacted with the action and movement untouched [FR-058]", grpETurnKinds(opened.Events()))
	}
	again, err := combat.NewInteractWithObjectCommand(opened.Tracker(), hero)
	if err != nil {
		t.Fatal(err)
	}
	res2, err = use.Run(ctx, again)
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := res2.(combat.InteractWithObjectRejected); !ok || r.Reason != (combat.TurnNeedsUtilizeActionRejection{}) {
		t.Errorf("a second interaction = %#v, want NeedsUtilizeAction [FR-058]", res2)
	}

	// The free interaction does not carry over (FR-040): on the hero's next turn it is available again.
	next := grpEAdvance(t, grpEAdvance(t, opened.Tracker()))
	if next.Active().Combatant != hero || next.Round() != 2 || next.Resources().Interaction() != combat.ResourceAvailable || next.Resources().MovedFeet() != 0 {
		t.Errorf("round %d active %v interaction %v moved %d, want a fresh turn for the hero [FR-040]", next.Round(), next.Active().Combatant,
			next.Resources().Interaction(), next.Resources().MovedFeet())
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-090, FR-091, FR-118, FR-119 and section 6 command errors (SRD 5.2.1, "Dropping Prone",
// p. 14; "Prone [Condition]", p. 186) - posture and interaction refusals in their order: unknown combatant, not the active combatant, Speed
// zero, a stand-up that costs more than the movement left, a creature that can only crawl; malformed commands are invalid input.
// Path: unhappy (state errors, boundary, invalid input).
func TestPostureAndInteractionRefusalsAndInvalidCommands(t *testing.T) {
	t.Parallel()
	ctx, hero, goblin, stranger := context.Background(), combat.MustCombatantID("hero"), combat.MustCombatantID("goblin"), combat.MustCombatantID("stranger")
	tr := grpETracker(t, 30, hero, goblin)
	posture := func(tracker combat.TurnTracker, who combat.CombatantID, change combat.PostureChange) combat.ChangePostureResult {
		t.Helper()
		cmd, err := combat.NewChangePostureCommand(tracker, who, change)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewChangePosture().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	if r, ok := posture(tr, stranger, combat.DropProneChange{}).(combat.ChangePostureRejected); !ok || r.Reason != (combat.TurnUnknownCombatantRejection{ID: stranger}) {
		t.Errorf("unknown combatant = %#v, want UnknownCombatant [FR-119]", r)
	}
	if r, ok := posture(tr, goblin, combat.DropProneChange{}).(combat.ChangePostureRejected); !ok || r.Reason != (combat.TurnNotActiveCombatantRejection{}) {
		t.Errorf("a combatant whose turn it is not = %#v, want NotActiveCombatant [FR-118]", r)
	}
	if r, ok := posture(tr, hero, combat.StandUpChange{Movement: combat.CrawlOnly{}}).(combat.ChangePostureRejected); !ok || r.Reason != (combat.TurnSpeedIsZeroRejection{}) {
		t.Errorf("a stand-up by a creature that can only crawl = %#v, want SpeedIsZero [FR-091]", r)
	}

	// 20 feet moved leaves 10 of 30; a stand-up costing 15 is 5 feet short; one costing 10 is accepted (the boundary).
	seg, err := combat.NewMoveSegment(combat.MoveSegmentInput{DistanceFeet: 20, Mode: combat.MovementModeWalk})
	if err != nil {
		t.Fatal(err)
	}
	mv, err := combat.NewMoveCommand(tr, hero, []combat.MoveSegment{seg})
	if err != nil {
		t.Fatal(err)
	}
	moved, err := combat.NewMove().Run(ctx, mv)
	if err != nil {
		t.Fatal(err)
	}
	after := moved.(combat.MoveApplied).Tracker()
	if r, ok := posture(after, hero, combat.StandUpChange{Movement: combat.CrawlOrRightItself{CostFeet: 15}}).(combat.ChangePostureRejected); !ok ||
		r.Reason != (combat.TurnInsufficientMovementRejection{ShortfallFeet: 5}) {
		t.Errorf("a 15 foot stand-up with 10 left = %#v, want InsufficientMovement short by 5 [FR-091]", r)
	}
	if r, ok := posture(after, hero, combat.StandUpChange{Movement: combat.CrawlOrRightItself{CostFeet: 10}}).(combat.ChangePostureApplied); !ok || r.Tracker().Resources().MovedFeet() != 30 {
		t.Errorf("a 10 foot stand-up with exactly 10 left = %#v, want accepted with 30 feet moved [FR-091]", r)
	}

	// A creature whose walking Speed is 0 cannot drop Prone (FR-090), read from the profile.
	rooted := grpETracker(t, 0, hero, goblin)
	if r, ok := posture(rooted, hero, combat.DropProneChange{}).(combat.ChangePostureRejected); !ok || r.Reason != (combat.TurnSpeedIsZeroRejection{}) {
		t.Errorf("drop prone at Speed 0 = %#v, want SpeedIsZero [FR-090]", r)
	}

	interact, err := combat.NewInteractWithObjectCommand(tr, goblin)
	if err != nil {
		t.Fatal(err)
	}
	if res, err := combat.NewInteractWithObject().Run(ctx, interact); err != nil {
		t.Fatal(err)
	} else if r, ok := res.(combat.InteractWithObjectRejected); !ok || r.Reason != (combat.TurnNotActiveCombatantRejection{}) {
		t.Errorf("interaction out of turn = %#v, want NotActiveCombatant [FR-147]", res)
	}

	if _, err := combat.NewChangePostureCommand(tr, hero, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil change err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewChangePostureCommand(tr, hero, combat.StandUpChange{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("stand-up without a ProneMovement err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewChangePostureCommand(tr, hero, combat.StandUpChange{Movement: combat.CrawlOrRightItself{CostFeet: -1}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative stand-up cost err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewChangePostureCommand(combat.TurnTracker{}, hero, combat.DropProneChange{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero tracker err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewInteractWithObjectCommand(tr, combat.CombatantID("")); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank id err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-155, FR-050, FR-040, FR-118 (SRD 5.2.1, "Level 2: Action Surge", p. 48) - a fighter
// takes the Attack action, is refused a second action, is granted an additional action that is not the Magic action, attacks again through
// it, and is refused a second grant; the grant is gone on the next turn.
// Path: happy.
func TestFighterActionSurgeAddsOneActionForOneTurn(t *testing.T) {
	t.Parallel()
	ctx, hero, goblin := context.Background(), combat.MustCombatantID("hero"), combat.MustCombatantID("goblin")
	tr := grpETracker(t, 30, hero, goblin)
	avail := combat.ConditionActionAvailabilityAvailable
	attack := func(tracker combat.TurnTracker) combat.TakeActionResult {
		t.Helper()
		cmd, err := combat.NewTakeActionCommand(tracker, hero, avail, combat.SimpleActionChoice{Action: combat.SimpleStandardActionAttack})
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewTakeAction().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	tr = attack(tr).(combat.TakeActionApplied).Tracker()
	if r, ok := attack(tr).(combat.TakeActionRejected); !ok || r.Reason != (combat.TurnActionAlreadyTakenRejection{}) {
		t.Fatalf("a second action before the grant = %#v, want ActionAlreadyTaken", r)
	}

	scope, err := combat.NewAnyActionBut(combat.StandardActionMagic)
	if err != nil {
		t.Fatal(err)
	}
	grant, err := combat.NewAdditionalActionGrant("action_surge", scope)
	if err != nil {
		t.Fatal(err)
	}
	grantCmd, err := combat.NewGrantAdditionalActionCommand(tr, hero, grant)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewGrantAdditionalAction().Run(ctx, grantCmd)
	if err != nil {
		t.Fatal(err)
	}
	granted := res.(combat.GrantAdditionalActionApplied)
	if !slices.Equal(grpETurnKinds(granted.Events()), []string{"additional_action_granted"}) {
		t.Fatalf("grant events = %v, want additional_action_granted", grpETurnKinds(granted.Events()))
	}
	if g, state := granted.Tracker().AdditionalAction(); !g.Equal(grant) || state != combat.ResourceAvailable {
		t.Fatalf("the tracker holds %v %v, want the grant available", g, state)
	}
	surged := attack(granted.Tracker()).(combat.TakeActionApplied).Tracker()
	if _, state := surged.AdditionalAction(); state != combat.ResourceSpent || surged.Resources().Action() != combat.ResourceSpent {
		t.Fatalf("after the surge attack the additional action is %v and the action %v, want both spent [FR-155]", state, surged.Resources().Action())
	}
	if r, ok := attack(surged).(combat.TakeActionRejected); !ok || r.Reason != (combat.TurnActionAlreadyTakenRejection{}) {
		t.Errorf("a third action = %#v, want ActionAlreadyTaken", r)
	}
	again, err := combat.NewGrantAdditionalActionCommand(surged, hero, grant)
	if err != nil {
		t.Fatal(err)
	}
	if res, err := combat.NewGrantAdditionalAction().Run(ctx, again); err != nil {
		t.Fatal(err)
	} else if r, ok := res.(combat.GrantAdditionalActionRejected); !ok || r.Reason != (combat.TurnAdditionalActionAlreadyGrantedRejection{}) {
		t.Errorf("a second grant in one turn = %#v, want AdditionalActionAlreadyGranted [FR-155]", res)
	}
	if next := grpEAdvance(t, grpEAdvance(t, surged)); next.Active().Combatant != hero || !func() bool { g, _ := next.AdditionalAction(); return g.IsZero() }() {
		t.Errorf("the next turn still holds a grant, want it cleared when the turn ended [FR-040]")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-155, FR-118 and section 6 command errors - an additional action is refused off-turn
// and for an unknown combatant, the Magic action is not permitted by a scope that excludes it (it spends the regular action instead), an
// OnlyActions grant with no action is invalid input, and the zero grant is invalid input.
// Path: unhappy (state errors, invalid input, rule interaction).
func TestAdditionalActionRefusalsScopeAndInvalidGrants(t *testing.T) {
	t.Parallel()
	ctx, hero, goblin, stranger := context.Background(), combat.MustCombatantID("hero"), combat.MustCombatantID("goblin"), combat.MustCombatantID("stranger")
	tr := grpETracker(t, 30, hero, goblin)
	scope, err := combat.NewAnyActionBut(combat.StandardActionMagic)
	if err != nil {
		t.Fatal(err)
	}
	grant, err := combat.NewAdditionalActionGrant("action_surge", scope)
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		who  combat.CombatantID
		want combat.TurnRejection
	}{
		"off turn":          {goblin, combat.TurnNotActiveCombatantRejection{}},
		"unknown combatant": {stranger, combat.TurnUnknownCombatantRejection{ID: stranger}},
	} {
		cmd, err := combat.NewGrantAdditionalActionCommand(tr, c.who, grant)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewGrantAdditionalAction().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		if r, ok := res.(combat.GrantAdditionalActionRejected); !ok || r.Reason != c.want {
			t.Errorf("%s: result = %#v, want %#v [FR-155, FR-118]", name, res, c.want)
		}
	}

	// A scope that excludes Magic does not permit it: the Magic action spends the regular action, leaving the additional one unspent.
	cmd, err := combat.NewGrantAdditionalActionCommand(tr, hero, grant)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewGrantAdditionalAction().Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	magic, err := combat.NewTakeActionCommand(res.(combat.GrantAdditionalActionApplied).Tracker(), hero, combat.ConditionActionAvailabilityAvailable,
		combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic})
	if err != nil {
		t.Fatal(err)
	}
	taken, err := combat.NewTakeAction().Run(ctx, magic)
	if err != nil {
		t.Fatal(err)
	}
	cast := taken.(combat.TakeActionApplied).Tracker()
	if _, state := cast.AdditionalAction(); state != combat.ResourceAvailable || cast.Resources().Action() != combat.ResourceSpent {
		t.Errorf("Magic action = additional %v action %v, want the regular action spent and the additional one kept [FR-155]", state, cast.Resources().Action())
	}

	if _, err := combat.NewOnlyActions(); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("OnlyActions with no action err = %v, want ErrInvalidInput [FR-155]", err)
	}
	if _, err := combat.NewAdditionalActionGrant("action_surge", nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("grant without a scope err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewGrantAdditionalActionCommand(tr, hero, combat.AdditionalActionGrant{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero grant err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-157, FR-035 (SRD 5.2.1, "Savage Attacker", p. 87, "Once per turn"; "Boon of Combat
// Prowess", Peerless Aim, p. 88) - a once-per-turn feat can be recorded once in a turn, is available again after the turn ends, and a feat
// usable once until the start of the owner's next turn stays recorded until that turn starts.
// Path: unhappy (state error).
func TestFeatUseWindowsOncePerTurnAndUntilNextTurnStart(t *testing.T) {
	t.Parallel()
	ctx, hero, goblin := context.Background(), combat.MustCombatantID("hero"), combat.MustCombatantID("goblin")
	tr := grpETracker(t, 30, hero, goblin)
	record := func(tracker combat.TurnTracker, who combat.CombatantID, feat content.FeatID, limit character.FeatUsageLimit) combat.RecordFeatUseResult {
		t.Helper()
		cmd, err := combat.NewRecordFeatUseCommand(tracker, who, feat, limit)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewRecordFeatUse().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	first := record(tr, hero, "savage_attacker", character.OncePerTurn{}).(combat.RecordFeatUseApplied)
	if !slices.Equal(grpETurnKinds(first.Events()), []string{"feat_use_recorded"}) {
		t.Fatalf("events = %v, want feat_use_recorded [FR-157]", grpETurnKinds(first.Events()))
	}
	if r, ok := record(first.Tracker(), hero, "savage_attacker", character.OncePerTurn{}).(combat.RecordFeatUseRejected); !ok || r.Reason != (combat.TurnFeatUseLimitedRejection{Feat: "savage_attacker"}) {
		t.Errorf("a second use in the same turn = %#v, want FeatUseLimited [FR-157]", r)
	}
	// A different feat, and the same feat by another combatant, are separate windows.
	if _, ok := record(first.Tracker(), hero, "alert", character.OncePerTurn{}).(combat.RecordFeatUseApplied); !ok {
		t.Error("a different feat was refused")
	}
	if _, ok := record(first.Tracker(), goblin, "savage_attacker", character.OncePerTurn{}).(combat.RecordFeatUseApplied); !ok {
		t.Error("the same feat by another combatant was refused")
	}

	both := record(first.Tracker(), hero, "peerless_aim", character.OnceUntilStartOfYourNextTurn{}).(combat.RecordFeatUseApplied).Tracker()
	afterTurn := grpEAdvance(t, both) // the hero's turn ends: the once-per-turn window closes
	if _, ok := record(afterTurn, hero, "savage_attacker", character.OncePerTurn{}).(combat.RecordFeatUseApplied); !ok {
		t.Error("the once-per-turn feat was still limited after the turn ended [FR-157]")
	}
	if _, ok := record(afterTurn, hero, "peerless_aim", character.OnceUntilStartOfYourNextTurn{}).(combat.RecordFeatUseRejected); !ok {
		t.Error("the until-next-turn feat was available in the middle of the window [FR-157]")
	}
	nextTurn := grpEAdvance(t, afterTurn) // the hero's next turn starts
	if nextTurn.Active().Combatant != hero {
		t.Fatalf("active = %v, want the hero", nextTurn.Active().Combatant)
	}
	if _, ok := record(nextTurn, hero, "peerless_aim", character.OnceUntilStartOfYourNextTurn{}).(combat.RecordFeatUseApplied); !ok {
		t.Error("the until-next-turn feat stayed limited after the owner's next turn started [FR-157]")
	}

	if r, ok := record(tr, combat.MustCombatantID("stranger"), "savage_attacker", character.OncePerTurn{}).(combat.RecordFeatUseRejected); !ok ||
		r.Reason != (combat.TurnUnknownCombatantRejection{ID: "stranger"}) {
		t.Errorf("unknown combatant = %#v, want UnknownCombatant", r)
	}
	if _, err := combat.NewRecordFeatUseCommand(tr, hero, "savage_attacker", character.NoUsageLimit{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a limit that is not turn state err = %v, want ErrInvalidInput [FR-157]", err)
	}
	if _, err := combat.NewRecordFeatUseCommand(tr, hero, "savage_attacker", nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil limit err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewRecordFeatUseCommand(tr, hero, "", character.OncePerTurn{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank feat err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-156, FR-019 (SRD 5.2.1, "Alert", Initiative Swap, p. 87) - after Initiative is rolled a
// character with Alert swaps counts with a willing ally: the counts are exchanged and the list order kept, and the swapped standings then
// establish an order in which the ally acts first; a swap that creates a tie leaves the tie to EstablishInitiativeOrder.
// Path: happy.
func TestAlertInitiativeSwapChangesWhoActsFirst(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	hero, ally, goblin := combat.MustCombatantID("hero"), combat.MustCombatantID("ally"), combat.MustCombatantID("goblin")
	standings := []combat.InitiativeStanding{{Combatant: hero, Count: 8}, {Combatant: ally, Count: 18}, {Combatant: goblin, Count: 12}}
	avail := combat.ConditionActionAvailabilityAvailable
	cmd, err := combat.NewSwapInitiativeCommand(standings, hero, ally, character.InitiativeSwap{}, avail, avail, combat.AllyConsentWilling)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewSwapInitiative().Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	swapped := res.(combat.SwapInitiativeApplied)
	want := []combat.InitiativeStanding{{Combatant: hero, Count: 18}, {Combatant: ally, Count: 8}, {Combatant: goblin, Count: 12}}
	if !slices.Equal(swapped.Standings(), want) || !slices.Equal(grpETurnKinds(swapped.Events()), []string{"initiative_swapped"}) {
		t.Fatalf("swap = %v %v, want the counts exchanged in place [FR-156]", swapped.Standings(), grpETurnKinds(swapped.Events()))
	}
	if standings[0].Count != 8 {
		t.Error("the swap altered the caller's standings")
	}

	speeds, _ := combat.NewMovementSpeeds(30)
	profile := func(id combat.CombatantID) combat.CombatantProfile {
		return combat.CombatantProfile{ID: id, Size: content.SizeMedium, Speeds: speeds}
	}
	profiles := []combat.CombatantProfile{profile(hero), profile(ally), profile(goblin)}
	est, err := combat.NewEstablishInitiativeOrderCommand(swapped.Standings(), profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	order, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	if tr := order.(combat.InitiativeOrderEstablished).Tracker(); tr.Active().Combatant != hero {
		t.Errorf("first to act = %v, want the hero at the count the ally rolled", tr.Active().Combatant)
	}

	// Swapping the hero (8) with the goblin (12) in a list where the ally also holds 12 creates a tie that EstablishInitiativeOrder reports.
	tied := []combat.InitiativeStanding{{Combatant: hero, Count: 18}, {Combatant: ally, Count: 12}, {Combatant: goblin, Count: 12}}
	tie, err := combat.NewSwapInitiativeCommand(tied, hero, goblin, character.InitiativeSwap{}, avail, avail, combat.AllyConsentWilling)
	if err != nil {
		t.Fatal(err)
	}
	res, err = combat.NewSwapInitiative().Run(ctx, tie)
	if err != nil {
		t.Fatal(err)
	}
	est, err = combat.NewEstablishInitiativeOrderCommand(res.(combat.SwapInitiativeApplied).Standings(), profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	if u, ok := result.(combat.InitiativeTiesUnresolved); !ok || len(u.Groups()) != 1 {
		t.Errorf("order after a swap that ties = %#v, want one unresolved tie left to the host [FR-156, FR-019]", result)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-156, FR-118 and section 6 command errors - a swap is refused for an unknown combatant
// (swapper, then ally), an incapacitated side and an unwilling ally, in that order; the same combatant on both sides, an empty list, a
// repeated standing and a cancelled context are errors.
// Path: unhappy (state errors, invalid input).
func TestInitiativeSwapRefusalsAndInvalidCommands(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a, b := combat.MustCombatantID("a"), combat.MustCombatantID("b")
	standings := []combat.InitiativeStanding{{Combatant: a, Count: 15}, {Combatant: b, Count: 10}}
	avail, none := combat.ConditionActionAvailabilityAvailable, combat.ConditionActionAvailabilityNone
	swap := func(swapper, ally combat.CombatantID, sav, aav combat.ConditionActionAvailability, consent combat.AllyConsent) combat.TurnRejection {
		t.Helper()
		cmd, err := combat.NewSwapInitiativeCommand(standings, swapper, ally, character.InitiativeSwap{}, sav, aav, consent)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewSwapInitiative().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res.(combat.SwapInitiativeRejected).Reason
	}
	if got := swap(a, b, avail, avail, combat.AllyConsentUnwilling); got != (combat.TurnAllyNotWillingRejection{}) {
		t.Errorf("unwilling ally = %#v, want AllyNotWilling", got)
	}
	if got := swap(a, b, none, avail, combat.AllyConsentWilling); got != (combat.TurnIncapacitatedRejection{}) {
		t.Errorf("incapacitated swapper = %#v, want incapacitated", got)
	}
	if got := swap(a, b, avail, none, combat.AllyConsentUnwilling); got != (combat.TurnIncapacitatedRejection{}) {
		t.Errorf("incapacitated ally who is also unwilling = %#v, want incapacitated first [FR-118 order]", got)
	}
	stranger := combat.MustCombatantID("stranger")
	if got := swap(a, stranger, avail, avail, combat.AllyConsentWilling); got != (combat.TurnUnknownCombatantRejection{ID: stranger}) {
		t.Errorf("an ally outside the standings = %#v, want UnknownCombatant", got)
	}

	cmd := func(list []combat.InitiativeStanding, swapper, ally combat.CombatantID) error {
		_, err := combat.NewSwapInitiativeCommand(list, swapper, ally, character.InitiativeSwap{}, avail, avail, combat.AllyConsentWilling)
		return err
	}
	if err := cmd(standings, a, a); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the same combatant on both sides err = %v, want ErrInvalidInput", err)
	}
	if err := cmd(nil, a, b); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("empty standings err = %v, want ErrInvalidInput", err)
	}
	if err := cmd([]combat.InitiativeStanding{{Combatant: a, Count: 1}, {Combatant: a, Count: 2}}, a, b); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a repeated combatant err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewSwapInitiativeCommand(standings, a, b, character.InitiativeSwap{}, avail, avail, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("undefined consent err = %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	ok, err := combat.NewSwapInitiativeCommand(standings, a, b, character.InitiativeSwap{}, avail, avail, combat.AllyConsentWilling)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := combat.NewSwapInitiative().Run(cancelled, ok); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context err = %v, want context.Canceled", err)
	}
	if _, err := combat.NewSwapInitiative().Run(ctx, combat.SwapInitiativeCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero command err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-116, FR-115, FR-117, FR-083, FR-089 (SRD 5.2.1, "Speed", "Changes to Your Speeds",
// p. 188) - a combatant with Speed 30 moves 20 feet, a spell halves their Speed to 15 and moved feet are not refunded (no movement remains);
// the lowered Speed persists to the next round, where 15 feet is allowed and 16 is short by 1; then a Fly speed is gained.
// Path: unhappy (state error, boundary).
func TestSlowedCombatantKeepsMovedFeetAndNewSpeedPersists(t *testing.T) {
	t.Parallel()
	ctx, hero, goblin := context.Background(), combat.MustCombatantID("hero"), combat.MustCombatantID("goblin")
	tr := grpETracker(t, 30, hero, goblin)
	move := func(tracker combat.TurnTracker, feet int, mode combat.MovementMode) combat.MoveResult {
		t.Helper()
		seg, err := combat.NewMoveSegment(combat.MoveSegmentInput{DistanceFeet: feet, Mode: mode})
		if err != nil {
			t.Fatal(err)
		}
		cmd, err := combat.NewMoveCommand(tracker, tracker.Active().Combatant, []combat.MoveSegment{seg})
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewMove().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	tr = move(tr, 20, combat.MovementModeWalk).(combat.MoveApplied).Tracker()

	halved := tr.Order() // keep the order for later comparison
	speedOf := func(tracker combat.TurnTracker) combat.MovementSpeeds {
		p, ok := tracker.ProfileOf(hero)
		if !ok {
			t.Fatal("the hero has no profile")
		}
		return p.Speeds
	}
	slow := speedOf(tr).Halved()
	update := func(tracker combat.TurnTracker, who combat.CombatantID, speeds combat.MovementSpeeds) combat.UpdateMovementSpeedsResult {
		t.Helper()
		cmd, err := combat.NewUpdateMovementSpeedsCommand(tracker, who, speeds)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewUpdateMovementSpeeds().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	slowed := update(tr, hero, slow).(combat.UpdateMovementSpeedsApplied)
	if !slices.Equal(grpETurnKinds(slowed.Events()), []string{"movement_speeds_updated"}) || slowed.Tracker().Resources().MovedFeet() != 20 {
		t.Fatalf("update = %v moved %d, want movement_speeds_updated and the 20 feet not refunded [FR-116]", grpETurnKinds(slowed.Events()), slowed.Tracker().Resources().MovedFeet())
	}
	if r, ok := move(slowed.Tracker(), 5, combat.MovementModeWalk).(combat.MoveRejected); !ok || r.Reason != (combat.TurnNoMovementRemainingRejection{SegmentIndex: 0, MovementLeft: -5}) {
		t.Fatalf("a move after the Speed fell below the feet moved = %#v, want NoMovementRemaining [FR-089, FR-115]", r)
	}
	_ = halved

	// Round 2: the profile keeps the lowered Speed (FR-117).
	round2 := grpEAdvance(t, grpEAdvance(t, slowed.Tracker()))
	if round2.Round() != 2 || round2.Active().Combatant != hero || speedOf(round2).Walk() != 15 || round2.Resources().MovedFeet() != 0 {
		t.Fatalf("round %d active %v Speed %d moved %d, want round 2 with Speed 15 and a fresh turn [FR-117]", round2.Round(), round2.Active().Combatant,
			speedOf(round2).Walk(), round2.Resources().MovedFeet())
	}
	if r, ok := move(round2, 16, combat.MovementModeWalk).(combat.MoveRejected); !ok || r.Reason != (combat.TurnInsufficientMovementRejection{SegmentIndex: 0, ShortfallFeet: 1}) {
		t.Errorf("16 feet at Speed 15 = %#v, want InsufficientMovement short by 1 [FR-115]", r)
	}
	if _, ok := move(round2, 15, combat.MovementModeWalk).(combat.MoveApplied); !ok {
		t.Error("15 feet at Speed 15 was refused")
	}

	// A flying Speed is gained mid-turn; the walking part is kept and a fly segment is now allowed.
	flying, err := combat.NewMovementSpeeds(15, combat.MovementSpeedEntry{Name: combat.SpeedNameFly, Feet: 60})
	if err != nil {
		t.Fatal(err)
	}
	winged := update(round2, hero, flying).(combat.UpdateMovementSpeedsApplied).Tracker()
	if fly, ok := speedOf(winged).Of(combat.SpeedNameFly); !ok || fly != 60 {
		t.Fatalf("Fly speed = %d, %v, want 60", fly, ok)
	}
	if _, ok := move(winged, 60, combat.MovementModeFly).(combat.MoveApplied); !ok {
		t.Error("a 60 foot flight was refused after the Fly speed was gained")
	}

	// Refusals and errors.
	stranger := combat.MustCombatantID("stranger")
	if r, ok := update(tr, stranger, slow).(combat.UpdateMovementSpeedsRejected); !ok || r.Reason != (combat.TurnUnknownCombatantRejection{ID: stranger}) {
		t.Errorf("unknown combatant = %#v, want UnknownCombatant [FR-116]", r)
	}
	if _, err := combat.NewUpdateMovementSpeedsCommand(combat.TurnTracker{}, hero, slow); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero tracker err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewUpdateMovementSpeedsCommand(tr, "", slow); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank id err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewMovementSpeeds(-5); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative walking Speed err = %v, want ErrInvalidInput", err)
	}
}
