package combat_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

const dispelSpec = "doc/specs/dispel-and-counterspell.md "

// dioUse builds the use case over r, failing the test on error.
func dioUse(tb testing.TB, r combat.Roller, opts ...combat.Option) combat.DispelOngoingSpells {
	tb.Helper()
	return luMust(combat.NewDispelOngoingSpells(r, opts...))
}

// dioRegistry is a goblin under a level 2 buff (a), a level 5 buff (b) and a Forcecage (c, never dispellable).
func dioRegistry(t *testing.T) (combat.OngoingSpellRegistry, combat.DispelTarget) {
	t.Helper()
	goblin := luMust(combat.NewDispelCreature(combat.MustCombatantID("goblin")))
	reg := luMust(combat.NewOngoingSpellRegistry([]combat.HeldSpell{
		dspHold(goblin, dspSpell(t, dspSpec{effect: "a", spell: "bless", level: 2})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "b", spell: "haste", level: 5, caster: "wizard", concentration: true})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "c", spell: "forcecage", level: 7, restriction: combat.DispelNeverDispellable{}})),
	}))
	return reg, goblin
}

// dioCommand is Dispel Magic cast at level 3 with a +2 modifier on goblin.
func dioCommand(t *testing.T) combat.DispelOngoingSpellsCommand {
	t.Helper()
	reg, goblin := dioRegistry(t)
	return combat.DispelOngoingSpellsCommand{Registry: reg, Target: goblin, CastLevel: character.MustSlotLevel(3),
		AbilityModifier: ability.MustScore(14).Modifier(), Modifiers: d20.NewRollModifiers(modifier.New(0))}
}

func dioApplied(tb testing.TB, res combat.DispelOngoingSpellsResult, err error) combat.OngoingSpellsDispelled {
	tb.Helper()
	r, ok := res.(combat.OngoingSpellsDispelled)
	if err != nil || !ok {
		tb.Fatalf("Run = %#v, %v, want OngoingSpellsDispelled", res, err)
	}
	return r
}

func dioEffects(spells []combat.OngoingSpell) []string {
	var out []string
	for _, s := range spells {
		out = append(out, s.Effect().ID().String())
	}
	return out
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-012, FR-013, FR-014, FR-016, FR-021, FR-023 - one d20 is rolled for the level 5 spell only (DC 15), the level 2 spell ends without a die and the Forcecage is never given one; a total equal to the DC ends the spell and a total one below leaves it (SRD 5.2.1, "Dispel Magic", p. 124; "Compare the Total to a Target Number", p. 6).
// Path: per case.
func TestDispelOngoingSpellsRollsOneCheckPerSpellThatNeedsOne(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		natural                 int
		wantEnded, wantSurvived []string
		wantTotal               int
	}{
		{"total equal to the DC ends the spell", "FR-021", "unhappy (boundary)", 13, []string{"a", "b"}, []string{"c"}, 15},
		{"total one below the DC leaves it", "FR-021", "unhappy (boundary)", 12, []string{"a"}, []string{"b", "c"}, 14},
		{"natural 20", "FR-020", "happy", 20, []string{"a", "b"}, []string{"c"}, 22},
		{"natural 1 has no special effect", "FR-020", "unhappy (boundary)", 1, []string{"a"}, []string{"b", "c"}, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.natural)
			res, err := dioUse(t, roller).Run(context.Background(), dioCommand(t))
			got := dioApplied(t, res, err)
			concl := got.Conclusion()
			if !slices.Equal(dioEffects(concl.Ended()), c.wantEnded) || !slices.Equal(dioEffects(concl.Survived()), c.wantSurvived) {
				t.Errorf("ended %v, survived %v, want %v and %v [%s%s]", dioEffects(concl.Ended()), dioEffects(concl.Survived()), c.wantEnded, c.wantSurvived, dispelSpec, c.requirement)
			}
			checks := got.Checks()
			if len(checks) != 1 || checks[0].Effect().String() != "b" || checks[0].Check().Total() != c.wantTotal || checks[0].Check().Target() != 15 {
				t.Errorf("checks = %+v, want one check for b with total %d against 15", checks, c.wantTotal)
			}
			if r := roller.Requests(); len(r) != 1 || r[0].Notation() != "1d20" {
				t.Errorf("requests = %v, want one 1d20", r)
			}
			if len(got.Plan().Steps()) != 3 {
				t.Errorf("plan has %d steps, want 3", len(got.Plan().Steps()))
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-020 - the check adds the spellcasting ability modifier and the extra Others of the command but never a Proficiency Bonus: a natural 12 with +2 and a Proficiency Bonus of +3 given in Modifiers is 14 against DC 15 and fails, and an extra +1 die total lifts it to 15 (SRD 5.2.1, "Dispel Magic", p. 124).
// Path: per case.
func TestDispelOngoingSpellsAddsNoProficiencyBonus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		modifiers d20.RollModifiers
		wantTotal int
		wantEnded bool
	}{
		{"a Proficiency Bonus in Modifiers is ignored", d20.NewRollModifiers(modifier.New(0)).WithProficiencyBonus(modifier.New(3)), 14, false},
		{"an extra modifier is added", d20.NewRollModifiers(modifier.New(0)).WithOthers(modifier.New(1)), 15, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := dioCommand(t)
			cmd.Modifiers = c.modifiers
			res, err := dioUse(t, dicetest.NewFixedRoller(12)).Run(context.Background(), cmd)
			got := dioApplied(t, res, err)
			ended := slices.Contains(dioEffects(got.Conclusion().Ended()), "b")
			if total := got.Checks()[0].Check().Total(); total != c.wantTotal || ended != c.wantEnded {
				t.Errorf("total = %d, ended = %v, want %d and %v [%sFR-020]", total, ended, c.wantTotal, c.wantEnded, dispelSpec)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-020, testing.md section 4 (rule interactions) - Advantage and Disadvantage together cancel to one d20; Advantage alone rolls two d20 and keeps the higher (SRD 5.2.1, "Advantage and Disadvantage", p. 7).
// Path: per case.
func TestDispelOngoingSpellsResolvesTheRollModeOnce(t *testing.T) {
	t.Parallel()
	adv := modifier.Advantage{}
	dis := modifier.Disadvantage{}
	cases := []struct {
		name          string
		circumstances []modifier.RollCircumstance
		faces         []int
		wantDice      int
		wantTotal     int
	}{
		{"both cancel to a normal roll", []modifier.RollCircumstance{adv, dis}, []int{9}, 1, 11},
		{"Advantage keeps the higher", []modifier.RollCircumstance{adv}, []int{4, 11}, 2, 13},
		{"Disadvantage keeps the lower", []modifier.RollCircumstance{dis}, []int{4, 11}, 2, 6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			cmd := dioCommand(t)
			cmd.Circumstances = c.circumstances
			res, err := dioUse(t, roller).Run(context.Background(), cmd)
			got := dioApplied(t, res, err)
			if n := len(roller.Requests()); n != 1 {
				t.Errorf("%d roller requests, want one (the dice of the mode are one expression)", n)
			}
			if r := roller.Requests()[0]; r.Count() != c.wantDice {
				t.Errorf("rolled %d dice, want %d [%sFR-020]", r.Count(), c.wantDice, dispelSpec)
			}
			if total := got.Checks()[0].Check().Total(); total != c.wantTotal {
				t.Errorf("total = %d, want %d", total, c.wantTotal)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-011, FR-023 - a target with no spells, and a target whose only spell is blocked, roll nothing and end nothing: the registry is equal to the input and no event is produced (SRD 5.2.1, "Dispel Magic", p. 124).
// Path: unhappy (empty and minimal).
func TestDispelOngoingSpellsWithNothingToEnd(t *testing.T) {
	t.Parallel()
	reg, goblin := dioRegistry(t)
	ogre := luMust(combat.NewDispelCreature(combat.MustCombatantID("ogre")))
	cage := luMust(combat.NewOngoingSpellRegistry([]combat.HeldSpell{dspHold(goblin, dspSpell(t, dspSpec{effect: "c", spell: "forcecage", level: 7, restriction: combat.DispelNeverDispellable{}}))}))
	cases := []struct {
		name   string
		reg    combat.OngoingSpellRegistry
		target combat.DispelTarget
	}{
		{"a target with no spells", reg, ogre},
		{"an empty registry", combat.OngoingSpellRegistry{}, goblin},
		{"only a blocked spell", cage, goblin},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller()
			cmd := dioCommand(t)
			cmd.Registry, cmd.Target = c.reg, c.target
			res, err := dioUse(t, roller).Run(context.Background(), cmd)
			got := dioApplied(t, res, err)
			if len(got.Conclusion().Ended()) != 0 || len(got.Conclusion().Events()) != 0 || !got.Conclusion().Registry().Equal(c.reg) || len(roller.Requests()) != 0 {
				t.Errorf("conclusion = %+v with %d requests, want nothing ended, an equal registry and no die [%sFR-023]", got.Conclusion(), len(roller.Requests()), dispelSpec)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-024, FR-025, FR-026 - the ended spells produce one OngoingSpellEnded each in plan order with the caster's Concentration flag, and the result registry no longer holds them (SRD 5.2.1, "Dispel Magic", p. 124; "Concentration", p. 179).
// Path: happy.
func TestDispelOngoingSpellsEventsAndRegistry(t *testing.T) {
	t.Parallel()
	cmd := dioCommand(t)
	res, err := dioUse(t, dicetest.NewFixedRoller(13)).Run(context.Background(), cmd)
	concl := dioApplied(t, res, err).Conclusion()
	var kinds []string
	var ends []bool
	for _, e := range concl.Events() {
		kinds = append(kinds, e.Kind())
		ends = append(ends, e.(combat.OngoingSpellEnded).EndsCasterConcentration())
	}
	if !slices.Equal(kinds, []string{"ongoing_spell_ended", "ongoing_spell_ended"}) || !slices.Equal(ends, []bool{false, true}) {
		t.Errorf("events = %v with concentration flags %v, want two endings, the second ending the caster's Concentration [%sFR-024, FR-025]", kinds, ends, dispelSpec)
	}
	if left := dioEffects(concl.Registry().SpellsOn(cmd.Target)); !slices.Equal(left, []string{"c"}) {
		t.Errorf("registry holds %v, want only c", left)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-027, testing.md section 4 (determinism, immutability) - the same command and the same dice give an equal result, and the command, its circumstances slice and its registry are not changed; the result returns copies of its slices.
// Path: unhappy (immutability).
func TestDispelOngoingSpellsLeavesTheCommandUnchanged(t *testing.T) {
	t.Parallel()
	cmd := dioCommand(t)
	cmd.Circumstances = []modifier.RollCircumstance{modifier.Advantage{}}
	before := cmd.Registry
	first, err := dioUse(t, dicetest.NewFixedRoller(5, 13)).Run(context.Background(), cmd)
	a := dioApplied(t, first, err)
	second, err := dioUse(t, dicetest.NewFixedRoller(5, 13)).Run(context.Background(), cmd)
	b := dioApplied(t, second, err)
	if !cmd.Registry.Equal(before) || len(cmd.Circumstances) != 1 || !before.Equal(dioCommand(t).Registry) {
		t.Error("Run changed its command")
	}
	if !a.Conclusion().Registry().Equal(b.Conclusion().Registry()) || a.Checks()[0].Check().Total() != b.Checks()[0].Check().Total() {
		t.Error("equal commands and dice gave different results (FR-027)")
	}
	checks := a.Checks()
	checks[0] = combat.DispelRolledCheck{}
	if a.Checks()[0].Effect().String() != "b" {
		t.Error("Checks returned the result's own slice")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-028, FR-029 - each spell from a different caster gets its own check, and only the spells on the chosen target are planned (SRD 5.2.1, "Dispel Magic", p. 124).
// Path: unhappy (rule interaction).
func TestDispelOngoingSpellsRollsEachSpellSeparately(t *testing.T) {
	t.Parallel()
	goblin := luMust(combat.NewDispelCreature(combat.MustCombatantID("goblin")))
	ogre := luMust(combat.NewDispelCreature(combat.MustCombatantID("ogre")))
	reg := luMust(combat.NewOngoingSpellRegistry([]combat.HeldSpell{
		dspHold(goblin, dspSpell(t, dspSpec{effect: "a", spell: "haste", level: 5, caster: "wizard", concentration: true})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "b", spell: "slow", level: 6, caster: "cleric", concentration: true})),
		dspHold(ogre, dspSpell(t, dspSpec{effect: "c", spell: "haste", level: 5, caster: "goblin", concentration: true})),
	}))
	roller := dicetest.NewFixedRoller(20, 1)
	cmd := dioCommand(t)
	cmd.Registry, cmd.Target = reg, goblin
	res, err := dioUse(t, roller).Run(context.Background(), cmd)
	got := dioApplied(t, res, err)
	if n := len(roller.Requests()); n != 2 {
		t.Errorf("%d dice rolled, want one per spell on the goblin (two)", n)
	}
	if ended := dioEffects(got.Conclusion().Ended()); !slices.Equal(ended, []string{"a"}) {
		t.Errorf("ended = %v, want only a (the second check fails)", ended)
	}
	if left := dioEffects(got.Conclusion().Registry().SpellsOn(ogre)); !slices.Equal(left, []string{"c"}) {
		t.Errorf("the ogre's spell = %v, want it kept", left)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-005 and plan P10 (active effects per creature 64) - a target holding 64 ongoing spells is concluded with every check rolled; one holding 65 gives DispelOngoingSpellsInvalid with DispelLimitExceeded and no die is rolled.
// Path: per case.
func TestDispelOngoingSpellsCapsTheSpellsOnATarget(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		count                   int
		wantLimit               bool
	}{
		{"at the cap", "P10", "unhappy (boundary)", 64, false},
		{"cap plus one", "P10", "unhappy (boundary)", 65, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			goblin := luMust(combat.NewDispelCreature(combat.MustCombatantID("goblin")))
			held := make([]combat.HeldSpell, c.count)
			for i := range held {
				held[i] = dspHold(goblin, dspSpell(t, dspSpec{effect: "e" + strconv.Itoa(i), spell: "haste", level: 5, caster: "wizard"}))
			}
			cmd := dioCommand(t)
			cmd.Registry, cmd.Target = luMust(combat.NewOngoingSpellRegistry(held)), goblin
			roller := dicetest.NewMaximumRoller()
			rec, opts := recording()
			res, err := dioUse(t, roller, opts...).Run(context.Background(), cmd)
			if c.wantLimit {
				inv, ok := res.(combat.DispelOngoingSpellsInvalid)
				if err != nil || !ok || inv.Problem() != combat.DispelLimitExceeded {
					t.Fatalf("Run = %#v, %v, want DispelOngoingSpellsInvalid{DispelLimitExceeded} [%s]", res, err, c.requirement)
				}
				want := map[string]any{"srd5e.combat.operation": "dispel_ongoing_spells", "srd5e.outcome": "failure", "srd5e.combat.result": "limit_exceeded", "srd5e.combat.result_class": "invalid"}
				if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !mapsEqual(ops[0], want) {
					t.Errorf("operations = %v, want one with %v", ops, want)
				}
				if spans := rec.Tracer.SpansNamed("srd5e.combat.dispel_ongoing_spells"); len(spans) != 1 || spans[0].Status != telemetry.StatusError {
					t.Errorf("spans = %+v, want one with error status", spans)
				}
				if rolled := len(roller.Requests()); rolled != 0 {
					t.Errorf("%d dice rolled, want none before the cap check", rolled)
				}
				return
			}
			got := dioApplied(t, res, err)
			if len(got.Checks()) != 64 || len(got.Conclusion().Ended()) != 64 || len(got.Conclusion().Events()) != 64 {
				t.Errorf("checks %d, ended %d, events %d, want 64 each [%s]", len(got.Checks()), len(got.Conclusion().Ended()), len(got.Conclusion().Events()), c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-015, errors-and-results FR-031 - an argument the plan refuses (a slot below 3, a target without an id) is an error matching srderr.ErrInvalidInput recorded on the span only: no operations sample, no die.
// Path: unhappy (invalid input).
func TestDispelOngoingSpellsRefusesWhatPlanRefuses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*combat.DispelOngoingSpellsCommand)
	}{
		{"slot 2", func(c *combat.DispelOngoingSpellsCommand) { c.CastLevel = character.MustSlotLevel(2) }},
		{"no slot", func(c *combat.DispelOngoingSpellsCommand) { c.CastLevel = character.SlotLevel{} }},
		{"no target", func(c *combat.DispelOngoingSpellsCommand) { c.Target = nil }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := dioCommand(t)
			c.mutate(&cmd)
			roller := dicetest.NewFixedRoller(10)
			rec, opts := recording()
			res, err := dioUse(t, roller, opts...).Run(context.Background(), cmd)
			if !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
				t.Fatalf("Run = %v, %v, want ErrInvalidInput [%sFR-015]", res, err, dispelSpec)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.dispel_ongoing_spells")
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(roller.Requests()) != 0 || len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")) != 0 {
				t.Errorf("spans = %+v, requests = %d, want one error span, no die and no operations sample", spans, len(roller.Requests()))
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md section 8 (errors-and-results FR-031, FR-034, FR-051) - a roller that fails, panics or answers another expression is a *srderr.PortError for port dice with a nil result and no partial result, recorded once: error status on the span, one srd5e.combat.errors count with error.type dice_roller_error, no operations count, no event, no log; the second of two checks failing is the same.
// Path: unhappy (failure of collaborator).
func TestDispelOngoingSpellsPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	second := func() combat.Roller {
		n := 0
		return dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
			if n++; n == 2 {
				return dice.Result{}, cause
			}
			return dicetest.NewMaximumRoller().Roll(e)
		})
	}()
	rollers := map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": dioWrongRoller(), "fails on the second check": second}
	goblin := luMust(combat.NewDispelCreature(combat.MustCombatantID("goblin")))
	reg := luMust(combat.NewOngoingSpellRegistry(dioHolds(t, goblin,
		dspSpec{effect: "b", spell: "haste", level: 5}, dspSpec{effect: "d", spell: "slow", level: 6})))
	for name, roller := range rollers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := dioCommand(t)
			cmd.Registry = reg
			rec, opts := recording()
			res, err := dioUse(t, roller, opts...).Run(context.Background(), cmd)
			envPortFailure(t, err)
			if res != nil {
				t.Errorf("result = %#v, want nil", res)
			}
			if (name == "fails" || name == "fails on the second check") && !errors.Is(err, cause) {
				t.Errorf("errors.Is(err, cause) = false for %v", err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.dispel_ongoing_spells")
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Errorf("spans = %+v, want one ended span with one recorded error", spans)
			}
			if got := values(measured(rec, "srd5e.combat.errors"), "error.type"); !slices.Equal(got, []string{"dice_roller_error"}) {
				t.Errorf("errors counted = %v, want one dice_roller_error", got)
			}
			for _, n := range []string{"srd5e.combat.operations", "srd5e.combat.events"} {
				if m := rec.Meter.MeasurementsNamed(n); len(m) != 0 {
					t.Errorf("%s = %v, want no sample after a port failure", n, m)
				}
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// dioHolds returns the entries of the specs on target.
func dioHolds(t *testing.T, target combat.DispelTarget, specs ...dspSpec) []combat.HeldSpell {
	t.Helper()
	var out []combat.HeldSpell
	for _, s := range specs {
		out = append(out, dspHold(target, dspSpell(t, s)))
	}
	return out
}

// Requirement: doc/specs/dispel-and-counterspell.md section 8, testing.md section 4 (cancellation, invalid input) - a nil roller and the zero use case are invalid input before any signal or roll; a cancelled context is returned unwrapped before any signal and any roll.
// Path: unhappy (invalid input; cancellation).
func TestDispelOngoingSpellsInvalidUse(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewDispelOngoingSpells(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewDispelOngoingSpells(nil) = %v, want ErrInvalidInput", err)
	}
	cmd := dioCommand(t)
	if res, err := (combat.DispelOngoingSpells{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero use case = %v, %v", res, err)
	}
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := dioUse(t, roller, opts...).Run(ctx, cmd); !errors.Is(err, context.Canceled) || res != nil {
		t.Errorf("cancelled context = %v, %v, want context.Canceled", res, err)
	}
	if len(roller.Requests()) != 0 || len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("a cancelled call must roll nothing and record no signal")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md section 7 and FR-056 - the shared signals under dispel_ongoing_spells: class applied, result completed (a spell ended) or dispel_nothing_ended, srd5e.outcome success, one ongoing_spell_ended event count per ended spell and no other attribute, no log, one duration sample; the result is the same with the zero-value Noop telemetry and with the recording double.
// Path: per case.
func TestDispelOngoingSpellsSignals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		natural                 int
		castLevel               int
		result                  string
		events                  int
	}{
		{"a spell ended", "FR-056", "happy", 13, 3, "completed", 2},
		{"a spell ended without a check", "FR-056", "happy", 1, 5, "completed", 2},
		{"a failed check leaves the level 5 spell", "FR-056", "unhappy (rule interaction)", 1, 3, "completed", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := dioCommand(t)
			cmd.CastLevel = character.MustSlotLevel(c.castLevel)
			rec, opts := recording()
			got, err := dioUse(t, dicetest.NewFixedRoller(c.natural), opts...).Run(context.Background(), cmd)
			plain, perr := dioUse(t, dicetest.NewFixedRoller(c.natural)).Run(context.Background(), cmd)
			a, b := dioApplied(t, got, err), dioApplied(t, plain, perr)
			if !a.Conclusion().Registry().Equal(b.Conclusion().Registry()) || len(a.Conclusion().Events()) != len(b.Conclusion().Events()) {
				t.Fatal("recording and Noop telemetry gave different results")
			}
			want := map[string]any{"srd5e.combat.operation": "dispel_ongoing_spells", "srd5e.outcome": "success", "srd5e.combat.result": c.result, "srd5e.combat.result_class": "applied"}
			spans := rec.Tracer.SpansNamed("srd5e.combat.dispel_ongoing_spells")
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 || !mapsEqual(attrMap(spans[0].Attrs), want) {
				t.Fatalf("spans = %+v, want one ended span with %v [%s%s]", spans, want, dispelSpec, c.requirement)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !mapsEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v", ops, want)
			}
			if ev := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); len(ev) != c.events || (c.events > 0 && ev[0] != "ongoing_spell_ended") {
				t.Errorf("events counted = %v, want %d ongoing_spell_ended", ev, c.events)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors samples = %d, want 0", n)
			}
			if d := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(d) != 1 {
				t.Errorf("duration samples = %d, want one", len(d))
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md section 7 - a Dispel Magic that ends no spell is recorded as dispel_nothing_ended (class applied, outcome success), with no event.
// Path: unhappy (rule interaction).
func TestDispelOngoingSpellsNothingEndedSignal(t *testing.T) {
	t.Parallel()
	goblin := luMust(combat.NewDispelCreature(combat.MustCombatantID("goblin")))
	cmd := dioCommand(t)
	cmd.Registry = luMust(combat.NewOngoingSpellRegistry(dioHolds(t, goblin, dspSpec{effect: "b", spell: "haste", level: 5})))
	rec, opts := recording()
	res, err := dioUse(t, dicetest.NewFixedRoller(1), opts...).Run(context.Background(), cmd)
	if got := dioApplied(t, res, err); len(got.Conclusion().Ended()) != 0 {
		t.Fatalf("ended = %v, want none", got.Conclusion().Ended())
	}
	want := map[string]any{"srd5e.combat.operation": "dispel_ongoing_spells", "srd5e.outcome": "success", "srd5e.combat.result": "dispel_nothing_ended", "srd5e.combat.result_class": "applied"}
	if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !mapsEqual(ops[0], want) {
		t.Errorf("operations = %v, want one with %v [%ssection 7]", ops, want, dispelSpec)
	}
	if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.events")); n != 0 {
		t.Errorf("events samples = %d, want 0", n)
	}
}

// ExampleDispelOngoingSpells casts Dispel Magic with a level 3 slot on a creature under a level 5 spell; the d20 of 13 and
// a modifier of +2 meet the DC of 15, so the spell ends.
func ExampleDispelOngoingSpells() {
	goblin, _ := combat.NewDispelCreature(combat.MustCombatantID("goblin"))
	src, _ := combat.SpellSource("haste")
	id, _ := combat.NewOngoingEffectID("haste-1")
	eff, _ := combat.NewOngoingEffect(id, combat.OngoingEffectSpell, src)
	level, _ := content.NewSpellLevel(5)
	remaining, _ := combat.NewSpellTimeRemaining(60)
	spell, _ := combat.NewOngoingSpell(combat.OngoingSpellParams{Effect: eff, CastAtLevel: level, Origin: combat.SpellOriginOrdinary,
		Remaining: remaining, Restriction: combat.DispelUnrestricted{}})
	reg, _ := combat.NewOngoingSpellRegistry([]combat.HeldSpell{{Target: goblin, Spell: spell}})
	use, _ := combat.NewDispelOngoingSpells(dicetest.NewFixedRoller(13))
	res, _ := use.Run(context.Background(), combat.DispelOngoingSpellsCommand{Registry: reg, Target: goblin, CastLevel: character.MustSlotLevel(3),
		AbilityModifier: ability.MustScore(14).Modifier(), Modifiers: d20.NewRollModifiers(modifier.New(0))})
	out := res.(combat.OngoingSpellsDispelled)
	fmt.Println(len(out.Conclusion().Ended()), out.Checks()[0].Check().Total())
	// Output: 1 15
}

// dioWrongRoller answers a d6 to every request, so a d20 request gets another expression's result.
func dioWrongRoller() combat.Roller {
	return dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dicetest.NewMinimumRoller().Roll(dice.MustExpression(1, dice.D6(), 0))
	})
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-022 and plan P10 - each Dispel Magic problem has a stable snake_case name, including limit_exceeded for the cap; the zero value is not a problem.
// Path: per case.
func TestDispelMagicProblemNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		problem combat.DispelMagicProblem
		want    string
	}{
		{combat.DispelMissingCheck, "missing_check"},
		{combat.DispelUnexpectedCheck, "unexpected_check"},
		{combat.DispelCheckTargetMismatch, "check_target_mismatch"},
		{combat.DispelNotAnAbilityCheck, "not_an_ability_check"},
		{combat.DispelPlanMismatch, "plan_mismatch"},
		{combat.DispelLimitExceeded, "limit_exceeded"},
		{0, "unknown"},
	}
	for _, c := range cases {
		if got := c.problem.String(); got != c.want {
			t.Errorf("String(%d) = %q, want %q [%sFR-022]", int(c.problem), got, c.want, dispelSpec)
		}
	}
}
