package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// SpellcastingFeature is the class feature that gives a class its spells:
// Spellcasting, or Pact Magic for the Warlock (SRD 5.2.1, "Warlock", p. 71).
// The zero value is invalid. Comparable and safe for concurrent use.
type SpellcastingFeature int

// The spellcasting features.
const (
	// FeatureSpellcasting is the Spellcasting class feature.
	FeatureSpellcasting SpellcastingFeature = iota + 1
	// FeaturePactMagic is the Warlock's Pact Magic class feature.
	FeaturePactMagic
)

// String returns "spellcasting", "pact_magic" or "unknown".
func (f SpellcastingFeature) String() string {
	switch f {
	case FeatureSpellcasting:
		return "spellcasting"
	case FeaturePactMagic:
		return "pact_magic"
	}
	return "unknown"
}

// SpellcastingFocusKind is a kind of Spellcasting Focus (SRD 5.2.1, the class
// "Spellcasting Focus" entries, pp. 32-78; Warlock "Pact of the Blade" and
// "Pact of the Tome", p. 74). The class kinds come from a class; FocusPactWeapon
// and FocusBookOfShadows are granted by a feature the host reports. The zero
// value is invalid. Comparable and safe for concurrent use.
type SpellcastingFocusKind int

// The Spellcasting Focus kinds.
const (
	// FocusMusicalInstrument is a Bard's focus.
	FocusMusicalInstrument SpellcastingFocusKind = iota + 1
	// FocusHolySymbol is a Cleric's or Paladin's focus.
	FocusHolySymbol
	// FocusDruidic is a Druid's or Ranger's focus.
	FocusDruidic
	// FocusArcane is a Sorcerer's, Warlock's or Wizard's focus.
	FocusArcane
	// FocusSpellbook is a Wizard's focus.
	FocusSpellbook
	// FocusPactWeapon is the pact weapon of Pact of the Blade.
	FocusPactWeapon
	// FocusBookOfShadows is the Book of Shadows of Pact of the Tome.
	FocusBookOfShadows
)

// String returns the snake_case name or "unknown".
func (k SpellcastingFocusKind) String() string {
	switch k {
	case FocusMusicalInstrument:
		return "musical_instrument"
	case FocusHolySymbol:
		return "holy_symbol"
	case FocusDruidic:
		return "druidic"
	case FocusArcane:
		return "arcane"
	case FocusSpellbook:
		return "spellbook"
	case FocusPactWeapon:
		return "pact_weapon"
	case FocusBookOfShadows:
		return "book_of_shadows"
	}
	return "unknown"
}

// Valid reports whether k is one of the seven kinds.
func (k SpellcastingFocusKind) Valid() bool {
	return k >= FocusMusicalInstrument && k <= FocusBookOfShadows
}

// IsFeatureGranted reports whether k is granted by a feature rather than by a
// class (FocusPactWeapon, FocusBookOfShadows).
func (k SpellcastingFocusKind) IsFeatureGranted() bool {
	return k == FocusPactWeapon || k == FocusBookOfShadows
}

// SpellcastingClass is one of the eight classes with the Spellcasting or Pact
// Magic feature (SRD 5.2.1, "Classes", pp. 31-78). Barbarian, Fighter, Monk and
// Rogue have neither. The zero value is invalid. Comparable and safe for
// concurrent use.
type SpellcastingClass int

// The spellcasting classes.
const (
	CasterBard SpellcastingClass = iota + 1
	CasterCleric
	CasterDruid
	CasterPaladin
	CasterRanger
	CasterSorcerer
	CasterWarlock
	CasterWizard
)

// SpellcastingClasses returns the eight classes in alphabetical order.
func SpellcastingClasses() []SpellcastingClass {
	return []SpellcastingClass{CasterBard, CasterCleric, CasterDruid, CasterPaladin, CasterRanger, CasterSorcerer, CasterWarlock, CasterWizard}
}

// Valid reports whether c is one of the eight classes.
func (c SpellcastingClass) Valid() bool { return c >= CasterBard && c <= CasterWizard }

// String returns the class's snake_case name (also its content.ClassID) or "unknown".
func (c SpellcastingClass) String() string {
	if !c.Valid() {
		return "unknown"
	}
	return [...]string{"bard", "cleric", "druid", "paladin", "ranger", "sorcerer", "warlock", "wizard"}[c-1]
}

// ClassID returns the content id of the class, the key in a Spell's Classes;
// the zero ClassID for an invalid c.
func (c SpellcastingClass) ClassID() content.ClassID {
	if !c.Valid() {
		return ""
	}
	return content.ClassID(c.String())
}

// Feature returns FeaturePactMagic for the Warlock and FeatureSpellcasting for
// the other seven; the zero value for an invalid c.
func (c SpellcastingClass) Feature() SpellcastingFeature {
	switch {
	case c == CasterWarlock:
		return FeaturePactMagic
	case c.Valid():
		return FeatureSpellcasting
	}
	return 0
}

// Ability returns the class's spellcasting ability (Table A): Charisma for
// Bard, Paladin, Sorcerer and Warlock, Wisdom for Cleric, Druid and Ranger,
// Intelligence for Wizard. It returns the zero Ability for an invalid c.
func (c SpellcastingClass) Ability() ability.Ability {
	switch c {
	case CasterBard, CasterPaladin, CasterSorcerer, CasterWarlock:
		return ability.Charisma
	case CasterCleric, CasterDruid, CasterRanger:
		return ability.Wisdom
	case CasterWizard:
		return ability.Intelligence
	}
	return 0
}

// FocusKinds returns the class kinds of Spellcasting Focus the class uses
// (Table A) in a new slice; the Wizard also has FocusSpellbook.
func (c SpellcastingClass) FocusKinds() []SpellcastingFocusKind {
	switch c {
	case CasterBard:
		return []SpellcastingFocusKind{FocusMusicalInstrument}
	case CasterCleric, CasterPaladin:
		return []SpellcastingFocusKind{FocusHolySymbol}
	case CasterDruid, CasterRanger:
		return []SpellcastingFocusKind{FocusDruidic}
	case CasterSorcerer, CasterWarlock:
		return []SpellcastingFocusKind{FocusArcane}
	case CasterWizard:
		return []SpellcastingFocusKind{FocusArcane, FocusSpellbook}
	}
	return nil
}

// SpellcastingClassFor returns the spellcasting class with content id id, and
// false for any other class.
func SpellcastingClassFor(id content.ClassID) (SpellcastingClass, bool) {
	for _, c := range SpellcastingClasses() {
		if c.ClassID() == id {
			return c, true
		}
	}
	return 0, false
}

// isHalfCaster reports whether c is a Paladin or Ranger (Table D).
func (c SpellcastingClass) isHalfCaster() bool { return c == CasterPaladin || c == CasterRanger }
