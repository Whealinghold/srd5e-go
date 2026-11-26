package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ActiveEffectEvent is the closed family of events the active effect rules produce. Each is an
// immutable value with a constant snake_case Kind, past tense. Its Combatants are the creature the
// effect is on, then the caster, without a repeat; for an effect on a place the caster only.
// Variants: ActiveEffectStarted, ActiveEffectEnded, ActiveEffectRepeatSaveMade,
// ActiveEffectStageAdvanced, ActiveEffectStoredDamageGrew, ActiveEffectSubjectMoved. Each embeds
// activeEffectEventBase. There is no accessor for a creature, because no function of this package
// returns a CombatantID; read them from Combatants.
type ActiveEffectEvent interface {
	Event
	activeEffectEvent()
	// EffectID returns the id of the effect the event is about.
	EffectID() ActiveEffectID
}

// activeEffectEventBase carries what every active effect event is about and puts the event in the
// Event and ActiveEffectEvent families.
type activeEffectEventBase struct {
	effect  ActiveEffectID
	subject EffectSubject
	caster  CombatantID
}

func (activeEffectEventBase) event()             {}
func (activeEffectEventBase) activeEffectEvent() {}

// EffectID returns the id of the effect.
func (b activeEffectEventBase) EffectID() ActiveEffectID { return b.effect }

// Combatants returns a new slice: the creature the effect is on, then the caster, once each.
func (b activeEffectEventBase) Combatants() []CombatantID {
	return effectCombatants(b.caster, b.subject)
}

// effectCombatants lists the creatures of subjects in order, then the caster, without a repeat.
func effectCombatants(caster CombatantID, subjects ...EffectSubject) []CombatantID {
	out := []CombatantID{}
	for _, s := range subjects {
		if c, ok := creatureOf(s); ok && !contains(out, c.id) {
			out = append(out, c.id)
		}
	}
	if !contains(out, caster) {
		out = append(out, caster)
	}
	return out
}

func contains(ids []CombatantID, id CombatantID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func baseOf(e ActiveEffect) activeEffectEventBase {
	return activeEffectEventBase{e.id, e.subject, e.caster}
}

// ActiveEffectStarted: an effect was added to the registry. Its Kind is active_effect_started.
type ActiveEffectStarted struct{ activeEffectEventBase }

// Kind returns "active_effect_started".
func (ActiveEffectStarted) Kind() string { return "active_effect_started" }

// ActiveEffectEnded: an effect left the registry, for the reason. Its Kind is active_effect_ended.
type ActiveEffectEnded struct {
	activeEffectEventBase
	reason EffectEndReason
}

// Kind returns "active_effect_ended".
func (ActiveEffectEnded) Kind() string { return "active_effect_ended" }

// Reason returns why the effect ended.
func (e ActiveEffectEnded) Reason() EffectEndReason { return e.reason }

// EffectSaveKind is which saving throw of an effect was made: a repeat at a turn boundary, a stage
// gate or a save against effect damage. The zero value is invalid.
type EffectSaveKind int

// The kinds of saving throw.
const (
	// EffectSaveRepeat is a RepeatSave trigger.
	EffectSaveRepeat EffectSaveKind = iota + 1
	// EffectSaveStageGate is the gate at the end of a stage.
	EffectSaveStageGate
	// EffectSaveDamage is a save against effect damage.
	EffectSaveDamage
)

// String returns "repeat", "stage_gate" or "damage", or "unknown".
func (k EffectSaveKind) String() string {
	switch k {
	case EffectSaveRepeat:
		return "repeat"
	case EffectSaveStageGate:
		return "stage_gate"
	case EffectSaveDamage:
		return "damage"
	}
	return "unknown"
}

// ActiveEffectRepeatSaveMade: the creature an effect is on made a repeated or gate saving throw.
// Its Kind is active_effect_repeat_save_made.
type ActiveEffectRepeatSaveMade struct {
	activeEffectEventBase
	ability ability.Ability
	kind    EffectSaveKind
	outcome d20.Outcome
}

// Kind returns "active_effect_repeat_save_made".
func (ActiveEffectRepeatSaveMade) Kind() string { return "active_effect_repeat_save_made" }

// Ability returns the ability of the save.
func (e ActiveEffectRepeatSaveMade) Ability() ability.Ability { return e.ability }

// SaveKind returns whether it was a repeat or a stage gate.
func (e ActiveEffectRepeatSaveMade) SaveKind() EffectSaveKind { return e.kind }

// Outcome returns the outcome of the save; a forced failure is a Failure.
func (e ActiveEffectRepeatSaveMade) Outcome() d20.Outcome { return e.outcome }

// ActiveEffectStageAdvanced: an effect entered its next stage. Its Kind is
// active_effect_stage_advanced.
type ActiveEffectStageAdvanced struct {
	activeEffectEventBase
	stage int
}

// Kind returns "active_effect_stage_advanced".
func (ActiveEffectStageAdvanced) Kind() string { return "active_effect_stage_advanced" }

// Stage returns the index of the stage entered.
func (e ActiveEffectStageAdvanced) Stage() int { return e.stage }

// ActiveEffectStoredDamageGrew: an effect's stored damage grew. Its Kind is
// active_effect_stored_damage_grew.
type ActiveEffectStoredDamageGrew struct {
	activeEffectEventBase
	stored dice.Expression
}

// Kind returns "active_effect_stored_damage_grew".
func (ActiveEffectStoredDamageGrew) Kind() string { return "active_effect_stored_damage_grew" }

// Stored returns the damage stored now.
func (e ActiveEffectStoredDamageGrew) Stored() dice.Expression { return e.stored }

// ActiveEffectSubjectMoved: an effect moved to a new subject. Its Combatants are the new subject,
// the old, then the caster. Its Kind is active_effect_subject_moved.
type ActiveEffectSubjectMoved struct {
	activeEffectEventBase
	previous EffectSubject
}

// Kind returns "active_effect_subject_moved".
func (ActiveEffectSubjectMoved) Kind() string { return "active_effect_subject_moved" }

// Combatants returns a new slice: the new subject, the old, then the caster, once each.
func (e ActiveEffectSubjectMoved) Combatants() []CombatantID {
	return effectCombatants(e.caster, e.subject, e.previous)
}

// Previous returns the subject the effect was on.
func (e ActiveEffectSubjectMoved) Previous() EffectSubject { return e.previous }

// Subject returns the subject the effect is on now.
func (e ActiveEffectSubjectMoved) Subject() EffectSubject { return e.subject }
