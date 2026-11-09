package character

import "slices"

// GainHeroicInspirationRejection says why a gain that gave the excess instance
// to another character was refused. The zero value is not a rejection.
type GainHeroicInspirationRejection int

const (
	// RecipientIsHolder: the choice names the holder, who already has it.
	RecipientIsHolder GainHeroicInspirationRejection = iota + 1
	// RecipientAlreadyInspired: the recipient already has Heroic Inspiration.
	RecipientAlreadyInspired
)

// String returns "recipient_is_holder", "recipient_already_inspired" or "unknown".
func (r GainHeroicInspirationRejection) String() string {
	switch r {
	case RecipientIsHolder:
		return "recipient_is_holder"
	case RecipientAlreadyInspired:
		return "recipient_already_inspired"
	}
	return "unknown"
}

// ExcessResolution is the closed set of outcomes of ResolveExcessInspiration.
// Variants: ExcessResolutionReceived, ExcessResolutionExcessLost,
// ExcessResolutionExcessGiven, ExcessResolutionRejected.
type ExcessResolution interface{ isExcessResolution() }

// ExcessResolutionReceived: the holder was not Inspired and now is.
type ExcessResolutionReceived struct {
	holder Character
	events []Event
}

// Holder returns the holder after the gain.
func (r ExcessResolutionReceived) Holder() Character { return r.holder }

// Events returns a new slice holding the one HeroicInspirationGained event.
func (r ExcessResolutionReceived) Events() []Event { return slices.Clone(r.events) }

// ExcessResolutionExcessLost: the holder was Inspired and the excess was lost.
type ExcessResolutionExcessLost struct {
	holder Character
	events []Event
}

// Holder returns the holder, unchanged.
func (r ExcessResolutionExcessLost) Holder() Character { return r.holder }

// Events returns a new, empty slice: nothing happened.
func (r ExcessResolutionExcessLost) Events() []Event { return slices.Clone(r.events) }

// ExcessResolutionExcessGiven: the holder was Inspired and the excess went to a
// recipient who lacked it.
type ExcessResolutionExcessGiven struct {
	holder, recipient Character
	events            []Event
}

// Holder returns the holder, unchanged.
func (r ExcessResolutionExcessGiven) Holder() Character { return r.holder }

// Recipient returns the recipient after the gain.
func (r ExcessResolutionExcessGiven) Recipient() Character { return r.recipient }

// Events returns a new slice holding the recipient's HeroicInspirationGained event.
func (r ExcessResolutionExcessGiven) Events() []Event { return slices.Clone(r.events) }

// ExcessResolutionRejected: the excess could not be given; nothing changed.
type ExcessResolutionRejected struct {
	reason GainHeroicInspirationRejection
}

// Reason returns why the give was refused.
func (r ExcessResolutionRejected) Reason() GainHeroicInspirationRejection { return r.reason }

func (ExcessResolutionReceived) isExcessResolution()    {}
func (ExcessResolutionExcessLost) isExcessResolution()  {}
func (ExcessResolutionExcessGiven) isExcessResolution() {}
func (ExcessResolutionRejected) isExcessResolution()    {}

// ResolveExcessInspiration decides what gaining Heroic Inspiration does to the
// holder and, for a give, the recipient (SRD 5.2.1, "Heroic Inspiration", "Only
// One at a Time", p. 8; "Rules Glossary", p. 183). A holder who is not Inspired
// receives it and the choice is not read. For an Inspired holder, in order: a
// give to the holder's own id is RecipientIsHolder, a give to an Inspired
// recipient is RecipientAlreadyInspired, a loss (or any other choice) is
// ExcessResolutionExcessLost, and otherwise the recipient receives it. The
// inputs are not altered. It is pure, never panics and returns no error.
func ResolveExcessInspiration(holder Character, choice ExcessInspirationChoice) ExcessResolution {
	granted := holder.GrantHeroicInspiration()
	if holder.inspiration != Inspired {
		return ExcessResolutionReceived{holder: granted.Character(), events: granted.events}
	}
	give, ok := choice.(GiveExcessInspirationTo)
	if !ok {
		return ExcessResolutionExcessLost{holder: granted.Character(), events: granted.events}
	}
	switch {
	case give.Recipient.id == holder.id:
		return ExcessResolutionRejected{reason: RecipientIsHolder}
	case give.Recipient.inspiration == Inspired:
		return ExcessResolutionRejected{reason: RecipientAlreadyInspired}
	}
	to := give.Recipient.GrantHeroicInspiration()
	return ExcessResolutionExcessGiven{holder: holder, recipient: to.Character(), events: to.events}
}
