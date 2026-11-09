package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ResolveLastChargeCommand is the input of ResolveLastCharge.Do: the holder, the
// item copy, its charges (0 after the spend that emptied it) and its rule.
// Construct it with a keyed literal. Do does not alter it.
type ResolveLastChargeCommand struct {
	CharacterID Identifier
	Instance    MagicItemInstanceID
	Charges     ItemCharges
	Rule        LastChargeRule
}

// ResolveLastChargeRejection says why a resolution was refused. The zero value is
// not a reason.
type ResolveLastChargeRejection int

// ChargesRemain: the item still has charges, so its last one was not expended.
const ChargesRemain ResolveLastChargeRejection = 1

// String returns "charges_remain" or "unknown".
func (r ResolveLastChargeRejection) String() string {
	if r == ChargesRemain {
		return ValueRejectionChargesRemain
	}
	return "unknown"
}

// ResolveLastChargeResult is the closed set of outcomes of ResolveLastCharge.Do.
// Variants: ResolveLastChargeResolved, ResolveLastChargeRejected.
type ResolveLastChargeResult interface{ isResolveLastChargeResult() }

// ResolveLastChargeResolved: the new charges, what became of the item and the
// event. A destroyed or nonmagical item has lost its magic, which the host records.
type ResolveLastChargeResolved struct {
	charges     ItemCharges
	consequence LastChargeConsequence
	event       MagicItemLastChargeResolved
}

// Charges returns the new charges.
func (r ResolveLastChargeResolved) Charges() ItemCharges { return r.charges }

// Consequence returns what became of the item.
func (r ResolveLastChargeResolved) Consequence() LastChargeConsequence { return r.consequence }

// Event returns the MagicItemLastChargeResolved event.
func (r ResolveLastChargeResolved) Event() MagicItemLastChargeResolved { return r.event }

// ResolveLastChargeRejected: the item still has charges; nothing was rolled.
type ResolveLastChargeRejected struct{ reason ResolveLastChargeRejection }

// Reason returns ChargesRemain.
func (r ResolveLastChargeRejected) Reason() ResolveLastChargeRejection { return r.reason }

func (ResolveLastChargeResolved) isResolveLastChargeResult() {}
func (ResolveLastChargeRejected) isResolveLastChargeResult() {}

// ResolveLastCharge is the use case that resolves the rule of an item whose last
// charge was expended (SRD 5.2.1, "Wand of Magic Missiles", p. 251; "Staff of
// Power", p. 246). Create it with NewResolveLastCharge; the zero value is not
// usable. It is safe for concurrent use only if its roller, telemetry and clock are.
type ResolveLastCharge struct {
	roller Roller
	sig    operationSignals
}

// NewResolveLastCharge returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewResolveLastCharge(roller Roller, opts ...Option) (ResolveLastCharge, error) {
	if roller == nil {
		return ResolveLastCharge{}, invalidInput("resolve last charge without a dice roller")
	}
	return ResolveLastCharge{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Do resolves the rule. Charges above 0 are ResolveLastChargeRejected with no
// request. NoLastChargeEffect and LastChargeAutomatic make no request;
// LastChargeRolled makes one request of 1d20 and, on a 20 with a regain, one more
// for the regain dice, capped at the maximum. The dice are rolled even when the
// face changes nothing.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero holder, a blank instance, zero charges or a nil or
// foreign rule, and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled. It returns a *srderr.PortError (error type "dice_roller_error") with
// a nil result when the roller fails or answers another expression; the failure
// is recorded once and never logged.
func (u ResolveLastCharge) Do(ctx context.Context, cmd ResolveLastChargeCommand) (ResolveLastChargeResult, error) {
	if u.roller == nil || cmd.CharacterID.IsZero() || !cmd.Instance.Valid() || cmd.Charges.IsZero() || !knownLastChargeRule(cmd.Rule) {
		return nil, invalidInput("resolve last charge: zero use case, holder or charges, blank instance, or nil rule")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanResolveLastCharge, ValueOperationResolveLastCharge)
	defer run.end(ctx)
	if cmd.Charges.current > 0 {
		run.finish(ctx, ValueResultRejected, ValueRejectionChargesRemain)
		return ResolveLastChargeRejected{reason: ChargesRemain}, nil
	}
	charges, consequence := cmd.Charges, LastChargeNone
	switch r := cmd.Rule.(type) {
	case LastChargeAutomatic:
		consequence = r.consequence
	case LastChargeRolled:
		res, err := dice.Roll(u.roller, dice.MustExpression(1, dice.D20(), 0))
		if err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
		var regain LastChargeRegain
		consequence, regain = r.ForFace(res.Total())
		if expr, ok := regain.Dice(); ok {
			again, err := dice.Roll(u.roller, expr)
			if err != nil {
				run.portFailure(ctx, err, srderr.ErrorType(err))
				return nil, err
			}
			charges = charges.withRegained(again.Total())
		}
	}
	run.span.SetAttributes(telemetry.String(AttrLastChargeConsequence, consequence.String()))
	run.finish(ctx, ValueResultResolved, "")
	return ResolveLastChargeResolved{charges: charges, consequence: consequence,
		event: MagicItemLastChargeResolved{id: cmd.CharacterID, instance: cmd.Instance, consequence: consequence}}, nil
}

func knownLastChargeRule(r LastChargeRule) bool {
	switch r := r.(type) {
	case NoLastChargeEffect:
		return true
	case LastChargeAutomatic:
		return r.consequence.Valid()
	case LastChargeRolled:
		return r.onOne.Valid()
	}
	return false
}
