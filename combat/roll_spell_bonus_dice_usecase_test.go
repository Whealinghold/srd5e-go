package combat_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func smBonusUse(tb testing.TB, r combat.Roller, opts ...combat.Option) combat.RollSpellBonusDice {
	tb.Helper()
	return luMust(combat.NewRollSpellBonusDice(r, opts...))
}

func smRolled(tb testing.TB, res combat.RollSpellBonusDiceResult, err error) combat.SpellBonusDiceRolled {
	tb.Helper()
	r, ok := res.(combat.SpellBonusDiceRolled)
	if err != nil || !ok {
		tb.Fatalf("Run = %#v, %v, want SpellBonusDiceRolled", res, err)
	}
	return r
}

func smDie(count, sides int, negative bool) combat.BonusDie {
	return combat.MustBonusDie(dice.MustExpression(count, luMust(dice.NewDie(sides)), 0), negative)
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-030, FR-031, FR-032, FR-034 - each die is rolled once through the roller in list order, one signed modifier per die (the rolled total, negated for a penalty die: a d4 showing 3 gives -3), with the rolls as evidence; no die means no request and an empty result (SRD 5.2.1, "Bless", p. 113; "Bane", p. 112; "Guidance", p. 138).
// Path: per case.
func TestRollSpellBonusDice(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		dice                    []combat.BonusDie
		faces                   []int
		wantRequests            []string
		wantMods                []modifier.Modifier
	}{
		{"one bonus die", "FR-030", "happy", []combat.BonusDie{smDie(1, 4, false)}, []int{3}, []string{"1d4"}, []modifier.Modifier{modifier.New(3)}},
		{"a Bane die is negative", "FR-032", "unhappy (boundary)", []combat.BonusDie{smDie(1, 4, true)}, []int{3}, []string{"1d4"}, []modifier.Modifier{modifier.New(-3)}},
		{"two dice in list order", "FR-030", "unhappy (rule interaction)", []combat.BonusDie{smDie(1, 4, false), smDie(1, 4, true), smDie(2, 6, false)}, []int{4, 1, 5, 6}, []string{"1d4", "1d4", "2d6"},
			[]modifier.Modifier{modifier.New(4), modifier.New(-1), modifier.New(11)}},
		{"the lowest and highest faces", "FR-032", "unhappy (boundary)", []combat.BonusDie{smDie(1, 4, true), smDie(1, 4, false)}, []int{1, 4}, []string{"1d4", "1d4"}, []modifier.Modifier{modifier.New(-1), modifier.New(4)}},
		{"no die", "FR-031", "unhappy (empty and minimal)", nil, nil, nil, nil},
		{"an empty slice", "FR-031", "unhappy (empty and minimal)", []combat.BonusDie{}, nil, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			cmd := combat.RollSpellBonusDiceCommand{Dice: c.dice}
			res, err := smBonusUse(t, roller).Run(context.Background(), cmd)
			got := smRolled(t, res, err)
			var requests []string
			for _, e := range roller.Requests() {
				requests = append(requests, e.Notation())
			}
			if !slices.Equal(got.Modifiers(), c.wantMods) || !slices.Equal(requests, c.wantRequests) || len(got.Rolls()) != len(c.wantMods) {
				t.Errorf("modifiers = %v, requests = %v, want %v and %v [%s%s]", got.Modifiers(), requests, c.wantMods, c.wantRequests, spellModifiersSpec, c.requirement)
			}
			for i, r := range got.Rolls() {
				if r.Expression() != c.dice[i].Expression() {
					t.Errorf("roll %d is of %v, want %v", i, r.Expression(), c.dice[i].Expression())
				}
			}
			again, err := smBonusUse(t, dicetest.NewFixedRoller(c.faces...)).Run(context.Background(), cmd)
			if a := smRolled(t, again, err); !slices.Equal(a.Modifiers(), got.Modifiers()) {
				t.Error("the same command and roller sequence must give an equal result (FR-035)")
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-033, FR-032, FR-014 - the dice are modifiers of the total: a natural 1 with a Bless die (+4) and a natural 20 with a Bane die (-4) stay a miss and a hit through the real d20.ResolveAttack, while a Bless die lifts a 10 total to a hit and a Bane die turns a 12 into a miss; with Advantage the roll has one die and the total may fall below 1 (SRD 5.2.1, "Rolling 20 or 1", p. 7; "D20 Tests", p. 6).
// Path: per case.
func TestBonusDiceAreModifiersOfTheTotal(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement string
		natural, bonus    int
		negative          bool
		face              int
		mode              modifier.RollMode
		dice              []int
		wantHit           bool
	}{
		{"natural 1 with Bless stays a miss", "FR-033", 1, 5, false, 4, modifier.Normal, []int{1}, false},
		{"natural 20 with Bane stays a hit", "FR-033", 20, 0, true, 4, modifier.Normal, []int{20}, true},
		{"Bless lifts a total of 10 to AC 12", "FR-030", 8, 2, false, 3, modifier.Normal, []int{8}, true},
		{"Bane turns a total of 12 into a miss", "FR-032", 10, 2, true, 1, modifier.Normal, []int{10}, false},
		{"Advantage rolls two d20 and one die", "FR-014", 6, 4, false, 2, modifier.WithAdvantage, []int{3, 6}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.face)
			res, err := smBonusUse(t, roller).Run(context.Background(), combat.RollSpellBonusDiceCommand{Dice: []combat.BonusDie{smDie(1, 4, c.negative)}})
			rolled := smRolled(t, res, err)
			roll := luMust(d20.NewRoll(c.mode, c.dice))
			mods := d20.NewRollModifiers(modifier.New(c.bonus)).WithOthers(rolled.Modifiers()...)
			attack := luMust(d20.ResolveAttack(roll, mods, 12, 20))
			if attack.Hit() != c.wantHit || len(roller.Requests()) != 1 {
				t.Errorf("hit = %v with %d dice rolled, want hit=%v and one die [%s%s]", attack.Hit(), len(roller.Requests()), c.wantHit, spellModifiersSpec, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-032, FR-008 of modifiers-and-advantage - a Bane die may take the total below 1: the modifier is -4 on a natural 1 and nothing clamps it.
// Path: unhappy (boundary).
func TestBaneDieMayTakeTheTotalBelowOne(t *testing.T) {
	t.Parallel()
	res, err := smBonusUse(t, dicetest.NewFixedRoller(4)).Run(context.Background(), combat.RollSpellBonusDiceCommand{Dice: []combat.BonusDie{smDie(1, 4, true)}})
	roll := luMust(d20.NewRoll(modifier.Normal, []int{1}))
	check := luMust(d20.ResolveCheck(d20.AbilityCheck, roll, d20.NewRollModifiers(modifier.New(0)).WithOthers(smRolled(t, res, err).Modifiers()...), 10))
	if check.Total() != -3 {
		t.Errorf("total = %d, want -3", check.Total())
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-035, FR-091, errors-and-results FR-031, FR-034 - a roller that fails (also on the second die), panics or answers another expression is a *srderr.PortError for port dice with a nil result and no partial result, recorded once: error status on the span, one srd5e.combat.errors count with error.type dice_roller_error, no operations count, no event, no log; errors.Is finds the cause.
// Path: unhappy (failure of collaborator).
func TestRollSpellBonusDicePortFailure(t *testing.T) {
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
	rollers := map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller(), "fails on the second die": second}
	cmd := combat.RollSpellBonusDiceCommand{Dice: []combat.BonusDie{smDie(1, 4, false), smDie(1, 4, true)}}
	for name, roller := range rollers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			res, err := smBonusUse(t, roller, opts...).Run(context.Background(), cmd)
			envPortFailure(t, err)
			if res != nil {
				t.Errorf("result = %#v, want nil", res)
			}
			if (name == "fails" || name == "fails on the second die") && !errors.Is(err, cause) {
				t.Errorf("errors.Is(err, cause) = false for %v", err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.roll_spell_bonus_dice")
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

// Requirement: doc/specs/spell-granted-modifiers.md FR-084, testing.md section 4 - a nil roller, the zero use case and a zero die are invalid input before any signal or roll; a cancelled context is returned unwrapped before any signal; the command is not changed.
// Path: unhappy (invalid input; cancellation).
func TestRollSpellBonusDiceInvalidUse(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewRollSpellBonusDice(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewRollSpellBonusDice(nil) = %v, want ErrInvalidInput", err)
	}
	valid := combat.RollSpellBonusDiceCommand{Dice: []combat.BonusDie{smDie(1, 4, false)}}
	if res, err := (combat.RollSpellBonusDice{}).Run(context.Background(), valid); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero use case = %v, %v", res, err)
	}
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(1)
	use := smBonusUse(t, roller, opts...)
	zero := combat.RollSpellBonusDiceCommand{Dice: []combat.BonusDie{smDie(1, 4, false), {}}}
	if res, err := use.Run(context.Background(), zero); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero die = %v, %v", res, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := use.Run(ctx, valid); !errors.Is(err, context.Canceled) || res != nil {
		t.Errorf("cancelled context = %v, %v, want context.Canceled", res, err)
	}
	if len(roller.Requests()) != 0 || len(rec.Tracer.SpansNamed("srd5e.combat.roll_spell_bonus_dice")) != 0 || len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")) != 0 {
		t.Error("an invalid use must roll nothing and record no signal")
	}
	if len(valid.Dice) != 1 || valid.Dice[0] != smDie(1, 4, false) {
		t.Error("Run altered the command")
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-090, FR-091, FR-092, section 7 - the shared signals under roll_spell_bonus_dice: class applied, result rolled (at least one die) or nothing_to_roll (none), no extra attribute, no event, no log, no id or amount in an attribute, one duration sample; the result is the same with the zero-value Noop telemetry and with the recording double.
// Path: per case.
func TestRollSpellBonusDiceSignals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		dice                    []combat.BonusDie
		faces                   []int
		result                  string
	}{
		{"one die", "FR-090", "happy", []combat.BonusDie{smDie(1, 4, false)}, []int{2}, "rolled"},
		{"two dice", "FR-090", "happy", []combat.BonusDie{smDie(1, 4, false), smDie(1, 4, true)}, []int{2, 3}, "rolled"},
		{"nothing to roll", "FR-090", "unhappy (empty and minimal)", nil, nil, "nothing_to_roll"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := combat.RollSpellBonusDiceCommand{Dice: c.dice}
			rec, opts := recording()
			got, err := smBonusUse(t, dicetest.NewFixedRoller(c.faces...), opts...).Run(context.Background(), cmd)
			plain, perr := smBonusUse(t, dicetest.NewFixedRoller(c.faces...)).Run(context.Background(), cmd)
			if a, b := smRolled(t, got, err), smRolled(t, plain, perr); !slices.Equal(a.Modifiers(), b.Modifiers()) {
				t.Fatalf("recording and Noop telemetry gave %v and %v", a.Modifiers(), b.Modifiers())
			}
			want := map[string]any{"srd5e.combat.operation": "roll_spell_bonus_dice", "srd5e.outcome": "success", "srd5e.combat.result": c.result, "srd5e.combat.result_class": "applied"}
			spans := rec.Tracer.SpansNamed("srd5e.combat.roll_spell_bonus_dice")
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 || !mapsEqual(attrMap(spans[0].Attrs), want) {
				t.Fatalf("spans = %+v, want one ended span with %v [%s%s]", spans, want, spellModifiersSpec, c.requirement)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !mapsEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v", ops, want)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")) + len(rec.Meter.MeasurementsNamed("srd5e.combat.events")); n != 0 {
				t.Errorf("errors/events samples = %d, want 0", n)
			}
			if d := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(d) != 1 {
				t.Errorf("duration samples = %d, want one", len(d))
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-023, FR-084 - the result returns copies of its modifiers and rolls.
// Path: unhappy (immutability).
func TestSpellBonusDiceRolledReturnsCopies(t *testing.T) {
	t.Parallel()
	res, err := smBonusUse(t, dicetest.NewFixedRoller(2)).Run(context.Background(), combat.RollSpellBonusDiceCommand{Dice: []combat.BonusDie{smDie(1, 4, false)}})
	r := smRolled(t, res, err)
	m, rolls := r.Modifiers(), r.Rolls()
	m[0], rolls[0] = modifier.New(9), dice.Result{}
	if r.Modifiers()[0] != modifier.New(2) || r.Rolls()[0].Total() != 2 {
		t.Error("an accessor returned the result's own slice")
	}
}

// ExampleRollSpellBonusDice rolls the Guidance die of a Stealth check with a fixed roller and adds it to a check.
func ExampleRollSpellBonusDice() {
	use, _ := combat.NewRollSpellBonusDice(dicetest.NewFixedRoller(3))
	guidance, _ := combat.NewBonusDie(dice.MustExpression(1, dice.D4(), 0), false)
	res, _ := use.Run(context.Background(), combat.RollSpellBonusDiceCommand{Dice: []combat.BonusDie{guidance}})
	rolled := res.(combat.SpellBonusDiceRolled)
	fmt.Println(rolled.Modifiers()[0].Value())
	// Output: 3
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-013, FR-033 - a bonus die applies to a Death Saving Throw: the rolled modifier goes in the Modifier parameter of MakeDeathSavingThrow, so a d20 of 8 plus a die of 2 reaches 10 and succeeds while 8 alone fails, and a natural 1 keeps its own rule (two failures) whatever the die (SRD 5.2.1, "Death Saving Throws", p. 17; "Bless", p. 113). The source of a Bless or Bane die is not modelled here.
// Path: per case.
func TestBonusDieAppliesToADeathSavingThrow(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		natural, die int
		bonus        bool
		want         combat.DeathSavingThrowOutcome
	}{
		{"8 and a die of 2 reaches 10", 8, 2, true, combat.DeathSavingThrowSuccess},
		{"8 alone fails", 8, 2, false, combat.DeathSavingThrowFailure},
		{"a natural 1 stays two failures with a die of 4", 1, 4, true, combat.DeathSavingThrowTwoFailures},
		{"a Bane die takes 10 to 7", 10, 3, true, combat.DeathSavingThrowFailure},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			mod := modifier.New(0)
			if c.bonus {
				negative := c.name == "a Bane die takes 10 to 7"
				res, err := smBonusUse(t, dicetest.NewFixedRoller(c.die)).Run(context.Background(), combat.RollSpellBonusDiceCommand{Dice: []combat.BonusDie{smDie(1, 4, negative)}})
				mod = smRolled(t, res, err).Modifiers()[0]
			}
			cmd := luMust(combat.NewMakeDeathSavingThrowCommand(subject(), dyingStatus(t, 20, 0, 0), modifier.Normal, mod))
			use := luMust(combat.NewMakeDeathSavingThrow(dicetest.NewFixedRoller(c.natural)))
			out, err := use.Run(context.Background(), cmd)
			applied, ok := out.(combat.DeathSavingThrowApplied)
			if err != nil || !ok || applied.Outcome() != c.want {
				t.Errorf("Death Saving Throw = %#v, %v, want %v [%sFR-013]", out, err, c.want, spellModifiersSpec)
			}
		})
	}
}
