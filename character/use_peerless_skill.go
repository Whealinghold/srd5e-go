package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// UsePeerlessSkillCommand is the input of UsePeerlessSkill.Run. Subclass is the Bard subclass the character holds ("" for
// none). Failed is the ability check or attack roll the Bard failed. Construct it with a keyed literal. Run does not alter it.
type UsePeerlessSkillCommand struct {
	Character Character
	Subclass  content.SubclassID
	Failed    AdjustableD20Result
}

// UsePeerlessSkillResult is the closed set of outcomes of UsePeerlessSkill.Run. Variants: UsePeerlessSkillApplied,
// UsePeerlessSkillRejected.
type UsePeerlessSkillResult interface{ isUsePeerlessSkillResult() }

// UsePeerlessSkillApplied is the outcome of a use: the Bardic Inspiration die was rolled and added. The use is expended
// only when the result became a success. Only Run builds one.
type UsePeerlessSkillApplied struct {
	character Character
	applied   PeerlessSkillApplied
}

// Character returns the new Character: one Bardic Inspiration use expended on a success, unchanged otherwise.
func (r UsePeerlessSkillApplied) Character() Character { return r.character }

// Applied returns the adjusted result, whether the use was spent, the pool and the events.
func (r UsePeerlessSkillApplied) Applied() PeerlessSkillApplied { return r.applied }

// Events returns a new slice holding the BardicInspirationExpended event when the use was spent, and none otherwise.
func (r UsePeerlessSkillApplied) Events() []Event { return r.applied.Events() }

// UsePeerlessSkillRejected is the outcome of a use a rule refused: the Character is unchanged and nothing was rolled.
// Construct it with a keyed literal.
type UsePeerlessSkillRejected struct{ Reason BardReason }

func (UsePeerlessSkillApplied) isUsePeerlessSkillResult()  {}
func (UsePeerlessSkillRejected) isUsePeerlessSkillResult() {}

// UsePeerlessSkill is the use case that uses a College of Lore Bard's Peerless Skill (SRD 5.2.1, "Level 14: Peerless
// Skill", p. 35): it plans the use, rolls the Bardic Die through the dice port only for a use that is allowed, applies it
// and emits the srd5e.character.feature_used signals. It holds no rules logic. Create it with NewUsePeerlessSkill; the zero
// value is not usable. Safe for concurrent use only if its roller, telemetry and clock are.
type UsePeerlessSkill struct{ bardUse }

// NewUsePeerlessSkill returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil roller.
func NewUsePeerlessSkill(roller Roller, opts ...Option) (UsePeerlessSkill, error) {
	u, err := newBardUse("use Peerless Skill", roller, opts)
	return UsePeerlessSkill{u}, err
}

// Run uses Peerless Skill for cmd.Character on cmd.Failed. The reasons are checked in the order of PlanPeerlessSkill.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character or a nil
// result, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It returns an error matching
// srderr.ErrInvalidInput for a zero result or a Bard that holds no pool, and a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or breaks its contract; the failure is recorded once
// and never logged. A refused use is a result, and so is a use whose result still fails.
func (u UsePeerlessSkill) Run(ctx context.Context, cmd UsePeerlessSkillCommand) (UsePeerlessSkillResult, error) {
	ctx, run, err := u.start(ctx, cmd.Character.IsZero() || cmd.Failed == nil, SpanUsePeerlessSkill, ValueFeaturePeerlessSkill)
	if err != nil {
		return nil, err
	}
	defer run.end(ctx)
	facts, isBard, err := readBardFacts(cmd.Character)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if !isBard {
		run.rejected(ctx, BardNotABard.String())
		return UsePeerlessSkillRejected{Reason: BardNotABard}, nil
	}
	plan, reason, err := PlanPeerlessSkill(cmd.Character.ID(), facts.level, cmd.Subclass, facts.pool, facts.charisma, cmd.Failed)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		return UsePeerlessSkillRejected{Reason: reason}, nil
	}
	rolled, err := rollOne(u.roller, plan.Die().Die())
	if err != nil {
		fail(ctx, run, err)
		return nil, err
	}
	applied, err := ApplyPeerlessSkill(plan, rolled)
	if err != nil {
		fail(ctx, run, err)
		return nil, err
	}
	next := cmd.Character
	if applied.spent {
		if next, err = WithFeatureState(next, BardicInspirationPoolKey(), applied.pool); err != nil { // unreachable: the key is valid
			run.commandError(err)
			return nil, err
		}
	}
	run.succeeded(ctx)
	return UsePeerlessSkillApplied{character: next, applied: applied}, nil
}
