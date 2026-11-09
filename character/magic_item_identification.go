package character

// IdentificationMethod is how an item's properties are learned. The zero value is invalid.
type IdentificationMethod int

// The methods (SRD 5.2.1, "Identifying a Magic Item", p. 102).
const (
	IdentifyByShortRestFocus IdentificationMethod = iota + 1
	IdentifyBySpell
)

// CurseKnowledge is what identification revealed of a curse.
type CurseKnowledge int

// CurseNotRevealed: identification never reveals a curse, whether or not the item bears one.
const CurseNotRevealed CurseKnowledge = 1

// IdentificationNotLearnedReason says why an item's properties were not learned.
// The zero value is not a reason.
type IdentificationNotLearnedReason int

// The reasons.
const (
	IdentifyWrongRestPurpose IdentificationNotLearnedReason = iota + 1
	IdentifyRestNotFocusedOnItem
	IdentifyRestInterrupted
)

// IdentificationOutcome is the closed set of results of identification.
// Variants: MagicItemPropertiesLearned, MagicItemPropertiesNotLearned.
type IdentificationOutcome interface{ isIdentificationOutcome() }

// MagicItemPropertiesLearned: the properties are known; Curse is CurseNotRevealed.
type MagicItemPropertiesLearned struct {
	Method IdentificationMethod
	Curse  CurseKnowledge
}

// MagicItemPropertiesNotLearned: nothing was learned, for Reason.
type MagicItemPropertiesNotLearned struct {
	Reason IdentificationNotLearnedReason
}

func (MagicItemPropertiesLearned) isIdentificationOutcome()    {}
func (MagicItemPropertiesNotLearned) isIdentificationOutcome() {}

// MagicItemIdentification identifies items. The zero value is ready to use.
type MagicItemIdentification struct{}

// ByShortRest learns the properties by a completed Short Rest focused on the one
// item, in physical contact, with purpose RestPurposeLearnProperties; otherwise
// it reports the first of wrong purpose, a different instance or no contact, an
// interrupted rest.
func (MagicItemIdentification) ByShortRest(instance MagicItemInstanceID, focus ShortRestFocus) IdentificationOutcome {
	switch {
	case focus.purpose != RestPurposeLearnProperties:
		return MagicItemPropertiesNotLearned{Reason: IdentifyWrongRestPurpose}
	case focus.instance != instance || focus.contact != InContact:
		return MagicItemPropertiesNotLearned{Reason: IdentifyRestNotFocusedOnItem}
	case !focus.isCompleted:
		return MagicItemPropertiesNotLearned{Reason: IdentifyRestInterrupted}
	}
	return MagicItemPropertiesLearned{Method: IdentifyByShortRestFocus, Curse: CurseNotRevealed}
}

// BySpell learns the properties with the Identify spell, which does not reveal a
// curse (SRD 5.2.1, "Cursed Items", p. 206).
func (MagicItemIdentification) BySpell() IdentificationOutcome {
	return MagicItemPropertiesLearned{Method: IdentifyBySpell, Curse: CurseNotRevealed}
}
