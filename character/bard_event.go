package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// BardEvent is something that happened to a Bard, in the past tense and immutable, the closed family of Bard events.
// Every one is also an Event and carries the Identifier it concerns. The interface cannot be implemented outside
// this package. Members: BardicInspirationGranted, BardicInspirationExpended, BardicInspirationExpired,
// BardicInspirationRestored, BardExpertiseGained.
type BardEvent interface {
	Event
	isBardEvent()
}

// The Bard feature names of BardicInspirationExpended.Feature.
const (
	bardFeatureInspiration = "bardic_inspiration"
	bardFeatureCutting     = "cutting_words"
	bardFeaturePeerless    = "peerless_skill"
)

// BardicInspirationGranted is the event of a Bardic Inspiration die given to another creature.
type BardicInspirationGranted struct {
	id        Identifier
	die       BardicDie
	remaining int
}

// CharacterID returns the id of the Bard.
func (e BardicInspirationGranted) CharacterID() Identifier { return e.id }

// Die returns the die granted.
func (e BardicInspirationGranted) Die() BardicDie { return e.die }

// Remaining returns the Bard's uses left after the grant.
func (e BardicInspirationGranted) Remaining() int { return e.remaining }

// BardicInspirationExpended is the event of a Bardic Inspiration die being rolled, by a recipient or through Cutting Words or Peerless Skill.
type BardicInspirationExpended struct {
	id      Identifier
	feature string
	die     BardicDie
	rolled  int
}

// CharacterID returns the id of the creature that rolled the die (the Bard for Cutting Words and Peerless Skill).
func (e BardicInspirationExpended) CharacterID() Identifier { return e.id }

// Feature returns "bardic_inspiration", "cutting_words" or "peerless_skill".
func (e BardicInspirationExpended) Feature() string { return e.feature }

// Die returns the die rolled.
func (e BardicInspirationExpended) Die() BardicDie { return e.die }

// Rolled returns the number rolled.
func (e BardicInspirationExpended) Rolled() int { return e.rolled }

// BardicInspirationExpired is the event of a Bardic Inspiration die lost because more than an hour passed since it was granted.
type BardicInspirationExpired struct {
	id  Identifier
	die BardicDie
}

// CharacterID returns the id of the creature that held the die.
func (e BardicInspirationExpired) CharacterID() Identifier { return e.id }

// Die returns the die lost.
func (e BardicInspirationExpired) Die() BardicDie { return e.die }

// BardicInspirationRestored is the event of expended uses of Bardic Inspiration regained.
type BardicInspirationRestored struct {
	id        Identifier
	uses      int
	remaining int
}

// CharacterID returns the id of the Bard.
func (e BardicInspirationRestored) CharacterID() Identifier { return e.id }

// Uses returns how many uses were regained.
func (e BardicInspirationRestored) Uses() int { return e.uses }

// Remaining returns the uses left after the recovery.
func (e BardicInspirationRestored) Remaining() int { return e.remaining }

// BardExpertiseGained is the event of Expertise gained at Bard level 2 or 9.
type BardExpertiseGained struct {
	id     Identifier
	skills []content.Skill
}

// CharacterID returns the id of the Bard.
func (e BardExpertiseGained) CharacterID() Identifier { return e.id }

// Skills returns the two skills, as a new slice.
func (e BardExpertiseGained) Skills() []content.Skill { return slices.Clone(e.skills) }

func (BardicInspirationGranted) isEvent()      {}
func (BardicInspirationExpended) isEvent()     {}
func (BardicInspirationExpired) isEvent()      {}
func (BardicInspirationRestored) isEvent()     {}
func (BardExpertiseGained) isEvent()           {}
func (BardicInspirationGranted) isBardEvent()  {}
func (BardicInspirationExpended) isBardEvent() {}
func (BardicInspirationExpired) isBardEvent()  {}
func (BardicInspirationRestored) isBardEvent() {}
func (BardExpertiseGained) isBardEvent()       {}
