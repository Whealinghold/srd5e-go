package combat

import (
	"context"
	"slices"
)

// AdvanceContainerTurnInput is the raw values of an AdvanceContainerTurnCommand. Build it with a keyed
// literal.
type AdvanceContainerTurnInput struct {
	// Containments is the creatures held.
	Containments Containments
	// Container is the monster whose turn reaches Boundary.
	Container CombatantID
	// Boundary is the start or the end of the container's turn.
	Boundary ContainerTurnBoundary
	// OccupantConditions holds the conditions of exactly the creatures the call disgorges: those of a record
	// that disgorges after its first damage, that has taken damage and that are among Survivors.
	OccupantConditions map[CombatantID]ConditionSet
	// Survivors are the held creatures still alive after the damage of this boundary. At the end boundary a
	// creature that disgorges after its first damage and is not among them is removed from the set.
	Survivors []CombatantID
}

// AdvanceContainerTurnCommand is the input of AdvanceContainerTurn.Run. Build it with
// NewAdvanceContainerTurnCommand; the zero value is invalid.
type AdvanceContainerTurnCommand struct{ in AdvanceContainerTurnInput }

// NewAdvanceContainerTurnCommand returns the command. It returns an error matching srderr.ErrInvalidInput
// for a blank container or an invalid boundary. It returns a LimitExceededError for more than
// 256 OccupantConditions or 256 Survivors (plan P10, before it reads one). It copies the map and the slice and never panics.
func NewAdvanceContainerTurnCommand(in AdvanceContainerTurnInput) (AdvanceContainerTurnCommand, error) {
	if in.Container.IsZero() || !in.Boundary.Valid() {
		return AdvanceContainerTurnCommand{}, invalidInput("advance needs a container and a boundary")
	}
	if err := overCap("occupant conditions", len(in.OccupantConditions), maxInputList); err != nil {
		return AdvanceContainerTurnCommand{}, err
	}
	if err := overCap("survivors", len(in.Survivors), maxInputList); err != nil {
		return AdvanceContainerTurnCommand{}, err
	}
	in.OccupantConditions = copyConditions(in.OccupantConditions)
	in.Survivors = slices.Clone(in.Survivors)
	return AdvanceContainerTurnCommand{in}, nil
}

// AdvanceContainerTurnResult is the closed set of outcomes of AdvanceContainerTurn.Run. Variants:
// AdvanceContainerTurnAdvanced, AdvanceContainerTurnInvalid.
type AdvanceContainerTurnResult interface{ isAdvanceContainerTurnResult() }

// AdvanceContainerTurnAdvanced: the boundary was applied.
type AdvanceContainerTurnAdvanced struct {
	containments Containments
	damage       []ContainmentDamageDue
	releases     []ContainmentRelease
	events       []Event
}

// Containments returns the set after the boundary.
func (r AdvanceContainerTurnAdvanced) Containments() Containments { return r.containments }

// DamageDue returns a new slice of the periodic damage the host rolls and applies, one per creature in
// the order they were taken in; empty for none.
func (r AdvanceContainerTurnAdvanced) DamageDue() []ContainmentDamageDue {
	return slices.Clone(r.damage)
}

// Releases returns a new slice of the creatures that stopped being held, disgorged or removed. A removed
// creature's Conditions is the empty set: the rules touch none.
func (r AdvanceContainerTurnAdvanced) Releases() []ContainmentRelease {
	return slices.Clone(r.releases)
}

// Events returns a copy of the events, in the order they happened.
func (r AdvanceContainerTurnAdvanced) Events() []Event { return slices.Clone(r.events) }

// AdvanceContainerTurnInvalid: the map of conditions does not match the creatures the call disgorges.
// Construct it with a keyed literal.
type AdvanceContainerTurnInvalid struct{ Problem ContainmentProblem }

func (AdvanceContainerTurnAdvanced) isAdvanceContainerTurnResult() {}
func (AdvanceContainerTurnInvalid) isAdvanceContainerTurnResult()  {}

// AdvanceContainerTurn is the use case of a container's turn boundary: it reports the periodic damage due
// (SRD 5.2.1, "Behir", p. 263; "Giant Toad", p. 354) and disgorges. It rolls nothing and has no port. Its
// zero value is silent and works. Safe for concurrent use.
type AdvanceContainerTurn struct{ containmentUse }

// NewAdvanceContainerTurn returns the use case. Options: WithTelemetry, WithClock.
func NewAdvanceContainerTurn(opts ...Option) AdvanceContainerTurn {
	return AdvanceContainerTurn{newContainmentUse(opts)}
}

// Run applies the boundary. At the start boundary it reports the damage due of every record inside whose
// periodic damage is due at the start of the turn; at the end boundary that of the records due at the end,
// except a deferred record, whose deferral is only cleared (FR-078: the Giant Frog's first tick is the end of
// its next turn, p. 351). A record trapped in a dead container takes none. Each due increments its record's
// DamageTicks. At the end boundary a record that disgorges after its first damage and has taken damage is
// released as disgorged when its creature is among the survivors, and removed as occupant removed when it is
// not (p. 351). The host rolls and applies the damage, reporting the creatures that survive it as Survivors.
//
// It records the shared signals, srd5e.combat.containment.ended for each creature that stops being held and
// srd5e.combat.containment.occupants for an advanced result. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero command, and ctx.Err() unwrapped, before any
// signal, for a cancelled ctx.
func (u AdvanceContainerTurn) Run(ctx context.Context, cmd AdvanceContainerTurnCommand) (AdvanceContainerTurnResult, error) {
	if cmd.in.Container.IsZero() {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationAdvanceContainerTurn, spanAdvanceContainerTurn}
	res, err := runContainment(ctx, u.containmentUse, op, func(context.Context) (AdvanceContainerTurnResult, error) { return advanceTurn(cmd.in), nil },
		func(r AdvanceContainerTurnResult) description {
			switch v := r.(type) {
			case AdvanceContainerTurnAdvanced:
				return description{Class: resultApplied, Name: valueResultAdvanced, Events: v.events}
			case AdvanceContainerTurnInvalid:
				return description{Class: resultInvalid, Name: v.Problem.String()}
			default:
				panic(unreachableError("advance container turn result", r))
			}
		})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(AdvanceContainerTurnAdvanced); ok {
		u.recordReleases(ctx, v.releases)
		u.recordSet(ctx, op, v.containments)
	}
	return res, nil
}

// advanceTurn is the work of Run.
func advanceTurn(in AdvanceContainerTurnInput) AdvanceContainerTurnResult {
	next, dues := in.Containments.advance(in.Container, in.Boundary)
	var disgorged, removed []Containment
	if in.Boundary == ContainerTurnBoundaryEnd {
		for _, r := range next.OccupantsOf(in.Container) {
			if !r.terms.disgorgeAfterFirstDamage || r.phase.IsTrapped() || r.damageTicks < 1 {
				continue
			}
			if slices.Contains(in.Survivors, r.Occupant) {
				disgorged = append(disgorged, r)
			} else {
				removed = append(removed, r)
			}
		}
	}
	affected := make([]CombatantID, len(disgorged))
	for i, r := range disgorged {
		affected[i] = r.Occupant
	}
	if p := conditionsProblem(in.OccupantConditions, affected); p != 0 {
		return AdvanceContainerTurnInvalid{p}
	}
	var releases []ContainmentRelease
	var events []Event
	release := func(r Containment, conds ConditionSet, cause ContainmentLeaveCause) {
		rel := ContainmentRules{}.ReleaseOnSet(conds, r, cause)
		releases, events = append(releases, rel), append(events, rel.events...)
		next = next.without(r.Occupant)
	}
	for _, r := range disgorged {
		release(r, in.OccupantConditions[r.Occupant], ContainmentLeaveCauseDisgorged)
	}
	for _, r := range removed {
		release(r, ConditionSet{}, ContainmentLeaveCauseOccupantRemoved)
	}
	return AdvanceContainerTurnAdvanced{next, dues, releases, events}
}
