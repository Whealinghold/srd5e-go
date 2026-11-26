package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpEEncounter starts an encounter of Medium combatants, each (id, side, count), through the public use cases.
type grpEMember struct {
	id    combat.CombatantID
	side  combat.Side
	count int
}

func grpEStartEncounter(t *testing.T, name string, members ...grpEMember) combat.Encounter {
	t.Helper()
	ctx := context.Background()
	speeds, err := combat.NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	var combatants []combat.Combatant
	for _, m := range members {
		standings = append(standings, combat.InitiativeStanding{Combatant: m.id, Count: combat.InitiativeCount(m.count)})
		profiles = append(profiles, combat.CombatantProfile{ID: m.id, Size: content.SizeMedium, Speeds: speeds})
		hp, err := combat.RestoreHitPointStatus(combat.MustHitPoints(10, 10, 0), combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
		if err != nil {
			t.Fatal(err)
		}
		c, err := combat.NewCombatant(combat.CombatantRef{ID: m.id, Kind: combat.MonsterCombatant{ID: "fixture"}}, m.side, hp, combat.ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
		combatants = append(combatants, c)
	}
	est, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	order, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	started, err := combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID(name), order.(combat.InitiativeOrderEstablished), combatants))
	if err != nil {
		t.Fatal(err)
	}
	return started.(combat.StartEncounterApplied).Encounter()
}

func grpEOrder(e combat.Encounter) []combat.CombatantID {
	var ids []combat.CombatantID
	for _, s := range e.Tracker().Order().Standings() {
		ids = append(ids, s.Combatant)
	}
	return ids
}

func grpECountOf(e combat.Encounter, id combat.CombatantID) combat.InitiativeCount {
	for _, s := range e.Tracker().Order().Standings() {
		if s.Combatant == id {
			return s.Count
		}
	}
	return -1
}

// Requirement: doc/specs/combatant-roster-changes.md FR-028, FR-002, FR-003 and doc/specs/mounted-and-underwater-combat.md FR-010, FR-014, FR-016,
// FR-021 (SRD 5.2.1, "Controlling a Mount", p. 16; "Mounting and Dismounting", p. 15) - a rider (Initiative 18) mounts a horse (Initiative 6)
// and controls it, so the host repositions the horse directly after the rider with the rider's count; the rider then dismounts for half their
// Speed and the horse goes back to its own count, which lands after the goblin (14).
// Path: happy.
func TestControlledMountSharesRidersInitiativeUntilDismount(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rider, horse, goblin := combat.MustCombatantID("rider"), combat.MustCombatantID("horse"), combat.MustCombatantID("goblin")
	e := grpEStartEncounter(t, "mount-scenario", grpEMember{rider, combat.SideA, 18}, grpEMember{goblin, combat.SideB, 14}, grpEMember{horse, combat.SideA, 6})
	if got := grpEOrder(e); !slices.Equal(got, []combat.CombatantID{rider, goblin, horse}) {
		t.Fatalf("starting order = %v, want rider, goblin, horse", got)
	}

	facts, err := combat.NewMountFacts(combat.MountWillingnessWilling, combat.MountAnatomyAppropriate, combat.MountTrainingTrainedToAcceptRider,
		combat.MountHabitatOther, combat.ExoticSaddleNotFitted)
	if err != nil {
		t.Fatal(err)
	}
	turn, err := combat.NewMountingTurnFacts(combat.RiderTurnActive, 30, 30)
	if err != nil {
		t.Fatal(err)
	}
	mount, err := combat.NewMountCreatureCommand(combat.MountCreatureInput{Rider: rider, Mount: horse, RiderSize: content.SizeMedium, MountSize: content.SizeLarge,
		Facts: facts, Turn: turn, DistanceFeet: 5, Control: combat.MountControlControlled})
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewMountCreature().Run(ctx, mount)
	if err != nil {
		t.Fatal(err)
	}
	mounted := res.(combat.MountCreatureApplied)
	if ev := mounted.Events()[0].(combat.CreatureMounted); ev.InitiativeChange() != (combat.MountInitiativeMatchesRider{Rider: rider}) {
		t.Fatalf("mount event = %+v, want the Initiative matched to the rider", ev)
	}

	// The host acts on the event: the horse takes its turn directly after the rider, with the rider's count (FR-028, FR-002).
	reposition := func(e combat.Encounter, who combat.CombatantID, pos combat.RosterPosition) combat.RepositionCombatantInEncounterResult {
		t.Helper()
		cmd, err := combat.NewRepositionCombatantInEncounterCommand(e, who, pos)
		if err != nil {
			t.Fatal(err)
		}
		out, err := combat.NewRepositionCombatantInEncounter().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	moved := reposition(e, horse, combat.RosterAfterCombatant{Anchor: rider}).(combat.RepositionCombatantInEncounterApplied)
	e = moved.Encounter()
	if got := grpEOrder(e); !slices.Equal(got, []combat.CombatantID{rider, horse, goblin}) || grpECountOf(e, horse) != 18 || e.Tracker().Active().Combatant != rider {
		t.Fatalf("order = %v horse count %d active %v, want rider, horse, goblin with the horse at 18 and the rider still active [FR-002, FR-028]",
			got, grpECountOf(e, horse), e.Tracker().Active().Combatant)
	}
	if evs := moved.Events(); len(evs) != 1 || evs[0].Kind() != "combatant_repositioned" {
		t.Errorf("events = %v, want the one combatant_repositioned [FR-028]", evs)
	}

	// The rider dismounts, spending half their Speed rounded down (15 feet).
	dismount, err := combat.NewDismountCreatureCommand(rider, mounted.RiderStatus(), turn)
	if err != nil {
		t.Fatal(err)
	}
	off, err := combat.NewDismountCreature().Run(ctx, dismount)
	if err != nil {
		t.Fatal(err)
	}
	down, ok := off.(combat.DismountCreatureApplied)
	if !ok || down.RiderStatus() != combat.NotMounted() || down.MovementCostFeet() != 15 || len(down.Events()) != 1 || down.Events()[0].Kind() != "creature_dismounted" {
		t.Fatalf("dismount = %#v, want NotMounted for 15 feet with creature_dismounted", off)
	}
	if ev := down.Events()[0].(combat.CreatureDismounted); ev.MovementCostFeet() != 15 || ev.Control() != combat.MountControlControlled {
		t.Errorf("event = %+v, want the cost 15 and the control the ride had", ev)
	}

	// The horse returns to its own count and comes after the goblin again.
	e = reposition(e, horse, combat.RosterAtInitiative{Count: 6, Tie: combat.TiePlacementAfterTied}).(combat.RepositionCombatantInEncounterApplied).Encounter()
	if got := grpEOrder(e); !slices.Equal(got, []combat.CombatantID{rider, goblin, horse}) || grpECountOf(e, horse) != 6 {
		t.Errorf("order after the dismount = %v horse count %d, want rider, goblin, horse at 6", got, grpECountOf(e, horse))
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-029 and section 6 command errors - repositioning is refused, in order, once the
// encounter has ended, for an unknown subject then an unknown anchor, and for the active combatant (an active anchor is allowed); a zero
// encounter, a blank id, an invalid position and an anchor that is the subject are invalid input; the refused encounter is returned unchanged.
// Path: unhappy (state errors, invalid input).
func TestRepositionRefusalsAndInvalidCommands(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a, b, c, stranger := combat.MustCombatantID("a"), combat.MustCombatantID("b"), combat.MustCombatantID("c"), combat.MustCombatantID("stranger")
	e := grpEStartEncounter(t, "reposition-refusals", grpEMember{a, combat.SideA, 18}, grpEMember{b, combat.SideB, 12}, grpEMember{c, combat.SideA, 6})
	run := func(e combat.Encounter, who combat.CombatantID, pos combat.RosterPosition) combat.RepositionCombatantInEncounterResult {
		t.Helper()
		cmd, err := combat.NewRepositionCombatantInEncounterCommand(e, who, pos)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewRepositionCombatantInEncounter().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	refused := func(name string, res combat.RepositionCombatantInEncounterResult, want combat.RosterRejection, unchanged combat.Encounter) {
		t.Helper()
		r, ok := res.(combat.RepositionCombatantInEncounterRejected)
		if !ok || r.Rejection != want || !r.Encounter().Equal(unchanged) {
			t.Errorf("%s = %#v, want rejection %#v with the encounter unchanged", name, res, want)
		}
	}
	refused("the active combatant", run(e, a, combat.RosterAtInitiative{Count: 1, Tie: combat.TiePlacementAfterTied}), combat.RosterCombatantIsActiveRejection{ID: a}, e)
	refused("an unknown subject", run(e, stranger, combat.RosterAfterCombatant{Anchor: a}), combat.RosterUnknownCombatantRejection{ID: stranger}, e)
	refused("an unknown anchor", run(e, c, combat.RosterAfterCombatant{Anchor: stranger}), combat.RosterUnknownCombatantRejection{ID: stranger}, e)
	if _, ok := run(e, c, combat.RosterAfterCombatant{Anchor: a}).(combat.RepositionCombatantInEncounterApplied); !ok {
		t.Error("an active anchor was refused, want it allowed [FR-029]")
	}

	end, err := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{Side: combat.SideB})
	if err != nil {
		t.Fatal(err)
	}
	over, err := combat.NewEndEncounter().Run(ctx, end)
	if err != nil {
		t.Fatal(err)
	}
	ended := over.(combat.EndEncounterApplied).Encounter()
	refused("an ended encounter", run(ended, c, combat.RosterAtInitiative{Count: 10, Tie: combat.TiePlacementAfterTied}), combat.RosterEncounterEndedRejection{}, ended)

	if _, err := combat.NewRepositionCombatantInEncounterCommand(combat.Encounter{}, c, combat.RosterAtInitiative{Count: 1, Tie: combat.TiePlacementAfterTied}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero encounter err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewRepositionCombatantInEncounterCommand(e, "", combat.RosterAtInitiative{Count: 1, Tie: combat.TiePlacementAfterTied}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank id err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewRepositionCombatantInEncounterCommand(e, c, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil position err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewRepositionCombatantInEncounterCommand(e, c, combat.RosterAfterCombatant{Anchor: c}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an anchor that is the subject err = %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	cmd, err := combat.NewRepositionCombatantInEncounterCommand(e, c, combat.RosterAtInitiative{Count: 10, Tie: combat.TiePlacementAfterTied})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := combat.NewRepositionCombatantInEncounter().Run(cancelled, cmd); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context err = %v, want context.Canceled", err)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-012, FR-015, FR-017 and section 6 command errors (SRD 5.2.1, "Mounting and Dismounting", p. 15) -
// a rider cannot dismount when it is not their turn, when they are not mounted, when no walking movement is left, or when the movement left
// is short of half their Speed (by the shortfall); a mount that is the rider, a blank rider and unvalidated turn facts are invalid input.
// Path: unhappy (state errors, boundary, invalid input).
func TestDismountRefusalsAndInvalidCommands(t *testing.T) {
	t.Parallel()
	ctx, rider, horse := context.Background(), combat.MustCombatantID("rider"), combat.MustCombatantID("horse")
	onHorse, err := combat.MountedOn(horse, combat.MountControlIndependent)
	if err != nil {
		t.Fatal(err)
	}
	facts := func(state combat.RiderTurnState, speed, left int) combat.MountingTurnFacts {
		t.Helper()
		f, err := combat.NewMountingTurnFacts(state, speed, left)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	dismount := func(status combat.RiderStatus, turn combat.MountingTurnFacts) combat.DismountCreatureResult {
		t.Helper()
		cmd, err := combat.NewDismountCreatureCommand(rider, status, turn)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewDismountCreature().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	reason := func(res combat.DismountCreatureResult) combat.DismountRejection {
		t.Helper()
		r, ok := res.(combat.DismountCreatureRejected)
		if !ok {
			t.Fatalf("result = %#v, want a refusal", res)
		}
		return r.Reason()
	}
	if got := reason(dismount(onHorse, facts(combat.RiderTurnNotActive, 30, 30))); got != (combat.MountingRiderNotActiveRejection{}) {
		t.Errorf("not the rider's turn = %#v, want rider_not_active", got)
	}
	if got := reason(dismount(combat.NotMounted(), facts(combat.RiderTurnActive, 30, 30))); got != (combat.MountingRiderNotMountedRejection{}) {
		t.Errorf("a rider on foot = %#v, want rider_not_mounted", got)
	}
	if got := reason(dismount(onHorse, facts(combat.RiderTurnActive, 30, 0))); got != (combat.MountingNoMovementRemainingRejection{}) {
		t.Errorf("no movement left = %#v, want no_movement_remaining", got)
	}
	short, ok := reason(dismount(onHorse, facts(combat.RiderTurnActive, 30, 5))).(combat.MountingInsufficientMovementRejection)
	if !ok || short.ShortfallFeet() != 10 {
		t.Errorf("5 feet left for a 15 foot cost = %#v, want insufficient_movement short by 10", short)
	}
	if ok := dismount(onHorse, facts(combat.RiderTurnActive, 30, 15)); func() bool { _, applied := ok.(combat.DismountCreatureApplied); return !applied }() {
		t.Error("exactly 15 feet left for a 15 foot cost was refused, want it accepted (the boundary)")
	}

	if _, err := combat.NewDismountCreatureCommand("", onHorse, facts(combat.RiderTurnActive, 30, 30)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank rider err = %v, want ErrInvalidInput", err)
	}
	self, err := combat.MountedOn(rider, combat.MountControlControlled)
	if err == nil {
		if _, err := combat.NewDismountCreatureCommand(rider, self, facts(combat.RiderTurnActive, 30, 30)); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("a rider that is their own mount err = %v, want ErrInvalidInput", err)
		}
	}
	if _, err := combat.NewDismountCreatureCommand(rider, onHorse, combat.MountingTurnFacts{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("unvalidated turn facts err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewMountingTurnFacts(combat.RiderTurnActive, -1, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative walking Speed err = %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	cmd, err := combat.NewDismountCreatureCommand(rider, onHorse, facts(combat.RiderTurnActive, 30, 30))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := combat.NewDismountCreature().Run(cancelled, cmd); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context err = %v, want context.Canceled", err)
	}
	if _, err := combat.NewDismountCreature().Run(ctx, combat.DismountCreatureCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero command err = %v, want ErrInvalidInput", err)
	}
}
