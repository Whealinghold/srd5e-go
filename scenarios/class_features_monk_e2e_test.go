package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// grpD2Monk is a Monk of level n with Dexterity 16 and Wisdom 16 (+3 each), the scores the Open Hand features read.
func grpD2Monk(t *testing.T, n int) character.Character {
	t.Helper()
	return grpD2Hero(t, scoresFor(t, 10, 16, 10, 10, 16, 10), grpD2CL("monk", n))
}

func grpD2MonkState(t *testing.T, c character.Character) character.MonkState {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.MonkStateKey())
	if err != nil || !ok {
		t.Fatalf("MonkState = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

// A level 6 Warrior of the Open Hand over the bundled Monk definition uses Wholeness of Body three times: each use rolls one Martial Arts die (a d8 at level 6), regains
// the roll plus the Wisdom modifier, and spends one use. The fourth is refused, a Short Rest does not return the uses and a Long Rest does (SRD 5.2.1, "Level 6: Wholeness of Body", p. 52).
// Requirement: doc/specs/class-monk.md FR-110, FR-111, FR-112, FR-124 - UseWholenessOfBody through the public API with the uses spent and restored by rest.
// Path: happy, with the unhappy boundary of the fourth use and of the rests.
func TestMonkWholenessOfBodyFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := must2(character.NewMonkProgression(grpD2Class(t, "monk")))
	hero := grpD2Monk(t, 6)
	cmd := func(c character.Character) character.UseWholenessOfBodyCommand {
		return character.UseWholenessOfBodyCommand{Character: c, Subclass: character.OpenHandSubclassID}
	}
	roller := dicetest.NewFixedRoller(3, 6, 1, 4)
	uc := must2(character.NewUseWholenessOfBody(prog, roller))
	for i, want := range []int{3 + 3, 6 + 3, 1 + 3} {
		res, ok := must2(uc.Run(ctx, cmd(hero))).(character.WholenessOfBodyApplied)
		if !ok || res.HitPointsToRegain() != want || len(res.Events()) != 1 {
			t.Fatalf("use %d = %#v, want %d Hit Points", i+1, res, want)
		}
		if ev, isEv := res.Events()[0].(character.WholenessOfBodyUsed); !isEv || ev.HitPointsToRegain() != want || ev.CharacterID() != hero.ID() {
			t.Errorf("use %d event = %#v", i+1, res.Events())
		}
		hero = res.Character()
		if got := grpD2MonkState(t, hero).WholenessOfBodySpent(); got != i+1 {
			t.Errorf("uses spent = %d, want %d", got, i+1)
		}
	}
	for _, e := range roller.Requests() {
		if e != dice.MustExpression(1, dice.D8(), 0) {
			t.Errorf("request %v, want 1d8", e)
		}
	}
	before := len(roller.Requests())
	if r, ok := must2(uc.Run(ctx, cmd(hero))).(character.WholenessOfBodyNotAvailable); !ok || r.Reason != character.MonkNoUsesRemaining || len(roller.Requests()) != before {
		t.Errorf("a fourth use = %#v, want no uses remaining and no roll", r)
	}
	spent := grpD2MonkState(t, hero)
	short := must2(character.WithFeatureState(hero, character.MonkStateKey(), spent.AfterRest(character.ShortRest)))
	if r, ok := must2(uc.Run(ctx, cmd(short))).(character.WholenessOfBodyNotAvailable); !ok || r.Reason != character.MonkNoUsesRemaining {
		t.Errorf("after a Short Rest = %#v, want still no uses", r)
	}
	long := must2(character.WithFeatureState(hero, character.MonkStateKey(), spent.AfterRest(character.LongRest)))
	if r, ok := must2(uc.Run(ctx, cmd(long))).(character.WholenessOfBodyApplied); !ok || r.HitPointsToRegain() != 4+3 {
		t.Errorf("after a Long Rest = %#v, want available (4 + 3)", r)
	}
}

// Every refusal of Wholeness of Body rolls no die: without the subclass a level 6 Monk is not eligible and a level 5 Monk is too low, a Fighter is not a Monk.
// Requirement: doc/specs/class-monk.md FR-112, FR-134, FR-135, FR-148 - the rejection reasons of UseWholenessOfBody.
// Path: unhappy (boundary, state error).
func TestMonkWholenessOfBodyRefusals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := must2(character.NewMonkProgression(grpD2Class(t, "monk")))
	for _, c := range []struct {
		name     string
		hero     character.Character
		subclass string
		want     character.MonkReason
	}{
		{"no subclass", grpD2Monk(t, 6), "", character.MonkNotEligible},
		{"level 5 with no subclass", grpD2Monk(t, 5), "", character.MonkLevelTooLow},
		{"no Monk levels", grpD2Flat(t, 10, grpD2CL("fighter", 6)), string(character.OpenHandSubclassID), character.MonkNotAMonk},
	} {
		roller := dicetest.NewFixedRoller(4)
		res, err := must2(character.NewUseWholenessOfBody(prog, roller)).Run(ctx, character.UseWholenessOfBodyCommand{Character: c.hero, Subclass: content.SubclassID(c.subclass)})
		if r, ok := res.(character.WholenessOfBodyNotAvailable); err != nil || !ok || r.Reason != c.want || len(roller.Requests()) != 0 {
			t.Errorf("%s = %#v, %v (requests %v), want not available %v", c.name, res, err, roller.Requests(), c.want)
		}
	}
}

// The failures of the two Monk use cases: an empty roller is a dice port failure; construction and commands that are zero are invalid input; a cancelled context is refused.
// Requirement: doc/specs/class-monk.md FR-126, FR-129, FR-147 - port failure, invalid input and cancellation of UseWholenessOfBody and EndQuiveringPalm.
// Path: unhappy (collaborator failure, invalid input, cancellation).
func TestMonkUseCaseFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := must2(character.NewMonkProgression(grpD2Class(t, "monk")))
	whole := character.UseWholenessOfBodyCommand{Character: grpD2Monk(t, 6), Subclass: character.OpenHandSubclassID}
	res, err := must2(character.NewUseWholenessOfBody(prog, dicetest.NewFixedRoller())).Run(ctx, whole)
	if res != nil {
		t.Errorf("Wholeness of Body with an empty roller returned %#v", res)
	}
	grpD2WantPortFailure(t, err, dice.Port, dicetest.ErrExhausted)
	_, err = character.NewUseWholenessOfBody(character.MonkProgression{}, dicetest.NewFixedRoller(1))
	grpD2WantInvalid(t, err, "Wholeness of Body with a zero progression")
	_, err = character.NewUseWholenessOfBody(prog, nil)
	grpD2WantInvalid(t, err, "Wholeness of Body with a nil roller")
	uc := must2(character.NewUseWholenessOfBody(prog, dicetest.NewFixedRoller(1)))
	_, err = uc.Run(ctx, character.UseWholenessOfBodyCommand{})
	grpD2WantInvalid(t, err, "Wholeness of Body with a zero character")
	_, err = character.UseWholenessOfBody{}.Run(ctx, whole)
	grpD2WantInvalid(t, err, "the zero Wholeness of Body use case")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := uc.Run(cancelled, whole); !errors.Is(err, context.Canceled) {
		t.Errorf("Wholeness of Body on a cancelled context: %v", err)
	}

	palm := character.EndQuiveringPalmCommand{Character: grpD2PalmMonk(t), End: character.QuiveringPalmEndWithEffect, SamePlane: true, Outcome: d20.Failure}
	pres, err := must2(character.NewEndQuiveringPalm(prog, dicetest.NewFixedRoller())).Run(ctx, palm)
	if pres != nil {
		t.Errorf("Quivering Palm with an empty roller returned %#v", pres)
	}
	grpD2WantPortFailure(t, err, dice.Port, dicetest.ErrExhausted)
	_, err = character.NewEndQuiveringPalm(character.MonkProgression{}, dicetest.NewFixedRoller(1))
	grpD2WantInvalid(t, err, "Quivering Palm with a zero progression")
	_, err = character.NewEndQuiveringPalm(prog, nil)
	grpD2WantInvalid(t, err, "Quivering Palm with a nil roller")
	end := must2(character.NewEndQuiveringPalm(prog, dicetest.NewFixedRoller(1)))
	_, err = end.Run(ctx, character.EndQuiveringPalmCommand{End: character.QuiveringPalmEndHarmlessly, SamePlane: true})
	grpD2WantInvalid(t, err, "Quivering Palm with a zero character")
	if _, err := end.Run(cancelled, palm); !errors.Is(err, context.Canceled) {
		t.Errorf("Quivering Palm on a cancelled context: %v", err)
	}
}

// grpD2PalmMonk is a level 17 Monk with Quivering Palm vibrations active on a target for 17 days and 4 Focus Points expended.
func grpD2PalmMonk(t *testing.T) character.Character {
	t.Helper()
	target := must2(character.NewQuiveringPalmTarget("orc-1"))
	return must2(character.WithFeatureState(grpD2Monk(t, 17), character.MonkStateKey(), character.MustMonkState(4, false, 0, must2(character.NewQuiveringPalmVibrations(target, 17)))))
}

// A level 17 Monk with vibrations active on an orc ends them. With effect, 10d12 are rolled and the target makes its Constitution saving throw against the Monk's Focus DC
// (8 + Proficiency Bonus 6 + Wisdom 3 = 17): a failure takes the total, a success half. Ending them harmlessly rolls nothing. Either clears the vibrations and keeps the
// expended Focus Points. With none active, on another plane, or with no Monk levels the end is refused and nothing is rolled (SRD 5.2.1, "Level 17: Quivering Palm", p. 52).
// Requirement: doc/specs/class-monk.md FR-117 to FR-120, FR-122, FR-123, FR-141 - EndQuiveringPalm through the public API.
// Path: happy (failed save, successful save, harmless) and unhappy (state errors).
func TestMonkEndQuiveringPalmFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := must2(character.NewMonkProgression(grpD2Class(t, "monk")))
	tens := func(face int) []int {
		out := make([]int, 10)
		for i := range out {
			out[i] = face
		}
		return out
	}
	for _, c := range []struct {
		name    string
		end     character.QuiveringPalmEnd
		outcome d20.Outcome
		damage  int
		dc      d20.DifficultyClass
		rolls   bool
	}{
		{"the target fails the save", character.QuiveringPalmEndWithEffect, d20.Failure, 60, 17, true},
		{"the target succeeds on the save", character.QuiveringPalmEndWithEffect, d20.Success, 30, 17, true},
		{"harmless", character.QuiveringPalmEndHarmlessly, 0, 0, 0, false},
	} {
		hero := grpD2PalmMonk(t)
		roller := dicetest.NewFixedRoller(tens(6)...)
		res, ok := must2(must2(character.NewEndQuiveringPalm(prog, roller)).Run(ctx, character.EndQuiveringPalmCommand{Character: hero, End: c.end, SamePlane: true, Outcome: c.outcome})).(character.QuiveringPalmEndApplied)
		if !ok || res.Resolution().DamageApplied() != c.damage || res.Resolution().DifficultyClass() != c.dc {
			t.Errorf("%s = %#v, want %d damage at DC %d", c.name, res, c.damage, c.dc)
			continue
		}
		after := grpD2MonkState(t, res.Character())
		if after.Vibrations().Active() || after.FocusExpended() != 4 || !grpD2MonkState(t, hero).Vibrations().Active() {
			t.Errorf("%s: vibrations active %v, Focus expended %d, the input must be unchanged", c.name, after.Vibrations().Active(), after.FocusExpended())
		}
		ev, isEv := res.Events()[0].(character.QuiveringPalmEnded)
		if len(res.Events()) != 1 || !isEv || ev.DamageApplied() != c.damage || ev.WithEffect() != (c.end == character.QuiveringPalmEndWithEffect) {
			t.Errorf("%s events = %#v", c.name, res.Events())
		}
		if reqs := roller.Requests(); c.rolls != (len(reqs) == 1 && reqs[0] == dice.MustExpression(10, dice.D12(), 0)) || (!c.rolls && len(reqs) != 0) {
			t.Errorf("%s requests = %v", c.name, reqs)
		}
	}
	for _, c := range []struct {
		name  string
		hero  character.Character
		end   character.QuiveringPalmEnd
		plane bool
		want  character.MonkReason
	}{
		{"none active", grpD2Monk(t, 17), character.QuiveringPalmEndHarmlessly, true, character.MonkNotEligible},
		{"another plane", grpD2PalmMonk(t), character.QuiveringPalmEndWithEffect, false, character.MonkNotEligible},
		{"no Monk levels", grpD2Flat(t, 10, grpD2CL("fighter", 17)), character.QuiveringPalmEndHarmlessly, true, character.MonkNotAMonk},
	} {
		roller := dicetest.NewFixedRoller()
		res, err := must2(character.NewEndQuiveringPalm(prog, roller)).Run(ctx, character.EndQuiveringPalmCommand{Character: c.hero, End: c.end, SamePlane: c.plane, Outcome: d20.Failure})
		if r, ok := res.(character.QuiveringPalmEndNotAvailable); err != nil || !ok || r.Reason != c.want || len(roller.Requests()) != 0 {
			t.Errorf("%s = %#v, %v, want not available %v and no roll", c.name, res, err, c.want)
		}
	}
	// An end with effect needs the target's saving throw outcome: without it the command is invalid input.
	_, err := must2(character.NewEndQuiveringPalm(prog, dicetest.NewFixedRoller())).Run(ctx, character.EndQuiveringPalmCommand{Character: grpD2PalmMonk(t), End: character.QuiveringPalmEndWithEffect, SamePlane: true})
	grpD2WantInvalid(t, err, "an end with effect and no saving throw outcome")
}
