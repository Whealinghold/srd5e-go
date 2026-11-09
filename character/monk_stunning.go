package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// StunningStrikeEffect is what a Stunning Strike does to the target, a closed set.
// Variants: StunApplied, StunResisted.
type StunningStrikeEffect interface{ isStunningStrikeEffect() }

// StunApplied is the effect of a failed save: the target has the Stunned condition
// until the start of the Monk's next turn. Construct it with a keyed literal.
type StunApplied struct {
	Condition content.ConditionKind
	Until     MonkEffectEnd
}

// StunResisted is the effect of a successful save: the target's Speed is halved and
// the next attack roll made against it has Advantage, until the start of the Monk's
// next turn. Construct it with a keyed literal.
type StunResisted struct {
	SpeedHalved, AdvantageOnNextAttack bool
	Until                              MonkEffectEnd
}

func (StunApplied) isStunningStrikeEffect()  {}
func (StunResisted) isStunningStrikeEffect() {}

// MonkStunningHit is the host's report of the hit a Stunning Strike would ride on.
// Construct it with a keyed literal.
type MonkStunningHit struct {
	// Hit reports that the attack roll hit a creature.
	Hit bool
	// MonkWeaponOrUnarmed reports that the attack used a Monk weapon or an Unarmed Strike.
	MonkWeaponOrUnarmed bool
	// UsedThisTurn reports that a Stunning Strike was already made this turn.
	UsedThisTurn bool
}

// StunningStrikeResolution is the outcome of a Stunning Strike. Only
// StunningStrike.Resolve builds one.
type StunningStrikeResolution struct {
	dc     d20.DifficultyClass
	effect StunningStrikeEffect
	state  MonkState
}

// DifficultyClass returns the save DC used: the Monk's Focus save DC.
func (r StunningStrikeResolution) DifficultyClass() d20.DifficultyClass { return r.dc }

// Effect returns StunApplied or StunResisted.
func (r StunningStrikeResolution) Effect() StunningStrikeEffect { return r.effect }

// State returns the new state: 1 Focus Point expended.
func (r StunningStrikeResolution) State() MonkState { return r.state }

// Event returns the StunningStrikeAttempted event for the character id.
func (r StunningStrikeResolution) Event(id Identifier) MonkEvent {
	_, stunned := r.effect.(StunApplied)
	return StunningStrikeAttempted{id: id, dc: r.dc, stunned: stunned}
}

// StunningStrike decides and resolves Stunning Strike (SRD 5.2.1, "Level 5: Stunning
// Strike", p. 51). It is stateless and pure; the zero value is ready to use. Safe for
// concurrent use.
type StunningStrike struct{}

// Availability reports whether the Monk can attempt a stunning strike on hit:
// MonkNotAMonk for zero stats, MonkLevelTooLow below level 5, MonkNotEligible without
// a hit made with a Monk weapon or an Unarmed Strike, MonkAlreadyUsed when one was made
// this turn, MonkNotEnoughFocusPoints with no Focus Point.
func (StunningStrike) Availability(stats MonkLevelStats, state MonkState, hit MonkStunningHit) MonkAvailability {
	return monkAvailability(
		monkGate(stats, monkExtraAttackLevel),
		monkIf(!hit.Hit || !hit.MonkWeaponOrUnarmed, MonkNotEligible),
		monkIf(hit.UsedThisTurn, MonkAlreadyUsed),
		monkIf(state.FocusPoints(stats).Available() < 1, MonkNotEnoughFocusPoints),
	)
}

// Resolve attempts the strike: 1 Focus Point is expended and the target's
// Constitution saving throw (outcome) against the Monk's Focus save DC decides the effect.
//
// It returns an error matching srderr.ErrInvalidInput for an outcome other than
// d20.Success or d20.Failure, or when Availability says unavailable. It never panics.
func (s StunningStrike) Resolve(stats MonkLevelStats, state MonkState, abilities MonkAbilities, hit MonkStunningHit, outcome d20.Outcome) (StunningStrikeResolution, error) {
	if outcome != d20.Success && outcome != d20.Failure {
		return StunningStrikeResolution{}, invalidInput("Stunning Strike saving throw outcome is neither success nor failure")
	}
	if _, ok := s.Availability(stats, state, hit).(MonkAvailable); !ok {
		return StunningStrikeResolution{}, invalidInput("Stunning Strike resolved while unavailable")
	}
	state.focusExpended++
	effect := StunningStrikeEffect(StunResisted{SpeedHalved: true, AdvantageOnNextAttack: true, Until: MonkEffectEndsAtStartOfMonkNextTurn})
	if outcome == d20.Failure {
		effect = StunApplied{Condition: content.ConditionStunned, Until: MonkEffectEndsAtStartOfMonkNextTurn}
	}
	return StunningStrikeResolution{dc: abilities.SaveDC(), effect: effect, state: state}, nil
}
