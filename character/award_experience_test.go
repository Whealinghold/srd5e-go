package character_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/character-aggregate.md FR-020, FR-068, FR-069, FR-070,
// FR-087, FR-103 - the use case returns only AwardExperienceSucceeded with no
// port, leaves the command alone, behaves the same with Noop and a recorder, and
// emits one span, one operations increment and one duration with the operation
// attribute only.
// Path: happy.
func TestAwardExperienceUseCase(t *testing.T) {
	t.Parallel()
	ch := restored(t, 300, 12, flatScores(t, 10), cl("fighter", 1))
	cmd := character.AwardExperienceCommand{Character: ch, Amount: character.MustExperienceAmount(50)}
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	res, err := character.NewAwardExperience(character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).Run(context.Background(), cmd)
	plain, err2 := character.NewAwardExperience().Run(context.Background(), cmd)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	ok := res.(character.AwardExperienceSucceeded)
	if ok.Character().ExperiencePoints().Value() != 350 || len(ok.Events()) != 1 || !ok.Character().Equal(plain.(character.AwardExperienceSucceeded).Character()) {
		t.Errorf("result = %#v", res)
	}
	if !cmd.Character.Equal(ch) {
		t.Error("FR-087: the command must be unchanged")
	}
	spans := rec.Tracer.Spans()
	if len(spans) != 1 || spans[0].Name != character.SpanAwardExperience || !spans[0].Ended || spans[0].Status != telemetry.StatusUnset {
		t.Fatalf("spans = %+v", spans)
	}
	sameAttrs(t, "span", spans[0].Attrs, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed"})
	sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs("award_experience", map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed"}))
	d := onlyOne(t, rec, character.MetricDuration)
	if d.Value != 0.25 {
		t.Errorf("duration = %v", d.Value)
	}
	sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "award_experience"})
	noLogsOrViolations(t, rec)
}

// Requirement: doc/specs/character-aggregate.md FR-022, FR-059 - the zero use
// case, character or amount is a validation error before any signal; a cancelled
// context is returned unwrapped.
// Path: per case.
func TestAwardExperienceInvalid(t *testing.T) {
	t.Parallel()
	ch := restored(t, 0, 12, flatScores(t, 10), cl("fighter", 1))
	uc := character.NewAwardExperience()
	for name, cmd := range map[string]character.AwardExperienceCommand{
		"zero character": {Amount: character.MustExperienceAmount(1)},
		"zero amount":    {Character: ch},
	} {
		if _, err := uc.Run(context.Background(), cmd); !isInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := (character.AwardExperience{}).Run(context.Background(), character.AwardExperienceCommand{Character: ch, Amount: character.MustExperienceAmount(1)}); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, character.AwardExperienceCommand{Character: ch, Amount: character.MustExperienceAmount(1)}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
}
