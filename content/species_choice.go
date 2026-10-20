package content

import "slices"

// The wire names of the three ChoiceOption variants (the "kind" of an option
// object, SX-1).
const (
	optionKindDamageType = "damageType"
	optionKindLineage    = "lineage"
	optionKindText       = "text"
)

// ChoiceOption is the closed set of options of a [SpeciesChoice] (SRD 5.2.1,
// "Dragonborn" p. 84, "Elf" p. 85, "Gnome" p. 85, "Goliath" pp. 85-86,
// "Tiefling" p. 86). Variants: DamageTypeOption, LineageOption, TextOption.
// Consumers switch over it exhaustively; the variants are immutable and safe
// for concurrent use, and copy their slices on construction and on read.
type ChoiceOption interface {
	// Name returns the printed name of the option.
	Name() string
	valid() bool
	isChoiceOption()
}

// optionKind returns the wire name of the variant of o, or "" when o is nil or
// not one of the three value variants.
func optionKind(o ChoiceOption) string {
	switch o.(type) {
	case DamageTypeOption:
		return optionKindDamageType
	case LineageOption:
		return optionKindLineage
	case TextOption:
		return optionKindText
	}
	return "" // nil or a pointer variant: not a member of the closed set (equivalent mutant: "" to "" is a no-op)
}

// DamageTypeOption is an option that is a damage type, such as a Draconic
// Ancestry. The zero value is invalid; build it with [NewDamageTypeOption].
// Comparable.
type DamageTypeOption struct {
	name       string
	damageType DamageType
}

// NewDamageTypeOption returns an option called name for damageType.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidName for a name that breaks the name rule (schema FR-027) and
// BlockInvalidMember for a damage type that is not a member, checked in that
// order. It never panics.
func NewDamageTypeOption(name string, damageType DamageType) (DamageTypeOption, error) {
	const block = "DamageTypeOption"
	if err := checkName(block, name); err != nil {
		return DamageTypeOption{}, err
	}
	if !damageType.Valid() {
		return DamageTypeOption{}, blockError(block, BlockInvalidMember)
	}
	return DamageTypeOption{name: name, damageType: damageType}, nil
}

// Name returns the option name.
func (o DamageTypeOption) Name() string { return o.name }

// DamageType returns the damage type.
func (o DamageTypeOption) DamageType() DamageType { return o.damageType }

func (o DamageTypeOption) valid() bool { return validName(o.name) && o.damageType.Valid() }

func (DamageTypeOption) isChoiceOption() {}

// lineageLevels returns the three levels of a lineage option, in order.
func lineageLevels() []int { return []int{1, 3, 5} }

// validLineageLevel reports whether level is one of the three lineage levels.
func validLineageLevel(level int) bool { return slices.Contains(lineageLevels(), level) }

// lineageLevelProblem applies the order rule of FR-061 to the levels of a
// lineage option, each already a valid level: a repeated level is
// BlockDuplicate at the index of the later occurrence, and distinct levels in
// another order are BlockConstraint (index -1, the whole list). It returns 0
// for the levels 1, 3, 5.
func lineageLevelProblem(levels []int) (BlockProblem, int) {
	for i, l := range levels {
		if slices.Contains(levels[:i], l) {
			return BlockDuplicate, i
		}
	}
	if !slices.Equal(levels, lineageLevels()) {
		return BlockConstraint, -1 // equivalent mutant: both callers discard the index, so -1 to 0 is unobservable
	}
	return 0, -1 // equivalent mutant: replaces the literal 0 with 0
}

// LevelBenefit is the printed benefit of a lineage option at one level: the
// verbatim text of the table cell and the spell it names (SRD 5.2.1, "Elven
// Lineages", p. 85; "Fiendish Legacies", p. 86). The zero value is invalid;
// build it with [NewLevelBenefit]. Comparable.
type LevelBenefit struct {
	level   int
	text    string
	spellID SpellID
}

// NewLevelBenefit returns the benefit at level, which must be 1, 3 or 5, with
// the verbatim text and the spell it names.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockInvalidMember
// for another level, then BlockBlankText or BlockInvalidCharacter for a text
// that breaks the text rule (schema FR-028); then the *[IDError] of
// [ParseSpellID] for a spell id that is empty or malformed. It never panics.
func NewLevelBenefit(level int, text string, spellID SpellID) (LevelBenefit, error) {
	if !validLineageLevel(level) {
		return LevelBenefit{}, blockError("LevelBenefit", BlockInvalidMember)
	}
	if err := checkText("LevelBenefit", text); err != nil {
		return LevelBenefit{}, err
	}
	if _, err := ParseSpellID(string(spellID)); err != nil {
		return LevelBenefit{}, err
	}
	return LevelBenefit{level: level, text: text, spellID: spellID}, nil
}

// Level returns the level, 1, 3 or 5.
func (b LevelBenefit) Level() int { return b.level }

// Text returns the verbatim text.
func (b LevelBenefit) Text() string { return b.text }

// SpellID returns the spell the benefit names.
func (b LevelBenefit) SpellID() SpellID { return b.spellID }

func (b LevelBenefit) valid() bool {
	return validLineageLevel(b.level) && validText(b.text) && b.spellID.Valid()
}

// LineageOption is an option with a benefit at each of the levels 1, 3 and 5,
// such as an Elven Lineage or a Fiendish Legacy. The zero value is invalid;
// build it with [NewLineageOption]. Comparable.
type LineageOption struct {
	name     string
	benefits [3]LevelBenefit
}

// NewLineageOption returns an option called name whose benefits are exactly
// three, for the levels 1, 3 and 5 in that order. The benefits are copied.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockInvalidName,
// then BlockOutOfRange for a number of benefits other than three,
// BlockInvalidMember for a benefit that is not valid, BlockDuplicate for a
// repeated level and BlockConstraint for distinct levels in another order. It
// never panics.
func NewLineageOption(name string, benefits []LevelBenefit) (LineageOption, error) {
	const block = "LineageOption"
	if err := checkName(block, name); err != nil {
		return LineageOption{}, err
	}
	if len(benefits) != len(lineageLevels()) {
		return LineageOption{}, blockError(block, BlockOutOfRange)
	}
	levels := make([]int, len(benefits))
	for i, b := range benefits {
		if !b.valid() {
			return LineageOption{}, blockError(block, BlockInvalidMember)
		}
		levels[i] = b.level
	}
	if p, _ := lineageLevelProblem(levels); p != 0 { // equivalent mutant: a problem is positive, so != and > agree
		return LineageOption{}, blockError(block, p)
	}
	return LineageOption{name: name, benefits: [3]LevelBenefit(benefits)}, nil
}

// Name returns the option name.
func (o LineageOption) Name() string { return o.name }

// LevelBenefits returns the three benefits, for the levels 1, 3 and 5, as a
// new slice.
func (o LineageOption) LevelBenefits() []LevelBenefit { return slices.Clone(o.benefits[:]) }

func (o LineageOption) valid() bool {
	levels := make([]int, len(o.benefits))
	for i, b := range o.benefits {
		if !b.valid() {
			return false
		}
		levels[i] = b.level
	}
	p, _ := lineageLevelProblem(levels)
	return validName(o.name) && p == 0 // equivalent mutant: a problem is never negative, so == and <= agree
}

func (LineageOption) isChoiceOption() {}

// TextOption is an option that is a name and its verbatim text, and the spells
// the text names, such as a Gnomish Lineage or a Giant Ancestry. The zero value
// is invalid; build it with [NewTextOption]. It holds a slice, so it is not
// comparable with ==; compare with [TextOption.Equal].
type TextOption struct {
	name     string
	text     string
	spellIDs []SpellID
}

// NewTextOption returns an option called name with the verbatim text and the
// spells the text names, which may be none. The spell ids are copied.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockInvalidName,
// then BlockBlankText or BlockInvalidCharacter for a text that breaks the text
// rule; then the *[IDError] of [ParseSpellID] for the first spell id that is
// empty or malformed. It never panics.
func NewTextOption(name, text string, spellIDs []SpellID) (TextOption, error) {
	if err := checkName("TextOption", name); err != nil {
		return TextOption{}, err
	}
	if err := checkText("TextOption", text); err != nil {
		return TextOption{}, err
	}
	for _, id := range spellIDs {
		if _, err := ParseSpellID(string(id)); err != nil {
			return TextOption{}, err
		}
	}
	return TextOption{name: name, text: text, spellIDs: cloneOrNil(spellIDs)}, nil
}

// Name returns the option name.
func (o TextOption) Name() string { return o.name }

// Text returns the verbatim text.
func (o TextOption) Text() string { return o.text }

// SpellIDs returns the spells the text names, possibly none, as a new slice.
func (o TextOption) SpellIDs() []SpellID { return slices.Clone(o.spellIDs) }

func (o TextOption) valid() bool {
	return validName(o.name) && validText(o.text) && !slices.ContainsFunc(o.spellIDs, func(id SpellID) bool { return !id.Valid() })
}

func (TextOption) isChoiceOption() {}

// SpeciesChoice is a trait that asks the player to choose from a closed list
// (SX-1): the name of the trait and its options, all of one variant. The zero
// value is invalid; build it with [NewSpeciesChoice]. It holds slices, so it is
// not comparable with ==; compare with [SpeciesChoice.Equal].
type SpeciesChoice struct {
	traitName string
	options   []ChoiceOption
}

// NewSpeciesChoice returns the choice of the trait called traitName among
// options, which must be at least one, with distinct names and all of one
// variant. The options are copied.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockInvalidName,
// BlockEmpty for no options, then for each option in order BlockInvalidMember
// (nil, a pointer to a variant, or not valid), BlockDuplicate (a name seen before) and BlockConstraint
// (a variant other than the first option's). It never panics.
func NewSpeciesChoice(traitName string, options []ChoiceOption) (SpeciesChoice, error) {
	const block = "SpeciesChoice"
	if err := checkName(block, traitName); err != nil {
		return SpeciesChoice{}, err
	}
	if len(options) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return SpeciesChoice{}, blockError(block, BlockEmpty)
	}
	names := make(map[string]struct{}, len(options))
	for _, o := range options {
		if optionKind(o) == "" || !o.valid() { // equivalent mutant: a string is never below "", so == and <= agree
			return SpeciesChoice{}, blockError(block, BlockInvalidMember)
		}
		if _, repeated := names[o.Name()]; repeated {
			return SpeciesChoice{}, blockError(block, BlockDuplicate)
		}
		names[o.Name()] = struct{}{}
		if optionKind(o) != optionKind(options[0]) {
			return SpeciesChoice{}, blockError(block, BlockConstraint)
		}
	}
	return SpeciesChoice{traitName: traitName, options: slices.Clone(options)}, nil
}

// TraitName returns the name of the trait the choice belongs to.
func (c SpeciesChoice) TraitName() string { return c.traitName }

// Options returns the options as a new slice.
func (c SpeciesChoice) Options() []ChoiceOption { return slices.Clone(c.options) }

func (c SpeciesChoice) valid() bool {
	_, err := NewSpeciesChoice(c.traitName, c.options)
	return err == nil
}

// MustDamageTypeOption is like [NewDamageTypeOption] but panics with its error
// when the arguments are invalid.
func MustDamageTypeOption(name string, damageType DamageType) DamageTypeOption {
	o, err := NewDamageTypeOption(name, damageType)
	if err != nil {
		panic(err)
	}
	return o
}

// MustLevelBenefit is like [NewLevelBenefit] but panics with its error when the
// arguments are invalid.
func MustLevelBenefit(level int, text string, spellID SpellID) LevelBenefit {
	b, err := NewLevelBenefit(level, text, spellID)
	if err != nil {
		panic(err)
	}
	return b
}

// MustLineageOption is like [NewLineageOption] but panics with its error when
// the arguments are invalid.
func MustLineageOption(name string, benefits []LevelBenefit) LineageOption {
	o, err := NewLineageOption(name, benefits)
	if err != nil {
		panic(err)
	}
	return o
}

// MustTextOption is like [NewTextOption] but panics with its error when the
// arguments are invalid.
func MustTextOption(name, text string, spellIDs []SpellID) TextOption {
	o, err := NewTextOption(name, text, spellIDs)
	if err != nil {
		panic(err)
	}
	return o
}

// MustSpeciesChoice is like [NewSpeciesChoice] but panics with its error when
// the arguments are invalid.
func MustSpeciesChoice(traitName string, options []ChoiceOption) SpeciesChoice {
	c, err := NewSpeciesChoice(traitName, options)
	if err != nil {
		panic(err)
	}
	return c
}
