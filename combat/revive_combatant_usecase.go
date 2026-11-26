package combat

import "context"

// The use case of returning a Dead creature to life (returning-to-life.md FR-016; combat-action-resolution.md
// FR-167, FR-170): it wraps Revive over host-held values, with no Encounter and no ActionLedger (Q-5), and returns
// Revive's own closed result unchanged.

// maxElapsedConditions is the cap of plan P10 on ElapsedConditions: a creature holds one explicit condition per
// kind, so at most 15 can lapse. maxEffectsPerCreature caps ElapsedEffects; the held Effects list is capped at the same number by NewOngoingEffects.
const maxElapsedConditions = 15

// ReviveCombatantCommand is the input of ReviveCombatant.Run: the command of Revive.
type ReviveCombatantCommand ReviveCreatureCommand

// ReviveCombatantResult is the closed set of outcomes of ReviveCombatant.Run: the answers of Revive, unchanged
// (a ReviveCreatureResult converts to it as it is). Variants: ReviveCreatureApplied, ReviveCreatureNotDead,
// ReviveCreatureRefused, ReviveCreatureHitPointsBelowOne, ReviveCreatureMaximumBelowOne.
type ReviveCombatantResult interface{ isReviveCreatureResult() }

// ReviveCombatant is the use case of returning a Dead creature to life: it wraps Revive. Create it with
// NewReviveCombatant. Safe for concurrent use if its telemetry and clock are.
type ReviveCombatant struct{ in instrumenter }

// NewReviveCombatant returns the use case. Options: WithTelemetry, WithClock.
func NewReviveCombatant(opts ...Option) ReviveCombatant {
	return ReviveCombatant{newWired(opts)}
}

// Run returns the creature to life. The result is Revive's own: completed (ReviveCreatureApplied, applied) or
// revive_not_dead, revive_refused, revive_hit_points_below_one or revive_maximum_below_one (refused, never invalid).
// It returns an error matching srderr.ErrInvalidInput, recorded on the span only, for more than 15 elapsed
// conditions or more than 64 elapsed effect ids or held effects (plan P10, limit_exceeded), and the context's
// error, unwrapped and before any signal, for a cancelled ctx.
func (u ReviveCombatant) Run(ctx context.Context, cmd ReviveCombatantCommand) (ReviveCombatantResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationReviveCombatant, spanReviveCombatant},
		func() (ReviveCombatantResult, error) {
			if len(cmd.ElapsedConditions) > maxElapsedConditions || len(cmd.ElapsedEffects) > maxEffectsPerCreature {
				return nil, invalidInput("a revival holds at most 15 elapsed conditions, and 64 elapsed effects (limit_exceeded)")
			}
			return Revive(ReviveCreatureCommand(cmd)), nil
		},
		func(r ReviveCombatantResult) description {
			switch v := r.(type) {
			case ReviveCreatureApplied:
				return appliedWith(valueResultCompleted, v.Events()...)
			case ReviveCreatureNotDead:
				return refusedWith(valueResultReviveNotDead)
			case ReviveCreatureRefused:
				return refusedWith(valueResultReviveRefused)
			case ReviveCreatureHitPointsBelowOne:
				return refusedWith(valueResultReviveHitPointsBelowOne)
			case ReviveCreatureMaximumBelowOne:
				return refusedWith(valueResultReviveMaximumBelowOne)
			default:
				panic(unreachableError("revive creature result", r))
			}
		})
}
