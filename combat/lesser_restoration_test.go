package combat_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case (table 4.7b, Lesser Restoration; Source: SRD 5.2.1, "Lesser Restoration" p. 144)
// Path: per case.
func TestApplyLesserRestoration(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		set                     combat.ConditionSet
		kind                    content.ConditionKind
		want                    string
		left                    []content.ConditionKind
	}{
		{"held", "FR-032; table 4.7b", "happy", meConditions(combat.Poisoned{}, combat.Blinded{}), content.ConditionPoisoned, "applied", []content.ConditionKind{content.ConditionBlinded}},
		{"not held", "FR-032; table 4.7b", "unhappy (state error)", meConditions(combat.Blinded{}), content.ConditionDeafened, "nothing", []content.ConditionKind{content.ConditionBlinded}},
		{"wrong kind", "FR-032; table 4.7b", "unhappy (invalid input)", meConditions(combat.Blinded{}), content.ConditionCharmed, "wrong_kind", []content.ConditionKind{content.ConditionBlinded}},
		{"one only", "FR-032; table 4.7b", "unhappy (rule interaction)", meConditions(combat.Blinded{}, combat.Deafened{}), content.ConditionBlinded, "applied", []content.ConditionKind{content.ConditionDeafened}},
		{"Paralyzed", "FR-032", "happy", meConditions(combat.Paralyzed{}), content.ConditionParalyzed, "applied", nil},
		{"Deafened", "FR-032", "happy", meConditions(combat.Deafened{}), content.ConditionDeafened, "applied", nil},
		{"an empty set", "FR-032", "unhappy (empty and minimal)", combat.ConditionSet{}, content.ConditionBlinded, "nothing", nil},
		{"the zero kind", "FR-032", "unhappy (invalid input)", meConditions(combat.Blinded{}), 0, "wrong_kind", []content.ConditionKind{content.ConditionBlinded}},
		{"Frightened is not on the list", "FR-032", "unhappy (invalid input)", meConditions(combat.Frightened{Source: gid("x")}), content.ConditionFrightened, "wrong_kind", []content.ConditionKind{content.ConditionFrightened}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/revival-and-restoration-spells.md (the case's requirement)
			before := c.set
			var got string
			var set combat.ConditionSet
			switch r := combat.ApplyLesserRestoration(gid("hero"), c.set, c.kind).(type) {
			case combat.LesserRestorationApplied:
				got, set = "applied", r.Set()
				if k := rrKinds(r.Events()); !slices.Equal(k, []string{"condition_removed"}) {
					t.Errorf("events = %v, want one condition_removed", k)
				}
			case combat.LesserRestorationNothingToEnd:
				got, set = "nothing", r.Set()
				if len(r.ImpliedBy()) != 0 {
					t.Errorf("ImpliedBy = %v, want none", r.ImpliedBy())
				}
			case combat.LesserRestorationWrongKind:
				got, set = "wrong_kind", c.set
				if r.Kind != c.kind {
					t.Errorf("WrongKind carries %v, want %v", r.Kind, c.kind)
				}
			}
			if got != c.want {
				t.Errorf("result = %s, want %s [%s; %s]", got, c.want, c.requirement, c.path)
			}
			if kinds := rrConditionKinds(set); !slices.Equal(kinds, append([]content.ConditionKind{}, c.left...)) {
				t.Errorf("conditions left = %v, want %v [%s]", kinds, c.left, c.requirement)
			}
			if c.set != before {
				t.Error("the input set changed")
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-032 - a zero subject ends nothing (ConditionSet.Remove treats it as not present), and the result for the same input is equal each call.
// Path: unhappy (invalid input).
func TestApplyLesserRestorationZeroSubject(t *testing.T) {
	t.Parallel()
	set := meConditions(combat.Blinded{})
	got := combat.ApplyLesserRestoration("", set, content.ConditionBlinded)
	if n, ok := got.(combat.LesserRestorationNothingToEnd); !ok || n.Set() != set {
		t.Errorf("a zero subject: %#v, want LesserRestorationNothingToEnd with the set unchanged", got)
	}
	a, b := combat.ApplyLesserRestoration(gid("hero"), set, content.ConditionBlinded), combat.ApplyLesserRestoration(gid("hero"), set, content.ConditionBlinded)
	if !reflect.DeepEqual(a, b) {
		t.Error("equal inputs gave unequal results")
	}
	ev := a.(combat.LesserRestorationApplied).Events()
	ev[0] = nil
	if a.(combat.LesserRestorationApplied).Events()[0] == nil {
		t.Error("Events is not a copy")
	}
}
