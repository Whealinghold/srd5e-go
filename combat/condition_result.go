package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// conditionUnchanged is the part of a result that changed nothing: the set as it
// was.
type conditionUnchanged struct{ set ConditionSet }

// Set returns the set after the call.
func (u conditionUnchanged) Set() ConditionSet { return u.set }

// conditionChange is the part of a result that changed the set: the new set and
// the events in the order they happened.
type conditionChange struct {
	set    ConditionSet
	events []ConditionEvent
}

// Set returns the set after the call.
func (c conditionChange) Set() ConditionSet { return c.set }

// Events returns a copy of the events, in the order they happened.
func (c conditionChange) Events() []ConditionEvent { return slices.Clone(c.events) }

// ApplyConditionToSetResult is the closed set of outcomes of applying a
// condition. Variants: ConditionAdded, ConditionAlreadyHeld, ConditionImmune,
// ExhaustionAtMaximum. Every variant reports the Set after the call; only
// ConditionAdded carries events.
type ApplyConditionToSetResult interface {
	isApplyConditionToSetResult()
	Set() ConditionSet
}

// ConditionAdded: the condition, or one more Exhaustion level, was added.
type ConditionAdded struct{ conditionChange }

// ConditionAlreadyHeld: the kind already holds, directly or by implication, so
// nothing changed. A second source of the same kind is not kept.
type ConditionAlreadyHeld struct{ conditionUnchanged }

// ConditionImmune: the creature is immune to the kind, so nothing changed.
type ConditionImmune struct{ conditionUnchanged }

// ExhaustionAtMaximum: the creature is at Exhaustion level 6, so nothing changed.
type ExhaustionAtMaximum struct{ conditionUnchanged }

func (ConditionAdded) isApplyConditionToSetResult()       {}
func (ConditionAlreadyHeld) isApplyConditionToSetResult() {}
func (ConditionImmune) isApplyConditionToSetResult()      {}
func (ExhaustionAtMaximum) isApplyConditionToSetResult()  {}

// RemoveConditionFromSetResult is the closed set of outcomes of removing a
// condition. Variants: ConditionCleared, ExhaustionLowered,
// ConditionRemovalNotPresent, ConditionRemovalImpliedBy. Every variant reports
// the Set after the call.
type RemoveConditionFromSetResult interface {
	isRemoveConditionFromSetResult()
	Set() ConditionSet
}

// ConditionCleared: the explicit condition was removed. The kind may still hold
// by implication.
type ConditionCleared struct{ conditionChange }

// ExhaustionLowered: Exhaustion went down one level and is still above level 0.
type ExhaustionLowered struct{ conditionChange }

// ConditionRemovalNotPresent: the creature does not have the kind, so nothing
// changed.
type ConditionRemovalNotPresent struct{ conditionUnchanged }

// ConditionRemovalImpliedBy: the kind holds only because other conditions imply
// it, so nothing changed. Remove those instead.
type ConditionRemovalImpliedBy struct {
	conditionUnchanged
	impliedBy []content.ConditionKind
}

// ImpliedBy returns a copy of the explicit kinds that imply the removed kind, in
// the order of content.ConditionKind.
func (r ConditionRemovalImpliedBy) ImpliedBy() []content.ConditionKind {
	return slices.Clone(r.impliedBy)
}

func (ConditionCleared) isRemoveConditionFromSetResult()           {}
func (ExhaustionLowered) isRemoveConditionFromSetResult()          {}
func (ConditionRemovalNotPresent) isRemoveConditionFromSetResult() {}
func (ConditionRemovalImpliedBy) isRemoveConditionFromSetResult()  {}

// ReduceExhaustionOnSetResult is the closed set of outcomes of reducing
// Exhaustion. Variants: ExhaustionReduced, ExhaustionEnded,
// ExhaustionReductionNotPresent. Every variant reports the Set after the call.
type ReduceExhaustionOnSetResult interface {
	isReduceExhaustionOnSetResult()
	Set() ConditionSet
}

// ExhaustionReduced: Exhaustion went down and is still above level 0.
type ExhaustionReduced struct{ conditionChange }

// ExhaustionEnded: Exhaustion reached level 0 and the condition ended.
type ExhaustionEnded struct{ conditionChange }

// ExhaustionReductionNotPresent: the creature has no Exhaustion, so nothing
// changed.
type ExhaustionReductionNotPresent struct{ conditionUnchanged }

func (ExhaustionReduced) isReduceExhaustionOnSetResult()             {}
func (ExhaustionEnded) isReduceExhaustionOnSetResult()               {}
func (ExhaustionReductionNotPresent) isReduceExhaustionOnSetResult() {}
