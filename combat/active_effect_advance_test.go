package combat_test

import (
	"context"
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// roundsSpec is an effect of the spell bless lasting rounds, cast by caster on t, started while active has the turn.
func roundsSpec(id string, rounds int, active string) aeSpec {
	return aeSpec{id: id, spell: "bless", caster: "caster", subject: aeCreature2("t"), template: aeSimpleNoT(rounds, 0, combat.CombiningOnTarget), active: active}
}

func untilSpec(id, owner string, b content.TurnBoundary, active string) aeSpec {
	tpl := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{
		aeMust(combat.NewEffectStage(combat.EffectStageParams{Duration: aeMust(combat.NewEffectLastsUntilTurnBoundary(combat.MustCombatantID(owner), b))}))}}))
	return aeSpec{id: id, spell: "ray_of_sickness", caster: "caster", subject: aeCreature2("t"), template: tpl, active: active}
}

func remaining(t *testing.T, reg combat.ActiveEffects, id string) int {
	t.Helper()
	return aeEffect(t, reg, id).RemainingRounds()
}

// Requirement: doc/specs/active-effects.md FR-020, table T rows T1 T2 T3 (SRD 5.2.1, "Duration", p. 106; "The Order of Combat", p. 13) - an anchored effect of N rounds loses one round at each start of its anchor's turn after the one it started in and expires at the start of the Nth later turn, so the anchor benefits on N turns; another combatant's turn does not count.
// Path: per case.
func TestRoundsCountTheAnchorsTurnStarts(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, roundsSpec("e", 10, "a"))
	for round := 2; round <= 10; round++ {
		reg = aeAdvance(t, reg, aeStartTick(t, "b"), "b").Effects()
		reg = aeAdvance(t, reg, aeStartTick(t, "a"), "a").Effects()
		if got := remaining(t, reg, "e"); got != 11-round {
			t.Fatalf("T1 round %d: remaining %d, want %d", round, got, 11-round)
		}
	}
	last := aeAdvance(t, reg, aeStartTick(t, "a"), "a")
	if last.Effects().Len() != 0 {
		t.Errorf("T1 round 11 start: the effect did not expire [FR-020]")
	}
	byB := aeSeq(t, roundsSpec("e", 3, "b"))
	byB = aeAdvance(t, byB, aeStartTick(t, "a"), "a").Effects()
	byB = aeAdvance(t, byB, aeStartTick(t, "b"), "b").Effects()
	if got := remaining(t, byB, "e"); got != 2 {
		t.Errorf("T2 anchored to B: remaining %d, want 2, counting only B's turn starts", got)
	}
	one := aeSeq(t, roundsSpec("e", 1, "a"))
	if aeAdvance(t, one, aeStartTick(t, "b"), "b").Effects().Len() != 1 {
		t.Error("T3 a one-round effect expired at another combatant's turn start")
	}
	if aeAdvance(t, one, aeStartTick(t, "a"), "a").Effects().Len() != 0 {
		t.Error("T3 a one-round effect did not expire at its anchor's next turn start")
	}
}

// Requirement: doc/specs/active-effects.md FR-020, FR-021, table T rows T4 T5 T6 T7 T11 - TimeElapsed subtracts in or out of combat and expires an effect at 0 or less (10 and 11 rounds on 10 both expire; 9 leaves 1); an unanchored effect loses a round at each round start and ignores turn starts; an indefinite effect never expires.
// Path: per case.
func TestTimeElapsedAndRoundStarts(t *testing.T) {
	t.Parallel()
	anchored := aeSeq(t, roundsSpec("e", 10, "a"))
	free := aeSeq(t, roundsSpec("e", 10, ""))
	cases := []struct {
		name string
		reg  combat.ActiveEffects
		tick combat.EffectClockTick
		want int // remaining rounds, or -1 for expired
	}{
		{"T4 three rounds elapse", anchored, aeElapsed(t, 3), 7},
		{"T5 nine rounds elapse", anchored, aeElapsed(t, 9), 1},
		{"T5 ten rounds elapse", anchored, aeElapsed(t, 10), -1},
		{"T5 eleven rounds elapse", anchored, aeElapsed(t, 11), -1},
		{"T6 a turn start leaves an unanchored effect alone", free, aeStartTick(t, "a"), 10},
		{"T7 a round start counts an unanchored effect", free, combat.RoundStartReached{}, 9},
		{"a round start leaves an anchored effect alone", anchored, combat.RoundStartReached{}, 10},
		{"a turn end leaves an anchored effect alone", anchored, aeEndTick(t, "a"), 10},
	}
	for _, c := range cases {
		reg := aeAdvance(t, c.reg, c.tick, "").Effects()
		switch {
		case c.want < 0 && reg.Len() != 0, c.want >= 0 && (reg.Len() != 1 || remaining(t, reg, "e") != c.want):
			t.Errorf("%s: registry %v remaining %d, want %d [FR-020/FR-021]", c.name, aeIDs(reg), c.want, c.want)
		}
	}
	forever := aeSeq(t, aeSpec{id: "f", spell: "bless", caster: "caster", subject: aeCreature2("t"), template: aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{
		Stages: []combat.EffectStage{aeMust(combat.NewEffectStage(combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}}))}}))})
	for _, tick := range []combat.EffectClockTick{aeElapsed(t, 100000), aeElapsed(t, math.MaxInt), combat.RoundStartReached{}, aeStartTick(t, "a"), aeEndTick(t, "t")} {
		if aeAdvance(t, forever, tick, "").Effects().Len() != 1 {
			t.Errorf("T11 an indefinite effect ended at %#v [FR-023]", tick)
		}
	}
	if _, err := combat.NewTimeElapsed(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewTimeElapsed(0) err = %v", err)
	}
	if aeElapsed(t, 4).(combat.TimeElapsed).Rounds() != 4 {
		t.Error("TimeElapsed.Rounds")
	}
}

// Requirement: doc/specs/active-effects.md FR-022, table T row T8 - EndOfCombatReached clears every anchor and changes no remaining rounds; a later combat then ticks the effect by round starts, and an effect that lasts until a turn boundary is unchanged.
// Path: unhappy (state error).
func TestCombatEnded(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, roundsSpec("e", 10, "a"), untilSpec("u", "t", content.BoundaryEnd, "t"))
	for range 4 {
		reg = aeAdvance(t, reg, aeStartTick(t, "a"), "a").Effects()
	}
	if remaining(t, reg, "e") != 6 {
		t.Fatalf("remaining %d, want 6", remaining(t, reg, "e"))
	}
	ended := aeAdvance(t, reg, combat.EndOfCombatReached{}, "")
	if len(ended.Events()) != 0 || remaining(t, ended.Effects(), "e") != 6 || aeEffect(t, ended.Effects(), "e").Anchored() {
		t.Errorf("T8 after EndOfCombatReached: remaining %d anchored %v, want 6 and false", remaining(t, ended.Effects(), "e"), aeEffect(t, ended.Effects(), "e").Anchored())
	}
	if !aeEffect(t, ended.Effects(), "u").SkipsCurrentTurnEnd() {
		t.Error("EndOfCombatReached changed an until-boundary effect")
	}
	next := aeAdvance(t, ended.Effects(), combat.RoundStartReached{}, "").Effects()
	if remaining(t, next, "e") != 5 {
		t.Errorf("after a round start remaining %d, want 5", remaining(t, next, "e"))
	}
}

// Requirement: doc/specs/active-effects.md FR-024, table T rows T9 T10 (SRD 5.2.1, "Ray of Sickness", p. 158; "Acid Arrow", p. 107) - an effect that lasts until a turn boundary expires at the first matching tick of its owner, except that one started on its owner's turn with boundary end skips that turn's end; a start boundary does not skip; outside combat the first matching tick counts.
// Path: per case.
func TestUntilTurnBoundary(t *testing.T) {
	t.Parallel()
	skip := aeSeq(t, untilSpec("u", "a", content.BoundaryEnd, "a"))
	if !aeEffect(t, skip, "u").SkipsCurrentTurnEnd() {
		t.Fatal("T9 an effect started on its owner's turn does not skip that turn's end")
	}
	if aeAdvance(t, skip, aeEndTick(t, "b"), "b").Effects().Len() != 1 {
		t.Error("another combatant's turn end expired the effect")
	}
	first := aeAdvance(t, skip, aeEndTick(t, "a"), "a").Effects()
	if first.Len() != 1 || aeEffect(t, first, "u").SkipsCurrentTurnEnd() {
		t.Error("T9 the current turn end did not clear the skip and keep the effect")
	}
	if aeAdvance(t, first, aeEndTick(t, "a"), "a").Effects().Len() != 0 {
		t.Error("T9 the next turn end did not expire the effect")
	}
	other := aeSeq(t, untilSpec("u", "b", content.BoundaryEnd, "a"))
	if aeAdvance(t, other, aeEndTick(t, "b"), "b").Effects().Len() != 0 {
		t.Error("T10 an effect until B's turn end did not expire at B's end")
	}
	start := aeSeq(t, untilSpec("u", "a", content.BoundaryStart, "a"))
	if aeEffect(t, start, "u").SkipsCurrentTurnEnd() || aeAdvance(t, start, aeEndTick(t, "a"), "a").Effects().Len() != 1 ||
		aeAdvance(t, start, aeStartTick(t, "a"), "a").Effects().Len() != 0 {
		t.Error("an effect until the start of its owner's turn skips or expires wrongly")
	}
	outside := aeSeq(t, untilSpec("u", "a", content.BoundaryEnd, ""))
	if aeAdvance(t, outside, aeEndTick(t, "a"), "").Effects().Len() != 0 {
		t.Error("started outside combat, the first matching tick did not count")
	}
}

// Requirement: doc/specs/active-effects.md FR-062, table T rows T12 T13 T17 - CombatantLeft clears the anchor of an effect anchored to the combatant, keeping its rounds; an effect until the departed owner's boundary becomes one round with no anchor that ends at the next round start; a combatant that anchors or owns nothing changes nothing.
// Path: unhappy (state error).
func TestCombatantLeft(t *testing.T) {
	t.Parallel()
	left := func(id string) combat.EffectClockTick {
		return aeMust(combat.NewCombatantLeft(combat.MustCombatantID(id)))
	}
	reg := aeSeq(t, roundsSpec("e", 10, "a"))
	reg = aeAdvance(t, reg, aeStartTick(t, "a"), "a").Effects()
	reg = aeAdvance(t, reg, aeStartTick(t, "a"), "a").Effects()
	gone := aeAdvance(t, reg, left("a"), "").Effects()
	if aeEffect(t, gone, "e").Anchored() || remaining(t, gone, "e") != 8 {
		t.Fatalf("T12 after CombatantLeft remaining %d, want 8 and no anchor", remaining(t, gone, "e"))
	}
	if got := remaining(t, aeAdvance(t, gone, combat.RoundStartReached{}, "").Effects(), "e"); got != 7 {
		t.Errorf("T12 next round start remaining %d, want 7", got)
	}
	if aeAdvance(t, reg, aeStartTick(t, "a"), "a").Effects().Len() != 1 {
		t.Fatal("setup")
	}
	owned := aeSeq(t, untilSpec("u", "b", content.BoundaryEnd, "a"))
	after := aeAdvance(t, owned, left("b"), "").Effects()
	e := aeEffect(t, after, "u")
	if e.Duration() != aeRounds(t, 1) || e.Anchored() || e.SkipsCurrentTurnEnd() {
		t.Errorf("T13 duration %v anchored %v, want one round and no anchor", e.Duration(), e.Anchored())
	}
	if aeAdvance(t, after, aeEndTick(t, "b"), "").Effects().Len() != 1 {
		t.Error("T13 the departed owner's turn end still expires the effect")
	}
	if aeAdvance(t, after, combat.RoundStartReached{}, "").Effects().Len() != 0 {
		t.Error("T13 the effect did not expire at the next round start")
	}
	same := aeAdvance(t, reg, left("nobody"), "")
	if !same.Effects().Equal(reg) || len(same.Events()) != 0 {
		t.Error("T17 a combatant that anchors nothing changed the registry")
	}
	if _, err := combat.NewCombatantLeft(""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewCombatantLeft(zero) err = %v", err)
	}
}

// Requirement: doc/specs/active-effects.md FR-026 - a tick that matches no effect leaves the registry equal to the input with no events and an empty plan; so does an empty registry.
// Path: unhappy (empty and minimal).
func TestTickMatchingNothing(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, roundsSpec("e", 10, "a"))
	for name, in := range map[string]combat.ActiveEffects{"a registry": reg, "an empty registry": {}} {
		out := aeAdvance(t, in, aeStartTick(t, "z"), "z")
		if !out.Effects().Equal(in) || len(out.Events()) != 0 || len(out.ConditionsToApply())+len(out.ConditionsToRemove())+len(out.DamagePlans()) != 0 {
			t.Errorf("%s: a tick for nobody changed something [FR-026]", name)
		}
	}
}

// Requirement: doc/specs/active-effects.md FR-061, table T rows T15 T16 (SRD 5.2.1, "Sleep", p. 163) - the overall clock ends the effect even mid-stage and runs the onEnd actions; advancing a stage never extends the effect past it: a remainder stage continues the overall clock instead of starting a fresh one.
// Path: unhappy (boundary).
func TestOverallClock(t *testing.T) {
	t.Parallel()
	boom := aeDeal(t, "1d4", content.DamageFire, combat.EffectAudienceSubject)
	tpl := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Overall: aeRounds(t, 4), Stages: []combat.EffectStage{
		aeMust(combat.NewEffectStage(combat.EffectStageParams{Duration: aeRounds(t, 10), OnEnd: []combat.EffectAction{boom}}))}}))
	reg := aeSeq(t, aeSpec{id: "o", spell: "bless", caster: "caster", subject: aeCreature2("t"), template: tpl, active: "a"})
	for range 3 {
		reg = aeAdvance(t, reg, aeStartTick(t, "a"), "a").Effects()
	}
	if left, ok := aeEffect(t, reg, "o").OverallRemainingRounds(); !ok || left != 1 || remaining(t, reg, "o") != 7 {
		t.Fatalf("after three turns overall %d (counting %v) stage %d, want 1 and 7", left, ok, remaining(t, reg, "o"))
	}
	out := aeAdvance(t, reg, aeStartTick(t, "a"), "a")
	if out.Effects().Len() != 0 || len(out.DamagePlans()) != 1 {
		t.Errorf("the overall clock did not end the effect mid-stage with its onEnd: %d effects, %d plans", out.Effects().Len(), len(out.DamagePlans()))
	}
	if ev := out.Events(); len(ev) != 1 || ev[0].(combat.ActiveEffectEnded).Reason() != combat.EffectExpired {
		t.Errorf("events = %v, want one expired", aeKinds(ev))
	}

	sleep := aeSeq(t, aeSpec{id: "sl", spell: "sleep", caster: "wiz", subject: aeCreature2("t"), template: aeSleep(t, "t", 13), active: "wiz",
		conditions: []content.ConditionKind{content.ConditionIncapacitated}})
	sleep = aeAdvance(t, sleep, aeStartTick(t, "wiz"), "wiz").Effects()
	sleep = aeAdvance(t, sleep, aeStartTick(t, "wiz"), "wiz").Effects()
	moved := aeAdvanceCmd(t, combat.AdvanceActiveEffectsCommand{Effects: sleep, Tick: aeEndTick(t, "t"), Active: "t", Saves: aeSaves("t", ability.Wisdom, aeRoll())}, 1).(combat.AdvanceActiveEffectsApplied)
	e := aeEffect(t, moved.Effects(), "sl")
	left, _ := e.OverallRemainingRounds()
	if e.Stage() != 1 || left != 8 || e.RemainingRounds() != 0 || e.Duration() != (combat.EffectLastsRemainder{}) {
		t.Errorf("T16 stage %d overall %d stage clock %d, want stage 2 with 8 left and no clock of its own", e.Stage(), left, e.RemainingRounds())
	}
	after := aeAdvance(t, moved.Effects(), aeElapsed(t, 7), "").Effects()
	if aeAdvance(t, after, aeElapsed(t, 1), "").Effects().Len() != 0 || after.Len() != 1 {
		t.Error("T16 the remainder stage did not end exactly with the overall clock")
	}
}

// Requirement: doc/specs/active-effects.md FR-021, FR-034, table T row T15 - TimeElapsed that overshoots a stage clock runs the gate once and the next stage starts with its full clock, no overshoot carried.
// Path: unhappy (boundary).
func TestOvershootDoesNotCarry(t *testing.T) {
	t.Parallel()
	gate := aeRepeatSave(t, ability.Wisdom, 10)
	tpl := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{
		aeMust(combat.NewEffectStage(combat.EffectStageParams{Duration: aeRounds(t, 10), Gate: gate})),
		aeMust(combat.NewEffectStage(combat.EffectStageParams{Duration: aeRounds(t, 10)}))}}))
	reg := aeSeq(t, aeSpec{id: "g", spell: "bless", caster: "caster", subject: aeCreature2("t"), template: tpl, active: "a"})
	roller := dicetest.NewFixedRoller(1, 20)
	uc, _ := combat.NewAdvanceActiveEffects(roller)
	res, err := uc.Run(context.Background(), combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeElapsed(t, 11), Saves: aeSaves("t", ability.Wisdom, aeRoll())})
	if err != nil {
		t.Fatal(err)
	}
	e := aeEffect(t, res.(combat.AdvanceActiveEffectsApplied).Effects(), "g")
	if e.Stage() != 1 || e.RemainingRounds() != 10 || len(roller.Requests()) != 1 {
		t.Errorf("stage %d remaining %d dice %d, want stage 1 with 10 left and one gate roll", e.Stage(), e.RemainingRounds(), len(roller.Requests()))
	}
}

// Requirement: doc/specs/active-effects.md FR-029, FR-030, FR-032, table T (SRD 5.2.1, "Hold Person", p. 141) - at the end of the held creature's turns it repeats the saving throw against the DC fixed in the template, even while Paralyzed by the effect; a success ends the effect (reason repeatSaveSucceeded) and plans the condition's removal; a failure leaves it unchanged; each save is one ActiveEffectRepeatSaveMade.
// Path: per case.
func TestRepeatSave(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, aeSpec{id: "hp", spell: "hold_person", caster: "wiz", concentration: true, subject: aeCreature2("orc"), template: aeHoldPerson(t, 14),
		conditions: []content.ConditionKind{content.ConditionParalyzed}, active: "wiz"})
	cases := []struct {
		name  string
		face  int
		saved bool
	}{{"14 meets the DC", 14, true}, {"13 is one short", 13, false}, {"natural 20", 20, true}, {"natural 1", 1, false}}
	for _, c := range cases {
		out := aeAdvanceCmd(t, combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"), Saves: aeSaves("orc", ability.Wisdom, aeRoll())}, c.face).(combat.AdvanceActiveEffectsApplied)
		ev := out.Events()
		save, isSave := ev[0].(combat.ActiveEffectRepeatSaveMade)
		if !isSave || save.Ability() != ability.Wisdom || save.SaveKind() != combat.EffectSaveRepeat || (save.Outcome() == d20.Success) != c.saved {
			t.Errorf("%s: first event %#v, want a repeat save with saved=%v", c.name, ev[0], c.saved)
			continue
		}
		combattest.CheckEvent(t, save, "active_effect_repeat_save_made", "orc", "wiz")
		if c.saved {
			want := []combat.EffectConditionPlan{{Effect: "hp", Target: "orc", Kind: content.ConditionParalyzed}}
			if out.Effects().Len() != 0 || !slices.Equal(out.ConditionsToRemove(), want) || len(ev) != 2 ||
				ev[1].(combat.ActiveEffectEnded).Reason() != combat.EffectRepeatSaveSucceeded {
				t.Errorf("%s: effects %d plans %v events %v, want the effect ended by the save", c.name, out.Effects().Len(), out.ConditionsToRemove(), aeKinds(ev))
			}
		} else if !out.Effects().Equal(reg) || len(ev) != 1 || len(out.ConditionsToRemove()) != 0 {
			t.Errorf("%s: a failed save changed the effect or planned a removal", c.name)
		}
	}
	other := aeAdvance(t, reg, aeEndTick(t, "wiz"), "wiz")
	if !other.Effects().Equal(reg) || len(other.Events()) != 0 {
		t.Error("the caster's turn end triggered the held creature's save")
	}
	startTick := aeAdvance(t, reg, aeStartTick(t, "orc"), "orc")
	if len(startTick.Events()) != 0 {
		t.Error("a turn start triggered a save due at the turn end")
	}
}

// Requirement: doc/specs/active-effects.md FR-031, FR-063 - a ChooseToFail or AutomaticFailure attempt is a failed repeat save without a d20; a firing save with no attempt is rejected missingSaveAttempt listing every missing pair before any die is rolled.
// Path: per case.
func TestRepeatSaveAttempts(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t,
		aeSpec{id: "hp1", spell: "hold_person", caster: "wiz", concentration: true, subject: aeCreature2("orc"), template: aeHoldPerson(t, 14)},
		aeSpec{id: "hp2", spell: "hold_person", caster: "wiz", concentration: true, subject: aeCreature2("orc"), template: aeHoldPerson(t, 14)})
	for name, attempt := range map[string]d20.SaveAttempt{"chooses to fail": d20.ChooseToFail{}, "automatic failure": d20.AutomaticFailure{}} {
		roller := dicetest.NewFixedRoller()
		uc, _ := combat.NewAdvanceActiveEffects(roller)
		res, err := uc.Run(context.Background(), combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"), Saves: aeSaves("orc", ability.Wisdom, attempt)})
		out, ok := res.(combat.AdvanceActiveEffectsApplied)
		if err != nil || !ok || len(roller.Requests()) != 0 || !out.Effects().Equal(reg) {
			t.Fatalf("%s: = %#v, %v, dice %d", name, res, err, len(roller.Requests()))
		}
		for _, ev := range out.Events() {
			if ev.(combat.ActiveEffectRepeatSaveMade).Outcome() != d20.Failure {
				t.Errorf("%s: a forced save did not fail", name)
			}
		}
	}
	roller := dicetest.NewMinimumRoller()
	uc, _ := combat.NewAdvanceActiveEffects(roller)
	res, err := uc.Run(context.Background(), combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc")})
	rej, ok := res.(combat.AdvanceActiveEffectsRejected)
	if err != nil || !ok || len(roller.Requests()) != 0 {
		t.Fatalf("missing attempt = %#v, %v, dice %d", res, err, len(roller.Requests()))
	}
	problems := rej.Rejection().Problems()
	if len(problems) != 1 || problems[0].Reason != combat.EffectRejectMissingSaveAttempt || problems[0].Combatant != "orc" || problems[0].Ability != ability.Wisdom {
		t.Errorf("problems = %v, want one missing pair (orc, Wisdom) [FR-031]", problems)
	}
	nilAttempt := aeSaves("orc", ability.Wisdom, nil)
	if res, _ := uc.Run(context.Background(), combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"), Saves: nilAttempt}); res.(combat.AdvanceActiveEffectsRejected).Rejection().Reasons()[0] != combat.EffectRejectMissingSaveAttempt {
		t.Error("a nil attempt is not missing")
	}
	two := aeSeq(t,
		aeSpec{id: "hp1", spell: "hold_person", caster: "wiz", concentration: true, subject: aeCreature2("orc"), template: aeHoldPerson(t, 14)},
		aeSpec{id: "hp2", spell: "other_hold", caster: "cleric", subject: aeCreature2("orc"), template: aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{
			aeMust(combat.NewEffectStage(combat.EffectStageParams{Duration: aeRounds(t, 5), Triggers: []combat.EffectTrigger{aeTrigger(t, combat.EffectTurnOfSubject, content.BoundaryEnd, aeRepeatSave(t, ability.Constitution, 12))}}))}}))})
	res, _ = uc.Run(context.Background(), combat.AdvanceActiveEffectsCommand{Effects: two, Tick: aeEndTick(t, "orc")})
	if got := len(res.(combat.AdvanceActiveEffectsRejected).Rejection().Problems()); got != 2 {
		t.Errorf("missing pairs = %d, want both listed", got)
	}
}

// Requirement: doc/specs/active-effects.md FR-028, FR-063 - several effects that fire at one tick are processed in the host's order or registry order; an order that omits, repeats or names an id that does not fire is rejected invalidOrder listing every problem, class invalid, and is checked before a missing save attempt.
// Path: per case.
func TestSimultaneousOrder(t *testing.T) {
	t.Parallel()
	mk := func(id, spell string) aeSpec {
		return aeSpec{id: id, spell: spell, caster: "wiz", subject: aeCreature2("orc"), template: aeHoldPerson(t, 10)}
	}
	reg := aeSeq(t, mk("a", "hold_one"), mk("b", "hold_two"), roundsSpec("quiet", 5, ""))
	run := func(order []string) combat.AdvanceActiveEffectsResult {
		var ids []combat.ActiveEffectID
		for _, o := range order {
			ids = append(ids, aeEffectID(o))
		}
		return aeAdvanceCmd(t, combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"), Order: ids,
			Saves: aeSaves("orc", ability.Wisdom, d20.ChooseToFail{})})
	}
	order := func(res combat.AdvanceActiveEffectsResult) []string {
		out := []string{}
		for _, e := range res.(combat.AdvanceActiveEffectsApplied).Events() {
			out = append(out, string(e.EffectID()))
		}
		return out
	}
	if got := order(run(nil)); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("registry order = %v, want [a b]", got)
	}
	if got := order(run([]string{"b", "a"})); !slices.Equal(got, []string{"b", "a"}) {
		t.Errorf("host order = %v, want [b a] [FR-028]", got)
	}
	bad := map[string][]string{"omits an id": {"a", "x"}, "repeats an id": {"a", "a", "b"}, "names an id that does not fire": {"a", "b", "quiet"}, "names an unknown id": {"a", "b", "zzz"}}
	for name, o := range bad {
		rej, ok := run(o).(combat.AdvanceActiveEffectsRejected)
		if !ok || rej.Rejection().Reasons()[0] != combat.EffectRejectInvalidOrder {
			t.Errorf("an order that %s was not rejected invalidOrder: %#v", name, run(o))
		}
	}
	rej := run([]string{"a", "a", "zzz"}).(combat.AdvanceActiveEffectsRejected).Rejection()
	var got []combat.ActiveEffectID
	for _, p := range rej.Problems() {
		got = append(got, p.Effect)
	}
	if !slices.Equal(got, []combat.ActiveEffectID{"a", "zzz", "b"}) {
		t.Errorf("problems name %v, want the repeat, the unknown id and the omitted b", got)
	}
	both := aeAdvanceCmd(t, combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"), Order: []combat.ActiveEffectID{"a"}}, 1)
	if r := both.(combat.AdvanceActiveEffectsRejected).Rejection(); !slices.Equal(r.Reasons(), []combat.ActiveEffectRejectionKind{combat.EffectRejectInvalidOrder}) {
		t.Errorf("invalidOrder with missingSaveAttempt lists %v, want only invalidOrder [FR-063]", r.Reasons())
	}
}

// Requirement: doc/specs/active-effects.md FR-027 (SRD 5.2.1, "Flaming Sphere", p. 132) - a creature-in-area trigger fires for the combatant of the tick when the host lists the effect in the area effects, at the matching boundary only, and plans damage for that creature; a caster trigger fires on the caster's turn; none fires for an effect not listed.
// Path: per case.
func TestTriggerOwners(t *testing.T) {
	t.Parallel()
	sphere := aeSeq(t, aeSpec{id: "sp", spell: "flaming_sphere", caster: "wiz", concentration: true, subject: aePlace(t, "sphere"), template: aeFlamingSphere(t, 13), active: "wiz"})
	area := []combat.ActiveEffectID{"sp"}
	res := aeAdvanceCmd(t, combat.AdvanceActiveEffectsCommand{Effects: sphere, Tick: aeEndTick(t, "orc"), AreaEffects: area})
	plans := res.(combat.AdvanceActiveEffectsApplied).DamagePlans()
	if len(plans) != 1 || !slices.Equal(plans[0].Targets(), []combat.CombatantID{"orc"}) || plans[0].Audience() != combat.EffectAudienceTurnTaker ||
		plans[0].Effect() != "sp" || plans[0].DamageType() != content.DamageFire || plans[0].Amount() != dice.MustParseExpression("2d6") {
		t.Fatalf("plans = %#v, want one for the orc [FR-035]", plans)
	}
	if s, has := plans[0].Save(); !has || s.Ability() != ability.Dexterity || s.OnSuccess() != combat.SaveSuccessHalfDamage {
		t.Errorf("save = %v, %v", s, has)
	}
	for name, cmd := range map[string]combat.AdvanceActiveEffectsCommand{
		"not listed in the area":  {Effects: sphere, Tick: aeEndTick(t, "orc")},
		"the wrong boundary":      {Effects: sphere, Tick: aeStartTick(t, "orc"), AreaEffects: area},
		"a round start":           {Effects: sphere, Tick: combat.RoundStartReached{}, AreaEffects: area},
		"another effect's area":   {Effects: sphere, Tick: aeEndTick(t, "orc"), AreaEffects: []combat.ActiveEffectID{"other"}},
		"time that elapsed":       {Effects: sphere, Tick: aeElapsed(t, 1), AreaEffects: area},
		"an empty-handed tick":    {Effects: sphere, Tick: combat.EndOfCombatReached{}, AreaEffects: area},
		"a departure of someone":  {Effects: sphere, Tick: aeMust(combat.NewCombatantLeft("orc")), AreaEffects: area},
		"a start of the caster's": {Effects: sphere, Tick: aeStartTick(t, "wiz"), AreaEffects: area},
	} {
		if got := aeAdvanceCmd(t, cmd).(combat.AdvanceActiveEffectsApplied); len(got.DamagePlans()) != 0 {
			t.Errorf("%s: a trigger fired", name)
		}
	}
	fire := func(reg combat.ActiveEffects, tick combat.EffectClockTick) combat.AdvanceActiveEffectsApplied {
		return aeAdvance(t, reg, tick, "")
	}
	blast := aeSeq(t, aeSpec{id: "bf", spell: "delayed_blast_fireball", caster: "wiz", concentration: true, subject: aePlace(t, "bead"), template: aeDelayedBlast(t), active: "wiz"})
	if len(fire(blast, aeEndTick(t, "wiz")).Events()) != 1 || len(fire(blast, aeEndTick(t, "orc")).Events()) != 0 {
		t.Error("a caster trigger did not fire on the caster's turn end only")
	}
	subjectPlace := aeSeq(t, aeSpec{id: "x", spell: "bless", caster: "wiz", subject: aePlace(t, "p"), template: aeHoldPerson(t, 10)})
	if len(fire(subjectPlace, aeEndTick(t, "wiz")).Events()) != 0 {
		t.Error("a subject trigger fired on an effect on a place")
	}
}

// Requirement: doc/specs/active-effects.md FR-025, FR-035, FR-038, table T row T14 (SRD 5.2.1, "Acid Arrow", p. 107) - at one tick effects whose duration ends expire first and run their onEnd actions, and an expired effect does not fire a trigger at that tick; a subject-audience action on a place plans no damage.
// Path: unhappy (rule interaction).
func TestExpiryBeforeTriggersAndOnEnd(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, aeSpec{id: "aa", spell: "acid_arrow", caster: "wiz", subject: aeCreature2("orc"), template: aeAcidArrow(t, "orc"), active: "wiz"})
	out := aeAdvance(t, reg, aeEndTick(t, "orc"), "orc")
	plans := out.DamagePlans()
	if out.Effects().Len() != 0 || len(plans) != 1 || !slices.Equal(plans[0].Targets(), []combat.CombatantID{"orc"}) || plans[0].Audience() != combat.EffectAudienceSubject ||
		plans[0].Amount() != dice.MustParseExpression("2d4") || plans[0].DamageType() != content.DamageAcid {
		t.Errorf("T14 plans = %#v with %d effects, want one subject plan naming the orc", plans, out.Effects().Len())
	}
	both := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{aeMust(combat.NewEffectStage(combat.EffectStageParams{
		Duration: aeUntil(t, "orc", content.BoundaryEnd),
		Triggers: []combat.EffectTrigger{aeTrigger(t, combat.EffectTurnOfSubject, content.BoundaryEnd, aeDeal(t, "1d6", content.DamageFire, combat.EffectAudienceSubject))},
		OnEnd:    []combat.EffectAction{aeDeal(t, "2d6", content.DamageFire, combat.EffectAudienceSubject)}}))}}))
	reg = aeSeq(t, aeSpec{id: "x", spell: "bless", caster: "wiz", subject: aeCreature2("orc"), template: both})
	plans = aeAdvance(t, reg, aeEndTick(t, "orc"), "").DamagePlans()
	if len(plans) != 1 || plans[0].Amount() != dice.MustParseExpression("2d6") {
		t.Errorf("FR-025 plans = %v, want only the onEnd plan: the expired effect fired no trigger", plans)
	}
	onPlace := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{aeMust(combat.NewEffectStage(combat.EffectStageParams{
		Duration: aeRounds(t, 1), OnEnd: []combat.EffectAction{aeDeal(t, "1d4", content.DamageFire, combat.EffectAudienceSubject)}}))}}))
	reg = aeSeq(t, aeSpec{id: "p", spell: "bless", caster: "wiz", subject: aePlace(t, "bead"), template: onPlace})
	if got := aeAdvance(t, reg, aeElapsed(t, 1), "").DamagePlans(); len(got) != 0 {
		t.Errorf("a subject action on a place planned %v [FR-035]", got)
	}
}

// Requirement: doc/specs/active-effects.md FR-037, FR-038, FR-064 (SRD 5.2.1, "Delayed Blast Fireball", p. 122) - stored damage grows at each end of the caster's turn (12d6 to 15d6 over three), explodes on the area when the effect ends with the stored amount, never grows on a suppressed effect, and saturates at the limit of an expression.
// Path: unhappy (boundary).
func TestStoredDamageGrowsAndExplodes(t *testing.T) {
	t.Parallel()
	blast := aeSeq(t, aeSpec{id: "bf", spell: "delayed_blast_fireball", caster: "wiz", concentration: true, subject: aePlace(t, "bead"), template: aeDelayedBlast(t), active: "wiz"})
	for range 3 {
		out := aeAdvance(t, blast, aeEndTick(t, "wiz"), "wiz")
		blast = out.Effects()
		combattest.CheckEvent(t, out.Events()[0], "active_effect_stored_damage_grew", "wiz")
	}
	if s, _ := aeEffect(t, blast, "bf").StoredDamage(); s != dice.MustParseExpression("15d6") {
		t.Fatalf("stored = %v, want 15d6", s)
	}
	if ev := aeAdvance(t, blast, aeEndTick(t, "wiz"), "wiz").Events()[0].(combat.ActiveEffectStoredDamageGrew); ev.Stored() != dice.MustParseExpression("16d6") {
		t.Errorf("stored after the fourth turn = %v", ev.Stored())
	}
	for range 9 {
		blast = aeAdvance(t, blast, aeStartTick(t, "wiz"), "wiz").Effects()
	}
	boom := aeAdvance(t, blast, aeStartTick(t, "wiz"), "wiz")
	plans := boom.DamagePlans()
	if boom.Effects().Len() != 0 || len(plans) != 1 || plans[0].Amount() != dice.MustParseExpression("15d6") || plans[0].Audience() != combat.EffectAudienceAreaMembers || len(plans[0].Targets()) != 0 {
		t.Errorf("detonation plans = %#v, want one 15d6 plan for the area members", plans)
	}
	second := aeSpec{id: "bf2", spell: "delayed_blast_fireball", caster: "wiz", concentration: true, subject: aePlace(t, "bead"), template: aeDelayedBlast(t), active: "wiz"}
	pair := aeSeq(t, aeSpec{id: "bf1", spell: "delayed_blast_fireball", caster: "wiz", concentration: true, subject: aePlace(t, "bead"), template: aeDelayedBlast(t), active: "wiz"}, second)
	pair = aeAdvance(t, pair, aeEndTick(t, "wiz"), "wiz").Effects()
	if s, _ := aeEffect(t, pair, "bf1").StoredDamage(); s != dice.MustParseExpression("12d6") {
		t.Errorf("a suppressed effect grew to %v [FR-037]", s)
	}
	if s, _ := aeEffect(t, pair, "bf2").StoredDamage(); s != dice.MustParseExpression("13d6") {
		t.Errorf("the applying effect did not grow: %v", s)
	}
	grow := aeMust(combat.NewGrowStoredDamage(dice.MustParseExpression("5d6+7")))
	big := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{StoredDamage: dice.MustParseExpression("999d6+999999"), Stages: []combat.EffectStage{
		aeMust(combat.NewEffectStage(combat.EffectStageParams{Duration: aeRounds(t, 3), Triggers: []combat.EffectTrigger{aeTrigger(t, combat.EffectTurnOfCaster, content.BoundaryEnd, grow)}}))}}))
	reg := aeSeq(t, aeSpec{id: "big", spell: "bless", caster: "wiz", subject: aePlace(t, "p"), template: big})
	if s, _ := aeEffect(t, aeAdvance(t, reg, aeEndTick(t, "wiz"), "").Effects(), "big").StoredDamage(); s != dice.MustParseExpression("1000d6+1000000") {
		t.Errorf("saturated growth = %v, want 1000d6+1000000", s)
	}
}

// Requirement: doc/specs/active-effects.md FR-034, FR-045, table T (SRD 5.2.1, "Sleep", p. 163) - a stage gate runs when the stage's duration ends: a success ends the effect (repeatSaveSucceeded), a failure advances to the next stage with an ActiveEffectStageAdvanced, the new stage's conditions to apply and the old stage's to remove, and its clock set from the new stage.
// Path: per case.
func TestSleepStageGate(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, aeSpec{id: "sl", spell: "sleep", caster: "wiz", subject: aeCreature2("orc"), template: aeSleep(t, "orc", 13), active: "wiz",
		conditions: []content.ConditionKind{content.ConditionIncapacitated}})
	saves := aeSaves("orc", ability.Wisdom, aeRoll())
	run := func(face int) combat.AdvanceActiveEffectsApplied {
		return aeAdvanceCmd(t, combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeEndTick(t, "orc"), Saves: saves, Active: "wiz"}, face).(combat.AdvanceActiveEffectsApplied)
	}
	// The Sleep was cast on the caster's turn, so the first end of the orc's turn runs the gate.
	pass := run(13)
	if pass.Effects().Len() != 0 || !slices.Equal(pass.ConditionsToRemove(), []combat.EffectConditionPlan{{Effect: "sl", Target: "orc", Kind: content.ConditionIncapacitated}}) ||
		len(pass.ConditionsToApply()) != 0 {
		t.Errorf("a passed gate: effects %d remove %v apply %v", pass.Effects().Len(), pass.ConditionsToRemove(), pass.ConditionsToApply())
	}
	fail := run(12)
	e := aeEffect(t, fail.Effects(), "sl")
	if e.Stage() != 1 || !e.AnchoredTo("wiz") || !slices.Equal(e.Conditions(), []content.ConditionKind{content.ConditionUnconscious}) {
		t.Errorf("a failed gate: stage %d conditions %v, want stage 1 holding Unconscious anchored at the active combatant", e.Stage(), e.Conditions())
	}
	if !slices.Equal(fail.ConditionsToApply(), []combat.EffectConditionPlan{{Effect: "sl", Target: "orc", Kind: content.ConditionUnconscious}}) ||
		!slices.Equal(fail.ConditionsToRemove(), []combat.EffectConditionPlan{{Effect: "sl", Target: "orc", Kind: content.ConditionIncapacitated}}) {
		t.Errorf("a failed gate planned apply %v remove %v", fail.ConditionsToApply(), fail.ConditionsToRemove())
	}
	kinds := aeKinds(fail.Events())
	if !slices.Equal(kinds, []string{"active_effect_repeat_save_made", "active_effect_stage_advanced"}) {
		t.Errorf("events = %v", kinds)
	}
	if adv := fail.Events()[1].(combat.ActiveEffectStageAdvanced); adv.Stage() != 1 {
		t.Errorf("stage advanced to %d", adv.Stage())
	}
	combattest.CheckEvent(t, fail.Events()[1], "active_effect_stage_advanced", "orc", "wiz")
	if save := fail.Events()[0].(combat.ActiveEffectRepeatSaveMade); save.SaveKind() != combat.EffectSaveStageGate {
		t.Errorf("save kind = %v, want stage_gate", save.SaveKind())
	}
}

// Requirement: doc/specs/active-effects.md FR-034 - a stage that ends with no gate advances; a stage with a gate on a place has no creature to save, so it advances without a roll.
// Path: unhappy (state error).
func TestStageWithoutAGateAndGateOnAPlace(t *testing.T) {
	t.Parallel()
	tpl := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{
		aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 1), Gate: aeRepeatSave(t, ability.Wisdom, 10)}),
		aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 1)})}}))
	reg := aeSeq(t, aeSpec{id: "pl", spell: "bless", caster: "c", subject: aePlace(t, "p"), template: tpl})
	roller := dicetest.NewFixedRoller()
	uc, _ := combat.NewAdvanceActiveEffects(roller)
	res, err := uc.Run(context.Background(), combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: aeElapsed(t, 1)})
	if err != nil || aeEffect(t, res.(combat.AdvanceActiveEffectsApplied).Effects(), "pl").Stage() != 1 || len(roller.Requests()) != 0 {
		t.Errorf("a gate on a place: %v, %v, dice %d", res, err, len(roller.Requests()))
	}
}

// Requirement: doc/specs/active-effects.md FR-027, FR-061, FR-062, FR-037 - a trigger at the start of the caster's turn fires at that start and not at the end; the overall clock expires an effect when time overshoots it; a departure of another combatant leaves an until-boundary effect alone; an effect whose overall clock ended needs no gate attempt; stored damage growth adds the modifiers.
// Path: unhappy (boundary).
func TestAdvanceBoundaryCases(t *testing.T) {
	t.Parallel()
	grow := aeMust(combat.NewGrowStoredDamage(dice.MustParseExpression("1d6+3")))
	startTrigger := aeTrigger(t, combat.EffectTurnOfCaster, content.BoundaryStart, grow)
	tpl := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{StoredDamage: dice.MustParseExpression("2d6+2"), Overall: aeRounds(t, 4),
		Stages: []combat.EffectStage{aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 10), Triggers: []combat.EffectTrigger{startTrigger}})}}))
	reg := aeSeq(t, aeSpec{id: "s", spell: "bless", caster: "wiz", subject: aePlace(t, "p"), template: tpl})
	if got := aeAdvance(t, reg, aeEndTick(t, "wiz"), "").Events(); len(got) != 0 {
		t.Errorf("a start trigger fired at the end of the turn: %v", aeKinds(got))
	}
	grew := aeAdvance(t, reg, aeStartTick(t, "wiz"), "")
	if s, _ := aeEffect(t, grew.Effects(), "s").StoredDamage(); s != dice.MustParseExpression("3d6+5") || len(grew.Events()) != 1 {
		t.Errorf("stored = %v, want 3d6+5 after one growth at the start of the caster's turn", s)
	}
	if aeAdvance(t, reg, aeElapsed(t, 5), "").Effects().Len() != 0 || aeAdvance(t, reg, aeElapsed(t, 4), "").Effects().Len() != 0 || aeAdvance(t, reg, aeElapsed(t, 3), "").Effects().Len() != 1 {
		t.Error("the overall clock of 4 rounds did not expire at 4 and 5 elapsed and hold at 3")
	}
	owned := aeSeq(t, untilSpec("u", "b", content.BoundaryEnd, "a"))
	left := aeMust(combat.NewCombatantLeft("zzz"))
	if after := aeAdvance(t, owned, left, "").Effects(); !after.Equal(owned) {
		t.Error("the departure of another combatant changed an until-boundary effect")
	}
	gated := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Overall: aeRounds(t, 2), Stages: []combat.EffectStage{
		aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 2), Gate: aeRepeatSave(t, ability.Wisdom, 10)}),
		aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 2)})}}))
	reg = aeSeq(t, aeSpec{id: "g", spell: "bless", caster: "wiz", subject: aeCreature2("orc"), template: gated})
	if out := aeAdvance(t, reg, aeElapsed(t, 2), ""); out.Effects().Len() != 0 || len(out.Events()) != 1 {
		t.Errorf("an effect whose overall clock ended needed a gate attempt or did not end: %v", aeKinds(out.Events()))
	}
}
