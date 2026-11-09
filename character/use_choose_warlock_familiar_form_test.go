package character_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/class-warlock.md FR-090 - the use case loads the monster of the form and stores it: an Imp is stored as a special form with a FamiliarFormChosen event, a CR 0 Beast as a normal form, a Beast of CR 1/4 offered as a normal form is refused; the command's Character is not altered.
// Path: per case (happy; unhappy: invalid input).
func TestChooseWarlockFamiliarFormStoresTheForm(t *testing.T) {
	t.Parallel()
	uc := must(character.NewChooseWarlockFamiliarForm(warMonsterSource(t)))
	c := warChar(t, 5, 16, "", warKnows("pact_of_the_chain"))
	for _, form := range []character.FamiliarForm{character.SpecialFamiliarForm("imp"), character.NormalFamiliarForm("cat")} {
		res, err := uc.Run(t.Context(), character.ChooseWarlockFamiliarFormCommand{Character: c, Form: form})
		got, ok := res.(character.ChooseWarlockFamiliarFormApplied)
		if err != nil || !ok {
			t.Fatalf("%v = %#v, %v", form, res, err)
		}
		pact, _, _ := character.ReadFeatureState(got.Character().FeatureStates(), character.WarlockPactStateKey())
		if stored, held := pact.Familiar().Form(); !held || stored != form {
			t.Errorf("stored form = %v, %v", stored, held)
		}
		if ev, isEv := got.Events()[0].(character.FamiliarFormChosen); !isEv || ev.Form() != form {
			t.Errorf("events = %v", got.Events())
		}
		got.Events()[0] = nil
		if got.Events()[0] == nil {
			t.Error("Events must return a copy")
		}
	}
	if _, held, _ := character.ReadFeatureState(c.FeatureStates(), character.WarlockPactStateKey()); held {
		t.Error("the command's Character was altered")
	}
	res, _ := uc.Run(t.Context(), character.ChooseWarlockFamiliarFormCommand{Character: c, Form: character.NormalFamiliarForm("wolf")})
	if rej, ok := res.(character.ChooseWarlockFamiliarFormRejected); !ok || rej.Rejection.Kind != character.WarlockFamiliarFormNotOffered || rej.Rejection.Monster != "wolf" {
		t.Errorf("CR 1/4 normal form = %#v", res)
	}
	if res, err := uc.Run(t.Context(), character.ChooseWarlockFamiliarFormCommand{Character: c}); res != nil || !srderrIsInvalid(err) {
		t.Errorf("zero form = %v, %v", res, err)
	}
}

// Requirement: doc/specs/class-warlock.md FR-042, FR-090, FR-141, FR-142 - a monster the source lacks is a ChooseWarlockFamiliarFormContentMissing naming the monster; a source that fails or answers with another entry is a *srderr.PortError recorded once.
// Path: per case (unhappy: missing content, collaborator failure).
func TestChooseWarlockFamiliarFormContentFailures(t *testing.T) {
	t.Parallel()
	c := warChar(t, 5, 16, "", warKnows("pact_of_the_chain"))
	warContentFailures(t, character.SpanChooseWarlockFamiliarForm, "familiar_form", []content.MissingContent{content.MonsterID("imp").Missing()},
		func(src character.ContentSource, opts ...character.Option) (any, error) {
			return must(character.NewChooseWarlockFamiliarForm(src, opts...)).Run(t.Context(), character.ChooseWarlockFamiliarFormCommand{Character: c, Form: character.SpecialFamiliarForm("imp")})
		}, func(res any) ([]content.MissingContent, bool) {
			m, ok := res.(character.ChooseWarlockFamiliarFormContentMissing)
			return m.Missing(), ok
		})
	if !srderrIsInvalid(errOf(character.NewChooseWarlockFamiliarForm(nil))) {
		t.Error("a nil source was accepted")
	}
	if _, err := (character.ChooseWarlockFamiliarForm{}).Run(t.Context(), character.ChooseWarlockFamiliarFormCommand{Character: c}); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
}

// Requirement: doc/specs/class-warlock.md FR-090, FR-148, FR-149 - a zero Character and a cancelled context are errors before any signal; a stored state of another type under the Warlock key and a zero form are errors recorded on the span only; the ContentMissing result returns a copy of the missing entries.
// Path: unhappy (invalid input, missing content).
func TestChooseWarlockFamiliarFormInvalidCommands(t *testing.T) {
	t.Parallel()
	c := warChar(t, 5, 16, "", warKnows("pact_of_the_chain"))
	form := character.SpecialFamiliarForm("imp")
	rec, opts := warTelemetry()
	uc := must(character.NewChooseWarlockFamiliarForm(warMonsterSource(t), opts...))
	if res, err := uc.Run(t.Context(), character.ChooseWarlockFamiliarFormCommand{Form: form}); res != nil || !srderrIsInvalid(err) {
		t.Errorf("zero Character = %v, %v", res, err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if res, err := uc.Run(cancelled, character.ChooseWarlockFamiliarFormCommand{Character: c, Form: form}); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v, %v", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 {
		t.Errorf("an invalid command left signals: %+v", rec.Tracer.Spans())
	}
	wrong := must(character.WithFeatureState(restored(t, 355000, 200, warScores(t, 16), cl("warlock", 5)), character.MustFeatureStateKey[character.BardicInspirationPool]("warlock"), character.BardicInspirationPool{}))
	if res, err := uc.Run(t.Context(), character.ChooseWarlockFamiliarFormCommand{Character: wrong, Form: form}); res != nil || !srderrIsInvalid(err) {
		t.Errorf("another type of state = %v, %v", res, err)
	}
	if res, err := uc.Run(t.Context(), character.ChooseWarlockFamiliarFormCommand{Character: c}); res != nil || !srderrIsInvalid(err) {
		t.Errorf("zero form = %v, %v", res, err)
	}
	spans := rec.Tracer.SpansNamed(character.SpanChooseWarlockFamiliarForm)
	if len(spans) != 2 || spans[0].Status != telemetry.StatusError || spans[1].Status != telemetry.StatusError ||
		len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)) != 0 {
		t.Errorf("an invalid state or form must be recorded on the span only: %+v", rec.Tracer.Spans())
	}
	res, err := must(character.NewChooseWarlockFamiliarForm(contenttest.NewFakeSource())).Run(t.Context(), character.ChooseWarlockFamiliarFormCommand{Character: c, Form: form})
	missing, ok := res.(character.ChooseWarlockFamiliarFormContentMissing)
	if err != nil || !ok || len(missing.Missing()) != 1 {
		t.Fatalf("missing = %#v, %v", res, err)
	}
	missing.Missing()[0] = content.MissingContent{}
	if missing.Missing()[0] == (content.MissingContent{}) {
		t.Error("Missing must return a copy")
	}
}
