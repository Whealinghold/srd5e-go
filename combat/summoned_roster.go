package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// linkPresence says where a linked creature is.
type linkPresence int

const (
	// presenceInEncounter: the creature is a combatant of the encounter.
	presenceInEncounter linkPresence = iota + 1
	// presencePocket: the familiar is in its pocket dimension.
	presencePocket
)

// FamiliarState is what the roster records about a familiar beyond its link: its form, creature type and origin, and whether the
// summoner's sight link is active (FR-043 to FR-055). The zero value is no familiar. Comparable and immutable.
type FamiliarState struct {
	form   character.FamiliarForm
	typ    content.CreatureType
	origin FamiliarOrigin
	sight  bool
}

// Form returns the form of the familiar.
func (s FamiliarState) Form() character.FamiliarForm { return s.form }

// CreatureType returns the creature type of the familiar: Celestial, Fey or Fiend for a normal form (Fey for Wild Companion), the
// printed type of a special form.
func (s FamiliarState) CreatureType() content.CreatureType { return s.typ }

// Origin returns what gave the familiar, that of the latest cast (FR-044).
func (s FamiliarState) Origin() FamiliarOrigin { return s.origin }

// SightLinkActive reports whether the summoner sees through the familiar's eyes until the start of the summoner's next turn.
func (s FamiliarState) SightLinkActive() bool { return s.sight }

// SummonedLink is the link of one summoned creature to its summoner: the spell, the cast level, where the creature is and the
// state each kind keeps (summoned-and-companion-creatures.md section 5). Only the roster builds one. The creature's CombatantRef
// is MonsterCombatant with its monster id (FR-008). Not comparable with ==; use Equal. The zero value is invalid.
type SummonedLink struct {
	creature, summoner CombatantID
	monster            content.MonsterID
	kind               SummonKind
	level              int
	side               Side
	presence           linkPresence
	window             ControlWindow
	undead             UndeadForm
	order              SummonedOrder
	retained           RosterPosition
	familiar           FamiliarState
	friends            []CombatantID
}

// Combatants returns a new slice, the creature first and its summoner after it.
func (l SummonedLink) Combatants() []CombatantID { return []CombatantID{l.creature, l.summoner} }

// Creature returns the id and kind of the creature: MonsterCombatant with its monster id.
func (l SummonedLink) Creature() CombatantRef {
	return CombatantRef{ID: l.creature, Kind: MonsterCombatant{ID: l.monster}}
}

// SummonKind returns the kind of the creature.
func (l SummonedLink) SummonKind() SummonKind { return l.kind }

// Spell returns the spell that made the creature.
func (l SummonedLink) Spell() content.SpellID { return l.kind.Spell() }

// CastLevel returns the level the spell was cast at.
func (l SummonedLink) CastLevel() int { return l.level }

// Side returns the side the creature fights on, its summoner's.
func (l SummonedLink) Side() Side { return l.side }

// InPocketDimension reports whether the familiar is in its pocket dimension and so not in the encounter.
func (l SummonedLink) InPocketDimension() bool { return l.presence == presencePocket }

// Placement returns whether the creature shares its summoner's Initiative count or has its own.
func (l SummonedLink) Placement() PlacementRule { return l.kind.Placement() }

// ControlWindow returns the control window of an undead creature and whether the link has one.
func (l SummonedLink) ControlWindow() (ControlWindow, bool) { return l.window, !l.window.IsZero() }

// UndeadForm returns the form of a creature made by Create Undead and whether the link is one.
func (l SummonedLink) UndeadForm() (UndeadForm, bool) { return l.undead, l.undead != 0 }

// Order returns the order the creature carries, nil when it has none.
func (l SummonedLink) Order() SummonedOrder { return l.order }

// RetainedPosition returns the RosterAtInitiative a familiar had when it joined, which a recall uses, and whether there is one.
func (l SummonedLink) RetainedPosition() (RosterPosition, bool) { return l.retained, l.retained != nil }

// Familiar returns the state of a familiar and whether the link is one.
func (l SummonedLink) Familiar() (FamiliarState, bool) { return l.familiar, l.kind == SummonFamiliar }

// DesignatedFriends returns a copy of the creatures a simulacrum is friendly to besides its summoner (FR-064).
func (l SummonedLink) DesignatedFriends() []CombatantID { return slices.Clone(l.friends) }

// FriendlyTo reports whether a simulacrum is friendly to the creature: its summoner and the creatures the summoner designated
// (SRD 5.2.1, "Simulacrum", p. 163; FR-064). It is false for the creature itself and for any other kind of link.
func (l SummonedLink) FriendlyTo(creature CombatantID) bool {
	return l.kind == SummonSimulacrum && creature != l.creature && (creature == l.summoner || slices.Contains(l.friends, creature))
}

// Equal reports whether l and o are the same link by value.
func (l SummonedLink) Equal(o SummonedLink) bool {
	return l.creature == o.creature && l.summoner == o.summoner && l.monster == o.monster && l.kind == o.kind && l.level == o.level &&
		l.side == o.side && l.presence == o.presence && l.window == o.window && l.undead == o.undead && l.order == o.order &&
		l.retained == o.retained && l.familiar == o.familiar && slices.Equal(l.friends, o.friends)
}

// summonedGone remembers a steed or familiar that went, so that a later cast can bring the same steed back (FR-056) and the
// summoner is marked FamiliarGone (FR-049).
type summonedGone struct {
	summoner, creature CombatantID
	kind               SummonKind
}

// SummonedRoster is the immutable record of the creatures summoned during an encounter and what ties them to their summoners,
// which the host stores beside the Encounter as it does a grapple record (summoned-and-companion-creatures.md section 1). Every
// transition returns a new roster and leaves its input unchanged. The zero value is the empty roster. At most one familiar, one
// steed and one simulacrum per summoner (FR-044, FR-056, FR-063) and at most 32 creatures per summoner (plan P10). Slices are
// copied on construction and on read. Not comparable with ==; use Equal. Not safe for concurrent mutation; it is never mutated.
type SummonedRoster struct {
	links  []SummonedLink
	gone   []summonedGone
	grants []SharedResistanceGrant
}

// Len returns the number of linked creatures, those in a pocket dimension included.
func (r SummonedRoster) Len() int { return len(r.links) }

// Links returns a copy of the links in the order the creatures were created.
func (r SummonedRoster) Links() []SummonedLink { return slices.Clone(r.links) }

// Link returns the link of the creature and whether the roster holds it.
func (r SummonedRoster) Link(creature CombatantID) (SummonedLink, bool) {
	if i := r.indexOf(creature); i >= 0 {
		return r.links[i], true
	}
	return SummonedLink{}, false
}

// LinksOf returns a copy of the links of the creatures the summoner made, in the order they were created.
func (r SummonedRoster) LinksOf(summoner CombatantID) []SummonedLink {
	var out []SummonedLink
	for _, l := range r.links {
		if l.summoner == summoner {
			out = append(out, l)
		}
	}
	return out
}

// SharedResistances returns a copy of the Resistances summoners share through a Draconic Spirit while its spell lasts (FR-027).
func (r SummonedRoster) SharedResistances() []SharedResistanceGrant { return slices.Clone(r.grants) }

// FamiliarGone reports whether the summoner's familiar went and has not been replaced (FR-049).
func (r SummonedRoster) FamiliarGone(summoner CombatantID) bool {
	return slices.ContainsFunc(r.gone, func(g summonedGone) bool { return g.summoner == summoner && g.kind == SummonFamiliar })
}

// Equal reports whether r and o hold the same links, remembered departures and grants.
func (r SummonedRoster) Equal(o SummonedRoster) bool {
	return len(r.links) == len(o.links) && slices.EqualFunc(r.links, o.links, SummonedLink.Equal) &&
		slices.Equal(r.gone, o.gone) && slices.Equal(r.grants, o.grants)
}

// indexOf returns the index of the link of the creature, -1 when there is none.
func (r SummonedRoster) indexOf(creature CombatantID) int {
	return slices.IndexFunc(r.links, func(l SummonedLink) bool { return l.creature == creature })
}

// familiarOf returns the summoner's familiar link, present or in the pocket dimension.
func (r SummonedRoster) familiarOf(summoner CombatantID) (SummonedLink, bool) {
	i := slices.IndexFunc(r.links, func(l SummonedLink) bool { return l.summoner == summoner && l.kind == SummonFamiliar })
	if i < 0 {
		return SummonedLink{}, false
	}
	return r.links[i], true
}

// countOf returns how many creatures the summoner has.
func (r SummonedRoster) countOf(summoner CombatantID) int {
	n := 0
	for _, l := range r.links {
		if l.summoner == summoner {
			n++
		}
	}
	return n
}

// createsCycle reports whether linking creature to summoner would close a cycle: summoner is creature, or is a creature of
// creature, directly or through other summoners (plan P10). It walks the chain of summoners iteratively with a visited set, so
// a chain of any length, even a corrupt one, ends.
func (r SummonedRoster) createsCycle(creature, summoner CombatantID) bool {
	visited := map[CombatantID]struct{}{}
	for cur := summoner; ; {
		if cur == creature {
			return true
		}
		if _, seen := visited[cur]; seen {
			return false
		}
		visited[cur] = struct{}{}
		i := r.indexOf(cur)
		if i < 0 {
			return false
		}
		cur = r.links[i].summoner
	}
}

// summonedDraft is a copy of a roster being changed by one transition, with the events and roster actions it produces.
type summonedDraft struct {
	links   []SummonedLink
	gone    []summonedGone
	grants  []SharedResistanceGrant
	events  []SummonEvent
	actions []SummonedRosterAction
	active  CombatantID
}

// draft returns a copy of r to change; active is the combatant the host's tracker has active.
func (r SummonedRoster) draft(active CombatantID) *summonedDraft {
	return &summonedDraft{links: slices.Clone(r.links), gone: slices.Clone(r.gone), grants: slices.Clone(r.grants), active: active}
}

// roster returns the changed roster.
func (d *summonedDraft) roster() SummonedRoster {
	return SummonedRoster{links: slices.Clone(d.links), gone: slices.Clone(d.gone), grants: slices.Clone(d.grants)}
}

// indexOf returns the index of the link of the creature, -1 when there is none.
func (d *summonedDraft) indexOf(creature CombatantID) int {
	return slices.IndexFunc(d.links, func(l SummonedLink) bool { return l.creature == creature })
}

// remember records a departed steed or familiar, replacing the earlier record of the summoner's kind.
func (d *summonedDraft) remember(l SummonedLink) {
	d.forget(l.summoner, l.kind)
	d.gone = append(d.gone, summonedGone{l.summoner, l.creature, l.kind})
}

// forget removes the record of the summoner's departed creature of the kind.
func (d *summonedDraft) forget(summoner CombatantID, kind SummonKind) {
	d.gone = slices.DeleteFunc(d.gone, func(g summonedGone) bool { return g.summoner == summoner && g.kind == kind })
}

// departure says how a creature leaves: the damage that carried over to a reverting object, and whether an undead creature stays
// as a body.
type departure struct {
	overflow int
	body     bool
}

// depart unlinks the creature and records SummonedCreatureDeparted and, for a creature in the encounter that does not stay as a
// body, the SummonedRosterLeave of FR-006 and FR-077: pending, with AfterTurn, when the creature is the active combatant. A
// familiar or steed that goes is remembered.
func (d *summonedDraft) depart(id CombatantID, cause SummonEndCause, how departure) {
	i := d.indexOf(id)
	l := d.links[i]
	d.links = slices.Delete(d.links, i, i+1)
	inEncounter := l.presence == presenceInEncounter && !how.body
	pending := inEncounter && l.creature == d.active
	if l.kind == SummonFamiliar || l.kind == SummonOtherworldlySteed {
		d.remember(l)
	}
	d.events = append(d.events, SummonedCreatureDeparted{
		summonEventBase: pair(l.creature, l.summoner), kind: l.kind, cause: cause, pendingRemoval: pending,
		dropsItems: l.kind.dropsItems() && !how.body, revertsToObject: l.kind == SummonAnimatedObject,
		revertsSnow: l.kind == SummonSimulacrum, leavesBody: how.body, overflow: how.overflow,
	})
	if inEncounter {
		d.actions = append(d.actions, SummonedRosterLeave{Creature: l.creature, Cause: cause.removal(), AfterTurn: pending})
	}
}

// applied returns the changed roster with the events and roster actions as an Applied transition.
func (d *summonedDraft) applied() SummonedApplied {
	return SummonedApplied{roster: d.roster(), events: slices.Clone(d.events), actions: slices.Clone(d.actions)}
}

// SummonedTransition is the outcome of one transition of a SummonedRoster. Variants: SummonedApplied, SummonedRejected,
// SummonedUnchanged.
type SummonedTransition interface{ isSummonedTransition() }

// SummonedApplied: the transition changed the roster. It holds the new roster, the events in the order they happened and the
// roster instructions for the host in creation order (FR-077), and for a summons the specs of the creatures to build. Only the
// roster builds one.
type SummonedApplied struct {
	roster      SummonedRoster
	events      []SummonEvent
	actions     []SummonedRosterAction
	creatures   []SummonedCreatureSpec
	reaction    FeatureReaction
	hasReaction bool
	lifeBond    int
}

// Roster returns the new roster.
func (a SummonedApplied) Roster() SummonedRoster { return a.roster }

// Events returns a copy of the events, in the order they happened.
func (a SummonedApplied) Events() []SummonEvent { return slices.Clone(a.events) }

// RosterActions returns a copy of the instructions for the encounter's roster, in creation order.
func (a SummonedApplied) RosterActions() []SummonedRosterAction { return slices.Clone(a.actions) }

// Creatures returns a copy of the specs of the creatures a summons made or whose form it changed.
func (a SummonedApplied) Creatures() []SummonedCreatureSpec { return slices.Clone(a.creatures) }

// Reaction returns the Reaction a delivered touch spent, the familiar's, and whether there is one (FR-047).
func (a SummonedApplied) Reaction() (FeatureReaction, bool) { return a.reaction, a.hasReaction }

// LifeBondHitPoints returns the Hit Points a steed regains through Life Bond, 0 when the change was not a Life Bond (FR-057).
func (a SummonedApplied) LifeBondHitPoints() int { return a.lifeBond }

// SummonedRejected: the rules refused the transition. Rejection says why; the roster is unchanged and there are no events.
type SummonedRejected struct{ Rejection SummonRejection }

// SummonedUnchanged: the transition had nothing to change, for a creature the roster does not hold, one that already departed or
// a spell that ends nothing (FR-026). It holds no state and no events.
type SummonedUnchanged struct{}

func (SummonedApplied) isSummonedTransition()   {}
func (SummonedRejected) isSummonedTransition()  {}
func (SummonedUnchanged) isSummonedTransition() {}

// rejected returns the Rejected transition for r.
func rejected(r SummonRejection) SummonedTransition { return SummonedRejected{Rejection: r} }

// SummonedCreatureSpec is what the host builds a summoned creature from: its stat block, side, Initiative placement and, for a
// steed, the MountFacts that let its rider control it (FR-018). Only SummonedRoster.Summon builds one. Its CombatantRef is
// MonsterCombatant with the monster id of the stat block (FR-008). Not comparable with ==; use the accessors.
type SummonedCreatureSpec struct {
	creature  CombatantID
	kind      SummonKind
	side      Side
	block     SummonedStatBlock
	mount     MountFacts
	monster   content.Monster
	hasMonst  bool
	placement PlacementRule
}

// Creature returns the id and kind of the creature.
func (s SummonedCreatureSpec) Creature() CombatantRef {
	return CombatantRef{ID: s.creature, Kind: MonsterCombatant{ID: s.block.monster}}
}

// SummonKind returns the kind of the creature.
func (s SummonedCreatureSpec) SummonKind() SummonKind { return s.kind }

// Side returns the side of the creature, its summoner's.
func (s SummonedCreatureSpec) Side() Side { return s.side }

// StatBlock returns the stat block with the numbers that depend on the cast.
func (s SummonedCreatureSpec) StatBlock() SummonedStatBlock { return s.block }

// MountFacts returns the facts of a steed as a mount and whether the spec has them.
func (s SummonedCreatureSpec) MountFacts() (MountFacts, bool) { return s.mount, s.mount.valid() }

// Monster returns the content stat block of an undead creature or familiar form and whether there is one.
func (s SummonedCreatureSpec) Monster() (content.Monster, bool) { return s.monster, s.hasMonst }

// Placement returns whether the creature shares its summoner's Initiative count or has its own.
func (s SummonedCreatureSpec) Placement() PlacementRule { return s.placement }
