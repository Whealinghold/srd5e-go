package character

import "github.com/Whealinghold/srd5e-go/content"

// MagicItemEvent is the closed family of events of magic item rules. It is an
// Event, so a host switches over it for the family and over Event for the
// context. Events are past tense and immutable and carry the Identifier of the
// character they concern, passed in by the caller and never invented. Members:
// MagicItemAttuned, MagicItemAttunementEnded, MagicItemConsumed,
// MagicItemChargesSpent, MagicItemRecharged, SentientControlTaken,
// SentientControlEnded.
type MagicItemEvent interface {
	Event
	isMagicItemEvent()
}

// AttunementEndReason is why Attunement to an item ended. The zero value is not
// a reason.
type AttunementEndReason int

// The reasons Attunement ends.
const (
	AttunementEndedPrerequisiteNoLongerMet AttunementEndReason = iota + 1
	AttunementEndedItemTooFarAway
	AttunementEndedDied
	AttunementEndedAnotherCreatureAttuned
	AttunementEndedByShortRest
	AttunementEndedCurseBroken
)

// String returns the snake_case name of the reason, or "unknown".
func (r AttunementEndReason) String() string {
	names := []string{"prerequisite_no_longer_met", "item_too_far_away", "died", "another_creature_attuned", "ended_by_short_rest", "curse_broken"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// MagicItemAttuned is the event of a creature becoming attuned to an item.
type MagicItemAttuned struct {
	id       Identifier
	item     content.MagicItemID
	instance MagicItemInstanceID
}

// CharacterID returns the holder.
func (e MagicItemAttuned) CharacterID() Identifier { return e.id }

// ItemID returns the item.
func (e MagicItemAttuned) ItemID() content.MagicItemID { return e.item }

// InstanceID returns the particular copy.
func (e MagicItemAttuned) InstanceID() MagicItemInstanceID { return e.instance }

// MagicItemAttunementEnded is the event of Attunement ending.
type MagicItemAttunementEnded struct {
	id       Identifier
	item     content.MagicItemID
	instance MagicItemInstanceID
	reason   AttunementEndReason
}

// CharacterID returns the holder.
func (e MagicItemAttunementEnded) CharacterID() Identifier { return e.id }

// ItemID returns the item.
func (e MagicItemAttunementEnded) ItemID() content.MagicItemID { return e.item }

// InstanceID returns the particular copy.
func (e MagicItemAttunementEnded) InstanceID() MagicItemInstanceID { return e.instance }

// Reason returns why Attunement ended.
func (e MagicItemAttunementEnded) Reason() AttunementEndReason { return e.reason }

// MagicItemConsumed is the event of a consumable item being used up.
type MagicItemConsumed struct {
	id       Identifier
	instance MagicItemInstanceID
}

// CharacterID returns the holder.
func (e MagicItemConsumed) CharacterID() Identifier { return e.id }

// InstanceID returns the particular copy.
func (e MagicItemConsumed) InstanceID() MagicItemInstanceID { return e.instance }

// MagicItemChargesSpent is the event of charges being expended.
type MagicItemChargesSpent struct {
	id       Identifier
	instance MagicItemInstanceID
	spent    int
}

// CharacterID returns the holder.
func (e MagicItemChargesSpent) CharacterID() Identifier { return e.id }

// InstanceID returns the particular copy.
func (e MagicItemChargesSpent) InstanceID() MagicItemInstanceID { return e.instance }

// Spent returns the number of charges expended.
func (e MagicItemChargesSpent) Spent() int { return e.spent }

// MagicItemRecharged is the event of an item regaining charges.
type MagicItemRecharged struct {
	id       Identifier
	instance MagicItemInstanceID
	regained int
}

// CharacterID returns the holder.
func (e MagicItemRecharged) CharacterID() Identifier { return e.id }

// InstanceID returns the particular copy.
func (e MagicItemRecharged) InstanceID() MagicItemInstanceID { return e.instance }

// Regained returns the number of charges regained.
func (e MagicItemRecharged) Regained() int { return e.regained }

// SentientControlTaken is the event of a sentient item taking control of its
// bearer for a number of hours.
type SentientControlTaken struct {
	id    Identifier
	hours int
}

// CharacterID returns the bearer.
func (e SentientControlTaken) CharacterID() Identifier { return e.id }

// Hours returns how long the control lasts, 1 to 12.
func (e SentientControlTaken) Hours() int { return e.hours }

// SentientControlEnded is the event of a bearer ending a sentient item's control.
type SentientControlEnded struct {
	id       Identifier
	instance MagicItemInstanceID
}

// CharacterID returns the bearer.
func (e SentientControlEnded) CharacterID() Identifier { return e.id }

// InstanceID returns the item.
func (e SentientControlEnded) InstanceID() MagicItemInstanceID { return e.instance }

func (MagicItemAttuned) isEvent()                  {}
func (MagicItemAttunementEnded) isEvent()          {}
func (MagicItemConsumed) isEvent()                 {}
func (MagicItemChargesSpent) isEvent()             {}
func (MagicItemRecharged) isEvent()                {}
func (SentientControlTaken) isEvent()              {}
func (SentientControlEnded) isEvent()              {}
func (MagicItemAttuned) isMagicItemEvent()         {}
func (MagicItemAttunementEnded) isMagicItemEvent() {}
func (MagicItemConsumed) isMagicItemEvent()        {}
func (MagicItemChargesSpent) isMagicItemEvent()    {}
func (MagicItemRecharged) isMagicItemEvent()       {}
func (SentientControlTaken) isMagicItemEvent()     {}
func (SentientControlEnded) isMagicItemEvent()     {}
