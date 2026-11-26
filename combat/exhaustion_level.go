package combat

// ExhaustionLevel is a level of the Exhaustion condition, 1 to 6 (SRD 5.2.1,
// "Exhaustion", p. 181). "No Exhaustion" is the absence of the condition, so the
// zero ExhaustionLevel is invalid; see IsZero. Comparable and safe for
// concurrent use.
type ExhaustionLevel struct{ n int }

const (
	// minExhaustionLevel is the lowest level of the condition.
	minExhaustionLevel = 1
	// maxExhaustionLevel is the highest level: "You die if your Exhaustion level is 6" (p. 181).
	maxExhaustionLevel = 6
)

// NewExhaustionLevel returns level n. It returns an error matching
// srderr.ErrInvalidInput for n below 1 or above 6. It never panics.
func NewExhaustionLevel(n int) (ExhaustionLevel, error) {
	if n < minExhaustionLevel || n > maxExhaustionLevel {
		return ExhaustionLevel{}, invalidInput("exhaustion level must be 1 to 6")
	}
	return ExhaustionLevel{n}, nil
}

// MustExhaustionLevel is NewExhaustionLevel for literals known to be valid. It
// panics with the error NewExhaustionLevel returns, and only then.
func MustExhaustionLevel(n int) ExhaustionLevel {
	l, err := NewExhaustionLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 6, or 0 for the invalid zero value.
func (l ExhaustionLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero ExhaustionLevel.
func (l ExhaustionLevel) IsZero() bool { return l.n == 0 }
