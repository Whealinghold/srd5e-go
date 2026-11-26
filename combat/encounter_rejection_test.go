package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/encounter-aggregate.md FR-095, table 4.10.1 - the srd5e.combat.result name of each rejection is the snake_case table value, and the hit point rule is reported as the carried reason.
// Path: per case.
func TestRejectionNames(t *testing.T) {
	t.Parallel()
	id := cid("a")
	cases := map[string]combat.EncounterRejection{
		"encounter_already_ended":         combat.EncounterAlreadyEndedRejection{},
		"unknown_combatant":               combat.UnknownCombatantRejection{ID: id},
		"combatant_is_dead":               combat.CombatantIsDeadRejection{ID: id},
		"already_has_condition":           combat.AlreadyHasConditionRejection{ID: id},
		"immune_to_condition":             combat.ImmuneToConditionRejection{ID: id},
		"does_not_have_condition":         combat.DoesNotHaveConditionRejection{ID: id},
		"condition_implied_by":            combat.ConditionImpliedByRejection{ID: id},
		"not_dying":                       combat.HitPointRuleRejection{ID: id, Reason: combat.NotDying},
		"not_stable":                      combat.HitPointRuleRejection{ID: id, Reason: combat.NotStable},
		"turn_order_changed":              combat.TurnOrderChangedRejection{},
		"death_saving_throw_already_made": combat.DeathSavingThrowAlreadyMadeRejection{ID: id},
	}
	for want, r := range cases {
		if got := r.String(); got != want {
			t.Errorf("%#v names %q, want %q [FR-095]", r, got, want)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-065 - ImpliedBy returns a copy: changing it leaves the rejection unchanged.
// Path: unhappy (immutability).
func TestConditionImpliedByReturnsACopy(t *testing.T) {
	t.Parallel()
	e := encWith(t, encC(t, "a", combat.SideA, pcStatus(t, 10, 20, 0), combat.Paralyzed{}))
	tr, err := e.RemoveCondition(cid("a"), content.ConditionIncapacitated)
	r := encNo(t, e, tr, err).(combat.ConditionImpliedByRejection)
	got := r.ImpliedBy()
	if !slices.Equal(got, []content.ConditionKind{content.ConditionParalyzed}) {
		t.Fatalf("implied by %v, want [paralyzed] [FR-065]", got)
	}
	got[0] = content.ConditionBlinded
	if r.ImpliedBy()[0] != content.ConditionParalyzed {
		t.Errorf("ImpliedBy is not a copy")
	}
}
