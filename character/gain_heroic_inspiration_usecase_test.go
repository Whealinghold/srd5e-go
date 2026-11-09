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

// gainOutcome names the variant of a result and returns its holder and events.
func gainOutcome(t *testing.T, res character.GainHeroicInspirationResult) (variant string, holder character.Character, events []character.Event) {
	t.Helper()
	switch r := res.(type) {
	case character.GainHeroicInspirationReceived:
		return "received", r.Holder(), r.Events()
	case character.GainHeroicInspirationExcessLost:
		return "lost", r.Holder(), r.Events()
	case character.GainHeroicInspirationExcessGiven:
		return "given", r.Holder(), r.Events()
	case character.GainHeroicInspirationRejected:
		return r.Reason().String(), character.Character{}, nil
	}
	t.Fatalf("unexpected result %#v", res)
	return "", character.Character{}, nil
}

// Requirement: doc/specs/heroic-inspiration.md FR-031, FR-032, FR-039, FR-037, FR-027 -
// Gain equals ResolveExcessInspiration mapped to the same-named result variant with
// the same holder, recipient, events and reason (differential over G1 to G9), and its
// events are the aggregate's.
// Path: per case.
func TestGainMapsEveryResolution(t *testing.T) {
	t.Parallel()
	for _, c := range gainCases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			choice, holder, _ := choiceFor(t, c)
			res, err := character.NewGainHeroicInspiration().Gain(context.Background(), character.GainHeroicInspirationCommand{Holder: holder, Excess: choice})
			if err != nil {
				t.Fatal(err)
			}
			want := character.ResolveExcessInspiration(holder, choice)
			var same bool
			switch w := want.(type) {
			case character.ExcessResolutionReceived:
				r, ok := res.(character.GainHeroicInspirationReceived)
				same = ok && r.Holder().Equal(w.Holder()) && reflect.DeepEqual(r.Events(), w.Events())
			case character.ExcessResolutionExcessLost:
				r, ok := res.(character.GainHeroicInspirationExcessLost)
				same = ok && r.Holder().Equal(w.Holder()) && reflect.DeepEqual(r.Events(), w.Events())
			case character.ExcessResolutionExcessGiven:
				r, ok := res.(character.GainHeroicInspirationExcessGiven)
				same = ok && r.Holder().Equal(w.Holder()) && r.Recipient().Equal(w.Recipient()) && reflect.DeepEqual(r.Events(), w.Events())
			case character.ExcessResolutionRejected:
				r, ok := res.(character.GainHeroicInspirationRejected)
				same = ok && r.Reason() == w.Reason()
			}
			if !same {
				t.Errorf("Gain = %#v, want the mapping of %#v [%s]", res, want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-011 - an accepted or rejected gain
// leaves the supplied characters unchanged, and the Events slice of Received,
// ExcessLost and ExcessGiven results is a copy.
// Path: unhappy (immutability).
func TestGainDoesNotMutateAndCopiesEvents(t *testing.T) {
	t.Parallel()
	free, held, friend, inspired := pc(t, "holder", character.NotInspired), pc(t, "holder", character.Inspired), pc(t, "friend", character.NotInspired), pc(t, "friend", character.Inspired)
	gain := character.NewGainHeroicInspiration()
	for name, cmd := range map[string]character.GainHeroicInspirationCommand{
		"received": {Holder: free, Excess: character.LoseExcessInspiration{}},
		"lost":     {Holder: held, Excess: character.LoseExcessInspiration{}},
		"given":    {Holder: held, Excess: character.GiveExcessInspirationTo{Recipient: friend}},
		"rejected": {Holder: held, Excess: character.GiveExcessInspirationTo{Recipient: inspired}},
	} {
		res, err := gain.Gain(context.Background(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		_, _, ev := gainOutcome(t, res)
		if len(ev) > 0 {
			ev[0] = nil
			if _, _, again := gainOutcome(t, res); again[0] == nil {
				t.Errorf("%s: Events shares its storage", name)
			}
		}
	}
	if !free.Equal(pc(t, "holder", character.NotInspired)) || !held.Equal(pc(t, "holder", character.Inspired)) || !friend.Equal(pc(t, "friend", character.NotInspired)) || !inspired.Equal(pc(t, "friend", character.Inspired)) {
		t.Error("Gain altered a supplied character")
	}
	lost, _ := gain.Gain(context.Background(), character.GainHeroicInspirationCommand{Holder: held})
	if _, _, ev := gainOutcome(t, lost); append(ev, nil) == nil || len(lost.(character.GainHeroicInspirationExcessLost).Events()) != 0 {
		t.Error("lost Events is not an independent empty slice")
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-028 - Gain refuses the zero use
// case, a zero holder and a give to a zero recipient with ErrInvalidInput, and a
// cancelled context with ctx.Err(), all before any signal.
// Path: unhappy (invalid input).
func TestGainInvalidInputAndCancellation(t *testing.T) {
	t.Parallel()
	held, rec := pc(t, "holder", character.Inspired), telemetrytest.NewRecorder()
	gain := character.NewGainHeroicInspiration(character.WithTelemetry(rec.Telemetry()))
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for name, run := range map[string]func() (character.GainHeroicInspirationResult, error){
		"zero use case": func() (character.GainHeroicInspirationResult, error) {
			return character.GainHeroicInspiration{}.Gain(context.Background(), character.GainHeroicInspirationCommand{Holder: held})
		},
		"zero holder": func() (character.GainHeroicInspirationResult, error) {
			return gain.Gain(context.Background(), character.GainHeroicInspirationCommand{})
		},
		"zero recipient": func() (character.GainHeroicInspirationResult, error) {
			return gain.Gain(context.Background(), character.GainHeroicInspirationCommand{Holder: held, Excess: character.GiveExcessInspirationTo{}})
		},
	} {
		if res, err := run(); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %#v, %v", name, res, err)
		}
	}
	res, err := gain.Gain(cancelled, character.GainHeroicInspirationCommand{Holder: held})
	if res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricOperations)) != 0 {
		t.Errorf("cancelled: %#v, %v", res, err)
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-028, FR-029, FR-036 - Gain records
// the gain_heroic_inspiration span (never an error status), one operations
// increment (operation gain, outcome success for gained, excess_lost and
// excess_given; failure with result rejected and the rejection otherwise), one
// duration with the operation only, and no log; results equal under the no-op and
// a recording bundle.
// Path: per case.
func TestGainSignals(t *testing.T) {
	t.Parallel()
	for _, c := range gainCases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			choice, holder, _ := choiceFor(t, c)
			cmd := character.GainHeroicInspirationCommand{Holder: holder, Excess: choice}
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			res, err := character.NewGainHeroicInspiration(character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).Gain(context.Background(), cmd)
			plain, err2 := character.NewGainHeroicInspiration().Gain(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("results differ: %#v, %v vs %#v, %v", res, err, plain, err2)
			}
			attrs := map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: map[string]string{"received": "gained", "lost": "excess_lost", "given": "excess_given"}[c.want]}
			if c.want == "recipient_is_holder" || c.want == "recipient_already_inspired" {
				attrs = map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: c.want}
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanGainHeroicInspiration || !spans[0].Ended || spans[0].Status != telemetry.StatusUnset || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "span", spans[0].Attrs, attrs)
			counter := map[string]any{character.AttrOperation: "gain"}
			for k, v := range attrs {
				counter[k] = v
			}
			ops := onlyOne(t, rec, character.MetricOperations)
			if ops.Value != 1 || ops.Unit != "{operation}" || ops.Kind != telemetrytest.KindCounter {
				t.Errorf("operations = %+v", ops)
			}
			sameAttrs(t, "operations", ops.Attrs, counter)
			d := onlyOne(t, rec, character.MetricDuration)
			if d.Value != 0.25 || d.Unit != "s" {
				t.Errorf("duration = %+v", d)
			}
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "gain"})
			noLogsOrViolations(t, rec)
		})
	}
}
