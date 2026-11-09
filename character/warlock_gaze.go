package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

const gazeOfTwoMinds content.FeatureOptionID = "gaze_of_two_minds"

// GazeWillingness says whether the creature is willing, a closed set. The zero value is invalid and read as not willing.
type GazeWillingness int

const (
	// GazeWilling: the creature is willing.
	GazeWilling GazeWillingness = iota + 1
	// GazeUnwilling: the creature is not.
	GazeUnwilling
)

// GazeContact says whether the Warlock touches the creature, a closed set. The zero value is invalid and read as not touching.
type GazeContact int

const (
	// GazeTouching: the Warlock touches the creature.
	GazeTouching GazeContact = iota + 1
	// GazeNotTouching: it does not.
	GazeNotTouching
)

// GazePlaneRelation says whether the creature is on the Warlock's plane of existence, a closed set. The zero value is invalid and read as a
// different plane.
type GazePlaneRelation int

const (
	// GazeSamePlane: the creature is on the same plane.
	GazeSamePlane GazePlaneRelation = iota + 1
	// GazeDifferentPlane: it is not.
	GazeDifferentPlane
)

// GazeTargetFacts are the facts about the creature a Gaze of Two Minds link begins with, supplied by the host. Construct it with a keyed
// literal.
type GazeTargetFacts struct {
	Creature    Identifier
	Willingness GazeWillingness
	Contact     GazeContact
}

// UseGazeOfTwoMinds begins or maintains the link with a creature as a Bonus Action (class-warlock.md FR-069 to FR-072, FR-151). With no link
// it begins one with target, which must be willing and touched; with an active link it extends it to the end of the Warlock's next turn if
// the creature (the one already linked) is on the same plane. A lapsed link cannot be maintained: pass the zero SensorLink to begin a new one.
// The link's senses and casting reach are read with SensorLink.LinkedSenses and SensorLink.CanCastFromLinkedSpace. It refuses with
// WarlockInvocationNotKnown (Option Gaze of Two Minds), WarlockLinkEnded, WarlockNotOnSamePlane, WarlockNotWilling, WarlockNotTouched and
// WarlockNoBonusAction.
func UseGazeOfTwoMinds(id Identifier, s WarlockState, pact WarlockPactState, target GazeTargetFacts, plane GazePlaneRelation, turn WarlockTurnFacts) WarlockPactTransition {
	link := pact.link
	switch {
	case !s.Knows(gazeOfTwoMinds):
		return pactNotKnown(gazeOfTwoMinds)
	case link.status == GazeLinkLapsed:
		return pactRefused(warlockRefuse(WarlockLinkEnded))
	case link.Active() && plane != GazeSamePlane:
		return pactRefused(warlockRefuse(WarlockNotOnSamePlane))
	case !link.Active() && target.Willingness != GazeWilling:
		return pactRefused(warlockRefuse(WarlockNotWilling))
	case !link.Active() && (target.Contact != GazeTouching):
		return pactRefused(warlockRefuse(WarlockNotTouched))
	case turn.BonusAction == ActionSpent:
		return pactRefused(warlockRefuse(WarlockNoBonusAction))
	}
	if link.Active() {
		return pactChanged(pact, GazeLinkMaintained{id: id, creature: link.creature})
	}
	pact.link = SensorLink{creature: target.Creature, status: GazeLinkActive}
	return pactChanged(pact, GazeLinkBegun{id: id, creature: target.Creature})
}
