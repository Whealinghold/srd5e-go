package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func demandsOf(t *testing.T) character.SentientDemands {
	t.Helper()
	return must(character.NewSentientDemands([]content.SentientDemand{content.SentientDemandKeepMeClose}))
}

var (
	d20Req = dice.MustExpression(1, dice.D20(), 0)
	d12Req = dice.MustExpression(1, dice.D12(), 0)
)

// Requirement: per case (doc/specs/magic-item-rules.md FR-098, FR-099, FR-123 - the conflict save is Charisma against DC 12 plus the
// item's modifier: a total at the DC resists, one below makes the demands, ChooseToFail and AutomaticFailure make none and fail).
// Path: per case.
func TestResolveSentientItemConflict(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Conflict", p. 208`
	cases := []struct {
		name, requirement, path string
		attempt                 d20.SaveAttempt
		faces                   []int
		requests                []dice.Expression
		demands                 bool
		forced                  bool
	}{
		{"total equals the DC", p, "unhappy (boundary)", rollSave(0), []int{12}, []dice.Expression{d20Req}, false, false},
		{"one below the DC", p, "unhappy (boundary)", rollSave(0), []int{11}, []dice.Expression{d20Req}, true, false},
		{"bearer's modifier lifts the total", p, "happy", rollSave(2), []int{10}, []dice.Expression{d20Req}, false, false},
		{"Advantage keeps the higher", p, "unhappy (rule interaction)", rollSave(0, modifier.Advantage{}), []int{3, 18}, []dice.Expression{dice.MustExpression(2, dice.D20(), 0)}, false, false},
		{"Disadvantage keeps the lower", p, "unhappy (rule interaction)", rollSave(0, modifier.Disadvantage{}), []int{3, 18}, []dice.Expression{dice.MustExpression(2, dice.D20(), 0)}, true, false},
		{"cancelling pair rolls one die", p, "unhappy (rule interaction)", rollSave(0, modifier.Advantage{}, modifier.Disadvantage{}), []int{12}, []dice.Expression{d20Req}, false, false},
		{"chosen to fail", p, "unhappy (rule interaction)", d20.ChooseToFail{}, nil, nil, true, true},
		{"automatic failure", p, "unhappy (rule interaction)", d20.AutomaticFailure{}, nil, nil, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			use := must(character.NewResolveSentientItemConflict(roller))
			res, err := use.Do(context.Background(), character.ResolveSentientItemConflictCommand{ItemCharisma: score(t, 10), Attempt: c.attempt, Demands: demandsOf(t)})
			if err != nil || !reflect.DeepEqual(roller.Requests(), c.requests) {
				t.Fatalf("err = %v, requests = %v, want %v", err, roller.Requests(), c.requests)
			}
			var save character.SentientSave
			switch r := res.(type) {
			case character.ResolveSentientItemConflictDemandsMade:
				save = r.Save()
				if !c.demands || len(r.Demands().Demands()) != 1 {
					t.Errorf("demands made = %+v [%s]", r, c.requirement)
				}
			case character.ResolveSentientItemConflictResisted:
				save = r.Save()
				if c.demands {
					t.Errorf("resisted = %+v [%s]", r, c.requirement)
				}
			default:
				t.Fatalf("result = %#v", res)
			}
			if _, isForced := save.(character.SentientSaveForced); isForced != c.forced {
				t.Errorf("save = %#v", save)
			}
			if rolled, ok := save.(character.SentientSaveRolled); ok && rolled.Check.Target() != 12 {
				t.Errorf("DC = %d", rolled.Check.Target())
			}
		})
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-099, FR-109 - an invalid demands list is rejected with no request; the zero use case,
// zero Charisma, a nil or foreign attempt and a cancelled context are refused before any signal; signals of both results.
// Path: unhappy (invalid input).
func TestResolveSentientItemConflictSignalsAndInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	roller := dicetest.NewFixedRoller(14, 1)
	use := must(character.NewResolveSentientItemConflict(roller, character.WithTelemetry(rec.Telemetry())))
	cmd := character.ResolveSentientItemConflictCommand{ItemCharisma: score(t, 10), Attempt: rollSave(0), Demands: demandsOf(t)}
	if _, err := use.Do(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	checkOneCall(t, rec, character.SpanResolveSentientConflict, "resolve_sentient_conflict", telemetry.StatusUnset, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "resisted", d20.AttrKind: "saving_throw"}, d20.AttrKind)
	rec.Clear()
	cmd.Demands = character.SentientDemands{}
	res, err := use.Do(context.Background(), cmd)
	if _, ok := res.(character.ResolveSentientItemConflictRejected); !ok || err != nil || len(roller.Requests()) != 1 {
		t.Fatalf("rejected = %#v, %v, requests %v", res, err, roller.Requests())
	}
	checkOneCall(t, rec, character.SpanResolveSentientConflict, "resolve_sentient_conflict", telemetry.StatusUnset, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "demands_invalid"})
	if _, err := character.NewResolveSentientItemConflict(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	good := character.ResolveSentientItemConflictCommand{ItemCharisma: score(t, 10), Attempt: rollSave(0), Demands: demandsOf(t)}
	for name, c := range map[string]character.ResolveSentientItemConflictCommand{
		"zero Charisma":   {Attempt: rollSave(0), Demands: demandsOf(t)},
		"nil attempt":     {ItemCharisma: score(t, 10), Demands: demandsOf(t)},
		"foreign attempt": {ItemCharisma: score(t, 10), Attempt: foreignAttempt{}, Demands: demandsOf(t)},
	} {
		if r, err := use.Do(context.Background(), c); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v %v", name, r, err)
		}
	}
	if r, err := (character.ResolveSentientItemConflict{}).Do(context.Background(), good); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", r, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	rec.Clear()
	if r, err := use.Do(cancelled, good); r != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v %v", r, err)
	}
}

func attemptControl(t *testing.T, roller character.Roller, cmd character.AttemptSentientItemControlCommand, opts ...character.Option) (character.AttemptSentientItemControlResult, error) {
	t.Helper()
	return must(character.NewAttemptSentientItemControl(roller, opts...)).Do(context.Background(), cmd)
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-101, FR-103, FR-104, FR-113 - a failed save takes control for 1d12 hours
// (Charmed, must follow the item's commands) with a SentientControlTaken event; a passed save resists with no duration roll; either
// way the power is spent until the next dawn; a spent power is rejected with no save and no request).
// Path: per case.
func TestAttemptSentientItemControl(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Conflict", p. 208`
	cmd := func(power character.SentientControlPower, a d20.SaveAttempt) character.AttemptSentientItemControlCommand {
		return character.AttemptSentientItemControlCommand{CharacterID: holder, Power: power, ItemCharisma: score(t, 14), Attempt: a}
	}
	// Charisma 14 is +2, so the DC is 14.
	taken, err := attemptControl(t, dicetest.NewFixedRoller(13, 7), cmd(0, rollSave(0)))
	tk, ok := taken.(character.AttemptSentientItemControlTaken)
	if err != nil || !ok || tk.Effect().Hours() != 7 || tk.Power() != character.ControlPowerSpentUntilNextDawn || tk.Event().CharacterID() != holder || tk.Event().Hours() != 7 || tk.Save().Failed() != true {
		t.Fatalf("taken = %#v, %v [%s]", taken, err, p)
	}
	roller := dicetest.NewFixedRoller(14)
	resisted, err := attemptControl(t, roller, cmd(0, rollSave(0)))
	rs, ok := resisted.(character.AttemptSentientItemControlResisted)
	if err != nil || !ok || rs.Power() != character.ControlPowerSpentUntilNextDawn || rs.Save().Failed() || !reflect.DeepEqual(roller.Requests(), []dice.Expression{d20Req}) {
		t.Fatalf("resisted = %#v, %v, requests %v", resisted, err, roller.Requests())
	}
	roller = dicetest.NewFixedRoller(1, 12)
	if res, err := attemptControl(t, roller, cmd(0, rollSave(0))); err != nil || !reflect.DeepEqual(roller.Requests(), []dice.Expression{d20Req, d12Req}) || res.(character.AttemptSentientItemControlTaken).Effect().Hours() != 12 {
		t.Errorf("save then 1d12: %v, requests %v", err, roller.Requests())
	}
	roller = dicetest.NewFixedRoller(5)
	if res, err := attemptControl(t, roller, cmd(0, d20.ChooseToFail{})); err != nil || !reflect.DeepEqual(roller.Requests(), []dice.Expression{d12Req}) || res.(character.AttemptSentientItemControlTaken).Save().(character.SentientSaveForced).Failure.Reason() != d20.ChosenToFail {
		t.Errorf("chosen to fail: %v, requests %v", err, roller.Requests())
	}
	roller = dicetest.NewFixedRoller()
	rejected, err := attemptControl(t, roller, cmd(character.ControlPowerSpentUntilNextDawn, rollSave(0)))
	if rejected != (character.AttemptSentientItemControlRejected{}) || err != nil || len(roller.Requests()) != 0 {
		t.Errorf("spent power: %#v, %v, requests %v", rejected, err, roller.Requests())
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-104, FR-105, FR-109 - signals of the three results, the no-op bundle returns the same
// result, and invalid input is refused before any signal.
// Path: per case.
func TestAttemptSentientItemControlSignalsAndInvalidInput(t *testing.T) {
	t.Parallel()
	cmd := character.AttemptSentientItemControlCommand{CharacterID: holder, ItemCharisma: score(t, 14), Attempt: rollSave(0)}
	span, op := character.SpanAttemptSentientControl, "attempt_sentient_control"
	for _, c := range []struct {
		name  string
		cmd   character.AttemptSentientItemControlCommand
		faces []int
		want  map[string]any
	}{
		{"taken", cmd, []int{1, 3}, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed", d20.AttrKind: "saving_throw"}},
		{"resisted", cmd, []int{20}, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "resisted", d20.AttrKind: "saving_throw"}},
		{"power spent", character.AttemptSentientItemControlCommand{CharacterID: holder, Power: character.ControlPowerSpentUntilNextDawn, ItemCharisma: score(t, 14), Attempt: rollSave(0)}, nil,
			map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "power_spent"}},
	} {
		rec := telemetrytest.NewRecorder()
		res, err := attemptControl(t, dicetest.NewFixedRoller(c.faces...), c.cmd, character.WithTelemetry(rec.Telemetry()))
		plain, err2 := attemptControl(t, dicetest.NewFixedRoller(c.faces...), c.cmd)
		if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
			t.Fatalf("%s: results differ: %#v %v vs %#v %v", c.name, res, err, plain, err2)
		}
		checkOneCall(t, rec, span, op, telemetry.StatusUnset, c.want, d20.AttrKind)
	}
	use := must(character.NewAttemptSentientItemControl(dicetest.NewFixedRoller(1)))
	for name, c := range map[string]character.AttemptSentientItemControlCommand{
		"zero bearer":   {ItemCharisma: score(t, 14), Attempt: rollSave(0)},
		"zero Charisma": {CharacterID: holder, Attempt: rollSave(0)},
		"nil attempt":   {CharacterID: holder, ItemCharisma: score(t, 14)},
		"unknown power": {CharacterID: holder, ItemCharisma: score(t, 14), Attempt: rollSave(0), Power: 9},
	} {
		if r, err := use.Do(context.Background(), c); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v %v", name, r, err)
		}
	}
	if _, err := character.NewAttemptSentientItemControl(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	if r, err := (character.AttemptSentientItemControl{}).Do(context.Background(), cmd); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", r, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := use.Do(cancelled, cmd); r != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v %v", r, err)
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-102, FR-122 - a controlled bearer who takes damage repeats the save: a success
// ends the control with a SentientControlEnded event, a failure keeps it).
// Path: per case.
func TestRepeatSentientControlSave(t *testing.T) {
	t.Parallel()
	cmd := character.RepeatSentientControlSaveCommand{CharacterID: holder, Instance: inst("sword"), ItemCharisma: score(t, 14), Attempt: rollSave(0)}
	for _, c := range []struct {
		name  string
		faces []int
		end   bool
		want  map[string]any
	}{
		{"success ends", []int{14}, true, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed", d20.AttrKind: "saving_throw"}},
		{"failure continues", []int{13}, false, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "unchanged", d20.AttrKind: "saving_throw"}},
	} {
		rec := telemetrytest.NewRecorder()
		use := must(character.NewRepeatSentientControlSave(dicetest.NewFixedRoller(c.faces...), character.WithTelemetry(rec.Telemetry())))
		res, err := use.Do(context.Background(), cmd)
		plain, err2 := must(character.NewRepeatSentientControlSave(dicetest.NewFixedRoller(c.faces...))).Do(context.Background(), cmd)
		if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
			t.Fatalf("%s: results differ", c.name)
		}
		switch r := res.(type) {
		case character.RepeatSentientControlSaveEnded:
			if !c.end || r.Event().CharacterID() != holder || r.Event().InstanceID() != "sword" || r.Save().Failed() {
				t.Errorf("%s: ended %+v", c.name, r)
			}
		case character.RepeatSentientControlSaveContinues:
			if c.end || !r.Save().Failed() {
				t.Errorf("%s: continues %+v", c.name, r)
			}
		}
		checkOneCall(t, rec, character.SpanRepeatSentientControlSave, "repeat_sentient_control_save", telemetry.StatusUnset, c.want, d20.AttrKind)
	}
	use := must(character.NewRepeatSentientControlSave(dicetest.NewFixedRoller(1)))
	for name, c := range map[string]character.RepeatSentientControlSaveCommand{
		"zero bearer":    {Instance: inst("s"), ItemCharisma: score(t, 14), Attempt: rollSave(0)},
		"blank instance": {CharacterID: holder, ItemCharisma: score(t, 14), Attempt: rollSave(0)},
		"zero Charisma":  {CharacterID: holder, Instance: inst("s"), Attempt: rollSave(0)},
		"nil attempt":    {CharacterID: holder, Instance: inst("s"), ItemCharisma: score(t, 14)},
	} {
		if r, err := use.Do(context.Background(), c); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v %v", name, r, err)
		}
	}
	if _, err := character.NewRepeatSentientControlSave(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	if r, err := (character.RepeatSentientControlSave{}).Do(context.Background(), cmd); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", r, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := use.Do(cancelled, cmd); r != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v %v", r, err)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-106 - a roller error, panic or answer for another expression comes back as the
// *srderr.PortError of the dice port with a nil result from each sentient use case, including the d12 of the control attempt, recorded
// once (error status, one increment with error.type and no result) and not logged.
// Path: per case.
func TestSentientUseCasesRollerFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	wrong := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dice.NewResult(dice.MustExpression(2, dice.D20(), 0), []dice.DieRoll{dice.MustDieRoll(dice.D20(), 1), dice.MustDieRoll(dice.D20(), 1)})
	})
	fail := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause })
	panics := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("boom") })
	// A roller that answers the save and fails the duration.
	saveThenFail := func() character.Roller {
		n := 0
		return dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
			n++
			if n == 1 {
				return dicetest.NewFixedRoller(1).Roll(e)
			}
			return dice.Result{}, cause
		})
	}
	charisma := score(t, 14)
	uses := map[string]func(character.Roller, ...character.Option) error{
		"conflict": func(r character.Roller, o ...character.Option) error {
			_, err := must(character.NewResolveSentientItemConflict(r, o...)).Do(context.Background(), character.ResolveSentientItemConflictCommand{ItemCharisma: charisma, Attempt: rollSave(0), Demands: demandsOf(t)})
			return err
		},
		"control": func(r character.Roller, o ...character.Option) error {
			_, err := must(character.NewAttemptSentientItemControl(r, o...)).Do(context.Background(), character.AttemptSentientItemControlCommand{CharacterID: holder, ItemCharisma: charisma, Attempt: rollSave(0)})
			return err
		},
		"repeat": func(r character.Roller, o ...character.Option) error {
			_, err := must(character.NewRepeatSentientControlSave(r, o...)).Do(context.Background(), character.RepeatSentientControlSaveCommand{CharacterID: holder, Instance: inst("s"), ItemCharisma: charisma, Attempt: rollSave(0)})
			return err
		},
	}
	for name, run := range uses {
		for rname, r := range map[string]character.Roller{"fails": fail, "panics": panics, "another expression": wrong} {
			rec := telemetrytest.NewRecorder()
			err := run(r, character.WithTelemetry(rec.Telemetry()))
			var pe *srderr.PortError
			if !errors.As(err, &pe) || pe.Port != dice.Port || srderr.ErrorType(err) != "dice_roller_error" {
				t.Errorf("%s/%s: err = %v", name, rname, err)
				continue
			}
			if rname == "fails" && !errors.Is(err, cause) {
				t.Errorf("%s/%s: cause lost", name, rname)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
				t.Errorf("%s/%s: spans = %+v", name, rname, spans)
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, map[string]any{character.AttrOperation: map[string]string{"conflict": "resolve_sentient_conflict", "control": "attempt_sentient_control", "repeat": "repeat_sentient_control_save"}[name], telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"})
			noLogsOrViolations(t, rec)
		}
	}
	rec := telemetrytest.NewRecorder()
	if err := uses["control"](saveThenFail(), character.WithTelemetry(rec.Telemetry())); !errors.Is(err, cause) || len(rec.Meter.MeasurementsNamed(character.MetricOperations)) != 1 {
		t.Errorf("duration roll failure: %v", err)
	}
	_ = ability.Strength
	_ = modifier.Advantage{}
}
