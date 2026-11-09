package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func basisOf(t *testing.T, name string) character.AbilityBasedArmorClassBasis {
	t.Helper()
	id := content.MustFeatureID(name + "_unarmored_defense")
	switch name {
	case "barbarian":
		return must(character.NewAbilityBasedArmorClassBasis(id, []ability.Ability{ability.Constitution}, true))
	case "monk":
		return must(character.NewAbilityBasedArmorClassBasis(id, []ability.Ability{ability.Wisdom}, false))
	case "sorcerer":
		return must(character.NewAbilityBasedArmorClassBasis(id, []ability.Ability{ability.Charisma}, true))
	}
	return must(must(character.NewAbilityBasedArmorClassBasis(content.MustFeatureID("mage_armor"), nil, true)).WithBaseValue(13))
}

// Requirement: doc/specs/derived-stats.md FR-081, FR-036 - an ability based
// basis has a base value of 10 unless changed (at least 1), keeps its abilities
// in SRD order without repeats, and rejects an invalid id or ability.
// Path: unhappy (invalid input).
func TestAbilityBasedArmorClassBasis(t *testing.T) {
	t.Parallel()
	id := content.MustFeatureID("monk_unarmored_defense")
	b := must(character.NewAbilityBasedArmorClassBasis(id, []ability.Ability{ability.Wisdom, ability.Strength, ability.Wisdom}, false))
	if b.ID() != id || b.BaseValue() != 10 || b.ShieldAllowed() || b.IsZero() ||
		!slices.Equal(b.Abilities(), []ability.Ability{ability.Strength, ability.Wisdom}) {
		t.Errorf("basis = %+v", b)
	}
	got := b.Abilities()
	got[0] = ability.Charisma
	if b.Abilities()[0] != ability.Strength {
		t.Error("Abilities exposes its slice")
	}
	if w, err := b.WithBaseValue(13); err != nil || w.BaseValue() != 13 || b.BaseValue() != 10 {
		t.Errorf("WithBaseValue(13) = %d, %v (receiver %d)", w.BaseValue(), err, b.BaseValue())
	}
	if w, err := b.WithBaseValue(1); err != nil || w.BaseValue() != 1 {
		t.Errorf("WithBaseValue(1) = %v, %v", w, err)
	}
	_, e0 := b.WithBaseValue(0)
	_, e1 := character.NewAbilityBasedArmorClassBasis("", nil, true)
	_, e2 := character.NewAbilityBasedArmorClassBasis(id, []ability.Ability{0}, true)
	wantInvalid(t, e0, "base value 0")
	wantInvalid(t, e1, "empty id")
	wantInvalid(t, e2, "invalid ability")
	if !(character.AbilityBasedArmorClassBasis{}).IsZero() {
		t.Error("zero basis is not IsZero")
	}
}

// Requirement: doc/specs/derived-stats.md FR-064, FR-037 - rejection and slot
// names are stable.
// Path: happy.
func TestArmorClassBasisNames(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.ArmorClassBasisRejection]string{
		character.BasisNotApplicable: "notApplicable", character.BasisAlternativeNotSupplied: "alternativeNotSupplied", 0: "unknown",
	} {
		if r.String() != want {
			t.Errorf("%d.String() = %q", r, r.String())
		}
	}
	for s, want := range map[character.ArmorSlot]string{character.BodyArmorSlot: "bodyArmor", character.ShieldSlot: "shield", 0: "unknown"} {
		if s.String() != want {
			t.Errorf("%d.String() = %q", s, s.String())
		}
	}
}
