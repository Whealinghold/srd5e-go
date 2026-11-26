package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// EncounterRejection is why an Encounter refused a command: a field of every
// rejected result. Each variant's String is the snake_case value of
// srd5e.combat.result; UnknownCombatantRejection is the one host-input rejection
// (class invalid), the others are refusals. Variants: EncounterAlreadyEndedRejection,
// UnknownCombatantRejection, CombatantIsDeadRejection, AlreadyHasConditionRejection,
// ImmuneToConditionRejection, DoesNotHaveConditionRejection, ConditionImpliedByRejection,
// HitPointRuleRejection, TurnOrderChangedRejection, DeathSavingThrowAlreadyMadeRejection.
type EncounterRejection interface {
	isEncounterRejection()
	String() string
}

// EncounterAlreadyEndedRejection: the encounter ended, so no command is accepted.
type EncounterAlreadyEndedRejection struct{}

// UnknownCombatantRejection: ID is not a combatant of the encounter.
type UnknownCombatantRejection struct{ ID CombatantID }

// CombatantIsDeadRejection: the combatant ID is Dead and is not changed.
type CombatantIsDeadRejection struct{ ID CombatantID }

// AlreadyHasConditionRejection: the combatant ID already has the condition Kind.
type AlreadyHasConditionRejection struct {
	ID   CombatantID
	Kind content.ConditionKind
}

// ImmuneToConditionRejection: the combatant ID is Immune to the condition Kind.
type ImmuneToConditionRejection struct {
	ID   CombatantID
	Kind content.ConditionKind
}

// DoesNotHaveConditionRejection: the combatant ID does not have the condition Kind.
type DoesNotHaveConditionRejection struct {
	ID   CombatantID
	Kind content.ConditionKind
}

// ConditionImpliedByRejection: the combatant ID holds Kind only because the
// conditions in ImpliedBy imply it. Not comparable with ==.
type ConditionImpliedByRejection struct {
	ID        CombatantID
	Kind      content.ConditionKind
	impliedBy []content.ConditionKind
}

// ImpliedBy returns a copy of the kinds that imply the condition.
func (r ConditionImpliedByRejection) ImpliedBy() []content.ConditionKind {
	return slices.Clone(r.impliedBy)
}

// HitPointRuleRejection: the hit point rules refused the command for the combatant
// ID with Reason (NotDying or NotStable; CreatureIsDead is never carried).
type HitPointRuleRejection struct {
	ID     CombatantID
	Reason HitPointRejection
}

// TurnOrderChangedRejection: the tracker's turn order is not the encounter's.
type TurnOrderChangedRejection struct{}

// DeathSavingThrowAlreadyMadeRejection: the combatant ID already made a Death
// Saving Throw since its turn started.
type DeathSavingThrowAlreadyMadeRejection struct{ ID CombatantID }

func (EncounterAlreadyEndedRejection) isEncounterRejection()       {}
func (UnknownCombatantRejection) isEncounterRejection()            {}
func (CombatantIsDeadRejection) isEncounterRejection()             {}
func (AlreadyHasConditionRejection) isEncounterRejection()         {}
func (ImmuneToConditionRejection) isEncounterRejection()           {}
func (DoesNotHaveConditionRejection) isEncounterRejection()        {}
func (ConditionImpliedByRejection) isEncounterRejection()          {}
func (HitPointRuleRejection) isEncounterRejection()                {}
func (TurnOrderChangedRejection) isEncounterRejection()            {}
func (DeathSavingThrowAlreadyMadeRejection) isEncounterRejection() {}

// String returns "encounter_already_ended".
func (EncounterAlreadyEndedRejection) String() string { return "encounter_already_ended" }

// String returns "unknown_combatant".
func (UnknownCombatantRejection) String() string { return "unknown_combatant" }

// String returns "combatant_is_dead".
func (CombatantIsDeadRejection) String() string { return "combatant_is_dead" }

// String returns "already_has_condition".
func (AlreadyHasConditionRejection) String() string { return "already_has_condition" }

// String returns "immune_to_condition".
func (ImmuneToConditionRejection) String() string { return "immune_to_condition" }

// String returns "does_not_have_condition".
func (DoesNotHaveConditionRejection) String() string { return "does_not_have_condition" }

// String returns "condition_implied_by".
func (ConditionImpliedByRejection) String() string { return "condition_implied_by" }

// String returns the carried reason: "not_dying", "not_stable" or "creature_is_dead".
func (r HitPointRuleRejection) String() string { return r.Reason.String() }

// String returns "turn_order_changed".
func (TurnOrderChangedRejection) String() string { return "turn_order_changed" }

// String returns "death_saving_throw_already_made".
func (DeathSavingThrowAlreadyMadeRejection) String() string {
	return "death_saving_throw_already_made"
}

// rejectionClass is the result class of r: only an unknown combatant is host
// input the engine cannot use (invalid); every other rejection is a refusal.
func rejectionClass(r EncounterRejection) combatResultClass {
	if _, ok := r.(UnknownCombatantRejection); ok {
		return resultInvalid
	}
	return resultRefused
}
