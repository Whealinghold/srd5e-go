package scenarios_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const grpFDC = 13

// grpFSave is a saving throw rolled with a natural face and no modifier against the printed DC 13.
func grpFSave(t *testing.T, face int) d20.CheckResult {
	t.Helper()
	roll := grpCMust(d20.NewRoll(modifier.Normal, []int{face}))
	return grpCMust(d20.ResolveCheck(d20.SavingThrow, roll, d20.NewRollModifiers(modifier.New(0)), grpFDC))
}

// grpFClause builds an effect clause with an optional Success rider and damage.
func grpFClause(label content.EffectClauseLabel, rider content.SaveSuccessRider, damage ...content.StatBlockDamage) content.EffectClause {
	return grpCMust(content.NewEffectClause(label, "T.", rider, damage))
}

// grpFSavePart is a monster whose first Action is a Wisdom saving throw (DC 13) with the clauses, at the position grpFRef.
func grpFSavePart(clauses ...content.EffectClause) content.Monster {
	st := grpCMust(content.NewStatBlockSavingThrow(ability.Wisdom, grpFDC, "", ""))
	part := grpCPart("Horrific Visage", content.StatBlockPartFields{SavingThrow: st, Clauses: clauses})
	return grpCMonster(func(f *content.MonsterFields) {
		f.Actions = []content.StatBlockPart{part, grpCUsage("Claw", content.PartUnlimited{})}
	})
}

var (
	grpFVisageRef = grpCRef(combat.MonsterPartActions, 0)
	grpFClawRef   = grpCRef(combat.MonsterPartActions, 1)
	grpFGhost     = grpCID("ghost")
	grpFHeroA     = grpCID("hero_a")
	grpFHeroB     = grpCID("hero_b")
)

// grpFVisage is the Horrific Visage of a ghost: a failed save is Frightened, a success makes the target immune to the part for 24 hours.
func grpFVisage(t *testing.T) content.Monster {
	t.Helper()
	day := grpCMust(content.NewImmuneToPartForHours(24))
	return grpFSavePart(grpFClause(content.ClauseFailure, nil), grpFClause(content.ClauseSuccess, day))
}

func grpFWindow(t *testing.T, target, source combat.CombatantID, ref combat.LimitedUsePartRef, d time.Duration) combat.SaveImmunityWindow {
	t.Helper()
	return grpCMust(combat.NewSaveImmunityWindow(target, source, ref, d))
}

func grpFEventKinds(events []combat.StatBlockActionEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

func grpFElapse(t *testing.T, s combat.SaveImmunities, d time.Duration) combat.SaveImmunityTransition {
	t.Helper()
	return grpCMust(s.Elapse(d))
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-039, FR-040, FR-041 - a hero succeeds on the ghost's Horrific Visage and the success grants a 24-hour window for exactly that hero, ghost and part, announced by one save_immunity_granted event; the hero is immune for 23 hours 59 minutes and is no longer immune when 24 hours have elapsed, which drops the window with one save_immunity_expired event; the engine reads no clock, so only the elapsed time the host reports counts.
// Path: happy.
func TestGrpFSaveImmunityWindowLifecycle(t *testing.T) {
	t.Parallel()
	part := grpFVisage(t).Actions()[0]
	results, err := combat.ResolveStatBlockSaveEffect(combat.StatBlockSaveEffectInput{
		Part: part, Ref: grpFVisageRef, Source: grpFGhost, Targets: []combat.CombatantID{grpFHeroA, grpFHeroB},
		Saves: map[combat.CombatantID]d20.CheckResult{grpFHeroA: grpFSave(t, 15), grpFHeroB: grpFSave(t, 4)},
	})
	if err != nil {
		t.Fatal(err)
	}
	won, ok := results[0].(combat.StatBlockSaveEffectApplies)
	if !ok {
		t.Fatalf("the hero that saved = %#v, want applies", results[0])
	}
	window, granted := won.Grant()
	if !granted || window.Remaining() != 24*time.Hour || window.Part() != grpFVisageRef || !slices.Equal(window.Combatants(), []combat.CombatantID{grpFHeroA, grpFGhost}) {
		t.Fatalf("grant = %+v %v, want 24 hours for hero_a against the ghost's Visage", window, granted)
	}
	if lost := results[1].(combat.StatBlockSaveEffectApplies); lost.DamageAdjustment() != combat.SaveEffectDamageUnchanged {
		t.Errorf("the hero that failed adjusts damage %v, want none", lost.DamageAdjustment())
	} else if _, gave := lost.Grant(); gave {
		t.Errorf("a failed save granted a window")
	}

	granting := combat.SaveImmunities{}.Grant(window)
	if !slices.Equal(grpFEventKinds(granting.Events()), []string{"save_immunity_granted"}) {
		t.Errorf("grant events = %v, want save_immunity_granted", grpFEventKinds(granting.Events()))
	}
	imm := granting.Immunities()
	if !imm.IsImmune(grpFHeroA, grpFGhost, grpFVisageRef) {
		t.Fatalf("hero_a is not immune right after the grant")
	}

	// 23 hours 59 minutes: still immune and nothing expired. One minute more is 24 hours: the window is gone.
	almost := grpFElapse(t, imm, 24*time.Hour-time.Minute)
	if !almost.Immunities().IsImmune(grpFHeroA, grpFGhost, grpFVisageRef) || len(almost.Events()) != 0 {
		t.Errorf("after 23h59m: immune %v events %v, want immune and none", almost.Immunities().IsImmune(grpFHeroA, grpFGhost, grpFVisageRef), grpFEventKinds(almost.Events()))
	}
	if left := almost.Immunities().Windows(); len(left) != 1 || left[0].Remaining() != time.Minute {
		t.Errorf("windows after 23h59m = %v, want one with a minute left", left)
	}
	done := grpFElapse(t, almost.Immunities(), time.Minute)
	if done.Immunities().IsImmune(grpFHeroA, grpFGhost, grpFVisageRef) || len(done.Immunities().Windows()) != 0 ||
		!slices.Equal(grpFEventKinds(done.Events()), []string{"save_immunity_expired"}) {
		t.Errorf("at 24h: windows %v events %v, want none and save_immunity_expired", done.Immunities().Windows(), grpFEventKinds(done.Events()))
	}
	// The same 24 hours in one report, and one nanosecond short of it.
	if exact := grpFElapse(t, imm, 24*time.Hour); exact.Immunities().IsImmune(grpFHeroA, grpFGhost, grpFVisageRef) {
		t.Errorf("immune after exactly 24 hours")
	}
	if short := grpFElapse(t, imm, 24*time.Hour-time.Nanosecond); !short.Immunities().IsImmune(grpFHeroA, grpFGhost, grpFVisageRef) {
		t.Errorf("not immune one nanosecond before 24 hours")
	}
	// Zero elapsed time changes nothing; the receiver is never changed.
	if zero := grpFElapse(t, imm, 0); !zero.Immunities().Equal(imm) || len(zero.Events()) != 0 {
		t.Errorf("zero elapsed time changed the windows or produced events")
	}
	if !imm.IsImmune(grpFHeroA, grpFGhost, grpFVisageRef) || len(imm.Windows()) != 1 {
		t.Errorf("Elapse changed the value it was called on")
	}
	// A negative duration is invalid input.
	if _, err := imm.Elapse(-time.Second); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative duration error = %v, want ErrInvalidInput", err)
	}
	// The zero value holds no window and is immune to nothing.
	var none combat.SaveImmunities
	if none.IsImmune(grpFHeroA, grpFGhost, grpFVisageRef) || len(grpFElapse(t, none, time.Hour).Events()) != 0 {
		t.Errorf("the zero SaveImmunities is immune to something or expires something")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-039, FR-040 - a window names the individual hero, the individual source creature and the part: another hero, another monster of the same kind and another part of the same monster are not covered.
// Path: unhappy (rule interaction).
func TestGrpFSaveImmunityCoversExactlyOneTriple(t *testing.T) {
	t.Parallel()
	imm := combat.SaveImmunities{}.Grant(grpFWindow(t, grpFHeroA, grpFGhost, grpFVisageRef, 24*time.Hour)).Immunities()
	cases := []struct {
		name   string
		target combat.CombatantID
		source combat.CombatantID
		part   combat.LimitedUsePartRef
		want   bool
	}{
		{"the same triple", grpFHeroA, grpFGhost, grpFVisageRef, true},
		{"another hero", grpFHeroB, grpFGhost, grpFVisageRef, false},
		{"another ghost of the same kind", grpFHeroA, grpCID("ghost_two"), grpFVisageRef, false},
		{"another part of the same ghost", grpFHeroA, grpFGhost, grpFClawRef, false},
		{"the same position in another section", grpFHeroA, grpFGhost, grpCRef(combat.MonsterPartBonusActions, 0), false},
		{"the ghost as the target and the hero as the source", grpFGhost, grpFHeroA, grpFVisageRef, false},
	}
	for _, c := range cases {
		if got := imm.IsImmune(c.target, c.source, c.part); got != c.want {
			t.Errorf("%s: IsImmune = %v, want %v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-041, FR-042 - granting a window for a triple that already has one keeps one window with the larger remaining time (the smaller grant never shortens it and windows never add); windows of different triples stay apart and expire in the order they were granted, each with its own event.
// Path: unhappy (rule interaction).
func TestGrpFSaveImmunityGrantsKeepTheLargerAndExpireInOrder(t *testing.T) {
	t.Parallel()
	day, half := 24*time.Hour, 12*time.Hour
	first := combat.SaveImmunities{}.Grant(grpFWindow(t, grpFHeroA, grpFGhost, grpFVisageRef, half))

	longer := first.Immunities().Grant(grpFWindow(t, grpFHeroA, grpFGhost, grpFVisageRef, day))
	if w := longer.Immunities().Windows(); len(w) != 1 || w[0].Remaining() != day {
		t.Errorf("a larger grant left windows %v, want one of 24 hours", w)
	}
	if ev := longer.Events(); len(ev) != 1 || ev[0].(combat.SaveImmunityGranted).Remaining() != day {
		t.Errorf("the grant event = %v, want one carrying the 24 hours now held", grpFEventKinds(ev))
	}
	shorter := longer.Immunities().Grant(grpFWindow(t, grpFHeroA, grpFGhost, grpFVisageRef, time.Hour))
	if w := shorter.Immunities().Windows(); len(w) != 1 || w[0].Remaining() != day {
		t.Errorf("a smaller grant left windows %v, want one still of 24 hours", w)
	}
	if ev := shorter.Events(); len(ev) != 1 || ev[0].(combat.SaveImmunityGranted).Remaining() != day {
		t.Errorf("the grant event after a smaller grant = %v, want the 24 hours still held", grpFEventKinds(ev))
	}
	if again := shorter.Immunities().Grant(grpFWindow(t, grpFHeroA, grpFGhost, grpFVisageRef, day)); len(again.Immunities().Windows()) != 1 || again.Immunities().Windows()[0].Remaining() != day {
		t.Errorf("a grant of the same length added time")
	}
	// A zero window changes nothing and produces no event.
	if noop := shorter.Immunities().Grant(combat.SaveImmunityWindow{}); len(noop.Events()) != 0 || !noop.Immunities().Equal(shorter.Immunities()) {
		t.Errorf("a zero window changed the immunities or produced an event")
	}

	// Two triples: hero_a for 12 hours and hero_b for 24. Windows keep the order they were granted; 12 hours expires the first only.
	two := first.Immunities().Grant(grpFWindow(t, grpFHeroB, grpFGhost, grpFVisageRef, day)).Immunities()
	if w := two.Windows(); len(w) != 2 || w[0].Combatants()[0] != grpFHeroA || w[1].Combatants()[0] != grpFHeroB {
		t.Fatalf("windows = %v, want hero_a then hero_b", w)
	}
	at12 := grpFElapse(t, two, half)
	if at12.Immunities().IsImmune(grpFHeroA, grpFGhost, grpFVisageRef) || !at12.Immunities().IsImmune(grpFHeroB, grpFGhost, grpFVisageRef) ||
		!slices.Equal(grpFEventKinds(at12.Events()), []string{"save_immunity_expired"}) {
		t.Errorf("after 12 hours: events %v, want hero_a expired and hero_b still immune", grpFEventKinds(at12.Events()))
	}
	// 24 hours in one report drops both, in the order they were granted.
	at24 := grpFElapse(t, two, day)
	kinds := grpFEventKinds(at24.Events())
	if len(at24.Immunities().Windows()) != 0 || !slices.Equal(kinds, []string{"save_immunity_expired", "save_immunity_expired"}) {
		t.Fatalf("after 24 hours: windows %v events %v, want both dropped", at24.Immunities().Windows(), kinds)
	}
	if !slices.Contains(at24.Events()[0].Combatants(), grpFHeroA) || !slices.Contains(at24.Events()[1].Combatants(), grpFHeroB) {
		t.Errorf("the expiries name %v then %v, want hero_a (granted first) then hero_b", at24.Events()[0].Combatants(), at24.Events()[1].Combatants())
	}
	// A rest is not a reset: the engine reads no clock, so nothing but Elapse shortens a window.
	if !two.IsImmune(grpFHeroA, grpFGhost, grpFVisageRef) {
		t.Errorf("a window changed without a report of elapsed time")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-043, FR-039 - a failed save, a success with another rider or none, and a creature that saves against itself grant no automatic window; a host builds a window directly for such a case, and the constructor refuses a window of no time, a zero or equal pair, or a zero part as invalid input.
// Path: unhappy (invalid input).
func TestGrpFSaveImmunityIsGrantedOnlyByTheDayRider(t *testing.T) {
	t.Parallel()
	day := grpCMust(content.NewImmuneToPartForHours(24))
	cases := []struct {
		name    string
		monster content.Monster
		face    int
		target  combat.CombatantID
		want    bool
	}{
		{"a success with the 24-hour rider", grpFSavePart(grpFClause(content.ClauseFailure, nil), grpFClause(content.ClauseSuccess, day)), 14, grpFHeroA, true},
		{"a failed save", grpFSavePart(grpFClause(content.ClauseFailure, nil), grpFClause(content.ClauseSuccess, day)), 12, grpFHeroA, false},
		{"a success with Half Damage Only", grpFSavePart(grpFClause(content.ClauseFailure, nil), grpFClause(content.ClauseSuccess, content.HalfDamageOnly{})), 14, grpFHeroA, false},
		{"a success with Half Damage", grpFSavePart(grpFClause(content.ClauseFailure, nil), grpFClause(content.ClauseSuccess, content.HalfDamage{})), 14, grpFHeroA, false},
		{"a success with no rider", grpFSavePart(grpFClause(content.ClauseFailure, nil), grpFClause(content.ClauseSuccess, nil)), 14, grpFHeroA, false},
		{"a part with no Success clause", grpFSavePart(grpFClause(content.ClauseFailure, nil)), 14, grpFHeroA, false},
		{"the ghost saving against its own part", grpFSavePart(grpFClause(content.ClauseFailure, nil), grpFClause(content.ClauseSuccess, day)), 14, grpFGhost, false},
	}
	for _, c := range cases {
		results, err := combat.ResolveStatBlockSaveEffect(combat.StatBlockSaveEffectInput{
			Part: c.monster.Actions()[0], Ref: grpFVisageRef, Source: grpFGhost, Targets: []combat.CombatantID{c.target},
			Saves: map[combat.CombatantID]d20.CheckResult{c.target: grpFSave(t, c.face)},
		})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if _, granted := results[0].(combat.StatBlockSaveEffectApplies).Grant(); granted != c.want {
			t.Errorf("%s: grant = %v, want %v", c.name, granted, c.want)
		}
	}

	// The host's own window (the ghost's Possession ending): positive time and distinct creatures only.
	if w, err := combat.NewSaveImmunityWindow(grpFHeroA, grpFGhost, grpFVisageRef, time.Nanosecond); err != nil || w.IsZero() || w.Remaining() != time.Nanosecond {
		t.Errorf("the smallest positive window = %+v, %v, want it built", w, err)
	}
	for name, args := range map[string]struct {
		target, source combat.CombatantID
		part           combat.LimitedUsePartRef
		d              time.Duration
	}{
		"no time":           {grpFHeroA, grpFGhost, grpFVisageRef, 0},
		"negative time":     {grpFHeroA, grpFGhost, grpFVisageRef, -time.Hour},
		"the same creature": {grpFGhost, grpFGhost, grpFVisageRef, time.Hour},
		"no target":         {combat.CombatantID(""), grpFGhost, grpFVisageRef, time.Hour},
		"no source":         {grpFHeroA, combat.CombatantID(""), grpFVisageRef, time.Hour},
		"no part":           {grpFHeroA, grpFGhost, combat.LimitedUsePartRef{}, time.Hour},
	} {
		if _, err := combat.NewSaveImmunityWindow(args.target, args.source, args.part, args.d); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// grpFResolve runs the use case for one target against the ghost's Visage with the host's immunities and a fixed natural face.
func grpFResolve(t *testing.T, e combat.Encounter, imm combat.SaveImmunities, target combat.CombatantID, roller *dicetest.FixedRoller) combat.ResolveStatBlockSaveCompleted {
	t.Helper()
	m := grpFVisage(t)
	cmd, err := combat.NewResolveStatBlockSaveCommand(combat.ResolveStatBlockSaveParams{
		Encounter: e, SourceID: grpFGhost, TargetID: target, Monster: m, Part: grpFVisageRef, Immunities: imm,
		Save:   d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))},
		Damage: []combat.Damage{grpCMust(combat.NewDamage(5, content.DamageNecrotic))}, Delivery: combat.OtherSource,
	})
	if err != nil {
		t.Fatal(err)
	}
	uc, err := combat.NewResolveStatBlockSave(roller)
	if err != nil {
		t.Fatal(err)
	}
	out, err := uc.Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	done, ok := out.(combat.ResolveStatBlockSaveCompleted)
	if !ok {
		t.Fatalf("outcome = %#v, want completed", out)
	}
	return done
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-039, FR-040, FR-041, FR-054 and doc/specs/combat-action-resolution.md FR-166 - a hero that succeeded on the ghost's Visage is immune on the next use: the use case rolls no die, reports the target as immune with no save and applies no damage; the host feeds the grant back into its immunities after the first success, and a hero who is not immune, or whose window has run its 24 hours, saves again.
// Path: happy, then unhappy (rule interaction).
func TestGrpFImmuneTargetTakesTheNoSavePath(t *testing.T) {
	t.Parallel()
	e := grpCEncounter(t, grpCPlayer(t, "ghost", combat.SideB, 50), grpCPlayer(t, "hero_a", combat.SideA, 30), grpCPlayer(t, "hero_b", combat.SideA, 30))
	hp := func(done combat.ResolveStatBlockSaveCompleted, id combat.CombatantID) int {
		c, _ := done.Encounter().Find(id)
		return c.HitPoints().HitPoints().Current()
	}

	// First use: hero_a rolls a 15 and succeeds, so the effect carries the 24-hour grant and takes the 5 damage in full (the 24-hour rider does not halve it).
	first := grpFResolve(t, e, combat.SaveImmunities{}, grpFHeroA, dicetest.NewFixedRoller(15))
	applies, ok := first.Effect().(combat.StatBlockSaveEffectApplies)
	if !ok {
		t.Fatalf("first use = %#v, want applies", first.Effect())
	}
	window, granted := applies.Grant()
	if _, rolled := first.Save(); !rolled || !granted {
		t.Fatalf("first use: rolled %v granted %v, want a save and a grant", rolled, granted)
	}
	imm := combat.SaveImmunities{}.Grant(window).Immunities()

	// Second use the same day: immune, no die, no save, no damage.
	noDice := dicetest.NewFixedRoller(1)
	second := grpFResolve(t, first.Encounter(), imm, grpFHeroA, noDice)
	if _, immune := second.Effect().(combat.StatBlockSaveTargetImmune); !immune {
		t.Fatalf("second use = %#v, want immune", second.Effect())
	}
	if _, rolled := second.Save(); rolled || len(noDice.Requests()) != 0 || len(second.Resolutions()) != 0 || hp(second, grpFHeroA) != hp(first, grpFHeroA) {
		t.Errorf("an immune target rolled %d dice, save %v, took %d resolutions, hp %d -> %d, want no roll and no damage", len(noDice.Requests()), rolled, len(second.Resolutions()), hp(first, grpFHeroA), hp(second, grpFHeroA))
	}

	// hero_b holds no window, so the same immunities do not cover them: they save (a 3 fails) and take the damage.
	other := grpFResolve(t, first.Encounter(), imm, grpFHeroB, dicetest.NewFixedRoller(3))
	if ap, ok := other.Effect().(combat.StatBlockSaveEffectApplies); !ok || ap.DamageAdjustment() != combat.SaveEffectDamageUnchanged {
		t.Fatalf("the hero without a window = %#v, want applies with no change", other.Effect())
	}
	if lost := 30 - hp(other, grpFHeroB); lost != 5 || len(other.Resolutions()) != 1 {
		t.Errorf("the hero without a window lost %d Hit Points in %d resolutions, want 5 in one", lost, len(other.Resolutions()))
	}

	// Elapsed time: 23h59m later the window still holds; 24 hours later it is gone and the hero saves again.
	almost := grpFElapse(t, imm, 24*time.Hour-time.Minute).Immunities()
	if _, immune := grpFResolve(t, first.Encounter(), almost, grpFHeroA, dicetest.NewFixedRoller(1)).Effect().(combat.StatBlockSaveTargetImmune); !immune {
		t.Errorf("not immune at 23h59m")
	}
	gone := grpFElapse(t, almost, time.Minute).Immunities()
	again := grpFResolve(t, first.Encounter(), gone, grpFHeroA, dicetest.NewFixedRoller(2))
	if _, rolled := again.Save(); !rolled {
		t.Errorf("a hero whose window ran out did not save")
	}
	if _, immune := again.Effect().(combat.StatBlockSaveTargetImmune); immune {
		t.Errorf("a hero whose window ran out is still immune")
	}
}

// grpFClauseRow is one row of the clause tables: the part's clauses in printed order and what a save does.
type grpFClauseRow struct {
	name string
	face int
	step combat.SaveFailureStep
	want []content.EffectClauseLabel
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-034, FR-055 - a failed save applies the Failure and Failure or Success clauses, the Failure by 5 or More clause when the total is at most the DC minus 5 (8 against DC 13, not 9), and the First, Second or Subsequent Failure clause the host's step names; a success applies the Success and Failure or Success clauses; the clauses come in printed order, the damage is never changed, and a step for a target that succeeded is ignored.
// Path: per case.
func TestGrpFSaveEffectClausesFollowTheFailureMargin(t *testing.T) {
	t.Parallel()
	fire := grpCDamage(7, dice.MustExpression(2, dice.D6(), 0), content.DamageFire)
	m := grpFSavePart(
		grpFClause(content.ClauseFailure, nil, fire),
		grpFClause(content.ClauseFailureByFiveOrMore, nil),
		grpFClause(content.ClauseFirstFailure, nil),
		grpFClause(content.ClauseSecondFailure, nil),
		grpFClause(content.ClauseSubsequentFailures, nil),
		grpFClause(content.ClauseSuccess, nil),
		grpFClause(content.ClauseFailureOrSuccess, nil),
	)
	f, byFive, first, second, later, success, either := content.ClauseFailure, content.ClauseFailureByFiveOrMore, content.ClauseFirstFailure,
		content.ClauseSecondFailure, content.ClauseSubsequentFailures, content.ClauseSuccess, content.ClauseFailureOrSuccess
	rows := []grpFClauseRow{
		{"a 8 is the DC minus 5: by five or more", 8, 0, []content.EffectClauseLabel{f, byFive, either}},
		{"a 9 fails by less than five", 9, 0, []content.EffectClauseLabel{f, either}},
		{"a 1 fails by five or more", 1, 0, []content.EffectClauseLabel{f, byFive, either}},
		{"a 12 is the last failure", 12, 0, []content.EffectClauseLabel{f, either}},
		{"the first failure", 12, combat.SaveFirstFailure, []content.EffectClauseLabel{f, first, either}},
		{"the second failure", 12, combat.SaveSecondFailure, []content.EffectClauseLabel{f, second, either}},
		{"a subsequent failure by five or more keeps printed order", 8, combat.SaveSubsequentFailures, []content.EffectClauseLabel{f, byFive, later, either}},
		{"a 13 is the DC and succeeds", 13, 0, []content.EffectClauseLabel{success, either}},
		{"a 20 succeeds", 20, 0, []content.EffectClauseLabel{success, either}},
		{"a step for a target that succeeded is ignored", 15, combat.SaveSecondFailure, []content.EffectClauseLabel{success, either}},
	}
	for _, r := range rows {
		steps := map[combat.CombatantID]combat.SaveFailureStep{}
		if r.step != 0 {
			steps[grpFHeroA] = r.step
		}
		out, err := combat.ResolveStatBlockSaveEffect(combat.StatBlockSaveEffectInput{
			Part: m.Actions()[0], Ref: grpFVisageRef, Source: grpFGhost, Targets: []combat.CombatantID{grpFHeroA},
			Saves: map[combat.CombatantID]d20.CheckResult{grpFHeroA: grpFSave(t, r.face)}, Steps: steps,
		})
		if err != nil {
			t.Fatalf("%s: %v", r.name, err)
		}
		applies := out[0].(combat.StatBlockSaveEffectApplies)
		var got []content.EffectClauseLabel
		for _, c := range applies.Clauses() {
			got = append(got, c.Label())
		}
		if !slices.Equal(got, r.want) || applies.DamageAdjustment() != combat.SaveEffectDamageUnchanged {
			t.Errorf("%s: clauses %v adjustment %v, want %v and no change", r.name, got, applies.DamageAdjustment(), r.want)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-035, FR-056 - a Success clause whose rider is Half Damage Only or Half Damage halves the damage on a success and never applies the Failure clause; Failure or Success still applies on both; a success with no rider, or one that grants immunity, leaves the damage unchanged; a failed save never halves.
// Path: per case.
func TestGrpFSaveEffectHalvesDamageOnlyWhenTheRiderSaysSo(t *testing.T) {
	t.Parallel()
	fire := grpCDamage(7, dice.MustExpression(2, dice.D6(), 0), content.DamageFire)
	day := grpCMust(content.NewImmuneToPartForHours(24))
	for _, c := range []struct {
		name  string
		rider content.SaveSuccessRider
		face  int
		want  combat.SaveEffectDamageAdjustment
	}{
		{"half damage only, success", content.HalfDamageOnly{}, 14, combat.SaveEffectDamageHalved},
		{"half damage only, failure", content.HalfDamageOnly{}, 12, combat.SaveEffectDamageUnchanged},
		{"half damage, success", content.HalfDamage{}, 14, combat.SaveEffectDamageHalved},
		{"half damage, failure", content.HalfDamage{}, 12, combat.SaveEffectDamageUnchanged},
		{"no rider, success", nil, 14, combat.SaveEffectDamageUnchanged},
		{"immunity rider, success", day, 14, combat.SaveEffectDamageUnchanged},
	} {
		m := grpFSavePart(grpFClause(content.ClauseFailure, nil, fire), grpFClause(content.ClauseSuccess, c.rider), grpFClause(content.ClauseFailureOrSuccess, nil))
		out, err := combat.ResolveStatBlockSaveEffect(combat.StatBlockSaveEffectInput{
			Part: m.Actions()[0], Ref: grpFVisageRef, Source: grpFGhost, Targets: []combat.CombatantID{grpFHeroA},
			Saves: map[combat.CombatantID]d20.CheckResult{grpFHeroA: grpFSave(t, c.face)},
		})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		applies := out[0].(combat.StatBlockSaveEffectApplies)
		hasFailure := slices.ContainsFunc(applies.Clauses(), func(cl content.EffectClause) bool { return cl.Label() == content.ClauseFailure })
		hasEither := slices.ContainsFunc(applies.Clauses(), func(cl content.EffectClause) bool { return cl.Label() == content.ClauseFailureOrSuccess })
		if applies.DamageAdjustment() != c.want || hasFailure != (c.face < grpFDC) || !hasEither {
			t.Errorf("%s: adjustment %v failure clause %v either clause %v, want %v, %v, true", c.name, applies.DamageAdjustment(), hasFailure, hasEither, c.want, c.face < grpFDC)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-053, FR-054, FR-039 - the resolver answers per target in the order given: an immune target is reported immune whatever its save, a target with no save is refused as missing, a save that is no saving throw or was rolled against another DC is refused as not this save, and a bad call (a part with no saving throw, a zero position or source, a zero, repeated or too many targets, an undefined step) is invalid input or over a cap.
// Path: unhappy (invalid input).
func TestGrpFSaveEffectRefusalsAndBadCalls(t *testing.T) {
	t.Parallel()
	m := grpFVisage(t)
	part := m.Actions()[0]
	imm := combat.SaveImmunities{}.Grant(grpFWindow(t, grpFHeroA, grpFGhost, grpFVisageRef, time.Hour)).Immunities()
	abilityCheck := grpCMust(d20.ResolveCheck(d20.AbilityCheck, grpCMust(d20.NewRoll(modifier.Normal, []int{15})), d20.NewRollModifiers(modifier.New(0)), grpFDC))
	otherDC := grpCMust(d20.ResolveCheck(d20.SavingThrow, grpCMust(d20.NewRoll(modifier.Normal, []int{15})), d20.NewRollModifiers(modifier.New(0)), grpFDC+1))
	targets := []combat.CombatantID{grpFHeroA, grpFHeroB, grpCID("hero_c"), grpCID("hero_d"), grpCID("hero_e")}
	out, err := combat.ResolveStatBlockSaveEffect(combat.StatBlockSaveEffectInput{
		Part: part, Ref: grpFVisageRef, Source: grpFGhost, Targets: targets, Immunities: imm,
		Saves: map[combat.CombatantID]d20.CheckResult{grpFHeroA: grpFSave(t, 1), grpCID("hero_c"): abilityCheck, grpCID("hero_d"): otherDC},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out[0].(combat.StatBlockSaveTargetImmune); !ok {
		t.Errorf("the immune target = %#v, want immune whatever its save", out[0])
	}
	want := map[int]combat.StatBlockSaveRefusal{1: combat.StatBlockSaveMissing, 2: combat.StatBlockSaveNotThisSave, 3: combat.StatBlockSaveNotThisSave}
	for i, reason := range want {
		if r, ok := out[i].(combat.StatBlockSaveTargetRefused); !ok || r.Reason() != reason {
			t.Errorf("target %d = %#v, want a refusal for %v", i, out[i], reason)
		}
	}
	if len(out) != len(targets) || !slices.Equal(out[0].Combatants(), []combat.CombatantID{grpFHeroA}) || !slices.Equal(out[4].Combatants(), []combat.CombatantID{grpCID("hero_e")}) {
		t.Errorf("results = %d, want one per target in the order given", len(out))
	}

	valid := combat.StatBlockSaveEffectInput{Part: part, Ref: grpFVisageRef, Source: grpFGhost, Targets: []combat.CombatantID{grpFHeroA}, Saves: map[combat.CombatantID]d20.CheckResult{grpFHeroA: grpFSave(t, 14)}}
	claw := m.Actions()[1]
	many := make([]combat.CombatantID, 257)
	for i := range many {
		many[i] = combat.CombatantID(fmt.Sprintf("target_%d", i))
	}
	bad := map[string]func(*combat.StatBlockSaveEffectInput){
		"a part with no saving throw": func(in *combat.StatBlockSaveEffectInput) { in.Part = claw },
		"a zero position":             func(in *combat.StatBlockSaveEffectInput) { in.Ref = combat.LimitedUsePartRef{} },
		"a zero source":               func(in *combat.StatBlockSaveEffectInput) { in.Source = combat.CombatantID("") },
		"a zero target":               func(in *combat.StatBlockSaveEffectInput) { in.Targets = []combat.CombatantID{combat.CombatantID("")} },
		"a repeated target":           func(in *combat.StatBlockSaveEffectInput) { in.Targets = []combat.CombatantID{grpFHeroA, grpFHeroA} },
		"an undefined step": func(in *combat.StatBlockSaveEffectInput) {
			in.Steps = map[combat.CombatantID]combat.SaveFailureStep{grpFHeroA: 9}
		},
		"a negative step": func(in *combat.StatBlockSaveEffectInput) {
			in.Steps = map[combat.CombatantID]combat.SaveFailureStep{grpFHeroA: -1}
		},
	}
	for name, edit := range bad {
		in := valid
		edit(&in)
		if _, err := combat.ResolveStatBlockSaveEffect(in); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	over := valid
	over.Targets = many
	var limit combat.LimitExceededError
	if _, err := combat.ResolveStatBlockSaveEffect(over); !errors.As(err, &limit) || limit.Max != 256 {
		t.Errorf("257 targets error = %v, want a LimitExceededError of 256", err)
	}
	// A step for a target that has no save, or that is immune, is ignored: no refusal, no error.
	ignored := valid
	ignored.Targets = []combat.CombatantID{grpFHeroA, grpFHeroB}
	ignored.Steps = map[combat.CombatantID]combat.SaveFailureStep{grpFHeroB: combat.SaveFirstFailure, grpCID("nobody"): combat.SaveSecondFailure}
	if res, err := combat.ResolveStatBlockSaveEffect(ignored); err != nil || len(res) != 2 {
		t.Errorf("steps for a target with no save: %v, %v, want results and no error", res, err)
	}
}
