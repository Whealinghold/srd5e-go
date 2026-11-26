package combat

import "time"

// StatBlockActionEvent is the closed family of events the stat-block action rules produce
// (monster-stat-block-actions.md FR-048). Each is an immutable value with a constant
// snake_case Kind, past tense, whose Combatants is a fresh copy: [source] for the three
// stat-block use events and [target, source] for the two immunity events. Variants:
// StatBlockPartUsed, MultiattackAttackRecorded, MultiattackAdditionalUseRecorded,
// MultiattackAttackReplaced, SaveImmunityGranted, SaveImmunityExpired. Each embeds
// statBlockActionEventBase.
type StatBlockActionEvent interface {
	Event
	statBlockActionEvent()
}

// statBlockActionEventBase carries the creatures an event is about and puts the event in
// the Event and StatBlockActionEvent families. target is zero for the monster-only events.
type statBlockActionEventBase struct{ source, target CombatantID }

func (statBlockActionEventBase) event()                {}
func (statBlockActionEventBase) statBlockActionEvent() {}

// Combatants returns a new slice holding the monster, or the target then the monster.
func (b statBlockActionEventBase) Combatants() []CombatantID {
	if b.target.IsZero() {
		return []CombatantID{b.source}
	}
	return []CombatantID{b.target, b.source}
}

// StatBlockPartUsed: a part of the stat block was used at the cost of its section. Its Kind
// is stat_block_part_used. Combatants is [monster].
type StatBlockPartUsed struct {
	statBlockActionEventBase
	part LimitedUsePartRef
	cost StatBlockActionCost
}

// Kind returns "stat_block_part_used".
func (StatBlockPartUsed) Kind() string { return "stat_block_part_used" }

// Part returns the part used.
func (e StatBlockPartUsed) Part() LimitedUsePartRef { return e.part }

// Cost returns the kind of action the use cost.
func (e StatBlockPartUsed) Cost() StatBlockActionCost { return e.cost }

// MultiattackAttackRecorded: an attack of the Multiattack was made. Its Kind is
// multiattack_attack_recorded. Combatants is [monster].
type MultiattackAttackRecorded struct {
	statBlockActionEventBase
	part   LimitedUsePartRef
	number int
}

// Kind returns "multiattack_attack_recorded".
func (MultiattackAttackRecorded) Kind() string { return "multiattack_attack_recorded" }

// Part returns the part the attack was made with.
func (e MultiattackAttackRecorded) Part() LimitedUsePartRef { return e.part }

// Number returns the position of the attack among those made, from 1.
func (e MultiattackAttackRecorded) Number() int { return e.number }

// MultiattackAdditionalUseRecorded: the Multiattack's additional use was made. Its Kind is
// multiattack_additional_use_recorded. Combatants is [monster].
type MultiattackAdditionalUseRecorded struct {
	statBlockActionEventBase
	part LimitedUsePartRef
}

// Kind returns "multiattack_additional_use_recorded".
func (MultiattackAdditionalUseRecorded) Kind() string { return "multiattack_additional_use_recorded" }

// Part returns the part used.
func (e MultiattackAdditionalUseRecorded) Part() LimitedUsePartRef { return e.part }

// MultiattackAttackReplaced: attacks of the Multiattack were replaced by a use of a part. Its
// Kind is multiattack_attack_replaced. Combatants is [monster].
type MultiattackAttackReplaced struct {
	statBlockActionEventBase
	part     LimitedUsePartRef
	replaced LimitedUsePartRef
}

// Kind returns "multiattack_attack_replaced".
func (MultiattackAttackReplaced) Kind() string { return "multiattack_attack_replaced" }

// Part returns the part used in place of the attack.
func (e MultiattackAttackReplaced) Part() LimitedUsePartRef { return e.part }

// Replaced returns the attack part whose attack was replaced.
func (e MultiattackAttackReplaced) Replaced() LimitedUsePartRef { return e.replaced }

// SaveImmunityGranted: a target became immune to a part of a creature. Its Kind is
// save_immunity_granted. Combatants is [target, source].
type SaveImmunityGranted struct {
	statBlockActionEventBase
	part      LimitedUsePartRef
	remaining time.Duration
}

// Kind returns "save_immunity_granted".
func (SaveImmunityGranted) Kind() string { return "save_immunity_granted" }

// Part returns the part the target is immune to.
func (e SaveImmunityGranted) Part() LimitedUsePartRef { return e.part }

// Remaining returns the time the immunity now has left.
func (e SaveImmunityGranted) Remaining() time.Duration { return e.remaining }

// SaveImmunityExpired: a target's immunity to a part ended. Its Kind is
// save_immunity_expired. Combatants is [target, source].
type SaveImmunityExpired struct {
	statBlockActionEventBase
	part LimitedUsePartRef
}

// Kind returns "save_immunity_expired".
func (SaveImmunityExpired) Kind() string { return "save_immunity_expired" }

// Part returns the part the target was immune to.
func (e SaveImmunityExpired) Part() LimitedUsePartRef { return e.part }
