package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// UndeadFortitudeFacts is what the host knows of the damage that may trigger Undead
// Fortitude: the damage taken, whether any of it was Radiant and whether the hit was a
// Critical Hit. The zero value is invalid. Construct with NewUndeadFortitudeFacts.
// Comparable and safe for concurrent use.
type UndeadFortitudeFacts struct {
	damageTaken int
	radiant     bool
	hit         HitKind
}

// NewUndeadFortitudeFacts returns the facts of a hit. damageTaken is DamageTaken.Total of
// the hit point transition (the damage after the pipeline, Temporary Hit Points included);
// dealtTypes holds the Damage Type of each instance of the command whose amount was above
// 0 after the pipeline (Immunity gives 0, so that type is absent); hit says whether it was
// a Critical Hit.
//
// It returns an error matching srderr.ErrInvalidInput for a damageTaken below 1, an empty
// dealtTypes, a Damage Type that is not valid or a hit kind that is not valid. It never
// panics.
func NewUndeadFortitudeFacts(damageTaken int, dealtTypes []content.DamageType, hit HitKind) (UndeadFortitudeFacts, error) {
	if damageTaken < 1 || len(dealtTypes) == 0 || !hit.Valid() {
		return UndeadFortitudeFacts{}, invalidInput("undead fortitude needs damage taken, the damage types dealt and a hit kind")
	}
	if slices.ContainsFunc(dealtTypes, func(t content.DamageType) bool { return !t.Valid() }) {
		return UndeadFortitudeFacts{}, invalidInput("undead fortitude got an invalid Damage Type")
	}
	return UndeadFortitudeFacts{damageTaken, slices.Contains(dealtTypes, content.DamageRadiant), hit}, nil
}

// DamageTaken returns the damage the hit dealt.
func (f UndeadFortitudeFacts) DamageTaken() int { return f.damageTaken }

// IsZero reports whether f is the invalid zero value.
func (f UndeadFortitudeFacts) IsZero() bool { return f.damageTaken == 0 }

// UndeadFortitudeExemption says why Undead Fortitude does not trigger. The zero value is
// invalid. Comparable and safe for concurrent use.
type UndeadFortitudeExemption int

const (
	// UndeadFortitudeNotReducedToZero: the hit did not reduce the creature to 0 Hit Points
	// (it was above 0 after it, it was already at 0, or it knocked the creature out). Also
	// the answer for a zero trait or zero facts, which show no drop.
	UndeadFortitudeNotReducedToZero UndeadFortitudeExemption = iota + 1
	// UndeadFortitudeCriticalHit: the damage was from a Critical Hit.
	UndeadFortitudeCriticalHit
	// UndeadFortitudeRadiantDamage: the damage was Radiant.
	UndeadFortitudeRadiantDamage
)

// String returns "not_reduced_to_zero", "critical_hit", "radiant_damage" or "unknown".
func (e UndeadFortitudeExemption) String() string {
	return enumName([]string{"not_reduced_to_zero", "critical_hit", "radiant_damage"}, int(e))
}

// UndeadFortitudeCheck is the closed set of answers of AssessUndeadFortitude. Variants:
// UndeadFortitudeNotTriggered, UndeadFortitudeSaveRequired.
type UndeadFortitudeCheck interface{ isUndeadFortitudeCheck() }

// UndeadFortitudeNotTriggered: the trait does not apply. Construct with a keyed literal.
type UndeadFortitudeNotTriggered struct{ Reason UndeadFortitudeExemption }

// UndeadFortitudeSaveRequired: the creature makes a saving throw of Ability against DC;
// the host rolls it through the d20 package and settles it with SettleUndeadFortitude.
// Construct with a keyed literal.
type UndeadFortitudeSaveRequired struct {
	DC      d20.DifficultyClass
	Ability ability.Ability
}

func (UndeadFortitudeNotTriggered) isUndeadFortitudeCheck() {}
func (UndeadFortitudeSaveRequired) isUndeadFortitudeCheck() {}

// AssessUndeadFortitude says whether Undead Fortitude applies to a hit that produced
// transition (SRD 5.2.1, "Zombie", p. 343). It applies only when the transition holds a
// DroppedToZeroHitPoints, so damage at 0 Hit Points, a knock-out and a hit that leaves the
// creature above 0 never trigger it. A Critical Hit voids it (UndeadFortitudeCriticalHit
// first), then any Radiant instance in dealtTypes (interpretation: one command is one hit,
// so "the damage" is Radiant if any instance is). Otherwise it returns a Constitution
// saving throw against DC 5 plus the damage taken, Temporary Hit Points included. Nothing
// is rolled or changed; it never fails.
func AssessUndeadFortitude(t UndeadFortitudeTrait, transition HitPointTransitionApplied, facts UndeadFortitudeFacts) UndeadFortitudeCheck {
	switch {
	case t.IsZero() || facts.IsZero() || !droppedToZero(transition.events):
		return UndeadFortitudeNotTriggered{UndeadFortitudeNotReducedToZero}
	case facts.hit == CriticalHit:
		return UndeadFortitudeNotTriggered{UndeadFortitudeCriticalHit}
	case facts.radiant:
		return UndeadFortitudeNotTriggered{UndeadFortitudeRadiantDamage}
	}
	return UndeadFortitudeSaveRequired{DC: d20.DifficultyClass(satAdd(5, facts.damageTaken)), Ability: ability.Constitution}
}

// droppedToZero reports whether events holds a DroppedToZeroHitPoints.
func droppedToZero(events []HitPointEvent) bool {
	return slices.ContainsFunc(events, func(e HitPointEvent) bool { _, ok := e.(DroppedToZeroHitPoints); return ok })
}

// UndeadFortitudeSettlement is the closed set of results of SettleUndeadFortitude.
// Variants: UndeadFortitudeSurvived, UndeadFortitudeFell.
type UndeadFortitudeSettlement interface{ isUndeadFortitudeSettlement() }

// UndeadFortitudeSurvived: the saving throw succeeded and the creature is at 1 Hit Point.
type UndeadFortitudeSurvived struct {
	status HitPointStatus
	events []Event
}

// Status returns the creature at 1 Hit Point, Alive, with no Temporary Hit Points.
func (s UndeadFortitudeSurvived) Status() HitPointStatus { return s.status }

// Events returns a copy of the events in order: the hit's events without Died and
// FellUnconscious (DamageTaken and DroppedToZeroHitPoints stay: they report what the
// damage did before the trait acted), BecameBloodied when the 1 Hit Point is Bloodied and
// the creature was not, then UndeadFortitudeSaved, which supersedes the drop to 0.
func (s UndeadFortitudeSurvived) Events() []Event { return slices.Clone(s.events) }

// UndeadFortitudeFell: the saving throw failed and the hit's status and events stand.
type UndeadFortitudeFell struct {
	status HitPointStatus
	events []Event
}

// Status returns the status of the hit's transition, unchanged.
func (f UndeadFortitudeFell) Status() HitPointStatus { return f.status }

// Events returns a copy of the hit's events, unchanged.
func (f UndeadFortitudeFell) Events() []Event { return slices.Clone(f.events) }

func (UndeadFortitudeSurvived) isUndeadFortitudeSettlement() {}
func (UndeadFortitudeFell) isUndeadFortitudeSettlement()     {}

// SettleUndeadFortitude applies the result of the saving throw check required (SRD 5.2.1,
// "Zombie", p. 343). With d20.Success the creature drops to 1 Hit Point instead: a status
// rebuilt from before's maximum, rule and Damage Type adjustments, Alive, with no
// Temporary Hit Points (UndeadFortitudeSurvived). With d20.Failure the hit's status and
// events stand (UndeadFortitudeFell). before is the status before the hit and transition
// the hit's result; subject is the creature.
//
// It returns an error matching srderr.ErrInvalidInput when check is not
// UndeadFortitudeSaveRequired, before or the transition's status is zero, subject is zero,
// outcome is neither d20.Success nor d20.Failure, the transition's status is neither Dead
// nor Dying, or before is at 0 Hit Points. It never panics.
func SettleUndeadFortitude(check UndeadFortitudeCheck, before HitPointStatus, transition HitPointTransitionApplied, subject CombatantID, outcome d20.Outcome) (UndeadFortitudeSettlement, error) {
	required, ok := check.(UndeadFortitudeSaveRequired)
	if !ok || before.IsZero() || transition.status.IsZero() || subject.IsZero() || (outcome != d20.Success && outcome != d20.Failure) {
		return nil, invalidInput("settling undead fortitude needs a required save, both statuses, a subject and an outcome")
	}
	switch transition.status.life.(type) {
	case Dead, Dying:
	default:
		return nil, invalidInput("undead fortitude settles a creature that is Dead or Dying")
	}
	if before.hp.current == 0 {
		return nil, invalidInput("undead fortitude needs a creature that was above 0 Hit Points")
	}
	events := make([]Event, len(transition.events))
	for i, e := range transition.events {
		events[i] = e
	}
	if outcome == d20.Failure {
		return UndeadFortitudeFell{transition.status, events}, nil
	}
	status, err := RestoreHitPointStatus(HitPoints{1, transition.status.hp.maximum, 0}, Alive{}, before.rule, before.aff)
	if err != nil {
		return nil, err
	}
	kept := slices.DeleteFunc(events, func(e Event) bool {
		switch e.(type) {
		case Died, FellUnconscious, BecameBloodied:
			return true
		}
		return false
	})
	if !bloodied(before.hp.current, before.hp.maximum) && bloodied(1, status.hp.maximum) {
		kept = append(kept, BecameBloodied{hitPointEventBase{subject}})
	}
	kept = append(kept, UndeadFortitudeSaved{monsterTraitEventBase{subject}, required.DC})
	return UndeadFortitudeSurvived{status, kept}, nil
}
