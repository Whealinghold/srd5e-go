package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/initiative-and-actions.md FR-018, FR-029, FR-035, FR-051, FR-066, FR-068 - roll Initiative for singles and a group with a fixed roller, resolve a tie, then walk two rounds with Dash, a Bonus Action, an Opportunity Attack and a readied action, through the public packages only.
// Path: happy.
func TestInitiativeAndTurnsScenario(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := combat.MustCombatantID
	hero, rogue, goblins := id("hero"), id("rogue"), []combat.CombatantID{id("goblin-1"), id("goblin-2")}

	// Initiative: hero +2 rolls 13, rogue +3 rolls 12 (both 15), the goblins roll once together: 15.
	person := func(mod int, ids ...combat.CombatantID) combat.InitiativeParticipant {
		p, err := combat.NewInitiativeParticipant(combat.InitiativeParticipantInput{IDs: ids, Modifier: modifier.New(mod)})
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	cmd, err := combat.NewRollInitiativeCommand(combat.InitiativeMethodRolled, []combat.InitiativeParticipant{person(2, hero), person(3, rogue), person(2, goblins...)})
	if err != nil {
		t.Fatal(err)
	}
	roll, err := combat.NewRollInitiative(dicetest.NewFixedRoller(13, 12, 8))
	if err != nil {
		t.Fatal(err)
	}
	rolled, err := roll.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	var standings []combat.InitiativeStanding
	for _, b := range rolled.(combat.RollInitiativeRolled).Breakdowns() {
		for _, c := range b.IDs() {
			standings = append(standings, combat.InitiativeStanding{Combatant: c, Count: b.Count()})
		}
	}
	profile := func(c combat.CombatantID, size content.CreatureSize) combat.CombatantProfile {
		s, err := combat.NewMovementSpeeds(30)
		if err != nil {
			t.Fatal(err)
		}
		return combat.CombatantProfile{ID: c, Size: size, Speeds: s}
	}
	profiles := []combat.CombatantProfile{profile(hero, content.SizeMedium), profile(rogue, content.SizeMedium), profile(goblins[0], content.SizeSmall), profile(goblins[1], content.SizeSmall)}

	// Two ties: the hero and the rogue at 15, and the two goblins (identical creatures rolled once) at 10; the host settles both (FR-019).
	establish := combat.NewEstablishInitiativeOrder()
	first, _ := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	res, err := establish.Run(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	tie, ok := res.(combat.InitiativeTiesUnresolved)
	if !ok || len(tie.Groups()) != 2 {
		t.Fatalf("result = %#v, want two unresolved ties (hero and rogue at 15, the two goblins at 10)", res)
	}
	tb, err := combat.NewTieBreak([]combat.CombatantID{rogue, hero})
	if err != nil {
		t.Fatal(err)
	}
	gb, err := combat.NewTieBreak(goblins)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := combat.NewEstablishInitiativeOrderCommand(standings, profiles, []combat.TieBreak{tb, gb})
	res, err = establish.Run(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	est := res.(combat.InitiativeOrderEstablished)
	tr := est.Tracker()
	var order []combat.CombatantID
	for _, s := range tr.Order().Standings() {
		order = append(order, s.Combatant)
	}
	if !slices.Equal(order, []combat.CombatantID{rogue, hero, goblins[0], goblins[1]}) || tr.Round() != 1 || tr.Active().Combatant != rogue {
		t.Fatalf("order %v round %d active %v, want rogue, hero, goblins in round 1 with the rogue active [FR-018]", order, tr.Round(), tr.Active())
	}

	// Round 1, the rogue: a Cunning Action Dash by Bonus Action, then a Move of 60 feet. The hero reacts with an Opportunity Attack.
	grant, err := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "cunning_action"}, combat.GrantsDash{Speed: combat.SpeedNameWalk})
	if err != nil {
		t.Fatal(err)
	}
	bonusCmd, _ := combat.NewTakeBonusActionCommand(tr, rogue, combat.ConditionActionAvailabilityAvailable, combat.EntitledToBonusAction{Grant: grant})
	bonus, err := combat.NewTakeBonusAction().Run(ctx, bonusCmd)
	if err != nil {
		t.Fatal(err)
	}
	tr = bonus.(combat.TakeBonusActionApplied).Tracker()
	seg, err := combat.NewMoveSegment(combat.MoveSegmentInput{DistanceFeet: 30, Mode: combat.MovementModeWalk})
	if err != nil {
		t.Fatal(err)
	}
	moveCmd, _ := combat.NewMoveCommand(tr, rogue, []combat.MoveSegment{seg, seg})
	moved, err := combat.NewMove().Run(ctx, moveCmd)
	if err != nil {
		t.Fatal(err)
	}
	tr = moved.(combat.MoveApplied).Tracker()
	if tr.Resources().MovedFeet() != 60 || tr.Resources().DashExtraFeet() != 30 {
		t.Fatalf("resources = %+v, want 60 ft moved with a 30 ft Dash extra [FR-051]", tr.Resources())
	}
	oaCmd, _ := combat.NewTakeReactionCommand(tr, hero, combat.ConditionActionAvailabilityAvailable,
		combat.OpportunityAttackReaction{Provoker: rogue, Means: combat.DepartureBySpeed, Visibility: combat.VisibilitySeen})
	oa, err := combat.NewTakeReaction().Run(ctx, oaCmd)
	if err != nil {
		t.Fatal(err)
	}
	tr = oa.(combat.TakeReactionApplied).Tracker()
	if !tr.ReactionSpent(hero) {
		t.Fatal("the Opportunity Attack did not spend the hero's Reaction [FR-068]")
	}
	againCmd, _ := combat.NewTakeReaction().Run(ctx, mustReaction(t, tr, hero, rogue))
	if r, ok := againCmd.(combat.TakeReactionRejected); !ok || r.Reason != (combat.TurnReactionAlreadyTakenRejection{}) {
		t.Fatalf("a second Reaction = %#v, want ReactionAlreadyTaken [FR-066]", againCmd)
	}

	// The hero readies an attack on its turn; the goblins act; round 2 begins; the readied action expires at the hero's next turn and its Reaction is regained.
	advance := func() {
		t.Helper()
		cmd, err := combat.NewAdvanceTrackerTurnCommand(tr, combat.TurnAdvance{})
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewAdvanceTrackerTurn().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		tr = res.(combat.AdvanceTrackerTurnApplied).Tracker()
	}
	advance() // the hero
	door := combat.MustReadyTriggerID("door opens")
	ready, err := combat.NewReadiedAction(door, combat.ReadiedActionResponse{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionAttack}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	acted, err := combat.NewTakeActionCommand(tr, hero, combat.ConditionActionAvailabilityAvailable, combat.ReadyChoice{Action: ready})
	if err != nil {
		t.Fatal(err)
	}
	readied, err := combat.NewTakeAction().Run(ctx, acted)
	if err != nil {
		t.Fatal(err)
	}
	tr = readied.(combat.TakeActionApplied).Tracker()
	if _, held := tr.ReadiedActionOf(hero); !held || tr.Active().Combatant != hero {
		t.Fatal("the hero holds no readied action")
	}
	advance() // goblin-1: the trigger fires and the hero's readied attack is taken as its Reaction
	fired, err := combat.NewTakeReactionCommand(tr, hero, combat.ConditionActionAvailabilityAvailable, combat.ReadiedActionReaction{Trigger: door})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := combat.NewTakeReaction().Run(ctx, fired)
	if err != nil {
		t.Fatal(err)
	}
	tr = resp.(combat.TakeReactionApplied).Tracker()
	if _, held := tr.ReadiedActionOf(hero); held || !tr.ReactionSpent(hero) {
		t.Fatal("the readied action was not taken as the Reaction [FR-073]")
	}
	advance() // goblin-2
	advance() // the rogue, round 2
	if tr.Round() != 2 || tr.Active().Combatant != rogue || !tr.ReactionSpent(hero) {
		t.Fatalf("round %d active %v hero spent %v, want round 2, the rogue, and the hero's Reaction still spent [FR-036, FR-029]", tr.Round(), tr.Active(), tr.ReactionSpent(hero))
	}
	advance() // the hero's turn in round 2
	if tr.ReactionSpent(hero) || tr.Active().Combatant != hero {
		t.Fatalf("Reaction spent %v, want it regained at the hero's own turn [FR-035]", tr.ReactionSpent(hero))
	}
}

func mustReaction(t *testing.T, tr combat.TurnTracker, reactor, provoker combat.CombatantID) combat.TakeReactionCommand {
	t.Helper()
	cmd, err := combat.NewTakeReactionCommand(tr, reactor, combat.ConditionActionAvailabilityAvailable,
		combat.OpportunityAttackReaction{Provoker: provoker, Means: combat.DepartureBySpeed, Visibility: combat.VisibilitySeen})
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}
