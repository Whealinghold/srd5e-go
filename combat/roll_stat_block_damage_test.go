package combat_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func sbRollUse(tb testing.TB, r combat.Roller, opts ...combat.Option) combat.RollStatBlockDamage {
	tb.Helper()
	return luMust(combat.NewRollStatBlockDamage(r, opts...))
}

func sbRolled(tb testing.TB, res combat.RollStatBlockDamageResult, err error) []combat.Damage {
	tb.Helper()
	r, ok := res.(combat.StatBlockDamageRolled)
	if err != nil || !ok {
		tb.Fatalf("Run = %#v, %v, want StatBlockDamageRolled", res, err)
	}
	return r.Damage()
}

func sbAssassinDamage() []content.StatBlockDamage {
	return []content.StatBlockDamage{sbDamage(7, sbExpr(1, 6, 4), content.DamagePiercing), sbDamage(17, sbExpr(5, 6, 0), content.DamagePoison)}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-044, FR-045, table AT-1 "number chosen", "dice chosen", "two instances, mixed" - for each instance in order, one roll for a
// dice choice and none for a number, the printed flat modifier added, the result floored at 0 and returned as a Damage with the instance's type; a number instance returns the
// printed number (SRD 5.2.1, "Damage Notation", p. 257; "Damage Rolls", p. 16).
// Path: per case.
func TestRollStatBlockDamage(t *testing.T) {
	t.Parallel()
	bash := sbDamage(5, sbExpr(1, 6, 2), content.DamageBludgeoning)
	negative := sbDamage(2, sbExpr(1, 4, -5), content.DamageFire)
	cases := []struct {
		name, requirement, path string
		damage                  []content.StatBlockDamage
		choices                 combat.DamageNotationChoices
		faces                   []int
		wantRequests            []string
		want                    []combat.Damage
	}{
		{"number chosen", `FR-026; AT-1 "number chosen"`, "happy", []content.StatBlockDamage{bash}, sbNumbers(t), nil, nil, []combat.Damage{combat.MustDamage(5, content.DamageBludgeoning)}},
		{"dice chosen", `FR-045; AT-1 "dice chosen"`, "happy", []content.StatBlockDamage{bash}, sbDiceChoices(t), []int{4}, []string{"1d6+2"}, []combat.Damage{combat.MustDamage(6, content.DamageBludgeoning)}},
		{"two instances, mixed", `FR-045; AT-1 "two instances, mixed"`, "unhappy (rule interaction)", sbAssassinDamage(), luMust(combat.NewDamageNotationChoices(combat.DamageNotationNumber, combat.DamageNotationDice)),
			[]int{6, 5, 4, 3, 2}, []string{"5d6"}, []combat.Damage{combat.MustDamage(7, content.DamagePiercing), combat.MustDamage(20, content.DamagePoison)}},
		{"two instances, dice in printed order", `FR-045`, "unhappy (rule interaction)", sbAssassinDamage(), sbDiceChoices(t),
			[]int{3, 1, 1, 1, 1, 1}, []string{"1d6+4", "5d6"}, []combat.Damage{combat.MustDamage(7, content.DamagePiercing), combat.MustDamage(5, content.DamagePoison)}},
		{"dice first then number", `FR-045`, "unhappy (rule interaction)", sbAssassinDamage(), luMust(combat.NewDamageNotationChoices(combat.DamageNotationDice, combat.DamageNotationNumber)),
			[]int{6}, []string{"1d6+4"}, []combat.Damage{combat.MustDamage(10, content.DamagePiercing), combat.MustDamage(17, content.DamagePoison)}},
		{"a negative total floors at zero", `FR-045; "Damage Rolls", p. 16`, "unhappy (boundary)", []content.StatBlockDamage{negative}, sbDiceChoices(t), []int{1}, []string{"1d4-5"}, []combat.Damage{combat.MustDamage(0, content.DamageFire)}},
		{"a total of exactly zero", `FR-045`, "unhappy (boundary)", []content.StatBlockDamage{negative}, sbDiceChoices(t), []int{4}, []string{"1d4-5"}, []combat.Damage{combat.MustDamage(0, content.DamageFire)}},
		{"a total of one", `FR-045`, "unhappy (boundary)", []content.StatBlockDamage{sbDamage(2, sbExpr(1, 4, -3), content.DamageFire)}, sbDiceChoices(t), []int{4}, []string{"1d4-3"}, []combat.Damage{combat.MustDamage(1, content.DamageFire)}},
		{"a printed number of zero", `FR-026`, "unhappy (boundary)", []content.StatBlockDamage{sbDamage(0, dice.Expression{}, content.DamageCold)}, sbNumbers(t), nil, nil, []combat.Damage{combat.MustDamage(0, content.DamageCold)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := sbRollUse(t, roller).Run(context.Background(), combat.RollStatBlockDamageCommand{Damage: c.damage, Choices: c.choices})
			if got := sbRolled(t, res, err); !slices.Equal(got, c.want) || !slices.Equal(notations(roller), c.wantRequests) {
				t.Errorf("damage %v from requests %v, want %v from %v [%s]", got, notations(roller), c.want, c.wantRequests, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-033, table SV-1 "damage rolled once" - the damage of one use is rolled once and shared by every target that saves: one call,
// one roll, and the one Damage list serves each target of the save resolver.
// Path: unhappy (rule interaction).
func TestStatBlockDamageIsRolledOnceForAllTargets(t *testing.T) {
	t.Parallel()
	part := sbSavePart("Cold Breath", sbSave(ability.Wisdom, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil, sbDamage(7, sbExpr(2, 6, 0), content.DamageCold)), sbClause(content.ClauseSuccess, content.HalfDamage{}))
	plan := luMust(combat.PlanStatBlockSaveEffect(part))
	roller := dicetest.NewFixedRoller(5, 6)
	res, err := sbRollUse(t, roller).Run(context.Background(), combat.RollStatBlockDamageCommand{Damage: plan.Damage(), Choices: sbDiceChoices(t)})
	rolled := sbRolled(t, res, err)
	saves := map[combat.CombatantID]d20.CheckResult{"a": sbSaveRoll(t, 3, sbDC), "b": sbSaveRoll(t, 20, sbDC), "c": sbSaveRoll(t, 4, sbDC)}
	out := sbResolve(t, part, saves, nil, "a", "b", "c")
	dealt := make([]int, len(out))
	for i, r := range out {
		var adjustments []combat.DamageAdjustment
		if sbApplies(t, r).DamageAdjustment() == combat.SaveEffectDamageHalved {
			adjustments = []combat.DamageAdjustment{combat.HalfDamage{}}
		}
		st := status(t, 1000, 1000, 0, combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
		tr, err := st.ApplyDamage(subject(), rolled, adjustments, combat.OtherSource, combat.NormalHit, nil)
		dealt[i] = applied(t, tr, err).Resolutions()[0].Dealt()
	}
	if len(roller.Requests()) != 1 || !slices.Equal(dealt, []int{11, 5, 11}) {
		t.Errorf("rolls = %v, damage dealt %v, want one roll of 2d6 and 11, 5, 11 for the three targets", notations(roller), dealt)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-046, errors-and-results FR-031, FR-034 - a roller that fails (also on the second roll), panics or answers another
// expression is a *srderr.PortError for port dice with a nil result, recorded once: error status on the span, one srd5e.combat.errors count with error.type dice_roller_error,
// no operations count and no log; errors.Is finds the cause.
// Path: unhappy (failure of collaborator).
func TestRollStatBlockDamagePortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	secondFails := func() combat.Roller {
		n := 0
		return dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
			if n++; n == 2 {
				return dice.Result{}, cause
			}
			return dicetest.NewMaximumRoller().Roll(e)
		})
	}()
	rollers := map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller(), "fails on the second roll": secondFails}
	for name, roller := range rollers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			res, err := sbRollUse(t, roller, opts...).Run(context.Background(), combat.RollStatBlockDamageCommand{Damage: sbAssassinDamage(), Choices: sbDiceChoices(t)})
			envPortFailure(t, err)
			if res != nil {
				t.Errorf("result = %#v, want nil", res)
			}
			if name == "fails" || name == "fails on the second roll" {
				if !errors.Is(err, cause) {
					t.Errorf("errors.Is(err, cause) = false for %v", err)
				}
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.roll_stat_block_damage")
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

// Requirement: doc/specs/monster-stat-block-actions.md FR-044, FR-027 - a nil roller, the zero use case, no damage instance, choices that do not fit, a dice choice for an instance
// with no dice and a damage type printed as text are invalid input before any signal or roll; a cancelled context is returned unwrapped before any signal; the command is not changed.
// Path: per case.
func TestRollStatBlockDamageInvalidUse(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewRollStatBlockDamage(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewRollStatBlockDamage(nil) = %v, want ErrInvalidInput", err)
	}
	described := luMust(content.NewStatBlockDamage(3, dice.Expression{}, luMust(content.NewDescribedDamageType("the chosen type")), false))
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(1, 2, 3, 4, 5, 6)
	use := sbRollUse(t, roller, opts...)
	one := luMust(combat.NewDamageNotationChoices(combat.DamageNotationDice))
	cases := []struct {
		name string
		cmd  combat.RollStatBlockDamageCommand
	}{
		{"no damage", combat.RollStatBlockDamageCommand{Choices: sbNumbers(t)}},
		{"no choices", combat.RollStatBlockDamageCommand{Damage: sbAssassinDamage()}},
		{"too few choices", combat.RollStatBlockDamageCommand{Damage: sbAssassinDamage(), Choices: one}},
		{"dice for no dice", combat.RollStatBlockDamageCommand{Damage: []content.StatBlockDamage{sbDamage(1, dice.Expression{}, content.DamageSlashing)}, Choices: sbDiceChoices(t)}},
		{"damage type as text", combat.RollStatBlockDamageCommand{Damage: []content.StatBlockDamage{described}, Choices: sbNumbers(t)}},
	}
	for _, c := range cases {
		if res, err := use.Run(context.Background(), c.cmd); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
			t.Errorf("%s: %v, %v, want nil and ErrInvalidInput", c.name, res, err)
		}
	}
	valid := combat.RollStatBlockDamageCommand{Damage: sbAssassinDamage(), Choices: sbDiceChoices(t)}
	if res, err := (combat.RollStatBlockDamage{}).Run(context.Background(), valid); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero use case: %v, %v, want nil and ErrInvalidInput", res, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := use.Run(cancelled, valid); !errors.Is(err, context.Canceled) || res != nil {
		t.Errorf("cancelled: %v, %v, want nil and context.Canceled", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 || len(roller.Requests()) != 0 {
		t.Error("an invalid call recorded a signal or rolled")
	}
	if !slices.EqualFunc(valid.Damage, sbAssassinDamage(), func(a, b content.StatBlockDamage) bool { return a == b }) {
		t.Error("the command changed")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md plan P13 - a damage instance that is the zero value (outside the range combat relies on) is an Invalid result found before any roll, whatever
// the instances before it: class invalid, error status, result content_number_out_of_range, no errors count and no roll.
// Path: unhappy (invalid input).
func TestRollStatBlockDamageInvalidInstanceRollsNothing(t *testing.T) {
	t.Parallel()
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(1, 2, 3)
	cmd := combat.RollStatBlockDamageCommand{
		Damage:  []content.StatBlockDamage{sbDamage(7, sbExpr(1, 6, 4), content.DamagePiercing), {}},
		Choices: luMust(combat.NewDamageNotationChoices(combat.DamageNotationDice, combat.DamageNotationNumber))}
	res, err := sbRollUse(t, roller, opts...).Run(context.Background(), cmd)
	if _, ok := res.(combat.StatBlockDamageInvalid); err != nil || !ok {
		t.Fatalf("Run = %#v, %v, want StatBlockDamageInvalid", res, err)
	}
	if len(roller.Requests()) != 0 {
		t.Errorf("rolled %v before finding the invalid instance", notations(roller))
	}
	spans := rec.Tracer.SpansNamed("srd5e.combat.roll_stat_block_damage")
	want := map[string]any{"srd5e.combat.operation": "roll_stat_block_damage", "srd5e.outcome": "failure", "srd5e.combat.result": "content_number_out_of_range", "srd5e.combat.result_class": "invalid"}
	if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 0 || !mapsEqual(attrMap(spans[0].Attrs), want) {
		t.Errorf("spans = %+v, want one error-status span with %v and no notation", spans, want)
	}
	if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
		t.Errorf("errors counted = %d, want 0", n)
	}
	checkTelemetryHygiene(t, rec)
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-047, section 7 - the shared signals under roll_stat_block_damage: class applied, result stat_block_damage_rolled (at least one die
// rolled) or stat_block_damage_fixed (numbers only), and the extra srd5e.combat.damage_notation (number, dice or mixed) on the span and the operations counter only; no event, no log, no
// id or amount in an attribute; the result is the same with the zero-value Noop telemetry and with the recording double.
// Path: per case.
func TestRollStatBlockDamageSignals(t *testing.T) {
	t.Parallel()
	mixed := luMust(combat.NewDamageNotationChoices(combat.DamageNotationNumber, combat.DamageNotationDice))
	cases := []struct {
		name, requirement, path string
		choices                 combat.DamageNotationChoices
		faces                   []int
		result, notation        string
	}{
		{"numbers only", "FR-047", "happy", sbNumbers(t), nil, "stat_block_damage_fixed", "number"},
		{"dice only", "FR-047", "happy", sbDiceChoices(t), []int{1, 1, 1, 1, 1, 1}, "stat_block_damage_rolled", "dice"},
		{"mixed", "FR-047", "unhappy (rule interaction)", mixed, []int{1, 1, 1, 1, 1}, "stat_block_damage_rolled", "mixed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := combat.RollStatBlockDamageCommand{Damage: sbAssassinDamage(), Choices: c.choices}
			rec, opts := recording()
			got, err := sbRollUse(t, dicetest.NewFixedRoller(c.faces...), opts...).Run(context.Background(), cmd)
			plain, perr := sbRollUse(t, dicetest.NewFixedRoller(c.faces...)).Run(context.Background(), cmd)
			if err != nil || perr != nil || !slices.Equal(sbRolled(t, got, err), sbRolled(t, plain, perr)) {
				t.Fatalf("recording and Noop telemetry gave %v, %v and %v, %v", got, err, plain, perr)
			}
			want := map[string]any{"srd5e.combat.operation": "roll_stat_block_damage", "srd5e.outcome": "success", "srd5e.combat.result": c.result,
				"srd5e.combat.result_class": "applied", "srd5e.combat.damage_notation": c.notation}
			spans := rec.Tracer.SpansNamed("srd5e.combat.roll_stat_block_damage")
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 || !mapsEqual(attrMap(spans[0].Attrs), want) {
				t.Fatalf("spans = %+v, want one ended span with %v [%s]", spans, want, statBlockSpec+c.requirement)
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

// Requirement: doc/specs/monster-stat-block-actions.md FR-049 - the result returns a copy of the damage, and the same command and rolls give equal results.
// Path: unhappy (immutability).
func TestStatBlockDamageRolledReturnsACopy(t *testing.T) {
	t.Parallel()
	res, err := sbRollUse(t, dicetest.NewFixedRoller()).Run(context.Background(), combat.RollStatBlockDamageCommand{Damage: sbAssassinDamage(), Choices: sbNumbers(t)})
	r := res.(combat.StatBlockDamageRolled)
	if err != nil {
		t.Fatal(err)
	}
	d := r.Damage()
	d[0] = combat.MustDamage(1, content.DamageFire)
	if r.Damage()[0] != combat.MustDamage(7, content.DamagePiercing) {
		t.Error("Damage returned the result's own slice")
	}
}

// FuzzRollStatBlockDamage checks that no choice list, instance count or roller face makes the use case panic, that it returns exactly one of a result and an
// error, and that a rolled amount is never negative.
func FuzzRollStatBlockDamage(f *testing.F) {
	f.Add([]byte{1, 2}, 2, 3)
	f.Add([]byte{}, 0, 0)
	f.Add([]byte{2, 2, 2}, 1, 9)
	f.Fuzz(func(t *testing.T, raw []byte, instances, face int) {
		if instances < 0 || instances > 6 {
			return
		}
		damage := make([]content.StatBlockDamage, instances)
		for i := range damage {
			damage[i] = sbDamage(i, sbExpr(1+i%2, 6, -3), content.DamageFire)
		}
		choices := make([]combat.DamageNotationChoice, len(raw))
		for i, b := range raw {
			choices[i] = combat.DamageNotationChoice(b%2 + 1)
		}
		c, err := combat.NewDamageNotationChoices(choices...)
		if err != nil {
			c = combat.DamageNotationChoices{}
		}
		res, err := sbRollUse(t, dicetest.NewMaximumRoller()).Run(context.Background(), combat.RollStatBlockDamageCommand{Damage: damage, Choices: c})
		if (res == nil) == (err == nil) {
			t.Fatalf("Run = %v, %v, want exactly one of a result and an error", res, err)
		}
		if r, ok := res.(combat.StatBlockDamageRolled); ok {
			for _, d := range r.Damage() {
				if d.Amount() < 0 {
					t.Fatalf("negative damage %v", d)
				}
			}
		}
	})
}

// ExampleRollStatBlockDamage rolls the dice of a part's damage with a fixed roller.
func ExampleRollStatBlockDamage() {
	slam := mustExampleDamage()
	choices, _ := combat.AllDamageNotation(combat.DamageNotationDice)
	use, _ := combat.NewRollStatBlockDamage(dicetest.NewFixedRoller(4))
	res, _ := use.Run(context.Background(), combat.RollStatBlockDamageCommand{Damage: []content.StatBlockDamage{slam}, Choices: choices})
	for _, d := range res.(combat.StatBlockDamageRolled).Damage() {
		fmt.Println(d.Amount(), d.Type())
	}
	// Output: 6 Bludgeoning
}

func mustExampleDamage() content.StatBlockDamage {
	ref, _ := content.NewSpecificDamageType(content.DamageBludgeoning)
	d, _ := content.NewStatBlockDamage(5, dice.MustExpression(1, dice.D6(), 2), ref, false)
	return d
}
