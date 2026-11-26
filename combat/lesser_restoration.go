package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// CastLesserRestorationResult is the closed set of answers of ApplyLesserRestoration.
// Variants: LesserRestorationApplied, LesserRestorationNothingToEnd,
// LesserRestorationWrongKind.
type CastLesserRestorationResult interface{ isCastLesserRestorationResult() }

// LesserRestorationApplied: the one condition ended. Set is the condition set after
// the call and Events holds the ConditionRemoved event.
type LesserRestorationApplied struct {
	set    ConditionSet
	events []Event
}

// Set returns the condition set after the call.
func (r LesserRestorationApplied) Set() ConditionSet { return r.set }

// Events returns a copy of the events in the order they happened.
func (r LesserRestorationApplied) Events() []Event { return slices.Clone(r.events) }

// LesserRestorationNothingToEnd: the creature does not hold that condition as an
// explicit condition, so nothing changed and no event was emitted. When another
// held condition only implies the kind, ImpliedBy lists those kinds.
type LesserRestorationNothingToEnd struct {
	set       ConditionSet
	impliedBy []content.ConditionKind
}

// Set returns the unchanged condition set.
func (r LesserRestorationNothingToEnd) Set() ConditionSet { return r.set }

// ImpliedBy returns a copy of the held kinds that imply the requested kind, empty
// when none does.
func (r LesserRestorationNothingToEnd) ImpliedBy() []content.ConditionKind {
	return slices.Clone(r.impliedBy)
}

// LesserRestorationWrongKind: Lesser Restoration cannot end that kind; nothing changed.
// Construct it with a keyed literal.
type LesserRestorationWrongKind struct{ Kind content.ConditionKind }

func (LesserRestorationApplied) isCastLesserRestorationResult()      {}
func (LesserRestorationNothingToEnd) isCastLesserRestorationResult() {}
func (LesserRestorationWrongKind) isCastLesserRestorationResult()    {}

// ApplyLesserRestoration ends one condition on the creature subject, of kind Blinded,
// Deafened, Paralyzed or Poisoned (SRD 5.2.1, "Lesser Restoration", p. 144), through
// ConditionSet.Remove. Any other kind gives LesserRestorationWrongKind; a kind not
// held as an explicit condition (a zero subject included) gives
// LesserRestorationNothingToEnd. It never panics and does not change set.
func ApplyLesserRestoration(subject CombatantID, set ConditionSet, kind content.ConditionKind) CastLesserRestorationResult {
	if !slices.Contains([]content.ConditionKind{content.ConditionBlinded, content.ConditionDeafened, content.ConditionParalyzed, content.ConditionPoisoned}, kind) {
		return LesserRestorationWrongKind{kind}
	}
	res := set.Remove(subject, kind)
	if c, ok := res.(ConditionCleared); ok {
		return LesserRestorationApplied{c.set, conditionEventList(c.events)}
	}
	return LesserRestorationNothingToEnd{set, impliedByOf(res)}
}

// impliedByOf returns the kinds that imply the kind of a Remove result that only
// held by implication, nil for any other result.
func impliedByOf(res RemoveConditionFromSetResult) []content.ConditionKind {
	if r, ok := res.(ConditionRemovalImpliedBy); ok {
		return r.ImpliedBy()
	}
	return nil
}

// conditionEventList returns condition events as Events.
func conditionEventList(events []ConditionEvent) []Event {
	var out []Event
	for _, e := range events {
		out = append(out, e)
	}
	return out
}
