package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-monk.md FR-038 to FR-042, FR-077 - the action and effect-end names are snake_case, an undefined value is unknown and the
// declared values are the first of their kind (1).
// Path: per case.
func TestMonkActionAndEffectEndNames(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ got, want string }{
		{character.MonkDisengage.String(), "disengage"}, {character.MonkDodge.String(), "dodge"}, {character.MonkDash.String(), "dash"}, {character.MonkAction(0).String(), "unknown"},
		{character.MonkAction(4).String(), "unknown"}, {character.MonkEffectEndsAtStartOfMonkNextTurn.String(), "start_of_monk_next_turn"}, {character.MonkEffectEnd(0).String(), "unknown"},
		{character.MonkEffectEnd(2).String(), "unknown"},
	} {
		if c.got != c.want {
			t.Errorf("name = %q, want %q", c.got, c.want)
		}
	}
	if character.MonkDisengage != 1 || character.MonkEffectEndsAtStartOfMonkNextTurn != 1 || character.MonkEffectEnd(1).String() != "start_of_monk_next_turn" {
		t.Error("the declared values must start at 1 so that the zero value is not one of them")
	}
}

// Requirement: doc/specs/class-monk.md FR-135, Table 4.11a - the first reason that applies wins, in the order notAMonk, levelTooLow, notEligible, alreadyUsed, noUsesRemaining,
// notEnoughFocusPoints: zero stats beat everything, the level gate is exact (the minimum level passes, one below fails) and an earlier reason beats a later one; shown through the services that
// compose them.
// Path: per case (unhappy: boundary, rule interaction).
func TestMonkAvailabilityOrder(t *testing.T) {
	t.Parallel()
	u := character.UncannyMetabolism{}
	used := mnkState(0, true, 0)
	wantReason(t, u.Availability(character.MonkLevelStats{}, used, false), character.MonkNotAMonk, "zero stats beat the rest")
	wantReason(t, u.Availability(mnkStats(t, 1), used, false), character.MonkLevelTooLow, "level beats eligibility")
	wantReason(t, u.Availability(mnkStats(t, 2), used, false), character.MonkNotEligible, "eligibility beats already used")
	wantReason(t, u.Availability(mnkStats(t, 2), used, true), character.MonkAlreadyUsed, "already used")
	wantReason(t, u.Availability(mnkStats(t, 2), mnkState(0, false, 0), true), 0, "nothing applies at the minimum level")
	// a Stunning Strike orders already used before the Focus Points
	s := character.StunningStrike{}
	spent := mnkState(5, false, 0)
	wantReason(t, s.Availability(mnkStats(t, 5), spent, character.MonkStunningHit{Hit: true, MonkWeaponOrUnarmed: true, UsedThisTurn: true}), character.MonkAlreadyUsed, "already used beats Focus Points")
	wantReason(t, s.Availability(mnkStats(t, 5), spent, monkHit), character.MonkNotEnoughFocusPoints, "Focus Points last")
	// a Wholeness of Body orders eligibility before the uses
	w := character.WholenessOfBody{}
	wantReason(t, w.Availability(mnkStats(t, 6), mnkState(0, false, 3), "", modifierOf(t, 3)), character.MonkNotEligible, "subclass before uses")
	wantReason(t, w.Availability(mnkStats(t, 6), mnkState(0, false, 3), openHand, modifierOf(t, 3)), character.MonkNoUsesRemaining, "uses")
}
