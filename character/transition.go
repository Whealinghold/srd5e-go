package character

import "slices"

// Transition is the closed outcome of a Character transition that a rule can
// refuse: R is the closed rejection family of the operation. Variants:
// TransitionChanged and TransitionRejected.
type Transition[R any] interface{ isTransition() }

// NoRejection is the R of a transition no rule can refuse.
type NoRejection struct{}

// TransitionChanged is the outcome of a transition that took effect: the new
// Character and the events it produced, in order (possibly none). Only the
// transitions build one.
type TransitionChanged[R any] struct {
	character Character
	events    []Event
}

// Character returns the new Character.
func (t TransitionChanged[R]) Character() Character { return t.character }

// Events returns a new slice of the events, in order.
func (t TransitionChanged[R]) Events() []Event { return slices.Clone(t.events) }

// TransitionRejected is the outcome of a transition a rule refused; the input
// Character is unchanged and no events were produced.
type TransitionRejected[R any] struct{ rejection R }

// Rejection returns the reason the transition was refused.
func (t TransitionRejected[R]) Rejection() R { return t.rejection }

func (TransitionChanged[R]) isTransition()  {}
func (TransitionRejected[R]) isTransition() {}

func changed[R any](c Character, events ...Event) TransitionChanged[R] {
	return TransitionChanged[R]{character: c, events: events}
}

func rejected[R any](r R) TransitionRejected[R] { return TransitionRejected[R]{rejection: r} }
