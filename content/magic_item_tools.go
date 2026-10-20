package content

import "slices"

// MagicItemToolRule is the tools needed to craft a magic item of a category, a
// closed family with three variants: [FixedTools], [ByBaseItemTools] and
// [TinkersOrBaseItemTools] (SRD 5.2.1, "Magic Item Tools", p. 207). Tools are
// ids of equipment Tool entries, resolved by id only. Consumers switch over it
// exhaustively. Every variant is immutable and safe for concurrent use.
type MagicItemToolRule interface{ isMagicItemToolRule() }

// FixedTools is a rule naming a fixed set of tools, any one of which suffices
// when there are several. The zero value is invalid; build it with
// [NewFixedTools]. It holds a slice, so compare it with [FixedTools.Equal].
type FixedTools struct{ tools []ItemID }

// NewFixedTools returns a rule of a copy of tools: a non-empty set of valid
// ids.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for an empty set, BlockInvalidMember for an invalid id and
// BlockDuplicate for a repeat. It never panics.
func NewFixedTools(tools []ItemID) (FixedTools, error) {
	if err := checkSet("FixedTools", tools, 1, noUpperBound, ItemID.Valid); err != nil {
		return FixedTools{}, err
	}
	return FixedTools{tools: slices.Clone(tools)}, nil
}

// Tools returns the tool ids as a new slice.
func (f FixedTools) Tools() []ItemID { return slices.Clone(f.tools) }

// Equal reports whether f and g name the same tools in the same order.
func (f FixedTools) Equal(g FixedTools) bool { return slices.Equal(f.tools, g.tools) }

func (FixedTools) isMagicItemToolRule() {}

// ByBaseItemTools is the rule of armor and weapons: the tools depend on the
// kind of armor or weapon, read from the base item. The zero value is valid.
type ByBaseItemTools struct{}

func (ByBaseItemTools) isMagicItemToolRule() {}

// TinkersOrBaseItemTools is the rule of wondrous items: Tinker's Tools or the
// tool required to make the nonmagical item the magic item is based on. The
// zero value is valid.
type TinkersOrBaseItemTools struct{}

func (TinkersOrBaseItemTools) isMagicItemToolRule() {}

// MagicItemToolTable is the Magic Item Tools table. It holds no state; its
// zero value is the table. Safe for concurrent use.
type MagicItemToolTable struct{}

// Of returns the rule of category c: Jeweler's Tools for rings, Woodcarver's
// Tools for rods, staffs and wands, Calligrapher's Supplies for scrolls,
// Alchemist's Supplies or a Herbalism Kit for potions, the tools of the base
// item for armor and weapons, and Tinker's Tools or those of the base item for
// wondrous items. It returns nil for a value that is not a category. It never
// panics.
func (MagicItemToolTable) Of(c MagicItemCategory) MagicItemToolRule {
	fixed := func(ids ...string) MagicItemToolRule {
		tools := make([]ItemID, len(ids))
		for i, id := range ids {
			tools[i] = ItemID(id)
		}
		return FixedTools{tools: tools}
	}
	switch c {
	case MagicItemArmor, MagicItemWeapon:
		return ByBaseItemTools{}
	case MagicItemPotion:
		return fixed("alchemists_supplies", "herbalism_kit")
	case MagicItemRing:
		return fixed("jewelers_tools")
	case MagicItemRod, MagicItemStaff, MagicItemWand:
		return fixed("woodcarvers_tools")
	case MagicItemScroll:
		return fixed("calligraphers_supplies")
	case MagicItemWondrousItem:
		return TinkersOrBaseItemTools{}
	}
	return nil
}
