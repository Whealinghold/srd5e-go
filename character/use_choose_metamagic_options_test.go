package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-sorcerer.md FR-035, FR-055 to FR-058, FR-099 - ChooseMetamagicOptions sets the options known (the entitled number at the level, at most one dropped) on the new Character with a MetamagicOptionsChosen event and
// leaves its input unchanged; a refusal lists every problem, a second replacement is refused and level 1 has no Metamagic.
// Path: per case (happy / unhappy: invalid input, state).
func TestChooseMetamagicOptionsRun(t *testing.T) {
	t.Parallel()
	uc := must(character.NewChooseMetamagicOptions(sorSource(t)))
	do := func(c character.Character, chosen []content.FeatureOptionID) character.ChooseMetamagicOptionsResult {
		return must(uc.Run(t.Context(), character.ChooseMetamagicOptionsCommand{Character: c, Chosen: chosen}))
	}
	c2 := sorChar(t, 2, 16, "", sorFresh(sorProfile(2, "")))
	before := c2.FeatureStates()
	ok, isApplied := do(c2, sorOptionIDs(character.MetamagicTwinned, character.MetamagicCareful)).(character.ChooseMetamagicOptionsApplied)
	if !isApplied || !slices.Equal(sorStateOf(t, ok.Character()).Metamagic(), []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicTwinned}) {
		t.Fatalf("applied = %#v", ok)
	}
	if ev, isEv := ok.Events()[0].(character.MetamagicOptionsChosen); len(ok.Events()) != 1 || !isEv || ev.CharacterID() != c2.ID() {
		t.Errorf("events = %#v", ok.Events())
	}
	if !c2.FeatureStates().Equal(before) || len(sorStateOf(t, c2).Metamagic()) != 0 {
		t.Error("the command's Character was altered")
	}
	ok.Events()[0] = nil
	if ok.Events()[0] == nil {
		t.Error("Events returned a shared slice")
	}
	held := ok.Character()
	if r, isRej := do(held, sorOptionIDs(character.MetamagicDistant, character.MetamagicEmpowered)).(character.ChooseMetamagicOptionsRejected); !isRej || r.Rejection.Kind != character.SorcererMetamagicReplacementInvalid {
		t.Errorf("two replacements = %#v", r)
	}
	if r, isApp := do(held, sorOptionIDs(character.MetamagicCareful, character.MetamagicDistant)).(character.ChooseMetamagicOptionsApplied); !isApp ||
		!slices.Equal(sorStateOf(t, r.Character()).Metamagic(), []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicDistant}) {
		t.Errorf("one replacement = %#v", r)
	}
	r, isRej := do(held, []content.FeatureOptionID{"careful_spell", "careful_spell", "lucky_spell"}).(character.ChooseMetamagicOptionsRejected)
	wantProblems := []character.MetamagicSelectionProblem{character.MetamagicSelectionWrongSize, character.MetamagicSelectionDuplicate, character.MetamagicSelectionUnknownOption}
	if !isRej || r.Rejection.Kind != character.SorcererMetamagicSelectionInvalid || !slices.Equal(r.Rejection.Problems, wantProblems) {
		t.Errorf("every problem = %#v, want %v", r, wantProblems)
	}
	c1 := sorChar(t, 1, 16, "", sorFresh(sorProfile(1, "")))
	if r, isRej := do(c1, nil).(character.ChooseMetamagicOptionsRejected); !isRej || r.Rejection.Kind != character.SorcererFeatureNotGained || r.Rejection.Feature != "metamagic" {
		t.Errorf("level 1 = %#v", r)
	}
	c10 := sorChar(t, 10, 16, "", sorWith(sorProfile(10, ""), 10, []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicTwinned}, nil))
	if r, isApp := do(c10, sorOptionIDs(character.MetamagicCareful, character.MetamagicTwinned, character.MetamagicSubtle, character.MetamagicSeeking)).(character.ChooseMetamagicOptionsApplied); !isApp ||
		len(sorStateOf(t, r.Character()).Metamagic()) != 4 {
		t.Errorf("level 10 = %#v", r)
	}
	if r, isRej := do(sorChar(t, 17, 16, "", sorFresh(sorProfile(17, ""))), sorOptionIDs(1, 2, 3, 4)).(character.ChooseMetamagicOptionsRejected); !isRej || !slices.Equal(r.Rejection.Problems, []character.MetamagicSelectionProblem{character.MetamagicSelectionWrongSize}) {
		t.Errorf("four at level 17 = %#v", r)
	}
}
