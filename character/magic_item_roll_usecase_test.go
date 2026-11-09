package character_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func recharger(t *testing.T, roller character.Roller, opts ...character.Option) character.RechargeMagicItem {
	t.Helper()
	return must(character.NewRechargeMagicItem(roller, opts...))
}

func rechargeCmd(charges character.ItemCharges, time content.ItemRechargeTime, recovery content.ChargeRecovery, moment character.RechargeMoment) character.RechargeMagicItemCommand {
	return character.RechargeMagicItemCommand{CharacterID: holder, Instance: inst("wand-1"), Charges: charges, Time: time, Recovery: recovery, Moment: moment}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-033, FR-035, FR-036, FR-037 and table "Charge recovery" - recharging applies the
// recovery when the moment matches the schedule, makes exactly one request for a rolled recovery and none otherwise, and never exceeds the maximum).
// Path: per case.
func TestRechargeMagicItem(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "The Next Dawn", "Charges", p. 206`
	d6 := content.ChargeRecovery(must(content.NewRegainsRolled(dice.MustExpression(1, dice.D6(), 0))))
	fixed3 := content.ChargeRecovery(must(content.NewRegainsFixed(3)))
	dawn, spec := content.RechargesAtNextDawn, content.RechargesAtSpecifiedTime
	cases := []struct {
		name, requirement, path string
		cmd                     character.RechargeMagicItemCommand
		faces                   []int
		wantCurrent             int
		notDue                  bool
		requests                int
	}{
		{"all at dawn", p, "happy", rechargeCmd(character.MustItemCharges(7, 2), dawn, content.RegainsAll{}, character.RechargeNextDawn), nil, 7, false, 0},
		{"fixed, below the expended", p, "happy", rechargeCmd(character.MustItemCharges(7, 2), dawn, fixed3, character.RechargeNextDawn), nil, 5, false, 0},
		{"fixed, above the expended", p, "unhappy (boundary)", rechargeCmd(character.MustItemCharges(7, 6), dawn, fixed3, character.RechargeNextDawn), nil, 7, false, 0},
		{"rolled, exactly expended", p, "unhappy (boundary)", rechargeCmd(character.MustItemCharges(7, 4), dawn, d6, character.RechargeNextDawn), []int{3}, 7, false, 1},
		{"rolled, below", p, "happy", rechargeCmd(character.MustItemCharges(7, 1), dawn, d6, character.RechargeNextDawn), []int{1}, 2, false, 1},
		{"rolled, above the expended is capped", p, "unhappy (boundary)", rechargeCmd(character.MustItemCharges(7, 4), dawn, d6, character.RechargeNextDawn), []int{6}, 7, false, 1},
		{"already full", p, "unhappy (empty case)", rechargeCmd(character.MustItemCharges(7, 7), dawn, content.RegainsAll{}, character.RechargeNextDawn), nil, 7, false, 0},
		{"no stated recovery", p, "unhappy (empty case)", rechargeCmd(character.MustItemCharges(7, 2), dawn, content.NoChargeRecovery{}, character.RechargeNextDawn), nil, 2, false, 0},
		{"specified time", p, "happy", rechargeCmd(character.MustItemCharges(7, 2), spec, content.RegainsAll{}, character.RechargeSpecifiedTime), nil, 7, false, 0},
		{"dawn reported for a specified-time item", p, "unhappy (state error)", rechargeCmd(character.MustItemCharges(7, 2), spec, d6, character.RechargeNextDawn), nil, 2, true, 0},
		{"specified time reported for a dawn item", p, "unhappy (state error)", rechargeCmd(character.MustItemCharges(7, 2), dawn, d6, character.RechargeSpecifiedTime), nil, 2, true, 0},
		{"does not recharge at dawn", p, "unhappy (state error)", rechargeCmd(character.MustItemCharges(7, 2), content.DoesNotRecharge, content.RegainsAll{}, character.RechargeNextDawn), nil, 2, true, 0},
		{"does not recharge at the specified time", p, "unhappy (state error)", rechargeCmd(character.MustItemCharges(7, 2), content.DoesNotRecharge, content.RegainsAll{}, character.RechargeSpecifiedTime), nil, 2, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := recharger(t, roller).Do(context.Background(), c.cmd)
			if err != nil || len(roller.Requests()) != c.requests {
				t.Fatalf("err = %v, %d requests [%s]", err, len(roller.Requests()), c.requirement)
			}
			if c.requests == 1 && !reflect.DeepEqual(roller.Requests()[0], dice.MustExpression(1, dice.D6(), 0)) {
				t.Errorf("request = %v", roller.Requests()[0])
			}
			switch r := res.(type) {
			case character.RechargeMagicItemApplied:
				ev := r.Event()
				if c.notDue || r.Charges().Current() != c.wantCurrent || r.Charges().Maximum() != 7 || ev.CharacterID() != holder || ev.InstanceID() != "wand-1" || ev.Regained() != c.wantCurrent-c.cmd.Charges.Current() {
					t.Errorf("applied = %+v [%s]", r, c.requirement)
				}
			case character.RechargeMagicItemNotDue:
				if !c.notDue || r.Charges() != c.cmd.Charges {
					t.Errorf("not due = %+v", r)
				}
			}
		})
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-105, FR-106, FR-109 - recharge records one span with the moment and one counter increment
// (not_due is a failure outcome without error status), the no-op bundle gives the same result, roller failures are recorded once and not
// logged, and invalid input is refused before any signal.
// Path: per case.
func TestRechargeMagicItemSignalsAndFailures(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		moment character.RechargeMoment
		want   map[string]any
	}{
		{"applied", character.RechargeNextDawn, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed"}},
		{"not due", character.RechargeSpecifiedTime, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "not_due"}},
	} {
		cmd := rechargeCmd(character.MustItemCharges(7, 2), content.RechargesAtNextDawn, content.RegainsAll{}, c.moment)
		rec := telemetrytest.NewRecorder()
		res, err := recharger(t, dicetest.NewFixedRoller(), character.WithTelemetry(rec.Telemetry())).Do(context.Background(), cmd)
		plain, err2 := recharger(t, dicetest.NewFixedRoller()).Do(context.Background(), cmd)
		if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
			t.Fatalf("%s: results differ", c.name)
		}
		want := map[string]any{character.AttrRechargeMoment: c.moment.String()}
		for k, v := range c.want {
			want[k] = v
		}
		checkOneCall(t, rec, character.SpanRechargeMagicItem, "recharge_magic_item", telemetry.StatusUnset, want, character.AttrRechargeMoment)
	}
	d6 := content.ChargeRecovery(must(content.NewRegainsRolled(dice.MustExpression(1, dice.D6(), 0))))
	cmd := rechargeCmd(character.MustItemCharges(7, 2), content.RechargesAtNextDawn, d6, character.RechargeNextDawn)
	cause := errors.New("boom")
	for name, r := range map[string]character.Roller{
		"fails":              dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }),
		"panics":             dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("boom") }),
		"another expression": dicetest.NewFixedRoller(4, 4),
	} {
		if name == "another expression" {
			r = dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
				return dice.NewResult(dice.MustExpression(2, dice.D6(), 0), []dice.DieRoll{dice.MustDieRoll(dice.D6(), 1), dice.MustDieRoll(dice.D6(), 1)})
			})
		}
		rec := telemetrytest.NewRecorder()
		res, err := recharger(t, r, character.WithTelemetry(rec.Telemetry())).Do(context.Background(), cmd)
		var pe *srderr.PortError
		if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || srderr.ErrorType(err) != "dice_roller_error" {
			t.Errorf("%s: %v %v", name, res, err)
		}
		if spans := rec.Tracer.Spans(); len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
			t.Errorf("%s: spans %+v", name, spans)
		}
		sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, map[string]any{character.AttrOperation: "recharge_magic_item", telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"})
		noLogsOrViolations(t, rec)
	}
	use := recharger(t, dicetest.NewFixedRoller())
	good := rechargeCmd(character.MustItemCharges(7, 2), content.RechargesAtNextDawn, content.RegainsAll{}, character.RechargeNextDawn)
	for name, mut := range map[string]func(*character.RechargeMagicItemCommand){
		"zero holder":       func(c *character.RechargeMagicItemCommand) { c.CharacterID = character.Identifier{} },
		"blank instance":    func(c *character.RechargeMagicItemCommand) { c.Instance = "" },
		"zero charges":      func(c *character.RechargeMagicItemCommand) { c.Charges = character.ItemCharges{} },
		"nil recovery":      func(c *character.RechargeMagicItemCommand) { c.Recovery = nil },
		"unknown time":      func(c *character.RechargeMagicItemCommand) { c.Time = 0 },
		"time past the set": func(c *character.RechargeMagicItemCommand) { c.Time = 9 },
		"unknown moment":    func(c *character.RechargeMagicItemCommand) { c.Moment = 0 },
	} {
		cmd := good
		mut(&cmd)
		if r, err := use.Do(context.Background(), cmd); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v %v", name, r, err)
		}
	}
	if _, err := character.NewRechargeMagicItem(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	if r, err := (character.RechargeMagicItem{}).Do(context.Background(), good); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", r, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := use.Do(cancelled, good); r != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v %v", r, err)
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-013, FR-014 - mixing N potions makes N - 1 requests of 1d100 and returns the
// table's effects in roll order; fewer than two potions are rejected with no request).
// Path: per case.
func TestRollPotionMiscibility(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Mixing Potions", p. 204`
	d100 := dice.MustExpression(1, dice.D100(), 0)
	cases := []struct {
		name, requirement, path string
		potions                 int
		faces                   []int
		wantRequests            int
		rejected                bool
	}{
		{"two potions", p, "happy", 2, []int{1}, 1, false},
		{"four potions in order", p, "happy", 4, []int{1, 50, 100}, 3, false},
		{"one potion", p, "unhappy (boundary)", 1, nil, 0, true},
		{"no potions", p, "unhappy (boundary)", 0, nil, 0, true},
		{"negative", p, "unhappy (boundary)", -1, nil, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := must(character.NewRollPotionMiscibility(roller)).Do(context.Background(), character.RollPotionMiscibilityCommand{Potions: c.potions})
			if err != nil || len(roller.Requests()) != c.wantRequests {
				t.Fatalf("err = %v, %d requests", err, len(roller.Requests()))
			}
			for _, r := range roller.Requests() {
				if r != d100 {
					t.Errorf("request = %v", r)
				}
			}
			if c.rejected {
				if r, ok := res.(character.RollPotionMiscibilityRejected); !ok || r.Reason() != character.PotionMixTooFewPotions {
					t.Errorf("= %#v", res)
				}
				return
			}
			got := res.(character.RollPotionMiscibilityRolled).Effects()
			var want []content.PotionMiscibilityEffect
			for _, f := range c.faces {
				want = append(want, content.PotionMiscibilityTable{}.MustForRoll(f))
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("effects = %v, want %v [%s]", got, want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-105, FR-106, FR-109 - mixing and the materials check record one span and one increment
// each, give the same result with the no-op bundle, record roller failures once and refuse invalid input before any signal.
// Path: per case.
func TestMixPotionsAndMaterialsSignalsAndFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mix := func(r character.Roller, o ...character.Option) character.RollPotionMiscibility {
		return must(character.NewRollPotionMiscibility(r, o...))
	}
	mats := func(r character.Roller, o ...character.Option) character.CheckCraftingMaterials {
		return must(character.NewCheckCraftingMaterials(r, o...))
	}
	for _, c := range []struct {
		name    string
		run     func(character.Roller, ...character.Option) (any, error)
		span    string
		op      string
		faces   []int
		want    map[string]any
		wantRes any
	}{
		{"mixed", func(r character.Roller, o ...character.Option) (any, error) {
			return mix(r, o...).Do(ctx, character.RollPotionMiscibilityCommand{Potions: 2})
		}, character.SpanMixPotions, "mix_potions", []int{50}, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "rolled"}, nil},
		{"too few", func(r character.Roller, o ...character.Option) (any, error) {
			return mix(r, o...).Do(ctx, character.RollPotionMiscibilityCommand{Potions: 1})
		}, character.SpanMixPotions, "mix_potions", nil, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "too_few_potions"}, nil},
		{"materials available", func(r character.Roller, o ...character.Option) (any, error) {
			return mats(r, o...).Do(ctx, character.CheckCraftingMaterialsCommand{Settlement: content.SettlementCity})
		}, character.SpanCheckCraftingMaterials, "check_crafting_materials", []int{75}, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed"}, character.CheckCraftingMaterialsAvailable{}},
		{"materials unavailable", func(r character.Roller, o ...character.Option) (any, error) {
			return mats(r, o...).Do(ctx, character.CheckCraftingMaterialsCommand{Settlement: content.SettlementCity})
		}, character.SpanCheckCraftingMaterials, "check_crafting_materials", []int{76}, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "unavailable"}, nil},
	} {
		rec := telemetrytest.NewRecorder()
		res, err := c.run(dicetest.NewFixedRoller(c.faces...), character.WithTelemetry(rec.Telemetry()))
		plain, err2 := c.run(dicetest.NewFixedRoller(c.faces...))
		if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
			t.Fatalf("%s: results differ: %#v %v vs %#v %v", c.name, res, err, plain, err2)
		}
		if c.wantRes != nil && res != c.wantRes {
			t.Errorf("%s: result = %#v", c.name, res)
		}
		checkOneCall(t, rec, c.span, c.op, telemetry.StatusUnset, c.want)
	}
	if u, ok := must(mats(dicetest.NewFixedRoller(76)).Do(ctx, character.CheckCraftingMaterialsCommand{Settlement: content.SettlementTown})).(character.CheckCraftingMaterialsUnavailable); !ok || u.WaitAtLeastDays() != 7 {
		t.Errorf("wait = %#v", u)
	}
	cause := errors.New("boom")
	fail := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause })
	for name, run := range map[string]func(character.Roller, ...character.Option) error{
		"mix": func(r character.Roller, o ...character.Option) error {
			_, e := mix(r, o...).Do(ctx, character.RollPotionMiscibilityCommand{Potions: 3})
			return e
		},
		"materials": func(r character.Roller, o ...character.Option) error {
			_, e := mats(r, o...).Do(ctx, character.CheckCraftingMaterialsCommand{Settlement: content.SettlementCity})
			return e
		},
	} {
		rec := telemetrytest.NewRecorder()
		err := run(fail, character.WithTelemetry(rec.Telemetry()))
		var pe *srderr.PortError
		if !errors.As(err, &pe) || !errors.Is(err, cause) || srderr.ErrorType(err) != "dice_roller_error" {
			t.Errorf("%s: %v", name, err)
		}
		if spans := rec.Tracer.Spans(); len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(rec.Meter.MeasurementsNamed(character.MetricOperations)) != 1 {
			t.Errorf("%s: spans %+v", name, spans)
		}
		noLogsOrViolations(t, rec)
		// The second request of a three-potion mix fails after the first answered.
	}
	n := 0
	second := dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
		if n++; n == 2 {
			return dice.Result{}, cause
		}
		return dicetest.NewFixedRoller(10).Roll(e)
	})
	if _, err := mix(second).Do(ctx, character.RollPotionMiscibilityCommand{Potions: 3}); !errors.Is(err, cause) {
		t.Errorf("second roll: %v", err)
	}
	for _, nilRoller := range []func() error{
		func() error { _, e := character.NewRollPotionMiscibility(nil); return e },
		func() error { _, e := character.NewCheckCraftingMaterials(nil); return e },
	} {
		if err := nilRoller(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("nil roller: %v", err)
		}
	}
	if _, err := mats(dicetest.NewFixedRoller(1)).Do(ctx, character.CheckCraftingMaterialsCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("unknown settlement: %v", err)
	}
	if _, err := (character.CheckCraftingMaterials{}).Do(ctx, character.CheckCraftingMaterialsCommand{Settlement: content.SettlementCity}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero materials: %v", err)
	}
	if _, err := (character.RollPotionMiscibility{}).Do(ctx, character.RollPotionMiscibilityCommand{Potions: 2}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero mix: %v", err)
	}
	for _, n := range []int{character.MaxMixedPotions + 1, math.MaxInt} {
		roller := dicetest.NewFixedRoller(1)
		if _, err := mix(roller).Do(ctx, character.RollPotionMiscibilityCommand{Potions: n}); !errors.Is(err, srderr.ErrInvalidInput) || len(roller.Requests()) != 0 {
			t.Errorf("%d potions: %v", n, err)
		}
	}
	max := dicetest.NewMinimumRoller()
	if res, err := mix(max).Do(ctx, character.RollPotionMiscibilityCommand{Potions: character.MaxMixedPotions}); err != nil || len(res.(character.RollPotionMiscibilityRolled).Effects()) != character.MaxMixedPotions-1 {
		t.Errorf("max potions: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := mix(dicetest.NewFixedRoller(1)).Do(cancelled, character.RollPotionMiscibilityCommand{Potions: 2}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled mix: %v", err)
	}
	if _, err := mats(dicetest.NewFixedRoller(1)).Do(cancelled, character.CheckCraftingMaterialsCommand{Settlement: content.SettlementCity}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled materials: %v", err)
	}
}
