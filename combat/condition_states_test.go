package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/conditions.md section 5 (named states) - each state is a typed enum whose String is its
// snake_case name, the zero value and a value past the last print "unknown".
// Path: unhappy (invalid input).
func TestConditionStatesString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		got  []fmt.Stringer
		want []string
	}{
		{"Sense", []fmt.Stringer{combat.SenseSight, combat.SenseHearing, combat.Sense(0), combat.Sense(3)}, []string{"sight", "hearing", "unknown", "unknown"}},
		{"CheckRequirement", []fmt.Stringer{combat.CheckRequirementSight, combat.CheckRequirementHearing, combat.CheckRequirementNone, combat.CheckRequirement(0)}, []string{"sight", "hearing", "none", "unknown"}},
		{"SightRelation", []fmt.Stringer{combat.SightRelationCanSee, combat.SightRelationCannotSee, combat.SightRelation(0)}, []string{"can_see", "cannot_see", "unknown"}},
		{"LineOfSight", []fmt.Stringer{combat.LineOfSightClear, combat.LineOfSightBlocked, combat.LineOfSight(0)}, []string{"clear", "blocked", "unknown"}},
		{"EffectDamage", []fmt.Stringer{combat.EffectDamageDamaging, combat.EffectDamageNotDamaging, combat.EffectDamage(0)}, []string{"damaging", "not_damaging", "unknown"}},
		{"ConditionPermission", []fmt.Stringer{combat.ConditionPermissionAllowed, combat.ConditionPermissionForbidden, combat.ConditionPermission(0)}, []string{"allowed", "forbidden", "unknown"}},
		{"SavingThrowEffect", []fmt.Stringer{combat.SavingThrowEffectAutomaticFailure, combat.SavingThrowEffectDisadvantage, combat.SavingThrowEffectUnaffected, combat.SavingThrowEffect(0)}, []string{"automatic_failure", "disadvantage", "unaffected", "unknown"}},
		{"AbilityCheckEffect", []fmt.Stringer{combat.AbilityCheckEffectAutomaticFailure, combat.AbilityCheckEffectUnaffected, combat.AbilityCheckEffect(0)}, []string{"automatic_failure", "unaffected", "unknown"}},
		{"AutomaticCritical", []fmt.Stringer{combat.AutomaticCriticalAutomatic, combat.AutomaticCriticalNotAutomatic, combat.AutomaticCritical(0)}, []string{"automatic", "not_automatic", "unknown"}},
		{"SpeedIncrease", []fmt.Stringer{combat.SpeedIncreaseAllowed, combat.SpeedIncreaseForbidden, combat.SpeedIncrease(0)}, []string{"allowed", "forbidden", "unknown"}},
		{"ConditionConcentration", []fmt.Stringer{combat.ConditionConcentrationIntact, combat.ConditionConcentrationBroken, combat.ConditionConcentration(0)}, []string{"intact", "broken", "unknown"}},
		{"Speech", []fmt.Stringer{combat.SpeechAble, combat.SpeechUnable, combat.Speech(0)}, []string{"able", "unable", "unknown"}},
		{"ConditionActionAvailability", []fmt.Stringer{combat.ConditionActionAvailabilityAvailable, combat.ConditionActionAvailabilityNone, combat.ConditionActionAvailability(0)}, []string{"available", "none", "unknown"}},
		{"HeldItems", []fmt.Stringer{combat.HeldItemsKept, combat.HeldItemsDropped, combat.HeldItems(0)}, []string{"kept", "dropped", "unknown"}},
		{"Awareness", []fmt.Stringer{combat.AwarenessAware, combat.AwarenessUnaware, combat.Awareness(0)}, []string{"aware", "unaware", "unknown"}},
		{"AllDamageResistance", []fmt.Stringer{combat.AllDamageResistanceNone, combat.AllDamageResistanceAll, combat.AllDamageResistance(0)}, []string{"none", "all", "unknown"}},
		{"EffectCondition", []fmt.Stringer{combat.EffectConditionAlways, combat.EffectConditionAttackerWithin5Feet, combat.EffectConditionAttackerBeyond5Feet,
			combat.EffectConditionSourceInLineOfSight, combat.EffectConditionTargetIsNotGrappler, combat.EffectConditionUnlessSeen, combat.EffectCondition(0), combat.EffectCondition(7)},
			[]string{"always", "attacker_within_5_feet", "attacker_beyond_5_feet", "source_in_line_of_sight", "target_is_not_grappler", "unless_seen", "unknown", "unknown"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for i, s := range c.got {
				if s.String() != c.want[i] {
					t.Errorf("%s value %d prints %q, want %q", c.name, i, s.String(), c.want[i])
				}
			}
		})
	}
}
