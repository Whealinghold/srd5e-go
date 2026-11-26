package combat

import (
	"context"
	"slices"
)

// ReleaseTrigger is why the host asks for creatures to be released. Variants: ContainerDied,
// EscapedCorpse, GrappleReview, OccupantGone.
type ReleaseTrigger interface{ isReleaseTrigger() }

// ContainerDied: the container died. Its death rule decides: release the creatures where it was, or trap them
// until each has climbed out (SRD 5.2.1, "Tarrasque", p. 332). Construct it with a keyed literal.
type ContainerDied struct {
	// Container is the monster that died.
	Container CombatantID
}

// EscapedCorpse: a creature trapped in a dead container spends movement to climb out (SRD 5.2.1, "Tarrasque",
// p. 332). Construct it with a keyed literal.
type EscapedCorpse struct {
	// Occupant is the trapped creature.
	Occupant CombatantID
	// MovementRemainingFeet is the movement it has left this turn.
	MovementRemainingFeet int
}

// GrappleReview: the host reviewed the grapples; every Engulfed-grappled creature whose container no longer
// holds it in Grapples is released (SRD 5.2.1, "Shambling Mound", p. 323: "Until the grapple ends"). Construct
// it with a keyed literal.
type GrappleReview struct {
	// Grapples is the grapples in force after the review.
	Grapples Grapples
}

// OccupantGone: the host removed the creature (dead, teleported or gone). Construct it with a keyed literal.
type OccupantGone struct {
	// Occupant is the creature that is gone.
	Occupant CombatantID
}

func (ContainerDied) isReleaseTrigger() {}
func (EscapedCorpse) isReleaseTrigger() {}
func (GrappleReview) isReleaseTrigger() {}
func (OccupantGone) isReleaseTrigger()  {}

// ReleaseOccupantsInput is the raw values of a ReleaseOccupantsCommand. Build it with a keyed literal.
type ReleaseOccupantsInput struct {
	// Containments is the creatures held.
	Containments Containments
	// Grapples is the grapples in force; a dead container's Engulfed-grappled creatures leave their grapples.
	Grapples Grapples
	// OccupantConditions holds the conditions of exactly the creatures whose conditions the call changes.
	OccupantConditions map[CombatantID]ConditionSet
	// Trigger is why creatures are released.
	Trigger ReleaseTrigger
}

// ReleaseOccupantsCommand is the input of ReleaseOccupants.Run. Build it with NewReleaseOccupantsCommand; the
// zero value is invalid.
type ReleaseOccupantsCommand struct{ in ReleaseOccupantsInput }

// NewReleaseOccupantsCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a
// trigger that is not one of the four or that names a blank id. It returns a
// LimitExceededError for more than 256 OccupantConditions (plan P10). It copies the map and never panics.
func NewReleaseOccupantsCommand(in ReleaseOccupantsInput) (ReleaseOccupantsCommand, error) {
	if err := overCap("occupant conditions", len(in.OccupantConditions), maxInputList); err != nil {
		return ReleaseOccupantsCommand{}, err
	}
	ok := false
	switch t := in.Trigger.(type) {
	case ContainerDied:
		ok = !t.Container.IsZero()
	case EscapedCorpse:
		ok = !t.Occupant.IsZero()
	case GrappleReview:
		ok = true
	case OccupantGone:
		ok = !t.Occupant.IsZero()
	}
	if !ok {
		return ReleaseOccupantsCommand{}, invalidInput("release needs one of the four triggers with its ids")
	}
	in.OccupantConditions = copyConditions(in.OccupantConditions)
	return ReleaseOccupantsCommand{in}, nil
}

// ReleaseOccupantsResult is the closed set of outcomes of ReleaseOccupants.Run. Variants:
// ReleaseOccupantsReleased, ReleaseOccupantsRejected, ReleaseOccupantsInvalid.
type ReleaseOccupantsResult interface{ isReleaseOccupantsResult() }

// ReleaseOccupantsReleased: the trigger took effect. It may have released nobody, as for a container that
// held nobody.
type ReleaseOccupantsReleased struct {
	containments  Containments
	grapples      Grapples
	releases      []ContainmentRelease
	trapped       []OccupantTrapped
	movementSpent int
	events        []Event
}

// Containments returns the set after the trigger.
func (r ReleaseOccupantsReleased) Containments() Containments { return r.containments }

// Grapples returns the grapples after the trigger.
func (r ReleaseOccupantsReleased) Grapples() Grapples { return r.grapples }

// Releases returns a new slice of the creatures that stopped being held. A creature that is gone has the
// empty set as its Conditions: the rules touch none.
func (r ReleaseOccupantsReleased) Releases() []ContainmentRelease { return slices.Clone(r.releases) }

// Trapped returns a new slice of the creatures a dead container keeps until they climb out.
func (r ReleaseOccupantsReleased) Trapped() []OccupantTrapped { return slices.Clone(r.trapped) }

// MovementSpentFeet returns the movement an EscapedCorpse spent, 0 for any other trigger.
func (r ReleaseOccupantsReleased) MovementSpentFeet() int { return r.movementSpent }

// Events returns a copy of the events, in the order they happened.
func (r ReleaseOccupantsReleased) Events() []Event { return slices.Clone(r.events) }

// ReleaseOccupantsRejected: the rules refuse the trigger in this state; nothing changed. Construct it with a
// keyed literal.
type ReleaseOccupantsRejected struct{ Rejection ContainmentRejection }

// ReleaseOccupantsInvalid: the map of conditions does not match the creatures the call changes. Construct it
// with a keyed literal.
type ReleaseOccupantsInvalid struct{ Problem ContainmentProblem }

func (ReleaseOccupantsReleased) isReleaseOccupantsResult() {}
func (ReleaseOccupantsRejected) isReleaseOccupantsResult() {}
func (ReleaseOccupantsInvalid) isReleaseOccupantsResult()  {}

// ReleaseOccupants is the use case of creatures that stop being held for a reason other than their own
// escape (SRD 5.2.1, "Tarrasque", p. 332; "Shambling Mound", p. 323). It rolls nothing and has no port. Its
// zero value is silent and works. Safe for concurrent use.
type ReleaseOccupants struct{ containmentUse }

// NewReleaseOccupants returns the use case. Options: WithTelemetry, WithClock.
func NewReleaseOccupants(opts ...Option) ReleaseOccupants {
	return ReleaseOccupants{newContainmentUse(opts)}
}

// Run applies the trigger. ContainerDied applies the container's death rule to each creature inside it:
// ReleaseOnDeath releases it (ending the grapple of an Engulfed-grappled creature first), EscapeFromCorpse
// traps it and removes the Restrained the containment added. EscapedCorpse releases a trapped creature (Prone)
// when its remaining movement is at least the distance, and is refused as not contained, not trapped or
// insufficient movement. GrappleReview releases every Engulfed-grappled creature its container no longer
// holds. OccupantGone drops the record and touches no condition; a creature that is not held is refused as
// not contained.
//
// It records the shared signals, srd5e.combat.containment.ended for each creature released and
// srd5e.combat.containment.occupants for a released result. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero command, and ctx.Err() unwrapped, before any signal,
// for a cancelled ctx.
func (u ReleaseOccupants) Run(ctx context.Context, cmd ReleaseOccupantsCommand) (ReleaseOccupantsResult, error) {
	if cmd.in.Trigger == nil {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationReleaseOccupants, spanReleaseOccupants}
	res, err := runContainment(ctx, u.containmentUse, op, func(context.Context) (ReleaseOccupantsResult, error) { return releaseOccupants(cmd.in), nil },
		func(r ReleaseOccupantsResult) description {
			switch v := r.(type) {
			case ReleaseOccupantsReleased:
				return description{Class: resultApplied, Name: valueResultReleased, Events: v.events}
			case ReleaseOccupantsRejected:
				return description{Class: resultRefused, Name: v.Rejection.String()}
			case ReleaseOccupantsInvalid:
				return description{Class: resultInvalid, Name: v.Problem.String()}
			default:
				panic(unreachableError("release occupants result", r))
			}
		})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(ReleaseOccupantsReleased); ok {
		u.recordReleases(ctx, v.releases)
		u.recordSet(ctx, op, v.containments)
	}
	return res, nil
}

// releasePlan is the creatures a trigger changes and how.
type releasePlan struct {
	records  []Containment
	cause    ContainmentLeaveCause
	trap     bool
	spent    int
	rejected ContainmentRejection
}

// releaseOccupants is the work of Run.
func releaseOccupants(in ReleaseOccupantsInput) ReleaseOccupantsResult {
	plan := planRelease(in)
	if plan.rejected != 0 {
		return ReleaseOccupantsRejected{plan.rejected}
	}
	affected := []CombatantID{}
	if plan.cause != ContainmentLeaveCauseOccupantRemoved {
		for _, r := range plan.records {
			affected = append(affected, r.Occupant)
		}
	}
	if p := conditionsProblem(in.OccupantConditions, affected); p != 0 {
		return ReleaseOccupantsInvalid{p}
	}
	out := ReleaseOccupantsReleased{containments: in.Containments, grapples: in.Grapples, movementSpent: plan.spent}
	for _, r := range plan.records {
		conds := in.OccupantConditions[r.Occupant]
		if plan.trap {
			feet := r.terms.deathRule.(EscapeFromCorpse).feet
			t, trapped := trap(conds, r, feet)
			out.trapped, out.events = append(out.trapped, t), append(out.events, t.events...)
			out.containments = out.containments.replacing(trapped)
			continue
		}
		var rel ContainmentRelease
		out.grapples, rel = releaseHeld(out.grapples, conds, r, plan.cause)
		out.releases, out.events = append(out.releases, rel), append(out.events, rel.events...)
		out.containments = out.containments.without(r.Occupant)
	}
	return out
}

// planRelease works out which records a trigger changes, the cause, and any refusal.
func planRelease(in ReleaseOccupantsInput) releasePlan {
	switch t := in.Trigger.(type) {
	case ContainerDied:
		plan := releasePlan{cause: ContainmentLeaveCauseContainerDied}
		for _, r := range in.Containments.OccupantsOf(t.Container) {
			if r.phase.IsTrapped() {
				continue
			}
			plan.records = append(plan.records, r)
			if _, corpse := r.terms.deathRule.(EscapeFromCorpse); corpse {
				plan.trap = true
			}
		}
		return plan
	case EscapedCorpse:
		held, ok := in.Containments.ContainerOf(t.Occupant).(InContainer)
		switch {
		case !ok:
			return releasePlan{rejected: ContainmentRejectionNotContained}
		case !held.record.phase.IsTrapped():
			return releasePlan{rejected: ContainmentRejectionNotTrapped}
		case t.MovementRemainingFeet < held.record.phase.trappedFeet:
			return releasePlan{rejected: ContainmentRejectionInsufficientMovement}
		}
		return releasePlan{records: []Containment{held.record}, cause: ContainmentLeaveCauseEscapedCorpse, spent: held.record.phase.trappedFeet}
	case GrappleReview:
		plan := releasePlan{cause: ContainmentLeaveCauseGrappleEnded}
		for _, r := range in.Containments.list {
			if g, held := t.Grapples.GrappleOf(r.Occupant).(InGrapple); r.terms.mode == ContainmentModeEngulfedGrappled &&
				!r.phase.IsTrapped() && (!held || g.grapple.Grappler != r.Container) {
				plan.records = append(plan.records, r)
			}
		}
		return plan
	case OccupantGone:
		held, ok := in.Containments.ContainerOf(t.Occupant).(InContainer)
		if !ok {
			return releasePlan{rejected: ContainmentRejectionNotContained}
		}
		return releasePlan{records: []Containment{held.record}, cause: ContainmentLeaveCauseOccupantRemoved}
	default:
		panic(unreachableError("release trigger", in.Trigger))
	}
}
