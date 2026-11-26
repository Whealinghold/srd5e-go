package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/shape-changing-spells.md FR-058 and section 7 - every rejection member has the snake_case name that is the value of srd5e.combat.result, and no other member exists.
// Path: per case.
func TestShapeShiftRejectionNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		got                     string
		want                    string
	}{
		{"target is dead", "FR-058", "happy", combat.TargetIsDead{}.String(), "target_is_dead"},
		{"target not willing", "FR-058", "happy", combat.TargetNotWilling{}.String(), "target_not_willing"},
		{"shapechange not self", "FR-058", "happy", combat.ShapechangeNotSelf{}.String(), "shapechange_not_self"},
		{"form ineligible", "FR-058", "happy", combat.FormIneligible{}.String(), "form_ineligible"},
		{"form change not allowed", "FR-058", "happy", combat.FormChangeNotAllowed{}.String(), "form_change_not_allowed"},
		{"not transformed", "FR-058", "happy", combat.NotTransformed{}.String(), "not_transformed"},
		{"actor not caster", "FR-058", "happy", combat.ActorNotCaster{}.String(), "actor_not_caster"},
		{"cannot end at will", "FR-058", "happy", combat.CannotEndAtWill.String(), "cannot_end_at_will"},
		{"end, target is dead", "FR-058", "happy", combat.EndTargetIsDead.String(), "target_is_dead"},
		{"the zero end rejection", "FR-058", "unhappy (invalid input)", combat.EndShapeShiftRejection(0).String(), "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.got != c.want {
				t.Errorf("name = %q, want %q [%s]", c.got, c.want, c.requirement)
			}
		})
	}
	var _ combat.BeginShapeShiftRejection = combat.TargetIsDead{}
	var _ combat.ChangeFormRejection = combat.FormIneligible{}
}

// Requirement: doc/specs/shape-changing-spells.md FR-058 - the rejection sets are closed: Begin refuses with exactly its four members and ChangeForm with its five; every refusal the operations return is one of them (an exhaustive switch over each set).
// Path: unhappy (invalid input).
func TestShapeShiftRejectionSetsAreClosed(t *testing.T) {
	t.Parallel()
	begin := []combat.BeginShapeShiftRejection{combat.TargetIsDead{}, combat.TargetNotWilling{}, combat.ShapechangeNotSelf{}, combat.FormIneligible{}}
	for _, r := range begin {
		switch r.(type) {
		case combat.TargetIsDead, combat.TargetNotWilling, combat.ShapechangeNotSelf, combat.FormIneligible:
		default:
			t.Errorf("%T is outside the Begin set", r)
		}
	}
	change := []combat.ChangeFormRejection{combat.TargetIsDead{}, combat.FormChangeNotAllowed{}, combat.NotTransformed{}, combat.ActorNotCaster{}, combat.FormIneligible{}}
	for _, r := range change {
		switch r.(type) {
		case combat.TargetIsDead, combat.FormChangeNotAllowed, combat.NotTransformed, combat.ActorNotCaster, combat.FormIneligible:
		default:
			t.Errorf("%T is outside the ChangeForm set", r)
		}
	}
}
