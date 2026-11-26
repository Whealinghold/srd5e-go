package combat

import "slices"

// AttackRecordOutcome is the closed set of outcomes of RecordAttack and RecordReplacement.
// Variants: AttackActionAttackRecorded, AttackActionAttackRefused.
type AttackRecordOutcome interface{ isAttackRecordOutcome() }

// AttackActionAttackRecorded: the attack was counted. Read the new progress and the one
// event with Progress and Events.
type AttackActionAttackRecorded struct {
	progress AttackActionProgress
	events   []AttackActionEvent
}

// AttackActionAttackRefused: the attack was not counted and the progress is unchanged.
type AttackActionAttackRefused struct {
	progress AttackActionProgress
	reason   AttackRefusal
}

func (AttackActionAttackRecorded) isAttackRecordOutcome() {}
func (AttackActionAttackRefused) isAttackRecordOutcome()  {}

// Progress returns the new progress.
func (r AttackActionAttackRecorded) Progress() AttackActionProgress { return r.progress }

// Events returns a copy of the events, in the order they happened: exactly one.
func (r AttackActionAttackRecorded) Events() []AttackActionEvent { return slices.Clone(r.events) }

// Progress returns the progress, unchanged.
func (r AttackActionAttackRefused) Progress() AttackActionProgress { return r.progress }

// Reason returns why the attack was refused.
func (r AttackActionAttackRefused) Reason() AttackRefusal { return r.reason }

// Events returns a new, empty slice: a refusal records nothing.
func (AttackActionAttackRefused) Events() []AttackActionEvent { return []AttackActionEvent{} }

// WeaponChangeOutcome is the closed set of outcomes of HandleWeapon. Variants:
// AttackActionWeaponChangeRecorded, AttackActionWeaponChangeRefused.
type WeaponChangeOutcome interface{ isWeaponChangeOutcome() }

// AttackActionWeaponChangeRecorded: the change was recorded in its slot. Read the new
// progress and the one event with Progress and Events.
type AttackActionWeaponChangeRecorded struct {
	progress AttackActionProgress
	events   []AttackActionEvent
}

// AttackActionWeaponChangeRefused: the change was not recorded and the progress is
// unchanged.
type AttackActionWeaponChangeRefused struct {
	progress AttackActionProgress
	reason   WeaponChangeRefusal
}

func (AttackActionWeaponChangeRecorded) isWeaponChangeOutcome() {}
func (AttackActionWeaponChangeRefused) isWeaponChangeOutcome()  {}

// Progress returns the new progress.
func (r AttackActionWeaponChangeRecorded) Progress() AttackActionProgress { return r.progress }

// Events returns a copy of the events, in the order they happened: exactly one.
func (r AttackActionWeaponChangeRecorded) Events() []AttackActionEvent { return slices.Clone(r.events) }

// Progress returns the progress, unchanged.
func (r AttackActionWeaponChangeRefused) Progress() AttackActionProgress { return r.progress }

// Reason returns why the change was refused.
func (r AttackActionWeaponChangeRefused) Reason() WeaponChangeRefusal { return r.reason }

// Events returns a new, empty slice: a refusal records nothing.
func (AttackActionWeaponChangeRefused) Events() []AttackActionEvent { return []AttackActionEvent{} }
