package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// ReadyCombatantSpellCommand is the input of ReadyCombatantSpell.Run: the arguments of ReadySpell. Construct it with a
// keyed literal.
type ReadyCombatantSpellCommand struct {
	Caster      CombatantID
	Trigger     ReadyTriggerID
	Spell       content.SpellID
	CastingTime content.CastingTime
}

// ReadyCombatantSpell is the use case of readying a spell (SRD 5.2.1, "Ready [Action]", p. 187): it wraps ReadySpell and
// returns its ReadyCombatantSpellResult. The result is completed (ReadySpellApplied, applied) or ready_not_an_action
// (ReadySpellNotAnAction, refused); a cast and its resources are the host's. Create it with NewReadyCombatantSpell.
type ReadyCombatantSpell struct{ in instrumenter }

// NewReadyCombatantSpell returns the use case. Options: WithTelemetry, WithClock.
func NewReadyCombatantSpell(opts ...Option) ReadyCombatantSpell {
	return ReadyCombatantSpell{newWired(opts)}
}

// Run readies the spell. It returns the errors described for the package.
func (u ReadyCombatantSpell) Run(ctx context.Context, cmd ReadyCombatantSpellCommand) (ReadyCombatantSpellResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationReadyCombatantSpell, spanReadyCombatantSpell},
		func() (ReadyCombatantSpellResult, error) {
			return ReadySpell(cmd.Caster, cmd.Trigger, cmd.Spell, cmd.CastingTime), nil
		},
		func(r ReadyCombatantSpellResult) description {
			switch v := r.(type) {
			case ReadySpellApplied:
				return appliedWith(valueResultCompleted, widen(v.Events)...)
			case ReadySpellNotAnAction:
				return refusedWith(valueResultReadyNotAnAction)
			default:
				panic(unreachableError("ready spell result", r))
			}
		})
}

// ReleaseCombatantSpellCommand is the input of ReleaseCombatantSpell.Run: the arguments of ReleaseHeldSpell.
type ReleaseCombatantSpellCommand struct {
	Held     HeldSpellInput
	Reaction ReactionTaken
}

// ReleaseCombatantSpell is the use case of releasing a held spell with a Reaction: it wraps ReleaseHeldSpell and
// returns its ReleaseCombatantSpellResult. The result is completed (applied), release_no_held_spell,
// release_not_a_readied_reaction or release_reaction_mismatch (refused). Create it with NewReleaseCombatantSpell.
type ReleaseCombatantSpell struct{ in instrumenter }

// NewReleaseCombatantSpell returns the use case. Options: WithTelemetry, WithClock.
func NewReleaseCombatantSpell(opts ...Option) ReleaseCombatantSpell {
	return ReleaseCombatantSpell{newWired(opts)}
}

// Run releases the held spell. It returns the errors described for the package.
func (u ReleaseCombatantSpell) Run(ctx context.Context, cmd ReleaseCombatantSpellCommand) (ReleaseCombatantSpellResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationReleaseCombatantSpell, spanReleaseCombatantSpell},
		func() (ReleaseCombatantSpellResult, error) { return ReleaseHeldSpell(cmd.Held, cmd.Reaction), nil },
		func(r ReleaseCombatantSpellResult) description {
			switch v := r.(type) {
			case ReleaseHeldSpellApplied:
				return appliedWith(valueResultCompleted, widen(v.Events)...)
			case ReleaseHeldSpellNoHeldSpell:
				return refusedWith(valueResultReleaseNoHeldSpell)
			case ReleaseHeldSpellNotAReadiedReaction:
				return refusedWith(valueResultReleaseNotAReadiedReaction)
			case ReleaseHeldSpellReactionMismatch:
				return refusedWith(valueResultReleaseReactionMismatch)
			default:
				panic(unreachableError("release held spell result", r))
			}
		})
}

// DissipateCombatantSpellCommand is the input of DissipateCombatantSpell.Run: the arguments of DissipateHeldSpell.
type DissipateCombatantSpellCommand struct {
	Held  HeldSpellInput
	Cause DissipationCause
}

// DissipateCombatantSpell is the use case of a held spell that dissipates: it wraps DissipateHeldSpell and returns its
// DissipateCombatantSpellResult. The result is completed (applied), dissipate_no_held_spell (refused) or dissipate_invalid_cause (invalid). Create it with
// NewDissipateCombatantSpell.
type DissipateCombatantSpell struct{ in instrumenter }

// NewDissipateCombatantSpell returns the use case. Options: WithTelemetry, WithClock.
func NewDissipateCombatantSpell(opts ...Option) DissipateCombatantSpell {
	return DissipateCombatantSpell{newWired(opts)}
}

// Run dissipates the held spell. It returns the errors described for the package.
func (u DissipateCombatantSpell) Run(ctx context.Context, cmd DissipateCombatantSpellCommand) (DissipateCombatantSpellResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationDissipateCombatantSpell, spanDissipateCombatantSpell},
		func() (DissipateCombatantSpellResult, error) { return DissipateHeldSpell(cmd.Held, cmd.Cause), nil },
		func(r DissipateCombatantSpellResult) description {
			switch v := r.(type) {
			case DissipateHeldSpellApplied:
				return appliedWith(valueResultCompleted, widen(v.Events)...)
			case DissipateHeldSpellNoHeldSpell:
				return refusedWith(valueResultDissipateNoHeldSpell)
			case DissipateHeldSpellInvalidCause:
				return invalidWith(valueResultDissipateInvalidCause)
			default:
				panic(unreachableError("dissipate held spell result", r))
			}
		})
}
