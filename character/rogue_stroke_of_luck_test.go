package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/class-rogue.md FR-086, FR-087, FR-088, FR-089 (SRD 5.2.1, "Level 20: Stroke of Luck", p. 63; "D20 Tests", p. 6)
// - at Rogue level 20 a failed rolled D20 Test has every die turned into a 20 (so the natural value is 20 under any roll mode), the
// feature is spent and a StrokeOfLuckUsed event carries the character id; otherwise it is not available, with every failing reason in
// check order and the state unchanged.
// Path: per case (the Stroke of Luck cases of the spec): happy / unhappy (rule interaction, state error, empty case, boundary).
func TestStrokeOfLuckUse(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("rogue-1")
	spent := must(character.NewRogueState(character.StrokeOfLuckStateSpent, nil, 0))
	rolled := func(mode modifier.RollMode, dice ...int) character.StrokeOfLuckTest {
		return character.StrokeOfLuckRolledTest{Roll: d20.MustRoll(mode, dice...), Outcome: d20.Failure}
	}
	passed := character.StrokeOfLuckRolledTest{Roll: d20.MustRoll(modifier.Normal, 18), Outcome: d20.Success}
	R := func(r ...character.StrokeOfLuckReason) []character.StrokeOfLuckReason { return r }
	low, spentR, notFailed, noRoll := character.StrokeOfLuckLevelTooLow, character.StrokeOfLuckAlreadySpent, character.StrokeOfLuckTestNotFailed, character.StrokeOfLuckNoRoll
	for _, c := range []struct {
		name, path string
		level      int
		state      character.RogueState
		test       character.StrokeOfLuckTest
		dice       []int
		reasons    []character.StrokeOfLuckReason
	}{
		{"normal roll", "happy", 20, character.InitialRogueState(), rolled(modifier.Normal, 7), []int{20}, nil},
		{"Advantage", "happy", 20, character.InitialRogueState(), rolled(modifier.WithAdvantage, 18, 3), []int{20, 20}, nil},
		{"Disadvantage", "unhappy (rule interaction)", 20, character.InitialRogueState(), rolled(modifier.WithDisadvantage, 5, 3), []int{20, 20}, nil},
		{"unrolled failure", "unhappy (empty case)", 20, character.InitialRogueState(), character.StrokeOfLuckUnrolledTest{}, nil, R(noRoll)},
		{"the test succeeded", "unhappy (rule interaction)", 20, character.InitialRogueState(), passed, nil, R(notFailed)},
		{"already spent", "unhappy (state error)", 20, spent, rolled(modifier.Normal, 7), nil, R(spentR)},
		{"level 19", "unhappy (boundary)", 19, character.InitialRogueState(), rolled(modifier.Normal, 7), nil, R(low)},
		{"level 19, spent, succeeded", "unhappy (rule interaction)", 19, spent, passed, nil, R(low, spentR, notFailed)},
		{"every reason but the test succeeded", "unhappy (rule interaction)", 1, spent, character.StrokeOfLuckUnrolledTest{}, nil, R(low, spentR, noRoll)},
		{"a rolled test with the zero roll", "unhappy (invalid input)", 20, character.InitialRogueState(), character.StrokeOfLuckRolledTest{Outcome: d20.Failure}, nil, R(noRoll)},
		{"a nil test", "unhappy (invalid input)", 20, character.InitialRogueState(), nil, nil, R(noRoll)},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.state
			got := character.StrokeOfLuck{}.Use(id, rl(c.level), c.state, c.test)
			if c.state != before {
				t.Error("the state argument changed")
			}
			if c.reasons != nil {
				n, ok := got.(character.StrokeOfLuckNotAvailable)
				if !ok || !slices.Equal(n.Reasons(), c.reasons) {
					t.Fatalf("result = %#v, want not available %v", got, c.reasons)
				}
				n.Reasons()[0] = 0
				if n.Reasons()[0] == 0 {
					t.Error("Reasons shares storage")
				}
				return
			}
			s, ok := got.(character.StrokeOfLuckSpent)
			if !ok {
				t.Fatalf("result = %#v, want spent", got)
			}
			if !slices.Equal(s.Roll().Dice(), c.dice) || s.Roll().Natural() != 20 || s.Roll().Mode() != c.test.(character.StrokeOfLuckRolledTest).Roll.Mode() {
				t.Errorf("roll = %v natural %d mode %v, want %v natural 20", s.Roll().Dice(), s.Roll().Natural(), s.Roll().Mode(), c.dice)
			}
			if s.State().StrokeOfLuck() != character.StrokeOfLuckStateSpent || s.Event().CharacterID() != id {
				t.Errorf("state %v, event for %v", s.State().StrokeOfLuck(), s.Event().CharacterID())
			}
		})
	}
}

// Requirement: doc/specs/class-rogue.md FR-087, FR-088, FR-109 - using Stroke of Luck twice is refused until a Short Rest or a Long
// Rest, after which it can be used again; the transitions are pure functions of the state.
// Path: unhappy (state error).
func TestStrokeOfLuckStateTransitions(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("rogue-1")
	test := character.StrokeOfLuckRolledTest{Roll: d20.MustRoll(modifier.Normal, 2), Outcome: d20.Failure}
	for _, kind := range []character.RestKind{character.ShortRest, character.LongRest} {
		first := character.StrokeOfLuck{}.Use(id, rl(20), character.InitialRogueState(), test).(character.StrokeOfLuckSpent)
		if _, ok := (character.StrokeOfLuck{}).Use(id, rl(20), first.State(), test).(character.StrokeOfLuckNotAvailable); !ok {
			t.Fatal("the second use was not refused")
		}
		if _, ok := (character.StrokeOfLuck{}).Use(id, rl(20), first.State().AfterRest(kind), test).(character.StrokeOfLuckSpent); !ok {
			t.Errorf("not available after %v", kind)
		}
	}
}
