package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// bardUse is what every Bard use case holds: its dice roller and the shared class feature signals.
type bardUse struct {
	roller Roller
	sig    featureSignals
	set    bool
}

// newBardUse returns the shared part of a Bard use case. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func newBardUse(what string, roller Roller, opts []Option) (bardUse, error) {
	if roller == nil {
		return bardUse{}, invalidInput(what + " without a dice roller")
	}
	return bardUse{roller: roller, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// start checks the command preconditions shared by the use cases and opens the span. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero use case or a command the caller found unusable, and
// ctx.Err() unwrapped, before any signal, for a cancelled context.
func (u bardUse) start(ctx context.Context, unusable bool, span, feature string) (context.Context, featureRun, error) {
	if !u.set || unusable {
		return ctx, featureRun{}, invalidInput("Bard use case: zero use case or an unusable command")
	}
	if err := ctx.Err(); err != nil {
		return ctx, featureRun{}, err
	}
	ctx, run := u.sig.start(ctx, span, ValueClassBard, feature)
	return ctx, run, nil
}

// fail records err: a returned *srderr.PortError once on the shared signals, any other error on the span only.
func fail(ctx context.Context, run featureRun, err error) {
	if et := srderr.ErrorType(err); et != "" {
		run.portFailure(ctx, err, et)
		return
	}
	run.commandError(err)
}

// bardFacts are what the Bard use cases read from a Character: the Bard level, the Charisma modifier of its ability
// scores and its Bardic Inspiration pool, with the maximum already recomputed.
type bardFacts struct {
	level    BardLevel
	charisma ability.Modifier
	pool     BardicInspirationPool
}

// readBardFacts returns the facts of c and true, false for a character with no Bard level, and an error matching
// srderr.ErrInvalidInput when the Bard holds no pool or another type of state sits under the key.
func readBardFacts(c Character) (bardFacts, bool, error) {
	level, ok := bardLevelOf(c)
	if !ok {
		return bardFacts{}, false, nil
	}
	charisma := c.AbilityScores().Modifier(ability.Charisma)
	pool, err := readBardicInspirationPool(c, charisma)
	return bardFacts{level: level, charisma: charisma, pool: pool}, err == nil, err
}
