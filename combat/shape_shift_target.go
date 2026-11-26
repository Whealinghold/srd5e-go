package combat

import "github.com/Whealinghold/srd5e-go/d20"

// TargetResponse is the target's part in a shape-shifting spell: willing, or unwilling with
// the outcome of its Wisdom saving throw. A willing target may choose to fail the save
// without rolling (SRD 5.2.1, "Saving Throws", p. 7). The zero (nil) TargetResponse is
// invalid. Variants: Willing, Unwilling.
type TargetResponse interface{ isTargetResponse() }

// Willing: the target accepts the spell and needs no roll.
type Willing struct{}

// Unwilling: the target resists and its saving throw has the outcome Save. Construct it with a
// keyed literal.
type Unwilling struct{ Save d20.Outcome }

func (Willing) isTargetResponse()   {}
func (Unwilling) isTargetResponse() {}

// TargetHitPoints is the Hit Point side of a target: its tracked status and the choice for
// Temporary Hit Points that do not stack, or no status at all, which is accepted exactly
// where the spell or mode grants no Temporary Hit Points. The zero (nil) TargetHitPoints is
// invalid. Variants: TrackedHitPoints, UntrackedHitPoints.
type TargetHitPoints interface{ isTargetHitPoints() }

// TrackedHitPoints is a target whose Hit Points the host tracks. Construct it with a keyed
// literal. Choice is read only when the form grants Temporary Hit Points.
type TrackedHitPoints struct {
	// Status is the target's hit point status; the zero status is invalid.
	Status HitPointStatus
	// Choice says which Temporary Hit Points the target keeps when it already holds some.
	Choice TemporaryHitPointsChoice
}

// UntrackedHitPoints is a target with no hit point status, such as an object. Life is its life
// state; nil means Alive. Construct it with a keyed literal.
type UntrackedHitPoints struct{ Life LifeState }

func (TrackedHitPoints) isTargetHitPoints()   {}
func (UntrackedHitPoints) isTargetHitPoints() {}

// valid reports whether hp is one of the two variants, with a usable status when tracked.
func validTargetHitPoints(hp TargetHitPoints) bool {
	switch h := hp.(type) {
	case TrackedHitPoints:
		return !h.Status.IsZero()
	case UntrackedHitPoints:
		return true
	}
	return false
}

// lifeOf returns the life state of the target.
func lifeOf(hp TargetHitPoints) LifeState {
	switch h := hp.(type) {
	case TrackedHitPoints:
		return h.Status.LifeState()
	case UntrackedHitPoints:
		return h.Life
	}
	return nil
}

func targetIsDead(hp TargetHitPoints) bool {
	_, dead := lifeOf(hp).(Dead)
	return dead
}

// statusOf returns the tracked status, and the zero status for an untracked target.
func statusOf(hp TargetHitPoints) HitPointStatus {
	if h, ok := hp.(TrackedHitPoints); ok {
		return h.Status
	}
	return HitPointStatus{}
}

// AnimatedObjectTerms are the terms of an object that True Polymorph turned into a creature
// (SRD 5.2.1, "True Polymorph", p. 170): it is Friendly to the caster and the caster's allies,
// takes its turns immediately after the caster's and obeys the caster's commands. The zero
// value holds no terms (see IsZero). Comparable and immutable.
type AnimatedObjectTerms struct{ caster CombatantID }

// IsZero reports whether a holds no terms.
func (a AnimatedObjectTerms) IsZero() bool { return a.caster.IsZero() }

// Combatants returns the creature the animated object serves, as a new one-element slice, and
// nothing for the zero terms.
func (a AnimatedObjectTerms) Combatants() []CombatantID {
	if a.IsZero() {
		return nil
	}
	return []CombatantID{a.caster}
}

// IsFriendlyToCasterAndAllies reports that the creature is Friendly to the caster and the caster's allies.
func (a AnimatedObjectTerms) IsFriendlyToCasterAndAllies() bool { return !a.IsZero() }

// TakesTurnsAfterCaster reports that the creature takes its turns immediately after the caster's.
func (a AnimatedObjectTerms) TakesTurnsAfterCaster() bool { return !a.IsZero() }

// ObeysCaster reports that the creature obeys the caster's commands.
func (a AnimatedObjectTerms) ObeysCaster() bool { return !a.IsZero() }

// successfulSave is the d20 outcome of a saving throw the target passed.
const successfulSave = d20.Success

// validSave reports whether o is a saving throw outcome.
func validSave(o d20.Outcome) bool { return o == d20.Success || o == d20.Failure }
