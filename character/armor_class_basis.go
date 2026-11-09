package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// ArmorClassBasis is a way to compute the base of Armor Class: only one is in
// effect at a time (SRD 5.2.1, "Armor Class", p. 7, p. 177). It is a closed set.
// Variants: StandardArmorClassBasis, ArmoredArmorClassBasis,
// AbilityBasedArmorClassBasis.
type ArmorClassBasis interface{ isArmorClassBasis() }

// StandardArmorClassBasis is 10 plus the Dexterity modifier, used when no armor
// is worn.
type StandardArmorClassBasis struct{}

// ArmoredArmorClassBasis is the base of the worn body armor.
type ArmoredArmorClassBasis struct{ Armor content.Armor }

// AbilityBasedArmorClassBasis is a base value plus Dexterity plus the modifiers
// of a set of abilities, which applies only when no body armor is worn and, when
// ShieldAllowed is false, no Shield is wielded (SRD 5.2.1, "Unarmored Defense",
// pp. 29 and 50; "Mage Armor", p. 145). It is a command input; the class and
// spell specs build it. It holds a slice, so it is not comparable with ==. The
// zero value is invalid; build it with NewAbilityBasedArmorClassBasis.
type AbilityBasedArmorClassBasis struct {
	id            content.FeatureID
	abilities     []ability.Ability
	shieldAllowed bool
	baseValue     int
}

func (StandardArmorClassBasis) isArmorClassBasis()     {}
func (ArmoredArmorClassBasis) isArmorClassBasis()      {}
func (AbilityBasedArmorClassBasis) isArmorClassBasis() {}

// DefaultArmorClassBaseValue is the base value of Armor Class: 10.
const DefaultArmorClassBaseValue = 10

// NewAbilityBasedArmorClassBasis returns the basis identified by id with a base
// value of 10, adding the modifiers of abilities (repeats ignored, kept in SRD
// order). It returns an error matching srderr.ErrInvalidInput for an invalid id
// or ability. It never panics.
func NewAbilityBasedArmorClassBasis(id content.FeatureID, abilities []ability.Ability, shieldAllowed bool) (AbilityBasedArmorClassBasis, error) {
	if !id.Valid() {
		return AbilityBasedArmorClassBasis{}, invalidInput("invalid feature id")
	}
	if slices.ContainsFunc(abilities, func(a ability.Ability) bool { return !validAbility(a) }) {
		return AbilityBasedArmorClassBasis{}, invalidInput("invalid ability")
	}
	var ordered []ability.Ability
	for _, a := range ability.All() {
		if slices.Contains(abilities, a) {
			ordered = append(ordered, a)
		}
	}
	return AbilityBasedArmorClassBasis{id: id, abilities: ordered, shieldAllowed: shieldAllowed, baseValue: DefaultArmorClassBaseValue}, nil
}

// WithBaseValue returns a copy whose base value is n (Mage Armor: 13). It
// returns an error matching srderr.ErrInvalidInput for n below 1.
func (b AbilityBasedArmorClassBasis) WithBaseValue(n int) (AbilityBasedArmorClassBasis, error) {
	if n < 1 {
		return AbilityBasedArmorClassBasis{}, invalidInput("Armor Class base value below 1")
	}
	b.baseValue = n
	return b, nil
}

// ID returns the id of the feature that grants the basis.
func (b AbilityBasedArmorClassBasis) ID() content.FeatureID { return b.id }

// Abilities returns a new slice of the abilities added, in SRD order.
func (b AbilityBasedArmorClassBasis) Abilities() []ability.Ability { return slices.Clone(b.abilities) }

// ShieldAllowed reports whether the basis applies while a Shield is wielded.
func (b AbilityBasedArmorClassBasis) ShieldAllowed() bool { return b.shieldAllowed }

// BaseValue returns the base value, 10 unless changed.
func (b AbilityBasedArmorClassBasis) BaseValue() int { return b.baseValue }

// IsZero reports whether b is the invalid zero value.
func (b AbilityBasedArmorClassBasis) IsZero() bool { return b.id == "" }

// ArmorClassBasisChoice names the basis a character chose to use. It is a
// closed set. Variants: ChooseStandard, ChooseArmored, ChooseAlternative.
type ArmorClassBasisChoice interface{ isArmorClassBasisChoice() }

// ChooseStandard chooses the standard basis, 10 plus Dexterity.
type ChooseStandard struct{}

// ChooseArmored chooses the worn body armor's basis.
type ChooseArmored struct{}

// ChooseAlternative chooses the alternative basis granted by feature ID.
type ChooseAlternative struct{ ID content.FeatureID }

func (ChooseStandard) isArmorClassBasisChoice()    {}
func (ChooseArmored) isArmorClassBasisChoice()     {}
func (ChooseAlternative) isArmorClassBasisChoice() {}

// ArmorClassBasisRejection says why a chosen basis was refused. The zero value
// is not a rejection.
type ArmorClassBasisRejection int

// The reasons a chosen basis is refused.
const (
	// BasisNotApplicable: the matrix of SRD 5.2.1, "Armor Class", p. 7 does not allow it with the armor worn.
	BasisNotApplicable ArmorClassBasisRejection = iota + 1
	// BasisAlternativeNotSupplied: the command supplies no alternative with the chosen id.
	BasisAlternativeNotSupplied
)

// String returns "notApplicable", "alternativeNotSupplied" or "unknown".
func (r ArmorClassBasisRejection) String() string {
	switch r {
	case BasisNotApplicable:
		return "notApplicable"
	case BasisAlternativeNotSupplied:
		return "alternativeNotSupplied"
	}
	return "unknown"
}

// ArmorSlot is where an armor id is worn: as body armor or as a Shield. The zero
// value is invalid.
type ArmorSlot int

// The armor slots.
const (
	// BodyArmorSlot: Light, Medium or Heavy armor, which sets a base Armor Class.
	BodyArmorSlot ArmorSlot = iota + 1
	// ShieldSlot: a Shield, which adds a bonus.
	ShieldSlot
)

// String returns "bodyArmor", "shield" or "unknown".
func (s ArmorSlot) String() string {
	switch s {
	case BodyArmorSlot:
		return "bodyArmor"
	case ShieldSlot:
		return "shield"
	}
	return "unknown"
}

// armorFitsSlot reports whether the armor's rule suits the slot: a base rule for
// body armor, a bonus rule for a Shield (content-equipment FR-024).
func armorFitsSlot(a content.Armor, slot ArmorSlot) bool {
	switch a.ArmorClass().(type) {
	case content.BaseRule:
		return slot == BodyArmorSlot
	case content.BonusRule:
		return slot == ShieldSlot
	}
	return false
}

// MustAbilityBasedArmorClassBasis is NewAbilityBasedArmorClassBasis for arguments known to be valid. It panics with the error
// NewAbilityBasedArmorClassBasis returns, and only then.
func MustAbilityBasedArmorClassBasis(id content.FeatureID, abilities []ability.Ability, shieldAllowed bool) AbilityBasedArmorClassBasis {
	b, err := NewAbilityBasedArmorClassBasis(id, abilities, shieldAllowed)
	if err != nil {
		panic(err)
	}
	return b
}
