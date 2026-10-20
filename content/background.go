package content

import (
	"math"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// The counts and limits of a [Background], shared by the constructors and the
// decoder.
const (
	// BackgroundAbilityScores is the number of distinct abilities a background
	// lists (SRD 5.2.1, "Ability Scores", p. 83).
	BackgroundAbilityScores = 3
	// BackgroundSkillProficiencies is the number of distinct skills a
	// background grants (SRD 5.2.1, "Skill Proficiencies", p. 83).
	BackgroundSkillProficiencies = 2
	// MaxEquipmentQuantity is the highest quantity of one equipment item, and
	// MaxEquipmentGold the highest gold of an option. The SRD sets neither;
	// the bounds keep the values the same on every platform integer width.
	MaxEquipmentQuantity = math.MaxInt32
	// MaxEquipmentGold: see [MaxEquipmentQuantity].
	MaxEquipmentGold = math.MaxInt32
)

// EquipmentItem is one item of a background's equipment option: its printed
// name, a parenthetical included, and a quantity (SRD 5.2.1, "Background
// Descriptions", p. 83). The zero value is invalid; build it with
// [NewEquipmentItem]. Comparable.
type EquipmentItem struct {
	name     string
	quantity int
}

// NewEquipmentItem returns an item called name, with a quantity from 1 to
// [MaxEquipmentQuantity].
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidName for a name that breaks the name rule (schema FR-027), then
// BlockOutOfRange for a quantity outside the range. It never panics.
func NewEquipmentItem(name string, quantity int) (EquipmentItem, error) {
	const block = "EquipmentItem"
	if err := checkName(block, name); err != nil {
		return EquipmentItem{}, err
	}
	if quantity < 1 || quantity > MaxEquipmentQuantity {
		return EquipmentItem{}, blockError(block, BlockOutOfRange)
	}
	return EquipmentItem{name: name, quantity: quantity}, nil
}

// Name returns the printed name.
func (i EquipmentItem) Name() string { return i.name }

// Quantity returns the quantity.
func (i EquipmentItem) Quantity() int { return i.quantity }

func (i EquipmentItem) valid() bool {
	return validName(i.name) && i.quantity >= 1 && i.quantity <= MaxEquipmentQuantity
}

// EquipmentOption is one of the two equipment options of a background: a list
// of items and an amount of gold. The zero value is a valid option with no
// items and no gold; build others with [NewEquipmentOption]. It holds a slice,
// so it is not comparable with ==; compare with [EquipmentOption.Equal].
type EquipmentOption struct {
	items []EquipmentItem
	gold  int
}

// NewEquipmentOption returns an option of items, which may be none, and gold,
// a whole number of Gold Pieces from 0 to [MaxEquipmentGold]. The items are
// copied.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for an item that is not valid, then BlockOutOfRange for
// gold outside the range. It never panics.
func NewEquipmentOption(items []EquipmentItem, gold int) (EquipmentOption, error) {
	const block = "EquipmentOption"
	for _, i := range items {
		if !i.valid() {
			return EquipmentOption{}, blockError(block, BlockInvalidMember)
		}
	}
	if gold < 0 || gold > MaxEquipmentGold {
		return EquipmentOption{}, blockError(block, BlockOutOfRange)
	}
	return EquipmentOption{items: cloneOrNil(items), gold: gold}, nil
}

// Items returns the items in printed order, as a new slice.
func (o EquipmentOption) Items() []EquipmentItem { return slices.Clone(o.items) }

// Gold returns the Gold Pieces.
func (o EquipmentOption) Gold() int { return o.gold }

// EquipmentOptions is the pair of options of a background, "A" and "B" on the
// wire (SRD 5.2.1, "Equipment", p. 83). The zero value holds two empty
// options. It is not comparable with ==; compare with [EquipmentOptions.Equal].
type EquipmentOptions struct{ a, b EquipmentOption }

// NewEquipmentOptions returns the pair of option a and option b. Both are
// already valid, so it cannot fail.
func NewEquipmentOptions(a, b EquipmentOption) EquipmentOptions {
	return EquipmentOptions{a: a, b: b}
}

// A returns option A.
func (o EquipmentOptions) A() EquipmentOption { return o.a }

// B returns option B.
func (o EquipmentOptions) B() EquipmentOption { return o.b }

// BackgroundFeatReference is the feat a background grants, by id, and the
// printed qualifier beside it, such as "Cleric" (SRD 5.2.1, "Acolyte", "Sage",
// p. 83). The id must name a feat of category Origin in the same set; that is
// checked by [ValidateSet], not here. The zero value is invalid; build it with
// [NewBackgroundFeatReference]. Comparable.
type BackgroundFeatReference struct {
	featID FeatID
	choice string
}

// NewBackgroundFeatReference returns the reference to featID with the printed
// qualifier choice, or none when choice is "".
//
// It returns the *[IDError] of [ParseFeatID] for an empty or malformed id, then
// a *[BlockError] wrapping srderr.ErrInvalidInput, BlockBlankText or
// BlockInvalidCharacter, for a non-empty choice that is blank, has a line feed
// or holds a rejected character. It never panics.
func NewBackgroundFeatReference(featID FeatID, choice string) (BackgroundFeatReference, error) {
	if _, err := ParseFeatID(string(featID)); err != nil {
		return BackgroundFeatReference{}, err
	}
	if choice != "" { // equivalent mutant: a non-empty string is above "", so != and > agree
		switch {
		case blankText(choice):
			return BackgroundFeatReference{}, blockError("BackgroundFeatReference", BlockBlankText)
		case !validLine(choice):
			return BackgroundFeatReference{}, blockError("BackgroundFeatReference", BlockInvalidCharacter)
		}
	}
	return BackgroundFeatReference{featID: featID, choice: choice}, nil
}

// FeatID returns the id of the feat.
func (r BackgroundFeatReference) FeatID() FeatID { return r.featID }

// Choice returns the printed qualifier, or "" when there is none.
func (r BackgroundFeatReference) Choice() string { return r.choice }

// BackgroundFields holds the members of a [Background] beyond its identity,
// for [NewBackground]. Construct it with a keyed literal.
type BackgroundFields struct {
	// AbilityScores lists exactly [BackgroundAbilityScores] distinct abilities
	// in printed order.
	AbilityScores []ability.Ability
	// Feat is the feat the background grants.
	Feat BackgroundFeatReference
	// SkillProficiencies lists exactly [BackgroundSkillProficiencies] distinct
	// skills.
	SkillProficiencies []Skill
	// ToolProficiency is the tool proficiency (required).
	ToolProficiency ToolProficiency
	// EquipmentOptions is the pair of equipment options.
	EquipmentOptions EquipmentOptions
}

// Background is a character background (SRD 5.2.1, "Character Backgrounds",
// p. 83). It lists the three abilities whose scores the background may raise,
// but not the increases themselves: those are a rule of SRD 5.2.1, "Ability
// Scores", p. 83, applied by the character context. It is an [Entry] of kind
// [KindBackground] and implements [References]: its feat must be an Origin
// feat. Immutable, with its collections copied on construction and on read;
// the zero Background is invalid. It holds slices, so it is not comparable
// with ==; compare with [Background.Equal] or [Set.Equal]. Safe for concurrent use.
type Background struct {
	id                 ID
	name               string
	source             SourceCitation
	abilityScores      []ability.Ability
	feat               BackgroundFeatReference
	skillProficiencies []Skill
	toolProficiency    ToolProficiency
	equipmentOptions   EquipmentOptions
}

var (
	_ Entry      = Background{}
	_ References = Background{}
)

// NewBackground returns the background id called name, with the members of f.
// A zero source means the entry cites none. Whether the feat is an Origin feat
// needs the feat itself, which a constructor cannot see: [ValidateSet] checks
// it.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError] for the first rule
// broken, in the order of the members: BlockInvalidName; for the abilities
// BlockOutOfRange (not [BackgroundAbilityScores]), BlockInvalidMember and
// BlockDuplicate; the *[IDError] of [ParseFeatID] for a zero feat reference;
// for the skills the same three; BlockMissing for no tool proficiency and
// BlockInvalidMember for one that is not valid. It never panics.
func NewBackground(id BackgroundID, name string, source SourceCitation, f BackgroundFields) (Background, error) {
	const block = "Background"
	pid, err := ParseID(string(id))
	if err != nil {
		return Background{}, err
	}
	if err := checkName(block, name); err != nil {
		return Background{}, err
	}
	if err := checkSet(block, f.AbilityScores, BackgroundAbilityScores, BackgroundAbilityScores, ability.Ability.Valid); err != nil {
		return Background{}, err
	}
	if _, err := ParseFeatID(string(f.Feat.featID)); err != nil {
		return Background{}, err
	}
	if err := checkSet(block, f.SkillProficiencies, BackgroundSkillProficiencies, BackgroundSkillProficiencies, Skill.Valid); err != nil {
		return Background{}, err
	}
	switch {
	case f.ToolProficiency == nil:
		return Background{}, blockError(block, BlockMissing)
	case !validToolProficiency(f.ToolProficiency):
		return Background{}, blockError(block, BlockInvalidMember)
	}
	return Background{
		id: pid, name: name, source: source, abilityScores: slices.Clone(f.AbilityScores), feat: f.Feat,
		skillProficiencies: slices.Clone(f.SkillProficiencies), toolProficiency: f.ToolProficiency, equipmentOptions: f.EquipmentOptions,
	}, nil
}

// MustBackground is like [NewBackground] but panics with its error when the
// arguments are invalid.
func MustBackground(id BackgroundID, name string, source SourceCitation, f BackgroundFields) Background {
	b, err := NewBackground(id, name, source, f)
	if err != nil {
		panic(err)
	}
	return b
}

// validToolProficiency reports whether t is a tool proficiency that its
// constructor would accept.
func validToolProficiency(t ToolProficiency) bool {
	switch v := t.(type) {
	case SpecificToolProficiency:
		return validName(v.name)
	case ChoiceToolProficiency:
		return validText(v.category)
	}
	return false // nil or a pointer to a variant: not a member of the closed set
}

// Key returns the key of the background: kind [KindBackground] and its id.
func (b Background) Key() Key { return NewKey(KindBackground, b.id) }

// ID returns the background id.
func (b Background) ID() BackgroundID { return BackgroundID(b.id.String()) }

// Name returns the background name.
func (b Background) Name() string { return b.name }

// Source returns the citation of the entry, or the zero [SourceCitation] when
// it cites none.
func (b Background) Source() SourceCitation { return b.source }

// AbilityScores returns the three abilities in printed order, as a new slice.
func (b Background) AbilityScores() []ability.Ability { return slices.Clone(b.abilityScores) }

// Feat returns the feat the background grants.
func (b Background) Feat() BackgroundFeatReference { return b.feat }

// SkillProficiencies returns the two skills, as a new slice.
func (b Background) SkillProficiencies() []Skill { return slices.Clone(b.skillProficiencies) }

// ToolProficiency returns the tool proficiency: a [SpecificToolProficiency] or
// a [ChoiceToolProficiency].
func (b Background) ToolProficiency() ToolProficiency { return b.toolProficiency }

// EquipmentOptions returns the pair of equipment options.
func (b Background) EquipmentOptions() EquipmentOptions { return b.equipmentOptions }

// References returns the one site of a background (FR-025): the field
// "feat/featId", naming a feat of category [FeatOrigin].
func (b Background) References() []Reference {
	return []Reference{MustReference("feat/featId", b.feat.featID.Key(), TargetHasFeatCategory{Category: FeatOrigin})}
}

// MustEquipmentItem is like [NewEquipmentItem] but panics with its error when
// the arguments are invalid.
func MustEquipmentItem(name string, quantity int) EquipmentItem {
	i, err := NewEquipmentItem(name, quantity)
	if err != nil {
		panic(err)
	}
	return i
}

// MustEquipmentOption is like [NewEquipmentOption] but panics with its error
// when the arguments are invalid.
func MustEquipmentOption(items []EquipmentItem, gold int) EquipmentOption {
	o, err := NewEquipmentOption(items, gold)
	if err != nil {
		panic(err)
	}
	return o
}

// MustBackgroundFeatReference is like [NewBackgroundFeatReference] but panics
// with its error when the arguments are invalid.
func MustBackgroundFeatReference(featID FeatID, choice string) BackgroundFeatReference {
	r, err := NewBackgroundFeatReference(featID, choice)
	if err != nil {
		panic(err)
	}
	return r
}
