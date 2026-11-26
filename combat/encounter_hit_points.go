package combat

import "github.com/Whealinghold/srd5e-go/content"

// becomeUnconscious applies Unconscious to c through ConditionSet.Apply. When it is
// added it returns the events; when it is already held or Immune it changes nothing and
// returns none (FR-027, FR-067). mark sets the knocked-out marker, only when added.
func (c *Combatant) becomeUnconscious(id CombatantID, mark bool) []ConditionEvent {
	added, ok := c.conditions.Apply(id, Unconscious{}).(ConditionAdded)
	if !ok {
		return nil
	}
	c.conditions = added.set
	c.knockedOut = c.knockedOut || mark
	return added.events
}

// wake removes Unconscious from c when it is held, which leaves it Prone, and clears the
// knocked-out marker (FR-031, FR-092, FR-098).
func (c *Combatant) wake(id CombatantID) []ConditionEvent {
	c.knockedOut = false
	cleared, ok := c.conditions.Remove(id, content.ConditionUnconscious).(ConditionCleared)
	if !ok {
		return nil
	}
	c.conditions = cleared.set
	return cleared.events
}

// settle commits an applied hit point transition to the combatant at i and adds the
// Unconscious consequences the root owns: it is applied after KnockedOut (marking the
// combatant) and after FellUnconscious (clearing the marker), and removed when the
// combatant returns from 0 Hit Points or a knocked-out combatant regains Hit Points.
// mark changes the combatant before the conditions are decided.
func (e Encounter) settle(i int, id CombatantID, a HitPointTransitionApplied, mark func(*Combatant)) encounterTransition {
	c := e.combatants[i]
	c.hp = a.status
	if mark != nil {
		mark(&c)
	}
	var conditions []ConditionEvent
	switch {
	case has[KnockedOut](a.events):
		conditions = c.becomeUnconscious(id, true)
	case has[FellUnconscious](a.events):
		c.knockedOut = false
		conditions = c.becomeUnconscious(id, false)
	case has[Recovered](a.events) || (has[HitPointsRestored](a.events) && c.knockedOut):
		conditions = c.wake(id)
	}
	events := append(e.wrapHitPoint(a.events), e.wrapCondition(conditions)...)
	return transitionApplied{e.replace(i, c), events, a.resolutions}
}

// commit settles the transition tr of the hit point rules for the combatant at i: an
// applied one as settle does, a refusal as HitPointRuleRejection.
func (e Encounter) commit(i int, id CombatantID, tr HitPointTransition, mark func(*Combatant)) encounterTransition {
	switch t := tr.(type) {
	case HitPointTransitionApplied:
		return e.settle(i, id, t, mark)
	default:
		return e.reject(HitPointRuleRejection{id, tr.(HitPointTransitionRefused).Rejection})
	}
}

// ApplyDamage applies damage to a living combatant through HitPointStatus.ApplyDamage
// and wraps the hit point events (SRD 5.2.1, "Damage Rolls", pp. 16-18). When the
// combatant falls to 0 Hit Points and is Dying it applies Unconscious through
// ConditionSet.Apply after them; a knock-out applies Unconscious after KnockedOut and
// marks the combatant. Resolutions carries one DamageResolution per Damage.
//
// It returns, first that applies, EncounterAlreadyEndedRejection,
// UnknownCombatantRejection and CombatantIsDeadRejection. It returns an error matching
// srderr.ErrInvalidInput for an invalid encounter or id, no Damage, an invalid Damage,
// nil adjustment, delivery or kind. It returns a LimitExceededError for more than 64 damage instances or 64 adjustments.
func (e Encounter) ApplyDamage(id CombatantID, damages []Damage, adjustments []DamageAdjustment, delivery DamageDelivery, kind HitKind) (encounterTransition, error) {
	if err := checkDamageInput(id, damages, adjustments, delivery, kind); err != nil {
		return nil, err
	}
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	tr, err := e.combatants[i].hp.ApplyDamage(id, damages, adjustments, delivery, kind, nil)
	if err != nil {
		return nil, err
	}
	return e.commit(i, id, tr, nil), nil
}

// rejected turns the result of target into a transition: the error, or the rejection.
func (e Encounter) rejected(rej EncounterRejection, err error) (encounterTransition, error) {
	if err != nil {
		return nil, err
	}
	return e.reject(rej), nil
}

// ApplyHealing restores Hit Points to a living combatant through HitPointStatus.Restore
// (SRD 5.2.1, "Healing", p. 17). A combatant that returns from 0 Hit Points, or a
// knocked-out one that regains Hit Points, loses Unconscious and is left Prone.
// Healing that restores nothing changes nothing.
//
// It returns the rejections of ApplyDamage. It returns an error matching
// srderr.ErrInvalidInput for an invalid encounter or id or a negative amount.
func (e Encounter) ApplyHealing(id CombatantID, amount int) (encounterTransition, error) {
	if amount < 0 {
		return nil, invalidInput("negative healing")
	}
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	tr, err := e.combatants[i].hp.Restore(id, amount, nil)
	if err != nil {
		return nil, err
	}
	return e.commit(i, id, tr, nil), nil
}

// PrecheckDeathSavingThrow returns the rejection ApplyDeathSavingThrow would give
// before any value is applied, and nil when a throw would be accepted. In order:
// EncounterAlreadyEndedRejection, UnknownCombatantRejection, CombatantIsDeadRejection,
// HitPointRuleRejection (NotDying), DeathSavingThrowAlreadyMadeRejection. It changes
// nothing and emits nothing. An invalid encounter or a zero id names no combatant and
// gives UnknownCombatantRejection.
func (e Encounter) PrecheckDeathSavingThrow(id CombatantID) EncounterRejection {
	i, rej, err := e.target(id)
	switch {
	case err != nil:
		return UnknownCombatantRejection{id}
	case rej != nil:
		return rej
	}
	return e.deathSavePrecheck(i, id)
}

// deathSavePrecheck is the command's own rejection for the combatant at i.
func (e Encounter) deathSavePrecheck(i int, id CombatantID) EncounterRejection {
	c := e.combatants[i]
	if r, no := c.hp.deathSaveRefusal(); no {
		return HitPointRuleRejection{id, r}
	}
	if c.dstMade {
		return DeathSavingThrowAlreadyMadeRejection{id}
	}
	return nil
}

// ApplyDeathSavingThrow applies a Death Saving Throw the host rolled: natural is the
// d20 face and total the figure with Exhaustion and Advantage applied (SRD 5.2.1,
// "Death Saving Throws", p. 17). It forwards both to HitPointStatus.ApplyDeathSave and
// sets the combatant's per-turn marker, which Start and AdvanceTurn clear when its turn
// begins; a natural 20 also removes Unconscious. The encounter does not check whose
// turn it is.
//
// It returns the rejections of PrecheckDeathSavingThrow. It returns an error matching
// srderr.ErrInvalidInput for an invalid encounter or id or a natural outside 1 to 20.
func (e Encounter) ApplyDeathSavingThrow(id CombatantID, natural, total int) (encounterTransition, error) {
	if natural < 1 || natural > 20 {
		return nil, invalidInput("death saving throw natural outside 1 to 20")
	}
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	if r := e.deathSavePrecheck(i, id); r != nil {
		return e.reject(r), nil
	}
	tr, err := e.combatants[i].hp.ApplyDeathSave(id, natural, total)
	if err != nil {
		return nil, err
	}
	return e.commit(i, id, tr, func(c *Combatant) { c.dstMade = true }), nil
}

// Stabilize makes a Dying combatant Stable through HitPointStatus.Stabilize (SRD
// 5.2.1, "Stabilizing a Character", p. 18); the success of the check is the host's
// input. It stays Unconscious. It returns the rejections of ApplyDamage and
// HitPointRuleRejection (NotDying), and an error as ApplyDamage does for an invalid
// encounter or id.
func (e Encounter) Stabilize(id CombatantID) (encounterTransition, error) {
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	tr, err := e.combatants[i].hp.Stabilize(id)
	if err != nil {
		return nil, err
	}
	return e.commit(i, id, tr, nil), nil
}

// RecoverStable gives a Stable combatant 1 Hit Point and makes it Alive through
// HitPointStatus.RecoverStable, once the host's delay is over, and removes Unconscious
// (SRD 5.2.1, "Stabilizing a Character", p. 18). It returns the rejections of
// ApplyDamage and HitPointRuleRejection (NotStable), and an error as ApplyDamage does
// for an invalid encounter or id.
func (e Encounter) RecoverStable(id CombatantID) (encounterTransition, error) {
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	tr, err := e.combatants[i].hp.RecoverStable(id)
	if err != nil {
		return nil, err
	}
	return e.commit(i, id, tr, nil), nil
}

// GainTemporaryHitPoints gives a living combatant Temporary Hit Points through
// HitPointStatus.GainTemporary (SRD 5.2.1, "Temporary Hit Points", p. 18); the choice
// decides between the held and the new amount. It returns the rejections of
// ApplyDamage and an error matching srderr.ErrInvalidInput for an invalid encounter or
// id, an amount below 1 or an invalid choice.
func (e Encounter) GainTemporaryHitPoints(id CombatantID, amount int, choice TemporaryHitPointsChoice) (encounterTransition, error) {
	if amount < 1 || !choice.Valid() {
		return nil, invalidInput("temporary hit points need an amount of 1 or more and a choice")
	}
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	tr, err := e.combatants[i].hp.GainTemporary(id, amount, choice)
	if err != nil {
		return nil, err
	}
	return e.commit(i, id, tr, nil), nil
}

// ReduceHitPointMaximum lowers a living combatant's Hit Point maximum through
// HitPointStatus.ReduceMaximum (SRD 5.2.1, "Hit Point Maximum of 0", p. 17); at 0 the
// combatant dies. It returns the rejections of ApplyDamage and an error matching
// srderr.ErrInvalidInput for an invalid encounter or id or a negative amount.
func (e Encounter) ReduceHitPointMaximum(id CombatantID, amount int) (encounterTransition, error) {
	if amount < 0 {
		return nil, invalidInput("negative maximum reduction")
	}
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	tr, err := e.combatants[i].hp.ReduceMaximum(id, amount)
	if err != nil {
		return nil, err
	}
	return e.commit(i, id, tr, nil), nil
}

// checkDamageInput returns an error matching srderr.ErrInvalidInput for inputs
// HitPointStatus.ApplyDamage would not take: no Damage, an invalid Damage, a nil
// adjustment, a delivery or a kind that is not valid. It judges them on a probe status,
// so it is the same check the hit point rules make, made before any rejection.
func checkDamageInput(id CombatantID, damages []Damage, adjustments []DamageAdjustment, delivery DamageDelivery, kind HitKind) error {
	probe := MustHitPointStatus(1, FallsUnconscious, NoAffinities())
	_, err := probe.ApplyDamage(id, damages, adjustments, delivery, kind, nil)
	return err
}
