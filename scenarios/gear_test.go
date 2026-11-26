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

// Requirement: doc/specs/deployed-and-restraining-gear.md section 8 (scenario), FR-005, FR-013, FR-023, FR-028, FR-041, FR-056 (SRD 5.2.1, "Hunting
// Trap", pp. 97-98) - a Hunting Trap is set, a creature steps on it and fails the Dexterity saving throw, is caught (Speed 0, the 1d4 damage and a hold
// the host begins), fails the escape check (1 Piercing damage) and then passes it; the trap stays sprung until reset, through the public API only.
// Path: happy.
func TestHuntingTrapCatchesACreatureThatThenEscapes(t *testing.T) {
	t.Parallel()
	ctx, owner, walker := context.Background(), combat.MustCombatantID("owner"), combat.MustCombatantID("walker")
	trapID := combat.MustDeployedGearID("trap")
	placed, err := combat.NewDeployGear().Run(ctx, mustDeploy(t, combat.DeployedGears{}, trapID, combat.DeployedGearKindHuntingTrap, owner, 30))
	if err != nil {
		t.Fatal(err)
	}
	gears := placed.(combat.DeployGearPlaced).Gears()

	cmd, err := combat.NewResolveGearEntryCommand(combat.ResolveGearEntryInput{Gears: gears, GearID: trapID, Creature: walker, Trigger: combat.AreaTriggerEnters,
		Stamp: combat.MustTurnStamp(1, walker), HoldID: combat.MustHoldID("grip"), Attempt: d20.RollAttempt{}})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := combat.NewResolveGearEntry(dicetest.NewFixedRoller(5, 3)) // a 5 fails the DC 13 save, then 1d4 = 3
	if err != nil {
		t.Fatal(err)
	}
	res, err := entry.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	failed, ok := res.(combat.ResolveGearEntryFailed)
	if !ok {
		t.Fatalf("entry = %#v, want failed", res)
	}
	effects := failed.Outcome().(combat.GearEntryFailed).Effects()
	if len(effects) != 3 || effects[0] != (combat.GearDamageEffect{Damage: combat.MustDamage(3, content.DamagePiercing)}) {
		t.Fatalf("effects = %#v, want 3 Piercing, Speed 0 and a caught hold", effects)
	}
	began, err := combat.GearHolds{}.Begin(effects[2].(combat.GearCaughtEffect).Hold)
	if err != nil {
		t.Fatal(err)
	}
	holds := began.(combat.GearHoldsApplied).Holds()
	if holds.RestrainsTarget(walker) {
		t.Fatal("a Hunting Trap does not restrain")
	}

	escape := func(face int) combat.ReleaseFromHoldResult {
		cmd, err := combat.NewReleaseFromHoldCommand(combat.ReleaseFromHoldInput{Holds: holds, HoldID: combat.MustHoldID("grip"), Method: combat.HoldReleaseMethodEscape,
			Actor: combat.ActorIsHeld{}, Actions: combat.ConditionActionAvailabilityAvailable, Attempt: d20.RollAttempt{}})
		if err != nil {
			t.Fatal(err)
		}
		use, err := combat.NewReleaseFromHold(dicetest.NewFixedRoller(face))
		if err != nil {
			t.Fatal(err)
		}
		out, err := use.Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	held, ok := escape(12).(combat.ReleaseFromHoldHeld)
	if dmg, has := held.Damage(); !ok || !has || dmg != combat.MustDamage(1, content.DamagePiercing) || !held.ActionSpent() {
		t.Fatalf("first escape = %#v, want held with 1 Piercing damage", held)
	}
	freed, ok := escape(13).(combat.ReleaseFromHoldReleased)
	if !ok || freed.Holds().Len() != 0 {
		t.Fatalf("second escape = %#v, want released", freed)
	}

	reset := func(h combat.GearHolds) combat.MaintainDeployedGearResult {
		c, err := combat.NewMaintainDeployedGearCommand(failed.Gears(), h, combat.ResetHuntingTrap{GearID: trapID})
		if err != nil {
			t.Fatal(err)
		}
		out, err := combat.NewMaintainDeployedGear().Run(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	if r, ok := reset(holds).(combat.MaintainDeployedGearRejected); !ok || r.Reason() != combat.GearRejectionTrapOccupied {
		t.Errorf("reset while the creature is caught = %#v, want trapOccupied", r)
	}
	if _, ok := reset(freed.Holds()).(combat.MaintainDeployedGearApplied); !ok {
		t.Error("reset after the creature is free must succeed")
	}
}

func mustDeploy(t *testing.T, set combat.DeployedGears, id combat.DeployedGearID, kind combat.DeployedGearKind, who combat.CombatantID, distance int) combat.DeployGearCommand {
	t.Helper()
	cmd, err := combat.NewDeployGearCommand(set, id, kind, who, distance, combat.GroundLevelLevel)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

// Requirement: doc/specs/deployed-and-restraining-gear.md section 8 (scenario), FR-017, FR-018 (SRD 5.2.1, "Caltrops", p. 96) - a creature enters Caltrops
// twice in one turn and saves once; on the next round it saves again.
// Path: unhappy (rule interaction).
func TestCaltropsAreSetOffOncePerTurn(t *testing.T) {
	t.Parallel()
	ctx, walker, id := context.Background(), combat.MustCombatantID("walker"), combat.MustDeployedGearID("caltrops")
	placed, err := combat.NewDeployGear().Run(ctx, mustDeploy(t, combat.DeployedGears{}, id, combat.DeployedGearKindCaltrops, combat.MustCombatantID("owner"), 5))
	if err != nil {
		t.Fatal(err)
	}
	gears, ledger := placed.(combat.DeployGearPlaced).Gears(), combat.EntryLedger{}
	enter := func(round int, face int) combat.ResolveGearEntryResult {
		cmd, err := combat.NewResolveGearEntryCommand(combat.ResolveGearEntryInput{Gears: gears, Ledger: ledger, GearID: id, Creature: walker,
			Trigger: combat.AreaTriggerEnters, Stamp: combat.MustTurnStamp(round, walker), HoldID: combat.MustHoldID("unused"), Attempt: d20.RollAttempt{}})
		if err != nil {
			t.Fatal(err)
		}
		use, err := combat.NewResolveGearEntry(dicetest.NewFixedRoller(face))
		if err != nil {
			t.Fatal(err)
		}
		out, err := use.Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	first := enter(1, 14)
	failed, ok := first.(combat.ResolveGearEntryFailed)
	if !ok {
		t.Fatalf("first entry = %#v, want failed", first)
	}
	ledger = failed.Ledger()
	if r, ok := enter(1, 20).(combat.ResolveGearEntryRejected); !ok || r.Reason() != combat.GearRejectionAlreadyTriggeredThisTurn {
		t.Errorf("second entry on the turn = %#v, want alreadyTriggeredThisTurn", r)
	}
	if _, ok := enter(2, 15).(combat.ResolveGearEntrySaved); !ok {
		t.Error("the next round the creature saves again, and 15 meets the DC")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md section 8 (scenario), FR-031, FR-046, FR-047 (SRD 5.2.1, "Manacles", p. 98) - Manacles bind a
// Grappled, unwilling Medium creature (a DC 13 Dexterity (Sleight of Hand) check); the hold gives Disadvantage on attack rolls, merged with a condition's
// Advantage through modifier.Resolve, and a key frees the creature without a roll.
// Path: happy; unhappy (rule interaction).
func TestManaclesBindAGrappledCreature(t *testing.T) {
	t.Parallel()
	ctx, victim := context.Background(), combat.MustCombatantID("victim")
	conditions := combat.MustConditionSet([]combat.Condition{combat.Grappled{Grappler: combat.MustCombatantID("brute")}}, nil)
	cmd, err := combat.NewBindCreatureCommand(combat.BindCreatureInput{HoldID: combat.MustHoldID("cuffs"), Kind: combat.BindingKindManacles, Binder: combat.MustCombatantID("jailer"),
		Target: victim, TargetSize: content.SizeMedium, TargetConditions: conditions, Willingness: combat.WillingnessUnwilling, DistanceFeet: 5,
		Restraint: combat.HoldRestraintDoesNotRestrain, Attempt: d20.RollAttempt{}})
	if err != nil {
		t.Fatal(err)
	}
	use, err := combat.NewBindCreature(dicetest.NewFixedRoller(13))
	if err != nil {
		t.Fatal(err)
	}
	res, err := use.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	bound, ok := res.(combat.BindCreatureBound)
	if !ok || bound.RestraintBegins() {
		t.Fatalf("bind = %#v, want bound with loose manacles that do not restrain", res)
	}
	circumstances := bound.Holds().AttackRollCircumstances(victim)
	if modifier.Resolve(circumstances...) != modifier.WithDisadvantage || modifier.Resolve(append(circumstances, modifier.Advantage{})...) != modifier.Normal {
		t.Errorf("circumstances = %v, want Disadvantage, cancelled by Advantage", circumstances)
	}
	keyCmd, err := combat.NewReleaseFromHoldCommand(combat.ReleaseFromHoldInput{Holds: bound.Holds(), HoldID: combat.MustHoldID("cuffs"), Method: combat.HoldReleaseMethodKey,
		Actor: combat.ActorIsHelper{}, Actions: combat.ConditionActionAvailabilityAvailable, Attempt: d20.RollAttempt{}})
	if err != nil {
		t.Fatal(err)
	}
	dice := dicetest.NewFixedRoller()
	release, err := combat.NewReleaseFromHold(dice)
	if err != nil {
		t.Fatal(err)
	}
	out, err := release.Run(ctx, keyCmd)
	if r, ok := out.(combat.ReleaseFromHoldReleased); err != nil || !ok || r.ActionSpent() || len(dice.Requests()) != 0 || len(r.Holds().AttackRollCircumstances(victim)) != 0 {
		t.Errorf("key = %#v, %v; want released with no roll and no Disadvantage", out, err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md section 8 (scenario), FR-037, FR-042, FR-043, FR-044 (SRD 5.2.1, "Net", p. 98) - a Net is thrown at
// a Large creature (a Dexterity save at DC 8 + Dex + PB, failed), it restrains the target, and damage dealt through ApplyDamageToObject to the Net's status
// destroys it, which frees the target.
// Path: happy.
func TestNetHoldsALargeCreatureUntilItIsDestroyed(t *testing.T) {
	t.Parallel()
	ctx, victim := context.Background(), combat.MustCombatantID("victim")
	dc := combat.NetRules{}.SaveDC(3, 2)
	cmd, err := combat.NewThrowNetCommand(combat.ThrowNetInput{HoldID: combat.MustHoldID("net"), Thrower: combat.MustCombatantID("thrower"), Target: victim,
		TargetSize: content.SizeLarge, DistanceFeet: 15, Sight: combat.ThrowerSightSeesTarget, SaveDC: dc, Attempt: d20.RollAttempt{}})
	if err != nil {
		t.Fatal(err)
	}
	use, err := combat.NewThrowNet(dicetest.NewFixedRoller(dcFace(dc) - 1))
	if err != nil {
		t.Fatal(err)
	}
	res, err := use.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	held, ok := res.(combat.ThrowNetHeld)
	if !ok || dc != 13 || !held.RestraintBegins() || !held.Holds().RestrainsTarget(victim) {
		t.Fatalf("throw = %#v (DC %d), want a restraining Net", res, dc)
	}
	hold, _ := held.Holds().HoldOf("net")
	blow, err := combat.NewObjectDamageInstance([]combat.Damage{combat.MustDamage(5, content.DamageSlashing)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	hit, err := combat.NewApplyDamageToObject().Run(ctx, combat.ApplyDamageToObjectCommand{Status: hold.Status(), Harm: blow})
	if err != nil {
		t.Fatal(err)
	}
	status := hit.(combat.ObjectDamageApplied).Status()
	tr, err := held.Holds().RecordNetDamage("net", status)
	if err != nil {
		t.Fatal(err)
	}
	freed := tr.(combat.GearHoldsApplied)
	if !status.IsDestroyed() || !freed.RestraintEnds() || freed.Holds().Len() != 0 || freed.Events()[0].(combat.GearHoldEnded).Cause() != combat.HoldEndCauseDestroyed {
		t.Errorf("destroyed Net = %#v, want the hold ended as destroyed and the restraint over", freed)
	}
}

// dcFace returns dc as a die face for a fixed roller, never above 20.
func dcFace(dc d20.DifficultyClass) int { return min(int(dc), 20) }
