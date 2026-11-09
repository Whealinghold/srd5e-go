package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

func palmStart(unarmed bool) character.QuiveringPalmStart {
	return character.QuiveringPalmStart{Target: mnkTarget(), HitWithUnarmedStrike: unarmed}
}

// Requirement: doc/specs/class-monk.md FR-114, FR-115, FR-116, FR-121, FR-135, FR-148, Table 4.11a - Quivering Palm starts with 4 Focus Points on an Unarmed
// Strike hit in the Warrior of the Open Hand from level 17, recording the target and days equal to the Monk level (the engine reads no clock); it is
// levelTooLow below 17 (before the points), notEligible for another subclass or a weapon hit (before the points), alreadyUsed while a creature is under the
// effect (before the points) and notEnoughFocusPoints with fewer than 4.
// Path: per case (happy, unhappy: boundary, state error, rule interaction).
func TestQuiveringPalmStart(t *testing.T) {
	t.Parallel()
	q := character.QuiveringPalm{}
	st := mnkStats(t, 17)
	res, err := q.Start(st, mnkState(0, false, 0), openHand, palmStart(true))
	if err != nil || res.Vibrations().DurationDays() != 17 || res.Vibrations().Target() != mnkTarget() || res.State().FocusExpended() != 4 || res.State().Vibrations() != res.Vibrations() {
		t.Fatalf("Start = %+v, %v", res, err)
	}
	if ev, ok := res.Event(character.MustIdentifier("pc-1")).(character.QuiveringPalmStarted); !ok || ev.DurationDays() != 17 || ev.Target() != mnkTarget() {
		t.Errorf("event = %#v", res.Event(character.MustIdentifier("pc-1")))
	}
	if res20 := must(q.Start(mnkStats(t, 20), mnkState(0, false, 0), openHand, palmStart(true))); res20.Vibrations().DurationDays() != 20 {
		t.Errorf("days at level 20 = %d, want 20", res20.Vibrations().DurationDays())
	}
	for _, c := range []struct {
		name     string
		stats    character.MonkLevelStats
		state    character.MonkState
		subclass content.SubclassID
		req      character.QuiveringPalmStart
		want     character.MonkReason
	}{
		{"enough points", st, mnkState(13, false, 0), openHand, palmStart(true), 0},
		{"3 points", st, mnkState(14, false, 0), openHand, palmStart(true), character.MonkNotEnoughFocusPoints},
		{"a Monk weapon with 3 points", st, mnkState(14, false, 0), openHand, palmStart(false), character.MonkNotEligible},
		{"level 16 and no points", mnkStats(t, 16), mnkState(16, false, 0), openHand, palmStart(true), character.MonkLevelTooLow},
		{"another subclass and no points", st, mnkState(17, false, 0), "other", palmStart(true), character.MonkNotEligible},
		{"no subclass", st, mnkState(0, false, 0), "", palmStart(true), character.MonkNotEligible},
		{"no target", st, mnkState(0, false, 0), openHand, character.QuiveringPalmStart{HitWithUnarmedStrike: true}, character.MonkNotEligible},
		{"another creature under the effect", st, character.MustMonkState(0, false, 0, mnkVibrations(3)), openHand, palmStart(true), character.MonkAlreadyUsed},
		{"under the effect and no points", st, character.MustMonkState(17, false, 0, mnkVibrations(3)), openHand, palmStart(true), character.MonkAlreadyUsed},
		{"zero stats", character.MonkLevelStats{}, mnkState(0, false, 0), openHand, palmStart(true), character.MonkNotAMonk},
	} {
		wantReason(t, q.StartAvailability(c.stats, c.state, c.subclass, c.req), c.want, c.name)
		if c.want != 0 {
			_, err := q.Start(c.stats, c.state, c.subclass, c.req)
			wantInvalid(t, err, c.name)
		}
	}
}

// Requirement: doc/specs/class-monk.md Table 4.10c, FR-117 to FR-120, FR-122, FR-123, FR-141 - ending with effect makes the target save against the Monk's Focus
// save DC and takes the 10d12 Force damage on a failure and half (rounded down) on a success: the four rows (all 1s failed: 10; all 12s failed: 120; all 2s
// succeeded: 10; all 1s succeeded: 5); the harmless end rolls nothing; both clear the vibrations; with no vibrations, or with effect on another plane, it is
// notEligible and the vibrations stay.
// Path: per case (rows of Table 4.10c, unhappy: state error).
func TestQuiveringPalmFinish(t *testing.T) {
	t.Parallel()
	q := character.QuiveringPalm{}
	st := mnkStats(t, 17)
	active := character.MustMonkState(4, false, 0, mnkVibrations(17))
	ab := mnkAbilities(t, 3, 3, 17)
	for _, c := range []struct {
		name    string
		outcome d20.Outcome
		total   int
		damage  int
	}{{"failed, lowest", d20.Failure, 10, 10}, {"failed, highest", d20.Failure, 120, 120}, {"successful, even total", d20.Success, 20, 10}, {"successful, lowest", d20.Success, 10, 5}, {"successful, odd total", d20.Success, 21, 10}} {
		res, err := q.Finish(st, active, ab, character.QuiveringPalmEndWithEffect, true, c.outcome, c.total)
		if err != nil || res.DamageApplied() != c.damage || res.DifficultyClass() != d20.DifficultyClass(8+3+6) || res.DamageType() != content.DamageForce ||
			res.State() != mnkState(4, false, 0) || res.End() != character.QuiveringPalmEndWithEffect {
			t.Errorf("%s: Finish = %+v, %v", c.name, res, err)
		}
		if ev := res.Event(character.MustIdentifier("pc-1")).(character.QuiveringPalmEnded); !ev.WithEffect() || ev.DamageApplied() != c.damage {
			t.Errorf("%s: event = %#v", c.name, ev)
		}
	}
	harmless, err := q.Finish(st, active, ab, character.QuiveringPalmEndHarmlessly, false, 0, 0)
	if err != nil || harmless.DamageApplied() != 0 || harmless.DifficultyClass() != 0 || harmless.State().Vibrations().Active() || harmless.End() != character.QuiveringPalmEndHarmlessly {
		t.Errorf("harmless end = %+v, %v", harmless, err)
	}
	if ev := harmless.Event(character.MustIdentifier("pc-1")).(character.QuiveringPalmEnded); ev.WithEffect() || ev.DamageApplied() != 0 {
		t.Errorf("harmless event = %#v", ev)
	}
	wantReason(t, q.EndAvailability(st, active, character.QuiveringPalmEndWithEffect, true), 0, "with effect")
	wantReason(t, q.EndAvailability(st, active, character.QuiveringPalmEndWithEffect, false), character.MonkNotEligible, "another plane")
	wantReason(t, q.EndAvailability(st, active, character.QuiveringPalmEndHarmlessly, false), 0, "harmless on another plane")
	wantReason(t, q.EndAvailability(st, mnkState(0, false, 0), character.QuiveringPalmEndHarmlessly, true), character.MonkNotEligible, "none active")
	wantReason(t, q.EndAvailability(st, active, 0, true), character.MonkNotEligible, "unknown way")
	wantReason(t, q.EndAvailability(character.MonkLevelStats{}, active, character.QuiveringPalmEndHarmlessly, true), character.MonkNotAMonk, "zero stats")
	for _, c := range []struct {
		name  string
		end   character.QuiveringPalmEnd
		plane bool
		o     d20.Outcome
		total int
	}{
		{"another plane", character.QuiveringPalmEndWithEffect, false, d20.Failure, 50}, {"total 9", character.QuiveringPalmEndWithEffect, true, d20.Failure, 9},
		{"total 121", character.QuiveringPalmEndWithEffect, true, d20.Failure, 121}, {"no outcome", character.QuiveringPalmEndWithEffect, true, 0, 50},
		{"harmless with dice", character.QuiveringPalmEndHarmlessly, true, 0, 50},
	} {
		_, err := q.Finish(st, active, ab, c.end, c.plane, c.o, c.total)
		wantInvalid(t, err, c.name)
	}
	if q.DamageDice() != dice.MustExpression(10, dice.D12(), 0) {
		t.Errorf("damage dice = %v, want 10d12", q.DamageDice())
	}
}

// Requirement: doc/specs/class-monk.md FR-115, FR-117, FR-119, FR-122 - a subclass id that sorts after the Warrior of the Open Hand is foreign too, an unknown way to end the vibrations (above the
// known ones) is notEligible, a harmless end with a negative roll total and an end with effect with an outcome above the valid ones are errors.
// Path: per case (unhappy: boundary, invalid input).
func TestQuiveringPalmBoundaries(t *testing.T) {
	t.Parallel()
	q := character.QuiveringPalm{}
	st := mnkStats(t, 17)
	active := character.MustMonkState(4, false, 0, mnkVibrations(17))
	wantReason(t, q.StartAvailability(st, mnkState(0, false, 0), "zzz_after", palmStart(true)), character.MonkNotEligible, "a later subclass")
	wantReason(t, q.EndAvailability(st, active, character.QuiveringPalmEnd(3), true), character.MonkNotEligible, "an unknown way")
	ab := mnkAbilities(t, 3, 3, 17)
	_, err := q.Finish(st, active, ab, character.QuiveringPalmEndHarmlessly, true, 0, -1)
	wantInvalid(t, err, "a negative total for a harmless end")
	_, err = q.Finish(st, active, ab, character.QuiveringPalmEndWithEffect, true, d20.Outcome(3), 50)
	wantInvalid(t, err, "an outcome above the valid ones")
	if v := must(q.Start(st, mnkState(0, false, 0), openHand, palmStart(true))).Vibrations(); v.Target() != mnkTarget() || v.DurationDays() != 17 {
		t.Errorf("vibrations = %+v", v)
	}
}
