package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/grappling.md FR-035, FR-036, table 4.10.1 - the escape options are Athletics (Strength) and
// Acrobatics (Dexterity), the ability is the skill's own, and an undefined option has neither.
// Path: per case.
func TestGrappleEscapeOptions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		option                  combat.GrappleEscapeOption
		skill                   content.Skill
		ability                 ability.Ability
		str                     string
	}{
		{"Athletics", `table 4.10.1; "Grappling", p. 182`, "happy", combat.GrappleEscapeOptionStrengthAthletics, content.SkillAthletics, ability.Strength, "strength_athletics"},
		{"Acrobatics", `table 4.10.1; "Grappling", p. 182`, "happy", combat.GrappleEscapeOptionDexterityAcrobatics, content.SkillAcrobatics, ability.Dexterity, "dexterity_acrobatics"},
		{"zero option", "FR-035", "unhappy (invalid input)", 0, 0, 0, "unknown"},
		{"option past the end", "FR-035", "unhappy (boundary)", 3, 0, 0, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.option.Skill() != c.skill || c.option.Ability() != c.ability || c.option.String() != c.str || c.option.Valid() != (c.skill != 0) {
				t.Errorf("option %d = %v/%v/%q/%v [%s]", int(c.option), c.option.Skill(), c.option.Ability(), c.option.String(), c.option.Valid(), c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-085, FR-087, FR-096, section 3 - each state enum has its snake_case names, and the end cause is
// valid exactly for its four members.
// Path: per case.
func TestGrappleStateNames(t *testing.T) {
	t.Parallel()
	check := func(name string, got, want []string) {
		t.Helper()
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s[%d] = %q, want %q", name, i, got[i], want[i])
			}
		}
	}
	check("PartState", []string{combat.PartStateHolding.String(), combat.PartStateNotHoldingGrapple.String(), combat.PartState(0).String()},
		[]string{"holding", "not_holding_grapple", "unknown"})
	check("HandAvailability", []string{combat.HandAvailabilityHandFree.String(), combat.HandAvailabilityNoHandFree.String(), combat.HandAvailability(0).String()},
		[]string{"hand_free", "no_hand_free", "unknown"})
	check("GrappleHold", []string{combat.GrappleHoldHolds.String(), combat.GrappleHoldGrapplerIncapacitated.String(), combat.GrappleHoldOutOfRange.String(), combat.GrappleHold(0).String()},
		[]string{"holds", "grappler_incapacitated", "out_of_range", "unknown"})
	check("GrappleNotPossibleReason", []string{combat.GrappleNotPossibleReasonTargetTooLarge.String(), combat.GrappleNotPossibleReasonNoFreeHand.String(), combat.GrappleNotPossibleReason(0).String()},
		[]string{"target_too_large", "no_free_hand", "unknown"})
	check("GrappleEndCause", []string{combat.GrappleEndCauseEscaped.String(), combat.GrappleEndCauseGrapplerIncapacitated.String(), combat.GrappleEndCauseOutOfRange.String(), combat.GrappleEndCauseReleased.String(), combat.GrappleEndCause(0).String()},
		[]string{"escaped", "grappler_incapacitated", "out_of_range", "released", "unknown"})
	for c, want := range map[combat.GrappleEndCause]bool{0: false, 1: true, 2: true, 3: true, 4: true, 5: false} {
		if c.Valid() != want {
			t.Errorf("GrappleEndCause(%d).Valid() = %v, want %v", int(c), c.Valid(), want)
		}
	}
}
