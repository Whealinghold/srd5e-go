package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// PreparedSpellSet is the spells a character has available each day: the
// prepared spells (chosen and always prepared) and the cantrips known or
// granted. Interpretation: a known cantrip counts as prepared, since it is
// available every day. The zero value is the empty set. It holds a slice, so
// compare two with [PreparedSpellSet.Equal]. Safe for concurrent use.
type PreparedSpellSet struct{ ids []content.SpellID }

// NewPreparedSpellSet returns the set of ids, copied, sorted and without duplicates.
func NewPreparedSpellSet(ids ...content.SpellID) PreparedSpellSet {
	out := slices.Clone(ids)
	slices.Sort(out)
	return PreparedSpellSet{ids: slices.Compact(out)}
}

// PreparedSpellSetFromSpellcasting returns the union, over classes, of the
// prepared spells (chosen and always prepared), the cantrips known (those
// chosen under an extra-cantrip allowance included) and the granted cantrips. A
// Wizard's spellbook spells count only when prepared. An empty input gives the
// empty set (SRD 5.2.1, "Prerequisites for the Scribe", p. 103).
func PreparedSpellSetFromSpellcasting(classes []ClassSpellcasting) PreparedSpellSet {
	var ids []content.SpellID
	for _, c := range classes {
		ids = slices.Concat(ids, c.prepared.ids, c.prepared.always, c.cantrips.ids)
		for _, g := range c.granted {
			ids = append(ids, g.spell)
		}
	}
	return NewPreparedSpellSet(ids...)
}

// Contains reports whether id is in the set.
func (s PreparedSpellSet) Contains(id content.SpellID) bool {
	_, found := slices.BinarySearch(s.ids, id)
	return found
}

// IDs returns the ids sorted, as a new slice.
func (s PreparedSpellSet) IDs() []content.SpellID { return slices.Clone(s.ids) }

// Equal reports whether s and o hold the same ids.
func (s PreparedSpellSet) Equal(o PreparedSpellSet) bool { return slices.Equal(s.ids, o.ids) }
