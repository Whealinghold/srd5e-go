package combat

import "fmt"

// combatResultClass classifies every result value of a combat use case for
// telemetry: resultApplied, resultRefused or resultInvalid. The zero value is
// not a class.
type combatResultClass int

const (
	// resultApplied: the action took effect.
	resultApplied combatResultClass = iota + 1
	// resultRefused: legal input the rules do not allow in the current state.
	resultRefused
	// resultInvalid: host input the engine cannot use.
	resultInvalid
)

// String returns the value of srd5e.combat.result_class.
func (c combatResultClass) String() string {
	switch c {
	case resultApplied:
		return valueResultClassApplied
	case resultRefused:
		return valueResultClassRefused
	case resultInvalid:
		return valueResultClassInvalid
	}
	return "unknown"
}

// Outcome returns the value of srd5e.outcome: success for applied, failure
// otherwise. It panics with an error for a value that is not a class.
func (c combatResultClass) Outcome() string {
	switch c {
	case resultApplied:
		return valueOutcomeSuccess
	case resultRefused, resultInvalid:
		return valueOutcomeFailure
	default:
		panic(fmt.Errorf("combat: unreachable result class %d", int(c)))
	}
}

// IsSpanError reports whether the span of this class gets error status: only
// invalid. A refusal is a normal rule outcome.
func (c combatResultClass) IsSpanError() bool { return c == resultInvalid }
