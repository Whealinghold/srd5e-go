package combat

// Burrowing (SRD 5.2.1, "Burrow Speed", p. 178). Pure and silent. It reads no speed: a
// burrow move by a creature with no Burrow Speed is refused by the movement segment rules.

// BurrowMedium is what a creature burrows through. The zero value is invalid.
type BurrowMedium int

// The five media, in the order of the SRD.
const (
	BurrowMediumSand BurrowMedium = iota + 1
	BurrowMediumEarth
	BurrowMediumMud
	BurrowMediumIce
	BurrowMediumSolidRock
)

// Valid reports whether m is one of the five media.
func (m BurrowMedium) Valid() bool { return m >= BurrowMediumSand && m <= BurrowMediumSolidRock }

// String returns "sand", "earth", "mud", "ice", "solid_rock" or "unknown".
func (m BurrowMedium) String() string {
	return enumName([]string{"sand", "earth", "mud", "ice", "solid_rock"}, int(m))
}

// SolidRockAccess says whether a trait lets the creature burrow through solid rock. The
// zero value is SolidRockAccessNone. The host decides that a trait allows it.
type SolidRockAccess int

// The two values.
const (
	SolidRockAccessNone SolidRockAccess = iota
	SolidRockAccessAllowedByTrait
)

// BurrowRefusal is why a burrow is refused. The zero value is invalid.
type BurrowRefusal int

// BurrowRefusalSolidRock is the one refusal: solid rock without a trait that allows it.
const BurrowRefusalSolidRock BurrowRefusal = iota + 1

// BurrowPermission is the closed set of verdicts of BurrowPermissionFor. Variants:
// BurrowAllowed, BurrowRefused.
//
//sumtype:decl
type BurrowPermission interface{ isBurrowPermission() }

// BurrowAllowed is a burrow the creature may make.
type BurrowAllowed struct{}

// BurrowRefused is a burrow the creature may not make. Build it with a keyed literal.
type BurrowRefused struct{ Reason BurrowRefusal }

func (BurrowAllowed) isBurrowPermission() {}
func (BurrowRefused) isBurrowPermission() {}

// BurrowPermissionFor returns whether a creature with a Burrow Speed may burrow through
// medium: sand, earth, mud and ice always, solid rock only with a trait that allows it. An
// invalid medium is refused with the zero BurrowRefusal. It cannot fail.
func BurrowPermissionFor(medium BurrowMedium, access SolidRockAccess) BurrowPermission {
	switch {
	case !medium.Valid():
		return BurrowRefused{}
	case medium == BurrowMediumSolidRock && access != SolidRockAccessAllowedByTrait:
		return BurrowRefused{BurrowRefusalSolidRock}
	}
	return BurrowAllowed{}
}
