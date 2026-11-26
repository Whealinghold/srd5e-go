package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
)

// ResolveRegurgitationInput is the raw values of a ResolveRegurgitationCommand. Build it with a keyed
// literal.
type ResolveRegurgitationInput struct {
	// Containments is the creatures held.
	Containments Containments
	// Container is the monster at the end of whose turn the threshold is checked.
	Container CombatantID
	// OccupantConditions holds the conditions of exactly the creatures the container holds, which a failed
	// save releases. It is read only when a save is due.
	OccupantConditions map[CombatantID]ConditionSet
	// Attempt is the container's Constitution saving throw: a d20.RollAttempt carrying the host's modifier,
	// d20.ChooseToFail or d20.AutomaticFailure. It is read only when a save is due.
	Attempt d20.SaveAttempt
}

// ResolveRegurgitationCommand is the input of ResolveRegurgitation.Run. Build it with
// NewResolveRegurgitationCommand; the zero value is invalid.
type ResolveRegurgitationCommand struct{ in ResolveRegurgitationInput }

// NewResolveRegurgitationCommand returns the command. It returns an error matching srderr.ErrInvalidInput
// for a blank container or an attempt that is not a roll, a choice to fail or an automatic failure. It
// returns a LimitExceededError for more than 256 OccupantConditions (plan P10). It copies the map and the attempt and never panics. It returns a LimitExceededError for a d20.RollAttempt of more than 256 circumstances (plan P10).
func NewResolveRegurgitationCommand(in ResolveRegurgitationInput) (ResolveRegurgitationCommand, error) {
	if err := overCap("occupant conditions", len(in.OccupantConditions), maxInputList); err != nil {
		return ResolveRegurgitationCommand{}, err
	}
	if err := attemptOverCap(in.Attempt); err != nil {
		return ResolveRegurgitationCommand{}, err
	}
	a, ok := checkSave(in.Attempt)
	if in.Container.IsZero() || !ok {
		return ResolveRegurgitationCommand{}, invalidInput("regurgitation needs a container and a saving throw attempt")
	}
	in.Attempt = a
	in.OccupantConditions = copyConditions(in.OccupantConditions)
	return ResolveRegurgitationCommand{in}, nil
}

// ResolveRegurgitationResult is the closed set of outcomes of ResolveRegurgitation.Run. Variants:
// ResolveRegurgitationRegurgitated, ResolveRegurgitationHeld, ResolveRegurgitationNotDue,
// ResolveRegurgitationInvalid. Every applied variant carries the set with every tally of the container reset.
type ResolveRegurgitationResult interface{ isResolveRegurgitationResult() }

// ResolveRegurgitationRegurgitated: the save failed and the container released every creature it held.
type ResolveRegurgitationRegurgitated struct {
	containments Containments
	check        ContainmentTest
	releases     []ContainmentRelease
	events       []Event
}

// Containments returns the set without the released creatures.
func (r ResolveRegurgitationRegurgitated) Containments() Containments { return r.containments }

// Check returns the container's saving throw.
func (r ResolveRegurgitationRegurgitated) Check() ContainmentTest { return r.check }

// Releases returns a new slice of the released creatures, in the order they were taken in; each lands
// within the terms' LandingFeet and is Prone unless it is immune to the condition.
func (r ResolveRegurgitationRegurgitated) Releases() []ContainmentRelease {
	return slices.Clone(r.releases)
}

// Events returns a copy of the events, in the order they happened.
func (r ResolveRegurgitationRegurgitated) Events() []Event { return slices.Clone(r.events) }

// ResolveRegurgitationHeld: the container passed its save and keeps what it holds.
type ResolveRegurgitationHeld struct {
	containments Containments
	check        ContainmentTest
}

// Containments returns the set with its tallies reset.
func (r ResolveRegurgitationHeld) Containments() Containments { return r.containments }

// Check returns the container's saving throw.
func (r ResolveRegurgitationHeld) Check() ContainmentTest { return r.check }

// ResolveRegurgitationNotDue: no creature dealt the threshold in one turn, so no die was rolled.
type ResolveRegurgitationNotDue struct{ containments Containments }

// Containments returns the set with its tallies reset.
func (r ResolveRegurgitationNotDue) Containments() Containments { return r.containments }

// ResolveRegurgitationInvalid: the map of conditions does not match the creatures the container holds. No
// die was rolled and no tally reset. Construct it with a keyed literal.
type ResolveRegurgitationInvalid struct{ Problem ContainmentProblem }

func (ResolveRegurgitationRegurgitated) isResolveRegurgitationResult() {}
func (ResolveRegurgitationHeld) isResolveRegurgitationResult()         {}
func (ResolveRegurgitationNotDue) isResolveRegurgitationResult()       {}
func (ResolveRegurgitationInvalid) isResolveRegurgitationResult()      {}

// ResolveRegurgitation is the use case of a swallowing container that took too much damage from inside
// (SRD 5.2.1, "Purple Worm", p. 317; "Tarrasque", p. 332). It composes the pure d20 functions with the dice
// port and never calls d20.Resolver or another use case. Create it with NewResolveRegurgitation; the zero
// value is not usable.
type ResolveRegurgitation struct {
	containmentUse
	flow attackFlow
}

// NewResolveRegurgitation returns the use case rolling through roller. Options: WithTelemetry, WithClock.
// It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewResolveRegurgitation(roller Roller, opts ...Option) (ResolveRegurgitation, error) {
	if roller == nil {
		return ResolveRegurgitation{}, invalidInput("regurgitation without a dice roller")
	}
	return ResolveRegurgitation{newContainmentUse(opts), attackFlow{roller}}, nil
}

// Run is due when a swallowed creature inside the container has dealt its container at least the
// threshold (equal triggers) in this turn; the damage of several creatures is never summed. If none has,
// the result is ResolveRegurgitationNotDue and no die is rolled. When due, the container makes a
// Constitution saving throw against the terms' DC: a failure releases every creature it holds as
// regurgitated (ResolveRegurgitationRegurgitated), a success keeps them (ResolveRegurgitationHeld). Every
// outcome resets the tally of every creature the container holds (FR-049), because a tally never carries
// into another turn: the host calls Run at the end of each turn in which it recorded damage from inside.
//
// It records the shared signals, srd5e.combat.containment.ended for each creature released and
// srd5e.combat.containment.occupants for a result with a set. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero use case or command, and ctx.Err() unwrapped,
// before any signal, for a cancelled ctx. It returns a *srderr.PortError (error type "dice_roller_error")
// with a nil result when the roller fails, panics or answers another expression; the failure is recorded
// once and never logged.
func (u ResolveRegurgitation) Run(ctx context.Context, cmd ResolveRegurgitationCommand) (ResolveRegurgitationResult, error) {
	if u.flow.roller == nil || cmd.in.Attempt == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationResolveRegurgitation, spanResolveRegurgitation}
	res, err := runContainment(ctx, u.containmentUse, op, func(context.Context) (ResolveRegurgitationResult, error) { return u.regurgitate(cmd.in) },
		func(r ResolveRegurgitationResult) description {
			switch v := r.(type) {
			case ResolveRegurgitationRegurgitated:
				return description{Class: resultApplied, Name: valueResultRegurgitated, Events: v.events}
			case ResolveRegurgitationHeld:
				return description{Class: resultApplied, Name: valueResultHeld}
			case ResolveRegurgitationNotDue:
				return description{Class: resultApplied, Name: valueResultNotDue}
			case ResolveRegurgitationInvalid:
				return description{Class: resultInvalid, Name: v.Problem.String()}
			default:
				panic(unreachableError("resolve regurgitation result", r))
			}
		})
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case ResolveRegurgitationRegurgitated:
		u.recordReleases(ctx, v.releases)
		u.recordSet(ctx, op, v.containments)
	case ResolveRegurgitationHeld:
		u.recordSet(ctx, op, v.containments)
	case ResolveRegurgitationNotDue:
		u.recordSet(ctx, op, v.containments)
	}
	return res, nil
}

// regurgitate is the work of Run.
func (u ResolveRegurgitation) regurgitate(in ResolveRegurgitationInput) (ResolveRegurgitationResult, error) {
	held := in.Containments.OccupantsOf(in.Container)
	var rule Regurgitation
	due := false
	for _, r := range held {
		if reg, ok := r.terms.Regurgitation(); ok && !r.phase.IsTrapped() && r.damageFromInside >= reg.threshold {
			rule, due = reg, true
		}
	}
	reset := in.Containments.withoutTallies(in.Container)
	if !due {
		return ResolveRegurgitationNotDue{reset}, nil
	}
	affected := make([]CombatantID, len(held))
	for i, r := range held {
		affected[i] = r.Occupant
	}
	if p := conditionsProblem(in.OccupantConditions, affected); p != 0 {
		return ResolveRegurgitationInvalid{p}, nil
	}
	test, err := u.flow.containmentTest(d20.SavingThrow, in.Attempt, rule.ConstitutionDC())
	if err != nil {
		return nil, err
	}
	if test.passed() {
		return ResolveRegurgitationHeld{reset, test}, nil
	}
	var releases []ContainmentRelease
	var events []Event
	for _, r := range held {
		rel := ContainmentRules{}.ReleaseOnSet(in.OccupantConditions[r.Occupant], r, ContainmentLeaveCauseRegurgitated)
		releases, events = append(releases, rel), append(events, rel.events...)
		reset = reset.without(r.Occupant)
	}
	return ResolveRegurgitationRegurgitated{reset, test, releases, events}, nil
}
