package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

var monkHit = character.MonkStunningHit{Hit: true, MonkWeaponOrUnarmed: true}

// Requirement: doc/specs/class-monk.md FR-076, FR-077, FR-078, FR-141 - a stunning strike spends 1 Focus Point and, against the Monk's Focus save DC, a failed
// Constitution save is StunApplied (the Stunned condition until the start of the Monk's next turn) and a successful one StunResisted (Speed halved and
// Advantage on the next attack roll against the target until then).
// Path: per case.
func TestStunningStrikeResolve(t *testing.T) {
	t.Parallel()
	stats, in := mnkStats(t, 5), mnkState(0, false, 0)
	ab := mnkAbilities(t, 3, 3, 5)
	failed, err := character.StunningStrike{}.Resolve(stats, in, ab, monkHit, d20.Failure)
	if err != nil || failed.DifficultyClass() != 14 || failed.State().FocusExpended() != 1 {
		t.Fatalf("failed save = %+v, %v", failed, err)
	}
	if e, ok := failed.Effect().(character.StunApplied); !ok || e.Condition != content.ConditionStunned || e.Until != character.MonkEffectEndsAtStartOfMonkNextTurn {
		t.Errorf("effect on a failed save = %#v", failed.Effect())
	}
	ev := failed.Event(character.MustIdentifier("pc-1")).(character.StunningStrikeAttempted)
	if !ev.Stunned() || ev.DifficultyClass() != 14 || ev.CharacterID() != character.MustIdentifier("pc-1") {
		t.Errorf("event = %#v", ev)
	}
	ok, err := character.StunningStrike{}.Resolve(stats, in, ab, monkHit, d20.Success)
	if err != nil || ok.State().FocusExpended() != 1 {
		t.Fatalf("successful save = %+v, %v", ok, err)
	}
	if e, isRes := ok.Effect().(character.StunResisted); !isRes || !e.SpeedHalved || !e.AdvantageOnNextAttack || e.Until != character.MonkEffectEndsAtStartOfMonkNextTurn {
		t.Errorf("effect on a successful save = %#v", ok.Effect())
	}
	if ok.Event(character.MustIdentifier("pc-1")).(character.StunningStrikeAttempted).Stunned() {
		t.Error("a resisted strike must not report Stunned")
	}
	if in != mnkState(0, false, 0) {
		t.Error("Resolve altered its input")
	}
}

// Requirement: doc/specs/class-monk.md FR-079, FR-080, FR-081, FR-135, Table 4.11a - Availability: levelTooLow below 5, notEligible for a miss or a hit with
// another weapon, alreadyUsed for a second strike in the turn (before the Focus Point check), notEnoughFocusPoints with none; Resolve refuses an outcome
// that is neither success nor failure and an unavailable use.
// Path: per case (unhappy: state error, rule interaction).
func TestStunningStrikeAvailability(t *testing.T) {
	t.Parallel()
	s := character.StunningStrike{}
	st5, none, spent := mnkStats(t, 5), mnkState(0, false, 0), mnkState(5, false, 0)
	wantReason(t, s.Availability(st5, none, monkHit), 0, "level 5")
	wantReason(t, s.Availability(mnkStats(t, 4), none, monkHit), character.MonkLevelTooLow, "level 4")
	wantReason(t, s.Availability(st5, none, character.MonkStunningHit{MonkWeaponOrUnarmed: true}), character.MonkNotEligible, "a miss")
	wantReason(t, s.Availability(st5, none, character.MonkStunningHit{Hit: true}), character.MonkNotEligible, "another weapon")
	wantReason(t, s.Availability(st5, none, character.MonkStunningHit{Hit: true, MonkWeaponOrUnarmed: true, UsedThisTurn: true}), character.MonkAlreadyUsed, "second strike")
	wantReason(t, s.Availability(st5, spent, character.MonkStunningHit{Hit: true, MonkWeaponOrUnarmed: true, UsedThisTurn: true}), character.MonkAlreadyUsed, "second strike, no points")
	wantReason(t, s.Availability(st5, spent, monkHit), character.MonkNotEnoughFocusPoints, "no points")
	wantReason(t, s.Availability(st5, spent, character.MonkStunningHit{}), character.MonkNotEligible, "a miss with no points")
	wantReason(t, s.Availability(character.MonkLevelStats{}, none, monkHit), character.MonkNotAMonk, "zero stats")
	_, err := s.Resolve(st5, none, mnkAbilities(t, 3, 3, 5), monkHit, 0)
	wantInvalid(t, err, "outcome")
	_, err = s.Resolve(st5, spent, mnkAbilities(t, 3, 3, 5), monkHit, d20.Failure)
	wantInvalid(t, err, "unavailable")
}

// Requirement: doc/specs/class-monk.md FR-076, FR-080 - the last Focus Point is enough for a strike, and an outcome above the valid ones, below them or zero is refused.
// Path: per case (unhappy: boundary, invalid input).
func TestStunningStrikeBoundaries(t *testing.T) {
	t.Parallel()
	st := mnkStats(t, 5) // 5 points
	wantReason(t, character.StunningStrike{}.Availability(st, mnkState(4, false, 0), monkHit), 0, "the last point")
	for _, bad := range []d20.Outcome{0, 3, -1} {
		_, err := character.StunningStrike{}.Resolve(st, mnkState(0, false, 0), mnkAbilities(t, 3, 3, 5), monkHit, bad)
		wantInvalid(t, err, "outcome")
	}
}
