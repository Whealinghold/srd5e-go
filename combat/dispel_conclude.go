package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
)

// DispelMagicProblem says why Dispel Magic cannot be concluded with the checks the
// host gave. The zero value is not a problem.
type DispelMagicProblem int

const (
	// DispelMissingCheck: a DispelNeedsCheck step has no check.
	DispelMissingCheck DispelMagicProblem = iota + 1
	// DispelUnexpectedCheck: a check names a spell the plan did not ask a check of.
	DispelUnexpectedCheck
	// DispelCheckTargetMismatch: a check's target is not the step's DC.
	DispelCheckTargetMismatch
	// DispelNotAnAbilityCheck: a check's kind is not an ability check.
	DispelNotAnAbilityCheck
	// DispelPlanMismatch: the registry no longer holds the spells the plan was made for.
	DispelPlanMismatch
	// DispelLimitExceeded: the target holds more ongoing spells than the cap of 64 (plan P10). The use case reports it
	// before any check is rolled; Conclude never returns it.
	DispelLimitExceeded
)

// String returns the snake_case name of the problem, or "unknown".
func (p DispelMagicProblem) String() string {
	return enumName([]string{"missing_check", "unexpected_check", "check_target_mismatch", "not_an_ability_check", "plan_mismatch", "limit_exceeded"}, int(p))
}

// DispelMagicResult is the closed result of concluding Dispel Magic. Its variants
// are DispelMagicApplied and DispelMagicInvalid.
type DispelMagicResult interface {
	isDispelMagicResult()
}

// DispelMagicApplied is a completed Dispel Magic. Registry is the registry after
// the spells that ended are gone. Ended holds the spells that ended, in plan order;
// Survived holds the checked spells that failed, then the blocked ones. Events
// lists what happened in order. With nothing ended, Registry equals the input and
// Events is empty.
type DispelMagicApplied struct {
	registry OngoingSpellRegistry
	ended    []OngoingSpell
	survived []OngoingSpell
	events   []SpellEndingEvent
}

// DispelMagicInvalid is a conclusion the checks given do not support: Problem says
// which. Nothing changes.
type DispelMagicInvalid struct{ problem DispelMagicProblem }

// Registry returns the registry after the ended spells are gone.
func (r DispelMagicApplied) Registry() OngoingSpellRegistry { return r.registry }

// Ended returns a copy of the spells that ended, in plan order.
func (r DispelMagicApplied) Ended() []OngoingSpell { return slices.Clone(r.ended) }

// Survived returns a copy of the spells that survived: failed checks, then blocked spells.
func (r DispelMagicApplied) Survived() []OngoingSpell { return slices.Clone(r.survived) }

// Events returns a copy of the events, in order.
func (r DispelMagicApplied) Events() []SpellEndingEvent { return slices.Clone(r.events) }

// Problem returns why the conclusion is invalid.
func (r DispelMagicInvalid) Problem() DispelMagicProblem { return r.problem }

func (DispelMagicApplied) isDispelMagicResult() {}
func (DispelMagicInvalid) isDispelMagicResult() {}

// Conclude applies the checks to the plan and returns the new registry and the
// events (dispel-and-counterspell.md FR-021 to FR-024, FR-059). checks maps the
// effect id of each DispelNeedsCheck step to the ability check the host resolved
// through d20.ResolveCheck; a successful total ends the spell and a total equal to
// the DC succeeds. It returns DispelMagicInvalid for: a registry that no longer
// holds the plan's spells (PlanMismatch, checked first); then, in plan order, a
// missing check, a check whose target is not the DC, and a check that is not an
// ability check; then any check for a spell the plan did not ask of (Unexpected).
// It changes nothing and never panics.
func (DispelMagic) Conclude(registry OngoingSpellRegistry, plan DispelMagicPlan, checks map[OngoingEffectID]d20.CheckResult) DispelMagicResult {
	if !planHeldBy(plan, registry) {
		return DispelMagicInvalid{problem: DispelPlanMismatch}
	}
	if p, bad := checkProblem(plan, checks); bad {
		return DispelMagicInvalid{problem: p}
	}
	ended, survived := sortOutcome(plan, checks)
	endedIDs := make([]OngoingEffectID, 0, len(ended))
	for _, s := range ended {
		endedIDs = append(endedIDs, s.Effect().ID())
	}
	events, parents := endingEvents(plan.target, registry, ended, endedIDs)
	all := slices.Concat(endedIDs, parents)
	return DispelMagicApplied{
		registry: registry.without(all),
		ended:    ended,
		survived: dropIDs(survived, parents),
		events:   events,
	}
}

// planHeldBy reports whether registry holds exactly the spells plan was made for.
func planHeldBy(plan DispelMagicPlan, registry OngoingSpellRegistry) bool {
	if plan.target == nil {
		return len(plan.steps) == 0 && len(registry.SpellsOn(nil)) == 0
	}
	ids := plan.ids()
	if len(registry.SpellsOn(plan.target)) != len(ids) {
		return false
	}
	for _, id := range ids {
		if !registry.holdsOn(plan.target, id) {
			return false
		}
	}
	return true
}

// checkProblem returns the first problem of the checks in plan order, or the
// unexpected-check problem, and whether there is one.
func checkProblem(plan DispelMagicPlan, checks map[OngoingEffectID]d20.CheckResult) (DispelMagicProblem, bool) {
	asked := make([]OngoingEffectID, 0, len(plan.steps))
	for _, step := range plan.steps {
		need, ok := step.(DispelNeedsCheck)
		if !ok {
			continue
		}
		id := need.Spell().Effect().ID()
		asked = append(asked, id)
		check, ok := checks[id]
		switch {
		case !ok:
			return DispelMissingCheck, true
		case check.Target() != need.DC():
			return DispelCheckTargetMismatch, true
		case check.Kind() != d20.AbilityCheck:
			return DispelNotAnAbilityCheck, true
		}
	}
	for id := range checks {
		if !slices.Contains(asked, id) {
			return DispelUnexpectedCheck, true
		}
	}
	return 0, false
}

// sortOutcome splits the plan into the spells that ended and the ones that survived.
func sortOutcome(plan DispelMagicPlan, checks map[OngoingEffectID]d20.CheckResult) (ended, survived []OngoingSpell) {
	var failed, blocked []OngoingSpell
	for _, step := range plan.steps {
		switch st := step.(type) {
		case DispelEnds:
			ended = append(ended, st.Spell())
		case DispelNeedsCheck:
			if checks[st.Spell().Effect().ID()].Outcome() == d20.Success {
				ended = append(ended, st.Spell())
			} else {
				failed = append(failed, st.Spell())
			}
		case DispelBlocked:
			blocked = append(blocked, st.Spell())
		}
	}
	return ended, append(failed, blocked...)
}

// endingEvents builds one OngoingSpellEnded per ended spell, and after the last
// ended part of a parent, the parent's event once no part of it remains (FR-024,
// FR-009). It returns the events and the ids of the parents that ended.
func endingEvents(target DispelTarget, registry OngoingSpellRegistry, ended []OngoingSpell, endedIDs []OngoingEffectID) ([]SpellEndingEvent, []OngoingEffectID) {
	last := make(map[OngoingEffectID]int)
	for i, s := range ended {
		if p := s.PartOf(); !p.IsZero() {
			last[p] = i
		}
	}
	var events []SpellEndingEvent
	var parents []OngoingEffectID
	remaining := registry.without(endedIDs).SpellsOn(target)
	for i, s := range ended {
		events = append(events, endedEvent(target, s, SpellEndDispelled))
		p := s.PartOf()
		if p.IsZero() || last[p] != i || slices.Contains(endedIDs, p) || !registry.holdsOn(target, p) || hasPartOf(remaining, p) {
			continue
		}
		for _, parent := range registry.SpellsOn(target) {
			if parent.Effect().ID() == p {
				events = append(events, endedEvent(target, parent, SpellEndAllPartsEnded))
				parents = append(parents, p)
			}
		}
	}
	return events, parents
}

// hasPartOf reports whether any spell in spells is a part of parent.
func hasPartOf(spells []OngoingSpell, parent OngoingEffectID) bool {
	for _, s := range spells {
		if s.PartOf() == parent {
			return true
		}
	}
	return false
}

// endedEvent builds the event of s ending on target for cause.
func endedEvent(target DispelTarget, s OngoingSpell, cause SpellEndCause) OngoingSpellEnded {
	return OngoingSpellEnded{
		target:                  target,
		effect:                  s.Effect().ID(),
		spell:                   s.SpellID(),
		castAtLevel:             s.CastAtLevel(),
		cause:                   cause,
		conditions:              s.Conditions(),
		endsCasterConcentration: s.RequiresConcentration() && !s.caster.IsZero(),
		caster:                  s.caster,
	}
}

// dropIDs returns spells minus those whose effect ids are in ids, order kept. A parent
// that ended by its last part is an event, not a survivor.
func dropIDs(spells []OngoingSpell, ids []OngoingEffectID) []OngoingSpell {
	var kept []OngoingSpell
	for _, s := range spells {
		if !slices.Contains(ids, s.Effect().ID()) {
			kept = append(kept, s)
		}
	}
	return kept
}
