package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/initiative-and-actions.md FR-151, FR-150 - the small enums name their values, an undefined value is "unknown", and the closed states are typed enums rather than booleans.
// Path: unhappy (boundary).
func TestTurnEnumNames(t *testing.T) {
	t.Parallel()
	cases := []struct{ got, want string }{
		{combat.ResourceAvailable.String(), "available"}, {combat.ResourceSpent.String(), "spent"}, {combat.ResourceState(0).String(), "unknown"},
		{combat.Disengaged.String(), "disengaged"}, {combat.NotDisengaged.String(), "not_disengaged"}, {combat.DisengageState(0).String(), "unknown"},
		{combat.DodgeBenefitsActive.String(), "active"}, {combat.DodgeBenefitsNone.String(), "none"}, {combat.DodgeBenefits(0).String(), "unknown"},
		{combat.AllyConsentWilling.String(), "willing"}, {combat.AllyConsentUnwilling.String(), "unwilling"}, {combat.AllyConsent(0).String(), "unknown"},
		{combat.AttackTimingRightBeforeLeavingReach.String(), "right_before_leaving_reach"}, {combat.AttackTiming(0).String(), "unknown"},
		{combat.DepartureByAction.String(), "action"}, {combat.DepartureByBonusAction.String(), "bonus_action"}, {combat.DepartureByReaction.String(), "reaction"},
		{combat.DepartureBySpeed.String(), "speed"}, {combat.DepartureMeans(0).String(), "unknown"},
		{combat.VisibilitySeen.String(), "seen"}, {combat.VisibilityUnseen.String(), "unseen"}, {combat.Visibility(0).String(), "unknown"},
		{combat.SpeedNameWalk.String(), "walk"}, {combat.SpeedNameSwim.String(), "swim"}, {combat.SpeedName(0).String(), "unknown"},
		{combat.MovementModeWalk.String(), "walk"}, {combat.MovementModeBurrow.String(), "burrow"}, {combat.MovementModeClimb.String(), "climb"},
		{combat.MovementModeCrawl.String(), "crawl"}, {combat.MovementModeFly.String(), "fly"}, {combat.MovementModeSwim.String(), "swim"}, {combat.MovementMode(0).String(), "unknown"},
		{combat.InitiativeMethodRolled.String(), "rolled"}, {combat.InitiativeMethodScore.String(), "score"}, {combat.InitiativeMethod(0).String(), "unknown"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("String() = %q, want %q", c.got, c.want)
		}
	}
	if combat.DepartureMeans(5).Valid() || !combat.DepartureBySpeed.Valid() || combat.Visibility(3).Valid() || combat.InitiativeMethod(3).Valid() || combat.AllyConsent(3).Valid() || combat.MovementMode(7).Valid() || combat.SpeedName(6).Valid() {
		t.Error("an undefined enum value is valid")
	}
	var zero combat.TurnResources
	if zero.Action() != 0 || zero.MovedFeet() != 0 {
		t.Errorf("zero TurnResources = %+v", zero)
	}
}
