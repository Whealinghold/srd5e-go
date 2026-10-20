package content

import "github.com/Whealinghold/srd5e-go/srderr"

// The limits of a [SpellLevel] (content-spells.md FR-009, FR-111).
const (
	// MinSpellLevel is the level of a cantrip.
	MinSpellLevel = 0
	// MaxSpellLevel is the highest spell level.
	MaxSpellLevel = 9
)

// ValidSpellLevel reports whether n is a spell level, 0 to 9 (SRD 5.2.1,
// "Spell Level", p. 104). The decoder and [NewSpellLevel] both use it.
func ValidSpellLevel(n int) bool { return n >= MinSpellLevel && n <= MaxSpellLevel }

// SpellLevel is the level of a spell, 0 to 9; level 0 is a cantrip (SRD 5.2.1,
// "Spell Level", p. 104). The zero SpellLevel is the cantrip level and is
// valid. A conversion such as SpellLevel(10) can bypass validation, so only
// values from [NewSpellLevel] or [MustSpellLevel] are known valid. Comparable
// and safe for concurrent use.
type SpellLevel int

// SpellLevelError reports an integer that is not a spell level. Value is the
// rejected integer. It matches srderr.ErrInvalidInput with errors.Is, and
// another *SpellLevelError with the same Value. Do not branch on the rendered
// text.
type SpellLevelError struct {
	Value int
}

// Error renders the error. The text is not a contract.
func (e *SpellLevelError) Error() string { return "content: invalid spell level" }

// Unwrap returns srderr.ErrInvalidInput.
func (e *SpellLevelError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *SpellLevelError (not a nil one) with the same
// Value.
func (e *SpellLevelError) Is(target error) bool {
	t, ok := target.(*SpellLevelError)
	return ok && t != nil && e != nil && t.Value == e.Value
}

// NewSpellLevel returns the level n, which must be 0 to 9.
//
// It returns a *[SpellLevelError] wrapping srderr.ErrInvalidInput for any
// other integer. It never panics.
func NewSpellLevel(n int) (SpellLevel, error) {
	if !ValidSpellLevel(n) {
		return 0, &SpellLevelError{Value: n}
	}
	return SpellLevel(n), nil
}

// MustSpellLevel is like [NewSpellLevel] but panics with its error when n is
// not a spell level.
func MustSpellLevel(n int) SpellLevel {
	l, err := NewSpellLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Int returns the level as an integer.
func (l SpellLevel) Int() int { return int(l) }

// Valid reports whether l is a spell level.
func (l SpellLevel) Valid() bool { return ValidSpellLevel(int(l)) }

// IsCantrip reports whether l is level 0, a cantrip.
func (l SpellLevel) IsCantrip() bool { return l == MinSpellLevel }
