package combat

import "slices"

// effectEnd is the outcome of endEffects: the rejection (if any), whether the cause was a move, and
// the work done.
type effectEnd struct {
	rejection ActiveEffectRejection
	moved     bool
	work      *effectWork
	effects   ActiveEffects
}

// endEffects applies an ending cause to the registry (FR-040 to FR-047). A rejection leaves the
// registry as it was. Ended effects leave it in registry order, each with one ActiveEffectEnded; no
// onEnd action runs for a dead subject (FR-038).
func endEffects(r ActiveEffects, cause EndingCause) effectEnd {
	w := newEffectWork(r)
	out := effectEnd{work: w, effects: r}
	each := func(match func(ActiveEffect) bool, reason EffectEndReason, runOnEnd bool) {
		for i, e := range w.entries {
			if match(e) {
				w.end(i, reason, runOnEnd, "")
			}
		}
	}
	switch v := cause.(type) {
	case ConcentrationLost:
		each(func(e ActiveEffect) bool { return e.binding == v.binding }, EffectConcentrationLost, true)
	case Dismissed:
		if out.rejection = dismissalProblems(r, v); len(out.rejection.problems) > 0 {
			return out
		}
		each(func(e ActiveEffect) bool { return slices.Contains(v.ids, e.id) }, EffectDismissed, true)
	case SubjectTookDamage:
		if v.total >= 1 {
			each(func(e ActiveEffect) bool { return e.OnCreature(v.subject) && e.stageDef().endsOnDamage }, EffectSubjectTookDamage, true)
		}
	case SubjectDied:
		each(func(e ActiveEffect) bool { return e.OnCreature(v.subject) && !e.template.survives }, EffectSubjectDied, false)
	case RemovedByEffect:
		var unknown []ActiveEffectProblem
		for _, id := range v.ids {
			if r.index(id) < 0 {
				unknown = append(unknown, ActiveEffectProblem{Reason: EffectRejectUnknownEffect, Effect: id})
			}
		}
		if out.rejection = rejection(unknown...); len(unknown) > 0 {
			return out
		}
		each(func(e ActiveEffect) bool { return slices.Contains(v.ids, e.id) }, EffectRemovedByEffect, true)
	case MoveEffectSubject:
		out.moved = true
		return moveEffect(out, v)
	}
	out.effects = w.finish()
	return out
}

// dismissalProblems checks a dismissal (FR-042): an Incapacitated caster alone, else every problem
// of every named effect, in the order unknown, not the caster, uses Concentration, not a Time Span.
func dismissalProblems(r ActiveEffects, d Dismissed) ActiveEffectRejection {
	if d.incapacitated {
		return rejection(ActiveEffectProblem{Reason: EffectRejectCasterIncapacitated})
	}
	var out []ActiveEffectProblem
	for _, id := range d.ids {
		i := r.index(id)
		if i < 0 {
			out = append(out, ActiveEffectProblem{Reason: EffectRejectUnknownEffect, Effect: id})
			continue
		}
		e := r.entries[i]
		if e.caster != d.caster {
			out = append(out, ActiveEffectProblem{Reason: EffectRejectNotTheCaster, Effect: id})
		}
		if !e.binding.IsZero() {
			out = append(out, ActiveEffectProblem{Reason: EffectRejectUsesConcentration, Effect: id})
		}
		if _, ok := e.duration.(EffectLastsRounds); !ok {
			out = append(out, ActiveEffectProblem{Reason: EffectRejectNotTimeSpan, Effect: id})
		}
	}
	return rejection(out...)
}

// moveEffect re-points an effect to a new creature (FR-060), keeping everything else. The reasons
// are checked in the order unknown, not movable, new subject not a creature, same subject.
func moveEffect(out effectEnd, m MoveEffectSubject) effectEnd {
	i := out.effects.index(m.id)
	reject := func(k ActiveEffectRejectionKind) effectEnd {
		out.rejection = rejection(ActiveEffectProblem{Reason: k, Effect: m.id})
		return out
	}
	if i < 0 {
		return reject(EffectRejectUnknownEffect)
	}
	e := out.effects.entries[i]
	switch {
	case !e.template.survives:
		return reject(EffectRejectNotMovable)
	case !isCreature(m.newSubject):
		return reject(EffectRejectNewSubjectNotCreature)
	case e.subject == m.newSubject:
		return reject(EffectRejectSameSubject)
	case out.effects.heldBy(m.newSubject.(CreatureEffectSubject)) >= maxEffectsPerCreature: // newSubject is a creature here
		return reject(EffectRejectLimitExceeded)
	}
	moved := e
	moved.subject = m.newSubject
	entries := slices.Clone(out.effects.entries)
	entries[i] = moved
	out.effects = ActiveEffects{entries}
	out.work.events = []ActiveEffectEvent{ActiveEffectSubjectMoved{baseOf(moved), e.subject}}
	return out
}

func isCreature(s EffectSubject) bool { _, ok := creatureOf(s); return ok }
