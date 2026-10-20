package content

// MagicItemCategory is the category a magic item's header prints, in the order
// of the Magic Item Categories table (SRD 5.2.1, "Magic Item Categories",
// p. 204). The wire spelling is the singular the headers print, spaces
// included ("Wondrous Item"); the table's plurals are not accepted. The zero
// MagicItemCategory is invalid. Comparable and safe for concurrent use.
type MagicItemCategory int

// The magic item categories.
const (
	MagicItemArmor MagicItemCategory = iota + 1
	MagicItemPotion
	MagicItemRing
	MagicItemRod
	MagicItemScroll
	MagicItemStaff
	MagicItemWand
	MagicItemWeapon
	MagicItemWondrousItem
)

func magicItemCategoryNames() []string {
	return []string{"Armor", "Potion", "Ring", "Rod", "Scroll", "Staff", "Wand", "Weapon", "Wondrous Item"}
}

// Valid reports whether c is one of the nine categories.
func (c MagicItemCategory) Valid() bool { return enumValid(magicItemCategoryNames(), int(c)) }

// String returns the printed singular name of c, or "unknown" for an
// undefined value.
func (c MagicItemCategory) String() string { return enumString(magicItemCategoryNames(), int(c)) }

// ParseMagicItemCategory returns the category spelled exactly s, such as
// "Wondrous Item". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string, including "Wondrous Items" and "armor". It never panics.
func ParseMagicItemCategory(s string) (MagicItemCategory, error) {
	v, err := parseEnum("MagicItemCategory", magicItemCategoryNames(), s)
	return MagicItemCategory(v), err
}
