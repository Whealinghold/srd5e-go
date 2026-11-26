package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ContainmentRules is the stateless domain service of the containment rules (SRD 5.2.1, "Gelatinous
// Cube", p. 287; "Purple Worm", p. 317). Its methods are pure, synchronous and silent, and change no
// Containments value. The zero value is ready to use; a consumer that wants to replace one declares its
// own interface. Safe for concurrent use.
type ContainmentRules struct{}

// CoverBetween returns ContainmentCoverTotal when the target is held by a container whose terms give Total
// Cover and the source is outside it: neither the target itself, nor its container, nor another creature
// held by that container (SRD 5.2.1, "Purple Worm", p. 317: cover against attacks and other effects
// outside). Otherwise it returns ContainmentCoverNone. A host feeds Total to the attack rules, where no
// attack is possible.
func (ContainmentRules) CoverBetween(s Containments, source, target CombatantID) ContainmentCover {
	in, ok := s.ContainerOf(target).(InContainer)
	if !ok || !in.record.terms.totalCover || source == target || source == in.record.Container {
		return ContainmentCoverNone
	}
	if peer, held := s.ContainerOf(source).(InContainer); held && peer.record.Container == in.record.Container {
		return ContainmentCoverNone
	}
	return ContainmentCoverTotal
}

// RestrictionsOn returns what holding the occupant restricts, read from its container's terms, and the
// restrictions of a creature that is not held (no cover, no suffocation, nothing forbidden) otherwise. The
// host passes NoVerbalSpells to the casting rules as "cannot speak" and NoTeleport to any teleport.
func (ContainmentRules) RestrictionsOn(s Containments, occupant CombatantID) ContainedRestrictions {
	in, ok := s.ContainerOf(occupant).(InContainer)
	if !ok {
		return ContainedRestrictions{Cover: ContainmentCoverNone, Suffocation: SuffocationRuleNone}
	}
	t := in.record.terms
	cover := ContainmentCoverNone
	if t.totalCover {
		cover = ContainmentCoverTotal
	}
	return ContainedRestrictions{Cover: cover, NoVerbalSpells: t.noVerbalSpells, NoTeleport: t.noTeleport, Suffocation: t.suffocation}
}

// MovesWith returns a new slice of the creatures that move with the container, in the order they were taken
// in (SRD 5.2.1, "Gelatinous Cube", p. 287; "Shambling Mound", p. 323).
func (ContainmentRules) MovesWith(s Containments, container CombatantID) []CombatantID {
	out := []CombatantID{}
	for _, r := range s.list {
		if r.Container == container {
			out = append(out, r.Occupant)
		}
	}
	return out
}

// CarriedFree returns a new slice of the creatures the container carries at no cost to its movement: all
// of those it holds, in the order they were taken in (SRD 5.2.1, "Shambling Mound", p. 323). The host
// leaves them out of Grapples.ExtraMovementCostPerFoot.
func (r ContainmentRules) CarriedFree(s Containments, container CombatantID) []CombatantID {
	return r.MovesWith(s, container)
}

// IsActionLocked says whether the container cannot take the named action: it holds at least one creature
// and its terms' LockedAction is the action (SRD 5.2.1, "Giant Frog", p. 351; "Giant Toad", p. 354).
func (ContainmentRules) IsActionLocked(s Containments, container CombatantID, actionName string) bool {
	for _, r := range s.list {
		if r.Container == container {
			return r.terms.lockedAction != "" && r.terms.lockedAction == actionName
		}
	}
	return false
}

// ContainmentRelease is the effect of releasing one creature, as ContainmentRules.ReleaseOnSet works it
// out. The zero value is invalid. Immutable and safe for concurrent use.
type ContainmentRelease struct {
	// Occupant is the creature that was released.
	Occupant         CombatantID
	cause            ContainmentLeaveCause
	conditions       ConditionSet
	events           []Event
	placement        ExitPlacement
	restoreBreathing bool
	prone            ConditionImposition
}

// IsZero reports whether r is the zero value, which ReleaseOnSet returns for an invalid record or cause.
func (r ContainmentRelease) IsZero() bool { return r.Occupant.IsZero() }

// Cause returns how the creature stopped being held.
func (r ContainmentRelease) Cause() ContainmentLeaveCause { return r.cause }

// Conditions returns the occupant's conditions after the release.
func (r ContainmentRelease) Conditions() ConditionSet { return r.conditions }

// Events returns a copy of the events: the ConditionRemoved of each condition the containment added, the
// OccupantReleased, then the ConditionApplied of Prone when the cause leaves the creature Prone.
func (r ContainmentRelease) Events() []Event { return slices.Clone(r.events) }

// Placement says where the host places the creature, or is nil for ContainmentLeaveCauseOccupantRemoved.
func (r ContainmentRelease) Placement() ExitPlacement { return r.placement }

// RestoreBreathing says the creature was suffocating, so the host restores its breathing.
func (r ContainmentRelease) RestoreBreathing() bool { return r.restoreBreathing }

// Prone returns what applying Prone did and whether it was applied at all.
func (r ContainmentRelease) Prone() (ConditionImposition, bool) { return r.prone, r.prone != 0 }

// ReleaseOnSet works out the release of the record's occupant for the cause on the occupant's conditions
// (SRD 5.2.1, "Condition", p. 179; "Suffocation [Hazard]", p. 189). It removes exactly the conditions the
// containment added, not a kind another source holds; emits one OccupantReleased; applies Prone for the
// causes regurgitated, disgorged and escaped corpse (ConditionImmune and an existing Prone are reported);
// reports that breathing is restored when the terms had suffocation; and places the creature
// (ContainmentLeaveCauseOccupantRemoved touches no condition and has no placement). It does not end the
// grapple of an Engulfed-grappled creature. It returns the zero ContainmentRelease for an invalid record or
// cause. The input set is unchanged.
func (ContainmentRules) ReleaseOnSet(conditions ConditionSet, r Containment, cause ContainmentLeaveCause) ContainmentRelease {
	if !r.valid() || !cause.Valid() {
		return ContainmentRelease{}
	}
	rel := ContainmentRelease{Occupant: r.Occupant, cause: cause, conditions: conditions,
		restoreBreathing: r.terms.suffocation != SuffocationRuleNone}
	if cause != ContainmentLeaveCauseOccupantRemoved {
		rel.conditions, rel.events = removeKinds(conditions, r.Occupant, r.added)
		rel.placement = NearestUnoccupiedSpace{}
	}
	if reg, ok := r.terms.Regurgitation(); ok && cause == ContainmentLeaveCauseRegurgitated {
		rel.placement = LandingWithin{reg.landing}
	}
	rel.events = append(rel.events, OccupantReleased{containmentEventBase{r.Occupant, r.Container}, cause})
	if cause.leavesProne() {
		var events []Event
		rel.conditions, events, rel.prone = impose(rel.conditions, r.Occupant, Prone{})
		rel.events = append(rel.events, events...)
	}
	return rel
}

// removeKinds removes the explicit conditions in mask from the set of target, returning the new set and
// the ConditionRemoved events of those that were held.
func removeKinds(set ConditionSet, target CombatantID, mask kindMask) (ConditionSet, []Event) {
	var events []Event
	for _, k := range kindsOf(mask) {
		res := set.Remove(target, k)
		set = res.Set()
		if cleared, ok := res.(ConditionCleared); ok {
			events = append(events, asConditionEvents(cleared.Events())...)
		}
	}
	return set, events
}

// impose applies c to the set of target and reports what that did.
func impose(set ConditionSet, target CombatantID, c Condition) (ConditionSet, []Event, ConditionImposition) {
	res := set.Apply(target, c)
	switch v := res.(type) {
	case ConditionAdded:
		return v.Set(), asConditionEvents(v.Events()), ConditionImpositionAdded
	case ConditionImmune:
		return v.Set(), nil, ConditionImpositionImmune
	default:
		return res.Set(), nil, ConditionImpositionAlreadyHeld
	}
}

// endHold ends the grapple that holds an Engulfed-grappled occupant: through Grapples.Escape and the cause
// Escaped when escaped is true, otherwise through Grapples.Release by the container and the cause
// Released (SRD 5.2.1, "Grappling", p. 182). The ending goes to GrappleRules.EndOnSet on the occupant's
// conditions, so ConditionRemoved then GrappleEnded come first. A grapple already gone changes nothing.
func endHold(g Grapples, c ConditionSet, container, occupant CombatantID, escaped bool) (Grapples, ConditionSet, []Event) {
	var t GrappleTransition
	cause := GrappleEndCauseReleased
	if escaped {
		t, cause = g.Escape(occupant), GrappleEndCauseEscaped
	} else {
		t = g.Release(container, occupant)
	}
	applied, ok := t.(GrappleTransitionApplied)
	if !ok {
		return g, c, nil
	}
	if e, ok := (GrappleRules{}).EndOnSet(c, occupant, cause).(GrappleEndOnSetApplied); ok {
		return applied.grapples, e.conditions, e.events
	}
	return applied.grapples, c, nil
}

// OccupantTrapped is the effect of a container's death on a creature that must climb out of the corpse. The
// zero value is invalid. Immutable and safe for concurrent use.
type OccupantTrapped struct {
	// Occupant is the trapped creature.
	Occupant   CombatantID
	feet       int
	conditions ConditionSet
	events     []Event
}

// Feet returns the movement the creature must spend to climb out.
func (t OccupantTrapped) Feet() int { return t.feet }

// Conditions returns the occupant's conditions: the containment's Restrained is gone, every other imposed
// condition stays (SRD 5.2.1, "Purple Worm", p. 317).
func (t OccupantTrapped) Conditions() ConditionSet { return t.conditions }

// Events returns a copy of the events: a ConditionRemoved for the Restrained the containment added, then
// ContainerTrapped.
func (t OccupantTrapped) Events() []Event { return slices.Clone(t.events) }

// trap moves the record to phase trapped for feet and works out the occupant's conditions: only the
// Restrained the containment itself added is removed.
func trap(conditions ConditionSet, r Containment, feet int) (OccupantTrapped, Containment) {
	set, events := removeKinds(conditions, r.Occupant, r.added&bit(content.ConditionRestrained))
	events = append(events, ContainerTrapped{containmentEventBase{r.Occupant, r.Container}, feet})
	r.added &^= bit(content.ConditionRestrained)
	r.phase = ContainmentPhase{feet}
	return OccupantTrapped{r.Occupant, feet, set, events}, r
}

// releaseHeld releases the record's occupant for the cause. An Engulfed-grappled occupant leaves with its
// grapple: for the causes escaped and pulled out the grapple ends through Grapples.Escape, for container died
// through Grapples.Release by the container, and its ConditionRemoved and GrappleEnded come first in the
// release's events. For every other cause the grapple is not touched.
func releaseHeld(g Grapples, conditions ConditionSet, r Containment, cause ContainmentLeaveCause) (Grapples, ContainmentRelease) {
	var ended []Event
	if r.terms.mode == ContainmentModeEngulfedGrappled &&
		(cause == ContainmentLeaveCauseEscaped || cause == ContainmentLeaveCausePulledOut || cause == ContainmentLeaveCauseContainerDied) {
		escaped := cause != ContainmentLeaveCauseContainerDied
		g, conditions, ended = endHold(g, conditions, r.Container, r.Occupant, escaped)
	}
	rel := ContainmentRules{}.ReleaseOnSet(conditions, r, cause)
	rel.events = append(ended, rel.events...)
	return g, rel
}
