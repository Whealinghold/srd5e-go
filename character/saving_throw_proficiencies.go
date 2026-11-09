package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// SavingThrowProficiencies is a set of abilities whose saving throws a
// character is proficient in (SRD 5.2.1, "Saving Throw Proficiencies", p. 9).
// It is an immutable value, comparable with ==. The zero value is the empty
// set. Safe for concurrent use.
type SavingThrowProficiencies struct{ set uint8 }

// NewSavingThrowProficiencies returns the set of abilities; order and repeats
// are irrelevant. It returns an error matching srderr.ErrInvalidInput for an
// invalid ability. It never panics.
func NewSavingThrowProficiencies(abilities ...ability.Ability) (SavingThrowProficiencies, error) {
	var s uint8
	for _, a := range abilities {
		if !validAbility(a) {
			return SavingThrowProficiencies{}, invalidInput("invalid ability")
		}
		s |= 1 << a
	}
	return SavingThrowProficiencies{set: s}, nil
}

// Has reports whether the set holds a.
func (p SavingThrowProficiencies) Has(a ability.Ability) bool {
	return validAbility(a) && p.set&(1<<a) != 0
}

// Union returns the set of the abilities in p or o.
func (p SavingThrowProficiencies) Union(o SavingThrowProficiencies) SavingThrowProficiencies {
	return SavingThrowProficiencies{set: p.set | o.set}
}

// Abilities returns a new slice of the abilities held, in SRD order.
func (p SavingThrowProficiencies) Abilities() []ability.Ability {
	var out []ability.Ability
	for _, a := range ability.All() {
		if p.Has(a) {
			out = append(out, a)
		}
	}
	return out
}

// classSavingThrows returns the saving throw proficiencies of a class
// definition (SRD 5.2.1, "Saving Throw Proficiencies", p. 9). Only the initial
// class of a character grants them (p. 25).
func classSavingThrows(def content.ClassDefinition) SavingThrowProficiencies {
	p, _ := NewSavingThrowProficiencies(def.SavingThrowProficiencies()...) // a decoded class holds valid abilities only
	return p
}
