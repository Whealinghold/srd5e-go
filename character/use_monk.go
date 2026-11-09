package character

import "context"

// monkUse is what every Monk use case holds: the progression, the dice port and the
// shared feature signals. It holds no rules logic.
type monkUse struct {
	progression MonkProgression
	roller      Roller
	sig         featureSignals
	set         bool
}

// newMonkUse returns the shared part of a Monk use case. It returns an error matching
// srderr.ErrInvalidInput for the zero progression or a nil roller.
func newMonkUse(progression MonkProgression, roller Roller, opts []Option) (monkUse, error) {
	if progression.IsZero() || roller == nil {
		return monkUse{}, invalidInput("Monk use case without a Monk progression or a dice roller")
	}
	return monkUse{progression: progression, roller: roller, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// check validates the command and the context before any signal: an error matching
// srderr.ErrInvalidInput for the zero use case or a zero Character, and ctx.Err()
// unwrapped for a cancelled context.
func (u monkUse) check(ctx context.Context, c Character) error {
	if err := monkCommandProblem(u.set, c); err != nil {
		return err
	}
	return ctx.Err()
}

// open starts the span and timer of the feature and derives the Monk context of c. The
// caller defers run.end. An error is recorded on the span as a command error.
func (u monkUse) open(ctx context.Context, c Character, span, feature string) (context.Context, featureRun, monkContext, error) {
	ctx, run := u.sig.start(ctx, span, ValueClassMonk, feature)
	mc, err := u.progression.contextOf(c)
	if err != nil {
		run.commandError(err)
	}
	return ctx, run, mc, err
}
