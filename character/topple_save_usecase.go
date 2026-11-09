package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ResolveToppleSaveCommand is the input of ToppleSaveResolver.ResolveToppleSave:
// the Topple offer and how the target attempts the saving throw (a
// d20.RollAttempt with the host's modifiers and circumstances, d20.ChooseToFail
// or d20.AutomaticFailure). Construct it with a keyed literal.
type ResolveToppleSaveCommand struct {
	Offer   ToppleOffer
	Attempt d20.SaveAttempt
}

// ToppleSaveBasis is how the save was decided: the closed set ToppleSaveRolled
// and ToppleSaveForced.
//
//sumtype:decl
type ToppleSaveBasis interface{ isToppleSaveBasis() }

// ToppleSaveRolled: the save was rolled.
type ToppleSaveRolled struct{ Check d20.CheckResult }

// ToppleSaveForced: the save failed without a roll.
type ToppleSaveForced struct{ Failure d20.ForcedFailureResult }

func (ToppleSaveRolled) isToppleSaveBasis() {}
func (ToppleSaveForced) isToppleSaveBasis() {}

// ToppleSaveDetail is the DC and how the save was decided, for diagnosis.
type ToppleSaveDetail struct {
	DC    d20.DifficultyClass
	Basis ToppleSaveBasis
}

// ResolveToppleSaveResult is the closed outcome of ResolveToppleSave:
// ToppleSaveTargetProne or ToppleSaveTargetResisted.
//
//sumtype:decl
type ResolveToppleSaveResult interface{ isResolveToppleSaveResult() }

// ToppleSaveTargetProne: the save failed. Condition describes the Prone
// condition as data; the engine does not apply it.
type ToppleSaveTargetProne struct{ Detail ToppleSaveDetail }

// Condition returns content.ConditionProne.
func (ToppleSaveTargetProne) Condition() content.ConditionKind { return content.ConditionProne }

// ToppleSaveTargetResisted: the save succeeded; there is no effect.
type ToppleSaveTargetResisted struct{ Detail ToppleSaveDetail }

func (ToppleSaveTargetProne) isResolveToppleSaveResult()    {}
func (ToppleSaveTargetResisted) isResolveToppleSaveResult() {}

// ToppleSaveResolver is the use case that makes the target's Constitution saving
// throw for a Topple offer (SRD 5.2.1, "Topple", p. 90). It composes the pure
// functions of package d20, calls no other use case (not d20.Resolver) and holds
// no rules logic. Create it with NewToppleSaveResolver; the zero value is not
// usable. It is safe for concurrent use only if its roller, telemetry and clock
// are.
type ToppleSaveResolver struct {
	roller Roller
	sig    operationSignals
}

// NewToppleSaveResolver returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewToppleSaveResolver(roller Roller, opts ...Option) (ToppleSaveResolver, error) {
	if roller == nil {
		return ToppleSaveResolver{}, invalidInput("resolve a Topple save without a dice roller")
	}
	return ToppleSaveResolver{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// ResolveToppleSave makes the save. A rolling attempt makes one request for 1d20
// (2d20 under Advantage or Disadvantage) and succeeds when the total reaches the
// DC; ChooseToFail and AutomaticFailure roll nothing and fail. A failed save is
// ToppleSaveTargetProne, a passed one ToppleSaveTargetResisted; both are normal
// outcomes.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero use case, a zero offer, or a nil or foreign attempt, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the
// roller fails, panics or breaks its contract; the failure is recorded once and
// never logged.
func (u ToppleSaveResolver) ResolveToppleSave(ctx context.Context, cmd ResolveToppleSaveCommand) (ResolveToppleSaveResult, error) {
	if u.roller == nil || cmd.Offer.ability == 0 {
		return nil, invalidInput("resolve a Topple save: zero use case or offer")
	}
	switch cmd.Attempt.(type) {
	case d20.RollAttempt, d20.ChooseToFail, d20.AutomaticFailure:
	default:
		return nil, invalidInput("resolve a Topple save: nil or foreign attempt")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanResolveToppleSave, ValueOperationResolveToppleSave)
	defer run.end(ctx)
	var (
		basis ToppleSaveBasis
		err   error
	)
	failed := true
	switch a := cmd.Attempt.(type) {
	case d20.RollAttempt:
		var roll d20.Roll
		roll, err = u.roll(a)
		if err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
		var check d20.CheckResult
		check, err = d20.ResolveCheck(d20.SavingThrow, roll, a.Modifiers, cmd.Offer.dc)
		if err != nil { // unreachable: the roll is built and the kind is a saving throw
			return nil, err
		}
		basis, failed = ToppleSaveRolled{Check: check}, check.Outcome() == d20.Failure
	case d20.ChooseToFail:
		basis, err = forcedFailure(d20.ChosenToFail)
	case d20.AutomaticFailure:
		basis, err = forcedFailure(d20.Automatic)
	}
	if err != nil { // unreachable: a saving throw may fail for either reason
		return nil, err
	}
	detail := ToppleSaveDetail{DC: cmd.Offer.dc, Basis: basis}
	if failed {
		run.finish(ctx, ValueResultTargetProne, "")
		return ToppleSaveTargetProne{Detail: detail}, nil
	}
	run.finish(ctx, ValueResultTargetResisted, "")
	return ToppleSaveTargetResisted{Detail: detail}, nil
}

func forcedFailure(reason d20.ForcedFailureReason) (ToppleSaveBasis, error) {
	f, err := d20.FailWithoutRoll(d20.SavingThrow, reason)
	return ToppleSaveForced{Failure: f}, err
}

// roll makes the one request of the attempt, sized by the roll mode its
// circumstances resolve to, and builds the d20.Roll. A roller failure, panic or
// broken contract comes back as a *srderr.PortError.
func (u ToppleSaveResolver) roll(a d20.RollAttempt) (d20.Roll, error) {
	mode := modifier.Resolve(a.Circumstances...)
	expr, err := dice.NewExpression(mode.DiceCount(), dice.D20(), 0)
	if err != nil { // unreachable: the count is 1 or 2 and the die is a d20
		return d20.Roll{}, err
	}
	res, err := dice.Roll(u.roller, expr)
	if err != nil {
		return d20.Roll{}, err
	}
	faces := make([]int, 0, mode.DiceCount())
	for _, r := range res.Rolls() {
		faces = append(faces, r.Value())
	}
	// An error is unreachable: dice.Roll vouches for the count and faces.
	return d20.NewRoll(mode, faces)
}
