package telemetry

// Level is the severity of a log event, in ascending order. It maps onto
// log/slog levels and onto OpenTelemetry severity. There is no level above
// LevelError: the module never logs fatally.
type Level int

// The log levels. The zero value is LevelTrace, the most verbose.
const (
	// LevelTrace is step-by-step detail for chasing one bug; off by default.
	LevelTrace Level = iota
	// LevelDebug is normal operation detail a developer wants while debugging.
	LevelDebug
	// LevelInfo is a significant lifecycle event; rare, never per call.
	LevelInfo
	// LevelWarn is something wrong but handled.
	LevelWarn
	// LevelError is a failure that is not surfaced to the caller.
	LevelError
)

// String returns the lowercase level name, or "invalid" for a value outside
// the defined levels.
func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "trace"
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "invalid"
	}
}

// Valid reports whether l is one of the defined levels.
func (l Level) Valid() bool { return l >= LevelTrace && l <= LevelError }
