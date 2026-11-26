package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// RegenerationState is what changes for a creature with Regeneration: whether its trait is
// suppressed until the start of its next turn. The host stores it with the creature's
// HitPointStatus, and every operation returns a new state (SRD 5.2.1, "Troll", p. 333).
// The zero value is invalid (see IsZero). Create it with NewRegenerationState. Comparable
// and safe for concurrent use.
type RegenerationState struct {
	trait      RegenerationTrait
	suppressed bool
}

// NewRegenerationState returns the state of trait, not suppressed. It returns an error
// matching srderr.ErrInvalidInput for a zero trait. It never panics.
func NewRegenerationState(t RegenerationTrait) (RegenerationState, error) {
	if t.IsZero() {
		return RegenerationState{}, invalidInput("a regeneration state needs a Regeneration trait")
	}
	return RegenerationState{trait: t}, nil
}

// IsZero reports whether s is the invalid zero state.
func (s RegenerationState) IsZero() bool { return s.trait.IsZero() }

// Trait returns the trait.
func (s RegenerationState) Trait() RegenerationTrait { return s.trait }

// Suppressed reports whether the trait does not function at the start of the next turn.
func (s RegenerationState) Suppressed() bool { return s.suppressed }

// RegenerationStep is the result of AfterDamage: the new state and zero or one
// RegenerationSuppressed event.
type RegenerationStep struct {
	state  RegenerationState
	events []MonsterTraitEvent
}

// State returns the new state.
func (s RegenerationStep) State() RegenerationState { return s.state }

// Events returns a copy of the events: RegenerationSuppressed only when the trait became
// suppressed.
func (s RegenerationStep) Events() []MonsterTraitEvent { return slices.Clone(s.events) }

// AfterDamage records damage dealt to the creature subject (SRD 5.2.1, "Troll", p. 333). A
// Damage Type in dealt that the trait lists suppresses it: the next turn start skips
// Regeneration. The event RegenerationSuppressed is added only when the trait was not
// already suppressed. The first form (no suppressing type), a dealt list with no matching
// type, a zero state and a zero subject return the state unchanged and no event. Pass the
// types of the instances that dealt more than 0 damage. It never fails.
func (s RegenerationState) AfterDamage(subject CombatantID, dealt []content.DamageType) RegenerationStep {
	if s.IsZero() || subject.IsZero() || s.suppressed || !s.trait.suppressedBy(dealt) {
		return RegenerationStep{state: s}
	}
	s.suppressed = true
	return RegenerationStep{s, []MonsterTraitEvent{RegenerationSuppressed{monsterTraitEventBase{subject}}}}
}

// RegenerationSkipReason says why Regeneration did not heal at a turn start. The zero value
// is invalid. Comparable and safe for concurrent use.
type RegenerationSkipReason int

const (
	// RegenerationSkipSuppressed: a suppressing Damage Type was taken since the last turn.
	RegenerationSkipSuppressed RegenerationSkipReason = iota + 1
	// RegenerationSkipNoHitPoints: the creature is Dead, or (first form) below 1 Hit Point.
	// Also the answer for a zero state, status or subject.
	RegenerationSkipNoHitPoints
	// RegenerationSkipSwarmCannotRegain: the monster is a swarm, which cannot regain Hit Points.
	RegenerationSkipSwarmCannotRegain
)

// String returns "suppressed", "no_hit_points", "swarm_cannot_regain" or "unknown".
func (r RegenerationSkipReason) String() string {
	return enumName([]string{"suppressed", "no_hit_points", "swarm_cannot_regain"}, int(r))
}

// RegenerationTurn is the closed set of results of RegenerationState.AtStartOfTurn. Every
// variant carries the resulting state. Variants: RegenerationHealed, RegenerationSkipped,
// RegenerationLetDie.
type RegenerationTurn interface {
	// State returns the resulting state.
	State() RegenerationState
	isRegenerationTurn()
}

// RegenerationHealed: the creature regained Hit Points (possibly 0 at its maximum). The
// transition is that of HitPointStatus.Restore.
type RegenerationHealed struct {
	state      RegenerationState
	transition HitPointTransitionApplied
}

// State returns the unchanged state.
func (h RegenerationHealed) State() RegenerationState { return h.state }

// Transition returns the result of the healing: the new status, the events
// (HitPointsRestored, and Recovered when the creature was at 0) and the Hit Points regained.
func (h RegenerationHealed) Transition() HitPointTransitionApplied { return h.transition }

// RegenerationSkipped: Regeneration did not function; the creature's status is unchanged.
type RegenerationSkipped struct {
	state  RegenerationState
	reason RegenerationSkipReason
}

// State returns the resulting state (not suppressed after a skip for suppression).
func (s RegenerationSkipped) State() RegenerationState { return s.state }

// Reason returns why it did not function.
func (s RegenerationSkipped) Reason() RegenerationSkipReason { return s.reason }

// RegenerationLetDie: the creature started its turn at 0 Hit Points and did not
// regenerate, so it dies. The status is Dead (MonsterDeath) at 0 Hit Points.
type RegenerationLetDie struct {
	state  RegenerationState
	status HitPointStatus
	events []Event
}

// State returns the resulting state (not suppressed).
func (d RegenerationLetDie) State() RegenerationState { return d.state }

// Status returns the Dead status.
func (d RegenerationLetDie) Status() HitPointStatus { return d.status }

// Events returns a copy of the events: Died with the cause MonsterDeath.
func (d RegenerationLetDie) Events() []Event { return slices.Clone(d.events) }

func (RegenerationHealed) isRegenerationTurn()  {}
func (RegenerationSkipped) isRegenerationTurn() {}
func (RegenerationLetDie) isRegenerationTurn()  {}

// AtStartOfTurn applies Regeneration at the start of the creature subject's turn (SRD
// 5.2.1, "Oni", p. 312; "Troll", p. 333). When it functions the creature regains the
// trait's amount through HitPointStatus.Restore (never above the maximum; at the maximum it
// regains 0 and adds no event) and the state is unchanged (RegenerationHealed).
//
// It returns RegenerationSkipped, with status unchanged, for the first that applies: the
// monster is a swarm (RegenerationSkipSwarmCannotRegain); the creature is Dead, or, in the
// first form only, has less than 1 Hit Point (RegenerationSkipNoHitPoints; the second form
// qualifies at 0 Hit Points, Stable included); the trait is suppressed
// (RegenerationSkipSuppressed, once: the returned state is not suppressed). A suppressed
// second-form creature at 0 Hit Points does not regenerate and dies: RegenerationLetDie.
// A zero state, status or subject is skipped as RegenerationSkipNoHitPoints. It never fails.
func (s RegenerationState) AtStartOfTurn(status HitPointStatus, subject CombatantID) RegenerationTurn {
	_, dead := status.life.(Dead)
	switch {
	case s.IsZero() || status.IsZero() || subject.IsZero():
		return RegenerationSkipped{s, RegenerationSkipNoHitPoints}
	case s.trait.swarm:
		return RegenerationSkipped{s, RegenerationSkipSwarmCannotRegain}
	case dead || (!s.trait.hasSuppression() && status.hp.current < 1):
		return RegenerationSkipped{s, RegenerationSkipNoHitPoints}
	case s.suppressed:
		s.suppressed = false
		if status.hp.current > 0 {
			return RegenerationSkipped{s, RegenerationSkipSuppressed}
		}
		hp := status.hp
		return RegenerationLetDie{s, status.with(hp, Dead{MonsterDeath}), []Event{Died{hitPointEventBase{subject}, MonsterDeath}}}
	}
	t, err := status.Restore(subject, s.trait.amount, nil)
	healed, ok := t.(HitPointTransitionApplied)
	if err != nil || !ok { // unreachable: the status is not zero and not Dead
		return RegenerationSkipped{s, RegenerationSkipNoHitPoints}
	}
	return RegenerationHealed{s, healed}
}

// RegenerationNotDeferredReason says why DeferRegenerationDeath did not defer a death. The
// zero value is invalid. Comparable and safe for concurrent use.
type RegenerationNotDeferredReason int

const (
	// RegenerationNoDeferral: the trait is the first form, which has no deferral, or is the
	// zero trait.
	RegenerationNoDeferral RegenerationNotDeferredReason = iota + 1
	// RegenerationNoDropToZero: the transition holds no DroppedToZeroHitPoints.
	RegenerationNoDropToZero
	// RegenerationCauseNotDeferrable: the creature is not Dying or Dead of Monster Death or
	// Massive Damage (a Hit Point maximum of 0 and Exhaustion are never deferred).
	RegenerationCauseNotDeferrable
)

// String returns "no_deferral", "no_drop_to_zero", "cause_not_deferrable" or "unknown".
func (r RegenerationNotDeferredReason) String() string {
	return enumName([]string{"no_deferral", "no_drop_to_zero", "cause_not_deferrable"}, int(r))
}

// RegenerationDeferral is the closed set of results of DeferRegenerationDeath. Variants:
// RegenerationDeferred, RegenerationNotDeferred.
type RegenerationDeferral interface{ isRegenerationDeferral() }

// RegenerationDeferred: the creature did not die; it is Stable at 0 Hit Points until its
// next turn start.
type RegenerationDeferred struct {
	status HitPointStatus
	events []Event
}

// Status returns the Stable status at 0 Hit Points, with the same maximum, rule and
// Damage Type adjustments.
func (d RegenerationDeferred) Status() HitPointStatus { return d.status }

// Events returns a copy of the events: the transition's without Died, FellUnconscious
// exactly once, then RegenerationDeathDeferred.
func (d RegenerationDeferred) Events() []Event { return slices.Clone(d.events) }

// RegenerationNotDeferred: the death stands. Construct with a keyed literal.
type RegenerationNotDeferred struct{ Reason RegenerationNotDeferredReason }

func (RegenerationDeferred) isRegenerationDeferral()    {}
func (RegenerationNotDeferred) isRegenerationDeferral() {}

// DeferRegenerationDeath keeps a second-form Regeneration creature from dying when it drops
// to 0 Hit Points (SRD 5.2.1, "Troll", p. 333: it "dies only if it starts its turn with 0
// Hit Points and doesn't regenerate"). When transition holds a DroppedToZeroHitPoints and
// its status is Dying or Dead of MonsterDeath or MassiveDamage, it returns
// RegenerationDeferred. Interpretation: the SRD names no state for a creature at 0 Hit Points
// that is not dead; Stable (0 Hit Points, Unconscious, no Death Saving Throws) is the one
// that fits and lets Healing return it to Alive. Otherwise it returns RegenerationNotDeferred
// with the first reason that applies. It never fails; subject is the creature.
func DeferRegenerationDeath(t RegenerationTrait, subject CombatantID, transition HitPointTransitionApplied) RegenerationDeferral {
	if t.IsZero() || subject.IsZero() || !t.hasSuppression() {
		return RegenerationNotDeferred{RegenerationNoDeferral}
	}
	if !droppedToZero(transition.events) {
		return RegenerationNotDeferred{RegenerationNoDropToZero}
	}
	switch l := transition.status.life.(type) {
	case Dying:
	case Dead:
		if l.cause != MonsterDeath && l.cause != MassiveDamage {
			return RegenerationNotDeferred{RegenerationCauseNotDeferrable}
		}
	default:
		return RegenerationNotDeferred{RegenerationCauseNotDeferrable}
	}
	hp := transition.status.hp
	hp.current = 0
	status, err := RestoreHitPointStatus(hp, Stable{}, transition.status.rule, transition.status.aff)
	if err != nil { // unreachable: a creature that dropped to 0 has a maximum of at least 1
		return RegenerationNotDeferred{RegenerationCauseNotDeferrable}
	}
	events := make([]Event, 0, len(transition.events)+2)
	fell := false
	for _, e := range transition.events {
		switch e.(type) {
		case Died:
			continue
		case FellUnconscious:
			fell = true
		}
		events = append(events, e)
	}
	if !fell {
		events = append(events, FellUnconscious{hitPointEventBase{subject}})
	}
	events = append(events, RegenerationDeathDeferred{monsterTraitEventBase{subject}})
	return RegenerationDeferred{status, events}
}
