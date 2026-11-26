package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// maCapMonster is a monster whose Multiattack allows 100 Claw attacks, one additional use of Roar and replacing an attack with Spit
// any number of times.
func maCapMonster(tb testing.TB) content.Monster {
	tb.Helper()
	rules := []content.ReplacementRule{sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceAnyNumber, sbOption("Spit"))}
	ma := sbMultiattack(sbVariant("", sbGroups(sbGroup(100, "Claw")), sbExtras(sbExtra("Roar")), rules...))
	return sbMonster(tb, &ma, sbPlain("Claw", "Spit", "Roar"), nil)
}

// maCapStart starts the Multiattack of m and returns its progress and the limited-use state.
func maCapStart(t *testing.T, m content.Monster) (combat.MultiattackProgress, combat.LimitedUseState) {
	t.Helper()
	state := luStart(t, m, combat.OutsideLair)
	out, err := combat.StartMultiattack(m, sbSelf(), state)
	started, ok := out.(combat.MultiattackStarted)
	if err != nil || !ok {
		t.Fatalf("StartMultiattack = %#v, %v", out, err)
	}
	return started.Progress(), state
}

// maCapRecorded returns the progress after a step that must have been recorded.
func maCapRecorded(t *testing.T, out combat.MultiattackStepOutcome, err error) combat.MultiattackProgress {
	t.Helper()
	rec, ok := out.(combat.MultiattackStepRecorded)
	if err != nil || !ok {
		t.Fatalf("step = %#v, %v, want it recorded", out, err)
	}
	return rec.Transition.Progress()
}

// Requirement: doc/specs/monster-stat-block-actions.md Limits (plan P10) - the 64 steps of a progress are its attacks, additional uses
// and replacements added together: 62 attacks, 1 additional use and 1 replacement fill it, and the next step of any kind is a
// LimitExceededError for the steps.
// Path: unhappy (boundary).
func TestMultiattackStepCapCountsEveryKindOfStep(t *testing.T) {
	t.Parallel()
	m := maCapMonster(t)
	p, state := maCapStart(t, m)
	claw, roar := sbAction(t, m, "Claw"), sbAction(t, m, "Roar")
	for range 62 {
		out, err := p.RecordAttack(claw, state)
		p = maCapRecorded(t, out, err)
	}
	out, err := p.RecordAdditionalUse(roar, state)
	p = maCapRecorded(t, out, err)
	out, err = p.RecordReplacement(sbOption("Spit"), claw, state)
	p = maCapRecorded(t, out, err)
	if n := len(p.Attacks()); n != 62 {
		t.Fatalf("attacks = %d, want 62", n)
	}
	if _, err := p.RecordAttack(claw, state); err == nil {
		t.Error("the 65th step (an attack): error = nil, want LimitExceededError")
	} else {
		wantLimit(t, err, "multiattack steps", 64)
	}
	if _, err := p.RecordReplacement(sbOption("Spit"), claw, state); err == nil {
		t.Error("the 65th step (a replacement): error = nil, want LimitExceededError")
	} else {
		wantLimit(t, err, "multiattack steps", 64)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md Limits (plan P10) - a progress that holds 64 replacements has 64 steps, so a
// step that is not a replacement is refused for the steps, not for the replacements.
// Path: unhappy (boundary).
func TestMultiattackAttackAfter64ReplacementsExceedsTheSteps(t *testing.T) {
	t.Parallel()
	m := maCapMonster(t)
	p, state := maCapStart(t, m)
	claw, roar := sbAction(t, m, "Claw"), sbAction(t, m, "Roar")
	for range 64 {
		out, err := p.RecordReplacement(sbOption("Spit"), claw, state)
		p = maCapRecorded(t, out, err)
	}
	if _, err := p.RecordAttack(claw, state); err == nil {
		t.Error("an attack after 64 replacements: error = nil, want LimitExceededError")
	} else {
		wantLimit(t, err, "multiattack steps", 64)
	}
	if _, err := p.RecordAdditionalUse(roar, state); err == nil {
		t.Error("an additional use after 64 replacements: error = nil, want LimitExceededError")
	} else {
		wantLimit(t, err, "multiattack steps", 64)
	}
}
