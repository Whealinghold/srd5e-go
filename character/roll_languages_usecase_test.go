package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func rollLanguagesUseCase(t *testing.T, roller character.Roller, opts ...character.Option) character.RollLanguages {
	t.Helper()
	uc, err := character.NewRollLanguages(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

// Requirement: doc/specs/character-creation.md FR-050, FR-051, FR-052 and the table "Language choice and roll cases" - two 1d12 requests
// are made in order and mapped; different languages are RollLanguagesSucceeded (rolls 3 and 5; 1 and 12), the same language is
// RollLanguagesDuplicate without a third roll (rolls 3 and 4; 10 and 11), and the rolled languages are accepted by CreateCharacter.
// Path: per case.
func TestRollLanguages(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		rolls                   []int
		want                    []content.Language
		duplicate               bool
	}{
		{"rolls 3 and 5", "p. 20", "happy (boundary)", []int{3, 5}, []content.Language{content.LanguageDwarvish, content.LanguageElvish}, false},
		{"rolls 1 and 12", "p. 20", "happy (boundary)", []int{1, 12}, []content.Language{content.LanguageCommonSignLanguage, content.LanguageOrc}, false},
		{"rolls 3 and 4", "p. 20", "unhappy (rule interaction)", []int{3, 4}, []content.Language{content.LanguageDwarvish}, true},
		{"rolls 10 and 11", "p. 20", "unhappy (boundary)", []int{10, 11}, []content.Language{content.LanguageHalfling}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.rolls...)
			res, err := rollLanguagesUseCase(t, roller).Run(context.Background(), character.RollLanguagesCommand{})
			if err != nil {
				t.Fatal(err)
			}
			d12 := dice.MustExpression(1, dice.MustDie(12), 0)
			if reqs := roller.Requests(); !reflect.DeepEqual(reqs, []dice.Expression{d12, d12}) {
				t.Errorf("requests = %v [%s]", reqs, c.requirement)
			}
			switch r := res.(type) {
			case character.RollLanguagesSucceeded:
				if c.duplicate || !slices.Equal(r.Languages(), c.want) || !slices.Equal(r.Rolls(), c.rolls) {
					t.Errorf("= %v %v", r.Languages(), r.Rolls())
				}
				cmd := crCommand()
				cmd.Languages = r.Languages()
				crCreated(t, crMust(t, cmd))
				r.Languages()[0], r.Rolls()[0] = 0, 0
				if r.Languages()[0] == 0 || r.Rolls()[0] == 0 {
					t.Error("a returned slice reached the stored value")
				}
			case character.RollLanguagesDuplicate:
				if !c.duplicate || r.Language() != c.want[0] || !slices.Equal(r.Rolls(), c.rolls) {
					t.Errorf("= %v %v", r.Language(), r.Rolls())
				}
				r.Rolls()[0] = 0
				if r.Rolls()[0] == 0 {
					t.Error("a returned slice reached the stored value")
				}
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-099, FR-097 - a success is success/rolled, a duplicate is a failure outcome with result
// duplicate, no error status, no error.type and no log; a roller failure is recorded once with error.type; the Noop gives an equal result.
// Path: per case.
func TestRollLanguagesSignals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		roller  character.Roller
		attrs   map[string]any
		status  telemetry.Status
		wantErr bool
	}{
		{"rolled", dicetest.NewFixedRoller(3, 5), map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "rolled"}, telemetry.StatusUnset, false},
		{"duplicate", dicetest.NewFixedRoller(3, 4), map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "duplicate"}, telemetry.StatusUnset, false},
		{"roller failure", dicetest.NewFixedRoller(3), map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"}, telemetry.StatusError, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			res, err := rollLanguagesUseCase(t, c.roller, character.WithTelemetry(rec.Telemetry())).Run(context.Background(), character.RollLanguagesCommand{})
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v", err)
			}
			if !c.wantErr {
				plain, _ := rollLanguagesUseCase(t, dicetest.NewFixedRoller(3, 5)).Run(context.Background(), character.RollLanguagesCommand{})
				if _, isDup := res.(character.RollLanguagesDuplicate); !isDup && !reflect.DeepEqual(res, plain) {
					t.Error("Noop and a recorder must give equal results")
				}
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != character.SpanRollLanguages || spans[0].Status != c.status {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs("roll_languages", c.attrs))
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-008, FR-045 - a roller that answers another expression, and one that breaks its contract
// with a face outside 1 to 12, are failures and build no result.
// Path: unhappy (collaborator failure).
func TestRollLanguagesBrokenRollers(t *testing.T) {
	t.Parallel()
	wrong := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dicetest.NewMinimumRoller().Roll(dice.MustExpression(1, dice.D4(), 0))
	})
	outside := dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
		return dice.NewResult(e, []dice.DieRoll{dice.MustDieRoll(dice.MustDie(12), 12)})
	})
	if res, err := rollLanguagesUseCase(t, wrong).Run(context.Background(), character.RollLanguagesCommand{}); res != nil || !errors.Is(err, srderr.ErrContractViolation) {
		t.Errorf("another expression = %v, %v", res, err)
	}
	if res, err := rollLanguagesUseCase(t, outside).Run(context.Background(), character.RollLanguagesCommand{}); err != nil || res == nil {
		t.Errorf("a roll of 12 twice = %v, %v", res, err)
	}
}

// Requirement: doc/specs/character-creation.md section 6 - a nil roller and the zero use case are invalid-input errors and a cancelled
// context is ctx.Err(), none rolling.
// Path: unhappy (invalid input).
func TestRollLanguagesInvalidInput(t *testing.T) {
	t.Parallel()
	if _, err := character.NewRollLanguages(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller error = %v", err)
	}
	if _, err := (character.RollLanguages{}).Run(context.Background(), character.RollLanguagesCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	roller := dicetest.NewFixedRoller(3, 5)
	if _, err := rollLanguagesUseCase(t, roller).Run(ctx, character.RollLanguagesCommand{}); !errors.Is(err, context.Canceled) || len(roller.Requests()) != 0 {
		t.Errorf("cancelled error = %v", err)
	}
}

// Requirement: doc/specs/character-creation.md FR-050, FR-051 - two rolls give a duplicate only when the languages are the same: rolls 5 and 3
// (Elvish, then Dwarvish) are two languages, whichever is the greater.
// Path: unhappy (boundary).
func TestRollLanguagesDescendingRollsAreNotADuplicate(t *testing.T) {
	t.Parallel()
	res, err := rollLanguagesUseCase(t, dicetest.NewFixedRoller(5, 3)).Run(context.Background(), character.RollLanguagesCommand{})
	got, ok := res.(character.RollLanguagesSucceeded)
	if err != nil || !ok || !slices.Equal(got.Languages(), []content.Language{content.LanguageElvish, content.LanguageDwarvish}) {
		t.Errorf("= %#v, %v", res, err)
	}
}
