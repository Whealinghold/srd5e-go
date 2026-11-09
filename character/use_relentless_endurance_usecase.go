package character

import "context"

// UseRelentlessEnduranceCommand is the input of UseRelentlessEndurance.Run: what
// happened when the Orc was reduced to 0 Hit Points, as the host reads it from combat.
// Construct it with a keyed literal. Run does not alter it.
type UseRelentlessEnduranceCommand struct {
	Character Character
	Traits    SpeciesTraits
	Outcome   ZeroHitPointsOutcome
}

// UseRelentlessEnduranceResult is the closed set of outcomes of
// UseRelentlessEndurance.Run. Variants: UseRelentlessEnduranceUsed,
// UseRelentlessEnduranceNotTriggered, UseRelentlessEnduranceRejected.
type UseRelentlessEnduranceResult interface{ isUseRelentlessEnduranceResult() }

// UseRelentlessEnduranceUsed is the outcome of a drop to 1 Hit Point instead of 0,
// with the use spent. Only Run builds one.
type UseRelentlessEnduranceUsed struct {
	traits  SpeciesTraits
	instead int
	event   RelentlessEnduranceUsed
}

// Traits returns the new SpeciesTraits.
func (r UseRelentlessEnduranceUsed) Traits() SpeciesTraits { return r.traits }

// HitPointsInstead returns the Hit Points the Orc has instead of 0: 1.
func (r UseRelentlessEnduranceUsed) HitPointsInstead() int { return r.instead }

// Event returns the RelentlessEnduranceUsed event.
func (r UseRelentlessEnduranceUsed) Event() RelentlessEnduranceUsed { return r.event }

// UseRelentlessEnduranceNotTriggered is the outcome for an Orc killed outright: the
// trait does not apply and nothing is spent (a normal outcome, not a rejection).
type UseRelentlessEnduranceNotTriggered struct{}

// UseRelentlessEnduranceRejected is the outcome of a use a rule refused. Construct it
// with a keyed literal.
type UseRelentlessEnduranceRejected struct{ Rejection SpeciesTraitRejection }

func (UseRelentlessEnduranceUsed) isUseRelentlessEnduranceResult()         {}
func (UseRelentlessEnduranceNotTriggered) isUseRelentlessEnduranceResult() {}
func (UseRelentlessEnduranceRejected) isUseRelentlessEnduranceResult()     {}

// UseRelentlessEndurance is the use case of the Orc trait Relentless Endurance (SRD
// 5.2.1, p. 86). It calls no port beyond telemetry. Create it with
// NewUseRelentlessEndurance; the zero value is not usable. It is safe for concurrent
// use only if its telemetry and clock are.
type UseRelentlessEndurance struct {
	sig operationSignals
	set bool
}

// NewUseRelentlessEndurance returns the use case. Options: WithTelemetry, WithClock;
// the others are ignored.
func NewUseRelentlessEndurance(opts ...Option) UseRelentlessEndurance {
	return UseRelentlessEndurance{sig: newOperationSignals(newOptions(opts)), set: true}
}

// Run checks, in this order: the zero SpeciesTraits (SpeciesBehaviourUnknown), a
// species that is not an Orc (SpeciesUseNotOffered), an Orc killed outright
// (UseRelentlessEnduranceNotTriggered, nothing spent) and a spent use
// (SpeciesUseExhausted); otherwise it spends the use.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case, a zero Character or a nil outcome, and ctx.Err() unwrapped, before any
// signal, when ctx is already cancelled. It never returns a srderr.PortError.
func (u UseRelentlessEndurance) Run(ctx context.Context, cmd UseRelentlessEnduranceCommand) (UseRelentlessEnduranceResult, error) {
	if !u.set || cmd.Character.IsZero() || cmd.Outcome == nil {
		return nil, invalidInput("use relentless endurance: zero use case or character, or no outcome")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanUseRelentlessEndurance, ValueOperationUseRelentlessEndurance)
	defer run.end(ctx)
	instead, triggered := RelentlessEndurance{}.Resolve(cmd.Outcome)
	reject := func(r SpeciesTraitRejection) (UseRelentlessEnduranceResult, error) {
		run.finish(ctx, ValueResultRejected, r.String())
		return UseRelentlessEnduranceRejected{Rejection: r}, nil
	}
	switch {
	case cmd.Traits.IsZero():
		return reject(SpeciesBehaviourUnknown{})
	case !cmd.Traits.Offers(RelentlessEnduranceUse):
		return reject(SpeciesUseNotOffered{})
	case !triggered:
		run.finish(ctx, ValueResultNotApplicable, "")
		return UseRelentlessEnduranceNotTriggered{}, nil
	case cmd.Traits.ExpendedOf(RelentlessEnduranceUse) >= RelentlessEnduranceUse.Limit(cmd.Character.TotalLevel()):
		return reject(SpeciesUseExhausted{})
	}
	next := cmd.Traits
	next.expended[RelentlessEnduranceUse-1]++
	run.finish(ctx, ValueResultChanged, "")
	return UseRelentlessEnduranceUsed{traits: next, instead: instead, event: RelentlessEnduranceUsed{id: cmd.Character.ID()}}, nil
}
