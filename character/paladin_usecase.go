package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// paladinUse is what every Paladin use case shares: the progression and the feature signals.
type paladinUse struct {
	progression PaladinProgression
	sig         featureSignals
	set         bool
}

func newPaladinUse(what string, p PaladinProgression, opts []Option) (paladinUse, error) {
	if p.IsZero() {
		return paladinUse{}, invalidInput(what + " without a Paladin progression")
	}
	return paladinUse{progression: p, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// precheck returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character, and
// ctx.Err() unwrapped for a cancelled context.
func (u paladinUse) precheck(ctx context.Context, c Character) error {
	if !u.set || c.IsZero() {
		return invalidInput("Paladin use case: zero use case or zero Character")
	}
	return ctx.Err()
}

// load reads the Paladin level and state of c. For a character with no Paladin level it records the not_a_paladin rejection and
// returns ok false. A state of another type under the Paladin key is recorded as a command error on the span.
func (u paladinUse) load(ctx context.Context, run featureRun, c Character) (PaladinLevel, PaladinState, bool, error) {
	level, ok := paladinLevelOf(c)
	if !ok {
		run.rejected(ctx, PaladinNotAPaladin.String())
		return PaladinLevel{}, PaladinState{}, false, nil
	}
	state, err := readPaladinState(c, level, u.progression.ChannelDivinityMaximum(level))
	if err != nil {
		run.commandError(err)
		return PaladinLevel{}, PaladinState{}, false, err
	}
	return level, state, true, nil
}

// oathChosen reports whether the subclass is the Oath of Devotion.
func oathChosen(subclass content.SubclassID) bool { return subclass == OathOfDevotionSubclassID }

// paladinReject records a rejection with the first reason and copies the reasons for the result.
func paladinReject(ctx context.Context, run featureRun, reasons []PaladinReason) []PaladinReason {
	run.rejected(ctx, reasons[0].String())
	return append([]PaladinReason{}, reasons...)
}

// paladinStore writes state back into c. The error is unreachable (the key is valid and a held state of another type was
// reported by load) and is recorded on the span.
func paladinStore(run featureRun, c Character, state PaladinState) (Character, error) {
	next, err := WithFeatureState(c, PaladinStateKey(), state)
	if err != nil {
		run.commandError(err)
		return Character{}, err
	}
	return next, nil
}
