package combat

// deathSaveRefusal says why a Death Saving Throw cannot be made, if it cannot.
func (s HitPointStatus) deathSaveRefusal() (HitPointRejection, bool) {
	switch s.life.(type) {
	case Dead:
		return CreatureIsDead, true
	case Dying:
		return 0, false
	}
	return NotDying, true
}

// ApplyDeathSave applies one Death Saving Throw of the creature subject (SRD
// 5.2.1, "Death Saving Throws", pp. 17-18). natural is the kept d20 (1 to 20)
// and total is the natural plus any modifier the host states, such as the
// Exhaustion reduction: a natural 1 is two failures and a natural 20 gives the
// creature 1 Hit Point (Alive, tallies reset, no success), whatever the total;
// otherwise a total of 10 or more is a success and a lower one a failure. The
// third success makes the creature Stable, the third failure Dead with cause
// DeathSavingThrowFailures. Events: DeathSavingThrowMade, then BecameStable, Died
// or HitPointsRestored and Recovered. The outcome is in the DeathSavingThrowMade
// event.
//
// It returns HitPointTransitionRefused(CreatureIsDead) for a Dead creature and
// HitPointTransitionRefused(NotDying) for an Alive or Stable one. It returns an
// error matching srderr.ErrInvalidInput for a zero subject or status or a
// natural outside 1 to 20. It never panics.
func (s HitPointStatus) ApplyDeathSave(subject CombatantID, natural, total int) (HitPointTransition, error) {
	if err := s.check(subject); err != nil {
		return nil, err
	}
	if natural < 1 || natural > 20 {
		return nil, invalidInput("death saving throw natural outside 1 to 20")
	}
	if r, no := s.deathSaveRefusal(); no {
		return refused(r), nil
	}
	base, hp, tally := hitPointEventBase{subject}, s.hp, s.life.(Dying)
	made := func(o DeathSavingThrowOutcome) HitPointEvent { return DeathSavingThrowMade{base, o} }
	die := func(o DeathSavingThrowOutcome) HitPointTransition {
		return applied(s.with(hp, Dead{DeathSavingThrowFailures}), made(o), Died{base, DeathSavingThrowFailures})
	}
	switch {
	case natural == 20:
		hp.current = 1
		return applied(s.with(hp, Alive{}), made(DeathSavingThrowRegainedHitPoint),
			HitPointsRestored{base, 1}, Recovered{base}), nil
	case natural == 1:
		if tally.failures+2 >= 3 {
			return die(DeathSavingThrowTwoFailures), nil
		}
		return applied(s.with(hp, Dying{tally.successes, tally.failures + 2}), made(DeathSavingThrowTwoFailures)), nil
	case total >= 10:
		if tally.successes+1 >= 3 {
			return applied(s.with(hp, Stable{}), made(DeathSavingThrowSuccess), BecameStable{base}), nil
		}
		return applied(s.with(hp, Dying{tally.successes + 1, tally.failures}), made(DeathSavingThrowSuccess)), nil
	}
	if tally.failures+1 >= 3 {
		return die(DeathSavingThrowFailure), nil
	}
	return applied(s.with(hp, Dying{tally.successes, tally.failures + 1}), made(DeathSavingThrowFailure)), nil
}

// Stabilize makes the Dying creature subject Stable with both tallies reset (SRD
// 5.2.1, "Stabilizing a Character", p. 18); the success of the Help action
// check is the host's input. Event: BecameStable.
//
// It returns HitPointTransitionRefused(CreatureIsDead) for a Dead creature and
// HitPointTransitionRefused(NotDying) for an Alive or Stable one. It returns an
// error matching srderr.ErrInvalidInput for a zero subject or status. It never
// panics.
func (s HitPointStatus) Stabilize(subject CombatantID) (HitPointTransition, error) {
	if err := s.check(subject); err != nil {
		return nil, err
	}
	if r, no := s.deathSaveRefusal(); no {
		return refused(r), nil
	}
	return applied(s.with(s.hp, Stable{}), BecameStable{hitPointEventBase{subject}}), nil
}

// RecoverStable gives the Stable creature subject 1 Hit Point and makes it Alive
// (SRD 5.2.1, "Stabilizing a Character", p. 18), once the host's delay is over.
// Events: HitPointsRestored (1), Recovered.
//
// It returns HitPointTransitionRefused(CreatureIsDead) for a Dead creature and
// HitPointTransitionRefused(NotStable) for one that is not Stable. It returns an
// error matching srderr.ErrInvalidInput for a zero subject or status. It never
// panics.
func (s HitPointStatus) RecoverStable(subject CombatantID) (HitPointTransition, error) {
	if err := s.check(subject); err != nil {
		return nil, err
	}
	if s.isDead() {
		return refused(CreatureIsDead), nil
	}
	if _, ok := s.life.(Stable); !ok {
		return refused(NotStable), nil
	}
	hp := s.hp
	hp.current = 1
	base := hitPointEventBase{subject}
	t := applied(s.with(hp, Alive{}), HitPointsRestored{base, 1}, Recovered{base})
	t.regained = 1
	return t, nil
}
