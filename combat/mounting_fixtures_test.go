package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func mntFacts(tb testing.TB, w combat.MountWillingness, a combat.MountAnatomy, tr combat.MountTraining, h combat.MountHabitat, s combat.ExoticSaddle) combat.MountFacts {
	tb.Helper()
	f, err := combat.NewMountFacts(w, a, tr, h, s)
	if err != nil {
		tb.Fatal(err)
	}
	return f
}

// mntGoodFacts is a willing, trained ground mount with an appropriate anatomy.
func mntGoodFacts(tb testing.TB) combat.MountFacts {
	tb.Helper()
	return mntFacts(tb, combat.MountWillingnessWilling, combat.MountAnatomyAppropriate, combat.MountTrainingTrainedToAcceptRider,
		combat.MountHabitatOther, combat.ExoticSaddleNotFitted)
}

func mntTurn(tb testing.TB, state combat.RiderTurnState, speed, left int) combat.MountingTurnFacts {
	tb.Helper()
	f, err := combat.NewMountingTurnFacts(state, speed, left)
	if err != nil {
		tb.Fatal(err)
	}
	return f
}

// mntInput is an input every rule accepts: a Medium rider, a Large mount, 5 feet apart, Speed 30 all unspent, controlled.
func mntInput(tb testing.TB) combat.MountCreatureInput {
	tb.Helper()
	return combat.MountCreatureInput{
		Rider: gid("rider"), Mount: gid("horse"), RiderSize: content.SizeMedium, MountSize: content.SizeLarge,
		Facts: mntGoodFacts(tb), Turn: mntTurn(tb, combat.RiderTurnActive, 30, 30), DistanceFeet: 5, Control: combat.MountControlControlled,
	}
}

func mntCommand(tb testing.TB, in combat.MountCreatureInput) combat.MountCreatureCommand {
	tb.Helper()
	c, err := combat.NewMountCreatureCommand(in)
	if err != nil {
		tb.Fatal(err)
	}
	return c
}

// mntMounted is a rider on the "horse" with the control.
func mntMounted(tb testing.TB, control combat.MountControl) combat.RiderStatus {
	tb.Helper()
	s, err := combat.MountedOn(gid("horse"), control)
	if err != nil {
		tb.Fatal(err)
	}
	return s
}

func mntDismountCommand(tb testing.TB, status combat.RiderStatus, turn combat.MountingTurnFacts) combat.DismountCreatureCommand {
	tb.Helper()
	c, err := combat.NewDismountCreatureCommand(gid("rider"), status, turn)
	if err != nil {
		tb.Fatal(err)
	}
	return c
}

// mntFall runs ResolveFallingOff for a rider mounted controlled on the "horse", landing space
// available, with a fixed roller that answers faces in order.
func mntFall(tb testing.TB, trigger combat.FallingOffTrigger, attempt d20.SaveAttempt, faces ...int) (combat.ResolveFallingOffResult, error) {
	tb.Helper()
	return mntFallWith(tb, dicetest.NewFixedRoller(faces...), mntMounted(tb, combat.MountControlControlled), trigger, attempt, combat.LandingSpaceUnoccupiedWithin5Feet)
}

func mntFallWith(tb testing.TB, r combat.Roller, status combat.RiderStatus, trigger combat.FallingOffTrigger, attempt d20.SaveAttempt,
	landing combat.LandingSpace, opts ...combat.Option) (combat.ResolveFallingOffResult, error) {
	tb.Helper()
	cmd, err := combat.NewResolveFallingOffCommand(gid("rider"), status, trigger, attempt, landing)
	if err != nil {
		tb.Fatal(err)
	}
	uc, err := combat.NewResolveFallingOff(r, opts...)
	if err != nil {
		tb.Fatal(err)
	}
	return uc.Run(context.Background(), cmd)
}

func modifierAdvantage() modifier.RollCircumstance { return modifier.Advantage{} }
