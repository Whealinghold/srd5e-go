package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// grpBStatus builds a restored status with the given life state, 0 current Hit Points and the given maximum.
func grpBStatus(t *testing.T, life combat.LifeState, current, maximum, temporary int) combat.HitPointStatus {
	t.Helper()
	st, err := combat.RestoreHitPointStatus(combat.MustHitPoints(current, maximum, temporary), life, combat.FallsUnconscious, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func grpBDead(t *testing.T, cause combat.DeathCause) combat.HitPointStatus {
	t.Helper()
	d, err := combat.NewDead(cause)
	if err != nil {
		t.Fatal(err)
	}
	return grpBStatus(t, d, 0, 20, 0)
}

// Requirement: doc/specs/hit-points-and-damage.md FR-067, FR-072, FR-073, FR-074 (SRD 5.2.1, "Stabilizing a Character", p. 18) - a Stable creature rolls its 1d4 recovery delay and, when the host says the
// delay is over, returns to 1 Hit Point and Alive; healing it first would have made recovery impossible (NotStable).
// Path: happy.
func TestGrpBStableCreatureRecoversAfterTheRolledDelay(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("brenna")
	rec := telemetrytest.NewRecorder()
	opts := []combat.Option{combat.WithTelemetry(rec.Telemetry())}
	stable := grpBStatus(t, combat.Stable{}, 0, 20, 0)

	roll, err := combat.NewRollStableRecoveryDelay(dicetest.NewFixedRoller(3), opts...)
	if err != nil {
		t.Fatal(err)
	}
	rolled, err := roll.Run(ctx, combat.RollStableRecoveryDelayCommand{})
	if err != nil {
		t.Fatal(err)
	}
	if h := rolled.(combat.StableRecoveryDelayRolled).Delay().Hours(); h != 3 {
		t.Fatalf("delay = %d hours, want the rolled 3", h)
	}

	cmd, err := combat.NewRecoverStableCreatureCommand(id, stable)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewRecoverStableCreature(opts...).Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	st := res.(combat.StableRecoveryApplied).Status()
	if hp := st.HitPoints(); hp.Current() != 1 || hp.Maximum() != 20 || st.LifeState() != (combat.Alive{}) {
		t.Errorf("recovered = %+v %v, want 1 of 20 and Alive", hp, st.LifeState())
	}
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-072, FR-073, FR-074, FR-075 - the delay roll fails through the roller port as a PortError, a nil roller and a cancelled ctx are errors before any roll,
// and recovery of an Alive, Dying or Dead creature is a refusal that leaves the status as it was.
// Path: unhappy (failure, invalid input, state error).
func TestGrpBStableRecoveryUnhappyPaths(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("brenna")

	down := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, errors.New("dice service down") })
	failing, err := combat.NewRollStableRecoveryDelay(down)
	if err != nil {
		t.Fatal(err)
	}
	var pe *srderr.PortError
	if res, err := failing.Run(ctx, combat.RollStableRecoveryDelayCommand{}); res != nil || !errors.As(err, &pe) || srderr.ErrorType(err) != "dice_roller_error" {
		t.Errorf("a failing roller = %v, %v, want no result and a dice_roller_error PortError", res, err)
	}
	if _, err := combat.NewRollStableRecoveryDelay(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil roller err = %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	ok, _ := combat.NewRollStableRecoveryDelay(dicetest.NewFixedRoller(2))
	if _, err := ok.Run(cancelled, combat.RollStableRecoveryDelayCommand{}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled ctx err = %v, want context.Canceled", err)
	}

	for name, c := range map[string]struct {
		status combat.HitPointStatus
		want   combat.HitPointRejection
	}{
		"alive": {combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities()), combat.NotStable},
		"dying": {grpBStatus(t, combat.Dying{}, 0, 20, 0), combat.NotStable},
		"dead":  {grpBDead(t, combat.MassiveDamage), combat.CreatureIsDead},
	} {
		cmd, err := combat.NewRecoverStableCreatureCommand(id, c.status)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewRecoverStableCreature().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		if got, isRej := res.(combat.StableRecoveryRejected); !isRej || got.Rejection != c.want {
			t.Errorf("%s: recovery = %#v, want a rejection %v", name, res, c.want)
		}
	}
	if _, err := combat.NewRecoverStableCreatureCommand(combat.CombatantID(""), grpBStatus(t, combat.Stable{}, 0, 20, 0)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero subject err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-029, FR-033, FR-034, FR-098, FR-075 (SRD 5.2.1, "Temporary Hit Points", p. 18) - Temporary Hit Points gained while Unconscious end with a Long Rest and
// leave the life state alone; ending none is an applied no-op; a Dead creature refuses.
// Path: happy and unhappy (boundary, state error).
func TestGrpBTemporaryHitPointsEndWithTheRest(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("aria")
	stable := grpBStatus(t, combat.Stable{}, 0, 20, 0)

	gain, err := combat.NewGainTemporaryHitPointsCommand(id, stable, 8, combat.AcceptNew)
	if err != nil {
		t.Fatal(err)
	}
	g, err := combat.NewGainTemporaryHitPoints().Run(ctx, gain)
	if err != nil {
		t.Fatal(err)
	}
	withTemp := g.(combat.TemporaryHitPointsGainApplied).Status()
	if withTemp.HitPoints().Temporary() != 8 || withTemp.LifeState() != (combat.Stable{}) {
		t.Fatalf("after the gain = %+v %v, want 8 temporary and still Stable", withTemp.HitPoints(), withTemp.LifeState())
	}

	end, err := combat.NewEndTemporaryHitPointsCommand(id, withTemp)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewEndTemporaryHitPoints().Run(ctx, end)
	if err != nil {
		t.Fatal(err)
	}
	ended := res.(combat.TemporaryHitPointsEndApplied)
	if ended.Status().HitPoints().Temporary() != 0 || len(ended.Events()) != 1 || ended.Events()[0].Kind() != "temporary_hit_points_ended" {
		t.Errorf("after the rest = %+v with events %v, want 0 temporary and one temporary_hit_points_ended", ended.Status().HitPoints(), ended.Events())
	}

	again, _ := combat.NewEndTemporaryHitPointsCommand(id, ended.Status())
	res, err = combat.NewEndTemporaryHitPoints().Run(ctx, again)
	if err != nil {
		t.Fatal(err)
	}
	if noop := res.(combat.TemporaryHitPointsEndApplied); noop.Status() != ended.Status() || len(noop.Events()) != 0 {
		t.Errorf("ending none changed %+v or produced events %v, want an applied no-op", noop.Status().HitPoints(), noop.Events())
	}

	dead, _ := combat.NewEndTemporaryHitPointsCommand(id, grpBDead(t, combat.MonsterDeath))
	res, err = combat.NewEndTemporaryHitPoints().Run(ctx, dead)
	if err != nil {
		t.Fatal(err)
	}
	if r, isRej := res.(combat.TemporaryHitPointsEndRejected); !isRej || r.Rejection != combat.CreatureIsDead {
		t.Errorf("ending on a Dead creature = %#v, want CreatureIsDead", res)
	}
	if _, err := combat.NewEndTemporaryHitPointsCommand(combat.CombatantID(""), ended.Status()); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero subject err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-046, FR-047, FR-093, FR-094, FR-075 (SRD 5.2.1, "Hit Point Maximum of 0", p. 17) - a reduction lowers the maximum and current Hit Points with it, a
// reduction of 0 is an applied no-op, a reduction to 0 kills with HitPointMaximumZero, a negative amount is invalid input and a Dead creature refuses.
// Path: happy and unhappy (boundary, invalid input, state error).
func TestGrpBHitPointMaximumReduction(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("corin")
	full := combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities())
	run := func(st combat.HitPointStatus, amount int) combat.ReduceHitPointMaximumResult {
		t.Helper()
		cmd, err := combat.NewReduceHitPointMaximumCommand(id, st, amount)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewReduceHitPointMaximum().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	reduced := run(full, 8).(combat.HitPointMaximumReductionApplied)
	if hp := reduced.Status().HitPoints(); hp.Maximum() != 12 || hp.Current() != 12 {
		t.Errorf("after -8 = %+v, want 12 of 12", hp)
	}
	if noop := run(full, 0).(combat.HitPointMaximumReductionApplied); noop.Status() != full || len(noop.Events()) != 0 {
		t.Errorf("a reduction of 0 changed the status or produced %v", noop.Events())
	}
	zero := run(reduced.Status(), 12).(combat.HitPointMaximumReductionApplied).Status()
	if d, isDead := zero.LifeState().(combat.Dead); !isDead || d.Cause() != combat.HitPointMaximumZero {
		t.Errorf("a maximum of 0 left %v, want Dead by HitPointMaximumZero", zero.LifeState())
	}
	if r, isRej := run(grpBDead(t, combat.MassiveDamage), 3).(combat.HitPointMaximumReductionRejected); !isRej || r.Rejection != combat.CreatureIsDead {
		t.Errorf("a Dead creature = %#v, want CreatureIsDead", r)
	}
	if _, err := combat.NewReduceHitPointMaximumCommand(id, full, -1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative amount err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-095, FR-075 (SRD 5.2.1, "Exhaustion", p. 181) - the host records the death of a creature at Exhaustion level 6: Alive and Dying creatures become Dead with
// cause Exhaustion and one died event; a creature already Dead refuses.
// Path: happy and unhappy (state error).
func TestGrpBExhaustionDeathIsRecordedOnce(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("hale")
	for name, st := range map[string]combat.HitPointStatus{
		"alive": combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities()),
		"dying": grpBStatus(t, combat.Dying{}, 0, 20, 0),
	} {
		cmd, err := combat.NewRecordExhaustionDeathCommand(id, st)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewRecordExhaustionDeath().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		applied := res.(combat.ExhaustionDeathApplied)
		if d, isDead := applied.Status().LifeState().(combat.Dead); !isDead || d.Cause() != combat.Exhaustion || len(applied.Events()) != 1 || applied.Events()[0].Kind() != "died" {
			t.Errorf("%s: %v with events %v, want Dead by Exhaustion and one died", name, applied.Status().LifeState(), applied.Events())
		}
	}
	cmd, _ := combat.NewRecordExhaustionDeathCommand(id, grpBDead(t, combat.MonsterDeath))
	res, err := combat.NewRecordExhaustionDeath().Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if r, isRej := res.(combat.ExhaustionDeathRejected); !isRej || r.Rejection != combat.CreatureIsDead {
		t.Errorf("a Dead creature = %#v, want CreatureIsDead", res)
	}
}
