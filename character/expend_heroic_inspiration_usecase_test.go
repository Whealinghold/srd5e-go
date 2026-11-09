package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func newExpender(t *testing.T, roller character.Roller, opts ...character.Option) character.ExpendHeroicInspiration {
	t.Helper()
	u, err := character.NewExpendHeroicInspiration(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func expendCmd(t *testing.T, c character.Character, die dice.Die, diceRolled, position int) character.ExpendHeroicInspirationCommand {
	t.Helper()
	return character.ExpendHeroicInspirationCommand{Character: c, Target: character.MustRerollTarget(die, diceRolled, position)}
}

// Requirement: doc/specs/heroic-inspiration.md FR-012, FR-014, FR-015, FR-016, FR-017,
// FR-018, FR-019, FR-034, table E1 to E7, E13 - Expend spends the Inspiration, makes
// exactly one request for one die of the target's size, and returns that face unchanged
// with the target's die and position and the one HeroicInspirationExpended event
// (SRD 5.2.1, "Heroic Inspiration", p. 8; "Rules Glossary", p. 183).
// Path: per case.
func TestExpendRerollsOneDie(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path     string
		die                         dice.Die
		diceRolled, position, serve int
	}{
		{"E1 d20, single roll", "FR-012", "happy", dice.D20(), 1, 0, 14},
		{"E2 two d20s, reroll the second", "FR-018", "unhappy (rule interaction)", dice.D20(), 2, 1, 11},
		{"E3 two d20s, reroll the first", "FR-018", "unhappy (rule interaction)", dice.D20(), 2, 0, 6},
		{"E4 damage die d6", "FR-017", "happy", dice.D6(), 3, 2, 4},
		{"E5 d4 lowest face", "FR-016", "unhappy (boundary)", dice.D4(), 1, 0, 1},
		{"E6 d8 highest face", "FR-016", "unhappy (boundary)", dice.D8(), 1, 0, 8},
		{"E6 d10 highest face", "FR-016", "unhappy (boundary)", dice.D10(), 1, 0, 10},
		{"E6 d12 highest face", "FR-016", "unhappy (boundary)", dice.D12(), 1, 0, 12},
		{"E7 d100 highest face", "FR-017", "unhappy (boundary)", dice.D100(), 1, 0, 100},
		{"E13 many dice, one rerolled", "FR-019", "unhappy (boundary)", dice.D6(), 8, 7, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.serve)
			hero := pc(t, "hero", character.Inspired)
			cmd := expendCmd(t, hero, c.die, c.diceRolled, c.position)
			res, err := newExpender(t, roller).Expend(context.Background(), cmd)
			r, ok := res.(character.ExpendHeroicInspirationRerolled)
			if err != nil || !ok {
				t.Fatalf("result = %#v, %v", res, err)
			}
			ev := r.Events()
			if r.Character().HeroicInspiration() != character.NotInspired || len(ev) != 1 {
				t.Errorf("character = %v, events = %v [%s]", r.Character().HeroicInspiration(), ev, c.requirement)
			}
			if e, ok := ev[0].(character.HeroicInspirationExpended); !ok || e.CharacterID() != hero.ID() {
				t.Errorf("event = %#v, want HeroicInspirationExpended for the character", ev[0])
			}
			if got := r.Rerolled(); got != character.MustRerolledDie(c.die, c.position, c.serve) {
				t.Errorf("rerolled = %#v, want die %v position %d value %d [%s]", got, c.die, c.position, c.serve, c.requirement)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, c.die, 0) {
				t.Errorf("requests = %v, want one %s", reqs, c.die)
			}
			if !hero.Equal(pc(t, "hero", character.Inspired)) || cmd.Target != character.MustRerollTarget(c.die, c.diceRolled, c.position) {
				t.Error("Expend altered its input [FR-026]")
			}
			r.Events()[0] = nil
			if r.Events()[0] == nil {
				t.Error("Events shares its storage [FR-026]")
			}
		})
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-013, FR-024, FR-035, table E8, E9 -
// a character who is not Inspired gets the aggregate's HeroicInspirationNotAvailable
// rejection with no roller request and no event; feeding the result of an accepted
// expenditure back in is refused the same way (SRD 5.2.1, p. 8, p. 183).
// Path: unhappy (state error).
func TestExpendRejectsWhenNotInspired(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(9, 9)
	use, hero := newExpender(t, roller), pc(t, "hero", character.NotInspired)
	res, err := use.Expend(context.Background(), expendCmd(t, hero, dice.D20(), 1, 0))
	if r, ok := res.(character.ExpendHeroicInspirationRejected); err != nil || !ok || r.Reason() != (character.HeroicInspirationNotAvailable{}) {
		t.Fatalf("E8: %#v, %v", res, err)
	}
	first, err := use.Expend(context.Background(), expendCmd(t, pc(t, "hero", character.Inspired), dice.D20(), 1, 0))
	if err != nil || len(roller.Requests()) != 1 {
		t.Fatalf("E1: %#v, %v, %d requests", first, err, len(roller.Requests()))
	}
	spent := first.(character.ExpendHeroicInspirationRerolled).Character()
	res, err = use.Expend(context.Background(), expendCmd(t, spent, dice.D20(), 1, 0))
	if _, ok := res.(character.ExpendHeroicInspirationRejected); err != nil || !ok || len(roller.Requests()) != 1 {
		t.Errorf("E9: %#v, %v, %d requests", res, err, len(roller.Requests()))
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-025 - Expend is deterministic: the
// same command and equal fixed rollers give equal results.
// Path: unhappy (determinism).
func TestExpendIsDeterministic(t *testing.T) {
	t.Parallel()
	cmd := expendCmd(t, pc(t, "hero", character.Inspired), dice.D20(), 2, 1)
	a, errA := newExpender(t, dicetest.NewFixedRoller(7)).Expend(context.Background(), cmd)
	b, errB := newExpender(t, dicetest.NewFixedRoller(7)).Expend(context.Background(), cmd)
	if errA != nil || errB != nil || !reflect.DeepEqual(a, b) {
		t.Errorf("results differ: %#v, %v vs %#v, %v", a, errA, b, errB)
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-022, FR-023, FR-030 - a roller
// error, panic or answer for another expression comes back as a *srderr.PortError of
// the dice port, unchanged, with a nil result and the cause reachable; it is recorded
// once (error status, one increment with error.type, no result, no log), and the
// result of the no-op bundle is the same.
// Path: per case.
func TestExpendRollerFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	wrong := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dice.NewResult(dice.MustExpression(2, dice.D20(), 0), []dice.DieRoll{dice.MustDieRoll(dice.D20(), 1), dice.MustDieRoll(dice.D20(), 1)})
	})
	cases := []struct {
		name, requirement, path string
		roller                  character.Roller
		wantIs                  error
	}{
		{"roller fails", "FR-022", "unhappy (failure)", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }), cause},
		{"roller panics", "FR-023", "unhappy (failure)", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("boom") }), srderr.ErrPortPanic},
		{"another expression", "FR-023", "unhappy (contract)", wrong, srderr.ErrContractViolation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			hero := pc(t, "hero", character.Inspired)
			cmd := expendCmd(t, hero, dice.D20(), 1, 0)
			rec := telemetrytest.NewRecorder()
			res, err := newExpender(t, c.roller, character.WithTelemetry(rec.Telemetry())).Expend(context.Background(), cmd)
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || srderr.ErrorType(err) != "dice_roller_error" || !errors.Is(err, c.wantIs) {
				t.Fatalf("result = %#v, err = %v [%s]", res, err, c.requirement)
			}
			if plainRes, plainErr := newExpender(t, c.roller).Expend(context.Background(), cmd); plainRes != nil || !errors.As(plainErr, &pe) {
				t.Errorf("no-op bundle: %#v, %v", plainRes, plainErr)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanExpendHeroicInspiration || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "span", spans[0].Attrs, map[string]any{telemetry.AttrOutcome: "failure"})
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, map[string]any{character.AttrOperation: "expend", telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"})
			noLogsOrViolations(t, rec)
			if !hero.Equal(pc(t, "hero", character.Inspired)) {
				t.Error("the caller's character changed")
			}
		})
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-028, FR-020, FR-021 - the
// constructor refuses a nil roller; Expend refuses the zero use case, a zero
// character, a zero target and a zero-Die target with ErrInvalidInput, and a
// cancelled context with ctx.Err(), all before any signal or roll.
// Path: unhappy (invalid input).
func TestExpendInvalidInputAndCancellation(t *testing.T) {
	t.Parallel()
	if _, err := character.NewExpendHeroicInspiration(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: err = %v", err)
	}
	roller, rec := dicetest.NewFixedRoller(5), telemetrytest.NewRecorder()
	use := newExpender(t, roller, character.WithTelemetry(rec.Telemetry()))
	hero := pc(t, "hero", character.Inspired)
	good := expendCmd(t, hero, dice.D20(), 1, 0)
	zeroDie := character.ExpendHeroicInspirationCommand{Character: hero, Target: character.MustRerollTarget(dice.Die{}, 1, 0)}
	for name, run := range map[string]func() (character.ExpendHeroicInspirationResult, error){
		"zero use case": func() (character.ExpendHeroicInspirationResult, error) {
			return character.ExpendHeroicInspiration{}.Expend(context.Background(), good)
		},
		"zero character": func() (character.ExpendHeroicInspirationResult, error) {
			return use.Expend(context.Background(), character.ExpendHeroicInspirationCommand{Target: good.Target})
		},
		"zero target": func() (character.ExpendHeroicInspirationResult, error) {
			return use.Expend(context.Background(), character.ExpendHeroicInspirationCommand{Character: hero})
		},
		"zero die": func() (character.ExpendHeroicInspirationResult, error) {
			return use.Expend(context.Background(), zeroDie)
		},
	} {
		if res, err := run(); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %#v, %v", name, res, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := use.Expend(cancelled, good)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricOperations)) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("cancelled: %#v, %v", res, err)
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-028, FR-029, FR-037 - Expend records
// the expend_heroic_inspiration span (no error status), one operations increment
// (expended/success, or rejected/failure with not_inspired), one duration with the
// operation only and no log; a gain then an expend record two separate, unnested
// spans and increments.
// Path: per case.
func TestExpendSignals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		state                   character.HeroicInspiration
		attrs                   map[string]any
	}{
		{"expended", "FR-029", "happy", character.Inspired, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "expended"}},
		{"rejected", "FR-029", "unhappy (state error)", character.NotInspired, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "not_inspired"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			cmd := expendCmd(t, pc(t, "hero", c.state), dice.D20(), 1, 0)
			res, err := newExpender(t, dicetest.NewFixedRoller(5), character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).Expend(context.Background(), cmd)
			plain, err2 := newExpender(t, dicetest.NewFixedRoller(5)).Expend(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("results differ: %#v, %v vs %#v, %v", res, err, plain, err2)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanExpendHeroicInspiration || !spans[0].Ended || spans[0].Status != telemetry.StatusUnset || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "span", spans[0].Attrs, c.attrs)
			counter := map[string]any{character.AttrOperation: "expend"}
			for k, v := range c.attrs {
				counter[k] = v
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, counter)
			d := onlyOne(t, rec, character.MetricDuration)
			if d.Value != 0.25 {
				t.Errorf("duration = %+v", d)
			}
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "expend"})
			noLogsOrViolations(t, rec)
		})
	}

	// A gain then an expend in one flow are two independent spans and increments.
	rec := telemetrytest.NewRecorder()
	tel := character.WithTelemetry(rec.Telemetry())
	g, err := character.NewGainHeroicInspiration(tel).Gain(context.Background(), character.GainHeroicInspirationCommand{Holder: pc(t, "hero", character.NotInspired)})
	if err != nil {
		t.Fatal(err)
	}
	holder := g.(character.GainHeroicInspirationReceived).Holder()
	if _, err := newExpender(t, dicetest.NewFixedRoller(5), tel).Expend(context.Background(), expendCmd(t, holder, dice.D20(), 1, 0)); err != nil {
		t.Fatal(err)
	}
	spans := rec.Tracer.Spans()
	if len(spans) != 2 || spans[0].ParentID != 0 || spans[1].ParentID != 0 || len(rec.Meter.MeasurementsNamed(character.MetricOperations)) != 2 || len(rec.Meter.MeasurementsNamed(character.MetricDuration)) != 2 {
		t.Errorf("spans = %+v, want two separate spans, increments and durations", spans)
	}
}
