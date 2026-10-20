package content

import "slices"

// ClassSpellList is the spell list of a class, derived from the classes each
// [Spell] names (SRD 5.2.1, "Class Spell Lists", p. 105): for each spell level
// 0 to 9, the ids of the spells whose classes contain the class. It is never
// stored on a class entry, so a list for a ClassID no loaded class holds is
// derived the same way. Build it with [DeriveClassSpellList]. The zero value
// is the empty list of no class. It holds slices, so it is not comparable with
// ==; use [ClassSpellList.Equal]. Safe for concurrent use.
type ClassSpellList struct {
	class   ClassID
	byLevel [MaxSpellLevel + 1][]SpellID
}

// DeriveClassSpellList returns the spell list of class over spells (content-spells.md
// FR-120, FR-121). Within a level the ids are in ordinal order of their string
// value, so the result does not depend on the order of spells, and a spell that
// appears twice is listed once. A class no spell names, and a non-caster, give
// empty levels. It does not alter spells and never panics.
func DeriveClassSpellList(class ClassID, spells []Spell) ClassSpellList {
	list := ClassSpellList{class: class}
	for _, s := range spells {
		if s.level.Valid() && slices.Contains(s.classes, class) {
			list.byLevel[s.level] = append(list.byLevel[s.level], s.ID())
		}
	}
	for i := range list.byLevel {
		slices.Sort(list.byLevel[i])
		list.byLevel[i] = slices.Compact(list.byLevel[i])
	}
	return list
}

// ClassID returns the class the list is for.
func (l ClassSpellList) ClassID() ClassID { return l.class }

// IDsAt returns the ids of the spells of level on the list, in ordinal order,
// as a new slice. It returns nil for a level outside 0 to 9 and for a level
// with no spell.
func (l ClassSpellList) IDsAt(level SpellLevel) []SpellID {
	if !level.Valid() {
		return nil
	}
	return slices.Clone(l.byLevel[level])
}

// Equal reports whether l and o are for the same class and hold the same ids
// at every level.
func (l ClassSpellList) Equal(o ClassSpellList) bool {
	if l.class != o.class {
		return false
	}
	for i := range l.byLevel {
		if !slices.Equal(l.byLevel[i], o.byLevel[i]) {
			return false
		}
	}
	return true
}
