package telemetry

import "context"

// Logger is the port through which the module emits log events.
//
// An event is a Level, a constant message and structured attributes supplied
// lazily. Implementations MUST call attrs at most once, and only when Enabled
// is true for that level, so building attributes costs nothing when the level
// is off. attrs may be nil. Log must accept any Level (even an invalid one),
// a nil attrs and a cancelled ctx without panicking. Logger returns no error: a
// backend handles its own failures, and a backend that panics is the host's bug;
// srd5e does not recover.
//
// Implementations MUST be safe for concurrent use: a d20.Resolver shares one
// Logger across every call, so Log and Enabled may run on many goroutines at
// once.
type Logger interface {
	// Enabled reports whether an event at level would be recorded.
	Enabled(ctx context.Context, level Level) bool
	// Log emits an event. msg is a short constant; details go in attrs. To
	// report an error use the Err attribute.
	Log(ctx context.Context, level Level, msg string, attrs func() []Attr)
}

// NoopLogger is a Logger that logs nothing. No level is enabled and attrs is
// never called. The zero value is ready to use.
type NoopLogger struct{}

var _ Logger = NoopLogger{}

// Enabled always returns false.
func (NoopLogger) Enabled(context.Context, Level) bool { return false }

// Log does nothing.
func (NoopLogger) Log(context.Context, Level, string, func() []Attr) {}
