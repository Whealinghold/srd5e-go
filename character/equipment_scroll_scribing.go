package character

import "github.com/Whealinghold/srd5e-go/content"

// ScrollScribing is the Spell Scroll Costs table (SRD 5.2.1, "Scribing Spell
// Scrolls", p. 103): one row for each spell level 0 to 9. Build it with
// [NewScrollScribing]; the zero value is "no table" ([ScrollScribing.IsZero]).
// It is an immutable value that holds its own copy of the rows.
type ScrollScribing struct {
	rows  [content.MaxSpellLevel + 1]content.SpellScrollCost
	valid bool
}

// NewScrollScribing returns the table of rows. It copies the rows, so later
// changes to the slice change nothing. It returns a *[TableCoverageError]
// (matching srderr.ErrInvalidInput) with the missing and repeated levels,
// each ordered by level, unless the rows cover each level 0 to 9 exactly once.
func NewScrollScribing(rows []content.SpellScrollCost) (ScrollScribing, error) {
	var counts [content.MaxSpellLevel + 1]int
	var t ScrollScribing
	for _, r := range rows {
		if level := r.SpellLevel(); content.ValidSpellLevel(level) {
			counts[level]++
			t.rows[level] = r
		}
	}
	if missing, repeated := coverage(counts); len(missing) > 0 || len(repeated) > 0 {
		return ScrollScribing{}, &TableCoverageError{Missing: missing, Repeated: repeated}
	}
	t.valid = true
	return t, nil
}

// MustScrollScribing is like [NewScrollScribing] but panics with its error when
// the rows do not cover each level once.
func MustScrollScribing(rows []content.SpellScrollCost) ScrollScribing {
	t, err := NewScrollScribing(rows)
	if err != nil {
		panic(err)
	}
	return t
}

// IsZero reports whether t is the zero value, "no table".
func (t ScrollScribing) IsZero() bool { return !t.valid }

// ForSpellLevel returns the row for level. It returns the zero row for the
// zero table or a level outside 0 to 9.
func (t ScrollScribing) ForSpellLevel(level content.SpellLevel) content.SpellScrollCost {
	if !t.valid || !content.ValidSpellLevel(int(level)) {
		return content.SpellScrollCost{}
	}
	return t.rows[level]
}
