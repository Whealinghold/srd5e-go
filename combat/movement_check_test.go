package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: per case
// Path: happy.
func TestRequirementFor(t *testing.T) {
	t.Parallel()
	const r = `doc/specs/movement-lifting-and-flight.md FR-031, FR-032 and cases 4.4a; SRD 5.2.1, "Long Jump", p. 185; "Climbing", p. 179; "Swimming", p. 189`
	cases := []struct {
		kind    combat.MovementCheckKind
		name    string
		skill   content.Skill
		ability ability.Ability
		dc      d20.DifficultyClass
		failure combat.MovementCheckFailureEffect
	}{
		{combat.MovementCheckKindLongJumpLanding, "long_jump_landing", content.SkillAcrobatics, ability.Dexterity, 10, combat.MovementCheckFailureEffectBecomesProne},
		{combat.MovementCheckKindLongJumpObstacle, "long_jump_obstacle", content.SkillAthletics, ability.Strength, 10, combat.MovementCheckFailureEffectHitsObstacle},
		{combat.MovementCheckKindClimbingHardSurface, "climbing_hard_surface", content.SkillAthletics, ability.Strength, 15, combat.MovementCheckFailureEffectNotSpecified},
		{combat.MovementCheckKindSwimmingRoughWater, "swimming_rough_water", content.SkillAthletics, ability.Strength, 15, combat.MovementCheckFailureEffectNotSpecified},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combat.RequirementFor(c.kind)
			if got.IsZero() || got.Kind() != c.kind || got.Kind().String() != c.name || got.Skill() != c.skill ||
				got.Ability() != c.ability || got.Ability() != got.Skill().Ability() || got.Target() != c.dc || got.OnFailure() != c.failure {
				t.Errorf("RequirementFor(%v) = skill %v ability %v DC %v failure %v; want %v %v %v %v [%s]",
					c.kind, got.Skill(), got.Ability(), got.Target(), got.OnFailure(), c.skill, c.ability, c.dc, c.failure, r)
			}
		})
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-035 - the four requirements are equal to themselves on every access and pairwise unequal.
// Path: unhappy (immutability).
func TestRequirementsAreEqualToThemselvesAndDistinct(t *testing.T) {
	t.Parallel()
	for a := combat.MovementCheckKindLongJumpLanding; a <= combat.MovementCheckKindSwimmingRoughWater; a++ {
		for b := combat.MovementCheckKindLongJumpLanding; b <= combat.MovementCheckKindSwimmingRoughWater; b++ {
			if same := combat.RequirementFor(a) == combat.RequirementFor(b); same != (a == b) {
				t.Errorf("RequirementFor(%v) == RequirementFor(%v) is %v", a, b, same)
			}
		}
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-031 - an invalid kind gives the zero requirement, which has no skill, ability, target or effect.
// Path: unhappy (invalid input).
func TestRequirementForInvalidKindIsZero(t *testing.T) {
	t.Parallel()
	for _, k := range []combat.MovementCheckKind{0, combat.MovementCheckKindSwimmingRoughWater + 1, -1} {
		got := combat.RequirementFor(k)
		if !got.IsZero() || got.Skill() != 0 || got.Target() != 0 || got.OnFailure() != 0 || got.Kind().Valid() || got.Kind().String() != "unknown" {
			t.Errorf("RequirementFor(%d) = %#v, want the zero requirement", k, got)
		}
	}
	if combat.MovementCheckFailureEffect(0).String() != "unknown" || combat.MovementCheckFailureEffectHitsObstacle.String() != "hits_obstacle" ||
		combat.MovementCheckFailureEffectBecomesProne.String() != "becomes_prone" || combat.MovementCheckFailureEffectNotSpecified.String() != "not_specified" {
		t.Error("failure effect names are wrong")
	}
}
