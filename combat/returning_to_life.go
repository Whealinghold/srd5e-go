package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// RevivalConsent is whether the spirit of a Dead creature accepts being revived
// (SRD 5.2.1, "Dead", p. 180: the spirit can refuse). The zero value is not a
// consent and counts as a refusal.
type RevivalConsent int

// The consents of a spirit.
const (
	// SpiritAccepts: the spirit is willing to return.
	SpiritAccepts RevivalConsent = iota + 1
	// SpiritRefuses: the spirit refuses to return.
	SpiritRefuses
)

// String returns "accepts", "refuses" or "unknown".
func (c RevivalConsent) String() string { return enumName([]string{"accepts", "refuses"}, int(c)) }

// ReviveCreatureCommand is the host's facts for a return to life. Construct it with
// a keyed literal.
type ReviveCreatureCommand struct {
	// Subject is the creature that returns; the zero id is invalid.
	Subject CombatantID
	// Status is the creature's Hit Point status; it must be Dead to be revived.
	Status HitPointStatus
	// HitPoints is the current Hit Points the revival effect determines.
	HitPoints int
	// Consent is the spirit's answer; only SpiritAccepts revives.
	Consent RevivalConsent
	// Conditions and Effects are what the creature had at death.
	Conditions ConditionSet
	Effects    OngoingEffects
	// ElapsedConditions are the explicit condition kinds whose duration the host
	// reports as over; the rest count as ongoing. Exhaustion is never dropped here.
	ElapsedConditions []content.ConditionKind
	// ElapsedEffects are the ids of the effects whose duration the host reports as
	// over; the rest count as ongoing.
	ElapsedEffects []OngoingEffectID
}

// ReviveCreatureResult is the closed set of answers of Revive. Every variant
// reports the creature's Hit Point status, conditions and effects after the call;
// a refusal reports them unchanged and emits no event. Variants:
// ReviveCreatureApplied, ReviveCreatureNotDead, ReviveCreatureRefused,
// ReviveCreatureHitPointsBelowOne, ReviveCreatureMaximumBelowOne.
type ReviveCreatureResult interface {
	isReviveCreatureResult()
	Status() HitPointStatus
	Conditions() ConditionSet
	Effects() OngoingEffects
}

// reviveState is the part of a result that reports the creature's state.
type reviveState struct {
	status     HitPointStatus
	conditions ConditionSet
	effects    OngoingEffects
}

// Status returns the Hit Point status after the call.
func (s reviveState) Status() HitPointStatus { return s.status }

// Conditions returns the condition set after the call.
func (s reviveState) Conditions() ConditionSet { return s.conditions }

// Effects returns the ongoing effects after the call.
func (s reviveState) Effects() OngoingEffects { return s.effects }

// ReviveCreatureApplied: the creature returned to life, Alive with the capped Hit
// Points and no Temporary Hit Points. Events holds, in order, one
// CreatureReturnedToLife, the ConditionRemoved of each condition dropped and the
// event of the Exhaustion reduction, if any.
type ReviveCreatureApplied struct {
	reviveState
	events []Event
}

// Events returns a copy of the events in the order they happened.
func (r ReviveCreatureApplied) Events() []Event { return slices.Clone(r.events) }

// ReviveCreatureNotDead: the creature is not Dead (or the subject is the zero id),
// so nothing changed.
type ReviveCreatureNotDead struct{ reviveState }

// ReviveCreatureRefused: the spirit refuses (or the consent is not SpiritAccepts),
// so nothing changed.
type ReviveCreatureRefused struct{ reviveState }

// ReviveCreatureHitPointsBelowOne: the revival Hit Points are below 1, so nothing
// changed.
type ReviveCreatureHitPointsBelowOne struct{ reviveState }

// ReviveCreatureMaximumBelowOne: the Hit Point maximum is below 1, so nothing
// changed.
type ReviveCreatureMaximumBelowOne struct{ reviveState }

func (ReviveCreatureApplied) isReviveCreatureResult()           {}
func (ReviveCreatureNotDead) isReviveCreatureResult()           {}
func (ReviveCreatureRefused) isReviveCreatureResult()           {}
func (ReviveCreatureHitPointsBelowOne) isReviveCreatureResult() {}
func (ReviveCreatureMaximumBelowOne) isReviveCreatureResult()   {}

// Revive returns a Dead creature to life (SRD 5.2.1, "Dead", p. 180). It decides
// in this order: the creature is not Dead (ReviveCreatureNotDead); the consent is
// not SpiritAccepts (ReviveCreatureRefused); the Hit Point maximum is below 1
// (ReviveCreatureMaximumBelowOne); the revival Hit Points are below 1
// (ReviveCreatureHitPointsBelowOne); otherwise ReviveCreatureApplied. The new
// status is Alive at the revival Hit Points capped at the maximum, with 0
// Temporary Hit Points and the rule and affinities kept; the input status is
// never changed. Explicit conditions the host reports elapsed are removed (an
// Exhaustion level is never dropped this way), the others stay; the creature
// returns with one fewer Exhaustion level, none left when it had one or none; the
// effects the host reports elapsed are dropped. An id or kind reported elapsed that
// the creature does not hold is ignored. It never panics and holds no state.
func Revive(c ReviveCreatureCommand) ReviveCreatureResult {
	same := reviveState{c.Status, c.Conditions, c.Effects}
	hp := c.Status.hp
	_, dead := c.Status.life.(Dead)
	switch {
	case !dead || c.Subject.IsZero():
		return ReviveCreatureNotDead{same}
	case c.Consent != SpiritAccepts:
		return ReviveCreatureRefused{same}
	case hp.maximum < 1:
		return ReviveCreatureMaximumBelowOne{same}
	case c.HitPoints < 1:
		return ReviveCreatureHitPointsBelowOne{same}
	}
	current := min(c.HitPoints, hp.maximum)
	events := []Event{CreatureReturnedToLife{revivalEventBase{c.Subject}, current}}
	conditions, dropped := dropElapsedConditions(c.Subject, c.Conditions, c.ElapsedConditions)
	events = append(events, dropped...)
	// Subject is non-zero and the level is 1, so ReduceExhaustion cannot fail.
	if r, err := conditions.ReduceExhaustion(c.Subject, 1); err == nil {
		conditions = r.Set()
		events = append(events, conditionEvents(r)...)
	}
	return ReviveCreatureApplied{reviveState{
		HitPointStatus{HitPoints{current, hp.maximum, 0}, Alive{}, c.Status.rule, c.Status.aff},
		conditions, c.Effects.without(c.ElapsedEffects)}, events}
}

// dropElapsedConditions removes every explicit condition other than Exhaustion
// whose kind is in elapsed, Unconscious first so that a Prone named after it is
// removed too (removing Unconscious leaves the creature Prone).
func dropElapsedConditions(id CombatantID, set ConditionSet, elapsed []content.ConditionKind) (ConditionSet, []Event) {
	var kinds []content.ConditionKind
	for _, c := range set.Explicit() {
		if k := c.Kind(); k != content.ConditionExhaustion && slices.Contains(elapsed, k) {
			kinds = append(kinds, k)
		}
	}
	slices.SortStableFunc(kinds, func(a, b content.ConditionKind) int { return unconsciousFirst(a) - unconsciousFirst(b) })
	var events []Event
	for _, k := range kinds {
		r := set.Remove(id, k)
		set = r.Set()
		events = append(events, conditionEvents(r)...)
	}
	return set, events
}

// conditionEvents returns the events of a condition result as Events, none for a
// result that changed nothing.
func conditionEvents(res any) []Event {
	e, ok := res.(interface{ Events() []ConditionEvent })
	if !ok {
		return nil
	}
	var out []Event
	for _, ev := range e.Events() {
		out = append(out, ev)
	}
	return out
}

// unconsciousFirst ranks Unconscious before every other kind.
func unconsciousFirst(k content.ConditionKind) int {
	if k == content.ConditionUnconscious {
		return 0
	}
	return 1
}
