package content

// ArmorCategory is Light, Medium, Heavy or Shield (SRD 5.2.1, "Armor", p.
// 92). The zero value is invalid. Comparable and safe for concurrent use.
type ArmorCategory int

// The armor categories.
const (
	ArmorLight ArmorCategory = iota + 1
	ArmorMedium
	ArmorHeavy
	ArmorShield
)

func armorCategoryNames() []string { return []string{"Light", "Medium", "Heavy", "Shield"} }

// Valid reports whether c is an armor category.
func (c ArmorCategory) Valid() bool { return enumValid(armorCategoryNames(), int(c)) }

// String returns the printed name of c, or "unknown".
func (c ArmorCategory) String() string { return enumString(armorCategoryNames(), int(c)) }

func parseArmorCategory(s string) (ArmorCategory, error) {
	v, err := parseEnum("ArmorCategory", armorCategoryNames(), s)
	return ArmorCategory(v), err
}

// DonDoffTime is the time to don or doff armor: a closed set with the
// variants [DonDoffMinutes] and [DonDoffByUtilize].
type DonDoffTime interface{ isDonDoffTime() }

// DonDoffMinutes is a time of a whole number of Minutes. The zero value is
// invalid; build it with [NewDonDoffMinutes]. Comparable.
type DonDoffMinutes struct{ minutes int }

func (DonDoffMinutes) isDonDoffTime() {}

// NewDonDoffMinutes returns a time of minutes Minutes. It returns a
// *[BlockError] (BlockOutOfRange) wrapping srderr.ErrInvalidInput for a
// number below 1.
func NewDonDoffMinutes(minutes int) (DonDoffMinutes, error) {
	if minutes < 1 || minutes > equipmentMaxInt {
		return DonDoffMinutes{}, blockError("DonDoffMinutes", BlockOutOfRange)
	}
	return DonDoffMinutes{minutes: minutes}, nil
}

// Minutes returns the number of Minutes.
func (t DonDoffMinutes) Minutes() int { return t.minutes }

// DonDoffByUtilize is the time of a Shield, which takes a Utilize action to
// don or doff. Its zero value is the one valid value.
type DonDoffByUtilize struct{}

func (DonDoffByUtilize) isDonDoffTime() {}

// DonTime returns the time to don armor of category c: 1 Minute for Light, 5
// for Medium, 10 for Heavy and a Utilize action for a Shield (SRD 5.2.1,
// "Armor", p. 92). It returns nil for an invalid category.
func (c ArmorCategory) DonTime() DonDoffTime {
	switch c {
	case ArmorLight:
		return DonDoffMinutes{minutes: 1}
	case ArmorMedium:
		return DonDoffMinutes{minutes: 5}
	case ArmorHeavy:
		return DonDoffMinutes{minutes: 10}
	case ArmorShield:
		return DonDoffByUtilize{}
	}
	return nil
}

// DoffTime returns the time to doff armor of category c: 1 Minute for Light
// and Medium, 5 for Heavy and a Utilize action for a Shield. It returns nil
// for an invalid category.
func (c ArmorCategory) DoffTime() DonDoffTime {
	switch c {
	case ArmorLight, ArmorMedium:
		return DonDoffMinutes{minutes: 1}
	case ArmorHeavy:
		return DonDoffMinutes{minutes: 5}
	case ArmorShield:
		return DonDoffByUtilize{}
	}
	return nil
}

// DexterityBonus is how much of the Dexterity modifier body armor adds to
// Armor Class: a closed set with the variants [FullDexterity],
// [CappedDexterity] and [NoDexterity].
type DexterityBonus interface{ isDexterityBonus() }

// FullDexterity adds the whole Dexterity modifier. Its zero value is the one
// valid value.
type FullDexterity struct{}

func (FullDexterity) isDexterityBonus() {}

// CappedDexterity adds the Dexterity modifier up to a maximum. The zero value
// is a maximum of 0, which is valid; build others with [NewCappedDexterity].
// Comparable.
type CappedDexterity struct{ maximum int }

func (CappedDexterity) isDexterityBonus() {}

// NewCappedDexterity returns a cap of maximum. It returns a *[BlockError]
// (BlockOutOfRange) wrapping srderr.ErrInvalidInput for a maximum below 0.
func NewCappedDexterity(maximum int) (CappedDexterity, error) {
	if maximum < 0 || maximum > equipmentMaxInt {
		return CappedDexterity{}, blockError("CappedDexterity", BlockOutOfRange)
	}
	return CappedDexterity{maximum: maximum}, nil
}

// Maximum returns the largest Dexterity modifier added.
func (c CappedDexterity) Maximum() int { return c.maximum }

// NoDexterity adds none of the Dexterity modifier. Its zero value is the one
// valid value.
type NoDexterity struct{}

func (NoDexterity) isDexterityBonus() {}

func validDexterity(d DexterityBonus) bool {
	switch v := d.(type) {
	case FullDexterity, NoDexterity:
		return true
	case CappedDexterity:
		_, err := NewCappedDexterity(v.maximum)
		return err == nil
	}
	return false
}

// Armor Class bounds of the data, package design sanity bounds (the SRD prints
// no limit).
const (
	armorBaseMax  = 30
	armorBonusMax = 10
)

// ArmorClassRule is how armor sets Armor Class: a closed set with the variants
// [BaseRule] (body armor) and [BonusRule] (a Shield). It is data with no
// evaluation method.
type ArmorClassRule interface{ isArmorClassRule() }

// BaseRule is a base Armor Class and the Dexterity treatment. The zero value
// is invalid; build it with [NewBaseRule]. Comparable.
type BaseRule struct {
	base      int
	dexterity DexterityBonus
}

func (BaseRule) isArmorClassRule() {}

// NewBaseRule returns a rule with base Armor Class base (1 to 30) and
// Dexterity treatment dex. It returns a *[BlockError] wrapping
// srderr.ErrInvalidInput: BlockOutOfRange for a base outside the range, then
// BlockInvalidMember for an invalid treatment.
func NewBaseRule(base int, dex DexterityBonus) (BaseRule, error) {
	switch {
	case base < 1 || base > armorBaseMax:
		return BaseRule{}, blockError("BaseRule", BlockOutOfRange)
	case !validDexterity(dex):
		return BaseRule{}, blockError("BaseRule", BlockInvalidMember)
	}
	return BaseRule{base: base, dexterity: dex}, nil
}

// Base returns the base Armor Class.
func (r BaseRule) Base() int { return r.base }

// Dexterity returns how the Dexterity modifier is added.
func (r BaseRule) Dexterity() DexterityBonus { return r.dexterity }

// BonusRule is a bonus to Armor Class, the rule of a Shield. The zero value
// is invalid; build it with [NewBonusRule]. Comparable.
type BonusRule struct{ bonus int }

func (BonusRule) isArmorClassRule() {}

// NewBonusRule returns a bonus of 1 to 10. It returns a *[BlockError]
// (BlockOutOfRange) wrapping srderr.ErrInvalidInput outside that range.
func NewBonusRule(bonus int) (BonusRule, error) {
	if bonus < 1 || bonus > armorBonusMax {
		return BonusRule{}, blockError("BonusRule", BlockOutOfRange)
	}
	return BonusRule{bonus: bonus}, nil
}

// Bonus returns the Armor Class bonus.
func (r BonusRule) Bonus() int { return r.bonus }

// Strength scores a Strength requirement may name, the range of ability
// scores.
const (
	armorMinStrength = 1
	armorMaxStrength = 30
)

// Armor is a row of the Armor table (SRD 5.2.1, "Armor", p. 92); a Shield is
// an Armor of category [ArmorShield] with a [BonusRule]. It is an
// [EquipmentEntry]. Immutable; the zero Armor is invalid. Comparable. Safe for
// concurrent use.
type Armor struct {
	equipmentHeader
	category ArmorCategory
	rule     ArmorClassRule
	strength int // 0 when the table prints "—"
	stealth  bool
	weight   Weight
	price    FixedCost
}

var _ EquipmentEntry = Armor{}

// NewArmor returns the armor id called name. The strength requirement is 1 to
// 30, or 0 for an armor printed without one.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockInvalidMember for an invalid category, rule, weight or price;
// BlockConstraint for a Shield without a [BonusRule] or body armor without a
// [BaseRule]; BlockOutOfRange for a strength requirement outside 1 to 30. It
// never panics.
func NewArmor(id ItemID, name string, source SourceCitation, category ArmorCategory, rule ArmorClassRule, strengthRequirement int, stealthDisadvantage bool, weight Weight, price FixedCost) (Armor, error) {
	const block = "Armor"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return Armor{}, err
	}
	if !category.Valid() || !validArmorRule(rule) || !validWeightSpec(weight) || !price.cost.Valid() {
		return Armor{}, blockError(block, BlockInvalidMember)
	}
	if _, bonus := rule.(BonusRule); bonus != (category == ArmorShield) {
		return Armor{}, blockError(block, BlockConstraint)
	}
	if strengthRequirement != 0 && (strengthRequirement < armorMinStrength || strengthRequirement > armorMaxStrength) {
		return Armor{}, blockError(block, BlockOutOfRange)
	}
	return Armor{
		equipmentHeader: h, category: category, rule: rule, strength: strengthRequirement,
		stealth: stealthDisadvantage, weight: weight, price: price,
	}, nil
}

func validArmorRule(r ArmorClassRule) bool {
	var err error
	switch v := r.(type) {
	case BaseRule:
		_, err = NewBaseRule(v.base, v.dexterity)
	case BonusRule:
		_, err = NewBonusRule(v.bonus)
	default:
		return false
	}
	return err == nil
}

// EquipmentCategory returns [EquipmentArmor].
func (Armor) EquipmentCategory() EquipmentCategory { return EquipmentArmor }

// Category returns Light, Medium, Heavy or Shield.
func (a Armor) Category() ArmorCategory { return a.category }

// ArmorClass returns the Armor Class rule.
func (a Armor) ArmorClass() ArmorClassRule { return a.rule }

// StrengthRequirement returns the Strength score the armor needs and true, or
// 0 and false when the table prints "—".
func (a Armor) StrengthRequirement() (int, bool) { return a.strength, a.strength != 0 }

// StealthDisadvantage reports whether the armor gives Disadvantage on Dexterity
// (Stealth) checks.
func (a Armor) StealthDisadvantage() bool { return a.stealth }

// Weight returns the weight.
func (a Armor) Weight() Weight { return a.weight }

// Price returns the price.
func (a Armor) Price() FixedCost { return a.price }
