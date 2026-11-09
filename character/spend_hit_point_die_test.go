package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func spendCmd(t *testing.T, pool, spent dieCounts, die content.HitPointDie, mod int, kind character.RestKind) character.SpendHitPointDieCommand {
	t.Helper()
	cmd, err := character.NewSpendHitPointDieCommand(restHero, poolOfDice(t, pool), spent, die, conMod(t, mod), kind)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

func newSpender(t *testing.T, roller character.Roller, opts ...character.Option) character.HitDieSpender {
	t.Helper()
	s, err := character.NewHitDieSpender(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-054, FR-056, table 4.8c - an
// allowed spend rolls one die of the chosen size exactly once with no modifier and
// returns the roll, the modifier, the Hit Points (at least 1, uncapped) and the
// HitPointDieSpent event (SRD 5.2.1, "Short Rest", p. 187).
// Path: per case.
func TestSpendHitPointDieRollsOnceAndRegains(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		die                     content.HitPointDie
		roll, mod, want         int
	}{
		{"HD-1 roll 4 with +2", "FR-054", "happy", d8, 4, 2, 6},
		{"HD-4 minimum roll with -3 raised to 1", "FR-056", "unhappy (boundary)", d8, 1, -3, 1},
		{"HD-7 maximum roll with +5", "FR-056", "unhappy (boundary: maximum roll)", d12, 12, 5, 17},
		{"HD-8 d6 minimum roll", "FR-056", "unhappy (boundary: minimum roll)", d6, 1, 0, 1},
		{"HD-6 d10 minimum roll with -5", "FR-056", "unhappy (boundary)", d10, 1, -5, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.roll)
			cmd := spendCmd(t, dieCounts{c.die: 2}, dieCounts{c.die: 1}, c.die, c.mod, character.ShortRest)
			res, err := newSpender(t, roller).SpendHitPointDie(context.Background(), cmd)
			a, ok := res.(character.SpendHitPointDieApplied)
			if err != nil || !ok {
				t.Fatalf("result = %#v, %v", res, err)
			}
			if a.Roll() != c.roll || a.ConstitutionModifier() != conMod(t, c.mod) || a.HitPointsRegained() != c.want || a.SpentHitDice().Count(c.die) != 2 {
				t.Errorf("applied = %#v, want %d Hit Points [%s]", a, c.want, c.requirement)
			}
			ev := a.HitPointDieSpent()
			if ev.CharacterID() != restHero || ev.Die() != c.die || ev.Roll() != c.roll || ev.ConstitutionModifier() != conMod(t, c.mod) || ev.HitPointsRegained() != c.want {
				t.Errorf("event = %#v", ev)
			}
			reqs := roller.Requests()
			if len(reqs) != 1 || reqs[0].String() != dice.MustExpression(1, dice.MustDie(c.die.Faces()), 0).String() {
				t.Errorf("roller requests = %v, want one die of %d faces and no modifier", reqs, c.die.Faces())
			}
		})
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-053, FR-038, FR-040, FR-045, table
// 4.8d SP-2, SP-5, SP-8, SP-12 - a rejected spend never calls the roller, changes
// nothing and names the reason; a spent count above the pool is a rejection, not an
// error or a construction problem.
// Path: per case.
func TestSpendHitPointDieRejectionsDoNotRoll(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		pool, spent             dieCounts
		die                     content.HitPointDie
		kind                    character.RestKind
		want                    character.HitDieSpendRejection
	}{
		{"SP-2 none remaining", "FR-040", "unhappy (boundary)", dieCounts{d10: 1}, dieCounts{d10: 1}, d10, character.ShortRest, character.NoneRemaining},
		{"SP-5 not in pool", "FR-040", "unhappy (invalid input)", dieCounts{d8: 5}, nil, d12, character.ShortRest, character.NotInPool},
		{"SP-8 long rest benefits", "FR-045", "unhappy (state error)", dieCounts{d10: 1}, nil, d10, character.LongRest, character.NotAShortRestBenefit},
		{"SP-12 spent exceeds pool", "FR-073", "unhappy (invalid input)", dieCounts{d8: 2}, dieCounts{d8: 3}, d8, character.ShortRest, character.SpentExceedsPool},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(1)
			res, err := newSpender(t, roller).SpendHitPointDie(context.Background(), spendCmd(t, c.pool, c.spent, c.die, 0, c.kind))
			if r, ok := res.(character.SpendHitPointDieRejected); err != nil || !ok || r.Reason() != c.want {
				t.Errorf("result = %#v, %v, want rejected %v [%s]", res, err, c.want, c.requirement)
			}
			if n := len(roller.Requests()); n != 0 {
				t.Errorf("roller called %d times, want 0", n)
			}
		})
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-055, errors-and-results.md FR-031 -
// a roller error, panic or answer for another expression comes back as a
// *srderr.PortError of the dice port, unchanged, with a nil result, and the cause
// stays reachable.
// Path: per case.
func TestSpendHitPointDieRollerFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	d8e := func(count int, die dice.Die, mod int) character.Roller {
		rolls := make([]dice.DieRoll, count)
		for i := range rolls {
			rolls[i] = dice.MustDieRoll(die, 1)
		}
		return dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
			return dice.NewResult(dice.MustExpression(count, die, mod), rolls)
		})
	}
	cases := []struct {
		name, requirement, path string
		roller                  character.Roller
		wantIs                  error
	}{
		{"roller fails", "FR-055", "unhappy (failure of a collaborator)", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }), cause},
		{"roller panics", "FR-055", "unhappy (failure of a collaborator)", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("boom") }), srderr.ErrPortPanic},
		{"another count", "FR-055", "unhappy (contract)", d8e(2, dice.D8(), 0), srderr.ErrContractViolation},
		{"another die", "FR-055", "unhappy (contract)", d8e(1, dice.D6(), 0), srderr.ErrContractViolation},
		{"another modifier", "FR-055", "unhappy (contract)", d8e(1, dice.D8(), 1), srderr.ErrContractViolation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			spender := newSpender(t, c.roller, character.WithTelemetry(rec.Telemetry()))
			res, err := spender.SpendHitPointDie(context.Background(), spendCmd(t, dieCounts{d8: 2}, nil, d8, 0, character.ShortRest))
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || srderr.ErrorType(err) != "dice_roller_error" || !errors.Is(err, c.wantIs) {
				t.Fatalf("result = %#v, err = %v [%s]", res, err, c.requirement)
			}
			// Recorded exactly once: error status, one counter, no result attribute, no log.
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
				t.Fatalf("spans = %+v", spans)
			}
			want := map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error", character.AttrHitDieFaces: int64(8)}
			sameAttrs(t, "counter", onlyOne(t, rec, character.MetricHitPointDiceSpent).Attrs, want)
			if n := len(rec.Meter.MeasurementsNamed(character.MetricHitDieHitPoints)); n != 0 {
				t.Errorf("%d Hit Point measurements for a failed spend", n)
			}
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-057, FR-073, section 7 - the
// constructors reject a nil roller, a zero id, an invalid die or kind, and a spent
// count below 1 (joined SpentCountBelowOne, matching ErrInvalidInput); the use case
// refuses the zero use case and command, and a cancelled context before any signal.
// Path: per case.
func TestSpendHitPointDieInvalidInput(t *testing.T) {
	t.Parallel()
	pool := poolOfDice(t, dieCounts{d8: 2})
	if _, err := character.NewHitDieSpender(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: err = %v", err)
	}
	for name, args := range map[string]struct {
		id   character.Identifier
		spnt dieCounts
		die  content.HitPointDie
		kind character.RestKind
		is   error
	}{
		"zero id":     {character.Identifier{}, nil, d8, character.ShortRest, srderr.ErrInvalidInput},
		"invalid die": {restHero, nil, 0, character.ShortRest, srderr.ErrInvalidInput}, "invalid kind": {restHero, nil, d8, 0, srderr.ErrInvalidInput},
		"zero spent count":  {restHero, dieCounts{d8: 0}, d8, character.ShortRest, character.SpentCountBelowOne},
		"invalid spent die": {restHero, dieCounts{0: 1}, d8, character.ShortRest, srderr.ErrInvalidInput},
	} {
		_, err := character.NewSpendHitPointDieCommand(args.id, pool, args.spnt, args.die, conMod(t, 0), args.kind)
		if !errors.Is(err, args.is) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if character.SpentCountBelowOne.String() != "spent_count_below_one" || character.SpendHitPointDieCommandProblem(0).String() != "unknown" || character.SpendHitPointDieCommandProblem(2).String() != "unknown" || character.SpentCountBelowOne.Error() == "" {
		t.Error("problem names are wrong")
	}

	roller := dicetest.NewFixedRoller(1)
	spender := newSpender(t, roller)
	cmd := spendCmd(t, dieCounts{d8: 2}, nil, d8, 0, character.ShortRest)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for name, run := range map[string]func() (character.SpendHitPointDieResult, error){
		"zero command": func() (character.SpendHitPointDieResult, error) {
			return spender.SpendHitPointDie(context.Background(), character.SpendHitPointDieCommand{})
		},
		"zero use case": func() (character.SpendHitPointDieResult, error) {
			return character.HitDieSpender{}.SpendHitPointDie(context.Background(), cmd)
		},
	} {
		if res, err := run(); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %#v, %v", name, res, err)
		}
	}
	rec := telemetrytest.NewRecorder()
	res, err := newSpender(t, roller, character.WithTelemetry(rec.Telemetry())).SpendHitPointDie(cancelled, cmd)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("cancelled: %#v, %v", res, err)
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-057, FR-048 - SpendHitPointDie
// returns an equal result with the no-op bundle and a recording one, and does not
// alter its command's spent dice.
// Path: happy.
func TestSpendHitPointDieResultIsIndependentOfTelemetry(t *testing.T) {
	t.Parallel()
	for _, kind := range []character.RestKind{character.ShortRest, character.LongRest} {
		cmd := spendCmd(t, dieCounts{d8: 2}, dieCounts{d8: 1}, d8, 1, kind)
		plain, err1 := newSpender(t, dicetest.NewFixedRoller(5)).SpendHitPointDie(context.Background(), cmd)
		seen, err2 := newSpender(t, dicetest.NewFixedRoller(5), character.WithTelemetry(telemetrytest.NewRecorder().Telemetry())).SpendHitPointDie(context.Background(), cmd)
		if err1 != nil || err2 != nil || !reflect.DeepEqual(plain, seen) {
			t.Errorf("results differ: %#v, %v vs %#v, %v", plain, err1, seen, err2)
		}
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-057, section 7 - a spend records
// the spend_hit_point_die span (no error status), one hit_point_dice_spent
// increment (result spent or rejected with the rejection, hit_die_faces, no
// error.type), the Hit Points histogram on a spend only, one duration with only
// rest_operation, and no log.
// Path: per case.
func TestSpendHitPointDieSignals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		cmd                     character.SpendHitPointDieCommand
		outcome, result         string
		rejection               string
		hitPoints               float64 // 0 when none is recorded
	}{
		{"spent", "section 7", "happy", spendCmd(t, dieCounts{d10: 2}, nil, d10, 2, character.ShortRest), "success", "spent", "", 7},
		{"rejected for no dice left", "section 7", "unhappy (state error)", spendCmd(t, dieCounts{d10: 1}, dieCounts{d10: 1}, d10, 0, character.ShortRest), "failure", "rejected", "none_remaining", 0},
		{"rejected for a long rest", "section 7", "unhappy (state error)", spendCmd(t, dieCounts{d10: 1}, nil, d10, 0, character.LongRest), "failure", "rejected", "not_a_short_rest_benefit", 0},
		{"rejected for spent above the pool", "FR-073", "unhappy (invalid input)", spendCmd(t, dieCounts{d10: 1}, dieCounts{d10: 2}, d10, 0, character.ShortRest), "failure", "rejected", "spent_exceeds_pool", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			if _, err := newSpender(t, dicetest.NewFixedRoller(5), character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).SpendHitPointDie(context.Background(), c.cmd); err != nil {
				t.Fatal(err)
			}
			attrs := map[string]any{telemetry.AttrOutcome: c.outcome, character.AttrResult: c.result}
			if c.rejection != "" {
				attrs[character.AttrRejection] = c.rejection
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanSpendHitPointDie || !spans[0].Ended || spans[0].Status != telemetry.StatusUnset || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v", spans)
			}
			spanAttrs := map[string]any{character.AttrHitDieFaces: int64(10)}
			counter := map[string]any{character.AttrHitDieFaces: int64(10)}
			for k, v := range attrs {
				spanAttrs[k], counter[k] = v, v
			}
			sameAttrs(t, "span", spans[0].Attrs, spanAttrs)
			spent := onlyOne(t, rec, character.MetricHitPointDiceSpent)
			if spent.Value != 1 || spent.Unit != "{spend}" || spent.Kind != telemetrytest.KindCounter {
				t.Errorf("spent = %+v", spent)
			}
			sameAttrs(t, "spent", spent.Attrs, counter)
			hp := rec.Meter.MeasurementsNamed(character.MetricHitDieHitPoints)
			if c.hitPoints == 0 && len(hp) != 0 {
				t.Errorf("Hit Points measured for a rejection: %+v", hp)
			}
			if c.hitPoints != 0 {
				if len(hp) != 1 || hp[0].Value != c.hitPoints || hp[0].Unit != "{hit_point}" {
					t.Fatalf("Hit Points = %+v, want %v", hp, c.hitPoints)
				}
				sameAttrs(t, "hit points", hp[0].Attrs, map[string]any{character.AttrHitDieFaces: int64(10)})
			}
			d := onlyOne(t, rec, character.MetricRestOperationDuration)
			if d.Value != 0.25 || d.Unit != "s" {
				t.Errorf("duration = %+v", d)
			}
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrRestOperation: "spend_hit_point_die"})
			noLogsOrViolations(t, rec)
		})
	}
}

// spendCmdFor is a spend of a d8 by the character id.
func spendCmdFor(t *testing.T, id character.Identifier) character.SpendHitPointDieCommand {
	t.Helper()
	cmd, err := character.NewSpendHitPointDieCommand(id, poolOfDice(t, dieCounts{d8: 2}), nil, d8, conMod(t, 0), character.ShortRest)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}
