package character_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/character-aggregate.md FR-068, FR-072, FR-088 - the use
// case returns Succeeded for a change and for a no-op (success, changed) and
// Rejected for a refusal (failure, rejected, the rejection string, span status
// not error); the default cap is 20; the command is unchanged.
// Path: per case.
func TestChangeAbilityScoreUseCase(t *testing.T) {
	t.Parallel()
	ch := restored(t, 400000, 76, scoresOf(t, 19, 10, 17, 10, 10, 10), cl("fighter", 8))
	for _, c := range []struct {
		name      string
		a         ability.Ability
		to, limit int
		rejection string // "" for a success
		events    int
	}{
		{"change", ability.Strength, 20, 0, "", 1}, {"constitution rise", ability.Constitution, 18, 0, "", 2},
		{"no-op", ability.Strength, 19, 0, "", 0},
		{"over the default cap", ability.Strength, 21, 0, "exceeds_maximum_score", 0},
		{"raised cap", ability.Strength, 22, 24, "", 1},
		{"constitution decrease", ability.Constitution, 14, 0, "constitution_modifier_decrease", 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			uc := character.NewChangeAbilityScore(character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now))
			cmd := character.ChangeAbilityScoreCommand{Character: ch, Ability: c.a, Score: ability.MustScore(c.to), MaximumScore: c.limit}
			res, err := uc.Run(context.Background(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			if !cmd.Character.Equal(ch) {
				t.Error("FR-087: the command must be unchanged")
			}
			want := opAttrs("change_ability_score", map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed"})
			if c.rejection == "" {
				ok, isOK := res.(character.ChangeAbilityScoreSucceeded)
				if !isOK || len(ok.Events()) != c.events || ok.Character().AbilityScores().Score(c.a).Value() != c.to {
					t.Fatalf("result = %#v", res)
				}
			} else {
				if _, isRej := res.(character.ChangeAbilityScoreRejected); !isRej {
					t.Fatalf("result = %#v", res)
				}
				want = opAttrs("change_ability_score", map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: c.rejection})
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanChangeAbilityScore || spans[0].Status != telemetry.StatusUnset || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, want)
			sameAttrs(t, "duration", onlyOne(t, rec, character.MetricDuration).Attrs, map[string]any{character.AttrOperation: "change_ability_score"})
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/character-aggregate.md FR-059, FR-074 - an invalid
// command (cap outside 20 to 30, zero character or score) is a validation error
// before any signal; a cancelled context is returned unwrapped.
// Path: per case.
func TestChangeAbilityScoreInvalid(t *testing.T) {
	t.Parallel()
	ch := restored(t, 0, 12, flatScores(t, 10), cl("fighter", 1))
	rec := telemetrytest.NewRecorder()
	uc := character.NewChangeAbilityScore(character.WithTelemetry(rec.Telemetry()))
	for name, cmd := range map[string]character.ChangeAbilityScoreCommand{
		"cap 19":         {Character: ch, Ability: ability.Strength, Score: ability.MustScore(12), MaximumScore: 19},
		"cap 31":         {Character: ch, Ability: ability.Strength, Score: ability.MustScore(12), MaximumScore: 31},
		"zero character": {Ability: ability.Strength, Score: ability.MustScore(12)},
		"zero score":     {Character: ch, Ability: ability.Strength},
	} {
		if _, err := uc.Run(context.Background(), cmd); !isInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := (character.ChangeAbilityScore{}).Run(context.Background(), character.ChangeAbilityScoreCommand{Character: ch, Ability: ability.Strength, Score: ability.MustScore(12)}); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, character.ChangeAbilityScoreCommand{Character: ch, Ability: ability.Strength, Score: ability.MustScore(12)}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if n := len(rec.Meter.Measurements()) + len(rec.Tracer.Spans()); n != 0 {
		t.Errorf("%d signals before validation", n)
	}
}
