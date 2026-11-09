package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

var restHero = character.MustIdentifier("hero")

func restCmd(t *testing.T, kind character.RestKind, variant character.LongRestVariant, end character.RestEnd, hp, level int, spacing character.LongRestSpacing, spent character.SpentHitDice) character.TakeRestCommand {
	t.Helper()
	rep, err := character.NewRestReport(kind, variant, end)
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := character.NewTakeRestCommand(restHero, rep, hp, level, spacing, spent)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

// simpleRest is a command with 12 Hit Points, no Exhaustion and nothing spent.
func simpleRest(t *testing.T, kind character.RestKind, end character.RestEnd) character.TakeRestCommand {
	t.Helper()
	return restCmd(t, kind, character.StandardLongRest, end, 12, 0, nil, character.SpentHitDice{})
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-049, FR-036, FR-014, FR-020 - a
// finished rest is TakeRestApplied with a RestApplied event naming the kind, the
// source and the effects; an interrupted Long Rest that earned Short Rest benefits
// is TakeRestAppliedAfterInterruption with both events.
// Path: happy.
func TestTakeRestApplied(t *testing.T) {
	t.Parallel()
	taker := character.NewRestTaker()
	spent := spentOf(t, dieCounts{d8: 2})

	short, ok := taker.TakeRest(context.Background(), simpleRest(t, character.ShortRest, ran(1, 0))).(character.TakeRestApplied)
	if !ok || short.Applied().BenefitsOf() != character.ShortRest || short.Applied().Source() != character.RestFinished || short.Applied().CharacterID() != restHero || len(short.Applied().Effects()) != 2 {
		t.Fatalf("short rest = %#v", short)
	}
	long, ok := taker.TakeRest(context.Background(), restCmd(t, character.LongRest, character.StandardLongRest, ran(8, 0), 12, 2, nil, spent)).(character.TakeRestApplied)
	wantEffects := benefitsOf(t, character.LongRest, 2, spent).Effects()
	if !ok || long.Applied().BenefitsOf() != character.LongRest || long.Applied().Source() != character.RestFinished || !reflect.DeepEqual(long.Applied().Effects(), wantEffects) || long.SpentHitDice().Total() != 0 {
		t.Fatalf("long rest = %#v", long)
	}
	// LI-11: stopped at the required hours is a finished rest and raises RestApplied only.
	late, ok := taker.TakeRest(context.Background(), simpleRest(t, character.LongRest, stopped(rIn, 8, 0))).(character.TakeRestApplied)
	if !ok || late.Applied().BenefitsOf() != character.LongRest || late.Applied().Source() != character.RestFinished {
		t.Fatalf("LI-11 = %#v", late)
	}

	after, ok := taker.TakeRest(context.Background(), restCmd(t, character.LongRest, character.StandardLongRest, stopped(rDmg, 3, 0), 12, 4, nil, spent)).(character.TakeRestAppliedAfterInterruption)
	if !ok || after.Applied().BenefitsOf() != character.ShortRest || after.Applied().Source() != character.LongRestInterrupted || len(after.Applied().Effects()) != 2 || !after.SpentHitDice().Equal(spent) {
		t.Fatalf("interrupted long rest = %#v", after)
	}
	if i := after.Interrupted(); i.Kind() != character.LongRest || i.Interruption() != rDmg || i.BenefitsGranted() != character.RestBenefitsOfShortRest || i.CharacterID() != restHero {
		t.Errorf("RestInterrupted = %#v", i)
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-050, FR-012, FR-070, table 4.8a
// LI-10 - blockers are evaluated first on the values at the start of the rest and
// are all listed; a Long Rest stopped by damage is reported with the Hit Points it
// started with and is not refused.
// Path: unhappy (state error).
func TestTakeRestNotStarted(t *testing.T) {
	t.Parallel()
	taker := character.NewRestTaker()
	cmd := restCmd(t, character.LongRest, character.StandardLongRest, ran(8, 0), 0, 6, hoursAgo(t, 10), character.SpentHitDice{})
	ns, ok := taker.TakeRest(context.Background(), cmd).(character.TakeRestNotStarted)
	if !ok || len(ns.Blockers()) != 3 {
		t.Fatalf("result = %#v", ns)
	}
	// Even a report that would be rejected is NotStarted when a blocker applies.
	cmd = restCmd(t, character.ShortRest, character.StandardLongRest, ran(1, 1), 0, 0, nil, character.SpentHitDice{})
	if _, ok := taker.TakeRest(context.Background(), cmd).(character.TakeRestNotStarted); !ok {
		t.Error("blockers were not evaluated before the report")
	}
	// LI-10: 12 Hit Points at the start, interrupted by damage.
	cmd = restCmd(t, character.LongRest, character.StandardLongRest, stopped(rDmg, 3, 0), 12, 0, nil, character.SpentHitDice{})
	if _, ok := taker.TakeRest(context.Background(), cmd).(character.TakeRestAppliedAfterInterruption); !ok {
		t.Errorf("LI-10 = %#v", taker.TakeRest(context.Background(), cmd))
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-051 - incomplete, stopped and
// rejected results return the spent dice unchanged; a stop without benefits
// raises RestInterrupted with RestBenefitsNone.
// Path: unhappy (state error).
func TestTakeRestNoBenefitResults(t *testing.T) {
	t.Parallel()
	taker, spent := character.NewRestTaker(), spentOf(t, dieCounts{d10: 1})
	cmd := func(kind character.RestKind, end character.RestEnd) character.TakeRestCommand {
		return restCmd(t, kind, character.StandardLongRest, end, 5, 3, nil, spent)
	}
	inc, ok := taker.TakeRest(context.Background(), cmd(character.LongRest, ran(7, 0))).(character.TakeRestIncomplete)
	if !ok || inc.HoursStillNeeded() != 1 || !inc.SpentHitDice().Equal(spent) {
		t.Errorf("incomplete = %#v", inc)
	}
	st, ok := taker.TakeRest(context.Background(), cmd(character.ShortRest, stopped(rEx, 1, 0))).(character.TakeRestStopped)
	if !ok || !st.SpentHitDice().Equal(spent) || st.Interrupted().BenefitsGranted() != character.RestBenefitsNone || st.Interrupted().Kind() != character.ShortRest || st.Interrupted().Interruption() != rEx {
		t.Errorf("stopped = %#v", st)
	}
	rej, ok := taker.TakeRest(context.Background(), cmd(character.ShortRest, ran(1, 1))).(character.TakeRestRejected)
	if !ok || rej.Problem() != character.ShortRestCannotBeResumed || !rej.SpentHitDice().Equal(spent) {
		t.Errorf("rejected = %#v", rej)
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md table 4.8g SC-1 to SC-6, FR-072 - only a
// completed Long Rest (a TakeRestApplied with BenefitsOf LongRest and source
// RestFinished) starts the 16-hour clock (SRD 5.2.1, "Long Rest", p. 185).
// Path: per case.
func TestOnlyACompletedLongRestStartsTheClock(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		kind                    character.RestKind
		end                     character.RestEnd
		want                    bool
	}{
		{"SC-1 completed long rest", "SC-1", "happy", character.LongRest, ran(8, 0), true},
		{"SC-2 interrupted long rest with short rest benefits", "SC-2", "unhappy (rule interaction)", character.LongRest, stopped(rDmg, 3, 0), false},
		{"SC-3 long rest stopped without benefits", "SC-3", "unhappy (state error)", character.LongRest, stopped(rDmg, 0, 0), false},
		{"SC-4 incomplete long rest", "SC-4", "unhappy (state error)", character.LongRest, ran(7, 0), false},
		{"SC-5 completed short rest", "SC-5", "unhappy (rule interaction)", character.ShortRest, ran(1, 0), false},
		{"SC-6 long rest stopped after the required hours", "SC-6", "unhappy (boundary)", character.LongRest, stopped(rDmg, 8, 0), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := character.NewRestTaker().TakeRest(context.Background(), simpleRest(t, c.kind, c.end))
			a, ok := res.(character.TakeRestApplied)
			if got := ok && a.Applied().BenefitsOf() == character.LongRest && a.Applied().Source() == character.RestFinished; got != c.want {
				t.Errorf("starts the clock = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-013, FR-066 - NewTakeRestCommand
// lists every problem once each, in input order; a zero id or report is an input
// error; a zero command is refused by TakeRest with a nil result.
// Path: per case.
func TestNewTakeRestCommand(t *testing.T) {
	t.Parallel()
	rep := character.MustRestReport(character.ShortRest, character.StandardLongRest, ran(1, 0))
	for name, c := range map[string]struct {
		hp, level int
		want      []character.TakeRestCommandProblem
	}{
		"negative Hit Points":  {-1, 0, []character.TakeRestCommandProblem{character.NegativeCurrentHitPoints}},
		"negative level":       {1, -1, []character.TakeRestCommandProblem{character.ExhaustionLevelOutOfRange}},
		"level 7":              {1, 7, []character.TakeRestCommandProblem{character.ExhaustionLevelOutOfRange}},
		"both":                 {-1, 9, []character.TakeRestCommandProblem{character.NegativeCurrentHitPoints, character.ExhaustionLevelOutOfRange}},
		"level 6 is valid":     {1, 6, nil},
		"zero Hit Points okay": {0, 0, nil},
	} {
		_, err := character.NewTakeRestCommand(restHero, rep, c.hp, c.level, nil, character.SpentHitDice{})
		if (err == nil) != (len(c.want) == 0) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Errorf("%s: err = %v", name, err)
		}
		for _, p := range []character.TakeRestCommandProblem{character.NegativeCurrentHitPoints, character.ExhaustionLevelOutOfRange} {
			want := false
			for _, w := range c.want {
				want = want || w == p
			}
			if errors.Is(err, p) != want {
				t.Errorf("%s: errors.Is(err, %v) = %v, want %v", name, p, !want, want)
			}
		}
	}
	for name, args := range map[string][2]any{"zero id": {character.Identifier{}, rep}, "zero report": {restHero, character.RestReport{}}} {
		if _, err := character.NewTakeRestCommand(args[0].(character.Identifier), args[1].(character.RestReport), 1, 0, nil, character.SpentHitDice{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if character.NegativeCurrentHitPoints.String() != "negative_current_hit_points" || character.ExhaustionLevelOutOfRange.String() != "exhaustion_level_out_of_range" || character.TakeRestCommandProblem(0).String() != "unknown" || character.ExhaustionLevelOutOfRange.Error() == "" {
		t.Error("problem names are wrong")
	}
	if got := character.NewRestTaker().TakeRest(context.Background(), character.TakeRestCommand{}); got != nil {
		t.Errorf("zero command result = %#v, want nil", got)
	}
	if got := (character.RestTaker{}).TakeRest(context.Background(), simpleRest(t, character.ShortRest, ran(1, 0))); got != nil {
		t.Errorf("zero RestTaker result = %#v, want nil", got)
	}
	defer func() {
		if e, ok := recover().(error); !ok || !errors.Is(e, srderr.ErrInvalidInput) {
			t.Error("MustTakeRestCommand did not panic with an ErrInvalidInput error")
		}
	}()
	character.MustTakeRestCommand(restHero, rep, -1, 0, nil, character.SpentHitDice{})
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-057 - TakeRest returns an equal
// result with the no-op bundle and with a recording one, and does not alter its
// command.
// Path: happy.
func TestTakeRestResultIsIndependentOfTelemetry(t *testing.T) {
	t.Parallel()
	cmds := []character.TakeRestCommand{
		simpleRest(t, character.ShortRest, ran(1, 0)), simpleRest(t, character.LongRest, ran(8, 0)), simpleRest(t, character.LongRest, stopped(rDmg, 3, 0)),
		simpleRest(t, character.LongRest, stopped(rDmg, 0, 0)), simpleRest(t, character.LongRest, ran(2, 0)), simpleRest(t, character.ShortRest, ran(1, 1)),
		restCmd(t, character.LongRest, character.StandardLongRest, ran(8, 0), 0, 0, nil, character.SpentHitDice{}),
	}
	for _, cmd := range cmds {
		rec := telemetrytest.NewRecorder()
		plain := character.NewRestTaker().TakeRest(context.Background(), cmd)
		observed := character.NewRestTaker(character.WithTelemetry(rec.Telemetry())).TakeRest(context.Background(), cmd)
		if !reflect.DeepEqual(plain, observed) {
			t.Errorf("results differ: %#v vs %#v", plain, observed)
		}
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-057, section 7 - each result is
// recorded once: the take_rest span (never an error status), the rests counter with
// the rest_kind, outcome, result, rest_benefits and, for not_started and rejected,
// the rejection; one duration with only the rest_operation attribute; no log.
// Path: per case.
func TestTakeRestSignals(t *testing.T) {
	t.Parallel()
	exhausted := restCmd(t, character.LongRest, character.StandardLongRest, ran(8, 0), 5, 6, nil, character.SpentHitDice{})
	cases := []struct {
		name, requirement, path string
		cmd                     character.TakeRestCommand
		kind, outcome, result   string
		benefits, rejection     string
	}{
		{"finished short rest", "section 7", "happy", simpleRest(t, character.ShortRest, ran(1, 0)), "short", "success", "applied", "short", ""},
		{"finished long rest", "section 7", "happy", simpleRest(t, character.LongRest, ran(8, 0)), "long", "success", "applied", "long", ""},
		{"interrupted long rest", "section 7", "happy", simpleRest(t, character.LongRest, stopped(rDmg, 3, 0)), "long", "success", "applied_after_interruption", "short", ""},
		{"stopped short rest", "section 7", "unhappy (state error)", simpleRest(t, character.ShortRest, stopped(rDmg, 1, 0)), "short", "failure", "stopped", "none", ""},
		{"incomplete long rest", "section 7", "unhappy (state error)", simpleRest(t, character.LongRest, ran(3, 0)), "long", "failure", "incomplete", "none", ""},
		{"rejected report", "FR-017", "unhappy (state error)", simpleRest(t, character.ShortRest, ran(1, 1)), "short", "failure", "rejected", "none", "short_rest_cannot_be_resumed"},
		{"not started: Hit Points", "FR-007", "unhappy (state error)", restCmd(t, character.ShortRest, character.StandardLongRest, ran(1, 0), 0, 0, nil, character.SpentHitDice{}), "short", "failure", "not_started", "none", "no_hit_points"},
		{"not started: Exhaustion", "FR-011", "unhappy (state error)", exhausted, "long", "failure", "not_started", "none", "exhaustion_level_six"},
		{"not started: spacing", "FR-009", "unhappy (state error)", restCmd(t, character.LongRest, character.StandardLongRest, ran(8, 0), 5, 0, hoursAgo(t, 4), character.SpentHitDice{}), "long", "failure", "not_started", "none", "too_soon_after_long_rest"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			character.NewRestTaker(character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).TakeRest(context.Background(), c.cmd)

			attrs := map[string]any{telemetry.AttrOutcome: c.outcome, character.AttrResult: c.result}
			if c.rejection != "" {
				attrs[character.AttrRejection] = c.rejection
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanTakeRest || !spans[0].Ended || spans[0].Status != telemetry.StatusUnset || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v", spans)
			}
			spanAttrs := map[string]any{character.AttrRestKind: c.kind}
			for k, v := range attrs {
				spanAttrs[k] = v
			}
			sameAttrs(t, "span", spans[0].Attrs, spanAttrs)
			rests := onlyOne(t, rec, character.MetricRests)
			counter := map[string]any{character.AttrRestKind: c.kind, character.AttrRestBenefits: c.benefits}
			for k, v := range attrs {
				counter[k] = v
			}
			if rests.Value != 1 || rests.Unit != "{rest}" || rests.Kind != telemetrytest.KindCounter {
				t.Errorf("rests = %+v", rests)
			}
			sameAttrs(t, "rests", rests.Attrs, counter)
			d := onlyOne(t, rec, character.MetricRestOperationDuration)
			if d.Value != 0.25 || d.Unit != "s" {
				t.Errorf("duration = %+v", d)
			}
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrRestOperation: "take_rest"})
			noLogsOrViolations(t, rec)
		})
	}
}
