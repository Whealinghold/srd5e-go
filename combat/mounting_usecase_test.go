package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// mntSignalRow is one result of one mounting use case with the signals it must produce.
type mntSignalRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	control, trigger        string // "" for absent
	events                  []string
	moved                   float64 // the sample of mounting_movement_spent, -1 for none
}

func mntSignalRows(t *testing.T) []mntSignalRow {
	t.Helper()
	ctx := context.Background()
	mount := func(change func(*combat.MountCreatureInput)) func(...combat.Option) (any, error) {
		in := mntInput(t)
		change(&in)
		cmd := mntCommand(t, in)
		return func(opts ...combat.Option) (any, error) { return combat.NewMountCreature(opts...).Run(ctx, cmd) }
	}
	dismount := func(status combat.RiderStatus, left int) func(...combat.Option) (any, error) {
		cmd := mntDismountCommand(t, status, mntTurn(t, combat.RiderTurnActive, 30, left))
		return func(opts ...combat.Option) (any, error) { return combat.NewDismountCreature(opts...).Run(ctx, cmd) }
	}
	fall := func(status combat.RiderStatus, trigger combat.FallingOffTrigger, attempt d20.SaveAttempt, faces ...int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			return mntFallWith(t, dicetest.NewFixedRoller(faces...), status, trigger, attempt, combat.LandingSpaceUnoccupiedWithin5Feet, opts...)
		}
	}
	spec := "doc/specs/mounted-and-underwater-combat.md "
	const a, r = "applied", "refused"
	ctrl, ind := mntMounted(t, combat.MountControlControlled), mntMounted(t, combat.MountControlIndependent)
	moved, riderProne := combat.FallingOffTriggerMountMovedAgainstItsWill, combat.FallingOffTriggerRiderKnockedProne
	return []mntSignalRow{
		{"mount_creature applied, controlled", spec + "FR-058 FR-059 FR-060 FR-061 FR-064", "happy", "mount_creature", mount(func(*combat.MountCreatureInput) {}), a, "applied", "controlled", "", []string{"creature_mounted"}, 15},
		{"mount_creature applied, independent", spec + "FR-061", "unhappy (rule interaction)", "mount_creature", mount(func(in *combat.MountCreatureInput) { in.Control = combat.MountControlIndependent }), a, "applied", "independent", "", []string{"creature_mounted"}, 15},
		{"mount_creature applied, cost 0 is recorded", spec + "FR-064", "unhappy (boundary)", "mount_creature", mount(func(in *combat.MountCreatureInput) { in.Turn = mntTurn(t, combat.RiderTurnActive, 1, 1) }), a, "applied", "controlled", "", []string{"creature_mounted"}, 0},
		{"mount_creature refused, mount_unwilling", spec + "FR-059 FR-060", "unhappy (state error)", "mount_creature", mount(func(in *combat.MountCreatureInput) {
			in.Facts = mntFacts(t, combat.MountWillingnessUnwilling, combat.MountAnatomyAppropriate, combat.MountTrainingTrainedToAcceptRider, combat.MountHabitatOther, combat.ExoticSaddleNotFitted)
		}), r, "mount_unwilling", "controlled", "", nil, -1},
		{"mount_creature refused, insufficient_movement", spec + "FR-060 FR-064", "unhappy (boundary)", "mount_creature", mount(func(in *combat.MountCreatureInput) { in.Turn = mntTurn(t, combat.RiderTurnActive, 30, 14) }), r, "insufficient_movement", "controlled", "", nil, -1},
		{"dismount_creature applied", spec + "FR-058 FR-061 FR-064", "happy", "dismount_creature", dismount(ind, 30), a, "applied", "independent", "", []string{"creature_dismounted"}, 15},
		{"dismount_creature refused, no_movement_remaining", spec + "FR-060", "unhappy (boundary)", "dismount_creature", dismount(ctrl, 0), r, "no_movement_remaining", "controlled", "", nil, -1},
		{"dismount_creature refused, rider_not_mounted has no control", spec + "FR-060 FR-061", "unhappy (state error)", "dismount_creature", dismount(combat.NotMounted(), 30), r, "rider_not_mounted", "", "", nil, -1},
		{"resolve_falling_off stayed_mounted", spec + "FR-059 FR-060 FR-061", "happy", "resolve_falling_off", fall(ctrl, riderProne, rollWith(0), 15), a, "stayed_mounted", "controlled", "rider_knocked_prone", nil, -1},
		{"resolve_falling_off fell_off", spec + "FR-059 FR-060 FR-061", "unhappy (state error)", "resolve_falling_off", fall(ind, moved, rollWith(0), 3), a, "fell_off", "independent", "mount_moved_against_its_will", []string{"fell_off_mount"}, -1},
		{"resolve_falling_off fell_off without a die", spec + "FR-034", "unhappy (rule interaction)", "resolve_falling_off", fall(ctrl, combat.FallingOffTriggerMountKnockedProne, d20.AutomaticFailure{}), a, "fell_off", "controlled", "mount_knocked_prone", []string{"fell_off_mount"}, -1},
		{"resolve_falling_off rider_not_mounted has no control", spec + "FR-060 FR-061", "unhappy (state error)", "resolve_falling_off", fall(combat.NotMounted(), riderProne, rollWith(0), 15), r, "rider_not_mounted", "", "rider_knocked_prone", nil, -1},
	}
}

// Requirement: per case
// Path: per case.
func TestMountingUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range mntSignalRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if plain, err := c.run(); err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-063]", got, plain, err)
			}
			outcome := map[string]string{"applied": "success", "refused": "failure"}[c.class]
			want := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			if c.control != "" {
				want["srd5e.combat.mount_control"] = c.control
			}
			if c.trigger != "" {
				want["srd5e.combat.falling_off_trigger"] = c.trigger
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span without error status [%s]", spans, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [FR-061]", sa, want)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [FR-058]", ops, want)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s", dur)
			}
			wantKinds := slices.Clone(c.events)
			if wantKinds == nil {
				wantKinds = []string{}
			}
			if counted := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(counted, wantKinds) {
				t.Errorf("event kinds counted = %v, want %v [FR-058]", counted, wantKinds)
			}
			spent := rec.Meter.MeasurementsNamed("srd5e.combat.mounting_movement_spent")
			switch {
			case c.moved < 0 && len(spent) != 0:
				t.Errorf("movement spent = %v on a call that charges nothing [FR-064]", spent)
			case c.moved >= 0 && (len(spent) != 1 || spent[0].Value != c.moved || !reflect.DeepEqual(attrMap(spent[0].Attrs), map[string]any{"srd5e.combat.operation": c.operation})):
				t.Errorf("movement spent = %+v, want one sample of %v for %s [FR-064]", spent, c.moved, c.operation)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0 for a result", n)
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-037, FR-062 - a roller that fails, panics or answers another expression is recorded once: error status and one recorded error on the span, one srd5e.combat.errors sample with error.type dice_roller_error, and no operations, events, movement or srd5e.d20_test signal; nothing is logged.
// Path: unhappy (collaborator failure, telemetry on failure).
func TestResolveFallingOffPortFailureIsRecordedOnce(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	for name, roller := range map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		rec, opts := recording()
		_, err := mntFallWith(t, roller, mntMounted(t, combat.MountControlControlled), combat.FallingOffTriggerMountMovedAgainstItsWill, rollWith(0, modifierAdvantage()), combat.LandingSpaceUnoccupiedWithin5Feet, opts...)
		envPortFailure(t, err)
		spans := rec.Tracer.SpansNamed("srd5e.combat.resolve_falling_off")
		if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
			t.Errorf("%s: spans = %+v, want one ended span with one recorded error", name, spans)
		}
		errs := measured(rec, "srd5e.combat.errors")
		if len(errs) != 1 || errs[0]["error.type"] != srderr.ErrorType(err) || errs[0]["error.type"] != "dice_roller_error" || errs[0]["srd5e.combat.operation"] != "resolve_falling_off" {
			t.Errorf("%s: errors = %v, want one dice_roller_error", name, errs)
		}
		for _, n := range []string{"operations", "events", "mounting_movement_spent"} {
			if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
				t.Errorf("%s: %s recorded %d on an error", name, n, len(m))
			}
		}
		checkTelemetryHygiene(t, rec)
	}
}
