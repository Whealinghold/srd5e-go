package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var lightR combat.LightRules

const specHS = "doc/specs/hiding-searching-and-senses.md "

// Requirement: per case
// Path: per case.
func TestObscurementOfLight(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		level                   content.LightLevel
		want                    content.Obscurement
	}{
		{"bright", specHS + `FR-003, SRD 5.2.1, "Bright Light", p. 178`, "happy", content.LightBright, content.ObscurementNone},
		{"dim", specHS + `FR-004, "Dim Light", p. 181`, "happy", content.LightDim, content.ObscurementLightly},
		{"darkness", specHS + `FR-005, "Darkness", p. 180`, "happy", content.LightDarkness, content.ObscurementHeavily},
		{"undefined level", specHS + "FR-003 (invalid input)", "unhappy (invalid input)", 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := lightR.ObscurementOf(c.level); got != c.want {
				t.Errorf("ObscurementOf(%v) = %v, want %v [%s]", c.level, got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestSpaceVisibility(t *testing.T) {
	t.Parallel()
	const o = `SRD 5.2.1, "Obscured Areas", p. 11`
	cases := []struct {
		name, requirement, path string
		light                   content.LightLevel
		kind                    content.DarknessKind
		other                   content.Obscurement
		want                    content.Obscurement
		wantReason              combat.SpaceVisibilityReason
	}{
		{"bright, nothing else", specHS + "FR-003, FR-007, " + o, "happy", content.LightBright, 0, 0, content.ObscurementNone, 0},
		{"dim", specHS + "FR-004", "happy", content.LightDim, 0, content.ObscurementNone, content.ObscurementLightly, 0},
		{"darkness", specHS + "FR-005", "happy", content.LightDarkness, content.DarknessNonmagical, 0, content.ObscurementHeavily, 0},
		{"bright with heavy fog", specHS + "FR-007", "unhappy (rule interaction)", content.LightBright, 0, content.ObscurementHeavily, content.ObscurementHeavily, 0},
		{"darkness with patchy fog", specHS + "FR-007", "unhappy (rule interaction)", content.LightDarkness, 0, content.ObscurementLightly, content.ObscurementHeavily, 0},
		{"dim with patchy fog", specHS + "FR-007", "unhappy (rule interaction)", content.LightDim, 0, content.ObscurementLightly, content.ObscurementLightly, 0},
		{"magical darkness", specHS + "FR-006", "happy", content.LightDarkness, content.DarknessMagical, 0, content.ObscurementHeavily, 0},
		{"magical flag on Dim Light", specHS + `FR-006, "Truesight", p. 190`, "unhappy (invalid input)", content.LightDim, content.DarknessMagical, 0, 0, combat.SpaceVisibilityReasonMagicalWithoutDarkness},
		{"magical flag on Bright Light", specHS + "FR-006", "unhappy (invalid input)", content.LightBright, content.DarknessMagical, 0, 0, combat.SpaceVisibilityReasonMagicalWithoutDarkness},
		{"undefined light", specHS + "FR-097", "unhappy (invalid input)", 0, 0, 0, 0, combat.SpaceVisibilityReasonInvalidLight},
		{"undefined kind", specHS + "FR-097", "unhappy (invalid input)", content.LightDarkness, 99, 0, 0, combat.SpaceVisibilityReasonInvalidDarknessKind},
		{"undefined obscurement", specHS + "FR-097", "unhappy (invalid input)", content.LightDarkness, 0, 99, 0, combat.SpaceVisibilityReasonInvalidObscurement},
		{"negative kind", specHS + "FR-097", "unhappy (invalid input)", content.LightDarkness, -1, 0, 0, combat.SpaceVisibilityReasonInvalidDarknessKind},
		{"negative obscurement", specHS + "FR-097", "unhappy (invalid input)", content.LightDarkness, 0, -1, 0, combat.SpaceVisibilityReasonInvalidObscurement},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			v, err := combat.NewSpaceVisibility(c.light, c.kind, c.other)
			if c.wantReason != 0 {
				var e *combat.SpaceVisibilityError
				if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &e) || e.Reason != c.wantReason || !errors.Is(err, &combat.SpaceVisibilityError{Reason: c.wantReason}) {
					t.Errorf("error = %v, want reason %v [%s]", err, c.wantReason, c.requirement)
				}
				if v.Valid() || v.Obscurement() != 0 {
					t.Error("rejected value must stay the invalid zero value")
				}
				return
			}
			if err != nil || !v.Valid() || v.Obscurement() != c.want {
				t.Errorf("NewSpaceVisibility = %v, %v, want obscurement %v [%s]", v.Obscurement(), err, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-006, FR-097 - the defaults (a zero kind is nonmagical, a zero other
// obscurement is none), the accessors, MustSpaceVisibility, and the reason names.
// Path: unhappy (invalid input).
func TestSpaceVisibilityAccessorsAndMust(t *testing.T) {
	t.Parallel()
	v := combat.MustSpaceVisibility(content.LightDarkness, 0, 0)
	if v.Light() != content.LightDarkness || v.DarknessKind() != content.DarknessNonmagical || v.OtherObscurement() != content.ObscurementNone {
		t.Errorf("defaults = %v %v %v", v.Light(), v.DarknessKind(), v.OtherObscurement())
	}
	if (combat.SpaceVisibility{}).Valid() {
		t.Error("the zero SpaceVisibility is valid")
	}
	if combat.MustSpaceVisibility(content.LightDim, 0, 0) == combat.MustSpaceVisibility(content.LightDim, 0, content.ObscurementLightly) {
		t.Error("equality ignores the other obscurement")
	}
	defer func() {
		if r := recover(); r == nil || !errors.Is(r.(error), srderr.ErrInvalidInput) {
			t.Errorf("MustSpaceVisibility panic = %v, want an ErrInvalidInput error", r)
		}
	}()
	combat.MustSpaceVisibility(content.LightDim, content.DarknessMagical, 0)
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-008 to FR-013 (the matrix of obscurement by check kind) - Disadvantage on
// Perception to see in Lightly Obscured, an automatic failure for a check requiring sight in Heavily Obscured, nothing
// otherwise (SRD 5.2.1, "Lightly Obscured", p. 184; "Heavily Obscured", p. 182).
// Path: per case.
func TestSightCheckEffectMatrix(t *testing.T) {
	t.Parallel()
	none, lightly, heavily := content.ObscurementNone, content.ObscurementLightly, content.ObscurementHeavily
	per, other, not := combat.SightCheckKindPerceptionToSee, combat.SightCheckKindOtherCheckRequiringSight, combat.SightCheckKindCheckNotRequiringSight
	un, dis, fail := combat.SightCheckEffectUnaffected, combat.SightCheckEffectDisadvantage, combat.SightCheckEffectAutomaticFailure
	cases := []struct {
		name, requirement, path string
		o                       content.Obscurement
		kind                    combat.SightCheckKind
		want                    combat.SightCheckEffect
	}{
		{"lightly, perception", "FR-008", "happy", lightly, per, dis},
		{"lightly, other requiring sight", "FR-009", "unhappy (boundary)", lightly, other, un},
		{"lightly, not requiring sight", "FR-010", "unhappy (boundary)", lightly, not, un},
		{"heavily, perception", "FR-011", "happy", heavily, per, fail},
		{"heavily, other requiring sight", "FR-011", "happy", heavily, other, fail},
		{"heavily, not requiring sight", "FR-012", "unhappy (boundary)", heavily, not, un},
		{"none, perception", "FR-013", "unhappy (empty and minimal)", none, per, un},
		{"none, other", "FR-013", "unhappy (empty and minimal)", none, other, un},
		{"none, not", "FR-013", "unhappy (empty and minimal)", none, not, un},
		{"undefined obscurement", "FR-013", "unhappy (invalid input)", 0, per, un},
		{"undefined kind", "FR-013", "unhappy (invalid input)", heavily, 0, un},
		{"undefined kind, lightly", "FR-013", "unhappy (invalid input)", lightly, 0, un},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := lightR.SightCheckEffect(c.o, c.kind); got != c.want {
				t.Errorf("SightCheckEffect(%v, %v) = %v, want %v [%s]", c.o, c.kind, got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-014, FR-015, FR-100 (SRD 5.2.1, "Heavily Obscured", p. 182; "Passive
// Perception", p. 186) - AttemptConditions is {blinded} for heavily and empty otherwise, Circumstances is [Disadvantage]
// for disadvantage and empty otherwise, both as fresh slices.
// Path: unhappy (rule interaction).
func TestAttemptConditionsAndCircumstances(t *testing.T) {
	t.Parallel()
	if got := lightR.AttemptConditions(content.ObscurementHeavily); !slices.Equal(got, []content.ConditionKind{content.ConditionBlinded}) {
		t.Errorf("AttemptConditions(heavily) = %v", got)
	}
	for _, o := range []content.Obscurement{content.ObscurementNone, content.ObscurementLightly, 0} {
		if got := lightR.AttemptConditions(o); got == nil || len(got) != 0 {
			t.Errorf("AttemptConditions(%v) = %#v, want empty", o, got)
		}
	}
	lightR.AttemptConditions(content.ObscurementHeavily)[0] = 0
	if lightR.AttemptConditions(content.ObscurementHeavily)[0] != content.ConditionBlinded {
		t.Error("AttemptConditions is not a fresh slice [FR-100]")
	}
	if got := lightR.Circumstances(combat.SightCheckEffectDisadvantage); len(got) != 1 || got[0] != (modifier.Disadvantage{}) {
		t.Errorf("Circumstances(disadvantage) = %v", got)
	}
	for _, e := range []combat.SightCheckEffect{combat.SightCheckEffectUnaffected, combat.SightCheckEffectAutomaticFailure, 0} {
		if got := lightR.Circumstances(e); got == nil || len(got) != 0 {
			t.Errorf("Circumstances(%v) = %#v, want empty", e, got)
		}
	}
	lightR.Circumstances(combat.SightCheckEffectDisadvantage)[0] = nil
	if lightR.Circumstances(combat.SightCheckEffectDisadvantage)[0] == nil {
		t.Error("Circumstances is not a fresh slice [FR-100]")
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md section 5 - every enum prints its snake_case name, "unknown" outside its range.
// Path: unhappy (boundary).
func TestLightEnumStrings(t *testing.T) {
	t.Parallel()
	got := []string{
		combat.SightCheckKindPerceptionToSee.String(), combat.SightCheckKindCheckNotRequiringSight.String(), combat.SightCheckKind(0).String(),
		combat.SightCheckEffectUnaffected.String(), combat.SightCheckEffectAutomaticFailure.String(), combat.SightCheckEffect(9).String(),
		combat.SpaceVisibilityReasonInvalidLight.String(), combat.SpaceVisibilityReasonMagicalWithoutDarkness.String(), combat.SpaceVisibilityReason(0).String(),
	}
	want := []string{"perception_to_see", "check_not_requiring_sight", "unknown", "unaffected", "automatic_failure", "unknown",
		"invalid_light", "magical_without_darkness", "unknown"}
	if !slices.Equal(got, want) {
		t.Errorf("strings = %v, want %v", got, want)
	}
}
