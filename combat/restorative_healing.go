package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// RestorativeSpell is a spell that restores Hit Points and ends conditions: Heal,
// Power Word Heal or Mass Heal (SRD 5.2.1, pp. 139, 154, 147). The zero value is not
// one.
type RestorativeSpell int

// The restorative healing spells.
const (
	// Heal restores 70 Hit Points, plus 10 per slot level above 6 (the host supplies the amount).
	Heal RestorativeSpell = iota + 1
	// PowerWordHeal restores all Hit Points.
	PowerWordHeal
	// MassHeal restores up to 700 Hit Points divided among creatures.
	MassHeal
)

// String returns "heal", "power_word_heal", "mass_heal" or "unknown".
func (s RestorativeSpell) String() string {
	return enumName([]string{"heal", "power_word_heal", "mass_heal"}, int(s))
}

// ConditionsEnded returns a new slice of the conditions the spell ends, in the order
// they are removed; nil for an invalid spell.
func (s RestorativeSpell) ConditionsEnded() []content.ConditionKind {
	switch s {
	case Heal, MassHeal:
		return []content.ConditionKind{content.ConditionBlinded, content.ConditionDeafened, content.ConditionPoisoned}
	case PowerWordHeal:
		return []content.ConditionKind{content.ConditionCharmed, content.ConditionFrightened, content.ConditionParalyzed,
			content.ConditionPoisoned, content.ConditionStunned}
	}
	return nil
}

// CastRestorativeHealingResult is the closed set of answers of ApplyRestorativeHealing.
// Variants: RestorativeHealingApplied, RestorativeHealingRejected,
// RestorativeHealingInvalidLevel.
type CastRestorativeHealingResult interface{ isCastRestorativeHealingResult() }

// RestorativeHealingApplied: the Hit Points were restored (possibly none) and the held
// conditions of the spell ended. Events holds, in order, the events of the Healing
// (HitPointsRestored, Recovered) and then one ConditionRemoved for each condition
// that ended.
type RestorativeHealingApplied struct {
	status     HitPointStatus
	conditions ConditionSet
	events     []Event
}

// Status returns the Hit Point status after the call.
func (r RestorativeHealingApplied) Status() HitPointStatus { return r.status }

// Conditions returns the condition set after the call.
func (r RestorativeHealingApplied) Conditions() ConditionSet { return r.conditions }

// Events returns a copy of the events in the order they happened.
func (r RestorativeHealingApplied) Events() []Event { return slices.Clone(r.events) }

// RestorativeHealingRejected: the creature is Dead, so nothing was restored or ended.
// Construct it with a keyed literal.
type RestorativeHealingRejected struct{ Rejection HitPointRejection }

// RestorativeHealingInvalidLevel: Heal was given a cast level below 6 or above 9;
// nothing changed. Construct it with a keyed literal.
type RestorativeHealingInvalidLevel struct{ CastLevel int }

func (RestorativeHealingApplied) isCastRestorativeHealingResult()      {}
func (RestorativeHealingRejected) isCastRestorativeHealingResult()     {}
func (RestorativeHealingInvalidLevel) isCastRestorativeHealingResult() {}

// ApplyRestorativeHealing restores Hit Points to the creature subject and ends the
// conditions of spell it holds explicitly, in the spell's order (SRD 5.2.1, "Heal",
// p. 139; "Power Word Heal", p. 154; "Mass Heal", p. 147). The Healing comes first
// through Restore, so a creature at 0 Hit Points becomes Alive with Recovered; the
// conditions end even when nothing was regained.
//
// The amount is, for Heal, the caller's amount from the spell's data (castLevel is
// only checked to be 6 to 9 and the package holds no formula); for Power Word Heal the
// status's Hit Point maximum (amount is ignored); for Mass Heal the creature's
// allotment, where 0 means the creature is not healed and nothing happens. Power
// Word Heal's Prone rule is left to the host.
//
// It decides in this order: Heal with a castLevel outside 6 to 9
// (RestorativeHealingInvalidLevel), Dead (RestorativeHealingRejected, nothing ended).
// It returns an error matching srderr.ErrInvalidInput for a spell that is not one
// of the three, a zero subject or status, and a negative amount. It never panics and
// changes none of its inputs.
func ApplyRestorativeHealing(spell RestorativeSpell, subject CombatantID, status HitPointStatus, set ConditionSet, amount, castLevel int) (CastRestorativeHealingResult, error) {
	kinds := spell.ConditionsEnded()
	if kinds == nil {
		return nil, invalidInput("not a restorative healing spell")
	}
	if err := status.check(subject); err != nil {
		return nil, err
	}
	if spell == Heal && (castLevel < 6 || castLevel > 9) {
		return RestorativeHealingInvalidLevel{castLevel}, nil
	}
	if status.isDead() {
		return RestorativeHealingRejected{CreatureIsDead}, nil
	}
	if spell == PowerWordHeal {
		amount = status.hp.maximum
	}
	if amount < 0 {
		return nil, invalidInput("negative healing")
	}
	if spell == MassHeal && amount == 0 {
		return RestorativeHealingApplied{status, set, nil}, nil
	}
	tr, err := status.Restore(subject, amount, nil)
	if err != nil {
		return nil, err
	}
	// The creature is not Dead, so Restore cannot refuse.
	a := tr.(HitPointTransitionApplied)
	var events []Event
	for _, e := range a.events {
		events = append(events, e)
	}
	for _, k := range kinds {
		if !slices.ContainsFunc(set.Explicit(), func(c Condition) bool { return c.Kind() == k }) {
			continue
		}
		r := set.Remove(subject, k)
		set = r.Set()
		events = append(events, conditionEvents(r)...)
	}
	return RestorativeHealingApplied{a.status, set, events}, nil
}
