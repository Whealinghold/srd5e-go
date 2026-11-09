package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// validSaveAttempt reports whether a is one of the three d20.SaveAttempt kinds.
func validSaveAttempt(a d20.SaveAttempt) bool {
	switch a.(type) {
	case d20.RollAttempt, d20.ChooseToFail, d20.AutomaticFailure:
		return true
	}
	return false
}

// saveKind is the span attribute of a use case that makes a saving throw.
func saveKind() telemetry.Attr { return telemetry.String(d20.AttrKind, d20.ValueKindSavingThrow) }

// ResolveSentientItemConflictCommand is the input of
// ResolveSentientItemConflict.Do: the item's Charisma, the bearer's saving throw
// attempt (normally a d20.RollAttempt with the bearer's own modifiers) and the
// item's demands. Construct it with a keyed literal. Do does not alter it.
type ResolveSentientItemConflictCommand struct {
	ItemCharisma ability.Score
	Attempt      d20.SaveAttempt
	Demands      SentientDemands
}

// ResolveSentientItemConflictResult is the closed set of outcomes of
// ResolveSentientItemConflict.Do. Variants: ResolveSentientItemConflictResisted,
// ResolveSentientItemConflictDemandsMade, ResolveSentientItemConflictRejected.
type ResolveSentientItemConflictResult interface{ isResolveSentientItemConflictResult() }

// ResolveSentientItemConflictResisted: the bearer passed the save.
type ResolveSentientItemConflictResisted struct{ save SentientSave }

// Save returns how the save was decided.
func (r ResolveSentientItemConflictResisted) Save() SentientSave { return r.save }

// ResolveSentientItemConflictDemandsMade: the bearer failed the save and the item
// makes its demands.
type ResolveSentientItemConflictDemandsMade struct {
	save    SentientSave
	demands SentientDemands
}

// Save returns how the save was decided.
func (r ResolveSentientItemConflictDemandsMade) Save() SentientSave { return r.save }

// Demands returns the item's demands.
func (r ResolveSentientItemConflictDemandsMade) Demands() SentientDemands { return r.demands }

// ResolveSentientItemConflictRejected: the demands list was not valid (built
// without NewSentientDemands); no save was made.
type ResolveSentientItemConflictRejected struct{}

func (ResolveSentientItemConflictResisted) isResolveSentientItemConflictResult()    {}
func (ResolveSentientItemConflictDemandsMade) isResolveSentientItemConflictResult() {}
func (ResolveSentientItemConflictRejected) isResolveSentientItemConflictResult()    {}

// ResolveSentientItemConflict is the use case that resolves a conflict between a
// sentient item and its bearer: the bearer's Charisma saving throw against DC 12
// plus the item's Charisma modifier (SRD 5.2.1, "Conflict", p. 208). Create it
// with NewResolveSentientItemConflict; the zero value is not usable. It is safe
// for concurrent use only if its roller, telemetry and clock are.
type ResolveSentientItemConflict struct {
	save sentientSaveRoll
	sig  operationSignals
}

// NewResolveSentientItemConflict returns the use case rolling through roller.
// Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil roller.
func NewResolveSentientItemConflict(roller Roller, opts ...Option) (ResolveSentientItemConflict, error) {
	if roller == nil {
		return ResolveSentientItemConflict{}, invalidInput("resolve a sentient item conflict without a dice roller")
	}
	return ResolveSentientItemConflict{save: sentientSaveRoll{roller: roller}, sig: newOperationSignals(newOptions(opts))}, nil
}

// Do resolves the conflict. A failed save, rolled, chosen or automatic, is
// ResolveSentientItemConflictDemandsMade; a passed one is ...Resisted; both are
// normal outcomes. An invalid demands list is ...Rejected and makes no request.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero item Charisma or a nil or foreign attempt, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the
// roller fails; the failure is recorded once and never logged.
func (u ResolveSentientItemConflict) Do(ctx context.Context, cmd ResolveSentientItemConflictCommand) (ResolveSentientItemConflictResult, error) {
	if u.save.roller == nil || cmd.ItemCharisma == (ability.Score{}) || !validSaveAttempt(cmd.Attempt) {
		return nil, invalidInput("resolve sentient item conflict: zero use case or Charisma, or nil or foreign attempt")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanResolveSentientConflict, ValueOperationResolveSentientConflict)
	defer run.end(ctx)
	if cmd.Demands.IsZero() || len(cmd.Demands.demands) > 4 {
		run.finish(ctx, ValueResultRejected, ValueRejectionDemandsInvalid)
		return ResolveSentientItemConflictRejected{}, nil
	}
	run.span.SetAttributes(saveKind())
	save, err := u.save.roll(cmd.Attempt, cmd.ItemCharisma)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if save.Failed() {
		run.finish(ctx, ValueResultDemandsMade, "")
		return ResolveSentientItemConflictDemandsMade{save: save, demands: cmd.Demands}, nil
	}
	run.finish(ctx, ValueResultResisted, "")
	return ResolveSentientItemConflictResisted{save: save}, nil
}

// AttemptSentientItemControlCommand is the input of AttemptSentientItemControl.Do:
// the bearer, whether the item can use its power, the item's Charisma and the
// bearer's saving throw attempt. Construct it with a keyed literal. Do does not
// alter it.
type AttemptSentientItemControlCommand struct {
	CharacterID  Identifier
	Power        SentientControlPower
	ItemCharisma ability.Score
	Attempt      d20.SaveAttempt
}

// AttemptSentientItemControlResult is the closed set of outcomes of
// AttemptSentientItemControl.Do. Variants: AttemptSentientItemControlTaken,
// AttemptSentientItemControlResisted, AttemptSentientItemControlRejected.
type AttemptSentientItemControlResult interface{ isAttemptSentientItemControlResult() }

// AttemptSentientItemControlTaken: the bearer failed the save and is controlled.
type AttemptSentientItemControlTaken struct {
	save   SentientSave
	effect SentientControlEffect
	power  SentientControlPower
	event  SentientControlTaken
}

// Save returns how the save was decided.
func (r AttemptSentientItemControlTaken) Save() SentientSave { return r.save }

// Effect returns the control: hours, Charmed, must follow the item's commands.
func (r AttemptSentientItemControlTaken) Effect() SentientControlEffect { return r.effect }

// Power returns ControlPowerSpentUntilNextDawn.
func (r AttemptSentientItemControlTaken) Power() SentientControlPower { return r.power }

// Event returns the SentientControlTaken event.
func (r AttemptSentientItemControlTaken) Event() SentientControlTaken { return r.event }

// AttemptSentientItemControlResisted: the bearer passed the save.
type AttemptSentientItemControlResisted struct {
	save  SentientSave
	power SentientControlPower
}

// Save returns how the save was decided.
func (r AttemptSentientItemControlResisted) Save() SentientSave { return r.save }

// Power returns ControlPowerSpentUntilNextDawn.
func (r AttemptSentientItemControlResisted) Power() SentientControlPower { return r.power }

// AttemptSentientItemControlRejected: the power is spent until the next dawn; no
// save was made.
type AttemptSentientItemControlRejected struct{}

func (AttemptSentientItemControlTaken) isAttemptSentientItemControlResult()    {}
func (AttemptSentientItemControlResisted) isAttemptSentientItemControlResult() {}
func (AttemptSentientItemControlRejected) isAttemptSentientItemControlResult() {}

// AttemptSentientItemControl is the use case in which a sentient item tries to
// take control of its bearer. Whether or not it succeeds, the item cannot use the
// power again until the next dawn. The host applies the Charmed condition. Create
// it with NewAttemptSentientItemControl; the zero value is not usable. It is safe
// for concurrent use only if its roller, telemetry and clock are.
type AttemptSentientItemControl struct {
	save sentientSaveRoll
	sig  operationSignals
}

// NewAttemptSentientItemControl returns the use case rolling through roller.
// Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil roller.
func NewAttemptSentientItemControl(roller Roller, opts ...Option) (AttemptSentientItemControl, error) {
	if roller == nil {
		return AttemptSentientItemControl{}, invalidInput("attempt sentient item control without a dice roller")
	}
	return AttemptSentientItemControl{save: sentientSaveRoll{roller: roller}, sig: newOperationSignals(newOptions(opts))}, nil
}

// Do makes the save, then on a failure rolls 1d12 for the hours. A spent power is
// AttemptSentientItemControlRejected and makes no save and no request.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero bearer or Charisma, an unknown power or a nil or foreign
// attempt, and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled. It returns a *srderr.PortError (error type "dice_roller_error") with
// a nil result when the roller fails; the failure is recorded once and never logged.
func (u AttemptSentientItemControl) Do(ctx context.Context, cmd AttemptSentientItemControlCommand) (AttemptSentientItemControlResult, error) {
	if u.save.roller == nil || cmd.CharacterID.IsZero() || cmd.ItemCharisma == (ability.Score{}) || !validSaveAttempt(cmd.Attempt) ||
		cmd.Power != ControlPowerAvailable && cmd.Power != ControlPowerSpentUntilNextDawn {
		return nil, invalidInput("attempt sentient item control: zero use case, bearer or Charisma, unknown power, or nil or foreign attempt")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanAttemptSentientControl, ValueOperationAttemptSentientControl)
	defer run.end(ctx)
	if cmd.Power == ControlPowerSpentUntilNextDawn {
		run.finish(ctx, ValueResultRejected, ValueRejectionPowerSpent)
		return AttemptSentientItemControlRejected{}, nil
	}
	run.span.SetAttributes(saveKind())
	save, err := u.save.roll(cmd.Attempt, cmd.ItemCharisma)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if !save.Failed() {
		run.finish(ctx, ValueResultResisted, "")
		return AttemptSentientItemControlResisted{save: save, power: ControlPowerSpentUntilNextDawn}, nil
	}
	expr, err := dice.NewExpression(1, dice.D12(), 0)
	if err != nil { // unreachable: one d12
		return nil, err
	}
	res, err := dice.Roll(u.save.roller, expr)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	effect, err := NewSentientControlEffect(res.Total())
	if err != nil { // unreachable: dice.Roll vouches for 1 to 12
		return nil, err
	}
	run.finish(ctx, ValueResultChanged, "")
	return AttemptSentientItemControlTaken{save: save, effect: effect, power: ControlPowerSpentUntilNextDawn,
		event: SentientControlTaken{id: cmd.CharacterID, hours: effect.hours}}, nil
}

// RepeatSentientControlSaveCommand is the input of RepeatSentientControlSave.Do:
// the bearer, the item copy controlling them, the item's Charisma and the
// bearer's saving throw attempt. Construct it with a keyed literal. Do does not
// alter it.
type RepeatSentientControlSaveCommand struct {
	CharacterID  Identifier
	Instance     MagicItemInstanceID
	ItemCharisma ability.Score
	Attempt      d20.SaveAttempt
}

// RepeatSentientControlSaveResult is the closed set of outcomes of
// RepeatSentientControlSave.Do. Variants: RepeatSentientControlSaveEnded,
// RepeatSentientControlSaveContinues.
type RepeatSentientControlSaveResult interface{ isRepeatSentientControlSaveResult() }

// RepeatSentientControlSaveEnded: the bearer passed the save and the control ends.
type RepeatSentientControlSaveEnded struct {
	save  SentientSave
	event SentientControlEnded
}

// Save returns how the save was decided.
func (r RepeatSentientControlSaveEnded) Save() SentientSave { return r.save }

// Event returns the SentientControlEnded event.
func (r RepeatSentientControlSaveEnded) Event() SentientControlEnded { return r.event }

// RepeatSentientControlSaveContinues: the bearer failed the save and stays controlled.
type RepeatSentientControlSaveContinues struct{ save SentientSave }

// Save returns how the save was decided.
func (r RepeatSentientControlSaveContinues) Save() SentientSave { return r.save }

func (RepeatSentientControlSaveEnded) isRepeatSentientControlSaveResult()     {}
func (RepeatSentientControlSaveContinues) isRepeatSentientControlSaveResult() {}

// RepeatSentientControlSave is the use case in which a controlled bearer who
// takes damage repeats the save. Create it with NewRepeatSentientControlSave; the
// zero value is not usable. It is safe for concurrent use only if its roller,
// telemetry and clock are.
type RepeatSentientControlSave struct {
	save sentientSaveRoll
	sig  operationSignals
}

// NewRepeatSentientControlSave returns the use case rolling through roller.
// Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil roller.
func NewRepeatSentientControlSave(roller Roller, opts ...Option) (RepeatSentientControlSave, error) {
	if roller == nil {
		return RepeatSentientControlSave{}, invalidInput("repeat a sentient control save without a dice roller")
	}
	return RepeatSentientControlSave{save: sentientSaveRoll{roller: roller}, sig: newOperationSignals(newOptions(opts))}, nil
}

// Do makes the save: a success ends the control, a failure keeps it.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero bearer, a blank instance, a zero Charisma or a nil or
// foreign attempt, and ctx.Err() unwrapped, before any signal, when ctx is
// already cancelled. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails; the failure is
// recorded once and never logged.
func (u RepeatSentientControlSave) Do(ctx context.Context, cmd RepeatSentientControlSaveCommand) (RepeatSentientControlSaveResult, error) {
	if u.save.roller == nil || cmd.CharacterID.IsZero() || !cmd.Instance.Valid() || cmd.ItemCharisma == (ability.Score{}) || !validSaveAttempt(cmd.Attempt) {
		return nil, invalidInput("repeat sentient control save: zero use case, bearer, instance or Charisma, or nil or foreign attempt")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRepeatSentientControlSave, ValueOperationRepeatSentientControl)
	defer run.end(ctx)
	run.span.SetAttributes(saveKind())
	save, err := u.save.roll(cmd.Attempt, cmd.ItemCharisma)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if save.Failed() {
		run.finish(ctx, ValueResultUnchanged, "")
		return RepeatSentientControlSaveContinues{save: save}, nil
	}
	run.finish(ctx, ValueResultChanged, "")
	return RepeatSentientControlSaveEnded{save: save, event: SentientControlEnded{id: cmd.CharacterID, instance: cmd.Instance}}, nil
}
