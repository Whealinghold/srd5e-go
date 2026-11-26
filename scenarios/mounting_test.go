package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/mounted-and-underwater-combat.md section 8 (scenario), FR-010, FR-019, FR-021, FR-028, FR-031, FR-039 - a rider
// with Speed 30 mounts a trained, willing Large horse during their move (cost 15, controlled, Initiative to match), a forced-movement
// effect triggers the DC 10 Dexterity save and a fixed roller fails it (the rider falls off and the event names Prone, which the caller
// applies to the rider's conditions), then underwater adds Resistance to Fire and on land the base returns, through the public API only.
// Path: happy.
func TestRiderMountsFallsOffAndFireIsResistedUnderwater(t *testing.T) {
	t.Parallel()
	ctx, rider, horse := context.Background(), combat.MustCombatantID("rider"), combat.MustCombatantID("horse")
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
	if mounted.MovementCostFeet() != 15 {
		t.Fatalf("cost = %d, want 15", mounted.MovementCostFeet())
	}
	ev := mounted.Events()[0].(combat.CreatureMounted)
	if ev.InitiativeChange() != (combat.MountInitiativeMatchesRider{Rider: rider}) || ev.Turn() != combat.MountTurnOnRidersTurn {
		t.Errorf("event = %+v, want the Initiative matched to the rider on the rider's turn", ev)
	}

	attempt := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(2))}
	fall, err := combat.NewResolveFallingOffCommand(rider, mounted.RiderStatus(), combat.FallingOffTriggerMountMovedAgainstItsWill, attempt, combat.LandingSpaceUnoccupiedWithin5Feet)
	if err != nil {
		t.Fatal(err)
	}
	uc, err := combat.NewResolveFallingOff(dicetest.NewFixedRoller(3))
	if err != nil {
		t.Fatal(err)
	}
	out, err := uc.Run(ctx, fall)
	if err != nil {
		t.Fatal(err)
	}
	fell, ok := out.(combat.ResolveFallingOffFellOff)
	if !ok || fell.RiderStatus() != combat.NotMounted() {
		t.Fatalf("result = %#v, want fell off and NotMounted (3 + 2 is under DC 10)", out)
	}
	landed := fell.Events()[0].(combat.FellOffMount)
	if landed.Condition() != content.ConditionProne {
		t.Fatalf("condition = %v, want Prone", landed.Condition())
	}
	if _, ok := (combat.ConditionSet{}).Apply(rider, combat.Prone{}).(combat.ConditionAdded); !ok {
		t.Error("the caller applies the Prone the event names")
	}

	wet := combat.UnderwaterFireResistanceAffinities(combat.NoAffinities(), combat.UnderwaterStatusUnderwater, combat.UnderwaterResistanceGranted)
	dry := combat.UnderwaterFireResistanceAffinities(combat.NoAffinities(), combat.UnderwaterStatusNotUnderwater, combat.UnderwaterResistanceGranted)
	taken := func(aff combat.DamageAffinities) int {
		st := combat.MustHitPointStatus(100, combat.FallsUnconscious, aff)
		tr, err := st.ApplyDamage(rider, []combat.Damage{combat.MustDamage(15, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit, nil)
		if err != nil {
			t.Fatal(err)
		}
		return tr.(combat.HitPointTransitionApplied).Resolutions()[0].Dealt()
	}
	if got, want := taken(wet), 7; got != want {
		t.Errorf("15 Fire damage underwater = %d, want %d", got, want)
	}
	if got, want := taken(dry), 15; got != want || !dry.Equal(combat.NoAffinities()) {
		t.Errorf("15 Fire damage on land = %d, want %d with the base returned", got, want)
	}
}
