package content

import "slices"

// MagicItem marks a [MagicItemEntry]: an interface with one exported marker
// method, implemented by MagicItemEntry alone in this module. Test for it with
// _, ok := entry.(MagicItem). The equipment gear entries that share a name
// with a catalogue entry do not implement it; look the catalogue entry up by
// id. The set is open: a consumer's own entry type may implement it in its own
// package.
type MagicItem interface {
	// IsMagicItem marks the type as a magic item catalogue entry.
	IsMagicItem()
}

// MagicItemFields holds the members of a [MagicItemEntry] beyond its
// identity, for [NewMagicItemEntry]. Construct it with a keyed literal.
type MagicItemFields struct {
	// Category is the header category (required).
	Category MagicItemCategory
	// CategoryNote is the text of the header's category parenthetical with the
	// outer parentheses removed; "" means the header prints none.
	CategoryNote string
	// Rarity is the header rarity (required).
	Rarity RaritySpec
	// Attunement is the header Attunement; nil means [NoAttunement].
	Attunement Attunement
	// Description is a non-empty list of text blocks in printed order.
	Description []TextBlock
}

// MagicItemEntry is a magic item of the catalogue (SRD 5.2.1, "Magic Items
// A-Z", pp. 209-253): the header facts the SRD prints as structure and the
// description as verbatim text blocks. It is an [Entry] of kind
// [KindMagicItem] and implements [MagicItem]. Charges, bonuses and effects stay
// text; per-item facts are the host-supplied [MagicItemFacts]. Immutable, with
// its collections copied on construction and on read; the zero MagicItemEntry
// is invalid. It holds slices, so it is not comparable with ==; compare with
// [MagicItemEntry.Equal] or [Set.Equal]. Safe for concurrent use.
type MagicItemEntry struct {
	id           ID
	name         string
	source       SourceCitation
	category     MagicItemCategory
	categoryNote string
	rarity       RaritySpec
	attunement   Attunement
	description  []TextBlock
}

var (
	_ Entry     = MagicItemEntry{}
	_ MagicItem = MagicItemEntry{}
)

// NewMagicItemEntry returns the magic item id called name, with the members of
// f. A zero source means the entry cites none.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError] for the first rule broken,
// in the order of the members: BlockInvalidName; BlockInvalidMember for a
// category that is not a member; BlockBlankText or BlockInvalidCharacter for a
// non-empty category note that breaks the text rule; BlockMissing for a nil
// rarity and BlockInvalidMember for an invalid one; BlockInvalidMember for an
// invalid Attunement; BlockEmpty and BlockInvalidMember for the description.
// It never panics.
func NewMagicItemEntry(id MagicItemID, name string, source SourceCitation, f MagicItemFields) (MagicItemEntry, error) {
	const block = "MagicItemEntry"
	pid, err := ParseID(string(id))
	if err != nil {
		return MagicItemEntry{}, err
	}
	if err := checkName(block, name); err != nil {
		return MagicItemEntry{}, err
	}
	if !f.Category.Valid() {
		return MagicItemEntry{}, blockError(block, BlockInvalidMember)
	}
	if f.CategoryNote != "" { // equivalent mutant: a non-empty string is above "", so != and > agree
		if err := checkText(block, f.CategoryNote); err != nil {
			return MagicItemEntry{}, err
		}
	}
	if f.Rarity == nil {
		return MagicItemEntry{}, blockError(block, BlockMissing)
	}
	if !validRaritySpec(f.Rarity) {
		return MagicItemEntry{}, blockError(block, BlockInvalidMember)
	}
	attunement := f.Attunement
	if attunement == nil {
		attunement = NoAttunement{}
	}
	if !validAttunement(attunement) {
		return MagicItemEntry{}, blockError(block, BlockInvalidMember)
	}
	if len(f.Description) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return MagicItemEntry{}, blockError(block, BlockEmpty)
	}
	if slices.ContainsFunc(f.Description, func(b TextBlock) bool { return !validTextBlock(b) }) {
		return MagicItemEntry{}, blockError(block, BlockInvalidMember)
	}
	return MagicItemEntry{
		id: pid, name: name, source: source, category: f.Category, categoryNote: f.CategoryNote,
		rarity: f.Rarity, attunement: attunement, description: slices.Clone(f.Description),
	}, nil
}

// MustMagicItemEntry is like [NewMagicItemEntry] but panics with its error when
// the arguments are invalid.
func MustMagicItemEntry(id MagicItemID, name string, source SourceCitation, f MagicItemFields) MagicItemEntry {
	v, err := NewMagicItemEntry(id, name, source, f)
	if err != nil {
		panic(err)
	}
	return v
}

// Key returns the key of the entry: kind [KindMagicItem] and its id.
func (e MagicItemEntry) Key() Key { return NewKey(KindMagicItem, e.id) }

// IsMagicItem marks the entry as a [MagicItem].
func (MagicItemEntry) IsMagicItem() {}

// ID returns the magic item id.
func (e MagicItemEntry) ID() MagicItemID { return MagicItemID(e.id.String()) }

// Name returns the heading exactly as printed.
func (e MagicItemEntry) Name() string { return e.name }

// Source returns the citation of the entry, or the zero [SourceCitation] when
// it cites none.
func (e MagicItemEntry) Source() SourceCitation { return e.source }

// Category returns the header category.
func (e MagicItemEntry) Category() MagicItemCategory { return e.category }

// CategoryNote returns the text of the category parenthetical, or "" when the
// header prints none.
func (e MagicItemEntry) CategoryNote() string { return e.categoryNote }

// Rarity returns the header rarity.
func (e MagicItemEntry) Rarity() RaritySpec { return e.rarity }

// Attunement returns the header Attunement, [NoAttunement] when it prints none.
func (e MagicItemEntry) Attunement() Attunement { return e.attunement }

// Description returns the description blocks in printed order, as a new slice.
func (e MagicItemEntry) Description() []TextBlock { return slices.Clone(e.description) }

// Equal reports whether e and o are the same entry: every member equal, the
// description blocks in the same order.
func (e MagicItemEntry) Equal(o MagicItemEntry) bool {
	return e.id == o.id && e.name == o.name && e.source == o.source && e.category == o.category &&
		e.categoryNote == o.categoryNote && raritySpecEqual(e.rarity, o.rarity) &&
		(e.attunement == o.attunement) && slices.EqualFunc(e.description, o.description, textBlockEqual)
}

// validTextBlock reports whether b is a variant its constructor would accept.
func validTextBlock(b TextBlock) bool {
	var err error
	switch v := b.(type) {
	case TextParagraph:
		_, err = NewTextParagraph(v.text)
	case TextEntry:
		_, err = NewTextEntry(v.name, v.text)
	case TextBulletList:
		_, err = NewTextBulletList(v.items)
	case TextTable:
		if v.hasTitle { // equivalent mutant: a title that was built is valid, so checking the table without it agrees
			_, err = NewTitledTextTable(v.title, v.header, v.rows)
		} else {
			_, err = NewTextTable(v.header, v.rows)
		}
	case TextEmbeddedStatBlock:
		_, err = NewTextEmbeddedStatBlock(v.lines)
	default:
		return false
	}
	return err == nil
}

// textBlockEqual reports whether a and b are the same variant with equal
// members.
func textBlockEqual(a, b TextBlock) bool {
	switch x := a.(type) {
	case TextBulletList:
		y, ok := b.(TextBulletList)
		return ok && x.Equal(y)
	case TextTable:
		y, ok := b.(TextTable)
		return ok && x.Equal(y)
	case TextEmbeddedStatBlock:
		y, ok := b.(TextEmbeddedStatBlock)
		return ok && x.Equal(y)
	}
	switch b.(type) {
	case TextBulletList, TextTable, TextEmbeddedStatBlock:
		return false
	}
	return a == b
}
