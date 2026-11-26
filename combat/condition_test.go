package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/conditions.md FR-002, FR-130 - there is exactly one Condition variant per ConditionKind, each
// reports its Kind, an exhaustive type switch covers all fifteen, and the kinds are in Glossary order (SRD 5.2.1,
// "Condition", p. 179).
// Path: happy.
func TestConditionHasOneVariantPerKindInGlossaryOrder(t *testing.T) {
	t.Parallel()
	glossary := []string{"Blinded", "Charmed", "Deafened", "Exhaustion", "Frightened", "Grappled", "Incapacitated", "Invisible",
		"Paralyzed", "Petrified", "Poisoned", "Prone", "Restrained", "Stunned", "Unconscious"}
	conds := allFifteen()
	if len(conds) != len(glossary) {
		t.Fatalf("fixture has %d conditions, want 15", len(conds))
	}
	for i, c := range conds {
		kind := content.ConditionKind(i + 1)
		if kind.String() != glossary[i] || c.Kind() != kind {
			t.Errorf("condition %d: kind %v (%s), variant reports %v, want %s", i, kind, kind.String(), c.Kind(), glossary[i])
		}
		var got content.ConditionKind
		switch c.(type) {
		case combat.Blinded:
			got = content.ConditionBlinded
		case combat.Charmed:
			got = content.ConditionCharmed
		case combat.Deafened:
			got = content.ConditionDeafened
		case combat.ExhaustionCondition:
			got = content.ConditionExhaustion
		case combat.Frightened:
			got = content.ConditionFrightened
		case combat.Grappled:
			got = content.ConditionGrappled
		case combat.Incapacitated:
			got = content.ConditionIncapacitated
		case combat.Invisible:
			got = content.ConditionInvisible
		case combat.Paralyzed:
			got = content.ConditionParalyzed
		case combat.Petrified:
			got = content.ConditionPetrified
		case combat.Poisoned:
			got = content.ConditionPoisoned
		case combat.Prone:
			got = content.ConditionProne
		case combat.Restrained:
			got = content.ConditionRestrained
		case combat.Stunned:
			got = content.ConditionStunned
		case combat.Unconscious:
			got = content.ConditionUnconscious
		}
		if got != kind {
			t.Errorf("type switch maps %T to %v, want %v", c, got, kind)
		}
	}
}

// Requirement: doc/specs/conditions.md FR-003, FR-004, FR-141 - two Conditions are equal under == when the kind and the
// source or level are the same, and not otherwise; a charmer is a creature or an item.
// Path: per case.
func TestConditionEquality(t *testing.T) {
	t.Parallel()
	item, _ := combat.NewCharmedByItem("sword")
	other, _ := combat.NewCharmedByItem("ring")
	cases := []struct {
		name, requirement, path string
		a, b                    combat.Condition
		equal                   bool
	}{
		{"same kind without data", "FR-004", "happy", combat.Blinded{}, combat.Blinded{}, true},
		{"different kinds", "FR-004", "unhappy (boundary)", combat.Blinded{}, combat.Deafened{}, false},
		{"same Frightened source", "FR-003, FR-004", "happy", combat.Frightened{Source: idA()}, combat.Frightened{Source: idA()}, true},
		{"different Frightened source", "FR-004", "unhappy (boundary)", combat.Frightened{Source: idA()}, combat.Frightened{Source: idB()}, false},
		{"different Grappled grappler", "FR-004", "unhappy (boundary)", combat.Grappled{Grappler: idA()}, combat.Grappled{Grappler: idB()}, false},
		{"same Exhaustion level", "FR-004", "happy", condLevel(2), condLevel(2), true},
		{"different Exhaustion level", "FR-004", "unhappy (boundary)", condLevel(2), condLevel(3), false},
		{"same creature charmer", "FR-141", "happy", combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}}, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}}, true},
		{"different creature charmer", "FR-141", "unhappy (boundary)", combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}}, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idB()}}, false},
		{"same item charmer", "FR-141", "happy", combat.Charmed{Charmer: item}, combat.Charmed{Charmer: item}, true},
		{"different item charmer", "FR-141", "unhappy (boundary)", combat.Charmed{Charmer: item}, combat.Charmed{Charmer: other}, false},
		{"creature versus item charmer", "FR-141", "unhappy (boundary)", combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}}, combat.Charmed{Charmer: item}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.a == c.b; got != c.equal {
				t.Errorf("%v == %v is %v, want %v [conditions.md %s]", c.a, c.b, got, c.equal, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-141 - NewCharmedByItem accepts an opaque non-blank id and rejects a blank one
// with an error matching ErrInvalidInput; the id is not trimmed.
// Path: unhappy (invalid input).
func TestNewCharmedByItem(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"", " ", "\t\n"} {
		if c, err := combat.NewCharmedByItem(id); !errors.Is(err, srderr.ErrInvalidInput) || c != (combat.CharmedByItem{}) {
			t.Errorf("NewCharmedByItem(%q) = %v, %v, want the zero value and ErrInvalidInput", id, c, err)
		}
	}
	c, err := combat.NewCharmedByItem(" sword ")
	if err != nil || c.ItemID() != " sword " {
		t.Errorf("NewCharmedByItem(\" sword \") = %q, %v, want the id unchanged", c.ItemID(), err)
	}
}

// Requirement: doc/specs/conditions.md FR-141, testing.md section 10 - NewCharmedByItem never panics and rejects
// exactly the blank ids.
// Path: unhappy (invalid input).
func FuzzNewCharmedByItem(f *testing.F) {
	for _, s := range []string{"", " ", "sword", " ", "a\x00b"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		c, err := combat.NewCharmedByItem(s)
		if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("NewCharmedByItem(%q) error %v does not match ErrInvalidInput", s, err)
		}
		if err == nil && c.ItemID() != s {
			t.Fatalf("NewCharmedByItem(%q) changed the id to %q", s, c.ItemID())
		}
	})
}

// Requirement: doc/specs/conditions.md FR-003, FR-141 - a Condition whose source or level is unset is invalid and
// NewConditionSet and the Apply command reject it; a nil Condition and a pointer to a variant are invalid too.
// Path: unhappy (invalid input).
func TestInvalidConditionsAreRejected(t *testing.T) {
	t.Parallel()
	bad := map[string]combat.Condition{
		"nil":                       nil,
		"Charmed without charmer":   combat.Charmed{},
		"Charmed by zero creature":  combat.Charmed{Charmer: combat.CharmedByCombatant{}},
		"Charmed by zero item":      combat.Charmed{Charmer: combat.CharmedByItem{}},
		"Frightened without source": combat.Frightened{},
		"Grappled without grappler": combat.Grappled{},
		"Exhaustion without level":  combat.ExhaustionCondition{},
		"pointer to a variant":      &combat.Blinded{},
	}
	for name, c := range bad {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := combat.NewConditionSet(condConds(c), nil); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("NewConditionSet(%s) error = %v, want ErrInvalidInput", name, err)
			}
			if _, err := combat.NewApplyConditionToSetCommand(combat.ConditionSet{}, subject(), c); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("NewApplyConditionToSetCommand(%s) error = %v, want ErrInvalidInput", name, err)
			}
			if _, err := combat.NewConditionApplied(subject(), c); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("NewConditionApplied(%s) error = %v, want ErrInvalidInput", name, err)
			}
			res := (combat.ConditionSet{}).Apply(subject(), c)
			if _, ok := res.(combat.ConditionAlreadyHeld); !ok || len(res.Set().Effective()) != 0 {
				t.Errorf("Apply(%s) = %#v, want ConditionAlreadyHeld and an empty set", name, res)
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-002, FR-003, FR-016 - each of the fifteen valid conditions is accepted by
// NewConditionSet, the Apply command and the applied event.
// Path: happy.
func TestEveryValidConditionIsAccepted(t *testing.T) {
	t.Parallel()
	for _, c := range allFifteen() {
		if s, err := combat.NewConditionSet(condConds(c), nil); err != nil || len(s.Explicit()) != 1 {
			t.Errorf("NewConditionSet(%v) = %v, %v, want it accepted", c, s, err)
		}
		if _, err := combat.NewApplyConditionToSetCommand(combat.ConditionSet{}, subject(), c); err != nil {
			t.Errorf("NewApplyConditionToSetCommand(%v) = %v, want it accepted", c, err)
		}
		if e, err := combat.NewConditionApplied(subject(), c); err != nil || e.Condition() != c {
			t.Errorf("NewConditionApplied(%v) = %v, %v, want it accepted", c, e, err)
		}
	}
}
