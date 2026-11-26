package combat

import (
	"context"
	"maps"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// containmentUse is what every containment use case holds: the instrumenter for the shared signals and
// the telemetry for the signals of monster-engulf-and-swallow.md section 7. The zero value is silent and
// usable.
type containmentUse struct {
	in  instrumenter
	tel telemetry.Telemetry
}

func newContainmentUse(opts []Option) containmentUse {
	o := newOptions(opts)
	return containmentUse{in: newInstrumenter(o.tel, o.clock), tel: o.tel}
}

// runContainment is run plus the one debug log of FR-073: the operation and the result name, built lazily.
func runContainment[R any](ctx context.Context, u containmentUse, op combatOperation,
	work func(context.Context) (R, error), describe func(R) description) (R, error) {
	name := ""
	res, err := run(ctx, u.in, op, work, func(r R) description {
		d := describe(r)
		name = d.Name
		return d
	})
	if err != nil {
		return res, err
	}
	u.tel.WithDefaults().Logger.Log(ctx, telemetry.LevelDebug, msgContainmentOperation, func() []telemetry.Attr {
		return []telemetry.Attr{telemetry.String(attrOperation, op.Value()), telemetry.String(attrResult, name)}
	})
	return res, nil
}

// recordSet records the number of records of a result that carries a Containments.
func (u containmentUse) recordSet(ctx context.Context, op combatOperation, s Containments) {
	u.tel.WithDefaults().Meter.Histogram(telemetry.Descriptor{Name: metricContainmentOccupants, Unit: "{containment}",
		Description: "Containment records in the set a result carries."}).
		Record(ctx, float64(s.Len()), telemetry.String(attrOperation, op.Value()))
}

// recordEnded counts one containment that ended for the cause.
func (u containmentUse) recordEnded(ctx context.Context, cause ContainmentLeaveCause) {
	u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricContainmentEnded, Unit: "{containment}",
		Description: "Containments that ended, by reason."}).Add(ctx, 1, telemetry.String(attrContainmentEndReason, cause.String()))
}

// recordReleases counts each release by its cause.
func (u containmentUse) recordReleases(ctx context.Context, releases []ContainmentRelease) {
	for _, r := range releases {
		u.recordEnded(ctx, r.cause)
	}
}

// ContainmentTest is a check or saving throw of the containment rules as d20 resolved it. Variants:
// ContainmentTestRolled, ContainmentTestForced, ContainmentTestNotRequired.
type ContainmentTest interface {
	isContainmentTest()
	passed() bool
}

// ContainmentTestRolled: the test was rolled.
type ContainmentTestRolled struct{ result d20.CheckResult }

// Result returns the resolved test.
func (t ContainmentTestRolled) Result() d20.CheckResult { return t.result }

// ContainmentTestForced: the test failed without a die, because the creature chose to fail it or an effect
// made it fail automatically.
type ContainmentTestForced struct{ result d20.ForcedFailureResult }

// Result returns the forced failure.
func (t ContainmentTestForced) Result() d20.ForcedFailureResult { return t.result }

// ContainmentTestNotRequired: the entry needs no test, so none was made.
type ContainmentTestNotRequired struct{}

func (ContainmentTestRolled) isContainmentTest()      {}
func (ContainmentTestForced) isContainmentTest()      {}
func (ContainmentTestNotRequired) isContainmentTest() {}

func (t ContainmentTestRolled) passed() bool    { return t.result.Outcome() == d20.Success }
func (ContainmentTestForced) passed() bool      { return false }
func (ContainmentTestNotRequired) passed() bool { return false }

// containmentTest makes one test of kind against dc through the gear flow: a RollAttempt resolves its roll mode once
// and rolls that many d20s, a forced failure draws no die.
func (u attackFlow) containmentTest(kind d20.Kind, attempt any, dc d20.DifficultyClass) (ContainmentTest, error) {
	t, err := u.gearTest(kind, attempt, dc)
	if err != nil {
		return nil, err
	}
	switch v := t.(type) {
	case GearTestRolled:
		return ContainmentTestRolled(v), nil
	case GearTestForced:
		return ContainmentTestForced(v), nil
	default:
		panic(unreachableError("gear test", t))
	}
}

// conditionsProblem compares the keys of have with the creatures the call affects: a missing one is
// ContainmentProblemMissingOccupantConditions (checked first), an extra one
// ContainmentProblemUnknownOccupant. It returns 0 when they match.
func conditionsProblem(have map[CombatantID]ConditionSet, affected []CombatantID) ContainmentProblem {
	want := make(map[CombatantID]struct{}, len(affected))
	for _, id := range affected {
		if _, ok := have[id]; !ok {
			return ContainmentProblemMissingOccupantConditions
		}
		want[id] = struct{}{}
	}
	for id := range have {
		if _, ok := want[id]; !ok {
			return ContainmentProblemUnknownOccupant
		}
	}
	return 0
}

// copyConditions returns a copy of m.
func copyConditions(m map[CombatantID]ConditionSet) map[CombatantID]ConditionSet {
	return maps.Clone(m)
}
