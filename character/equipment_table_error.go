package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// TableCoverageError reports decoded rows that do not cover each spell level 0
// to 9 exactly once. [NewScrollScribing] fills Missing and Repeated;
// [NewSpellcastingServices] fills Uncovered and Overlapping. Each list is
// ordered by level. It matches srderr.ErrInvalidInput with errors.Is, and
// another *TableCoverageError with the same four lists.
type TableCoverageError struct {
	Missing     []content.SpellLevel
	Repeated    []content.SpellLevel
	Uncovered   []content.SpellLevel
	Overlapping []content.SpellLevel
}

// Error renders the error. The text is not a contract.
func (e *TableCoverageError) Error() string {
	return "character: table does not cover each spell level once"
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *TableCoverageError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *TableCoverageError with the same lists.
func (e *TableCoverageError) Is(target error) bool {
	t, ok := target.(*TableCoverageError)
	return ok && t != nil && e != nil &&
		slices.Equal(e.Missing, t.Missing) && slices.Equal(e.Repeated, t.Repeated) &&
		slices.Equal(e.Uncovered, t.Uncovered) && slices.Equal(e.Overlapping, t.Overlapping)
}

// coverage counts how many rows cover each spell level and returns the levels
// covered never and more than once.
func coverage(counts [content.MaxSpellLevel + 1]int) (none, many []content.SpellLevel) {
	for level, n := range counts {
		switch {
		case n == 0:
			none = append(none, content.SpellLevel(level))
		case n > 1:
			many = append(many, content.SpellLevel(level))
		}
	}
	return none, many
}
