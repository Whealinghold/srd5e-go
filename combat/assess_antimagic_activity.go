package combat

import (
	"context"
	"slices"
)

// AssessAntimagicActivityCommand is the input of AssessAntimagicActivity.Run. Construct it with a keyed
// literal. Run copies the slice and changes nothing in the command.
type AssessAntimagicActivityCommand struct {
	// Activity is what the creature tries to do.
	Activity AntimagicActivity
	// ActorInside is whether the actor is inside the aura (FR-045 to FR-047: for a magic item, the item or
	// what it is on; for teleport, the origin or destination).
	ActorInside bool
	// TargetsInside are the targets inside the aura, each named once; ignored for ActivityUseMagicItemProperty.
	TargetsInside []SpellTargetRef
}

// AssessAntimagicActivityResult is the closed set of outcomes of AssessAntimagicActivity.Run. Variants:
// AntimagicActivityAssessed.
type AssessAntimagicActivityResult interface{ isAssessAntimagicActivityResult() }

// AntimagicActivityAssessed is the verdict on an activity: AntimagicPermits or AntimagicBlocks. Not
// comparable with ==.
type AntimagicActivityAssessed struct{ verdict AntimagicVerdict }

// Verdict returns AntimagicPermits or AntimagicBlocks.
func (r AntimagicActivityAssessed) Verdict() AntimagicVerdict { return r.verdict }

func (AntimagicActivityAssessed) isAssessAntimagicActivityResult() {}

// AssessAntimagicActivity is the use case that asks whether an Antimagic Field blocks an activity (SRD 5.2.1,
// "Antimagic Field", p. 109). The host asks it before it calls DispelOngoingSpells or CounterCastSpell
// (FR-019, FR-048). It rolls nothing and calls no other use case. Create it with NewAssessAntimagicActivity;
// the zero value is usable. It is safe for concurrent use only if its telemetry and clock are.
type AssessAntimagicActivity struct{ in instrumenter }

// NewAssessAntimagicActivity returns the use case. Options: WithTelemetry, WithClock.
func NewAssessAntimagicActivity(opts ...Option) AssessAntimagicActivity {
	o := newOptions(opts)
	return AssessAntimagicActivity{newInstrumenter(o.tel, o.clock)}
}

// Run decides the activity of cmd with AntimagicField.Blocks. It records the shared signals under the
// operation assess_antimagic_activity with the result completed (AntimagicPermits, applied) or
// antimagic_blocked (AntimagicBlocks, refused). It never logs and produces no event. It returns an error
// matching srderr.ErrInvalidInput, recorded on the span only, for an activity that is not defined or a
// target named twice or more than 256 targets inside (a LimitExceededError), and ctx.Err() unwrapped, before any signal, for a cancelled ctx.
func (u AssessAntimagicActivity) Run(ctx context.Context, cmd AssessAntimagicActivityCommand) (AssessAntimagicActivityResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	targets := slices.Clone(cmd.TargetsInside)
	res, err := run(ctx, u.in, combatOperation{valueOperationAssessAntimagicActivity, spanAssessAntimagicActivity},
		func(context.Context) (AssessAntimagicActivityResult, error) {
			verdict, err := AntimagicField{}.Blocks(cmd.Activity, cmd.ActorInside, targets)
			if err != nil {
				return nil, err
			}
			return AntimagicActivityAssessed{verdict}, nil
		},
		func(r AssessAntimagicActivityResult) description {
			switch v := r.(type) {
			case AntimagicActivityAssessed:
				if _, blocked := v.verdict.(AntimagicBlocks); blocked {
					return description{Class: resultRefused, Name: valueResultAntimagicBlocked}
				}
				return description{Class: resultApplied, Name: valueResultCompleted}
			default:
				panic(unreachableError("assess antimagic activity result", r))
			}
		})
	if err != nil {
		return nil, err
	}
	return res, nil
}
