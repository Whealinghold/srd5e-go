package combat

// AttackActionEvent is the closed family of events the Attack action records. Each is an
// immutable value comparable with ==, about the one combatant making the action. Variants:
// AttackMadeInAttackAction, AttackReplacedInAttackAction, WeaponChangedInAttackAction. Each embeds attackActionEventBase.
type AttackActionEvent interface {
	Event
	attackActionEvent()
}

// attackActionEventBase carries the combatant every Attack action event is about and the
// markers that put it in the Event and AttackActionEvent families.
type attackActionEventBase struct{ combatant CombatantID }

func (attackActionEventBase) event()             {}
func (attackActionEventBase) attackActionEvent() {}

// Combatants returns a new slice holding the one combatant.
func (b attackActionEventBase) Combatants() []CombatantID { return []CombatantID{b.combatant} }

func newAttackActionBase(c CombatantID, number int) (attackActionEventBase, error) {
	if c.IsZero() || number < 1 {
		return attackActionEventBase{}, invalidInput("attack action event needs a combatant and a 1-based number")
	}
	return attackActionEventBase{c}, nil
}

// AttackMadeInAttackAction: an attack was made with a means as part of an Attack action.
// Its Kind is attack_made_in_attack_action.
type AttackMadeInAttackAction struct {
	attackActionEventBase
	means  AttackMeans
	number int
}

// NewAttackMadeInAttackAction returns the event. It returns an error matching
// srderr.ErrInvalidInput for a blank combatant, an invalid means or a number below 1.
func NewAttackMadeInAttackAction(c CombatantID, means AttackMeans, number int) (AttackMadeInAttackAction, error) {
	if !means.Valid() {
		return AttackMadeInAttackAction{}, invalidInput("attack made needs a valid means")
	}
	b, err := newAttackActionBase(c, number)
	if err != nil {
		return AttackMadeInAttackAction{}, err
	}
	return AttackMadeInAttackAction{b, means, number}, nil
}

// Kind returns "attack_made_in_attack_action".
func (AttackMadeInAttackAction) Kind() string { return "attack_made_in_attack_action" }

// Means returns the means of the attack.
func (e AttackMadeInAttackAction) Means() AttackMeans { return e.means }

// Number returns the 1-based number of the attack in the action.
func (e AttackMadeInAttackAction) Number() int { return e.number }

// AttackReplacedInAttackAction: an attack of an Attack action was replaced or forgone. Its
// Kind is attack_replaced_in_attack_action.
type AttackReplacedInAttackAction struct {
	attackActionEventBase
	replacement AttackReplacement
	number      int
}

// NewAttackReplacedInAttackAction returns the event. It returns an error matching
// srderr.ErrInvalidInput for a blank combatant, an invalid replacement or a number below 1.
func NewAttackReplacedInAttackAction(c CombatantID, r AttackReplacement, number int) (AttackReplacedInAttackAction, error) {
	if !r.Valid() {
		return AttackReplacedInAttackAction{}, invalidInput("attack replaced needs a valid replacement")
	}
	b, err := newAttackActionBase(c, number)
	if err != nil {
		return AttackReplacedInAttackAction{}, err
	}
	return AttackReplacedInAttackAction{b, r, number}, nil
}

// Kind returns "attack_replaced_in_attack_action".
func (AttackReplacedInAttackAction) Kind() string { return "attack_replaced_in_attack_action" }

// Replacement returns the kind and source of the replacement.
func (e AttackReplacedInAttackAction) Replacement() AttackReplacement { return e.replacement }

// Number returns the 1-based number of the attack in the action.
func (e AttackReplacedInAttackAction) Number() int { return e.number }

// WeaponChangedInAttackAction: a weapon was equipped or unequipped as part of an Attack
// action, in the slot of an attack. Its Kind is weapon_changed_in_attack_action.
type WeaponChangedInAttackAction struct {
	attackActionEventBase
	change AttackActionWeaponChange
	timing AttackActionWeaponChangeTiming
	slot   int
}

// NewWeaponChangedInAttackAction returns the event. It returns an error matching
// srderr.ErrInvalidInput for a blank combatant, an invalid change or timing or a slot
// below 1.
func NewWeaponChangedInAttackAction(c CombatantID, change AttackActionWeaponChange,
	timing AttackActionWeaponChangeTiming, slot int) (WeaponChangedInAttackAction, error) {
	if !change.Valid() || !timing.Valid() {
		return WeaponChangedInAttackAction{}, invalidInput("weapon changed needs a valid change and timing")
	}
	b, err := newAttackActionBase(c, slot)
	if err != nil {
		return WeaponChangedInAttackAction{}, err
	}
	return WeaponChangedInAttackAction{b, change, timing, slot}, nil
}

// Kind returns "weapon_changed_in_attack_action".
func (WeaponChangedInAttackAction) Kind() string { return "weapon_changed_in_attack_action" }

// Change returns the weapon change made.
func (e WeaponChangedInAttackAction) Change() AttackActionWeaponChange { return e.change }

// ChangeKind returns whether the change equips or unequips a weapon.
func (e WeaponChangedInAttackAction) ChangeKind() WeaponChangeKind { return e.change.Change() }

// Timing returns whether the change came before or after the attack.
func (e WeaponChangedInAttackAction) Timing() AttackActionWeaponChangeTiming { return e.timing }

// Slot returns the 1-based number of the attack whose slot the change used.
func (e WeaponChangedInAttackAction) Slot() int { return e.slot }
