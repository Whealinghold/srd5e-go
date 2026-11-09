package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-123 - NewHigherLevelStep refuses a zero class, a
// nil gain and an invalid subclass id, and reports the optional parts as absent when not given.
// Path: per case.
func TestNewHigherLevelStep(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	adv := must(character.NewAdvancementClass(f.ID(), f.HitPointDie(), f.PrimaryAbility()))
	gain := classFixedGain(f)
	for _, c := range []struct {
		name     string
		class    character.AdvancementClass
		gain     character.HitPointGain
		subclass content.SubclassID
	}{
		{"zero class", character.AdvancementClass{}, gain, ""},
		{"nil gain", adv, nil, ""},
		{"invalid subclass", adv, gain, "A"},
	} {
		if _, err := character.NewHigherLevelStep(c.class, c.gain, c.subclass, character.AbilityScoreIncrease{}); !isInvalid(err) {
			t.Errorf("%s: error = %v, want invalid input", c.name, err)
		}
	}
	s := must(character.NewHigherLevelStep(adv, gain, "", character.AbilityScoreIncrease{}))
	if _, has := s.Subclass(); has {
		t.Error("an empty subclass id must mean none")
	}
	if _, has := s.Increase(); has {
		t.Error("the zero increase must mean none")
	}
	if s.IsZero() || !(character.HigherLevelStep{}).IsZero() || s.Class().Class() != "fighter" || s.HitPointGain() != gain {
		t.Error("step accessors must return what was given")
	}
	s2 := must(character.NewHigherLevelStep(adv, gain, "champion", plusTwo(ability.Strength)))
	if sub, has := s2.Subclass(); !has || sub != "champion" {
		t.Error("the subclass must be reported")
	}
	if inc, has := s2.Increase(); !has || !inc.Equal(plusTwo(ability.Strength)) {
		t.Error("the increase must be reported")
	}
}
