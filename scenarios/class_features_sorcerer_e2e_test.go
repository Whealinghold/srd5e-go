package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// grpD2Sorcerer is a Sorcerer of level n with Charisma 16 holding a fresh state (the points of its level and both Innate Sorcery uses) and the given Metamagic.
func grpD2Sorcerer(t *testing.T, n int, known ...character.MetamagicEffect) character.Character {
	t.Helper()
	profile := must2(character.NewSorcererProfile(character.MustSorcererLevel(n), ""))
	state := must2(character.RestoreSorcererState(profile, character.SorcererStateParts{SorceryPoints: profile.Level().SorceryPointsMaximum(), InnateSorceryUses: 2, Metamagic: known}))
	hero := grpD2Hero(t, scoresFor(t, 10, 10, 10, 10, 10, 16), grpD2CL("sorcerer", n))
	return must2(character.WithFeatureState(hero, character.SorcererStateKey(), state))
}

func grpD2Metamagic(effects ...character.MetamagicEffect) []content.FeatureOptionID {
	var out []content.FeatureOptionID
	for _, e := range effects {
		out = append(out, e.OptionID())
	}
	return out
}

func grpD2SorcererKnown(t *testing.T, c character.Character) []character.MetamagicEffect {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.SorcererStateKey())
	if err != nil || !ok {
		t.Fatalf("SorcererState = (%v, %v, %v)", s, ok, err)
	}
	return s.Metamagic()
}

// A level 2 Sorcerer over the bundled Sorcerer definition chooses its two Metamagic options, replaces one of them at the next opportunity, and at level 10 holds four. The command's
// character is never altered and the result carries one MetamagicOptionsChosen event (SRD 5.2.1, "Level 2: Metamagic", p. 65: two options at level 2, two more at levels 10 and 17,
// and one option can be swapped when a Sorcerer gains a level).
// Requirement: doc/specs/class-sorcerer.md FR-035, FR-055 to FR-058, FR-099 - ChooseMetamagicOptions through the public API.
// Path: happy.
func TestSorcererChooseMetamagicOptionsFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc := must2(character.NewChooseMetamagicOptions(must2(data.NewBundledSource())))
	do := func(c character.Character, chosen []content.FeatureOptionID) character.ChooseMetamagicOptionsResult {
		return must2(uc.Run(ctx, character.ChooseMetamagicOptionsCommand{Character: c, Chosen: chosen}))
	}
	hero := grpD2Sorcerer(t, 2)
	before := hero.FeatureStates()
	chosen, ok := do(hero, grpD2Metamagic(character.MetamagicTwinned, character.MetamagicCareful)).(character.ChooseMetamagicOptionsApplied)
	if !ok || !slices.Equal(grpD2SorcererKnown(t, chosen.Character()), []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicTwinned}) {
		t.Fatalf("two options at level 2 = %#v", chosen)
	}
	if ev, isEv := chosen.Events()[0].(character.MetamagicOptionsChosen); len(chosen.Events()) != 1 || !isEv || ev.CharacterID() != hero.ID() {
		t.Errorf("events = %#v", chosen.Events())
	}
	if !hero.FeatureStates().Equal(before) || len(grpD2SorcererKnown(t, hero)) != 0 {
		t.Error("the command's character was altered")
	}
	// One replacement is allowed, two are not.
	one, ok := do(chosen.Character(), grpD2Metamagic(character.MetamagicCareful, character.MetamagicDistant)).(character.ChooseMetamagicOptionsApplied)
	if !ok || !slices.Equal(grpD2SorcererKnown(t, one.Character()), []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicDistant}) {
		t.Errorf("one replacement = %#v", one)
	}
	if r, isRej := do(chosen.Character(), grpD2Metamagic(character.MetamagicDistant, character.MetamagicEmpowered)).(character.ChooseMetamagicOptionsRejected); !isRej || r.Rejection.Kind != character.SorcererMetamagicReplacementInvalid {
		t.Errorf("two replacements = %#v", r)
	}
	// Level 10 holds four.
	ten := grpD2Sorcerer(t, 10, character.MetamagicCareful, character.MetamagicTwinned)
	four, ok := do(ten, grpD2Metamagic(character.MetamagicCareful, character.MetamagicTwinned, character.MetamagicSubtle, character.MetamagicSeeking)).(character.ChooseMetamagicOptionsApplied)
	if !ok || len(grpD2SorcererKnown(t, four.Character())) != 4 {
		t.Errorf("four options at level 10 = %#v", four)
	}
}

// Every refusal and failure of ChooseMetamagicOptions: all selection problems are reported together, level 1 has no Metamagic, a Fighter is not a Sorcerer, a source without the
// Sorcerer is content-missing, a failing source is a content port failure, and bad construction or commands are invalid input.
// Requirement: doc/specs/class-sorcerer.md FR-035, FR-055 to FR-058, FR-099 - the rejection, content-missing and failure results of ChooseMetamagicOptions.
// Path: unhappy (invalid input, state error, missing content, collaborator failure, cancellation).
func TestSorcererChooseMetamagicOptionsRefusals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundled := must2(data.NewBundledSource())
	uc := must2(character.NewChooseMetamagicOptions(bundled))
	held := grpD2Sorcerer(t, 2, character.MetamagicCareful, character.MetamagicTwinned)
	r, isRej := must2(uc.Run(ctx, character.ChooseMetamagicOptionsCommand{Character: held, Chosen: []content.FeatureOptionID{"careful_spell", "careful_spell", "lucky_spell"}})).(character.ChooseMetamagicOptionsRejected)
	want := []character.MetamagicSelectionProblem{character.MetamagicSelectionWrongSize, character.MetamagicSelectionDuplicate, character.MetamagicSelectionUnknownOption}
	if !isRej || r.Rejection.Kind != character.SorcererMetamagicSelectionInvalid || !slices.Equal(r.Rejection.Problems, want) {
		t.Errorf("every problem = %#v, want %v", r, want)
	}
	level1 := must2(uc.Run(ctx, character.ChooseMetamagicOptionsCommand{Character: grpD2Sorcerer(t, 1)}))
	if r, ok := level1.(character.ChooseMetamagicOptionsRejected); !ok || r.Rejection.Kind != character.SorcererFeatureNotGained || r.Rejection.Feature != "metamagic" {
		t.Errorf("level 1 = %#v", level1)
	}
	fighter := must2(uc.Run(ctx, character.ChooseMetamagicOptionsCommand{Character: grpD2Flat(t, 10, grpD2CL("fighter", 4))}))
	if r, ok := fighter.(character.ChooseMetamagicOptionsRejected); !ok || r.Rejection.Kind != character.SorcererNotASorcerer {
		t.Errorf("a Fighter = %#v", fighter)
	}
	empty := must2(character.NewChooseMetamagicOptions(contenttest.NewFakeSource()))
	missing, ok := must2(empty.Run(ctx, character.ChooseMetamagicOptionsCommand{Character: held, Chosen: grpD2Metamagic(character.MetamagicCareful, character.MetamagicTwinned)})).(character.ChooseMetamagicOptionsContentMissing)
	if !ok || !slices.Equal(missing.Missing(), []content.MissingContent{character.SorcererClassID.Missing()}) {
		t.Errorf("a source without the Sorcerer = %#v", missing)
	}
	boom := errors.New("source down")
	res, err := must2(character.NewChooseMetamagicOptions(contenttest.NewFakeSource().FailLookup(boom))).Run(ctx, character.ChooseMetamagicOptionsCommand{Character: held})
	if res != nil {
		t.Errorf("a failing source returned %#v", res)
	}
	grpD2WantPortFailure(t, err, content.Port, boom)
	_, err = character.NewChooseMetamagicOptions(nil)
	grpD2WantInvalid(t, err, "a nil source")
	_, err = uc.Run(ctx, character.ChooseMetamagicOptionsCommand{})
	grpD2WantInvalid(t, err, "a zero character")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := uc.Run(cancelled, character.ChooseMetamagicOptionsCommand{Character: held}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context: %v", err)
	}
}
