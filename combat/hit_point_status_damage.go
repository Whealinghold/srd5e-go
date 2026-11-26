package combat

import (
	"errors"

	"github.com/Whealinghold/srd5e-go/character"
)

// bloodied reports whether current Hit Points are Bloodied for a maximum, through
// the one predicate of the module. A maximum below 1 only occurs for a Dead
// creature, which is never judged, so the false branch is unreachable.
func bloodied(current, maximum int) bool {
	m, err := character.NewHitPointMaximum(maximum)
	if err != nil {
		return false
	}
	b, err := character.IsBloodied(current, m)
	return err == nil && b
}

// check rejects a zero status or subject, the programmer errors every operation
// shares.
func (s HitPointStatus) check(subject CombatantID) error {
	if subject.IsZero() {
		return invalidInput("operation without a subject")
	}
	if s.IsZero() {
		return invalidInput("operation on a zero hit point status")
	}
	return nil
}

func (s HitPointStatus) isDead() bool {
	_, ok := s.life.(Dead)
	return ok
}

func refused(r HitPointRejection) HitPointTransitionRefused {
	return HitPointTransitionRefused{Rejection: r}
}

// with returns s with new Hit Points and life state.
func (s HitPointStatus) with(hp HitPoints, life LifeState) HitPointStatus {
	s.hp, s.life = hp, life
	return s
}

// ApplyDamage applies damage to the creature subject (SRD 5.2.1, "Hit Points"
// to "Damage at 0 Hit Points", pp. 16-18).
//
// Each Damage (and each unpreventable one, which skips the adjustments, Half
// Damage, Resistance and Immunity but takes Vulnerability) goes through the
// pipeline alone; the post-pipeline amounts are summed, Temporary Hit Points
// absorb first, and the zero-Hit-Point rules (a Death Saving Throw failure, two
// from a CriticalHit, and Massive Damage) apply once to that sum. The events
// are those of hit-points-and-damage.md table 4.9a, in order. HitPointsLost of
// DamageTaken is the sum less what Temporary Hit Points absorbed, overkill
// included. A MeleeAttackKnockOut that would drop an Alive creature to 0 Hit
// Points leaves it at 1.
//
// It returns HitPointTransitionRefused(CreatureIsDead) for a Dead creature. It
// returns an error matching srderr.ErrInvalidInput for a zero subject or
// status, no Damage at all, an invalid Damage, adjustment, delivery or kind. It returns a LimitExceededError for more
// than 64 Damage instances (damages and unpreventable together) or more than 64 adjustments (plan P10). It
// never panics; sums saturate at math.MaxInt.
func (s HitPointStatus) ApplyDamage(subject CombatantID, damages []Damage, adjustments []DamageAdjustment,
	delivery DamageDelivery, kind HitKind, unpreventable []Damage) (HitPointTransition, error) {
	if err := s.check(subject); err != nil {
		return nil, err
	}
	if err := errors.Join(overCap("damage instances", len(damages)+len(unpreventable), maxDamageParts),
		overCap("damage adjustments", len(adjustments), maxDamageParts)); err != nil {
		return nil, err
	}
	if len(damages)+len(unpreventable) == 0 || !delivery.Valid() || !kind.Valid() {
		return nil, invalidInput("apply damage needs damage, a delivery and a hit kind")
	}
	for _, d := range append(append([]Damage(nil), damages...), unpreventable...) {
		if d.amount < 0 || !d.typ.Valid() {
			return nil, invalidInput("apply damage got an invalid Damage")
		}
	}
	for _, a := range adjustments {
		if a == nil {
			return nil, invalidInput("apply damage got a nil adjustment")
		}
	}
	if s.isDead() {
		return refused(CreatureIsDead), nil
	}
	var resolutions []DamageResolution
	total := 0
	for i, d := range append(append([]Damage(nil), damages...), unpreventable...) {
		r := resolveDamage(d, adjustments, s.aff, i >= len(damages))
		resolutions = append(resolutions, r)
		total = satAdd(total, r.Dealt())
	}
	if total == 0 {
		t := applied(s)
		t.resolutions = resolutions
		return t, nil
	}
	return s.takeDamage(subject, resolutions, total, delivery, kind), nil
}

// takeDamage applies a summed, post-pipeline total above 0 to a creature that is
// not Dead.
func (s HitPointStatus) takeDamage(subject CombatantID, resolutions []DamageResolution, total int, delivery DamageDelivery, kind HitKind) HitPointTransitionApplied {
	base := hitPointEventBase{subject}
	hp := s.hp
	absorbed := min(hp.temporary, total)
	remainder := total - absorbed
	hp.temporary -= absorbed
	events := []HitPointEvent{DamageTaken{base, resolutions, absorbed, remainder}}
	if absorbed > 0 && hp.temporary == 0 {
		events = append(events, TemporaryHitPointsEnded{base})
	}
	finish := func(life LifeState, more ...HitPointEvent) HitPointTransitionApplied {
		t := applied(s.with(hp, life), append(events, more...)...)
		t.resolutions = resolutions
		return t
	}
	kill := func(cause DeathCause, more ...HitPointEvent) HitPointTransitionApplied {
		hp.current = 0
		return finish(Dead{cause}, append(more, Died{base, cause})...)
	}
	if remainder == 0 { // fully absorbed: damage taken, nothing else changes
		return finish(s.life)
	}
	var more []HitPointEvent
	life := s.life
	switch cur := s.life.(type) {
	case Alive:
		before, current := bloodied(hp.current, hp.maximum), hp.current
		hp.current = max(0, current-remainder)
		knockedOut := hp.current == 0 && delivery == MeleeAttackKnockOut
		switch {
		case knockedOut:
			hp.current = 1
		case hp.current > 0:
		case s.rule == DiesInstantly:
			return kill(MonsterDeath, DroppedToZeroHitPoints{base})
		case remainder-current >= hp.maximum:
			return kill(MassiveDamage, DroppedToZeroHitPoints{base})
		default:
			life = Dying{}
		}
		if !before && bloodied(hp.current, hp.maximum) {
			more = append(more, BecameBloodied{base})
		}
		switch {
		case knockedOut:
			more = append(more, KnockedOut{base})
		case hp.current == 0:
			more = append(more, DroppedToZeroHitPoints{base}, FellUnconscious{base})
		}
	default: // Dying or Stable: the rules of "Damage at 0 Hit Points"
		tally, isDying := cur.(Dying)
		if !isDying {
			more = append(more, LostStability{base}) // a Stable creature starts again with no tallies
		}
		failures := tally.failures + 1
		if kind == CriticalHit {
			failures++
		}
		switch {
		case remainder >= hp.maximum:
			return kill(MassiveDamage, more...)
		case failures >= 3:
			return kill(DeathSavingThrowFailures, more...)
		}
		life = Dying{tally.successes, failures}
	}
	return finish(life, more...)
}
