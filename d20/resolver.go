package d20

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Roller is the dice source Resolver needs: one synchronous method. Any
// dice.Roller satisfies it, as does a hand-written fake. Resolver wraps every
// call with dice.Roll, which recovers a panic and rejects a result that
// answers a different expression, so an implementation need not defend itself.
//
// A Resolver shared across goroutines calls Roll from all of them at once, so
// the Roller MUST then be safe for concurrent use. A Roller used by one
// goroutine at a time (dice.SeededRoller is one) needs no locking.
type Roller interface {
	Roll(dice.Expression) (dice.Result, error)
}

// Resolver is the use case that resolves one D20 Test: it derives the roll
// mode from the attempt's circumstances, makes exactly one roller request
// sized by that mode (a 1d20, or a 2d20 under Advantage or Disadvantage; none
// for a failure without a roll), hands the faces to the rules functions and
// returns the [Result]. It holds no rules logic and no state beyond its
// injected ports. A Resolver is safe for concurrent use only if every member it
// shares across calls is: the Roller, the telemetry Logger, Tracer and Meter
// (with their spans and instruments) and the clock. The Resolver itself never
// mutates after [NewResolver] returns.
//
// Create it with [NewResolver]. The zero value is not usable.
type Resolver struct {
	roller    Roller
	tel       telemetry.Telemetry
	clock     func() time.Time
	resolved  telemetry.Counter
	duration  telemetry.Histogram
	modifiers telemetry.Histogram
	natural   telemetry.Histogram
	errors    telemetry.Counter
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithTelemetry sets the telemetry the Resolver emits to. Nil members of t
// default to no-ops. Without this option the Resolver is silent.
//
// The members of t MUST be safe for concurrent use if the Resolver is shared
// across goroutines; see the telemetry port documentation.
func WithTelemetry(t telemetry.Telemetry) Option {
	return func(r *Resolver) { r.tel = t }
}

// WithClock sets the clock that times calls for the duration histogram; the
// default is time.Now. A nil clock is ignored.
//
// The clock MUST be safe for concurrent use if the Resolver is shared across
// goroutines.
func WithClock(now func() time.Time) Option {
	return func(r *Resolver) {
		if now != nil {
			r.clock = now
		}
	}
}

// NewResolver returns a Resolver that rolls through roller. The telemetry
// instruments are created once, here.
//
// It returns an *InvalidError with ReasonMissingRoller when roller is nil.
func NewResolver(roller Roller, opts ...Option) (Resolver, error) {
	if roller == nil {
		return Resolver{}, invalid(ReasonMissingRoller, "")
	}
	r := Resolver{roller: roller, clock: time.Now}
	for _, opt := range opts {
		opt(&r)
	}
	r.tel = r.tel.WithDefaults()
	m := r.tel.Meter
	r.resolved = m.Counter(telemetry.Descriptor{Name: MetricResolved, Unit: "{test}", Description: "Resolved D20 Tests, by outcome."})
	r.duration = m.Histogram(telemetry.Descriptor{Name: MetricDuration, Unit: "s", Description: "Duration of resolving one D20 Test."})
	r.modifiers = m.Histogram(telemetry.Descriptor{Name: MetricModifiers, Unit: "{modifier}", Description: "Modifiers in one D20 Test roll."})
	r.natural = m.Histogram(telemetry.Descriptor{Name: MetricNatural, Unit: "{face}", Description: "The d20 face a rolled D20 Test used, 1 to 20."})
	r.errors = m.Counter(telemetry.Descriptor{Name: MetricErrors, Unit: "{error}", Description: "Dice roller failures leaving the use case."})
	return r, nil
}

// Resolve resolves cmd and returns the result, which is never nil without an
// error. A failed check, a missed attack and a failure without a roll are
// normal results, not errors.
//
// It returns an *InvalidError (matching srderr.ErrInvalidInput) before any
// signal is emitted when the Resolver is the zero value (ReasonMissingRoller), cmd is nil or not a command variant, its ability
// label is not valid, or its attempt is nil or not a variant. It returns a
// *srderr.PortError for a roller that fails, panics (matching
// srderr.ErrPortPanic) or answers a different expression (matching
// srderr.ErrContractViolation); the error is recorded once on the span and the
// errors counter. ctx is passed to the telemetry only. Cancelling ctx neither aborts nor alters the result or metrics.
func (r Resolver) Resolve(ctx context.Context, cmd Command) (Result, error) {
	if r.roller == nil {
		return nil, invalid(ReasonMissingRoller, "zero Resolver; use NewResolver")
	}
	switch cmd.(type) {
	case AbilityCheckCommand, SavingThrowCommand, AttackRollCommand:
	default:
		return nil, invalid(ReasonUnknownCommand, "command is nil, a pointer or a foreign type")
	}
	if err := cmd.validate(); err != nil {
		return nil, err
	}
	kind := telemetry.String(AttrKind, cmd.kind().String())
	attrs := []telemetry.Attr{kind, telemetry.String(AttrAbility, cmd.abilityLabel().String())}

	ctx, span := r.tel.Tracer.Start(ctx, SpanResolve, attrs...)
	defer span.End()
	timer := telemetry.StartTimer(r.clock, r.duration)
	defer timer.Stop(ctx, kind)

	res, err := r.resolve(cmd)
	if err != nil {
		r.recordFailure(ctx, span, kind, err)
		return nil, err
	}
	r.recordSuccess(ctx, span, kind, attrs, res)
	return res, nil
}

func (r Resolver) resolve(cmd Command) (Result, error) {
	switch c := cmd.(type) {
	case AbilityCheckCommand:
		switch a := c.Attempt.(type) {
		case RollAttempt:
			return r.rolledCheck(c.Ability, AbilityCheck, a, c.Target)
		case AutomaticFailure:
			return forced(c.Ability, AbilityCheck, Automatic, c.Target)
		}
	case SavingThrowCommand:
		switch a := c.Attempt.(type) {
		case RollAttempt:
			return r.rolledCheck(c.Ability, SavingThrow, a, c.Target)
		case ChooseToFail:
			return forced(c.Ability, SavingThrow, ChosenToFail, c.Target)
		case AutomaticFailure:
			return forced(c.Ability, SavingThrow, Automatic, c.Target)
		}
	case AttackRollCommand:
		roll, err := r.roll(c.Attempt)
		if err != nil {
			return nil, err
		}
		attack, err := ResolveAttack(roll, c.Attempt.Modifiers, c.Target, DefaultCriticalRange)
		if err != nil {
			return nil, err
		}
		return NewRolledAttack(c.Ability, attack)
	}
	return nil, invalid(ReasonUnknownCommand, "attempt is a pointer or a foreign type")
}

func (r Resolver) rolledCheck(a ability.Ability, kind Kind, attempt RollAttempt, target DifficultyClass) (Result, error) {
	roll, err := r.roll(attempt)
	if err != nil {
		return nil, err
	}
	check, err := ResolveCheck(kind, roll, attempt.Modifiers, target)
	if err != nil {
		return nil, err
	}
	return NewRolledCheck(a, check)
}

func forced(a ability.Ability, kind Kind, reason ForcedFailureReason, target DifficultyClass) (Result, error) {
	f, err := FailWithoutRoll(kind, reason)
	if err != nil {
		return nil, err
	}
	return NewForcedFailure(a, f, target)
}

// roll makes the one roller request of an attempt: the mode resolved once from
// the circumstances, one expression sized by that mode.
func (r Resolver) roll(a RollAttempt) (Roll, error) {
	mode := modifier.Resolve(a.Circumstances...)
	res, err := dice.Roll(r.roller, dice.MustExpression(mode.DiceCount(), dice.D20(), 0))
	if err != nil {
		return Roll{}, err
	}
	faces := make([]int, 0, mode.DiceCount())
	for _, dr := range res.Rolls() {
		faces = append(faces, dr.Value())
	}
	roll, err := NewRoll(mode, faces)
	if err != nil {
		// dice.Roll vouches for the count and faces; reaching here means the
		// contract check above has a gap.
		return Roll{}, srderr.ContractViolation(dice.Port, "roll", err.Error())
	}
	return roll, nil
}

func (r Resolver) recordSuccess(ctx context.Context, span telemetry.Span, kind telemetry.Attr, attrs []telemetry.Attr, res Result) {
	var more []telemetry.Attr
	switch v := res.(type) {
	case RolledCheck:
		more = append(more, telemetry.String(AttrRollMode, v.check.roll.mode.String()))
		r.recordRoll(ctx, kind, v.check.modifiers, v.check.roll)
	case RolledAttack:
		more = append(more, telemetry.String(AttrRollMode, v.attack.roll.mode.String()))
		r.recordRoll(ctx, kind, v.attack.modifiers, v.attack.roll)
	case ForcedFailure:
		more = append(more, telemetry.String(AttrForcedFailureReason, v.failure.reason.String()))
	}
	more = append(more, telemetry.String(telemetry.AttrOutcome, res.Outcome().String()))
	if v, ok := res.(RolledAttack); ok {
		more = append(more, telemetry.Bool(AttrCriticalHit, v.attack.critical))
	}
	span.SetAttributes(more...)
	r.resolved.Add(ctx, 1, slices.Concat(attrs, more)...)
}

func (r Resolver) recordRoll(ctx context.Context, kind telemetry.Attr, m RollModifiers, roll Roll) {
	r.modifiers.Record(ctx, float64(m.Count()), kind)
	r.natural.Record(ctx, float64(roll.natural), kind)
}

func (r Resolver) recordFailure(ctx context.Context, span telemetry.Span, kind telemetry.Attr, err error) {
	failure := telemetry.String(AttrRollerFailure, rollerFailure(err))
	span.SetAttributes(failure)
	span.RecordError(err)
	span.SetStatus(telemetry.StatusError)
	attrs := []telemetry.Attr{kind}
	if t := srderr.ErrorType(err); t != "" {
		attrs = append(attrs, telemetry.String(telemetry.AttrErrorType, t))
	}
	r.errors.Add(ctx, 1, slices.Concat(attrs, []telemetry.Attr{failure})...)
}

// rollerFailure classifies err for the roller_failure attribute: a roller that
// broke its contract is invalid_result, anything else is unavailable.
func rollerFailure(err error) string {
	if errors.Is(err, srderr.ErrContractViolation) {
		return ValueRollerInvalidResult
	}
	return ValueRollerUnavailable
}
