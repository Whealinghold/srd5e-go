package character

import "context"

// RenewFeatUsesCommand is the input of RenewFeatUses.Run. Construct it with a
// keyed literal. Run does not alter it.
type RenewFeatUsesCommand struct {
	Character Character
	Feats     TakenFeats
	Renewal   FeatRenewal
}

// RenewFeatUsesResult is the closed set of outcomes of RenewFeatUses.Run.
// Variants: RenewFeatUsesSucceeded.
type RenewFeatUsesResult interface{ isRenewFeatUsesResult() }

// RenewFeatUsesSucceeded is the outcome of a renewal, possibly of nothing. Only
// Run builds one.
type RenewFeatUsesSucceeded struct{ change FeatsChanged }

// Feats returns the new TakenFeats; the same feats when nothing was renewed.
func (r RenewFeatUsesSucceeded) Feats() TakenFeats { return r.change.feats }

// Events returns a new slice of the events: one FeatUsesRenewed, or none when
// nothing was expended or listed the moment.
func (r RenewFeatUsesSucceeded) Events() []FeatEvent { return r.change.Events() }

func (RenewFeatUsesSucceeded) isRenewFeatUsesResult() {}

// RenewFeatUses is the use case that renews expended feat uses at a rest or at
// Initiative. The host calls it after each rest. It calls no port beyond
// telemetry. Create it with NewRenewFeatUses; the zero value is not usable. It is
// safe for concurrent use only if its telemetry and clock are.
type RenewFeatUses struct {
	sig operationSignals
	set bool
}

// NewRenewFeatUses returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored.
func NewRenewFeatUses(opts ...Option) RenewFeatUses {
	return RenewFeatUses{sig: newOperationSignals(newOptions(opts)), set: true}
}

// Run applies cmd through FeatUses.Renew.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero Character or an invalid renewal, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled. It never returns a
// srderr.PortError.
func (u RenewFeatUses) Run(ctx context.Context, cmd RenewFeatUsesCommand) (RenewFeatUsesResult, error) {
	if !u.set || cmd.Character.IsZero() || !cmd.Renewal.Valid() {
		return nil, invalidInput("renew feat uses: zero use case or character, or an invalid renewal")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRenewFeatUses, ValueOperationRenewFeatUses)
	defer run.end(ctx)
	run.finish(ctx, ValueResultChanged, "")
	return RenewFeatUsesSucceeded{change: renewFeatUses(cmd.Character, cmd.Feats, cmd.Renewal).(FeatsChanged)}, nil
}
