package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// idA and idB are two distinct creatures; subject() (hero) is the one conditions are applied to.
func idA() combat.CombatantID { return combat.MustCombatantID("a") }
func idB() combat.CombatantID { return combat.MustCombatantID("b") }

// condLevel is the Exhaustion condition at level n.
func condLevel(n int) combat.ExhaustionCondition {
	return combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(n)}
}

// condSet builds a set from conditions, with the given immunities.
func condSet(tb testing.TB, conds []combat.Condition, immune ...content.ConditionKind) combat.ConditionSet {
	tb.Helper()
	s, err := combat.NewConditionSet(conds, immune)
	if err != nil {
		tb.Fatalf("NewConditionSet: %v", err)
	}
	return s
}

// condKinds is shorthand for a list of kinds.
func condKinds(ks ...content.ConditionKind) []content.ConditionKind { return ks }

// condConds is shorthand for a list of conditions.
func condConds(cs ...combat.Condition) []combat.Condition { return cs }

// condEvents returns the events of a change result, nil for a result without them.
func condEvents(res any) []combat.ConditionEvent {
	if e, ok := res.(interface {
		Events() []combat.ConditionEvent
	}); ok {
		return e.Events()
	}
	return nil
}

// allFifteen is one valid Condition of each kind, in Glossary order.
func allFifteen() []combat.Condition {
	return condConds(combat.Blinded{}, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}}, combat.Deafened{},
		condLevel(1), combat.Frightened{Source: idA()}, combat.Grappled{Grappler: idA()}, combat.Incapacitated{},
		combat.Invisible{}, combat.Paralyzed{}, combat.Petrified{}, combat.Poisoned{}, combat.Prone{},
		combat.Restrained{}, combat.Stunned{}, combat.Unconscious{})
}
