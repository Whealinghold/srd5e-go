package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/active-effects.md FR-024 - an effect that lasts until a turn boundary expires only at that boundary of its owner's turn: the owner's id sorts between two other combatants' ids, and the other boundary of the owner's turn leaves it alone.
// Path: unhappy (boundary).
func TestUntilBoundaryIgnoresOtherCombatantsAndBoundaries(t *testing.T) {
	t.Parallel()
	start := aeSeq(t, untilSpec("u", "m", content.BoundaryStart, ""))
	end := aeSeq(t, untilSpec("u", "m", content.BoundaryEnd, ""))
	for _, who := range []string{"a", "z"} {
		for name, c := range map[string]struct {
			reg  combat.ActiveEffects
			tick combat.EffectClockTick
		}{
			"start boundary, start of " + who: {start, aeStartTick(t, who)},
			"start boundary, end of " + who:   {start, aeEndTick(t, who)},
			"end boundary, start of " + who:   {end, aeStartTick(t, who)},
			"end boundary, end of " + who:     {end, aeEndTick(t, who)},
		} {
			if aeAdvance(t, c.reg, c.tick, "").Effects().Len() != 1 {
				t.Errorf("%s: an effect owned by m expired [FR-024]", name)
			}
		}
	}
	if aeAdvance(t, end, aeStartTick(t, "m"), "").Effects().Len() != 1 {
		t.Error("an effect until the end of m's turn expired at the start of m's turn")
	}
	if aeAdvance(t, start, aeEndTick(t, "m"), "").Effects().Len() != 1 {
		t.Error("an effect until the start of m's turn expired at the end of m's turn")
	}
	if aeAdvance(t, start, aeStartTick(t, "m"), "").Effects().Len() != 0 || aeAdvance(t, end, aeEndTick(t, "m"), "").Effects().Len() != 0 {
		t.Error("an effect did not expire at its own boundary")
	}
}

// Requirement: doc/specs/active-effects.md FR-027, FR-062 - a trigger of the caster's turn fires only on the caster's turn, and a combatant that leaves changes only the effects anchored to or owned by that combatant; ids on both sides of the caster's id are other combatants.
// Path: unhappy (boundary).
func TestCasterTriggerAndDepartureOfOthers(t *testing.T) {
	t.Parallel()
	blast := aeSeq(t, aeSpec{id: "bf", spell: "delayed_blast_fireball", caster: "m", concentration: true, subject: aePlace(t, "bead"), template: aeDelayedBlast(t), active: "m"})
	for _, who := range []string{"a", "z"} {
		if got := aeAdvance(t, blast, aeEndTick(t, who), "").Events(); len(got) != 0 {
			t.Errorf("the end of %s's turn fired the trigger of caster m: %v", who, aeKinds(got))
		}
	}
	if got := aeAdvance(t, blast, aeEndTick(t, "m"), "").Events(); len(got) != 1 {
		t.Errorf("the end of the caster's turn fired %d events, want 1", len(got))
	}
	anchored := aeSeq(t, roundsSpec("e", 10, "m"), untilSpec("u", "m", content.BoundaryEnd, "x"))
	for _, who := range []string{"a", "z"} {
		left := aeMust(combat.NewCombatantLeft(combat.MustCombatantID(who)))
		after := aeAdvance(t, anchored, left, "").Effects()
		if !aeEffect(t, after, "e").AnchoredTo("m") || !after.Equal(anchored) {
			t.Errorf("the departure of %s changed an effect of m [FR-062]", who)
		}
	}
}

// Requirement: doc/specs/active-effects.md FR-028, FR-063 - an order that names an id of no effect is rejected invalidOrder even when that id sorts before every effect that fires.
// Path: unhappy (invalid input).
func TestOrderNamingAnIdBelowEveryEffect(t *testing.T) {
	t.Parallel()
	mk := func(id, spell string) aeSpec {
		return aeSpec{id: id, spell: spell, caster: "wiz", subject: aeCreature2("orc"), template: aeHoldPerson(t, 10)}
	}
	reg := aeSeq(t, mk("b", "hold_one"), mk("c", "hold_two"))
	res := aeAdvanceCmd(t, combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"),
		Order: []combat.ActiveEffectID{"b", "c", "a"}, Saves: aeSaves("orc", ability.Wisdom, aeRoll())}, 1)
	rej, ok := res.(combat.AdvanceActiveEffectsRejected)
	if !ok || !slices.Equal(rej.Rejection().Reasons(), []combat.ActiveEffectRejectionKind{combat.EffectRejectInvalidOrder}) {
		t.Errorf("order naming the unknown id a = %#v, want invalidOrder", res)
	}
}

// Requirement: doc/specs/active-effects.md FR-034, FR-056 - a dice port failure while a stage gate rolls is returned as a *srderr.PortError with no partial result.
// Path: unhappy (collaborator failure).
func TestStageGateRollerFailure(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, aeSpec{id: "sl", spell: "sleep", caster: "wiz", subject: aeCreature2("orc"), template: aeSleep(t, "orc", 13), active: "wiz",
		conditions: []content.ConditionKind{content.ConditionIncapacitated}})
	cause := errors.New("roller down")
	uc, _ := combat.NewAdvanceActiveEffects(dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }))
	res, err := uc.Run(context.Background(), combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"), Active: "wiz",
		Saves: aeSaves("orc", ability.Wisdom, aeRoll())})
	var pe *srderr.PortError
	if res != nil || !errors.As(err, &pe) || !errors.Is(err, cause) {
		t.Errorf("stage gate with a failing roller = %v, %v, want a nil result and the port error", res, err)
	}
}
