package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// SentientConflict holds the rules of a conflict between a sentient item and its
// bearer. The zero value is ready to use.
type SentientConflict struct{}

// DC returns the bearer's Charisma saving throw DC: 12 plus the item's Charisma
// modifier (SRD 5.2.1, "Conflict", p. 208).
func (SentientConflict) DC(charisma ability.Score) d20.DifficultyClass {
	return d20.DifficultyClass(12 + charisma.Modifier().Value())
}

// SentientDemandsReason says why a demands list was refused. The zero value is not a reason.
type SentientDemandsReason int

// The reasons.
const (
	SentientDemandsEmpty SentientDemandsReason = iota + 1
	SentientDemandsDuplicate
	SentientDemandsTooMany
)

// SentientDemandsRejected is the error NewSentientDemands returns. It matches
// srderr.ErrInvalidInput.
//
//nolint:errname // the name is the spec's (FR-119); it is an error with typed data
type SentientDemandsRejected struct{ Reason SentientDemandsReason }

// Error renders the error. The text is not a contract.
func (e *SentientDemandsRejected) Error() string { return "character: sentient demands rejected" }

// Unwrap returns srderr.ErrInvalidInput.
func (e *SentientDemandsRejected) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *SentientDemandsRejected with the same Reason.
func (e *SentientDemandsRejected) Is(target error) bool {
	t, ok := target.(*SentientDemandsRejected)
	return ok && t != nil && e != nil && *t == *e
}

// SentientDemands are one to four distinct demands. Immutable; the zero value
// holds none and is invalid.
type SentientDemands struct{ demands []content.SentientDemand }

// NewSentientDemands returns the demands, copying the slice. It returns a
// *SentientDemandsRejected (matching srderr.ErrInvalidInput) for none, a repeat
// or more than four, and an error matching srderr.ErrInvalidInput for an invalid
// demand.
func NewSentientDemands(demands []content.SentientDemand) (SentientDemands, error) {
	switch {
	case len(demands) == 0:
		return SentientDemands{}, &SentientDemandsRejected{Reason: SentientDemandsEmpty}
	case len(demands) > 4:
		return SentientDemands{}, &SentientDemandsRejected{Reason: SentientDemandsTooMany}
	}
	for i, d := range demands {
		if !d.Valid() {
			return SentientDemands{}, invalidInput("sentient demands: invalid demand")
		}
		if slices.Contains(demands[:i], d) {
			return SentientDemands{}, &SentientDemandsRejected{Reason: SentientDemandsDuplicate}
		}
	}
	return SentientDemands{demands: slices.Clone(demands)}, nil
}

// Demands returns a new slice of the demands.
func (d SentientDemands) Demands() []content.SentientDemand { return slices.Clone(d.demands) }

// IsZero reports whether d holds no demand.
func (d SentientDemands) IsZero() bool { return len(d.demands) == 0 }

// SentientSave is how a bearer's saving throw in a conflict was decided: the
// closed set SentientSaveRolled and SentientSaveForced.
type SentientSave interface {
	// Failed reports whether the save failed.
	Failed() bool
	isSentientSave()
}

// SentientSaveRolled: the save was rolled.
type SentientSaveRolled struct{ Check d20.CheckResult }

// SentientSaveForced: the save failed without a roll.
type SentientSaveForced struct{ Failure d20.ForcedFailureResult }

// Failed reports whether the rolled save failed.
func (s SentientSaveRolled) Failed() bool { return s.Check.Outcome() == d20.Failure }

// Failed is always true.
func (SentientSaveForced) Failed() bool { return true }

func (SentientSaveRolled) isSentientSave() {}
func (SentientSaveForced) isSentientSave() {}

// SentientControlPower is whether a sentient item can try to take control. The
// zero value is ControlPowerAvailable.
type SentientControlPower int

// The power states.
const (
	ControlPowerAvailable SentientControlPower = iota
	ControlPowerSpentUntilNextDawn
)

// AfterDawn returns ControlPowerAvailable: the item can use the power again at
// the next dawn.
func (SentientControlPower) AfterDawn() SentientControlPower { return ControlPowerAvailable }

// SentientControlEffect is the control a sentient item takes: for Hours the
// bearer is Charmed and must try to follow the item's commands. The host applies
// the condition. Immutable and comparable; the zero value is invalid.
type SentientControlEffect struct{ hours int }

// NewSentientControlEffect returns the effect of hours, 1 to 12. It returns an
// error matching srderr.ErrInvalidInput otherwise.
func NewSentientControlEffect(hours int) (SentientControlEffect, error) {
	if hours < 1 || hours > 12 {
		return SentientControlEffect{}, invalidInput("sentient control effect: hours outside 1 to 12")
	}
	return SentientControlEffect{hours: hours}, nil
}

// Hours returns how long the control lasts.
func (e SentientControlEffect) Hours() int { return e.hours }

// Condition returns content.ConditionCharmed.
func (SentientControlEffect) Condition() content.ConditionKind { return content.ConditionCharmed }

// MustTryToFollowItemCommands returns true.
func (SentientControlEffect) MustTryToFollowItemCommands() bool { return true }
