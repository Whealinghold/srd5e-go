package character

import (
	"context"
	"errors"
	"time"

	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// TakeRestCommandProblem is a reason NewTakeRestCommand refused host numbers.
// The zero value is not a problem. It is an error matching
// srderr.ErrInvalidInput.
//
//nolint:errname // an enum of problems, named by the spec
type TakeRestCommandProblem int

// The problems of a TakeRestCommand, in input order.
const (
	// NegativeCurrentHitPoints: the current Hit Points are negative.
	NegativeCurrentHitPoints TakeRestCommandProblem = iota + 1 //nolint:errname // a member of the problem enum
	// ExhaustionLevelOutOfRange: the Exhaustion level is outside 0 to 6.
	ExhaustionLevelOutOfRange //nolint:errname // a member of the problem enum
)

// String returns the snake_case name of p, or "unknown".
func (p TakeRestCommandProblem) String() string {
	switch p {
	case NegativeCurrentHitPoints:
		return "negative_current_hit_points"
	case ExhaustionLevelOutOfRange:
		return "exhaustion_level_out_of_range"
	}
	return "unknown"
}

// Error renders the problem. The text is not a contract.
func (p TakeRestCommandProblem) Error() string {
	return "character: invalid take rest command: " + p.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (TakeRestCommandProblem) Unwrap() error { return srderr.ErrInvalidInput }

// TakeRestCommand is the input of RestTaker.TakeRest: the Character, what the
// host reports about the rest, and the values the rules need. CurrentHitPoints,
// ExhaustionLevel and the LongRestSpacing are those at the START of the rest, not
// when it is reported: a Long Rest stopped by damage that leaves the creature at
// 0 Hit Points is reported with the Hit Points it started with, and is not
// refused. ExhaustionLevel is also the level a Long Rest lowers, the same by
// contract. Only a TakeRestApplied whose RestApplied has BenefitsOf LongRest and
// Source RestFinished counts as a finished Long Rest: the host updates its
// LongRestSpacing only then. The host owns time and reports only periods that
// meet the sleep and activity limits (see RestEnd). Build it with
// NewTakeRestCommand; the zero command is invalid. TakeRest does not alter it.
type TakeRestCommand struct {
	id               Identifier
	report           RestReport
	currentHitPoints int
	exhaustionLevel  int
	spacing          LongRestSpacing
	spent            SpentHitDice
}

// NewTakeRestCommand returns the command for the rest in report. spacing may be
// nil, meaning NoLongRestFinishedYet.
//
// It returns an error matching srderr.ErrInvalidInput for a zero id or report,
// and otherwise an errors.Join of every TakeRestCommandProblem, once each, in
// input order, each matching srderr.ErrInvalidInput. It never panics.
func NewTakeRestCommand(id Identifier, report RestReport, currentHitPoints, exhaustionLevel int, spacing LongRestSpacing, spent SpentHitDice) (TakeRestCommand, error) {
	if id.IsZero() || report.IsZero() {
		return TakeRestCommand{}, invalidInput("take rest command with a zero id or report")
	}
	var problems []error
	if currentHitPoints < 0 {
		problems = append(problems, NegativeCurrentHitPoints)
	}
	if exhaustionLevel < 0 || exhaustionLevel > maxExhaustionLevel {
		problems = append(problems, ExhaustionLevelOutOfRange)
	}
	if len(problems) > 0 {
		return TakeRestCommand{}, errors.Join(problems...)
	}
	return TakeRestCommand{id: id, report: report, currentHitPoints: currentHitPoints, exhaustionLevel: exhaustionLevel, spacing: spacing, spent: spent}, nil
}

// MustTakeRestCommand is NewTakeRestCommand for known-valid input. It panics with
// the error NewTakeRestCommand returns, and only then.
func MustTakeRestCommand(id Identifier, report RestReport, currentHitPoints, exhaustionLevel int, spacing LongRestSpacing, spent SpentHitDice) TakeRestCommand {
	c, err := NewTakeRestCommand(id, report, currentHitPoints, exhaustionLevel, spacing, spent)
	if err != nil {
		panic(err)
	}
	return c
}

// TakeRestResult is the closed set of outcomes of RestTaker.TakeRest. Variants:
// TakeRestApplied, TakeRestAppliedAfterInterruption, TakeRestNotStarted,
// TakeRestIncomplete, TakeRestStopped, TakeRestRejected.
type TakeRestResult interface{ isTakeRestResult() }

// TakeRestApplied: the rest finished; the host applies the effects of Applied.
type TakeRestApplied struct {
	spent   SpentHitDice
	applied RestApplied
}

// SpentHitDice returns the spent Hit Point Dice after the rest.
func (r TakeRestApplied) SpentHitDice() SpentHitDice { return r.spent }

// Applied returns the RestApplied event, whose Effects are the effects.
func (r TakeRestApplied) Applied() RestApplied { return r.applied }

// TakeRestAppliedAfterInterruption: a Long Rest was interrupted but earned the
// benefits of a Short Rest.
type TakeRestAppliedAfterInterruption struct {
	spent       SpentHitDice
	applied     RestApplied
	interrupted RestInterrupted
}

// SpentHitDice returns the spent Hit Point Dice after the rest.
func (r TakeRestAppliedAfterInterruption) SpentHitDice() SpentHitDice { return r.spent }

// Applied returns the RestApplied event, whose Effects are the effects.
func (r TakeRestAppliedAfterInterruption) Applied() RestApplied { return r.applied }

// Interrupted returns the RestInterrupted event.
func (r TakeRestAppliedAfterInterruption) Interrupted() RestInterrupted { return r.interrupted }

// TakeRestNotStarted: a blocker stopped the rest from starting; nothing changed.
type TakeRestNotStarted struct{ decision RestMayNotStart }

// Blockers returns a new slice of every blocker, in the order of CheckRestStart.
func (r TakeRestNotStarted) Blockers() []RestStartBlocker { return r.decision.Blockers() }

// TakeRestIncomplete: the rest ran its course but was too short; nothing changed.
type TakeRestIncomplete struct {
	spent            SpentHitDice
	hoursStillNeeded int
}

// SpentHitDice returns the spent Hit Point Dice, unchanged.
func (r TakeRestIncomplete) SpentHitDice() SpentHitDice { return r.spent }

// HoursStillNeeded returns how many more hours the rest needed, at least 1.
func (r TakeRestIncomplete) HoursStillNeeded() int { return r.hoursStillNeeded }

// TakeRestStopped: the rest was interrupted and granted nothing.
type TakeRestStopped struct {
	spent       SpentHitDice
	interrupted RestInterrupted
}

// SpentHitDice returns the spent Hit Point Dice, unchanged.
func (r TakeRestStopped) SpentHitDice() SpentHitDice { return r.spent }

// Interrupted returns the RestInterrupted event.
func (r TakeRestStopped) Interrupted() RestInterrupted { return r.interrupted }

// TakeRestRejected: the report is well-typed but a rule refuses it; nothing
// changed.
type TakeRestRejected struct {
	spent   SpentHitDice
	problem RestReportRejection
}

// SpentHitDice returns the spent Hit Point Dice, unchanged.
func (r TakeRestRejected) SpentHitDice() SpentHitDice { return r.spent }

// Problem returns why the report was refused.
func (r TakeRestRejected) Problem() RestReportRejection { return r.problem }

func (TakeRestApplied) isTakeRestResult()                  {}
func (TakeRestAppliedAfterInterruption) isTakeRestResult() {}
func (TakeRestNotStarted) isTakeRestResult()               {}
func (TakeRestIncomplete) isTakeRestResult()               {}
func (TakeRestStopped) isTakeRestResult()                  {}
func (TakeRestRejected) isTakeRestResult()                 {}

// RestTaker is the use case that turns a reported rest into what it confers and
// emits the srd5e.character.take_rest span and the rests counter. It holds no
// rules logic, calls no port other than telemetry, reads no Hit Points, clock or
// Character, and returns effects as data for the host to apply. Create it with
// NewRestTaker; the zero value is not usable. It is safe for concurrent use only
// if its telemetry and clock are.
type RestTaker struct {
	tel   telemetry.Telemetry
	clock func() time.Time
	rests telemetry.Counter
	dur   telemetry.Histogram
	set   bool
}

// NewRestTaker returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored.
func NewRestTaker(opts ...Option) RestTaker {
	o := newOptions(opts)
	m := o.tel.Meter
	return RestTaker{
		tel:   o.tel,
		clock: o.clock,
		rests: m.Counter(telemetry.Descriptor{Name: MetricRests, Unit: "{rest}", Description: "Rests taken, by kind and result."}),
		dur:   m.Histogram(telemetry.Descriptor{Name: MetricRestOperationDuration, Unit: "s", Description: "Duration of one rest use case call."}),
		set:   true,
	}
}

// TakeRest resolves cmd: whether the rest could start (checked first, on the
// values at the start of the rest), how it ended, and what it confers. Every
// outcome is the one CheckRestStart, ResolveRest and NewRestBenefits return for
// the same inputs. It records one result per call and never an error status: it
// calls no port.
//
// It returns a nil result, before any signal, for the zero RestTaker or a command
// not built by NewTakeRestCommand. It never panics for a built command.
func (u RestTaker) TakeRest(ctx context.Context, cmd TakeRestCommand) TakeRestResult {
	if !u.set || cmd.report.IsZero() {
		return nil
	}
	ctx, span := u.tel.Tracer.Start(ctx, SpanTakeRest, telemetry.String(AttrRestKind, cmd.report.kind.String()))
	defer span.End()
	timer := telemetry.StartTimer(u.clock, u.dur)
	defer timer.Stop(ctx, telemetry.String(AttrRestOperation, ValueRestOperationTakeRest))

	res := takeRest(cmd)
	result, benefits, rejection := describeTakeRest(res)
	outcome := telemetry.OutcomeFailure
	if benefits != ValueRestBenefitsNone {
		outcome = telemetry.OutcomeSuccess
	}
	attrs := []telemetry.Attr{telemetry.String(telemetry.AttrOutcome, outcome), telemetry.String(AttrResult, result)}
	if rejection != "" {
		attrs = append(attrs, telemetry.String(AttrRejection, rejection))
	}
	span.SetAttributes(attrs...)
	u.rests.Add(ctx, 1, append(attrs, telemetry.String(AttrRestKind, cmd.report.kind.String()), telemetry.String(AttrRestBenefits, benefits))...)
	return res
}

// takeRest composes the domain functions for cmd.
func takeRest(cmd TakeRestCommand) TakeRestResult {
	start, err := CheckRestStart(cmd.report.kind, cmd.currentHitPoints, cmd.exhaustionLevel, cmd.spacing)
	switch d := start.(type) {
	case RestMayNotStart:
		return TakeRestNotStarted{decision: d}
	case RestMayStart:
	default:
		panic(err) // unreachable: the command was validated
	}
	resolution, err := ResolveRest(cmd.report)
	switch r := resolution.(type) {
	case RestCompleted:
		return takeRestApplied(cmd, r.kind, RestFinished)
	case RestStoppedWithShortRestBenefits:
		applied := takeRestApplied(cmd, ShortRest, LongRestInterrupted)
		return TakeRestAppliedAfterInterruption{
			spent: applied.spent, applied: applied.applied,
			interrupted: RestInterrupted{id: cmd.id, kind: cmd.report.kind, interruption: r.interruption, benefitsGranted: RestBenefitsOfShortRest},
		}
	case RestStoppedWithoutBenefits:
		return TakeRestStopped{spent: cmd.spent, interrupted: RestInterrupted{id: cmd.id, kind: cmd.report.kind, interruption: r.interruption, benefitsGranted: RestBenefitsNone}}
	case RestIncomplete:
		return TakeRestIncomplete{spent: cmd.spent, hoursStillNeeded: r.hoursStillNeeded}
	case RestReportRejected:
		return TakeRestRejected{spent: cmd.spent, problem: r.problem}
	default:
		panic(err) // unreachable: the report was validated
	}
}

// takeRestApplied builds the applied result for the benefits of kind.
func takeRestApplied(cmd TakeRestCommand, kind RestKind, source RestSource) TakeRestApplied {
	b := restBenefits(kind, cmd.exhaustionLevel, cmd.spent)
	return TakeRestApplied{spent: b.spent, applied: RestApplied{id: cmd.id, benefitsOf: kind, source: source, effects: b.effects}}
}

// describeTakeRest maps a result to its result, rest_benefits and rejection
// attribute values; the rejection is "" for a result that has none.
func describeTakeRest(res TakeRestResult) (result, benefits, rejection string) {
	switch r := res.(type) {
	case TakeRestApplied:
		b := ValueRestBenefitsLong
		if r.applied.benefitsOf == ShortRest {
			b = ValueRestBenefitsShort
		}
		return ValueResultApplied, b, ""
	case TakeRestAppliedAfterInterruption:
		return ValueResultAppliedAfterInterruption, ValueRestBenefitsShort, ""
	case TakeRestNotStarted:
		return ValueResultNotStarted, ValueRestBenefitsNone, blockerValue(r.decision.blockers[0])
	case TakeRestIncomplete:
		return ValueResultIncomplete, ValueRestBenefitsNone, ""
	case TakeRestStopped:
		return ValueResultStopped, ValueRestBenefitsNone, ""
	case TakeRestRejected:
		return ValueResultRejected, ValueRestBenefitsNone, r.problem.String()
	default:
		panic(invalidInput("unknown take rest result")) // unreachable: TakeRestResult is closed
	}
}

// blockerValue maps a blocker to its ValueRejection* value.
func blockerValue(b RestStartBlocker) string {
	switch b.(type) {
	case RestStartNoHitPoints:
		return ValueRejectionNoHitPoints
	case RestStartExhaustionLevelSix:
		return ValueRejectionExhaustionLevelSix
	}
	return ValueRejectionTooSoonAfterLongRest
}
