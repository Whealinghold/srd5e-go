package content

// Kind identifies a category of content entry. The zero Kind is invalid.
//
// The values are declared in the order the decoder, [Set] and the ordering
// rules rely on, so a value is never inserted before an existing one. Kind
// is comparable and safe for concurrent use.
type Kind int

// The kinds of content entry, named for the singular of the SRD 5.2.1
// heading (SRD 5.2.1, "Contents", p. 2). The SRD "Animals" section is a
// stat-block section and maps to [KindMonster]. The toolbox catalogue kinds
// poison, trap and contagion arrive with their rows in Phase 5.
const (
	// KindSpecies is a Character Species.
	KindSpecies Kind = iota + 1
	// KindBackground is a Character Background.
	KindBackground
	// KindCharacterClass is a Class; its wire spelling is "characterClass".
	KindCharacterClass
	// KindFeat is a Feat.
	KindFeat
	// KindEquipment is a piece of Equipment.
	KindEquipment
	// KindSpell is a Spell.
	KindSpell
	// KindMagicItem is a Magic Item.
	KindMagicItem
	// KindMonster is a Monster or an Animal stat block.
	KindMonster
)

// String returns the wire spelling of k, or "unknown" for an undefined value.
func (k Kind) String() string {
	switch k {
	case KindSpecies:
		return "species"
	case KindBackground:
		return "background"
	case KindCharacterClass:
		return "characterClass"
	case KindFeat:
		return "feat"
	case KindEquipment:
		return "equipment"
	case KindSpell:
		return "spell"
	case KindMagicItem:
		return "magicItem"
	case KindMonster:
		return "monster"
	}
	return "unknown"
}

// Valid reports whether k is one of the declared kinds.
func (k Kind) Valid() bool { return k >= KindSpecies && k <= KindMonster }
