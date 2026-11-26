package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// effectStart is everything startEffect reads.
type effectStart struct {
	effects               ActiveEffects
	id                    ActiveEffectID
	origin                EffectOrigin
	caster                CombatantID
	subject               EffectSubject
	template              EffectTemplate
	binding               ConcentrationBinding
	requiresConcentration bool
	conditions            []content.ConditionKind
	active                CombatantID
}

// startEffect adds one effect at the end of the registry (FR-013) or says why not (FR-014): a
// duplicate id, then a binding that does not match the spell, then a caster already bound to
// another Concentration subject. It runs no clock and applies no condition.
func startEffect(s effectStart) (ActiveEffects, []ActiveEffectEvent, ActiveEffectRejection) {
	if s.effects.index(s.id) >= 0 {
		return s.effects, nil, rejection(ActiveEffectProblem{Reason: EffectRejectDuplicateID, Effect: s.id})
	}
	if !bindingMatches(s) {
		return s.effects, nil, rejection(ActiveEffectProblem{Reason: EffectRejectBindingMismatch, Effect: s.id})
	}
	if !s.binding.IsZero() && slices.ContainsFunc(s.effects.entries, func(e ActiveEffect) bool {
		return e.caster == s.caster && !e.binding.IsZero() && e.binding.subject != s.binding.subject
	}) {
		return s.effects, nil, rejection(ActiveEffectProblem{Reason: EffectRejectConcentrationConflict, Effect: s.id})
	}
	if c, ok := creatureOf(s.subject); ok && s.effects.heldBy(c) >= maxEffectsPerCreature {
		return s.effects, nil, rejection(ActiveEffectProblem{Reason: EffectRejectLimitExceeded, Effect: s.id, Combatant: c.id})
	}
	e := ActiveEffect{id: s.id, origin: s.origin, caster: s.caster, subject: s.subject, binding: s.binding,
		template: s.template, stored: s.template.stored}
	if _, ok := creatureOf(s.subject); ok {
		e.held = slices.Clone(s.conditions)
	}
	if r, ok := s.template.overall.(EffectLastsRounds); ok {
		e.overallLeft = r.rounds
	}
	e = e.enter(0, s.active)
	entries := append(slices.Clone(s.effects.entries), e)
	return ActiveEffects{entries}, []ActiveEffectEvent{ActiveEffectStarted{baseOf(e)}}, ActiveEffectRejection{}
}

// bindingMatches reports whether the binding is the one the spell calls for (FR-012): a spell that
// needs Concentration has the binding of its caster on that spell, any other effect none.
func bindingMatches(s effectStart) bool {
	o, isSpell := s.origin.(SpellEffectOrigin)
	if !isSpell || !s.requiresConcentration {
		return s.binding.IsZero()
	}
	return s.binding.caster == s.caster && s.binding.subject == character.NewConcentrationSubjectSpell(o.spell)
}
