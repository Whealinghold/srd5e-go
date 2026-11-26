package combat

// WeaponChangeKind says whether a weapon change equips or unequips a weapon (SRD 5.2.1,
// "Equipping and Unequipping Weapons", p. 177). The zero value is not a kind.
type WeaponChangeKind int

// The two kinds of weapon change.
const (
	WeaponChangeKindEquip WeaponChangeKind = iota + 1
	WeaponChangeKindUnequip
)

// String returns "equip", "unequip" or "unknown".
func (k WeaponChangeKind) String() string { return enumName([]string{"equip", "unequip"}, int(k)) }

// AttackActionWeaponChange is one weapon change an Attack action may make: "Equipping a
// weapon includes drawing it from a sheath or picking it up. Unequipping a weapon
// includes sheathing, stowing, or dropping it" (SRD 5.2.1, p. 177). It is distinct from
// WeaponWielding of the character context. The zero value is not a change. Comparable
// and safe for concurrent use.
type AttackActionWeaponChange int

// The seven weapon changes: three equip and four unequip.
const (
	WeaponChangeDrawFromSheath AttackActionWeaponChange = iota + 1
	WeaponChangePickUp
	WeaponChangeEquipOther
	WeaponChangeSheathe
	WeaponChangeStow
	WeaponChangeDrop
	WeaponChangeUnequipOther
)

// Valid reports whether c is one of the seven changes.
func (c AttackActionWeaponChange) Valid() bool {
	return c >= WeaponChangeDrawFromSheath && c <= WeaponChangeUnequipOther
}

// Change returns WeaponChangeKindEquip for drawing, picking up and any other equip,
// WeaponChangeKindUnequip for sheathing, stowing, dropping and any other unequip, and
// the zero kind for an invalid change.
func (c AttackActionWeaponChange) Change() WeaponChangeKind {
	switch c {
	case WeaponChangeDrawFromSheath, WeaponChangePickUp, WeaponChangeEquipOther:
		return WeaponChangeKindEquip
	case WeaponChangeSheathe, WeaponChangeStow, WeaponChangeDrop, WeaponChangeUnequipOther:
		return WeaponChangeKindUnequip
	}
	return 0
}

// String returns the snake_case name of c, or "unknown".
func (c AttackActionWeaponChange) String() string {
	return enumName([]string{"draw_from_sheath", "pick_up", "equip_other", "sheathe", "stow", "drop", "unequip_other"}, int(c))
}

// AttackActionWeaponChangeTiming says whether a weapon change comes before or after the
// attack it belongs to. The zero value is not a timing.
type AttackActionWeaponChangeTiming int

// The two timings.
const (
	WeaponChangeBeforeAttack AttackActionWeaponChangeTiming = iota + 1
	WeaponChangeAfterAttack
)

// Valid reports whether t is one of the two timings.
func (t AttackActionWeaponChangeTiming) Valid() bool {
	return t >= WeaponChangeBeforeAttack && t <= WeaponChangeAfterAttack
}

// String returns "before_attack", "after_attack" or "unknown".
func (t AttackActionWeaponChangeTiming) String() string {
	return enumName([]string{"before_attack", "after_attack"}, int(t))
}

// WeaponChangeRefusal says why HandleWeapon refused a change. The zero value is not a
// refusal.
type WeaponChangeRefusal int

// The three reasons a weapon change is refused.
const (
	// WeaponChangeRefusalNoAttackMade: a change after an attack, and none has been made.
	WeaponChangeRefusalNoAttackMade WeaponChangeRefusal = iota + 1
	// WeaponChangeRefusalNoAttackRemaining: a change before an attack, and none can be made.
	WeaponChangeRefusalNoAttackRemaining
	// WeaponChangeRefusalAlreadyUsedForThisAttack: the attack's slot already holds a change.
	WeaponChangeRefusalAlreadyUsedForThisAttack
)

// String returns the snake_case name of r, or "unknown".
func (r WeaponChangeRefusal) String() string {
	return enumName([]string{"no_attack_made", "no_attack_remaining", "already_used_for_this_attack"}, int(r))
}

// AttackRefusal says why an attack was refused. The zero value is not a refusal.
type AttackRefusal int

// AttackRefusalLimitReached: the attack would exceed the limit of the allowance.
const AttackRefusalLimitReached AttackRefusal = 1

// String returns "attack_limit_reached" or "unknown".
func (r AttackRefusal) String() string { return enumName([]string{"attack_limit_reached"}, int(r)) }
