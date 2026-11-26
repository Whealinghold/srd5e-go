package combat

import "slices"

// InitiateTelepathicContactCommand is the host's facts for starting a contact.
// Construct it with a keyed literal. Its zero value is invalid and is rejected as
// TelepathicContactRejectionInvalidCommand. Sight is deliberately absent: a telepath
// need not see the contacted creature (SRD 5.2.1, "Telepathy", p. 190).
type InitiateTelepathicContactCommand struct {
	// State is the telepath's current contact (the zero value is NoContact).
	State TelepathicContactState
	// Telepath and Contacted are the two creatures.
	Telepath, Contacted CombatantID
	// Capability is the telepath's telepathy.
	Capability TelepathyCapability
	// TelepathConditions and ContactedConditions are the creatures' conditions.
	TelepathConditions, ContactedConditions ConditionSet
	// DistanceFeet is the distance between them, 0 or more.
	DistanceFeet int
	// Understanding is what the contacted creature understands.
	Understanding ContactedUnderstanding
	// ReplyPolicy is stored in the contact; the zero value is ContactedMayReply.
	ReplyPolicy ReplyPolicy
}

// InitiateTelepathicContactResult is the closed set of answers of
// InitiateTelepathicContact. Variants: InitiateTelepathicContactMade,
// InitiateTelepathicContactRejected, InitiateTelepathicContactAlreadyMade.
type InitiateTelepathicContactResult interface{ isInitiateTelepathicContactResult() }

// InitiateTelepathicContactMade: the contact exists. Events holds
// TelepathicContactBroken (when it replaced a contact with another creature) and
// then TelepathicContactMade.
type InitiateTelepathicContactMade struct {
	state  TelepathicContactState
	events []TelepathyEvent
}

// State returns the new contact state.
func (r InitiateTelepathicContactMade) State() TelepathicContactState { return r.state }

// Events returns a copy of the events, in order.
func (r InitiateTelepathicContactMade) Events() []TelepathyEvent { return slices.Clone(r.events) }

// InitiateTelepathicContactRejected: no contact was made and the state is unchanged.
type InitiateTelepathicContactRejected struct{ Reason TelepathicContactRejection }

// InitiateTelepathicContactAlreadyMade: the telepath is already in contact with that creature; nothing changed.
type InitiateTelepathicContactAlreadyMade struct{}

func (InitiateTelepathicContactMade) isInitiateTelepathicContactResult()        {}
func (InitiateTelepathicContactRejected) isInitiateTelepathicContactResult()    {}
func (InitiateTelepathicContactAlreadyMade) isInitiateTelepathicContactResult() {}

// InitiateTelepathicContact starts a contact, taking no action (SRD 5.2.1,
// "Telepathy", p. 190). Rejections are reported in the order of
// TelepathicContactRejection: invalid command, initiator not telepathic, initiator
// Incapacitated, contacted Incapacitated, out of range (the range is inclusive),
// contacted cannot understand. Contact with the creature already in contact is
// AlreadyMade; contact with another creature breaks the old contact first. It
// returns a result for every input, never an error, and never panics.
func InitiateTelepathicContact(c InitiateTelepathicContactCommand) InitiateTelepathicContactResult {
	reject := func(r TelepathicContactRejection) InitiateTelepathicContactResult {
		return InitiateTelepathicContactRejected{r}
	}
	switch {
	case c.Telepath.IsZero() || c.Contacted.IsZero() || c.Telepath == c.Contacted || c.DistanceFeet < 0 || !c.ReplyPolicy.valid() ||
		(c.State.IsInContact() && c.State.telepath != c.Telepath):
		return reject(TelepathicContactRejectionInvalidCommand)
	case !c.Capability.IsTelepathic():
		return reject(TelepathicContactRejectionInitiatorNotTelepathic)
	case (ConditionEffects{}).IsIncapacitated(c.TelepathConditions):
		return reject(TelepathicContactRejectionInitiatorIncapacitated)
	case (ConditionEffects{}).IsIncapacitated(c.ContactedConditions):
		return reject(TelepathicContactRejectionContactedIncapacitated)
	case c.DistanceFeet > c.Capability.RangeFeet():
		return reject(TelepathicContactRejectionContactedOutOfRange)
	case !c.Understanding.understands():
		return reject(TelepathicContactRejectionContactedCannotUnderstand)
	case c.State.IsInContact() && c.State.contacted == c.Contacted:
		return InitiateTelepathicContactAlreadyMade{}
	}
	var events []TelepathyEvent
	if c.State.IsInContact() {
		events = append(events, TelepathicContactBroken{telepathyEventBase{c.Telepath, c.State.contacted}, BreakCauseTelepathContactedAnother})
	}
	events = append(events, TelepathicContactMade{telepathyEventBase{c.Telepath, c.Contacted}})
	return InitiateTelepathicContactMade{InContact(c.Telepath, c.Contacted, c.Capability.RangeFeet(), c.ReplyPolicy), events}
}

// EndTelepathicContactResult is the closed set of answers of EndTelepathicContact.
// Variants: EndTelepathicContactApplied, EndTelepathicContactNoContact.
type EndTelepathicContactResult interface{ isEndTelepathicContactResult() }

// EndTelepathicContactApplied: the contact ended. The new state is NoContact and
// Events holds one TelepathicContactEnded.
type EndTelepathicContactApplied struct{ events []TelepathyEvent }

// State returns NoContact.
func (EndTelepathicContactApplied) State() TelepathicContactState { return TelepathicContactState{} }

// Events returns a copy of the events.
func (r EndTelepathicContactApplied) Events() []TelepathyEvent { return slices.Clone(r.events) }

// EndTelepathicContactNoContact: the creature has no contact to end; nothing changed.
type EndTelepathicContactNoContact struct{}

func (EndTelepathicContactApplied) isEndTelepathicContactResult()   {}
func (EndTelepathicContactNoContact) isEndTelepathicContactResult() {}

// EndTelepathicContact ends the contact the telepath holds, taking no action (SRD
// 5.2.1, "Telepathy", p. 190). A state with no contact, or one held by another
// telepath, is NoContact. It never panics.
func EndTelepathicContact(state TelepathicContactState, telepath CombatantID) EndTelepathicContactResult {
	if !state.IsInContact() || state.telepath != telepath {
		return EndTelepathicContactNoContact{}
	}
	return EndTelepathicContactApplied{[]TelepathyEvent{TelepathicContactEnded{telepathyEventBase{state.telepath, state.contacted}}}}
}

// ReportTelepathIncapacitatedResult is the closed set of answers of
// ReportTelepathIncapacitated. Variants: ReportTelepathIncapacitatedBroken,
// ReportTelepathIncapacitatedUnchanged.
type ReportTelepathIncapacitatedResult interface{ isReportTelepathIncapacitatedResult() }

// ReportTelepathIncapacitatedBroken: the contact broke. The new state is
// NoContact and Events holds one TelepathicContactBroken with BreakCauseIncapacitated.
type ReportTelepathIncapacitatedBroken struct{ events []TelepathyEvent }

// State returns NoContact.
func (ReportTelepathIncapacitatedBroken) State() TelepathicContactState {
	return TelepathicContactState{}
}

// Events returns a copy of the events.
func (r ReportTelepathIncapacitatedBroken) Events() []TelepathyEvent { return slices.Clone(r.events) }

// ReportTelepathIncapacitatedUnchanged: the creature is not part of a contact, or is not Incapacitated; nothing changed.
type ReportTelepathIncapacitatedUnchanged struct{}

func (ReportTelepathIncapacitatedBroken) isReportTelepathIncapacitatedResult()    {}
func (ReportTelepathIncapacitatedUnchanged) isReportTelepathIncapacitatedResult() {}

// ReportTelepathIncapacitated tells the contact that a creature's conditions
// changed. A contact breaks at once when either creature of the pair has the
// Incapacitated condition (SRD 5.2.1, "Telepathy", p. 190; "Incapacitated", p. 184).
// A creature outside the pair, a creature that is not Incapacitated, or NoContact
// is Unchanged. It never panics.
func ReportTelepathIncapacitated(state TelepathicContactState, creature CombatantID, conditions ConditionSet) ReportTelepathIncapacitatedResult {
	if !state.IsInPair(creature) || !(ConditionEffects{}).IsIncapacitated(conditions) {
		return ReportTelepathIncapacitatedUnchanged{}
	}
	return ReportTelepathIncapacitatedBroken{[]TelepathyEvent{TelepathicContactBroken{telepathyEventBase{state.telepath, state.contacted}, BreakCauseIncapacitated}}}
}

// ReportTelepathDistanceResult is the closed set of answers of
// ReportTelepathDistance. Variants: ReportTelepathDistanceBroken,
// ReportTelepathDistanceUnchanged.
type ReportTelepathDistanceResult interface{ isReportTelepathDistanceResult() }

// ReportTelepathDistanceBroken: the contacted creature left the range. The new
// state is NoContact and Events holds one TelepathicContactBroken with BreakCauseContactedLeftRange.
type ReportTelepathDistanceBroken struct{ events []TelepathyEvent }

// State returns NoContact.
func (ReportTelepathDistanceBroken) State() TelepathicContactState { return TelepathicContactState{} }

// Events returns a copy of the events.
func (r ReportTelepathDistanceBroken) Events() []TelepathyEvent { return slices.Clone(r.events) }

// ReportTelepathDistanceUnchanged: no contact, or the contacted creature is within range; nothing changed.
type ReportTelepathDistanceUnchanged struct{}

func (ReportTelepathDistanceBroken) isReportTelepathDistanceResult()    {}
func (ReportTelepathDistanceUnchanged) isReportTelepathDistanceResult() {}

// ReportTelepathDistance tells the contact the current distance between the pair.
// The contact breaks when the contacted creature is beyond the range (SRD 5.2.1,
// "Telepathy", p. 190); a distance equal to the range keeps it. A negative
// distance cannot exist and breaks nothing. It never panics.
func ReportTelepathDistance(state TelepathicContactState, distanceFeet int) ReportTelepathDistanceResult {
	if !state.IsInContact() || distanceFeet <= state.rangeFeet {
		return ReportTelepathDistanceUnchanged{}
	}
	return ReportTelepathDistanceBroken{[]TelepathyEvent{TelepathicContactBroken{telepathyEventBase{state.telepath, state.contacted}, BreakCauseContactedLeftRange}}}
}

// TelepathicReplyResult is the closed set of answers of TelepathicReply. Variants:
// TelepathicReplyAllowed, TelepathicReplyNotPermitted, TelepathicReplyNoContact,
// TelepathicReplyNotContacted.
type TelepathicReplyResult interface{ isTelepathicReplyResult() }

// TelepathicReplyAllowed: the contacted creature may reply mentally.
type TelepathicReplyAllowed struct{}

// TelepathicReplyNotPermitted: the contact's ReplyPolicy forbids a reply.
type TelepathicReplyNotPermitted struct{}

// TelepathicReplyNoContact: there is no contact.
type TelepathicReplyNoContact struct{}

// TelepathicReplyNotContacted: there is a contact, but the replier is not its contacted creature.
type TelepathicReplyNotContacted struct{}

func (TelepathicReplyAllowed) isTelepathicReplyResult()      {}
func (TelepathicReplyNotPermitted) isTelepathicReplyResult() {}
func (TelepathicReplyNoContact) isTelepathicReplyResult()    {}
func (TelepathicReplyNotContacted) isTelepathicReplyResult() {}

// TelepathicReply says whether replier may reply mentally to the telepath: only the
// contacted creature of an existing contact, and only when the ReplyPolicy allows
// it, whether or not it is telepathic (SRD 5.2.1, "Telepathy", p. 190). It never panics.
func TelepathicReply(state TelepathicContactState, replier CombatantID) TelepathicReplyResult {
	switch {
	case !state.IsInContact():
		return TelepathicReplyNoContact{}
	case replier != state.contacted:
		return TelepathicReplyNotContacted{}
	case state.policy == ContactedMayNotReply:
		return TelepathicReplyNotPermitted{}
	}
	return TelepathicReplyAllowed{}
}
