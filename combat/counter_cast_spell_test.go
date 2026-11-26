package combat_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ccsUse builds the use case over r, failing the test on error.
func ccsUse(tb testing.TB, r combat.Roller, opts ...combat.Option) combat.CounterCastSpell {
	tb.Helper()
	return luMust(combat.NewCounterCastSpell(r, opts...))
}

// ccsRoll is a Constitution save with a +1 modifier.
func ccsRoll(circumstances ...modifier.RollCircumstance) d20.SaveAttempt {
	return d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(1)), Circumstances: circumstances}
}

// ccsCommand is the goblin countering a wizard's Fireball (V and S, a slot, a Magic action) from 30 feet with a DC 13
// and a save of +1.
func ccsCommand(t *testing.T) combat.CounterCastSpellCommand {
	t.Helper()
	return combat.CounterCastSpellCommand{
		Cast:           dspObservation(t, character.CastSourceSlot, combat.NewUsedComponents(true, true, false), false, content.NewCastingTimeMagicAction()),
		Counterspeller: dspCreature(t, "goblin"), DistanceFeet: 30, SeesCaster: true, SaveDC: dspDC(t), CasterSave: ccsRoll(),
	}
}

func ccsConcluded(tb testing.TB, res combat.CounterCastSpellResult, err error) combat.CounterCastSpellConcluded {
	tb.Helper()
	r, ok := res.(combat.CounterCastSpellConcluded)
	if err != nil || !ok {
		tb.Fatalf("Run = %#v, %v, want CounterCastSpellConcluded", res, err)
	}
	return r
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-033, FR-034 - the casting creature's Constitution save is rolled against the counterspeller's DC 13: a total of 13 resists (the boundary) and a total of 12 counters the cast (SRD 5.2.1, "Counterspell", p. 120; "Compare the Total to a Target Number", p. 6).
// Path: per case.
func TestCounterCastSpellRollsTheSave(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		natural                 int
		wantCountered           bool
		wantTotal               int
	}{
		{"total equal to the DC resists", "FR-033", "unhappy (boundary)", 12, false, 13},
		{"total one below the DC counters", "FR-034", "unhappy (boundary)", 11, true, 12},
		{"natural 1 fails", "FR-034", "unhappy (boundary)", 1, true, 2},
		{"natural 20 succeeds", "FR-033", "happy", 20, false, 21},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.natural)
			res, err := ccsUse(t, roller).Run(context.Background(), ccsCommand(t))
			got := ccsConcluded(t, res, err)
			rolled, ok := got.Save().(combat.CounterspellRolledSave)
			if !ok || rolled.Check().Total() != c.wantTotal || rolled.Check().Target() != 13 || rolled.Check().Kind() != d20.SavingThrow {
				t.Fatalf("save = %#v, want a saving throw with total %d against 13 [%s%s]", got.Save(), c.wantTotal, dispelSpec, c.requirement)
			}
			_, countered := got.Result().(combat.CounterspellCountered)
			if _, resisted := got.Result().(combat.CounterspellResistedResult); countered != c.wantCountered || resisted == c.wantCountered {
				t.Errorf("result = %#v, want countered = %v", got.Result(), c.wantCountered)
			}
			if r := roller.Requests(); len(r) != 1 || r[0].Notation() != "1d20" {
				t.Errorf("requests = %v, want one 1d20", r)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-033, FR-034 - a save the creature chooses to fail or fails automatically draws no die and counters the cast; the evidence is the forced failure (SRD 5.2.1, "Counterspell", p. 120; "Saving Throws", p. 6).
// Path: per case.
func TestCounterCastSpellForcedFailureDrawsNoDie(t *testing.T) {
	t.Parallel()
	for name, attempt := range map[string]d20.SaveAttempt{"chosen to fail": d20.ChooseToFail{}, "automatic failure": d20.AutomaticFailure{}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller()
			cmd := ccsCommand(t)
			cmd.CasterSave = attempt
			res, err := ccsUse(t, roller).Run(context.Background(), cmd)
			got := ccsConcluded(t, res, err)
			if _, ok := got.Save().(combat.CounterspellForcedFailure); !ok {
				t.Errorf("save = %#v, want a forced failure", got.Save())
			}
			if _, ok := got.Result().(combat.CounterspellCountered); !ok || len(roller.Requests()) != 0 {
				t.Errorf("result = %#v with %d requests, want countered with no die [%sFR-033]", got.Result(), len(roller.Requests()), dispelSpec)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-034 to FR-037 - a countered cast reports the slot not expended only for a slot, the prior Concentration ending only for a Concentration spell, and the wasted action of its casting time (SRD 5.2.1, "Counterspell", p. 120; "Concentration", p. 179).
// Path: per case.
func TestCounterCastSpellReportsWhatTheCounteredCastCost(t *testing.T) {
	t.Parallel()
	reaction := luMust(content.NewCastingTimeReaction("a trigger"))
	minute := luMust(content.NewCastingTimeMinutes(1))
	cases := []struct {
		name, requirement, path string
		source                  character.CastSource
		concentration           bool
		time                    content.CastingTime
		wantSlot, wantPrior     bool
		wantWasted              combat.WastedActionCost
	}{
		{"a slot and a Magic action", "FR-034", "happy", character.CastSourceSlot, false, content.NewCastingTimeMagicAction(), true, false, combat.WastedMagicAction},
		{"a cantrip restores no slot", "FR-037", "unhappy (rule interaction)", character.CastSourceCantrip, false, content.NewCastingTimeMagicAction(), false, false, combat.WastedMagicAction},
		{"a Concentration spell ends prior Concentration", "FR-036", "unhappy (rule interaction)", character.CastSourceSlot, true, content.NewCastingTimeMagicAction(), true, true, combat.WastedMagicAction},
		{"a Reaction cast", "FR-035", "happy", character.CastSourceSlot, false, reaction, true, false, combat.WastedReaction},
		{"a one minute cast wastes no action", "FR-035", "unhappy (boundary)", character.CastSourceRitual, false, minute, false, false, combat.WastedNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := ccsCommand(t)
			cmd.Cast = dspObservation(t, c.source, combat.NewUsedComponents(true, true, false), c.concentration, c.time)
			res, err := ccsUse(t, dicetest.NewFixedRoller(1)).Run(context.Background(), cmd)
			countered, ok := ccsConcluded(t, res, err).Result().(combat.CounterspellCountered)
			if !ok || countered.SlotNotExpended() != c.wantSlot || countered.PriorConcentrationEnds() != c.wantPrior || countered.Wasted() != c.wantWasted {
				t.Errorf("result = %#v, want slot %v, prior concentration %v, wasted %v [%s%s]", countered, c.wantSlot, c.wantPrior, c.wantWasted, dispelSpec, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-031, FR-032 and table T-C - a cast the counterspeller cannot counter gives CounterCastSpellIneligible with the first refusal and no die: not seen, 61 feet, no components (SRD 5.2.1, "Counterspell", p. 120).
// Path: per case.
func TestCounterCastSpellIneligibleDrawsNoDie(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		sees                    bool
		distance                int
		used                    combat.UsedComponents
		want                    combat.CounterspellRefusal
	}{
		{"caster not seen", "FR-031", "unhappy (rule interaction)", false, 10, combat.NewUsedComponents(true, false, false), combat.CounterspellCasterNotSeen},
		{"61 feet", "FR-031", "unhappy (boundary)", true, 61, combat.NewUsedComponents(true, false, false), combat.CounterspellOutOfRange},
		{"no components (Subtle Spell)", "FR-032", "unhappy (rule interaction)", true, 10, combat.NewUsedComponents(false, false, false), combat.CounterspellNoComponents},
		{"not seen, far and no components", "FR-031", "unhappy (rule interaction)", false, 90, combat.NewUsedComponents(false, false, false), combat.CounterspellCasterNotSeen},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := ccsCommand(t)
			cmd.SeesCaster, cmd.DistanceFeet = c.sees, c.distance
			cmd.Cast = dspObservation(t, character.CastSourceSlot, c.used, false, content.NewCastingTimeMagicAction())
			roller := dicetest.NewFixedRoller()
			res, err := ccsUse(t, roller).Run(context.Background(), cmd)
			got, ok := res.(combat.CounterCastSpellIneligible)
			if err != nil || !ok || got.Refusal() != c.want || len(roller.Requests()) != 0 {
				t.Errorf("Run = %#v, %v with %d requests, want ineligible %v and no die [%s%s]", res, err, len(roller.Requests()), c.want, dispelSpec, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-033, testing.md section 4 (rule interactions) - Advantage and Disadvantage on the save cancel to one d20; Disadvantage alone rolls two d20 and keeps the lower (SRD 5.2.1, "Advantage and Disadvantage", p. 7).
// Path: per case.
func TestCounterCastSpellResolvesTheSaveRollMode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		circumstances []modifier.RollCircumstance
		faces         []int
		wantDice      int
		wantTotal     int
	}{
		{"both cancel", []modifier.RollCircumstance{modifier.Advantage{}, modifier.Disadvantage{}}, []int{8}, 1, 9},
		{"Disadvantage keeps the lower", []modifier.RollCircumstance{modifier.Disadvantage{}}, []int{15, 4}, 2, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			cmd := ccsCommand(t)
			cmd.CasterSave = ccsRoll(c.circumstances...)
			res, err := ccsUse(t, roller).Run(context.Background(), cmd)
			rolled := ccsConcluded(t, res, err).Save().(combat.CounterspellRolledSave)
			if r := roller.Requests(); len(r) != 1 || r[0].Count() != c.wantDice || rolled.Check().Total() != c.wantTotal {
				t.Errorf("requests = %v, total = %d, want %d dice and total %d [%sFR-033]", r, rolled.Check().Total(), c.wantDice, c.wantTotal, dispelSpec)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-039 - a Counterspell in response resolves first and is its own call: the host runs the response, then the original; a countered response is not cast, so the second call is made only if the response is resisted (two independent calls, each with its own die).
// Path: unhappy (rule interaction).
func TestCounterCastSpellChainIsTwoCalls(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(2, 17)
	use := ccsUse(t, roller)
	response := ccsCommand(t) // the wizard's side counters the goblin's Counterspell: its save fails
	first, err := use.Run(context.Background(), response)
	if _, ok := ccsConcluded(t, first, err).Result().(combat.CounterspellCountered); !ok {
		t.Fatalf("first = %#v, want countered", first)
	}
	original := ccsCommand(t) // the goblin's Counterspell was countered, so the original cast is not countered by it: the host stops here
	if r := roller.Requests(); len(r) != 1 {
		t.Fatalf("%d dice rolled after the first call, want one", len(r))
	}
	second, err := use.Run(context.Background(), original)
	if _, ok := ccsConcluded(t, second, err).Result().(combat.CounterspellResistedResult); !ok || len(roller.Requests()) != 2 {
		t.Errorf("second = %#v with %d dice, want resisted and one die more [%sFR-039]", second, len(roller.Requests()), dispelSpec)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-041, FR-031, testing.md section 4 (invalid input) - a zero cast, a zero counterspeller, a negative distance and a save attempt that is neither a roll nor a forced failure are errors matching srderr.ErrInvalidInput recorded on the span only: no die, no operations sample.
// Path: per case.
func TestCounterCastSpellRefusesBadCommands(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*combat.CounterCastSpellCommand)
	}{
		{"zero cast", func(c *combat.CounterCastSpellCommand) { c.Cast = combat.CastObservation{} }},
		{"zero counterspeller", func(c *combat.CounterCastSpellCommand) { c.Counterspeller = "" }},
		{"negative distance", func(c *combat.CounterCastSpellCommand) { c.DistanceFeet = -1 }},
		{"no save attempt", func(c *combat.CounterCastSpellCommand) { c.CasterSave = nil }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := ccsCommand(t)
			c.mutate(&cmd)
			roller := dicetest.NewFixedRoller(10)
			rec, opts := recording()
			res, err := ccsUse(t, roller, opts...).Run(context.Background(), cmd)
			if !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
				t.Fatalf("Run = %v, %v, want ErrInvalidInput", res, err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.counter_cast_spell")
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(roller.Requests()) != 0 || len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")) != 0 {
				t.Errorf("spans = %+v, requests = %d, want one error span, no die and no operations sample", spans, len(roller.Requests()))
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md section 8 (errors-and-results FR-031, FR-034, FR-051) - a roller that fails, panics or answers another expression is a *srderr.PortError for port dice with a nil result, recorded once: error status on the span, one srd5e.combat.errors count with error.type dice_roller_error, no operations count, no event, no log.
// Path: unhappy (failure of collaborator).
func TestCounterCastSpellPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	rollers := map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": dioWrongRoller()}
	for name, roller := range rollers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			res, err := ccsUse(t, roller, opts...).Run(context.Background(), ccsCommand(t))
			envPortFailure(t, err)
			if res != nil {
				t.Errorf("result = %#v, want nil", res)
			}
			if name == "fails" && !errors.Is(err, cause) {
				t.Errorf("errors.Is(err, cause) = false for %v", err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.counter_cast_spell")
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

// Requirement: doc/specs/dispel-and-counterspell.md section 8, testing.md section 4 (cancellation, immutability) - a nil roller and the zero use case are invalid input before any signal or roll; a cancelled context is returned unwrapped before any signal and any roll; the command is not changed.
// Path: unhappy (invalid input; cancellation).
func TestCounterCastSpellInvalidUse(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewCounterCastSpell(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewCounterCastSpell(nil) = %v, want ErrInvalidInput", err)
	}
	cmd := ccsCommand(t)
	if res, err := (combat.CounterCastSpell{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero use case = %v, %v", res, err)
	}
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(10)
	use := ccsUse(t, roller, opts...)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := use.Run(ctx, cmd); !errors.Is(err, context.Canceled) || res != nil {
		t.Errorf("cancelled context = %v, %v, want context.Canceled", res, err)
	}
	if len(roller.Requests()) != 0 || len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("a cancelled call must roll nothing and record no signal")
	}
	if _, err := use.Run(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if want := ccsCommand(t); cmd.Cast != want.Cast || cmd.DistanceFeet != want.DistanceFeet || cmd.Counterspeller != want.Counterspeller {
		t.Error("Run changed its command")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md section 7 and FR-042, FR-056 - the shared signals under counter_cast_spell: completed (countered, event spell_countered), counterspell_resisted (event counterspell_resisted) both applied with outcome success, counterspell_ineligible refused with outcome failure and no event and no error status; no log, one duration sample; the result is the same with the zero-value Noop telemetry and with the recording double.
// Path: per case.
func TestCounterCastSpellSignals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		mutate                  func(*combat.CounterCastSpellCommand)
		natural                 int
		result, class, outcome  string
		event                   string
		spanError               bool
	}{
		{"countered", "FR-042", "happy", func(*combat.CounterCastSpellCommand) {}, 3, "completed", "applied", "success", "spell_countered", false},
		{"resisted", "FR-042", "happy", func(*combat.CounterCastSpellCommand) {}, 19, "counterspell_resisted", "applied", "success", "counterspell_resisted", false},
		{"ineligible", "FR-056", "unhappy (rule interaction)", func(c *combat.CounterCastSpellCommand) { c.DistanceFeet = 61 }, 10, "counterspell_ineligible", "refused", "failure", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := ccsCommand(t)
			c.mutate(&cmd)
			rec, opts := recording()
			got, err := ccsUse(t, dicetest.NewFixedRoller(c.natural), opts...).Run(context.Background(), cmd)
			plain, perr := ccsUse(t, dicetest.NewFixedRoller(c.natural)).Run(context.Background(), cmd)
			if err != nil || perr != nil || fmt.Sprintf("%#v", got) != fmt.Sprintf("%#v", plain) {
				t.Fatalf("recording and Noop telemetry gave %#v, %v and %#v, %v", got, err, plain, perr)
			}
			want := map[string]any{"srd5e.combat.operation": "counter_cast_spell", "srd5e.outcome": c.outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			spans := rec.Tracer.SpansNamed("srd5e.combat.counter_cast_spell")
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != c.spanError || len(spans[0].Errors) != 0 || !mapsEqual(attrMap(spans[0].Attrs), want) {
				t.Fatalf("spans = %+v, want one ended span with %v [%s%s]", spans, want, dispelSpec, c.requirement)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !mapsEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v", ops, want)
			}
			events := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event")
			if (c.event == "") != (len(events) == 0) || (c.event != "" && !slices.Equal(events, []string{c.event})) {
				t.Errorf("events counted = %v, want [%s]", events, c.event)
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

// ExampleCounterCastSpell counters a Fireball whose caster rolls a 3 on its Constitution save against DC 13.
func ExampleCounterCastSpell() {
	wizard := combat.MustCombatantID("wizard")
	cast, _ := combat.NewCastObservation("fireball", wizard, character.CastSourceSlot, combat.NewUsedComponents(true, true, true), false, content.NewCastingTimeMagicAction())
	use, _ := combat.NewCounterCastSpell(dicetest.NewFixedRoller(3))
	level, _ := character.NewLevel(3)
	dc := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(level)).SaveDC()
	res, _ := use.Run(context.Background(), combat.CounterCastSpellCommand{Cast: cast, Counterspeller: combat.MustCombatantID("goblin"), DistanceFeet: 30, SeesCaster: true,
		SaveDC: dc, CasterSave: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}})
	_, countered := res.(combat.CounterCastSpellConcluded).Result().(combat.CounterspellCountered)
	fmt.Println(countered)
	// Output: true
}
