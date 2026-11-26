package combat

import "github.com/Whealinghold/srd5e-go/d20"

// MonsterTraitEvent is the closed family of events the monster trait rules produce
// (monster-traits.md FR-090). Each is an immutable value, comparable with ==, named in
// the past tense, with a constant snake_case Kind and Combatants equal to the one
// creature it is about. Variants: LegendaryResistanceUsed, UndeadFortitudeSaved,
// RegenerationSuppressed, RegenerationDeathDeferred, ParryUsed. Each embeds
// monsterTraitEventBase.
type MonsterTraitEvent interface {
	Event
	monsterTraitEvent()
}

// monsterTraitEventBase carries the creature an event is about and puts the event in the
// Event and MonsterTraitEvent families.
type monsterTraitEventBase struct{ subject CombatantID }

func (monsterTraitEventBase) event()             {}
func (monsterTraitEventBase) monsterTraitEvent() {}

// Combatants returns a new slice holding the creature.
func (b monsterTraitEventBase) Combatants() []CombatantID { return []CombatantID{b.subject} }

// LegendaryResistanceUsed: the creature chose to succeed on a saving throw it failed. Its
// Kind is legendary_resistance_used.
type LegendaryResistanceUsed struct{ monsterTraitEventBase }

// Kind returns "legendary_resistance_used".
func (LegendaryResistanceUsed) Kind() string { return "legendary_resistance_used" }

// UndeadFortitudeSaved: the creature made the Constitution saving throw of Undead
// Fortitude and dropped to 1 Hit Point instead of 0. Its Kind is undead_fortitude_saved.
type UndeadFortitudeSaved struct {
	monsterTraitEventBase
	dc d20.DifficultyClass
}

// Kind returns "undead_fortitude_saved".
func (UndeadFortitudeSaved) Kind() string { return "undead_fortitude_saved" }

// DC returns the difficulty class of the saving throw.
func (e UndeadFortitudeSaved) DC() d20.DifficultyClass { return e.dc }

// RegenerationSuppressed: the creature took a Damage Type that stops its Regeneration on
// its next turn. Its Kind is regeneration_suppressed.
type RegenerationSuppressed struct{ monsterTraitEventBase }

// Kind returns "regeneration_suppressed".
func (RegenerationSuppressed) Kind() string { return "regeneration_suppressed" }

// RegenerationDeathDeferred: the creature dropped to 0 Hit Points and its Regeneration
// kept it from dying until it starts a turn without regenerating. Its Kind is
// regeneration_death_deferred.
type RegenerationDeathDeferred struct{ monsterTraitEventBase }

// Kind returns "regeneration_death_deferred".
func (RegenerationDeathDeferred) Kind() string { return "regeneration_death_deferred" }

// ParryUsed: the creature raised its Armor Class against a melee attack with Parry. Its
// Kind is parry_used.
type ParryUsed struct {
	monsterTraitEventBase
	bonus      int
	becameMiss bool
}

// Kind returns "parry_used".
func (ParryUsed) Kind() string { return "parry_used" }

// Bonus returns the number added to Armor Class.
func (e ParryUsed) Bonus() int { return e.bonus }

// BecameMiss reports whether the attack that hit now misses.
func (e ParryUsed) BecameMiss() bool { return e.becameMiss }
