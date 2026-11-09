package character

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// defaultRetryAfter is the wait after a failed Influence check when the GM sets
// none (SRD 5.2.1, "Influence [Action]", p. 184).
const defaultRetryAfter = 24 * time.Hour

// InfluenceAttempt is how a hesitant monster's Influence check is made: the
// closed set InfluenceRoll and InfluenceAutomaticFailure.
//
//sumtype:decl
type InfluenceAttempt interface{ isInfluenceAttempt() }

// InfluenceAutomaticFailure: the check cannot succeed (for example it needs
// sight or hearing while Blinded or Deafened, SRD 5.2.1, pp. 177, 181). The
// caller decides; nothing is rolled and the attitude is not applied.
type InfluenceAutomaticFailure struct{}

// InfluenceRoll holds the caller's modifiers and circumstances for a rolled
// Influence check: an ability modifier per ability, the optional Proficiency
// Bonus, other modifiers and extra circumstances (Charmed, Help, Perfume, ...).
// Build it with NewInfluenceRoll; it copies every collection. The zero value
// holds no ability modifier. It is not comparable with ==; use Equal.
type InfluenceRoll struct {
	abilityModifiers map[ability.Ability]modifier.Modifier
	prof             modifier.Modifier
	hasProf          bool
	others           []modifier.Modifier
	circumstances    []modifier.RollCircumstance
}

func (InfluenceRoll) isInfluenceAttempt()             {}
func (InfluenceAutomaticFailure) isInfluenceAttempt() {}

// RollOption configures NewInfluenceRoll.
type RollOption func(*InfluenceRoll)

// WithProficiencyBonus adds the Proficiency Bonus to the check.
func WithProficiencyBonus(b modifier.Modifier) RollOption {
	return func(r *InfluenceRoll) { r.prof, r.hasProf = b, true }
}

// WithOtherModifiers appends other modifiers, in order.
func WithOtherModifiers(ms ...modifier.Modifier) RollOption {
	return func(r *InfluenceRoll) { r.others = append(r.others, ms...) }
}

// WithCircumstances appends roll circumstances, in order, that follow the
// attitude's own when the roll mode is resolved.
func WithCircumstances(cs ...modifier.RollCircumstance) RollOption {
	return func(r *InfluenceRoll) { r.circumstances = append(r.circumstances, cs...) }
}

// NewInfluenceRoll returns an InfluenceRoll over abilityModifiers (copied). It
// does not fail; NewInfluenceHesitantCommand checks that the approach's ability
// has an entry.
func NewInfluenceRoll(abilityModifiers map[ability.Ability]modifier.Modifier, opts ...RollOption) InfluenceRoll {
	r := InfluenceRoll{abilityModifiers: maps.Clone(abilityModifiers)}
	for _, o := range opts {
		o(&r)
	}
	r.others = slices.Clone(r.others)
	r.circumstances = slices.Clone(r.circumstances)
	return r
}

// AbilityModifiers returns a copy of the ability modifiers.
func (r InfluenceRoll) AbilityModifiers() map[ability.Ability]modifier.Modifier {
	return maps.Clone(r.abilityModifiers)
}

// ProficiencyBonus returns the Proficiency Bonus and whether one was set.
func (r InfluenceRoll) ProficiencyBonus() (modifier.Modifier, bool) { return r.prof, r.hasProf }

// OtherModifiers returns a copy of the other modifiers.
func (r InfluenceRoll) OtherModifiers() []modifier.Modifier { return slices.Clone(r.others) }

// Circumstances returns a copy of the caller's circumstances.
func (r InfluenceRoll) Circumstances() []modifier.RollCircumstance {
	return slices.Clone(r.circumstances)
}

// Equal reports whether r and o hold the same values in the same order.
func (r InfluenceRoll) Equal(o InfluenceRoll) bool {
	return r.prof == o.prof && r.hasProf == o.hasProf &&
		maps.Equal(r.abilityModifiers, o.abilityModifiers) &&
		slices.Equal(r.others, o.others) && slices.Equal(r.circumstances, o.circumstances)
}

// InfluenceCommand is the GM's determination of the monster's response to the
// urging (SRD 5.2.1, "Influence [Action]", p. 184): the closed set
// InfluenceWillingCommand, InfluenceUnwillingCommand and
// InfluenceHesitantCommand. The determination is input, never inferred.
//
//sumtype:decl
type InfluenceCommand interface{ isInfluenceCommand() }

// InfluenceWillingCommand: the monster is willing; no check is needed.
type InfluenceWillingCommand struct{}

// InfluenceUnwillingCommand: the monster is unwilling; no check is needed.
type InfluenceUnwillingCommand struct{}

// InfluenceHesitantCommand: the monster is hesitant and an ability check
// decides. Build it with NewInfluenceHesitantCommand; the zero value is invalid.
// It is not comparable with ==; use Equal.
type InfluenceHesitantCommand struct {
	approach   InfluenceApproach
	target     InfluenceTarget
	attempt    InfluenceAttempt
	gmDC       d20.DifficultyClass
	hasGMDC    bool
	retryAfter time.Duration
	hasRetry   bool
}

func (InfluenceWillingCommand) isInfluenceCommand()   {}
func (InfluenceUnwillingCommand) isInfluenceCommand() {}
func (InfluenceHesitantCommand) isInfluenceCommand()  {}

// HesitantOption configures NewInfluenceHesitantCommand.
type HesitantOption func(*InfluenceHesitantCommand)

// WithGMDC sets the Difficulty Class instead of the default.
func WithGMDC(dc d20.DifficultyClass) HesitantOption {
	return func(c *InfluenceHesitantCommand) { c.gmDC, c.hasGMDC = dc, true }
}

// WithGMRetryAfter sets the wait after a failed check instead of 24 hours. Zero
// is accepted.
func WithGMRetryAfter(d time.Duration) HesitantOption {
	return func(c *InfluenceHesitantCommand) { c.retryAfter, c.hasRetry = d, true }
}

// NewInfluenceHesitantCommand returns the command to urge target by approach
// with attempt.
//
// It returns an error matching srderr.ErrInvalidInput, and the zero command, for
// an invalid approach or target, a nil or foreign attempt, a negative GM wait, or
// an InfluenceRoll with no ability modifier for the approach's ability. It never
// panics.
func NewInfluenceHesitantCommand(approach InfluenceApproach, target InfluenceTarget, attempt InfluenceAttempt, opts ...HesitantOption) (InfluenceHesitantCommand, error) {
	c := InfluenceHesitantCommand{approach: approach, target: target, attempt: attempt}
	for _, o := range opts {
		o(&c)
	}
	switch a := attempt.(type) {
	case InfluenceAutomaticFailure:
	case InfluenceRoll:
		if _, ok := a.abilityModifiers[approach.Ability()]; !ok {
			return InfluenceHesitantCommand{}, invalidInput("influence command: no ability modifier for the approach's ability")
		}
	default:
		return InfluenceHesitantCommand{}, invalidInput("influence command: nil or foreign attempt")
	}
	switch {
	case !approach.Valid():
		return InfluenceHesitantCommand{}, invalidInput("influence command: invalid approach")
	case !target.Valid():
		return InfluenceHesitantCommand{}, invalidInput("influence command: invalid target")
	case c.hasRetry && c.retryAfter < 0:
		return InfluenceHesitantCommand{}, invalidInput("influence command: negative GM wait")
	}
	return c, nil
}

// MustInfluenceHesitantCommand is NewInfluenceHesitantCommand for test literals;
// it panics with the error NewInfluenceHesitantCommand returns.
func MustInfluenceHesitantCommand(approach InfluenceApproach, target InfluenceTarget, attempt InfluenceAttempt, opts ...HesitantOption) InfluenceHesitantCommand {
	c, err := NewInfluenceHesitantCommand(approach, target, attempt, opts...)
	if err != nil {
		panic(err)
	}
	return c
}

// Approach returns the approach.
func (c InfluenceHesitantCommand) Approach() InfluenceApproach { return c.approach }

// Target returns the target.
func (c InfluenceHesitantCommand) Target() InfluenceTarget { return c.target }

// Attempt returns the attempt.
func (c InfluenceHesitantCommand) Attempt() InfluenceAttempt { return c.attempt }

// GMDC returns the GM-set Difficulty Class and whether one was set.
func (c InfluenceHesitantCommand) GMDC() (d20.DifficultyClass, bool) { return c.gmDC, c.hasGMDC }

// GMRetryAfter returns the GM-set wait and whether one was set.
func (c InfluenceHesitantCommand) GMRetryAfter() (time.Duration, bool) {
	return c.retryAfter, c.hasRetry
}

// Equal reports whether c and o hold the same values.
func (c InfluenceHesitantCommand) Equal(o InfluenceHesitantCommand) bool {
	if c.approach != o.approach || c.target != o.target || c.gmDC != o.gmDC || c.hasGMDC != o.hasGMDC ||
		c.retryAfter != o.retryAfter || c.hasRetry != o.hasRetry {
		return false
	}
	switch a := c.attempt.(type) {
	case InfluenceRoll:
		b, ok := o.attempt.(InfluenceRoll)
		return ok && a.Equal(b)
	case InfluenceAutomaticFailure:
		_, ok := o.attempt.(InfluenceAutomaticFailure)
		return ok
	}
	return o.attempt == nil
}

// ResolveInfluenceResult is the closed outcome of ResolveInfluence:
// ResolveInfluenceFulfilled, ResolveInfluenceRefused,
// ResolveInfluenceCheckSucceeded, ResolveInfluenceCheckFailed or
// ResolveInfluenceCheckForcedFailure. None is an error, and none changes the
// monster's attitude: the SRD states no such rule.
//
//sumtype:decl
type ResolveInfluenceResult interface{ isResolveInfluenceResult() }

// ResolveInfluenceFulfilled: the willing monster fulfils the request.
type ResolveInfluenceFulfilled struct{}

// ResolveInfluenceRefused: the unwilling monster does not comply.
type ResolveInfluenceRefused struct{}

// influenceCheck is what the three check results share.
type influenceCheck struct {
	approach InfluenceApproach
	attitude Attitude
	dc       d20.DifficultyClass
	basis    DCBasis
}

// ResolveInfluenceCheckSucceeded: the check reached the DC and the monster does
// as urged. Compare with Equal.
type ResolveInfluenceCheckSucceeded struct {
	influenceCheck
	check d20.CheckResult
}

// ResolveInfluenceCheckFailed: the check missed the DC; the same urging may not
// be tried again before RetryAfter. Compare with Equal.
type ResolveInfluenceCheckFailed struct {
	influenceCheck
	check      d20.CheckResult
	retryAfter time.Duration
}

// ResolveInfluenceCheckForcedFailure: the attempt was an InfluenceAutomaticFailure,
// so nothing was rolled. Compare with ==.
type ResolveInfluenceCheckForcedFailure struct {
	influenceCheck
	check      d20.ForcedFailureResult
	retryAfter time.Duration
}

func (ResolveInfluenceFulfilled) isResolveInfluenceResult()          {}
func (ResolveInfluenceRefused) isResolveInfluenceResult()            {}
func (ResolveInfluenceCheckSucceeded) isResolveInfluenceResult()     {}
func (ResolveInfluenceCheckFailed) isResolveInfluenceResult()        {}
func (ResolveInfluenceCheckForcedFailure) isResolveInfluenceResult() {}

// Approach returns the approach of the urging.
func (r influenceCheck) Approach() InfluenceApproach { return r.approach }

// Attitude returns the attitude the urging was made against.
func (r influenceCheck) Attitude() Attitude { return r.attitude }

// DC returns the Difficulty Class used.
func (r influenceCheck) DC() d20.DifficultyClass { return r.dc }

// DCBasis returns how the Difficulty Class was chosen.
func (r influenceCheck) DCBasis() DCBasis { return r.basis }

// Check returns the rolled check.
func (r ResolveInfluenceCheckSucceeded) Check() d20.CheckResult { return r.check }

// Check returns the rolled check.
func (r ResolveInfluenceCheckFailed) Check() d20.CheckResult { return r.check }

// RetryAfter returns the wait before the same urging may be tried again.
func (r ResolveInfluenceCheckFailed) RetryAfter() time.Duration { return r.retryAfter }

// Check returns the forced failure.
func (r ResolveInfluenceCheckForcedFailure) Check() d20.ForcedFailureResult { return r.check }

// RetryAfter returns the wait before the same urging may be tried again.
func (r ResolveInfluenceCheckForcedFailure) RetryAfter() time.Duration { return r.retryAfter }

// Equal reports whether r and o hold the same values.
func (r ResolveInfluenceCheckSucceeded) Equal(o ResolveInfluenceCheckSucceeded) bool {
	return r.influenceCheck == o.influenceCheck && r.check.Equal(o.check)
}

// Equal reports whether r and o hold the same values.
func (r ResolveInfluenceCheckFailed) Equal(o ResolveInfluenceCheckFailed) bool {
	return r.influenceCheck == o.influenceCheck && r.check.Equal(o.check) && r.retryAfter == o.retryAfter
}

// InfluenceResolver is the use case that resolves the effect of the Influence
// action (SRD 5.2.1, "Influence [Action]", p. 184). It rolls the d20s through
// its Roller and hands them to the pure functions of package d20; it adds no
// dice logic and calls no other use case (not d20.Resolver). Spending the action
// belongs to the host. Create it with NewInfluenceResolver; the zero value is
// not usable. It is safe for concurrent use if its roller, telemetry and clock are.
type InfluenceResolver struct {
	roller   Roller
	tel      telemetry.Telemetry
	clock    func() time.Time
	resolved telemetry.Counter
	errs     telemetry.Counter
	duration telemetry.Histogram
}

// NewInfluenceResolver returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored.
//
// It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewInfluenceResolver(roller Roller, opts ...Option) (InfluenceResolver, error) {
	if roller == nil {
		return InfluenceResolver{}, invalidInput("resolve influence without a dice roller")
	}
	o := newOptions(opts)
	m := o.tel.Meter
	return InfluenceResolver{
		roller:   roller,
		tel:      o.tel,
		clock:    o.clock,
		resolved: m.Counter(telemetry.Descriptor{Name: MetricInfluenceResolved, Unit: "{influence}", Description: "Completed Influence calls, by response and outcome."}),
		errs:     m.Counter(telemetry.Descriptor{Name: MetricInfluenceErrors, Unit: "{error}", Description: "Port failures leaving ResolveInfluence, by error type."}),
		duration: m.Histogram(telemetry.Descriptor{Name: MetricInfluenceDuration, Unit: "s", Description: "Duration of one ResolveInfluence call."}),
	}, nil
}

// ResolveInfluence resolves the Influence action for the GM's determination cmd.
// A willing monster fulfils the request and an unwilling one refuses, with no
// roll. A hesitant monster gets one ability check: a rolled attempt makes one
// request for 1d20 (2d20 under Advantage or Disadvantage) whose mode is the
// attitude's circumstances followed by the attempt's, and reaches the DC or not
// (a failed check carries RetryAfter, 24 hours unless the GM set one); an
// InfluenceAutomaticFailure rolls nothing. All five results are normal outcomes.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero use case or a nil, foreign or zero command, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled. It returns the
// *srderr.PortError of the dice port (error type "dice_roller_error") with a
// nil result when the roller fails, panics or breaks its contract; the failure
// is recorded once and never logged. No log event is emitted.
func (u InfluenceResolver) ResolveInfluence(ctx context.Context, cmd InfluenceCommand) (ResolveInfluenceResult, error) {
	if u.roller == nil {
		return nil, invalidInput("resolve influence: zero use case")
	}
	hesitant, isHesitant := cmd.(InfluenceHesitantCommand)
	switch cmd.(type) {
	case InfluenceWillingCommand, InfluenceUnwillingCommand:
	case InfluenceHesitantCommand:
		if !hesitant.approach.Valid() {
			return nil, invalidInput("resolve influence: zero hesitant command")
		}
	default:
		return nil, invalidInput("resolve influence: nil or foreign command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var start []telemetry.Attr
	if isHesitant {
		start = []telemetry.Attr{
			telemetry.String(AttrInfluenceApproach, hesitant.approach.String()),
			telemetry.String(AttrInfluenceAttitude, hesitant.target.attitude.String()),
		}
	}
	ctx, span := u.tel.Tracer.Start(ctx, SpanResolveInfluence, start...)
	defer span.End()
	timer := telemetry.StartTimer(u.clock, u.duration)
	defer timer.Stop(ctx) // runs before span.End

	switch cmd.(type) {
	case InfluenceWillingCommand:
		u.record(ctx, span, start, ValueInfluenceFulfilled, telemetry.OutcomeSuccess)
		return ResolveInfluenceFulfilled{}, nil
	case InfluenceUnwillingCommand:
		u.record(ctx, span, start, ValueInfluenceRefused, telemetry.OutcomeFailure)
		return ResolveInfluenceRefused{}, nil
	}
	return u.resolveHesitant(ctx, span, start, hesitant)
}

func (u InfluenceResolver) resolveHesitant(ctx context.Context, span telemetry.Span, start []telemetry.Attr, cmd InfluenceHesitantCommand) (ResolveInfluenceResult, error) {
	dc, basis := DefaultDC(cmd.target.intelligence), DefaultRule
	if cmd.hasGMDC {
		dc, basis = cmd.gmDC, GMSet
	}
	wait := defaultRetryAfter
	if cmd.hasRetry {
		wait = cmd.retryAfter
	}
	base := influenceCheck{approach: cmd.approach, attitude: cmd.target.attitude, dc: dc, basis: basis}

	attempt, isRoll := cmd.attempt.(InfluenceRoll)
	if !isRoll {
		// unreachable error: an ability check may fail automatically
		forced, err := d20.FailWithoutRoll(d20.AbilityCheck, d20.Automatic)
		if err != nil {
			return nil, err
		}
		u.record(ctx, span, start, ValueInfluenceCheckFailed, telemetry.OutcomeFailure)
		return ResolveInfluenceCheckForcedFailure{influenceCheck: base, check: forced, retryAfter: wait}, nil
	}

	roll, err := u.roll(attempt, cmd.target.attitude)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.StatusError)
		u.errs.Add(ctx, 1, telemetry.String(telemetry.AttrErrorType, srderr.ErrorType(err)))
		return nil, err
	}
	mods := d20.NewRollModifiers(attempt.abilityModifiers[cmd.approach.Ability()])
	if attempt.hasProf {
		mods = mods.WithProficiencyBonus(attempt.prof)
	}
	mods = mods.WithOthers(attempt.others...)
	check, err := d20.ResolveCheck(d20.AbilityCheck, roll, mods, dc)
	if err != nil { // unreachable: the roll is built and the kind is an ability check
		return nil, err
	}
	mode := telemetry.String(d20.AttrRollMode, roll.Mode().String())
	if check.Outcome() == d20.Success {
		u.record(ctx, span, start, ValueInfluenceCheckSucceeded, telemetry.OutcomeSuccess, mode)
		return ResolveInfluenceCheckSucceeded{influenceCheck: base, check: check}, nil
	}
	u.record(ctx, span, start, ValueInfluenceCheckFailed, telemetry.OutcomeFailure, mode)
	return ResolveInfluenceCheckFailed{influenceCheck: base, check: check, retryAfter: wait}, nil
}

// roll makes the one request, sized by the roll mode of the attitude's
// circumstances followed by the attempt's, and builds the d20.Roll. A roller
// failure, panic or broken contract comes back as a *srderr.PortError.
func (u InfluenceResolver) roll(a InfluenceRoll, att Attitude) (d20.Roll, error) {
	mode := modifier.Resolve(append(AttitudeCircumstances(att), a.circumstances...)...)
	expr, err := dice.NewExpression(mode.DiceCount(), dice.D20(), 0)
	if err != nil { // unreachable: the count is 1 or 2 and the die is a d20
		return d20.Roll{}, err
	}
	res, err := dice.Roll(u.roller, expr)
	if err != nil {
		return d20.Roll{}, err
	}
	faces := make([]int, 0, mode.DiceCount())
	for _, r := range res.Rolls() {
		faces = append(faces, r.Value())
	}
	// An error is unreachable: dice.Roll vouches for the count and faces.
	return d20.NewRoll(mode, faces)
}

// record sets the result attributes on the span and increments the resolved
// counter once. start holds the hesitant attributes (approach, attitude), which
// go on the counter too; extra is the roll mode of a rolled attempt.
func (u InfluenceResolver) record(ctx context.Context, span telemetry.Span, start []telemetry.Attr, response, outcome string, extra ...telemetry.Attr) {
	attrs := append([]telemetry.Attr{telemetry.String(AttrInfluenceResponse, response), telemetry.String(telemetry.AttrOutcome, outcome)}, extra...)
	span.SetAttributes(attrs...)
	u.resolved.Add(ctx, 1, append(attrs, start...)...)
}
