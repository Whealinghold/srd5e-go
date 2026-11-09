package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/character-aggregate.md section 5 (AdvancementClass) - the
// class needs a valid id, die and requirement; the zero value is invalid.
// Path: unhappy (invalid input).
func TestNewAdvancementClass(t *testing.T) {
	t.Parallel()
	a, err := character.NewAdvancementClass("fighter", content.HitDieD10, anyOf(str, dex))
	if err != nil || a.Class() != "fighter" || a.HitPointDie() != content.HitDieD10 || len(a.PrimaryAbility().Abilities()) != 2 || a.IsZero() {
		t.Fatalf("valid class: %v", err)
	}
	for name, f := range map[string]func() (character.AdvancementClass, error){
		"id": func() (character.AdvancementClass, error) {
			return character.NewAdvancementClass("", content.HitDieD10, allOf(str))
		},
		"die": func() (character.AdvancementClass, error) {
			return character.NewAdvancementClass("fighter", 0, allOf(str))
		},
		"nil": func() (character.AdvancementClass, error) {
			return character.NewAdvancementClass("fighter", content.HitDieD10, nil)
		},
		"zero allOf": func() (character.AdvancementClass, error) {
			return character.NewAdvancementClass("fighter", content.HitDieD10, content.AllOf{})
		},
	} {
		if _, err := f(); !isInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if !(character.AdvancementClass{}).IsZero() {
		t.Error("zero must report IsZero")
	}
}
