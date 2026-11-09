package character_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// failingRollers are the three ways the dice port breaks its contract: it fails, it panics, it answers another expression.
func failingRollers(cause error) map[string]character.Roller {
	return map[string]character.Roller{
		"fails":  dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }),
		"panics": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("boom") }),
		"another expression": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
			return dice.NewResult(dice.MustExpression(2, dice.D6(), 0), []dice.DieRoll{dice.MustDieRoll(dice.D6(), 1), dice.MustDieRoll(dice.D6(), 1)})
		}),
	}
}

// wantPortFailure checks a use case returned a dice_roller_error and recorded it once, with no log.
func wantPortFailure(t *testing.T, name, op string, rec *telemetrytest.Recorder, res any, err error) {
	t.Helper()
	var pe *srderr.PortError
	if res != nil {
		t.Errorf("%s: result = %v", name, res)
	}
	if !errors.As(err, &pe) || pe.Port != dice.Port || srderr.ErrorType(err) != "dice_roller_error" {
		t.Errorf("%s: err = %v", name, err)
	}
	if spans := rec.Tracer.Spans(); len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
		t.Errorf("%s: spans %+v", name, spans)
	}
	sameAttrs(t, name+" operations", onlyOne(t, rec, character.MetricOperations).Attrs, map[string]any{character.AttrOperation: op, telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"})
	noLogsOrViolations(t, rec)
}

func potionRoller(t *testing.T, r character.Roller, opts ...character.Option) character.RollPotionOfHealing {
	t.Helper()
	return must(character.NewRollPotionOfHealing(r, opts...))
}

// Requirement: doc/specs/magic-item-effects.md FR-024, FR-025 (SRD 5.2.1, "Potions of Healing", p. 236) - the greater, superior and supreme grades
// make exactly one request for the grade's expression and return the total (the range ends for the minimum and maximum rollers); the Common grade makes
// no request and reports the roll belongs elsewhere.
// Path: per case.
func TestRollPotionOfHealing(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		rarity content.ItemRarity
		roller *dicetest.FixedRoller
		grade  character.PotionOfHealingGrade
		amount int
	}{
		{"greater minimum", content.RarityUncommon, dicetest.NewMinimumRoller(), character.GradeGreater, 8},
		{"greater maximum", content.RarityUncommon, dicetest.NewMaximumRoller(), character.GradeGreater, 20},
		{"superior minimum", content.RarityRare, dicetest.NewMinimumRoller(), character.GradeSuperior, 16},
		{"superior maximum", content.RarityRare, dicetest.NewMaximumRoller(), character.GradeSuperior, 40},
		{"supreme minimum", content.RarityVeryRare, dicetest.NewMinimumRoller(), character.GradeSupreme, 30},
		{"supreme maximum", content.RarityVeryRare, dicetest.NewMaximumRoller(), character.GradeSupreme, 60},
		{"greater listed faces", content.RarityUncommon, dicetest.NewFixedRoller(1, 2, 3, 4), character.GradeGreater, 14},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := potionRoller(t, c.roller).Do(context.Background(), character.RollPotionOfHealingCommand{Rarity: c.rarity})
			r, ok := res.(character.RollPotionOfHealingRolled)
			if err != nil || !ok || r.Grade() != c.grade || r.Amount() != c.amount {
				t.Fatalf("result = %+v %v", res, err)
			}
			if reqs := c.roller.Requests(); len(reqs) != 1 || reqs[0] != c.grade.Dice() {
				t.Errorf("requests = %v", reqs)
			}
		})
	}
	for _, c := range []struct {
		name   string
		rarity content.ItemRarity
	}{{"common", content.RarityCommon}} {
		roller := dicetest.NewFixedRoller()
		res, err := potionRoller(t, roller).Do(context.Background(), character.RollPotionOfHealingCommand{Rarity: c.rarity})
		if r, ok := res.(character.RollPotionOfHealingStandardRolledElsewhere); err != nil || !ok || r.Grade() != character.GradeStandard || len(roller.Requests()) != 0 {
			t.Errorf("%s = %+v %v", c.name, res, err)
		}
	}
	for _, rarity := range []content.ItemRarity{content.RarityLegendary, content.RarityArtifact} {
		roller := dicetest.NewFixedRoller()
		res, err := potionRoller(t, roller).Do(context.Background(), character.RollPotionOfHealingCommand{Rarity: rarity})
		if r, ok := res.(character.RollPotionOfHealingNotListed); err != nil || !ok || r.Rarity() != rarity || len(roller.Requests()) != 0 {
			t.Errorf("%s = %+v %v", rarity, res, err)
		}
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-067 - RollPotionOfHealing gives the same result with the no-op and a recording telemetry, records one span
// and one counter increment per call with outcome success or failure only, and logs nothing; a rolled amount is not clamped.
// Path: per case.
func TestRollPotionOfHealingSignals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		rarity  content.ItemRarity
		outcome string
		result  string
	}{
		{"rolled", content.RarityRare, "success", "rolled"},
		{"not listed", content.RarityLegendary, "failure", "not_listed"},
		{"rolled elsewhere", content.RarityCommon, "failure", "rolled_elsewhere"},
	} {
		cmd := character.RollPotionOfHealingCommand{Rarity: c.rarity}
		rec := telemetrytest.NewRecorder()
		res, err := potionRoller(t, dicetest.NewMaximumRoller(), character.WithTelemetry(rec.Telemetry())).Do(context.Background(), cmd)
		plain, err2 := potionRoller(t, dicetest.NewMaximumRoller()).Do(context.Background(), cmd)
		if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
			t.Fatalf("%s: results differ", c.name)
		}
		checkOneCall(t, rec, character.SpanRollPotionOfHealing, "roll_potion_healing", telemetry.StatusUnset, map[string]any{telemetry.AttrOutcome: c.outcome, character.AttrResult: c.result})
		noLogsOrViolations(t, rec)
	}
	cause := errors.New("boom")
	for name, r := range failingRollers(cause) {
		rec := telemetrytest.NewRecorder()
		res, err := potionRoller(t, r, character.WithTelemetry(rec.Telemetry())).Do(context.Background(), character.RollPotionOfHealingCommand{Rarity: content.RarityRare})
		wantPortFailure(t, name, "roll_potion_healing", rec, res, err)
		if name == "fails" && !errors.Is(err, cause) {
			t.Error("the cause is lost")
		}
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-067 - an invalid rarity, a zero use case and a cancelled context are refused before any signal.
// Path: unhappy (invalid input).
func TestRollPotionOfHealingInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	use := potionRoller(t, dicetest.NewFixedRoller(), character.WithTelemetry(rec.Telemetry()))
	for _, rarity := range []content.ItemRarity{0, content.RarityArtifact + 1} {
		if r, err := use.Do(context.Background(), character.RollPotionOfHealingCommand{Rarity: rarity}); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("rarity %d: %v %v", rarity, r, err)
		}
	}
	if r, err := (character.RollPotionOfHealing{}).Do(context.Background(), character.RollPotionOfHealingCommand{Rarity: content.RarityRare}); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", r, err)
	}
	if _, err := character.NewRollPotionOfHealing(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := use.Do(cancelled, character.RollPotionOfHealingCommand{Rarity: content.RarityRare}); r != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v %v", r, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("a refused call emitted a signal")
	}
}

func ringRoller(t *testing.T, r character.Roller, opts ...character.Option) character.RollSpellStoringRingLevels {
	t.Helper()
	return must(character.NewRollSpellStoringRingLevels(r, opts...))
}

// Requirement: doc/specs/magic-item-effects.md FR-049, FR-067 (SRD 5.2.1, "Ring of Spell Storing", p. 239) - one request of 1d6 gives levels roll - 1
// (a 1 gives 0 and a 6 gives 5); a failing, panicking or wrong-expression roller is a recorded dice_roller_error; refused input emits nothing; the
// no-op and a recording telemetry give the same result.
// Path: per case.
func TestRollSpellStoringRingLevels(t *testing.T) {
	t.Parallel()
	for faces, want := range map[int]int{1: 0, 2: 1, 3: 2, 4: 3, 5: 4, 6: 5} {
		roller := dicetest.NewFixedRoller(faces)
		rec := telemetrytest.NewRecorder()
		res, err := ringRoller(t, roller, character.WithTelemetry(rec.Telemetry())).Do(context.Background(), character.RollSpellStoringRingLevelsCommand{})
		if r, ok := res.(character.RollSpellStoringRingLevelsRolled); err != nil || !ok || r.Levels() != want {
			t.Errorf("face %d = %+v %v", faces, res, err)
		}
		if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D6(), 0) {
			t.Errorf("requests = %v", reqs)
		}
		plain, _ := ringRoller(t, dicetest.NewFixedRoller(faces)).Do(context.Background(), character.RollSpellStoringRingLevelsCommand{})
		if !reflect.DeepEqual(res, plain) {
			t.Error("telemetry changed the result")
		}
		checkOneCall(t, rec, character.SpanRollSpellStoringLevels, "roll_spell_storing_levels", telemetry.StatusUnset, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "rolled"})
		noLogsOrViolations(t, rec)
	}
	cause := errors.New("boom")
	for name, r := range failingRollers(cause) {
		rec := telemetrytest.NewRecorder()
		res, err := ringRoller(t, r, character.WithTelemetry(rec.Telemetry())).Do(context.Background(), character.RollSpellStoringRingLevelsCommand{})
		wantPortFailure(t, name, "roll_spell_storing_levels", rec, res, err)
	}
	if r, err := (character.RollSpellStoringRingLevels{}).Do(context.Background(), character.RollSpellStoringRingLevelsCommand{}); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", r, err)
	}
	if _, err := character.NewRollSpellStoringRingLevels(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := ringRoller(t, dicetest.NewFixedRoller(1)).Do(cancelled, character.RollSpellStoringRingLevelsCommand{}); r != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v %v", r, err)
	}
}

// Requirement: doc/specs/magic-item-effects.md section 7 - the signal names, attribute keys and values of the three use cases are a public contract:
// pinned here, listed in doc/observability/signals.md, and (the attribute values) in doc/rules/observability.md section 5.
// Path: happy.
func TestChargedItemSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile("../doc/rules/observability.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ got, want string }{
		{character.SpanRollPotionOfHealing, "srd5e.character.roll_potion_healing"},
		{character.SpanResolveLastCharge, "srd5e.character.resolve_last_charge"},
		{character.SpanRollSpellStoringLevels, "srd5e.character.roll_spell_storing_levels"},
		{character.AttrLastChargeConsequence, "srd5e.character.last_charge_consequence"},
		{character.ValueOperationRollPotionOfHealing, "roll_potion_healing"},
		{character.ValueOperationResolveLastCharge, "resolve_last_charge"},
		{character.ValueOperationRollSpellStoringLevels, "roll_spell_storing_levels"},
		{character.ValueResultResolved, "resolved"},
		{character.ValueResultNotListed, "not_listed"},
		{character.ValueResultRolledElsewhere, "rolled_elsewhere"},
		{character.ValueRejectionChargesRemain, "charges_remain"},
		{character.ChargesRemain.String(), "charges_remain"},
		{character.LastChargeNone.String(), "none"},
		{character.LastChargeDestroyed.String(), "destroyed"},
		{character.LastChargeBecomesNonmagical.String(), "becomes_nonmagical"},
		{character.LastChargeRetainsBonusLosesOtherProperties.String(), "retains_bonus"},
	} {
		if c.got != c.want {
			t.Errorf("signal = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") {
			t.Errorf("%q is missing from doc/observability/signals.md", c.want)
		}
		if span := strings.HasPrefix(c.want, "srd5e.character.") && c.want != character.AttrLastChargeConsequence; !span && !strings.Contains(string(rules), c.want) {
			t.Errorf("%q is missing from doc/rules/observability.md section 5", c.want)
		}
	}
	if character.ResolveLastChargeRejection(0).String() != "unknown" {
		t.Error("zero rejection name")
	}
}
