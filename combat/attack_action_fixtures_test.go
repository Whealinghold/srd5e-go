package combat_test

import (
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// replacementOf builds a replace attack by the Breath Weapon feature, the shape a host records.
func replacementOf(tb testing.TB) combat.AttackReplacement {
	tb.Helper()
	r, err := combat.NewAttackReplacement(combat.AttackReplacementKindReplace, combat.ReplacementByFeature(content.MustFeatureID("breath_weapon")))
	if err != nil {
		tb.Fatal(err)
	}
	return r
}

// entryOf maps the table token P, O, U or R to an attack entry.
func entryOf(tb testing.TB, token string) combat.AttackEntry {
	tb.Helper()
	switch token {
	case "P":
		return combat.MadeAttack(combat.AttackMeansPactWeapon)
	case "O":
		return combat.MadeAttack(combat.AttackMeansOtherWeapon)
	case "U":
		return combat.MadeAttack(combat.AttackMeansUnarmedStrike)
	case "R":
		return combat.ReplacedAttack(replacementOf(tb))
	}
	tb.Fatalf("unknown table token %q", token)
	return combat.AttackEntry{}
}

func entriesOf(tb testing.TB, tokens string) []combat.AttackEntry {
	tb.Helper()
	var out []combat.AttackEntry
	for _, tok := range strings.Fields(tokens) {
		out = append(out, entryOf(tb, tok))
	}
	return out
}

func allowanceOf(tb testing.TB, general, pact int) combat.AttackAllowance {
	tb.Helper()
	a, err := combat.NewAttackAllowance(general, pact)
	if err != nil {
		tb.Fatal(err)
	}
	return a
}

// progressOf restores a progress for "hero" with the history tokens and no slot used.
func progressOf(tb testing.TB, general, pact int, history string) combat.AttackActionProgress {
	tb.Helper()
	p, err := combat.NewAttackActionProgress(gid("hero"), allowanceOf(tb, general, pact), entriesOf(tb, history), nil)
	if err != nil {
		tb.Fatal(err)
	}
	return p
}

// recorded returns the progress and the one event of a recorded attack, or fails.
func recorded(tb testing.TB, out combat.AttackRecordOutcome) (combat.AttackActionProgress, []combat.AttackActionEvent) {
	tb.Helper()
	r, ok := out.(combat.AttackActionAttackRecorded)
	if !ok {
		tb.Fatalf("outcome = %#v, want recorded", out)
	}
	return r.Progress(), r.Events()
}

// changeRecorded returns the progress and events of a recorded weapon change, or fails.
func changeRecorded(tb testing.TB, out combat.WeaponChangeOutcome) (combat.AttackActionProgress, []combat.AttackActionEvent) {
	tb.Helper()
	r, ok := out.(combat.AttackActionWeaponChangeRecorded)
	if !ok {
		tb.Fatalf("outcome = %#v, want recorded", out)
	}
	return r.Progress(), r.Events()
}
