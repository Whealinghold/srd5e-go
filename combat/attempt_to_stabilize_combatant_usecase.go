package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// AttemptToStabilizeCombatantParams are the facts of one attempt to stabilize a Dying creature. Construct with a
// keyed literal; NewAttemptToStabilizeCombatantCommand checks them.
type AttemptToStabilizeCombatantParams struct {
	// Encounter is the encounter the attempt is made in.
	Encounter Encounter
	// Ledger is the ActionLedger of the host; the zero value is a new one.
	Ledger ActionLedger
	// HelperID and PatientID name the two combatants; the helper may be the patient, who then cannot act.
	HelperID, PatientID CombatantID
	// Check is the helper's Wisdom (Medicine) check with the modifiers the host knows (ability, proficiency):
	// a rolled attempt or an automatic failure. The conditions, Exhaustion and Help of the helper are added.
	Check d20.CheckAttempt
	// FearSource says whether the helper's source of fear is in line of sight.
	FearSource LineOfSight
	// Requires says what the check needs of the helper's senses; the zero value is CheckRequirementNone.
	Requires CheckRequirement
}

// AttemptToStabilizeCombatantCommand is the immutable input of AttemptToStabilizeCombatant.Run. Build it with
// NewAttemptToStabilizeCombatantCommand; the zero value is invalid.
type AttemptToStabilizeCombatantCommand struct {
	p AttemptToStabilizeCombatantParams
}

// NewAttemptToStabilizeCombatantCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for a zero encounter, a blank id, a check that is neither a RollAttempt nor an
// AutomaticFailure or a requirement that is not defined. It never panics.
func NewAttemptToStabilizeCombatantCommand(p AttemptToStabilizeCombatantParams) (AttemptToStabilizeCombatantCommand, error) {
	switch p.Check.(type) {
	case d20.RollAttempt, d20.AutomaticFailure:
	default:
		return AttemptToStabilizeCombatantCommand{}, invalidInput("a check is a rolled attempt or an automatic failure")
	}
	if p.Encounter.IsZero() || p.HelperID.IsZero() || p.PatientID.IsZero() ||
		(p.Requires != 0 && p.Requires != CheckRequirementSight && p.Requires != CheckRequirementHearing && p.Requires != CheckRequirementNone) {
		return AttemptToStabilizeCombatantCommand{}, invalidInput("a stabilizing attempt needs an encounter, a helper, a patient and a requirement")
	}
	return AttemptToStabilizeCombatantCommand{p}, nil
}

// AttemptToStabilizeCombatantResult is the closed set of outcomes of AttemptToStabilizeCombatant.Run. Variants:
// AttemptToStabilizeCombatantCompleted, AttemptToStabilizeCombatantRejected, AttemptToStabilizeCombatantInvalid.
type AttemptToStabilizeCombatantResult interface{ isAttemptToStabilizeCombatantResult() }

// AttemptToStabilizeCombatantCompleted: the helper spent its action and made the check. It holds the new
// encounter and ledger, the events in order, the check and whether the patient was stabilized. A failed check
// leaves the action spent and the patient as it was.
type AttemptToStabilizeCombatantCompleted struct {
	ledgerCompleted
	check      TestResolution
	stabilized bool
}

// AttemptToStabilizeCombatantRejected: a rule stopped the attempt before the action was spent and any die
// rolled; the encounter and ledger are as they came in.
type AttemptToStabilizeCombatantRejected struct{ ledgerRejected }

// AttemptToStabilizeCombatantInvalid: the host named a combatant the encounter does not hold; the encounter and
// ledger are as they came in.
type AttemptToStabilizeCombatantInvalid struct{ ledgerInvalid }

func (AttemptToStabilizeCombatantCompleted) isAttemptToStabilizeCombatantResult() {}
func (AttemptToStabilizeCombatantRejected) isAttemptToStabilizeCombatantResult()  {}
func (AttemptToStabilizeCombatantInvalid) isAttemptToStabilizeCombatantResult()   {}

// Check returns the helper's check against DC 10.
func (c AttemptToStabilizeCombatantCompleted) Check() TestResolution { return c.check }

// Stabilized reports whether the check succeeded and the patient is Stable.
func (c AttemptToStabilizeCombatantCompleted) Stabilized() bool { return c.stabilized }

// AttemptToStabilizeCombatant is the use case of stabilizing a Dying creature (SRD 5.2.1, "Stabilizing a
// Character", p. 18): the helper takes the Help action and makes a DC 10 Wisdom (Medicine) check. It plays the
// role of StabilizeCombatant and adds the action and the roll. Create it with NewAttemptToStabilizeCombatant.
type AttemptToStabilizeCombatant struct{ resolutionUse }

// NewAttemptToStabilizeCombatant returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewAttemptToStabilizeCombatant(roller Roller, opts ...Option) (AttemptToStabilizeCombatant, error) {
	if roller == nil {
		return AttemptToStabilizeCombatant{}, invalidInput("a stabilizing attempt needs a dice roller")
	}
	return AttemptToStabilizeCombatant{newResolutionUse(roller, opts)}, nil
}

// Run makes the attempt. It stops, before the action is spent and any die rolled, in this order: the encounter
// ended; an unknown, then a dead helper or patient (the helper first within each); a patient that is not Dying,
// a knocked-out creature at 1 Hit Point included; the helper's tracker refusal (an Incapacitated helper, among
// others). It then adds the contributions of the helper's own conditions to its check, the Exhaustion reduction
// and a pending Help benefit for a Wisdom (Medicine) check, and rolls; a forced failure rolls no die. It
// returns the errors described for the package.
func (u AttemptToStabilizeCombatant) Run(ctx context.Context, cmd AttemptToStabilizeCombatantCommand) (AttemptToStabilizeCombatantResult, error) {
	if u.flow.roller == nil || cmd.p.Check == nil {
		return nil, invalidInput("zero use case or command")
	}
	op := combatOperation{valueOperationAttemptToStabilizeCombatant, spanAttemptToStabilizeCombatant}
	return execute(ctx, u.resolutionUse, op, func() (AttemptToStabilizeCombatantResult, error) { return u.attempt(cmd.p) })
}

// attempt is the work of Run.
func (u AttemptToStabilizeCombatant) attempt(p AttemptToStabilizeCombatantParams) (AttemptToStabilizeCombatantResult, error) {
	rejected := func(r ResolutionRejection) AttemptToStabilizeCombatantResult {
		return AttemptToStabilizeCombatantRejected{ledgerRejected{resolutionRejected{r, p.Encounter}, p.Ledger}}
	}
	if s := preconditions(p.Encounter, p.HelperID, p.PatientID); s.stopped() {
		if s.problem != nil {
			return AttemptToStabilizeCombatantInvalid{ledgerInvalid{resolutionInvalid{s.problem, p.Encounter}, p.Ledger}}, nil
		}
		return rejected(s.rejection), nil
	}
	probe, err := p.Encounter.Stabilize(p.PatientID)
	if err != nil {
		return nil, err
	}
	if rej := probe.Rejection(); rej != nil {
		return rejected(EncounterStateRejection{rej}), nil
	}
	helper, _ := p.Encounter.Find(p.HelperID)
	step, err := p.Encounter.tracker.TakeAction(p.HelperID, availabilityOf(helper.conditions), SimpleActionChoice{SimpleStandardActionHelp})
	if err != nil {
		return nil, err
	}
	applied, ok := step.(TrackerStepApplied)
	if !ok {
		return rejected(TrackerRejection{step.(TrackerStepRejected).Reason}), nil
	}

	attempt, helps, helpEvents := p.Check, p.Ledger.helps, []HelpEvent(nil)
	requires := p.Requires
	if requires == 0 {
		requires = CheckRequirementNone
	}
	adj := (AttackConditionInputs{}).Check(helper.conditions, p.FearSource, requires)
	if _, forced := adj.Forced(); forced {
		if _, rolled := attempt.(d20.RollAttempt); rolled {
			attempt = d20.AutomaticFailure{}
		}
	}
	if a, rolled := attempt.(d20.RollAttempt); rolled {
		a.Circumstances = slices.Concat(a.Circumstances, adj.Circumstances())
		a.Modifiers = a.Modifiers.WithOthers(adj.Modifiers()...)
		if used, ok := helps.ConsumeForAbilityCheck(p.HelperID, AssistedSkill(content.SkillMedicine)).(AbilityCheckHelpUsed); ok {
			a.Circumstances = append(slices.Clone(a.Circumstances), used.Circumstances()...)
			helps, helpEvents = used.Benefits(), used.Events()
		}
		attempt = a
	}
	check, err := u.flow.resolveCheck(attempt, StabilizingCheck{}.DC())
	if err != nil {
		return nil, err
	}

	r := &resolution{enc: p.Encounter}
	if err := r.commitTurn(applied); err != nil {
		return nil, err
	}
	for _, e := range helpEvents {
		r.add(e)
	}
	stabilized := check.Outcome() == d20.Success
	if stabilized {
		tr, err := r.enc.Stabilize(p.PatientID)
		if err != nil {
			return nil, err
		}
		r.mustCommit(tr)
	}
	r.prompts(p.Encounter)
	// No events cap check: the events of this call are bounded by its per-call lists (plan P10), far under 4096.
	if r.exitInvalid() {
		return nil, invalidInput("the encounter is not valid after the call")
	}
	return AttemptToStabilizeCombatantCompleted{ledgerCompleted{resolutionCompleted{r.enc, r.events},
		p.Ledger.withHelps(helps)}, testResolutionOf(check), stabilized}, nil
}
