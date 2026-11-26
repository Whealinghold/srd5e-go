package combat_test

import (
	"context"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/hit-points-and-damage.md section 6 - a use case is silent without WithTelemetry; a nil
// clock is ignored (time.Now stays); a zero-value use case is silent and works (the zero Telemetry is Noop).
// Path: unhappy (boundary).
func TestOptions(t *testing.T) {
	t.Parallel()
	cmd, _ := combat.NewStabilizeCreatureCommand(subject(), dyingStatus(t, 20, 0, 0))
	rec := telemetrytest.NewRecorder()
	uc := combat.NewStabilizeCreature(combat.WithTelemetry(rec.Telemetry()), combat.WithClock(nil))
	if _, err := uc.Run(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if d := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(d) != 1 || d[0].Value < 0 {
		t.Errorf("duration = %+v, want one non-negative sample from time.Now", d)
	}
	for name, u := range map[string]combat.StabilizeCreature{"default": combat.NewStabilizeCreature(), "zero": {}, "clock only": combat.NewStabilizeCreature(combat.WithClock(time.Now))} {
		if res, err := u.Run(context.Background(), cmd); err != nil || res == nil {
			t.Errorf("%s = %v, %v", name, res, err)
		}
	}
}
