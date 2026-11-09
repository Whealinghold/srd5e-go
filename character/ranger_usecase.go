package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// rangerUse is what every Ranger use case shares: the progression and the feature signals.
type rangerUse struct {
	progression RangerProgression
	sig         featureSignals
	set         bool
}

func newRangerUse(what string, p RangerProgression, opts []Option) (rangerUse, error) {
	if p.IsZero() {
		return rangerUse{}, invalidInput(what + " without a Ranger progression")
	}
	return rangerUse{progression: p, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// precheck returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character, and
// ctx.Err() unwrapped for a cancelled context.
func (u rangerUse) precheck(ctx context.Context, c Character) error {
	if !u.set || c.IsZero() {
		return invalidInput("Ranger use case: zero use case or zero Character")
	}
	return ctx.Err()
}

// load reads the Ranger level and state of c. For a character with no Ranger level it records the not_a_ranger rejection and
// returns ok false. A state of another type under the Ranger key is recorded as a command error on the span.
func (u rangerUse) load(ctx context.Context, run featureRun, c Character) (RangerLevel, RangerState, bool, error) {
	level, ok := rangerLevelOf(c)
	if !ok {
		run.rejected(ctx, RangerNotARanger.String())
		return RangerLevel{}, RangerState{}, false, nil
	}
	state, err := readRangerState(c)
	if err != nil {
		run.commandError(err)
		return RangerLevel{}, RangerState{}, false, err
	}
	return level, state, true, nil
}

// facts returns the RangerFacts of level and state for the subclass the Ranger holds.
func (u rangerUse) facts(level RangerLevel, state RangerState, subclass content.SubclassID) RangerFacts {
	return RangerFacts{Progression: u.progression, Level: level, State: state, Subclass: subclass}
}
