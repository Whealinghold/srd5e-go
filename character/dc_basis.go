package character

// DCBasis records how an Influence check's Difficulty Class was chosen (SRD
// 5.2.1, "Influence [Action]", p. 184). The zero DCBasis is invalid.
// Comparable and safe for concurrent use.
type DCBasis int

// The bases of a Difficulty Class.
const (
	// DefaultRule: the default DC, 15 or the monster's Intelligence score,
	// whichever is higher.
	DefaultRule DCBasis = iota + 1
	// GMSet: a DC the Game Master set.
	GMSet
)

// Valid reports whether b is DefaultRule or GMSet.
func (b DCBasis) Valid() bool { return b == DefaultRule || b == GMSet }

// String returns "default_rule" or "gm_set", or "unknown" for an undefined value.
func (b DCBasis) String() string {
	switch b {
	case DefaultRule:
		return "default_rule"
	case GMSet:
		return "gm_set"
	}
	return "unknown"
}
