package character

import "strings"

// MagicItemInstanceID names one particular magic item, a copy of a content
// MagicItemID. The host supplies it. Build one with NewMagicItemInstanceID; the
// zero value is invalid. Comparable and safe for concurrent use.
type MagicItemInstanceID string

// NewMagicItemInstanceID returns s as an id. It returns an error matching
// srderr.ErrInvalidInput for a blank s. It never panics.
func NewMagicItemInstanceID(s string) (MagicItemInstanceID, error) {
	if strings.TrimSpace(s) == "" {
		return "", invalidInput("blank magic item instance id")
	}
	return MagicItemInstanceID(s), nil
}

// String returns the id as supplied.
func (i MagicItemInstanceID) String() string { return string(i) }

// Valid reports whether i is not blank.
func (i MagicItemInstanceID) Valid() bool { return strings.TrimSpace(string(i)) != "" }

// PairID names the pair a worn item belongs to (a boot and its twin). The host
// supplies it. Build one with NewPairID; the zero value means "no pair".
// Comparable and safe for concurrent use.
type PairID string

// NewPairID returns s as a pair id. It returns an error matching
// srderr.ErrInvalidInput for a blank s. It never panics.
func NewPairID(s string) (PairID, error) {
	if strings.TrimSpace(s) == "" {
		return "", invalidInput("blank pair id")
	}
	return PairID(s), nil
}

// String returns the id as supplied.
func (p PairID) String() string { return string(p) }
