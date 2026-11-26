package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// MoveCombatantCommand is the immutable input of MoveCombatant.Run. Build it with NewMoveCombatantCommand; the
// zero value is invalid.
type MoveCombatantCommand struct {
	encounter Encounter
	combatant CombatantID
	request   MovementRequest
}

// NewMoveCombatantCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a zero
// encounter, a blank id, a nil request or a MoveAlong or StandUp not built by NewMoveAlong or NewStandUp. It
// never panics.
func NewMoveCombatantCommand(encounter Encounter, combatant CombatantID, request MovementRequest) (MoveCombatantCommand, error) {
	if encounter.IsZero() || combatant.IsZero() {
		return MoveCombatantCommand{}, invalidInput("a move needs an encounter and a combatant")
	}
	switch v := request.(type) {
	case MoveAlong:
		if _, err := NewMoveAlong(v.segments); err != nil {
			return MoveCombatantCommand{}, err
		}
	case DropProne:
	case StandUp:
		if _, err := NewStandUp(v.baseWalkingSpeedFeet); err != nil {
			return MoveCombatantCommand{}, err
		}
	default:
		return MoveCombatantCommand{}, invalidInput("a move needs a request")
	}
	return MoveCombatantCommand{encounter, combatant, request}, nil
}

// MoveCombatantResult is the closed set of outcomes of MoveCombatant.Run. Variants: MoveCombatantCompleted,
// MoveCombatantRejected, MoveCombatantInvalid.
type MoveCombatantResult interface{ isMoveCombatantResult() }

// MoveCombatantCompleted: the request was carried out. It holds the new encounter and the events in order.
// A move provokes no Opportunity Attack here (FR-025): the host decides who is provoked.
type MoveCombatantCompleted struct{ resolutionCompleted }

// MoveCombatantRejected: a rule stopped the request; the encounter is as it came in.
type MoveCombatantRejected struct{ resolutionRejected }

// MoveCombatantInvalid: the host named a combatant the encounter does not hold; the encounter is as it came in.
type MoveCombatantInvalid struct{ resolutionInvalid }

func (MoveCombatantCompleted) isMoveCombatantResult() {}
func (MoveCombatantRejected) isMoveCombatantResult()  {}
func (MoveCombatantInvalid) isMoveCombatantResult()   {}

// MoveCombatant is the use case of moving, dropping Prone and standing up (SRD 5.2.1, "Movement and Position",
// p. 14; "Prone [Condition]", p. 186). The tracker spends the movement; the conditions of the mover, which the
// tracker's profile Speeds do not follow, restrict it (FR-057 to FR-061, FR-124). Create it with
// NewMoveCombatant; the zero value is silent and works. It rolls no die.
type MoveCombatant struct{ resolutionUse }

// NewMoveCombatant returns the use case. Options: WithTelemetry, WithClock.
func NewMoveCombatant(opts ...Option) MoveCombatant {
	return MoveCombatant{newResolutionUse(nil, opts)}
}

// Run carries out the request, ending at the first stop: the stops of the encounter, the tracker's, then the
// restrictions of the mover's conditions and posture. Dropping Prone applies Prone and standing up removes it
// through the encounter, so an Immune mover is a rejection and nothing changes. It returns the results and
// errors described for the package.
func (u MoveCombatant) Run(ctx context.Context, cmd MoveCombatantCommand) (MoveCombatantResult, error) {
	if cmd.request == nil {
		return nil, invalidInput("zero command")
	}
	op := combatOperation{valueOperationMoveCombatant, spanMoveCombatant}
	return execute(ctx, u.resolutionUse, op, func() (MoveCombatantResult, error) { return u.move(cmd) })
}

// move is the work of Run.
func (u MoveCombatant) move(cmd MoveCombatantCommand) (MoveCombatantResult, error) {
	rejected := func(r ResolutionRejection) MoveCombatantResult {
		return MoveCombatantRejected{resolutionRejected{r, cmd.encounter}}
	}
	if s := preconditions(cmd.encounter, cmd.combatant); s.stopped() {
		if s.problem != nil {
			return MoveCombatantInvalid{resolutionInvalid{s.problem, cmd.encounter}}, nil
		}
		return rejected(s.rejection), nil
	}
	mover, _ := cmd.encounter.Find(cmd.combatant)
	tracker := cmd.encounter.tracker
	profile, _ := tracker.ProfileOf(cmd.combatant)
	walk := profile.Speeds.Walk()

	var step TrackerStep
	var restriction ResolutionRejection
	var err error
	switch v := cmd.request.(type) {
	case MoveAlong:
		step, err = tracker.Move(cmd.combatant, v.segments)
		if err == nil {
			restriction, err = (MovementChecks{}).Move(mover.conditions, walk, v.segments)
		}
	case DropProne:
		step, err = tracker.ChangePosture(cmd.combatant, DropProneChange{})
		if err == nil {
			restriction, err = (MovementChecks{}).DropProne(mover.conditions, walk)
		}
	case StandUp:
		var movement ProneMovement
		if movement, restriction, err = (MovementChecks{}).StandUp(mover.conditions, v.baseWalkingSpeedFeet); err == nil && movement != nil {
			step, err = tracker.ChangePosture(cmd.combatant, StandUpChange{movement})
		}
	}
	if err != nil {
		return nil, err
	}
	// The tracker's refusal comes before the restrictions of the conditions (FR-010 steps 5 and 6).
	if s, ok := step.(TrackerStepRejected); ok {
		return rejected(TrackerRejection{s.Reason}), nil
	}
	if restriction != nil {
		return rejected(restriction), nil
	}
	applied := step.(TrackerStepApplied) // a nil step only comes with a restriction
	r := &resolution{enc: cmd.encounter}
	if err := r.commitTurn(applied); err != nil {
		return nil, err
	}
	switch cmd.request.(type) {
	case DropProne:
		tr, err := r.enc.ApplyCondition(cmd.combatant, Prone{})
		if err != nil {
			return nil, err
		}
		if rej, _ := r.commit(tr); rej != nil {
			return rejectedByRoot(rej, rejected)
		}
	case StandUp:
		tr, err := r.enc.RemoveCondition(cmd.combatant, content.ConditionProne)
		if err != nil {
			return nil, err
		}
		r.mustCommit(tr)
	}
	r.prompts(cmd.encounter)
	// No events cap check: the events of this call are bounded by its per-call lists (plan P10), far under 4096.
	if r.exitInvalid() {
		return nil, invalidInput("the encounter is not valid after the call")
	}
	return MoveCombatantCompleted{resolutionCompleted{r.enc, r.events}}, nil
}

// rejectedByRoot ends a call whose posture change the root refused after the tracker accepted it: the only
// refusal that can happen is Immunity to Prone (FR-062); the encounter stays as it came in. Any other
// refusal is a broken invariant.
func rejectedByRoot(rej EncounterRejection, rejected func(ResolutionRejection) MoveCombatantResult) (MoveCombatantResult, error) {
	switch rej.(type) {
	case ImmuneToConditionRejection:
		return rejected(EncounterStateRejection{rej}), nil
	default:
		panic(unreachableError("encounter rejection", rej))
	}
}
