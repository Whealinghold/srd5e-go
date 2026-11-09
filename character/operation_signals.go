package character

import (
	"context"
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// operationSignals are the instruments the aggregate use cases share: the
// operations counter and the duration histogram, created once.
type operationSignals struct {
	tel   telemetry.Telemetry
	clock func() time.Time
	ops   telemetry.Counter
	dur   telemetry.Histogram
}

func newOperationSignals(o options) operationSignals {
	m := o.tel.Meter
	return operationSignals{
		tel:   o.tel,
		clock: o.clock,
		ops:   m.Counter(telemetry.Descriptor{Name: MetricOperations, Unit: "{operation}", Description: "Character use case calls, by operation and outcome."}),
		dur:   m.Histogram(telemetry.Descriptor{Name: MetricDuration, Unit: "s", Description: "Duration of one character use case call."}),
	}
}

// operationRun is one call in flight: its span and timer.
type operationRun struct {
	sig   operationSignals
	op    string
	span  telemetry.Span
	timer telemetry.Timer
	extra []telemetry.Attr
}

// start opens the span name and the timer of operation op. extra attributes go
// on the span and on the counter.
func (s operationSignals) start(ctx context.Context, name, op string, extra ...telemetry.Attr) (context.Context, operationRun) {
	ctx, span := s.tel.Tracer.Start(ctx, name, extra...)
	return ctx, operationRun{sig: s, op: op, span: span, timer: telemetry.StartTimer(s.clock, s.dur), extra: extra}
}

// end stops the timer with the operation attribute and ends the span; defer it.
func (r operationRun) end(ctx context.Context) {
	r.timer.Stop(ctx, telemetry.String(AttrOperation, r.op))
	r.span.End()
}

// finish records a returned result once. result is a ValueResult* value;
// rejection is a ValueRejection* value or "". A content_missing result is
// recorded as a failure with the error status; a rejection and a target_prone
// result (a failed save) are failures without it; every other result (changed,
// gained, expended, ...) is a success.
func (r operationRun) finish(ctx context.Context, result, rejection string) {
	r.record(ctx, r.sig.ops, true, result, rejection)
}

// record is finish onto counter ctr: withOp adds the operation attribute to the
// counter; the use case attributes follow.
func (r operationRun) record(ctx context.Context, ctr telemetry.Counter, withOp bool, result, rejection string) {
	outcome := telemetry.OutcomeSuccess
	switch result {
	case ValueResultRejected, ValueResultContentMissing, ValueResultContentIncomplete, ValueResultTargetProne, ValueResultNotDue,
		ValueResultNotListed, ValueResultRolledElsewhere, ValueResultDuplicate:
		outcome = telemetry.OutcomeFailure
	case ValueResultUnavailable: // a blocked preview is a failure; an unavailable crafting check is not
		if r.op == ValueOperationPreview {
			outcome = telemetry.OutcomeFailure
		}
	}
	attrs := []telemetry.Attr{telemetry.String(telemetry.AttrOutcome, outcome), telemetry.String(AttrResult, result)}
	if rejection != "" {
		attrs = append(attrs, telemetry.String(AttrRejection, rejection))
	}
	if result == ValueResultContentMissing || result == ValueResultContentIncomplete {
		r.span.SetStatus(telemetry.StatusError)
	}
	r.span.SetAttributes(attrs...)
	var all []telemetry.Attr
	if withOp {
		all = append(all, telemetry.String(AttrOperation, r.op))
	}
	ctr.Add(ctx, 1, slices.Concat(all, attrs, r.extra)...)
}

// commandError records an invalid-command error on the span only: the recorded
// error and error status, no counter increment and no log.
func (r operationRun) commandError(err error) {
	r.span.RecordError(err)
	r.span.SetStatus(telemetry.StatusError)
}

// portFailure records a returned *srderr.PortError once: error status, the
// recorded error, outcome failure and error.type, with no result attribute.
func (r operationRun) portFailure(ctx context.Context, err error, errorType string) {
	r.span.RecordError(err)
	r.span.SetStatus(telemetry.StatusError)
	outcome := telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure)
	r.span.SetAttributes(outcome)
	r.sig.ops.Add(ctx, 1, append([]telemetry.Attr{telemetry.String(AttrOperation, r.op), outcome, telemetry.String(telemetry.AttrErrorType, errorType)}, r.extra...)...)
}

// withAttrs adds attributes learned after the call started to the span and to the
// counter record of the run; the duration histogram keeps the operation only.
func (r operationRun) withAttrs(attrs ...telemetry.Attr) operationRun {
	r.span.SetAttributes(attrs...)
	r.extra = slices.Concat(r.extra, attrs)
	return r
}
