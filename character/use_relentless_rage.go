package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseRelentlessRageCommand is the input of UseRelentlessRage.Run: the character, the host
// fact that it dropped to 0 Hit Points and did not die outright, the RollMode of the
// Constitution saving throw and its modifiers (from the derived stats: the Constitution
// modifier, and the Proficiency Bonus only with Constitution save proficiency, which a
// multiclass Barbarian lacks). Construct it with a keyed literal. Run does not alter it.
type UseRelentlessRageCommand struct {
	Character Character
	Facts     BarbarianHostFacts
	RollMode  modifier.RollMode
	Modifiers d20.RollModifiers
}

// UseRelentlessRageResult is the closed set of outcomes of UseRelentlessRage.Run. Variants:
// RelentlessRageSucceeded, RelentlessRageStillFailed, RelentlessRageNotAvailable.
type UseRelentlessRageResult interface{ isUseRelentlessRageResult() }

type relentlessRageUsed struct {
	character Character
	hitPoints int
	check     d20.CheckResult
	events    []Event
}

// RelentlessRageSucceeded is the outcome of a Constitution saving throw that met the DC. Only
// Run builds one.
type RelentlessRageSucceeded struct{ used relentlessRageUsed }

// Character returns the new Character, with one more use counted since the last rest.
func (r RelentlessRageSucceeded) Character() Character { return r.used.character }

// HitPoints returns the Hit Points to set: twice the Barbarian level, for the host to apply.
func (r RelentlessRageSucceeded) HitPoints() int { return r.used.hitPoints }

// Check returns the saving throw, whose Roll holds the d20s rolled.
func (r RelentlessRageSucceeded) Check() d20.CheckResult { return r.used.check }

// NextDC returns the Difficulty Class of the next use.
func (r RelentlessRageSucceeded) NextDC() int {
	return int(r.used.check.Target()) + relentlessRageDCStep
}

// Events returns a new slice holding the RelentlessRageUsed event.
func (r RelentlessRageSucceeded) Events() []Event { return slices.Clone(r.used.events) }

// RelentlessRageStillFailed is the outcome of a Constitution saving throw below the DC: a normal
// rule outcome, not an error, and the use is counted all the same. Only Run builds one.
type RelentlessRageStillFailed struct{ used relentlessRageUsed }

// Character returns the new Character, with one more use counted since the last rest.
func (r RelentlessRageStillFailed) Character() Character { return r.used.character }

// Check returns the saving throw, whose Roll holds the d20s rolled.
func (r RelentlessRageStillFailed) Check() d20.CheckResult { return r.used.check }

// NextDC returns the Difficulty Class of the next use.
func (r RelentlessRageStillFailed) NextDC() int {
	return int(r.used.check.Target()) + relentlessRageDCStep
}

// Events returns a new slice holding the RelentlessRageUsed event.
func (r RelentlessRageStillFailed) Events() []Event { return slices.Clone(r.used.events) }

// RelentlessRageNotAvailable is the outcome of a use a rule refused: the Character is unchanged
// and the roller was not called. Construct it with a keyed literal.
type RelentlessRageNotAvailable struct{ Reason BarbarianReason }

func (RelentlessRageSucceeded) isUseRelentlessRageResult()    {}
func (RelentlessRageStillFailed) isUseRelentlessRageResult()  {}
func (RelentlessRageNotAvailable) isUseRelentlessRageResult() {}

// UseRelentlessRage is the use case that uses a Barbarian's Relentless Rage (SRD 5.2.1, "Level
// 11: Relentless Rage", p. 30): it checks the use, rolls the d20s of the saving throw through the
// dice port only when it is available, composes the pure d20.ResolveCheck and emits the
// srd5e.character.feature_used signals. It holds no rules logic. Create it with
// NewUseRelentlessRage; the zero value is not usable. It is safe for concurrent use only if its
// roller, telemetry and clock are.
type UseRelentlessRage struct {
	progression BarbarianProgression
	roller      Roller
	sig         featureSignals
	set         bool
}

// NewUseRelentlessRage returns the use case. Options: WithTelemetry, WithClock; the others are
// ignored. It returns an error matching srderr.ErrInvalidInput for the zero progression or a nil
// roller.
func NewUseRelentlessRage(progression BarbarianProgression, roller Roller, opts ...Option) (UseRelentlessRage, error) {
	if progression.IsZero() || roller == nil {
		return UseRelentlessRage{}, invalidInput("use Relentless Rage without a Barbarian progression or a dice roller")
	}
	return UseRelentlessRage{progression: progression, roller: roller, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// Run uses Relentless Rage for cmd.Character. The reasons are checked in the order not a
// Barbarian, feature not gained, Rage not active, not dropped to 0 Hit Points; only an available
// use rolls: one d20 for a Normal mode, two for Advantage or Disadvantage.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a
// zero Character or an unknown RollMode, and ctx.Err() unwrapped, before any signal, when ctx is
// already cancelled. It returns a *srderr.PortError (error type "dice_roller_error") with a nil
// result when the roller fails, panics or breaks its contract; the Character is unchanged and the
// failure is recorded once and never logged. A refused use is a RelentlessRageNotAvailable result and
// a failed save is a RelentlessRageStillFailed result.
func (u UseRelentlessRage) Run(ctx context.Context, cmd UseRelentlessRageCommand) (UseRelentlessRageResult, error) {
	if err := barbarianCommandProblem(u.set, cmd.Character); err != nil {
		return nil, err
	}
	if cmd.RollMode < modifier.Normal || cmd.RollMode > modifier.WithDisadvantage {
		return nil, invalidInput("use Relentless Rage with an unknown roll mode")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanUseRelentlessRage, ValueClassBarbarian, ValueFeatureRelentlessRage)
	defer run.end(ctx)
	bc, reason, err := u.progression.contextOf(cmd.Character)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if reason == 0 {
		if un, ok := (RelentlessRage{}).Availability(bc.stats, bc.state, cmd.Facts).(RelentlessRageUnavailable); ok {
			reason = un.Reason
		}
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		return RelentlessRageNotAvailable{Reason: reason}, nil
	}
	count := 1 // a Normal roll is one d20, Advantage or Disadvantage two
	if cmd.RollMode != modifier.Normal {
		count = 2
	}
	dice20 := make([]int, count)
	for i := range dice20 {
		if dice20[i], err = rollOne(u.roller, dice.D20()); err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
	}
	roll, err := d20.NewRoll(cmd.RollMode, dice20)
	if err != nil { // unreachable: dice.Roll answers 1 to 20 and the count follows the mode
		run.commandError(err)
		return nil, err
	}
	check, err := d20.ResolveCheck(d20.SavingThrow, roll, cmd.Modifiers, d20.DifficultyClass(bc.state.RelentlessRageDC()))
	if err != nil { // unreachable: the roll is not zero and the kind is a saving throw
		run.commandError(err)
		return nil, err
	}
	res, err := RelentlessRage{}.Resolve(bc.stats, bc.state, check)
	if err != nil { // unreachable: the use was available and the check was built for it
		run.commandError(err)
		return nil, err
	}
	var out relentlessOutcome
	succeeded := false
	switch r := res.(type) {
	case RelentlessRageResolutionSucceeded:
		out, succeeded = r.outcome, true
	default:
		out = res.(RelentlessRageResolutionStillFailed).outcome
	}
	next, err := WithFeatureState(cmd.Character, BarbarianStateKey(), out.state)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		run.commandError(err)
		return nil, err
	}
	used := relentlessRageUsed{
		character: next, hitPoints: out.hitPoints, check: out.check,
		events: []Event{RelentlessRageUsed{id: next.ID(), succeeded: succeeded, hitPoints: out.hitPoints, nextDC: out.state.RelentlessRageDC()}},
	}
	if succeeded {
		run.succeeded(ctx)
		return RelentlessRageSucceeded{used: used}, nil
	}
	run.stillFailed(ctx)
	return RelentlessRageStillFailed{used: used}, nil
}
