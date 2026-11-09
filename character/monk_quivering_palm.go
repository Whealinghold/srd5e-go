package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

const (
	quiveringPalmCost       = 4
	quiveringPalmDiceCount  = 10
	quiveringPalmDiceSides  = 12
	quiveringPalmDamageDice = "10d12"
)

// QuiveringPalmEnd says how the Monk ends the vibrations. The zero value is not an end.
type QuiveringPalmEnd int

// The ways to end the vibrations.
const (
	// QuiveringPalmEndHarmlessly: no action required, no save, no damage, no roll.
	QuiveringPalmEndHarmlessly QuiveringPalmEnd = iota + 1
	// QuiveringPalmEndWithEffect: by an action, or by forgoing an attack of the Attack action; the target saves.
	QuiveringPalmEndWithEffect
)

// QuiveringPalmStart is a hit the Monk would start the vibrations with. Construct it
// with a keyed literal.
type QuiveringPalmStart struct {
	// Target is the creature hit.
	Target QuiveringPalmTarget
	// HitWithUnarmedStrike reports that the hit was an Unarmed Strike, not another Monk weapon.
	HitWithUnarmedStrike bool
}

// QuiveringPalmStartResolution is the outcome of starting the vibrations. Only
// QuiveringPalm.Start builds one.
type QuiveringPalmStartResolution struct {
	vibrations QuiveringPalmVibrations
	state      MonkState
}

// Vibrations returns the vibrations: the target and the days equal to the Monk level.
func (r QuiveringPalmStartResolution) Vibrations() QuiveringPalmVibrations { return r.vibrations }

// State returns the new state: 4 Focus Points expended and the vibrations recorded.
func (r QuiveringPalmStartResolution) State() MonkState { return r.state }

// Event returns the QuiveringPalmStarted event for the character id.
func (r QuiveringPalmStartResolution) Event(id Identifier) MonkEvent {
	return QuiveringPalmStarted{id: id, target: r.vibrations.target, days: r.vibrations.days}
}

// QuiveringPalmEndResolution is the outcome of ending the vibrations. Only
// QuiveringPalm.Finish builds one.
type QuiveringPalmEndResolution struct {
	end    QuiveringPalmEnd
	dc     d20.DifficultyClass
	damage int
	state  MonkState
}

// End returns how the vibrations ended.
func (r QuiveringPalmEndResolution) End() QuiveringPalmEnd { return r.end }

// DifficultyClass returns the Constitution save DC used (the Monk's Focus save DC); 0 for a harmless end.
func (r QuiveringPalmEndResolution) DifficultyClass() d20.DifficultyClass { return r.dc }

// DamageApplied returns the Force damage: 10d12 on a failed save, half of it (rounded down) on a successful one; 0 for a harmless end.
func (r QuiveringPalmEndResolution) DamageApplied() int { return r.damage }

// DamageType returns Force.
func (QuiveringPalmEndResolution) DamageType() content.DamageType { return content.DamageForce }

// State returns the new state: no vibrations.
func (r QuiveringPalmEndResolution) State() MonkState { return r.state }

// Event returns the QuiveringPalmEnded event for the character id.
func (r QuiveringPalmEndResolution) Event(id Identifier) MonkEvent {
	return QuiveringPalmEnded{id: id, withEffect: r.end == QuiveringPalmEndWithEffect, damage: r.damage}
}

// QuiveringPalm decides and resolves the Warrior of the Open Hand's Quivering Palm
// (SRD 5.2.1, "Level 17: Quivering Palm", p. 52). It is stateless and pure and reads no
// clock; the zero value is ready to use. Safe for concurrent use.
type QuiveringPalm struct{}

// StartAvailability reports whether the Monk can start vibrations: MonkNotAMonk for zero
// stats, MonkLevelTooLow below level 17, MonkNotEligible for a subclass other than the
// Warrior of the Open Hand, a hit that is not an Unarmed Strike or no target, MonkAlreadyUsed
// while a creature is already under the effect, MonkNotEnoughFocusPoints with fewer than 4.
func (QuiveringPalm) StartAvailability(stats MonkLevelStats, state MonkState, subclass content.SubclassID, req QuiveringPalmStart) MonkAvailability {
	return monkAvailability(
		monkGate(stats, openHandPalmLevel),
		monkIf(subclass != OpenHandSubclassID || !req.HitWithUnarmedStrike || req.Target.IsZero(), MonkNotEligible),
		monkIf(state.vibrations.Active(), MonkAlreadyUsed),
		monkIf(state.FocusPoints(stats).Available() < quiveringPalmCost, MonkNotEnoughFocusPoints),
	)
}

// Start expends 4 Focus Points and records vibrations on the target lasting days equal
// to the Monk level. It returns an error matching srderr.ErrInvalidInput when
// StartAvailability says unavailable. It never panics.
func (q QuiveringPalm) Start(stats MonkLevelStats, state MonkState, subclass content.SubclassID, req QuiveringPalmStart) (QuiveringPalmStartResolution, error) {
	if _, ok := q.StartAvailability(stats, state, subclass, req).(MonkAvailable); !ok {
		return QuiveringPalmStartResolution{}, invalidInput("Quivering Palm started while unavailable")
	}
	v := QuiveringPalmVibrations{target: req.Target, days: stats.level.n} // the target is non-zero and the level at least 1
	state.focusExpended += quiveringPalmCost
	state.vibrations = v
	return QuiveringPalmStartResolution{vibrations: v, state: state}, nil
}

// EndAvailability reports whether the Monk can end the vibrations: MonkNotAMonk for zero
// stats; MonkNotEligible with no vibrations active, an unknown way to end, or an end with
// effect while the Monk and the target are not on the same plane of existence (samePlane).
// A harmless end needs no action and no plane.
func (QuiveringPalm) EndAvailability(stats MonkLevelStats, state MonkState, end QuiveringPalmEnd, samePlane bool) MonkAvailability {
	return monkAvailability(
		monkGate(stats, 1),
		monkIf(!state.vibrations.Active() || (end != QuiveringPalmEndHarmlessly && end != QuiveringPalmEndWithEffect) ||
			(end == QuiveringPalmEndWithEffect && !samePlane), MonkNotEligible),
	)
}

// DamageDice returns the dice of the damage when the vibrations end with effect: 10d12.
func (QuiveringPalm) DamageDice() dice.Expression {
	return dice.MustExpression(quiveringPalmDiceCount, dice.MustDie(quiveringPalmDiceSides), 0)
}

// Finish ends the vibrations. For an end with effect, outcome is the target's Constitution
// saving throw against the Monk's Focus save DC and rolledTotal the total of the 10d12;
// the damage is the total on a failure and half of it on a success. A harmless end takes
// rolledTotal 0 and ignores outcome.
//
// It returns an error matching srderr.ErrInvalidInput for a rolledTotal outside 10 to 120
// (or not 0 for a harmless end), an outcome other than d20.Success or d20.Failure on an
// end with effect, or when EndAvailability says unavailable. It never panics.
func (q QuiveringPalm) Finish(stats MonkLevelStats, state MonkState, abilities MonkAbilities, end QuiveringPalmEnd, samePlane bool, outcome d20.Outcome, rolledTotal int) (QuiveringPalmEndResolution, error) {
	if _, ok := q.EndAvailability(stats, state, end, samePlane).(MonkAvailable); !ok {
		return QuiveringPalmEndResolution{}, invalidInput("Quivering Palm ended while unavailable")
	}
	state.vibrations = QuiveringPalmVibrations{}
	if end == QuiveringPalmEndHarmlessly {
		if rolledTotal != 0 {
			return QuiveringPalmEndResolution{}, invalidInput("a harmless end of Quivering Palm rolls no dice")
		}
		return QuiveringPalmEndResolution{end: end, state: state}, nil
	}
	if (outcome != d20.Success && outcome != d20.Failure) || rolledTotal < quiveringPalmDiceCount || rolledTotal > quiveringPalmDiceCount*quiveringPalmDiceSides {
		return QuiveringPalmEndResolution{}, invalidInput("Quivering Palm outcome or " + quiveringPalmDamageDice + " total " + strconv.Itoa(rolledTotal) + " invalid")
	}
	damage := rolledTotal
	if outcome == d20.Success {
		damage /= 2
	}
	return QuiveringPalmEndResolution{end: end, dc: abilities.SaveDC(), damage: damage, state: state}, nil
}
