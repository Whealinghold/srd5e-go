package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/conditions.md FR-113, FR-114, FR-139 - the three condition events are immutable values with
// the constant kinds condition_applied, condition_removed and condition_level_reduced, about exactly the target, with
// a fresh Combatants copy, comparable with ==, and carrying their condition, kind or level.
// Path: happy.
func TestConditionEvents(t *testing.T) {
	t.Parallel()
	app := condApplied(combat.Frightened{Source: idA()})
	rem := condRemoved(cG)
	red := condLowered(4)
	combattest.CheckEvent(t, app, "condition_applied", subject())
	combattest.CheckEvent(t, rem, "condition_removed", subject())
	combattest.CheckEvent(t, red, "condition_level_reduced", subject())
	combattest.CheckEventKinds(t, []combat.Event{app, rem, red})
	var _ combat.ConditionEvent = app
	if app.Condition() != (combat.Frightened{Source: idA()}) || rem.ConditionKind() != cG || red.Level().Value() != 4 {
		t.Errorf("accessors = %v, %v, %v", app.Condition(), rem.ConditionKind(), red.Level())
	}
	if app != condApplied(combat.Frightened{Source: idA()}) || app == condApplied(combat.Frightened{Source: idB()}) {
		t.Errorf("ConditionApplied is not compared by value")
	}
	if rem != condRemoved(cG) || rem == condRemoved(cB) || red != condLowered(4) || red == condLowered(3) {
		t.Errorf("events are not compared by value")
	}
}

// Requirement: doc/specs/conditions.md FR-113, FR-114 - a constructor rejects a zero target and an invalid payload with an
// error matching ErrInvalidInput.
// Path: unhappy (invalid input).
func TestConditionEventConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewConditionApplied("", combat.Blinded{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewConditionApplied with a zero target: %v", err)
	}
	if _, err := combat.NewConditionRemoved("", cB); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewConditionRemoved with a zero target: %v", err)
	}
	if _, err := combat.NewConditionRemoved(subject(), 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewConditionRemoved of kind 0: %v", err)
	}
	if _, err := combat.NewConditionLevelReduced("", combat.MustExhaustionLevel(1)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewConditionLevelReduced with a zero target: %v", err)
	}
	if _, err := combat.NewConditionLevelReduced(subject(), combat.ExhaustionLevel{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewConditionLevelReduced with the zero level: %v", err)
	}
}
