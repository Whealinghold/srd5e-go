package character

import (
	"context"
	"slices"
)

// GainHeroicInspirationCommand is the input of GainHeroicInspiration.Gain: the
// holder gaining it, and what to do with the excess instance if the holder
// already has one. There is no parameter for what granted it. Construct it with
// a keyed literal. Gain does not alter it.
type GainHeroicInspirationCommand struct {
	Holder Character
	Excess ExcessInspirationChoice
}

// GainHeroicInspirationResult is the closed set of outcomes of
// GainHeroicInspiration.Gain. Variants: GainHeroicInspirationReceived,
// GainHeroicInspirationExcessLost, GainHeroicInspirationExcessGiven,
// GainHeroicInspirationRejected.
type GainHeroicInspirationResult interface{ isGainHeroicInspirationResult() }

// GainHeroicInspirationReceived: the holder was not Inspired and now is.
type GainHeroicInspirationReceived struct {
	holder Character
	events []Event
}

// Holder returns the holder after the gain.
func (r GainHeroicInspirationReceived) Holder() Character { return r.holder }

// Events returns a new slice holding the one HeroicInspirationGained event.
func (r GainHeroicInspirationReceived) Events() []Event { return slices.Clone(r.events) }

// GainHeroicInspirationExcessLost: the holder was Inspired and the excess was lost.
type GainHeroicInspirationExcessLost struct {
	holder Character
	events []Event
}

// Holder returns the holder, unchanged.
func (r GainHeroicInspirationExcessLost) Holder() Character { return r.holder }

// Events returns a new, empty slice: nothing happened.
func (r GainHeroicInspirationExcessLost) Events() []Event { return slices.Clone(r.events) }

// GainHeroicInspirationExcessGiven: the excess went to a recipient who lacked it.
type GainHeroicInspirationExcessGiven struct {
	holder, recipient Character
	events            []Event
}

// Holder returns the holder, unchanged.
func (r GainHeroicInspirationExcessGiven) Holder() Character { return r.holder }

// Recipient returns the recipient after the gain.
func (r GainHeroicInspirationExcessGiven) Recipient() Character { return r.recipient }

// Events returns a new slice holding the recipient's HeroicInspirationGained event.
func (r GainHeroicInspirationExcessGiven) Events() []Event { return slices.Clone(r.events) }

// GainHeroicInspirationRejected: the excess could not be given; nothing changed
// and no event was produced. Resubmit with LoseExcessInspiration if that is wanted.
type GainHeroicInspirationRejected struct {
	reason GainHeroicInspirationRejection
}

// Reason returns why the give was refused.
func (r GainHeroicInspirationRejected) Reason() GainHeroicInspirationRejection { return r.reason }

func (GainHeroicInspirationReceived) isGainHeroicInspirationResult()    {}
func (GainHeroicInspirationExcessLost) isGainHeroicInspirationResult()  {}
func (GainHeroicInspirationExcessGiven) isGainHeroicInspirationResult() {}
func (GainHeroicInspirationRejected) isGainHeroicInspirationResult()    {}

// GainHeroicInspiration is the use case that gains Heroic Inspiration at the
// host boundary, whoever grants it (SRD 5.2.1, "Heroic Inspiration", p. 8). It
// maps ResolveExcessInspiration and emits the
// srd5e.character.gain_heroic_inspiration span and the shared character
// signals. It holds no rules logic and needs no port beyond telemetry. Create
// it with NewGainHeroicInspiration; the zero value is not usable. It is safe
// for concurrent use only if its telemetry and clock are.
type GainHeroicInspiration struct {
	sig operationSignals
	set bool
}

// NewGainHeroicInspiration returns the use case. Options: WithTelemetry,
// WithClock; the others are ignored.
func NewGainHeroicInspiration(opts ...Option) GainHeroicInspiration {
	return GainHeroicInspiration{sig: newOperationSignals(newOptions(opts)), set: true}
}

// Gain gains Heroic Inspiration for cmd.Holder, resolving an excess instance by
// cmd.Excess. A refused give is a GainHeroicInspirationRejected result.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero use case, a zero Holder or a give to a zero Recipient, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled. It calls no port
// and never returns a srderr.PortError.
func (u GainHeroicInspiration) Gain(ctx context.Context, cmd GainHeroicInspirationCommand) (GainHeroicInspirationResult, error) {
	if give, ok := cmd.Excess.(GiveExcessInspirationTo); !u.set || cmd.Holder.IsZero() || (ok && give.Recipient.IsZero()) {
		return nil, invalidInput("gain Heroic Inspiration: zero use case, holder or recipient")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanGainHeroicInspiration, ValueOperationGain)
	defer run.end(ctx)
	switch r := ResolveExcessInspiration(cmd.Holder, cmd.Excess).(type) {
	case ExcessResolutionReceived:
		run.finish(ctx, ValueResultGained, "")
		return GainHeroicInspirationReceived(r), nil
	case ExcessResolutionExcessLost:
		run.finish(ctx, ValueResultExcessLost, "")
		return GainHeroicInspirationExcessLost(r), nil
	case ExcessResolutionExcessGiven:
		run.finish(ctx, ValueResultExcessGiven, "")
		return GainHeroicInspirationExcessGiven(r), nil
	default:
		rj := r.(ExcessResolutionRejected)
		run.finish(ctx, ValueResultRejected, rj.reason.String())
		return GainHeroicInspirationRejected(rj), nil
	}
}
