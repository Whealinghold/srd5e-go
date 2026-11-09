package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/spellcasting.md FR-001, FR-002, FR-068 and Table A - the
// eight classes, their feature, spellcasting ability and class focus kinds.
// Path: happy.
func TestSpellcastingClassTableA(t *testing.T) {
	t.Parallel()
	cases := []struct {
		class   character.SpellcastingClass
		name    string
		feature character.SpellcastingFeature
		ability ability.Ability
		focuses []character.SpellcastingFocusKind
	}{
		{character.CasterBard, "bard", character.FeatureSpellcasting, ability.Charisma, []character.SpellcastingFocusKind{character.FocusMusicalInstrument}},
		{character.CasterCleric, "cleric", character.FeatureSpellcasting, ability.Wisdom, []character.SpellcastingFocusKind{character.FocusHolySymbol}},
		{character.CasterDruid, "druid", character.FeatureSpellcasting, ability.Wisdom, []character.SpellcastingFocusKind{character.FocusDruidic}},
		{character.CasterPaladin, "paladin", character.FeatureSpellcasting, ability.Charisma, []character.SpellcastingFocusKind{character.FocusHolySymbol}},
		{character.CasterRanger, "ranger", character.FeatureSpellcasting, ability.Wisdom, []character.SpellcastingFocusKind{character.FocusDruidic}},
		{character.CasterSorcerer, "sorcerer", character.FeatureSpellcasting, ability.Charisma, []character.SpellcastingFocusKind{character.FocusArcane}},
		{character.CasterWarlock, "warlock", character.FeaturePactMagic, ability.Charisma, []character.SpellcastingFocusKind{character.FocusArcane}},
		{character.CasterWizard, "wizard", character.FeatureSpellcasting, ability.Intelligence, []character.SpellcastingFocusKind{character.FocusArcane, character.FocusSpellbook}},
	}
	if got := len(character.SpellcastingClasses()); got != len(cases) {
		t.Fatalf("SpellcastingClasses has %d classes, want %d", got, len(cases))
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if !c.class.Valid() || c.class.String() != c.name || c.class.ClassID() != classOf(c.name) {
				t.Errorf("class %s: Valid=%v String=%q ClassID=%q", c.name, c.class.Valid(), c.class, c.class.ClassID())
			}
			if c.class.Feature() != c.feature || c.class.Ability() != c.ability {
				t.Errorf("%s: feature %v ability %v, want %v %v", c.name, c.class.Feature(), c.class.Ability(), c.feature, c.ability)
			}
			f := c.class.FocusKinds()
			if len(f) != len(c.focuses) || f[0] != c.focuses[0] || (len(f) > 1 && f[1] != c.focuses[1]) {
				t.Errorf("%s focus kinds = %v, want %v", c.name, f, c.focuses)
			}
			if got, ok := character.SpellcastingClassFor(classOf(c.name)); !ok || got != c.class {
				t.Errorf("SpellcastingClassFor(%s) = %v, %v", c.name, got, ok)
			}
		})
	}
}

// Requirement: doc/specs/spellcasting.md FR-001, FR-008 - a class without the
// Spellcasting feature is not a spellcasting class, and the zero class is invalid.
// Path: unhappy (invalid input).
func TestSpellcastingClassRejectsOthers(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"fighter", "barbarian", "monk", "rogue"} {
		if _, ok := character.SpellcastingClassFor(classOf(name)); ok {
			t.Errorf("%s is a spellcasting class", name)
		}
	}
	var zero character.SpellcastingClass
	if zero.Valid() || zero.String() != "unknown" || zero.ClassID() != "" || zero.Feature() != 0 || zero.Ability() != 0 || zero.FocusKinds() != nil {
		t.Errorf("zero class is not inert: %v", zero)
	}
	if character.SpellcastingClass(9).Valid() {
		t.Error("class 9 is valid")
	}
}

// Requirement: doc/specs/spellcasting.md FR-129 - SpellcastingFocusKind has exactly
// seven values; only the pact weapon and Book of Shadows are feature-granted.
// Path: unhappy (boundary).
func TestSpellcastingFocusKinds(t *testing.T) {
	t.Parallel()
	want := []struct {
		k       character.SpellcastingFocusKind
		name    string
		granted bool
	}{
		{character.FocusMusicalInstrument, "musical_instrument", false}, {character.FocusHolySymbol, "holy_symbol", false},
		{character.FocusDruidic, "druidic", false}, {character.FocusArcane, "arcane", false}, {character.FocusSpellbook, "spellbook", false},
		{character.FocusPactWeapon, "pact_weapon", true}, {character.FocusBookOfShadows, "book_of_shadows", true},
	}
	for i, w := range want {
		if !w.k.Valid() || w.k.String() != w.name || w.k.IsFeatureGranted() != w.granted || int(w.k) != i+1 {
			t.Errorf("kind %d = %v granted %v", i+1, w.k, w.k.IsFeatureGranted())
		}
	}
	for _, bad := range []character.SpellcastingFocusKind{0, 8, 100} {
		if bad.Valid() || bad.String() != "unknown" || bad.IsFeatureGranted() {
			t.Errorf("kind %d is valid", bad)
		}
	}
	if character.FeatureSpellcasting.String() != "spellcasting" || character.FeaturePactMagic.String() != "pact_magic" || character.SpellcastingFeature(0).String() != "unknown" {
		t.Error("SpellcastingFeature.String")
	}
}
