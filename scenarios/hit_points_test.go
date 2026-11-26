package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/hit-points-and-damage.md section 8 scenario 1 (SRD 5.2.1, "Dropping to 0 Hit Points",
// "Death Saving Throws", "Stabilizing a Character", "Healing", pp. 17-18) - a Critical Hit drops a character to 0
// Hit Points, Death Saving Throws fail and succeed, the character is stabilized, recovers and is healed, through
// the public API only.
// Path: happy.
func TestCriticalHitDeathSavesStabilizeAndHeal(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("aria")
	rec := telemetrytest.NewRecorder()
	opts := []combat.Option{combat.WithTelemetry(rec.Telemetry())}
	st := combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities())

	hit := func(st combat.HitPointStatus, n int, kind combat.HitKind) combat.HitPointStatus {
		cmd, err := combat.NewApplyDamageCommand(id, st, []combat.Damage{combat.MustDamage(n, content.DamageSlashing)}, nil, combat.OtherSource, kind, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewApplyDamage(opts...).Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res.(combat.DamageApplied).Status()
	}
	save := func(st combat.HitPointStatus, face int) combat.HitPointStatus {
		cmd, err := combat.NewMakeDeathSavingThrowCommand(id, st, modifier.Normal, modifier.New(0))
		if err != nil {
			t.Fatal(err)
		}
		uc, err := combat.NewMakeDeathSavingThrow(dicetest.NewFixedRoller(face), opts...)
		if err != nil {
			t.Fatal(err)
		}
		res, err := uc.Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res.(combat.DeathSavingThrowApplied).Status()
	}

	st = hit(st, 20-5, combat.NormalHit) // 5 Hit Points left
	st = hit(st, 8, combat.NormalHit)    // drops to 0 with 3 over: Dying
	if st.LifeState() != (combat.Dying{}) || !st.MakesDeathSavingThrows() {
		t.Fatalf("after the drop = %v, want Dying", st.LifeState())
	}
	st = hit(st, 2, combat.CriticalHit) // a Critical Hit at 0 adds two failures
	if d := st.LifeState().(combat.Dying); d.Failures() != 2 || d.Successes() != 0 {
		t.Fatalf("after the Critical Hit = %v, want two failures", d)
	}
	st = save(st, 14) // a success
	cmd, err := combat.NewStabilizeCreatureCommand(id, st)
	if err != nil {
		t.Fatal(err)
	}
	stab, err := combat.NewStabilizeCreature(opts...).Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	st = stab.(combat.StabilizationApplied).Status()
	if st.LifeState() != (combat.Stable{}) || st.MakesDeathSavingThrows() {
		t.Fatalf("after stabilizing = %v", st.LifeState())
	}
	heal, err := combat.NewRestoreHitPointsCommand(id, st, 7, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewRestoreHitPoints(opts...).Run(ctx, heal)
	if err != nil {
		t.Fatal(err)
	}
	healed := res.(combat.HealingApplied)
	if hp := healed.Status().HitPoints(); hp.Current() != 7 || healed.Status().LifeState() != (combat.Alive{}) || healed.HitPointsRegained() != 7 {
		t.Errorf("after healing = %+v %v", hp, healed.Status().LifeState())
	}
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md section 8 scenario 2 (SRD 5.2.1, "Lose Temporary Hit Points
// First", "Order of Application", pp. 17-18) - Temporary Hit Points, Resistance and Vulnerability together: 28 Fire
// damage with a -5 adjustment against Resistance and Vulnerability to Fire is 22, and Temporary Hit Points take it
// first.
// Path: unhappy (rule interaction).
func TestTemporaryHitPointsResistanceAndVulnerability(t *testing.T) {
	t.Parallel()
	aff, err := combat.NewDamageAffinities([]combat.DamageScope{combat.DamageOfType{Type: content.DamageFire}},
		[]content.DamageType{content.DamageFire}, nil)
	if err != nil {
		t.Fatal(err)
	}
	st := combat.MustHitPointStatus(50, combat.FallsUnconscious, aff)
	gain, err := combat.NewGainTemporaryHitPointsCommand("aria", st, 10, combat.KeepCurrent)
	if err != nil {
		t.Fatal(err)
	}
	g, err := combat.NewGainTemporaryHitPoints().Run(context.Background(), gain)
	if err != nil {
		t.Fatal(err)
	}
	st = g.(combat.TemporaryHitPointsGainApplied).Status()
	cmd, err := combat.NewApplyDamageCommand("aria", st, []combat.Damage{combat.MustDamage(28, content.DamageFire)},
		[]combat.DamageAdjustment{combat.NewFlatAdjustment(-5, nil)}, combat.OtherSource, combat.NormalHit, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewApplyDamage().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	applied := res.(combat.DamageApplied)
	r := applied.Resolutions()[0]
	if r.AfterAdjustments() != 23 || r.AfterResistance() != 11 || r.Dealt() != 22 {
		t.Errorf("resolution = %d, %d, %d, want 23, 11, 22", r.AfterAdjustments(), r.AfterResistance(), r.Dealt())
	}
	if hp := applied.Status().HitPoints(); hp.Temporary() != 0 || hp.Current() != 50-12 {
		t.Errorf("hit points = %+v, want 38 and 0 temporary", hp)
	}
}
