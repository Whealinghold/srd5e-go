package telemetry

import "context"

// Status is the status of a Span.
type Status int

// The span statuses. The zero value is StatusUnset.
const (
	// StatusUnset is the default, and what a normal rule outcome leaves.
	StatusUnset Status = iota
	// StatusOK means the operation completed without failure.
	StatusOK
	// StatusError means the operation failed: it returned an error or a
	// failure result.
	StatusError
)

// String returns the lowercase status name, or "invalid".
func (s Status) String() string {
	switch s {
	case StatusUnset:
		return "unset"
	case StatusOK:
		return "ok"
	case StatusError:
		return "error"
	default:
		return "invalid"
	}
}

// Span is the handle through which work describes its span. Obtain one from
// Tracer.Start and defer End immediately.
//
// End is idempotent. Other methods must not be called after End; a recording
// double reports that as a violation. Implementations must tolerate nil or
// empty arguments without panicking.
type Span interface {
	// SetAttributes adds attributes, replacing earlier values for the same key.
	SetAttributes(attrs ...Attr)
	// SetStatus sets the status. The last call wins.
	SetStatus(status Status)
	// RecordError appends err to the span. It does not change the status and
	// does not end the span. A nil err is ignored.
	RecordError(err error)
	// End ends the span. Calling it again has no effect.
	End()
}

// Tracer is the port through which the module runs work inside a named span.
//
// Names follow srd5e.<context>.<noun> (see ValidName). The implementation
// measures duration with its own clock and links parent and child spans through
// the context it returns. Start must return a non-nil Span and a context that
// still carries the caller's values, and must tolerate a cancelled ctx.
//
// Start returns no error: a backend handles its own failures, and a backend that
// panics is the host's bug; srd5e does not recover.
//
// Implementations, and the Spans they return, MUST be safe for concurrent use:
// a d20.Resolver shares one Tracer across every call, so Start may run on many
// goroutines at once.
type Tracer interface {
	// Start begins a span called name with the given initial attributes and
	// returns the context to pass to child work. The caller defers Span.End
	// immediately and passes the returned context to child work.
	Start(ctx context.Context, name string, attrs ...Attr) (context.Context, Span)
}

// NoopSpan is a Span that records nothing and accepts every call at any time.
type NoopSpan struct{}

var _ Span = NoopSpan{}

// SetAttributes does nothing.
func (NoopSpan) SetAttributes(...Attr) {}

// SetStatus does nothing.
func (NoopSpan) SetStatus(Status) {}

// RecordError does nothing.
func (NoopSpan) RecordError(error) {}

// End does nothing.
func (NoopSpan) End() {}

// NoopTracer is a Tracer that traces nothing. It returns ctx unchanged and a
// NoopSpan, reads no clock and validates nothing. The zero value is ready to
// use.
type NoopTracer struct{}

var _ Tracer = NoopTracer{}

// Start returns ctx unchanged and a NoopSpan.
func (NoopTracer) Start(ctx context.Context, _ string, _ ...Attr) (context.Context, Span) {
	return ctx, NoopSpan{}
}
