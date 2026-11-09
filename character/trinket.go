package character

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// TrinketCount is the number of entries of the Trinkets table, a 1d100 table
// (SRD 5.2.1, "Trinkets", pp. 26-27).
const TrinketCount = 100

// Trinket is one entry of the Trinkets table: the description text and nothing
// else. It is an immutable value, comparable with ==. The zero Trinket is
// invalid; build it with NewTrinket. Safe for concurrent use.
type Trinket struct{ description string }

// NewTrinket returns the Trinket with description, kept verbatim. It returns a
// *TrinketError (matching srderr.ErrInvalidInput) for a blank description. It
// never panics.
func NewTrinket(description string) (Trinket, error) {
	if strings.TrimSpace(description) == "" {
		return Trinket{}, &TrinketError{Problem: TrinketBlankDescription}
	}
	return Trinket{description: description}, nil
}

// Description returns the text as it was given.
func (t Trinket) Description() string { return t.description }

// IsZero reports whether t is the invalid zero Trinket.
func (t Trinket) IsZero() bool { return t.description == "" }

// TrinketProblem names an invariant of the Trinkets table that an input
// violates. The zero value is not a problem.
type TrinketProblem int

// The Trinket problems.
const (
	// TrinketBlankDescription: a description is empty or only whitespace. For a
	// table entry, InvalidValue is its roll, 1 to 100.
	TrinketBlankDescription TrinketProblem = iota + 1
	// TrinketTableSize: the table does not hold exactly 100 entries.
	// InvalidValue is the count given.
	TrinketTableSize
	// TrinketRollOutOfRange: a roll outside 1 to 100. InvalidValue is the roll.
	TrinketRollOutOfRange
)

// String returns "blankDescription", "tableSize", "rollOutOfRange" or "unknown".
func (p TrinketProblem) String() string {
	switch p {
	case TrinketBlankDescription:
		return "blankDescription"
	case TrinketTableSize:
		return "tableSize"
	case TrinketRollOutOfRange:
		return "rollOutOfRange"
	}
	return "unknown"
}

// TrinketError reports an invalid Trinket argument with the offending value. It
// matches srderr.ErrInvalidInput with errors.Is.
type TrinketError struct {
	Problem      TrinketProblem
	InvalidValue int
}

// Error renders the error. The text is not a contract.
func (e *TrinketError) Error() string { return "character: invalid trinket: " + e.Problem.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (e *TrinketError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *TrinketError with the same Problem and
// InvalidValue.
func (e *TrinketError) Is(target error) bool {
	t, ok := target.(*TrinketError)
	return ok && t != nil && e != nil && *t == *e
}

// TrinketTable is the Trinkets table: exactly 100 entries, one for each d100
// result 01 to 99 and 00 (a roll of 100), in that order. The entries are game
// content a host supplies from its own copy of SRD 5.2.1; nothing in this module
// bundles or decodes them. It is an immutable value, comparable with ==. The zero
// TrinketTable is invalid; see IsZero. Safe for concurrent use.
type TrinketTable struct{ entries [TrinketCount]Trinket }

// NewTrinketTable returns the table of entries, copied; entries[0] is the roll 1
// and entries[99] the entry printed as 00.
//
// It returns a *TrinketError (matching srderr.ErrInvalidInput) with
// TrinketTableSize for a count other than 100, then with TrinketBlankDescription
// for the first zero Trinket. It never panics.
func NewTrinketTable(entries []Trinket) (TrinketTable, error) {
	if len(entries) != TrinketCount {
		return TrinketTable{}, &TrinketError{Problem: TrinketTableSize, InvalidValue: len(entries)}
	}
	if i := slices.IndexFunc(entries, Trinket.IsZero); i >= 0 {
		return TrinketTable{}, &TrinketError{Problem: TrinketBlankDescription, InvalidValue: i + 1}
	}
	var t TrinketTable
	copy(t.entries[:], entries)
	return t, nil
}

// MustTrinketTable is NewTrinketTable for literals and tests. It panics with the
// error NewTrinketTable returns, and only then.
func MustTrinketTable(entries []Trinket) TrinketTable {
	t, err := NewTrinketTable(entries)
	if err != nil {
		panic(err)
	}
	return t
}

// EntryFor returns the entry of roll: the entry numbered roll for 1 to 99 and the
// entry printed as 00 for 100.
//
// It returns a *TrinketError (matching srderr.ErrInvalidInput) with
// TrinketRollOutOfRange for a roll outside 1 to 100, and with TrinketTableSize
// for the zero table. It never panics.
func (t TrinketTable) EntryFor(roll int) (Trinket, error) {
	switch {
	case t.IsZero():
		return Trinket{}, &TrinketError{Problem: TrinketTableSize}
	case roll < 1 || roll > TrinketCount:
		return Trinket{}, &TrinketError{Problem: TrinketRollOutOfRange, InvalidValue: roll}
	}
	return t.entries[roll-1], nil
}

// IsZero reports whether t is the invalid zero TrinketTable.
func (t TrinketTable) IsZero() bool { return t.entries[0].IsZero() }
