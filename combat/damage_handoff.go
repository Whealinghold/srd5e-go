package combat

// DamageHandoff maps what an attack carries to what the hit point rules take (combat-action-resolution.md
// FR-036, FR-037). It never sees an attack result type. It is pure and stateless; the zero value is usable.
type DamageHandoff struct{}

// Delivery returns the DamageDelivery of an attack: a melee attack with KnockOut is MeleeAttackKnockOut,
// a melee attack without it MeleeAttackWithoutKnockOut, a ranged attack OtherSource (SRD 5.2.1, "Knocking
// Out a Creature", p. 17). It returns an error matching srderr.ErrInvalidInput for KnockOut with a ranged
// attack, a delivery or a choice that is not defined.
func (DamageHandoff) Delivery(delivery AttackDelivery, choice KnockOutChoice) (DamageDelivery, error) {
	if !choice.Valid() {
		return 0, invalidInput("a knock out choice is not defined")
	}
	switch delivery {
	case AttackDeliveryMelee:
		if choice == KnockOut {
			return MeleeAttackKnockOut, nil
		}
		return MeleeAttackWithoutKnockOut, nil
	case AttackDeliveryRanged:
		if choice == KnockOut {
			return 0, invalidInput("only a melee attack knocks a creature out")
		}
		return OtherSource, nil
	}
	return 0, invalidInput("an attack delivery is not defined")
}
