package character

// FreeCastingOutcome is whether a Boon of Spell Recall spared a spell slot, a
// closed set of two (SRD 5.2.1, "Boon of Spell Recall", Free Casting, p. 88). The
// zero value is invalid. Comparable and safe for concurrent use.
type FreeCastingOutcome int

// The outcomes.
const (
	// FreeCastingSpared: the slot is not expended.
	FreeCastingSpared FreeCastingOutcome = iota + 1
	// FreeCastingSpent: the slot is expended as usual.
	FreeCastingSpent
)

// Valid reports whether o is one of the two outcomes.
func (o FreeCastingOutcome) Valid() bool { return o == FreeCastingSpared || o == FreeCastingSpent }

// String returns "spared", "spent" or "unknown".
func (o FreeCastingOutcome) String() string { return featNameOf(int(o), "spared", "spent") }

// FreeCastingRule decides the Free Casting of Boon of Spell Recall. The zero value
// is ready to use. It is stateless and safe for concurrent use.
type FreeCastingRule struct{}

// Applies reports whether the benefit applies to a slot of level l: it does for
// levels 1 to 4 and not for 5 to 9 or the zero SlotLevel.
func (FreeCastingRule) Applies(l SlotLevel) bool { return l.Value() >= 1 && l.Value() <= 4 }

// Resolve decides, from one roll of 1d4, whether a slot of level l is spared: it
// is exactly when the roll equals the slot's level, otherwise it is spent.
//
// It returns an error matching srderr.ErrInvalidInput for a level the benefit does
// not apply to or a roll outside 1 to 4. It never panics.
func (r FreeCastingRule) Resolve(l SlotLevel, roll int) (FreeCastingOutcome, error) {
	if !r.Applies(l) || roll < 1 || roll > 4 {
		return 0, invalidInput("free casting of an inapplicable slot level or a roll outside 1d4")
	}
	return freeCastingOutcome(l, roll), nil
}

// freeCastingOutcome is Resolve for arguments already checked.
func freeCastingOutcome(l SlotLevel, roll int) FreeCastingOutcome {
	if roll == l.Value() {
		return FreeCastingSpared
	}
	return FreeCastingSpent
}

// GreatWeaponFighting is the damage die rule of the Great Weapon Fighting feat
// (SRD 5.2.1, p. 88). The zero value is ready to use. It is stateless and safe for
// concurrent use.
type GreatWeaponFighting struct{}

// Treat returns 3 for a rolled damage die of 1 or 2 and the roll for every other
// value. rolled is one face of one die. It returns an error matching
// srderr.ErrInvalidInput for a roll below 1. It never panics.
func (GreatWeaponFighting) Treat(rolled int) (int, error) {
	if rolled < 1 {
		return 0, invalidInput("a damage die rolled below 1")
	}
	return max(rolled, 3), nil
}
