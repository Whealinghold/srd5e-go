package content

import "slices"

// ItemRarity is the rarity of a magic item, in the order of the Magic Item
// Rarities and Values table (SRD 5.2.1, p. 206). The wire spelling is the
// printed name ("Very Rare"). It carries no value or price; see
// [ItemRarityFacts]. The zero ItemRarity is invalid. Comparable and safe for
// concurrent use.
type ItemRarity int

// The item rarities.
const (
	RarityCommon ItemRarity = iota + 1
	RarityUncommon
	RarityRare
	RarityVeryRare
	RarityLegendary
	RarityArtifact
)

func itemRarityNames() []string {
	return []string{"Common", "Uncommon", "Rare", "Very Rare", "Legendary", "Artifact"}
}

// Valid reports whether r is one of the six rarities.
func (r ItemRarity) Valid() bool { return enumValid(itemRarityNames(), int(r)) }

// String returns the printed name of r, or "unknown" for an undefined value.
func (r ItemRarity) String() string { return enumString(itemRarityNames(), int(r)) }

// ParseItemRarity returns the rarity spelled exactly s, such as "Very Rare".
// Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string, including "Very rare". It never panics.
func ParseItemRarity(s string) (ItemRarity, error) {
	v, err := parseEnum("ItemRarity", itemRarityNames(), s)
	return ItemRarity(v), err
}

// RaritySpec is the rarity a magic item's header prints, a closed family with
// three variants: [SingleRarity] (a header such as "Rare"), [RarityVaries]
// (the words "Rarity Varies") and [RarityByVariant] (a header such as
// "Uncommon (+1), Rare (+2), or Very Rare (+3)"). Consumers switch over it
// exhaustively. Every variant is immutable and safe for concurrent use.
type RaritySpec interface{ isRaritySpec() }

// SingleRarity is one rarity for the whole item. The zero value is invalid;
// build it with [NewSingleRarity]. Comparable.
type SingleRarity struct{ rarity ItemRarity }

// NewSingleRarity returns the single rarity r.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember when r is not one of the six rarities. It never panics.
func NewSingleRarity(r ItemRarity) (SingleRarity, error) {
	if !r.Valid() {
		return SingleRarity{}, blockError("SingleRarity", BlockInvalidMember)
	}
	return SingleRarity{rarity: r}, nil
}

// Rarity returns the rarity.
func (s SingleRarity) Rarity() ItemRarity { return s.rarity }

func (SingleRarity) isRaritySpec() {}

// RarityVaries is the header words "Rarity Varies": the rarity depends on a
// variant the item's text describes. It carries no data and its zero value is
// valid. Comparable.
type RarityVaries struct{}

func (RarityVaries) isRaritySpec() {}

// RarityOption is one rarity of a [RarityByVariant] header and the verbatim
// text of its parenthetical (SRD 5.2.1, "Horn of Valhalla", p. 226). The zero
// value is invalid; build it with [NewRarityOption]. Comparable.
type RarityOption struct {
	rarity  ItemRarity
	variant string
}

// NewRarityOption returns an option of rarity r for variant, which is kept
// verbatim.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for a rarity that is not a member, then BlockBlankText or
// BlockInvalidCharacter for a variant that breaks the text rule. It never
// panics.
func NewRarityOption(r ItemRarity, variant string) (RarityOption, error) {
	if !r.Valid() {
		return RarityOption{}, blockError("RarityOption", BlockInvalidMember)
	}
	if err := checkText("RarityOption", variant); err != nil {
		return RarityOption{}, err
	}
	return RarityOption{rarity: r, variant: variant}, nil
}

// Rarity returns the rarity of the option.
func (o RarityOption) Rarity() ItemRarity { return o.rarity }

// Variant returns the verbatim parenthetical text, such as "+1".
func (o RarityOption) Variant() string { return o.variant }

// MinRarityOptions is the fewest options a [RarityByVariant] holds.
const MinRarityOptions = 2

// RarityByVariant is one rarity per variant, in printed order. The zero value
// is invalid; build it with [NewRarityByVariant]. It holds a slice, so compare
// it with [RarityByVariant.Equal].
type RarityByVariant struct{ options []RarityOption }

// NewRarityByVariant returns a header of a copy of options, which are
// [MinRarityOptions] or more, each valid, with distinct variants. A rarity may
// repeat.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockOutOfRange for fewer than two options, BlockInvalidMember for an
// invalid option, BlockDuplicate for a repeated variant. It never panics.
func NewRarityByVariant(options []RarityOption) (RarityByVariant, error) {
	const block = "RarityByVariant"
	if len(options) < MinRarityOptions {
		return RarityByVariant{}, blockError(block, BlockOutOfRange)
	}
	seen := make(map[string]struct{}, len(options))
	for _, o := range options {
		if _, err := NewRarityOption(o.rarity, o.variant); err != nil {
			return RarityByVariant{}, blockError(block, BlockInvalidMember)
		}
		if _, dup := seen[o.variant]; dup {
			return RarityByVariant{}, blockError(block, BlockDuplicate)
		}
		seen[o.variant] = struct{}{}
	}
	return RarityByVariant{options: slices.Clone(options)}, nil
}

// Options returns the options in printed order, as a new slice.
func (r RarityByVariant) Options() []RarityOption { return slices.Clone(r.options) }

// Equal reports whether r and s hold the same options in the same order.
func (r RarityByVariant) Equal(s RarityByVariant) bool { return slices.Equal(r.options, s.options) }

func (RarityByVariant) isRaritySpec() {}

// validRaritySpec reports whether r is a variant its constructor would
// accept.
func validRaritySpec(r RaritySpec) bool {
	switch v := r.(type) {
	case SingleRarity:
		_, err := NewSingleRarity(v.rarity)
		return err == nil
	case RarityVaries:
		return true
	case RarityByVariant:
		_, err := NewRarityByVariant(v.options)
		return err == nil
	}
	return false // nil or a pointer to a variant: not a member of the closed set
}

// raritySpecEqual reports whether a and b are the same variant with equal
// members.
func raritySpecEqual(a, b RaritySpec) bool {
	if x, ok := a.(RarityByVariant); ok {
		y, ok := b.(RarityByVariant)
		return ok && x.Equal(y)
	}
	if _, ok := b.(RarityByVariant); ok {
		return false
	}
	return a == b
}
