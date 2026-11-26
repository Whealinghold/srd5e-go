package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
)

const (
	allDamageBit     = 1 << 13
	damageTypesCount = 13
)

// DamageAffinities are the Damage Types a creature resists, is vulnerable to
// and is immune to (SRD 5.2.1, "Resistance and Vulnerability" and "Immunity",
// p. 17). Duplicates collapse, so several sources of one Resistance count as
// one. It is comparable and immutable, and the zero value is valid and equals
// NoAffinities().
type DamageAffinities struct{ resist, vuln, immune uint16 }

// NoAffinities returns affinities with no Resistance, Vulnerability or Immunity.
func NoAffinities() DamageAffinities { return DamageAffinities{} }

// ResistantToAllDamage returns Resistance to all damage (a magic item's
// Resistance to all damage, SRD 5.2.1, "Magic Item Resilience", p. 206).
func ResistantToAllDamage() DamageAffinities { return DamageAffinities{resist: allDamageBit} }

// ImmuneToAllDamage returns Immunity to every Damage Type.
func ImmuneToAllDamage() DamageAffinities { return DamageAffinities{immune: 1<<damageTypesCount - 1} }

func typeBit(t content.DamageType) uint16 { return 1 << (uint(t) - 1) }

// NewDamageAffinities returns the affinities of the given resistances,
// vulnerabilities and immunities. Empty or nil slices are fine and duplicates
// collapse.
//
// It returns an error matching srderr.ErrInvalidInput for a Damage Type that is
// not valid or a DamageScope that is not AllDamage or a valid DamageOfType. It
// never panics.
func NewDamageAffinities(resistances []DamageScope, vulnerabilities, immunities []content.DamageType) (DamageAffinities, error) {
	var a DamageAffinities
	for _, s := range resistances {
		switch s := s.(type) {
		case AllDamage:
			a.resist |= allDamageBit
		case DamageOfType:
			if !s.Type.Valid() {
				return DamageAffinities{}, invalidInput("resistance to an invalid Damage Type")
			}
			a.resist |= typeBit(s.Type)
		default:
			return DamageAffinities{}, invalidInput("resistance with no scope")
		}
	}
	for _, t := range vulnerabilities {
		if !t.Valid() {
			return DamageAffinities{}, invalidInput("vulnerability to an invalid Damage Type")
		}
		a.vuln |= typeBit(t)
	}
	for _, t := range immunities {
		if !t.Valid() {
			return DamageAffinities{}, invalidInput("immunity to an invalid Damage Type")
		}
		a.immune |= typeBit(t)
	}
	return a, nil
}

func (a DamageAffinities) resists(t content.DamageType) bool {
	return a.resist&(allDamageBit|typeBit(t)) != 0
}
func (a DamageAffinities) vulnerable(t content.DamageType) bool { return a.vuln&typeBit(t) != 0 }
func (a DamageAffinities) immuneTo(t content.DamageType) bool   { return a.immune&typeBit(t) != 0 }

// types lists the Damage Types set in mask, in table order.
func types(mask uint16) []content.DamageType {
	var out []content.DamageType
	for t := content.DamageAcid; t <= content.DamageThunder; t++ {
		if mask&typeBit(t) != 0 {
			out = append(out, t)
		}
	}
	return out
}

// Resistances returns a new slice of the resistances: AllDamage first when
// present, then one DamageOfType per Damage Type in table order.
func (a DamageAffinities) Resistances() []DamageScope {
	var out []DamageScope
	if a.resist&allDamageBit != 0 {
		out = append(out, AllDamage{})
	}
	for _, t := range types(a.resist) {
		out = append(out, DamageOfType{Type: t})
	}
	return out
}

// Vulnerabilities returns a new slice of the Damage Types in table order.
func (a DamageAffinities) Vulnerabilities() []content.DamageType { return types(a.vuln) }

// Immunities returns a new slice of the Damage Types in table order.
func (a DamageAffinities) Immunities() []content.DamageType { return types(a.immune) }

// Equal reports whether a and o hold the same affinities, whatever the order
// they were given in.
func (a DamageAffinities) Equal(o DamageAffinities) bool { return a == o }
