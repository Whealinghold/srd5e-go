package combat

import "github.com/Whealinghold/srd5e-go/content"

// Apply applies c to the creature target, deciding in this order (FR-125): the
// creature is immune to the kind, stored or derived (ConditionImmune); the kind
// already holds, directly or by implication (ConditionAlreadyHeld, the original
// source kept); Exhaustion is at level 6 (ExhaustionAtMaximum); otherwise
// ConditionAdded with one ConditionApplied. Applying Exhaustion ignores the level
// c holds and adds exactly one (SRD 5.2.1, "Exhaustion", p. 181). The receiver is
// not changed.
//
// A zero target or an invalid condition changes nothing and gives
// ConditionAlreadyHeld; the use case's command rejects them with an error first.
func (s ConditionSet) Apply(target CombatantID, c Condition) ApplyConditionToSetResult {
	if target.IsZero() || !validCondition(c) {
		return ConditionAlreadyHeld{conditionUnchanged{s}}
	}
	k := c.Kind()
	if (s.immune|s.derivedImmunities())&bit(k) != 0 {
		return ConditionImmune{conditionUnchanged{s}}
	}
	if k == content.ConditionExhaustion {
		n := s.exhaustionLevel()
		if n >= maxExhaustionLevel {
			return ExhaustionAtMaximum{conditionUnchanged{s}}
		}
		c = ExhaustionCondition{ExhaustionLevel{n + 1}}
	} else if s.effectiveMask()&bit(k) != 0 {
		return ConditionAlreadyHeld{conditionUnchanged{s}}
	}
	return ConditionAdded{conditionChange{s.with(c), []ConditionEvent{ConditionApplied{conditionEventBase{target}, c}}}}
}

// Remove removes the condition of kind from the creature target, deciding in
// this order (FR-125): an explicit condition, even one an immunity suppresses, is
// removed (ConditionCleared with one ConditionRemoved, or ExhaustionLowered with
// one ConditionLevelReduced for Exhaustion above level 1); a kind that holds only
// by implication gives ConditionRemovalImpliedBy; anything else
// ConditionRemovalNotPresent. Removing Unconscious leaves the creature Prone: the
// returned set holds Prone and no event reports it, so read the set (p. 191). The
// receiver is not changed.
//
// A zero target or an invalid kind changes nothing and gives
// ConditionRemovalNotPresent.
func (s ConditionSet) Remove(target CombatantID, kind content.ConditionKind) RemoveConditionFromSetResult {
	if target.IsZero() || !kind.Valid() {
		return ConditionRemovalNotPresent{conditionUnchanged{s}}
	}
	if s.slots[kind] != nil {
		if n := s.exhaustionLevel(); kind == content.ConditionExhaustion && n > 1 {
			lower := ExhaustionLevel{n - 1}
			return ExhaustionLowered{conditionChange{s.with(ExhaustionCondition{lower}),
				[]ConditionEvent{ConditionLevelReduced{conditionEventBase{target}, lower}}}}
		}
		next := s.without(kind)
		if kind == content.ConditionUnconscious && s.effectiveMask()&bit(kind) != 0 &&
			s.immune&bit(content.ConditionProne) == 0 && next.slots[content.ConditionProne] == nil {
			next = next.with(Prone{})
		}
		return ConditionCleared{conditionChange{next, []ConditionEvent{ConditionRemoved{conditionEventBase{target}, kind}}}}
	}
	if s.effectiveMask()&bit(kind) != 0 {
		return ConditionRemovalImpliedBy{conditionUnchanged{s}, s.implying(kind)}
	}
	return ConditionRemovalNotPresent{conditionUnchanged{s}}
}

// implying lists the remaining explicit kinds whose definition implies kind.
func (s ConditionSet) implying(kind content.ConditionKind) []content.ConditionKind {
	var out []content.ConditionKind
	for _, k := range kindsOf(s.remaining()) {
		for _, implied := range definitionOf(k).implies {
			if implied == kind {
				out = append(out, k)
			}
		}
	}
	return out
}

// ReduceExhaustion takes levels Exhaustion levels off the creature target
// (SRD 5.2.1, "Exhaustion", p. 181). At least one level stays: ExhaustionReduced
// with one ConditionLevelReduced. Reaching 0 or going below ends the condition:
// ExhaustionEnded with one ConditionRemoved. It reduces the stored level even when
// an immunity suppresses it. Without Exhaustion the result is
// ExhaustionReductionNotPresent. The receiver is not changed.
//
// It returns an error matching srderr.ErrInvalidInput, checked first, for levels
// below 1, then for a zero target, and the nil result with it. It never panics.
func (s ConditionSet) ReduceExhaustion(target CombatantID, levels int) (ReduceExhaustionOnSetResult, error) {
	if levels < 1 {
		return nil, invalidInput("levels must be at least 1")
	}
	if target.IsZero() {
		return nil, invalidInput("zero target")
	}
	n := s.exhaustionLevel()
	switch {
	case n == 0:
		return ExhaustionReductionNotPresent{conditionUnchanged{s}}, nil
	case levels < n:
		lower := ExhaustionLevel{n - levels}
		return ExhaustionReduced{conditionChange{s.with(ExhaustionCondition{lower}),
			[]ConditionEvent{ConditionLevelReduced{conditionEventBase{target}, lower}}}}, nil
	}
	return ExhaustionEnded{conditionChange{s.without(content.ConditionExhaustion),
		[]ConditionEvent{ConditionRemoved{conditionEventBase{target}, content.ConditionExhaustion}}}}, nil
}
