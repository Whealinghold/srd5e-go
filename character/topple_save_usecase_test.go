package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

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

// toppleDC13 returns the offer of a Maul hit: modifier +3, Proficiency Bonus +2, DC 13.
func toppleDC13(t *testing.T) character.ToppleOffer {
	t.Helper()
	ws := weaponFixtures(t)
	o, err := character.OfferMastery(character.MasteryAttackFacts{
		WeaponID: "maul", Weapon: ws["maul"], Access: character.MasteryFromChoice{Choice: choiceOf(t, "maul")}, Outcome: hit(t),
		AbilityModifier: modifierOf(t, 3), ProficiencyBonus: character.ProficiencyBonusFor(character.MustLevel(1)),
	})
	offer, ok := o.(character.ToppleOffer)
	if err != nil || !ok || offer.DC() != 13 {
		t.Fatalf("offer = %#v, %v", o, err)
	}
	return offer
}

func newToppler(t *testing.T, roller character.Roller, opts ...character.Option) character.ToppleSaveResolver {
	t.Helper()
	u, err := character.NewToppleSaveResolver(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func rollSave(mods int, circumstances ...modifier.RollCircumstance) d20.RollAttempt {
	return d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(mods)), Circumstances: circumstances}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-097, FR-098, FR-100, FR-122, FR-104 - a rolled save
// against DC 13 succeeds at 13 and above (total 12 fails, 13 and 14 succeed), makes exactly one
// request of 1d20 (2d20 under Advantage or Disadvantage), and both results carry the DC and the
// d20.CheckResult (SRD 5.2.1, "Topple", p. 90).
// Path: per case.
func TestResolveToppleSaveRolled(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		faces                   []int
		attempt                 d20.RollAttempt
		wantProne               bool
		wantRequest             dice.Expression
	}{
		{"total 12", "FR-097", "unhappy (boundary)", []int{12}, rollSave(0), true, dice.MustExpression(1, dice.D20(), 0)},
		{"total 13", "FR-097", "unhappy (boundary)", []int{13}, rollSave(0), false, dice.MustExpression(1, dice.D20(), 0)},
		{"total 14", "FR-097", "happy", []int{14}, rollSave(0), false, dice.MustExpression(1, dice.D20(), 0)},
		{"modifier lifts the total", "FR-097", "unhappy (rule interaction)", []int{10}, rollSave(3), false, dice.MustExpression(1, dice.D20(), 0)},
		{"natural 20 is no automatic success", "FR-097", "unhappy (boundary)", []int{20}, rollSave(-8), true, dice.MustExpression(1, dice.D20(), 0)},
		{"Advantage keeps the higher", "FR-122", "unhappy (rule interaction)", []int{5, 15}, rollSave(0, modifier.Advantage{}), false, dice.MustExpression(2, dice.D20(), 0)},
		{"Disadvantage keeps the lower", "FR-122", "unhappy (rule interaction)", []int{5, 15}, rollSave(0, modifier.Disadvantage{}), true, dice.MustExpression(2, dice.D20(), 0)},
		{"Advantage and Disadvantage cancel", "FR-122", "unhappy (rule interaction)", []int{13}, rollSave(0, modifier.Advantage{}, modifier.Disadvantage{}), false, dice.MustExpression(1, dice.D20(), 0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := newToppler(t, roller).ResolveToppleSave(context.Background(), character.ResolveToppleSaveCommand{Offer: toppleDC13(t), Attempt: c.attempt})
			if err != nil {
				t.Fatal(err)
			}
			var detail character.ToppleSaveDetail
			switch r := res.(type) {
			case character.ToppleSaveTargetProne:
				detail = r.Detail
				if !c.wantProne || r.Condition() != content.ConditionProne {
					t.Errorf("prone, want resisted [%s, %s]", c.requirement, c.path)
				}
			case character.ToppleSaveTargetResisted:
				detail = r.Detail
				if c.wantProne {
					t.Errorf("resisted, want prone [%s, %s]", c.requirement, c.path)
				}
			}
			rolled, ok := detail.Basis.(character.ToppleSaveRolled)
			if !ok || detail.DC != 13 || rolled.Check.Target() != 13 || rolled.Check.Kind() != d20.SavingThrow {
				t.Errorf("detail = %#v", detail)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != c.wantRequest {
				t.Errorf("requests = %v, want one %v", reqs, c.wantRequest)
			}
		})
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-099, FR-100, FR-104 - ChooseToFail and AutomaticFailure
// fail the save without a request, carrying the forced-failure reason.
// Path: unhappy (rule interaction).
func TestResolveToppleSaveForcedFailure(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		attempt d20.SaveAttempt
		reason  d20.ForcedFailureReason
	}{"chosen to fail": {d20.ChooseToFail{}, d20.ChosenToFail}, "automatic": {d20.AutomaticFailure{}, d20.Automatic}} {
		roller := dicetest.NewFixedRoller(20)
		res, err := newToppler(t, roller).ResolveToppleSave(context.Background(), character.ResolveToppleSaveCommand{Offer: toppleDC13(t), Attempt: c.attempt})
		prone, ok := res.(character.ToppleSaveTargetProne)
		forced, isForced := prone.Detail.Basis.(character.ToppleSaveForced)
		if err != nil || !ok || !isForced || forced.Failure.Reason() != c.reason || prone.Detail.DC != 13 || len(roller.Requests()) != 0 {
			t.Errorf("%s: %#v, %v, %d requests", name, res, err, len(roller.Requests()))
		}
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-101, FR-111 - a roller error, panic or answer for another
// expression comes back as the *srderr.PortError of the dice port with a nil result, recorded once
// (error status, one increment with error.type and no result, no log); the no-op bundle gives the
// same error.
// Path: per case.
func TestResolveToppleSaveRollerFailures(t *testing.T) {
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
		{"roller fails", "FR-101", "unhappy (failure of collaborator)", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }), cause},
		{"roller panics", "FR-101", "unhappy (failure of collaborator)", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("boom") }), srderr.ErrPortPanic},
		{"another expression", "FR-101", "unhappy (failure of collaborator)", wrong, srderr.ErrContractViolation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.ResolveToppleSaveCommand{Offer: toppleDC13(t), Attempt: rollSave(0)}
			rec := telemetrytest.NewRecorder()
			res, err := newToppler(t, c.roller, character.WithTelemetry(rec.Telemetry())).ResolveToppleSave(context.Background(), cmd)
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || srderr.ErrorType(err) != "dice_roller_error" || !errors.Is(err, c.wantIs) {
				t.Fatalf("result = %#v, err = %v [%s]", res, err, c.requirement)
			}
			if plainRes, plainErr := newToppler(t, c.roller).ResolveToppleSave(context.Background(), cmd); plainRes != nil || !errors.As(plainErr, &pe) {
				t.Errorf("no-op bundle: %#v, %v", plainRes, plainErr)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanResolveToppleSave || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "span", spans[0].Attrs, map[string]any{telemetry.AttrOutcome: "failure"})
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, map[string]any{character.AttrOperation: "resolve_topple_save", telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"})
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-103, FR-107, FR-111 - the constructor refuses a nil
// roller; ResolveToppleSave refuses the zero use case, a zero offer and a nil or foreign attempt
// with ErrInvalidInput, and a cancelled context with ctx.Err(), all before any signal or roll.
// Path: unhappy (invalid input).
func TestResolveToppleSaveInvalidInputAndCancellation(t *testing.T) {
	t.Parallel()
	if _, err := character.NewToppleSaveResolver(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: err = %v", err)
	}
	roller, rec := dicetest.NewFixedRoller(5), telemetrytest.NewRecorder()
	use := newToppler(t, roller, character.WithTelemetry(rec.Telemetry()))
	offer := toppleDC13(t)
	for name, run := range map[string]func() (character.ResolveToppleSaveResult, error){
		"zero use case": func() (character.ResolveToppleSaveResult, error) {
			return character.ToppleSaveResolver{}.ResolveToppleSave(context.Background(), character.ResolveToppleSaveCommand{Offer: offer, Attempt: rollSave(0)})
		},
		"zero offer": func() (character.ResolveToppleSaveResult, error) {
			return use.ResolveToppleSave(context.Background(), character.ResolveToppleSaveCommand{Attempt: rollSave(0)})
		},
		"nil attempt": func() (character.ResolveToppleSaveResult, error) {
			return use.ResolveToppleSave(context.Background(), character.ResolveToppleSaveCommand{Offer: offer})
		},
		"foreign attempt": func() (character.ResolveToppleSaveResult, error) {
			return use.ResolveToppleSave(context.Background(), character.ResolveToppleSaveCommand{Offer: offer, Attempt: foreignAttempt{}})
		},
	} {
		if res, err := run(); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %#v, %v", name, res, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := use.ResolveToppleSave(cancelled, character.ResolveToppleSaveCommand{Offer: offer, Attempt: rollSave(0)})
	if res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricOperations)) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("cancelled: %#v, %v", res, err)
	}
}

// foreignAttempt is a SaveAttempt of this test package: not one of the three d20 variants.
type foreignAttempt struct{ d20.SaveAttempt }

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-111, FR-102 - a resolved save records the
// resolve_topple_save span (no error status), one operations increment with the result and outcome
// (target_prone is failure, target_resisted is success) and one duration with the operation only,
// no log and no d20_test signal; the no-op bundle gives the same result.
// Path: per case.
func TestResolveToppleSaveSignals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		face                    int
		attrs                   map[string]any
	}{
		{"target_prone", "FR-111", "happy", 5, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "target_prone"}},
		{"target_resisted", "FR-111", "happy", 19, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "target_resisted"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			cmd := character.ResolveToppleSaveCommand{Offer: toppleDC13(t), Attempt: rollSave(0)}
			res, err := newToppler(t, dicetest.NewFixedRoller(c.face), character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).ResolveToppleSave(context.Background(), cmd)
			plain, err2 := newToppler(t, dicetest.NewFixedRoller(c.face)).ResolveToppleSave(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("results differ: %#v, %v vs %#v, %v", res, err, plain, err2)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != "srd5e.character.resolve_topple_save" || !spans[0].Ended || spans[0].Status != telemetry.StatusUnset || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "span", spans[0].Attrs, c.attrs)
			counter := map[string]any{character.AttrOperation: "resolve_topple_save"}
			for k, v := range c.attrs {
				counter[k] = v
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, counter)
			d := onlyOne(t, rec, character.MetricDuration)
			if d.Value != 0.25 {
				t.Errorf("duration = %+v", d)
			}
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "resolve_topple_save"})
			noLogsOrViolations(t, rec)
		})
	}
}
