package combat

// Restore gives the creature subject back Hit Points (SRD 5.2.1, "Healing",
// p. 17). Current Hit Points rise by amount but never above the Hit Point
// maximum or the ceiling (nil is NoHealingCeiling); a current already above a
// ceiling is never lowered. Temporary Hit Points are untouched. Regaining any Hit
// Points at 0 makes the creature Alive and resets its tallies. Events:
// HitPointsRestored, then Recovered when it was at 0. Healing that regains
// nothing changes nothing and emits nothing; HitPointsRegained of the result is
// the amount regained.
//
// It returns HitPointTransitionRefused(CreatureIsDead) for a Dead creature. It
// returns an error matching srderr.ErrInvalidInput for a zero subject or
// status, a negative amount or a ceiling that is neither of the two variants. It
// never panics.
func (s HitPointStatus) Restore(subject CombatantID, amount int, ceiling HealingCeiling) (HitPointTransition, error) {
	if amount < 0 {
		return nil, invalidInput("negative healing")
	}
	limit := s.hp.maximum
	switch c := ceiling.(type) {
	case nil, NoHealingCeiling:
	case HealingCeilingAt:
		limit = min(limit, c.hitPoints)
	default:
		return nil, invalidInput("unknown healing ceiling")
	}
	if err := s.check(subject); err != nil {
		return nil, err
	}
	if s.isDead() {
		return refused(CreatureIsDead), nil
	}
	regained := max(0, min(amount, limit-s.hp.current))
	if regained == 0 {
		return applied(s), nil
	}
	hp, life := s.hp, s.life
	atZero := hp.current == 0
	hp.current += regained
	events := []HitPointEvent{HitPointsRestored{hitPointEventBase{subject}, regained}}
	if atZero {
		life = Alive{}
		events = append(events, Recovered{hitPointEventBase{subject}})
	}
	t := applied(s.with(hp, life), events...)
	t.regained = regained
	return t, nil
}

// GainTemporary gives the creature subject Temporary Hit Points (SRD 5.2.1,
// "Temporary Hit Points", p. 18). They do not stack: with none held the amount is
// set; with some held the choice decides between the held and the new amount,
// and an offer equal to the held amount changes nothing. They are not Healing,
// do not raise current Hit Points and do not change the LifeState. Event:
// TemporaryHitPointsGained when the amount changed, none otherwise.
//
// It returns HitPointTransitionRefused(CreatureIsDead) for a Dead creature. It
// returns an error matching srderr.ErrInvalidInput for a zero subject or status,
// an amount below 1 or a choice that is not valid. It never panics.
func (s HitPointStatus) GainTemporary(subject CombatantID, amount int, choice TemporaryHitPointsChoice) (HitPointTransition, error) {
	if amount < 1 || !choice.Valid() {
		return nil, invalidInput("temporary hit points need an amount of 1 or more and a choice")
	}
	if err := s.check(subject); err != nil {
		return nil, err
	}
	if s.isDead() {
		return refused(CreatureIsDead), nil
	}
	held := s.hp.temporary
	if held == amount || (held > 0 && choice == KeepCurrent) {
		return applied(s), nil
	}
	hp := s.hp
	hp.temporary = amount
	return applied(s.with(hp, s.life), TemporaryHitPointsGained{hitPointEventBase{subject}, amount}), nil
}

// EndTemporary ends the creature subject's Temporary Hit Points, as when a Long
// Rest is finished (SRD 5.2.1, "Duration", p. 18). Event:
// TemporaryHitPointsEnded when some were held, none otherwise.
//
// It returns HitPointTransitionRefused(CreatureIsDead) for a Dead creature. It
// returns an error matching srderr.ErrInvalidInput for a zero subject or status.
// It never panics.
func (s HitPointStatus) EndTemporary(subject CombatantID) (HitPointTransition, error) {
	if err := s.check(subject); err != nil {
		return nil, err
	}
	if s.isDead() {
		return refused(CreatureIsDead), nil
	}
	if s.hp.temporary == 0 {
		return applied(s), nil
	}
	hp := s.hp
	hp.temporary = 0
	return applied(s.with(hp, s.life), TemporaryHitPointsEnded{hitPointEventBase{subject}}), nil
}

// ReduceMaximum lowers the creature subject's Hit Point maximum by amount, not
// below 0, and its current Hit Points to the new maximum if lower (SRD 5.2.1,
// "Hit Point Maximum of 0", p. 17). A maximum that reaches 0 makes the creature
// Dead with cause HitPointMaximumZero. Events: HitPointMaximumReduced, then
// Died. An amount of 0 changes nothing and emits nothing.
//
// It returns HitPointTransitionRefused(CreatureIsDead) for a Dead creature. It
// returns an error matching srderr.ErrInvalidInput for a zero subject or status
// or a negative amount. It never panics.
func (s HitPointStatus) ReduceMaximum(subject CombatantID, amount int) (HitPointTransition, error) {
	if amount < 0 {
		return nil, invalidInput("negative maximum reduction")
	}
	if err := s.check(subject); err != nil {
		return nil, err
	}
	if s.isDead() {
		return refused(CreatureIsDead), nil
	}
	if amount == 0 {
		return applied(s), nil
	}
	base, hp, life := hitPointEventBase{subject}, s.hp, s.life
	hp.maximum = max(0, s.hp.maximum-amount)
	hp.current = min(hp.current, hp.maximum)
	events := []HitPointEvent{HitPointMaximumReduced{base, s.hp.maximum, hp.maximum}}
	if hp.maximum == 0 {
		life = Dead{HitPointMaximumZero}
		events = append(events, Died{base, HitPointMaximumZero})
	}
	return applied(s.with(hp, life), events...), nil
}

// RestoreMaximum returns a reduced Hit Point maximum to normalMaximum, the
// value the host states (SRD 5.2.1, "Long Rest", p. 185). A maximum is never
// lowered, so a normalMaximum at or below the current one changes nothing.
// Current Hit Points, the LifeState and Temporary Hit Points stay. Event:
// HitPointMaximumRestored when the maximum rose.
//
// It returns HitPointTransitionRefused(CreatureIsDead) for a Dead creature. It
// returns an error matching srderr.ErrInvalidInput for a zero subject or status
// or a normalMaximum below 1. It never panics.
func (s HitPointStatus) RestoreMaximum(subject CombatantID, normalMaximum int) (HitPointTransition, error) {
	if normalMaximum < 1 {
		return nil, invalidInput("normal maximum below 1")
	}
	if err := s.check(subject); err != nil {
		return nil, err
	}
	if s.isDead() {
		return refused(CreatureIsDead), nil
	}
	if normalMaximum <= s.hp.maximum {
		return applied(s), nil
	}
	hp := s.hp
	hp.maximum = normalMaximum
	return applied(s.with(hp, s.life), HitPointMaximumRestored{hitPointEventBase{subject}, s.hp.maximum, normalMaximum}), nil
}

// RecordExhaustionDeath records the death of a creature whose Exhaustion reached
// level 6 (SRD 5.2.1, Rules Glossary, "Exhaustion", p. 181). It makes a creature
// that is not Dead Dead with cause Exhaustion and emits Died.
//
// It returns HitPointTransitionRefused(CreatureIsDead) for a Dead creature. It
// returns an error matching srderr.ErrInvalidInput for a zero subject or status.
// It never panics.
func (s HitPointStatus) RecordExhaustionDeath(subject CombatantID) (HitPointTransition, error) {
	if err := s.check(subject); err != nil {
		return nil, err
	}
	if s.isDead() {
		return refused(CreatureIsDead), nil
	}
	hp := s.hp
	hp.current = 0
	return applied(s.with(hp, Dead{Exhaustion}), Died{hitPointEventBase{subject}, Exhaustion}), nil
}
