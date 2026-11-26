package combat

import (
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// CasterStrain is the toll a Resurrection cast takes on its caster until a Long
// Rest is finished (SRD 5.2.1, "Resurrection", p. 159). The zero value is
// StrainNone. Comparable and safe for concurrent use.
type CasterStrain int

// The strains of a caster.
const (
	// StrainNone: the caster is not taxed.
	StrainNone CasterStrain = iota
	// Strained: the caster cannot cast spells and has Disadvantage on D20 Tests.
	Strained
)

// String returns "none", "strained" or "unknown".
func (s CasterStrain) String() string {
	if s < StrainNone {
		return "unknown"
	}
	return enumName([]string{"none", "strained"}, int(s)+1)
}

// CanCastSpells reports whether the caster may cast spells, cantrips included:
// false while Strained.
func (s CasterStrain) CanCastSpells() bool { return s == StrainNone }

// RollCircumstances returns a fresh slice of the circumstances every D20 Test of
// the caster has: one modifier.Disadvantage while Strained, none otherwise. The
// host merges them with other sources through modifier.Resolve.
func (s CasterStrain) RollCircumstances() []modifier.RollCircumstance {
	if s == StrainNone {
		return []modifier.RollCircumstance{}
	}
	return []modifier.RollCircumstance{modifier.Disadvantage{}}
}

// AfterRest returns StrainNone for character.LongRest and an equal value for every
// other kind. The host passes RestApplied.BenefitsOf().
func (s CasterStrain) AfterRest(kind character.RestKind) CasterStrain {
	if kind == character.LongRest {
		return StrainNone
	}
	return s
}
