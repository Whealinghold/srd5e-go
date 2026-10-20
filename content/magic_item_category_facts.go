package content

// ActivationTime is how long activating a magic item takes, a closed set of
// five members (SRD 5.2.1, "Using a Potion", p. 204; "Activating a Magic Item",
// p. 206). [ActivationDescribedElsewhere] means the item's description governs
// (an oil that "might take longer", a Spell Scroll that takes the spell's
// casting time) and is valid for any category. How the combat context maps
// the members to its action types is not decided here. The zero ActivationTime
// is invalid. Comparable and safe for concurrent use.
type ActivationTime int

// The activation times.
const (
	ActivationMagicAction ActivationTime = iota + 1
	ActivationBonusAction
	ActivationAction
	ActivationReaction
	ActivationDescribedElsewhere
)

func activationTimeNames() []string {
	return []string{"Magic action", "Bonus Action", "Action", "Reaction", "Described elsewhere"}
}

// Valid reports whether t is one of the five activation times.
func (t ActivationTime) Valid() bool { return enumValid(activationTimeNames(), int(t)) }

// String returns the name of t, or "unknown" for an undefined value.
func (t ActivationTime) String() string { return enumString(activationTimeNames(), int(t)) }

// MagicItemCategoryFacts holds the static defaults of a magic item category
// ("Unless its description notes otherwise"): whether it is consumable, an
// Arcane Focus, a nonmagical Quarterstaff or must be worn, and its activation
// time (SRD 5.2.1, pp. 204-206). A departure by one item is an override in its
// [MagicItemFacts]. Get one from [MagicItemCategoryFacts.Of]; the zero value
// is what Of returns for a category that is not a member. Comparable and safe
// for concurrent use.
type MagicItemCategoryFacts struct {
	consumable, arcaneFocus, nonmagicalQuarterstaff, mustBeWorn bool
	activationTime                                              ActivationTime
}

// Of returns the defaults of category c, exactly as the Category facts table
// of content-magic-items.md: consumable for potions and scrolls only, an Arcane
// Focus for rods, staffs and wands only, a nonmagical Quarterstaff for staffs
// only, worn for armor and rings only, and a Bonus Action to activate a potion
// and a Magic action for every other category. It returns the zero facts for
// a value that is not a category. It never panics.
func (MagicItemCategoryFacts) Of(c MagicItemCategory) MagicItemCategoryFacts {
	if !c.Valid() {
		return MagicItemCategoryFacts{}
	}
	f := MagicItemCategoryFacts{activationTime: ActivationMagicAction}
	switch c {
	case MagicItemPotion:
		f.consumable, f.activationTime = true, ActivationBonusAction
	case MagicItemScroll:
		f.consumable = true
	case MagicItemRod, MagicItemWand:
		f.arcaneFocus = true
	case MagicItemStaff:
		f.arcaneFocus, f.nonmagicalQuarterstaff = true, true
	case MagicItemArmor, MagicItemRing:
		f.mustBeWorn = true
	case MagicItemWeapon, MagicItemWondrousItem:
		// the defaults of the zero facts: nothing is set
	}
	return f
}

// Consumable reports whether items of the category are consumed on use.
func (f MagicItemCategoryFacts) Consumable() bool { return f.consumable }

// ArcaneFocus reports whether items of the category are Arcane Focuses.
func (f MagicItemCategoryFacts) ArcaneFocus() bool { return f.arcaneFocus }

// NonmagicalQuarterstaff reports whether items of the category are also a
// nonmagical Quarterstaff.
func (f MagicItemCategoryFacts) NonmagicalQuarterstaff() bool { return f.nonmagicalQuarterstaff }

// MustBeWorn reports whether items of the category must be worn to work.
func (f MagicItemCategoryFacts) MustBeWorn() bool { return f.mustBeWorn }

// ActivationTime returns the default activation time.
func (f MagicItemCategoryFacts) ActivationTime() ActivationTime { return f.activationTime }
