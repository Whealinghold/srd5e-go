package combat

import "github.com/Whealinghold/srd5e-go/srderr"

// RestorationRangeError reports a number outside its allowed range, from a
// constructor of the revival and restoration spells (NewDeathCircumstances,
// NewReviveOrdeal, AidIncrease, NewMassHealAllotment and the increase and
// ledger constructors). It matches srderr.ErrInvalidInput with errors.Is and
// carries the name of the quantity and the rejected value, never display text.
type RestorationRangeError struct {
	// Name is the quantity that was out of range, such as "time_since_death".
	Name string
	// Value is the rejected value (a duration is in nanoseconds).
	Value int64
}

// Error renders the error. The text is not a contract.
func (e *RestorationRangeError) Error() string {
	return "combat: " + e.Name + " is outside its range"
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *RestorationRangeError) Unwrap() error { return srderr.ErrInvalidInput }

func rangeError(name string, value int64) error { return &RestorationRangeError{name, value} }
