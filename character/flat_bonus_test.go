package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func itemBonus(t *testing.T, n int) character.FlatBonus {
	t.Helper()
	return must(character.NewFlatBonus(modifier.New(n), character.MagicItemFlatBonusSource{Item: content.MustMagicItemID("ring_of_protection")}))
}

// Requirement: doc/specs/derived-stats.md FR-038 - a FlatBonus holds a Modifier
// and a typed FlatBonusSource; a missing source is invalid.
// Path: per case.
func TestNewFlatBonus(t *testing.T) {
	t.Parallel()
	sources := map[string]character.FlatBonusSource{
		"spell":   character.SpellFlatBonusSource{Spell: content.MustSpellID("shield_of_faith")},
		"magic":   character.MagicItemFlatBonusSource{Item: content.MustMagicItemID("ring_of_protection")},
		"feature": character.FeatureFlatBonusSource{Feature: content.MustFeatureID("fighter_defense")},
	}
	for name, src := range sources {
		b, err := character.NewFlatBonus(modifier.New(2), src)
		if err != nil || b.Modifier().Value() != 2 || b.Source() != src || b.IsZero() {
			t.Errorf("%s: bonus = %v, %v", name, b, err)
		}
	}
	if _, err := character.NewFlatBonus(modifier.New(1), nil); err == nil {
		t.Error("nil source accepted")
	} else {
		wantInvalid(t, err, "nil source")
	}
	if !(character.FlatBonus{}).IsZero() {
		t.Error("zero FlatBonus is not IsZero")
	}
}

// Requirement: doc/specs/derived-stats.md FR-079 - an absent ability set means
// all six saves; an empty non-absent set, an invalid ability or a zero bonus is
// rejected.
// Path: per case.
func TestNewFlatSavingThrowBonus(t *testing.T) {
	t.Parallel()
	all, err := character.NewFlatSavingThrowBonus(itemBonus(t, 1), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range ability.All() {
		if !all.Covers(a) {
			t.Errorf("all-saves bonus does not cover %v", a)
		}
	}
	scoped, err := character.NewFlatSavingThrowBonus(itemBonus(t, 1), []ability.Ability{ability.Dexterity})
	if err != nil || !scoped.Covers(ability.Dexterity) || scoped.Covers(ability.Strength) || scoped.Bonus().Modifier().Value() != 1 || scoped.IsZero() {
		t.Errorf("scoped bonus = %v, %v", scoped, err)
	}
	_, e1 := character.NewFlatSavingThrowBonus(itemBonus(t, 1), []ability.Ability{})
	_, e2 := character.NewFlatSavingThrowBonus(itemBonus(t, 1), []ability.Ability{0})
	_, e3 := character.NewFlatSavingThrowBonus(character.FlatBonus{}, nil)
	wantInvalid(t, e1, "empty scope")
	wantInvalid(t, e2, "invalid ability")
	wantInvalid(t, e3, "zero bonus")
	if !(character.FlatSavingThrowBonus{}).IsZero() {
		t.Error("zero FlatSavingThrowBonus is not IsZero")
	}
}
