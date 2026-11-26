package combat_test

import (
	"context"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/hit-points-and-damage.md Limits (plan P10) - one ApplyDamage takes at most 64 Damage instances (damages and unpreventable together) and at most 64 adjustments; one more of either is a LimitExceededError and the status is unchanged.
// Path: unhappy (boundary).
func TestApplyDamageCaps(t *testing.T) {
	t.Parallel()
	st := pcStatus(t, 20, 20, 0)
	t.Run("damage instances", func(t *testing.T) {
		t.Parallel()
		runCap(t, "damage instances", 64, func(n int) error {
			ds := make([]combat.Damage, n)
			for i := range ds {
				ds[i] = fire(0)
			}
			_, err := st.ApplyDamage(subject(), ds[:n/2], nil, combat.OtherSource, combat.NormalHit, ds[n/2:])
			return err
		})
	})
	t.Run("adjustments", func(t *testing.T) {
		t.Parallel()
		runCap(t, "damage adjustments", 64, func(n int) error {
			adj := make([]combat.DamageAdjustment, n)
			for i := range adj {
				adj[i] = combat.FlatAdjustment{}
			}
			_, err := st.ApplyDamage(subject(), []combat.Damage{fire(1)}, adj, combat.OtherSource, combat.NormalHit, nil)
			return err
		})
	})
	t.Run("the command constructor reports the same cap", func(t *testing.T) {
		t.Parallel()
		_, err := combat.NewApplyDamageCommand(subject(), st, make([]combat.Damage, 65), nil, combat.OtherSource, combat.NormalHit, nil)
		wantLimit(t, err, "damage instances", 64)
	})
}

// Requirement: doc/specs/hit-points-and-damage.md FR-102, plan Arithmetic overflow - two Damage instances of math.MaxInt sum to math.MaxInt, not a wrapped negative: the creature dies of Massive Damage and DamageTaken reports math.MaxInt.
// Path: unhappy (boundary).
func TestApplyDamageTwoMaxIntInstancesSaturate(t *testing.T) {
	t.Parallel()
	tr, err := pcStatus(t, 20, 20, 0).ApplyDamage(subject(), []combat.Damage{fire(math.MaxInt), fire(math.MaxInt)}, nil, combat.OtherSource, combat.NormalHit, nil)
	ev := applied(t, tr, err).Events()
	taken, ok := firstEvent[combat.DamageTaken](ev)
	if !ok || taken.Total() != math.MaxInt || taken.HitPointsLost() != math.MaxInt {
		t.Fatalf("DamageTaken = %v (found %v), want total and lost %d", taken, ok, math.MaxInt)
	}
	if _, died := firstEvent[combat.Died](ev); !died {
		t.Error("two MaxInt instances did not kill")
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-102, plan Arithmetic overflow - with math.MaxInt Temporary Hit Points, two MaxInt instances are absorbed whole: the sum saturates, nothing is lost and the Temporary Hit Points end.
// Path: unhappy (boundary).
func TestApplyDamageTemporarySumSaturates(t *testing.T) {
	t.Parallel()
	tr, err := pcStatus(t, 20, 20, math.MaxInt).ApplyDamage(subject(), []combat.Damage{fire(math.MaxInt)}, nil, combat.OtherSource, combat.NormalHit, []combat.Damage{fire(math.MaxInt)})
	a := applied(t, tr, err)
	taken, _ := firstEvent[combat.DamageTaken](a.Events())
	if taken.AbsorbedByTemporary() != math.MaxInt || taken.HitPointsLost() != 0 || taken.Total() != math.MaxInt {
		t.Errorf("DamageTaken = absorbed %d lost %d total %d", taken.AbsorbedByTemporary(), taken.HitPointsLost(), taken.Total())
	}
	if a.Status().HitPoints() != combat.MustHitPoints(20, 20, 0) {
		t.Errorf("hit points = %v, want 20/20 with no temporary", a.Status().HitPoints())
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-054, plan Arithmetic overflow - a modifier of math.MaxInt on a Death Saving Throw saturates the total, so a natural 2 succeeds; math.MinInt saturates low, so a natural 19 fails.
// Path: per case.
func TestMakeDeathSavingThrowSaturatesTheTotal(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement string
		face, mod         int
		want              combat.DeathSavingThrowOutcome
	}{
		{"MaxInt modifier on a 2", "overflow high", 2, math.MaxInt, combat.DeathSavingThrowSuccess},
		{"MinInt modifier on a 19", "overflow low", 19, math.MinInt, combat.DeathSavingThrowFailure},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := combat.NewMakeDeathSavingThrowCommand(subject(), dyingStatus(t, 20, 0, 0), modifier.Normal, modifier.New(c.mod))
			if err != nil {
				t.Fatal(err)
			}
			uc, err := combat.NewMakeDeathSavingThrow(dicetest.NewFixedRoller(c.face))
			if err != nil {
				t.Fatal(err)
			}
			res, err := uc.Run(context.Background(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			if got := res.(combat.DeathSavingThrowApplied).Outcome(); got != c.want {
				t.Errorf("outcome = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/hit-points-and-damage.md Limits (plan P10) - the Encounter method, the ApplyDamageToCombatant command and the DealDamageToCombatant command each take at most 64 Damage instances and at most 64 adjustments, with the same LimitExceededError as HitPointStatus.ApplyDamage.
// Path: unhappy (boundary).
func TestEncounterDamagePathsCaps(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	paths := map[string]func(ds []combat.Damage, adj []combat.DamageAdjustment) error{
		"Encounter.ApplyDamage": func(ds []combat.Damage, adj []combat.DamageAdjustment) error {
			_, err := e.ApplyDamage(cid("a"), ds, adj, combat.OtherSource, combat.NormalHit)
			return err
		},
		"ApplyDamageToCombatant command": func(ds []combat.Damage, adj []combat.DamageAdjustment) error {
			_, err := combat.NewApplyDamageToCombatantCommand(e, cid("a"), ds, adj, combat.OtherSource, combat.NormalHit)
			return err
		},
		"DealDamageToCombatant command": func(ds []combat.Damage, adj []combat.DamageAdjustment) error {
			_, err := combat.NewDealDamageToCombatantCommand(e, cid("a"), ds, adj, combat.OtherSource, combat.NormalHit)
			return err
		},
	}
	for name, build := range paths {
		t.Run(name+" instances", func(t *testing.T) {
			t.Parallel()
			runCap(t, "damage instances", 64, func(n int) error {
				ds := make([]combat.Damage, n)
				for i := range ds {
					ds[i] = fire(0)
				}
				return build(ds, nil)
			})
		})
		t.Run(name+" adjustments", func(t *testing.T) {
			t.Parallel()
			runCap(t, "damage adjustments", 64, func(n int) error {
				adj := make([]combat.DamageAdjustment, n)
				for i := range adj {
					adj[i] = combat.FlatAdjustment{}
				}
				return build([]combat.Damage{fire(1)}, adj)
			})
		})
	}
}
