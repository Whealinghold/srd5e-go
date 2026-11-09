package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-rogue.md FR-108, FR-109 - RogueState is stored and read through FeatureStates under the key "rogue",
// an absent state is the initial one (zero value, Stroke of Luck available), it compares by value and is not equal to another
// feature's state.
// Path: happy.
func TestRogueStateStoredUnderItsKey(t *testing.T) {
	t.Parallel()
	if character.RogueStateKey().Name() != "rogue" {
		t.Errorf("key = %q, want rogue", character.RogueStateKey().Name())
	}
	var states character.FeatureStates
	if _, ok, err := character.ReadFeatureState(states, character.RogueStateKey()); ok || err != nil {
		t.Fatalf("an empty set holds a Rogue state: %v, %v", ok, err)
	}
	spent := must(character.NewRogueState(character.StrokeOfLuckStateSpent, []character.RogueExpertiseWindow{character.ExpertiseLevelOne}, content.LanguageElvish))
	states = must(character.SetFeatureState(states, character.RogueStateKey(), spent))
	got, ok, err := character.ReadFeatureState(states, character.RogueStateKey())
	if !ok || err != nil || got != spent || !got.Equal(spent) || got.Equal(character.InitialRogueState()) || got.Equal(character.InitialFighterState()) {
		t.Errorf("read = %v, %v, %v", got, ok, err)
	}
	if character.InitialRogueState() != (character.RogueState{}) || character.InitialRogueState().StrokeOfLuck() != character.StrokeOfLuckStateAvailable {
		t.Error("the initial state is not the zero value with Stroke of Luck available")
	}
}

// Requirement: doc/specs/class-rogue.md FR-108 - NewRogueState refuses an undefined Stroke of Luck state, window or language with
// ErrInvalidInput and records the windows and language it is given; WithExpertise marks the window of a selection and returns a
// new state.
// Path: per case (unhappy: invalid input).
func TestNewRogueState(t *testing.T) {
	t.Parallel()
	_, err := character.NewRogueState(character.StrokeOfLuckState(5), nil, 0)
	wantInvalid(t, err, "stroke")
	_, err = character.NewRogueState(character.StrokeOfLuckStateAvailable, []character.RogueExpertiseWindow{3}, 0)
	wantInvalid(t, err, "window")
	_, err = character.NewRogueState(character.StrokeOfLuckStateAvailable, nil, content.Language(99))
	wantInvalid(t, err, "language")
	s := must(character.NewRogueState(character.StrokeOfLuckStateAvailable, []character.RogueExpertiseWindow{character.ExpertiseLevelSix, character.ExpertiseLevelOne}, content.LanguageCommon))
	if !slices.Equal(s.ExpertiseWindowsUsed(), []character.RogueExpertiseWindow{character.ExpertiseLevelOne, character.ExpertiseLevelSix}) || s.OtherLanguage() != content.LanguageCommon {
		t.Errorf("state = %v %v", s.ExpertiseWindowsUsed(), s.OtherLanguage())
	}
	sel := must(character.RogueExpertise{}.Select(rl(1), []content.Skill{skillStealth, skillSleight}, nil, nil, []content.Skill{skillStealth, skillSleight}))
	one := character.InitialRogueState().WithExpertise(sel)
	if !slices.Equal(one.ExpertiseWindowsUsed(), []character.RogueExpertiseWindow{character.ExpertiseLevelOne}) || len(character.InitialRogueState().ExpertiseWindowsUsed()) != 0 {
		t.Errorf("WithExpertise = %v", one.ExpertiseWindowsUsed())
	}
	if got := one.WithExpertise(character.RogueExpertiseSelection{}); got != one {
		t.Error("an empty selection changed the state")
	}
	six := must(character.RogueExpertise{}.Select(rl(6), []content.Skill{skillStealth, skillSleight, skillPerc, skillAcro}, nil, []character.RogueExpertiseWindow{character.ExpertiseLevelOne}, []content.Skill{skillPerc, skillAcro}))
	if got := one.WithExpertise(six).ExpertiseWindowsUsed(); len(got) != 2 {
		t.Errorf("both windows = %v", got)
	}
	for st, want := range map[character.StrokeOfLuckState]string{character.StrokeOfLuckStateAvailable: "available", character.StrokeOfLuckStateSpent: "spent", 7: "unknown"} {
		if st.String() != want || st.Valid() != (want != "unknown") {
			t.Errorf("state %d = %q valid %v", st, st.String(), st.Valid())
		}
	}
}

// Requirement: doc/specs/class-rogue.md FR-088 (SRD 5.2.1, "Level 20: Stroke of Luck", p. 63; "Short Rest", p. 187; "Long Rest", p.
// 185) - a Short Rest or a Long Rest makes a spent Stroke of Luck available; any other kind and an available state change nothing,
// and the other fields are kept.
// Path: per case (Short Rest, Long Rest, an undefined kind): happy / unhappy (state error).
func TestRogueStateAfterRest(t *testing.T) {
	t.Parallel()
	spent := must(character.NewRogueState(character.StrokeOfLuckStateSpent, []character.RogueExpertiseWindow{character.ExpertiseLevelOne}, content.LanguageElvish))
	for _, c := range []struct {
		kind character.RestKind
		want character.StrokeOfLuckState
	}{{character.ShortRest, character.StrokeOfLuckStateAvailable}, {character.LongRest, character.StrokeOfLuckStateAvailable}, {character.RestKind(0), character.StrokeOfLuckStateSpent}} {
		got := spent.AfterRest(c.kind)
		if got.StrokeOfLuck() != c.want || got.OtherLanguage() != content.LanguageElvish || len(got.ExpertiseWindowsUsed()) != 1 || spent.StrokeOfLuck() != character.StrokeOfLuckStateSpent {
			t.Errorf("AfterRest(%v) = %v; want %v, other fields kept, receiver unchanged", c.kind, got, c.want)
		}
	}
	if got := character.InitialRogueState().AfterRest(character.LongRest); got != character.InitialRogueState() {
		t.Errorf("an available state changed: %v", got)
	}
}
