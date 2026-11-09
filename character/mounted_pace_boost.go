package character

// MountedBoostState is whether a group's mounts may still move twice the pace
// distance for 1 hour. The zero value is invalid. Comparable.
type MountedBoostState int

// The states of a MountedPaceBoost.
const (
	// MountedBoostAvailable: the doubled hour has not been used since the last rest.
	MountedBoostAvailable MountedBoostState = iota + 1
	// MountedBoostSpent: the mounts need a Short or Long Rest.
	MountedBoostSpent
)

// String returns "available", "spent" or "unknown".
func (s MountedBoostState) String() string {
	switch s {
	case MountedBoostAvailable:
		return "available"
	case MountedBoostSpent:
		return "spent"
	}
	return "unknown"
}

// MountedBoostRefusalReason says why a MountedBoostSpendRefused. The zero value
// is not a reason.
type MountedBoostRefusalReason int

// The reasons a spend is refused.
const (
	// MountedBoostMountsNeedRest: the mounts have not had a Short or Long Rest
	// since the last doubled hour.
	MountedBoostMountsNeedRest MountedBoostRefusalReason = iota + 1
)

// String returns "mounts_need_rest" or "unknown".
func (r MountedBoostRefusalReason) String() string {
	if r == MountedBoostMountsNeedRest {
		return "mounts_need_rest"
	}
	return "unknown"
}

// MountedPaceBoostEvent is the closed family of events of a MountedPaceBoost. It
// is an Event. Members: MountedPaceBoostSpent, MountedPaceBoostRestored.
type MountedPaceBoostEvent interface {
	Event
	isMountedPaceBoostEvent()
}

// MountedPaceBoostSpent is the event of the mounts moving twice the pace distance
// for 1 hour.
type MountedPaceBoostSpent struct{ id Identifier }

// MountedPaceBoostRestored is the event of the mounts having rested, so the
// doubled hour is available again.
type MountedPaceBoostRestored struct{ id Identifier }

// CharacterID returns the id of the Character whose mounts these are.
func (e MountedPaceBoostSpent) CharacterID() Identifier { return e.id }

// CharacterID returns the id of the Character whose mounts these are.
func (e MountedPaceBoostRestored) CharacterID() Identifier { return e.id }

func (MountedPaceBoostSpent) isEvent()                    {}
func (MountedPaceBoostRestored) isEvent()                 {}
func (MountedPaceBoostSpent) isMountedPaceBoostEvent()    {}
func (MountedPaceBoostRestored) isMountedPaceBoostEvent() {}

// MountedPaceBoost tracks the one hour a set of mounts that travel together may
// move twice the pace distance, after which they need a Short or Long Rest (SRD
// 5.2.1, "Travel Pace", p. 12). It belongs to one Character and is immutable: Spend
// and RestFinished return new values. Build it with NewMountedPaceBoost; the zero
// value is invalid. Comparable and safe for concurrent use.
type MountedPaceBoost struct {
	id    Identifier
	spent bool
}

// NewMountedPaceBoost returns an available boost for the Character id. It returns
// a *ExplorationError (matching srderr.ErrInvalidInput) for the zero Identifier.
func NewMountedPaceBoost(id Identifier) (MountedPaceBoost, error) {
	if id.IsZero() {
		return MountedPaceBoost{}, &ExplorationError{Input: ExplorationCharacterID}
	}
	return MountedPaceBoost{id: id}, nil
}

// State returns MountedBoostAvailable or MountedBoostSpent, or the zero state for
// the zero MountedPaceBoost.
func (b MountedPaceBoost) State() MountedBoostState {
	switch {
	case b.id.IsZero():
		return 0
	case b.spent:
		return MountedBoostSpent
	}
	return MountedBoostAvailable
}

// CharacterID returns the id of the Character whose mounts these are.
func (b MountedPaceBoost) CharacterID() Identifier { return b.id }

// MountedBoostSpendResult is the closed outcome of Spend. Variants:
// MountedBoostSpendAccepted, MountedBoostSpendRefused.
type MountedBoostSpendResult interface{ isMountedBoostSpendResult() }

// MountedBoostSpendAccepted is the outcome of spending an available boost.
type MountedBoostSpendAccepted struct {
	boost MountedPaceBoost
	event MountedPaceBoostSpent
}

// Boost returns the spent boost.
func (r MountedBoostSpendAccepted) Boost() MountedPaceBoost { return r.boost }

// Event returns the event produced.
func (r MountedBoostSpendAccepted) Event() MountedPaceBoostSpent { return r.event }

// MountedBoostSpendRefused is the outcome of spending a boost that is not
// available: the boost is unchanged and no event is produced.
type MountedBoostSpendRefused struct {
	boost  MountedPaceBoost
	reason MountedBoostRefusalReason
}

// Boost returns the unchanged boost.
func (r MountedBoostSpendRefused) Boost() MountedPaceBoost { return r.boost }

// Reason returns why the spend was refused.
func (r MountedBoostSpendRefused) Reason() MountedBoostRefusalReason { return r.reason }

func (MountedBoostSpendAccepted) isMountedBoostSpendResult() {}
func (MountedBoostSpendRefused) isMountedBoostSpendResult()  {}

// Spend uses the doubled hour. An available boost gives a
// MountedBoostSpendAccepted with a spent boost and one MountedPaceBoostSpent; a
// spent one gives a MountedBoostSpendRefused (MountedBoostMountsNeedRest). It
// never fails otherwise and never panics.
func (b MountedPaceBoost) Spend() MountedBoostSpendResult {
	if b.spent {
		return MountedBoostSpendRefused{boost: b, reason: MountedBoostMountsNeedRest}
	}
	return MountedBoostSpendAccepted{boost: MountedPaceBoost{id: b.id, spent: true}, event: MountedPaceBoostSpent{id: b.id}}
}

// MountedBoostRestResult is the closed outcome of RestFinished. Variants:
// MountedBoostRestRestored, MountedBoostRestUnchanged.
type MountedBoostRestResult interface{ isMountedBoostRestResult() }

// MountedBoostRestRestored is the outcome of a rest ending on a spent boost.
type MountedBoostRestRestored struct {
	boost MountedPaceBoost
	event MountedPaceBoostRestored
}

// Boost returns the available boost.
func (r MountedBoostRestRestored) Boost() MountedPaceBoost { return r.boost }

// Event returns the event produced.
func (r MountedBoostRestRestored) Event() MountedPaceBoostRestored { return r.event }

// MountedBoostRestUnchanged is the outcome of a rest ending on a boost that was
// already available: nothing changes and no event is produced.
type MountedBoostRestUnchanged struct{ boost MountedPaceBoost }

// Boost returns the unchanged boost.
func (r MountedBoostRestUnchanged) Boost() MountedPaceBoost { return r.boost }

func (MountedBoostRestRestored) isMountedBoostRestResult()  {}
func (MountedBoostRestUnchanged) isMountedBoostRestResult() {}

// RestFinished reports a Short Rest or Long Rest that conferred its benefits; the
// two restore the mounts alike, so it takes no rest kind. A spent boost gives a
// MountedBoostRestRestored with an available boost and one MountedPaceBoostRestored;
// an available one gives a MountedBoostRestUnchanged. A rest that conferred no
// benefits is not reported. It never fails and never panics.
func (b MountedPaceBoost) RestFinished() MountedBoostRestResult {
	if !b.spent {
		return MountedBoostRestUnchanged{boost: b}
	}
	return MountedBoostRestRestored{boost: MountedPaceBoost{id: b.id}, event: MountedPaceBoostRestored{id: b.id}}
}
