package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/active-effects.md FR-040 - the causes have validating constructors; a nil or forged cause is an error matching ErrInvalidInput recorded on the span only; the cause copies its ids.
// Path: unhappy (invalid input).
func TestEndingCauseConstructorsAndMalformedCommands(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewConcentrationLost("", character.NewConcentrationSubjectSpell("x")); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero caster err = %v", err)
	}
	if _, err := combat.NewConcentrationLost("w", nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil subject err = %v", err)
	}
	if _, err := combat.NewDismissed("", false); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero dismisser err = %v", err)
	}
	for name, cause := range map[string]combat.EndingCause{"nil": nil, "zero ConcentrationLost": combat.ConcentrationLost{}, "zero Dismissed": combat.Dismissed{},
		"zero SubjectTookDamage": combat.SubjectTookDamage{}, "zero SubjectDied": combat.SubjectDied{}, "zero MoveEffectSubject": combat.MoveEffectSubject{}} {
		res, err := combat.NewEndActiveEffects().Run(context.Background(), combat.EndActiveEffectsCommand{Cause: cause})
		if res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: = %v, %v, want an ErrInvalidInput error", name, res, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := combat.NewEndActiveEffects().Run(ctx, combat.EndActiveEffectsCommand{Cause: combat.NewRemovedByEffect()}); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v, %v", res, err)
	}
}

// Requirement: doc/specs/active-effects.md FR-040 - the seven end reasons have snake_case names (the values of srd5e.combat.active_effects.end_reason) and nothing else is valid.
// Path: unhappy (boundary).
func TestEffectEndReasonNames(t *testing.T) {
	t.Parallel()
	want := []string{"expired", "repeat_save_succeeded", "concentration_lost", "dismissed", "subject_took_damage", "subject_died", "removed_by_effect"}
	for i, w := range want {
		r := combat.EffectEndReason(i + 1)
		if !r.Valid() || r.String() != w {
			t.Errorf("reason %d = %q valid %v, want %q", i+1, r.String(), r.Valid(), w)
		}
	}
	for _, bad := range []combat.EffectEndReason{0, 8, -1} {
		if bad.Valid() || bad.String() != "unknown" {
			t.Errorf("reason %d is valid or named", bad)
		}
	}
}
