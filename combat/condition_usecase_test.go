package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// conditionRow is one variant of one condition use case.
type conditionRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	events                  []string
	activeKind              string
	activeCount             float64
}

func conditionRows(t *testing.T) []conditionRow {
	t.Helper()
	ctx := context.Background()
	apply := func(s combat.ConditionSet, c combat.Condition) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewApplyConditionToSetCommand(s, subject(), c)
			if err != nil {
				return nil, err
			}
			return combat.NewApplyConditionToSet(opts...).Run(ctx, cmd)
		}
	}
	remove := func(s combat.ConditionSet, k content.ConditionKind) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewRemoveConditionFromSetCommand(s, subject(), k)
			if err != nil {
				return nil, err
			}
			return combat.NewRemoveConditionFromSet(opts...).Run(ctx, cmd)
		}
	}
	reduce := func(s combat.ConditionSet, n int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewReduceExhaustionOnSetCommand(s, subject(), n)
			if err != nil {
				return nil, err
			}
			return combat.NewReduceExhaustionOnSet(opts...).Run(ctx, cmd)
		}
	}
	const table = "conditions.md section 4.11 Result classification"
	const (
		a, r = "applied", "refused"
	)
	blinded := condSet(t, condConds(combat.Blinded{}))
	paralyzed := condSet(t, condConds(combat.Paralyzed{}))
	return []conditionRow{
		{"apply_condition_to_set added", table + "; FR-119", "happy", "apply_condition_to_set", apply(combat.ConditionSet{}, combat.Unconscious{}), a, "added", []string{"condition_applied"}, "unconscious", 3},
		{"apply_condition_to_set already_held", table + "; FR-120", "unhappy (rule interaction)", "apply_condition_to_set", apply(blinded, combat.Blinded{}), r, "already_held", []string{}, "blinded", 1},
		{"apply_condition_to_set immune", table + "; FR-120", "unhappy (rule interaction)", "apply_condition_to_set", apply(condSet(t, nil, cB), combat.Blinded{}), r, "immune", []string{}, "blinded", 0},
		{"apply_condition_to_set at_maximum", table + "; FR-120", "unhappy (boundary)", "apply_condition_to_set", apply(condSet(t, condConds(condLevel(6))), condLevel(1)), r, "at_maximum", []string{}, "exhaustion", 1},
		{"remove_condition_from_set cleared", table + "; FR-119", "happy", "remove_condition_from_set", remove(blinded, cB), a, "cleared", []string{"condition_removed"}, "blinded", 0},
		{"remove_condition_from_set lowered", table + "; FR-119", "happy", "remove_condition_from_set", remove(condSet(t, condConds(condLevel(3))), cE), a, "lowered", []string{"condition_level_reduced"}, "exhaustion", 1},
		{"remove_condition_from_set removal_not_present", table + "; FR-120", "unhappy (state error)", "remove_condition_from_set", remove(blinded, cD), r, "removal_not_present", []string{}, "deafened", 1},
		{"remove_condition_from_set implied_by", table + "; FR-120", "unhappy (state error)", "remove_condition_from_set", remove(paralyzed, cI), r, "implied_by", []string{}, "incapacitated", 2},
		{"reduce_exhaustion_on_set reduced", table + "; FR-119", "happy", "reduce_exhaustion_on_set", reduce(condSet(t, condConds(condLevel(3))), 1), a, "reduced", []string{"condition_level_reduced"}, "exhaustion", 1},
		{"reduce_exhaustion_on_set ended", table + "; FR-119", "unhappy (boundary)", "reduce_exhaustion_on_set", reduce(condSet(t, condConds(condLevel(1), combat.Blinded{})), 1), a, "ended", []string{"condition_removed"}, "exhaustion", 1},
		{"reduce_exhaustion_on_set reduction_not_present", table + "; FR-120", "unhappy (state error)", "reduce_exhaustion_on_set", reduce(blinded, 1), r, "reduction_not_present", []string{}, "exhaustion", 1},
	}
}

// Requirement: per case
// Path: per case.
func TestConditionUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range conditionRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			plain, err := c.run()
			if err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-118]", got, plain, err)
			}
			outcome := "success"
			if c.class == "refused" {
				outcome = "failure"
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span without error status [%s]", spans, c.requirement)
			}
			want := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [%s]", ops, want, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v", sa, want)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s", dur)
			}
			if kinds := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(kinds, c.events) {
				t.Errorf("event kinds counted = %v, want %v", kinds, c.events)
			}
			active := rec.Meter.MeasurementsNamed("srd5e.combat.condition.active")
			wantActive := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.combat.condition.kind": c.activeKind}
			if len(active) != 1 || active[0].Value != c.activeCount || !reflect.DeepEqual(attrMap(active[0].Attrs), wantActive) {
				t.Errorf("condition.active = %+v, want %v of %v [FR-122]", active, c.activeCount, wantActive)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0 for a result", n)
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/conditions.md FR-116, FR-117 - each use case returns the outcome, set and events of the domain
// operation.
// Path: happy.
func TestConditionUseCasesMatchTheDomainOperations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := condSet(t, condConds(combat.Unconscious{}, condLevel(2)))
	cmdA, _ := combat.NewApplyConditionToSetCommand(s, subject(), combat.Poisoned{})
	if got, err := combat.NewApplyConditionToSet().Run(ctx, cmdA); err != nil || !reflect.DeepEqual(got, s.Apply(subject(), combat.Poisoned{})) {
		t.Errorf("ApplyConditionToSet = %#v, %v, want the domain result", got, err)
	}
	cmdR, _ := combat.NewRemoveConditionFromSetCommand(s, subject(), cU)
	if got, err := combat.NewRemoveConditionFromSet().Run(ctx, cmdR); err != nil || !reflect.DeepEqual(got, s.Remove(subject(), cU)) {
		t.Errorf("RemoveConditionFromSet = %#v, %v, want the domain result", got, err)
	}
	cmdE, _ := combat.NewReduceExhaustionOnSetCommand(s, subject(), 1)
	want, _ := s.ReduceExhaustion(subject(), 1)
	if got, err := combat.NewReduceExhaustionOnSet().Run(ctx, cmdE); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("ReduceExhaustionOnSet = %#v, %v, want the domain result", got, err)
	}
	var zero combat.ApplyConditionToSet
	if got, err := zero.Run(ctx, cmdA); err != nil || !reflect.DeepEqual(got, s.Apply(subject(), combat.Poisoned{})) {
		t.Errorf("the zero ApplyConditionToSet = %#v, %v, want a working silent use case", got, err)
	}
}

// Requirement: doc/specs/conditions.md FR-121, FR-045 - ReduceExhaustionOnSet with levels below 1 returns the validation
// error unchanged (not a port failure), the span records it once with error status, the duration sample exists and
// operations, events, errors and condition.active do not.
// Path: unhappy (invalid input).
func TestReduceExhaustionOnSetRecordsTheValidationErrorOnTheSpan(t *testing.T) {
	t.Parallel()
	for _, levels := range []int{0, -1} {
		rec, opts := recording()
		cmd, err := combat.NewReduceExhaustionOnSetCommand(condSet(t, condConds(condLevel(3))), subject(), levels)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewReduceExhaustionOnSet(opts...).Run(context.Background(), cmd)
		var port *srderr.PortError
		if res != nil || !errors.Is(err, srderr.ErrInvalidInput) || errors.As(err, &port) {
			t.Fatalf("Run(levels %d) = %v, %v, want nil and a plain ErrInvalidInput", levels, res, err)
		}
		spans := rec.Tracer.SpansNamed("srd5e.combat.reduce_exhaustion_on_set")
		if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !errors.Is(spans[0].Errors[0], srderr.ErrInvalidInput) {
			t.Errorf("spans = %+v, want one with error status and the error recorded once", spans)
		}
		if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration")); n != 1 {
			t.Errorf("duration samples = %d, want 1", n)
		}
		for _, name := range []string{"srd5e.combat.operations", "srd5e.combat.events", "srd5e.combat.condition.active"} {
			if n := len(rec.Meter.MeasurementsNamed(name)); n != 0 {
				t.Errorf("%s recorded %d samples on the error path, want 0", name, n)
			}
		}
		assertInvalidInputCounted(t, rec, "reduce_exhaustion_on_set")
		if got := rec.Logger.Records(); len(got) != 0 {
			t.Errorf("logged %v, want nothing", got)
		}
	}
}

// Requirement: doc/specs/conditions.md FR-119, testing.md section 4 - a zero command is invalid input and a cancelled context
// is returned unwrapped, both before any signal.
// Path: unhappy (invalid input).
func TestConditionUseCasesRejectZeroCommandAndCancelledContext(t *testing.T) {
	t.Parallel()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	rec, opts := recording()
	s := condSet(t, condConds(combat.Blinded{}))
	ap, _ := combat.NewApplyConditionToSetCommand(s, subject(), combat.Prone{})
	rm, _ := combat.NewRemoveConditionFromSetCommand(s, subject(), cB)
	rd, _ := combat.NewReduceExhaustionOnSetCommand(s, subject(), 1)
	calls := map[string][2]func(context.Context) error{
		"apply": {func(c context.Context) error { _, err := combat.NewApplyConditionToSet(opts...).Run(c, ap); return err },
			func(c context.Context) error {
				_, err := combat.NewApplyConditionToSet(opts...).Run(c, combat.ApplyConditionToSetCommand{})
				return err
			}},
		"remove": {func(c context.Context) error {
			_, err := combat.NewRemoveConditionFromSet(opts...).Run(c, rm)
			return err
		},
			func(c context.Context) error {
				_, err := combat.NewRemoveConditionFromSet(opts...).Run(c, combat.RemoveConditionFromSetCommand{})
				return err
			}},
		"reduce": {func(c context.Context) error {
			_, err := combat.NewReduceExhaustionOnSet(opts...).Run(c, rd)
			return err
		},
			func(c context.Context) error {
				_, err := combat.NewReduceExhaustionOnSet(opts...).Run(c, combat.ReduceExhaustionOnSetCommand{})
				return err
			}},
	}
	for name, c := range calls {
		if err := c[0](cancelled); !errors.Is(err, context.Canceled) {
			t.Errorf("%s with a cancelled ctx = %v, want context.Canceled", name, err)
		}
		if err := c[1](context.Background()); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s with the zero command = %v, want ErrInvalidInput", name, err)
		}
	}
	if n := len(rec.Tracer.Spans()) + len(rec.Meter.Measurements()); n != 0 {
		t.Errorf("signals recorded before validation: %d", n)
	}
}

// Requirement: doc/specs/conditions.md FR-119 - a command constructor rejects a zero target and, for a kind, an invalid kind,
// with an error matching ErrInvalidInput.
// Path: unhappy (invalid input).
func TestConditionCommandConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	s := combat.ConditionSet{}
	if _, err := combat.NewApplyConditionToSetCommand(s, "", combat.Blinded{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("apply with a zero target: %v", err)
	}
	if _, err := combat.NewRemoveConditionFromSetCommand(s, "", cB); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("remove with a zero target: %v", err)
	}
	if _, err := combat.NewRemoveConditionFromSetCommand(s, subject(), 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("remove of kind 0: %v", err)
	}
	if _, err := combat.NewReduceExhaustionOnSetCommand(s, "", 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("reduce with a zero target: %v", err)
	}
}
