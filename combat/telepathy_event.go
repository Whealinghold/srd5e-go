package combat

// TelepathyEvent is the closed family of events the telepathy rules produce. Each
// is an immutable value comparable with ==, about the telepath and the contacted
// creature (its Combatants is exactly [telepath, contacted]) with a constant
// snake_case Kind. Variants: TelepathicContactMade, TelepathicContactBroken,
// TelepathicContactEnded. Each embeds telepathyEventBase. No function of this
// package returns a CombatantID; read the creatures from Combatants.
type TelepathyEvent interface {
	Event
	telepathyEvent()
}

// telepathyEventBase carries the pair every telepathy event is about and puts the event in the Event and TelepathyEvent families.
type telepathyEventBase struct{ telepath, contacted CombatantID }

func (telepathyEventBase) event()          {}
func (telepathyEventBase) telepathyEvent() {}

// Combatants returns a new slice [telepath, contacted].
func (b telepathyEventBase) Combatants() []CombatantID { return []CombatantID{b.telepath, b.contacted} }

func newTelepathyBase(telepath, contacted CombatantID) (telepathyEventBase, error) {
	if telepath.IsZero() || contacted.IsZero() || telepath == contacted {
		return telepathyEventBase{}, invalidInput("telepathy event needs two different creatures")
	}
	return telepathyEventBase{telepath, contacted}, nil
}

// TelepathicContactMade: the telepath made contact with the creature. Its Kind is telepathic_contact_made.
type TelepathicContactMade struct{ telepathyEventBase }

// NewTelepathicContactMade returns the event. It returns an error matching
// srderr.ErrInvalidInput for a blank id or equal ids. It never panics.
func NewTelepathicContactMade(telepath, contacted CombatantID) (TelepathicContactMade, error) {
	b, err := newTelepathyBase(telepath, contacted)
	return TelepathicContactMade{b}, err
}

// Kind returns "telepathic_contact_made".
func (TelepathicContactMade) Kind() string { return "telepathic_contact_made" }

// TelepathicContactBroken: the contact ended because of the cause. Its Kind is telepathic_contact_broken.
type TelepathicContactBroken struct {
	telepathyEventBase
	cause BreakCause
}

// NewTelepathicContactBroken returns the event. It returns an error matching
// srderr.ErrInvalidInput for a blank id, equal ids or an undefined cause. It never panics.
func NewTelepathicContactBroken(telepath, contacted CombatantID, cause BreakCause) (TelepathicContactBroken, error) {
	b, err := newTelepathyBase(telepath, contacted)
	if err != nil {
		return TelepathicContactBroken{}, err
	}
	if cause < BreakCauseIncapacitated || cause > BreakCauseTelepathContactedAnother {
		return TelepathicContactBroken{}, invalidInput("telepathic contact broken needs a cause")
	}
	return TelepathicContactBroken{b, cause}, nil
}

// Kind returns "telepathic_contact_broken".
func (TelepathicContactBroken) Kind() string { return "telepathic_contact_broken" }

// Cause returns why the contact was broken.
func (e TelepathicContactBroken) Cause() BreakCause { return e.cause }

// TelepathicContactEnded: the telepath ended the contact. Its Kind is telepathic_contact_ended.
type TelepathicContactEnded struct{ telepathyEventBase }

// NewTelepathicContactEnded returns the event. It returns an error matching
// srderr.ErrInvalidInput for a blank id or equal ids. It never panics.
func NewTelepathicContactEnded(telepath, contacted CombatantID) (TelepathicContactEnded, error) {
	b, err := newTelepathyBase(telepath, contacted)
	return TelepathicContactEnded{b}, err
}

// Kind returns "telepathic_contact_ended".
func (TelepathicContactEnded) Kind() string { return "telepathic_contact_ended" }
