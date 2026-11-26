package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/encounter-aggregate.md FR-083, FR-082 - NewCombatant rejects an unusable part with ErrInvalidInput and adds no invariant between the valid parts (a Dying combatant without Unconscious is accepted); Equal differs in each of the seven parts.
// Path: unhappy (invalid input).
func TestCombatantConstructorAndEquality(t *testing.T) {
	t.Parallel()
	ref := combat.CombatantRef{ID: cid("a"), Kind: combat.MonsterCombatant{ID: "goblin"}}
	hp, set := pcStatus(t, 5, 5, 0), combat.ConditionSet{}
	for name, f := range map[string]func() (combat.Combatant, error){
		"zero id": func() (combat.Combatant, error) {
			return combat.NewCombatant(combat.CombatantRef{Kind: ref.Kind}, combat.SideA, hp, set)
		},
		"nil kind": func() (combat.Combatant, error) {
			return combat.NewCombatant(combat.CombatantRef{ID: ref.ID}, combat.SideA, hp, set)
		},
		"bad side": func() (combat.Combatant, error) { return combat.NewCombatant(ref, 0, hp, set) },
		"zero hp": func() (combat.Combatant, error) {
			return combat.NewCombatant(ref, combat.SideA, combat.HitPointStatus{}, set)
		},
		"third side": func() (combat.Combatant, error) { return combat.NewCombatant(ref, 3, hp, set) },
	} {
		if c, err := f(); !errors.Is(err, srderr.ErrInvalidInput) || !c.IsZero() {
			t.Errorf("%s: NewCombatant = %v, %v, want a zero combatant and ErrInvalidInput [FR-083]", name, c, err)
		}
	}
	base, err := combat.NewCombatant(ref, combat.SideA, dyingStatus(t, 20, 0, 0), set)
	if err != nil {
		t.Fatalf("a Dying combatant without Unconscious is accepted [FR-083]: %v", err)
	}
	other := func(r combat.CombatantRef, s combat.Side, h combat.HitPointStatus, cs combat.ConditionSet) combat.Combatant {
		c, err := combat.NewCombatant(r, s, h, cs)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	d := dyingStatus(t, 20, 0, 0)
	same := other(ref, combat.SideA, d, set)
	diff := []combat.Combatant{
		other(combat.CombatantRef{ID: cid("z"), Kind: ref.Kind}, combat.SideA, d, set),
		other(combat.CombatantRef{ID: ref.ID, Kind: combat.MonsterCombatant{ID: "orc"}}, combat.SideA, d, set),
		other(ref, combat.SideB, d, set),
		other(ref, combat.SideA, pcStatus(t, 5, 5, 0), set),
		other(ref, combat.SideA, d, condSet(t, condConds(combat.Blinded{}))),
		base.WithKnockedOut(true),
		base.WithDeathSavingThrowMade(true),
	}
	if !base.Equal(same) || base.IsZero() {
		t.Errorf("equal combatants are not equal [FR-082]")
	}
	for i, d := range diff {
		if base.Equal(d) {
			t.Errorf("case %d differs in one part yet is equal [FR-082]", i)
		}
	}
	if base.WithKnockedOut(true).WithKnockedOut(false) != base || !base.WithKnockedOut(true).KnockedOut() || !base.WithDeathSavingThrowMade(true).DeathSavingThrowMade() {
		t.Errorf("the marker setters do not return a changed copy")
	}
	if _, dead := base.Death(); dead {
		t.Errorf("a Dying combatant has a death")
	}
	if cause, ok := other(ref, combat.SideA, deadStatus(t, combat.Exhaustion), set).Death(); !ok || cause != combat.Exhaustion {
		t.Errorf("Death = %v, %v, want exhaustion [FR-068]", cause, ok)
	}
	if base.Side() != combat.SideA || base.Ref() != ref || !base.Conditions().Equal(set) {
		t.Errorf("accessors do not return the parts")
	}
}
