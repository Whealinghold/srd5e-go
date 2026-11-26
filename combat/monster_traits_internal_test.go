package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/monster-traits.md FR-045, FR-047 - only Monster Death and Massive Damage are deferred, every other cause of a Dead creature with a drop is not (forged transitions: the public operations never pair a drop with those causes).
// Path: unhappy (rule interaction).
func TestDeferRegenerationDeathPerCause(t *testing.T) {
	t.Parallel()
	trait, err := NewRegenerationTrait(LimitedUsePartRef{MonsterPartTraits, 0}, 15, []content.DamageType{content.DamageFire})
	if err != nil {
		t.Fatal(err)
	}
	id := CombatantID("troll")
	for cause := MonsterDeath; cause <= Exhaustion; cause++ {
		status, serr := RestoreHitPointStatus(HitPoints{0, 84, 0}, Dead{cause}, DiesInstantly, NoAffinities())
		if serr != nil {
			t.Fatal(serr)
		}
		tr := HitPointTransitionApplied{status: status, events: []HitPointEvent{DroppedToZeroHitPoints{hitPointEventBase{id}}}}
		_, deferred := DeferRegenerationDeath(trait, id, tr).(RegenerationDeferred)
		if want := cause == MonsterDeath || cause == MassiveDamage; deferred != want {
			t.Errorf("cause %v: deferred = %v, want %v", cause, deferred, want)
		}
	}
}
