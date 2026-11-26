package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// ConditionSet is the conditions on one creature: the explicit Conditions (at
// most one per kind) and the kinds the creature is immune to. The conditions that
// actually hold, the effective ones, are derived and never stored: an immune
// condition is suppressed, not deleted (SRD 5.2.1, "Immunity", pp. 17, 183), and
// Paralyzed, Petrified, Stunned and Unconscious imply Incapacitated, Unconscious
// also Prone. The set holds no id. The zero ConditionSet is the empty set and is
// valid. It is immutable and comparable with ==; Equal is the same test. Safe for
// concurrent use.
type ConditionSet struct {
	slots  [16]Condition // indexed by content.ConditionKind; nil when not held
	immune uint16        // bit k set: immune to kind k
}

// kindMask is a set of condition kinds, one bit per kind.
type kindMask = uint16

func bit(k content.ConditionKind) kindMask { return 1 << uint(k) }

// NewConditionSet returns the set holding explicit with the given immunities, in
// any order. It returns an error matching srderr.ErrInvalidInput for a nil or
// invalid Condition (an unset source or level, a blank item charmer), for two
// Conditions of one kind (a Condition does not stack with itself, p. 179) and
// for an invalid immunity kind. It copies its arguments and never panics.
func NewConditionSet(explicit []Condition, immunities []content.ConditionKind) (ConditionSet, error) {
	var s ConditionSet
	for _, c := range explicit {
		if !validCondition(c) {
			return ConditionSet{}, invalidInput("invalid condition")
		}
		if s.slots[c.Kind()] != nil {
			return ConditionSet{}, invalidInput("two conditions of one kind")
		}
		s.slots[c.Kind()] = c
	}
	for _, k := range immunities {
		if !k.Valid() {
			return ConditionSet{}, invalidInput("invalid immunity kind")
		}
		s.immune |= bit(k)
	}
	return s, nil
}

// MustConditionSet is NewConditionSet for literals known to be valid. It panics
// with the error NewConditionSet returns, and only then.
func MustConditionSet(explicit []Condition, immunities []content.ConditionKind) ConditionSet {
	s, err := NewConditionSet(explicit, immunities)
	if err != nil {
		panic(err)
	}
	return s
}

// kindsOf lists the kinds in m in the order of content.ConditionKind.
func kindsOf(m kindMask) []content.ConditionKind {
	var out []content.ConditionKind
	for k := content.ConditionBlinded; k <= content.ConditionUnconscious; k++ {
		if m&bit(k) != 0 {
			out = append(out, k)
		}
	}
	return out
}

// explicitMask is the kinds held as explicit conditions.
func (s ConditionSet) explicitMask() kindMask {
	var m kindMask
	for k := content.ConditionBlinded; k <= content.ConditionUnconscious; k++ {
		if s.slots[k] != nil {
			m |= bit(k)
		}
	}
	return m
}

// remaining is step 2 of the pipeline (FR-124): the explicit kinds that stored
// immunities do not suppress.
func (s ConditionSet) remaining() kindMask { return s.explicitMask() &^ s.immune }

// derivedImmunities are the ImmuneTo effects of the remaining conditions.
func (s ConditionSet) derivedImmunities() kindMask {
	var m kindMask
	for _, k := range kindsOf(s.remaining()) {
		for _, e := range definitionOf(k).effects {
			if i, ok := e.(ImmuneTo); ok {
				m |= bit(i.Kind)
			}
		}
	}
	return m
}

// effectiveMask runs the pipeline of FR-124: explicit conditions, minus stored
// immunities, plus implied kinds, minus stored and derived immunities.
func (s ConditionSet) effectiveMask() kindMask {
	m := s.remaining()
	for _, k := range kindsOf(m) {
		for _, implied := range definitionOf(k).implies {
			m |= bit(implied)
		}
	}
	return m &^ (s.immune | s.derivedImmunities())
}

// Effective returns the kinds that hold on the creature, each once, in the
// order of content.ConditionKind. It is a new slice.
func (s ConditionSet) Effective() []content.ConditionKind { return kindsOf(s.effectiveMask()) }

// Explicit returns a copy of the explicit conditions, at most one per kind, in
// the order of content.ConditionKind. A caller reads a source from it, such as
// the Grappled variant's grappler, because the removal event carries no source.
func (s ConditionSet) Explicit() []Condition {
	var out []Condition
	for _, k := range kindsOf(s.explicitMask()) {
		out = append(out, s.slots[k])
	}
	return out
}

// Immunities returns the stored immunities in the order of content.ConditionKind.
// It is a new slice.
func (s ConditionSet) Immunities() []content.ConditionKind { return kindsOf(s.immune) }

// Equal reports whether s and o hold the same explicit conditions and
// immunities, whatever order they were supplied in.
func (s ConditionSet) Equal(o ConditionSet) bool { return s == o }

// with returns s holding c in place of any condition of its kind.
func (s ConditionSet) with(c Condition) ConditionSet {
	s.slots[c.Kind()] = c
	return s
}

// without returns s without the explicit condition of kind k.
func (s ConditionSet) without(k content.ConditionKind) ConditionSet {
	s.slots[k] = nil
	return s
}

// exhaustionLevel is the stored Exhaustion level, 0 when there is none.
func (s ConditionSet) exhaustionLevel() int {
	if e, ok := s.slots[content.ConditionExhaustion].(ExhaustionCondition); ok {
		return e.Level.n
	}
	return 0
}

// effectiveExhaustionLevel is the Exhaustion level that counts: 0 when the
// condition is absent or suppressed.
func (s ConditionSet) effectiveExhaustionLevel() int {
	if s.effectiveMask()&bit(content.ConditionExhaustion) == 0 {
		return 0
	}
	return s.exhaustionLevel()
}

// each calls f for every effect of every effective condition, in the order of
// content.ConditionKind.
func (s ConditionSet) each(f func(k content.ConditionKind, e ConditionEffect)) {
	for _, k := range kindsOf(s.effectiveMask()) {
		for _, e := range definitionOf(k).effects {
			f(k, e)
		}
	}
}
