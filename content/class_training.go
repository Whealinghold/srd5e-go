package content

import "slices"

// WeaponTrainingCategory is the category of weapon a class is trained in:
// Simple or Martial (SRD 5.2.1, "Weapons", p. 89). The wire spelling is the
// printed name. The zero value is invalid. Comparable and safe for
// concurrent use.
type WeaponTrainingCategory int

// The weapon training categories.
const (
	WeaponTrainingSimple WeaponTrainingCategory = iota + 1
	WeaponTrainingMartial
)

func weaponTrainingNames() []string { return []string{"Simple", "Martial"} }

// Valid reports whether c is Simple or Martial.
func (c WeaponTrainingCategory) Valid() bool { return enumValid(weaponTrainingNames(), int(c)) }

// String returns the printed name of c, or "unknown" for an undefined value.
func (c WeaponTrainingCategory) String() string { return enumString(weaponTrainingNames(), int(c)) }

// ParseWeaponTrainingCategory returns the category spelled exactly s
// ("Simple" or "Martial"); matching is case-sensitive.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseWeaponTrainingCategory(s string) (WeaponTrainingCategory, error) {
	v, err := parseEnum("WeaponTrainingCategory", weaponTrainingNames(), s)
	return WeaponTrainingCategory(v), err
}

// ArmorTrainingCategory is a category of armor training of a class: Light,
// Medium, Heavy or Shield (SRD 5.2.1, "Core Barbarian Traits", p. 28). The
// wire spelling is the printed name. The zero value is invalid. Comparable
// and safe for concurrent use.
type ArmorTrainingCategory int

// The armor training categories.
const (
	ArmorTrainingLight ArmorTrainingCategory = iota + 1
	ArmorTrainingMedium
	ArmorTrainingHeavy
	ArmorTrainingShield
)

func armorTrainingNames() []string { return []string{"Light", "Medium", "Heavy", "Shield"} }

// Valid reports whether c is one of the four categories.
func (c ArmorTrainingCategory) Valid() bool { return enumValid(armorTrainingNames(), int(c)) }

// String returns the printed name of c, or "unknown" for an undefined value.
func (c ArmorTrainingCategory) String() string { return enumString(armorTrainingNames(), int(c)) }

// ParseArmorTrainingCategory returns the category spelled exactly s, such as
// "Light"; matching is case-sensitive.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseArmorTrainingCategory(s string) (ArmorTrainingCategory, error) {
	v, err := parseEnum("ArmorTrainingCategory", armorTrainingNames(), s)
	return ArmorTrainingCategory(v), err
}

// SkillChoice is the skill proficiencies a class lets a character pick: a
// closed set with two variants, [SkillChoiceListed] ("Choose 2: ...") and
// [SkillChoiceAny] ("Choose any 3 skills"). Consumers switch over it
// exhaustively. The variants are immutable and safe for concurrent use.
type SkillChoice interface {
	// Picks returns how many skills the character chooses, at least 1.
	Picks() int
	isSkillChoice()
}

// SkillChoiceListed is a choice of Picks skills from a listed set. The zero
// value is invalid; build it with [NewSkillChoiceListed]. It holds a slice,
// so it is not comparable with ==.
type SkillChoiceListed struct {
	picks  int
	skills []Skill
}

// NewSkillChoiceListed returns a choice of picks skills from skills, which
// are copied and kept in order.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for picks below 1, for no skill or for fewer skills
// than picks, BlockInvalidMember for an invalid skill and BlockDuplicate for a
// repeated one. It never panics.
func NewSkillChoiceListed(picks int, skills []Skill) (SkillChoiceListed, error) {
	const block = "SkillChoiceListed"
	if picks < 1 { // a larger count fails the length check below, since a set holds at most 18 skills
		return SkillChoiceListed{}, blockError(block, BlockOutOfRange)
	}
	if err := checkSet(block, skills, picks, noUpperBound, Skill.Valid); err != nil {
		return SkillChoiceListed{}, err
	}
	return SkillChoiceListed{picks: picks, skills: slices.Clone(skills)}, nil
}

// Picks returns how many skills the character chooses.
func (c SkillChoiceListed) Picks() int { return c.picks }

// Skills returns the listed skills in printed order, as a new slice.
func (c SkillChoiceListed) Skills() []Skill { return slices.Clone(c.skills) }

func (SkillChoiceListed) isSkillChoice() {}

// SkillChoiceAny is a choice of Picks skills from all skills. The zero value
// is invalid; build it with [NewSkillChoiceAny]. Comparable.
type SkillChoiceAny struct{ picks int }

// NewSkillChoiceAny returns a choice of picks skills from any skill.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for picks outside 1 to [ClassMaxCount]. It never panics.
func NewSkillChoiceAny(picks int) (SkillChoiceAny, error) {
	if picks < 1 || picks > ClassMaxCount {
		return SkillChoiceAny{}, blockError("SkillChoiceAny", BlockOutOfRange)
	}
	return SkillChoiceAny{picks: picks}, nil
}

// Picks returns how many skills the character chooses.
func (c SkillChoiceAny) Picks() int { return c.picks }

func (SkillChoiceAny) isSkillChoice() {}

// validSkillChoice reports whether c is a variant built by its constructor.
func validSkillChoice(c SkillChoice) bool {
	switch v := c.(type) {
	case SkillChoiceListed:
		_, err := NewSkillChoiceListed(v.picks, v.skills)
		return err == nil
	case SkillChoiceAny:
		_, err := NewSkillChoiceAny(v.picks)
		return err == nil
	}
	return false // nil or a pointer to a variant: not a member of the closed set
}

// WeaponProficiency is a weapon training of a class: every weapon of a
// category, or only those with at least one of some properties ("Martial
// weapons that have the Light property", SRD 5.2.1, "Core Monk Traits", p.
// 49). The zero value is invalid; build it with [NewWeaponProficiency]. It
// holds a slice, so it is not comparable with ==.
type WeaponProficiency struct {
	category WeaponTrainingCategory
	anyOf    []WeaponProperty
}

// NewWeaponProficiency returns the training in category, restricted to
// weapons that have at least one of requiresAnyProperty. A nil list means
// every weapon of the category; the properties are copied and kept in order.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for a category that is not a member, then for the
// properties BlockOutOfRange (a non-nil empty list), BlockInvalidMember and
// BlockDuplicate. It never panics.
func NewWeaponProficiency(category WeaponTrainingCategory, requiresAnyProperty []WeaponProperty) (WeaponProficiency, error) {
	const block = "WeaponProficiency"
	if !category.Valid() {
		return WeaponProficiency{}, blockError(block, BlockInvalidMember)
	}
	if requiresAnyProperty != nil {
		if err := checkSet(block, requiresAnyProperty, 1, noUpperBound, WeaponProperty.Valid); err != nil {
			return WeaponProficiency{}, err
		}
	}
	return WeaponProficiency{category: category, anyOf: slices.Clone(requiresAnyProperty)}, nil
}

// Category returns the weapon category.
func (w WeaponProficiency) Category() WeaponTrainingCategory { return w.category }

// RequiresAnyProperty returns the properties a weapon needs at least one of,
// as a new slice, or nil when the training covers every weapon of the
// category.
func (w WeaponProficiency) RequiresAnyProperty() []WeaponProperty { return slices.Clone(w.anyOf) }

func (w WeaponProficiency) valid() bool {
	_, err := NewWeaponProficiency(w.category, w.anyOf)
	return err == nil
}

// checkWeaponProficiencies checks a list of valid weapon trainings with at
// most one element per category: BlockInvalidMember, BlockDuplicate.
func checkWeaponProficiencies(block string, items []WeaponProficiency) error {
	for i, w := range items {
		if !w.valid() {
			return blockError(block, BlockInvalidMember)
		}
		if slices.ContainsFunc(items[:i], func(o WeaponProficiency) bool { return o.category == w.category }) {
			return blockError(block, BlockDuplicate)
		}
	}
	return nil
}

// ClassToolProficiency is a tool proficiency of a class: a closed set with two
// variants, [ClassToolProficiencySpecific] (a named tool) and
// [ClassToolProficiencyChoice] ("Choose 3 Musical Instruments"). It is the
// class-side type, distinct from [ToolProficiency]. Consumers switch over it
// exhaustively. The variants are immutable and safe for concurrent use.
type ClassToolProficiency interface{ isClassToolProficiency() }

// ClassToolProficiencySpecific is proficiency with one named tool. The zero
// value is invalid; build it with [NewClassToolProficiencySpecific].
// Comparable.
type ClassToolProficiencySpecific struct{ name string }

// NewClassToolProficiencySpecific returns proficiency with the tool called
// name.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidName when name breaks the name rule (schema FR-027). It never
// panics.
func NewClassToolProficiencySpecific(name string) (ClassToolProficiencySpecific, error) {
	if err := checkName("ClassToolProficiencySpecific", name); err != nil {
		return ClassToolProficiencySpecific{}, err
	}
	return ClassToolProficiencySpecific{name: name}, nil
}

// Name returns the tool name.
func (c ClassToolProficiencySpecific) Name() string { return c.name }

func (ClassToolProficiencySpecific) isClassToolProficiency() {}

// ClassToolProficiencyChoice is proficiency with Picks tools chosen from some
// categories. The zero value is invalid; build it with
// [NewClassToolProficiencyChoice]. It holds a slice, so it is not comparable
// with ==.
type ClassToolProficiencyChoice struct {
	categories []string
	picks      int
}

// NewClassToolProficiencyChoice returns a choice of picks tools from
// categories, which are copied and kept in order.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockEmpty for
// no category, BlockInvalidMember for a category that is not a non-blank
// single line, BlockDuplicate for a repeated one and BlockOutOfRange for picks
// outside 1 to [ClassMaxCount]. It never panics.
func NewClassToolProficiencyChoice(categories []string, picks int) (ClassToolProficiencyChoice, error) {
	const block = "ClassToolProficiencyChoice"
	if len(categories) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return ClassToolProficiencyChoice{}, blockError(block, BlockEmpty)
	}
	if err := checkSet(block, categories, 1, noUpperBound, validLine); err != nil {
		return ClassToolProficiencyChoice{}, err
	}
	if picks < 1 || picks > ClassMaxCount {
		return ClassToolProficiencyChoice{}, blockError(block, BlockOutOfRange)
	}
	return ClassToolProficiencyChoice{categories: slices.Clone(categories), picks: picks}, nil
}

// Categories returns the tool categories in printed order, as a new slice.
func (c ClassToolProficiencyChoice) Categories() []string { return slices.Clone(c.categories) }

// Picks returns how many tools the character chooses.
func (c ClassToolProficiencyChoice) Picks() int { return c.picks }

func (ClassToolProficiencyChoice) isClassToolProficiency() {}

func validClassToolProficiency(p ClassToolProficiency) bool {
	switch v := p.(type) {
	case ClassToolProficiencySpecific:
		_, err := NewClassToolProficiencySpecific(v.name)
		return err == nil
	case ClassToolProficiencyChoice:
		_, err := NewClassToolProficiencyChoice(v.categories, v.picks)
		return err == nil
	}
	return false
}

// ClassEquipmentItem is one line of a starting equipment option: a name as
// printed, in the singular, and a count (SRD 5.2.1, "Core Fighter Traits", p.
// 47). Names are not resolved to equipment entries. The zero value is
// invalid. Comparable.
type ClassEquipmentItem struct {
	name     string
	quantity int
}

// NewClassEquipmentItem returns the item called name, quantity of it.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidName for a name that breaks the name rule and BlockOutOfRange
// for a quantity outside 1 to [ClassMaxCount]. It never panics.
func NewClassEquipmentItem(name string, quantity int) (ClassEquipmentItem, error) {
	const block = "ClassEquipmentItem"
	if err := checkName(block, name); err != nil {
		return ClassEquipmentItem{}, err
	}
	if quantity < 1 || quantity > ClassMaxCount {
		return ClassEquipmentItem{}, blockError(block, BlockOutOfRange)
	}
	return ClassEquipmentItem{name: name, quantity: quantity}, nil
}

// Name returns the item name.
func (i ClassEquipmentItem) Name() string { return i.name }

// Quantity returns how many of the item the option holds.
func (i ClassEquipmentItem) Quantity() int { return i.quantity }

func (i ClassEquipmentItem) valid() bool {
	_, err := NewClassEquipmentItem(i.name, i.quantity)
	return err == nil
}

// ClassEquipmentOption is one lettered option of a class's starting
// equipment: items (possibly none) and gold pieces. It holds a slice, so it
// is not comparable with ==.
type ClassEquipmentOption struct {
	items []ClassEquipmentItem
	gold  int
}

// NewClassEquipmentOption returns an option of items, copied and kept in
// order, and gold pieces.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for an item that is not valid and BlockOutOfRange for
// gold outside 0 to [ClassMaxCount]. It never panics.
func NewClassEquipmentOption(items []ClassEquipmentItem, gold int) (ClassEquipmentOption, error) {
	const block = "ClassEquipmentOption"
	if slices.ContainsFunc(items, func(i ClassEquipmentItem) bool { return !i.valid() }) {
		return ClassEquipmentOption{}, blockError(block, BlockInvalidMember)
	}
	if gold < 0 || gold > ClassMaxCount {
		return ClassEquipmentOption{}, blockError(block, BlockOutOfRange)
	}
	return ClassEquipmentOption{items: cloneOrNil(items), gold: gold}, nil
}

// Items returns the items in printed order, as a new slice.
func (o ClassEquipmentOption) Items() []ClassEquipmentItem { return slices.Clone(o.items) }

// Gold returns the gold pieces of the option.
func (o ClassEquipmentOption) Gold() int { return o.gold }

// ClassStartingEquipment is the starting equipment of a class: options A and
// B, and for the Fighter a third, C. It holds slices, so it is not comparable
// with ==.
type ClassStartingEquipment struct {
	a, b, c ClassEquipmentOption
	hasC    bool
}

// NewClassStartingEquipment returns the equipment of options a and b, and of
// c as well when one is given.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for more than one c. It never panics.
func NewClassStartingEquipment(a, b ClassEquipmentOption, c ...ClassEquipmentOption) (ClassStartingEquipment, error) {
	const block = "ClassStartingEquipment"
	if len(c) > 1 {
		return ClassStartingEquipment{}, blockError(block, BlockOutOfRange)
	}
	e := ClassStartingEquipment{a: a, b: b}
	if len(c) == 1 { // equivalent mutant: more than one c was rejected above, so == and >= agree
		e.c, e.hasC = c[0], true
	}
	return e, nil
}

// A returns option A.
func (e ClassStartingEquipment) A() ClassEquipmentOption { return e.a }

// B returns option B.
func (e ClassStartingEquipment) B() ClassEquipmentOption { return e.b }

// C returns option C and true, or the zero option and false when the class
// has only A and B.
func (e ClassStartingEquipment) C() (ClassEquipmentOption, bool) { return e.c, e.hasC }

// MulticlassGrants is what a first level in a class grants beyond its level 1
// features (SRD 5.2.1, "As a Multiclass Character", p. 28 and the like). Its
// subset rule against the class's core traits is checked by
// [NewClassDefinition]. It holds slices, so it is not comparable with ==.
type MulticlassGrants struct {
	weapons []WeaponProficiency
	armor   []ArmorTrainingCategory
	tools   []ClassToolProficiency
	skills  SkillChoice
}

// NewMulticlassGrants returns the grants of the given parts, copied and kept
// in order. skills is nil when the class grants no skill; any list may be
// empty.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember or BlockDuplicate for the weapon trainings, BlockOutOfRange,
// BlockInvalidMember or BlockDuplicate for the armor categories,
// BlockInvalidMember for a tool proficiency or a skill choice that is not
// valid. It never panics.
func NewMulticlassGrants(weapons []WeaponProficiency, armor []ArmorTrainingCategory, tools []ClassToolProficiency, skills SkillChoice) (MulticlassGrants, error) {
	const block = "MulticlassGrants"
	if err := checkWeaponProficiencies(block, weapons); err != nil {
		return MulticlassGrants{}, err
	}
	if err := checkSet(block, armor, 0, len(armorTrainingNames()), ArmorTrainingCategory.Valid); err != nil {
		return MulticlassGrants{}, err
	}
	if slices.ContainsFunc(tools, func(p ClassToolProficiency) bool { return !validClassToolProficiency(p) }) {
		return MulticlassGrants{}, blockError(block, BlockInvalidMember)
	}
	if skills != nil && !validSkillChoice(skills) {
		return MulticlassGrants{}, blockError(block, BlockInvalidMember)
	}
	return MulticlassGrants{weapons: cloneOrNil(weapons), armor: cloneOrNil(armor), tools: cloneOrNil(tools), skills: skills}, nil
}

// WeaponProficiencies returns the weapon trainings, as a new slice.
func (g MulticlassGrants) WeaponProficiencies() []WeaponProficiency { return slices.Clone(g.weapons) }

// ArmorTraining returns the armor categories, as a new slice.
func (g MulticlassGrants) ArmorTraining() []ArmorTrainingCategory { return slices.Clone(g.armor) }

// ToolProficiencies returns the tool proficiencies, as a new slice.
func (g MulticlassGrants) ToolProficiencies() []ClassToolProficiency { return slices.Clone(g.tools) }

// SkillChoice returns the skill choice, or nil when the class grants no
// skill.
func (g MulticlassGrants) SkillChoice() SkillChoice { return g.skills }
