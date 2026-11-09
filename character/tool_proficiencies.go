package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ToolProficiencies is the set of tools a creature has proficiency with, by tool
// id (SRD 5.2.1, "Tool Proficiency", p. 93). Each variant of a family (Gaming
// Set, Musical Instrument) needs its own proficiency, so a family id never
// stands for a variant. The zero value is the empty set. It holds a slice, so
// compare two with [ToolProficiencies.Equal], not ==. Safe for concurrent use.
type ToolProficiencies struct{ ids []content.ItemID }

// NewToolProficiencies returns the set of ids, copied, sorted and without
// duplicates. The order of ids does not matter. It cannot fail.
func NewToolProficiencies(ids ...content.ItemID) ToolProficiencies {
	out := slices.Clone(ids)
	slices.Sort(out)
	return ToolProficiencies{ids: slices.Compact(out)}
}

// MonsterToolProficiencies returns the proficiencies of a monster: exactly the
// tools of its stat block (SRD 5.2.1, "Tool Proficiency", p. 93).
func MonsterToolProficiencies(statBlockTools ...content.ItemID) ToolProficiencies {
	return NewToolProficiencies(statBlockTools...)
}

// Contains reports whether id is in the set. Proficiency is per entry id.
func (p ToolProficiencies) Contains(id content.ItemID) bool {
	_, found := slices.BinarySearch(p.ids, id)
	return found
}

// Plus returns a new set that also holds id; p is unchanged.
func (p ToolProficiencies) Plus(id content.ItemID) ToolProficiencies {
	return NewToolProficiencies(append(slices.Clone(p.ids), id)...)
}

// IDs returns the ids sorted, as a new slice.
func (p ToolProficiencies) IDs() []content.ItemID { return slices.Clone(p.ids) }

// Equal reports whether p and o hold the same ids.
func (p ToolProficiencies) Equal(o ToolProficiencies) bool { return slices.Equal(p.ids, o.ids) }
