package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// The ability checks the movement rules name (SRD 5.2.1, "Long Jump", p. 185; "Climbing",
// p. 179; "Swimming", p. 189). They are described, never rolled: the host resolves one
// through the d20 context when its GM invokes the option.

// MovementCheckKind names one of the four checks. The zero MovementCheckKind is invalid.
type MovementCheckKind int

// The four checks.
const (
	// MovementCheckKindLongJumpLanding: landing a Long Jump in Difficult Terrain.
	MovementCheckKindLongJumpLanding MovementCheckKind = iota + 1
	// MovementCheckKindLongJumpObstacle: clearing a low obstacle in a Long Jump.
	MovementCheckKindLongJumpObstacle
	// MovementCheckKindClimbingHardSurface: climbing a slippery surface or one with few handholds.
	MovementCheckKindClimbingHardSurface
	// MovementCheckKindSwimmingRoughWater: swimming in rough water.
	MovementCheckKindSwimmingRoughWater
)

// Valid reports whether k is one of the four kinds.
func (k MovementCheckKind) Valid() bool {
	return k >= MovementCheckKindLongJumpLanding && k <= MovementCheckKindSwimmingRoughWater
}

// String returns "long_jump_landing", "long_jump_obstacle", "climbing_hard_surface",
// "swimming_rough_water" or "unknown".
func (k MovementCheckKind) String() string {
	return enumName([]string{"long_jump_landing", "long_jump_obstacle", "climbing_hard_surface", "swimming_rough_water"}, int(k))
}

// MovementCheckFailureEffect is what failing a check means. The zero value is invalid.
type MovementCheckFailureEffect int

// The three meanings.
const (
	// MovementCheckFailureEffectBecomesProne: the creature has the Prone condition.
	MovementCheckFailureEffectBecomesProne MovementCheckFailureEffect = iota + 1
	// MovementCheckFailureEffectHitsObstacle: the creature hits the obstacle.
	MovementCheckFailureEffectHitsObstacle
	// MovementCheckFailureEffectNotSpecified: the SRD states no consequence.
	MovementCheckFailureEffectNotSpecified
)

// String returns "becomes_prone", "hits_obstacle", "not_specified" or "unknown".
func (e MovementCheckFailureEffect) String() string {
	return enumName([]string{"becomes_prone", "hits_obstacle", "not_specified"}, int(e))
}

// MovementCheckRequirement is an ability check the movement rules name: its skill (and so
// its ability), its Difficulty Class and what failing means. It carries no roll, total or
// outcome. Only RequirementFor makes one, so there are four values, each equal to itself
// and unequal to the others. The zero value is invalid (IsZero). Comparable with ==.
type MovementCheckRequirement struct{ kind MovementCheckKind }

// RequirementFor returns the requirement of kind, or the zero requirement for an invalid kind.
func RequirementFor(kind MovementCheckKind) MovementCheckRequirement {
	if !kind.Valid() {
		return MovementCheckRequirement{}
	}
	return MovementCheckRequirement{kind}
}

// IsZero reports whether r is the invalid zero requirement.
func (r MovementCheckRequirement) IsZero() bool { return r.kind == 0 }

// Kind returns which check r is.
func (r MovementCheckRequirement) Kind() MovementCheckKind { return r.kind }

// Skill returns the skill of the check, Dexterity (Acrobatics) for a landing and Strength
// (Athletics) for the others; the zero requirement has the zero Skill.
func (r MovementCheckRequirement) Skill() content.Skill {
	switch r.kind {
	case MovementCheckKindLongJumpLanding:
		return content.SkillAcrobatics
	case MovementCheckKindLongJumpObstacle, MovementCheckKindClimbingHardSurface, MovementCheckKindSwimmingRoughWater:
		return content.SkillAthletics
	}
	return 0
}

// Ability returns the ability of the check, always the ability of Skill.
func (r MovementCheckRequirement) Ability() ability.Ability { return r.Skill().Ability() }

// Target returns the Difficulty Class: 10 for the Long Jump checks and 15 for climbing and
// swimming. The zero requirement has the zero class.
func (r MovementCheckRequirement) Target() d20.DifficultyClass {
	switch r.kind {
	case MovementCheckKindLongJumpLanding, MovementCheckKindLongJumpObstacle:
		return d20.DCEasy
	case MovementCheckKindClimbingHardSurface, MovementCheckKindSwimmingRoughWater:
		return d20.DCMedium
	}
	return 0
}

// OnFailure returns what failing the check means; the zero requirement has the zero effect.
func (r MovementCheckRequirement) OnFailure() MovementCheckFailureEffect {
	switch r.kind {
	case MovementCheckKindLongJumpLanding:
		return MovementCheckFailureEffectBecomesProne
	case MovementCheckKindLongJumpObstacle:
		return MovementCheckFailureEffectHitsObstacle
	case MovementCheckKindClimbingHardSurface, MovementCheckKindSwimmingRoughWater:
		return MovementCheckFailureEffectNotSpecified
	}
	return 0
}
