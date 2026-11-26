package combat

import "slices"

// MonsterLimitedUseEvent is the closed family of events the monster limited-use
// rules produce. Each is an immutable value with a constant snake_case Kind, past
// tense, whose Combatants is a fresh copy holding the one monster combatant.
// Variants: LimitedUsePartSpent, MonsterRechargeRolled, LimitedUsesRestored,
// LegendaryActionTaken, LegendaryActionUsesRegained, MonsterLairStatusChanged.
// Each embeds monsterLimitedUseEventBase.
type MonsterLimitedUseEvent interface {
	Event
	monsterLimitedUseEvent()
}

// monsterLimitedUseEventBase carries the monster combatant an event is about and
// puts the event in the Event and MonsterLimitedUseEvent families.
type monsterLimitedUseEventBase struct{ combatant CombatantID }

func (monsterLimitedUseEventBase) event()                  {}
func (monsterLimitedUseEventBase) monsterLimitedUseEvent() {}

// Combatants returns a new slice holding the monster.
func (b monsterLimitedUseEventBase) Combatants() []CombatantID { return []CombatantID{b.combatant} }

// LimitedUsePartSpent: a use of a part or a spell was spent. Its Kind is
// limited_use_part_spent. Combatants is [monster].
type LimitedUsePartSpent struct {
	monsterLimitedUseEventBase
	subject   LimitedUseSubject
	remaining int
}

// Kind returns "limited_use_part_spent".
func (LimitedUsePartSpent) Kind() string { return "limited_use_part_spent" }

// Subject returns the part or spell spent (PartSubject or SpellSubject).
func (e LimitedUsePartSpent) Subject() LimitedUseSubject { return e.subject }

// Remaining returns the uses remaining, 0 for a spent Recharge part.
func (e LimitedUsePartSpent) Remaining() int { return e.remaining }

// MonsterRechargeRolled: the 1d6 of a spent Recharge part was rolled at the start
// of the monster's turn. Its Kind is monster_recharge_rolled. Combatants is
// [monster].
type MonsterRechargeRolled struct {
	monsterLimitedUseEventBase
	part   LimitedUsePartRef
	face   int
	result RechargeRollResult
}

// Kind returns "monster_recharge_rolled".
func (MonsterRechargeRolled) Kind() string { return "monster_recharge_rolled" }

// Part returns the part rolled for.
func (e MonsterRechargeRolled) Part() LimitedUsePartRef { return e.part }

// Face returns the face of the d6.
func (e MonsterRechargeRolled) Face() int { return e.face }

// Result returns RechargeRegained or RechargeNotRegained.
func (e MonsterRechargeRolled) Result() RechargeRollResult { return e.result }

// LimitedUsesRestored: a rest restored uses or recharged parts. Its Kind is
// limited_uses_restored. Combatants is [monster]. It holds a slice, so it is not
// comparable with ==.
type LimitedUsesRestored struct {
	monsterLimitedUseEventBase
	subjects []LimitedUseSubject
}

// Kind returns "limited_uses_restored".
func (LimitedUsesRestored) Kind() string { return "limited_uses_restored" }

// Subjects returns, as a new slice and in printed order (parts, then spells), the
// parts and spells that changed.
func (e LimitedUsesRestored) Subjects() []LimitedUseSubject { return slices.Clone(e.subjects) }

// LegendaryActionTaken: a Legendary Action was taken and a use spent. Its Kind is
// legendary_action_taken. Combatants is [monster].
type LegendaryActionTaken struct {
	monsterLimitedUseEventBase
	window    LegendaryActionWindow
	remaining int
}

// Kind returns "legendary_action_taken".
func (LegendaryActionTaken) Kind() string { return "legendary_action_taken" }

// Window returns the window it was taken in.
func (e LegendaryActionTaken) Window() LegendaryActionWindow { return e.window }

// Remaining returns the Legendary Action uses left.
func (e LegendaryActionTaken) Remaining() int { return e.remaining }

// LegendaryActionUsesRegained: the monster regained spent Legendary Action uses at
// the start of its turn. Its Kind is legendary_action_uses_regained. Combatants is
// [monster].
type LegendaryActionUsesRegained struct {
	monsterLimitedUseEventBase
	count int
}

// Kind returns "legendary_action_uses_regained".
func (LegendaryActionUsesRegained) Kind() string { return "legendary_action_uses_regained" }

// Count returns the number of uses regained, 1 or more.
func (e LegendaryActionUsesRegained) Count() int { return e.count }

// MonsterLairStatusChanged: the host reported another lair status. Its Kind is
// monster_lair_status_changed. Combatants is [monster].
type MonsterLairStatusChanged struct {
	monsterLimitedUseEventBase
	from, to LairStatus
}

// Kind returns "monster_lair_status_changed".
func (MonsterLairStatusChanged) Kind() string { return "monster_lair_status_changed" }

// From returns the earlier status.
func (e MonsterLairStatusChanged) From() LairStatus { return e.from }

// To returns the new status.
func (e MonsterLairStatusChanged) To() LairStatus { return e.to }
