package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// SummonEvent is the closed family of events the rules for summoned and companion creatures produce (summoned-and-companion-
// creatures.md FR-069). Each is an immutable past-tense value with a constant snake_case Kind and a Combatants list that is
// never empty and never holds an id twice, a fresh copy each call. None is an envelope. The summon kind of an event is read
// from SummonKind, since Kind is the name of the event. Events are compared with reflect.DeepEqual or by their accessors, since
// they hold lists. Variants: SummonedCreatureJoined, SummonedCreatureCommanded, SummonedCreatureDeparted, SharedResistanceEnded,
// SummonControlReasserted, SummonControlLapsed, FamiliarFormChanged, FamiliarDismissedTemporarily, FamiliarReappeared,
// FamiliarSightLinkStarted, FamiliarSightLinkEnded, SimulacrumRepaired. Each embeds summonEventBase.
type SummonEvent interface {
	Event
	summonEvent()
}

// summonEventBase carries the creatures an event is about and puts the event in the Event and SummonEvent families.
type summonEventBase struct{ ids []CombatantID }

func (summonEventBase) event()       {}
func (summonEventBase) summonEvent() {}

// Combatants returns a new slice of the creatures the event is about, the creature first and the summoner after it.
func (b summonEventBase) Combatants() []CombatantID { return slices.Clone(b.ids) }

// pair returns the base of an event about a creature and its summoner.
func pair(creature, summoner CombatantID) summonEventBase {
	return summonEventBase{[]CombatantID{creature, summoner}}
}

// SummonedCreatureJoined: a summoned creature was brought into the fight on Side. Kind summoned_creature_joined. Combatants is
// [creature, summoner].
type SummonedCreatureJoined struct {
	summonEventBase
	kind      SummonKind
	side      Side
	level     int
	placement PlacementRule
	grant     SharedResistanceGrant
}

// Kind returns "summoned_creature_joined".
func (SummonedCreatureJoined) Kind() string { return "summoned_creature_joined" }

// SummonKind returns the kind of the creature.
func (e SummonedCreatureJoined) SummonKind() SummonKind { return e.kind }

// Side returns the side the creature fights on, its summoner's (FR-001).
func (e SummonedCreatureJoined) Side() Side { return e.side }

// CastLevel returns the level the spell was cast at.
func (e SummonedCreatureJoined) CastLevel() int { return e.level }

// Placement returns whether the creature shares its summoner's Initiative count or has its own.
func (e SummonedCreatureJoined) Placement() PlacementRule { return e.placement }

// SharedResistance returns the Resistance the summoner shares through a Draconic Spirit and whether there is one (FR-027).
func (e SummonedCreatureJoined) SharedResistance() (SharedResistanceGrant, bool) {
	return e.grant, e.grant.Damage != 0
}

// SummonedCreatureCommanded: the summoner gave an order to creatures of one spell. Kind summoned_creature_commanded. Combatants is
// [summoner, the creatures in the order given].
type SummonedCreatureCommanded struct {
	summonEventBase
	kind  SummonKind
	order SummonedOrder
}

// Kind returns "summoned_creature_commanded".
func (SummonedCreatureCommanded) Kind() string { return "summoned_creature_commanded" }

// SummonKind returns the kind of the creatures commanded.
func (e SummonedCreatureCommanded) SummonKind() SummonKind { return e.kind }

// Order returns the order given.
func (e SummonedCreatureCommanded) Order() SummonedOrder { return e.order }

// SummonedCreatureDeparted: a summoned creature stopped being linked to its summoner. Kind summoned_creature_departed.
// Combatants is [creature, summoner]. The host removes the creature through RemoveCombatantFromEncounter, whose CombatantRemoved
// event carries the removal, except for an undead creature that died, which stays as a body (FR-025, FR-026).
type SummonedCreatureDeparted struct {
	summonEventBase
	kind                         SummonKind
	cause                        SummonEndCause
	pendingRemoval, dropsItems   bool
	revertsToObject, revertsSnow bool
	leavesBody                   bool
	overflow                     int
}

// Kind returns "summoned_creature_departed".
func (SummonedCreatureDeparted) Kind() string { return "summoned_creature_departed" }

// SummonKind returns the kind of the creature.
func (e SummonedCreatureDeparted) SummonKind() SummonKind { return e.kind }

// Cause returns why the creature went.
func (e SummonedCreatureDeparted) Cause() SummonEndCause { return e.cause }

// PendingRemoval reports whether the creature is the active combatant, so the host ends its turn before it removes it (FR-077).
func (e SummonedCreatureDeparted) PendingRemoval() bool { return e.pendingRemoval }

// DropsCarriedItems reports whether the creature leaves what it carried behind.
func (e SummonedCreatureDeparted) DropsCarriedItems() bool { return e.dropsItems }

// RevertsToObject reports whether an animated object turns back into the object, with the Overflow damage (FR-020).
func (e SummonedCreatureDeparted) RevertsToObject() bool { return e.revertsToObject }

// RevertsToSnow reports whether a simulacrum melts away into snow (FR-024).
func (e SummonedCreatureDeparted) RevertsToSnow() bool { return e.revertsSnow }

// LeavesBody reports whether an undead creature that dropped to 0 Hit Points stays as a body, dying under Monster Death
// (FR-025).
func (e SummonedCreatureDeparted) LeavesBody() bool { return e.leavesBody }

// Overflow returns the damage that carries over to the object an animated object reverts to (FR-021), 0 for any other
// departure.
func (e SummonedCreatureDeparted) Overflow() int { return e.overflow }

// SharedResistanceEnded: the Resistance a Draconic Spirit's summoner shared ended with the spell, so the host removes it. Kind
// shared_resistance_ended. Combatants is [summoner].
type SharedResistanceEnded struct {
	summonEventBase
	grant SharedResistanceGrant
}

// Kind returns "shared_resistance_ended".
func (SharedResistanceEnded) Kind() string { return "shared_resistance_ended" }

// Grant returns the Resistance that ended.
func (e SharedResistanceEnded) Grant() SharedResistanceGrant { return e.grant }

// SummonControlReasserted: the summoner took control of undead creatures again for 24 hours. Kind summon_control_reasserted.
// Combatants is [summoner, the creatures].
type SummonControlReasserted struct {
	summonEventBase
	kind SummonKind
}

// Kind returns "summon_control_reasserted".
func (SummonControlReasserted) Kind() string { return "summon_control_reasserted" }

// SummonKind returns the kind of the undead creatures.
func (e SummonControlReasserted) SummonKind() SummonKind { return e.kind }

// SummonControlLapsed: the 24 hours of control over an undead creature ran out. It is not destroyed and stays linked, and what it
// does is the Game Master's (FR-040). Kind summon_control_lapsed. Combatants is [creature, summoner].
type SummonControlLapsed struct {
	summonEventBase
	kind SummonKind
}

// Kind returns "summon_control_lapsed".
func (SummonControlLapsed) Kind() string { return "summon_control_lapsed" }

// SummonKind returns the kind of the undead creature.
func (e SummonControlLapsed) SummonKind() SummonKind { return e.kind }

// FamiliarFormChanged: casting Find Familiar again changed the form of the familiar the summoner has (FR-044). Kind
// familiar_form_changed. Combatants is [familiar, summoner].
type FamiliarFormChanged struct {
	summonEventBase
	form    character.FamiliarForm
	typ     content.CreatureType
	origin  FamiliarOrigin
	maximum int
}

// Kind returns "familiar_form_changed".
func (FamiliarFormChanged) Kind() string { return "familiar_form_changed" }

// Form returns the new form.
func (e FamiliarFormChanged) Form() character.FamiliarForm { return e.form }

// CreatureType returns the creature type of the familiar.
func (e FamiliarFormChanged) CreatureType() content.CreatureType { return e.typ }

// Origin returns the origin of the latest cast.
func (e FamiliarFormChanged) Origin() FamiliarOrigin { return e.origin }

// HitPointMaximum returns the Hit Point maximum of the new form, which the familiar's current Hit Points become (Q-6).
func (e FamiliarFormChanged) HitPointMaximum() int { return e.maximum }

// FamiliarDismissedTemporarily: the familiar went to its pocket dimension (FR-050). Kind familiar_dismissed_temporarily.
// Combatants is [familiar, summoner].
type FamiliarDismissedTemporarily struct {
	summonEventBase
	pendingRemoval bool
}

// Kind returns "familiar_dismissed_temporarily".
func (FamiliarDismissedTemporarily) Kind() string { return "familiar_dismissed_temporarily" }

// DropsCarriedItems returns true: the familiar leaves what it carried behind.
func (FamiliarDismissedTemporarily) DropsCarriedItems() bool { return true }

// ActionCost returns StandardActionMagic: the host takes SimpleActionChoice(magic) (FR-055).
func (FamiliarDismissedTemporarily) ActionCost() StandardAction { return StandardActionMagic }

// PendingRemoval reports whether the familiar is the active combatant (FR-077).
func (e FamiliarDismissedTemporarily) PendingRemoval() bool { return e.pendingRemoval }

// FamiliarReappeared: the familiar came back from its pocket dimension with the Hit Points it had (FR-051). Kind
// familiar_reappeared. Combatants is [familiar, summoner].
type FamiliarReappeared struct{ summonEventBase }

// Kind returns "familiar_reappeared".
func (FamiliarReappeared) Kind() string { return "familiar_reappeared" }

// ActionCost returns StandardActionMagic: the host takes SimpleActionChoice(magic) (FR-055).
func (FamiliarReappeared) ActionCost() StandardAction { return StandardActionMagic }

// FamiliarSightLinkStarted: the summoner sees through the familiar's eyes and hears what it hears until the start of the
// summoner's next turn (FR-046). Kind familiar_sight_link_started. Combatants is [familiar, summoner].
type FamiliarSightLinkStarted struct{ summonEventBase }

// Kind returns "familiar_sight_link_started".
func (FamiliarSightLinkStarted) Kind() string { return "familiar_sight_link_started" }

// FamiliarSightLinkEnded: the sight link ended at the start of the summoner's turn. Kind familiar_sight_link_ended. Combatants is
// [familiar, summoner].
type FamiliarSightLinkEnded struct{ summonEventBase }

// Kind returns "familiar_sight_link_ended".
func (FamiliarSightLinkEnded) Kind() string { return "familiar_sight_link_ended" }

// SimulacrumRepaired: a simulacrum regained Hit Points through its repair (FR-062). Kind simulacrum_repaired. Combatants is
// [simulacrum, summoner].
type SimulacrumRepaired struct {
	summonEventBase
	restored, goldPieces int
}

// Kind returns "simulacrum_repaired".
func (SimulacrumRepaired) Kind() string { return "simulacrum_repaired" }

// HitPointsRestored returns the Hit Points restored.
func (e SimulacrumRepaired) HitPointsRestored() int { return e.restored }

// GoldPieces returns the cost of the components, 100 GP for each Hit Point restored.
func (e SimulacrumRepaired) GoldPieces() int { return e.goldPieces }
