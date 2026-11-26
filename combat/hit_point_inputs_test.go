package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-107, FR-108 - the closed enums name their members, the
// zero value is invalid, and the names are the srd5e.combat attribute values (refusal reason, death save result).
// Path: per case.
func TestHitPointEnums(t *testing.T) {
	t.Parallel()
	strs := []struct {
		got, want string
	}{
		{combat.DiesInstantly.String(), "dies_instantly"}, {combat.FallsUnconscious.String(), "falls_unconscious"}, {combat.ZeroHitPointsRule(0).String(), "unknown"},
		{combat.CreatureIsDead.String(), "creature_is_dead"}, {combat.NotDying.String(), "not_dying"}, {combat.NotStable.String(), "not_stable"}, {combat.HitPointRejection(0).String(), "unknown"},
		{combat.DeathSavingThrowSuccess.String(), "success"}, {combat.DeathSavingThrowFailure.String(), "failure"},
		{combat.DeathSavingThrowTwoFailures.String(), "critical_failure"}, {combat.DeathSavingThrowRegainedHitPoint.String(), "critical_success"}, {combat.DeathSavingThrowOutcome(0).String(), "unknown"},
		{combat.MonsterDeath.String(), "monster_death"}, {combat.HitPointMaximumZero.String(), "hit_point_maximum_zero"}, {combat.MassiveDamage.String(), "massive_damage"},
		{combat.DeathSavingThrowFailures.String(), "death_saving_throw_failures"}, {combat.Exhaustion.String(), "exhaustion"}, {combat.DeathCause(0).String(), "unknown"},
	}
	for _, c := range strs {
		if c.got != c.want {
			t.Errorf("String() = %q, want %q", c.got, c.want)
		}
	}
	if !combat.DiesInstantly.Valid() || !combat.FallsUnconscious.Valid() || combat.ZeroHitPointsRule(0).Valid() || combat.ZeroHitPointsRule(3).Valid() {
		t.Errorf("ZeroHitPointsRule validity wrong")
	}
	if combat.DeathCause(0).Valid() || combat.DeathCause(6).Valid() || !combat.Exhaustion.Valid() {
		t.Errorf("DeathCause validity wrong")
	}
}
