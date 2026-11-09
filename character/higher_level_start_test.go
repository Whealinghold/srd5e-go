package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// fighterPlan builds the n Fighter steps that take a level 1 Fighter to level n+1, all with the
// fixed Hit Point value. The Champion is chosen on the level 3 step when champion is set, and
// incs maps a total level to the increase carried by the step that reaches it.
func fighterPlan(t *testing.T, f content.ClassDefinition, n int, champion bool, incs map[int]character.AbilityScoreIncrease) []character.HigherLevelStep {
	t.Helper()
	steps := make([]character.HigherLevelStep, n)
	for i := range steps {
		total, sub := i+2, ""
		if champion && total == 3 {
			sub = "champion"
		}
		steps[i] = hlsStep(t, f, classFixedGain(f), sub, incs[total])
	}
	return steps
}

func advanceOf(t *testing.T, start character.Character, level int, defs map[content.ClassID]content.ClassDefinition, steps []character.HigherLevelStep) character.HigherLevelStartOutcome {
	t.Helper()
	out, err := character.HigherLevelStart{}.Advance(start, character.MustLevel(level), defs, steps)
	if err != nil {
		t.Fatalf("Advance: %v", err)
	}
	return out
}

func builtOf(t *testing.T, out character.HigherLevelStartOutcome) character.HigherLevelStartBuilt {
	t.Helper()
	b, ok := out.(character.HigherLevelStartBuilt)
	if !ok {
		t.Fatalf("outcome = %#v, want HigherLevelStartBuilt", out)
	}
	return b
}

func refusalOf(t *testing.T, out character.HigherLevelStartOutcome) character.HigherLevelStartRejection {
	t.Helper()
	r, ok := out.(character.HigherLevelStartRefused)
	if !ok {
		t.Fatalf("outcome = %#v, want HigherLevelStartRefused", out)
	}
	return r.Reason
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-065, Table SL (SRD 5.2.1, "Starting at
// Higher Levels", p. 24) - the starting Experience Points are the minimum for the level.
// Path: per case.
func TestStartingExperienceTableSL(t *testing.T) {
	t.Parallel()
	for level, want := range map[int]int{1: 0, 2: 300, 3: 900, 5: 6500, 10: 64000, 17: 225000, 20: 355000} {
		if got := (character.HigherLevelStart{}).StartingExperience(character.MustLevel(level)).Value(); got != want {
			t.Errorf("StartingExperience(%d) = %d, want %d", level, got, want)
		}
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-066 - the plan holds the starting
// level minus 1 steps, else the rejection carries the expected and the actual count.
// Path: per case.
func TestCheckPlanLength(t *testing.T) {
	t.Parallel()
	rules := character.HigherLevelStart{}
	for _, c := range []struct {
		name         string
		level, steps int
		want         character.HigherLevelStartRejection
	}{
		{"level 1, no steps", 1, 0, nil},
		{"level 3, two steps", 3, 2, nil},
		{"level 20, nineteen steps", 20, 19, nil},
		{"level 3, one step", 3, 1, character.HigherLevelStartPlanLengthMismatch{ExpectedSteps: 2, ActualSteps: 1}},
		{"level 2, two steps", 2, 2, character.HigherLevelStartPlanLengthMismatch{ExpectedSteps: 1, ActualSteps: 2}},
		{"level 1, one step", 1, 1, character.HigherLevelStartPlanLengthMismatch{ExpectedSteps: 0, ActualSteps: 1}},
	} {
		if got := rules.CheckPlanLength(character.MustLevel(c.level), c.steps); got != c.want {
			t.Errorf("%s: CheckPlanLength = %#v, want %#v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-065, FR-067, FR-068, FR-076, Table SH
// rows "level 3, fixed", "highest level", "level 1, no steps" - a plan of fixed steps gives the
// class levels, total level and Hit Point maximum the same LevelUps give, the events in order,
// no consumed grant, and leaves the start unchanged.
// Path: happy.
func TestAdvanceFixedPlans(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	for _, c := range []struct {
		name      string
		level     int
		champion  bool
		wantMax   int
		wantLevel int
	}{
		{"level 3, fixed", 3, true, 28, 3},
		{"highest level", 20, true, 12 + 19*8, 20},
		{"level 2, no subclass owed yet", 2, false, 20, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			start := hlsStart(t, f, hlsScores(t, 15), c.level)
			snapshot := start
			b := builtOf(t, advanceOf(t, start, c.level, classMap(f), fighterPlan(t, f, c.level-1, c.champion, nil)))
			got := b.Character()
			if got.TotalLevel().Value() != c.wantLevel || got.ClassLevels().LevelIn("fighter") != c.wantLevel ||
				got.HitPointMaximum().Value() != c.wantMax || got.HitDicePool().Total() != c.wantLevel ||
				got.ExperiencePoints() != (character.HigherLevelStart{}).StartingExperience(character.MustLevel(c.level)) {
				t.Errorf("Character = level %d, max %d, XP %d, want level %d, max %d [FR-065, FR-068]",
					got.TotalLevel().Value(), got.HitPointMaximum().Value(), got.ExperiencePoints().Value(), c.wantLevel, c.wantMax)
			}
			if len(b.ConsumedGrants()) != 0 {
				t.Errorf("ConsumedGrants = %v, want none [FR-118]", b.ConsumedGrants())
			}
			if !start.Equal(snapshot) {
				t.Error("the start character was altered")
			}
		})
	}
	out := advanceOf(t, hlsStart(t, f, hlsScores(t, 15), 1), 1, classMap(f), nil)
	if b := builtOf(t, out); len(b.Events()) != 0 || b.Character().TotalLevel().Value() != 1 || b.Character().ExperiencePoints().Value() != 0 {
		t.Errorf("level 1 with no steps = %+v, want the unchanged level 1 character [FR-067]", b)
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-068, FR-076, FR-118, Table SH rows "level 3,
// fixed" and "no increase, nothing consumed" - the events are ordered LevelGained, SubclassChosen
// on the step that chooses, and the built result's slices are copies.
// Path: happy.
func TestAdvanceEventOrderAndCopies(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	b := builtOf(t, advanceOf(t, hlsStart(t, f, hlsScores(t, 15), 3), 3, classMap(f), fighterPlan(t, f, 2, true, nil)))
	events := b.Events()
	if len(events) != 3 {
		t.Fatalf("%d events, want 3: %#v", len(events), events)
	}
	if lg, ok := events[0].(character.LevelGained); !ok || lg.TotalLevel().Value() != 2 {
		t.Errorf("event 0 = %#v, want LevelGained to level 2", events[0])
	}
	if lg, ok := events[1].(character.LevelGained); !ok || lg.TotalLevel().Value() != 3 {
		t.Errorf("event 1 = %#v, want LevelGained to level 3", events[1])
	}
	if sc, ok := events[2].(character.SubclassChosen); !ok || sc.Subclass() != "champion" || sc.ClassLevel() != 3 {
		t.Errorf("event 2 = %#v, want SubclassChosen champion at class level 3", events[2])
	}
	events[0] = nil
	if b.Events()[0] == nil {
		t.Error("Events must return a copy")
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-070 - for every starting level 2 to 20, applying
// all steps with the starting Experience Points never meets LevelUpInsufficientExperience or
// LevelUpMaximumLevel.
// Path: unhappy (boundary).
func TestAdvanceNeverMeetsExperienceOrMaximumLevel(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	for level := 2; level <= 20; level++ {
		out := advanceOf(t, hlsStart(t, f, hlsScores(t, 15), level), level, classMap(f), fighterPlan(t, f, level-1, level >= 3, nil))
		if _, ok := out.(character.HigherLevelStartBuilt); !ok {
			t.Errorf("starting level %d = %#v, want the plan built", level, out)
		}
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-068, FR-069, Table SH rows "multiclass second
// class" and "multiclass prerequisite unmet" - a second class joins at level 1 with its Hit Die
// when the Primary Abilities are 13 or more; otherwise the step is rejected with the short
// abilities and the total level it would have reached, and no later step is applied.
// Path: per case.
func TestAdvanceMulticlass(t *testing.T) {
	t.Parallel()
	barbarian, druid := advBarbarian(t), advDruid(t)
	for _, c := range []struct {
		name      string
		strength  int
		wantShort []ability.Ability
	}{
		{"prerequisites met", 13, nil},
		{"strength 12", 12, []ability.Ability{ability.Strength}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			scores := scoresOf(t, c.strength, 10, 14, 10, 13, 10)
			start := hlsStart(t, barbarian, scores, 2)
			steps := []character.HigherLevelStep{hlsStep(t, druid, classFixedGain(druid), "", character.AbilityScoreIncrease{})}
			out := advanceOf(t, start, 2, classMap(barbarian, druid), steps)
			if c.wantShort == nil {
				got := builtOf(t, out).Character()
				if got.ClassLevels().LevelIn("barbarian") != 1 || got.ClassLevels().LevelIn("druid") != 1 || got.HitDicePool().Total() != 2 {
					t.Errorf("class levels = %v, want Barbarian 1 and Druid 1", got.ClassLevels().Entries())
				}
				return
			}
			r, ok := refusalOf(t, out).(character.HigherLevelStartLevelUpRejected)
			prereq, isPrereq := r.Rejection.(character.LevelUpPrerequisitesNotMet)
			if !ok || r.AtLevel.Value() != 2 || !isPrereq || !reflect.DeepEqual(prereq.ShortAbilities(), c.wantShort) {
				t.Errorf("refusal = %#v, want LevelUpRejected(AtLevel 2, PrerequisitesNotMet %v)", r, c.wantShort)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-069, FR-106, Table SH rows "mid-plan
// rejection after rolled steps" and "order: step before subclass required" - the first step LevelUp
// refuses stops the build at that step's total level, before the subclass owed is judged.
// Path: unhappy (state error).
func TestAdvanceStopsAtTheFirstRejectedStep(t *testing.T) {
	t.Parallel()
	f, druid := hlsFighter(t), advDruid(t)
	scores := scoresOf(t, 15, 10, 14, 10, 12, 10) // Wisdom 12: the Druid prerequisite is unmet
	for _, c := range []struct {
		name  string
		level int
		steps func() []character.HigherLevelStep
		at    int
	}{
		{"fighter, then an unmet druid, then fighter", 4, func() []character.HigherLevelStep {
			return []character.HigherLevelStep{hlsStep(t, f, classFixedGain(f), "", character.AbilityScoreIncrease{}),
				hlsStep(t, druid, classFixedGain(druid), "", character.AbilityScoreIncrease{}), hlsStep(t, f, classFixedGain(f), "", character.AbilityScoreIncrease{})}
		}, 3},
		{"step before subclass required", 3, func() []character.HigherLevelStep {
			return []character.HigherLevelStep{hlsStep(t, druid, classFixedGain(druid), "", character.AbilityScoreIncrease{}),
				hlsStep(t, f, classFixedGain(f), "", character.AbilityScoreIncrease{})}
		}, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out := advanceOf(t, hlsStart(t, f, scores, c.level), c.level, classMap(f, druid), c.steps())
			r, ok := refusalOf(t, out).(character.HigherLevelStartLevelUpRejected)
			if !ok || r.AtLevel.Value() != c.at {
				t.Errorf("refusal = %#v, want LevelUpRejected at level %d", r, c.at)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-071, FR-073, FR-106, Table SH rows "subclass below
// its level", "subclass missing at 3" and "subclass not required at 2" - a subclass chosen below the
// subclass level is refused with the reason and the level; a class at its subclass level without
// one is owed (listed once); at level 2 none is owed.
// Path: per case.
func TestAdvanceSubclass(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	start2, start3 := hlsStart(t, f, hlsScores(t, 15), 2), hlsStart(t, f, hlsScores(t, 15), 3)
	below := []character.HigherLevelStep{hlsStep(t, f, classFixedGain(f), "champion", character.AbilityScoreIncrease{})}
	r, ok := refusalOf(t, advanceOf(t, start2, 2, classMap(f), below)).(character.HigherLevelStartSubclassRejected)
	if !ok || r.AtLevel.Value() != 2 || r.Rejection.Reason != character.SubclassBelowSubclassLevel {
		t.Errorf("below the subclass level: %#v", r)
	}
	unknown := fighterPlan(t, f, 2, false, nil)
	unknown[1] = hlsStep(t, f, classFixedGain(f), "thief", character.AbilityScoreIncrease{})
	r, ok = refusalOf(t, advanceOf(t, start3, 3, classMap(f), unknown)).(character.HigherLevelStartSubclassRejected)
	if !ok || r.AtLevel.Value() != 3 || r.Rejection.Reason != character.SubclassUnknownSubclass {
		t.Errorf("unlisted subclass: %#v", r)
	}
	owed, ok := refusalOf(t, advanceOf(t, start3, 3, classMap(f), fighterPlan(t, f, 2, false, nil))).(character.HigherLevelStartSubclassRequired)
	if !ok || !reflect.DeepEqual(owed.Classes(), []content.ClassID{"fighter"}) {
		t.Errorf("missing subclass: %#v, want Required([fighter])", owed)
	}
	owed.Classes()[0] = "x"
	if owed.Classes()[0] != "fighter" {
		t.Error("Classes must return a copy")
	}
	builtOf(t, advanceOf(t, start2, 2, classMap(f), fighterPlan(t, f, 1, false, nil)))
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-073 - the classes owed a subclass are listed
// in the order they first appear in the ClassLevels.
// Path: unhappy (state error).
func TestAdvanceSubclassRequiredListsClassesInOrder(t *testing.T) {
	t.Parallel()
	f, cleric := hlsFighter(t), advCleric(t, "life_domain")
	scores := scoresOf(t, 15, 10, 14, 10, 13, 10)
	none := character.AbilityScoreIncrease{}
	fs, cs := hlsStep(t, f, classFixedGain(f), "", none), hlsStep(t, cleric, classFixedGain(cleric), "", none)
	// Fighter 3 and Cleric 3: both are owed a subclass, in the order the classes were taken.
	out := advanceOf(t, hlsStart(t, f, scores, 6), 6, classMap(f, cleric), []character.HigherLevelStep{fs, fs, cs, cs, cs})
	owed, ok := refusalOf(t, out).(character.HigherLevelStartSubclassRequired)
	if !ok || !reflect.DeepEqual(owed.Classes(), []content.ClassID{"fighter", "cleric"}) {
		t.Errorf("owed = %#v, want fighter then cleric", owed)
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-066, Table SH rows "plan too short" and "plan
// too long" - Advance refuses a plan of the wrong length before applying any step.
// Path: unhappy (invalid input).
func TestAdvanceRefusesAPlanOfTheWrongLength(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	for _, c := range []struct{ level, steps, wantExpected int }{{3, 1, 2}, {2, 2, 1}} {
		out := advanceOf(t, hlsStart(t, f, hlsScores(t, 15), c.level), c.level, classMap(f), fighterPlan(t, f, c.steps, true, nil))
		want := character.HigherLevelStartPlanLengthMismatch{ExpectedSteps: c.wantExpected, ActualSteps: c.steps}
		if got := refusalOf(t, out); got != want {
			t.Errorf("level %d with %d steps: %#v, want %#v", c.level, c.steps, got, want)
		}
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-072, FR-113, FR-118, Table SH rows "ability
// increase at level 4", "Constitution increase, retroactive Hit Points", "later step sees earlier
// increase", "repeated grant at level 6", "repeated grant at level 8 accepted" - an increase that
// matches a grant of the class level reached is applied after the level and subclass, a later step
// sees the earlier scores, a Constitution increase raises the Hit Point maximum retroactively, each
// applied increase is reported as one ConsumedFeatGrant naming the class level and the entry (a
// repeated grant names the level 4 entry), and no FeatTaken event is emitted.
// Path: per case.
func TestAdvanceAbilityScoreIncreases(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	entry := content.FeatureID("fighter_ability_score_improvement")
	epic := must(character.NewEpicBoonIncrease(ability.Strength))
	for _, c := range []struct {
		name         string
		level        int
		incs         map[int]character.AbilityScoreIncrease
		wantStrength int
		wantMax      int
		wantConsumed []character.ConsumedFeatGrant
		lastEvents   []string
	}{
		{"increase at level 4", 4, map[int]character.AbilityScoreIncrease{4: plusTwo(ability.Strength)}, 17, 12 + 3*8,
			[]character.ConsumedFeatGrant{{Class: "fighter", ClassLevel: 4, Feature: entry}}, []string{"AbilityScoreChanged"}},
		{"Constitution increase, retroactive Hit Points", 4, map[int]character.AbilityScoreIncrease{4: plusTwo(ability.Constitution)}, 15, 40,
			[]character.ConsumedFeatGrant{{Class: "fighter", ClassLevel: 4, Feature: entry}}, []string{"AbilityScoreChanged", "HitPointMaximumChanged"}},
		{"later step sees earlier increase", 8, map[int]character.AbilityScoreIncrease{4: plusTwo(ability.Strength), 8: plusTwo(ability.Strength)}, 19, 12 + 7*8,
			[]character.ConsumedFeatGrant{{Class: "fighter", ClassLevel: 4, Feature: entry}, {Class: "fighter", ClassLevel: 8, Feature: entry}}, []string{"AbilityScoreChanged"}},
		{"repeated grant at level 6", 6, map[int]character.AbilityScoreIncrease{6: plusTwo(ability.Strength)}, 17, 12 + 5*8,
			[]character.ConsumedFeatGrant{{Class: "fighter", ClassLevel: 6, Feature: entry}}, []string{"AbilityScoreChanged"}},
		{"repeated grant at level 8 accepted", 8, map[int]character.AbilityScoreIncrease{8: plusTwo(ability.Strength)}, 17, 12 + 7*8,
			[]character.ConsumedFeatGrant{{Class: "fighter", ClassLevel: 8, Feature: entry}}, []string{"AbilityScoreChanged"}},
		{"Ability Score Improvement at an Epic Boon level", 19, map[int]character.AbilityScoreIncrease{19: plusTwo(ability.Strength)}, 17, 12 + 18*8,
			[]character.ConsumedFeatGrant{{Class: "fighter", ClassLevel: 19, Feature: "fighter_epic_boon"}}, []string{"AbilityScoreChanged"}},
		{"Epic Boon over 20", 19, map[int]character.AbilityScoreIncrease{19: epic}, 16, 12 + 18*8,
			[]character.ConsumedFeatGrant{{Class: "fighter", ClassLevel: 19, Feature: "fighter_epic_boon"}}, []string{"AbilityScoreChanged"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			start := hlsStart(t, f, hlsScores(t, 15), c.level)
			b := builtOf(t, advanceOf(t, start, c.level, classMap(f), fighterPlan(t, f, c.level-1, true, c.incs)))
			if got := b.Character().AbilityScores().Score(ability.Strength).Value(); got != c.wantStrength {
				t.Errorf("Strength = %d, want %d", got, c.wantStrength)
			}
			if got := b.Character().HitPointMaximum().Value(); got != c.wantMax {
				t.Errorf("Hit Point maximum = %d, want %d", got, c.wantMax)
			}
			if got := b.ConsumedGrants(); !reflect.DeepEqual(got, c.wantConsumed) {
				t.Errorf("ConsumedGrants = %v, want %v [FR-118]", got, c.wantConsumed)
			}
			events := b.Events()
			var tail []string
			for _, e := range events[len(events)-len(c.lastEvents):] {
				tail = append(tail, reflect.TypeOf(e).Name())
			}
			if !reflect.DeepEqual(tail, c.lastEvents) {
				t.Errorf("last events = %v, want %v [FR-076]", tail, c.lastEvents)
			}
			for _, e := range events {
				if _, taken := e.(character.FeatTaken); taken {
					t.Error("a consumed grant must not emit FeatTaken [FR-118]")
				}
			}
			b.ConsumedGrants()[0] = character.ConsumedFeatGrant{}
			if b.ConsumedGrants()[0] == (character.ConsumedFeatGrant{}) {
				t.Error("ConsumedGrants must return a copy")
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-113, FR-106, Table SH rows "increase at a level
// with no grant", "increase between grants", "Epic Boon size at an ASI level", "subclass-feature
// row has no grant", "order: not granted before refused" - an increase at a level with no matching
// grant, or an Epic Boon over 20 at a General grant, is NotGranted at the total level of the step,
// and NotGranted is reported before the increaser's refusal.
// Path: per case.
func TestAdvanceIncreaseNotGranted(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	epic := must(character.NewEpicBoonIncrease(ability.Strength))
	for _, c := range []struct {
		name     string
		level    int
		strength int
		incs     map[int]character.AbilityScoreIncrease
		at       int
	}{
		{"increase at a level with no grant", 2, 15, map[int]character.AbilityScoreIncrease{2: plusTwo(ability.Strength)}, 2},
		{"increase between grants", 5, 15, map[int]character.AbilityScoreIncrease{5: plusTwo(ability.Strength)}, 5},
		{"Epic Boon size at an ASI level", 4, 15, map[int]character.AbilityScoreIncrease{4: epic}, 4},
		{"subclass-feature row has no grant", 7, 15, map[int]character.AbilityScoreIncrease{7: plusTwo(ability.Strength)}, 7},
		{"not granted before refused", 8, 20, map[int]character.AbilityScoreIncrease{5: plusTwo(ability.Strength)}, 5},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out := advanceOf(t, hlsStart(t, f, hlsScores(t, c.strength), c.level), c.level, classMap(f), fighterPlan(t, f, c.level-1, c.level >= 3, c.incs))
			if got := refusalOf(t, out); got != (character.HigherLevelStartIncreaseNotGranted{AtLevel: character.MustLevel(c.at)}) {
				t.Errorf("refusal = %#v, want IncreaseNotGranted at level %d", got, c.at)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-072, Table SH rows "increase refused at level 8"
// and "rejected plan reports no consumed grant" - an increase the increaser refuses stops the build
// with the abilities over the maximum and the maximum, and a refused plan reports no consumed grant.
// Path: unhappy (boundary).
func TestAdvanceIncreaseRefused(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	incs := map[int]character.AbilityScoreIncrease{4: plusTwo(ability.Strength), 8: plusTwo(ability.Strength)}
	out := advanceOf(t, hlsStart(t, f, hlsScores(t, 17), 8), 8, classMap(f), fighterPlan(t, f, 7, true, incs))
	r, ok := refusalOf(t, out).(character.HigherLevelStartIncreaseRejected)
	over, isOver := r.Rejection.(character.IncreaseExceedsMaximum)
	if !ok || r.AtLevel.Value() != 8 || !isOver || !reflect.DeepEqual(over.Abilities(), []ability.Ability{ability.Strength}) || over.MaximumScore() != 20 {
		t.Errorf("refusal = %#v, want IncreaseRejected(AtLevel 8, IncreaseExceedsMaximum [Strength], 20)", r)
	}
	if _, built := out.(character.HigherLevelStartBuilt); built {
		t.Error("a refused plan must not be built, so it reports no consumed grant")
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-123 - Advance returns an error
// matching srderr.ErrInvalidInput for a zero start, a zero level, a start above level 1, a zero step
// and a missing class definition, and never panics.
// Path: per case.
func TestAdvanceInvalidInput(t *testing.T) {
	t.Parallel()
	f, druid := hlsFighter(t), advDruid(t)
	start := hlsStart(t, f, hlsScores(t, 15), 2)
	step := hlsStep(t, f, classFixedGain(f), "", character.AbilityScoreIncrease{})
	leveled := builtOf(t, advanceOf(t, start, 2, classMap(f), []character.HigherLevelStep{step})).Character()
	for _, c := range []struct {
		name  string
		start character.Character
		level character.Level
		defs  map[content.ClassID]content.ClassDefinition
		steps []character.HigherLevelStep
	}{
		{"zero start", character.Character{}, character.MustLevel(2), classMap(f), []character.HigherLevelStep{step}},
		{"zero level", start, character.Level{}, classMap(f), nil},
		{"start above level 1", leveled, character.MustLevel(3), classMap(f), []character.HigherLevelStep{step, step}},
		{"zero step", start, character.MustLevel(2), classMap(f), []character.HigherLevelStep{{}}},
		{"no definition for a step class", start, character.MustLevel(2), classMap(f), []character.HigherLevelStep{hlsStep(t, druid, classFixedGain(druid), "", character.AbilityScoreIncrease{})}},
		{"no definition for the initial class", start, character.MustLevel(2), classMap(druid), []character.HigherLevelStep{step}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out, err := character.HigherLevelStart{}.Advance(c.start, c.level, c.defs, c.steps)
			if out != nil || !isInvalid(err) {
				t.Errorf("Advance = %#v, %v, want an invalid-input error", out, err)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-113 - only a General grant admits an Ability
// Score Improvement and only an Epic Boon grant admits an increase of up to 30: a row that grants an Origin
// or a Fighting Style feat admits neither, so the increase is NotGranted.
// Path: per case.
func TestAdvanceIncreaseNeedsAGeneralOrEpicBoonGrant(t *testing.T) {
	t.Parallel()
	for _, category := range []string{"Origin", "Fighting Style"} {
		t.Run(category, func(t *testing.T) {
			t.Parallel()
			f := advClass(t, advSpec{id: "fighter", die: "D10", kind: "anyOf", abilities: []string{"strength", "dexterity"}, subclass: []string{"champion"},
				replace: [][2]string{{`"category": "General"`, `"category": "` + category + `"`}}})
			incs := map[int]character.AbilityScoreIncrease{4: plusTwo(ability.Strength)}
			out := advanceOf(t, hlsStart(t, f, hlsScores(t, 15), 4), 4, classMap(f), fighterPlan(t, f, 3, true, incs))
			if got := refusalOf(t, out); got != (character.HigherLevelStartIncreaseNotGranted{AtLevel: character.MustLevel(4)}) {
				t.Errorf("refusal = %#v, want IncreaseNotGranted at level 4 for a %s grant", got, category)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-068, FR-123 - a step whose Hit Point gain does not fit
// its class, and a state of another type under a class's subclass key, are invalid-input errors, not rejections.
// Path: per case.
func TestAdvanceReturnsTheErrorsOfTheRulesItComposes(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	start := hlsStart(t, f, hlsScores(t, 15), 3)
	badState := must(character.WithFeatureState(start, character.MustFeatureStateKey[otherState]("subclass_fighter"), otherState{}))
	d6Gain := character.MustRolledHitPointGain(3, content.HitDieD6)
	for _, c := range []struct {
		name  string
		start character.Character
		steps []character.HigherLevelStep
	}{
		{"a rolled gain of another die", start, []character.HigherLevelStep{hlsStep(t, f, d6Gain, "", character.AbilityScoreIncrease{}), hlsStep(t, f, classFixedGain(f), "champion", character.AbilityScoreIncrease{})}},
		{"a stored state of another type, at the subclass choice", badState, fighterPlan(t, f, 2, true, nil)},
		{"a stored state of another type, at the subclass owed check", badState, fighterPlan(t, f, 2, false, nil)},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out, err := character.HigherLevelStart{}.Advance(c.start, character.MustLevel(3), classMap(f), c.steps)
			if out != nil || !isInvalid(err) {
				t.Errorf("Advance = %#v, %v, want an invalid-input error", out, err)
			}
		})
	}
}
