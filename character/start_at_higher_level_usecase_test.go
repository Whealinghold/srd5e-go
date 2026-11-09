package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func startUseCase(t *testing.T, src character.ContentSource, roller character.Roller, opts ...character.Option) character.StartAtHigherLevel {
	t.Helper()
	uc, err := character.NewStartAtHigherLevel(append([]character.Option{character.WithSource(src), character.WithRoller(roller)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return uc
}

func recordedStart(t *testing.T, src character.ContentSource, roller character.Roller) (character.StartAtHigherLevel, *telemetrytest.Recorder) {
	t.Helper()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	return startUseCase(t, src, roller, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)), rec
}

func fixedStep(class, subclass string, inc character.AbilityScoreIncrease) character.StartingLevelStep {
	return character.StartingLevelStep{ClassID: content.ClassID(class), HitPointChoice: character.TakeFixedValue{}, SubclassID: content.SubclassID(subclass), AbilityScoreIncrease: inc}
}

func rolledStep(class, subclass string) character.StartingLevelStep {
	return character.StartingLevelStep{ClassID: content.ClassID(class), HitPointChoice: character.RollHitDie{}, SubclassID: content.SubclassID(subclass)}
}

// fighterSteps are the n Fighter steps to level n+1, rolled or fixed, with the Champion chosen on
// the level 3 step and incs keyed by the total level of the step.
func fighterSteps(n int, rolled bool, incs map[int]character.AbilityScoreIncrease) []character.StartingLevelStep {
	steps := make([]character.StartingLevelStep, n)
	for i := range steps {
		total, sub := i+2, ""
		if total == 3 {
			sub = "champion"
		}
		if rolled {
			steps[i] = rolledStep("fighter", sub)
		} else {
			steps[i] = fixedStep("fighter", sub, incs[total])
		}
		steps[i].AbilityScoreIncrease = incs[total]
	}
	return steps
}

func startCmd(t *testing.T, level, strength int, steps []character.StartingLevelStep) character.StartAtHigherLevelCommand {
	t.Helper()
	return character.StartAtHigherLevelCommand{
		CharacterID: character.MustIdentifier("pc-1"), ClassID: "fighter", SpeciesID: "human", BackgroundID: "soldier",
		AbilityScores: hlsScores(t, strength), StartingLevel: character.MustLevel(level), Steps: steps,
	}
}

func startOf(t *testing.T, uc character.StartAtHigherLevel, cmd character.StartAtHigherLevelCommand) character.StartAtHigherLevelResult {
	t.Helper()
	res, err := uc.Start(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	return res
}

func startedOf(t *testing.T, res character.StartAtHigherLevelResult) character.StartAtHigherLevelStarted {
	t.Helper()
	s, ok := res.(character.StartAtHigherLevelStarted)
	if !ok {
		t.Fatalf("result = %#v, want StartAtHigherLevelStarted", res)
	}
	return s
}

func rejectionOf(t *testing.T, res character.StartAtHigherLevelResult) character.HigherLevelStartRejection {
	t.Helper()
	r, ok := res.(character.StartAtHigherLevelRejected)
	if !ok {
		t.Fatalf("result = %#v, want StartAtHigherLevelRejected", res)
	}
	return r.Reason
}

func rolledDice(r *dicetest.FixedRoller) []int {
	var out []int
	for _, e := range r.Requests() {
		out = append(out, e.Die().Sides())
	}
	return out
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-065, FR-068, FR-074, FR-075, FR-076, FR-119, Table SH
// rows "level 3, fixed", "level 3, rolled, all minimum", "level 3, rolled, all maximum", "level 3,
// mixed", "accepted plan, rolled steps" - the Hit Point maximum follows the choices (fixed 8 a
// level; a roll of 1 clamps to 3 with Constitution +2; a top roll gives 12), the roller is asked for
// exactly one die of the class's size per rolled step in step order and never for a fixed step,
// the Experience Points are the minimum for the level, and the events start with Created.
// Path: per case.
func TestStartAtHigherLevelHitPointsAndRolls(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		steps    []character.StartingLevelStep
		level    int
		roller   func() *dicetest.FixedRoller
		wantMax  int
		wantDice []int
	}{
		{"level 3, fixed", fighterSteps(2, false, nil), 3, dicetest.NewMinimumRoller, 28, nil},
		{"level 3, rolled, all minimum", fighterSteps(2, true, nil), 3, dicetest.NewMinimumRoller, 18, []int{10, 10}},
		{"level 3, rolled, all maximum", fighterSteps(2, true, nil), 3, dicetest.NewMaximumRoller, 36, []int{10, 10}},
		{"level 3, mixed", []character.StartingLevelStep{rolledStep("fighter", ""), fixedStep("fighter", "champion", character.AbilityScoreIncrease{})}, 3,
			func() *dicetest.FixedRoller { return dicetest.NewFixedRoller(5) }, 12 + 7 + 8, []int{10}},
		{"accepted plan, rolled steps", fighterSteps(3, true, nil), 4, dicetest.NewMinimumRoller, 12 + 3*3, []int{10, 10, 10}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := c.roller()
			res := startOf(t, startUseCase(t, hlsSource(t, hlsFighter(t)), roller), startCmd(t, c.level, 15, c.steps))
			s := startedOf(t, res)
			if got := s.Character().HitPointMaximum().Value(); got != c.wantMax {
				t.Errorf("Hit Point maximum = %d, want %d [FR-075]", got, c.wantMax)
			}
			if got := rolledDice(roller); !reflect.DeepEqual(got, c.wantDice) {
				t.Errorf("rolled dice = %v, want %v [FR-074]", got, c.wantDice)
			}
			if s.Character().TotalLevel().Value() != c.level || s.Character().ExperiencePoints() != (character.HigherLevelStart{}).StartingExperience(character.MustLevel(c.level)) {
				t.Errorf("level %d, XP %d [FR-065]", s.Character().TotalLevel().Value(), s.Character().ExperiencePoints().Value())
			}
			events := s.Events()
			if _, ok := events[0].(character.Created); !ok || len(events) < c.level {
				t.Errorf("events = %#v, want Created first [FR-076]", events)
			}
			if len(s.ConsumedGrants()) != 0 {
				t.Errorf("ConsumedGrants = %v, want none", s.ConsumedGrants())
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-068, FR-074, Table SH row "multiclass second
// class" - rolled steps in different classes roll the die of their own class in step order, and a
// multiclass second class joins at level 1.
// Path: happy.
func TestStartAtHigherLevelRollsEachClassDie(t *testing.T) {
	t.Parallel()
	barbarian, druid := advBarbarian(t), advDruid(t)
	cmd := startCmd(t, 3, 13, []character.StartingLevelStep{rolledStep("druid", ""), rolledStep("barbarian", "")})
	cmd.ClassID = "barbarian"
	cmd.AbilityScores = scoresOf(t, 13, 10, 14, 10, 13, 10)
	roller := dicetest.NewMinimumRoller()
	s := startedOf(t, startOf(t, startUseCase(t, hlsSource(t, barbarian, druid), roller), cmd))
	if got := rolledDice(roller); !reflect.DeepEqual(got, []int{8, 12}) {
		t.Errorf("rolled dice = %v, want a d8 then a d12", got)
	}
	got := s.Character().ClassLevels()
	if got.LevelIn("barbarian") != 2 || got.LevelIn("druid") != 1 {
		t.Errorf("class levels = %v, want Barbarian 2 and Druid 1", got.Entries())
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-067 - a starting level of 1 with no steps gives
// the Character and the one Created event that New gives for the same arguments and 0 XP, and
// rolls nothing.
// Path: unhappy (empty and minimal).
func TestStartAtHigherLevelOne(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	roller := dicetest.NewMinimumRoller()
	s := startedOf(t, startOf(t, startUseCase(t, hlsSource(t, f), roller), startCmd(t, 1, 15, nil)))
	want := hlsStart(t, f, hlsScores(t, 15), 1)
	if !s.Character().Equal(want) || !reflect.DeepEqual(s.Events(), character.MustNew(character.NewParams{
		ID: character.MustIdentifier("pc-1"), Class: "fighter", HitPointDie: f.HitPointDie(), Species: "human", Background: "soldier", AbilityScores: hlsScores(t, 15),
	}).Events()) || len(roller.Requests()) != 0 {
		t.Errorf("result = %#v, want what New gives", s)
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-119, FR-068, Table SH row "StartingLevelStep
// conversion" - the use case gives the same character and events as the equivalent HigherLevelSteps
// through HigherLevelStart, after the Created event.
// Path: happy.
func TestStartAtHigherLevelMatchesTheDomainService(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	s := startedOf(t, startOf(t, startUseCase(t, hlsSource(t, f), dicetest.NewMinimumRoller()), startCmd(t, 3, 15, fighterSteps(2, false, nil))))
	b := builtOf(t, advanceOf(t, hlsStart(t, f, hlsScores(t, 15), 3), 3, classMap(f), fighterPlan(t, f, 2, true, nil)))
	if !s.Character().Equal(b.Character()) || !reflect.DeepEqual(s.Events()[1:], b.Events()) {
		t.Errorf("the use case and the domain service disagree: %#v vs %#v", s, b)
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-072, FR-076, FR-118, Table SH rows "ability
// increase at level 4", "later step sees earlier increase" - an applied increase is reported as a
// consumed grant, in step order, and the step's events follow its LevelGained.
// Path: happy.
func TestStartAtHigherLevelReportsConsumedGrants(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	incs := map[int]character.AbilityScoreIncrease{4: plusTwo(ability.Strength), 8: plusTwo(ability.Strength)}
	s := startedOf(t, startOf(t, startUseCase(t, hlsSource(t, f), dicetest.NewMinimumRoller()), startCmd(t, 8, 15, fighterSteps(7, false, incs))))
	entry := content.FeatureID("fighter_ability_score_improvement")
	want := []character.ConsumedFeatGrant{{Class: "fighter", ClassLevel: 4, Feature: entry}, {Class: "fighter", ClassLevel: 8, Feature: entry}}
	if got := s.ConsumedGrants(); !reflect.DeepEqual(got, want) || s.Character().AbilityScores().Score(ability.Strength).Value() != 19 {
		t.Errorf("ConsumedGrants = %v, Strength %d, want %v and 19", got, s.Character().AbilityScores().Score(ability.Strength).Value(), want)
	}
	s.ConsumedGrants()[0] = character.ConsumedFeatGrant{}
	s.Events()[0] = nil
	if s.ConsumedGrants()[0] == (character.ConsumedFeatGrant{}) || s.Events()[0] == nil {
		t.Error("the result must return copies")
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-066, FR-069, FR-071, FR-073, FR-074, FR-078, FR-106,
// FR-113, Table SH - each rule rejection is a Rejected result of the right variant, found before any die
// is rolled (the roller is asked for nothing), with the order of FR-106 between them.
// Path: per case.
func TestStartAtHigherLevelRejections(t *testing.T) {
	t.Parallel()
	f, druid := hlsFighter(t), advDruid(t)
	for _, c := range []struct {
		name     string
		cmd      func() character.StartAtHigherLevelCommand
		wantType string
		check    func(t *testing.T, r character.HigherLevelStartRejection)
	}{
		{"plan too short, with rolled steps", func() character.StartAtHigherLevelCommand {
			return startCmd(t, 3, 15, []character.StartingLevelStep{rolledStep("fighter", "")})
		}, "PlanLengthMismatch", func(t *testing.T, r character.HigherLevelStartRejection) {
			t.Helper()
			if r != (character.HigherLevelStartPlanLengthMismatch{ExpectedSteps: 2, ActualSteps: 1}) {
				t.Errorf("rejection = %#v", r)
			}
		}},
		{"plan too long", func() character.StartAtHigherLevelCommand {
			return startCmd(t, 2, 15, fighterSteps(2, false, nil))
		}, "PlanLengthMismatch", func(t *testing.T, r character.HigherLevelStartRejection) {
			t.Helper()
			if r != (character.HigherLevelStartPlanLengthMismatch{ExpectedSteps: 1, ActualSteps: 2}) {
				t.Errorf("rejection = %#v", r)
			}
		}},
		{"starting score above 20, rolled steps", func() character.StartAtHigherLevelCommand {
			return startCmd(t, 3, 21, fighterSteps(2, true, nil))
		}, "CreationInvalid", func(t *testing.T, r character.HigherLevelStartRejection) {
			t.Helper()
			p := r.(character.HigherLevelStartCreationInvalid).Problems()
			if len(p) != 1 || !reflect.DeepEqual(p[0], character.MustStartingScoreAboveMaximum([]ability.Ability{ability.Strength})) {
				t.Errorf("problems = %#v, want [StartingScoreAboveMaximum Strength]", p)
			}
			p[0] = nil
			if r.(character.HigherLevelStartCreationInvalid).Problems()[0] == nil {
				t.Error("Problems must return a copy")
			}
		}},
		{"mid-plan rejection after rolled steps", func() character.StartAtHigherLevelCommand {
			cmd := startCmd(t, 4, 15, []character.StartingLevelStep{rolledStep("fighter", ""), rolledStep("druid", ""), rolledStep("fighter", "")})
			cmd.AbilityScores = scoresOf(t, 15, 10, 14, 10, 12, 10)
			return cmd
		}, "LevelUpRejected", func(t *testing.T, r character.HigherLevelStartRejection) {
			t.Helper()
			if got := r.(character.HigherLevelStartLevelUpRejected); got.AtLevel.Value() != 3 {
				t.Errorf("rejection = %#v, want AtLevel 3", got)
			}
		}},
		{"subclass required, with rolled steps", func() character.StartAtHigherLevelCommand {
			return startCmd(t, 3, 15, []character.StartingLevelStep{rolledStep("fighter", ""), rolledStep("fighter", "")})
		}, "SubclassRequired", func(t *testing.T, r character.HigherLevelStartRejection) {
			t.Helper()
			if got := r.(character.HigherLevelStartSubclassRequired).Classes(); !reflect.DeepEqual(got, []content.ClassID{"fighter"}) {
				t.Errorf("classes = %v", got)
			}
		}},
		{"subclass below its level", func() character.StartAtHigherLevelCommand {
			return startCmd(t, 2, 15, []character.StartingLevelStep{rolledStep("fighter", "champion")})
		}, "SubclassRejected", func(t *testing.T, r character.HigherLevelStartRejection) {
			t.Helper()
			if got := r.(character.HigherLevelStartSubclassRejected); got.AtLevel.Value() != 2 || got.Rejection.Reason != character.SubclassBelowSubclassLevel {
				t.Errorf("rejection = %#v", got)
			}
		}},
		{"increase at a level with no grant", func() character.StartAtHigherLevelCommand {
			return startCmd(t, 2, 15, []character.StartingLevelStep{{ClassID: "fighter", HitPointChoice: character.RollHitDie{}, AbilityScoreIncrease: plusTwo(ability.Strength)}})
		}, "IncreaseNotGranted", func(t *testing.T, r character.HigherLevelStartRejection) {
			t.Helper()
			if r != (character.HigherLevelStartIncreaseNotGranted{AtLevel: character.MustLevel(2)}) {
				t.Errorf("rejection = %#v", r)
			}
		}},
		{"increase refused at level 8", func() character.StartAtHigherLevelCommand {
			return startCmd(t, 8, 17, fighterSteps(7, true, map[int]character.AbilityScoreIncrease{4: plusTwo(ability.Strength), 8: plusTwo(ability.Strength)}))
		}, "IncreaseRejected", func(t *testing.T, r character.HigherLevelStartRejection) {
			t.Helper()
			got := r.(character.HigherLevelStartIncreaseRejected)
			if _, ok := got.Rejection.(character.IncreaseExceedsMaximum); !ok || got.AtLevel.Value() != 8 {
				t.Errorf("rejection = %#v", got)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewMinimumRoller()
			res := startOf(t, startUseCase(t, hlsSource(t, f, druid), roller), c.cmd())
			c.check(t, rejectionOf(t, res))
			if n := len(roller.Requests()); n != 0 {
				t.Errorf("%d dice rolled for a rejected plan, want 0 [FR-074]", n)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-077, FR-106, Table SH rows "class missing", "class
// missing, with rolled steps", "order: plan length before content", "order: content before creation" - a
// class, the species or the background the source lacks is ContentMissing listing every missing entry
// once, ordered by kind then id, nothing rolled; the plan length is judged before any lookup and
// missing content before the starting scores.
// Path: per case.
func TestStartAtHigherLevelContentMissing(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	t.Run("a step class", func(t *testing.T) {
		t.Parallel()
		roller := dicetest.NewMinimumRoller()
		cmd := startCmd(t, 3, 15, []character.StartingLevelStep{rolledStep("fighter", ""), rolledStep("druid", "")})
		res := startOf(t, startUseCase(t, hlsSource(t, f), roller), cmd)
		m, ok := res.(character.StartAtHigherLevelContentMissing)
		if !ok || len(m.Missing()) != 1 || m.Missing()[0].ID().String() != "druid" || len(roller.Requests()) != 0 {
			t.Fatalf("result = %#v, rolls %d", res, len(roller.Requests()))
		}
		m.Missing()[0] = content.MissingContent{}
		if m.Missing()[0].ID().String() != "druid" {
			t.Error("Missing must return a copy")
		}
	})
	t.Run("several at once, deduplicated and ordered", func(t *testing.T) {
		t.Parallel()
		cmd := startCmd(t, 3, 15, []character.StartingLevelStep{fixedStep("druid", "", character.AbilityScoreIncrease{}), fixedStep("druid", "", character.AbilityScoreIncrease{})})
		cmd.ClassID = "fighter"
		res := startOf(t, startUseCase(t, contenttest.NewFakeSource(), dicetest.NewMinimumRoller()), cmd)
		m, ok := res.(character.StartAtHigherLevelContentMissing)
		if !ok {
			t.Fatalf("result = %#v", res)
		}
		var got [][2]string
		for _, x := range m.Missing() {
			got = append(got, [2]string{x.Kind().String(), x.ID().String()})
		}
		if len(got) != 4 {
			t.Fatalf("missing = %v, want the species, the background and the two classes once each", got)
		}
		kinds := m.Missing()
		for i := 1; i < len(kinds); i++ {
			if kinds[i-1].Kind() > kinds[i].Kind() || (kinds[i-1].Kind() == kinds[i].Kind() && kinds[i-1].ID().String() >= kinds[i].ID().String()) {
				t.Errorf("missing %v is not ordered by kind then id", got)
			}
		}
	})
	t.Run("plan length before any lookup", func(t *testing.T) {
		t.Parallel()
		src := contenttest.NewFakeSource()
		res := startOf(t, startUseCase(t, src, dicetest.NewMinimumRoller()), startCmd(t, 3, 15, []character.StartingLevelStep{fixedStep("druid", "", character.AbilityScoreIncrease{})}))
		if _, ok := rejectionOf(t, res).(character.HigherLevelStartPlanLengthMismatch); !ok || len(lookupKeys(src)) != 0 {
			t.Errorf("result = %#v, lookups %d, want a plan length rejection and no lookup", res, len(lookupKeys(src)))
		}
	})
	t.Run("content before creation", func(t *testing.T) {
		t.Parallel()
		src := contenttest.NewFakeSource(f, hlsBackground(t))
		res := startOf(t, startUseCase(t, src, dicetest.NewMinimumRoller()), startCmd(t, 3, 21, fighterSteps(2, false, nil)))
		m, ok := res.(character.StartAtHigherLevelContentMissing)
		if !ok || len(m.Missing()) != 1 || m.Missing()[0].ID().String() != "human" {
			t.Errorf("result = %#v, want ContentMissing([human])", res)
		}
	})
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-079 - a *srderr.PortError from the content source or
// the roller is returned as the error with a nil result and the cause reachable; a roller that answers
// another expression is a contract violation after the rolls already made; a value off the die is a
// roller error; no character is built.
// Path: per case.
func TestStartAtHigherLevelPortFailures(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	cause := errors.New("down")
	t.Run("content source", func(t *testing.T) {
		t.Parallel()
		got, err := startUseCase(t, hlsSource(t, f).FailLookup(cause), dicetest.NewMinimumRoller()).Start(context.Background(), startCmd(t, 3, 15, fighterSteps(2, false, nil)))
		var pe *srderr.PortError
		if got != nil || !errors.As(err, &pe) || pe.Port != content.Port || srderr.ErrorType(err) != "content_error" || !errors.Is(err, cause) {
			t.Errorf("= %v, %v", got, err)
		}
	})
	t.Run("roller fails after the first roll", func(t *testing.T) {
		t.Parallel()
		roller := dicetest.NewFixedRoller(4)
		got, err := startUseCase(t, hlsSource(t, f), roller).Start(context.Background(), startCmd(t, 3, 15, fighterSteps(2, true, nil)))
		var pe *srderr.PortError
		if got != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, dicetest.ErrExhausted) || len(roller.Requests()) != 1 {
			t.Errorf("= %v, %v, rolls %d", got, err, len(roller.Requests()))
		}
	})
	t.Run("a roll off the die", func(t *testing.T) {
		t.Parallel()
		for _, v := range []int{0, 11} {
			got, err := startUseCase(t, hlsSource(t, f), dicetest.NewFixedRoller(v)).Start(context.Background(), startCmd(t, 2, 15, fighterSteps(1, true, nil)))
			var pe *srderr.PortError
			if got != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, dicetest.ErrValueOutOfRange) {
				t.Errorf("roll %d: = %v, %v", v, got, err)
			}
		}
	})
	t.Run("a roller that answers another expression", func(t *testing.T) {
		t.Parallel()
		wrong := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
			return dicetest.NewMinimumRoller().Roll(dice.MustExpression(1, dice.D4(), 0))
		})
		got, err := startUseCase(t, hlsSource(t, f), wrong).Start(context.Background(), startCmd(t, 2, 15, fighterSteps(1, true, nil)))
		if got != nil || !errors.Is(err, srderr.ErrContractViolation) || srderr.ErrorType(err) != "dice_roller_error" {
			t.Errorf("= %v, %v", got, err)
		}
	})
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-080 - Start alters neither its command nor its
// steps, copies the steps, and gives an equal result for the same command, content and fixed roller.
// Path: unhappy (immutability, determinism).
func TestStartAtHigherLevelDoesNotAlterItsCommand(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	cmd := startCmd(t, 4, 15, fighterSteps(3, true, map[int]character.AbilityScoreIncrease{4: plusTwo(ability.Strength)}))
	before := startCmd(t, 4, 15, fighterSteps(3, true, map[int]character.AbilityScoreIncrease{4: plusTwo(ability.Strength)}))
	a := startOf(t, startUseCase(t, hlsSource(t, f), dicetest.NewFixedRoller(3, 6, 9)), cmd)
	b := startOf(t, startUseCase(t, hlsSource(t, f), dicetest.NewFixedRoller(3, 6, 9)), cmd)
	if !reflect.DeepEqual(cmd, before) || !reflect.DeepEqual(a, b) {
		t.Error("the command was altered or equal inputs gave different results")
	}
	cmd.Steps[0].ClassID = "druid"
	if startedOf(t, a).Character().ClassLevels().LevelIn("fighter") != 4 {
		t.Error("the result shares the command's steps")
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-123 - an invalid command, the zero use case and
// a cancelled context are errors before any lookup or signal; a missing source or roller fails the
// constructor.
// Path: per case.
func TestStartAtHigherLevelInvalidCommands(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	src := hlsSource(t, f)
	uc, rec := recordedStart(t, src, dicetest.NewMinimumRoller())
	good := func() character.StartAtHigherLevelCommand { return startCmd(t, 2, 15, fighterSteps(1, false, nil)) }
	for name, edit := range map[string]func(*character.StartAtHigherLevelCommand){
		"zero id":             func(c *character.StartAtHigherLevelCommand) { c.CharacterID = character.Identifier{} },
		"invalid class":       func(c *character.StartAtHigherLevelCommand) { c.ClassID = "" },
		"invalid species":     func(c *character.StartAtHigherLevelCommand) { c.SpeciesID = "" },
		"invalid background":  func(c *character.StartAtHigherLevelCommand) { c.BackgroundID = "" },
		"zero scores":         func(c *character.StartAtHigherLevelCommand) { c.AbilityScores = ability.Scores{} },
		"zero level":          func(c *character.StartAtHigherLevelCommand) { c.StartingLevel = character.Level{} },
		"nil choice":          func(c *character.StartAtHigherLevelCommand) { c.Steps[0].HitPointChoice = nil },
		"invalid step class":  func(c *character.StartAtHigherLevelCommand) { c.Steps[0].ClassID = "" },
		"invalid subclass id": func(c *character.StartAtHigherLevelCommand) { c.Steps[0].SubclassID = "A" },
	} {
		cmd := good()
		edit(&cmd)
		if res, err := uc.Start(context.Background(), cmd); res != nil || !isInvalid(err) {
			t.Errorf("%s: = %v, %v", name, res, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Start(ctx, good()); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if n := len(rec.Meter.Measurements()) + len(rec.Tracer.Spans()) + len(lookupKeys(src)); n != 0 {
		t.Errorf("%d signals or lookups before validation", n)
	}
	if _, err := (character.StartAtHigherLevel{}).Start(context.Background(), good()); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	if _, err := character.NewStartAtHigherLevel(character.WithSource(src)); !isInvalid(err) {
		t.Errorf("missing roller: %v", err)
	}
	if _, err := character.NewStartAtHigherLevel(character.WithRoller(dicetest.NewMinimumRoller())); !isInvalid(err) {
		t.Errorf("missing source: %v", err)
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-121, FR-122, section 7 - a started plan is
// success/started, a rejected one is failure/rejected with its rejection and no error status, missing
// content is failure/content_missing with error status and no error.type, a port failure carries error.type
// and no result and is recorded once; the steps histogram records the plan size with no attributes; the
// duration carries the operation only; nothing is logged; Noop and a recorder give equal results.
// Path: per case.
func TestStartAtHigherLevelSignals(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	for _, c := range []struct {
		name       string
		cmd        func() character.StartAtHigherLevelCommand
		source     func(*contenttest.FakeSource) *contenttest.FakeSource
		attrs      map[string]any
		spanStatus telemetry.Status
		loadSpan   bool
		loadStatus telemetry.Status
		errors     int
		wantErr    bool
	}{
		{name: "started", cmd: func() character.StartAtHigherLevelCommand { return startCmd(t, 3, 15, fighterSteps(2, false, nil)) },
			attrs: map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "started"}, loadSpan: true},
		{name: "rejected", cmd: func() character.StartAtHigherLevelCommand {
			return startCmd(t, 3, 15, []character.StartingLevelStep{fixedStep("fighter", "", character.AbilityScoreIncrease{}), fixedStep("fighter", "", character.AbilityScoreIncrease{})})
		}, attrs: map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "subclass_required"}, loadSpan: true},
		{name: "plan length mismatch", cmd: func() character.StartAtHigherLevelCommand { return startCmd(t, 3, 15, fighterSteps(1, false, nil)) },
			attrs: map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "plan_length_mismatch"}},
		{name: "creation invalid", cmd: func() character.StartAtHigherLevelCommand { return startCmd(t, 2, 21, fighterSteps(1, false, nil)) },
			attrs: map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "creation_invalid"}, loadSpan: true},
		{name: "content missing", cmd: func() character.StartAtHigherLevelCommand { return startCmd(t, 2, 15, fighterSteps(1, false, nil)) },
			source: func(*contenttest.FakeSource) *contenttest.FakeSource { return contenttest.NewFakeSource() },
			attrs:  map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "content_missing"}, spanStatus: telemetry.StatusError, loadSpan: true, loadStatus: telemetry.StatusError},
		{name: "content port failure", cmd: func() character.StartAtHigherLevelCommand { return startCmd(t, 2, 15, fighterSteps(1, false, nil)) },
			source: func(s *contenttest.FakeSource) *contenttest.FakeSource { return s.FailLookup(errors.New("down")) },
			attrs:  map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "content_error"}, spanStatus: telemetry.StatusError, loadSpan: true, loadStatus: telemetry.StatusError, errors: 1, wantErr: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			src := hlsSource(t, f)
			if c.source != nil {
				src = c.source(src)
			}
			uc, rec := recordedStart(t, src, dicetest.NewMinimumRoller())
			cmd := c.cmd()
			res, err := uc.Start(context.Background(), cmd)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v", err)
			}
			plain, _ := startUseCase(t, src, dicetest.NewMinimumRoller()).Start(context.Background(), cmd)
			if !c.wantErr && !reflect.DeepEqual(res, plain) {
				t.Error("Noop and a recorder must give equal results")
			}
			spans := rec.Tracer.Spans()
			wantSpans := 1
			if c.loadSpan {
				wantSpans = 2
			}
			if len(spans) != wantSpans || spans[0].Name != character.SpanAdvancementStartAtHigherLevel || spans[0].Status != c.spanStatus || len(spans[0].Errors) != c.errors {
				t.Fatalf("spans = %+v", spans)
			}
			if c.loadSpan && (spans[1].Name != character.SpanAdvancementLoadContent || spans[1].ParentID != spans[0].ID || spans[1].Status != c.loadStatus) {
				t.Errorf("load span = %+v", spans[1])
			}
			ops := onlyOne(t, rec, character.MetricOperations)
			sameAttrs(t, "operations", ops.Attrs, opAttrs("start_at_higher_level", c.attrs))
			d := onlyOne(t, rec, character.MetricDuration)
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "start_at_higher_level"})
			steps := onlyOne(t, rec, character.MetricStartSteps)
			sameAttrs(t, "steps", steps.Attrs, map[string]any{})
			if ops.Value != 1 || d.Value != 0.25 || d.Unit != "s" || steps.Value != float64(len(cmd.Steps)) || steps.Unit != "{step}" {
				t.Errorf("operations %+v, duration %+v, steps %+v", ops, d, steps)
			}
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-122 - a roller failure is recorded exactly once: error
// status, the recorded error, one operations increment with error.type "dice_roller_error" and no
// result attribute, and no log.
// Path: unhappy (telemetry on failure).
func TestStartAtHigherLevelRollerFailureIsRecordedOnce(t *testing.T) {
	t.Parallel()
	uc, rec := recordedStart(t, hlsSource(t, hlsFighter(t)), dicetest.NewFixedRoller())
	if _, err := uc.Start(context.Background(), startCmd(t, 2, 15, fighterSteps(1, true, nil))); err == nil {
		t.Fatal("want the roller failure")
	}
	spans := rec.Tracer.Spans()
	if len(spans) != 2 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || spans[1].Status != telemetry.StatusUnset {
		t.Fatalf("spans = %+v", spans)
	}
	sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs,
		opAttrs("start_at_higher_level", map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"}))
	noLogsOrViolations(t, rec)
}
