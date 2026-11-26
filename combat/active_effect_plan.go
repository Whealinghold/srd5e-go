package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// EffectConditionPlan is one condition for the host to apply with ApplyConditionToSet or to
// remove with RemoveConditionFromSet. This package applies none (FR-013). Construct it with a keyed
// literal. Comparable and safe for concurrent use.
type EffectConditionPlan struct {
	// Effect is the effect the condition is held for.
	Effect ActiveEffectID
	// Target is the creature the condition is on.
	Target CombatantID
	// Kind is the condition.
	Kind content.ConditionKind
}

// EffectDamagePlan is damage an effect calls for, for ResolveEffectDamage and then ApplyDamage.
// Nothing has been rolled or applied (FR-035). Immutable and safe for concurrent use. The zero
// value is invalid.
type EffectDamagePlan struct {
	effect   ActiveEffectID
	audience EffectDamageAudience
	targets  []CombatantID
	amount   dice.Expression
	typ      content.DamageType
	save     EffectDamageSave
}

// Effect returns the effect the damage is for.
func (p EffectDamagePlan) Effect() ActiveEffectID { return p.effect }

// Audience returns who the damage is for.
func (p EffectDamagePlan) Audience() EffectDamageAudience { return p.audience }

// Targets returns the creatures the plan names, as a new slice: the creature of a subject audience,
// the combatant of the tick for a turn taker, none for area members, which the host lists.
func (p EffectDamagePlan) Targets() []CombatantID { return slices.Clone(p.targets) }

// Amount returns the damage to roll once for all targets.
func (p EffectDamagePlan) Amount() dice.Expression { return p.amount }

// DamageType returns the Damage Type.
func (p EffectDamagePlan) DamageType() content.DamageType { return p.typ }

// Save returns the save that can reduce the damage and whether there is one.
func (p EffectDamagePlan) Save() (EffectDamageSave, bool) { return p.save, !p.save.IsZero() }

// IsZero reports whether p is the invalid zero plan.
func (p EffectDamagePlan) IsZero() bool { return p.effect.IsZero() }

// ActiveEffectRejectionKind is a reason a command was refused. The zero value is not a reason.
type ActiveEffectRejectionKind int

// The reasons, by use case. Refused ones are legal input the rules do not allow in the current
// state; the others are input the engine cannot use (FR-058).
const (
	// EffectRejectDuplicateID (StartActiveEffect, refused): the id is already in the registry.
	EffectRejectDuplicateID ActiveEffectRejectionKind = iota + 1
	// EffectRejectBindingMismatch (StartActiveEffect, invalid): the binding does not match the spell.
	EffectRejectBindingMismatch
	// EffectRejectConcentrationConflict (StartActiveEffect, refused): the caster is bound to another subject.
	EffectRejectConcentrationConflict
	// EffectRejectInvalidOrder (AdvanceActiveEffects, invalid): the order is not a permutation of the effects that fire.
	EffectRejectInvalidOrder
	// EffectRejectMissingSaveAttempt (AdvanceActiveEffects, ResolveEffectDamage, invalid): a needed save has no attempt.
	EffectRejectMissingSaveAttempt
	// EffectRejectCasterIncapacitated (Dismissed, refused): the caster has the Incapacitated condition.
	EffectRejectCasterIncapacitated
	// EffectRejectUnknownEffect (invalid): an id is not in the registry.
	EffectRejectUnknownEffect
	// EffectRejectNotTheCaster (Dismissed, refused): another creature cast the effect.
	EffectRejectNotTheCaster
	// EffectRejectUsesConcentration (Dismissed, refused): the effect ends with the Concentration.
	EffectRejectUsesConcentration
	// EffectRejectNotTimeSpan (Dismissed, refused): the effect is not a Time Span effect.
	EffectRejectNotTimeSpan
	// EffectRejectNoTargets (ResolveEffectDamage, refused): no creature was listed.
	EffectRejectNoTargets
	// EffectRejectDuplicateTarget (ResolveEffectDamage, invalid): a creature was listed twice.
	EffectRejectDuplicateTarget
	// EffectRejectNotMovable (MoveEffectSubject, refused): the template cannot be moved.
	EffectRejectNotMovable
	// EffectRejectNewSubjectNotCreature (MoveEffectSubject, invalid): the new subject is a place.
	EffectRejectNewSubjectNotCreature
	// EffectRejectSameSubject (MoveEffectSubject, refused): the effect is already on that subject.
	EffectRejectSameSubject
	// EffectRejectLimitExceeded (StartActiveEffect, MoveEffectSubject, invalid): the creature already holds 64 effects (plan P10).
	EffectRejectLimitExceeded
)

func rejectionNames() []string {
	return []string{"duplicate_id", "binding_mismatch", "concentration_conflict", "invalid_order", "missing_save_attempt",
		"caster_incapacitated", "unknown_effect", "not_the_caster", "uses_concentration", "not_time_span", "no_targets",
		"duplicate_target", "not_movable", "new_subject_not_creature", "same_subject", "limit_exceeded"}
}

// String returns the snake_case name of k (the value of srd5e.combat.active_effects.rejection), or
// "unknown".
func (k ActiveEffectRejectionKind) String() string {
	if k < 1 || int(k) > len(rejectionNames()) {
		return "unknown"
	}
	return rejectionNames()[k-1]
}

// refusedOnly reports whether k is a refusal rather than unusable input (FR-058).
func (k ActiveEffectRejectionKind) refusedOnly() bool {
	return !slices.Contains([]ActiveEffectRejectionKind{EffectRejectInvalidOrder, EffectRejectBindingMismatch, EffectRejectMissingSaveAttempt,
		EffectRejectUnknownEffect, EffectRejectDuplicateTarget, EffectRejectNewSubjectNotCreature, EffectRejectLimitExceeded}, k)
}

// ActiveEffectProblem is one reason with what it is about. Construct it with a keyed literal.
type ActiveEffectProblem struct {
	// Reason is why the command was refused.
	Reason ActiveEffectRejectionKind
	// Effect is the effect the problem is about; zero when it is about none.
	Effect ActiveEffectID
	// Combatant is the creature the problem is about; zero when it is about none.
	Combatant CombatantID
	// Ability is the ability of a missing save; zero for other reasons.
	Ability ability.Ability
}

// ActiveEffectRejection lists every problem of the first failing group of checks, in the fixed order
// of the specification (FR-063). It is held by every rejected result. Immutable and safe for
// concurrent use. The zero value lists nothing.
type ActiveEffectRejection struct{ problems []ActiveEffectProblem }

func rejection(problems ...ActiveEffectProblem) ActiveEffectRejection {
	return ActiveEffectRejection{slices.Clone(problems)}
}

// Problems returns every problem, in order, as a new slice.
func (r ActiveEffectRejection) Problems() []ActiveEffectProblem { return slices.Clone(r.problems) }

// Reasons returns the reasons without repeats, in the order first met, as a new slice.
func (r ActiveEffectRejection) Reasons() []ActiveEffectRejectionKind {
	var out []ActiveEffectRejectionKind
	for _, p := range r.problems {
		if !slices.Contains(out, p.Reason) {
			out = append(out, p.Reason)
		}
	}
	return out
}

// Equal reports whether r and o list the same problems.
func (r ActiveEffectRejection) Equal(o ActiveEffectRejection) bool {
	return slices.Equal(r.problems, o.problems)
}

// class is invalid if any problem is, else refused (FR-063).
func (r ActiveEffectRejection) class() combatResultClass {
	for _, p := range r.problems {
		if !p.Reason.refusedOnly() {
			return resultInvalid
		}
	}
	return resultRefused
}

// firstReason is the first reason, for the telemetry attribute.
func (r ActiveEffectRejection) firstReason() ActiveEffectRejectionKind {
	if len(r.problems) == 0 {
		return 0
	}
	return r.problems[0].Reason
}
