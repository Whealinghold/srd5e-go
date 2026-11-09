package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func useRelentlessRage(t *testing.T, roller character.Roller, opts ...character.Option) character.UseRelentlessRage {
	t.Helper()
	return must(character.NewUseRelentlessRage(brbProgression(t), roller, opts...))
}

var dropped = character.BarbarianHostFacts{DroppedToZeroHitPoints: true}

func brbRagingBarbarian(t *testing.T, level, uses int) character.Character {
	t.Helper()
	state := brbEnter(t, level, character.MustBarbarianState(0, uses), "")
	return brbWith(t, brbBarbarian(t, level), state)
}

func runRelentless(t *testing.T, c character.Character, mode modifier.RollMode, mods d20.RollModifiers, rolls ...int) (character.UseRelentlessRageResult, *dicetest.FixedRoller) {
	t.Helper()
	roller := dicetest.NewFixedRoller(rolls...)
	res, err := useRelentlessRage(t, roller).Run(context.Background(), character.UseRelentlessRageCommand{Character: c, Facts: dropped, RollMode: mode, Modifiers: mods})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res, roller
}

// Requirement: doc/specs/class-barbarian.md FR-065, FR-066, FR-084, FR-115, FR-126 and table 4.8.1 (RR-01 to RR-05, RR-10 to RR-17) - UseRelentlessRage rolls one d20 for a Normal roll, builds the Roll, composes ResolveCheck against the DC
// (10, then 5 more per use) and, on a success, returns Hit Points of twice the Barbarian level; a failure is a normal StillFailed result; every use is counted and the next DC rises by 5; a natural 20 with a total below the DC fails and
// a natural 1 with a total at or above it succeeds; the modifiers are added as given (no Proficiency Bonus for a multiclass Barbarian).
// Path: per case (RR rows).
func TestUseRelentlessRageRows(t *testing.T) {
	t.Parallel()
	con2 := d20.NewRollModifiers(modifier.New(2))
	con2prof4 := d20.NewRollModifiers(modifier.New(2)).WithProficiencyBonus(modifier.New(4))
	for _, c := range []struct {
		label       string
		level, uses int
		roll        int
		mods        d20.RollModifiers
		success     bool
		hit, nextDC int
	}{
		{"RR-01", 11, 0, 10, d20.NewRollModifiers(modifier.New(0)), true, 22, 15}, {"RR-02", 11, 0, 9, d20.NewRollModifiers(modifier.New(0)), false, 0, 15},
		{"RR-03", 11, 1, 15, d20.NewRollModifiers(modifier.New(0)), true, 22, 20}, {"RR-04", 14, 2, 19, d20.NewRollModifiers(modifier.New(0)), false, 0, 25},
		{"RR-05", 20, 0, 12, d20.NewRollModifiers(modifier.New(0)), true, 40, 15},
		{"RR-10", 11, 0, 20, d20.NewRollModifiers(modifier.New(-11)), false, 0, 15}, {"RR-11", 11, 0, 1, d20.NewRollModifiers(modifier.New(11)), true, 22, 15},
		{"RR-12", 11, 0, 8, con2, true, 22, 15}, {"RR-13", 11, 0, 7, con2, false, 0, 15}, {"RR-14", 11, 1, 13, con2, true, 22, 20}, {"RR-15", 11, 1, 12, con2, false, 0, 20},
		{"RR-16", 11, 0, 6, con2, false, 0, 15}, {"RR-17", 11, 0, 6, con2prof4, true, 22, 15},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			in := brbRagingBarbarian(t, c.level, c.uses)
			res, roller := runRelentless(t, in, modifier.Normal, c.mods, c.roll)
			var (
				next   character.Character
				check  d20.CheckResult
				events []character.Event
				hit    int
				nextDC int
			)
			switch r := res.(type) {
			case character.RelentlessRageSucceeded:
				next, check, events, hit, nextDC = r.Character(), r.Check(), r.Events(), r.HitPoints(), r.NextDC()
			case character.RelentlessRageStillFailed:
				next, check, events, nextDC = r.Character(), r.Check(), r.Events(), r.NextDC()
			default:
				t.Fatalf("result = %#v", res)
			}
			_, succeeded := res.(character.RelentlessRageSucceeded)
			state := brbStateOf(t, next)
			if succeeded != c.success || hit != c.hit || nextDC != c.nextDC || state.RelentlessRageUses() != c.uses+1 || state.RelentlessRageDC() != c.nextDC || !state.RageActive() ||
				check.Roll().Natural() != c.roll || int(check.Target()) != 10+5*c.uses || (check.Outcome() == d20.Success) != c.success {
				t.Errorf("success %v hit %d next DC %d state %v check %v", succeeded, hit, nextDC, state, check)
			}
			ev, ok := events[0].(character.RelentlessRageUsed)
			if len(events) != 1 || !ok || ev.Succeeded() != c.success || ev.HitPoints() != c.hit || ev.NextDC() != c.nextDC || ev.CharacterID() != in.ID() {
				t.Errorf("events = %#v", events)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D20(), 0) {
				t.Errorf("requests = %v, want exactly one 1d20", reqs)
			}
			if got := brbStateOf(t, in); got.RelentlessRageUses() != c.uses {
				t.Errorf("Run altered the input: %v", got)
			}
			events[0] = nil
			if r, isOK := res.(character.RelentlessRageSucceeded); isOK && r.Events()[0] == nil {
				t.Error("Events returned a shared slice")
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-065, FR-084, FR-115 - Advantage and Disadvantage roll two d20s through the dice port and the Roll keeps the higher or the lower one; the same fixed rolls give the same result every run.
// Path: per case (happy / unhappy: determinism).
func TestUseRelentlessRageRollModes(t *testing.T) {
	t.Parallel()
	none := d20.NewRollModifiers(modifier.New(0))
	for _, c := range []struct {
		name    string
		mode    modifier.RollMode
		rolls   []int
		natural int
		success bool
	}{
		{"advantage keeps the higher", modifier.WithAdvantage, []int{3, 12}, 12, true}, {"disadvantage keeps the lower", modifier.WithDisadvantage, []int{3, 12}, 3, false},
		{"advantage, both low", modifier.WithAdvantage, []int{3, 4}, 4, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var first character.UseRelentlessRageResult
			for run := range 2 {
				res, roller := runRelentless(t, brbRagingBarbarian(t, 11, 0), c.mode, none, c.rolls...)
				var check d20.CheckResult
				switch r := res.(type) {
				case character.RelentlessRageSucceeded:
					check = r.Check()
				case character.RelentlessRageStillFailed:
					check = r.Check()
				}
				if check.Roll().Natural() != c.natural || (check.Outcome() == d20.Success) != c.success || len(roller.Requests()) != 2 || check.Roll().Mode() != c.mode {
					t.Errorf("check %v requests %v", check, roller.Requests())
				}
				if run == 0 {
					first = res
				} else if !reflect.DeepEqual(first, res) {
					t.Errorf("runs differ: %#v vs %#v", first, res)
				}
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-067, FR-087, RR-08, RR-09 - with no Barbarian levels (even with a stored state), no Rage, level 10 or no drop to 0 Hit Points the result is RelentlessRageNotAvailable with the first
// reason in the order not a Barbarian, feature not gained, Rage not active, not dropped; the Character is unchanged and the roller is not called.
// Path: per case (unhappy: state error, boundary).
func TestUseRelentlessRageNotAvailable(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		char  character.Character
		facts character.BarbarianHostFacts
		want  character.BarbarianReason
	}{
		{"RR-08 inactive", brbWith(t, brbBarbarian(t, 11), character.InitialBarbarianState()), dropped, character.BarbarianRageNotActive},
		{"RR-09 level 10", brbRagingBarbarian(t, 10, 0), dropped, character.BarbarianFeatureNotGained},
		{"not dropped", brbRagingBarbarian(t, 11, 0), character.BarbarianHostFacts{}, character.BarbarianNotDroppedToZero},
		{"no Barbarian levels", fgtChar(t, cl("wizard", 11)), dropped, character.BarbarianNotABarbarian},
		{"no Barbarian levels with a stored state", brbWith(t, fgtChar(t, cl("wizard", 11)), brbRaging(t, 11, 0)), dropped, character.BarbarianNotABarbarian},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(5)
			res, err := useRelentlessRage(t, roller).Run(context.Background(), character.UseRelentlessRageCommand{Character: c.char, Facts: c.facts})
			if got, ok := res.(character.RelentlessRageNotAvailable); err != nil || !ok || got.Reason != c.want || len(roller.Requests()) != 0 {
				t.Errorf("= %#v, %v; want not available %v", res, err, c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-093 - a roller that returns its error, panics, is exhausted or answers outside its contract is a *srderr.PortError for dice.Port with the cause reachable, the result is nil, and a second
// die failing after the first is the same.
// Path: per case (unhappy: collaborator failure).
func TestUseRelentlessRageRollerFailures(t *testing.T) {
	t.Parallel()
	boom := errors.New("roller down")
	for _, c := range []struct {
		name   string
		roller character.Roller
		mode   modifier.RollMode
		cause  error
	}{
		{"an error", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, boom }), modifier.Normal, boom},
		{"exhausted", dicetest.NewFixedRoller(), modifier.Normal, dicetest.ErrExhausted},
		{"a panic", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(boom) }), modifier.Normal, boom},
		{"a value out of range", dicetest.NewFixedRoller(21), modifier.Normal, dicetest.ErrValueOutOfRange},
		{"the second die", dicetest.NewFixedRoller(5), modifier.WithAdvantage, dicetest.ErrExhausted},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := useRelentlessRage(t, c.roller).Run(context.Background(), character.UseRelentlessRageCommand{Character: brbRagingBarbarian(t, 11, 0), Facts: dropped, RollMode: c.mode})
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, c.cause) {
				t.Errorf("= %v, %v; want a PortError for dice with the cause", res, err)
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-088, FR-115 - the result is identical with the no-op and a recording telemetry; the constructor refuses the zero progression and a nil roller, and Run refuses the zero use case, a zero
// Character, an unknown roll mode, a foreign state under the Barbarian key and a cancelled context before any signal.
// Path: per case (unhappy: invalid input, cancellation).
func TestUseRelentlessRageInvalidInputAndNoop(t *testing.T) {
	t.Parallel()
	none := d20.NewRollModifiers(modifier.New(0))
	good := character.UseRelentlessRageCommand{Character: brbRagingBarbarian(t, 11, 0), Facts: dropped, Modifiers: none}
	rec, opts := fgtTelemetry()
	withRec, err := useRelentlessRage(t, dicetest.NewFixedRoller(14), opts...).Run(context.Background(), good)
	plain, err2 := useRelentlessRage(t, dicetest.NewFixedRoller(14)).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) || len(rec.Tracer.Spans()) != 1 {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	_, err = character.NewUseRelentlessRage(character.BarbarianProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewUseRelentlessRage(brbProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.UseRelentlessRage{}.Run(context.Background(), good)
	wantInvalid(t, err, "zero use case")
	uc := useRelentlessRage(t, dicetest.NewFixedRoller(1))
	_, err = uc.Run(context.Background(), character.UseRelentlessRageCommand{})
	wantInvalid(t, err, "zero character")
	for _, mode := range []modifier.RollMode{-1, 3} {
		bad := good
		bad.RollMode = mode
		_, err = uc.Run(context.Background(), bad)
		wantInvalid(t, err, "unknown roll mode")
	}
	foreign := good
	foreign.Character = must(character.WithFeatureState(brbBarbarian(t, 11), character.MustFeatureStateKey[character.ChosenSubclass]("barbarian"), must(character.NewChosenSubclass("x"))))
	_, err = uc.Run(context.Background(), foreign)
	wantInvalid(t, err, "foreign state")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	rec2, opts2 := fgtTelemetry()
	roller := dicetest.NewFixedRoller(1)
	res, err := useRelentlessRage(t, roller, opts2...).Run(cancelled, good)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec2.Tracer.Spans()))
	}
}

// Requirement: doc/specs/class-barbarian.md FR-089, FR-091, FR-093, FR-124, FR-126 - a success records one span (class barbarian, feature relentless_rage, outcome success), one feature_used increment, one duration and one FINE log; a failed
// save has outcome failure with no rejection, no error status and a log (FR-126); a rule rejection records the first reason; a roller failure sets the error status, increments feature_used once with failure and no error.type and the
// errors counter once with dice_roller_error, with no log; a foreign state is a command error on the span only.
// Path: per case (happy / unhappy: state error, collaborator failure).
func TestFeatureSignalsUseRelentlessRage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	none := d20.NewRollModifiers(modifier.New(0))
	run := func(t *testing.T, roller character.Roller, c character.Character, facts character.BarbarianHostFacts) (*telemetrytest.Recorder, error) {
		t.Helper()
		rec, opts := fgtTelemetry()
		_, err := useRelentlessRage(t, roller, opts...).Run(ctx, character.UseRelentlessRageCommand{Character: c, Facts: facts, Modifiers: none})
		return rec, err
	}
	base := wantCall{span: character.SpanUseRelentlessRage, class: character.ValueClassBarbarian, feature: "relentless_rage"}
	t.Run("a success", func(t *testing.T) {
		t.Parallel()
		rec, err := run(t, dicetest.NewFixedRoller(14), brbRagingBarbarian(t, 11, 0), dropped)
		w := base
		w.outcome, w.logged = "success", true
		checkFeatureCall(t, rec, w)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("a failed save", func(t *testing.T) {
		t.Parallel()
		rec, _ := run(t, dicetest.NewFixedRoller(2), brbRagingBarbarian(t, 11, 0), dropped)
		w := base
		w.outcome, w.logged = "failure", true
		checkFeatureCall(t, rec, w)
	})
	t.Run("a rejection", func(t *testing.T) {
		t.Parallel()
		rec, _ := run(t, dicetest.NewFixedRoller(14), brbBarbarian(t, 11), dropped)
		w := base
		w.outcome, w.rejection, w.logged = "failure", "rage_not_active", true
		checkFeatureCall(t, rec, w)
	})
	t.Run("a roller failure", func(t *testing.T) {
		t.Parallel()
		rec, _ := run(t, dicetest.NewFixedRoller(), brbRagingBarbarian(t, 11, 0), dropped)
		w := base
		w.outcome, w.status, w.errorType = "failure", telemetry.StatusError, "dice_roller_error"
		checkFeatureCall(t, rec, w)
	})
	t.Run("a foreign state", func(t *testing.T) {
		t.Parallel()
		foreign := must(character.WithFeatureState(brbBarbarian(t, 11), character.MustFeatureStateKey[character.ChosenSubclass]("barbarian"), must(character.NewChosenSubclass("x"))))
		rec, err := run(t, dicetest.NewFixedRoller(14), foreign, dropped)
		wantInvalid(t, err, "foreign state")
		spans := rec.Tracer.Spans()
		if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 ||
			len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)) != 0 || len(rec.Logger.Records()) != 0 {
			t.Errorf("spans %+v, feature_used %v", spans, rec.Meter.MeasurementsNamed(character.MetricFeatureUsed))
		}
	})
}
