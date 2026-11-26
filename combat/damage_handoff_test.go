package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestDamageHandoffDelivery(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		delivery                combat.AttackDelivery
		choice                  combat.KnockOutChoice
		want                    combat.DamageDelivery
		ok                      bool
	}{
		{"melee", resSpec + "FR-037, 4.6.1", "happy", combat.AttackDeliveryMelee, combat.NoKnockOut, combat.MeleeAttackWithoutKnockOut, true},
		{"melee knock out", resSpec + "FR-037, 4.6.1", "unhappy (rule interaction)", combat.AttackDeliveryMelee, combat.KnockOut, combat.MeleeAttackKnockOut, true},
		{"ranged", resSpec + "FR-037, 4.6.1", "unhappy (invalid input)", combat.AttackDeliveryRanged, combat.NoKnockOut, combat.OtherSource, true},
		{"ranged knock out", resSpec + "FR-037, FR-136", "unhappy (invalid input)", combat.AttackDeliveryRanged, combat.KnockOut, 0, false},
		{"undefined delivery", resSpec + "FR-037", "unhappy (invalid input)", combat.AttackDelivery(0), combat.NoKnockOut, 0, false},
		{"undefined choice", resSpec + "FR-037", "unhappy (invalid input)", combat.AttackDeliveryMelee, combat.KnockOutChoice(5), 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.DamageHandoff{}).Delivery(c.delivery, c.choice)
			if c.ok != (err == nil) || got != c.want || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("= %v, %v, want %v, ok=%v [%s]", got, err, c.want, c.ok, c.requirement)
			}
		})
	}
}
