package slogtelemetry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// levelTrace is one step below slog.LevelDebug, the usual convention for trace.
const levelTrace = slog.LevelDebug - 4

// Logger implements telemetry.Logger on a *slog.Logger. The zero Logger, and a
// Logger made from a nil *slog.Logger, enable no level and log nothing. It is
// safe for concurrent use as far as the wrapped slog.Logger is.
type Logger struct {
	l *slog.Logger
}

var _ telemetry.Logger = Logger{}

// New returns a Logger that writes to l. A nil l yields a Logger that logs
// nothing. New never panics and creates no handler.
func New(l *slog.Logger) Logger { return Logger{l: l} }

// Enabled reports whether the wrapped logger's handler is enabled for level.
func (a Logger) Enabled(ctx context.Context, level telemetry.Level) bool {
	return a.l != nil && a.l.Enabled(ctx, slogLevel(level))
}

// Log writes the event through the wrapped logger. attrs is called at most
// once, and only when the level is enabled. An invalid Attr is skipped.
func (a Logger) Log(ctx context.Context, level telemetry.Level, msg string, attrs func() []telemetry.Attr) {
	if !a.Enabled(ctx, level) {
		return
	}
	var out []slog.Attr
	if attrs != nil {
		for _, at := range attrs() {
			if at.Valid() {
				out = append(out, slogAttr(at))
			}
		}
	}
	a.l.LogAttrs(ctx, slogLevel(level), msg, out...)
}

// slogLevel maps a level; a level outside the defined set is clamped to the
// nearest defined one.
func slogLevel(l telemetry.Level) slog.Level {
	switch {
	case l <= telemetry.LevelTrace:
		return levelTrace
	case l == telemetry.LevelDebug:
		return slog.LevelDebug
	case l == telemetry.LevelInfo:
		return slog.LevelInfo
	case l == telemetry.LevelWarn:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

func slogAttr(a telemetry.Attr) slog.Attr {
	switch v := a.Value().(type) {
	case string:
		return slog.String(a.Key(), v)
	case int64:
		return slog.Int64(a.Key(), v)
	case float64:
		return slog.Float64(a.Key(), v)
	case bool:
		return slog.Bool(a.Key(), v)
	default:
		return slog.String(a.Key(), fmt.Sprint(v))
	}
}
