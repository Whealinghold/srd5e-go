package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-108, table 4.10b - damage_resolved increments once per
// instance with the lowercase Damage Type and the effect chosen by the first matching rule; hit_points_lost
// records one sample per applied command (the Hit Points that reached Hit Points, 0 when absorbed or nothing was
// dealt).
// Path: per case.
func TestApplyDamageSignals(t *testing.T) {
	t.Parallel()
	table := "table 4.10b"
	cases := []struct {
		name, path string
		aff        combat.DamageAffinities
		damages    []combat.Damage
		adj        []combat.DamageAdjustment
		effects    [][2]string
		lost       float64
	}{
		{"incoming 0 under Immunity is unchanged", "unhappy (empty and minimal)", affinities(t, nil, nil, damageTypes(content.DamageFire)), []combat.Damage{fire(0)}, nil, [][2]string{{"fire", "unchanged"}}, 0},
		{"Immunity", "happy", affinities(t, nil, nil, damageTypes(content.DamageFire)), []combat.Damage{fire(9)}, nil, [][2]string{{"fire", "immune"}}, 0},
		{"1 Fire under Resistance reduces to zero", "unhappy (boundary)", affinities(t, fireScope(), nil, nil), []combat.Damage{fire(1)}, nil, [][2]string{{"fire", "reduced_to_zero"}}, 0},
		{"Resistance only", "happy", affinities(t, fireScope(), nil, nil), []combat.Damage{fire(10)}, nil, [][2]string{{"fire", "resisted"}}, 5},
		{"Vulnerability only", "happy", affinities(t, nil, damageTypes(content.DamageFire), nil), []combat.Damage{fire(10)}, nil, [][2]string{{"fire", "vulnerable"}}, 20},
		{"both", "unhappy (rule interaction)", affinities(t, fireScope(), damageTypes(content.DamageFire), nil), []combat.Damage{fire(10)}, nil, [][2]string{{"fire", "resisted_and_vulnerable"}}, 10},
		{"flat adjustment only is unchanged", "unhappy (rule interaction)", combat.NoAffinities(), []combat.Damage{fire(10)}, []combat.DamageAdjustment{combat.NewFlatAdjustment(2, nil)}, [][2]string{{"fire", "unchanged"}}, 12},
		{"penalty that floors at zero", "unhappy (boundary)", combat.NoAffinities(), []combat.Damage{fire(2)}, []combat.DamageAdjustment{combat.NewFlatAdjustment(-9, nil)}, [][2]string{{"fire", "reduced_to_zero"}}, 0},
		{"two instances, one increment each", "unhappy (rule interaction)", affinities(t, fireScope(), nil, nil),
			[]combat.Damage{fire(10), combat.MustDamage(4, content.DamageSlashing)}, nil, [][2]string{{"fire", "resisted"}, {"slashing", "unchanged"}}, 9},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			cmd, err := combat.NewApplyDamageCommand(subject(), status(t, 50, 50, 0, combat.Alive{}, combat.FallsUnconscious, c.aff), c.damages, c.adj, combat.OtherSource, combat.NormalHit, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := combat.NewApplyDamage(opts...).Run(context.Background(), cmd); err != nil {
				t.Fatal(err)
			}
			var got [][2]string
			for _, m := range measured(rec, "srd5e.combat.damage_resolved") {
				got = append(got, [2]string{m["srd5e.combat.damage_type"].(string), m["srd5e.combat.damage_effect"].(string)})
			}
			if !slices.Equal(got, c.effects) {
				t.Errorf("damage_resolved = %v, want %v [%s]", got, c.effects, table)
			}
			lost := rec.Meter.MeasurementsNamed("srd5e.combat.hit_points_lost")
			if len(lost) != 1 || lost[0].Value != c.lost || len(lost[0].Attrs) != 0 {
				t.Errorf("hit_points_lost = %+v, want one sample of %v without attributes", lost, c.lost)
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-108 - a rejection records no damage_resolved and no
// hit_points_lost sample; an absorbed command records 0 lost.
// Path: unhappy (rule interaction).
func TestApplyDamageRejectionRecordsNoDamageSignals(t *testing.T) {
	t.Parallel()
	rec, opts := recording()
	cmd, _ := combat.NewApplyDamageCommand(subject(), deadStatus(t, combat.MassiveDamage), []combat.Damage{fire(3)}, nil, combat.OtherSource, combat.NormalHit, nil)
	res, err := combat.NewApplyDamage(opts...).Run(context.Background(), cmd)
	if err != nil || res != (combat.DamageRejected{Rejection: combat.CreatureIsDead}) {
		t.Fatalf("= %v, %v", res, err)
	}
	if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.damage_resolved")) + len(rec.Meter.MeasurementsNamed("srd5e.combat.hit_points_lost")); n != 0 {
		t.Errorf("damage signals = %d, want 0", n)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-085, FR-094, FR-076 - the command rejects what the operation
// rejects with ErrInvalidInput, and copies its slices so a later change by the caller does not alter it.
// Path: unhappy (invalid input).
func TestNewApplyDamageCommand(t *testing.T) {
	t.Parallel()
	pc := pcStatus(t, 10, 20, 0)
	if _, err := combat.NewApplyDamageCommand(subject(), pc, nil, nil, combat.OtherSource, combat.NormalHit, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("no damage error = %v", err)
	}
	if _, err := combat.NewApplyDamageCommand("", pc, []combat.Damage{fire(1)}, nil, combat.OtherSource, combat.NormalHit, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("no subject error = %v", err)
	}
	damages := []combat.Damage{fire(5)}
	cmd, err := combat.NewApplyDamageCommand(subject(), pc, damages, nil, combat.OtherSource, combat.NormalHit, nil)
	if err != nil {
		t.Fatal(err)
	}
	damages[0] = fire(99)
	res, err := combat.NewApplyDamage().Run(context.Background(), cmd)
	if err != nil || res.(combat.DamageApplied).Status().HitPoints().Current() != 5 {
		t.Errorf("result = %v, %v, want the damage the command was built with", res, err)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-114 - an unpreventable list alone is a valid command.
// Path: happy.
func TestApplyDamageUnpreventableOnly(t *testing.T) {
	t.Parallel()
	cmd, err := combat.NewApplyDamageCommand(subject(), pcStatus(t, 10, 20, 0), nil, nil, combat.OtherSource, combat.NormalHit, []combat.Damage{fire(4)})
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewApplyDamage().Run(context.Background(), cmd)
	if err != nil || res.(combat.DamageApplied).Status().HitPoints().Current() != 6 {
		t.Errorf("= %v, %v", res, err)
	}
}
