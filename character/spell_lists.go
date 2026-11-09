package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// PreparedSpells is a class's prepared level 1+ spells in the order chosen, plus
// the always-prepared spells, which count as prepared for casting but not
// against the prepared limit (SRD 5.2.1, "Always-Prepared Spells", p. 104). It
// holds slices, so compare two with reflect.DeepEqual, not ==. The zero value
// holds none.
type PreparedSpells struct{ ids, always []content.SpellID }

// Spells returns a new slice of the chosen prepared spells, always-prepared ones
// excluded.
func (p PreparedSpells) Spells() []content.SpellID { return slices.Clone(p.ids) }

// AlwaysPrepared returns a new slice of the always-prepared spells.
func (p PreparedSpells) AlwaysPrepared() []content.SpellID { return slices.Clone(p.always) }

// Count returns the number of chosen prepared spells (what the limit counts).
func (p PreparedSpells) Count() int { return len(p.ids) }

// IsPrepared reports whether id is prepared, chosen or always-prepared.
func (p PreparedSpells) IsPrepared(id content.SpellID) bool {
	return slices.Contains(p.ids, id) || slices.Contains(p.always, id)
}

// CantripsKnown is the cantrips a class has chosen, in order, without those a
// feature grants (SRD 5.2.1, "Cantrip", p. 178). The zero value holds none.
type CantripsKnown struct{ ids []content.SpellID }

// Spells returns a new slice of the known cantrips.
func (c CantripsKnown) Spells() []content.SpellID { return slices.Clone(c.ids) }

// Count returns the number of known cantrips.
func (c CantripsKnown) Count() int { return len(c.ids) }

// Spellbook is the set of spells in a Wizard's spellbook (SRD 5.2.1, "Wizard",
// p. 78). The zero Spellbook is empty.
type Spellbook struct{ ids []content.SpellID }

// NewSpellbook returns the spellbook holding ids, copied, without repeats, in
// first-seen order.
func NewSpellbook(ids []content.SpellID) Spellbook {
	var out []content.SpellID
	for _, id := range ids {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return Spellbook{ids: out}
}

// Spells returns a new slice of the spells in the book.
func (b Spellbook) Spells() []content.SpellID { return slices.Clone(b.ids) }

// Contains reports whether the book holds id.
func (b Spellbook) Contains(id content.SpellID) bool { return slices.Contains(b.ids, id) }

// GrantedCantrip is a cantrip a feature grants to a class, such as Blessed
// Warrior or Druidic Warrior (SRD 5.2.1, pp. 54, 59): the spell, the class whose
// ability and DC it uses, and the class whose spell list the feature names. It
// is comparable. The zero GrantedCantrip is invalid.
type GrantedCantrip struct {
	spell content.SpellID
	class SpellcastingClass
	list  content.ClassID
}

// NewGrantedCantrip returns the granted cantrip. It returns an error matching
// srderr.ErrInvalidInput for an invalid spell id, class or list. It never panics.
func NewGrantedCantrip(spell content.SpellID, class SpellcastingClass, list content.ClassID) (GrantedCantrip, error) {
	if !spell.Valid() || !class.Valid() || !list.Valid() {
		return GrantedCantrip{}, invalidInput("granted cantrip: invalid spell, class or list")
	}
	return GrantedCantrip{spell: spell, class: class, list: list}, nil
}

// Spell returns the cantrip's id.
func (g GrantedCantrip) Spell() content.SpellID { return g.spell }

// Class returns the class whose spellcasting ability the cantrip uses.
func (g GrantedCantrip) Class() SpellcastingClass { return g.class }

// ListClass returns the class whose spell list the granting feature names.
func (g GrantedCantrip) ListClass() content.ClassID { return g.list }
