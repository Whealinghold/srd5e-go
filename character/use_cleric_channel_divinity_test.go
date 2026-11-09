package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func useChannel(t *testing.T, roller character.Roller, opts ...character.Option) character.UseClericChannelDivinity {
	t.Helper()
	return must(character.NewUseClericChannelDivinity(clcProgression(t), roller, opts...))
}

func dcOf(t *testing.T, n int) character.SpellSaveDC {
	t.Helper()
	return character.Stats(character.CasterCleric, modifierOf(t, n-10), character.ProficiencyBonusFor(character.MustLevel(1))).SaveDC()
}

var visible = character.DivineSparkTarget{Key: "ally", Visible: true, DistanceFeet: 20}

func sparkCmd(c character.Character, mode character.SparkMode, subclass content.SubclassID) character.UseClericChannelDivinityCommand {
	return character.UseClericChannelDivinityCommand{Character: c, Subclass: subclass, Effect: character.DivineSparkEffect{Target: visible, Mode: mode, DamageType: content.DamageNecrotic}}
}

func runChannel(t *testing.T, roller *dicetest.FixedRoller, cmd character.UseClericChannelDivinityCommand) character.UseClericChannelDivinityResult {
	t.Helper()
	res, err := useChannel(t, roller).Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res
}

// Requirement: doc/specs/class-cleric.md FR-039, FR-045, FR-048 to FR-055, FR-117, FR-119, table 4.7.1 (SRD 5.2.1, "Level 2: Channel Divinity" (Divine Spark), p. 37) - Divine Spark rolls exactly the number of d8s of
// the Cleric level in one call, adds the Wisdom modifier, expends one use and returns the healing or the Constitution save at the spell save DC with the damage on a failed and a successful save, and one
// ClericChannelDivinityUsed event; a level 3 Cleric / level 5 Fighter rolls one die; the input Character is unchanged.
// Path: per case (the level bands, healing and damage): happy / unhappy (boundary).
func TestUseClericChannelDivinityDivineSpark(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name       string
		char       character.Character
		mode       character.SparkMode
		rolls      []int
		wis        int
		heal, f, p int
		remaining  int
	}{
		{"level 2 healing", clcCleric(t, 2), character.SparkHealing, []int{5}, 3, 8, 0, 0, 1},
		{"level 2 damage", clcCleric(t, 2), character.SparkDamage, []int{5}, 3, 0, 8, 4, 1},
		{"level 7 two dice", clcCleric(t, 7), character.SparkDamage, []int{3, 8}, 4, 0, 15, 7, 2},
		{"level 13 three dice", clcCleric(t, 13), character.SparkHealing, []int{8, 8, 8}, 5, 29, 0, 0, 2},
		{"level 18 four dice", clcCleric(t, 18), character.SparkDamage, []int{1, 1, 1, 1}, 5, 0, 9, 4, 3},
		{"Fighter 5 / Cleric 3 rolls one die", clcChar(t, cl("fighter", 5), cl("cleric", 3)), character.SparkHealing, []int{4}, 1, 5, 0, 0, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := sparkCmd(c.char, c.mode, "")
			cmd.Wisdom, cmd.SpellSaveDC = modifierOf(t, c.wis), dcOf(t, 14)
			roller := dicetest.NewFixedRoller(c.rolls...)
			r, ok := runChannel(t, roller, cmd).(character.UseClericChannelDivinityApplied)
			if !ok {
				t.Fatal("not applied")
			}
			out := r.Outcome().(character.DivineSparkOutcome)
			save, dc, hasSave := out.SavingThrow()
			if out.Resolution().HitPointsToRestore() != c.heal || out.Resolution().DamageOnFailedSave() != c.f || out.Resolution().DamageOnSuccessfulSave() != c.p || out.Target() != "ally" ||
				hasSave != (c.mode == character.SparkDamage) || save != ability.Constitution || dc != d20.DifficultyClass(14) {
				t.Errorf("outcome = %+v, save %v dc %v %v", out.Resolution(), save, dc, hasSave)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(len(c.rolls), dice.D8(), 0) {
				t.Errorf("requests = %v, want one %dd8", reqs, len(c.rolls))
			}
			if got := clcResourcesOf(t, r.Character()); got.ChannelDivinityRemaining() != c.remaining || got != r.Resources() {
				t.Errorf("resources = %+v, want %d remaining", got, c.remaining)
			}
			ev, ok := r.Events()[0].(character.ClericChannelDivinityUsed)
			if len(r.Events()) != 1 || !ok || ev.Effect() != "divine_spark" || ev.Remaining() != c.remaining || ev.CharacterID() != c.char.ID() {
				t.Errorf("events = %#v", r.Events())
			}
			if _, held, _ := character.ReadFeatureState(c.char.FeatureStates(), character.ClericResourcesKey()); held {
				t.Error("Run altered the input")
			}
			r.Events()[0] = nil
			if r.Events()[0] == nil {
				t.Error("Events returned a shared slice")
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-055, FR-114, FR-115, table 4.12.4 SH-04, SH-05, SH-10 to SH-12 and CF-08 (SRD 5.2.1, "Level 17: Supreme Healing", p. 40) - a Life Domain Cleric of level 17 heals with a Divine Spark whose d8s count
// as 8 each and calls the roller zero times; in damage mode, with another subclass or at level 16 the dice are rolled.
// Path: per case (SH rows): happy / unhappy (rule interaction).
func TestUseClericChannelDivinitySupremeHealing(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		level    int
		mode     character.SparkMode
		subclass content.SubclassID
		rolled   bool
		healing  int
	}{
		{"SH-04 / SH-11 / CF-08 healing at 17", 17, character.SparkHealing, "life_domain", false, 27},
		{"SH-06 healing at 20", 20, character.SparkHealing, "life_domain", false, 35},
		{"SH-05 / SH-12 damage at 17", 17, character.SparkDamage, "life_domain", true, 0},
		{"SH-10 another subclass", 17, character.SparkHealing, "other", true, 6},
		{"no subclass", 17, character.SparkHealing, "", true, 6},
		{"level 16", 16, character.SparkHealing, "life_domain", true, 6},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := sparkCmd(clcCleric(t, c.level), c.mode, c.subclass)
			cmd.Wisdom = modifierOf(t, 3)
			diceCount := 3
			if c.level == 20 {
				diceCount = 4
			}
			roller := dicetest.NewFixedRoller(slices.Repeat([]int{1}, diceCount)...)
			var r character.UseClericChannelDivinityApplied
			if c.rolled {
				r = runChannel(t, roller, cmd).(character.UseClericChannelDivinityApplied)
			} else {
				res, err := useChannel(t, dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("the roller must not be called") })).Run(context.Background(), cmd)
				if err != nil {
					t.Fatal(err)
				}
				r = res.(character.UseClericChannelDivinityApplied)
			}
			got := r.Outcome().(character.DivineSparkOutcome).Resolution()
			if c.mode == character.SparkHealing && got.HitPointsToRestore() != c.healing {
				t.Errorf("healing = %d, want %d", got.HitPointsToRestore(), c.healing)
			}
			if c.mode == character.SparkDamage && len(roller.Requests()) != 1 {
				t.Errorf("damage mode requests = %v, want one roll", roller.Requests())
			}
		})
	}
}

func undead(key string, d int) character.UndeadTarget {
	return character.UndeadTarget{Key: key, Type: content.CreatureTypeUndead, DistanceFeet: d}
}

// Requirement: doc/specs/class-cleric.md FR-045, FR-058 to FR-067, FR-119, FR-131, table 4.8.1 SU-09 to SU-12 (SRD 5.2.1, "Level 2: Channel Divinity" (Turn Undead), "Level 5: Sear Undead", p. 37) - Turn Undead reports the
// targets, the Wisdom save at the spell save DC, Frightened and Incapacitated for 1 minute, the movement away and the early endings; with SearUndeadUse at level 5 it rolls the d8s once whatever the number of targets (none
// included) and reports the Radiant total; with skip it rolls nothing and reports no Sear Undead data, at any level.
// Path: per case (SU-09 to SU-12, empty list, 30 feet): happy / unhappy (boundary, empty and minimal, rule interaction).
func TestUseClericChannelDivinityTurnUndead(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		level   int
		targets []character.UndeadTarget
		sear    character.SearUndeadChoice
		rolls   []int
		total   int
	}{
		{"SU-09 skip at level 5", 5, []character.UndeadTarget{undead("z", 10)}, character.SearUndeadSkip, nil, 0},
		{"SU-11 use with no targets", 5, nil, character.SearUndeadUse, []int{2, 4, 6}, 12},
		{"SU-12 use with three Undead", 5, []character.UndeadTarget{undead("a", 10), undead("b", 30), undead("c", 0)}, character.SearUndeadUse, []int{2, 4, 6}, 12},
		{"empty list and skip", 2, nil, character.SearUndeadSkip, nil, 0},
		{"skip at level 4", 4, []character.UndeadTarget{undead("a", 30)}, character.SearUndeadSkip, nil, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.rolls...)
			cmd := character.UseClericChannelDivinityCommand{Character: clcCleric(t, c.level), SpellSaveDC: dcOf(t, 15), Wisdom: modifierOf(t, 3),
				Effect: character.TurnUndeadEffect{Targets: c.targets, SearUndead: c.sear}}
			r, ok := runChannel(t, roller, cmd).(character.UseClericChannelDivinityApplied)
			if !ok {
				t.Fatal("not applied")
			}
			out := r.Outcome().(character.TurnUndeadOutcome)
			save, dc := out.SavingThrow()
			sear, hasSear := out.SearUndead()
			if len(out.Targets()) != len(c.targets) || save != ability.Wisdom || dc != 15 || out.DurationMinutes() != 1 || !out.MovesAwayFromCleric() ||
				!slices.Equal(out.ConditionsOnFailedSave(), []content.ConditionKind{content.ConditionFrightened, content.ConditionIncapacitated}) || len(out.EarlyEndings()) != 3 {
				t.Errorf("outcome = %+v", out)
			}
			if hasSear != (c.rolls != nil) || sear.Total() != c.total || (hasSear && sear.EndsEffect()) {
				t.Errorf("Sear Undead = %+v, %v; want total %d", sear, hasSear, c.total)
			}
			wantReqs := 0
			if c.rolls != nil {
				wantReqs = 1
			}
			if reqs := roller.Requests(); len(reqs) != wantReqs {
				t.Errorf("requests = %v, want %d", reqs, wantReqs)
			}
			if got := clcResourcesOf(t, r.Character()); got.ChannelDivinityRemaining() != clcProgression(t).StatsAt(character.MustClericLevel(c.level)).ChannelDivinity()-1 {
				t.Errorf("resources = %+v", got)
			}
			targets := out.Targets()
			if len(targets) > 0 {
				targets[0] = "changed"
				if out.Targets()[0] == "changed" {
					t.Error("Targets returned a shared slice")
				}
			}
		})
	}
}

func lifeTarget(key string, cur, max, dist, alloc int) character.PreserveLifeTarget {
	return character.PreserveLifeTarget{Key: key, CurrentHitPoints: cur, HitPointMaximum: max, DistanceFeet: dist, Allocation: alloc}
}

// Requirement: doc/specs/class-cleric.md FR-105 to FR-111, FR-117, table 4.12.3 PL-01, PL-09, PL-11, PL-13, PL-16 (SRD 5.2.1, "Level 3: Preserve Life", p. 40) - Preserve Life expends one use and reports the pool (five times
// the Cleric level, so 15 for a Cleric 3 / Fighter 5) and the allocation to each Bloodied target within 30 feet, including no target, the Cleric itself and a total below the pool; it rolls no dice.
// Path: per case (PL rows): happy / unhappy (boundary, empty and minimal).
func TestUseClericChannelDivinityPreserveLife(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		char    character.Character
		targets []character.PreserveLifeTarget
		pool    int
	}{
		{"PL-01", clcCleric(t, 3), []character.PreserveLifeTarget{lifeTarget("a", 4, 20, 10, 6)}, 15},
		{"PL-09", clcCleric(t, 3), []character.PreserveLifeTarget{lifeTarget("a", 0, 40, 10, 8), lifeTarget("b", 0, 40, 10, 7)}, 15},
		{"PL-11", clcCleric(t, 3), []character.PreserveLifeTarget{lifeTarget("a", 0, 40, 10, 14)}, 15},
		{"PL-13 the Cleric itself", clcCleric(t, 3), []character.PreserveLifeTarget{lifeTarget("cleric", 2, 30, 0, 13)}, 15},
		{"PL-16 no target", clcCleric(t, 3), nil, 15},
		{"Cleric 3 / Fighter 5 has a pool of 15", clcChar(t, cl("fighter", 5), cl("cleric", 3)), []character.PreserveLifeTarget{lifeTarget("a", 4, 20, 10, 6)}, 15},
		{"level 20", clcCleric(t, 20), []character.PreserveLifeTarget{lifeTarget("a", 0, 80, 30, 20)}, 100},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller()
			r, ok := runChannel(t, roller, character.UseClericChannelDivinityCommand{Character: c.char, Subclass: "life_domain", Effect: character.PreserveLifeEffect{Targets: c.targets}}).(character.UseClericChannelDivinityApplied)
			if !ok {
				t.Fatal("not applied")
			}
			out := r.Outcome().(character.PreserveLifeOutcome)
			if out.Pool() != c.pool || len(out.Allocations()) != len(c.targets) || len(roller.Requests()) != 0 {
				t.Errorf("outcome pool %d allocations %v, requests %v", out.Pool(), out.Allocations(), roller.Requests())
			}
			for i, a := range out.Allocations() {
				if a.Key != c.targets[i].Key || a.HitPoints != c.targets[i].Allocation {
					t.Errorf("allocation %d = %+v", i, a)
				}
			}
			if ev := r.Events()[0].(character.ClericChannelDivinityUsed); ev.Effect() != "preserve_life" {
				t.Errorf("event = %+v", ev)
			}
			if len(out.Allocations()) > 0 {
				out.Allocations()[0].HitPoints = 99
				if out.Allocations()[0].HitPoints == 99 {
					t.Error("Allocations returned a shared slice")
				}
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-040, FR-041, FR-046, FR-056, FR-057, FR-061, FR-062, FR-066, FR-106 to FR-110, FR-128, FR-132, tables 4.7.2 and 4.14.2 RJ-01 to RJ-11 (SRD 5.2.1, "Level 2: Channel Divinity", p. 37; "Level 3:
// Preserve Life", p. 40) - the first reason that holds is reported in the order below_level_2, effect_not_available, no_uses_remaining, then by effect; a refusal expends no use, rolls nothing, changes nothing and names the
// effect, the target and the numbers behind it.
// Path: per case (RJ rows and every reason): unhappy (state error, boundary, invalid input, rule interaction).
func TestUseClericChannelDivinityRejections(t *testing.T) {
	t.Parallel()
	noUses := func(level int) character.Character { return clcWith(t, clcCleric(t, level), clcUses(2, 0)) }
	far := character.UndeadTarget{Key: "wolf", Type: content.CreatureTypeBeast, DistanceFeet: 35}
	for _, c := range []struct {
		name     string
		char     character.Character
		subclass content.SubclassID
		effect   character.ClericChannelEffect
		want     character.UseClericChannelDivinityRejected
	}{
		{"RJ-01 below level 2", clcWith(t, clcCleric(t, 1), clcUses(0, 0)), "", character.TurnUndeadEffect{Targets: []character.UndeadTarget{far}}, character.UseClericChannelDivinityRejected{Reason: character.ClericBelowLevel2, Effect: "turn_undead"}},
		{"RJ-02 Preserve Life without Life Domain", noUses(2), "", character.PreserveLifeEffect{}, character.UseClericChannelDivinityRejected{Reason: character.ClericEffectNotAvailable, Effect: "preserve_life"}},
		{"Preserve Life, another subclass", clcCleric(t, 5), "other", character.PreserveLifeEffect{}, character.UseClericChannelDivinityRejected{Reason: character.ClericEffectNotAvailable, Effect: "preserve_life"}},
		{"Preserve Life, Life Domain at level 2", clcCleric(t, 2), "life_domain", character.PreserveLifeEffect{}, character.UseClericChannelDivinityRejected{Reason: character.ClericEffectNotAvailable, Effect: "preserve_life"}},
		{"RJ-03 no uses", noUses(2), "", character.TurnUndeadEffect{Targets: []character.UndeadTarget{far}}, character.UseClericChannelDivinityRejected{Reason: character.ClericNoUsesRemaining, Effect: "turn_undead"}},
		{"RJ-04 not Undead", clcCleric(t, 2), "", character.TurnUndeadEffect{Targets: []character.UndeadTarget{far}}, character.UseClericChannelDivinityRejected{Reason: character.ClericTargetNotUndead, Effect: "turn_undead", Target: "wolf"}},
		{"RJ-05 31 feet", clcCleric(t, 2), "", character.TurnUndeadEffect{Targets: []character.UndeadTarget{undead("z", 31)}}, character.UseClericChannelDivinityRejected{Reason: character.ClericTargetOutOfRange, Effect: "turn_undead", Target: "z"}},
		{"RJ-06 Sear Undead not gained", clcCleric(t, 4), "", character.TurnUndeadEffect{Targets: []character.UndeadTarget{far}, SearUndead: character.SearUndeadUse}, character.UseClericChannelDivinityRejected{Reason: character.ClericSearUndeadNotGained, Effect: "turn_undead"}},
		{"SU-10 level 4 with use", clcCleric(t, 4), "", character.TurnUndeadEffect{SearUndead: character.SearUndeadUse}, character.UseClericChannelDivinityRejected{Reason: character.ClericSearUndeadNotGained, Effect: "turn_undead"}},
		{"DT-02 Spark 31 feet", clcCleric(t, 2), "", character.DivineSparkEffect{Target: character.DivineSparkTarget{Key: "t", Visible: true, DistanceFeet: 31}, Mode: character.SparkHealing},
			character.UseClericChannelDivinityRejected{Reason: character.ClericTargetOutOfRange, Effect: "divine_spark", Target: "t"}},
		{"DT-04 Spark on itself", clcCleric(t, 2), "", character.DivineSparkEffect{Target: character.DivineSparkTarget{Key: "me", IsSelf: true, Visible: true}, Mode: character.SparkHealing},
			character.UseClericChannelDivinityRejected{Reason: character.ClericTargetIsSelf, Effect: "divine_spark", Target: "me"}},
		{"DT-05 Spark not visible", clcCleric(t, 2), "", character.DivineSparkEffect{Target: character.DivineSparkTarget{Key: "t", DistanceFeet: 10}, Mode: character.SparkHealing},
			character.UseClericChannelDivinityRejected{Reason: character.ClericTargetNotVisible, Effect: "divine_spark", Target: "t"}},
		{"Spark with Fire damage", clcCleric(t, 2), "", character.DivineSparkEffect{Target: character.DivineSparkTarget{Key: "t", Visible: true}, Mode: character.SparkDamage, DamageType: content.DamageFire},
			character.UseClericChannelDivinityRejected{Reason: character.ClericDamageTypeNotAllowed, Effect: "divine_spark", Target: "t"}},
		{"RJ-07 duplicate target", clcCleric(t, 3), "life_domain", character.PreserveLifeEffect{Targets: []character.PreserveLifeTarget{lifeTarget("a", 4, 20, 35, 1), lifeTarget("a", 4, 20, 10, 1)}},
			character.UseClericChannelDivinityRejected{Reason: character.ClericDuplicateTarget, Effect: "preserve_life", Target: "a"}},
		{"RJ-08 out of range", clcCleric(t, 3), "life_domain", character.PreserveLifeEffect{Targets: []character.PreserveLifeTarget{lifeTarget("a", 19, 20, 35, 1)}},
			character.UseClericChannelDivinityRejected{Reason: character.ClericTargetOutOfRange, Effect: "preserve_life", Target: "a"}},
		{"RJ-09 not Bloodied", clcCleric(t, 3), "life_domain", character.PreserveLifeEffect{Targets: []character.PreserveLifeTarget{lifeTarget("a", 11, 20, 10, -1)}},
			character.UseClericChannelDivinityRejected{Reason: character.ClericTargetNotBloodied, Effect: "preserve_life", Target: "a"}},
		{"RJ-10 negative allocation", clcCleric(t, 3), "life_domain", character.PreserveLifeEffect{Targets: []character.PreserveLifeTarget{lifeTarget("a", 4, 20, 10, -1)}},
			character.UseClericChannelDivinityRejected{Reason: character.ClericNegativeAllocation, Effect: "preserve_life", Target: "a"}},
		{"RJ-11 exceeds room before pool", clcCleric(t, 3), "life_domain", character.PreserveLifeEffect{Targets: []character.PreserveLifeTarget{lifeTarget("a", 4, 20, 10, 16)}},
			character.UseClericChannelDivinityRejected{Reason: character.ClericAllocationExceedsRoom, Effect: "preserve_life", Target: "a", Room: 6}},
		{"PL-10 exceeds pool", clcCleric(t, 3), "life_domain", character.PreserveLifeEffect{Targets: []character.PreserveLifeTarget{lifeTarget("a", 0, 40, 10, 8), lifeTarget("b", 0, 40, 10, 8)}},
			character.UseClericChannelDivinityRejected{Reason: character.ClericAllocationExceedsPool, Effect: "preserve_life", Pool: 15, Sum: 16}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller()
			cmd := character.UseClericChannelDivinityCommand{Character: c.char, Subclass: c.subclass, Effect: c.effect, Wisdom: modifierOf(t, 3)}
			res, err := useChannel(t, roller).Run(context.Background(), cmd)
			if err != nil || res != character.UseClericChannelDivinityResult(c.want) {
				t.Fatalf("= %#v, %v; want %#v", res, err, c.want)
			}
			if len(roller.Requests()) != 0 {
				t.Errorf("requests = %v, want none", roller.Requests())
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-122 - a character with no Cleric levels is UseClericChannelDivinityNotACleric even when Cleric states are stored; nothing is rolled.
// Path: per case (unhappy: state error).
func TestUseClericChannelDivinityNotACleric(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(4)
	for _, c := range []character.Character{fgtFighter(t, 5), clcWith(t, fgtFighter(t, 5), clcUses(2, 2))} {
		res, err := useChannel(t, roller).Run(context.Background(), sparkCmd(c, character.SparkHealing, ""))
		if _, ok := res.(character.UseClericChannelDivinityNotACleric); err != nil || !ok {
			t.Errorf("= %#v, %v", res, err)
		}
	}
	if len(roller.Requests()) != 0 {
		t.Errorf("requests = %v", roller.Requests())
	}
}

// Requirement: doc/specs/class-cleric.md FR-130, table 4.14.4 CF-01, CF-02 - a roller that returns its error, panics, is exhausted or answers for another expression is a *srderr.PortError for dice.Port with the cause
// reachable; the result is nil and the Character, with no use expended, is unchanged.
// Path: per case (unhappy: collaborator failure).
func TestUseClericChannelDivinityRollerFailures(t *testing.T) {
	t.Parallel()
	boom := errors.New("roller down")
	other := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dice.NewResult(dice.MustExpression(1, dice.D20(), 0), []dice.DieRoll{must(dice.NewDieRoll(dice.D20(), 7))})
	})
	spark, sear := sparkCmd(clcCleric(t, 5), character.SparkHealing, ""), character.UseClericChannelDivinityCommand{Character: clcCleric(t, 5), Effect: character.TurnUndeadEffect{SearUndead: character.SearUndeadUse}, Wisdom: modifierOf(t, 2)}
	for _, c := range []struct {
		name   string
		roller character.Roller
		cmd    character.UseClericChannelDivinityCommand
		cause  error
	}{
		{"CF-01 an error", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, boom }), spark, boom},
		{"exhausted", dicetest.NewFixedRoller(), spark, dicetest.ErrExhausted},
		{"a panic", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(boom) }), spark, boom},
		{"CF-02 another expression for Sear Undead", other, sear, srderr.ErrContractViolation},
		{"Sear Undead error", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, boom }), sear, boom},
	} {
		res, err := useChannel(t, c.roller).Run(context.Background(), c.cmd)
		var pe *srderr.PortError
		if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, c.cause) {
			t.Errorf("%s: = %v, %v; want a PortError for dice with the cause", c.name, res, err)
		}
	}
}

// Requirement: doc/specs/class-cleric.md FR-123, FR-126 - the result is identical with the no-op and with a recording telemetry; the constructor refuses the zero progression and a nil roller, and Run refuses the zero use case,
// a zero Character, a nil effect, a negative distance, a foreign type of state and a cancelled context, the last before any signal.
// Path: per case (unhappy: invalid input, cancellation).
func TestUseClericChannelDivinityInvalidInputAndNoop(t *testing.T) {
	t.Parallel()
	good := sparkCmd(clcCleric(t, 5), character.SparkHealing, "")
	rec, opts := fgtTelemetry()
	withRec, err := useChannel(t, dicetest.NewFixedRoller(4), opts...).Run(context.Background(), good)
	plain, err2 := useChannel(t, dicetest.NewFixedRoller(4)).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	if len(rec.Tracer.Spans()) != 1 {
		t.Errorf("spans = %d, want 1", len(rec.Tracer.Spans()))
	}
	_, err = character.NewUseClericChannelDivinity(character.ClericProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewUseClericChannelDivinity(clcProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.UseClericChannelDivinity{}.Run(context.Background(), good)
	wantInvalid(t, err, "zero use case")
	_, err = useChannel(t, dicetest.NewFixedRoller(1)).Run(context.Background(), character.UseClericChannelDivinityCommand{Effect: good.Effect})
	wantInvalid(t, err, "zero character")
	_, err = useChannel(t, dicetest.NewFixedRoller(1)).Run(context.Background(), character.UseClericChannelDivinityCommand{Character: good.Character})
	wantInvalid(t, err, "nil effect")
	negative := sparkCmd(clcCleric(t, 5), character.SparkHealing, "")
	negative.Effect = character.DivineSparkEffect{Target: character.DivineSparkTarget{Key: "t", Visible: true, DistanceFeet: -1}, Mode: character.SparkHealing}
	_, err = useChannel(t, dicetest.NewFixedRoller(1)).Run(context.Background(), negative)
	wantInvalid(t, err, "negative distance")
	foreign := must(character.WithFeatureState(clcCleric(t, 5), character.MustFeatureStateKey[character.ChosenSubclass]("cleric_resources"), must(character.NewChosenSubclass("life_domain"))))
	_, err = useChannel(t, dicetest.NewFixedRoller(1)).Run(context.Background(), sparkCmd(foreign, character.SparkHealing, ""))
	wantInvalid(t, err, "foreign state")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	rec2, opts2 := fgtTelemetry()
	roller := dicetest.NewFixedRoller(1)
	res, err := useChannel(t, roller, opts2...).Run(cancelled, good)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec2.Tracer.Spans()))
	}
}
