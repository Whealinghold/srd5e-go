package combat

import "context"

// TelepathyRequest is the change of a telepathic contact a ChangeTelepathicContact call carries (telepathy.md FR-023):
// one variant for each operation of the owner. Variants: InitiateContact, EndContact, ReportIncapacitated,
// ReportDistance, ReplyToContact.
type TelepathyRequest interface{ isTelepathyRequest() }

// InitiateContact: a telepath makes contact (InitiateTelepathicContact). The ReplyPolicy of the command defaults to
// ContactedMayReply.
type InitiateContact struct {
	Command InitiateTelepathicContactCommand
}

// EndContact: the telepath ends its contact (EndTelepathicContact).
type EndContact struct {
	State    TelepathicContactState
	Telepath CombatantID
}

// ReportIncapacitated: the host reports a creature's conditions (ReportTelepathIncapacitated).
type ReportIncapacitated struct {
	State      TelepathicContactState
	Creature   CombatantID
	Conditions ConditionSet
}

// ReportDistance: the host reports the distance between the two creatures (ReportTelepathDistance).
type ReportDistance struct {
	State        TelepathicContactState
	DistanceFeet int
}

// ReplyToContact: the contacted creature replies (TelepathicReply).
type ReplyToContact struct {
	State   TelepathicContactState
	Replier CombatantID
}

func (InitiateContact) isTelepathyRequest()     {}
func (EndContact) isTelepathyRequest()          {}
func (ReportIncapacitated) isTelepathyRequest() {}
func (ReportDistance) isTelepathyRequest()      {}
func (ReplyToContact) isTelepathyRequest()      {}

// ChangeTelepathicContactCommand is the input of ChangeTelepathicContact.Run. Construct it with a keyed literal.
type ChangeTelepathicContactCommand struct{ Request TelepathyRequest }

// ChangeTelepathicContactResult is the closed set of answers of ChangeTelepathicContact.Run: the owner's own results,
// unchanged, of the five operations. Variants: InitiateTelepathicContactMade, InitiateTelepathicContactRejected,
// InitiateTelepathicContactAlreadyMade, EndTelepathicContactApplied, EndTelepathicContactNoContact,
// ReportTelepathIncapacitatedBroken, ReportTelepathIncapacitatedUnchanged, ReportTelepathDistanceBroken,
// ReportTelepathDistanceUnchanged, TelepathicReplyAllowed, TelepathicReplyNotPermitted, TelepathicReplyNoContact,
// TelepathicReplyNotContacted.
type ChangeTelepathicContactResult interface{ isChangeTelepathicContactResult() }

func (InitiateTelepathicContactMade) isChangeTelepathicContactResult()        {}
func (InitiateTelepathicContactRejected) isChangeTelepathicContactResult()    {}
func (InitiateTelepathicContactAlreadyMade) isChangeTelepathicContactResult() {}
func (EndTelepathicContactApplied) isChangeTelepathicContactResult()          {}
func (EndTelepathicContactNoContact) isChangeTelepathicContactResult()        {}
func (ReportTelepathIncapacitatedBroken) isChangeTelepathicContactResult()    {}
func (ReportTelepathIncapacitatedUnchanged) isChangeTelepathicContactResult() {}
func (ReportTelepathDistanceBroken) isChangeTelepathicContactResult()         {}
func (ReportTelepathDistanceUnchanged) isChangeTelepathicContactResult()      {}
func (TelepathicReplyAllowed) isChangeTelepathicContactResult()               {}
func (TelepathicReplyNotPermitted) isChangeTelepathicContactResult()          {}
func (TelepathicReplyNoContact) isChangeTelepathicContactResult()             {}
func (TelepathicReplyNotContacted) isChangeTelepathicContactResult()          {}

// ChangeTelepathicContact is the use case of changing a telepathic contact (telepathy.md FR-023): it wraps the five
// operations of the owner and returns the owner's result of the one asked for. The result is completed (every
// success, applied), contact_initiator_not_telepathic, contact_initiator_incapacitated, contact_contacted_incapacitated,
// contact_out_of_range, contact_cannot_understand, contact_already_made, contact_reply_not_permitted,
// contact_reply_not_contacted or contact_none (refused) or contact_invalid (a malformed initiation, invalid). Create it
// with NewChangeTelepathicContact.
type ChangeTelepathicContact struct{ in instrumenter }

// NewChangeTelepathicContact returns the use case. Options: WithTelemetry, WithClock.
func NewChangeTelepathicContact(opts ...Option) ChangeTelepathicContact {
	return ChangeTelepathicContact{newWired(opts)}
}

// Run makes the change. It returns an error matching srderr.ErrInvalidInput for a nil or unknown request, and the
// errors described for the package.
func (u ChangeTelepathicContact) Run(ctx context.Context, cmd ChangeTelepathicContactCommand) (ChangeTelepathicContactResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationChangeTelepathicContact, spanChangeTelepathicContact},
		func() (ChangeTelepathicContactResult, error) {
			switch v := cmd.Request.(type) {
			case InitiateContact:
				return InitiateTelepathicContact(v.Command).(ChangeTelepathicContactResult), nil
			case EndContact:
				return EndTelepathicContact(v.State, v.Telepath).(ChangeTelepathicContactResult), nil
			case ReportIncapacitated:
				return ReportTelepathIncapacitated(v.State, v.Creature, v.Conditions).(ChangeTelepathicContactResult), nil
			case ReportDistance:
				return ReportTelepathDistance(v.State, v.DistanceFeet).(ChangeTelepathicContactResult), nil
			case ReplyToContact:
				return TelepathicReply(v.State, v.Replier).(ChangeTelepathicContactResult), nil
			}
			return nil, invalidInput("a telepathic contact change needs a request")
		}, describeTelepathy)
}

// describeTelepathy classifies a result of ChangeTelepathicContact (telepathy.md section 7).
func describeTelepathy(r ChangeTelepathicContactResult) description {
	switch v := r.(type) {
	case InitiateTelepathicContactMade:
		return appliedWith(valueResultCompleted, widen(v.Events())...)
	case EndTelepathicContactApplied:
		return appliedWith(valueResultCompleted, widen(v.Events())...)
	case ReportTelepathIncapacitatedBroken:
		return appliedWith(valueResultCompleted, widen(v.Events())...)
	case ReportTelepathIncapacitatedUnchanged, ReportTelepathDistanceUnchanged, TelepathicReplyAllowed:
		return appliedWith(valueResultCompleted)
	case ReportTelepathDistanceBroken:
		return appliedWith(valueResultCompleted, widen(v.Events())...)
	case InitiateTelepathicContactRejected:
		if v.Reason == TelepathicContactRejectionInvalidCommand {
			return invalidWith(valueResultContactInvalid)
		}
		return refusedWith(contactRefusalName(v.Reason))
	case InitiateTelepathicContactAlreadyMade:
		return refusedWith(valueResultContactAlreadyMade)
	case TelepathicReplyNotPermitted:
		return refusedWith(valueResultContactReplyNotPermitted)
	case TelepathicReplyNotContacted:
		return refusedWith(valueResultContactReplyNotContacted)
	case EndTelepathicContactNoContact, TelepathicReplyNoContact:
		return refusedWith(valueResultContactNone)
	default:
		panic(unreachableError("telepathic contact result", r))
	}
}

// contactRefusalName is the srd5e.combat.result value of a refused initiation.
func contactRefusalName(r TelepathicContactRejection) string {
	switch r {
	case TelepathicContactRejectionInitiatorNotTelepathic:
		return valueResultContactInitiatorNotTelepathic
	case TelepathicContactRejectionInitiatorIncapacitated:
		return valueResultContactInitiatorIncapacitated
	case TelepathicContactRejectionContactedIncapacitated:
		return valueResultContactContactedIncapacitated
	case TelepathicContactRejectionContactedOutOfRange:
		return valueResultContactOutOfRange
	case TelepathicContactRejectionContactedCannotUnderstand:
		return valueResultContactCannotUnderstand
	case TelepathicContactRejectionInvalidCommand: // classified invalid by describeTelepathy; named for the exhaustive switch
		return valueResultContactInvalid
	default:
		panic(unreachableError("telepathic contact rejection", r))
	}
}
