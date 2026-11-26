package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/active-effects.md FR-056 - a dice port failure or a panic is returned as a *srderr.PortError wrapped once with no partial result, and errors.Is reaches the cause; a cancelled context returns ctx.Err() before anything.
// Path: unhappy (collaborator failure).
func TestAdvanceRollerFailures(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, aeSpec{id: "hp", spell: "hold_person", caster: "wiz", subject: aeCreature2("orc"), template: aeHoldPerson(t, 14)})
	cmd := combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"), Saves: aeSaves("orc", ability.Wisdom, aeRoll())}
	cause := errors.New("roller down")
	cases := map[string]struct {
		roller combat.Roller
		is     error
	}{
		"fails":  {dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }), cause},
		"panics": {dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("boom") }), srderr.ErrPortPanic},
	}
	for name, c := range cases {
		uc, _ := combat.NewAdvanceActiveEffects(c.roller)
		res, err := uc.Run(context.Background(), cmd)
		var pe *srderr.PortError
		if res != nil || !errors.As(err, &pe) || !errors.Is(err, c.is) || pe.Port != "dice_roller" {
			t.Errorf("%s: = %v, %v, want a nil result and a dice_roller PortError reaching the cause", name, res, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	uc, _ := combat.NewAdvanceActiveEffects(dicetest.NewMinimumRoller())
	if res, err := uc.Run(ctx, cmd); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v, %v", res, err)
	}
	if _, err := combat.NewAdvanceActiveEffects(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller err = %v", err)
	}
	var zero combat.AdvanceActiveEffects
	if res, err := zero.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case = %v, %v", res, err)
	}
	for _, tick := range []combat.EffectClockTick{nil, combat.TimeElapsed{}, combat.TurnStartReached{}, combat.TurnEndReached{}, combat.CombatantLeft{}} {
		bad := cmd
		bad.Tick = tick
		if res, err := uc.Run(context.Background(), bad); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("tick %#v = %v, %v, want an ErrInvalidInput error", tick, res, err)
		}
	}
}

// Requirement: doc/specs/active-effects.md FR-020 (stacked conditions and Advantage and Disadvantage together) - a repeated save by a RollAttempt that carries both Advantage and Disadvantage rolls one d20 (they cancel), and a mixed registry ticks each effect independently.
// Path: unhappy (rule interaction).
func TestRepeatSaveWithAdvantageAndDisadvantage(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, aeSpec{id: "hp", spell: "hold_person", caster: "wiz", subject: aeCreature2("orc"), template: aeHoldPerson(t, 14)}, roundsSpec("b", 3, ""))
	attempt := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(2)), Circumstances: aeBothModes()}
	roller := dicetest.NewFixedRoller(12)
	uc, _ := combat.NewAdvanceActiveEffects(roller)
	res, err := uc.Run(context.Background(), combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"), Saves: aeSaves("orc", ability.Wisdom, attempt)})
	if err != nil || len(roller.Requests()) != 1 || roller.Requests()[0].Count() != 1 {
		t.Fatalf("dice = %v, %v, want one d20", roller.Requests(), err)
	}
	if got := res.(combat.AdvanceActiveEffectsApplied).Effects(); got.Len() != 1 || !slices.Equal(aeIDs(got), []string{"b"}) {
		t.Errorf("12 + 2 = 14 meets the DC, so only the other effect remains: %v", aeIDs(got))
	}
}
