package combat_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: spell-effects.md FR-050 - a spell whose instances or amounts are read from a character-level table, cast with a report that has no character level, is a command error that names the missing level, before any rejection reason is weighed and before any die is rolled.
// Path: unhappy (invalid input).
func TestMutgapCSpellNeedingACharacterLevelRefusesAReportWithoutOne(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		spell   content.Spell
		targets []combat.SpellTarget
	}{
		"instances from a table": {seEldritchBlast(), []combat.SpellTarget{seWithFacts("a")}},
		// Three targets are too many for Acid Splash: were the level not checked first, this would be a rejection.
		"amount from a table": {seAcidSplash(), []combat.SpellTarget{seTarget("a", nil), seTarget("b", nil), seTarget("c", nil)}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller()
			res, err := seRun(t, seCommand(c.spell, seReport(string(c.spell.ID()), 0, 0), c.targets...), roller)
			if !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) || !strings.Contains(err.Error(), "character level") {
				t.Errorf("Run = %#v, %v; want an invalid-command error naming the character level [%s FR-050]", res, err, spellEffectsSpec)
			}
			if len(roller.Requests()) != 0 {
				t.Errorf("requests = %v, want none", roller.Requests())
			}
		})
	}
}

// Requirement: spell-effects.md FR-015, FR-033 - a chosen Damage Type that is not one of the 13 is a command error for a spell that offers a choice, not a rejection reason.
// Path: unhappy (invalid input).
func TestMutgapCSpellRefusesAnUndefinedChosenDamageType(t *testing.T) {
	t.Parallel()
	cmd := seCommand(seChromaticOrb(), seReport("chromatic_orb", 1, 0), seWithFacts("a"))
	for _, bad := range []content.DamageType{-1, 99} {
		cmd.ChosenDamageType = bad
		if res, err := seRun(t, cmd, dicetest.NewFixedRoller()); !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
			t.Errorf("chosen %d: Run = %#v, %v; want an invalid-command error [%sFR-015]", bad, res, err, spellEffectsSpec)
		}
	}
}

// Requirement: spell-effects.md FR-031, FR-057 - a spell with several instances and no target is not an instance-assignment mismatch: there is nothing to assign.
// Path: unhappy (boundary).
func TestMutgapCSpellWithNoTargetsHasNoAssignmentMismatch(t *testing.T) {
	t.Parallel()
	res, err := seRun(t, seCommand(seMagicMissile(), seReport("magic_missile", 1, 0)), dicetest.NewFixedRoller(seFaces(1, 3)...))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rej, ok := res.(combat.SpellEffectsRejected); ok && slices.Contains(rej.Rejection().Reasons(), combat.InstanceAssignmentMismatch) {
		t.Errorf("reasons = %v, want no instanceAssignmentMismatch for a cast with no target [%sFR-031]", rej.Rejection().Reasons(), spellEffectsSpec)
	}
}
