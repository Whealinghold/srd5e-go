package combat

import "github.com/Whealinghold/srd5e-go/content"

// ConditionEvent is the closed family of events the condition rules produce.
// Each is an immutable value about one creature (its Combatants is exactly
// [target]) with a constant snake_case Kind. Variants: ConditionApplied,
// ConditionRemoved, ConditionLevelReduced. Each embeds conditionEventBase, which
// carries the target and the markers. Consumers read the set a result returns
// and do not rebuild it from events alone.
type ConditionEvent interface {
	Event
	conditionEvent()
}

// conditionEventBase carries the target every condition event is about and the
// markers that put the event in the Event and ConditionEvent families.
type conditionEventBase struct{ subject CombatantID }

func (conditionEventBase) event()          {}
func (conditionEventBase) conditionEvent() {}

// Combatants returns a new slice holding the target.
func (b conditionEventBase) Combatants() []CombatantID { return []CombatantID{b.subject} }

func newConditionBase(target CombatantID) (conditionEventBase, error) {
	if target.IsZero() {
		return conditionEventBase{}, invalidInput("event without a target")
	}
	return conditionEventBase{target}, nil
}

// ConditionApplied: the condition was applied to the creature. For Exhaustion it
// holds the level reached. Its Kind is condition_applied. Comparable with ==.
type ConditionApplied struct {
	conditionEventBase
	condition Condition
}

// NewConditionApplied returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero target or an invalid condition.
func NewConditionApplied(target CombatantID, c Condition) (ConditionApplied, error) {
	b, err := newConditionBase(target)
	if err != nil || !validCondition(c) {
		return ConditionApplied{}, invalidInput("condition applied needs a target and a valid condition")
	}
	return ConditionApplied{b, c}, nil
}

// Kind returns "condition_applied".
func (ConditionApplied) Kind() string { return "condition_applied" }

// Condition returns the condition that was applied.
func (e ConditionApplied) Condition() Condition { return e.condition }

// ConditionRemoved: the condition of the kind ended on the creature. Its Kind is
// condition_removed. Comparable with ==.
type ConditionRemoved struct {
	conditionEventBase
	kind content.ConditionKind
}

// NewConditionRemoved returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero target or an invalid kind.
func NewConditionRemoved(target CombatantID, kind content.ConditionKind) (ConditionRemoved, error) {
	b, err := newConditionBase(target)
	if err != nil || !kind.Valid() {
		return ConditionRemoved{}, invalidInput("condition removed needs a target and a valid kind")
	}
	return ConditionRemoved{b, kind}, nil
}

// Kind returns "condition_removed".
func (ConditionRemoved) Kind() string { return "condition_removed" }

// ConditionKind returns the kind of condition that ended.
func (e ConditionRemoved) ConditionKind() content.ConditionKind { return e.kind }

// ConditionLevelReduced: the creature's Exhaustion level went down to a level
// still above 0. Its Kind is condition_level_reduced. Comparable with ==.
type ConditionLevelReduced struct {
	conditionEventBase
	level ExhaustionLevel
}

// NewConditionLevelReduced returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero target or the zero level.
func NewConditionLevelReduced(target CombatantID, level ExhaustionLevel) (ConditionLevelReduced, error) {
	b, err := newConditionBase(target)
	if err != nil || level.IsZero() {
		return ConditionLevelReduced{}, invalidInput("condition level reduced needs a target and a level")
	}
	return ConditionLevelReduced{b, level}, nil
}

// Kind returns "condition_level_reduced".
func (ConditionLevelReduced) Kind() string { return "condition_level_reduced" }

// Level returns the new Exhaustion level.
func (e ConditionLevelReduced) Level() ExhaustionLevel { return e.level }
