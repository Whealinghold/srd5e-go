package combat

import (
	"context"
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// gearUse is what every gear use case holds: the instrumenter for the shared signals and the
// telemetry for the signals of deployed-and-restraining-gear.md section 7. The zero value is
// silent and usable.
type gearUse struct {
	in  instrumenter
	tel telemetry.Telemetry
}

func newGearUse(opts []Option) gearUse {
	o := newOptions(opts)
	return gearUse{in: newInstrumenter(o.tel, o.clock), tel: o.tel}
}

// recordEntry counts one area-entry outcome of gear of kind.
func (u gearUse) recordEntry(ctx context.Context, kind DeployedGearKind, entry string) {
	u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricGearEntries, Unit: "{entry}",
		Description: "Area-entry outcomes of deployed gear."}).
		Add(ctx, 1, telemetry.String(attrGearKind, kind.String()), telemetry.String(attrGearEntry, entry))
}

// recordHolds records the size of the holds a result carries and counts each hold that ended.
func (u gearUse) recordHolds(ctx context.Context, op combatOperation, holds GearHolds, events []GearEvent) {
	meter := u.tel.WithDefaults().Meter
	meter.Histogram(telemetry.Descriptor{Name: metricGearHoldsInForce, Unit: "{hold}",
		Description: "Holds in force in a result."}).Record(ctx, float64(holds.Len()), telemetry.String(attrOperation, op.Value()))
	ended := meter.Counter(telemetry.Descriptor{Name: metricGearHoldEnded, Unit: "{hold}", Description: "Holds that ended, by kind and cause."})
	for _, e := range events {
		if h, ok := e.(GearHoldEnded); ok {
			ended.Add(ctx, 1, telemetry.String(attrGearKind, h.kind.String()), telemetry.String(attrGearEndCause, h.cause.String()))
		}
	}
}

// gearProblem returns the problem of a *GearError and whether err is one.
func gearProblem(err error) (GearProblem, bool) {
	var ge *GearError
	if errors.As(err, &ge) {
		return ge.Problem, true
	}
	return 0, false
}

// GearTest is a check or saving throw of the gear rules as d20 resolved it. The held result's
// outcome is Outcome. Variants: GearTestRolled, GearTestForced.
type GearTest interface {
	isGearTest()
	Outcome() d20.Outcome
}

// GearTestRolled: the test was rolled.
type GearTestRolled struct{ result d20.CheckResult }

// Result returns the resolved check.
func (t GearTestRolled) Result() d20.CheckResult { return t.result }

// Outcome returns the outcome of the check.
func (t GearTestRolled) Outcome() d20.Outcome { return t.result.Outcome() }

// GearTestForced: the test failed without a die, because the creature chose to fail it or an
// effect made it fail automatically.
type GearTestForced struct{ result d20.ForcedFailureResult }

// Result returns the forced failure.
func (t GearTestForced) Result() d20.ForcedFailureResult { return t.result }

// Outcome returns d20.Failure.
func (GearTestForced) Outcome() d20.Outcome { return d20.Failure }

func (GearTestRolled) isGearTest() {}
func (GearTestForced) isGearTest() {}

// gearTest makes one test of kind against dc: a RollAttempt resolves its roll mode once, rolls
// that many d20s and compares with d20.ResolveCheck; a forced failure draws no die. The caller
// has checked that the attempt is a RollAttempt or a forced failure the kind allows.
func (f attackFlow) gearTest(kind d20.Kind, attempt any, dc d20.DifficultyClass) (GearTest, error) {
	reason := d20.Automatic
	switch a := attempt.(type) {
	case d20.RollAttempt:
		roll, err := f.rollD20(modifier.Resolve(a.Circumstances...))
		if err != nil {
			return nil, err
		}
		res, err := d20.ResolveCheck(kind, roll, a.Modifiers, dc)
		return GearTestRolled{res}, err
	case d20.ChooseToFail:
		reason = d20.ChosenToFail
	}
	res, err := d20.FailWithoutRoll(kind, reason)
	return GearTestForced{res}, err
}

// rollTrapDamage rolls the 1d4 of a failed Hunting Trap save.
func (f attackFlow) rollTrapDamage() (int, error) {
	res, err := dice.Roll(f.roller, dice.MustExpression(1, dice.D4(), 0))
	if err != nil {
		return 0, err
	}
	return res.Total(), nil
}

// checkSave returns a copy of a for a command: a RollAttempt (circumstances copied),
// ChooseToFail or AutomaticFailure, else false.
func checkSave(a d20.SaveAttempt) (d20.SaveAttempt, bool) {
	switch v := a.(type) {
	case d20.RollAttempt:
		v.Circumstances = slices.Clone(v.Circumstances)
		return v, true
	case d20.ChooseToFail, d20.AutomaticFailure:
		return v, true
	}
	return nil, false
}

// checkCheck returns a copy of a for a command: a RollAttempt (circumstances copied) or
// AutomaticFailure, else false.
func checkCheck(a d20.CheckAttempt) (d20.CheckAttempt, bool) {
	switch v := a.(type) {
	case d20.RollAttempt:
		v.Circumstances = slices.Clone(v.Circumstances)
		return v, true
	case d20.AutomaticFailure:
		return v, true
	}
	return nil, false
}

// attemptOverCap returns a LimitExceededError when a is a d20.RollAttempt of more than 256 circumstances
// (plan P10), and nil otherwise.
func attemptOverCap(a any) error {
	if r, ok := a.(d20.RollAttempt); ok {
		return overCap("attempt circumstances", len(r.Circumstances), maxInputList)
	}
	return nil
}
