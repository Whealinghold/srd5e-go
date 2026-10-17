package telemetrytest

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Violations reported by the recording doubles. Each reported error matches one
// of these with errors.Is.
var (
	// ErrInvalidName is a span or instrument name that is not
	// srd5e.<context>.<noun> (see telemetry.ValidName).
	ErrInvalidName = errors.New("invalid signal name")
	// ErrInvalidAttr is an attribute with an empty key or no value.
	ErrInvalidAttr = errors.New("invalid attribute")
	// ErrSpanEnded is a write to a span that has already ended.
	ErrSpanEnded = errors.New("span already ended")
	// ErrNegativeCount is a negative counter increment.
	ErrNegativeCount = errors.New("counter increment is negative")
	// ErrInstrumentConflict is an instrument name redefined with a different
	// kind, unit or description.
	ErrInstrumentConflict = errors.New("instrument redefined differently")
)

// violations collects misuse. The zero value is ready to use.
type violations struct {
	mu   sync.Mutex
	list []error
}

func (v *violations) add(err error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.list = append(v.list, err)
}

func (v *violations) snapshot() []error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.list)
}

func (v *violations) reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.list = nil
}

// validAttrs returns the valid attributes of attrs, reporting each invalid one.
func (v *violations) validAttrs(attrs []telemetry.Attr) []telemetry.Attr {
	out := make([]telemetry.Attr, 0, len(attrs))
	for _, a := range attrs {
		if !a.Valid() {
			v.add(fmt.Errorf("%w: key %q", ErrInvalidAttr, a.Key()))
			continue
		}
		out = append(out, a)
	}
	return out
}

// Violator is implemented by every recording double.
type Violator interface {
	Violations() []error
}

// RequireNoViolations fails tb for each violation vs has collected.
func RequireNoViolations(tb testing.TB, vs ...Violator) {
	tb.Helper()
	for _, v := range vs {
		for _, err := range v.Violations() {
			tb.Errorf("telemetry misuse: %v", err)
		}
	}
}
