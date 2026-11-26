package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// MakeDeathSavingThrowCommand is the input of MakeDeathSavingThrow.Run: the
// creature, its status, the roll mode (Advantage or Disadvantage) and a modifier
// that carries any Exhaustion reduction (SRD 5.2.1, "Death Saving Throws", p.
// 17; "Exhaustion", p. 181). Build it with NewMakeDeathSavingThrowCommand; the
// zero value is invalid.
type MakeDeathSavingThrowCommand struct {
	subject CombatantID
	status  HitPointStatus
	mode    modifier.RollMode
	mod     modifier.Modifier
}

// NewMakeDeathSavingThrowCommand returns the command. It returns an error
// matching srderr.ErrInvalidInput for a zero subject or status or an undefined
// roll mode.
func NewMakeDeathSavingThrowCommand(subject CombatantID, status HitPointStatus, mode modifier.RollMode, mod modifier.Modifier) (MakeDeathSavingThrowCommand, error) {
	if err := status.check(subject); err != nil {
		return MakeDeathSavingThrowCommand{}, err
	}
	if mode.String() == "unknown" {
		return MakeDeathSavingThrowCommand{}, invalidInput("undefined roll mode")
	}
	return MakeDeathSavingThrowCommand{subject, status, mode, mod}, nil
}

// MakeDeathSavingThrowResult is the closed set of outcomes of
// MakeDeathSavingThrow.Run. Variants: DeathSavingThrowApplied,
// DeathSavingThrowRejected.
type MakeDeathSavingThrowResult interface{ isMakeDeathSavingThrowResult() }

// DeathSavingThrowApplied is a throw that was made: what it did, the new status
// and the events.
type DeathSavingThrowApplied struct {
	appliedTransition
	outcome DeathSavingThrowOutcome
}

// Outcome returns what the throw did.
func (r DeathSavingThrowApplied) Outcome() DeathSavingThrowOutcome { return r.outcome }

// DeathSavingThrowRejected is a throw the rules refuse: the creature is Dead or
// not Dying. Nothing was rolled. Construct it with a keyed literal.
type DeathSavingThrowRejected struct{ Rejection HitPointRejection }

func (DeathSavingThrowApplied) isMakeDeathSavingThrowResult()  {}
func (DeathSavingThrowRejected) isMakeDeathSavingThrowResult() {}

// MakeDeathSavingThrow is the use case of a Death Saving Throw, a D20 Test that
// adds no ability modifier or proficiency. It uses the dice port. Create it with
// NewMakeDeathSavingThrow; the zero value is not usable. It is safe for
// concurrent use only if its roller, telemetry and clock are.
type MakeDeathSavingThrow struct {
	hitPointUse
	roller Roller
}

// NewMakeDeathSavingThrow returns the use case rolling through roller. Options:
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput
// for a nil roller.
func NewMakeDeathSavingThrow(roller Roller, opts ...Option) (MakeDeathSavingThrow, error) {
	if roller == nil {
		return MakeDeathSavingThrow{}, invalidInput("death saving throw without a dice roller")
	}
	return MakeDeathSavingThrow{newHitPointUse(opts), roller}, nil
}

// Run refuses a creature that is not Dying without rolling; otherwise it makes
// one request of 1d20 (2d20 under Advantage or Disadvantage), keeps the die the
// mode selects and applies it with the modifier through
// HitPointStatus.ApplyDeathSave. It records the shared signals plus
// srd5e.combat.death_saves, the refusal and the deaths.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero use case or command, and ctx.Err() unwrapped, before any signal, for
// a cancelled ctx. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or
// answers another expression; the failure is recorded once and never logged.
// Under the P12 rule no srd5e.d20_test signal is emitted. A refusal is a
// DeathSavingThrowRejected result, never an error.
func (u MakeDeathSavingThrow) Run(ctx context.Context, cmd MakeDeathSavingThrowCommand) (MakeDeathSavingThrowResult, error) {
	if u.roller == nil {
		return nil, invalidInput("zero use case")
	}
	if err := u.begin(ctx, cmd.status); err != nil {
		return nil, err
	}
	tr, err := u.run(ctx, combatOperation{valueOperationMakeDeathSavingThrow, spanMakeDeathSavingThrow},
		func() (HitPointTransition, error) {
			if r, no := cmd.status.deathSaveRefusal(); no {
				return refused(r), nil
			}
			res, err := dice.Roll(u.roller, dice.MustExpression(cmd.mode.DiceCount(), dice.D20(), 0))
			if err != nil {
				return nil, err
			}
			var faces []int
			for _, r := range res.Rolls() {
				faces = append(faces, r.Value())
			}
			natural, err := cmd.mode.Select(faces)
			if err != nil {
				return nil, err
			}
			return cmd.status.ApplyDeathSave(cmd.subject, natural, satAdd(natural, cmd.mod.Value()))
		},
		func(a HitPointTransitionApplied) string {
			switch outcome := a.events[0].(DeathSavingThrowMade).outcome; {
			case has[Died](a.events):
				return valueResultKilled
			case has[Recovered](a.events):
				return valueResultRecovered
			case has[BecameStable](a.events):
				return valueResultStabilized
			case outcome == DeathSavingThrowSuccess:
				return valueResultSucceeded
			}
			return valueResultFailed
		})
	if err != nil {
		return nil, err
	}
	return convert(tr, func(a HitPointTransitionApplied) MakeDeathSavingThrowResult {
		outcome := a.events[0].(DeathSavingThrowMade).outcome
		u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricDeathSaves, Unit: "{roll}",
			Description: "Death Saving Throws, by result."}).Add(ctx, 1, telemetry.String(attrDeathSaveResult, outcome.String()))
		return DeathSavingThrowApplied{appliedTransition{a.status, a.events}, outcome}
	}, func(r HitPointRejection) MakeDeathSavingThrowResult { return DeathSavingThrowRejected{Rejection: r} }), nil
}
