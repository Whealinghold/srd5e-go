package combat

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// ErrTelepathyRange is matched, with srderr.ErrInvalidInput, by the error of
// NewTelepathic for a range below 1 foot.
var ErrTelepathyRange = fmt.Errorf("combat: telepathy range below 1 foot: %w", srderr.ErrInvalidInput)

// TelepathyCapability is what a creature can do with telepathy: nothing, or start
// contact within a range in feet (SRD 5.2.1, "Telepathy", p. 190). The zero value
// is NotTelepathic, a creature without telepathy. Build a telepath with
// NewTelepathic. Immutable and comparable; safe for concurrent use.
type TelepathyCapability struct{ rangeFeet int }

// NewTelepathic returns a capability with the given range in whole feet. It returns
// an error matching ErrTelepathyRange and srderr.ErrInvalidInput for a range below
// 1. It never panics.
func NewTelepathic(rangeFeet int) (TelepathyCapability, error) {
	if rangeFeet < 1 {
		return TelepathyCapability{}, ErrTelepathyRange
	}
	return TelepathyCapability{rangeFeet}, nil
}

// MustTelepathic is NewTelepathic for literals known to be valid. It panics with
// the error NewTelepathic returns, and only then.
func MustTelepathic(rangeFeet int) TelepathyCapability {
	c, err := NewTelepathic(rangeFeet)
	if err != nil {
		panic(err)
	}
	return c
}

// IsTelepathic reports whether the creature can start telepathic contact.
func (c TelepathyCapability) IsTelepathic() bool { return c.rangeFeet > 0 }

// RangeFeet returns the range in feet, 0 for a creature without telepathy.
func (c TelepathyCapability) RangeFeet() int { return c.rangeFeet }

// ReplyPolicy says whether the contacted creature may reply mentally. The zero
// value is ContactedMayReply, the rule's default.
type ReplyPolicy int

// The reply policies.
const (
	// ContactedMayReply: the contacted creature may reply (the default).
	ContactedMayReply ReplyPolicy = iota
	// ContactedMayNotReply: the contacted creature may not reply, as a stat block can say.
	ContactedMayNotReply
)

// String returns "contacted_may_reply", "contacted_may_not_reply" or "unknown".
func (p ReplyPolicy) String() string {
	switch p {
	case ContactedMayReply:
		return "contacted_may_reply"
	case ContactedMayNotReply:
		return "contacted_may_not_reply"
	}
	return "unknown"
}

func (p ReplyPolicy) valid() bool { return p == ContactedMayReply || p == ContactedMayNotReply }

// ContactedUnderstanding is what the contacted creature understands. The zero
// value is ContactedUnderstandingNeither, which refuses contact; so does any
// undefined value.
type ContactedUnderstanding int

// The kinds of understanding (SRD 5.2.1, "Telepathy", p. 190).
const (
	// ContactedUnderstandingNeither: the creature understands no language and is not telepathic.
	ContactedUnderstandingNeither ContactedUnderstanding = iota
	// ContactedUnderstandingALanguage: the creature understands at least one language, whichever it is.
	ContactedUnderstandingALanguage
	// ContactedUnderstandingTelepathic: the creature is telepathic itself.
	ContactedUnderstandingTelepathic
)

// String returns "neither", "a_language", "telepathic" or "unknown".
func (u ContactedUnderstanding) String() string {
	switch u {
	case ContactedUnderstandingNeither:
		return "neither"
	case ContactedUnderstandingALanguage:
		return "a_language"
	case ContactedUnderstandingTelepathic:
		return "telepathic"
	}
	return "unknown"
}

func (u ContactedUnderstanding) understands() bool {
	return u == ContactedUnderstandingALanguage || u == ContactedUnderstandingTelepathic
}

// TelepathicContactRejection is why a contact was not made. The zero value is invalid.
type TelepathicContactRejection int

// The rejections, in the order InitiateTelepathicContact reports them.
const (
	// TelepathicContactRejectionInvalidCommand: the command is malformed (a blank or equal id, a negative distance, or a state of another telepath).
	TelepathicContactRejectionInvalidCommand TelepathicContactRejection = iota + 1
	// TelepathicContactRejectionInitiatorNotTelepathic: the initiator has no telepathy.
	TelepathicContactRejectionInitiatorNotTelepathic
	// TelepathicContactRejectionInitiatorIncapacitated: the initiator has the Incapacitated condition.
	TelepathicContactRejectionInitiatorIncapacitated
	// TelepathicContactRejectionContactedIncapacitated: the contacted creature has the Incapacitated condition.
	TelepathicContactRejectionContactedIncapacitated
	// TelepathicContactRejectionContactedOutOfRange: the contacted creature is beyond the range.
	TelepathicContactRejectionContactedOutOfRange
	// TelepathicContactRejectionContactedCannotUnderstand: the contacted creature neither understands a language nor is telepathic.
	TelepathicContactRejectionContactedCannotUnderstand
)

// String returns the snake_case name of r or "unknown".
func (r TelepathicContactRejection) String() string {
	return enumName([]string{"invalid_command", "initiator_not_telepathic", "initiator_incapacitated", "contacted_incapacitated",
		"contacted_out_of_range", "contacted_cannot_understand"}, int(r))
}

// BreakCause is why a telepathic contact was broken. The zero value is invalid.
type BreakCause int

// The causes (SRD 5.2.1, "Telepathy", p. 190).
const (
	// BreakCauseIncapacitated: the telepath or the contacted creature became Incapacitated.
	BreakCauseIncapacitated BreakCause = iota + 1
	// BreakCauseContactedLeftRange: the contacted creature is no longer within range.
	BreakCauseContactedLeftRange
	// BreakCauseTelepathContactedAnother: the telepath contacted a different creature.
	BreakCauseTelepathContactedAnother
)

// String returns the snake_case name of c or "unknown".
func (c BreakCause) String() string {
	return enumName([]string{"incapacitated", "contacted_left_range", "telepath_contacted_another"}, int(c))
}

// TelepathicContactState is the host-held fact of one telepath's contact: none,
// or a contacted creature, the range and the reply policy. At most one contact
// per telepath. The zero value is NoContact. Immutable and comparable (the policy
// takes part in equality); safe for concurrent use.
type TelepathicContactState struct {
	telepath, contacted CombatantID
	rangeFeet           int
	policy              ReplyPolicy
}

// InContact returns the state of a contact. For a blank id, equal ids, a range
// below 1 or an undefined policy it returns the zero state, NoContact, because no
// contact with those values exists. It never panics.
func InContact(telepath, contacted CombatantID, rangeFeet int, replyPolicy ReplyPolicy) TelepathicContactState {
	if telepath.IsZero() || contacted.IsZero() || telepath == contacted || rangeFeet < 1 || !replyPolicy.valid() {
		return TelepathicContactState{}
	}
	return TelepathicContactState{telepath, contacted, rangeFeet, replyPolicy}
}

// IsInContact reports whether a contact exists.
func (s TelepathicContactState) IsInContact() bool { return s.rangeFeet > 0 }

// IsInPair reports whether id is the telepath or the contacted creature of an existing contact.
func (s TelepathicContactState) IsInPair(id CombatantID) bool {
	return s.IsInContact() && (id == s.telepath || id == s.contacted)
}

// RangeFeet returns the range of the contact, 0 for NoContact.
func (s TelepathicContactState) RangeFeet() int { return s.rangeFeet }

// ReplyPolicy returns the reply policy of the contact, ContactedMayReply for NoContact.
func (s TelepathicContactState) ReplyPolicy() ReplyPolicy { return s.policy }

// Combatants returns a new slice [telepath, contacted], nil for NoContact.
func (s TelepathicContactState) Combatants() []CombatantID {
	if !s.IsInContact() {
		return nil
	}
	return []CombatantID{s.telepath, s.contacted}
}
