package d20_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/d20"
)

func TestEnumStrings(t *testing.T) {
	t.Parallel()
	tests := []struct{ got, want string }{
		{d20.AbilityCheck.String(), "ability_check"},
		{d20.SavingThrow.String(), "saving_throw"},
		{d20.AttackRoll.String(), "attack_roll"},
		{d20.Kind(0).String(), "unknown"},
		{d20.Kind(9).String(), "unknown"},
		{d20.Success.String(), "success"},
		{d20.Failure.String(), "failure"},
		{d20.Outcome(0).String(), "unknown"},
		{d20.ChosenToFail.String(), "chosen_to_fail"},
		{d20.Automatic.String(), "automatic"},
		{d20.ForcedFailureReason(0).String(), "unknown"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("String() = %q, want %q", tc.got, tc.want)
		}
	}
}

func TestKindValid(t *testing.T) {
	t.Parallel()
	for k, want := range map[d20.Kind]bool{0: false, d20.AbilityCheck: true, d20.SavingThrow: true, d20.AttackRoll: true, 4: false} {
		if got := k.Valid(); got != want {
			t.Errorf("Kind(%d).Valid() = %v, want %v", k, got, want)
		}
	}
}

func TestTypicalDifficultyClasses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		dc   d20.DifficultyClass
		want int
	}{
		{d20.DCVeryEasy, 5}, {d20.DCEasy, 10}, {d20.DCMedium, 15},
		{d20.DCHard, 20}, {d20.DCVeryHard, 25}, {d20.DCNearlyImpossible, 30},
	}
	for _, tc := range tests {
		if int(tc.dc) != tc.want {
			t.Errorf("DC = %d, want %d", tc.dc, tc.want)
		}
	}
}

func TestSignalNamesArePinned(t *testing.T) {
	t.Parallel()
	tests := []struct{ got, want string }{
		{d20.SpanResolve, "srd5e.d20_test.resolve"},
		{d20.MetricResolved, "srd5e.d20_test.resolved"},
		{d20.MetricDuration, "srd5e.d20_test.duration"},
		{d20.MetricModifiers, "srd5e.d20_test.modifiers"},
		{d20.MetricNatural, "srd5e.d20_test.natural"},
		{d20.MetricErrors, "srd5e.d20_test.errors"},
		{d20.AttrKind, "srd5e.d20_test.kind"},
		{d20.AttrAbility, "srd5e.ability"},
		{d20.AttrRollMode, "srd5e.d20_test.roll_mode"},
		{d20.AttrCriticalHit, "srd5e.d20_test.critical_hit"},
		{d20.AttrRollerFailure, "srd5e.d20_test.roller_failure"},
		{d20.AttrForcedFailureReason, "srd5e.d20_test.forced_failure_reason"},
		{d20.ValueKindAbilityCheck, "ability_check"},
		{d20.ValueKindSavingThrow, "saving_throw"},
		{d20.ValueKindAttackRoll, "attack_roll"},
		{d20.ValueRollerUnavailable, "unavailable"},
		{d20.ValueRollerInvalidResult, "invalid_result"},
		{d20.ValueForcedChosenToFail, "chosen_to_fail"},
		{d20.ValueForcedAutomatic, "automatic"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("signal constant = %q, want %q", tc.got, tc.want)
		}
	}
}
