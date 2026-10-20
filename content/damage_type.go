package content

// DamageType is one of the 13 damage types, in table order (SRD 5.2.1,
// "Damage Types", p. 180). It carries identity only; how damage is handled is
// the combat context's. The zero DamageType is invalid. Comparable and safe
// for concurrent use.
type DamageType int

// The damage types.
const (
	DamageAcid DamageType = iota + 1
	DamageBludgeoning
	DamageCold
	DamageFire
	DamageForce
	DamageLightning
	DamageNecrotic
	DamagePiercing
	DamagePoison
	DamagePsychic
	DamageRadiant
	DamageSlashing
	DamageThunder
)

func damageTypeNames() []string {
	return []string{
		"Acid", "Bludgeoning", "Cold", "Fire", "Force", "Lightning", "Necrotic",
		"Piercing", "Poison", "Psychic", "Radiant", "Slashing", "Thunder",
	}
}

// Valid reports whether d is one of the 13 damage types.
func (d DamageType) Valid() bool { return enumValid(damageTypeNames(), int(d)) }

// String returns the printed name of d, or "unknown" for an undefined value.
func (d DamageType) String() string { return enumString(damageTypeNames(), int(d)) }

// ParseDamageType returns the damage type spelled exactly s, such as
// "Bludgeoning". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseDamageType(s string) (DamageType, error) {
	v, err := parseEnum("DamageType", damageTypeNames(), s)
	return DamageType(v), err
}
