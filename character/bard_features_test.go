package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const lore = character.CollegeOfLoreSubclassID

// Requirement: doc/specs/class-bard.md FR-072 to FR-075, FR-104, FR-107 (SRD 5.2.1, "Level 3: Cutting Words", p. 35; "Rolling 20 or 1", p. 7) - the Cutting Words table: the number rolled is subtracted through adjustedBy(-rolled) from a succeeded
// ability check or attack roll, or from a damage total held at zero with the clamp recorded; a natural 20 attack stays a critical hit; the use is spent every time; the pool, the die and the event are reported.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestCuttingWordsTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		trigger character.CuttingWordsTrigger
		roll    int
		total   int
		want    d20.Outcome
		damage  int
		clamped bool
	}{
		{"success becomes failure", character.CuttingWordsRoll{Result: brdAbility(t, 15, 0, 15)}, 1, 14, d20.Failure, 0, false},
		{"still meets the DC", character.CuttingWordsRoll{Result: brdAbility(t, 17, 0, 15)}, 2, 15, d20.Success, 0, false},
		{"comfortably succeeds", character.CuttingWordsRoll{Result: brdAbility(t, 20, 0, 15)}, 4, 16, d20.Success, 0, false},
		{"attack stops hitting", character.CuttingWordsRoll{Result: brdAttack(t, 14, 0, 13)}, 2, 12, d20.Failure, 0, false},
		{"attack still hits", character.CuttingWordsRoll{Result: brdAttack(t, 14, 0, 13)}, 1, 13, d20.Success, 0, false},
		{"natural 20 attack stays a critical hit", character.CuttingWordsRoll{Result: brdAttack(t, 20, 5, 30)}, 8, 17, d20.Success, 0, false},
		{"damage reduced", character.CuttingWordsDamage{Total: 9}, 4, 0, 0, 5, false},
		{"roll exceeds the damage", character.CuttingWordsDamage{Total: 3}, 8, 0, 0, 0, true},
		{"roll equals the damage", character.CuttingWordsDamage{Total: 5}, 5, 0, 0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			pool := brdPool(t, 3, 2)
			plan, reason, err := character.PlanCuttingWords(brdID, character.MustBardLevel(5), lore, pool, brdMod(t, 16), c.trigger)
			if err != nil || reason != 0 || plan.Die() != character.BardicD8 {
				t.Fatalf("plan = %v, %v, %v", plan.Die(), reason, err)
			}
			got, err := character.ApplyCuttingWords(plan, c.roll)
			if err != nil {
				t.Fatal(err)
			}
			if got.Pool().Remaining() != 1 || got.Event().Feature() != "cutting_words" || got.Event().Rolled() != c.roll || got.Event().Die() != character.BardicD8 || got.Damage() != c.damage || got.Clamped() != c.clamped {
				t.Errorf("pool %d, event %s/%d, damage %d, clamped %v", got.Pool().Remaining(), got.Event().Feature(), got.Event().Rolled(), got.Damage(), got.Clamped())
			}
			if _, isRoll := c.trigger.(character.CuttingWordsRoll); isRoll {
				if got.Result() == nil || got.Result().Total() != c.total || got.Result().Outcome() != c.want {
					t.Errorf("result = %v, want total %d, %v", got.Result(), c.total, c.want)
				}
			} else if got.Result() != nil {
				t.Errorf("a damage trigger returned the result %v", got.Result())
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-074, FR-119 - Cutting Words is refused before any roll, in the order not_a_bard, feature_unavailable (below level 3, or any subclass but the College of Lore), no_uses_remaining, ineligible_trigger (a saving
// throw, a check that was not a success); levels 2 and 3 are tested; an invalid trigger is an ErrInvalidInput error.
// Path: per case (unhappy: boundary, state error, invalid input).
func TestPlanCuttingWordsRefusals(t *testing.T) {
	t.Parallel()
	ok := character.CuttingWordsRoll{Result: brdAbility(t, 16, 0, 15)}
	for _, c := range []struct {
		name     string
		level    int
		subclass content.SubclassID
		pool     character.BardicInspirationPool
		trigger  character.CuttingWordsTrigger
		reason   character.BardReason
	}{
		{"level 3", 3, lore, brdPool(t, 2, 1), ok, 0},
		{"level 2", 2, lore, brdPool(t, 2, 1), ok, character.BardFeatureUnavailable},
		{"no subclass", 3, "", brdPool(t, 2, 1), ok, character.BardFeatureUnavailable},
		{"another subclass", 9, "other", brdPool(t, 2, 1), ok, character.BardFeatureUnavailable},
		{"no uses", 3, lore, brdPool(t, 2, 0), ok, character.BardNoUsesRemaining},
		{"not a Bard", 0, lore, brdPool(t, 2, 1), ok, character.BardNotABard},
		{"a saving throw", 3, lore, brdPool(t, 2, 1), character.CuttingWordsRoll{Result: brdSave(t, 16, 0, 15)}, character.BardIneligibleTrigger},
		{"a failed check", 3, lore, brdPool(t, 2, 1), character.CuttingWordsRoll{Result: brdAbility(t, 5, 0, 15)}, character.BardIneligibleTrigger},
		{"unavailable before no uses", 2, lore, brdPool(t, 2, 0), character.CuttingWordsRoll{Result: brdSave(t, 5, 0, 15)}, character.BardFeatureUnavailable},
		{"no uses before the trigger", 3, lore, brdPool(t, 2, 0), character.CuttingWordsRoll{Result: brdSave(t, 5, 0, 15)}, character.BardNoUsesRemaining},
		{"zero damage", 3, lore, brdPool(t, 2, 1), character.CuttingWordsDamage{}, 0},
	} {
		level := character.BardLevel{}
		if c.level > 0 {
			level = character.MustBardLevel(c.level)
		}
		if _, reason, err := character.PlanCuttingWords(brdID, level, c.subclass, c.pool, brdMod(t, 14), c.trigger); err != nil || reason != c.reason {
			t.Errorf("%s: reason %v, error %v, want %v", c.name, reason, err, c.reason)
		}
	}
	for name, trigger := range map[string]character.CuttingWordsTrigger{"nil": nil, "negative damage": character.CuttingWordsDamage{Total: -1}, "zero result": character.CuttingWordsRoll{}, "nil result": character.CuttingWordsRoll{Result: nil}} {
		if _, _, err := character.PlanCuttingWords(brdID, character.MustBardLevel(5), lore, brdPool(t, 2, 1), brdMod(t, 14), trigger); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s trigger: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, err := character.ApplyCuttingWords(character.CuttingWordsPlan{}, 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero plan: error = %v", err)
	}
	plan, _, _ := character.PlanCuttingWords(brdID, character.MustBardLevel(5), lore, brdPool(t, 2, 1), brdMod(t, 14), ok)
	if _, err := character.ApplyCuttingWords(plan, 9); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a roll above the d8: error = %v", err)
	}
}

// Requirement: doc/specs/class-bard.md FR-081, FR-082, FR-104 (SRD 5.2.1, "Level 14: Peerless Skill", p. 35) - the Peerless Skill table with a d10: the number rolled is added through adjustedBy(rolled); the use is spent only when the result becomes a success;
// a natural 1 attack stays a miss and spends nothing.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestPeerlessSkillTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		failed character.AdjustableD20Result
		roll   int
		total  int
		want   d20.Outcome
		spent  bool
	}{
		{"reaches the DC", brdAbility(t, 10, 0, 15), 5, 15, d20.Success, true},
		{"one short of the DC", brdAbility(t, 10, 0, 15), 4, 14, d20.Failure, false},
		{"well over", brdAbility(t, 14, 0, 15), 10, 24, d20.Success, true},
		{"natural 1 attack stays a miss", brdAttack(t, 1, 0, 5), 9, 10, d20.Failure, false},
		{"an attack roll that becomes a hit", brdAttack(t, 8, 0, 13), 5, 13, d20.Success, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			plan, reason, err := character.PlanPeerlessSkill(brdID, character.MustBardLevel(14), lore, brdPool(t, 3, 2), brdMod(t, 16), c.failed)
			if err != nil || reason != 0 || plan.Die() != character.BardicD10 {
				t.Fatalf("plan = %v, %v, %v", plan.Die(), reason, err)
			}
			got, err := character.ApplyPeerlessSkill(plan, c.roll)
			if err != nil || got.Result().Total() != c.total || got.Result().Outcome() != c.want || got.Spent() != c.spent {
				t.Fatalf("applied = %d, %v, spent %v (%v); want %d, %v, %v", got.Result().Total(), got.Result().Outcome(), got.Spent(), err, c.total, c.want, c.spent)
			}
			wantPool, wantEvents := 2, 0
			if c.spent {
				wantPool, wantEvents = 1, 1
			}
			ev := got.Events()
			if got.Pool().Remaining() != wantPool || len(ev) != wantEvents {
				t.Fatalf("pool %d, %d events; want %d and %d", got.Pool().Remaining(), len(ev), wantPool, wantEvents)
			}
			if c.spent {
				if e, isExpended := ev[0].(character.BardicInspirationExpended); !isExpended || e.Feature() != "peerless_skill" || e.Rolled() != c.roll || e.Die() != character.BardicD10 {
					t.Errorf("event = %#v", ev[0])
				}
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-083, FR-119 - Peerless Skill is refused before any roll, in the order not_a_bard, feature_unavailable (levels 13 and 14 tested; another subclass), no_uses_remaining, ineligible_trigger (a saving throw, a result
// that was not a failure); an invalid result is an ErrInvalidInput error, as are a zero plan and a roll outside the die.
// Path: per case (unhappy: boundary, state error, invalid input).
func TestPlanPeerlessSkillRefusals(t *testing.T) {
	t.Parallel()
	failed := brdAbility(t, 5, 0, 15)
	for _, c := range []struct {
		name     string
		level    int
		subclass content.SubclassID
		pool     character.BardicInspirationPool
		result   character.AdjustableD20Result
		reason   character.BardReason
	}{
		{"level 14", 14, lore, brdPool(t, 2, 1), failed, 0},
		{"level 13", 13, lore, brdPool(t, 2, 1), failed, character.BardFeatureUnavailable},
		{"another subclass", 14, "other", brdPool(t, 2, 1), failed, character.BardFeatureUnavailable},
		{"no uses", 14, lore, brdPool(t, 2, 0), failed, character.BardNoUsesRemaining},
		{"not a Bard", 0, lore, brdPool(t, 2, 1), failed, character.BardNotABard},
		{"a saving throw", 14, lore, brdPool(t, 2, 1), brdSave(t, 5, 0, 15), character.BardIneligibleTrigger},
		{"a success", 14, lore, brdPool(t, 2, 1), brdAbility(t, 18, 0, 15), character.BardIneligibleTrigger},
	} {
		level := character.BardLevel{}
		if c.level > 0 {
			level = character.MustBardLevel(c.level)
		}
		if _, reason, err := character.PlanPeerlessSkill(brdID, level, c.subclass, c.pool, brdMod(t, 14), c.result); err != nil || reason != c.reason {
			t.Errorf("%s: reason %v, error %v, want %v", c.name, reason, err, c.reason)
		}
	}
	for name, result := range map[string]character.AdjustableD20Result{"nil": nil, "zero": character.AdjustableAttack{}} {
		if _, _, err := character.PlanPeerlessSkill(brdID, character.MustBardLevel(14), lore, brdPool(t, 2, 1), brdMod(t, 14), result); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s result: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, err := character.ApplyPeerlessSkill(character.PeerlessSkillPlan{}, 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero plan: error = %v", err)
	}
	plan, _, _ := character.PlanPeerlessSkill(brdID, character.MustBardLevel(14), lore, brdPool(t, 2, 1), brdMod(t, 14), failed)
	if _, err := character.ApplyPeerlessSkill(plan, 11); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a roll above the d10: error = %v", err)
	}
}

// Requirement: doc/specs/class-bard.md FR-084, FR-085, FR-086, FR-105, FR-106 (SRD 5.2.1, "Level 7: Countercharm", p. 33; "Advantage", p. 176; "They Don't Stack", p. 8) - the roll-mode table: no circumstances give two d20 with Advantage, one or two
// Disadvantage cancel to one d20, one Advantage stays two d20 (no stacking), one of each is one d20; the three gates run in the order level (6 and 7 tested), effect (Charmed or Frightened), save, before any roll.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestPlanCountercharm(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	failedSave := brdCheck(t, d20.SavingThrow, 5, 0, 15)
	charmed := []content.ConditionKind{content.ConditionCharmed}
	for _, c := range []struct {
		name  string
		circs []modifier.RollCircumstance
		mode  modifier.RollMode
		dice  int
	}{
		{"none", nil, modifier.WithAdvantage, 2},
		{"one Disadvantage", []modifier.RollCircumstance{dis}, modifier.Normal, 1},
		{"two Disadvantage", []modifier.RollCircumstance{dis, dis}, modifier.Normal, 1},
		{"one Advantage", []modifier.RollCircumstance{adv}, modifier.WithAdvantage, 2},
		{"one of each", []modifier.RollCircumstance{adv, dis}, modifier.Normal, 1},
	} {
		plan, reason, err := character.PlanCountercharm(character.MustBardLevel(7), failedSave, c.circs, charmed)
		if err != nil || reason != 0 || plan.Mode() != c.mode || plan.DiceCount() != c.dice {
			t.Errorf("%s: mode %v, %d dice, reason %v, error %v; want %v and %d", c.name, plan.Mode(), plan.DiceCount(), reason, err, c.mode, c.dice)
		}
	}
	for _, c := range []struct {
		name   string
		level  int
		failed d20.CheckResult
		conds  []content.ConditionKind
		reason character.BardReason
	}{
		{"level 7", 7, failedSave, []content.ConditionKind{content.ConditionFrightened}, 0},
		{"level 6", 6, failedSave, charmed, character.BardBelowRequiredLevel},
		{"not a Bard", 0, failedSave, charmed, character.BardNotABard},
		{"neither condition", 7, failedSave, []content.ConditionKind{content.ConditionPoisoned}, character.BardIneligibleEffect},
		{"no conditions", 7, failedSave, nil, character.BardIneligibleEffect},
		{"the save passed", 7, brdCheck(t, d20.SavingThrow, 18, 0, 15), charmed, character.BardSaveNotFailed},
		{"an ability check", 7, brdCheck(t, d20.AbilityCheck, 5, 0, 15), charmed, character.BardSaveNotFailed},
		{"level before effect before save", 6, brdCheck(t, d20.SavingThrow, 18, 0, 15), nil, character.BardBelowRequiredLevel},
		{"effect before save", 7, brdCheck(t, d20.SavingThrow, 18, 0, 15), nil, character.BardIneligibleEffect},
	} {
		level := character.BardLevel{}
		if c.level > 0 {
			level = character.MustBardLevel(c.level)
		}
		if _, reason, err := character.PlanCountercharm(level, c.failed, nil, c.conds); err != nil || reason != c.reason {
			t.Errorf("%s: reason %v, error %v, want %v", c.name, reason, err, c.reason)
		}
	}
	if _, _, err := character.PlanCountercharm(character.MustBardLevel(7), d20.CheckResult{}, nil, charmed); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero result: error = %v, want ErrInvalidInput", err)
	}
	circs := []modifier.RollCircumstance{dis}
	_, _, _ = character.PlanCountercharm(character.MustBardLevel(7), failedSave, circs, charmed)
	if len(circs) != 1 {
		t.Error("PlanCountercharm altered the circumstances")
	}
}

// Requirement: doc/specs/class-bard.md FR-085, FR-087, FR-106 - the rerolled save keeps the original modifiers and DC, reports both outcomes, takes the higher of two d20 under Advantage from the d20 package and the single d20 when the circumstances
// cancel; a zero plan and faces that do not match the mode are ErrInvalidInput errors.
// Path: per case (happy / unhappy: invalid input).
func TestResolveCountercharm(t *testing.T) {
	t.Parallel()
	failed := brdCheck(t, d20.SavingThrow, 5, 2, 15)
	plan, _, err := character.PlanCountercharm(character.MustBardLevel(7), failed, nil, []content.ConditionKind{content.ConditionCharmed})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		faces   []int
		natural int
		want    d20.Outcome
	}{{[]int{3, 17}, 17, d20.Success}, {[]int{17, 3}, 17, d20.Success}, {[]int{3, 5}, 5, d20.Failure}, {[]int{13, 13}, 13, d20.Success}} {
		got, err := character.ResolveCountercharm(plan, c.faces)
		if err != nil || got.Rerolled().Roll().Natural() != c.natural || got.Rerolled().Outcome() != c.want || got.Rerolled().Total() != c.natural+2 ||
			got.Rerolled().Target() != failed.Target() || !got.Rerolled().Modifiers().Equal(failed.Modifiers()) || !got.Failed().Equal(failed) {
			t.Errorf("faces %v: natural %d, outcome %v, error %v", c.faces, got.Rerolled().Roll().Natural(), got.Rerolled().Outcome(), err)
		}
	}
	single, _, _ := character.PlanCountercharm(character.MustBardLevel(7), failed, []modifier.RollCircumstance{modifier.Disadvantage{}}, []content.ConditionKind{content.ConditionFrightened})
	if got, err := character.ResolveCountercharm(single, []int{12}); err != nil || got.Rerolled().Roll().Natural() != 12 || got.Rerolled().Roll().Mode() != modifier.Normal {
		t.Errorf("a single d20: %v, %v", got.Rerolled().Roll(), err)
	}
	for name, run := range map[string]func() error{
		"zero plan": func() error {
			_, e := character.ResolveCountercharm(character.CountercharmPlan{}, []int{10, 11})
			return e
		},
		"too few faces":   func() error { _, e := character.ResolveCountercharm(plan, []int{10}); return e },
		"face above 20":   func() error { _, e := character.ResolveCountercharm(plan, []int{10, 21}); return e },
		"too many faces":  func() error { _, e := character.ResolveCountercharm(single, []int{10, 11}); return e },
		"no faces at all": func() error { _, e := character.ResolveCountercharm(plan, nil); return e },
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-064, FR-065 (SRD 5.2.1, "Level 2: Jack of All Trades", p. 32; "Round Down", p. 187) - Jack of All Trades adds half the Proficiency Bonus rounded down to a check with a skill the Bard lacks (+1 at +2 and +3, +2 at +4
// and +5, +3 at +6); not below level 2, not for a proficient or Expertise skill, not for a check with no skill, not when the check already uses the Proficiency Bonus.
// Path: per case (the table of the spec): happy / unhappy (boundary, rule interaction, invalid input).
func TestJackOfAllTrades(t *testing.T) {
	t.Parallel()
	athletics, stealth := content.SkillAthletics, content.SkillStealth
	profs := must(character.NewSkillProficiencies([]content.Skill{stealth}, []content.Skill{stealth}))
	for _, c := range []struct {
		name  string
		level int
		bonus int
		skill *content.Skill
		uses  bool
		want  int // -1: none
	}{
		{"level 2, lacks Athletics", 2, 2, &athletics, false, 1},
		{"level 5, rounds down", 5, 3, &athletics, false, 1},
		{"level 9", 9, 4, &athletics, false, 2},
		{"level 13, rounds down", 13, 5, &athletics, false, 2},
		{"level 17", 17, 6, &athletics, false, 3},
		{"proficient skill with Expertise", 9, 4, &stealth, false, -1},
		{"ability check with no skill", 5, 3, nil, false, -1},
		{"level 1", 1, 2, &athletics, false, -1},
		{"not a Bard", 0, 2, &athletics, false, -1},
		{"also uses the Proficiency Bonus through a tool", 5, 3, &athletics, true, -1},
	} {
		level := character.BardLevel{}
		if c.level > 0 {
			level = character.MustBardLevel(c.level)
		}
		bonus := character.ProficiencyBonusFor(character.MustLevel(map[int]int{2: 1, 3: 5, 4: 9, 5: 13, 6: 17}[c.bonus]))
		got, ok := character.JackOfAllTrades(level, bonus, c.skill, profs, c.uses)
		if (c.want >= 0) != ok || (ok && got.Value() != c.want) {
			t.Errorf("%s: %v, %v; want %d", c.name, got, ok, c.want)
		}
	}
	plain := must(character.NewSkillProficiencies([]content.Skill{athletics}, nil))
	if _, ok := character.JackOfAllTrades(character.MustBardLevel(5), character.ProficiencyBonusFor(character.MustLevel(5)), &athletics, plain, false); ok {
		t.Error("Jack of All Trades applied to a proficient skill")
	}
}

// Requirement: doc/specs/class-bard.md FR-096 (SRD 5.2.1, "Level 20: Words of Creation", p. 33) - a second target is allowed exactly at Bard level 20 and only for Power Word Heal and Power Word Kill; levels 19 and 20 are tested.
// Path: per case (unhappy: boundary).
func TestAllowsSecondTarget(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		level int
		spell content.SpellID
		want  bool
	}{{20, "power_word_heal", true}, {20, "power_word_kill", true}, {19, "power_word_heal", false}, {19, "power_word_kill", false}, {20, "fireball", false}, {20, "", false}, {1, "power_word_kill", false}} {
		if got := character.AllowsSecondTarget(character.MustBardLevel(c.level), c.spell); got != c.want {
			t.Errorf("level %d, %q = %v, want %v", c.level, c.spell, got, c.want)
		}
	}
	if character.AllowsSecondTarget(character.BardLevel{}, "power_word_kill") {
		t.Error("the zero level allows a second target")
	}
}
