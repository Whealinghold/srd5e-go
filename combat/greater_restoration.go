package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// GreaterRestorationChoice is the one effect a Greater Restoration cast applies
// ("one of the following effects", SRD 5.2.1, "Greater Restoration", p. 137). It is a
// closed set; a cast cannot take two choices because the value holds one. Variants:
// EndOneExhaustionLevel, EndCharmed, EndPetrified, EndCurse,
// EndAbilityScoreReduction, EndHitPointMaximumReduction.
type GreaterRestorationChoice interface{ isGreaterRestorationChoice() }

// EndOneExhaustionLevel reduces the creature's Exhaustion by one level.
type EndOneExhaustionLevel struct{}

// EndCharmed ends the Charmed condition.
type EndCharmed struct{}

// EndPetrified ends the Petrified condition.
type EndPetrified struct{}

// EndCurse ends the curse with the given id, including the target's Attunement
// to a cursed item (the host ends that). Construct it with a keyed literal.
type EndCurse struct{ ID OngoingEffectID }

// EndAbilityScoreReduction ends a reduction to an ability score. The package holds no
// ability score reduction, so the host resolves it.
type EndAbilityScoreReduction struct{}

// EndHitPointMaximumReduction ends a reduction of Amount to the Hit Point maximum.
// Construct it with a keyed literal.
type EndHitPointMaximumReduction struct{ Amount int }

func (EndOneExhaustionLevel) isGreaterRestorationChoice()       {}
func (EndCharmed) isGreaterRestorationChoice()                  {}
func (EndPetrified) isGreaterRestorationChoice()                {}
func (EndCurse) isGreaterRestorationChoice()                    {}
func (EndAbilityScoreReduction) isGreaterRestorationChoice()    {}
func (EndHitPointMaximumReduction) isGreaterRestorationChoice() {}

// GreaterRestorationCommand is the host's facts for a Greater Restoration cast.
// Construct it with a keyed literal; the creature's state is read and never changed.
type GreaterRestorationCommand struct {
	// Subject is the creature the spell is cast on.
	Subject CombatantID
	// Choice is the one effect applied.
	Choice GreaterRestorationChoice
	// Conditions, Effects and Status are the creature's state.
	Conditions ConditionSet
	Effects    OngoingEffects
	Status     HitPointStatus
}

// CastGreaterRestorationResult is the closed set of answers of ApplyGreaterRestoration.
// Variants: GreaterRestorationApplied, GreaterRestorationNothingToEnd,
// GreaterRestorationNotACurse, GreaterRestorationHostResolved,
// GreaterRestorationRejected, GreaterRestorationInvalid.
type CastGreaterRestorationResult interface{ isCastGreaterRestorationResult() }

// GreaterRestorationApplied: the choice took effect. Conditions, Effects and Status
// are the creature's state after the call. CurseSource is the EffectSource of the
// curse that ended (the zero value for any other choice), so the host can end Attunement
// when it is a magic item. Events holds the events of the change, in order.
type GreaterRestorationApplied struct {
	conditions  ConditionSet
	effects     OngoingEffects
	status      HitPointStatus
	curseSource EffectSource
	events      []Event
}

// Conditions returns the condition set after the call.
func (r GreaterRestorationApplied) Conditions() ConditionSet { return r.conditions }

// Effects returns the ongoing effects after the call.
func (r GreaterRestorationApplied) Effects() OngoingEffects { return r.effects }

// Status returns the Hit Point status after the call.
func (r GreaterRestorationApplied) Status() HitPointStatus { return r.status }

// CurseSource returns the source of the curse that ended, or the zero EffectSource.
func (r GreaterRestorationApplied) CurseSource() EffectSource { return r.curseSource }

// Events returns a copy of the events in the order they happened.
func (r GreaterRestorationApplied) Events() []Event { return slices.Clone(r.events) }

// GreaterRestorationNothingToEnd: the creature has nothing the choice could end (no
// Exhaustion, the condition not held explicitly, no such effect, or an amount of 0);
// nothing changed and no event was emitted.
type GreaterRestorationNothingToEnd struct{}

// GreaterRestorationNotACurse: the effect exists but is not a curse; nothing changed.
type GreaterRestorationNotACurse struct{}

// GreaterRestorationHostResolved: EndAbilityScoreReduction changes no combat state;
// the host applies it to its own ability scores.
type GreaterRestorationHostResolved struct{}

// GreaterRestorationRejected: a Hit Point maximum reduction was chosen for a Dead
// creature; nothing changed. Construct it with a keyed literal.
type GreaterRestorationRejected struct{ Rejection HitPointRejection }

// GreaterRestorationInvalid: the command has no subject, no choice, a choice of an
// unknown type, a blank curse id or a negative amount; nothing changed.
type GreaterRestorationInvalid struct{}

func (GreaterRestorationApplied) isCastGreaterRestorationResult()      {}
func (GreaterRestorationNothingToEnd) isCastGreaterRestorationResult() {}
func (GreaterRestorationNotACurse) isCastGreaterRestorationResult()    {}
func (GreaterRestorationHostResolved) isCastGreaterRestorationResult() {}
func (GreaterRestorationRejected) isCastGreaterRestorationResult()     {}
func (GreaterRestorationInvalid) isCastGreaterRestorationResult()      {}

// ApplyGreaterRestoration applies the one choice of a Greater Restoration cast (SRD
// 5.2.1, "Greater Restoration", p. 137). EndOneExhaustionLevel is ReduceExhaustion by 1;
// EndCharmed and EndPetrified are ConditionSet.Remove (Petrified never ends a
// condition another held condition implies); EndCurse is CurseRemoval with
// GreaterRestorationSpell and reports the curse's source; EndHitPointMaximumReduction
// is RestoreMaximum; EndAbilityScoreReduction is GreaterRestorationHostResolved.
//
// It never panics and does not change the command's state.
func ApplyGreaterRestoration(c GreaterRestorationCommand) CastGreaterRestorationResult {
	if c.Subject.IsZero() {
		return GreaterRestorationInvalid{}
	}
	base := GreaterRestorationApplied{conditions: c.Conditions, effects: c.Effects, status: c.Status}
	switch ch := c.Choice.(type) {
	case EndOneExhaustionLevel:
		// The subject is non-zero and the level is 1, so ReduceExhaustion cannot fail.
		res, _ := c.Conditions.ReduceExhaustion(c.Subject, 1)
		return conditionChangeResult(base, res)
	case EndCharmed:
		return conditionChangeResult(base, c.Conditions.Remove(c.Subject, content.ConditionCharmed))
	case EndPetrified:
		return conditionChangeResult(base, c.Conditions.Remove(c.Subject, content.ConditionPetrified))
	case EndCurse:
		source := effectSourceOf(c.Effects, ch.ID)
		switch res := (CurseRemoval{}).Remove(c.Subject, c.Effects, ch.ID, GreaterRestorationSpell).(type) {
		case RemoveCurseApplied:
			base.effects, base.curseSource = res.effects, source
			for _, e := range res.events {
				base.events = append(base.events, e)
			}
			return base
		case RemoveCurseNotACurse:
			return GreaterRestorationNotACurse{}
		}
		return GreaterRestorationNothingToEnd{}
	case EndAbilityScoreReduction:
		return GreaterRestorationHostResolved{}
	case EndHitPointMaximumReduction:
		res, err := RestoreMaximum(c.Subject, c.Status, ch.Amount)
		if err != nil {
			return GreaterRestorationInvalid{}
		}
		switch r := res.(type) {
		case RestoreMaximumApplied:
			if len(r.events) == 0 {
				return GreaterRestorationNothingToEnd{}
			}
			base.status, base.events = r.status, r.events
			return base
		case RestoreMaximumRejected:
			return GreaterRestorationRejected(r)
		}
	}
	return GreaterRestorationInvalid{}
}

// conditionChangeResult turns the result of a condition removal into the answer of
// ApplyGreaterRestoration: Applied when a condition or level ended, NothingToEnd otherwise.
func conditionChangeResult(base GreaterRestorationApplied, res any) CastGreaterRestorationResult {
	r, ok := res.(interface {
		Set() ConditionSet
		Events() []ConditionEvent
	})
	if !ok {
		return GreaterRestorationNothingToEnd{}
	}
	base.conditions, base.events = r.Set(), conditionEventList(r.Events())
	return base
}

// effectSourceOf returns the source of the effect id in effects, the zero EffectSource when not held.
func effectSourceOf(effects OngoingEffects, id OngoingEffectID) EffectSource {
	e, _ := effects.Find(id) // not held gives the zero effect, whose zero source is the answer
	return e.source
}
