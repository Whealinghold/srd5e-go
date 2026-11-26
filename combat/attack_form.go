package combat

// AttackDelivery says how an attack reaches its target: Melee within reach,
// Ranged within a range (SRD 5.2.1, "Melee Attacks", "Ranged Attacks", p. 15).
// The zero value is not a delivery.
type AttackDelivery int

// The two deliveries.
const (
	AttackDeliveryMelee AttackDelivery = iota + 1
	AttackDeliveryRanged
)

// String returns "melee", "ranged" or "unknown".
func (d AttackDelivery) String() string { return enumName([]string{"melee", "ranged"}, int(d)) }

// AttackSource says what makes the attack roll: a weapon, a spell or the body
// (SRD 5.2.1, "Spell Attack", p. 188; "Unarmed Strike", p. 190; "Weapon Attack",
// p. 191). The zero value is not a source.
type AttackSource int

// The three sources.
const (
	AttackSourceWeapon AttackSource = iota + 1
	AttackSourceSpell
	AttackSourceUnarmedStrike
)

// String returns "weapon", "spell", "unarmed_strike" or "unknown".
func (s AttackSource) String() string {
	return enumName([]string{"weapon", "spell", "unarmed_strike"}, int(s))
}

// AttackForm is one of the five legal pairs of delivery and source (SRD 5.2.1,
// "Making an Attack", p. 14): MeleeWeaponAttack, RangedWeaponAttack,
// MeleeSpellAttack, RangedSpellAttack and UnarmedStrikeAttack, which is always
// melee. It carries no default ability. The zero value is not a form.
type AttackForm int

// The five forms.
const (
	MeleeWeaponAttack AttackForm = iota + 1
	RangedWeaponAttack
	MeleeSpellAttack
	RangedSpellAttack
	UnarmedStrikeAttack
)

// Valid reports whether f is one of the five forms.
func (f AttackForm) Valid() bool { return f >= MeleeWeaponAttack && f <= UnarmedStrikeAttack }

// Delivery returns the delivery of f, or the zero AttackDelivery for an invalid form.
func (f AttackForm) Delivery() AttackDelivery {
	switch f {
	case MeleeWeaponAttack, MeleeSpellAttack, UnarmedStrikeAttack:
		return AttackDeliveryMelee
	case RangedWeaponAttack, RangedSpellAttack:
		return AttackDeliveryRanged
	}
	return 0
}

// Source returns the source of f, or the zero AttackSource for an invalid form.
func (f AttackForm) Source() AttackSource {
	switch f {
	case MeleeWeaponAttack, RangedWeaponAttack:
		return AttackSourceWeapon
	case MeleeSpellAttack, RangedSpellAttack:
		return AttackSourceSpell
	case UnarmedStrikeAttack:
		return AttackSourceUnarmedStrike
	}
	return 0
}

// String returns the srd5e.combat.attack.form value: "melee_weapon",
// "ranged_weapon", "melee_spell", "ranged_spell", "unarmed_strike" or "unknown".
func (f AttackForm) String() string {
	return enumName([]string{"melee_weapon", "ranged_weapon", "melee_spell", "ranged_spell", "unarmed_strike"}, int(f))
}
