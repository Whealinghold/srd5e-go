package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestNewFeatGrant(t *testing.T) {
	t.Parallel()
	classFeat := character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_fighting_style"}
	for _, c := range []struct {
		name, requirement, path string
		source                  character.FeatSource
		scope                   character.FeatGrantScope
		ok                      bool
	}{
		{"background, specified feat", featsSpec + ` FR-004, Table G "Criminal background"; SRD 5.2.1, p. 83`, "happy", character.BackgroundFeat{Background: "criminal"}, character.SpecifiedFeat{Feat: "alert", Required: character.NoRequiredSpellList{}}, true},
		{"background, required list", featsSpec + " FR-076, Table G", "happy", character.BackgroundFeat{Background: "acolyte"}, character.SpecifiedFeat{Feat: "magic_initiate", Required: character.RequiredSpellListIs{Class: "cleric"}}, true},
		{"custom background, category", featsSpec + " FR-004, Table G", "happy", character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin}, true},
		{"species trait", featsSpec + " FR-004, Table G", "happy", character.SpeciesTraitFeat{Species: "human", Trait: "human_versatile"}, character.FeatFromCategory{Category: content.FeatOrigin}, true},
		{"class feature, any feat", featsSpec + " FR-004, Table G", "happy", classFeat, character.AnyFeat{}, true},
		{"eldritch invocation", featsSpec + " FR-004, Table G", "happy", character.EldritchInvocationFeat{Option: "lessons_of_the_first_ones"}, character.FeatFromCategory{Category: content.FeatOrigin}, true},
		{"bonus feat", featsSpec + " FR-004, Table G", "happy", character.BonusFeat{}, character.AnyFeat{}, true},
		{"nil source", featsSpec + " FR-004", "unhappy (invalid input)", nil, character.AnyFeat{}, false},
		{"invalid background", featsSpec + " FR-004", "unhappy (invalid input)", character.BackgroundFeat{}, character.AnyFeat{}, false},
		{"invalid species trait", featsSpec + " FR-004", "unhappy (invalid input)", character.SpeciesTraitFeat{Species: "human"}, character.AnyFeat{}, false},
		{"invalid species", featsSpec + " FR-004", "unhappy (invalid input)", character.SpeciesTraitFeat{Trait: "human_versatile"}, character.AnyFeat{}, false},
		{"invalid class feature", featsSpec + " FR-004", "unhappy (invalid input)", character.ClassFeatureFeat{Class: "fighter"}, character.AnyFeat{}, false},
		{"invalid class", featsSpec + " FR-004", "unhappy (invalid input)", character.ClassFeatureFeat{Feature: "x"}, character.AnyFeat{}, false},
		{"invalid invocation", featsSpec + " FR-004", "unhappy (invalid input)", character.EldritchInvocationFeat{}, character.AnyFeat{}, false},
		{"nil scope", featsSpec + " FR-004", "unhappy (invalid input)", classFeat, nil, false},
		{"invalid category", featsSpec + " FR-004", "unhappy (invalid input)", classFeat, character.FeatFromCategory{}, false},
		{"invalid specified feat", featsSpec + " FR-004", "unhappy (invalid input)", classFeat, character.SpecifiedFeat{Required: character.NoRequiredSpellList{}}, false},
		{"nil required list", featsSpec + " FR-076", "unhappy (invalid input)", classFeat, character.SpecifiedFeat{Feat: "alert"}, false},
		{"invalid required list", featsSpec + " FR-076", "unhappy (invalid input)", classFeat, character.SpecifiedFeat{Feat: "alert", Required: character.RequiredSpellListIs{}}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			g, err := character.NewFeatGrant(c.source, c.scope)
			if c.ok {
				if err != nil || g.IsZero() || g.Source() != c.source || g.Scope() != c.scope {
					t.Errorf("= %v, %v [%s]", g, err, c.requirement)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || !g.IsZero() || g.Source() != nil || g.Scope() != nil {
				t.Errorf("= %v, %v, want an invalid input error [%s]", g, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-065 - HeldFeatures is an immutable, sorted set without repeats; a blank id is refused with ErrInvalidInput; the zero value is empty.
// Path: per case.
func TestHeldFeatures(t *testing.T) {
	t.Parallel()
	ids := []content.FeatureID{"wizard_spellcasting", "fighter_fighting_style", "wizard_spellcasting"}
	h, err := character.NewHeldFeatures(ids...)
	if err != nil || !slices.Equal(h.IDs(), []content.FeatureID{"fighter_fighting_style", "wizard_spellcasting"}) {
		t.Fatalf("= %v, %v", h, err)
	}
	ids[0] = "x"
	h.IDs()[0] = "y"
	if !h.Contains("wizard_spellcasting") || !h.Contains("fighter_fighting_style") || h.Contains("x") || h.Contains("y") {
		t.Error("the set must be copied on construction and on read")
	}
	if !h.Equal(must(character.NewHeldFeatures("fighter_fighting_style", "wizard_spellcasting"))) || h.Equal(character.HeldFeatures{}) {
		t.Error("Equal must compare the sets")
	}
	if (character.HeldFeatures{}).Contains("x") || len((character.HeldFeatures{}).IDs()) != 0 {
		t.Error("the zero value is empty")
	}
	if _, err := character.NewHeldFeatures("fighter_fighting_style", ""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank id: %v", err)
	}
	if _, err := character.NewHeldFeatures("Not Valid"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("malformed id: %v", err)
	}
	many := make([]content.FeatureID, 1025)
	for i := range many {
		many[i] = content.FeatureID("f" + string(rune('a'+i%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i/676)))
	}
	if _, err := character.NewHeldFeatures(many...); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("1025 features: %v", err)
	}
	if _, err := character.NewHeldFeatures(many[:1024]...); err != nil {
		t.Errorf("1024 features: %v", err)
	}
}
