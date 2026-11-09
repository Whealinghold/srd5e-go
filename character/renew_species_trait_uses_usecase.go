package character

import "context"

// RenewSpeciesTraitUsesCommand is the input of RenewSpeciesTraitUses.Run: the
// RestApplied event of this character. Construct it with a keyed literal. Run does
// not alter it.
type RenewSpeciesTraitUsesCommand struct {
	Character Character
	Traits    SpeciesTraits
	Rest      RestApplied
}

// RenewSpeciesTraitUsesResult is the closed set of outcomes of
// RenewSpeciesTraitUses.Run. Variants: RenewSpeciesTraitUsesSucceeded,
// RenewSpeciesTraitUsesRejected.
type RenewSpeciesTraitUsesResult interface{ isRenewSpeciesTraitUsesResult() }

// RenewSpeciesTraitUsesSucceeded is the outcome of a renewal, possibly of nothing.
// Only Run builds one.
type RenewSpeciesTraitUsesSucceeded struct{ change SpeciesTraitsChanged }

// Traits returns the new SpeciesTraits; the same traits when nothing was renewed.
func (r RenewSpeciesTraitUsesSucceeded) Traits() SpeciesTraits { return r.change.traits }

// Events returns a new slice of the events: one SpeciesUsesRenewed, or none when
// nothing was expended and no window opened.
func (r RenewSpeciesTraitUsesSucceeded) Events() []SpeciesTraitEvent { return r.change.Events() }

// Windows returns a new slice of the windows the rest opened, for the host to act on.
func (r RenewSpeciesTraitUsesSucceeded) Windows() []SpeciesWindow { return r.change.Windows() }

// RenewSpeciesTraitUsesRejected is the outcome of a renewal a rule refused. Construct
// it with a keyed literal.
type RenewSpeciesTraitUsesRejected struct{ Rejection SpeciesTraitRejection }

func (RenewSpeciesTraitUsesSucceeded) isRenewSpeciesTraitUsesResult() {}
func (RenewSpeciesTraitUsesRejected) isRenewSpeciesTraitUsesResult()  {}

// RenewSpeciesTraitUses is the use case that renews species trait uses after a rest.
// The host calls it with each RestApplied. It calls no port beyond telemetry. Create
// it with NewRenewSpeciesTraitUses; the zero value is not usable. It is safe for
// concurrent use only if its telemetry and clock are.
type RenewSpeciesTraitUses struct {
	sig operationSignals
	set bool
}

// NewRenewSpeciesTraitUses returns the use case. Options: WithTelemetry, WithClock;
// the others are ignored.
func NewRenewSpeciesTraitUses(opts ...Option) RenewSpeciesTraitUses {
	return RenewSpeciesTraitUses{sig: newOperationSignals(newOptions(opts)), set: true}
}

// Run applies cmd through SpeciesUses.Renew.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case or a zero Character, and ctx.Err() unwrapped, before any signal, when ctx is
// already cancelled. It never returns a srderr.PortError. An event of another character
// is a RenewSpeciesTraitUsesRejected result.
func (u RenewSpeciesTraitUses) Run(ctx context.Context, cmd RenewSpeciesTraitUsesCommand) (RenewSpeciesTraitUsesResult, error) {
	if !u.set || cmd.Character.IsZero() {
		return nil, invalidInput("renew species trait uses: zero use case or character")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRenewSpeciesTraitUses, ValueOperationRenewSpeciesTraitUses)
	defer run.end(ctx)
	tr, _ := SpeciesUses{}.Renew(cmd.Character, cmd.Traits, cmd.Rest) // the character is not zero: no error
	switch t := tr.(type) {
	case SpeciesTraitsChanged:
		run.finish(ctx, ValueResultChanged, "")
		return RenewSpeciesTraitUsesSucceeded{change: t}, nil
	default:
		rj := t.(SpeciesTraitsRejected).Rejection
		run.finish(ctx, ValueResultRejected, rj.String())
		return RenewSpeciesTraitUsesRejected{Rejection: rj}, nil
	}
}
