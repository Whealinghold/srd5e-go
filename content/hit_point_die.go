package content

// HitPointDie is the die a class rolls for Hit Points: D6, D8, D10 or D12
// (SRD 5.2.1, "Core Barbarian Traits", p. 28, and the Core Traits tables of
// the other classes). It is separate from the kernel dice.Die so that it
// cannot name any other die. Its static fact is [HitPointDie.Faces]; to roll
// it, convert with dice.NewDie(h.Faces()). The zero HitPointDie is invalid.
// Comparable and safe for concurrent use.
type HitPointDie int

// The Hit Point Dice, in ascending order.
const (
	HitDieD6 HitPointDie = iota + 1
	HitDieD8
	HitDieD10
	HitDieD12
)

func hitPointDieNames() []string { return []string{"D6", "D8", "D10", "D12"} }

// Valid reports whether h is one of the four Hit Point Dice.
func (h HitPointDie) Valid() bool { return enumValid(hitPointDieNames(), int(h)) }

// String returns the class-table spelling of h ("D8"), or "unknown" for an
// undefined value.
func (h HitPointDie) String() string { return enumString(hitPointDieNames(), int(h)) }

// Faces returns the number of faces of h: 6, 8, 10 or 12. It returns 0 when h
// is invalid.
func (h HitPointDie) Faces() int {
	switch h {
	case HitDieD6:
		return 6
	case HitDieD8:
		return 8
	case HitDieD10:
		return 10
	case HitDieD12:
		return 12
	}
	return 0 // equivalent mutant: replaces the literal 0 with 0
}

// ParseHitPointDie returns the die spelled exactly s: "D6", "D8", "D10" or
// "D12". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string, including "d8" and "D4". It never panics.
func ParseHitPointDie(s string) (HitPointDie, error) {
	v, err := parseEnum("HitPointDie", hitPointDieNames(), s)
	return HitPointDie(v), err
}
