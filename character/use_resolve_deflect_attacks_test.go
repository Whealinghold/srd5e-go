package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func useDeflect(t *testing.T, roller character.Roller, opts ...character.Option) character.ResolveDeflectAttacks {
	t.Helper()
	return must(character.NewResolveDeflectAttacks(mnkProgression(t), roller, opts...))
}

// Requirement: doc/specs/class-monk.md Table 4.6a, FR-051 to FR-055, FR-062, FR-127 - ResolveDeflectAttacks rolls one 1d10, uses the Character's Dexterity modifier (+3) and
// Monk level, returns the reduction and the damage after, leaves the Character unchanged and emits AttackDeflected; rows: low roll, a reduction above the damage, Fire only at 13.
// Path: per case (rows of Table 4.6a).
func TestResolveDeflectAttacksApplied(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name                         string
		level, roll, total, red, aft int
		types                        []content.DamageType
		redirect                     bool
	}{
		{"low roll", 3, 1, 20, 7, 13, bludgeoning, false}, {"reduction above the damage", 3, 10, 10, 16, 0, bludgeoning, true}, {"Fire only at 13", 13, 5, 30, 21, 9, fireOnly, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := mnkMonk(t, c.level)
			roller := dicetest.NewFixedRoller(c.roll)
			res, err := useDeflect(t, roller).Run(context.Background(), character.ResolveDeflectAttacksCommand{Character: in, DamageTypes: c.types, TotalDamage: c.total})
			r, ok := res.(character.DeflectAttacksApplied)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", res, err)
			}
			if r.Resolution().Reduction() != c.red || r.Resolution().DamageAfter() != c.aft || r.Resolution().CanRedirect() != c.redirect || !r.Character().Equal(in) {
				t.Errorf("= %+v", r.Resolution())
			}
			ev, isEv := r.Events()[0].(character.AttackDeflected)
			if len(r.Events()) != 1 || !isEv || ev.Rolled() != c.roll || ev.Reduction() != c.red || ev.DamageAfter() != c.aft || ev.CharacterID() != in.ID() {
				t.Errorf("events = %#v", r.Events())
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D10(), 0) {
				t.Errorf("requests = %v, want one 1d10", reqs)
			}
		})
	}
}

// Requirement: doc/specs/class-monk.md FR-052, FR-063, FR-023, FR-135, Table 4.11a - no Monk levels is notAMonk, level 2 is levelTooLow (also for Fire), Fire only below 13 is
// notEligible and does not roll; a negative damage total is an invalid command refused before any signal.
// Path: per case (unhappy: rule interaction, boundary).
func TestResolveDeflectAttacksNotAvailable(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		char  character.Character
		types []content.DamageType
		want  character.MonkReason
	}{
		{"no Monk levels", mnkChar(t, cl("fighter", 5)), bludgeoning, character.MonkNotAMonk}, {"level 2, Fire only", mnkMonk(t, 2), fireOnly, character.MonkLevelTooLow},
		{"level 3, Fire only", mnkMonk(t, 3), fireOnly, character.MonkNotEligible}, {"no types", mnkMonk(t, 3), nil, character.MonkNotEligible},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(5)
			res, err := useDeflect(t, roller).Run(context.Background(), character.ResolveDeflectAttacksCommand{Character: c.char, DamageTypes: c.types, TotalDamage: 10})
			if got, ok := res.(character.DeflectAttacksNotAvailable); err != nil || !ok || got.Reason != c.want || len(roller.Requests()) != 0 {
				t.Fatalf("= %#v, %v; want not available %v", res, err, c.want)
			}
		})
	}
	_, err := useDeflect(t, dicetest.NewFixedRoller(5)).Run(context.Background(), character.ResolveDeflectAttacksCommand{Character: mnkMonk(t, 3), DamageTypes: bludgeoning, TotalDamage: -1})
	wantInvalid(t, err, "negative total")
	foreign := must(character.WithFeatureState(mnkMonk(t, 3), character.MustFeatureStateKey[character.ChosenSubclass]("monk"), must(character.NewChosenSubclass("warrior_of_the_open_hand"))))
	res, err := useDeflect(t, dicetest.NewFixedRoller(5)).Run(context.Background(), character.ResolveDeflectAttacksCommand{Character: foreign, DamageTypes: bludgeoning, TotalDamage: 5})
	wantInvalid(t, err, "foreign state")
	if res != nil {
		t.Errorf("result = %v, want nil", res)
	}
}

// Requirement: doc/specs/class-monk.md FR-126, FR-129, FR-147 - a failing dice port is a *srderr.PortError with the cause; equal results with the no-op and a recording
// telemetry; invalid construction, the zero use case, a zero Character and a cancelled context are refused before any signal; success, rejection and port failure
// signals.
// Path: per case (unhappy: collaborator failure, invalid input, cancellation).
func TestResolveDeflectAttacksFailuresAndSignals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cmd := character.ResolveDeflectAttacksCommand{Character: mnkMonk(t, 5), DamageTypes: bludgeoning, TotalDamage: 12}
	for _, c := range mnkFailingRollers() {
		res, err := useDeflect(t, c.roller).Run(ctx, cmd)
		mnkWantPortFailure(t, res, err, c.cause)
	}
	rec, opts := mnkTelemetry()
	withRec, err := useDeflect(t, dicetest.NewFixedRoller(4), opts...).Run(ctx, cmd)
	plain, err2 := useDeflect(t, dicetest.NewFixedRoller(4)).Run(ctx, cmd)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanResolveDeflectAttacks, feature: "deflect_attacks", status: telemetry.StatusUnset, outcome: "success", logged: true})
	rec, opts = mnkTelemetry()
	_, _ = useDeflect(t, dicetest.NewFixedRoller(4), opts...).Run(ctx, character.ResolveDeflectAttacksCommand{Character: mnkMonk(t, 2), DamageTypes: bludgeoning, TotalDamage: 12})
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanResolveDeflectAttacks, feature: "deflect_attacks", status: telemetry.StatusUnset, outcome: "failure", rejection: "level_too_low", logged: true})
	rec, opts = mnkTelemetry()
	_, _ = useDeflect(t, dicetest.NewFixedRoller(), opts...).Run(ctx, cmd)
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanResolveDeflectAttacks, feature: "deflect_attacks", status: telemetry.StatusError, outcome: "failure", errorType: "dice_roller_error"})
	_, err = character.NewResolveDeflectAttacks(character.MonkProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewResolveDeflectAttacks(mnkProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.ResolveDeflectAttacks{}.Run(ctx, cmd)
	wantInvalid(t, err, "zero use case")
	_, err = useDeflect(t, dicetest.NewFixedRoller(1)).Run(ctx, character.ResolveDeflectAttacksCommand{})
	wantInvalid(t, err, "zero character")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	rec2, opts2 := mnkTelemetry()
	res, err := useDeflect(t, dicetest.NewFixedRoller(1), opts2...).Run(cancelled, cmd)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec2.Tracer.Spans()))
	}
}

// Requirement: doc/specs/class-monk.md FR-054 - a negative damage total is refused before any signal and any roll, and a total of 0 is valid.
// Path: per case (unhappy: invalid input, boundary).
func TestResolveDeflectAttacksRefusesNegativeTotalBeforeAnySignal(t *testing.T) {
	t.Parallel()
	rec, opts := mnkTelemetry()
	roller := dicetest.NewFixedRoller(5)
	res, err := useDeflect(t, roller, opts...).Run(context.Background(), character.ResolveDeflectAttacksCommand{Character: mnkMonk(t, 3), DamageTypes: bludgeoning, TotalDamage: -1})
	wantInvalid(t, err, "negative total")
	if res != nil || len(rec.Tracer.Spans()) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("result %v, spans %d, requests %v; want nothing", res, len(rec.Tracer.Spans()), roller.Requests())
	}
	zero, err := useDeflect(t, dicetest.NewFixedRoller(5)).Run(context.Background(), character.ResolveDeflectAttacksCommand{Character: mnkMonk(t, 3), DamageTypes: bludgeoning})
	if r, ok := zero.(character.DeflectAttacksApplied); err != nil || !ok || r.Resolution().DamageAfter() != 0 {
		t.Errorf("a zero total = %#v, %v", zero, err)
	}
}
