package character

import (
	"math"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// ExperiencePoints is a character's Experience Points total, 0 or more (SRD
// 5.2.1, "Create Your Character", p. 19). It owns the Character Advancement
// table's XP column (p. 23). It is an immutable value, comparable with ==; the
// zero value is a valid total of 0. Safe for concurrent use.
type ExperiencePoints struct{ n int }

// ExperiencePointsReason says why NewExperiencePoints rejected a value. The
// zero value is not a reason.
type ExperiencePointsReason int

// ExperiencePointsNegative: the value is below 0.
const ExperiencePointsNegative ExperiencePointsReason = 1

// String returns "negative" or "unknown".
func (r ExperiencePointsReason) String() string {
	if r == ExperiencePointsNegative {
		return "negative"
	}
	return "unknown"
}

// ExperiencePointsError reports a rejected total. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type ExperiencePointsError struct{ Reason ExperiencePointsReason }

// Error renders the error. The text is not a contract.
func (e *ExperiencePointsError) Error() string {
	return "character: invalid Experience Points: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ExperiencePointsError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *ExperiencePointsError with the same Reason.
func (e *ExperiencePointsError) Is(target error) bool {
	t, ok := target.(*ExperiencePointsError)
	return ok && t != nil && e != nil && *t == *e
}

// NewExperiencePoints returns the total n. It returns a *ExperiencePointsError
// (matching srderr.ErrInvalidInput) for a negative n and accepts 0. It never
// panics.
func NewExperiencePoints(n int) (ExperiencePoints, error) {
	if n < 0 {
		return ExperiencePoints{}, &ExperiencePointsError{Reason: ExperiencePointsNegative}
	}
	return ExperiencePoints{n: n}, nil
}

// MustExperiencePoints is NewExperiencePoints for constants known to be valid.
// It panics with the error NewExperiencePoints returns, and only then.
func MustExperiencePoints(n int) ExperiencePoints {
	xp, err := NewExperiencePoints(n)
	if err != nil {
		panic(err)
	}
	return xp
}

// Value returns the total as a plain integer.
func (xp ExperiencePoints) Value() int { return xp.n }

// MinimumForLevel returns the Experience Points needed for level: the XP column
// of the Character Advancement table (SRD 5.2.1, p. 23). The receiver is not
// read. The zero Level gives the zero total.
func (ExperiencePoints) MinimumForLevel(level Level) ExperiencePoints {
	table := [MaxLevel]int{
		0, 300, 900, 2700, 6500, 14000, 23000, 34000, 48000, 64000,
		85000, 100000, 120000, 140000, 165000, 195000, 225000, 265000, 305000, 355000,
	}
	if level.IsZero() {
		return ExperiencePoints{}
	}
	return ExperiencePoints{n: table[level.n-1]}
}

// LevelFor returns the highest level whose minimum xp equals or exceeds (SRD
// 5.2.1, "Level Advancement", p. 23), at most 20.
func (xp ExperiencePoints) LevelFor() Level {
	level := Level{n: MinLevel}
	for n := MinLevel + 1; n <= MaxLevel; n++ {
		if xp.n >= xp.MinimumForLevel(Level{n: n}).n {
			level = Level{n: n}
		}
	}
	return level
}

// plus returns xp raised by amount, saturating at the largest int instead of
// wrapping.
func (xp ExperiencePoints) plus(amount ExperienceAmount) ExperiencePoints {
	if amount.n > math.MaxInt-xp.n {
		return ExperiencePoints{n: math.MaxInt}
	}
	return ExperiencePoints{n: xp.n + amount.n}
}
