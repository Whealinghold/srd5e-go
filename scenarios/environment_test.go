package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/environment-hazards-and-objects.md section 8 scenario, FR-016, FR-019, FR-023, FR-009 (SRD 5.2.1, "Falling [Hazard]",
// p. 182; "Burning [Hazard]", p. 178) - a creature falls 30 feet into water, the host resolves the landing check and passes the Damage and
// HalfDamage to ApplyDamage, then lands Prone through LandAfterFall; a burning creature rolls on the ground and is Prone.
// Path: happy.
func TestFallIntoWaterHalvesDamageAndLandsProne(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("aria")
	fall, err := combat.NewResolveFall(dicetest.NewFixedRoller(4, 4, 3)) // 3d6 = 11
	if err != nil {
		t.Fatal(err)
	}
	res, err := fall.Run(ctx, combat.ResolveFallCommand{Distance: combat.MustFallDistance(30),
		Surface: combat.FallIntoLiquid{Landing: combat.LandingCheckMade{Outcome: d20.Success}}})
	if err != nil {
		t.Fatal(err)
	}
	rolled := res.(combat.FallDamageRolled)
	cmd, err := combat.NewApplyDamageCommand(id, combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities()),
		[]combat.Damage{rolled.Damage()}, rolled.Adjustments(), combat.OtherSource, combat.NormalHit, nil)
	if err != nil {
		t.Fatal(err)
	}
	hit, err := combat.NewApplyDamage().Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	status := hit.(combat.DamageApplied).Status()
	if got := status.HitPoints().Current(); got != 15 { // 11 halved, rounded down: 5
		t.Fatalf("Hit Points after the fall = %d, want 15", got)
	}
	landed, err := combat.NewLandAfterFall().Run(ctx, combat.LandAfterFallCommand{Target: id, DamageTaken: 20 - status.HitPoints().Current()})
	if err != nil {
		t.Fatal(err)
	}
	prone, ok := landed.(combat.FallLandedProne)
	if !ok || len(prone.Conditions().Effective()) != 1 || prone.Conditions().Effective()[0] != content.ConditionProne {
		t.Fatalf("landing = %#v, want Prone", landed)
	}

	burning, err := combat.NewExtinguishBurning().Run(ctx, combat.ExtinguishBurningCommand{State: combat.BurningStateBurning,
		Method: combat.ExtinguishMethodRollOnTheGround, Target: id})
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := burning.(combat.BurningExtinguished); !ok || e.State() != combat.BurningStateNotBurning || e.Events()[len(e.Events())-1].Kind() != "burning_ended" {
		t.Errorf("extinguish = %#v, want the fire out with burning_ended last", burning)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md section 8 scenario, FR-081, FR-082, FR-075 (SRD 5.2.1, "Damage Threshold", p. 180) - an
// object with a Damage Threshold of 10 survives a 9 and is destroyed by a 40, with the events in order.
// Path: happy.
func TestObjectWithDamageThresholdSurvivesNineAndIsDestroyedByForty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	threshold, err := combat.NewDamageThreshold(10)
	if err != nil {
		t.Fatal(err)
	}
	door, err := combat.NewObjectStatus(content.SizeLarge, 15, combat.MustHitPoints(27, 27, 0), threshold, combat.ObjectAffinities{}.Standard(), nil)
	if err != nil {
		t.Fatal(err)
	}
	blow := func(n int) combat.ObjectHarm {
		i, err := combat.NewObjectDamageInstance([]combat.Damage{combat.MustDamage(n, content.DamageBludgeoning)}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return i
	}
	uc := combat.NewApplyDamageToObject()
	res, err := uc.Run(ctx, combat.ApplyDamageToObjectCommand{Status: door, Harm: blow(9)})
	if err != nil {
		t.Fatal(err)
	}
	glanced := res.(combat.ObjectDamageApplied)
	if glanced.Status() != door || glanced.Threshold() != combat.ThresholdNotMet || glanced.Events()[0].Kind() != "object_damage_deflected" {
		t.Fatalf("a 9 = %#v, want the door unharmed", glanced)
	}
	res, err = uc.Run(ctx, combat.ApplyDamageToObjectCommand{Status: door, Harm: blow(40)})
	if err != nil {
		t.Fatal(err)
	}
	smashed := res.(combat.ObjectDamageApplied)
	if !smashed.Status().IsDestroyed() || len(smashed.Events()) != 2 || smashed.Events()[0].Kind() != "object_damaged" || smashed.Events()[1].Kind() != "object_destroyed" {
		t.Fatalf("a 40 = %#v, want the door destroyed", smashed)
	}
	again, err := uc.Run(ctx, combat.ApplyDamageToObjectCommand{Status: smashed.Status(), Harm: blow(1)})
	if err != nil || again != (combat.ObjectDamageRejected{Rejection: combat.ObjectDamageRejectionAlreadyDestroyed}) {
		t.Errorf("a blow to a destroyed door = %#v, %v, want the rejection", again, err)
	}
}
