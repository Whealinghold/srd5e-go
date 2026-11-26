package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-108 - hit_points_restored records one sample per applied
// command: the amount regained, 0 included, and none for a rejection.
// Path: per case.
func TestRestoreHitPointsSignals(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		st   combat.HitPointStatus
		want []float64
	}{
		"regained 6":   {pcStatus(t, 14, 20, 0), []float64{6}},
		"already full": {pcStatus(t, 20, 20, 0), []float64{0}},
		"dead":         {deadStatus(t, combat.MassiveDamage), nil},
	} {
		rec, opts := recording()
		cmd, err := combat.NewRestoreHitPointsCommand(subject(), c.st, 8, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewRestoreHitPoints(opts...).Run(context.Background(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		var got []float64
		for _, m := range rec.Meter.MeasurementsNamed("srd5e.combat.hit_points_restored") {
			got = append(got, m.Value)
		}
		if len(got) != len(c.want) || len(got) == 1 && got[0] != c.want[0] {
			t.Errorf("%s: samples = %v, want %v", name, got, c.want)
		}
		if a, ok := res.(combat.HealingApplied); ok && float64(a.HitPointsRegained()) != c.want[0] {
			t.Errorf("%s: regained = %d", name, a.HitPointsRegained())
		}
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-037, FR-115 - the command rejects a negative amount and an
// unknown ceiling; a nil ceiling is NoHealingCeiling.
// Path: unhappy (invalid input).
func TestNewRestoreHitPointsCommand(t *testing.T) {
	t.Parallel()
	pc := pcStatus(t, 10, 20, 0)
	if _, err := combat.NewRestoreHitPointsCommand(subject(), pc, -1, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative error = %v", err)
	}
	ceiling, _ := combat.NewHealingCeilingAt(12)
	cmd, err := combat.NewRestoreHitPointsCommand(subject(), pc, 9, ceiling)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := combat.NewRestoreHitPoints().Run(context.Background(), cmd)
	if res.(combat.HealingApplied).Status().HitPoints().Current() != 12 {
		t.Errorf("ceiling ignored")
	}
}
