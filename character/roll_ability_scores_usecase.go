package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RollAbilityScoresCommand is the input of RollAbilityScores.Run. It has no fields;
// it keeps the call shape of the other use cases.
type RollAbilityScoresCommand struct{}

// RollAbilityScoresResult is the closed set of outcomes of RollAbilityScores.Run.
// Variants: RollAbilityScoresSucceeded.
type RollAbilityScoresResult interface{ isRollAbilityScoresResult() }

// RollAbilityScoresSucceeded is the six rolls of Random Generation, in roll order.
// The use case does not assign them. Only Run builds one.
type RollAbilityScoresSucceeded struct{ scores RolledAbilityScores }

// Scores returns the six rolls.
func (r RollAbilityScoresSucceeded) Scores() RolledAbilityScores { return r.scores }

func (RollAbilityScoresSucceeded) isRollAbilityScoresResult() {}

// RollAbilityScores is the use case that rolls the six scores of Random Generation:
// 4d6 six times, each total the highest three dice (SRD 5.2.1, "Random Generation",
// p. 21). It uses the dice port and telemetry only and is stateless. Create it with
// NewRollAbilityScores; the zero value is not usable. It is safe for concurrent use
// only if its roller, telemetry and clock are.
type RollAbilityScores struct {
	roller Roller
	sig    rollSignals
}

// NewRollAbilityScores returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewRollAbilityScores(roller Roller, opts ...Option) (RollAbilityScores, error) {
	if roller == nil {
		return RollAbilityScores{}, invalidInput("roll ability scores without a dice roller")
	}
	return RollAbilityScores{roller: roller, sig: newRollSignals(newOptions(opts))}, nil
}

// Run rolls exactly six 4d6 requests in order and records each total in the
// ability_score_roll histogram.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled.
// It returns a *srderr.PortError (error type "dice_roller_error") with a nil result
// when the roller fails, panics or answers another expression; the failure is recorded
// once and never logged, and no total is recorded.
func (u RollAbilityScores) Run(ctx context.Context, _ RollAbilityScoresCommand) (RollAbilityScoresResult, error) {
	if u.roller == nil {
		return nil, invalidInput("zero RollAbilityScores; use NewRollAbilityScores")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRollAbilityScores, ValueOperationRollAbilityScores)
	defer run.end(ctx)
	var out RolledAbilityScores
	for i := range out.scores {
		res, err := dice.Roll(u.roller, dice.MustExpression(4, dice.D6(), 0))
		if err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
		total, err := res.SumOfHighest(3)
		if err != nil { // unreachable: four dice were rolled
			run.commandError(err)
			return nil, err
		}
		for j, d := range res.Rolls() {
			out.scores[i].dice[j] = d.Value()
		}
		out.scores[i].total = total
	}
	for _, s := range out.scores {
		u.sig.totals.Record(ctx, float64(s.total))
	}
	run.finish(ctx, ValueResultRolled, "")
	return RollAbilityScoresSucceeded{scores: out}, nil
}
