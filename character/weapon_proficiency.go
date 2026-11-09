package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// WeaponProficiencyStatus says whether a character is proficient with a weapon
// (SRD 5.2.1, "Weapon Proficiency", p. 89). Not being proficient never rejects
// an attack: only the Proficiency Bonus is withheld. The zero value is not a
// status.
type WeaponProficiencyStatus int

const (
	// WeaponProficient: the Proficiency Bonus is added to the attack roll.
	WeaponProficient WeaponProficiencyStatus = iota + 1
	// WeaponNotProficient: the Proficiency Bonus is not added.
	WeaponNotProficient
)

// String returns "proficient", "notProficient" or "unknown".
func (s WeaponProficiencyStatus) String() string {
	switch s {
	case WeaponProficient:
		return "proficient"
	case WeaponNotProficient:
		return "notProficient"
	}
	return "unknown"
}

// WeaponProficiencyRule is one way to be proficient with a weapon: the closed
// set CategoryProficiency, CategoryWithAnyPropertyProficiency and
// AnyStatBlockWeaponProficiency.
//
//sumtype:decl
type WeaponProficiencyRule interface {
	isWeaponProficiencyRule()
	matches(w content.Weapon) bool
}

// CategoryProficiency is proficiency with every weapon of one category
// (Simple or Martial), and none of the other (SRD 5.2.1, "Weapons", p. 89).
type CategoryProficiency content.WeaponCategory

func (CategoryProficiency) isWeaponProficiencyRule() {}

func (c CategoryProficiency) matches(w content.Weapon) bool {
	return w.Category() == content.WeaponCategory(c)
}

// CategoryWithAnyPropertyProficiency is proficiency with the weapons of one
// category that have at least one of a set of properties (the Rogue's Martial
// weapons with Finesse or Light, SRD 5.2.1, "Core Rogue Traits", p. 61). It
// matches the property names only, ignoring their parameters. Build it with
// NewCategoryWithAnyPropertyProficiency; the zero value matches nothing.
type CategoryWithAnyPropertyProficiency struct {
	category   content.WeaponCategory
	properties []content.WeaponProperty
}

// NewCategoryWithAnyPropertyProficiency returns the rule for category and
// properties. It returns an error matching srderr.ErrInvalidInput, naming
// properties, for an empty set, and naming category for an invalid category.
func NewCategoryWithAnyPropertyProficiency(category content.WeaponCategory, properties []content.WeaponProperty) (CategoryWithAnyPropertyProficiency, error) {
	if !category.Valid() {
		return CategoryWithAnyPropertyProficiency{}, invalidInput("category of a property proficiency")
	}
	if len(properties) == 0 {
		return CategoryWithAnyPropertyProficiency{}, invalidInput("properties of a property proficiency: empty")
	}
	return CategoryWithAnyPropertyProficiency{category: category, properties: slices.Clone(properties)}, nil
}

// MustCategoryWithAnyPropertyProficiency is NewCategoryWithAnyPropertyProficiency
// for values known to be valid. It panics with the error that function returns,
// and only then.
func MustCategoryWithAnyPropertyProficiency(category content.WeaponCategory, properties []content.WeaponProperty) CategoryWithAnyPropertyProficiency {
	r, err := NewCategoryWithAnyPropertyProficiency(category, properties)
	if err != nil {
		panic(err)
	}
	return r
}

// Category returns the weapon category.
func (r CategoryWithAnyPropertyProficiency) Category() content.WeaponCategory { return r.category }

// Properties returns the properties as a new slice.
func (r CategoryWithAnyPropertyProficiency) Properties() []content.WeaponProperty {
	return slices.Clone(r.properties)
}

// Equal reports whether r and o have the same category and properties in order.
func (r CategoryWithAnyPropertyProficiency) Equal(o CategoryWithAnyPropertyProficiency) bool {
	return r.category == o.category && slices.Equal(r.properties, o.properties)
}

func (CategoryWithAnyPropertyProficiency) isWeaponProficiencyRule() {}

func (r CategoryWithAnyPropertyProficiency) matches(w content.Weapon) bool {
	if w.Category() != r.category {
		return false
	}
	return slices.ContainsFunc(w.Properties(), func(u content.WeaponPropertyUse) bool {
		return slices.Contains(r.properties, u.Property())
	})
}

// AnyStatBlockWeaponProficiency is a monster's proficiency with every weapon in
// its stat block (SRD 5.2.1, "Weapon Proficiency", p. 89). The host asks only
// about stat-block weapons.
type AnyStatBlockWeaponProficiency struct{}

func (AnyStatBlockWeaponProficiency) isWeaponProficiencyRule() {}

func (AnyStatBlockWeaponProficiency) matches(content.Weapon) bool { return true }

// WeaponProficiencies is the set of rules a creature is proficient by: it is
// proficient with a weapon when at least one rule matches. The zero value has
// no rule. Immutable; the rules are copied on construction. Safe for concurrent
// use.
type WeaponProficiencies struct{ rules []WeaponProficiencyRule }

// NewWeaponProficiencies returns the proficiencies made of rules. A nil rule is
// ignored.
func NewWeaponProficiencies(rules ...WeaponProficiencyRule) WeaponProficiencies {
	return WeaponProficiencies{rules: slices.DeleteFunc(slices.Clone(rules), func(r WeaponProficiencyRule) bool { return r == nil })}
}

// StatusFor returns WeaponProficient when any rule matches w, otherwise
// WeaponNotProficient.
func (p WeaponProficiencies) StatusFor(w content.Weapon) WeaponProficiencyStatus {
	for _, r := range p.rules {
		if r.matches(w) {
			return WeaponProficient
		}
	}
	return WeaponNotProficient
}

// WeaponProficienciesFromClassData builds one rule per element of a class's
// weapon training: CategoryProficiency without a property restriction,
// otherwise CategoryWithAnyPropertyProficiency. An empty slice gives no rule.
// It does not validate the data.
func WeaponProficienciesFromClassData(training []content.WeaponProficiency) WeaponProficiencies {
	rules := make([]WeaponProficiencyRule, 0, len(training))
	for _, t := range training {
		category := content.WeaponSimple
		if t.Category() == content.WeaponTrainingMartial {
			category = content.WeaponMartial
		}
		if props := t.RequiresAnyProperty(); len(props) > 0 {
			rules = append(rules, MustCategoryWithAnyPropertyProficiency(category, props))
		} else {
			rules = append(rules, CategoryProficiency(category))
		}
	}
	return NewWeaponProficiencies(rules...)
}
