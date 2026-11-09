package character

import (
	"errors"
	"math"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// RestEnd is how the host reports that a rest ended. Hours are totals across all
// segments of one rest, excluding the interruptions themselves, as whole hours
// the host has floored: a Long Rest stopped after 3 hours, resumed and stopped
// again after 2 more hours is reported as 5 hours with 1 interruption resumed.
// The package has no clock, and the host must report only periods that meet the
// sleep and activity limits of the rest (SRD 5.2.1, "Long Rest", p. 185: at
// least 6 hours asleep and no more than 2 hours of light activity; "Short Rest",
// p. 187: nothing more strenuous than reading, talking, eating or standing
// watch). Variants: RestRanItsCourse, RestStopped.
type RestEnd interface{ isRestEnd() }

// RestRanItsCourse says the rest was not stopped and lasted hoursRested after
// being resumed interruptionsResumed times. Build it with NewRestRanItsCourse.
type RestRanItsCourse struct{ hoursRested, interruptionsResumed int }

// NewRestRanItsCourse returns the end of a rest that was not stopped. The values
// are checked by NewRestReport, not here.
func NewRestRanItsCourse(hoursRested, interruptionsResumed int) RestRanItsCourse {
	return RestRanItsCourse{hoursRested: hoursRested, interruptionsResumed: interruptionsResumed}
}

// HoursRested returns the total hours rested.
func (e RestRanItsCourse) HoursRested() int { return e.hoursRested }

// InterruptionsResumed returns how many times the rest was resumed.
func (e RestRanItsCourse) InterruptionsResumed() int { return e.interruptionsResumed }

// RestStopped says the rest was stopped by interruption after hoursRestedBefore
// hours, having been resumed interruptionsResumed times before. Build it with
// NewRestStopped.
type RestStopped struct {
	interruption                            RestInterruption
	hoursRestedBefore, interruptionsResumed int
}

// NewRestStopped returns the end of a rest that was stopped. The values are
// checked by NewRestReport, not here.
func NewRestStopped(interruption RestInterruption, hoursRestedBefore, interruptionsResumed int) RestStopped {
	return RestStopped{interruption: interruption, hoursRestedBefore: hoursRestedBefore, interruptionsResumed: interruptionsResumed}
}

// Interruption returns what stopped the rest.
func (e RestStopped) Interruption() RestInterruption { return e.interruption }

// HoursRestedBefore returns the total hours rested before the stop.
func (e RestStopped) HoursRestedBefore() int { return e.hoursRestedBefore }

// InterruptionsResumed returns how many times the rest was resumed before.
func (e RestStopped) InterruptionsResumed() int { return e.interruptionsResumed }

func (RestRanItsCourse) isRestEnd() {}
func (RestStopped) isRestEnd()      {}

// RestReportProblem is one reason NewRestReport refused host numbers. The zero
// value is not a problem. It is an error matching srderr.ErrInvalidInput.
//
//nolint:errname // an enum of problems, named by the spec
type RestReportProblem int

// The problems of a rest report, in input order.
const (
	// NegativeHoursRested: the hours of RestRanItsCourse are negative.
	NegativeHoursRested RestReportProblem = iota + 1 //nolint:errname // a member of the problem enum
	// NegativeHoursRestedBefore: the hours of RestStopped are negative.
	NegativeHoursRestedBefore //nolint:errname // a member of the problem enum
	// NegativeInterruptionsResumed: the resumed count is negative.
	NegativeInterruptionsResumed //nolint:errname // a member of the problem enum
)

// String returns the snake_case name of p, or "unknown".
func (p RestReportProblem) String() string {
	switch p {
	case NegativeHoursRested:
		return "negative_hours_rested"
	case NegativeHoursRestedBefore:
		return "negative_hours_rested_before"
	case NegativeInterruptionsResumed:
		return "negative_interruptions_resumed"
	}
	return "unknown"
}

// Error renders the problem. The text is not a contract.
func (p RestReportProblem) Error() string { return "character: invalid rest report: " + p.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (RestReportProblem) Unwrap() error { return srderr.ErrInvalidInput }

// RestReport is what the host reports about one rest: its kind, the Long Rest
// variant and how it ended (see RestEnd for the hours it reports). The values of
// the report that decide whether the rest could start (Hit Points, Exhaustion
// level, spacing) are on TakeRestCommand and are those at the start of the rest.
// Build it with NewRestReport; the zero RestReport is invalid. Comparable and
// safe for concurrent use.
type RestReport struct {
	kind    RestKind
	variant LongRestVariant
	end     RestEnd
}

// NewRestReport returns the report of a rest of kind, with the Long Rest variant
// (which a Short Rest ignores but must still be valid) and the end.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid kind or
// variant, a nil end or an invalid interruption, and otherwise an errors.Join of
// every RestReportProblem, once each, in input order, each matching
// srderr.ErrInvalidInput. A resumed Short Rest is valid input: ResolveRest
// rejects it. It never panics.
func NewRestReport(kind RestKind, variant LongRestVariant, end RestEnd) (RestReport, error) {
	if !kind.Valid() || !variant.Valid() {
		return RestReport{}, invalidInput("rest report with an invalid kind or variant")
	}
	var problems []error
	add := func(bad bool, p RestReportProblem) {
		if bad {
			problems = append(problems, p)
		}
	}
	switch e := end.(type) {
	case RestRanItsCourse:
		add(e.hoursRested < 0, NegativeHoursRested)
		add(e.interruptionsResumed < 0, NegativeInterruptionsResumed)
	case RestStopped:
		if !e.interruption.Valid() {
			return RestReport{}, invalidInput("rest report with an invalid interruption")
		}
		add(e.hoursRestedBefore < 0, NegativeHoursRestedBefore)
		add(e.interruptionsResumed < 0, NegativeInterruptionsResumed)
	default:
		return RestReport{}, invalidInput("rest report without an end")
	}
	if len(problems) > 0 {
		return RestReport{}, errors.Join(problems...)
	}
	return RestReport{kind: kind, variant: variant, end: end}, nil
}

// MustRestReport is NewRestReport for known-valid input. It panics with the
// error NewRestReport returns, and only then.
func MustRestReport(kind RestKind, variant LongRestVariant, end RestEnd) RestReport {
	r, err := NewRestReport(kind, variant, end)
	if err != nil {
		panic(err)
	}
	return r
}

// Kind returns the kind of rest.
func (r RestReport) Kind() RestKind { return r.kind }

// Variant returns the Long Rest variant.
func (r RestReport) Variant() LongRestVariant { return r.variant }

// End returns how the rest ended.
func (r RestReport) End() RestEnd { return r.end }

// IsZero reports whether r is the invalid zero RestReport.
func (r RestReport) IsZero() bool { return r.end == nil }

// RestReportRejection is why ResolveRest refused a well-typed report. Its one
// value is ShortRestCannotBeResumed. The zero value is not a rejection.
type RestReportRejection int

// ShortRestCannotBeResumed: a Short Rest report with InterruptionsResumed above
// 0. The SRD gives resumption for a Long Rest only (SRD 5.2.1, "Long Rest",
// p. 185).
const ShortRestCannotBeResumed RestReportRejection = iota + 1

// String returns "short_rest_cannot_be_resumed" or "unknown".
func (r RestReportRejection) String() string {
	if r == ShortRestCannotBeResumed {
		return "short_rest_cannot_be_resumed"
	}
	return "unknown"
}

// RestResolution is what ResolveRest makes of a report. Variants: RestCompleted,
// RestIncomplete, RestStoppedWithShortRestBenefits, RestStoppedWithoutBenefits,
// RestReportRejected.
type RestResolution interface{ isRestResolution() }

// RestCompleted: the rest finished and grants the benefits of its kind.
type RestCompleted struct{ kind RestKind }

// Kind returns the kind of rest whose benefits are granted.
func (r RestCompleted) Kind() RestKind { return r.kind }

// RestIncomplete: the rest ran its course but was too short; it grants nothing.
type RestIncomplete struct{ hoursStillNeeded int }

// HoursStillNeeded returns how many more hours the rest needed, at least 1.
func (r RestIncomplete) HoursStillNeeded() int { return r.hoursStillNeeded }

// RestStoppedWithShortRestBenefits: a Long Rest interrupted after at least 1
// hour but before its required hours; it grants the benefits of a Short Rest.
type RestStoppedWithShortRestBenefits struct{ interruption RestInterruption }

// Interruption returns what stopped the rest.
func (r RestStoppedWithShortRestBenefits) Interruption() RestInterruption { return r.interruption }

// RestStoppedWithoutBenefits: the rest was interrupted and grants nothing.
type RestStoppedWithoutBenefits struct{ interruption RestInterruption }

// Interruption returns what stopped the rest.
func (r RestStoppedWithoutBenefits) Interruption() RestInterruption { return r.interruption }

// RestReportRejected: the report is well-typed but a rule refuses it.
type RestReportRejected struct{ problem RestReportRejection }

// Problem returns why the report was refused.
func (r RestReportRejected) Problem() RestReportRejection { return r.problem }

func (RestCompleted) isRestResolution()                    {}
func (RestIncomplete) isRestResolution()                   {}
func (RestStoppedWithShortRestBenefits) isRestResolution() {}
func (RestStoppedWithoutBenefits) isRestResolution()       {}
func (RestReportRejected) isRestResolution()               {}

// ResolveRest decides what a reported rest confers (SRD 5.2.1, "Short Rest",
// p. 187; "Long Rest", p. 185). A Short Rest needs 1 hour and any interruption
// leaves it without benefits; a resumed Short Rest is rejected. A Long Rest
// needs its base hours (8, or 4 with Trance) plus 1 for each interruption
// resumed. A Long Rest stopped at or after those hours is completed (inference:
// the required time had passed); stopped before them it earns the benefits of a
// Short Rest if at least 1 hour was rested, otherwise nothing.
//
// It returns an error matching srderr.ErrInvalidInput for the zero report. It
// never panics.
func ResolveRest(report RestReport) (RestResolution, error) {
	if report.IsZero() {
		return nil, invalidInput("resolve of a zero rest report")
	}
	var hours, resumed int
	var stop *RestInterruption
	switch e := report.end.(type) {
	case RestRanItsCourse:
		hours, resumed = e.hoursRested, e.interruptionsResumed
	case RestStopped:
		hours, resumed, stop = e.hoursRestedBefore, e.interruptionsResumed, &e.interruption
	}
	if report.kind == ShortRest && resumed > 0 {
		return RestReportRejected{problem: ShortRestCannotBeResumed}, nil
	}
	// Required hours are base+resumed. Compare as hours-resumed >= base so a huge
	// resumed count cannot wrap the sum negative (both are non-negative here).
	base := report.kind.MinimumHours(report.variant)
	switch {
	case hours-resumed >= base && (stop == nil || report.kind == LongRest):
		return RestCompleted{kind: report.kind}, nil
	case stop == nil:
		return RestIncomplete{hoursStillNeeded: hoursStillNeeded(base, hours, resumed)}, nil
	case report.kind == LongRest && hours >= 1:
		return RestStoppedWithShortRestBenefits{interruption: *stop}, nil
	}
	return RestStoppedWithoutBenefits{interruption: *stop}, nil
}

// hoursStillNeeded is base+resumed-hours, clamped to math.MaxInt instead of wrapping.
func hoursStillNeeded(base, hours, resumed int) int {
	need := base - hours // hours >= 0, so this cannot overflow
	if need > 0 && resumed > math.MaxInt-need {
		return math.MaxInt
	}
	return need + resumed
}
