package character

import (
	"context"
	"time"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// featureSignals are the instruments every class feature use case shares: the
// feature_used counter, its duration histogram and its port failure counter. A
// class spec adds only its own attribute values (srd5e.class, srd5e.feature), never
// an instrument of its own.
type featureSignals struct {
	tel   telemetry.Telemetry
	clock func() time.Time
	used  telemetry.Counter
	dur   telemetry.Histogram
	errs  telemetry.Counter
}

func newFeatureSignals(o options) featureSignals {
	m := o.tel.Meter
	return featureSignals{
		tel:   o.tel,
		clock: o.clock,
		used:  m.Counter(telemetry.Descriptor{Name: MetricFeatureUsed, Unit: "{use}", Description: "Class feature uses, by class, feature, outcome and rejection."}),
		dur:   m.Histogram(telemetry.Descriptor{Name: MetricFeatureUsedDuration, Unit: "s", Description: "Duration of one class feature call."}),
		errs:  m.Counter(telemetry.Descriptor{Name: MetricFeatureUsedErrors, Unit: "{error}", Description: "Port failures leaving a class feature use case, by class, feature and error type."}),
	}
}

// featureRun is one class feature call in flight: its span, timer and the class and
// feature attributes every record carries.
type featureRun struct {
	sig   featureSignals
	span  telemetry.Span
	timer telemetry.Timer
	ident []telemetry.Attr
}

// start opens the span name for class and feature and starts the timer. The class is a parameter because every class reuses it.
func (s featureSignals) start(ctx context.Context, name, class, feature string) (context.Context, featureRun) {
	ident := []telemetry.Attr{telemetry.String(AttrClass, class), telemetry.String(AttrFeature, feature)}
	ctx, span := s.tel.Tracer.Start(ctx, name, ident...)
	return ctx, featureRun{sig: s, span: span, timer: telemetry.StartTimer(s.clock, s.dur), ident: ident}
}

// end records the duration with the class and feature and ends the span; defer it.
func (r featureRun) end(ctx context.Context) {
	r.timer.Stop(ctx, r.ident...)
	r.span.End()
}

// record counts the call once with outcome and, when it is not empty, rejection.
func (r featureRun) record(ctx context.Context, outcome, rejection string) {
	attrs := append([]telemetry.Attr{}, r.ident...)
	attrs = append(attrs, telemetry.String(telemetry.AttrOutcome, outcome))
	if rejection != "" {
		attrs = append(attrs, telemetry.String(AttrRejection, rejection))
	}
	r.span.SetAttributes(attrs[len(r.ident):]...)
	r.sig.used.Add(ctx, 1, attrs...)
}

// logResolved emits the one FINE event of a call that is not an error.
func (r featureRun) logResolved(ctx context.Context, outcome string) {
	r.sig.tel.Logger.Log(ctx, telemetry.LevelDebug, LogClassFeatureResolved, func() []telemetry.Attr {
		return append(append([]telemetry.Attr{}, r.ident...), telemetry.String(telemetry.AttrOutcome, outcome))
	})
}

// succeeded records a success.
func (r featureRun) succeeded(ctx context.Context) {
	r.record(ctx, telemetry.OutcomeSuccess, "")
	r.logResolved(ctx, telemetry.OutcomeSuccess)
}

// rejected records a rule rejection: outcome failure and the first reason, no error
// status.
func (r featureRun) rejected(ctx context.Context, reason string) {
	r.record(ctx, telemetry.OutcomeFailure, reason)
	r.logResolved(ctx, telemetry.OutcomeFailure)
}

// stillFailed records a normal rule outcome that did not succeed (Tactical Mind or
// Indomitable that still fails): outcome failure, no rejection, no error status.
func (r featureRun) stillFailed(ctx context.Context) {
	r.record(ctx, telemetry.OutcomeFailure, "")
	r.logResolved(ctx, telemetry.OutcomeFailure)
}

// contentMissing records a content-missing result: error status, outcome failure,
// the content_missing rejection, no log.
func (r featureRun) contentMissing(ctx context.Context) {
	r.span.SetStatus(telemetry.StatusError)
	r.record(ctx, telemetry.OutcomeFailure, ValueRejectionContentMissing)
}

// commandError records an invalid-command error on the span only.
func (r featureRun) commandError(err error) {
	r.span.RecordError(err)
	r.span.SetStatus(telemetry.StatusError)
}

// portFailure records a returned *srderr.PortError once: the recorded error and error
// status on the span, one feature_used increment with outcome failure and no
// error.type or rejection, and one errors increment with error.type. Nothing is logged.
func (r featureRun) portFailure(ctx context.Context, err error, errorType string) {
	r.span.RecordError(err)
	r.span.SetStatus(telemetry.StatusError)
	r.record(ctx, telemetry.OutcomeFailure, "")
	r.sig.errs.Add(ctx, 1, append(append([]telemetry.Attr{}, r.ident...), telemetry.String(telemetry.AttrErrorType, errorType))...)
}
