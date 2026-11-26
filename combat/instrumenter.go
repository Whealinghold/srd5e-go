package combat

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// description is how a use case classifies its result for telemetry: the
// result class, the srd5e.combat.result name, the events produced and optional
// closed-value extras (span and operations counter only).
type description struct {
	Class  combatResultClass
	Name   string
	Events []Event
	Extras []telemetry.Attr
}

// instrumenter is the one helper through which combat use cases record their
// shared signals. Its zero value is silent. Safe for concurrent use.
type instrumenter struct {
	tel   telemetry.Telemetry
	clock func() time.Time
}

// newInstrumenter returns an instrumenter over t. A zero t becomes Noop and a
// nil clock becomes time.Now.
func newInstrumenter(t telemetry.Telemetry, clock func() time.Time) instrumenter {
	return instrumenter{tel: t.WithDefaults(), clock: clock}
}

// run wraps work in the span of op, times it and records the result that
// describe classifies.
//
// A non-nil error from work is recorded on the span and returned unchanged; a
// *srderr.PortError, or an error matching srderr.ErrInvalidInput, also increments srd5e.combat.errors once, and no operations
// or events are counted. A result is counted once under srd5e.combat.operations
// and each event under srd5e.combat.events. If describe returns an extra whose
// key is a shared attribute or is not owned by combat or d20, run records the
// error on the span and returns the zero R and an error matching
// srderr.ErrInvalidInput. A panic from work or describe propagates unchanged
// with the span ended.
func run[R any](ctx context.Context, in instrumenter, op combatOperation,
	work func(context.Context) (R, error), describe func(R) description) (R, error) {
	var zero R
	tel := in.tel.WithDefaults()
	clock := in.clock
	if clock == nil {
		clock = time.Now
	}
	opAttr := telemetry.String(attrOperation, op.Value())
	ctx, span := tel.Tracer.Start(ctx, op.SpanName(), opAttr)
	defer span.End()
	timer := telemetry.StartTimer(clock, tel.Meter.Histogram(telemetry.Descriptor{
		Name: metricOperationDuration, Unit: "s", Description: "Duration of one combat use case call."}))
	defer timer.Stop(ctx, opAttr)

	res, err := work(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.StatusError)
		if et := errorType(err); et != "" {
			tel.Meter.Counter(telemetry.Descriptor{Name: metricErrors, Unit: "{error}",
				Description: "Errors leaving a combat use case: port failures and invalid input."}).
				Add(ctx, 1, opAttr, telemetry.String(telemetry.AttrErrorType, et))
		}
		return res, err
	}

	d := describe(res)
	if err := checkExtras(d.Extras); err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.StatusError)
		return zero, err
	}
	attrs := []telemetry.Attr{
		telemetry.String(telemetry.AttrOutcome, d.Class.Outcome()),
		telemetry.String(attrResult, d.Name),
		telemetry.String(attrResultClass, d.Class.String()),
	}
	attrs = slices.Concat(attrs, d.Extras)
	span.SetAttributes(attrs...)
	if d.Class.IsSpanError() {
		span.SetStatus(telemetry.StatusError)
	}
	tel.Meter.Counter(telemetry.Descriptor{Name: metricOperations, Unit: "{operation}",
		Description: "Combat use case calls, by operation and result."}).
		Add(ctx, 1, slices.Concat([]telemetry.Attr{opAttr}, attrs)...)
	events := tel.Meter.Counter(telemetry.Descriptor{Name: metricEvents, Unit: "{event}",
		Description: "Combat domain events produced, by kind."})
	produced := 0
	for _, e := range d.Events {
		if e != nil {
			events.Add(ctx, 1, telemetry.String(attrEvent, e.Kind()))
			produced++
		}
	}
	tel.Meter.Histogram(telemetry.Descriptor{Name: metricEventsPerOperation, Unit: "{event}",
		Description: "Events one combat use case call produced, by operation."}).
		Record(ctx, float64(produced), opAttr)
	recordEndCauses(ctx, tel.Meter, d.Events)
	return res, nil
}

// checkExtras rejects an extra that is a shared attribute or whose key no
// combat or d20 signal owns.
func checkExtras(extras []telemetry.Attr) error {
	for _, a := range extras {
		switch k := a.Key(); {
		case k == attrOperation, k == attrResult, k == attrResultClass, k == attrEvent,
			k == telemetry.AttrOutcome, k == telemetry.AttrErrorType:
			return invalidInput("extra is a shared attribute")
		case !strings.HasPrefix(k, spanPrefix) && !d20Key(k):
			return invalidInput("extra key is not owned by combat")
		}
	}
	return nil
}

// d20Key reports whether k is an attribute key d20 exports.
func d20Key(k string) bool {
	switch k {
	case d20.AttrKind, d20.AttrAbility, d20.AttrRollMode, d20.AttrCriticalHit, d20.AttrRollerFailure, d20.AttrForcedFailureReason:
		return true
	}
	return false
}

// errorType is the error.type of err: the port's for a port failure, else
// invalid_input for an error matching srderr.ErrInvalidInput (a malformed
// command), else "" (counted nowhere; the span still holds it).
func errorType(err error) string {
	if et := srderr.ErrorType(err); et != "" {
		return et
	}
	if errors.Is(err, srderr.ErrInvalidInput) {
		return valueErrorTypeInvalidInput
	}
	return ""
}
