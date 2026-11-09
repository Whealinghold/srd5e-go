package character_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// mnkDef is the bundled Monk ClassDefinition (content/data/class_monk.json).
func mnkDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	src, err := data.NewBundledSource()
	if err != nil {
		tb.Fatal(err)
	}
	res, err := src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("monk")))
	if err != nil {
		tb.Fatal(err)
	}
	return res.(content.LookupFound).Entry.(content.ClassDefinition)
}

func mnkProgression(tb testing.TB) character.MonkProgression {
	tb.Helper()
	return must(character.NewMonkProgression(mnkDef(tb)))
}

// mnkStats are the stats of the Monk level n.
func mnkStats(tb testing.TB, n int) character.MonkLevelStats {
	tb.Helper()
	return mnkProgression(tb).StatsAt(character.MustMonkLevel(n))
}

// mnkChar is a character with the given class levels, enough Experience Points for level 20, Dexterity 16 (+3), Wisdom 16 (+3), every other score 10,
// and a stored maximum of 200.
func mnkChar(t *testing.T, entries ...character.ClassLevel) character.Character {
	t.Helper()
	return restored(t, 355000, 200, scoresOf(t, 10, 16, 10, 10, 16, 10), entries...)
}

// mnkMonk is a Monk of level n and nothing else.
func mnkMonk(t *testing.T, n int) character.Character {
	t.Helper()
	return mnkChar(t, cl("monk", n))
}

// mnkWith returns c holding the MonkState state.
func mnkWith(t *testing.T, c character.Character, state character.MonkState) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.MonkStateKey(), state))
}

// mnkStateOf reads the MonkState c stores.
func mnkStateOf(t *testing.T, c character.Character) character.MonkState {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.MonkStateKey())
	if err != nil || !ok {
		t.Fatalf("MonkState = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

// mnkState builds a MonkState from the expended Focus Points, the Uncanny Metabolism flag and the spent Wholeness of Body uses.
func mnkState(focus int, uncanny bool, wholeness int) character.MonkState {
	return character.MustMonkState(focus, uncanny, wholeness, character.QuiveringPalmVibrations{})
}

// mnkVibrations are vibrations on the target "orc-1" lasting days days.
func mnkVibrations(days int) character.QuiveringPalmVibrations {
	return must(character.NewQuiveringPalmVibrations(mnkTarget(), days))
}

func mnkTarget() character.QuiveringPalmTarget {
	return must(character.NewQuiveringPalmTarget("orc-1"))
}

// mod is the ability modifier n.
func mod(t *testing.T, n int) ability.Modifier {
	t.Helper()
	return modifierOf(t, n)
}

// mnkAbilities are Dexterity +dex, Wisdom +wis and the Proficiency Bonus of a character of total level level.
func mnkAbilities(t *testing.T, dex, wis, level int) character.MonkAbilities {
	t.Helper()
	return character.MonkAbilities{Dexterity: mod(t, dex), Wisdom: mod(t, wis), Proficiency: character.ProficiencyBonusFor(character.MustLevel(level))}
}

// mnkTelemetry is a recording bundle and the options that inject it with a clock stepping one second per read.
func mnkTelemetry() (*telemetrytest.Recorder, []character.Option) {
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(1000, 0), time.Second)
	return rec, []character.Option{character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)}
}

// mnkFailingRollers are the ways the dice port fails, each with the cause that must stay reachable.
func mnkFailingRollers() []struct {
	name   string
	roller character.Roller
	cause  error
} {
	boom := errors.New("roller down")
	return []struct {
		name   string
		roller character.Roller
		cause  error
	}{
		{"an error", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, boom }), boom},
		{"exhausted", dicetest.NewFixedRoller(), dicetest.ErrExhausted},
		{"a panic", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(boom) }), boom},
	}
}

// mnkWantPortFailure asserts a nil result and a *srderr.PortError for the dice port that keeps the cause.
func mnkWantPortFailure(t *testing.T, res any, err, cause error) {
	t.Helper()
	var pe *srderr.PortError
	if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, cause) {
		t.Errorf("= %v, %v; want a nil result and a PortError for dice with the cause", res, err)
	}
}
