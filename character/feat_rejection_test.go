package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/feats-behaviour.md section 5, section 7 - each rejection variant prints its snake_case name, the value of the srd5e.character.rejection attribute.
// Path: per case.
func TestFeatRejectionStrings(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		want string
		rj   character.FeatRejection
	}{
		{"feat_behaviour_unknown", character.FeatBehaviourUnknown{}}, {"feat_outside_grant", character.FeatOutsideGrant{}}, {"feat_not_repeatable", character.FeatNotRepeatable{}},
		{"feat_prerequisite_not_met", character.FeatPrerequisiteNotMet{}}, {"feat_choices_invalid", character.FeatChoicesInvalid{}}, {"feat_increase_rejected", character.FeatIncreaseRejected{}},
		{"spell_change_not_allowed_now", character.SpellChangeNotAllowedNow{}}, {"spell_change_already_used", character.SpellChangeAlreadyUsed{}},
		{"feat_use_already_expended", character.FeatUseAlreadyExpended{}}, {"feat_not_held", character.FeatNotHeld{}}, {"feat_has_no_limited_use", character.FeatHasNoLimitedUse{}},
		{"feat_not_replaceable", character.FeatNotReplaceable{}},
	} {
		if c.rj.String() != c.want {
			t.Errorf("%T = %q, want %q", c.rj, c.rj.String(), c.want)
		}
	}
	if len((character.FeatPrerequisiteNotMet{}).Unmet()) != 0 || len((character.FeatChoicesInvalid{}).Problems()) != 0 {
		t.Error("a zero rejection lists nothing")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-015, FR-019, FR-057 - the unmet clauses and the choice problems of a rejection are copies, and a rejection carries the feat it concerns.
// Path: unhappy (immutability).
func TestFeatRejectionCollectionsAreCopies(t *testing.T) {
	t.Parallel()
	prereq := takeRejected(t, charAt(t, 3), character.TakenFeats{}, grapplerFeat(), anyGrant(), singleOf(0+1), character.HeldFeatures{}).(character.FeatPrerequisiteNotMet)
	if len(prereq.Unmet()) != 2 {
		t.Fatalf("unmet = %v", prereq.Unmet())
	}
	prereq.Unmet()[0] = nil
	if prereq.Unmet()[0] == nil {
		t.Error("Unmet must return a copy")
	}
	invalid := takeRejected(t, charAt(t, 8), character.TakenFeats{}, skilledFeat(), customOrigin(), character.NoFeatChoices{}, character.HeldFeatures{}).(character.FeatChoicesInvalid)
	invalid.Problems()[0] = character.FeatChoiceProblem{}
	if invalid.Problems()[0].Name != character.FeatChoiceFeatChoicesShape {
		t.Error("Problems must return a copy")
	}
	if (character.FeatNotHeld{Feat: "alert"}).Feat != content.FeatID("alert") {
		t.Error("a rejection carries its feat")
	}
}
