package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// ExpendFeatUseCommand is the input of ExpendFeatUse.Run. Construct it with a
// keyed literal. Run does not alter it.
type ExpendFeatUseCommand struct {
	Character Character
	Feats     TakenFeats
	Feat      content.FeatID
}

// ExpendFeatUseResult is the closed set of outcomes of ExpendFeatUse.Run.
// Variants: ExpendFeatUseSucceeded, ExpendFeatUseRejected.
type ExpendFeatUseResult interface{ isExpendFeatUseResult() }

// ExpendFeatUseSucceeded is the outcome of a use expended. Only Run builds one.
type ExpendFeatUseSucceeded struct{ change FeatsChanged }

// Feats returns the new TakenFeats.
func (r ExpendFeatUseSucceeded) Feats() TakenFeats { return r.change.feats }

// Event returns the FeatUseExpended event.
func (r ExpendFeatUseSucceeded) Event() FeatUseExpended { return r.change.events[0].(FeatUseExpended) }

// ExpendFeatUseRejected is the outcome of an expenditure a rule refused. Construct
// it with a keyed literal.
type ExpendFeatUseRejected struct{ Rejection FeatRejection }

func (ExpendFeatUseSucceeded) isExpendFeatUseResult() {}
func (ExpendFeatUseRejected) isExpendFeatUseResult()  {}

// ExpendFeatUse is the use case that expends the limited use of a held feat (Magic
// Initiate's free cast, Improve Fate). It calls no port beyond telemetry. Create it
// with NewExpendFeatUse; the zero value is not usable. It is safe for concurrent
// use only if its telemetry and clock are.
type ExpendFeatUse struct {
	sig operationSignals
	set bool
}

// NewExpendFeatUse returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored.
func NewExpendFeatUse(opts ...Option) ExpendFeatUse {
	return ExpendFeatUse{sig: newOperationSignals(newOptions(opts)), set: true}
}

// Run applies cmd through FeatUses.Expend.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero Character or an invalid feat id, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled. It never returns a
// srderr.PortError. A refused expenditure is an ExpendFeatUseRejected result.
func (u ExpendFeatUse) Run(ctx context.Context, cmd ExpendFeatUseCommand) (ExpendFeatUseResult, error) {
	if !u.set || cmd.Character.IsZero() || !cmd.Feat.Valid() {
		return nil, invalidInput("expend a feat use: zero use case or character, or an invalid feat id")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanExpendFeatUse, ValueOperationExpendFeatUse)
	defer run.end(ctx)
	switch t := expendFeatUse(cmd.Character, cmd.Feats, cmd.Feat).(type) {
	case FeatsChanged:
		run.finish(ctx, ValueResultChanged, "")
		return ExpendFeatUseSucceeded{change: t}, nil
	default:
		rj := t.(FeatsRejected).Rejection
		run.finish(ctx, ValueResultRejected, rj.String())
		return ExpendFeatUseRejected{Rejection: rj}, nil
	}
}
