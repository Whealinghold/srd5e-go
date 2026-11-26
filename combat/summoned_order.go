package combat

import (
	"time"

	"github.com/Whealinghold/srd5e-go/content"
)

// OrderToken is an opaque, host-defined name of an action, a movement or a place in an order (FR-012). The engine resolves none
// of them. The empty token is invalid.
type OrderToken string

// SummonedOrder is what the summoner told a creature to do on its next turn (FR-012). Variants: SpecificOrder, GeneralOrder.
// Both compare by value.
type SummonedOrder interface{ isSummonedOrder() }

// SpecificOrder is an action and a movement for the creature's next turn. Build it with a keyed literal.
type SpecificOrder struct{ Action, Movement OrderToken }

// GeneralOrder is a general instruction, such as guarding a place. Build it with a keyed literal.
type GeneralOrder struct{ Label OrderToken }

func (SpecificOrder) isSummonedOrder() {}
func (GeneralOrder) isSummonedOrder()  {}

// validSummonedOrder reports whether o is a variant whose tokens are not empty.
func validSummonedOrder(o SummonedOrder) bool {
	switch v := o.(type) {
	case SpecificOrder:
		return v.Action != "" && v.Movement != ""
	case GeneralOrder:
		return v.Label != ""
	}
	return false
}

// SummonedTurnPlan is what a summoned creature does on its turn (FR-013). Variants: TurnPlanOrdered, TurnPlanDefaultDodge.
type SummonedTurnPlan interface{ isSummonedTurnPlan() }

// TurnPlanOrdered: the creature carries out Order.
type TurnPlanOrdered struct{ Order SummonedOrder }

// TurnPlanDefaultDodge: the creature has no order, so it takes the Dodge action and moves only to avoid harm (SRD 5.2.1, "Dodge",
// p. 181).
type TurnPlanDefaultDodge struct{}

func (TurnPlanOrdered) isSummonedTurnPlan()      {}
func (TurnPlanDefaultDodge) isSummonedTurnPlan() {}

// ControlWindowStart is the 24 hours an undead creature stays under control after it is raised or control is reasserted (SRD
// 5.2.1, "Animate Dead", p. 108; FR-040).
const ControlWindowStart = 24 * time.Hour

// ControlWindow is the time left of the control the summoner has over an undead creature (FR-040). The zero value is no window
// (the creature is not an undead creature). A window that lapsed never revives. Comparable and immutable.
type ControlWindow struct {
	left   time.Duration
	lapsed bool
	set    bool
}

// IsZero reports whether w is the zero value, no window.
func (w ControlWindow) IsZero() bool { return !w.set }

// Left returns the time left, never negative, 0 once the window lapsed.
func (w ControlWindow) Left() time.Duration { return w.left }

// Lapsed reports whether the window ran out.
func (w ControlWindow) Lapsed() bool { return w.lapsed }

// startedWindow returns a window of the full 24 hours.
func startedWindow() ControlWindow { return ControlWindow{left: ControlWindowStart, set: true} }

// elapse returns w with d taken off; at 0 or below it is lapsed. A lapsed window stays lapsed.
func (w ControlWindow) elapse(d time.Duration) ControlWindow {
	if !w.set || w.lapsed {
		return w
	}
	if d >= w.left {
		return ControlWindow{lapsed: true, set: true}
	}
	return ControlWindow{left: w.left - d, set: true}
}

// FamiliarAttackGrant says a feature lets a familiar take the Attack action (FR-048). The zero value is no grant.
type FamiliarAttackGrant int

// The two grants, supplied by the Pact of the Chain's riders of the character context.
const (
	// FamiliarAttackViaReaction: the Warlock forwent an attack so the familiar attacks with its Reaction.
	FamiliarAttackViaReaction FamiliarAttackGrant = iota + 1
	// FamiliarAttackViaBonusActionCommand: the Warlock commanded the attack as a Bonus Action.
	FamiliarAttackViaBonusActionCommand
)

// Valid reports whether g is one of the two grants.
func (g FamiliarAttackGrant) Valid() bool {
	return g == FamiliarAttackViaReaction || g == FamiliarAttackViaBonusActionCommand
}

// String returns "via_reaction", "via_bonus_action_command" or "none".
func (g FamiliarAttackGrant) String() string {
	if !g.Valid() {
		return "none"
	}
	return enumName([]string{"via_reaction", "via_bonus_action_command"}, int(g))
}

// SharedResistanceGrant is the Resistance a Draconic Spirit's summoner has while the spell lasts (SRD 5.2.1, "Draconic Spirit",
// p. 166; FR-027). Build it with a keyed literal. It compares by value.
type SharedResistanceGrant struct {
	Summoner CombatantID
	Spell    content.SpellID
	Damage   content.DamageType
}

// SummonedRosterAction is the instruction a transition returns for the roster of the encounter: the host passes a join to
// AddCombatantToEncounter and a leave to RemoveCombatantFromEncounter. This package never calls a roster operation (FR-077).
// Variants: SummonedRosterJoin, SummonedRosterLeave.
type SummonedRosterAction interface{ isSummonedRosterAction() }

// SummonedRosterJoin tells the host to add Creature at Position with Reason (always JoinReasonSummoned). Build it with a keyed
// literal. It compares by value.
type SummonedRosterJoin struct {
	Creature CombatantID
	Position RosterPosition
	Reason   JoinReason
}

// SummonedRosterLeave tells the host to remove Creature with Cause. When AfterTurn is true the creature is the active combatant:
// the host ends the turn with AdvanceCombatTurn first and only then removes it. It compares by value.
type SummonedRosterLeave struct {
	Creature  CombatantID
	Cause     RemovalCause
	AfterTurn bool
}

func (SummonedRosterJoin) isSummonedRosterAction()  {}
func (SummonedRosterLeave) isSummonedRosterAction() {}
