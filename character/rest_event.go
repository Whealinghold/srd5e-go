package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// RestEvent is the closed family of events of rests and Hit Point Dice. It is an
// Event, so a host switches over it for the rest family and over Event for the
// context. Events are past tense and immutable. Members: RestApplied,
// RestInterrupted, HitPointDieSpent.
type RestEvent interface {
	Event
	isRestEvent()
}

// RestApplied is the event of the benefits of a rest being applied: the kind
// whose benefits they are (ShortRest for an interrupted Long Rest), the source
// and the effects. The host applies the effects.
type RestApplied struct {
	id         Identifier
	benefitsOf RestKind
	source     RestSource
	effects    []RestEffect
}

// CharacterID returns the id of the Character.
func (e RestApplied) CharacterID() Identifier { return e.id }

// BenefitsOf returns the kind of rest whose benefits were applied.
func (e RestApplied) BenefitsOf() RestKind { return e.benefitsOf }

// Source returns RestFinished, or LongRestInterrupted for an interrupted Long
// Rest that earned the benefits of a Short Rest.
func (e RestApplied) Source() RestSource { return e.source }

// Effects returns a new slice of the effects, in order.
func (e RestApplied) Effects() []RestEffect { return slices.Clone(e.effects) }

// RestBenefitsGranted says what an interrupted rest still conferred. The zero
// value is invalid.
type RestBenefitsGranted int

// The benefits an interrupted rest can leave.
const (
	// RestBenefitsOfShortRest: an interrupted Long Rest granted the benefits of a
	// Short Rest.
	RestBenefitsOfShortRest RestBenefitsGranted = iota + 1
	// RestBenefitsNone: nothing was granted.
	RestBenefitsNone
)

// RestInterrupted is the event of a rest being stopped: the kind of rest that
// was reported, what stopped it and what it still granted. A stop after the
// required hours is a finished rest and raises RestApplied only.
type RestInterrupted struct {
	id              Identifier
	kind            RestKind
	interruption    RestInterruption
	benefitsGranted RestBenefitsGranted
}

// CharacterID returns the id of the Character.
func (e RestInterrupted) CharacterID() Identifier { return e.id }

// Kind returns the kind of rest that was reported.
func (e RestInterrupted) Kind() RestKind { return e.kind }

// Interruption returns what stopped the rest.
func (e RestInterrupted) Interruption() RestInterruption { return e.interruption }

// BenefitsGranted returns what the stopped rest still granted.
func (e RestInterrupted) BenefitsGranted() RestBenefitsGranted { return e.benefitsGranted }

// HitPointDieSpent is the event of one Hit Point Die being spent: the die, the
// roll, the Constitution modifier and the Hit Points regained (not capped by the
// Hit Point maximum).
type HitPointDieSpent struct {
	id                   Identifier
	die                  content.HitPointDie
	roll                 int
	constitutionModifier ability.Modifier
	hitPointsRegained    int
}

// CharacterID returns the id of the Character.
func (e HitPointDieSpent) CharacterID() Identifier { return e.id }

// Die returns the Hit Point Die spent.
func (e HitPointDieSpent) Die() content.HitPointDie { return e.die }

// Roll returns the face rolled.
func (e HitPointDieSpent) Roll() int { return e.roll }

// ConstitutionModifier returns the Constitution modifier added to the roll.
func (e HitPointDieSpent) ConstitutionModifier() ability.Modifier { return e.constitutionModifier }

// HitPointsRegained returns the Hit Points regained, at least 1.
func (e HitPointDieSpent) HitPointsRegained() int { return e.hitPointsRegained }

func (RestApplied) isEvent()      {}
func (RestInterrupted) isEvent()  {}
func (HitPointDieSpent) isEvent() {}

func (RestApplied) isRestEvent()      {}
func (RestInterrupted) isRestEvent()  {}
func (HitPointDieSpent) isRestEvent() {}
