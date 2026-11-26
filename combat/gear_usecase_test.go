package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// gearSignalRow is one result of one gear use case with the signals it must produce.
type gearSignalRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	events                  []string
	entries                 [][2]string // (gear kind, entry) of srd5e.combat.gear.entries
	holds                   float64     // sample of srd5e.combat.gear.holds_in_force, -1 for none
	ended                   [][2]string // (hold kind, end cause) of srd5e.combat.gear.hold_ended
}

func gearSignalRows(t *testing.T) []gearSignalRow {
	t.Helper()
	ctx := context.Background()
	deploy := func(set combat.DeployedGears, id string, kind combat.DeployedGearKind, distance int) func(...combat.Option) (any, error) {
		cmd, err := combat.NewDeployGearCommand(set, gearID(id), kind, gid("owner"), distance, combat.GroundLevelLevel)
		if err != nil {
			t.Fatal(err)
		}
		return func(opts ...combat.Option) (any, error) { return combat.NewDeployGear(opts...).Run(ctx, cmd) }
	}
	maintain := func(set combat.DeployedGears, m combat.GearMaintenance) func(...combat.Option) (any, error) {
		cmd, err := combat.NewMaintainDeployedGearCommand(set, combat.GearHolds{}, m)
		if err != nil {
			t.Fatal(err)
		}
		return func(opts ...combat.Option) (any, error) { return combat.NewMaintainDeployedGear(opts...).Run(ctx, cmd) }
	}
	entry := func(set combat.DeployedGears, change func(*combat.ResolveGearEntryInput), faces ...int) func(...combat.Option) (any, error) {
		in := gearEntryIn(set)
		change(&in)
		cmd, err := combat.NewResolveGearEntryCommand(in)
		if err != nil {
			t.Fatal(err)
		}
		return func(opts ...combat.Option) (any, error) {
			u, err := combat.NewResolveGearEntry(dicetest.NewFixedRoller(faces...), opts...)
			if err != nil {
				return nil, err
			}
			return u.Run(ctx, cmd)
		}
	}
	net := func(change func(*combat.ThrowNetInput), faces ...int) func(...combat.Option) (any, error) {
		in := netIn()
		change(&in)
		cmd, err := combat.NewThrowNetCommand(in)
		if err != nil {
			t.Fatal(err)
		}
		return func(opts ...combat.Option) (any, error) {
			u, _ := combat.NewThrowNet(dicetest.NewFixedRoller(faces...), opts...)
			return u.Run(ctx, cmd)
		}
	}
	bind := func(change func(*combat.BindCreatureInput), faces ...int) func(...combat.Option) (any, error) {
		in := bindIn(t)
		change(&in)
		cmd, err := combat.NewBindCreatureCommand(in)
		if err != nil {
			t.Fatal(err)
		}
		return func(opts ...combat.Option) (any, error) {
			u, _ := combat.NewBindCreature(dicetest.NewFixedRoller(faces...), opts...)
			return u.Run(ctx, cmd)
		}
	}
	release := func(kind combat.HoldKind, change func(*combat.ReleaseFromHoldInput), faces ...int) func(...combat.Option) (any, error) {
		in := releaseIn(relHolds(t, kind))
		change(&in)
		cmd, err := combat.NewReleaseFromHoldCommand(in)
		if err != nil {
			t.Fatal(err)
		}
		return func(opts ...combat.Option) (any, error) {
			u, _ := combat.NewReleaseFromHold(dicetest.NewFixedRoller(faces...), opts...)
			return u.Run(ctx, cmd)
		}
	}
	nothing := func(*combat.ThrowNetInput) {}
	const a, r, i = "applied", "refused", "invalid"
	caltrops, trap, oil := combat.DeployedGearKindCaltrops, combat.DeployedGearKindHuntingTrap, combat.DeployedGearKindPouredOil
	spec := gearSpec
	return []gearSignalRow{
		{"deploy_gear placed", spec + "FR-064 FR-065", "happy", "deploy_gear", deploy(combat.DeployedGears{}, "new", caltrops, 5), a, "placed", []string{"deployed_gear_placed"}, nil, -1, nil},
		{"deploy_gear refused, out_of_reach", spec + "FR-065", "unhappy (boundary)", "deploy_gear", deploy(combat.DeployedGears{}, "new", caltrops, 6), r, "out_of_reach", nil, nil, -1, nil},
		{"deploy_gear invalid, negative_distance", spec + "FR-065 FR-068", "unhappy (invalid input)", "deploy_gear", deploy(combat.DeployedGears{}, "new", caltrops, -1), i, "negative_distance", nil, nil, -1, nil},
		{"maintain_deployed_gear applied", spec + "FR-064 FR-065", "happy", "maintain_deployed_gear", maintain(gearOne(t, oil), combat.LightOil{GearID: "g", Stamp: gearStamp(1, "a")}), a, "applied", []string{"poured_oil_lit"}, nil, -1, nil},
		{"maintain_deployed_gear applied, nothing burns out", spec + "FR-012", "unhappy (empty)", "maintain_deployed_gear", maintain(combat.DeployedGears{}, combat.ExpireBurningOil{EndedTurn: gearStamp(1, "a")}), a, "applied", nil, nil, -1, nil},
		{"maintain_deployed_gear refused, not_recoverable", spec + "FR-065", "unhappy (state error)", "maintain_deployed_gear", maintain(gearOne(t, trap), combat.RecoverGear{GearID: "g"}), r, "not_recoverable", nil, nil, -1, nil},
		{"resolve_gear_entry saved", spec + "FR-066", "happy", "resolve_gear_entry", entry(gearOne(t, caltrops), func(*combat.ResolveGearEntryInput) {}, 15), a, "saved", nil, [][2]string{{"caltrops", "save_passed"}}, -1, nil},
		{"resolve_gear_entry failed", spec + "FR-066", "unhappy (boundary)", "resolve_gear_entry", entry(gearOne(t, caltrops), func(*combat.ResolveGearEntryInput) {}, 14), a, "failed", nil, [][2]string{{"caltrops", "save_failed"}}, -1, nil},
		{"resolve_gear_entry trap failed", spec + "FR-066 FR-063", "unhappy (rule interaction)", "resolve_gear_entry", entry(gearOne(t, trap), func(*combat.ResolveGearEntryInput) {}, 3, 2), a, "failed", []string{"hunting_trap_sprung"}, [][2]string{{"hunting_trap", "save_failed"}}, -1, nil},
		{"resolve_gear_entry damaged", spec + "FR-066", "happy", "resolve_gear_entry", entry(gearBurning(t, 1, "walker"), func(*combat.ResolveGearEntryInput) {}), a, "damaged", nil, [][2]string{{"poured_oil", "damaged"}}, -1, nil},
		{"resolve_gear_entry refused, already_triggered_this_turn", spec + "FR-066", "unhappy (state error)", "resolve_gear_entry", entry(gearBurning(t, 1, "walker"), func(in *combat.ResolveGearEntryInput) {
			in.Ledger = combat.EntryLedger{}.Record(combat.EntryRecord{GearID: "g", Creature: "walker", Stamp: in.Stamp})
		}), r, "already_triggered_this_turn", nil, [][2]string{{"poured_oil", "already_triggered"}}, -1, nil},
		{"resolve_gear_entry refused, oil_not_burning", spec + "FR-066", "unhappy (state error)", "resolve_gear_entry", entry(gearOne(t, oil), func(*combat.ResolveGearEntryInput) {}), r, "oil_not_burning", nil, [][2]string{{"poured_oil", "not_active"}}, -1, nil},
		{"resolve_gear_entry refused, trap_already_sprung", spec + "FR-066", "unhappy (state error)", "resolve_gear_entry", entry(gearSprung(t), func(*combat.ResolveGearEntryInput) {}), r, "trap_already_sprung", nil, [][2]string{{"hunting_trap", "not_active"}}, -1, nil},
		{"resolve_gear_entry refused, trigger_not_applicable", spec + "FR-066", "unhappy (invalid input)", "resolve_gear_entry", entry(gearOne(t, caltrops), func(in *combat.ResolveGearEntryInput) { in.Trigger = combat.AreaTriggerEndsTurn }), r, "trigger_not_applicable", nil, [][2]string{{"caltrops", "not_applicable"}}, -1, nil},
		{"resolve_gear_entry refused, unknown_gear is not an entry", spec + "FR-066", "unhappy (state error)", "resolve_gear_entry", entry(gearOne(t, caltrops), func(in *combat.ResolveGearEntryInput) { in.GearID = "zz" }), r, "unknown_gear", nil, nil, -1, nil},
		{"resolve_gear_entry invalid, end_turn_stamp_mismatch is not an entry", spec + "FR-021 FR-066", "unhappy (invalid input)", "resolve_gear_entry", entry(gearBurning(t, 1, "walker"), func(in *combat.ResolveGearEntryInput) {
			in.Trigger, in.Stamp = combat.AreaTriggerEndsTurn, gearStamp(1, "other")
		}), i, "end_turn_stamp_mismatch", nil, nil, -1, nil},
		{"throw_net held", spec + "FR-064 FR-067", "happy", "throw_net", net(nothing, 1), a, "held", []string{"gear_hold_started"}, nil, 1, nil},
		{"throw_net saved", spec + "FR-065", "unhappy (boundary)", "throw_net", net(nothing, 13), a, "saved", nil, nil, -1, nil},
		{"throw_net auto_succeeded", spec + "FR-038", "unhappy (boundary)", "throw_net", net(func(in *combat.ThrowNetInput) { in.TargetSize = content.SizeHuge }), a, "auto_succeeded", nil, nil, -1, nil},
		{"throw_net refused, out_of_reach", spec + "FR-065", "unhappy (boundary)", "throw_net", net(func(in *combat.ThrowNetInput) { in.DistanceFeet = 16 }), r, "out_of_reach", nil, nil, -1, nil},
		{"bind_creature bound", spec + "FR-064 FR-067", "happy", "bind_creature", bind(func(*combat.BindCreatureInput) {}, 20), a, "bound", []string{"gear_hold_started"}, nil, 1, nil},
		{"bind_creature failed", spec + "FR-065", "unhappy (boundary)", "bind_creature", bind(func(*combat.BindCreatureInput) {}, 12), a, "failed", nil, nil, -1, nil},
		{"bind_creature refused, target_willing", spec + "FR-065", "unhappy (rule interaction)", "bind_creature", bind(func(in *combat.BindCreatureInput) { in.Willingness = combat.WillingnessWilling }), r, "target_willing", nil, nil, -1, nil},
		{"release_from_hold released", spec + "FR-064 FR-067", "happy", "release_from_hold", release(combat.HoldKindNet, func(*combat.ReleaseFromHoldInput) {}, 20), a, "released", []string{"gear_hold_ended"}, nil, 0, [][2]string{{"net", "escaped"}}},
		{"release_from_hold released with the key", spec + "FR-055 FR-067", "happy", "release_from_hold", release(combat.HoldKindManacles, func(in *combat.ReleaseFromHoldInput) { in.Method = combat.HoldReleaseMethodKey }), a, "released", []string{"gear_hold_ended"}, nil, 0, [][2]string{{"manacles", "unlocked"}}},
		{"release_from_hold held", spec + "FR-065", "unhappy (boundary)", "release_from_hold", release(combat.HoldKindNet, func(*combat.ReleaseFromHoldInput) {}, 9), a, "held", nil, nil, -1, nil},
		{"release_from_hold refused, no_action", spec + "FR-065", "unhappy (state error)", "release_from_hold", release(combat.HoldKindNet, func(in *combat.ReleaseFromHoldInput) { in.Actions = combat.ConditionActionAvailabilityNone }), r, "no_action", nil, nil, -1, nil},
	}
}

// Requirement: per case
// Path: per case.
func TestGearUseCaseSignals(t *testing.T) {
	t.Parallel()
	ids := []string{"walker", "victim", "owner", "thrower", "binder"}
	for _, c := range gearSignalRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if plain, err := c.run(); err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-062]", got, plain, err)
			}
			outcome := map[string]string{"applied": "success", "refused": "failure", "invalid": "failure"}[c.class]
			want := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != (c.class == "invalid") || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, error status for invalid only [%s]", spans, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [FR-064]", sa, want)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [FR-065]", ops, want)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s", dur)
			}
			wantKinds := slices.Clone(c.events)
			if wantKinds == nil {
				wantKinds = []string{}
			}
			if counted := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(counted, wantKinds) {
				t.Errorf("event kinds counted = %v, want %v", counted, wantKinds)
			}
			var entries [][2]string
			for _, m := range measured(rec, "srd5e.combat.gear.entries") {
				entries = append(entries, [2]string{m["srd5e.combat.gear.kind"].(string), m["srd5e.combat.gear.entry"].(string)})
			}
			if !reflect.DeepEqual(entries, c.entries) {
				t.Errorf("gear entries = %v, want %v [FR-066]", entries, c.entries)
			}
			var ended [][2]string
			for _, m := range measured(rec, "srd5e.combat.gear.hold_ended") {
				ended = append(ended, [2]string{m["srd5e.combat.gear.kind"].(string), m["srd5e.combat.gear.end_cause"].(string)})
			}
			if !reflect.DeepEqual(ended, c.ended) {
				t.Errorf("holds ended = %v, want %v [FR-067]", ended, c.ended)
			}
			inForce := rec.Meter.MeasurementsNamed("srd5e.combat.gear.holds_in_force")
			switch {
			case c.holds < 0 && len(inForce) != 0:
				t.Errorf("holds in force = %+v for a result without holds", inForce)
			case c.holds >= 0 && (len(inForce) != 1 || inForce[0].Value != c.holds || !reflect.DeepEqual(attrMap(inForce[0].Attrs), map[string]any{"srd5e.combat.operation": c.operation})):
				t.Errorf("holds in force = %+v, want one sample of %v for %s [FR-067]", inForce, c.holds, c.operation)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0 for a result", n)
			}
			for _, m := range rec.Meter.Measurements() {
				for _, at := range m.Attrs {
					if s, ok := at.Value().(string); ok && slices.Contains(ids, s) {
						t.Errorf("%s attribute %s carries the id %q [FR-068]", m.Name, at.Key(), s)
					}
				}
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-061, FR-068, errors-and-results.md FR-031 - a roller that fails, panics or answers another expression is recorded once by each rolling use case: error status and one recorded error on the span, one srd5e.combat.errors sample with error.type dice_roller_error, and no operations, events, entries or holds signal; nothing is logged.
// Path: unhappy (failure of collaborator, telemetry on failure).
func TestGearUseCasePortFailureIsRecordedOnce(t *testing.T) {
	t.Parallel()
	adv := rollWith(0, modifier.Advantage{})
	uses := map[string]func(combat.Roller, ...combat.Option) error{
		"resolve_gear_entry": func(rl combat.Roller, o ...combat.Option) error {
			in := gearEntryIn(gearOne(t, combat.DeployedGearKindCaltrops))
			in.Attempt = adv
			_, err := gearEntry(t, in, rl, o...)
			return err
		},
		"throw_net": func(rl combat.Roller, o ...combat.Option) error {
			in := netIn()
			in.Attempt = adv
			_, err := netRun(t, in, rl, o...)
			return err
		},
		"bind_creature": func(rl combat.Roller, o ...combat.Option) error {
			in := bindIn(t)
			in.Attempt = adv
			_, err := bindRun(t, in, rl, o...)
			return err
		},
		"release_from_hold": func(rl combat.Roller, o ...combat.Option) error {
			in := releaseIn(relHolds(t, combat.HoldKindNet))
			in.Attempt = adv
			_, err := releaseRun(t, in, rl, o...)
			return err
		},
	}
	for op, run := range uses {
		for name, roller := range map[string]combat.Roller{"fails": envFailingRoller(errors.New("roller down")), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
			rec, opts := recording()
			err := run(roller, opts...)
			envPortFailure(t, err)
			spans := rec.Tracer.SpansNamed("srd5e.combat." + op)
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Errorf("%s %s: spans = %+v, want one ended span with one recorded error", op, name, spans)
			}
			errs := measured(rec, "srd5e.combat.errors")
			if len(errs) != 1 || errs[0]["error.type"] != srderr.ErrorType(err) || errs[0]["error.type"] != "dice_roller_error" || errs[0]["srd5e.combat.operation"] != op {
				t.Errorf("%s %s: errors = %v, want one dice_roller_error", op, name, errs)
			}
			for _, n := range []string{"operations", "events", "gear.entries", "gear.holds_in_force", "gear.hold_ended"} {
				if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
					t.Errorf("%s %s: %s recorded %d on an error", op, name, n, len(m))
				}
			}
			checkTelemetryHygiene(t, rec)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-062, FR-068 - a call whose ctx is already cancelled, or whose command is the zero command, is answered before any signal: nothing is recorded.
// Path: unhappy (cancellation, invalid input).
func TestGearUseCaseEarlyErrorsRecordNothing(t *testing.T) {
	t.Parallel()
	rec, opts := recording()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd, _ := combat.NewDeployGearCommand(combat.DeployedGears{}, "g", combat.DeployedGearKindCaltrops, "a", 0, combat.GroundLevelLevel)
	if _, err := combat.NewDeployGear(opts...).Run(ctx, cmd); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if _, err := combat.NewDeployGear(opts...).Run(context.Background(), combat.DeployGearCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}
	if len(rec.Meter.Measurements()) != 0 || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("recorded %d measurements and %d spans, want none", len(rec.Meter.Measurements()), len(rec.Tracer.Spans()))
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-066, FR-068 - the gear attribute keys and values are the catalogued names; none is an id, a Difficulty Class, a distance or free text (SRD-free package design).
// Path: unhappy (telemetry).
func TestGearSignalNamesArePinned(t *testing.T) {
	t.Parallel()
	rec, opts := recording()
	if _, err := gearEntry(t, gearEntryIn(gearOne(t, combat.DeployedGearKindCaltrops)), dicetest.NewFixedRoller(1), opts...); err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, m := range rec.Meter.Measurements() {
		names = append(names, m.Name)
		for _, a := range m.Attrs {
			if !strings.HasPrefix(a.Key(), "srd5e.") && a.Key() != "error.type" {
				t.Errorf("attribute key %q is not a catalogued key", a.Key())
			}
		}
	}
	for _, want := range []string{"srd5e.combat.gear.entries", "srd5e.combat.operations", "srd5e.combat.operation_duration"} {
		if !slices.Contains(names, want) {
			t.Errorf("measurements %v lack %s", names, want)
		}
	}
}
