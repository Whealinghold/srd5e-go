package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/modifier"
)

// RollCombatantDeathSavingThrowCommand is the immutable input of RollCombatantDeathSavingThrow.Run. Build it
// with NewRollCombatantDeathSavingThrowCommand; the zero value is invalid.
type RollCombatantDeathSavingThrowCommand struct {
	encounter     Encounter
	combatant     CombatantID
	circumstances []modifier.RollCircumstance
	modifiers     []modifier.Modifier
}

// NewRollCombatantDeathSavingThrowCommand returns the command, taking its own copy of the lists. circumstances
// are the Advantage and Disadvantage the host states (no condition gives one); modifiers are the host's
// modifiers to the total; both default to none. It returns an error matching srderr.ErrInvalidInput for a zero
// encounter, a blank id or a nil circumstance. It never panics.
func NewRollCombatantDeathSavingThrowCommand(encounter Encounter, combatant CombatantID, circumstances []modifier.RollCircumstance, modifiers []modifier.Modifier) (RollCombatantDeathSavingThrowCommand, error) {
	if encounter.IsZero() || combatant.IsZero() || slices.Contains(circumstances, nil) {
		return RollCombatantDeathSavingThrowCommand{}, invalidInput("a Death Saving Throw needs an encounter, a combatant and circumstances that are not nil")
	}
	return RollCombatantDeathSavingThrowCommand{encounter, combatant, slices.Clone(circumstances), slices.Clone(modifiers)}, nil
}

// RollCombatantDeathSavingThrowResult is the closed set of outcomes of RollCombatantDeathSavingThrow.Run.
// Variants: RollCombatantDeathSavingThrowCompleted, RollCombatantDeathSavingThrowRejected,
// RollCombatantDeathSavingThrowInvalid.
type RollCombatantDeathSavingThrowResult interface{ isRollCombatantDeathSavingThrowResult() }

// RollCombatantDeathSavingThrowCompleted: the throw was rolled and applied. It holds the new encounter, the
// events in order, the natural face and the total.
type RollCombatantDeathSavingThrowCompleted struct {
	resolutionCompleted
	natural, total int
}

// RollCombatantDeathSavingThrowRejected: a rule stopped the throw before any die; the encounter is as it came in.
type RollCombatantDeathSavingThrowRejected struct{ resolutionRejected }

// RollCombatantDeathSavingThrowInvalid: the host named a combatant the encounter does not hold; the encounter is
// as it came in.
type RollCombatantDeathSavingThrowInvalid struct{ resolutionInvalid }

func (RollCombatantDeathSavingThrowCompleted) isRollCombatantDeathSavingThrowResult() {}
func (RollCombatantDeathSavingThrowRejected) isRollCombatantDeathSavingThrowResult()  {}
func (RollCombatantDeathSavingThrowInvalid) isRollCombatantDeathSavingThrowResult()   {}

// Natural returns the face of the d20 that counts: the kept die under Advantage or Disadvantage.
func (c RollCombatantDeathSavingThrowCompleted) Natural() int { return c.natural }

// Total returns the natural face plus the modifiers less the Exhaustion reduction; it decides success unless
// the face is 1 or 20.
func (c RollCombatantDeathSavingThrowCompleted) Total() int { return c.total }

// RollCombatantDeathSavingThrow is the use case of a Death Saving Throw (SRD 5.2.1, "Death Saving Throws",
// p. 17). It plays the role of ApplyDeathSavingThrowToCombatant and rolls the die itself. Create it with
// NewRollCombatantDeathSavingThrow.
type RollCombatantDeathSavingThrow struct{ resolutionUse }

// NewRollCombatantDeathSavingThrow returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewRollCombatantDeathSavingThrow(roller Roller, opts ...Option) (RollCombatantDeathSavingThrow, error) {
	if roller == nil {
		return RollCombatantDeathSavingThrow{}, invalidInput("a Death Saving Throw needs a dice roller")
	}
	return RollCombatantDeathSavingThrow{newResolutionUse(roller, opts)}, nil
}

// Run rolls the throw. It stops, before any die, for an ended encounter, an unknown combatant, a dead one, one
// that is not Dying (NotDying), one that is not the active combatant (DeathSavingThrowOutOfTurnRejection) and
// one that already made its throw this turn (DeathSavingThrowAlreadyMade). Otherwise it rolls 1d20, or 2d20
// under net Advantage or Disadvantage, takes the kept die as the natural face, adds the modifiers and subtracts
// the Exhaustion reduction of the roller's own conditions, and hands both to the encounter. It never asks the
// conditions for an ability. It returns the errors described for the package.
func (u RollCombatantDeathSavingThrow) Run(ctx context.Context, cmd RollCombatantDeathSavingThrowCommand) (RollCombatantDeathSavingThrowResult, error) {
	if u.flow.roller == nil || cmd.encounter.IsZero() {
		return nil, invalidInput("zero use case or command")
	}
	op := combatOperation{valueOperationRollCombatantDeathSavingThrow, spanRollCombatantDeathSavingThrow}
	return execute(ctx, u.resolutionUse, op, func() (RollCombatantDeathSavingThrowResult, error) { return u.roll(cmd) })
}

// roll is the work of Run.
func (u RollCombatantDeathSavingThrow) roll(cmd RollCombatantDeathSavingThrowCommand) (RollCombatantDeathSavingThrowResult, error) {
	if s := preconditions(cmd.encounter, cmd.combatant); s.stopped() {
		if s.problem != nil {
			return RollCombatantDeathSavingThrowInvalid{resolutionInvalid{s.problem, cmd.encounter}}, nil
		}
		return RollCombatantDeathSavingThrowRejected{resolutionRejected{s.rejection, cmd.encounter}}, nil
	}
	if rej := (ResolutionPrecedence{}).DeathSavingThrow(cmd.encounter, cmd.combatant); rej != nil {
		return RollCombatantDeathSavingThrowRejected{resolutionRejected{rej, cmd.encounter}}, nil
	}
	roller, _ := cmd.encounter.Find(cmd.combatant)
	roll, err := u.flow.rollD20(modifier.Resolve(cmd.circumstances...))
	if err != nil {
		return nil, err
	}
	natural := roll.Natural()
	total := natural - (AttackConditionInputs{}).ExhaustionReduction(roller.conditions)
	for _, m := range cmd.modifiers {
		total = satAdd(total, m.Value())
	}
	tr, err := cmd.encounter.ApplyDeathSavingThrow(cmd.combatant, natural, total)
	if err != nil {
		return nil, err
	}
	r := &resolution{enc: cmd.encounter}
	r.mustCommit(tr)
	r.prompts(cmd.encounter)
	if r.overLimit() {
		return RollCombatantDeathSavingThrowInvalid{resolutionInvalid{LimitExceededProblem{}, cmd.encounter}}, nil
	}
	if r.exitInvalid() {
		return nil, invalidInput("the encounter is not valid after the call")
	}
	return RollCombatantDeathSavingThrowCompleted{resolutionCompleted{r.enc, r.events}, natural, total}, nil
}
