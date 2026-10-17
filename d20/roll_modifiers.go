package d20

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/modifier"
)

// RollModifiers are the modifiers added to a D20 Test ("Add Modifiers", SRD
// 5.2.1, p. 6): exactly one ability modifier, at most one Proficiency Bonus
// and any number of others. The shape enforces the "at most one" rules, so the
// Proficiency Bonus cannot be added twice ("The Bonus Doesn't Stack", p. 8).
// Modifiers combine by plain signed addition.
//
// A RollModifiers is immutable and not comparable with ==; use [RollModifiers.Equal].
// The zero value is an ability modifier of +0 and nothing else.
type RollModifiers struct {
	ability modifier.Modifier
	prof    modifier.Modifier
	hasProf bool
	others  []modifier.Modifier
}

// NewRollModifiers returns modifiers holding only the ability modifier.
func NewRollModifiers(ability modifier.Modifier) RollModifiers {
	return RollModifiers{ability: ability}
}

// WithProficiencyBonus returns a copy with the Proficiency Bonus set to b,
// replacing any earlier one.
func (r RollModifiers) WithProficiencyBonus(b modifier.Modifier) RollModifiers {
	r.prof, r.hasProf = b, true
	return r
}

// WithOthers returns a copy with ms appended to the other modifiers. The
// receiver and ms are not retained.
func (r RollModifiers) WithOthers(ms ...modifier.Modifier) RollModifiers {
	r.others = slices.Concat(r.others, ms)
	return r
}

// Ability returns the ability modifier.
func (r RollModifiers) Ability() modifier.Modifier { return r.ability }

// ProficiencyBonus returns the Proficiency Bonus and whether one is set.
func (r RollModifiers) ProficiencyBonus() (modifier.Modifier, bool) { return r.prof, r.hasProf }

// Others returns a copy of the other modifiers; empty when there are none.
func (r RollModifiers) Others() []modifier.Modifier { return slices.Clone(r.others) }

// Count returns how many modifiers are held: the ability modifier, the
// Proficiency Bonus when set, and each other. At least 1.
func (r RollModifiers) Count() int {
	n := 1 + len(r.others)
	if r.hasProf {
		n++
	}
	return n
}

// Total returns the sum of every modifier held, with no clamping.
func (r RollModifiers) Total() modifier.Modifier {
	t := r.ability.Add(r.prof)
	return t.Add(modifier.Sum(r.others...))
}

// Equal reports whether r and o hold equal modifiers in the same slots, with
// the other modifiers in the same order.
func (r RollModifiers) Equal(o RollModifiers) bool {
	return r.ability == o.ability && r.prof == o.prof && r.hasProf == o.hasProf &&
		slices.Equal(r.others, o.others)
}
