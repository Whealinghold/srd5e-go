package character

import "context"

// ExpendSpeciesTraitUseCommand is the input of ExpendSpeciesTraitUse.Run. Construct
// it with a keyed literal. Run does not alter it.
type ExpendSpeciesTraitUseCommand struct {
	Character Character
	Traits    SpeciesTraits
	Request   SpeciesUseRequest
}

// ExpendSpeciesTraitUseResult is the closed set of outcomes of
// ExpendSpeciesTraitUse.Run. Variants: ExpendSpeciesTraitUseSucceeded,
// ExpendSpeciesTraitUseRejected.
type ExpendSpeciesTraitUseResult interface{ isExpendSpeciesTraitUseResult() }

// ExpendSpeciesTraitUseSucceeded is the outcome of a use expended. Only Run builds one.
type ExpendSpeciesTraitUseSucceeded struct{ change SpeciesTraitsChanged }

// Traits returns the new SpeciesTraits.
func (r ExpendSpeciesTraitUseSucceeded) Traits() SpeciesTraits { return r.change.traits }

// Effect returns the effect the trait declares.
func (r ExpendSpeciesTraitUseSucceeded) Effect() SpeciesTraitEffect { return r.change.effect }

// Event returns the SpeciesUseExpended event.
func (r ExpendSpeciesTraitUseSucceeded) Event() SpeciesUseExpended {
	return r.change.events[0].(SpeciesUseExpended)
}

// ExpendSpeciesTraitUseRejected is the outcome of an expenditure a rule refused.
// Construct it with a keyed literal.
type ExpendSpeciesTraitUseRejected struct{ Rejection SpeciesTraitRejection }

func (ExpendSpeciesTraitUseSucceeded) isExpendSpeciesTraitUseResult() {}
func (ExpendSpeciesTraitUseRejected) isExpendSpeciesTraitUseResult()  {}

// ExpendSpeciesTraitUse is the use case that expends a limited use of a species trait
// and returns the effect it declares. It calls no port beyond telemetry. Create it
// with NewExpendSpeciesTraitUse; the zero value is not usable. It is safe for
// concurrent use only if its telemetry and clock are.
type ExpendSpeciesTraitUse struct {
	sig operationSignals
	set bool
}

// NewExpendSpeciesTraitUse returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored.
func NewExpendSpeciesTraitUse(opts ...Option) ExpendSpeciesTraitUse {
	return ExpendSpeciesTraitUse{sig: newOperationSignals(newOptions(opts)), set: true}
}

// Run applies cmd through SpeciesUses.Expend.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case, a zero Character or a nil or invalid request, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled. It never returns a
// srderr.PortError. A refused expenditure is an ExpendSpeciesTraitUseRejected result.
func (u ExpendSpeciesTraitUse) Run(ctx context.Context, cmd ExpendSpeciesTraitUseCommand) (ExpendSpeciesTraitUseResult, error) {
	use, ok := requestedUse(cmd.Request)
	if !u.set || cmd.Character.IsZero() || !ok {
		return nil, invalidInput("expend a species trait use: zero use case or character, or an invalid request")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanExpendSpeciesTraitUse, ValueOperationExpendSpeciesTraitUse)
	defer run.end(ctx)
	switch t := expendSpeciesUse(cmd.Character, cmd.Traits, cmd.Request, use).(type) {
	case SpeciesTraitsChanged:
		run.finish(ctx, ValueResultChanged, "")
		return ExpendSpeciesTraitUseSucceeded{change: t}, nil
	default:
		rj := t.(SpeciesTraitsRejected).Rejection
		run.finish(ctx, ValueResultRejected, rj.String())
		return ExpendSpeciesTraitUseRejected{Rejection: rj}, nil
	}
}
