package character

import (
	"context"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ContentSource is the content lookup the character use cases need: one method.
// Any content.Source satisfies it, as does contenttest.FakeSource. Expected
// absence is content.LookupNotFound; an infrastructure failure is the returned
// error, which the use case wraps once with srderr.GuardValue. A source shared
// across goroutines MUST be safe for concurrent use.
type ContentSource interface {
	Lookup(ctx context.Context, key content.Key) (content.LookupResult, error)
}

// options is what the character use cases are configured with.
type options struct {
	source ContentSource
	roller Roller
	tel    telemetry.Telemetry
	clock  func() time.Time
}

// Option configures a character use case.
type Option func(*options)

// WithSource sets the content source. A use case that reads content returns an
// error matching srderr.ErrInvalidInput from its constructor without one.
func WithSource(s ContentSource) Option {
	return func(o *options) { o.source = s }
}

// WithRoller sets the dice roller. A use case that rolls returns an error matching
// srderr.ErrInvalidInput from its constructor without one.
func WithRoller(r Roller) Option {
	return func(o *options) { o.roller = r }
}

// WithTelemetry sets the telemetry a use case emits to. Nil members of t
// default to no-ops. Without this option a use case is silent. The members MUST
// be safe for concurrent use if the use case is shared across goroutines.
func WithTelemetry(t telemetry.Telemetry) Option {
	return func(o *options) { o.tel = t }
}

// WithClock sets the clock that times calls for the duration histogram; the
// default is time.Now. A nil clock is ignored. It MUST be safe for concurrent
// use if the use case is shared across goroutines.
func WithClock(now func() time.Time) Option {
	return func(o *options) {
		if now != nil {
			o.clock = now
		}
	}
}

// newOptions applies opts over the defaults (no-op telemetry, time.Now).
func newOptions(opts []Option) options {
	o := options{clock: time.Now}
	for _, opt := range opts {
		opt(&o)
	}
	o.tel = o.tel.WithDefaults()
	return o
}
