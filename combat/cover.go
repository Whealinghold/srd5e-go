package combat

// Cover is the degree of cover between an attacker and a target (SRD 5.2.1,
// "Cover", p. 15): HalfCover (+2 to AC and Dexterity saving throws),
// ThreeQuartersCover (+5) and TotalCover (the target cannot be targeted
// directly). The zero value is no cover and is valid. Comparable and safe for
// concurrent use.
type Cover int

// The three degrees, from least to most protective.
const (
	HalfCover Cover = iota + 1
	ThreeQuartersCover
	TotalCover
)

// Valid reports whether c is no cover or one of the three degrees.
func (c Cover) Valid() bool { return c >= 0 && c <= TotalCover }

// ACBonus returns the bonus to Armor Class: 2, 5, or 0 for no cover and for
// Total Cover, which has no bonus because the target cannot be targeted.
func (c Cover) ACBonus() int { return c.bonus() }

// DexteritySaveBonus returns the bonus to Dexterity saving throws, as ACBonus.
func (c Cover) DexteritySaveBonus() int { return c.bonus() }

func (c Cover) bonus() int {
	switch c {
	case HalfCover:
		return 2
	case ThreeQuartersCover:
		return 5
	case TotalCover:
		return 0
	}
	return 0
}

// String returns the srd5e.combat.attack.cover value: "none", "half",
// "three_quarters", "total" or "unknown".
func (c Cover) String() string {
	if c == 0 {
		return "none"
	}
	return enumName([]string{"half", "three_quarters", "total"}, int(c))
}

// CoverSource is one thing that gives cover and whether the attack originates
// on the opposite side of it. Construct with a keyed literal.
type CoverSource struct {
	Cover                        Cover
	AttackOriginatesOppositeSide bool
}

// CoverResolver is the stateless domain service that picks the cover that
// applies. The zero value is ready to use. Safe for concurrent use.
type CoverResolver struct{}

// Applicable returns the most protective cover among the sources whose
// AttackOriginatesOppositeSide is true; degrees are never added (SRD 5.2.1,
// "Cover", p. 15). It returns no cover (0) for no applicable source, and ignores
// a source whose Cover is not valid.
func (CoverResolver) Applicable(sources []CoverSource) Cover {
	var best Cover
	for _, s := range sources {
		if s.AttackOriginatesOppositeSide && s.Cover.Valid() && s.Cover > best {
			best = s.Cover
		}
	}
	return best
}
