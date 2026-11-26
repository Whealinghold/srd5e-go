package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func gearID(s string) combat.DeployedGearID { return combat.MustDeployedGearID(s) }
func hid(s string) combat.HoldID            { return combat.MustHoldID(s) }

// gearStamp is the turn of round r of the creature who.
func gearStamp(r int, who string) combat.TurnStamp { return combat.MustTurnStamp(r, gid(who)) }

// gearsApplied unwraps an applied transition of DeployedGears.
func gearsApplied(tb testing.TB, tr combat.DeployedGearsTransition, err error) combat.DeployedGearsApplied {
	tb.Helper()
	if err != nil {
		tb.Fatal(err)
	}
	a, ok := tr.(combat.DeployedGearsApplied)
	if !ok {
		tb.Fatalf("transition = %#v, want applied", tr)
	}
	return a
}

// gearDeploy sets out a gear of kind with the id for "owner", within reach, on level ground.
func gearDeploy(tb testing.TB, set combat.DeployedGears, id string, kind combat.DeployedGearKind) combat.DeployedGears {
	tb.Helper()
	tr, err := set.Deploy(gearID(id), kind, gid("owner"), 0, combat.GroundLevelLevel)
	return gearsApplied(tb, tr, err).Gears()
}

// gearOne is a set holding one gear with the id "g" of kind.
func gearOne(tb testing.TB, kind combat.DeployedGearKind) combat.DeployedGears {
	tb.Helper()
	return gearDeploy(tb, combat.DeployedGears{}, "g", kind)
}

// gearBurning is a set holding Oil "g" lit at (round, who): it burns through (round+2, who).
func gearBurning(tb testing.TB, round int, who string) combat.DeployedGears {
	tb.Helper()
	tr, err := gearOne(tb, combat.DeployedGearKindPouredOil).Light(gearID("g"), gearStamp(round, who))
	return gearsApplied(tb, tr, err).Gears()
}

// gearSprung is a set holding a Hunting Trap "g" that "walker" sprang.
func gearSprung(tb testing.TB) combat.DeployedGears {
	tb.Helper()
	tr, err := gearOne(tb, combat.DeployedGearKindHuntingTrap).Spring(gearID("g"), gid("walker"))
	return gearsApplied(tb, tr, err).Gears()
}

// gearOf returns the gear "g" of the set.
func gearOf(tb testing.TB, set combat.DeployedGears) combat.DeployedGear {
	tb.Helper()
	g, ok := set.Find(gearID("g"))
	if !ok {
		tb.Fatal("no gear g")
	}
	return g
}

// holdIn is the input of a valid hold of kind with the id on "victim".
func holdIn(id string, kind combat.HoldKind) combat.GearHoldInput {
	in := combat.GearHoldInput{ID: hid(id), Kind: kind, Target: gid("victim"), Restraint: combat.HoldRestraintRestrains}
	if kind == combat.HoldKindHuntingTrap {
		in.Restraint, in.GearID, in.MovementLimitFeet = combat.HoldRestraintDoesNotRestrain, gearID("trap"), 3
	}
	return in
}

func gearHold(tb testing.TB, in combat.GearHoldInput) combat.GearHold {
	tb.Helper()
	h, err := combat.NewGearHold(in)
	if err != nil {
		tb.Fatal(err)
	}
	return h
}

func gearHolds(tb testing.TB, holds ...combat.GearHold) combat.GearHolds {
	tb.Helper()
	s, err := combat.NewGearHolds(holds)
	if err != nil {
		tb.Fatal(err)
	}
	return s
}

// holdsApplied unwraps an applied transition of GearHolds.
func holdsApplied(tb testing.TB, tr combat.GearHoldsTransition, err error) combat.GearHoldsApplied {
	tb.Helper()
	if err != nil {
		tb.Fatal(err)
	}
	a, ok := tr.(combat.GearHoldsApplied)
	if !ok {
		tb.Fatalf("transition = %#v, want applied", tr)
	}
	return a
}

// gearNames lists the gear event kinds in order.
func gearKinds(events []combat.GearEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// gearConds is the effective conditions of a target that is held down: Restrained.
func gearHeldDown(tb testing.TB) combat.ConditionSet {
	tb.Helper()
	return condSet(tb, condConds(combat.Restrained{}))
}

// gearEntryUse builds ResolveGearEntry over roller.
func gearEntryUse(tb testing.TB, roller combat.Roller, opts ...combat.Option) combat.ResolveGearEntry {
	tb.Helper()
	u, err := combat.NewResolveGearEntry(roller, opts...)
	if err != nil {
		tb.Fatal(err)
	}
	return u
}

// gearEntryIn is the input of an entry of "walker" into the gear "g" of set, at (1, walker): a
// plain save with a +0 modifier.
func gearEntryIn(set combat.DeployedGears) combat.ResolveGearEntryInput {
	return combat.ResolveGearEntryInput{Gears: set, GearID: gearID("g"), Creature: gid("walker"), Trigger: combat.AreaTriggerEnters,
		Stamp: gearStamp(1, "walker"), HoldID: hid("h"), Attempt: rollWith(0)}
}

func gearEntry(tb testing.TB, in combat.ResolveGearEntryInput, roller combat.Roller, opts ...combat.Option) (combat.ResolveGearEntryResult, error) {
	tb.Helper()
	cmd, err := combat.NewResolveGearEntryCommand(in)
	if err != nil {
		tb.Fatal(err)
	}
	return gearEntryUse(tb, roller, opts...).Run(context.Background(), cmd)
}

// netIn is the input of a Net thrown by "thrower" at "victim", 10 feet away, seen, DC 13.
func netIn() combat.ThrowNetInput {
	return combat.ThrowNetInput{HoldID: hid("net"), Thrower: gid("thrower"), Target: gid("victim"), TargetSize: content.SizeMedium,
		DistanceFeet: 10, Sight: combat.ThrowerSightSeesTarget, SaveDC: 13, Attempt: rollWith(0)}
}

func netRun(tb testing.TB, in combat.ThrowNetInput, roller combat.Roller, opts ...combat.Option) (combat.ThrowNetResult, error) {
	tb.Helper()
	cmd, err := combat.NewThrowNetCommand(in)
	if err != nil {
		tb.Fatal(err)
	}
	u, err := combat.NewThrowNet(roller, opts...)
	if err != nil {
		tb.Fatal(err)
	}
	return u.Run(context.Background(), cmd)
}

// bindIn is the input of "binder" tying "victim", Restrained, unwilling, 5 feet away, with a Chain.
func bindIn(tb testing.TB) combat.BindCreatureInput {
	tb.Helper()
	return combat.BindCreatureInput{HoldID: hid("chain"), Kind: combat.BindingKindChain, Binder: gid("binder"), Target: gid("victim"),
		TargetSize: content.SizeMedium, TargetConditions: gearHeldDown(tb), Willingness: combat.WillingnessUnwilling, DistanceFeet: 5,
		Restraint: combat.HoldRestraintRestrains, Attempt: rollWith(0)}
}

func bindRun(tb testing.TB, in combat.BindCreatureInput, roller combat.Roller, opts ...combat.Option) (combat.BindCreatureResult, error) {
	tb.Helper()
	cmd, err := combat.NewBindCreatureCommand(in)
	if err != nil {
		tb.Fatal(err)
	}
	u, err := combat.NewBindCreature(roller, opts...)
	if err != nil {
		tb.Fatal(err)
	}
	return u.Run(context.Background(), cmd)
}

// releaseIn is the input of the held "victim" trying to escape the hold "h" in holds.
func releaseIn(holds combat.GearHolds) combat.ReleaseFromHoldInput {
	return combat.ReleaseFromHoldInput{Holds: holds, HoldID: hid("h"), Method: combat.HoldReleaseMethodEscape, Actor: combat.ActorIsHeld{},
		Actions: combat.ConditionActionAvailabilityAvailable, Attempt: rollWith(0)}
}

func releaseRun(tb testing.TB, in combat.ReleaseFromHoldInput, roller combat.Roller, opts ...combat.Option) (combat.ReleaseFromHoldResult, error) {
	tb.Helper()
	cmd, err := combat.NewReleaseFromHoldCommand(in)
	if err != nil {
		tb.Fatal(err)
	}
	u, err := combat.NewReleaseFromHold(roller, opts...)
	if err != nil {
		tb.Fatal(err)
	}
	return u.Run(context.Background(), cmd)
}

// withAdvantage makes the net save roll two d20s, so a roller answering one expression is caught.
func withAdvantage(in combat.ThrowNetInput) combat.ThrowNetInput {
	in.Attempt = rollWith(0, modifier.Advantage{})
	return in
}

func withBindAdvantage(in combat.BindCreatureInput) combat.BindCreatureInput {
	in.Attempt = rollWith(0, modifier.Advantage{})
	return in
}
