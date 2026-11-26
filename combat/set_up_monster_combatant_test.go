package combat_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func msUse(t *testing.T, r combat.Roller, opts ...combat.Option) combat.SetUpMonsterCombatant {
	t.Helper()
	return luMust(combat.NewSetUpMonsterCombatant(r, opts...))
}

var (
	msBat = dice.MustExpression(1, dice.D4(), -1)
	msRat = dice.MustExpression(2, dice.D6(), 0)
)

// Requirement: doc/specs/monster-combatant-setup.md FR-037, FR-038, FR-048, FR-049 - with UseAverageHitPoints the use case makes no request of the roller and returns the setup with the printed average; with RollHitPointDice exactly one request for the printed Hit Point Dice including the flat modifier, wrapped as RolledMonsterHitPoints; the options of the command reach the setup (SRD 5.2.1, "Hit Points", p. 255).
// Path: per case.
func TestSetUpMonsterCombatant(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		cmd                     combat.SetUpMonsterCombatantCommand
		faces                   []int
		wantRequests            []dice.Expression
		wantMaximum             int
		wantRolled              bool
	}{
		{"average", "FR-038", "happy", combat.SetUpMonsterCombatantCommand{Monster: msHitPoints(t, 7, msRat), HitPoints: combat.UseAverageHitPoints{}}, []int{6, 6}, nil, 7, false},
		{"roll", "FR-038", "happy", combat.SetUpMonsterCombatantCommand{Monster: msHitPoints(t, 7, msRat), HitPoints: combat.RollHitPointDice{}}, []int{3, 4}, []dice.Expression{msRat}, 7, true},
		{"roll with a flat modifier", "FR-038", "happy", combat.SetUpMonsterCombatantCommand{Monster: msMonster(t, nil), HitPoints: combat.RollHitPointDice{}}, slicesRepeat(10, 20),
			[]dice.Expression{dice.MustExpression(20, dice.D10(), 40)}, 240, true},
		{"roll raised to one", "FR-010 FR-038", "unhappy (boundary)", combat.SetUpMonsterCombatantCommand{Monster: msHitPoints(t, 1, msBat), HitPoints: combat.RollHitPointDice{}}, []int{1}, []dice.Expression{msBat}, 1, true},
		{"size chosen and options passed on", "FR-037", "happy", combat.SetUpMonsterCombatantCommand{Monster: msMonster(t, msTwoSizes), Size: content.SizeSmall, HitPoints: combat.UseAverageHitPoints{},
			ZeroHitPointsRule: combat.FallsUnconscious, InitiativeMode: combat.MonsterInitiativeFixed, ExtraAffinities: combat.ResistantToAllDamage()}, nil, nil, 150, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := setupSpec + c.requirement
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := msUse(t, roller).Run(context.Background(), c.cmd)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, req)
			}
			applied, ok := res.(combat.SetUpMonsterCombatantApplied)
			if !ok {
				t.Fatalf("result = %#v, want Applied [%s]", res, req)
			}
			if !slices.Equal(roller.Requests(), c.wantRequests) {
				t.Errorf("requests = %v, want %v [%s]", roller.Requests(), c.wantRequests, req)
			}
			if got := applied.Setup().HitPointStatus().HitPoints().Maximum(); got != c.wantMaximum {
				t.Errorf("maximum = %d, want %d [%s]", got, c.wantMaximum, req)
			}
			if _, rolled := applied.HitPoints().(combat.RolledMonsterHitPoints); rolled != c.wantRolled {
				t.Errorf("HitPoints() = %T, want rolled %v [%s]", applied.HitPoints(), c.wantRolled, req)
			}
			rule, mode := c.cmd.ZeroHitPointsRule, c.cmd.InitiativeMode
			if rule == 0 {
				rule = combat.DiesInstantly
			}
			if mode == 0 {
				mode = combat.MonsterInitiativeRolled
			}
			want := luMust(combat.NewMonsterSetup(c.cmd.Monster, applied.HitPoints(), combat.WithSize(c.cmd.Size), combat.WithZeroHitPointsRule(rule),
				combat.WithInitiativeMode(mode), combat.WithExtraAffinities(c.cmd.ExtraAffinities)))
			if !applied.Setup().Equal(want) {
				t.Errorf("the setup differs from NewMonsterSetup with the same inputs [%s]", req)
			}
		})
	}
}

func slicesRepeat(face, n int) []int { return slices.Repeat([]int{face}, n) }

// Requirement: doc/specs/monster-combatant-setup.md FR-037, FR-039, FR-042 - a size problem, or a monster outside the ranges combat relies on (the zero Monster included), is an Invalid result found before any roll: no request, no die consumed, and the result holds the problems in the order of the enum, as a copy.
// Path: per case.
func TestSetUpMonsterCombatantInvalidConsumesNoDie(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		cmd                     combat.SetUpMonsterCombatantCommand
		want                    []combat.MonsterSetupProblem
	}{
		{"size required", "FR-039", "unhappy (invalid input)", combat.SetUpMonsterCombatantCommand{Monster: msMonster(t, msTwoSizes), HitPoints: combat.RollHitPointDice{}}, []combat.MonsterSetupProblem{combat.SizeChoiceRequired}},
		{"size not offered", "FR-039", "unhappy (invalid input)", combat.SetUpMonsterCombatantCommand{Monster: msMonster(t, nil), Size: content.SizeTiny, HitPoints: combat.RollHitPointDice{}}, []combat.MonsterSetupProblem{combat.SizeNotOffered}},
		{"the zero monster", "FR-039", "unhappy (missing content)", combat.SetUpMonsterCombatantCommand{HitPoints: combat.RollHitPointDice{}}, []combat.MonsterSetupProblem{combat.ContentNumberOutOfRange}},
		{"size and content", "FR-039", "unhappy (invalid input)", combat.SetUpMonsterCombatantCommand{Monster: content.Monster{}, Size: content.SizeTiny, HitPoints: combat.UseAverageHitPoints{}},
			[]combat.MonsterSetupProblem{combat.SizeNotOffered, combat.ContentNumberOutOfRange}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(1, 2, 3)
			res, err := msUse(t, roller).Run(context.Background(), c.cmd)
			inv, ok := res.(combat.SetUpMonsterCombatantInvalid)
			if err != nil || !ok || !slices.Equal(inv.Problems(), c.want) {
				t.Fatalf("Run = %#v, %v, want Invalid with %v [%s]", res, err, c.want, setupSpec+c.requirement)
			}
			if n, _ := roller.Remaining(); n != 3 || len(roller.Requests()) != 0 {
				t.Errorf("remaining %d, requests %v, want no die consumed", n, roller.Requests())
			}
			p := inv.Problems()
			p[0] = 0
			if !slices.Equal(inv.Problems(), c.want) {
				t.Error("Problems is not a copy")
			}
		})
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-040, errors-and-results FR-031, FR-034, FR-051 - a roller that fails, panics or answers another expression is a *srderr.PortError for port dice with a nil result, recorded once (error status and one srd5e.combat.errors count with error.type dice_roller_error, no operations count, no clamp count, no log); errors.Is finds the cause.
// Path: unhappy (failure of collaborator).
func TestSetUpMonsterCombatantPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	rollers := map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller()}
	for name, roller := range rollers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			cmd := combat.SetUpMonsterCombatantCommand{Monster: msMonster(t, nil), HitPoints: combat.RollHitPointDice{}}
			res, err := msUse(t, roller, opts...).Run(context.Background(), cmd)
			envPortFailure(t, err)
			if res != nil {
				t.Errorf("result = %#v, want nil", res)
			}
			if name == "fails" && !errors.Is(err, cause) {
				t.Errorf("errors.Is(err, cause) = false for %v", err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.set_up_monster_combatant")
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Errorf("spans = %+v, want one ended span with one recorded error", spans)
			}
			if got := values(measured(rec, "srd5e.combat.errors"), "error.type"); !slices.Equal(got, []string{"dice_roller_error"}) {
				t.Errorf("errors counted = %v, want one dice_roller_error", got)
			}
			for _, n := range []string{"srd5e.combat.operations", "srd5e.combat.events", "srd5e.combat.monster_hit_points_clamped"} {
				if m := rec.Meter.MeasurementsNamed(n); len(m) != 0 {
					t.Errorf("%s = %v, want no sample after a port failure", n, m)
				}
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-037, FR-041 - a nil roller, the zero use case, a command with no Hit Points method and an undefined option are invalid input before any signal; a cancelled context is returned unwrapped before any signal; the command is not changed.
// Path: unhappy (invalid input, cancellation, immutability).
func TestSetUpMonsterCombatantInvalidUse(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewSetUpMonsterCombatant(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewSetUpMonsterCombatant(nil) = %v, want ErrInvalidInput", err)
	}
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(3, 4)
	use := msUse(t, roller, opts...)
	m := msHitPoints(t, 7, msRat)
	ok := combat.SetUpMonsterCombatantCommand{Monster: m, HitPoints: combat.RollHitPointDice{}}
	bad := map[string]combat.SetUpMonsterCombatantCommand{
		"no method":          {Monster: m},
		"undefined rule":     {Monster: m, HitPoints: combat.RollHitPointDice{}, ZeroHitPointsRule: 9},
		"undefined mode":     {Monster: m, HitPoints: combat.RollHitPointDice{}, InitiativeMode: 9},
		"negative rule":      {Monster: m, HitPoints: combat.RollHitPointDice{}, ZeroHitPointsRule: -1},
		"zero use case only": ok,
	}
	for name, cmd := range bad {
		u := use
		if name == "zero use case only" {
			u = combat.SetUpMonsterCombatant{}
		}
		if res, err := u.Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
			t.Errorf("%s: %v, %v, want nil and ErrInvalidInput", name, res, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := use.Run(cancelled, ok); !errors.Is(err, context.Canceled) || res != nil {
		t.Errorf("cancelled: %v, %v, want nil and context.Canceled", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 || len(roller.Requests()) != 0 {
		t.Error("an invalid call recorded a signal or rolled")
	}
	if !ok.Monster.Equal(m) {
		t.Error("the command changed")
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-041, FR-044 - the same command and the same roll give an equal result, with the zero-value Noop telemetry and with the recording double; two different rolls give different results; Equal compares the setup and the Hit Points choice.
// Path: unhappy (determinism).
func TestSetUpMonsterCombatantIsDeterministicAndTelemetryNeverChangesIt(t *testing.T) {
	t.Parallel()
	cmd := combat.SetUpMonsterCombatantCommand{Monster: msHitPoints(t, 7, msRat), HitPoints: combat.RollHitPointDice{}}
	run := func(opts []combat.Option, faces ...int) combat.SetUpMonsterCombatantApplied {
		res, err := msUse(t, dicetest.NewFixedRoller(faces...), opts...).Run(context.Background(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res.(combat.SetUpMonsterCombatantApplied)
	}
	_, opts := recording()
	plain, again, recorded, other := run(nil, 3, 4), run(nil, 3, 4), run(opts, 3, 4), run(nil, 5, 6)
	if !plain.Equal(again) || !plain.Equal(recorded) {
		t.Error("equal commands and rolls gave unequal results")
	}
	if plain.Equal(other) {
		t.Error("different rolls gave equal results")
	}
	avg, err := msUse(t, dicetest.NewFixedRoller()).Run(context.Background(), combat.SetUpMonsterCombatantCommand{Monster: cmd.Monster, HitPoints: combat.UseAverageHitPoints{}})
	if err != nil {
		t.Fatal(err)
	}
	a := avg.(combat.SetUpMonsterCombatantApplied)
	if a.Equal(plain) || plain.Equal(a) || !a.Equal(a) {
		t.Error("a printed average and a roll of the same total compare equal, or a result differs from itself")
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-042, FR-043, FR-048 to FR-051, table 4.10a - the shared signals under set_up_monster_combatant: class applied with result set_up_average or set_up_rolled and the method extra on the span and the operations count only, class invalid with the problem's result, error status and no errors count; srd5e.combat.monster_hit_points_clamped once for a raised total only; no event, no log, and no attribute carries a MonsterID or a Hit Point value.
// Path: per case.
func TestSetUpMonsterCombatantSignals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		cmd                     combat.SetUpMonsterCombatantCommand
		faces                   []int
		result, class, outcome  string
		method                  string
		clamped                 int
	}{
		{"average", "FR-048 FR-049", "happy", combat.SetUpMonsterCombatantCommand{Monster: msHitPoints(t, 1, msBat), HitPoints: combat.UseAverageHitPoints{}}, nil, "set_up_average", "applied", "success", "average", 0},
		{"rolled", "FR-048 FR-049", "happy", combat.SetUpMonsterCombatantCommand{Monster: msHitPoints(t, 7, msRat), HitPoints: combat.RollHitPointDice{}}, []int{3, 4}, "set_up_rolled", "applied", "success", "rolled", 0},
		{"rolled and raised to one", "FR-050", "unhappy (boundary)", combat.SetUpMonsterCombatantCommand{Monster: msHitPoints(t, 1, msBat), HitPoints: combat.RollHitPointDice{}}, []int{1}, "set_up_rolled", "applied", "success", "rolled", 1},
		{"rolled to one is not raised", "FR-050", "unhappy (boundary)", combat.SetUpMonsterCombatantCommand{Monster: msHitPoints(t, 1, msBat), HitPoints: combat.RollHitPointDice{}}, []int{2}, "set_up_rolled", "applied", "success", "rolled", 0},
		{"size choice required", "FR-042 FR-048", "unhappy (telemetry)", combat.SetUpMonsterCombatantCommand{Monster: msMonster(t, msTwoSizes), HitPoints: combat.RollHitPointDice{}}, nil, "size_choice_required", "invalid", "failure", "", 0},
		{"size not offered", "FR-042 FR-048", "unhappy (telemetry)", combat.SetUpMonsterCombatantCommand{Monster: msMonster(t, nil), Size: content.SizeTiny, HitPoints: combat.RollHitPointDice{}}, nil, "size_not_offered", "invalid", "failure", "", 0},
		{"content out of range", "FR-042 FR-048", "unhappy (telemetry)", combat.SetUpMonsterCombatantCommand{HitPoints: combat.RollHitPointDice{}}, nil, "content_number_out_of_range", "invalid", "failure", "", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := setupSpec + c.requirement
			rec, opts := recording()
			if _, err := msUse(t, dicetest.NewFixedRoller(c.faces...), opts...).Run(context.Background(), c.cmd); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"srd5e.combat.operation": "set_up_monster_combatant", "srd5e.outcome": c.outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			if c.method != "" {
				want["srd5e.combat.monster_hit_points_method"] = c.method
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.set_up_monster_combatant")
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != (c.class == "invalid") || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, error status only when invalid, no recorded error [%s]", spans, req)
			}
			if sa := attrMap(spans[0].Attrs); !mapsEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [%s]", sa, want, req)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !mapsEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [%s]", ops, want, req)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")) + len(rec.Meter.MeasurementsNamed("srd5e.combat.events")); n != 0 {
				t.Errorf("errors/events samples = %d, want 0 (FR-042, FR-043)", n)
			}
			clamps := rec.Meter.MeasurementsNamed("srd5e.combat.monster_hit_points_clamped")
			if len(clamps) != c.clamped || (len(clamps) == 1 && (clamps[0].Value != 1 || len(clamps[0].Attrs) != 0)) {
				t.Errorf("clamp samples = %+v, want %d of 1 with no attributes [%s]", clamps, c.clamped, req)
			}
			checkTelemetryHygiene(t, rec)
			for _, m := range rec.Meter.Measurements() {
				for _, a := range m.Attrs {
					if s, ok := a.Value().(string); ok && (s == "fixture_beast" || s == "1" || s == "7") {
						t.Errorf("%s attribute %s = %q carries an id or a Hit Point value (FR-051)", m.Name, a.Key(), s)
					}
				}
			}
		})
	}
}

func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range b {
		if a[k] != v {
			return false
		}
	}
	return true
}

// Requirement: doc/rules/testing.md section 10 - the use case never panics for any combination of size, method and option values, and returns either a result or an error, never both.
// Path: unhappy (invalid input).
func FuzzSetUpMonsterCombatant(f *testing.F) {
	f.Add(0, 0, 0, 0, 0)
	f.Add(3, 1, 1, 2, 1)
	f.Add(-1, 2, 99, -4, 0)
	f.Fuzz(func(t *testing.T, size, method, rule, mode, sizes int) {
		m := msMonster(t, func(f *content.MonsterFields) {
			if sizes%2 != 0 {
				msTwoSizes(f)
			}
		})
		var hp combat.MonsterHitPointsMethod
		switch method % 3 {
		case 0:
			hp = combat.UseAverageHitPoints{}
		case 1:
			hp = combat.RollHitPointDice{}
		}
		res, err := msUse(t, dicetest.NewMaximumRoller()).Run(context.Background(), combat.SetUpMonsterCombatantCommand{
			Monster: m, Size: content.CreatureSize(size), HitPoints: hp, ZeroHitPointsRule: combat.ZeroHitPointsRule(rule), InitiativeMode: combat.MonsterInitiativeMode(mode)})
		if (res == nil) == (err == nil) {
			t.Fatalf("Run = %v, %v, want exactly one of a result and an error", res, err)
		}
	})
}

// ExampleSetUpMonsterCombatant sets up one creature of a stat block with rolled Hit Points.
func ExampleSetUpMonsterCombatant() {
	m := msHitPoints(&testing.T{}, 7, msRat)
	use, _ := combat.NewSetUpMonsterCombatant(dicetest.NewFixedRoller(3, 4))
	res, _ := use.Run(context.Background(), combat.SetUpMonsterCombatantCommand{Monster: m, HitPoints: combat.RollHitPointDice{}})
	setup := res.(combat.SetUpMonsterCombatantApplied).Setup()
	fmt.Println(setup.HitPointStatus().HitPoints().Maximum(), setup.Size(), setup.Speeds().Walk())
	// Output: 7 Large 10
}

// Requirement: doc/specs/monster-combatant-setup.md FR-041 - results compare unequal when only the setup differs, when only the roll differs, and when one holds a roll and the other the average; equal averages compare equal.
// Path: unhappy (determinism).
func TestSetUpMonsterCombatantAppliedEqual(t *testing.T) {
	t.Parallel()
	run := func(m content.Monster, method combat.MonsterHitPointsMethod, faces ...int) combat.SetUpMonsterCombatantApplied {
		res, err := msUse(t, dicetest.NewFixedRoller(faces...)).Run(context.Background(), combat.SetUpMonsterCombatantCommand{Monster: m, HitPoints: method})
		if err != nil {
			t.Fatal(err)
		}
		return res.(combat.SetUpMonsterCombatantApplied)
	}
	m := msHitPoints(t, 7, msRat)
	avg, avgAgain := run(m, combat.UseAverageHitPoints{}), run(m, combat.UseAverageHitPoints{})
	otherMonster := run(msMonster(t, func(f *content.MonsterFields) {
		f.ArmorClass = 3
		f.HitPoints = luMust(content.NewHitPointsEntry(7, msRat))
	}), combat.UseAverageHitPoints{})
	rolled := run(m, combat.RollHitPointDice{}, 3, 4)
	if !avg.Equal(avgAgain) || avg.Equal(otherMonster) || avg.Equal(rolled) || rolled.Equal(avg) || rolled.Equal(run(m, combat.RollHitPointDice{}, 3, 5)) || !rolled.Equal(run(m, combat.RollHitPointDice{}, 3, 4)) {
		t.Error("Equal does not compare the setup and the Hit Points choice")
	}
	sameRollOtherSetup := run(msMonster(t, func(f *content.MonsterFields) {
		f.ArmorClass = 3
		f.HitPoints = luMust(content.NewHitPointsEntry(7, msRat))
	}), combat.RollHitPointDice{}, 3, 4)
	if rolled.Equal(sameRollOtherSetup) {
		t.Error("results with the same roll and different setups are equal")
	}
}
