package combat_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// heldBane starts n Bane effects on the creature "t", each with its own id and no Concentration.
func heldBane(t *testing.T, n int) combat.ActiveEffects {
	t.Helper()
	var reg combat.ActiveEffects
	for i := range n {
		reg = aeStart(t, reg, bane2(fmt.Sprintf("e%03d", i), "t"))
	}
	return reg
}

// Requirement: doc/specs/active-effects.md plan P10 - a creature holds at most 64 active effects; the 65th start is an invalid rejection with the limit_exceeded reason, the registry unchanged and no event.
// Path: per case.
func TestStartActiveEffectCapPerCreature(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		held                    int
		subject                 string
		wantStarted             bool
	}{
		{"63 held, a 64th starts", "P10 at the cap", "unhappy (boundary)", 63, "t", true},
		{"64 held, a 65th is refused", "P10 cap plus one", "unhappy (boundary)", 64, "t", false},
		{"64 held on t, another creature still starts", "P10 per creature", "unhappy (boundary)", 64, "u", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			reg := heldBane(t, c.held)
			cmd := bane2("new", c.subject).command(t, reg)
			res, err := combat.NewStartActiveEffect().Run(context.Background(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			if _, started := res.(combat.StartActiveEffectStarted); started != c.wantStarted {
				t.Fatalf("result = %#v, want started %v [%s]", res, c.wantStarted, c.requirement)
			}
			if c.wantStarted {
				return
			}
			rej := res.(combat.StartActiveEffectRejected).Rejection()
			if got := rej.Reasons(); !slices.Equal(got, []combat.ActiveEffectRejectionKind{combat.EffectRejectLimitExceeded}) || rej.Problems()[0].Effect != "new" {
				t.Errorf("reasons = %v, problems %v [%s]", got, rej.Problems(), c.requirement)
			}
			if reg.Len() != c.held {
				t.Errorf("registry changed: len %d", reg.Len())
			}
		})
	}
}

// Requirement: doc/specs/active-effects.md plan P10, FR-060 - moving an effect onto a creature that already holds 64 is an invalid limit_exceeded rejection, and onto one that holds 63 is accepted.
// Path: per case.
func TestMoveEffectSubjectCapPerCreature(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement string
		held              int
		ok                bool
	}{{"63 held, the move makes 64", "P10 at the cap", 63, true}, {"64 held, the move makes 65", "P10 cap plus one", 64, false}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			reg := aeStart(t, heldBane(t, c.held), mark("m", "ranger", "x"))
			res := aeEnd(t, reg, aeMust(combat.NewMoveEffectSubject("m", aeCreature2("t"))))
			_, moved := res.(combat.EndActiveEffectsMoved)
			if moved != c.ok {
				t.Fatalf("result = %#v, want moved %v [%s]", res, c.ok, c.requirement)
			}
			if !c.ok {
				if got := res.(combat.EndActiveEffectsRejected).Rejection().Reasons(); !slices.Equal(got, []combat.ActiveEffectRejectionKind{combat.EffectRejectLimitExceeded}) {
					t.Errorf("reasons = %v [%s]", got, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/active-effects.md plan P10 - NewActiveEffects rejects a registry that gives one creature more than 64 effects, with an ErrInvalidInput error, and accepts exactly 64.
// Path: per case.
func TestNewActiveEffectsCapPerCreature(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement string
		n                 int
		ok                bool
	}{{"64 effects", "P10 at the cap", 64, true}, {"65 effects", "P10 cap plus one", 65, false}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			reg := heldBane(t, 64)
			effects := reg.Effects()
			if c.n == 65 {
				effects = append(effects, aeEffect(t, aeStart(t, combat.ActiveEffects{}, bane2("extra", "t")), "extra"))
			}
			got, err := combat.NewActiveEffects(effects...)
			if c.ok != (err == nil) || (!c.ok && (!errors.Is(err, srderr.ErrInvalidInput) || got.Len() != 0)) {
				t.Errorf("NewActiveEffects(%d) = len %d, %v [%s]", c.n, got.Len(), err, c.requirement)
			}
		})
	}
}
