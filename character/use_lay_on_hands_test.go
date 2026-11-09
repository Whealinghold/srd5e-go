package character_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func useLayOnHands(t *testing.T, opts ...character.Option) character.UseLayOnHands {
	t.Helper()
	return must(character.NewUseLayOnHands(pldProgression(t), opts...))
}

// Requirement: doc/specs/class-paladin.md FR-047 to FR-056, FR-105, FR-123, FR-124 - UseLayOnHands applies an accepted use to the pool (restore 10 of 25 leaves 15), returns the Hit Points, the conditions removed,
// the Bonus Action cost and one LayOnHandsUsed event with the Character id, the Hit Points, the conditions and the pool left; a character with no state starts with a full pool.
// Path: per case (happy / unhappy: boundary, no stored state).
func TestUseLayOnHandsApplied(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name      string
		char      character.Character
		hp        int
		remove    []content.ConditionKind
		left, hit int
	}{
		{"restore part", pldWith(t, pldPaladin(t, 5), pldFull(t, 5)), 10, nil, 15, 10},
		{"restore all", pldPaladin(t, 5), 25, nil, 0, 25},
		{"no stored state", pldPaladin(t, 14), 0, []content.ConditionKind{cBlinded, cStunned}, 60, 0},
		{"cure Poisoned exactly", pldWith(t, pldPaladin(t, 1), pldFull(t, 1)), 0, []content.ConditionKind{cPoisoned}, 0, 0},
	} {
		before := c.char
		res, err := useLayOnHands(t).Run(context.Background(), character.UseLayOnHandsCommand{Character: c.char, HitPoints: c.hp, Remove: c.remove})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		a, ok := res.(character.UseLayOnHandsApplied)
		if !ok {
			t.Fatalf("%s: result = %#v", c.name, res)
		}
		if got := pldStateOf(t, a.Character()); got.LayOnHandsRemaining() != c.left || !got.Equal(a.State()) || a.Outcome().HitPoints() != c.hit || !c.char.Equal(before) {
			t.Errorf("%s: pool %d, outcome %+v", c.name, got.LayOnHandsRemaining(), a.Outcome())
		}
		ev, ok := a.Events()[0].(character.LayOnHandsUsed)
		if len(a.Events()) != 1 || !ok || ev.CharacterID() != c.char.ID() || ev.HitPoints() != c.hit || ev.Remaining() != c.left || len(ev.Conditions()) != len(c.remove) {
			t.Errorf("%s: events = %#v", c.name, a.Events())
		}
		a.Events()[0] = nil
		if a.Events()[0] == nil {
			t.Errorf("%s: Events returned shared storage", c.name)
		}
		if len(ev.Conditions()) > 0 {
			ev.Conditions()[0] = cProne
			if ev.Conditions()[0] == cProne {
				t.Errorf("%s: Conditions returned shared storage", c.name)
			}
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-048, FR-053, FR-055, FR-105, FR-106 - a rejected use reports every reason in the order of FR-055, leaves the Character unchanged and emits no event; a character with no
// Paladin level is NotAPaladin.
// Path: per case (unhappy: boundary, rule interaction, state error).
func TestUseLayOnHandsRejected(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		char   character.Character
		hp     int
		remove []content.ConditionKind
		want   []string
	}{
		{"one too many", pldPaladin(t, 5), 26, nil, []string{"exceeds_pool"}},
		{"empty pool", pldWith(t, pldPaladin(t, 5), pldState(t, 5, character.PaladinStateParts{ChannelDivinityRemaining: 2})), 1, nil, []string{"exceeds_pool"}},
		{"two reasons", pldWith(t, pldPaladin(t, 13), pldState(t, 13, character.PaladinStateParts{LayOnHandsRemaining: 4, ChannelDivinityRemaining: 3})), 1, []content.ConditionKind{cProne, cBlinded},
			[]string{"condition_not_removable", "restoring_touch_not_available", "exceeds_pool"}},
		{"empty request", pldPaladin(t, 5), 0, nil, []string{"empty_request"}},
		{"negative amount", pldPaladin(t, 5), -1, []content.ConditionKind{cProne}, []string{"invalid_amount", "condition_not_removable"}},
	} {
		before := c.char
		res, err := useLayOnHands(t).Run(context.Background(), character.UseLayOnHandsCommand{Character: c.char, HitPoints: c.hp, Remove: c.remove})
		rj, ok := res.(character.UseLayOnHandsRejected)
		if err != nil || !ok || !slices.Equal(pldReasons(rj.Reasons()), c.want) || !c.char.Equal(before) {
			t.Errorf("%s: result = %#v, %v; want reasons %v", c.name, res, err, c.want)
			continue
		}
		rj.Reasons()[0] = 0
		if rj.Reasons()[0] == 0 {
			t.Errorf("%s: Reasons returned shared storage", c.name)
		}
	}
	res, err := useLayOnHands(t).Run(context.Background(), character.UseLayOnHandsCommand{Character: fgtFighter(t, 5), HitPoints: 1})
	if _, ok := res.(character.UseLayOnHandsNotAPaladin); !ok || err != nil {
		t.Errorf("a Fighter = %#v, %v", res, err)
	}
}

// Requirement: doc/specs/class-paladin.md FR-101, FR-109, FR-110 - the use case gives the same result with the no-op and a recording telemetry; the zero use case, a zero Character, a cancelled context and a foreign state
// are errors matching srderr.ErrInvalidInput (or the context error) returned before any signal where stated, and the constructor rejects the zero progression.
// Path: per case (unhappy: invalid input).
func TestUseLayOnHandsTelemetryAndErrors(t *testing.T) {
	t.Parallel()
	cmd := character.UseLayOnHandsCommand{Character: pldPaladin(t, 5), HitPoints: 10}
	quiet, err1 := useLayOnHands(t).Run(context.Background(), cmd)
	rec, opts := fgtTelemetry()
	loud, err2 := useLayOnHands(t, opts...).Run(context.Background(), cmd)
	if err1 != nil || err2 != nil || quiet.(character.UseLayOnHandsApplied).State() != loud.(character.UseLayOnHandsApplied).State() {
		t.Errorf("results differ: %#v / %#v", quiet, loud)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanUseLayOnHands, "lay_on_hands", ""))
	if _, err := character.NewUseLayOnHands(character.PaladinProgression{}); err == nil {
		t.Error("the zero progression was accepted")
	} else {
		wantInvalid(t, err, "constructor")
	}
	if _, err := (character.UseLayOnHands{}).Run(context.Background(), cmd); err == nil {
		t.Error("the zero use case ran")
	} else {
		wantInvalid(t, err, "zero use case")
	}
	if _, err := useLayOnHands(t).Run(context.Background(), character.UseLayOnHandsCommand{HitPoints: 1}); err == nil {
		t.Error("a zero Character ran")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec2, opts2 := fgtTelemetry()
	if _, err := useLayOnHands(t, opts2...).Run(ctx, cmd); !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 {
		t.Errorf("cancelled = %v, spans %d", err, len(rec2.Tracer.Spans()))
	}
	rec3, opts3 := fgtTelemetry()
	if res, err := useLayOnHands(t, opts3...).Run(context.Background(), character.UseLayOnHandsCommand{Character: pldForeign(t, pldPaladin(t, 5)), HitPoints: 1}); res != nil || err == nil {
		t.Errorf("foreign state = %v, %v", res, err)
	} else {
		wantInvalid(t, err, "foreign state")
		spans := rec3.Tracer.Spans()
		if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || len(rec3.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 || len(rec3.Logger.Records()) != 0 {
			t.Errorf("foreign state signals: spans %+v", spans)
		}
	}
}
