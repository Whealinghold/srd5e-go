package telemetry

import "strings"

// Attribute keys shared by every signal the module emits. The strings are a
// public contract: renaming one breaks consumers' dashboards and alerts.
const (
	// AttrOutcome is the outcome of an operation: OutcomeSuccess or
	// OutcomeFailure. A normal rule outcome (a failed check) is a success of
	// the operation; a returned failure result is a failure.
	AttrOutcome = "srd5e.outcome"
	// AttrErrorType is the kind of a returned error, from srderr.ErrorType. It
	// is set only for a returned error, never for a failure result.
	AttrErrorType = "error.type"
)

// Values of AttrOutcome.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
)

// namePrefix is the first segment of every span and instrument name.
const namePrefix = "srd5e"

// ValidName reports whether name has the shape of a span or instrument name:
// srd5e.<context>.<noun>, at least three dot-separated segments, the first
// being "srd5e" and each other segment a lowercase letter followed by lowercase
// letters, digits or underscores.
func ValidName(name string) bool {
	segs := strings.Split(name, ".")
	if len(segs) < 3 || segs[0] != namePrefix {
		return false
	}
	for _, s := range segs[1:] {
		if !validSegment(s) {
			return false
		}
	}
	return true
}

func validSegment(s string) bool {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}
