package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/conditions.md FR-074 - a saving throw with an undefined ability is unaffected by conditions that make saves with named abilities fail.
// Path: unhappy (invalid input).
func TestSavingThrowWithAnUndefinedAbilityIsUnaffected(t *testing.T) {
	t.Parallel()
	s := condSet(t, condConds(combat.Paralyzed{}))
	if got := (combat.ConditionEffects{}).SavingThrow(s, ability.Ability(0)); got != combat.SavingThrowEffectUnaffected {
		t.Errorf("SavingThrow(Paralyzed, ability 0) = %v, want unaffected", got)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-012 - MustContainments returns the Containments of a valid list, including the empty one, and panics only for an invalid list.
// Path: happy.
func TestMustContainmentsAcceptsAnEmptyList(t *testing.T) {
	t.Parallel()
	if got := combat.MustContainments(nil).Len(); got != 0 {
		t.Errorf("MustContainments(nil).Len() = %d, want 0", got)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-012 - the terms of a container refuse a locked action name with leading or trailing white space and keep one without.
// Path: unhappy (invalid input).
func TestContainmentTermsLockedActionWhitespace(t *testing.T) {
	t.Parallel()
	terms := func(locked string) error {
		_, err := combat.NewContainmentTerms(combat.ContainmentTermsInput{Mode: combat.ContainmentModeEngulfed, MaxSize: content.SizeLarge,
			Capacity: ctCap(1, 1), Entry: combat.EntryAutomatic{}, Suffocation: combat.SuffocationRuleNone, DeathRule: combat.ReleaseOnDeath{},
			ActionKind: combat.ContainmentActionKindBonusAction, LockedAction: locked})
		return err
	}
	for locked, wantErr := range map[string]bool{"": false, "Slam": false, " Slam": true, "Slam ": true, "\tSlam": true} {
		if err := terms(locked); (err != nil) != wantErr {
			t.Errorf("LockedAction %q: err = %v, want an error: %v", locked, err, wantErr)
		}
	}
}
